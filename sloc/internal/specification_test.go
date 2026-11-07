package sloc_test

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	sloc "local/james-orcales/sloc/internal"
)

// Test_Classify_Comments verifies line comments across languages, a trailing comment
// counting as code, documentation comments, and blanks.
func Test_Classify_Comments(t *testing.T) {
	run_classify_cases(t, sloc.Language_For_Test("Go"), []classify_case{
		{"go line comment", "// hi\n", 0, 1, 0},
		{"go code", "package main\n", 1, 0, 0},
		{"go trailing comment is code", "total++ // bump\n", 1, 0, 0},
		{"go doc comment", "/// doc\n", 0, 1, 0},
		{"go code, blank, comment", "a()\n\n// c\n", 1, 1, 1},
	})
	run_classify_cases(t, sloc.Language_For_Test("Python"), []classify_case{
		{"python hash comment", "# hi\n", 0, 1, 0},
		{"python code", "x = 1\n", 1, 0, 0},
		{"python trailing comment is code", "x = 1  # set\n", 1, 0, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("JavaScript"), []classify_case{
		{"javascript line comment", "// hi\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("TypeScript"), []classify_case{
		{"typescript line comment", "// t\n", 0, 1, 0},
		{"typescript type annotation is code", "let x: number = 1;\n", 1, 0, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("Lua"), []classify_case{
		{"lua line comment", "-- hi\n", 0, 1, 0},
		{"lua code", "local x = 1\n", 1, 0, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("Odin"), []classify_case{
		{"odin line comment", "// hi\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("Zig"), []classify_case{
		{"zig line comment", "// hi\n", 0, 1, 0},
		{"zig doc comment", "/// doc\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("C"), []classify_case{
		{"c line comment", "// hi\n", 0, 1, 0},
		{"c block comment", "/* c */\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("Shell"), []classify_case{
		{"shell comment", "# hi\n", 0, 1, 0},
		{"shell hash in string is code", "echo \"a # b\"\n", 1, 0, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("SQL"), []classify_case{
		{"sql line comment", "-- hi\n", 0, 1, 0},
		{"sql block comment", "/* c */\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("PHP"), []classify_case{
		{"php slash comment", "// hi\n", 0, 1, 0},
		{"php hash comment", "# hi\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("Erlang"), []classify_case{
		{"erlang comment", "% hi\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("Common Lisp"), []classify_case{
		{"lisp line comment", "; hi\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("CMake"), []classify_case{
		{"cmake comment", "# hi\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("Fish"), []classify_case{
		{"fish comment", "# hi\n", 0, 1, 0},
		{"fish hash in string is code", "echo \"a # b\"\n", 1, 0, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("Nushell"), []classify_case{
		{"nushell comment", "# hi\n", 0, 1, 0},
	})
}

// Test_Classify_Strings verifies a comment token inside a string is code, never a
// comment — the reason a naive prefix match is insufficient.
func Test_Classify_Strings(t *testing.T) {
	run_classify_cases(t, sloc.Language_For_Test("Go"), []classify_case{
		{"line-comment token in a string", "x := \"http://foo\"\n", 1, 0, 0},
		{"block-comment token in a string", "s := \"/* not a comment */\"\n", 1, 0, 0},
		{"escaped quote in a string", "s := \"she said \\\"hi\\\"\"\n", 1, 0, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("Python"), []classify_case{
		{"hash in a string is code", "s = \"a # b\"\n", 1, 0, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("JavaScript"), []classify_case{
		{"slashes in a string", "let u = \"http://x\";\n", 1, 0, 0},
	})
}

// Test_Classify_Block_Comments verifies Go block comments do not nest while Rust's do,
// and that JavaScript block comments span lines.
func Test_Classify_Block_Comments(t *testing.T) {
	run_classify_cases(t, sloc.Language_For_Test("Go"), []classify_case{
		{"single line block", "/* c */\n", 0, 1, 0},
		{"block spanning lines", "/* a\nb\n*/\n", 0, 3, 0},
		{"code after close", "/* a */ x()\n", 1, 0, 0},
		{"code before open, spanning", "x() /* open\nstill\n*/ y()\n", 2, 1, 0},
		{"go does not nest", "/* a /* b */ c */\n", 1, 0, 0},
		{"blank line inside block is blank", "/*\n   \n*/\n", 0, 2, 1},
	})
	run_classify_cases(t, sloc.Language_For_Test("Rust"), []classify_case{
		{"nested on one line", "/* a /* b */ c */\n", 0, 1, 0},
		{"nested across lines then code", "/* a /* b\nstill */ c */ x()\n", 1, 1, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("JavaScript"), []classify_case{
		{"javascript block comment", "/* c */\n", 0, 1, 0},
		{"javascript block spans", "/* a\nb\n*/\n", 0, 3, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("Odin"), []classify_case{
		{"odin nests", "/* a /* b */ c */\n", 0, 1, 0},
		{"odin nested across lines then code", "/* a /* b\nx */ y */ z()\n", 1, 1, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("Lua"), []classify_case{
		{"lua block on one line", "--[[ c ]]\n", 0, 1, 0},
		{"lua block spans", "--[[ a\nb\n]]\n", 0, 3, 0},
		{"lua leveled block ignores short close", "--[==[ a ]] b ]==]\n", 0, 1, 0},
		{"lua code after block", "--[[x]] y()\n", 1, 0, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("Swift"), []classify_case{
		{"swift nests", "/* a /* b */ c */\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("HTML"), []classify_case{
		{"html comment", "<!-- c -->\n", 0, 1, 0},
		{"html comment spans", "<!-- a\nb\n-->\n", 0, 3, 0},
		{"html code", "<p>hi</p>\n", 1, 0, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("CSS"), []classify_case{
		{"css block comment", "/* c */\n", 0, 1, 0},
		{"css rule is code", "a { color: red; }\n", 1, 0, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("Haskell"), []classify_case{
		{"haskell nests", "{- a {- b -} c -}\n", 0, 1, 0},
		{"haskell line comment", "-- x\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("OCaml"), []classify_case{
		{"ocaml nests", "(* a (* b *) c *)\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("Julia"), []classify_case{
		{"julia nests", "#= a #= b =# c =#\n", 0, 1, 0},
		{"julia line comment", "# x\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("Nim"), []classify_case{
		{"nim nests", "#[ a #[ b ]# c ]#\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("Common Lisp"), []classify_case{
		{"lisp block nests", "#| a #| b |# c |#\n", 0, 1, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("CMake"), []classify_case{
		{"cmake bracket comment spans", "#[[ a\nb ]]\n", 0, 2, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("Markdown"), []classify_case{
		{"markdown html comment", "<!-- c -->\n", 0, 1, 0},
		{"markdown text is code", "# Heading\n", 1, 0, 0},
	})
}

// Test_Classify_Raw_Strings verifies multi-line verbatim strings make their inner
// comment tokens inert across Go, Rust, Python, JavaScript, and TypeScript.
func Test_Classify_Raw_Strings(t *testing.T) {
	run_classify_cases(t, sloc.Language_For_Test("Go"), []classify_case{
		{"single line backtick with tokens", "s := `/* x */`\n", 1, 0, 0},
		{"multi-line backtick hides close token", "a := `start\n*/ text\nend`\n", 3, 0, 0},
		{"blank line inside backtick is blank", "a := `x\n\ny`\n", 2, 0, 1},
		{"line comment after backtick close", "s := `x` // c\n", 1, 0, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("Rust"), []classify_case{
		{"r# raw string", "let s = r#\"a\"#;\n", 1, 0, 0},
		{"hash mismatch does not close", "let s = r##\"has \"# inside\"##;\n", 1, 0, 0},
		{"multi-line raw hides tokens", "let s = r#\"line1\n*/ // not\nend\"#;\n", 3, 0, 0},
		{"raw identifier is not a raw string", "let r#match = 1;\n", 1, 0, 0},
		{"byte raw string", "let b = br#\"x\"#;\n", 1, 0, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("Python"), []classify_case{
		{"single-line docstring", "\"\"\"doc\"\"\"\n", 1, 0, 0},
		{"docstring spans as code", "def f():\n \"\"\"\n d # x\n \"\"\"\n p\n", 5, 0, 0},
		{"blank line inside docstring is blank", "\"\"\"\n\ndoc\n\"\"\"\n", 3, 0, 1},
		{"single-quote docstring", "'''\ndoc\n'''\n", 3, 0, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("JavaScript"), []classify_case{
		{"template hides tokens", "let s = `start\n// no\nend`;\n", 3, 0, 0},
		{"blank line inside template is blank", "let s = `a\n\nb`;\n", 2, 0, 1},
	})
	run_classify_cases(t, sloc.Language_For_Test("TypeScript"), []classify_case{
		{"template literal spans", "const s = `x\nnot a comment\nend`;\n", 3, 0, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("Odin"), []classify_case{
		{"odin backtick raw string", "s := `/* x */`\n", 1, 0, 0},
		{"odin multi-line backtick", "s := `a\n// b\nc`\n", 3, 0, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("Lua"), []classify_case{
		{"lua long string", "local s = [[ a ]]\n", 1, 0, 0},
		{"lua long string hides line comment", "local s = [[a\n-- no\nb]]\n", 3, 0, 0},
		{"lua leveled long string", "local s = [==[ ]] ]==]\n", 1, 0, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("Zig"), []classify_case{
		{"zig string with slashes is code", "const u = \"http://x\";\n", 1, 0, 0},
		{"zig multiline string is code", "const s =\n    \\\\ text\n;\n", 3, 0, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("Java"), []classify_case{
		{"java text block spans", "var s = \"\"\"\n// not\n\"\"\";\n", 3, 0, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("TOML"), []classify_case{
		{"toml triple string spans", "s = \"\"\"\n# not\n\"\"\"\n", 3, 0, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("F#"), []classify_case{
		{"fsharp triple string spans", "let s = \"\"\"\n// no\n\"\"\"\n", 3, 0, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("Elixir"), []classify_case{
		{"elixir heredoc spans", "x = \"\"\"\n# no\n\"\"\"\n", 3, 0, 0},
	})
}

// Test_Classify_Heredocs verifies a heredoc body is code with its comment tokens inert
// until the terminator line, while the shift operator opens nothing.
func Test_Classify_Heredocs(t *testing.T) {
	run_classify_cases(t, sloc.Language_For_Test("Shell"), []classify_case{
		{"heredoc body is code", "cat <<EOF\n# not a comment\nEOF\n", 3, 0, 0},
		{"heredoc with dash", "cat <<-END\nbody\nEND\n", 3, 0, 0},
		{"quoted heredoc", "cat <<'EOF'\n# x\nEOF\n", 3, 0, 0},
		{"shift is not a heredoc", "x=$((a << b))\n# c\n", 1, 1, 0},
	})
	run_classify_cases(t, sloc.Language_For_Test("Ruby"), []classify_case{
		{"ruby squiggly heredoc", "sql = <<~SQL\n# not a comment\nSQL\n", 3, 0, 0},
	})
}

// Test_Classify_Characters_And_Lifetimes verifies a Rust lifetime tick opens nothing
// while real character literals are code.
func Test_Classify_Characters_And_Lifetimes(t *testing.T) {
	run_classify_cases(t, sloc.Language_For_Test("Rust"), []classify_case{
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
	run_classify_cases(t, sloc.Language_For_Test("Go"), []classify_case{
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
	by_extension := func(key string) (language sloc.Language, recognized sloc.Recognition) {
		return sloc.Language_For_Extension(sloc.Extension(key))
	}
	by_filename := func(key string) (language sloc.Language, recognized sloc.Recognition) {
		optional, found := sloc.Language_For_Filename(sloc.File_Name(key))
		return sloc.Language(optional), found
	}
	check_detection(t, by_extension, detection_extensions_core())
	check_detection(t, by_extension, detection_extensions_rest())
	check_detection(t, by_filename, detection_filenames())
	if _, recognized := sloc.Language_For_Extension(sloc.Extension(".txt")); recognized {
		t.Error(".txt should be unrecognized")
	}
}

// Test_Count_Classification keeps classification supplied by the root so a deterministic
// host can model that computation without replacing the filesystem or aggregation contract.
func Test_Count_Classification(t *testing.T) {
	source := []byte("the real scanner would count this as one code line\n")
	want := sloc.File_Partition{Code: 7, Comment: 11, Blank: 13, Dropped: 17}
	report, err := sloc.Count(sloc.Count_Input{
		File_System: fstest.MapFS{
			"modeled.go": &fstest.MapFile{Data: source},
		},
		Classifier: sloc.File_Classifier{
			Kind: sloc.FILE_CLASSIFIER_KIND_MODEL,
			Classifications: sloc.File_Classifications{
				"modeled.go": want,
			},
		},
		Concurrency: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Files) != 1 {
		t.Fatalf("files = %d, want 1", len(report.Files))
	}
	if report.Files[0].Counts != sloc.Counts(want) {
		t.Errorf("counts = %+v, want %+v", report.Files[0].Counts, want)
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
	}, Classifier: sloc.File_Classifier{Kind: sloc.FILE_CLASSIFIER_KIND_BYTES}})
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
	report, err := sloc.Count(sloc.Count_Input{
		File_System: file_system,
		Classifier: sloc.File_Classifier{
			Kind: sloc.FILE_CLASSIFIER_KIND_BYTES,
		},
	})
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
		Classifier: sloc.File_Classifier{
			Kind: sloc.FILE_CLASSIFIER_KIND_BYTES,
		},
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
		Classifier: sloc.File_Classifier{
			Kind: sloc.FILE_CLASSIFIER_KIND_BYTES,
		},
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
	}, Classifier: sloc.File_Classifier{Kind: sloc.FILE_CLASSIFIER_KIND_BYTES}})
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
	}, Classifier: sloc.File_Classifier{Kind: sloc.FILE_CLASSIFIER_KIND_BYTES}})
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
		if bool(by_path[path].Is_Test) != want {
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
			Counts: sloc.Counts{Code: 8000, Comment: 3456, Blank: 789}},
		{Path: "g.rs", Language: "Rust", Counts: sloc.Counts{Code: 1000}},
	}
	output := strings.Builder{}
	sloc.Render(&output, sloc.Render_Input{Report: sloc.Report{Files: files}})
	rule := strings.Repeat("─", 56)
	want := strings.Join([]string{
		rule,
		" Language  Files   Lines   Code  Comments  Blanks  %Code",
		rule,
		" Systems",
		"   Rust        1   1,000  1,000         0       0",
		" Managed",
		"   Go          1  12,245  8,000     3,456     789",
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
		"",
	}, "\n")
	if output.String() != want {
		t.Errorf("render mismatch:\n got=%q\nwant=%q", output.String(), want)
	}
}

// Test_Render_No_Total verifies a report states each language and stops there. Nothing
// sums the run, in the table or in the JSON.
func Test_Render_No_Total(t *testing.T) {
	files := []sloc.File_Count{
		{Path: "a.go", Language: "Go", Counts: sloc.Counts{Code: 10, Comment: 2, Blank: 3}},
		{Path: "b.rs", Language: "Rust",
			Counts: sloc.Counts{Code: 7, Comment: 1, Blank: 0}},
	}
	table := strings.Builder{}
	sloc.Render(&table, sloc.Render_Input{Report: sloc.Report{Files: files}})
	if strings.Contains(table.String(), "Total") {
		t.Errorf("table sums the run:\n%s", table.String())
	}
	document := strings.Builder{}
	if err := sloc.Render_Json(
		&document, sloc.File_Counts(files),
	); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(document.String(), "total") {
		t.Errorf("json sums the run:\n%s", document.String())
	}
}

// Test_Render_JSON verifies the JSON output: a compact top-level array of flat language rows,
// each carrying its category and source/test split.
func Test_Render_JSON(t *testing.T) {
	files := []sloc.File_Count{
		{Path: "a.go", Language: "Go", Counts: sloc.Counts{Code: 10, Comment: 2, Blank: 3}},
		{Path: "a_test.go", Language: "Go", Is_Test: true,
			Counts: sloc.Counts{Code: 4, Comment: 1, Blank: 1}},
	}
	output := strings.Builder{}
	if err := sloc.Render_Json(
		&output, sloc.File_Counts(files),
	); err != nil {
		t.Fatal(err)
	}
	want := `[{"name":"Go","category":"Managed","source_files":1,"source_code":10,"source_comments":2,"source_blanks":3,"tests_files":1,"tests_code":4,"tests_comments":1,"tests_blanks":1}]`
	if output.String() != want {
		t.Errorf("json mismatch:\n got=%q\nwant=%q", output.String(), want)
	}
}

// Test_Render_Dropped verifies the table's last section reports how many lines were
// wider than the scan window and so were classified from a prefix.
func Test_Render_Dropped(t *testing.T) {
	wide := strings.Repeat("a", sloc.LINE_BYTES_MAX+1) + "\n"
	counts := sloc.Classify_File(sloc.Classify_File_Input{
		Path:     "fixture.go",
		Source:   sloc.Source(wide + "b\n"),
		Language: sloc.Language_For_Test("Go"),
	})
	if counts.Dropped != 1 {
		t.Errorf("dropped = %d, want 1", counts.Dropped)
	}
	if counts.Code != 2 {
		t.Errorf("code = %d, want 2: a wide line still counts", counts.Code)
	}
	files := []sloc.File_Count{{
		Path: "w.go", Language: "Go", Counts: sloc.Counts(counts),
	}}
	output := strings.Builder{}
	sloc.Render(&output, sloc.Render_Input{Report: sloc.Report{Files: files}})
	if !strings.Contains(output.String(), " Dropped") {
		t.Errorf("table has no Dropped section:\n%s", output.String())
	}
	// At the bound the tally stops counting and the cell says so.
	saturated := []sloc.File_Count{{Path: "w.go", Language: "Go", Counts: sloc.Counts{
		Code: 1, Comment: 0, Blank: 0, Dropped: sloc.DROPPED_COUNT_MAX,
	}}}
	output = strings.Builder{}
	sloc.Render(&output, sloc.Render_Input{Report: sloc.Report{Files: saturated}})
	if !strings.Contains(output.String(), "100k+") {
		t.Errorf("saturated tally is not marked:\n%s", output.String())
	}
	// A file left out of the totals is named, and only what happened is listed.
	report, err := sloc.Count(sloc.Count_Input{
		File_System: fstest.MapFS{
			"blob.go": &fstest.MapFile{Data: []byte("package\x00main\n")},
			"good.go": &fstest.MapFile{Data: []byte("a\n")},
		},
		Is_Ignored:     nil,
		Include_Hidden: false,
		Classifier: sloc.File_Classifier{
			Kind: sloc.FILE_CLASSIFIER_KIND_BYTES,
		},
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	output = strings.Builder{}
	sloc.Render(&output, sloc.Render_Input{Report: sloc.Report(report)})
	if !strings.Contains(output.String(), "files binary") {
		t.Errorf("a dropped binary file is not reported:\n%s", output.String())
	}
	if strings.Contains(output.String(), "files oversized") {
		t.Errorf("a tally of zero should print nothing:\n%s", output.String())
	}
}

// Test_Limitations locks a documented inaccuracy: a JavaScript regex beginning with a
// star opens a false block comment that bleeds into the next line.
// Test_Bounds_Line verifies a line wider than the scan window still counts as exactly
// one line: the totals stay exact for any input, only the tail goes unclassified.
func Test_Bounds_Line(t *testing.T) {
	wide := strings.Repeat("a", sloc.LINE_BYTES_MAX+904) + "\n"
	counts := sloc.Classify_File(sloc.Classify_File_Input{
		Path:     "fixture.go",
		Source:   sloc.Source(wide),
		Language: sloc.Language_For_Test("Go"),
	})
	want := sloc.File_Partition{Code: 1, Comment: 0, Blank: 0, Dropped: 1}
	if counts != want {
		t.Errorf("wide line: got %+v, want %+v", counts, want)
	}
	// The window truncates what is read, so a comment opening past it is not seen.
	late := strings.Repeat(" ", sloc.LINE_BYTES_MAX) + "// c\n"
	counts = sloc.Classify_File(sloc.Classify_File_Input{
		Path:     "fixture.go",
		Source:   sloc.Source(late),
		Language: sloc.Language_For_Test("Go"),
	})
	want = sloc.File_Partition{Code: 0, Comment: 0, Blank: 1, Dropped: 1}
	if counts != want {
		t.Errorf("comment past the window: got %+v, want %+v", counts, want)
	}
}

// Test_Bounds_File verifies a file past the source bound is left out of the report the
// way a binary one is, and that the rest of the tree is still counted.
func Test_Bounds_File(t *testing.T) {
	over := strings.Repeat("a\n", sloc.SOURCE_BYTES_MAX)
	report, err := sloc.Count(sloc.Count_Input{
		File_System: fstest.MapFS{
			"big.go":  &fstest.MapFile{Data: []byte(over)},
			"fine.go": &fstest.MapFile{Data: []byte("a\n")},
		},
		Is_Ignored:     nil,
		Include_Hidden: false,
		Classifier: sloc.File_Classifier{
			Kind: sloc.FILE_CLASSIFIER_KIND_BYTES,
		},
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if len(report.Files) != 1 {
		t.Fatalf("got %d files, want only the one within the bound", len(report.Files))
	}
	if report.Files[0].Path != "fine.go" {
		t.Errorf("counted %q, want fine.go", report.Files[0].Path)
	}
}

// Test_Bounds_Overflow verifies a tree holding more recognized files than the file bound
// is counted up to that many, and that the walk carries on past it so the report can say
// how many it could not take.
func Test_Bounds_Overflow(t *testing.T) {
	disk := fstest.MapFS{}
	const OVER = 32
	for index := range sloc.FILES_COUNT_MAX + OVER {
		disk[fmt.Sprintf("f%06d.go", index)] = &fstest.MapFile{Data: []byte("a\n")}
	}
	report, err := sloc.Count(sloc.Count_Input{
		File_System:    disk,
		Is_Ignored:     nil,
		Include_Hidden: false,
		Classifier: sloc.File_Classifier{
			Kind: sloc.FILE_CLASSIFIER_KIND_BYTES,
		},
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if len(report.Files) != sloc.FILES_COUNT_MAX {
		t.Errorf("counted %d files, want the bound %d",
			len(report.Files), sloc.FILES_COUNT_MAX)
	}
	// The point of walking past the bound is that the excess is a known number rather
	// than an unstated "there were more".
	if report.Skipped.Overflow != OVER {
		t.Errorf("overflow = %d, want %d", report.Skipped.Overflow, OVER)
	}
}

// Test_Bounds_Depth verifies the block-comment depth saturates at its bound rather
// than growing with the file, so a comment nested past it closes early.
func Test_Bounds_Depth(t *testing.T) {
	deep := strings.Repeat("/*", 300) + "\nstill inside\n"
	counts := sloc.Classify_File(sloc.Classify_File_Input{
		Path:     "fixture.rs",
		Source:   sloc.Source(deep),
		Language: sloc.Language_For_Test("Rust"),
	})
	want := sloc.File_Partition{Code: 0, Comment: 2, Blank: 0, Dropped: 1}
	if counts != want {
		t.Errorf("deep nesting: got %+v, want %+v", counts, want)
	}
}

// Test_Host_File_Read verifies that Main applies the source bound to an explicitly
// named file before the classifier receives it.
func Test_Host_File_Read(t *testing.T) {
	source := bytes.Repeat([]byte{'a'}, sloc.SOURCE_BYTES_MAX+1)
	tracked := &counted_file{File: test_file(source)}
	input := classifier_main_input(sloc.File_Classifier{
		Kind: sloc.FILE_CLASSIFIER_KIND_MODEL,
		Classifications: sloc.File_Classifications{
			"modeled.go": {Code: 1, Dropped: 1},
		},
	}, nil)
	input.File = func(name sloc.File_Path) (file fs.File, err error) {
		return tracked, nil
	}
	if code := sloc.Main(input); code != sloc.EXIT_SUCCESS {
		t.Fatalf("exit = %d, want %d", code, sloc.EXIT_SUCCESS)
	}
	if tracked.Read_Count != sloc.SOURCE_BYTES_MAX {
		t.Errorf("read bytes = %d, want %d", tracked.Read_Count, sloc.SOURCE_BYTES_MAX)
	}
}

// Test_Host_Git_Ignore verifies that Main constructs the scoped Git command and turns
// its kept-file output into the walk predicate.
func Test_Host_Git_Ignore(t *testing.T) {
	disk := fstest.MapFS{
		"good.go":    &fstest.MapFile{Data: []byte("a\n")},
		"ignored.go": &fstest.MapFile{Data: []byte("b\n")},
	}
	output := strings.Builder{}
	command_name := ""
	command_arguments := []string{}
	code := sloc.Main(sloc.Main_Input{
		Arguments:    sloc.Arguments{"sloc", "-files", "."},
		Output:       &output,
		Error_Output: io.Discard,
		File_System:  func(root sloc.Root) (file_system fs.FS) { return disk },
		Path_Information: func(
			name sloc.File_Path,
		) (information fs.FileInfo, err error) {
			return test_information(true), nil
		},
		File: func(name sloc.File_Path) (file fs.File, err error) {
			return test_file(nil), nil
		},
		Command: func(
			name string, arguments []string,
		) (command_output []byte, err error) {
			command_name = name
			command_arguments = append(command_arguments, arguments...)
			return []byte("good.go\x00"), nil
		},
		Classifier:  sloc.File_Classifier{Kind: sloc.FILE_CLASSIFIER_KIND_BYTES},
		Concurrency: 1,
	})
	if code != sloc.EXIT_SUCCESS {
		t.Fatalf("exit = %d, want %d", code, sloc.EXIT_SUCCESS)
	}
	if command_name != "git" {
		t.Errorf("command = %q, want git", command_name)
	}
	want_arguments := "-C . ls-files -z --cached --others --exclude-standard -- ."
	if strings.Join(command_arguments, " ") != want_arguments {
		t.Errorf("arguments = %q, want %q",
			strings.Join(command_arguments, " "), want_arguments)
	}
	if !strings.Contains(output.String(), "good.go") {
		t.Errorf("kept file is absent:\n%s", output.String())
	}
	if strings.Contains(output.String(), "ignored.go") {
		t.Errorf("ignored file is present:\n%s", output.String())
	}
}

// Test_Host_Workers locks the internal worker policy that the executable root uses.
func Test_Host_Workers(t *testing.T) {
	if sloc.WORKERS_PER_PROCESSOR != 4 {
		t.Errorf("workers per processor = %d, want 4", sloc.WORKERS_PER_PROCESSOR)
	}
}

// Test_Limitations verifies the documented misclassifications.
func Test_Limitations(t *testing.T) {
	run_classify_cases(t, sloc.Language_For_Test("JavaScript"), []classify_case{
		{"regex opens false block comment", "x = /*/\ny = 2\n", 1, 1, 0},
	})
}

// Counted_file records bytes because the file-size cap must remain inside Main.
type counted_file struct {
	fs.File
	Read_Count int
}

// Read records only bytes that the bounded reader requested from the file.
func (file *counted_file) Read(buffer []byte) (read_count int, err error) {
	read_count, err = file.File.Read(buffer)
	file.Read_Count += read_count
	return read_count, err
}

// A classify_case pins the exact code, comment, and blank tally a source must produce.
type classify_case struct {
	Name    string
	Source  string
	Code    sloc.Code_Count
	Comment sloc.Comment_Count
	Blank   sloc.Blank_Count
}

// Checks each case's exact line partition.
func run_classify_cases(
	t *testing.T, language sloc.Seeded_Language, cases []classify_case,
) {
	t.Helper()
	for _, one := range cases {
		counts := sloc.Classify_File(sloc.Classify_File_Input{
			Path:     "fixture",
			Source:   sloc.Source(one.Source),
			Language: language,
		})
		want := sloc.File_Partition{
			Code: one.Code, Comment: one.Comment, Blank: one.Blank,
		}
		if counts != want {
			t.Errorf("%s: got %+v, want %+v\nsource: %q",
				one.Name, counts, want, one.Source)
		}
	}
}

// Indexes a report's files by path for order-independent assertions.
func report_by_path(report sloc.Root_Report) (by_path map[string]sloc.File_Count) {
	by_path = map[string]sloc.File_Count{}
	for _, file := range report.Files {
		by_path[string(file.Path)] = file
	}
	return by_path
}

// Checks that each key resolves through lookup to the expected language name.
func check_detection(
	t *testing.T,
	lookup func(string) (language sloc.Language, recognized sloc.Recognition),
	cases map[string]string,
) {
	t.Helper()
	for key, want := range cases {
		language, recognized := lookup(key)
		if !recognized {
			t.Errorf("%s not recognized", key)
			continue
		}
		if string(language.Name) != want {
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
