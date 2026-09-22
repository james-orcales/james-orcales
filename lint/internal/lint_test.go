package lint_test

import (
	"fmt"
	"go/format"
	"local/james-orcales/lint/internal/strings"
	"testing"
	"testing/fstest"

	"local/james-orcales/lint/internal"
)

const DOCTRINE_SHARED_LIBRARY_MODULE_PATH = "github.com/james-orcales/" +
	"james-orcales/shared"
const DOCTRINE_SHARED_LIBRARY_GO_MODULE = "module " +
	DOCTRINE_SHARED_LIBRARY_MODULE_PATH + "\n"
const DOCTRINE_BINARY_GO_MODULE = "module example.com/mybinary\n"

// The repo is one module now, so every fixture needs a single root go.mod or the
// single-module check fires. lint_main injects this when a fixture omits one. Its
// path is the repo's, so a top-level component dir resolves to this path joined
// with the dir — e.g. a "shared" dir is github.com/james-orcales/james-orcales/shared.
const DOCTRINE_ROOT_GO_MODULE = "module github.com/james-orcales/james-orcales\n"

// The shared module is identified by its workspace-root-relative directory, not
// its import path. Doctrine-table fixtures put the shared library's go.mod at
// shared/go.mod, so its Root — and the value Shared_Component must carry —
// is "shared".
const DOCTRINE_SHARED_COMPONENT_DIRECTORY = "shared"

// Snapshot fixtures instead put the shared library's go.mod at the MapFS root,
// so there its Root is ".". Binary modules in those fixtures live in named
// subdirectories, so "." selects only the shared library.
const DOCTRINE_SHARED_COMPONENT_AT_ROOT = "."

// Satisfies the rule that every binary module declares exactly one free func
// Main in its top-level pkg/ package. Injected into binary-module fixtures
// that exist to exercise other rules so the entry-point check doesn't bleed an
// extra diagnostic into them.
const DOCTRINE_BINARY_PACKAGE_MAIN = "// Package entry is a fixture.\n" +
	"package entry\n\n// Main is a fixture entry point.\nfunc Main() { return }\n"

const FIXTURE_INVARIANT_IMPORT_PATH = "github.com/james-orcales/james-orcales/" +
	"shared/sim/aver/v2/aver_default"
const FIXTURE_INVARIANT_IMPORT = "import \"" + FIXTURE_INVARIANT_IMPORT_PATH + "\"\n"

// Fixture_const_hi declares a file-scope upper bound used by Distinct_Boundary
// Hi positions in test fixtures so the new assertion-bound-named-constant
// rule is satisfied. Append to the import block, before any non-import decl.
const FIXTURE_CONSTANT_HI = "\nconst FIXTURE_HI = 100\n"

// FIXTURE_DECLARATION_CALLEE defines a parameterless g() returning a non-nil
// pointer, used by declaration-coverage fixtures so the test cases focus on
// the call-site shape rather than re-deriving a valid callee each time.
const FIXTURE_DECLARATION_CALLEE = "func g() (out *int) {\n" +
	"\treturn nil\n" +
	"}\n\n"

// FIXTURE_DECLARATION_CALLEE_PAIR is the two-return analogue of g(), used by
// fixtures exercising multi-LHS declarations.
const FIXTURE_DECLARATION_CALLEE_PAIR = "func g() (a *int, b *int) {\n" +
	"\tdefer func() {\n" +
	"a != nil, \"a is non-nil\"),\n" +
	"b != nil, \"b is non-nil\"),\n" +
	"\t\t)\n" +
	"\t}()\n" +
	"\treturn nil, nil\n" +
	"}\n\n"

// FIXTURE_CLEAN_GO is the canonical valid-Go fixture used by tests that
// need an accompanying .go file but don't care about its specific shape.
const FIXTURE_CLEAN_GO = "package main\n\n" +
	"import \"github.com/james-orcales/james-orcales/" +
	"shared/sim/aver/v2\"\n\n" +
	"const FIXTURE_HI = 100\n\n" +
	"func f() (result int) {\n" +
	"\tdefer func() {\n" +
	"\t\tinvariant.Cross_Product(\n" +
	"\t\t\tinvariant.Distinct_Boundary(" +
	"&aver.Boundary_Input[int]{\n" +
	"\t\t\t\tX: result, Lo: 0, Hi: FIXTURE_HI}),\n" +
	"\t\t\tinvariant.Always(" +
	"result == 0, \"result is zero\"),\n" +
	"\t\t)\n" +
	"\t}()\n" +
	"\treturn 1\n" +
	"}\n"

// TestMain wires Recorder_Run_Test_Main: registers source files for coverage,
// runs the test suite, then reports any never-fired assertion sites.
const PRELUDE_SINGLE = "package fixture\n\n" +
	FIXTURE_INVARIANT_IMPORT + FIXTURE_DECLARATION_CALLEE
const PRELUDE_PAIR = "package fixture\n\n" +
	FIXTURE_INVARIANT_IMPORT + FIXTURE_DECLARATION_CALLEE_PAIR
const PRELUDE_WITH_H = PRELUDE_SINGLE +
	"func h(p *int) (out int) {\n" +
	"\tdefer func() {\n" +
	"\t\tinvariant.Cross_Product(\n" +
	"\t\t\tinvariant.Distinct_Boundary(" +
	"&aver.Boundary_Input[int]{" +
	"X: out, Lo: 0, Hi: 1}),\n" +
	"\t\t)\n" +
	"\t}()\n" +
	"\tinvariant.Cross_Product(" +
	"aver.Always(p != nil, \"p is non-nil\"))\n" +
	"\treturn 0\n" +
	"}\n\n"

// Builds a package fixture with one function.
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
	t.Parallel()
	for _, tt := range []struct {
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
			Want_Diag: "shadows a variable in an outer scope",
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
			Want_Diag: "shadows a variable in an outer scope",
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			fsys_map := make(fstest.MapFS)
			for k, v := range tt.Files {
				fsys_map[k] = &fstest.MapFile{Data: gofmt_must(t, v)}
			}
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}
			code := lint_main(t, &lint.Main_Input{
				Fsys: fsys_map, Stdout: stdout, Stderr: stderr,
			})

			output := stdout.String()
			if tt.Want_Diag == "" {
				if code != 0 {
					t.Errorf("expected exit 0, got %d; output: %s",
						code, output)
				}
			} else {
				if !strings.Contains(stdout.String(), tt.Want_Diag) {
					t.Errorf("expected output containing %q, got: %s",
						tt.Want_Diag, output)
				}
			}
		})
	}
}

// Additional cases, split to keep each function within the length limit.
func Test_Variable_Shadow_Part2(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

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
			Want_Diag: "shadows a variable in an outer scope",
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
	} {
		t.Run(tt.Name, func(t *testing.T) {
			fsys_map := make(fstest.MapFS)
			for k, v := range tt.Files {
				fsys_map[k] = &fstest.MapFile{Data: gofmt_must(t, v)}
			}
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}
			code := lint_main(t, &lint.Main_Input{
				Fsys: fsys_map, Stdout: stdout, Stderr: stderr,
			})

			output := stdout.String()
			if tt.Want_Diag == "" {
				if code != 0 {
					t.Errorf("expected exit 0, got %d; output: %s",
						code, output)
				}
			} else {
				if !strings.Contains(stdout.String(), tt.Want_Diag) {
					t.Errorf("expected output containing %q, got: %s",
						tt.Want_Diag, output)
				}
			}
		})
	}
}

// Additional cases, split to keep each function within the length limit.
func Test_Variable_Shadow_Part3(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "nested blocks exercise scope.Parent.Parent != nil branch",
			Files: map[string]string{
				"test.go": `package main
func f() {
	x := 1
	{
		{
			x := 2
		}
	}
}`,
			},
			Want_Diag: "shadows a variable in an outer scope",
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			fsys_map := make(fstest.MapFS)
			for k, v := range tt.Files {
				fsys_map[k] = &fstest.MapFile{Data: gofmt_must(t, v)}
			}
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}
			code := lint_main(t, &lint.Main_Input{
				Fsys: fsys_map, Stdout: stdout, Stderr: stderr,
			})

			output := stdout.String()
			if tt.Want_Diag == "" {
				if code != 0 {
					t.Errorf("expected exit 0, got %d; output: %s",
						code, output)
				}
			} else {
				if !strings.Contains(stdout.String(), tt.Want_Diag) {
					t.Errorf("expected output containing %q, got: %s",
						tt.Want_Diag, output)
				}
			}
		})
	}
}

// Test_Variable_Shadow_Walker_Nested_Scope_Branch exercises the walker
// family with a range/decl/assign nested inside an inner block — the
// shape that flips Sometimes(scope_value.Parent.Parent == nil) to its
// false branch for push_range_statement, declaration_statement,
// assign_statement, and the variables they recurse through. The bare
// for-init at the function-scope top also leaves scope_value.Parent.Names
// empty when check_assign_define fires, hitting the Lo bucket on the
// parent-names axis.
func Test_Variable_Shadow_Walker_Nested_Scope_Branch(t *testing.T) {
	t.Parallel()
	source := `package main
func f() {
	for index := 0; index < 10; index++ {
		c := index
		c += 1
	}
	xs := []int{1, 2}
	{
		for _, y := range xs {
			b := y
			b += 1
		}
		var z int
		z += 1
		w := 1
		w += 1
		if true {
			a := 2
			a += 1
		}
	}
}`
	fsys_map := fstest.MapFS{"test.go": &fstest.MapFile{Data: gofmt_must(t, source)}}
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	code := lint_main(t, &lint.Main_Input{Fsys: fsys_map, Stdout: stdout, Stderr: stderr})
	if code != 0 {
		t.Errorf("expected exit 0, got %d; output: %s", code, stdout.String())
	}
}

// Test_No_Discard verifies the ban on `_ = ...` and all-blank tuple
// assignments, with the mixed-blank and interface-satisfaction exceptions.
func Test_No_Discard(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{Name: "single discard assign",
			Files: map[string]string{"test.go": `package main
func f() {
	_ = g()
}`}, Want_Diag: "blank identifier hides the value"},

		{Name: "two discards short decl",
			Files: map[string]string{"test.go": `package main
func f() {
	_, _ := g()
}`}, Want_Diag: "blank identifier hides the value"},

		{Name: "three discards assign",
			Files: map[string]string{"test.go": `package main
func f() {
	_, _, _ = g()
}`}, Want_Diag: "blank identifier hides the value"},

		{Name: "var blank no type",
			Files: map[string]string{"test.go": `package main
var _ = g()`}, Want_Diag: "blank identifier hides the value"},

		{Name: "mixed lhs short decl allowed",
			Files: map[string]string{"test.go": `package main
func f() {
	_, x := g()
	_ = x
}`}, Want_Diag: "blank identifier hides the value"},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			fsys_map := make(fstest.MapFS)
			for k, v := range tt.Files {
				fsys_map[k] = &fstest.MapFile{Data: gofmt_must(t, v)}
			}
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}
			code := lint_main(t, &lint.Main_Input{
				Fsys: fsys_map, Stdout: stdout, Stderr: stderr,
			})
			output := stdout.String()
			if tt.Want_Diag == "" {
				if code != 0 {
					t.Errorf("expected exit 0, got %d; output: %s",
						code, output)
				}
				return
			}
			if !strings.Contains(stdout.String(), tt.Want_Diag) {
				t.Errorf("expected output containing %q, got: %s",
					tt.Want_Diag, output)
			}
		})
	}
}

// Additional cases, split to keep each function within the length limit.
func Test_No_Discard_Part2(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{Name: "mixed lhs short decl only",
			Files: map[string]string{"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func f() (result int) {
	defer func() {
	}()
	_, x := g()
	return x
}`}, Want_Diag: ""},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			fsys_map := make(fstest.MapFS)
			for k, v := range tt.Files {
				fsys_map[k] = &fstest.MapFile{Data: gofmt_must(t, v)}
			}
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}
			code := lint_main(t, &lint.Main_Input{
				Fsys: fsys_map, Stdout: stdout, Stderr: stderr,
			})
			output := stdout.String()
			if tt.Want_Diag == "" {
				if code != 0 {
					t.Errorf("expected exit 0, got %d; output: %s",
						code, output)
				}
				return
			}
			if !strings.Contains(stdout.String(), tt.Want_Diag) {
				t.Errorf("expected output containing %q, got: %s",
					tt.Want_Diag, output)
			}
		})
	}
}

// Test_Public_Struct_Fields verifies that unexported struct fields (named and
// embedded) are flagged with a rename suggestion.
func Test_Public_Struct_Fields(t *testing.T) {
	t.Parallel()
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
			Want_Diag: "Rename bar",
		},
		{
			Name: "uppercase named field allowed",
			Files: map[string]string{
				"test.go": `package main

type Foo struct {
	// Bar is a fixture.
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
			Want_Diag: "Rename bar",
		},
		{
			Name: "embedded uppercase type allowed",
			Files: map[string]string{
				"test.go": `package main

type Foo struct {
	Bar
}

type Bar struct{}
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Public_Struct_Fields_Part2 covers the embedded-pointer case, split off
// to keep each function within the length limit.
func Test_Public_Struct_Fields_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "embedded pointer to uppercase type allowed",
			Files: map[string]string{
				"test.go": `package main

type Foo struct {
	*Bar
}

type Bar struct{}
`,
			},
			Want_Diag: "",
		},
	})
}

// Test_Exported_Type_Exposes_Private verifies that exported struct field
// types and exported aliases that resolve to an unexported named identifier
// are flagged (with pointer unwrapping and same-file recurse).
func Test_Exported_Type_Exposes_Private(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{Name: "direct lowercase field flagged",
			Files: map[string]string{"test.go": `package main
type Foo struct { F bar }
type bar int
`}, Want_Diag: "The public type Foo contains the private type bar"},
		{Name: "pointer to lowercase flagged",
			Files: map[string]string{"test.go": `package main
type Foo struct { F *bar }
type bar int
`}, Want_Diag: "The public type Foo contains the private type bar"},
		{Name: "double pointer flagged",
			Files: map[string]string{"test.go": `package main
type Foo struct { F **bar }
type bar int
`}, Want_Diag: "The public type Foo contains the private type bar"},
		{Name: "embedded lowercase flagged",
			Files: map[string]string{"test.go": `package main
type Foo struct { bar }
type bar struct{}
`}, Want_Diag: "The public type Foo contains the private type bar"},
		{Name: "transitive via same-file exported flagged",
			Files: map[string]string{"test.go": `package main
type Foo struct { M Middle }
type Middle struct { F bar }
type bar int
`}, Want_Diag: "The public type Foo contains the private type bar"},
		{Name: "alias to unexported flagged",
			Files: map[string]string{"test.go": `package main
type Foo = bar
type bar int
`}, Want_Diag: "The public type Foo contains the private type bar"},
		{Name: "pointer alias to unexported flagged",
			Files: map[string]string{"test.go": `package main
type Foo = *bar
type bar int
`}, Want_Diag: "The public type Foo contains the private type bar"},
	}
	run_diag_table(t, tests)
}

// Test_Exported_Type_Exposes_Private_Allows covers field-position negatives:
// builtins, qualified selectors, and a self-referential cycle.
func Test_Exported_Type_Exposes_Private_Allows(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{Name: "builtins allowed",
			Files: map[string]string{"test.go": `package main
type Foo struct {
	// I is a fixture.
	I int
	// S is a fixture.
	S string
	// E is a fixture.
	E error
	// B is a fixture.
	B bool
	// R is a fixture.
	R rune
	// Y is a fixture.
	Y byte
}
`}, Want_Diag: ""},
		{Name: "qualified type allowed",
			Files: map[string]string{"test.go": `package main
import "time"
type Foo struct {
	// T is a fixture.
	T time.Time
}
`}, Want_Diag: ""},
		{Name: "self-cycle allowed",
			Files: map[string]string{"test.go": `package main
type Node struct {
	// Next is a fixture.
	Next *Node
	// V is a fixture.
	V    int
}
`}, Want_Diag: ""},
	}
	run_diag_table(t, tests)
}

// Test_Exported_Type_Exposes_Private_Allows_Part2 covers the remaining
// negative cases: mutual recurse, slice element positions (out of scope),
// unexported parents, aliases to exported types, and _test.go exemption.
func Test_Exported_Type_Exposes_Private_Allows_Part2(t *testing.T) {
	t.Parallel()
	// Every case asserts the exposes-private rule stays silent (Forbid). The
	// cases whose fixture declares an unexported package-level type in non-test
	// source now also earn the exported-type diagnostic, so they expect a nonzero
	// exit with "is not exported"; the exported and _test.go cases stay clean.
	run_doctrine_diag_table(t, []struct {
		Name       string
		Files      map[string]string
		Want_Diags []string
		Forbid     []string
	}{
		{Name: "mutual recurse allowed",
			Files: map[string]string{"test.go": `package main
type A struct {
	// B is a fixture.
	B *B
}
type B struct {
	// A is a fixture.
	A *A
}
`}, Want_Diags: nil, Forbid: []string{"contains private"}},
		{Name: "slice of unexported allowed",
			Files: map[string]string{"test.go": `package main
type Foo struct {
	// Xs is a fixture.
	Xs []bar
}
type bar int
`}, Want_Diags: []string{"is not exported"}, Forbid: []string{"contains private"}},
		{Name: "unexported parent allowed",
			Files: map[string]string{"test.go": `package main
type foo struct { X bar }
type bar int
`}, Want_Diags: []string{"is not exported"}, Forbid: []string{"contains private"}},
		{Name: "alias to exported allowed",
			Files: map[string]string{"test.go": `package main
type Foo = Bar
type Bar int
`}, Want_Diags: nil, Forbid: []string{"contains private"}},
		{Name: "pointer alias to exported allowed",
			Files: map[string]string{"test.go": `package main
type Foo = *Bar
type Bar int
`}, Want_Diags: nil, Forbid: []string{"contains private"}},
		{Name: "_test.go file skipped",
			Files: map[string]string{"foo_test.go": `package foo_test
type Foo struct { F bar }
type bar int
`}, Want_Diags: nil, Forbid: []string{"contains private"}},
	})
}

// Test_No_Naked_Return verifies that a bare `return` statement inside a
// value-returning function (or closure) is flagged. Void functions and
// explicit `return X` statements pass clean.
func Test_No_Naked_Return(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{Name: "func with named return and bare return flagged",
			Files: map[string]string{"test.go": `package main
func f() (x int) { return }
`}, Want_Diag: "Do not use a naked return"},

		{Name: "method with named return and bare return flagged",
			Files: map[string]string{"test.go": `package main
type S struct{}
func (s *S) f() (x int) { return }
`}, Want_Diag: "Do not use a naked return"},

		{Name: "closure with named return and bare return flagged",
			Files: map[string]string{"test.go": `package main
func g() (out int) {
	cb := func() (x int) { return }
	out = cb()
	return out
}
`}, Want_Diag: "Do not use a naked return"},

		{Name: "blank-named return with bare return flagged",
			Files: map[string]string{"test.go": `package main
func f() (_ int) { return }
`}, Want_Diag: "Do not use a naked return"},

		{Name: "multiple bare returns each flagged",
			Files: map[string]string{"test.go": `package main
func f(c bool) (x int) {
	if c {
		return
	}
	return
}
`}, Want_Diag: "Do not use a naked return"},

		{Name: "void early-exit allowed",
			Files: map[string]string{"test.go": `package main
func f() {
	if true {
		return
	}
}
`}, Want_Diag: ""},

		{Name: "void trailing return allowed",
			Files: map[string]string{"test.go": `package main
func f() { return }
`}, Want_Diag: ""},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_No_Naked_Return_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{Name: "explicit return allowed",
			Files: map[string]string{"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func f() (x int) {
	defer func() {
	}()
	return 0
}
`}, Want_Diag: ""},

		{Name: "void closure inside value-returning func allowed",
			Files: map[string]string{"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func f() (output int) {
	defer func() {
	}()
	callback := func() {}
	callback()
	return 0
}
`}, Want_Diag: ""},
	})
}

// Test_No_Iota verifies that any use of the iota identifier is flagged.
func Test_No_Iota(t *testing.T) {
	t.Parallel()
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
			Want_Diag: "Do not use iota",
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

// Test_Exported_Type verifies that an unexported package-level type declaration
// or alias is flagged, while exported types, function-local types, _test.go
// files, and the temporarily-exempt ./lint tree are left clean.
func Test_Exported_Type(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name     string
		Filename string
		Source   string
		Want     bool
	}{
		{
			Name:     "unexported struct type flagged",
			Filename: "widget.go",
			Source:   "package fixture\n\ntype widget struct{ X int }\n",
			Want:     true,
		},
		{
			Name:     "unexported alias flagged",
			Filename: "widget.go",
			Source:   "package fixture\n\ntype widget = int\n",
			Want:     true,
		},
		{
			Name:     "exported type clean",
			Filename: "widget.go",
			Source:   "package main\n\ntype Widget int\n",
			Want:     false,
		},
		{
			Name:     "function local type ignored",
			Filename: "widget.go",
			Source: "package main\n\nfunc F() {\n" +
				"\ttype key int\n\tvar value key\n\tvalue += 1\n}\n",
			Want: false,
		},
		{
			Name:     "test file exempt",
			Filename: "widget_test.go",
			Source:   "package fixture\n\ntype widget struct{ X int }\n",
			Want:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			diags, err := lint.Check_Source(tt.Filename, tt.Source)
			if err != nil {
				t.Fatalf("Check_Source: %v", err)
			}
			found := false
			for _, d := range diags {
				if strings.Contains(d.Message, "is not exported") {
					found = true
				}
			}
			if found != tt.Want {
				t.Fatalf("exported-type fired=%v, want=%v; diags=%v",
					found, tt.Want, diags)
			}
		})
	}
}

// Test_No_Fallthrough verifies a switch-case fallthrough is flagged.
func Test_No_Fallthrough(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "fallthrough flagged",
			Files: map[string]string{
				"test.go": `package main

func f() (n int) {
	switch n {
	case 1:
		fallthrough
	case 2:
		n = 3
	}
	return n
}
`,
			},
			Want_Diag: "Do not use fallthrough",
		},
		{
			Name: "switch without fallthrough allowed",
			Files: map[string]string{
				"test.go": `package main

func f() (n int) {
	switch n {
	case 1:
		n = 2
	case 2:
		n = 3
	}
	return n
}
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_No_Blank_Import verifies a blank import is flagged.
func Test_No_Blank_Import(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "blank import flagged",
			Files: map[string]string{
				"test.go": `package main

import _ "strings"
`,
			},
			Want_Diag: "Do not use a blank import",
		},
		{
			Name: "named import allowed",
			Files: map[string]string{
				"test.go": `package main

import "fmt"

func f() (s string) { return fmt.Sprint("x") }
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_No_Grouped_Declaration_Flagged verifies that any parenthesized
// var/const/type declaration is flagged regardless of spec count.
func Test_No_Grouped_Declaration_Flagged(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "grouped const multi-spec flagged",
			Files: map[string]string{
				"test.go": `package main

const (
	foo = "foo"
	bar = "bar"
)
`,
			},
			Want_Diag: "Do not use a grouped declaration",
		},
		{
			Name: "grouped const single-spec flagged",
			Files: map[string]string{
				"test.go": `package main

const (
	foo = "foo"
)
`,
			},
			Want_Diag: "Do not use a grouped declaration",
		},
		{
			Name: "grouped var multi-spec flagged",
			Files: map[string]string{
				"test.go": `package main

var (
	foo = "foo"
	bar = "bar"
)
`,
			},
			Want_Diag: "Do not use a grouped declaration",
		},
		{
			Name: "grouped type multi-spec flagged",
			Files: map[string]string{
				"test.go": `package main

type (
	foo int
	bar string
)
`,
			},
			Want_Diag: "Do not use a grouped declaration",
		},
	}
	run_diag_table(t, tests)
}

// Test_No_Grouped_Declaration_Allowed verifies the negative side: ungrouped
// single and multi-name declarations stay clean, and grouped imports remain
// exempt (gofmt owns that form).
func Test_No_Grouped_Declaration_Allowed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "individual const allowed",
			Files: map[string]string{
				"test.go": `package main

const FOO = "foo"
const BAR = "bar"
`,
			},
			Want_Diag: "",
		},
		{
			Name: "multi-name const at package allowed",
			Files: map[string]string{
				"test.go": `package main

const A, B = 1, 2
`,
			},
			Want_Diag: "",
		},
		{
			Name: "multi-name var in function allowed",
			Files: map[string]string{
				"test.go": `package main

func main() {
	var a, b int
	print(a, b)
}
`,
			},
			Want_Diag: "",
		},
		{
			Name: "grouped import allowed",
			Files: map[string]string{
				"test.go": `package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println(time.Now())
}
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_No_Third_Party_Struct_Tag_Flagged verifies that any struct field
// carrying a tag whose key is not in the stdlib set {json, xml, asn1} is
// flagged. Mixed tags emit one diagnostic per disallowed key.
func Test_No_Third_Party_Struct_Tag_Flagged(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "yaml tag flagged",
			Files: map[string]string{
				"test.go": `package main

func main() {
	var foo Foo
	print(foo.Name)
}

type Foo struct {
	// Name is a fixture.
	Name string ` + "`yaml:\"name\"`" + `
}
`,
			},
			Want_Diag: "yaml",
		},

		{
			Name: "validate tag flagged",
			Files: map[string]string{
				"test.go": `package main

func main() {
	var foo Foo
	print(foo.Value)
}

type Foo struct {
	// Value is a fixture.
	Value int ` + "`validate:\"required\"`" + `
}
`,
			},
			Want_Diag: "validate",
		},

		{
			Name: "mixed json+yaml flags only yaml",
			Files: map[string]string{
				"test.go": `package main

func main() {
	var foo Foo
	print(foo.Name)
}

type Foo struct {
	// Name is a fixture.
	Name string ` + "`json:\"name\" yaml:\"name\"`" + `
}
`,
			},
			Want_Diag: "yaml",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_No_Third_Party_Struct_Tag_Flagged_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "nested anonymous struct field tag flagged",
			Files: map[string]string{
				"test.go": `package main

func main() {
	var outer Outer
	print(outer.Inner.Name)
}

type Outer struct {
	// Inner is a fixture.
	Inner struct {
		Name string ` + "`yaml:\"name\"`" + `
	}
}
`,
			},
			Want_Diag: "yaml",
		},
	})
}

// Test_No_Third_Party_Struct_Tag_Allowed verifies the negative side: pure
// stdlib tags (json, xml, asn1) and combinations of them lint clean, and
// untagged fields are unaffected.
func Test_No_Third_Party_Struct_Tag_Allowed(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "json tag allowed",
			Files: map[string]string{
				"test.go": `package main

func main() {
	var foo Foo
	print(foo.Name)
}

type Foo struct {
	// Name is a fixture.
	Name string ` + "`json:\"name\"`" + `
}
`,
			},
			Want_Diag: "",
		},

		{
			Name: "xml tag allowed",
			Files: map[string]string{
				"test.go": `package main

func main() {
	var foo Foo
	print(foo.Name)
}

type Foo struct {
	// Name is a fixture.
	Name string ` + "`xml:\"name,attr\"`" + `
}
`,
			},
			Want_Diag: "",
		},

		{
			Name: "asn1 tag allowed",
			Files: map[string]string{
				"test.go": `package main

func main() {
	var foo Foo
	print(foo.Name)
}

type Foo struct {
	// Name is a fixture.
	Name string ` + "`asn1:\"utf8\"`" + `
}
`,
			},
			Want_Diag: "",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_No_Third_Party_Struct_Tag_Allowed_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "json+xml combo allowed",
			Files: map[string]string{
				"test.go": `package main

func main() {
	var foo Foo
	print(foo.Name)
}

type Foo struct {
	// Name is a fixture.
	Name string ` + "`json:\"name\" xml:\"name\"`" + `
}
`,
			},
			Want_Diag: "",
		},

		{
			Name: "untagged struct field allowed",
			Files: map[string]string{
				"test.go": `package main

func main() {
	var foo Foo
	print(foo.Name)
}

type Foo struct {
	// Name is a fixture.
	Name string
}
`,
			},
			Want_Diag: "",
		},
	})
}

// Test_No_Third_Party_Struct_Tag_Malformed_Ignored verifies that a tag
// literal the parser can't decompose into key/value pairs is silently
// ignored (no diagnostic) — there's nothing to bind a key name to.
func Test_No_Third_Party_Struct_Tag_Malformed_Ignored(t *testing.T) {
	t.Parallel()
	source := `package main

func main() {
	var foo Foo
	print(foo.Name)
}

type Foo struct {
	// Name is a fixture.
	Name string ` + "`malformed`" + `
}
`
	fsys_map := fstest.MapFS{"test.go": &fstest.MapFile{Data: gofmt_must(t, source)}}
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	code := lint_main(t, &lint.Main_Input{Fsys: fsys_map, Stdout: stdout, Stderr: stderr})
	if code != 0 {
		t.Errorf("expected exit 0, got %d; output: %s", code, stdout.String())
	}
}

// Test_Keyed_Struct_Init_Flagged verifies that same-file positional struct
// literals are flagged.
func Test_Keyed_Struct_Init_Flagged(t *testing.T) {
	t.Parallel()
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
			Want_Diag: "keyed fields",
		},
	}
	run_diag_table(t, tests)
}

// Test_Keyed_Struct_Init_Allowed verifies that same-file keyed struct
// literals, slice literals, and empty struct literals are not flagged.
func Test_Keyed_Struct_Init_Allowed(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "same-file keyed allowed",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

type Foo struct {
	// A is a fixture.
	A int
	// B is a fixture.
	B int
}

func make_v() (result Foo) {
	defer func() {
	}()
	return Foo{A: 1, B: 2}
}
`,
			},
			Want_Diag: "",
		},

		{
			Name: "slice literal allowed",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func make_v() (result []int) {
	defer func() {
	}()
	return []int{1, 2, 3}
}
`,
			},
			Want_Diag: "",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_Keyed_Struct_Init_Allowed_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "empty struct literal allowed",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

type Foo struct{}

func make_v() (result Foo) {
	defer func() {
	}()
	return Foo{}
}
`,
			},
			Want_Diag: "",
		},
	})
}

// Test_No_Dot_Import verifies that dot imports (`import . "pkg"`) are flagged
// while regular and named imports are allowed.
func Test_No_Dot_Import(t *testing.T) {
	t.Parallel()
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
			Want_Diag: "Do not use a dot import",
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
	t.Parallel()
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
			Want_Diag: "Do not use func init",
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

// Renders a lint.json for a fixture: lint.Main now reads the
// config from its Fsys, so every fixture needs one. An empty shared_component is
// replaced with a sentinel matching no fixture module, preserving the historical
// "no module is shared" behavior of the field-free runs the tests used to make.
func test_lint_json(t *testing.T, shared_component string, allowlist []string) (data []byte) {
	t.Helper()
	if shared_component == "" {
		shared_component = "lint_test_no_shared_component"
	}
	data = []byte(configuration_document(map[string]any{
		"shared_component":         shared_component,
		"instrumentation_packages": allowlist,
		// The behavioral suite isolates one rule per fixture; the type-invariant
		// rule fires on every typed fixture, so disable it wholesale here. Its own
		// coverage lives in the Test_Invariants_* doctrine tests and the
		// Test_Type_Invariant_* behavioral tests, which drive the rule directly.
		"opt_out_assertion_mandate_packages": []string{"**"},
		// Likewise the deterministic tier now binds every pure package by default,
		// so a fixture using time/sync/a channel to exercise another rule would trip
		// it; "**" releases the whole tree. The Test_Deterministic_* tests drive the
		// tier directly.
		"pure_but_indeterministic_packages": []string{"**"},
	}))
	return data
}

// Test_lint_json_recurse_exempt_input pairs the shared component with the one
// opt_out_recursion_ban entry a recurse fixture needs.
type test_lint_json_recurse_exempt_input struct {
	// Shared_Component is the workspace's shared library directory.
	Shared_Component string
	// Exempt is the single opt_out_recursion_ban glob.
	Exempt string
}

// The default lint.json with one opt_out_recursion_ban entry added, for the
// fixtures that prove an exempt file contributes nothing to the call graph.
func test_lint_json_recurse_exempt(
	t *testing.T, input *test_lint_json_recurse_exempt_input,
) (data []byte) {
	t.Helper()
	return []byte(configuration_document(map[string]any{
		"shared_component":      input.Shared_Component,
		"opt_out_recursion_ban": []string{input.Exempt},
	}))
}

// Wraps lint.Main, seeding the fixture's MapFS with a default lint.json
// unless one is already present. Tests that need a specific shared_component or
// allowlist pre-seed their own (run_shared_component_output, run_global_api_case,
// run_snapshot_verbatim) and this leaves that untouched.
func lint_main(t *testing.T, input *lint.Main_Input) (code int) {
	t.Helper()
	if fsys, ok := input.Fsys.(fstest.MapFS); ok {
		if _, present := fsys["lint.json"]; !present {
			fsys["lint.json"] = &fstest.MapFile{Data: test_lint_json(t, "", nil)}
		}
		if _, present := fsys["go.mod"]; !present {
			fsys["go.mod"] = &fstest.MapFile{Data: []byte(DOCTRINE_ROOT_GO_MODULE)}
		}
	}
	return lint.Main(input)
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
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}
			code := lint_main(t, &lint.Main_Input{
				Fsys: fsys_map, Stdout: stdout, Stderr: stderr,
			})
			output := stdout.String()
			if tt.Want_Diag == "" {
				if code != 0 {
					t.Errorf("expected exit 0, got %d; output: %s",
						code, output)
				}
				return
			}
			if !strings.Contains(stdout.String(), tt.Want_Diag) {
				t.Errorf("expected output containing %q, got: %s",
					tt.Want_Diag, output)
			}
		})
	}
}

// Test_Scope_Prefix_Filters_Diagnostics verifies the Scope_Prefix filter
// suppresses diagnostics from files outside the prefix while keeping
// in-scope diagnostics in the output. Exercises the within=false bucket
// of diagnostic_within_scope.
func Test_Scope_Prefix_Filters_Diagnostics(t *testing.T) {
	t.Parallel()
	source := "package main\n\n" +
		"func main() { return }\n" +
		"func f() {\n" +
		"\tx := 1\n" +
		"\t_ = x\n" +
		"}\n"
	fsys_map := fstest.MapFS{
		"in_scope/test.go":     &fstest.MapFile{Data: gofmt_must(t, source)},
		"out_of_scope/test.go": &fstest.MapFile{Data: gofmt_must(t, source)},
	}
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	lint_main(t, &lint.Main_Input{
		Fsys:         fsys_map,
		Stdout:       stdout,
		Stderr:       stderr,
		Scope_Prefix: "in_scope/",
	})
	output := stdout.String()
	if strings.Contains(stdout.String(), "out_of_scope/") {
		t.Errorf("expected out_of_scope/ diagnostics to be filtered; got: %s", output)
	}
}

// Test_Scope_Prefix_Must_Exist verifies that a Scope_Prefix naming a path
// absent from Fsys is a hard error (exit 2), not a silent clean pass. Without
// the existence check, `lint doesnt_exist` filters every file out of an
// unsatisfiable scope and reports "all checks passed" — a typo'd path then
// reads as a green run, the worst failure mode for a checker.
func Test_Scope_Prefix_Must_Exist(t *testing.T) {
	t.Parallel()
	source := "package main\n\nfunc main() { return }\n"
	fsys_map := fstest.MapFS{
		"real/test.go": &fstest.MapFile{Data: gofmt_must(t, source)},
	}
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	code := lint_main(t, &lint.Main_Input{
		Fsys:         fsys_map,
		Stdout:       stdout,
		Stderr:       stderr,
		Scope_Prefix: "doesnt_exist",
	})
	if code != 2 {
		t.Errorf("absent scope must hard-error (exit 2); got %d, stdout: %s",
			code, stdout.String())
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
				if strings.Has_Suffix(k, ".go") {
					fsys_map[k] = &fstest.MapFile{Data: gofmt_must(t, v)}
					continue
				}
				fsys_map[k] = &fstest.MapFile{
					Data: []byte(v)}
			}
			code, output := lint_output_minus(t, fsys_map)
			if tt.Want_Diags == nil {
				if code != 0 {
					t.Errorf("expected exit 0, got %d; output:\n%s",
						code, output)
				}
				return
			}
			if code == 0 {
				t.Errorf("expected nonzero exit; output:\n%s", output)
			}
			for _, w := range tt.Want_Diags {
				if !strings.Contains(output, w) {
					t.Errorf("expected output containing %q; got:\n%s",
						w, output)
				}
			}
			for _, f := range tt.Forbid {
				if strings.Contains(output, f) {
					t.Errorf("expected output NOT to contain %q; got:\n%s",
						f, output)
				}
			}
		})
	}
}

// Mirrors lint.Main's stdout emission — tier-2 gating and the same
// "position: message" line format — but drops single-module diagnostics.
func lint_output_minus(
	t *testing.T, fsys fstest.MapFS,
) (code int, output string) {
	t.Helper()
	if _, present := fsys["go.mod"]; !present {
		fsys["go.mod"] = &fstest.MapFile{Data: []byte(DOCTRINE_ROOT_GO_MODULE)}
	}
	all, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys: fsys, Shared_Component: DOCTRINE_SHARED_COMPONENT_DIRECTORY,
		// The doctrine table targets other rules; the invariant rules (which fire
		// on every typed field/param/return) and the deterministic tier (which now
		// binds every pure package) are exercised by their own tests, so disable
		// both wholesale here — "." exempts the invariant tree, "**" the pure tree.
		Invariant_Exempt_Packages: []string{"**"},
		Pure_But_Indeterministic:  []string{"**"},
	})
	if err != nil {
		t.Fatalf("Check_File_System: %v", err)
	}
	var kept []lint.Diagnostic
	for _, d := range all {
		// Doctrine-table fixtures predate the single-module collapse and still
		// carry per-directory go.mod files; the single-module rule has its own
		// tests, so drop its diagnostics here.
		if d.Name == "single-module" {
			continue
		}
		kept = append(kept, d)
	}
	has_tier1 := false
	for _, d := range kept {
		if d.Tier == 1 {
			has_tier1 = true
			break
		}
	}
	builder := &strings.Builder{}
	emitted := 0
	for _, d := range kept {
		if has_tier1 {
			if d.Tier == 2 {
				continue
			}
		}
		emitted++
		fmt.Fprintf(builder, "%s: %s\n", d.Position, d.Message)
	}
	if emitted > 0 {
		return 1, builder.String()
	}
	return 0, builder.String()
}

// Test_Helper_Function_Names_Allowed verifies helper is ordinary vocabulary now that the invariant
// doctrine uses it as the precise name for companion functions.
func Test_Helper_Function_Names_Allowed(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "bare helper allowed",
			Files: map[string]string{
				"test.go": `package main

func helper() { return }
`,
			},
			Want_Diag: "",
		},

		{
			Name: "suffix helper allowed",
			Files: map[string]string{
				"test.go": `package main

func read_helper() { return }
`,
			},
			Want_Diag: "",
		},

		{
			Name: "capitalized Helper allowed",
			Files: map[string]string{
				"test.go": `package main

func Helper_Func() { return }
`,
			},
			Want_Diag: "",
		},

		{
			Name: "helper as middle segment allowed",
			Files: map[string]string{
				"test.go": `package main

func read_helper_thing() { return }
`,
			},
			Want_Diag: "",
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
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_Banned_Identifiers_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "helpme not flagged",
			Files: map[string]string{
				"test.go": `package main

func helpme() { return }
`,
			},
			Want_Diag: "",
		},
	})
}

// Test_Naming_Constant_Casing verifies the const-specific carve-out of the Name
// Style rule: every const must be SCREAMING_SNAKE_CASE, not the general Ada_Case
// or snake_case its export status would otherwise dictate — including a name
// that already happens to satisfy ada_case_re's acronym leniency
// (Retry_Count_Max), which must still fail since it isn't fully uppercase. The
// rule reaches past package-level exported consts to unexported and
// function-local consts alike.
func Test_Naming_Constant_Casing(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "exported const not screaming is flagged",
			Files: map[string]string{
				"test.go": `package main

const BadName = 0
`,
			},
			Want_Diag: "BadName -> BAD_NAME",
		},
		{
			Name: "exported const already passing ada_case_re must still flag",
			Files: map[string]string{
				"test.go": `package main

const Retry_Count_Max = 3
`,
			},
			Want_Diag: "Retry_Count_Max -> RETRY_COUNT_MAX",
		},
		{
			Name: "exported const already screaming is clean",
			Files: map[string]string{
				"test.go": `package main

const RETRY_COUNT_MAX = 3
`,
			},
			Want_Diag: "",
		},
		{
			Name: "unexported const is flagged",
			Files: map[string]string{
				"test.go": `package main

const retry_count_max = 3
`,
			},
			Want_Diag: "retry_count_max -> RETRY_COUNT_MAX",
		},
		{
			Name: "function-local const is flagged",
			Files: map[string]string{
				"test.go": `package main

func F() {
	const Local_Thing = 1
	println(Local_Thing)
}
`,
			},
			Want_Diag: "Local_Thing -> LOCAL_THING",
		},
	})
}

// Test_Variable_Shadow_Package_Wide verifies the outer scope a local is judged
// against is the whole package, not the one file: a sibling file's top-level
// name shadows just as the file's own does. An external test package shares the
// directory but not the name space, so its names stay out.
func Test_Variable_Shadow_Package_Wide(t *testing.T) {
	t.Parallel()
	const SHADOW = `The variable "helper" shadows a variable in an outer scope`
	sibling := fixture_package("foo") +
		"\nfunc helper() (value int) {\n\treturn 0\n}\n"
	body := "func f() (value int) {\n\thelper := 1\n\treturn helper\n}\n"
	workspace := func(clause string) (files map[string]string) {
		return map[string]string{
			"lib/foo/foo.go":      sibling,
			"lib/foo/foo_test.go": clause + "\n\n" + body,
		}
	}
	internal := run_shared_component_output(t, workspace("package foo"), "lib")
	if !strings.Contains(internal, SHADOW) {
		t.Fatalf("a sibling file's declaration must shadow; got: %s", internal)
	}
	external := run_shared_component_output(t, workspace("package foo_test"), "lib")
	if strings.Contains(external, SHADOW) {
		t.Fatalf("package foo_test is a separate name space; got: %s", external)
	}
}

// Test_Array_Capacity verifies that an array type's capacity must be a named
// constant: a literal is flagged wherever the type appears, a named constant is
// not, and neither `[...]T{…}` nor a slice carries a capacity to judge.
func Test_Array_Capacity(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "literal on a type declaration",
			Files: map[string]string{"test.go": `package main

type Buffer [16]byte
`},
			Want_Diag: "The array capacity 16 is a literal",
		},
		{
			Name: "literal nested in a result type",
			Files: map[string]string{"test.go": `package main

func f() (buffer *[4]int) {
	return nil
}
`},
			Want_Diag: "array capacity 4 is a literal",
		},
		{
			Name: "named constant",
			Files: map[string]string{"test.go": `package main

// BUFFER_SIZE_MAX is a fixture.
const BUFFER_SIZE_MAX = 16

func f() (first byte) {
	var buffer [BUFFER_SIZE_MAX]byte
	return buffer[0]
}
`},
			Want_Diag: "",
		},
		{
			Name: "ellipsis defers to the compiler",
			Files: map[string]string{"test.go": `package main

func f() (first int) {
	digits := [...]int{1, 2, 3}
	return digits[0]
}
`},
			Want_Diag: "",
		},
		{
			Name: "slice carries no capacity",
			Files: map[string]string{"test.go": `package main

func f() (buffer []byte) {
	return nil
}
`},
			Want_Diag: "",
		},
	})
}

// Test_No_Empty_Function_Body verifies that empty-body functions and methods are
// flagged while interface method signatures (Body == nil) are allowed.
func Test_No_Empty_Function_Body(t *testing.T) {
	t.Parallel()
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

type T struct {
	// X is a fixture.
	X int
}

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

// Test_Closure_Bodies pins closures to call sequences only, while test fixtures
// keep freedom to build compact callbacks with local setup and checks.
func Test_Closure_Bodies(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "empty closure allowed",
			Files: map[string]string{"rule.go": `package main

func f() {
	callback := func() {}
	callback()
}
`},
			Want_Diag: "",
		},
		{
			Name: "one call closure allowed",
			Files: map[string]string{"rule.go": `package main

func called() { return }

func f() {
	callback := func() { called() }
	callback()
}
`},
			Want_Diag: "",
		},
		{
			Name: "two call closure allowed",
			Files: map[string]string{"rule.go": `package main

func called() { return }

func f() {
	callback := func() {
		called()
		called()
	}
	callback()
}
`},
			Want_Diag: "",
		},
		{
			Name: "non-call closure flagged",
			Files: map[string]string{"rule.go": `package main

func f() {
	callback := func() { return }
	callback()
}
`},
			Want_Diag: "closure body must be empty or contain only function calls",
		},
	})
}

// Test_Closure_Bodies_Part2 covers returned delegation, assignments, return
// position, ordinary functions, and test-file exemption without breaching
// function size limit.
func Test_Closure_Bodies_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "return call closure allowed",
			Files: map[string]string{"rule.go": `package main

func called() (result int) { return 0 }

func f() {
	callback := func() (result int) { return called() }
	callback()
}
`},
			Want_Diag: "",
		},
		{
			Name: "assignment call closure flagged",
			Files: map[string]string{"rule.go": `package main

func called() (result int) { return 0 }

func f() {
	callback := func() { result := called() }
	callback()
}
`},
			Want_Diag: "closure body must be empty or contain only function calls",
		},
		{
			Name: "calls then return call closure allowed",
			Files: map[string]string{"rule.go": `package main

func called() (result int) { return 0 }

func f() {
	callback := func() (result int) {
		called()
		return called()
	}
	callback()
}
`},
			Want_Diag: "",
		},
		{
			Name: "return before last call flagged",
			Files: map[string]string{"rule.go": `package main

func called() (result int) { return 0 }

func f() {
	callback := func() (result int) {
		return called()
		called()
	}
	callback()
}
`},
			Want_Diag: "closure body must be empty or contain only function calls",
		},
		{
			Name: "function declaration unaffected",
			Files: map[string]string{"rule.go": `package main

func called() { return }

func f() {
	called()
	called()
}
`},
			Want_Diag: "",
		},
		{
			Name: "test file exempt",
			Files: map[string]string{"rule_test.go": `package main

func called() { return }

func f() {
	callback := func() {
		called()
		called()
	}
	callback()
}
`},
			Want_Diag: "",
		},
	})
}

// Test_Closure_Bodies_Call_Arguments preserves argument construction inside
// one direct delegation call; banning it would force pointless wrappers.
func Test_Closure_Bodies_Call_Arguments(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "call argument may contain call",
			Files: map[string]string{"rule.go": `package main

func input() (result int) { return 0 }
func called(value int) { return }

func f() {
	callback := func() { called(input()) }
	callback()
}
`},
			Want_Diag: "",
		},
	})
}

// Test_No_Interfaces verifies method-bearing interfaces fail wherever they appear.
func Test_No_Interfaces(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "named method-set interface flagged",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func main() (result int) {
	defer func() {
	}()
	return 0
}

type Iface interface {
	M()
}
`,
			},
			Want_Diag: "Do not declare an interface",
		},

		{
			Name: "inline interface in parameter flagged",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func main() (result int) {
	defer func() {
	}()
	return 0
}

func F(x interface{ M() }) (result int) { return 0 }
`,
			},
			Want_Diag: "Do not declare an interface",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_No_Interfaces_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

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
			Want_Diag: "Do not declare an interface",
		},
	})
}

// Test_No_Interfaces_Empty verifies the any alias is flagged.
func Test_No_Interfaces_Empty(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "any parameter flagged",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func main() (result int) {
	defer func() {
	}()
	return F(0)
}

func F(x any) (result int) {
	defer func() {
	}()
	return 0
}
`,
			},
			Want_Diag: "Do not use any",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_No_Interfaces_Empty_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "empty interface flagged",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func main() (result int) {
	defer func() {
	}()
	return F(0)
}

func F(x interface{}) (result int) {
	defer func() {
	}()
	return 0
}
`,
			},
			Want_Diag: "Do not declare an interface",
		},

		{
			Name: "interface and any in test file allowed",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func main() (result int) {
	defer func() {
	}()
	return 0
}
`,
				"fixture_test.go": `package main

import "testing"

// Test_Fixture is a fixture.
func Test_Fixture(t *testing.T) {
	var value any = 0
	var empty interface{} = 0
	var method interface{ M() }
}
`,
			},
			Want_Diag: "",
		},
	})
}

// Test_No_Generics rejects generic functions and types while preserving ordinary indexing.
func Test_No_Generics(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "generic function flagged",
			Files: map[string]string{
				"test.go": `package main

func Identity[Value any](value Value) (result Value) { return value }
`,
			},
			Want_Diag: "Do not use generics",
		},
		{
			Name: "generic type alias flagged",
			Files: map[string]string{
				"test.go": `package main

type Values[Value any] = []Value
`,
			},
			Want_Diag: "Do not use generics",
		},
		{
			Name: "generic test helper flagged",
			Files: map[string]string{
				"test_test.go": `package main

func helper[Value any](value Value) (result Value) { return value }
`,
			},
			Want_Diag: "Do not use generics",
		},
		{
			Name: "generated generic function flagged",
			Files: map[string]string{
				"test.go": `// Code generated by fixture. DO NOT EDIT.
package main

func Identity[Value any](value Value) (result Value) { return value }
`,
			},
			Want_Diag: "Do not use generics",
		},
		{
			Name: "ordinary indexing allowed",
			Files: map[string]string{
				"test.go": `package main

func value_at(values []int, index int) (result int) { return values[index] }
`,
			},
			Want_Diag: "",
		},
	})
}

// Test_Shared_Generics_Exemptions keeps exceptions limited to exact package paths.
func Test_Shared_Generics_Exemptions(t *testing.T) {
	t.Parallel()
	for _, directory := range []string{"shared/testify", "shared/slices", "shared/sort"} {
		for _, suffix := range []string{"", "_extra", "/nested"} {
			for _, filename := range []string{"generic.go", "generic_test.go"} {
				t.Run(directory+suffix+"/"+filename, func(t *testing.T) {
					files := fstest.MapFS{
						directory + suffix + "/" + filename: &fstest.MapFile{Data: []byte(`package fixture

type Box[Value any] struct { Value Value }

func Identity[Value any](value Value) (result Value) { return value }
`)},
					}
					diags, err := lint.Check_File_System(&lint.Check_File_System_Input{
						Fsys: files,
					})
					if err != nil {
						t.Fatalf("Check_File_System: %v", err)
					}
					count := 0
					for _, diagnostic := range diags {
						if strings.Contains(diagnostic.Message, "Do not use generics") {
							count++
						}
					}
					want := 0
					if suffix != "" {
						want = 2
					}
					if count != want {
						t.Fatalf("generic diagnostics: got %d, want %d", count, want)
					}
				})
			}
		}
	}
}

// Test_Instrumentation_Packages_Permit_Generics keeps one instrumentation adapter
// usable across arbitrary observed types.
func Test_Instrumentation_Packages_Permit_Generics(t *testing.T) {
	t.Parallel()
	files := fstest.MapFS{
		"instrumentation/generic.go": &fstest.MapFile{Data: []byte(`package instrumentation

type Box[Value any] struct {
	Value Value
}

func Identity[Value any](value Value) (result Value) { return value }
`)},
	}
	diags, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys:                     files,
		Instrumentation_Packages: []string{"instrumentation/**"},
	})
	if err != nil {
		t.Fatalf("Check_File_System: %v", err)
	}
	for _, diagnostic := range diags {
		if strings.Contains(diagnostic.Message, "Do not use generics") {
			t.Fatal("instrumentation packages must permit generics")
		}
	}
}

// Test_No_Interfaces_Exempt_Package verifies opt_out_interface_ban permits both
// the empty interface and its any alias in a listed package.
func Test_No_Interfaces_Exempt_Package(t *testing.T) {
	t.Parallel()
	files := fstest.MapFS{
		"dynamic/box.go": &fstest.MapFile{Data: []byte(`package dynamic

// Box holds a caller-owned value.
type Box struct {
	// Value is the held value.
	Value any
}

// Unwrap returns the held value.
func Unwrap(box Box) (result interface{}) { return box.Value }
`)},
	}
	diags, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys:             files,
		Interface_Exempt: []string{"dynamic/**"},
	})
	if err != nil {
		t.Fatalf("Check_File_System: %v", err)
	}
	for _, diagnostic := range diags {
		if strings.Contains(diagnostic.Message, "Do not use any") ||
			strings.Contains(diagnostic.Message, "Do not declare an interface") {
			t.Fatal("opt_out_interface_ban packages must permit any and interface{}")
		}
	}
}

// Test_No_Method_Stdlib_String_Flagged verifies interface satisfaction grants no escape.
func Test_No_Method_Stdlib_String_Flagged(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "stdlib Stringer match flagged",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func main() (result int) {
	defer func() {
	}()
	return 0
}

type T struct {
	// X is a fixture.
	X int
}

func (t T) String() (result string) {
	defer func() {
	}()
	return ""
}
`,
			},
			Want_Diag: "Methods are banned",
		},
	})
}

// Test_No_Method_Stdlib_Error_Flagged covers error.Error shape.
func Test_No_Method_Stdlib_Error_Flagged(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "stdlib error match flagged",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func main() (result int) {
	defer func() {
	}()
	return 0
}

type T struct {
	// X is a fixture.
	X int
}

func (t T) Error() (result string) {
	defer func() {
	}()
	return ""
}
`,
			},
			Want_Diag: "Methods are banned",
		},
	})
}

// Test_No_Method_Stdlib_Read_Flagged covers io.Reader.Read shape.
func Test_No_Method_Stdlib_Read_Flagged(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "stdlib Read match flagged",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func main() (result int) {
	defer func() {
	}()
	return 0
}

type T struct {
	// X is a fixture.
	X int
}

func (t T) Read(p []byte) (n int, err error) {
	defer func() {
	}()
	return 0, nil
}
`,
			},
			Want_Diag: "Methods are banned",
		},
	}
	run_diag_table(t, tests)
}

// Test_No_Method_Stdlib_Scan_Flagged covers interface{} normalization.
func Test_No_Method_Stdlib_Scan_Flagged(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "stdlib Scan flagged",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func main() (result int) {
	defer func() {
	}()
	return 0
}

type T struct {
	// X is a fixture.
	X int
}

func (t T) Scan(x int) (err error) {
	return nil
}
`,
			},
			Want_Diag: "Methods are banned",
		},
	})
}

// Test_No_Method_Free_Function_Allowed keeps free functions outside method rule.
func Test_No_Method_Free_Function_Allowed(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "free function ignored",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func main() (result int) {
	defer func() {
	}()
	return F()
}

func F() (result int) {
	defer func() {
	}()
	return 0
}
`,
			},
			Want_Diag: "",
		},
	})
}

// Test_No_Method_Assertion_Builder_Allowed pins exception to owner and exact receiver.
func Test_No_Method_Assertion_Builder_Allowed(t *testing.T) {
	t.Parallel()
	source_text := "// Package aver is a fixture.\n" +
		"package aver\n\n" +
		"// Assertion_Builder is a fixture.\n" +
		"type Assertion_Builder struct{}\n\n" +
		"// Sometimes is a fixture builder link.\n" +
		"func (builder Assertion_Builder) Sometimes() (next Assertion_Builder) {\n" +
		"\treturn builder\n}\n\n" +
		"func (builder *Assertion_Builder) assertion_plan() (next *Assertion_Builder) {\n" +
		"\treturn builder\n}\n"
	for _, filename := range []string{
		"shared/sim/aver/enforcement.go",
		"shared/sim/aver/default/aver_default_noop.go",
	} {
		diags, err := lint.Check_Source(filename, source_text)
		if err != nil {
			t.Fatalf("Check_Source(%q): %v", filename, err)
		}
		for _, diagnostic := range diags {
			if diagnostic.Tier == 1 {
				t.Fatalf("builder fixture has tier-one diagnostic at %s: %s",
					filename, diagnostic.Message)
			}
			if strings.Contains(diagnostic.Message, "Methods are banned") {
				t.Errorf("assertion builder method flagged at %s", filename)
			}
		}
	}
}

// Test_No_Method_Assertion_Builder_Boundary rejects impersonation outside exact owner.
func Test_No_Method_Assertion_Builder_Boundary(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Filename string
		Package  string
		Receiver string
	}{
		{Filename: "shared/sim/aver/enforcement.go", Package: "aver", Receiver: "Other"},
		{Filename: "shared/sim/aver/child/method.go", Package: "aver",
			Receiver: "Assertion_Builder"},
		{Filename: "other/enforcement.go", Package: "aver", Receiver: "Assertion_Builder"},
		{Filename: "shared/sim/aver/enforcement.go", Package: "other",
			Receiver: "Assertion_Builder"},
		{Filename: "fixture_test.go", Package: "fixture_test", Receiver: "Assertion_Builder"},
	}
	for _, test := range tests {
		source_text := "// Package " + test.Package + " is a fixture.\n" +
			"package " + test.Package + "\n\n" +
			"// " + test.Receiver + " is a fixture.\n" +
			"type " + test.Receiver + " struct{}\n\n" +
			"// Link is a fixture.\n" +
			"func (value " + test.Receiver + ") Link() { return }\n"
		diags, err := lint.Check_Source(test.Filename, source_text)
		if err != nil {
			t.Fatalf("Check_Source(%q): %v", test.Filename, err)
		}
		found := false
		for _, diagnostic := range diags {
			if strings.Contains(diagnostic.Message, "Methods are banned") {
				found = true
			}
		}
		if !found {
			t.Errorf("method exemption escaped filename=%q package=%q receiver=%q",
				test.Filename, test.Package, test.Receiver)
		}
	}
}

// Test_No_Method_Write_Flagged covers former stdlib name with wrong signature.
func Test_No_Method_Write_Flagged(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "Write with wrong result list flagged",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func main() (result int) {
	defer func() {
	}()
	return 0
}

type T struct {
	// X is a fixture.
	X int
}

func (t T) Write(p []byte) (err error) {
	return nil
}
`,
			},
			Want_Diag: "Methods are banned",
		},
	})
}

// Test_No_Method_Unknown_Name_Flagged covers method without former interface exception.
func Test_No_Method_Unknown_Name_Flagged(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "unknown method name flagged",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func main() (result int) {
	defer func() {
	}()
	return 0
}

type T struct {
	// X is a fixture.
	X int
}

func (t T) Foo() (result int) {
	defer func() {
	}()
	return 0
}
`,
			},
			Want_Diag: "Methods are banned",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_Test_Package_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "non-test file unaffected",
			Files: map[string]string{
				"foo.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func f() (result int) {
	defer func() {
	}()
	return 1
}
`,
			},
			Want_Diag: "",
		},
	})
}

// Test_No_Recursion verifies that direct and mutual recurse cycles are
// detected via the per-file Ident-based call graph.
func Test_No_Recursion(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{Name: "direct recurse",
			Files: map[string]string{"test.go": `package main
func f() {
	f()
}`}, Want_Diag: "recurse"},

		{Name: "recurse inside nested block",
			Files: map[string]string{"test.go": `package main
func f(x int) (result int) {
	if x > 0 {
		return f(x - 1)
	}
	return 0
}`}, Want_Diag: "recurse"},

		{Name: "no recurse: calls other function",
			Files: map[string]string{"test.go": `package main
func f_inner() { return }
func f() {
	f_inner()
}`}, Want_Diag: ""},

		{Name: "no recurse: no calls",
			Files: map[string]string{"test.go": `package main
func f() { return }`}, Want_Diag: ""},

		{Name: "mutual 2-cycle",
			Files: map[string]string{"test.go": `package main
func entry() { a(); b() }
func a() { b() }
func b() { a() }`}, Want_Diag: "recurse in a cycle"},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			fsys_map := make(fstest.MapFS)
			for k, v := range tt.Files {
				fsys_map[k] = &fstest.MapFile{Data: gofmt_must(t, v)}
			}
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}
			code := lint_main(t, &lint.Main_Input{
				Fsys: fsys_map, Stdout: stdout, Stderr: stderr,
			})

			output := stdout.String()
			if tt.Want_Diag == "" {
				if code != 0 {
					t.Errorf("expected exit 0, got %d; output: %s",
						code, output)
				}
			} else {
				if !strings.Contains(stdout.String(), tt.Want_Diag) {
					t.Errorf("expected output containing %q, got: %s",
						tt.Want_Diag, output)
				}
			}
		})
	}
}

// Additional cases, split to keep each function within the length limit.
func Test_No_Recursion_Part2(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{Name: "3-cycle",
			Files: map[string]string{"test.go": `package main
func entry() { a(); b(); c() }
func a() { b() }
func b() { c() }
func c() { a() }`}, Want_Diag: "recurse in a cycle"},

		{Name: "non-cycle chain",
			Files: map[string]string{"test.go": `package main
func a() { a_b() }
func a_b() { a_b_c() }
func a_b_c() { return }`}, Want_Diag: ""},

		{Name: "cycle plus non-cycle helper",
			Files: map[string]string{"test.go": `package main
func entry() { a(); b(); inner() }
func a() { b(); inner() }
func b() { a() }
func inner() { return }`}, Want_Diag: "recurse in a cycle"},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			fsys_map := make(fstest.MapFS)
			for k, v := range tt.Files {
				fsys_map[k] = &fstest.MapFile{Data: gofmt_must(t, v)}
			}
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}
			code := lint_main(t, &lint.Main_Input{
				Fsys: fsys_map, Stdout: stdout, Stderr: stderr,
			})

			output := stdout.String()
			if tt.Want_Diag == "" {
				if code != 0 {
					t.Errorf("expected exit 0, got %d; output: %s",
						code, output)
				}
			} else {
				if !strings.Contains(stdout.String(), tt.Want_Diag) {
					t.Errorf("expected output containing %q, got: %s",
						tt.Want_Diag, output)
				}
			}
		})
	}
}

// Test_Main_First verifies that main, Main, and TestMain must be the first
// function declaration in the file, with TestMain exempt from snake/Ada casing.
func Test_Main_First(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{Name: "main not first",
			Files: map[string]string{"test.go": `package main
func other() { return }
func main() { return }`}, Want_Diag: "first in the file"},

		{Name: "Main not first",
			Files: map[string]string{"test.go": `package implementation
func helper() { return }
func Main() { return }`}, Want_Diag: "first in the file"},

		{Name: "main is first",
			Files: map[string]string{"test.go": `package main
func main() { return }
func other() { return }`}, Want_Diag: ""},

		{Name: "Main is first",
			Files: map[string]string{"test.go": `// Package impl is a fixture.
package implementation
// Main is the entry point.
func Main() { return }
func inner() { return }`}, Want_Diag: ""},

		{Name: "no main: no diag",
			Files: map[string]string{"test.go": `// Package x is a fixture.
package x
func a() { return }
func b() { return }`}, Want_Diag: ""},

		{Name: "decls before main: still OK",
			Files: map[string]string{"test.go": `package main
const X = 1
type T int
func main() { return }`}, Want_Diag: ""},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			fsys_map := make(fstest.MapFS)
			for k, v := range tt.Files {
				fsys_map[k] = &fstest.MapFile{Data: gofmt_must(t, v)}
			}
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}
			code := lint_main(t, &lint.Main_Input{
				Fsys: fsys_map, Stdout: stdout, Stderr: stderr,
			})

			output := stdout.String()
			if tt.Want_Diag == "" {
				if code != 0 {
					t.Errorf("expected exit 0, got %d; output: %s",
						code, output)
				}
			} else {
				if !strings.Contains(stdout.String(), tt.Want_Diag) {
					t.Errorf("expected output containing %q, got: %s",
						tt.Want_Diag, output)
				}
			}
		})
	}
}
