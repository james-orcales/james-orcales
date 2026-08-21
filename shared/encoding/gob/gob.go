// Package gob implements bounded gob scalar framing on caller-owned storage.
package gob

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
)

// PREFIX_SIZE is the signed byte-count prefix before multi-byte integers.
const PREFIX_SIZE = bytes.SLICE_SIZE_MINIMUM + 1

// UNSIGNED_DIRECT_BIT_COUNT leaves the lead bit for negative byte counts.
const UNSIGNED_DIRECT_BIT_COUNT = bits.BIT_COUNT_8_MAXIMUM - 1

// UNSIGNED_DIRECT_MAXIMUM is the largest one-byte unsigned value.
const UNSIGNED_DIRECT_MAXIMUM = 1<<UNSIGNED_DIRECT_BIT_COUNT - 1

// INTEGER_PAYLOAD_SIZE_MAXIMUM follows the unsigned word width.
const INTEGER_PAYLOAD_SIZE_MAXIMUM = bits.BIT_COUNT_64_MAXIMUM /
	bits.BIT_COUNT_8_MAXIMUM

// INTEGER_ENCODED_SIZE_MAXIMUM includes the largest payload prefix.
const INTEGER_ENCODED_SIZE_MAXIMUM = PREFIX_SIZE + INTEGER_PAYLOAD_SIZE_MAXIMUM

// INTEGER_POSITION_MAXIMUM is the byte after the largest truncated payload.
const INTEGER_POSITION_MAXIMUM = INTEGER_ENCODED_SIZE_MAXIMUM

// COMPLEX_REAL_INDEX starts the real component.
const COMPLEX_REAL_INDEX = bytes.SLICE_SIZE_MINIMUM

// COMPLEX_IMAGINARY_INDEX follows the real component.
const COMPLEX_IMAGINARY_INDEX = COMPLEX_REAL_INDEX + 1

// COMPLEX_COMPONENT_COUNT ends after the imaginary component.
const COMPLEX_COMPONENT_COUNT = COMPLEX_IMAGINARY_INDEX + 1

// COMPLEX_ENCODED_SIZE_MINIMUM is two direct unsigned values.
const COMPLEX_ENCODED_SIZE_MINIMUM = PREFIX_SIZE * COMPLEX_COMPONENT_COUNT

// COMPLEX_ENCODED_SIZE_MAXIMUM is two maximum-width unsigned values.
const COMPLEX_ENCODED_SIZE_MAXIMUM = INTEGER_ENCODED_SIZE_MAXIMUM * COMPLEX_COMPONENT_COUNT

// ENCODED_SIZE_MAXIMUM follows the repository byte-slice boundary.
const ENCODED_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// COUNT_PAYLOAD_SIZE_MAXIMUM follows the bounded content count width.
const COUNT_PAYLOAD_SIZE_MAXIMUM = bits.BIT_COUNT_16_MAXIMUM /
	bits.BIT_COUNT_8_MAXIMUM

// COUNT_ENCODED_SIZE_MAXIMUM includes its negative byte-count prefix.
const COUNT_ENCODED_SIZE_MAXIMUM = PREFIX_SIZE + COUNT_PAYLOAD_SIZE_MAXIMUM

// BYTES_SIZE_MAXIMUM leaves room for the largest bounded count prefix.
const BYTES_SIZE_MAXIMUM = ENCODED_SIZE_MAXIMUM - COUNT_ENCODED_SIZE_MAXIMUM

// POSITION_MAXIMUM is the largest truncated bounded bytes position.
const POSITION_MAXIMUM = ENCODED_SIZE_MAXIMUM

// STATUS_OK means the operation completed.
const STATUS_OK = 0

// STATUS_OUTPUT_TOO_SMALL means caller output cannot hold the exact value.
const STATUS_OUTPUT_TOO_SMALL = STATUS_OK + 1

// STATUS_STORAGE_INVALID means byte content overlaps encoded output.
const STATUS_STORAGE_INVALID = STATUS_OUTPUT_TOO_SMALL + 1

// STATUS_INPUT_INVALID means input is truncated or not canonically encoded.
const STATUS_INPUT_INVALID = STATUS_STORAGE_INVALID + 1

// Unsigned is one gob unsigned integer.
type Unsigned uint64

// Unsigned_Invariants covers the complete wire-value domain.
func Unsigned_Invariants(value Unsigned, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Integer is one gob signed integer.
type Integer int64

// Integer_Invariants covers the complete signed wire-value domain.
func Integer_Invariants(value Integer, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Boolean is one gob truth value.
type Boolean bool

// Boolean_Invariants covers false and true wire values.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A gob boolean is true.").
		Ensure()
}

// Boolean_Output excludes unreachable scalar widths from its contract.
type Boolean_Output []byte

// Boolean_Output_Invariants permits refusal storage or its sole encoded byte.
func Boolean_Output_Invariants(value Boolean_Output, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(len(value), bytes.SLICE_SIZE_MINIMUM, PREFIX_SIZE).
		Ensure()
}

// Float_64_Bits preserves every IEEE binary64 encoding without float arithmetic.
type Float_64_Bits uint64

// Float_64_Bits_Invariants covers finite, infinite, and NaN bit patterns.
func Float_64_Bits_Invariants(value Float_64_Bits, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Complex_Encoded must hold both independently sized float components.
type Complex_Encoded []byte

// Complex_Encoded_Invariants follows two fixed-width scalar boundaries.
func Complex_Encoded_Invariants(value Complex_Encoded, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM, COMPLEX_ENCODED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Complex_Output keeps aggregate storage bounded before either component writes.
type Complex_Output []byte

// Complex_Output_Invariants follows two fixed-width scalar boundaries.
func Complex_Output_Invariants(value Complex_Output, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM, COMPLEX_ENCODED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Bytes is bounded caller content or a borrowed decoded value.
type Bytes []byte

// Bytes_Invariants leaves room for its largest count prefix.
func Bytes_Invariants(value Bytes, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, BYTES_SIZE_MAXIMUM).
		Ensure()
}

// Integer_Encoded is one bounded scalar prefix.
type Integer_Encoded []byte

// Integer_Encoded_Invariants follows the fixed word-width encoding.
func Integer_Encoded_Invariants(value Integer_Encoded, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM, INTEGER_ENCODED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Integer_Output is caller-owned scalar storage.
type Integer_Output []byte

// Integer_Output_Invariants follows the fixed word-width encoding.
func Integer_Output_Invariants(value Integer_Output, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM, INTEGER_ENCODED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Bytes_Encoded is bounded count-prefixed input.
type Bytes_Encoded []byte

// Bytes_Encoded_Invariants enforces the shared encoded boundary.
func Bytes_Encoded_Invariants(value Bytes_Encoded, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Bytes_Output is caller-owned count-prefixed storage.
type Bytes_Output []byte

// Bytes_Output_Invariants enforces the shared destination boundary.
func Bytes_Output_Invariants(value Bytes_Output, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Integer_Size_Count is one direct byte or prefix plus payload.
type Integer_Size_Count int

// Integer_Size_Count_Invariants follows the fixed word width.
func Integer_Size_Count_Invariants(
	value Integer_Size_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), PREFIX_SIZE, INTEGER_ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Boolean_Size_Count has one legal value because gob truth encoding is direct.
type Boolean_Size_Count int

// Boolean_Size_Count_Invariants rejects broad integer-size claims.
func Boolean_Size_Count_Invariants(
	value Boolean_Size_Count, _ aver.Namespace,
) {
	aver.Always(
		int(value) == PREFIX_SIZE, "Gob boolean storage is exactly one byte.",
	)
}

// Boolean_Encoded_Count excludes unreachable multi-byte success counts.
type Boolean_Encoded_Count int

// Boolean_Encoded_Count_Invariants includes refusal and exact success.
func Boolean_Encoded_Count_Invariants(
	value Boolean_Encoded_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Int(int(value), bytes.SLICE_SIZE_MINIMUM, PREFIX_SIZE).
		Ensure()
}

// Complex_Size_Count cannot be shorter than two complete scalar values.
type Complex_Size_Count int

// Complex_Size_Count_Invariants follows the sum of two scalar widths.
func Complex_Size_Count_Invariants(
	value Complex_Size_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), COMPLEX_ENCODED_SIZE_MINIMUM, COMPLEX_ENCODED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Complex_Encoded_Count reserves zero for refusal and excludes partial output.
type Complex_Encoded_Count int

// Complex_Encoded_Count_Invariants removes the unreachable one-byte aggregate.
func Complex_Encoded_Count_Invariants(
	value Complex_Encoded_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, COMPLEX_ENCODED_SIZE_MAXIMUM,
			PREFIX_SIZE, PREFIX_SIZE, PREFIX_SIZE, PREFIX_SIZE,
		).
		Ensure()
}

// Complex_Consumed_Count is zero on refusal or both complete components.
type Complex_Consumed_Count int

// Complex_Consumed_Count_Invariants removes partial-component success.
func Complex_Consumed_Count_Invariants(
	value Complex_Consumed_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, COMPLEX_ENCODED_SIZE_MAXIMUM,
			PREFIX_SIZE, PREFIX_SIZE, PREFIX_SIZE, PREFIX_SIZE,
		).
		Ensure()
}

// Complex_Position can identify malformed bytes in either component.
type Complex_Position int

// Complex_Position_Invariants includes success zero and the final truncated byte.
func Complex_Position_Invariants(value Complex_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, COMPLEX_ENCODED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Integer_Encoded_Count is zero on refusal or exact scalar bytes.
type Integer_Encoded_Count int

// Integer_Encoded_Count_Invariants includes every scalar encoding size.
func Integer_Encoded_Count_Invariants(
	value Integer_Encoded_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, INTEGER_ENCODED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Integer_Consumed_Count is zero on refusal or one scalar prefix.
type Integer_Consumed_Count int

// Integer_Consumed_Count_Invariants follows fixed scalar input.
func Integer_Consumed_Count_Invariants(
	value Integer_Consumed_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, INTEGER_ENCODED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Integer_Position is one-based invalid scalar input or zero.
type Integer_Position int

// Integer_Position_Invariants includes every truncated scalar boundary.
func Integer_Position_Invariants(value Integer_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, INTEGER_POSITION_MAXIMUM).
		Ensure()
}

// Bytes_Size_Count is exact bounded count prefix plus content.
type Bytes_Size_Count int

// Bytes_Size_Count_Invariants excludes zero because even empty content has a count.
func Bytes_Size_Count_Invariants(value Bytes_Size_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), PREFIX_SIZE, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Bytes_Encoded_Count is zero on refusal or exact encoded bytes.
type Bytes_Encoded_Count int

// Bytes_Encoded_Count_Invariants includes every bounded result.
func Bytes_Encoded_Count_Invariants(value Bytes_Encoded_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Bytes_Consumed_Count is zero on refusal or one count-prefixed value.
type Bytes_Consumed_Count int

// Bytes_Consumed_Count_Invariants includes every bounded input prefix.
func Bytes_Consumed_Count_Invariants(
	value Bytes_Consumed_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Bytes_Position is one-based invalid bytes input or zero.
type Bytes_Position int

// Bytes_Position_Invariants includes unexpected end after maximum input.
func Bytes_Position_Invariants(value Bytes_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, POSITION_MAXIMUM).
		Ensure()
}

// Integer_Encode_Status excludes overlap because scalar values own no storage.
type Integer_Encode_Status uint8

// Integer_Encode_Status_Invariants lists both scalar encode outcomes.
func Integer_Encode_Status_Invariants(
	value Integer_Encode_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), STATUS_OK, STATUS_OUTPUT_TOO_SMALL).
		Ensure()
}

// Bytes_Encode_Status adds caller-storage overlap refusal.
type Bytes_Encode_Status uint8

// Bytes_Encode_Status_Invariants lists every bytes encode outcome.
func Bytes_Encode_Status_Invariants(
	value Bytes_Encode_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), STATUS_OK, STATUS_OUTPUT_TOO_SMALL, STATUS_STORAGE_INVALID,
		).
		Ensure()
}

// Decode_Status lists canonical success or malformed input.
type Decode_Status uint8

// Decode_Status_Invariants excludes encode-only refusals.
func Decode_Status_Invariants(value Decode_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), STATUS_OK, STATUS_INPUT_INVALID).
		Ensure()
}

// Unsigned_Size reports exact canonical gob scalar storage.
func Unsigned_Size(value Unsigned) (size Integer_Size_Count) {
	defer func() { Integer_Size_Count_Invariants(size, "Unsigned_Size.size") }()
	Unsigned_Invariants(value, "Unsigned_Size.value")
	if value <= UNSIGNED_DIRECT_MAXIMUM {
		return PREFIX_SIZE
	}
	size = PREFIX_SIZE
	value_tail := value
	for value_tail > 0 {
		size++
		value_tail >>= bits.BIT_COUNT_8_MAXIMUM
	}
	return size
}

// Integer_Size reports exact storage after signed-to-unsigned mapping.
func Integer_Size(value Integer) (size Integer_Size_Count) {
	defer func() { Integer_Size_Count_Invariants(size, "Integer_Size.size") }()
	Integer_Invariants(value, "Integer_Size.value")
	return Unsigned_Size(integer_unsigned(value))
}

// Encode_Unsigned_Into rejects short output before changing caller storage.
func Encode_Unsigned_Into(destination Integer_Output, value Unsigned) (
	count Integer_Encoded_Count, status Integer_Encode_Status,
) {
	defer func() {
		Integer_Encoded_Count_Invariants(count, "Encode_Unsigned_Into.count")
		Integer_Encode_Status_Invariants(status, "Encode_Unsigned_Into.status")
	}()
	Integer_Output_Invariants(destination, "Encode_Unsigned_Into.destination")
	Unsigned_Invariants(value, "Encode_Unsigned_Into.value")
	required := Unsigned_Size(value)
	if len(destination) < int(required) {
		return 0, STATUS_OUTPUT_TOO_SMALL
	}
	encode_unsigned_unchecked(destination[:required], value)
	return Integer_Encoded_Count(required), STATUS_OK
}

// Encode_Integer_Into maps sign without intermediate storage.
func Encode_Integer_Into(destination Integer_Output, value Integer) (
	count Integer_Encoded_Count, status Integer_Encode_Status,
) {
	defer func() {
		Integer_Encoded_Count_Invariants(count, "Encode_Integer_Into.count")
		Integer_Encode_Status_Invariants(status, "Encode_Integer_Into.status")
	}()
	Integer_Output_Invariants(destination, "Encode_Integer_Into.destination")
	Integer_Invariants(value, "Encode_Integer_Into.value")
	unsigned := integer_unsigned(value)
	required := Unsigned_Size(unsigned)
	if len(destination) < int(required) {
		return 0, STATUS_OUTPUT_TOO_SMALL
	}
	encode_unsigned_unchecked(destination[:required], unsigned)
	return Integer_Encoded_Count(required), STATUS_OK
}

// Decode_Unsigned consumes one canonical scalar and leaves trailing input untouched.
func Decode_Unsigned(source Integer_Encoded) (
	value Unsigned, consumed Integer_Consumed_Count,
	position Integer_Position, status Decode_Status,
) {
	defer func() {
		Unsigned_Invariants(value, "Decode_Unsigned.value")
		Integer_Consumed_Count_Invariants(consumed, "Decode_Unsigned.consumed")
		Integer_Position_Invariants(position, "Decode_Unsigned.position")
		Decode_Status_Invariants(status, "Decode_Unsigned.status")
	}()
	Integer_Encoded_Invariants(source, "Decode_Unsigned.source")
	if len(source) == bytes.SLICE_SIZE_MINIMUM {
		return 0, 0, 1, STATUS_INPUT_INVALID
	}
	first := source[bytes.SLICE_SIZE_MINIMUM]
	if first <= UNSIGNED_DIRECT_MAXIMUM {
		return Unsigned(first), PREFIX_SIZE, 0, STATUS_OK
	}
	payload_size := -int(int8(first))
	if payload_size > INTEGER_PAYLOAD_SIZE_MAXIMUM {
		return 0, 0, 1, STATUS_INPUT_INVALID
	}
	if payload_size > len(source)-PREFIX_SIZE {
		return 0, 0, Integer_Position(len(source) + 1), STATUS_INPUT_INVALID
	}
	for index := PREFIX_SIZE; index < PREFIX_SIZE+payload_size; index++ {
		value = value<<bits.BIT_COUNT_8_MAXIMUM | Unsigned(source[index])
	}
	return value, Integer_Consumed_Count(PREFIX_SIZE + payload_size), 0, STATUS_OK
}

// Decode_Integer reverses sign mapping after canonical unsigned decoding.
func Decode_Integer(source Integer_Encoded) (
	value Integer, consumed Integer_Consumed_Count,
	position Integer_Position, status Decode_Status,
) {
	defer func() {
		Integer_Invariants(value, "Decode_Integer.value")
		Integer_Consumed_Count_Invariants(consumed, "Decode_Integer.consumed")
		Integer_Position_Invariants(position, "Decode_Integer.position")
		Decode_Status_Invariants(status, "Decode_Integer.status")
	}()
	Integer_Encoded_Invariants(source, "Decode_Integer.source")
	unsigned, consumed, position, status := Decode_Unsigned(source)
	if status != STATUS_OK {
		return 0, 0, position, STATUS_INPUT_INVALID
	}
	if unsigned&1 == 0 {
		return Integer(unsigned >> 1), consumed, 0, STATUS_OK
	}
	return Integer(^(unsigned >> 1)), consumed, 0, STATUS_OK
}

// Boolean_Size reports one canonical unsigned truth byte.
func Boolean_Size(value Boolean) (size Boolean_Size_Count) {
	defer func() { Boolean_Size_Count_Invariants(size, "Boolean_Size.size") }()
	Boolean_Invariants(value, "Boolean_Size.value")
	return PREFIX_SIZE
}

// Encode_Boolean_Into writes one unsigned truth value into caller storage.
func Encode_Boolean_Into(destination Boolean_Output, value Boolean) (
	count Boolean_Encoded_Count, status Integer_Encode_Status,
) {
	defer func() {
		Boolean_Encoded_Count_Invariants(count, "Encode_Boolean_Into.count")
		Integer_Encode_Status_Invariants(status, "Encode_Boolean_Into.status")
	}()
	Boolean_Output_Invariants(destination, "Encode_Boolean_Into.destination")
	Boolean_Invariants(value, "Encode_Boolean_Into.value")
	if len(destination) < PREFIX_SIZE {
		return 0, STATUS_OUTPUT_TOO_SMALL
	}
	destination[bytes.SLICE_SIZE_MINIMUM] = 0
	if value {
		destination[bytes.SLICE_SIZE_MINIMUM] = 1
	}
	return PREFIX_SIZE, STATUS_OK
}

// Decode_Boolean maps every canonical nonzero unsigned value to true.
func Decode_Boolean(source Integer_Encoded) (
	value Boolean, consumed Integer_Consumed_Count,
	position Integer_Position, status Decode_Status,
) {
	defer func() {
		Boolean_Invariants(value, "Decode_Boolean.value")
		Integer_Consumed_Count_Invariants(consumed, "Decode_Boolean.consumed")
		Integer_Position_Invariants(position, "Decode_Boolean.position")
		Decode_Status_Invariants(status, "Decode_Boolean.status")
	}()
	Integer_Encoded_Invariants(source, "Decode_Boolean.source")
	unsigned, consumed, position, status := Decode_Unsigned(source)
	if status != STATUS_OK {
		return false, 0, position, STATUS_INPUT_INVALID
	}
	return Boolean(unsigned != 0), consumed, 0, STATUS_OK
}

// Float_Size reports gob storage after byte-reversing IEEE bits.
func Float_Size(value Float_64_Bits) (size Integer_Size_Count) {
	defer func() { Integer_Size_Count_Invariants(size, "Float_Size.size") }()
	Float_64_Bits_Invariants(value, "Float_Size.value")
	return Unsigned_Size(float_unsigned(value))
}

// Encode_Float_Into writes byte-reversed IEEE bits as one gob unsigned value.
func Encode_Float_Into(destination Integer_Output, value Float_64_Bits) (
	count Integer_Encoded_Count, status Integer_Encode_Status,
) {
	defer func() {
		Integer_Encoded_Count_Invariants(count, "Encode_Float_Into.count")
		Integer_Encode_Status_Invariants(status, "Encode_Float_Into.status")
	}()
	Integer_Output_Invariants(destination, "Encode_Float_Into.destination")
	Float_64_Bits_Invariants(value, "Encode_Float_Into.value")
	return Encode_Unsigned_Into(destination, float_unsigned(value))
}

// Decode_Float restores IEEE bits from one byte-reversed gob unsigned value.
func Decode_Float(source Integer_Encoded) (
	value Float_64_Bits, consumed Integer_Consumed_Count,
	position Integer_Position, status Decode_Status,
) {
	defer func() {
		Float_64_Bits_Invariants(value, "Decode_Float.value")
		Integer_Consumed_Count_Invariants(consumed, "Decode_Float.consumed")
		Integer_Position_Invariants(position, "Decode_Float.position")
		Decode_Status_Invariants(status, "Decode_Float.status")
	}()
	Integer_Encoded_Invariants(source, "Decode_Float.source")
	unsigned, consumed, position, status := Decode_Unsigned(source)
	if status != STATUS_OK {
		return 0, 0, position, STATUS_INPUT_INVALID
	}
	reversed := bits.Reverse_Bytes_64(bits.Word_64(unsigned))
	return Float_64_Bits(reversed), consumed, 0, STATUS_OK
}

// Complex_Size keeps component lengths independent because gob concatenates their frames.
func Complex_Size(
	real_bits Float_64_Bits, imaginary_bits Float_64_Bits,
) (size Complex_Size_Count) {
	defer func() { Complex_Size_Count_Invariants(size, "Complex_Size.size") }()
	Float_64_Bits_Invariants(real_bits, "Complex_Size.real_bits")
	Float_64_Bits_Invariants(imaginary_bits, "Complex_Size.imaginary_bits")
	return Complex_Size_Count(Float_Size(real_bits)) +
		Complex_Size_Count(Float_Size(imaginary_bits))
}

// Encode_Complex_Into checks aggregate capacity before either scalar can mutate output.
func Encode_Complex_Into(
	destination Complex_Output,
	real_bits Float_64_Bits, imaginary_bits Float_64_Bits,
) (count Complex_Encoded_Count, status Integer_Encode_Status) {
	defer func() {
		Complex_Encoded_Count_Invariants(count, "Encode_Complex_Into.count")
		Integer_Encode_Status_Invariants(status, "Encode_Complex_Into.status")
	}()
	Complex_Output_Invariants(destination, "Encode_Complex_Into.destination")
	Float_64_Bits_Invariants(real_bits, "Encode_Complex_Into.real_bits")
	Float_64_Bits_Invariants(imaginary_bits, "Encode_Complex_Into.imaginary_bits")
	real_unsigned := float_unsigned(real_bits)
	imaginary_unsigned := float_unsigned(imaginary_bits)
	real_size := Unsigned_Size(real_unsigned)
	imaginary_size := Unsigned_Size(imaginary_unsigned)
	required := Complex_Size_Count(real_size) + Complex_Size_Count(imaginary_size)
	if len(destination) < int(required) {
		return 0, STATUS_OUTPUT_TOO_SMALL
	}
	encode_unsigned_unchecked(destination[:real_size], real_unsigned)
	encode_unsigned_unchecked(destination[real_size:required], imaginary_unsigned)
	return Complex_Encoded_Count(required), STATUS_OK
}

// Decode_Complex reports second-component failures at aggregate byte positions.
func Decode_Complex(source Complex_Encoded) (
	real_bits Float_64_Bits, imaginary_bits Float_64_Bits,
	consumed Complex_Consumed_Count, position Complex_Position, status Decode_Status,
) {
	defer func() {
		Float_64_Bits_Invariants(real_bits, "Decode_Complex.real_bits")
		Float_64_Bits_Invariants(imaginary_bits, "Decode_Complex.imaginary_bits")
		Complex_Consumed_Count_Invariants(consumed, "Decode_Complex.consumed")
		Complex_Position_Invariants(position, "Decode_Complex.position")
		Decode_Status_Invariants(status, "Decode_Complex.status")
	}()
	Complex_Encoded_Invariants(source, "Decode_Complex.source")
	real_end_count := len(source)
	if real_end_count > INTEGER_ENCODED_SIZE_MAXIMUM {
		real_end_count = INTEGER_ENCODED_SIZE_MAXIMUM
	}
	real_bits, real_count, real_position, real_status := Decode_Float(
		Integer_Encoded(source[:real_end_count]),
	)
	if real_status != STATUS_OK {
		return 0, 0, 0, Complex_Position(real_position), STATUS_INPUT_INVALID
	}
	imaginary_source := source[real_count:]
	imaginary_end_count := len(imaginary_source)
	if imaginary_end_count > INTEGER_ENCODED_SIZE_MAXIMUM {
		imaginary_end_count = INTEGER_ENCODED_SIZE_MAXIMUM
	}
	imaginary_bits, imaginary_count, imaginary_position, imaginary_status :=
		Decode_Float(Integer_Encoded(imaginary_source[:imaginary_end_count]))
	if imaginary_status != STATUS_OK {
		return 0, 0, 0,
			Complex_Position(int(real_count) + int(imaginary_position)),
			STATUS_INPUT_INVALID
	}
	consumed = Complex_Consumed_Count(int(real_count) + int(imaginary_count))
	return real_bits, imaginary_bits, consumed, 0, STATUS_OK
}

// Bytes_Size reports exact count prefix plus borrowed content.
func Bytes_Size(source Bytes) (size Bytes_Size_Count) {
	defer func() { Bytes_Size_Count_Invariants(size, "Bytes_Size.size") }()
	Bytes_Invariants(source, "Bytes_Size.source")
	return Bytes_Size_Count(Unsigned_Size(Unsigned(len(source)))) +
		Bytes_Size_Count(len(source))
}

// Encode_Bytes_Into rejects capacity and overlap before caller mutation.
func Encode_Bytes_Into(destination Bytes_Output, source Bytes) (
	count Bytes_Encoded_Count, status Bytes_Encode_Status,
) {
	defer func() {
		Bytes_Encoded_Count_Invariants(count, "Encode_Bytes_Into.count")
		Bytes_Encode_Status_Invariants(status, "Encode_Bytes_Into.status")
	}()
	Bytes_Output_Invariants(destination, "Encode_Bytes_Into.destination")
	Bytes_Invariants(source, "Encode_Bytes_Into.source")
	if bytes.Overlap(bytes.Slice(destination), bytes.Slice(source)) {
		return 0, STATUS_STORAGE_INVALID
	}
	required := Bytes_Size(source)
	if len(destination) < int(required) {
		return 0, STATUS_OUTPUT_TOO_SMALL
	}
	prefix_size := Unsigned_Size(Unsigned(len(source)))
	encode_unsigned_unchecked(destination[:prefix_size], Unsigned(len(source)))
	copy(destination[prefix_size:], source)
	return Bytes_Encoded_Count(required), STATUS_OK
}

// Decode_Bytes consumes one bounded count-prefixed value and borrows its content.
func Decode_Bytes(source Bytes_Encoded) (
	value Bytes, consumed Bytes_Consumed_Count,
	position Bytes_Position, status Decode_Status,
) {
	defer func() {
		Bytes_Invariants(value, "Decode_Bytes.value")
		Bytes_Consumed_Count_Invariants(consumed, "Decode_Bytes.consumed")
		Bytes_Position_Invariants(position, "Decode_Bytes.position")
		Decode_Status_Invariants(status, "Decode_Bytes.status")
	}()
	Bytes_Encoded_Invariants(source, "Decode_Bytes.source")
	prefix_end_count := len(source)
	if prefix_end_count > INTEGER_ENCODED_SIZE_MAXIMUM {
		prefix_end_count = INTEGER_ENCODED_SIZE_MAXIMUM
	}
	size, prefix_size, integer_position, integer_status := Decode_Unsigned(
		Integer_Encoded(source[:prefix_end_count]),
	)
	if integer_status != STATUS_OK {
		return nil, 0, Bytes_Position(integer_position), STATUS_INPUT_INVALID
	}
	if size > BYTES_SIZE_MAXIMUM {
		return nil, 0, Bytes_Position(prefix_size), STATUS_INPUT_INVALID
	}
	content_start := int(prefix_size)
	if int(size) > len(source)-content_start {
		return nil, 0, Bytes_Position(len(source) + 1), STATUS_INPUT_INVALID
	}
	end_index := content_start + int(size)
	return Bytes(source[content_start:end_index]), Bytes_Consumed_Count(end_index), 0, STATUS_OK
}

func integer_unsigned(value Integer) (unsigned Unsigned) {
	defer func() { Unsigned_Invariants(unsigned, "integer_unsigned.unsigned") }()
	Integer_Invariants(value, "integer_unsigned.value")
	unsigned = Unsigned(uint64(value) << 1)
	if value < 0 {
		unsigned = ^unsigned
	}
	return unsigned
}

func float_unsigned(value Float_64_Bits) (unsigned Unsigned) {
	defer func() { Unsigned_Invariants(unsigned, "float_unsigned.unsigned") }()
	Float_64_Bits_Invariants(value, "float_unsigned.value")
	return Unsigned(bits.Reverse_Bytes_64(bits.Word_64(value)))
}

func encode_unsigned_unchecked[Destination ~[]byte](
	destination Destination, value Unsigned,
) {
	Unsigned_Invariants(value, "encode_unsigned_unchecked.value")
	if value <= UNSIGNED_DIRECT_MAXIMUM {
		destination[bytes.SLICE_SIZE_MINIMUM] = byte(value)
		return
	}
	payload_size := len(destination) - PREFIX_SIZE
	destination[bytes.SLICE_SIZE_MINIMUM] = byte(-payload_size)
	value_tail := value
	for index := len(destination) - 1; index >= PREFIX_SIZE; index-- {
		destination[index] = byte(value_tail)
		value_tail >>= bits.BIT_COUNT_8_MAXIMUM
	}
}
