package token_test

import (
	"testing"

	"local/james-orcales/shared/go/token"
	"local/james-orcales/shared/testify"
)

// Test_Scan binds the Scan specification leaf before fixture declarations.
func Test_Scan(t *testing.T) {
	test_scan(t)
}

// Test_Whitespace binds the Whitespace specification leaf before fixture declarations.
func Test_Whitespace(t *testing.T) {
	test_whitespace(t)
}

// Test_Comments binds the Comments specification leaf before fixture declarations.
func Test_Comments(t *testing.T) {
	test_comments(t)
}

// Test_Identifiers binds the Identifiers specification leaf before fixture declarations.
func Test_Identifiers(t *testing.T) {
	test_identifiers(t)
}

// Test_Keywords binds the Keywords specification leaf before fixture declarations.
func Test_Keywords(t *testing.T) {
	test_keywords(t)
}

// Test_Number_Literals binds the Number Literals specification leaf before fixture declarations.
func Test_Number_Literals(t *testing.T) {
	test_number_literals(t)
}

// Test_Character_Literals binds the Character Literals leaf before fixture declarations.
func Test_Character_Literals(t *testing.T) {
	test_character_literals(t)
}

// Test_String_Literals binds the String Literals specification leaf before fixture declarations.
func Test_String_Literals(t *testing.T) {
	test_string_literals(t)
}

// Test_Operators binds the Operators specification leaf before fixture declarations.
func Test_Operators(t *testing.T) {
	test_operators(t)
}

// Test_Semicolon_Insertion binds the Semicolon Insertion leaf before fixture declarations.
func Test_Semicolon_Insertion(t *testing.T) {
	test_semicolon_insertion(t)
}

// Test_Illegal_Input binds the Illegal Input specification leaf before fixture declarations.
func Test_Illegal_Input(t *testing.T) {
	test_illegal_input(t)
}

// Test_Positions binds the Positions specification leaf before fixture declarations.
func Test_Positions(t *testing.T) {
	test_positions(t)
}

// Test_Bounds binds the Bounds specification leaf before fixture declarations.
func Test_Bounds(t *testing.T) {
	test_bounds(t)
}

// Test_Allocation binds the Allocation specification leaf before fixture declarations.
func Test_Allocation(t *testing.T) {
	test_allocation(t)
}

const TEST_TOKEN_COUNT_MAXIMUM = 256

const TEST_LARGE_SIZE = token.SOURCE_SIZE_MAXIMUM

const TEST_BLANK_SATURATION = 300

const TEST_KEYWORD_SOURCE = "break case chan const continue default defer else " +
	"fallthrough for func go goto if import interface map package range return " +
	"select struct switch type var"

const TEST_OPERATOR_SOURCE = "<<= >>= &^= ... += -= *= /= %= &= |= ^= << >> &^ && " +
	"|| <- ++ -- == != <= >= := + - * / % & | ^ < > = ! ( ) [ ] { } , . ; : ~"

type allocation_fixture struct {
	Scanner token.Scanner
	Token   token.Token
	Text    token.Source
	Index   token.Line_Index
	Indexed token.Boolean
	Line    token.Line
	Column  token.Column
}

func scan_kinds(text string) (kinds []token.Kind) {
	scanner := token.Scanner{Source: token.Source(text)}
	for range TEST_TOKEN_COUNT_MAXIMUM {
		one := token.Scan(&scanner)
		if one.Kind == token.KIND_END_OF_FILE {
			return kinds
		}
		kinds = append(kinds, one.Kind)
	}
	return kinds
}

// Sweeps every token of every listed source through Text, so the accessor sees each token class,
// each position, and each width the scanner can produce.
func text_sweep(t *testing.T) {
	sources := []string{"", "a", "+=", "a=b", "~", "#", "'a'"}
	for _, text := range sources {
		source := token.Source(text)
		scanner := token.Scanner{Source: source}
		for range TEST_TOKEN_COUNT_MAXIMUM {
			one := token.Scan(&scanner)
			testify.Equal(t, int(one.Size), len(token.Text(source, one)),
				"the text of a token in %q spans its size", text)
			if one.Kind == token.KIND_END_OF_FILE {
				break
			}
		}
	}
}

func filled_source(head string, filler byte, foot string) (source token.Source) {
	buffer := make([]byte, TEST_LARGE_SIZE)
	for index := range buffer {
		buffer[index] = filler
	}
	copy(buffer, head)
	copy(buffer[TEST_LARGE_SIZE-len(foot):], foot)
	return token.Source(buffer)
}

func test_scan(t *testing.T) {
	source := token.Source("package p")
	scanner := token.Scanner{Source: source}
	first := token.Scan(&scanner)
	testify.Equal(t, token.KIND_PACKAGE, first.Kind, "package keyword kind")
	testify.Equal(t, 0, int(first.Offset), "package keyword offset")
	testify.Equal(t, "package", string(token.Text(source, first)), "package keyword text")
	second := token.Scan(&scanner)
	testify.Equal(t, token.KIND_IDENTIFIER, second.Kind, "package name kind")
	testify.Equal(t, 8, int(second.Offset), "package name offset")
	testify.Equal(t, 1, int(second.Size), "package name size")
	third := token.Scan(&scanner)
	testify.Equal(t, token.KIND_SEMICOLON, third.Kind, "inserted semicolon kind")
	testify.Equal(t, 0, int(third.Size), "inserted semicolon size")
	testify.Equal(t, "", string(token.Text(source, third)), "inserted semicolon text")
	fourth := token.Scan(&scanner)
	testify.Equal(t, token.KIND_END_OF_FILE, fourth.Kind, "end of file kind")
	fifth := token.Scan(&scanner)
	testify.Equal(t, token.KIND_END_OF_FILE, fifth.Kind, "repeated end of file kind")
	testify.Equal(t, []token.Kind{
		token.KIND_IDENTIFIER, token.KIND_ASSIGN, token.KIND_IDENTIFIER,
		token.KIND_SEMICOLON,
	}, scan_kinds("a=b"), "adjacent single-byte tokens")
	text_sweep(t)
}

// Builds a source that opens with the run of empty lines this test names.
func blank_opening(lines int) (text string) {
	buffer := make([]byte, lines+1)
	for index := range lines {
		buffer[index] = '\n'
	}
	buffer[lines] = 'a'
	return string(buffer)
}

// States the empty line count each token carries, which a caller reads to write the spacing of
// one source back out.
func blank_counts(t *testing.T) {
	for _, one := range []struct {
		Lines int
		Count int
	}{
		{Lines: 0, Count: 0}, {Lines: 1, Count: 1}, {Lines: 2, Count: 2},
		{Lines: 3, Count: 3},
		{Lines: TEST_BLANK_SATURATION, Count: token.BLANK_COUNT_MAXIMUM},
	} {
		source := token.Source(blank_opening(one.Lines))
		scanner := token.Scanner{Source: source}
		first := token.Scan(&scanner)
		testify.Equal(t, one.Count, int(first.Blanks),
			"a token behind %d empty lines counts them", one.Lines)
		testify.Equal(t, "a", string(token.Text(source, first)),
			"the counted token still spans its own bytes")
		// The scan that follows carries the crossed count into its own guard, so the
		// saturating value is seen where the cursor reads it and not only where it is set.
		testify.Equal(t, token.KIND_SEMICOLON, token.Scan(&scanner).Kind,
			"the counted token closes its line")
	}
	pair := token.Scanner{Source: token.Source("a\n\nb\n")}
	token.Scan(&pair)
	testify.Equal(t, 1, int(token.Scan(&pair).Blanks),
		"one empty line between two tokens counts one")
	testify.Equal(t, token.KIND_IDENTIFIER, token.Scan(&pair).Kind,
		"the token behind the empty line follows it")
	scanner := token.Scanner{Source: token.Source("a\n\n\nb\nc\n")}
	testify.Equal(t, 0, int(token.Scan(&scanner).Blanks), "the first token counts none")
	testify.Equal(t, 2, int(token.Scan(&scanner).Blanks),
		"the semicolon that closes a line counts the empty lines behind it")
}

func test_whitespace(t *testing.T) {
	blank_counts(t)
	testify.Empty(t, scan_kinds(""), "empty source")
	testify.Empty(t, scan_kinds(" \t\r\n"), "whitespace only source")
	testify.Equal(t, []token.Kind{
		token.KIND_IDENTIFIER, token.KIND_IDENTIFIER, token.KIND_SEMICOLON,
	}, scan_kinds("a\t b"), "separated identifiers")
}

func test_comments(t *testing.T) {
	testify.Equal(t, []token.Kind{token.KIND_COMMENT}, scan_kinds("//"),
		"shortest line comment")
	testify.Equal(t, []token.Kind{token.KIND_COMMENT}, scan_kinds("//x"), "line comment")
	testify.Equal(t, []token.Kind{token.KIND_COMMENT}, scan_kinds("/*x"),
		"unclosed general comment")
	testify.Equal(t, []token.Kind{token.KIND_SLASH}, scan_kinds("/"),
		"a lone slash opens no comment")
	testify.Equal(t, []token.Kind{
		token.KIND_IDENTIFIER, token.KIND_COMMENT, token.KIND_SEMICOLON,
	}, scan_kinds("a//x\n"), "line comment keeps the semicolon its line owes")
	testify.Equal(t, []token.Kind{
		token.KIND_IDENTIFIER, token.KIND_COMMENT, token.KIND_IDENTIFIER,
		token.KIND_SEMICOLON,
	}, scan_kinds("a/*x*/b"), "general comment on one line")
	testify.Equal(t, []token.Kind{
		token.KIND_IDENTIFIER, token.KIND_COMMENT, token.KIND_SEMICOLON,
		token.KIND_IDENTIFIER, token.KIND_SEMICOLON,
	}, scan_kinds("a/*\n*/b"), "general comment holding a line feed")
	source := token.Source("/*ab*/")
	scanner := token.Scanner{Source: source}
	one := token.Scan(&scanner)
	testify.Equal(t, "/*ab*/", string(token.Text(source, one)), "general comment text")
}

func test_identifiers(t *testing.T) {
	testify.Equal(t, []token.Kind{token.KIND_IDENTIFIER, token.KIND_SEMICOLON},
		scan_kinds("a"), "one-byte identifier")
	testify.Equal(t, []token.Kind{
		token.KIND_IDENTIFIER, token.KIND_IDENTIFIER, token.KIND_SEMICOLON,
	}, scan_kinds("_x9 Ada_Case"), "underscore and digit members")
	testify.Equal(t, []token.Kind{token.KIND_ILLEGAL}, scan_kinds("\xff"),
		"a byte above ASCII opens no identifier")
	testify.Equal(t, []token.Kind{token.KIND_IDENTIFIER, token.KIND_ILLEGAL},
		scan_kinds("a\xff"), "a byte above ASCII ends an identifier")
	source := token.Source("Ada_Case")
	scanner := token.Scanner{Source: source}
	one := token.Scan(&scanner)
	testify.Equal(t, "Ada_Case", string(token.Text(source, one)), "identifier text")
}

func test_keywords(t *testing.T) {
	testify.Equal(t, []token.Kind{
		token.KIND_BREAK, token.KIND_CASE, token.KIND_CHANNEL, token.KIND_CONSTANT,
		token.KIND_CONTINUE, token.KIND_DEFAULT, token.KIND_DEFER, token.KIND_ELSE,
		token.KIND_FALLTHROUGH, token.KIND_FOR, token.KIND_FUNCTION, token.KIND_GO,
		token.KIND_GOTO, token.KIND_IF, token.KIND_IMPORT, token.KIND_INTERFACE,
		token.KIND_MAP, token.KIND_PACKAGE, token.KIND_RANGE, token.KIND_RETURN,
		token.KIND_SELECT, token.KIND_STRUCTURE, token.KIND_SWITCH, token.KIND_TYPE,
		token.KIND_VARIABLE,
	}, scan_kinds(TEST_KEYWORD_SOURCE), "every Go keyword")
	testify.Equal(t, []token.Kind{token.KIND_IDENTIFIER, token.KIND_SEMICOLON},
		scan_kinds("breaks"), "a keyword prefix stays an identifier")
}

func test_number_literals(t *testing.T) {
	integers := "0 1 42 0b101 0o17 017 0xFF 1_000"
	testify.Equal(t, []token.Kind{
		token.KIND_INTEGER, token.KIND_INTEGER, token.KIND_INTEGER, token.KIND_INTEGER,
		token.KIND_INTEGER, token.KIND_INTEGER, token.KIND_INTEGER, token.KIND_INTEGER,
		token.KIND_SEMICOLON,
	}, scan_kinds(integers), "integer literals")
	floats := "1.5 .5 1e9 1E+9 0x1p-2 1.5e-3"
	testify.Equal(t, []token.Kind{
		token.KIND_FLOAT, token.KIND_FLOAT, token.KIND_FLOAT, token.KIND_FLOAT,
		token.KIND_FLOAT, token.KIND_FLOAT, token.KIND_SEMICOLON,
	}, scan_kinds(floats), "floating-point literals")
	testify.Equal(t, []token.Kind{
		token.KIND_IMAGINARY, token.KIND_IMAGINARY, token.KIND_IMAGINARY,
		token.KIND_SEMICOLON,
	}, scan_kinds("3i 1.5i 0b1i"), "imaginary literals")
	testify.Equal(t, []token.Kind{
		token.KIND_INTEGER, token.KIND_PLUS, token.KIND_INTEGER, token.KIND_SEMICOLON,
	}, scan_kinds("1+2"), "a sign outside an exponent ends a number")
	testify.Equal(t, []token.Kind{token.KIND_PERIOD}, scan_kinds("."),
		"a lone period opens no fraction")
	testify.Equal(t, []token.Kind{
		token.KIND_PERIOD, token.KIND_IDENTIFIER, token.KIND_SEMICOLON,
	}, scan_kinds(".a"), "a period before a letter opens no fraction")
	source := token.Source("0x1p-2")
	scanner := token.Scanner{Source: source}
	one := token.Scan(&scanner)
	testify.Equal(t, "0x1p-2", string(token.Text(source, one)), "hexadecimal float text")
}

func test_character_literals(t *testing.T) {
	testify.Equal(t, []token.Kind{
		token.KIND_CHARACTER, token.KIND_CHARACTER, token.KIND_CHARACTER,
		token.KIND_CHARACTER, token.KIND_SEMICOLON,
	}, scan_kinds(`'a' '\n' '\'' '\\'`), "character literals and escapes")
	testify.Equal(t, []token.Kind{token.KIND_ILLEGAL}, scan_kinds("'a"),
		"unclosed character literal at the end of the source")
	testify.Equal(t, []token.Kind{token.KIND_ILLEGAL}, scan_kinds("'a\n"),
		"unclosed character literal at the end of the line")
	testify.Equal(t, []token.Kind{token.KIND_ILLEGAL}, scan_kinds("'"),
		"a lone apostrophe closes no literal")
	source := token.Source(`'\''`)
	scanner := token.Scanner{Source: source}
	one := token.Scan(&scanner)
	testify.Equal(t, `'\''`, string(token.Text(source, one)), "escaped apostrophe text")
}

func test_string_literals(t *testing.T) {
	testify.Equal(t, []token.Kind{
		token.KIND_STRING, token.KIND_STRING, token.KIND_STRING, token.KIND_SEMICOLON,
	}, scan_kinds(`"a" "\"" "\n"`), "interpreted literals and escapes")
	testify.Equal(t, []token.Kind{token.KIND_STRING, token.KIND_SEMICOLON},
		scan_kinds("`a\nb`"), "a raw literal holds a line feed")
	testify.Equal(t, []token.Kind{token.KIND_ILLEGAL}, scan_kinds(`"a`),
		"unclosed interpreted literal")
	testify.Equal(t, []token.Kind{token.KIND_ILLEGAL}, scan_kinds("`a"),
		"unclosed raw literal")
	testify.Equal(t, []token.Kind{token.KIND_ILLEGAL}, scan_kinds(`"`),
		"a lone quotation mark closes no literal")
	testify.Equal(t, []token.Kind{token.KIND_ILLEGAL}, scan_kinds("`"),
		"a lone back quote closes no literal")
	source := token.Source("`a\nb`")
	scanner := token.Scanner{Source: source}
	one := token.Scan(&scanner)
	testify.Equal(t, "`a\nb`", string(token.Text(source, one)), "raw literal text")
}

func test_operators(t *testing.T) {
	testify.Equal(t, []token.Kind{
		token.KIND_SHIFT_LEFT_ASSIGN, token.KIND_SHIFT_RIGHT_ASSIGN,
		token.KIND_AND_NOT_ASSIGN, token.KIND_ELLIPSIS, token.KIND_PLUS_ASSIGN,
		token.KIND_MINUS_ASSIGN, token.KIND_STAR_ASSIGN, token.KIND_SLASH_ASSIGN,
		token.KIND_PERCENT_ASSIGN, token.KIND_AND_ASSIGN, token.KIND_OR_ASSIGN,
		token.KIND_EXCLUSIVE_OR_ASSIGN, token.KIND_SHIFT_LEFT, token.KIND_SHIFT_RIGHT,
		token.KIND_AND_NOT, token.KIND_LOGICAL_AND, token.KIND_LOGICAL_OR,
		token.KIND_ARROW, token.KIND_INCREMENT, token.KIND_DECREMENT, token.KIND_EQUAL,
		token.KIND_NOT_EQUAL, token.KIND_LESS_EQUAL, token.KIND_GREATER_EQUAL,
		token.KIND_DEFINE, token.KIND_PLUS, token.KIND_MINUS, token.KIND_STAR,
		token.KIND_SLASH, token.KIND_PERCENT, token.KIND_AND, token.KIND_OR,
		token.KIND_EXCLUSIVE_OR, token.KIND_LESS, token.KIND_GREATER, token.KIND_ASSIGN,
		token.KIND_NOT, token.KIND_PARENTHESIS_LEFT, token.KIND_PARENTHESIS_RIGHT,
		token.KIND_BRACKET_LEFT, token.KIND_BRACKET_RIGHT, token.KIND_BRACE_LEFT,
		token.KIND_BRACE_RIGHT, token.KIND_COMMA, token.KIND_PERIOD,
		token.KIND_SEMICOLON, token.KIND_COLON, token.KIND_TILDE,
	}, scan_kinds(TEST_OPERATOR_SOURCE), "every Go operator and delimiter")
	testify.Equal(t, []token.Kind{token.KIND_AND_NOT_ASSIGN},
		scan_kinds("&^="), "the longest match wins")
	testify.Equal(t, []token.Kind{token.KIND_PLUS_ASSIGN}, scan_kinds("+="),
		"a two-byte operator at the end of the source")
}

func test_semicolon_insertion(t *testing.T) {
	closers := []string{"a\n", "1\n", "1.5\n", "1i\n", "'a'\n", "\"a\"\n", "break\n",
		"continue\n", "fallthrough\n", "return\n", "++\n", "--\n", ")\n", "]\n", "}\n"}
	for _, one := range closers {
		kinds := scan_kinds(one)
		testify.Equal(t, token.KIND_SEMICOLON, kinds[len(kinds)-1],
			"a line feed after %q closes the statement", one)
	}
	openers := []string{"+\n", "~\n", "#\n", "(\n", ",\n"}
	for _, one := range openers {
		kinds := scan_kinds(one)
		testify.Not_Equal(t, token.KIND_SEMICOLON, kinds[len(kinds)-1],
			"a line feed after %q closes no statement", one)
	}
	testify.Equal(t, []token.Kind{
		token.KIND_IDENTIFIER, token.KIND_SEMICOLON, token.KIND_IDENTIFIER,
		token.KIND_SEMICOLON,
	}, scan_kinds("a\nb"), "one semicolon for each closed line")
	testify.Equal(t, []token.Kind{token.KIND_IDENTIFIER, token.KIND_SEMICOLON},
		scan_kinds("a"), "the end of the source closes the final line")
}

func test_illegal_input(t *testing.T) {
	singles := []string{"#", "$", "@", "?", "\\", "\x00", "\x01", "\x02", "\xff"}
	for _, one := range singles {
		source := token.Source(one)
		scanner := token.Scanner{Source: source}
		scanned := token.Scan(&scanner)
		testify.Equal(t, token.KIND_ILLEGAL, scanned.Kind,
			"the byte %q opens no token", one)
		testify.Equal(t, 1, int(scanned.Size), "the byte %q scans one byte wide", one)
	}
	triples := []string{"\x00\x00\x00", "\x01\x01\x01", "\x02\x02\x02", "\xff\xff\xff"}
	for _, one := range triples {
		testify.Equal(t, []token.Kind{
			token.KIND_ILLEGAL, token.KIND_ILLEGAL, token.KIND_ILLEGAL,
		}, scan_kinds(one), "three illegal bytes of %q advance the cursor", one)
	}
}

func test_bounds(t *testing.T) {
	t.Parallel()
	large_identifier_bounds(t)
	large_number_bounds(t)
	large_literal_bounds(t)
	large_comment_bounds(t)
	large_operator_bounds(t)
	oversized_source_bounds(t)
}

func large_identifier_bounds(t *testing.T) {
	source := filled_source("a", 'a', "a")
	scanner := token.Scanner{Source: source}
	one := token.Scan(&scanner)
	testify.Equal(t, token.KIND_IDENTIFIER, one.Kind, "maximum identifier kind")
	testify.Equal(t, TEST_LARGE_SIZE, int(one.Size), "maximum identifier size")
	testify.Equal(t, TEST_LARGE_SIZE, len(token.Text(source, one)), "maximum token text")
	two := token.Scan(&scanner)
	testify.Equal(t, token.KIND_SEMICOLON, two.Kind, "maximum identifier semicolon")
	testify.Equal(t, TEST_LARGE_SIZE, int(two.Offset), "maximum offset")
	three := token.Scan(&scanner)
	testify.Equal(t, token.KIND_END_OF_FILE, three.Kind, "maximum end of file")
	testify.Empty(t, token.Text(source, three), "the end of file spans no byte")
}

func large_number_bounds(t *testing.T) {
	source := filled_source("0", '0', "0")
	scanner := token.Scanner{Source: source}
	one := token.Scan(&scanner)
	testify.Equal(t, token.KIND_INTEGER, one.Kind, "maximum number kind")
	testify.Equal(t, TEST_LARGE_SIZE, int(one.Size), "maximum number size")
}

func large_literal_bounds(t *testing.T) {
	source := filled_source("`", 'a', "`")
	scanner := token.Scanner{Source: source}
	one := token.Scan(&scanner)
	testify.Equal(t, token.KIND_STRING, one.Kind, "maximum raw literal kind")
	testify.Equal(t, TEST_LARGE_SIZE, int(one.Size), "maximum raw literal size")
}

func large_comment_bounds(t *testing.T) {
	source := filled_source("/*", 'a', "*/")
	scanner := token.Scanner{Source: source}
	one := token.Scan(&scanner)
	testify.Equal(t, token.KIND_COMMENT, one.Kind, "maximum comment kind")
	testify.Equal(t, TEST_LARGE_SIZE, int(one.Size), "maximum comment size")
}

func large_operator_bounds(t *testing.T) {
	period := filled_source(".", ' ', " ")
	period_scanner := token.Scanner{Source: period}
	testify.Equal(t, token.KIND_PERIOD, token.Scan(&period_scanner).Kind,
		"maximum tail behind a period")
	slash := filled_source("/", ' ', " ")
	slash_scanner := token.Scanner{Source: slash}
	testify.Equal(t, token.KIND_SLASH, token.Scan(&slash_scanner).Kind,
		"maximum tail behind a slash")
}

func oversized_source_bounds(t *testing.T) {
	buffer := make([]byte, TEST_LARGE_SIZE+1)
	scanner := token.Scanner{Source: token.Source(buffer)}
	testify.Panics(t, func() { token.Scan(&scanner) }, "oversized source")
}

func test_allocation(t *testing.T) {
	fixture := allocation_fixture{}
	source := token.Source("a + 1")
	checks := []struct {
		Name string
		Call func()
	}{
		{Name: "Scan", Call: func() {
			fixture.Scanner = token.Scanner{Source: source}
			fixture.Token = token.Scan(&fixture.Scanner)
		}},
		{Name: "Text", Call: func() {
			fixture.Text = token.Text(source, fixture.Token)
		}},
		{Name: "Index_Lines", Call: func() {
			fixture.Indexed = token.Index_Lines(&fixture.Index, source)
		}},
		{Name: "Position_Of", Call: func() {
			fixture.Line, fixture.Column = token.Position_Of(&fixture.Index, 2)
		}},
	}
	for _, check := range checks {
		t.Run(check.Name, func(t *testing.T) { testify.Zero_Allocation(t, check.Call) })
	}
	testify.Not_Empty(t, fixture.Text, "the scanned token spans source bytes")
	testify.True(t, bool(fixture.Indexed), "the allocation fixture indexes its source")
	testify.Equal(t, token.Line(1), fixture.Line, "the allocation fixture reads one line")
}

func test_positions(t *testing.T) {
	index := new(token.Line_Index)
	source := token.Source("package one\n\nfunc Fold() {\n\treturn\n}\n")
	testify.True(t, bool(token.Index_Lines(index, source)), "a source of few lines indexes")
	for _, one := range []struct {
		Offset token.Offset
		Line   token.Line
		Column token.Column
	}{
		{0, 1, 1},
		{1, 1, 2},
		{11, 1, 12},
		{12, 2, 1},
		{13, 3, 1},
		{27, 4, 1},
		{28, 4, 2},
		{token.Offset(len(source)), 6, 1},
	} {
		line, column := token.Position_Of(index, one.Offset)
		testify.Equal(t, one.Line, line, "the offset %d stands on its line", one.Offset)
		testify.Equal(t, one.Column, column, "the offset %d names its column",
			one.Offset)
	}
	positions_empty(t, index)
	positions_widest(t, index)
	positions_past_bound(t, index)
}

func positions_empty(t *testing.T, index *token.Line_Index) {
	t.Helper()
	for _, one := range []token.Source{
		token.Source(""), token.Source("a"), token.Source("a\n"),
	} {
		testify.True(t, bool(token.Index_Lines(index, one)),
			"a source of %d bytes indexes", len(one))
		line, column := token.Position_Of(index, 0)
		testify.Equal(t, token.Line(1), line, "a short source opens on its first line")
		testify.Equal(t, token.Column(1), column,
			"a short source opens at its first column")
	}
	line, column := token.Position_Of(index, 2)
	testify.Equal(t, token.Line(2), line, "the byte after a line feed opens the next line")
	testify.Equal(t, token.Column(1), column, "the next line opens at the first column")
}

func positions_widest(t *testing.T, index *token.Line_Index) {
	t.Helper()
	wide := make([]byte, token.SOURCE_SIZE_MAXIMUM)
	for slot := range wide {
		wide[slot] = 'a'
	}
	testify.True(t, bool(token.Index_Lines(index, token.Source(wide))),
		"a source of one widest line indexes")
	line, column := token.Position_Of(index, token.Offset(len(wide)-1))
	testify.Equal(t, token.Line(1), line, "one line holds the widest source")
	testify.Equal(t, token.Column(token.COLUMN_MAXIMUM-1), column,
		"the final byte of the widest line names its column")
	line, column = token.Position_Of(index, token.OFFSET_MAXIMUM)
	testify.Equal(t, token.Line(1), line,
		"the offset past the widest source stands on its line")
	testify.Equal(t, token.Column(token.COLUMN_MAXIMUM), column,
		"the offset past the widest source names the widest column")
}

func positions_past_bound(t *testing.T, index *token.Line_Index) {
	t.Helper()
	filled := make([]byte, token.LINE_COUNT_MAXIMUM-1)
	for slot := range filled {
		filled[slot] = '\n'
	}
	testify.True(t, bool(token.Index_Lines(index, token.Source(filled))),
		"a source of every line the index holds indexes")
	line, column := token.Position_Of(index, token.Offset(len(filled)))
	testify.Equal(t, token.Line(token.LINE_MAXIMUM), line, "the final line reads back")
	testify.Equal(t, token.Column(1), column, "the final line opens at the first column")
	line, _ = token.Position_Of(index, 1)
	testify.Equal(t, token.Line(2), line, "the second line reads back")
	past := make([]byte, token.LINE_COUNT_MAXIMUM)
	for slot := range past {
		past[slot] = '\n'
	}
	testify.False(t, bool(token.Index_Lines(index, token.Source(past))),
		"a source past the line bound is refused")
}
