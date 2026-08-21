package scanner_test

import (
	"testing"
	standard_scanner "text/scanner"

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/testify"
	"local/james-orcales/shared/text/scanner"
)

// TestMain keeps allocation probes on production assertion paths.
func TestMain(m *testing.M) {
	aver.Run_Test_Main(m)
}

type source_reader struct {
	Text   string
	Offset int
}

func (subject *source_reader) Read(destination []byte) (count int, failure error) {
	if subject.Offset == len(subject.Text) {
		return TEST_COUNT_ZERO, source_end{}
	}
	count = copy(destination, subject.Text[subject.Offset:])
	subject.Offset += count
	if subject.Offset == len(subject.Text) {
		return count, source_end{}
	}
	return count, nil
}

type source_end struct{}

func (source_end) Error() (message string) {
	return "source end"
}

// Test_Standard_Library_Tokens compares bounded inputs against upstream scanner sequence.
func Test_Standard_Library_Tokens(t *testing.T) {
	cases := [...]string{
		"",
		"a b_c 世",
		"0 0123 0o77 0x1f 0b10 1_000 1.2 3e4 0x1p2",
		"'a' '\\n' \"text\" `raw\ntext`",
		"// line\n/* block */x",
		"+ - * / . , ; : ( ) [ ] { }",
	}
	for _, source := range cases {
		reader := source_reader{Text: source}
		standard := new(standard_scanner.Scanner).Init(&reader)
		standard.Error = func(*standard_scanner.Scanner, string) {}
		var shared scanner.Scanner
		scanner.Scanner_Init(&shared, source_from(t, source))
		done := false
		for !done {
			standard_token := standard.Scan()
			shared_token := scanner.Scanner_Scan(&shared)
			testify.Equal(t, scanner.Token(standard_token), shared_token)
			testify.Equal(
				t,
				standard.TokenText(),
				string(scanner.Scanner_Token_Text(&shared)),
			)
			if standard_token == standard_scanner.EOF {
				done = true
			}
		}
	}
}

// Test_Standard_Library_Positions compares token starts and next position.
func Test_Standard_Library_Positions(t *testing.T) {
	const SOURCE = "one\n  世 two"
	reader := source_reader{Text: SOURCE}
	standard := new(standard_scanner.Scanner).Init(&reader)
	standard.Error = func(*standard_scanner.Scanner, string) {}
	standard.Filename = "one.go"
	var shared scanner.Scanner
	shared.Position.Filename = "one.go"
	scanner.Scanner_Init(&shared, source_from(t, SOURCE))
	done := false
	for !done {
		standard_token := standard.Scan()
		shared_token := scanner.Scanner_Scan(&shared)
		testify.Equal(t, scanner.Token(standard_token), shared_token)
		testify.Equal(t, scanner.Offset(standard.Offset), shared.Position.Offset)
		testify.Equal(t, scanner.Line(standard.Line), shared.Position.Line)
		testify.Equal(t, scanner.Column(standard.Column), shared.Position.Column)
		standard_position := standard.Pos()
		shared_position := scanner.Scanner_Position(&shared)
		testify.Equal(t, scanner.Offset(standard_position.Offset), shared_position.Offset)
		testify.Equal(t, scanner.Line(standard_position.Line), shared_position.Line)
		testify.Equal(t, scanner.Column(standard_position.Column), shared_position.Column)
		if standard_token == standard_scanner.EOF {
			done = true
		}
	}
}

// Test_Standard_Library_Lexical_Corpus covers upstream token and diagnostic classes.
func Test_Standard_Library_Lexical_Corpus(t *testing.T) {
	for _, source := range [...]string{
		"_äöü _本 äöü 本 a۰۱۸ foo६४ bar９８７６",
		"0 00 0b101 0B101 0o377 0O377 0377 0x123abcDEF 0X123abcDEF",
		"0. 1. .42 1.25 1e9 1E9 1e+10 1e-10 0x1.fp2 0X1.FP2",
		"1.5e 1.5E 1e+ 1e- 08 0b2 0xg 1__2",
		`' ' '本' '\n' '\777' '\xff' '\u263a' '\U0000ffAB'`,
		`" " "本" "\n" "\777" "\xff" "\u263a" "\U0000ffAB"`,
		"`` `\\` `raw\ntext`",
		"// line\n/**/ /***/ /* // comment */ /*\ncomment\n*/",
	} {
		compare_scanners(t, source, scanner.GO_TOKENS)
		compare_scanners(t, source, scanner.GO_TOKENS&^scanner.SKIP_COMMENTS)
	}
}

// Test_Standard_Library_Short_Exhaustive checks every short lexical-byte arrangement.
func Test_Standard_Library_Short_Exhaustive(t *testing.T) {
	alphabet := [...]byte{
		byte(TEST_COUNT_ZERO), bits.WORD_8_MAXIMUM,
		'0', '1', 'x', 'e', '_', '.', '/', '*', '\'', '"', '`', '\\',
	}
	modes := [...]scanner.Mode{
		scanner.MODE_MINIMUM,
		scanner.GO_TOKENS,
		scanner.GO_TOKENS &^ scanner.SKIP_COMMENTS,
		scanner.SCAN_IDENTIFIERS,
		scanner.SCAN_INTEGERS,
		scanner.SCAN_FLOATS,
		scanner.SCAN_CHARACTERS,
		scanner.SCAN_STRINGS | scanner.SCAN_RAW_STRINGS,
		scanner.SCAN_COMMENTS,
	}
	var source [len("abc")]byte
	for size := TEST_COUNT_ZERO; size <= len(source); size++ {
		combination_count := TEST_COUNT_ONE
		for range size {
			combination_count *= len(alphabet)
		}
		for combination := range combination_count {
			value := combination
			for index := range size {
				source[index] = alphabet[value%len(alphabet)]
				value /= len(alphabet)
			}
			text := string(source[:size])
			for _, mode := range modes {
				compare_scanners(t, text, mode)
			}
		}
	}
}

func compare_scanners(t *testing.T, source string, mode scanner.Mode) {
	t.Helper()
	reader := source_reader{Text: source}
	standard := new(standard_scanner.Scanner).Init(&reader)
	standard_error_count := TEST_COUNT_ZERO
	standard.Error = func(_ *standard_scanner.Scanner, message string) {
		if message != "source end" {
			standard_error_count++
		}
	}
	standard.Mode = uint(mode)
	validated, validation_status := scanner.Source_Validate(
		scanner.Source_Unvalidated(source),
	)
	testify.Equal_Values(t, scanner.STATUS_OK, validation_status)
	var shared scanner.Scanner
	scanner.Scanner_Init(&shared, validated)
	shared.Mode = mode
	done := false
	for !done {
		standard_token := standard.Scan()
		shared_token := scanner.Scanner_Scan(&shared)
		if scanner.Token(standard_token) != shared_token {
			t.Fatalf("source %q mode %d: shared token %d, standard token %d",
				source, mode, shared_token, standard_token)
		}
		if standard.TokenText() != string(scanner.Scanner_Token_Text(&shared)) {
			t.Fatalf("source %q mode %d: token text differs", source, mode)
		}
		if scanner.Error_Count(standard_error_count) != shared.Error_Count {
			t.Fatalf("source %q mode %d: error count differs", source, mode)
		}
		if standard_token == standard_scanner.EOF {
			done = true
		}
	}
}
