// Package strconv converts between text and the Go scalar types. It is a port of the
// standard library package of the same name. Four differences follow from the house
// rules: every text and buffer domain has a declared size limit, an out-of-domain Base
// or Bit_Size panics instead of returning an error value, the printable and graphic
// facts come from unicode instead of a private table, and float and complex conversion
// is absent because the deterministic tier admits no float type.
package strconv

import (
	"errors"

	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/math/fixedpoint"
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

// PLAIN_ASCII_CHARACTER_TEXTS holds one three-byte literal for each printable ASCII value.
const PLAIN_ASCII_CHARACTER_TEXTS = "' ''!''\"''#''$''%''&'''''('')''*''+'',''-''.'" +
	"'/''0''1''2''3''4''5''6''7''8''9'':'';''<''='" +
	"'>''?''@''A''B''C''D''E''F''G''H''I''J''K''L'" +
	"'M''N''O''P''Q''R''S''T''U''V''W''X''Y''Z''['" +
	"'\\'']''^''_''`''a''b''c''d''e''f''g''h''i''j'" +
	"'k''l''m''n''o''p''q''r''s''t''u''v''w''x''y'" +
	"'z''{''|''}''~'"

// DOUBLE_QUOTED_PLAIN_EXCLUSIONS select the bytes that require literal decoding.
const DOUBLE_QUOTED_PLAIN_EXCLUSIONS = "\"\\\n"

// DECIMAL_PAIRS holds each two-digit decimal value at twice its value.
const DECIMAL_PAIRS = "00010203040506070809" +
	"10111213141516171819" +
	"20212223242526272829" +
	"30313233343536373839" +
	"40414243444546474849" +
	"50515253545556575859" +
	"60616263646566676869" +
	"70717273747576777879" +
	"80818283848586878889" +
	"90919293949596979899"

// UNICODE_REPLACEMENT stands in for text that is not valid UTF-8.
const UNICODE_REPLACEMENT = Code_Point(utf8.REPLACEMENT_CHARACTER)

// TEXT_SIZE_MINIMUM is the size of empty text.
const TEXT_SIZE_MINIMUM = 0

// TEXT_SIZE_MAXIMUM caps the text a conversion reads. A conversion holds its whole
// input and its whole output in memory, thus the cap is what keeps both bounded.
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

// ESCAPE_TAIL_SIZE_MAXIMUM is the text that a two-character escape leaves in a body.
const ESCAPE_TAIL_SIZE_MAXIMUM = TEXT_SIZE_MAXIMUM - BYTE_ESCAPE_SIZE

// QUOTED_TEXT_SIZE_MINIMUM is the size of an empty literal, two quote marks.
const QUOTED_TEXT_SIZE_MINIMUM = LITERAL_TEXT_SIZE_MINIMUM

// QUOTED_TEXT_SIZE_MAXIMUM is the size of a literal whose every input byte needs a
// four-character byte escape.
const QUOTED_TEXT_SIZE_MAXIMUM = TEXT_SIZE_MAXIMUM*BYTE_ESCAPE_SIZE +
	QUOTED_TEXT_SIZE_MINIMUM

// QUOTED_CAPACITY_NUMERATOR reserves three parts of each quoted-text estimate.
const QUOTED_CAPACITY_NUMERATOR = 3

// QUOTED_CAPACITY_DENOMINATOR makes that estimate one and a half input lengths. This
// helps escaped text without charging plain text for the worst case.
const QUOTED_CAPACITY_DENOMINATOR = 2

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

// ESCAPE_DESTINATION_SIZE_MINIMUM is the opening quote mark, which the buffer holds
// before the first character.
const ESCAPE_DESTINATION_SIZE_MINIMUM = 1

// ESCAPE_DESTINATION_SIZE_MAXIMUM is the opening quote mark and every byte escape but
// the last, which is the largest buffer that one more character can extend.
const ESCAPE_DESTINATION_SIZE_MAXIMUM = ESCAPE_DESTINATION_SIZE_MINIMUM +
	(TEXT_SIZE_MAXIMUM-1)*BYTE_ESCAPE_SIZE

// ESCAPE_LETTER_SIZE is the backslash and the letter that open a digit escape.
const ESCAPE_LETTER_SIZE = 2

// ESCAPE_LETTER_BUFFER_SIZE_MINIMUM is the opening quote mark and one escape
// letter.
const ESCAPE_LETTER_BUFFER_SIZE_MINIMUM = ESCAPE_DESTINATION_SIZE_MINIMUM +
	ESCAPE_LETTER_SIZE

// ESCAPE_LETTER_BUFFER_SIZE_MAXIMUM adds one escape letter to the largest
// destination.
const ESCAPE_LETTER_BUFFER_SIZE_MAXIMUM = ESCAPE_DESTINATION_SIZE_MAXIMUM +
	ESCAPE_LETTER_SIZE

// ESCAPE_SEQUENCE_BUFFER_SIZE_MINIMUM is the opening quote mark and the shortest
// escape sequence.
const ESCAPE_SEQUENCE_BUFFER_SIZE_MINIMUM = ESCAPE_DESTINATION_SIZE_MINIMUM +
	ESCAPE_SEQUENCE_SIZE_MINIMUM

// HEXADECIMAL_BUFFER_SIZE_MINIMUM is the opening quote mark, one escape letter,
// and the digits of the shortest digit escape.
const HEXADECIMAL_BUFFER_SIZE_MINIMUM = ESCAPE_LETTER_BUFFER_SIZE_MINIMUM +
	HEXADECIMAL_DIGIT_COUNT

// ESCAPED_BUFFER_SIZE_MINIMUM is the opening quote mark and one plain character.
const ESCAPED_BUFFER_SIZE_MINIMUM = ESCAPE_DESTINATION_SIZE_MINIMUM + 1

// ESCAPED_BUFFER_SIZE_MAXIMUM adds the last byte escape to the largest destination.
const ESCAPED_BUFFER_SIZE_MAXIMUM = ESCAPE_DESTINATION_SIZE_MAXIMUM + BYTE_ESCAPE_SIZE

// ASCII_QUOTE_DESTINATION_SIZE_MINIMUM is the opening quote before any body byte.
const ASCII_QUOTE_DESTINATION_SIZE_MINIMUM = ESCAPE_DESTINATION_SIZE_MINIMUM

// ASCII_QUOTE_DESTINATION_SIZE_MAXIMUM leaves room for one final byte escape.
const ASCII_QUOTE_DESTINATION_SIZE_MAXIMUM = ESCAPE_DESTINATION_SIZE_MAXIMUM

// ASCII_QUOTE_DESTINATION_SIZE_HOLE cannot occur: a non-ASCII character adds at least
// two bytes before the ASCII writer can run again.
const ASCII_QUOTE_DESTINATION_SIZE_HOLE = ESCAPED_BUFFER_SIZE_MINIMUM

// OPEN_QUOTED_BUFFER_SIZE_MINIMUM is one opening quote and no body.
const OPEN_QUOTED_BUFFER_SIZE_MINIMUM = ESCAPE_DESTINATION_SIZE_MINIMUM

// OPEN_QUOTED_BUFFER_SIZE_MAXIMUM is the body before its closing quote.
const OPEN_QUOTED_BUFFER_SIZE_MAXIMUM = ESCAPED_BUFFER_SIZE_MAXIMUM

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

// BUFFER_SIZE_MAXIMUM caps the buffer an append conversion extends.
const BUFFER_SIZE_MAXIMUM = 4096

// BOOLEAN_BUFFER_SIZE_MAXIMUM is a full buffer and the longest Boolean word.
const BOOLEAN_BUFFER_SIZE_MAXIMUM = BUFFER_SIZE_MAXIMUM + BOOLEAN_TEXT_SIZE_FALSE

// INTEGER_BUFFER_SIZE_MAXIMUM is a full buffer and the widest signed number.
const INTEGER_BUFFER_SIZE_MAXIMUM = BUFFER_SIZE_MAXIMUM + INTEGER_TEXT_SIZE_MAXIMUM

// DIGIT_BUFFER_SIZE_MAXIMUM is a full buffer and the widest unsigned number.
const DIGIT_BUFFER_SIZE_MAXIMUM = BUFFER_SIZE_MAXIMUM + DIGIT_TEXT_SIZE_MAXIMUM

// QUOTED_BUFFER_SIZE_MAXIMUM is a full buffer and the longest literal.
const QUOTED_BUFFER_SIZE_MAXIMUM = BUFFER_SIZE_MAXIMUM + QUOTED_TEXT_SIZE_MAXIMUM

// CHARACTER_BUFFER_SIZE_MAXIMUM is a full buffer and the longest character literal.
const CHARACTER_BUFFER_SIZE_MAXIMUM = BUFFER_SIZE_MAXIMUM + CHARACTER_TEXT_SIZE_MAXIMUM

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

// FIXED_POINT_BUFFER_SIZE_MAXIMUM is a full buffer and the longest fixed-point text.
const FIXED_POINT_BUFFER_SIZE_MAXIMUM = BUFFER_SIZE_MAXIMUM + FIXED_POINT_TEXT_SIZE_MAXIMUM

// FIXED_POINT_TEXT_SIZE_MINIMUM is the size of a one-digit number. It names the size
// fixedpoint declares, because fixedpoint writes the text this package appends.
const FIXED_POINT_TEXT_SIZE_MINIMUM = fixedpoint.TEXT_SIZE_MINIMUM

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

// DECIMAL_CHUNK_BASE is the largest decimal power that fits in 32 bits.
const DECIMAL_CHUNK_BASE uint64 = 1_000_000_000

// DECIMAL_CHUNK_SHIFT cheaply detects every value that can need a nine-digit chunk.
const DECIMAL_CHUNK_SHIFT = 29

// DECIMAL_CHUNK_PAIR_COUNT is the pair count before the leading digit of a chunk.
const DECIMAL_CHUNK_PAIR_COUNT = 4

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
const INTEGER_64_MINIMUM_DECIMAL_TEXT Integer_Text = "-9223372036854775808"

// INTEGER_64_MAXIMUM_DECIMAL_TEXT is the decimal form of the upper storage boundary.
const INTEGER_64_MAXIMUM_DECIMAL_TEXT Integer_Text = "9223372036854775807"

// UNSIGNED_64_MINIMUM is the smallest unsigned 64-bit integer.
const UNSIGNED_64_MINIMUM uint64 = bits.WORD_64_MINIMUM

// UNSIGNED_64_MAXIMUM is the largest unsigned 64-bit integer.
const UNSIGNED_64_MAXIMUM uint64 = bits.WORD_64_MAXIMUM

// SIGNED_MAGNITUDE_MINIMUM is the magnitude of zero.
const SIGNED_MAGNITUDE_MINIMUM uint64 = bits.WORD_64_MINIMUM

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
func Prefix_Text_Invariants(value Prefix_Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Text_Invariants(value Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Number_Text is text that holds at least one character of a number.
type Number_Text string

// Number_Text_Invariants bounds the text that a number reader takes.
func Number_Text_Invariants(value Number_Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), NUMBER_TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Literal_Text is text that holds at least the two quote marks of a literal.
type Literal_Text string

// Literal_Text_Invariants bounds the text that a literal reader takes.
func Literal_Text_Invariants(value Literal_Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), LITERAL_TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Tail_Text is the text that follows one character of a literal body.
type Tail_Text string

// Tail_Text_Invariants bounds the text that one character leaves.
func Tail_Text_Invariants(value Tail_Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, TAIL_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Body_Text is the text that the two quote marks of a literal leave.
type Body_Text string

// Body_Text_Invariants bounds the text that a two-character prefix leaves.
func Body_Text_Invariants(value Body_Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, BODY_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Escape_Tail_Text is the text that an escape sequence leaves in a literal body.
type Escape_Tail_Text string

// Escape_Tail_Text_Invariants bounds the text that a two-character escape leaves.
func Escape_Tail_Text_Invariants(value Escape_Tail_Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, ESCAPE_TAIL_SIZE_MAXIMUM).
		Ensure()
}

// Quoted_Text is a Go string literal.
type Quoted_Text string

// Quoted_Text_Invariants bounds a string literal, whose every input byte can need a
// four-character escape.
func Quoted_Text_Invariants(value Quoted_Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), QUOTED_TEXT_SIZE_MINIMUM, QUOTED_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Character_Text is a Go character literal.
type Character_Text string

// Character_Text_Invariants bounds a character literal.
func Character_Text_Invariants(value Character_Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), CHARACTER_TEXT_SIZE_MINIMUM, CHARACTER_TEXT_SIZE_MAXIMUM,
		).
		Ensure()
}

// Escape_Destination is the literal a written character extends. It holds at least the
// opening quote mark.
type Escape_Destination []byte

// Escape_Destination_Invariants bounds the literal that one more character extends.
func Escape_Destination_Invariants(
	value Escape_Destination, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), ESCAPE_DESTINATION_SIZE_MINIMUM,
			ESCAPE_DESTINATION_SIZE_MAXIMUM,
		).
		Ensure()
}

// ASCII_Quote_Destination is the open literal before an ASCII run. A two-byte value is
// absent because only non-ASCII processing can precede a later run.
type ASCII_Quote_Destination []byte

// ASCII_Quote_Destination_Invariants bounds each reachable ASCII-run boundary.
func ASCII_Quote_Destination_Invariants(
	value ASCII_Quote_Destination, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Holed_Int(
			len(value), ASCII_QUOTE_DESTINATION_SIZE_MINIMUM,
			ASCII_QUOTE_DESTINATION_SIZE_MAXIMUM,
			ASCII_QUOTE_DESTINATION_SIZE_HOLE, ASCII_QUOTE_DESTINATION_SIZE_HOLE,
			ASCII_QUOTE_DESTINATION_SIZE_HOLE, ASCII_QUOTE_DESTINATION_SIZE_HOLE,
		).
		Ensure()
}

// Open_Quoted_Buffer is a quoted literal after its body and before its closing quote.
type Open_Quoted_Buffer []byte

// Open_Quoted_Buffer_Invariants bounds every completed open literal body.
func Open_Quoted_Buffer_Invariants(value Open_Quoted_Buffer, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), OPEN_QUOTED_BUFFER_SIZE_MINIMUM,
			OPEN_QUOTED_BUFFER_SIZE_MAXIMUM,
		).
		Ensure()
}

// Escape_Sequence_Buffer is the literal that holds one more escape sequence.
type Escape_Sequence_Buffer []byte

// Escape_Sequence_Buffer_Invariants bounds the literal that gained one escape.
func Escape_Sequence_Buffer_Invariants(
	value Escape_Sequence_Buffer, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), ESCAPE_SEQUENCE_BUFFER_SIZE_MINIMUM,
			ESCAPED_BUFFER_SIZE_MAXIMUM,
		).
		Ensure()
}

// Hexadecimal_Buffer is the literal that holds one more digit escape.
type Hexadecimal_Buffer []byte

// Hexadecimal_Buffer_Invariants bounds the literal that gained the digits.
func Hexadecimal_Buffer_Invariants(
	value Hexadecimal_Buffer, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), HEXADECIMAL_BUFFER_SIZE_MINIMUM,
			ESCAPED_BUFFER_SIZE_MAXIMUM,
		).
		Ensure()
}

// Escape_Letter_Buffer is the literal that holds an escape letter and awaits the
// digits of that escape.
type Escape_Letter_Buffer []byte

// Escape_Letter_Buffer_Invariants bounds the literal that the digits extend.
func Escape_Letter_Buffer_Invariants(
	value Escape_Letter_Buffer, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), ESCAPE_LETTER_BUFFER_SIZE_MINIMUM,
			ESCAPE_LETTER_BUFFER_SIZE_MAXIMUM,
		).
		Ensure()
}

// Escaped_Buffer is the literal that holds one more written character.
type Escaped_Buffer []byte

// Escaped_Buffer_Invariants bounds the literal that gained one character.
func Escaped_Buffer_Invariants(value Escaped_Buffer, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), ESCAPED_BUFFER_SIZE_MINIMUM, ESCAPED_BUFFER_SIZE_MAXIMUM,
		).
		Ensure()
}

// Escape_Sequence_Text is the written form of one character that needs an escape.
type Escape_Sequence_Text string

// Escape_Sequence_Text_Invariants bounds one escape sequence.
func Escape_Sequence_Text_Invariants(
	value Escape_Sequence_Text, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), ESCAPE_SEQUENCE_SIZE_MINIMUM, ESCAPE_SIZE_MAXIMUM).
		Ensure()
}

// Hexadecimal_Digits_Text is the digits that one escape sequence writes.
type Hexadecimal_Digits_Text string

// Hexadecimal_Digits_Text_Invariants states the three escape widths.
func Hexadecimal_Digits_Text_Invariants(
	value Hexadecimal_Digits_Text, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_3_Int(
			len(value), HEXADECIMAL_DIGIT_COUNT, SHORT_UNICODE_DIGIT_COUNT,
			LONG_UNICODE_DIGIT_COUNT,
		).
		Ensure()
}

// Boolean_Text is the written form of a Boolean.
type Boolean_Text string

// Boolean_Text_Invariants states the two Boolean words.
func Boolean_Text_Invariants(value Boolean_Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Int(len(value), BOOLEAN_TEXT_SIZE_TRUE, BOOLEAN_TEXT_SIZE_FALSE).
		Ensure()
}

// Digit_Text is the written form of an unsigned number.
type Digit_Text string

// Digit_Text_Invariants bounds the digits of an unsigned number.
func Digit_Text_Invariants(value Digit_Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), DIGIT_TEXT_SIZE_MINIMUM, DIGIT_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Integer_Text is the written form of a signed number.
type Integer_Text string

// Integer_Text_Invariants bounds the sign and the digits of a signed number.
func Integer_Text_Invariants(value Integer_Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), INTEGER_TEXT_SIZE_MINIMUM, INTEGER_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Decimal_Text is the written form of a machine integer in base ten.
type Decimal_Text string

// Decimal_Text_Invariants bounds a machine integer in base ten.
func Decimal_Text_Invariants(value Decimal_Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), DECIMAL_TEXT_SIZE_MINIMUM, DECIMAL_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Fraction_Text is the digits that follow the decimal point.
type Fraction_Text string

// Fraction_Text_Invariants bounds the fraction digits that the reader keeps.
func Fraction_Text_Invariants(value Fraction_Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, FRACTION_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Fraction_Units is a fraction in fixed-point units.
type Fraction_Units uint64

// Fraction_Units_Invariants bounds a fraction to one whole, which the rounding of a
// fraction of nines reaches.
func Fraction_Units_Invariants(value Fraction_Units, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), FRACTION_UNITS_MINIMUM, FRACTION_UNITS_MAXIMUM).
		Ensure()
}

// Fixed_Point_Buffer is a buffer that holds one more fixed-point number.
type Fixed_Point_Buffer []byte

// Fixed_Point_Buffer_Invariants bounds a buffer and the number it gained.
func Fixed_Point_Buffer_Invariants(value Fixed_Point_Buffer, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), FIXED_POINT_TEXT_SIZE_MINIMUM, FIXED_POINT_BUFFER_SIZE_MAXIMUM,
		).
		Ensure()
}

// Buffer is the byte slice that an append conversion extends.
type Buffer []byte

// Buffer_Invariants bounds the buffer an append conversion takes.
func Buffer_Invariants(value Buffer, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), BUFFER_SIZE_MINIMUM, BUFFER_SIZE_MAXIMUM).
		Ensure()
}

// Boolean_Buffer is a buffer that holds one more Boolean word.
type Boolean_Buffer []byte

// Boolean_Buffer_Invariants bounds a buffer and the Boolean word it gained.
func Boolean_Buffer_Invariants(value Boolean_Buffer, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), BOOLEAN_TEXT_SIZE_TRUE, BOOLEAN_BUFFER_SIZE_MAXIMUM).
		Ensure()
}

// Integer_Buffer is a buffer that holds one more signed number.
type Integer_Buffer []byte

// Integer_Buffer_Invariants bounds a buffer and the number it gained.
func Integer_Buffer_Invariants(value Integer_Buffer, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), INTEGER_TEXT_SIZE_MINIMUM, INTEGER_BUFFER_SIZE_MAXIMUM).
		Ensure()
}

// Digit_Buffer is a buffer that holds one more unsigned number.
type Digit_Buffer []byte

// Digit_Buffer_Invariants bounds a buffer and the number it gained.
func Digit_Buffer_Invariants(value Digit_Buffer, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), DIGIT_TEXT_SIZE_MINIMUM, DIGIT_BUFFER_SIZE_MAXIMUM).
		Ensure()
}

// Quoted_Buffer is a buffer that holds one more string literal.
type Quoted_Buffer []byte

// Quoted_Buffer_Invariants bounds a buffer and the literal it gained.
func Quoted_Buffer_Invariants(value Quoted_Buffer, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), QUOTED_TEXT_SIZE_MINIMUM, QUOTED_BUFFER_SIZE_MAXIMUM).
		Ensure()
}

// Character_Buffer is a buffer that holds one more character literal.
type Character_Buffer []byte

// Character_Buffer_Invariants bounds a buffer and the literal it gained.
func Character_Buffer_Invariants(value Character_Buffer, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), CHARACTER_TEXT_SIZE_MINIMUM, CHARACTER_BUFFER_SIZE_MAXIMUM,
		).
		Ensure()
}

// Boolean is one true or false value.
type Boolean bool

// Boolean_Invariants records both Boolean states.
func Boolean_Invariants(value Boolean, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "A Boolean value is true.").
		Ensure()
}

// Base is the radix of a conversion.
type Base int

// Base_Invariants bounds a radix to the range the digit symbols cover.
func Base_Invariants(value Base, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), BASE_MINIMUM, BASE_MAXIMUM).
		Ensure()
}

// Implied_Base is a radix that zero defers to the prefix of the text.
type Implied_Base int

// Implied_Base_Invariants bounds a radix, admitting zero and excluding one.
func Implied_Base_Invariants(value Implied_Base, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Bit_Size_Invariants(value Bit_Size, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), BIT_SIZE_MINIMUM, BIT_SIZE_MAXIMUM).
		Ensure()
}

// Signed_Integer is a signed 64-bit value.
type Signed_Integer int64

// Signed_Integer_Invariants states the complete signed 64-bit domain.
func Signed_Integer_Invariants(value Signed_Integer, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), INTEGER_64_MINIMUM, INTEGER_64_MAXIMUM).
		Ensure()
}

// Unsigned_Integer is an unsigned 64-bit value.
type Unsigned_Integer uint64

// Unsigned_Integer_Invariants states the complete unsigned 64-bit domain.
func Unsigned_Integer_Invariants(value Unsigned_Integer, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), UNSIGNED_64_MINIMUM, UNSIGNED_64_MAXIMUM).
		Ensure()
}

// Signed_Magnitude is the absolute value of a signed 64-bit integer.
type Signed_Magnitude uint64

// Signed_Magnitude_Invariants bounds a magnitude through the smallest signed value.
func Signed_Magnitude_Invariants(value Signed_Magnitude, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(
			uint64(value), SIGNED_MAGNITUDE_MINIMUM, SIGNED_MAGNITUDE_MAXIMUM,
		).
		Ensure()
}

// Machine_Integer is a signed value of the platform integer width.
type Machine_Integer int

// Machine_Integer_Invariants states the complete platform integer domain.
//
//go:nosplit
func Machine_Integer_Invariants(value Machine_Integer, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), MACHINE_INTEGER_MINIMUM, MACHINE_INTEGER_MAXIMUM).
		Ensure()
}

// Character is one rune, which the caller of a quote conversion can give any value.
type Character rune

// Character_Invariants states the complete rune storage domain.
func Character_Invariants(value Character, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int32(int32(value), CHARACTER_MINIMUM, CHARACTER_MAXIMUM).
		Ensure()
}

// Latin_1_Character is one character of the first two Unicode blocks, which the
// printable question answers without a table.
type Latin_1_Character rune

// Latin_1_Character_Invariants bounds a character to the Latin-1 block.
func Latin_1_Character_Invariants(value Latin_1_Character, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int32(int32(value), CODE_POINT_MINIMUM, int32(LATIN_1_MAXIMUM)).
		Ensure()
}

// Code_Point is one Unicode code point.
type Code_Point rune

// Code_Point_Invariants bounds a value to the code point range.
func Code_Point_Invariants(value Code_Point, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int32(int32(value), CODE_POINT_MINIMUM, CODE_POINT_MAXIMUM).
		Ensure()
}

// Byte_Point is a code point that one byte of an octal escape names.
type Byte_Point rune

// Byte_Point_Invariants bounds a value to the code points of one byte.
func Byte_Point_Invariants(value Byte_Point, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int32(int32(value), CODE_POINT_MINIMUM, BYTE_POINT_MAXIMUM).
		Ensure()
}

// Text_Byte is one byte of text.
type Text_Byte uint8

// Text_Byte_Invariants states the complete byte domain.
func Text_Byte_Invariants(value Text_Byte, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), TEXT_BYTE_MINIMUM, TEXT_BYTE_MAXIMUM).
		Ensure()
}

// Octal_Digit_Byte is one octal digit symbol.
type Octal_Digit_Byte uint8

// Octal_Digit_Byte_Invariants bounds a symbol to the octal digits.
func Octal_Digit_Byte_Invariants(value Octal_Digit_Byte, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), OCTAL_DIGIT_MINIMUM, OCTAL_DIGIT_MAXIMUM).
		Ensure()
}

// Folded_Byte is a byte that a lowercase fold gives.
type Folded_Byte uint8

// Folded_Byte_Invariants bounds a folded byte, which holds the bit the fold sets.
func Folded_Byte_Invariants(value Folded_Byte, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), FOLDED_BYTE_MINIMUM, FOLDED_BYTE_MAXIMUM).
		Ensure()
}

// Digit_Value is the value of one digit symbol.
type Digit_Value uint8

// Digit_Value_Invariants bounds a digit to the largest radix.
func Digit_Value_Invariants(value Digit_Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), DIGIT_VALUE_MINIMUM, DIGIT_VALUE_MAXIMUM).
		Ensure()
}

// Hexadecimal_Value is the value of one hexadecimal digit.
type Hexadecimal_Value uint8

// Hexadecimal_Value_Invariants bounds a digit to base sixteen.
func Hexadecimal_Value_Invariants(value Hexadecimal_Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), HEXADECIMAL_VALUE_MINIMUM, HEXADECIMAL_VALUE_MAXIMUM).
		Ensure()
}

// Quote_Mark is the quote character of a literal, zero meaning neither.
type Quote_Mark uint8

// Quote_Mark_Invariants states the three literal forms.
func Quote_Mark_Invariants(value Quote_Mark, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(uint8(value), QUOTE_MARK_NONE, QUOTE_MARK_DOUBLE, QUOTE_MARK_SINGLE).
		Ensure()
}

// Literal_Quote_Mark is the quote character that opens and closes a literal.
type Literal_Quote_Mark uint8

// Literal_Quote_Mark_Invariants states the two literal forms.
func Literal_Quote_Mark_Invariants(value Literal_Quote_Mark, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(uint8(value), QUOTE_MARK_DOUBLE, QUOTE_MARK_SINGLE).
		Ensure()
}

// Digit_Count is the digit count that one escape sequence reads or writes.
type Digit_Count int

// Digit_Count_Invariants states the three escape widths.
func Digit_Count_Invariants(value Digit_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Int(
			int(value), HEXADECIMAL_DIGIT_COUNT, SHORT_UNICODE_DIGIT_COUNT,
			LONG_UNICODE_DIGIT_COUNT,
		).
		Ensure()
}

// Quoted_Capacity is the reserved storage for one quoted text.
type Quoted_Capacity int

// Quoted_Capacity_Invariants bounds storage from no body through all escaped bytes.
func Quoted_Capacity_Invariants(value Quoted_Capacity, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Holed_Int(
			int(value), TEXT_SIZE_MINIMUM, QUOTED_TEXT_SIZE_MAXIMUM,
			QUOTED_TEXT_SIZE_MINIMUM, QUOTED_TEXT_SIZE_MINIMUM,
			QUOTED_TEXT_SIZE_MINIMUM, QUOTED_TEXT_SIZE_MINIMUM,
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

// Format_Boolean writes true or false.
func Format_Boolean(value Boolean) (text Boolean_Text) {
	defer func() {
		Boolean_Text_Invariants(text, "format_boolean.text")
	}()
	Boolean_Invariants(value, "format_boolean.value")
	if value {
		return "true"
	}
	return "false"
}

// Append_Boolean writes true or false into the destination.
func Append_Boolean(destination Buffer, value Boolean) (extended Boolean_Buffer) {
	defer func() {
		Boolean_Buffer_Invariants(extended, "append_boolean.extended")
	}()
	Buffer_Invariants(destination, "append_boolean.destination")
	Boolean_Invariants(value, "append_boolean.value")
	return append(Boolean_Buffer(destination), []byte(Format_Boolean(value))...)
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

// Format_Unsigned_Integer writes an unsigned value in the given radix.
func Format_Unsigned_Integer(value Unsigned_Integer, base Base) (text Digit_Text) {
	defer func() {
		Digit_Text_Invariants(text, "format_unsigned_integer.text")
	}()
	Unsigned_Integer_Invariants(value, "format_unsigned_integer.value")
	Base_Invariants(base, "format_unsigned_integer.base")
	return digits_of(value, base)
}

// Format_Integer writes a signed value in the given radix, the sign leading the digits.
func Format_Integer(value Signed_Integer, base Base) (text Integer_Text) {
	defer func() {
		Integer_Text_Invariants(text, "format_integer.text")
	}()
	Signed_Integer_Invariants(value, "format_integer.value")
	Base_Invariants(base, "format_integer.base")
	magnitude := uint64(value)
	if value < 0 {
		magnitude = -magnitude
	}
	if base == DECIMAL_BASE {
		if int64(value) == INTEGER_64_MINIMUM {
			return INTEGER_64_MINIMUM_DECIMAL_TEXT
		}
		if int64(value) == INTEGER_64_MAXIMUM {
			return INTEGER_64_MAXIMUM_DECIMAL_TEXT
		}
		var decimal [DECIMAL_TEXT_SIZE_MAXIMUM]byte
		free_count := len(decimal)
		for magnitude>>DECIMAL_CHUNK_SHIFT != 0 {
			var chunk uint32
			magnitude, chunk = magnitude/DECIMAL_CHUNK_BASE,
				uint32(magnitude%DECIMAL_CHUNK_BASE)
			for range DECIMAL_CHUNK_PAIR_COUNT {
				var pair uint32
				chunk, pair = chunk/DECIMAL_PAIR_BASE,
					chunk%DECIMAL_PAIR_BASE*DECIMAL_PAIR_INDEX_SCALE
				free_count -= 2
				decimal[free_count], decimal[free_count+1] =
					DECIMAL_PAIRS[pair], DECIMAL_PAIRS[pair+1]
			}
			free_count--
			decimal[free_count] = DECIMAL_PAIRS[chunk*2+1]
			if magnitude == 0 {
				break
			}
		}
		if magnitude > 0 {
			chunk := uint32(magnitude)
			for chunk >= DECIMAL_PAIR_BASE {
				var pair uint32
				chunk, pair = chunk/DECIMAL_PAIR_BASE,
					chunk%DECIMAL_PAIR_BASE*DECIMAL_PAIR_INDEX_SCALE
				free_count -= 2
				decimal[free_count], decimal[free_count+1] =
					DECIMAL_PAIRS[pair], DECIMAL_PAIRS[pair+1]
			}
			free_count--
			pair := chunk * DECIMAL_PAIR_INDEX_SCALE
			decimal[free_count] = DECIMAL_PAIRS[pair+1]
			if chunk >= DECIMAL_BASE {
				free_count--
				decimal[free_count] = DECIMAL_PAIRS[pair]
			}
		}
		if value == 0 {
			free_count--
			decimal[free_count] = DIGIT_SYMBOLS[0]
		}
		if value < 0 {
			free_count--
			decimal[free_count] = '-'
		}
		return Integer_Text(decimal[free_count:])
	}
	return radix_integer_text(Signed_Magnitude(magnitude), Boolean(value < 0), base)
}

// Writes a signed magnitude in a non-decimal radix.
func radix_integer_text(
	magnitude Signed_Magnitude, negative Boolean, base Base,
) (text Integer_Text) {
	defer func() {
		Integer_Text_Invariants(text, "radix_integer_text.text")
	}()
	Signed_Magnitude_Invariants(magnitude, "radix_integer_text.magnitude")
	Boolean_Invariants(negative, "radix_integer_text.negative")
	Base_Invariants(base, "radix_integer_text.base")
	// The sign and digits share one scratch so converting the completed slice makes
	// the only owned string. Building the unsigned text first would allocate it too.
	var scratch [INTEGER_TEXT_SIZE_MAXIMUM]byte
	free_count := len(scratch)
	radix := uint64(base)
	for free_count > 0 {
		free_count--
		scratch[free_count] = DIGIT_SYMBOLS[uint64(magnitude)%radix]
		magnitude = Signed_Magnitude(uint64(magnitude) / radix)
		if magnitude == 0 {
			break
		}
	}
	if negative {
		free_count--
		scratch[free_count] = '-'
	}
	return Integer_Text(scratch[free_count:])
}

// Format_Decimal writes a machine integer in base ten.
func Format_Decimal(value Machine_Integer) (text Decimal_Text) {
	defer func() {
		Decimal_Text_Invariants(text, "format_decimal.text")
	}()
	Machine_Integer_Invariants(value, "format_decimal.value")
	return Decimal_Text(Format_Integer(Signed_Integer(value), DECIMAL_BASE))
}

// Append_Integer writes a signed value into the destination.
func Append_Integer(
	destination Buffer, value Signed_Integer, base Base,
) (extended Integer_Buffer) {
	defer func() {
		Integer_Buffer_Invariants(extended, "append_integer.extended")
	}()
	Buffer_Invariants(destination, "append_integer.destination")
	Signed_Integer_Invariants(value, "append_integer.value")
	Base_Invariants(base, "append_integer.base")
	return append(Integer_Buffer(destination), []byte(Format_Integer(value, base))...)
}

// Append_Unsigned_Integer writes an unsigned value into the destination.
func Append_Unsigned_Integer(
	destination Buffer, value Unsigned_Integer, base Base,
) (extended Digit_Buffer) {
	defer func() {
		Digit_Buffer_Invariants(extended, "append_unsigned_integer.extended")
	}()
	Buffer_Invariants(destination, "append_unsigned_integer.destination")
	Unsigned_Integer_Invariants(value, "append_unsigned_integer.value")
	Base_Invariants(base, "append_unsigned_integer.base")
	digits := Format_Unsigned_Integer(value, base)
	return append(Digit_Buffer(destination), []byte(digits)...)
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

// Format_Fixed_Point writes a fixed-point number with a set number of fraction digits.
// The fixedpoint package owns the rendering, thus this form only names it here.
func Format_Fixed_Point(
	value fixedpoint.Number, digits fixedpoint.Digit_Count,
) (text fixedpoint.Text) {
	defer func() {
		fixedpoint.Text_Invariants(text, "format_fixed_point.text")
	}()
	fixedpoint.Number_Invariants(value, "format_fixed_point.value")
	fixedpoint.Digit_Count_Invariants(digits, "format_fixed_point.digits")
	return fixedpoint.Format(value, digits)
}

// Append_Fixed_Point writes a fixed-point number into the destination.
func Append_Fixed_Point(
	destination Buffer, value fixedpoint.Number, digits fixedpoint.Digit_Count,
) (extended Fixed_Point_Buffer) {
	defer func() {
		Fixed_Point_Buffer_Invariants(extended, "append_fixed_point.extended")
	}()
	Buffer_Invariants(destination, "append_fixed_point.destination")
	fixedpoint.Number_Invariants(value, "append_fixed_point.value")
	fixedpoint.Digit_Count_Invariants(digits, "append_fixed_point.digits")
	text := Format_Fixed_Point(value, digits)
	return append(Fixed_Point_Buffer(destination), []byte(text)...)
}

// Quote returns a double-quoted Go string literal for the text.
func Quote(text Text) (quoted Quoted_Text) {
	defer func() {
		Quoted_Text_Invariants(quoted, "quote.quoted")
	}()
	Text_Invariants(text, "quote.text")
	return quoted_text(text, false, false)
}

// Quote_To_ASCII returns a double-quoted Go string literal that escapes every character
// above the ASCII range.
func Quote_To_ASCII(text Text) (quoted Quoted_Text) {
	defer func() {
		Quoted_Text_Invariants(quoted, "quote_to_ascii.quoted")
	}()
	Text_Invariants(text, "quote_to_ascii.text")
	return quoted_text(text, true, false)
}

// Quote_To_Graphic returns a double-quoted Go string literal that keeps every graphic
// character.
func Quote_To_Graphic(text Text) (quoted Quoted_Text) {
	defer func() {
		Quoted_Text_Invariants(quoted, "quote_to_graphic.quoted")
	}()
	Text_Invariants(text, "quote_to_graphic.text")
	return quoted_text(text, false, true)
}

// Append_Quote writes the double-quoted literal for the text into the destination.
func Append_Quote(destination Buffer, text Text) (extended Quoted_Buffer) {
	defer func() {
		Quoted_Buffer_Invariants(extended, "append_quote.extended")
	}()
	Buffer_Invariants(destination, "append_quote.destination")
	Text_Invariants(text, "append_quote.text")
	literal := quoted_text(text, false, false)
	return append(Quoted_Buffer(destination), []byte(literal)...)
}

// Append_Quote_To_ASCII writes the ASCII-only literal for the text into the
// destination.
func Append_Quote_To_ASCII(destination Buffer, text Text) (extended Quoted_Buffer) {
	defer func() {
		Quoted_Buffer_Invariants(extended, "append_quote_to_ascii.extended")
	}()
	Buffer_Invariants(destination, "append_quote_to_ascii.destination")
	Text_Invariants(text, "append_quote_to_ascii.text")
	literal := quoted_text(text, true, false)
	return append(Quoted_Buffer(destination), []byte(literal)...)
}

// Append_Quote_To_Graphic writes the graphic literal for the text into the destination.
func Append_Quote_To_Graphic(destination Buffer, text Text) (extended Quoted_Buffer) {
	defer func() {
		Quoted_Buffer_Invariants(extended, "append_quote_to_graphic.extended")
	}()
	Buffer_Invariants(destination, "append_quote_to_graphic.destination")
	Text_Invariants(text, "append_quote_to_graphic.text")
	literal := quoted_text(text, false, true)
	return append(Quoted_Buffer(destination), []byte(literal)...)
}

// Quote_Rune returns a single-quoted Go character literal for the character.
func Quote_Rune(value Character) (quoted Character_Text) {
	defer func() {
		Character_Text_Invariants(quoted, "quote_rune.quoted")
	}()
	Character_Invariants(value, "quote_rune.value")
	if value >= Character(ASCII_PRINT_MINIMUM) {
		if value <= Character(ASCII_PRINT_MAXIMUM) {
			if value != QUOTE_MARK_SINGLE {
				if value != '\\' {
					index := int(value-Character(ASCII_PRINT_MINIMUM)) *
						CHARACTER_TEXT_SIZE_MINIMUM
					end := index + CHARACTER_TEXT_SIZE_MINIMUM
					plain := PLAIN_ASCII_CHARACTER_TEXTS[index:end]
					return Character_Text(plain)
				}
			}
		}
	}
	return character_text(value, false, false)
}

// Quote_Rune_To_ASCII returns a single-quoted Go character literal that escapes every
// character above the ASCII range.
func Quote_Rune_To_ASCII(value Character) (quoted Character_Text) {
	defer func() {
		Character_Text_Invariants(quoted, "quote_rune_to_ascii.quoted")
	}()
	Character_Invariants(value, "quote_rune_to_ascii.value")
	return character_text(value, true, false)
}

// Quote_Rune_To_Graphic returns a single-quoted Go character literal that keeps every
// graphic character.
func Quote_Rune_To_Graphic(value Character) (quoted Character_Text) {
	defer func() {
		Character_Text_Invariants(quoted, "quote_rune_to_graphic.quoted")
	}()
	Character_Invariants(value, "quote_rune_to_graphic.value")
	return character_text(value, false, true)
}

// Append_Quote_Rune writes the character literal into the destination.
func Append_Quote_Rune(destination Buffer, value Character) (extended Character_Buffer) {
	defer func() {
		Character_Buffer_Invariants(extended, "append_quote_rune.extended")
	}()
	Buffer_Invariants(destination, "append_quote_rune.destination")
	Character_Invariants(value, "append_quote_rune.value")
	literal := character_text(value, false, false)
	return append(Character_Buffer(destination), []byte(literal)...)
}

// Append_Quote_Rune_To_ASCII writes the ASCII-only character literal into the
// destination.
func Append_Quote_Rune_To_ASCII(
	destination Buffer, value Character,
) (extended Character_Buffer) {
	defer func() {
		Character_Buffer_Invariants(extended, "append_quote_rune_to_ascii.extended")
	}()
	Buffer_Invariants(destination, "append_quote_rune_to_ascii.destination")
	Character_Invariants(value, "append_quote_rune_to_ascii.value")
	literal := character_text(value, true, false)
	return append(Character_Buffer(destination), []byte(literal)...)
}

// Append_Quote_Rune_To_Graphic writes the graphic character literal into the
// destination.
func Append_Quote_Rune_To_Graphic(
	destination Buffer, value Character,
) (extended Character_Buffer) {
	defer func() {
		Character_Buffer_Invariants(extended, "append_quote_rune_to_graphic.extended")
	}()
	Buffer_Invariants(destination, "append_quote_rune_to_graphic.destination")
	Character_Invariants(value, "append_quote_rune_to_graphic.value")
	literal := character_text(value, false, true)
	return append(Character_Buffer(destination), []byte(literal)...)
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

// Unquote reads a single-quoted, double-quoted, or backquoted Go literal and returns
// the value it holds.
func Unquote(text Text) (value Body_Text, err error) {
	defer func() {
		Body_Text_Invariants(value, "unquote.value")
	}()
	Text_Invariants(text, "unquote.text")
	if len(text) >= LITERAL_TEXT_SIZE_MINIMUM {
		if text[0] == QUOTE_MARK_DOUBLE {
			if text[len(text)-1] == QUOTE_MARK_DOUBLE {
				body := text[1 : len(text)-1]
				if strings.Index_Byte_Or_Non_ASCII(
					strings.Text(body), DOUBLE_QUOTED_PLAIN_EXCLUSIONS,
				) == strings.INDEX_ABSENT {
					return Body_Text(body), nil
				}
			}
		}
	}
	unquoted, rest, unquote_err := unquote_prefix(text, true)
	if len(rest) > 0 {
		return "", Error_Syntax
	}
	return Body_Text(unquoted), unquote_err
}

// Quoted_Prefix returns the literal at the start of the text, quote marks included.
func Quoted_Prefix(text Text) (quoted Prefix_Text, err error) {
	defer func() {
		Prefix_Text_Invariants(quoted, "quoted_prefix.quoted")
	}()
	Text_Invariants(text, "quoted_prefix.text")
	prefix, _, prefix_error := unquote_prefix(text, false)
	return Prefix_Text(prefix), prefix_error
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

// Writes the value into a scratch array from the last digit backward, thus the divide
// that finds each digit needs no second pass to reverse the text.
func digits_of(value Unsigned_Integer, base Base) (text Digit_Text) {
	defer func() {
		Digit_Text_Invariants(text, "digits_of.text")
	}()
	Unsigned_Integer_Invariants(value, "digits_of.value")
	Base_Invariants(base, "digits_of.base")
	var scratch [DIGIT_TEXT_SIZE_MAXIMUM]byte
	rest := uint64(value)
	radix := uint64(base)
	free_count := len(scratch)
	for free_count > 0 {
		free_count--
		scratch[free_count] = DIGIT_SYMBOLS[rest%radix]
		rest /= radix
		if rest == 0 {
			break
		}
	}
	return Digit_Text(scratch[free_count:])
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

// Writes consecutive ASCII characters into a literal without Unicode decoding.
func append_quoted_ascii(
	destination ASCII_Quote_Destination, text Text,
) (extended Open_Quoted_Buffer, rest Text) {
	defer func() {
		Open_Quoted_Buffer_Invariants(extended, "append_quoted_ascii.extended")
		Text_Invariants(rest, "append_quoted_ascii.rest")
	}()
	ASCII_Quote_Destination_Invariants(destination, "append_quoted_ascii.destination")
	Text_Invariants(text, "append_quoted_ascii.text")
	quote_mark := Literal_Quote_Mark(QUOTE_MARK_DOUBLE)
	extended = Open_Quoted_Buffer(destination)
	rest = text
	for len(rest) > 0 {
		if rest[0] >= byte(utf8.CHARACTER_SELF) {
			return extended, rest
		}
		plain_size := 0
		for plain_size < len(rest) {
			// The scalar path owns the final character and therefore still proves the
			// minimum and maximum destination boundaries of the shared escape writer.
			if len(rest)-plain_size == ESCAPE_SIZE_MINIMUM {
				break
			}
			character := rest[plain_size]
			if character < byte(ASCII_PRINT_MINIMUM) {
				break
			}
			if character > byte(ASCII_PRINT_MAXIMUM) {
				break
			}
			if character == byte(quote_mark) {
				break
			}
			if character == '\\' {
				break
			}
			plain_size++
		}
		if plain_size > 0 {
			extended = append(extended, rest[:plain_size]...)
			rest = rest[plain_size:]
			continue
		}
		if len(rest) == ESCAPE_SIZE_MINIMUM {
			extended = Open_Quoted_Buffer(append_escape(
				Escape_Destination(extended), Code_Point(rest[0]), quote_mark,
				false, false,
			))
			rest = rest[1:]
			continue
		}
		character := rest[0]
		escape_letter := byte(0)
		if int(character) < len(CONTROL_ESCAPE_LETTERS) {
			escape_letter = CONTROL_ESCAPE_LETTERS[character]
		}
		if escape_letter != 0 {
			extended = append(extended, '\\', escape_letter)
		} else if character == '\\' {
			extended = append(extended, `\\`...)
		} else if character == byte(quote_mark) {
			extended = append(extended, '\\', character)
		} else {
			extended = append(extended, '\\', 'x')
			extended = append(extended, HEXADECIMAL_SYMBOLS[character>>4])
			extended = append(extended, HEXADECIMAL_SYMBOLS[character&0xf])
		}
		rest = rest[1:]
	}
	return extended, rest
}

// Gives the initial allocation enough space when an early escape predicts growth.
func quoted_capacity(text Text) (capacity Quoted_Capacity) {
	defer func() {
		Quoted_Capacity_Invariants(capacity, "quoted_capacity.capacity")
	}()
	Text_Invariants(text, "quoted_capacity.text")
	capacity_size := QUOTED_CAPACITY_NUMERATOR * len(text) / QUOTED_CAPACITY_DENOMINATOR
	if len(text) == 0 {
		return Quoted_Capacity(capacity_size)
	}
	first := text[0]
	first_needs_escape := first < byte(ASCII_PRINT_MINIMUM)
	if first == QUOTE_MARK_DOUBLE {
		first_needs_escape = true
	}
	if first == '\\' {
		first_needs_escape = true
	}
	if first == ASCII_BYTE_MAXIMUM {
		first_needs_escape = true
	}
	if first_needs_escape {
		// A leading escape predicts enough growth that the maximum bounded size
		// costs less than repeated allocation and copying.
		capacity_size = len(text)*BYTE_ESCAPE_SIZE + QUOTED_TEXT_SIZE_MINIMUM
	}
	return Quoted_Capacity(capacity_size)
}

// Reports whether the complete text holds basic CJK unified ideographs.
func cjk_unified_ideographs(text Text) (complete Boolean) {
	defer func() {
		Boolean_Invariants(complete, "cjk_unified_ideographs.complete")
	}()
	Text_Invariants(text, "cjk_unified_ideographs.text")
	if len(text) == 0 {
		return false
	}
	if len(text)%utf8.CHARACTER_SIZE_THREE != 0 {
		return false
	}
	for position := 0; position < len(text); position += utf8.CHARACTER_SIZE_THREE {
		first := text[position]
		second := text[position+1]
		third := text[position+2]
		if first < utf8.FIRST_BYTE_THREE {
			return false
		}
		if first >= utf8.FIRST_BYTE_FOUR {
			return false
		}
		if second < utf8.CONTINUATION_MINIMUM {
			return false
		}
		if second > utf8.CONTINUATION_MAXIMUM {
			return false
		}
		if third < utf8.CONTINUATION_MINIMUM {
			return false
		}
		if third > utf8.CONTINUATION_MAXIMUM {
			return false
		}
		point := ucd.Character(first&utf8.FIRST_MASK_THREE)<<
			(2*utf8.CONTINUATION_PAYLOAD_BIT_COUNT) |
			ucd.Character(second&utf8.CONTINUATION_MASK)<<
				utf8.CONTINUATION_PAYLOAD_BIT_COUNT |
			ucd.Character(third&utf8.CONTINUATION_MASK)
		if point < ucd.CJK_UNIFIED_IDEOGRAPHS_MINIMUM {
			return false
		}
		if point > ucd.CJK_UNIFIED_IDEOGRAPHS_MAXIMUM {
			return false
		}
	}
	return true
}

// Writes the quoted form of the text.
func quoted_text(
	text Text, ascii_only Boolean, graphic_only Boolean,
) (quoted Quoted_Text) {
	defer func() { Quoted_Text_Invariants(quoted, "quoted_text.quoted") }()
	Text_Invariants(text, "quoted_text.text")
	Boolean_Invariants(ascii_only, "quoted_text.ascii_only")
	Boolean_Invariants(graphic_only, "quoted_text.graphic_only")
	quote_mark := Literal_Quote_Mark(QUOTE_MARK_DOUBLE)
	if !ascii_only {
		if cjk_unified_ideographs(text) {
			built := make([]byte, 0, len(text)+QUOTED_TEXT_SIZE_MINIMUM)
			built = append(built, byte(quote_mark))
			built = append(built, text...)
			built = append(built, byte(quote_mark))
			return Quoted_Text(built)
		}
	}
	built := make([]byte, 0, int(quoted_capacity(text)))
	built = append(built, byte(quote_mark))
	opened, rest := append_quoted_ascii(ASCII_Quote_Destination(built), text)
	built = []byte(opened)
	for len(rest) > 0 {
		if rest[0] < byte(utf8.CHARACTER_SELF) {
			extended, tail := append_quoted_ascii(
				ASCII_Quote_Destination(built), rest,
			)
			built, rest = []byte(extended), tail
			continue
		}
		value, width := utf8.Decode_Character_Text(utf8.Text(rest))
		if width == 1 {
			if value == utf8.REPLACEMENT_CHARACTER {
				// Invalid UTF-8 names the byte, not the replacement character.
				built = append(
					built, '\\', 'x', HEXADECIMAL_SYMBOLS[rest[0]>>4],
					HEXADECIMAL_SYMBOLS[rest[0]&0xf],
				)
				rest = rest[width:]
				continue
			}
		}
		if !ascii_only {
			printable := Boolean(false)
			if Code_Point(value) <= Code_Point(LATIN_1_MAXIMUM) {
				printable = Is_Print(Character(value))
			} else {
				// Above Latin-1, use the shared table directly.
				// This avoids two assertion boundaries in the loop.
				printable = Boolean(ucd.Is_Print(ucd.Character(value)))
			}
			if printable {
				built = append(built, rest[:width]...)
				rest = rest[width:]
				continue
			}
			if graphic_only {
				if Is_Graphic(Character(value)) {
					built = append(built, rest[:width]...)
					rest = rest[width:]
					continue
				}
			}
		}
		built = []byte(append_escape(
			built, Code_Point(value), quote_mark, ascii_only, graphic_only,
		))
		rest = rest[width:]
	}
	built = append(built, byte(quote_mark))
	return Quoted_Text(built)
}

// Writes the quoted form of one character, the replacement character standing in for a
// value that is not a code point.
func character_text(
	value Character, ascii_only Boolean, graphic_only Boolean,
) (quoted Character_Text) {
	defer func() {
		Character_Text_Invariants(quoted, "character_text.quoted")
	}()
	Character_Invariants(value, "character_text.value")
	Boolean_Invariants(ascii_only, "character_text.ascii_only")
	Boolean_Invariants(graphic_only, "character_text.graphic_only")
	quote_mark := Literal_Quote_Mark(QUOTE_MARK_SINGLE)
	point := Code_Point(value)
	if !utf8.Valid_Character(utf8.Character(value)) {
		point = UNICODE_REPLACEMENT
	}
	if point >= Code_Point(ASCII_PRINT_MINIMUM) {
		if point <= Code_Point(ASCII_PRINT_MAXIMUM) {
			switch point {
			case Code_Point(quote_mark), '\\':
			default:
				// Every quote policy keeps plain printable ASCII, so no Unicode or
				// escape question can change this three-byte literal.
				return Character_Text([]byte{
					byte(quote_mark), byte(point), byte(quote_mark),
				})
			}
		}
	}
	built := make([]byte, 0, CHARACTER_TEXT_SIZE_MAXIMUM)
	built = append(built, byte(quote_mark))
	built = []byte(append_escape(built, point, quote_mark, ascii_only, graphic_only))
	built = append(built, byte(quote_mark))
	return Character_Text(built)
}

// Writes one character, escaped when the form demands it.
func append_escape(
	destination Escape_Destination, value Code_Point, quote_mark Literal_Quote_Mark,
	ascii_only Boolean, graphic_only Boolean,
) (extended Escaped_Buffer) {
	defer func() {
		Escaped_Buffer_Invariants(extended, "append_escape.extended")
	}()
	Escape_Destination_Invariants(destination, "append_escape.destination")
	Code_Point_Invariants(value, "append_escape.value")
	Literal_Quote_Mark_Invariants(quote_mark, "append_escape.quote_mark")
	Boolean_Invariants(ascii_only, "append_escape.ascii_only")
	Boolean_Invariants(graphic_only, "append_escape.graphic_only")
	if value == Code_Point(quote_mark) {
		return Escaped_Buffer(append(destination, '\\', byte(value)))
	}
	if value == '\\' {
		return Escaped_Buffer(append(destination, '\\', '\\'))
	}
	if ascii_only {
		if value < Code_Point(utf8.CHARACTER_SELF) {
			if Is_Print(Character(value)) {
				return Escaped_Buffer(append(destination, byte(value)))
			}
		}
		return Escaped_Buffer(append_escape_sequence(destination, value))
	}
	if stays_unescaped(value, graphic_only) {
		if value < Code_Point(utf8.CHARACTER_SELF) {
			// One byte is its own UTF-8 sequence, thus the common character needs no
			// encoder and the literal gains it without an allocation.
			return Escaped_Buffer(append(destination, byte(value)))
		}
		// The utf8 library bounds the buffer it extends at its own sequence limit, and
		// a literal outgrows that limit. Thus the character goes into a scratch of one
		// sequence, and the literal takes the bytes of that scratch.
		var scratch [utf8.UTF_MAXIMUM]byte
		encoded := utf8.Append_Character(scratch[:0], utf8.Character(value))
		return Escaped_Buffer(append(destination, encoded...))
	}
	return Escaped_Buffer(append_escape_sequence(destination, value))
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

// Writes the escape sequence of one character.
func append_escape_sequence(
	destination Escape_Destination, value Code_Point,
) (extended Escape_Sequence_Buffer) {
	defer func() {
		Escape_Sequence_Buffer_Invariants(
			extended, "append_escape_sequence.extended",
		)
	}()
	Escape_Destination_Invariants(destination, "append_escape_sequence.destination")
	Code_Point_Invariants(value, "append_escape_sequence.value")
	switch value {
	case '\a':
		return Escape_Sequence_Buffer(append(destination, `\a`...))
	case '\b':
		return Escape_Sequence_Buffer(append(destination, `\b`...))
	case '\f':
		return Escape_Sequence_Buffer(append(destination, `\f`...))
	case '\n':
		return Escape_Sequence_Buffer(append(destination, `\n`...))
	case '\r':
		return Escape_Sequence_Buffer(append(destination, `\r`...))
	case '\t':
		return Escape_Sequence_Buffer(append(destination, `\t`...))
	case '\v':
		return Escape_Sequence_Buffer(append(destination, `\v`...))
	}
	switch {
	case value < ' ', value == '':
		return Escape_Sequence_Buffer(append_hexadecimal(
			Escape_Letter_Buffer(append(destination, `\x`...)), value,
			HEXADECIMAL_DIGIT_COUNT,
		))
	case value < 0x10000:
		return Escape_Sequence_Buffer(append_hexadecimal(
			Escape_Letter_Buffer(append(destination, `\u`...)), value,
			SHORT_UNICODE_DIGIT_COUNT,
		))
	}
	return Escape_Sequence_Buffer(append_hexadecimal(
		Escape_Letter_Buffer(append(destination, `\U`...)), value,
		LONG_UNICODE_DIGIT_COUNT,
	))
}

// Writes the character as a fixed count of hexadecimal digits, the most significant one
// first. The escape letter precedes them, thus the buffer is two bytes longer than the
// buffer that one whole escape extends.
func append_hexadecimal(
	destination Escape_Letter_Buffer, value Code_Point, digit_count Digit_Count,
) (extended Hexadecimal_Buffer) {
	defer func() {
		Hexadecimal_Buffer_Invariants(extended, "append_hexadecimal.extended")
	}()
	Escape_Letter_Buffer_Invariants(destination, "append_hexadecimal.destination")
	Code_Point_Invariants(value, "append_hexadecimal.value")
	Digit_Count_Invariants(digit_count, "append_hexadecimal.digit_count")
	extended = Hexadecimal_Buffer(destination)
	for shift := (int(digit_count) - 1) * 4; shift >= 0; shift -= 4 {
		extended = append(extended, HEXADECIMAL_SYMBOLS[(value>>uint(shift))&0xf])
	}
	return extended
}

// Reads the literal at the start of the text and returns it, the text that follows it,
// and the reason it is not a literal.
func unquote_prefix(text Text, unescape Boolean) (value Text, rest Body_Text, err error) {
	defer func() {
		Text_Invariants(value, "unquote_prefix.value")
		Body_Text_Invariants(rest, "unquote_prefix.rest")
	}()
	Text_Invariants(text, "unquote_prefix.text")
	Boolean_Invariants(unescape, "unquote_prefix.unescape")
	if len(text) < 2 {
		return "", Body_Text(text), Error_Syntax
	}
	switch text[0] {
	case '`':
		return unquote_backquoted(Literal_Text(text), unescape)
	case QUOTE_MARK_DOUBLE, QUOTE_MARK_SINGLE:
		return unquote_escaped(
			Literal_Text(text), Literal_Quote_Mark(text[0]), unescape,
		)
	}
	return "", Body_Text(text[2:]), Error_Syntax
}

// Reads a raw literal, which holds no escape and drops each carriage return.
func unquote_backquoted(
	text Literal_Text, unescape Boolean,
) (value Text, rest Body_Text, err error) {
	defer func() {
		Text_Invariants(value, "unquote_backquoted.value")
		Body_Text_Invariants(rest, "unquote_backquoted.rest")
	}()
	Literal_Text_Invariants(text, "unquote_backquoted.text")
	Boolean_Invariants(unescape, "unquote_backquoted.unescape")
	end_offset := int(strings.Index_Byte(strings.Text(text[1:]), '`'))
	if end_offset < 0 {
		return "", Body_Text(text[2:]), Error_Syntax
	}
	end_offset += 2
	if !unescape {
		return Text(text[:end_offset]), Body_Text(text[end_offset:]), nil
	}
	body := text[1 : end_offset-1]
	if strings.Index_Byte(strings.Text(body), '\r') < 0 {
		return Text(body), Body_Text(text[end_offset:]), nil
	}
	// The Go specification drops a carriage return from the value of a raw literal.
	kept := make([]byte, 0, len(body))
	for _, character := range []byte(body) {
		if character != '\r' {
			kept = append(kept, character)
		}
	}
	return Text(kept), Body_Text(text[end_offset:]), nil
}

// Reads an interpreted literal one character at a time.
func unquote_escaped(
	text Literal_Text, quote_mark Literal_Quote_Mark, unescape Boolean,
) (value Text, rest Body_Text, err error) {
	defer func() {
		Text_Invariants(value, "unquote_escaped.value")
		Body_Text_Invariants(rest, "unquote_escaped.rest")
	}()
	Literal_Text_Invariants(text, "unquote_escaped.text")
	Literal_Quote_Mark_Invariants(quote_mark, "unquote_escaped.quote_mark")
	Boolean_Invariants(unescape, "unquote_escaped.unescape")
	end_offset := int(strings.Index_Byte(
		strings.Text(text[1:]), strings.Byte(quote_mark),
	))
	if end_offset < 0 {
		return unquote_escaped_body(text, quote_mark, unescape)
	}
	end_offset += LITERAL_TEXT_SIZE_MINIMUM
	body := Body_Text(text[1 : end_offset-1])
	if strings.Index_Byte(strings.Text(body), '\\') >= 0 {
		return unquote_escaped_body(text, quote_mark, unescape)
	}
	if strings.Index_Byte(strings.Text(body), '\n') >= 0 {
		return unquote_escaped_body(text, quote_mark, unescape)
	}
	if !holds_one_value(body, quote_mark) {
		return unquote_escaped_body(text, quote_mark, unescape)
	}
	if unescape {
		return Text(body), Body_Text(text[end_offset:]), nil
	}
	return Text(text[:end_offset]), Body_Text(text[end_offset:]), nil
}

// Reports whether a literal body that holds no escape is the value of its literal. A
// double-quoted body must be valid text, and a single-quoted body must be one character.
func holds_one_value(body Body_Text, quote_mark Literal_Quote_Mark) (yes Boolean) {
	defer func() {
		Boolean_Invariants(yes, "holds_one_value.yes")
	}()
	Body_Text_Invariants(body, "holds_one_value.body")
	Literal_Quote_Mark_Invariants(quote_mark, "holds_one_value.quote_mark")
	if quote_mark == QUOTE_MARK_DOUBLE {
		return Boolean(utf8.Valid_Text(utf8.Text(body)))
	}
	character, size := utf8.Decode_Character_Text(utf8.Text(body))
	if int(size) != len(body) {
		return false
	}
	if character != utf8.REPLACEMENT_CHARACTER {
		return true
	}
	return Boolean(int(size) != utf8.CHARACTER_SIZE_MINIMUM)
}

// Reads an interpreted literal one character at a time, which an escape needs.
func unquote_escaped_body(
	text Literal_Text, quote_mark Literal_Quote_Mark, unescape Boolean,
) (value Text, rest Body_Text, err error) {
	defer func() {
		Text_Invariants(value, "unquote_escaped_body.value")
		Body_Text_Invariants(rest, "unquote_escaped_body.rest")
	}()
	Literal_Text_Invariants(text, "unquote_escaped_body.text")
	Literal_Quote_Mark_Invariants(quote_mark, "unquote_escaped_body.quote_mark")
	Boolean_Invariants(unescape, "unquote_escaped_body.unescape")
	body := Text(text[1:])
	kept := make([]byte, 0, len(text))
	for len(body) > 0 {
		first := body[0]
		if first == byte(quote_mark) {
			break
		}
		if first == '\n' {
			// A literal newline cannot occur inside an interpreted literal.
			return "", Body_Text(text[2:]), Error_Syntax
		}
		if first < byte(utf8.CHARACTER_SELF) {
			if first != '\\' {
				body = body[1:]
				if unescape {
					kept = append(kept, first)
				}
				if quote_mark == QUOTE_MARK_SINGLE {
					break
				}
				continue
			}
		}
		point, multibyte, tail, character_err := Unquote_Character(
			body, Quote_Mark(quote_mark),
		)
		if character_err != nil {
			return "", Body_Text(text[2:]), Error_Syntax
		}
		body = Text(tail)
		// A single byte stays one byte, so a hexadecimal escape can name a byte that is
		// not valid UTF-8.
		switch {
		case point < Code_Point(utf8.CHARACTER_SELF):
			kept = append(kept, byte(point))
		case !bool(multibyte):
			kept = append(kept, byte(point))
		default:
			kept = utf8.Append_Character(kept, utf8.Character(point))
		}
		if quote_mark == QUOTE_MARK_SINGLE {
			break
		}
	}
	if len(body) == 0 {
		return "", Body_Text(text[2:]), Error_Syntax
	}
	if body[0] != byte(quote_mark) {
		return "", Body_Text(text[2:]), Error_Syntax
	}
	body = body[1:]
	if !unescape {
		return Text(text[:len(text)-len(body)]), Body_Text(body), nil
	}
	return Text(kept), Body_Text(body), nil
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
