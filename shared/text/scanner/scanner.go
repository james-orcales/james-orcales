// Package scanner tokenizes bounded UTF-8 text without owned storage or implicit IO.
package scanner

import (
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/strconv"
	"local/james-orcales/shared/strings"
	"local/james-orcales/shared/unicode/ucd"
	"local/james-orcales/shared/unicode/utf8"
)

// SOURCE_SIZE_MINIMUM keeps empty input valid.
const SOURCE_SIZE_MINIMUM = strings.TEXT_SIZE_MINIMUM

// SOURCE_SIZE_MAXIMUM reuses repository text boundary.
const SOURCE_SIZE_MAXIMUM = strings.TEXT_SIZE_MAXIMUM

// SOURCE_SIZE_UNVALIDATED_MAXIMUM admits one rejected boundary witness.
const SOURCE_SIZE_UNVALIDATED_MAXIMUM = SOURCE_SIZE_MAXIMUM + utf8.CHARACTER_SIZE_MINIMUM

// FILENAME_SIZE_MINIMUM keeps unnamed input valid.
const FILENAME_SIZE_MINIMUM = strings.TEXT_SIZE_MINIMUM

// FILENAME_SIZE_MAXIMUM reuses repository text boundary.
const FILENAME_SIZE_MAXIMUM = strings.TEXT_SIZE_MAXIMUM

// OFFSET_MINIMUM is first source byte.
const OFFSET_MINIMUM = strings.TEXT_SIZE_MINIMUM

// OFFSET_MAXIMUM includes byte boundary after maximum source.
const OFFSET_MAXIMUM = SOURCE_SIZE_MAXIMUM

// LINE_INVALID invalidates token start after initialization and character reads.
const LINE_INVALID = strings.TEXT_SIZE_MINIMUM

// LINE_MINIMUM is first source line.
const LINE_MINIMUM = LINE_INVALID + utf8.CHARACTER_SIZE_MINIMUM

// LINE_MAXIMUM includes empty line after source made only of line feeds.
const LINE_MAXIMUM = SOURCE_SIZE_MAXIMUM + LINE_MINIMUM

// COLUMN_INVALID accompanies invalid token position.
const COLUMN_INVALID = strings.TEXT_SIZE_MINIMUM

// COLUMN_MINIMUM is first character column.
const COLUMN_MINIMUM = COLUMN_INVALID + utf8.CHARACTER_SIZE_MINIMUM

// COLUMN_MAXIMUM includes boundary after maximum one-byte line.
const COLUMN_MAXIMUM = SOURCE_SIZE_MAXIMUM + COLUMN_MINIMUM

// CHARACTER_INDEX_MINIMUM is first character of identifier.
const CHARACTER_INDEX_MINIMUM = strings.TEXT_SIZE_MINIMUM

// CHARACTER_INDEX_MAXIMUM admits one character per source byte.
const CHARACTER_INDEX_MAXIMUM = SOURCE_SIZE_MAXIMUM

// TOKEN_END_MARKER_COUNT is negative character result count.
const TOKEN_END_MARKER_COUNT = utf8.CHARACTER_SIZE_MINIMUM

// TOKEN_PREDEFINED_COUNT is negative lexical token count.
const TOKEN_PREDEFINED_COUNT = int(-TOKEN_COMMENT)

// CHARACTER_MINIMUM includes only end marker below Unicode range.
const CHARACTER_MINIMUM int32 = -TOKEN_END_MARKER_COUNT

// CHARACTER_MAXIMUM is final Unicode code point.
const CHARACTER_MAXIMUM int32 = utf8.DECODED_CHARACTER_MAXIMUM

// TOKEN_MINIMUM is final predefined token marker.
const TOKEN_MINIMUM int32 = int32(-TOKEN_PREDEFINED_COUNT)

// TOKEN_MAXIMUM is final Unicode code point.
const TOKEN_MAXIMUM int32 = utf8.DECODED_CHARACTER_MAXIMUM

// TOKEN_EOF marks exhausted source.
const TOKEN_EOF Token = -TOKEN_END_MARKER_COUNT

// TOKEN_IDENTIFIER marks identifier.
const TOKEN_IDENTIFIER = TOKEN_EOF - TOKEN_END_MARKER_COUNT

// TOKEN_INTEGER marks integer literal.
const TOKEN_INTEGER = TOKEN_IDENTIFIER - TOKEN_END_MARKER_COUNT

// TOKEN_FLOAT marks floating-point literal.
const TOKEN_FLOAT = TOKEN_INTEGER - TOKEN_END_MARKER_COUNT

// TOKEN_CHARACTER marks character literal.
const TOKEN_CHARACTER = TOKEN_FLOAT - TOKEN_END_MARKER_COUNT

// TOKEN_STRING marks interpreted string literal.
const TOKEN_STRING = TOKEN_CHARACTER - TOKEN_END_MARKER_COUNT

// TOKEN_RAW_STRING marks raw string literal.
const TOKEN_RAW_STRING = TOKEN_STRING - TOKEN_END_MARKER_COUNT

// TOKEN_COMMENT marks line or general comment.
const TOKEN_COMMENT = TOKEN_RAW_STRING - TOKEN_END_MARKER_COUNT

// SCAN_IDENTIFIERS enables identifier tokens.
const SCAN_IDENTIFIERS Mode = Mode(utf8.CHARACTER_SIZE_MINIMUM) << -TOKEN_IDENTIFIER

// SCAN_INTEGERS enables integer tokens without floating-point forms.
const SCAN_INTEGERS Mode = Mode(utf8.CHARACTER_SIZE_MINIMUM) << -TOKEN_INTEGER

// SCAN_FLOATS enables integer and floating-point tokens.
const SCAN_FLOATS Mode = Mode(utf8.CHARACTER_SIZE_MINIMUM) << -TOKEN_FLOAT

// SCAN_CHARACTERS enables character literal tokens.
const SCAN_CHARACTERS Mode = Mode(utf8.CHARACTER_SIZE_MINIMUM) << -TOKEN_CHARACTER

// SCAN_STRINGS enables interpreted string literal tokens.
const SCAN_STRINGS Mode = Mode(utf8.CHARACTER_SIZE_MINIMUM) << -TOKEN_STRING

// SCAN_RAW_STRINGS enables raw string literal tokens.
const SCAN_RAW_STRINGS Mode = Mode(utf8.CHARACTER_SIZE_MINIMUM) << -TOKEN_RAW_STRING

// SCAN_COMMENTS enables comment tokens.
const SCAN_COMMENTS Mode = Mode(utf8.CHARACTER_SIZE_MINIMUM) << -TOKEN_COMMENT

// SKIP_COMMENTS turns enabled comments into whitespace.
const SKIP_COMMENTS Mode = SCAN_COMMENTS << utf8.CHARACTER_SIZE_MINIMUM

// MODE_MINIMUM disables every compound token.
const MODE_MINIMUM Mode = Mode(strings.TEXT_SIZE_MINIMUM)

// MODE_MAXIMUM includes every defined mode bit.
const MODE_MAXIMUM = SCAN_IDENTIFIERS | SCAN_INTEGERS | SCAN_FLOATS |
	SCAN_CHARACTERS | SCAN_STRINGS | SCAN_RAW_STRINGS | SCAN_COMMENTS | SKIP_COMMENTS

// MODE_VALUE_MINIMUM gives invariant primitive domain.
const MODE_VALUE_MINIMUM uint = uint(strings.TEXT_SIZE_MINIMUM)

// MODE_BIT_COUNT includes unused zero and EOF shifts before lexical bits.
const MODE_BIT_COUNT = TOKEN_PREDEFINED_COUNT + utf8.CHARACTER_SIZE_TWO

// MODE_VALUE_MAXIMUM gives invariant primitive domain.
const MODE_VALUE_MAXIMUM uint = uint(utf8.CHARACTER_SIZE_MINIMUM)<<MODE_BIT_COUNT -
	utf8.CHARACTER_SIZE_MINIMUM

// GO_TOKENS recognizes Go literal forms and skips comments.
const GO_TOKENS = SCAN_IDENTIFIERS | SCAN_FLOATS | SCAN_CHARACTERS |
	SCAN_STRINGS | SCAN_RAW_STRINGS | SCAN_COMMENTS | SKIP_COMMENTS

// GO_WHITESPACE selects Go whitespace characters.
const GO_WHITESPACE Whitespace = Whitespace(utf8.CHARACTER_SIZE_MINIMUM)<<'\t' |
	Whitespace(utf8.CHARACTER_SIZE_MINIMUM)<<'\n' |
	Whitespace(utf8.CHARACTER_SIZE_MINIMUM)<<'\r' |
	Whitespace(utf8.CHARACTER_SIZE_MINIMUM)<<' '

// WHITESPACE_MINIMUM makes no character whitespace.
const WHITESPACE_MINIMUM Whitespace = Whitespace(strings.TEXT_SIZE_MINIMUM)

// WHITESPACE_CHARACTER_MAXIMUM follows standard Scanner contract.
const WHITESPACE_CHARACTER_MAXIMUM = ' '

// WHITESPACE_MAXIMUM holds every defined whitespace bit.
const WHITESPACE_MAXIMUM Whitespace = Whitespace(utf8.CHARACTER_SIZE_MINIMUM)<<
	(WHITESPACE_CHARACTER_MAXIMUM+utf8.CHARACTER_SIZE_MINIMUM) -
	utf8.CHARACTER_SIZE_MINIMUM

// STATUS_OK means requested operation completed.
const STATUS_OK = strings.TEXT_SIZE_MINIMUM

// STATUS_INPUT_INVALID means hostile input exceeded package boundary.
const STATUS_INPUT_INVALID = STATUS_OK + utf8.CHARACTER_SIZE_MINIMUM

// STATUS_OUTPUT_TOO_SMALL means caller storage cannot hold complete text.
const STATUS_OUTPUT_TOO_SMALL = STATUS_INPUT_INVALID + utf8.CHARACTER_SIZE_MINIMUM

// ERROR_NONE is zero Error value.
const ERROR_NONE = strings.TEXT_SIZE_MINIMUM

// ERROR_INVALID_UTF8 reports one invalid encoding byte.
const ERROR_INVALID_UTF8 = ERROR_NONE + utf8.CHARACTER_SIZE_MINIMUM

// ERROR_INVALID_NUL reports forbidden NUL character.
const ERROR_INVALID_NUL = ERROR_INVALID_UTF8 + utf8.CHARACTER_SIZE_MINIMUM

// ERROR_LITERAL_NOT_TERMINATED reports open character or string literal.
const ERROR_LITERAL_NOT_TERMINATED = ERROR_INVALID_NUL + utf8.CHARACTER_SIZE_MINIMUM

// ERROR_INVALID_CHARACTER_LITERAL reports other than one literal character.
const ERROR_INVALID_CHARACTER_LITERAL = ERROR_LITERAL_NOT_TERMINATED + utf8.CHARACTER_SIZE_MINIMUM

// ERROR_INVALID_ESCAPE reports malformed literal escape.
const ERROR_INVALID_ESCAPE = ERROR_INVALID_CHARACTER_LITERAL + utf8.CHARACTER_SIZE_MINIMUM

// ERROR_INVALID_RADIX_POINT reports fraction on binary or octal literal.
const ERROR_INVALID_RADIX_POINT = ERROR_INVALID_ESCAPE + utf8.CHARACTER_SIZE_MINIMUM

// ERROR_LITERAL_HAS_NO_DIGITS reports prefix without digits.
const ERROR_LITERAL_HAS_NO_DIGITS = ERROR_INVALID_RADIX_POINT + utf8.CHARACTER_SIZE_MINIMUM

// ERROR_EXPONENT_WRONG_MANTISSA reports exponent letter for wrong base.
const ERROR_EXPONENT_WRONG_MANTISSA = ERROR_LITERAL_HAS_NO_DIGITS + utf8.CHARACTER_SIZE_MINIMUM

// ERROR_EXPONENT_HAS_NO_DIGITS reports exponent without decimal digits.
const ERROR_EXPONENT_HAS_NO_DIGITS = ERROR_EXPONENT_WRONG_MANTISSA + utf8.CHARACTER_SIZE_MINIMUM

// ERROR_HEXADECIMAL_MANTISSA_HAS_NO_EXPONENT reports hexadecimal float without p exponent.
const ERROR_HEXADECIMAL_MANTISSA_HAS_NO_EXPONENT = ERROR_EXPONENT_HAS_NO_DIGITS +
	utf8.CHARACTER_SIZE_MINIMUM

// ERROR_INVALID_DIGIT reports digit outside literal base.
const ERROR_INVALID_DIGIT = ERROR_HEXADECIMAL_MANTISSA_HAS_NO_EXPONENT + utf8.CHARACTER_SIZE_MINIMUM

// ERROR_INVALID_SEPARATOR reports underscore outside successive digits.
const ERROR_INVALID_SEPARATOR = ERROR_INVALID_DIGIT + utf8.CHARACTER_SIZE_MINIMUM

// ERROR_COMMENT_NOT_TERMINATED reports open general comment.
const ERROR_COMMENT_NOT_TERMINATED = ERROR_INVALID_SEPARATOR + utf8.CHARACTER_SIZE_MINIMUM

// ERROR_CODE_MAXIMUM is final emitted scanner error.
const ERROR_CODE_MAXIMUM = ERROR_COMMENT_NOT_TERMINATED

// ERROR_CODE_VALUE_MINIMUM gives invariant primitive domain.
const ERROR_CODE_VALUE_MINIMUM uint8 = uint8(strings.TEXT_SIZE_MINIMUM)

// ERROR_CODE_COUNT is zero value plus emitted scanner error count.
const ERROR_CODE_COUNT uint8 = uint8(
	ERROR_CODE_MAXIMUM - ERROR_NONE + utf8.CHARACTER_SIZE_MINIMUM,
)

// ERROR_CODE_VALUE_MAXIMUM gives invariant primitive domain.
const ERROR_CODE_VALUE_MAXIMUM uint8 = ERROR_CODE_COUNT - utf8.CHARACTER_SIZE_MINIMUM

// REPORT_CODE_VALUE_MINIMUM is first emitted diagnostic code.
const REPORT_CODE_VALUE_MINIMUM uint8 = ERROR_CODE_VALUE_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// REPORT_CODE_VALUE_MAXIMUM is final emitted diagnostic code.
const REPORT_CODE_VALUE_MAXIMUM = ERROR_CODE_VALUE_MAXIMUM

// ERROR_LINE_MINIMUM is first line containing an offending character or boundary.
const ERROR_LINE_MINIMUM = LINE_MINIMUM

// ERROR_LINE_MAXIMUM is final line containing a byte in maximum source.
const ERROR_LINE_MAXIMUM = SOURCE_SIZE_MAXIMUM

// ERROR_COLUMN_MINIMUM is first column containing an offending character or boundary.
const ERROR_COLUMN_MINIMUM = COLUMN_MINIMUM

// ERROR_COLUMN_MAXIMUM includes an unterminated token report at maximum source boundary.
const ERROR_COLUMN_MAXIMUM = COLUMN_MAXIMUM

// ERROR_COUNT_MINIMUM means scanner reported no error.
const ERROR_COUNT_MINIMUM = strings.TEXT_SIZE_MINIMUM

// ERROR_END_REPORT_COUNT admits one structural report after final byte.
const ERROR_END_REPORT_COUNT = utf8.CHARACTER_SIZE_MINIMUM

// ERROR_COUNT_MAXIMUM admits one final unterminated-literal report beyond byte reports.
const ERROR_COUNT_MAXIMUM = SOURCE_SIZE_MAXIMUM + ERROR_END_REPORT_COUNT

// TOKEN_LABEL_SIZE_MAXIMUM is longest predefined token label.
const TOKEN_LABEL_SIZE_MAXIMUM = len("RawString")

// TOKEN_TEXT_SIZE_MINIMUM is shortest predefined or quoted token form.
const TOKEN_TEXT_SIZE_MINIMUM = len("EOF")

// TOKEN_TEXT_SIZE_MAXIMUM follows quoted character because it exceeds every label.
const TOKEN_TEXT_SIZE_MAXIMUM = strconv.CHARACTER_TEXT_SIZE_MAXIMUM

// POSITION_SEPARATOR_COUNT counts separators before line and column.
const POSITION_SEPARATOR_COUNT = len("::")

// POSITION_COMPONENT_COUNT counts decimal line and column.
const POSITION_COMPONENT_COUNT = POSITION_SEPARATOR_COUNT

// POSITION_COORDINATE_DIGIT_COUNT_MAXIMUM is exact while source bound stays between decimal
// kilo and ten kilo thresholds.
const POSITION_COORDINATE_DIGIT_COUNT_MAXIMUM = SOURCE_SIZE_MAXIMUM / bits.KILOBYTE_BYTES

// POSITION_TEXT_SIZE_MINIMUM is shortest nonempty caller filename.
const POSITION_TEXT_SIZE_MINIMUM = FILENAME_SIZE_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// POSITION_TEXT_SIZE_MAXIMUM holds maximum filename and decimal suffix.
const POSITION_TEXT_SIZE_MAXIMUM = FILENAME_SIZE_MAXIMUM + POSITION_SEPARATOR_COUNT +
	POSITION_COMPONENT_COUNT*POSITION_COORDINATE_DIGIT_COUNT_MAXIMUM

// DEFAULT_FILENAME names input without caller filename.
const DEFAULT_FILENAME = "<input>"

// Source_Unvalidated is hostile borrowed text before boundary check.
type Source_Unvalidated string

// Source_Unvalidated_Invariants bounds validation work itself.
func Source_Unvalidated_Invariants(value Source_Unvalidated, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SOURCE_SIZE_MINIMUM, SOURCE_SIZE_UNVALIDATED_MAXIMUM).
		Ensure()
}

// Source is validated borrowed scanner text.
type Source string

// Source_Invariants keeps every scan inside text boundary.
func Source_Invariants(value Source, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Text is borrowed source token view.
type Text string

// Text_Invariants keeps returned view inside source boundary.
func Text_Invariants(value Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Number_Literal is token source containing at least one separator.
type Number_Literal string

// Number_Literal_Invariants keeps separator validation on numeric token bounds.
func Number_Literal_Invariants(value Number_Literal, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), NUMBER_LITERAL_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Filename is bounded borrowed diagnostic name.
type Filename string

// Filename_Invariants keeps position formatting bounded.
func Filename_Invariants(value Filename, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), FILENAME_SIZE_MINIMUM, FILENAME_SIZE_MAXIMUM).
		Ensure()
}

// Offset is zero-based byte boundary.
type Offset int

// Offset_Invariants includes boundary after maximum source.
func Offset_Invariants(value Offset, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), OFFSET_MINIMUM, OFFSET_MAXIMUM).
		Ensure()
}

// Line is one-based source line or zero invalid token position.
type Line int

// Line_Invariants includes invalid state and maximum trailing line.
func Line_Invariants(value Line, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), LINE_INVALID, LINE_MAXIMUM).
		Ensure()
}

// Column is one-based character column or zero invalid token position.
type Column int

// Column_Invariants includes invalid state and maximum trailing boundary.
func Column_Invariants(value Column, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), COLUMN_INVALID, COLUMN_MAXIMUM).
		Ensure()
}

// Character_Index is zero-based identifier character index.
type Character_Index int

// Character_Index_Invariants follows one character per source byte.
func Character_Index_Invariants(value Character_Index, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), CHARACTER_INDEX_MINIMUM, CHARACTER_INDEX_MAXIMUM).
		Ensure()
}

// BASE_BINARY is smallest literal radix.
const BASE_BINARY = strconv.BASE_MINIMUM

// BASE_OCTAL is one binary digit for each octal bit.
const BASE_OCTAL = BASE_BINARY * BASE_BINARY * BASE_BINARY

// BASE_DECIMAL follows octal by one binary radix.
const BASE_DECIMAL = strconv.DECIMAL_BASE

// BASE_HEXADECIMAL is one binary digit beyond octal width.
const BASE_HEXADECIMAL = BASE_OCTAL * BASE_BINARY

// DIGIT_FLAGS_NONE records no digit and no separator.
const DIGIT_FLAGS_NONE = strings.TEXT_SIZE_MINIMUM

// DIGIT_FLAGS_DIGIT records at least one digit.
const DIGIT_FLAGS_DIGIT = DIGIT_FLAGS_NONE + utf8.CHARACTER_SIZE_MINIMUM

// DIGIT_FLAGS_SEPARATOR records at least one separator.
const DIGIT_FLAGS_SEPARATOR = DIGIT_FLAGS_DIGIT << utf8.CHARACTER_SIZE_MINIMUM

// DIGIT_FLAGS_BOTH records digit and separator presence.
const DIGIT_FLAGS_BOTH = DIGIT_FLAGS_DIGIT | DIGIT_FLAGS_SEPARATOR

// CHARACTER_COUNT_MINIMUM admits empty literal contents.
const CHARACTER_COUNT_MINIMUM = strings.TEXT_SIZE_MINIMUM

// QUOTED_LITERAL_DELIMITER_COUNT reserves opening and closing quote bytes.
const QUOTED_LITERAL_DELIMITER_COUNT = utf8.CHARACTER_SIZE_TWO

// CHARACTER_COUNT_MAXIMUM leaves both quoted literal delimiters inside source.
const CHARACTER_COUNT_MAXIMUM = SOURCE_SIZE_MAXIMUM - QUOTED_LITERAL_DELIMITER_COUNT

// CHARACTER_LITERAL_COUNT is required decoded character count inside character quotes.
const CHARACTER_LITERAL_COUNT = CHARACTER_COUNT_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// ESCAPE_DIGIT_COUNT_HEXADECIMAL_BYTE is one hexadecimal byte width.
const ESCAPE_DIGIT_COUNT_HEXADECIMAL_BYTE = BASE_BINARY

// ESCAPE_DIGIT_COUNT_OCTAL is standard octal escape width.
const ESCAPE_DIGIT_COUNT_OCTAL = ESCAPE_DIGIT_COUNT_HEXADECIMAL_BYTE + utf8.CHARACTER_SIZE_MINIMUM

// ESCAPE_DIGIT_COUNT_UNICODE_SHORT is two hexadecimal byte widths.
const ESCAPE_DIGIT_COUNT_UNICODE_SHORT = ESCAPE_DIGIT_COUNT_HEXADECIMAL_BYTE * BASE_BINARY

// ESCAPE_DIGIT_COUNT_UNICODE_LONG is two short Unicode widths.
const ESCAPE_DIGIT_COUNT_UNICODE_LONG = ESCAPE_DIGIT_COUNT_UNICODE_SHORT * BASE_BINARY

// ESCAPE_ACTION_INVALID rejects unknown escape introducer.
const ESCAPE_ACTION_INVALID = strings.TEXT_SIZE_MINIMUM

// ESCAPE_ACTION_DIRECT consumes one standard escaped character.
const ESCAPE_ACTION_DIRECT = ESCAPE_ACTION_INVALID + utf8.CHARACTER_SIZE_MINIMUM

// ESCAPE_ACTION_OCTAL consumes standard octal digits.
const ESCAPE_ACTION_OCTAL = ESCAPE_ACTION_DIRECT + utf8.CHARACTER_SIZE_MINIMUM

// ESCAPE_ACTION_HEXADECIMAL_BYTE consumes one hexadecimal byte.
const ESCAPE_ACTION_HEXADECIMAL_BYTE = ESCAPE_ACTION_OCTAL + utf8.CHARACTER_SIZE_MINIMUM

// ESCAPE_ACTION_UNICODE_SHORT consumes short Unicode digits.
const ESCAPE_ACTION_UNICODE_SHORT = ESCAPE_ACTION_HEXADECIMAL_BYTE +
	utf8.CHARACTER_SIZE_MINIMUM

// ESCAPE_ACTION_UNICODE_LONG consumes long Unicode digits.
const ESCAPE_ACTION_UNICODE_LONG = ESCAPE_ACTION_UNICODE_SHORT + utf8.CHARACTER_SIZE_MINIMUM

// DIGIT_VALUE_MINIMUM is zero digit value.
const DIGIT_VALUE_MINIMUM = strings.TEXT_SIZE_MINIMUM

// DIGIT_VALUE_MAXIMUM is first value outside hexadecimal radix.
const DIGIT_VALUE_MAXIMUM = BASE_HEXADECIMAL

// SEPARATOR_INDEX_ABSENT reports no invalid separator.
const SEPARATOR_INDEX_ABSENT = strings.TEXT_SIZE_MINIMUM

// SEPARATOR_INDEX_MAXIMUM is final byte in maximum source.
const SEPARATOR_INDEX_MAXIMUM = SOURCE_SIZE_MAXIMUM - utf8.CHARACTER_SIZE_MINIMUM

// NUMBER_LITERAL_SIZE_MINIMUM is shortest number containing a separator.
const NUMBER_LITERAL_SIZE_MINIMUM = len("0_")

// NUMBER_PREFIX_SIZE charges zero and radix selector bytes.
const NUMBER_PREFIX_SIZE = len("0x")

// QUOTE_CHARACTER opens and closes character literals.
const QUOTE_CHARACTER = '\''

// QUOTE_STRING opens and closes interpreted string literals.
const QUOTE_STRING = '"'

// COMMENT_KIND_BLOCK starts a general comment.
const COMMENT_KIND_BLOCK = '*'

// COMMENT_KIND_LINE starts a line comment.
const COMMENT_KIND_LINE = '/'

// DECIMAL_CHARACTER_MINIMUM is first decimal digit.
const DECIMAL_CHARACTER_MINIMUM = '0'

// DECIMAL_CHARACTER_MAXIMUM is final decimal digit.
const DECIMAL_CHARACTER_MAXIMUM = '9'

// INVALID_DIGIT_MINIMUM is smallest digit rejected by binary radix.
const INVALID_DIGIT_MINIMUM = BASE_BINARY

// INVALID_DIGIT_MAXIMUM is one beyond largest decimal digit value.
const INVALID_DIGIT_MAXIMUM = BASE_DECIMAL

// INVALID_DIGIT_ABSENT is outside every decimal digit value.
const INVALID_DIGIT_ABSENT = INVALID_DIGIT_MAXIMUM

// SLASH_TOKEN is individual slash before comment recognition.
const SLASH_TOKEN = '/'

// LOWER_CASE_OFFSET sets ASCII lowercase bit without a lookup table.
const LOWER_CASE_OFFSET = 'a' - 'A'

// EXPONENT_CHARACTER_DECIMAL introduces decimal exponent digits.
const EXPONENT_CHARACTER_DECIMAL = 'e'

// EXPONENT_CHARACTER_BINARY introduces binary exponent digits.
const EXPONENT_CHARACTER_BINARY = 'p'

// EXPONENT_ACTION_NONE leaves current character untouched.
const EXPONENT_ACTION_NONE = strings.TEXT_SIZE_MINIMUM

// EXPONENT_ACTION_SCAN consumes exponent digits.
const EXPONENT_ACTION_SCAN = EXPONENT_ACTION_NONE + utf8.CHARACTER_SIZE_MINIMUM

// EXPONENT_ACTION_WRONG_MANTISSA scans after reporting mismatched exponent base.
const EXPONENT_ACTION_WRONG_MANTISSA = EXPONENT_ACTION_SCAN + utf8.CHARACTER_SIZE_MINIMUM

// EXPONENT_ACTION_ABSENT reports hexadecimal fraction without binary exponent.
const EXPONENT_ACTION_ABSENT = EXPONENT_ACTION_WRONG_MANTISSA +
	utf8.CHARACTER_SIZE_MINIMUM

// Base selects binary, octal, decimal, or hexadecimal digits.
type Base int

// Base_Invariants lists scanner literal bases.
func Base_Invariants(value Base, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_4_Int(
			int(value), BASE_BINARY, BASE_OCTAL, BASE_DECIMAL, BASE_HEXADECIMAL,
		).
		Ensure()
}

// Escape_Base selects octal or hexadecimal escape digits.
type Escape_Base int

// Escape_Base_Invariants excludes numeric literal bases unused by escapes.
func Escape_Base_Invariants(value Escape_Base, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Int(int(value), BASE_OCTAL, BASE_HEXADECIMAL).
		Ensure()
}

// Quote selects one quoted literal delimiter.
type Quote rune

// Quote_Invariants lists both quoted literal delimiters.
func Quote_Invariants(value Quote, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Int32(int32(value), QUOTE_STRING, QUOTE_CHARACTER).
		Ensure()
}

// Decimal_Character is one decimal digit that begins a number.
type Decimal_Character rune

// Decimal_Character_Invariants excludes non-decimal number starts.
func Decimal_Character_Invariants(value Decimal_Character, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int32(int32(value), DECIMAL_CHARACTER_MINIMUM, DECIMAL_CHARACTER_MAXIMUM).
		Ensure()
}

// Invalid_Digit is absent or one digit value rejected by literal base.
type Invalid_Digit int

// Invalid_Digit_Invariants bounds deferred invalid-digit reporting.
func Invalid_Digit_Invariants(value Invalid_Digit, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), INVALID_DIGIT_MINIMUM, INVALID_DIGIT_MAXIMUM).
		Ensure()
}

// Number_Token is integer or floating-point numeric result.
type Number_Token rune

// Number_Token_Invariants lists both numeric token kinds.
func Number_Token_Invariants(value Number_Token, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Int32(int32(value), int32(TOKEN_FLOAT), int32(TOKEN_INTEGER)).
		Ensure()
}

// Exponent_Action selects scalar work after exponent classification.
type Exponent_Action uint8

// Exponent_Action_Invariants lists every exponent scan outcome.
func Exponent_Action_Invariants(value Exponent_Action, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value),
			EXPONENT_ACTION_NONE,
			EXPONENT_ACTION_SCAN,
			EXPONENT_ACTION_WRONG_MANTISSA,
			EXPONENT_ACTION_ABSENT,
		).
		Ensure()
}

// Slash_Token is slash character or recognized comment.
type Slash_Token rune

// Slash_Token_Invariants lists both slash scan outcomes.
func Slash_Token_Invariants(value Slash_Token, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Int32(int32(value), int32(TOKEN_COMMENT), SLASH_TOKEN).
		Ensure()
}

// Digit_Flags records digit and separator presence bits.
type Digit_Flags uint8

// Digit_Flags_Invariants lists both independent presence bits.
func Digit_Flags_Invariants(value Digit_Flags, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value),
			DIGIT_FLAGS_NONE,
			DIGIT_FLAGS_DIGIT,
			DIGIT_FLAGS_SEPARATOR,
			DIGIT_FLAGS_BOTH,
		).
		Ensure()
}

// Radix_Digit_Flags records whether legacy octal prefix contributes zero digit.
type Radix_Digit_Flags uint8

// Radix_Digit_Flags_Invariants excludes separator states before digit scanning.
func Radix_Digit_Flags_Invariants(
	value Radix_Digit_Flags, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_Uint8(uint8(value), DIGIT_FLAGS_NONE, DIGIT_FLAGS_DIGIT).
		Ensure()
}

// Character_Count counts decoded token characters.
type Character_Count int

// Character_Count_Invariants follows one character per source byte.
func Character_Count_Invariants(value Character_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), CHARACTER_COUNT_MINIMUM, CHARACTER_COUNT_MAXIMUM).
		Ensure()
}

// Escape_Digit_Count is fixed digit width after one escape prefix.
type Escape_Digit_Count int

// Escape_Digit_Count_Invariants lists octal and Unicode escape widths.
func Escape_Digit_Count_Invariants(
	value Escape_Digit_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_4_Int(
			int(value),
			ESCAPE_DIGIT_COUNT_HEXADECIMAL_BYTE,
			ESCAPE_DIGIT_COUNT_OCTAL,
			ESCAPE_DIGIT_COUNT_UNICODE_SHORT,
			ESCAPE_DIGIT_COUNT_UNICODE_LONG,
		).
		Ensure()
}

// Escape_Action selects scalar work after backslash decoding.
type Escape_Action uint8

// Escape_Action_Invariants bounds every escape classification.
func Escape_Action_Invariants(value Escape_Action, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), ESCAPE_ACTION_INVALID, ESCAPE_ACTION_UNICODE_LONG).
		Ensure()
}

// Digit_Value is hexadecimal value or first invalid sentinel.
type Digit_Value int

// Digit_Value_Invariants includes every hexadecimal value and sentinel.
func Digit_Value_Invariants(value Digit_Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), DIGIT_VALUE_MINIMUM, DIGIT_VALUE_MAXIMUM).
		Ensure()
}

// Separator_Index is invalid underscore byte or absence.
type Separator_Index int

// Separator_Index_Invariants includes absence and final source byte.
func Separator_Index_Invariants(value Separator_Index, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), SEPARATOR_INDEX_ABSENT, SEPARATOR_INDEX_MAXIMUM).
		Ensure()
}

// Character is one decoded source character or TOKEN_EOF.
type Character rune

// Character_Invariants excludes lexical token markers from character results.
func Character_Invariants(value Character, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int32(int32(value), CHARACTER_MINIMUM, CHARACTER_MAXIMUM).
		Ensure()
}

// Source_Offset is next undecoded source byte boundary.
type Source_Offset int

// Source_Offset_Invariants stays inside source boundary.
func Source_Offset_Invariants(value Source_Offset, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), OFFSET_MINIMUM, OFFSET_MAXIMUM).
		Ensure()
}

// Source_Line is line state after decoded look-ahead.
type Source_Line int

// Source_Line_Invariants includes zero storage and trailing line.
func Source_Line_Invariants(value Source_Line, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), LINE_INVALID, LINE_MAXIMUM).
		Ensure()
}

// Source_Column is column state after decoded look-ahead.
type Source_Column int

// Source_Column_Invariants includes zero storage and trailing boundary.
func Source_Column_Invariants(value Source_Column, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), COLUMN_INVALID, COLUMN_MAXIMUM).
		Ensure()
}

// Look_Ahead_Character is decoded source look-ahead or EOF.
type Look_Ahead_Character rune

// Look_Ahead_Character_Invariants includes zero storage and runtime characters.
func Look_Ahead_Character_Invariants(
	value Look_Ahead_Character, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int32(int32(value), CHARACTER_MINIMUM, CHARACTER_MAXIMUM).
		Ensure()
}

// Character_Offset is look-ahead source byte opening.
type Character_Offset int

// Character_Offset_Invariants stays inside source boundary.
func Character_Offset_Invariants(value Character_Offset, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), OFFSET_MINIMUM, OFFSET_MAXIMUM).
		Ensure()
}

// Character_Line is look-ahead source line.
type Character_Line int

// Character_Line_Invariants includes zero storage and trailing line.
func Character_Line_Invariants(value Character_Line, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), LINE_INVALID, LINE_MAXIMUM).
		Ensure()
}

// Character_Column is look-ahead character column.
type Character_Column int

// Character_Column_Invariants includes zero storage and trailing boundary.
func Character_Column_Invariants(value Character_Column, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), COLUMN_INVALID, COLUMN_MAXIMUM).
		Ensure()
}

// Token_Start is prior token source opening.
type Token_Start int

// Token_Start_Invariants stays inside source boundary.
func Token_Start_Invariants(value Token_Start, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), OFFSET_MINIMUM, OFFSET_MAXIMUM).
		Ensure()
}

// Token_End is prior token source closing.
type Token_End int

// Token_End_Invariants stays inside source boundary.
func Token_End_Invariants(value Token_End, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), OFFSET_MINIMUM, OFFSET_MAXIMUM).
		Ensure()
}

// Look_Ahead records available decoded character.
type Look_Ahead bool

// Look_Ahead_Invariants reaches absent and present look-ahead.
func Look_Ahead_Invariants(value Look_Ahead, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "Scanner look-ahead is present.").
		Ensure()
}

// Initialization distinguishes zero storage from initialized scanner.
type Initialization bool

// Initialization_Invariants reaches zero and initialized scanner state.
func Initialization_Invariants(value Initialization, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "Scanner state is initialized.").
		Ensure()
}

// Token is one predefined token or decoded source character.
type Token rune

// Token_Invariants spans predefined markers and Unicode characters.
func Token_Invariants(value Token, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int32(int32(value), TOKEN_MINIMUM, TOKEN_MAXIMUM).
		Ensure()
}

// Mode is bounded compound-token recognition bit set.
type Mode uint

// Mode_Invariants rejects undefined configuration bits.
func Mode_Invariants(value Mode, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint(uint(value), MODE_VALUE_MINIMUM, MODE_VALUE_MAXIMUM).
		Ensure()
}

// Whitespace selects source characters through U+0020.
type Whitespace uint64

// Whitespace_Invariants rejects bits outside standard Scanner contract.
func Whitespace_Invariants(value Whitespace, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), uint64(WHITESPACE_MINIMUM), uint64(WHITESPACE_MAXIMUM)).
		Ensure()
}

// Validation_Status reports hostile source admission.
type Validation_Status uint8

// Validation_Status_Invariants lists validation outcomes.
func Validation_Status_Invariants(value Validation_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID)).
		Ensure()
}

// Status reports caller-storage formatting outcome.
type Status uint8

// Status_Invariants lists formatter outcomes.
func Status_Invariants(value Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_OUTPUT_TOO_SMALL)).
		Ensure()
}

// Token_Output is bounded caller-owned token formatter storage.
type Token_Output []byte

// Token_Output_Invariants prevents token formatting into unrelated storage.
func Token_Output_Invariants(value Token_Output, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, TOKEN_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Position_Output is bounded caller-owned position formatter storage.
type Position_Output []byte

// Position_Output_Invariants prevents position formatting into unrelated storage.
func Position_Output_Invariants(value Position_Output, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, POSITION_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Token_Count is written token byte count.
type Token_Count int

// Token_Count_Invariants stays inside longest token form.
func Token_Count_Invariants(value Token_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), TOKEN_TEXT_SIZE_MINIMUM, TOKEN_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Position_Count is written position byte count.
type Position_Count int

// Position_Count_Invariants stays inside longest position form.
func Position_Count_Invariants(value Position_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), POSITION_TEXT_SIZE_MINIMUM, POSITION_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Error_Code is typed scanner diagnostic without owned text.
type Error_Code uint8

// Error_Code_Invariants includes zero report and every emitted code.
func Error_Code_Invariants(value Error_Code, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), ERROR_CODE_VALUE_MINIMUM, ERROR_CODE_VALUE_MAXIMUM).
		Ensure()
}

// Report_Code is one diagnostic code the scanner can emit.
type Report_Code uint8

// Report_Code_Invariants excludes zero no-report state.
func Report_Code_Invariants(value Report_Code, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), REPORT_CODE_VALUE_MINIMUM, REPORT_CODE_VALUE_MAXIMUM).
		Ensure()
}

// Lexical_Report_Code is one structural token diagnostic emitted after decoding.
type Lexical_Report_Code uint8

// Lexical_Report_Code_Invariants excludes decoder-only encoding diagnostics.
func Lexical_Report_Code_Invariants(
	value Lexical_Report_Code, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint8(
			uint8(value),
			uint8(ERROR_LITERAL_NOT_TERMINATED),
			uint8(ERROR_COMMENT_NOT_TERMINATED),
		).
		Ensure()
}

// Error_Line is one source line containing an error.
type Error_Line int

// Error_Line_Invariants excludes invalid and trailing empty line states.
func Error_Line_Invariants(value Error_Line, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), ERROR_LINE_MINIMUM, ERROR_LINE_MAXIMUM).
		Ensure()
}

// Error_Column is one source column containing an error.
type Error_Column int

// Error_Column_Invariants excludes invalid and trailing empty column states.
func Error_Column_Invariants(value Error_Column, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), ERROR_COLUMN_MINIMUM, ERROR_COLUMN_MAXIMUM).
		Ensure()
}

// Report_Line is decoded line before zero-state normalization.
type Report_Line int

// Report_Line_Invariants excludes trailing line without a character.
func Report_Line_Invariants(value Report_Line, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), LINE_INVALID, ERROR_LINE_MAXIMUM).
		Ensure()
}

// Report_Column is decoded character column used by reports.
type Report_Column int

// Report_Column_Invariants excludes absent character columns.
func Report_Column_Invariants(value Report_Column, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), ERROR_COLUMN_MINIMUM, ERROR_COLUMN_MAXIMUM).
		Ensure()
}

// Error_Position names an offending character or source boundary.
type Error_Position struct {
	// Filename stays borrowed so diagnostics own no text.
	Filename Filename
	// Offset counts source bytes from zero.
	Offset Offset
	// Line names an existing source line.
	Line Error_Line
	// Column names an existing source column.
	Column Error_Column
}

// Error_Position_Invariants composes reachable diagnostic coordinates.
func Error_Position_Invariants(value Error_Position, namespace invariant.Namespace) {
	Filename_Invariants(value.Filename, namespace)
	Offset_Invariants(value.Offset, namespace)
	Error_Line_Invariants(value.Line, namespace)
	Error_Column_Invariants(value.Column, namespace)
}

// Error_Count counts reports from bounded source.
type Error_Count int

// Error_Count_Invariants derives work from source boundary.
func Error_Count_Invariants(value Error_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), ERROR_COUNT_MINIMUM, ERROR_COUNT_MAXIMUM).
		Ensure()
}

// Boolean gives scanner decisions one coverage identity.
type Boolean bool

// Boolean_Invariants reaches both scanner decision results.
func Boolean_Invariants(value Boolean, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "A scanner decision is true.").
		Ensure()
}

// Position names source byte, line, and character column.
type Position struct {
	// Filename stays borrowed so diagnostics own no text.
	Filename Filename
	// Offset counts source bytes from zero.
	Offset Offset
	// Line counts source lines from one or remains invalid at zero.
	Line Line
	// Column counts decoded characters from one or remains invalid at zero.
	Column Column
}

// Position_Invariants composes coordinates common to valid and invalid positions.
func Position_Invariants(value Position, namespace invariant.Namespace) {
	Filename_Invariants(value.Filename, namespace)
	Offset_Invariants(value.Offset, namespace)
	Line_Invariants(value.Line, namespace)
	Column_Invariants(value.Column, namespace)
}

// Error carries scalar context needed to classify malformed input.
type Error struct {
	// Code identifies diagnostic without owned message.
	Code Error_Code
	// Character identifies offending decoded input or EOF.
	Character Character
	// Position identifies offending source boundary.
	Position Position
}

// Error_Invariants keeps callbacks on bounded value state.
func Error_Invariants(value Error, namespace invariant.Namespace) {
	Error_Code_Invariants(value.Code, namespace)
	Character_Invariants(value.Character, namespace)
	Position_Invariants(value.Position, namespace)
}

// Identifier_Function injects identifier policy without package storage.
type Identifier_Function func(character Character, index Character_Index) (accepted bool)

// Error_Function injects diagnostic handling without exposing mutable scanner state.
type Error_Function func(report Error)

// Scanner_Decoder holds only state changed by one consumed character.
type Scanner_Decoder struct {
	// Source stays borrowed so cursor owns no text.
	Source Source
	// Source_Offset prevents decoding same bytes twice.
	Source_Offset Source_Offset
	// Line survives look-ahead consumption.
	Line Source_Line
	// Column survives look-ahead consumption.
	Column Source_Column
	// Character_Offset preserves opening before decoder advances.
	Character_Offset Character_Offset
	// Character_Line preserves line before decoder advances.
	Character_Line Character_Line
	// Character_Column preserves column before decoder advances.
	Character_Column Character_Column
}

// Scanner_Decoder_Invariants keeps consumed state inside source boundaries.
func Scanner_Decoder_Invariants(value Scanner_Decoder, namespace invariant.Namespace) {
	Source_Invariants(value.Source, namespace)
	Source_Offset_Invariants(value.Source_Offset, namespace)
	Source_Line_Invariants(value.Line, namespace)
	Source_Column_Invariants(value.Column, namespace)
	Character_Offset_Invariants(value.Character_Offset, namespace)
	Character_Line_Invariants(value.Character_Line, namespace)
	Character_Column_Invariants(value.Character_Column, namespace)
}

// Scanner_Cursor adds only cache state needed by public look-ahead.
type Scanner_Cursor struct {
	// Decoder stays separate because token helpers consume known characters.
	Decoder Scanner_Decoder
	// Character avoids decoding same look-ahead twice.
	Character Look_Ahead_Character
	// Looked distinguishes cached NUL from absent look-ahead.
	Looked Look_Ahead
}

// Scanner_Cursor_Invariants composes decoder and public look-ahead state.
func Scanner_Cursor_Invariants(value Scanner_Cursor, namespace invariant.Namespace) {
	Scanner_Decoder_Invariants(value.Decoder, namespace)
	Look_Ahead_Character_Invariants(value.Character, namespace)
	Look_Ahead_Invariants(value.Looked, namespace)
}

// Scanner_Diagnostics holds policy and count changed by reports.
type Scanner_Diagnostics struct {
	// Error stays injected so decoder performs no IO.
	Error Error_Function
	// Error_Count stays separate so callback cannot mutate scanner.
	Error_Count Error_Count
	// Filename joins reports without giving callback scanner access.
	Filename Filename
}

// Scanner_Diagnostics_Invariants bounds report state without scanner access.
func Scanner_Diagnostics_Invariants(
	value Scanner_Diagnostics, namespace invariant.Namespace,
) {
	Error_Count_Invariants(value.Error_Count, namespace)
	Filename_Invariants(value.Filename, namespace)
}

// Scanner_Token_Bounds holds source boundaries changed during token scanning.
type Scanner_Token_Bounds struct {
	// Start survives nested literal scanning.
	Start Token_Start
	// End permits separator validation before public scan completes.
	End Token_End
}

// Scanner_Token_Bounds_Invariants keeps token boundaries inside source.
func Scanner_Token_Bounds_Invariants(
	value Scanner_Token_Bounds, namespace invariant.Namespace,
) {
	Token_Start_Invariants(value.Start, namespace)
	Token_End_Invariants(value.End, namespace)
}

// Scanner holds borrowed source and bounded scalar cursor state.
type Scanner struct {
	// Error injects caller diagnostic policy.
	Error Error_Function
	// Error_Count bounds accumulated malformed-input reports.
	Error_Count Error_Count
	// Mode selects compound token forms.
	Mode Mode
	// Whitespace selects skipped control characters.
	Whitespace Whitespace
	// Identifier injects custom identifier policy.
	Identifier Identifier_Function
	// Position holds most recent token start and caller filename.
	Position Position
	// Cursor separates decoder mutation from configuration.
	Cursor Scanner_Cursor
	// Token separates token mutation from decoder state.
	Token Scanner_Token_Bounds
	// Initialized distinguishes zero state from empty source.
	Initialized Initialization
}

// Scanner_Invariants bounds zero storage and initialized runtime state.
func Scanner_Invariants(value Scanner, namespace invariant.Namespace) {
	Error_Count_Invariants(value.Error_Count, namespace)
	Mode_Invariants(value.Mode, namespace)
	Whitespace_Invariants(value.Whitespace, namespace)
	Position_Invariants(value.Position, namespace)
	Scanner_Cursor_Invariants(value.Cursor, namespace)
	Scanner_Token_Bounds_Invariants(value.Token, namespace)
	Initialization_Invariants(value.Initialized, namespace)
}

// Source_Validate changes hostile text into bounded scanner source.
func Source_Validate(
	unvalidated Source_Unvalidated,
) (source Source, status Validation_Status) {
	defer func() {
		Source_Invariants(source, "Source_Validate.source")
		Validation_Status_Invariants(status, "Source_Validate.status")
	}()
	Source_Unvalidated_Invariants(unvalidated, "Source_Validate.unvalidated")
	if len(unvalidated) > SOURCE_SIZE_MAXIMUM {
		return "", Validation_Status(STATUS_INPUT_INVALID)
	}
	return Source(unvalidated), Validation_Status(STATUS_OK)
}

// Position_Valid reports whether position names a source line.
func Position_Valid(position Position) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "Position_Valid.valid") }()
	Position_Invariants(position, "Position_Valid.position")
	Line_Invariants(position.Line, "Position_Valid.line")
	Column_Invariants(position.Column, "Position_Valid.column")
	return Boolean(position.Line > LINE_INVALID)
}

// Scanner_Init resets scanner while retaining caller filename policy.
func Scanner_Init(subject *Scanner, source Source) {
	Scanner_Invariants(*subject, "Scanner_Init.subject.input")
	Source_Invariants(source, "Scanner_Init.source")
	Filename_Invariants(subject.Position.Filename, "Scanner_Init.filename")
	filename := subject.Position.Filename
	*subject = Scanner{
		Mode:       GO_TOKENS,
		Whitespace: GO_WHITESPACE,
		Position: Position{
			Filename: filename,
		},
		Cursor: Scanner_Cursor{
			Decoder: Scanner_Decoder{
				Source: source,
				Line:   LINE_MINIMUM,
			},
		},
		Initialized: true,
	}
}

// Scanner_Peek returns next character without advancing public cursor.
func Scanner_Peek(subject *Scanner) (character Character) {
	defer func() { Character_Invariants(character, "Scanner_Peek.character") }()
	Scanner_Invariants(*subject, "Scanner_Peek")
	diagnostics := Scanner_Diagnostics{
		Error:       subject.Error,
		Error_Count: subject.Error_Count,
		Filename:    subject.Position.Filename,
	}
	character = scanner_peek_unchecked(&subject.Cursor, &diagnostics)
	subject.Error_Count = diagnostics.Error_Count
	return character
}

// Scanner_Next consumes next character and invalidates token start.
func Scanner_Next(subject *Scanner) (character Character) {
	defer func() { Character_Invariants(character, "Scanner_Next.character") }()
	Scanner_Invariants(*subject, "Scanner_Next")
	subject.Position.Line = LINE_INVALID
	subject.Position.Column = COLUMN_INVALID
	diagnostics := Scanner_Diagnostics{
		Error:       subject.Error,
		Error_Count: subject.Error_Count,
		Filename:    subject.Position.Filename,
	}
	character = scanner_peek_unchecked(&subject.Cursor, &diagnostics)
	subject.Error_Count = diagnostics.Error_Count
	if character != Character(TOKEN_EOF) {
		subject.Cursor.Looked = false
	}
	return character
}

// Scanner_Scan consumes next configured token.
func Scanner_Scan(subject *Scanner) (token Token) {
	defer func() {
		Token_Invariants(token, "Scanner_Scan.token")
		Error_Count_Invariants(subject.Error_Count, "Scanner_Scan.error_count")
	}()
	Scanner_Invariants(*subject, "Scanner_Scan")
	Mode_Invariants(subject.Mode, "Scanner_Scan.mode")
	Whitespace_Invariants(subject.Whitespace, "Scanner_Scan.whitespace")
	diagnostics := Scanner_Diagnostics{
		Error:       subject.Error,
		Error_Count: subject.Error_Count,
		Filename:    subject.Position.Filename,
	}
	character := scanner_peek_unchecked(&subject.Cursor, &diagnostics)
	subject.Position.Line = LINE_INVALID
	subject.Position.Column = COLUMN_INVALID
	done := false
	for !done {
		for bool(scanner_whitespace_unchecked(subject.Whitespace, character)) {
			character = scanner_advance_unchecked(&subject.Cursor.Decoder, &diagnostics)
		}
		subject.Token.Start = Token_Start(subject.Cursor.Decoder.Character_Offset)
		subject.Position = Position{
			Filename: subject.Position.Filename,
			Offset:   Offset(subject.Cursor.Decoder.Character_Offset),
			Line:     Line(subject.Cursor.Decoder.Character_Line),
			Column:   Column(subject.Cursor.Decoder.Character_Column),
		}
		var skip Boolean
		token, character, skip = scan_token_unchecked(
			&subject.Cursor.Decoder,
			&diagnostics,
			subject.Token.Start,
			subject.Mode,
			subject.Identifier,
			character,
		)
		done = !bool(skip)
	}

	subject.Token.End = Token_End(subject.Cursor.Decoder.Character_Offset)
	subject.Cursor.Character = Look_Ahead_Character(character)
	subject.Error_Count = diagnostics.Error_Count
	return token
}

// Scanner_Position returns boundary immediately after prior token or character.
func Scanner_Position(subject *Scanner) (position Position) {
	defer func() { Position_Invariants(position, "Scanner_Position.position") }()
	Scanner_Invariants(*subject, "Scanner_Position")
	if bool(subject.Cursor.Looked) {
		return Position{
			Filename: subject.Position.Filename,
			Offset:   Offset(subject.Cursor.Decoder.Character_Offset),
			Line:     Line(subject.Cursor.Decoder.Character_Line),
			Column:   Column(subject.Cursor.Decoder.Character_Column),
		}
	}
	return Position{
		Filename: subject.Position.Filename,
		Offset:   Offset(subject.Cursor.Decoder.Source_Offset),
		Line:     Line(subject.Cursor.Decoder.Line),
		Column: Column(
			subject.Cursor.Decoder.Column + Source_Column(COLUMN_MINIMUM),
		),
	}
}

// Scanner_Token_Text returns source view for prior Scan.
func Scanner_Token_Text(subject *Scanner) (text Text) {
	defer func() { Text_Invariants(text, "Scanner_Token_Text.text") }()
	Scanner_Invariants(*subject, "Scanner_Token_Text")
	invariant.Always(
		Token_End(subject.Token.Start) <= subject.Token.End,
		"Scanner token start does not follow token end.",
	)
	invariant.Always(
		subject.Token.End <= Token_End(len(subject.Cursor.Decoder.Source)),
		"Scanner token end stays inside source.",
	)
	return Text(subject.Cursor.Decoder.Source[subject.Token.Start:subject.Token.End])
}

// Token_Append_Into writes printable token into caller storage.
func Token_Append_Into(
	destination Token_Output, token Token,
) (count Token_Count, status Status) {
	defer func() {
		Token_Count_Invariants(count, "Token_Append_Into.count")
		Status_Invariants(status, "Token_Append_Into.status")
	}()
	Token_Output_Invariants(destination, "Token_Append_Into.destination")
	Token_Invariants(token, "Token_Append_Into.token")
	label := ""
	switch token {
	case TOKEN_EOF:
		label = "EOF"
	case TOKEN_IDENTIFIER:
		label = "Ident"
	case TOKEN_INTEGER:
		label = "Int"
	case TOKEN_FLOAT:
		label = "Float"
	case TOKEN_CHARACTER:
		label = "Char"
	case TOKEN_STRING:
		label = "String"
	case TOKEN_RAW_STRING:
		label = "RawString"
	case TOKEN_COMMENT:
		label = "Comment"
	}
	if label != "" {
		if len(destination) < len(label) {
			return Token_Count(len(label)), Status(STATUS_OUTPUT_TOO_SMALL)
		}
		return Token_Count(copy(destination, label)), Status(STATUS_OK)
	}
	var storage [strconv.CHARACTER_TEXT_SIZE_MAXIMUM]byte
	required := strconv.Quote_Rune_Into(
		storage[:], strconv.Character(token),
	)
	if len(destination) < int(required) {
		return Token_Count(required), Status(STATUS_OUTPUT_TOO_SMALL)
	}
	return Token_Count(copy(destination, storage[:required])), Status(STATUS_OK)
}

// Position_Append_Into writes standard position form into caller storage.
func Position_Append_Into(
	destination Position_Output, position Position,
) (count Position_Count, status Status) {
	defer func() {
		Position_Count_Invariants(count, "Position_Append_Into.count")
		Status_Invariants(status, "Position_Append_Into.status")
	}()
	Position_Output_Invariants(destination, "Position_Append_Into.destination")
	Position_Invariants(position, "Position_Append_Into.position")
	filename := string(position.Filename)
	if filename == "" {
		filename = DEFAULT_FILENAME
	}
	if !Position_Valid(position) {
		if len(destination) < len(filename) {
			return Position_Count(len(filename)), Status(STATUS_OUTPUT_TOO_SMALL)
		}
		return Position_Count(copy(destination, filename)), Status(STATUS_OK)
	}
	var line_storage [strconv.DECIMAL_TEXT_SIZE_MAXIMUM]byte
	var column_storage [strconv.DECIMAL_TEXT_SIZE_MAXIMUM]byte
	line_count := strconv.Format_Decimal_Into(
		line_storage[:], strconv.Machine_Integer(position.Line),
	)
	column_count := strconv.Format_Decimal_Into(
		column_storage[:], strconv.Machine_Integer(position.Column),
	)
	required := len(filename) + POSITION_SEPARATOR_COUNT +
		int(line_count) + int(column_count)
	if len(destination) < required {
		return Position_Count(required), Status(STATUS_OUTPUT_TOO_SMALL)
	}
	written := copy(destination, filename)
	destination[written] = ':'
	written++
	written += copy(destination[written:], line_storage[:line_count])
	destination[written] = ':'
	written++
	written += copy(destination[written:], column_storage[:column_count])
	return Position_Count(written), Status(STATUS_OK)
}

func scan_token_unchecked(
	cursor *Scanner_Decoder,
	diagnostics *Scanner_Diagnostics,
	start Token_Start,
	mode Mode,
	identifier_function Identifier_Function,
	character Character,
) (
	token Token, next Character, skip Boolean,
) {
	defer func() {
		Token_Invariants(token, "scan_token_unchecked.token")
		Character_Invariants(next, "scan_token_unchecked.next")
		Boolean_Invariants(skip, "scan_token_unchecked.skip")
	}()
	Scanner_Decoder_Invariants(*cursor, "scan_token_unchecked.cursor")
	Scanner_Diagnostics_Invariants(*diagnostics, "scan_token_unchecked.diagnostics")
	Token_Start_Invariants(start, "scan_token_unchecked.start")
	Mode_Invariants(mode, "scan_token_unchecked.mode")
	Character_Invariants(character, "scan_token_unchecked.character")
	token = Token(character)
	identifier := scanner_identifier_unchecked(
		identifier_function, character, CHARACTER_INDEX_MINIMUM)
	if bool(identifier) {
		if mode&SCAN_IDENTIFIERS != MODE_MINIMUM {
			return TOKEN_IDENTIFIER, scan_identifier_unchecked(
				cursor, diagnostics, identifier_function,
			), false
		}
		return token, scanner_advance_unchecked(cursor, diagnostics), false
	}
	if bool(decimal_unchecked(character)) {
		if mode&(SCAN_INTEGERS|SCAN_FLOATS) != MODE_MINIMUM {
			number, next_character := scan_number_unchecked(
				cursor, diagnostics, start,
				Decimal_Character(character), false,
				Boolean(mode&SCAN_FLOATS != MODE_MINIMUM),
			)
			return Token(number), next_character, false
		}
		return token, scanner_advance_unchecked(cursor, diagnostics), false
	}
	return scan_symbol_unchecked(cursor, diagnostics, start, mode, character)
}

func scan_symbol_unchecked(
	cursor *Scanner_Decoder,
	diagnostics *Scanner_Diagnostics,
	start Token_Start,
	mode Mode,
	character Character,
) (token Token, next Character, skip Boolean) {
	defer func() {
		Token_Invariants(token, "scan_symbol_unchecked.token")
		Character_Invariants(next, "scan_symbol_unchecked.next")
		Boolean_Invariants(skip, "scan_symbol_unchecked.skip")
	}()
	Scanner_Decoder_Invariants(*cursor, "scan_symbol_unchecked.cursor")
	Scanner_Diagnostics_Invariants(*diagnostics, "scan_symbol_unchecked.diagnostics")
	Token_Start_Invariants(start, "scan_symbol_unchecked.start")
	Mode_Invariants(mode, "scan_symbol_unchecked.mode")
	Character_Invariants(character, "scan_symbol_unchecked.character")
	token = Token(character)
	switch character {
	case Character(TOKEN_EOF):
		return token, character, false
	case '"':
		if mode&SCAN_STRINGS != MODE_MINIMUM {
			scan_string_unchecked(cursor, diagnostics, QUOTE_STRING)
			token = TOKEN_STRING
		}
		return token, scanner_advance_unchecked(cursor, diagnostics), false
	case '\'':
		if mode&SCAN_CHARACTERS != MODE_MINIMUM {
			count := scan_string_unchecked(cursor, diagnostics, QUOTE_CHARACTER)
			if count != CHARACTER_LITERAL_COUNT {
				scanner_report_unchecked(
					diagnostics, ERROR_INVALID_CHARACTER_LITERAL, character,
					cursor.Character_Offset, Report_Line(cursor.Character_Line),
					Report_Column(cursor.Character_Column))
			}
			token = TOKEN_CHARACTER
		}
		return token, scanner_advance_unchecked(cursor, diagnostics), false
	case '.':
		character = scanner_advance_unchecked(cursor, diagnostics)
		if bool(decimal_unchecked(character)) {
			if mode&SCAN_FLOATS != MODE_MINIMUM {
				var number Number_Token
				number, character = scan_number_unchecked(
					cursor, diagnostics, start,
					Decimal_Character(character), true,
					Boolean(mode&SCAN_FLOATS != MODE_MINIMUM),
				)
				token = Token(number)
			}
		}
		return token, character, false
	case '/':
		slash, character, skipped := scan_slash_unchecked(
			cursor,
			diagnostics,
			Boolean(mode&SCAN_COMMENTS != MODE_MINIMUM),
			Boolean(mode&SKIP_COMMENTS != MODE_MINIMUM),
		)
		return Token(slash), character, skipped
	case '`':
		if mode&SCAN_RAW_STRINGS != MODE_MINIMUM {
			scan_raw_string_unchecked(cursor, diagnostics)
			token = TOKEN_RAW_STRING
		}
		return token, scanner_advance_unchecked(cursor, diagnostics), false
	default:
		return token, scanner_advance_unchecked(cursor, diagnostics), false
	}
}

func scan_slash_unchecked(
	cursor *Scanner_Decoder,
	diagnostics *Scanner_Diagnostics,
	comments Boolean,
	skip_comments Boolean,
) (result Slash_Token, next Character, skip Boolean) {
	defer func() {
		Slash_Token_Invariants(result, "scan_slash_unchecked.result")
		Character_Invariants(next, "scan_slash_unchecked.next")
		Boolean_Invariants(skip, "scan_slash_unchecked.skip")
	}()
	Scanner_Decoder_Invariants(*cursor, "scan_slash_unchecked.cursor")
	Scanner_Diagnostics_Invariants(*diagnostics, "scan_slash_unchecked.diagnostics")
	Boolean_Invariants(comments, "scan_slash_unchecked.comments")
	Boolean_Invariants(skip_comments, "scan_slash_unchecked.skip_comments")
	character := scanner_advance_unchecked(cursor, diagnostics)
	comment := character == Character(COMMENT_KIND_LINE)
	if character == Character(COMMENT_KIND_BLOCK) {
		comment = true
	}
	if !comment {
		return SLASH_TOKEN, character, false
	}
	if !bool(comments) {
		return SLASH_TOKEN, character, false
	}
	if character == Character(COMMENT_KIND_LINE) {
		character = scanner_advance_unchecked(cursor, diagnostics)
		for character != '\n' {
			if character < Character(utf8.DECODED_CHARACTER_MINIMUM) {
				break
			}
			character = scanner_advance_unchecked(cursor, diagnostics)
		}
	} else {
		character = scanner_advance_unchecked(cursor, diagnostics)
		closed := false
		for !closed {
			if character < Character(utf8.DECODED_CHARACTER_MINIMUM) {
				scanner_report_unchecked(
					diagnostics, ERROR_COMMENT_NOT_TERMINATED, character,
					cursor.Character_Offset, Report_Line(cursor.Character_Line),
					Report_Column(cursor.Character_Column))
				break
			}
			previous := character
			character = scanner_advance_unchecked(cursor, diagnostics)
			if previous == Character(COMMENT_KIND_BLOCK) {
				if character == Character(COMMENT_KIND_LINE) {
					closed = true
				}
			}
		}
		if closed {
			character = scanner_advance_unchecked(cursor, diagnostics)
		}
	}
	if bool(skip_comments) {
		return SLASH_TOKEN, character, true
	}
	return Slash_Token(TOKEN_COMMENT), character, false
}

func scanner_peek_unchecked(
	cursor *Scanner_Cursor, diagnostics *Scanner_Diagnostics,
) (character Character) {
	defer func() { Character_Invariants(character, "scanner_peek_unchecked.character") }()
	Scanner_Cursor_Invariants(*cursor, "scanner_peek_unchecked.cursor")
	Scanner_Diagnostics_Invariants(*diagnostics, "scanner_peek_unchecked.diagnostics")
	if bool(cursor.Looked) {
		return Character(cursor.Character)
	}
	character = scanner_decode_unchecked(&cursor.Decoder, diagnostics)
	if cursor.Decoder.Character_Offset == CHARACTER_INDEX_MINIMUM {
		if character == '\uFEFF' {
			character = scanner_decode_unchecked(&cursor.Decoder, diagnostics)
		}
	}
	cursor.Character = Look_Ahead_Character(character)
	cursor.Looked = true
	return character
}

func scanner_decode_unchecked(
	cursor *Scanner_Decoder, diagnostics *Scanner_Diagnostics,
) (character Character) {
	defer func() { Character_Invariants(character, "scanner_decode_unchecked.character") }()
	Scanner_Decoder_Invariants(*cursor, "scanner_decode_unchecked.cursor")
	Scanner_Diagnostics_Invariants(*diagnostics, "scanner_decode_unchecked.diagnostics")
	line := Error_Line(cursor.Line)
	if line > ERROR_LINE_MAXIMUM {
		line = ERROR_LINE_MAXIMUM
	}
	column := Error_Column(cursor.Column + Source_Column(COLUMN_MINIMUM))
	if column > ERROR_COLUMN_MAXIMUM {
		column = ERROR_COLUMN_MAXIMUM
	}
	position := Error_Position{
		Filename: diagnostics.Filename,
		Offset:   Offset(cursor.Source_Offset),
		Line:     line,
		Column:   column,
	}
	cursor.Character_Offset = Character_Offset(position.Offset)
	cursor.Character_Line = Character_Line(position.Line)
	cursor.Character_Column = Character_Column(position.Column)
	if int(cursor.Source_Offset) == len(cursor.Source) {
		return Character(TOKEN_EOF)
	}
	tail := utf8.Text(cursor.Source[cursor.Source_Offset:])
	decoded, size := utf8.Decode_Character_Text(tail)
	character = Character(decoded)
	first := cursor.Source[cursor.Source_Offset]
	cursor.Source_Offset += Source_Offset(size)
	if cursor.Column < COLUMN_MAXIMUM {
		cursor.Column++
	}
	if character == Character(utf8.REPLACEMENT_CHARACTER) {
		if size == utf8.CHARACTER_SIZE_MINIMUM {
			if first >= byte(utf8.CHARACTER_SELF) {
				scanner_error_unchecked(
					diagnostics, ERROR_INVALID_UTF8, character, position,
				)
			}
		}
	}
	if character == Character(utf8.DECODED_CHARACTER_MINIMUM) {
		scanner_error_unchecked(
			diagnostics, ERROR_INVALID_NUL, character, position,
		)
	}
	if character == '\n' {
		if cursor.Line < LINE_MAXIMUM {
			cursor.Line++
		}
		cursor.Column = COLUMN_INVALID
	}
	return character
}

func scanner_advance_unchecked(
	cursor *Scanner_Decoder, diagnostics *Scanner_Diagnostics,
) (character Character) {
	defer func() {
		Character_Invariants(character, "scanner_advance_unchecked.character")
	}()
	Scanner_Decoder_Invariants(*cursor, "scanner_advance_unchecked.cursor")
	Scanner_Diagnostics_Invariants(*diagnostics, "scanner_advance_unchecked.diagnostics")
	return scanner_decode_unchecked(cursor, diagnostics)
}

func scanner_whitespace_unchecked(
	whitespace Whitespace, character Character,
) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "scanner_whitespace_unchecked.yes") }()
	Whitespace_Invariants(whitespace, "scanner_whitespace_unchecked.whitespace")
	Character_Invariants(character, "scanner_whitespace_unchecked.character")
	if character < Character(utf8.DECODED_CHARACTER_MINIMUM) {
		return false
	}
	if character > WHITESPACE_CHARACTER_MAXIMUM {
		return false
	}
	return Boolean(
		whitespace&
			(Whitespace(utf8.CHARACTER_SIZE_MINIMUM)<<uint(character)) !=
			WHITESPACE_MINIMUM,
	)
}

func scanner_identifier_unchecked(
	identifier Identifier_Function, character Character, index Character_Index,
) (accepted Boolean) {
	defer func() {
		Boolean_Invariants(accepted, "scanner_identifier_unchecked.accepted")
	}()
	Character_Invariants(character, "scanner_identifier_unchecked.character")
	Character_Index_Invariants(index, "scanner_identifier_unchecked.index")
	if character < Character(utf8.DECODED_CHARACTER_MINIMUM) {
		return false
	}
	if identifier != nil {
		return Boolean(identifier(character, index))
	}
	if character == '_' {
		return true
	}
	if bool(ucd.Is_Letter(ucd.Character(character))) {
		return true
	}
	if index == CHARACTER_INDEX_MINIMUM {
		return false
	}
	return Boolean(ucd.Is_Digit(ucd.Character(character)))
}

func scan_identifier_unchecked(
	cursor *Scanner_Decoder,
	diagnostics *Scanner_Diagnostics,
	identifier Identifier_Function,
) (character Character) {
	defer func() { Character_Invariants(character, "scan_identifier_unchecked.character") }()
	Scanner_Decoder_Invariants(*cursor, "scan_identifier_unchecked.cursor")
	Scanner_Diagnostics_Invariants(*diagnostics, "scan_identifier_unchecked.diagnostics")
	character = scanner_advance_unchecked(cursor, diagnostics)
	index := Character_Index(utf8.CHARACTER_SIZE_MINIMUM)
	for bool(scanner_identifier_unchecked(identifier, character, index)) {
		character = scanner_advance_unchecked(cursor, diagnostics)
		index++
	}
	return character
}

func scan_number_unchecked(
	cursor *Scanner_Decoder,
	diagnostics *Scanner_Diagnostics,
	start Token_Start,
	decimal Decimal_Character,
	seen_dot Boolean,
	floats Boolean,
) (token Number_Token, next Character) {
	defer func() {
		Number_Token_Invariants(token, "scan_number_unchecked.token")
		Character_Invariants(next, "scan_number_unchecked.next")
	}()
	Scanner_Decoder_Invariants(*cursor, "scan_number_unchecked.cursor")
	Scanner_Diagnostics_Invariants(*diagnostics, "scan_number_unchecked.diagnostics")
	Token_Start_Invariants(start, "scan_number_unchecked.start")
	Decimal_Character_Invariants(decimal, "scan_number_unchecked.character")
	Boolean_Invariants(seen_dot, "scan_number_unchecked.seen_dot")
	Boolean_Invariants(floats, "scan_number_unchecked.floats")
	character, base, explicit, token, digits_and_separators, invalid :=
		scan_mantissa_unchecked(
			cursor, diagnostics, decimal, seen_dot, floats)
	action := exponent_action_unchecked(character, base, explicit, token, floats)
	if action == EXPONENT_ACTION_ABSENT {
		scanner_report_unchecked(
			diagnostics, ERROR_HEXADECIMAL_MANTISSA_HAS_NO_EXPONENT, character,
			cursor.Character_Offset, Report_Line(cursor.Character_Line),
			Report_Column(cursor.Character_Column))
	}
	scan_exponent := action == EXPONENT_ACTION_SCAN
	if action == EXPONENT_ACTION_WRONG_MANTISSA {
		scanner_report_unchecked(
			diagnostics, ERROR_EXPONENT_WRONG_MANTISSA, character,
			cursor.Character_Offset, Report_Line(cursor.Character_Line),
			Report_Column(cursor.Character_Column))
		scan_exponent = true
	}
	if scan_exponent {
		character = scanner_advance_unchecked(cursor, diagnostics)
		if character == '+' {
			character = scanner_advance_unchecked(cursor, diagnostics)
		}
		if character == '-' {
			character = scanner_advance_unchecked(cursor, diagnostics)
		}
		var found Digit_Flags
		var exponent_invalid Invalid_Digit
		character, found, exponent_invalid = scan_digits_unchecked(
			cursor, diagnostics, character, BASE_DECIMAL)
		if exponent_invalid != INVALID_DIGIT_ABSENT {
			scanner_report_unchecked(
				diagnostics, ERROR_INVALID_DIGIT,
				Character(DECIMAL_CHARACTER_MINIMUM+exponent_invalid),
				cursor.Character_Offset, Report_Line(cursor.Character_Line),
				Report_Column(cursor.Character_Column))
		}
		digits_and_separators |= found
		if found&DIGIT_FLAGS_DIGIT == DIGIT_FLAGS_NONE {
			scanner_report_unchecked(
				diagnostics, ERROR_EXPONENT_HAS_NO_DIGITS, character,
				cursor.Character_Offset, Report_Line(cursor.Character_Line),
				Report_Column(cursor.Character_Column))
		}
		token = Number_Token(TOKEN_FLOAT)
	}
	scan_number_finish_unchecked(
		diagnostics, cursor.Source, start, cursor.Character_Offset,
		Report_Line(cursor.Character_Line), Report_Column(cursor.Character_Column),
		character, token, digits_and_separators, invalid,
	)
	return token, character
}

func scan_number_finish_unchecked(
	diagnostics *Scanner_Diagnostics,
	source Source,
	start Token_Start,
	end Character_Offset,
	line Report_Line,
	column Report_Column,
	character Character,
	token Number_Token,
	flags Digit_Flags,
	invalid Invalid_Digit,
) {
	Scanner_Diagnostics_Invariants(*diagnostics, "scan_number_finish_unchecked.diagnostics")
	Source_Invariants(source, "scan_number_finish_unchecked.source")
	Token_Start_Invariants(start, "scan_number_finish_unchecked.start")
	Character_Offset_Invariants(end, "scan_number_finish_unchecked.end")
	Report_Line_Invariants(line, "scan_number_finish_unchecked.line")
	Report_Column_Invariants(column, "scan_number_finish_unchecked.column")
	Character_Invariants(character, "scan_number_finish_unchecked.character")
	Number_Token_Invariants(token, "scan_number_finish_unchecked.token")
	Digit_Flags_Invariants(flags, "scan_number_finish_unchecked.flags")
	Invalid_Digit_Invariants(invalid, "scan_number_finish_unchecked.invalid")
	if token == Number_Token(TOKEN_INTEGER) {
		if invalid != INVALID_DIGIT_ABSENT {
			scanner_report_unchecked(
				diagnostics, ERROR_INVALID_DIGIT,
				Character(DECIMAL_CHARACTER_MINIMUM+invalid),
				end, line, column)
		}
	}
	if flags&DIGIT_FLAGS_SEPARATOR != DIGIT_FLAGS_NONE {
		literal := Number_Literal(source[start:end])
		if invalid_separator_unchecked(literal) != SEPARATOR_INDEX_ABSENT {
			scanner_report_unchecked(
				diagnostics, ERROR_INVALID_SEPARATOR, character,
				end, line, column)
		}
	}
}

func scan_mantissa_unchecked(
	cursor *Scanner_Decoder,
	diagnostics *Scanner_Diagnostics,
	decimal Decimal_Character,
	seen_dot Boolean,
	floats Boolean,
) (
	character Character,
	base Base,
	explicit Boolean,
	token Number_Token,
	flags Digit_Flags,
	invalid Invalid_Digit,
) {
	defer func() {
		Character_Invariants(character, "scan_mantissa_unchecked.character")
		Base_Invariants(base, "scan_mantissa_unchecked.base")
		Boolean_Invariants(explicit, "scan_mantissa_unchecked.explicit")
		Number_Token_Invariants(token, "scan_mantissa_unchecked.token")
		Digit_Flags_Invariants(flags, "scan_mantissa_unchecked.flags")
		Invalid_Digit_Invariants(invalid, "scan_mantissa_unchecked.invalid")
	}()
	Scanner_Decoder_Invariants(*cursor, "scan_mantissa_unchecked.cursor")
	Scanner_Diagnostics_Invariants(*diagnostics, "scan_mantissa_unchecked.diagnostics")
	Decimal_Character_Invariants(decimal, "scan_mantissa_unchecked.decimal")
	Boolean_Invariants(seen_dot, "scan_mantissa_unchecked.seen_dot")
	Boolean_Invariants(floats, "scan_mantissa_unchecked.floats")
	character, base, explicit = Character(decimal), Base(BASE_DECIMAL), false
	invalid = Invalid_Digit(INVALID_DIGIT_ABSENT)
	token = Number_Token(TOKEN_INTEGER)
	if !bool(seen_dot) {
		var prefix_flags Radix_Digit_Flags
		character, base, explicit, prefix_flags =
			scan_radix_unchecked(cursor, diagnostics, decimal)
		flags = Digit_Flags(prefix_flags)
		var found Digit_Flags
		character, found, invalid = scan_digits_unchecked(
			cursor, diagnostics, character, base)
		flags |= found
		if character == '.' {
			if bool(floats) {
				character = scanner_advance_unchecked(cursor, diagnostics)
				seen_dot = true
			}
		}
	}
	if seen_dot {
		token = Number_Token(TOKEN_FLOAT)
		if bool(radix_point_invalid(base, explicit)) {
			scanner_report_unchecked(
				diagnostics, ERROR_INVALID_RADIX_POINT, character,
				cursor.Character_Offset, Report_Line(cursor.Character_Line),
				Report_Column(cursor.Character_Column))
		}
		var found Digit_Flags
		var fraction_invalid Invalid_Digit
		character, found, fraction_invalid = scan_digits_unchecked(
			cursor, diagnostics, character, base)
		if invalid == INVALID_DIGIT_ABSENT {
			invalid = fraction_invalid
		}
		flags |= found
	}
	if flags&DIGIT_FLAGS_DIGIT == DIGIT_FLAGS_NONE {
		scanner_report_unchecked(
			diagnostics, ERROR_LITERAL_HAS_NO_DIGITS, character,
			cursor.Character_Offset, Report_Line(cursor.Character_Line),
			Report_Column(cursor.Character_Column))
	}
	return character, base, explicit, token, flags, invalid
}

func scan_radix_unchecked(
	cursor *Scanner_Decoder,
	diagnostics *Scanner_Diagnostics,
	decimal Decimal_Character,
) (
	next Character,
	base Base,
	explicit Boolean,
	digits_and_separators Radix_Digit_Flags,
) {
	defer func() {
		Character_Invariants(next, "scan_radix_unchecked.next")
		Base_Invariants(base, "scan_radix_unchecked.base")
		Boolean_Invariants(explicit, "scan_radix_unchecked.explicit")
		Radix_Digit_Flags_Invariants(
			digits_and_separators, "scan_radix_unchecked.digits_and_separators",
		)
	}()
	Scanner_Decoder_Invariants(*cursor, "scan_radix_unchecked.cursor")
	Scanner_Diagnostics_Invariants(*diagnostics, "scan_radix_unchecked.diagnostics")
	Decimal_Character_Invariants(decimal, "scan_radix_unchecked.character")
	character := Character(decimal)
	if character != '0' {
		return character, BASE_DECIMAL, false, Radix_Digit_Flags(DIGIT_FLAGS_NONE)
	}
	character = scanner_advance_unchecked(cursor, diagnostics)
	switch LOWER_CASE_OFFSET | character {
	case 'x':
		return scanner_advance_unchecked(cursor, diagnostics), BASE_HEXADECIMAL, true,
			Radix_Digit_Flags(DIGIT_FLAGS_NONE)
	case 'o':
		return scanner_advance_unchecked(cursor, diagnostics), BASE_OCTAL, true,
			Radix_Digit_Flags(DIGIT_FLAGS_NONE)
	case 'b':
		return scanner_advance_unchecked(cursor, diagnostics), BASE_BINARY, true,
			Radix_Digit_Flags(DIGIT_FLAGS_NONE)
	default:
		return character, BASE_OCTAL, false, Radix_Digit_Flags(DIGIT_FLAGS_DIGIT)
	}
}

func radix_point_invalid(base Base, explicit Boolean) (invalid Boolean) {
	defer func() { Boolean_Invariants(invalid, "radix_point_invalid.invalid") }()
	Base_Invariants(base, "radix_point_invalid.base")
	Boolean_Invariants(explicit, "radix_point_invalid.explicit")
	if !bool(explicit) {
		return false
	}
	return Boolean(base != BASE_HEXADECIMAL)
}

func exponent_action_unchecked(
	character Character,
	base Base,
	explicit Boolean,
	token Number_Token,
	floats Boolean,
) (action Exponent_Action) {
	defer func() { Exponent_Action_Invariants(action, "exponent_action_unchecked.action") }()
	Character_Invariants(character, "exponent_action_unchecked.character")
	Base_Invariants(base, "exponent_action_unchecked.base")
	Boolean_Invariants(explicit, "exponent_action_unchecked.explicit")
	Number_Token_Invariants(token, "exponent_action_unchecked.token")
	Boolean_Invariants(floats, "exponent_action_unchecked.floats")
	exponent := LOWER_CASE_OFFSET | character
	if exponent != EXPONENT_CHARACTER_DECIMAL {
		if exponent != EXPONENT_CHARACTER_BINARY {
			if bool(explicit) {
				if base == BASE_HEXADECIMAL {
					if token == Number_Token(TOKEN_FLOAT) {
						return EXPONENT_ACTION_ABSENT
					}
				}
			}
			return EXPONENT_ACTION_NONE
		}
	}
	if !bool(floats) {
		return EXPONENT_ACTION_NONE
	}
	if exponent == EXPONENT_CHARACTER_DECIMAL {
		if bool(explicit) {
			return EXPONENT_ACTION_WRONG_MANTISSA
		}
		return EXPONENT_ACTION_SCAN
	}
	if !bool(explicit) {
		return EXPONENT_ACTION_WRONG_MANTISSA
	}
	if base != BASE_HEXADECIMAL {
		return EXPONENT_ACTION_WRONG_MANTISSA
	}
	return EXPONENT_ACTION_SCAN
}

func scan_digits_unchecked(
	cursor *Scanner_Decoder,
	diagnostics *Scanner_Diagnostics,
	character Character,
	base Base,
) (
	next Character,
	digits_and_separators Digit_Flags,
	invalid Invalid_Digit,
) {
	defer func() {
		Character_Invariants(next, "scan_digits_unchecked.next")
		Digit_Flags_Invariants(digits_and_separators, "scan_digits_unchecked.flags")
		Invalid_Digit_Invariants(invalid, "scan_digits_unchecked.invalid")
	}()
	Scanner_Decoder_Invariants(*cursor, "scan_digits_unchecked.cursor")
	Scanner_Diagnostics_Invariants(*diagnostics, "scan_digits_unchecked.diagnostics")
	Character_Invariants(character, "scan_digits_unchecked.character")
	Base_Invariants(base, "scan_digits_unchecked.base")
	invalid = Invalid_Digit(INVALID_DIGIT_ABSENT)
	if base <= BASE_DECIMAL {
		maximum := Character('0' + base)
		for bool(decimal_unchecked(character)) || character == '_' {
			seen := Digit_Flags(DIGIT_FLAGS_DIGIT)
			if character == '_' {
				seen = DIGIT_FLAGS_SEPARATOR
			} else if character >= maximum {
				if invalid == INVALID_DIGIT_ABSENT {
					invalid = Invalid_Digit(
						character - DECIMAL_CHARACTER_MINIMUM,
					)
				}
			}
			digits_and_separators |= seen
			character = scanner_advance_unchecked(cursor, diagnostics)
		}
		return character, digits_and_separators, invalid
	}
	for bool(hexadecimal_unchecked(character)) || character == '_' {
		seen := Digit_Flags(DIGIT_FLAGS_DIGIT)
		if character == '_' {
			seen = DIGIT_FLAGS_SEPARATOR
		}
		digits_and_separators |= seen
		character = scanner_advance_unchecked(cursor, diagnostics)
	}
	return character, digits_and_separators, invalid
}

func scan_string_unchecked(
	cursor *Scanner_Decoder, diagnostics *Scanner_Diagnostics, quote Quote,
) (count Character_Count) {
	defer func() { Character_Count_Invariants(count, "scan_string_unchecked.count") }()
	Scanner_Decoder_Invariants(*cursor, "scan_string_unchecked.cursor")
	Scanner_Diagnostics_Invariants(*diagnostics, "scan_string_unchecked.diagnostics")
	Quote_Invariants(quote, "scan_string_unchecked.quote")
	delimiter := Character(quote)
	character_count := Character_Count(CHARACTER_COUNT_MINIMUM)
	character := scanner_advance_unchecked(cursor, diagnostics)
	for character != delimiter {
		if bool(string_character_invalid(character)) {
			scanner_report_unchecked(
				diagnostics, ERROR_LITERAL_NOT_TERMINATED, character,
				cursor.Character_Offset, Report_Line(cursor.Character_Line),
				Report_Column(cursor.Character_Column))
			return character_count
		}
		if character == '\\' {
			character = scanner_advance_unchecked(cursor, diagnostics)
			action := escape_action_unchecked(character, quote)
			base := Escape_Base(BASE_HEXADECIMAL)
			digit_count := Escape_Digit_Count(CHARACTER_COUNT_MINIMUM)
			switch action {
			case ESCAPE_ACTION_DIRECT:
				character = scanner_advance_unchecked(cursor, diagnostics)
			case ESCAPE_ACTION_OCTAL:
				base = BASE_OCTAL
				digit_count = ESCAPE_DIGIT_COUNT_OCTAL
			case ESCAPE_ACTION_HEXADECIMAL_BYTE:
				digit_count = ESCAPE_DIGIT_COUNT_HEXADECIMAL_BYTE
				character = scanner_advance_unchecked(cursor, diagnostics)
			case ESCAPE_ACTION_UNICODE_SHORT:
				digit_count = ESCAPE_DIGIT_COUNT_UNICODE_SHORT
				character = scanner_advance_unchecked(cursor, diagnostics)
			case ESCAPE_ACTION_UNICODE_LONG:
				digit_count = ESCAPE_DIGIT_COUNT_UNICODE_LONG
				character = scanner_advance_unchecked(cursor, diagnostics)
			default:
				scanner_report_unchecked(
					diagnostics, ERROR_INVALID_ESCAPE, character,
					cursor.Character_Offset, Report_Line(cursor.Character_Line),
					Report_Column(cursor.Character_Column))
			}
			for digit_count > CHARACTER_COUNT_MINIMUM {
				if digit_value_unchecked(character) >= Digit_Value(base) {
					break
				}
				character = scanner_advance_unchecked(cursor, diagnostics)
				digit_count--
			}
			if digit_count > CHARACTER_COUNT_MINIMUM {
				scanner_report_unchecked(
					diagnostics, ERROR_INVALID_ESCAPE, character,
					cursor.Character_Offset, Report_Line(cursor.Character_Line),
					Report_Column(cursor.Character_Column))
			}
		} else {
			character = scanner_advance_unchecked(cursor, diagnostics)
		}
		character_count++
	}
	return character_count
}

func escape_action_unchecked(
	character Character, quote Quote,
) (action Escape_Action) {
	defer func() { Escape_Action_Invariants(action, "escape_action_unchecked.action") }()
	Character_Invariants(character, "escape_action_unchecked.character")
	Quote_Invariants(quote, "escape_action_unchecked.quote")
	switch character {
	case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', Character(quote):
		return ESCAPE_ACTION_DIRECT
	case '0', '1', '2', '3', '4', '5', '6', '7':
		return ESCAPE_ACTION_OCTAL
	case 'x':
		return ESCAPE_ACTION_HEXADECIMAL_BYTE
	case 'u':
		return ESCAPE_ACTION_UNICODE_SHORT
	case 'U':
		return ESCAPE_ACTION_UNICODE_LONG
	default:
		return ESCAPE_ACTION_INVALID
	}
}

func string_character_invalid(character Character) (invalid Boolean) {
	defer func() { Boolean_Invariants(invalid, "string_character_invalid.invalid") }()
	Character_Invariants(character, "string_character_invalid.character")
	if character == '\n' {
		return true
	}
	return Boolean(character < Character(utf8.DECODED_CHARACTER_MINIMUM))
}

func scan_raw_string_unchecked(
	cursor *Scanner_Decoder, diagnostics *Scanner_Diagnostics,
) {
	Scanner_Decoder_Invariants(*cursor, "scan_raw_string_unchecked.cursor")
	Scanner_Diagnostics_Invariants(*diagnostics, "scan_raw_string_unchecked.diagnostics")
	character := scanner_advance_unchecked(cursor, diagnostics)
	for character != '`' {
		if character < Character(utf8.DECODED_CHARACTER_MINIMUM) {
			scanner_report_unchecked(
				diagnostics, ERROR_LITERAL_NOT_TERMINATED, character,
				cursor.Character_Offset, Report_Line(cursor.Character_Line),
				Report_Column(cursor.Character_Column))
			return
		}
		character = scanner_advance_unchecked(cursor, diagnostics)
	}
}

func scanner_report_unchecked(
	diagnostics *Scanner_Diagnostics,
	code Lexical_Report_Code,
	character Character,
	offset Character_Offset,
	line Report_Line,
	column Report_Column,
) {
	Scanner_Diagnostics_Invariants(*diagnostics, "scanner_report_unchecked.diagnostics")
	Lexical_Report_Code_Invariants(code, "scanner_report_unchecked.code")
	Character_Invariants(character, "scanner_report_unchecked.character")
	Character_Offset_Invariants(offset, "scanner_report_unchecked.offset")
	Report_Line_Invariants(line, "scanner_report_unchecked.line")
	Report_Column_Invariants(column, "scanner_report_unchecked.column")
	report_line := Error_Line(line)
	if report_line < ERROR_LINE_MINIMUM {
		report_line = ERROR_LINE_MINIMUM
	}
	position := Error_Position{
		Filename: diagnostics.Filename,
		Offset:   Offset(offset),
		Line:     report_line,
		Column:   Error_Column(column),
	}
	scanner_error_unchecked(diagnostics, Report_Code(code), character, position)
}

func scanner_error_unchecked(
	diagnostics *Scanner_Diagnostics,
	code Report_Code,
	character Character,
	position Error_Position,
) {
	Scanner_Diagnostics_Invariants(*diagnostics, "scanner_error_unchecked.diagnostics")
	Report_Code_Invariants(code, "scanner_error_unchecked.code")
	Character_Invariants(character, "scanner_error_unchecked.character")
	Error_Position_Invariants(position, "scanner_error_unchecked.position")
	if diagnostics.Error_Count < ERROR_COUNT_MAXIMUM {
		diagnostics.Error_Count++
	}
	report := Error{
		Code:      Error_Code(code),
		Character: character,
		Position: Position{
			Filename: position.Filename,
			Offset:   position.Offset,
			Line:     Line(position.Line),
			Column:   Column(position.Column),
		},
	}
	if diagnostics.Error != nil {
		diagnostics.Error(report)
	}
}

func decimal_unchecked(character Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "decimal_unchecked.yes") }()
	Character_Invariants(character, "decimal_unchecked.character")
	if character < '0' {
		return false
	}
	return Boolean(character <= '9')
}

func hexadecimal_unchecked(character Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "hexadecimal_unchecked.yes") }()
	Character_Invariants(character, "hexadecimal_unchecked.character")
	if decimal_unchecked(character) {
		return true
	}
	lower := LOWER_CASE_OFFSET | character
	if lower < 'a' {
		return false
	}
	return Boolean(lower <= 'f')
}

func digit_value_unchecked(character Character) (value Digit_Value) {
	defer func() { Digit_Value_Invariants(value, "digit_value_unchecked.value") }()
	Character_Invariants(character, "digit_value_unchecked.character")
	if decimal_unchecked(character) {
		return Digit_Value(character - '0')
	}
	lower := LOWER_CASE_OFFSET | character
	if lower >= 'a' {
		if lower <= 'f' {
			return Digit_Value(lower - 'a' + BASE_DECIMAL)
		}
	}
	return DIGIT_VALUE_MAXIMUM
}

func invalid_separator_unchecked(literal Number_Literal) (position Separator_Index) {
	defer func() {
		Separator_Index_Invariants(position, "invalid_separator_unchecked.position")
	}()
	Number_Literal_Invariants(literal, "invalid_separator_unchecked.literal")
	prefix := Character(' ')
	digit := Character('.')
	index := SEPARATOR_INDEX_ABSENT
	if len(literal) >= NUMBER_PREFIX_SIZE {
		if literal[SEPARATOR_INDEX_ABSENT] == '0' {
			prefix = LOWER_CASE_OFFSET |
				Character(literal[utf8.CHARACTER_SIZE_MINIMUM])
			if prefix == 'x' {
				digit = '0'
				index = NUMBER_PREFIX_SIZE
			}
			if prefix == 'o' {
				digit = '0'
				index = NUMBER_PREFIX_SIZE
			}
			if prefix == 'b' {
				digit = '0'
				index = NUMBER_PREFIX_SIZE
			}
		}
	}
	for ; index < len(literal); index++ {
		previous := digit
		digit = Character(literal[index])
		switch {
		case digit == '_':
			if previous != '0' {
				return Separator_Index(index)
			}
		case bool(decimal_unchecked(digit)):
			digit = '0'
		case prefix == 'x':
			if hexadecimal_unchecked(digit) {
				digit = '0'
			} else {
				if previous == '_' {
					return Separator_Index(index - utf8.CHARACTER_SIZE_MINIMUM)
				}
				digit = '.'
			}
		default:
			if previous == '_' {
				return Separator_Index(index - utf8.CHARACTER_SIZE_MINIMUM)
			}
			digit = '.'
		}
	}
	if digit == '_' {
		return Separator_Index(len(literal) - utf8.CHARACTER_SIZE_MINIMUM)
	}
	return SEPARATOR_INDEX_ABSENT
}
