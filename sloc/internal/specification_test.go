package sloc_test

import (
	"strings"
	"testing"
	"testing/fstest"

	sloc "github.com/james-orcales/james-orcales/sloc/internal"
)

// Test_Classify_Comments verifies line comments across languages, a trailing comment
// counting as code, documentation comments, and blanks.
func Test_Classify_Comments(t *testing.T) {
	run_classify_cases(t, sloc.Language_Go(), []classify_case{
		{"go line comment", "// hi\n", 0, 1, 0},
		{"go code", "package main\n", 1, 0, 0},
		{"go trailing comment is code", "total++ // bump\n", 1, 0, 0},
		{"go doc comment", "/// doc\n", 0, 1, 0},
		{"go code, blank, comment", "a()\n\n// c\n", 1, 1, 1},
	})
	run_classify_cases(t, sloc.Language_Python(), []classify_case{
		{"python hash comment", "# hi\n", 0, 1, 0},
		{"python code", "x = 1\n", 1, 0, 0},
		{"python trailing comment is code", "x = 1  # set\n", 1, 0, 0},
	})
	run_classify_cases(t, sloc.Language_Java_Script(), []classify_case{
		{"javascript line comment", "// hi\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_Type_Script(), []classify_case{
		{"typescript line comment", "// t\n", 0, 1, 0},
		{"typescript type annotation is code", "let x: number = 1;\n", 1, 0, 0},
	})
	run_classify_cases(t, sloc.Language_Lua(), []classify_case{
		{"lua line comment", "-- hi\n", 0, 1, 0},
		{"lua code", "local x = 1\n", 1, 0, 0},
	})
	run_classify_cases(t, sloc.Language_Odin(), []classify_case{
		{"odin line comment", "// hi\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_Zig(), []classify_case{
		{"zig line comment", "// hi\n", 0, 1, 0},
		{"zig doc comment", "/// doc\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_C(), []classify_case{
		{"c line comment", "// hi\n", 0, 1, 0},
		{"c block comment", "/* c */\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_Shell(), []classify_case{
		{"shell comment", "# hi\n", 0, 1, 0},
		{"shell hash in string is code", "echo \"a # b\"\n", 1, 0, 0},
	})
	run_classify_cases(t, sloc.Language_Sql(), []classify_case{
		{"sql line comment", "-- hi\n", 0, 1, 0},
		{"sql block comment", "/* c */\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_Php(), []classify_case{
		{"php slash comment", "// hi\n", 0, 1, 0},
		{"php hash comment", "# hi\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_Erlang(), []classify_case{
		{"erlang comment", "% hi\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_Common_Lisp(), []classify_case{
		{"lisp line comment", "; hi\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_Cmake(), []classify_case{
		{"cmake comment", "# hi\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_Fish(), []classify_case{
		{"fish comment", "# hi\n", 0, 1, 0},
		{"fish hash in string is code", "echo \"a # b\"\n", 1, 0, 0},
	})
	run_classify_cases(t, sloc.Language_Nushell(), []classify_case{
		{"nushell comment", "# hi\n", 0, 1, 0},
	})
}

// Test_Classify_Strings verifies a comment token inside a string is code, never a
// comment — the reason a naive prefix match is insufficient.
func Test_Classify_Strings(t *testing.T) {
	run_classify_cases(t, sloc.Language_Go(), []classify_case{
		{"line-comment token in a string", "x := \"http://foo\"\n", 1, 0, 0},
		{"block-comment token in a string", "s := \"/* not a comment */\"\n", 1, 0, 0},
		{"escaped quote in a string", "s := \"she said \\\"hi\\\"\"\n", 1, 0, 0},
	})
	run_classify_cases(t, sloc.Language_Python(), []classify_case{
		{"hash in a string is code", "s = \"a # b\"\n", 1, 0, 0},
	})
	run_classify_cases(t, sloc.Language_Java_Script(), []classify_case{
		{"slashes in a string", "let u = \"http://x\";\n", 1, 0, 0},
	})
}

// Test_Classify_Block_Comments verifies Go block comments do not nest while Rust's do,
// and that JavaScript block comments span lines.
func Test_Classify_Block_Comments(t *testing.T) {
	run_classify_cases(t, sloc.Language_Go(), []classify_case{
		{"single line block", "/* c */\n", 0, 1, 0},
		{"block spanning lines", "/* a\nb\n*/\n", 0, 3, 0},
		{"code after close", "/* a */ x()\n", 1, 0, 0},
		{"code before open, spanning", "x() /* open\nstill\n*/ y()\n", 2, 1, 0},
		{"go does not nest", "/* a /* b */ c */\n", 1, 0, 0},
		{"blank line inside block is blank", "/*\n   \n*/\n", 0, 2, 1},
	})
	run_classify_cases(t, sloc.Language_Rust(), []classify_case{
		{"nested on one line", "/* a /* b */ c */\n", 0, 1, 0},
		{"nested across lines then code", "/* a /* b\nstill */ c */ x()\n", 1, 1, 0},
	})
	run_classify_cases(t, sloc.Language_Java_Script(), []classify_case{
		{"javascript block comment", "/* c */\n", 0, 1, 0},
		{"javascript block spans", "/* a\nb\n*/\n", 0, 3, 0},
	})
	run_classify_cases(t, sloc.Language_Odin(), []classify_case{
		{"odin nests", "/* a /* b */ c */\n", 0, 1, 0},
		{"odin nested across lines then code", "/* a /* b\nx */ y */ z()\n", 1, 1, 0},
	})
	run_classify_cases(t, sloc.Language_Lua(), []classify_case{
		{"lua block on one line", "--[[ c ]]\n", 0, 1, 0},
		{"lua block spans", "--[[ a\nb\n]]\n", 0, 3, 0},
		{"lua leveled block ignores short close", "--[==[ a ]] b ]==]\n", 0, 1, 0},
		{"lua code after block", "--[[x]] y()\n", 1, 0, 0},
	})
	run_classify_cases(t, sloc.Language_Swift(), []classify_case{
		{"swift nests", "/* a /* b */ c */\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_Html(), []classify_case{
		{"html comment", "<!-- c -->\n", 0, 1, 0},
		{"html comment spans", "<!-- a\nb\n-->\n", 0, 3, 0},
		{"html code", "<p>hi</p>\n", 1, 0, 0},
	})
	run_classify_cases(t, sloc.Language_Css(), []classify_case{
		{"css block comment", "/* c */\n", 0, 1, 0},
		{"css rule is code", "a { color: red; }\n", 1, 0, 0},
	})
	run_classify_cases(t, sloc.Language_Haskell(), []classify_case{
		{"haskell nests", "{- a {- b -} c -}\n", 0, 1, 0},
		{"haskell line comment", "-- x\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_Ocaml(), []classify_case{
		{"ocaml nests", "(* a (* b *) c *)\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_Julia(), []classify_case{
		{"julia nests", "#= a #= b =# c =#\n", 0, 1, 0},
		{"julia line comment", "# x\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_Nim(), []classify_case{
		{"nim nests", "#[ a #[ b ]# c ]#\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_Common_Lisp(), []classify_case{
		{"lisp block nests", "#| a #| b |# c |#\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_Cmake(), []classify_case{
		{"cmake bracket comment spans", "#[[ a\nb ]]\n", 0, 2, 0},
	})
	run_classify_cases(t, sloc.Language_Markdown(), []classify_case{
		{"markdown html comment", "<!-- c -->\n", 0, 1, 0},
		{"markdown text is code", "# Heading\n", 1, 0, 0},
	})
}

// Test_Classify_Raw_Strings verifies multi-line verbatim strings make their inner
// comment tokens inert across Go, Rust, Python, JavaScript, and TypeScript.
func Test_Classify_Raw_Strings(t *testing.T) {
	run_classify_cases(t, sloc.Language_Go(), []classify_case{
		{"single line backtick with tokens", "s := `/* x */`\n", 1, 0, 0},
		{"multi-line backtick hides close token", "a := `start\n*/ text\nend`\n", 3, 0, 0},
		{"blank line inside backtick is blank", "a := `x\n\ny`\n", 2, 0, 1},
		{"line comment after backtick close", "s := `x` // c\n", 1, 0, 0},
	})
	run_classify_cases(t, sloc.Language_Rust(), []classify_case{
		{"r# raw string", "let s = r#\"a\"#;\n", 1, 0, 0},
		{"hash mismatch does not close", "let s = r##\"has \"# inside\"##;\n", 1, 0, 0},
		{"multi-line raw hides tokens", "let s = r#\"line1\n*/ // not\nend\"#;\n", 3, 0, 0},
		{"raw identifier is not a raw string", "let r#match = 1;\n", 1, 0, 0},
		{"byte raw string", "let b = br#\"x\"#;\n", 1, 0, 0},
	})
	run_classify_cases(t, sloc.Language_Python(), []classify_case{
		{"single-line docstring", "\"\"\"doc\"\"\"\n", 1, 0, 0},
		{"docstring spans as code", "def f():\n \"\"\"\n d # x\n \"\"\"\n p\n", 5, 0, 0},
		{"blank line inside docstring is blank", "\"\"\"\n\ndoc\n\"\"\"\n", 3, 0, 1},
		{"single-quote docstring", "'''\ndoc\n'''\n", 3, 0, 0},
	})
	run_classify_cases(t, sloc.Language_Java_Script(), []classify_case{
		{"template hides tokens", "let s = `start\n// no\nend`;\n", 3, 0, 0},
		{"blank line inside template is blank", "let s = `a\n\nb`;\n", 2, 0, 1},
	})
	run_classify_cases(t, sloc.Language_Type_Script(), []classify_case{
		{"template literal spans", "const s = `x\nnot a comment\nend`;\n", 3, 0, 0},
	})
	run_classify_cases(t, sloc.Language_Odin(), []classify_case{
		{"odin backtick raw string", "s := `/* x */`\n", 1, 0, 0},
		{"odin multi-line backtick", "s := `a\n// b\nc`\n", 3, 0, 0},
	})
	run_classify_cases(t, sloc.Language_Lua(), []classify_case{
		{"lua long string", "local s = [[ a ]]\n", 1, 0, 0},
		{"lua long string hides line comment", "local s = [[a\n-- no\nb]]\n", 3, 0, 0},
		{"lua leveled long string", "local s = [==[ ]] ]==]\n", 1, 0, 0},
	})
	run_classify_cases(t, sloc.Language_Zig(), []classify_case{
		{"zig string with slashes is code", "const u = \"http://x\";\n", 1, 0, 0},
		{"zig multiline string is code", "const s =\n    \\\\ text\n;\n", 3, 0, 0},
	})
	run_classify_cases(t, sloc.Language_Java(), []classify_case{
		{"java text block spans", "var s = \"\"\"\n// not\n\"\"\";\n", 3, 0, 0},
	})
	run_classify_cases(t, sloc.Language_Toml(), []classify_case{
		{"toml triple string spans", "s = \"\"\"\n# not\n\"\"\"\n", 3, 0, 0},
	})
	run_classify_cases(t, sloc.Language_F_Sharp(), []classify_case{
		{"fsharp triple string spans", "let s = \"\"\"\n// no\n\"\"\"\n", 3, 0, 0},
	})
	run_classify_cases(t, sloc.Language_Elixir(), []classify_case{
		{"elixir heredoc spans", "x = \"\"\"\n# no\n\"\"\"\n", 3, 0, 0},
	})
}

// Test_Classify_Heredocs verifies a heredoc body is code with its comment tokens inert
// until the terminator line, while the shift operator opens nothing.
func Test_Classify_Heredocs(t *testing.T) {
	run_classify_cases(t, sloc.Language_Shell(), []classify_case{
		{"heredoc body is code", "cat <<EOF\n# not a comment\nEOF\n", 3, 0, 0},
		{"heredoc with dash", "cat <<-END\nbody\nEND\n", 3, 0, 0},
		{"quoted heredoc", "cat <<'EOF'\n# x\nEOF\n", 3, 0, 0},
		{"shift is not a heredoc", "x=$((a << b))\n# c\n", 1, 1, 0},
	})
	run_classify_cases(t, sloc.Language_Ruby(), []classify_case{
		{"ruby squiggly heredoc", "sql = <<~SQL\n# not a comment\nSQL\n", 3, 0, 0},
	})
}

// Test_Classify_Characters_And_Lifetimes verifies a Rust lifetime tick opens nothing
// while real character literals are code.
func Test_Classify_Characters_And_Lifetimes(t *testing.T) {
	run_classify_cases(t, sloc.Language_Rust(), []classify_case{
		{"lifetime is not a char", "fn f<'a>(x: &'a i32) {}\n", 1, 0, 0},
		{"char literal slash", "let c = '/';\n", 1, 0, 0},
		{"escaped quote char", "let c = '\\'';\n", 1, 0, 0},
		{"char then line comment is code", "let c = 'x'; // c\n", 1, 0, 0},
		{"lifetime then string with token", "let s: &'a str = \"//x\";\n", 1, 0, 0},
	})
}

// Test_Classify_Newlines verifies a trailing newline adds no phantom line, a final
// unterminated line still counts, and the empty file is zero.
func Test_Classify_Newlines(t *testing.T) {
	run_classify_cases(t, sloc.Language_Go(), []classify_case{
		{"empty file", "", 0, 0, 0},
		{"no trailing newline", "foo()", 1, 0, 0},
		{"trailing newline no phantom", "foo()\n", 1, 0, 0},
		{"comment without newline", "// c", 0, 1, 0},
		{"only blank lines", "\n\n\n", 0, 0, 3},
		{"whitespace-only line is blank", "   \t\n", 0, 0, 1},
	})
}

// Test_Languages_Detection verifies each seeded extension resolves to its language and
// an unknown extension resolves to nothing.
func Test_Languages_Detection(t *testing.T) {
	check_detection(t, sloc.Language_For_Extension, detection_extensions_core())
	check_detection(t, sloc.Language_For_Extension, detection_extensions_rest())
	check_detection(t, sloc.Language_For_Filename, detection_filenames())
	if _, recognized := sloc.Language_For_Extension(".txt"); recognized {
		t.Error(".txt should be unrecognized")
	}
}

// Test_Count_Extensions verifies only files with a recognized extension are counted,
// across nested directories, and that unknown extensions are skipped.
func Test_Count_Extensions(t *testing.T) {
	report, err := sloc.Count(sloc.Count_Input{File_System: fstest.MapFS{
		"main.go":    &fstest.MapFile{Data: []byte("package main\n// c\n\n")},
		"sub/lib.rs": &fstest.MapFile{Data: []byte("fn main() {}\n")},
		"readme.txt": &fstest.MapFile{Data: []byte("hello\n")},
		"data.json":  &fstest.MapFile{Data: []byte("{}\n")},
	}})
	if err != nil {
		t.Fatal(err)
	}
	by_path := report_by_path(report)
	if len(by_path) != 2 {
		t.Fatalf("expected 2 counted files, got %v", report.Files)
	}
	if by_path["main.go"].Language != "Go" {
		t.Errorf("main.go language = %q", by_path["main.go"].Language)
	}
	if got := by_path["main.go"].Counts; got != (sloc.Counts{Code: 1, Comment: 1, Blank: 1}) {
		t.Errorf("main.go counts = %+v", got)
	}
	if got := by_path["sub/lib.rs"].Counts; got != (sloc.Counts{Code: 1}) {
		t.Errorf("sub/lib.rs counts = %+v", got)
	}
}

// Test_Count_Hidden verifies dot-prefixed entries are skipped by default and counted
// only with Include_Hidden.
func Test_Count_Hidden(t *testing.T) {
	file_system := fstest.MapFS{
		"main.go":           &fstest.MapFile{Data: []byte("package main\n")},
		".env.go":           &fstest.MapFile{Data: []byte("package secret\n")},
		".hidden/buried.go": &fstest.MapFile{Data: []byte("package buried\n")},
	}
	report, err := sloc.Count(sloc.Count_Input{File_System: file_system})
	if err != nil {
		t.Fatal(err)
	}
	by_path := report_by_path(report)
	if len(by_path) != 1 {
		t.Fatalf("default should count only main.go, got %v", report.Files)
	}
	if by_path["main.go"].Language != "Go" {
		t.Fatalf("expected main.go as Go, got %v", report.Files)
	}
	with_hidden, hidden_err := sloc.Count(sloc.Count_Input{
		File_System:    file_system,
		Include_Hidden: true,
	})
	if hidden_err != nil {
		t.Fatal(hidden_err)
	}
	if len(report_by_path(with_hidden)) != 3 {
		t.Fatalf("Include_Hidden should count all 3, got %v", with_hidden.Files)
	}
}

// Test_Count_Exclusion verifies the injected predicate prunes an ignored directory
// and skips an ignored file.
func Test_Count_Exclusion(t *testing.T) {
	file_system := fstest.MapFS{
		"keep.go":       &fstest.MapFile{Data: []byte("package keep\n")},
		"ignore_me.go":  &fstest.MapFile{Data: []byte("package skip\n")},
		"vendor/dep.go": &fstest.MapFile{Data: []byte("package vendored\n")},
	}
	report, err := sloc.Count(sloc.Count_Input{
		File_System: file_system,
		Is_Ignored: func(relative_path string, is_directory bool) (ignored bool) {
			if relative_path == "vendor" {
				return true
			}
			return relative_path == "ignore_me.go"
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	by_path := report_by_path(report)
	if len(by_path) != 1 {
		t.Fatalf("expected only keep.go, got %v", report.Files)
	}
	if by_path["keep.go"].Path != "keep.go" {
		t.Fatalf("expected keep.go, got %v", report.Files)
	}
}

// Test_Count_Binary verifies a file with a recognized extension but binary content is
// skipped rather than miscounted.
func Test_Count_Binary(t *testing.T) {
	report, err := sloc.Count(sloc.Count_Input{File_System: fstest.MapFS{
		"real.go": &fstest.MapFile{Data: []byte("package main\n")},
		"blob.go": &fstest.MapFile{Data: []byte("package\x00main\n")},
	}})
	if err != nil {
		t.Fatal(err)
	}
	by_path := report_by_path(report)
	if len(by_path) != 1 {
		t.Fatalf("expected only real.go, got %v", report.Files)
	}
	if by_path["real.go"].Path != "real.go" {
		t.Fatalf("expected real.go, got %v", report.Files)
	}
}

// Test_Count_Tests verifies a file is marked a test by its directory or its per-language
// filename convention.
func Test_Count_Tests(t *testing.T) {
	report, err := sloc.Count(sloc.Count_Input{File_System: fstest.MapFS{
		"main.go":        &fstest.MapFile{Data: []byte("package main\n")},
		"main_test.go":   &fstest.MapFile{Data: []byte("package main\n")},
		"tests/integ.rs": &fstest.MapFile{Data: []byte("fn t() {}\n")},
		"__tests__/x.ts": &fstest.MapFile{Data: []byte("test()\n")},
		"app.spec.ts":    &fstest.MapFile{Data: []byte("test()\n")},
	}})
	if err != nil {
		t.Fatal(err)
	}
	by_path := report_by_path(report)
	expected := map[string]bool{
		"main.go":        false,
		"main_test.go":   true,
		"tests/integ.rs": true,
		"__tests__/x.ts": true,
		"app.spec.ts":    true,
	}
	for path, want := range expected {
		if by_path[path].Is_Test != want {
			t.Errorf("%s Is_Test = %v, want %v", path, by_path[path].Is_Test, want)
		}
	}
}

// Test_Render_Table verifies the default table groups languages by category, with a
// grand Total.
func Test_Render_Table(t *testing.T) {
	files := []sloc.File_Count{
		{Path: "a.go", Language: "Go", Counts: sloc.Counts{Code: 10, Comment: 2, Blank: 3}},
		{Path: "b.go", Language: "Go", Counts: sloc.Counts{Code: 5, Comment: 0, Blank: 1}},
		{Path: "c.rs", Language: "Rust",
			Counts: sloc.Counts{Code: 7, Comment: 1, Blank: 0}},
	}
	output := strings.Builder{}
	sloc.Render(&output, sloc.Render_Input{Report: sloc.Report{Files: files}})
	rule := strings.Repeat("─", 54)
	want := strings.Join([]string{
		rule,
		" Language  Files  Lines  Code  Comments  Blanks  %Code",
		rule,
		" Systems",
		"   Rust        1      8     7         1       0",
		" Managed",
		"   Go          2     21    15         2       4",
		rule,
		" Total         3     29    22         3       4",
		rule,
		"",
	}, "\n")
	if output.String() != want {
		t.Errorf("render mismatch:\n got=%q\nwant=%q", output.String(), want)
	}
}

// Test_Render_Files verifies --files breaks each language into its files, indented
// under the language row, with no per-file file count.
func Test_Render_Files(t *testing.T) {
	files := []sloc.File_Count{
		{Path: "a.go", Language: "Go", Counts: sloc.Counts{Code: 3, Comment: 1, Blank: 0}},
		{Path: "sub/b.rs", Language: "Rust",
			Counts: sloc.Counts{Code: 2, Comment: 0, Blank: 1}},
	}
	output := strings.Builder{}
	sloc.Render(&output, sloc.Render_Input{Report: sloc.Report{Files: files}, Show_Files: true})
	rule := strings.Repeat("─", 58)
	want := strings.Join([]string{
		rule,
		" Language      Files  Lines  Code  Comments  Blanks  %Code",
		rule,
		" Systems",
		"   Rust            1      3     2         0       1",
		"     sub/b.rs             3     2         0       1",
		" Managed",
		"   Go              1      4     3         1       0",
		"     a.go                 4     3         1       0",
		rule,
		" Total             2      7     5         1       1",
		rule,
		"",
	}, "\n")
	if output.String() != want {
		t.Errorf("render mismatch:\n got=%q\nwant=%q", output.String(), want)
	}
}

// Test_Render_Thousands verifies counts of a thousand or more are grouped with commas.
func Test_Render_Thousands(t *testing.T) {
	files := []sloc.File_Count{
		{Path: "g.go", Language: "Go",
			Counts: sloc.Counts{Code: 12000, Comment: 3456, Blank: 789}},
		{Path: "g.rs", Language: "Rust", Counts: sloc.Counts{Code: 1000000}},
	}
	output := strings.Builder{}
	sloc.Render(&output, sloc.Render_Input{Report: sloc.Report{Files: files}})
	rule := strings.Repeat("─", 63)
	want := strings.Join([]string{
		rule,
		" Language  Files      Lines       Code  Comments  Blanks  %Code",
		rule,
		" Systems",
		"   Rust        1  1,000,000  1,000,000         0       0",
		" Managed",
		"   Go          1     16,245     12,000     3,456     789",
		rule,
		" Total         2  1,016,245  1,012,000     3,456     789",
		rule,
		"",
	}, "\n")
	if output.String() != want {
		t.Errorf("render mismatch:\n got=%q\nwant=%q", output.String(), want)
	}
}

// Test_Render_Tests verifies a language with test files splits into source and test
// sub-rows, and that the Total splits too.
func Test_Render_Tests(t *testing.T) {
	files := []sloc.File_Count{
		{Path: "a.go", Language: "Go", Counts: sloc.Counts{Code: 10, Comment: 2, Blank: 3}},
		{Path: "a_test.go", Language: "Go", Is_Test: true,
			Counts: sloc.Counts{Code: 4, Comment: 1, Blank: 1}},
		{Path: "b.rs", Language: "Rust",
			Counts: sloc.Counts{Code: 7, Comment: 1, Blank: 0}},
	}
	output := strings.Builder{}
	sloc.Render(&output, sloc.Render_Input{Report: sloc.Report{Files: files}})
	rule := strings.Repeat("─", 56)
	want := strings.Join([]string{
		rule,
		" Language    Files  Lines  Code  Comments  Blanks  %Code",
		rule,
		" Systems",
		"   Rust          1      8     7         1       0",
		" Managed",
		"   Go            2     21    14         3       4",
		"     source      1     15    10         2       3  71.4%",
		"     tests       1      6     4         1       1  28.6%",
		rule,
		" Total           3     29    21         4       4",
		"   source        2     23    17         3       3  81.0%",
		"   tests         1      6     4         1       1  19.0%",
		rule,
		"",
	}, "\n")
	if output.String() != want {
		t.Errorf("render mismatch:\n got=%q\nwant=%q", output.String(), want)
	}
}

// Test_Render_JSON verifies the JSON output: a languages array carrying each language's
// category and source/test split, and a total.
func Test_Render_JSON(t *testing.T) {
	files := []sloc.File_Count{
		{Path: "a.go", Language: "Go", Counts: sloc.Counts{Code: 10, Comment: 2, Blank: 3}},
		{Path: "a_test.go", Language: "Go", Is_Test: true,
			Counts: sloc.Counts{Code: 4, Comment: 1, Blank: 1}},
	}
	output := strings.Builder{}
	if err := sloc.Render_Json(&output, sloc.Report{Files: files}); err != nil {
		t.Fatal(err)
	}
	want := `{
  "languages": [
    {
      "name": "Go",
      "category": "Managed",
      "files": 2,
      "code": 14,
      "comments": 3,
      "blanks": 4,
      "source": {
        "files": 1,
        "code": 10,
        "comments": 2,
        "blanks": 3
      },
      "tests": {
        "files": 1,
        "code": 4,
        "comments": 1,
        "blanks": 1
      }
    }
  ],
  "total": {
    "files": 2,
    "code": 14,
    "comments": 3,
    "blanks": 4,
    "source": {
      "files": 1,
      "code": 10,
      "comments": 2,
      "blanks": 3
    },
    "tests": {
      "files": 1,
      "code": 4,
      "comments": 1,
      "blanks": 1
    }
  }
}
`
	if output.String() != want {
		t.Errorf("json mismatch:\n got=%q\nwant=%q", output.String(), want)
	}
}

// Test_Limitations locks a documented inaccuracy: a JavaScript regex beginning with a
// star opens a false block comment that bleeds into the next line.
func Test_Limitations(t *testing.T) {
	run_classify_cases(t, sloc.Language_Java_Script(), []classify_case{
		{"regex opens false block comment", "x = /*/\ny = 2\n", 1, 1, 0},
	})
}

// A classify_case pins the exact code, comment, and blank tally a source must produce.
type classify_case struct {
	Name    string
	Source  string
	Code    int
	Comment int
	Blank   int
}

// Checks each case's exact line partition.
func run_classify_cases(t *testing.T, language sloc.Language, cases []classify_case) {
	t.Helper()
	for _, one := range cases {
		counts := sloc.Classify_File(sloc.Classify_File_Input{
			Source:   []byte(one.Source),
			Language: language,
		})
		want := sloc.Counts{Code: one.Code, Comment: one.Comment, Blank: one.Blank}
		if counts != want {
			t.Errorf("%s: got %+v, want %+v\nsource: %q",
				one.Name, counts, want, one.Source)
		}
	}
}

// Indexes a report's files by path for order-independent assertions.
func report_by_path(report sloc.Report) (by_path map[string]sloc.File_Count) {
	by_path = map[string]sloc.File_Count{}
	for _, file := range report.Files {
		by_path[file.Path] = file
	}
	return by_path
}

// Checks that each key resolves through lookup to the expected language name.
func check_detection(
	t *testing.T,
	lookup func(string) (language sloc.Language, recognized bool),
	cases map[string]string,
) {
	t.Helper()
	for key, want := range cases {
		language, recognized := lookup(key)
		if !recognized {
			t.Errorf("%s not recognized", key)
			continue
		}
		if language.Name != want {
			t.Errorf("%s -> %q, want %q", key, language.Name, want)
		}
	}
}

// Returns detection expectations for the original and C-style/markup extensions.
func detection_extensions_core() (expected map[string]string) {
	return map[string]string{
		".go": "Go", ".rs": "Rust", ".py": "Python",
		".js": "JavaScript", ".jsx": "JavaScript", ".mjs": "JavaScript",
		".cjs": "JavaScript", ".ts": "TypeScript", ".tsx": "TypeScript",
		".lua": "Lua", ".odin": "Odin", ".zig": "Zig",
		".c": "C", ".h": "C", ".cpp": "C++", ".hpp": "C++",
		".cs": "C#", ".java": "Java", ".swift": "Swift", ".kt": "Kotlin",
		".scala": "Scala", ".sh": "Shell", ".bash": "Shell",
		".rb": "Ruby", ".yaml": "YAML", ".yml": "YAML", ".toml": "TOML",
		".sql": "SQL", ".mk": "Makefile", ".html": "HTML", ".htm": "HTML",
		".xml": "XML", ".svg": "XML", ".css": "CSS", ".scss": "SCSS",
		".less": "LESS", ".m": "Objective-C", ".mm": "Objective-C",
		".dart": "Dart", ".php": "PHP", ".sol": "Solidity",
		".groovy": "Groovy", ".gradle": "Groovy", ".v": "Verilog",
		".sv": "Verilog", ".glsl": "GLSL", ".frag": "GLSL", ".hlsl": "HLSL",
		".ino": "Arduino", ".proto": "Protobuf", ".thrift": "Thrift",
		".jsonc": "JSONC", ".json5": "JSONC", ".tf": "HCL", ".hcl": "HCL",
		".nix": "Nix",
	}
}

// Returns detection expectations for the remaining extensions.
func detection_extensions_rest() (expected map[string]string) {
	return map[string]string{
		".md": "Markdown", ".markdown": "Markdown", ".vue": "Vue",
		".svelte": "Svelte", ".astro": "Astro", ".xaml": "XAML",
		".xsl": "XSLT", ".xslt": "XSLT", ".hs": "Haskell", ".ml": "OCaml",
		".mli": "OCaml", ".fs": "F#", ".jl": "Julia", ".nim": "Nim",
		".lisp": "Common Lisp", ".cl": "Common Lisp", ".scm": "Scheme",
		".rkt": "Racket", ".clj": "Clojure", ".edn": "Clojure",
		".el": "Emacs Lisp", ".erl": "Erlang", ".f90": "Fortran",
		".f": "Fortran", ".adb": "Ada", ".ads": "Ada", ".d": "D",
		".pas": "Pascal", ".pp": "Pascal", ".r": "R", ".ex": "Elixir",
		".exs": "Elixir", ".cr": "Crystal", ".ps1": "PowerShell",
		".cmake": "CMake", ".tcl": "Tcl", ".pl": "Perl", ".pm": "Perl",
		".tex": "TeX", ".vb": "Visual Basic", ".fish": "Fish", ".nu": "Nushell",
	}
}

// Returns detection expectations for special filenames.
func detection_filenames() (expected map[string]string) {
	return map[string]string{
		"Makefile": "Makefile", "Dockerfile": "Dockerfile", "CMakeLists.txt": "CMake",
	}
}
