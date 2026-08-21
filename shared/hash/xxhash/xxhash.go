// Package xxhash is XXH64 (Yann Collet's xxHash): a fast non-cryptographic 64-bit hash for
// fingerprints, deduplication, and in-memory hash tables keyed by trusted data. It is deliberately
// not collision-resistant against an adversary — for maps keyed by untrusted input use a keyed
// hash, and for unpredictable output use crypto/prng.
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
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/simulation/aver/default"
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

// Lane separates one input word from mutable mixing state.
type Lane uint64

// Lane_Invariants preserves complete input-word domain.
func Lane_Invariants(value Lane, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Seed selects one deterministic XXH64 function.
type Seed uint64

// Seed_Invariants preserves complete seed domain.
func Seed_Invariants(value Seed, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Value is complete XXH64 result domain.
type Value uint64

// Value_Invariants preserves complete result domain.
func Value_Invariants(value Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Accumulator is one full-width XXH64 mixing state.
type Accumulator uint64

// Accumulator_Invariants preserves complete mixing-state domain.
func Accumulator_Invariants(value Accumulator, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// STATE_ACCUMULATOR_1_INDEX starts opaque state with first parallel lane.
const STATE_ACCUMULATOR_1_INDEX = 0

// STATE_ACCUMULATOR_2_INDEX follows first parallel lane.
const STATE_ACCUMULATOR_2_INDEX = STATE_ACCUMULATOR_1_INDEX + 1

// STATE_ACCUMULATOR_3_INDEX follows second parallel lane.
const STATE_ACCUMULATOR_3_INDEX = STATE_ACCUMULATOR_2_INDEX + 1

// STATE_ACCUMULATOR_4_INDEX follows third parallel lane.
const STATE_ACCUMULATOR_4_INDEX = STATE_ACCUMULATOR_3_INDEX + 1

// STATE_TOTAL_BYTES_INDEX keeps message length after parallel lanes.
const STATE_TOTAL_BYTES_INDEX = STATE_ACCUMULATOR_4_INDEX + 1

// STATE_SEED_INDEX keeps reset identity after message length.
const STATE_SEED_INDEX = STATE_TOTAL_BYTES_INDEX + 1

// STATE_BUFFER_FILL_INDEX keeps partial-stripe length after seed.
const STATE_BUFFER_FILL_INDEX = STATE_SEED_INDEX + 1

// STATE_WORD_COUNT derives exact caller-owned opaque storage.
const STATE_WORD_COUNT = STATE_BUFFER_FILL_INDEX + 1

// Source is one bounded input chunk.
type Source []byte

// Source_Invariants applies repository byte-call bound.
func Source_Invariants(value Source, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Tail is remainder after every complete stripe leaves one sub-stripe suffix.
type Tail []byte

// Tail_Invariants binds final mixing to one incomplete stripe.
func Tail_Invariants(value Tail, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), BUFFER_FILL_MINIMUM, BUFFER_FILL_MAXIMUM).
		Ensure()
}

// Digest is the streaming XXH64 state. Construct it with New_Digest; the zero value is usable only
// after a Digest_Reset. Fields are transparent, like prng.Xoshiro's State.
type Digest struct {
	// State keeps opaque machine words in fixed caller storage. Private element meaning keeps
	// machine-width implementation values from pretending to have narrower semantic domains.
	State [STATE_WORD_COUNT]uint64
	// Buffer holds bytes that did not complete a stripe, held until the next Write or Sum.
	Buffer [STRIPE_BYTES]byte
}

// Digest_Invariants composes caller-owned streaming state.
func Digest_Invariants(value Digest, namespace aver.Namespace) {
	aver.Always(
		value.State[STATE_BUFFER_FILL_INDEX] <= BUFFER_FILL_MAXIMUM,
		"Partial stripe fill cannot hold one complete stripe.",
	)
	aver.Always(
		value.State[STATE_BUFFER_FILL_INDEX] <=
			value.State[STATE_TOTAL_BYTES_INDEX],
		"Partial stripe cannot contain more bytes than complete message.",
	)
}

// Hash returns XXH64 of source under seed. Seed zero is common default.
func Hash(source Source, seed Seed) (hash Value) {
	defer func() { Value_Invariants(hash, "Hash.hash") }()
	Source_Invariants(source, "Hash.source")
	Seed_Invariants(seed, "Hash.seed")
	total_bytes := uint64(len(source))
	var accumulator Accumulator
	if len(source) >= STRIPE_BYTES {
		accumulator_1 := Accumulator(uint64(seed) + PRIME64_1 + PRIME64_2)
		accumulator_2 := Accumulator(uint64(seed) + PRIME64_2)
		accumulator_3 := Accumulator(seed)
		accumulator_4 := Accumulator(uint64(seed) - PRIME64_1)
		for len(source) >= STRIPE_BYTES {
			accumulator_1 = xxhash_round(
				accumulator_1,
				read_lane((*[binary.UINT_64_SIZE]byte)(
					source[0:binary.UINT_64_SIZE],
				)),
			)
			accumulator_2 = xxhash_round(
				accumulator_2,
				read_lane((*[binary.UINT_64_SIZE]byte)(
					source[binary.UINT_64_SIZE:binary.UINT_64_SIZE*2],
				)),
			)
			accumulator_3 = xxhash_round(
				accumulator_3,
				read_lane((*[binary.UINT_64_SIZE]byte)(
					source[binary.UINT_64_SIZE*2:binary.UINT_64_SIZE*3],
				)),
			)
			accumulator_4 = xxhash_round(
				accumulator_4,
				read_lane((*[binary.UINT_64_SIZE]byte)(
					source[binary.UINT_64_SIZE*3:STRIPE_BYTES],
				)),
			)
			source = source[STRIPE_BYTES:]
		}
		accumulator = Accumulator(bits.Rotate_Left_64(bits.Word_64(accumulator_1), 1)) +
			Accumulator(bits.Rotate_Left_64(bits.Word_64(accumulator_2), 7)) +
			Accumulator(bits.Rotate_Left_64(bits.Word_64(accumulator_3), 12)) +
			Accumulator(bits.Rotate_Left_64(bits.Word_64(accumulator_4), 18))
		accumulator = xxhash_merge_accumulator(accumulator, Lane(accumulator_1))
		accumulator = xxhash_merge_accumulator(accumulator, Lane(accumulator_2))
		accumulator = xxhash_merge_accumulator(accumulator, Lane(accumulator_3))
		accumulator = xxhash_merge_accumulator(accumulator, Lane(accumulator_4))
	} else {
		accumulator = Accumulator(uint64(seed) + PRIME64_5)
	}
	accumulator += Accumulator(total_bytes)
	accumulator = xxhash_consume_tail(accumulator, Tail(source))
	return Value(xxhash_avalanche(accumulator))
}

// New_Digest returns a streaming Digest seeded with seed, ready to Write.
func New_Digest(seed Seed) (digest Digest) {
	defer func() { Digest_Invariants(digest, "New_Digest.digest") }()
	Seed_Invariants(seed, "New_Digest.seed")
	digest.State[STATE_SEED_INDEX] = uint64(seed)
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
	if digest.State[STATE_TOTAL_BYTES_INDEX] > bits.WORD_64_MAXIMUM-uint64(len(data)) {
		panic("xxhash: message exceeds bound")
	}
	consumed = len(data)
	digest.State[STATE_TOTAL_BYTES_INDEX] += uint64(consumed)

	// Not enough buffered plus new to complete a stripe: stash it and wait for more.
	buffer_fill := int(digest.State[STATE_BUFFER_FILL_INDEX])
	if buffer_fill+len(data) < STRIPE_BYTES {
		copy(digest.Buffer[buffer_fill:], data)
		digest.State[STATE_BUFFER_FILL_INDEX] += uint64(len(data))
		return consumed, nil
	}

	// Finish the buffered partial stripe with the head of data, then fold it.
	if buffer_fill > 0 {
		filled := copy(digest.Buffer[buffer_fill:], data)
		data = data[filled:]
		digest_process_stripe(digest, &digest.Buffer)
		digest.State[STATE_BUFFER_FILL_INDEX] = 0
	}

	// Fold whole stripes straight from data.
	for len(data) >= STRIPE_BYTES {
		digest_process_stripe(digest, (*[STRIPE_BYTES]byte)(data[:STRIPE_BYTES]))
		data = data[STRIPE_BYTES:]
	}

	// Stash the sub-stripe remainder for the next Write or the final Sum.
	copy(digest.Buffer[:], data)
	digest.State[STATE_BUFFER_FILL_INDEX] = uint64(len(data))
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
	if digest.State[STATE_TOTAL_BYTES_INDEX] >= STRIPE_BYTES {
		accumulator = Accumulator(bits.Rotate_Left_64(
			bits.Word_64(digest.State[STATE_ACCUMULATOR_1_INDEX]), 1,
		)) + Accumulator(bits.Rotate_Left_64(
			bits.Word_64(digest.State[STATE_ACCUMULATOR_2_INDEX]), 7,
		)) + Accumulator(bits.Rotate_Left_64(
			bits.Word_64(digest.State[STATE_ACCUMULATOR_3_INDEX]), 12,
		)) + Accumulator(bits.Rotate_Left_64(
			bits.Word_64(digest.State[STATE_ACCUMULATOR_4_INDEX]), 18,
		))
		accumulator = xxhash_merge_accumulator(
			accumulator, Lane(digest.State[STATE_ACCUMULATOR_1_INDEX]),
		)
		accumulator = xxhash_merge_accumulator(
			accumulator, Lane(digest.State[STATE_ACCUMULATOR_2_INDEX]),
		)
		accumulator = xxhash_merge_accumulator(
			accumulator, Lane(digest.State[STATE_ACCUMULATOR_3_INDEX]),
		)
		accumulator = xxhash_merge_accumulator(
			accumulator, Lane(digest.State[STATE_ACCUMULATOR_4_INDEX]),
		)
	} else {
		accumulator = Accumulator(digest.State[STATE_SEED_INDEX] + PRIME64_5)
	}
	accumulator += Accumulator(digest.State[STATE_TOTAL_BYTES_INDEX])
	buffer_fill := digest.State[STATE_BUFFER_FILL_INDEX]
	accumulator = xxhash_consume_tail(accumulator, Tail(digest.Buffer[:buffer_fill]))
	return Value(xxhash_avalanche(accumulator))
}

// Resets the four accumulators to their seeded values and clears the byte count and buffer.
func digest_reset_accumulators(digest *Digest) {
	Digest_Invariants(*digest, "digest_reset_accumulators.digest.input")
	seed := digest.State[STATE_SEED_INDEX]
	digest.State[STATE_ACCUMULATOR_1_INDEX] = seed + PRIME64_1 + PRIME64_2
	digest.State[STATE_ACCUMULATOR_2_INDEX] = seed + PRIME64_2
	digest.State[STATE_ACCUMULATOR_3_INDEX] = seed
	digest.State[STATE_ACCUMULATOR_4_INDEX] = seed - PRIME64_1
	digest.State[STATE_TOTAL_BYTES_INDEX] = 0
	digest.State[STATE_BUFFER_FILL_INDEX] = 0
	Digest_Invariants(*digest, "digest_reset_accumulators.digest.output")
}

// Folds one 32-byte stripe into the four accumulators, one 8-byte lane each.
func digest_process_stripe(digest *Digest, stripe *[STRIPE_BYTES]byte) {
	Digest_Invariants(*digest, "digest_process_stripe.digest.input")
	digest.State[STATE_ACCUMULATOR_1_INDEX] = uint64(xxhash_round(
		Accumulator(digest.State[STATE_ACCUMULATOR_1_INDEX]),
		read_lane((*[binary.UINT_64_SIZE]byte)(stripe[0:binary.UINT_64_SIZE])),
	))
	digest.State[STATE_ACCUMULATOR_2_INDEX] = uint64(xxhash_round(
		Accumulator(digest.State[STATE_ACCUMULATOR_2_INDEX]),
		read_lane((*[binary.UINT_64_SIZE]byte)(
			stripe[binary.UINT_64_SIZE:binary.UINT_64_SIZE*2],
		)),
	))
	digest.State[STATE_ACCUMULATOR_3_INDEX] = uint64(xxhash_round(
		Accumulator(digest.State[STATE_ACCUMULATOR_3_INDEX]),
		read_lane((*[binary.UINT_64_SIZE]byte)(
			stripe[binary.UINT_64_SIZE*2:binary.UINT_64_SIZE*3],
		)),
	))
	digest.State[STATE_ACCUMULATOR_4_INDEX] = uint64(xxhash_round(
		Accumulator(digest.State[STATE_ACCUMULATOR_4_INDEX]),
		read_lane((*[binary.UINT_64_SIZE]byte)(
			stripe[binary.UINT_64_SIZE*3:STRIPE_BYTES],
		)),
	))
	Digest_Invariants(*digest, "digest_process_stripe.digest.output")
}

// Reads eight bytes as a little-endian lane.
func read_lane(source *[binary.UINT_64_SIZE]byte) (word Lane) {
	defer func() { Lane_Invariants(word, "read_lane.word") }()
	return Lane(binary.Uint_64(binary.Bytes(source[:]), binary.LITTLE_ENDIAN))
}

// Folds one lane into an accumulator: scale by PRIME64_2, rotate left 31, multiply by PRIME64_1.
func xxhash_round(accumulator Accumulator, word Lane) (mixed Accumulator) {
	defer func() { Accumulator_Invariants(mixed, "xxhash_round.mixed") }()
	Accumulator_Invariants(accumulator, "xxhash_round.accumulator")
	Lane_Invariants(word, "xxhash_round.word")
	accumulator += Accumulator(uint64(word) * PRIME64_2)
	accumulator = Accumulator(bits.Rotate_Left_64(bits.Word_64(accumulator), 31))
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
func xxhash_consume_tail(accumulator Accumulator, tail Tail) (mixed Accumulator) {
	defer func() { Accumulator_Invariants(mixed, "xxhash_consume_tail.mixed") }()
	Accumulator_Invariants(accumulator, "xxhash_consume_tail.accumulator")
	Tail_Invariants(tail, "xxhash_consume_tail.tail")
	for len(tail) >= binary.UINT_64_SIZE {
		accumulator ^= xxhash_round(0, read_lane(
			(*[binary.UINT_64_SIZE]byte)(tail[0:binary.UINT_64_SIZE]),
		))
		accumulator = Accumulator(bits.Rotate_Left_64(bits.Word_64(accumulator), 27))*
			PRIME64_1 + PRIME64_4
		tail = tail[binary.UINT_64_SIZE:]
	}
	if len(tail) >= binary.UINT_32_SIZE {
		word := binary.Uint_32(
			binary.Bytes(tail[0:binary.UINT_32_SIZE]), binary.LITTLE_ENDIAN,
		)
		accumulator ^= Accumulator(uint64(word) * PRIME64_1)
		accumulator = Accumulator(bits.Rotate_Left_64(bits.Word_64(accumulator), 23))*
			PRIME64_2 + PRIME64_3
		tail = tail[binary.UINT_32_SIZE:]
	}
	for len(tail) >= 1 {
		accumulator ^= Accumulator(uint64(tail[0]) * PRIME64_5)
		accumulator = Accumulator(bits.Rotate_Left_64(bits.Word_64(accumulator), 11)) *
			PRIME64_1
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
