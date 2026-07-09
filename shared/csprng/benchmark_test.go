package csprng_test

import (
	"testing"

	"local/james-orcales/shared/csprng"
)

// Benchmark_Uint64 measures the raw 64-bit draw, the hot path that must not allocate.
func Benchmark_Uint64(b *testing.B) {
	generator := csprng.New([32]byte{1})
	for b.Loop() {
		csprng.Generator_Uint64(&generator)
	}
}

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
		csprng.Generator_Below(&generator, 100)
	}
}
