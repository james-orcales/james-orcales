package strconv_test

import (
	"math"
	"strings"
	"testing"

	"local/james-orcales/shared/math/fixedpoint"
	"local/james-orcales/shared/strconv"
	"local/james-orcales/shared/testify"
)

// Test_Boolean verifies the accepted spellings, the two written forms, and the appended
// form.
func Test_Boolean(t *testing.T) {
	t.Parallel()
	for _, text := range []strconv.Text{"1", "t", "T", "true", "TRUE", "True"} {
		value, parse_error := strconv.Parse_Boolean(text)
		testify.No_Error(t, parse_error, "Parse_Boolean(%q)", text)
		testify.True(t, bool(value), "Parse_Boolean(%q)", text)
	}
	for _, text := range []strconv.Text{"0", "f", "F", "false", "FALSE", "False"} {
		value, parse_error := strconv.Parse_Boolean(text)
		testify.No_Error(t, parse_error, "Parse_Boolean(%q)", text)
		testify.False(t, bool(value), "Parse_Boolean(%q)", text)
	}
	for _, text := range []strconv.Text{"", "t2", "tru", "TrUe"} {
		_, parse_error := strconv.Parse_Boolean(text)
		testify.Error_Is(t, parse_error, strconv.Error_Syntax, "Parse_Boolean(%q)", text)
	}
	testify.Equal(t, strconv.Boolean_Text("true"), strconv.Format_Boolean(true))
	testify.Equal(t, strconv.Boolean_Text("false"), strconv.Format_Boolean(false))
	testify.Equal(t, "true", string(strconv.Append_Boolean(nil, true)))
	for _, size := range buffer_sizes() {
		appended := strconv.Append_Boolean(buffer_of(size), false)
		testify.Equal(t, size+5, len(appended), "Append_Boolean over %d bytes", size)
	}
}

// Test_Text_To_Integer verifies the explicit bases, the implied bases, the underscore
// separator, the width limits, and the two errors.
func Test_Text_To_Integer(t *testing.T) {
	t.Parallel()
	unsigned_cases(t)
	signed_cases(t)
	byte_domain_cases(t)
	decimal_cases(t)
}

// Test_Integer_To_Text verifies each base, the sign, the width extremes, and the
// appended forms.
func Test_Integer_To_Text(t *testing.T) {
	t.Parallel()
	for value, want := range map[strconv.Unsigned_Integer]strconv.Digit_Text{
		0: "0", 1: "1", 2: "10", 3: "11",
	} {
		text := strconv.Format_Unsigned_Integer(value, 2)
		testify.Equal(t, want, text, "Format_Unsigned_Integer(%d, 2)", value)
	}
	testify.Equal(t, strconv.Digit_Text("z"), strconv.Format_Unsigned_Integer(35, 36))
	testify.Equal(t, strconv.Digit_Text("18446744073709551615"),
		strconv.Format_Unsigned_Integer(math.MaxUint64, 10))
	testify.Equal(t, 64, len(strconv.Format_Unsigned_Integer(math.MaxUint64, 2)),
		"the unsigned maximum in base two")
	testify.Equal(t, strconv.Integer_Text("0"), strconv.Format_Integer(0, 10))
	testify.Equal(t, strconv.Integer_Text("-1"), strconv.Format_Integer(-1, 10))
	testify.Equal(t, strconv.Integer_Text("2"), strconv.Format_Integer(2, 10))
	testify.Equal(t, strconv.Integer_Text("10"), strconv.Format_Integer(2, 2))
	testify.Equal(t, strconv.Integer_Text("-ff"), strconv.Format_Integer(-255, 16))
	testify.Equal(t, strconv.Integer_Text("9223372036854775807"),
		strconv.Format_Integer(math.MaxInt64, 10))
	testify.Equal(t, 65, len(strconv.Format_Integer(math.MinInt64, 2)),
		"the signed minimum in base two")
	decimal_text_cases(t)
	append_integer_cases(t)
}

// Test_Fixed_Point verifies the decimal reader, the two written forms, and the storage
// extremes of a fixed-point number.
func Test_Fixed_Point(t *testing.T) {
	t.Parallel()
	for text, want := range map[strconv.Text]fixedpoint.Number{
		"0": 0, "1": fixedpoint.SCALE, "2": 2 * fixedpoint.SCALE,
		"-1": -fixedpoint.SCALE, "0.5": fixedpoint.SCALE / 2,
		"-0.5": -fixedpoint.SCALE / 2, "0.000001": 1, "0.000002": 2,
		"-0.000001": -1, ".5": fixedpoint.SCALE / 2, "5.": 5 * fixedpoint.SCALE,
		"+2.25": 2*fixedpoint.SCALE + fixedpoint.SCALE/4,
	} {
		value, parse_error := strconv.Parse_Fixed_Point(text)
		testify.No_Error(t, parse_error, "Parse_Fixed_Point(%q)", text)
		testify.Equal(t, want, value, "Parse_Fixed_Point(%q)", text)
	}
	smallest, smallest_error := strconv.Parse_Fixed_Point("-8796093022208")
	testify.No_Error(t, smallest_error, "the fixed-point minimum")
	testify.Equal(t, fixedpoint.Number(math.MinInt64), smallest)
	largest, largest_error := strconv.Parse_Fixed_Point("8796093022207.999999")
	testify.No_Error(t, largest_error, "the fixed-point maximum")
	testify.Equal(t, fixedpoint.Number(math.MaxInt64), largest)
	for _, text := range []strconv.Text{"", "abc", "1.2.3", "1e5", "-", ".", "1.a", "+"} {
		_, reject := strconv.Parse_Fixed_Point(text)
		testify.Error_Is(t, reject, strconv.Error_Syntax, "Parse_Fixed_Point(%q)", text)
	}
	for _, text := range []strconv.Text{"8796093022208", "-8796093022209"} {
		_, reject := strconv.Parse_Fixed_Point(text)
		testify.Error_Is(t, reject, strconv.Error_Range, "Parse_Fixed_Point(%q)", text)
	}
	fixed_point_limit_cases(t)
	fixed_point_text_cases(t)
}

// Test_Domain_Errors verifies that the two errors stay distinct and that an
// out-of-domain Base, Bit_Size, or size panics.
func Test_Domain_Errors(t *testing.T) {
	t.Parallel()
	testify.Not_Error_Is(t, strconv.Error_Syntax, strconv.Error_Range,
		"the two errors stay distinct")
	_, range_error := strconv.Parse_Unsigned_Integer("18446744073709551616", 10, 0)
	testify.Error_Is(t, range_error, strconv.Error_Range, "the unsigned overflow")
	testify.Panics(t, func() { strconv.Format_Integer(1, 37) }, "a Base of 37")
	testify.Panics(t, func() { strconv.Format_Integer(1, 1) }, "a Base of 1")
	testify.Panics(t, func() { strconv.Parse_Integer("1", 1, 0) }, "an Implied_Base of 1")
	testify.Panics(t, func() { strconv.Parse_Integer("1", 10, 65) }, "a Bit_Size of 65")
	oversize := strconv.Text(strings.Repeat("1", strconv.TEXT_SIZE_MAXIMUM+1))
	testify.Panics(t, func() { strconv.Quote(oversize) }, "a Text above the size limit")
	large := buffer_of(strconv.BUFFER_SIZE_MAXIMUM + 1)
	testify.Panics(t, func() { strconv.Append_Quote(large, "") },
		"a Buffer above the size limit")
}

// Test_Quote_Forms verifies the three string forms, the three character forms, and the
// appended forms.
func Test_Quote_Forms(t *testing.T) {
	t.Parallel()
	testify.Equal(t, strconv.Quoted_Text(`""`), strconv.Quote(""))
	testify.Equal(t, strconv.Quoted_Text(`"a\tb"`), strconv.Quote("a\tb"))
	testify.Equal(t, strconv.Quoted_Text(`"a\"b\\c"`), strconv.Quote(`a"b\c`))
	testify.Equal(t, strconv.Quoted_Text(`"héllo"`), strconv.Quote("héllo"))
	testify.Equal(t, strconv.Quoted_Text(`"水"`), strconv.Quote("水"))
	for _, text := range []strconv.Text{"", "a", "ab"} {
		testify.Equal(t, len(text)+2, len(strconv.Quote(text)), "Quote(%q)", text)
		testify.Equal(t, len(text)+2, len(strconv.Quote_To_ASCII(text)),
			"Quote_To_ASCII(%q)", text)
		testify.Equal(t, len(text)+2, len(strconv.Quote_To_Graphic(text)),
			"Quote_To_Graphic(%q)", text)
	}
	testify.Equal(t, strconv.Character_Text(`'a'`), strconv.Quote_Rune_To_ASCII('a'))
	testify.Equal(t, strconv.Quoted_Text(`"\xff"`), strconv.Quote("\xff"))
	testify.Equal(t, strconv.Quoted_Text(`"h\u00e9llo"`),
		strconv.Quote_To_ASCII("héllo"))
	testify.Equal(t, strconv.Quoted_Text("\" \""), strconv.Quote_To_Graphic(" "))
	testify.Equal(t, strconv.Quoted_Text(`"\n"`), strconv.Quote_To_Graphic("\n"))
	quote_rune_cases(t)
	append_quote_cases(t)
}

// Test_Backquote_Form verifies the text that stays unchanged inside backquotes and the
// text that does not.
func Test_Backquote_Form(t *testing.T) {
	t.Parallel()
	for _, text := range []strconv.Text{"", "a", "ab", "a\tb", "héllo", "~"} {
		testify.True(t, bool(strconv.Can_Backquote(text)), "Can_Backquote(%q)", text)
	}
	for _, text := range []strconv.Text{
		"`", "a\nb", "\x00", "\x01", "\x02", "\x7f", "\ufeff", "\xff",
	} {
		testify.False(t, bool(strconv.Can_Backquote(text)), "Can_Backquote(%q)", text)
	}
}

// Test_Unquote_Forms verifies each literal form, the prefix form, and the one-character
// form.
func Test_Unquote_Forms(t *testing.T) {
	t.Parallel()
	unquote_cases(t)
	quoted_prefix_cases(t)
	unquote_character_cases(t)
	escape_cases(t)
}

// Test_Printability verifies the printable and graphic classifications, including the
// extremes of the Character domain.
func Test_Printability(t *testing.T) {
	t.Parallel()
	for _, value := range []strconv.Character{'a', ' ', '☺', 0x00e9} {
		testify.True(t, bool(strconv.Is_Print(value)), "Is_Print(%d)", value)
	}
	for _, value := range []strconv.Character{
		0, 1, 2, '\n', 0x7f, 0x00ad, -1, math.MinInt32, math.MaxInt32,
	} {
		testify.False(t, bool(strconv.Is_Print(value)), "Is_Print(%d)", value)
	}
	testify.True(t, bool(strconv.Is_Graphic(' ')), "Is_Graphic of a no-break space")
	for _, value := range []strconv.Character{
		0, 1, 2, '\n', -1, math.MinInt32, math.MaxInt32,
	} {
		testify.False(t, bool(strconv.Is_Graphic(value)), "Is_Graphic(%d)", value)
	}
}

// Test_Size_Limits verifies that every conversion reaches its declared size limit.
func Test_Size_Limits(t *testing.T) {
	t.Parallel()
	full := text_of(strconv.TEXT_SIZE_MAXIMUM, '1')
	invalid := text_of(strconv.TEXT_SIZE_MAXIMUM, '\xff')
	destination := buffer_of(strconv.BUFFER_SIZE_MAXIMUM)
	testify.Equal(t, strconv.QUOTED_TEXT_SIZE_MAXIMUM, len(strconv.Quote(invalid)),
		"Quote of invalid bytes")
	testify.Equal(t, strconv.QUOTED_TEXT_SIZE_MAXIMUM,
		len(strconv.Quote_To_ASCII(invalid)), "Quote_To_ASCII of invalid bytes")
	testify.Equal(t, strconv.QUOTED_TEXT_SIZE_MAXIMUM,
		len(strconv.Quote_To_Graphic(invalid)), "Quote_To_Graphic of invalid bytes")
	// The final ASCII byte makes the ASCII writer receive the largest open literal
	// that one more byte escape can extend.
	mixed := invalid[:len(invalid)-1] + "a"
	testify.Equal(t,
		strconv.QUOTED_TEXT_SIZE_MAXIMUM-strconv.BYTE_ESCAPE_SIZE+
			strconv.ESCAPE_SIZE_MINIMUM,
		len(strconv.Quote(mixed)), "Quote of invalid bytes followed by ASCII")
	testify.Equal(t, strconv.QUOTED_BUFFER_SIZE_MAXIMUM,
		len(strconv.Append_Quote(destination, invalid)), "Append_Quote")
	testify.Equal(t, strconv.QUOTED_BUFFER_SIZE_MAXIMUM,
		len(strconv.Append_Quote_To_ASCII(destination, invalid)),
		"Append_Quote_To_ASCII")
	testify.Equal(t, strconv.QUOTED_BUFFER_SIZE_MAXIMUM,
		len(strconv.Append_Quote_To_Graphic(destination, invalid)),
		"Append_Quote_To_Graphic")
	// An invalid byte takes the inline escape of the quote writer, thus only a valid
	// character that needs a byte escape drives the character writer to its limit.
	control := text_of(strconv.TEXT_SIZE_MAXIMUM, '\x01')
	testify.Equal(t, strconv.QUOTED_TEXT_SIZE_MAXIMUM, len(strconv.Quote(control)),
		"Quote of control characters")
	testify.Equal(t, strconv.QUOTED_TEXT_SIZE_MAXIMUM,
		len(strconv.Quote_To_ASCII(control)), "Quote_To_ASCII of control characters")
	testify.True(t, bool(strconv.Can_Backquote(full)), "a full text of digits")
	full_size_reads(t, full)
	full_size_literals(t)
}

// Exercises Parse_Unsigned_Integer over every base form and both width extremes.
func unsigned_cases(t *testing.T) {
	for text, want := range map[strconv.Text]strconv.Unsigned_Integer{
		"0": 0, "1": 1, "2": 2, "0b11": 3, "0o17": 15, "0x1f": 31, "017": 15,
		"1_000": 1000, "0x_f": 15,
	} {
		value, parse_error := strconv.Parse_Unsigned_Integer(text, 0, 0)
		testify.No_Error(t, parse_error, "Parse_Unsigned_Integer(%q)", text)
		testify.Equal(t, want, value, "Parse_Unsigned_Integer(%q)", text)
	}
	value, _ := strconv.Parse_Unsigned_Integer("z", 36, 0)
	testify.Equal(t, strconv.Unsigned_Integer(35), value, "the digit z in base 36")
	binary, _ := strconv.Parse_Unsigned_Integer("11", 2, 0)
	testify.Equal(t, strconv.Unsigned_Integer(3), binary, "11 in base two")
	narrow, _ := strconv.Parse_Unsigned_Integer("1", 10, 1)
	testify.Equal(t, strconv.Unsigned_Integer(1), narrow, "one bit holds the value one")
	pair, _ := strconv.Parse_Unsigned_Integer("3", 10, 2)
	testify.Equal(t, strconv.Unsigned_Integer(3), pair, "two bits hold the value three")
	widest, parse_error := strconv.Parse_Unsigned_Integer("18446744073709551615", 10, 64)
	testify.No_Error(t, parse_error, "the unsigned maximum")
	testify.Equal(t, strconv.Unsigned_Integer(math.MaxUint64), widest)
	for _, text := range []strconv.Text{"", "_1", "1__0", "1_", "0x", "g", "-1", "0b2"} {
		_, reject := strconv.Parse_Unsigned_Integer(text, 0, 0)
		testify.Error(t, reject, "Parse_Unsigned_Integer(%q)", text)
	}
	_, explicit := strconv.Parse_Unsigned_Integer("1_0", 10, 0)
	testify.Error(t, explicit, "an explicit base rejects an underscore")
}

// Exercises Parse_Integer over both signs and both width extremes.
func signed_cases(t *testing.T) {
	for text, want := range map[strconv.Text]strconv.Signed_Integer{
		"0": 0, "1": 1, "2": 2, "-1": -1, "+7": 7, "-0x10": -16,
	} {
		value, parse_error := strconv.Parse_Integer(text, 0, 0)
		testify.No_Error(t, parse_error, "Parse_Integer(%q)", text)
		testify.Equal(t, want, value, "Parse_Integer(%q)", text)
	}
	// An explicit base ten takes the decimal reader, which no other case reaches now
	// that Parse_Decimal reads a short number on its own.
	single, _ := strconv.Parse_Integer("2", 10, 0)
	testify.Equal(t, strconv.Signed_Integer(2), single, "the decimal reader reads 2")
	pair, _ := strconv.Parse_Integer("12", 10, 0)
	testify.Equal(t, strconv.Signed_Integer(12), pair, "the decimal reader reads 12")
	smallest, _ := strconv.Parse_Integer("-9223372036854775808", 10, 64)
	testify.Equal(t, strconv.Signed_Integer(math.MinInt64), smallest)
	largest, _ := strconv.Parse_Integer("9223372036854775807", 10, 64)
	testify.Equal(t, strconv.Signed_Integer(math.MaxInt64), largest)
	base_36, _ := strconv.Parse_Integer("z", 36, 0)
	testify.Equal(t, strconv.Signed_Integer(35), base_36, "the digit z in base 36")
	binary, _ := strconv.Parse_Integer("11", 2, 0)
	testify.Equal(t, strconv.Signed_Integer(3), binary, "11 in base two")
	_, narrow := strconv.Parse_Integer("1", 10, 1)
	testify.Error_Is(t, narrow, strconv.Error_Range, "one bit cannot hold positive one")
	negative, _ := strconv.Parse_Integer("-1", 10, 2)
	testify.Equal(t, strconv.Signed_Integer(-1), negative)
	_, overflow := strconv.Parse_Integer("128", 10, 8)
	testify.Error_Is(t, overflow, strconv.Error_Range, "eight bits cannot hold 128")
	byte_minimum, _ := strconv.Parse_Integer("-128", 10, 8)
	testify.Equal(t, strconv.Signed_Integer(-128), byte_minimum)
	for _, text := range []strconv.Text{"", "-", "+", "-g"} {
		_, reject := strconv.Parse_Integer(text, 10, 0)
		testify.Error_Is(t, reject, strconv.Error_Syntax, "Parse_Integer(%q)", text)
	}
}

// Drives the digit reader and the separator rule over the whole byte domain.
func byte_domain_cases(t *testing.T) {
	for _, text := range []strconv.Text{"\x00", "\x01", "\x02", "\xff", "0\x001",
		"0\x011", "0\x021", "0\xff1", "_\x00\x01\x02\xff"} {
		_, reject := strconv.Parse_Unsigned_Integer(text, 0, 0)
		testify.Error(t, reject, "Parse_Unsigned_Integer(%q)", text)
	}
	for _, remainder := range []byte{0, 1, 2, 0xff} {
		text := strconv.Text([]byte{'1', '_', '1', remainder})
		_, reject := strconv.Parse_Unsigned_Integer(text, 0, 0)
		testify.Error(t, reject, "the remaining byte %d", remainder)
	}
	separated, _ := strconv.Parse_Unsigned_Integer("1_1", 0, 0)
	testify.Equal(t, strconv.Unsigned_Integer(11), separated,
		"an underscore between two digits")
	_, lone := strconv.Parse_Unsigned_Integer("_", 0, 0)
	testify.Error(t, lone, "a lone underscore")
	hexadecimal, _ := strconv.Parse_Unsigned_Integer("0x1_1", 0, 0)
	testify.Equal(t, strconv.Unsigned_Integer(17), hexadecimal,
		"an underscore inside hexadecimal digits")
	after_zero, _ := strconv.Parse_Unsigned_Integer("0_1", 0, 0)
	testify.Equal(t, strconv.Unsigned_Integer(1), after_zero,
		"an underscore after a zero prefix")
}

// Exercises Parse_Decimal over both machine integer extremes.
func decimal_cases(t *testing.T) {
	for text, want := range map[strconv.Text]strconv.Machine_Integer{
		"0": 0, "1": 1, "2": 2, "-1": -1,
	} {
		value, parse_error := strconv.Parse_Decimal(text)
		testify.No_Error(t, parse_error, "Parse_Decimal(%q)", text)
		testify.Equal(t, want, value, "Parse_Decimal(%q)", text)
	}
	smallest, smallest_error := strconv.Parse_Decimal("-9223372036854775808")
	testify.No_Error(t, smallest_error, "the machine minimum")
	testify.Equal(t, strconv.Machine_Integer(math.MinInt), smallest)
	largest, largest_error := strconv.Parse_Decimal("9223372036854775807")
	testify.No_Error(t, largest_error, "the machine maximum")
	testify.Equal(t, strconv.Machine_Integer(math.MaxInt), largest)
	_, reject := strconv.Parse_Decimal("")
	testify.Error_Is(t, reject, strconv.Error_Syntax, "Parse_Decimal of empty text")
}

// Exercises Format_Decimal over both machine integer extremes.
func decimal_text_cases(t *testing.T) {
	for value, want := range map[strconv.Machine_Integer]strconv.Decimal_Text{
		0: "0", 1: "1", 2: "2", -1: "-1",
	} {
		testify.Equal(t, want, strconv.Format_Decimal(value), "Format_Decimal(%d)", value)
	}
	testify.Equal(t, strconv.Decimal_Text("9223372036854775807"),
		strconv.Format_Decimal(math.MaxInt))
	testify.Equal(t, strconv.DECIMAL_TEXT_SIZE_MAXIMUM,
		len(strconv.Format_Decimal(math.MinInt)), "the machine minimum in base ten")
}

// Exercises both append forms over every buffer size and both width extremes.
func append_integer_cases(t *testing.T) {
	testify.Equal(t, "0", string(strconv.Append_Integer(nil, 0, 10)))
	testify.Equal(t, "1", string(strconv.Append_Integer(nil, 1, 10)))
	testify.Equal(t, "-1", string(strconv.Append_Integer(nil, -1, 10)))
	testify.Equal(t, "2", string(strconv.Append_Integer(nil, 2, 10)))
	testify.Equal(t, "1y2p0ij32e8e7",
		string(strconv.Append_Integer(nil, math.MaxInt64, 36)))
	testify.Equal(t, "0", string(strconv.Append_Unsigned_Integer(nil, 0, 10)))
	testify.Equal(t, "1", string(strconv.Append_Unsigned_Integer(nil, 1, 10)))
	testify.Equal(t, "10", string(strconv.Append_Unsigned_Integer(nil, 10, 10)))
	testify.Equal(t, "2", string(strconv.Append_Unsigned_Integer(nil, 2, 36)))
	for _, size := range buffer_sizes() {
		signed := strconv.Append_Integer(buffer_of(size), math.MinInt64, 2)
		testify.Equal(t, size+strconv.INTEGER_TEXT_SIZE_MAXIMUM, len(signed),
			"Append_Integer over %d bytes", size)
		unsigned := strconv.Append_Unsigned_Integer(buffer_of(size), math.MaxUint64, 2)
		testify.Equal(t, size+strconv.DIGIT_TEXT_SIZE_MAXIMUM, len(unsigned),
			"Append_Unsigned_Integer over %d bytes", size)
	}
}

// Exercises the three character forms over the whole Character domain.
func quote_rune_cases(t *testing.T) {
	testify.Equal(t, strconv.Character_Text(`'a'`), strconv.Quote_Rune('a'))
	testify.Equal(t, strconv.Character_Text(`'\''`), strconv.Quote_Rune('\''))
	testify.Equal(t, strconv.Character_Text(`'☺'`), strconv.Quote_Rune('☺'))
	testify.Equal(t, strconv.Character_Text(`'\U0010ffff'`),
		strconv.Quote_Rune(0x10ffff))
	separator := strconv.Quote_Rune(0x2028)
	testify.Equal(t, strconv.SHORT_UNICODE_DIGIT_COUNT+4, len(separator),
		"a line separator")
	testify.Equal(t, strconv.Character_Text(`'\u263a'`),
		strconv.Quote_Rune_To_ASCII('☺'))
	testify.Equal(t, strconv.Character_Text(`'\n'`), strconv.Quote_Rune_To_Graphic('\n'))
	testify.Equal(t, strconv.Character_Text("' '"), strconv.Quote_Rune_To_Graphic(' '))
	for _, value := range []strconv.Character{
		0, 1, 2, -1, math.MinInt32, math.MaxInt32, 0x10ffff,
	} {
		testify.True(t, len(strconv.Quote_Rune(value)) >=
			strconv.CHARACTER_TEXT_SIZE_MINIMUM, "Quote_Rune(%d)", value)
		testify.True(t, len(strconv.Quote_Rune_To_ASCII(value)) >=
			strconv.CHARACTER_TEXT_SIZE_MINIMUM, "Quote_Rune_To_ASCII(%d)", value)
		testify.True(t, len(strconv.Quote_Rune_To_Graphic(value)) >=
			strconv.CHARACTER_TEXT_SIZE_MINIMUM, "Quote_Rune_To_Graphic(%d)", value)
	}
}

// Exercises every append form of a literal.
func append_quote_cases(t *testing.T) {
	testify.Equal(t, `""`, string(strconv.Append_Quote(nil, "")))
	testify.Equal(t, `x"a"`, string(strconv.Append_Quote(strconv.Buffer("x"), "a")))
	testify.Equal(t, `'a'`, string(strconv.Append_Quote_Rune(nil, 'a')))
	testify.Equal(t, `'a'`, string(strconv.Append_Quote_Rune_To_ASCII(nil, 'a')))
	testify.Equal(t, `'a'`, string(strconv.Append_Quote_Rune_To_Graphic(nil, 'a')))
	testify.Equal(t, `xx"ab"`, string(strconv.Append_Quote(buffer_of(2), "ab")))
	for _, text := range []strconv.Text{"", "a", "ab"} {
		ascii := strconv.Append_Quote_To_ASCII(nil, text)
		testify.Equal(t, len(text)+2, len(ascii), "Append_Quote_To_ASCII(nil, %q)", text)
		graphic := strconv.Append_Quote_To_Graphic(nil, text)
		testify.Equal(t, len(text)+2, len(graphic),
			"Append_Quote_To_Graphic(nil, %q)", text)
	}
	for _, value := range []strconv.Character{
		0, 1, 2, -1, math.MinInt32, math.MaxInt32, 0x10ffff,
	} {
		testify.True(t, len(strconv.Append_Quote_Rune(nil, value)) >=
			strconv.CHARACTER_TEXT_SIZE_MINIMUM, "Append_Quote_Rune(%d)", value)
		testify.True(t, len(strconv.Append_Quote_Rune_To_ASCII(nil, value)) >=
			strconv.CHARACTER_TEXT_SIZE_MINIMUM,
			"Append_Quote_Rune_To_ASCII(%d)", value)
		testify.True(t, len(strconv.Append_Quote_Rune_To_Graphic(nil, value)) >=
			strconv.CHARACTER_TEXT_SIZE_MINIMUM,
			"Append_Quote_Rune_To_Graphic(%d)", value)
	}
	append_quote_size_cases(t)
}

// Exercises every append form of a literal over every buffer size.
func append_quote_size_cases(t *testing.T) {
	widest := strconv.Append_Quote_Rune_To_Graphic(
		buffer_of(strconv.BUFFER_SIZE_MAXIMUM), 0x10ffff,
	)
	testify.Equal(t, strconv.CHARACTER_BUFFER_SIZE_MAXIMUM, len(widest),
		"Append_Quote_Rune_To_Graphic at the buffer size limit")
	for _, size := range buffer_sizes() {
		ascii := strconv.Append_Quote_To_ASCII(buffer_of(size), "é")
		testify.Equal(t, size+8, len(ascii), "Append_Quote_To_ASCII over %d bytes", size)
		graphic := strconv.Append_Quote_To_Graphic(buffer_of(size), "a")
		testify.Equal(t, size+3, len(graphic),
			"Append_Quote_To_Graphic over %d bytes", size)
		plain := strconv.Append_Quote_Rune(buffer_of(size), 0x10ffff)
		testify.Equal(t, size+strconv.CHARACTER_TEXT_SIZE_MAXIMUM, len(plain),
			"Append_Quote_Rune over %d bytes", size)
		limited := strconv.Append_Quote_Rune_To_ASCII(buffer_of(size), 0x10ffff)
		testify.Equal(t, size+strconv.CHARACTER_TEXT_SIZE_MAXIMUM, len(limited),
			"Append_Quote_Rune_To_ASCII over %d bytes", size)
		visible := strconv.Append_Quote_Rune_To_Graphic(buffer_of(size), 'a')
		testify.Equal(t, size+3, len(visible),
			"Append_Quote_Rune_To_Graphic over %d bytes", size)
	}
}

// Exercises Unquote over each literal form and the text it rejects.
func unquote_cases(t *testing.T) {
	for text, want := range map[strconv.Text]strconv.Body_Text{
		`""`: "", `"a"`: "a", `"ab"`: "ab", `"a\tb"`: "a\tb", `"\x41"`: "A",
		`"\101"`: "A", `"\000"`: "\x00", `"\001"`: "\x01", `"\002"`: "\x02",
		`"\377"`: "\xff", `"\x00"`: "\x00", `"\x01"`: "\x01", `"\x02"`: "\x02",
		`"\xff"`: "\xff", `"☺"`: "☺", `"\U0010ffff"`: "\U0010ffff",
		"`raw\\n`": `raw\n`, "``": "", "`a`": "a", "`ab`": "ab", "`a\rb`": "ab",
		`'a'`: "a", `'\''`: "'",
		`"\a\b\f\n\r\t\v\\"`: "\a\b\f\n\r\t\v\\", `"\""`: `"`, `"\n\n"`: "\n\n",
	} {
		value, unquote_error := strconv.Unquote(text)
		testify.No_Error(t, unquote_error, "Unquote(%s)", text)
		testify.Equal(t, want, value, "Unquote(%s)", text)
	}
	for _, text := range []strconv.Text{
		"", `"`, `"a`, `'ab'`, `"a"b`, "\"\n\"", `"\q"`, `"\x4"`, `"\400"`, `"\08"`,
		"x", `"\u00"`, `"\ud800"`, "`a", `"'"a`,
	} {
		_, reject := strconv.Unquote(text)
		testify.Error_Is(t, reject, strconv.Error_Syntax, "Unquote(%s)", text)
	}
}

// Exercises Quoted_Prefix over each literal form and each remainder size.
func quoted_prefix_cases(t *testing.T) {
	for text, want := range map[strconv.Text]strconv.Prefix_Text{
		`""`: `""`, `""a`: `""`, `""ab`: `""`, `"a" rest`: `"a"`,
		"``x": "``", "`a`bc": "`a`", `'a'b`: `'a'`, `"\x41"z`: `"\x41"`,
	} {
		prefix, prefix_error := strconv.Quoted_Prefix(text)
		testify.No_Error(t, prefix_error, "Quoted_Prefix(%s)", text)
		testify.Equal(t, want, prefix, "Quoted_Prefix(%s)", text)
	}
	for _, text := range []strconv.Text{"", `"`, "x", "`a", `"a`} {
		_, reject := strconv.Quoted_Prefix(text)
		testify.Error_Is(t, reject, strconv.Error_Syntax, "Quoted_Prefix(%s)", text)
	}
}

// Exercises Unquote_Character over each quote mark, each remainder size, and the whole
// byte domain of an escape letter.
func unquote_character_cases(t *testing.T) {
	value, multibyte, tail, character_error := strconv.Unquote_Character("☺!", '"')
	testify.No_Error(t, character_error, "Unquote_Character of a symbol")
	testify.Equal(t, strconv.Code_Point('☺'), value)
	testify.True(t, bool(multibyte), "a symbol is a multibyte character")
	testify.Equal(t, strconv.Tail_Text("!"), tail)
	for _, text := range []strconv.Text{"a", "ab", "abc"} {
		_, _, rest, plain_error := strconv.Unquote_Character(text, 0)
		testify.No_Error(t, plain_error, "Unquote_Character(%q)", text)
		testify.Equal(t, len(text)-1, len(rest), "Unquote_Character(%q)", text)
	}
	_, _, _, empty := strconv.Unquote_Character("", 0)
	testify.Error_Is(t, empty, strconv.Error_Syntax, "Unquote_Character of empty text")
	_, _, _, double := strconv.Unquote_Character(`"`, '"')
	testify.Error_Is(t, double, strconv.Error_Syntax,
		"a double-quoted body escapes its own quote mark")
	_, _, _, single := strconv.Unquote_Character(`'`, '\'')
	testify.Error_Is(t, single, strconv.Error_Syntax,
		"a single-quoted body escapes its own quote mark")
	_, _, _, other := strconv.Unquote_Character(`\'`, '"')
	testify.Error_Is(t, other, strconv.Error_Syntax,
		"a double-quoted body rejects the other quote escape")
	quote, _, _, quote_error := strconv.Unquote_Character(`"`, 0)
	testify.No_Error(t, quote_error, "a bodiless form admits both quote marks")
	testify.Equal(t, strconv.Code_Point('"'), quote)
}

// Drives every escape letter and every escape width.
func escape_cases(t *testing.T) {
	for _, first := range []byte{0, 1, 2, 0xff, 'q', '8', '9'} {
		text := strconv.Text([]byte{'\\', first})
		_, _, _, reject := strconv.Unquote_Character(text, 0)
		testify.Error_Is(t, reject, strconv.Error_Syntax, "the escape letter %d", first)
	}
	for _, text := range []strconv.Text{`\`, `\x`, `\x4`, `\u12`, `\U1234`, `\0`, `\01`} {
		_, _, _, reject := strconv.Unquote_Character(text, 0)
		testify.Error_Is(t, reject, strconv.Error_Syntax, "the short escape %s", text)
	}
	for _, text := range []strconv.Text{`\x0g`, `\uzzzz`, `\018`, `\778`} {
		_, _, _, reject := strconv.Unquote_Character(text, 0)
		testify.Error_Is(t, reject, strconv.Error_Syntax, "the invalid escape %s", text)
	}
	for _, text := range []strconv.Text{`\x41`, `\x41a`, `\x41ab`, `\101`, `\101a`,
		`\101ab`, `\n`, `\na`, `\nab`} {
		_, _, _, escape_error := strconv.Unquote_Character(text, 0)
		testify.No_Error(t, escape_error, "the escape %s", text)
	}
	for _, digit := range []byte{0, 1, 2, 0xff} {
		text := strconv.Text([]byte{'\\', 'x', digit, digit})
		_, _, _, reject := strconv.Unquote_Character(text, 0)
		testify.Error_Is(t, reject, strconv.Error_Syntax,
			"the hexadecimal digit %d", digit)
	}
	fifteen, _, _, _ := strconv.Unquote_Character(`\x0f`, 0)
	testify.Equal(t, strconv.Code_Point(15), fifteen, "the hexadecimal digit maximum")
	one, _, _, _ := strconv.Unquote_Character(`\x01`, 0)
	testify.Equal(t, strconv.Code_Point(1), one, "the hexadecimal digit one")
	two, _, _, _ := strconv.Unquote_Character(`\x02`, 0)
	testify.Equal(t, strconv.Code_Point(2), two, "the hexadecimal digit two")
	byte_maximum, _, _, _ := strconv.Unquote_Character(`\377`, 0)
	testify.Equal(t, strconv.Code_Point(0xff), byte_maximum, "the octal byte maximum")
	zero, _, _, _ := strconv.Unquote_Character(`\000`, 0)
	testify.Equal(t, strconv.Code_Point(0), zero, "the octal zero")
}

// Drives every reader with a text of the maximum size.
func full_size_reads(t *testing.T, full strconv.Text) {
	_, boolean_error := strconv.Parse_Boolean(full)
	testify.Error(t, boolean_error, "a full text is not a Boolean")
	_, unsigned_error := strconv.Parse_Unsigned_Integer(full, 0, 0)
	testify.Error_Is(t, unsigned_error, strconv.Error_Range, "a full text of digits")
	_, signed_error := strconv.Parse_Integer(full, 10, 0)
	testify.Error_Is(t, signed_error, strconv.Error_Range, "a full text of digits")
	_, decimal_error := strconv.Parse_Decimal(full)
	testify.Error_Is(t, decimal_error, strconv.Error_Range, "a full text of digits")
	separated := strconv.Text("1_" + strings.Repeat("1", strconv.TEXT_SIZE_MAXIMUM-2))
	_, separated_error := strconv.Parse_Unsigned_Integer(separated, 0, 0)
	testify.Error_Is(t, separated_error, strconv.Error_Range, "a full separated text")
	_, unquote_error := strconv.Unquote(full)
	testify.Error_Is(t, unquote_error, strconv.Error_Syntax, "a full text of digits")
	_, prefix_error := strconv.Quoted_Prefix(full)
	testify.Error_Is(t, prefix_error, strconv.Error_Syntax, "a full text of digits")
	_, _, tail, character_error := strconv.Unquote_Character(full, 0)
	testify.No_Error(t, character_error, "a full text")
	testify.Equal(t, strconv.TAIL_TEXT_SIZE_MAXIMUM, len(tail),
		"one character of a full text")
}

// Drives every literal reader with a literal of the maximum size.
func full_size_literals(t *testing.T) {
	body := strings.Repeat("a", strconv.TEXT_SIZE_MAXIMUM-2)
	raw := strconv.Text("`" + body + "`")
	value, raw_error := strconv.Unquote(raw)
	testify.No_Error(t, raw_error, "a full raw literal")
	testify.Equal(t, strconv.BODY_TEXT_SIZE_MAXIMUM, len(value), "a full raw literal")
	prefix, prefix_error := strconv.Quoted_Prefix(raw)
	testify.No_Error(t, prefix_error, "a full raw literal prefix")
	testify.Equal(t, strconv.TEXT_SIZE_MAXIMUM, len(prefix), "a full raw literal prefix")
	quoted := strconv.Text(`"` + body + `"`)
	quoted_prefix, quoted_prefix_error := strconv.Quoted_Prefix(quoted)
	testify.No_Error(t, quoted_prefix_error, "a full literal prefix")
	testify.Equal(t, strconv.TEXT_SIZE_MAXIMUM, len(quoted_prefix),
		"a full literal prefix")
	interpreted, quoted_error := strconv.Unquote(quoted)
	testify.No_Error(t, quoted_error, "a full interpreted literal")
	testify.Equal(t, strconv.BODY_TEXT_SIZE_MAXIMUM, len(interpreted),
		"a full interpreted literal")
	// A body that holds a character above the ASCII range takes the decoder, thus only
	// such a body drives the decoder to its limits.
	accented := strconv.Text(`"` + strings.Repeat("é", (strconv.TEXT_SIZE_MAXIMUM-2)/2) +
		`"`)
	accented_value, accented_error := strconv.Unquote(accented)
	testify.No_Error(t, accented_error, "a full accented literal")
	testify.Equal(t, strconv.BODY_TEXT_SIZE_MAXIMUM, len(accented_value),
		"a full accented literal")
	single_byte := strconv.Text("\"\x80\"")
	single_byte_value, single_byte_error := strconv.Unquote(single_byte)
	testify.No_Error(t, single_byte_error, "a body of one byte that is not valid text")
	testify.Equal(t, "�", string(single_byte_value),
		"a byte that is not valid text reads as the replacement character")
	// A literal that holds an escape takes the reader that walks each character, thus
	// only such a literal drives that reader to its limits.
	escaped_literal := strconv.Text(
		`"\n` + strings.Repeat("a", strconv.TEXT_SIZE_MAXIMUM-4) + `"`,
	)
	escaped_prefix, escaped_prefix_error := strconv.Quoted_Prefix(escaped_literal)
	testify.No_Error(t, escaped_prefix_error, "a full escaped literal prefix")
	testify.Equal(t, strconv.TEXT_SIZE_MAXIMUM, len(escaped_prefix),
		"a full escaped literal prefix")
	unterminated := strconv.Text(
		`"` + strings.Repeat("a", strconv.TEXT_SIZE_MAXIMUM-1),
	)
	_, unterminated_error := strconv.Quoted_Prefix(unterminated)
	testify.Error_Is(t, unterminated_error, strconv.Error_Syntax,
		"a full unterminated literal")
	escape := strconv.Text("\\n" + strings.Repeat("a", strconv.TEXT_SIZE_MAXIMUM-2))
	_, _, _, escape_error := strconv.Unquote_Character(escape, 0)
	testify.No_Error(t, escape_error, "a full escape body")
	hexadecimal := strconv.Text(
		"\\x41" + strings.Repeat("a", strconv.TEXT_SIZE_MAXIMUM-4),
	)
	_, _, _, hexadecimal_error := strconv.Unquote_Character(hexadecimal, 0)
	testify.No_Error(t, hexadecimal_error, "a full hexadecimal body")
	octal := strconv.Text("\\101" + strings.Repeat("a", strconv.TEXT_SIZE_MAXIMUM-4))
	_, _, _, octal_error := strconv.Unquote_Character(octal, 0)
	testify.No_Error(t, octal_error, "a full octal body")
	filler := strings.Repeat("a", strconv.TEXT_SIZE_MAXIMUM-2)
	remainder := strconv.Text(`""` + filler)
	_, remainder_error := strconv.Quoted_Prefix(remainder)
	testify.No_Error(t, remainder_error, "a full remainder")
	raw_remainder := strconv.Text("``" + filler)
	_, raw_remainder_error := strconv.Quoted_Prefix(raw_remainder)
	testify.No_Error(t, raw_remainder_error, "a full raw remainder")
}

// Exercises the reader over the fraction it keeps, the fraction it drops, and the whole
// part it rejects.
func fixed_point_limit_cases(t *testing.T) {
	rounded, rounded_error := strconv.Parse_Fixed_Point("0.9999999999")
	testify.No_Error(t, rounded_error, "a fraction of nines")
	testify.Equal(t, fixedpoint.Number(fixedpoint.SCALE), rounded,
		"a fraction of nines rounds to one whole")
	_, unsigned_maximum := strconv.Parse_Fixed_Point("18446744073709551615")
	testify.Error_Is(t, unsigned_maximum, strconv.Error_Range,
		"the unsigned maximum exceeds the fixed-point domain")
	_, above := strconv.Parse_Fixed_Point("18446744073709551616")
	testify.Error_Is(t, above, strconv.Error_Range,
		"a whole part above the unsigned maximum")
	nine_digits, nine_error := strconv.Parse_Fixed_Point("0.100000000")
	testify.No_Error(t, nine_error, "nine fraction digits")
	tenth, tenth_error := strconv.Parse_Fixed_Point("0.1")
	testify.No_Error(t, tenth_error, "one fraction digit")
	testify.Equal(t, tenth, nine_digits, "nine fraction digits give one tenth")
	full := strconv.Text("." + strings.Repeat("1", strconv.TEXT_SIZE_MAXIMUM-1))
	_, full_error := strconv.Parse_Fixed_Point(full)
	testify.No_Error(t, full_error, "a full fraction")
}

// Exercises the two written forms of a fixed-point number over every digit count and
// every buffer size.
func fixed_point_text_cases(t *testing.T) {
	whole := fixedpoint.From_Integer(5)
	for digits, want := range map[fixedpoint.Digit_Count]fixedpoint.Text{
		0: "5", 1: "5.0", 2: "5.00", 6: "5.000000",
	} {
		text := strconv.Format_Fixed_Point(fixedpoint.Number(whole), digits)
		testify.Equal(t, want, text, "Format_Fixed_Point(5, %d)", digits)
	}
	testify.Equal(t, fixedpoint.Text("0"), strconv.Format_Fixed_Point(0, 0))
	testify.Equal(t, fixedpoint.Text("-1"),
		strconv.Format_Fixed_Point(-fixedpoint.SCALE, 0))
	widest := strconv.Format_Fixed_Point(math.MinInt64, 6)
	testify.Equal(t, strconv.FIXED_POINT_TEXT_SIZE_MAXIMUM, len(widest),
		"the fixed-point minimum at six digits")
	round_trip, round_trip_error := strconv.Parse_Fixed_Point(
		strconv.Text(strconv.Format_Fixed_Point(fixedpoint.Number(whole), 6)),
	)
	testify.No_Error(t, round_trip_error, "the round trip")
	testify.Equal(t, fixedpoint.Number(whole), round_trip, "the round trip")
	for _, size := range buffer_sizes() {
		appended := strconv.Append_Fixed_Point(buffer_of(size), math.MinInt64, 6)
		testify.Equal(t, size+strconv.FIXED_POINT_TEXT_SIZE_MAXIMUM, len(appended),
			"Append_Fixed_Point over %d bytes", size)
	}
	testify.Equal(t, "0", string(strconv.Append_Fixed_Point(nil, 0, 0)))
	testify.Equal(t, "-1",
		string(strconv.Append_Fixed_Point(nil, -fixedpoint.SCALE, 0)))
	for _, value := range []fixedpoint.Number{math.MaxInt64, 1, 2, -1} {
		for _, digits := range []fixedpoint.Digit_Count{0, 1, 2, 6} {
			text := strconv.Format_Fixed_Point(value, digits)
			appended := strconv.Append_Fixed_Point(nil, value, digits)
			testify.Equal(t, string(text), string(appended),
				"Append_Fixed_Point(%d, %d)", value, digits)
		}
	}
}

// Returns the buffer sizes that the Buffer domain names.
func buffer_sizes() (sizes []int) {
	return []int{0, 1, 2, strconv.BUFFER_SIZE_MAXIMUM}
}

// Returns a buffer of the given size.
func buffer_of(size int) (buffer strconv.Buffer) {
	return strconv.Buffer(strings.Repeat("x", size))
}

// Returns text of the given size, every byte the same.
func text_of(size int, filler byte) (text strconv.Text) {
	return strconv.Text(strings.Repeat(string([]byte{filler}), size))
}
