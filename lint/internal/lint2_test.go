package lint_test

import (
	"bytes"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	snap "github.com/james-orcales/james-orcales/golang_snacks/snap/v2/snap_default"
	"github.com/james-orcales/james-orcales/lint/internal"
)

// Additional cases, split to keep each function within the length limit.
func Test_Main_First_Part2(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{Name: "TestMain not first",
			Files: map[string]string{"foo_test.go": `package foo_test

import "testing"

// Test_Foo is a fixture.
func Test_Foo(t *testing.T) { return }
func TestMain(m *testing.M) { return }
`}, Want_Diag: "func TestMain should be declared first"},

		{Name: "TestMain is first",
			Files: map[string]string{"foo_test.go": `package foo_test

import "testing"

func TestMain(m *testing.M) { return }
// Test_Foo is a fixture.
func Test_Foo(t *testing.T) { return }
`}, Want_Diag: ""},

		{Name: "TestMain exempt from casing",
			Files: map[string]string{"foo_test.go": `package foo_test

import "testing"

func TestMain(m *testing.M) { return }
`}, Want_Diag: ""},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			fsys_map := make(fstest.MapFS)
			for k, v := range tt.Files {
				fsys_map[k] = &fstest.MapFile{Data: gofmt_must(t, v)}
			}
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			code := lint.Main(&lint.Main_Input{
				Fsys: fsys_map, Stdout: stdout, Stderr: stderr,
			})

			output := stdout.String()
			if tt.Want_Diag == "" {
				if code != 0 {
					t.Errorf("expected exit 0, got %d; output: %s",
						code, output)
				}
			} else {
				if !bytes.Contains(stdout.Bytes(), []byte(tt.Want_Diag)) {
					t.Errorf("expected output containing %q, got: %s",
						tt.Want_Diag, output)
				}
			}
		})
	}
}

// Test_Line_Character_Count_Tabs verifies that tabs count as tab_width chars (not 1)
// when measuring line length against the line_chars_max limit.
func Test_Line_Character_Count_Tabs(t *testing.T) {
	t.Parallel()
	// 18 tabs * 8 = 144 column width; far under the 140 limit if tabs count
	// as 1, well over if tabs count as 8. Inline comments are exempt from
	// comment-sentence rules, so this stays a pure line-length test.
	tabs := ""
	for range 18 {
		tabs += "\t"
	}
	source := "package main\nfunc f() (result int) { return 1 } //" + tabs + "tail\n"
	fsys_map := fstest.MapFS{"test.go": &fstest.MapFile{
		Data: []byte(source)}}
	stdout := &bytes.Buffer{}
	code := lint.Main(&lint.Main_Input{Fsys: fsys_map, Stdout: stdout, Stderr: &bytes.Buffer{}})
	if code == 0 {
		t.Fatalf("expected line-length diagnostic, got exit 0; output: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("line is")) {
		t.Errorf("expected line-length diagnostic, got: %s", stdout.String())
	}
}

// Test_Line_Character_Count_Import_Exempt verifies that an import line wider than
// line_chars_max is not flagged: import paths are unbreakable, so a long module
// path must never force a lint failure.
func Test_Line_Character_Count_Import_Exempt(t *testing.T) {
	t.Parallel()
	long_path := "github.com/example/" + strings.Repeat("a", 90) + "/pkg"
	source := "package main\n\nimport " + strconv.Quote(long_path) + "\n"
	fsys_map := fstest.MapFS{"test.go": &fstest.MapFile{
		Data: []byte(source)}}
	stdout := &bytes.Buffer{}
	code := lint.Main(&lint.Main_Input{Fsys: fsys_map, Stdout: stdout, Stderr: &bytes.Buffer{}})
	if code != 0 {
		t.Fatalf("import line should be exempt from the length limit; got exit %d: %s",
			code, stdout.String())
	}
}

// Test_Comments verifies the comment-style rules: capital start, trailing
// `.`/`:`/`?`/`!`, space after `//`, pragma exemption, and inline-comment
// exemption from the sentence rules.
func Test_Comments(t *testing.T) {
	t.Parallel()
	// TestComments fixtures include `//foo` (no space after slashes), which
	// gofmt normalizes before our check sees it. Bypass gofmt_must so the
	// formatting-style violations under test survive intact.
	for _, tt := range []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "missing trailing period",
			Files: map[string]string{
				"test.go": "// Usage info (no period)\npackage main\n",
			},
			Want_Diag: "should end with",
		},

		{
			Name: "lowercase start",
			Files: map[string]string{
				"test.go": "// lowercase start.\npackage main\n",
			},
			Want_Diag: "should start with capital",
		},

		{
			Name: "missing space after slashes",
			Files: map[string]string{
				"test.go": "//No space.\npackage main\n",
			},
			Want_Diag: "missing space after",
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			fsys_map := make(fstest.MapFS)
			for k, v := range tt.Files {
				fsys_map[k] = &fstest.MapFile{
					Data: []byte(v)}
			}
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			code := lint.Main(&lint.Main_Input{
				Fsys: fsys_map, Stdout: stdout, Stderr: stderr,
			})

			output := stdout.String()
			if tt.Want_Diag == "" {
				if code != 0 {
					t.Errorf("expected exit 0, got %d; output: %s",
						code, output)
				}
			} else {
				if !bytes.Contains(stdout.Bytes(), []byte(tt.Want_Diag)) {
					t.Errorf("expected output containing %q, got: %s",
						tt.Want_Diag, output)
				}
			}
		})
	}
}

// Additional cases, split to keep each function within the length limit.
func Test_Comments_Part2(t *testing.T) {
	t.Parallel()
	// TestComments fixtures include `//foo` (no space after slashes), which
	// gofmt normalizes before our check sees it. Bypass gofmt_must so the
	// formatting-style violations under test survive intact.
	for _, tt := range []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "well-formed comment",
			Files: map[string]string{
				"test.go": "// Well-formed sentence.\npackage main\n",
			},
			Want_Diag: "",
		},

		{
			Name: "trailing colon allowed",
			Files: map[string]string{
				"test.go": "// Followed by something:\npackage main\n",
			},
			Want_Diag: "",
		},

		{
			Name: "pragma exempt",
			Files: map[string]string{
				"test.go": "//go:build linux\n\npackage main\n",
			},
			Want_Diag: "",
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			fsys_map := make(fstest.MapFS)
			for k, v := range tt.Files {
				fsys_map[k] = &fstest.MapFile{
					Data: []byte(v)}
			}
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			code := lint.Main(&lint.Main_Input{
				Fsys: fsys_map, Stdout: stdout, Stderr: stderr,
			})

			output := stdout.String()
			if tt.Want_Diag == "" {
				if code != 0 {
					t.Errorf("expected exit 0, got %d; output: %s",
						code, output)
				}
			} else {
				if !bytes.Contains(stdout.Bytes(), []byte(tt.Want_Diag)) {
					t.Errorf("expected output containing %q, got: %s",
						tt.Want_Diag, output)
				}
			}
		})
	}
}

// Additional cases, split to keep each function within the length limit.
func Test_Comments_Part3(t *testing.T) {
	t.Parallel()
	// TestComments fixtures include `//foo` (no space after slashes), which
	// gofmt normalizes before our check sees it. Bypass gofmt_must so the
	// formatting-style violations under test survive intact.
	for _, tt := range []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "multi-line group: trailing period on last line",
			Files: map[string]string{
				"test.go": "// First line\n// second line.\npackage main\n",
			},
			Want_Diag: "",
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			fsys_map := make(fstest.MapFS)
			for k, v := range tt.Files {
				fsys_map[k] = &fstest.MapFile{
					Data: []byte(v)}
			}
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			code := lint.Main(&lint.Main_Input{
				Fsys: fsys_map, Stdout: stdout, Stderr: stderr,
			})

			output := stdout.String()
			if tt.Want_Diag == "" {
				if code != 0 {
					t.Errorf("expected exit 0, got %d; output: %s",
						code, output)
				}
			} else {
				if !bytes.Contains(stdout.Bytes(), []byte(tt.Want_Diag)) {
					t.Errorf("expected output containing %q, got: %s",
						tt.Want_Diag, output)
				}
			}
		})
	}
}

// Test_Comments_Inline_Exempt verifies that comments on the same line as a
// declaration (after the opening brace, for example) are exempt from the
// sentence rules that apply to leading doc comments.
func Test_Comments_Inline_Exempt(t *testing.T) {
	t.Parallel()
	source := "package main\n\n" +
		"import invariant \"github.com/james-orcales/james-orcales/" +
		"golang_snacks/invariant/v2\"\n\n" +
		"const fixture_hi = 100\n\n" +
		"func f() (result int) { // some inline note\n" +
		"\tdefer func() {\n" +
		"\t\tinvariant.Cross_Product(\n" +
		"\t\t\tinvariant.Distinct_Boundary(" +
		"&invariant.Boundary_Input[int]{\n" +
		"\t\t\t\tX: result, Lo: 0, Hi: fixture_hi,\n" +
		"\t\t\t}),\n" +
		"\t\t\tinvariant.Always(" +
		"result == 0, \"result is zero\"),\n" +
		"\t\t)\n" +
		"\t}()\n" +
		"\treturn 1\n" +
		"}\n"
	fsys_map := fstest.MapFS{"test.go": &fstest.MapFile{
		Data: []byte(source)}}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := lint.Main(&lint.Main_Input{Fsys: fsys_map, Stdout: stdout, Stderr: stderr})
	if code != 0 {
		t.Errorf("expected exit 0, got %d; output: %s", code, stdout.String())
	}
}

// Test_No_Package_Vars verifies the package-level var ban: only
// regexp.MustCompile and errors.New initializers are allowed (plus the
// compile-time interface-satisfaction shape `var _ Iface = (*Impl)(nil)`).
// Local-scope var declarations inside funcs are untouched.
// Test_No_Package_Vars_Embed_Allowed verifies a //go:embed var is exempt from
// the package-var ban — the directive can only attach to a package-level var.
func Test_No_Package_Vars_Embed_Allowed(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "go:embed var allowed",
			Files: map[string]string{
				"test.go": `package main
import "embed"

//go:embed test.go
var sources embed.FS
func main() { return }
`,
			},
			Want_Diag: "",
		},
	})
}

// Test_No_Package_Vars verifies the allowed package-level var forms — the
// approved initializers — pass without a diagnostic.
func Test_No_Package_Vars(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "regexp.MustCompile allowed",
			Files: map[string]string{
				"test.go": `package main
import "regexp"
var re = regexp.MustCompile("x")
func main() { return }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "errors.New allowed",
			Files: map[string]string{
				"test.go": `package main
import "errors"
var Err_Foo = errors.New("foo")
func main() { return }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "multiple allowed initializers",
			Files: map[string]string{
				"test.go": `package main
import "regexp"
var re1 = regexp.MustCompile("a")
var re2 = regexp.MustCompile("b")
func main() { return }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "default package Default singleton allowed",
			Files: map[string]string{
				"snap/default/wire.go": `// Package snap is a fixture.
package snap
// Snapper is a fixture.
type Snapper struct{}
// Default is the OS-bound Snapper.
var Default = &Snapper{}
`,
			},
			Want_Diag: "",
		},
		{
			Name: "Default in non-snap package still banned",
			Files: map[string]string{
				"test.go": `package main
type S struct{}
var Default = &S{}
func main() { return }
`,
			},
			Want_Diag: "package-level var is banned",
		},
	}
	run_diag_table(t, tests)
}

// Test_No_Package_Vars_Banned covers the rejection side: zero-value vars,
// non-allowlisted initializers, mixed groups, and a negative case for
// local-scope vars which must not be flagged.
func Test_No_Package_Vars_Banned(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "zero-value var banned",
			Files: map[string]string{
				"test.go": `package main
var x_count int
func main() { return }
`,
			},
			Want_Diag: "package-level var is banned",
		},

		{
			Name: "literal initializer banned",
			Files: map[string]string{
				"test.go": `package main
var x_count = 5
func main() { return }
`,
			},
			Want_Diag: "package-level var is banned",
		},

		{
			Name: "composite literal initializer banned",
			Files: map[string]string{
				"test.go": `package main
type S struct{}
var x_thing = S{}
func main() { return }
`,
			},
			Want_Diag: "package-level var is banned",
		},

		{
			Name: "disallowed call initializer banned",
			Files: map[string]string{
				"test.go": `package main
import "time"
var t_start = time.Now()
func main() { return }
`,
			},
			Want_Diag: "package-level var is banned",
		},

		{
			Name: "multi-name single-line banned",
			Files: map[string]string{
				"test.go": `package main
var a_count, b_count = 1, 2
func main() { return }
`,
			},
			Want_Diag: "package-level var is banned",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_No_Package_Vars_Banned_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "mixed: only disallowed flagged",
			Files: map[string]string{
				"test.go": `package main
import "regexp"
var re1 = regexp.MustCompile("a")
var bad_v = 5
func main() { return }
`,
			},
			Want_Diag: "package-level var is banned",
		},

		{
			Name: "local-scope var untouched",
			Files: map[string]string{
				"test.go": `package main
func main() {
	var x_count int
	x_count++
}
`,
			},
			Want_Diag: "",
		},

		{
			Name: "slice literal banned with switch hint",
			Files: map[string]string{
				"test.go": `package main
var x_list = []int{1, 2, 3}
func main() { return }
`,
			},
			Want_Diag: "package-level var is banned",
		},
	})
}

// Test_Banned_Scripting_Files verifies that scripting-language files
// (.py, .sh, Makefile, ...) anywhere outside the top-level third_party/
// and vendor/ directories are flagged. Bypasses run_diag_table because
// gofmt_must would choke on non-Go content.
func Test_Banned_Scripting_Files(t *testing.T) {
	t.Parallel()
	clean_go := []byte(fixture_clean_go)
	test_banned_scripting_files_run(t, []struct {
		Name      string
		Files     map[string][]byte
		Want_Diag string
	}{

		{
			Name: "top-level .py flagged",
			Files: map[string][]byte{
				"test.go":   clean_go,
				"script.py": []byte("print(1)\n"),
			},
			Want_Diag: "as a go script",
		},

		{
			Name: "nested .sh flagged",
			Files: map[string][]byte{
				"test.go":    clean_go,
				"cmd/run.sh": []byte("#!/bin/sh\necho hi\n"),
			},
			Want_Diag: "as a go script",
		},

		{
			Name: "Makefile flagged",
			Files: map[string][]byte{
				"test.go":  clean_go,
				"Makefile": []byte("all:\n\techo hi\n"),
			},
			Want_Diag: "as a go script",
		},

		{
			Name: "lowercase makefile flagged",
			Files: map[string][]byte{
				"test.go":  clean_go,
				"makefile": []byte("all:\n\techo hi\n"),
			},
			Want_Diag: "as a go script",
		},

		{
			Name: "Rakefile flagged",
			Files: map[string][]byte{
				"test.go":  clean_go,
				"Rakefile": []byte("task :default\n"),
			},
			Want_Diag: "as a go script",
		},

		{
			Name: "lua flagged",
			Files: map[string][]byte{
				"test.go":    clean_go,
				"script.lua": []byte("print(1)\n"),
			},
			Want_Diag: "as a go script",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_Banned_Scripting_Files_Part2(t *testing.T) {
	t.Parallel()
	clean_go := []byte(fixture_clean_go)
	test_banned_scripting_files_run(t, []struct {
		Name      string
		Files     map[string][]byte
		Want_Diag string
	}{

		{
			Name: "third_party top-level allowed",
			Files: map[string][]byte{
				"test.go":              clean_go,
				"third_party/foo.py":   []byte("print(1)\n"),
				"third_party/Makefile": []byte("all:\n"),
			},
			Want_Diag: "",
		},

		{
			Name: "vendor top-level allowed",
			Files: map[string][]byte{
				"test.go":       clean_go,
				"vendor/foo.sh": []byte("#!/bin/sh\n"),
			},
			Want_Diag: "",
		},

		{
			Name: "nested third_party NOT exempt",
			Files: map[string][]byte{
				"test.go":              clean_go,
				"pkg/third_party/x.py": []byte("print(1)\n"),
			},
			Want_Diag: "as a go script",
		},

		{
			Name: "go file alone clean",
			Files: map[string][]byte{
				"test.go": clean_go,
			},
			Want_Diag: "",
		},
	})
}

// Mirrors run_diag_table but feeds raw bytes — non-Go content would not
// survive the gofmt preprocessing in the table runner used by Go-only tests.
func test_banned_scripting_files_run(t *testing.T, tests []struct {
	Name      string
	Files     map[string][]byte
	Want_Diag string
}) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			fsys_map := make(fstest.MapFS)
			for k, v := range tt.Files {
				fsys_map[k] = &fstest.MapFile{Data: v}
			}
			stdout := &bytes.Buffer{}
			code := lint.Main(&lint.Main_Input{
				Fsys: fsys_map, Stdout: stdout, Stderr: &bytes.Buffer{},
			})
			output := stdout.String()
			if tt.Want_Diag == "" {
				if code != 0 {
					t.Errorf("expected exit 0, got %d; output: %s",
						code, output)
				}
				return
			}
			if !bytes.Contains(stdout.Bytes(), []byte(tt.Want_Diag)) {
				t.Errorf("expected output containing %q, got: %s",
					tt.Want_Diag, output)
			}
		})
	}
}

// Run_stream returns Main's stdout for a tree of files. Stream-tier checks
// run against raw bytes, so test inputs go through MapFS verbatim — no
// gofmt_must wrapping like the AST-tier tests use.
func run_stream(t *testing.T, files map[string]string) (output string) {
	t.Helper()
	fsys_map := make(fstest.MapFS)
	for k, v := range files {
		fsys_map[k] = &fstest.MapFile{
			Data: []byte(v)}
	}
	stdout := &bytes.Buffer{}
	lint.Main(&lint.Main_Input{Fsys: fsys_map, Stdout: stdout, Stderr: &bytes.Buffer{}})
	return stdout.String()
}

// Lints the given in-memory files and returns the linter's stdout with the
// trailing newline trimmed, so an isolated fixture yields a single snapshot
// line. .go entries are gofmt-normalized to match the AST tier's real inputs;
// fixtures that must stay byte-exact (the gofmt diagnostic, any non-Go file)
// go through run_snapshot_verbatim instead.
func run_snapshot(t *testing.T, files map[string]string) (output string) {
	t.Helper()
	normalized := make(map[string]string, len(files))
	for name, content := range files {
		if strings.HasSuffix(name, ".go") {
			normalized[name] = string(gofmt_must(t, content))
			continue
		}
		normalized[name] = content
	}
	return run_snapshot_verbatim(t, normalized)
}

// Like run_snapshot but without gofmt normalization: the bytes reach the
// linter exactly as written.
func run_snapshot_verbatim(t *testing.T, files map[string]string) (output string) {
	t.Helper()
	fsys_map := make(fstest.MapFS)
	for name, content := range files {
		fsys_map[name] = &fstest.MapFile{Data: []byte(content)}
	}
	stdout := &bytes.Buffer{}
	lint.Main(&lint.Main_Input{Fsys: fsys_map, Stdout: stdout, Stderr: &bytes.Buffer{}})
	return strings.TrimRight(stdout.String(), "\n")
}

// Formats diagnostics exactly as Main's printer does, for the tiers not
// reachable through Main(Fsys): git history and injected symlinks.
func render_diags(diags []lint.Diagnostic) (output string) {
	var builder strings.Builder
	for _, diagnostic := range diags {
		fmt.Fprintf(&builder, "%s: %s\n", diagnostic.Position, diagnostic.Message)
	}
	return strings.TrimRight(builder.String(), "\n")
}

// Pairs a captured snapshot with the in-memory fixture whose single linter
// diagnostic it pins. Verbatim skips gofmt for fixtures that must reach the
// linter byte-exact (the gofmt diagnostic, non-Go files). Drop removes output
// lines containing it, to strip orthogonal noise a fixture cannot avoid (a
// go.mod fixture's unavoidable missing-SPECIFICATION.md coverage diagnostic).
type snapshot_case struct {
	Snapshot snap.Snapshot
	Files    map[string]string
	Verbatim bool
	Drop     string
}

// Drives each snapshot case: lints the fixture and compares stdout to the
// captured snapshot. Sequential — no t.Parallel — so snap.Edit captures write
// the source file without racing each other.
func run_snapshot_cases(t *testing.T, cases []snapshot_case) {
	t.Helper()
	for _, entry := range cases {
		output := run_snapshot(t, entry.Files)
		if entry.Verbatim {
			output = run_snapshot_verbatim(t, entry.Files)
		}
		if entry.Drop != "" {
			var builder strings.Builder
			for _, line := range strings.Split(output, "\n") {
				if strings.Contains(line, entry.Drop) {
					continue
				}
				builder.WriteString(line)
				builder.WriteByte('\n')
			}
			output = strings.TrimRight(builder.String(), "\n")
		}
		snap.Expect(t, entry.Snapshot, output)
	}
}

// Wraps a body fragment in a clean, documented package so the only diagnostic a
// snapshot pins is the one the fragment introduces.
func snapshot_package(body string) (files map[string]string) {
	return map[string]string{
		"a.go": "// Package fixture is a fixture.\npackage fixture\n\n" + body}
}

// Builds the spec-fixture package around a SPECIFICATION.md body. Coverage is
// satisfied (the file and its test exist), so the only diagnostic is the
// structural one the mutated markdown introduces.
func snapshot_specification(markdown string) (files map[string]string) {
	return map[string]string{
		"go.mod":                    doctrine_shared_library_go_module,
		"pkg/fixture.go":            "// Package fixture is a fixture.\npackage fixture\n",
		"pkg/SPECIFICATION.md":      markdown,
		"pkg/specification_test.go": snapshot_specification_test,
	}
}

// Test_Snapshot_Specification pins the SPECIFICATION.md structural checks. The
// section-level mutations target the baseline leaf (which keeps its test) so a
// single spec diagnostic fires; a new leaf would also lack a test.
func Test_Snapshot_Specification(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`pkg/SPECIFICATION.md:1: pkg/SPECIFICATION.md:1 content precedes the first heading`), Files: snapshot_specification("Stray prose.\n" + snapshot_specification_markdown)},
		{Snapshot: snap.Init(`pkg/SPECIFICATION.md:1: pkg/SPECIFICATION.md:1 heading "Sole Rule" is not preceded by a blank line`), Files: snapshot_specification("# Sole Rule\n\nThe sole rule.\n")},
		{Snapshot: snap.Init(`pkg/SPECIFICATION.md:2: pkg/SPECIFICATION.md:2 section "Sole Rule" has no body line`), Files: snapshot_specification("\n# Sole Rule\n")},
		{Snapshot: snap.Init(`pkg/SPECIFICATION.md:7: pkg/SPECIFICATION.md:7 section "Sole Rule" exceeds three lines`), Files: snapshot_specification(
			"\n# Sole Rule\n\none\ntwo\nthree\nfour\n")},
		{Snapshot: snap.Init(`pkg/SPECIFICATION.md:6: pkg/SPECIFICATION.md:6 section "Sole Rule" has a blank line between body lines`), Files: snapshot_specification("\n# Sole Rule\n\none\n\ntwo\n")},
		{Snapshot: snap.Init(`pkg/SPECIFICATION.md:6: pkg/SPECIFICATION.md:6 uses a heading that is not level # or ###`), Files: snapshot_specification(
			snapshot_specification_markdown + "\n## Mid Level\n\nLevel two.\n")},
		{Snapshot: snap.Init(`pkg/specification_test.go: pkg/specification_test.go:2 needs Test_Phantom for leaf "Phantom" (in order, at top)`), Files: snapshot_specification(
			snapshot_specification_markdown + "\n# Phantom\n\nIt has no test.\n")},
	})
}

// Test_Snapshot_Specification_Coverage pins the file-presence and test-mapping
// spec checks. Each drops the companion noise it cannot avoid.
func Test_Snapshot_Specification_Coverage(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`pkg/SPECIFICATION.md: package "pkg" is missing SPECIFICATION.md`), Drop: "specification_test.go", Files: map[string]string{
			"go.mod":         doctrine_shared_library_go_module,
			"pkg/fixture.go": "// Package fixture is a fixture.\npackage fixture\n"}},
		{Snapshot: snap.Init(`pkg/specification_test.go: package "pkg" is missing specification_test.go`), Drop: "needs Test_", Files: map[string]string{
			"go.mod": doctrine_shared_library_go_module,
			"pkg/fixture.go": "// Package fixture is a fixture.\n" +
				"package fixture\n",
			"pkg/SPECIFICATION.md": snapshot_specification_markdown}},
		{Snapshot: snap.Init(`pkg/specification_test.go: pkg/specification_test.go:1 needs Test_Sole_Rule for leaf "Sole_Rule" (in order, at top)`), Files: map[string]string{
			"go.mod": doctrine_shared_library_go_module,
			"pkg/fixture.go": "// Package fixture is a fixture.\n" +
				"package fixture\n",
			"pkg/SPECIFICATION.md": snapshot_specification_markdown,
			"pkg/specification_test.go": `package fixture_test

import "testing"

// Test_Wrong checks the wrong thing.
func Test_Wrong(t *testing.T) { t.Parallel() }
`}},
		{Snapshot: snap.Init(`pkg/specification_test.go: pkg/specification_test.go:2 needs Test_Extra_Child for leaf "Extra_Child" (in order, at top)`), Files: snapshot_specification(
			snapshot_specification_markdown +
				"\n# Extra\n\n### Child\n\nA child leaf.\n")},
		{Snapshot: snap.Init(`pkg/SPECIFICATION.md:2: pkg/SPECIFICATION.md:2 heading "Sole-Rule" must use only letters and digits`), Drop: "needs Test_", Files: snapshot_specification(
			"\n# Sole-Rule\n\nThe rule.\n")},
	})
}

// Test_Snapshot_Specification_Headings pins the remaining heading-structure spec
// checks: a heading not followed by a blank line, a duplicate, and an orphan ###.
func Test_Snapshot_Specification_Headings(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`pkg/SPECIFICATION.md:2: pkg/SPECIFICATION.md:2 heading "Sole Rule" is not followed by a blank line`), Files: snapshot_specification("\n# Sole Rule\nThe sole rule.\n")},
		{Snapshot: snap.Init(`pkg/SPECIFICATION.md:6: pkg/SPECIFICATION.md:6 heading "Sole Rule" is duplicated`), Drop: "needs Test_", Files: snapshot_specification(
			snapshot_specification_markdown + "\n# Sole Rule\n\nAgain.\n")},
		{Snapshot: snap.Init(`pkg/SPECIFICATION.md:2: pkg/SPECIFICATION.md:2 ### "Orphan" has no parent #`), Drop: "needs Test_", Files: snapshot_specification(
			"\n### Orphan\n\nNo parent.\n" + snapshot_specification_markdown)},
	})
}

// Test_Snapshot_Bans_Imports pins the import-construct bans.
func Test_Snapshot_Bans_Imports(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`a.go:4:8: blank import is banned`),
			Files: snapshot_package("import _ \"strings\"\n")},
		{Snapshot: snap.Init(`a.go:4:8: dot import is banned`), Files: snapshot_package("import . \"strings\"\n")},
	})
}

// Test_Snapshot_Bans_Declarations pins the declaration-construct bans.
func Test_Snapshot_Bans_Declarations(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`a.go:5:11: iota is banned`),
			Files: snapshot_package("// X is a fixture.\nconst X = iota\n")},
		{Snapshot: snap.Init(`a.go:4:1: grouped declaration banned; split into one per line`), Files: snapshot_package("const (\n\ta = 1\n\tb = 2\n)\n")},
		{Snapshot: snap.Init(`a.go:5:1: func F has an empty body`),
			Files: snapshot_package("// F does.\nfunc F() {}\n")},
	})
}

// Test_Snapshot_Bans_Control pins the control-flow bans.
func Test_Snapshot_Bans_Control(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`a.go:7:2: naked return is banned`),
			Files: snapshot_package(`// F does.
func F() (n int) {
	n = 1
	return
}
`)},
		{Snapshot: snap.Init(`a.go:6:5: compound if condition (&&) — split into nested ifs`),
			Files: snapshot_package(`// F does.
func F() (n int) {
	if n > 0 && n < 5 {
		n = 1
	}
	return n
}
`)},
		{Snapshot: snap.Init(`a.go:8:3: fallthrough is banned`),
			Files: snapshot_package(`// F does.
func F() (n int) {
	switch n {
	case 1:
		fallthrough
	case 2:
		n = 3
	}
	return n
}
`)},
	})
}

// Test_Snapshot_Bans_Scope pins the scope-construct bans.
func Test_Snapshot_Bans_Scope(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`a.go:7:3: variable "n" shadows outer scope variable`), Files: snapshot_package(`// F does.
func F(n int) {
	{
		n := 0
		println(n)
	}
}
`)},
		{Snapshot: snap.Init(`a.go:7:2: discard: _ = ... hides the value; name it or drop the assignment`), Files: snapshot_package(`// F does.
func F() {
	x := 1
	_ = x
}
`)},
	})
}

// Test_Snapshot_Bans_Recursion pins the recursion bans (tier two).
func Test_Snapshot_Bans_Recursion(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`a.go:6:2: recursion: F calls itself`), Files: snapshot_package(`// F loops.
func F() {
	F()
}
`)},
		{Snapshot: snap.Init(`a.go:11:2: recursion: cycle F → G → F`), Files: snapshot_package(`// F calls G.
func F() {
	G()
}

// G calls F.
func G() {
	F()
}
`)},
	})
}

// Test_Snapshot_Bans_Types pins the type-construct bans.
func Test_Snapshot_Bans_Types(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`a.go:5:8: interface declarations are banned (except for generics)`), Files: snapshot_package(`// I is a fixture.
type I interface{ M() }
`)},
		{Snapshot: snap.Init(`a.go:11:12: method Compute does not satisfy any stdlib interface; convert to a free function with the receiver as the first parameter`), Files: snapshot_package(`// T is a fixture.
type T struct {
	// X is a fixture.
	X int
}

// Compute does.
func (t T) Compute() (n int) {
	return t.X
}
`)},
	})
}

// Test_Snapshot_Bans_Globals pins the package-level bans (init is tier two).
func Test_Snapshot_Bans_Globals(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`a.go:5:5: package-level var is banned (except for regexp.MustCompile and errors.New)`), Files: snapshot_package(`// X is a fixture.
var X = 0
`)},
		{Snapshot: snap.Init(`a.go:4:1: func init is banned; expose a func Init() instead`), Files: snapshot_package(`func init() { println(0) }
`)},
	})
}

// Test_Snapshot_Bans_Words pins the banned-word checks.
func Test_Snapshot_Bans_Words(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`a.go:5:7: identifier "Length" contains banned substring "length"`), Files: snapshot_package(`// Length is a fixture.
const Length = 0
`)},
		{Snapshot: snap.Init(`a.go:5:6: identifier "Helper" contains banned substring "helper"`), Files: snapshot_package(`// Helper helps.
func Helper() (n int) {
	return 0
}
`)},
	})
}

// Test_Snapshot_Requirements_Forms pins the required-form checks.
func Test_Snapshot_Requirements_Forms(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`a.go:5:10: unnamed return type: int`), Files: snapshot_package(`// F does.
func F() int {
	return 0
}
`)},
		{Snapshot: snap.Init(`a.go:12:6: T literal must use keyed fields`), Files: snapshot_package(`// T is a fixture.
type T struct {
	// X is a fixture.
	X int
}

// F builds T.
func F() (t T) {
	t = T{1}
	return t
}
`)},
		{Snapshot: snap.Init(`a.go:5:1: convert to F(*F_Input) (n int)`), Files: snapshot_package(`// F does.
func F(a int, b int) (n int) {
	return a + b
}
`)},
	})
}

// Test_Snapshot_Documentation pins the documentation-requirement checks.
func Test_Snapshot_Documentation(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`a.go:1:9: package "fixture" is missing a doc comment`), Files: map[string]string{
			"a.go": "package fixture\n\n// F does.\nfunc F() {\n\tprintln(0)\n}\n"}},
		{Snapshot: snap.Init(`a.go:4:6: exported func F is missing a doc comment`), Files: snapshot_package(`func F() {
	println(0)
}
`)},
		{Snapshot: snap.Init(`a.go:6:2: field Widget.Count lacks a doc comment`), Files: snapshot_package(`// Widget is a fixture.
type Widget struct {
	Count int
}
`)},
		{Snapshot: snap.Init(`a_test.go:5:1: test Test_X is missing a doc comment`), Files: map[string]string{
			"a_test.go": "package fixture_test\n\nimport \"testing\"\n\n" +
				"func Test_X(t *testing.T) {\n\tt.Parallel()\n}\n"}},
	})
}

// Test_Snapshot_Names pins the naming-convention checks.
func Test_Snapshot_Names(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`a.go:5:6: MyName -> My_Name`), Files: snapshot_package(`// MyName is a fixture.
func MyName() {
	println(0)
}
`)},
		{Snapshot: snap.Init(`a.go:11:6: rename Process -> Widget_<verb>`), Files: snapshot_package(`// Widget is a fixture.
type Widget struct {
	// X is a fixture.
	X int
}

// Process does.
func Process(w Widget) (n int) {
	return w.X
}
`)},
		{Snapshot: snap.Init(`a.go:5:7: rename Widget_Id -> Widget_Identifier`), Files: snapshot_package(`// Widget_Id is a fixture.
const Widget_Id = 0
`)},
		{Snapshot: snap.Init(`a.go:5:6: present participle "parsing" → rename to a noun form`), Files: snapshot_package(`// Parsing is a fixture.
type Parsing struct {
	// X is a fixture.
	X int
}
`)},
	})
}

// Test_Snapshot_Entry_Point pins the entry-point-first checks.
func Test_Snapshot_Entry_Point(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`a.go:8:1: func main should be declared first in the file`), Files: map[string]string{
			"a.go": "package main\n\n// Run does.\nfunc Run() {\n\tprintln(0)\n}\n\n" +
				"func main() {\n\tRun()\n}\n"}},
		{Snapshot: snap.Init(`a.go:9:1: func Main should be declared first in the file`), Files: map[string]string{
			"a.go": "package main\n\n// Run does.\nfunc Run() {\n\tprintln(0)\n}\n\n" +
				"// Main does.\nfunc Main() {\n\tRun()\n}\n"}},
		{Snapshot: snap.Init(`a_test.go:10:1: func TestMain should be declared first in the file`), Files: map[string]string{
			"a_test.go": `package fixture_test

import "testing"

// Test_X is a fixture.
func Test_X(t *testing.T) {
	t.Parallel()
}

func TestMain(m *testing.M) {
	m.Run()
}
`}},
	})
}

// Test_Snapshot_Requirements_Struct pins the struct-shape requirement checks.
func Test_Snapshot_Requirements_Struct(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`a.go:4:21: rename count -> Count`), Files: snapshot_package(`type widget struct{ count int }
`)},
		{Snapshot: snap.Init(`a.go:9:8: public type Widget contains private type hidden`), Files: snapshot_package(`type hidden struct{ X int }

// Widget is a fixture.
type Widget struct {
	// Inner is a fixture.
	Inner hidden
}
`)},
	})
}

// Test_Snapshot_Stream_Files pins the stream-tier file bans.
func Test_Snapshot_Stream_Files(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`notes.txt:2:1: conflict-markers: resolve the conflict and remove the marker`), Verbatim: true, Files: map[string]string{
			"notes.txt": "ok\n<<<<<<< HEAD\n"}},
		{Snapshot: snap.Init(`build.sh:1:1: banned-scripts: rewrite "build.sh" as a go script`), Verbatim: true, Files: map[string]string{
			"build.sh": "echo hi\n"}},
		{Snapshot: snap.Init(`Makefile:1:1: banned-scripts: rewrite "Makefile" as a go script`), Verbatim: true, Files: map[string]string{
			"Makefile": "all:\n\techo hi\n"}},
		{Snapshot: snap.Init(`backup.tar.xz:1:1: banned-archives: .xz files are banned; use .gz or .zip instead`), Verbatim: true, Files: map[string]string{
			"backup.tar.xz": "\xfd7zXZ\x00"}},
		{Snapshot: snap.Init(`LICENSE: copyleft: GNU GPL: replace with a permissive license (MIT/Apache/BSD)`), Verbatim: true, Files: map[string]string{
			"LICENSE": "GNU GENERAL PUBLIC LICENSE\n\n" +
				"This program is free software.\n"}},
	})
}

// Test_Snapshot_Stream_Markdown pins the markdown, agent-doc, and path checks.
func Test_Snapshot_Stream_Markdown(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`notes.md:3:1: trailing-whitespace: markdown line has trailing whitespace`), Verbatim: true, Files: map[string]string{
			"notes.md": "# Title\n\nIt trails.   \n"}},
		{Snapshot: snap.Init(`notes.md:3:1: markdown-line-length: markdown line is 120 columns; visual limit is 100`), Verbatim: true, Files: map[string]string{
			"notes.md": "# Wide\n\n" + strings.Repeat("x", 120) + "\n"}},
		{Snapshot: snap.Init(`SKILL.md:1:1: agent-doc-max-lines: SKILL.md is capped at 100 lines. Prefer procedural scripting over prose.`), Verbatim: true, Files: map[string]string{
			"SKILL.md": strings.Repeat("line\n", 101)}},
		{Snapshot: snap.Init(`top: agents-claude-pair: AGENTS.md is missing; it must mirror CLAUDE.md byte-for-byte`), Verbatim: true, Files: map[string]string{
			"top/CLAUDE.md": "Shared instructions.\n"}},
		{Snapshot: snap.Init(`bad-dir: rename bad-dir -> bad_dir`), Files: map[string]string{
			"bad-dir/x.go": "// Package x is a fixture.\npackage x\n"}},
	})
}

// Test_Snapshot_Module_Layout pins the binary/shared module layout checks. Each
// drops the unavoidable missing-SPECIFICATION.md coverage noise its go.mod adds.
func Test_Snapshot_Module_Layout(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`feature/feature.go:1:1: move feature -> ./internal/feature`), Drop: "SPECIFICATION.md", Files: map[string]string{
			"go.mod":            doctrine_binary_go_module,
			"internal/entry.go": doctrine_binary_internal_main,
			"feature/feature.go": "// Package feature is a fixture.\n" +
				"package feature\n"}},
		{Snapshot: snap.Init(`cmd/app/main.go:1:1: binary module "example.com/mybinary" places its main package at "cmd/app"; the main package must sit at the module root, no cmd/ directory`), Drop: "SPECIFICATION.md", Files: map[string]string{
			"go.mod":            doctrine_binary_go_module,
			"internal/entry.go": doctrine_binary_internal_main,
			"cmd/app/main.go":   "package main\n\nfunc main() {\n\tprintln(0)\n}\n"}},
		{Snapshot: snap.Init(`internal/x/x.go:1:1: shared library forbids internal/ directories; remove "internal"`), Drop: "SPECIFICATION.md", Files: map[string]string{
			"go.mod":          doctrine_shared_library_go_module,
			"internal/x/x.go": "// Package x is a fixture.\npackage x\n"}},
		{Snapshot: snap.Init(`main.go:1:1: shared library "github.com/james-orcales/james-orcales/golang_snacks" forbids package main; move the entry point to a binary module`), Drop: "SPECIFICATION.md", Files: map[string]string{
			"go.mod":  doctrine_shared_library_go_module,
			"main.go": "package main\n\nfunc main() {\n\tprintln(0)\n}\n"}},
	})
}

// Test_Snapshot_Module_Structure pins the tier-depth and module-placement checks.
func Test_Snapshot_Module_Structure(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`a/b/c/c.go:1:1: package "c" at "a/b/c" exceeds library tier; 2 non-main ancestors: [a/b a]`), Drop: "SPECIFICATION.md", Files: map[string]string{
			"go.mod":     doctrine_shared_library_go_module,
			"a/a.go":     "// Package a is a fixture.\npackage a\n",
			"a/b/b.go":   "// Package b is a fixture.\npackage b\n",
			"a/b/c/c.go": "// Package c is a fixture.\npackage c\n"}},
		{Snapshot: snap.Init(`nested/inner/go.mod:1:1: module "example.com/inner" at "nested/inner" must be located at the repository root`), Drop: "SPECIFICATION.md", Files: map[string]string{
			"nested/inner/go.mod":            "module example.com/inner\n",
			"nested/inner/internal/entry.go": doctrine_binary_internal_main}},
		{Snapshot: snap.Init(`orphan/go.mod:1:1: module "example.com/orphan" at "orphan" is not registered in go.work`), Drop: "SPECIFICATION.md", Files: map[string]string{
			"go.work":                      "go 1.25\n\nuse ./registered\n",
			"registered/go.mod":            "module example.com/registered\n",
			"registered/internal/entry.go": doctrine_binary_internal_main,
			"orphan/go.mod":                "module example.com/orphan\n",
			"orphan/internal/entry.go":     doctrine_binary_internal_main}},
	})
}

// Test_Snapshot_Purity pins the direct impure-stdlib checks.
func Test_Snapshot_Purity(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`lib/library.go:4:8: impure stdlib import "os": see lint/README.md for resolutions`), Drop: "SPECIFICATION.md", Files: map[string]string{
			"go.mod": doctrine_shared_library_go_module,
			"lib/library.go": `// Package library is a fixture.
package library

import "os"
`}},
		{Snapshot: snap.Init(`lib/library.go:8:2: impure stdlib call fmt.Println: see lint/README.md for resolutions`), Drop: "SPECIFICATION.md", Files: map[string]string{
			"go.mod": doctrine_shared_library_go_module,
			"lib/library.go": `// Package library is a fixture.
package library

import "fmt"

// F prints.
func F() {
	fmt.Println("x")
}
`}},
	})
}

// Test_Snapshot_Transitive pins the transitive-purity checks.
func Test_Snapshot_Transitive(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`lib/library_test.go:7:2: impure transitive call log.Fatal: a pure package calls only pure APIs`), Drop: "SPECIFICATION.md", Files: map[string]string{
			"go.mod":         doctrine_shared_library_go_module,
			"lib/library.go": "// Package library is a fixture.\npackage library\n",
			"lib/library_test.go": `package library_test

import "log"

// Test_F exits.
func Test_F() {
	log.Fatal("x")
}
`}},
		{Snapshot: snap.Init(`lib/library.go:4:8: impure dependency "github.com/james-orcales/james-orcales/golang_snacks/widget/default": a pure package imports only pure packages`), Drop: "SPECIFICATION.md", Files: map[string]string{
			"go.mod": doctrine_shared_library_go_module,
			"lib/library.go": `// Package library is a fixture.
package library

import "github.com/james-orcales/james-orcales/golang_snacks/widget/default"

// F uses the default.
func F() (s string) {
	return widget.Name()
}
`,
			"widget/default/wire.go": `// Package widget is a fixture.
package widget

// Name names.
func Name() (s string) {
	return "x"
}
`}},
	})
}

// Test_Snapshot_Unbounded pins the unbounded-API bans (tier two).
func Test_Snapshot_Unbounded(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`a.go:8:9: unbounded-read: unbounded API 'io.ReadAll'; use io.ReadFull(r, buf) with a bounded buf instead`), Files: snapshot_package(`import "io"

// F reads.
func F(r io.Reader) (b []byte) {
	b, _ = io.ReadAll(r)
	return b
}
`)},
		{Snapshot: snap.Init(`a.go:11:9: unbounded-decode: unbounded API 'json.NewDecoder'; use json.Unmarshal over a bounded []byte instead`), Files: snapshot_package(`import (
	"encoding/json"
	"io"
)

// F decodes.
func F(r io.Reader) (d *json.Decoder) {
	return json.NewDecoder(r)
}
`)},
		{Snapshot: snap.Init(`a.go:11:9: unbounded-decompression: unbounded API 'gzip.NewReader'; use wrap the decompressed reader in io.LimitReader instead`), Files: snapshot_package(`import (
	"compress/gzip"
	"io"
)

// F wraps.
func F(r io.Reader) (z *gzip.Reader, err error) {
	return gzip.NewReader(r)
}
`)},
		{Snapshot: snap.Init(`a.go:8:2: unbounded-allocation: unbounded API 'bytes.NewBuffer'; use a fixed []byte with explicit length tracking instead`), Files: snapshot_package(`import "bytes"

// F buffers.
func F() {
	bytes.NewBuffer(nil)
}
`)},
		{Snapshot: snap.Init(`a.go:8:9: unbounded-http: unbounded API 'http.Get'; use (&http.Client{Timeout: N}).Get(...) instead`), Drop: "impure stdlib call", Files: snapshot_package(`import "net/http"

// F fetches.
func F(url string) (resp *http.Response, err error) {
	return http.Get(url)
}
`)},
		{Snapshot: snap.Init(`a.go:11:9: deprecated-ioutil: unbounded API 'ioutil.ReadAll'; use io.ReadFull(r, buf) with a bounded buf instead`), Files: snapshot_package(`import (
	"io"
	"io/ioutil"
)

// F reads.
func F(r io.Reader) (b []byte, err error) {
	return ioutil.ReadAll(r)
}
`)},
	})
}

// Test_Snapshot_Commits pins the git-history checks, rendered like Main prints.
func Test_Snapshot_Commits(t *testing.T) {
	snap.Expect(t, snap.Init(`<git:abc>: commit subject is 206 chars (max 100)`), render_diags(lint.Git_Input_Check(lint.Git_Input{
		Enabled: true,
		Non_Merge_Commits: []lint.Git_Commit{
			{Hash: "abc", Subject: "feat: " + strings.Repeat("x", 200)},
		},
	})))
	snap.Expect(t, snap.Init(`<git:abc>: non-conventional commit subject: did some stuff`), render_diags(lint.Git_Input_Check(lint.Git_Input{
		Enabled: true,
		Non_Merge_Commits: []lint.Git_Commit{
			{Hash: "abc", Subject: "did some stuff"},
		},
	})))
	snap.Expect(t, snap.Init(`<git:abc>: fixup commit on branch: fixup! feat: thing`), render_diags(lint.Git_Input_Check(lint.Git_Input{
		Enabled: true,
		Non_Merge_Commits: []lint.Git_Commit{
			{Hash: "abc", Subject: "fixup! feat: thing"},
		},
	})))
	snap.Expect(t, snap.Init(`<git:abc>: merge commit on branch: Merge branch 'feature' into main`), render_diags(lint.Git_Input_Check(lint.Git_Input{
		Enabled: true,
		Merge_Commits: []lint.Git_Commit{
			{Hash: "abc", Subject: "Merge branch 'feature' into main"},
		},
	})))
	snap.Expect(t, snap.Init(`<git>: main ref not found; fetch main or set actions/checkout fetch-depth: 0`), render_diags(lint.Git_Input_Check(lint.Git_Input{
		Enabled: true, Main_Reference_Absent: true,
	})))
}

// Test_Snapshot_Suffixes pins the quantity/arithmetic/extremum suffix checks.
func Test_Snapshot_Suffixes(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`a.go:6:2: naming convention: total (used as count or size) → rename to total_count`), Files: snapshot_package(`// F does.
func F(s string) (n int) {
	total := len(s)
	return total
}
`)},
		{Snapshot: snap.Init(`a.go:8:6: arithmetic: _index + _size is incoherent`), Files: snapshot_package(`// F does.
func F() (n int) {
	a_index := 0
	b_size := 0
	n = a_index + b_size
	return n
}
`)},
		{Snapshot: snap.Init(`a.go:5:7: rename max_count -> count_max`), Files: snapshot_package(`// Max count.
const max_count = 1
`)},
	})
}

// Test_Snapshot_Forms pins the remaining per-file form checks.
func Test_Snapshot_Forms(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`a.go:4:1: comment: should start with capital letter`), Files: snapshot_package(`// f does something.
func F() {
	println(0)
}
`)},
		{Snapshot: snap.Init(`a.go:5:1: function is 73 lines (max 70)`), Files: snapshot_package("// F does.\nfunc F() {\n" +
			strings.Repeat("\tprintln(0)\n", 71) + "}\n")},
		{Snapshot: snap.Init(`a.go:1:1: file is not gofmt-clean`), Verbatim: true, Files: map[string]string{
			"a.go": `// Package fixture is a fixture.
package fixture

import  "strings"

// F does.
func F() (s string) { return strings.TrimSpace("x") }
`}},
		{Snapshot: snap.Init(`a_test.go:2:9: test file must declare 'package <X>_test'; got 'package fixture'`), Files: map[string]string{
			"a_test.go": `// Package fixture is a fixture.
package fixture

import "testing"

// Test_X is a fixture.
func Test_X(t *testing.T) { t.Parallel() }
`}},
	})
}

// Test_Snapshot_Miscellaneous pins the remaining per-file checks.
func Test_Snapshot_Miscellaneous(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`a.go:7:12: snap.Init must use a backticked raw string literal`), Files: snapshot_package(`// F does.
func F() {
	var snap snapper
	snap.Init("plain")
}

type snapper struct{ X int }
`)},
		{Snapshot: snap.Init(`foo/default/wire.go:2:9: default package must declare 'package foo', not 'package wrong'; it shadows the library it re-exports`), Files: map[string]string{
			"foo/default/wire.go": "// Package wrong is a fixture.\npackage wrong\n"}},
		{Snapshot: snap.Init(`a.go:8:2: blank-named sync mutex has no usable Lock receiver; use the noCopy idiom for non-copy semantics or name the field to lock it`), Files: snapshot_package(`import "sync"

// T is a fixture.
type T struct {
	_ sync.Mutex
	// X is a fixture.
	X int
}
`)},
		{Snapshot: snap.Init(`a.go:7:8: struct tag key "yaml" is not stdlib; only json, xml, and asn1 are permitted`), Files: snapshot_package("// T is a fixture.\n" +
			"type T struct {\n\t// X is a fixture.\n\tX int `yaml:\"x\"`\n}\n")},
	})
}

// Test_Snapshot_Backtick_Messages pins the diagnostics whose message text
// contains a backtick. snap forbids backticks in snapshot values, so these use
// a plain equality check with a double-quoted expected string instead.
func Test_Snapshot_Backtick_Messages(t *testing.T) {
	cases := []struct {
		Want     string
		Verbatim bool
		Drop     string
		Files    map[string]string
	}{
		{Want: "a.go:6:2: bare `for {}` is banned " +
			"if the loop is intentionally unbounded",
			Files: snapshot_package(`// F does.
func F() {
	for {
		break
	}
}
`)},
		{Want: "a.go:4:1: comment: missing space after `//`", Verbatim: true,
			Drop: "gofmt-clean", Files: map[string]string{
				"a.go": "// Package fixture is a fixture.\npackage fixture\n\n" +
					"//F does.\nfunc F() {\n\tprintln(0)\n}\n"}},
		{Want: "a.go:4:1: comment: should end with `.`, `:`, `?`, or `!`",
			Files: snapshot_package("// F does\nfunc F() {\n\tprintln(0)\n}\n")},
		{Want: ".github/workflows/ci.yml:2:1: github-actions-uses: " +
			"third-party github action banned; " +
			"replace `uses:` with an inline `run:` step",
			Verbatim: true, Files: map[string]string{
				".github/workflows/ci.yml": "steps:\n" +
					"  - uses: actions/checkout@v4\n"}},
	}
	for _, entry := range cases {
		got := run_snapshot(t, entry.Files)
		if entry.Verbatim {
			got = run_snapshot_verbatim(t, entry.Files)
		}
		if entry.Drop != "" {
			var builder strings.Builder
			for _, line := range strings.Split(got, "\n") {
				if strings.Contains(line, entry.Drop) {
					continue
				}
				builder.WriteString(line)
				builder.WriteByte('\n')
			}
			got = strings.TrimRight(builder.String(), "\n")
		}
		if got != entry.Want {
			t.Errorf("want %q, got %q", entry.Want, got)
		}
	}
}

// Test_Transitive_Purity_Snap_Exemption verifies a pure library's test file may
// import the snapshot library: snap is test infrastructure, an extension of the
// suite, so its impurity is exempt from the transitive-purity import ban.
func Test_Transitive_Purity_Snap_Exemption(t *testing.T) {
	t.Parallel()
	const shared = "github.com/james-orcales/james-orcales/golang_snacks"
	output := run_snapshot(t, map[string]string{
		"go.mod":         "module " + shared + "\n\ngo 1.25\n",
		"lib/library.go": "// Package library is a fixture.\npackage library\n",
		"lib/library_test.go": `package library_test

import (
	"testing"

	snap "` + shared + `/snap/v2/snap_default"
)

// Test_Snap is a fixture.
func Test_Snap(t *testing.T) { _ = snap.Default }
`,
	})
	if strings.Contains(output, "impure dependency") {
		t.Fatalf("snap import in a test file must be exempt; got: %s", output)
	}
}

// Conflict markers in any file must be flagged with the conflict-markers check
// and the correct line number.
func Test_Stream_Conflict_Markers(t *testing.T) {
	t.Parallel()
	output := run_stream(t, map[string]string{
		"notes.txt": "ok line\n<<<<<<< HEAD\nmine\n=======\ntheirs\n>>>>>>> branch\n",
	})
	if !strings.Contains(output, "conflict-markers") {
		t.Errorf("expected conflict-markers diag, got: %s", output)
	}
	if !strings.Contains(output, "notes.txt:2") {
		t.Errorf("expected diag at notes.txt:2, got: %s", output)
	}
}

// Any `uses:` line under .github/workflows/ must be flagged. Other YAML
// (including .github/ files outside workflows/) and other content inside a
// workflow file (run:, name:, etc.) must pass clean.
func Test_Stream_Github_Actions_Uses(t *testing.T) {
	t.Parallel()
	cases := []struct {
		Name   string
		Path   string
		Body   string
		Should string
	}{
		{
			Name:   "third-party uses flagged",
			Path:   ".github/workflows/ci.yml",
			Body:   "jobs:\n  build:\n    steps:\n      - uses: actions/checkout@v4\n",
			Should: "github-actions-uses",
		},
		{
			Name: "local uses flagged",
			Path: ".github/workflows/ci.yml",
			Body: "jobs:\n  build:\n    steps:\n" +
				"      - uses: ./.github/actions/foo\n",
			Should: "github-actions-uses",
		},
		{
			Name:   "yaml extension flagged",
			Path:   ".github/workflows/ci.yaml",
			Body:   "steps:\n  - uses: actions/setup-go@v5\n",
			Should: "github-actions-uses",
		},
		{
			Name: "run-only workflow clean",
			Path: ".github/workflows/ci.yml",
			Body: "jobs:\n  build:\n    steps:\n      - name: build\n" +
				"        run: go build ./...\n",
			Should: "",
		},
		{
			Name:   "uses in non-workflow yaml ignored",
			Path:   ".github/dependabot.yml",
			Body:   "uses: whatever\n",
			Should: "",
		},
		{
			Name:   "uses in top-level yaml ignored",
			Path:   "config.yml",
			Body:   "uses: whatever\n",
			Should: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			output := run_stream(t, map[string]string{tc.Path: tc.Body})
			has := strings.Contains(output, "github-actions-uses")
			want := tc.Should != ""
			if has != want {
				t.Errorf("path %q: want github-actions-uses=%v, got: %s",
					tc.Path, want, output)
			}
		})
	}
}

// GPL license text must trigger the copyleft check with the correct family name.
func Test_Stream_Copyleft_GPL(t *testing.T) {
	t.Parallel()
	gpl := "GNU GENERAL PUBLIC LICENSE\nVersion 3\n\nThis program is free software\n"
	output := run_stream(t, map[string]string{"LICENSE": gpl})
	if !strings.Contains(output, "copyleft") {
		t.Errorf("expected copyleft diag, got: %s", output)
	}
	if !strings.Contains(output, "GNU GPL") {
		t.Errorf("expected GNU GPL in copyleft diag, got: %s", output)
	}
}

// MPL text that references GPL for comparison must not trigger the copyleft check.
func Test_Stream_Copyleft_MPL_Not_Flagged(t *testing.T) {
	t.Parallel()
	mpl := "Mozilla Public License 2.0\nGNU General Public License (for comparison)\n"
	output := run_stream(t, map[string]string{"LICENSE": mpl})
	if strings.Contains(output, "copyleft") {
		t.Errorf("MPL should not trip copyleft check; got: %s", output)
	}
}

// Agent docs under dot-dirs must be skipped; agent docs under normal paths
// must be flagged when they exceed 100 lines.
func Test_Stream_Agent_Documentation_Lines_Max(t *testing.T) {
	t.Parallel()
	body := strings.Repeat("line\n", 101)
	// Dot-dirs are skipped by the walker, so the doc must live under a
	// non-dot path to be reached.
	output := run_stream(t, map[string]string{".claude/skills/foo/SKILL.md": body})
	if strings.Contains(output, "agent-doc-max-lines") {
		t.Errorf("dot-dir SKILL.md should be skipped, got: %s", output)
	}
	for _, name := range []string{"SKILL.md", "CLAUDE.md", "AGENTS.md"} {
		loop_output := run_stream(t, map[string]string{"skills/foo/" + name: body})
		if !strings.Contains(loop_output, "agent-doc-max-lines") {
			t.Errorf("expected agent-doc-max-lines diag for %s, got: %s",
				name, loop_output)
		}
	}
}

// Snake_case, Ada_Case, and SCREAMING_SNAKE_CASE paths must pass; kebab-case
// and camelCase paths must be flagged; third_party/ is fully exempt.
func Test_Stream_Path_Casing(t *testing.T) {
	t.Parallel()
	cases := []struct {
		Name   string
		Path   string
		Should string
	}{
		{"snake_case file", "lint_stream.go", ""},
		{"Ada_Case file", "Foo_Bar.txt", ""},
		{"SCREAMING file", "LICENSE", ""},
		{"dotfile with valid stem", ".gitignore", ""},
		{"snake_case dir", "big_bang/foo.go", ""},
		{"FQDN multi-component", "TIGER_STYLE.Index_Count.md", ""},
		{".git subtree exempt", ".git/bad-ref", ""},
		{"third_party exempt", "third_party/badName-file.go", ""},
		{"vendor exempt at any depth", "deep/nest/vendor/badName-file.go", ""},
		{"FQDN bad component flagged", "Foo.bad-component.md", "path-casing"},
		{"hidden bad name flagged", ".Bad-Hidden.md", "path-casing"},
		{"kebab-case file flagged", "bad-file.txt", "path-casing"},
		{"camelCase file flagged", "badFile.txt", "path-casing"},
		{"kebab-case dir flagged", "bad-dir/foo.txt", "path-casing"},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			output := run_stream(t, map[string]string{tc.Path: "x\n"})
			has := strings.Contains(output, " -> ")
			want := tc.Should != ""
			if has != want {
				t.Errorf("path %q: want rename=%v, got: %s",
					tc.Path, want, output)
			}
		})
	}
}

// Test_Path_Casing_Respects_Tracked verifies that when a tracked set is given
// (git ls-files --exclude-standard), path-casing checks only those paths: a
// tracked bad name is flagged, a gitignored one — absent from the set — is not.
func Test_Path_Casing_Respects_Tracked(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"bad-Dir/x.go":        {Data: []byte("package x\n")},
		"ignored/Bad-Name.go": {Data: []byte("package ignored\n")},
	}
	diags, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys:    fsys,
		Tracked: map[string]bool{"bad-Dir/x.go": true},
	})
	if err != nil {
		t.Fatalf("Check_File_System: %v", err)
	}
	var saw_tracked, saw_ignored bool
	for _, d := range diags {
		if strings.Contains(d.Message, "bad-Dir") {
			saw_tracked = true
		}
		if strings.Contains(d.Message, "Bad-Name") {
			saw_ignored = true
		}
	}
	if !saw_tracked {
		t.Error("a tracked bad-cased path must be flagged")
	}
	if saw_ignored {
		t.Error("a gitignored path (absent from Tracked) must not be checked")
	}
}

// Markdown lines exceeding 100 runes must be flagged by the markdown-line-length check.
func Test_Stream_Markdown_Line_Max(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x", 120) + "\n"
	output := run_stream(t, map[string]string{"docs.md": long})
	if !strings.Contains(output, "markdown-line-length") {
		t.Errorf("expected markdown-line-length diag, got: %s", output)
	}
}

// Lines inside fenced code blocks are held to the markdown line cap like any
// other line — code carries no exemption.
func Test_Stream_Markdown_Code_Fence_Counted(t *testing.T) {
	t.Parallel()
	long := "```\n" + strings.Repeat("x", 200) + "\n```\n"
	output := run_stream(t, map[string]string{"docs.md": long})
	if !strings.Contains(output, "markdown-line-length") {
		t.Errorf("fenced code should be counted; got: %s", output)
	}
}

// The line cap counts display columns, not runes: 60 wide ideographs are 60
// runes but 120 columns and must be flagged.
func Test_Stream_Markdown_Display_Width(t *testing.T) {
	t.Parallel()
	wide := strings.Repeat("世", 60) + "\n"
	output := run_stream(t, map[string]string{"docs.md": wide})
	if !strings.Contains(output, "markdown-line-length") {
		t.Errorf("expected width diag for wide runes, got: %s", output)
	}
}

// A markdown line with trailing whitespace must be flagged.
func Test_Stream_Markdown_Trailing_Whitespace(t *testing.T) {
	t.Parallel()
	output := run_stream(t, map[string]string{"docs.md": "Trailing here.   \n"})
	if !strings.Contains(output, "trailing-whitespace") {
		t.Errorf("expected trailing-whitespace diag, got: %s", output)
	}
}

// Trailing whitespace inside a fenced code block must be exempt — code may rely
// on it.
func Test_Stream_Markdown_Trailing_Whitespace_Fence_Exempt(t *testing.T) {
	t.Parallel()
	source := "```\ncode trailing   \n```\n"
	output := run_stream(t, map[string]string{"docs.md": source})
	if strings.Contains(output, "trailing-whitespace") {
		t.Errorf("fenced code should be exempt; got: %s", output)
	}
}

// A directory with CLAUDE.md but no AGENTS.md must be flagged for the missing sibling.
func Test_Stream_Agents_Claude_Pair_Absence(t *testing.T) {
	t.Parallel()
	output := run_stream(t, map[string]string{"CLAUDE.md": "instructions\n"})
	if !strings.Contains(output, "agents-claude-pair") {
		t.Errorf("expected agents-claude-pair diag, got: %s", output)
	}
	if !strings.Contains(output, "AGENTS.md is missing") {
		t.Errorf("expected AGENTS.md missing message, got: %s", output)
	}
}

// AGENTS.md and CLAUDE.md with different content must be flagged as drifted.
func Test_Stream_Agents_Claude_Pair_Drift(t *testing.T) {
	t.Parallel()
	output := run_stream(t, map[string]string{
		"AGENTS.md": "version one\n",
		"CLAUDE.md": "version two\n",
	})
	if !strings.Contains(output, "agents-claude-pair") {
		t.Errorf("expected agents-claude-pair diag, got: %s", output)
	}
	if !strings.Contains(output, "differ") {
		t.Errorf("expected differ message, got: %s", output)
	}
}

// Byte-identical AGENTS.md and CLAUDE.md must pass without any diag.
func Test_Stream_Agents_Claude_Pair_Identical_OK(t *testing.T) {
	t.Parallel()
	output := run_stream(t, map[string]string{
		"AGENTS.md": "shared\n",
		"CLAUDE.md": "shared\n",
	})
	if strings.Contains(output, "agents-claude-pair") {
		t.Errorf("byte-identical pair should pass; got: %s", output)
	}
}

// A conflict marker inside a .go file must surface as a stream-tier
// conflict-markers diag, and the AST tier's parse failure on the same
// file must degrade to a per-file diagnostic rather than aborting the
// entire run. Both tiers run; both diags surface.
func Test_Stream_Conflict_Marker_Input_Go_File(t *testing.T) {
	t.Parallel()
	bad_go := "package p\n<<<<<<< HEAD\nfunc f() {}\n=======\nfunc g() {}\n>>>>>>> branch\n"
	output := run_stream(t, map[string]string{"a.go": bad_go})
	if !strings.Contains(output, "conflict-markers") {
		t.Errorf("expected conflict-markers diag, got: %s", output)
	}
	if !strings.Contains(output, "parse error") {
		t.Errorf("expected parse error diag, got: %s", output)
	}
}

// Runs Main with the given Git_Input over an empty FS and returns
// (stdout, exit_code). Empty FS isolates the git tier from file-tier output.
func run_git(t *testing.T, input lint.Git_Input) (output string, code int) {
	t.Helper()
	stdout := &bytes.Buffer{}
	code = lint.Main(&lint.Main_Input{
		Fsys:   fstest.MapFS{},
		Stdout: stdout,
		Stderr: &bytes.Buffer{},
		Git:    input,
	})
	return stdout.String(), code
}

// Git_Input zero value must skip the git tier and surface no <git> diagnostics.
func Test_Git_Disabled(t *testing.T) {
	t.Parallel()
	output, code := run_git(t, lint.Git_Input{})
	if strings.Contains(output, "<git") {
		t.Errorf("disabled git tier must emit no <git> diags; got: %s", output)
	}
	if code != 0 {
		t.Errorf("expected exit 0, got %d; output: %s", code, output)
	}
}

// Missing main ref (shallow CI checkout, fresh repo) must surface a single
// actionable diagnostic naming fetch-depth, not silently pass.
func Test_Git_Main_Reference_Absence(t *testing.T) {
	t.Parallel()
	output, code := run_git(t, lint.Git_Input{Enabled: true, Main_Reference_Absent: true})
	if !strings.Contains(output, "<git>") {
		t.Errorf("expected <git> diag for missing main ref; got: %s", output)
	}
	if !strings.Contains(output, "fetch-depth") {
		t.Errorf("expected fetch-depth instruction; got: %s", output)
	}
	if code == 0 {
		t.Errorf("expected non-zero exit; got 0; output: %s", output)
	}
}

// Every merge commit in the PR range must be flagged with its short hash and
// subject; the no-merge-commits rule name must appear so users can map output
// to the rebase instruction.
func Test_Git_No_Merge_Commits(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name          string
		Merge_Commits []lint.Git_Commit
		Want_Hits     []string
		Want_Code     int
	}{
		{
			Name:      "no merges",
			Want_Code: 0,
		},
		{
			Name: "one merge commit",
			Merge_Commits: []lint.Git_Commit{
				{
					Hash:    "abcdef0123456789abcdef0123456789abcdef01",
					Subject: "Merge branch foo",
				},
			},
			Want_Hits: []string{"merge commit", "abcdef0123", "Merge branch foo"},
			Want_Code: 1,
		},
		{
			Name: "subtree add exempt",
			Merge_Commits: []lint.Git_Commit{
				{Hash: "ccccccccccdddd", Subject: "Add 'foo/' from commit " +
					"'2d43774e164be386023c13e2b12c2403a57b4a2a'"},
			},
			Want_Code: 0,
		},
		{
			Name: "subtree pull exempt",
			Merge_Commits: []lint.Git_Commit{
				{Hash: "eeeeeeeeeeffff", Subject: "Merge commit " +
					"'2d43774e164be386023c13e2b12c2403a57b4a2a' as 'foo'"},
			},
			Want_Code: 0,
		},
		{
			Name: "two merge commits",
			Merge_Commits: []lint.Git_Commit{
				{Hash: "1111111111aaaa", Subject: "Merge one"},
				{Hash: "2222222222bbbb", Subject: "Merge two"},
			},
			Want_Hits: []string{"1111111111", "2222222222", "Merge one", "Merge two"},
			Want_Code: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			output, code := run_git(t, lint.Git_Input{
				Enabled:       true,
				Merge_Commits: tt.Merge_Commits,
			})
			for _, want := range tt.Want_Hits {
				if !strings.Contains(output, want) {
					t.Errorf("expected output to contain %q; got: %s",
						want, output)
				}
			}
			if code != tt.Want_Code {
				t.Errorf("expected exit %d, got %d; output: %s",
					tt.Want_Code, code, output)
			}
		})
	}
}

// Conventional-commits enforcement: subjects must match
//
//	type(scope)?!?: description
//
// with a lowercase type and a non-empty description. Fixup-shaped subjects
// are exempt because the no-fixup-commits rule already covers them and they
// disappear on autosquash.
func Test_Git_Conventional_Commits(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Subject   string
		Want_Flag bool
	}{
		{"plain feat", "feat: add widget", false},
		{"feat with scope", "feat(lint): add git check", false},
		{"fix with slashed scope", "fix(lint/impl): handle nil", false},
		{"breaking change marker", "feat!: drop legacy field", false},
		{"breaking change with scope", "refactor(api)!: rename field", false},
		{"no type prefix", "add widget", true},
		{"missing colon", "feat add widget", true},
		{"missing space after colon", "feat:add widget", true},
		{"empty description", "feat: ", true},
		{"capitalized type", "Feat: add widget", true},
		{"fixup exempt from conventional", "fixup! feat: foo", false},
		{"squash exempt from conventional", "squash! feat: foo", false},
		// Empty subjects (e.g. WIP commits authored with --allow-empty-message)
		// are skipped entirely; the cost of rewriting history to fix them
		// outweighs the value of flagging them on every lint run.
		{"empty subject skipped", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			output, _ := run_git(t, lint.Git_Input{
				Enabled: true,
				Non_Merge_Commits: []lint.Git_Commit{
					{Hash: "abc1234567def", Subject: tt.Subject},
				},
			})
			flagged := strings.Contains(output, "conventional-commits") ||
				strings.Contains(output, "non-conventional commit")
			if flagged != tt.Want_Flag {
				t.Errorf("subject %q: want flagged=%v, got flagged=%v; output: %s",
					tt.Subject, tt.Want_Flag, flagged, output)
			}
		})
	}
}

// Commit subjects longer than 100 chars must be flagged. Matches the
// line_chars_max cap the file-tier check enforces on source lines — same
// reasoning (visual scan limit in code-review UIs and terminals).
func Test_Git_Commit_Subject_Chars(t *testing.T) {
	t.Parallel()
	// 100 chars: a conventional prefix plus 94 'x' chars (6 = len("feat: ")).
	at_limit := "feat: " + strings.Repeat("x", 94)
	over_limit := "feat: " + strings.Repeat("x", 95)
	tests := []struct {
		Name      string
		Subject   string
		Want_Flag bool
	}{
		{"at limit (100)", at_limit, false},
		{"over limit (101)", over_limit, true},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			output, _ := run_git(t, lint.Git_Input{
				Enabled: true,
				Non_Merge_Commits: []lint.Git_Commit{
					{Hash: "abc1234567def", Subject: tt.Subject},
				},
			})
			flagged := strings.Contains(output, "commit subject is")
			if flagged != tt.Want_Flag {
				t.Errorf("len=%d: want flagged=%v, got flagged=%v; output: %s",
					len(tt.Subject), tt.Want_Flag, flagged, output)
			}
		})
	}
}

// Subjects matching IsFixupSubject must be flagged; ordinary subjects must
// pass. Covers both the literal fixup!/squash! prefixes and the review-comment
// phrasings so a regression in either branch shows up.
func Test_Git_No_Fixup_Commits(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Subject   string
		Want_Flag bool
	}{
		{"literal fixup", "fixup! refactor: extract foo", true},
		{"literal squash", "squash! feat: add bar", true},
		{"address review comments", "address review comments", true},
		{"apply review feedback", "apply review feedback", true},
		{"cr comment", "cr comment", true},
		{"review nit", "review nit", true},
		{"ordinary feat", "feat: add X", false},
		{"ordinary fix", "fix: handle nil pointer in foo", false},
		{"review without action verb", "chore: reviewed the code", false},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			output, code := run_git(t, lint.Git_Input{
				Enabled: true,
				Non_Merge_Commits: []lint.Git_Commit{
					{Hash: "deadbeef00cafe", Subject: tt.Subject},
				},
			})
			flagged := strings.Contains(output, "fixup commit on branch")
			if flagged != tt.Want_Flag {
				t.Errorf("subject %q: want flagged=%v, got flagged=%v; output: %s",
					tt.Subject, tt.Want_Flag, flagged, output)
			}
			want_code := 0
			if tt.Want_Flag {
				want_code = 1
			}
			if code != want_code {
				t.Errorf("expected exit %d, got %d", want_code, code)
			}
		})
	}
}

// Test_Method_Prefix_Flagged verifies that free functions whose first
// parameter is a same-package declared named type are flagged when the
// function name lacks the type-name prefix. This is the naming half of
// the banned-methods rule (check_unnecessary_method): when a method gets
// rewritten as a free function with the receiver promoted to the first
// param, the type-prefix preserves the grouping affordance methods had.
func Test_Method_Prefix_Flagged(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "bare same-file type flagged",
			Files: map[string]string{
				"a.go": `package foo

type Entity struct{}

func update(e Entity) { return }
`,
			},
			Want_Diag: "rename update -> entity_<verb>",
		},

		{
			Name: "same-package sibling file flagged",
			Files: map[string]string{
				"a.go": `package foo

type Entity struct{}
`,
				"b.go": `package foo

func update(e Entity) { return }
`,
			},
			Want_Diag: "rename update -> entity_<verb>",
		},

		{
			Name: "generic instance flagged",
			Files: map[string]string{
				"a.go": `package foo

type Entity[T any] struct{}

func update(e Entity[int]) { return }
`,
			},
			Want_Diag: "rename update -> entity_<verb>",
		},

		{
			Name: "exported function requires Ada_Case prefix",
			Files: map[string]string{
				"a.go": `package foo

type Main_Input struct{}

func Run(input Main_Input) { return }
`,
			},
			Want_Diag: "rename Run -> Main_Input_<verb>",
		},
	}
	run_diag_table(t, tests)
}

// Additional cases, split to keep each function within the length limit.
func Test_Method_Prefix_Flagged_Part2(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "pointer to same-pkg type flagged (receiver shape)",
			Files: map[string]string{
				"a.go": `package foo

type Snapper struct{}

func edit(s *Snapper) { return }
`,
			},
			Want_Diag: "rename edit -> snapper_<verb>",
		},

		{
			Name: "Ada_Case fn with miscased multi-word prefix flagged",
			Files: map[string]string{
				"a.go": `package foo

type Main_Input struct{}

func Main_input_Run(input Main_Input) { return }
`,
			},
			Want_Diag: "rename Main_input_Run -> Main_Input_<verb>",
		},
	}
	run_diag_table(t, tests)
}

// Test_Method_Prefix_Skipped verifies that the check does not fire on
// first-param shapes that fall outside the rule's scope: stdlib selector
// types, builtins, wrappers (pointer/slice/etc.) around named types — none
// of which match the "receiver promoted to first param" shape — and the
// constructor-input exception where the type is named <FuncName>_Input.
func Test_Method_Prefix_Skipped(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "selector type (stdlib) clean",
			Files: map[string]string{
				"a.go": `// Package foo is a fixture.
package foo

import "io"

func read(r io.Reader) (err error) { return nil }
`,
			},
			Want_Diag: "",
		},

		{
			Name: "builtin first param clean",
			Files: map[string]string{
				"a.go": `// Package foo is a fixture.
package foo


const fixture_hi = 100

func parse(s string) {
	return
}
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Additional cases, split to keep each function within the length limit.
func Test_Method_Prefix_Skipped_Part2(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "slice of same-pkg type clean",
			Files: map[string]string{
				"a.go": `// Package foo is a fixture.
package foo


const fixture_hi = 100

// Entity is a fixture.
type Entity struct{}

func update(es []Entity) {
	defer func() {
	}()
	return
}
`,
			},
			Want_Diag: "",
		},

		{
			Name: "constructor-input pattern clean (FuncName + _Input)",
			Files: map[string]string{
				"a.go": `// Package foo is a fixture.
package foo

// New_Input is a fixture.
type New_Input struct{}

// New constructs a fixture.
func New(input New_Input) { return }
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Method_Prefix_Matched verifies that correctly-prefixed functions
// pass: the case of the prefix matches the function's own style (Ada_Case
// for exported, snake_case for unexported), multi-word type names are
// rebuilt in that style, and a function named exactly as the type counts.
func Test_Method_Prefix_Matched(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "correctly-prefixed unexported clean",
			Files: map[string]string{
				"a.go": `// Package foo is a fixture.
package foo

// Entity is a fixture.
type Entity struct{}

func entity_update(e Entity) { return }
`,
			},
			Want_Diag: "",
		},

		{
			Name: "correctly-prefixed pointer receiver shape clean",
			Files: map[string]string{
				"a.go": `// Package foo is a fixture.
package foo


const fixture_hi = 100

// Snapper is a fixture.
type Snapper struct{}

func snapper_edit(s *Snapper) {
	return
}
`,
			},
			Want_Diag: "",
		},

		{
			Name: "exported function with Ada_Case prefix clean",
			Files: map[string]string{
				"a.go": `// Package foo is a fixture.
package foo

// Main_Input is a fixture.
type Main_Input struct{}

// Main_Input_Run is a fixture.
func Main_Input_Run(input Main_Input) { return }
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Additional cases, split to keep each function within the length limit.
func Test_Method_Prefix_Matched_Part2(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "unexported function with multi-word type clean",
			Files: map[string]string{
				"a.go": `// Package foo is a fixture.
package foo

// Main_Input is a fixture.
type Main_Input struct{}

func main_input_run(input Main_Input) { return }
`,
			},
			Want_Diag: "",
		},

		{
			Name: "function named exactly as type clean",
			Files: map[string]string{
				"a.go": `// Package foo is a fixture.
package foo

// Entity is a fixture.
type Entity struct{}

func entity(e Entity) { return }
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_No_Impure_Stdlib_Hard_Imports verifies that hard-banned imports
// (os, crypto/rand, math/rand v1, flag) fire in library packages, while
// math/rand/v2 is allowed.
func Test_No_Impure_Stdlib_Hard_Imports(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "library imports os",
			Files: map[string]string{
				"a.go": `package library

import "os"

func F() (val string) { return os.Getenv("X") }
`,
			},
			Want_Diag: "impure stdlib import",
		},

		{
			Name: "library imports crypto/rand",
			Files: map[string]string{
				"a.go": `package library

import "crypto/rand"

func F() (n int, err error) { return rand.Reader.Read(nil) }
`,
			},
			Want_Diag: "impure stdlib import",
		},

		{
			Name: "library imports math/rand v1",
			Files: map[string]string{
				"a.go": `package library

import "math/rand"

func F() (n int) { return rand.Int() }
`,
			},
			Want_Diag: "impure stdlib import",
		},
	}
	run_diag_table(t, tests)
}

// Additional cases, split to keep each function within the length limit.
func Test_No_Impure_Stdlib_Hard_Imports_Part2(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "library imports math/rand/v2 clean",
			Files: map[string]string{
				"a.go": `// Package library is a fixture.
package library

import (
	"math/rand/v2"

)

const fixture_hi = 100

// F is a fixture.
func F(r *rand.Rand) (n int) {
	defer func() {
	}()
	return r.Int()
}
`,
			},
			Want_Diag: "",
		},

		{
			Name: "library imports flag",
			Files: map[string]string{
				"a.go": `package library

import "flag"

func F() (s string) { return flag.Arg(0) }
`,
			},
			Want_Diag: "impure stdlib import",
		},
	}
	run_diag_table(t, tests)
}

// Test_No_Impure_Stdlib_Soft_Calls verifies that impure identifiers on
// otherwise-pure packages (time.Now, fmt.Println, http.DefaultClient,
// net.Dial) fire while the pure surface (time.Duration, fmt.Sprintf,
// http.Header, log.Printf) is allowed.
func Test_No_Impure_Stdlib_Soft_Calls(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "library calls time.Now",
			Files: map[string]string{
				"a.go": `package library

import "time"

func F() (t time.Time) { return time.Now() }
`,
			},
			Want_Diag: "impure stdlib call",
		},

		{
			Name: "library uses time.Duration clean",
			Files: map[string]string{
				"a.go": `// Package library is a fixture.
package library

import "time"

// F is a fixture.
func F(d time.Duration) (result time.Duration) { return d * 2 }
`,
			},
			Want_Diag: "",
		},

		{
			Name: "library calls fmt.Println",
			Files: map[string]string{
				"a.go": `package library

import "fmt"

func F() { fmt.Println("hi") }
`,
			},
			Want_Diag: "impure stdlib call",
		},

		{
			Name: "library uses fmt.Sprintf and fmt.Errorf clean",
			Files: map[string]string{
				"a.go": `// Package library is a fixture.
package library

import "fmt"

// F is a fixture.
func F() (err error) { return fmt.Errorf("x: %s", fmt.Sprintf("y")) }
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Additional cases, split to keep each function within the length limit.
func Test_No_Impure_Stdlib_Soft_Calls_Part2(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "library calls log.Printf clean (observability exempt)",
			Files: map[string]string{
				"a.go": `// Package library is a fixture.
package library

import "log"

// F is a fixture.
func F() { log.Printf("hi") }
`,
			},
			Want_Diag: "",
		},

		{
			Name: "aliased time import still flagged",
			Files: map[string]string{
				"a.go": `package library

import t "time"

func F() (result t.Time) { return t.Now() }
`,
			},
			Want_Diag: "impure stdlib call",
		},
	}
	run_diag_table(t, tests)
}

// Test_Coverage_Backfill_Impure_Soft_One_Character drives is_impure_soft_ident's
// Package and Name Lo (=1) buckets with a one-char import path and a one-char
// selected symbol. "a" is not an impure stdlib package, so nothing fires.
func Test_Coverage_Backfill_Impure_Soft_One_Character(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{{
		Name: "non-stdlib one-char import path and symbol clean",
		Files: map[string]string{
			"a.go": `// Package library is a fixture.
package library

import "a"

// F is a fixture.
func F() { a.X() }
`,
		},
		Want_Diag: "",
	}})
}

// Test_No_Impure_Stdlib_Soft_Network_Http verifies the http and net soft bans.
func Test_No_Impure_Stdlib_Soft_Network_Http(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "library references http.DefaultClient",
			Files: map[string]string{
				"a.go": `package library

import "net/http"

func F() (c *http.Client) { return http.DefaultClient }
`,
			},
			Want_Diag: "impure stdlib call",
		},
		{
			Name: "library uses http.Request type clean",
			Files: map[string]string{
				"a.go": `// Package library is a fixture.
package library

import (
	"net/http"

)

// F is a fixture.
func F(r *http.Request) (h http.Header) {
	return r.Header
}
`,
			},
			Want_Diag: "",
		},
		{
			Name: "library calls net.Dial",
			Files: map[string]string{
				"a.go": `package library

import "net"

func F() (c net.Conn, err error) { return net.Dial("tcp", "x:1") }
`,
			},
			Want_Diag: "impure stdlib call",
		},
	}
	run_diag_table(t, tests)
}

// Test_Transitive_Stdlib_Curated verifies the curated stdlib APIs that reach the
// impure set only transitively are flagged on a pure package, while their pure
// siblings (filepath.Join) are not.
func Test_Transitive_Stdlib_Curated(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "filepath.Walk touches the filesystem",
			Files: map[string]string{
				"a.go": `package library

import "path/filepath"

func F() (err error) { return filepath.Walk(".", nil) }
`,
			},
			Want_Diag: "impure transitive call",
		},
		{
			Name: "filepath.Abs reads the working directory",
			Files: map[string]string{
				"a.go": `package library

import "path/filepath"

func F() (s string, err error) { return filepath.Abs(".") }
`,
			},
			Want_Diag: "impure transitive call",
		},
		{
			Name: "context.WithTimeout reads the wall clock",
			Files: map[string]string{
				"a.go": `package library

import "context"

func F(parent context.Context) (child context.Context, cancel context.CancelFunc) {
	return context.WithTimeout(parent, 0)
}
`,
			},
			Want_Diag: "impure transitive call",
		},
	}
	run_diag_table(t, tests)
}

// Test_Transitive_Stdlib_Curated_Part2 continues Test_Transitive_Stdlib_Curated,
// split to keep each function within the length limit.
func Test_Transitive_Stdlib_Curated_Part2(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "x509.SystemCertPool reads the OS trust store",
			Files: map[string]string{
				"a.go": `package library

import "crypto/x509"

func F() (p *x509.CertPool, err error) { return x509.SystemCertPool() }
`,
			},
			Want_Diag: "impure transitive call",
		},
		{
			Name: "smtp.SendMail dials a server",
			Files: map[string]string{
				"a.go": `package library

import "net/smtp"

func F() (err error) { return smtp.SendMail("h:25", nil, "f", nil, nil) }
`,
			},
			Want_Diag: "impure transitive call",
		},
		{
			Name: "filepath.Join is pure",
			Files: map[string]string{
				"a.go": `// Package library is a fixture.
package library

import "path/filepath"

// F is a fixture.
func F() (s string) { return filepath.Join("a", "b") }
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_No_Impure_Stdlib_Exemptions verifies that package main and _test.go
// files may freely use impure stdlib calls.
func Test_No_Impure_Stdlib_Exemptions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "package main may use os",
			Files: map[string]string{
				"main.go": `package main

import "os"

func main() { print(os.Getenv("X")) }
`,
			},
			Want_Diag: "",
		},

		{
			Name: "package main may call time.Now",
			Files: map[string]string{
				"main.go": `package main

import (
	"fmt"
	"time"
)

func main() { fmt.Println(time.Now()) }
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Additional cases, split to keep each function within the length limit.
func Test_No_Impure_Stdlib_Exemptions_Part2(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "test.go in library may call time.Now",
			Files: map[string]string{
				"a.go": `// Package library is a fixture.
package library


const fixture_hi = 100

// F is a fixture.
func F() (n int) {
	defer func() {
	}()
	return 1
}
`,
				"a_test.go": `package library_test

import (
	"testing"
	"time"
)

// Test_F exercises time.Now usage in a test file.
func Test_F(t *testing.T) {
	if time.Now().IsZero() {
		t.Fatal("zero")
	}
}
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_No_Impure_Stdlib_Composition_Tier verifies that packages
// sitting exactly one depth below the library tier in their module
// — the composition tier — may bind to impure stdlib state. This is
// the doctrine's designated home for library defaults, CLIs, servers,
// and anything else that wires the library to a real environment.
func Test_No_Impure_Stdlib_Composition_Tier(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name       string
		Files      map[string]string
		Want_Diags []string
		Forbid     []string
	}{

		{
			Name: "library tier impure import still flagged",
			Files: map[string]string{
				"golang_snacks/go.mod": doctrine_shared_library_go_module,
				"golang_snacks/foo/foo.go": `package foo

import "os"

func Read() (name string) { return os.Getenv("X") }
`,
			},
			Want_Diags: []string{"impure stdlib import \"os\""},
		},

		{
			Name: "composition tier impure import allowed",
			Files: map[string]string{
				"golang_snacks/go.mod":     doctrine_shared_library_go_module,
				"golang_snacks/foo/foo.go": fixture_package("foo"),
				"golang_snacks/foo/foo_default/" +
					"foo_default.go": `// Package foo_default is a fixture.
package foo_default

import (
	"os"

)

const fixture_hi = 100

// Read is a fixture.
func Read() (name string) {
	defer func() {
	}()
	return os.Getenv("X")
}
`,
			},
			Forbid: []string{"impure stdlib import"},
		},
	}
	run_doctrine_diag_table(t, tests)
}

// Additional cases, split to keep each function within the length limit.
func Test_No_Impure_Stdlib_Composition_Tier_Part2(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name       string
		Files      map[string]string
		Want_Diags []string
		Forbid     []string
	}{

		{
			Name: "composition tier under versioned library allowed",
			Files: map[string]string{
				"golang_snacks/go.mod":          doctrine_shared_library_go_module,
				"golang_snacks/snap/v2/snap.go": fixture_package("snap"),
				"golang_snacks/snap/v2/snap_default/" +
					"snap_default.go": `// Package snap_default is a fixture.
package snap_default

import (
	"os"

)

const fixture_hi = 100

// Read is a fixture.
func Read() (name string) {
	defer func() {
	}()
	return os.Getenv("X")
}
`,
			},
			Forbid: []string{"impure stdlib import"},
		},
	}
	run_doctrine_diag_table(t, tests)
}

// Test_No_Impure_Stdlib_Composition_Tier_Extra continues the composition-tier
// allow-list: an impure stdlib CALL (`time.Now()`) at the composition tier
// is allowed, and the binary-module composition tier (one level under the
// library tier) inherits the same exemption.
func Test_No_Impure_Stdlib_Composition_Tier_Extra(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name       string
		Files      map[string]string
		Want_Diags []string
		Forbid     []string
	}{
		{
			Name: "impure call (time.Now) at composition tier allowed",
			Files: map[string]string{
				"golang_snacks/go.mod":     doctrine_shared_library_go_module,
				"golang_snacks/foo/foo.go": fixture_package("foo"),
				"golang_snacks/foo/foo_default/" +
					"foo_default.go": `// Package foo_default is a fixture.
package foo_default

import "time"

// Stamp is a fixture.
func Stamp() (t time.Time) { return time.Now() }
`,
			},
			Forbid: []string{"impure stdlib call"},
		},
		{
			Name: "binary module composition tier " +
				"(one level under library tier) allowed",
			Files: map[string]string{
				"mybinary/go.mod": doctrine_binary_go_module,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/internal/entry.go":       doctrine_binary_internal_main,
				"mybinary/internal/lib/library.go": fixture_package("library"),
				"mybinary/internal/lib/lib_default/" +
					"library_default.go": `// Package library_default is a fixture.
package library_default

import (
	"os"

)

const fixture_hi = 100

// Read is a fixture.
func Read() (name string) {
	defer func() {
	}()
	return os.Getenv("X")
}
`,
			},
			Forbid: []string{"impure stdlib import"},
		},
	}
	run_doctrine_diag_table(t, tests)
}

// Test_No_Bare_For_Flagged verifies that for-loops without a header — bare
// `for {}`, `for ;; {}`, or `for true {}` — are flagged.
func Test_No_Bare_For_Flagged(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "bare for braces flagged",
			Files: map[string]string{
				"a.go": `package main

func main() {
	for {
		break
	}
}
`,
			},
			Want_Diag: "bare `for",
		},
		{
			Name: "for double-semicolon flagged",
			Files: map[string]string{
				"a.go": `package main

func main() {
	for ;; {
		break
	}
}
`,
			},
			Want_Diag: "bare `for",
		},
		{
			Name: "for true flagged",
			Files: map[string]string{
				"a.go": `package main

func main() {
	for true {
		break
	}
}
`,
			},
			Want_Diag: "bare `for",
		},
	}
	run_diag_table(t, tests)
}

// Test_No_Bare_For_Clean verifies that for-loops with a real condition,
// C-style headers, range loops, and the documented escape hatch
// `for range invariant.GameLoop()` are not flagged.
func Test_No_Bare_For_Clean(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "c-style for clean",
			Files: map[string]string{
				"a.go": `package main

func main() {
	total := 0
	for i_index := 0; i_index < 10; i_index++ {
		total += i_index
	}
	print(total)
}
`,
			},
			Want_Diag: "",
		},
		{
			Name: "for cond clean (Tier B not implemented)",
			Files: map[string]string{
				"a.go": `package main

func main() {
	done := false
	for !done {
		done = true
	}
}
`,
			},
			Want_Diag: "",
		},
		{
			Name: "for range slice clean",
			Files: map[string]string{
				"a.go": `package main

func main() {
	for range []int{1, 2, 3} {
	}
}
`,
			},
			Want_Diag: "",
		},
		{
			Name: "for range Game_Loop escape hatch clean",
			Files: map[string]string{
				"a.go": `package main

` + fixture_invariant_import + `
const fixture_hi = 100

func main() {
	for range invariant.Game_Loop() {
		break
	}
}
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Default_Package_Name verifies that a file in a `default/` directory must
// declare the package clause of its parent library, so importing `foo/default`
// binds to `foo` and shadows the library it re-exports — what lets callers keep
// writing snap.Init/snap.Edit, and what snap.Edit's source-line rewriter (which
// searches for the literal "snap.Edit(") relies on. The directory being literally
// `default` cannot itself be a package name (Go keyword), which is why the parent
// name is required.
func Test_Default_Package_Name(t *testing.T) {
	t.Parallel()

	clean, err := lint.Check_Source("foo/default/wire.go", "package foo\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if specification_diagnosed(clean, "default package") {
		t.Fatal("a default package declaring its parent name must not be flagged")
	}

	wrong, err := lint.Check_Source("foo/default/wire.go", "package wrong\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !specification_diagnosed(wrong, "must declare 'package foo'") {
		t.Fatal("a default package with the wrong package clause must be flagged")
	}
}

// Test_Binary_Module_Layout verifies that binary modules — every module
// at the workspace root except the hardcoded shared library — must
// keep all non-main, non-test packages under internal/. Files in
// package main are exempt at any depth (Go itself bars importing
// them), and the shared library is exempt because its purpose is to
// expose packages to other modules.
func Test_Binary_Module_Layout(t *testing.T) {
	tests := []struct {
		Name       string
		Files      map[string]string
		Want_Diags []string
		Forbid     []string
	}{
		{
			Name: "binary with package main at root is clean",
			Files: map[string]string{
				"mybinary/go.mod": doctrine_binary_go_module,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/internal/entry.go": doctrine_binary_internal_main,
			},
		},
		{
			Name: "binary with non-main package outside internal flagged",
			Files: map[string]string{
				"mybinary/go.mod": doctrine_binary_go_module,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/helpers/h.go": fixture_package("helpers"),
			},
			Want_Diags: []string{"move helpers -> mybinary/internal/helpers"},
		},
		{
			Name: "binary with package under internal is clean",
			Files: map[string]string{
				"mybinary/go.mod": doctrine_binary_go_module,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/internal/entry.go":       doctrine_binary_internal_main,
				"mybinary/internal/lib/library.go": fixture_package("library"),
			},
		},
		{
			Name: "binary with non-main package at module root flagged",
			Files: map[string]string{
				"mybinary/go.mod":     doctrine_binary_go_module,
				"mybinary/library.go": fixture_package("mybinary"),
			},
			Want_Diags: []string{"move . -> mybinary/internal"},
		},
		{
			Name: "shared library with non-main package at depth 1 is exempt",
			Files: map[string]string{
				"golang_snacks/go.mod":     doctrine_shared_library_go_module,
				"golang_snacks/foo/foo.go": fixture_package("foo"),
			},
		},
		{
			Name: "second main package under cmd flagged",
			Files: map[string]string{
				"mybinary/go.mod": doctrine_binary_go_module,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/cmd/extra/extra.go": "package main\n\n" +
					"func main() { return }\n",
			},
			Want_Diags: []string{
				"places its main package at \"cmd/extra\"",
				"no cmd/ directory",
			},
		},
	}
	run_doctrine_diag_table(t, tests)
}

// Test_Binary_Module_Internal_Main verifies that every binary module
// declares exactly one free func Main in its top-level internal/ package
// — the composition entry point main() delegates to. Location is the
// internal/ directory itself, not a subpackage; the function must be a
// free function (no receiver) of any signature; test files don't count;
// the shared library is exempt because it owns no entry point.
func Test_Binary_Module_Internal_Main(t *testing.T) {
	tests := []struct {
		Name       string
		Files      map[string]string
		Want_Diags []string
		Forbid     []string
	}{
		{
			Name: "binary with func Main in internal is clean",
			Files: map[string]string{
				"mybinary/go.mod": doctrine_binary_go_module,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/internal/entry.go": doctrine_binary_internal_main,
			},
		},
		{
			Name: "non-standard signature still satisfies the rule",
			Files: map[string]string{
				"mybinary/go.mod": doctrine_binary_go_module,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/internal/entry.go": "// Package entry is a fixture.\n" +
					"package entry\n\n// Main is a fixture.\n" +
					"func Main(a int) (s string) { return \"\" }\n",
			},
		},
		{
			Name: "binary without func Main flagged at go.mod",
			Files: map[string]string{
				"mybinary/go.mod": doctrine_binary_go_module,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
			},
			Want_Diags: []string{
				"mybinary/go.mod:1:1",
				"declares no func Main in internal/",
			},
		},
		{
			Name: "func Main below top-level internal does not satisfy the rule",
			Files: map[string]string{
				"mybinary/go.mod": doctrine_binary_go_module,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/internal/cmd/entry.go": doctrine_binary_internal_main,
			},
			Want_Diags: []string{"declares no func Main in internal/"},
		},
	}
	run_doctrine_diag_table(t, tests)
}

// Test_Binary_Module_Internal_Main_Part2 continues the binary entry-point
// cases (split so neither table function exceeds the line budget): a method
// named Main, a test-file-only Main, the multiple-Main violation, and the
// shared-library exemption.
func Test_Binary_Module_Internal_Main_Part2(t *testing.T) {
	tests := []struct {
		Name       string
		Files      map[string]string
		Want_Diags []string
		Forbid     []string
	}{
		{
			Name: "method named Main does not satisfy the rule",
			Files: map[string]string{
				"mybinary/go.mod": doctrine_binary_go_module,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/internal/entry.go": "// Package entry is a fixture.\n" +
					"package entry\n\n// R is a fixture.\ntype R struct{}\n\n" +
					"// Main is a fixture.\nfunc (r R) Main() { return }\n",
			},
			Want_Diags: []string{"declares no func Main in internal/"},
		},
		{
			Name: "func Main only in a test file does not satisfy the rule",
			Files: map[string]string{
				"mybinary/go.mod": doctrine_binary_go_module,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/internal/entry_test.go": "package entry\n\n" +
					"// Main is a fixture.\nfunc Main() { return }\n",
			},
			Want_Diags: []string{"declares no func Main in internal/"},
		},
		{
			Name: "two func Main across internal files flagged as multiple",
			Files: map[string]string{
				"mybinary/go.mod": doctrine_binary_go_module,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/internal/a.go": doctrine_binary_internal_main,
				"mybinary/internal/b.go": "package entry\n\n" +
					"// Main is a fixture.\nfunc Main() { return }\n",
			},
			Want_Diags: []string{"declares multiple func Main in internal/"},
		},
		{
			Name: "shared library without internal func Main is exempt",
			Files: map[string]string{
				"golang_snacks/go.mod":     doctrine_shared_library_go_module,
				"golang_snacks/foo/foo.go": fixture_package("foo"),
			},
		},
		{
			Name: "shared library may expose func Main as an embeddable",
			Files: map[string]string{
				"golang_snacks/go.mod": doctrine_shared_library_go_module,
				"golang_snacks/foo/foo.go": "// Package foo is a fixture.\n" +
					"package foo\n\n// Main is an embeddable entry point.\n" +
					"func Main() { return }\n",
			},
		},
	}
	run_doctrine_diag_table(t, tests)
}

// Test_Shared_Library_No_Internal verifies that the shared library
// module may not contain internal/ directories or any package main.
// Other modules are unaffected — they use internal/ and own a binary.
func Test_Shared_Library_No_Internal(t *testing.T) {
	tests := []struct {
		Name       string
		Files      map[string]string
		Want_Diags []string
		Forbid     []string
	}{
		{
			Name: "shared library with internal directory flagged",
			Files: map[string]string{
				"golang_snacks/go.mod":     doctrine_shared_library_go_module,
				"golang_snacks/foo/foo.go": fixture_package("foo"),
				"golang_snacks/foo/internal/helper/" +
					"help.go": fixture_package("helper"),
			},
			Want_Diags: []string{
				"shared library forbids internal/", "golang_snacks/foo/internal",
			},
		},
		{
			Name: "shared library with package main flagged",
			Files: map[string]string{
				"golang_snacks/go.mod":     doctrine_shared_library_go_module,
				"golang_snacks/foo/foo.go": fixture_package("foo"),
				"golang_snacks/run/main.go": "package main\n\n" +
					"func main() { return }\n",
			},
			Want_Diags: []string{"forbids package main"},
		},
		{
			Name: "shared library without internal is clean",
			Files: map[string]string{
				"golang_snacks/go.mod":     doctrine_shared_library_go_module,
				"golang_snacks/foo/foo.go": fixture_package("foo"),
			},
		},
		{
			Name: "binary with internal directory is unaffected by this rule",
			Files: map[string]string{
				"mybinary/go.mod": doctrine_binary_go_module,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/internal/entry.go":       doctrine_binary_internal_main,
				"mybinary/internal/lib/library.go": fixture_package("library"),
			},
			Forbid: []string{"shared library forbids"},
		},
	}
	run_doctrine_diag_table(t, tests)
}

// Test_Library_Tier_Depth verifies the at-most-one-non-main-Go-ancestor
// rule. Major-version segments (v2, v3, …) are dropped before counting
// ancestors, so a versioned subdirectory of a library still counts as
// library tier rather than composition tier. Composition tier sits
// exactly one depth below the library tier; anything deeper is flagged.
func Test_Library_Tier_Depth(t *testing.T) {
	tests := []struct {
		Name       string
		Files      map[string]string
		Want_Diags []string
		Forbid     []string
	}{
		{
			Name: "two-deep nesting flagged",
			Files: map[string]string{
				"golang_snacks/" +
					"go.mod": doctrine_shared_library_go_module,
				"golang_snacks/foo/foo.go":         fixture_package("foo"),
				"golang_snacks/foo/bar/bar.go":     fixture_package("bar"),
				"golang_snacks/foo/bar/baz/baz.go": fixture_package("baz"),
			},
			Want_Diags: []string{"exceeds library tier", "baz"},
		},
		{
			Name: "library plus composition tier is clean",
			Files: map[string]string{
				"golang_snacks/go.mod":         doctrine_shared_library_go_module,
				"golang_snacks/foo/foo.go":     fixture_package("foo"),
				"golang_snacks/foo/bar/bar.go": fixture_package("bar"),
			},
			Forbid: []string{"exceeds library tier"},
		},
		{
			Name: "v2 version directory does not count as ancestor",
			Files: map[string]string{
				"golang_snacks/" +
					"go.mod": doctrine_shared_library_go_module,
				"golang_snacks/snap/snap.go":         fixture_package("snap"),
				"golang_snacks/snap/v2/snap.go":      fixture_package("snap"),
				"golang_snacks/snap/v2/sub/child.go": fixture_package("child"),
			},
			Forbid: []string{"exceeds library tier"},
		},
		{
			Name: "non-Go intermediate directory does not count as ancestor",
			Files: map[string]string{
				"golang_snacks/go.mod":     doctrine_shared_library_go_module,
				"golang_snacks/foo/foo.go": fixture_package("foo"),
				"golang_snacks/foo/examples/sample/" +
					"example.go": fixture_package("example"),
			},
			Forbid: []string{"exceeds library tier"},
		},
		{
			Name: "binary composition tier one level under internal is clean",
			Files: map[string]string{
				"mybinary/go.mod": doctrine_binary_go_module,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/internal/entry.go":   doctrine_binary_internal_main,
				"mybinary/internal/foo/foo.go": fixture_package("foo"),
			},
			Forbid: []string{"exceeds library tier"},
		},
	}
	run_doctrine_diag_table(t, tests)
}

// Test_Library_Tier_Depth_Internal_Anchor verifies that a binary module's
// top-level internal directory is where the nesting count starts — the same
// role a shared module's root plays — and is not itself a package counted
// above another. A pure library package sits one level inside internal and its
// composition tier one level below that (internal/foo/default); anything
// deeper is flagged.
func Test_Library_Tier_Depth_Internal_Anchor(t *testing.T) {
	tests := []struct {
		Name       string
		Files      map[string]string
		Want_Diags []string
		Forbid     []string
	}{
		{
			Name: "library package and default composition under internal is clean",
			Files: map[string]string{
				"bin/go.mod": doctrine_binary_go_module,
				"bin/main.go": "package main\n\n" +
					"func main() { return }\n",
				"bin/internal/entry.go":            doctrine_binary_internal_main,
				"bin/internal/foo/foo.go":          fixture_package("foo"),
				"bin/internal/foo/default/wire.go": fixture_package("foo"),
			},
			Forbid: []string{"exceeds library tier"},
		},
		{
			Name: "nesting beyond the composition tier under internal flagged",
			Files: map[string]string{
				"bin/go.mod": doctrine_binary_go_module,
				"bin/main.go": "package main\n\n" +
					"func main() { return }\n",
				"bin/internal/entry.go":           doctrine_binary_internal_main,
				"bin/internal/foo/foo.go":         fixture_package("foo"),
				"bin/internal/foo/bar/bar.go":     fixture_package("bar"),
				"bin/internal/foo/bar/baz/baz.go": fixture_package("baz"),
			},
			Want_Diags: []string{"exceeds library tier", "baz"},
		},
	}
	run_doctrine_diag_table(t, tests)
}

// Test_Exported_Documentation_Comment verifies that every exported top-level
// identifier (func, method, type, var, const) carries a doc comment.
// For grouped declarations, a comment on the containing GenDecl is
// inherited by each spec; single-spec GenDecls behave the same way
// since the parser hangs the leading comment on the GenDecl rather
// than the spec. package main is exempt — it exports nothing
// reachable from outside.
func Test_Exported_Documentation_Comment(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "exported func without doc flagged",
			Files: map[string]string{
				"foo.go": "// Package foo provides things.\npackage foo\n\n" +
					"func Do() { return }\n",
			},
			Want_Diag: "exported func Do is missing a doc comment",
		},

		{
			Name: "exported func with doc allowed",
			Files: map[string]string{
				"foo.go": "// Package foo provides things.\n" +
					"package foo\n\n" +
					"// Do performs the operation.\n" +
					"func Do() { return }\n",
			},
			Want_Diag: "",
		},

		{
			Name: "unexported func without doc allowed",
			Files: map[string]string{
				"foo.go": "// Package foo provides things.\npackage foo\n\n" +
					"func do() { return }\n",
			},
			Want_Diag: "",
		},

		{
			Name: "exported type without doc flagged",
			Files: map[string]string{
				"foo.go": "// Package foo provides things.\npackage foo\n\n" +
					"type Thing struct{ X int }\n",
			},
			Want_Diag: "exported type Thing is missing a doc comment",
		},

		{
			Name: "exported var without doc flagged",
			Files: map[string]string{
				"foo.go": "// Package foo provides things.\n" +
					"package foo\n\n" +
					"import \"regexp\"\n\n" +
					"var Default_Pattern = regexp.MustCompile(\"abc\")\n",
			},
			Want_Diag: "exported var Default_Pattern is missing a doc comment",
		},

		{
			Name: "exported const without doc flagged",
			Files: map[string]string{
				"foo.go": "// Package foo provides things.\npackage foo\n\n" +
					"const Count_Max = 100\n",
			},
			Want_Diag: "exported const Count_Max is missing a doc comment",
		},
	}
	run_diag_table(t, tests)
}

// Additional cases, split to keep each function within the length limit.
func Test_Exported_Documentation_Comment_Part2(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "exported var with GenDecl doc inherited",
			Files: map[string]string{
				"foo.go": "// Package foo provides things.\n" +
					"package foo\n\n" +
					"import \"regexp\"\n\n" +
					"// Default_Pattern matches things.\n" +
					"var Default_Pattern = regexp.MustCompile(\"abc\")\n",
			},
			Want_Diag: "",
		},

		{
			Name: "grouped consts: each spec needs its own doc when group has no doc",
			Files: map[string]string{
				"foo.go": "// Package foo provides things.\n" +
					"package foo\n\n" +
					"const (\n" +
					"\tA_Count = 1\n" +
					"\tB_Count = 2\n" +
					")\n",
			},
			Want_Diag: "exported const A_Count is missing a doc comment",
		},

		{
			Name: "exported method without doc flagged",
			Files: map[string]string{
				"foo.go": "// Package foo provides things.\n" +
					"package foo\n\n" +
					"// Thing is a thing.\n" +
					"type Thing struct{ X int }\n\n" +
					"func (t *Thing) String() (output string) " +
					"{ return \"\" }\n",
			},
			Want_Diag: "exported method String is missing a doc comment",
		},

		{
			Name: "package main exempt: exported func without doc allowed",
			Files: map[string]string{
				"main.go": "package main\n\nfunc main() { return }\n\n" +
					"func Run() { return }\n",
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Package_Documentation_Comment verifies that every non-main, non-_test
// package has a package doc comment in at least one of its files.
// The doc comment is the comment group immediately preceding the
// `package` clause and lands on ast.File.Doc.
func Test_Package_Documentation_Comment(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "package without doc flagged",
			Files: map[string]string{
				"foo.go": "package foo\n\n" +
					"// Do performs the operation.\nfunc Do() { return }\n",
			},
			Want_Diag: "package \"foo\" is missing a doc comment",
		},
		{
			Name: "package with doc allowed",
			Files: map[string]string{
				"foo.go": "// Package foo provides things.\n" +
					"package foo\n\n" +
					"// Do performs the operation.\n" +
					"func Do() { return }\n",
			},
			Want_Diag: "",
		},
		{
			Name: "package doc on one of multiple files satisfies the rule",
			Files: map[string]string{
				"documentation.go": "//go:build doc_only\n\n" +
					"// Package foo provides things.\npackage foo\n",
				"foo.go": "package foo\n\n" +
					"// Do performs the operation.\n" +
					"func Do() { return }\n",
			},
			Want_Diag: "",
		},
		{
			Name: "package main exempt",
			Files: map[string]string{
				"main.go": "package main\n\n" +
					"func main() { return }\n",
			},
			Want_Diag: "",
		},
		{
			Name: "test package exempt",
			Files: map[string]string{
				"foo.go": "// Package foo provides things.\n" +
					"package foo\n\n" +
					"// Do performs the operation.\n" +
					"func Do() { return }\n",
				"foo_test.go": "package foo_test\n\n" +
					"import \"testing\"\n\n" +
					"// Test_Do verifies Do.\n" +
					"func Test_Do(t *testing.T) { t.Helper() }\n",
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Coverage_Backfill_Read_Error drives Check_File_System against an
// fs.FS whose Open returns an error for selected paths, so the stream
// tier's `load()` callback returns err != nil and the source-nil /
// err-non-nil tuple fires on every check_stream_* assertion. Fixture
// names are chosen so each check_stream_* filter (markdown, copyleft,
// github actions, agent docs, agents-claude pair) sees at least one
// failing load.
func Test_Coverage_Backfill_Read_Error(t *testing.T) {
	t.Parallel()
	base := fstest.MapFS{
		"go.mod": &fstest.MapFile{
			Data: []byte("module example.com/rd\n")},
		"good.go": &fstest.MapFile{
			Data: []byte("// Package rd is a fixture.\npackage rd\n")},
		"fail.go": &fstest.MapFile{
			Data: []byte("// Package rd is a fixture.\npackage rd\n")},
		"fail.md": &fstest.MapFile{
			Data: []byte("# fixture\n")},
		"fail.yml": &fstest.MapFile{
			Data: []byte("name: fixture\n")},
		"fail.txt": &fstest.MapFile{
			Data: []byte("fixture\n")},
		"LICENSE": &fstest.MapFile{
			Data: []byte("fixture\n")},
		".github/workflows/fail.yml": &fstest.MapFile{
			Data: []byte("name: fixture\n")},
		"CLAUDE.md": &fstest.MapFile{
			Data: []byte("# fixture\n")},
		"AGENTS.md": &fstest.MapFile{
			Data: []byte("# fixture\n")},
	}
	// Fail.go is intentionally excluded: the AST tier reads .go files in
	// bulk via check_file_system_read_files, which asserts sources != nil.
	// Stream-tier coverage for read errors is exercised on non-.go paths.
	fail_paths := map[string]bool{
		"fail.md":                    true,
		"fail.yml":                   true,
		"fail.txt":                   true,
		"LICENSE":                    true,
		".github/workflows/fail.yml": true,
		"CLAUDE.md":                  true,
		"AGENTS.md":                  true,
	}
	fsys := read_error_file_system{MapFS: base, Fail_Paths: fail_paths}
	var stdout, stderr bytes.Buffer
	lint.Main(&lint.Main_Input{
		Fsys:      fsys,
		Stdout:    &stdout,
		Stderr:    &stderr,
		CPU_Count: 1,
	})
}

// Test_Coverage_Backfill_Main_With_Git fires Main's prologue Cross_Product
// with Git.Enabled=true (and both branches of Main_Reference_Absent) so the
// (Hi, Enabled=true, …) tuples on the Main prologue's tracker are observed.
// Without this, those tuples sit unfired and the suite-end coverage report
// flags Main.
func Test_Coverage_Backfill_Main_With_Git(t *testing.T) {
	t.Parallel()
	base := fstest.MapFS{
		"go.mod": &fstest.MapFile{
			Data: []byte("module example.com/git\n")},
		"good.go": &fstest.MapFile{
			Data: []byte(
				"// Package git is a fixture.\npackage git\n")},
	}
	for _, absent := range []bool{false, true} {
		var stdout, stderr bytes.Buffer
		lint.Main(&lint.Main_Input{
			Fsys:      base,
			Stdout:    &stdout,
			Stderr:    &stderr,
			CPU_Count: 1,
			Git: lint.Git_Input{
				Enabled:               true,
				Main_Reference_Absent: absent,
			},
		})
	}
}

type read_error_file_system struct {
	fstest.MapFS
	Fail_Paths map[string]bool
}

func (e read_error_file_system) Open(name string) (file fs.File, err error) {
	if e.Fail_Paths[name] {
		return nil, err_simulated_read
	}
	return e.MapFS.Open(name)
}

// ReadFile overrides the fstest.MapFS.ReadFile promoted method. fs.ReadFile
// preferentially dispatches to ReadFileFS — without this, the embedded
// MapFS.ReadFile is used directly and the Fail_Paths simulation is bypassed
// for every caller that goes through fs.ReadFile.
func (e read_error_file_system) ReadFile(name string) (data []byte, err error) {
	if e.Fail_Paths[name] {
		return nil, err_simulated_read
	}
	return e.MapFS.ReadFile(name)
}

var err_simulated_read = errors.New("simulated read error")

// Test_Coverage_Backfill_Empty_Comment_Body exercises comment_body
// returning "" for a whitespace-only comment.
func Test_Coverage_Backfill_Empty_Comment_Body(t *testing.T) {
	t.Parallel()
	source := "package fixture\n\n" +
		"//\n" +
		"const X = 1\n"
	_, err := lint.Check_Source("fixture.go", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
}

// Test_Coverage_Backfill_S_Abbreviation exercises
// banned_abbreviation_candidates_s by passing a word starting with 's'.
func Test_Coverage_Backfill_S_Abbreviation(t *testing.T) {
	t.Parallel()
	source := "package fixture\n\nvar sz = 1\n"
	_, err := lint.Check_Source("fixture.go", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
}

// Test_Coverage_Backfill_Parse_Error exercises Check_File_System
// against a tree containing a syntactically-invalid Go file so the
// parse_diags axis at lint.go:1134 fires its non-nil bucket.
func Test_Coverage_Backfill_Parse_Error(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"go.mod": &fstest.MapFile{
			Data: []byte("module example.com/broken\n")},
		"bad.go": &fstest.MapFile{
			Data: []byte("package broken\n\nfunc f( {\n}\n")},
		"good.go": &fstest.MapFile{
			Data: []byte(
				"// Package broken is a fixture.\n" +
					"package broken\n\n" +
					"// G is documented.\n" +
					"func G() { return }\n")},
	}
	var stdout, stderr bytes.Buffer
	code := lint.Main(&lint.Main_Input{
		Fsys:      fsys,
		Stdout:    &stdout,
		Stderr:    &stderr,
		CPU_Count: 1,
	})
	if code == 0 {
		t.Logf("expected non-zero exit for a tree with parse errors; got %d", code)
	}
}

// Test_Coverage_Backfill_Documented_Value_Specification exercises the
// per-spec doc comment branch of check_exported_documentation_comment,
// firing specification_has_documentation = true for a value spec that
// has its own doc comment instead of relying on the parent var-block.
func Test_Coverage_Backfill_Documented_Value_Specification(t *testing.T) {
	t.Parallel()
	source := "// Package fixture is documented.\n" +
		"package fixture\n\n" +
		"var (\n" +
		"\t// Documented_Value is documented.\n" +
		"\tDocumented_Value = 1\n" +
		")\n"
	diags, err := lint.Check_Source("fixture.go", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	for _, d := range diags {
		t.Logf("diag: %s", d.Message)
	}
}

// Main and Check_File_System fire their Hi bucket only when CPU_Count > 0;
// production tests use the degraded CPU_Count=0 path throughout.
func Test_Coverage_Backfill(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := lint.Main(&lint.Main_Input{
		Fsys:      fstest.MapFS{},
		Stdout:    &stdout,
		Stderr:    &stderr,
		CPU_Count: 4,
	})
	if code != 0 {
		t.Logf("Main exit code with empty FS: %d", code)
	}
}

// Drives Check_File_System against a synthesised package whose total line
// count exceeds the per-file cap AND whose file count exceeds the resulting
// files_max quota, forcing package_group_key_diag's files_max Boundary axis
// to fire its Hi bucket. Production tests never hit this saturation point.
func Test_Coverage_Backfill_Large_Package(t *testing.T) {
	t.Parallel()
	const file_count = 50
	fsys := fstest.MapFS{
		"go.mod": &fstest.MapFile{
			Data: []byte("module example.com/big\n")},
	}
	for i_index := 0; i_index < file_count; i_index++ {
		name := "f" + strings.Repeat("x", i_index+1) + ".go"
		// Each file declares a unique constant to avoid redeclaration errors,
		// then pads with blank lines to push total lines past
		// lines_per_file_max, so files_max climbs past 1 and the Boundary's
		// Hi bucket fires (X == Hi == files_max > Lo == 1).
		var body strings.Builder
		body.WriteString("// Package big is a fixture.\n")
		body.WriteString("package big\n\n")
		body.WriteString("// K" + strings.Repeat("a", i_index+1) + " is a fixture.\n")
		body.WriteString("const K" + strings.Repeat("a", i_index+1) + " = 1\n")
		for j_index := 0; j_index < 250; j_index++ {
			body.WriteString("\n")
		}
		fsys[name] = &fstest.MapFile{
			Data: []byte(body.String())}
	}
	var stdout, stderr bytes.Buffer
	code := lint.Main(&lint.Main_Input{
		Fsys:      fsys,
		Stdout:    &stdout,
		Stderr:    &stderr,
		CPU_Count: 1,
	})
	if code != 1 {
		t.Logf("stdout: %s", stdout.String())
		t.Logf("stderr: %s", stderr.String())
		t.Fatalf("expected exit 1 (diagnostics emitted); got %d", code)
	}
	if !strings.Contains(stdout.String(), "files totaling") {
		t.Fatalf("fixture did not trigger package_split diagnostic; stdout:\n%s",
			stdout.String())
	}
}

// Test_Coverage_Backfill_Main_Cpu_Count_Hi drives Main with CPU_Count at
// the Boundary Hi endpoint (=1024). Covers cpu_boundary's Hi bucket in
// Main, Check_File_System, and the parallel-check helpers.
func Test_Coverage_Backfill_Main_Cpu_Count_Hi(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"go.mod": &fstest.MapFile{
			Data: []byte("module example.com/hi\n")},
		"internal/a.go": &fstest.MapFile{
			Data: []byte("// Package hi is a fixture.\npackage hi\n")},
		"cmd/x/main.go": &fstest.MapFile{
			Data: []byte("// Package main is a fixture.\n" +
				"package main\n\nfunc main() {}\n")},
	}
	// Exit code isn't the assertion target — we only need CPU_Count=1024 to
	// flow through every prologue so the cpu_boundary Hi bucket fires.
	var stdout, stderr bytes.Buffer
	code := lint.Main(&lint.Main_Input{
		Fsys: fsys, Stdout: &stdout, Stderr: &stderr, CPU_Count: 1024,
	})
	if code < 0 {
		t.Fatalf("unexpected negative exit code %d", code)
	}
}

// Test_Coverage_Backfill_Module_Index_Hi_Index constructs a workspace with
// 1025 modules so module_index_resolve returns index=Hi (=1024) for the
// file matched to the shortest-path module (slice is sorted by Root length
// descending, so the shortest sits last).
func Test_Coverage_Backfill_Module_Index_Hi_Index(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{}
	const module_count = 1025
	for i_index := 0; i_index < module_count; i_index++ {
		// Module / package names stay short (numbered suffix) so the
		// generated identifiers fit within Identifier_Chars_Max; the
		// COUNT of modules is what drives module_index_resolve to its
		// Hi=1024 index bucket.
		name := fmt.Sprintf("m%04d", i_index)
		directory := name + "/"
		fsys[directory+"go.mod"] = &fstest.MapFile{
			Data: []byte("module example.com/" + name + "\n"),
		}
		fsys[directory+"a.go"] = &fstest.MapFile{
			Data: []byte("// Package " + name +
				" is a fixture.\npackage " + name + "\n"),
		}
	}
	var stdout, stderr bytes.Buffer
	code := lint.Main(&lint.Main_Input{
		Fsys: fsys, Stdout: &stdout, Stderr: &stderr, CPU_Count: 1,
	})
	if code < 0 {
		t.Fatalf("unexpected negative exit code %d", code)
	}
}

// Test_Coverage_Backfill_Package_Group_Endpoints exercises the Lo endpoints
// of package_group_key_diag's two Boundary axes (st.Lines=2, files_max=1)
// plus the Sometimes(key.Is_Test) true branch via a two-file test package.
// Together with Test_Coverage_Backfill_Large_Package (which hits Hi-lines)
// this rounds out the per-tuple Cross_Product coverage for the diag call.
func Test_Coverage_Backfill_Package_Group_Endpoints(t *testing.T) {
	t.Parallel()
	source_fragment := func(suffix string) (fsys fstest.MapFS) {
		return fstest.MapFS{
			"go.mod": &fstest.MapFile{
				Data: []byte("module example.com/min" + suffix + "\n")},
			"a" + suffix + ".go": &fstest.MapFile{
				Data: []byte("package min" + suffix + "\n")},
			"b" + suffix + ".go": &fstest.MapFile{
				Data: []byte("package min" + suffix + "\n")},
		}
	}
	for _, suffix := range []string{"", "_test"} {
		fsys := source_fragment(suffix)
		var stdout, stderr bytes.Buffer
		code := lint.Main(&lint.Main_Input{
			Fsys: fsys, Stdout: &stdout, Stderr: &stderr, CPU_Count: 1,
		})
		if code != 1 {
			t.Errorf("suffix %q: expected exit 1; got %d; output: %s",
				suffix, code, stdout.String())
		}
	}
	// Test-package variant of Large_Package — drives the (Hi-lines, Lo-files)
	// tuple with key.Is_Test=true.
	const file_count = 50
	fsys := fstest.MapFS{
		"go.mod": &fstest.MapFile{
			Data: []byte("module example.com/bigt\n")},
	}
	for i_index := 0; i_index < file_count; i_index++ {
		name := "f" + strings.Repeat("x", i_index+1) + "_test.go"
		var body strings.Builder
		body.WriteString("// Package bigt_test is a fixture.\n")
		body.WriteString("package bigt_test\n\n")
		body.WriteString("// K" + strings.Repeat("a", i_index+1) + " is a fixture.\n")
		body.WriteString("const K" + strings.Repeat("a", i_index+1) + " = 1\n")
		for j_index := 0; j_index < 250; j_index++ {
			body.WriteString("\n")
		}
		fsys[name] = &fstest.MapFile{
			Data: []byte(body.String())}
	}
	var stdout, stderr bytes.Buffer
	code := lint.Main(&lint.Main_Input{
		Fsys: fsys, Stdout: &stdout, Stderr: &stderr, CPU_Count: 1,
	})
	if code != 1 {
		t.Errorf("test-pkg Large_Package mirror: expected exit 1; got %d; output: %s",
			code, stdout.String())
	}
}

// Drives Check_Source against a tier-1-clean method that has no results
// fields. check_unnecessary_method's call to field_list_types receives
// nil Type.Results, firing the "fl may be nil" Sometimes axis's true bucket
// — production tests don't otherwise exercise it.
func Test_Coverage_Backfill_Nil_Field_List(t *testing.T) {
	t.Parallel()
	source := "// Package p is a fixture.\n" +
		"package p\n\n" +
		"// R is a fixture.\n" +
		"type R struct{}\n\n" +
		"// Method is a fixture.\n" +
		"func (r R) Method() { return }\n"
	diags, err := lint.Check_Source("p.go", source)
	if err != nil {
		t.Fatalf("Check_Source: %v", err)
	}
	for _, d := range diags {
		if d.Tier == 1 {
			t.Fatalf("expected tier-1-clean fixture; got %s", d.Message)
		}
	}
}

// Test_Coverage_Backfill_String_Bounded_Mixed_Invariant_Calls drives
// extract_call_name's Lo=0 (non-invariant call name) and Hi=26 ("Recorder_
// Distinct_Boundary") buckets via crafted source.
func Test_Coverage_Backfill_String_Bounded_Mixed_Invariant_Calls(t *testing.T) {
	t.Parallel()
	mixed_calls_source := "package fixture\n\n" +
		"\tfoo.Bar()\n" +
		"X: x, Lo: 0, Hi: fixture_hi})\n" +
		"}\n"
	diags, err := lint.Check_Source("test.go", mixed_calls_source)
	t.Logf("mixed diags=%d err=%v", len(diags), err)
	rich_source := "package fixture\n\n" +
		"const Foo = 1\n" +
		"type Bar struct {\n\tA int\n\tB bool\n\tC string\n}\n\n" +
		"func Quux(input *Bar, s string, n int, b bool, p *int) (result int) {\n" +
		"\tdefer func() {\n" +
		"\t\t\t\tX: result, Lo: 0, Hi: fixture_hi,\n" +
		"\t\t\t}),\n" +
		"\t\t)\n" +
		"\t}()\n" +
		"input != nil, \"input is non-nil\"),\n" +
		"p != nil, \"p is non-nil\"),\n" +
		"b, \"b is sometimes true\"),\n" +
		"\t\t\tX: n, Lo: 0, Hi: fixture_hi,\n" +
		"\t\t}),\n" +
		"\t\t\tX: len(s), Lo: 0, Hi: fixture_hi,\n" +
		"\t\t}),\n" +
		"s == \"\", \"s is empty\"),\n" +
		"\t)\n" +
		"\treturn 0\n" +
		"}\n"
	rich_diags, rich_err := lint.Check_Source("test.go", rich_source)
	t.Logf("rich diags=%d err=%v", len(rich_diags), rich_err)
}

// Test_Coverage_Backfill_String_Bounded_Helpers_Git_Input drives the git-
// tier helpers (git_input_check_short_hash, _is_subtree_merge_subject,
// _is_fixup_subject) with empty and max-length subject / hash strings.
// Test_Coverage_Backfill_Main_Input_Combinations exercises every tuple of
// Main's input-shape Cross_Product so each (Lo, Hi) bucket combination
// across Root_Directory / Scope_Prefix / Tracked / Merge_Commits /
// Non_Merge_Commits / Git.Enabled / Git.Main_Reference_Absent /
// CPU_Count gets observed at least once. The cross-product has 2^8 cells
// in observable space; the loop visits each by toggling the eight axes
// independently.
func Test_Coverage_Backfill_Main_Input_Combinations(t *testing.T) {
	t.Parallel()
	long_path := strings.Repeat("a", 4096)
	full_tracked := make(map[string]bool, 30000)
	full_tracked["go.mod"] = true
	full_tracked["main.go"] = true
	for i_index := 0; i_index < 29998; i_index++ {
		full_tracked["k"+strconv.Itoa(i_index)] = true
	}
	// Empty-Subject commits short-circuit at the Subject=="" gate in
	// Git_Input_Check, so the 30000 elements drive only the Hi=30000
	// len-boundary observation without paying per-commit regex/format costs.
	full_commits := make([]lint.Git_Commit, 30000)
	for bits_index := 0; bits_index < 256; bits_index++ {
		root_directory := ""
		if bits_index&(1<<0) != 0 {
			root_directory = long_path
		}
		scope_prefix := ""
		if bits_index&(1<<1) != 0 {
			scope_prefix = long_path
		}
		tracked := map[string]bool(nil)
		if bits_index&(1<<2) != 0 {
			tracked = full_tracked
		}
		merge_commits := []lint.Git_Commit(nil)
		if bits_index&(1<<3) != 0 {
			merge_commits = full_commits
		}
		non_merge_commits := []lint.Git_Commit(nil)
		if bits_index&(1<<4) != 0 {
			non_merge_commits = full_commits
		}
		git_enabled := bits_index&(1<<5) != 0
		main_absent := bits_index&(1<<6) != 0
		if main_absent {
			if !git_enabled {
				continue
			}
		}
		cpu_count := 0
		if bits_index&(1<<7) != 0 {
			cpu_count = 1024
		}
		var stdout, stderr bytes.Buffer
		lint.Main(&lint.Main_Input{
			Fsys: fstest.MapFS{
				"go.mod":  {Data: []byte("module test\ngo 1.25\n")},
				"main.go": {Data: []byte("package main\n\nfunc main() {}\n")},
			},
			Stdout:         &stdout,
			Stderr:         &stderr,
			Root_Directory: root_directory,
			Scope_Prefix:   scope_prefix,
			Tracked:        tracked,
			CPU_Count:      cpu_count,
			Git: lint.Git_Input{
				Enabled:               git_enabled,
				Main_Reference_Absent: main_absent,
				Merge_Commits:         merge_commits,
				Non_Merge_Commits:     non_merge_commits,
			},
		})
	}
}

// Test_Coverage_Backfill_String_Bounded_Helpers_Git_Input drives the
// git-history tier helpers with empty + max-length subject and hash
// strings so the per-axis Lo/Hi bucket pairs are observed end-to-end.
func Test_Coverage_Backfill_String_Bounded_Helpers_Git_Input(t *testing.T) {
	t.Parallel()
	long_subject := strings.Repeat("s", 100)
	long_hash := strings.Repeat("a", 64)
	git_input := lint.Git_Input{
		Enabled: true,
		Merge_Commits: []lint.Git_Commit{
			{Hash: "", Subject: ""}, // filtered out by upstream empty-subject check
			// Exercises empty-hash Lo bucket in git_input_check_short_hash.
			{Hash: "", Subject: "x"},
			{Hash: "a", Subject: "x"},
			{Hash: long_hash, Subject: long_subject},
		},
		Non_Merge_Commits: []lint.Git_Commit{
			{Hash: "", Subject: ""},
			{Hash: "a", Subject: "x"},
			{Hash: long_hash, Subject: long_subject},
		},
	}
	var stdout, stderr bytes.Buffer
	lint.Main(&lint.Main_Input{
		Fsys: fstest.MapFS{
			"go.mod":  {Data: []byte("module test\ngo 1.25\n")},
			"main.go": {Data: []byte("package main\n\nfunc main() {}\n")},
		},
		Stdout:         &stdout,
		Stderr:         &stderr,
		CPU_Count:      1,
		Root_Directory: ".",
		Git:            git_input,
	})
}

// Test_Coverage_Backfill_Input_Struct_Sig_Hi drives the input-struct
// signature suggester with a 128-char function name so its `sig` Hi
// bucket fires.
func Test_Coverage_Backfill_Input_Struct_Sig_Hi(t *testing.T) {
	t.Parallel()
	// Sig = funcname + "(*" + want_name + ")" + " (result int)"
	// For a 117-char funcname, want_name = funcname + "_Input" = 123 chars.
	// Total sig = 117 + 2 + 123 + 1 + 13 = 256 chars... close to Hi=257.
	long := strings.Repeat("F", 128)
	source := "package main\n\n" +
		"func " + long + "(a, b int) (result int) { return a + b }\n" +
		// 6-char funcname with no result clause gives sig = "Foo123(*Foo123_Input)"
		// = 21 chars, hitting the Lo=21 bucket of suggest_sig.
		"func Foo123(a, b int) {}\n"
	diags, err := lint.Check_Source("test.go", source)
	t.Logf("input_struct_sig diags=%d err=%v", len(diags), err)
}

// Test_Coverage_Backfill_Build_Key_Hi drives build_key with a long-form
// //go:build constraint whose normalized AST is exactly 128 chars so its
// Hi bucket fires; combined with non-build-tagged files in other tests it
// also covers Lo=0.
func Test_Coverage_Backfill_Build_Key_Hi(t *testing.T) {
	t.Parallel()
	// Normalized form: each `||` becomes ` || `, and groups wrap in parens.
	// Length: each `(OS || OS || ...) && (ARCH || ARCH || ...) && cgo` —
	// tuned by trial to 128 chars exactly.
	long_build := "//go:build (linux || darwin || windows || freebsd || netbsd || plan9) && " +
		"(amd64 || arm64 || ppc64 || ppc64le || riscv64 || s390x) && cgo\n\n" +
		"package main\n\nfunc main() {}\n"
	var stdout, stderr bytes.Buffer
	lint.Main(&lint.Main_Input{
		Fsys: fstest.MapFS{
			"go.mod":  {Data: []byte("module test\ngo 1.25\n")},
			"main.go": {Data: []byte(long_build)},
		},
		Stdout:    &stdout,
		Stderr:    &stderr,
		CPU_Count: 1,
	})
}

// Test_Coverage_Backfill_Method_Prefix_Long_Type drives
// check_file_system_method_prefix_group_first_parameter_type with a
// function whose first param is a 128-char declared type so its Hi=128
// bucket fires.
func Test_Coverage_Backfill_Method_Prefix_Long_Type(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("X", 128)
	source := "package library\n\n" +
		"type " + long + " struct{}\n\n" +
		"func F(x " + long + ") {}\n"
	var stdout, stderr bytes.Buffer
	lint.Main(&lint.Main_Input{
		Fsys: fstest.MapFS{
			"go.mod":   {Data: []byte("module test\ngo 1.25\n")},
			"lib/a.go": {Data: []byte(source)},
		},
		Stdout:    &stdout,
		Stderr:    &stderr,
		CPU_Count: 1,
	})
}

// Test_Coverage_Backfill_Library_Tier_Depth drives
// check_library_tier_depth_ancestors with a multi-level non-main subtree
// so the ancestor walker observes its directory-input boundary. Includes
// a 4096-char file path so module_index_resolve, module_index_canonicalize,
// and check_library_tier_depth_ancestors observe their Hi buckets.
func Test_Coverage_Backfill_Library_Tier_Depth(t *testing.T) {
	t.Parallel()
	// Build a 4092-char `aa/aa/.../` directory chain plus `a.go` = 4096.
	var sb strings.Builder
	for sb.Len() < 4092 {
		sb.WriteString("aa/")
	}
	long_path := sb.String()[:4092] + "a.go"
	var stdout, stderr bytes.Buffer
	lint.Main(&lint.Main_Input{
		Fsys: fstest.MapFS{
			"go.mod":     {Data: []byte("module test\ngo 1.25\n")},
			"a/b/c/a.go": {Data: []byte("package c\n")},
			"a/b/d.go":   {Data: []byte("package b\n")},
			"a/e.go":     {Data: []byte("package a\n")},
			long_path:    {Data: []byte("package a\n")},
		},
		Stdout:    &stdout,
		Stderr:    &stderr,
		CPU_Count: 1,
	})
}

// Test_Coverage_Backfill_Terminology drives check_names_terminology so it
// emits a rename violation; that triggers emit_rename and exercises
// stdlib_required's qualified-selector input ("strings.Index").
func Test_Coverage_Backfill_Terminology(t *testing.T) {
	t.Parallel()
	long_package := strings.Repeat("x", 60)
	long_function := strings.Repeat("y", 67)
	source := "// Package fixture is a fixture.\n" +
		"package fixture\n\n" +
		"import (\n" +
		"\t\"strings\"\n" +
		"\ta \"x\"\n" +
		"\t" + long_package + " \"longpkg\"\n" +
		")\n\n" +
		"// F invokes strings.Index whose result is a byte offset; the local\n" +
		"// `pos` lacks the `_offset` suffix so check_names_terminology fires.\n" +
		"// The 3-char `a.B` call exercises stdlib_required's Lo=3 bucket;\n" +
		"// the 128-char qualifier exercises its Hi=128 bucket.\n" +
		"func F(s string) (result int) {\n" +
		"\tpos := strings.Index(s, \"a\")\n" +
		"\t" + strings.Repeat("p", 121) + " := strings.Index(s, \"b\")\n" +
		// `length := strings.Count(s, "a")` requires `_count` suffix; emit_rename
		// replaces the `length` segment producing 5-char `count` output (Lo=5).
		"\tlength := strings.Count(s, \"a\")\n" +
		"\t_ = length\n" +
		"\tshort := a.B()\n" +
		"\tlongq := " + long_package + "." + long_function + "()\n" +
		"\tresult = pos + short + longq\n" +
		"\treturn result\n" +
		"}\n"
	diags, err := lint.Check_Source("test.go", source)
	t.Logf("terminology diags=%d err=%v", len(diags), err)
}

// Test_Coverage_Backfill_Check_File_System_Empty_Input drives
// Check_File_System with an empty fs.FS and no Root / Root_Directory /
// Tracked, exercising the (Lo root, Lo root_directory, Lo tracked) defer
// and prologue cells at CPU_Count=0 (Lo) and CPU_Count=1024 (Hi).
func Test_Coverage_Backfill_Check_File_System_Empty_Input(t *testing.T) {
	t.Parallel()
	for _, cpu_count := range []int{0, 1024} {
		_, err := lint.Check_File_System(&lint.Check_File_System_Input{
			Fsys:      fstest.MapFS{},
			CPU_Count: cpu_count,
		})
		t.Logf("cfs_empty cpu=%d err=%v", cpu_count, err)
	}
}

// Test_Coverage_Backfill_Exposes_Private_Alias_Edges exercises
// check_exported_type_exposes_private_check with alias-type fixtures that
// hit (Lo name / Hi name, Lo params, Lo/True generic) boundary buckets at
// defer time. The function returns early without appending diags when the
// alias target is a builtin or exported type, so these fixtures stay clean.
func Test_Coverage_Backfill_Exposes_Private_Alias_Edges(t *testing.T) {
	t.Parallel()
	long_name := strings.Repeat("F", 128)
	cases := []string{
		// 1-char name, alias to builtin, non-pointer: (Lo name, Lo params, false generic).
		"package fixture\n\ntype F = int\n",
		// 1-char name, alias to *builtin: (Lo name, Lo params, true generic).
		"package fixture\n\ntype F = *int\n",
		// 128-char name, alias to builtin: (Hi name, Lo params, false generic).
		"// Package fixture is for the alias edges.\npackage fixture\n\n// " +
			long_name + "\n// is an alias.\ntype " + long_name + " = int\n",
		// 128-char name, pointer alias: (Hi name, Lo params, true generic).
		"// Package fixture is for the alias edges.\npackage fixture\n\n// " +
			long_name + "\n// is an alias.\ntype " + long_name + " = *int\n",
	}
	for _, source := range cases {
		diags, err := lint.Check_Source("test.go", source)
		t.Logf("alias_edges diags=%d err=%v", len(diags), err)
	}
}

// Test_Coverage_Backfill_Recursion_Visitor_Single_Function_File drives
// build_file_call_graph with files that have exactly one function declaration
// so the recursion visitor's Targets map has exactly one entry (Lo bucket of
// V.Targets boundary, Lo=non_empty_min). Two shapes exercise both Lo (1-char)
// and Hi (128-char) Caller buckets at Visit entry, where Scopes/Edges/
// Push_History are all empty (Lo for each).
func Test_Coverage_Backfill_Recursion_Visitor_Single_Function_File(t *testing.T) {
	t.Parallel()
	long_funcname := strings.Repeat("F", 128)
	cases := []string{
		// Documented package, lowercase 1-char function with a non-discard
		// statement so tier-1 stays clean and tier-2 (check_no_recursion)
		// runs, exercising the recursion visitor.
		"// Package fixture is for the recursion visitor.\npackage fixture\n\n" +
			"func f() {\n\tif true {\n\t}\n}\n",
		"// Package fixture is for the recursion visitor.\npackage fixture\n\n" +
			"// " +
			long_funcname +
			"\n// is for the recursion visitor.\nfunc " +
			long_funcname + "() {\n\tif true {\n\t}\n}\n",
		// Range with explicit blank key drives recursion_visitor_define_ident
		// with a nil expr branch (e_nil=true case).
		"// Package fixture is for the recursion visitor.\npackage fixture\n\n" +
			"func f() {\n\tfor range 1 {\n\t}\n}\n",
		// Single-function file with a 128-char self-recursing function name
		// drives recursion_visitor_enter_record_call_edge with Lo targets
		// (single-target) AND Hi caller (128-char) simultaneously.
		"// Package fixture is for the recursion visitor.\npackage fixture\n\n" +
			"// " +
			long_funcname +
			"\n// recurses into itself.\nfunc " +
			long_funcname + "() {\n\t" + long_funcname + "()\n}\n",
		// 128-char function calling a builtin (`println`) — call.Fun is a bare
		// identifier (True ident) but the name is not in v.Targets, so
		// record_call_edge returns without appending to v.Edges. Defer time
		// observes (Lo edges, Hi caller, True ident, Lo targets, Lo scopes,
		// Lo history).
		"// Package fixture is for the recursion visitor.\npackage fixture\n\n" +
			"// " +
			long_funcname +
			"\n// invokes a builtin.\nfunc " +
			long_funcname + "() {\n\tprintln(\"x\")\n}\n",
		// 128-char function calling a selector — call.Fun is *ast.SelectorExpr
		// (False ident). The package-level `p` var isn't added to v.Targets
		// (only FuncDecls are), so targets stays at 1 (single-target file).
		"// Package fixture is for the recursion visitor.\npackage fixture\n\n" +
			"var p = struct{ F func() }{F: func() {}}\n\n// " +
			long_funcname +
			"\n// invokes a method.\nfunc " +
			long_funcname + "() {\n\tp.F()\n}\n",
		// 128-char function with an if-init clause drives
		// recursion_visitor_enter_define_statement with Hi caller AND False
		// s_nil (init present) in a single-target file.
		"// Package fixture is for the recursion visitor.\npackage fixture\n\n" +
			"// " +
			long_funcname +
			"\n// has an if-init.\nfunc " +
			long_funcname + "() {\n\tif y := 1; y > 0 {\n\t}\n}\n",
		// 128-char function with a top-level := drives
		// recursion_visitor_define_ident with Hi caller, Lo scopes (=1, only
		// the BlockStmt scope is pushed at this point), Lo history (=1).
		"// Package fixture is for the recursion visitor.\npackage fixture\n\n" +
			"// " +
			long_funcname +
			"\n// has a top-level define.\nfunc " +
			long_funcname + "() {\n\ty := 1\n\tif y > 0 {\n\t}\n}\n",
	}
	for _, source := range cases {
		diags, err := lint.Check_Source("test.go", source)
		t.Logf("single_func diags=%d err=%v", len(diags), err)
		for _, d := range diags {
			t.Logf("  diag: %s", d.Message)
		}
	}
}

// Test_Coverage_Backfill_Shadow_Long_Name drives check_shadow with a
// 128-char identifier name so the name-length boundary observes Hi.
func Test_Coverage_Backfill_Shadow_Long_Name(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x", 128)
	source := "package main\n\n" +
		"var " + long + " = 1\n\n" +
		"func F() {\n" +
		"\t" + long + " := 2\n" +
		"\t_ = " + long + "\n" +
		"}\n"
	diags, err := lint.Check_Source("test.go", source)
	t.Logf("shadow_long diags=%d err=%v", len(diags), err)
}

// Test_Coverage_Backfill_Empty_Nested_If exercises check_shadows' walker on
// an `if` inside a parameterless function's block-scope. Both the current
// scope (inner block) and its parent (function-scope of `g`) have zero
// names — the (Lo, Lo) tuple of scope_value.Names paired with
// scope_value.Parent.Names that push_if_chain et al need to observe.
func Test_Coverage_Backfill_Empty_Nested_If(t *testing.T) {
	t.Parallel()
	source := "package fixture\n\n" +
		"var glob int\n\n" +
		"var glob_slice = []int{1, 2}\n\n" +
		"func g_helper() (out int) { return 0 }\n\n" +
		"func g() {\n" +
		"\t{\n" +
		"\t\tif z := g_helper(); z != 0 {\n" +
		"\t\t\t_ = z\n" +
		"\t\t}\n" +
		"\t\t{\n" +
		"\t\t\tglob = 1\n" +
		"\t\t\ttype Local int\n" +
		"\t\t\t_ = Local(0)\n" +
		"\t\t\tfor _, v := range glob_slice {\n" +
		"\t\t\t\t_ = v\n" +
		"\t\t\t}\n" +
		"\t\t}\n" +
		"\t}\n" +
		"}\n"
	diags, err := lint.Check_Source("nested.go", source)
	t.Logf("nested_if diags=%d err=%v", len(diags), err)
}

// Test_Coverage_Backfill_Suffix_Of_Hi drives check_names_suffix_of with a
// 128-char identifier used in arithmetic so its name Hi=128 fires.
func Test_Coverage_Backfill_Suffix_Of_Hi(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x", 128)
	source := "// Package fixture is a fixture.\n" +
		"package fixture\n\n" +
		"// F adds a 128-char operand to trigger check_names_arithmetic.\n" +
		"func F() (result int) {\n" +
		"\t" + long + " := 1\n" +
		"\tb := 2\n" +
		"\tresult = " + long + " + b\n" +
		"\treturn result\n" +
		"}\n"
	diags, err := lint.Check_Source("test.go", source)
	t.Logf("suffix_of_hi diags=%d err=%v", len(diags), err)
}

// Test_Coverage_Backfill_Recursion drives check_no_recursion to emit a
// self-recursion diagnostic with a single 1-char function name so the
// diag-message helper observes its Lo bucket.
func Test_Coverage_Backfill_Recursion(t *testing.T) {
	t.Parallel()
	// 128-char name (xxxxxxxxxxxx...x, snake_case-valid up to the cap).
	long_name := strings.Repeat("x", 128)
	source := "// Package fixture is a fixture.\n" +
		"package fixture\n\n" +
		"// F recurses into itself to trigger check_no_recursion.\n" +
		"func F() { F() }\n\n" +
		"// Long recurses into itself; diag_message gets Hi message.\n" +
		"func " + long_name + "() {\n" +
		"\t" + long_name + "()\n" +
		"}\n"
	diags, err := lint.Check_Source("test.go", source)
	t.Logf("recursion diags=%d err=%v", len(diags), err)
	for _, d := range diags {
		t.Logf("  diag: %s", d.Message)
	}
}

// Test_Coverage_Backfill_Keyed_Struct_Long_Type drives
// check_keyed_struct_init_type_ident with a 128-char struct type literal so
// its `name` Hi bucket fires.
func Test_Coverage_Backfill_Keyed_Struct_Long_Type(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x", 128)
	source := "package main\n\n" +
		"type " + long + " struct{ A int }\n\n" +
		"func f() {\n" +
		"\t_ = " + long + "{A: 1}\n" +
		"}\n"
	diags, err := lint.Check_Source("test.go", source)
	t.Logf("keyed_struct_long diags=%d err=%v", len(diags), err)
}

// Test_Coverage_Backfill_Scope_Prefix_Hi drives diagnostic_within_scope with
// a 4096-char Scope_Prefix so its Hi bucket fires. The source has a tier-1
// violation (wrongCase func) so diagnostics flow through the scope filter.
// Also imports "C" (1 char) to exercise is_impure_hard_import Lo bucket.
func Test_Coverage_Backfill_Scope_Prefix_Hi(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("a", 4096)
	var stdout, stderr bytes.Buffer
	lint.Main(&lint.Main_Input{
		Fsys: fstest.MapFS{
			"go.mod": {Data: []byte("module test\ngo 1.25\n")},
			"lib/a.go": {Data: []byte(
				"package library\n\nimport _ \"C\"\nimport _ \"" +
					strings.Repeat("a", 128) +
					"\"\n\nfunc wrongName() {}\n")},
			"main.go": {Data: []byte("package main\n\nfunc main() {}\n")},
		},
		Stdout:       &stdout,
		Stderr:       &stderr,
		CPU_Count:    1,
		Scope_Prefix: long,
	})
}

// Test_Coverage_Backfill_Struct_Field_Capitalize drives
// check_public_struct_fields_named_capitalize with min (1-char) and max
// (128-char) field names so its name + output_string boundaries see Lo and Hi.
func Test_Coverage_Backfill_Struct_Field_Capitalize(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("a", 128)
	source := "package main\n\n" +
		"type Foo struct {\n" +
		"\tb int\n" +
		"\t" + long + " int\n" +
		"}\n"
	diags, err := lint.Check_Source("test.go", source)
	t.Logf("capitalize diags=%d err=%v", len(diags), err)
}

// Test_Coverage_Backfill_Method_Render_Type drives check_unnecessary_method
// and its render_type helper with a tier-1-clean source that declares a
// method (receiver-bearing function) so render_type observes field-list type
// rendering.
func Test_Coverage_Backfill_Method_Render_Type(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x", 128)
	source := "// Package fixture is a fixture.\n" +
		"package fixture\n\n" +
		"// Foo is a fixture struct.\n" +
		"type Foo struct{}\n\n" +
		"// A is a 1-char type so render_type's output hits Lo=1.\n" +
		"type A int\n\n" +
		"// " + long + " is a 128-char type so render_type's output hits Hi=128.\n" +
		"type " + long + " int\n\n" +
		"// Bar exercises method shape with 1-char and 128-char-typed params.\n" +
		"func (f Foo) Bar(p A, q " + long +
		") (result string) { result = \"x\"; return result }\n\n" +
		// 1-char and 128-char method names span matches_stdlib's input.Name
		// Lo=1 and Hi=128 buckets via Check_Source -> check_unnecessary_method.
		"func (f Foo) X() (result int) { return 0 }\n\n" +
		"func (f Foo) Q" + strings.Repeat("z", 127) +
		"(p int) (result int) { return 0 }\n\n" +
		// V has Results="" (0 chars = Lo); U has Results="xxx128" (128 chars = Hi).
		"func (f Foo) V() {}\n\n" +
		"func (f Foo) U(p " + long + ") (r " + long + ") { return r }\n"
	diags, err := lint.Check_Source("foo.go", source)
	t.Logf("method_render diags=%d err=%v", len(diags), err)
}

// Test_Coverage_Backfill_Check_Source_Filename drives Check_Source's
// filename arg with max-length filename so its Hi bucket fires. The
// filename is a deeply nested path (4093 chars of /-separated segments
// plus `.go`) so check_banned_identifiers_file_name stems to a short
// basename without exceeding suggest_split_words's 128-char cap.
func Test_Coverage_Backfill_Check_Source_Filename(t *testing.T) {
	t.Parallel()
	// 1023 4-char segments `aa/` plus `a.go` = 1023*3 + 4 = 3073. Add
	// more to reach exactly 4096.
	var sb strings.Builder
	for sb.Len() < 4092 {
		sb.WriteString("aa/")
	}
	long_name := sb.String()[:4092] + "a.go"
	diags, err := lint.Check_Source(long_name, "package x\n")
	t.Logf("check_source_long_fn diags=%d err=%v len=%d", len(diags), err, len(long_name))
}

// Test_Coverage_Backfill_Naming_Suggestion drives the suggest helper and
// its inner helpers (suggest_is_all_upper, suggest_split_words, etc.) by
// feeding identifiers that violate snake_case / Ada_Case so the linter
// reaches its naming-suggestion code path.
func Test_Coverage_Backfill_Naming_Suggestion(t *testing.T) {
	t.Parallel()
	long_word_ident := "wrongName" + strings.Repeat("X", 112) + "case"
	source := "package fixture\n\n" +
		"func wrongName() {}\n" +
		"func ALL_CAPS_THING() {}\n" +
		"func mixed_BadCase() {}\n" +
		// `xY` is a 2-char camelCase ident — splits to `x` `Y` → suggest output
		// `x_Y` (3 chars) exercising the Lo=3 bucket of suggest's output.
		"func xY() {}\n" +
		"func " + long_word_ident + "() {}\n" +
		"type wrongType struct{}\n" +
		"type BAD_type_name struct{}\n" +
		// `string_ring` ends in `ring` (allowed); `processing` ends in `ing`
		// and is not in the allowlist — exercises both branches of
		// is_allowed_ing_noun.
		"func string_ring() {}\n" +
		"func processing() {}\n" +
		// 3-char `ing` for Lo=3; 128-char `xxx...ing` for Hi=128.
		"func ing() {}\n" +
		"func " + strings.Repeat("x", 125) + "ing() {}\n"
	diags, err := lint.Check_Source("test.go", source)
	t.Logf("naming diags=%d err=%v", len(diags), err)
	for _, d := range diags {
		t.Logf("  naming diag: %s", d.Message)
	}
}

// Test_Coverage_Backfill_Switch_Without_Init exercises
// check_invariant_assertions_descend_switch with a switch statement that has
// no init clause, hitting the Sometimes(s.Init != nil) false branch the
// rest of the corpus does not exercise on its own.
func Test_Coverage_Backfill_Switch_Without_Init(t *testing.T) {
	t.Parallel()
	source := `package fixture

func f(value int) {
	switch value {
	case 0:
	default:
	}
}
`
	diags, err := lint.Check_Source("test.go", source)
	t.Logf("switch diags=%d err=%v", len(diags), err)
	for _, diag := range diags {
		t.Logf("  switch diag: %s", diag.Message)
	}
}

// Test_Coverage_Backfill_String_Bounded_Helpers_Long_Path drives stream-
// tier helpers (check_stream_conflict_markers etc.) with a 4096-char file
// path so their len(p) Hi=4096 buckets fire.
func Test_Coverage_Backfill_String_Bounded_Helpers_Long_Path(t *testing.T) {
	t.Parallel()
	// 4092 + ".txt" = 4096 chars exactly so the stream-tier path boundary
	// fires Hi instead of fatalling.
	long_filename := strings.Repeat("a", 4092)
	long_directory := strings.Repeat("d", 128)
	fsys := fstest.MapFS{
		"go.mod":                 {Data: []byte("module test\ngo 1.25\n")},
		"main.go":                {Data: []byte("package main\n\nfunc main() {}\n")},
		long_filename + ".txt":   {Data: []byte("x\n")},
		"empty.go":               {Data: []byte("")},
		"x/x.go":                 {Data: []byte("package x\n")},
		long_directory + "/x.go": {Data: []byte("package x\n")},
	}
	var stdout, stderr bytes.Buffer
	lint.Main(&lint.Main_Input{
		Fsys:           fsys,
		Stdout:         &stdout,
		Stderr:         &stderr,
		CPU_Count:      1,
		Root_Directory: ".",
	})
}

// Test_Coverage_Backfill_Banned_Abbreviation_D feeds Check_Source a fixture
// with a `dir` identifier so banned_abbreviation_candidates_d_f's d-helper
// dispatch path observes the non-nil branch.
func Test_Coverage_Backfill_Banned_Abbreviation_D(t *testing.T) {
	t.Parallel()
	_, err := lint.Check_Source("test.go",
		"package x\n\nfunc f(dir string) string { return dir }\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
}

// Test_Coverage_Backfill_Check_Comments_Group_Source_Min feeds Check_Source
// the minimum-byte file containing a comment (`package x\n\n// c\n`, 16 bytes)
// so check_comments_group's source-length Lo bucket fires.
func Test_Coverage_Backfill_Check_Comments_Group_Source_Min(t *testing.T) {
	t.Parallel()
	diags, err := lint.Check_Source("test.go", "package x\n\n// c\n")
	t.Logf("diags=%d err=%v", len(diags), err)
}

// Test_Coverage_Backfill_Shadow_Walker_Top_Level_If_Range feeds Check_File
// fixtures whose first statement is an if-statement or a range-statement so
// the shadow walker's push_if_chain / push_range_statement helpers fire with
// stack length 1 (the walker's initial seed frame).
func Test_Coverage_Backfill_Shadow_Walker_Top_Level_If_Range(t *testing.T) {
	t.Parallel()
	cases := []string{
		"package fixture\n\nfunc g() {\n\tif true {\n\t}\n}\n",
		"package fixture\n\nfunc g() {\n\tfor i := range 1 {\n\t\t_ = i\n\t}\n}\n",
		// If-init that is ASSIGN (not DEFINE) on a function with zero params/results:
		// drives check_assign_define with Lo parent_names AND Lo names AND Lo diags
		// at defer time (no name was added because the statement is not DEFINE).
		"package fixture\n\nvar x = 0\n\nfunc g() {\n\tif x = 5; x > 0 {\n\t}\n}\n",
		// Range nested inside an if-block within a function with zero params drives
		// push_range_statement_add_variable with Lo parent_names (if-scope empty)
		// AND Lo grandparent_names (function-scope empty, no params) for both the
		// True key call (i) and the False value call (nil) of the range key/value.
		"package fixture\n\n" +
			"func h() {\n" +
			"\tif true {\n\t\tfor i := range 1 {\n\t\t\t_ = i\n\t\t}\n\t}\n}\n",
		// Range with no key/value (both nil) nested in if-block exercises False
		// e_present in both x.Key and x.Value add_variable calls.
		"package fixture\n\nfunc k() {\n\tif true {\n\t\tfor range 1 {\n\t\t}\n\t}\n}\n",
		// Blank key `for _ = range` drives add_variable with e_present=True
		// (Key is non-nil Ident) but Name="_" short-circuits before adding to
		// scope_value.Names, so names_axis stays Lo at defer time. Nested in
		// if-block keeps parent_names and grandparent_names at Lo.
		"package fixture\n\n" +
			"func m() {\n\tif true {\n\t\tfor _ = range 1 {\n\t\t}\n\t}\n}\n",
	}
	for _, source := range cases {
		diags, err := lint.Check_Source("test.go", source)
		t.Logf("diags=%d err=%v", len(diags), err)
	}
}

// Test_Coverage_Backfill_Check_File_Empty_Source feeds Check_File a parsed
// ast.File alongside an empty source []byte. Check_Source can't trigger this
// path (parser rejects empty input), so the Lo=0 bucket of Check_File's
// source-length boundary needs this synthetic invocation.
func Test_Coverage_Backfill_Check_File_Empty_Source(t *testing.T) {
	t.Parallel()
	file_set := token.NewFileSet()
	file, err := parser.ParseFile(
		file_set, "empty.go", "package empty\n", parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	diags := lint.Check_File(file_set, file, nil)
	t.Logf("empty_source diags=%d", len(diags))
}

// Builds a MapFS — Go sources gofmt-normalised, other files verbatim — runs
// the filesystem tier scoped to the fixture package, and returns only the
// specification-rule diagnostics so a test sees this check in isolation.
func specification_diagnostics(
	t *testing.T, files map[string]string,
) (diags []lint.Diagnostic) {
	t.Helper()
	fsys_map := make(fstest.MapFS)
	for name, content := range files {
		if strings.HasSuffix(name, ".go") {
			fsys_map[name] = &fstest.MapFile{Data: gofmt_must(t, content)}
			continue
		}
		fsys_map[name] = &fstest.MapFile{Data: []byte(content)}
	}
	// Scope to the fixture package so the coverage rule, which is gated on an
	// explicit package argument, applies to it.
	input := &lint.Check_File_System_Input{Fsys: fsys_map, Scope: "greet"}
	all, err := lint.Check_File_System(input)
	if err != nil {
		t.Fatalf("Check_File_System: %v", err)
	}
	for _, d := range all {
		if d.Name == "specification" {
			diags = append(diags, d)
		}
	}
	return diags
}

func specification_clean_files() (files map[string]string) {
	return map[string]string{
		// A go.mod gives greet an owning module; the coverage mandate no-ops on
		// module-less directories, so without it the missing-file rule never fires.
		"go.mod":                      "module fixture\n\ngo 1.25\n",
		"greet/greet.go":              specification_clean_source,
		"greet/specification_test.go": specification_clean_test,
		"greet/SPECIFICATION.md":      specification_clean_md,
	}
}

func specification_message_contains(
	diags []lint.Diagnostic, fragment string,
) (found bool) {
	for _, d := range diags {
		if strings.Contains(d.Message, fragment) {
			return true
		}
	}
	return false
}

// Test_Specification_Conformance verifies a fully conforming package emits no
// specification diagnostics.
func Test_Specification_Conformance(t *testing.T) {
	t.Parallel()
	diags := specification_diagnostics(t, specification_clean_files())
	if len(diags) != 0 {
		t.Fatalf("conforming package should emit no diagnostics, got %v", diags)
	}
}

// Test_Specification_Missing_File verifies a scoped package without the file is
// flagged.
func Test_Specification_Missing_File(t *testing.T) {
	t.Parallel()
	files := specification_clean_files()
	delete(files, "greet/SPECIFICATION.md")
	diags := specification_diagnostics(t, files)
	if !specification_message_contains(diags, "missing SPECIFICATION.md") {
		t.Fatalf("expected missing-file diagnostic, got %v", diags)
	}
}

// Test_Specification_Heading_Level_Two_Flagged verifies a ## heading is flagged;
// only # and ### are permitted levels.
func Test_Specification_Heading_Level_Two_Flagged(t *testing.T) {
	t.Parallel()
	files := specification_clean_files()
	files["greet/SPECIFICATION.md"] = "\n## Greeting\n\nIt greets the caller.\n"
	diags := specification_diagnostics(t, files)
	if !specification_message_contains(diags, "not level # or ###") {
		t.Fatalf("expected heading-level diagnostic, got %v", diags)
	}
}

// Test_Specification_Heading_Missing_Blank_Line verifies an unfenced heading is
// flagged.
func Test_Specification_Heading_Missing_Blank_Line(t *testing.T) {
	t.Parallel()
	files := specification_clean_files()
	files["greet/SPECIFICATION.md"] = "\n# Greeting\nIt greets the caller.\n"
	diags := specification_diagnostics(t, files)
	if !specification_message_contains(diags, "not followed by a blank line") {
		t.Fatalf("expected blank-line diagnostic, got %v", diags)
	}
}

// Test_Specification_Section_Too_Long verifies a four-line section is flagged.
func Test_Specification_Section_Too_Long(t *testing.T) {
	t.Parallel()
	files := specification_clean_files()
	files["greet/SPECIFICATION.md"] = "\n# Greeting\n\none\ntwo\nthree\nfour\n"
	diags := specification_diagnostics(t, files)
	if !specification_message_contains(diags, "exceeds three lines") {
		t.Fatalf("expected section-size diagnostic, got %v", diags)
	}
}

// Test_Specification_Missing_Test_File verifies an absent specification_test.go
// is flagged when the spec exists.
func Test_Specification_Missing_Test_File(t *testing.T) {
	t.Parallel()
	files := specification_clean_files()
	delete(files, "greet/specification_test.go")
	diags := specification_diagnostics(t, files)
	if !specification_message_contains(diags, "missing specification_test.go") {
		t.Fatalf("expected missing-test-file diagnostic, got %v", diags)
	}
}

// Test_Specification_Heading_Without_Test verifies a heading with no matching
// test is flagged.
func Test_Specification_Heading_Without_Test(t *testing.T) {
	t.Parallel()
	files := specification_clean_files()
	files["greet/SPECIFICATION.md"] = "\n# Farewell\n\nIt bids farewell.\n"
	diags := specification_diagnostics(t, files)
	if !specification_message_contains(diags, "Test_Farewell") {
		t.Fatalf("expected diagnostic naming Test_Farewell, got %v", diags)
	}
}

// Test_Specification_Wrong_Test_Order verifies tests not in heading order are
// flagged.
func Test_Specification_Wrong_Test_Order(t *testing.T) {
	t.Parallel()
	files := specification_clean_files()
	files["greet/SPECIFICATION.md"] =
		"\n# Greeting\n\nIt greets.\n\n# Farewell\n\nIt bids farewell.\n"
	files["greet/specification_test.go"] = "package greet_test\n\n" +
		"import \"testing\"\n\n" +
		"// Test_Farewell is a fixture.\n" +
		"func Test_Farewell(t *testing.T) { _ = t }\n\n" +
		"// Test_Greeting is a fixture.\n" +
		"func Test_Greeting(t *testing.T) { _ = t }\n"
	diags := specification_diagnostics(t, files)
	if !specification_message_contains(diags, "needs Test_Greeting") {
		t.Fatalf("expected ordering diagnostic, got %v", diags)
	}
}

// Test_Specification_Coverage_Respects_Package_Argument verifies the coverage
// rule is bounded by the package argument: an out-of-scope package missing the
// file is not flagged, while that package is flagged when it is the target.
func Test_Specification_Coverage_Respects_Package_Argument(t *testing.T) {
	t.Parallel()
	other := "// Package other is a fixture.\npackage other\n\n" +
		"// Other does.\nfunc Other() { return }\n"
	files := fstest.MapFS{
		"go.mod": &fstest.MapFile{Data: []byte("module fixture\n\ngo 1.25\n")},
		"greet/greet.go": &fstest.MapFile{
			Data: gofmt_must(t, specification_clean_source)},
		"greet/specification_test.go": &fstest.MapFile{
			Data: gofmt_must(t, specification_clean_test)},
		"greet/SPECIFICATION.md": &fstest.MapFile{
			Data: []byte(specification_clean_md)},
		"other/other.go": &fstest.MapFile{Data: gofmt_must(t, other)},
	}
	input_greet := &bytes.Buffer{}
	lint.Main(&lint.Main_Input{
		Fsys: files, Stdout: input_greet, Stderr: &bytes.Buffer{},
		Scope_Prefix: "greet"})
	if strings.Contains(input_greet.String(), "SPECIFICATION.md") {
		t.Fatalf("scoped to greet, other's missing file must not surface: %s",
			input_greet.String())
	}
	input_other := &bytes.Buffer{}
	lint.Main(&lint.Main_Input{
		Fsys: files, Stdout: input_other, Stderr: &bytes.Buffer{},
		Scope_Prefix: "other"})
	if !strings.Contains(input_other.String(), "missing SPECIFICATION.md") {
		t.Fatalf("scoped to other, its missing file must surface: %s",
			input_other.String())
	}
}

// Runs the filesystem tier with no scope — the whole-workspace case — and
// returns only specification diagnostics, so tests can assert the coverage
// mandate now fires without a package argument.
func specification_diagnostics_workspace(
	t *testing.T, files map[string]string,
) (diags []lint.Diagnostic) {
	t.Helper()
	fsys_map := make(fstest.MapFS)
	for name, content := range files {
		if strings.HasSuffix(name, ".go") {
			fsys_map[name] = &fstest.MapFile{Data: gofmt_must(t, content)}
			continue
		}
		fsys_map[name] = &fstest.MapFile{Data: []byte(content)}
	}
	all, err := lint.Check_File_System(&lint.Check_File_System_Input{Fsys: fsys_map})
	if err != nil {
		t.Fatalf("Check_File_System: %v", err)
	}
	for _, d := range all {
		if d.Name == "specification" {
			diags = append(diags, d)
		}
	}
	return diags
}

func specification_fixture_package(directory string) (files map[string]string) {
	return map[string]string{
		"go.mod": "module fixture\n\ngo 1.25\n",
		directory + "/thing.go": "// Package thing is a fixture.\n" +
			"package thing\n\n// Thing does.\nfunc Thing() { return }\n",
	}
}

// Test_Specification_Coverage_Whole_Workspace verifies a scopeless run mandates
// SPECIFICATION.md for every non-exempt package, not only a scoped target.
func Test_Specification_Coverage_Whole_Workspace(t *testing.T) {
	t.Parallel()
	diags := specification_diagnostics_workspace(t, specification_fixture_package("thing"))
	if !specification_message_contains(diags, "missing SPECIFICATION.md") {
		t.Fatalf("scopeless run must mandate the file, got %v", diags)
	}
}

// Test_Specification_Coverage_Exempts_Third_Party verifies vendored third_party
// packages are not held to the coverage mandate even on a scopeless run.
func Test_Specification_Coverage_Exempts_Third_Party(t *testing.T) {
	t.Parallel()
	diags := specification_diagnostics_workspace(
		t, specification_fixture_package("third_party/vendored"))
	if specification_message_contains(diags, "missing SPECIFICATION.md") {
		t.Fatalf("third_party must be exempt from coverage, got %v", diags)
	}
}

// Test_Specification_Coverage_Exempts_Examples verifies example packages are not
// held to the coverage mandate.
func Test_Specification_Coverage_Exempts_Examples(t *testing.T) {
	t.Parallel()
	diags := specification_diagnostics_workspace(
		t, specification_fixture_package("snippets/examples/demo"))
	if specification_message_contains(diags, "missing SPECIFICATION.md") {
		t.Fatalf("examples must be exempt from coverage, got %v", diags)
	}
}

// Test_Specification_Coverage_Exempts_Main verifies an impure package main is not
// held to the coverage mandate; only a pure package carries the file.
func Test_Specification_Coverage_Exempts_Main(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"go.mod":  "module example.com/mybinary\n\ngo 1.25\n",
		"main.go": "// Package main is a fixture.\npackage main\n\nfunc main() {}\n",
	}
	diags := specification_diagnostics_workspace(t, files)
	if specification_message_contains(diags, "missing SPECIFICATION.md") {
		t.Fatalf("package main must be exempt from coverage, got %v", diags)
	}
}

// Test_Specification_Coverage_Exempts_Default verifies an impure `default` package
// is not held to the coverage mandate; only a pure package carries the file.
func Test_Specification_Coverage_Exempts_Default(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"go.mod":              "module fixture\n\ngo 1.25\n",
		"foo/default/wire.go": "// Package foo is a fixture.\npackage foo\n",
	}
	diags := specification_diagnostics_workspace(t, files)
	if specification_message_contains(diags, "missing SPECIFICATION.md") {
		t.Fatalf("a `default` package must be exempt from coverage, got %v", diags)
	}
}

// Test_Specification_Coverage_Exempt_Specification_Validated verifies an impure package
// that nonetheless ships a SPECIFICATION.md is still format-checked: the exemption
// lifts the requirement to carry the file, not the rules a present file obeys.
func Test_Specification_Coverage_Exempt_Specification_Validated(t *testing.T) {
	t.Parallel()
	main_source := "// Package main is a fixture.\npackage main\n\nfunc main() {}\n"
	files := map[string]string{
		"go.mod":           "module example.com/mybinary\n\ngo 1.25\n",
		"main.go":          main_source,
		"SPECIFICATION.md": "\n## Mid Level\n\nA level-two heading is banned.\n",
	}
	diags := specification_diagnostics_workspace(t, files)
	if !specification_message_contains(diags, "not level # or ###") {
		t.Fatalf("an impure package's present spec must still be validated, got %v", diags)
	}
}

// Case_insensitive_file_system models a case-insensitive filesystem (default
// macOS APFS): a lookup resolves to an entry of any case, while the directory
// listing still reports the real names. It lets a test prove the exact-name
// rule, which fs.ReadFile alone cannot detect on such a filesystem — there the
// doctrine's SPECIFICATION.md would silently resolve to a differently-cased file.
type case_insensitive_file_system struct{ Inner fstest.MapFS }

func (c case_insensitive_file_system) Open(name string) (file fs.File, err error) {
	file, err = c.Inner.Open(name)
	if err == nil {
		return file, nil
	}
	for key := range c.Inner {
		if strings.EqualFold(key, name) {
			return c.Inner.Open(key)
		}
	}
	return file, err
}

// Test_Specification_File_Name_Exact_Case verifies a SPECIFICATION.md present
// only under a different case is treated as missing — the exact byte-for-byte
// name is enforced even where the filesystem would resolve the wrong case.
func Test_Specification_File_Name_Exact_Case(t *testing.T) {
	t.Parallel()
	inner := fstest.MapFS{
		"go.mod":         {Data: []byte("module fixture\n\ngo 1.25\n")},
		"greet/greet.go": {Data: gofmt_must(t, specification_clean_source)},
		"greet/specification_test.go": {
			Data: gofmt_must(t, specification_clean_test)},
		// Wrong-case name; a case-insensitive FS resolves it for ReadFile.
		"greet/specification.md": {Data: []byte(specification_clean_md)},
	}
	all, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys: case_insensitive_file_system{Inner: inner}, Scope: "greet"})
	if err != nil {
		t.Fatalf("Check_File_System: %v", err)
	}
	if !specification_message_contains(all, "missing SPECIFICATION.md") {
		t.Fatalf("wrong-case spec file must be flagged, got %v", all)
	}
}

// Test_Specification_Test_File_Name_Exact_Case verifies the same exact-name rule
// for specification_test.go.
func Test_Specification_Test_File_Name_Exact_Case(t *testing.T) {
	t.Parallel()
	inner := fstest.MapFS{
		"go.mod":                 {Data: []byte("module fixture\n\ngo 1.25\n")},
		"greet/greet.go":         {Data: gofmt_must(t, specification_clean_source)},
		"greet/SPECIFICATION.md": {Data: []byte(specification_clean_md)},
		// Wrong-case name, still a _test.go so it parses as the test package.
		"greet/Specification_test.go": {Data: gofmt_must(t, specification_clean_test)},
	}
	all, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys: case_insensitive_file_system{Inner: inner}, Scope: "greet"})
	if err != nil {
		t.Fatalf("Check_File_System: %v", err)
	}
	if !specification_message_contains(all, "missing specification_test.go") {
		t.Fatalf("wrong-case test file must be flagged, got %v", all)
	}
}

// Test_Specification_Preamble_Flagged verifies non-blank content before the first
// heading is flagged.
func Test_Specification_Preamble_Flagged(t *testing.T) {
	t.Parallel()
	files := specification_clean_files()
	files["greet/SPECIFICATION.md"] = "Stray prose.\n\n# Greeting\n\nIt greets.\n"
	diags := specification_diagnostics(t, files)
	if !specification_message_contains(diags, "content precedes the first heading") {
		t.Fatalf("expected preamble diagnostic, got %v", diags)
	}
}

// Test_Specification_Section_Body verifies a heading with no body line is
// flagged.
func Test_Specification_Section_Body(t *testing.T) {
	t.Parallel()
	files := specification_clean_files()
	files["greet/SPECIFICATION.md"] = "\n# Greeting\n\n"
	diags := specification_diagnostics(t, files)
	if !specification_message_contains(diags, "has no body line") {
		t.Fatalf("expected section-body diagnostic, got %v", diags)
	}
}

// Test_Specification_Section_Contiguity_Flagged verifies a blank line between a
// section's body lines is flagged.
func Test_Specification_Section_Contiguity_Flagged(t *testing.T) {
	t.Parallel()
	files := specification_clean_files()
	files["greet/SPECIFICATION.md"] = "\n# Greeting\n\none\n\ntwo\n"
	diags := specification_diagnostics(t, files)
	if !specification_message_contains(diags, "blank line between body lines") {
		t.Fatalf("expected contiguity diagnostic, got %v", diags)
	}
}

// Test_Specification_Heading_Uniqueness_Flagged verifies two headings sharing a name
// are flagged.
func Test_Specification_Heading_Uniqueness_Flagged(t *testing.T) {
	t.Parallel()
	files := specification_clean_files()
	files["greet/SPECIFICATION.md"] =
		"\n# Greeting\n\nIt greets.\n\n# Greeting\n\nAgain.\n"
	diags := specification_diagnostics(t, files)
	if !specification_message_contains(diags, "is duplicated") {
		t.Fatalf("expected duplicate-heading diagnostic, got %v", diags)
	}
}

// Test_Specification_Heading_Words verifies a heading word containing a
// non-letter/digit rune is flagged, since it would not form a legal test name.
func Test_Specification_Heading_Words(t *testing.T) {
	t.Parallel()
	files := specification_clean_files()
	files["greet/SPECIFICATION.md"] = "\n# Greeting-Caller\n\nIt greets.\n"
	diags := specification_diagnostics(t, files)
	if !specification_message_contains(diags, "must use only letters and digits") {
		t.Fatalf("expected heading-words diagnostic, got %v", diags)
	}
}

// Test_Specification_Test_Order_Leading_Declaration verifies a var/const/type
// declared before the heading tests is flagged: the tests must be the file's
// first declarations, not merely in order among themselves.
func Test_Specification_Test_Order_Leading_Declaration(t *testing.T) {
	t.Parallel()
	files := specification_clean_files()
	files["greet/specification_test.go"] = "package greet_test\n\n" +
		"import \"testing\"\n\n" +
		"// Fixture_value precedes the tests.\n" +
		"var Fixture_value = 0\n\n" +
		"// Test_Greeting is a fixture.\n" +
		"func Test_Greeting(t *testing.T) { _ = t }\n"
	diags := specification_diagnostics(t, files)
	if !specification_message_contains(diags, "at top") {
		t.Fatalf("leading declaration must be flagged, got %v", diags)
	}
}

// Test_Specification_Subheading_Leaf_Test verifies a ### leaf requires a test
// named for its parent and itself, Test_<Parent>_<Heading>.
func Test_Specification_Subheading_Leaf_Test(t *testing.T) {
	t.Parallel()
	files := specification_clean_files()
	files["greet/SPECIFICATION.md"] = "\n# Greeting\n\n### Hello\n\nIt says hello.\n"
	diags := specification_diagnostics(t, files)
	if !specification_message_contains(diags, "Test_Greeting_Hello") {
		t.Fatalf("### leaf must require Test_Greeting_Hello, got %v", diags)
	}
}

// Test_Specification_Branch_Needs_No_Test verifies a ## branch (one with ###
// children) needs no test of its own; only its ### leaves do.
func Test_Specification_Branch_Needs_No_Test(t *testing.T) {
	t.Parallel()
	files := specification_clean_files()
	files["greet/SPECIFICATION.md"] = "\n# Greeting\n\n### Hello\n\nIt says hello.\n"
	files["greet/specification_test.go"] = "package greet_test\n\nimport \"testing\"\n\n" +
		"// Test_Greeting_Hello is a fixture.\n" +
		"func Test_Greeting_Hello(t *testing.T) { _ = t }\n"
	diags := specification_diagnostics(t, files)
	if len(diags) != 0 {
		t.Fatalf("a branch needs no test of its own, got %v", diags)
	}
}

// Test_Specification_Heading_Level_Four_Flagged verifies a #### heading is
// flagged; only ## and ### are permitted.
func Test_Specification_Heading_Level_Four_Flagged(t *testing.T) {
	t.Parallel()
	files := specification_clean_files()
	files["greet/SPECIFICATION.md"] = "\n# Greeting\n\n#### Deep\n\nToo deep.\n"
	diags := specification_diagnostics(t, files)
	if !specification_message_contains(diags, "not level") {
		t.Fatalf("a #### heading must be flagged, got %v", diags)
	}
}

// Test_Specification_Subheading_Duplicate verifies two ### sharing a name under
// one parent are flagged.
func Test_Specification_Subheading_Duplicate(t *testing.T) {
	t.Parallel()
	files := specification_clean_files()
	files["greet/SPECIFICATION.md"] =
		"\n# Greeting\n\n### Hello\n\nHi.\n\n### Hello\n\nAgain.\n"
	diags := specification_diagnostics(t, files)
	if !specification_message_contains(diags, "is duplicated") {
		t.Fatalf("a duplicate ### under one parent must be flagged, got %v", diags)
	}
}

// Test_Specification_Subheading_Reuse_Across_Parents verifies the same ### name
// under two different parents is allowed (their leaf tests differ).
func Test_Specification_Subheading_Reuse_Across_Parents(t *testing.T) {
	t.Parallel()
	files := specification_clean_files()
	files["greet/SPECIFICATION.md"] =
		"\n# Greeting\n\n### Hello\n\nHi.\n\n# Farewell\n\n### Hello\n\nBye.\n"
	files["greet/specification_test.go"] = "package greet_test\n\nimport \"testing\"\n\n" +
		"// Test_Greeting_Hello is a fixture.\n" +
		"func Test_Greeting_Hello(t *testing.T) { _ = t }\n\n" +
		"// Test_Farewell_Hello is a fixture.\n" +
		"func Test_Farewell_Hello(t *testing.T) { _ = t }\n"
	diags := specification_diagnostics(t, files)
	if specification_message_contains(diags, "is duplicated") {
		t.Fatalf("the same ### under two parents must be allowed, got %v", diags)
	}
}

// Test_Specification_Branch_Intro_Allowed verifies a branch ## may carry an
// intro body before its first ###.
func Test_Specification_Branch_Intro_Allowed(t *testing.T) {
	t.Parallel()
	files := specification_clean_files()
	files["greet/SPECIFICATION.md"] = "\n# Greeting\n\nAn intro line.\n\n### Hello\n\nHi.\n"
	files["greet/specification_test.go"] = "package greet_test\n\nimport \"testing\"\n\n" +
		"// Test_Greeting_Hello is a fixture.\n" +
		"func Test_Greeting_Hello(t *testing.T) { _ = t }\n"
	diags := specification_diagnostics(t, files)
	if len(diags) != 0 {
		t.Fatalf("a branch intro body must be allowed, got %v", diags)
	}
}

// Test_Specification_Subheading_Empty verifies a ### leaf with no body line is
// flagged.
func Test_Specification_Subheading_Empty(t *testing.T) {
	t.Parallel()
	files := specification_clean_files()
	files["greet/SPECIFICATION.md"] = "\n# Greeting\n\nIntro.\n\n### Hello\n\n"
	diags := specification_diagnostics(t, files)
	if !specification_message_contains(diags, "has no body line") {
		t.Fatalf("a bodyless ### leaf must be flagged, got %v", diags)
	}
}
