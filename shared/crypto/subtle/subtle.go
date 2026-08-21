// Package subtle provides bounded constant-time cryptographic helpers.
package subtle

import (
	"unsafe"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/simulation/aver/default"
)

// SOURCE_SIZE_MINIMUM admits empty comparison and XOR operands.
const SOURCE_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// SOURCE_SIZE_MAXIMUM follows repository byte-slice bound.
const SOURCE_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// DESTINATION_SIZE_MINIMUM admits empty copy and XOR output.
const DESTINATION_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// DESTINATION_SIZE_MAXIMUM follows repository byte-slice bound.
const DESTINATION_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// COUNT_MINIMUM reports empty XOR output.
const COUNT_MINIMUM = SOURCE_SIZE_MINIMUM

// COUNT_MAXIMUM reports one complete bounded XOR operand.
const COUNT_MAXIMUM = SOURCE_SIZE_MAXIMUM

// DECISION_FALSE selects second value or preserves destination.
const DECISION_FALSE Decision = Decision(bits.WORD_8_MINIMUM)

// DECISION_TRUE selects first value or copies source.
const DECISION_TRUE Decision = DECISION_FALSE + binary.UINT_8_SIZE

// INTEGER_MINIMUM is smallest selectable machine integer.
const INTEGER_MINIMUM Integer = Integer(bits.INTEGER_MINIMUM)

// INTEGER_MAXIMUM is largest selectable machine integer.
const INTEGER_MAXIMUM Integer = Integer(bits.INTEGER_MAXIMUM)

// BYTE_MINIMUM is smallest byte equality operand.
const BYTE_MINIMUM Byte = Byte(bits.WORD_8_MINIMUM)

// BYTE_MAXIMUM is largest byte equality operand.
const BYTE_MAXIMUM Byte = Byte(bits.WORD_8_MAXIMUM)

// INTEGER_32_MINIMUM is smallest 32-bit equality operand.
const INTEGER_32_MINIMUM Integer_32 = Integer_32(bits.INTEGER_32_MINIMUM)

// INTEGER_32_MAXIMUM is largest 32-bit equality operand.
const INTEGER_32_MAXIMUM Integer_32 = Integer_32(bits.INTEGER_32_MAXIMUM)

// NONNEGATIVE_INTEGER_MINIMUM is crypto/subtle lower comparison bound.
const NONNEGATIVE_INTEGER_MINIMUM Nonnegative_Integer = Nonnegative_Integer(bits.WORD_32_MINIMUM)

// NONNEGATIVE_INTEGER_MAXIMUM is crypto/subtle upper comparison bound.
const NONNEGATIVE_INTEGER_MAXIMUM Nonnegative_Integer = Nonnegative_Integer(bits.INTEGER_32_MAXIMUM)

// Decision is zero or one. Runtime invariant checks would disclose secret results or selectors.
type Decision uint8

// Decision_Invariants is for non-secret validation outside constant-time paths.
func Decision_Invariants(value Decision, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(DECISION_FALSE), uint8(DECISION_TRUE)).
		Ensure()
}

// Integer spans selectable machine values without narrowing crypto/subtle behavior.
type Integer int

// Integer_Invariants is for non-secret validation outside constant-time paths.
func Integer_Invariants(value Integer, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), int(INTEGER_MINIMUM), int(INTEGER_MAXIMUM)).
		Ensure()
}

// Byte spans every fixed-width byte equality operand.
type Byte uint8

// Byte_Invariants is for non-secret validation outside constant-time paths.
func Byte_Invariants(value Byte, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(BYTE_MINIMUM), uint8(BYTE_MAXIMUM)).
		Ensure()
}

// Integer_32 spans every fixed-width signed equality operand.
type Integer_32 int32

// Integer_32_Invariants is for non-secret validation outside constant-time paths.
func Integer_32_Invariants(value Integer_32, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), int32(INTEGER_32_MINIMUM), int32(INTEGER_32_MAXIMUM)).
		Ensure()
}

// Nonnegative_Integer is one caller-validated value in crypto/subtle comparison domain.
type Nonnegative_Integer int

// Nonnegative_Integer_Invariants is for non-secret validation outside constant-time paths.
func Nonnegative_Integer_Invariants(
	value Nonnegative_Integer,
	namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), int(NONNEGATIVE_INTEGER_MINIMUM),
			int(NONNEGATIVE_INTEGER_MAXIMUM),
		).
		Ensure()
}

// Source is one bounded secret byte operand.
type Source []byte

// Source_Invariants bounds work while never inspecting secret bytes.
func Source_Invariants(value Source, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Destination is bounded caller-owned output storage.
type Destination []byte

// Destination_Invariants bounds work while never inspecting stored bytes.
func Destination_Invariants(value Destination, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), DESTINATION_SIZE_MINIMUM, DESTINATION_SIZE_MAXIMUM).
		Ensure()
}

// Count is bytes written by XOR_Bytes.
type Count int

// Count_Invariants binds output work to one bounded operand.
func Count_Invariants(value Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Overlap keeps address validation separate from secret contents.
type Overlap bool

// Overlap_Invariants covers both caller-storage relationships.
func Overlap_Invariants(value Overlap, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "XOR destination has partial source overlap.").
		Ensure()
}

// Constant_Time_Compare reports equal contents without branching on contents.
func Constant_Time_Compare(left Source, right Source) (decision Decision) {
	defer func() { Decision_Invariants(decision, "constant_time_compare.decision") }()
	Source_Invariants(left, "constant_time_compare.left")
	Source_Invariants(right, "constant_time_compare.right")
	if len(left) > SOURCE_SIZE_MAXIMUM {
		panic("subtle: source exceeds size bound")
	}
	if len(right) > SOURCE_SIZE_MAXIMUM {
		panic("subtle: source exceeds size bound")
	}
	if len(left) != len(right) {
		return DECISION_FALSE
	}
	difference := BYTE_MINIMUM
	for index := range left {
		difference |= Byte(left[index] ^ right[index])
	}
	difference_32 := uint32(difference)
	return Decision(
		(difference_32 - uint32(binary.UINT_8_SIZE)) >>
			(bits.BIT_COUNT_32_MAXIMUM - binary.UINT_8_SIZE),
	)
}

// Constant_Time_Select selects left for one and right for zero without secret branches.
func Constant_Time_Select(
	selector Decision,
	left Integer,
	right Integer,
) (selected Integer) {
	defer func() { Integer_Invariants(selected, "constant_time_select.selected") }()
	Decision_Invariants(selector, "constant_time_select.selector")
	Integer_Invariants(left, "constant_time_select.left")
	Integer_Invariants(right, "constant_time_select.right")
	mask := Integer(selector) - Integer(DECISION_TRUE)
	return ^mask&left | mask&right
}

// Constant_Time_Byte_Equal reports byte equality without branching on either byte.
func Constant_Time_Byte_Equal(left Byte, right Byte) (decision Decision) {
	defer func() { Decision_Invariants(decision, "constant_time_byte_equal.decision") }()
	Byte_Invariants(left, "constant_time_byte_equal.left")
	Byte_Invariants(right, "constant_time_byte_equal.right")
	difference := uint32(left ^ right)
	return Decision(
		(difference - uint32(binary.UINT_8_SIZE)) >>
			(bits.BIT_COUNT_32_MAXIMUM - binary.UINT_8_SIZE),
	)
}

// Constant_Time_Int_32_Equal reports 32-bit equality without branching on either integer.
func Constant_Time_Integer_32_Equal(left Integer_32, right Integer_32) (decision Decision) {
	defer func() {
		Decision_Invariants(decision, "constant_time_integer_32_equal.decision")
	}()
	Integer_32_Invariants(left, "constant_time_integer_32_equal.left")
	Integer_32_Invariants(right, "constant_time_integer_32_equal.right")
	difference := uint64(uint32(left ^ right))
	return Decision(
		(difference - uint64(binary.UINT_8_SIZE)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE),
	)
}

// Constant_Time_Copy preserves destination for zero and copies source for one.
func Constant_Time_Copy(selector Decision, destination Destination, source Source) {
	Decision_Invariants(selector, "constant_time_copy.selector")
	Destination_Invariants(destination, "constant_time_copy.destination")
	Source_Invariants(source, "constant_time_copy.source")
	if len(destination) > DESTINATION_SIZE_MAXIMUM {
		panic("subtle: destination exceeds size bound")
	}
	if len(source) > SOURCE_SIZE_MAXIMUM {
		panic("subtle: source exceeds size bound")
	}
	aver.Always(
		len(destination) == len(source),
		"Constant-time copy source and destination have equal sizes.",
	)
	if len(destination) != len(source) {
		panic("subtle: slices have different lengths")
	}
	preserve_mask := byte(selector - DECISION_TRUE)
	copy_mask := ^preserve_mask
	for index := range destination {
		destination[index] = destination[index]&preserve_mask | source[index]&copy_mask
	}
}

// Constant_Time_Less_Or_Equal compares caller-validated nonnegative 31-bit operands.
func Constant_Time_Less_Or_Equal(
	left Nonnegative_Integer,
	right Nonnegative_Integer,
) (decision Decision) {
	defer func() { Decision_Invariants(decision, "constant_time_less_or_equal.decision") }()
	Nonnegative_Integer_Invariants(left, "constant_time_less_or_equal.left")
	Nonnegative_Integer_Invariants(right, "constant_time_less_or_equal.right")
	difference := uint32(left) - uint32(right) - uint32(binary.UINT_8_SIZE)
	return Decision(
		difference >> (bits.BIT_COUNT_32_MAXIMUM - binary.UINT_8_SIZE),
	)
}

// XOR_Bytes accepts exact aliases because each byte is read before its output byte is stored.
func XOR_Bytes(destination Destination, left Source, right Source) (count Count) {
	defer func() { Count_Invariants(count, "xor_bytes.count") }()
	Destination_Invariants(destination, "xor_bytes.destination")
	Source_Invariants(left, "xor_bytes.left")
	Source_Invariants(right, "xor_bytes.right")
	if len(destination) > DESTINATION_SIZE_MAXIMUM {
		panic("subtle: destination exceeds size bound")
	}
	if len(left) > SOURCE_SIZE_MAXIMUM {
		panic("subtle: source exceeds size bound")
	}
	if len(right) > SOURCE_SIZE_MAXIMUM {
		panic("subtle: source exceeds size bound")
	}
	count = Count(min(len(left), len(right)))
	if int(count) > len(destination) {
		panic("subtle: destination is too short")
	}
	if bool(inexact_overlap(destination[:count], left[:count])) {
		panic("subtle: destination has partial overlap")
	}
	if bool(inexact_overlap(destination[:count], right[:count])) {
		panic("subtle: destination has partial overlap")
	}
	for index := range int(count) {
		destination[index] = left[index] ^ right[index]
	}
	return count
}

// Address distance rejects partial aliases in constant work independent of secret contents.
func inexact_overlap(left Destination, right Source) (overlap Overlap) {
	defer func() { Overlap_Invariants(overlap, "inexact_overlap.overlap") }()
	Destination_Invariants(left, "inexact_overlap.left")
	Source_Invariants(right, "inexact_overlap.right")
	if len(left) == DESTINATION_SIZE_MINIMUM {
		return Overlap(false)
	}
	if len(right) == SOURCE_SIZE_MINIMUM {
		return Overlap(false)
	}
	left_address := uintptr(unsafe.Pointer(unsafe.SliceData(left)))
	right_address := uintptr(unsafe.Pointer(unsafe.SliceData(right)))
	if left_address == right_address {
		return Overlap(false)
	}
	if left_address < right_address {
		return Overlap(right_address-left_address < uintptr(len(left)))
	}
	return Overlap(left_address-right_address < uintptr(len(right)))
}
