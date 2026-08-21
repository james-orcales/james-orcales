package scanner_test

import (
	"testing"

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/strconv"
	"local/james-orcales/shared/strings"
	"local/james-orcales/shared/testify"
	"local/james-orcales/shared/text/scanner"
	"local/james-orcales/shared/unicode/utf8"
)

// Test_Source keeps hostile size refusal outside scanner state mutation.
func Test_Source(t *testing.T) {
	maximum := text_repeat('x', scanner.SOURCE_SIZE_MAXIMUM)
	source, status := scanner.Source_Validate(scanner.Source_Unvalidated(maximum))
	testify.Equal_Values(t, scanner.STATUS_OK, status)
	testify.Equal(t, maximum, string(source))

	oversized := maximum + "x"
	source, status = scanner.Source_Validate(scanner.Source_Unvalidated(oversized))
	testify.Equal_Values(t, scanner.STATUS_INPUT_INVALID, status)
	testify.Equal(t, "", string(source))
}

// Test_Tokens protects standard token kinds and caller recognition policy.
func Test_Tokens(t *testing.T) {
	source := source_from(t, "alpha 0x2a 3.5 'x' \"go\" `raw` // note\n")
	var subject scanner.Scanner
	scanner.Scanner_Init(&subject, source)
	wanted := [...]struct {
		Token scanner.Token
		Text  string
	}{
		{scanner.TOKEN_IDENTIFIER, "alpha"},
		{scanner.TOKEN_INTEGER, "0x2a"},
		{scanner.TOKEN_FLOAT, "3.5"},
		{scanner.TOKEN_CHARACTER, "'x'"},
		{scanner.TOKEN_STRING, "\"go\""},
		{scanner.TOKEN_RAW_STRING, "`raw`"},
		{scanner.TOKEN_EOF, ""},
	}
	for _, one := range wanted {
		token := scanner.Scanner_Scan(&subject)
		testify.Equal(t, one.Token, token)
		testify.Equal(t, one.Text, string(scanner.Scanner_Token_Text(&subject)))
	}
	test_mode(t)
	test_identifier_function(t)
}

// Test_Characters protects byte-order-mark elision and non-consuming Peek.
func Test_Characters(t *testing.T) {
	test_characters(t)
}

// Test_Diagnostics keeps malformed input observable without owned text or default IO.
func Test_Diagnostics(t *testing.T) {
	test_diagnostics(t)
}

// Test_Positions keeps line and character column parity across multibyte input.
func Test_Positions(t *testing.T) {
	test_position(t)
}

// Test_Text keeps borrowed token text and caller-owned formatted text.
func Test_Text(t *testing.T) {
	test_text(t)
}

// Test_Bounds reaches each boundary through production entry points.
func Test_Bounds(t *testing.T) {
	test_formulas(t)
	test_bounds(t)
}

// Test_Allocation measures every public path with assertion tracking enabled.
func Test_Allocation(t *testing.T) {
	test_allocation(t)
	test_scan_branch_allocation(t)
}

const TEST_COUNT_ZERO = scanner.SOURCE_SIZE_MINIMUM
const TEST_COUNT_ONE = TEST_COUNT_ZERO + utf8.CHARACTER_SIZE_MINIMUM
const TEST_COUNT_TWO = TEST_COUNT_ONE + TEST_COUNT_ONE
const TEST_COUNT_THREE = TEST_COUNT_TWO + TEST_COUNT_ONE
const TEST_COUNT_TWELVE = strconv.DECIMAL_BASE + TEST_COUNT_TWO

// Lexical and diagnostic formulas must stay attached to authoritative domains.
func test_formulas(t *testing.T) {
	testify.Equal(t, strings.TEXT_SIZE_MAXIMUM, scanner.SOURCE_SIZE_MAXIMUM)
	testify.Equal(t, int(-scanner.TOKEN_COMMENT), scanner.TOKEN_PREDEFINED_COUNT)
	testify.Equal(t,
		scanner.TOKEN_PREDEFINED_COUNT+utf8.CHARACTER_SIZE_TWO,
		scanner.MODE_BIT_COUNT,
	)
	testify.Equal_Values(t,
		scanner.ERROR_CODE_MAXIMUM-scanner.ERROR_NONE+utf8.CHARACTER_SIZE_MINIMUM,
		scanner.ERROR_CODE_COUNT,
	)
	testify.Equal(t, len("::"), scanner.POSITION_SEPARATOR_COUNT)
	testify.Equal(t, scanner.POSITION_SEPARATOR_COUNT, scanner.POSITION_COMPONENT_COUNT)
	testify.Equal(t,
		scanner.SOURCE_SIZE_MAXIMUM/bits.KILOBYTE_BYTES,
		scanner.POSITION_COORDINATE_DIGIT_COUNT_MAXIMUM,
	)
	testify.Equal(t, strconv.BASE_MINIMUM, scanner.BASE_BINARY)
	testify.Equal(t, strconv.DECIMAL_BASE, scanner.BASE_DECIMAL)
	testify.Equal(t, utf8.CHARACTER_SIZE_TWO, scanner.QUOTED_LITERAL_DELIMITER_COUNT)
}

func test_mode(t *testing.T) {
	var subject scanner.Scanner
	scanner.Scanner_Init(&subject, source_from(t, "name 12 // note\n"))
	subject.Mode = scanner.SCAN_IDENTIFIERS | scanner.SCAN_COMMENTS
	wanted := [...]scanner.Token{
		scanner.TOKEN_IDENTIFIER,
		'1',
		'2',
		scanner.TOKEN_COMMENT,
		scanner.TOKEN_EOF,
	}
	for _, token := range wanted {
		testify.Equal(t, token, scanner.Scanner_Scan(&subject))
	}
}

func test_characters(t *testing.T) {
	var subject scanner.Scanner
	scanner.Scanner_Init(&subject, source_from(t, "\uFEFFa世"))
	testify.Equal(t, scanner.Character('a'), scanner.Scanner_Peek(&subject))
	testify.Equal(t, scanner.Character('a'), scanner.Scanner_Peek(&subject))
	testify.Equal(t, scanner.Character('a'), scanner.Scanner_Next(&subject))
	testify.Equal(t, scanner.Character('世'), scanner.Scanner_Next(&subject))
	testify.Equal(t, scanner.Character(scanner.TOKEN_EOF), scanner.Scanner_Next(&subject))
}

func test_position(t *testing.T) {
	var subject scanner.Scanner
	subject.Position.Filename = "one.go"
	scanner.Scanner_Init(&subject, source_from(t, "a\n世"))
	testify.Equal(t, scanner.TOKEN_IDENTIFIER, scanner.Scanner_Scan(&subject))
	testify.Equal(t, scanner.Line(TEST_COUNT_ONE), subject.Position.Line)
	testify.Equal(t, scanner.Column(TEST_COUNT_ONE), subject.Position.Column)
	testify.Equal(t, scanner.Offset(TEST_COUNT_ZERO), subject.Position.Offset)
	testify.Equal(t, scanner.TOKEN_IDENTIFIER, scanner.Scanner_Scan(&subject))
	testify.Equal(t, scanner.Line(TEST_COUNT_TWO), subject.Position.Line)
	testify.Equal(t, scanner.Column(TEST_COUNT_ONE), subject.Position.Column)
	testify.Equal(t, scanner.Offset(TEST_COUNT_TWO), subject.Position.Offset)

	position := scanner.Scanner_Position(&subject)
	testify.True(t, bool(scanner.Position_Valid(position)))
	testify.Equal(t, scanner.Offset(len("a\n世")), position.Offset)
	testify.Equal(t, scanner.Line(TEST_COUNT_TWO), position.Line)
	testify.Equal(t, scanner.Column(TEST_COUNT_TWO), position.Column)
}

func test_identifier_function(t *testing.T) {
	var subject scanner.Scanner
	scanner.Scanner_Init(&subject, source_from(t, "%word"))
	subject.Identifier = func(
		character scanner.Character, index scanner.Character_Index,
	) (accepted bool) {
		return character == '%' && index == scanner.CHARACTER_INDEX_MINIMUM ||
			character >= 'a' && character <= 'z' &&
				index > scanner.CHARACTER_INDEX_MINIMUM
	}
	testify.Equal(t, scanner.TOKEN_IDENTIFIER, scanner.Scanner_Scan(&subject))
	testify.Equal(t, "%word", string(scanner.Scanner_Token_Text(&subject)))
}

func test_diagnostics(t *testing.T) {
	cases := [...]struct {
		Source string
		Code   scanner.Error_Code
	}{
		{"\x00", scanner.ERROR_INVALID_NUL},
		{"\xff", scanner.ERROR_INVALID_UTF8},
		{"\"open", scanner.ERROR_LITERAL_NOT_TERMINATED},
		{"/* open", scanner.ERROR_COMMENT_NOT_TERMINATED},
		{"0x", scanner.ERROR_LITERAL_HAS_NO_DIGITS},
	}
	for _, one := range cases {
		var subject scanner.Scanner
		var observed scanner.Error
		scanner.Scanner_Init(&subject, source_from(t, one.Source))
		subject.Error = func(report scanner.Error) {
			observed = report
		}
		scanner.Scanner_Scan(&subject)
		testify.Equal(t, one.Code, observed.Code)
		testify.True(t, bool(scanner.Position_Valid(observed.Position)))
		testify.True(t, subject.Error_Count > scanner.ERROR_COUNT_MINIMUM)
	}
}

func test_text(t *testing.T) {
	var token_storage [scanner.TOKEN_TEXT_SIZE_MAXIMUM]byte
	token_count, status := scanner.Token_Append_Into(
		token_storage[:], scanner.TOKEN_IDENTIFIER,
	)
	testify.Equal_Values(t, scanner.STATUS_OK, status)
	testify.Equal(t, "Ident", string(token_storage[:token_count]))
	token_count, status = scanner.Token_Append_Into(token_storage[:], '\n')
	testify.Equal_Values(t, scanner.STATUS_OK, status)
	testify.Equal(t, `'\n'`, string(token_storage[:token_count]))

	position := scanner.Position{
		Filename: "one.go",
		Offset:   TEST_COUNT_ZERO,
		Line:     TEST_COUNT_TWELVE,
		Column:   TEST_COUNT_THREE,
	}
	var position_storage [scanner.POSITION_TEXT_SIZE_MAXIMUM]byte
	position_count, status := scanner.Position_Append_Into(position_storage[:], position)
	testify.Equal_Values(t, scanner.STATUS_OK, status)
	testify.Equal(t, "one.go:12:3", string(position_storage[:position_count]))
	position_count, status = scanner.Position_Append_Into(
		position_storage[:], scanner.Position{},
	)
	testify.Equal_Values(t, scanner.STATUS_OK, status)
	testify.Equal(t, "<input>", string(position_storage[:position_count]))

	position_count, status = scanner.Position_Append_Into(
		position_storage[:TEST_COUNT_ZERO], position,
	)
	testify.Equal(t, scanner.Position_Count(len("one.go:12:3")), position_count)
	testify.Equal_Values(t, scanner.STATUS_OUTPUT_TOO_SMALL, status)
}

func test_allocation(t *testing.T) {
	valid := scanner.Source_Unvalidated("alpha 42")
	var source scanner.Source
	var validation_status scanner.Validation_Status
	var subject scanner.Scanner
	var token scanner.Token
	var character scanner.Character
	var text scanner.Text
	var position scanner.Position
	var position_valid scanner.Boolean
	var token_count scanner.Token_Count
	var position_count scanner.Position_Count
	var status scanner.Status
	var token_storage [scanner.TOKEN_TEXT_SIZE_MAXIMUM]byte
	var position_storage [scanner.POSITION_TEXT_SIZE_MAXIMUM]byte

	testify.Zero_Allocation(t, func() {
		source, validation_status = scanner.Source_Validate(valid)
	})
	testify.Zero_Allocation(t, func() {
		scanner.Scanner_Init(&subject, source)
	})
	testify.Zero_Allocation(t, func() {
		character = scanner.Scanner_Peek(&subject)
	})
	testify.Zero_Allocation(t, func() {
		character = scanner.Scanner_Next(&subject)
	})
	scanner.Scanner_Init(&subject, source)
	testify.Zero_Allocation(t, func() {
		token = scanner.Scanner_Scan(&subject)
	})
	testify.Zero_Allocation(t, func() {
		text = scanner.Scanner_Token_Text(&subject)
	})
	testify.Zero_Allocation(t, func() {
		position = scanner.Scanner_Position(&subject)
	})
	testify.Zero_Allocation(t, func() {
		position_valid = scanner.Position_Valid(position)
	})
	testify.Zero_Allocation(t, func() {
		token_count, status = scanner.Token_Append_Into(token_storage[:], token)
	})
	testify.Zero_Allocation(t, func() {
		position_count, status = scanner.Position_Append_Into(
			position_storage[:], position,
		)
	})

	testify.Equal_Values(t, scanner.STATUS_OK, validation_status)
	testify.True(t, character != scanner.Character(scanner.TOKEN_EOF))
	testify.True(t, len(text) > scanner.SOURCE_SIZE_MINIMUM)
	testify.True(t, token_count > TEST_COUNT_ZERO)
	testify.True(t, position_count > TEST_COUNT_ZERO)
	testify.True(t, bool(position_valid))
	testify.Equal_Values(t, scanner.STATUS_OK, status)
}

func test_scan_branch_allocation(t *testing.T) {
	t.Helper()
	cases := [...]struct {
		Source     string
		Mode       scanner.Mode
		Identifier scanner.Identifier_Function
	}{
		{
			Source: "alpha 0b1 0o7 0xF 1.25e2 'x' \"go\" `raw` // note\n/* block */",
			Mode:   scanner.GO_TOKENS,
		},
		{
			Source:     "$value\x00\xff",
			Mode:       scanner.SCAN_IDENTIFIERS,
			Identifier: allocation_identifier,
		},
		{Source: "0xg 1e+ 'x \"unterminated /*", Mode: scanner.GO_TOKENS},
	}
	for _, one := range cases {
		source := source_from(t, one.Source)
		var subject scanner.Scanner
		var token scanner.Token
		testify.Zero_Allocation(t, func() {
			scanner.Scanner_Init(&subject, source)
			subject.Mode = one.Mode
			subject.Identifier = one.Identifier
			subject.Error = allocation_error
			done := false
			for !done {
				token = scanner.Scanner_Scan(&subject)
				done = token == scanner.TOKEN_EOF
			}
		})
		testify.Equal_Values(t, scanner.TOKEN_EOF, token)
	}
}

func allocation_identifier(
	character scanner.Character, index scanner.Character_Index,
) (accepted bool) {
	if character == '$' {
		return true
	}
	return index > scanner.CHARACTER_INDEX_MINIMUM
}

func allocation_error(report scanner.Error) {
	scanner.Position_Valid(report.Position)
}

func test_bounds(t *testing.T) {
	test_scanner_domains(t)
	test_lexical_cursor_domains(t)
	test_escape_domains(t)
	test_initialization_domains(t)
	test_character_domains(t)
	test_configuration_domains(t)
	test_token_domains(t)
	test_position_domains(t)
	test_scanner_position_domains(t)
	test_error_count_domains(t)
	test_error_position_domains(t)
}

func test_lexical_cursor_domains(t *testing.T) {
	test_lexical_starter_domains(t)
	test_lexical_state_tail_domains(t)
}

func test_lexical_starter_domains(t *testing.T) {
	starters := [...]scanner.Character{'a', '0', '1', '"', '/', '`'}
	for _, starter := range starters {
		minimum := scanner_domain_state("", "", TEST_COUNT_ZERO)
		minimum.Mode = scanner.Mode(scanner.MODE_VALUE_MAXIMUM)
		minimum.Whitespace = scanner.WHITESPACE_MINIMUM
		minimum.Cursor.Character = scanner.Look_Ahead_Character(starter)
		minimum.Cursor.Looked = true
		scanner.Scanner_Scan(&minimum)

		maximum_source := scanner.Source(
			text_repeat('x', scanner.SOURCE_SIZE_MAXIMUM),
		)
		maximum_filename := scanner.Filename(
			text_repeat('f', scanner.FILENAME_SIZE_MAXIMUM),
		)
		maximum := scanner_domain_state(
			maximum_source, maximum_filename, scanner.SOURCE_SIZE_MAXIMUM,
		)
		maximum = scanner_maximum_lexical_state(maximum)
		maximum.Mode = scanner.Mode(scanner.MODE_VALUE_MAXIMUM)
		maximum.Whitespace = scanner.WHITESPACE_MINIMUM
		maximum.Cursor.Character = scanner.Look_Ahead_Character(starter)
		maximum.Cursor.Looked = true
		scanner.Scanner_Scan(&maximum)

		for _, value := range [...]int{TEST_COUNT_ONE, TEST_COUNT_TWO} {
			source := scanner.Source(text_repeat('x', value))
			state := scanner_domain_state(
				source, scanner.Filename(text_repeat('f', value)), value,
			)
			state.Cursor.Decoder.Source_Offset = scanner.Source_Offset(value)
			state.Mode = scanner.Mode(scanner.MODE_VALUE_MAXIMUM)
			state.Whitespace = scanner.WHITESPACE_MINIMUM
			state.Cursor.Character = scanner.Look_Ahead_Character(starter)
			state.Cursor.Looked = true
			scanner.Scanner_Scan(&state)
		}
	}
}

func test_lexical_state_tail_domains(t *testing.T) {
	tails := [...]struct {
		Starter scanner.Character
		Tail    string
	}{
		{'/', "/"},
		{'/', "*"},
		{'"', "\\x"},
		{'"', "\\777"},
		{'"', "\\u0000"},
		{'"', "\\U00000000"},
		{'"', "\\\x01"},
		{'"', "\\\x02"},
		{'"', "\\" + string(rune(scanner.TOKEN_MAXIMUM))},
		{'0', "x0p0"},
		{'1', "e0"},
		{'1', "p0"},
		{'.', "0"},
	}
	for _, one := range tails {
		padding_count := scanner.SOURCE_SIZE_MAXIMUM - len(one.Tail)
		source := scanner.Source(text_repeat('x', padding_count) + one.Tail)
		state := scanner_domain_state(
			source,
			scanner.Filename(text_repeat('f', scanner.FILENAME_SIZE_MAXIMUM)),
			scanner.SOURCE_SIZE_MAXIMUM,
		)
		state = scanner_maximum_lexical_state(state)
		state.Cursor.Decoder.Source_Offset = scanner.Source_Offset(padding_count)
		state.Mode = scanner.Mode(scanner.MODE_VALUE_MAXIMUM)
		state.Whitespace = scanner.WHITESPACE_MINIMUM
		state.Cursor.Character = scanner.Look_Ahead_Character(one.Starter)
		state.Cursor.Looked = true
		scanner.Scanner_Scan(&state)

		for _, value := range [...]int{TEST_COUNT_ONE, TEST_COUNT_TWO} {
			small := scanner_domain_state(
				scanner.Source(one.Tail),
				scanner.Filename(text_repeat('f', value)),
				value,
			)
			small.Cursor.Decoder.Source_Offset = scanner.OFFSET_MINIMUM
			small.Mode = scanner.Mode(scanner.MODE_VALUE_MAXIMUM)
			small.Whitespace = scanner.WHITESPACE_MINIMUM
			small.Cursor.Character = scanner.Look_Ahead_Character(one.Starter)
			small.Cursor.Looked = true
			scanner.Scanner_Scan(&small)
		}
	}
}

func scanner_maximum_lexical_state(input scanner.Scanner) (maximum scanner.Scanner) {
	maximum = input
	maximum.Error_Count = scanner.ERROR_COUNT_MAXIMUM
	maximum.Cursor.Decoder.Line = scanner.LINE_MAXIMUM
	maximum.Cursor.Decoder.Column = scanner.COLUMN_MAXIMUM
	maximum.Cursor.Decoder.Character_Line = scanner.LINE_MAXIMUM
	maximum.Cursor.Decoder.Character_Column = scanner.COLUMN_MAXIMUM
	return maximum
}

func test_escape_domains(t *testing.T) {
	maximum := string(rune(scanner.TOKEN_MAXIMUM))
	sources := [...]string{
		"\"\\x",
		"\"\\x" + maximum + "\"",
		"\"\\x\x00\"",
		"\"\\x\x01\"",
		"\"\\x\x02\"",
		"\"\\x00",
		"\"\\x00\x00\"",
		"\"\\x00\x01\"",
		"\"\\x00\x02\"",
		"\"\\x00" + maximum + "\"",
		"\"\\x00\"",
		"'\\123'",
		"\"\\u0000\"",
		"\"\\U00000000\"",
		"\"\\x10\"",
		"\"\\x20\"",
	}
	for _, source := range sources {
		var subject scanner.Scanner
		scanner.Scanner_Init(&subject, source_from(t, source))
		scanner.Scanner_Scan(&subject)
	}
	test_lexical_tail_domains(t, maximum)
}

func test_lexical_tail_domains(t *testing.T, maximum string) {
	sources := [...]string{
		"0b.0",
		"0o.0",
		"0x.0",
		".9",
		"9",
		"0x_",
		"0_",
		"0b2.0",
		"0b9.0",
		"0b2",
		"0x" + maximum,
		"1" + maximum,
		"x" + maximum,
		"/" + maximum,
		"/*x*/" + maximum,
	}
	for _, source := range sources {
		var subject scanner.Scanner
		scanner.Scanner_Init(&subject, source_from(t, source))
		scanner.Scanner_Scan(&subject)
	}
	for _, character := range [...]rune{TEST_COUNT_ZERO, TEST_COUNT_ONE, TEST_COUNT_TWO} {
		tail := string(character)
		for _, source := range [...]string{
			"0x" + tail,
			"1" + tail,
			"x" + tail,
			"/" + tail,
			"/*x*/" + tail,
		} {
			var subject scanner.Scanner
			scanner.Scanner_Init(&subject, source_from(t, source))
			scanner.Scanner_Scan(&subject)
		}
	}

	separator_source := text_repeat(
		'1', scanner.SOURCE_SIZE_MAXIMUM-utf8.CHARACTER_SIZE_MINIMUM,
	) + "_"
	var separator scanner.Scanner
	scanner.Scanner_Init(&separator, source_from(t, separator_source))
	scanner.Scanner_Scan(&separator)

	string_source := "\"" + text_repeat(
		'x', scanner.SOURCE_SIZE_MAXIMUM-scanner.QUOTED_LITERAL_DELIMITER_COUNT,
	) + "\""
	var string_subject scanner.Scanner
	scanner.Scanner_Init(&string_subject, source_from(t, string_source))
	scanner.Scanner_Scan(&string_subject)
}

func test_error_position_domains(t *testing.T) {
	cases := [...]struct {
		Filename scanner.Filename
		Source   string
	}{
		{"f", "\x00"},
		{"ff", "\n\x00"},
		{
			scanner.Filename(text_repeat('f', scanner.FILENAME_SIZE_MAXIMUM)),
			text_repeat(
				'\n', scanner.SOURCE_SIZE_MAXIMUM-utf8.CHARACTER_SIZE_MINIMUM,
			) + "\x00",
		},
	}
	for _, one := range cases {
		var subject scanner.Scanner
		subject.Position.Filename = one.Filename
		scanner.Scanner_Init(&subject, source_from(t, one.Source))
		scanner.Scanner_Scan(&subject)
	}
}

func test_scanner_domains(t *testing.T) {
	maximum_source := scanner.Source(text_repeat('x', scanner.SOURCE_SIZE_MAXIMUM))
	maximum_filename := scanner.Filename(
		text_repeat('f', scanner.FILENAME_SIZE_MAXIMUM),
	)
	states := [...]scanner.Scanner{
		{},
		{
			Cursor: scanner.Scanner_Cursor{
				Character: scanner.Look_Ahead_Character(scanner.TOKEN_EOF),
				Looked:    true,
			},
			Initialized: true,
		},
		scanner_domain_state("x", "f", TEST_COUNT_ONE),
		scanner_domain_state("xx", "ff", TEST_COUNT_TWO),
		{
			Error_Count: scanner.ERROR_COUNT_MAXIMUM,
			Mode:        scanner.Mode(scanner.MODE_VALUE_MAXIMUM),
			Whitespace:  scanner.WHITESPACE_MAXIMUM,
			Position: scanner.Position{
				Filename: maximum_filename,
				Offset:   scanner.OFFSET_MAXIMUM,
				Line:     scanner.LINE_MAXIMUM,
				Column:   scanner.COLUMN_MAXIMUM,
			},
			Cursor: scanner.Scanner_Cursor{
				Decoder: scanner.Scanner_Decoder{
					Source:           maximum_source,
					Source_Offset:    scanner.SOURCE_SIZE_MAXIMUM,
					Line:             scanner.LINE_MAXIMUM,
					Column:           scanner.COLUMN_MAXIMUM,
					Character_Offset: scanner.OFFSET_MAXIMUM,
					Character_Line:   scanner.LINE_MAXIMUM,
					Character_Column: scanner.COLUMN_MAXIMUM,
				},
				Character: scanner.Look_Ahead_Character(
					scanner.TOKEN_MAXIMUM,
				),
				Looked: true,
			},
			Token: scanner.Scanner_Token_Bounds{
				Start: scanner.SOURCE_SIZE_MAXIMUM,
				End:   scanner.SOURCE_SIZE_MAXIMUM,
			},
			Initialized: true,
		},
	}
	for _, state := range states {
		initialized := state
		scanner.Scanner_Init(&initialized, state.Cursor.Decoder.Source)
		peek := state
		scanner.Scanner_Peek(&peek)
		next := state
		scanner.Scanner_Next(&next)
		scan := state
		scanner.Scanner_Scan(&scan)
		position := state
		scanner.Scanner_Position(&position)
		text := state
		scanner.Scanner_Token_Text(&text)
	}
}

func scanner_domain_state(source scanner.Source, filename scanner.Filename, value int) (
	state scanner.Scanner,
) {
	state.Error_Count = scanner.Error_Count(value)
	state.Mode = scanner.Mode(value)
	state.Whitespace = scanner.Whitespace(value)
	state.Position = scanner.Position{
		Filename: filename,
		Offset:   scanner.Offset(value),
		Line:     scanner.Line(value),
		Column:   scanner.Column(value),
	}
	state.Cursor = scanner.Scanner_Cursor{
		Decoder: scanner.Scanner_Decoder{
			Source:           source,
			Source_Offset:    scanner.Source_Offset(value),
			Line:             scanner.Source_Line(value),
			Column:           scanner.Source_Column(value),
			Character_Offset: scanner.Character_Offset(value),
			Character_Line:   scanner.Character_Line(value),
			Character_Column: scanner.Character_Column(value),
		},
		Character: scanner.Look_Ahead_Character(value),
		Looked:    true,
	}
	state.Token = scanner.Scanner_Token_Bounds{
		Start: scanner.Token_Start(value),
		End:   scanner.Token_End(value),
	}
	state.Initialized = true
	return state
}

func test_initialization_domains(t *testing.T) {
	maximum_source := source_from(
		t, text_repeat('x', scanner.SOURCE_SIZE_MAXIMUM),
	)
	filenames := [...]scanner.Filename{
		"",
		"x",
		"xx",
		scanner.Filename(text_repeat('f', scanner.FILENAME_SIZE_MAXIMUM)),
	}
	for _, filename := range filenames {
		var subject scanner.Scanner
		subject.Position.Filename = filename
		scanner.Scanner_Init(&subject, maximum_source)
	}
}

func test_character_domains(t *testing.T) {
	sources := [...]string{
		"",
		string(rune(scanner.TOKEN_MAXIMUM)),
		"\x00",
		"\x01",
		"\x02",
	}
	for _, source := range sources {
		var peek scanner.Scanner
		scanner.Scanner_Init(&peek, source_from(t, source))
		scanner.Scanner_Peek(&peek)
		var next scanner.Scanner
		scanner.Scanner_Init(&next, source_from(t, source))
		scanner.Scanner_Next(&next)
	}
}

func test_configuration_domains(t *testing.T) {
	modes := [...]scanner.Mode{
		scanner.MODE_MINIMUM,
		scanner.Mode(TEST_COUNT_ONE),
		scanner.Mode(TEST_COUNT_TWO),
		scanner.Mode(scanner.MODE_VALUE_MAXIMUM),
	}
	for _, mode := range modes {
		var subject scanner.Scanner
		scanner.Scanner_Init(&subject, source_from(t, "x"))
		subject.Mode = mode
		scanner.Scanner_Scan(&subject)
	}
	whitespace_values := [...]scanner.Whitespace{
		scanner.WHITESPACE_MINIMUM,
		scanner.Whitespace(TEST_COUNT_ONE),
		scanner.Whitespace(TEST_COUNT_TWO),
		scanner.WHITESPACE_MAXIMUM,
	}
	for _, whitespace := range whitespace_values {
		var subject scanner.Scanner
		scanner.Scanner_Init(&subject, source_from(t, "x"))
		subject.Whitespace = whitespace
		scanner.Scanner_Scan(&subject)
	}
}

func test_token_domains(t *testing.T) {
	sources := [...]string{
		"\x00",
		"\x01",
		"\x02",
		string(rune(scanner.TOKEN_MAXIMUM)),
	}
	for _, source := range sources {
		var subject scanner.Scanner
		scanner.Scanner_Init(&subject, source_from(t, source))
		subject.Mode = scanner.MODE_MINIMUM
		scanner.Scanner_Scan(&subject)
	}
	var comment scanner.Scanner
	scanner.Scanner_Init(&comment, source_from(t, "//x"))
	comment.Mode = scanner.SCAN_COMMENTS
	scanner.Scanner_Scan(&comment)

	var maximum_source scanner.Scanner
	scanner.Scanner_Init(
		&maximum_source,
		source_from(t, text_repeat('x', scanner.SOURCE_SIZE_MAXIMUM)),
	)
	scanner.Scanner_Scan(&maximum_source)
	scanner.Scanner_Token_Text(&maximum_source)

	var destination [scanner.TOKEN_TEXT_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		TEST_COUNT_ZERO, TEST_COUNT_ONE, TEST_COUNT_TWO, scanner.TOKEN_TEXT_SIZE_MAXIMUM,
	} {
		scanner.Token_Append_Into(destination[:size], scanner.TOKEN_EOF)
	}
	tokens := [...]scanner.Token{
		scanner.TOKEN_COMMENT,
		scanner.TOKEN_EOF,
		scanner.Token(TEST_COUNT_ZERO),
		scanner.Token(TEST_COUNT_ONE),
		scanner.Token(TEST_COUNT_TWO),
		scanner.Token(scanner.TOKEN_MAXIMUM),
		scanner.TOKEN_INTEGER,
		scanner.TOKEN_CHARACTER,
		scanner.TOKEN_FLOAT,
	}
	for _, token := range tokens {
		scanner.Token_Append_Into(destination[:], token)
	}
}

func test_position_domains(t *testing.T) {
	maximum_filename := scanner.Filename(
		text_repeat('f', scanner.FILENAME_SIZE_MAXIMUM),
	)
	positions := [...]scanner.Position{
		{},
		{Filename: "x"},
		{Filename: "xx"},
		{Filename: "x", Offset: TEST_COUNT_ONE, Line: TEST_COUNT_ONE,
			Column: TEST_COUNT_ONE},
		{Filename: "xx", Offset: TEST_COUNT_TWO, Line: TEST_COUNT_TWO,
			Column: TEST_COUNT_TWO},
		{Filename: "xxx", Offset: TEST_COUNT_THREE, Line: TEST_COUNT_THREE,
			Column: TEST_COUNT_THREE},
		{
			Filename: maximum_filename,
			Offset:   scanner.OFFSET_MAXIMUM,
			Line:     scanner.LINE_MAXIMUM,
			Column:   scanner.COLUMN_MAXIMUM,
		},
	}
	var destination [scanner.POSITION_TEXT_SIZE_MAXIMUM]byte
	for _, position := range positions {
		scanner.Position_Valid(position)
		scanner.Position_Append_Into(destination[:], position)
	}
	for _, size := range [...]int{
		TEST_COUNT_ZERO, TEST_COUNT_ONE, TEST_COUNT_TWO, scanner.POSITION_TEXT_SIZE_MAXIMUM,
	} {
		scanner.Position_Append_Into(
			destination[:size], positions[len(positions)-utf8.CHARACTER_SIZE_MINIMUM],
		)
	}
}

func test_scanner_position_domains(t *testing.T) {
	maximum_filename := scanner.Filename(
		text_repeat('f', scanner.FILENAME_SIZE_MAXIMUM),
	)
	cases := [...]struct {
		Filename scanner.Filename
		Source   string
		Scan     bool
	}{
		{"", "", false},
		{"x", "x", true},
		{"xx", "\n\n", true},
		{maximum_filename, text_repeat('x', scanner.SOURCE_SIZE_MAXIMUM), true},
		{"", text_repeat('\n', scanner.SOURCE_SIZE_MAXIMUM), true},
	}
	for _, one := range cases {
		var subject scanner.Scanner
		subject.Position.Filename = one.Filename
		scanner.Scanner_Init(&subject, source_from(t, one.Source))
		if one.Scan {
			scanner.Scanner_Scan(&subject)
		}
		scanner.Scanner_Position(&subject)
	}
}

func test_error_count_domains(t *testing.T) {
	var two scanner.Scanner
	scanner.Scanner_Init(&two, source_from(t, "'"))
	scanner.Scanner_Scan(&two)
	testify.Equal(t, scanner.Error_Count(TEST_COUNT_TWO), two.Error_Count)

	maximum := make([]byte, scanner.SOURCE_SIZE_MAXIMUM)
	for index := TEST_COUNT_ZERO; index < len(maximum)-utf8.CHARACTER_SIZE_MINIMUM; index++ {
		maximum[index] = bits.WORD_8_MAXIMUM
	}
	maximum[len(maximum)-utf8.CHARACTER_SIZE_MINIMUM] = '\''
	var subject scanner.Scanner
	scanner.Scanner_Init(&subject, source_from(t, string(maximum)))
	for scanner.Scanner_Scan(&subject) != scanner.TOKEN_EOF {
	}
	testify.Equal(t, scanner.Error_Count(scanner.ERROR_COUNT_MAXIMUM), subject.Error_Count)
}

func source_from(t *testing.T, value string) (source scanner.Source) {
	t.Helper()
	source, status := scanner.Source_Validate(scanner.Source_Unvalidated(value))
	testify.Equal_Values(t, scanner.STATUS_OK, status)
	return source
}

func text_repeat(character byte, size int) (text string) {
	storage := make([]byte, size)
	for index := range storage {
		storage[index] = character
	}
	return string(storage)
}
