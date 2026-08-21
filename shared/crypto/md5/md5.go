// Package md5 computes RFC 1321 MD5 with bounded caller-owned state and output.
//
// MD5 is cryptographically broken. Use only where protocol compatibility requires it.
package md5

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
)

// STATE_LANE_COUNT stores one result as 32-bit words.
const STATE_LANE_COUNT = binary.UINT_32_SIZE

// DIGEST_BIT_COUNT derives the result width from its state words.
const DIGEST_BIT_COUNT = STATE_LANE_COUNT * bits.BIT_COUNT_32_MAXIMUM

// DIGEST_SIZE converts result width to bytes.
const DIGEST_SIZE = DIGEST_BIT_COUNT / binary.BITS_PER_BYTE

// STATE_WORD_COUNT counts compression lanes and lifecycle identity.
const STATE_WORD_COUNT = STATE_LANE_COUNT + binary.UINT_8_SIZE

// STATE_READY_MARKER separates initialized state from zero caller storage.
const STATE_READY_MARKER = true

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

// BUFFER_SOURCE_SIZE_MINIMUM excludes empty writes handled before packing.
const BUFFER_SOURCE_SIZE_MINIMUM = binary.UINT_8_SIZE

// BUFFER_SOURCE_SIZE_MAXIMUM prevents packing one complete block as partial state.
const BUFFER_SOURCE_SIZE_MAXIMUM = BUFFER_COUNT_MAXIMUM

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

// ROUND_CONSTANT_MINIMUM is the least RFC table word.
const ROUND_CONSTANT_MINIMUM uint32 = 0x02441453

// ROUND_CONSTANT_MAXIMUM is the greatest RFC table word.
const ROUND_CONSTANT_MAXIMUM uint32 = 0xffff5bb1

// ROTATION_MINIMUM is the least RFC rotation distance.
const ROTATION_MINIMUM uint8 = STATE_LANE_COUNT

// ROTATION_MAXIMUM is the greatest RFC rotation distance.
const ROTATION_MAXIMUM uint8 = bits.BIT_COUNT_32_MAXIMUM - binary.UINT_64_SIZE - binary.UINT_8_SIZE

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
		"MD5 compression input contains complete blocks.",
	)
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

// Output_Count is required MD5 width on complete or short output.
type Output_Count uint8

// Output_Count_Invariants fixes required MD5 width.
func Output_Count_Invariants(value Output_Count, _ aver.Namespace) {
	aver.Always(
		uint8(value) == uint8(OUTPUT_COUNT_REQUIRED),
		"MD5 output count equals required digest width.",
	)
}

// Output_Status reports caller output capacity.
type Output_Status uint8

// Output_Status_Invariants covers complete and short output.
func Output_Status_Invariants(value Output_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(OUTPUT_STATUS_OK), uint8(OUTPUT_STATUS_TOO_SMALL)).
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

// State_Lane_0 exposes first compression word without sharing one invariant axis.
type State_Lane_0 uint32

// State_Lane_0_Invariants preserves complete word domain.
func State_Lane_0_Invariants(value State_Lane_0, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, bits.WORD_32_MAXIMUM).
		Ensure()
}

// State_Lane_1 exposes second compression word without sharing one invariant axis.
type State_Lane_1 uint32

// State_Lane_1_Invariants preserves complete word domain.
func State_Lane_1_Invariants(value State_Lane_1, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, bits.WORD_32_MAXIMUM).
		Ensure()
}

// State_Lane_2 exposes third compression word without sharing one invariant axis.
type State_Lane_2 uint32

// State_Lane_2_Invariants preserves complete word domain.
func State_Lane_2_Invariants(value State_Lane_2, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, bits.WORD_32_MAXIMUM).
		Ensure()
}

// State_Lane_3 exposes fourth compression word without sharing one invariant axis.
type State_Lane_3 uint32

// State_Lane_3_Invariants preserves complete word domain.
func State_Lane_3_Invariants(value State_Lane_3, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, bits.WORD_32_MAXIMUM).
		Ensure()
}

// State keeps compression words independently visible to invariants.
type State struct {
	// Lane_0 avoids array-hidden state.
	Lane_0 State_Lane_0
	// Lane_1 avoids array-hidden state.
	Lane_1 State_Lane_1
	// Lane_2 avoids array-hidden state.
	Lane_2 State_Lane_2
	// Lane_3 avoids array-hidden state.
	Lane_3 State_Lane_3
}

// State_Invariants exposes every word.
func State_Invariants(value State, namespace aver.Namespace) {
	State_Lane_0_Invariants(value.Lane_0, namespace)
	State_Lane_1_Invariants(value.Lane_1, namespace)
	State_Lane_2_Invariants(value.Lane_2, namespace)
	State_Lane_3_Invariants(value.Lane_3, namespace)
}

// State_Destination names mutable compression state.
type State_Destination *State

// State_Destination_Invariants composes state when storage exists.
func State_Destination_Invariants(value State_Destination, namespace aver.Namespace) {
	if value == nil {
		return
	}
	State_Invariants(*value, namespace)
}

// Round_Index identifies one RFC 1321 compression step.
type Round_Index uint8

// Round_Index_Invariants covers every compression step.
func Round_Index_Invariants(value Round_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), ROUND_INDEX_MINIMUM, ROUND_INDEX_MAXIMUM).
		Ensure()
}

// Round_Quarter_Index identifies one step inside a 16-step round quarter.
type Round_Quarter_Index uint8

// Round_Quarter_Index_Invariants covers every quarter-local step.
func Round_Quarter_Index_Invariants(value Round_Quarter_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), ROUND_QUARTER_INDEX_MINIMUM, ROUND_QUARTER_INDEX_MAXIMUM,
		).
		Ensure()
}

// Round_Constant stores one RFC 1321 additive word.
type Round_Constant uint32

// Round_Constant_Invariants preserves the full word needed by the table.
func Round_Constant_Invariants(value Round_Constant, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), ROUND_CONSTANT_MINIMUM, ROUND_CONSTANT_MAXIMUM).
		Ensure()
}

// Rotation stores one RFC 1321 rotation distance.
type Rotation uint8

// Rotation_Invariants preserves the 32-bit rotation domain.
func Rotation_Invariants(value Rotation, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), ROTATION_MINIMUM, ROTATION_MAXIMUM).
		Ensure()
}

// Size is MD5 digest byte width.
type Size int

// Size_Invariants fixes MD5 digest width.
func Size_Invariants(value Size, _ aver.Namespace) {
	aver.Always(int(value) == DIGEST_SIZE, "MD5 digest size equals result width.")
}

// Block_Size is MD5 compression-block byte width.
type Block_Size int

// Block_Size_Invariants fixes MD5 compression-block width.
func Block_Size_Invariants(value Block_Size, _ aver.Namespace) {
	aver.Always(int(value) == BLOCK_SIZE, "MD5 block size equals compression width.")
}

// Ready separates initialized state from zero caller storage.
type Ready bool

// Ready_Invariants requires tests to witness both lifecycle states.
func Ready_Invariants(value Ready, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "MD5 state is initialized.").
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
	aver.Always(len(value) == BLOCK_SIZE, "MD5 scratch has one complete block.")
}

// Buffer_Lane_1 exposes first packed word without sharing one invariant axis.
type Buffer_Lane_1 uint64

// Buffer_Lane_1_Invariants preserves complete word domain.
func Buffer_Lane_1_Invariants(value Buffer_Lane_1, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_2 exposes second packed word without sharing one invariant axis.
type Buffer_Lane_2 uint64

// Buffer_Lane_2_Invariants preserves complete word domain.
func Buffer_Lane_2_Invariants(value Buffer_Lane_2, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_3 exposes third packed word without sharing one invariant axis.
type Buffer_Lane_3 uint64

// Buffer_Lane_3_Invariants preserves complete word domain.
func Buffer_Lane_3_Invariants(value Buffer_Lane_3, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_4 exposes fourth packed word without sharing one invariant axis.
type Buffer_Lane_4 uint64

// Buffer_Lane_4_Invariants preserves complete word domain.
func Buffer_Lane_4_Invariants(value Buffer_Lane_4, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_5 exposes fifth packed word without sharing one invariant axis.
type Buffer_Lane_5 uint64

// Buffer_Lane_5_Invariants preserves complete word domain.
func Buffer_Lane_5_Invariants(value Buffer_Lane_5, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_6 exposes sixth packed word without sharing one invariant axis.
type Buffer_Lane_6 uint64

// Buffer_Lane_6_Invariants preserves complete word domain.
func Buffer_Lane_6_Invariants(value Buffer_Lane_6, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_7 exposes seventh packed word without sharing one invariant axis.
type Buffer_Lane_7 uint64

// Buffer_Lane_7_Invariants preserves complete word domain.
func Buffer_Lane_7_Invariants(value Buffer_Lane_7, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_8 exposes eighth packed word without sharing one invariant axis.
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

// Buffer_Destination names mutable packed storage without copying eight words.
type Buffer_Destination *Buffer

// Buffer_Destination_Invariants composes packed storage when present.
func Buffer_Destination_Invariants(value Buffer_Destination, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Buffer_Invariants(*value, namespace)
}

// Storage groups streaming data whose internal helpers can mutate it.
type Storage struct {
	State
	Buffer
	// Buffer_Count identifies live Buffer prefix.
	Buffer_Count Buffer_Count
	// Message_Size counts accepted bytes for final bit-count encoding.
	Message_Size Message_Size
}

// Storage_Invariants composes mutable hash data without lifecycle policy.
func Storage_Invariants(value Storage, namespace aver.Namespace) {
	State_Invariants(value.State, namespace)
	Buffer_Invariants(value.Buffer, namespace)
	Buffer_Count_Invariants(value.Buffer_Count, namespace)
	Message_Size_Invariants(value.Message_Size, namespace)
	aver.Always(
		uint64(value.Buffer_Count) == uint64(value.Message_Size)%BLOCK_SIZE,
		"Partial MD5 block equals message remainder.",
	)
}

// Storage_Destination names mutable hash data after lifecycle validation.
type Storage_Destination *Storage

// Storage_Destination_Invariants composes hash data when storage exists.
func Storage_Destination_Invariants(value Storage_Destination, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Storage_Invariants(*value, namespace)
}

// Digest is caller-owned streaming MD5 state.
type Digest struct {
	Storage
	// Ready rejects zero caller storage before compression state is trusted.
	Ready Ready
}

// Digest_Invariants composes all scalar state and partial-block relation.
func Digest_Invariants(value Digest, namespace aver.Namespace) {
	Storage_Invariants(value.Storage, namespace)
	Ready_Invariants(value.Ready, namespace)
}

// Digest_Handle names mutable streaming state.
type Digest_Handle *Digest

// Digest_Handle_Invariants composes state when storage exists.
func Digest_Handle_Invariants(value Digest_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Digest_Invariants(*value, namespace)
}

// Scalar packing preserves copy-safe, stack-owned digest state.
func buffer_write(
	buffer Buffer_Destination, position Buffer_Count, source Buffer_Source,
) {
	Buffer_Destination_Invariants(buffer, "buffer_write.buffer")
	Buffer_Count_Invariants(position, "buffer_write.position")
	Buffer_Source_Invariants(source, "buffer_write.source")
	aver.Always(
		int(position)+len(source) <= BLOCK_SIZE,
		"MD5 partial bytes fit one compression block.",
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

// Digest_Init establishes RFC 1321 initial state in caller storage.
func Digest_Init(digest Digest_Handle) {
	Digest_Handle_Invariants(digest, "Digest_Init.digest.input")
	digest.State = State{
		Lane_0: State_Lane_0(INITIAL_STATE_0),
		Lane_1: State_Lane_1(INITIAL_STATE_1),
		Lane_2: State_Lane_2(INITIAL_STATE_2),
		Lane_3: State_Lane_3(INITIAL_STATE_3),
	}
	digest.Buffer = Buffer{}
	digest.Buffer_Count = BUFFER_COUNT_MINIMUM
	digest.Message_Size = Message_Size(MESSAGE_SIZE_MINIMUM)
	digest.Ready = STATE_READY_MARKER
	aver.Always(
		digest.Buffer_Count == BUFFER_COUNT_MINIMUM,
		"Fresh MD5 state has no buffered bytes.",
	)
	aver.Always(
		uint64(digest.Message_Size) == MESSAGE_SIZE_MINIMUM,
		"Fresh MD5 state has no accepted bytes.",
	)
}

// Digest_Reset discards message while retaining caller storage.
func Digest_Reset(digest Digest_Handle) {
	Digest_Handle_Invariants(digest, "Digest_Reset.digest.input")
	digest_require(digest)
	Digest_Init(digest)
}

// Digest_Write consumes one bounded source completely.
func Digest_Write(digest Digest_Handle, source Source) (count Count) {
	defer func() { Count_Invariants(count, "Digest_Write.count") }()
	Digest_Handle_Invariants(digest, "Digest_Write.digest.input")
	Source_Invariants(source, "Digest_Write.source")
	digest_require(digest)
	aver.Always(
		uint64(digest.Message_Size) <= MESSAGE_SIZE_MAXIMUM-uint64(len(source)),
		"MD5 message preserves final bit-count width.",
	)
	digest_write(Storage_Destination(&digest.Storage), source)
	Storage_Invariants(digest.Storage, "Digest_Write.digest.output")
	return Count(len(source))
}

// Digest_Sum_Into writes complete digest or leaves short storage untouched.
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
	if len(destination) < DIGEST_SIZE {
		return OUTPUT_COUNT_REQUIRED, OUTPUT_STATUS_TOO_SMALL
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
		binary.Word_64(message_bits), binary.LITTLE_ENDIAN,
	)
	block(State_Destination(&copy_digest.State), Blocks(final_blocks[:final_size]))
	lanes := [...]uint32{
		uint32(copy_digest.State.Lane_0), uint32(copy_digest.State.Lane_1),
		uint32(copy_digest.State.Lane_2), uint32(copy_digest.State.Lane_3),
	}
	for index := range STATE_LANE_COUNT {
		start := index * binary.UINT_32_SIZE
		binary.Put_Uint_32(
			binary.Bytes(destination[start:start+binary.UINT_32_SIZE]),
			binary.Word_32(lanes[index]), binary.LITTLE_ENDIAN,
		)
	}
	return OUTPUT_COUNT_REQUIRED, OUTPUT_STATUS_OK
}

// Digest_Clone_Into copies live state without aliasing caller storage.
func Digest_Clone_Into(destination Digest_Handle, source Digest_Handle) {
	Digest_Handle_Invariants(destination, "Digest_Clone_Into.destination.input")
	Digest_Handle_Invariants(source, "Digest_Clone_Into.source")
	digest_require(source)
	*destination = *source
	Storage_Invariants(destination.Storage, "Digest_Clone_Into.destination.output")
}

// Digest_Size reports MD5 output width.
func Digest_Size(digest Digest_Handle) (size Size) {
	defer func() { Size_Invariants(size, "Digest_Size.size") }()
	Digest_Handle_Invariants(digest, "Digest_Size.digest")
	digest_require(digest)
	return DIGEST_SIZE
}

// Digest_Block_Size reports MD5 compression block width.
func Digest_Block_Size(digest Digest_Handle) (size Block_Size) {
	defer func() { Block_Size_Invariants(size, "Digest_Block_Size.size") }()
	Digest_Handle_Invariants(digest, "Digest_Block_Size.digest")
	digest_require(digest)
	return BLOCK_SIZE
}

// Checksum_Into computes one bounded source directly into caller storage.
func Checksum_Into(
	destination Destination, source Source,
) (count Output_Count, status Output_Status) {
	defer func() {
		Output_Count_Invariants(count, "Checksum_Into.count")
		Output_Status_Invariants(status, "Checksum_Into.status")
	}()
	Destination_Invariants(destination, "Checksum_Into.destination")
	Source_Invariants(source, "Checksum.source")
	var digest Digest
	Digest_Init(&digest)
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

func block(state State_Destination, source Blocks) {
	State_Destination_Invariants(state, "block.state.input")
	Blocks_Invariants(source, "block.source")
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
		a := uint32(state.Lane_0)
		b := uint32(state.Lane_1)
		c := uint32(state.Lane_2)
		d := uint32(state.Lane_3)
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
			constant := round_constant(index)
			round_value := a + combined + uint32(constant)
			round_value += message[message_index]
			rotation := round_rotation(index)
			rotated := bits.Rotate_Left_32(
				bits.Word_32(round_value), bits.Rotation(rotation),
			)
			a, d, c, b = d, c, b, b+uint32(rotated)
		}
		state.Lane_0 += State_Lane_0(a)
		state.Lane_1 += State_Lane_1(b)
		state.Lane_2 += State_Lane_2(c)
		state.Lane_3 += State_Lane_3(d)
		source = source[BLOCK_SIZE:]
	}
}

func round_rotation(index Round_Index) (rotation Rotation) {
	defer func() { Rotation_Invariants(rotation, "round_rotation.rotation") }()
	Round_Index_Invariants(index, "round_rotation.index")
	rotations := [...]Rotation{
		7, 12, 17, 22, 7, 12, 17, 22, 7, 12, 17, 22, 7, 12, 17, 22,
		5, 9, 14, 20, 5, 9, 14, 20, 5, 9, 14, 20, 5, 9, 14, 20,
		4, 11, 16, 23, 4, 11, 16, 23, 4, 11, 16, 23, 4, 11, 16, 23,
		6, 10, 15, 21, 6, 10, 15, 21, 6, 10, 15, 21, 6, 10, 15, 21,
	}
	return rotations[index]
}

// RFC derives K[i] as floor(abs(sin(i+1))*2^32); exact words avoid floating-point drift.
func round_constant(index Round_Index) (constant Round_Constant) {
	defer func() { Round_Constant_Invariants(constant, "round_constant.constant") }()
	Round_Index_Invariants(index, "round_constant.index")
	constants := [...]Round_Constant{
		0xd76aa478, 0xe8c7b756, 0x242070db, 0xc1bdceee,
		0xf57c0faf, 0x4787c62a, 0xa8304613, 0xfd469501,
		0x698098d8, 0x8b44f7af, 0xffff5bb1, 0x895cd7be,
		0x6b901122, 0xfd987193, 0xa679438e, 0x49b40821,
		0xf61e2562, 0xc040b340, 0x265e5a51, 0xe9b6c7aa,
		0xd62f105d, 0x02441453, 0xd8a1e681, 0xe7d3fbc8,
		0x21e1cde6, 0xc33707d6, 0xf4d50d87, 0x455a14ed,
		0xa9e3e905, 0xfcefa3f8, 0x676f02d9, 0x8d2a4c8a,
		0xfffa3942, 0x8771f681, 0x6d9d6122, 0xfde5380c,
		0xa4beea44, 0x4bdecfa9, 0xf6bb4b60, 0xbebfbc70,
		0x289b7ec6, 0xeaa127fa, 0xd4ef3085, 0x04881d05,
		0xd9d4d039, 0xe6db99e5, 0x1fa27cf8, 0xc4ac5665,
		0xf4292244, 0x432aff97, 0xab9423a7, 0xfc93a039,
		0x655b59c3, 0x8f0ccc92, 0xffeff47d, 0x85845dd1,
		0x6fa87e4f, 0xfe2ce6e0, 0xa3014314, 0x4e0811a1,
		0xf7537e82, 0xbd3af235, 0x2ad7d2bb, 0xeb86d391,
	}
	return constants[index]
}

func digest_require(digest Digest_Handle) {
	Digest_Handle_Invariants(digest, "digest_require.digest")
	aver.Always(
		digest.Ready == STATE_READY_MARKER,
		"MD5 operations require Digest_Init.",
	)
}
