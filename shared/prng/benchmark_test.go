package prng_test

import (
	"testing"

	"github.com/james-orcales/james-orcales/shared/prng"
)

// Benchmark_Next measures the raw draw, the hot path that must not allocate.
func Benchmark_Next(b *testing.B) {
	generator := prng.New(1)
	for b.Loop() {
		prng.Generator_Next(&generator)
	}
}

// Benchmark_Below measures a bounded draw, including its precondition assertion.
func Benchmark_Below(b *testing.B) {
	generator := prng.New(1)
	for b.Loop() {
		prng.Generator_Below(&generator, 100)
	}
}

// Benchmark_Boolean measures a coin flip.
func Benchmark_Boolean(b *testing.B) {
	generator := prng.New(1)
	for b.Loop() {
		prng.Generator_Boolean(&generator)
	}
}

// Benchmark_Chance measures an integer-ratio probability check.
func Benchmark_Chance(b *testing.B) {
	generator := prng.New(1)
	probability := prng.Ratio{Numerator: 8, Denominator: 100}
	for b.Loop() {
		prng.Generator_Chance(&generator, probability)
	}
}

// Benchmark_Element measures a uniform pick from a slice.
func Benchmark_Element(b *testing.B) {
	generator := prng.New(1)
	items := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	for b.Loop() {
		prng.Generator_Element(&generator, items)
	}
}

// Benchmark_Sample measures a weighted draw over a fixed distribution.
func Benchmark_Sample(b *testing.B) {
	generator := prng.New(1)
	distribution := prng.New_Distribution([]int{0, 1, 2, 3}, []uint64{10, 20, 30, 40})
	for b.Loop() {
		prng.Generator_Sample(&generator, distribution)
	}
}

// Benchmark_Shuffle measures an in-place permutation of a ten-element slice.
func Benchmark_Shuffle(b *testing.B) {
	generator := prng.New(1)
	items := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	for b.Loop() {
		prng.Generator_Shuffle(&generator, items)
	}
}

// Benchmark_Split measures deriving an independent child generator.
func Benchmark_Split(b *testing.B) {
	generator := prng.New(1)
	for b.Loop() {
		prng.Generator_Split(&generator)
	}
}
