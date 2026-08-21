// Package strconv converts between text and the Go scalar types. It is a port of the
// standard library package of the same name. Five differences follow from house
// rules: every text and buffer domain has a declared size limit, an out-of-domain Base
// or Bit_Size panics instead of returning an error value, the printable and graphic
// facts come from unicode instead of a private table, and float and complex conversion
// is absent because deterministic tier admits no float type. Every writer fills caller
// storage and owns no heap memory.
package strconv

import (
	"errors"

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/math/fixedpoint"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/strings"
	"local/james-orcales/shared/unicode/ucd"
	"local/james-orcales/shared/unicode/utf8"
)

// MACHINE_INTEGER_BITS is the width in bits of an int or a uint on this platform.
const MACHINE_INTEGER_BITS = bits.WORD_SIZE

// DIGIT_SYMBOLS holds one symbol for each digit value, lowercase above nine.
const DIGIT_SYMBOLS = "0123456789abcdefghijklmnopqrstuvwxyz"

// HEXADECIMAL_SYMBOLS holds the symbols an escape sequence writes.
const HEXADECIMAL_SYMBOLS = "0123456789abcdef"

// CONTROL_ESCAPE_LETTERS maps each named ASCII control to its escape letter. A zero
// marks a control that needs a hexadecimal escape.
const CONTROL_ESCAPE_LETTERS = "\x00\x00\x00\x00\x00\x00\x00abtnvfr"

// UNICODE_REPLACEMENT stands in for text that is not valid UTF-8.
const UNICODE_REPLACEMENT = Code_Point(utf8.REPLACEMENT_CHARACTER)

// TEXT_SIZE_MINIMUM is the size of empty text.
const TEXT_SIZE_MINIMUM = 0

// TEXT_SIZE_MAXIMUM caps text conversion reads.
const TEXT_SIZE_MAXIMUM = 4096

// PREFIX_TEXT_SIZE_HOLE is the one size that no literal has, because a literal holds
// two quote marks.
const PREFIX_TEXT_SIZE_HOLE = 1

// NUMBER_TEXT_SIZE_MINIMUM is the size of the shortest number, one digit.
const NUMBER_TEXT_SIZE_MINIMUM = 1

// LITERAL_TEXT_SIZE_MINIMUM is the size of the shortest literal, two quote marks.
const LITERAL_TEXT_SIZE_MINIMUM = 2

// TAIL_TEXT_SIZE_MAXIMUM is the text that follows one character of a literal body.
const TAIL_TEXT_SIZE_MAXIMUM = TEXT_SIZE_MAXIMUM - 1

// BODY_TEXT_SIZE_MAXIMUM is the text that a two-character prefix leaves.
const BODY_TEXT_SIZE_MAXIMUM = TEXT_SIZE_MAXIMUM - LITERAL_TEXT_SIZE_MINIMUM

// UNQUOTED_TEXT_SIZE_MAXIMUM admits one replacement character for each invalid input
// byte in an interpreted literal.
const UNQUOTED_TEXT_SIZE_MAXIMUM = BODY_TEXT_SIZE_MAXIMUM * utf8.CHARACTER_SIZE_THREE

// ESCAPE_TAIL_SIZE_MAXIMUM is the text that a two-character escape leaves in a body.
const ESCAPE_TAIL_SIZE_MAXIMUM = TEXT_SIZE_MAXIMUM - BYTE_ESCAPE_SIZE

// QUOTED_TEXT_SIZE_MINIMUM is the size of an empty literal, two quote marks.
const QUOTED_TEXT_SIZE_MINIMUM = LITERAL_TEXT_SIZE_MINIMUM

// QUOTED_TEXT_SIZE_MAXIMUM is the size of a literal whose every input byte needs a
// four-character byte escape.
const QUOTED_TEXT_SIZE_MAXIMUM = TEXT_SIZE_MAXIMUM*BYTE_ESCAPE_SIZE +
	QUOTED_TEXT_SIZE_MINIMUM

// CHARACTER_TEXT_SIZE_MINIMUM is the size of a literal that holds one plain character.
const CHARACTER_TEXT_SIZE_MINIMUM = LITERAL_TEXT_SIZE_MINIMUM + ESCAPE_SIZE_MINIMUM

// CHARACTER_TEXT_SIZE_MAXIMUM is the size of a literal that holds one long escape.
const CHARACTER_TEXT_SIZE_MAXIMUM = 12

// ESCAPE_SIZE_MINIMUM is the size of one character that needs no escape.
const ESCAPE_SIZE_MINIMUM = 1

// ESCAPE_SEQUENCE_SIZE_MINIMUM is the size of a one-letter escape sequence.
const ESCAPE_SEQUENCE_SIZE_MINIMUM = 2

// BYTE_ESCAPE_SIZE is the size of the escape that names one byte: a backslash, the
// letter x, and two digits.
const BYTE_ESCAPE_SIZE = 4

// ASCII_BYTE_MAXIMUM is the largest byte of ASCII text.
const ASCII_BYTE_MAXIMUM uint8 = 127

// OCTAL_DIGIT_MINIMUM is the first octal digit symbol.
const OCTAL_DIGIT_MINIMUM uint8 = '0'

// OCTAL_DIGIT_MAXIMUM is the last octal digit symbol.
const OCTAL_DIGIT_MAXIMUM uint8 = '7'

// ESCAPE_SIZE_MAXIMUM is the size of the longest escape sequence.
const ESCAPE_SIZE_MAXIMUM = 10

// BOOLEAN_TEXT_SIZE_TRUE is the size of the word true.
const BOOLEAN_TEXT_SIZE_TRUE = 4

// BOOLEAN_TEXT_SIZE_FALSE is the size of the word false.
const BOOLEAN_TEXT_SIZE_FALSE = 5

// DIGIT_TEXT_SIZE_MINIMUM is the size of a one-digit number.
const DIGIT_TEXT_SIZE_MINIMUM = 1

// DIGIT_TEXT_SIZE_MAXIMUM is the digit count of the largest unsigned integer in base
// two, thus no conversion needs a longer scratch array.
const DIGIT_TEXT_SIZE_MAXIMUM = 64

// INTEGER_TEXT_SIZE_MINIMUM is the size of a one-digit number.
const INTEGER_TEXT_SIZE_MINIMUM = 1

// INTEGER_TEXT_SIZE_MAXIMUM adds the sign to the digits of the widest number.
const INTEGER_TEXT_SIZE_MAXIMUM = DIGIT_TEXT_SIZE_MAXIMUM + 1

// DECIMAL_TEXT_SIZE_MINIMUM is the size of a one-digit number.
const DECIMAL_TEXT_SIZE_MINIMUM = 1

// DECIMAL_TEXT_SIZE_MAXIMUM is the size of the smallest machine integer in base ten.
const DECIMAL_TEXT_SIZE_MAXIMUM = 20

// BUFFER_SIZE_MINIMUM is the size of an empty buffer.
const BUFFER_SIZE_MINIMUM = 0

// BUFFER_SIZE_MAXIMUM is caller storage for largest possible written form.
const BUFFER_SIZE_MAXIMUM = QUOTED_TEXT_SIZE_MAXIMUM

// FRACTION_TEXT_SIZE_MAXIMUM is how many fraction digits the reader keeps. A tenth
// digit cannot change a value that holds one part in 2^20, thus the reader drops it and
// the power of ten stays inside a signed 64-bit product.
const FRACTION_TEXT_SIZE_MAXIMUM = 9

// FRACTION_UNITS_MINIMUM is the fraction of a whole number.
const FRACTION_UNITS_MINIMUM uint64 = bits.WORD_64_MINIMUM

// FRACTION_UNITS_MAXIMUM is one whole, which a fraction of nines rounds up to. It names the scale
// fixedpoint declares, thus one edit there keeps this bound correct.
const FRACTION_UNITS_MAXIMUM uint64 = fixedpoint.SCALE

// FIXED_POINT_UNITS_POSITIVE_MAXIMUM is the largest fixed-point storage value.
const FIXED_POINT_UNITS_POSITIVE_MAXIMUM uint64 = uint64(bits.INTEGER_64_MAXIMUM)

// FIXED_POINT_UNITS_NEGATIVE_MAXIMUM is the magnitude of the smallest fixed-point
// storage value.
const FIXED_POINT_UNITS_NEGATIVE_MAXIMUM uint64 = SIGNED_MAGNITUDE_MAXIMUM

// FIXED_POINT_TEXT_SIZE_MINIMUM is the size of a one-digit number. It names the size
// fixedpoint declares, because fixedpoint writes the text this package appends.
const FIXED_POINT_TEXT_SIZE_MINIMUM = fixedpoint.DECIMAL_DIGITS_SIZE_MINIMUM

// FIXED_POINT_TEXT_SIZE_MAXIMUM is a sign, the widest whole part, a point, and six
// fraction digits.
const FIXED_POINT_TEXT_SIZE_MAXIMUM = fixedpoint.TEXT_SIZE_MAXIMUM

// BASE_MINIMUM is the smallest radix a conversion accepts.
const BASE_MINIMUM = 2

// BASE_MAXIMUM is the largest radix the digit symbols cover.
const BASE_MAXIMUM = 36

// DECIMAL_BASE keeps each base-ten conversion on one radix fact.
const DECIMAL_BASE = 10

// DECIMAL_PAIR_BASE is the value count in the decimal-pair table.
const DECIMAL_PAIR_BASE = DECIMAL_BASE * DECIMAL_BASE

// DECIMAL_PAIR_INDEX_SCALE is the character count of each decimal pair.
const DECIMAL_PAIR_INDEX_SCALE = 2

// DECIMAL_CUBE_BASE is the value count in one three-digit decimal group.
const DECIMAL_CUBE_BASE = DECIMAL_PAIR_BASE * DECIMAL_BASE

// DECIMAL_QUAD_BASE is the value count in one four-digit decimal group.
const DECIMAL_QUAD_BASE = DECIMAL_PAIR_BASE * DECIMAL_PAIR_BASE

// DECIMAL_QUAD_SIZE is the character count of one four-digit decimal group.
const DECIMAL_QUAD_SIZE = DECIMAL_PAIR_INDEX_SCALE * DECIMAL_PAIR_INDEX_SCALE

// DECIMAL_QUINTET_SIZE is the character count of one five-digit decimal group.
const DECIMAL_QUINTET_SIZE = DECIMAL_QUAD_SIZE + 1

// DECIMAL_QUINTET_HIGH_NIBBLES identifies the ASCII digit block in five byte lanes.
const DECIMAL_QUINTET_HIGH_NIBBLES uint64 = 0xf0f0f0f0f0

// DECIMAL_QUINTET_ZERO_HIGH_NIBBLES is the high-nibble value of five ASCII digits.
const DECIMAL_QUINTET_ZERO_HIGH_NIBBLES uint64 = 0x3030303030

// DECIMAL_QUINTET_LOW_NIBBLES extracts five decimal digit values from ASCII bytes.
const DECIMAL_QUINTET_LOW_NIBBLES uint64 = 0x0f0f0f0f0f

// DECIMAL_QUINTET_INVALID_BIAS carries each low nibble that is greater than nine.
const DECIMAL_QUINTET_INVALID_BIAS uint64 = 0x0606060606

// DECIMAL_QUINTET_INVALID_CARRIES selects each carry from a biased digit lane.
const DECIMAL_QUINTET_INVALID_CARRIES uint64 = 0x1010101010

// DECIMAL_PAIR_FOLD combines adjacent digit lanes into decimal pairs.
const DECIMAL_PAIR_FOLD uint64 = 2_561

// DECIMAL_PAIR_LANES retain four folded decimal pairs.
const DECIMAL_PAIR_LANES uint64 = 0x00ff00ff00ff00ff

// IMPLIED_BASE_MINIMUM is the value that takes the radix from the text prefix.
const IMPLIED_BASE_MINIMUM = 0

// IMPLIED_BASE_MAXIMUM is the largest radix the digit symbols cover.
const IMPLIED_BASE_MAXIMUM = BASE_MAXIMUM

// IMPLIED_BASE_HOLE is the one value between the bounds that names no radix.
const IMPLIED_BASE_HOLE = 1

// BIT_SIZE_MINIMUM is the value that means the machine integer width.
const BIT_SIZE_MINIMUM = bits.BIT_COUNT_MINIMUM

// BIT_SIZE_MAXIMUM is the widest integer the package converts.
const BIT_SIZE_MAXIMUM = bits.BIT_COUNT_64_MAXIMUM

// INTEGER_64_MINIMUM is the smallest signed 64-bit integer.
const INTEGER_64_MINIMUM int64 = bits.INTEGER_64_MINIMUM

// INTEGER_64_MAXIMUM is the largest signed 64-bit integer.
const INTEGER_64_MAXIMUM int64 = bits.INTEGER_64_MAXIMUM

// INTEGER_64_MINIMUM_DECIMAL_TEXT is the decimal form of the lower storage boundary.
const INTEGER_64_MINIMUM_DECIMAL_TEXT Text = "-9223372036854775808"

// INTEGER_64_MAXIMUM_DECIMAL_TEXT is the decimal form of the upper storage boundary.
const INTEGER_64_MAXIMUM_DECIMAL_TEXT Text = "9223372036854775807"

// UNSIGNED_64_MINIMUM is the smallest unsigned 64-bit integer.
const UNSIGNED_64_MINIMUM uint64 = bits.WORD_64_MINIMUM

// UNSIGNED_64_MAXIMUM is the largest unsigned 64-bit integer.
const UNSIGNED_64_MAXIMUM uint64 = bits.WORD_64_MAXIMUM

// SIGNED_MAGNITUDE_MAXIMUM is the magnitude of the smallest signed 64-bit integer.
const SIGNED_MAGNITUDE_MAXIMUM uint64 = uint64(bits.INTEGER_64_MAXIMUM) + 1

// MACHINE_INTEGER_MINIMUM is the smallest int.
const MACHINE_INTEGER_MINIMUM int = bits.INTEGER_MINIMUM

// MACHINE_INTEGER_MAXIMUM is the largest int.
const MACHINE_INTEGER_MAXIMUM int = bits.INTEGER_MAXIMUM

// CHARACTER_MINIMUM is the smallest rune, which no valid text holds.
const CHARACTER_MINIMUM int32 = bits.INTEGER_32_MINIMUM

// CHARACTER_MAXIMUM is the largest rune, which no valid text holds.
const CHARACTER_MAXIMUM int32 = bits.INTEGER_32_MAXIMUM

// LATIN_1_MAXIMUM is the last character of the Latin-1 block, which is the last
// character that the printable question answers without a table.
const LATIN_1_MAXIMUM Character = 0xff

// ASCII_PRINT_MINIMUM is the space, the first printable ASCII character.
const ASCII_PRINT_MINIMUM Character = 0x20

// ASCII_PRINT_MAXIMUM is the tilde, the last printable ASCII character.
const ASCII_PRINT_MAXIMUM Character = 0x7e

// LATIN_1_PRINT_MINIMUM is the inverted exclamation mark, the first printable character
// above the ASCII block.
const LATIN_1_PRINT_MINIMUM Character = 0xa1

// SOFT_HYPHEN is the one Latin-1 character in the printable runs that a reader does not
// see, thus the printable set excludes it.
const SOFT_HYPHEN Character = 0xad

// NO_BREAK_SPACE is the one Latin-1 character that is graphic and is not printable.
const NO_BREAK_SPACE Character = 0xa0

// CODE_POINT_MINIMUM is the first code point.
const CODE_POINT_MINIMUM int32 = 0

// CODE_POINT_MAXIMUM is the last code point.
const CODE_POINT_MAXIMUM int32 = 0x10ffff

// BYTE_POINT_MAXIMUM is the last code point that one byte names.
const BYTE_POINT_MAXIMUM int32 = 255

// TEXT_BYTE_MINIMUM is the smallest byte of text.
const TEXT_BYTE_MINIMUM uint8 = bits.WORD_8_MINIMUM

// TEXT_BYTE_MAXIMUM is the largest byte of text.
const TEXT_BYTE_MAXIMUM uint8 = bits.WORD_8_MAXIMUM

// INVALID_TEXT_BYTE_MINIMUM is first byte that can fail UTF-8 decoding alone.
const INVALID_TEXT_BYTE_MINIMUM uint8 = uint8(utf8.CHARACTER_SELF)

// INVALID_TEXT_BYTE_MAXIMUM is largest byte of text.
const INVALID_TEXT_BYTE_MAXIMUM uint8 = bits.WORD_8_MAXIMUM

// FOLDED_BYTE_MINIMUM is the smallest byte that a lowercase fold can give, because the
// fold sets one bit that every smaller byte lacks.
const FOLDED_BYTE_MINIMUM uint8 = 32

// FOLDED_BYTE_MAXIMUM is the largest byte that a lowercase fold can give.
const FOLDED_BYTE_MAXIMUM uint8 = bits.WORD_8_MAXIMUM

// DIGIT_VALUE_MINIMUM is the value of the digit zero.
const DIGIT_VALUE_MINIMUM uint8 = 0

// DIGIT_VALUE_MAXIMUM is the value of the last digit symbol.
const DIGIT_VALUE_MAXIMUM uint8 = 35

// HEXADECIMAL_VALUE_MINIMUM is the value of the digit zero.
const HEXADECIMAL_VALUE_MINIMUM uint8 = 0

// HEXADECIMAL_VALUE_MAXIMUM is the value of the digit f.
const HEXADECIMAL_VALUE_MAXIMUM uint8 = 15

// QUOTE_MARK_NONE permits both quote characters unescaped.
const QUOTE_MARK_NONE = 0

// QUOTE_MARK_DOUBLE opens and closes a string literal.
const QUOTE_MARK_DOUBLE = '"'

// QUOTE_MARK_SINGLE opens and closes a character literal.
const QUOTE_MARK_SINGLE = '\''

// HEXADECIMAL_DIGIT_COUNT is the digit count of a byte escape.
const HEXADECIMAL_DIGIT_COUNT = 2

// SHORT_UNICODE_DIGIT_COUNT is the digit count of a two-byte code point escape.
const SHORT_UNICODE_DIGIT_COUNT = 4

// LONG_UNICODE_DIGIT_COUNT is the digit count of a four-byte code point escape.
const LONG_UNICODE_DIGIT_COUNT = 8

// SHORT_UNICODE_ESCAPE_SIZE is slash, escape letter, and four digits.
const SHORT_UNICODE_ESCAPE_SIZE = 6

// LONG_UNICODE_ESCAPE_SIZE is slash, escape letter, and eight digits.
const LONG_UNICODE_ESCAPE_SIZE = 10

// OCTAL_ESCAPE_DIGIT_COUNT is the digit count that follows the first octal digit.
const OCTAL_ESCAPE_DIGIT_COUNT = 2

// OCTAL_ESCAPE_MAXIMUM is the largest byte an octal escape can name.
const OCTAL_ESCAPE_MAXIMUM = 255

// Error_Syntax reports text that does not spell a value of the target type.
var Error_Syntax = errors.New("strconv: invalid syntax")

// Error_Range reports a value that the target width cannot hold.
var Error_Range = errors.New("strconv: value out of range")

// Prefix_Text is the literal at the start of a text, which is either absent or
// holds at least its two quote marks.
type Prefix_Text string

// Prefix_Text_Invariants bounds a literal prefix, excluding the one size that no
// literal has.
func Prefix_Text_Invariants(value Prefix_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int(
			len(value), TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM,
			PREFIX_TEXT_SIZE_HOLE, PREFIX_TEXT_SIZE_HOLE,
			PREFIX_TEXT_SIZE_HOLE, PREFIX_TEXT_SIZE_HOLE,
		).
		Ensure()
}

// Text is text that a conversion reads.
type Text string

// Text_Invariants bounds the text a conversion reads.
//
//go:nosplit
func Text_Invariants(value Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Number_Text is text that holds at least one character of a number.
type Number_Text string

// Number_Text_Invariants bounds the text that a number reader takes.
func Number_Text_Invariants(value Number_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), NUMBER_TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Literal_Text is text that holds at least the two quote marks of a literal.
type Literal_Text string

// Literal_Text_Invariants bounds the text that a literal reader takes.
func Literal_Text_Invariants(value Literal_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), LITERAL_TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Tail_Text is the text that follows one character of a literal body.
type Tail_Text string

// Tail_Text_Invariants bounds the text that one character leaves.
func Tail_Text_Invariants(value Tail_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, TAIL_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Body_Text is the text that the two quote marks of a literal leave.
type Body_Text string

// Body_Text_Invariants bounds the text that a two-character prefix leaves.
func Body_Text_Invariants(value Body_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, BODY_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Escape_Tail_Text is the text that an escape sequence leaves in a literal body.
type Escape_Tail_Text string

// Escape_Tail_Text_Invariants bounds the text that a two-character escape leaves.
func Escape_Tail_Text_Invariants(value Escape_Tail_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, ESCAPE_TAIL_SIZE_MAXIMUM).
		Ensure()
}

// Escape_Destination is exact caller storage for one body character.
type Escape_Destination []byte

// Escape_Destination_Invariants bounds plain UTF-8 and escaped widths.
func Escape_Destination_Invariants(
	value Escape_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), ESCAPE_SIZE_MINIMUM, ESCAPE_SIZE_MAXIMUM).
		Ensure()
}

// Escape_Sequence_Buffer is exact storage for one mandatory escape.
type Escape_Sequence_Buffer []byte

// Escape_Sequence_Buffer_Invariants states four mandatory escape widths.
func Escape_Sequence_Buffer_Invariants(
	value Escape_Sequence_Buffer, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_4_Int(
			len(value), ESCAPE_SEQUENCE_SIZE_MINIMUM, BYTE_ESCAPE_SIZE,
			SHORT_UNICODE_ESCAPE_SIZE, LONG_UNICODE_ESCAPE_SIZE,
		).
		Ensure()
}

// Unquoted_Buffer is exact caller storage for one decoded literal value.
type Unquoted_Buffer []byte

// Unquoted_Buffer_Invariants bounds decoded literal width.
func Unquoted_Buffer_Invariants(value Unquoted_Buffer, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, UNQUOTED_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Fraction_Text is the digits that follow the decimal point.
type Fraction_Text string

// Fraction_Text_Invariants bounds the fraction digits that the reader keeps.
func Fraction_Text_Invariants(value Fraction_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, FRACTION_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Fraction_Units is a fraction in fixed-point units.
type Fraction_Units uint64

// Fraction_Units_Invariants bounds a fraction to one whole, which the rounding of a
// fraction of nines reaches.
func Fraction_Units_Invariants(value Fraction_Units, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), FRACTION_UNITS_MINIMUM, FRACTION_UNITS_MAXIMUM).
		Ensure()
}

// Buffer is caller-owned storage for a written conversion.
type Buffer []byte

// Buffer_Invariants bounds caller-owned conversion storage.
func Buffer_Invariants(value Buffer, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), BUFFER_SIZE_MINIMUM, BUFFER_SIZE_MAXIMUM).
		Ensure()
}

// Boolean_Count is byte count of one written Boolean.
type Boolean_Count int

// Boolean_Count_Invariants states both Boolean text widths.
func Boolean_Count_Invariants(value Boolean_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(int(value), BOOLEAN_TEXT_SIZE_TRUE, BOOLEAN_TEXT_SIZE_FALSE).
		Ensure()
}

// Digit_Text_Count is byte count of one written unsigned integer.
type Digit_Text_Count int

// Digit_Text_Count_Invariants bounds unsigned integer width.
func Digit_Text_Count_Invariants(value Digit_Text_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), DIGIT_TEXT_SIZE_MINIMUM, DIGIT_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Integer_Text_Count is byte count of one written signed integer.
type Integer_Text_Count int

// Integer_Text_Count_Invariants bounds signed integer width.
func Integer_Text_Count_Invariants(value Integer_Text_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), INTEGER_TEXT_SIZE_MINIMUM, INTEGER_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Decimal_Text_Count is byte count of one written machine integer.
type Decimal_Text_Count int

// Decimal_Text_Count_Invariants bounds machine integer decimal width.
func Decimal_Text_Count_Invariants(value Decimal_Text_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), DECIMAL_TEXT_SIZE_MINIMUM, DECIMAL_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Fixed_Point_Text_Count is byte count of one written fixed-point number.
type Fixed_Point_Text_Count int

// Fixed_Point_Text_Count_Invariants bounds fixed-point text width.
func Fixed_Point_Text_Count_Invariants(
	value Fixed_Point_Text_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), FIXED_POINT_TEXT_SIZE_MINIMUM, FIXED_POINT_TEXT_SIZE_MAXIMUM,
		).
		Ensure()
}

// Quoted_Text_Count is byte count of one written string literal.
type Quoted_Text_Count int

// Quoted_Text_Count_Invariants bounds string literal width.
func Quoted_Text_Count_Invariants(value Quoted_Text_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), QUOTED_TEXT_SIZE_MINIMUM, QUOTED_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Character_Text_Count is byte count of one written character literal.
type Character_Text_Count int

// Character_Text_Count_Invariants bounds character literal width.
func Character_Text_Count_Invariants(
	value Character_Text_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), CHARACTER_TEXT_SIZE_MINIMUM, CHARACTER_TEXT_SIZE_MAXIMUM,
		).
		Ensure()
}

// Escape_Text_Count is byte count of one written literal body character.
type Escape_Text_Count int

// Escape_Text_Count_Invariants bounds plain UTF-8 and escaped widths.
func Escape_Text_Count_Invariants(value Escape_Text_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), ESCAPE_SIZE_MINIMUM, ESCAPE_SIZE_MAXIMUM).
		Ensure()
}

// Escape_Sequence_Count is byte count of one mandatory escape.
type Escape_Sequence_Count int

// Escape_Sequence_Count_Invariants states four mandatory escape widths.
func Escape_Sequence_Count_Invariants(
	value Escape_Sequence_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_4_Int(
			int(value), ESCAPE_SEQUENCE_SIZE_MINIMUM, BYTE_ESCAPE_SIZE,
			SHORT_UNICODE_ESCAPE_SIZE, LONG_UNICODE_ESCAPE_SIZE,
		).
		Ensure()
}

// Decoded_Text_Count is byte count of one decoded character.
type Decoded_Text_Count int

// Decoded_Text_Count_Invariants states UTF-8 sequence widths.
func Decoded_Text_Count_Invariants(value Decoded_Text_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Int(
			int(value), utf8.CHARACTER_SIZE_MINIMUM, utf8.CHARACTER_SIZE_TWO,
			utf8.CHARACTER_SIZE_THREE, utf8.CHARACTER_SIZE_MAXIMUM,
		).
		Ensure()
}

// Unquoted_Text_Count is byte count of one decoded literal value.
type Unquoted_Text_Count int

// Unquoted_Text_Count_Invariants bounds decoded literal width.
func Unquoted_Text_Count_Invariants(
	value Unquoted_Text_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), TEXT_SIZE_MINIMUM, UNQUOTED_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Boolean is one true or false value.
type Boolean bool

// Boolean_Invariants records both Boolean states.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A Boolean value is true.").
		Ensure()
}

// Base is the radix of a conversion.
type Base int

// Base_Invariants bounds a radix to the range the digit symbols cover.
func Base_Invariants(value Base, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), BASE_MINIMUM, BASE_MAXIMUM).
		Ensure()
}

// Implied_Base is a radix that zero defers to the prefix of the text.
type Implied_Base int

// Implied_Base_Invariants bounds a radix, admitting zero and excluding one.
func Implied_Base_Invariants(value Implied_Base, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int(
			int(value), IMPLIED_BASE_MINIMUM, IMPLIED_BASE_MAXIMUM,
			IMPLIED_BASE_HOLE, IMPLIED_BASE_HOLE, IMPLIED_BASE_HOLE, IMPLIED_BASE_HOLE,
		).
		Ensure()
}

// Bit_Size is the width in bits that a parsed value must fit, zero meaning the machine
// integer width.
type Bit_Size int

// Bit_Size_Invariants bounds a width to the integer types the package converts.
func Bit_Size_Invariants(value Bit_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), BIT_SIZE_MINIMUM, BIT_SIZE_MAXIMUM).
		Ensure()
}

// Signed_Integer is a signed 64-bit value.
type Signed_Integer int64

// Signed_Integer_Invariants states the complete signed 64-bit domain.
func Signed_Integer_Invariants(value Signed_Integer, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), INTEGER_64_MINIMUM, INTEGER_64_MAXIMUM).
		Ensure()
}

// Unsigned_Integer is an unsigned 64-bit value.
type Unsigned_Integer uint64

// Unsigned_Integer_Invariants states the complete unsigned 64-bit domain.
func Unsigned_Integer_Invariants(value Unsigned_Integer, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), UNSIGNED_64_MINIMUM, UNSIGNED_64_MAXIMUM).
		Ensure()
}

// Machine_Integer is a signed value of the platform integer width.
type Machine_Integer int

// Machine_Integer_Invariants states the complete platform integer domain.
//
//go:nosplit
func Machine_Integer_Invariants(value Machine_Integer, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), MACHINE_INTEGER_MINIMUM, MACHINE_INTEGER_MAXIMUM).
		Ensure()
}

// Character is one rune, which the caller of a quote conversion can give any value.
type Character rune

// Character_Invariants states the complete rune storage domain.
func Character_Invariants(value Character, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), CHARACTER_MINIMUM, CHARACTER_MAXIMUM).
		Ensure()
}

// Latin_1_Character is one character of the first two Unicode blocks, which the
// printable question answers without a table.
type Latin_1_Character rune

// Latin_1_Character_Invariants bounds a character to the Latin-1 block.
func Latin_1_Character_Invariants(value Latin_1_Character, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), CODE_POINT_MINIMUM, int32(LATIN_1_MAXIMUM)).
		Ensure()
}

// Code_Point is one Unicode code point.
type Code_Point rune

// Code_Point_Invariants bounds a value to the code point range.
func Code_Point_Invariants(value Code_Point, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), CODE_POINT_MINIMUM, CODE_POINT_MAXIMUM).
		Ensure()
}

// Byte_Point is a code point that one byte of an octal escape names.
type Byte_Point rune

// Byte_Point_Invariants bounds a value to the code points of one byte.
func Byte_Point_Invariants(value Byte_Point, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), CODE_POINT_MINIMUM, BYTE_POINT_MAXIMUM).
		Ensure()
}

// Text_Byte is one byte of text.
type Text_Byte uint8

// Text_Byte_Invariants states the complete byte domain.
func Text_Byte_Invariants(value Text_Byte, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), TEXT_BYTE_MINIMUM, TEXT_BYTE_MAXIMUM).
		Ensure()
}

// Invalid_Text_Byte is one non-ASCII byte that failed UTF-8 decoding.
type Invalid_Text_Byte uint8

// Invalid_Text_Byte_Invariants bounds failed leading byte.
func Invalid_Text_Byte_Invariants(value Invalid_Text_Byte, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), INVALID_TEXT_BYTE_MINIMUM, INVALID_TEXT_BYTE_MAXIMUM,
		).
		Ensure()
}

// Octal_Digit_Byte is one octal digit symbol.
type Octal_Digit_Byte uint8

// Octal_Digit_Byte_Invariants bounds a symbol to the octal digits.
func Octal_Digit_Byte_Invariants(value Octal_Digit_Byte, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), OCTAL_DIGIT_MINIMUM, OCTAL_DIGIT_MAXIMUM).
		Ensure()
}

// Folded_Byte is a byte that a lowercase fold gives.
type Folded_Byte uint8

// Folded_Byte_Invariants bounds a folded byte, which holds the bit the fold sets.
func Folded_Byte_Invariants(value Folded_Byte, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), FOLDED_BYTE_MINIMUM, FOLDED_BYTE_MAXIMUM).
		Ensure()
}

// Digit_Value is the value of one digit symbol.
type Digit_Value uint8

// Digit_Value_Invariants bounds a digit to the largest radix.
func Digit_Value_Invariants(value Digit_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), DIGIT_VALUE_MINIMUM, DIGIT_VALUE_MAXIMUM).
		Ensure()
}

// Hexadecimal_Value is the value of one hexadecimal digit.
type Hexadecimal_Value uint8

// Hexadecimal_Value_Invariants bounds a digit to base sixteen.
func Hexadecimal_Value_Invariants(value Hexadecimal_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), HEXADECIMAL_VALUE_MINIMUM, HEXADECIMAL_VALUE_MAXIMUM).
		Ensure()
}

// Quote_Mark is the quote character of a literal, zero meaning neither.
type Quote_Mark uint8

// Quote_Mark_Invariants states the three literal forms.
func Quote_Mark_Invariants(value Quote_Mark, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(uint8(value), QUOTE_MARK_NONE, QUOTE_MARK_DOUBLE, QUOTE_MARK_SINGLE).
		Ensure()
}

// Literal_Quote_Mark is the quote character that opens and closes a literal.
type Literal_Quote_Mark uint8

// Literal_Quote_Mark_Invariants states the two literal forms.
func Literal_Quote_Mark_Invariants(value Literal_Quote_Mark, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), QUOTE_MARK_DOUBLE, QUOTE_MARK_SINGLE).
		Ensure()
}

// Digit_Count is the digit count that one escape sequence reads or writes.
type Digit_Count int

// Digit_Count_Invariants states the three escape widths.
func Digit_Count_Invariants(value Digit_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Int(
			int(value), HEXADECIMAL_DIGIT_COUNT, SHORT_UNICODE_DIGIT_COUNT,
			LONG_UNICODE_DIGIT_COUNT,
		).
		Ensure()
}

// Parse_Boolean reads the accepted spellings of a Boolean.
func Parse_Boolean(text Text) (value Boolean, err error) {
	defer func() {
		Boolean_Invariants(value, "parse_boolean.value")
	}()
	Text_Invariants(text, "parse_boolean.text")
	switch text {
	case "1", "t", "T", "true", "TRUE", "True":
		return true, nil
	case "0", "f", "F", "false", "FALSE", "False":
		return false, nil
	}
	return false, Error_Syntax
}

// Format_Boolean_Into writes true or false into caller storage.
func Format_Boolean_Into(destination Buffer, value Boolean) (count Boolean_Count) {
	defer func() { Boolean_Count_Invariants(count, "format_boolean_into.count") }()
	Buffer_Invariants(destination, "format_boolean_into.destination")
	Boolean_Invariants(value, "format_boolean_into.value")
	word := "false"
	if value {
		word = "true"
	}
	count = Boolean_Count(len(word))
	aver.Always(
		int(count) <= len(destination),
		"Format_Boolean_Into destination holds complete result.",
	)
	copy(destination, word)
	return count
}

// Parse_Unsigned_Integer reads an unsigned value. A sign prefix is not permitted.
func Parse_Unsigned_Integer(
	text Text, base Implied_Base, bit_size Bit_Size,
) (value Unsigned_Integer, err error) {
	defer func() {
		Unsigned_Integer_Invariants(value, "parse_unsigned_integer.value")
	}()
	Text_Invariants(text, "parse_unsigned_integer.text")
	Implied_Base_Invariants(base, "parse_unsigned_integer.base")
	Bit_Size_Invariants(bit_size, "parse_unsigned_integer.bit_size")
	if text == "" {
		return 0, Error_Syntax
	}
	radix, digits := resolve_base(Number_Text(text), base)
	underscores := Boolean(base == 0)
	// The separator rule reads the whole text, thus it runs before the digit reader,
	// which skips an underscore without a rule of its own.
	if underscores {
		if strings.Index_Byte(strings.Text(text), '_') >= 0 {
			if !underscores_separate_digits(Number_Text(text)) {
				return 0, Error_Syntax
			}
		}
	}
	return accumulate_digits(digits, radix, bit_size, underscores)
}

// Parse_Integer reads a signed value, the sign preceding any base prefix.
func Parse_Integer(
	text Text, base Implied_Base, bit_size Bit_Size,
) (value Signed_Integer, err error) {
	defer func() {
		Signed_Integer_Invariants(value, "parse_integer.value")
	}()
	Text_Invariants(text, "parse_integer.text")
	Implied_Base_Invariants(base, "parse_integer.base")
	Bit_Size_Invariants(bit_size, "parse_integer.bit_size")
	if text == "" {
		return 0, Error_Syntax
	}
	if base == DECIMAL_BASE {
		if bit_size == BIT_SIZE_MAXIMUM {
			switch text {
			case Text(INTEGER_64_MINIMUM_DECIMAL_TEXT):
				return Signed_Integer(INTEGER_64_MINIMUM), nil
			case Text(INTEGER_64_MAXIMUM_DECIMAL_TEXT):
				return Signed_Integer(INTEGER_64_MAXIMUM), nil
			}
		}
	}
	negative := Boolean(text[0] == '-')
	digits := text
	switch text[0] {
	case '+', '-':
		digits = text[1:]
	}
	width := int(bit_size)
	if width == 0 {
		width = MACHINE_INTEGER_BITS
	}
	cutoff := uint64(1) << uint(width-1)
	if base == DECIMAL_BASE {
		return parse_decimal_digits(digits, bit_size, negative)
	}
	magnitude, parse_err := Parse_Unsigned_Integer(digits, base, bit_size)
	if parse_err != nil {
		if !errors.Is(parse_err, Error_Range) {
			return 0, parse_err
		}
	}
	if !negative {
		if uint64(magnitude) >= cutoff {
			return Signed_Integer(cutoff - 1), Error_Range
		}
		return Signed_Integer(magnitude), nil
	}
	if uint64(magnitude) > cutoff {
		return Signed_Integer(-int64(cutoff)), Error_Range
	}
	return Signed_Integer(-int64(magnitude)), nil
}

// Reads a signed decimal number without the unsigned reader. Base ten is the common
// case, thus it earns a reader that needs no base prefix and no separator rule.
func parse_decimal_digits(
	digits Text, bit_size Bit_Size, negative Boolean,
) (value Signed_Integer, err error) {
	defer func() {
		Signed_Integer_Invariants(value, "parse_decimal_digits.value")
	}()
	Text_Invariants(digits, "parse_decimal_digits.digits")
	Bit_Size_Invariants(bit_size, "parse_decimal_digits.bit_size")
	Boolean_Invariants(negative, "parse_decimal_digits.negative")
	if len(digits) < NUMBER_TEXT_SIZE_MINIMUM {
		return 0, Error_Syntax
	}
	width := int(bit_size)
	if width == 0 {
		width = MACHINE_INTEGER_BITS
	}
	cutoff := uint64(1) << uint(width-1)
	maximum, limit := cutoff, Signed_Integer(cutoff-1)
	if negative {
		limit = Signed_Integer(-int64(cutoff))
	} else {
		maximum--
	}
	magnitude := uint64(0)
	for byte_index := range len(digits) {
		digit := digits[byte_index] - '0'
		if digit > DECIMAL_BASE-1 {
			return 0, Error_Syntax
		}
		// Compare first because an overflow cannot select the saturation
		// value.
		if magnitude > maximum/DECIMAL_BASE {
			return limit, Error_Range
		}
		if magnitude == maximum/DECIMAL_BASE {
			if uint64(digit) > maximum%DECIMAL_BASE {
				return limit, Error_Range
			}
		}
		magnitude = magnitude*DECIMAL_BASE + uint64(digit)
	}
	if negative {
		return Signed_Integer(-int64(magnitude)), nil
	}
	return Signed_Integer(magnitude), nil
}

// Parse_Decimal reads a base ten value at the machine integer width.
//
//go:nosplit
func Parse_Decimal(text Text) (value Machine_Integer, err error) {
	defer func() {
		Machine_Integer_Invariants(value, "parse_decimal.value")
	}()
	Text_Invariants(text, "parse_decimal.text")
	digits := text
	negative := false
	if len(text) == DECIMAL_QUINTET_SIZE {
		if text[0] == '+' {
			return parse_decimal_general(text)
		}
		if text[0] == '-' {
			return parse_decimal_general(text)
		}
	} else if len(text) == DECIMAL_QUINTET_SIZE+1 {
		if text[0] != '+' {
			if text[0] != '-' {
				return parse_decimal_general(text)
			}
		}
		negative = text[0] == '-'
		digits = text[1:]
	} else {
		return parse_decimal_general(text)
	}
	packed := uint64(digits[0]) |
		uint64(digits[1])<<bits.BIT_COUNT_8_MAXIMUM |
		uint64(digits[2])<<(2*bits.BIT_COUNT_8_MAXIMUM) |
		uint64(digits[3])<<(3*bits.BIT_COUNT_8_MAXIMUM) |
		uint64(digits[4])<<(4*bits.BIT_COUNT_8_MAXIMUM)
	if packed&DECIMAL_QUINTET_HIGH_NIBBLES !=
		DECIMAL_QUINTET_ZERO_HIGH_NIBBLES {
		return 0, Error_Syntax
	}
	digit_values := packed & DECIMAL_QUINTET_LOW_NIBBLES
	if (digit_values+DECIMAL_QUINTET_INVALID_BIAS)&
		DECIMAL_QUINTET_INVALID_CARRIES != 0 {
		return 0, Error_Syntax
	}
	pairs := digit_values * DECIMAL_PAIR_FOLD >> bits.BIT_COUNT_8_MAXIMUM &
		DECIMAL_PAIR_LANES
	parsed := int(pairs&uint64(TEXT_BYTE_MAXIMUM))*DECIMAL_CUBE_BASE +
		int(pairs>>(2*bits.BIT_COUNT_8_MAXIMUM)&uint64(TEXT_BYTE_MAXIMUM))*
			DECIMAL_BASE +
		int(digit_values>>(4*bits.BIT_COUNT_8_MAXIMUM)&uint64(TEXT_BYTE_MAXIMUM))
	if negative {
		parsed = -parsed
	}
	return Machine_Integer(parsed), nil
}

// Reads every decimal width outside the five-digit fast interval.
func parse_decimal_general(text Text) (value Machine_Integer, err error) {
	defer func() {
		Machine_Integer_Invariants(value, "parse_decimal_general.value")
	}()
	Text_Invariants(text, "parse_decimal_general.text")
	if len(text) > TEXT_SIZE_MINIMUM {
		if len(text) < DECIMAL_TEXT_SIZE_MAXIMUM-1 {
			original := text
			switch text[0] {
			case '+', '-':
				text = text[1:]
				if len(text) < DECIMAL_TEXT_SIZE_MINIMUM {
					return 0, Error_Syntax
				}
			}
			parsed := 0
			byte_index := 0
			for ; byte_index < len(text)%DECIMAL_QUAD_SIZE; byte_index++ {
				digit := text[byte_index] - '0'
				if digit >= DECIMAL_BASE {
					return 0, Error_Syntax
				}
				parsed = parsed*DECIMAL_BASE + int(digit)
			}
			for ; byte_index < len(text); byte_index += DECIMAL_QUAD_SIZE {
				thousands := text[byte_index] - '0'
				hundreds := text[byte_index+1] - '0'
				tens := text[byte_index+2] - '0'
				units := text[byte_index+3] - '0'
				if thousands >= DECIMAL_BASE {
					return 0, Error_Syntax
				}
				if hundreds >= DECIMAL_BASE {
					return 0, Error_Syntax
				}
				if tens >= DECIMAL_BASE {
					return 0, Error_Syntax
				}
				if units >= DECIMAL_BASE {
					return 0, Error_Syntax
				}
				parsed = parsed*DECIMAL_QUAD_BASE +
					int(thousands)*DECIMAL_CUBE_BASE
				parsed += int(hundreds)*DECIMAL_PAIR_BASE +
					int(tens)*DECIMAL_BASE + int(units)
			}
			if original[0] == '-' {
				parsed = -parsed
			}
			return Machine_Integer(parsed), nil
		}
	}
	wide, parse_err := Parse_Integer(text, DECIMAL_BASE, 0)
	return Machine_Integer(wide), parse_err
}

// Format_Unsigned_Integer_Into writes unsigned value into caller storage.
func Format_Unsigned_Integer_Into(
	destination Buffer, value Unsigned_Integer, base Base,
) (count Digit_Text_Count) {
	defer func() {
		Digit_Text_Count_Invariants(count, "format_unsigned_integer_into.count")
	}()
	Buffer_Invariants(destination, "format_unsigned_integer_into.destination")
	Unsigned_Integer_Invariants(value, "format_unsigned_integer_into.value")
	Base_Invariants(base, "format_unsigned_integer_into.base")
	return Digit_Text_Count(integer_into(destination, value, false, base))
}

// Format_Integer_Into writes signed value into caller storage.
func Format_Integer_Into(
	destination Buffer, value Signed_Integer, base Base,
) (count Integer_Text_Count) {
	defer func() { Integer_Text_Count_Invariants(count, "format_integer_into.count") }()
	Buffer_Invariants(destination, "format_integer_into.destination")
	Signed_Integer_Invariants(value, "format_integer_into.value")
	Base_Invariants(base, "format_integer_into.base")
	magnitude := uint64(value)
	if value < 0 {
		magnitude = -magnitude
	}
	return integer_into(destination, Unsigned_Integer(magnitude), Boolean(value < 0), base)
}

// Format_Decimal_Into writes machine integer into caller storage.
func Format_Decimal_Into(
	destination Buffer, value Machine_Integer,
) (count Decimal_Text_Count) {
	defer func() { Decimal_Text_Count_Invariants(count, "format_decimal_into.count") }()
	Buffer_Invariants(destination, "format_decimal_into.destination")
	Machine_Integer_Invariants(value, "format_decimal_into.value")
	return Decimal_Text_Count(Format_Integer_Into(
		destination, Signed_Integer(value), DECIMAL_BASE,
	))
}

// Writes sign and magnitude after proving caller storage holds complete text.
func integer_into(
	destination Buffer, magnitude Unsigned_Integer, negative Boolean, base Base,
) (count Integer_Text_Count) {
	defer func() { Integer_Text_Count_Invariants(count, "integer_into.count") }()
	Buffer_Invariants(destination, "integer_into.destination")
	Unsigned_Integer_Invariants(magnitude, "integer_into.magnitude")
	Boolean_Invariants(negative, "integer_into.negative")
	Base_Invariants(base, "integer_into.base")
	radix := uint64(base)
	digit_count := int(digit_count_of(magnitude, base))
	text_count := digit_count
	if negative {
		text_count++
	}
	count = Integer_Text_Count(text_count)
	aver.Always(
		text_count <= len(destination),
		"Integer destination holds complete result.",
	)
	rest := uint64(magnitude)
	write_index := text_count
	for range digit_count {
		write_index--
		destination[write_index] = DIGIT_SYMBOLS[rest%radix]
		rest /= radix
	}
	if negative {
		destination[0] = '-'
	}
	return count
}

// Counts digits before writer changes caller storage.
func digit_count_of(
	magnitude Unsigned_Integer, base Base,
) (count Digit_Text_Count) {
	defer func() { Digit_Text_Count_Invariants(count, "digit_count_of.count") }()
	Unsigned_Integer_Invariants(magnitude, "digit_count_of.magnitude")
	Base_Invariants(base, "digit_count_of.base")
	count = DIGIT_TEXT_SIZE_MINIMUM
	for rest := uint64(magnitude); rest >= uint64(base); rest /= uint64(base) {
		count++
	}
	return count
}

// Parse_Fixed_Point reads decimal text into a fixed-point number, which is what the
// deterministic tier holds in place of a float. It rounds the fraction onto the
// fixed-point grid, half away from zero.
func Parse_Fixed_Point(text Text) (value fixedpoint.Number, err error) {
	defer func() {
		fixedpoint.Number_Invariants(value, "parse_fixed_point.value")
	}()
	Text_Invariants(text, "parse_fixed_point.text")
	body := text
	negative := Boolean(false)
	if len(body) > 0 {
		switch body[0] {
		case '+':
			body = body[1:]
		case '-':
			negative = true
			body = body[1:]
		}
	}
	whole_text := body
	fraction_text := Text("")
	point_offset := int(strings.Index_Byte(strings.Text(body), '.'))
	if point_offset >= 0 {
		whole_text = body[:point_offset]
		fraction_text = body[point_offset+1:]
	}
	if len(whole_text) == 0 {
		if len(fraction_text) == 0 {
			return 0, Error_Syntax
		}
	}
	whole := Unsigned_Integer(0)
	if len(whole_text) > 0 {
		parsed, whole_error := Parse_Unsigned_Integer(whole_text, DECIMAL_BASE, 64)
		if whole_error != nil {
			return 0, whole_error
		}
		whole = parsed
	}
	units, fraction_error := fraction_units(kept_fraction(Tail_Text(fraction_text)))
	if fraction_error != nil {
		return 0, fraction_error
	}
	return fixed_point_of(whole, units, negative)
}

// Format_Fixed_Point_Into writes fixed-point number into caller storage.
func Format_Fixed_Point_Into(
	destination Buffer, value fixedpoint.Number, digits fixedpoint.Digit_Count,
) (count Fixed_Point_Text_Count) {
	defer func() {
		Fixed_Point_Text_Count_Invariants(count, "format_fixed_point_into.count")
	}()
	Buffer_Invariants(destination, "format_fixed_point_into.destination")
	fixedpoint.Number_Invariants(value, "format_fixed_point_into.value")
	fixedpoint.Digit_Count_Invariants(digits, "format_fixed_point_into.digits")
	negative := Boolean(value < 0)
	magnitude := uint64(value)
	if negative {
		magnitude = uint64(-int64(value))
	}
	power := int64(1)
	for range int(digits) {
		power *= DECIMAL_BASE
	}
	product_high, product_low := bits.Multiply_64(
		bits.Word_64(magnitude), bits.Multiplier_64(power),
	)
	rounded, carry := bits.Add_64(
		bits.Word_64(product_low), 1<<(fixedpoint.FRACTIONAL_BITS-1), 0,
	)
	scaled := (uint64(product_high)+uint64(carry))<<
		(64-fixedpoint.FRACTIONAL_BITS) |
		uint64(rounded)>>fixedpoint.FRACTIONAL_BITS
	if scaled == 0 {
		negative = false
	}
	unsigned_power := uint64(power)
	whole := Unsigned_Integer(scaled / unsigned_power)
	whole_count := int(digit_count_of(whole, DECIMAL_BASE))
	if negative {
		whole_count++
	}
	text_count := whole_count
	if digits > 0 {
		text_count += 1 + int(digits)
	}
	count = Fixed_Point_Text_Count(text_count)
	aver.Always(
		text_count <= len(destination),
		"Fixed-point destination holds complete result.",
	)
	integer_into(destination[:whole_count], whole, negative, DECIMAL_BASE)
	if digits == 0 {
		return count
	}
	destination[whole_count] = '.'
	fraction := scaled % unsigned_power
	for index := text_count - 1; index > whole_count; index-- {
		destination[index] = byte('0' + fraction%DECIMAL_BASE)
		fraction /= DECIMAL_BASE
	}
	return count
}

// Quote_Into writes double-quoted Go string literal into caller storage.
func Quote_Into(destination Buffer, text Text) (count Quoted_Text_Count) {
	defer func() { Quoted_Text_Count_Invariants(count, "quote_into.count") }()
	Buffer_Invariants(destination, "quote_into.destination")
	Text_Invariants(text, "quote_into.text")
	return quoted_into(destination, text, false, false)
}

// Quote_To_ASCII_Into writes ASCII-only string literal into caller storage.
func Quote_To_ASCII_Into(destination Buffer, text Text) (count Quoted_Text_Count) {
	defer func() { Quoted_Text_Count_Invariants(count, "quote_to_ascii_into.count") }()
	Buffer_Invariants(destination, "quote_to_ascii_into.destination")
	Text_Invariants(text, "quote_to_ascii_into.text")
	return quoted_into(destination, text, true, false)
}

// Quote_To_Graphic_Into writes graphic string literal into caller storage.
func Quote_To_Graphic_Into(destination Buffer, text Text) (count Quoted_Text_Count) {
	defer func() { Quoted_Text_Count_Invariants(count, "quote_to_graphic_into.count") }()
	Buffer_Invariants(destination, "quote_to_graphic_into.destination")
	Text_Invariants(text, "quote_to_graphic_into.text")
	return quoted_into(destination, text, false, true)
}

// Quote_Rune_Into writes character literal into caller storage.
func Quote_Rune_Into(destination Buffer, value Character) (count Character_Text_Count) {
	defer func() { Character_Text_Count_Invariants(count, "quote_rune_into.count") }()
	Buffer_Invariants(destination, "quote_rune_into.destination")
	Character_Invariants(value, "quote_rune_into.value")
	return character_into(destination, value, false, false)
}

// Quote_Rune_To_ASCII_Into writes ASCII-only character literal into caller storage.
func Quote_Rune_To_ASCII_Into(
	destination Buffer, value Character,
) (count Character_Text_Count) {
	defer func() {
		Character_Text_Count_Invariants(count, "quote_rune_to_ascii_into.count")
	}()
	Buffer_Invariants(destination, "quote_rune_to_ascii_into.destination")
	Character_Invariants(value, "quote_rune_to_ascii_into.value")
	return character_into(destination, value, true, false)
}

// Quote_Rune_To_Graphic_Into writes graphic character literal into caller storage.
func Quote_Rune_To_Graphic_Into(
	destination Buffer, value Character,
) (count Character_Text_Count) {
	defer func() {
		Character_Text_Count_Invariants(count, "quote_rune_to_graphic_into.count")
	}()
	Buffer_Invariants(destination, "quote_rune_to_graphic_into.destination")
	Character_Invariants(value, "quote_rune_to_graphic_into.value")
	return character_into(destination, value, false, true)
}

// Can_Backquote reports whether the text stays unchanged inside backquotes.
func Can_Backquote(text Text) (yes Boolean) {
	defer func() {
		Boolean_Invariants(yes, "can_backquote.yes")
	}()
	Text_Invariants(text, "can_backquote.text")
	rest := text
	for len(rest) > 0 {
		first := rest[0]
		if first < byte(utf8.CHARACTER_SELF) {
			rest = rest[1:]
			if first < ' ' {
				if first != '\t' {
					return false
				}
				continue
			}
			if first == '`' {
				return false
			}
			if first == ASCII_BYTE_MAXIMUM {
				return false
			}
			continue
		}
		value, width := utf8.Decode_Character_Text(utf8.Text(rest))
		rest = rest[width:]
		if width == 1 {
			return false
		}
		if value == '\ufeff' {
			// A byte order mark is invisible in the text.
			return false
		}
	}
	return true
}

// Is_Print reports whether the character is printable, which is what the unicode
// package defines: a letter, a mark, a number, punctuation, a symbol, or the ASCII
// space.
func Is_Print(value Character) (yes Boolean) {
	defer func() {
		Boolean_Invariants(yes, "is_print.yes")
	}()
	Character_Invariants(value, "is_print.value")
	if value < 0 {
		return false
	}
	// The first two blocks answer this question without a table, and they hold the
	// characters that ordinary text is made of. The ucd tables decode one hexadecimal
	// blob for each category name at each call, thus a table read costs near a
	// thousand times what this comparison costs.
	if value <= LATIN_1_MAXIMUM {
		return latin_1_is_print(Latin_1_Character(value))
	}
	return Boolean(ucd.Is_Print(ucd.Character(value)))
}

// Reports whether a Latin-1 character is printable. The two printable runs are the
// ASCII characters from the space through the tilde, and the Latin-1 characters from
// the inverted exclamation mark through the end, less the soft hyphen, which a reader
// does not see.
func latin_1_is_print(value Latin_1_Character) (yes Boolean) {
	defer func() {
		Boolean_Invariants(yes, "latin_1_is_print.yes")
	}()
	Latin_1_Character_Invariants(value, "latin_1_is_print.value")
	if value < Latin_1_Character(ASCII_PRINT_MINIMUM) {
		return false
	}
	if value <= Latin_1_Character(ASCII_PRINT_MAXIMUM) {
		return true
	}
	if value < Latin_1_Character(LATIN_1_PRINT_MINIMUM) {
		return false
	}
	return Boolean(value != Latin_1_Character(SOFT_HYPHEN))
}

// Is_Graphic reports whether the character is graphic, which adds the spaces of
// category Zs to the printable set.
func Is_Graphic(value Character) (yes Boolean) {
	defer func() {
		Boolean_Invariants(yes, "is_graphic.yes")
	}()
	Character_Invariants(value, "is_graphic.value")
	if value < 0 {
		return false
	}
	// The graphic set adds one Latin-1 character to the printable set: the no-break
	// space, which category Zs holds.
	if value <= LATIN_1_MAXIMUM {
		if value == NO_BREAK_SPACE {
			return true
		}
		return latin_1_is_print(Latin_1_Character(value))
	}
	return Boolean(ucd.Is_Graphic(ucd.Character(value)))
}

// Unquote_Into decodes complete Go literal into caller storage.
func Unquote_Into(
	destination Buffer, text Text,
) (count Unquoted_Text_Count, err error) {
	defer func() { Unquoted_Text_Count_Invariants(count, "unquote_into.count") }()
	Buffer_Invariants(destination, "unquote_into.destination")
	Text_Invariants(text, "unquote_into.text")
	prefix, value_count, measure_error := measure_literal(text)
	if measure_error != nil {
		return 0, measure_error
	}
	if len(prefix) != len(text) {
		return 0, Error_Syntax
	}
	count = value_count
	aver.Always(
		int(count) <= len(destination),
		"Unquote destination holds complete result.",
	)
	write_literal_value(Unquoted_Buffer(destination[:int(count)]), Literal_Text(text))
	return count, nil
}

// Quoted_Prefix returns the literal at the start of the text, quote marks included.
func Quoted_Prefix(text Text) (quoted Prefix_Text, err error) {
	defer func() {
		Prefix_Text_Invariants(quoted, "quoted_prefix.quoted")
	}()
	Text_Invariants(text, "quoted_prefix.text")
	prefix, _, prefix_error := measure_literal(text)
	return prefix, prefix_error
}

// Unquote_Character decodes the first character of a literal body. The quote mark names
// the literal form, thus it selects the quote escape the body permits.
func Unquote_Character(
	text Text, quote_mark Quote_Mark,
) (value Code_Point, multibyte Boolean, tail Tail_Text, err error) {
	defer func() {
		Code_Point_Invariants(value, "unquote_character.value")
		Boolean_Invariants(multibyte, "unquote_character.multibyte")
		Tail_Text_Invariants(tail, "unquote_character.tail")
	}()
	Text_Invariants(text, "unquote_character.text")
	Quote_Mark_Invariants(quote_mark, "unquote_character.quote_mark")
	if len(text) == 0 {
		return 0, false, "", Error_Syntax
	}
	first := text[0]
	if first == byte(quote_mark) {
		if quote_mark != QUOTE_MARK_NONE {
			return 0, false, "", Error_Syntax
		}
	}
	if first >= byte(utf8.CHARACTER_SELF) {
		decoded, width := utf8.Decode_Character_Text(utf8.Text(text))
		return Code_Point(decoded), true, Tail_Text(text[width:]), nil
	}
	if first != '\\' {
		return Code_Point(first), false, Tail_Text(text[1:]), nil
	}
	if len(text) <= 1 {
		return 0, false, "", Error_Syntax
	}
	value, multibyte, body, err := unquote_escape(
		Text_Byte(text[1]), Body_Text(text[2:]), quote_mark,
	)
	return value, multibyte, Tail_Text(body), err
}

// Measures first literal and decoded value without writing.
func measure_literal(
	text Text,
) (prefix Prefix_Text, count Unquoted_Text_Count, err error) {
	defer func() {
		Prefix_Text_Invariants(prefix, "measure_literal.prefix")
		Unquoted_Text_Count_Invariants(count, "measure_literal.count")
	}()
	Text_Invariants(text, "measure_literal.text")
	if len(text) < LITERAL_TEXT_SIZE_MINIMUM {
		return "", 0, Error_Syntax
	}
	if text[0] == '`' {
		end_offset := int(strings.Index_Byte(strings.Text(text[1:]), '`'))
		if end_offset < 0 {
			return "", 0, Error_Syntax
		}
		end_offset += LITERAL_TEXT_SIZE_MINIMUM
		for _, character := range []byte(text[1 : end_offset-1]) {
			if character != '\r' {
				count++
			}
		}
		return Prefix_Text(text[:end_offset]), count, nil
	}
	if text[0] != QUOTE_MARK_DOUBLE {
		if text[0] != QUOTE_MARK_SINGLE {
			return "", 0, Error_Syntax
		}
	}
	quote_mark := Quote_Mark(text[0])
	body := Text(text[1:])
	for len(body) > 0 {
		if body[0] == byte(quote_mark) {
			if quote_mark == QUOTE_MARK_SINGLE {
				if count == 0 {
					return "", 0, Error_Syntax
				}
			}
			boundary := len(text) - len(body) + 1
			return Prefix_Text(text[:boundary]), count, nil
		}
		if body[0] == '\n' {
			return "", 0, Error_Syntax
		}
		point, multibyte, tail, character_error := Unquote_Character(body, quote_mark)
		if character_error != nil {
			return "", 0, Error_Syntax
		}
		count += Unquoted_Text_Count(decoded_character_size(point, multibyte))
		body = Text(tail)
		if quote_mark == QUOTE_MARK_SINGLE {
			if len(body) == 0 {
				return "", 0, Error_Syntax
			}
			if body[0] != QUOTE_MARK_SINGLE {
				return "", 0, Error_Syntax
			}
			boundary := len(text) - len(body) + 1
			return Prefix_Text(text[:boundary]), count, nil
		}
	}
	return "", 0, Error_Syntax
}

// Gives decoded byte width for one literal body character.
func decoded_character_size(value Code_Point, multibyte Boolean) (size Decoded_Text_Count) {
	defer func() { Decoded_Text_Count_Invariants(size, "decoded_character_size.size") }()
	Code_Point_Invariants(value, "decoded_character_size.value")
	Boolean_Invariants(multibyte, "decoded_character_size.multibyte")
	if value < Code_Point(utf8.CHARACTER_SELF) {
		return ESCAPE_SIZE_MINIMUM
	}
	if !multibyte {
		return ESCAPE_SIZE_MINIMUM
	}
	return Decoded_Text_Count(utf8.Character_Size(utf8.Character(value)))
}

// Writes already-measured complete literal value.
func write_literal_value(destination Unquoted_Buffer, text Literal_Text) {
	Unquoted_Buffer_Invariants(destination, "write_literal_value.destination")
	Literal_Text_Invariants(text, "write_literal_value.text")
	if text[0] == '`' {
		written := 0
		for _, character := range []byte(text[1 : len(text)-1]) {
			if character != '\r' {
				destination[written] = character
				written++
			}
		}
		return
	}
	quote_mark := Quote_Mark(text[0])
	body := Text(text[1:])
	written := 0
	for body[0] != byte(quote_mark) {
		point, multibyte, tail, character_error := Unquote_Character(body, quote_mark)
		aver.Always(
			character_error == nil,
			"Measured literal remains valid while writing.",
		)
		size := int(decoded_character_size(point, multibyte))
		if size == ESCAPE_SIZE_MINIMUM {
			destination[written] = byte(point)
		} else {
			utf8.Encode_Character(
				utf8.Bytes(destination[written:written+size]),
				utf8.Character(point),
			)
		}
		written += size
		body = Text(tail)
	}
}

// Drops each fraction digit that the fixed-point grid cannot hold. The decimal point
// precedes the digits, thus they are the text that one character leaves.
func kept_fraction(text Tail_Text) (kept Fraction_Text) {
	defer func() {
		Fraction_Text_Invariants(kept, "kept_fraction.kept")
	}()
	Tail_Text_Invariants(text, "kept_fraction.text")
	if len(text) > FRACTION_TEXT_SIZE_MAXIMUM {
		return Fraction_Text(text[:FRACTION_TEXT_SIZE_MAXIMUM])
	}
	return Fraction_Text(text)
}

// Reads the fraction digits into fixed-point units, rounding half away from zero.
func fraction_units(digits Fraction_Text) (units Fraction_Units, err error) {
	defer func() {
		Fraction_Units_Invariants(units, "fraction_units.units")
	}()
	Fraction_Text_Invariants(digits, "fraction_units.digits")
	if len(digits) == 0 {
		return 0, nil
	}
	parsed, parse_error := Parse_Unsigned_Integer(Text(digits), DECIMAL_BASE, 64)
	if parse_error != nil {
		return 0, parse_error
	}
	power := uint64(1)
	for digit_index := 0; digit_index < len(digits); digit_index++ {
		power *= DECIMAL_BASE
	}
	// The scaled fraction is a 128-bit product, thus the multiply cannot lose a digit
	// that a shift of a 64-bit word would drop. Half the power rounds away from zero.
	high, low := bits.Multiply_64(bits.Word_64(parsed), fixedpoint.SCALE)
	rounded, carry := bits.Add_64(bits.Word_64(low), bits.Addend_64(power/2), 0)
	quotient, _ := bits.Divide_64(
		bits.Dividend_High_64(uint64(high)+uint64(carry)),
		bits.Dividend_Low_64(rounded), bits.Divisor_64(power),
	)
	return Fraction_Units(quotient), nil
}

// Joins a whole part and a fraction into one fixed-point number, rejecting a magnitude
// that the storage cannot hold.
func fixed_point_of(
	whole Unsigned_Integer, units Fraction_Units, negative Boolean,
) (value fixedpoint.Number, err error) {
	defer func() {
		fixedpoint.Number_Invariants(value, "fixed_point_of.value")
	}()
	Unsigned_Integer_Invariants(whole, "fixed_point_of.whole")
	Fraction_Units_Invariants(units, "fixed_point_of.units")
	Boolean_Invariants(negative, "fixed_point_of.negative")
	magnitude_maximum := FIXED_POINT_UNITS_POSITIVE_MAXIMUM
	if negative {
		magnitude_maximum = FIXED_POINT_UNITS_NEGATIVE_MAXIMUM
	}
	if uint64(whole) > magnitude_maximum/fixedpoint.SCALE {
		return 0, Error_Range
	}
	// The whole part is already inside the storage domain, thus the product holds one
	// word and the fraction cannot carry out of it.
	_, low := bits.Multiply_64(bits.Word_64(whole), fixedpoint.SCALE)
	scaled, _ := bits.Add_64(bits.Word_64(low), bits.Addend_64(units), 0)
	magnitude := uint64(scaled)
	if magnitude > magnitude_maximum {
		return 0, Error_Range
	}
	if negative {
		// The negation of the largest magnitude wraps to the smallest storage value,
		// which is the value the text names.
		return fixedpoint.Number(-int64(magnitude)), nil
	}
	return fixedpoint.Number(magnitude), nil
}

// Reads the radix from the text prefix when the caller states none.
func resolve_base(text Number_Text, base Implied_Base) (radix Base, digits Number_Text) {
	defer func() {
		Base_Invariants(radix, "resolve_base.radix")
		Number_Text_Invariants(digits, "resolve_base.digits")
	}()
	Number_Text_Invariants(text, "resolve_base.text")
	Implied_Base_Invariants(base, "resolve_base.base")
	if base != 0 {
		return Base(base), text
	}
	if text[0] != '0' {
		return DECIMAL_BASE, text
	}
	if len(text) < 3 {
		return 8, text
	}
	switch lowercase(Text_Byte(text[1])) {
	case 'b':
		return 2, text[2:]
	case 'o':
		return 8, text[2:]
	case 'x':
		return 16, text[2:]
	}
	return 8, text[1:]
}

// Reads each digit of the text into one unsigned value. The cutoff catches an overflow
// before the multiply that would wrap.
func accumulate_digits(
	digits Number_Text, radix Base, bit_size Bit_Size, underscores Boolean,
) (value Unsigned_Integer, err error) {
	defer func() {
		Unsigned_Integer_Invariants(value, "accumulate_digits.value")
	}()
	Number_Text_Invariants(digits, "accumulate_digits.digits")
	Base_Invariants(radix, "accumulate_digits.radix")
	Bit_Size_Invariants(bit_size, "accumulate_digits.bit_size")
	Boolean_Invariants(underscores, "accumulate_digits.underscores")
	width := int(bit_size)
	if width == 0 {
		width = MACHINE_INTEGER_BITS
	}
	value_maximum := uint64(1)<<uint(width) - 1
	cutoff := UNSIGNED_64_MAXIMUM/uint64(radix) + 1
	total := uint64(0)
	for _, character := range []byte(digits) {
		if character == '_' {
			if !underscores {
				return 0, Error_Syntax
			}
			continue
		}
		digit, known := digit_value(Text_Byte(character))
		if !known {
			return 0, Error_Syntax
		}
		if uint64(digit) >= uint64(radix) {
			return 0, Error_Syntax
		}
		if total >= cutoff {
			return Unsigned_Integer(value_maximum), Error_Range
		}
		total *= uint64(radix)
		if total+uint64(digit) < total {
			return Unsigned_Integer(value_maximum), Error_Range
		}
		total += uint64(digit)
		if total > value_maximum {
			return Unsigned_Integer(value_maximum), Error_Range
		}
	}
	return Unsigned_Integer(total), nil
}

// Reports whether each underscore of the text sits between two digits or between a base
// prefix and a digit. One pass over the whole text lets the digit reader skip an
// underscore without a second rule.
func underscores_separate_digits(text Number_Text) (yes Boolean) {
	defer func() {
		Boolean_Invariants(yes, "underscores_separate_digits.yes")
	}()
	Number_Text_Invariants(text, "underscores_separate_digits.text")
	// The previous class: a caret starts the number, a zero is a digit or a prefix, an
	// underscore is itself, and an exclamation mark is any other character.
	previous := byte('^')
	body := text
	switch body[0] {
	case '-', '+':
		body = body[1:]
	}
	start_index := 0
	hexadecimal := false
	if len(body) >= 2 {
		if body[0] == '0' {
			switch lowercase(Text_Byte(body[1])) {
			case 'b', 'o', 'x':
				start_index = 2
				previous = '0'
				hexadecimal = lowercase(Text_Byte(body[1])) == 'x'
			}
		}
	}
	for character_index := start_index; character_index < len(body); character_index++ {
		character := Text_Byte(body[character_index])
		if separates_digits(character, Boolean(hexadecimal)) {
			previous = '0'
			continue
		}
		if character == '_' {
			if previous != '0' {
				return false
			}
			previous = '_'
			continue
		}
		if previous == '_' {
			return false
		}
		previous = '!'
	}
	return Boolean(previous != '_')
}

// Reads the value of one digit symbol in the largest radix.
func digit_value(character Text_Byte) (digit Digit_Value, known Boolean) {
	defer func() {
		Digit_Value_Invariants(digit, "digit_value.digit")
		Boolean_Invariants(known, "digit_value.known")
	}()
	Text_Byte_Invariants(character, "digit_value.character")
	if character >= '0' {
		if character <= '9' {
			return Digit_Value(character - '0'), true
		}
	}
	if lowercase(character) >= 'a' {
		if lowercase(character) <= 'z' {
			alphabetic_value := Digit_Value(lowercase(character) - 'a')
			return alphabetic_value + Digit_Value(DECIMAL_BASE), true
		}
	}
	return 0, false
}

// Folds a letter to its lowercase form. It leaves a lowercase letter unchanged and can
// map a character that is not a letter to another character that is not a letter, which
// the callers admit because each one tests the result against a letter range.
func lowercase(character Text_Byte) (folded Folded_Byte) {
	defer func() {
		Folded_Byte_Invariants(folded, "lowercase.folded")
	}()
	Text_Byte_Invariants(character, "lowercase.character")
	return Folded_Byte(character | ('x' - 'X'))
}

// Reports whether the character is a digit that an underscore can separate.
func separates_digits(character Text_Byte, hexadecimal Boolean) (yes Boolean) {
	defer func() {
		Boolean_Invariants(yes, "separates_digits.yes")
	}()
	Text_Byte_Invariants(character, "separates_digits.character")
	Boolean_Invariants(hexadecimal, "separates_digits.hexadecimal")
	if character >= '0' {
		if character <= '9' {
			return true
		}
	}
	if !hexadecimal {
		return false
	}
	if lowercase(character) >= 'a' {
		if lowercase(character) <= 'f' {
			return true
		}
	}
	return false
}

// Writes quoted text only after exact size proves destination sufficient.
func quoted_into(
	destination Buffer, text Text, ascii_only Boolean, graphic_only Boolean,
) (count Quoted_Text_Count) {
	defer func() { Quoted_Text_Count_Invariants(count, "quoted_into.count") }()
	Buffer_Invariants(destination, "quoted_into.destination")
	Text_Invariants(text, "quoted_into.text")
	Boolean_Invariants(ascii_only, "quoted_into.ascii_only")
	Boolean_Invariants(graphic_only, "quoted_into.graphic_only")
	count = quoted_text_size(text, ascii_only, graphic_only)
	aver.Always(
		int(count) <= len(destination),
		"Quoted text destination holds complete result.",
	)
	destination[0] = QUOTE_MARK_DOUBLE
	written := 1
	rest := text
	for len(rest) > 0 {
		value, width := utf8.Decode_Character_Text(utf8.Text(rest))
		if width == utf8.CHARACTER_SIZE_MINIMUM {
			if value == utf8.REPLACEMENT_CHARACTER {
				if rest[0] >= byte(utf8.CHARACTER_SELF) {
					invalid_byte := Invalid_Text_Byte(rest[0])
					Invalid_Text_Byte_Invariants(
						invalid_byte, "quoted_into.invalid_byte",
					)
					destination[written], destination[written+1] = '\\', 'x'
					destination[written+2] =
						HEXADECIMAL_SYMBOLS[invalid_byte>>4]
					destination[written+3] =
						HEXADECIMAL_SYMBOLS[invalid_byte&0xf]
					written += BYTE_ESCAPE_SIZE
					rest = rest[1:]
					continue
				}
			}
		}
		body_count := escaped_text_size(
			Code_Point(value), QUOTE_MARK_DOUBLE, ascii_only, graphic_only,
		)
		escape_into(
			Escape_Destination(destination[written:written+int(body_count)]),
			Code_Point(value), QUOTE_MARK_DOUBLE,
			ascii_only, graphic_only,
		)
		written += int(body_count)
		rest = rest[int(width):]
	}
	destination[written] = QUOTE_MARK_DOUBLE
	return count
}

// Counts quoted text without borrowing hidden scratch storage.
func quoted_text_size(
	text Text, ascii_only Boolean, graphic_only Boolean,
) (count Quoted_Text_Count) {
	defer func() { Quoted_Text_Count_Invariants(count, "quoted_text_size.count") }()
	Text_Invariants(text, "quoted_text_size.text")
	Boolean_Invariants(ascii_only, "quoted_text_size.ascii_only")
	Boolean_Invariants(graphic_only, "quoted_text_size.graphic_only")
	count = QUOTED_TEXT_SIZE_MINIMUM
	for len(text) > 0 {
		value, width := utf8.Decode_Character_Text(utf8.Text(text))
		if width == utf8.CHARACTER_SIZE_MINIMUM {
			if value == utf8.REPLACEMENT_CHARACTER {
				if text[0] >= byte(utf8.CHARACTER_SELF) {
					count += BYTE_ESCAPE_SIZE
					text = text[1:]
					continue
				}
			}
		}
		count += Quoted_Text_Count(escaped_text_size(
			Code_Point(value), QUOTE_MARK_DOUBLE, ascii_only, graphic_only,
		))
		text = text[int(width):]
	}
	return count
}

// Writes one quoted character only after exact size proves destination sufficient.
func character_into(
	destination Buffer, value Character, ascii_only Boolean, graphic_only Boolean,
) (count Character_Text_Count) {
	defer func() { Character_Text_Count_Invariants(count, "character_into.count") }()
	Buffer_Invariants(destination, "character_into.destination")
	Character_Invariants(value, "character_into.value")
	Boolean_Invariants(ascii_only, "character_into.ascii_only")
	Boolean_Invariants(graphic_only, "character_into.graphic_only")
	point := Code_Point(value)
	if !utf8.Valid_Character(utf8.Character(value)) {
		point = UNICODE_REPLACEMENT
	}
	body_count := escaped_text_size(
		point, QUOTE_MARK_SINGLE, ascii_only, graphic_only,
	)
	count = Character_Text_Count(int(body_count) + LITERAL_TEXT_SIZE_MINIMUM)
	aver.Always(
		int(count) <= len(destination),
		"Quoted character destination holds complete result.",
	)
	destination[0] = QUOTE_MARK_SINGLE
	escape_into(
		Escape_Destination(destination[1:1+int(body_count)]), point, QUOTE_MARK_SINGLE,
		ascii_only, graphic_only,
	)
	destination[int(count)-1] = QUOTE_MARK_SINGLE
	return count
}

// Counts one body character under quote policy.
func escaped_text_size(
	value Code_Point, quote_mark Literal_Quote_Mark,
	ascii_only Boolean, graphic_only Boolean,
) (count Escape_Text_Count) {
	defer func() { Escape_Text_Count_Invariants(count, "escaped_text_size.count") }()
	Code_Point_Invariants(value, "escaped_text_size.value")
	Literal_Quote_Mark_Invariants(quote_mark, "escaped_text_size.quote_mark")
	Boolean_Invariants(ascii_only, "escaped_text_size.ascii_only")
	Boolean_Invariants(graphic_only, "escaped_text_size.graphic_only")
	if value == Code_Point(quote_mark) {
		return ESCAPE_SEQUENCE_SIZE_MINIMUM
	}
	if value == '\\' {
		return ESCAPE_SEQUENCE_SIZE_MINIMUM
	}
	if ascii_only {
		if value < Code_Point(utf8.CHARACTER_SELF) {
			if Is_Print(Character(value)) {
				return ESCAPE_SIZE_MINIMUM
			}
		}
		return Escape_Text_Count(escape_sequence_size(value))
	}
	if stays_unescaped(value, graphic_only) {
		return Escape_Text_Count(utf8.Character_Size(utf8.Character(value)))
	}
	return Escape_Text_Count(escape_sequence_size(value))
}

// Counts mandatory escape representation.
func escape_sequence_size(value Code_Point) (count Escape_Sequence_Count) {
	defer func() {
		Escape_Sequence_Count_Invariants(count, "escape_sequence_size.count")
	}()
	Code_Point_Invariants(value, "escape_sequence_size.value")
	if value < Code_Point(len(CONTROL_ESCAPE_LETTERS)) {
		if CONTROL_ESCAPE_LETTERS[value] != 0 {
			return ESCAPE_SEQUENCE_SIZE_MINIMUM
		}
	}
	switch {
	case value < ' ', value == Code_Point(ASCII_BYTE_MAXIMUM):
		return BYTE_ESCAPE_SIZE
	case value < 0x10000:
		return SHORT_UNICODE_ESCAPE_SIZE
	default:
		return LONG_UNICODE_ESCAPE_SIZE
	}
}

// Writes one body character into proven storage.
func escape_into(
	destination Escape_Destination, value Code_Point, quote_mark Literal_Quote_Mark,
	ascii_only Boolean, graphic_only Boolean,
) (count Escape_Text_Count) {
	defer func() { Escape_Text_Count_Invariants(count, "escape_into.count") }()
	Escape_Destination_Invariants(destination, "escape_into.destination")
	Code_Point_Invariants(value, "escape_into.value")
	Literal_Quote_Mark_Invariants(quote_mark, "escape_into.quote_mark")
	Boolean_Invariants(ascii_only, "escape_into.ascii_only")
	Boolean_Invariants(graphic_only, "escape_into.graphic_only")
	count = escaped_text_size(value, quote_mark, ascii_only, graphic_only)
	aver.Always(
		int(count) <= len(destination),
		"Escaped character destination holds complete result.",
	)
	if value == Code_Point(quote_mark) {
		destination[0], destination[1] = '\\', byte(value)
		return count
	}
	if value == '\\' {
		destination[0], destination[1] = '\\', byte(value)
		return count
	}
	if ascii_only {
		if value < Code_Point(utf8.CHARACTER_SELF) {
			if Is_Print(Character(value)) {
				destination[0] = byte(value)
				return count
			}
		}
		write_escape_sequence(Escape_Sequence_Buffer(destination[:int(count)]), value)
		return count
	}
	if stays_unescaped(value, graphic_only) {
		utf8.Encode_Character(
			utf8.Bytes(destination[:int(count)]), utf8.Character(value),
		)
		return count
	}
	write_escape_sequence(Escape_Sequence_Buffer(destination[:int(count)]), value)
	return count
}

// Writes mandatory escape representation into proven storage.
func write_escape_sequence(destination Escape_Sequence_Buffer, value Code_Point) {
	Escape_Sequence_Buffer_Invariants(destination, "write_escape_sequence.destination")
	Code_Point_Invariants(value, "write_escape_sequence.value")
	count := escape_sequence_size(value)
	aver.Always(
		int(count) <= len(destination),
		"Escape sequence destination holds complete result.",
	)
	if value < Code_Point(len(CONTROL_ESCAPE_LETTERS)) {
		letter := CONTROL_ESCAPE_LETTERS[value]
		if letter != 0 {
			destination[0], destination[1] = '\\', letter
			return
		}
	}
	digit_count := HEXADECIMAL_DIGIT_COUNT
	escape_letter := byte('x')
	if value >= ' ' {
		if value != Code_Point(ASCII_BYTE_MAXIMUM) {
			digit_count = SHORT_UNICODE_DIGIT_COUNT
			escape_letter = 'u'
			if value >= 0x10000 {
				digit_count = LONG_UNICODE_DIGIT_COUNT
				escape_letter = 'U'
			}
		}
	}
	destination[0], destination[1] = '\\', escape_letter
	for index := digit_count - 1; index >= 0; index-- {
		destination[index+2] = HEXADECIMAL_SYMBOLS[value&0xf]
		value >>= 4
	}
}

// Reports whether the quoted form keeps the character as it is.
func stays_unescaped(value Code_Point, graphic_only Boolean) (yes Boolean) {
	defer func() {
		Boolean_Invariants(yes, "stays_unescaped.yes")
	}()
	Code_Point_Invariants(value, "stays_unescaped.value")
	Boolean_Invariants(graphic_only, "stays_unescaped.graphic_only")
	if Is_Print(Character(value)) {
		return true
	}
	if !graphic_only {
		return false
	}
	return Is_Graphic(Character(value))
}

// Decodes the escape sequence that follows a backslash.
func unquote_escape(
	first Text_Byte, body Body_Text, quote_mark Quote_Mark,
) (value Code_Point, multibyte Boolean, tail Body_Text, err error) {
	defer func() {
		Code_Point_Invariants(value, "unquote_escape.value")
		Boolean_Invariants(multibyte, "unquote_escape.multibyte")
		Body_Text_Invariants(tail, "unquote_escape.tail")
	}()
	Text_Byte_Invariants(first, "unquote_escape.first")
	Body_Text_Invariants(body, "unquote_escape.body")
	Quote_Mark_Invariants(quote_mark, "unquote_escape.quote_mark")
	switch first {
	case 'a':
		return '\a', false, body, nil
	case 'b':
		return '\b', false, body, nil
	case 'f':
		return '\f', false, body, nil
	case 'n':
		return '\n', false, body, nil
	case 'r':
		return '\r', false, body, nil
	case 't':
		return '\t', false, body, nil
	case 'v':
		return '\v', false, body, nil
	case '\\':
		return '\\', false, body, nil
	}
	digit_count, hexadecimal := escape_digit_count(first)
	if hexadecimal {
		point, wide, rest, escape_err := escape_tail(body, digit_count)
		return point, wide, Body_Text(rest), escape_err
	}
	switch first {
	case '0', '1', '2', '3', '4', '5', '6', '7':
		point, rest, escape_err := octal_tail(Octal_Digit_Byte(first), body)
		return Code_Point(point), false, Body_Text(rest), escape_err
	case QUOTE_MARK_SINGLE, QUOTE_MARK_DOUBLE:
		if first != Text_Byte(quote_mark) {
			return 0, false, "", Error_Syntax
		}
		return Code_Point(first), false, body, nil
	}
	return 0, false, "", Error_Syntax
}

// Reads how many hexadecimal digits an escape letter takes. A letter that starts no
// hexadecimal escape reports the shortest count, which the caller then ignores.
func escape_digit_count(
	first Text_Byte,
) (digit_count Digit_Count, hexadecimal Boolean) {
	defer func() {
		Digit_Count_Invariants(digit_count, "escape_digit_count.digit_count")
		Boolean_Invariants(hexadecimal, "escape_digit_count.hexadecimal")
	}()
	Text_Byte_Invariants(first, "escape_digit_count.first")
	switch first {
	case 'x':
		return HEXADECIMAL_DIGIT_COUNT, true
	case 'u':
		return SHORT_UNICODE_DIGIT_COUNT, true
	case 'U':
		return LONG_UNICODE_DIGIT_COUNT, true
	}
	return HEXADECIMAL_DIGIT_COUNT, false
}

// Reads the digits of a byte or code point escape. A byte escape stays one byte, thus
// only the two code point forms report a multibyte value.
func escape_tail(
	body Body_Text, digit_count Digit_Count,
) (value Code_Point, multibyte Boolean, tail Escape_Tail_Text, err error) {
	defer func() {
		Code_Point_Invariants(value, "escape_tail.value")
		Boolean_Invariants(multibyte, "escape_tail.multibyte")
		Escape_Tail_Text_Invariants(tail, "escape_tail.tail")
	}()
	Body_Text_Invariants(body, "escape_tail.body")
	Digit_Count_Invariants(digit_count, "escape_tail.digit_count")
	if len(body) < int(digit_count) {
		return 0, false, "", Error_Syntax
	}
	decoded := Code_Point(0)
	for digit_index := 0; digit_index < int(digit_count); digit_index++ {
		digit, known := hexadecimal_value(Text_Byte(body[digit_index]))
		if !known {
			return 0, false, "", Error_Syntax
		}
		decoded = decoded<<4 | Code_Point(digit)
	}
	rest := Escape_Tail_Text(body[digit_count:])
	if digit_count == HEXADECIMAL_DIGIT_COUNT {
		return decoded, false, rest, nil
	}
	if !utf8.Valid_Character(utf8.Character(decoded)) {
		return 0, false, "", Error_Syntax
	}
	return decoded, true, rest, nil
}

// Reads the two digits that follow the first digit of an octal byte escape.
func octal_tail(
	first Octal_Digit_Byte, body Body_Text,
) (value Byte_Point, tail Escape_Tail_Text, err error) {
	defer func() {
		Byte_Point_Invariants(value, "octal_tail.value")
		Escape_Tail_Text_Invariants(tail, "octal_tail.tail")
	}()
	Octal_Digit_Byte_Invariants(first, "octal_tail.first")
	Body_Text_Invariants(body, "octal_tail.body")
	if len(body) < OCTAL_ESCAPE_DIGIT_COUNT {
		return 0, "", Error_Syntax
	}
	decoded := int32(first) - '0'
	for digit_index := 0; digit_index < OCTAL_ESCAPE_DIGIT_COUNT; digit_index++ {
		digit := int32(body[digit_index]) - '0'
		if digit < 0 {
			return 0, "", Error_Syntax
		}
		if digit > 7 {
			return 0, "", Error_Syntax
		}
		decoded = decoded<<3 | digit
	}
	if decoded > OCTAL_ESCAPE_MAXIMUM {
		return 0, "", Error_Syntax
	}
	return Byte_Point(decoded), Escape_Tail_Text(body[OCTAL_ESCAPE_DIGIT_COUNT:]), nil
}

// Reads the value of one hexadecimal digit.
func hexadecimal_value(character Text_Byte) (digit Hexadecimal_Value, known Boolean) {
	defer func() {
		Hexadecimal_Value_Invariants(digit, "hexadecimal_value.digit")
		Boolean_Invariants(known, "hexadecimal_value.known")
	}()
	Text_Byte_Invariants(character, "hexadecimal_value.character")
	if character >= '0' {
		if character <= '9' {
			return Hexadecimal_Value(character - '0'), true
		}
	}
	if lowercase(character) >= 'a' {
		if lowercase(character) <= 'f' {
			return Hexadecimal_Value(lowercase(character)-'a') +
				Hexadecimal_Value(DECIMAL_BASE), true
		}
	}
	return 0, false
}
