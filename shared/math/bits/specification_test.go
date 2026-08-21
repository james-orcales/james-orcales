package bits_test

import (
	"testing"

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Word_Size verifies that the machine word width is one of the two widths Go
// defines, and that it agrees with the size of a machine word.
func Test_Word_Size(t *testing.T) {
	t.Parallel()
	if bits.WORD_SIZE != 32 {
		testify.Equal_Values(t, 64, bits.WORD_SIZE,
			"WORD_SIZE = %d, want 32 or 64", bits.WORD_SIZE)

	}
	testify.Equal_Values(t, bits.WORD_SIZE, bits.Leading_Zeros(0),
		"Leading_Zeros(0) = %d, want %d", bits.Leading_Zeros(0), bits.WORD_SIZE)

}

// Test_Integer_Limits verifies that each width limit holds the value the width admits and
// that one more step would leave the width.
func Test_Integer_Limits(t *testing.T) {
	t.Parallel()
	// A constant cannot hold one step past its own width, thus the step happens at run
	// time where the value wraps instead.
	signed_8 := bits.INTEGER_8_MAXIMUM
	signed_8++
	testify.Equal_Values(t, bits.INTEGER_8_MINIMUM, signed_8,
		"the 8-bit limits must wrap into each other")

	signed_16 := bits.INTEGER_16_MAXIMUM
	signed_16++
	testify.Equal_Values(t, bits.INTEGER_16_MINIMUM, signed_16,
		"the 16-bit limits must wrap into each other")

	signed_32 := bits.INTEGER_32_MAXIMUM
	signed_32++
	testify.Equal_Values(t, bits.INTEGER_32_MINIMUM, signed_32,
		"the 32-bit limits must wrap into each other")

	signed_64 := bits.INTEGER_64_MAXIMUM
	signed_64++
	testify.Equal_Values(t, bits.INTEGER_64_MINIMUM, signed_64,
		"the 64-bit limits must wrap into each other")

	unsigned_8 := bits.WORD_8_MAXIMUM
	unsigned_8++
	testify.Equal_Values(t, 0, unsigned_8,
		"the unsigned 8-bit limit must wrap to zero")

	unsigned_64 := bits.WORD_64_MAXIMUM
	unsigned_64++
	testify.Equal_Values(t, 0, unsigned_64,
		"the unsigned 64-bit limit must wrap to zero")

	machine := bits.INTEGER_MAXIMUM
	machine++
	testify.Equal_Values(t, bits.INTEGER_MINIMUM, machine,
		"the machine limits must wrap into each other")

	unsigned_machine := bits.WORD_MAXIMUM
	unsigned_machine++
	testify.Equal_Values(t, 0, unsigned_machine,
		"the unsigned machine limit must wrap to zero")
	testify.Equal_Values(t, 65535, bits.WORD_16_MAXIMUM,
		"UNSIGNED_16_MAXIMUM = %d, want 65535", bits.WORD_16_MAXIMUM)
	testify.Equal_Values(t, 4294967295, bits.WORD_32_MAXIMUM,
		"UNSIGNED_32_MAXIMUM = %d, want 4294967295", bits.WORD_32_MAXIMUM)

}

// Test_Decimal_Digit_Bound verifies shared fixed-point ceiling stays above scaled log10(2).
func Test_Decimal_Digit_Bound(t *testing.T) {
	t.Parallel()
	testify.Equal_Values(t,
		1<<bits.DECIMAL_DIGIT_BINARY_LOGARITHM_SHIFT,
		bits.DECIMAL_DIGIT_BINARY_LOGARITHM_SCALE,
		"decimal binary-logarithm scale follows its precision")
	testify.Equal_Values(t, 1233, bits.DECIMAL_DIGIT_BINARY_LOGARITHM_CEILING,
		"scale-12 ceiling remains least integer above scaled log10(2)")
}

// Test_Byte_Units verifies that decimal units use powers of 1000 and that IEC units use
// powers of 1024. The two ladders must stay separate because the same prefix magnitude
// names different byte counts.
func Test_Byte_Units(t *testing.T) {
	t.Parallel()
	const DECIMAL_RADIX = 10
	const SI_PREFIX_STEP = DECIMAL_RADIX * DECIMAL_RADIX * DECIMAL_RADIX
	units := [...]struct {
		Name     string
		Actual   uint64
		Expected uint64
	}{
		{"KILOBYTE_BYTES", bits.KILOBYTE_BYTES, SI_PREFIX_STEP},
		{"MEGABYTE_BYTES", bits.MEGABYTE_BYTES, SI_PREFIX_STEP * SI_PREFIX_STEP},
		{"GIGABYTE_BYTES", bits.GIGABYTE_BYTES,
			SI_PREFIX_STEP * SI_PREFIX_STEP * SI_PREFIX_STEP},
		{"TERABYTE_BYTES", bits.TERABYTE_BYTES,
			SI_PREFIX_STEP * SI_PREFIX_STEP * SI_PREFIX_STEP * SI_PREFIX_STEP},
		{"PETABYTE_BYTES", bits.PETABYTE_BYTES,
			SI_PREFIX_STEP * SI_PREFIX_STEP * SI_PREFIX_STEP * SI_PREFIX_STEP *
				SI_PREFIX_STEP},
		{"EXABYTE_BYTES", bits.EXABYTE_BYTES,
			SI_PREFIX_STEP * SI_PREFIX_STEP * SI_PREFIX_STEP * SI_PREFIX_STEP *
				SI_PREFIX_STEP * SI_PREFIX_STEP},
		{"KIBIBYTE_BYTES", bits.KIBIBYTE_BYTES, 1 << 10},
		{"MEBIBYTE_BYTES", bits.MEBIBYTE_BYTES, 1 << 20},
		{"GIBIBYTE_BYTES", bits.GIBIBYTE_BYTES, 1 << 30},
		{"TEBIBYTE_BYTES", bits.TEBIBYTE_BYTES, 1 << 40},
		{"PEBIBYTE_BYTES", bits.PEBIBYTE_BYTES, 1 << 50},
		{"EXBIBYTE_BYTES", bits.EXBIBYTE_BYTES, 1 << 60},
	}
	for _, unit := range units {
		testify.Equal_Values(t, unit.Expected, unit.Actual,
			"%s = %d, want %d", unit.Name, unit.Actual, unit.Expected)
	}
}

// Test_Leading_Zeros verifies the count of zero bits above the highest set bit at each
// width, including the full-width result a zero operand gives.
func Test_Leading_Zeros(t *testing.T) {
	t.Parallel()
	testify.Equal_Values(t, 8, bits.Leading_Zeros_8(0),
		"Leading_Zeros_8(0) = %d, want 8", bits.Leading_Zeros_8(0))
	testify.Equal_Values(t, 7, bits.Leading_Zeros_8(1),
		"Leading_Zeros_8(1) = %d, want 7", bits.Leading_Zeros_8(1))
	testify.Equal_Values(t, 0, bits.Leading_Zeros_8(0x80),
		"Leading_Zeros_8(0x80) = %d, want 0", bits.Leading_Zeros_8(0x80))
	testify.Equal_Values(t, 7, bits.Leading_Zeros_16(0x0100),
		"Leading_Zeros_16(0x0100) = %d, want 7", bits.Leading_Zeros_16(0x0100))
	testify.Equal_Values(t, 15, bits.Leading_Zeros_32(0x00010000),

		"Leading_Zeros_32(0x00010000) = %d, want 15",
		bits.Leading_Zeros_32(0x00010000))
	testify.Equal_Values(t, 0, bits.Leading_Zeros_64(1<<63),
		"Leading_Zeros_64(1<<63) = %d, want 0", bits.Leading_Zeros_64(1<<63))
	testify.Equal_Values(t, 64, bits.Leading_Zeros_64(0),
		"Leading_Zeros_64(0) = %d, want 64", bits.Leading_Zeros_64(0))

}

// Test_Trailing_Zeros verifies the count of zero bits below the lowest set bit at each
// width, including the full-width result a zero operand gives.
func Test_Trailing_Zeros(t *testing.T) {
	t.Parallel()
	testify.Equal_Values(t, 8, bits.Trailing_Zeros_8(0),
		"Trailing_Zeros_8(0) = %d, want 8", bits.Trailing_Zeros_8(0))
	testify.Equal_Values(t, 0, bits.Trailing_Zeros_8(1),
		"Trailing_Zeros_8(1) = %d, want 0", bits.Trailing_Zeros_8(1))
	testify.Equal_Values(t, 7, bits.Trailing_Zeros_8(0x80),
		"Trailing_Zeros_8(0x80) = %d, want 7", bits.Trailing_Zeros_8(0x80))
	testify.Equal_Values(t, 8, bits.Trailing_Zeros_16(0x0100),
		"Trailing_Zeros_16(0x0100) = %d, want 8", bits.Trailing_Zeros_16(0x0100))
	testify.Equal_Values(t, 16, bits.Trailing_Zeros_32(0x00010000),

		"Trailing_Zeros_32(0x00010000) = %d, want 16",
		bits.Trailing_Zeros_32(0x00010000))
	testify.Equal_Values(t, 63, bits.Trailing_Zeros_64(1<<63),
		"Trailing_Zeros_64(1<<63) = %d, want 63", bits.Trailing_Zeros_64(1<<63))
	testify.Equal_Values(t, bits.WORD_SIZE, bits.Trailing_Zeros(0),
		"Trailing_Zeros(0) = %d, want %d", bits.Trailing_Zeros(0), bits.WORD_SIZE)

}

// Test_Ones_Count verifies the count of set bits at each width, and the stated relation
// that a set-bit count and a leading-zero count never both reach the full width.
func Test_Ones_Count(t *testing.T) {
	t.Parallel()
	testify.Equal_Values(t, 0, bits.Ones_Count_8(0),
		"Ones_Count_8(0) = %d, want 0", bits.Ones_Count_8(0))
	testify.Equal_Values(t, 8, bits.Ones_Count_8(0xff),
		"Ones_Count_8(0xff) = %d, want 8", bits.Ones_Count_8(0xff))
	testify.Equal_Values(t, 8, bits.Ones_Count_16(0xff00),
		"Ones_Count_16(0xff00) = %d, want 8", bits.Ones_Count_16(0xff00))
	testify.Equal_Values(t, 32, bits.Ones_Count_32(0xffffffff),
		"Ones_Count_32(0xffffffff) = %d, want 32", bits.Ones_Count_32(0xffffffff))
	testify.Equal_Values(t, 64, bits.Ones_Count_64(0xffffffffffffffff),
		"Ones_Count_64(max) = %d, want 64", bits.Ones_Count_64(0xffffffffffffffff))
	testify.Equal_Values(t, 0, bits.Ones_Count(0),
		"Ones_Count(0) = %d, want 0", bits.Ones_Count(0))

	// A full set-bit count leaves no leading zero, and a full leading-zero count leaves
	// no set bit; the two counts cannot both reach the width.
	for value_index := 0; value_index < 256; value_index++ {
		ones := bits.Ones_Count_8(bits.Word_8(value_index))
		zeros := bits.Leading_Zeros_8(bits.Word_8(value_index))
		if ones == 8 {
			testify.Not_Equal_Values(t, 8, zeros,
				"value %d reports both counts full", value_index)

		}
	}
}

// Test_Bit_Size verifies the position above the highest set bit at each width, and its
// stated relation to the leading-zero count.
func Test_Bit_Size(t *testing.T) {
	t.Parallel()
	testify.Equal_Values(t, 0, bits.Bit_Size_8(0),
		"Bit_Size_8(0) = %d, want 0", bits.Bit_Size_8(0))
	testify.Equal_Values(t, 1, bits.Bit_Size_8(1),
		"Bit_Size_8(1) = %d, want 1", bits.Bit_Size_8(1))
	testify.Equal_Values(t, 8, bits.Bit_Size_8(0xff),
		"Bit_Size_8(0xff) = %d, want 8", bits.Bit_Size_8(0xff))
	testify.Equal_Values(t, 9, bits.Bit_Size_16(0x0100),
		"Bit_Size_16(0x0100) = %d, want 9", bits.Bit_Size_16(0x0100))
	testify.Equal_Values(t, 17, bits.Bit_Size_32(0x00010000),
		"Bit_Size_32(0x00010000) = %d, want 17", bits.Bit_Size_32(0x00010000))
	testify.Equal_Values(t, 64, bits.Bit_Size_64(1<<63),
		"Bit_Size_64(1<<63) = %d, want 64", bits.Bit_Size_64(1<<63))
	testify.Equal_Values(t, 0, bits.Bit_Size(0),
		"Bit_Size(0) = %d, want 0", bits.Bit_Size(0))

	// The width less the leading-zero count gives the bit size.
	for value_index := 0; value_index < 256; value_index++ {
		expected := 8 - bits.Leading_Zeros_8(bits.Word_8(value_index))
		testify.Equal_Values(t, expected, bits.Bit_Size_8(bits.Word_8(value_index)),
			"Bit_Size_8(%d) disagrees with the zero count", value_index)

	}
}

// Test_Rotation verifies the left turn at each width, the right turn a negative Rotation
// gives, and the identity a full turn gives.
func Test_Rotation(t *testing.T) {
	t.Parallel()
	testify.Equal_Values(t, 0x03, bits.Rotate_Left_8(0x81, 1),
		"Rotate_Left_8(0x81,1) = %#x, want 0x03", bits.Rotate_Left_8(0x81, 1))
	testify.Equal_Values(t, 0xc0, bits.Rotate_Left_8(0x81, -1),
		"Rotate_Left_8(0x81,-1) = %#x, want 0xc0", bits.Rotate_Left_8(0x81, -1))
	testify.Equal_Values(t, 0x81, bits.Rotate_Left_8(0x81, 8),
		"Rotate_Left_8(0x81,8) = %#x, want 0x81", bits.Rotate_Left_8(0x81, 8))
	testify.Equal_Values(t, 0x0003, bits.Rotate_Left_16(0x8001, 1),
		"Rotate_Left_16(0x8001,1) = %#x, want 0x0003",
		bits.Rotate_Left_16(0x8001, 1))
	testify.Equal_Values(t, 0x00000003, bits.Rotate_Left_32(0x80000001, 1),
		"Rotate_Left_32(0x80000001,1) = %#x, want 0x3",
		bits.Rotate_Left_32(0x80000001, 1))
	testify.Equal_Values(t, 3, bits.Rotate_Left_64(1<<63|1, 1),
		"Rotate_Left_64(1<<63|1,1) = %#x, want 3", bits.Rotate_Left_64(1<<63|1, 1))
	testify.Equal_Values(t, 1, bits.Rotate_Left(1, 0),
		"Rotate_Left(1,0) = %#x, want 1", bits.Rotate_Left(1, 0))

}

// Test_Reversal verifies the mirror of the bit positions at each width, and that two
// applications return the operand.
func Test_Reversal(t *testing.T) {
	t.Parallel()
	testify.Equal_Values(t, 0x80, bits.Reverse_8(0x01),
		"Reverse_8(0x01) = %#x, want 0x80", bits.Reverse_8(0x01))
	testify.Equal_Values(t, 0x8000, bits.Reverse_16(0x0001),
		"Reverse_16(0x0001) = %#x, want 0x8000", bits.Reverse_16(0x0001))
	testify.Equal_Values(t, 0x80000000, bits.Reverse_32(0x00000001),
		"Reverse_32(1) = %#x, want 0x80000000", bits.Reverse_32(0x00000001))
	testify.Equal_Values(t, bits.Word_64(1)<<63, bits.Reverse_64(1),
		"Reverse_64(1) = %#x, want 1<<63", bits.Reverse_64(1))
	testify.Equal_Values(t, 0, bits.Reverse(0),
		"Reverse(0) = %#x, want 0", bits.Reverse(0))

	for value_index := 0; value_index < 256; value_index++ {
		twice := bits.Reverse_8(bits.Reverse_8(bits.Word_8(value_index)))
		testify.Equal_Values(t, bits.Word_8(value_index), twice,
			"Reverse_8 applied twice to %d gives %d", value_index, twice)

	}
}

// Test_Byte_Reversal verifies the mirror of the byte positions at each width, and that
// the bits inside each byte stay in place.
func Test_Byte_Reversal(t *testing.T) {
	t.Parallel()
	testify.Equal_Values(t, 0x0201, bits.Reverse_Bytes_16(0x0102),
		"Reverse_Bytes_16(0x0102) = %#x, want 0x0201",
		bits.Reverse_Bytes_16(0x0102))
	testify.Equal_Values(t, 0x04030201, bits.Reverse_Bytes_32(0x01020304),
		"Reverse_Bytes_32(0x01020304) = %#x, want 0x04030201",
		bits.Reverse_Bytes_32(0x01020304))
	testify.Equal_Values(t, 0x0807060504030201, bits.Reverse_Bytes_64(0x0102030405060708),
		"Reverse_Bytes_64 = %#x, want 0x0807060504030201",
		bits.Reverse_Bytes_64(0x0102030405060708))
	testify.Equal_Values(t, 0, bits.Reverse_Bytes(0),
		"Reverse_Bytes(0) = %#x, want 0", bits.Reverse_Bytes(0))
	// The bits inside one byte keep their order, thus a single byte survives unchanged.
	testify.Equal_Values(t, 0x8100, bits.Reverse_Bytes_16(0x0081),
		"Reverse_Bytes_16(0x0081) = %#x, want 0x8100",
		bits.Reverse_Bytes_16(0x0081))

}

// Test_Addition verifies the low word and the carry that each width returns, and the
// chain of two calls that adds a double-width operand.
func Test_Addition(t *testing.T) {
	t.Parallel()
	sum_32, carry_32 := bits.Add_32(1, 2, 0)
	testify.Equal_Values(t, 3, sum_32,
		"Add_32(1,2,0) sum = %d, want 3", sum_32)
	testify.Equal_Values(t, 0, carry_32,
		"Add_32(1,2,0) carry = %d, want 0", carry_32)

	sum_32, carry_32 = bits.Add_32(0xffffffff, 1, 0)
	testify.Equal_Values(t, 0, sum_32,
		"Add_32(max,1,0) sum = %d, want 0", sum_32)
	testify.Equal_Values(t, 1, carry_32,
		"Add_32(max,1,0) carry = %d, want 1", carry_32)

	sum_64, carry_64 := bits.Add_64(0xffffffffffffffff, 0, 1)
	testify.Equal_Values(t, 0, sum_64,
		"Add_64(max,0,1) sum = %d, want 0", sum_64)
	testify.Equal_Values(t, 1, carry_64,
		"Add_64(max,0,1) carry = %d, want 1", carry_64)

	// A chain carries between the words of a 128-bit addend.
	low, carry := bits.Add_64(0xffffffffffffffff, 1, 0)
	high, _ := bits.Add_64(0, 0, bits.Carry_In(carry))
	testify.Equal_Values(t, 0, low,
		"chained low = %d, want 0", low)
	testify.Equal_Values(t, 1, high,
		"chained high = %d, want 1", high)

	word_sum, word_carry := bits.Add_Word(1, 1, 0)
	testify.Equal_Values(t, 2, word_sum,
		"Add_Word(1,1,0) sum = %d, want 2", word_sum)
	testify.Equal_Values(t, 0, word_carry,
		"Add_Word(1,1,0) carry = %d, want 0", word_carry)

}

// Test_Subtraction verifies the low word and the borrow that each width returns.
func Test_Subtraction(t *testing.T) {
	t.Parallel()
	difference_32, borrow_32 := bits.Subtract_32(3, 1, 0)
	testify.Equal_Values(t, 2, difference_32,
		"Subtract_32(3,1,0) difference = %d, want 2", difference_32)
	testify.Equal_Values(t, 0, borrow_32,
		"Subtract_32(3,1,0) borrow = %d, want 0", borrow_32)

	difference_32, borrow_32 = bits.Subtract_32(0, 1, 0)
	testify.Equal_Values(t, 0xffffffff, difference_32,
		"Subtract_32(0,1,0) difference = %#x, want 0xffffffff", difference_32)
	testify.Equal_Values(t, 1, borrow_32,
		"Subtract_32(0,1,0) borrow = %d, want 1", borrow_32)

	difference_64, borrow_64 := bits.Subtract_64(0, 0, 1)
	testify.Equal_Values(t, bits.Word_64(bits.WORD_64_MAXIMUM), difference_64,
		"Subtract_64(0,0,1) difference = %#x, want max", difference_64)
	testify.Equal_Values(t, 1, borrow_64,
		"Subtract_64(0,0,1) borrow = %d, want 1", borrow_64)

	word_difference, word_borrow := bits.Subtract_Word(2, 1, 0)
	testify.Equal_Values(t, 1, word_difference,
		"Subtract_Word(2,1,0) difference = %d, want 1", word_difference)
	testify.Equal_Values(t, 0, word_borrow,
		"Subtract_Word(2,1,0) borrow = %d, want 0", word_borrow)

}

// Test_Multiplication verifies the high word and the low word of the double-width
// product at each width.
func Test_Multiplication(t *testing.T) {
	t.Parallel()
	high_32, low_32 := bits.Multiply_32(0x10000, 0x10000)
	testify.Equal_Values(t, 1, high_32,
		"Multiply_32(0x10000,0x10000) high = %d, want 1", high_32)
	testify.Equal_Values(t, 0, low_32,
		"Multiply_32(0x10000,0x10000) low = %d, want 0", low_32)

	high_64, low_64 := bits.Multiply_64(1<<32, 1<<32)
	testify.Equal_Values(t, 1, high_64,
		"Multiply_64(1<<32,1<<32) high = %d, want 1", high_64)
	testify.Equal_Values(t, 0, low_64,
		"Multiply_64(1<<32,1<<32) low = %d, want 0", low_64)

	high_64, low_64 = bits.Multiply_64(3, 5)
	testify.Equal_Values(t, 0, high_64,
		"Multiply_64(3,5) high = %d, want 0", high_64)
	testify.Equal_Values(t, 15, low_64,
		"Multiply_64(3,5) low = %d, want 15", low_64)

	word_high, word_low := bits.Multiply_Word(2, 3)
	testify.Equal_Values(t, 0, word_high,
		"Multiply_Word(2,3) high = %d, want 0", word_high)
	testify.Equal_Values(t, 6, word_low,
		"Multiply_Word(2,3) low = %d, want 6", word_low)

}

// Test_Division verifies the quotient and the remainder of a double-width dividend, and
// the remainder that the Remainder functions return for a quotient that overflows.
func Test_Division(t *testing.T) {
	t.Parallel()
	quotient_32, remainder_32 := bits.Divide_32(0, 100, 7)
	testify.Equal_Values(t, 14, quotient_32,
		"Divide_32(0,100,7) quotient = %d, want 14", quotient_32)
	testify.Equal_Values(t, 2, remainder_32,
		"Divide_32(0,100,7) remainder = %d, want 2", remainder_32)

	quotient_64, remainder_64 := bits.Divide_64(1, 0, 2)
	testify.Equal_Values(t, bits.Word_64(1)<<63, quotient_64,
		"Divide_64(1,0,2) quotient = %#x, want 1<<63", quotient_64)
	testify.Equal_Values(t, 0, remainder_64,
		"Divide_64(1,0,2) remainder = %d, want 0", remainder_64)

	word_quotient, word_remainder := bits.Divide_Word(0, 9, 4)
	testify.Equal_Values(t, 2, word_quotient,
		"Divide_Word(0,9,4) quotient = %d, want 2", word_quotient)
	testify.Equal_Values(t, 1, word_remainder,
		"Divide_Word(0,9,4) remainder = %d, want 1", word_remainder)
	// The Remainder functions accept a high word that a Divide function rejects.
	testify.Equal_Values(t, 0, bits.Remainder_64(1, 0, 1),
		"Remainder_64(1,0,1) = %d, want 0", bits.Remainder_64(1, 0, 1))
	testify.Equal_Values(t, 5, bits.Remainder_32(3, 0, 7),
		"Remainder_32(3,0,7) = %d, want 5", bits.Remainder_32(3, 0, 7))
	testify.Equal_Values(t, 1, bits.Remainder_Word(0, 9, 4),
		"Remainder_Word(0,9,4) = %d, want 1", bits.Remainder_Word(0, 9, 4))

}

// Test_Domain_Errors verifies that a zero divisor, an overflowing quotient, and a carry
// above one each panic instead of returning an error value.
func Test_Domain_Errors(t *testing.T) {
	t.Parallel()
	testify.Panics(t, func() { bits.Divide_64(0, 1, 0) },
		"Divide_64 with a zero divisor must panic")
	testify.Panics(t, func() { bits.Divide_64(2, 0, 1) },
		"Divide_64 with an overflowing quotient must panic")
	testify.Panics(t, func() { bits.Remainder_64(0, 1, 0) },
		"Remainder_64 with a zero divisor must panic")
	testify.Panics(t, func() { bits.Add_64(0, 0, 2) },
		"Add_64 with a carry above one must panic")
	testify.Panics(t, func() { bits.Subtract_64(0, 0, 2) },
		"Subtract_64 with a borrow above one must panic")

}
