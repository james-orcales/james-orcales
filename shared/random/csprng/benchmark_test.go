package csprng_test

import (
	"testing"

	"local/james-orcales/shared/random/csprng"
)

// Benchmark_Read measures filling a 32-byte buffer through io.Reader.
func Benchmark_Read(b *testing.B) {
	generator := csprng.New([32]byte{1})
	buffer := make([]byte, 32)
	for b.Loop() {
		generator.Read(buffer)
	}
}

// Benchmark_Below measures a bounded draw, with its precondition and Lemire rejection.
func Benchmark_Below(b *testing.B) {
	generator := csprng.New([32]byte{1})
	for b.Loop() {
		csprng.Generator_Below(&generator, csprng.Bound(100))
	}
}
