// Package sha256 computes FIPS 180-4 SHA-224 and SHA-256 with caller-owned state.
package sha256

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
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

// STATE_WORD_COUNT counts compression lanes and lifecycle identity.
const STATE_WORD_COUNT = STATE_LANE_COUNT + binary.UINT_8_SIZE

// STATE_READY_MARKER separates initialized state from zero caller storage.
const STATE_READY_MARKER = true

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

// BUFFER_SOURCE_SIZE_MINIMUM excludes empty writes handled before packing.
const BUFFER_SOURCE_SIZE_MINIMUM = binary.UINT_8_SIZE

// BUFFER_SOURCE_SIZE_MAXIMUM prevents packing one complete block as partial state.
const BUFFER_SOURCE_SIZE_MAXIMUM = BUFFER_COUNT_MAXIMUM

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

// ROUND_INDEX_MINIMUM is first compression step.
const ROUND_INDEX_MINIMUM uint8 = bits.WORD_8_MINIMUM

// ROUND_INDEX_MAXIMUM is final compression step.
const ROUND_INDEX_MAXIMUM uint8 = uint8(ROUND_COUNT - binary.UINT_8_SIZE)

// ROUND_CONSTANT_MINIMUM is least FIPS table word.
const ROUND_CONSTANT_MINIMUM uint32 = 0x06ca6351

// ROUND_CONSTANT_MAXIMUM is greatest FIPS table word.
const ROUND_CONSTANT_MAXIMUM uint32 = 0xf40e3585

// Source is one bounded input chunk.
type Source []byte

// Source_Invariants binds one call to repository byte boundary.
func Source_Invariants(value Source, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Blocks is one or more complete bounded compression blocks.
type Blocks []byte

// Blocks_Invariants excludes empty and partial compression input.
func Blocks_Invariants(value Blocks, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), BLOCK_SIZE, SOURCE_SIZE_MAXIMUM).
		Ensure()
	aver.Always(
		len(value)%BLOCK_SIZE == SOURCE_SIZE_MINIMUM,
		"SHA-256 compression input contains complete blocks.",
	)
}

// Block is one complete compression block.
type Block []byte

// Block_Invariants fixes schedule input to one block.
func Block_Invariants(value Block, _ aver.Namespace) {
	aver.Always(len(value) == BLOCK_SIZE, "SHA-256 schedule input equals one block.")
}

// Destination is bounded caller-owned output storage.
type Destination []byte

// Destination_Invariants binds output to repository byte boundary.
func Destination_Invariants(value Destination, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), DESTINATION_SIZE_MINIMUM, DESTINATION_SIZE_MAXIMUM).
		Ensure()
}

// Count is bytes consumed by one bounded write.
type Count int

// Count_Invariants covers complete source count.
func Count_Invariants(value Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Output_Count is selected digest width on complete or short output.
type Output_Count uint8

// Output_Count_Invariants admits both selected digest widths.
func Output_Count_Invariants(value Output_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(OUTPUT_COUNT_224_REQUIRED),
			uint8(OUTPUT_COUNT_256_REQUIRED),
		).
		Ensure()
}

// Output_Status reports caller output capacity.
type Output_Status uint8

// Output_Status_Invariants covers complete and short output.
func Output_Status_Invariants(value Output_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(OUTPUT_STATUS_OK), uint8(OUTPUT_STATUS_TOO_SMALL)).
		Ensure()
}

// Kind selects SHA-224 or SHA-256.
type Kind uint8

// Kind_Invariants admits two FIPS 180-4 functions.
func Kind_Invariants(value Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(KIND_SHA_224), uint8(KIND_SHA_256)).
		Ensure()
}

// Buffer_Count is live bytes in partial block storage.
type Buffer_Count uint8

// Buffer_Count_Invariants excludes complete blocks.
func Buffer_Count_Invariants(value Buffer_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), BUFFER_COUNT_MINIMUM, BUFFER_COUNT_MAXIMUM).
		Ensure()
}

// Message_Size is bytes accepted by current digest.
type Message_Size uint64

// Message_Size_Invariants preserves exact final bit-count encoding.
func Message_Size_Invariants(value Message_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), MESSAGE_SIZE_MINIMUM, MESSAGE_SIZE_MAXIMUM).
		Ensure()
}

// State_Lane_0 exposes first compression word.
type State_Lane_0 uint32

// State_Lane_0_Invariants preserves complete word domain.
func State_Lane_0_Invariants(value State_Lane_0, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, bits.WORD_32_MAXIMUM).
		Ensure()
}

// State_Lane_1 exposes second compression word.
type State_Lane_1 uint32

// State_Lane_1_Invariants preserves complete word domain.
func State_Lane_1_Invariants(value State_Lane_1, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, bits.WORD_32_MAXIMUM).
		Ensure()
}

// State_Lane_2 exposes third compression word.
type State_Lane_2 uint32

// State_Lane_2_Invariants preserves complete word domain.
func State_Lane_2_Invariants(value State_Lane_2, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, bits.WORD_32_MAXIMUM).
		Ensure()
}

// State_Lane_3 exposes fourth compression word.
type State_Lane_3 uint32

// State_Lane_3_Invariants preserves complete word domain.
func State_Lane_3_Invariants(value State_Lane_3, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, bits.WORD_32_MAXIMUM).
		Ensure()
}

// State_Lane_4 exposes fifth compression word.
type State_Lane_4 uint32

// State_Lane_4_Invariants preserves complete word domain.
func State_Lane_4_Invariants(value State_Lane_4, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, bits.WORD_32_MAXIMUM).
		Ensure()
}

// State_Lane_5 exposes sixth compression word.
type State_Lane_5 uint32

// State_Lane_5_Invariants preserves complete word domain.
func State_Lane_5_Invariants(value State_Lane_5, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, bits.WORD_32_MAXIMUM).
		Ensure()
}

// State_Lane_6 exposes seventh compression word.
type State_Lane_6 uint32

// State_Lane_6_Invariants preserves complete word domain.
func State_Lane_6_Invariants(value State_Lane_6, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, bits.WORD_32_MAXIMUM).
		Ensure()
}

// State_Lane_7 exposes eighth compression word.
type State_Lane_7 uint32

// State_Lane_7_Invariants preserves complete word domain.
func State_Lane_7_Invariants(value State_Lane_7, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, bits.WORD_32_MAXIMUM).
		Ensure()
}

// State keeps compression words independently visible.
type State struct {
	// Lane_0 avoids array-hidden state.
	Lane_0 State_Lane_0
	// Lane_1 avoids array-hidden state.
	Lane_1 State_Lane_1
	// Lane_2 avoids array-hidden state.
	Lane_2 State_Lane_2
	// Lane_3 avoids array-hidden state.
	Lane_3 State_Lane_3
	// Lane_4 avoids array-hidden state.
	Lane_4 State_Lane_4
	// Lane_5 avoids array-hidden state.
	Lane_5 State_Lane_5
	// Lane_6 avoids array-hidden state.
	Lane_6 State_Lane_6
	// Lane_7 avoids array-hidden state.
	Lane_7 State_Lane_7
}

// State_Invariants exposes every compression word.
func State_Invariants(value State, namespace aver.Namespace) {
	State_Lane_0_Invariants(value.Lane_0, namespace)
	State_Lane_1_Invariants(value.Lane_1, namespace)
	State_Lane_2_Invariants(value.Lane_2, namespace)
	State_Lane_3_Invariants(value.Lane_3, namespace)
	State_Lane_4_Invariants(value.Lane_4, namespace)
	State_Lane_5_Invariants(value.Lane_5, namespace)
	State_Lane_6_Invariants(value.Lane_6, namespace)
	State_Lane_7_Invariants(value.Lane_7, namespace)
}

// State_Destination names mutable compression state.
type State_Destination *State

// State_Destination_Invariants composes state when present.
func State_Destination_Invariants(value State_Destination, namespace aver.Namespace) {
	if value == nil {
		return
	}
	State_Invariants(*value, namespace)
}

// Ready separates initialized state from zero caller storage.
type Ready bool

// Ready_Invariants covers both lifecycle states.
func Ready_Invariants(value Ready, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "SHA-256 state is initialized.").
		Ensure()
}

// Buffer_Source excludes complete blocks handled by compression loop.
type Buffer_Source []byte

// Buffer_Source_Invariants keeps partial input inside one block.
func Buffer_Source_Invariants(value Buffer_Source, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), BUFFER_SOURCE_SIZE_MINIMUM, BUFFER_SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Block_Destination names exact scratch used during scalar unpacking.
type Block_Destination []byte

// Block_Destination_Invariants prevents partial serialization.
func Block_Destination_Invariants(value Block_Destination, _ aver.Namespace) {
	aver.Always(len(value) == BLOCK_SIZE, "SHA-256 scratch has one complete block.")
}

// Buffer_Lane_1 exposes first packed word.
type Buffer_Lane_1 uint64

// Buffer_Lane_1_Invariants preserves complete word domain.
func Buffer_Lane_1_Invariants(value Buffer_Lane_1, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_2 exposes second packed word.
type Buffer_Lane_2 uint64

// Buffer_Lane_2_Invariants preserves complete word domain.
func Buffer_Lane_2_Invariants(value Buffer_Lane_2, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_3 exposes third packed word.
type Buffer_Lane_3 uint64

// Buffer_Lane_3_Invariants preserves complete word domain.
func Buffer_Lane_3_Invariants(value Buffer_Lane_3, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_4 exposes fourth packed word.
type Buffer_Lane_4 uint64

// Buffer_Lane_4_Invariants preserves complete word domain.
func Buffer_Lane_4_Invariants(value Buffer_Lane_4, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_5 exposes fifth packed word.
type Buffer_Lane_5 uint64

// Buffer_Lane_5_Invariants preserves complete word domain.
func Buffer_Lane_5_Invariants(value Buffer_Lane_5, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_6 exposes sixth packed word.
type Buffer_Lane_6 uint64

// Buffer_Lane_6_Invariants preserves complete word domain.
func Buffer_Lane_6_Invariants(value Buffer_Lane_6, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_7 exposes seventh packed word.
type Buffer_Lane_7 uint64

// Buffer_Lane_7_Invariants preserves complete word domain.
func Buffer_Lane_7_Invariants(value Buffer_Lane_7, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_8 exposes eighth packed word.
type Buffer_Lane_8 uint64

// Buffer_Lane_8_Invariants preserves complete word domain.
func Buffer_Lane_8_Invariants(value Buffer_Lane_8, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer packs one incomplete block without fixed-array state.
type Buffer struct {
	// Lane_1 avoids array-hidden state.
	Lane_1 Buffer_Lane_1
	// Lane_2 avoids array-hidden state.
	Lane_2 Buffer_Lane_2
	// Lane_3 avoids array-hidden state.
	Lane_3 Buffer_Lane_3
	// Lane_4 avoids array-hidden state.
	Lane_4 Buffer_Lane_4
	// Lane_5 avoids array-hidden state.
	Lane_5 Buffer_Lane_5
	// Lane_6 avoids array-hidden state.
	Lane_6 Buffer_Lane_6
	// Lane_7 avoids array-hidden state.
	Lane_7 Buffer_Lane_7
	// Lane_8 avoids array-hidden state.
	Lane_8 Buffer_Lane_8
}

// Buffer_Invariants exposes each packed lane separately.
func Buffer_Invariants(value Buffer, namespace aver.Namespace) {
	Buffer_Lane_1_Invariants(value.Lane_1, namespace)
	Buffer_Lane_2_Invariants(value.Lane_2, namespace)
	Buffer_Lane_3_Invariants(value.Lane_3, namespace)
	Buffer_Lane_4_Invariants(value.Lane_4, namespace)
	Buffer_Lane_5_Invariants(value.Lane_5, namespace)
	Buffer_Lane_6_Invariants(value.Lane_6, namespace)
	Buffer_Lane_7_Invariants(value.Lane_7, namespace)
	Buffer_Lane_8_Invariants(value.Lane_8, namespace)
}

// Buffer_Destination names mutable packed storage.
type Buffer_Destination *Buffer

// Buffer_Destination_Invariants composes packed storage when present.
func Buffer_Destination_Invariants(value Buffer_Destination, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Buffer_Invariants(*value, namespace)
}

// Round_Index identifies one compression step.
type Round_Index uint8

// Round_Index_Invariants covers every compression step.
func Round_Index_Invariants(value Round_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), ROUND_INDEX_MINIMUM, ROUND_INDEX_MAXIMUM).
		Ensure()
}

// Round_Constant stores one FIPS 180-4 additive word.
type Round_Constant uint32

// Round_Constant_Invariants spans actual constant-table bounds.
func Round_Constant_Invariants(value Round_Constant, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), ROUND_CONSTANT_MINIMUM, ROUND_CONSTANT_MAXIMUM).
		Ensure()
}

// Schedule holds expanded words for one compression block.
type Schedule []uint32

// Schedule_Invariants fixes one expanded block width.
func Schedule_Invariants(value Schedule, _ aver.Namespace) {
	aver.Always(len(value) == int(ROUND_COUNT), "SHA-256 schedule has derived width.")
}

// Size is one supported digest width.
type Size int

// Size_Invariants admits SHA-224 and SHA-256 widths.
func Size_Invariants(value Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(int(value), DIGEST_224_SIZE, DIGEST_256_SIZE).
		Ensure()
}

// Block_Size is shared SHA-224 and SHA-256 compression-block width.
type Block_Size int

// Block_Size_Invariants fixes shared compression-block width.
func Block_Size_Invariants(value Block_Size, _ aver.Namespace) {
	aver.Always(
		int(value) == BLOCK_SIZE,
		"SHA-256 block size equals compression width.",
	)
}

// Storage holds mutable SHA-224 or SHA-256 state.
type Storage struct {
	// State holds eight compression lanes.
	State State
	// Buffer holds one incomplete block.
	Buffer Buffer
	// Buffer_Count identifies live Buffer prefix.
	Buffer_Count Buffer_Count
	// Message_Size counts accepted bytes for final bit-count encoding.
	Message_Size Message_Size
	// Kind retains selected function across Reset.
	Kind Kind
}

// Storage_Invariants composes selected function and partial-block relation.
func Storage_Invariants(value Storage, namespace aver.Namespace) {
	State_Invariants(value.State, namespace)
	Buffer_Invariants(value.Buffer, namespace)
	Buffer_Count_Invariants(value.Buffer_Count, namespace)
	Message_Size_Invariants(value.Message_Size, namespace)
	Kind_Invariants(value.Kind, namespace)
	aver.Always(
		uint64(value.Buffer_Count) == uint64(value.Message_Size)%BLOCK_SIZE,
		"Partial SHA-256 block equals message remainder.",
	)
}

// Storage_Destination names mutable state after lifecycle validation.
type Storage_Destination *Storage

// Storage_Destination_Invariants composes storage when present.
func Storage_Destination_Invariants(value Storage_Destination, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Storage_Invariants(*value, namespace)
}

// Digest is caller-owned SHA-224 or SHA-256 state.
type Digest struct {
	// Storage remains embedded for direct field access.
	Storage
	// Ready rejects zero caller storage.
	Ready Ready
}

// Digest_Invariants composes state and lifecycle identity.
func Digest_Invariants(value Digest, namespace aver.Namespace) {
	Storage_Invariants(value.Storage, namespace)
	Ready_Invariants(value.Ready, namespace)
}

// Digest_Handle names mutable streaming state.
type Digest_Handle *Digest

// Digest_Handle_Invariants composes state when present.
func Digest_Handle_Invariants(value Digest_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Digest_Invariants(*value, namespace)
}

// Scalar packing preserves copy-safe digest ownership.
func buffer_write(
	buffer Buffer_Destination, position Buffer_Count, source Buffer_Source,
) {
	Buffer_Destination_Invariants(buffer, "buffer_write.buffer")
	Buffer_Count_Invariants(position, "buffer_write.position")
	Buffer_Source_Invariants(source, "buffer_write.source")
	aver.Always(
		int(position)+len(source) <= BLOCK_SIZE,
		"SHA-256 partial bytes fit one compression block.",
	)
	for _, item := range source {
		shift := uint(position%binary.UINT_64_SIZE) * binary.BITS_PER_BYTE
		mask := uint64(bits.WORD_8_MAXIMUM) << shift
		word := uint64(item) << shift
		switch position / binary.UINT_64_SIZE {
		case 0:
			buffer.Lane_1 = buffer.Lane_1&^Buffer_Lane_1(mask) | Buffer_Lane_1(word)
		case 1:
			buffer.Lane_2 = buffer.Lane_2&^Buffer_Lane_2(mask) | Buffer_Lane_2(word)
		case 2:
			buffer.Lane_3 = buffer.Lane_3&^Buffer_Lane_3(mask) | Buffer_Lane_3(word)
		case 3:
			buffer.Lane_4 = buffer.Lane_4&^Buffer_Lane_4(mask) | Buffer_Lane_4(word)
		case 4:
			buffer.Lane_5 = buffer.Lane_5&^Buffer_Lane_5(mask) | Buffer_Lane_5(word)
		case 5:
			buffer.Lane_6 = buffer.Lane_6&^Buffer_Lane_6(mask) | Buffer_Lane_6(word)
		case 6:
			buffer.Lane_7 = buffer.Lane_7&^Buffer_Lane_7(mask) | Buffer_Lane_7(word)
		case 7:
			buffer.Lane_8 = buffer.Lane_8&^Buffer_Lane_8(mask) | Buffer_Lane_8(word)
		}
		position++
	}
}

// Unpacking only into bounded scratch preserves zero-allocation ownership.
func buffer_copy(buffer Buffer, destination Block_Destination) {
	Buffer_Invariants(buffer, "buffer_copy.buffer")
	Block_Destination_Invariants(destination, "buffer_copy.destination")
	for position := range BLOCK_SIZE {
		var lane uint64
		switch position / binary.UINT_64_SIZE {
		case 0:
			lane = uint64(buffer.Lane_1)
		case 1:
			lane = uint64(buffer.Lane_2)
		case 2:
			lane = uint64(buffer.Lane_3)
		case 3:
			lane = uint64(buffer.Lane_4)
		case 4:
			lane = uint64(buffer.Lane_5)
		case 5:
			lane = uint64(buffer.Lane_6)
		case 6:
			lane = uint64(buffer.Lane_7)
		case 7:
			lane = uint64(buffer.Lane_8)
		}
		shift := uint(position%binary.UINT_64_SIZE) * binary.BITS_PER_BYTE
		destination[position] = byte(lane >> shift)
	}
}

// Digest_Init establishes selected FIPS 180-4 initial state.
func Digest_Init(digest Digest_Handle, kind Kind) {
	Digest_Handle_Invariants(digest, "Digest_Init.digest.input")
	Kind_Invariants(kind, "Digest_Init.kind")
	digest.Kind = kind
	digest_reset(Storage_Destination(&digest.Storage))
	digest.Ready = STATE_READY_MARKER
	aver.Always(
		digest.Buffer_Count == BUFFER_COUNT_MINIMUM,
		"Fresh SHA-256 state has no buffered bytes.",
	)
	aver.Always(
		uint64(digest.Message_Size) == MESSAGE_SIZE_MINIMUM,
		"Fresh SHA-256 state has no accepted bytes.",
	)
	Kind_Invariants(digest.Kind, "Digest_Init.digest.kind.output")
}

// Digest_Reset discards message while retaining selected function.
func Digest_Reset(digest Digest_Handle) {
	Digest_Handle_Invariants(digest, "Digest_Reset.digest.input")
	digest_require(digest)
	digest_reset(Storage_Destination(&digest.Storage))
	aver.Always(
		digest.Buffer_Count == BUFFER_COUNT_MINIMUM,
		"Fresh reset SHA-256 state has no buffered bytes.",
	)
	aver.Always(
		uint64(digest.Message_Size) == MESSAGE_SIZE_MINIMUM,
		"Fresh reset SHA-256 state has no accepted bytes.",
	)
	Kind_Invariants(digest.Kind, "Digest_Reset.digest.kind.output")
}

// Digest_Write consumes one bounded source completely.
func Digest_Write(digest Digest_Handle, source Source) (count Count) {
	defer func() { Count_Invariants(count, "Digest_Write.count") }()
	Digest_Handle_Invariants(digest, "Digest_Write.digest.input")
	Source_Invariants(source, "Digest_Write.source")
	digest_require(digest)
	aver.Always(
		uint64(digest.Message_Size) <= MESSAGE_SIZE_MAXIMUM-uint64(len(source)),
		"SHA-256 message preserves final bit-count width.",
	)
	digest_write(Storage_Destination(&digest.Storage), source)
	Storage_Invariants(digest.Storage, "Digest_Write.digest.output")
	return Count(len(source))
}

// Digest_Sum_Into writes selected complete digest or leaves short storage untouched.
func Digest_Sum_Into(
	digest Digest_Handle, destination Destination,
) (count Output_Count, status Output_Status) {
	defer func() {
		Output_Count_Invariants(count, "Digest_Sum_Into.count")
		Output_Status_Invariants(status, "Digest_Sum_Into.status")
	}()
	Digest_Handle_Invariants(digest, "Digest_Sum_Into.digest")
	Destination_Invariants(destination, "Digest_Sum_Into.destination")
	digest_require(digest)
	size := Digest_Size(digest)
	count = Output_Count(size)
	if len(destination) < int(size) {
		return count, OUTPUT_STATUS_TOO_SMALL
	}
	copy_digest := *digest
	var final_blocks [FINAL_BLOCK_CAPACITY]byte
	buffer_count := int(digest.Buffer_Count)
	var buffered [BLOCK_SIZE]byte
	buffer_copy(digest.Buffer, Block_Destination(buffered[:]))
	copy(final_blocks[:buffer_count], buffered[:buffer_count])
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
	block(State_Destination(&copy_digest.State), Blocks(final_blocks[:final_size]))
	lanes := [...]uint32{
		uint32(copy_digest.State.Lane_0), uint32(copy_digest.State.Lane_1),
		uint32(copy_digest.State.Lane_2), uint32(copy_digest.State.Lane_3),
		uint32(copy_digest.State.Lane_4), uint32(copy_digest.State.Lane_5),
		uint32(copy_digest.State.Lane_6), uint32(copy_digest.State.Lane_7),
	}
	for index := range int(size) / binary.UINT_32_SIZE {
		start := index * binary.UINT_32_SIZE
		binary.Put_Uint_32(
			binary.Bytes(destination[start:start+binary.UINT_32_SIZE]),
			binary.Word_32(lanes[index]), binary.BIG_ENDIAN,
		)
	}
	return count, OUTPUT_STATUS_OK
}

// Digest_Clone_Into copies live state without aliasing caller storage.
func Digest_Clone_Into(destination Digest_Handle, source Digest_Handle) {
	Digest_Handle_Invariants(destination, "Digest_Clone_Into.destination.input")
	Digest_Handle_Invariants(source, "Digest_Clone_Into.source")
	digest_require(source)
	*destination = *source
	Storage_Invariants(destination.Storage, "Digest_Clone_Into.destination.output")
}

// Digest_Size reports selected output width.
func Digest_Size(digest Digest_Handle) (size Size) {
	defer func() { Size_Invariants(size, "Digest_Size.size") }()
	Digest_Handle_Invariants(digest, "Digest_Size.digest")
	digest_require(digest)
	if digest.Kind == KIND_SHA_224 {
		return DIGEST_224_SIZE
	}
	return DIGEST_256_SIZE
}

// Digest_Block_Size reports shared compression block width.
func Digest_Block_Size(digest Digest_Handle) (size Block_Size) {
	defer func() { Block_Size_Invariants(size, "Digest_Block_Size.size") }()
	Digest_Handle_Invariants(digest, "Digest_Block_Size.digest")
	digest_require(digest)
	return BLOCK_SIZE
}

// Checksum_Into computes selected function directly into caller storage.
func Checksum_Into(
	destination Destination, kind Kind, source Source,
) (count Output_Count, status Output_Status) {
	defer func() {
		Output_Count_Invariants(count, "Checksum_Into.count")
		Output_Status_Invariants(status, "Checksum_Into.status")
	}()
	Destination_Invariants(destination, "Checksum_Into.destination")
	Kind_Invariants(kind, "Checksum_Into.kind")
	Source_Invariants(source, "Checksum_Into.source")
	var digest Digest
	Digest_Init(&digest, kind)
	Digest_Write(&digest, source)
	return Digest_Sum_Into(&digest, destination)
}

func digest_write(digest Storage_Destination, source Source) {
	Storage_Destination_Invariants(digest, "digest_write.digest.input")
	Source_Invariants(source, "digest_write.source")
	digest.Message_Size += Message_Size(len(source))
	if digest.Buffer_Count > BUFFER_COUNT_MINIMUM {
		copied := min(BLOCK_SIZE-int(digest.Buffer_Count), len(source))
		if copied > SOURCE_SIZE_MINIMUM {
			buffer_write(
				Buffer_Destination(&digest.Buffer), digest.Buffer_Count,
				Buffer_Source(source[:copied]),
			)
			digest.Buffer_Count += Buffer_Count(copied)
			source = source[copied:]
		}
		if int(digest.Buffer_Count) == BLOCK_SIZE {
			var buffered [BLOCK_SIZE]byte
			buffer_copy(digest.Buffer, Block_Destination(buffered[:]))
			block(State_Destination(&digest.State), Blocks(buffered[:]))
			digest.Buffer = Buffer{}
			digest.Buffer_Count = BUFFER_COUNT_MINIMUM
		}
	}
	if len(source) >= BLOCK_SIZE {
		complete_size := len(source) / BLOCK_SIZE * BLOCK_SIZE
		block(State_Destination(&digest.State), Blocks(source[:complete_size]))
		source = source[complete_size:]
	}
	if len(source) > SOURCE_SIZE_MINIMUM {
		digest.Buffer = Buffer{}
		buffer_write(Buffer_Destination(&digest.Buffer), 0, Buffer_Source(source))
		digest.Buffer_Count = Buffer_Count(len(source))
	}
	Storage_Destination_Invariants(digest, "digest_write.digest.output")
}

func digest_reset(digest Storage_Destination) {
	Storage_Destination_Invariants(digest, "digest_reset.digest.input")
	// Exact FIPS words avoid platform-dependent floating-point recomputation.
	if digest.Kind == KIND_SHA_224 {
		digest.State = State{
			Lane_0: 0xc1059ed8,
			Lane_1: 0x367cd507,
			Lane_2: 0x3070dd17,
			Lane_3: 0xf70e5939,
			Lane_4: 0xffc00b31,
			Lane_5: 0x68581511,
			Lane_6: 0x64f98fa7,
			Lane_7: 0xbefa4fa4,
		}
	} else {
		digest.State = State{
			Lane_0: 0x6a09e667,
			Lane_1: 0xbb67ae85,
			Lane_2: 0x3c6ef372,
			Lane_3: 0xa54ff53a,
			Lane_4: 0x510e527f,
			Lane_5: 0x9b05688c,
			Lane_6: 0x1f83d9ab,
			Lane_7: 0x5be0cd19,
		}
	}
	digest.Buffer = Buffer{}
	digest.Buffer_Count = BUFFER_COUNT_MINIMUM
	digest.Message_Size = Message_Size(MESSAGE_SIZE_MINIMUM)
	aver.Always(
		digest.Buffer_Count == BUFFER_COUNT_MINIMUM,
		"Reset SHA-256 state has no buffered bytes.",
	)
	aver.Always(
		uint64(digest.Message_Size) == MESSAGE_SIZE_MINIMUM,
		"Reset SHA-256 state has no accepted bytes.",
	)
	Kind_Invariants(digest.Kind, "digest_reset.digest.kind.output")
}

func block(state State_Destination, source Blocks) {
	State_Destination_Invariants(state, "block.state.input")
	Blocks_Invariants(source, "block.source")
	const STATE_D_INDEX = binary.UINT_32_SIZE - binary.UINT_8_SIZE
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
		var schedule_storage [ROUND_COUNT]uint32
		schedule := Schedule(schedule_storage[:])
		schedule_make(schedule, Block(source[:BLOCK_SIZE]))
		a := uint32(state.Lane_0)
		b := uint32(state.Lane_1)
		c := uint32(state.Lane_2)
		d := uint32(state.Lane_3)
		e := uint32(state.Lane_4)
		f := uint32(state.Lane_5)
		g := uint32(state.Lane_6)
		h := uint32(state.Lane_7)
		for index := Round_Index(ROUND_INDEX_MINIMUM); index < ROUND_COUNT; index++ {
			sigma_1 := bits.Rotate_Left_32(bits.Word_32(e), STATE_E_ROTATION_FIRST)
			sigma_1 ^= bits.Rotate_Left_32(bits.Word_32(e), STATE_E_ROTATION_SECOND)
			sigma_1 ^= bits.Rotate_Left_32(bits.Word_32(e), STATE_E_ROTATION_THIRD)
			choice := e&f ^ ^e&g
			temporary_1 := h + uint32(sigma_1) + choice
			temporary_1 += uint32(round_constant(index))
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
		state.Lane_0 += State_Lane_0(a)
		state.Lane_1 += State_Lane_1(b)
		state.Lane_2 += State_Lane_2(c)
		state.Lane_3 += State_Lane_3(d)
		state.Lane_4 += State_Lane_4(e)
		state.Lane_5 += State_Lane_5(f)
		state.Lane_6 += State_Lane_6(g)
		state.Lane_7 += State_Lane_7(h)
		source = source[BLOCK_SIZE:]
	}
}

func schedule_make(destination Schedule, source Block) {
	Schedule_Invariants(destination, "schedule_make.destination")
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
		destination[index] = uint32(binary.Uint_32(
			binary.Bytes(word_source), binary.BIG_ENDIAN,
		))
	}
	for index := Round_Index(MESSAGE_WORD_COUNT); index < ROUND_COUNT; index++ {
		value_1 := destination[index-SCHEDULE_VALUE_1_INDEX]
		sigma_1 := bits.Rotate_Left_32(
			bits.Word_32(value_1), SCHEDULE_VALUE_1_ROTATION_FIRST,
		)
		sigma_1 ^= bits.Rotate_Left_32(
			bits.Word_32(value_1), SCHEDULE_VALUE_1_ROTATION_SECOND,
		)
		sigma_1 ^= bits.Word_32(value_1 >> SCHEDULE_VALUE_1_SHIFT)
		value_2 := destination[index-SCHEDULE_VALUE_2_INDEX]
		sigma_2 := bits.Rotate_Left_32(
			bits.Word_32(value_2), SCHEDULE_VALUE_2_ROTATION_FIRST,
		)
		sigma_2 ^= bits.Rotate_Left_32(
			bits.Word_32(value_2), SCHEDULE_VALUE_2_ROTATION_SECOND,
		)
		sigma_2 ^= bits.Word_32(value_2 >> SCHEDULE_VALUE_2_SHIFT)
		destination[index] = uint32(sigma_1) +
			destination[index-SCHEDULE_SUM_1_INDEX]
		destination[index] += uint32(sigma_2) +
			destination[index-SCHEDULE_SUM_2_INDEX]
	}
}

// FIPS derives K[i] from fractional cube roots; exact words prevent floating-point drift.
func round_constant(index Round_Index) (constant Round_Constant) {
	defer func() { Round_Constant_Invariants(constant, "round_constant.constant") }()
	Round_Index_Invariants(index, "round_constant.index")
	constants := [...]Round_Constant{
		0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5,
		0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
		0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3,
		0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
		0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc,
		0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
		0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7,
		0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
		0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13,
		0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
		0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3,
		0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
		0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5,
		0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
		0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208,
		0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
	}
	return constants[index]
}

func digest_require(digest Digest_Handle) {
	Digest_Handle_Invariants(digest, "digest_require.digest")
	aver.Always(
		digest.Ready == STATE_READY_MARKER,
		"SHA-256 operations require Digest_Init.",
	)
}
