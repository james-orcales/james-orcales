package csprng_test

import (
	"testing"

	"local/james-orcales/shared/random/csprng"
)

// SEED_BYTE_COUNT keeps the benchmark input equal to one ChaCha20 key.
const SEED_BYTE_COUNT = 32

// Benchmark_Read measures filling a 32-byte buffer through io.Reader.
func Benchmark_Read(b *testing.B) {
	generator := csprng.New([SEED_BYTE_COUNT]byte{1}, csprng.CURSOR_MIN)
	buffer := make([]byte, 32)
	for b.Loop() {
		generator.Read(buffer)
	}
}

// Benchmark_Below measures a bounded draw, with its precondition and Lemire rejection.
func Benchmark_Below(b *testing.B) {
	generator := csprng.New([SEED_BYTE_COUNT]byte{1}, csprng.CURSOR_MIN)
	for b.Loop() {
		csprng.Generator_Below(&generator, csprng.Bound(100))
	}
}
