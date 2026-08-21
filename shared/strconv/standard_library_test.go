package strconv

import (
	"testing"

	"local/james-orcales/shared/testify"
)

// The tables below hold the cases of the Go standard library conversion tests, which the
// BSD-3 license beside this file covers. Three kinds of case differ from the source: a
// case whose expected error is an invalid base or bit size states Panics, because this
// package asserts those two domains instead of reporting them; a float case is absent,
// because the deterministic tier admits no float type; and the reader of each table
// reads keyed fields, because the linter rejects a positional literal.

// One case of the standard library ParseUint table.
type unsigned_case struct {
	// Input is the text the reader takes.
	Input string
	// Output is the value the reader must give.
	Output uint64
	// Error is the error the reader must report.
	Error error
	// Panics marks a case that leaves the asserted domain of this package.
	Panics bool
}

// One case of the standard library ParseUint table of explicit bases.
type unsigned_base_case struct {
	// Input is the text the reader takes.
	Input string
	// Base is the radix the caller states.
	Base int
	// Output is the value the reader must give.
	Output uint64
	// Error is the error the reader must report.
	Error error
	// Panics marks a case that leaves the asserted domain of this package.
	Panics bool
}

// One case of the standard library ParseInt table.
type signed_case struct {
	// Input is the text the reader takes.
	Input string
	// Output is the value the reader must give.
	Output int64
	// Error is the error the reader must report.
	Error error
	// Panics marks a case that leaves the asserted domain of this package.
	Panics bool
}

// One case of the standard library ParseInt table of explicit bases.
type signed_base_case struct {
	// Input is the text the reader takes.
	Input string
	// Base is the radix the caller states.
	Base int
	// Output is the value the reader must give.
	Output int64
	// Error is the error the reader must report.
	Error error
	// Panics marks a case that leaves the asserted domain of this package.
	Panics bool
}

// One case of the standard library 32-bit ParseUint table.
type unsigned_narrow_case struct {
	// Input is the text the reader takes.
	Input string
	// Output is the value the reader must give.
	Output uint32
	// Error is the error the reader must report.
	Error error
	// Panics marks a case that leaves the asserted domain of this package.
	Panics bool
}

// One case of the standard library 32-bit ParseInt table.
type signed_narrow_case struct {
	// Input is the text the reader takes.
	Input string
	// Output is the value the reader must give.
	Output int32
	// Error is the error the reader must report.
	Error error
	// Panics marks a case that leaves the asserted domain of this package.
	Panics bool
}

// One case of the standard library FormatInt table.
type signed_text_case struct {
	// Input is the value the writer takes.
	Input int64
	// Base is the radix the caller states.
	Base int
	// Output is the text the writer must give.
	Output string
}

// One case of the standard library FormatUint table.
type unsigned_text_case struct {
	// Input is the value the writer takes.
	Input uint64
	// Base is the radix the caller states.
	Base int
	// Output is the text the writer must give.
	Output string
}

// One case of the standard library ParseBool table.
type boolean_case struct {
	// Input is the text the reader takes.
	Input string
	// Output is the value the reader must give.
	Output bool
	// Error is the error the reader must report.
	Error error
	// Panics marks a case that leaves the asserted domain of this package.
	Panics bool
}

// One case of the standard library Quote table.
type quote_case struct {
	// Input is the text the writer takes.
	Input string
	// Output is the literal Quote must give.
	Output string
	// ASCII_Output is the literal Quote_To_ASCII must give.
	ASCII_Output string
	// Graphic_Output is the literal Quote_To_Graphic must give.
	Graphic_Output string
}

// One case of the standard library QuoteRune table.
type quote_rune_case struct {
	// Input is the character the writer takes.
	Input rune
	// Output is the literal Quote_Rune must give.
	Output string
	// ASCII_Output is the literal Quote_Rune_To_ASCII must give.
	ASCII_Output string
	// Graphic_Output is the literal Quote_Rune_To_Graphic must give.
	Graphic_Output string
}

// One case of the standard library CanBackquote table.
type backquote_case struct {
	// Input is the text the reader takes.
	Input string
	// Output is the report the reader must give.
	Output bool
}

// One case of the standard library FormatUint table of every decimal width.
type varlen_case struct {
	// Input is the value the writer takes.
	Input uint64
	// Output is the text the writer must give.
	Output string
}

// One case of the standard library Unquote table.
type unquote_case struct {
	// Input is the literal the reader takes.
	Input string
	// Output is the value the reader must give.
	Output string
}

// Test_Standard_Library_Parse_Unsigned reads the standard library ParseUint table.
func Test_Standard_Library_Parse_Unsigned(t *testing.T) {
	t.Parallel()
	cases := unsigned_cases()
	for _, one := range cases {
		value, parse_error := Parse_Unsigned_Integer(Text(one.Input), 10, 64)
		check_error(t, parse_error, one.Error, "Parse_Unsigned_Integer", one.Input)
		testify.Equal(t, one.Output, uint64(value),
			"Parse_Unsigned_Integer(%q)", one.Input)
	}
	// The standard library reads the same table at the machine width, where the widest
	// case still fits because this repository builds for 64-bit targets.
	for _, one := range cases {
		value, parse_error := Parse_Unsigned_Integer(Text(one.Input), 10, 0)
		check_error(t, parse_error, one.Error, "Parse_Unsigned_Integer", one.Input)
		testify.Equal(t, one.Output, uint64(value),
			"Parse_Unsigned_Integer(%q, 10, 0)", one.Input)
	}
}

// Test_Standard_Library_Parse_Unsigned_Base reads the standard library ParseUint table
// of explicit bases.
func Test_Standard_Library_Parse_Unsigned_Base(t *testing.T) {
	t.Parallel()
	cases := unsigned_base_cases()
	for _, one := range cases {
		if one.Panics {
			domain_case(t, one.Input, one.Base)
			continue
		}
		value, parse_error := Parse_Unsigned_Integer(
			Text(one.Input), Implied_Base(one.Base), 64,
		)
		check_error(t, parse_error, one.Error, "Parse_Unsigned_Integer", one.Input)
		testify.Equal(t, one.Output, uint64(value),
			"Parse_Unsigned_Integer(%q, %d)", one.Input, one.Base)
	}
}

// Test_Standard_Library_Parse_Signed reads the standard library ParseInt table,
// which is also the Atoi table because Atoi is base ten at the machine width.
func Test_Standard_Library_Parse_Signed(t *testing.T) {
	t.Parallel()
	cases := signed_cases()
	standard_library_decimal_cases(t, cases)
	for _, one := range cases {
		value, parse_error := Parse_Integer(Text(one.Input), 10, 64)
		check_error(t, parse_error, one.Error, "Parse_Integer", one.Input)
		testify.Equal(t, one.Output, int64(value), "Parse_Integer(%q)", one.Input)
	}
	// The standard library reads the same table at the machine width.
	for _, one := range cases {
		value, parse_error := Parse_Integer(Text(one.Input), 10, 0)
		check_error(t, parse_error, one.Error, "Parse_Integer", one.Input)
		testify.Equal(t, one.Output, int64(value),
			"Parse_Integer(%q, 10, 0)", one.Input)
	}
}

// Test_Standard_Library_Parse_Signed_Base reads the standard library ParseInt table of
// explicit bases.
func Test_Standard_Library_Parse_Signed_Base(t *testing.T) {
	t.Parallel()
	cases := signed_base_cases()
	for _, one := range cases {
		if one.Panics {
			domain_case(t, one.Input, one.Base)
			continue
		}
		value, parse_error := Parse_Integer(Text(one.Input), Implied_Base(one.Base), 64)
		check_error(t, parse_error, one.Error, "Parse_Integer", one.Input)
		testify.Equal(t, one.Output, int64(value),
			"Parse_Integer(%q, %d)", one.Input, one.Base)
	}
}

// Test_Standard_Library_Parse_Narrow reads the standard library 32-bit tables.
func Test_Standard_Library_Parse_Narrow(t *testing.T) {
	t.Parallel()
	for _, one := range unsigned_narrow_cases() {
		value, parse_error := Parse_Unsigned_Integer(Text(one.Input), 10, 32)
		check_error(t, parse_error, one.Error, "Parse_Unsigned_Integer", one.Input)
		testify.Equal(t, one.Output, uint32(value),
			"Parse_Unsigned_Integer(%q, 10, 32)", one.Input)
	}
	for _, one := range signed_narrow_cases() {
		value, parse_error := Parse_Integer(Text(one.Input), 10, 32)
		check_error(t, parse_error, one.Error, "Parse_Integer", one.Input)
		testify.Equal(t, one.Output, int32(value),
			"Parse_Integer(%q, 10, 32)", one.Input)
	}
}

// Reads the Atoi cases of the standard library signed table.
func standard_library_decimal_cases(t *testing.T, cases []signed_case) {
	for _, one := range cases {
		value, parse_error := Parse_Decimal(Text(one.Input))
		check_error(t, parse_error, one.Error, "Parse_Decimal", one.Input)
		testify.Equal(t, one.Output, int64(value), "Parse_Decimal(%q)", one.Input)
	}
}

// Test_Standard_Library_Format_Integer reads the standard library FormatInt and
// FormatUint tables.
func Test_Standard_Library_Format_Integer(t *testing.T) {
	t.Parallel()
	var signed_storage [INTEGER_TEXT_SIZE_MAXIMUM]byte
	for _, one := range signed_text_cases() {
		count := Format_Integer_Into(
			signed_storage[:], Signed_Integer(one.Input), Base(one.Base),
		)
		testify.Equal(t, one.Output, string(signed_storage[:int(count)]),
			"Format_Integer(%d, %d)", one.Input, one.Base)
	}
	var unsigned_storage [DIGIT_TEXT_SIZE_MAXIMUM]byte
	for _, one := range unsigned_text_cases() {
		count := Format_Unsigned_Integer_Into(
			unsigned_storage[:], Unsigned_Integer(one.Input), Base(one.Base),
		)
		testify.Equal(t, one.Output, string(unsigned_storage[:int(count)]),
			"Format_Unsigned_Integer(%d, %d)", one.Input, one.Base)
	}
}

// Test_Standard_Library_Boolean reads the standard library ParseBool table.
func Test_Standard_Library_Boolean(t *testing.T) {
	t.Parallel()
	for _, one := range boolean_cases() {
		value, parse_error := Parse_Boolean(Text(one.Input))
		check_error(t, parse_error, one.Error, "Parse_Boolean", one.Input)
		testify.Equal(t, one.Output, bool(value), "Parse_Boolean(%q)", one.Input)
	}
	var storage [BOOLEAN_TEXT_SIZE_FALSE]byte
	true_count := Format_Boolean_Into(storage[:], true)
	testify.Equal(t, "true", string(storage[:int(true_count)]))
	false_count := Format_Boolean_Into(storage[:], false)
	testify.Equal(t, "false", string(storage[:int(false_count)]))
}

// Test_Standard_Library_Quote reads the standard library Quote tables.
func Test_Standard_Library_Quote(t *testing.T) {
	t.Parallel()
	var storage [QUOTED_TEXT_SIZE_MAXIMUM]byte
	for _, one := range quote_cases() {
		count := Quote_Into(storage[:], Text(one.Input))
		testify.Equal(t, one.Output, string(storage[:int(count)]),
			"Quote(%q)", one.Input)
		count = Quote_To_ASCII_Into(storage[:], Text(one.Input))
		testify.Equal(t, one.ASCII_Output, string(storage[:int(count)]),
			"Quote_To_ASCII(%q)", one.Input)
		count = Quote_To_Graphic_Into(storage[:], Text(one.Input))
		testify.Equal(t, one.Graphic_Output, string(storage[:int(count)]),
			"Quote_To_Graphic(%q)", one.Input)
	}
}

// Test_Standard_Library_Quote_Rune reads the standard library QuoteRune tables.
func Test_Standard_Library_Quote_Rune(t *testing.T) {
	t.Parallel()
	var storage [CHARACTER_TEXT_SIZE_MAXIMUM]byte
	for _, one := range quote_rune_cases() {
		value := Character(one.Input)
		count := Quote_Rune_Into(storage[:], value)
		testify.Equal(t, one.Output, string(storage[:int(count)]),
			"Quote_Rune(%d)", one.Input)
		count = Quote_Rune_To_ASCII_Into(storage[:], value)
		testify.Equal(t, one.ASCII_Output, string(storage[:int(count)]),
			"Quote_Rune_To_ASCII(%d)", one.Input)
		count = Quote_Rune_To_Graphic_Into(storage[:], value)
		testify.Equal(t, one.Graphic_Output, string(storage[:int(count)]),
			"Quote_Rune_To_Graphic(%d)", one.Input)
	}
}

// Test_Standard_Library_Backquote reads the standard library CanBackquote table.
func Test_Standard_Library_Backquote(t *testing.T) {
	t.Parallel()
	for _, one := range backquote_cases() {
		testify.Equal(t, one.Output, bool(Can_Backquote(Text(one.Input))),
			"Can_Backquote(%q)", one.Input)
	}
}

// Test_Standard_Library_Unquote reads the standard library Unquote tables, including the
// literals it must reject.
func Test_Standard_Library_Unquote(t *testing.T) {
	t.Parallel()
	var storage [UNQUOTED_TEXT_SIZE_MAXIMUM]byte
	for _, one := range unquote_cases() {
		count, unquote_error := Unquote_Into(storage[:], Text(one.Input))
		testify.No_Error(t, unquote_error, "Unquote(%s)", one.Input)
		testify.Equal(t, one.Output, string(storage[:int(count)]), "Unquote(%s)", one.Input)
	}
	for _, one := range misquoted_cases() {
		count, unquote_error := Unquote_Into(storage[:], Text(one))
		testify.Equal(t, 0, int(count), "Unquote(%s)", one)
		testify.Error_Is(t, unquote_error, Error_Syntax, "Unquote(%s)", one)
	}
}

// Test_Standard_Library_Printability checks every boundary used by quoting decisions.
func Test_Standard_Library_Printability(t *testing.T) {
	t.Parallel()
	for _, one := range []struct {
		Point   rune
		Print   bool
		Graphic bool
	}{
		{Point: -1},
		{Point: 0},
		{Point: '\n'},
		{Point: ' ', Print: true, Graphic: true},
		{Point: '\u00a0', Graphic: true},
		{Point: 'A', Print: true, Graphic: true},
		{Point: '世', Print: true, Graphic: true},
		{Point: 0x0378},
		{Point: '\U0010ffff'},
		{Point: 0x110000},
	} {
		testify.Equal(t, one.Print, bool(Is_Print(Character(one.Point))),
			"Is_Print(%U)", one.Point)
		testify.Equal(t, one.Graphic, bool(Is_Graphic(Character(one.Point))),
			"Is_Graphic(%U)", one.Point)
	}
}

// Test_Standard_Library_Format_Varlen reads the standard library FormatUint table of
// every decimal width.
func Test_Standard_Library_Format_Varlen(t *testing.T) {
	t.Parallel()
	var storage [DIGIT_TEXT_SIZE_MAXIMUM]byte
	for _, one := range varlen_cases() {
		count := Format_Unsigned_Integer_Into(storage[:], Unsigned_Integer(one.Input), 10)
		testify.Equal(t, one.Output, string(storage[:int(count)]),
			"Format_Unsigned_Integer(%d, 10)", one.Input)
	}
}

// Test_Standard_Library_Unquote_Invalid_UTF8 reads the standard library cases of a
// literal that holds a byte which is not valid UTF-8.
func Test_Standard_Library_Unquote_Invalid_UTF8(t *testing.T) {
	t.Parallel()
	var storage [UNQUOTED_TEXT_SIZE_MAXIMUM]byte
	cases := make([]unquote_case, 0, 4)
	cases = append(cases,
		unquote_case{Input: "\"foo\"", Output: "foo"},
		unquote_case{Input: "\"\xc0\"", Output: "\xef\xbf\xbd"},
		unquote_case{Input: "\"a\xc0\"", Output: "a\xef\xbf\xbd"},
		unquote_case{Input: "\"\\t\xc0\"", Output: "\t\xef\xbf\xbd"},
	)
	for _, one := range cases {
		count, unquote_error := Unquote_Into(storage[:], Text(one.Input))
		testify.No_Error(t, unquote_error, "Unquote(%q)", one.Input)
		testify.Equal(t, one.Output, string(storage[:int(count)]), "Unquote(%q)", one.Input)
	}
	_, reject := Unquote_Into(storage[:], "\"foo")
	testify.Error_Is(t, reject, Error_Syntax, "an unterminated literal")
}

// Test_Standard_Library_Bit_Size reads the standard library bit size cases. The two
// legal widths convert, and the two illegal widths leave the domain this package
// asserts, thus they panic where the standard library reports an error.
func Test_Standard_Library_Bit_Size(t *testing.T) {
	t.Parallel()
	for _, size := range []Bit_Size{0, 64} {
		_, signed_error := Parse_Integer("0", 0, size)
		testify.No_Error(t, signed_error, "Parse_Integer(0, 0, %d)", size)
		_, unsigned_error := Parse_Unsigned_Integer("0", 0, size)
		testify.No_Error(t, unsigned_error, "Parse_Unsigned_Integer(0, 0, %d)", size)
	}
	for _, size := range []Bit_Size{-1, 65} {
		testify.Panics(t, func() { Parse_Integer("0", 0, size) },
			"a Bit_Size of %d", size)
	}
}

// Reports the error a case states, which is absent for a case that converts.
func check_error(t *testing.T, given error, wanted error, name string, input string) {
	t.Helper()
	if wanted == nil {
		testify.No_Error(t, given, "%s(%q)", name, input)
		return
	}
	testify.Error_Is(t, given, wanted, "%s(%q)", name, input)
}

// Reports that a base outside the asserted domain panics.
func domain_case(t *testing.T, text string, base int) {
	t.Helper()
	testify.Panics(t, func() { Parse_Integer(Text(text), Implied_Base(base), 64) },
		"Parse_Integer(%q, %d)", text, base)
}

// Writes part 1 of the standard library parseUint64Tests table into the cases.
func unsigned_cases_1(cases []unsigned_case) (extended []unsigned_case) {
	return append(cases,
		unsigned_case{Input: "", Output: 0, Error: Error_Syntax},
		unsigned_case{Input: "0", Output: 0},
		unsigned_case{Input: "1", Output: 1},
		unsigned_case{Input: "12345", Output: 12345},
		unsigned_case{Input: "012345", Output: 12345},
		unsigned_case{Input: "12345x", Output: 0, Error: Error_Syntax},
		unsigned_case{Input: "98765432100", Output: 98765432100},
		unsigned_case{Input: "18446744073709551615", Output: 1<<64 - 1},
		unsigned_case{
			Input:  "18446744073709551616",
			Output: 1<<64 - 1,
			Error:  Error_Range,
		},
		unsigned_case{
			Input:  "18446744073709551620",
			Output: 1<<64 - 1,
			Error:  Error_Range,
		},
		unsigned_case{Input: "1_2_3_4_5", Output: 0, Error: Error_Syntax},
		unsigned_case{Input: "_12345", Output: 0, Error: Error_Syntax},
		unsigned_case{Input: "1__2345", Output: 0, Error: Error_Syntax},
		unsigned_case{Input: "12345_", Output: 0, Error: Error_Syntax},
		unsigned_case{Input: "-0", Output: 0, Error: Error_Syntax},
		unsigned_case{Input: "-1", Output: 0, Error: Error_Syntax},
		unsigned_case{Input: "+1", Output: 0, Error: Error_Syntax},
	)
}

// Holds the whole standard library parseUint64Tests table. One allocation, sized to the
// case count, holds it.
func unsigned_cases() (cases []unsigned_case) {
	cases = make([]unsigned_case, 0, 17)
	cases = unsigned_cases_1(cases)
	return cases
}

// Writes part 1 of the standard library parseUint64BaseTests table into the cases.
func unsigned_base_cases_1(cases []unsigned_base_case) (extended []unsigned_base_case) {
	return append(cases,
		unsigned_base_case{Input: "", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "0", Base: 0, Output: 0},
		unsigned_base_case{Input: "0x", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "0X", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "1", Base: 0, Output: 1},
		unsigned_base_case{Input: "12345", Base: 0, Output: 12345},
		unsigned_base_case{Input: "012345", Base: 0, Output: 012345},
		unsigned_base_case{Input: "0x12345", Base: 0, Output: 0x12345},
		unsigned_base_case{Input: "0X12345", Base: 0, Output: 0x12345},
		unsigned_base_case{Input: "12345x", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{
			Input:  "0xabcdefg123",
			Base:   0,
			Output: 0,
			Error:  Error_Syntax,
		},
		unsigned_base_case{
			Input:  "123456789abc",
			Base:   0,
			Output: 0,
			Error:  Error_Syntax,
		},
		unsigned_base_case{Input: "98765432100", Base: 0, Output: 98765432100},
		unsigned_base_case{Input: "18446744073709551615", Base: 0, Output: 1<<64 - 1},
		unsigned_base_case{
			Input:  "18446744073709551616",
			Base:   0,
			Output: 1<<64 - 1,
			Error:  Error_Range,
		},
		unsigned_base_case{
			Input:  "18446744073709551620",
			Base:   0,
			Output: 1<<64 - 1,
			Error:  Error_Range,
		},
		unsigned_base_case{Input: "0xFFFFFFFFFFFFFFFF", Base: 0, Output: 1<<64 - 1},
		unsigned_base_case{
			Input:  "0x10000000000000000",
			Base:   0,
			Output: 1<<64 - 1,
			Error:  Error_Range,
		},
		unsigned_base_case{Input: "01777777777777777777777", Base: 0, Output: 1<<64 - 1},
		unsigned_base_case{
			Input:  "01777777777777777777778",
			Base:   0,
			Output: 0,
			Error:  Error_Syntax,
		},
		unsigned_base_case{
			Input:  "02000000000000000000000",
			Base:   0,
			Output: 1<<64 - 1,
			Error:  Error_Range,
		},
		unsigned_base_case{Input: "0200000000000000000000", Base: 0, Output: 1 << 61},
	)
}

// Writes part 2 of the standard library parseUint64BaseTests table into the cases.
func unsigned_base_cases_2(cases []unsigned_base_case) (extended []unsigned_base_case) {
	return append(cases,
		unsigned_base_case{Input: "0b", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "0B", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "0b101", Base: 0, Output: 5},
		unsigned_base_case{Input: "0B101", Base: 0, Output: 5},
		unsigned_base_case{Input: "0o", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "0O", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "0o377", Base: 0, Output: 255},
		unsigned_base_case{Input: "0O377", Base: 0, Output: 255},
		unsigned_base_case{Input: "1_2_3_4_5", Base: 0, Output: 12345},
		unsigned_base_case{Input: "_12345", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "1__2345", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "12345_", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "1_2_3_4_5", Base: 10, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "_12345", Base: 10, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "1__2345", Base: 10, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "12345_", Base: 10, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "0x_1_2_3_4_5", Base: 0, Output: 0x12345},
		unsigned_base_case{Input: "_0x12345", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "0x__12345", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "0x1__2345", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "0x1234__5", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "0x12345_", Base: 0, Output: 0, Error: Error_Syntax},
	)
}

// Writes part 3 of the standard library parseUint64BaseTests table into the cases.
func unsigned_base_cases_3(cases []unsigned_base_case) (extended []unsigned_base_case) {
	return append(cases,
		unsigned_base_case{Input: "1_2_3_4_5", Base: 16, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "_12345", Base: 16, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "1__2345", Base: 16, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "1234__5", Base: 16, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "12345_", Base: 16, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "0_1_2_3_4_5", Base: 0, Output: 012345},
		unsigned_base_case{Input: "_012345", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "0__12345", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "01234__5", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "012345_", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "0o_1_2_3_4_5", Base: 0, Output: 012345},
		unsigned_base_case{Input: "_0o12345", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "0o__12345", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "0o1234__5", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "0o12345_", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{
			Input:  "0_1_2_3_4_5",
			Base:   8,
			Output: 0,
			Error:  Error_Syntax,
		},
		unsigned_base_case{Input: "_012345", Base: 8, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "0__12345", Base: 8, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "01234__5", Base: 8, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "012345_", Base: 8, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "0b_1_0_1", Base: 0, Output: 5},
		unsigned_base_case{Input: "_0b101", Base: 0, Output: 0, Error: Error_Syntax},
	)
}

// Writes part 4 of the standard library parseUint64BaseTests table into the cases.
func unsigned_base_cases_4(cases []unsigned_base_case) (extended []unsigned_base_case) {
	return append(cases,
		unsigned_base_case{Input: "0b__101", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "0b1__01", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "0b10__1", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "0b101_", Base: 0, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "1_0_1", Base: 2, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "_101", Base: 2, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "1_01", Base: 2, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "10_1", Base: 2, Output: 0, Error: Error_Syntax},
		unsigned_base_case{Input: "101_", Base: 2, Output: 0, Error: Error_Syntax},
	)
}

// Holds the whole standard library parseUint64BaseTests table. One allocation, sized to the
// case count, holds it.
func unsigned_base_cases() (cases []unsigned_base_case) {
	cases = make([]unsigned_base_case, 0, 75)
	cases = unsigned_base_cases_1(cases)
	cases = unsigned_base_cases_2(cases)
	cases = unsigned_base_cases_3(cases)
	cases = unsigned_base_cases_4(cases)
	return cases
}

// Writes part 1 of the standard library parseInt64Tests table into the cases.
func signed_cases_1(cases []signed_case) (extended []signed_case) {
	return append(cases,
		signed_case{Input: "", Output: 0, Error: Error_Syntax},
		signed_case{Input: "0", Output: 0},
		signed_case{Input: "-0", Output: 0},
		signed_case{Input: "+0", Output: 0},
		signed_case{Input: "1", Output: 1},
		signed_case{Input: "-1", Output: -1},
		signed_case{Input: "+1", Output: 1},
		signed_case{Input: "12345", Output: 12345},
		signed_case{Input: "-12345", Output: -12345},
		signed_case{Input: "012345", Output: 12345},
		signed_case{Input: "-012345", Output: -12345},
		signed_case{Input: "98765432100", Output: 98765432100},
		signed_case{Input: "-98765432100", Output: -98765432100},
		signed_case{Input: "9223372036854775807", Output: 1<<63 - 1},
		signed_case{Input: "-9223372036854775807", Output: -(1<<63 - 1)},
		signed_case{Input: "9223372036854775808", Output: 1<<63 - 1, Error: Error_Range},
		signed_case{Input: "-9223372036854775808", Output: -1 << 63},
		signed_case{Input: "9223372036854775809", Output: 1<<63 - 1, Error: Error_Range},
		signed_case{Input: "-9223372036854775809", Output: -1 << 63, Error: Error_Range},
		signed_case{Input: "-1_2_3_4_5", Output: 0, Error: Error_Syntax},
		signed_case{Input: "-_12345", Output: 0, Error: Error_Syntax},
		signed_case{Input: "_12345", Output: 0, Error: Error_Syntax},
		signed_case{Input: "1__2345", Output: 0, Error: Error_Syntax},
		signed_case{Input: "12345_", Output: 0, Error: Error_Syntax},
		signed_case{Input: "123%45", Output: 0, Error: Error_Syntax},
	)
}

// Holds the whole standard library parseInt64Tests table. One allocation, sized to the
// case count, holds it.
func signed_cases() (cases []signed_case) {
	cases = make([]signed_case, 0, 25)
	cases = signed_cases_1(cases)
	return cases
}

// Writes part 1 of the standard library parseInt64BaseTests table into the cases.
func signed_base_cases_1(cases []signed_base_case) (extended []signed_base_case) {
	return append(cases,
		signed_base_case{Input: "", Base: 0, Output: 0, Error: Error_Syntax},
		signed_base_case{Input: "0", Base: 0, Output: 0},
		signed_base_case{Input: "-0", Base: 0, Output: 0},
		signed_base_case{Input: "1", Base: 0, Output: 1},
		signed_base_case{Input: "-1", Base: 0, Output: -1},
		signed_base_case{Input: "12345", Base: 0, Output: 12345},
		signed_base_case{Input: "-12345", Base: 0, Output: -12345},
		signed_base_case{Input: "012345", Base: 0, Output: 012345},
		signed_base_case{Input: "-012345", Base: 0, Output: -012345},
		signed_base_case{Input: "0x12345", Base: 0, Output: 0x12345},
		signed_base_case{Input: "-0X12345", Base: 0, Output: -0x12345},
		signed_base_case{Input: "12345x", Base: 0, Output: 0, Error: Error_Syntax},
		signed_base_case{Input: "-12345x", Base: 0, Output: 0, Error: Error_Syntax},
		signed_base_case{Input: "98765432100", Base: 0, Output: 98765432100},
		signed_base_case{Input: "-98765432100", Base: 0, Output: -98765432100},
		signed_base_case{Input: "9223372036854775807", Base: 0, Output: 1<<63 - 1},
		signed_base_case{Input: "-9223372036854775807", Base: 0, Output: -(1<<63 - 1)},
		signed_base_case{
			Input:  "9223372036854775808",
			Base:   0,
			Output: 1<<63 - 1,
			Error:  Error_Range,
		},
		signed_base_case{Input: "-9223372036854775808", Base: 0, Output: -1 << 63},
		signed_base_case{
			Input:  "9223372036854775809",
			Base:   0,
			Output: 1<<63 - 1,
			Error:  Error_Range,
		},
		signed_base_case{
			Input:  "-9223372036854775809",
			Base:   0,
			Output: -1 << 63,
			Error:  Error_Range,
		},
		signed_base_case{Input: "g", Base: 17, Output: 16},
	)
}

// Writes part 2 of the standard library parseInt64BaseTests table into the cases.
func signed_base_cases_2(cases []signed_base_case) (extended []signed_base_case) {
	return append(cases,
		signed_base_case{Input: "10", Base: 25, Output: 25},
		signed_base_case{
			Input:  "holycow",
			Base:   35,
			Output: (((((17*35+24)*35+21)*35+34)*35+12)*35+24)*35 + 32,
		},
		signed_base_case{
			Input:  "holycow",
			Base:   36,
			Output: (((((17*36+24)*36+21)*36+34)*36+12)*36+24)*36 + 32,
		},
		signed_base_case{Input: "0", Base: 2, Output: 0},
		signed_base_case{Input: "-1", Base: 2, Output: -1},
		signed_base_case{Input: "1010", Base: 2, Output: 10},
		signed_base_case{Input: "1000000000000000", Base: 2, Output: 1 << 15},
		signed_base_case{
			Input:  "111111111111111111111111111111111111111111111111111111111111111",
			Base:   2,
			Output: 1<<63 - 1,
		},
		signed_base_case{
			Input:  "1000000000000000000000000000000000000000000000000000000000000000",
			Base:   2,
			Output: 1<<63 - 1,
			Error:  Error_Range,
		},
		signed_base_case{
			Input:  "-1000000000000000000000000000000000000000000000000000000000000000",
			Base:   2,
			Output: -1 << 63,
		},
		signed_base_case{
			Input:  "-1000000000000000000000000000000000000000000000000000000000000001",
			Base:   2,
			Output: -1 << 63,
			Error:  Error_Range,
		},
		signed_base_case{Input: "-10", Base: 8, Output: -8},
		signed_base_case{Input: "57635436545", Base: 8, Output: 057635436545},
		signed_base_case{Input: "100000000", Base: 8, Output: 1 << 24},
		signed_base_case{Input: "10", Base: 16, Output: 16},
		signed_base_case{
			Input:  "-123456789abcdef",
			Base:   16,
			Output: -0x123456789abcdef,
		},
		signed_base_case{Input: "7fffffffffffffff", Base: 16, Output: 1<<63 - 1},
		signed_base_case{Input: "-0x_1_2_3_4_5", Base: 0, Output: -0x12345},
		signed_base_case{Input: "0x_1_2_3_4_5", Base: 0, Output: 0x12345},
		signed_base_case{Input: "-_0x12345", Base: 0, Output: 0, Error: Error_Syntax},
		signed_base_case{Input: "_-0x12345", Base: 0, Output: 0, Error: Error_Syntax},
		signed_base_case{Input: "_0x12345", Base: 0, Output: 0, Error: Error_Syntax},
	)
}

// Writes part 3 of the standard library parseInt64BaseTests table into the cases.
func signed_base_cases_3(cases []signed_base_case) (extended []signed_base_case) {
	return append(cases,
		signed_base_case{Input: "0x__12345", Base: 0, Output: 0, Error: Error_Syntax},
		signed_base_case{Input: "0x1__2345", Base: 0, Output: 0, Error: Error_Syntax},
		signed_base_case{Input: "0x1234__5", Base: 0, Output: 0, Error: Error_Syntax},
		signed_base_case{Input: "0x12345_", Base: 0, Output: 0, Error: Error_Syntax},
		signed_base_case{Input: "-0_1_2_3_4_5", Base: 0, Output: -012345},
		signed_base_case{Input: "0_1_2_3_4_5", Base: 0, Output: 012345},
		signed_base_case{Input: "-_012345", Base: 0, Output: 0, Error: Error_Syntax},
		signed_base_case{Input: "_-012345", Base: 0, Output: 0, Error: Error_Syntax},
		signed_base_case{Input: "_012345", Base: 0, Output: 0, Error: Error_Syntax},
		signed_base_case{Input: "0__12345", Base: 0, Output: 0, Error: Error_Syntax},
		signed_base_case{Input: "01234__5", Base: 0, Output: 0, Error: Error_Syntax},
		signed_base_case{Input: "012345_", Base: 0, Output: 0, Error: Error_Syntax},
		signed_base_case{Input: "+0xf", Base: 0, Output: 0xf},
		signed_base_case{Input: "-0xf", Base: 0, Output: -0xf},
		signed_base_case{Input: "0x+f", Base: 0, Output: 0, Error: Error_Syntax},
		signed_base_case{Input: "0x-f", Base: 0, Output: 0, Error: Error_Syntax},
	)
}

// Holds the whole standard library parseInt64BaseTests table. One allocation, sized to the
// case count, holds it.
func signed_base_cases() (cases []signed_base_case) {
	cases = make([]signed_base_case, 0, 60)
	cases = signed_base_cases_1(cases)
	cases = signed_base_cases_2(cases)
	cases = signed_base_cases_3(cases)
	return cases
}

// Writes part 1 of the standard library parseUint32Tests table into the cases.
func unsigned_narrow_cases_1(cases []unsigned_narrow_case) (extended []unsigned_narrow_case) {
	return append(cases,
		unsigned_narrow_case{Input: "", Output: 0, Error: Error_Syntax},
		unsigned_narrow_case{Input: "0", Output: 0},
		unsigned_narrow_case{Input: "1", Output: 1},
		unsigned_narrow_case{Input: "12345", Output: 12345},
		unsigned_narrow_case{Input: "012345", Output: 12345},
		unsigned_narrow_case{Input: "12345x", Output: 0, Error: Error_Syntax},
		unsigned_narrow_case{Input: "987654321", Output: 987654321},
		unsigned_narrow_case{Input: "4294967295", Output: 1<<32 - 1},
		unsigned_narrow_case{Input: "4294967296", Output: 1<<32 - 1, Error: Error_Range},
		unsigned_narrow_case{Input: "1_2_3_4_5", Output: 0, Error: Error_Syntax},
		unsigned_narrow_case{Input: "_12345", Output: 0, Error: Error_Syntax},
		unsigned_narrow_case{Input: "_12345", Output: 0, Error: Error_Syntax},
		unsigned_narrow_case{Input: "1__2345", Output: 0, Error: Error_Syntax},
		unsigned_narrow_case{Input: "12345_", Output: 0, Error: Error_Syntax},
	)
}

// Holds the whole standard library parseUint32Tests table. One allocation, sized to the
// case count, holds it.
func unsigned_narrow_cases() (cases []unsigned_narrow_case) {
	cases = make([]unsigned_narrow_case, 0, 14)
	cases = unsigned_narrow_cases_1(cases)
	return cases
}

// Writes part 1 of the standard library parseInt32Tests table into the cases.
func signed_narrow_cases_1(cases []signed_narrow_case) (extended []signed_narrow_case) {
	return append(cases,
		signed_narrow_case{Input: "", Output: 0, Error: Error_Syntax},
		signed_narrow_case{Input: "0", Output: 0},
		signed_narrow_case{Input: "-0", Output: 0},
		signed_narrow_case{Input: "1", Output: 1},
		signed_narrow_case{Input: "-1", Output: -1},
		signed_narrow_case{Input: "12345", Output: 12345},
		signed_narrow_case{Input: "-12345", Output: -12345},
		signed_narrow_case{Input: "012345", Output: 12345},
		signed_narrow_case{Input: "-012345", Output: -12345},
		signed_narrow_case{Input: "12345x", Output: 0, Error: Error_Syntax},
		signed_narrow_case{Input: "-12345x", Output: 0, Error: Error_Syntax},
		signed_narrow_case{Input: "987654321", Output: 987654321},
		signed_narrow_case{Input: "-987654321", Output: -987654321},
		signed_narrow_case{Input: "2147483647", Output: 1<<31 - 1},
		signed_narrow_case{Input: "-2147483647", Output: -(1<<31 - 1)},
		signed_narrow_case{Input: "2147483648", Output: 1<<31 - 1, Error: Error_Range},
		signed_narrow_case{Input: "-2147483648", Output: -1 << 31},
		signed_narrow_case{Input: "2147483649", Output: 1<<31 - 1, Error: Error_Range},
		signed_narrow_case{Input: "-2147483649", Output: -1 << 31, Error: Error_Range},
		signed_narrow_case{Input: "-1_2_3_4_5", Output: 0, Error: Error_Syntax},
		signed_narrow_case{Input: "-_12345", Output: 0, Error: Error_Syntax},
		signed_narrow_case{Input: "_12345", Output: 0, Error: Error_Syntax},
		signed_narrow_case{Input: "1__2345", Output: 0, Error: Error_Syntax},
		signed_narrow_case{Input: "12345_", Output: 0, Error: Error_Syntax},
		signed_narrow_case{Input: "123%45", Output: 0, Error: Error_Syntax},
	)
}

// Holds the whole standard library parseInt32Tests table. One allocation, sized to the
// case count, holds it.
func signed_narrow_cases() (cases []signed_narrow_case) {
	cases = make([]signed_narrow_case, 0, 25)
	cases = signed_narrow_cases_1(cases)
	return cases
}

// Writes part 1 of the standard library itob64tests table into the cases.
func signed_text_cases_1(cases []signed_text_case) (extended []signed_text_case) {
	return append(cases,
		signed_text_case{Input: 0, Base: 10, Output: "0"},
		signed_text_case{Input: 1, Base: 10, Output: "1"},
		signed_text_case{Input: -1, Base: 10, Output: "-1"},
		signed_text_case{Input: 12345678, Base: 10, Output: "12345678"},
		signed_text_case{Input: -987654321, Base: 10, Output: "-987654321"},
		signed_text_case{Input: 1<<31 - 1, Base: 10, Output: "2147483647"},
		signed_text_case{Input: -1<<31 + 1, Base: 10, Output: "-2147483647"},
		signed_text_case{Input: 1 << 31, Base: 10, Output: "2147483648"},
		signed_text_case{Input: -1 << 31, Base: 10, Output: "-2147483648"},
		signed_text_case{Input: 1<<31 + 1, Base: 10, Output: "2147483649"},
		signed_text_case{Input: -1<<31 - 1, Base: 10, Output: "-2147483649"},
		signed_text_case{Input: 1<<32 - 1, Base: 10, Output: "4294967295"},
		signed_text_case{Input: -1<<32 + 1, Base: 10, Output: "-4294967295"},
		signed_text_case{Input: 1 << 32, Base: 10, Output: "4294967296"},
		signed_text_case{Input: -1 << 32, Base: 10, Output: "-4294967296"},
		signed_text_case{Input: 1<<32 + 1, Base: 10, Output: "4294967297"},
		signed_text_case{Input: -1<<32 - 1, Base: 10, Output: "-4294967297"},
		signed_text_case{Input: 1 << 50, Base: 10, Output: "1125899906842624"},
		signed_text_case{Input: 1<<63 - 1, Base: 10, Output: "9223372036854775807"},
		signed_text_case{Input: -1<<63 + 1, Base: 10, Output: "-9223372036854775807"},
		signed_text_case{Input: -1 << 63, Base: 10, Output: "-9223372036854775808"},
		signed_text_case{Input: 0, Base: 2, Output: "0"},
	)
}

// Writes part 2 of the standard library itob64tests table into the cases.
func signed_text_cases_2(cases []signed_text_case) (extended []signed_text_case) {
	return append(cases,
		signed_text_case{Input: 10, Base: 2, Output: "1010"},
		signed_text_case{Input: -1, Base: 2, Output: "-1"},
		signed_text_case{Input: 1 << 15, Base: 2, Output: "1000000000000000"},
		signed_text_case{Input: -8, Base: 8, Output: "-10"},
		signed_text_case{Input: 057635436545, Base: 8, Output: "57635436545"},
		signed_text_case{Input: 1 << 24, Base: 8, Output: "100000000"},
		signed_text_case{Input: 16, Base: 16, Output: "10"},
		signed_text_case{
			Input:  -0x123456789abcdef,
			Base:   16,
			Output: "-123456789abcdef",
		},
		signed_text_case{Input: 1<<63 - 1, Base: 16, Output: "7fffffffffffffff"},
		signed_text_case{
			Input:  1<<63 - 1,
			Base:   2,
			Output: "111111111111111111111111111111111111111111111111111111111111111",
		},
		signed_text_case{
			Input:  -1 << 63,
			Base:   2,
			Output: "-1000000000000000000000000000000000000000000000000000000000000000",
		},
		signed_text_case{Input: 16, Base: 17, Output: "g"},
		signed_text_case{Input: 25, Base: 25, Output: "10"},
		signed_text_case{
			Input:  (((((17*35+24)*35+21)*35+34)*35+12)*35+24)*35 + 32,
			Base:   35,
			Output: "holycow",
		},
		signed_text_case{
			Input:  (((((17*36+24)*36+21)*36+34)*36+12)*36+24)*36 + 32,
			Base:   36,
			Output: "holycow",
		},
	)
}

// Holds the whole standard library itob64tests table. One allocation, sized to the
// case count, holds it.
func signed_text_cases() (cases []signed_text_case) {
	cases = make([]signed_text_case, 0, 37)
	cases = signed_text_cases_1(cases)
	cases = signed_text_cases_2(cases)
	return cases
}

// Writes part 1 of the standard library uitob64tests table into the cases.
func unsigned_text_cases_1(cases []unsigned_text_case) (extended []unsigned_text_case) {
	return append(cases,
		unsigned_text_case{Input: 1<<63 - 1, Base: 10, Output: "9223372036854775807"},
		unsigned_text_case{Input: 1 << 63, Base: 10, Output: "9223372036854775808"},
		unsigned_text_case{Input: 1<<63 + 1, Base: 10, Output: "9223372036854775809"},
		unsigned_text_case{Input: 1<<64 - 2, Base: 10, Output: "18446744073709551614"},
		unsigned_text_case{Input: 1<<64 - 1, Base: 10, Output: "18446744073709551615"},
		unsigned_text_case{
			Input:  1<<64 - 1,
			Base:   2,
			Output: "1111111111111111111111111111111111111111111111111111111111111111",
		},
	)
}

// Holds the whole standard library uitob64tests table. One allocation, sized to the
// case count, holds it.
func unsigned_text_cases() (cases []unsigned_text_case) {
	cases = make([]unsigned_text_case, 0, 6)
	cases = unsigned_text_cases_1(cases)
	return cases
}

// Writes part 1 of the standard library varlenUints table into the cases.
func varlen_cases_1(cases []varlen_case) (extended []varlen_case) {
	return append(cases,
		varlen_case{Input: 1, Output: "1"},
		varlen_case{Input: 12, Output: "12"},
		varlen_case{Input: 123, Output: "123"},
		varlen_case{Input: 1234, Output: "1234"},
		varlen_case{Input: 12345, Output: "12345"},
		varlen_case{Input: 123456, Output: "123456"},
		varlen_case{Input: 1234567, Output: "1234567"},
		varlen_case{Input: 12345678, Output: "12345678"},
		varlen_case{Input: 123456789, Output: "123456789"},
		varlen_case{Input: 1234567890, Output: "1234567890"},
		varlen_case{Input: 12345678901, Output: "12345678901"},
		varlen_case{Input: 123456789012, Output: "123456789012"},
		varlen_case{Input: 1234567890123, Output: "1234567890123"},
		varlen_case{Input: 12345678901234, Output: "12345678901234"},
		varlen_case{Input: 123456789012345, Output: "123456789012345"},
		varlen_case{Input: 1234567890123456, Output: "1234567890123456"},
		varlen_case{Input: 12345678901234567, Output: "12345678901234567"},
		varlen_case{Input: 123456789012345678, Output: "123456789012345678"},
		varlen_case{Input: 1234567890123456789, Output: "1234567890123456789"},
		varlen_case{Input: 12345678901234567890, Output: "12345678901234567890"},
	)
}

// Holds the whole standard library varlenUints table. One allocation, sized to the
// case count, holds it.
func varlen_cases() (cases []varlen_case) {
	cases = make([]varlen_case, 0, 20)
	cases = varlen_cases_1(cases)
	return cases
}

// Writes part 1 of the standard library atobtests table into the cases.
func boolean_cases_1(cases []boolean_case) (extended []boolean_case) {
	return append(cases,
		boolean_case{Input: "", Output: false, Error: Error_Syntax},
		boolean_case{Input: "asdf", Output: false, Error: Error_Syntax},
		boolean_case{Input: "0", Output: false},
		boolean_case{Input: "f", Output: false},
		boolean_case{Input: "F", Output: false},
		boolean_case{Input: "FALSE", Output: false},
		boolean_case{Input: "false", Output: false},
		boolean_case{Input: "False", Output: false},
		boolean_case{Input: "1", Output: true},
		boolean_case{Input: "t", Output: true},
		boolean_case{Input: "T", Output: true},
		boolean_case{Input: "TRUE", Output: true},
		boolean_case{Input: "true", Output: true},
		boolean_case{Input: "True", Output: true},
	)
}

// Holds the whole standard library atobtests table. One allocation, sized to the
// case count, holds it.
func boolean_cases() (cases []boolean_case) {
	cases = make([]boolean_case, 0, 14)
	cases = boolean_cases_1(cases)
	return cases
}

// Writes part 1 of the standard library quotetests table into the cases.
func quote_cases_1(cases []quote_case) (extended []quote_case) {
	return append(cases,
		quote_case{
			Input:          "\a\b\f\r\n\t\v",
			Output:         `"\a\b\f\r\n\t\v"`,
			ASCII_Output:   `"\a\b\f\r\n\t\v"`,
			Graphic_Output: `"\a\b\f\r\n\t\v"`,
		},
		quote_case{
			Input:          "\\",
			Output:         `"\\"`,
			ASCII_Output:   `"\\"`,
			Graphic_Output: `"\\"`,
		},
		quote_case{
			Input:          "abc\xffdef",
			Output:         `"abc\xffdef"`,
			ASCII_Output:   `"abc\xffdef"`,
			Graphic_Output: `"abc\xffdef"`,
		},
		quote_case{
			Input:          "\u263a",
			Output:         `"☺"`,
			ASCII_Output:   `"\u263a"`,
			Graphic_Output: `"☺"`,
		},
		quote_case{
			Input:          "\U0010ffff",
			Output:         `"\U0010ffff"`,
			ASCII_Output:   `"\U0010ffff"`,
			Graphic_Output: `"\U0010ffff"`,
		},
		quote_case{
			Input:          "\x04",
			Output:         `"\x04"`,
			ASCII_Output:   `"\x04"`,
			Graphic_Output: `"\x04"`,
		},
		quote_case{
			Input:          "!\u00a0!\u2000!\u3000!",
			Output:         `"!\u00a0!\u2000!\u3000!"`,
			ASCII_Output:   `"!\u00a0!\u2000!\u3000!"`,
			Graphic_Output: "\"!\u00a0!\u2000!\u3000!\"",
		},
		quote_case{
			Input:          "\x7f",
			Output:         `"\x7f"`,
			ASCII_Output:   `"\x7f"`,
			Graphic_Output: `"\x7f"`,
		},
	)
}

// Holds the whole standard library quotetests table. One allocation, sized to the
// case count, holds it.
func quote_cases() (cases []quote_case) {
	cases = make([]quote_case, 0, 8)
	cases = quote_cases_1(cases)
	return cases
}

// Writes part 1 of the standard library quoterunetests table into the cases.
func quote_rune_cases_1(cases []quote_rune_case) (extended []quote_rune_case) {
	return append(cases,
		quote_rune_case{
			Input:          'a',
			Output:         `'a'`,
			ASCII_Output:   `'a'`,
			Graphic_Output: `'a'`,
		},
		quote_rune_case{
			Input:          '\a',
			Output:         `'\a'`,
			ASCII_Output:   `'\a'`,
			Graphic_Output: `'\a'`,
		},
		quote_rune_case{
			Input:          '\\',
			Output:         `'\\'`,
			ASCII_Output:   `'\\'`,
			Graphic_Output: `'\\'`,
		},
		quote_rune_case{
			Input:          0xFF,
			Output:         `'ÿ'`,
			ASCII_Output:   `'\u00ff'`,
			Graphic_Output: `'ÿ'`,
		},
		quote_rune_case{
			Input:          0x263a,
			Output:         `'☺'`,
			ASCII_Output:   `'\u263a'`,
			Graphic_Output: `'☺'`,
		},
		quote_rune_case{
			Input:          0xdead,
			Output:         `'�'`,
			ASCII_Output:   `'\ufffd'`,
			Graphic_Output: `'�'`,
		},
		quote_rune_case{
			Input:          0xfffd,
			Output:         `'�'`,
			ASCII_Output:   `'\ufffd'`,
			Graphic_Output: `'�'`,
		},
		quote_rune_case{
			Input:          0x0010ffff,
			Output:         `'\U0010ffff'`,
			ASCII_Output:   `'\U0010ffff'`,
			Graphic_Output: `'\U0010ffff'`,
		},
		quote_rune_case{
			Input:          0x0010ffff + 1,
			Output:         `'�'`,
			ASCII_Output:   `'\ufffd'`,
			Graphic_Output: `'�'`,
		},
		quote_rune_case{
			Input:          0x04,
			Output:         `'\x04'`,
			ASCII_Output:   `'\x04'`,
			Graphic_Output: `'\x04'`,
		},
	)
}

// Writes part 2 of the standard library quoterunetests table into the cases.
func quote_rune_cases_2(cases []quote_rune_case) (extended []quote_rune_case) {
	return append(cases,
		quote_rune_case{
			Input:          '\u00a0',
			Output:         `'\u00a0'`,
			ASCII_Output:   `'\u00a0'`,
			Graphic_Output: "'\u00a0'",
		},
		quote_rune_case{
			Input:          '\u2000',
			Output:         `'\u2000'`,
			ASCII_Output:   `'\u2000'`,
			Graphic_Output: "'\u2000'",
		},
		quote_rune_case{
			Input:          '\u3000',
			Output:         `'\u3000'`,
			ASCII_Output:   `'\u3000'`,
			Graphic_Output: "'\u3000'",
		},
	)
}

// Holds the whole standard library quoterunetests table. One allocation, sized to the
// case count, holds it.
func quote_rune_cases() (cases []quote_rune_case) {
	cases = make([]quote_rune_case, 0, 13)
	cases = quote_rune_cases_1(cases)
	cases = quote_rune_cases_2(cases)
	return cases
}

// Writes part 1 of the standard library canbackquotetests table into the cases.
func backquote_cases_1(cases []backquote_case) (extended []backquote_case) {
	return append(cases,
		backquote_case{Input: "`", Output: false},
		backquote_case{Input: string(rune(0)), Output: false},
		backquote_case{Input: string(rune(1)), Output: false},
		backquote_case{Input: string(rune(2)), Output: false},
		backquote_case{Input: string(rune(3)), Output: false},
		backquote_case{Input: string(rune(4)), Output: false},
		backquote_case{Input: string(rune(5)), Output: false},
		backquote_case{Input: string(rune(6)), Output: false},
		backquote_case{Input: string(rune(7)), Output: false},
		backquote_case{Input: string(rune(8)), Output: false},
		backquote_case{Input: string(rune(9)), Output: true},
		backquote_case{Input: string(rune(10)), Output: false},
		backquote_case{Input: string(rune(11)), Output: false},
		backquote_case{Input: string(rune(12)), Output: false},
		backquote_case{Input: string(rune(13)), Output: false},
		backquote_case{Input: string(rune(14)), Output: false},
		backquote_case{Input: string(rune(15)), Output: false},
		backquote_case{Input: string(rune(16)), Output: false},
		backquote_case{Input: string(rune(17)), Output: false},
		backquote_case{Input: string(rune(18)), Output: false},
		backquote_case{Input: string(rune(19)), Output: false},
		backquote_case{Input: string(rune(20)), Output: false},
	)
}

// Writes part 2 of the standard library canbackquotetests table into the cases.
func backquote_cases_2(cases []backquote_case) (extended []backquote_case) {
	return append(cases,
		backquote_case{Input: string(rune(21)), Output: false},
		backquote_case{Input: string(rune(22)), Output: false},
		backquote_case{Input: string(rune(23)), Output: false},
		backquote_case{Input: string(rune(24)), Output: false},
		backquote_case{Input: string(rune(25)), Output: false},
		backquote_case{Input: string(rune(26)), Output: false},
		backquote_case{Input: string(rune(27)), Output: false},
		backquote_case{Input: string(rune(28)), Output: false},
		backquote_case{Input: string(rune(29)), Output: false},
		backquote_case{Input: string(rune(30)), Output: false},
		backquote_case{Input: string(rune(31)), Output: false},
		backquote_case{Input: string(rune(0x7F)), Output: false},
		backquote_case{Input: `' !"#$%&'()*+,-./:;<=>?@[\]^_{|}~`, Output: true},
		backquote_case{Input: `0123456789`, Output: true},
		backquote_case{Input: `ABCDEFGHIJKLMNOPQRSTUVWXYZ`, Output: true},
		backquote_case{Input: `abcdefghijklmnopqrstuvwxyz`, Output: true},
		backquote_case{Input: `☺`, Output: true},
		backquote_case{Input: "\x80", Output: false},
		backquote_case{Input: "a\xe0\xa0z", Output: false},
		backquote_case{Input: "\ufeffabc", Output: false},
		backquote_case{Input: "a\ufeffz", Output: false},
	)
}

// Holds the whole standard library canbackquotetests table. One allocation, sized to the
// case count, holds it.
func backquote_cases() (cases []backquote_case) {
	cases = make([]backquote_case, 0, 43)
	cases = backquote_cases_1(cases)
	cases = backquote_cases_2(cases)
	return cases
}

// Writes part 1 of the standard library unquotetests table into the cases.
func unquote_cases_1(cases []unquote_case) (extended []unquote_case) {
	return append(cases,
		unquote_case{Input: `""`, Output: ""},
		unquote_case{Input: `"a"`, Output: "a"},
		unquote_case{Input: `"abc"`, Output: "abc"},
		unquote_case{Input: `"☺"`, Output: "☺"},
		unquote_case{Input: `"hello world"`, Output: "hello world"},
		unquote_case{Input: `"\xFF"`, Output: "\xFF"},
		unquote_case{Input: `"\377"`, Output: "\377"},
		unquote_case{Input: `"\u1234"`, Output: "\u1234"},
		unquote_case{Input: `"\U00010111"`, Output: "\U00010111"},
		unquote_case{Input: `"\U0001011111"`, Output: "\U0001011111"},
		unquote_case{Input: `"\a\b\f\n\r\t\v\\\""`, Output: "\a\b\f\n\r\t\v\\\""},
		unquote_case{Input: `"'"`, Output: "'"},
		unquote_case{Input: `'a'`, Output: "a"},
		unquote_case{Input: `'☹'`, Output: "☹"},
		unquote_case{Input: `'\a'`, Output: "\a"},
		unquote_case{Input: `'\x10'`, Output: "\x10"},
		unquote_case{Input: `'\377'`, Output: "\377"},
		unquote_case{Input: `'\u1234'`, Output: "\u1234"},
		unquote_case{Input: `'\U00010111'`, Output: "\U00010111"},
		unquote_case{Input: `'\t'`, Output: "\t"},
		unquote_case{Input: `' '`, Output: " "},
		unquote_case{Input: `'\''`, Output: "'"},
		unquote_case{Input: `'"'`, Output: "\""},
		unquote_case{Input: "``", Output: ``},
		unquote_case{Input: "`a`", Output: `a`},
		unquote_case{Input: "`abc`", Output: `abc`},
		unquote_case{Input: "`☺`", Output: `☺`},
		unquote_case{Input: "`hello world`", Output: `hello world`},
		unquote_case{Input: "`\\xFF`", Output: `\xFF`},
		unquote_case{Input: "`\\377`", Output: `\377`},
	)
}

// Writes part 2 of the standard library unquotetests table into the cases.
func unquote_cases_2(cases []unquote_case) (extended []unquote_case) {
	return append(cases,
		unquote_case{Input: "`\\`", Output: `\`},
		unquote_case{Input: "`\n`", Output: "\n"},
		unquote_case{Input: "`	`", Output: `	`},
		unquote_case{Input: "` `", Output: ` `},
		unquote_case{Input: "`a\rb`", Output: "ab"},
	)
}

// Holds the whole standard library unquotetests table. One allocation, sized to the
// case count, holds it.
func unquote_cases() (cases []unquote_case) {
	cases = make([]unquote_case, 0, 35)
	cases = unquote_cases_1(cases)
	cases = unquote_cases_2(cases)
	return cases
}

// Writes part 1 of the standard library misquoted table into the cases.
func misquoted_cases_1(cases []string) (extended []string) {
	return append(cases,
		``,
		`"`,
		`"a`,
		`"'`,
		`b"`,
		`"\"`,
		`"\9"`,
		`"\19"`,
		`"\129"`,
		`'\'`,
		`'\9'`,
		`'\19'`,
		`'\129'`,
		`'ab'`,
		`"\x1!"`,
		`"\U12345678"`,
		`"\z"`,
		"`",
		"`xxx",
		"``x\r",
		"`\"",
		`"\'"`,
	)
}

// Writes part 2 of the standard library misquoted table into the cases.
func misquoted_cases_2(cases []string) (extended []string) {
	return append(cases,
		`'\"'`,
		"\"\n\"",
		"\"\\n\n\"",
		"'\n'",
		`"\udead"`,
		`"\ud83d\ude4f"`,
	)
}

// Holds the whole standard library misquoted table. One allocation, sized to the
// case count, holds it.
func misquoted_cases() (cases []string) {
	cases = make([]string, 0, 28)
	cases = misquoted_cases_1(cases)
	cases = misquoted_cases_2(cases)
	return cases
}

// The benchmarks below measure each conversion beside the standard library conversion
// that it ports. Run them with the tag that makes an assertion inert, because an
// ordinary build records coverage and measures the recorder as well:
//
//	go test -tags invariant_noop ./shared/strconv/ -run '^$' -bench . -count 8
//
// The empty name pattern matches no test, thus the run holds the benchmarks alone.
//
// Each pair uses b.Loop so discarded results stay alive and optimization cannot remove
// either conversion.
//
// A house benchmark and its standard library pair take the same input, thus the two
// results divide into one ratio.

// Benchmark_Quote_ASCII_House writes a literal of plain text.
func Benchmark_Quote_ASCII_House(b *testing.B) {
	text := Text(benchmark_text("a", TEXT_SIZE_MAXIMUM))
	var storage [QUOTED_TEXT_SIZE_MAXIMUM]byte
	b.ResetTimer()
	for b.Loop() {
		Quote_Into(storage[:], text)
	}
}

// Benchmark_Quote_CJK_House writes a literal of three-byte characters.
func Benchmark_Quote_CJK_House(b *testing.B) {
	text := Text(benchmark_text("一", TEXT_SIZE_MAXIMUM/3))
	var storage [QUOTED_TEXT_SIZE_MAXIMUM]byte
	b.ResetTimer()
	for b.Loop() {
		Quote_Into(storage[:], text)
	}
}

// Benchmark_Quote_Control_House writes a literal whose every byte needs an escape.
func Benchmark_Quote_Control_House(b *testing.B) {
	text := Text(benchmark_text("\x01", TEXT_SIZE_MAXIMUM))
	var storage [QUOTED_TEXT_SIZE_MAXIMUM]byte
	b.ResetTimer()
	for b.Loop() {
		Quote_Into(storage[:], text)
	}
}

// Benchmark_Unquote_Plain_House reads a literal that holds no escape.
func Benchmark_Unquote_Plain_House(b *testing.B) {
	text := Text(`"` + benchmark_text("a", TEXT_SIZE_MAXIMUM-2) + `"`)
	var storage [UNQUOTED_TEXT_SIZE_MAXIMUM]byte
	b.ResetTimer()
	for b.Loop() {
		Unquote_Into(storage[:], text)
	}
}

// Benchmark_Unquote_Escaped_House reads a literal that holds one escape.
func Benchmark_Unquote_Escaped_House(b *testing.B) {
	text := Text(`"\n` + benchmark_text("a", TEXT_SIZE_MAXIMUM-4) + `"`)
	var storage [UNQUOTED_TEXT_SIZE_MAXIMUM]byte
	b.ResetTimer()
	for b.Loop() {
		Unquote_Into(storage[:], text)
	}
}

// Benchmark_Parse_Integer_House reads the widest signed number.
func Benchmark_Parse_Integer_House(b *testing.B) {
	for b.Loop() {
		Parse_Integer("9223372036854775807", 10, 64)
	}
}

// Benchmark_Parse_Decimal_House reads a short signed number.
func Benchmark_Parse_Decimal_House(b *testing.B) {
	for b.Loop() {
		Parse_Decimal("-12345")
	}
}

// Benchmark_Format_Integer_House writes the smallest signed number.
func Benchmark_Format_Integer_House(b *testing.B) {
	var storage [INTEGER_TEXT_SIZE_MAXIMUM]byte
	for b.Loop() {
		Format_Integer_Into(storage[:], -9223372036854775808, 10)
	}
}

// Benchmark_Quote_Rune_House writes one character literal.
func Benchmark_Quote_Rune_House(b *testing.B) {
	var storage [CHARACTER_TEXT_SIZE_MAXIMUM]byte
	for b.Loop() {
		Quote_Rune_Into(storage[:], 'a')
	}
}

// Benchmark_Can_Backquote_House reads a text that stays unchanged inside backquotes.
func Benchmark_Can_Backquote_House(b *testing.B) {
	text := Text(benchmark_text("a", TEXT_SIZE_MAXIMUM))
	b.ResetTimer()
	for b.Loop() {
		Can_Backquote(text)
	}
}

// Returns text that repeats one character.
func benchmark_text(character string, count int) (text string) {
	built := make([]byte, 0, len(character)*count)
	for index := 0; index < count; index++ {
		built = append(built, character...)
	}
	return string(built)
}
