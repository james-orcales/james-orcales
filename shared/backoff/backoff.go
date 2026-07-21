// Package backoff computes retry delays and drives retries on the shared/io loop.
// It is a dependency-injected port of github.com/cenkalti/backoff: the upstream
// package is float-based (Multiplier 1.5, RandomizationFactor 0.5), draws jitter
// from the global math/rand, and waits on a real time.Timer under a context — none
// of which the house linter allows and deterministic simulation cannot replay. Here
// the growth factor and jitter are integer prng.Ratios, the jitter entropy is an
// injected prng.Generator, and durations are shared/time.Duration, so a schedule
// reproduces bit-for-bit from a seed.
//
// The linter bans user interfaces, so the upstream BackOff interface becomes a
// struct of closures, Policy, chosen by value — the house vtable, like time.Clock.
// It also forbids a library from holding the io.Driver (only a harness drives the
// loop), so Retry is submit-only: it runs an attempt, and on failure arms an
// io.Timeout that runs the next attempt when the harness next pumps the loop,
// delivering the outcome to a callback rather than blocking for a return value.
package backoff

import (
	"errors"
	"fmt"

	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/io"
	"local/james-orcales/shared/random/prng"
	"local/james-orcales/shared/time"
)

// STOP is the delay a Policy's Next returns to signal that no more retries should be made.
const STOP time.Duration = -1

// DEFAULT_INITIAL_INTERVAL is the first delay New_Exponential grows from.
const DEFAULT_INITIAL_INTERVAL = 500 * time.MILLISECOND

// DEFAULT_INTERVAL_MAX caps the growing interval New_Exponential produces.
const DEFAULT_INTERVAL_MAX = 60 * time.SECOND

// Error_Permanent is the sentinel Permanent wraps; Retry stops at once when the
// operation's error matches it.
var Error_Permanent = errors.New("backoff: permanent error")

// Error_Exhausted is the sentinel Retry returns when it runs out of tries or the policy stops.
var Error_Exhausted = errors.New("backoff: retries exhausted")

// Error_Elapsed_Max is the sentinel Retry returns when the next wait would exceed Elapsed_Time_Max.
var Error_Elapsed_Max = errors.New("backoff: maximum elapsed time exceeded")

// Policy is a retry-delay strategy expressed as a struct of closures — the house
// vtable that replaces the upstream BackOff interface. Construct one with Constant,
// Zero, Stopped, or Exponential; the zero Policy is unusable.
type Policy struct {
	// Next returns the delay before the next attempt, or STOP to give up.
	Next func() (delay time.Duration)
	// Reset returns the policy to its initial state; call it before reuse.
	Reset func()
}

// Constant returns a Policy that always waits interval.
func Constant(interval time.Duration) (policy Policy) {
	return Policy{
		Next:  func() (delay time.Duration) { return interval },
		Reset: func() {},
	}
}

// Zero returns a Policy that never waits, retrying immediately.
func Zero() (policy Policy) {
	return Policy{
		Next:  func() (delay time.Duration) { return 0 },
		Reset: func() {},
	}
}

// Stopped returns a Policy that never retries.
func Stopped() (policy Policy) {
	return Policy{
		Next:  func() (delay time.Duration) { return STOP },
		Reset: func() {},
	}
}

// Exponential_Input configures Exponential. Multiplier and Jitter are integer ratios
// rather than floats so the schedule reproduces bit-for-bit across machines.
type Exponential_Input struct {
	// Initial_Interval is the first interval, before any growth.
	Initial_Interval time.Duration
	// Interval_Max caps the growing interval (not the jittered result).
	Interval_Max time.Duration
	// Multiplier is the integer ratio the interval grows by each attempt, e.g. {3,2} is 1.5x.
	Multiplier prng.Ratio
	// Jitter is the integer ratio of random spread per interval, e.g. {1,2} is half.
	Jitter prng.Ratio
	// Generator is the injected entropy the jitter draws from, for a reproducible spread.
	Generator *prng.Generator
}

// Exponential returns a Policy whose interval grows by Multiplier each attempt,
// capped at Interval_Max, with Jitter random spread drawn from Generator.
func Exponential(input *Exponential_Input) (policy Policy) {
	current := input.Initial_Interval
	return Policy{
		Next: func() (delay time.Duration) {
			spread := jitter(current, input.Jitter, input.Generator)
			current = grow(current, input.Multiplier)
			if current > input.Interval_Max {
				current = input.Interval_Max
			}
			return spread
		},
		Reset: func() {
			current = input.Initial_Interval
		},
	}
}

// New_Exponential returns an Exponential Policy with the classic defaults: a 500ms
// initial interval, a 60s cap, 1.5x growth, and half-interval jitter from generator.
func New_Exponential(generator *prng.Generator) (policy Policy) {
	return Exponential(&Exponential_Input{
		Initial_Interval: DEFAULT_INITIAL_INTERVAL,
		Interval_Max:     DEFAULT_INTERVAL_MAX,
		Multiplier:       prng.Ratio{Numerator: 3, Denominator: 2},
		Jitter:           prng.Ratio{Numerator: 1, Denominator: 2},
		Generator:        generator,
	})
}

// Multiplies interval by multiplier in integers, the upstream exponential step.
func grow(interval time.Duration, multiplier prng.Ratio) (grown time.Duration) {
	numerator := time.Duration(multiplier.Numerator)
	denominator := time.Duration(multiplier.Denominator)
	return interval * numerator / denominator
}

// Spreads interval by a random offset up to factor of itself either way, drawn from
// generator; a zero factor returns interval unchanged.
func jitter(
	interval time.Duration, factor prng.Ratio, generator *prng.Generator,
) (spread time.Duration) {
	if factor.Numerator == 0 {
		return interval
	}
	delta := interval * time.Duration(factor.Numerator) / time.Duration(factor.Denominator)
	span := int(2*delta + 1)
	offset := time.Duration(prng.Generator_Below(generator, span))
	return interval - delta + offset
}

// Permanent marks cause as non-retryable, so Retry surfaces it immediately.
func Permanent(cause error) (permanent error) {
	return fmt.Errorf("%w: %w", Error_Permanent, cause)
}

// Retry_After_Error asks Retry to wait a specific Duration before the next attempt,
// overriding the policy for that one step.
type Retry_After_Error struct {
	// Duration is the delay Retry waits before retrying.
	Duration time.Duration
	// Cause is the underlying error, reported if retrying ultimately stops.
	Cause error
}

// Retry_After returns an error that makes Retry wait duration before the next attempt.
func Retry_After(duration time.Duration, cause error) (err error) {
	return Retry_After_Error{Duration: duration, Cause: cause}
}

// Error renders the retry-after error in nanoseconds, satisfying the error interface.
func (retry_after Retry_After_Error) Error() (message string) {
	return fmt.Sprintf(
		"backoff: retry after %d ns: %v", int64(retry_after.Duration), retry_after.Cause)
}

// Operation is the function Retry attempts. Return nil to succeed, a Permanent error
// to stop at once, or a Retry_After error to override the next delay.
type Operation[T any] func() (result T, err error)

// Retry_Input configures Retry. It holds an io.IO to submit the between-attempt wait,
// never an io.Driver — a library submits IO but only a harness drives the loop.
type Retry_Input struct {
	// Timer submits the between-attempt wait (io's Sleep equivalent).
	Timer *io.IO
	// Policy computes the delay before each retry.
	Policy Policy
	// Tries_Max bounds total attempts; must be positive (the house bans unbounded loops).
	Tries_Max uint
	// Clock measures elapsed time for Elapsed_Time_Max; required only when that is set.
	Clock time.Clock
	// Elapsed_Time_Max stops retrying once the next wait would exceed it; zero disables it.
	Elapsed_Time_Max time.Duration
	// Notify, when set, is called after each failed attempt that will be retried.
	Notify func(err error, delay time.Duration)
}

// Retry attempts operation up to Tries_Max times and delivers the outcome to done:
// the first success, a Permanent error at once, or an exhausted or elapsed error
// carrying the last failure. The first attempt runs synchronously; each later attempt
// is armed as an io.Timeout that fires when the harness next drives the loop, so the
// wait rides the io timeline without this library ever holding the Driver.
func Retry[T any](input *Retry_Input, operation Operation[T], done func(result T, err error)) {
	invariant.Always(input.Tries_Max > 0, "backoff retry tries max is positive")
	input.Policy.Reset()
	started := time.Moment(0)
	if input.Elapsed_Time_Max > 0 {
		started = input.Clock.Now_Monotonic()
	}
	var completion io.Completion
	try_index := uint(0)
	var attempt io.Timeout_Callback
	attempt = func(*io.Completion, error) {
		result, attempt_error := operation()
		try_index++
		if attempt_error == nil {
			done(result, nil)
			return
		}
		if errors.Is(attempt_error, Error_Permanent) {
			done(result, attempt_error)
			return
		}
		if try_index >= input.Tries_Max {
			done(result, fmt.Errorf("%w: %w", Error_Exhausted, attempt_error))
			return
		}
		delay := input.Policy.Next()
		var retry_after Retry_After_Error
		if errors.As(attempt_error, &retry_after) {
			delay = retry_after.Duration
			input.Policy.Reset()
		}
		if delay == STOP {
			done(result, fmt.Errorf("%w: %w", Error_Exhausted, attempt_error))
			return
		}
		if input.Elapsed_Time_Max > 0 {
			elapsed := time.Duration(input.Clock.Now_Monotonic() - started)
			if elapsed+delay > input.Elapsed_Time_Max {
				done(result, fmt.Errorf("%w: %w", Error_Elapsed_Max, attempt_error))
				return
			}
		}
		if input.Notify != nil {
			input.Notify(attempt_error, delay)
		}
		input.Timer.Timeout(&completion, attempt, delay)
	}
	attempt(nil, nil)
}
