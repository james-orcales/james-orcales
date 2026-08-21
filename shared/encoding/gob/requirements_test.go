package gob_test

import (
	"testing"

	"local/james-orcales/shared/encoding/gob"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Integer protects gob's complement-based signed mapping and scalar boundaries.
func Test_Integer(t *testing.T) {
	var destination [gob.INTEGER_ENCODED_SIZE_MAXIMUM]byte
	tests := [...]struct {
		Value    gob.Integer
		Expected []byte
	}{
		{0, []byte{0}},
		{1, []byte{2}},
		{2, []byte{4}},
		{-1, []byte{1}},
		{64, []byte{0xff, 0x80}},
		{-65, []byte{0xff, 0x81}},
		{gob.Integer(bits.INTEGER_64_MAXIMUM), []byte{
			0xf8, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfe,
		}},
		{gob.Integer(bits.INTEGER_64_MINIMUM), []byte{
			0xf8, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		}},
	}
	for _, one := range tests {
		size := gob.Integer_Size(one.Value)
		testify.Equal(t, gob.Integer_Size_Count(len(one.Expected)), size)
		count, status := gob.Encode_Integer_Into(destination[:size], one.Value)
		testify.Equal_Values(t, gob.STATUS_OK, status)
		testify.Equal(t, one.Expected, destination[:count])
		value, consumed, position, decode_status := gob.Decode_Integer(destination[:count])
		testify.Equal_Values(t, gob.STATUS_OK, decode_status)
		testify.Equal(t, one.Value, value)
		testify.Equal(t, gob.Integer_Consumed_Count(count), consumed)
		testify.Equal(t, gob.Integer_Position(0), position)
	}
}

// Test_Boolean protects gob unsigned truth encoding and nonzero decoding.
func Test_Boolean(t *testing.T) {
	var destination [gob.INTEGER_ENCODED_SIZE_MAXIMUM]byte
	for _, one := range [...]struct {
		Value    gob.Boolean
		Expected byte
	}{
		{false, 0},
		{true, 1},
	} {
		size := gob.Boolean_Size(one.Value)
		testify.Equal(t, gob.Boolean_Size_Count(1), size)
		count, status := gob.Encode_Boolean_Into(destination[:size], one.Value)
		testify.Equal_Values(t, gob.STATUS_OK, status)
		testify.Equal(t, one.Expected, destination[0])
		value, consumed, position, decode_status := gob.Decode_Boolean(destination[:count])
		testify.Equal_Values(t, gob.STATUS_OK, decode_status)
		testify.Equal(t, one.Value, value)
		testify.Equal(t, gob.Integer_Consumed_Count(count), consumed)
		testify.Equal(t, gob.Integer_Position(0), position)
	}
	value, _, _, status := gob.Decode_Boolean(gob.Integer_Encoded{2})
	testify.Equal_Values(t, gob.STATUS_OK, status)
	testify.Equal(t, gob.Boolean(true), value)
}

// Test_Float protects byte-reversed IEEE bits used by gob floating-point values.
func Test_Float(t *testing.T) {
	var destination [gob.INTEGER_ENCODED_SIZE_MAXIMUM]byte
	tests := [...]struct {
		Value    gob.Float_64_Bits
		Expected []byte
	}{
		{0, []byte{0}},
		{TEST_FLOAT_64_ONE_BITS, []byte{0xfe, 0xf0, 0x3f}},
		{TEST_FLOAT_64_TWO_BITS, []byte{0x40}},
	}
	for _, one := range tests {
		size := gob.Float_Size(one.Value)
		testify.Equal(t, gob.Integer_Size_Count(len(one.Expected)), size)
		count, status := gob.Encode_Float_Into(destination[:size], one.Value)
		testify.Equal_Values(t, gob.STATUS_OK, status)
		testify.Equal(t, one.Expected, destination[:count])
		value, consumed, position, decode_status := gob.Decode_Float(destination[:count])
		testify.Equal_Values(t, gob.STATUS_OK, decode_status)
		testify.Equal(t, one.Value, value)
		testify.Equal(t, gob.Integer_Consumed_Count(count), consumed)
		testify.Equal(t, gob.Integer_Position(0), position)
	}
}

// Test_Complex protects real-before-imaginary framing and independent scalar lengths.
func Test_Complex(t *testing.T) {
	var destination [gob.COMPLEX_ENCODED_SIZE_MAXIMUM]byte
	real_bits := TEST_FLOAT_64_ONE_BITS
	imaginary_bits := TEST_FLOAT_64_TWO_BITS
	expected := []byte{0xfe, 0xf0, 0x3f, 0x40}
	size := gob.Complex_Size(real_bits, imaginary_bits)
	testify.Equal(t, gob.Complex_Size_Count(len(expected)), size)
	count, status := gob.Encode_Complex_Into(
		destination[:size], real_bits, imaginary_bits,
	)
	testify.Equal_Values(t, gob.STATUS_OK, status)
	testify.Equal(t, expected, destination[:count])
	actual_real, actual_imaginary, consumed, position, decode_status :=
		gob.Decode_Complex(gob.Complex_Encoded(destination[:count]))
	testify.Equal_Values(t, gob.STATUS_OK, decode_status)
	testify.Equal(t, real_bits, actual_real)
	testify.Equal(t, imaginary_bits, actual_imaginary)
	testify.Equal(t, gob.Complex_Consumed_Count(count), consumed)
	testify.Equal(t, gob.Complex_Position(0), position)
}

// Test_Bytes protects count framing and borrowed content.
func Test_Bytes(t *testing.T) {
	var destination [gob.ENCODED_SIZE_MAXIMUM]byte
	source := gob.Bytes("gob")
	size := gob.Bytes_Size(source)
	testify.Equal(t, gob.Bytes_Size_Count(len(source)+1), size)
	count, status := gob.Encode_Bytes_Into(destination[:size], source)
	testify.Equal_Values(t, gob.STATUS_OK, status)
	testify.Equal(t, []byte{3, 'g', 'o', 'b'}, destination[:count])
	destination[count] = 0
	decoded, consumed, position, decode_status := gob.Decode_Bytes(destination[:count+1])
	testify.Equal_Values(t, gob.STATUS_OK, decode_status)
	testify.Equal(t, source, decoded)
	testify.Equal(t, gob.Bytes_Consumed_Count(count), consumed)
	testify.Equal(t, gob.Bytes_Position(0), position)
}

// Test_Bounds protects canonical decoding and every caller-controlled size.
func Test_Bounds(t *testing.T) {
	test_unsigned(t)
	test_wide_unsigned(t)
	test_invalid(t)
	test_refusals(t)
	test_domains(t)
}

// Test_Allocation protects every status and maximum-size path from heap ownership.
func Test_Allocation(t *testing.T) {
	test_integer_size_allocation(t)
	test_integer_encode_allocation(t)
	test_integer_decode_allocation(t)
	test_boolean_allocation(t)
	test_float_allocation(t)
	test_complex_allocation(t)
	test_bytes_allocation(t)
}

const TEST_FLOAT_64_SIGN_BIT_COUNT = gob.PREFIX_SIZE
const TEST_FLOAT_64_EXPONENT_BIT_COUNT = 11
const TEST_FLOAT_64_BASE = gob.PREFIX_SIZE + gob.PREFIX_SIZE
const TEST_FLOAT_64_MANTISSA_BIT_COUNT = bits.BIT_COUNT_64_MAXIMUM -
	TEST_FLOAT_64_SIGN_BIT_COUNT - TEST_FLOAT_64_EXPONENT_BIT_COUNT
const TEST_FLOAT_64_EXPONENT_COUNT = TEST_FLOAT_64_BASE <<
	(TEST_FLOAT_64_EXPONENT_BIT_COUNT - gob.PREFIX_SIZE)
const TEST_FLOAT_64_EXPONENT_BIAS = TEST_FLOAT_64_EXPONENT_COUNT/
	TEST_FLOAT_64_BASE - gob.PREFIX_SIZE

const TEST_FLOAT_64_ONE_BITS = gob.Float_64_Bits(
	uint64(TEST_FLOAT_64_EXPONENT_BIAS) << TEST_FLOAT_64_MANTISSA_BIT_COUNT,
)

const TEST_FLOAT_64_TWO_BITS = gob.Float_64_Bits(
	uint64(TEST_FLOAT_64_EXPONENT_BIAS+gob.PREFIX_SIZE) << TEST_FLOAT_64_MANTISSA_BIT_COUNT,
)

func test_unsigned(t *testing.T) {
	var destination [gob.INTEGER_ENCODED_SIZE_MAXIMUM]byte
	tests := [...]struct {
		Value    gob.Unsigned
		Expected []byte
	}{
		{0, []byte{0}},
		{gob.UNSIGNED_DIRECT_MAXIMUM, []byte{0x7f}},
		{gob.UNSIGNED_DIRECT_MAXIMUM + 1, []byte{0xff, 0x80}},
		{1 << bits.BIT_COUNT_8_MAXIMUM, []byte{0xfe, 0x01, 0x00}},
		{gob.Unsigned(bits.WORD_64_MAXIMUM), []byte{
			0xf8, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		}},
	}
	for _, one := range tests {
		size := gob.Unsigned_Size(one.Value)
		testify.Equal(t, gob.Integer_Size_Count(len(one.Expected)), size)
		count, status := gob.Encode_Unsigned_Into(destination[:size], one.Value)
		testify.Equal_Values(t, gob.STATUS_OK, status)
		testify.Equal(t, one.Expected, destination[:count])
		value, consumed, _, decode_status := gob.Decode_Unsigned(destination[:count])
		testify.Equal_Values(t, gob.STATUS_OK, decode_status)
		testify.Equal(t, one.Value, value)
		testify.Equal(t, gob.Integer_Consumed_Count(count), consumed)
	}
}

func test_wide_unsigned(t *testing.T) {
	for _, one := range [...]struct {
		Source   gob.Integer_Encoded
		Unsigned gob.Unsigned
		Integer  gob.Integer
	}{
		{gob.Integer_Encoded{0xff, 0}, 0, 0},
		{gob.Integer_Encoded{0xff, 1}, 1, -1},
		{gob.Integer_Encoded{0xfe, 0, 0x80}, gob.UNSIGNED_DIRECT_MAXIMUM + 1, 64},
	} {
		value, consumed, position, status := gob.Decode_Unsigned(one.Source)
		testify.Equal_Values(t, gob.STATUS_OK, status)
		testify.Equal(t, one.Unsigned, value)
		testify.Equal(t, gob.Integer_Consumed_Count(len(one.Source)), consumed)
		testify.Equal(t, gob.Integer_Position(0), position)
		integer, consumed, position, status := gob.Decode_Integer(one.Source)
		testify.Equal_Values(t, gob.STATUS_OK, status)
		testify.Equal(t, one.Integer, integer)
		testify.Equal(t, gob.Integer_Consumed_Count(len(one.Source)), consumed)
		testify.Equal(t, gob.Integer_Position(0), position)
	}
}

func test_invalid(t *testing.T) {
	integer_tests := [...]struct {
		Source   gob.Integer_Encoded
		Position gob.Integer_Position
	}{
		{nil, 1},
		{gob.Integer_Encoded{0xf7}, 1},
		{gob.Integer_Encoded{0xff}, 2},
		{gob.Integer_Encoded{0xf8, 1, 1, 1, 1, 1, 1, 1}, 9},
	}
	for _, one := range integer_tests {
		_, _, position, status := gob.Decode_Unsigned(one.Source)
		testify.Equal_Values(t, gob.STATUS_INPUT_INVALID, status)
		testify.Equal(t, one.Position, position)
		_, _, position, status = gob.Decode_Integer(one.Source)
		testify.Equal_Values(t, gob.STATUS_INPUT_INVALID, status)
		testify.Equal(t, one.Position, position)
	}
	_, _, position, status := gob.Decode_Bytes(gob.Bytes_Encoded{0xff})
	testify.Equal_Values(t, gob.STATUS_INPUT_INVALID, status)
	testify.Equal(t, gob.Bytes_Position(2), position)
	_, _, position, status = gob.Decode_Bytes(gob.Bytes_Encoded{4, 'a', 'b', 'c'})
	testify.Equal_Values(t, gob.STATUS_INPUT_INVALID, status)
	testify.Equal(t, gob.Bytes_Position(5), position)
	_, _, position, status = gob.Decode_Bytes(gob.Bytes_Encoded{0xfe, 0x0f, 0xfe})
	testify.Equal_Values(t, gob.STATUS_INPUT_INVALID, status)
	testify.Equal(t, gob.Bytes_Position(3), position)
	var maximum_position [gob.ENCODED_SIZE_MAXIMUM - 1]byte
	maximum_position[0] = 0xfe
	maximum_position[1] = 0x0f
	maximum_position[2] = 0xfd
	_, _, position, status = gob.Decode_Bytes(maximum_position[:])
	testify.Equal_Values(t, gob.STATUS_INPUT_INVALID, status)
	testify.Equal(t, gob.Bytes_Position(gob.POSITION_MAXIMUM), position)
}

func test_refusals(t *testing.T) {
	var oversized_integer [gob.INTEGER_ENCODED_SIZE_MAXIMUM + 1]byte
	var oversized_encoded [gob.ENCODED_SIZE_MAXIMUM + 1]byte
	var oversized_bytes [gob.BYTES_SIZE_MAXIMUM + 1]byte
	testify.Panics(t, func() { gob.Decode_Unsigned(oversized_integer[:]) })
	testify.Panics(t, func() { gob.Decode_Bytes(oversized_encoded[:]) })
	testify.Panics(t, func() { gob.Bytes_Size(oversized_bytes[:]) })
	var integer_destination [gob.INTEGER_ENCODED_SIZE_MAXIMUM]byte
	_, integer_status := gob.Encode_Unsigned_Into(integer_destination[:0], 0)
	testify.Equal_Values(t, gob.STATUS_OUTPUT_TOO_SMALL, integer_status)
	var destination [gob.ENCODED_SIZE_MAXIMUM]byte
	_, bytes_status := gob.Encode_Bytes_Into(destination[:0], nil)
	testify.Equal_Values(t, gob.STATUS_OUTPUT_TOO_SMALL, bytes_status)
	overlap := gob.Bytes(destination[:1])
	_, bytes_status = gob.Encode_Bytes_Into(destination[:], overlap)
	testify.Equal_Values(t, gob.STATUS_STORAGE_INVALID, bytes_status)
}

func test_domains(t *testing.T) {
	var integer_destination [gob.INTEGER_ENCODED_SIZE_MAXIMUM]byte
	for _, value := range [...]gob.Unsigned{0, 1, 2, gob.Unsigned(bits.WORD_64_MAXIMUM)} {
		gob.Encode_Unsigned_Into(integer_destination[:], value)
	}
	for _, value := range [...]gob.Integer{
		gob.Integer(bits.INTEGER_64_MINIMUM), gob.Integer(bits.INTEGER_64_MINIMUM + 1),
		0, 1, 2, gob.Integer(bits.INTEGER_64_MAXIMUM),
	} {
		gob.Encode_Integer_Into(integer_destination[:], value)
	}
	for _, source := range [...]gob.Integer_Encoded{
		nil, {0}, {1}, {2}, {4},
		{0xf8, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfe},
		{0xf8, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
	} {
		gob.Decode_Unsigned(source)
		gob.Decode_Integer(source)
	}
	test_boolean_domains(integer_destination[:])
	test_float_domains(integer_destination[:])
	var complex_destination [gob.COMPLEX_ENCODED_SIZE_MAXIMUM]byte
	test_complex_domains(complex_destination[:])
	var destination [gob.ENCODED_SIZE_MAXIMUM]byte
	for _, source := range [...]gob.Bytes{nil, {'a'}, {'a', 'b'}} {
		gob.Bytes_Size(source)
		gob.Encode_Bytes_Into(destination[:len(source)+1], source)
	}
	for _, source := range [...]gob.Bytes_Encoded{nil, {0}, {1, 'a'}, {2, 'a', 'b'}} {
		gob.Decode_Bytes(source)
	}
	maximum := gob.Bytes(make([]byte, gob.BYTES_SIZE_MAXIMUM))
	count, status := gob.Encode_Bytes_Into(destination[:], maximum)
	testify.Equal(t, gob.Bytes_Encoded_Count(gob.ENCODED_SIZE_MAXIMUM), count)
	testify.Equal_Values(t, gob.STATUS_OK, status)
	decoded, consumed, _, decode_status := gob.Decode_Bytes(destination[:count])
	testify.Equal_Values(t, gob.STATUS_OK, decode_status)
	testify.Equal(t, maximum, decoded)
	testify.Equal(t, gob.Bytes_Consumed_Count(count), consumed)
}

func test_boolean_domains(destination gob.Integer_Output) {
	gob.Boolean_Size(false)
	gob.Boolean_Size(true)
	gob.Encode_Boolean_Into(destination[:0], false)
	gob.Encode_Boolean_Into(destination[:gob.PREFIX_SIZE], true)
	gob.Encode_Boolean_Into(destination[:2], false)
	gob.Encode_Boolean_Into(destination[:], true)
	for _, source := range [...]gob.Integer_Encoded{
		nil,
		{0},
		{0xff},
		{0xff, 0x80},
		{0xf8, 1, 1, 1, 1, 1, 1, 1},
		{0xf8, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
	} {
		gob.Decode_Boolean(source)
	}
}

func test_float_domains(destination gob.Integer_Output) {
	shift := bits.BIT_COUNT_64_MAXIMUM - bits.BIT_COUNT_8_MAXIMUM
	for _, value := range [...]gob.Float_64_Bits{
		0,
		1,
		2,
		gob.Float_64_Bits(1 << shift),
		gob.Float_64_Bits(2 << shift),
		gob.Float_64_Bits(1 << (bits.BIT_COUNT_64_MAXIMUM - 1)),
		gob.Float_64_Bits(bits.WORD_64_MAXIMUM),
	} {
		gob.Float_Size(value)
	}
	gob.Encode_Float_Into(destination[:0], 0)
	gob.Encode_Float_Into(destination[:1], 0)
	gob.Encode_Float_Into(
		destination[:2], gob.Float_64_Bits(1<<(bits.BIT_COUNT_64_MAXIMUM-1)),
	)
	gob.Encode_Float_Into(destination[:], 1)
	gob.Encode_Float_Into(destination[:], 2)
	gob.Encode_Float_Into(destination[:], gob.Float_64_Bits(bits.WORD_64_MAXIMUM))
	for _, source := range [...]gob.Integer_Encoded{
		nil,
		{0},
		{0xff},
		{0xff, 0x80},
		{0xf8, 1, 1, 1, 1, 1, 1, 1},
		{0xf8, 1, 0, 0, 0, 0, 0, 0, 0},
		{0xf8, 2, 0, 0, 0, 0, 0, 0, 0},
		{0xf8, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
	} {
		gob.Decode_Float(source)
	}
}

func test_complex_domains(destination gob.Complex_Output) {
	maximum := gob.Float_64_Bits(bits.WORD_64_MAXIMUM)
	for _, value := range [...]gob.Float_64_Bits{0, 1, 2, maximum} {
		gob.Complex_Size(value, value)
	}
	gob.Encode_Complex_Into(destination[:0], 0, 0)
	gob.Encode_Complex_Into(destination[:1], 0, 0)
	gob.Encode_Complex_Into(destination[:gob.COMPLEX_ENCODED_SIZE_MINIMUM], 0, 0)
	for _, value := range [...]gob.Float_64_Bits{1, 2, maximum} {
		count, _ := gob.Encode_Complex_Into(destination[:], value, value)
		gob.Decode_Complex(gob.Complex_Encoded(destination[:count]))
	}
	for _, source := range [...]gob.Complex_Encoded{
		nil,
		{0},
		{0, 0},
		{
			0xf8, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xf8, 1, 1, 1, 1, 1, 1, 1,
		},
		{
			0xf8, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xf8, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		},
	} {
		gob.Decode_Complex(source)
	}
}

func test_integer_size_allocation(t *testing.T) {
	var size gob.Integer_Size_Count
	testify.Zero_Allocation(t, func() {
		size = gob.Unsigned_Size(gob.Unsigned(bits.WORD_64_MAXIMUM))
	})
	testify.Equal(t, gob.Integer_Size_Count(gob.INTEGER_ENCODED_SIZE_MAXIMUM), size)
	testify.Zero_Allocation(t, func() {
		size = gob.Integer_Size(gob.Integer(bits.INTEGER_64_MINIMUM))
	})
	testify.Equal(t, gob.Integer_Size_Count(gob.INTEGER_ENCODED_SIZE_MAXIMUM), size)
}

func test_integer_encode_allocation(t *testing.T) {
	var destination [gob.INTEGER_ENCODED_SIZE_MAXIMUM]byte
	var count gob.Integer_Encoded_Count
	var status gob.Integer_Encode_Status
	testify.Zero_Allocation(t, func() {
		count, status = gob.Encode_Unsigned_Into(
			destination[:], gob.Unsigned(bits.WORD_64_MAXIMUM),
		)
	})
	testify.Equal_Values(t, gob.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		count, status = gob.Encode_Unsigned_Into(destination[:0], 0)
	})
	testify.Equal_Values(t, gob.STATUS_OUTPUT_TOO_SMALL, status)
	testify.Zero_Allocation(t, func() {
		count, status = gob.Encode_Integer_Into(
			destination[:], gob.Integer(bits.INTEGER_64_MINIMUM),
		)
	})
	testify.Equal_Values(t, gob.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		count, status = gob.Encode_Integer_Into(destination[:0], 0)
	})
	testify.Equal_Values(t, gob.STATUS_OUTPUT_TOO_SMALL, status)
	testify.True(t, count <= gob.INTEGER_ENCODED_SIZE_MAXIMUM)
}

func test_integer_decode_allocation(t *testing.T) {
	encoded := gob.Integer_Encoded{0xf8, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	invalid := gob.Integer_Encoded{0xff}
	var consumed gob.Integer_Consumed_Count
	var position gob.Integer_Position
	var unsigned gob.Unsigned
	var integer gob.Integer
	var status gob.Decode_Status
	testify.Zero_Allocation(t, func() {
		unsigned, consumed, position, status = gob.Decode_Unsigned(encoded)
	})
	testify.Equal_Values(t, gob.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		unsigned, consumed, position, status = gob.Decode_Unsigned(invalid)
	})
	testify.Equal_Values(t, gob.STATUS_INPUT_INVALID, status)
	testify.Zero_Allocation(t, func() {
		integer, consumed, position, status = gob.Decode_Integer(encoded)
	})
	testify.Equal_Values(t, gob.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		integer, consumed, position, status = gob.Decode_Integer(invalid)
	})
	testify.Equal_Values(t, gob.STATUS_INPUT_INVALID, status)
	testify.True(t, consumed <= gob.INTEGER_ENCODED_SIZE_MAXIMUM)
	testify.True(t, position <= gob.INTEGER_POSITION_MAXIMUM)
	testify.True(t, unsigned <= gob.Unsigned(bits.WORD_64_MAXIMUM))
	testify.True(t, integer <= gob.Integer(bits.INTEGER_64_MAXIMUM))
}

func test_boolean_allocation(t *testing.T) {
	var destination [gob.PREFIX_SIZE]byte
	encoded := gob.Integer_Encoded{0xff, 0x80}
	var size gob.Boolean_Size_Count
	var count gob.Boolean_Encoded_Count
	var consumed gob.Integer_Consumed_Count
	var position gob.Integer_Position
	var value gob.Boolean
	var encode_status gob.Integer_Encode_Status
	var decode_status gob.Decode_Status
	testify.Zero_Allocation(t, func() { size = gob.Boolean_Size(true) })
	testify.Equal(t, gob.Boolean_Size_Count(gob.PREFIX_SIZE), size)
	testify.Zero_Allocation(t, func() {
		count, encode_status = gob.Encode_Boolean_Into(destination[:], true)
	})
	testify.Equal_Values(t, gob.STATUS_OK, encode_status)
	testify.Zero_Allocation(t, func() {
		count, encode_status = gob.Encode_Boolean_Into(destination[:0], false)
	})
	testify.Equal_Values(t, gob.STATUS_OUTPUT_TOO_SMALL, encode_status)
	testify.Zero_Allocation(t, func() {
		value, consumed, position, decode_status = gob.Decode_Boolean(encoded)
	})
	testify.Equal_Values(t, gob.STATUS_OK, decode_status)
	testify.True(t, bool(value))
	testify.True(t, count <= gob.PREFIX_SIZE)
	testify.True(t, consumed <= gob.INTEGER_ENCODED_SIZE_MAXIMUM)
	testify.True(t, position <= gob.INTEGER_POSITION_MAXIMUM)
}

func test_float_allocation(t *testing.T) {
	var destination [gob.INTEGER_ENCODED_SIZE_MAXIMUM]byte
	encoded := gob.Integer_Encoded{
		0xf8, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	}
	invalid := gob.Integer_Encoded{0xff}
	maximum := gob.Float_64_Bits(bits.WORD_64_MAXIMUM)
	var size gob.Integer_Size_Count
	var count gob.Integer_Encoded_Count
	var consumed gob.Integer_Consumed_Count
	var position gob.Integer_Position
	var value gob.Float_64_Bits
	var encode_status gob.Integer_Encode_Status
	var decode_status gob.Decode_Status
	testify.Zero_Allocation(t, func() { size = gob.Float_Size(maximum) })
	testify.Equal(t, gob.Integer_Size_Count(gob.INTEGER_ENCODED_SIZE_MAXIMUM), size)
	testify.Zero_Allocation(t, func() {
		count, encode_status = gob.Encode_Float_Into(destination[:], maximum)
	})
	testify.Equal_Values(t, gob.STATUS_OK, encode_status)
	testify.Zero_Allocation(t, func() {
		count, encode_status = gob.Encode_Float_Into(destination[:0], 0)
	})
	testify.Equal_Values(t, gob.STATUS_OUTPUT_TOO_SMALL, encode_status)
	testify.Zero_Allocation(t, func() {
		value, consumed, position, decode_status = gob.Decode_Float(encoded)
	})
	testify.Equal_Values(t, gob.STATUS_OK, decode_status)
	testify.Zero_Allocation(t, func() {
		value, consumed, position, decode_status = gob.Decode_Float(invalid)
	})
	testify.Equal_Values(t, gob.STATUS_INPUT_INVALID, decode_status)
	testify.True(t, count <= gob.INTEGER_ENCODED_SIZE_MAXIMUM)
	testify.True(t, consumed <= gob.INTEGER_ENCODED_SIZE_MAXIMUM)
	testify.True(t, position <= gob.INTEGER_POSITION_MAXIMUM)
	testify.True(t, value <= maximum)
}

func test_complex_allocation(t *testing.T) {
	var destination [gob.COMPLEX_ENCODED_SIZE_MAXIMUM]byte
	maximum := gob.Float_64_Bits(bits.WORD_64_MAXIMUM)
	var size gob.Complex_Size_Count
	var count gob.Complex_Encoded_Count
	var consumed gob.Complex_Consumed_Count
	var position gob.Complex_Position
	var real_bits gob.Float_64_Bits
	var imaginary_bits gob.Float_64_Bits
	var encode_status gob.Integer_Encode_Status
	var decode_status gob.Decode_Status
	testify.Zero_Allocation(t, func() {
		size = gob.Complex_Size(maximum, maximum)
	})
	testify.Equal(t, gob.Complex_Size_Count(gob.COMPLEX_ENCODED_SIZE_MAXIMUM), size)
	testify.Zero_Allocation(t, func() {
		count, encode_status = gob.Encode_Complex_Into(destination[:], maximum, maximum)
	})
	testify.Equal_Values(t, gob.STATUS_OK, encode_status)
	testify.Zero_Allocation(t, func() {
		count, encode_status = gob.Encode_Complex_Into(destination[:0], 0, 0)
	})
	testify.Equal_Values(t, gob.STATUS_OUTPUT_TOO_SMALL, encode_status)
	testify.Zero_Allocation(t, func() {
		real_bits, imaginary_bits, consumed, position, decode_status =
			gob.Decode_Complex(destination[:])
	})
	testify.Equal_Values(t, gob.STATUS_OK, decode_status)
	testify.Zero_Allocation(t, func() {
		real_bits, imaginary_bits, consumed, position, decode_status =
			gob.Decode_Complex(destination[:gob.PREFIX_SIZE])
	})
	testify.Equal_Values(t, gob.STATUS_INPUT_INVALID, decode_status)
	testify.True(t, count <= gob.COMPLEX_ENCODED_SIZE_MAXIMUM)
	testify.True(t, consumed <= gob.COMPLEX_ENCODED_SIZE_MAXIMUM)
	testify.True(t, position <= gob.COMPLEX_ENCODED_SIZE_MAXIMUM)
	testify.True(t, real_bits <= maximum)
	testify.True(t, imaginary_bits <= maximum)
}

func test_bytes_allocation(t *testing.T) {
	var destination [gob.ENCODED_SIZE_MAXIMUM]byte
	maximum := gob.Bytes(make([]byte, gob.BYTES_SIZE_MAXIMUM))
	overlap := gob.Bytes(destination[:1])
	invalid := gob.Bytes_Encoded{2, 0}
	var count gob.Bytes_Encoded_Count
	var consumed gob.Bytes_Consumed_Count
	var position gob.Bytes_Position
	var decoded gob.Bytes
	var size gob.Bytes_Size_Count
	var encode_status gob.Bytes_Encode_Status
	var decode_status gob.Decode_Status
	testify.Zero_Allocation(t, func() {
		size = gob.Bytes_Size(maximum)
	})
	testify.Equal(t, gob.Bytes_Size_Count(gob.ENCODED_SIZE_MAXIMUM), size)
	testify.Zero_Allocation(t, func() {
		count, encode_status = gob.Encode_Bytes_Into(destination[:], maximum)
	})
	testify.Equal_Values(t, gob.STATUS_OK, encode_status)
	testify.Zero_Allocation(t, func() {
		count, encode_status = gob.Encode_Bytes_Into(destination[:0], nil)
	})
	testify.Equal_Values(t, gob.STATUS_OUTPUT_TOO_SMALL, encode_status)
	testify.Zero_Allocation(t, func() {
		count, encode_status = gob.Encode_Bytes_Into(destination[:], overlap)
	})
	testify.Equal_Values(t, gob.STATUS_STORAGE_INVALID, encode_status)
	testify.Zero_Allocation(t, func() {
		decoded, consumed, position, decode_status = gob.Decode_Bytes(destination[:])
	})
	testify.Equal_Values(t, gob.STATUS_OK, decode_status)
	testify.Zero_Allocation(t, func() {
		decoded, consumed, position, decode_status = gob.Decode_Bytes(invalid)
	})
	testify.Equal_Values(t, gob.STATUS_INPUT_INVALID, decode_status)
	testify.True(t, count <= gob.ENCODED_SIZE_MAXIMUM)
	testify.True(t, consumed <= gob.ENCODED_SIZE_MAXIMUM)
	testify.True(t, position <= gob.POSITION_MAXIMUM)
	testify.True(t, len(decoded) <= gob.BYTES_SIZE_MAXIMUM)
}
