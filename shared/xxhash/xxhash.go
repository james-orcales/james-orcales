// Package xxhash is XXH64 (Yann Collet's xxHash): a fast non-cryptographic 64-bit hash for
// fingerprints, deduplication, and in-memory hash tables keyed by trusted data. It is deliberately
// not collision-resistant against an adversary — for maps keyed by untrusted input use a keyed
// hash, and for unpredictable output use csprng.
//
// Hash is the one-shot form over a whole slice. Digest is the streaming form and an io.Writer (the
// one method the house style permits), so io.Copy can feed it and Digest_Sum64 reads the result.
// Both produce identical output for identical bytes and seed.
//
// The algorithm and constants follow the xxHash specification; XXH64 is by Yann Collet.
package xxhash

import (
	"encoding/binary"
	"math/bits"
)

// PRIME64_1 is the primary multiplier, applied after every rotate to spread bits (xxHash spec).
const PRIME64_1 = 0x9E3779B185EBCA87

// PRIME64_2 is the multiplier each input lane is scaled by before folding into an accumulator.
const PRIME64_2 = 0xC2B2AE3D27D4EB4F

// PRIME64_3 is the additive constant of the four-byte tail step.
const PRIME64_3 = 0x165667B19E3779F9

// PRIME64_4 is the additive constant of accumulator merging and the eight-byte tail step.
const PRIME64_4 = 0x85EBCA77C2B2AE63

// PRIME64_5 is the seed offset for inputs shorter than one stripe and the single-byte tail step.
const PRIME64_5 = 0x27D4EB2F165667C5

// STRIPE_BYTES is the block XXH64 consumes at a time: four 8-byte lanes, one per accumulator.
const STRIPE_BYTES = 32

// A lane is a 64-bit word read little-endian from the input — the unit a round folds into an
// accumulator. It is its own type so the round and merge helpers' two operands do not share a type
// (the house input-struct rule), naming the input word for what it is, distinct from state.
type lane uint64

// Digest is the streaming XXH64 state. Construct it with New_Digest; the zero value is usable only
// after a Digest_Reset. Fields are transparent, like prng.Generator's State.
type Digest struct {
	// Accumulator1 through Accumulator4 are the four lanes of XXH64 state; a stripe folds one
	// 8-byte lane into each, so a wide CPU can absorb four lanes at once.
	Accumulator1 uint64
	// Accumulator2 absorbs the second lane of each stripe.
	Accumulator2 uint64
	// Accumulator3 absorbs the third lane of each stripe.
	Accumulator3 uint64
	// Accumulator4 absorbs the fourth lane of each stripe.
	Accumulator4 uint64
	// Total_Bytes is every byte ever written, added into the hash before the final mix; it also
	// selects the short-input path at Sum time.
	Total_Bytes uint64
	// Seed is the construction seed, kept for Digest_Reset and the short-input offset.
	Seed uint64
	// Buffer holds bytes that did not complete a stripe, held until the next Write or Sum.
	Buffer [STRIPE_BYTES]byte
	// Buffer_Fill is how many bytes of Buffer are live (zero to STRIPE_BYTES minus one).
	Buffer_Fill int
}

// Hash returns the XXH64 of data under seed. seed zero is the common default.
func Hash(data []byte, seed uint64) (hash uint64) {
	total_bytes := uint64(len(data))
	var accumulator uint64
	if len(data) >= STRIPE_BYTES {
		accumulator_1 := seed + PRIME64_1 + PRIME64_2
		accumulator_2 := seed + PRIME64_2
		accumulator_3 := seed
		accumulator_4 := seed - PRIME64_1
		for len(data) >= STRIPE_BYTES {
			accumulator_1 = xxhash_round(accumulator_1, read_lane(data[0:8]))
			accumulator_2 = xxhash_round(accumulator_2, read_lane(data[8:16]))
			accumulator_3 = xxhash_round(accumulator_3, read_lane(data[16:24]))
			accumulator_4 = xxhash_round(accumulator_4, read_lane(data[24:32]))
			data = data[STRIPE_BYTES:]
		}
		accumulator = bits.RotateLeft64(accumulator_1, 1) +
			bits.RotateLeft64(accumulator_2, 7) +
			bits.RotateLeft64(accumulator_3, 12) +
			bits.RotateLeft64(accumulator_4, 18)
		accumulator = xxhash_merge_accumulator(accumulator, lane(accumulator_1))
		accumulator = xxhash_merge_accumulator(accumulator, lane(accumulator_2))
		accumulator = xxhash_merge_accumulator(accumulator, lane(accumulator_3))
		accumulator = xxhash_merge_accumulator(accumulator, lane(accumulator_4))
	} else {
		accumulator = seed + PRIME64_5
	}
	accumulator += total_bytes
	accumulator = xxhash_consume_tail(accumulator, data)
	return xxhash_avalanche(accumulator)
}

// New_Digest returns a streaming Digest seeded with seed, ready to Write.
func New_Digest(seed uint64) (digest Digest) {
	digest.Seed = seed
	digest_reset_accumulators(&digest)
	return digest
}

// Digest_Reset returns digest to the state of a fresh New_Digest with the same seed, so the state
// can be reused for another hash without reallocating.
func Digest_Reset(digest *Digest) {
	digest_reset_accumulators(digest)
}

// Write folds data into the running hash and reports every byte consumed with a nil error. It is
// the one method the house style permits: it makes *Digest an io.Writer, so io.Copy can stream
// into it. A Write never fails.
func (digest *Digest) Write(data []byte) (consumed int, err error) {
	consumed = len(data)
	digest.Total_Bytes += uint64(consumed)

	// Not enough buffered plus new to complete a stripe: stash it and wait for more.
	if digest.Buffer_Fill+len(data) < STRIPE_BYTES {
		copy(digest.Buffer[digest.Buffer_Fill:], data)
		digest.Buffer_Fill += len(data)
		return consumed, nil
	}

	// Finish the buffered partial stripe with the head of data, then fold it.
	if digest.Buffer_Fill > 0 {
		filled := copy(digest.Buffer[digest.Buffer_Fill:], data)
		data = data[filled:]
		digest_process_stripe(digest, digest.Buffer[:])
		digest.Buffer_Fill = 0
	}

	// Fold whole stripes straight from data.
	for len(data) >= STRIPE_BYTES {
		digest_process_stripe(digest, data[:STRIPE_BYTES])
		data = data[STRIPE_BYTES:]
	}

	// Stash the sub-stripe remainder for the next Write or the final Sum.
	copy(digest.Buffer[:], data)
	digest.Buffer_Fill = len(data)
	return consumed, nil
}

// Digest_Sum64 returns the XXH64 of everything written so far, without disturbing the state, so
// more can be written afterward.
func Digest_Sum64(digest *Digest) (hash uint64) {
	var accumulator uint64
	// With no completed stripe the short-input path is used; otherwise the four accumulators
	// converge. Total_Bytes, not Buffer_Fill, decides, since a long input can leave a tail.
	if digest.Total_Bytes >= STRIPE_BYTES {
		accumulator = bits.RotateLeft64(digest.Accumulator1, 1) +
			bits.RotateLeft64(digest.Accumulator2, 7) +
			bits.RotateLeft64(digest.Accumulator3, 12) +
			bits.RotateLeft64(digest.Accumulator4, 18)
		accumulator = xxhash_merge_accumulator(accumulator, lane(digest.Accumulator1))
		accumulator = xxhash_merge_accumulator(accumulator, lane(digest.Accumulator2))
		accumulator = xxhash_merge_accumulator(accumulator, lane(digest.Accumulator3))
		accumulator = xxhash_merge_accumulator(accumulator, lane(digest.Accumulator4))
	} else {
		accumulator = digest.Seed + PRIME64_5
	}
	accumulator += digest.Total_Bytes
	accumulator = xxhash_consume_tail(accumulator, digest.Buffer[:digest.Buffer_Fill])
	return xxhash_avalanche(accumulator)
}

// Resets the four accumulators to their seeded values and clears the byte count and buffer.
func digest_reset_accumulators(digest *Digest) {
	digest.Accumulator1 = digest.Seed + PRIME64_1 + PRIME64_2
	digest.Accumulator2 = digest.Seed + PRIME64_2
	digest.Accumulator3 = digest.Seed
	digest.Accumulator4 = digest.Seed - PRIME64_1
	digest.Total_Bytes = 0
	digest.Buffer_Fill = 0
}

// Folds one 32-byte stripe into the four accumulators, one 8-byte lane each.
func digest_process_stripe(digest *Digest, stripe []byte) {
	digest.Accumulator1 = xxhash_round(digest.Accumulator1, read_lane(stripe[0:8]))
	digest.Accumulator2 = xxhash_round(digest.Accumulator2, read_lane(stripe[8:16]))
	digest.Accumulator3 = xxhash_round(digest.Accumulator3, read_lane(stripe[16:24]))
	digest.Accumulator4 = xxhash_round(digest.Accumulator4, read_lane(stripe[24:32]))
}

// Reads eight bytes as a little-endian lane.
func read_lane(data []byte) (word lane) {
	return lane(binary.LittleEndian.Uint64(data))
}

// Folds one lane into an accumulator: scale by PRIME64_2, rotate left 31, multiply by PRIME64_1.
func xxhash_round(accumulator uint64, word lane) (mixed uint64) {
	accumulator += uint64(word) * PRIME64_2
	accumulator = bits.RotateLeft64(accumulator, 31)
	return accumulator * PRIME64_1
}

// Merges one converged accumulator (passed as a lane) into the running hash during finalization.
func xxhash_merge_accumulator(accumulator uint64, word lane) (merged uint64) {
	accumulator ^= xxhash_round(0, word)
	accumulator *= PRIME64_1
	return accumulator + PRIME64_4
}

// Consumes the up-to-31-byte tail: 8-byte lanes, then a 4-byte lane, then single bytes, each with
// its own rotate and primes (xxHash spec step 5). It runs after the byte count has been added.
func xxhash_consume_tail(accumulator uint64, tail []byte) (mixed uint64) {
	for len(tail) >= 8 {
		accumulator ^= xxhash_round(0, read_lane(tail[0:8]))
		accumulator = bits.RotateLeft64(accumulator, 27)*PRIME64_1 + PRIME64_4
		tail = tail[8:]
	}
	if len(tail) >= 4 {
		accumulator ^= uint64(binary.LittleEndian.Uint32(tail[0:4])) * PRIME64_1
		accumulator = bits.RotateLeft64(accumulator, 23)*PRIME64_2 + PRIME64_3
		tail = tail[4:]
	}
	for len(tail) >= 1 {
		accumulator ^= uint64(tail[0]) * PRIME64_5
		accumulator = bits.RotateLeft64(accumulator, 11) * PRIME64_1
		tail = tail[1:]
	}
	return accumulator
}

// Applies the final avalanche so every input bit can affect every output bit (xxHash spec step 6).
func xxhash_avalanche(accumulator uint64) (mixed uint64) {
	accumulator ^= accumulator >> 33
	accumulator *= PRIME64_2
	accumulator ^= accumulator >> 29
	accumulator *= PRIME64_3
	accumulator ^= accumulator >> 32
	return accumulator
}
