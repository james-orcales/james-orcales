// Package crc64 computes reflected 64-bit cyclic redundancy checks with caller-owned tables,
// state, and output.
package crc64

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
)

// TABLE_ENTRY_COUNT covers every possible low checksum byte.
const TABLE_ENTRY_COUNT = 1 << bits.BIT_COUNT_8_MAXIMUM

// TABLE_POLYNOMIAL_INDEX stores table identity after reduction entries.
const TABLE_POLYNOMIAL_INDEX = TABLE_ENTRY_COUNT

// TABLE_READY_INDEX stores initialization marker after polynomial identity.
const TABLE_READY_INDEX = TABLE_POLYNOMIAL_INDEX + 1

// TABLE_WORD_COUNT holds entries, polynomial identity, and initialization marker.
const TABLE_WORD_COUNT = TABLE_READY_INDEX + 1

// TABLE_READY_MARKER separates initialized table storage from zero caller storage.
const TABLE_READY_MARKER = bits.WORD_64_MAXIMUM

// DIGEST_SIZE is one 64-bit checksum in bytes.
const DIGEST_SIZE = bits.BIT_COUNT_64_MAXIMUM / binary.BITS_PER_BYTE

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

// OUTPUT_COUNT_EMPTY means no partial checksum reached caller storage.
const OUTPUT_COUNT_EMPTY Output_Count = 0

// OUTPUT_COUNT_COMPLETE means one complete checksum reached caller storage.
const OUTPUT_COUNT_COMPLETE Output_Count = DIGEST_SIZE

// OUTPUT_STATUS_OK means one complete checksum reached caller storage.
const OUTPUT_STATUS_OK Output_Status = Output_Status(bits.WORD_8_MINIMUM)

// OUTPUT_STATUS_TOO_SMALL leaves short caller storage untouched.
const OUTPUT_STATUS_TOO_SMALL Output_Status = OUTPUT_STATUS_OK + 1

// STATE_IDENTITY is standard CRC-64 state version.
const STATE_IDENTITY = "crc\x02"

// STATE_IDENTITY_SIZE is standard state prefix width.
const STATE_IDENTITY_SIZE = len(STATE_IDENTITY)

// STATE_TABLE_POSITION follows state identity.
const STATE_TABLE_POSITION = STATE_IDENTITY_SIZE

// STATE_DIGEST_POSITION follows table identity.
const STATE_DIGEST_POSITION = STATE_TABLE_POSITION + DIGEST_SIZE

// STATE_SIZE holds identity, table checksum, and current checksum.
const STATE_SIZE = STATE_DIGEST_POSITION + DIGEST_SIZE

// ISO is the reflected ISO 3309 polynomial.
const ISO Polynomial = 0xd800000000000000

// ECMA is the reflected ECMA 182 polynomial.
const ECMA Polynomial = 0xc96c5795d7870f42

// STATE_OUTPUT_STATUS_OK means complete state reached caller storage.
const STATE_OUTPUT_STATUS_OK State_Output_Status = State_Output_Status(bits.WORD_8_MINIMUM)

// STATE_OUTPUT_STATUS_TOO_SMALL means caller storage cannot hold standard state.
const STATE_OUTPUT_STATUS_TOO_SMALL State_Output_Status = STATE_OUTPUT_STATUS_OK + 1

// STATE_INPUT_STATUS_OK means validated state replaced current checksum.
const STATE_INPUT_STATUS_OK State_Input_Status = State_Input_Status(bits.WORD_8_MINIMUM)

// STATE_INPUT_STATUS_SIZE_INVALID rejects trailing or missing bytes.
const STATE_INPUT_STATUS_SIZE_INVALID State_Input_Status = STATE_INPUT_STATUS_OK + 1

// STATE_INPUT_STATUS_IDENTIFIER_INVALID rejects another algorithm or version.
const STATE_INPUT_STATUS_IDENTIFIER_INVALID State_Input_Status = State_Input_Status(
	STATE_INPUT_STATUS_SIZE_INVALID + 1,
)

// STATE_INPUT_STATUS_TABLE_INVALID rejects state built for another table.
const STATE_INPUT_STATUS_TABLE_INVALID State_Input_Status = State_Input_Status(
	STATE_INPUT_STATUS_IDENTIFIER_INVALID + 1,
)

// STATE_COUNT_EMPTY means no partial state reached caller storage.
const STATE_COUNT_EMPTY State_Count = 0

// STATE_COUNT_COMPLETE means one complete standard state reached caller storage.
const STATE_COUNT_COMPLETE State_Count = State_Count(STATE_SIZE)

// Polynomial is any reflected 64-bit CRC polynomial.
type Polynomial uint64

// Polynomial_Invariants keeps caller-selected polynomial inside one machine word.
func Polynomial_Invariants(value Polynomial, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
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

// Output_Count is bytes populated by Digest_Sum_Into.
type Output_Count uint8

// Output_Count_Invariants excludes partial checksum output.
func Output_Count_Invariants(value Output_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(OUTPUT_COUNT_EMPTY), uint8(OUTPUT_COUNT_COMPLETE),
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

// Digest_Value is the complete CRC-64 result domain.
type Digest_Value uint64

// Digest_Value_Invariants preserves every possible CRC-64 result.
func Digest_Value_Invariants(value Digest_Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Table is caller-owned entries plus initialization identity.
type Table [TABLE_WORD_COUNT]uint64

// Table_Invariants binds table identity to its construction polynomial.
func Table_Invariants(value Table, namespace invariant.Namespace) {
	Polynomial_Invariants(Polynomial(value[TABLE_POLYNOMIAL_INDEX]), namespace)
}

// Digest is caller-owned streaming checksum and immutable table state.
type Digest struct {
	// Checksum is current standard state.
	Checksum Digest_Value
	// Table retains initialized caller table by value.
	Table Table
}

// Digest_Invariants composes current checksum with table identity.
func Digest_Invariants(value Digest, namespace invariant.Namespace) {
	Digest_Value_Invariants(value.Checksum, namespace)
	Table_Invariants(value.Table, namespace)
}

// State_Count is either no state or one complete standard state.
type State_Count uint8

// State_Count_Invariants excludes partial state.
func State_Count_Invariants(value State_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(STATE_COUNT_EMPTY), uint8(STATE_COUNT_COMPLETE)).
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
		Enum_4_Uint8(
			uint8(value), uint8(STATE_INPUT_STATUS_OK),
			uint8(STATE_INPUT_STATUS_SIZE_INVALID),
			uint8(STATE_INPUT_STATUS_IDENTIFIER_INVALID),
			uint8(STATE_INPUT_STATUS_TABLE_INVALID),
		).
		Ensure()
}

// Table_Make_Into derives each entry from caller-selected polynomial instead of caching global
// tables whose lifetime and initialization would sit outside dependency injection.
func Table_Make_Into(table *Table, polynomial Polynomial) {
	Table_Invariants(*table, "Table_Make_Into.table.input")
	Polynomial_Invariants(polynomial, "Table_Make_Into.polynomial")
	for index := range TABLE_ENTRY_COUNT {
		checksum := uint64(index)
		for range bits.BIT_COUNT_8_MAXIMUM {
			if checksum&1 == 0 {
				checksum >>= 1
				continue
			}
			checksum = checksum>>1 ^ uint64(polynomial)
		}
		table[index] = checksum
	}
	table[TABLE_POLYNOMIAL_INDEX] = uint64(polynomial)
	table[TABLE_READY_INDEX] = TABLE_READY_MARKER
	Table_Invariants(*table, "Table_Make_Into.table.output")
}

// Update continues a checksum over one bounded source using caller-owned table state.
func Update(
	checksum Digest_Value, table *Table, source Source,
) (updated Digest_Value) {
	defer func() { Digest_Value_Invariants(updated, "Update.updated") }()
	Digest_Value_Invariants(checksum, "Update.checksum")
	Table_Invariants(*table, "Update.table")
	Source_Invariants(source, "Update.source")
	table_require(table)
	if len(source) > SOURCE_SIZE_MAXIMUM {
		panic("crc64: source exceeds bound")
	}
	register := ^uint64(checksum)
	for _, value := range source {
		register = table[byte(register)^value] ^ register>>bits.BIT_COUNT_8_MAXIMUM
	}
	return Digest_Value(^register)
}

// Checksum computes one bounded source with caller-selected table.
func Checksum(source Source, table *Table) (checksum Digest_Value) {
	defer func() { Digest_Value_Invariants(checksum, "Checksum.checksum") }()
	Source_Invariants(source, "Checksum.source")
	Table_Invariants(*table, "Checksum.table")
	table_require(table)
	if len(source) > SOURCE_SIZE_MAXIMUM {
		panic("crc64: source exceeds bound")
	}
	return Update(0, table, source)
}

// Digest_Init copies immutable table state so no external pointer can replace it during a stream.
func Digest_Init(digest *Digest, table *Table) {
	Digest_Invariants(*digest, "Digest_Init.digest.input")
	Table_Invariants(*table, "Digest_Init.table")
	table_require(table)
	digest.Checksum = 0
	digest.Table = *table
	invariant.Always(digest.Checksum == 0, "Fresh CRC-64 state starts at zero.")
	Table_Invariants(digest.Table, "Digest_Init.digest.table.output")
}

// Digest_Reset retains caller-selected polynomial while discarding prior bytes.
func Digest_Reset(digest *Digest) {
	Digest_Invariants(*digest, "Digest_Reset.digest.input")
	digest_require(digest)
	digest.Checksum = 0
	invariant.Always(digest.Checksum == 0, "Reset CRC-64 state starts at zero.")
	Table_Invariants(digest.Table, "Digest_Reset.digest.table.output")
}

// Digest_Write consumes one bounded source completely.
func Digest_Write(digest *Digest, source Source) (count Count) {
	defer func() { Count_Invariants(count, "Digest_Write.count") }()
	Digest_Invariants(*digest, "Digest_Write.digest.input")
	Source_Invariants(source, "Digest_Write.source")
	digest_require(digest)
	if len(source) > SOURCE_SIZE_MAXIMUM {
		panic("crc64: source exceeds bound")
	}
	digest.Checksum = Update(digest.Checksum, &digest.Table, source)
	Digest_Invariants(*digest, "Digest_Write.digest.output")
	return Count(len(source))
}

// Digest_Sum_64 observes state without consuming it.
func Digest_Sum_64(digest *Digest) (checksum Digest_Value) {
	defer func() { Digest_Value_Invariants(checksum, "Digest_Sum_64.checksum") }()
	Digest_Invariants(*digest, "Digest_Sum_64.digest")
	digest_require(digest)
	return digest.Checksum
}

// Digest_Sum_Into writes a complete big-endian checksum or leaves short storage untouched.
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
		panic("crc64: destination exceeds bound")
	}
	if len(destination) < DIGEST_SIZE {
		return OUTPUT_COUNT_EMPTY, OUTPUT_STATUS_TOO_SMALL
	}
	value := uint64(digest.Checksum)
	for index := range DIGEST_SIZE {
		shift := bits.BIT_COUNT_64_MAXIMUM - binary.BITS_PER_BYTE*(index+1)
		destination[index] = byte(value >> shift)
	}
	return OUTPUT_COUNT_COMPLETE, OUTPUT_STATUS_OK
}

// Digest_Clone_Into keeps table and checksum state in caller storage.
func Digest_Clone_Into(destination *Digest, source *Digest) {
	Digest_Invariants(*destination, "Digest_Clone_Into.destination.input")
	Digest_Invariants(*source, "Digest_Clone_Into.source")
	digest_require(source)
	*destination = *source
	Digest_Invariants(*destination, "Digest_Clone_Into.destination.output")
}

// Digest_Marshal_Into emits standard state into caller storage.
func Digest_Marshal_Into(
	digest *Digest, destination Destination,
) (count State_Count, status State_Output_Status) {
	defer func() {
		State_Count_Invariants(count, "Digest_Marshal_Into.count")
		State_Output_Status_Invariants(status, "Digest_Marshal_Into.status")
	}()
	Digest_Invariants(*digest, "Digest_Marshal_Into.digest")
	Destination_Invariants(destination, "Digest_Marshal_Into.destination")
	digest_require(digest)
	if len(destination) > DESTINATION_SIZE_MAXIMUM {
		panic("crc64: destination exceeds bound")
	}
	if len(destination) < STATE_SIZE {
		return STATE_COUNT_EMPTY, STATE_OUTPUT_STATUS_TOO_SMALL
	}
	copy(destination[:STATE_IDENTITY_SIZE], STATE_IDENTITY)
	var identity_table Table
	Table_Make_Into(&identity_table, ISO)
	register := ^uint64(0)
	for entry_index := range TABLE_ENTRY_COUNT {
		entry := digest.Table[entry_index]
		for byte_index := range DIGEST_SIZE {
			shift := bits.BIT_COUNT_64_MAXIMUM - binary.BITS_PER_BYTE*(byte_index+1)
			item := byte(entry >> shift)
			register = identity_table[byte(register)^item] ^
				register>>bits.BIT_COUNT_8_MAXIMUM
		}
	}
	table_identity := ^register
	for index := range DIGEST_SIZE {
		shift := bits.BIT_COUNT_64_MAXIMUM - binary.BITS_PER_BYTE*(index+1)
		destination[STATE_TABLE_POSITION+index] = byte(table_identity >> shift)
		destination[STATE_DIGEST_POSITION+index] = byte(uint64(digest.Checksum) >> shift)
	}
	return STATE_COUNT_COMPLETE, STATE_OUTPUT_STATUS_OK
}

// Digest_Unmarshal changes checksum only after identity, size, and table validation.
func Digest_Unmarshal(digest *Digest, source Source) (status State_Input_Status) {
	defer func() { State_Input_Status_Invariants(status, "Digest_Unmarshal.status") }()
	Digest_Invariants(*digest, "Digest_Unmarshal.digest.input")
	Source_Invariants(source, "Digest_Unmarshal.source")
	digest_require(digest)
	if len(source) > SOURCE_SIZE_MAXIMUM {
		panic("crc64: source exceeds bound")
	}
	if len(source) < STATE_IDENTITY_SIZE {
		return STATE_INPUT_STATUS_IDENTIFIER_INVALID
	}
	if source[0] != STATE_IDENTITY[0] {
		return STATE_INPUT_STATUS_IDENTIFIER_INVALID
	}
	if source[1] != STATE_IDENTITY[1] {
		return STATE_INPUT_STATUS_IDENTIFIER_INVALID
	}
	if source[2] != STATE_IDENTITY[2] {
		return STATE_INPUT_STATUS_IDENTIFIER_INVALID
	}
	if source[STATE_IDENTITY_SIZE-1] != STATE_IDENTITY[STATE_IDENTITY_SIZE-1] {
		return STATE_INPUT_STATUS_IDENTIFIER_INVALID
	}
	if len(source) != STATE_SIZE {
		return STATE_INPUT_STATUS_SIZE_INVALID
	}
	var table_identity uint64
	var checksum uint64
	for index := range DIGEST_SIZE {
		shift := bits.BIT_COUNT_64_MAXIMUM - binary.BITS_PER_BYTE*(index+1)
		table_identity |= uint64(source[STATE_TABLE_POSITION+index]) << shift
		checksum |= uint64(source[STATE_DIGEST_POSITION+index]) << shift
	}
	var identity_table Table
	Table_Make_Into(&identity_table, ISO)
	register := ^uint64(0)
	for entry_index := range TABLE_ENTRY_COUNT {
		entry := digest.Table[entry_index]
		for byte_index := range DIGEST_SIZE {
			shift := bits.BIT_COUNT_64_MAXIMUM - binary.BITS_PER_BYTE*(byte_index+1)
			item := byte(entry >> shift)
			register = identity_table[byte(register)^item] ^
				register>>bits.BIT_COUNT_8_MAXIMUM
		}
	}
	if table_identity != ^register {
		return STATE_INPUT_STATUS_TABLE_INVALID
	}
	digest.Checksum = Digest_Value(checksum)
	Digest_Invariants(*digest, "Digest_Unmarshal.digest.output")
	return STATE_INPUT_STATUS_OK
}

// Table and digest readiness prevent zero caller storage from becoming attacker-selected state.
func table_require(table *Table) {
	Table_Invariants(*table, "table_require.table")
	invariant.Always(
		table[TABLE_READY_INDEX] == TABLE_READY_MARKER,
		"CRC-64 table operations require Table_Make_Into.",
	)
	if table[TABLE_READY_INDEX] != TABLE_READY_MARKER {
		panic("crc64: table is not initialized")
	}
}

func digest_require(digest *Digest) {
	Digest_Invariants(*digest, "digest_require.digest")
	ready := digest.Table[TABLE_READY_INDEX] == TABLE_READY_MARKER
	invariant.Always(
		ready,
		"CRC-64 digest operations require Digest_Init.",
	)
	if !ready {
		panic("crc64: digest is not initialized")
	}
}
