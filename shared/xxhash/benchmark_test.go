package xxhash_test

import (
	"testing"

	"local/james-orcales/shared/xxhash"
)

// Benchmark_Hash measures the one-shot hash over a mid-sized buffer — the throughput headline.
func Benchmark_Hash(b *testing.B) {
	data := make([]byte, 1024)
	b.SetBytes(int64(len(data)))
	for b.Loop() {
		xxhash.Hash(data, 0)
	}
}

// Benchmark_Digest measures streaming the same buffer through Write then Sum.
func Benchmark_Digest(b *testing.B) {
	data := make([]byte, 1024)
	b.SetBytes(int64(len(data)))
	for b.Loop() {
		digest := xxhash.New_Digest(0)
		digest.Write(data)
		xxhash.Digest_Sum64(&digest)
	}
}
