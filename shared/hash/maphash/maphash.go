// Package maphash provides an explicitly keyed bounded hash for byte sequences and text.
package maphash

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
)

// TEXT_SIZE_MINIMUM admits an empty bounded text write.
const TEXT_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// TEXT_SIZE_MAXIMUM reuses the same byte boundary because UTF-8 text is hashed as bytes.
const TEXT_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// BITS_PER_BYTE exposes the width used by standard little-endian Sum output.
const BITS_PER_BYTE = binary.BITS_PER_BYTE

// BLOCK_SIZE is the SipHash message-word width.
const BLOCK_SIZE = bits.BIT_COUNT_64_MAXIMUM / BITS_PER_BYTE

// DIGEST_SIZE is one 64-bit value in bytes.
const DIGEST_SIZE = bits.BIT_COUNT_64_MAXIMUM / BITS_PER_BYTE

// TAIL_COUNT_MINIMUM is an empty partial word.
const TAIL_COUNT_MINIMUM = 0

// TAIL_COUNT_MAXIMUM leaves a complete word for immediate compression.
const TAIL_COUNT_MAXIMUM = BLOCK_SIZE - 1

// TAIL_BYTE_MINIMUM admits a zero message byte in any partial-word position.
const TAIL_BYTE_MINIMUM uint8 = bits.WORD_8_MINIMUM

// TAIL_BYTE_MAXIMUM keeps every byte value because message bytes are opaque.
const TAIL_BYTE_MAXIMUM uint8 = bits.WORD_8_MAXIMUM

// TAIL_INDEX_MINIMUM is the lowest little-endian word position.
const TAIL_INDEX_MINIMUM = 0

// TAIL_INDEX_MAXIMUM reaches the top word position because a completed word drains through Tail.
const TAIL_INDEX_MAXIMUM = BLOCK_SIZE - 1

// TOTAL_COUNT_MINIMUM is an empty message.
const TOTAL_COUNT_MINIMUM uint32 = bits.WORD_32_MINIMUM

// TOTAL_COUNT_MAXIMUM bounds one logical message to a 32-bit byte count.
const TOTAL_COUNT_MAXIMUM uint32 = bits.WORD_32_MAXIMUM

// MESSAGE_SIZE_MAXIMUM is default logical message bound.
const MESSAGE_SIZE_MAXIMUM Message_Size_Maximum = Message_Size_Maximum(TOTAL_COUNT_MAXIMUM)

// SIP_INITIAL_0 separates first keyed state lane.
const SIP_INITIAL_0 uint64 = 0x736f6d6570736575

// SIP_INITIAL_1 separates second keyed state lane.
const SIP_INITIAL_1 uint64 = 0x646f72616e646f6d

// SIP_INITIAL_2 separates third keyed state lane.
const SIP_INITIAL_2 uint64 = 0x6c7967656e657261

// SIP_INITIAL_3 separates fourth keyed state lane.
const SIP_INITIAL_3 uint64 = 0x7465646279746573

// SIP_COMPRESSION_ROUNDS is SipHash-2-4's rounds per message word.
const SIP_COMPRESSION_ROUNDS = 2

// SIP_FINALIZATION_ROUNDS is SipHash-2-4's rounds after the final word.
const SIP_FINALIZATION_ROUNDS = 4

// SIP_FINAL_MARKER separates finalization from ordinary message words.
const SIP_FINAL_MARKER uint64 = 0xff

// SIP_ROTATION_0 is first rotation distance fixed by SipHash-2-4.
const SIP_ROTATION_0 = 13

// SIP_ROTATION_1 is second rotation distance fixed by SipHash-2-4.
const SIP_ROTATION_1 = 32

// SIP_ROTATION_2 is third rotation distance fixed by SipHash-2-4.
const SIP_ROTATION_2 = 16

// SIP_ROTATION_3 is fourth rotation distance fixed by SipHash-2-4.
const SIP_ROTATION_3 = 21

// SIP_ROTATION_4 is fifth rotation distance fixed by SipHash-2-4.
const SIP_ROTATION_4 = 17

// SIP_ROTATION_5 is sixth rotation distance fixed by SipHash-2-4.
const SIP_ROTATION_5 = 32

// SOURCE_SIZE_MINIMUM is shared per-call input boundary.
const SOURCE_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// SOURCE_SIZE_MAXIMUM is shared per-call input boundary.
const SOURCE_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// DESTINATION_SIZE_MINIMUM admits short-output status paths.
const DESTINATION_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// DESTINATION_SIZE_MAXIMUM is shared caller-output boundary.
const DESTINATION_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// OUTPUT_COUNT_EMPTY means no partial digest reached caller storage.
const OUTPUT_COUNT_EMPTY Output_Count = 0

// OUTPUT_COUNT_COMPLETE means one complete digest reached caller storage.
const OUTPUT_COUNT_COMPLETE Output_Count = DIGEST_SIZE

// OUTPUT_STATUS_OK means one complete digest reached caller storage.
const OUTPUT_STATUS_OK Output_Status = Output_Status(bits.WORD_8_MINIMUM)

// OUTPUT_STATUS_TOO_SMALL leaves short caller storage untouched.
const OUTPUT_STATUS_TOO_SMALL Output_Status = OUTPUT_STATUS_OK + 1

// WRITE_STATUS_OK means complete source joined logical message.
const WRITE_STATUS_OK Write_Status = Write_Status(bits.WORD_8_MINIMUM)

// WRITE_STATUS_MESSAGE_TOO_LARGE leaves state unchanged.
const WRITE_STATUS_MESSAGE_TOO_LARGE Write_Status = WRITE_STATUS_OK + 1

// COUNT_MINIMUM is empty input.
const COUNT_MINIMUM = SOURCE_SIZE_MINIMUM

// COUNT_MAXIMUM is one complete bounded source.
const COUNT_MAXIMUM = SOURCE_SIZE_MAXIMUM

// Text is one bounded string chunk.
type Text string

// Text_Invariants binds UTF-8 bytes to the same per-call boundary.
func Text_Invariants(value Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Source is one bounded input chunk.
type Source []byte

// Source_Invariants binds one call to repository byte boundary.
func Source_Invariants(value Source, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
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

// Count_Invariants covers accepted source count or zero on refusal.
func Count_Invariants(value Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Write_Status reports logical-message acceptance.
type Write_Status uint8

// Write_Status_Invariants covers success and total-bound refusal.
func Write_Status_Invariants(value Write_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(WRITE_STATUS_OK), uint8(WRITE_STATUS_MESSAGE_TOO_LARGE),
		).
		Ensure()
}

// Write_Output keeps accepted count and status in one invariant chain.
type Write_Output struct {
	// Count reports consumed source bytes.
	Count Count
	// Status classifies same write.
	Status Write_Status
}

// Write_Output_Invariants composes matching count and status domains.
func Write_Output_Invariants(value Write_Output, namespace aver.Namespace) {
	Count_Invariants(value.Count, namespace)
	Write_Status_Invariants(value.Status, namespace)
}

// Output_Count is bytes populated by Hash_Sum_Into.
type Output_Count uint8

// Output_Count_Invariants excludes partial hash output.
func Output_Count_Invariants(value Output_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(OUTPUT_COUNT_EMPTY), uint8(OUTPUT_COUNT_COMPLETE),
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

// Output keeps returned count and status in one invariant chain.
type Output struct {
	// Count reports initialized destination bytes.
	Count Output_Count
	// Status classifies same destination write.
	Status Output_Status
}

// Output_Invariants composes matching count and status domains.
func Output_Invariants(value Output, namespace aver.Namespace) {
	Output_Count_Invariants(value.Count, namespace)
	Output_Status_Invariants(value.Status, namespace)
}

// Byte is one explicit single-byte write.
type Byte uint8

// Byte_Invariants preserves every byte value.
func Byte_Invariants(value Byte, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM).
		Ensure()
}

// Key_0 is the first caller-injected SipHash key word.
type Key_0 uint64

// Key_0_Invariants preserves the full injected key domain.
func Key_0_Invariants(value Key_0, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Key_1 is the second caller-injected SipHash key word.
type Key_1 uint64

// Key_1_Invariants preserves the full injected key domain.
func Key_1_Invariants(value Key_1, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Output_Mask lets one injected seed select an independent final mapping.
type Output_Mask uint64

// Output_Mask_Invariants preserves the full injected mask domain.
func Output_Mask_Invariants(value Output_Mask, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Seed is the complete explicit function identity.
type Seed struct {
	// Key_0 and Key_1 must not both be zero for initialization.
	Key_0 Key_0
	// Key_1 supplies the second half of the SipHash key.
	Key_1 Key_1
	// Output_Mask makes equal keyed state selectable as a distinct final function.
	Output_Mask Output_Mask
}

// Seed_Invariants covers each injected word; initialization separately rejects an unkeyed seed.
func Seed_Invariants(value Seed, namespace aver.Namespace) {
	Key_0_Invariants(value.Key_0, namespace)
	Key_1_Invariants(value.Key_1, namespace)
	Output_Mask_Invariants(value.Output_Mask, namespace)
}

// Tail_Count is partial bytes awaiting one complete SipHash word.
type Tail_Count int

// Tail_Count_Invariants excludes a complete word because it is compressed immediately.
func Tail_Count_Invariants(value Tail_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), TAIL_COUNT_MINIMUM, TAIL_COUNT_MAXIMUM).
		Ensure()
}

// Tail_Byte_0 is the partial-word byte at bits 0 through 7.
type Tail_Byte_0 uint8

// Tail_Byte_0_Invariants keeps the full byte domain because message bytes are opaque.
func Tail_Byte_0_Invariants(value Tail_Byte_0, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), TAIL_BYTE_MINIMUM, TAIL_BYTE_MAXIMUM).
		Ensure()
}

// Tail_Byte_1 is the partial-word byte at bits 8 through 15.
type Tail_Byte_1 uint8

// Tail_Byte_1_Invariants keeps the full byte domain because message bytes are opaque.
func Tail_Byte_1_Invariants(value Tail_Byte_1, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), TAIL_BYTE_MINIMUM, TAIL_BYTE_MAXIMUM).
		Ensure()
}

// Tail_Byte_2 is the partial-word byte at bits 16 through 23.
type Tail_Byte_2 uint8

// Tail_Byte_2_Invariants keeps the full byte domain because message bytes are opaque.
func Tail_Byte_2_Invariants(value Tail_Byte_2, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), TAIL_BYTE_MINIMUM, TAIL_BYTE_MAXIMUM).
		Ensure()
}

// Tail_Byte_3 is the partial-word byte at bits 24 through 31.
type Tail_Byte_3 uint8

// Tail_Byte_3_Invariants keeps the full byte domain because message bytes are opaque.
func Tail_Byte_3_Invariants(value Tail_Byte_3, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), TAIL_BYTE_MINIMUM, TAIL_BYTE_MAXIMUM).
		Ensure()
}

// Tail_Byte_4 is the partial-word byte at bits 32 through 39.
type Tail_Byte_4 uint8

// Tail_Byte_4_Invariants keeps the full byte domain because message bytes are opaque.
func Tail_Byte_4_Invariants(value Tail_Byte_4, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), TAIL_BYTE_MINIMUM, TAIL_BYTE_MAXIMUM).
		Ensure()
}

// Tail_Byte_5 is the partial-word byte at bits 40 through 47.
type Tail_Byte_5 uint8

// Tail_Byte_5_Invariants keeps the full byte domain because message bytes are opaque.
func Tail_Byte_5_Invariants(value Tail_Byte_5, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), TAIL_BYTE_MINIMUM, TAIL_BYTE_MAXIMUM).
		Ensure()
}

// Tail_Byte_6 is the partial-word byte at bits 48 through 55.
type Tail_Byte_6 uint8

// Tail_Byte_6_Invariants keeps the full byte domain because message bytes are opaque.
func Tail_Byte_6_Invariants(value Tail_Byte_6, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), TAIL_BYTE_MINIMUM, TAIL_BYTE_MAXIMUM).
		Ensure()
}

// Tail_Byte_7 is the partial-word byte at bits 56 through 63.
type Tail_Byte_7 uint8

// Tail_Byte_7_Invariants keeps the full byte domain because message bytes are opaque.
func Tail_Byte_7_Invariants(value Tail_Byte_7, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), TAIL_BYTE_MINIMUM, TAIL_BYTE_MAXIMUM).
		Ensure()
}

// Tail_Index selects one little-endian position of the partial word.
type Tail_Index int

// Tail_Index_Invariants reaches the top position because a completed word drains through Tail.
func Tail_Index_Invariants(value Tail_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), TAIL_INDEX_MINIMUM, TAIL_INDEX_MAXIMUM).
		Ensure()
}

// Tail is the partial SipHash message word. One field per position, each with its own defined
// type, because a fixed array field is banned and one coverage tree holds a type one time.
type Tail struct {
	// Byte_0 sits at bits 0 through 7.
	Byte_0 Tail_Byte_0
	// Byte_1 sits at bits 8 through 15.
	Byte_1 Tail_Byte_1
	// Byte_2 sits at bits 16 through 23.
	Byte_2 Tail_Byte_2
	// Byte_3 sits at bits 24 through 31.
	Byte_3 Tail_Byte_3
	// Byte_4 sits at bits 32 through 39.
	Byte_4 Tail_Byte_4
	// Byte_5 sits at bits 40 through 47.
	Byte_5 Tail_Byte_5
	// Byte_6 sits at bits 48 through 55.
	Byte_6 Tail_Byte_6
	// Byte_7 sits at bits 56 through 63.
	Byte_7 Tail_Byte_7
}

// Tail_Invariants composes every position so each byte keeps its own coverage axes.
func Tail_Invariants(value Tail, namespace aver.Namespace) {
	Tail_Byte_0_Invariants(value.Byte_0, namespace)
	Tail_Byte_1_Invariants(value.Byte_1, namespace)
	Tail_Byte_2_Invariants(value.Byte_2, namespace)
	Tail_Byte_3_Invariants(value.Byte_3, namespace)
	Tail_Byte_4_Invariants(value.Byte_4, namespace)
	Tail_Byte_5_Invariants(value.Byte_5, namespace)
	Tail_Byte_6_Invariants(value.Byte_6, namespace)
	Tail_Byte_7_Invariants(value.Byte_7, namespace)
}

// Tail_Handle gives partial-word state one pointer identity.
type Tail_Handle *Tail

// Tail_Handle_Invariants composes present state.
func Tail_Handle_Invariants(value Tail_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Tail_Invariants(*value, namespace)
}

// Message_Size_Maximum is caller-selected logical message bound.
type Message_Size_Maximum uint32

// Message_Size_Maximum_Invariants binds configuration to package limit.
func Message_Size_Maximum_Invariants(value Message_Size_Maximum, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), TOTAL_COUNT_MINIMUM, TOTAL_COUNT_MAXIMUM).
		Ensure()
}

// State_0 is first full-domain SipHash lane.
type State_0 uint64

// State_0_Invariants preserves every first-lane bit pattern.
func State_0_Invariants(value State_0, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// State_1 is second full-domain SipHash lane.
type State_1 uint64

// State_1_Invariants preserves every second-lane bit pattern.
func State_1_Invariants(value State_1, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// State_2 is third full-domain SipHash lane.
type State_2 uint64

// State_2_Invariants preserves every third-lane bit pattern.
func State_2_Invariants(value State_2, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// State_3 is fourth full-domain SipHash lane.
type State_3 uint64

// State_3_Invariants preserves every fourth-lane bit pattern.
func State_3_Invariants(value State_3, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Total_Count is accepted logical-message bytes.
type Total_Count uint32

// Total_Count_Invariants binds accepted bytes to logical-message storage.
func Total_Count_Invariants(value Total_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), TOTAL_COUNT_MINIMUM, TOTAL_COUNT_MAXIMUM).
		Ensure()
}

// READY_EMPTY marks caller storage before initialization.
const READY_EMPTY Ready = false

// READY_COMPLETE marks keyed initialized state.
const READY_COMPLETE Ready = true

// Ready reports whether caller storage contains keyed state.
type Ready bool

// Ready_Invariants covers both lifecycle states.
func Ready_Invariants(value Ready, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Map hash state is initialized.").
		Ensure()
}

// Hash is caller-owned SipHash streaming state.
type Hash struct {
	// Seed is explicit function identity retained across Reset.
	Seed Seed
	// State_0 is first SipHash lane.
	State_0 State_0
	// State_1 is second SipHash lane.
	State_1 State_1
	// State_2 is third SipHash lane.
	State_2 State_2
	// State_3 is fourth SipHash lane.
	State_3 State_3
	// Tail holds partial word bytes until compression. Positions at and past Tail_Count keep
	// stale bytes, because a flush never clears them and no reader looks past Tail_Count.
	Tail Tail
	// Tail_Count is partial bytes currently stored in Tail.
	Tail_Count Tail_Count
	// Total_Count is accepted bytes in current logical message.
	Total_Count Total_Count
	// Message_Size_Maximum is caller-injected logical-message bound.
	Message_Size_Maximum Message_Size_Maximum
	// Ready separates zero caller storage from keyed state.
	Ready Ready
}

// Hash_Invariants covers all bounded and full-width state; runtime entries separately require
// Ready because Hash_Init must accept zero caller storage.
func Hash_Invariants(value Hash, namespace aver.Namespace) {
	Seed_Invariants(value.Seed, namespace)
	State_0_Invariants(value.State_0, namespace)
	State_1_Invariants(value.State_1, namespace)
	State_2_Invariants(value.State_2, namespace)
	State_3_Invariants(value.State_3, namespace)
	Tail_Invariants(value.Tail, namespace)
	Tail_Count_Invariants(value.Tail_Count, namespace)
	Total_Count_Invariants(value.Total_Count, namespace)
	Message_Size_Maximum_Invariants(value.Message_Size_Maximum, namespace)
	Ready_Invariants(value.Ready, namespace)
}

// Hash_Handle gives caller state one pointer identity.
type Hash_Handle *Hash

// Hash_Handle_Invariants composes present state.
func Hash_Handle_Invariants(value Hash_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Hash_Invariants(*value, namespace)
}

// Value is the complete keyed 64-bit result domain.
type Value uint64

// Value_Invariants preserves every keyed result.
func Value_Invariants(value Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Hash_Init applies package logical-message bound.
func Hash_Init(value Hash_Handle, seed Seed) {
	Hash_Handle_Invariants(value, "Hash_Init.value")
	Seed_Invariants(seed, "Hash_Init.seed")
	Hash_Init_Bounded(value, seed, MESSAGE_SIZE_MAXIMUM)
}

// Hash_Init_Bounded injects key and logical-message bound into opaque caller storage.
func Hash_Init_Bounded(value Hash_Handle, seed Seed, maximum Message_Size_Maximum) {
	Hash_Handle_Invariants(value, "Hash_Init_Bounded.value.input")
	Seed_Invariants(seed, "Hash_Init_Bounded.seed")
	Message_Size_Maximum_Invariants(maximum, "Hash_Init_Bounded.maximum")
	key := uint64(seed.Key_0) | uint64(seed.Key_1)
	aver.Always(key != 0, "Hash_Init_Bounded requires at least one nonzero key word.")
	value.Seed = seed
	value.State_0 = State_0(uint64(seed.Key_0) ^ SIP_INITIAL_0)
	value.State_1 = State_1(uint64(seed.Key_1) ^ SIP_INITIAL_1)
	value.State_2 = State_2(uint64(seed.Key_0) ^ SIP_INITIAL_2)
	value.State_3 = State_3(uint64(seed.Key_1) ^ SIP_INITIAL_3)
	value.Tail = Tail{}
	value.Tail_Count = 0
	value.Total_Count = 0
	value.Message_Size_Maximum = maximum
	value.Ready = READY_COMPLETE
	aver.Always(
		value.Ready == READY_COMPLETE, "Hash_Init_Bounded produces keyed state.",
	)
}

// Hash_Write absorbs one bounded source or leaves state unchanged at the logical-message bound.
func Hash_Write(
	value Hash_Handle, source Source,
) (output Write_Output) {
	defer func() { Write_Output_Invariants(output, "Hash_Write.output") }()
	Hash_Handle_Invariants(value, "Hash_Write.value.input")
	Source_Invariants(source, "Hash_Write.source")
	defer func() { Hash_Handle_Invariants(value, "Hash_Write.value.output") }()
	// Empty input cannot touch keyed lanes. Structural storage remains safe to inspect before
	// lifecycle rejection; nonempty input never crosses readiness boundary.
	if len(source) == 0 {
		if value.Ready != READY_COMPLETE {
			hash_write_blocks(value, source)
		}
	}
	aver.Always(value.Ready == READY_COMPLETE, "Hash_Write requires Hash_Init.")
	aver.Always(
		len(source) <= SOURCE_SIZE_MAXIMUM,
		"Hash_Write source stays within source bound.",
	)
	source_size := len(source)
	capacity := uint32(value.Message_Size_Maximum) - uint32(value.Total_Count)
	if uint64(len(source)) > uint64(capacity) {
		return Write_Output{Count: 0, Status: WRITE_STATUS_MESSAGE_TOO_LARGE}
	}
	value.Total_Count += Total_Count(len(source))
	hash_write_blocks(value, source)
	return Write_Output{Count: Count(source_size), Status: WRITE_STATUS_OK}
}

// Complete words move through one helper so Write keeps validation and bound policy visible.
func hash_write_blocks(value Hash_Handle, source Source) {
	Hash_Handle_Invariants(value, "hash_write_blocks.value.input")
	Source_Invariants(source, "hash_write_blocks.source")
	// Partial word first, one position at a time. Tail_Count sits at BLOCK_SIZE only until the
	// word loop drains it, and Hash_Invariants never runs in between.
	for value.Tail_Count != 0 && value.Tail_Count != BLOCK_SIZE && len(source) != 0 {
		set_tail_byte(&value.Tail, Tail_Index(value.Tail_Count), Byte(source[0]))
		value.Tail_Count++
		source = source[1:]
	}
	for value.Tail_Count == BLOCK_SIZE || len(source) >= BLOCK_SIZE {
		var word uint64
		if value.Tail_Count == BLOCK_SIZE {
			for index := range BLOCK_SIZE {
				item := tail_byte_at(value.Tail, Tail_Index(index))
				word |= uint64(item) << (BITS_PER_BYTE * index)
			}
			value.Tail_Count = 0
		} else {
			for index := range BLOCK_SIZE {
				word |= uint64(source[index]) << (BITS_PER_BYTE * index)
			}
			source = source[BLOCK_SIZE:]
		}
		state_0 := uint64(value.State_0)
		state_1 := uint64(value.State_1)
		state_2 := uint64(value.State_2)
		state_3 := uint64(value.State_3) ^ word
		for range SIP_COMPRESSION_ROUNDS {
			state_0 += state_1
			state_1 = state_1<<SIP_ROTATION_0 |
				state_1>>(bits.BIT_COUNT_64_MAXIMUM-SIP_ROTATION_0)
			state_1 ^= state_0
			state_0 = state_0<<SIP_ROTATION_1 |
				state_0>>(bits.BIT_COUNT_64_MAXIMUM-SIP_ROTATION_1)
			state_2 += state_3
			state_3 = state_3<<SIP_ROTATION_2 |
				state_3>>(bits.BIT_COUNT_64_MAXIMUM-SIP_ROTATION_2)
			state_3 ^= state_2
			state_0 += state_3
			state_3 = state_3<<SIP_ROTATION_3 |
				state_3>>(bits.BIT_COUNT_64_MAXIMUM-SIP_ROTATION_3)
			state_3 ^= state_0
			state_2 += state_1
			state_1 = state_1<<SIP_ROTATION_4 |
				state_1>>(bits.BIT_COUNT_64_MAXIMUM-SIP_ROTATION_4)
			state_1 ^= state_2
			state_2 = state_2<<SIP_ROTATION_5 |
				state_2>>(bits.BIT_COUNT_64_MAXIMUM-SIP_ROTATION_5)
		}
		value.State_0 = State_0(state_0 ^ word)
		value.State_1 = State_1(state_1)
		value.State_2 = State_2(state_2)
		value.State_3 = State_3(state_3)
	}
	// Tail is empty whenever bytes remain: the fill loop consumed the source or completed the
	// word the block loop drained.
	if len(source) != 0 {
		for index := range len(source) {
			set_tail_byte(&value.Tail, Tail_Index(index), Byte(source[index]))
		}
		value.Tail_Count = Tail_Count(len(source))
	}
	Hash_Handle_Invariants(value, "hash_write_blocks.value.output")
}

// One read over Tail positions, so Sum and the word flush share it.
func tail_byte_at(value Tail, index Tail_Index) (item Byte) {
	defer func() { Byte_Invariants(item, "tail_byte_at.item") }()
	Tail_Invariants(value, "tail_byte_at.value")
	Tail_Index_Invariants(index, "tail_byte_at.index")
	switch index {
	case 0:
		item = Byte(value.Byte_0)
	case 1:
		item = Byte(value.Byte_1)
	case 2:
		item = Byte(value.Byte_2)
	case 3:
		item = Byte(value.Byte_3)
	case 4:
		item = Byte(value.Byte_4)
	case 5:
		item = Byte(value.Byte_5)
	case 6:
		item = Byte(value.Byte_6)
	case 7:
		item = Byte(value.Byte_7)
	}
	return item
}

// One write over Tail positions, so the partial-word fill stays byte-wise.
func set_tail_byte(value Tail_Handle, index Tail_Index, item Byte) {
	Tail_Handle_Invariants(value, "set_tail_byte.value")
	Tail_Index_Invariants(index, "set_tail_byte.index")
	Byte_Invariants(item, "set_tail_byte.item")
	switch index {
	case 0:
		value.Byte_0 = Tail_Byte_0(item)
	case 1:
		value.Byte_1 = Tail_Byte_1(item)
	case 2:
		value.Byte_2 = Tail_Byte_2(item)
	case 3:
		value.Byte_3 = Tail_Byte_3(item)
	case 4:
		value.Byte_4 = Tail_Byte_4(item)
	case 5:
		value.Byte_5 = Tail_Byte_5(item)
	case 6:
		value.Byte_6 = Tail_Byte_6(item)
	case 7:
		value.Byte_7 = Tail_Byte_7(item)
	}
}

// Hash_Write_Text uses bounded stack conversion so string support does not allocate.
func Hash_Write_Text(
	value Hash_Handle, text Text,
) (output Write_Output) {
	defer func() { Write_Output_Invariants(output, "Hash_Write_Text.output") }()
	Hash_Handle_Invariants(value, "Hash_Write_Text.value")
	Text_Invariants(text, "Hash_Write_Text.text")
	aver.Always(value.Ready == READY_COMPLETE, "Hash_Write_Text requires Hash_Init.")
	aver.Always(
		len(text) <= TEXT_SIZE_MAXIMUM,
		"Hash_Write_Text text stays within text bound.",
	)
	var source [TEXT_SIZE_MAXIMUM]byte
	copy(source[:], text)
	return Hash_Write(value, source[:len(text)])
}

// Hash_Write_Byte avoids caller slice construction while preserving total bound status.
func Hash_Write_Byte(value Hash_Handle, item Byte) (status Write_Status) {
	defer func() { Write_Status_Invariants(status, "Hash_Write_Byte.status") }()
	Hash_Handle_Invariants(value, "Hash_Write_Byte.value")
	Byte_Invariants(item, "Hash_Write_Byte.item")
	aver.Always(value.Ready == READY_COMPLETE, "Hash_Write_Byte requires Hash_Init.")
	source := [binary.UINT_8_SIZE]byte{byte(item)}
	return Hash_Write(value, source[:]).Status
}

// Hash_Sum_64 finalizes a copy of state so more bytes may follow.
func Hash_Sum_64(value Hash_Handle) (result Value) {
	defer func() { Value_Invariants(result, "Hash_Sum_64.result") }()
	Hash_Handle_Invariants(value, "Hash_Sum_64.value")
	aver.Always(value.Ready == READY_COMPLETE, "Hash_Sum_64 requires Hash_Init.")
	state_0 := uint64(value.State_0)
	state_1 := uint64(value.State_1)
	state_2 := uint64(value.State_2)
	state_3 := uint64(value.State_3)
	word := uint64(value.Total_Count) << (bits.BIT_COUNT_64_MAXIMUM - BITS_PER_BYTE)
	for index := range int(value.Tail_Count) {
		item := tail_byte_at(value.Tail, Tail_Index(index))
		word |= uint64(item) << (BITS_PER_BYTE * index)
	}
	state_3 ^= word
	for range SIP_COMPRESSION_ROUNDS {
		state_0 += state_1
		state_1 = state_1<<SIP_ROTATION_0 |
			state_1>>(bits.BIT_COUNT_64_MAXIMUM-SIP_ROTATION_0)
		state_1 ^= state_0
		state_0 = state_0<<SIP_ROTATION_1 |
			state_0>>(bits.BIT_COUNT_64_MAXIMUM-SIP_ROTATION_1)
		state_2 += state_3
		state_3 = state_3<<SIP_ROTATION_2 |
			state_3>>(bits.BIT_COUNT_64_MAXIMUM-SIP_ROTATION_2)
		state_3 ^= state_2
		state_0 += state_3
		state_3 = state_3<<SIP_ROTATION_3 |
			state_3>>(bits.BIT_COUNT_64_MAXIMUM-SIP_ROTATION_3)
		state_3 ^= state_0
		state_2 += state_1
		state_1 = state_1<<SIP_ROTATION_4 |
			state_1>>(bits.BIT_COUNT_64_MAXIMUM-SIP_ROTATION_4)
		state_1 ^= state_2
		state_2 = state_2<<SIP_ROTATION_5 |
			state_2>>(bits.BIT_COUNT_64_MAXIMUM-SIP_ROTATION_5)
	}
	state_0 ^= word
	state_2 ^= SIP_FINAL_MARKER
	for range SIP_FINALIZATION_ROUNDS {
		state_0 += state_1
		state_1 = state_1<<SIP_ROTATION_0 |
			state_1>>(bits.BIT_COUNT_64_MAXIMUM-SIP_ROTATION_0)
		state_1 ^= state_0
		state_0 = state_0<<SIP_ROTATION_1 |
			state_0>>(bits.BIT_COUNT_64_MAXIMUM-SIP_ROTATION_1)
		state_2 += state_3
		state_3 = state_3<<SIP_ROTATION_2 |
			state_3>>(bits.BIT_COUNT_64_MAXIMUM-SIP_ROTATION_2)
		state_3 ^= state_2
		state_0 += state_3
		state_3 = state_3<<SIP_ROTATION_3 |
			state_3>>(bits.BIT_COUNT_64_MAXIMUM-SIP_ROTATION_3)
		state_3 ^= state_0
		state_2 += state_1
		state_1 = state_1<<SIP_ROTATION_4 |
			state_1>>(bits.BIT_COUNT_64_MAXIMUM-SIP_ROTATION_4)
		state_1 ^= state_2
		state_2 = state_2<<SIP_ROTATION_5 |
			state_2>>(bits.BIT_COUNT_64_MAXIMUM-SIP_ROTATION_5)
	}
	return Value(state_0 ^ state_1 ^ state_2 ^ state_3 ^ uint64(value.Seed.Output_Mask))
}

// Hash_Sum_Into writes the standard little-endian maphash value.
func Hash_Sum_Into(
	value Hash_Handle, destination Destination,
) (output Output) {
	defer func() { Output_Invariants(output, "Hash_Sum_Into.output") }()
	Hash_Handle_Invariants(value, "Hash_Sum_Into.value")
	Destination_Invariants(destination, "Hash_Sum_Into.destination")
	aver.Always(value.Ready == READY_COMPLETE, "Hash_Sum_Into requires Hash_Init.")
	aver.Always(
		len(destination) <= DESTINATION_SIZE_MAXIMUM,
		"Hash_Sum_Into destination stays within destination bound.",
	)
	if len(destination) < DIGEST_SIZE {
		return Output{Count: OUTPUT_COUNT_EMPTY, Status: OUTPUT_STATUS_TOO_SMALL}
	}
	result := Hash_Sum_64(value)
	for index := range DIGEST_SIZE {
		destination[index] = byte(uint64(result) >> (BITS_PER_BYTE * index))
	}
	return Output{Count: OUTPUT_COUNT_COMPLETE, Status: OUTPUT_STATUS_OK}
}

// Hash_Seed returns explicit function identity without exposing ambient process state.
func Hash_Seed(value Hash_Handle) (seed Seed) {
	defer func() { Seed_Invariants(seed, "Hash_Seed.seed") }()
	Hash_Handle_Invariants(value, "Hash_Seed.value")
	aver.Always(value.Ready == READY_COMPLETE, "Hash_Seed requires Hash_Init.")
	return value.Seed
}

// Hash_Message_Size_Maximum returns injected logical-message bound.
func Hash_Message_Size_Maximum(value Hash_Handle) (maximum Message_Size_Maximum) {
	defer func() {
		Message_Size_Maximum_Invariants(maximum, "Hash_Message_Size_Maximum.maximum")
	}()
	Hash_Handle_Invariants(value, "Hash_Message_Size_Maximum.value")
	aver.Always(
		value.Ready == READY_COMPLETE,
		"Hash_Message_Size_Maximum requires Hash_Init.",
	)
	return value.Message_Size_Maximum
}

// Hash_Reset discards bytes while retaining explicit Seed.
func Hash_Reset(value Hash_Handle) {
	Hash_Handle_Invariants(value, "Hash_Reset.value.input")
	aver.Always(value.Ready == READY_COMPLETE, "Hash_Reset requires Hash_Init.")
	seed := value.Seed
	maximum := value.Message_Size_Maximum
	Hash_Init_Bounded(value, seed, maximum)
}

// Hash_Set_Seed discards bytes and installs a different explicit function identity.
func Hash_Set_Seed(value Hash_Handle, seed Seed) {
	Hash_Handle_Invariants(value, "Hash_Set_Seed.value.input")
	Seed_Invariants(seed, "Hash_Set_Seed.seed")
	aver.Always(value.Ready == READY_COMPLETE, "Hash_Set_Seed requires Hash_Init.")
	key := uint64(seed.Key_0) | uint64(seed.Key_1)
	aver.Always(key != 0, "Hash_Set_Seed requires at least one nonzero key word.")
	Hash_Init_Bounded(value, seed, value.Message_Size_Maximum)
}

// Hash_Clone_Into keeps source and result in caller storage.
func Hash_Clone_Into(destination Hash_Handle, source Hash_Handle) {
	Hash_Handle_Invariants(destination, "Hash_Clone_Into.destination.input")
	Hash_Handle_Invariants(source, "Hash_Clone_Into.source")
	defer func() {
		Hash_Handle_Invariants(destination, "Hash_Clone_Into.destination.output")
	}()
	aver.Always(
		source.Ready == READY_COMPLETE,
		"Hash_Clone_Into requires initialized source.",
	)
	*destination = *source
}

// Bytes hashes one bounded source under explicit Seed.
func Bytes(seed Seed, source Source) (result Value) {
	defer func() { Value_Invariants(result, "Bytes.result") }()
	Seed_Invariants(seed, "Bytes.seed")
	Source_Invariants(source, "Bytes.source")
	key := uint64(seed.Key_0) | uint64(seed.Key_1)
	aver.Always(key != 0, "Bytes requires at least one nonzero key word.")
	aver.Always(
		len(source) <= SOURCE_SIZE_MAXIMUM,
		"Bytes source stays within source bound.",
	)
	var value Hash
	Hash_Init(&value, seed)
	output := Hash_Write(&value, source)
	aver.Always(
		output.Count == Count(len(source)), "Bytes consumes complete bounded source.",
	)
	aver.Always(
		output.Status == WRITE_STATUS_OK, "Bytes stays inside fresh message bound.",
	)
	return Hash_Sum_64(&value)
}

// String hashes one bounded text value without heap conversion.
func String(seed Seed, text Text) (result Value) {
	defer func() { Value_Invariants(result, "String.result") }()
	Seed_Invariants(seed, "String.seed")
	Text_Invariants(text, "String.text")
	key := uint64(seed.Key_0) | uint64(seed.Key_1)
	aver.Always(key != 0, "String requires at least one nonzero key word.")
	aver.Always(
		len(text) <= TEXT_SIZE_MAXIMUM,
		"String text stays within text bound.",
	)
	var value Hash
	Hash_Init(&value, seed)
	output := Hash_Write_Text(&value, text)
	aver.Always(
		output.Count == Count(len(text)), "String consumes complete bounded text.",
	)
	aver.Always(
		output.Status == WRITE_STATUS_OK, "String stays inside fresh message bound.",
	)
	return Hash_Sum_64(&value)
}
