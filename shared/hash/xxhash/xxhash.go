// Package xxhash is XXH64 (Yann Collet's xxHash): a fast non-cryptographic 64-bit hash for
// fingerprints, deduplication, and in-memory hash tables keyed by trusted data. It is deliberately
// not collision-resistant against an adversary — for maps keyed by untrusted input use a keyed
// hash, and for unpredictable output use crypto/prng.
//
// Hash is one-shot form. Digest_Init, Digest_Write, and Digest_Sum_64 provide streaming form.
// Both produce identical output for identical bytes and seed.
//
// The algorithm and constants follow the xxHash specification; XXH64 is by Yann Collet.
package xxhash

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
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

// Accumulator_1 is first streaming mixing lane.
type Accumulator_1 uint64

// Accumulator_1_Invariants preserves complete first-lane state.
func Accumulator_1_Invariants(value Accumulator_1, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Accumulator_2 is second streaming mixing lane.
type Accumulator_2 uint64

// Accumulator_2_Invariants preserves complete second-lane state.
func Accumulator_2_Invariants(value Accumulator_2, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Accumulator_3 is third streaming mixing lane.
type Accumulator_3 uint64

// Accumulator_3_Invariants preserves complete third-lane state.
func Accumulator_3_Invariants(value Accumulator_3, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Accumulator_4 is fourth streaming mixing lane.
type Accumulator_4 uint64

// Accumulator_4_Invariants preserves complete fourth-lane state.
func Accumulator_4_Invariants(value Accumulator_4, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

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

// Count is bytes consumed by one bounded write.
type Count int

// Count_Invariants covers complete source count.
func Count_Invariants(value Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Message_Size is bytes accepted by current digest.
type Message_Size uint64

// Message_Size_Invariants preserves overflow-safe message accounting.
func Message_Size_Invariants(value Message_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Fill is live bytes in partial-stripe storage.
type Buffer_Fill uint8

// Buffer_Fill_Invariants excludes complete stripes.
func Buffer_Fill_Invariants(value Buffer_Fill, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), BUFFER_FILL_MINIMUM, BUFFER_FILL_MAXIMUM).
		Ensure()
}

// Buffer_Source is bytes copied into one partial stripe.
type Buffer_Source []byte

// Buffer_Source_Invariants excludes complete stripes.
func Buffer_Source_Invariants(value Buffer_Source, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), BUFFER_FILL_MINIMUM, BUFFER_FILL_MAXIMUM).
		Ensure()
}

// Buffer_Lane_1 packs first eight partial bytes.
type Buffer_Lane_1 uint64

// Buffer_Lane_1_Invariants preserves every packed first lane.
func Buffer_Lane_1_Invariants(value Buffer_Lane_1, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_2 packs second eight partial bytes.
type Buffer_Lane_2 uint64

// Buffer_Lane_2_Invariants preserves every packed second lane.
func Buffer_Lane_2_Invariants(value Buffer_Lane_2, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_3 packs third eight partial bytes.
type Buffer_Lane_3 uint64

// Buffer_Lane_3_Invariants preserves every packed third lane.
func Buffer_Lane_3_Invariants(value Buffer_Lane_3, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_4 packs fourth eight partial bytes.
type Buffer_Lane_4 uint64

// Buffer_Lane_4_Invariants preserves every packed fourth lane.
func Buffer_Lane_4_Invariants(value Buffer_Lane_4, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer packs partial bytes into little-endian lanes without fixed-array ownership.
type Buffer struct {
	// Lane_1 holds positions 0 through 7.
	Lane_1 Buffer_Lane_1
	// Lane_2 holds positions 8 through 15.
	Lane_2 Buffer_Lane_2
	// Lane_3 holds positions 16 through 23.
	Lane_3 Buffer_Lane_3
	// Lane_4 holds positions 24 through 31.
	Lane_4 Buffer_Lane_4
}

// Buffer_Invariants covers each opaque lane through one buffer chain.
func Buffer_Invariants(value Buffer, namespace aver.Namespace) {
	Buffer_Lane_1_Invariants(value.Lane_1, namespace)
	Buffer_Lane_2_Invariants(value.Lane_2, namespace)
	Buffer_Lane_3_Invariants(value.Lane_3, namespace)
	Buffer_Lane_4_Invariants(value.Lane_4, namespace)
}

// Four words avoid fixed storage while keeping partial bytes caller-owned.
func buffer_write(
	buffer Buffer, position Buffer_Fill, source Buffer_Source,
) (updated Buffer) {
	defer func() { Buffer_Invariants(updated, "buffer_write.updated") }()
	Buffer_Invariants(buffer, "buffer_write.buffer")
	Buffer_Fill_Invariants(position, "buffer_write.position")
	Buffer_Source_Invariants(source, "buffer_write.source")
	updated = buffer
	for _, item := range source {
		shift := uint(position%binary.UINT_64_SIZE) * binary.BITS_PER_BYTE
		mask := uint64(bits.WORD_8_MAXIMUM) << shift
		word := uint64(item) << shift
		switch position / binary.UINT_64_SIZE {
		case 0:
			updated.Lane_1 = Buffer_Lane_1(uint64(updated.Lane_1)&^mask | word)
		case 1:
			updated.Lane_2 = Buffer_Lane_2(uint64(updated.Lane_2)&^mask | word)
		case 2:
			updated.Lane_3 = Buffer_Lane_3(uint64(updated.Lane_3)&^mask | word)
		case 3:
			updated.Lane_4 = Buffer_Lane_4(uint64(updated.Lane_4)&^mask | word)
		default:
			panic("xxhash: partial stripe exceeds bound")
		}
		position++
	}
	return updated
}

// Digest is caller-owned streaming XXH64 state.
type Digest struct {
	// Accumulator_1 is first parallel lane.
	Accumulator_1 Accumulator_1
	// Accumulator_2 is second parallel lane.
	Accumulator_2 Accumulator_2
	// Accumulator_3 is third parallel lane.
	Accumulator_3 Accumulator_3
	// Accumulator_4 is fourth parallel lane.
	Accumulator_4 Accumulator_4
	// Total_Bytes keeps final length mixing exact.
	Total_Bytes Message_Size
	// Seed retains reset identity.
	Seed Seed
	// Buffer_Fill selects live Buffer prefix.
	Buffer_Fill Buffer_Fill
	// Buffer keeps partial bytes without fixed-array fields.
	Buffer Buffer
}

// Digest_Invariants composes caller-owned streaming state.
func Digest_Invariants(value Digest, namespace aver.Namespace) {
	Accumulator_1_Invariants(value.Accumulator_1, namespace)
	Accumulator_2_Invariants(value.Accumulator_2, namespace)
	Accumulator_3_Invariants(value.Accumulator_3, namespace)
	Accumulator_4_Invariants(value.Accumulator_4, namespace)
	Message_Size_Invariants(value.Total_Bytes, namespace)
	Seed_Invariants(value.Seed, namespace)
	Buffer_Fill_Invariants(value.Buffer_Fill, namespace)
	Buffer_Invariants(value.Buffer, namespace)
}

// Digest_Handle keeps caller-owned streaming state nonnil.
type Digest_Handle *Digest

// Digest_Handle_Invariants states state behind required handle.
func Digest_Handle_Invariants(value Digest_Handle, namespace aver.Namespace) {
	aver.Always(value != nil, "XXH64 digest handle exists.")
	Buffer_Invariants(value.Buffer, namespace)
	aver.Tree(value, namespace).
		Range_Uint64(
			uint64(value.Accumulator_1), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM,
		).
		Range_Uint64(
			uint64(value.Accumulator_2), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM,
		).
		Range_Uint64(
			uint64(value.Accumulator_3), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM,
		).
		Range_Uint64(
			uint64(value.Accumulator_4), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM,
		).
		Range_Uint64(
			uint64(value.Total_Bytes), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM,
		).
		Range_Uint64(
			uint64(value.Seed), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM,
		).
		Range_Uint8(
			uint8(value.Buffer_Fill), BUFFER_FILL_MINIMUM, BUFFER_FILL_MAXIMUM,
		).
		Ensure()
	aver.Always(
		uint64(value.Buffer_Fill) <= uint64(value.Total_Bytes),
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
				Lane(binary.Uint_64(
					binary.Bytes(source[0:binary.UINT_64_SIZE]),
					binary.LITTLE_ENDIAN,
				)),
			)
			accumulator_2 = xxhash_round(
				accumulator_2,
				Lane(binary.Uint_64(
					binary.Bytes(
						source[binary.UINT_64_SIZE:binary.UINT_64_SIZE*2],
					),
					binary.LITTLE_ENDIAN,
				)),
			)
			accumulator_3 = xxhash_round(
				accumulator_3,
				Lane(binary.Uint_64(
					binary.Bytes(
						source[binary.UINT_64_SIZE*2:binary.UINT_64_SIZE*3],
					),
					binary.LITTLE_ENDIAN,
				)),
			)
			accumulator_4 = xxhash_round(
				accumulator_4,
				Lane(binary.Uint_64(
					binary.Bytes(source[binary.UINT_64_SIZE*3:STRIPE_BYTES]),
					binary.LITTLE_ENDIAN,
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

// Digest_Init establishes seeded state in caller storage.
func Digest_Init(digest Digest_Handle, seed Seed) {
	Digest_Handle_Invariants(digest, "Digest_Init.digest.input")
	Seed_Invariants(seed, "Digest_Init.seed")
	digest.Accumulator_1 = Accumulator_1(uint64(seed) + PRIME64_1 + PRIME64_2)
	digest.Accumulator_2 = Accumulator_2(uint64(seed) + PRIME64_2)
	digest.Accumulator_3 = Accumulator_3(seed)
	digest.Accumulator_4 = Accumulator_4(uint64(seed) - PRIME64_1)
	digest.Total_Bytes = 0
	digest.Seed = seed
	digest.Buffer_Fill = 0
	digest.Buffer = Buffer{
		Lane_1: Buffer_Lane_1(seed), Lane_2: Buffer_Lane_2(seed),
		Lane_3: Buffer_Lane_3(seed), Lane_4: Buffer_Lane_4(seed),
	}
}

// Digest_Reset discards bytes while retaining seed.
func Digest_Reset(digest Digest_Handle) {
	Digest_Handle_Invariants(digest, "Digest_Reset.digest.input")
	Digest_Init(digest, digest.Seed)
}

// Digest_Write consumes one bounded source completely.
func Digest_Write(digest Digest_Handle, source Source) (consumed Count) {
	defer func() { Count_Invariants(consumed, "Digest_Write.consumed") }()
	Digest_Handle_Invariants(digest, "Digest_Write.digest.input")
	Source_Invariants(source, "Digest_Write.source")
	defer func() { Digest_Handle_Invariants(digest, "Digest_Write.digest.output") }()
	if len(source) > SOURCE_SIZE_MAXIMUM {
		panic("xxhash: source exceeds bound")
	}
	if uint64(digest.Total_Bytes) > bits.WORD_64_MAXIMUM-uint64(len(source)) {
		panic("xxhash: message exceeds bound")
	}
	consumed = Count(len(source))
	digest.Total_Bytes += Message_Size(len(source))

	process_lanes := func(lane_1 Lane, lane_2 Lane, lane_3 Lane, lane_4 Lane) {
		digest.Accumulator_1 = Accumulator_1(xxhash_round(
			Accumulator(digest.Accumulator_1), lane_1))
		digest.Accumulator_2 = Accumulator_2(xxhash_round(
			Accumulator(digest.Accumulator_2), lane_2))
		digest.Accumulator_3 = Accumulator_3(xxhash_round(
			Accumulator(digest.Accumulator_3), lane_3))
		digest.Accumulator_4 = Accumulator_4(xxhash_round(
			Accumulator(digest.Accumulator_4), lane_4))
	}

	buffer_fill := int(digest.Buffer_Fill)
	if buffer_fill+len(source) < STRIPE_BYTES {
		digest.Buffer = buffer_write(digest.Buffer, digest.Buffer_Fill,
			Buffer_Source(source))
		digest.Buffer_Fill += Buffer_Fill(len(source))
		return consumed
	}

	// Buffered head must fold before direct source stripes to preserve byte order.
	if buffer_fill > 0 {
		head_size := STRIPE_BYTES - buffer_fill
		digest.Buffer = buffer_write(
			digest.Buffer, digest.Buffer_Fill, Buffer_Source(source[:head_size]),
		)
		process_lanes(Lane(digest.Buffer.Lane_1), Lane(digest.Buffer.Lane_2),
			Lane(digest.Buffer.Lane_3), Lane(digest.Buffer.Lane_4))
		source = source[head_size:]
		digest.Buffer_Fill = 0
	}

	for len(source) >= STRIPE_BYTES {
		process_lanes(
			Lane(binary.Uint_64(
				binary.Bytes(source[0:binary.UINT_64_SIZE]), binary.LITTLE_ENDIAN,
			)),
			Lane(binary.Uint_64(
				binary.Bytes(source[binary.UINT_64_SIZE:binary.UINT_64_SIZE*2]),
				binary.LITTLE_ENDIAN,
			)),
			Lane(binary.Uint_64(
				binary.Bytes(source[binary.UINT_64_SIZE*2:binary.UINT_64_SIZE*3]),
				binary.LITTLE_ENDIAN,
			)),
			Lane(binary.Uint_64(
				binary.Bytes(source[binary.UINT_64_SIZE*3:STRIPE_BYTES]),
				binary.LITTLE_ENDIAN,
			)),
		)
		source = source[STRIPE_BYTES:]
	}

	digest.Buffer = buffer_write(digest.Buffer, 0, Buffer_Source(source))
	digest.Buffer_Fill = Buffer_Fill(len(source))
	return consumed
}

// Digest_Sum64 returns the XXH64 of everything written so far, without disturbing the state, so
// more can be written afterward.
func Digest_Sum_64(digest Digest_Handle) (hash Value) {
	defer func() { Value_Invariants(hash, "Digest_Sum_64.hash") }()
	Digest_Handle_Invariants(digest, "Digest_Sum_64.digest")
	var accumulator Accumulator
	// With no completed stripe the short-input path is used; otherwise the four accumulators
	// converge. Total_Bytes, not Buffer_Fill, decides, since a long input can leave a tail.
	if digest.Total_Bytes >= STRIPE_BYTES {
		accumulator = Accumulator(bits.Rotate_Left_64(
			bits.Word_64(digest.Accumulator_1), 1,
		)) + Accumulator(bits.Rotate_Left_64(
			bits.Word_64(digest.Accumulator_2), 7,
		)) + Accumulator(bits.Rotate_Left_64(
			bits.Word_64(digest.Accumulator_3), 12,
		)) + Accumulator(bits.Rotate_Left_64(
			bits.Word_64(digest.Accumulator_4), 18,
		))
		accumulator = xxhash_merge_accumulator(
			accumulator, Lane(digest.Accumulator_1),
		)
		accumulator = xxhash_merge_accumulator(
			accumulator, Lane(digest.Accumulator_2),
		)
		accumulator = xxhash_merge_accumulator(
			accumulator, Lane(digest.Accumulator_3),
		)
		accumulator = xxhash_merge_accumulator(
			accumulator, Lane(digest.Accumulator_4),
		)
	} else {
		accumulator = Accumulator(uint64(digest.Seed) + PRIME64_5)
	}
	accumulator += Accumulator(digest.Total_Bytes)
	var tail_storage [STRIPE_BYTES]byte
	for position := range int(digest.Buffer_Fill) {
		var lane Lane
		switch position / binary.UINT_64_SIZE {
		case 0:
			lane = Lane(digest.Buffer.Lane_1)
		case 1:
			lane = Lane(digest.Buffer.Lane_2)
		case 2:
			lane = Lane(digest.Buffer.Lane_3)
		case 3:
			lane = Lane(digest.Buffer.Lane_4)
		}
		shift := uint(position%binary.UINT_64_SIZE) * binary.BITS_PER_BYTE
		tail_storage[position] = byte(uint64(lane) >> shift)
	}
	accumulator = xxhash_consume_tail(
		accumulator, Tail(tail_storage[:digest.Buffer_Fill]),
	)
	return Value(xxhash_avalanche(accumulator))
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
		word := Lane(binary.Uint_64(
			binary.Bytes(tail[0:binary.UINT_64_SIZE]), binary.LITTLE_ENDIAN,
		))
		accumulator ^= xxhash_round(0, word)
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
