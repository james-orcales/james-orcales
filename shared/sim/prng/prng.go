// Package prng is a deterministic, integer-only pseudo-random number generator.
//
// It mirrors TigerBeetle's stdx.PRNG: a xoshiro256++ generator seeded by splitmix64, with no
// floating point anywhere in its surface. A probability is an integer Ratio, never a float, so a
// run reproduces bit-for-bit across machines from one seed — the property deterministic simulation
// testing is built on, where a failure replays exactly from the seed that found it.
//
// The house linter bans methods, so each draw is a free function named after its first parameter's
// type. Seed with New; the zero Xoshiro is unusable.
//
//	generator := prng.New(seed)
//	victim := prng.Xoshiro_Below(&generator, replica_count)
//	if prng.Xoshiro_Chance(&generator, prng.Ratio{Numerator: 8, Denominator: 100}) {
//	    drop_packet()
//	}
//
// xoshiro256++ and splitmix64 are public domain (Blackman and Vigna).
package prng

import (
	"local/james-orcales/shared/crypto/prng"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
)

// The splitmix64 increment, derived from the golden ratio, strides the seed state.
const SPLIT_MIX_INCREMENT = 0x9e3779b97f4a7c15

// The first splitmix64 multiplier that avalanches the strided state.
const SPLIT_MIX_MULTIPLIER_FIRST = 0xbf58476d1ce4e5b9

// The second splitmix64 multiplier that avalanches the strided state.
const SPLIT_MIX_MULTIPLIER_SECOND = 0x94d049bb133111eb

// DISTRIBUTION_COUNT_MAXIMUM keeps weighted sampling linear and stack-owned.
const DISTRIBUTION_COUNT_MAXIMUM = 32

// PERMUTATION_COUNT_MAXIMUM shares the one collection width used by this package.
const PERMUTATION_COUNT_MAXIMUM = DISTRIBUTION_COUNT_MAXIMUM

// PERMUTATION_COUNT_MINIMUM admits an empty shuffle.
const PERMUTATION_COUNT_MINIMUM = 0

// SEED_MINIMUM keeps every deterministic replay seed available.
const SEED_MINIMUM Seed = Seed(bits.WORD_64_MINIMUM)

// SEED_MAXIMUM keeps every deterministic replay seed available.
const SEED_MAXIMUM Seed = Seed(bits.WORD_64_MAXIMUM)

// WORD_MINIMUM preserves the complete xoshiro output domain.
const WORD_MINIMUM Word = Word(bits.WORD_64_MINIMUM)

// WORD_MAXIMUM preserves the complete xoshiro output domain.
const WORD_MAXIMUM Word = Word(bits.WORD_64_MAXIMUM)

// WEIGHT_MINIMUM permits an outcome that must never be selected.
const WEIGHT_MINIMUM Weight = Weight(bits.WORD_64_MINIMUM)

// WEIGHT_MAXIMUM prevents a narrower accidental cumulative domain.
const WEIGHT_MAXIMUM Weight = Weight(bits.WORD_64_MAXIMUM)

// RATIO_DENOMINATOR_MINIMUM prevents probability division by zero.
const RATIO_DENOMINATOR_MINIMUM Ratio_Denominator = 1

// RATIO_DENOMINATOR_MAXIMUM keeps every positive probability total available.
const RATIO_DENOMINATOR_MAXIMUM Ratio_Denominator = Ratio_Denominator(WEIGHT_MAXIMUM)

// BOUND_MINIMUM prevents division by zero in rejection sampling.
const BOUND_MINIMUM Bound = 1

// BOUND_MAXIMUM keeps each result representable as an Index.
const BOUND_MAXIMUM Bound = Bound(bits.INTEGER_MAXIMUM)

// INDEX_MINIMUM matches every half-open draw range.
const INDEX_MINIMUM Index = 0

// INDEX_MAXIMUM is one below the largest accepted bound.
const INDEX_MAXIMUM Index = Index(bits.INTEGER_MAXIMUM - 1)

// DRAW_BOUND_MINIMUM prevents rejection-sampling division by zero.
const DRAW_BOUND_MINIMUM Draw_Bound = 1

// DRAW_BOUND_MAXIMUM keeps the complete unsigned bound domain.
const DRAW_BOUND_MAXIMUM Draw_Bound = Draw_Bound(bits.WORD_64_MAXIMUM)

// DRAW_MINIMUM matches every half-open unsigned draw range.
const DRAW_MINIMUM Draw = 0

// DRAW_MAXIMUM is one below the largest accepted unsigned bound.
const DRAW_MAXIMUM Draw = Draw(bits.WORD_64_MAXIMUM - 1)

// DISTRIBUTION_COUNT_MINIMUM keeps the total bucket accessible.
const DISTRIBUTION_COUNT_MINIMUM = 1

// Seed names replay identity separately from generated output.
type Seed uint64

// Seed_Invariants preserves all replay identities.
func Seed_Invariants(seed Seed, namespace aver.Namespace) {
	aver.Tree(seed, namespace).
		Range_Uint64(uint64(seed), uint64(SEED_MINIMUM), uint64(SEED_MAXIMUM)).
		Ensure()
}

// Word names raw generator state and output separately from weights.
type Word uint64

// Word_Invariants preserves every xoshiro output.
func Word_Invariants(word Word, namespace aver.Namespace) {
	aver.Tree(word, namespace).
		Range_Uint64(uint64(word), uint64(WORD_MINIMUM), uint64(WORD_MAXIMUM)).
		Ensure()
}

// Weight separates probability mass from random output.
type Weight uint64

// Weight_Invariants permits zero-mass buckets without narrowing totals.
func Weight_Invariants(weight Weight, namespace aver.Namespace) {
	aver.Tree(weight, namespace).
		Range_Uint64(uint64(weight), uint64(WEIGHT_MINIMUM), uint64(WEIGHT_MAXIMUM)).
		Ensure()
}

// Ratio_Numerator prevents two probability fields from sharing one invariant subject.
type Ratio_Numerator Weight

// Ratio_Numerator_Invariants preserves zero through complete favorable mass.
func Ratio_Numerator_Invariants(value Ratio_Numerator, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), uint64(WEIGHT_MINIMUM), uint64(WEIGHT_MAXIMUM)).
		Ensure()
}

// Ratio_Denominator prevents two probability fields from sharing one invariant subject.
type Ratio_Denominator Weight

// Ratio_Denominator_Invariants preserves every representable total mass.
func Ratio_Denominator_Invariants(value Ratio_Denominator, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(
			uint64(value),
			uint64(RATIO_DENOMINATOR_MINIMUM),
			uint64(RATIO_DENOMINATOR_MAXIMUM),
		).
		Ensure()
}

// Bound makes the positive exclusive draw limit explicit.
type Bound int

// Bound_Invariants rejects zero before unsigned rejection sampling.
func Bound_Invariants(bound Bound, namespace aver.Namespace) {
	aver.Tree(bound, namespace).
		Range_Int(int(bound), int(BOUND_MINIMUM), int(BOUND_MAXIMUM)).
		Ensure()
}

// Index keeps a draw representable by Go slice indexing.
type Index int

// Index_Invariants preserves every result below the largest Bound.
func Index_Invariants(index Index, namespace aver.Namespace) {
	aver.Tree(index, namespace).
		Range_Int(int(index), int(INDEX_MINIMUM), int(INDEX_MAXIMUM)).
		Ensure()
}

// Draw_Bound separates positive unsigned limits from zero-weight buckets.
type Draw_Bound uint64

// Draw_Bound_Invariants preserves every positive unsigned limit.
func Draw_Bound_Invariants(bound Draw_Bound, namespace aver.Namespace) {
	aver.Tree(bound, namespace).
		Range_Uint64(
			uint64(bound), uint64(DRAW_BOUND_MINIMUM), uint64(DRAW_BOUND_MAXIMUM),
		).
		Ensure()
}

// Draw is the half-open result of one unsigned bounded draw.
type Draw uint64

// Draw_Invariants excludes the unreachable largest word.
func Draw_Invariants(draw Draw, namespace aver.Namespace) {
	aver.Tree(draw, namespace).
		Range_Uint64(uint64(draw), uint64(DRAW_MINIMUM), uint64(DRAW_MAXIMUM)).
		Ensure()
}

// Boolean lets both random decisions become coverage obligations.
type Boolean bool

// Boolean_Invariants requires both decision branches across package runs.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A random decision is true.").
		Ensure()
}

// Permutation is caller-owned index storage a shuffle reorders in place. A slice, not a fixed
// array, because the house rule keeps every aggregate caller-sized; the width bound is what
// keeps a shuffle's work finite.
type Permutation []Index

// Permutation_Invariants bounds shuffle work.
func Permutation_Invariants(permutation Permutation, namespace aver.Namespace) {
	aver.Tree(permutation, namespace).
		Range_Int(len(permutation), PERMUTATION_COUNT_MINIMUM, PERMUTATION_COUNT_MAXIMUM).
		Ensure()
}

// Outcomes are the values a Distribution draws from, in the caller's own unit.
type Outcomes []Word

// Outcomes_Invariants keeps weighted sampling linear.
func Outcomes_Invariants(outcomes Outcomes, namespace aver.Namespace) {
	aver.Tree(outcomes, namespace).
		Range_Int(len(outcomes), DISTRIBUTION_COUNT_MINIMUM, DISTRIBUTION_COUNT_MAXIMUM).
		Ensure()
}

// Weights is caller-owned probability mass, one entry per outcome. New_Distribution turns it
// cumulative in place, thus the caller's storage is the table.
type Weights []Weight

// Weights_Invariants keeps weighted sampling linear.
func Weights_Invariants(weights Weights, namespace aver.Namespace) {
	aver.Tree(weights, namespace).
		Range_Int(len(weights), DISTRIBUTION_COUNT_MINIMUM, DISTRIBUTION_COUNT_MAXIMUM).
		Ensure()
}

// Xoshiro_First is s[0] of xoshiro256++. Each state word has its own type because one chain
// holds each type one time, and a generator holds four words.
type Xoshiro_First Word

// Xoshiro_First_Invariants preserves the complete word domain. The witnesses this obliges
// under New come from inverting splitmix64 in the test: each step is a bijection.
func Xoshiro_First_Invariants(value Xoshiro_First, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), uint64(WORD_MINIMUM), uint64(WORD_MAXIMUM)).
		Ensure()
}

// Xoshiro_Second is s[1] of xoshiro256++.
type Xoshiro_Second Word

// Xoshiro_Second_Invariants preserves the complete word domain. The witnesses this obliges
// under New come from inverting splitmix64 in the test: each step is a bijection.
func Xoshiro_Second_Invariants(value Xoshiro_Second, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), uint64(WORD_MINIMUM), uint64(WORD_MAXIMUM)).
		Ensure()
}

// Xoshiro_Third is s[2] of xoshiro256++.
type Xoshiro_Third Word

// Xoshiro_Third_Invariants preserves the complete word domain. The witnesses this obliges
// under New come from inverting splitmix64 in the test: each step is a bijection.
func Xoshiro_Third_Invariants(value Xoshiro_Third, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), uint64(WORD_MINIMUM), uint64(WORD_MAXIMUM)).
		Ensure()
}

// Xoshiro_Fourth is s[3] of xoshiro256++.
type Xoshiro_Fourth Word

// Xoshiro_Fourth_Invariants preserves the complete word domain. The witnesses this obliges
// under New come from inverting splitmix64 in the test: each step is a bijection.
func Xoshiro_Fourth_Invariants(value Xoshiro_Fourth, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), uint64(WORD_MINIMUM), uint64(WORD_MAXIMUM)).
		Ensure()
}

// Xoshiro is the state of a xoshiro256++ pseudo-random generator: the four words s[0] through
// s[3] of the reference, as four fields because the house rule keeps a small aggregate as named
// members. Construct it with New; the zero value is degenerate, since an all-zero xoshiro state
// emits only zeros.
type Xoshiro struct {
	// First is s[0].
	First Xoshiro_First
	// Second is s[1].
	Second Xoshiro_Second
	// Third is s[2].
	Third Xoshiro_Third
	// Fourth is s[3].
	Fourth Xoshiro_Fourth
}

// Xoshiro_Invariants rejects xoshiro's absorbing all-zero state.
func Xoshiro_Invariants(generator Xoshiro, namespace aver.Namespace) {
	Xoshiro_First_Invariants(generator.First, namespace)
	Xoshiro_Second_Invariants(generator.Second, namespace)
	Xoshiro_Third_Invariants(generator.Third, namespace)
	Xoshiro_Fourth_Invariants(generator.Fourth, namespace)
	aver.Always(generator != Xoshiro{}, "A generator has nonzero xoshiro state.")
}

// Xoshiro_Pointer names caller-owned generator storage: every draw advances it in place.
type Xoshiro_Pointer *Xoshiro

// Xoshiro_Pointer_Invariants admits absent storage; each draw asserts presence.
func Xoshiro_Pointer_Invariants(value Xoshiro_Pointer, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Xoshiro_Invariants(*value, namespace)
}

// Ratio is an integer probability, used instead of a float so a run reproduces bit-for-bit.
type Ratio struct {
	// Numerator is the count of favorable outcomes; it must not exceed Denominator.
	Numerator Ratio_Numerator
	// Denominator is the total count of outcomes; it must be positive.
	Denominator Ratio_Denominator
}

// Ratio_Invariants rejects division by zero and probability above one.
func Ratio_Invariants(ratio Ratio, namespace aver.Namespace) {
	Ratio_Numerator_Invariants(ratio.Numerator, namespace)
	Ratio_Denominator_Invariants(ratio.Denominator, namespace)
	aver.Always(ratio.Denominator > 0, "A probability denominator is positive.")
	aver.Always(
		Weight(ratio.Numerator) <= Weight(ratio.Denominator),
		"A probability numerator does not exceed its denominator.",
	)
}

// Distribution is a set of weighted outcomes Sample draws from, with the cumulative weights
// precomputed. Sample finds the bucket by a linear scan, so keep the entry count to 32 or fewer.
type Distribution struct {
	// Outcomes are the values Sample may return, positionally paired with Cumulative.
	Outcomes Outcomes
	// Cumulative is the running sum of each outcome's weight; the final entry is the total.
	Cumulative Weights
}

// Distribution_Invariants keeps the live table nonempty, paired, and drawable.
func Distribution_Invariants(distribution Distribution, namespace aver.Namespace) {
	Outcomes_Invariants(distribution.Outcomes, namespace)
	Weights_Invariants(distribution.Cumulative, namespace)
	aver.Always(
		len(distribution.Outcomes) == len(distribution.Cumulative),
		"A distribution pairs each outcome with one cumulative weight.",
	)
	aver.Always(
		distribution.Cumulative[len(distribution.Cumulative)-1] > 0,
		"A distribution has positive total weight.",
	)
}

// New seeds a Xoshiro from one seed, expanding it through splitmix64 into the four words of
// xoshiro256++ state. The zero Xoshiro is degenerate, so always construct through New.
func New(seed Seed) (generator Xoshiro) {
	defer func() { Xoshiro_Invariants(generator, "new.generator") }()
	Seed_Invariants(seed, "new.seed")
	state := seed + SPLIT_MIX_INCREMENT
	generator.First = Xoshiro_First(split_mix(state))
	state += SPLIT_MIX_INCREMENT
	generator.Second = Xoshiro_Second(split_mix(state))
	state += SPLIT_MIX_INCREMENT
	generator.Third = Xoshiro_Third(split_mix(state))
	state += SPLIT_MIX_INCREMENT
	generator.Fourth = Xoshiro_Fourth(split_mix(state))
	return generator
}

// The splitmix64 avalanche of one strided state.
func split_mix(state Seed) (word Word) {
	defer func() { Word_Invariants(word, "split_mix.word") }()
	Seed_Invariants(state, "split_mix.state")
	value := Word(state)
	value = (value ^ (value >> 30)) * SPLIT_MIX_MULTIPLIER_FIRST
	value = (value ^ (value >> 27)) * SPLIT_MIX_MULTIPLIER_SECOND
	return value ^ (value >> 31)
}

// Xoshiro_Next advances the xoshiro256++ state and returns the next value. It is the raw draw
// every other function builds on, and the one hot path that must not allocate.
func Xoshiro_Next(generator Xoshiro_Pointer) (value Word) {
	defer func() { Word_Invariants(value, "xoshiro_next.value") }()
	Xoshiro_Pointer_Invariants(generator, "xoshiro_next.generator")
	aver.Always(generator != nil, "A draw advances caller-owned generator storage.")
	first, second := Word(generator.First), Word(generator.Second)
	third, fourth := Word(generator.Third), Word(generator.Fourth)
	result := Word(bits.Rotate_Left_64(bits.Word_64(first+fourth), 23)) + first
	shifted := second << 17
	third ^= first
	fourth ^= second
	second ^= third
	first ^= fourth
	third ^= shifted
	fourth = Word(bits.Rotate_Left_64(bits.Word_64(fourth), 45))
	generator.First, generator.Second = Xoshiro_First(first), Xoshiro_Second(second)
	generator.Third, generator.Fourth = Xoshiro_Third(third), Xoshiro_Fourth(fourth)
	return result
}

// Xoshiro_Below returns a value in the half-open range zero to bound, never bound itself.
func Xoshiro_Below(generator Xoshiro_Pointer, bound Bound) (value Index) {
	defer func() { Index_Invariants(value, "xoshiro_below.value") }()
	Xoshiro_Pointer_Invariants(generator, "xoshiro_below.generator")
	Bound_Invariants(bound, "xoshiro_below.bound")
	aver.Always(generator != nil, "A bounded draw advances caller-owned generator storage.")
	return Index(xoshiro_below_unsigned(generator, Draw_Bound(bound)))
}

// Xoshiro_Boolean returns true or false with equal probability, from the top state bit.
func Xoshiro_Boolean(generator Xoshiro_Pointer) (value Boolean) {
	defer func() { Boolean_Invariants(value, "xoshiro_boolean.value") }()
	Xoshiro_Pointer_Invariants(generator, "xoshiro_boolean.generator")
	aver.Always(generator != nil, "A coin flip advances caller-owned generator storage.")
	return Xoshiro_Next(generator)>>63 != 0
}

// Xoshiro_Chance returns true at a frequency tracking the integer Ratio, with no floating point.
func Xoshiro_Chance(generator Xoshiro_Pointer, probability Ratio) (value Boolean) {
	defer func() { Boolean_Invariants(value, "xoshiro_chance.value") }()
	Xoshiro_Pointer_Invariants(generator, "xoshiro_chance.generator")
	Ratio_Invariants(probability, "xoshiro_chance.probability")
	aver.Always(generator != nil, "A weighted flip advances caller-owned generator storage.")
	return Draw_Bound(xoshiro_below_unsigned(
		generator, Draw_Bound(probability.Denominator),
	)) < Draw_Bound(probability.Numerator)
}

// New_Distribution builds a Distribution from outcomes and their integer weights, turning
// weights into the cumulative table in place: the caller's storage is the table Sample draws
// against. The slices must be equal length and hold a positive total.
func New_Distribution(outcomes Outcomes, weights Weights) (distribution Distribution) {
	defer func() { Distribution_Invariants(distribution, "new_distribution.distribution") }()
	Outcomes_Invariants(outcomes, "new_distribution.outcomes")
	Weights_Invariants(weights, "new_distribution.weights")
	aver.Always(len(outcomes) == len(weights), "Each outcome carries one weight.")
	running_total := Weight(0)
	for index := 0; index < len(weights); index++ {
		Weight_Invariants(weights[index], "new_distribution.weight")
		aver.Always(
			weights[index] <= WEIGHT_MAXIMUM-running_total,
			"Distribution cumulative weight does not overflow.",
		)
		running_total += weights[index]
		weights[index] = running_total
	}
	aver.Always(running_total > 0, "prng distribution total is positive")
	return Distribution{Outcomes: outcomes, Cumulative: weights}
}

// Xoshiro_Sample returns an outcome at a frequency tracking its integer weight in distribution.
func Xoshiro_Sample(generator Xoshiro_Pointer, distribution Distribution) (item Word) {
	defer func() { Word_Invariants(item, "xoshiro_sample.item") }()
	Xoshiro_Pointer_Invariants(generator, "xoshiro_sample.generator")
	Distribution_Invariants(distribution, "xoshiro_sample.distribution")
	aver.Always(generator != nil, "A sample advances caller-owned generator storage.")
	cumulative := distribution.Cumulative
	count := len(cumulative)
	total := cumulative[count-1]
	roll := xoshiro_below_unsigned(generator, Draw_Bound(total))
	for index := 0; index < count; index++ {
		if Weight(roll) < cumulative[index] {
			return distribution.Outcomes[index]
		}
	}
	return distribution.Outcomes[count-1]
}

// Bimodal_Fast separates the common mode from the rare mode's invariant subject.
type Bimodal_Fast Word

// Bimodal_Fast_Invariants preserves the caller's complete unit domain.
func Bimodal_Fast_Invariants(value Bimodal_Fast, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), uint64(WORD_MINIMUM), uint64(WORD_MAXIMUM)).
		Ensure()
}

// Bimodal_Slow separates the rare mode from the common mode's invariant subject.
type Bimodal_Slow Word

// Bimodal_Slow_Invariants preserves the caller's complete unit domain.
func Bimodal_Slow_Invariants(value Bimodal_Slow, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), uint64(WORD_MINIMUM), uint64(WORD_MAXIMUM)).
		Ensure()
}

// Bimodal_Distribution_Input configures a two-mode distribution.
type Bimodal_Distribution_Input struct {
	// Fast is the value of the common mode.
	Fast Bimodal_Fast
	// Slow is the value of the rare mode.
	Slow Bimodal_Slow
	// Slow_Chance is the probability that a draw takes the Slow mode.
	Slow_Chance Ratio
}

// Bimodal_Distribution_Input_Invariants composes both modes and their probability.
func Bimodal_Distribution_Input_Invariants(
	input Bimodal_Distribution_Input, namespace aver.Namespace,
) {
	Bimodal_Fast_Invariants(input.Fast, namespace)
	Bimodal_Slow_Invariants(input.Slow, namespace)
	Ratio_Invariants(input.Slow_Chance, namespace)
}

// BIMODAL_OUTCOME_COUNT is the storage a bimodal table fills: one entry per mode.
const BIMODAL_OUTCOME_COUNT = 2

// Bimodal_Outcomes is caller storage for a bimodal table. Its own type, narrower than
// Outcomes, because a two-mode table never fits one slot: the domain starts at two.
type Bimodal_Outcomes []Word

// Bimodal_Outcomes_Invariants admits storage from two modes up to the shared width.
func Bimodal_Outcomes_Invariants(outcomes Bimodal_Outcomes, namespace aver.Namespace) {
	aver.Tree(outcomes, namespace).
		Range_Int(len(outcomes), BIMODAL_OUTCOME_COUNT, DISTRIBUTION_COUNT_MAXIMUM).
		Ensure()
}

// Bimodal_Weights is caller storage for a bimodal table's mass, paired with Bimodal_Outcomes.
type Bimodal_Weights []Weight

// Bimodal_Weights_Invariants admits storage from two modes up to the shared width.
func Bimodal_Weights_Invariants(weights Bimodal_Weights, namespace aver.Namespace) {
	aver.Tree(weights, namespace).
		Range_Int(len(weights), BIMODAL_OUTCOME_COUNT, DISTRIBUTION_COUNT_MAXIMUM).
		Ensure()
}

// Bimodal_Fill writes a two-mode table into caller storage: the Fast value with the complement
// of Slow_Chance, the Slow value with Slow_Chance, and nothing in between. Values carry the
// caller's own unit. The first two entries of each slice are written; the caller hands them to
// New_Distribution. It returns nothing because a returned Distribution would state the full
// table domain for a table that is always two entries.
func Bimodal_Fill(
	input Bimodal_Distribution_Input, outcomes Bimodal_Outcomes, weights Bimodal_Weights,
) {
	Bimodal_Distribution_Input_Invariants(input, "bimodal_fill.input")
	Bimodal_Outcomes_Invariants(outcomes, "bimodal_fill.outcomes")
	Bimodal_Weights_Invariants(weights, "bimodal_fill.weights")
	outcomes[0], outcomes[1] = Word(input.Fast), Word(input.Slow)
	weights[0] = Weight(input.Slow_Chance.Denominator) - Weight(input.Slow_Chance.Numerator)
	weights[1] = Weight(input.Slow_Chance.Numerator)
}

// Percentile_25 keeps its boundary independent from the other percentile fields.
type Percentile_25 Word

// Percentile_25_Invariants preserves the caller's complete unit domain.
func Percentile_25_Invariants(value Percentile_25, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), uint64(WORD_MINIMUM), uint64(WORD_MAXIMUM)).
		Ensure()
}

// Percentile_50 keeps its boundary independent from the other percentile fields.
type Percentile_50 Word

// Percentile_50_Invariants preserves the caller's complete unit domain.
func Percentile_50_Invariants(value Percentile_50, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), uint64(WORD_MINIMUM), uint64(WORD_MAXIMUM)).
		Ensure()
}

// Percentile_75 keeps its boundary independent from the other percentile fields.
type Percentile_75 Word

// Percentile_75_Invariants preserves the caller's complete unit domain.
func Percentile_75_Invariants(value Percentile_75, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), uint64(WORD_MINIMUM), uint64(WORD_MAXIMUM)).
		Ensure()
}

// Percentile_95 keeps its boundary independent from the other percentile fields.
type Percentile_95 Word

// Percentile_95_Invariants preserves the caller's complete unit domain.
func Percentile_95_Invariants(value Percentile_95, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), uint64(WORD_MINIMUM), uint64(WORD_MAXIMUM)).
		Ensure()
}

// Percentile_99 keeps its boundary independent from the other percentile fields.
type Percentile_99 Word

// Percentile_99_Invariants preserves the caller's complete unit domain.
func Percentile_99_Invariants(value Percentile_99, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), uint64(WORD_MINIMUM), uint64(WORD_MAXIMUM)).
		Ensure()
}

// Percentile_100 keeps its boundary independent from the other percentile fields.
type Percentile_100 Word

// Percentile_100_Invariants preserves the caller's complete unit domain.
func Percentile_100_Invariants(value Percentile_100, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), uint64(WORD_MINIMUM), uint64(WORD_MAXIMUM)).
		Ensure()
}

// Percentile_Distribution_Input gives the value at each of six percentiles of a distribution.
type Percentile_Distribution_Input struct {
	// P25 is the value at the twenty-fifth percentile.
	P25 Percentile_25
	// P50 is the value at the median.
	P50 Percentile_50
	// P75 is the value at the seventy-fifth percentile.
	P75 Percentile_75
	// P95 is the value at the ninety-fifth percentile.
	P95 Percentile_95
	// P99 is the value at the ninety-ninth percentile.
	P99 Percentile_99
	// P100 is the ceiling value the top one percent returns.
	P100 Percentile_100
}

// Percentile_Distribution_Input_Invariants keeps all six caller-unit values representable.
func Percentile_Distribution_Input_Invariants(
	input Percentile_Distribution_Input, namespace aver.Namespace,
) {
	Percentile_25_Invariants(input.P25, namespace)
	Percentile_50_Invariants(input.P50, namespace)
	Percentile_75_Invariants(input.P75, namespace)
	Percentile_95_Invariants(input.P95, namespace)
	Percentile_99_Invariants(input.P99, namespace)
	Percentile_100_Invariants(input.P100, namespace)
}

// PERCENTILE_OUTCOME_COUNT is the storage a percentile table fills: one entry per percentile.
const PERCENTILE_OUTCOME_COUNT = 6

// Percentile_Outcomes is caller storage for a percentile table. Its own type, narrower than
// Outcomes, because six percentiles never fit fewer slots: the domain starts at six.
type Percentile_Outcomes []Word

// Percentile_Outcomes_Invariants admits storage from six percentiles up to the shared width.
func Percentile_Outcomes_Invariants(outcomes Percentile_Outcomes, namespace aver.Namespace) {
	aver.Tree(outcomes, namespace).
		Range_Int(len(outcomes), PERCENTILE_OUTCOME_COUNT, DISTRIBUTION_COUNT_MAXIMUM).
		Ensure()
}

// Percentile_Weights is caller storage for a percentile table's mass, paired with
// Percentile_Outcomes.
type Percentile_Weights []Weight

// Percentile_Weights_Invariants admits storage from six percentiles up to the shared width.
func Percentile_Weights_Invariants(weights Percentile_Weights, namespace aver.Namespace) {
	aver.Tree(weights, namespace).
		Range_Int(len(weights), PERCENTILE_OUTCOME_COUNT, DISTRIBUTION_COUNT_MAXIMUM).
		Ensure()
}

// Percentile_Fill writes a table into caller storage from the six percentile values, weighted
// by the mass between them, so a draw reproduces those percentiles; the top one percent
// returns P100. The first six entries of each slice are written; the caller hands them to
// New_Distribution. It returns nothing for the reason Bimodal_Fill gives.
func Percentile_Fill(
	input Percentile_Distribution_Input, outcomes Percentile_Outcomes,
	weights Percentile_Weights,
) {
	Percentile_Distribution_Input_Invariants(input, "percentile_fill.input")
	Percentile_Outcomes_Invariants(outcomes, "percentile_fill.outcomes")
	Percentile_Weights_Invariants(weights, "percentile_fill.weights")
	outcomes[0], outcomes[1] = Word(input.P25), Word(input.P50)
	outcomes[2], outcomes[3] = Word(input.P75), Word(input.P95)
	outcomes[4], outcomes[5] = Word(input.P99), Word(input.P100)
	weights[0], weights[1], weights[2] = 25, 25, 25
	weights[3], weights[4], weights[5] = 20, 4, 1
}

// Xoshiro_Shuffle reorders the permutation in place by Fisher-Yates, so each ordering is
// equally likely. The caller applies the permutation to whatever it indexes.
func Xoshiro_Shuffle(generator Xoshiro_Pointer, permutation Permutation) {
	Xoshiro_Pointer_Invariants(generator, "xoshiro_shuffle.generator")
	Permutation_Invariants(permutation, "xoshiro_shuffle.permutation")
	aver.Always(generator != nil, "A shuffle advances caller-owned generator storage.")
	for index := len(permutation) - 1; index > 0; index-- {
		swap_index := Xoshiro_Below(generator, Bound(index+1))
		swapped := permutation[swap_index]
		permutation[swap_index] = permutation[index]
		permutation[index] = swapped
	}
}

// Xoshiro_Split returns a child Xoshiro seeded from one draw of the parent, an independent
// stream so a draw in one cannot perturb the other.
func Xoshiro_Split(generator Xoshiro_Pointer) (child Xoshiro) {
	defer func() { Xoshiro_Invariants(child, "xoshiro_split.child") }()
	Xoshiro_Pointer_Invariants(generator, "xoshiro_split.generator")
	aver.Always(generator != nil, "A split advances caller-owned generator storage.")
	return New(Seed(Xoshiro_Next(generator)))
}

// Xoshiro_To_Source binds caller-owned state into the crypto/prng Source vtable without a
// captured function environment: the simulation stand-in for a ChaCha source, same slot, xoshiro
// words, so a signer or key builder under test replays every entropy draw from the run seed. This
// call is the one place a fake enters a cryptographic parameter, so a grep for it finds every
// test that signs with predictable bytes. Fork the generator first; a draw through this source
// spends the same stream as every other draw on it. Never bind it in a production root.
func Xoshiro_To_Source(generator Xoshiro_Pointer) (source prng.Source) {
	defer func() { prng.Source_Invariants(source, "xoshiro_to_source.source") }()
	Xoshiro_Pointer_Invariants(generator, "xoshiro_to_source.generator")
	aver.Always(generator != nil, "A Source requires Xoshiro storage.")
	return prng.Source{
		State: (*Xoshiro)(generator),
		Next:  xoshiro_source_next,
	}
}

// The vtable slot: one xoshiro draw.
func xoshiro_source_next(state prng.Backend_State) (value prng.Word) {
	defer func() { prng.Word_Invariants(value, "xoshiro_source_next.value") }()
	prng.Backend_State_Invariants(state, "xoshiro_source_next.state")
	generator, accepted := state.(*Xoshiro)
	aver.Always(accepted, "Xoshiro source state has Xoshiro storage.")
	aver.Always(generator != nil, "Xoshiro source storage exists.")
	return prng.Word(Xoshiro_Next(generator))
}

// Returns a value in the half-open range zero to bound using Lemire's method, so the result is
// unbiased, not skewed the way a plain modulo would be. The caller guarantees bound is positive.
func xoshiro_below_unsigned(generator Xoshiro_Pointer, bound Draw_Bound) (value Draw) {
	defer func() { Draw_Invariants(value, "xoshiro_below_unsigned.value") }()
	Xoshiro_Pointer_Invariants(generator, "xoshiro_below_unsigned.generator")
	Draw_Bound_Invariants(bound, "xoshiro_below_unsigned.bound")
	aver.Always(generator != nil, "An unsigned draw advances caller-owned generator storage.")
	word := Xoshiro_Next(generator)
	high_word, low_word := bits.Multiply_64(
		bits.Word_64(word), bits.Multiplier_64(bound),
	)
	high, low := Draw(high_word), Draw(low_word)
	if Draw_Bound(low) < bound {
		threshold := (-bound) % bound
		for Draw_Bound(low) < threshold {
			word = Xoshiro_Next(generator)
			high_word, low_word = bits.Multiply_64(
				bits.Word_64(word), bits.Multiplier_64(bound),
			)
			high, low = Draw(high_word), Draw(low_word)
		}
	}
	return high
}
