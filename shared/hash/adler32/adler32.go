// Package adler32 computes the Adler-32 checksum with bounded caller-owned state and output.
package adler32

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
)

// MODULUS is the largest prime below one 16-bit word, selected by RFC 1950.
const MODULUS = 65521

// COMPONENT_MINIMUM is the smallest reduced Adler component.
const COMPONENT_MINIMUM uint16 = bits.WORD_16_MINIMUM

// COMPONENT_MAXIMUM is the largest residue below MODULUS.
const COMPONENT_MAXIMUM uint16 = MODULUS - 1

// COMPONENT_SIZE is the bit width of each Adler accumulator inside one checksum.
const COMPONENT_SIZE = bits.BIT_COUNT_16_MAXIMUM

// DIGEST_SIZE is one 32-bit checksum in bytes.
const DIGEST_SIZE = bits.BIT_COUNT_32_MAXIMUM / binary.BITS_PER_BYTE

// BLOCK_SIZE preserves the standard Adler-32 streaming contract.
const BLOCK_SIZE = DIGEST_SIZE

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

// OUTPUT_COUNT_EMPTY means no partial checksum reached caller storage.
const OUTPUT_COUNT_EMPTY Output_Count = 0

// OUTPUT_COUNT_COMPLETE means one complete checksum reached caller storage.
const OUTPUT_COUNT_COMPLETE Output_Count = DIGEST_SIZE

// OUTPUT_STATUS_OK means one complete checksum reached caller storage.
const OUTPUT_STATUS_OK Output_Status = Output_Status(bits.WORD_8_MINIMUM)

// OUTPUT_STATUS_TOO_SMALL leaves short caller storage untouched.
const OUTPUT_STATUS_TOO_SMALL Output_Status = OUTPUT_STATUS_OK + 1

// STATE_IDENTITY_SIZE is the standard three-byte name plus one format-version byte.
const STATE_IDENTITY_SIZE = len("adl\x01")

// STATE_SIZE is the standard identity followed by one checksum.
const STATE_SIZE = STATE_IDENTITY_SIZE + DIGEST_SIZE

// STATE_OUTPUT_STATUS_OK means complete state reached caller storage.
const STATE_OUTPUT_STATUS_OK State_Output_Status = State_Output_Status(bits.WORD_8_MINIMUM)

// STATE_OUTPUT_STATUS_TOO_SMALL means caller storage cannot hold standard state.
const STATE_OUTPUT_STATUS_TOO_SMALL State_Output_Status = STATE_OUTPUT_STATUS_OK + 1

// STATE_INPUT_STATUS_OK means validated state replaced caller state.
const STATE_INPUT_STATUS_OK State_Input_Status = State_Input_Status(bits.WORD_8_MINIMUM)

// STATE_INPUT_STATUS_SIZE_INVALID rejects serialized state with trailing or missing bytes.
const STATE_INPUT_STATUS_SIZE_INVALID State_Input_Status = STATE_INPUT_STATUS_OK + 1

// STATE_INPUT_STATUS_IDENTIFIER_INVALID rejects state owned by another algorithm or version.
const STATE_INPUT_STATUS_IDENTIFIER_INVALID State_Input_Status = STATE_INPUT_STATUS_SIZE_INVALID + 1

// STATE_COUNT_EMPTY means no partial state reached caller storage.
const STATE_COUNT_EMPTY State_Count = 0

// STATE_COUNT_COMPLETE means one complete standard state reached caller storage.
const STATE_COUNT_COMPLETE State_Count = State_Count(STATE_SIZE)

// READY_EMPTY marks caller storage before initialization.
const READY_EMPTY Ready = false

// READY_COMPLETE marks state established by Init or Unmarshal.
const READY_COMPLETE Ready = true

// Ready reports whether caller storage contains established state.
type Ready bool

// Ready_Invariants covers both lifecycle states.
func Ready_Invariants(value Ready, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Adler-32 digest is initialized.").
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

// Count_Invariants covers complete source count.
func Count_Invariants(value Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Output_Count is bytes populated by Digest_Sum_Into.
type Output_Count uint8

// Output_Count_Invariants excludes partial checksum output.
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

// Digest is caller-owned initialized Adler-32 streaming state.
type Digest struct {
	// Value is standard checksum state.
	Value Digest_Value
	// Ready separates zero storage from arbitrary standard state accepted by Unmarshal.
	Ready Ready
}

// Digest_Invariants preserves every serialized state the standard library accepts.
func Digest_Invariants(value Digest, namespace aver.Namespace) {
	Digest_Value_Invariants(value.Value, namespace)
	Ready_Invariants(value.Ready, namespace)
}

// Digest_Handle gives caller state one pointer identity.
type Digest_Handle *Digest

// Digest_Handle_Invariants composes present state.
func Digest_Handle_Invariants(value Digest_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Digest_Invariants(*value, namespace)
}

// Digest_Value is a direct observation of streaming state, including restored arbitrary state.
type Digest_Value uint32

// Digest_Value_Invariants preserves every value standard state can contain.
func Digest_Value_Invariants(value Digest_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, bits.WORD_32_MAXIMUM).
		Ensure()
}

// Sum_1 is the low Adler component after modular reduction.
type Sum_1 uint16

// Sum_1_Invariants keeps computed low state below MODULUS.
func Sum_1_Invariants(value Sum_1, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), COMPONENT_MINIMUM, COMPONENT_MAXIMUM).
		Ensure()
}

// Sum_2 is the high Adler component after modular reduction.
type Sum_2 uint16

// Sum_2_Invariants keeps computed high state below MODULUS.
func Sum_2_Invariants(value Sum_2, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), COMPONENT_MINIMUM, COMPONENT_MAXIMUM).
		Ensure()
}

// Value is a computed checksum whose components cannot contain arbitrary serialized state.
type Value struct {
	// Sum_1 is the low checksum word.
	Sum_1 Sum_1
	// Sum_2 is the high checksum word.
	Sum_2 Sum_2
}

// Value_Invariants composes the two independently reduced components.
func Value_Invariants(value Value, namespace aver.Namespace) {
	Sum_1_Invariants(value.Sum_1, namespace)
	Sum_2_Invariants(value.Sum_2, namespace)
}

// State_Count is either no state or one complete standard state.
type State_Count uint8

// State_Count_Invariants excludes partial serialized state.
func State_Count_Invariants(value State_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATE_COUNT_EMPTY), uint8(STATE_COUNT_COMPLETE),
		).
		Ensure()
}

// State_Output_Status reports caller state-output capacity.
type State_Output_Status uint8

// State_Output_Status_Invariants covers both state-output outcomes.
func State_Output_Status_Invariants(value State_Output_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATE_OUTPUT_STATUS_OK),
			uint8(STATE_OUTPUT_STATUS_TOO_SMALL),
		).
		Ensure()
}

// State_Output keeps serialized count and status in one invariant chain.
type State_Output struct {
	// Count reports initialized state bytes.
	Count State_Count
	// Status classifies same state write.
	Status State_Output_Status
}

// State_Output_Invariants composes matching state count and status domains.
func State_Output_Invariants(value State_Output, namespace aver.Namespace) {
	State_Count_Invariants(value.Count, namespace)
	State_Output_Status_Invariants(value.Status, namespace)
}

// State_Input_Status reports hostile state-input validation.
type State_Input_Status uint8

// State_Input_Status_Invariants covers every state-input outcome.
func State_Input_Status_Invariants(value State_Input_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATE_INPUT_STATUS_OK),
			uint8(STATE_INPUT_STATUS_SIZE_INVALID),
			uint8(STATE_INPUT_STATUS_IDENTIFIER_INVALID),
		).
		Ensure()
}

// Digest_Init establishes the RFC 1950 initial low accumulator of one.
func Digest_Init(digest Digest_Handle) {
	Digest_Handle_Invariants(digest, "Digest_Init.digest.input")
	digest.Value = 1
	digest.Ready = READY_COMPLETE
	aver.Always(digest.Value == 1, "Fresh Adler-32 state starts at one.")
}

// Digest_Reset makes existing caller storage equal to freshly initialized state.
func Digest_Reset(digest Digest_Handle) {
	Digest_Handle_Invariants(digest, "Digest_Reset.digest.input")
	digest_require(digest)
	digest.Value = 1
	aver.Always(digest.Value == 1, "Reset Adler-32 state starts at one.")
}

// Digest_Write can defer reduction because one bounded call stays below uint32 overflow even when
// restored state begins with full 16-bit components.
func Digest_Write(digest Digest_Handle, source Source) (count Count) {
	defer func() { Count_Invariants(count, "Digest_Write.count") }()
	Digest_Handle_Invariants(digest, "Digest_Write.digest.input")
	Source_Invariants(source, "Digest_Write.source")
	defer func() { Digest_Handle_Invariants(digest, "Digest_Write.digest.output") }()
	digest_require(digest)
	aver.Always(
		len(source) <= SOURCE_SIZE_MAXIMUM,
		"Digest_Write source stays within source bound.",
	)
	// Standard state accepts arbitrary uint32 values; empty input must not normalize a restored
	// value whose 16-bit components sit above MODULUS.
	if len(source) != 0 {
		sum_1 := uint32(digest.Value) & uint32(bits.WORD_16_MAXIMUM)
		sum_2 := uint32(digest.Value) >> COMPONENT_SIZE
		for _, value := range source {
			sum_1 += uint32(value)
			sum_2 += sum_1
		}
		sum_1 %= MODULUS
		sum_2 %= MODULUS
		digest.Value = Digest_Value(sum_2<<COMPONENT_SIZE | sum_1)
	}
	return Count(len(source))
}

// Digest_Sum_32 observes state without consuming it.
func Digest_Sum_32(digest Digest_Handle) (checksum Digest_Value) {
	defer func() { Digest_Value_Invariants(checksum, "Digest_Sum_32.checksum") }()
	Digest_Handle_Invariants(digest, "Digest_Sum_32.digest")
	digest_require(digest)
	return digest.Value
}

// Digest_Sum_Into writes a complete big-endian checksum or leaves short storage untouched.
func Digest_Sum_Into(
	digest Digest_Handle, destination Destination,
) (output Output) {
	defer func() { Output_Invariants(output, "Digest_Sum_Into.output") }()
	Digest_Handle_Invariants(digest, "Digest_Sum_Into.digest")
	Destination_Invariants(destination, "Digest_Sum_Into.destination")
	digest_require(digest)
	aver.Always(
		len(destination) <= DESTINATION_SIZE_MAXIMUM,
		"Digest_Sum_Into destination stays within destination bound.",
	)
	if len(destination) < DIGEST_SIZE {
		return Output{Count: OUTPUT_COUNT_EMPTY, Status: OUTPUT_STATUS_TOO_SMALL}
	}
	value := uint32(digest.Value)
	destination[0] = byte(value >> (bits.BIT_COUNT_32_MAXIMUM - binary.BITS_PER_BYTE))
	destination[1] = byte(value >> (bits.BIT_COUNT_32_MAXIMUM - binary.BITS_PER_BYTE*2))
	destination[2] = byte(value >> (bits.BIT_COUNT_32_MAXIMUM - binary.BITS_PER_BYTE*3))
	destination[DIGEST_SIZE-1] = byte(value)
	return Output{Count: OUTPUT_COUNT_COMPLETE, Status: OUTPUT_STATUS_OK}
}

// Checksum computes one bounded source without owning state or result storage.
func Checksum(source Source) (checksum Value) {
	defer func() { Value_Invariants(checksum, "Checksum.checksum") }()
	Source_Invariants(source, "Checksum.source")
	aver.Always(
		len(source) <= SOURCE_SIZE_MAXIMUM,
		"Checksum source stays within source bound.",
	)
	var digest Digest
	Digest_Init(&digest)
	Digest_Write(&digest, source)
	return Value{
		Sum_1: Sum_1(uint32(digest.Value) & uint32(bits.WORD_16_MAXIMUM)),
		Sum_2: Sum_2(uint32(digest.Value) >> COMPONENT_SIZE),
	}
}

// Digest_Marshal_Into emits exactly the state understood by the standard library.
func Digest_Marshal_Into(
	digest Digest_Handle, destination Destination,
) (output State_Output) {
	defer func() { State_Output_Invariants(output, "Digest_Marshal_Into.output") }()
	Digest_Handle_Invariants(digest, "Digest_Marshal_Into.digest")
	Destination_Invariants(destination, "Digest_Marshal_Into.destination")
	digest_require(digest)
	aver.Always(
		len(destination) <= DESTINATION_SIZE_MAXIMUM,
		"Digest_Marshal_Into destination stays within destination bound.",
	)
	if len(destination) < STATE_SIZE {
		return State_Output{
			Count: STATE_COUNT_EMPTY, Status: STATE_OUTPUT_STATUS_TOO_SMALL,
		}
	}
	destination[0] = 'a'
	destination[1] = 'd'
	destination[2] = 'l'
	destination[STATE_IDENTITY_SIZE-1] = 1
	value := uint32(digest.Value)
	destination[STATE_IDENTITY_SIZE] = byte(
		value >> (bits.BIT_COUNT_32_MAXIMUM - binary.BITS_PER_BYTE),
	)
	destination[STATE_IDENTITY_SIZE+1] = byte(
		value >> (bits.BIT_COUNT_32_MAXIMUM - binary.BITS_PER_BYTE*2),
	)
	destination[STATE_IDENTITY_SIZE+2] = byte(
		value >> (bits.BIT_COUNT_32_MAXIMUM - binary.BITS_PER_BYTE*3),
	)
	destination[STATE_SIZE-1] = byte(value)
	return State_Output{Count: STATE_COUNT_COMPLETE, Status: STATE_OUTPUT_STATUS_OK}
}

// Digest_Unmarshal validates hostile bytes before replacing caller state.
func Digest_Unmarshal(digest Digest_Handle, source Source) (status State_Input_Status) {
	defer func() { State_Input_Status_Invariants(status, "Digest_Unmarshal.status") }()
	Digest_Handle_Invariants(digest, "Digest_Unmarshal.digest.input")
	Source_Invariants(source, "Digest_Unmarshal.source")
	defer func() { Digest_Handle_Invariants(digest, "Digest_Unmarshal.digest.output") }()
	aver.Always(
		len(source) <= SOURCE_SIZE_MAXIMUM,
		"Digest_Unmarshal source stays within source bound.",
	)
	if len(source) < STATE_IDENTITY_SIZE {
		return STATE_INPUT_STATUS_IDENTIFIER_INVALID
	}
	if source[0] != 'a' {
		return STATE_INPUT_STATUS_IDENTIFIER_INVALID
	}
	if source[1] != 'd' {
		return STATE_INPUT_STATUS_IDENTIFIER_INVALID
	}
	if source[2] != 'l' {
		return STATE_INPUT_STATUS_IDENTIFIER_INVALID
	}
	if source[STATE_IDENTITY_SIZE-1] != 1 {
		return STATE_INPUT_STATUS_IDENTIFIER_INVALID
	}
	if len(source) != STATE_SIZE {
		return STATE_INPUT_STATUS_SIZE_INVALID
	}
	digest.Value = Digest_Value(
		uint32(source[STATE_IDENTITY_SIZE])<<
			(bits.BIT_COUNT_32_MAXIMUM-binary.BITS_PER_BYTE) |
			uint32(source[STATE_IDENTITY_SIZE+1])<<
				(bits.BIT_COUNT_32_MAXIMUM-binary.BITS_PER_BYTE*2) |
			uint32(source[STATE_IDENTITY_SIZE+2])<<
				(bits.BIT_COUNT_32_MAXIMUM-binary.BITS_PER_BYTE*3) |
			uint32(source[STATE_SIZE-1]),
	)
	digest.Ready = READY_COMPLETE
	return STATE_INPUT_STATUS_OK
}

// Digest_Clone_Into keeps both source and result in caller storage.
func Digest_Clone_Into(destination Digest_Handle, source Digest_Handle) {
	Digest_Handle_Invariants(destination, "Digest_Clone_Into.destination.input")
	Digest_Handle_Invariants(source, "Digest_Clone_Into.source")
	defer func() {
		Digest_Handle_Invariants(destination, "Digest_Clone_Into.destination.output")
	}()
	digest_require(source)
	*destination = *source
}

// Readiness blocks zero caller storage from becoming attacker-selected checksum state.
func digest_require(digest Digest_Handle) {
	Digest_Handle_Invariants(digest, "digest_require.digest")
	aver.Always(
		digest.Ready == READY_COMPLETE,
		"Adler-32 operations require Digest_Init or Digest_Unmarshal.",
	)
}
