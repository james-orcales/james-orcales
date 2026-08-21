// Package md5 computes RFC 1321 MD5 with bounded caller-owned state and output.
//
// MD5 is cryptographically broken. Use only where protocol compatibility requires it.
package md5

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
)

// STATE_LANE_COUNT stores one result as 32-bit words.
const STATE_LANE_COUNT = binary.UINT_32_SIZE

// DIGEST_BIT_COUNT derives the result width from its state words.
const DIGEST_BIT_COUNT = STATE_LANE_COUNT * bits.BIT_COUNT_32_MAXIMUM

// DIGEST_SIZE converts result width to bytes.
const DIGEST_SIZE = DIGEST_BIT_COUNT / binary.BITS_PER_BYTE

// STATE_READY_INDEX follows compression lanes inside caller storage.
const STATE_READY_INDEX = STATE_LANE_COUNT

// STATE_WORD_COUNT holds compression lanes and initialization marker.
const STATE_WORD_COUNT = STATE_READY_INDEX + binary.UINT_8_SIZE

// STATE_READY_MARKER separates initialized state from zero caller storage.
const STATE_READY_MARKER = bits.WORD_32_MAXIMUM

// MESSAGE_WORD_COUNT is 32-bit words in one compression block.
const MESSAGE_WORD_COUNT = bits.BIT_COUNT_16_MAXIMUM

// BLOCK_SIZE derives compression width from its message words.
const BLOCK_SIZE = MESSAGE_WORD_COUNT * binary.UINT_32_SIZE

// BLOCK_BIT_COUNT converts compression width to bits.
const BLOCK_BIT_COUNT = BLOCK_SIZE * binary.BITS_PER_BYTE

// MESSAGE_BIT_COUNT_SIZE is trailing encoded message-bit count width.
const MESSAGE_BIT_COUNT_SIZE = binary.UINT_64_SIZE

// PADDING_BOUNDARY leaves trailing length inside final block.
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

// OUTPUT_COUNT_REQUIRED is required MD5 output width.
const OUTPUT_COUNT_REQUIRED Output_Count = DIGEST_SIZE

// OUTPUT_STATUS_OK means complete digest reached caller storage.
const OUTPUT_STATUS_OK Output_Status = Output_Status(bits.WORD_8_MINIMUM)

// OUTPUT_STATUS_TOO_SMALL leaves short caller storage untouched.
const OUTPUT_STATUS_TOO_SMALL Output_Status = OUTPUT_STATUS_OK + binary.UINT_8_SIZE

// INITIAL_STATE_0 is RFC 1321 first chaining word.
const INITIAL_STATE_0 uint32 = 0x67452301

// INITIAL_STATE_1 is RFC 1321 second chaining word.
const INITIAL_STATE_1 uint32 = 0xefcdab89

// INITIAL_STATE_2 derives the RFC complement pair instead of duplicating its value.
const INITIAL_STATE_2 uint32 = ^INITIAL_STATE_0

// INITIAL_STATE_3 derives the RFC complement pair instead of duplicating its value.
const INITIAL_STATE_3 uint32 = ^INITIAL_STATE_1

// ROUND_QUARTER_COUNT is the RFC nonlinear-function count.
const ROUND_QUARTER_COUNT = binary.UINT_32_SIZE

// ROUND_COUNT derives total steps from one message traversal per quarter.
const ROUND_COUNT Round_Index = ROUND_QUARTER_COUNT * MESSAGE_WORD_COUNT

// ROUND_1_END is boundary after first nonlinear function.
const ROUND_1_END Round_Index = ROUND_COUNT / ROUND_QUARTER_COUNT

// ROUND_2_END is boundary after second nonlinear function.
const ROUND_2_END Round_Index = ROUND_1_END * binary.UINT_16_SIZE

// ROUND_3_END is boundary after third nonlinear function.
const ROUND_3_END Round_Index = ROUND_1_END * (binary.UINT_16_SIZE + binary.UINT_8_SIZE)

// ROUND_LANE_COUNT is rotation cycle width inside each quarter.
const ROUND_LANE_COUNT Round_Index = STATE_LANE_COUNT

// MESSAGE_WORD_MASK reduces formula-selected word to one 16-word block.
const MESSAGE_WORD_MASK Round_Index = MESSAGE_WORD_COUNT - binary.UINT_8_SIZE

// ROUND_INDEX_MINIMUM is first compression step.
const ROUND_INDEX_MINIMUM uint8 = bits.WORD_8_MINIMUM

// ROUND_INDEX_MAXIMUM is final compression step.
const ROUND_INDEX_MAXIMUM uint8 = uint8(ROUND_COUNT - binary.UINT_8_SIZE)

// ROUND_QUARTER_INDEX_MINIMUM is first step inside one 16-step quarter.
const ROUND_QUARTER_INDEX_MINIMUM uint8 = bits.WORD_8_MINIMUM

// ROUND_QUARTER_INDEX_MAXIMUM is final step inside one 16-step quarter.
const ROUND_QUARTER_INDEX_MAXIMUM uint8 = uint8(ROUND_1_END - binary.UINT_8_SIZE)

// ROUND_CONSTANT_WORD_COUNT stores one 32-bit word per additive constant.
const ROUND_CONSTANT_WORD_COUNT = binary.UINT_8_SIZE

// ROTATION_DISTANCE_COUNT stores one distance per compression step.
const ROTATION_DISTANCE_COUNT = ROUND_CONSTANT_WORD_COUNT

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
		"MD5 compression input contains complete blocks.",
	)
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

// Output_Count is required MD5 width on complete or short output.
type Output_Count uint8

// Output_Count_Invariants fixes required MD5 width.
func Output_Count_Invariants(value Output_Count, _ invariant.Namespace) {
	invariant.Always(
		uint8(value) == uint8(OUTPUT_COUNT_REQUIRED),
		"MD5 output count equals required digest width.",
	)
}

// Output_Status reports caller output capacity.
type Output_Status uint8

// Output_Status_Invariants covers complete and short output.
func Output_Status_Invariants(value Output_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(OUTPUT_STATUS_OK), uint8(OUTPUT_STATUS_TOO_SMALL)).
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

// State holds MD5 compression lanes and caller-storage identity.
type State [STATE_WORD_COUNT]uint32

// State_Invariants fixes compression state and identity storage width.
func State_Invariants(value State, _ invariant.Namespace) {
	invariant.Always(len(value) == STATE_WORD_COUNT, "MD5 state storage has fixed width.")
}

// Round_Index identifies one RFC 1321 compression step.
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

// Round_Constant stores one opaque RFC 1321 additive word.
type Round_Constant [ROUND_CONSTANT_WORD_COUNT]uint32

// Round_Constant_Invariants fixes one-word constant storage.
func Round_Constant_Invariants(value Round_Constant, _ invariant.Namespace) {
	invariant.Always(
		len(value) == ROUND_CONSTANT_WORD_COUNT,
		"MD5 round constant occupies one word.",
	)
}

// Rotation stores one opaque RFC 1321 rotation distance.
type Rotation [ROTATION_DISTANCE_COUNT]bits.Rotation

// Rotation_Invariants fixes one-distance storage.
func Rotation_Invariants(value Rotation, _ invariant.Namespace) {
	invariant.Always(
		len(value) == ROTATION_DISTANCE_COUNT,
		"MD5 round rotation occupies one distance.",
	)
}

// Size is MD5 digest byte width.
type Size int

// Size_Invariants fixes MD5 digest width.
func Size_Invariants(value Size, _ invariant.Namespace) {
	invariant.Always(int(value) == DIGEST_SIZE, "MD5 digest size equals result width.")
}

// Block_Size is MD5 compression-block byte width.
type Block_Size int

// Block_Size_Invariants fixes MD5 compression-block width.
func Block_Size_Invariants(value Block_Size, _ invariant.Namespace) {
	invariant.Always(int(value) == BLOCK_SIZE, "MD5 block size equals compression width.")
}

// Value is one MD5 digest in wire byte order.
type Value [DIGEST_SIZE]byte

// Value_Invariants fixes digest storage width.
func Value_Invariants(value Value, _ invariant.Namespace) {
	invariant.Always(len(value) == DIGEST_SIZE, "MD5 digest has derived width.")
}

// Digest is caller-owned streaming MD5 state.
type Digest struct {
	// State holds four compression lanes.
	State State
	// Buffer holds one incomplete block.
	Buffer [BLOCK_SIZE]byte
	// Buffer_Count identifies live Buffer prefix.
	Buffer_Count Buffer_Count
	// Message_Size counts accepted bytes for final length encoding.
	Message_Size Message_Size
}

// Digest_Invariants composes all scalar state and partial-block relation.
func Digest_Invariants(value Digest, namespace invariant.Namespace) {
	State_Invariants(value.State, namespace)
	Buffer_Count_Invariants(value.Buffer_Count, namespace)
	Message_Size_Invariants(value.Message_Size, namespace)
	invariant.Always(
		uint64(value.Buffer_Count) == uint64(value.Message_Size)%BLOCK_SIZE,
		"Partial MD5 block equals message remainder.",
	)
}

// Digest_Init establishes RFC 1321 initial state in caller storage.
func Digest_Init(digest *Digest) {
	Digest_Invariants(*digest, "Digest_Init.digest.input")
	digest.State = State{
		INITIAL_STATE_0, INITIAL_STATE_1, INITIAL_STATE_2, INITIAL_STATE_3,
		STATE_READY_MARKER,
	}
	digest.Buffer = [BLOCK_SIZE]byte{}
	digest.Buffer_Count = BUFFER_COUNT_MINIMUM
	digest.Message_Size = Message_Size(MESSAGE_SIZE_MINIMUM)
	State_Invariants(digest.State, "Digest_Init.digest.state.output")
	invariant.Always(
		digest.Buffer_Count == BUFFER_COUNT_MINIMUM,
		"Fresh MD5 state has no buffered bytes.",
	)
	invariant.Always(
		uint64(digest.Message_Size) == MESSAGE_SIZE_MINIMUM,
		"Fresh MD5 state has no accepted bytes.",
	)
}

// Digest_Reset discards message while retaining caller storage.
func Digest_Reset(digest *Digest) {
	Digest_Invariants(*digest, "Digest_Reset.digest.input")
	digest_require(digest)
	Digest_Init(digest)
}

// Digest_Write consumes one bounded source completely.
func Digest_Write(digest *Digest, source Source) (count Count) {
	defer func() { Count_Invariants(count, "Digest_Write.count") }()
	Digest_Invariants(*digest, "Digest_Write.digest.input")
	Source_Invariants(source, "Digest_Write.source")
	digest_require(digest)
	if len(source) > SOURCE_SIZE_MAXIMUM {
		panic("md5: source exceeds bound")
	}
	if uint64(digest.Message_Size) > MESSAGE_SIZE_MAXIMUM-uint64(len(source)) {
		panic("md5: message exceeds bound")
	}
	digest_write(digest, source)
	Digest_Invariants(*digest, "Digest_Write.digest.output")
	return Count(len(source))
}

// Digest_Sum observes current state without consuming it.
func Digest_Sum(digest *Digest) (value Value) {
	defer func() { Value_Invariants(value, "Digest_Sum.value") }()
	Digest_Invariants(*digest, "Digest_Sum.digest")
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
		binary.Word_64(message_bits), binary.LITTLE_ENDIAN,
	)
	block(&copy_digest.State, Blocks(final_blocks[:final_size]))
	for index := range STATE_LANE_COUNT {
		start := index * binary.UINT_32_SIZE
		binary.Put_Uint_32(
			binary.Bytes(value[start:start+binary.UINT_32_SIZE]),
			binary.Word_32(copy_digest.State[index]), binary.LITTLE_ENDIAN,
		)
	}
	return value
}

// Digest_Sum_Into writes complete digest or leaves short storage untouched.
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
		panic("md5: destination exceeds bound")
	}
	if len(destination) < DIGEST_SIZE {
		return OUTPUT_COUNT_REQUIRED, OUTPUT_STATUS_TOO_SMALL
	}
	value := Digest_Sum(digest)
	copy(destination[:DIGEST_SIZE], value[:])
	return OUTPUT_COUNT_REQUIRED, OUTPUT_STATUS_OK
}

// Digest_Clone_Into copies live state without aliasing caller storage.
func Digest_Clone_Into(destination *Digest, source *Digest) {
	Digest_Invariants(*destination, "Digest_Clone_Into.destination.input")
	Digest_Invariants(*source, "Digest_Clone_Into.source")
	digest_require(source)
	*destination = *source
	Digest_Invariants(*destination, "Digest_Clone_Into.destination.output")
}

// Digest_Size reports MD5 output width.
func Digest_Size(digest *Digest) (size Size) {
	defer func() { Size_Invariants(size, "Digest_Size.size") }()
	Digest_Invariants(*digest, "Digest_Size.digest")
	digest_require(digest)
	return DIGEST_SIZE
}

// Digest_Block_Size reports MD5 compression block width.
func Digest_Block_Size(digest *Digest) (size Block_Size) {
	defer func() { Block_Size_Invariants(size, "Digest_Block_Size.size") }()
	Digest_Invariants(*digest, "Digest_Block_Size.digest")
	digest_require(digest)
	return BLOCK_SIZE
}

// Checksum computes one bounded source without retained state.
func Checksum(source Source) (value Value) {
	defer func() { Value_Invariants(value, "Checksum.value") }()
	Source_Invariants(source, "Checksum.source")
	if len(source) > SOURCE_SIZE_MAXIMUM {
		panic("md5: source exceeds bound")
	}
	var digest Digest
	Digest_Init(&digest)
	Digest_Write(&digest, source)
	return Digest_Sum(&digest)
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

func block(state *State, source Blocks) {
	State_Invariants(*state, "block.state.input")
	Blocks_Invariants(source, "block.source")
	const STATE_A_INDEX = bytes.SLICE_SIZE_MINIMUM
	const STATE_B_INDEX = STATE_A_INDEX + binary.UINT_8_SIZE
	const STATE_C_INDEX = STATE_B_INDEX + binary.UINT_8_SIZE
	const STATE_D_INDEX = STATE_C_INDEX + binary.UINT_8_SIZE
	const WORD_BYTE_A_INDEX = bytes.SLICE_SIZE_MINIMUM
	const WORD_BYTE_B_INDEX = WORD_BYTE_A_INDEX + binary.UINT_8_SIZE
	const WORD_BYTE_C_INDEX = WORD_BYTE_B_INDEX + binary.UINT_8_SIZE
	const WORD_BYTE_D_INDEX = WORD_BYTE_C_INDEX + binary.UINT_8_SIZE
	const ROUND_2_MESSAGE_FACTOR Round_Index = STATE_LANE_COUNT + binary.UINT_8_SIZE
	const ROUND_2_MESSAGE_OFFSET Round_Index = binary.UINT_8_SIZE
	const ROUND_3_MESSAGE_FACTOR Round_Index = STATE_LANE_COUNT - binary.UINT_8_SIZE
	const ROUND_3_MESSAGE_OFFSET Round_Index = ROUND_2_MESSAGE_FACTOR
	const ROUND_4_MESSAGE_FACTOR Round_Index = STATE_LANE_COUNT*binary.UINT_16_SIZE -
		binary.UINT_8_SIZE
	for len(source) >= BLOCK_SIZE {
		var message [MESSAGE_WORD_COUNT]uint32
		for index := range message {
			start := index * binary.UINT_32_SIZE
			message[index] = uint32(source[start+WORD_BYTE_A_INDEX]) |
				uint32(source[start+WORD_BYTE_B_INDEX])<<binary.BITS_PER_BYTE |
				uint32(source[start+WORD_BYTE_C_INDEX])<<
					(binary.BITS_PER_BYTE*WORD_BYTE_C_INDEX) |
				uint32(source[start+WORD_BYTE_D_INDEX])<<
					(binary.BITS_PER_BYTE*WORD_BYTE_D_INDEX)
		}
		a := state[STATE_A_INDEX]
		b := state[STATE_B_INDEX]
		c := state[STATE_C_INDEX]
		d := state[STATE_D_INDEX]
		for index := Round_Index(ROUND_INDEX_MINIMUM); index < ROUND_COUNT; index++ {
			var combined uint32
			var message_index Round_Index
			if index < ROUND_1_END {
				combined = b&c | ^b&d
				message_index = index
			} else if index < ROUND_2_END {
				combined = d&b | ^d&c
				message_position := index*ROUND_2_MESSAGE_FACTOR +
					ROUND_2_MESSAGE_OFFSET
				message_index = message_position & MESSAGE_WORD_MASK
			} else if index < ROUND_3_END {
				combined = b ^ c ^ d
				message_position := index*ROUND_3_MESSAGE_FACTOR +
					ROUND_3_MESSAGE_OFFSET
				message_index = message_position & MESSAGE_WORD_MASK
			} else {
				combined = c ^ (b | ^d)
				message_index = (index * ROUND_4_MESSAGE_FACTOR) & MESSAGE_WORD_MASK
			}
			constant := round_constant(index)[bytes.SLICE_SIZE_MINIMUM]
			round_value := a + combined + constant
			round_value += message[message_index]
			rotation := round_rotation(index)[bytes.SLICE_SIZE_MINIMUM]
			rotated := bits.Rotate_Left_32(bits.Word_32(round_value), rotation)
			a, d, c, b = d, c, b, b+uint32(rotated)
		}
		state[STATE_A_INDEX] += a
		state[STATE_B_INDEX] += b
		state[STATE_C_INDEX] += c
		state[STATE_D_INDEX] += d
		source = source[BLOCK_SIZE:]
	}
	State_Invariants(*state, "block.state.output")
}

func round_rotation(index Round_Index) (rotation Rotation) {
	defer func() { Rotation_Invariants(rotation, "round_rotation.rotation") }()
	Round_Index_Invariants(index, "round_rotation.index")
	if index < ROUND_1_END {
		return round_1_rotation(Round_Quarter_Index(index))
	}
	if index < ROUND_2_END {
		return round_2_rotation(Round_Quarter_Index(index - ROUND_1_END))
	}
	if index < ROUND_3_END {
		return round_3_rotation(Round_Quarter_Index(index - ROUND_2_END))
	}
	return round_4_rotation(Round_Quarter_Index(index - ROUND_3_END))
}

func round_1_rotation(index Round_Quarter_Index) (rotation Rotation) {
	defer func() { Rotation_Invariants(rotation, "round_1_rotation.rotation") }()
	Round_Quarter_Index_Invariants(index, "round_1_rotation.index")
	const STATE_A_INDEX = bytes.SLICE_SIZE_MINIMUM
	const STATE_B_INDEX = STATE_A_INDEX + binary.UINT_8_SIZE
	const STATE_C_INDEX = STATE_B_INDEX + binary.UINT_8_SIZE
	const ROTATION_STEP bits.Rotation = bits.Rotation(STATE_LANE_COUNT + binary.UINT_8_SIZE)
	const ROTATION_A bits.Rotation = bits.Rotation(
		bits.BIT_COUNT_8_MAXIMUM - binary.UINT_8_SIZE,
	)
	const ROTATION_B bits.Rotation = ROTATION_A + ROTATION_STEP
	const ROTATION_C bits.Rotation = ROTATION_B + ROTATION_STEP
	const ROTATION_D bits.Rotation = ROTATION_C + ROTATION_STEP
	lane := index % Round_Quarter_Index(ROUND_LANE_COUNT)
	switch lane {
	case STATE_A_INDEX:
		return Rotation{ROTATION_A}
	case STATE_B_INDEX:
		return Rotation{ROTATION_B}
	case STATE_C_INDEX:
		return Rotation{ROTATION_C}
	default:
		return Rotation{ROTATION_D}
	}
}

func round_2_rotation(index Round_Quarter_Index) (rotation Rotation) {
	defer func() { Rotation_Invariants(rotation, "round_2_rotation.rotation") }()
	Round_Quarter_Index_Invariants(index, "round_2_rotation.index")
	const STATE_A_INDEX = bytes.SLICE_SIZE_MINIMUM
	const STATE_B_INDEX = STATE_A_INDEX + binary.UINT_8_SIZE
	const STATE_C_INDEX = STATE_B_INDEX + binary.UINT_8_SIZE
	const ROTATION_A bits.Rotation = bits.Rotation(STATE_LANE_COUNT + binary.UINT_8_SIZE)
	const ROTATION_B bits.Rotation = ROTATION_A + bits.Rotation(STATE_LANE_COUNT)
	const ROTATION_C bits.Rotation = ROTATION_B + bits.Rotation(
		STATE_LANE_COUNT+binary.UINT_8_SIZE,
	)
	const ROTATION_D bits.Rotation = ROTATION_C + bits.Rotation(
		STATE_LANE_COUNT+binary.UINT_16_SIZE,
	)
	lane := index % Round_Quarter_Index(ROUND_LANE_COUNT)
	switch lane {
	case STATE_A_INDEX:
		return Rotation{ROTATION_A}
	case STATE_B_INDEX:
		return Rotation{ROTATION_B}
	case STATE_C_INDEX:
		return Rotation{ROTATION_C}
	default:
		return Rotation{ROTATION_D}
	}
}

func round_3_rotation(index Round_Quarter_Index) (rotation Rotation) {
	defer func() { Rotation_Invariants(rotation, "round_3_rotation.rotation") }()
	Round_Quarter_Index_Invariants(index, "round_3_rotation.index")
	const STATE_A_INDEX = bytes.SLICE_SIZE_MINIMUM
	const STATE_B_INDEX = STATE_A_INDEX + binary.UINT_8_SIZE
	const STATE_C_INDEX = STATE_B_INDEX + binary.UINT_8_SIZE
	const ROTATION_STEP_WIDE bits.Rotation = bits.Rotation(
		STATE_LANE_COUNT*binary.UINT_16_SIZE - binary.UINT_8_SIZE,
	)
	const ROTATION_STEP_MIDDLE bits.Rotation = bits.Rotation(
		STATE_LANE_COUNT + binary.UINT_8_SIZE,
	)
	const ROTATION_A bits.Rotation = bits.Rotation(STATE_LANE_COUNT)
	const ROTATION_B bits.Rotation = ROTATION_A + ROTATION_STEP_WIDE
	const ROTATION_C bits.Rotation = ROTATION_B + ROTATION_STEP_MIDDLE
	const ROTATION_D bits.Rotation = ROTATION_C + ROTATION_STEP_WIDE
	lane := index % Round_Quarter_Index(ROUND_LANE_COUNT)
	switch lane {
	case STATE_A_INDEX:
		return Rotation{ROTATION_A}
	case STATE_B_INDEX:
		return Rotation{ROTATION_B}
	case STATE_C_INDEX:
		return Rotation{ROTATION_C}
	default:
		return Rotation{ROTATION_D}
	}
}

func round_4_rotation(index Round_Quarter_Index) (rotation Rotation) {
	defer func() { Rotation_Invariants(rotation, "round_4_rotation.rotation") }()
	Round_Quarter_Index_Invariants(index, "round_4_rotation.index")
	const STATE_A_INDEX = bytes.SLICE_SIZE_MINIMUM
	const STATE_B_INDEX = STATE_A_INDEX + binary.UINT_8_SIZE
	const STATE_C_INDEX = STATE_B_INDEX + binary.UINT_8_SIZE
	const ROTATION_A bits.Rotation = bits.Rotation(
		bits.BIT_COUNT_8_MAXIMUM - binary.UINT_16_SIZE,
	)
	const ROTATION_B bits.Rotation = ROTATION_A + bits.Rotation(STATE_LANE_COUNT)
	const ROTATION_C bits.Rotation = ROTATION_B + bits.Rotation(
		STATE_LANE_COUNT+binary.UINT_8_SIZE,
	)
	const ROTATION_D bits.Rotation = ROTATION_C + bits.Rotation(
		STATE_LANE_COUNT+binary.UINT_16_SIZE,
	)
	lane := index % Round_Quarter_Index(ROUND_LANE_COUNT)
	switch lane {
	case STATE_A_INDEX:
		return Rotation{ROTATION_A}
	case STATE_B_INDEX:
		return Rotation{ROTATION_B}
	case STATE_C_INDEX:
		return Rotation{ROTATION_C}
	default:
		return Rotation{ROTATION_D}
	}
}

// RFC derives K[i] as floor(abs(sin(i+1))*2^32); exact words avoid floating-point drift.
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
		0xd76aa478, 0xe8c7b756, 0x242070db, 0xc1bdceee,
		0xf57c0faf, 0x4787c62a, 0xa8304613, 0xfd469501,
		0x698098d8, 0x8b44f7af, 0xffff5bb1, 0x895cd7be,
		0x6b901122, 0xfd987193, 0xa679438e, 0x49b40821,
	}
	return Round_Constant{constants[index]}
}

func round_constant_16_31(index Round_Quarter_Index) (constant Round_Constant) {
	defer func() { Round_Constant_Invariants(constant, "round_constant_16_31.constant") }()
	Round_Quarter_Index_Invariants(index, "round_constant_16_31.index")
	constants := [...]uint32{
		0xf61e2562, 0xc040b340, 0x265e5a51, 0xe9b6c7aa,
		0xd62f105d, 0x02441453, 0xd8a1e681, 0xe7d3fbc8,
		0x21e1cde6, 0xc33707d6, 0xf4d50d87, 0x455a14ed,
		0xa9e3e905, 0xfcefa3f8, 0x676f02d9, 0x8d2a4c8a,
	}
	return Round_Constant{constants[index]}
}

func round_constant_32_47(index Round_Quarter_Index) (constant Round_Constant) {
	defer func() { Round_Constant_Invariants(constant, "round_constant_32_47.constant") }()
	Round_Quarter_Index_Invariants(index, "round_constant_32_47.index")
	constants := [...]uint32{
		0xfffa3942, 0x8771f681, 0x6d9d6122, 0xfde5380c,
		0xa4beea44, 0x4bdecfa9, 0xf6bb4b60, 0xbebfbc70,
		0x289b7ec6, 0xeaa127fa, 0xd4ef3085, 0x04881d05,
		0xd9d4d039, 0xe6db99e5, 0x1fa27cf8, 0xc4ac5665,
	}
	return Round_Constant{constants[index]}
}

func round_constant_48_63(index Round_Quarter_Index) (constant Round_Constant) {
	defer func() { Round_Constant_Invariants(constant, "round_constant_48_63.constant") }()
	Round_Quarter_Index_Invariants(index, "round_constant_48_63.index")
	constants := [...]uint32{
		0xf4292244, 0x432aff97, 0xab9423a7, 0xfc93a039,
		0x655b59c3, 0x8f0ccc92, 0xffeff47d, 0x85845dd1,
		0x6fa87e4f, 0xfe2ce6e0, 0xa3014314, 0x4e0811a1,
		0xf7537e82, 0xbd3af235, 0x2ad7d2bb, 0xeb86d391,
	}
	return Round_Constant{constants[index]}
}

func digest_require(digest *Digest) {
	Digest_Invariants(*digest, "digest_require.digest")
	invariant.Always(
		digest.State[STATE_READY_INDEX] == STATE_READY_MARKER,
		"MD5 operations require Digest_Init.",
	)
	if digest.State[STATE_READY_INDEX] != STATE_READY_MARKER {
		panic("md5: digest is not initialized")
	}
}
