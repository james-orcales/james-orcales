// Package sha256 computes FIPS 180-4 SHA-224 and SHA-256 with caller-owned state.
package sha256

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
)

// STATE_LANE_COUNT stores full SHA-2 chaining state.
const STATE_LANE_COUNT = binary.BITS_PER_BYTE

// DIGEST_224_LANE_COUNT removes the one truncated full-state lane.
const DIGEST_224_LANE_COUNT = STATE_LANE_COUNT - binary.UINT_8_SIZE

// DIGEST_224_BIT_COUNT derives SHA-224 width from retained state lanes.
const DIGEST_224_BIT_COUNT = DIGEST_224_LANE_COUNT * bits.BIT_COUNT_32_MAXIMUM

// DIGEST_256_BIT_COUNT derives SHA-256 width from full state lanes.
const DIGEST_256_BIT_COUNT = STATE_LANE_COUNT * bits.BIT_COUNT_32_MAXIMUM

// DIGEST_224_SIZE converts SHA-224 width to bytes.
const DIGEST_224_SIZE = DIGEST_224_BIT_COUNT / binary.BITS_PER_BYTE

// DIGEST_256_SIZE converts SHA-256 width to bytes.
const DIGEST_256_SIZE = DIGEST_256_BIT_COUNT / binary.BITS_PER_BYTE

// STATE_READY_INDEX follows compression lanes inside caller storage.
const STATE_READY_INDEX = STATE_LANE_COUNT

// STATE_WORD_COUNT holds compression lanes and initialization marker.
const STATE_WORD_COUNT = STATE_READY_INDEX + binary.UINT_8_SIZE

// STATE_READY_MARKER separates initialized state from zero caller storage.
const STATE_READY_MARKER = bits.WORD_32_MAXIMUM

// MESSAGE_WORD_COUNT is input words in one compression block.
const MESSAGE_WORD_COUNT = bits.BIT_COUNT_16_MAXIMUM

// BLOCK_SIZE derives compression width from its message words.
const BLOCK_SIZE = MESSAGE_WORD_COUNT * binary.UINT_32_SIZE

// MESSAGE_BIT_COUNT_SIZE is trailing encoded message-bit count width.
const MESSAGE_BIT_COUNT_SIZE = binary.UINT_64_SIZE

// PADDING_BOUNDARY leaves trailing bit count inside final block.
const PADDING_BOUNDARY = BLOCK_SIZE - MESSAGE_BIT_COUNT_SIZE

// FINAL_BLOCK_CAPACITY holds worst-case two-block finalization.
const FINAL_BLOCK_CAPACITY = BLOCK_SIZE * binary.UINT_16_SIZE

// PADDING_MARKER starts required one-bit suffix.
const PADDING_MARKER = binary.UINT_8_SIZE << (binary.BITS_PER_BYTE - binary.UINT_8_SIZE)

// SOURCE_SIZE_MINIMUM admits empty input.
const SOURCE_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// SOURCE_SIZE_MAXIMUM follows repository byte-slice bound.
const SOURCE_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// DESTINATION_SIZE_MINIMUM admits short-output status paths.
const DESTINATION_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// DESTINATION_SIZE_MAXIMUM follows repository byte-slice bound.
const DESTINATION_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// COUNT_MINIMUM is empty input.
const COUNT_MINIMUM = SOURCE_SIZE_MINIMUM

// COUNT_MAXIMUM is one complete bounded source.
const COUNT_MAXIMUM = SOURCE_SIZE_MAXIMUM

// BUFFER_COUNT_MINIMUM is empty partial block.
const BUFFER_COUNT_MINIMUM = SOURCE_SIZE_MINIMUM

// BUFFER_COUNT_MAXIMUM leaves complete blocks compressed immediately.
const BUFFER_COUNT_MAXIMUM = BLOCK_SIZE - binary.UINT_8_SIZE

// MESSAGE_SIZE_MINIMUM is empty message.
const MESSAGE_SIZE_MINIMUM uint64 = bits.WORD_64_MINIMUM

// MESSAGE_SIZE_MAXIMUM preserves exact 64-bit encoded bit count.
const MESSAGE_SIZE_MAXIMUM uint64 = bits.WORD_64_MAXIMUM / binary.BITS_PER_BYTE

// OUTPUT_COUNT_224_REQUIRED is required SHA-224 output width.
const OUTPUT_COUNT_224_REQUIRED Output_Count = DIGEST_224_SIZE

// OUTPUT_COUNT_256_REQUIRED is required SHA-256 output width.
const OUTPUT_COUNT_256_REQUIRED Output_Count = DIGEST_256_SIZE

// OUTPUT_STATUS_OK means complete digest reached caller storage.
const OUTPUT_STATUS_OK Output_Status = Output_Status(bits.WORD_8_MINIMUM)

// OUTPUT_STATUS_TOO_SMALL leaves short caller storage untouched.
const OUTPUT_STATUS_TOO_SMALL Output_Status = OUTPUT_STATUS_OK + binary.UINT_8_SIZE

// KIND_SHA_224 selects SHA-224 initial state and truncation.
const KIND_SHA_224 Kind = Kind(bits.WORD_8_MINIMUM)

// KIND_SHA_256 selects SHA-256 initial state and complete output.
const KIND_SHA_256 Kind = KIND_SHA_224 + binary.UINT_8_SIZE

// ROUND_SECTION_COUNT is the bounded round-constant lookup count.
const ROUND_SECTION_COUNT = binary.UINT_32_SIZE

// ROUND_SECTION_SIZE matches one initial message-word schedule.
const ROUND_SECTION_SIZE = MESSAGE_WORD_COUNT

// ROUND_COUNT derives total steps from equal lookup sections.
const ROUND_COUNT Round_Index = ROUND_SECTION_COUNT * ROUND_SECTION_SIZE

// ROUND_1_END is first lookup boundary.
const ROUND_1_END Round_Index = ROUND_SECTION_SIZE

// ROUND_2_END is second lookup boundary.
const ROUND_2_END Round_Index = ROUND_SECTION_SIZE * binary.UINT_16_SIZE

// ROUND_3_END is third lookup boundary.
const ROUND_3_END Round_Index = ROUND_SECTION_SIZE * (binary.UINT_16_SIZE + binary.UINT_8_SIZE)

// ROUND_INDEX_MINIMUM is first compression step.
const ROUND_INDEX_MINIMUM uint8 = bits.WORD_8_MINIMUM

// ROUND_INDEX_MAXIMUM is final compression step.
const ROUND_INDEX_MAXIMUM uint8 = uint8(ROUND_COUNT - binary.UINT_8_SIZE)

// ROUND_QUARTER_INDEX_MINIMUM is first step inside one 16-step quarter.
const ROUND_QUARTER_INDEX_MINIMUM uint8 = bits.WORD_8_MINIMUM

// ROUND_QUARTER_INDEX_MAXIMUM is final step inside one 16-step quarter.
const ROUND_QUARTER_INDEX_MAXIMUM uint8 = uint8(ROUND_SECTION_SIZE - binary.UINT_8_SIZE)

// ROUND_CONSTANT_WORD_COUNT stores one 32-bit word per additive constant.
const ROUND_CONSTANT_WORD_COUNT = binary.UINT_8_SIZE

// Source is one bounded input chunk.
type Source []byte

// Source_Invariants binds one call to repository byte boundary.
func Source_Invariants(value Source, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Blocks is one or more complete bounded compression blocks.
type Blocks []byte

// Blocks_Invariants excludes empty and partial compression input.
func Blocks_Invariants(value Blocks, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), BLOCK_SIZE, SOURCE_SIZE_MAXIMUM).
		Ensure()
	invariant.Always(
		len(value)%BLOCK_SIZE == SOURCE_SIZE_MINIMUM,
		"SHA-256 compression input contains complete blocks.",
	)
}

// Block is one complete compression block.
type Block []byte

// Block_Invariants fixes schedule input to one block.
func Block_Invariants(value Block, _ invariant.Namespace) {
	invariant.Always(len(value) == BLOCK_SIZE, "SHA-256 schedule input equals one block.")
}

// Destination is bounded caller-owned output storage.
type Destination []byte

// Destination_Invariants binds output to repository byte boundary.
func Destination_Invariants(value Destination, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), DESTINATION_SIZE_MINIMUM, DESTINATION_SIZE_MAXIMUM).
		Ensure()
}

// Count is bytes consumed by one bounded write.
type Count int

// Count_Invariants covers complete source count.
func Count_Invariants(value Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Output_Count is selected digest width on complete or short output.
type Output_Count uint8

// Output_Count_Invariants admits both selected digest widths.
func Output_Count_Invariants(value Output_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(OUTPUT_COUNT_224_REQUIRED),
			uint8(OUTPUT_COUNT_256_REQUIRED),
		).
		Ensure()
}

// Output_Status reports caller output capacity.
type Output_Status uint8

// Output_Status_Invariants covers complete and short output.
func Output_Status_Invariants(value Output_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(OUTPUT_STATUS_OK), uint8(OUTPUT_STATUS_TOO_SMALL)).
		Ensure()
}

// Kind selects SHA-224 or SHA-256.
type Kind uint8

// Kind_Invariants admits two FIPS 180-4 functions.
func Kind_Invariants(value Kind, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(KIND_SHA_224), uint8(KIND_SHA_256)).
		Ensure()
}

// Buffer_Count is live bytes in partial block storage.
type Buffer_Count uint8

// Buffer_Count_Invariants excludes complete blocks.
func Buffer_Count_Invariants(value Buffer_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), BUFFER_COUNT_MINIMUM, BUFFER_COUNT_MAXIMUM).
		Ensure()
}

// Message_Size is bytes accepted by current digest.
type Message_Size uint64

// Message_Size_Invariants preserves exact final bit-count encoding.
func Message_Size_Invariants(value Message_Size, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), MESSAGE_SIZE_MINIMUM, MESSAGE_SIZE_MAXIMUM).
		Ensure()
}

// State holds SHA-256 compression lanes and caller-storage identity.
type State [STATE_WORD_COUNT]uint32

// State_Invariants fixes compression state and identity storage width.
func State_Invariants(value State, _ invariant.Namespace) {
	invariant.Always(
		len(value) == STATE_WORD_COUNT,
		"SHA-256 state storage has fixed width.",
	)
}

// Round_Index identifies one compression step.
type Round_Index uint8

// Round_Index_Invariants covers every compression step.
func Round_Index_Invariants(value Round_Index, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), ROUND_INDEX_MINIMUM, ROUND_INDEX_MAXIMUM).
		Ensure()
}

// Round_Quarter_Index identifies one step inside a 16-step round quarter.
type Round_Quarter_Index uint8

// Round_Quarter_Index_Invariants covers every quarter-local step.
func Round_Quarter_Index_Invariants(value Round_Quarter_Index, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(
			uint8(value), ROUND_QUARTER_INDEX_MINIMUM, ROUND_QUARTER_INDEX_MAXIMUM,
		).
		Ensure()
}

// Round_Constant stores one opaque FIPS 180-4 additive word.
type Round_Constant [ROUND_CONSTANT_WORD_COUNT]uint32

// Round_Constant_Invariants fixes one-word constant storage.
func Round_Constant_Invariants(value Round_Constant, _ invariant.Namespace) {
	invariant.Always(
		len(value) == ROUND_CONSTANT_WORD_COUNT,
		"SHA-256 round constant occupies one word.",
	)
}

// Schedule holds expanded words for one compression block.
type Schedule [ROUND_COUNT]uint32

// Schedule_Invariants fixes one expanded block width.
func Schedule_Invariants(value Schedule, _ invariant.Namespace) {
	invariant.Always(len(value) == int(ROUND_COUNT), "SHA-256 schedule has derived width.")
}

// Size is one supported digest width.
type Size int

// Size_Invariants admits SHA-224 and SHA-256 widths.
func Size_Invariants(value Size, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Int(int(value), DIGEST_224_SIZE, DIGEST_256_SIZE).
		Ensure()
}

// Block_Size is shared SHA-224 and SHA-256 compression-block width.
type Block_Size int

// Block_Size_Invariants fixes shared compression-block width.
func Block_Size_Invariants(value Block_Size, _ invariant.Namespace) {
	invariant.Always(
		int(value) == BLOCK_SIZE,
		"SHA-256 block size equals compression width.",
	)
}

// Value_224 is one SHA-224 digest in wire byte order.
type Value_224 [DIGEST_224_SIZE]byte

// Value_224_Invariants fixes SHA-224 storage width.
func Value_224_Invariants(value Value_224, _ invariant.Namespace) {
	invariant.Always(len(value) == DIGEST_224_SIZE, "SHA-224 digest has derived width.")
}

// Value_256 is one SHA-256 digest in wire byte order.
type Value_256 [DIGEST_256_SIZE]byte

// Value_256_Invariants fixes SHA-256 storage width.
func Value_256_Invariants(value Value_256, _ invariant.Namespace) {
	invariant.Always(len(value) == DIGEST_256_SIZE, "SHA-256 digest has derived width.")
}

// Digest is caller-owned SHA-224 or SHA-256 state.
type Digest struct {
	// State holds eight compression lanes.
	State State
	// Buffer holds one incomplete block.
	Buffer [BLOCK_SIZE]byte
	// Buffer_Count identifies live Buffer prefix.
	Buffer_Count Buffer_Count
	// Message_Size counts accepted bytes for final bit-count encoding.
	Message_Size Message_Size
	// Kind retains selected function across Reset.
	Kind Kind
}

// Digest_Invariants composes selected function and partial-block relation.
func Digest_Invariants(value Digest, namespace invariant.Namespace) {
	State_Invariants(value.State, namespace)
	Buffer_Count_Invariants(value.Buffer_Count, namespace)
	Message_Size_Invariants(value.Message_Size, namespace)
	Kind_Invariants(value.Kind, namespace)
	invariant.Always(
		uint64(value.Buffer_Count) == uint64(value.Message_Size)%BLOCK_SIZE,
		"Partial SHA-256 block equals message remainder.",
	)
}

// Digest_Init establishes selected FIPS 180-4 initial state.
func Digest_Init(digest *Digest, kind Kind) {
	Digest_Invariants(*digest, "Digest_Init.digest.input")
	Kind_Invariants(kind, "Digest_Init.kind")
	digest.Kind = kind
	digest_reset(digest)
	State_Invariants(digest.State, "Digest_Init.digest.state.output")
	invariant.Always(
		digest.Buffer_Count == BUFFER_COUNT_MINIMUM,
		"Fresh SHA-256 state has no buffered bytes.",
	)
	invariant.Always(
		uint64(digest.Message_Size) == MESSAGE_SIZE_MINIMUM,
		"Fresh SHA-256 state has no accepted bytes.",
	)
	Kind_Invariants(digest.Kind, "Digest_Init.digest.kind.output")
}

// Digest_Reset discards message while retaining selected function.
func Digest_Reset(digest *Digest) {
	Digest_Invariants(*digest, "Digest_Reset.digest.input")
	digest_require(digest)
	digest_reset(digest)
	State_Invariants(digest.State, "Digest_Reset.digest.state.output")
	invariant.Always(
		digest.Buffer_Count == BUFFER_COUNT_MINIMUM,
		"Fresh reset SHA-256 state has no buffered bytes.",
	)
	invariant.Always(
		uint64(digest.Message_Size) == MESSAGE_SIZE_MINIMUM,
		"Fresh reset SHA-256 state has no accepted bytes.",
	)
	Kind_Invariants(digest.Kind, "Digest_Reset.digest.kind.output")
}

// Digest_Write consumes one bounded source completely.
func Digest_Write(digest *Digest, source Source) (count Count) {
	defer func() { Count_Invariants(count, "Digest_Write.count") }()
	Digest_Invariants(*digest, "Digest_Write.digest.input")
	Source_Invariants(source, "Digest_Write.source")
	digest_require(digest)
	if len(source) > SOURCE_SIZE_MAXIMUM {
		panic("sha256: source exceeds bound")
	}
	if uint64(digest.Message_Size) > MESSAGE_SIZE_MAXIMUM-uint64(len(source)) {
		panic("sha256: message exceeds bound")
	}
	digest_write(digest, source)
	Digest_Invariants(*digest, "Digest_Write.digest.output")
	return Count(len(source))
}

// Digest_Sum_224 observes selected SHA-224 state without consuming it.
func Digest_Sum_224(digest *Digest) (value Value_224) {
	defer func() { Value_224_Invariants(value, "Digest_Sum_224.value") }()
	Digest_Invariants(*digest, "Digest_Sum_224.digest")
	digest_require_kind(digest, KIND_SHA_224)
	full := digest_sum_full(digest)
	copy(value[:], full[:DIGEST_224_SIZE])
	return value
}

// Digest_Sum_256 observes selected SHA-256 state without consuming it.
func Digest_Sum_256(digest *Digest) (value Value_256) {
	defer func() { Value_256_Invariants(value, "Digest_Sum_256.value") }()
	Digest_Invariants(*digest, "Digest_Sum_256.digest")
	digest_require_kind(digest, KIND_SHA_256)
	return digest_sum_full(digest)
}

// Digest_Sum_Into writes selected complete digest or leaves short storage untouched.
func Digest_Sum_Into(
	digest *Digest, destination Destination,
) (count Output_Count, status Output_Status) {
	defer func() {
		Output_Count_Invariants(count, "Digest_Sum_Into.count")
		Output_Status_Invariants(status, "Digest_Sum_Into.status")
	}()
	Digest_Invariants(*digest, "Digest_Sum_Into.digest")
	Destination_Invariants(destination, "Digest_Sum_Into.destination")
	digest_require(digest)
	if len(destination) > DESTINATION_SIZE_MAXIMUM {
		panic("sha256: destination exceeds bound")
	}
	size := Digest_Size(digest)
	count = Output_Count(size)
	if len(destination) < int(size) {
		return count, OUTPUT_STATUS_TOO_SMALL
	}
	full := digest_sum_full(digest)
	copy(destination[:size], full[:size])
	return count, OUTPUT_STATUS_OK
}

// Digest_Clone_Into copies live state without aliasing caller storage.
func Digest_Clone_Into(destination *Digest, source *Digest) {
	Digest_Invariants(*destination, "Digest_Clone_Into.destination.input")
	Digest_Invariants(*source, "Digest_Clone_Into.source")
	digest_require(source)
	*destination = *source
	Digest_Invariants(*destination, "Digest_Clone_Into.destination.output")
}

// Digest_Size reports selected output width.
func Digest_Size(digest *Digest) (size Size) {
	defer func() { Size_Invariants(size, "Digest_Size.size") }()
	Digest_Invariants(*digest, "Digest_Size.digest")
	digest_require(digest)
	if digest.Kind == KIND_SHA_224 {
		return DIGEST_224_SIZE
	}
	return DIGEST_256_SIZE
}

// Digest_Block_Size reports shared compression block width.
func Digest_Block_Size(digest *Digest) (size Block_Size) {
	defer func() { Block_Size_Invariants(size, "Digest_Block_Size.size") }()
	Digest_Invariants(*digest, "Digest_Block_Size.digest")
	digest_require(digest)
	return BLOCK_SIZE
}

// Checksum_224 computes one bounded source without retained state.
func Checksum_224(source Source) (value Value_224) {
	defer func() { Value_224_Invariants(value, "Checksum_224.value") }()
	Source_Invariants(source, "Checksum_224.source")
	if len(source) > SOURCE_SIZE_MAXIMUM {
		panic("sha256: source exceeds bound")
	}
	var digest Digest
	Digest_Init(&digest, KIND_SHA_224)
	Digest_Write(&digest, source)
	return Digest_Sum_224(&digest)
}

// Checksum_256 computes one bounded source without retained state.
func Checksum_256(source Source) (value Value_256) {
	defer func() { Value_256_Invariants(value, "Checksum_256.value") }()
	Source_Invariants(source, "Checksum_256.source")
	if len(source) > SOURCE_SIZE_MAXIMUM {
		panic("sha256: source exceeds bound")
	}
	var digest Digest
	Digest_Init(&digest, KIND_SHA_256)
	Digest_Write(&digest, source)
	return Digest_Sum_256(&digest)
}

func digest_sum_full(digest *Digest) (value Value_256) {
	defer func() { Value_256_Invariants(value, "digest_sum_full.value") }()
	Digest_Invariants(*digest, "digest_sum_full.digest")
	digest_require(digest)
	copy_digest := *digest
	var final_blocks [FINAL_BLOCK_CAPACITY]byte
	buffer_count := int(digest.Buffer_Count)
	copy(final_blocks[:buffer_count], digest.Buffer[:buffer_count])
	final_blocks[buffer_count] = PADDING_MARKER
	final_size := BLOCK_SIZE
	if buffer_count+binary.UINT_8_SIZE > PADDING_BOUNDARY {
		final_size = FINAL_BLOCK_CAPACITY
	}
	message_bits := uint64(digest.Message_Size) * binary.BITS_PER_BYTE
	binary.Put_Uint_64(
		binary.Bytes(final_blocks[final_size-MESSAGE_BIT_COUNT_SIZE:final_size]),
		binary.Word_64(message_bits), binary.BIG_ENDIAN,
	)
	block(&copy_digest.State, Blocks(final_blocks[:final_size]))
	for index := range STATE_LANE_COUNT {
		start := index * binary.UINT_32_SIZE
		binary.Put_Uint_32(
			binary.Bytes(value[start:start+binary.UINT_32_SIZE]),
			binary.Word_32(copy_digest.State[index]), binary.BIG_ENDIAN,
		)
	}
	return value
}

func digest_write(digest *Digest, source Source) {
	Digest_Invariants(*digest, "digest_write.digest.input")
	Source_Invariants(source, "digest_write.source")
	digest.Message_Size += Message_Size(len(source))
	if digest.Buffer_Count > BUFFER_COUNT_MINIMUM {
		copied := copy(digest.Buffer[digest.Buffer_Count:], source)
		digest.Buffer_Count += Buffer_Count(copied)
		source = source[copied:]
		if int(digest.Buffer_Count) == BLOCK_SIZE {
			block(&digest.State, Blocks(digest.Buffer[:]))
			digest.Buffer_Count = BUFFER_COUNT_MINIMUM
		}
	}
	if len(source) >= BLOCK_SIZE {
		complete_size := len(source) / BLOCK_SIZE * BLOCK_SIZE
		block(&digest.State, Blocks(source[:complete_size]))
		source = source[complete_size:]
	}
	if len(source) > SOURCE_SIZE_MINIMUM {
		digest.Buffer_Count = Buffer_Count(copy(digest.Buffer[:], source))
	}
	Digest_Invariants(*digest, "digest_write.digest.output")
}

func digest_reset(digest *Digest) {
	Digest_Invariants(*digest, "digest_reset.digest.input")
	Kind_Invariants(digest.Kind, "digest_reset.kind")
	// Exact FIPS words avoid recomputing square-root fractions through floating point.
	if digest.Kind == KIND_SHA_224 {
		digest.State = State{
			0xc1059ed8, 0x367cd507, 0x3070dd17, 0xf70e5939,
			0xffc00b31, 0x68581511, 0x64f98fa7, 0xbefa4fa4,
		}
	} else {
		digest.State = State{
			0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a,
			0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19,
		}
	}
	digest.State[STATE_READY_INDEX] = STATE_READY_MARKER
	digest.Buffer = [BLOCK_SIZE]byte{}
	digest.Buffer_Count = BUFFER_COUNT_MINIMUM
	digest.Message_Size = Message_Size(MESSAGE_SIZE_MINIMUM)
	State_Invariants(digest.State, "digest_reset.digest.state.output")
	invariant.Always(
		digest.Buffer_Count == BUFFER_COUNT_MINIMUM,
		"Reset SHA-256 state has no buffered bytes.",
	)
	invariant.Always(
		uint64(digest.Message_Size) == MESSAGE_SIZE_MINIMUM,
		"Reset SHA-256 state has no accepted bytes.",
	)
	Kind_Invariants(digest.Kind, "digest_reset.digest.kind.output")
}

func block(state *State, source Blocks) {
	State_Invariants(*state, "block.state.input")
	Blocks_Invariants(source, "block.source")
	const STATE_A_INDEX = bytes.SLICE_SIZE_MINIMUM
	const STATE_B_INDEX = STATE_A_INDEX + binary.UINT_8_SIZE
	const STATE_C_INDEX = STATE_B_INDEX + binary.UINT_8_SIZE
	const STATE_D_INDEX = STATE_C_INDEX + binary.UINT_8_SIZE
	const STATE_E_INDEX = STATE_D_INDEX + binary.UINT_8_SIZE
	const STATE_F_INDEX = STATE_E_INDEX + binary.UINT_8_SIZE
	const STATE_G_INDEX = STATE_F_INDEX + binary.UINT_8_SIZE
	const STATE_H_INDEX = STATE_G_INDEX + binary.UINT_8_SIZE
	const STATE_E_ROTATION_FIRST bits.Rotation = -bits.Rotation(
		bits.BIT_COUNT_8_MAXIMUM - binary.UINT_16_SIZE,
	)
	const STATE_E_ROTATION_SECOND bits.Rotation = -bits.Rotation(
		bits.BIT_COUNT_8_MAXIMUM + STATE_D_INDEX,
	)
	const STATE_E_ROTATION_THIRD bits.Rotation = -bits.Rotation(
		bits.BIT_COUNT_32_MAXIMUM - bits.BIT_COUNT_8_MAXIMUM + binary.UINT_8_SIZE,
	)
	const STATE_A_ROTATION_FIRST bits.Rotation = -bits.Rotation(binary.UINT_16_SIZE)
	const STATE_A_ROTATION_SECOND bits.Rotation = -bits.Rotation(
		bits.BIT_COUNT_16_MAXIMUM - STATE_D_INDEX,
	)
	const STATE_A_ROTATION_THIRD bits.Rotation = -bits.Rotation(
		bits.BIT_COUNT_32_MAXIMUM - bits.BIT_COUNT_8_MAXIMUM - binary.UINT_16_SIZE,
	)
	for len(source) >= BLOCK_SIZE {
		schedule := schedule_make(Block(source[:BLOCK_SIZE]))
		a := state[STATE_A_INDEX]
		b := state[STATE_B_INDEX]
		c := state[STATE_C_INDEX]
		d := state[STATE_D_INDEX]
		e := state[STATE_E_INDEX]
		f := state[STATE_F_INDEX]
		g := state[STATE_G_INDEX]
		h := state[STATE_H_INDEX]
		for index := Round_Index(ROUND_INDEX_MINIMUM); index < ROUND_COUNT; index++ {
			sigma_1 := bits.Rotate_Left_32(bits.Word_32(e), STATE_E_ROTATION_FIRST)
			sigma_1 ^= bits.Rotate_Left_32(bits.Word_32(e), STATE_E_ROTATION_SECOND)
			sigma_1 ^= bits.Rotate_Left_32(bits.Word_32(e), STATE_E_ROTATION_THIRD)
			choice := e&f ^ ^e&g
			temporary_1 := h + uint32(sigma_1) + choice
			temporary_1 += round_constant(index)[bytes.SLICE_SIZE_MINIMUM]
			temporary_1 += schedule[index]
			sigma_0 := bits.Rotate_Left_32(bits.Word_32(a), STATE_A_ROTATION_FIRST)
			sigma_0 ^= bits.Rotate_Left_32(bits.Word_32(a), STATE_A_ROTATION_SECOND)
			sigma_0 ^= bits.Rotate_Left_32(bits.Word_32(a), STATE_A_ROTATION_THIRD)
			majority := a&b ^ a&c ^ b&c
			temporary_2 := uint32(sigma_0) + majority
			next_e := d + temporary_1
			a, b, c, d = temporary_1+temporary_2, a, b, c
			e, f, g, h = next_e, e, f, g
		}
		state[STATE_A_INDEX] += a
		state[STATE_B_INDEX] += b
		state[STATE_C_INDEX] += c
		state[STATE_D_INDEX] += d
		state[STATE_E_INDEX] += e
		state[STATE_F_INDEX] += f
		state[STATE_G_INDEX] += g
		state[STATE_H_INDEX] += h
		source = source[BLOCK_SIZE:]
	}
	State_Invariants(*state, "block.state.output")
}

func schedule_make(source Block) (schedule Schedule) {
	defer func() { Schedule_Invariants(schedule, "schedule_make.schedule") }()
	Block_Invariants(source, "schedule_make.source")
	const STATE_D_INDEX = binary.UINT_32_SIZE - binary.UINT_8_SIZE
	const STATE_H_INDEX = binary.BITS_PER_BYTE - binary.UINT_8_SIZE
	const SCHEDULE_VALUE_1_INDEX Round_Index = binary.UINT_16_SIZE
	const SCHEDULE_VALUE_1_ROTATION_FIRST bits.Rotation = -bits.Rotation(
		bits.BIT_COUNT_16_MAXIMUM + binary.UINT_8_SIZE,
	)
	const SCHEDULE_VALUE_1_ROTATION_SECOND bits.Rotation = -bits.Rotation(
		bits.BIT_COUNT_16_MAXIMUM + STATE_D_INDEX,
	)
	const SCHEDULE_VALUE_1_SHIFT = bits.BIT_COUNT_8_MAXIMUM + binary.UINT_16_SIZE
	const SCHEDULE_VALUE_2_INDEX Round_Index = MESSAGE_WORD_COUNT - binary.UINT_8_SIZE
	const SCHEDULE_VALUE_2_ROTATION_FIRST bits.Rotation = -bits.Rotation(
		bits.BIT_COUNT_8_MAXIMUM - binary.UINT_8_SIZE,
	)
	const SCHEDULE_VALUE_2_ROTATION_SECOND bits.Rotation = -bits.Rotation(
		bits.BIT_COUNT_16_MAXIMUM + binary.UINT_16_SIZE,
	)
	const SCHEDULE_VALUE_2_SHIFT = STATE_D_INDEX
	const SCHEDULE_SUM_1_INDEX Round_Index = STATE_H_INDEX
	const SCHEDULE_SUM_2_INDEX Round_Index = MESSAGE_WORD_COUNT
	for index := range MESSAGE_WORD_COUNT {
		start := index * binary.UINT_32_SIZE
		word_source := source[start : start+binary.UINT_32_SIZE]
		schedule[index] = uint32(binary.Uint_32(
			binary.Bytes(word_source), binary.BIG_ENDIAN,
		))
	}
	for index := Round_Index(MESSAGE_WORD_COUNT); index < ROUND_COUNT; index++ {
		value_1 := schedule[index-SCHEDULE_VALUE_1_INDEX]
		sigma_1 := bits.Rotate_Left_32(
			bits.Word_32(value_1), SCHEDULE_VALUE_1_ROTATION_FIRST,
		)
		sigma_1 ^= bits.Rotate_Left_32(
			bits.Word_32(value_1), SCHEDULE_VALUE_1_ROTATION_SECOND,
		)
		sigma_1 ^= bits.Word_32(value_1 >> SCHEDULE_VALUE_1_SHIFT)
		value_2 := schedule[index-SCHEDULE_VALUE_2_INDEX]
		sigma_2 := bits.Rotate_Left_32(
			bits.Word_32(value_2), SCHEDULE_VALUE_2_ROTATION_FIRST,
		)
		sigma_2 ^= bits.Rotate_Left_32(
			bits.Word_32(value_2), SCHEDULE_VALUE_2_ROTATION_SECOND,
		)
		sigma_2 ^= bits.Word_32(value_2 >> SCHEDULE_VALUE_2_SHIFT)
		schedule[index] = uint32(sigma_1) +
			schedule[index-SCHEDULE_SUM_1_INDEX]
		schedule[index] += uint32(sigma_2) +
			schedule[index-SCHEDULE_SUM_2_INDEX]
	}
	return schedule
}

// FIPS derives K[i] from fractional cube roots of primes; exact words avoid floating-point drift.
func round_constant(index Round_Index) (constant Round_Constant) {
	defer func() { Round_Constant_Invariants(constant, "round_constant.constant") }()
	Round_Index_Invariants(index, "round_constant.index")
	if index < ROUND_1_END {
		return round_constant_0_15(Round_Quarter_Index(index))
	}
	if index < ROUND_2_END {
		return round_constant_16_31(Round_Quarter_Index(index - ROUND_1_END))
	}
	if index < ROUND_3_END {
		return round_constant_32_47(Round_Quarter_Index(index - ROUND_2_END))
	}
	return round_constant_48_63(Round_Quarter_Index(index - ROUND_3_END))
}

func round_constant_0_15(index Round_Quarter_Index) (constant Round_Constant) {
	defer func() { Round_Constant_Invariants(constant, "round_constant_0_15.constant") }()
	Round_Quarter_Index_Invariants(index, "round_constant_0_15.index")
	constants := [...]uint32{
		0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5,
		0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
		0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3,
		0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
	}
	return Round_Constant{constants[index]}
}

func round_constant_16_31(index Round_Quarter_Index) (constant Round_Constant) {
	defer func() { Round_Constant_Invariants(constant, "round_constant_16_31.constant") }()
	Round_Quarter_Index_Invariants(index, "round_constant_16_31.index")
	constants := [...]uint32{
		0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc,
		0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
		0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7,
		0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
	}
	return Round_Constant{constants[index]}
}

func round_constant_32_47(index Round_Quarter_Index) (constant Round_Constant) {
	defer func() { Round_Constant_Invariants(constant, "round_constant_32_47.constant") }()
	Round_Quarter_Index_Invariants(index, "round_constant_32_47.index")
	constants := [...]uint32{
		0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13,
		0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
		0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3,
		0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
	}
	return Round_Constant{constants[index]}
}

func round_constant_48_63(index Round_Quarter_Index) (constant Round_Constant) {
	defer func() { Round_Constant_Invariants(constant, "round_constant_48_63.constant") }()
	Round_Quarter_Index_Invariants(index, "round_constant_48_63.index")
	constants := [...]uint32{
		0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5,
		0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
		0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208,
		0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
	}
	return Round_Constant{constants[index]}
}

func digest_require(digest *Digest) {
	Digest_Invariants(*digest, "digest_require.digest")
	invariant.Always(
		digest.State[STATE_READY_INDEX] == STATE_READY_MARKER,
		"SHA-256 operations require Digest_Init.",
	)
	if digest.State[STATE_READY_INDEX] != STATE_READY_MARKER {
		panic("sha256: digest is not initialized")
	}
}

func digest_require_kind(digest *Digest, kind Kind) {
	Digest_Invariants(*digest, "digest_require_kind.digest")
	Kind_Invariants(kind, "digest_require_kind.kind")
	digest_require(digest)
	invariant.Always(digest.Kind == kind, "Digest output kind must match initialized kind.")
	if digest.Kind != kind {
		panic("sha256: digest kind does not match output")
	}
}
