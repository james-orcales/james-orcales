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
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
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
const STRIPE_LANE_COUNT = 4

// STRIPE_BYTES follows four 64-bit lanes from XXH64 specification.
const STRIPE_BYTES = STRIPE_LANE_COUNT * binary.UINT_64_SIZE

// BUFFER_FILL_MINIMUM is empty partial-stripe storage.
const BUFFER_FILL_MINIMUM = 0

// BUFFER_FILL_MAXIMUM leaves every complete stripe ready for immediate compression.
const BUFFER_FILL_MAXIMUM = STRIPE_BYTES - 1

// SOURCE_SIZE_MINIMUM is empty bounded input.
const SOURCE_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// SOURCE_SIZE_MAXIMUM shares repository byte-call bound.
const SOURCE_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// Lane is a 64-bit word read little-endian from input — unit one round folds into an
// accumulator. It is its own type so the round and merge helpers' two operands do not share a type
// (the house input-struct rule), naming the input word for what it is, distinct from state.
type Lane uint64

// Lane_Invariants preserves complete input-word domain.
func Lane_Invariants(value Lane, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Seed selects one deterministic XXH64 function.
type Seed uint64

// Seed_Invariants preserves complete seed domain.
func Seed_Invariants(value Seed, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Value is complete XXH64 result domain.
type Value uint64

// Value_Invariants preserves complete result domain.
func Value_Invariants(value Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Accumulator is one full-width XXH64 mixing state.
type Accumulator uint64

// Accumulator_Invariants preserves complete mixing-state domain.
func Accumulator_Invariants(value Accumulator, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Accumulator_1 owns first parallel lane state.
type Accumulator_1 uint64

// Accumulator_1_Invariants preserves complete first-lane domain.
func Accumulator_1_Invariants(value Accumulator_1, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Accumulator_2 owns second parallel lane state.
type Accumulator_2 uint64

// Accumulator_2_Invariants preserves complete second-lane domain.
func Accumulator_2_Invariants(value Accumulator_2, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Accumulator_3 owns third parallel lane state.
type Accumulator_3 uint64

// Accumulator_3_Invariants preserves complete third-lane domain.
func Accumulator_3_Invariants(value Accumulator_3, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Accumulator_4 owns fourth parallel lane state.
type Accumulator_4 uint64

// Accumulator_4_Invariants preserves complete fourth-lane domain.
func Accumulator_4_Invariants(value Accumulator_4, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Total_Bytes counts accepted bytes across bounded writes.
type Total_Bytes uint64

// Total_Bytes_Invariants preserves complete logical-message domain.
func Total_Bytes_Invariants(value Total_Bytes, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Fill counts live bytes in partial-stripe storage.
type Buffer_Fill int

// Buffer_Fill_Invariants excludes complete stripes because Write compresses them immediately.
func Buffer_Fill_Invariants(value Buffer_Fill, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), BUFFER_FILL_MINIMUM, BUFFER_FILL_MAXIMUM).
		Ensure()
}

// Source is one bounded input chunk.
type Source []byte

// Source_Invariants applies repository byte-call bound.
func Source_Invariants(value Source, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Digest is the streaming XXH64 state. Construct it with New_Digest; the zero value is usable only
// after a Digest_Reset. Fields are transparent, like prng.Generator's State.
type Digest struct {
	// Accumulator1 through Accumulator4 are the four lanes of XXH64 state; a stripe folds one
	// 8-byte lane into each, so a wide CPU can absorb four lanes at once.
	Accumulator_1 Accumulator_1
	// Accumulator2 absorbs the second lane of each stripe.
	Accumulator_2 Accumulator_2
	// Accumulator3 absorbs the third lane of each stripe.
	Accumulator_3 Accumulator_3
	// Accumulator4 absorbs the fourth lane of each stripe.
	Accumulator_4 Accumulator_4
	// Total_Bytes is every byte ever written, added into the hash before the final mix; it also
	// selects the short-input path at Sum time.
	Total_Bytes Total_Bytes
	// Seed is the construction seed, kept for Digest_Reset and the short-input offset.
	Seed Seed
	// Buffer holds bytes that did not complete a stripe, held until the next Write or Sum.
	Buffer [STRIPE_BYTES]byte
	// Buffer_Fill is how many bytes of Buffer are live (zero to STRIPE_BYTES minus one).
	Buffer_Fill Buffer_Fill
}

// Digest_Invariants composes caller-owned streaming state.
func Digest_Invariants(value Digest, namespace invariant.Namespace) {
	Accumulator_1_Invariants(value.Accumulator_1, namespace)
	Accumulator_2_Invariants(value.Accumulator_2, namespace)
	Accumulator_3_Invariants(value.Accumulator_3, namespace)
	Accumulator_4_Invariants(value.Accumulator_4, namespace)
	Total_Bytes_Invariants(value.Total_Bytes, namespace)
	Seed_Invariants(value.Seed, namespace)
	Buffer_Fill_Invariants(value.Buffer_Fill, namespace)
	invariant.Always(
		uint64(value.Buffer_Fill) <= uint64(value.Total_Bytes),
		"Partial stripe cannot contain more bytes than complete message.",
	)
}

// Hash returns XXH64 of source under seed. Seed zero is common default.
func Hash(source Source, seed Seed) (hash Value) {
	defer func() { Value_Invariants(hash, "Hash.hash") }()
	Source_Invariants(source, "Hash.source")
	Seed_Invariants(seed, "Hash.seed")
	total_bytes := Total_Bytes(len(source))
	var accumulator Accumulator
	if len(source) >= STRIPE_BYTES {
		accumulator_1 := Accumulator(uint64(seed) + PRIME64_1 + PRIME64_2)
		accumulator_2 := Accumulator(uint64(seed) + PRIME64_2)
		accumulator_3 := Accumulator(seed)
		accumulator_4 := Accumulator(uint64(seed) - PRIME64_1)
		for len(source) >= STRIPE_BYTES {
			accumulator_1 = xxhash_round(
				accumulator_1, read_lane(source[0:binary.UINT_64_SIZE]),
			)
			accumulator_2 = xxhash_round(
				accumulator_2,
				read_lane(source[binary.UINT_64_SIZE:binary.UINT_64_SIZE*2]),
			)
			accumulator_3 = xxhash_round(
				accumulator_3,
				read_lane(source[binary.UINT_64_SIZE*2:binary.UINT_64_SIZE*3]),
			)
			accumulator_4 = xxhash_round(
				accumulator_4,
				read_lane(source[binary.UINT_64_SIZE*3:STRIPE_BYTES]),
			)
			source = source[STRIPE_BYTES:]
		}
		accumulator = rotate_left(accumulator_1, 1) +
			rotate_left(accumulator_2, 7) +
			rotate_left(accumulator_3, 12) +
			rotate_left(accumulator_4, 18)
		accumulator = xxhash_merge_accumulator(accumulator, Lane(accumulator_1))
		accumulator = xxhash_merge_accumulator(accumulator, Lane(accumulator_2))
		accumulator = xxhash_merge_accumulator(accumulator, Lane(accumulator_3))
		accumulator = xxhash_merge_accumulator(accumulator, Lane(accumulator_4))
	} else {
		accumulator = Accumulator(uint64(seed) + PRIME64_5)
	}
	accumulator += Accumulator(total_bytes)
	accumulator = xxhash_consume_tail(accumulator, source)
	return Value(xxhash_avalanche(accumulator))
}

// New_Digest returns a streaming Digest seeded with seed, ready to Write.
func New_Digest(seed Seed) (digest Digest) {
	defer func() { Digest_Invariants(digest, "New_Digest.digest") }()
	Seed_Invariants(seed, "New_Digest.seed")
	digest.Seed = seed
	digest_reset_accumulators(&digest)
	return digest
}

// Digest_Reset returns digest to the state of a fresh New_Digest with the same seed, so the state
// can be reused for another hash without reallocating.
func Digest_Reset(digest *Digest) {
	Digest_Invariants(*digest, "Digest_Reset.digest.input")
	digest_reset_accumulators(digest)
	Digest_Invariants(*digest, "Digest_Reset.digest.output")
}

// Write folds data into the running hash and reports every byte consumed with a nil error. It is
// the one method the house style permits: it makes *Digest an io.Writer, so io.Copy can stream
// into it. A Write never fails.
func (digest *Digest) Write(data []byte) (consumed int, err error) {
	Digest_Invariants(*digest, "Digest.Write.digest.input")
	Source_Invariants(Source(data), "Digest.Write.data")
	defer func() { Digest_Invariants(*digest, "Digest.Write.digest.output") }()
	if len(data) > SOURCE_SIZE_MAXIMUM {
		panic("xxhash: source exceeds bound")
	}
	if uint64(digest.Total_Bytes) > bits.WORD_64_MAXIMUM-uint64(len(data)) {
		panic("xxhash: message exceeds bound")
	}
	consumed = len(data)
	digest.Total_Bytes += Total_Bytes(consumed)

	// Not enough buffered plus new to complete a stripe: stash it and wait for more.
	if int(digest.Buffer_Fill)+len(data) < STRIPE_BYTES {
		copy(digest.Buffer[digest.Buffer_Fill:], data)
		digest.Buffer_Fill += Buffer_Fill(len(data))
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
	digest.Buffer_Fill = Buffer_Fill(len(data))
	return consumed, nil
}

// Digest_Sum64 returns the XXH64 of everything written so far, without disturbing the state, so
// more can be written afterward.
func Digest_Sum_64(digest *Digest) (hash Value) {
	defer func() { Value_Invariants(hash, "Digest_Sum_64.hash") }()
	Digest_Invariants(*digest, "Digest_Sum_64.digest")
	var accumulator Accumulator
	// With no completed stripe the short-input path is used; otherwise the four accumulators
	// converge. Total_Bytes, not Buffer_Fill, decides, since a long input can leave a tail.
	if digest.Total_Bytes >= STRIPE_BYTES {
		accumulator = rotate_left(Accumulator(digest.Accumulator_1), 1) +
			rotate_left(Accumulator(digest.Accumulator_2), 7) +
			rotate_left(Accumulator(digest.Accumulator_3), 12) +
			rotate_left(Accumulator(digest.Accumulator_4), 18)
		accumulator = xxhash_merge_accumulator(accumulator, Lane(digest.Accumulator_1))
		accumulator = xxhash_merge_accumulator(accumulator, Lane(digest.Accumulator_2))
		accumulator = xxhash_merge_accumulator(accumulator, Lane(digest.Accumulator_3))
		accumulator = xxhash_merge_accumulator(accumulator, Lane(digest.Accumulator_4))
	} else {
		accumulator = Accumulator(uint64(digest.Seed) + PRIME64_5)
	}
	accumulator += Accumulator(digest.Total_Bytes)
	accumulator = xxhash_consume_tail(accumulator, digest.Buffer[:digest.Buffer_Fill])
	return Value(xxhash_avalanche(accumulator))
}

// Resets the four accumulators to their seeded values and clears the byte count and buffer.
func digest_reset_accumulators(digest *Digest) {
	Digest_Invariants(*digest, "digest_reset_accumulators.digest.input")
	digest.Accumulator_1 = Accumulator_1(uint64(digest.Seed) + PRIME64_1 + PRIME64_2)
	digest.Accumulator_2 = Accumulator_2(uint64(digest.Seed) + PRIME64_2)
	digest.Accumulator_3 = Accumulator_3(digest.Seed)
	digest.Accumulator_4 = Accumulator_4(uint64(digest.Seed) - PRIME64_1)
	digest.Total_Bytes = 0
	digest.Buffer_Fill = 0
	Digest_Invariants(*digest, "digest_reset_accumulators.digest.output")
}

// Folds one 32-byte stripe into the four accumulators, one 8-byte lane each.
func digest_process_stripe(digest *Digest, stripe Source) {
	Digest_Invariants(*digest, "digest_process_stripe.digest.input")
	Source_Invariants(stripe, "digest_process_stripe.stripe")
	invariant.Always(len(stripe) == STRIPE_BYTES, "Stripe has four complete lanes.")
	digest.Accumulator_1 = Accumulator_1(xxhash_round(
		Accumulator(digest.Accumulator_1), read_lane(stripe[0:binary.UINT_64_SIZE]),
	))
	digest.Accumulator_2 = Accumulator_2(xxhash_round(
		Accumulator(digest.Accumulator_2),
		read_lane(stripe[binary.UINT_64_SIZE:binary.UINT_64_SIZE*2]),
	))
	digest.Accumulator_3 = Accumulator_3(xxhash_round(
		Accumulator(digest.Accumulator_3),
		read_lane(stripe[binary.UINT_64_SIZE*2:binary.UINT_64_SIZE*3]),
	))
	digest.Accumulator_4 = Accumulator_4(xxhash_round(
		Accumulator(digest.Accumulator_4),
		read_lane(stripe[binary.UINT_64_SIZE*3:STRIPE_BYTES]),
	))
	Digest_Invariants(*digest, "digest_process_stripe.digest.output")
}

// Reads eight bytes as a little-endian lane.
func read_lane(source Source) (word Lane) {
	defer func() { Lane_Invariants(word, "read_lane.word") }()
	Source_Invariants(source, "read_lane.source")
	invariant.Always(
		len(source) == binary.UINT_64_SIZE,
		"Lane source has one complete machine word.",
	)
	return Lane(binary.Uint_64(binary.Bytes(source), binary.LITTLE_ENDIAN))
}

// Folds one lane into an accumulator: scale by PRIME64_2, rotate left 31, multiply by PRIME64_1.
func xxhash_round(accumulator Accumulator, word Lane) (mixed Accumulator) {
	defer func() { Accumulator_Invariants(mixed, "xxhash_round.mixed") }()
	Accumulator_Invariants(accumulator, "xxhash_round.accumulator")
	Lane_Invariants(word, "xxhash_round.word")
	accumulator += Accumulator(uint64(word) * PRIME64_2)
	accumulator = rotate_left(accumulator, 31)
	return accumulator * PRIME64_1
}

// Merges one converged accumulator (passed as a lane) into the running hash during finalization.
func xxhash_merge_accumulator(
	accumulator Accumulator, word Lane,
) (merged Accumulator) {
	defer func() { Accumulator_Invariants(merged, "xxhash_merge_accumulator.merged") }()
	Accumulator_Invariants(accumulator, "xxhash_merge_accumulator.accumulator")
	Lane_Invariants(word, "xxhash_merge_accumulator.word")
	accumulator ^= xxhash_round(0, word)
	accumulator *= PRIME64_1
	return accumulator + PRIME64_4
}

// Consumes the up-to-31-byte tail: 8-byte lanes, then a 4-byte lane, then single bytes, each with
// its own rotate and primes (xxHash spec step 5). It runs after the byte count has been added.
func xxhash_consume_tail(accumulator Accumulator, tail Source) (mixed Accumulator) {
	defer func() { Accumulator_Invariants(mixed, "xxhash_consume_tail.mixed") }()
	Accumulator_Invariants(accumulator, "xxhash_consume_tail.accumulator")
	Source_Invariants(tail, "xxhash_consume_tail.tail")
	for len(tail) >= binary.UINT_64_SIZE {
		accumulator ^= xxhash_round(0, read_lane(tail[0:binary.UINT_64_SIZE]))
		accumulator = rotate_left(accumulator, 27)*PRIME64_1 + PRIME64_4
		tail = tail[binary.UINT_64_SIZE:]
	}
	if len(tail) >= binary.UINT_32_SIZE {
		word := binary.Uint_32(
			binary.Bytes(tail[0:binary.UINT_32_SIZE]), binary.LITTLE_ENDIAN,
		)
		accumulator ^= Accumulator(
			uint64(word) * PRIME64_1,
		)
		accumulator = rotate_left(accumulator, 23)*PRIME64_2 + PRIME64_3
		tail = tail[binary.UINT_32_SIZE:]
	}
	for len(tail) >= 1 {
		accumulator ^= Accumulator(uint64(tail[0]) * PRIME64_5)
		accumulator = rotate_left(accumulator, 11) * PRIME64_1
		tail = tail[1:]
	}
	return accumulator
}

// Applies the final avalanche so every input bit can affect every output bit (xxHash spec step 6).
func xxhash_avalanche(accumulator Accumulator) (mixed Accumulator) {
	defer func() { Accumulator_Invariants(mixed, "xxhash_avalanche.mixed") }()
	Accumulator_Invariants(accumulator, "xxhash_avalanche.accumulator")
	accumulator ^= accumulator >> 33
	accumulator *= PRIME64_2
	accumulator ^= accumulator >> 29
	accumulator *= PRIME64_3
	accumulator ^= accumulator >> 32
	return accumulator
}

func rotate_left(value Accumulator, rotation bits.Rotation) (rotated Accumulator) {
	defer func() { Accumulator_Invariants(rotated, "rotate_left.rotated") }()
	Accumulator_Invariants(value, "rotate_left.value")
	bits.Rotation_Invariants(rotation, "rotate_left.rotation")
	return Accumulator(bits.Rotate_Left_64(bits.Word_64(value), rotation))
}
