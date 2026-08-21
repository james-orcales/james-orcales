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
	"unsafe"

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

// XOSHIRO_STATE_WORD_COUNT is the fixed state width of xoshiro256++.
const XOSHIRO_STATE_WORD_COUNT = 4

// DISTRIBUTION_COUNT_MAXIMUM keeps weighted sampling linear and stack-owned.
const DISTRIBUTION_COUNT_MAXIMUM = 32

// ITEM_COUNT_MAXIMUM shares the one fixed collection width used by this package.
const ITEM_COUNT_MAXIMUM Item_Count = DISTRIBUTION_COUNT_MAXIMUM

// ITEM_COUNT_MINIMUM admits an empty shuffle while Element rejects it separately.
const ITEM_COUNT_MINIMUM Item_Count = 0

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
const DISTRIBUTION_COUNT_MINIMUM Distribution_Count = 1

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

// Item_Count keeps generic collection bounds on a concrete invariant subject.
type Item_Count int

// Item_Count_Invariants rejects hostile selection and shuffle work.
func Item_Count_Invariants(count Item_Count, namespace aver.Namespace) {
	aver.Tree(count, namespace).
		Range_Int(int(count), int(ITEM_COUNT_MINIMUM), int(ITEM_COUNT_MAXIMUM)).
		Ensure()
}

// Items keeps selection and shuffle storage fixed and caller-owned.
type Items[T any] [ITEM_COUNT_MAXIMUM]T

// Items_Invariants makes the generic aggregate a fixed-width singleton.
func Items_Invariants[T any](items Items[T], _ aver.Namespace) {
	aver.Always(
		len(items) == int(ITEM_COUNT_MAXIMUM),
		"Random item storage keeps its fixed width.",
	)
}

// Weights stays at the fixed Distribution storage width.
type Weights [DISTRIBUTION_COUNT_MAXIMUM]Weight

// Weights_Invariants makes cumulative storage a fixed-width singleton.
func Weights_Invariants(weights Weights, _ aver.Namespace) {
	aver.Always(
		len(weights) == DISTRIBUTION_COUNT_MAXIMUM,
		"Distribution weights keep their fixed width.",
	)
}

// Distribution_Count separates live buckets from zeroed fixed storage.
type Distribution_Count int

// Distribution_Count_Invariants keeps every live bucket addressable.
func Distribution_Count_Invariants(count Distribution_Count, namespace aver.Namespace) {
	aver.Tree(count, namespace).
		Range_Int(
			int(count), int(DISTRIBUTION_COUNT_MINIMUM), DISTRIBUTION_COUNT_MAXIMUM,
		).
		Ensure()
}

// Xoshiro is the state of a xoshiro256++ pseudo-random generator. Construct it with New; the zero
// value is degenerate, since an all-zero xoshiro state emits only zeros.
type Xoshiro struct {
	// State is the four 64-bit words of xoshiro256++ internal state.
	State [XOSHIRO_STATE_WORD_COUNT]Word
}

// Xoshiro_Invariants rejects xoshiro's absorbing all-zero state.
func Xoshiro_Invariants(generator Xoshiro, _ aver.Namespace) {
	aver.Always(
		generator.State != [XOSHIRO_STATE_WORD_COUNT]Word{},
		"A generator has nonzero xoshiro state.",
	)
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
type Distribution[T any] struct {
	// Outcomes are the values Sample may return, positionally paired with Cumulative.
	Outcomes Items[T]
	// Cumulative is the running sum of each outcome's weight; the final entry is the total.
	Cumulative Weights
}

// Distribution_Invariants keeps the live fixed table nonempty and drawable.
func Distribution_Invariants[T any](
	distribution Distribution[T], namespace aver.Namespace,
) {
	Items_Invariants(distribution.Outcomes, namespace)
	Weights_Invariants(distribution.Cumulative, namespace)
	aver.Always(
		distribution.Cumulative[DISTRIBUTION_COUNT_MAXIMUM-1] > 0,
		"A distribution has positive total weight.",
	)
}

// New seeds a Xoshiro from one seed, expanding it through splitmix64 into the four words of
// xoshiro256++ state. The zero Xoshiro is degenerate, so always construct through New.
func New(seed Seed) (generator Xoshiro) {
	defer func() { Xoshiro_Invariants(generator, "new.generator") }()
	Seed_Invariants(seed, "new.seed")
	state := seed
	for index := 0; index < XOSHIRO_STATE_WORD_COUNT; index++ {
		state += SPLIT_MIX_INCREMENT
		value := Word(state)
		value = (value ^ (value >> 30)) * SPLIT_MIX_MULTIPLIER_FIRST
		value = (value ^ (value >> 27)) * SPLIT_MIX_MULTIPLIER_SECOND
		generator.State[index] = value ^ (value >> 31)
	}
	return generator
}

// Xoshiro_Next advances the xoshiro256++ state and returns the next value. It is the raw draw
// every other function builds on, and the one hot path that must not allocate.
func Xoshiro_Next(generator *Xoshiro) (value Word) {
	defer func() { Word_Invariants(value, "xoshiro_next.value") }()
	Xoshiro_Invariants(*generator, "xoshiro_next.generator")
	result := Word(bits.Rotate_Left_64(
		bits.Word_64(generator.State[0]+generator.State[3]), 23,
	)) + generator.State[0]
	shifted := generator.State[1] << 17
	generator.State[2] ^= generator.State[0]
	generator.State[3] ^= generator.State[1]
	generator.State[1] ^= generator.State[2]
	generator.State[0] ^= generator.State[3]
	generator.State[2] ^= shifted
	generator.State[3] = Word(bits.Rotate_Left_64(bits.Word_64(generator.State[3]), 45))
	return result
}

// Xoshiro_Below returns a value in the half-open range zero to bound, never bound itself.
func Xoshiro_Below(generator *Xoshiro, bound Bound) (value Index) {
	defer func() { Index_Invariants(value, "xoshiro_below.value") }()
	Xoshiro_Invariants(*generator, "xoshiro_below.generator")
	Bound_Invariants(bound, "xoshiro_below.bound")
	return Index(xoshiro_below_unsigned(generator, Draw_Bound(bound)))
}

// Xoshiro_Element returns one uniformly chosen element of items; an empty slice panics.
func Xoshiro_Element[T any](
	generator *Xoshiro, items *Items[T], count Item_Count,
) (item T) {
	Xoshiro_Invariants(*generator, "xoshiro_element.generator")
	Items_Invariants(*items, "xoshiro_element.items")
	Item_Count_Invariants(count, "xoshiro_element.count")
	aver.Always(count > 0, "prng element count is not empty")
	return items[Xoshiro_Below(generator, Bound(count))]
}

// Xoshiro_Boolean returns true or false with equal probability, from the top state bit.
func Xoshiro_Boolean(generator *Xoshiro) (value Boolean) {
	defer func() { Boolean_Invariants(value, "xoshiro_boolean.value") }()
	Xoshiro_Invariants(*generator, "xoshiro_boolean.generator")
	return Xoshiro_Next(generator)>>63 != 0
}

// Xoshiro_Chance returns true at a frequency tracking the integer Ratio, with no floating point.
func Xoshiro_Chance(generator *Xoshiro, probability Ratio) (value Boolean) {
	defer func() { Boolean_Invariants(value, "xoshiro_chance.value") }()
	Xoshiro_Invariants(*generator, "xoshiro_chance.generator")
	Ratio_Invariants(probability, "xoshiro_chance.probability")
	return Draw_Bound(xoshiro_below_unsigned(
		generator, Draw_Bound(probability.Denominator),
	)) < Draw_Bound(probability.Numerator)
}

// New_Distribution builds a Distribution from outcomes and their integer weights, precomputing the
// cumulative table Sample draws against. The slices must be equal length and hold a positive total.
func New_Distribution[T any](
	outcomes Items[T], weights Weights, count Distribution_Count,
) (distribution Distribution[T]) {
	defer func() { Distribution_Invariants(distribution, "new_distribution.distribution") }()
	Items_Invariants(outcomes, "new_distribution.outcomes")
	Weights_Invariants(weights, "new_distribution.weights")
	Distribution_Count_Invariants(count, "new_distribution.count")
	running_total := Weight(0)
	for index := 0; index < int(count); index++ {
		Weight_Invariants(weights[index], "new_distribution.weight")
		aver.Always(
			weights[index] <= WEIGHT_MAXIMUM-running_total,
			"Distribution cumulative weight does not overflow.",
		)
		running_total += weights[index]
		distribution.Outcomes[index] = outcomes[index]
		distribution.Cumulative[index] = running_total
	}
	aver.Always(running_total > 0, "prng distribution total is positive")
	for index := int(count); index < DISTRIBUTION_COUNT_MAXIMUM; index++ {
		distribution.Cumulative[index] = running_total
	}
	return distribution
}

// Xoshiro_Sample returns an outcome at a frequency tracking its integer weight in distribution.
func Xoshiro_Sample[T any](generator *Xoshiro, distribution Distribution[T]) (item T) {
	Xoshiro_Invariants(*generator, "xoshiro_sample.generator")
	Distribution_Invariants(distribution, "xoshiro_sample.distribution")
	cumulative := distribution.Cumulative
	count := DISTRIBUTION_COUNT_MAXIMUM
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

// Bimodal_Distribution returns a two-mode table: the Fast value with the complement of
// Slow_Chance, the Slow value with Slow_Chance, and nothing in between. Values carry the
// caller's own unit.
func Bimodal_Distribution(
	input *Bimodal_Distribution_Input,
) (distribution Distribution[Word]) {
	defer func() {
		Distribution_Invariants(distribution, "bimodal_distribution.distribution")
	}()
	Bimodal_Distribution_Input_Invariants(*input, "bimodal_distribution.input")
	return New_Distribution(
		Items[Word]{Word(input.Fast), Word(input.Slow)},
		Weights{
			Weight(input.Slow_Chance.Denominator) - Weight(input.Slow_Chance.Numerator),
			Weight(input.Slow_Chance.Numerator),
		},
		2,
	)
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

// Percentile_Distribution builds a table from the six percentile values, weighted by the mass
// between them, so a draw reproduces those percentiles; the top one percent returns P100.
func Percentile_Distribution(
	input *Percentile_Distribution_Input,
) (distribution Distribution[Word]) {
	defer func() {
		Distribution_Invariants(distribution, "percentile_distribution.distribution")
	}()
	Percentile_Distribution_Input_Invariants(*input, "percentile_distribution.input")
	return New_Distribution(
		Items[Word]{
			Word(input.P25),
			Word(input.P50),
			Word(input.P75),
			Word(input.P95),
			Word(input.P99),
			Word(input.P100),
		},
		Weights{25, 25, 25, 20, 4, 1},
		6,
	)
}

// Xoshiro_Shuffle reorders items in place by Fisher-Yates, so each ordering is equally likely.
func Xoshiro_Shuffle[T any](
	generator *Xoshiro, items *Items[T], count Item_Count,
) {
	Xoshiro_Invariants(*generator, "xoshiro_shuffle.generator")
	Items_Invariants(*items, "xoshiro_shuffle.items")
	Item_Count_Invariants(count, "xoshiro_shuffle.count")
	for index := int(count) - 1; index > 0; index-- {
		swap_index := Xoshiro_Below(generator, Bound(index+1))
		items[index], items[swap_index] = items[swap_index], items[index]
	}
}

// Xoshiro_Split returns a child Xoshiro seeded from one draw of the parent, an independent
// stream so a draw in one cannot perturb the other.
func Xoshiro_Split(generator *Xoshiro) (child Xoshiro) {
	defer func() { Xoshiro_Invariants(child, "xoshiro_split.child") }()
	Xoshiro_Invariants(*generator, "xoshiro_split.generator")
	return New(Seed(Xoshiro_Next(generator)))
}

// Xoshiro_To_Source binds caller-owned state into the crypto/prng Source vtable without a
// captured function environment: the simulation stand-in for a ChaCha source, same slot, xoshiro
// words, so a signer or key builder under test replays every entropy draw from the run seed. This
// call is the one place a fake enters a cryptographic parameter, so a grep for it finds every
// test that signs with predictable bytes. Fork the generator first; a draw through this source
// spends the same stream as every other draw on it. Never bind it in a production root.
func Xoshiro_To_Source(generator *Xoshiro) (source prng.Source) {
	defer func() { prng.Source_Invariants(source, "xoshiro_to_source.source") }()
	Xoshiro_Invariants(*generator, "xoshiro_to_source.generator")
	source = prng.Source{
		State: unsafe.Pointer(generator),
		Next:  xoshiro_source_next,
	}
	return source
}

// The vtable slot: one xoshiro draw.
func xoshiro_source_next(state unsafe.Pointer) (value prng.Word) {
	defer func() { prng.Word_Invariants(value, "xoshiro_source_next.value") }()
	return prng.Word(Xoshiro_Next((*Xoshiro)(state)))
}

// Returns a value in the half-open range zero to bound using Lemire's method, so the result is
// unbiased, not skewed the way a plain modulo would be. The caller guarantees bound is positive.
func xoshiro_below_unsigned(generator *Xoshiro, bound Draw_Bound) (value Draw) {
	defer func() { Draw_Invariants(value, "xoshiro_below_unsigned.value") }()
	Xoshiro_Invariants(*generator, "xoshiro_below_unsigned.generator")
	Draw_Bound_Invariants(bound, "xoshiro_below_unsigned.bound")
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
