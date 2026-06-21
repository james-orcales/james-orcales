// Package prng is a deterministic, integer-only pseudo-random number generator.
//
// It mirrors TigerBeetle's stdx.PRNG: a xoshiro256++ generator seeded by splitmix64, with no
// floating point anywhere in its surface. A probability is an integer Ratio, never a float, so a
// run reproduces bit-for-bit across machines from one seed — the property deterministic simulation
// testing is built on, where a failure replays exactly from the seed that found it.
//
// The house linter bans methods, so each draw is a free function named after its first parameter's
// type. Seed with New; the zero Generator is unusable.
//
//	generator := prng.New(seed)
//	victim := prng.Generator_Below(&generator, replica_count)
//	if prng.Generator_Chance(&generator, prng.Ratio{Numerator: 8, Denominator: 100}) {
//	    drop_packet()
//	}
//
// xoshiro256++ and splitmix64 are public domain (Blackman and Vigna).
package prng

import (
	"math/bits"

	invariant "github.com/james-orcales/james-orcales/shared/invariant/default"
)

// The splitmix64 increment, derived from the golden ratio, strides the seed state.
const split_mix_increment = 0x9e3779b97f4a7c15

// The first splitmix64 multiplier that avalanches the strided state.
const split_mix_multiplier_first = 0xbf58476d1ce4e5b9

// The second splitmix64 multiplier that avalanches the strided state.
const split_mix_multiplier_second = 0x94d049bb133111eb

// Generator is the state of a xoshiro256++ pseudo-random generator. Construct it with New; the zero
// value is degenerate, since an all-zero xoshiro state emits only zeros.
type Generator struct {
	// State is the four 64-bit words of xoshiro256++ internal state.
	State [4]uint64
}

// Ratio is an integer probability, used instead of a float so a run reproduces bit-for-bit.
type Ratio struct {
	// Numerator is the count of favorable outcomes; it must not exceed Denominator.
	Numerator uint64
	// Denominator is the total count of outcomes; it must be positive.
	Denominator uint64
}

// Distribution is a set of weighted outcomes Sample draws from, with the cumulative weights
// precomputed. Sample finds the bucket by a linear scan, so keep the entry count to 32 or fewer.
type Distribution[T any] struct {
	// Outcomes are the values Sample may return, positionally paired with Cumulative.
	Outcomes []T
	// Cumulative is the running sum of each outcome's weight; the final entry is the total.
	Cumulative []uint64
}

// New seeds a Generator from one seed, expanding it through splitmix64 into the four words of
// xoshiro256++ state. The zero Generator is degenerate, so always construct through New.
func New(seed uint64) (generator Generator) {
	state := seed
	generator.State[0] = split_mix_64(&state)
	generator.State[1] = split_mix_64(&state)
	generator.State[2] = split_mix_64(&state)
	generator.State[3] = split_mix_64(&state)
	return generator
}

// Generator_Next advances the xoshiro256++ state and returns the next value. It is the raw draw
// every other function builds on, and the one hot path that must not allocate.
func Generator_Next(generator *Generator) (value uint64) {
	result := bits.RotateLeft64(generator.State[0]+generator.State[3], 23) + generator.State[0]
	shifted := generator.State[1] << 17
	generator.State[2] ^= generator.State[0]
	generator.State[3] ^= generator.State[1]
	generator.State[1] ^= generator.State[2]
	generator.State[0] ^= generator.State[3]
	generator.State[2] ^= shifted
	generator.State[3] = bits.RotateLeft64(generator.State[3], 45)
	return result
}

// Generator_Below returns a value in the half-open range zero to bound, never bound itself.
func Generator_Below(generator *Generator, bound int) (value int) {
	invariant.Always(bound > 0, "prng below bound is positive")
	return int(generator_below_unsigned(generator, uint64(bound)))
}

// Generator_Element returns one uniformly chosen element of items; an empty slice panics.
func Generator_Element[T any](generator *Generator, items []T) (item T) {
	invariant.Always(len(items) > 0, "prng element slice is not empty")
	return items[Generator_Below(generator, len(items))]
}

// Generator_Boolean returns true or false with equal probability, from the top state bit.
func Generator_Boolean(generator *Generator) (value bool) {
	return Generator_Next(generator)>>63 != 0
}

// Generator_Chance returns true at a frequency tracking the integer Ratio, with no floating point.
func Generator_Chance(generator *Generator, probability Ratio) (value bool) {
	invariant.Always(probability.Denominator > 0, "prng chance denominator is positive")
	invariant.Always(
		probability.Numerator <= probability.Denominator,
		"prng chance numerator within denominator",
	)
	return generator_below_unsigned(generator, probability.Denominator) < probability.Numerator
}

// New_Distribution builds a Distribution from outcomes and their integer weights, precomputing the
// cumulative table Sample draws against. The slices must be equal length and hold a positive total.
func New_Distribution[T any](outcomes []T, weights []uint64) (distribution Distribution[T]) {
	invariant.Always(len(outcomes) == len(weights), "prng distribution outcomes match weights")
	invariant.Always(len(weights) > 0, "prng distribution is not empty")
	cumulative := make([]uint64, len(weights))
	running_total := uint64(0)
	for index := 0; index < len(weights); index++ {
		running_total += weights[index]
		cumulative[index] = running_total
	}
	invariant.Always(running_total > 0, "prng distribution total is positive")
	distribution.Outcomes = outcomes
	distribution.Cumulative = cumulative
	return distribution
}

// Generator_Sample returns an outcome at a frequency tracking its integer weight in distribution.
func Generator_Sample[T any](generator *Generator, distribution Distribution[T]) (item T) {
	cumulative := distribution.Cumulative
	count := len(cumulative)
	total := cumulative[count-1]
	roll := generator_below_unsigned(generator, total)
	for index := 0; index < count; index++ {
		if roll < cumulative[index] {
			return distribution.Outcomes[index]
		}
	}
	return distribution.Outcomes[count-1]
}

// Bimodal_Distribution_Input configures a two-mode distribution.
type Bimodal_Distribution_Input struct {
	// Fast is the value of the common mode.
	Fast uint64
	// Slow is the value of the rare mode.
	Slow uint64
	// Slow_Chance is the probability that a draw takes the Slow mode.
	Slow_Chance Ratio
}

// Bimodal_Distribution returns a two-mode table: the Fast value with the complement of
// Slow_Chance, the Slow value with Slow_Chance, and nothing in between. Values carry the
// caller's own unit.
func Bimodal_Distribution(input *Bimodal_Distribution_Input) (distribution Distribution[uint64]) {
	invariant.Always(
		input.Slow_Chance.Denominator > 0,
		"bimodal slow chance denominator is positive",
	)
	invariant.Always(
		input.Slow_Chance.Numerator <= input.Slow_Chance.Denominator,
		"bimodal slow chance does not exceed one",
	)
	return New_Distribution(
		[]uint64{input.Fast, input.Slow},
		[]uint64{
			input.Slow_Chance.Denominator - input.Slow_Chance.Numerator,
			input.Slow_Chance.Numerator,
		},
	)
}

// Percentile_Distribution_Input gives the value at each of six percentiles of a distribution.
type Percentile_Distribution_Input struct {
	// P25 is the value at the twenty-fifth percentile.
	P25 uint64
	// P50 is the value at the median.
	P50 uint64
	// P75 is the value at the seventy-fifth percentile.
	P75 uint64
	// P95 is the value at the ninety-fifth percentile.
	P95 uint64
	// P99 is the value at the ninety-ninth percentile.
	P99 uint64
	// P100 is the ceiling value the top one percent returns.
	P100 uint64
}

// Percentile_Distribution builds a table from the six percentile values, weighted by the mass
// between them, so a draw reproduces those percentiles; the top one percent returns P100.
func Percentile_Distribution(
	input *Percentile_Distribution_Input,
) (distribution Distribution[uint64]) {
	return New_Distribution(
		[]uint64{input.P25, input.P50, input.P75, input.P95, input.P99, input.P100},
		[]uint64{25, 25, 25, 20, 4, 1},
	)
}

// Generator_Shuffle reorders items in place by Fisher-Yates, so each ordering is equally likely.
func Generator_Shuffle[T any](generator *Generator, items []T) {
	for index := len(items) - 1; index > 0; index-- {
		swap_index := Generator_Below(generator, index+1)
		items[index], items[swap_index] = items[swap_index], items[index]
	}
}

// Generator_Split returns a child Generator seeded from one draw of the parent, an independent
// stream so a draw in one cannot perturb the other.
func Generator_Split(generator *Generator) (child Generator) {
	return New(Generator_Next(generator))
}

// Returns a value in the half-open range zero to bound using Lemire's method, so the result is
// unbiased, not skewed the way a plain modulo would be. The caller guarantees bound is positive.
func generator_below_unsigned(generator *Generator, bound uint64) (value uint64) {
	random := Generator_Next(generator)
	high, low := bits.Mul64(random, bound)
	if low < bound {
		threshold := (-bound) % bound
		for low < threshold {
			random = Generator_Next(generator)
			high, low = bits.Mul64(random, bound)
		}
	}
	return high
}

// Advances a splitmix64 state and returns the next value. New uses it to expand one seed into the
// four words of xoshiro256++ state, matching the seeding TigerBeetle's stdx.PRNG uses.
func split_mix_64(state *uint64) (value uint64) {
	*state += split_mix_increment
	value = *state
	value = (value ^ (value >> 30)) * split_mix_multiplier_first
	value = (value ^ (value >> 27)) * split_mix_multiplier_second
	return value ^ (value >> 31)
}
