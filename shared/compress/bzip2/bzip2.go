// Package bzip2 decodes bzip2 streams into fixed caller storage.
package bzip2

import "local/james-orcales/shared/invariant/default"

// FILE_MAGIC identifies bzip2 container.
const FILE_MAGIC = 0x425a

// BLOCK_MAGIC identifies compressed block.
const BLOCK_MAGIC = 0x314159265359

// FINAL_MAGIC identifies stream checksum trailer.
const FINAL_MAGIC = 0x177245385090

// BLOCK_SIZE_UNIT converts header level into transform item count.
const BLOCK_SIZE_UNIT = 100_000

// BYTE_VALUE_COUNT covers full byte alphabet.
const BYTE_VALUE_COUNT = 256

// SYMBOL_COUNT_MAXIMUM includes two run symbols.
const SYMBOL_COUNT_MAXIMUM = 258

// HUFFMAN_TREE_COUNT_MAXIMUM bounds per-block tree set.
const HUFFMAN_TREE_COUNT_MAXIMUM = 6

// HUFFMAN_CODE_SIZE_MAXIMUM bounds one bzip2 canonical code.
const HUFFMAN_CODE_SIZE_MAXIMUM = 20

// SELECTOR_GROUP_SIZE fixes symbols decoded per tree selector.
const SELECTOR_GROUP_SIZE = 50

// REPEAT_COUNT_MAXIMUM prevents hostile run-count overflow.
const REPEAT_COUNT_MAXIMUM = 2 * 1024 * 1024

// BYTE_COUNT_MINIMUM is empty caller storage or input.
const BYTE_COUNT_MINIMUM = 0

// BYTE_COUNT_MAXIMUM matches the caller's per-stream resource boundary.
const BYTE_COUNT_MAXIMUM = 64 * 1024 * 1024

// TRANSFORM_COUNT_MAXIMUM is the largest declared bzip2 block.
const TRANSFORM_COUNT_MAXIMUM = 9 * BLOCK_SIZE_UNIT

// BLOCK_COUNT_MINIMUM is an empty decoded block.
const BLOCK_COUNT_MINIMUM = 0

// BLOCK_ITEM_COUNT_MINIMUM is one decoded transform item.
const BLOCK_ITEM_COUNT_MINIMUM = 1

// SYMBOL_COUNT_MINIMUM is one used byte.
const SYMBOL_COUNT_MINIMUM = 1

// ALPHABET_COUNT_MINIMUM is one used byte plus two run symbols.
const ALPHABET_COUNT_MINIMUM = SYMBOL_COUNT_MINIMUM + 2

// TREE_COUNT_MINIMUM is the smallest bzip2 tree set.
const TREE_COUNT_MINIMUM = 2

// SELECTOR_COUNT_MINIMUM is one selector.
const SELECTOR_COUNT_MINIMUM = 1

// SELECTOR_COUNT_MAXIMUM is the encoded fifteen-bit maximum.
const SELECTOR_COUNT_MAXIMUM = 1<<15 - 1

// SELECTOR_INDEX_MINIMUM is the first selector.
const SELECTOR_INDEX_MINIMUM = 0

// GROUP_COUNT_MINIMUM is an empty selector group.
const GROUP_COUNT_MINIMUM = 0

// TREE_INDEX_MINIMUM is the first tree.
const TREE_INDEX_MINIMUM = 0

// TREE_INDEX_MAXIMUM is the final tree slot.
const TREE_INDEX_MAXIMUM = HUFFMAN_TREE_COUNT_MAXIMUM - 1

// REPEAT_COUNT_MINIMUM is no pending run.
const REPEAT_COUNT_MINIMUM = 0

// BYTE_VALUE_MINIMUM is the first byte.
const BYTE_VALUE_MINIMUM = 0

// BYTE_VALUE_MAXIMUM is the final byte.
const BYTE_VALUE_MAXIMUM = 255

// POSITION_MINIMUM is the first alphabet member.
const POSITION_MINIMUM = 0

// POSITION_MAXIMUM is the final move-to-front byte position.
const POSITION_MAXIMUM = BYTE_VALUE_MAXIMUM

// HUFFMAN_SYMBOL_MAXIMUM is the end symbol for the largest alphabet.
const HUFFMAN_SYMBOL_MAXIMUM = SYMBOL_COUNT_MAXIMUM - 1

// FIRST_POSITION_MINIMUM is the first transform item.
const FIRST_POSITION_MINIMUM = 0

// FIRST_POSITION_MAXIMUM is the final largest transform item.
const FIRST_POSITION_MAXIMUM = TRANSFORM_COUNT_MAXIMUM - 1

// BIT_READ_COUNT_MINIMUM is one requested bit.
const BIT_READ_COUNT_MINIMUM = 1

// BIT_READ_COUNT_MAXIMUM is the widest bzip2 marker.
const BIT_READ_COUNT_MAXIMUM = 48

// BIT_VALUE_MINIMUM is an all-zero bit value.
const BIT_VALUE_MINIMUM = 0

// BIT_VALUE_MAXIMUM is the widest all-one bit value.
const BIT_VALUE_MAXIMUM = 1<<BIT_READ_COUNT_MAXIMUM - 1

// BIT_BUFFER_MINIMUM is an empty bit remainder.
const BIT_BUFFER_MINIMUM = 0

// BIT_BUFFER_MAXIMUM is the seven-bit remainder after one read.
const BIT_BUFFER_MAXIMUM = 127

// BIT_COUNT_MINIMUM is an empty bit remainder.
const BIT_COUNT_MINIMUM = 0

// BIT_COUNT_MAXIMUM is the widest bit remainder.
const BIT_COUNT_MAXIMUM = 7

// CHECKSUM_SIZE is CRC-32 storage width.
const CHECKSUM_SIZE = 4

// BLOCK_SOURCE_SIZE_MAXIMUM follows the wrapper, marker, and block checksum.
const BLOCK_SOURCE_SIZE_MAXIMUM = BYTE_COUNT_MAXIMUM - 14

// BIT_SOURCE_SIZE_MAXIMUM excludes the four-byte stream header.
const BIT_SOURCE_SIZE_MAXIMUM = BYTE_COUNT_MAXIMUM - 4

// SYMBOL_SOURCE_SIZE_MAXIMUM follows randomized and original-position fields.
const SYMBOL_SOURCE_SIZE_MAXIMUM = BYTE_COUNT_MAXIMUM - 18

// TREE_SOURCE_SIZE_MAXIMUM follows the smallest nonempty symbol bitmap.
const TREE_SOURCE_SIZE_MAXIMUM = BYTE_COUNT_MAXIMUM - 22

// SELECTOR_SOURCE_SIZE_MAXIMUM follows tree and selector counts.
const SELECTOR_SOURCE_SIZE_MAXIMUM = BYTE_COUNT_MAXIMUM - 24

// PAYLOAD_SOURCE_SIZE_MAXIMUM follows the smallest complete tree set.
const PAYLOAD_SOURCE_SIZE_MAXIMUM = BYTE_COUNT_MAXIMUM - 26

// REMAINDER_SOURCE_SIZE_MAXIMUM follows the smallest decoded block.
const REMAINDER_SOURCE_SIZE_MAXIMUM = BYTE_COUNT_MAXIMUM - 27

// TRAILER_SOURCE_SIZE_MAXIMUM follows an empty stream marker.
const TRAILER_SOURCE_SIZE_MAXIMUM = BYTE_COUNT_MAXIMUM - 10

// SELECTOR_BIT_BUFFER_MAXIMUM is the five-bit selector-list remainder.
const SELECTOR_BIT_BUFFER_MAXIMUM = 31

// SYMBOL_BIT_COUNT is the remainder after the fixed block prefix.
const SYMBOL_BIT_COUNT = 7

// SELECTOR_BIT_COUNT is the remainder after the selector count.
const SELECTOR_BIT_COUNT = 5

// Status reports decode result without interface boxing or owned error text.
type Status uint8

// Status_Invariants states every decode result.
func Status_Invariants(value Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_OUTPUT_TOO_SMALL), uint8(STATUS_WORKSPACE_TOO_SMALL),
		).
		Ensure()
}

// STATUS_OK means compressed input filled destination successfully.
const STATUS_OK Status = 0

// STATUS_INPUT_INVALID means compressed input violated bzip2 format.
const STATUS_INPUT_INVALID Status = 1

// STATUS_OUTPUT_TOO_SMALL means decoded output exceeded destination.
const STATUS_OUTPUT_TOO_SMALL Status = 2

// STATUS_WORKSPACE_TOO_SMALL means transform storage cannot fit declared block.
const STATUS_WORKSPACE_TOO_SMALL Status = 3

// Header_Status reports the three header outcomes.
type Header_Status uint8

// Header_Status_Invariants states every header outcome.
func Header_Status_Invariants(value Header_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(HEADER_STATUS_OK), uint8(HEADER_STATUS_INPUT_INVALID),
			uint8(HEADER_STATUS_WORKSPACE_TOO_SMALL),
		).
		Ensure()
}

// HEADER_STATUS_OK means the header and workspace are valid.
const HEADER_STATUS_OK Header_Status = 0

// HEADER_STATUS_INPUT_INVALID means the header is malformed.
const HEADER_STATUS_INPUT_INVALID Header_Status = 1

// HEADER_STATUS_WORKSPACE_TOO_SMALL means the declared block cannot fit.
const HEADER_STATUS_WORKSPACE_TOO_SMALL Header_Status = 3

// Emit_Status reports the three block-emission outcomes.
type Emit_Status uint8

// Emit_Status_Invariants states every block-emission outcome.
func Emit_Status_Invariants(value Emit_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(EMIT_STATUS_OK), uint8(EMIT_STATUS_INPUT_INVALID),
			uint8(EMIT_STATUS_OUTPUT_TOO_SMALL),
		).
		Ensure()
}

// EMIT_STATUS_OK means the transform filled caller storage.
const EMIT_STATUS_OK Emit_Status = 0

// EMIT_STATUS_INPUT_INVALID means the transform chain escaped its block.
const EMIT_STATUS_INPUT_INVALID Emit_Status = 1

// EMIT_STATUS_OUTPUT_TOO_SMALL means decoded bytes exceeded caller storage.
const EMIT_STATUS_OUTPUT_TOO_SMALL Emit_Status = 2

// Destination is caller-owned decoded storage.
type Destination []byte

// Destination_Invariants bounds one decoded stream.
func Destination_Invariants(value Destination, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_MAXIMUM).
		Ensure()
}

// Transform is caller-owned inverse-transform storage.
type Transform []uint32

// Transform_Invariants bounds one bzip2 block workspace.
func Transform_Invariants(value Transform, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), BLOCK_COUNT_MINIMUM, TRANSFORM_COUNT_MAXIMUM).
		Ensure()
}

// Block_Storage is workspace reserved for one declared block.
type Block_Storage []uint32

// Block_Storage_Invariants bounds declared block workspace.
func Block_Storage_Invariants(value Block_Storage, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), BLOCK_SIZE_UNIT, TRANSFORM_COUNT_MAXIMUM).
		Ensure()
}

// Block is one populated inverse-transform table.
type Block []uint32

// Block_Invariants bounds one nonempty transform table.
func Block_Invariants(value Block, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), BLOCK_ITEM_COUNT_MINIMUM, TRANSFORM_COUNT_MAXIMUM).
		Ensure()
}

// Compressed is caller-owned bzip2 input.
type Compressed []byte

// Compressed_Invariants bounds one encoded stream.
func Compressed_Invariants(value Compressed, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_MAXIMUM).
		Ensure()
}

// Bit_Source is encoded input after the stream header.
type Bit_Source []byte

// Bit_Source_Invariants bounds the encoded bitstream remainder.
func Bit_Source_Invariants(value Bit_Source, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), BYTE_COUNT_MINIMUM, BIT_SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Count is decoded stream byte count.
type Count int

// Count_Invariants bounds one decoded stream count.
func Count_Invariants(value Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_MAXIMUM).
		Ensure()
}

// Block_Count is one transform item count.
type Block_Count int

// Block_Count_Invariants bounds one decoded block.
func Block_Count_Invariants(value Block_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), BLOCK_COUNT_MINIMUM, TRANSFORM_COUNT_MAXIMUM).
		Ensure()
}

// Block_Limit is one declared block item limit.
type Block_Limit int

// Block_Limit_Invariants bounds one declared bzip2 block.
func Block_Limit_Invariants(value Block_Limit, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), BLOCK_SIZE_UNIT, TRANSFORM_COUNT_MAXIMUM).
		Ensure()
}

// Symbol_Count is one used byte alphabet count.
type Symbol_Count int

// Symbol_Count_Invariants bounds the used byte alphabet.
func Symbol_Count_Invariants(value Symbol_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), SYMBOL_COUNT_MINIMUM, BYTE_VALUE_COUNT).
		Ensure()
}

// Alphabet_Count includes two run symbols.
type Alphabet_Count int

// Alphabet_Count_Invariants bounds one Huffman alphabet.
func Alphabet_Count_Invariants(value Alphabet_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), ALPHABET_COUNT_MINIMUM, SYMBOL_COUNT_MAXIMUM).
		Ensure()
}

// Tree_Count is one block's Huffman tree count.
type Tree_Count int

// Tree_Count_Invariants bounds one tree set.
func Tree_Count_Invariants(value Tree_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), TREE_COUNT_MINIMUM, HUFFMAN_TREE_COUNT_MAXIMUM).
		Ensure()
}

// Selector_Count is one block's selector count.
type Selector_Count int

// Selector_Count_Invariants bounds the encoded selector count.
func Selector_Count_Invariants(value Selector_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), SELECTOR_COUNT_MINIMUM, SELECTOR_COUNT_MAXIMUM).
		Ensure()
}

// Selector_Index is one selector position.
type Selector_Index int

// Selector_Index_Invariants bounds one selector position.
func Selector_Index_Invariants(value Selector_Index, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), SELECTOR_INDEX_MINIMUM, SELECTOR_COUNT_MAXIMUM).
		Ensure()
}

// Group_Count is symbols decoded with one selector.
type Group_Count int

// Group_Count_Invariants bounds one selector group.
func Group_Count_Invariants(value Group_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), GROUP_COUNT_MINIMUM, SELECTOR_GROUP_SIZE).
		Ensure()
}

// Tree_Index is one tree slot.
type Tree_Index int

// Tree_Index_Invariants bounds one tree slot.
func Tree_Index_Invariants(value Tree_Index, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), TREE_INDEX_MINIMUM, TREE_INDEX_MAXIMUM).
		Ensure()
}

// Repeat_Count is one pending run count.
type Repeat_Count int

// Repeat_Count_Invariants bounds hostile run arithmetic.
func Repeat_Count_Invariants(value Repeat_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), REPEAT_COUNT_MINIMUM, REPEAT_COUNT_MAXIMUM).
		Ensure()
}

// Byte_Value is one decoded byte.
type Byte_Value uint8

// Byte_Value_Invariants states the complete byte alphabet.
func Byte_Value_Invariants(value Byte_Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), BYTE_VALUE_MINIMUM, BYTE_VALUE_MAXIMUM).
		Ensure()
}

// Position is one move-to-front position.
type Position int

// Position_Invariants bounds the largest alphabet position.
func Position_Invariants(value Position, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), POSITION_MINIMUM, POSITION_MAXIMUM).
		Ensure()
}

// Huffman_Symbol is one decoded alphabet member.
type Huffman_Symbol uint16

// Huffman_Symbol_Invariants bounds the largest block alphabet.
func Huffman_Symbol_Invariants(value Huffman_Symbol, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint16(uint16(value), POSITION_MINIMUM, HUFFMAN_SYMBOL_MAXIMUM).
		Ensure()
}

// First_Position is one inverse-transform start position.
type First_Position uint32

// First_Position_Invariants bounds the largest block position.
func First_Position_Invariants(value First_Position, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint32(uint32(value), FIRST_POSITION_MINIMUM, FIRST_POSITION_MAXIMUM).
		Ensure()
}

// Boolean is one decoder report.
type Boolean bool

// Boolean_Invariants states both report values.
func Boolean_Invariants(value Boolean, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "The decoder report is true.").
		Ensure()
}

// Bit_Read_Count is one bounded reader request.
type Bit_Read_Count uint

// Bit_Read_Count_Invariants bounds one reader request.
func Bit_Read_Count_Invariants(value Bit_Read_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint(uint(value), BIT_READ_COUNT_MINIMUM, BIT_READ_COUNT_MAXIMUM).
		Ensure()
}

// Bit_Value is one bounded reader result.
type Bit_Value uint64

// Bit_Value_Invariants bounds the widest reader result.
func Bit_Value_Invariants(value Bit_Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), BIT_VALUE_MINIMUM, BIT_VALUE_MAXIMUM).
		Ensure()
}

// Bit_Buffer is unread low-order storage.
type Bit_Buffer uint64

// Bit_Buffer_Invariants bounds the remainder after one reader operation.
func Bit_Buffer_Invariants(value Bit_Buffer, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), BIT_BUFFER_MINIMUM, BIT_BUFFER_MAXIMUM).
		Ensure()
}

// Bit_Count is unread bit count.
type Bit_Count uint

// Bit_Count_Invariants bounds the remainder after one reader operation.
func Bit_Count_Invariants(value Bit_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint(uint(value), BIT_COUNT_MINIMUM, BIT_COUNT_MAXIMUM).
		Ensure()
}

// Checksum is one CRC-32 value in network byte order.
type Checksum [CHECKSUM_SIZE]byte

// Checksum_Invariants fixes CRC-32 storage width.
func Checksum_Invariants(value Checksum, namespace invariant.Namespace) {
	invariant.Always(
		len(value) == CHECKSUM_SIZE,
		"A CRC checksum occupies exactly four bytes.",
	)
}

// Selector_Order is one move-to-front tree order.
type Selector_Order []byte

// Selector_Order_Invariants bounds one tree order.
func Selector_Order_Invariants(value Selector_Order, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), TREE_COUNT_MINIMUM, HUFFMAN_TREE_COUNT_MAXIMUM).
		Ensure()
}

// Move_Order is one move-to-front alphabet.
type Move_Order []byte

// Move_Order_Invariants bounds the largest move-to-front alphabet.
func Move_Order_Invariants(value Move_Order, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SYMBOL_COUNT_MINIMUM, BYTE_VALUE_COUNT).
		Ensure()
}

// Code_Sizes is one Huffman alphabet's code widths.
type Code_Sizes []uint8

// Code_Sizes_Invariants bounds one Huffman alphabet.
func Code_Sizes_Invariants(value Code_Sizes, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), ALPHABET_COUNT_MINIMUM, SYMBOL_COUNT_MAXIMUM).
		Ensure()
}

// Character_Counts is the fixed byte histogram.
type Character_Counts []uint32

// Character_Counts_Invariants fixes the byte histogram width.
func Character_Counts_Invariants(value Character_Counts, namespace invariant.Namespace) {
	invariant.Always(
		len(value) == BYTE_VALUE_COUNT,
		"A character histogram has one slot for every byte.",
	)
}

// Bit_Reader holds bounded input cursor.
type Bit_Reader struct {
	// Source remains caller-owned compressed bytes.
	Source Bit_Source
	// Bits retains unread low-order bzip2 bits.
	Bits Bit_Buffer
	// Bits_Count bounds valid low-order bits.
	Bits_Count Bit_Count
}

// Bit_Reader_Invariants composes bounded reader storage.
func Bit_Reader_Invariants(value Bit_Reader, namespace invariant.Namespace) {
	Bit_Source_Invariants(value.Source, namespace)
	Bit_Buffer_Invariants(value.Bits, namespace)
	Bit_Count_Invariants(value.Bits_Count, namespace)
}

// Block_Compressed is encoded input after a block marker and checksum.
type Block_Compressed []byte

// Block_Compressed_Invariants bounds one encoded block remainder.
func Block_Compressed_Invariants(value Block_Compressed, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), BYTE_COUNT_MINIMUM, BLOCK_SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Symbol_Reader is reader state after the fixed block prefix.
type Symbol_Reader Bit_Reader

// Symbol_Reader_Invariants bounds symbol-bitmap entry state.
func Symbol_Reader_Invariants(value Symbol_Reader, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value.Source), BYTE_COUNT_MINIMUM, SYMBOL_SOURCE_SIZE_MAXIMUM).
		Range_Uint64(uint64(value.Bits), BIT_BUFFER_MINIMUM, BIT_BUFFER_MAXIMUM).
		Ensure()
	invariant.Always(
		value.Bits_Count == SYMBOL_BIT_COUNT,
		"The fixed block prefix leaves seven unread bits.",
	)
}

// Tree_Reader is reader state after one nonempty symbol bitmap.
type Tree_Reader Bit_Reader

// Tree_Reader_Invariants bounds tree-header entry state.
func Tree_Reader_Invariants(value Tree_Reader, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value.Source), BYTE_COUNT_MINIMUM, TREE_SOURCE_SIZE_MAXIMUM).
		Range_Uint64(uint64(value.Bits), BIT_BUFFER_MINIMUM, BIT_BUFFER_MAXIMUM).
		Ensure()
	invariant.Always(
		value.Bits_Count == SYMBOL_BIT_COUNT,
		"Every complete symbol bitmap leaves seven unread bits.",
	)
}

// Selector_List_Reader is reader state before selector unary codes.
type Selector_List_Reader Bit_Reader

// Selector_List_Reader_Invariants bounds selector-list entry state.
func Selector_List_Reader_Invariants(
	value Selector_List_Reader, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(len(value.Source), BYTE_COUNT_MINIMUM, SELECTOR_SOURCE_SIZE_MAXIMUM).
		Range_Uint64(
			uint64(value.Bits), BIT_BUFFER_MINIMUM, SELECTOR_BIT_BUFFER_MAXIMUM,
		).
		Ensure()
	invariant.Always(
		value.Bits_Count == SELECTOR_BIT_COUNT,
		"Tree and selector counts leave five unread bits.",
	)
}

// Selector_Reader is reader state while consuming selector unary codes.
type Selector_Reader Bit_Reader

// Selector_Reader_Invariants bounds one selector-code boundary.
func Selector_Reader_Invariants(value Selector_Reader, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value.Source), BYTE_COUNT_MINIMUM, SELECTOR_SOURCE_SIZE_MAXIMUM).
		Range_Uint64(uint64(value.Bits), BIT_BUFFER_MINIMUM, BIT_BUFFER_MAXIMUM).
		Range_Uint(uint(value.Bits_Count), BIT_COUNT_MINIMUM, BIT_COUNT_MAXIMUM).
		Ensure()
}

// Decoder_Reader is reader state while constructing canonical trees.
type Decoder_Reader Bit_Reader

// Decoder_Reader_Invariants bounds tree-construction state.
func Decoder_Reader_Invariants(value Decoder_Reader, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value.Source), BYTE_COUNT_MINIMUM, SELECTOR_SOURCE_SIZE_MAXIMUM).
		Range_Uint64(uint64(value.Bits), BIT_BUFFER_MINIMUM, BIT_BUFFER_MAXIMUM).
		Range_Uint(uint(value.Bits_Count), BIT_COUNT_MINIMUM, BIT_COUNT_MAXIMUM).
		Ensure()
}

// Payload_Reader is reader state while decoding block symbols.
type Payload_Reader Bit_Reader

// Payload_Reader_Invariants bounds compressed payload state.
func Payload_Reader_Invariants(value Payload_Reader, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value.Source), BYTE_COUNT_MINIMUM, PAYLOAD_SOURCE_SIZE_MAXIMUM).
		Range_Uint64(uint64(value.Bits), BIT_BUFFER_MINIMUM, BIT_BUFFER_MAXIMUM).
		Range_Uint(uint(value.Bits_Count), BIT_COUNT_MINIMUM, BIT_COUNT_MAXIMUM).
		Ensure()
}

// Trailer_Reader is reader state after the final marker.
type Trailer_Reader Bit_Reader

// Trailer_Reader_Invariants bounds stream-checksum state.
func Trailer_Reader_Invariants(value Trailer_Reader, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value.Source), BYTE_COUNT_MINIMUM, TRAILER_SOURCE_SIZE_MAXIMUM).
		Range_Uint64(uint64(value.Bits), BIT_BUFFER_MINIMUM, BIT_BUFFER_MAXIMUM).
		Range_Uint(uint(value.Bits_Count), BIT_COUNT_MINIMUM, BIT_COUNT_MAXIMUM).
		Ensure()
}

// Remainder_Reader is reader state after one decoded block.
type Remainder_Reader Bit_Reader

// Remainder_Reader_Invariants bounds state returned to stream framing.
func Remainder_Reader_Invariants(value Remainder_Reader, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value.Source), BYTE_COUNT_MINIMUM, REMAINDER_SOURCE_SIZE_MAXIMUM,
		).
		Range_Uint64(uint64(value.Bits), BIT_BUFFER_MINIMUM, BIT_BUFFER_MAXIMUM).
		Range_Uint(uint(value.Bits_Count), BIT_COUNT_MINIMUM, BIT_COUNT_MAXIMUM).
		Ensure()
}

// Huffman_Decoder holds canonical code tables in fixed storage.
type Huffman_Decoder struct {
	// Counts groups canonical symbols by bit count.
	Counts [HUFFMAN_CODE_SIZE_MAXIMUM + 1]uint16
	// Symbols stores canonical order without owned slices.
	Symbols [SYMBOL_COUNT_MAXIMUM]uint16
}

// Huffman_Decoder_Invariants bounds the fixed canonical tables.
func Huffman_Decoder_Invariants(
	value *Huffman_Decoder, namespace invariant.Namespace,
) {
	invariant.Always(
		value.Counts[0] <= SYMBOL_COUNT_MAXIMUM,
		"A Huffman decoder counts no more symbols than its fixed table holds.",
	)
}

// Block_State holds bounded scratch arrays outside decode control flow.
type Block_State struct {
	// Character_Count supports inverse BWT.
	Character_Count [BYTE_VALUE_COUNT]uint32
	// Trees holds all selector targets.
	Trees [HUFFMAN_TREE_COUNT_MAXIMUM]Huffman_Decoder
	// Symbols holds present byte alphabet in move-to-front order.
	Symbols [BYTE_VALUE_COUNT]byte
	// Code_Sizes reconstructs one canonical tree at time.
	Code_Sizes [SYMBOL_COUNT_MAXIMUM]uint8
}

// Block_State_Invariants bounds fixed block scratch storage.
func Block_State_Invariants(value *Block_State, namespace invariant.Namespace) {
	invariant.Always(
		len(value.Symbols) == BYTE_VALUE_COUNT,
		"Block scratch has one symbol slot for every byte.",
	)
}

// Tree_Selection owns mutable selector progress.
type Tree_Selection struct {
	// Selector_Index is the next selector position.
	Selector_Index Selector_Index
	// Decoded_Count is symbols decoded with the current tree.
	Decoded_Count Group_Count
	// Current_Tree is the selected tree slot.
	Current_Tree Tree_Index
}

// Tree_Selection_Invariants composes selector progress.
func Tree_Selection_Invariants(value *Tree_Selection, namespace invariant.Namespace) {
	Selector_Index_Invariants(value.Selector_Index, namespace)
	Group_Count_Invariants(value.Decoded_Count, namespace)
	Tree_Index_Invariants(value.Current_Tree, namespace)
}

// Decode_Into requires transform storage because inverse BWT needs one uint32 per block byte.
func Decode_Into(
	destination Destination, transform Transform, compressed Compressed,
) (count Count, status Status) {
	defer func() {
		Count_Invariants(count, "Decode_Into.count")
		Status_Invariants(status, "Decode_Into.status")
	}()
	Destination_Invariants(destination, "Decode_Into.destination")
	Transform_Invariants(transform, "Decode_Into.transform")
	Compressed_Invariants(compressed, "Decode_Into.compressed")
	block_item_count_maximum, header_status := decode_header(compressed, transform)
	if header_status != HEADER_STATUS_OK {
		return 0, Status(header_status)
	}

	reader := Bit_Reader{Source: Bit_Source(compressed[4:])}
	var file_checksum uint32
	for more := true; more; {
		marker, present := bit_reader_read(&reader, 48)
		if !present {
			return count, STATUS_INPUT_INVALID
		}
		switch marker {
		case BLOCK_MAGIC:
			want_block_checksum, checksum_present := bit_reader_read(&reader, 32)
			if !checksum_present {
				return count, STATUS_INPUT_INVALID
			}
			file_checksum = file_checksum<<1 | file_checksum>>31
			file_checksum ^= uint32(want_block_checksum)
			block_count, first, remainder, block_valid := decode_block(
				Block_Compressed(reader.Source),
				Block_Storage(transform[:int(block_item_count_maximum)]),
				block_item_count_maximum,
			)
			if !block_valid {
				return count, STATUS_INPUT_INVALID
			}
			reader = Bit_Reader(remainder)
			var block_checksum Checksum
			var emit_status Emit_Status
			count, block_checksum, emit_status = emit_block(
				destination, count, Block(transform[:block_count]), first,
			)
			if emit_status != EMIT_STATUS_OK {
				return count, Status(emit_status)
			}
			want_checksum := Checksum{
				byte(want_block_checksum >> 24), byte(want_block_checksum >> 16),
				byte(want_block_checksum >> 8), byte(want_block_checksum),
			}
			if block_checksum != want_checksum {
				return count, STATUS_INPUT_INVALID
			}
		case FINAL_MAGIC:
			checksum := Checksum{
				byte(file_checksum >> 24), byte(file_checksum >> 16),
				byte(file_checksum >> 8), byte(file_checksum),
			}
			if !decode_trailer((*Trailer_Reader)(&reader), checksum) {
				return count, STATUS_INPUT_INVALID
			}
			return count, STATUS_OK
		default:
			return count, STATUS_INPUT_INVALID
		}
	}
	return count, STATUS_INPUT_INVALID
}

func decode_header(
	compressed Compressed, transform Transform,
) (block_count Block_Limit, status Header_Status) {
	defer func() {
		Block_Limit_Invariants(block_count, "decode_header.block_count")
		Header_Status_Invariants(status, "decode_header.status")
	}()
	Compressed_Invariants(compressed, "decode_header.compressed")
	Transform_Invariants(transform, "decode_header.transform")
	if len(compressed) < 4 {
		return BLOCK_SIZE_UNIT, HEADER_STATUS_INPUT_INVALID
	}
	magic := uint16(compressed[0])<<8 | uint16(compressed[1])
	if magic != FILE_MAGIC {
		return BLOCK_SIZE_UNIT, HEADER_STATUS_INPUT_INVALID
	}
	entropy := compressed[2]
	if entropy != 'h' {
		return BLOCK_SIZE_UNIT, HEADER_STATUS_INPUT_INVALID
	}
	level := compressed[3]
	if level < '1' {
		return BLOCK_SIZE_UNIT, HEADER_STATUS_INPUT_INVALID
	}
	if level > '9' {
		return BLOCK_SIZE_UNIT, HEADER_STATUS_INPUT_INVALID
	}
	block_count = Block_Limit(int(level-'0') * BLOCK_SIZE_UNIT)
	if len(transform) < int(block_count) {
		return BLOCK_SIZE_UNIT, HEADER_STATUS_WORKSPACE_TOO_SMALL
	}
	return block_count, HEADER_STATUS_OK
}

func decode_trailer(
	reader *Trailer_Reader,
	file_checksum Checksum,
) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "decode_trailer.valid") }()
	Trailer_Reader_Invariants(*reader, "decode_trailer.reader")
	Checksum_Invariants(file_checksum, "decode_trailer.file_checksum")
	want_file_checksum, checksum_present := bit_reader_read((*Bit_Reader)(reader), 32)
	if !checksum_present {
		return false
	}
	want_checksum := Checksum{
		byte(want_file_checksum >> 24), byte(want_file_checksum >> 16),
		byte(want_file_checksum >> 8), byte(want_file_checksum),
	}
	if file_checksum != want_checksum {
		return false
	}
	reader.Bits_Count -= reader.Bits_Count % 8
	if len(reader.Source) != 0 {
		return false
	}
	return true
}

func decode_block(
	compressed Block_Compressed,
	transform Block_Storage,
	block_item_count_maximum Block_Limit,
) (
	block_count Block_Count, first First_Position,
	remainder Remainder_Reader, valid Boolean,
) {
	defer func() {
		Block_Count_Invariants(block_count, "decode_block.block_count")
		First_Position_Invariants(first, "decode_block.first")
		Remainder_Reader_Invariants(remainder, "decode_block.remainder")
		Boolean_Invariants(valid, "decode_block.valid")
	}()
	Block_Compressed_Invariants(compressed, "decode_block.compressed")
	Block_Storage_Invariants(transform, "decode_block.transform")
	Block_Limit_Invariants(block_item_count_maximum, "decode_block.block_item_count_maximum")
	reader := Bit_Reader{Source: Bit_Source(compressed)}
	randomized, available := bit_reader_read(&reader, 1)
	if !available {
		return 0, 0, remainder, false
	}
	if randomized != 0 {
		return 0, 0, remainder, false
	}
	original_position, available := bit_reader_read(&reader, 24)
	if !available {
		return 0, 0, remainder, false
	}
	var state Block_State
	symbol_count, symbols_valid := block_symbols((*Symbol_Reader)(&reader), &state)
	if !symbols_valid {
		return 0, 0, remainder, false
	}
	tree_count, selector_count, selector_reader, trees_valid := block_trees(
		(*Tree_Reader)(&reader), &state, symbol_count,
	)
	if !trees_valid {
		return 0, 0, remainder, false
	}
	block_count, valid = block_payload(
		(*Payload_Reader)(&reader),
		&selector_reader,
		transform,
		block_item_count_maximum,
		&state,
		symbol_count,
		tree_count,
		selector_count,
	)
	if !valid {
		return 0, 0, remainder, false
	}
	if original_position >= Bit_Value(block_count) {
		return 0, 0, remainder, false
	}
	first = inverse_transform(
		Block(transform[:block_count]), First_Position(original_position),
		state.Character_Count[:],
	)
	return block_count, first, Remainder_Reader(reader), true
}

func block_symbols(
	reader *Symbol_Reader, state *Block_State,
) (count Symbol_Count, valid Boolean) {
	defer func() {
		Symbol_Count_Invariants(count, "block_symbols.count")
		Boolean_Invariants(valid, "block_symbols.valid")
	}()
	Symbol_Reader_Invariants(*reader, "block_symbols.reader")
	Block_State_Invariants(state, "block_symbols.state")
	count = SYMBOL_COUNT_MINIMUM
	base_reader := (*Bit_Reader)(reader)
	used_bitmap, available := bit_reader_read(base_reader, 16)
	if !available {
		return SYMBOL_COUNT_MINIMUM, false
	}
	count = 0
	for group := uint64(0); group < 16; group++ {
		if used_bitmap&(1<<(15-group)) == 0 {
			continue
		}
		present_bitmap, present := bit_reader_read(base_reader, 16)
		if !present {
			return SYMBOL_COUNT_MINIMUM, false
		}
		for member := uint64(0); member < 16; member++ {
			if present_bitmap&(1<<(15-member)) != 0 {
				state.Symbols[count] = byte(group*16 + member)
				count++
			}
		}
	}
	if count == 0 {
		return SYMBOL_COUNT_MINIMUM, false
	}
	return count, true
}

func block_trees(
	reader *Tree_Reader,
	state *Block_State,
	symbol_count Symbol_Count,
) (
	tree_count Tree_Count, selector_count Selector_Count,
	selector_reader Selector_List_Reader, valid Boolean,
) {
	defer func() {
		Tree_Count_Invariants(tree_count, "block_trees.tree_count")
		Selector_Count_Invariants(selector_count, "block_trees.selector_count")
		Selector_List_Reader_Invariants(selector_reader, "block_trees.selector_reader")
		Boolean_Invariants(valid, "block_trees.valid")
	}()
	Tree_Reader_Invariants(*reader, "block_trees.reader")
	Block_State_Invariants(state, "block_trees.state")
	Symbol_Count_Invariants(symbol_count, "block_trees.symbol_count")
	tree_count = TREE_COUNT_MINIMUM
	selector_count = SELECTOR_COUNT_MINIMUM
	selector_reader.Bits_Count = SELECTOR_BIT_COUNT
	base_reader := (*Bit_Reader)(reader)
	tree_count_value, available := bit_reader_read(base_reader, 3)
	if !available {
		return tree_count, selector_count, selector_reader, false
	}
	if tree_count_value < 2 {
		return tree_count, selector_count, selector_reader, false
	}
	if tree_count_value > HUFFMAN_TREE_COUNT_MAXIMUM {
		return tree_count, selector_count, selector_reader, false
	}
	tree_count = Tree_Count(tree_count_value)
	selector_count_value, available := bit_reader_read(base_reader, 15)
	if !available {
		return tree_count, selector_count, selector_reader, false
	}
	if selector_count_value == 0 {
		return tree_count, selector_count, selector_reader, false
	}
	selector_count = Selector_Count(selector_count_value)
	selector_reader = Selector_List_Reader(*reader)
	if !selectors_skip((*Selector_List_Reader)(reader), tree_count, selector_count) {
		return tree_count, selector_count, selector_reader, false
	}
	if !block_tree_decoders(
		(*Decoder_Reader)(reader), state, tree_count, Alphabet_Count(symbol_count+2),
	) {
		return tree_count, selector_count, selector_reader, false
	}
	return tree_count, selector_count, selector_reader, true
}

func selectors_skip(
	reader *Selector_List_Reader, tree_count Tree_Count, selector_count Selector_Count,
) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "selectors_skip.valid") }()
	Selector_List_Reader_Invariants(*reader, "selectors_skip.reader")
	Tree_Count_Invariants(tree_count, "selectors_skip.tree_count")
	Selector_Count_Invariants(selector_count, "selectors_skip.selector_count")
	var selector_order [HUFFMAN_TREE_COUNT_MAXIMUM]byte
	move_to_front_range(selector_order[:tree_count])
	for selector_index := 0; selector_index < int(selector_count); selector_index++ {
		_, selector_present := selector_read(
			(*Selector_Reader)(reader), selector_order[:tree_count],
		)
		if !selector_present {
			return false
		}
	}
	return true
}

func block_tree_decoders(
	reader *Decoder_Reader,
	state *Block_State,
	tree_count Tree_Count,
	alphabet_count Alphabet_Count,
) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "block_tree_decoders.valid") }()
	Decoder_Reader_Invariants(*reader, "block_tree_decoders.reader")
	Block_State_Invariants(state, "block_tree_decoders.state")
	Tree_Count_Invariants(tree_count, "block_tree_decoders.tree_count")
	Alphabet_Count_Invariants(alphabet_count, "block_tree_decoders.alphabet_count")
	base_reader := (*Bit_Reader)(reader)
	for tree_index := 0; tree_index < int(tree_count); tree_index++ {
		code_size_value, code_size_present := bit_reader_read(base_reader, 5)
		if !code_size_present {
			return false
		}
		code_size := int(code_size_value)
		for symbol_index := 0; symbol_index < int(alphabet_count); symbol_index++ {
			for more := true; more; {
				if code_size < 1 {
					return false
				}
				if code_size > HUFFMAN_CODE_SIZE_MAXIMUM {
					return false
				}
				changed, change_present := bit_reader_read(base_reader, 1)
				if !change_present {
					return false
				}
				if changed == 0 {
					more = false
					continue
				}
				direction, direction_present := bit_reader_read(base_reader, 1)
				if !direction_present {
					return false
				}
				if direction != 0 {
					code_size--
				} else {
					code_size++
				}
			}
			state.Code_Sizes[symbol_index] = uint8(code_size)
		}
		if !huffman_build(&state.Trees[tree_index], state.Code_Sizes[:alphabet_count]) {
			return false
		}
	}
	return true
}

func block_payload(reader *Payload_Reader, selectors *Selector_List_Reader, transform Block_Storage,
	block_item_count_maximum Block_Limit, state *Block_State,
	symbol_count Symbol_Count, tree_count Tree_Count, selector_count Selector_Count,
) (block_count Block_Count, valid Boolean) {
	defer func() {
		Block_Count_Invariants(block_count, "block_payload.block_count")
		Boolean_Invariants(valid, "block_payload.valid")
	}()
	Payload_Reader_Invariants(*reader, "block_payload.reader")
	Selector_List_Reader_Invariants(*selectors, "block_payload.selector_reader")
	Block_Storage_Invariants(transform, "block_payload.transform")
	Block_Limit_Invariants(block_item_count_maximum, "block_payload.block_item_count_maximum")
	Block_State_Invariants(state, "block_payload.state")
	Symbol_Count_Invariants(symbol_count, "block_payload.symbol_count")
	Tree_Count_Invariants(tree_count, "block_payload.tree_count")
	Selector_Count_Invariants(selector_count, "block_payload.selector_count")
	var selector_order [HUFFMAN_TREE_COUNT_MAXIMUM]byte
	move_to_front_range(selector_order[:tree_count])
	var symbol_order [BYTE_VALUE_COUNT]byte
	copy(symbol_order[:symbol_count], state.Symbols[:symbol_count])
	selection := Tree_Selection{Decoded_Count: SELECTOR_GROUP_SIZE}
	repeat := Repeat_Count(0)
	repeat_power := 0
	clear(state.Character_Count[:])
	selection_reader := (*Selector_Reader)(selectors)
	for more := true; more; {
		if !block_tree_select(selection_reader, selector_order[:tree_count],
			selector_count, &selection) {
			return 0, false
		}
		symbol, symbol_present := huffman_read(reader, &state.Trees[selection.Current_Tree])
		if !symbol_present {
			return 0, false
		}
		selection.Decoded_Count++
		if symbol < 2 {
			if repeat == 0 {
				repeat_power = 1
			}
			repeat += Repeat_Count(repeat_power << symbol)
			repeat_power <<= 1
			if repeat > REPEAT_COUNT_MAXIMUM {
				return 0, false
			}
			continue
		}
		if repeat > 0 {
			block_count, valid = block_repeat(
				transform, block_count, block_item_count_maximum,
				repeat, Byte_Value(symbol_order[0]), state.Character_Count[:],
			)
			if !valid {
				return 0, false
			}
			repeat = 0
		}
		if int(symbol) == int(symbol_count)+1 {
			return block_count, true
		}
		position := Position(symbol) - 1
		if int(position) >= int(symbol_count) {
			return 0, false
		}
		if int(block_count) == int(block_item_count_maximum) {
			return 0, false
		}
		value := move_to_front_decode(symbol_order[:symbol_count], position)
		transform[block_count] = uint32(value)
		state.Character_Count[value]++
		block_count++
	}
	return 0, false
}

func block_tree_select(
	selector_reader *Selector_Reader,
	selector_order Selector_Order,
	selector_count Selector_Count,
	selection *Tree_Selection,
) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "block_tree_select.valid") }()
	Selector_Reader_Invariants(*selector_reader, "block_tree_select.selector_reader")
	Selector_Order_Invariants(selector_order, "block_tree_select.selector_order")
	Selector_Count_Invariants(selector_count, "block_tree_select.selector_count")
	Tree_Selection_Invariants(selection, "block_tree_select.selection")
	if selection.Decoded_Count != SELECTOR_GROUP_SIZE {
		return true
	}
	if selection.Selector_Index == Selector_Index(selector_count) {
		return false
	}
	selected, selector_present := selector_read(selector_reader, selector_order)
	if !selector_present {
		return false
	}
	selection.Current_Tree = Tree_Index(selected)
	selection.Selector_Index++
	selection.Decoded_Count = 0
	return true
}

func block_repeat(
	transform Block_Storage,
	block_count Block_Count,
	block_item_count_maximum Block_Limit,
	repeat_count Repeat_Count,
	value Byte_Value,
	character_count Character_Counts,
) (next_count Block_Count, valid Boolean) {
	defer func() {
		Block_Count_Invariants(next_count, "block_repeat.next_count")
		Boolean_Invariants(valid, "block_repeat.valid")
	}()
	Block_Storage_Invariants(transform, "block_repeat.transform")
	Block_Count_Invariants(block_count, "block_repeat.block_count")
	Block_Limit_Invariants(block_item_count_maximum, "block_repeat.block_item_count_maximum")
	Repeat_Count_Invariants(repeat_count, "block_repeat.repeat_count")
	Byte_Value_Invariants(value, "block_repeat.value")
	Character_Counts_Invariants(character_count, "block_repeat.character_count")
	count_available := int(block_item_count_maximum) - int(block_count)
	if int(repeat_count) > count_available {
		return 0, false
	}
	for repeat_index := 0; repeat_index < int(repeat_count); repeat_index++ {
		transform[block_count] = uint32(value)
		character_count[value]++
		block_count++
	}
	return block_count, true
}

func emit_block(
	destination Destination,
	count Count,
	transform Block,
	position First_Position,
) (next_count Count, checksum Checksum, status Emit_Status) {
	defer func() {
		Count_Invariants(next_count, "emit_block.next_count")
		Checksum_Invariants(checksum, "emit_block.checksum")
		Emit_Status_Invariants(status, "emit_block.status")
	}()
	Destination_Invariants(destination, "emit_block.destination")
	Count_Invariants(count, "emit_block.count")
	Block_Invariants(transform, "emit_block.transform")
	First_Position_Invariants(position, "emit_block.position")
	last := -1
	equal_count := 0
	checksum_word := ^(uint32(checksum[0])<<24 |
		uint32(checksum[1])<<16 | uint32(checksum[2])<<8 | uint32(checksum[3]))
	var checksum_table [BYTE_VALUE_COUNT]uint32
	checksum_table_fill(&checksum_table)
	status = EMIT_STATUS_OK
emission:
	for used_index := 0; used_index < len(transform); used_index++ {
		if uint32(position) >= uint32(len(transform)) {
			status = EMIT_STATUS_INPUT_INVALID
			break
		}
		position = First_Position(transform[position])
		value := byte(position)
		position >>= 8
		if equal_count == 3 {
			for repeat := int(value); repeat > 0; repeat-- {
				if int(count) == len(destination) {
					status = EMIT_STATUS_OUTPUT_TOO_SMALL
					break emission
				}
				destination[count] = byte(last)
				count++
				checksum_index := byte(checksum_word>>24) ^ byte(last)
				checksum_word = checksum_table[checksum_index] ^ checksum_word<<8
			}
			equal_count = 0
			last = -1
			continue
		}
		if last == int(value) {
			equal_count++
		} else {
			equal_count = 0
		}
		last = int(value)
		if int(count) == len(destination) {
			status = EMIT_STATUS_OUTPUT_TOO_SMALL
			break
		}
		destination[count] = value
		count++
		checksum_index := byte(checksum_word>>24) ^ value
		checksum_word = checksum_table[checksum_index] ^ checksum_word<<8
	}
	checksum_word = ^checksum_word
	checksum = Checksum{
		byte(checksum_word >> 24), byte(checksum_word >> 16),
		byte(checksum_word >> 8), byte(checksum_word),
	}
	return count, checksum, status
}

func checksum_table_fill(destination *[BYTE_VALUE_COUNT]uint32) {
	const POLYNOMIAL = 0x04c11db7
	for value := range BYTE_VALUE_COUNT {
		table_value := uint32(value) << 24
		for range 8 {
			if table_value&0x80000000 != 0 {
				table_value = table_value<<1 ^ POLYNOMIAL
			} else {
				table_value <<= 1
			}
		}
		destination[value] = table_value
	}
}

func selector_read(
	reader *Selector_Reader, order Selector_Order,
) (tree Tree_Index, present Boolean) {
	defer func() {
		Tree_Index_Invariants(tree, "selector_read.tree")
		Boolean_Invariants(present, "selector_read.present")
	}()
	Selector_Reader_Invariants(*reader, "selector_read.reader")
	Selector_Order_Invariants(order, "selector_read.order")
	base_reader := (*Bit_Reader)(reader)
	position := Position(0)
	for int(position) < len(order) {
		continued, available := bit_reader_read(base_reader, 1)
		if !available {
			return 0, false
		}
		if continued == 0 {
			return Tree_Index(move_to_front_decode(Move_Order(order), position)), true
		}
		position++
	}
	return 0, false
}

func move_to_front_range(order Selector_Order) {
	Selector_Order_Invariants(order, "move_to_front_range.order")
	for index := range order {
		order[index] = byte(index)
	}
}

func move_to_front_decode(
	order Move_Order, position Position,
) (value Byte_Value) {
	defer func() { Byte_Value_Invariants(value, "move_to_front_decode.value") }()
	Move_Order_Invariants(order, "move_to_front_decode.order")
	Position_Invariants(position, "move_to_front_decode.position")
	value = Byte_Value(order[position])
	copy(order[1:position+1], order[:position])
	order[0] = byte(value)
	return value
}

func huffman_build(
	decoder *Huffman_Decoder, sizes Code_Sizes,
) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "huffman_build.valid") }()
	Huffman_Decoder_Invariants(decoder, "huffman_build.decoder")
	Code_Sizes_Invariants(sizes, "huffman_build.sizes")
	clear(decoder.Counts[:])
	clear(decoder.Symbols[:])
	maximum := 0
	for _, size := range sizes {
		if size == 0 {
			return false
		}
		if size > HUFFMAN_CODE_SIZE_MAXIMUM {
			return false
		}
		decoder.Counts[size]++
		if int(size) > maximum {
			maximum = int(size)
		}
	}
	space := 1
	for size_index := 1; size_index <= maximum; size_index++ {
		space = space*2 - int(decoder.Counts[size_index])
		if space < 0 {
			return false
		}
	}
	if space != 0 {
		return false
	}
	var offsets [HUFFMAN_CODE_SIZE_MAXIMUM + 1]uint16
	for size_index := 1; size_index < HUFFMAN_CODE_SIZE_MAXIMUM; size_index++ {
		offsets[size_index+1] = offsets[size_index] + decoder.Counts[size_index]
	}
	for symbol, size := range sizes {
		position := offsets[size]
		decoder.Symbols[position] = uint16(symbol)
		offsets[size]++
	}
	return true
}

func huffman_read(
	reader *Payload_Reader,
	decoder *Huffman_Decoder,
) (symbol Huffman_Symbol, present Boolean) {
	defer func() {
		Huffman_Symbol_Invariants(symbol, "huffman_read.symbol")
		Boolean_Invariants(present, "huffman_read.present")
	}()
	Payload_Reader_Invariants(*reader, "huffman_read.reader")
	Huffman_Decoder_Invariants(decoder, "huffman_read.decoder")
	base_reader := (*Bit_Reader)(reader)
	var code uint32
	var first uint32
	var symbol_index uint32
	for size_index := 1; size_index <= HUFFMAN_CODE_SIZE_MAXIMUM; size_index++ {
		bit, available := bit_reader_read(base_reader, 1)
		if !available {
			return 0, false
		}
		code = code<<1 | uint32(bit)
		count := uint32(decoder.Counts[size_index])
		if code >= first {
			if code-first < count {
				return Huffman_Symbol(
					decoder.Symbols[symbol_index+code-first]), true
			}
		}
		symbol_index += count
		first = (first + count) << 1
	}
	return 0, false
}

func inverse_transform(
	transform Block,
	original_position First_Position,
	character_count Character_Counts,
) (first First_Position) {
	defer func() { First_Position_Invariants(first, "inverse_transform.first") }()
	Block_Invariants(transform, "inverse_transform.transform")
	First_Position_Invariants(original_position, "inverse_transform.original_position")
	Character_Counts_Invariants(character_count, "inverse_transform.character_count")
	var sum uint32
	for index := range character_count {
		sum += character_count[index]
		character_count[index] = sum - character_count[index]
	}
	for index := range transform {
		value := transform[index] & 0xff
		transform[character_count[value]] |= uint32(index) << 8
		character_count[value]++
	}
	return First_Position(transform[original_position] >> 8)
}

func bit_reader_read(
	reader *Bit_Reader, count Bit_Read_Count,
) (value Bit_Value, available Boolean) {
	defer func() {
		Bit_Value_Invariants(value, "bit_reader_read.value")
		Boolean_Invariants(available, "bit_reader_read.available")
	}()
	Bit_Reader_Invariants(*reader, "bit_reader_read.reader")
	Bit_Read_Count_Invariants(count, "bit_reader_read.count")
	for reader.Bits_Count < Bit_Count(count) {
		if len(reader.Source) == 0 {
			return 0, false
		}
		reader.Bits = reader.Bits<<8 | Bit_Buffer(reader.Source[0])
		reader.Source = reader.Source[1:]
		reader.Bits_Count += 8
	}
	shift := reader.Bits_Count - Bit_Count(count)
	value = Bit_Value(reader.Bits >> shift)
	if count < 64 {
		value &= Bit_Value(1<<count) - 1
	}
	reader.Bits_Count -= Bit_Count(count)
	reader.Bits &= Bit_Buffer(1<<reader.Bits_Count) - 1
	return value, true
}
