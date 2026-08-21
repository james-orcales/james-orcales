// Package bits gives the bit-level operations of the Go standard library math/bits, with
// the repository naming and an assertion at each boundary. The operations are pure integer
// arithmetic, thus the deterministic tier can hold them: one seed gives one result on
// every platform. The linter bans methods, so every operation is a free function named for
// what it does, with the value it acts on first.
package bits

import (
	"local/james-orcales/shared/simulation/aver/default"
)

// WORD_SIZE is the bit width of a machine word. This repository builds for 64-bit targets
// only, thus the width is a plain constant and no operation branches on it.
const WORD_SIZE = 64

// BIT_COUNT_MINIMUM is the smallest number of bits an operation can count.
const BIT_COUNT_MINIMUM = 0

// BIT_COUNT_8_MAXIMUM is the largest number of bits an 8-bit operation can count.
const BIT_COUNT_8_MAXIMUM = 8

// BIT_COUNT_16_MAXIMUM is the largest number of bits a 16-bit operation can count.
const BIT_COUNT_16_MAXIMUM = 16

// BIT_COUNT_32_MAXIMUM is the largest number of bits a 32-bit operation can count.
const BIT_COUNT_32_MAXIMUM = 32

// BIT_COUNT_64_MAXIMUM is the largest number of bits a 64-bit operation can count.
const BIT_COUNT_64_MAXIMUM = 64

// BIT_COUNT_WORD_MAXIMUM is the largest number of bits a machine-word operation can count.
const BIT_COUNT_WORD_MAXIMUM = WORD_SIZE

// CARRY_MINIMUM is the smallest carry or borrow.
const CARRY_MINIMUM uint64 = 0

// CARRY_MAXIMUM is the largest carry or borrow. A word addition of two operands and one
// carry cannot produce a carry above one.
const CARRY_MAXIMUM uint64 = 1

// ROTATION_MINIMUM is the smallest rotation distance.
const ROTATION_MINIMUM = -1 << 62

// ROTATION_MAXIMUM is the largest rotation distance.
const ROTATION_MAXIMUM = 1<<62 - 1

// INTEGER_8_MINIMUM is the smallest signed 8-bit integer. The signed limits sit beside the
// unsigned ones because both state the same kind of fact: what one machine width holds.
const INTEGER_8_MINIMUM int8 = -128

// INTEGER_8_MAXIMUM is the largest signed 8-bit integer.
const INTEGER_8_MAXIMUM int8 = 127

// INTEGER_16_MINIMUM is the smallest signed 16-bit integer.
const INTEGER_16_MINIMUM int16 = -32768

// INTEGER_16_MAXIMUM is the largest signed 16-bit integer.
const INTEGER_16_MAXIMUM int16 = 32767

// INTEGER_32_MINIMUM is the smallest signed 32-bit integer.
const INTEGER_32_MINIMUM int32 = -2147483648

// INTEGER_32_MAXIMUM is the largest signed 32-bit integer.
const INTEGER_32_MAXIMUM int32 = 2147483647

// INTEGER_64_MINIMUM is the smallest signed 64-bit integer.
const INTEGER_64_MINIMUM int64 = -9223372036854775808

// INTEGER_64_MAXIMUM is the largest signed 64-bit integer.
const INTEGER_64_MAXIMUM int64 = 9223372036854775807

// WORD_8_MINIMUM is the smallest 8-bit word.
const WORD_8_MINIMUM uint8 = 0

// WORD_8_MAXIMUM is the largest 8-bit word.
const WORD_8_MAXIMUM uint8 = 255

// WORD_16_MINIMUM is the smallest 16-bit word.
const WORD_16_MINIMUM uint16 = 0

// WORD_16_MAXIMUM is the largest 16-bit word.
const WORD_16_MAXIMUM uint16 = 65535

// WORD_32_MINIMUM is the smallest 32-bit word.
const WORD_32_MINIMUM uint32 = 0

// WORD_32_MAXIMUM is the largest 32-bit word.
const WORD_32_MAXIMUM uint32 = 4294967295

// WORD_64_MINIMUM is the smallest 64-bit word.
const WORD_64_MINIMUM uint64 = 0

// WORD_64_MAXIMUM is the largest 64-bit word.
const WORD_64_MAXIMUM uint64 = 18446744073709551615

// WORD_MINIMUM is the smallest machine word.
const WORD_MINIMUM uint = 0

// WORD_MAXIMUM is the largest machine word. The word is 64 bits wide here, thus each
// machine limit is the 64-bit limit under another name.
const WORD_MAXIMUM uint = uint(WORD_64_MAXIMUM)

// INTEGER_MAXIMUM is the largest signed machine integer.
const INTEGER_MAXIMUM int = int(INTEGER_64_MAXIMUM)

// INTEGER_MINIMUM is the smallest signed machine integer.
const INTEGER_MINIMUM int = int(INTEGER_64_MINIMUM)

// DECIMAL_DIGIT_BINARY_LOGARITHM_SHIFT selects proven fixed-point log10(2) precision.
const DECIMAL_DIGIT_BINARY_LOGARITHM_SHIFT = 12

// DECIMAL_DIGIT_BINARY_LOGARITHM_SCALE derives denominator from selected precision.
const DECIMAL_DIGIT_BINARY_LOGARITHM_SCALE = 1 << DECIMAL_DIGIT_BINARY_LOGARITHM_SHIFT

// DECIMAL_DIGIT_BINARY_LOGARITHM_CEILING is least scale-12 integer above log10(2).
const DECIMAL_DIGIT_BINARY_LOGARITHM_CEILING = 1233

// KILOBYTE_BYTES uses the SI base because the kilo prefix specifies 1000 bytes.
const KILOBYTE_BYTES = 10 * 10 * 10

// MEGABYTE_BYTES derives from KILOBYTE_BYTES so the decimal ladder has one base.
const MEGABYTE_BYTES = KILOBYTE_BYTES * 10 * 10 * 10

// GIGABYTE_BYTES derives from MEGABYTE_BYTES so the decimal ladder has one base.
const GIGABYTE_BYTES = MEGABYTE_BYTES * 10 * 10 * 10

// TERABYTE_BYTES derives from GIGABYTE_BYTES so the decimal ladder has one base.
const TERABYTE_BYTES = GIGABYTE_BYTES * 10 * 10 * 10

// PETABYTE_BYTES derives from TERABYTE_BYTES so the decimal ladder has one base.
const PETABYTE_BYTES = TERABYTE_BYTES * 10 * 10 * 10

// EXABYTE_BYTES derives from PETABYTE_BYTES so the decimal ladder has one base.
const EXABYTE_BYTES = PETABYTE_BYTES * 10 * 10 * 10

// KIBIBYTE_BYTES uses the IEC base to keep binary quantities distinct from SI quantities.
const KIBIBYTE_BYTES = 1 << 10

// MEBIBYTE_BYTES uses its IEC exponent so the source states the binary definition.
const MEBIBYTE_BYTES = 1 << 20

// GIBIBYTE_BYTES uses its IEC exponent so the source states the binary definition.
const GIBIBYTE_BYTES = 1 << 30

// TEBIBYTE_BYTES uses its IEC exponent so the source states the binary definition.
const TEBIBYTE_BYTES = 1 << 40

// PEBIBYTE_BYTES uses its IEC exponent so the source states the binary definition.
const PEBIBYTE_BYTES = 1 << 50

// EXBIBYTE_BYTES uses its IEC exponent so the source states the binary definition.
const EXBIBYTE_BYTES = 1 << 60

// DIVISOR_MINIMUM is the smallest divisor. A zero divisor has no quotient, thus the divisor
// domain starts one above the word domain.
const DIVISOR_MINIMUM = 1

// PRODUCT_HIGH_32_MAXIMUM is the largest high word of a 32-bit product. The largest product
// is the square of the largest word, which is two words short of the full double width,
// thus the high word never reaches its own maximum.
const PRODUCT_HIGH_32_MAXIMUM uint32 = WORD_32_MAXIMUM - 1

// PRODUCT_HIGH_64_MAXIMUM is the largest high word of a 64-bit product.
const PRODUCT_HIGH_64_MAXIMUM uint64 = WORD_64_MAXIMUM - 1

// PRODUCT_HIGH_WORD_MAXIMUM is the largest high word of a machine-word product.
const PRODUCT_HIGH_WORD_MAXIMUM uint = WORD_MAXIMUM - 1

// DIVIDEND_HIGH_32_MAXIMUM is the largest high word a one-word quotient admits. The high
// word must stay below the divisor, thus it stops one short of the largest word.
const DIVIDEND_HIGH_32_MAXIMUM uint32 = PRODUCT_HIGH_32_MAXIMUM

// DIVIDEND_HIGH_64_MAXIMUM is the largest 64-bit high word a one-word quotient admits.
const DIVIDEND_HIGH_64_MAXIMUM uint64 = PRODUCT_HIGH_64_MAXIMUM

// DIVIDEND_HIGH_WORD_MAXIMUM is the largest machine-word high word a quotient admits.
const DIVIDEND_HIGH_WORD_MAXIMUM uint = PRODUCT_HIGH_WORD_MAXIMUM

// DIVISION_REMAINDER_32_MAXIMUM is the largest 32-bit remainder. A remainder stays below
// its divisor, thus it stops one short of the largest word.
const DIVISION_REMAINDER_32_MAXIMUM uint32 = PRODUCT_HIGH_32_MAXIMUM

// DIVISION_REMAINDER_64_MAXIMUM is the largest 64-bit remainder.
const DIVISION_REMAINDER_64_MAXIMUM uint64 = PRODUCT_HIGH_64_MAXIMUM

// DIVISION_REMAINDER_WORD_MAXIMUM is the largest machine-word remainder.
const DIVISION_REMAINDER_WORD_MAXIMUM uint = PRODUCT_HIGH_WORD_MAXIMUM

// Word_8 is an 8-bit unsigned operand.
type Word_8 uint8

// Word_8_Invariants states the complete 8-bit word domain.
func Word_8_Invariants(value Word_8, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), WORD_8_MINIMUM, WORD_8_MAXIMUM).
		Ensure()
}

// Word_16 is a 16-bit unsigned operand.
type Word_16 uint16

// Word_16_Invariants states the complete 16-bit word domain.
func Word_16_Invariants(value Word_16, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), WORD_16_MINIMUM, WORD_16_MAXIMUM).
		Ensure()
}

// Word_32 is a 32-bit unsigned operand.
type Word_32 uint32

// Word_32_Invariants states the complete 32-bit word domain.
func Word_32_Invariants(value Word_32, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), WORD_32_MINIMUM, WORD_32_MAXIMUM).
		Ensure()
}

// Word_64 is a 64-bit unsigned operand.
type Word_64 uint64

// Word_64_Invariants states the complete 64-bit word domain.
func Word_64_Invariants(value Word_64, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), WORD_64_MINIMUM, WORD_64_MAXIMUM).
		Ensure()
}

// Word is a machine-word unsigned operand.
type Word uint

// Word_Invariants states the complete machine-word domain.
func Word_Invariants(value Word, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint(uint(value), WORD_MINIMUM, WORD_MAXIMUM).
		Ensure()
}

// Bit_Count_8 is a number of bits an 8-bit operation counts.
type Bit_Count_8 int

// Bit_Count_8_Invariants bounds a count to the 8-bit operand width.
func Bit_Count_8_Invariants(value Bit_Count_8, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), BIT_COUNT_MINIMUM, BIT_COUNT_8_MAXIMUM).
		Ensure()
}

// Bit_Count_16 is a number of bits a 16-bit operation counts.
type Bit_Count_16 int

// Bit_Count_16_Invariants bounds a count to the 16-bit operand width.
func Bit_Count_16_Invariants(value Bit_Count_16, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), BIT_COUNT_MINIMUM, BIT_COUNT_16_MAXIMUM).
		Ensure()
}

// Bit_Count_32 is a number of bits a 32-bit operation counts.
type Bit_Count_32 int

// Bit_Count_32_Invariants bounds a count to the 32-bit operand width.
func Bit_Count_32_Invariants(value Bit_Count_32, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), BIT_COUNT_MINIMUM, BIT_COUNT_32_MAXIMUM).
		Ensure()
}

// Bit_Count_64 is a number of bits a 64-bit operation counts.
type Bit_Count_64 int

// Bit_Count_64_Invariants bounds a count to the 64-bit operand width.
func Bit_Count_64_Invariants(value Bit_Count_64, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), BIT_COUNT_MINIMUM, BIT_COUNT_64_MAXIMUM).
		Ensure()
}

// Bit_Count_Word is a number of bits a machine-word operation counts.
type Bit_Count_Word int

// Bit_Count_Word_Invariants bounds a count to the machine-word width.
func Bit_Count_Word_Invariants(value Bit_Count_Word, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), BIT_COUNT_MINIMUM, BIT_COUNT_WORD_MAXIMUM).
		Ensure()
}

// Rotation is a rotation distance in bits. A negative distance turns the bits right.
type Rotation int

// Rotation_Invariants bounds a rotation distance. The operation masks the distance to the
// operand width, thus the bound keeps the value readable and rejects nothing real.
func Rotation_Invariants(value Rotation, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), ROTATION_MINIMUM, ROTATION_MAXIMUM).
		Ensure()
}

// Carry_In is the carry that enters an addition, zero or one.
type Carry_In uint64

// Carry_In_Invariants bounds an entering carry to zero or one.
func Carry_In_Invariants(value Carry_In, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint64(uint64(value), CARRY_MINIMUM, CARRY_MAXIMUM).
		Ensure()
}

// Carry_Output is the carry that leaves an addition, zero or one.
type Carry_Output uint64

// Carry_Output_Invariants bounds a leaving carry to zero or one.
func Carry_Output_Invariants(value Carry_Output, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint64(uint64(value), CARRY_MINIMUM, CARRY_MAXIMUM).
		Ensure()
}

// Borrow_In is the borrow that enters a subtraction, zero or one.
type Borrow_In uint64

// Borrow_In_Invariants bounds an entering borrow to zero or one.
func Borrow_In_Invariants(value Borrow_In, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint64(uint64(value), CARRY_MINIMUM, CARRY_MAXIMUM).
		Ensure()
}

// Borrow_Output is the borrow that leaves a subtraction, zero or one.
type Borrow_Output uint64

// Borrow_Output_Invariants bounds a leaving borrow to zero or one.
func Borrow_Output_Invariants(value Borrow_Output, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint64(uint64(value), CARRY_MINIMUM, CARRY_MAXIMUM).
		Ensure()
}

// Addend_32 is the 32-bit operand an addition adds to an augend.
type Addend_32 uint32

// Addend_32_Invariants states the complete 32-bit word domain.
func Addend_32_Invariants(value Addend_32, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), WORD_32_MINIMUM, WORD_32_MAXIMUM).
		Ensure()
}

// Addend_64 is the 64-bit operand an addition adds to an augend.
type Addend_64 uint64

// Addend_64_Invariants states the complete 64-bit word domain.
func Addend_64_Invariants(value Addend_64, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), WORD_64_MINIMUM, WORD_64_MAXIMUM).
		Ensure()
}

// Addend_Word is the machine-word operand an addition adds to an augend.
type Addend_Word uint

// Addend_Word_Invariants states the complete machine-word domain.
func Addend_Word_Invariants(value Addend_Word, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint(uint(value), WORD_MINIMUM, WORD_MAXIMUM).
		Ensure()
}

// Subtrahend_32 is the 32-bit operand a subtraction removes from a minuend.
type Subtrahend_32 uint32

// Subtrahend_32_Invariants states the complete 32-bit word domain.
func Subtrahend_32_Invariants(value Subtrahend_32, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), WORD_32_MINIMUM, WORD_32_MAXIMUM).
		Ensure()
}

// Subtrahend_64 is the 64-bit operand a subtraction removes from a minuend.
type Subtrahend_64 uint64

// Subtrahend_64_Invariants states the complete 64-bit word domain.
func Subtrahend_64_Invariants(value Subtrahend_64, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), WORD_64_MINIMUM, WORD_64_MAXIMUM).
		Ensure()
}

// Subtrahend_Word is the machine-word operand a subtraction removes from a minuend.
type Subtrahend_Word uint

// Subtrahend_Word_Invariants states the complete machine-word domain.
func Subtrahend_Word_Invariants(value Subtrahend_Word, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint(uint(value), WORD_MINIMUM, WORD_MAXIMUM).
		Ensure()
}

// Multiplier_32 is the second 32-bit factor of a product.
type Multiplier_32 uint32

// Multiplier_32_Invariants states the complete 32-bit word domain.
func Multiplier_32_Invariants(value Multiplier_32, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), WORD_32_MINIMUM, WORD_32_MAXIMUM).
		Ensure()
}

// Multiplier_64 is the second 64-bit factor of a product.
type Multiplier_64 uint64

// Multiplier_64_Invariants states the complete 64-bit word domain.
func Multiplier_64_Invariants(value Multiplier_64, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), WORD_64_MINIMUM, WORD_64_MAXIMUM).
		Ensure()
}

// Multiplier_Word is the second machine-word factor of a product.
type Multiplier_Word uint

// Multiplier_Word_Invariants states the complete machine-word domain.
func Multiplier_Word_Invariants(value Multiplier_Word, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint(uint(value), WORD_MINIMUM, WORD_MAXIMUM).
		Ensure()
}

// Product_High_32 is the upper 32 bits of a 32-bit product.
type Product_High_32 uint32

// Product_High_32_Invariants bounds the high word of a product. The largest product falls
// two words short of the full double width, thus the high word stops one short.
func Product_High_32_Invariants(value Product_High_32, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), WORD_32_MINIMUM, PRODUCT_HIGH_32_MAXIMUM).
		Ensure()
}

// Product_High_64 is the upper 64 bits of a 64-bit product.
type Product_High_64 uint64

// Product_High_64_Invariants bounds the high word of a product.
func Product_High_64_Invariants(value Product_High_64, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), WORD_64_MINIMUM, PRODUCT_HIGH_64_MAXIMUM).
		Ensure()
}

// Product_High_Word is the upper machine word of a machine-word product.
type Product_High_Word uint

// Product_High_Word_Invariants bounds the high word of a product.
func Product_High_Word_Invariants(value Product_High_Word, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint(uint(value), WORD_MINIMUM, PRODUCT_HIGH_WORD_MAXIMUM).
		Ensure()
}

// Product_Low_32 is the lower 32 bits of a 32-bit product.
type Product_Low_32 uint32

// Product_Low_32_Invariants states the complete 32-bit word domain.
func Product_Low_32_Invariants(value Product_Low_32, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), WORD_32_MINIMUM, WORD_32_MAXIMUM).
		Ensure()
}

// Product_Low_64 is the lower 64 bits of a 64-bit product.
type Product_Low_64 uint64

// Product_Low_64_Invariants states the complete 64-bit word domain.
func Product_Low_64_Invariants(value Product_Low_64, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), WORD_64_MINIMUM, WORD_64_MAXIMUM).
		Ensure()
}

// Product_Low_Word is the lower machine word of a machine-word product.
type Product_Low_Word uint

// Product_Low_Word_Invariants states the complete machine-word domain.
func Product_Low_Word_Invariants(value Product_Low_Word, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint(uint(value), WORD_MINIMUM, WORD_MAXIMUM).
		Ensure()
}

// Dividend_High_32 is the upper 32 bits of a dividend whose quotient fits one word.
type Dividend_High_32 uint32

// Dividend_High_32_Invariants bounds the high word to the one-word quotient domain. The
// high word must stay below the divisor, thus it never reaches the largest word.
func Dividend_High_32_Invariants(value Dividend_High_32, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), WORD_32_MINIMUM, DIVIDEND_HIGH_32_MAXIMUM).
		Ensure()
}

// Dividend_High_64 is the upper 64 bits of a dividend whose quotient fits one word.
type Dividend_High_64 uint64

// Dividend_High_64_Invariants bounds the high word to the one-word quotient domain.
func Dividend_High_64_Invariants(value Dividend_High_64, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), WORD_64_MINIMUM, DIVIDEND_HIGH_64_MAXIMUM).
		Ensure()
}

// Dividend_High_Word is the upper machine word of a dividend whose quotient fits one word.
type Dividend_High_Word uint

// Dividend_High_Word_Invariants bounds the high word to the one-word quotient domain.
func Dividend_High_Word_Invariants(
	value Dividend_High_Word, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint(uint(value), WORD_MINIMUM, DIVIDEND_HIGH_WORD_MAXIMUM).
		Ensure()
}

// Dividend_Low_32 is the lower 32 bits of a dividend.
type Dividend_Low_32 uint32

// Dividend_Low_32_Invariants states the complete 32-bit word domain.
func Dividend_Low_32_Invariants(value Dividend_Low_32, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), WORD_32_MINIMUM, WORD_32_MAXIMUM).
		Ensure()
}

// Dividend_Low_64 is the lower 64 bits of a dividend.
type Dividend_Low_64 uint64

// Dividend_Low_64_Invariants states the complete 64-bit word domain.
func Dividend_Low_64_Invariants(value Dividend_Low_64, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), WORD_64_MINIMUM, WORD_64_MAXIMUM).
		Ensure()
}

// Dividend_Low_Word is the lower machine word of a dividend.
type Dividend_Low_Word uint

// Dividend_Low_Word_Invariants states the complete machine-word domain.
func Dividend_Low_Word_Invariants(value Dividend_Low_Word, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint(uint(value), WORD_MINIMUM, WORD_MAXIMUM).
		Ensure()
}

// High_Word_32 is the upper 32 bits of a dividend of any size.
type High_Word_32 uint32

// High_Word_32_Invariants states the complete 32-bit word domain. A remainder admits any
// high word, thus this domain is wider than the one-word quotient domain.
func High_Word_32_Invariants(value High_Word_32, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), WORD_32_MINIMUM, WORD_32_MAXIMUM).
		Ensure()
}

// High_Word_64 is the upper 64 bits of a dividend of any size.
type High_Word_64 uint64

// High_Word_64_Invariants states the complete 64-bit word domain.
func High_Word_64_Invariants(value High_Word_64, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), WORD_64_MINIMUM, WORD_64_MAXIMUM).
		Ensure()
}

// High_Word is the upper machine word of a dividend of any size.
type High_Word uint

// High_Word_Invariants states the complete machine-word domain.
func High_Word_Invariants(value High_Word, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint(uint(value), WORD_MINIMUM, WORD_MAXIMUM).
		Ensure()
}

// Low_Word_32 is the lower 32 bits of a dividend of any size.
type Low_Word_32 uint32

// Low_Word_32_Invariants states the complete 32-bit word domain.
func Low_Word_32_Invariants(value Low_Word_32, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), WORD_32_MINIMUM, WORD_32_MAXIMUM).
		Ensure()
}

// Low_Word_64 is the lower 64 bits of a dividend of any size.
type Low_Word_64 uint64

// Low_Word_64_Invariants states the complete 64-bit word domain.
func Low_Word_64_Invariants(value Low_Word_64, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), WORD_64_MINIMUM, WORD_64_MAXIMUM).
		Ensure()
}

// Low_Word is the lower machine word of a dividend of any size.
type Low_Word uint

// Low_Word_Invariants states the complete machine-word domain.
func Low_Word_Invariants(value Low_Word, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint(uint(value), WORD_MINIMUM, WORD_MAXIMUM).
		Ensure()
}

// Divisor_32 is the 32-bit value a division divides by.
type Divisor_32 uint32

// Divisor_32_Invariants keeps a divisor away from zero. A zero divisor has no quotient,
// thus the domain starts at one.
func Divisor_32_Invariants(value Divisor_32, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), DIVISOR_MINIMUM, WORD_32_MAXIMUM).
		Ensure()
}

// Divisor_64 is the 64-bit value a division divides by.
type Divisor_64 uint64

// Divisor_64_Invariants keeps a divisor away from zero.
func Divisor_64_Invariants(value Divisor_64, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), DIVISOR_MINIMUM, WORD_64_MAXIMUM).
		Ensure()
}

// Divisor_Word is the machine-word value a division divides by.
type Divisor_Word uint

// Divisor_Word_Invariants keeps a divisor away from zero.
func Divisor_Word_Invariants(value Divisor_Word, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint(uint(value), DIVISOR_MINIMUM, WORD_MAXIMUM).
		Ensure()
}

// Division_Remainder_32 is the 32-bit remainder of a division.
type Division_Remainder_32 uint32

// Division_Remainder_32_Invariants bounds a remainder. A remainder stays below its
// divisor, thus it never reaches the largest word.
func Division_Remainder_32_Invariants(
	value Division_Remainder_32, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), WORD_32_MINIMUM, DIVISION_REMAINDER_32_MAXIMUM).
		Ensure()
}

// Division_Remainder_64 is the 64-bit remainder of a division.
type Division_Remainder_64 uint64

// Division_Remainder_64_Invariants bounds a remainder below the largest word.
func Division_Remainder_64_Invariants(
	value Division_Remainder_64, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), WORD_64_MINIMUM, DIVISION_REMAINDER_64_MAXIMUM).
		Ensure()
}

// Division_Remainder_Word is the machine-word remainder of a division.
type Division_Remainder_Word uint

// Division_Remainder_Word_Invariants bounds a remainder below the largest word.
func Division_Remainder_Word_Invariants(
	value Division_Remainder_Word, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint(uint(value), WORD_MINIMUM, DIVISION_REMAINDER_WORD_MAXIMUM).
		Ensure()
}

// Bit_Size_64 returns the smallest number of bits that holds the operand, which is one
// above the position of the highest set bit. A binary search over the halves finds the
// position in seven steps rather than the sixty-four a shift loop needs.
func Bit_Size_64(value Word_64) (count Bit_Count_64) {
	defer func() { Bit_Count_64_Invariants(count, "bit_size_64.count") }()
	Word_64_Invariants(value, "bit_size_64.value")
	residue := uint64(value)
	count = 0
	if residue >= 1<<32 {
		residue >>= 32
		count += 32
	}
	if residue >= 1<<16 {
		residue >>= 16
		count += 16
	}
	if residue >= 1<<8 {
		residue >>= 8
		count += 8
	}
	if residue >= 1<<4 {
		residue >>= 4
		count += 4
	}
	if residue >= 1<<2 {
		residue >>= 2
		count += 2
	}
	if residue >= 1<<1 {
		residue >>= 1
		count += 1
	}
	if residue >= 1 {
		count += 1
	}
	return count
}

// Bit_Size_8 returns the smallest number of bits that holds an 8-bit operand.
func Bit_Size_8(value Word_8) (count Bit_Count_8) {
	defer func() { Bit_Count_8_Invariants(count, "bit_size_8.count") }()
	Word_8_Invariants(value, "bit_size_8.value")
	return Bit_Count_8(Bit_Size_64(Word_64(value)))
}

// Bit_Size_16 returns the smallest number of bits that holds a 16-bit operand.
func Bit_Size_16(value Word_16) (count Bit_Count_16) {
	defer func() { Bit_Count_16_Invariants(count, "bit_size_16.count") }()
	Word_16_Invariants(value, "bit_size_16.value")
	return Bit_Count_16(Bit_Size_64(Word_64(value)))
}

// Bit_Size_32 returns the smallest number of bits that holds a 32-bit operand.
func Bit_Size_32(value Word_32) (count Bit_Count_32) {
	defer func() { Bit_Count_32_Invariants(count, "bit_size_32.count") }()
	Word_32_Invariants(value, "bit_size_32.value")
	return Bit_Count_32(Bit_Size_64(Word_64(value)))
}

// Bit_Size returns the smallest number of bits that holds a machine word.
func Bit_Size(value Word) (count Bit_Count_Word) {
	defer func() { Bit_Count_Word_Invariants(count, "bit_size.count") }()
	Word_Invariants(value, "bit_size.value")
	return Bit_Count_Word(Bit_Size_64(Word_64(value)))
}

// Leading_Zeros_8 counts the zero bits above the highest set bit of an 8-bit operand.
func Leading_Zeros_8(value Word_8) (count Bit_Count_8) {
	defer func() { Bit_Count_8_Invariants(count, "leading_zeros_8.count") }()
	Word_8_Invariants(value, "leading_zeros_8.value")
	return BIT_COUNT_8_MAXIMUM - Bit_Size_8(value)
}

// Leading_Zeros_16 counts the zero bits above the highest set bit of a 16-bit operand.
func Leading_Zeros_16(value Word_16) (count Bit_Count_16) {
	defer func() { Bit_Count_16_Invariants(count, "leading_zeros_16.count") }()
	Word_16_Invariants(value, "leading_zeros_16.value")
	return BIT_COUNT_16_MAXIMUM - Bit_Size_16(value)
}

// Leading_Zeros_32 counts the zero bits above the highest set bit of a 32-bit operand.
func Leading_Zeros_32(value Word_32) (count Bit_Count_32) {
	defer func() { Bit_Count_32_Invariants(count, "leading_zeros_32.count") }()
	Word_32_Invariants(value, "leading_zeros_32.value")
	return BIT_COUNT_32_MAXIMUM - Bit_Size_32(value)
}

// Leading_Zeros_64 counts the zero bits above the highest set bit of a 64-bit operand.
func Leading_Zeros_64(value Word_64) (count Bit_Count_64) {
	defer func() { Bit_Count_64_Invariants(count, "leading_zeros_64.count") }()
	Word_64_Invariants(value, "leading_zeros_64.value")
	return BIT_COUNT_64_MAXIMUM - Bit_Size_64(value)
}

// Leading_Zeros counts the zero bits above the highest set bit of a machine word.
func Leading_Zeros(value Word) (count Bit_Count_Word) {
	defer func() { Bit_Count_Word_Invariants(count, "leading_zeros.count") }()
	Word_Invariants(value, "leading_zeros.value")
	return BIT_COUNT_WORD_MAXIMUM - Bit_Size(value)
}

// Trailing_Zeros_64 counts the zero bits below the lowest set bit of a 64-bit operand. The
// negation of a value shares only the lowest set bit with it, thus the size of that single
// bit gives the count.
func Trailing_Zeros_64(value Word_64) (count Bit_Count_64) {
	defer func() { Bit_Count_64_Invariants(count, "trailing_zeros_64.count") }()
	Word_64_Invariants(value, "trailing_zeros_64.value")
	if value == 0 {
		return BIT_COUNT_64_MAXIMUM
	}
	lowest := uint64(value) & (^uint64(value) + 1)
	return Bit_Size_64(Word_64(lowest)) - 1
}

// Trailing_Zeros_8 counts the zero bits below the lowest set bit of an 8-bit operand.
func Trailing_Zeros_8(value Word_8) (count Bit_Count_8) {
	defer func() { Bit_Count_8_Invariants(count, "trailing_zeros_8.count") }()
	Word_8_Invariants(value, "trailing_zeros_8.value")
	if value == 0 {
		return BIT_COUNT_8_MAXIMUM
	}
	return Bit_Count_8(Trailing_Zeros_64(Word_64(value)))
}

// Trailing_Zeros_16 counts the zero bits below the lowest set bit of a 16-bit operand.
func Trailing_Zeros_16(value Word_16) (count Bit_Count_16) {
	defer func() { Bit_Count_16_Invariants(count, "trailing_zeros_16.count") }()
	Word_16_Invariants(value, "trailing_zeros_16.value")
	if value == 0 {
		return BIT_COUNT_16_MAXIMUM
	}
	return Bit_Count_16(Trailing_Zeros_64(Word_64(value)))
}

// Trailing_Zeros_32 counts the zero bits below the lowest set bit of a 32-bit operand.
func Trailing_Zeros_32(value Word_32) (count Bit_Count_32) {
	defer func() { Bit_Count_32_Invariants(count, "trailing_zeros_32.count") }()
	Word_32_Invariants(value, "trailing_zeros_32.value")
	if value == 0 {
		return BIT_COUNT_32_MAXIMUM
	}
	return Bit_Count_32(Trailing_Zeros_64(Word_64(value)))
}

// Trailing_Zeros counts the zero bits below the lowest set bit of a machine word.
func Trailing_Zeros(value Word) (count Bit_Count_Word) {
	defer func() { Bit_Count_Word_Invariants(count, "trailing_zeros.count") }()
	Word_Invariants(value, "trailing_zeros.value")
	if value == 0 {
		return BIT_COUNT_WORD_MAXIMUM
	}
	return Bit_Count_Word(Trailing_Zeros_64(Word_64(value)))
}

// Ones_Count_64 counts the set bits of a 64-bit operand. Three masked additions fold the
// per-bit counts into per-byte counts, and one multiplication sums the bytes into the top
// byte, so no loop over the bits is necessary.
func Ones_Count_64(value Word_64) (count Bit_Count_64) {
	defer func() { Bit_Count_64_Invariants(count, "ones_count_64.count") }()
	Word_64_Invariants(value, "ones_count_64.value")
	residue := uint64(value)
	residue = residue>>1&0x5555555555555555 + residue&0x5555555555555555
	residue = residue>>2&0x3333333333333333 + residue&0x3333333333333333
	residue = (residue>>4 + residue) & 0x0f0f0f0f0f0f0f0f
	return Bit_Count_64(residue * 0x0101010101010101 >> 56)
}

// Ones_Count_8 counts the set bits of an 8-bit operand.
func Ones_Count_8(value Word_8) (count Bit_Count_8) {
	defer func() { Bit_Count_8_Invariants(count, "ones_count_8.count") }()
	Word_8_Invariants(value, "ones_count_8.value")
	return Bit_Count_8(Ones_Count_64(Word_64(value)))
}

// Ones_Count_16 counts the set bits of a 16-bit operand.
func Ones_Count_16(value Word_16) (count Bit_Count_16) {
	defer func() { Bit_Count_16_Invariants(count, "ones_count_16.count") }()
	Word_16_Invariants(value, "ones_count_16.value")
	return Bit_Count_16(Ones_Count_64(Word_64(value)))
}

// Ones_Count_32 counts the set bits of a 32-bit operand.
func Ones_Count_32(value Word_32) (count Bit_Count_32) {
	defer func() { Bit_Count_32_Invariants(count, "ones_count_32.count") }()
	Word_32_Invariants(value, "ones_count_32.value")
	return Bit_Count_32(Ones_Count_64(Word_64(value)))
}

// Ones_Count counts the set bits of a machine word.
func Ones_Count(value Word) (count Bit_Count_Word) {
	defer func() { Bit_Count_Word_Invariants(count, "ones_count.count") }()
	Word_Invariants(value, "ones_count.value")
	return Bit_Count_Word(Ones_Count_64(Word_64(value)))
}

// Rotate_Left_8 turns the bits of an 8-bit operand left and returns the bits that leave the
// top to the bottom. The mask reduces the distance to one turn, and it reduces a negative
// distance to the matching right turn.
func Rotate_Left_8(value Word_8, rotation Rotation) (rotated Word_8) {
	defer func() { Word_8_Invariants(rotated, "rotate_left_8.rotated") }()
	Word_8_Invariants(value, "rotate_left_8.value")
	Rotation_Invariants(rotation, "rotate_left_8.rotation")
	distance := uint(rotation) & (BIT_COUNT_8_MAXIMUM - 1)
	return value<<distance | value>>(BIT_COUNT_8_MAXIMUM-distance)
}

// Rotate_Left_16 turns the bits of a 16-bit operand left by a distance.
func Rotate_Left_16(value Word_16, rotation Rotation) (rotated Word_16) {
	defer func() { Word_16_Invariants(rotated, "rotate_left_16.rotated") }()
	Word_16_Invariants(value, "rotate_left_16.value")
	Rotation_Invariants(rotation, "rotate_left_16.rotation")
	distance := uint(rotation) & (BIT_COUNT_16_MAXIMUM - 1)
	return value<<distance | value>>(BIT_COUNT_16_MAXIMUM-distance)
}

// Rotate_Left_32 turns the bits of a 32-bit operand left by a distance.
func Rotate_Left_32(value Word_32, rotation Rotation) (rotated Word_32) {
	defer func() { Word_32_Invariants(rotated, "rotate_left_32.rotated") }()
	Word_32_Invariants(value, "rotate_left_32.value")
	Rotation_Invariants(rotation, "rotate_left_32.rotation")
	distance := uint(rotation) & (BIT_COUNT_32_MAXIMUM - 1)
	return value<<distance | value>>(BIT_COUNT_32_MAXIMUM-distance)
}

// Rotate_Left_64 turns the bits of a 64-bit operand left by a distance.
func Rotate_Left_64(value Word_64, rotation Rotation) (rotated Word_64) {
	defer func() { Word_64_Invariants(rotated, "rotate_left_64.rotated") }()
	Word_64_Invariants(value, "rotate_left_64.value")
	Rotation_Invariants(rotation, "rotate_left_64.rotation")
	distance := uint(rotation) & (BIT_COUNT_64_MAXIMUM - 1)
	return value<<distance | value>>(BIT_COUNT_64_MAXIMUM-distance)
}

// Rotate_Left turns the bits of a machine word left by a distance.
func Rotate_Left(value Word, rotation Rotation) (rotated Word) {
	defer func() { Word_Invariants(rotated, "rotate_left.rotated") }()
	Word_Invariants(value, "rotate_left.value")
	Rotation_Invariants(rotation, "rotate_left.rotation")
	distance := uint(rotation) & (WORD_SIZE - 1)
	return value<<distance | value>>(WORD_SIZE-distance)
}

// Reverse_8 exchanges the bit at each position of an 8-bit operand with the bit at the
// mirror position. Each masked step exchanges the pairs one distance wider, thus three
// steps reverse the whole operand.
func Reverse_8(value Word_8) (reversed Word_8) {
	defer func() { Word_8_Invariants(reversed, "reverse_8.reversed") }()
	Word_8_Invariants(value, "reverse_8.value")
	residue := uint8(value)
	residue = residue>>1&0x55 | residue&0x55<<1
	residue = residue>>2&0x33 | residue&0x33<<2
	residue = residue>>4 | residue<<4
	return Word_8(residue)
}

// Reverse_16 exchanges the bit at each position of a 16-bit operand with the bit at the
// mirror position.
func Reverse_16(value Word_16) (reversed Word_16) {
	defer func() { Word_16_Invariants(reversed, "reverse_16.reversed") }()
	Word_16_Invariants(value, "reverse_16.value")
	residue := uint16(value)
	residue = residue>>1&0x5555 | residue&0x5555<<1
	residue = residue>>2&0x3333 | residue&0x3333<<2
	residue = residue>>4&0x0f0f | residue&0x0f0f<<4
	return Word_16(residue>>8 | residue<<8)
}

// Reverse_32 exchanges the bit at each position of a 32-bit operand with the bit at the
// mirror position.
func Reverse_32(value Word_32) (reversed Word_32) {
	defer func() { Word_32_Invariants(reversed, "reverse_32.reversed") }()
	Word_32_Invariants(value, "reverse_32.value")
	residue := uint32(value)
	residue = residue>>1&0x55555555 | residue&0x55555555<<1
	residue = residue>>2&0x33333333 | residue&0x33333333<<2
	residue = residue>>4&0x0f0f0f0f | residue&0x0f0f0f0f<<4
	return Reverse_Bytes_32(Word_32(residue))
}

// Reverse_64 exchanges the bit at each position of a 64-bit operand with the bit at the
// mirror position.
func Reverse_64(value Word_64) (reversed Word_64) {
	defer func() { Word_64_Invariants(reversed, "reverse_64.reversed") }()
	Word_64_Invariants(value, "reverse_64.value")
	residue := uint64(value)
	residue = residue>>1&0x5555555555555555 | residue&0x5555555555555555<<1
	residue = residue>>2&0x3333333333333333 | residue&0x3333333333333333<<2
	residue = residue>>4&0x0f0f0f0f0f0f0f0f | residue&0x0f0f0f0f0f0f0f0f<<4
	return Reverse_Bytes_64(Word_64(residue))
}

// Reverse exchanges the bit at each position of a machine word with the bit at the mirror
// position.
func Reverse(value Word) (reversed Word) {
	defer func() { Word_Invariants(reversed, "reverse.reversed") }()
	Word_Invariants(value, "reverse.value")
	return Word(Reverse_64(Word_64(value)))
}

// Reverse_Bytes_16 exchanges the two bytes of a 16-bit operand and leaves the bits inside
// each byte in place.
func Reverse_Bytes_16(value Word_16) (reversed Word_16) {
	defer func() { Word_16_Invariants(reversed, "reverse_bytes_16.reversed") }()
	Word_16_Invariants(value, "reverse_bytes_16.value")
	return value>>8 | value<<8
}

// Reverse_Bytes_32 exchanges the byte at each position of a 32-bit operand with the byte at
// the mirror position.
func Reverse_Bytes_32(value Word_32) (reversed Word_32) {
	defer func() { Word_32_Invariants(reversed, "reverse_bytes_32.reversed") }()
	Word_32_Invariants(value, "reverse_bytes_32.value")
	residue := uint32(value)
	residue = residue>>8&0x00ff00ff | residue&0x00ff00ff<<8
	return Word_32(residue>>16 | residue<<16)
}

// Reverse_Bytes_64 exchanges the byte at each position of a 64-bit operand with the byte at
// the mirror position.
func Reverse_Bytes_64(value Word_64) (reversed Word_64) {
	defer func() { Word_64_Invariants(reversed, "reverse_bytes_64.reversed") }()
	Word_64_Invariants(value, "reverse_bytes_64.value")
	residue := uint64(value)
	residue = residue>>8&0x00ff00ff00ff00ff | residue&0x00ff00ff00ff00ff<<8
	residue = residue>>16&0x0000ffff0000ffff | residue&0x0000ffff0000ffff<<16
	return Word_64(residue>>32 | residue<<32)
}

// Reverse_Bytes exchanges the byte at each position of a machine word with the byte at the
// mirror position.
func Reverse_Bytes(value Word) (reversed Word) {
	defer func() { Word_Invariants(reversed, "reverse_bytes.reversed") }()
	Word_Invariants(value, "reverse_bytes.value")
	return Word(Reverse_Bytes_64(Word_64(value)))
}

// Add_64 adds two 64-bit operands and an entering carry. The sum wraps, thus the carry
// comes from the sign bits: a carry leaves when both operands set the top bit, or when one
// of them sets it and the sum does not.
func Add_64(
	augend Word_64, addend Addend_64, carry Carry_In,
) (sum Word_64, carry_output Carry_Output) {
	defer func() {
		Word_64_Invariants(sum, "add_64.sum")
		Carry_Output_Invariants(carry_output, "add_64.carry_output")
	}()
	Word_64_Invariants(augend, "add_64.augend")
	Addend_64_Invariants(addend, "add_64.addend")
	Carry_In_Invariants(carry, "add_64.carry")
	left := uint64(augend)
	right := uint64(addend)
	total := left + right + uint64(carry)
	carry_bit := ((left & right) | ((left | right) &^ total)) >> 63
	return Word_64(total), Carry_Output(carry_bit)
}

// Add_32 adds two 32-bit operands and an entering carry.
func Add_32(
	augend Word_32, addend Addend_32, carry Carry_In,
) (sum Word_32, carry_output Carry_Output) {
	defer func() {
		Word_32_Invariants(sum, "add_32.sum")
		Carry_Output_Invariants(carry_output, "add_32.carry_output")
	}()
	Word_32_Invariants(augend, "add_32.augend")
	Addend_32_Invariants(addend, "add_32.addend")
	Carry_In_Invariants(carry, "add_32.carry")
	total := uint64(augend) + uint64(addend) + uint64(carry)
	return Word_32(total), Carry_Output(total >> 32)
}

// Add_Word adds two machine-word operands and an entering carry.
func Add_Word(
	augend Word, addend Addend_Word, carry Carry_In,
) (sum Word, carry_output Carry_Output) {
	defer func() {
		Word_Invariants(sum, "add_word.sum")
		Carry_Output_Invariants(carry_output, "add_word.carry_output")
	}()
	Word_Invariants(augend, "add_word.augend")
	Addend_Word_Invariants(addend, "add_word.addend")
	Carry_In_Invariants(carry, "add_word.carry")
	wide, wide_carry := Add_64(Word_64(augend), Addend_64(addend), carry)
	return Word(wide), wide_carry
}

// Subtract_64 removes a 64-bit subtrahend and an entering borrow from a minuend. The
// difference wraps, thus the borrow comes from the sign bits.
func Subtract_64(
	minuend Word_64, subtrahend Subtrahend_64, borrow Borrow_In,
) (difference Word_64, borrow_output Borrow_Output) {
	defer func() {
		Word_64_Invariants(difference, "subtract_64.difference")
		Borrow_Output_Invariants(borrow_output, "subtract_64.borrow_output")
	}()
	Word_64_Invariants(minuend, "subtract_64.minuend")
	Subtrahend_64_Invariants(subtrahend, "subtract_64.subtrahend")
	Borrow_In_Invariants(borrow, "subtract_64.borrow")
	left := uint64(minuend)
	right := uint64(subtrahend)
	total := left - right - uint64(borrow)
	borrow_bit := ((^left & right) | (^(left ^ right) & total)) >> 63
	return Word_64(total), Borrow_Output(borrow_bit)
}

// Subtract_32 removes a 32-bit subtrahend and an entering borrow from a minuend.
func Subtract_32(
	minuend Word_32, subtrahend Subtrahend_32, borrow Borrow_In,
) (difference Word_32, borrow_output Borrow_Output) {
	defer func() {
		Word_32_Invariants(difference, "subtract_32.difference")
		Borrow_Output_Invariants(borrow_output, "subtract_32.borrow_output")
	}()
	Word_32_Invariants(minuend, "subtract_32.minuend")
	Subtrahend_32_Invariants(subtrahend, "subtract_32.subtrahend")
	Borrow_In_Invariants(borrow, "subtract_32.borrow")
	left := uint32(minuend)
	right := uint32(subtrahend)
	total := left - right - uint32(borrow)
	borrow_bit := ((^left & right) | (^(left ^ right) & total)) >> 31
	return Word_32(total), Borrow_Output(borrow_bit)
}

// Subtract_Word removes a machine-word subtrahend and an entering borrow from a minuend.
func Subtract_Word(
	minuend Word, subtrahend Subtrahend_Word, borrow Borrow_In,
) (difference Word, borrow_output Borrow_Output) {
	defer func() {
		Word_Invariants(difference, "subtract_word.difference")
		Borrow_Output_Invariants(borrow_output, "subtract_word.borrow_output")
	}()
	Word_Invariants(minuend, "subtract_word.minuend")
	Subtrahend_Word_Invariants(subtrahend, "subtract_word.subtrahend")
	Borrow_In_Invariants(borrow, "subtract_word.borrow")
	wide, wide_borrow := Subtract_64(Word_64(minuend), Subtrahend_64(subtrahend), borrow)
	return Word(wide), wide_borrow
}

// Multiply_64 multiplies two 64-bit operands into a 128-bit product. Each operand splits
// into two 32-bit halves, thus four products of half width hold the result without an
// overflow, and the carries between them assemble the two words.
func Multiply_64(
	multiplicand Word_64, multiplier Multiplier_64,
) (high Product_High_64, low Product_Low_64) {
	defer func() {
		Product_High_64_Invariants(high, "multiply_64.high")
		Product_Low_64_Invariants(low, "multiply_64.low")
	}()
	Word_64_Invariants(multiplicand, "multiply_64.multiplicand")
	Multiplier_64_Invariants(multiplier, "multiply_64.multiplier")
	left := uint64(multiplicand)
	right := uint64(multiplier)
	left_low := left & 0xffffffff
	left_high := left >> 32
	right_low := right & 0xffffffff
	right_high := right >> 32
	partial := left_low * right_low
	middle_first := left_high*right_low + partial>>32
	middle_second := left_low*right_high + middle_first&0xffffffff
	upper := left_high*right_high + middle_first>>32 + middle_second>>32
	return Product_High_64(upper), Product_Low_64(left * right)
}

// Multiply_32 multiplies two 32-bit operands into a 64-bit product.
func Multiply_32(
	multiplicand Word_32, multiplier Multiplier_32,
) (high Product_High_32, low Product_Low_32) {
	defer func() {
		Product_High_32_Invariants(high, "multiply_32.high")
		Product_Low_32_Invariants(low, "multiply_32.low")
	}()
	Word_32_Invariants(multiplicand, "multiply_32.multiplicand")
	Multiplier_32_Invariants(multiplier, "multiply_32.multiplier")
	product := uint64(multiplicand) * uint64(multiplier)
	return Product_High_32(product >> 32), Product_Low_32(product)
}

// Multiply_Word multiplies two machine-word operands into a double-width product.
func Multiply_Word(
	multiplicand Word, multiplier Multiplier_Word,
) (high Product_High_Word, low Product_Low_Word) {
	defer func() {
		Product_High_Word_Invariants(high, "multiply_word.high")
		Product_Low_Word_Invariants(low, "multiply_word.low")
	}()
	Word_Invariants(multiplicand, "multiply_word.multiplicand")
	Multiplier_Word_Invariants(multiplier, "multiply_word.multiplier")
	wide_high, wide_low := Multiply_64(Word_64(multiplicand), Multiplier_64(multiplier))
	return Product_High_Word(wide_high), Product_Low_Word(wide_low)
}

// Divide_64 divides a 128-bit dividend by a 64-bit divisor. A high word that is not below
// the divisor gives a quotient wider than one word, thus that input panics.
func Divide_64(
	high Dividend_High_64, low Dividend_Low_64, divisor Divisor_64,
) (quotient Word_64, remainder Division_Remainder_64) {
	defer func() {
		Word_64_Invariants(quotient, "divide_64.quotient")
		Division_Remainder_64_Invariants(remainder, "divide_64.remainder")
	}()
	Dividend_High_64_Invariants(high, "divide_64.high")
	Dividend_Low_64_Invariants(low, "divide_64.low")
	Divisor_64_Invariants(divisor, "divide_64.divisor")
	aver.Always(
		uint64(high) < uint64(divisor),
		"A quotient of a 128-bit dividend must fit one word.",
	)
	return divide_words(high, low, divisor)
}

// Knuth's algorithm D at two 32-bit places. It normalizes the divisor so its top bit is
// set, which bounds the trial quotient error at two, then corrects each of the two digits
// at most twice. The correction runs inline because a trial digit is a value of the loop
// and not a boundary any caller can reach.
func divide_words(
	high Dividend_High_64, low Dividend_Low_64, divisor Divisor_64,
) (quotient Word_64, rest Division_Remainder_64) {
	defer func() {
		Word_64_Invariants(quotient, "divide_words.quotient")
		Division_Remainder_64_Invariants(rest, "divide_words.rest")
	}()
	Dividend_High_64_Invariants(high, "divide_words.high")
	Dividend_Low_64_Invariants(low, "divide_words.low")
	Divisor_64_Invariants(divisor, "divide_words.divisor")
	if high == 0 {
		whole := uint64(low) / uint64(divisor)
		return Word_64(whole), Division_Remainder_64(uint64(low) % uint64(divisor))
	}
	shift := uint(Leading_Zeros_64(Word_64(divisor)))
	scaled := uint64(divisor) << shift
	divisor_high := scaled >> 32
	divisor_low := scaled & 0xffffffff
	top := uint64(high)<<shift | uint64(low)>>(64-shift)
	rest_places := uint64(low) << shift
	middle := rest_places >> 32
	bottom := rest_places & 0xffffffff
	first := top / divisor_high
	partial := top - first*divisor_high
	// The correction runs inline at each place. A trial digit is a value of the loop and
	// not a boundary a caller can reach, thus it carries no domain to state.
	for index := 0; index < 2; index++ {
		too_large := false
		if first >= 1<<32 {
			too_large = true
		}
		if first*divisor_low > (1<<32)*partial+middle {
			too_large = true
		}
		if !too_large {
			break
		}
		first--
		partial += divisor_high
		if partial >= 1<<32 {
			break
		}
	}
	joined := top*(1<<32) + middle - first*scaled
	second := joined / divisor_high
	partial = joined - second*divisor_high
	for index := 0; index < 2; index++ {
		too_large := false
		if second >= 1<<32 {
			too_large = true
		}
		if second*divisor_low > (1<<32)*partial+bottom {
			too_large = true
		}
		if !too_large {
			break
		}
		second--
		partial += divisor_high
		if partial >= 1<<32 {
			break
		}
	}
	last := joined*(1<<32) + bottom - second*scaled
	return Word_64(first*(1<<32) + second), Division_Remainder_64(last >> shift)
}

// Divide_32 divides a 64-bit dividend by a 32-bit divisor.
func Divide_32(
	high Dividend_High_32, low Dividend_Low_32, divisor Divisor_32,
) (quotient Word_32, remainder Division_Remainder_32) {
	defer func() {
		Word_32_Invariants(quotient, "divide_32.quotient")
		Division_Remainder_32_Invariants(remainder, "divide_32.remainder")
	}()
	Dividend_High_32_Invariants(high, "divide_32.high")
	Dividend_Low_32_Invariants(low, "divide_32.low")
	Divisor_32_Invariants(divisor, "divide_32.divisor")
	aver.Always(
		uint32(high) < uint32(divisor),
		"A quotient of a 64-bit dividend must fit one word.",
	)
	dividend := uint64(high)<<32 | uint64(low)
	whole := dividend / uint64(divisor)
	return Word_32(whole), Division_Remainder_32(dividend % uint64(divisor))
}

// Divide_Word divides a double-width dividend by a machine-word divisor.
func Divide_Word(
	high Dividend_High_Word, low Dividend_Low_Word, divisor Divisor_Word,
) (quotient Word, remainder Division_Remainder_Word) {
	defer func() {
		Word_Invariants(quotient, "divide_word.quotient")
		Division_Remainder_Word_Invariants(remainder, "divide_word.remainder")
	}()
	Dividend_High_Word_Invariants(high, "divide_word.high")
	Dividend_Low_Word_Invariants(low, "divide_word.low")
	Divisor_Word_Invariants(divisor, "divide_word.divisor")
	wide_quotient, wide_rest := Divide_64(
		Dividend_High_64(high), Dividend_Low_64(low), Divisor_64(divisor))
	return Word(wide_quotient), Division_Remainder_Word(wide_rest)
}

// Remainder_64 returns the remainder of a 128-bit dividend and a 64-bit divisor. The
// remainder of the high word alone is below the divisor, thus reducing the high word first
// admits any dividend that Divide_64 would reject.
func Remainder_64(
	high High_Word_64, low Low_Word_64, divisor Divisor_64,
) (remainder Division_Remainder_64) {
	defer func() {
		Division_Remainder_64_Invariants(remainder, "remainder_64.remainder")
	}()
	High_Word_64_Invariants(high, "remainder_64.high")
	Low_Word_64_Invariants(low, "remainder_64.low")
	Divisor_64_Invariants(divisor, "remainder_64.divisor")
	reduced := Dividend_High_64(uint64(high) % uint64(divisor))
	_, rest := divide_words(reduced, Dividend_Low_64(low), divisor)
	return rest
}

// Remainder_32 returns the remainder of a 64-bit dividend and a 32-bit divisor.
func Remainder_32(
	high High_Word_32, low Low_Word_32, divisor Divisor_32,
) (remainder Division_Remainder_32) {
	defer func() {
		Division_Remainder_32_Invariants(remainder, "remainder_32.remainder")
	}()
	High_Word_32_Invariants(high, "remainder_32.high")
	Low_Word_32_Invariants(low, "remainder_32.low")
	Divisor_32_Invariants(divisor, "remainder_32.divisor")
	dividend := uint64(high)<<32 | uint64(low)
	return Division_Remainder_32(dividend % uint64(divisor))
}

// Remainder_Word returns the remainder of a double-width dividend and a machine-word
// divisor.
func Remainder_Word(
	high High_Word, low Low_Word, divisor Divisor_Word,
) (remainder Division_Remainder_Word) {
	defer func() {
		Division_Remainder_Word_Invariants(remainder, "remainder_word.remainder")
	}()
	High_Word_Invariants(high, "remainder_word.high")
	Low_Word_Invariants(low, "remainder_word.low")
	Divisor_Word_Invariants(divisor, "remainder_word.divisor")
	wide := Remainder_64(High_Word_64(high), Low_Word_64(low), Divisor_64(divisor))
	return Division_Remainder_Word(wide)
}
