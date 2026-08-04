package bits

import (
	"testing"

	"local/james-orcales/shared/testify"
)

// DE_BRUIJN_64 is the 64-bit de Bruijn sequence the standard library rotates and reverses
// in its own tests. Every 6-bit window of it is distinct, thus a shift of it exposes a
// wrong bit at any position.
const DE_BRUIJN_64 uint64 = 0x03f79d71b4ca8b09

// WORD_32_ALL_ONES is the largest 32-bit word, the standard library test operand.
const WORD_32_ALL_ONES uint32 = 1<<32 - 1

// BYTE_VALUE_COUNT is how many distinct values one byte holds.
const BYTE_VALUE_COUNT = 256

// PAIR_SIZE is the size of a value and its mirror.
const PAIR_SIZE = 2

// ARITHMETIC_ROW_SIZE is the size of one standard library arithmetic row: two operands, a
// carry, a result, and the carry that leaves.
const ARITHMETIC_ROW_SIZE = 5

// REMAINDER_ROW_SIZE is the size of one standard library remainder row: the two dividend
// words, the divisor, and the remainder.
const REMAINDER_ROW_SIZE = 4

// Holds the reference counts for one 8-bit value.
type reference_entry struct {
	// Zeros_Above is the count of zero bits above the highest set bit.
	Zeros_Above int
	// Zeros_Below is the count of zero bits below the lowest set bit.
	Zeros_Below int
	// Ones is the count of set bits.
	Ones int
}

// Builds the reference counts for all 256 8-bit values with plain bit walks. The walks
// share no code with the functions under test, thus they are independent evidence. The
// linter bans a package variable and a function init, so each test asks for the table.
func reference_table() (table [BYTE_VALUE_COUNT]reference_entry) {
	table[0] = reference_entry{Zeros_Above: 8, Zeros_Below: 8, Ones: 0}
	for value_index := 1; value_index < BYTE_VALUE_COUNT; value_index++ {
		above := 0
		walk := value_index
		for walk&0x80 == 0 {
			above++
			walk <<= 1
		}
		below := 0
		walk = value_index
		for walk&1 == 0 {
			below++
			walk >>= 1
		}
		ones := 0
		walk = value_index
		for walk != 0 {
			ones += walk & 1
			walk >>= 1
		}
		table[value_index] = reference_entry{
			Zeros_Above: above, Zeros_Below: below, Ones: ones,
		}
	}
	return table
}

// Test_Standard_Word_Size verifies that the declared word width matches the width a
// machine word really holds.
func Test_Standard_Word_Size(t *testing.T) {
	t.Parallel()
	width := 0
	for probe := WORD_MAXIMUM; probe != 0; probe >>= 1 {
		width++
	}
	testify.Equal_Values(t, WORD_SIZE, width,
		"WORD_SIZE = %d, want %d", WORD_SIZE, width)

}

// Test_Standard_Leading_Zeros walks all 256 byte values through every shift that keeps
// them inside each width, and compares the count against the reference table.
func Test_Standard_Leading_Zeros(t *testing.T) {
	t.Parallel()
	table := reference_table()
	for value_index := 0; value_index < BYTE_VALUE_COUNT; value_index++ {
		for shift_index := 0; shift_index < 64-8; shift_index++ {
			compare_leading_zeros(t, uint64(value_index)<<uint(shift_index),
				table[value_index].Zeros_Above, shift_index)
		}
	}
}

// Compares the leading-zero count of one shifted byte at every width it fits. A shift
// moves the highest set bit up, thus it lowers the count by the same amount.
func compare_leading_zeros(t *testing.T, shifted uint64, above int, shift int) {
	t.Helper()
	if shifted <= 1<<8-1 {
		want := above - shift
		if shifted == 0 {
			want = 8
		}
		testify.Equal_Values(t, want, int(Leading_Zeros_8(Word_8(shifted))),
			"Leading_Zeros_8(%#02x) is not %d", shifted, want)

	}
	if shifted <= 1<<16-1 {
		want := above - shift + 8
		if shifted == 0 {
			want = 16
		}
		testify.Equal_Values(t, want, int(Leading_Zeros_16(Word_16(shifted))),
			"Leading_Zeros_16(%#04x) is not %d", shifted, want)

	}
	if shifted <= 1<<32-1 {
		want := above - shift + 24
		if shifted == 0 {
			want = 32
		}
		testify.Equal_Values(t, want, int(Leading_Zeros_32(Word_32(shifted))),
			"Leading_Zeros_32(%#08x) is not %d", shifted, want)

	}
	want := above - shift + 56
	if shifted == 0 {
		want = 64
	}
	testify.Equal_Values(t, want, int(Leading_Zeros_64(Word_64(shifted))),
		"Leading_Zeros_64(%#016x) is not %d", shifted, want)
	testify.Equal_Values(t, want, int(Leading_Zeros(Word(shifted))),
		"Leading_Zeros(%#016x) is not %d", shifted, want)

}

// Test_Standard_Trailing_Zeros walks all 256 byte values through every shift and compares
// the count against the reference table.
func Test_Standard_Trailing_Zeros(t *testing.T) {
	t.Parallel()
	table := reference_table()
	for value_index := 0; value_index < BYTE_VALUE_COUNT; value_index++ {
		for shift_index := 0; shift_index < 64-8; shift_index++ {
			compare_trailing_zeros(t, uint64(value_index)<<uint(shift_index),
				table[value_index].Zeros_Below+shift_index)
		}
	}
}

// Compares the trailing-zero count of one shifted byte at every width it fits. A zero
// operand has no set bit, thus each width answers with its own full width.
func compare_trailing_zeros(t *testing.T, shifted uint64, want int) {
	t.Helper()
	if shifted <= 1<<8-1 {
		expected := want
		if shifted == 0 {
			expected = 8
		}
		testify.Equal_Values(t, expected, int(Trailing_Zeros_8(Word_8(shifted))),
			"Trailing_Zeros_8(%#02x) is not %d", shifted, expected)

	}
	if shifted <= 1<<16-1 {
		expected := want
		if shifted == 0 {
			expected = 16
		}
		testify.Equal_Values(t, expected, int(Trailing_Zeros_16(Word_16(shifted))),
			"Trailing_Zeros_16(%#04x) is not %d", shifted, expected)

	}
	if shifted <= 1<<32-1 {
		expected := want
		if shifted == 0 {
			expected = 32
		}
		testify.Equal_Values(t, expected, int(Trailing_Zeros_32(Word_32(shifted))),
			"Trailing_Zeros_32(%#08x) is not %d", shifted, expected)

	}
	expected := want
	if shifted == 0 {
		expected = 64
	}
	testify.Equal_Values(t, expected, int(Trailing_Zeros_64(Word_64(shifted))),
		"Trailing_Zeros_64(%#016x) is not %d", shifted, expected)
	testify.Equal_Values(t, expected, int(Trailing_Zeros(Word(shifted))),
		"Trailing_Zeros(%#016x) is not %d", shifted, expected)

}

// Test_Standard_Ones_Count walks all 256 byte values through every shift and compares the
// set-bit count against the reference table. A shift moves the bits but keeps their
// number, thus the count never changes.
func Test_Standard_Ones_Count(t *testing.T) {
	t.Parallel()
	table := reference_table()
	for value_index := 0; value_index < BYTE_VALUE_COUNT; value_index++ {
		want := table[value_index].Ones
		for shift_index := 0; shift_index < 64-8; shift_index++ {
			shifted := uint64(value_index) << uint(shift_index)
			if shifted <= 1<<8-1 {
				testify.Equal_Values(t, want, int(Ones_Count_8(Word_8(shifted))),
					"Ones_Count_8(%#02x) is not %d", shifted, want)

			}
			if shifted <= 1<<16-1 {
				testify.Equal_Values(t, want, int(Ones_Count_16(Word_16(shifted))),
					"Ones_Count_16(%#04x) is not %d", shifted, want)

			}
			if shifted <= 1<<32-1 {
				testify.Equal_Values(t, want, int(Ones_Count_32(Word_32(shifted))),
					"Ones_Count_32(%#08x) is not %d", shifted, want)

			}
			testify.Equal_Values(t, want, int(Ones_Count_64(Word_64(shifted))),
				"Ones_Count_64(%#016x) is not %d", shifted, want)
			testify.Equal_Values(t, want, int(Ones_Count(Word(shifted))),
				"Ones_Count(%#016x) is not %d", shifted, want)

		}
	}
}

// Test_Standard_Bit_Size walks all 256 byte values through every shift and compares the
// bit size against the reference table.
func Test_Standard_Bit_Size(t *testing.T) {
	t.Parallel()
	table := reference_table()
	for value_index := 0; value_index < BYTE_VALUE_COUNT; value_index++ {
		byte_width := 8 - table[value_index].Zeros_Above
		for shift_index := 0; shift_index < 64-8; shift_index++ {
			shifted := uint64(value_index) << uint(shift_index)
			want := 0
			if shifted != 0 {
				want = byte_width + shift_index
			}
			if shifted <= 1<<8-1 {
				testify.Equal_Values(t, want, int(Bit_Size_8(Word_8(shifted))),
					"Bit_Size_8(%#02x) is not %d", shifted, want)

			}
			if shifted <= 1<<16-1 {
				testify.Equal_Values(t, want, int(Bit_Size_16(Word_16(shifted))),
					"Bit_Size_16(%#04x) is not %d", shifted, want)

			}
			if shifted <= 1<<32-1 {
				testify.Equal_Values(t, want, int(Bit_Size_32(Word_32(shifted))),
					"Bit_Size_32(%#08x) is not %d", shifted, want)

			}
			testify.Equal_Values(t, want, int(Bit_Size_64(Word_64(shifted))),
				"Bit_Size_64(%#016x) is not %d", shifted, want)
			testify.Equal_Values(t, want, int(Bit_Size(Word(shifted))),
				"Bit_Size(%#016x) is not %d", shifted, want)

		}
	}
}

// Test_Standard_Rotate_Left turns the de Bruijn sequence by every distance from zero to
// one hundred twenty seven at each width, and turns the result back by the same distance.
func Test_Standard_Rotate_Left(t *testing.T) {
	t.Parallel()
	// The conversion runs through a variable, because Go rejects a constant that does not
	// fit the narrower type even when the truncation is the intent.
	sequence := DE_BRUIJN_64
	for distance := uint(0); distance < 128; distance++ {
		rotation := Rotation(distance)
		value_8 := Word_8(sequence)
		want_8 := value_8<<(distance&0x7) | value_8>>(8-distance&0x7)
		testify.Equal_Values(t, want_8, Rotate_Left_8(value_8, rotation),
			"Rotate_Left_8(%#02x, %d) is wrong", value_8, distance)
		testify.Equal_Values(t, value_8, Rotate_Left_8(want_8, -rotation),
			"Rotate_Left_8(%#02x, -%d) does not undo", want_8, distance)

		value_16 := Word_16(sequence)
		want_16 := value_16<<(distance&0xf) | value_16>>(16-distance&0xf)
		testify.Equal_Values(t, want_16, Rotate_Left_16(value_16, rotation),
			"Rotate_Left_16(%#04x, %d) is wrong", value_16, distance)
		testify.Equal_Values(t, value_16, Rotate_Left_16(want_16, -rotation),
			"Rotate_Left_16(%#04x, -%d) does not undo", want_16, distance)

		value_32 := Word_32(sequence)
		want_32 := value_32<<(distance&0x1f) | value_32>>(32-distance&0x1f)
		testify.Equal_Values(t, want_32, Rotate_Left_32(value_32, rotation),
			"Rotate_Left_32(%#08x, %d) is wrong", value_32, distance)
		testify.Equal_Values(t, value_32, Rotate_Left_32(want_32, -rotation),
			"Rotate_Left_32(%#08x, -%d) does not undo", want_32, distance)

		value_64 := Word_64(DE_BRUIJN_64)
		want_64 := value_64<<(distance&0x3f) | value_64>>(64-distance&0x3f)
		testify.Equal_Values(t, want_64, Rotate_Left_64(value_64, rotation),
			"Rotate_Left_64(%#016x, %d) is wrong", value_64, distance)
		testify.Equal_Values(t, value_64, Rotate_Left_64(want_64, -rotation),
			"Rotate_Left_64(%#016x, -%d) does not undo", want_64, distance)

		word := Word(DE_BRUIJN_64)
		want_word := word<<(distance&0x3f) | word>>(64-distance&0x3f)
		testify.Equal_Values(t, want_word, Rotate_Left(word, rotation),
			"Rotate_Left(%#016x, %d) is wrong", word, distance)
		testify.Equal_Values(t, word, Rotate_Left(want_word, -rotation),
			"Rotate_Left(%#016x, -%d) does not undo", want_word, distance)

	}
}

// Test_Standard_Reverse mirrors each single bit to its opposite position, then checks the
// patterns the standard library pins, in both directions.
func Test_Standard_Reverse(t *testing.T) {
	t.Parallel()
	for position := uint(0); position < 64; position++ {
		compare_reverse(t, uint64(1)<<position, uint64(1)<<(63-position))
	}
	for _, pair := range [][PAIR_SIZE]uint64{
		{0, 0},
		{0x1, 0x8 << 60}, {0x2, 0x4 << 60}, {0x3, 0xc << 60}, {0x4, 0x2 << 60},
		{0x5, 0xa << 60}, {0x6, 0x6 << 60}, {0x7, 0xe << 60}, {0x8, 0x1 << 60},
		{0x9, 0x9 << 60}, {0xa, 0x5 << 60}, {0xb, 0xd << 60}, {0xc, 0x3 << 60},
		{0xd, 0xb << 60}, {0xe, 0x7 << 60}, {0xf, 0xf << 60},
		{0x5686487, 0xe12616a000000000},
		{0x0123456789abcdef, 0xf7b3d591e6a2c480},
	} {
		compare_reverse(t, pair[0], pair[1])
		compare_reverse(t, pair[1], pair[0])
	}
}

// Compares the reversal of one value at every width. The narrower widths take the top
// bits of the expected 64-bit result, because the mirror of a shorter word is the high
// end of the mirror of the longer one.
func compare_reverse(t *testing.T, value uint64, want uint64) {
	t.Helper()
	testify.Equal_Values(t, Word_8(want>>(64-8)), Reverse_8(Word_8(value)),
		"Reverse_8(%#02x) is wrong", uint8(value))
	testify.Equal_Values(t, Word_16(want>>(64-16)), Reverse_16(Word_16(value)),
		"Reverse_16(%#04x) is wrong", uint16(value))
	testify.Equal_Values(t, Word_32(want>>(64-32)), Reverse_32(Word_32(value)),
		"Reverse_32(%#08x) is wrong", uint32(value))
	testify.Equal_Values(t, Word_64(want), Reverse_64(Word_64(value)),
		"Reverse_64(%#016x) is wrong", value)
	testify.Equal_Values(t, Word(want), Reverse(Word(value)),
		"Reverse(%#016x) is wrong", value)

}

// Test_Standard_Reverse_Bytes mirrors the byte positions of the de Bruijn sequence at each
// width and checks that two applications return the operand.
func Test_Standard_Reverse_Bytes(t *testing.T) {
	t.Parallel()
	for shift := uint(0); shift < 64; shift++ {
		value := Word_64(DE_BRUIJN_64 >> shift)
		testify.Equal_Values(t, value, Reverse_Bytes_64(Reverse_Bytes_64(value)),
			"Reverse_Bytes_64 twice on %#016x does not return it", value)

		narrow := Word_32(value)
		testify.Equal_Values(t, narrow, Reverse_Bytes_32(Reverse_Bytes_32(narrow)),
			"Reverse_Bytes_32 twice on %#08x does not return it", narrow)

		short := Word_16(value)
		testify.Equal_Values(t, short, Reverse_Bytes_16(Reverse_Bytes_16(short)),
			"Reverse_Bytes_16 twice on %#04x does not return it", short)

		word := Word(value)
		testify.Equal_Values(t, word, Reverse_Bytes(Reverse_Bytes(word)),
			"Reverse_Bytes twice on %#016x does not return it", word)

	}
	testify.Equal_Values(t, 0x0201, Reverse_Bytes_16(0x0102),
		"Reverse_Bytes_16(0x0102) must be 0x0201")
	testify.Equal_Values(t, 0x04030201, Reverse_Bytes_32(0x01020304),
		"Reverse_Bytes_32(0x01020304) must be 0x04030201")
	testify.Equal_Values(t, 0x0807060504030201, Reverse_Bytes_64(0x0102030405060708),
		"Reverse_Bytes_64 must mirror the eight bytes")

}

// Test_Standard_Add_Subtract_64 checks the sum and the carry against the standard library
// table, in both operand orders, and checks that subtraction undoes each addition.
func Test_Standard_Add_Subtract_64(t *testing.T) {
	t.Parallel()
	largest := WORD_64_MAXIMUM
	for _, row := range [][ARITHMETIC_ROW_SIZE]uint64{
		{0, 0, 0, 0, 0},
		{0, 1, 0, 1, 0},
		{0, 0, 1, 1, 0},
		{0, 1, 1, 2, 0},
		{12345, 67890, 0, 80235, 0},
		{12345, 67890, 1, 80236, 0},
		{largest, 1, 0, 0, 1},
		{largest, 0, 1, 0, 1},
		{largest, 1, 1, 1, 1},
		{largest, largest, 0, largest - 1, 1},
		{largest, largest, 1, largest, 1},
	} {
		first, second, carry, sum, carry_output := row[0], row[1], row[2], row[3], row[4]
		compare_add_64(t, first, second, carry, sum, carry_output)
		compare_add_64(t, second, first, carry, sum, carry_output)
		compare_subtract_64(t, sum, first, carry, second, carry_output)
		compare_subtract_64(t, sum, second, carry, first, carry_output)
	}
}

// Compares one 64-bit addition against the expected sum and carry.
func compare_add_64(
	t *testing.T, augend uint64, addend uint64, carry uint64,
	want_sum uint64, want_carry uint64,
) {
	t.Helper()
	sum, carry_output := Add_64(Word_64(augend), Addend_64(addend), Carry_In(carry))
	testify.Equal_Values(t, want_sum, uint64(sum),
		"Add_64(%#x,%#x,%#x) sum = %#x, want %#x",
		augend, addend, carry, sum, want_sum)
	testify.Equal_Values(t, want_carry, uint64(carry_output),
		"Add_64(%#x,%#x,%#x) carry = %#x, want %#x",
		augend, addend, carry, carry_output, want_carry)

}

// Compares one 64-bit subtraction against the expected difference and borrow.
func compare_subtract_64(
	t *testing.T, minuend uint64, subtrahend uint64, borrow uint64,
	want_difference uint64, want_borrow uint64,
) {
	t.Helper()
	difference, borrow_output := Subtract_64(
		Word_64(minuend), Subtrahend_64(subtrahend), Borrow_In(borrow))
	testify.Equal_Values(t, want_difference, uint64(difference),
		"Subtract_64(%#x,%#x,%#x) = %#x, want %#x",
		minuend, subtrahend, borrow, difference, want_difference)
	testify.Equal_Values(t, want_borrow, uint64(borrow_output),
		"Subtract_64(%#x,%#x,%#x) borrow = %#x, want %#x",
		minuend, subtrahend, borrow, borrow_output, want_borrow)

}

// Test_Standard_Add_Subtract_32 checks the 32-bit sum and carry against the standard
// library table in both operand orders.
func Test_Standard_Add_Subtract_32(t *testing.T) {
	t.Parallel()
	largest := WORD_32_ALL_ONES
	for _, row := range [][ARITHMETIC_ROW_SIZE]uint32{
		{0, 0, 0, 0, 0},
		{0, 1, 0, 1, 0},
		{0, 0, 1, 1, 0},
		{0, 1, 1, 2, 0},
		{12345, 67890, 0, 80235, 0},
		{12345, 67890, 1, 80236, 0},
		{largest, 1, 0, 0, 1},
		{largest, 0, 1, 0, 1},
		{largest, 1, 1, 1, 1},
		{largest, largest, 0, largest - 1, 1},
		{largest, largest, 1, largest, 1},
	} {
		first, second, carry, sum, carry_output := row[0], row[1], row[2], row[3], row[4]
		compare_add_32(t, first, second, carry, sum, carry_output)
		compare_add_32(t, second, first, carry, sum, carry_output)
		compare_subtract_32(t, sum, first, carry, second, carry_output)
		compare_subtract_32(t, sum, second, carry, first, carry_output)
	}
}

// Compares one 32-bit addition against the expected sum and carry.
func compare_add_32(
	t *testing.T, augend uint32, addend uint32, carry uint32,
	want_sum uint32, want_carry uint32,
) {
	t.Helper()
	sum, carry_output := Add_32(Word_32(augend), Addend_32(addend), Carry_In(carry))
	testify.Equal_Values(t, want_sum, uint32(sum),
		"Add_32(%#x,%#x,%#x) sum = %#x, want %#x",
		augend, addend, carry, sum, want_sum)
	testify.Equal_Values(t, want_carry, uint32(carry_output),
		"Add_32(%#x,%#x,%#x) carry = %#x, want %#x",
		augend, addend, carry, carry_output, want_carry)

}

// Compares one 32-bit subtraction against the expected difference and borrow.
func compare_subtract_32(
	t *testing.T, minuend uint32, subtrahend uint32, borrow uint32,
	want_difference uint32, want_borrow uint32,
) {
	t.Helper()
	difference, borrow_output := Subtract_32(
		Word_32(minuend), Subtrahend_32(subtrahend), Borrow_In(uint64(borrow)))
	testify.Equal_Values(t, want_difference, uint32(difference),
		"Subtract_32(%#x,%#x,%#x) = %#x, want %#x",
		minuend, subtrahend, borrow, difference, want_difference)
	testify.Equal_Values(t, want_borrow, uint32(borrow_output),
		"Subtract_32(%#x,%#x,%#x) borrow = %#x, want %#x",
		minuend, subtrahend, borrow, borrow_output, want_borrow)

}

// Test_Standard_Add_Subtract_Word checks the machine-word sum and carry against the
// standard library table in both operand orders.
func Test_Standard_Add_Subtract_Word(t *testing.T) {
	t.Parallel()
	largest := uint64(WORD_MAXIMUM)
	for _, row := range [][ARITHMETIC_ROW_SIZE]uint64{
		{0, 0, 0, 0, 0},
		{0, 1, 0, 1, 0},
		{0, 0, 1, 1, 0},
		{0, 1, 1, 2, 0},
		{12345, 67890, 0, 80235, 0},
		{12345, 67890, 1, 80236, 0},
		{largest, 1, 0, 0, 1},
		{largest, 0, 1, 0, 1},
		{largest, 1, 1, 1, 1},
		{largest, largest, 0, largest - 1, 1},
		{largest, largest, 1, largest, 1},
	} {
		first, second, carry, sum, carry_output := row[0], row[1], row[2], row[3], row[4]
		compare_add_word(t, first, second, carry, sum, carry_output)
		compare_add_word(t, second, first, carry, sum, carry_output)
	}
}

// Compares one machine-word addition and the subtraction that undoes it.
func compare_add_word(
	t *testing.T, augend uint64, addend uint64, carry uint64,
	want_sum uint64, want_carry uint64,
) {
	t.Helper()
	sum, carry_output := Add_Word(Word(augend), Addend_Word(addend), Carry_In(carry))
	testify.Equal_Values(t, want_sum, uint64(sum),
		"Add_Word(%#x,%#x,%#x) sum = %#x, want %#x",
		augend, addend, carry, sum, want_sum)
	testify.Equal_Values(t, want_carry, uint64(carry_output),
		"Add_Word(%#x,%#x,%#x) carry = %#x, want %#x",
		augend, addend, carry, carry_output, want_carry)

	difference, borrow := Subtract_Word(
		Word(want_sum), Subtrahend_Word(augend), Borrow_In(carry))
	testify.Equal_Values(t, addend, uint64(difference),
		"Subtract_Word(%#x,%#x,%#x) = %#x, want %#x",
		want_sum, augend, carry, difference, addend)
	testify.Equal_Values(t, want_carry, uint64(borrow),
		"Subtract_Word(%#x,%#x,%#x) borrow = %#x, want %#x",
		want_sum, augend, carry, borrow, want_carry)

}

// Test_Standard_Multiply_Divide_64 checks the 64-bit product and the division that undoes
// it against the standard library table, in both factor orders.
func Test_Standard_Multiply_Divide_64(t *testing.T) {
	t.Parallel()
	largest := WORD_64_MAXIMUM
	for _, row := range [][ARITHMETIC_ROW_SIZE]uint64{
		{1 << 63, 2, 1, 0, 1},
		{0x3626229738a3b9, 0xd8988a9f1cc4a61, 0x2dd0712657fe8, 0x9dd6a3364c358319, 13},
		{largest, largest, largest - 1, 1, 42},
	} {
		first, second := row[0], row[1]
		high, low, rest := row[2], row[3], row[4]
		compare_multiply_64(t, first, second, high, low)
		compare_multiply_64(t, second, first, high, low)
		compare_divide_64(t, high, low+rest, second, first, rest)
		compare_divide_64(t, high, low+rest, first, second, rest)
	}
}

// Compares one 64-bit product against the expected two words.
func compare_multiply_64(
	t *testing.T, multiplicand uint64, multiplier uint64,
	want_high uint64, want_low uint64,
) {
	t.Helper()
	high, low := Multiply_64(Word_64(multiplicand), Multiplier_64(multiplier))
	testify.Equal_Values(t, want_high, uint64(high),
		"Multiply_64(%#x,%#x) high = %#x, want %#x",
		multiplicand, multiplier, high, want_high)
	testify.Equal_Values(t, want_low, uint64(low),
		"Multiply_64(%#x,%#x) low = %#x, want %#x",
		multiplicand, multiplier, low, want_low)

}

// Compares one 64-bit division against the expected quotient and remainder, and checks
// that the remainder form agrees.
func compare_divide_64(
	t *testing.T, high uint64, low uint64, divisor uint64,
	want_quotient uint64, want_rest uint64,
) {
	t.Helper()
	quotient, rest := Divide_64(
		Dividend_High_64(high), Dividend_Low_64(low), Divisor_64(divisor))
	testify.Equal_Values(t, want_quotient, uint64(quotient),
		"Divide_64(%#x,%#x,%#x) quotient = %#x, want %#x",
		high, low, divisor, quotient, want_quotient)
	testify.Equal_Values(t, want_rest, uint64(rest),
		"Divide_64(%#x,%#x,%#x) remainder = %#x, want %#x",
		high, low, divisor, rest, want_rest)

	alone := Remainder_64(High_Word_64(high), Low_Word_64(low), Divisor_64(divisor))
	testify.Equal_Values(t, want_rest, uint64(alone),
		"Remainder_64(%#x,%#x,%#x) = %#x, want %#x",
		high, low, divisor, alone, want_rest)

}

// Test_Standard_Multiply_Divide_32 checks the 32-bit product and the division that undoes
// it against the standard library table, in both factor orders.
func Test_Standard_Multiply_Divide_32(t *testing.T) {
	t.Parallel()
	largest := WORD_32_ALL_ONES
	for _, row := range [][ARITHMETIC_ROW_SIZE]uint32{
		{1 << 31, 2, 1, 0, 1},
		{0xc47dfa8c, 50911, 0x98a4, 0x998587f4, 13},
		{largest, largest, largest - 1, 1, 42},
	} {
		first, second := row[0], row[1]
		high, low, rest := row[2], row[3], row[4]
		compare_multiply_32(t, first, second, high, low)
		compare_multiply_32(t, second, first, high, low)
		compare_divide_32(t, high, low+rest, second, first, rest)
		compare_divide_32(t, high, low+rest, first, second, rest)
	}
}

// Compares one 32-bit product against the expected two words.
func compare_multiply_32(
	t *testing.T, multiplicand uint32, multiplier uint32,
	want_high uint32, want_low uint32,
) {
	t.Helper()
	high, low := Multiply_32(Word_32(multiplicand), Multiplier_32(multiplier))
	testify.Equal_Values(t, want_high, uint32(high),
		"Multiply_32(%#x,%#x) high = %#x, want %#x",
		multiplicand, multiplier, high, want_high)
	testify.Equal_Values(t, want_low, uint32(low),
		"Multiply_32(%#x,%#x) low = %#x, want %#x",
		multiplicand, multiplier, low, want_low)

}

// Compares one 32-bit division against the expected quotient and remainder.
func compare_divide_32(
	t *testing.T, high uint32, low uint32, divisor uint32,
	want_quotient uint32, want_rest uint32,
) {
	t.Helper()
	quotient, rest := Divide_32(
		Dividend_High_32(high), Dividend_Low_32(low), Divisor_32(divisor))
	testify.Equal_Values(t, want_quotient, uint32(quotient),
		"Divide_32(%#x,%#x,%#x) quotient = %#x, want %#x",
		high, low, divisor, quotient, want_quotient)
	testify.Equal_Values(t, want_rest, uint32(rest),
		"Divide_32(%#x,%#x,%#x) remainder = %#x, want %#x",
		high, low, divisor, rest, want_rest)

	alone := Remainder_32(High_Word_32(high), Low_Word_32(low), Divisor_32(divisor))
	testify.Equal_Values(t, want_rest, uint32(alone),
		"Remainder_32(%#x,%#x,%#x) = %#x, want %#x",
		high, low, divisor, alone, want_rest)

}

// Test_Standard_Multiply_Divide_Word checks the machine-word product and the division that
// undoes it against the standard library table.
func Test_Standard_Multiply_Divide_Word(t *testing.T) {
	t.Parallel()
	largest := uint64(WORD_MAXIMUM)
	for _, row := range [][ARITHMETIC_ROW_SIZE]uint64{
		{1 << (WORD_SIZE - 1), 2, 1, 0, 1},
		{largest, largest, largest - 1, 1, 42},
	} {
		first, second := row[0], row[1]
		high, low, rest := row[2], row[3], row[4]
		compare_multiply_word(t, first, second, high, low)
		compare_multiply_word(t, second, first, high, low)
		compare_divide_word(t, high, low+rest, second, first, rest)
		compare_divide_word(t, high, low+rest, first, second, rest)
	}
}

// Compares one machine-word product against the expected two words.
func compare_multiply_word(
	t *testing.T, multiplicand uint64, multiplier uint64,
	want_high uint64, want_low uint64,
) {
	t.Helper()
	high, low := Multiply_Word(Word(multiplicand), Multiplier_Word(multiplier))
	testify.Equal_Values(t, want_high, uint64(high),
		"Multiply_Word(%#x,%#x) high = %#x, want %#x",
		multiplicand, multiplier, high, want_high)
	testify.Equal_Values(t, want_low, uint64(low),
		"Multiply_Word(%#x,%#x) low = %#x, want %#x",
		multiplicand, multiplier, low, want_low)

}

// Compares one machine-word division against the expected quotient and remainder.
func compare_divide_word(
	t *testing.T, high uint64, low uint64, divisor uint64,
	want_quotient uint64, want_rest uint64,
) {
	t.Helper()
	quotient, rest := Divide_Word(
		Dividend_High_Word(high), Dividend_Low_Word(low), Divisor_Word(divisor))
	testify.Equal_Values(t, want_quotient, uint64(quotient),
		"Divide_Word(%#x,%#x,%#x) quotient = %#x, want %#x",
		high, low, divisor, quotient, want_quotient)
	testify.Equal_Values(t, want_rest, uint64(rest),
		"Divide_Word(%#x,%#x,%#x) remainder = %#x, want %#x",
		high, low, divisor, rest, want_rest)

	alone := Remainder_Word(High_Word(high), Low_Word(low), Divisor_Word(divisor))
	testify.Equal_Values(t, want_rest, uint64(alone),
		"Remainder_Word(%#x,%#x,%#x) = %#x, want %#x",
		high, low, divisor, alone, want_rest)

}

// Test_Standard_Divide_Panics verifies that a quotient wider than one word and a zero
// divisor each panic, at every width. The standard library raises a runtime error, and
// this package raises an assertion failure, thus the test asks only that it panics.
func Test_Standard_Divide_Panics(t *testing.T) {
	t.Parallel()
	testify.Panics(t, func() { Divide_Word(1, 0, 1) },
		"Divide_Word must panic when the divisor is not above the high word")
	testify.Panics(t, func() { Divide_32(1, 0, 1) },
		"Divide_32 must panic when the divisor is not above the high word")
	testify.Panics(t, func() { Divide_64(1, 0, 1) },
		"Divide_64 must panic when the divisor is not above the high word")
	testify.Panics(t, func() { Divide_Word(1, 0, 0) },
		"Divide_Word must panic on a zero divisor")
	testify.Panics(t, func() { Divide_32(1, 0, 0) },
		"Divide_32 must panic on a zero divisor")
	testify.Panics(t, func() { Divide_64(1, 0, 0) },
		"Divide_64 must panic on a zero divisor")

}

// Test_Standard_Add_Subtract_Carry_Panics verifies that a carry or a borrow above one
// panics rather than entering the arithmetic.
func Test_Standard_Add_Subtract_Carry_Panics(t *testing.T) {
	t.Parallel()
	testify.Panics(t, func() { Add_64(1, 1, 2) },
		"Add_64 must panic on a carry above one")
	testify.Panics(t, func() { Subtract_64(1, 1, 2) },
		"Subtract_64 must panic on a borrow above one")
	testify.Panics(t, func() { Add_32(1, 1, 2) },
		"Add_32 must panic on a carry above one")
	testify.Panics(t, func() { Subtract_32(1, 1, 2) },
		"Subtract_32 must panic on a borrow above one")

}

// Test_Standard_Remainder_32 checks that the remainder form agrees with the division for a
// thousand divisors that keep the quotient in one word.
func Test_Standard_Remainder_32(t *testing.T) {
	t.Parallel()
	high, low := uint32(510510), uint32(9699690)
	divisor := uint32(510510 + 1)
	for step_index := 0; step_index < 1000; step_index++ {
		rest := Remainder_32(
			High_Word_32(high), Low_Word_32(low), Divisor_32(divisor))
		_, expected := Divide_32(
			Dividend_High_32(high), Dividend_Low_32(low), Divisor_32(divisor))
		testify.Equal_Values(t, uint32(expected), uint32(rest),
			"Remainder_32(%d,%d,%d) = %d, but the division gives %d",
			high, low, divisor, rest, expected)

		divisor += 13
	}
}

// Test_Standard_Remainder_32_Overflow checks the remainder for the dividends whose
// quotient does not fit one word, against the same division at twice the width.
func Test_Standard_Remainder_32_Overflow(t *testing.T) {
	t.Parallel()
	high, low := uint32(510510), uint32(9699690)
	divisor := uint32(7)
	for step_index := 0; step_index < 1000; step_index++ {
		rest := Remainder_32(
			High_Word_32(high), Low_Word_32(low), Divisor_32(divisor))
		wide := uint64(high)<<32 | uint64(low)
		_, expected := Divide_64(0, Dividend_Low_64(wide), Divisor_64(divisor))
		testify.Equal_Values(t, uint32(expected), uint32(rest),
			"Remainder_32(%d,%d,%d) = %d, but the wide division gives %d",
			high, low, divisor, rest, expected)

		divisor += 13
	}
}

// Test_Standard_Remainder_64 checks that the remainder form agrees with the division for a
// thousand divisors that keep the quotient in one word.
func Test_Standard_Remainder_64(t *testing.T) {
	t.Parallel()
	high, low := uint64(510510), uint64(9699690)
	divisor := uint64(510510 + 1)
	for step_index := 0; step_index < 1000; step_index++ {
		rest := Remainder_64(
			High_Word_64(high), Low_Word_64(low), Divisor_64(divisor))
		_, expected := Divide_64(
			Dividend_High_64(high), Dividend_Low_64(low), Divisor_64(divisor))
		testify.Equal_Values(t, uint64(expected), uint64(rest),
			"Remainder_64(%d,%d,%d) = %d, but the division gives %d",
			high, low, divisor, rest, expected)

		divisor += 13
	}
}

// Test_Standard_Remainder_64_Overflow checks the remainder against the values the standard
// library pins for dividends whose quotient does not fit one word.
func Test_Standard_Remainder_64_Overflow(t *testing.T) {
	t.Parallel()
	for _, row := range [][REMAINDER_ROW_SIZE]uint64{
		{42, 1119, 42, 27},
		{42, 1119, 38, 9},
		{42, 1119, 26, 23},
		{469, 0, 467, 271},
		{469, 0, 113, 58},
		{111111, 111111, 1171, 803},
		{3968194946088682615, 3192705705065114702, 1000037, 56067},
	} {
		high, low, divisor, want := row[0], row[1], row[2], row[3]
		testify.False(t, high < divisor,
			"Remainder_64(%d,%d,%d) does not overflow a quotient",
			high, low, divisor)

		rest := Remainder_64(
			High_Word_64(high), Low_Word_64(low), Divisor_64(divisor))
		testify.Equal_Values(t, want, uint64(rest),
			"Remainder_64(%d,%d,%d) = %d, want %d",
			high, low, divisor, rest, want)

	}
}

// Test_Standard_Rotation_Ends drives the two ends of the rotation domain, which the
// standard library has no counterpart for because its distance is a plain machine integer.
func Test_Standard_Rotation_Ends(t *testing.T) {
	t.Parallel()
	for _, rotation := range []Rotation{ROTATION_MINIMUM, ROTATION_MAXIMUM} {
		testify.Equal_Values(t,
			Rotate_Left_8(0xa5, rotation),
			Rotate_Left_8(0xa5, rotation),
			"Rotate_Left_8 at rotation %d is not deterministic", rotation)
		testify.Equal_Values(t,
			Rotate_Left_16(0xa5a5, rotation),
			Rotate_Left_16(0xa5a5, rotation),
			"Rotate_Left_16 at rotation %d is not deterministic", rotation)
		testify.Equal_Values(t,
			Rotate_Left_32(0xa5a5a5a5, rotation),
			Rotate_Left_32(0xa5a5a5a5, rotation),
			"Rotate_Left_32 at rotation %d is not deterministic", rotation)

		value := Word_64(DE_BRUIJN_64)
		testify.Equal_Values(t,
			Rotate_Left_64(value, rotation),
			Rotate_Left_64(value, rotation),
			"Rotate_Left_64 at rotation %d is not deterministic", rotation)

		word := Word(DE_BRUIJN_64)
		testify.Equal_Values(t, Rotate_Left(word, rotation), Rotate_Left(word, rotation),
			"Rotate_Left at rotation %d is not deterministic", rotation)

	}
}

// Sweeps the 8-bit domain ends and the small values around zero.
func sweep_8() (values []Word_8) {
	return []Word_8{0, 1, 2, 3, 0x7f, 0x80, 0xfe, 0xff}
}

// Sweeps the 16-bit domain ends and the small values around zero. The two single-byte
// values are there so a byte reversal lands on one and on two.
func sweep_16() (values []Word_16) {
	return []Word_16{
		0, 1, 2, 3, 0x0100, 0x0200, 0x7fff, 0x8000, 0xfffe, 0xffff,
	}
}

// Sweeps the 32-bit domain ends and the small values around zero.
func sweep_32() (values []Word_32) {
	return []Word_32{0, 1, 2, 3, 0x7fffffff, 0x80000000, 0xfffffffe, 0xffffffff}
}

// Sweeps the 64-bit domain ends and the small values around zero. The two top-byte values
// are there so a byte reversal lands on one and on two.
func sweep_64() (values []Word_64) {
	return []Word_64{
		0, 1, 2, 3, 1 << 56, 1 << 57, 1 << 62, 1 << 63,
		0xfffffffffffffffe, 0xffffffffffffffff,
	}
}

// Test_Domain_Ends_Of_The_Counts drives each count over the ends of its operand domain.
// The standard library tests walk a byte table, thus they never reach the widest operand.
func Test_Domain_Ends_Of_The_Counts(t *testing.T) {
	t.Parallel()
	for _, value := range sweep_8() {
		Leading_Zeros_8(value)
		Trailing_Zeros_8(value)
		Ones_Count_8(value)
		Bit_Size_8(value)
		Reverse_8(value)
	}
	for _, value := range sweep_16() {
		Leading_Zeros_16(value)
		Trailing_Zeros_16(value)
		Ones_Count_16(value)
		Bit_Size_16(value)
		Reverse_16(value)
		Reverse_Bytes_16(value)
	}
	for _, value := range sweep_32() {
		Leading_Zeros_32(value)
		Trailing_Zeros_32(value)
		Ones_Count_32(value)
		Bit_Size_32(value)
		Reverse_32(value)
		Reverse_Bytes_32(value)
	}
	for _, value := range sweep_64() {
		Leading_Zeros_64(value)
		Trailing_Zeros_64(value)
		Ones_Count_64(value)
		Bit_Size_64(value)
		Reverse_64(value)
		Reverse_Bytes_64(value)
		Leading_Zeros(Word(value))
		Trailing_Zeros(Word(value))
		Ones_Count(Word(value))
		Bit_Size(Word(value))
		Reverse(Word(value))
		Reverse_Bytes(Word(value))
	}
}

// Test_Domain_Ends_Of_The_Rotations drives each rotation over the ends of both domains.
func Test_Domain_Ends_Of_The_Rotations(t *testing.T) {
	t.Parallel()
	distances := []Rotation{
		ROTATION_MINIMUM, -65, -64, -3, -2, -1, 0, 1, 2, 3, 63, 64, 65,
		ROTATION_MAXIMUM,
	}
	for _, rotation := range distances {
		for _, value := range sweep_8() {
			Rotate_Left_8(value, rotation)
		}
		for _, value := range sweep_16() {
			Rotate_Left_16(value, rotation)
		}
		for _, value := range sweep_32() {
			Rotate_Left_32(value, rotation)
		}
		for _, value := range sweep_64() {
			Rotate_Left_64(value, rotation)
			Rotate_Left(Word(value), rotation)
		}
	}
}

// Test_Domain_Ends_Of_The_Arithmetic drives the addition, the subtraction, and the
// multiplication over the ends of every operand domain and both carries.
func Test_Domain_Ends_Of_The_Arithmetic(t *testing.T) {
	t.Parallel()
	for _, carry := range []Carry_In{0, 1} {
		for _, left := range sweep_32() {
			for _, right := range sweep_32() {
				Add_32(left, Addend_32(right), carry)
				Subtract_32(left, Subtrahend_32(right), Borrow_In(carry))
				Multiply_32(left, Multiplier_32(right))
			}
		}
		for _, left := range sweep_64() {
			for _, right := range sweep_64() {
				Add_64(left, Addend_64(right), carry)
				Subtract_64(left, Subtrahend_64(right), Borrow_In(carry))
				Multiply_64(left, Multiplier_64(right))
				Add_Word(Word(left), Addend_Word(right), carry)
				Subtract_Word(Word(left), Subtrahend_Word(right), Borrow_In(carry))
				Multiply_Word(Word(left), Multiplier_Word(right))
			}
		}
	}
}

// Test_Domain_Ends_Of_The_Division drives the division and the remainder over every sweep
// triple whose quotient fits one word, and over the wide dividends the remainder admits.
func Test_Domain_Ends_Of_The_Division(t *testing.T) {
	t.Parallel()
	for _, divisor := range sweep_64() {
		if divisor == 0 {
			continue
		}
		for _, high := range sweep_64() {
			for _, low := range sweep_64() {
				drive_division_64(high, low, divisor)
			}
		}
	}
	for _, divisor := range sweep_32() {
		if divisor == 0 {
			continue
		}
		for _, high := range sweep_32() {
			for _, low := range sweep_32() {
				drive_division_32(high, low, divisor)
			}
		}
	}
}

// Drives the 64-bit and machine-word division and remainder for one triple. The remainder
// admits any high word, thus only the division needs the quotient to fit.
func drive_division_64(high Word_64, low Word_64, divisor Word_64) {
	Remainder_64(High_Word_64(high), Low_Word_64(low), Divisor_64(divisor))
	Remainder_Word(High_Word(high), Low_Word(low), Divisor_Word(divisor))
	if uint64(high) >= uint64(divisor) {
		return
	}
	Divide_64(Dividend_High_64(high), Dividend_Low_64(low), Divisor_64(divisor))
	Divide_Word(
		Dividend_High_Word(high), Dividend_Low_Word(low), Divisor_Word(divisor))
}

// Drives the 32-bit division and remainder for one triple.
func drive_division_32(high Word_32, low Word_32, divisor Word_32) {
	Remainder_32(High_Word_32(high), Low_Word_32(low), Divisor_32(divisor))
	if uint32(high) >= uint32(divisor) {
		return
	}
	Divide_32(Dividend_High_32(high), Dividend_Low_32(low), Divisor_32(divisor))
}
