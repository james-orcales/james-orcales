package lint_test

import (
	"bytes"
	"go/format"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/james-orcales/james-orcales/golang_snacks/invariant"
	"github.com/james-orcales/james-orcales/lint/internal"
)

func TestMain(m *testing.M) {
	invariant.RunTestMain(m)
}

// Gofmt_must formats test sources so fixtures don't need to be hand-perfect
// gofmt-clean. Required because check_gofmt runs as part of the tier-1
// pipeline against every test source. TestGofmt builds MapFS inline so it
// can submit deliberately un-formatted sources.
const (
	doctrine_shared_library_go_module = "module github.com/james-orcales/james-orcales/golang_snacks\n"
	doctrine_binary_go_module         = "module example.com/mybinary\n"
)

// Doctrine fixtures must satisfy check_package_documentation_comment
// and check_exported_documentation_comment. A single minimal-content
// builder keeps the doctrine-focused tests from carrying tangential
// boilerplate.
func fixture_package(name string) (source string) {
	return "// Package " + name + " is a fixture.\n" +
		"package " + name + "\n\n" +
		"// Function is a fixture.\n" +
		"func Function() { return }\n"
}

func gofmt_must(t *testing.T, source string) (result []byte) {
	t.Helper()
	formatted, err := format.Source([]byte(source))
	if err != nil {
		t.Fatalf("gofmt_must: %v\nsource:\n%s", err, source)
	}
	return formatted
}

// Test_Variable_Shadow verifies that variables declared in inner scopes that shadow
// outer-scope names are flagged.
func Test_Variable_Shadow(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "shadowing in block",
			Files: map[string]string{
				"test.go": `package main
func f() {
	x := 1
	{
		x := 2
	}
}`,
			},
			Want_Diag: "shadows outer scope variable",
		},
		{
			Name: "shadowing in for loop",
			Files: map[string]string{
				"test.go": `package main
func f() {
	x := 1
	for i := 0; i < 10; i++ {
		x := i
	}
}`,
			},
			Want_Diag: "shadows outer scope variable",
		},
		{
			Name: "shadowing in range",
			Files: map[string]string{
				"test.go": `package main
func f() {
	val := 1
	for _, val := range []int{} {
	}
}`,
			},
			Want_Diag: "shadows outer scope variable",
		},
		{
			Name: "no shadowing",
			Files: map[string]string{
				"test.go": `package main
func f() {
	x := 1
	y := 2
}`,
			},
			Want_Diag: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			fsys_map := make(fstest.MapFS)
			for k, v := range tt.Files {
				fsys_map[k] = &fstest.MapFile{Data: gofmt_must(t, v)}
			}
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			code := lint.Main(&lint.Main_Input{Fsys: fsys_map, Stdout: stdout, Stderr: stderr})

			output := stdout.String()
			if tt.Want_Diag == "" {
				if code != 0 {
					t.Errorf("expected exit 0, got %d; output: %s", code, output)
				}
			} else {
				if !bytes.Contains(stdout.Bytes(), []byte(tt.Want_Diag)) {
					t.Errorf("expected output containing %q, got: %s", tt.Want_Diag, output)
				}
			}
		})
	}
}

// Test_No_Discard verifies the ban on `_ = ...` and all-blank tuple
// assignments, with the mixed-blank and interface-satisfaction exceptions.
func Test_No_Discard(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{Name: "single discard assign", Files: map[string]string{"test.go": `package main
func f() {
	_ = g()
}`}, Want_Diag: "discard"},
		{Name: "two discards short decl", Files: map[string]string{"test.go": `package main
func f() {
	_, _ := g()
}`}, Want_Diag: "discard"},
		{Name: "three discards assign", Files: map[string]string{"test.go": `package main
func f() {
	_, _, _ = g()
}`}, Want_Diag: "discard"},
		{Name: "var blank no type", Files: map[string]string{"test.go": `package main
var _ = g()`}, Want_Diag: "discard"},
		{Name: "mixed lhs short decl allowed", Files: map[string]string{"test.go": `package main
func f() {
	_, x := g()
	_ = x
}`}, Want_Diag: "discard"},
		{Name: "mixed lhs short decl only", Files: map[string]string{"test.go": `package main
func f() (result int) {
	_, x := g()
	return x
}`}, Want_Diag: ""},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			fsys_map := make(fstest.MapFS)
			for k, v := range tt.Files {
				fsys_map[k] = &fstest.MapFile{Data: gofmt_must(t, v)}
			}
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			code := lint.Main(&lint.Main_Input{Fsys: fsys_map, Stdout: stdout, Stderr: stderr})
			output := stdout.String()
			if tt.Want_Diag == "" {
				if code != 0 {
					t.Errorf("expected exit 0, got %d; output: %s", code, output)
				}
				return
			}
			if !bytes.Contains(stdout.Bytes(), []byte(tt.Want_Diag)) {
				t.Errorf("expected output containing %q, got: %s", tt.Want_Diag, output)
			}
		})
	}
}

// Test_Public_Struct_Fields verifies that unexported struct fields (named and
// embedded) are flagged with a rename suggestion.
func Test_Public_Struct_Fields(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "lowercase named field flagged",
			Files: map[string]string{
				"test.go": `package main

type Foo struct {
	bar int
}
`,
			},
			Want_Diag: "rename bar",
		},
		{
			Name: "uppercase named field allowed",
			Files: map[string]string{
				"test.go": `package main

type Foo struct {
	Bar int
}
`,
			},
			Want_Diag: "",
		},
		{
			Name: "embedded lowercase type flagged",
			Files: map[string]string{
				"test.go": `package main

type Foo struct {
	bar
}
`,
			},
			Want_Diag: "rename bar",
		},
		{
			Name: "embedded uppercase type allowed",
			Files: map[string]string{
				"test.go": `package main

type Foo struct {
	Bar
}
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_No_Iota verifies that any use of the iota identifier is flagged.
func Test_No_Iota(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "const iota flagged",
			Files: map[string]string{
				"test.go": `package main

const X = iota
`,
			},
			Want_Diag: "iota is banned",
		},
		{
			Name: "const literal allowed",
			Files: map[string]string{
				"test.go": `package main

const X = 0
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Keyed_Struct_Init verifies that same-file positional struct literals
// are flagged while slice literals and empty struct literals are allowed.
func Test_Keyed_Struct_Init(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "same-file positional flagged",
			Files: map[string]string{
				"test.go": `package main

type Foo struct {
	A int
	B int
}

func make_v() (result Foo) { return Foo{1, 2} }
`,
			},
			Want_Diag: "keyed",
		},
		{
			Name: "same-file keyed allowed",
			Files: map[string]string{
				"test.go": `package main

type Foo struct {
	A int
	B int
}

func make_v() (result Foo) { return Foo{A: 1, B: 2} }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "slice literal allowed",
			Files: map[string]string{
				"test.go": `package main

func make_v() (result []int) { return []int{1, 2, 3} }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "empty struct literal allowed",
			Files: map[string]string{
				"test.go": `package main

type Foo struct {
	A int
}

func make_v() (result Foo) { return Foo{} }
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Gofmt verifies that gofmt-dirty sources are flagged. Bypasses
// gofmt_must so the deliberate formatting violations survive.
func Test_Gofmt(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "indented top-level decl flagged",
			Files: map[string]string{
				"test.go": "package main\n\n\tfunc f() (result int) { return 1 }\n",
			},
			Want_Diag: "gofmt",
		},
		{
			Name: "clean source allowed",
			Files: map[string]string{
				"test.go": "package main\n\nfunc f() (result int) { return 1 }\n",
			},
			Want_Diag: "",
		},
	}
	// TestGofmt bypasses run_diag_table's gofmt_must preprocessing —
	// reformatting the fixture source would defeat the check under test.
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			fsys_map := make(fstest.MapFS)
			for k, v := range tt.Files {
				fsys_map[k] = &fstest.MapFile{Data: []byte(v)}
			}
			stdout := &bytes.Buffer{}
			code := lint.Main(&lint.Main_Input{Fsys: fsys_map, Stdout: stdout, Stderr: &bytes.Buffer{}})
			output := stdout.String()
			if tt.Want_Diag == "" {
				if code != 0 {
					t.Errorf("expected exit 0, got %d; output: %s", code, output)
				}
				return
			}
			if !bytes.Contains(stdout.Bytes(), []byte(tt.Want_Diag)) {
				t.Errorf("expected output containing %q, got: %s", tt.Want_Diag, output)
			}
		})
	}
}

// Test_No_Dot_Import verifies that dot imports (`import . "pkg"`) are flagged
// while regular and named imports are allowed.
func Test_No_Dot_Import(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "dot import flagged",
			Files: map[string]string{
				"test.go": `package main

import . "fmt"
`,
			},
			Want_Diag: "dot import",
		},
		{
			Name: "named import allowed",
			Files: map[string]string{
				"test.go": `package main

import "fmt"
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_No_Function_Init verifies that `func init` is flagged while exported `Init`
// is left alone.
func Test_No_Function_Init(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "func init flagged",
			Files: map[string]string{
				"test.go": `package main

func init() { return }
`,
			},
			Want_Diag: "func init",
		},
		{
			Name: "exported Init allowed",
			Files: map[string]string{
				"test.go": `package main

func Init() { return }
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

func run_diag_table(t *testing.T, tests []struct {
	Name      string
	Files     map[string]string
	Want_Diag string
}) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			fsys_map := make(fstest.MapFS)
			for k, v := range tt.Files {
				fsys_map[k] = &fstest.MapFile{Data: gofmt_must(t, v)}
			}
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			code := lint.Main(&lint.Main_Input{Fsys: fsys_map, Stdout: stdout, Stderr: stderr})
			output := stdout.String()
			if tt.Want_Diag == "" {
				if code != 0 {
					t.Errorf("expected exit 0, got %d; output: %s", code, output)
				}
				return
			}
			if !bytes.Contains(stdout.Bytes(), []byte(tt.Want_Diag)) {
				t.Errorf("expected output containing %q, got: %s", tt.Want_Diag, output)
			}
		})
	}
}

// Doctrine fixtures include non-Go files (go.mod) and benefit from
// expressing per-file expectations (a fixture often must lint clean
// against everything *except* the doctrine rule under test, since the
// general-purpose run_diag_table substring match would otherwise pick
// up unrelated diagnostics). Want_Diags == nil means the run must exit
// 0; an empty slice means the run must exit nonzero but no specific
// substring is asserted.
func run_doctrine_diag_table(t *testing.T, tests []struct {
	Name       string
	Files      map[string]string
	Want_Diags []string
	Forbid     []string
}) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			fsys_map := make(fstest.MapFS)
			for k, v := range tt.Files {
				if strings.HasSuffix(k, ".go") {
					fsys_map[k] = &fstest.MapFile{Data: gofmt_must(t, v)}
					continue
				}
				fsys_map[k] = &fstest.MapFile{Data: []byte(v)}
			}
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			code := lint.Main(&lint.Main_Input{Fsys: fsys_map, Stdout: stdout, Stderr: stderr})
			output := stdout.String()
			if tt.Want_Diags == nil {
				if code != 0 {
					t.Errorf("expected exit 0, got %d; output:\n%s", code, output)
				}
				return
			}
			if code == 0 {
				t.Errorf("expected nonzero exit; output:\n%s", output)
			}
			for _, w := range tt.Want_Diags {
				if !bytes.Contains(stdout.Bytes(), []byte(w)) {
					t.Errorf("expected output containing %q; got:\n%s", w, output)
				}
			}
			for _, f := range tt.Forbid {
				if bytes.Contains(stdout.Bytes(), []byte(f)) {
					t.Errorf("expected output NOT to contain %q; got:\n%s", f, output)
				}
			}
		})
	}
}

// Test_Banned_Identifiers verifies that function names containing banned
// segments (helper, util*) are flagged, with case-insensitive segment matching
// and no false positives on substrings like "helpme".
func Test_Banned_Identifiers(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "bare helper flagged",
			Files: map[string]string{
				"test.go": `package main

func helper() { return }
`,
			},
			Want_Diag: "banned word 'helper'",
		},
		{
			Name: "suffix helper flagged",
			Files: map[string]string{
				"test.go": `package main

func read_helper() { return }
`,
			},
			Want_Diag: "banned word 'helper'",
		},
		{
			Name: "capitalized Helper flagged",
			Files: map[string]string{
				"test.go": `package main

func Helper_Func() { return }
`,
			},
			Want_Diag: "banned word 'helper'",
		},
		{
			Name: "helper as middle segment flagged",
			Files: map[string]string{
				"test.go": `package main

func read_helper_thing() { return }
`,
			},
			Want_Diag: "banned word 'helper'",
		},
		{
			Name: "no helper segment allowed",
			Files: map[string]string{
				"test.go": `package main

func read_sector() { return }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "helpme not flagged",
			Files: map[string]string{
				"test.go": `package main

func helpme() { return }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "function name util flagged",
			Files: map[string]string{
				"test.go": `package main

func parse_util() { return }
`,
			},
			Want_Diag: "banned word 'util'",
		},
		{
			Name: "function name Utilities flagged",
			Files: map[string]string{
				"test.go": `package main

func Make_Utilities() { return }
`,
			},
			Want_Diag: "banned word 'utilities'",
		},
	}
	run_diag_table(t, tests)
}

// Test_Banned_Decl_Sites_Local_Vars verifies that the universal banned-segment
// check reaches local var/const/range/assign-define declarations. len(...)
// and cap(...) call sites are exempt because they appear in callee position,
// never as a declared name.
func Test_Banned_Declaration_Sites_Local_Vars(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "var name length flagged",
			Files: map[string]string{
				"test.go": `package main

func F() (result int) {
	length := 0
	return length
}
`,
			},
			Want_Diag: "banned word 'length'",
		},
		{
			Name: "const name length flagged",
			Files: map[string]string{
				"test.go": `package main

const max_length = 10
`,
			},
			Want_Diag: "banned word 'length'",
		},
		{
			Name: "range key length flagged",
			Files: map[string]string{
				"test.go": `package main

func F(xs []int) (result int) {
	for length, v := range xs {
		result =length + v
	}
	return result
}
`,
			},
			Want_Diag: "banned word 'length'",
		},
		{
			Name: "var with utils segment flagged",
			Files: map[string]string{
				"test.go": `package main

func F() (result int) {
	utils_count := 0
	return utils_count
}
`,
			},
			Want_Diag: "banned word 'utils'",
		},
		{
			Name: "len builtin call site exempt",
			Files: map[string]string{
				"test.go": `package main

func F(xs []int) (result int) { return len(xs) }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "cap builtin call site exempt",
			Files: map[string]string{
				"test.go": `package main

func F(xs []int) (result int) { return cap(xs) }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "clean local decls allowed",
			Files: map[string]string{
				"test.go": `package main

func F(buffer []byte) (result int) {
	n := 0
	return n
}
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Banned_Declaration_Sites_Signatures verifies that the universal banned-segment
// check reaches parameter, named-return, and struct-field declarations.
func Test_Banned_Declaration_Sites_Signatures(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "param name with len segment flagged",
			Files: map[string]string{
				"test.go": `package main

func F(buf_len int) (result int) { return buf_len }
`,
			},
			Want_Diag: "banned word 'len'",
		},
		{
			Name: "named return with length segment flagged",
			Files: map[string]string{
				"test.go": `package main

func F() (length int) { return 0 }
`,
			},
			Want_Diag: "banned word 'length'",
		},
		{
			Name: "struct field with Length segment flagged",
			Files: map[string]string{
				"test.go": `package main

type S struct{ Buf_Length int }
`,
			},
			Want_Diag: "banned word 'length'",
		},
		{
			Name: "clean signature allowed",
			Files: map[string]string{
				"test.go": `package main

type S struct{ Count int }

func F(buffer []byte) (result int) { return 0 }
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Naming_For_Loop verifies that a C-style for-loop induction variable
// (`for i := 0; i < N; i++`) must be named with an _index suffix.
func Test_Naming_For_Loop(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "bare i induction flagged",
			Files: map[string]string{
				"test.go": `package main

func F(n int) (result int) {
	for i := 0; i < n; i++ {
		result = result + i
	}
	return result
}
`,
			},
			Want_Diag: "i (used as index) → rename to i_index",
		},
		{
			Name: "i_index induction allowed",
			Files: map[string]string{
				"test.go": `package main

func F(n int) (result int) {
	for i_index := 0; i_index < n; i_index++ {
		result = result + i_index
	}
	return result
}
`,
			},
			Want_Diag: "",
		},
		{
			Name: "non-induction for-loop not triggered",
			Files: map[string]string{
				"test.go": `package main

func F() (result int) {
	for condition := true; condition; condition = false {
		result = result + 1
	}
	return result
}
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Naming_Make verifies that make's count/size argument, when a bare
// declared identifier, must carry _count (or _size when the element type
// is byte).
func Test_Naming_Make(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "make slice of T with bare n flagged",
			Files: map[string]string{
				"test.go": `package main

func F(n int) (result []int) {
	result =make([]int, n)
	return result
}
`,
			},
			Want_Diag: "n (used as count) → rename to n_count",
		},
		{
			Name: "make slice of byte with bare n flagged as size",
			Files: map[string]string{
				"test.go": `package main

func F(n int) (result []byte) {
	result =make([]byte, n)
	return result
}
`,
			},
			Want_Diag: "n (used as size) → rename to n_size",
		},
		{
			Name: "make map with bare n flagged as count",
			Files: map[string]string{
				"test.go": `package main

func F(n int) (result map[string]int) {
	result =make(map[string]int, n)
	return result
}
`,
			},
			Want_Diag: "n (used as count) → rename to n_count",
		},
		{
			Name: "make with suffixed arg allowed",
			Files: map[string]string{
				"test.go": `package main

func F(n_count int) (result []int) {
	result =make([]int, n_count)
	return result
}
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Naming_Stdlib_Allowlist verifies that the result of allowlisted
// stdlib calls (strings.Index → offset, binary.Size → size, etc.) requires
// the matching suffix on its assignment target.
func Test_Naming_Stdlib_Allowlist(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "strings.Index result requires _offset",
			Files: map[string]string{
				"test.go": `package main

import "strings"

func F(s string) (result int) {
	pos := strings.Index(s, "x")
	return pos
}
`,
			},
			Want_Diag: "pos (used as offset) → rename to pos_offset",
		},
		{
			Name: "binary.Size result requires _size",
			Files: map[string]string{
				"test.go": `package main

import "encoding/binary"

func F(v int32) (result int) {
	n := binary.Size(v)
	return n
}
`,
			},
			Want_Diag: "n (used as size) → rename to n_size",
		},
		{
			Name: "method .Len() requires _size",
			Files: map[string]string{
				"test.go": `package main

import "bytes"

func F(b *bytes.Buffer) (result int) {
	n := b.Len()
	return n
}
`,
			},
			Want_Diag: "n (used as size) → rename to n_size",
		},
	}
	run_diag_table(t, tests)
}

// Test_Naming_Element_Count_Result verifies that v := len(x) / v := cap(x) require
// v to carry a _count or _size suffix (either accepted; size is the
// stylistic override when the count is in bytes).
func Test_Naming_Element_Count_Result(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "bare n from len flagged",
			Files: map[string]string{
				"test.go": `package main

func F(xs []int) (result int) {
	n := len(xs)
	return n
}
`,
			},
			Want_Diag: "n (used as count or size) → rename to n_count",
		},
		{
			Name: "bare n from cap flagged",
			Files: map[string]string{
				"test.go": `package main

func F(xs []int) (result int) {
	n := cap(xs)
	return n
}
`,
			},
			Want_Diag: "n (used as count or size) → rename to n_count",
		},
		{
			Name: "n_count from len allowed",
			Files: map[string]string{
				"test.go": `package main

func F(xs []int) (result int) {
	n_count := len(xs)
	return n_count
}
`,
			},
			Want_Diag: "",
		},
		{
			Name: "n_size from len allowed",
			Files: map[string]string{
				"test.go": `package main

func F(buffer []byte) (result int) {
	n_size := len(buffer)
	return n_size
}
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Naming_Rename_Suggestion verifies the replace-or-append fix logic:
// names with no existing terminology segment get the term appended; tier-1
// bans `length` and `len` as segments so the replace branch is exercised
// via that path (a name like `buf_length` is rejected by tier 1 and the
// terminology rename, when triggered, replaces the offending segment).
func Test_Naming_Rename_Suggestion(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "bare ident gets term appended",
			Files: map[string]string{
				"test.go": `package main

func F(n int) (result int) {
	for i := 0; i < n; i++ {
		result = result + i
	}
	return result
}
`,
			},
			Want_Diag: "i (used as index) → rename to i_index",
		},
		{
			Name: "Ada_Case preserved in suggestion",
			Files: map[string]string{
				"test.go": `package main

func F(N int) (result int) {
	for I := 0; I < N; I++ {
		result = result + I
	}
	return result
}
`,
			},
			Want_Diag: "I (used as index) → rename to I_Index",
		},
	}
	run_diag_table(t, tests)
}

// Test_Naming_Arithmetic verifies the tier-3 operand-suffix invariant:
// when both operands of `+` or `-` carry recognized suffixes, the
// combination must match the table (_index - _index = _count, etc.).
// Mismatched assignment LHS suffix is also flagged.
func Test_Naming_Arithmetic(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "offset plus count is incoherent",
			Files: map[string]string{
				"test.go": `package main

func F() (result int) {
	a_offset := 0
	b_count := 0
	return a_offset + b_count
}
`,
			},
			Want_Diag: "_offset + _count",
		},
		{
			Name: "index minus index assigned to bare flagged",
			Files: map[string]string{
				"test.go": `package main

func F() (result int) {
	a_index := 0
	b_index := 0
	pos := a_index - b_index
	return pos
}
`,
			},
			Want_Diag: "pos",
		},
		{
			Name: "valid index minus index assigned to count allowed",
			Files: map[string]string{
				"test.go": `package main

func F() (result int) {
	a_index := 0
	b_index := 0
	position_count := a_index - b_index
	return position_count
}
`,
			},
			Want_Diag: "",
		},
		{
			Name: "valid offset plus size assigned to offset allowed",
			Files: map[string]string{
				"test.go": `package main

func F() (result int) {
	a_offset := 0
	b_size := 0
	position_offset := a_offset + b_size
	return position_offset
}
`,
			},
			Want_Diag: "",
		},
		{
			Name: "unsuffixed operands skipped",
			Files: map[string]string{
				"test.go": `package main

func F() (result int) {
	a := 0
	b := 0
	return a + b
}
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Naming_Abbreviations_Flagged covers identifiers containing tokenized
// words from the abbreviation denylist — single-candidate (`cfg`), ambiguous
// multi-candidate (`res`), and various declaration sites (type, field, func).
func Test_Naming_Abbreviations_Flagged(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "var with cfg word flagged single candidate",
			Files: map[string]string{
				"test.go": `package main

func F() (x int) {
	cfg_path := 0
	return cfg_path
}
`,
			},
			Want_Diag: `rename cfg_path -> config_path`,
		},
		{
			Name: "var with ambiguous res flagged with candidates",
			Files: map[string]string{
				"test.go": `package main

func F() (x int) {
	user_res := 0
	return user_res
}
`,
			},
			Want_Diag: `rename user_res -> user_response,user_result,user_resource,user_reserve`,
		},
		{
			Name: "type name with mgr flagged",
			Files: map[string]string{
				"test.go": `package main

type Pool_Mgr struct {
	Items int
}
`,
			},
			Want_Diag: `rename Pool_Mgr -> Pool_Manager`,
		},
		{
			Name: "struct field with btn flagged",
			Files: map[string]string{
				"test.go": `package main

type Form struct {
	Submit_Btn int
}
`,
			},
			Want_Diag: `rename Submit_Btn -> Submit_Button`,
		},
		{
			Name: "func name with cb flagged",
			Files: map[string]string{
				"test.go": `package main

func Run_Cb() (x int) { return 0 }
`,
			},
			Want_Diag: `rename Run_Cb -> Run_Callback`,
		},
	}
	run_diag_table(t, tests)
}

// Test_Naming_Abbreviations_Exempt covers identifiers that look like
// abbreviations but are exempt: Go-language idioms (err, ctx, fmt, len, cap,
// min, max), single-letter loop counters (Tiger Style sort/matrix primitives),
// and clean code with no abbreviation hits.
func Test_Naming_Abbreviations_Exempt(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "err exempt",
			Files: map[string]string{
				"test.go": `package main

import "errors"

func F() (x int) {
	err := errors.New("e")
	if err != nil {
		return 1
	}
	return 0
}
`,
			},
			Want_Diag: "",
		},
		{
			Name: "single-letter loop counter exempt",
			Files: map[string]string{
				"test.go": `package main

func F(n int) (x int) {
	for i_index := 0; i_index < n; i_index++ {
		x = x + i_index
	}
	return x
}
`,
			},
			Want_Diag: "",
		},
		{
			Name: "clean code no abbreviation hit",
			Files: map[string]string{
				"test.go": `package main

func Compute(value int) (result int) {
	total := value + 1
	return total
}
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Naming_Participles verifies that declared identifiers whose final
// tokenized word ends in "ing" and isn't in nouns_suffixed_by_ing are
// flagged. Gerund-nouns (String, Mapping, Encoding, etc.) and the
// Stringer interface's String method are not flagged.
func Test_Naming_Participles(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "func name preparing flagged",
			Files: map[string]string{
				"test.go": `package main

func Preparing() (x int) { return 0 }
`,
			},
			Want_Diag: `present participle "preparing"`,
		},
		{
			Name: "type name processing flagged",
			Files: map[string]string{
				"test.go": `package main

type Data_Processing struct {
	Items int
}
`,
			},
			Want_Diag: `present participle "processing"`,
		},
		{
			Name: "var name rendering flagged",
			Files: map[string]string{
				"test.go": `package main

func F() (x int) {
	page_rendering := 0
	return page_rendering
}
`,
			},
			Want_Diag: `present participle "rendering"`,
		},
		{
			Name: "Mapping allowed gerund-noun",
			Files: map[string]string{
				"test.go": `package main

type Key_Mapping struct {
	Entries int
}
`,
			},
			Want_Diag: "",
		},
		{
			Name: "Encoding allowed gerund-noun",
			Files: map[string]string{
				"test.go": `package main

type Wire_Encoding struct {
	Bytes int
}
`,
			},
			Want_Diag: "",
		},
		{
			Name: "String method on type allowed via noun allowlist",
			Files: map[string]string{
				"test.go": `package main

type Color int

func (c Color) String() (result string) { return "" }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "no -ing suffix not flagged",
			Files: map[string]string{
				"test.go": `package main

func Compute(value int) (result int) { return value }
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Test_Document_Comment verifies that every Test_* function in a _test.go file
// has a doc comment, with TestMain exempt and non-Test functions unaffected.
func Test_Test_Document_Comment(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "test without doc flagged",
			Files: map[string]string{
				"foo_test.go": `package foo_test

import "testing"

func Test_Foo(t *testing.T) { t.Helper() }
`,
			},
			Want_Diag: "test Test_Foo is missing a doc comment",
		},
		{
			Name: "test with doc allowed",
			Files: map[string]string{
				"foo_test.go": `package foo_test

import "testing"

// Test_Foo verifies foo behaves correctly.
func Test_Foo(t *testing.T) { t.Helper() }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "TestMain exempt",
			Files: map[string]string{
				"foo_test.go": `package foo_test

import "testing"

func TestMain(m *testing.M) { m.Run() }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "non-test func in test file unaffected",
			Files: map[string]string{
				"foo_test.go": `package foo_test

func inner() { return }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "Test-prefixed func in non-test file unaffected",
			Files: map[string]string{
				"foo.go": `package main

func Test_Like_Name() { return }
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Package_Split_Threshold verifies that packages with multiple files
// under the 10K-line threshold are flagged, with source / test / build-tag
// groups counted separately, build-tag detection done via go/build/constraint.
func Test_Package_Split_Threshold(t *testing.T) {
	source_a := "// Package foo is a fixture.\npackage foo\n\n// A is a fixture.\nfunc A() { return }\n"
	source_b := "package foo\n\n// B is a fixture.\nfunc B() { return }\n"
	source_c := "package foo\n\n// C is a fixture.\nfunc C() { return }\n"
	test_a := "package foo_test\n\nimport \"testing\"\n\n" +
		"// Test_A verifies A.\nfunc Test_A(t *testing.T) { t.Helper() }\n"
	test_b := "package foo_test\n\nimport \"testing\"\n\n" +
		"// Test_B verifies B.\nfunc Test_B(t *testing.T) { t.Helper() }\n"
	linux_a := "//go:build linux\n\n// Package foo is a fixture.\npackage foo\n\n// A is a fixture.\nfunc A() { return }\n"
	linux_b := "//go:build linux\n\npackage foo\n\n// B is a fixture.\nfunc B() { return }\n"
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name:      "two small source files in same pkg flagged",
			Files:     map[string]string{"a.go": source_a, "b.go": source_b},
			Want_Diag: "has 2 source files",
		},
		{
			Name:      "single small source file allowed",
			Files:     map[string]string{"a.go": source_a},
			Want_Diag: "",
		},
		{
			Name:      "source and test files counted separately",
			Files:     map[string]string{"a.go": source_a, "a_test.go": test_a},
			Want_Diag: "",
		},
		{
			Name:      "two test files in same pkg flagged",
			Files:     map[string]string{"a.go": source_a, "a_test.go": test_a, "b_test.go": test_b},
			Want_Diag: "has 2 test files",
		},
		{
			Name:      "build tag splits source group",
			Files:     map[string]string{"a.go": source_a, "a_linux.go": linux_b},
			Want_Diag: "",
		},
		{
			Name:      "two files sharing a build tag flagged",
			Files:     map[string]string{"a.go": linux_a, "a_extra.go": linux_b, "plain.go": source_c},
			Want_Diag: "has 2 source files",
		},
	}
	run_diag_table(t, tests)
}

// Test_Snap_Backtick verifies that the first argument to snap.Init / snap.Edit
// must be a backticked raw string literal; double-quoted string literals are
// flagged, non-literal arguments are unaffected.
func Test_Snap_Backtick(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "snap.Init with double-quoted string flagged",
			Files: map[string]string{
				"test.go": "package main\n\nimport \"x/snap\"\n\nfunc f() { snap.Init(\"foo\") }\n",
			},
			Want_Diag: "snap.Init must use a backticked",
		},
		{
			Name: "snap.Edit with double-quoted string flagged",
			Files: map[string]string{
				"test.go": "package main\n\nimport \"x/snap\"\n\nfunc f() { snap.Edit(\"foo\") }\n",
			},
			Want_Diag: "snap.Edit must use a backticked",
		},
		{
			Name: "snap.Init with backticked string allowed",
			Files: map[string]string{
				"test.go": "package main\n\nimport \"x/snap\"\n\nfunc f() { snap.Init(`foo`) }\n",
			},
			Want_Diag: "",
		},
		{
			Name: "snap.Other not in scope",
			Files: map[string]string{
				"test.go": "package main\n\nimport \"x/snap\"\n\nfunc f() { snap.Other(\"foo\") }\n",
			},
			Want_Diag: "",
		},
		{
			Name: "non-literal arg allowed",
			Files: map[string]string{
				"test.go": "package main\n\nimport \"x/snap\"\n\nfunc f(s string) { snap.Init(s) }\n",
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Banned_Package_And_File_Names verifies that the universal banned
// segments (util*) are flagged in package names and file names too.
func Test_Banned_Package_And_File_Names(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "package name utils flagged",
			Files: map[string]string{
				"test.go": `package utils

func f() (result int) { return 1 }
`,
			},
			Want_Diag: "banned word 'utils'",
		},
		{
			Name: "package name with util segment flagged",
			Files: map[string]string{
				"test.go": `package string_util

func f() (result int) { return 1 }
`,
			},
			Want_Diag: "banned word 'util'",
		},
		{
			Name: "file name utils.go flagged",
			Files: map[string]string{
				"utils.go": `package main

func f() (result int) { return 1 }
`,
			},
			Want_Diag: "banned word 'utils'",
		},
		{
			Name: "file name with utility segment flagged",
			Files: map[string]string{
				"string_utility.go": `package main

func f() (result int) { return 1 }
`,
			},
			Want_Diag: "banned word 'utility'",
		},
		{
			Name: "clean file and package allowed",
			Files: map[string]string{
				"sector.go": `// Package sector is a fixture.
package sector

func read() { return }
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Input_Struct verifies that functions with ≥2 same-type parameters must
// accept a single *<Func>_Input pointer, with snake/Ada casing matching and
// variadics preserved in the suggested signature.
func Test_Input_Struct(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "two ints without input struct flagged",
			Files: map[string]string{
				"test.go": `package main

func F(a, b int) (result int) { return a + b }
`,
			},
			Want_Diag: "convert to F(*F_Input) (result int)",
		},
		{
			Name: "input struct directly above allowed",
			Files: map[string]string{
				"test.go": `package main

type F_Input struct {
	A int
	B int
}

func F(input *F_Input) (result int) { return input.A + input.B }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "two separate int fields flagged",
			Files: map[string]string{
				"test.go": `package main

func F(a int, b int) (result int) { return a + b }
`,
			},
			Want_Diag: "convert to F(*F_Input) (result int)",
		},
		{
			Name: "variadic preserved in suggestion",
			Files: map[string]string{
				"test.go": `package main

func F(a, b int, extra ...string) (result int) { return a + b + len(extra) }
`,
			},
			Want_Diag: "convert to F(*F_Input, extra ...string) (result int)",
		},
		{
			Name: "snake case function snake case input",
			Files: map[string]string{
				"test.go": `package main

type f_input struct {
	A int
	B int
}

func f(input *f_input) (result int) { return input.A + input.B }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "variadic only does not trigger",
			Files: map[string]string{
				"test.go": `package main

func F(args ...int) (result int) { return len(args) }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "mixed types do not trigger",
			Files: map[string]string{
				"test.go": `package main

func F(a int, b string) (result int) { return a + len(b) }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "single param does not trigger",
			Files: map[string]string{
				"test.go": `package main

func F(a int) (result int) { return a }
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_No_Empty_Function_Body verifies that empty-body functions and methods are
// flagged while interface method signatures (Body == nil) are allowed.
func Test_No_Empty_Function_Body(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "empty function flagged",
			Files: map[string]string{
				"test.go": `package main

func F() {}
`,
			},
			Want_Diag: "func F has an empty body",
		},
		{
			Name: "empty method flagged",
			Files: map[string]string{
				"test.go": `package main

type T struct{ X int }

func (T) M() {}
`,
			},
			Want_Diag: "func M has an empty body",
		},
		{
			Name: "non-empty body allowed",
			Files: map[string]string{
				"test.go": `package main

func F() { return }
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_No_Interfaces verifies that interface types with method elements are
// flagged wherever they appear (TypeSpec, inline in signatures, type
// assertions). Type-element-only interfaces (generic constraints built from
// unions and approximations) are allowed because they carry no method set.
// `any` and bare `interface{}` are allowed as the empty interface.
func Test_No_Interfaces(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "named method-set interface flagged",
			Files: map[string]string{
				"test.go": `package main

func main() (result int) { return 0 }

type Iface interface {
	M()
}
`,
			},
			Want_Diag: "interface method sets are banned",
		},
		{
			Name: "inline interface in parameter flagged",
			Files: map[string]string{
				"test.go": `package main

func main() (result int) { return 0 }

func F(x interface{ M() }) (result int) { return 0 }
`,
			},
			Want_Diag: "interface method sets are banned",
		},
		{
			Name: "type assertion with method-set interface flagged",
			Files: map[string]string{
				"test.go": `package main

func main() (result int) {
	var x any = 0
	_, _ = x.(interface{ M() })
	return 0
}
`,
			},
			Want_Diag: "interface method sets are banned",
		},
		{
			Name: "any parameter allowed",
			Files: map[string]string{
				"test.go": `package main

func main() (result int) { return F(0) }

func F(x any) (result int) { return 0 }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "generic type-constraint interface allowed",
			Files: map[string]string{
				"test.go": `package main

func main() (result int) { return F(0) }

type _Number interface {
	~int | ~int64
}

func F[T _Number](x T) (result int) { return 0 }
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Unnecessary_Method verifies that receiver methods whose name+signature
// does not match a stdlib interface method are flagged with a recommendation
// to convert to a free function. The set of legal targets is the closed
// stdlib table; third-party interface satisfaction is rejected by policy.
// Test_Unnecessary_Method_Allowed covers receiver methods whose name and
// signature match an entry in the stdlib interface table — these must pass
// without diagnostic. Free functions are also covered as a control.
func Test_Unnecessary_Method_Allowed(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "stdlib Stringer match allowed",
			Files: map[string]string{
				"test.go": `package main

func main() (result int) { return 0 }

type T struct{ X int }

func (t T) String() (result string) { return "" }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "stdlib error match allowed",
			Files: map[string]string{
				"test.go": `package main

func main() (result int) { return 0 }

type T struct{ X int }

func (t T) Error() (result string) { return "" }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "stdlib Read match allowed",
			Files: map[string]string{
				"test.go": `package main

func main() (result int) { return 0 }

type T struct{ X int }

func (t T) Read(p []byte) (n int, err error) { return 0, nil }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "stdlib Scan with any allowed",
			Files: map[string]string{
				"test.go": `package main

func main() (result int) { return 0 }

type T struct{ X int }

func (t T) Scan(x any) (err error) { return nil }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "free function ignored",
			Files: map[string]string{
				"test.go": `package main

func main() (result int) { return F() }

func F() (result int) { return 0 }
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Unnecessary_Method_Flagged covers receiver methods that the stdlib
// table does not cover — either name unknown or signature wrong-shape — and
// must be flagged with the "convert to free function" instruction.
func Test_Unnecessary_Method_Flagged(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "Write with wrong result list flagged",
			Files: map[string]string{
				"test.go": `package main

func main() (result int) { return 0 }

type T struct{ X int }

func (t T) Write(p []byte) (err error) { return nil }
`,
			},
			Want_Diag: "does not satisfy any stdlib interface",
		},
		{
			Name: "unknown method name flagged",
			Files: map[string]string{
				"test.go": `package main

func main() (result int) { return 0 }

type T struct{ X int }

func (t T) Foo() (result int) { return 0 }
`,
			},
			Want_Diag: "does not satisfy any stdlib interface",
		},
	}
	run_diag_table(t, tests)
}

// Test_Test_Package verifies that _test.go files must declare
// `package <X>_test`; main, main_test, and whitebox packages are flagged.
func Test_Test_Package(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "whitebox package flagged",
			Files: map[string]string{
				"foo_test.go": `package foo

func f() (result int) { return 1 }
`,
			},
			Want_Diag: "test file must declare",
		},
		{
			Name: "package main flagged",
			Files: map[string]string{
				"foo_test.go": `package main

func f() (result int) { return 1 }
`,
			},
			Want_Diag: "test file must declare",
		},
		{
			Name: "package main_test flagged",
			Files: map[string]string{
				"foo_test.go": `package main_test

func f() (result int) { return 1 }
`,
			},
			Want_Diag: "test file must declare",
		},
		{
			Name: "blackbox _test package allowed",
			Files: map[string]string{
				"foo_test.go": `package foo_test

func f() (result int) { return 1 }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "non-test file unaffected",
			Files: map[string]string{
				"foo.go": `package main

func f() (result int) { return 1 }
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_No_Recursion verifies that direct and mutual recursion cycles are
// detected via the per-file Ident-based call graph.
func Test_No_Recursion(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{Name: "direct recursion", Files: map[string]string{"test.go": `package main
func f() {
	f()
}`}, Want_Diag: "recursion"},
		{Name: "recursion inside nested block", Files: map[string]string{"test.go": `package main
func f(x int) (result int) {
	if x > 0 {
		return f(x - 1)
	}
	return 0
}`}, Want_Diag: "recursion"},
		{Name: "no recursion: calls other function", Files: map[string]string{"test.go": `package main
func f_inner() { return }
func f() {
	f_inner()
}`}, Want_Diag: ""},
		{Name: "no recursion: no calls", Files: map[string]string{"test.go": `package main
func f() { return }`}, Want_Diag: ""},
		{Name: "mutual 2-cycle", Files: map[string]string{"test.go": `package main
func entry() { a(); b() }
func a() { b() }
func b() { a() }`}, Want_Diag: "cycle"},
		{Name: "3-cycle", Files: map[string]string{"test.go": `package main
func entry() { a(); b(); c() }
func a() { b() }
func b() { c() }
func c() { a() }`}, Want_Diag: "cycle"},
		{Name: "non-cycle chain", Files: map[string]string{"test.go": `package main
func a() { a_b() }
func a_b() { a_b_c() }
func a_b_c() { return }`}, Want_Diag: ""},
		{Name: "cycle plus non-cycle helper", Files: map[string]string{"test.go": `package main
func entry() { a(); b(); inner() }
func a() { b(); inner() }
func b() { a() }
func inner() { return }`}, Want_Diag: "cycle"},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			fsys_map := make(fstest.MapFS)
			for k, v := range tt.Files {
				fsys_map[k] = &fstest.MapFile{Data: gofmt_must(t, v)}
			}
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			code := lint.Main(&lint.Main_Input{Fsys: fsys_map, Stdout: stdout, Stderr: stderr})

			output := stdout.String()
			if tt.Want_Diag == "" {
				if code != 0 {
					t.Errorf("expected exit 0, got %d; output: %s", code, output)
				}
			} else {
				if !bytes.Contains(stdout.Bytes(), []byte(tt.Want_Diag)) {
					t.Errorf("expected output containing %q, got: %s", tt.Want_Diag, output)
				}
			}
		})
	}
}

// Test_Tiered_Checks verifies that when any tier-1 diagnostic fires, tier-2
// checks (currently recursion) are suppressed so they can safely rely on
// tier-1 contracts.
func Test_Tiered_Checks(t *testing.T) {
	// When any tier-1 diagnostic fires, tier-2 checks (currently just recursion)
	// must be skipped so they can rely on tier-1 contracts (e.g. check_shadowing).
	fsys_map := fstest.MapFS{
		"test.go": &fstest.MapFile{Data: []byte(
			"// missing period at end of this comment\n" +
				"package main\n" +
				"func main() { return }\n" +
				"func f() { f() }\n",
		)},
	}
	stdout := &bytes.Buffer{}
	code := lint.Main(&lint.Main_Input{Fsys: fsys_map, Stdout: stdout, Stderr: &bytes.Buffer{}})
	if code == 0 {
		t.Fatalf("expected non-zero exit due to tier-1 diagnostic, got 0; output: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("should end with")) {
		t.Errorf("expected tier-1 (comment) diagnostic, got: %s", stdout.String())
	}
	if bytes.Contains(stdout.Bytes(), []byte("recursion")) {
		t.Errorf("tier-2 (recursion) diagnostic should be suppressed when tier-1 fails, got: %s", stdout.String())
	}
}

// Test_Main_First verifies that main, Main, and TestMain must be the first
// function declaration in the file, with TestMain exempt from snake/Ada casing.
func Test_Main_First(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{Name: "main not first", Files: map[string]string{"test.go": `package main
func other() { return }
func main() { return }`}, Want_Diag: "should be declared first"},
		{Name: "Main not first", Files: map[string]string{"test.go": `package impl
func helper() { return }
func Main() { return }`}, Want_Diag: "should be declared first"},
		{Name: "main is first", Files: map[string]string{"test.go": `package main
func main() { return }
func other() { return }`}, Want_Diag: ""},
		{Name: "Main is first", Files: map[string]string{"test.go": `// Package impl is a fixture.
package impl
// Main is the entry point.
func Main() { return }
func inner() { return }`}, Want_Diag: ""},
		{Name: "no main: no diag", Files: map[string]string{"test.go": `// Package x is a fixture.
package x
func a() { return }
func b() { return }`}, Want_Diag: ""},
		{Name: "decls before main: still OK", Files: map[string]string{"test.go": `package main
const x = 1
type T int
func main() { return }`}, Want_Diag: ""},
		{Name: "TestMain not first", Files: map[string]string{"foo_test.go": `package foo_test

import "testing"

// Test_Foo is a fixture.
func Test_Foo(t *testing.T) { return }
func TestMain(m *testing.M) { return }
`}, Want_Diag: "func TestMain should be declared first"},
		{Name: "TestMain is first", Files: map[string]string{"foo_test.go": `package foo_test

import "testing"

func TestMain(m *testing.M) { return }
// Test_Foo is a fixture.
func Test_Foo(t *testing.T) { return }
`}, Want_Diag: ""},
		{Name: "TestMain exempt from casing", Files: map[string]string{"foo_test.go": `package foo_test

import "testing"

func TestMain(m *testing.M) { return }
`}, Want_Diag: ""},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			fsys_map := make(fstest.MapFS)
			for k, v := range tt.Files {
				fsys_map[k] = &fstest.MapFile{Data: gofmt_must(t, v)}
			}
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			code := lint.Main(&lint.Main_Input{Fsys: fsys_map, Stdout: stdout, Stderr: stderr})

			output := stdout.String()
			if tt.Want_Diag == "" {
				if code != 0 {
					t.Errorf("expected exit 0, got %d; output: %s", code, output)
				}
			} else {
				if !bytes.Contains(stdout.Bytes(), []byte(tt.Want_Diag)) {
					t.Errorf("expected output containing %q, got: %s", tt.Want_Diag, output)
				}
			}
		})
	}
}

// Test_Line_Character_Count_Tabs verifies that tabs count as tab_width chars (not 1)
// when measuring line length against the max_line_chars limit.
func Test_Line_Character_Count_Tabs(t *testing.T) {
	// 18 tabs * 8 = 144 column width; far under the 140 limit if tabs count
	// as 1, well over if tabs count as 8. Inline comments are exempt from
	// comment-sentence rules, so this stays a pure line-length test.
	tabs := ""
	for range 18 {
		tabs += "\t"
	}
	source := "package main\nfunc f() (result int) { return 1 } //" + tabs + "tail\n"
	fsys_map := fstest.MapFS{"test.go": &fstest.MapFile{Data: []byte(source)}}
	stdout := &bytes.Buffer{}
	code := lint.Main(&lint.Main_Input{Fsys: fsys_map, Stdout: stdout, Stderr: &bytes.Buffer{}})
	if code == 0 {
		t.Fatalf("expected line-length diagnostic, got exit 0; output: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("line is")) {
		t.Errorf("expected line-length diagnostic, got: %s", stdout.String())
	}
}

// Test_Comments verifies the comment-style rules: capital start, trailing
// `.`/`:`/`?`/`!`, space after `//`, pragma exemption, and inline-comment
// exemption from the sentence rules.
func Test_Comments(t *testing.T) {
	tests := []struct {
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
		{
			Name: "inline comment exempt from sentence rules",
			Files: map[string]string{
				"test.go": "package main\n\nfunc f() (result int) { return 1 } // some inline note\n",
			},
			Want_Diag: "",
		},
		{
			Name: "multi-line group: trailing period on last line",
			Files: map[string]string{
				"test.go": "// First line\n// second line.\npackage main\n",
			},
			Want_Diag: "",
		},
	}

	// TestComments fixtures include `//foo` (no space after slashes), which
	// gofmt normalizes before our check sees it. Bypass gofmt_must so the
	// formatting-style violations under test survive intact.
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			fsys_map := make(fstest.MapFS)
			for k, v := range tt.Files {
				fsys_map[k] = &fstest.MapFile{Data: []byte(v)}
			}
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			code := lint.Main(&lint.Main_Input{Fsys: fsys_map, Stdout: stdout, Stderr: stderr})

			output := stdout.String()
			if tt.Want_Diag == "" {
				if code != 0 {
					t.Errorf("expected exit 0, got %d; output: %s", code, output)
				}
			} else {
				if !bytes.Contains(stdout.Bytes(), []byte(tt.Want_Diag)) {
					t.Errorf("expected output containing %q, got: %s", tt.Want_Diag, output)
				}
			}
		})
	}
}

// Test_No_Package_Vars verifies the package-level var ban: only
// regexp.MustCompile and errors.New initializers are allowed (plus the
// compile-time interface-satisfaction shape `var _ Iface = (*Impl)(nil)`).
// Local-scope var declarations inside funcs are untouched.
func Test_No_Package_Vars(t *testing.T) {
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
			Name: "grouped allowed initializers",
			Files: map[string]string{
				"test.go": `package main
import "regexp"
var (
	re1 = regexp.MustCompile("a")
	re2 = regexp.MustCompile("b")
)
func main() { return }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "snap.Default singleton allowed",
			Files: map[string]string{
				"snap.go": `// Package snap is a fixture.
package snap
// Snapper is a fixture.
type Snapper struct{ A int }
// Default is the OS-bound Snapper.
var Default = &Snapper{A: 1}
`,
			},
			Want_Diag: "",
		},
		{
			Name: "Default in non-snap package still banned",
			Files: map[string]string{
				"test.go": `package main
type S struct{ A int }
var Default = &S{A: 1}
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
	tests := []struct {
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
type S struct{ A int }
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
		{
			Name: "grouped mixed: only disallowed flagged",
			Files: map[string]string{
				"test.go": `package main
import "regexp"
var (
	re1   = regexp.MustCompile("a")
	bad_v = 5
)
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
	}
	run_diag_table(t, tests)
}

// Test_Banned_Scripting_Files verifies that scripting-language files
// (.py, .sh, Makefile, ...) anywhere outside the top-level third_party/
// and vendor/ directories are flagged. Bypasses run_diag_table because
// gofmt_must would choke on non-Go content.
func Test_Banned_Scripting_Files(t *testing.T) {
	clean_go := []byte("package main\n\nfunc f() (result int) { return 1 }\n")
	tests := []struct {
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
			Want_Diag: "banned scripting file",
		},
		{
			Name: "nested .sh flagged",
			Files: map[string][]byte{
				"test.go":    clean_go,
				"cmd/run.sh": []byte("#!/bin/sh\necho hi\n"),
			},
			Want_Diag: "banned scripting file",
		},
		{
			Name: "Makefile flagged",
			Files: map[string][]byte{
				"test.go":  clean_go,
				"Makefile": []byte("all:\n\techo hi\n"),
			},
			Want_Diag: "banned scripting file",
		},
		{
			Name: "lowercase makefile flagged",
			Files: map[string][]byte{
				"test.go":  clean_go,
				"makefile": []byte("all:\n\techo hi\n"),
			},
			Want_Diag: "banned scripting file",
		},
		{
			Name: "Rakefile flagged",
			Files: map[string][]byte{
				"test.go":  clean_go,
				"Rakefile": []byte("task :default\n"),
			},
			Want_Diag: "banned scripting file",
		},
		{
			Name: "lua flagged",
			Files: map[string][]byte{
				"test.go":    clean_go,
				"script.lua": []byte("print(1)\n"),
			},
			Want_Diag: "banned scripting file",
		},
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
			Want_Diag: "banned scripting file",
		},
		{
			Name: "go file alone clean",
			Files: map[string][]byte{
				"test.go": clean_go,
			},
			Want_Diag: "",
		},
	}
	test_banned_scripting_files_run(t, tests)
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
			code := lint.Main(&lint.Main_Input{Fsys: fsys_map, Stdout: stdout, Stderr: &bytes.Buffer{}})
			output := stdout.String()
			if tt.Want_Diag == "" {
				if code != 0 {
					t.Errorf("expected exit 0, got %d; output: %s", code, output)
				}
				return
			}
			if !bytes.Contains(stdout.Bytes(), []byte(tt.Want_Diag)) {
				t.Errorf("expected output containing %q, got: %s", tt.Want_Diag, output)
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
		fsys_map[k] = &fstest.MapFile{Data: []byte(v)}
	}
	stdout := &bytes.Buffer{}
	lint.Main(&lint.Main_Input{Fsys: fsys_map, Stdout: stdout, Stderr: &bytes.Buffer{}})
	return stdout.String()
}

// Conflict markers in any file must be flagged with the conflict-markers check
// and the correct line number.
func Test_Stream_Conflict_Markers(t *testing.T) {
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
			Name:   "local uses flagged",
			Path:   ".github/workflows/ci.yml",
			Body:   "jobs:\n  build:\n    steps:\n      - uses: ./.github/actions/foo\n",
			Should: "github-actions-uses",
		},
		{
			Name:   "yaml extension flagged",
			Path:   ".github/workflows/ci.yaml",
			Body:   "steps:\n  - uses: actions/setup-go@v5\n",
			Should: "github-actions-uses",
		},
		{
			Name:   "run-only workflow clean",
			Path:   ".github/workflows/ci.yml",
			Body:   "jobs:\n  build:\n    steps:\n      - name: build\n        run: go build ./...\n",
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
				t.Errorf("path %q: want github-actions-uses=%v, got: %s", tc.Path, want, output)
			}
		})
	}
}

// GPL license text must trigger the copyleft check with the correct family name.
func Test_Stream_Copyleft_GPL(t *testing.T) {
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
	mpl := "Mozilla Public License 2.0\nGNU General Public License (for comparison)\n"
	output := run_stream(t, map[string]string{"LICENSE": mpl})
	if strings.Contains(output, "copyleft") {
		t.Errorf("MPL should not trip copyleft check; got: %s", output)
	}
}

// Files exceeding 1 MiB must be flagged by the max-file-size check.
func Test_Stream_Max_File_Size(t *testing.T) {
	big := strings.Repeat("a", (1<<20)+1)
	output := run_stream(t, map[string]string{"blob.bin": big})
	if !strings.Contains(output, "max-file-size") {
		t.Errorf("expected max-file-size diag on blob.bin, got: %s", output)
	}
	if !strings.Contains(output, "blob.bin") {
		t.Errorf("expected blob.bin in diag, got: %s", output)
	}
}

// Agent docs under dot-dirs must be skipped; agent docs under normal paths
// must be flagged when they exceed 100 lines.
func Test_Stream_Agent_Documentation_Max_Lines(t *testing.T) {
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
			t.Errorf("expected agent-doc-max-lines diag for %s, got: %s", name, loop_output)
		}
	}
}

// Snake_case, Ada_Case, and SCREAMING_SNAKE_CASE paths must pass; kebab-case
// and camelCase paths must be flagged; third_party/ is fully exempt.
func Test_Stream_Path_Casing(t *testing.T) {
	cases := []struct {
		Name   string
		Path   string
		Should string
	}{
		{"snake_case file", "lint_stream.go", ""},
		{"Ada_Case file", "Foo_Bar.txt", ""},
		{"SCREAMING file", "LICENSE", ""},
		{"hidden file exempt", ".gitignore", ""},
		{"snake_case dir", "big_bang/foo.go", ""},
		{"third_party exempt", "third_party/badName-file.go", ""},
		{"kebab-case file flagged", "bad-file.txt", "path-casing"},
		{"camelCase file flagged", "badFile.txt", "path-casing"},
		{"kebab-case dir flagged", "bad-dir/foo.txt", "path-casing"},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			output := run_stream(t, map[string]string{tc.Path: "x\n"})
			has := strings.Contains(output, "path-casing")
			want := tc.Should != ""
			if has != want {
				t.Errorf("path %q: want path-casing=%v, got: %s", tc.Path, want, output)
			}
		})
	}
}

// Markdown lines exceeding 100 runes must be flagged by the markdown-line-length check.
func Test_Stream_Markdown_Line_Max(t *testing.T) {
	long := strings.Repeat("x", 120) + "\n"
	output := run_stream(t, map[string]string{"docs.md": long})
	if !strings.Contains(output, "markdown-line-length") {
		t.Errorf("expected markdown-line-length diag, got: %s", output)
	}
}

// Lines inside fenced code blocks must be exempt from the markdown line cap.
func Test_Stream_Markdown_Code_Fence_Exempt(t *testing.T) {
	long := "```\n" + strings.Repeat("x", 200) + "\n```\n"
	output := run_stream(t, map[string]string{"docs.md": long})
	if strings.Contains(output, "markdown-line-length") {
		t.Errorf("fenced code should be exempt; got: %s", output)
	}
}

// A directory with CLAUDE.md but no AGENTS.md must be flagged for the missing sibling.
func Test_Stream_Agents_Claude_Pair_Absence(t *testing.T) {
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
	bad_go := "package p\n<<<<<<< HEAD\nfunc f() {}\n=======\nfunc g() {}\n>>>>>>> branch\n"
	output := run_stream(t, map[string]string{"a.go": bad_go})
	if !strings.Contains(output, "conflict-markers") {
		t.Errorf("expected conflict-markers diag, got: %s", output)
	}
	if !strings.Contains(output, "parse error") {
		t.Errorf("expected parse error diag, got: %s", output)
	}
}

// Runs lint.Main with the given Git_Input over an empty FS and returns
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
				{Hash: "abcdef0123456789abcdef0123456789abcdef01", Subject: "Merge branch foo"},
			},
			Want_Hits: []string{"merge commit", "abcdef0123", "Merge branch foo"},
			Want_Code: 1,
		},
		{
			Name: "subtree add exempt",
			Merge_Commits: []lint.Git_Commit{
				{Hash: "ccccccccccdddd", Subject: "Add 'foo/' from commit '2d43774e164be386023c13e2b12c2403a57b4a2a'"},
			},
			Want_Code: 0,
		},
		{
			Name: "subtree pull exempt",
			Merge_Commits: []lint.Git_Commit{
				{Hash: "eeeeeeeeeeffff", Subject: "Merge commit '2d43774e164be386023c13e2b12c2403a57b4a2a' as 'foo'"},
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
					t.Errorf("expected output to contain %q; got: %s", want, output)
				}
			}
			if code != tt.Want_Code {
				t.Errorf("expected exit %d, got %d; output: %s", tt.Want_Code, code, output)
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
			flagged := strings.Contains(output, "conventional-commits") || strings.Contains(output, "non-conventional commit")
			if flagged != tt.Want_Flag {
				t.Errorf("subject %q: want flagged=%v, got flagged=%v; output: %s",
					tt.Subject, tt.Want_Flag, flagged, output)
			}
		})
	}
}

// Commit subjects longer than 100 chars must be flagged. Matches the
// max_line_chars cap the file-tier check enforces on source lines — same
// reasoning (visual scan limit in code-review UIs and terminals).
func Test_Git_Commit_Subject_Chars(t *testing.T) {
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
			Want_Diag: "rename to entity_<verb>",
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
			Want_Diag: "rename to entity_<verb>",
		},
		{
			Name: "generic instance flagged",
			Files: map[string]string{
				"a.go": `package foo

type Entity[T any] struct{}

func update(e Entity[int]) { return }
`,
			},
			Want_Diag: "rename to entity_<verb>",
		},
		{
			Name: "exported function requires Ada_Case prefix",
			Files: map[string]string{
				"a.go": `package foo

type Main_Input struct{}

func Run(input Main_Input) { return }
`,
			},
			Want_Diag: "rename to Main_Input_<verb>",
		},
		{
			Name: "pointer to same-pkg type flagged (receiver shape)",
			Files: map[string]string{
				"a.go": `package foo

type Snapper struct{}

func edit(s *Snapper) { return }
`,
			},
			Want_Diag: "rename to snapper_<verb>",
		},
		{
			Name: "Ada_Case fn with miscased multi-word prefix flagged",
			Files: map[string]string{
				"a.go": `package foo

type Main_Input struct{}

func Main_input_Run(input Main_Input) { return }
`,
			},
			Want_Diag: "rename to Main_Input_<verb>",
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

func parse(s string) { return }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "slice of same-pkg type clean",
			Files: map[string]string{
				"a.go": `// Package foo is a fixture.
package foo

// Entity is a fixture.
type Entity struct{}

func update(es []Entity) { return }
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

// Snapper is a fixture.
type Snapper struct{}

func snapper_edit(s *Snapper) { return }
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

// Test_No_Ambient_Stdlib_Hard_Imports verifies that hard-banned imports
// (os, crypto/rand, math/rand v1, flag) fire in library packages, while
// math/rand/v2 is allowed.
func Test_No_Ambient_Stdlib_Hard_Imports(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "library imports os",
			Files: map[string]string{
				"a.go": `package lib

import "os"

func F() (val string) { return os.Getenv("X") }
`,
			},
			Want_Diag: "ambient stdlib import",
		},
		{
			Name: "library imports crypto/rand",
			Files: map[string]string{
				"a.go": `package lib

import "crypto/rand"

func F() (n int, err error) { return rand.Reader.Read(nil) }
`,
			},
			Want_Diag: "ambient stdlib import",
		},
		{
			Name: "library imports math/rand v1",
			Files: map[string]string{
				"a.go": `package lib

import "math/rand"

func F() (n int) { return rand.Int() }
`,
			},
			Want_Diag: "ambient stdlib import",
		},
		{
			Name: "library imports math/rand/v2 clean",
			Files: map[string]string{
				"a.go": `// Package lib is a fixture.
package lib

import "math/rand/v2"

// F is a fixture.
func F(r *rand.Rand) (n int) { return r.Int() }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "library imports flag",
			Files: map[string]string{
				"a.go": `package lib

import "flag"

func F() (s string) { return flag.Arg(0) }
`,
			},
			Want_Diag: "ambient stdlib import",
		},
	}
	run_diag_table(t, tests)
}

// Test_No_Ambient_Stdlib_Soft_Calls verifies that ambient identifiers on
// otherwise-pure packages (time.Now, fmt.Println, http.DefaultClient,
// net.Dial) fire while the pure surface (time.Duration, fmt.Sprintf,
// http.Header, log.Printf) is allowed.
func Test_No_Ambient_Stdlib_Soft_Calls(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "library calls time.Now",
			Files: map[string]string{
				"a.go": `package lib

import "time"

func F() (t time.Time) { return time.Now() }
`,
			},
			Want_Diag: "ambient stdlib call",
		},
		{
			Name: "library uses time.Duration clean",
			Files: map[string]string{
				"a.go": `// Package lib is a fixture.
package lib

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
				"a.go": `package lib

import "fmt"

func F() { fmt.Println("hi") }
`,
			},
			Want_Diag: "ambient stdlib call",
		},
		{
			Name: "library uses fmt.Sprintf and fmt.Errorf clean",
			Files: map[string]string{
				"a.go": `// Package lib is a fixture.
package lib

import "fmt"

// F is a fixture.
func F() (err error) { return fmt.Errorf("x: %s", fmt.Sprintf("y")) }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "library calls log.Printf clean (observability exempt)",
			Files: map[string]string{
				"a.go": `// Package lib is a fixture.
package lib

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
				"a.go": `package lib

import t "time"

func F() (result t.Time) { return t.Now() }
`,
			},
			Want_Diag: "ambient stdlib call",
		},
	}
	run_diag_table(t, tests)
}

// Test_No_Ambient_Stdlib_Soft_Network_Http verifies the http and net soft bans.
func Test_No_Ambient_Stdlib_Soft_Network_Http(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "library references http.DefaultClient",
			Files: map[string]string{
				"a.go": `package lib

import "net/http"

func F() (c *http.Client) { return http.DefaultClient }
`,
			},
			Want_Diag: "ambient stdlib call",
		},
		{
			Name: "library uses http.Request type clean",
			Files: map[string]string{
				"a.go": `// Package lib is a fixture.
package lib

import "net/http"

// F is a fixture.
func F(r *http.Request) (h http.Header) { return r.Header }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "library calls net.Dial",
			Files: map[string]string{
				"a.go": `package lib

import "net"

func F() (c net.Conn, err error) { return net.Dial("tcp", "x:1") }
`,
			},
			Want_Diag: "ambient stdlib call",
		},
	}
	run_diag_table(t, tests)
}

// Test_No_Ambient_Stdlib_Exemptions verifies that package main and _test.go
// files may freely use ambient stdlib calls.
func Test_No_Ambient_Stdlib_Exemptions(t *testing.T) {
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
		{
			Name: "test.go in library may call time.Now",
			Files: map[string]string{
				"a.go": `// Package lib is a fixture.
package lib

// F is a fixture.
func F() (n int) { return 1 }
`,
				"a_test.go": `package lib_test

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

// Test_No_Ambient_Stdlib_Composition_Tier verifies that packages
// sitting exactly one depth below the library tier in their module
// — the composition tier — may bind to ambient stdlib state. This is
// the doctrine's designated home for library defaults, CLIs, servers,
// and anything else that wires the library to a real environment.
func Test_No_Ambient_Stdlib_Composition_Tier(t *testing.T) {
	tests := []struct {
		Name       string
		Files      map[string]string
		Want_Diags []string
		Forbid     []string
	}{
		{
			Name: "library tier ambient import still flagged",
			Files: map[string]string{
				"golang_snacks/go.mod": doctrine_shared_library_go_module,
				"golang_snacks/foo/foo.go": `package foo

import "os"

func Read() (name string) { return os.Getenv("X") }
`,
			},
			Want_Diags: []string{"ambient stdlib import \"os\""},
		},
		{
			Name: "composition tier ambient import allowed",
			Files: map[string]string{
				"golang_snacks/go.mod":     doctrine_shared_library_go_module,
				"golang_snacks/foo/foo.go": fixture_package("foo"),
				"golang_snacks/foo/foo_default/foo_default.go": `// Package foo_default is a fixture.
package foo_default

import "os"

// Read is a fixture.
func Read() (name string) { return os.Getenv("X") }
`,
			},
			Forbid: []string{"ambient stdlib import"},
		},
		{
			Name: "composition tier under versioned library allowed",
			Files: map[string]string{
				"golang_snacks/go.mod":          doctrine_shared_library_go_module,
				"golang_snacks/snap/v2/snap.go": fixture_package("snap"),
				"golang_snacks/snap/v2/snap_default/snap_default.go": `// Package snap_default is a fixture.
package snap_default

import "os"

// Read is a fixture.
func Read() (name string) { return os.Getenv("X") }
`,
			},
			Forbid: []string{"ambient stdlib import"},
		},
		{
			Name: "ambient call (time.Now) at composition tier allowed",
			Files: map[string]string{
				"golang_snacks/go.mod":     doctrine_shared_library_go_module,
				"golang_snacks/foo/foo.go": fixture_package("foo"),
				"golang_snacks/foo/foo_default/foo_default.go": `// Package foo_default is a fixture.
package foo_default

import "time"

// Stamp is a fixture.
func Stamp() (t time.Time) { return time.Now() }
`,
			},
			Forbid: []string{"ambient stdlib call"},
		},
		{
			Name: "binary module composition tier (one level under library tier) allowed",
			Files: map[string]string{
				"mybinary/go.mod":              doctrine_binary_go_module,
				"mybinary/main.go":             "package main\n\nfunc main() { return }\n",
				"mybinary/internal/lib/lib.go": fixture_package("lib"),
				"mybinary/internal/lib/lib_default/lib_default.go": `// Package lib_default is a fixture.
package lib_default

import "os"

// Read is a fixture.
func Read() (name string) { return os.Getenv("X") }
`,
			},
			Forbid: []string{"ambient stdlib import"},
		},
	}
	run_doctrine_diag_table(t, tests)
}

// Test_No_Bare_For_Flagged verifies that for-loops without a header — bare
// `for {}`, `for ;; {}`, or `for true {}` — are flagged.
func Test_No_Bare_For_Flagged(t *testing.T) {
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
			Name: "for range GameLoop escape hatch clean",
			Files: map[string]string{
				"a.go": `package main

import "github.com/james-orcales/james-orcales/golang_snacks/invariant"

func main() {
	for range invariant.GameLoop() {
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

// Test_Default_Package_Alias verifies that *_default packages must be imported
// with their prefix as the alias (snap_default → snap). This is what lets
// callers keep writing snap.Init/snap.Edit, and is required by snap.Edit's
// source-line rewriter which searches for the literal "snap.Edit(".
func Test_Default_Package_Alias(t *testing.T) {
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "snap_default without alias flagged",
			Files: map[string]string{
				"a.go": `package main

import (
	"fmt"

	"example.com/snap/v2/snap_default"
)

func main() { fmt.Println(snap_default.Default) }
`,
			},
			Want_Diag: `must be imported with alias "snap"`,
		},
		{
			Name: "snap_default with wrong alias flagged",
			Files: map[string]string{
				"a.go": `package main

import (
	"fmt"

	wrong "example.com/snap/v2/snap_default"
)

func main() { fmt.Println(wrong.Default) }
`,
			},
			Want_Diag: `must be imported with alias "snap"`,
		},
		{
			Name: "snap_default with snap alias clean",
			Files: map[string]string{
				"a.go": `package main

import (
	"fmt"

	snap "example.com/snap/v2/snap_default"
)

func main() { fmt.Println(snap.Default) }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "non-default import unaffected",
			Files: map[string]string{
				"a.go": `package main

import (
	"fmt"

	"example.com/snap/v2"
)

func main() { fmt.Println(snap.Snapper{}) }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "foo_default expects foo alias",
			Files: map[string]string{
				"a.go": `package main

import (
	"fmt"

	foo "example.com/foo/foo_default"
)

func main() { fmt.Println(foo.Default) }
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
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
				"mybinary/go.mod":  doctrine_binary_go_module,
				"mybinary/main.go": "package main\n\nfunc main() { return }\n",
			},
		},
		{
			Name: "binary with non-main package outside internal flagged",
			Files: map[string]string{
				"mybinary/go.mod":       doctrine_binary_go_module,
				"mybinary/main.go":      "package main\n\nfunc main() { return }\n",
				"mybinary/helpers/h.go": fixture_package("helpers"),
			},
			Want_Diags: []string{"forbids package \"helpers\"", "mybinary/internal/"},
		},
		{
			Name: "binary with package under internal is clean",
			Files: map[string]string{
				"mybinary/go.mod":                  doctrine_binary_go_module,
				"mybinary/main.go":                 "package main\n\nfunc main() { return }\n",
				"mybinary/internal/lib/library.go": fixture_package("lib"),
			},
		},
		{
			Name: "binary with non-main package at module root flagged",
			Files: map[string]string{
				"mybinary/go.mod":     doctrine_binary_go_module,
				"mybinary/library.go": fixture_package("mybinary"),
			},
			Want_Diags: []string{"forbids package \"mybinary\""},
		},
		{
			Name: "shared library with non-main package at depth 1 is exempt",
			Files: map[string]string{
				"golang_snacks/go.mod":     doctrine_shared_library_go_module,
				"golang_snacks/foo/foo.go": fixture_package("foo"),
			},
		},
		{
			Name: "package main at any depth in binary is exempt",
			Files: map[string]string{
				"mybinary/go.mod":             doctrine_binary_go_module,
				"mybinary/main.go":            "package main\n\nfunc main() { return }\n",
				"mybinary/cmd/extra/extra.go": "package main\n\nfunc main() { return }\n",
			},
		},
	}
	run_doctrine_diag_table(t, tests)
}

// Test_Shared_Library_No_Internal verifies that the shared library
// module may not contain internal/ directories. Other modules are
// unaffected by this rule — they're expected to use internal/.
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
				"golang_snacks/go.mod":                      doctrine_shared_library_go_module,
				"golang_snacks/foo/foo.go":                  fixture_package("foo"),
				"golang_snacks/foo/internal/helper/help.go": fixture_package("helper"),
			},
			Want_Diags: []string{"shared library forbids internal/", "golang_snacks/foo/internal"},
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
				"mybinary/go.mod":                  doctrine_binary_go_module,
				"mybinary/main.go":                 "package main\n\nfunc main() { return }\n",
				"mybinary/internal/lib/library.go": fixture_package("lib"),
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
				"golang_snacks/go.mod":             doctrine_shared_library_go_module,
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
				"golang_snacks/go.mod":             doctrine_shared_library_go_module,
				"golang_snacks/snap/snap.go":       fixture_package("snap"),
				"golang_snacks/snap/v2/snap.go":    fixture_package("snap"),
				"golang_snacks/snap/v2/sub/sub.go": fixture_package("sub"),
			},
			Forbid: []string{"exceeds library tier"},
		},
		{
			Name: "non-Go intermediate directory does not count as ancestor",
			Files: map[string]string{
				"golang_snacks/go.mod":                         doctrine_shared_library_go_module,
				"golang_snacks/foo/foo.go":                     fixture_package("foo"),
				"golang_snacks/foo/examples/sample/example.go": fixture_package("example"),
			},
			Forbid: []string{"exceeds library tier"},
		},
		{
			Name: "main package at depth does not count as ancestor",
			Files: map[string]string{
				"mybinary/go.mod":                  doctrine_binary_go_module,
				"mybinary/main.go":                 "package main\n\nfunc main() { return }\n",
				"mybinary/internal/foo/foo.go":     fixture_package("foo"),
				"mybinary/internal/foo/bar/bar.go": fixture_package("bar"),
			},
			Forbid: []string{"exceeds library tier"},
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
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "exported func without doc flagged",
			Files: map[string]string{
				"foo.go": "// Package foo provides things.\npackage foo\n\nfunc Do() { return }\n",
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
				"foo.go": "// Package foo provides things.\npackage foo\n\nfunc do() { return }\n",
			},
			Want_Diag: "",
		},
		{
			Name: "exported type without doc flagged",
			Files: map[string]string{
				"foo.go": "// Package foo provides things.\npackage foo\n\ntype Thing struct{ X int }\n",
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
				"foo.go": "// Package foo provides things.\npackage foo\n\nconst Max_Count = 100\n",
			},
			Want_Diag: "exported const Max_Count is missing a doc comment",
		},
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
					"func (t *Thing) String() (output string) { return \"\" }\n",
			},
			Want_Diag: "exported method String is missing a doc comment",
		},
		{
			Name: "package main exempt: exported func without doc allowed",
			Files: map[string]string{
				"main.go": "package main\n\nfunc main() { return }\n\nfunc Run() { return }\n",
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
				"foo.go": "package foo\n\n// Do performs the operation.\nfunc Do() { return }\n",
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
				"doc.go": "//go:build doc_only\n\n" +
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
				"main.go": "package main\n\nfunc main() { return }\n",
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
