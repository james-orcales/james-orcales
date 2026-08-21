package prng_test

import (
	"testing"

	"local/james-orcales/shared/crypto/prng"
)

// SEED_BYTE_COUNT keeps the benchmark input equal to one ChaCha20 key.
const SEED_BYTE_COUNT = 32

// Benchmark_Read measures filling a 32-byte caller sink.
func Benchmark_Read(b *testing.B) {
	seed := make(prng.Seed, SEED_BYTE_COUNT)
	seed[0] = 1
	var generator prng.Chacha
	prng.Chacha_Init(&generator, seed, prng.CURSOR_MIN)
	buffer := make([]byte, 32)
	for b.Loop() {
		prng.Chacha_Read(&generator, buffer)
	}
}

// Benchmark_Below measures a bounded draw, with its precondition and Lemire rejection.
func Benchmark_Below(b *testing.B) {
	seed := make(prng.Seed, SEED_BYTE_COUNT)
	seed[0] = 1
	var generator prng.Chacha
	prng.Chacha_Init(&generator, seed, prng.CURSOR_MIN)
	for b.Loop() {
		prng.Chacha_Below(&generator, prng.Bound(100))
	}
}
