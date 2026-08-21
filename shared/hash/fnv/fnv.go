// Package fnv computes FNV-1 and FNV-1a at 32, 64, and 128 bits with bounded caller-owned state
// and output.
package fnv

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
)

// DIGEST_32_SIZE is one 32-bit value in bytes.
const DIGEST_32_SIZE = bits.BIT_COUNT_32_MAXIMUM / binary.BITS_PER_BYTE

// DIGEST_64_SIZE is one 64-bit value in bytes.
const DIGEST_64_SIZE = bits.BIT_COUNT_64_MAXIMUM / binary.BITS_PER_BYTE

// DIGEST_128_SIZE is two 64-bit words.
const DIGEST_128_SIZE = DIGEST_64_SIZE * 2

// BLOCK_SIZE preserves the standard byte-grain streaming contract.
const BLOCK_SIZE = binary.UINT_8_SIZE

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

// OUTPUT_32_COUNT_EMPTY means no partial 32-bit value reached caller storage.
const OUTPUT_32_COUNT_EMPTY Output_32_Count = 0

// OUTPUT_32_COUNT_COMPLETE means one complete 32-bit value reached caller storage.
const OUTPUT_32_COUNT_COMPLETE Output_32_Count = DIGEST_32_SIZE

// OUTPUT_64_COUNT_EMPTY means no partial 64-bit value reached caller storage.
const OUTPUT_64_COUNT_EMPTY Output_64_Count = 0

// OUTPUT_64_COUNT_COMPLETE means one complete 64-bit value reached caller storage.
const OUTPUT_64_COUNT_COMPLETE Output_64_Count = DIGEST_64_SIZE

// OUTPUT_128_COUNT_EMPTY means no partial 128-bit value reached caller storage.
const OUTPUT_128_COUNT_EMPTY Output_128_Count = 0

// OUTPUT_128_COUNT_COMPLETE means one complete 128-bit value reached caller storage.
const OUTPUT_128_COUNT_COMPLETE Output_128_Count = DIGEST_128_SIZE

// OUTPUT_STATUS_OK means one complete value reached caller storage.
const OUTPUT_STATUS_OK Output_Status = Output_Status(bits.WORD_8_MINIMUM)

// OUTPUT_STATUS_TOO_SMALL leaves short caller storage untouched.
const OUTPUT_STATUS_TOO_SMALL Output_Status = OUTPUT_STATUS_OK + 1

// STATE_IDENTITY_SIZE is every standard FNV state prefix width.
const STATE_IDENTITY_SIZE = len("fnv\x01")

// STATE_VALUE_POSITION follows state identity.
const STATE_VALUE_POSITION = STATE_IDENTITY_SIZE

// STATE_128_LOW_POSITION follows the high state word.
const STATE_128_LOW_POSITION = STATE_VALUE_POSITION + DIGEST_64_SIZE

// STATE_32_SIZE holds identity and one 32-bit state.
const STATE_32_SIZE = STATE_VALUE_POSITION + DIGEST_32_SIZE

// STATE_64_SIZE holds identity and one 64-bit state.
const STATE_64_SIZE = STATE_VALUE_POSITION + DIGEST_64_SIZE

// STATE_128_SIZE holds identity and one 128-bit state.
const STATE_128_SIZE = STATE_VALUE_POSITION + DIGEST_128_SIZE

// STATE_32_1_IDENTITY selects FNV-1 at 32 bits.
const STATE_32_1_IDENTITY State_Identity = "fnv\x01"

// STATE_32_1A_IDENTITY selects FNV-1a at 32 bits.
const STATE_32_1A_IDENTITY State_Identity = "fnv\x02"

// STATE_64_1_IDENTITY selects FNV-1 at 64 bits.
const STATE_64_1_IDENTITY State_Identity = "fnv\x03"

// STATE_64_1A_IDENTITY selects FNV-1a at 64 bits.
const STATE_64_1A_IDENTITY State_Identity = "fnv\x04"

// STATE_128_1_IDENTITY selects FNV-1 at 128 bits.
const STATE_128_1_IDENTITY State_Identity = "fnv\x05"

// STATE_128_1A_IDENTITY selects FNV-1a at 128 bits.
const STATE_128_1A_IDENTITY State_Identity = "fnv\x06"

// OFFSET_32 is the standard 32-bit offset basis.
const OFFSET_32 Value_32 = 2166136261

// OFFSET_64 is the standard 64-bit offset basis.
const OFFSET_64 Value_64 = 14695981039346656037

// OFFSET_128_LOW is the low word of the standard 128-bit offset basis.
const OFFSET_128_LOW Low = 0x62b821756295c58d

// OFFSET_128_HIGH is the high word of the standard 128-bit offset basis.
const OFFSET_128_HIGH High = 0x6c62272e07bb0142

// PRIME_32 is the standard 32-bit FNV prime.
const PRIME_32 Value_32 = 16777619

// PRIME_64 is the standard 64-bit FNV prime.
const PRIME_64 Value_64 = 1099511628211

// PRIME_128_LOW is the low word of the standard 128-bit FNV prime.
const PRIME_128_LOW uint64 = 0x13b

// PRIME_128_SHIFT is the single high bit position of the standard 128-bit FNV prime.
const PRIME_128_SHIFT = bits.BIT_COUNT_32_MAXIMUM - binary.BITS_PER_BYTE

// KIND_1 multiplies state before folding each byte.
const KIND_1 Kind = Kind(bits.WORD_8_MINIMUM)

// KIND_1A folds each byte before multiplying state.
const KIND_1A Kind = KIND_1 + 1

// STATE_OUTPUT_STATUS_OK means complete state reached caller storage.
const STATE_OUTPUT_STATUS_OK State_Output_Status = State_Output_Status(bits.WORD_8_MINIMUM)

// STATE_OUTPUT_STATUS_TOO_SMALL means caller storage cannot hold standard state.
const STATE_OUTPUT_STATUS_TOO_SMALL State_Output_Status = STATE_OUTPUT_STATUS_OK + 1

// STATE_INPUT_STATUS_OK means validated state replaced current state.
const STATE_INPUT_STATUS_OK State_Input_Status = State_Input_Status(bits.WORD_8_MINIMUM)

// STATE_INPUT_STATUS_SIZE_INVALID rejects trailing or missing bytes.
const STATE_INPUT_STATUS_SIZE_INVALID State_Input_Status = STATE_INPUT_STATUS_OK + 1

// STATE_INPUT_STATUS_IDENTIFIER_INVALID rejects another width, kind, or version.
const STATE_INPUT_STATUS_IDENTIFIER_INVALID State_Input_Status = STATE_INPUT_STATUS_SIZE_INVALID + 1

// STATE_32_COUNT_EMPTY means no partial 32-bit state reached caller storage.
const STATE_32_COUNT_EMPTY State_32_Count = 0

// STATE_32_COUNT_COMPLETE means one complete 32-bit state reached caller storage.
const STATE_32_COUNT_COMPLETE State_32_Count = State_32_Count(STATE_32_SIZE)

// STATE_64_COUNT_EMPTY means no partial 64-bit state reached caller storage.
const STATE_64_COUNT_EMPTY State_64_Count = 0

// STATE_64_COUNT_COMPLETE means one complete 64-bit state reached caller storage.
const STATE_64_COUNT_COMPLETE State_64_Count = State_64_Count(STATE_64_SIZE)

// STATE_128_COUNT_EMPTY means no partial 128-bit state reached caller storage.
const STATE_128_COUNT_EMPTY State_128_Count = 0

// STATE_128_COUNT_COMPLETE means one complete 128-bit state reached caller storage.
const STATE_128_COUNT_COMPLETE State_128_Count = State_128_Count(STATE_128_SIZE)

// State_Identity is one standard width-and-kind state prefix.
type State_Identity string

// State_Identity_Invariants fixes the standard prefix width.
func State_Identity_Invariants(value State_Identity, _ invariant.Namespace) {
	invariant.Always(
		len(value) == STATE_IDENTITY_SIZE,
		"FNV state identity has standard width.",
	)
}

// Kind selects byte folding order.
type Kind uint8

// Kind_Invariants admits exactly FNV-1 and FNV-1a.
func Kind_Invariants(value Kind, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(KIND_1), uint8(KIND_1A)).
		Ensure()
}

// Source is one bounded input chunk.
type Source []byte

// Source_Invariants binds one call to repository byte boundary.
func Source_Invariants(value Source, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
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

// Output_32_Count is empty or one complete 32-bit value.
type Output_32_Count uint8

// Output_32_Count_Invariants excludes partial output.
func Output_32_Count_Invariants(value Output_32_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(OUTPUT_32_COUNT_EMPTY), uint8(OUTPUT_32_COUNT_COMPLETE),
		).
		Ensure()
}

// Output_64_Count is empty or one complete 64-bit value.
type Output_64_Count uint8

// Output_64_Count_Invariants excludes partial output.
func Output_64_Count_Invariants(value Output_64_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(OUTPUT_64_COUNT_EMPTY), uint8(OUTPUT_64_COUNT_COMPLETE),
		).
		Ensure()
}

// Output_128_Count is empty or one complete 128-bit value.
type Output_128_Count uint8

// Output_128_Count_Invariants excludes partial output.
func Output_128_Count_Invariants(value Output_128_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(OUTPUT_128_COUNT_EMPTY),
			uint8(OUTPUT_128_COUNT_COMPLETE),
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

// Value_32 is complete 32-bit FNV state.
type Value_32 uint32

// Value_32_Invariants preserves every possible state.
func Value_32_Invariants(value Value_32, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, bits.WORD_32_MAXIMUM).
		Ensure()
}

// Value_64 is complete 64-bit FNV state.
type Value_64 uint64

// Value_64_Invariants preserves every possible state.
func Value_64_Invariants(value Value_64, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// High is the high word of complete 128-bit FNV state.
type High uint64

// High_Invariants preserves every possible high word.
func High_Invariants(value High, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Low is the low word of complete 128-bit FNV state.
type Low uint64

// Low_Invariants preserves every possible low word.
func Low_Invariants(value Low, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Value_128 is complete 128-bit FNV state.
type Value_128 struct {
	// High holds the most-significant word.
	High High
	// Low holds the least-significant word.
	Low Low
}

// Value_128_Invariants composes both independently full-width words.
func Value_128_Invariants(value Value_128, namespace invariant.Namespace) {
	High_Invariants(value.High, namespace)
	Low_Invariants(value.Low, namespace)
}

// READY_EMPTY marks caller storage before initialization.
const READY_EMPTY Ready = 0

// READY_COMPLETE marks state established by width-specific Init.
const READY_COMPLETE Ready = READY_EMPTY + 1

// Ready separates zero caller storage from valid FNV-1 state whose Kind is zero.
type Ready uint8

// Ready_Invariants admits zero storage and initialized state.
func Ready_Invariants(value Ready, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(READY_EMPTY), uint8(READY_COMPLETE)).
		Ensure()
}

// Digest_32 is caller-owned 32-bit streaming state.
type Digest_32 struct {
	// Kind selects FNV-1 or FNV-1a.
	Kind Kind
	// Value is current state.
	Value Value_32
	// Ready separates zero storage from valid FNV-1 state whose Kind is zero.
	Ready Ready
}

// Digest_32_Invariants composes algorithm identity with current state.
func Digest_32_Invariants(value Digest_32, namespace invariant.Namespace) {
	Kind_Invariants(value.Kind, namespace)
	Value_32_Invariants(value.Value, namespace)
	Ready_Invariants(value.Ready, namespace)
}

// Digest_64 is caller-owned 64-bit streaming state.
type Digest_64 struct {
	// Kind selects FNV-1 or FNV-1a.
	Kind Kind
	// Value is current state.
	Value Value_64
	// Ready separates zero storage from valid FNV-1 state whose Kind is zero.
	Ready Ready
}

// Digest_64_Invariants composes algorithm identity with current state.
func Digest_64_Invariants(value Digest_64, namespace invariant.Namespace) {
	Kind_Invariants(value.Kind, namespace)
	Value_64_Invariants(value.Value, namespace)
	Ready_Invariants(value.Ready, namespace)
}

// Digest_128 is caller-owned 128-bit streaming state.
type Digest_128 struct {
	// Kind selects FNV-1 or FNV-1a.
	Kind Kind
	// Value is current state.
	Value Value_128
	// Ready separates zero storage from valid FNV-1 state whose Kind is zero.
	Ready Ready
}

// Digest_128_Invariants composes algorithm identity with current state.
func Digest_128_Invariants(value Digest_128, namespace invariant.Namespace) {
	Kind_Invariants(value.Kind, namespace)
	Value_128_Invariants(value.Value, namespace)
	Ready_Invariants(value.Ready, namespace)
}

// State_32_Count is either no state or one complete 32-bit state.
type State_32_Count uint8

// State_32_Count_Invariants excludes partial state.
func State_32_Count_Invariants(value State_32_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATE_32_COUNT_EMPTY), uint8(STATE_32_COUNT_COMPLETE),
		).
		Ensure()
}

// State_64_Count is either no state or one complete 64-bit state.
type State_64_Count uint8

// State_64_Count_Invariants excludes partial state.
func State_64_Count_Invariants(value State_64_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATE_64_COUNT_EMPTY), uint8(STATE_64_COUNT_COMPLETE),
		).
		Ensure()
}

// State_128_Count is either no state or one complete 128-bit state.
type State_128_Count uint8

// State_128_Count_Invariants excludes partial state.
func State_128_Count_Invariants(value State_128_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATE_128_COUNT_EMPTY), uint8(STATE_128_COUNT_COMPLETE),
		).
		Ensure()
}

// State_Output_Status reports caller state-output capacity.
type State_Output_Status uint8

// State_Output_Status_Invariants covers complete and short output.
func State_Output_Status_Invariants(value State_Output_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATE_OUTPUT_STATUS_OK),
			uint8(STATE_OUTPUT_STATUS_TOO_SMALL),
		).
		Ensure()
}

// State_Input_Status reports hostile state validation.
type State_Input_Status uint8

// State_Input_Status_Invariants covers every state rejection.
func State_Input_Status_Invariants(value State_Input_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATE_INPUT_STATUS_OK),
			uint8(STATE_INPUT_STATUS_SIZE_INVALID),
			uint8(STATE_INPUT_STATUS_IDENTIFIER_INVALID),
		).
		Ensure()
}

// Digest_32_Init establishes width-specific offset basis and caller-selected Kind.
func Digest_32_Init(digest *Digest_32, kind Kind) {
	Digest_32_Invariants(*digest, "Digest_32_Init.digest.input")
	Kind_Invariants(kind, "Digest_32_Init.kind")
	kind_require(kind)
	digest.Kind = kind
	digest.Value = OFFSET_32
	digest.Ready = READY_COMPLETE
	invariant.Always(
		digest.Kind == kind,
		"Initialized FNV-32 retains caller Kind.",
	)
	invariant.Always(
		digest.Value == OFFSET_32,
		"Initialized FNV-32 uses offset basis.",
	)
}

// Digest_32_Reset retains Kind while discarding prior bytes.
func Digest_32_Reset(digest *Digest_32) {
	Digest_32_Invariants(*digest, "Digest_32_Reset.digest.input")
	digest_32_require(digest)
	digest.Value = OFFSET_32
	invariant.Always(
		digest.Value == OFFSET_32,
		"Reset FNV-32 uses offset basis.",
	)
	Kind_Invariants(digest.Kind, "Digest_32_Reset.digest.kind.output")
}

// Digest_32_Write consumes one bounded source completely.
func Digest_32_Write(digest *Digest_32, source Source) (count Count) {
	defer func() { Count_Invariants(count, "Digest_32_Write.count") }()
	Digest_32_Invariants(*digest, "Digest_32_Write.digest.input")
	Source_Invariants(source, "Digest_32_Write.source")
	defer func() { Digest_32_Invariants(*digest, "Digest_32_Write.digest.output") }()
	digest_32_require(digest)
	if len(source) > SOURCE_SIZE_MAXIMUM {
		panic("fnv: source exceeds bound")
	}
	value := digest.Value
	for _, item := range source {
		if digest.Kind == KIND_1A {
			value ^= Value_32(item)
		}
		value *= PRIME_32
		if digest.Kind == KIND_1 {
			value ^= Value_32(item)
		}
	}
	digest.Value = value
	return Count(len(source))
}

// Digest_32_Sum observes state without consuming it.
func Digest_32_Sum(digest *Digest_32) (value Value_32) {
	defer func() { Value_32_Invariants(value, "Digest_32_Sum.value") }()
	Digest_32_Invariants(*digest, "Digest_32_Sum.digest")
	digest_32_require(digest)
	return digest.Value
}

// Digest_32_Sum_Into writes a complete big-endian value or leaves short storage untouched.
func Digest_32_Sum_Into(
	digest *Digest_32, destination Destination,
) (count Output_32_Count, status Output_Status) {
	defer func() {
		Output_32_Count_Invariants(count, "Digest_32_Sum_Into.count")
		Output_Status_Invariants(status, "Digest_32_Sum_Into.status")
	}()
	Digest_32_Invariants(*digest, "Digest_32_Sum_Into.digest")
	Destination_Invariants(destination, "Digest_32_Sum_Into.destination")
	digest_32_require(digest)
	if len(destination) > DESTINATION_SIZE_MAXIMUM {
		panic("fnv: destination exceeds bound")
	}
	if len(destination) < DIGEST_32_SIZE {
		return OUTPUT_32_COUNT_EMPTY, OUTPUT_STATUS_TOO_SMALL
	}
	value := uint32(digest.Value)
	for index := range DIGEST_32_SIZE {
		shift := bits.BIT_COUNT_32_MAXIMUM - binary.BITS_PER_BYTE*(index+1)
		destination[index] = byte(value >> shift)
	}
	return OUTPUT_32_COUNT_COMPLETE, OUTPUT_STATUS_OK
}

// Digest_32_Clone_Into keeps source and result in caller storage.
func Digest_32_Clone_Into(destination *Digest_32, source *Digest_32) {
	Digest_32_Invariants(*destination, "Digest_32_Clone_Into.destination.input")
	Digest_32_Invariants(*source, "Digest_32_Clone_Into.source")
	defer func() {
		Digest_32_Invariants(*destination, "Digest_32_Clone_Into.destination.output")
	}()
	digest_32_require(source)
	*destination = *source
}

// Digest_64_Init establishes width-specific offset basis and caller-selected Kind.
func Digest_64_Init(digest *Digest_64, kind Kind) {
	Digest_64_Invariants(*digest, "Digest_64_Init.digest.input")
	Kind_Invariants(kind, "Digest_64_Init.kind")
	kind_require(kind)
	digest.Kind = kind
	digest.Value = OFFSET_64
	digest.Ready = READY_COMPLETE
	invariant.Always(
		digest.Kind == kind,
		"Initialized FNV-64 retains caller Kind.",
	)
	invariant.Always(
		digest.Value == OFFSET_64,
		"Initialized FNV-64 uses offset basis.",
	)
}

// Digest_64_Reset retains Kind while discarding prior bytes.
func Digest_64_Reset(digest *Digest_64) {
	Digest_64_Invariants(*digest, "Digest_64_Reset.digest.input")
	digest_64_require(digest)
	digest.Value = OFFSET_64
	invariant.Always(
		digest.Value == OFFSET_64,
		"Reset FNV-64 uses offset basis.",
	)
	Kind_Invariants(digest.Kind, "Digest_64_Reset.digest.kind.output")
}

// Digest_64_Write consumes one bounded source completely.
func Digest_64_Write(digest *Digest_64, source Source) (count Count) {
	defer func() { Count_Invariants(count, "Digest_64_Write.count") }()
	Digest_64_Invariants(*digest, "Digest_64_Write.digest.input")
	Source_Invariants(source, "Digest_64_Write.source")
	defer func() { Digest_64_Invariants(*digest, "Digest_64_Write.digest.output") }()
	digest_64_require(digest)
	if len(source) > SOURCE_SIZE_MAXIMUM {
		panic("fnv: source exceeds bound")
	}
	value := digest.Value
	for _, item := range source {
		if digest.Kind == KIND_1A {
			value ^= Value_64(item)
		}
		value *= PRIME_64
		if digest.Kind == KIND_1 {
			value ^= Value_64(item)
		}
	}
	digest.Value = value
	return Count(len(source))
}

// Digest_64_Sum observes state without consuming it.
func Digest_64_Sum(digest *Digest_64) (value Value_64) {
	defer func() { Value_64_Invariants(value, "Digest_64_Sum.value") }()
	Digest_64_Invariants(*digest, "Digest_64_Sum.digest")
	digest_64_require(digest)
	return digest.Value
}

// Digest_64_Sum_Into writes a complete big-endian value or leaves short storage untouched.
func Digest_64_Sum_Into(
	digest *Digest_64, destination Destination,
) (count Output_64_Count, status Output_Status) {
	defer func() {
		Output_64_Count_Invariants(count, "Digest_64_Sum_Into.count")
		Output_Status_Invariants(status, "Digest_64_Sum_Into.status")
	}()
	Digest_64_Invariants(*digest, "Digest_64_Sum_Into.digest")
	Destination_Invariants(destination, "Digest_64_Sum_Into.destination")
	digest_64_require(digest)
	if len(destination) > DESTINATION_SIZE_MAXIMUM {
		panic("fnv: destination exceeds bound")
	}
	if len(destination) < DIGEST_64_SIZE {
		return OUTPUT_64_COUNT_EMPTY, OUTPUT_STATUS_TOO_SMALL
	}
	value := uint64(digest.Value)
	for index := range DIGEST_64_SIZE {
		shift := bits.BIT_COUNT_64_MAXIMUM - binary.BITS_PER_BYTE*(index+1)
		destination[index] = byte(value >> shift)
	}
	return OUTPUT_64_COUNT_COMPLETE, OUTPUT_STATUS_OK
}

// Digest_64_Clone_Into keeps source and result in caller storage.
func Digest_64_Clone_Into(destination *Digest_64, source *Digest_64) {
	Digest_64_Invariants(*destination, "Digest_64_Clone_Into.destination.input")
	Digest_64_Invariants(*source, "Digest_64_Clone_Into.source")
	defer func() {
		Digest_64_Invariants(*destination, "Digest_64_Clone_Into.destination.output")
	}()
	digest_64_require(source)
	*destination = *source
}

// Digest_128_Init establishes width-specific offset basis and caller-selected Kind.
func Digest_128_Init(digest *Digest_128, kind Kind) {
	Digest_128_Invariants(*digest, "Digest_128_Init.digest.input")
	Kind_Invariants(kind, "Digest_128_Init.kind")
	kind_require(kind)
	digest.Kind = kind
	digest.Value = Value_128{High: OFFSET_128_HIGH, Low: OFFSET_128_LOW}
	digest.Ready = READY_COMPLETE
	invariant.Always(
		digest.Kind == kind,
		"Initialized FNV-128 retains caller Kind.",
	)
	invariant.Always(
		digest.Value.High == OFFSET_128_HIGH,
		"Initialized FNV-128 uses high offset basis.",
	)
	invariant.Always(
		digest.Value.Low == OFFSET_128_LOW,
		"Initialized FNV-128 uses low offset basis.",
	)
}

// Digest_128_Reset retains Kind while discarding prior bytes.
func Digest_128_Reset(digest *Digest_128) {
	Digest_128_Invariants(*digest, "Digest_128_Reset.digest.input")
	digest_128_require(digest)
	digest.Value = Value_128{High: OFFSET_128_HIGH, Low: OFFSET_128_LOW}
	invariant.Always(
		digest.Value.High == OFFSET_128_HIGH,
		"Reset FNV-128 uses high offset basis.",
	)
	invariant.Always(
		digest.Value.Low == OFFSET_128_LOW,
		"Reset FNV-128 uses low offset basis.",
	)
	Kind_Invariants(digest.Kind, "Digest_128_Reset.digest.kind.output")
}

// Digest_128_Write consumes one bounded source completely.
func Digest_128_Write(digest *Digest_128, source Source) (count Count) {
	defer func() { Count_Invariants(count, "Digest_128_Write.count") }()
	Digest_128_Invariants(*digest, "Digest_128_Write.digest.input")
	Source_Invariants(source, "Digest_128_Write.source")
	defer func() { Digest_128_Invariants(*digest, "Digest_128_Write.digest.output") }()
	digest_128_require(digest)
	if len(source) > SOURCE_SIZE_MAXIMUM {
		panic("fnv: source exceeds bound")
	}
	high := uint64(digest.Value.High)
	low := uint64(digest.Value.Low)
	for _, item := range source {
		if digest.Kind == KIND_1A {
			low ^= uint64(item)
		}
		// Expanding the double-width product keeps the fixed FNV prime out of a public
		// full-domain multiplier contract. The same four half-word products remain.
		left_low := PRIME_128_LOW & uint64(bits.WORD_32_MAXIMUM)
		left_high := PRIME_128_LOW >> bits.BIT_COUNT_32_MAXIMUM
		right_low := low & uint64(bits.WORD_32_MAXIMUM)
		right_high := low >> bits.BIT_COUNT_32_MAXIMUM
		partial := left_low * right_low
		middle_first := left_high*right_low + partial>>bits.BIT_COUNT_32_MAXIMUM
		middle_second := left_low*right_high + middle_first&uint64(bits.WORD_32_MAXIMUM)
		upper := left_high*right_high + middle_first>>bits.BIT_COUNT_32_MAXIMUM +
			middle_second>>bits.BIT_COUNT_32_MAXIMUM
		product_low := PRIME_128_LOW * low
		product_high := upper + low<<PRIME_128_SHIFT + PRIME_128_LOW*high
		low = product_low
		high = product_high
		if digest.Kind == KIND_1 {
			low ^= uint64(item)
		}
	}
	digest.Value = Value_128{High: High(high), Low: Low(low)}
	return Count(len(source))
}

// Digest_128_Sum observes state without consuming it.
func Digest_128_Sum(digest *Digest_128) (value Value_128) {
	defer func() { Value_128_Invariants(value, "Digest_128_Sum.value") }()
	Digest_128_Invariants(*digest, "Digest_128_Sum.digest")
	digest_128_require(digest)
	return digest.Value
}

// Digest_128_Sum_Into writes a complete big-endian value or leaves short storage untouched.
func Digest_128_Sum_Into(
	digest *Digest_128, destination Destination,
) (count Output_128_Count, status Output_Status) {
	defer func() {
		Output_128_Count_Invariants(count, "Digest_128_Sum_Into.count")
		Output_Status_Invariants(status, "Digest_128_Sum_Into.status")
	}()
	Digest_128_Invariants(*digest, "Digest_128_Sum_Into.digest")
	Destination_Invariants(destination, "Digest_128_Sum_Into.destination")
	digest_128_require(digest)
	if len(destination) > DESTINATION_SIZE_MAXIMUM {
		panic("fnv: destination exceeds bound")
	}
	if len(destination) < DIGEST_128_SIZE {
		return OUTPUT_128_COUNT_EMPTY, OUTPUT_STATUS_TOO_SMALL
	}
	high := uint64(digest.Value.High)
	low := uint64(digest.Value.Low)
	low_position := DIGEST_64_SIZE
	for index := range DIGEST_64_SIZE {
		shift := bits.BIT_COUNT_64_MAXIMUM - binary.BITS_PER_BYTE*(index+1)
		destination[index] = byte(high >> shift)
		destination[low_position] = byte(low >> shift)
		low_position++
	}
	return OUTPUT_128_COUNT_COMPLETE, OUTPUT_STATUS_OK
}

// Digest_128_Clone_Into keeps source and result in caller storage.
func Digest_128_Clone_Into(destination *Digest_128, source *Digest_128) {
	Digest_128_Invariants(*destination, "Digest_128_Clone_Into.destination.input")
	Digest_128_Invariants(*source, "Digest_128_Clone_Into.source")
	defer func() {
		Digest_128_Invariants(*destination, "Digest_128_Clone_Into.destination.output")
	}()
	digest_128_require(source)
	*destination = *source
}

// Digest_32_Marshal_Into emits standard width-and-kind state into caller storage.
func Digest_32_Marshal_Into(
	digest *Digest_32, destination Destination,
) (count State_32_Count, status State_Output_Status) {
	defer func() {
		State_32_Count_Invariants(count, "Digest_32_Marshal_Into.count")
		State_Output_Status_Invariants(status, "Digest_32_Marshal_Into.status")
	}()
	Digest_32_Invariants(*digest, "Digest_32_Marshal_Into.digest")
	Destination_Invariants(destination, "Digest_32_Marshal_Into.destination")
	digest_32_require(digest)
	if len(destination) > DESTINATION_SIZE_MAXIMUM {
		panic("fnv: destination exceeds bound")
	}
	if len(destination) < STATE_32_SIZE {
		return STATE_32_COUNT_EMPTY, STATE_OUTPUT_STATUS_TOO_SMALL
	}
	copy(
		destination[:STATE_IDENTITY_SIZE],
		state_32_identity(digest.Kind),
	)
	for index := range DIGEST_32_SIZE {
		shift := bits.BIT_COUNT_32_MAXIMUM - binary.BITS_PER_BYTE*(index+1)
		destination[STATE_VALUE_POSITION+index] = byte(
			uint32(digest.Value) >> shift,
		)
	}
	return STATE_32_COUNT_COMPLETE, STATE_OUTPUT_STATUS_OK
}

// Digest_32_Unmarshal changes state only after exact width-and-kind validation.
func Digest_32_Unmarshal(digest *Digest_32, source Source) (status State_Input_Status) {
	defer func() { State_Input_Status_Invariants(status, "Digest_32_Unmarshal.status") }()
	Digest_32_Invariants(*digest, "Digest_32_Unmarshal.digest.input")
	Source_Invariants(source, "Digest_32_Unmarshal.source")
	defer func() { Digest_32_Invariants(*digest, "Digest_32_Unmarshal.digest.output") }()
	digest_32_require(digest)
	if len(source) > SOURCE_SIZE_MAXIMUM {
		panic("fnv: source exceeds bound")
	}
	if !state_identity_match(
		source,
		state_32_identity(digest.Kind),
	) {
		return STATE_INPUT_STATUS_IDENTIFIER_INVALID
	}
	if len(source) != STATE_32_SIZE {
		return STATE_INPUT_STATUS_SIZE_INVALID
	}
	var value uint32
	for index := range DIGEST_32_SIZE {
		shift := bits.BIT_COUNT_32_MAXIMUM - binary.BITS_PER_BYTE*(index+1)
		value |= uint32(source[STATE_VALUE_POSITION+index]) << shift
	}
	digest.Value = Value_32(value)
	return STATE_INPUT_STATUS_OK
}

// Digest_64_Marshal_Into emits standard width-and-kind state into caller storage.
func Digest_64_Marshal_Into(
	digest *Digest_64, destination Destination,
) (count State_64_Count, status State_Output_Status) {
	defer func() {
		State_64_Count_Invariants(count, "Digest_64_Marshal_Into.count")
		State_Output_Status_Invariants(status, "Digest_64_Marshal_Into.status")
	}()
	Digest_64_Invariants(*digest, "Digest_64_Marshal_Into.digest")
	Destination_Invariants(destination, "Digest_64_Marshal_Into.destination")
	digest_64_require(digest)
	if len(destination) > DESTINATION_SIZE_MAXIMUM {
		panic("fnv: destination exceeds bound")
	}
	if len(destination) < STATE_64_SIZE {
		return STATE_64_COUNT_EMPTY, STATE_OUTPUT_STATUS_TOO_SMALL
	}
	copy(
		destination[:STATE_IDENTITY_SIZE],
		state_64_identity(digest.Kind),
	)
	for index := range DIGEST_64_SIZE {
		shift := bits.BIT_COUNT_64_MAXIMUM - binary.BITS_PER_BYTE*(index+1)
		destination[STATE_VALUE_POSITION+index] = byte(
			uint64(digest.Value) >> shift,
		)
	}
	return STATE_64_COUNT_COMPLETE, STATE_OUTPUT_STATUS_OK
}

// Digest_64_Unmarshal changes state only after exact width-and-kind validation.
func Digest_64_Unmarshal(digest *Digest_64, source Source) (status State_Input_Status) {
	defer func() { State_Input_Status_Invariants(status, "Digest_64_Unmarshal.status") }()
	Digest_64_Invariants(*digest, "Digest_64_Unmarshal.digest.input")
	Source_Invariants(source, "Digest_64_Unmarshal.source")
	defer func() { Digest_64_Invariants(*digest, "Digest_64_Unmarshal.digest.output") }()
	digest_64_require(digest)
	if len(source) > SOURCE_SIZE_MAXIMUM {
		panic("fnv: source exceeds bound")
	}
	if !state_identity_match(
		source,
		state_64_identity(digest.Kind),
	) {
		return STATE_INPUT_STATUS_IDENTIFIER_INVALID
	}
	if len(source) != STATE_64_SIZE {
		return STATE_INPUT_STATUS_SIZE_INVALID
	}
	var value uint64
	for index := range DIGEST_64_SIZE {
		shift := bits.BIT_COUNT_64_MAXIMUM - binary.BITS_PER_BYTE*(index+1)
		value |= uint64(source[STATE_VALUE_POSITION+index]) << shift
	}
	digest.Value = Value_64(value)
	return STATE_INPUT_STATUS_OK
}

// Digest_128_Marshal_Into emits standard width-and-kind state into caller storage.
func Digest_128_Marshal_Into(
	digest *Digest_128, destination Destination,
) (count State_128_Count, status State_Output_Status) {
	defer func() {
		State_128_Count_Invariants(count, "Digest_128_Marshal_Into.count")
		State_Output_Status_Invariants(status, "Digest_128_Marshal_Into.status")
	}()
	Digest_128_Invariants(*digest, "Digest_128_Marshal_Into.digest")
	Destination_Invariants(destination, "Digest_128_Marshal_Into.destination")
	digest_128_require(digest)
	if len(destination) > DESTINATION_SIZE_MAXIMUM {
		panic("fnv: destination exceeds bound")
	}
	if len(destination) < STATE_128_SIZE {
		return STATE_128_COUNT_EMPTY, STATE_OUTPUT_STATUS_TOO_SMALL
	}
	copy(
		destination[:STATE_IDENTITY_SIZE],
		state_128_identity(digest.Kind),
	)
	for index := range DIGEST_64_SIZE {
		shift := bits.BIT_COUNT_64_MAXIMUM - binary.BITS_PER_BYTE*(index+1)
		destination[STATE_VALUE_POSITION+index] = byte(
			uint64(digest.Value.High) >> shift,
		)
		destination[STATE_128_LOW_POSITION+index] = byte(
			uint64(digest.Value.Low) >> shift,
		)
	}
	return STATE_128_COUNT_COMPLETE, STATE_OUTPUT_STATUS_OK
}

// Digest_128_Unmarshal changes state only after exact width-and-kind validation.
func Digest_128_Unmarshal(digest *Digest_128, source Source) (status State_Input_Status) {
	defer func() { State_Input_Status_Invariants(status, "Digest_128_Unmarshal.status") }()
	Digest_128_Invariants(*digest, "Digest_128_Unmarshal.digest.input")
	Source_Invariants(source, "Digest_128_Unmarshal.source")
	defer func() { Digest_128_Invariants(*digest, "Digest_128_Unmarshal.digest.output") }()
	digest_128_require(digest)
	if len(source) > SOURCE_SIZE_MAXIMUM {
		panic("fnv: source exceeds bound")
	}
	if !state_identity_match(
		source,
		state_128_identity(digest.Kind),
	) {
		return STATE_INPUT_STATUS_IDENTIFIER_INVALID
	}
	if len(source) != STATE_128_SIZE {
		return STATE_INPUT_STATUS_SIZE_INVALID
	}
	var high uint64
	var low uint64
	for index := range DIGEST_64_SIZE {
		shift := bits.BIT_COUNT_64_MAXIMUM - binary.BITS_PER_BYTE*(index+1)
		high |= uint64(source[STATE_VALUE_POSITION+index]) << shift
		low |= uint64(source[STATE_128_LOW_POSITION+index]) << shift
	}
	digest.Value = Value_128{High: High(high), Low: Low(low)}
	return STATE_INPUT_STATUS_OK
}

// Readiness blocks zero caller storage; Kind validation survives disabled invariants.
func kind_require(kind Kind) {
	Kind_Invariants(kind, "kind_require.kind")
	valid := kind == KIND_1 || kind == KIND_1A
	invariant.Always(valid, "FNV Kind is FNV-1 or FNV-1a.")
	if !valid {
		panic("fnv: kind is invalid")
	}
}

func digest_32_require(digest *Digest_32) {
	Digest_32_Invariants(*digest, "digest_32_require.digest")
	kind_require(digest.Kind)
	invariant.Always(
		digest.Ready == READY_COMPLETE,
		"FNV-32 operations require Digest_32_Init.",
	)
	if digest.Ready != READY_COMPLETE {
		panic("fnv: digest-32 is not initialized")
	}
}

func digest_64_require(digest *Digest_64) {
	Digest_64_Invariants(*digest, "digest_64_require.digest")
	kind_require(digest.Kind)
	invariant.Always(
		digest.Ready == READY_COMPLETE,
		"FNV-64 operations require Digest_64_Init.",
	)
	if digest.Ready != READY_COMPLETE {
		panic("fnv: digest-64 is not initialized")
	}
}

func digest_128_require(digest *Digest_128) {
	Digest_128_Invariants(*digest, "digest_128_require.digest")
	kind_require(digest.Kind)
	invariant.Always(
		digest.Ready == READY_COMPLETE,
		"FNV-128 operations require Digest_128_Init.",
	)
	if digest.Ready != READY_COMPLETE {
		panic("fnv: digest-128 is not initialized")
	}
}

func state_32_identity(kind Kind) (identity State_Identity) {
	defer func() { State_Identity_Invariants(identity, "state_32_identity.identity") }()
	Kind_Invariants(kind, "state_32_identity.kind")
	if kind == KIND_1 {
		return STATE_32_1_IDENTITY
	}
	return STATE_32_1A_IDENTITY
}

func state_64_identity(kind Kind) (identity State_Identity) {
	defer func() { State_Identity_Invariants(identity, "state_64_identity.identity") }()
	Kind_Invariants(kind, "state_64_identity.kind")
	if kind == KIND_1 {
		return STATE_64_1_IDENTITY
	}
	return STATE_64_1A_IDENTITY
}

func state_128_identity(kind Kind) (identity State_Identity) {
	defer func() { State_Identity_Invariants(identity, "state_128_identity.identity") }()
	Kind_Invariants(kind, "state_128_identity.kind")
	if kind == KIND_1 {
		return STATE_128_1_IDENTITY
	}
	return STATE_128_1A_IDENTITY
}

func state_identity_match(
	source Source, identity State_Identity,
) (match bytes.Boolean) {
	defer func() { bytes.Boolean_Invariants(match, "state_identity_match.match") }()
	Source_Invariants(source, "state_identity_match.source")
	State_Identity_Invariants(identity, "state_identity_match.identity")
	if len(source) < STATE_IDENTITY_SIZE {
		return bytes.Boolean(false)
	}
	for index := range STATE_IDENTITY_SIZE {
		if source[index] != identity[index] {
			return bytes.Boolean(false)
		}
	}
	return bytes.Boolean(true)
}
