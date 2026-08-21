// Package backoff computes retry delays and submits retries to simulation/time timeline.
// It is a dependency-injected port of github.com/cenkalti/backoff: the upstream
// package is float-based (Multiplier 1.5, RandomizationFactor 0.5), draws jitter
// from the global math/rand, and waits on a real time.Timer under a context — none
// of which the house linter allows and deterministic simulation cannot replay. Here
// the growth factor and jitter are integer prng.Ratios, the jitter entropy is an
// injected prng.Generator, and durations derive from simulation/time.Duration, so a schedule
// reproduces bit-for-bit from a seed.
//
// Linter bans user interfaces. Policy becomes static procedures over explicit caller-owned
// Policy_State. No closure captures state. Library never holds time.Driver. Retry submits one
// timeout. Its callback records work; root rearms runner and reads Retry_Status.
package backoff

import (
	"errors"
	"unsafe"

	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/random/prng"
	"local/james-orcales/shared/simulation/time"
	"local/james-orcales/shared/slices"
)

// STOP is the delay a Policy's Next returns to signal that no more retries should be made.
const STOP Delay = -1

// DEFAULT_INITIAL_INTERVAL is the first delay New_Exponential grows from.
const DEFAULT_INITIAL_INTERVAL = 500 * time.MILLISECOND

// DEFAULT_INTERVAL_MAX caps the growing interval New_Exponential produces.
const DEFAULT_INTERVAL_MAX = 60 * time.SECOND

// Initial_Interval is first or constant delay.
type Initial_Interval time.Duration

// Initial_Interval_Invariants bounds first delay to one process uptime year.
func Initial_Interval_Invariants(interval Initial_Interval, namespace invariant.Namespace) {
	invariant.Tree(interval, namespace).
		Range_Int64(int64(interval), INTERVAL_MINIMUM_INT64, INTERVAL_MAXIMUM_INT64).
		Ensure()
}

// Current_Interval is mutable exponential base.
type Current_Interval time.Duration

// Current_Interval_Invariants bounds mutable exponential base.
func Current_Interval_Invariants(interval Current_Interval, namespace invariant.Namespace) {
	invariant.Tree(interval, namespace).
		Range_Int64(int64(interval), INTERVAL_MINIMUM_INT64, INTERVAL_MAXIMUM_INT64).
		Ensure()
}

// Maximum_Interval caps exponential base.
type Maximum_Interval time.Duration

// Maximum_Interval_Invariants bounds exponential cap.
func Maximum_Interval_Invariants(interval Maximum_Interval, namespace invariant.Namespace) {
	invariant.Tree(interval, namespace).
		Range_Int64(int64(interval), INTERVAL_MINIMUM_INT64, INTERVAL_MAXIMUM_INT64).
		Ensure()
}

// INTERVAL_MAXIMUM matches bounded process uptime.
const INTERVAL_MAXIMUM = Initial_Interval(time.DAY * 365)

// INTERVAL_MINIMUM_INT64 starts immediate retry.
const INTERVAL_MINIMUM_INT64 int64 = 0

// INTERVAL_MAXIMUM_INT64 supplies canonical range bound.
const INTERVAL_MAXIMUM_INT64 int64 = int64(INTERVAL_MAXIMUM)

// Delay is one emitted retry wait or STOP.
type Delay time.Duration

// Delay_Invariants bounds delay to complete timeline duration domain.
func Delay_Invariants(delay Delay, namespace invariant.Namespace) {
	invariant.Tree(delay, namespace).
		Range_Int64(int64(delay), DELAY_MINIMUM, DELAY_MAXIMUM).
		Ensure()
}

// DELAY_MINIMUM includes STOP.
const DELAY_MINIMUM int64 = int64(STOP)

// DELAY_MAXIMUM matches bounded retry interval.
const DELAY_MAXIMUM int64 = int64(INTERVAL_MAXIMUM)

// Wait is nonnegative emitted delay before policy stop translation.
type Wait time.Duration

// Wait_Invariants bounds nonnegative wait.
func Wait_Invariants(wait Wait, namespace invariant.Namespace) {
	invariant.Tree(wait, namespace).
		Range_Int64(int64(wait), INTERVAL_MINIMUM_INT64, INTERVAL_MAXIMUM_INT64).
		Ensure()
}

// Retry_Delay is nonnegative Retry_After override.
type Retry_Delay time.Duration

// Retry_Delay_Invariants bounds override wait.
func Retry_Delay_Invariants(delay Retry_Delay, namespace invariant.Namespace) {
	invariant.Tree(delay, namespace).
		Range_Int64(int64(delay), INTERVAL_MINIMUM_INT64, INTERVAL_MAXIMUM_INT64).
		Ensure()
}

// Boolean is bounded binary report.
type Boolean bool

// Boolean_Invariants requires both report values across package runs.
func Boolean_Invariants(value Boolean, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "Backoff report is true.").
		Ensure()
}

// RATIO_PART_MINIMUM is the smallest unsigned ratio part.
const RATIO_PART_MINIMUM uint64 = uint64(bits.WORD_64_MINIMUM)

// RATIO_PART_POSITIVE_MINIMUM is the first denominator and growth numerator.
const RATIO_PART_POSITIVE_MINIMUM = RATIO_PART_MINIMUM + 1

// RATIO_PART_MAXIMUM is the complete unsigned ratio-part domain.
const RATIO_PART_MAXIMUM uint64 = bits.WORD_64_MAXIMUM

// Multiplier_Numerator is positive growth mass.
type Multiplier_Numerator uint64

// Multiplier_Numerator_Invariants preserves complete storage while constructor validates use.
func Multiplier_Numerator_Invariants(
	value Multiplier_Numerator, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint64(
			uint64(value), RATIO_PART_POSITIVE_MINIMUM, RATIO_PART_MAXIMUM,
		).
		Ensure()
}

// Multiplier_Denominator is positive unchanged mass.
type Multiplier_Denominator uint64

// Multiplier_Denominator_Invariants preserves complete storage while constructor validates use.
func Multiplier_Denominator_Invariants(
	value Multiplier_Denominator, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint64(
			uint64(value), RATIO_PART_POSITIVE_MINIMUM, RATIO_PART_MAXIMUM,
		).
		Ensure()
}

// Multiplier is an exponential growth ratio, where numerator is never below denominator in use.
type Multiplier struct {
	// Numerator is the growth mass.
	Numerator Multiplier_Numerator
	// Denominator is the unchanged interval mass.
	Denominator Multiplier_Denominator
}

// Multiplier_Invariants rejects a shrinking interval.
func Multiplier_Invariants(multiplier Multiplier, namespace invariant.Namespace) {
	Multiplier_Numerator_Invariants(multiplier.Numerator, namespace)
	Multiplier_Denominator_Invariants(multiplier.Denominator, namespace)
	invariant.Always(
		multiplier.Numerator >= Multiplier_Numerator(multiplier.Denominator),
		"Multiplier never shrinks interval.",
	)
}

// Jitter_Numerator is spread mass, with zero selecting no jitter.
type Jitter_Numerator uint64

// Jitter_Numerator_Invariants preserves every representable spread mass.
func Jitter_Numerator_Invariants(value Jitter_Numerator, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), RATIO_PART_MINIMUM, RATIO_PART_MAXIMUM).
		Ensure()
}

// Jitter_Denominator is positive interval mass.
type Jitter_Denominator uint64

// Jitter_Denominator_Invariants preserves complete storage while constructor validates use.
func Jitter_Denominator_Invariants(value Jitter_Denominator, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(
			uint64(value), RATIO_PART_POSITIVE_MINIMUM, RATIO_PART_MAXIMUM,
		).
		Ensure()
}

// Jitter is the fraction of one interval available to the random spread.
type Jitter struct {
	// Numerator is random spread mass.
	Numerator Jitter_Numerator
	// Denominator is complete interval mass.
	Denominator Jitter_Denominator
}

// Jitter_Invariants rejects spread beyond a complete interval.
func Jitter_Invariants(jitter Jitter, namespace invariant.Namespace) {
	Jitter_Numerator_Invariants(jitter.Numerator, namespace)
	Jitter_Denominator_Invariants(jitter.Denominator, namespace)
	invariant.Always(
		jitter.Numerator <= Jitter_Numerator(jitter.Denominator),
		"Jitter never exceeds complete interval.",
	)
}

// Stored_Multiplier_Numerator includes zero while caller storage is uninitialized.
type Stored_Multiplier_Numerator uint64

// Stored_Multiplier_Numerator_Invariants spans complete caller storage.
func Stored_Multiplier_Numerator_Invariants(
	value Stored_Multiplier_Numerator, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), RATIO_PART_MINIMUM, RATIO_PART_MAXIMUM).
		Ensure()
}

// Stored_Multiplier_Denominator includes zero while caller storage is uninitialized.
type Stored_Multiplier_Denominator uint64

// Stored_Multiplier_Denominator_Invariants spans complete caller storage.
func Stored_Multiplier_Denominator_Invariants(
	value Stored_Multiplier_Denominator, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), RATIO_PART_MINIMUM, RATIO_PART_MAXIMUM).
		Ensure()
}

// Stored_Multiplier separates uninitialized policy storage from configured input.
type Stored_Multiplier struct {
	// Numerator is stored growth mass.
	Numerator Stored_Multiplier_Numerator
	// Denominator is stored unchanged mass.
	Denominator Stored_Multiplier_Denominator
}

// Stored_Multiplier_Invariants composes complete caller storage.
func Stored_Multiplier_Invariants(value Stored_Multiplier, namespace invariant.Namespace) {
	Stored_Multiplier_Numerator_Invariants(value.Numerator, namespace)
	Stored_Multiplier_Denominator_Invariants(value.Denominator, namespace)
}

// Stored_Jitter_Numerator spans zero through complete unsigned storage.
type Stored_Jitter_Numerator uint64

// Stored_Jitter_Numerator_Invariants spans complete caller storage.
func Stored_Jitter_Numerator_Invariants(
	value Stored_Jitter_Numerator, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), RATIO_PART_MINIMUM, RATIO_PART_MAXIMUM).
		Ensure()
}

// Stored_Jitter_Denominator includes zero while caller storage is uninitialized.
type Stored_Jitter_Denominator uint64

// Stored_Jitter_Denominator_Invariants spans complete caller storage.
func Stored_Jitter_Denominator_Invariants(
	value Stored_Jitter_Denominator, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), RATIO_PART_MINIMUM, RATIO_PART_MAXIMUM).
		Ensure()
}

// Stored_Jitter separates uninitialized policy storage from configured input.
type Stored_Jitter struct {
	// Numerator is stored spread mass.
	Numerator Stored_Jitter_Numerator
	// Denominator is stored interval mass.
	Denominator Stored_Jitter_Denominator
}

// Stored_Jitter_Invariants composes complete caller storage.
func Stored_Jitter_Invariants(value Stored_Jitter, namespace invariant.Namespace) {
	Stored_Jitter_Numerator_Invariants(value.Numerator, namespace)
	Stored_Jitter_Denominator_Invariants(value.Denominator, namespace)
}

// Generator_Storage includes zero while caller policy storage is uninitialized.
type Generator_Storage prng.Generator

// Generator_Storage_Invariants fixes the xoshiro state width without requiring initialization.
func Generator_Storage_Invariants(generator Generator_Storage, _ invariant.Namespace) {
	invariant.Always(
		len(generator.State) == prng.GENERATOR_STATE_WORD_COUNT,
		"Generator storage keeps the xoshiro state width.",
	)
}

// Required_Generator is active caller-owned jitter entropy.
type Required_Generator *prng.Generator

// Required_Generator_Invariants rejects missing entropy before dereferencing caller storage.
func Required_Generator_Invariants(generator Required_Generator, namespace invariant.Namespace) {
	invariant.Always(generator != nil, "Jitter has caller-owned entropy.")
	prng.Generator_Invariants(*generator, namespace)
}

// Error_Permanent is the sentinel Permanent wraps; Retry stops at once when the
// operation's error matches it.
var Error_Permanent = errors.New("backoff: permanent error")

// Error_Exhausted is the sentinel Retry returns when it runs out of tries or the policy stops.
var Error_Exhausted = errors.New("backoff: retries exhausted")

// Error_Elapsed_Max is the sentinel Retry returns when the next wait would exceed Elapsed_Time_Max.
var Error_Elapsed_Max = errors.New("backoff: maximum elapsed time exceeded")

// Policy is static retry procedure table over explicit caller-owned state.
type Policy struct {
	// State borrows concrete policy state where needed.
	State unsafe.Pointer
	// Next computes one delay without captured closure storage.
	Next func(state unsafe.Pointer) (delay Delay)
	// Reset restores concrete policy state.
	Reset func(state unsafe.Pointer)
}

// Policy_Invariants requires complete static procedure table.
func Policy_Invariants(policy Policy, _ invariant.Namespace) {
	invariant.Always(policy.Next != nil, "Policy has next procedure.")
	invariant.Always(policy.Reset != nil, "Policy has reset procedure.")
}

// Policy_Kind selects closed retry strategy.
type Policy_Kind uint8

// Policy_Kind_Invariants closes dispatch to implemented strategies.
func Policy_Kind_Invariants(kind Policy_Kind, namespace invariant.Namespace) {
	invariant.Tree(kind, namespace).
		Enum_3_Uint8(
			uint8(kind), uint8(POLICY_KIND_CONSTANT), uint8(POLICY_KIND_STOPPED),
			uint8(POLICY_KIND_EXPONENTIAL),
		).
		Ensure()
}

// POLICY_KIND_CONSTANT selects fixed delay.
const POLICY_KIND_CONSTANT Policy_Kind = 0

// POLICY_KIND_STOPPED selects no retry.
const POLICY_KIND_STOPPED Policy_Kind = 1

// POLICY_KIND_EXPONENTIAL selects growing jittered delay.
const POLICY_KIND_EXPONENTIAL Policy_Kind = 2

// Policy_State stores caller-owned state for every closed strategy.
type Policy_State struct {
	// Kind selects closed dispatch.
	Kind Policy_Kind
	// Initial_Interval is fixed delay or exponential reset base.
	Initial_Interval Initial_Interval
	// Current_Interval is next exponential base.
	Current_Interval Current_Interval
	// Interval_Max caps exponential base.
	Interval_Max Maximum_Interval
	// Multiplier grows exponential base.
	Multiplier Stored_Multiplier
	// Jitter spreads exponential delay.
	Jitter Stored_Jitter
	// Generator supplies deterministic jitter.
	Generator Generator_Storage
}

// Policy_State_Invariants closes kind and bounds interval state.
func Policy_State_Invariants(state Policy_State, namespace invariant.Namespace) {
	Policy_Kind_Invariants(state.Kind, namespace)
	Initial_Interval_Invariants(state.Initial_Interval, namespace)
	Current_Interval_Invariants(state.Current_Interval, namespace)
	Maximum_Interval_Invariants(state.Interval_Max, namespace)
	Stored_Multiplier_Invariants(state.Multiplier, namespace)
	Stored_Jitter_Invariants(state.Jitter, namespace)
	Generator_Storage_Invariants(state.Generator, namespace)
	invariant.Always(
		state.Initial_Interval <= Initial_Interval(state.Interval_Max),
		"Policy initial interval fits maximum.",
	)
}

// Constant initializes caller state and returns fixed-delay policy.
func Constant(state *Policy_State, interval Initial_Interval) (policy Policy) {
	defer func() { Policy_Invariants(policy, "constant.policy") }()
	Policy_State_Invariants(*state, "constant.state")
	Initial_Interval_Invariants(interval, "constant.interval")
	*state = Policy_State{
		Kind: POLICY_KIND_CONSTANT, Initial_Interval: interval,
		Current_Interval: Current_Interval(interval),
		Interval_Max:     Maximum_Interval(interval),
		Multiplier:       Stored_Multiplier{Numerator: 1, Denominator: 1},
		Jitter:           Stored_Jitter{Denominator: 1},
		Generator:        Generator_Storage(prng.New(1)),
	}
	return Policy{State: unsafe.Pointer(state), Next: policy_next, Reset: policy_reset}
}

// Zero returns a Policy that never waits, retrying immediately.
func Zero(state *Policy_State) (policy Policy) {
	defer func() { Policy_Invariants(policy, "zero.policy") }()
	Policy_State_Invariants(*state, "zero.state")
	*state = Policy_State{
		Kind:       POLICY_KIND_CONSTANT,
		Multiplier: Stored_Multiplier{Numerator: 1, Denominator: 1},
		Jitter:     Stored_Jitter{Denominator: 1},
		Generator:  Generator_Storage(prng.New(1)),
	}
	return Policy{State: unsafe.Pointer(state), Next: policy_next, Reset: policy_reset}
}

// Stopped returns a Policy that never retries.
func Stopped(state *Policy_State) (policy Policy) {
	defer func() { Policy_Invariants(policy, "stopped.policy") }()
	Policy_State_Invariants(*state, "stopped.state")
	*state = Policy_State{
		Kind:       POLICY_KIND_STOPPED,
		Multiplier: Stored_Multiplier{Numerator: 1, Denominator: 1},
		Jitter:     Stored_Jitter{Denominator: 1},
		Generator:  Generator_Storage(prng.New(1)),
	}
	return Policy{State: unsafe.Pointer(state), Next: policy_next, Reset: policy_reset}
}

// Exponential_Input configures Exponential. Multiplier and Jitter are integer ratios
// rather than floats so the schedule reproduces bit-for-bit across machines.
type Exponential_Input struct {
	// Initial_Interval is the first interval, before any growth.
	Initial_Interval Initial_Interval
	// Interval_Max caps the growing interval (not the jittered result).
	Interval_Max Maximum_Interval
	// Multiplier is the integer ratio the interval grows by each attempt, e.g. {3,2} is 1.5x.
	Multiplier Multiplier
	// Jitter is the integer ratio of random spread per interval, e.g. {1,2} is half.
	Jitter Jitter
	// Generator is the injected entropy the jitter draws from, for a reproducible spread.
	Generator prng.Generator
}

// Exponential_Input_Invariants bounds configured duration and ratio domains.
func Exponential_Input_Invariants(input Exponential_Input, namespace invariant.Namespace) {
	Initial_Interval_Invariants(input.Initial_Interval, namespace)
	Maximum_Interval_Invariants(input.Interval_Max, namespace)
	Multiplier_Invariants(input.Multiplier, namespace)
	Jitter_Invariants(input.Jitter, namespace)
	prng.Generator_Invariants(input.Generator, namespace)
	invariant.Always(
		input.Initial_Interval <= Initial_Interval(input.Interval_Max),
		"Exponential initial interval fits maximum.",
	)
	exponential_ratio_validate(input.Multiplier, input.Jitter, input.Generator)
}

func exponential_ratio_validate(
	multiplier Multiplier, jitter_ratio Jitter, generator prng.Generator,
) {
	Multiplier_Invariants(multiplier, "exponential_ratio_validate.multiplier")
	Jitter_Invariants(jitter_ratio, "exponential_ratio_validate.jitter_ratio")
	prng.Generator_Invariants(generator, "exponential_ratio_validate.generator")
}

// Exponential initializes caller state and returns growing jittered policy.
func Exponential(state *Policy_State, input *Exponential_Input) (policy Policy) {
	defer func() { Policy_Invariants(policy, "exponential.policy") }()
	Policy_State_Invariants(*state, "exponential.state")
	Exponential_Input_Invariants(*input, "exponential.input")
	*state = Policy_State{
		Kind:             POLICY_KIND_EXPONENTIAL,
		Initial_Interval: input.Initial_Interval,
		Current_Interval: Current_Interval(input.Initial_Interval),
		Interval_Max:     input.Interval_Max,
		Multiplier: Stored_Multiplier{
			Numerator:   Stored_Multiplier_Numerator(input.Multiplier.Numerator),
			Denominator: Stored_Multiplier_Denominator(input.Multiplier.Denominator),
		},
		Jitter: Stored_Jitter{
			Numerator:   Stored_Jitter_Numerator(input.Jitter.Numerator),
			Denominator: Stored_Jitter_Denominator(input.Jitter.Denominator),
		},
		Generator: Generator_Storage(input.Generator),
	}
	return Policy{State: unsafe.Pointer(state), Next: policy_next, Reset: policy_reset}
}

// Policy_Next returns next delay through static procedure table.
func Policy_Next(policy Policy) (delay Delay) {
	defer func() { Delay_Invariants(delay, "policy_next.delay") }()
	Policy_Invariants(policy, "policy_next.policy")
	return policy.Next(policy.State)
}

// Policy_Reset restores concrete state through static procedure table.
func Policy_Reset(policy Policy) {
	Policy_Invariants(policy, "policy_reset.policy")
	policy.Reset(policy.State)
}

// New_Exponential returns an Exponential Policy with the classic defaults: a 500ms
// initial interval, a 60s cap, 1.5x growth, and half-interval jitter from generator.
func New_Exponential(
	state *Policy_State, generator prng.Generator,
) (policy Policy) {
	defer func() { Policy_Invariants(policy, "new_exponential.policy") }()
	Policy_State_Invariants(*state, "new_exponential.state")
	prng.Generator_Invariants(generator, "new_exponential.generator")
	return Exponential(state, &Exponential_Input{
		Initial_Interval: Initial_Interval(DEFAULT_INITIAL_INTERVAL),
		Interval_Max:     Maximum_Interval(DEFAULT_INTERVAL_MAX),
		Multiplier:       Multiplier{Numerator: 3, Denominator: 2},
		Jitter:           Jitter{Numerator: 1, Denominator: 2},
		Generator:        generator,
	})
}

func policy_next(pointer unsafe.Pointer) (delay Delay) {
	defer func() { Delay_Invariants(delay, "policy_procedure_next.delay") }()
	state := (*Policy_State)(pointer)
	switch state.Kind {
	case POLICY_KIND_STOPPED:
		return STOP
	case POLICY_KIND_EXPONENTIAL:
		factor := Jitter{
			Numerator:   Jitter_Numerator(state.Jitter.Numerator),
			Denominator: Jitter_Denominator(state.Jitter.Denominator),
		}
		delay = Delay(jitter(
			state.Current_Interval, factor,
			Required_Generator((*prng.Generator)(&state.Generator)),
		))
		multiplier := Multiplier{
			Numerator:   Multiplier_Numerator(state.Multiplier.Numerator),
			Denominator: Multiplier_Denominator(state.Multiplier.Denominator),
		}
		state.Current_Interval = grow(
			state.Current_Interval, multiplier,
		)
		if state.Current_Interval > Current_Interval(state.Interval_Max) {
			state.Current_Interval = Current_Interval(state.Interval_Max)
		}
		return delay
	default:
		return Delay(state.Initial_Interval)
	}
}

func policy_reset(pointer unsafe.Pointer) {
	state := (*Policy_State)(pointer)
	state.Current_Interval = Current_Interval(state.Initial_Interval)
}

// Multiplies interval by multiplier in integers, the upstream exponential step.
func grow(interval Current_Interval, multiplier Multiplier) (grown Current_Interval) {
	defer func() { Current_Interval_Invariants(grown, "grow.grown") }()
	Current_Interval_Invariants(interval, "grow.interval")
	Multiplier_Invariants(multiplier, "grow.multiplier")
	if multiplier.Denominator == 0 {
		return Current_Interval(INTERVAL_MAXIMUM)
	}
	numerator := uint64(multiplier.Numerator)
	denominator := uint64(multiplier.Denominator)
	if numerator < denominator {
		numerator = denominator
	}
	high, low := bits.Multiply_64(
		bits.Word_64(interval), bits.Multiplier_64(numerator),
	)
	if uint64(high) >= denominator {
		return Current_Interval(INTERVAL_MAXIMUM)
	}
	quotient, _ := bits.Divide_64(
		bits.Dividend_High_64(high), bits.Dividend_Low_64(low),
		bits.Divisor_64(denominator),
	)
	if uint64(quotient) > uint64(INTERVAL_MAXIMUM) {
		return Current_Interval(INTERVAL_MAXIMUM)
	}
	return Current_Interval(quotient)
}

// Spreads interval by a random offset up to factor of itself either way, drawn from
// generator; a zero factor returns interval unchanged.
func jitter(
	interval Current_Interval, factor Jitter, generator Required_Generator,
) (spread Wait) {
	defer func() { Wait_Invariants(spread, "jitter.spread") }()
	Current_Interval_Invariants(interval, "jitter.interval")
	Jitter_Invariants(factor, "jitter.factor")
	Required_Generator_Invariants(generator, "jitter.generator")
	if factor.Numerator == 0 {
		return Wait(interval)
	}
	if factor.Denominator == 0 {
		return Wait(interval)
	}
	numerator := uint64(factor.Numerator)
	denominator := uint64(factor.Denominator)
	if numerator > denominator {
		numerator = denominator
	}
	high, low := bits.Multiply_64(bits.Word_64(interval), bits.Multiplier_64(numerator))
	quotient, _ := bits.Divide_64(
		bits.Dividend_High_64(high), bits.Dividend_Low_64(low),
		bits.Divisor_64(denominator),
	)
	delta := Current_Interval(quotient)
	span := int(2*delta + 1)
	offset := Current_Interval(prng.Generator_Below(generator, prng.Bound(span)))
	value := interval - delta + offset
	if value > Current_Interval(INTERVAL_MAXIMUM) {
		return Wait(INTERVAL_MAXIMUM)
	}
	return Wait(value)
}

// Permanent marks cause as non-retryable, so Retry surfaces it immediately.
func Permanent(storage *Permanent_Error, cause error) (permanent error) {
	Permanent_Error_Invariants(*storage, "permanent.storage")
	defer func() { Permanent_Error_Invariants(*storage, "permanent.error") }()
	storage.Cause = cause
	return storage
}

// Permanent_Error keeps cause without format-owned storage.
type Permanent_Error struct {
	// Cause is original failure.
	Cause error
}

// Permanent_Error_Invariants requires original failure.
func Permanent_Error_Invariants(permanent Permanent_Error, _ invariant.Namespace) {
	invariant.Always((&permanent).Error() != "", "Permanent error has text.")
}

// Error satisfies error without formatting storage.
func (permanent *Permanent_Error) Error() (message string) {
	return "backoff: permanent error"
}

// Retry_After_Error asks Retry to wait a specific Duration before the next attempt,
// overriding the policy for that one step.
type Retry_After_Error struct {
	// Duration is the delay Retry waits before retrying.
	Duration Retry_Delay
	// Cause is the underlying error, reported if retrying ultimately stops.
	Cause error
}

// Retry_After_Error_Invariants bounds override delay and requires original failure.
func Retry_After_Error_Invariants(
	retry_after *Retry_After_Error, namespace invariant.Namespace,
) {
	invariant.Always(retry_after != nil, "Retry-after error has caller storage.")
	Retry_Delay_Invariants(retry_after.Duration, namespace)
}

// Retry_After returns an error that makes Retry wait duration before the next attempt.
func Retry_After(storage *Retry_After_Error, duration Retry_Delay, cause error) (err error) {
	Retry_Delay_Invariants(duration, "retry_after.duration")
	Retry_After_Error_Invariants(storage, "retry_after.storage")
	defer func() { Retry_After_Error_Invariants(storage, "retry_after.error") }()
	storage.Duration = duration
	storage.Cause = cause
	return storage
}

// Error renders the retry-after error in nanoseconds, satisfying the error interface.
func (retry_after *Retry_After_Error) Error() (message string) {
	return "backoff: retry after"
}

// Exhausted_Error keeps final failure without format-owned storage.
type Exhausted_Error struct {
	// Cause is final failure.
	Cause error
}

// Exhausted_Error_Invariants requires final failed attempt.
func Exhausted_Error_Invariants(exhausted Exhausted_Error, _ invariant.Namespace) {
	invariant.Always((&exhausted).Error() != "", "Exhausted error has text.")
}

// Error satisfies error without formatting storage.
func (exhausted *Exhausted_Error) Error() (message string) {
	return "backoff: retries exhausted"
}

// Elapsed_Max_Error keeps final failure without format-owned storage.
type Elapsed_Limit_Error struct {
	// Cause is final failure.
	Cause error
}

// Elapsed_Error_Max_Invariants requires final failure.
func Elapsed_Limit_Error_Invariants(elapsed Elapsed_Limit_Error, _ invariant.Namespace) {
	invariant.Always((&elapsed).Error() != "", "Elapsed error has text.")
}

// Error satisfies error without formatting storage.
func (elapsed *Elapsed_Limit_Error) Error() (message string) {
	return "backoff: maximum elapsed time exceeded"
}

// Error_Matches checks package sentinels and stored cause without allocating wrapper chains.
func Error_Matches(err error, target error) (matched Boolean) {
	defer func() { Boolean_Invariants(matched, "error_matches.matched") }()
	switch typed := err.(type) {
	case *Permanent_Error:
		return Boolean(target == Error_Permanent || errors.Is(typed.Cause, target))
	case *Exhausted_Error:
		return Boolean(target == Error_Exhausted || errors.Is(typed.Cause, target))
	case *Elapsed_Limit_Error:
		return Boolean(target == Error_Elapsed_Max || errors.Is(typed.Cause, target))
	case *Retry_After_Error:
		return Boolean(errors.Is(typed.Cause, target))
	default:
		return Boolean(errors.Is(err, target))
	}
}

// Operation is the function Retry attempts. Return nil to succeed, a Permanent error
// to stop at once, or a Retry_After error to override the next delay.
type Operation[T any] func(state *Retry_State[T]) (result T, err error)

// Try_Count counts bounded attempts.
type Try_Count uint

// Try_Count_Invariants bounds attempt count to shared collection capacity.
func Try_Count_Invariants(count Try_Count, namespace invariant.Namespace) {
	invariant.Tree(count, namespace).
		Range_Uint(uint(count), TRY_COUNT_MINIMUM_UINT, TRY_COUNT_MAXIMUM_UINT).
		Ensure()
}

// TRY_COUNT_MINIMUM is one initial attempt.
const TRY_COUNT_MINIMUM Try_Count = 1

// TRY_COUNT_MAXIMUM bounds attempts by shared collection capacity.
const TRY_COUNT_MAXIMUM = Try_Count(slices.SLICE_COUNT_MAXIMUM)

// COUNT_MINIMUM_UINT starts attempt progress.
const COUNT_MINIMUM_UINT uint = 0

// TRY_COUNT_MAXIMUM_UINT supplies canonical range bound.
const TRY_COUNT_MAXIMUM_UINT uint = uint(TRY_COUNT_MAXIMUM)

// TRY_COUNT_MINIMUM_UINT supplies canonical range bound.
const TRY_COUNT_MINIMUM_UINT uint = uint(TRY_COUNT_MINIMUM)

// Attempt_Count counts attempts already made.
type Attempt_Count uint

// Attempt_Count_Invariants bounds progress by attempt capacity.
func Attempt_Count_Invariants(count Attempt_Count, namespace invariant.Namespace) {
	invariant.Tree(count, namespace).
		Range_Uint(uint(count), COUNT_MINIMUM_UINT, TRY_COUNT_MAXIMUM_UINT).
		Ensure()
}

// Elapsed_Limit bounds complete retry timeline.
type Elapsed_Limit time.Duration

// Elapsed_Limit_Invariants keeps retry timeline inside process uptime.
func Elapsed_Limit_Invariants(limit Elapsed_Limit, namespace invariant.Namespace) {
	invariant.Tree(limit, namespace).
		Range_Int64(int64(limit), INTERVAL_MINIMUM_INT64, INTERVAL_MAXIMUM_INT64).
		Ensure()
}

// Started_Moment stores bounded retry origin.
type Started_Moment time.Monotonic_Moment

// Started_Moment_Invariants keeps retry origin inside process uptime.
func Started_Moment_Invariants(started Started_Moment, namespace invariant.Namespace) {
	invariant.Tree(started, namespace).
		Range_Int64(int64(started), INTERVAL_MINIMUM_INT64, INTERVAL_MAXIMUM_INT64).
		Ensure()
}

// Retry_Input configures Retry. It holds timeline submission, never Driver.
type Retry_Input struct {
	// Timer submits between-attempt wait on injected timeline.
	Timer time.Timeline
	// Policy computes the delay before each retry.
	Policy Policy
	// Tries_Max bounds total attempts; must be positive (the house bans unbounded loops).
	Tries_Max Try_Count
	// Clock measures elapsed time for Elapsed_Time_Max; required only when that is set.
	Clock time.Clock
	// Elapsed_Time_Max stops retrying once the next wait would exceed it; zero disables it.
	Elapsed_Time_Max Elapsed_Limit
	// Notify, when set, is called after each failed attempt that will be retried.
	Notify func(context unsafe.Pointer, err error, delay Delay)
}

// Retry_Input_Invariants bounds total attempts.
func Retry_Input_Invariants(input Retry_Input, namespace invariant.Namespace) {
	time.Timeline_Invariants(input.Timer, namespace)
	Policy_Invariants(input.Policy, namespace)
	Try_Count_Invariants(input.Tries_Max, namespace)
	time.Clock_Invariants(input.Clock, namespace)
	Elapsed_Limit_Invariants(input.Elapsed_Time_Max, namespace)
}

// Retry_State is caller-owned storage retained across timer completion.
type Retry_State[T any] struct {
	// Completion retains timer lifecycle and records when root has work to rearm.
	Completion time.Completion
	// Input retains dependencies across asynchronous attempts.
	Input Retry_Input
	// Operation is caller procedure retried.
	Operation Operation[T]
	// Context carries caller state into Operation and Notify.
	Context unsafe.Pointer
	// Result retains terminal operation value for root.
	Result T
	// Result_Error retains terminal operation failure for root.
	Result_Error error
	// Stopped reports terminal state.
	Stopped Boolean
	// Try_Count is attempts already made.
	Try_Count Attempt_Count
	// Started is monotonic origin for elapsed bound.
	Started Started_Moment
	// Exhausted is caller-owned terminal error storage.
	Exhausted Exhausted_Error
	// Elapsed is caller-owned elapsed terminal error storage.
	Elapsed Elapsed_Limit_Error
}

// Retry_State_Invariants keeps attempt count inside configured bound.
func Retry_State_Invariants[T any](state *Retry_State[T], namespace invariant.Namespace) {
	invariant.Always(state != nil, "Retry has caller state.")
	Retry_Input_Invariants(state.Input, namespace)
	Boolean_Invariants(state.Stopped, namespace)
	Attempt_Count_Invariants(state.Try_Count, namespace)
	Started_Moment_Invariants(state.Started, namespace)
	Exhausted_Error_Invariants(state.Exhausted, namespace)
	Elapsed_Limit_Error_Invariants(state.Elapsed, namespace)
	invariant.Always(
		state.Try_Count <= Attempt_Count(state.Input.Tries_Max),
		"Retry state attempt count stays within configured bound.",
	)
}

// Retry initializes runner and executes first attempt. Later waits only record work;
// root calls Retry_Rearm after driving timeline.
func Retry[T any](
	state *Retry_State[T], operation Operation[T],
) {
	Retry_State_Invariants(state, "retry.state")
	invariant.Always(operation != nil, "Retry has operation.")
	invariant.Always(!state.Completion.Armed, "Retry state has no armed wait.")
	state.Operation = operation
	var zero T
	state.Result = zero
	state.Result_Error = nil
	state.Stopped = false
	state.Try_Count = 0
	state.Started = 0
	state.Exhausted = Exhausted_Error{}
	state.Elapsed = Elapsed_Limit_Error{}
	state.Completion.Data = RETRY_WORK_READY
	Policy_Reset(state.Input.Policy)
	if state.Input.Elapsed_Time_Max > 0 {
		state.Started = Started_Moment(time.Clock_Now_Monotonic(state.Input.Clock))
	}
	Retry_Rearm(state)
}

// Retry_Rearm executes one continuation recorded by completed wait.
func Retry_Rearm[T any](state *Retry_State[T]) (rearmed Boolean) {
	defer func() { Boolean_Invariants(rearmed, "retry_rearm.rearmed") }()
	Retry_State_Invariants(state, "retry_rearm.state")
	if state.Stopped {
		return false
	}
	if state.Completion.Data != RETRY_WORK_READY {
		return false
	}
	invariant.Always(state.Operation != nil, "Active Retry has operation.")
	state.Completion.Data = RETRY_WORK_IDLE
	result, attempt_error := state.Operation(state)
	state.Try_Count++
	if attempt_error == nil {
		state.Result, state.Result_Error, state.Stopped = result, nil, true
		return true
	}
	if Error_Matches(attempt_error, Error_Permanent) {
		state.Result, state.Result_Error, state.Stopped = result, attempt_error, true
		return true
	}
	if state.Try_Count >= Attempt_Count(state.Input.Tries_Max) {
		state.Exhausted.Cause = attempt_error
		state.Result, state.Result_Error, state.Stopped = result, &state.Exhausted, true
		return true
	}
	delay := Policy_Next(state.Input.Policy)
	retry_after, has_retry_after := attempt_error.(*Retry_After_Error)
	if has_retry_after {
		delay = Delay(retry_after.Duration)
		Policy_Reset(state.Input.Policy)
	}
	if delay == STOP {
		state.Exhausted.Cause = attempt_error
		state.Result, state.Result_Error, state.Stopped = result, &state.Exhausted, true
		return true
	}
	wait_started := time.Clock_Now_Monotonic(state.Input.Clock)
	if state.Try_Count > 1 {
		wait_started = state.Completion.Ready_At
	}
	if state.Input.Elapsed_Time_Max > 0 {
		elapsed := time.Duration(wait_started - time.Monotonic_Moment(state.Started))
		if elapsed+time.Duration(delay) > time.Duration(state.Input.Elapsed_Time_Max) {
			state.Elapsed.Cause = attempt_error
			state.Result, state.Result_Error, state.Stopped =
				result, &state.Elapsed, true
			return true
		}
	}
	if state.Input.Notify != nil {
		state.Input.Notify(state.Context, attempt_error, Delay(delay))
	}
	// Root rearm follows driver return. Remove that handoff so virtual resolution cannot
	// become hidden backoff between attempts.
	now := time.Clock_Now_Monotonic(state.Input.Clock)
	invariant.Always(now >= wait_started, "Retry wait starts on monotonic past.")
	duration := time.Duration(delay)
	late := time.Duration(now - wait_started)
	if duration <= late {
		state.Completion.Data = RETRY_WORK_READY
		return true
	}
	time.Timeline_Timeout(
		state.Input.Timer, &state.Completion, duration-late, retry_wait_complete,
	)
	return true
}

// Retry_Work_Queued reports completed wait root must rearm.
func Retry_Work_Queued[T any](state *Retry_State[T]) (queued Boolean) {
	defer func() { Boolean_Invariants(queued, "retry_work_queued.queued") }()
	Retry_State_Invariants(state, "retry_work_queued.state")
	return Boolean(!state.Stopped && state.Completion.Data == RETRY_WORK_READY)
}

// Retry_Stopped reports terminal result is ready.
func Retry_Stopped[T any](state *Retry_State[T]) (stopped Boolean) {
	defer func() { Boolean_Invariants(stopped, "retry_stopped.stopped") }()
	Retry_State_Invariants(state, "retry_stopped.state")
	return state.Stopped
}

// Retry_Status reports progress and retained result without confusing in-flight nil with success.
func Retry_Status[T any](state *Retry_State[T]) (
	stopped Boolean, result T, err error,
) {
	defer func() { Boolean_Invariants(stopped, "retry_status.stopped") }()
	Retry_State_Invariants(state, "retry_status.state")
	return state.Stopped, state.Result, state.Result_Error
}

// RETRY_WORK_IDLE means no completed wait awaits root.
const RETRY_WORK_IDLE = 0

// RETRY_WORK_READY means root must call Retry_Rearm.
const RETRY_WORK_READY = 1

func retry_wait_complete(completion *time.Completion) {
	completion.Data = RETRY_WORK_READY
}
