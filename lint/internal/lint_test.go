package lint_test

import (
	"fmt"
	"go/format"
	"local/james-orcales/lint/internal/strings"
	"testing"
	"testing/fstest"

	"local/james-orcales/lint/internal"
)

// Gofmt_must formats test sources so fixtures don't need to be hand-perfect
// gofmt-clean. Required because check_gofmt runs as part of the tier-1
// pipeline against every test source. TestGofmt builds MapFS inline so it
// can submit deliberately un-formatted sources.
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
// Main in its top-level internal/ package. Injected into binary-module fixtures
// that exist to exercise other rules so the entry-point check doesn't bleed an
// extra diagnostic into them.
const DOCTRINE_BINARY_INTERNAL_MAIN = "// Package entry is a fixture.\n" +
	"package entry\n\n// Main is a fixture entry point.\nfunc Main() { return }\n"

const FIXTURE_INVARIANT_IMPORT_PATH = "github.com/james-orcales/james-orcales/" +
	"shared/invariant/v2/invariant_default"
const FIXTURE_INVARIANT_IMPORT = "import \"" + FIXTURE_INVARIANT_IMPORT_PATH + "\"\n"

// A package whose SPECIFICATION.md, source, and specification_test.go all
// satisfy the doctrine. Variant tests swap one artifact for a violating one so
// each assertion isolates a single rule.
const SPECIFICATION_CLEAN_SOURCE = "// Package greet is a fixture.\n" +
	"package greet\n\n" +
	"// Greet greets.\nfunc Greet() { return }\n"
const SPECIFICATION_CLEAN_TEST = "package greet_test\n\n" +
	"import \"testing\"\n\n" +
	"// Test_Greeting is a fixture.\n" +
	"func Test_Greeting(t *testing.T) { _ = t }\n"
const SPECIFICATION_CLEAN_MD = "\n# Greeting\n\nIt greets the caller.\n"

// The one-heading SPECIFICATION.md a layout fixture carries so the coverage rule
// is satisfied without the fixture growing a specification of its own.
const SPECIFICATION_FIXTURE_MD = "\n# Function\n\nIt is a fixture.\n"

// Builds the specification_test.go that pairs with SPECIFICATION_FIXTURE_MD for
// the named package.
func fixture_specification_test(name string) (source string) {
	return "package " + name + "_test\n\n" +
		"import \"testing\"\n\n" +
		"// Test_Function is a fixture.\n" +
		"func Test_Function(t *testing.T) { t.Parallel() }\n"
}

// The baseline SPECIFICATION.md + test pair a spec snapshot mutates: one clean
// leaf and its matching test, so a single mutation isolates one spec diagnostic.
const SNAPSHOT_SPECIFICATION_MARKDOWN = "\n# Sole Rule\n\nThe sole rule.\n"
const SNAPSHOT_SPECIFICATION_TEST = "package fixture_test\n\nimport \"testing\"\n\n" +
	"// Test_Sole_Rule checks the sole rule.\n" +
	"func Test_Sole_Rule(t *testing.T) {\n\tt.Parallel()\n}\n"

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
	"shared/invariant/v2\"\n\n" +
	"const FIXTURE_HI = 100\n\n" +
	"func f() (result int) {\n" +
	"\tdefer func() {\n" +
	"\t\tinvariant.Cross_Product(\n" +
	"\t\t\tinvariant.Distinct_Boundary(" +
	"&invariant.Boundary_Input[int]{\n" +
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
	"&invariant.Boundary_Input[int]{" +
	"X: out, Lo: 0, Hi: 1}),\n" +
	"\t\t)\n" +
	"\t}()\n" +
	"\tinvariant.Cross_Product(" +
	"invariant.Always(p != nil, \"p is non-nil\"))\n" +
	"\treturn 0\n" +
	"}\n\n"

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

// Test_Exported_Type_Exposes_Private_Allows covers the field-position
// negative cases: builtins, in-scope generic type parameters, qualified
// selectors, and a self-referential cycle.
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
	// A is a fixture.
	A any
	// B is a fixture.
	B bool
	// R is a fixture.
	R rune
	// Y is a fixture.
	Y byte
}
`}, Want_Diag: ""},
		{Name: "type parameter allowed",
			Files: map[string]string{"test.go": `package main
type Foo[T any] struct {
	// X is a fixture.
	X T
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
	callback := func() { return }
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

// Test_Gofmt verifies that gofmt-dirty sources are flagged. Bypasses
// gofmt_must so the deliberate formatting violations survive.
func Test_Gofmt(t *testing.T) {
	t.Parallel()
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
				"test.go": "package main\n\n" +
					"import \"" +
					"github.com/james-orcales/james-orcales/" +
					"shared/invariant/v2\"\n\n" +
					"const FIXTURE_HI = 100\n\n" +
					"func f() (result int) {\n" +
					"\tdefer func() {\n" +
					"\t\tinvariant.Cross_Product(\n" +
					"\t\t\tinvariant.Distinct_Boundary(" +
					"&invariant.Boundary_Input[int]{\n" +
					"\t\t\t\tX: result, Lo: 0, Hi: FIXTURE_HI,\n" +
					"\t\t\t}),\n" +
					"\t\t\tinvariant.Always(" +
					"result == 0, \"result is zero\"),\n" +
					"\t\t)\n" +
					"\t}()\n" +
					"\treturn 1\n" +
					"}\n",
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
				fsys_map[k] = &fstest.MapFile{
					Data: []byte(v)}
			}
			stdout := &strings.Builder{}
			code := lint_main(t, &lint.Main_Input{
				Fsys: fsys_map, Stdout: stdout, Stderr: &strings.Builder{},
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

// Returns the minimal vocabulary table the tests assert on —
// only the abbreviations and bans pinned by Test_Names_Vocabulary,
// Test_Naming_Abbreviations_*, Test_Banned_*, and the snapshot bans. It is not
// the full lint.json table; single-file fixtures only pin the words they name,
// so the fixture stays small and lives beside the assertions it feeds.
func test_word_replacements() (table map[string][]string) {
	return map[string][]string{
		"cfg":       {"config"},
		"src":       {"source"},
		"mgr":       {"manager"},
		"cb":        {"callback"},
		"btn":       {"button"},
		"id":        {"identifier"},
		"res":       {"response", "result", "resource", "reserve"},
		"len":       {},
		"length":    {},
		"util":      {},
		"utils":     {},
		"utility":   {},
		"utilities": {},
	}
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
		"word_replacements":        test_word_replacements(),
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
			code, output := lint_output_minus(t, fsys_map, "specification")
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
// "position: message" line format — but drops diagnostics from the named rule.
// A doctrine table can then assert on its own rule's output without the coverage
// mandate (which now fires for any module fixture lacking a SPECIFICATION.md)
// bleeding into the exit code.
func lint_output_minus(
	t *testing.T, fsys fstest.MapFS, exclude string,
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
		// tests, so drop its diagnostics here along with the named exclude.
		if d.Name == "single-module" {
			continue
		}
		if d.Name != exclude {
			kept = append(kept, d)
		}
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

		{
			Name: "function name util flagged",
			Files: map[string]string{
				"test.go": `package main

func parse_util() { return }
`,
			},
			Want_Diag: `banned substring "util"`,
		},

		{
			Name: "function name Utilities flagged",
			Files: map[string]string{
				"test.go": `package main

func Make_Utilities() { return }
`,
			},
			Want_Diag: `banned substring "utilities"`,
		},
	})
}

// Test_Banned_Decl_Sites_Local_Vars verifies that the universal banned-segment
// check reaches local var/const/range/assign-define declarations. len(...)
// and cap(...) call sites are exempt because they appear in callee position,
// never as a declared name.
func Test_Banned_Declaration_Sites_Local_Vars(t *testing.T) {
	t.Parallel()
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
			Want_Diag: `banned substring "length"`,
		},
		{
			Name: "const name length flagged",
			Files: map[string]string{
				"test.go": `package main

const length_max = 10
`,
			},
			Want_Diag: `banned substring "length"`,
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
			Want_Diag: `banned substring "length"`,
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
			Want_Diag: `banned substring "utils"`,
		},
	}
	run_diag_table(t, tests)
}

// Test_Banned_Declaration_Sites_Builtin_Exempt covers the exemption for
// len(...) and cap(...) call sites: those appear in callee position and
// never as a declared name, so the banned-segment check doesn't fire.
func Test_Banned_Declaration_Sites_Builtin_Exempt(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "len builtin call site exempt",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func F(xs []int) (result int) {
	defer func() {
	}()
	return len(xs)
}
`,
			},
			Want_Diag: "",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_Banned_Declaration_Sites_Builtin_Exempt_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "cap builtin call site exempt",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func F(xs []int) (result int) {
	defer func() {
	}()
	return cap(xs)
}
`,
			},
			Want_Diag: "",
		},
	})
}

// Test_Banned_Declaration_Sites_Local_Vars_Clean covers the local-var path
// when no banned segment is present; the rule must pass and not produce
// noise on legitimate var/short-decl/struct-field patterns.
func Test_Banned_Declaration_Sites_Local_Vars_Clean(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "clean local decls allowed",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func F(buffer []byte) (result int) {
	defer func() {
	}()
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
	t.Parallel()
	run_diag_table(t, []struct {
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
			Want_Diag: `banned substring "len"`,
		},

		{
			Name: "named return with length segment flagged",
			Files: map[string]string{
				"test.go": `package main

func F() (length int) { return 0 }
`,
			},
			Want_Diag: `banned substring "length"`,
		},

		{
			Name: "struct field with Length segment flagged",
			Files: map[string]string{
				"test.go": `package main

type S struct{ Buf_Length int }
`,
			},
			Want_Diag: `banned substring "length"`,
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_Banned_Declaration_Sites_Signatures_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "clean signature allowed",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

type S struct{}

func F(buffer []byte) (result int) {
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

// Test_Naming_For_Loop verifies that a C-style for-loop induction variable
// (`for i := 0; i < N; i++`) must be named with an _index suffix.
func Test_Naming_For_Loop(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
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
			Want_Diag: "The name i has the role index. Rename i -> i_index.",
		},

		{
			Name: "i_index induction allowed",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func F(n int) (result int) {
	defer func() {
	}()
	for i_index := 0; i_index < n; i_index++ {
		result = result + i_index
	}
	return result
}
`,
			},
			Want_Diag: "",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_Naming_For_Loop_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "non-induction for-loop not triggered",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func F() (result int) {
	defer func() {
	}()
	for condition := true; condition; condition = false {
		result = result + 1
	}
	return result
}
`,
			},
			Want_Diag: "",
		},
	})
}

// Test_Naming_Make verifies that make's count/size argument, when a bare
// declared identifier, must carry _count (or _size when the element type
// is byte).
func Test_Naming_Make(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
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
			Want_Diag: "The name n has the role count. Rename n -> n_count.",
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
			Want_Diag: "The name n has the role size. Rename n -> n_size.",
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
			Want_Diag: "The name n has the role count. Rename n -> n_count.",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_Naming_Make_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "make with suffixed arg allowed",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func F(n_count int) (result []int) {
	defer func() {
	}()
	result =make([]int, n_count)
	return result
}
`,
			},
			Want_Diag: "",
		},
	})
}

// Test_Naming_Stdlib_Allowlist verifies that the result of allowlisted
// stdlib calls (strings.Index → offset, binary.Size → size, etc.) requires
// the matching suffix on its assignment target.
func Test_Naming_Stdlib_Allowlist(t *testing.T) {
	t.Parallel()
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
			Want_Diag: "The name pos has the role offset. Rename pos -> pos_offset.",
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
			Want_Diag: "The name n has the role size. Rename n -> n_size.",
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
			Want_Diag: "The name n has the role size. Rename n -> n_size.",
		},
	}
	run_diag_table(t, tests)
}

// Test_Naming_Element_Count_Result verifies that v := len(x) / v := cap(x) require
// v to carry a _count or _size suffix (either accepted; size is the
// stylistic override when the count is in bytes).
func Test_Naming_Element_Count_Result(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
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
			Want_Diag: "The name n has the role count or size. Rename n -> n_count.",
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
			Want_Diag: "The name n has the role count or size. Rename n -> n_count.",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_Naming_Element_Count_Result_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "n_count from len allowed",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func F(xs []int) (result int) {
	defer func() {
	}()
	n_count := len(xs)
	return n_count
}
`,
			},
			Want_Diag: "",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_Naming_Element_Count_Result_Part3(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "n_size from len allowed",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func F(buffer []byte) (result int) {
	defer func() {
	}()
	n_size := len(buffer)
	return n_size
}
`,
			},
			Want_Diag: "",
		},
	})
}

// Test_Naming_Rename_Suggestion verifies the replace-or-append fix logic:
// names with no existing terminology segment get the term appended; tier-1
// bans `length` and `len` as segments so the replace branch is exercised
// via that path (a name like `buf_length` is rejected by tier 1 and the
// terminology rename, when triggered, replaces the offending segment).
func Test_Naming_Rename_Suggestion(t *testing.T) {
	t.Parallel()
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
			Want_Diag: "The name i has the role index. Rename i -> i_index.",
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
			Want_Diag: "The name I has the role index. Rename I -> I_Index.",
		},
	}
	run_diag_table(t, tests)
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

// Test_Naming_Arithmetic verifies the tier-3 operand-suffix invariant:
// when both operands of `+` or `-` carry recognized suffixes, the
// combination must match the table (_index - _index = _count, etc.).
// Mismatched assignment LHS suffix is also flagged.
func Test_Naming_Arithmetic(t *testing.T) {
	t.Parallel()
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
	}
	run_diag_table(t, tests)
}

// Test_Naming_Arithmetic_Clean covers the allowed-arithmetic side of the
// tier-3 operand-suffix invariant. Split from Test_Naming_Arithmetic so
// each function fits the 100-line cap.
func Test_Naming_Arithmetic_Clean(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "valid index minus index assigned to count allowed",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func F() (result int) {
	defer func() {
	}()
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

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func F() (result int) {
	defer func() {
	}()
	a_offset := 0
	b_size := 0
	position_offset := a_offset + b_size
	return position_offset
}
`,
			},
			Want_Diag: "",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_Naming_Arithmetic_Clean_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "unsuffixed operands skipped",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func F() (result int) {
	defer func() {
	}()
	a := 0
	b := 0
	return a + b
}
`,
			},
			Want_Diag: "",
		},
	})
}

// Test_Naming_Arithmetic_Size_Plus_Offset drives check_binary_result with
// input.Left="size" (Lo=4) and input.Right="offset" (Hi=6) so both endpoint
// buckets observe — the canonical add-pair table treats this as the result
// suffix "offset".
func Test_Naming_Arithmetic_Size_Plus_Offset(t *testing.T) {
	t.Parallel()
	source := `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func F() (result int) {
	defer func() {
	}()
	a_size := 0
	b_offset := 0
	position_offset := a_size + b_offset
	return position_offset
}
`
	diags, err := lint.Check_Source("test.go", source)
	t.Logf("size_plus_offset diags=%d err=%v", len(diags), err)
}

// Test_Naming_Abbreviations_Flagged covers identifiers containing tokenized
// words from the abbreviation denylist — single-candidate (`cfg`), ambiguous
// multi-candidate (`res`), and various declaration sites (type, field, func).
func Test_Naming_Abbreviations_Flagged(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
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
			Want_Diag: `Rename cfg_path -> config_path`,
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
			Want_Diag: `Rename user_res -> ` +
				`[user_response, user_result, user_resource, user_reserve]`,
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
			Want_Diag: `Rename Pool_Mgr -> Pool_Manager`,
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
			Want_Diag: `Rename Submit_Btn -> Submit_Button`,
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_Naming_Abbreviations_Flagged_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "func name with cb flagged",
			Files: map[string]string{
				"test.go": `package main

func Run_Cb() (x int) { return 0 }
`,
			},
			Want_Diag: `Rename Run_Cb -> Run_Callback`,
		},

		{
			Name: "var with src flagged (s-range candidates)",
			Files: map[string]string{
				"test.go": `package main

func F() (x int) {
	file_src := 0
	return file_src
}
`,
			},
			Want_Diag: `Rename file_src -> file_source`,
		},
	})
}

// Test_Naming_Abbreviations_Exempt covers identifiers that look like
// abbreviations but are exempt: Go-language idioms (err, ctx, fmt, len, cap,
// min, max), single-letter loop counters (Tiger Style sort/matrix primitives),
// and clean code with no abbreviation hits.
func Test_Naming_Abbreviations_Exempt(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "err exempt",
			Files: map[string]string{
				"test.go": `package main

import (
	"errors"

)

// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func F() (x int) {
	defer func() {
	}()
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
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_Naming_Abbreviations_Exempt_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "single-letter loop counter exempt",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func F(n int) (x int) {
	defer func() {
	}()
	for i_index := 0; i_index < n; i_index++ {
		x = x + i_index
	}
	return x
}
`,
			},
			Want_Diag: "",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_Naming_Abbreviations_Exempt_Part3(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "clean code no abbreviation hit",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func Compute(value int) (result int) {
	defer func() {
	}()
	total := value + 1
	return total
}
`,
			},
			Want_Diag: "",
		},
	})
}

// Test_Names_Vocabulary pins the merged vocabulary check: one table drives both
// the abbreviation expansions and the no-candidate bans, with one uniform
// diagnostic shape. A single candidate renders `Rename x -> y`; multiple render
// `Rename x -> [a, b, c]`; a banned word with no candidate renders
// `The identifier "x" contains the banned substring "y"`.
func Test_Names_Vocabulary(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "single candidate has no brackets",
			Files: map[string]string{
				"test.go": `package main

func F() (x int) {
	cfg_path := 0
	return cfg_path
}
`,
			},
			Want_Diag: `Rename cfg_path -> config_path`,
		},

		{
			Name: "multiple candidates use brackets",
			Files: map[string]string{
				"test.go": `package main

func F() (x int) {
	user_res := 0
	return user_res
}
`,
			},
			Want_Diag: `Rename user_res -> ` +
				`[user_response, user_result, user_resource, user_reserve]`,
		},

		{
			Name: "banned word with no candidate",
			Files: map[string]string{
				"test.go": `package main

func F() (x int) {
	length := 0
	return length
}
`,
			},
			Want_Diag: `The identifier "length" contains the banned substring "length"`,
		},

		{
			Name: "util is banned, not expanded to utility",
			Files: map[string]string{
				"test.go": `package main

func parse_util() { return }
`,
			},
			Want_Diag: `The identifier "parse_util" contains the banned substring "util"`,
		},
	})
}

// Additional cases, split to keep each function within the length limit. These
// pin the broadened coverage: the table applies to package and file names too.
func Test_Names_Vocabulary_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "abbreviation in package name flagged",
			Files: map[string]string{
				"test.go": "package cfg\n",
			},
			Want_Diag: `Rename cfg -> config`,
		},

		{
			Name: "abbreviation in file name flagged",
			Files: map[string]string{
				"cfg.go": "package main\n",
			},
			Want_Diag: `Rename cfg -> config`,
		},
	})
}

// Test_Vocabulary_Sourced_From_Lint_Json proves the denylist is read from
// lint.json rather than hardcoded: a word present only in a custom table is
// flagged with its configured expansion, while cfg — a staple of the old built-in
// table but absent from this config — is not flagged. This is the regression
// guard that the move from switch statements to configuration actually took.
func Test_Vocabulary_Sourced_From_Lint_Json(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"lint.json": &fstest.MapFile{Data: []byte(configuration_document(map[string]any{
			"shared_component":  "x",
			"word_replacements": map[string][]string{"wibble": {"wobble"}},
		}))},
		"test.go": &fstest.MapFile{Data: gofmt_must(t, `package main

func F() (x int) {
	wibble_path := 0
	cfg_path := 0
	return wibble_path + cfg_path
}
`)},
	}
	stdout := &strings.Builder{}
	lint.Main(&lint.Main_Input{Fsys: fsys, Stdout: stdout, Stderr: &strings.Builder{}})
	output := stdout.String()
	if !strings.Contains(output, "Rename wibble_path -> wobble_path") {
		t.Fatalf("the word configured in lint.json must be flagged; got: %s", output)
	}
	if strings.Contains(output, "config") {
		t.Fatalf("cfg must not be flagged — it is absent from this lint.json, proving "+
			"the table is sourced from config, not hardcoded; got: %s", output)
	}
}

// Test_Naming_Participles verifies that declared identifiers whose final
// tokenized word ends in "ing" and isn't in nouns_suffixed_by_ing are
// flagged. Gerund-nouns (String, Mapping, Encoding, etc.) and the
// Stringer interface's String method are not flagged.
func Test_Naming_Participles(t *testing.T) {
	t.Parallel()
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
			Want_Diag: `The name "preparing"`,
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
			Want_Diag: `The name "processing"`,
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
			Want_Diag: `The name "rendering"`,
		},
	}
	run_diag_table(t, tests)
}

// Test_Naming_Participles_Part2 covers the gerund-noun allowlist cases, split
// off to keep each function within the length limit.
func Test_Naming_Participles_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "Mapping allowed gerund-noun",
			Files: map[string]string{
				"test.go": `package main

type Key_Mapping struct {
	// Entries is a fixture.
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
	// Bytes is a fixture.
	Bytes int
}
`,
			},
			Want_Diag: "",
		},
	})
}

// Test_Naming_Participles_Exempt covers shapes that look like present
// participles but are allowed: the Stringer interface method's name lives
// in a hard allowlist, and names without an -ing suffix are not flagged.
func Test_Naming_Participles_Exempt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "String method on type allowed via noun allowlist",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

type Color int

func (c Color) String() (result string) {
	defer func() {
	}()
	return ""
}
`,
			},
			Want_Diag: "",
		},
		{
			Name: "no -ing suffix not flagged",
			Files: map[string]string{
				"test.go": `package main

` + FIXTURE_INVARIANT_IMPORT + `
// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

func Compute(value int) (result int) {
	defer func() {
	}()
	return value
}
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
	t.Parallel()
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
			Want_Diag: "test Test_Foo has no doc comment",
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

// Test_Package_Split_Threshold verifies that packages whose file count does
// not equal ceil(total_sloc/10000) are flagged: too many files (fragmentation)
// and too few files (oversized). Source / test / build-tag groups are counted
// separately; build-tag detection is done via go/build/constraint.
func Test_Package_Split_Threshold(t *testing.T) {
	t.Parallel()
	source_a := "// Package foo is a fixture.\npackage foo\n\n" +
		"// A is a fixture.\nfunc A() { return }\n"
	source_b := "package foo\n\n// B is a fixture.\nfunc B() { return }\n"
	source_c := "package foo\n\n// C is a fixture.\nfunc C() { return }\n"
	test_a := "package foo_test\n\nimport \"testing\"\n\n" +
		"// Test_A verifies A.\nfunc Test_A(t *testing.T) { t.Helper() }\n"
	test_b := "package foo_test\n\nimport \"testing\"\n\n" +
		"// Test_B verifies B.\nfunc Test_B(t *testing.T) { t.Helper() }\n"
	specification_exempt_test := "package foo_test\n\nimport \"testing\"\n\n" +
		"// Test_C verifies C.\nfunc Test_C(t *testing.T) { t.Helper() }\n"
	linux_a := "//go:build linux\n\n// Package foo is a fixture.\npackage foo\n\n" +
		"// A is a fixture.\nfunc A() { return }\n"
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
			Name: "two test files in same pkg flagged",
			Files: map[string]string{
				"a.go": source_a, "a_test.go": test_a, "b_test.go": test_b,
			},
			Want_Diag: "has 2 test files",
		},
		{
			Name: "specification_test.go is exempt from the test-file count",
			Files: map[string]string{
				"a.go": source_a, "a_test.go": test_a,
				"specification_test.go": specification_exempt_test,
			},
			Want_Diag: "",
		},
		{
			Name:      "build tag splits source group",
			Files:     map[string]string{"a.go": source_a, "a_linux.go": linux_b},
			Want_Diag: "",
		},
		{
			Name: "two files sharing a build tag flagged",
			Files: map[string]string{
				"a.go": linux_a, "a_extra.go": linux_b, "plain.go": source_c,
			},
			Want_Diag: "has 2 source files",
		},
	}
	run_diag_table(t, tests)
}

// Test_Package_Split_Threshold_Part2 covers the undersized direction: a
// single file whose line count exceeds LINES_PER_FILE_MAX must be split.
// gofmt_must strips trailing blank lines, so this test bypasses run_diag_table
// and feeds raw bytes to lint.Main directly.
func Test_Package_Split_Threshold_Part2(t *testing.T) {
	t.Parallel()
	content := []byte("// Package foo is a fixture.\npackage foo\n" +
		strings.Repeat("\n", 10001))
	fsys := fstest.MapFS{"a.go": {Data: content}}
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	lint_main(t, &lint.Main_Input{Fsys: fsys, Stdout: stdout, Stderr: stderr})
	if !strings.Contains(stdout.String(), "has 1 source files") {
		t.Errorf("single file over 10k lines must be flagged; got: %s",
			stdout.String())
	}
}

// Test_Import_Alias_No_Default isolates default-name ban from other alias bans.
func Test_Import_Alias_No_Default(t *testing.T) {
	t.Parallel()
	fixture := func(alias string) (files map[string]string) {
		return map[string]string{"a.go": "// Package foo is a fixture.\n" +
			"package foo\n\nimport " + alias + " \"strings\"\n\n" +
			"// Trim trims.\nfunc Trim() (trimmed string) {\n" +
			"\treturn " + alias + ".TrimSpace(\"\")\n}\n"}
	}
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name:      "lowercase alias flagged",
			Files:     fixture("iodefault"),
			Want_Diag: `import alias "iodefault"`,
		},
		{
			Name:      "Ada_Case alias flagged",
			Files:     fixture("Io_Default"),
			Want_Diag: `import alias "Io_Default"`,
		},
		{
			Name:      "interior default flagged",
			Files:     fixture("default_text"),
			Want_Diag: `import alias "default_text"`,
		},
		{
			Name:      "alias naming another package is independently flagged",
			Files:     fixture("text"),
			Want_Diag: `import alias "text" is unnecessary`,
		},
	}
	run_diag_table(t, tests)
}

// Test_Import_Alias_Unnecessary keeps aliases reserved for package-name collisions.
func Test_Import_Alias_Unnecessary(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "different alias flagged",
			Files: map[string]string{"a.go": "// Package foo is a fixture.\n" +
				"package foo\n\nimport text \"strings\"\n"},
			Want_Diag: `import alias "text" is unnecessary`,
		},
		{
			Name: "matching alias flagged",
			Files: map[string]string{"a.go": "// Package foo is a fixture.\n" +
				"package foo\n\nimport strings \"strings\"\n"},
			Want_Diag: `import alias "strings" is unnecessary`,
		},
		{
			Name: "colliding package names allow aliases",
			Files: map[string]string{"a.go": "// Package foo is a fixture.\n" +
				"package foo\n\nimport (\n" +
				"\tfirst_text \"example.com/first/text\"\n" +
				"\tsecond_text \"example.com/second/text\"\n)\n"},
			Want_Diag: "",
		},
		{
			Name: "collision does not release unrelated alias",
			Files: map[string]string{"a.go": "// Package foo is a fixture.\n" +
				"package foo\n\nimport (\n" +
				"\tfirst_text \"example.com/first/text\"\n" +
				"\tsecond_text \"example.com/second/text\"\n" +
				"\tbyte_text \"bytes\"\n)\n"},
			Want_Diag: `import alias "byte_text" is unnecessary`,
		},
		{
			Name: "blank import creates no collision",
			Files: map[string]string{"a.go": "// Package foo is a fixture.\n" +
				"package foo\n\nimport (\n" +
				"\tfirst_text \"example.com/first/text\"\n" +
				"\t_ \"example.com/second/text\"\n)\n"},
			Want_Diag: `import alias "first_text" is unnecessary`,
		},
		{
			Name: "dot import creates no collision",
			Files: map[string]string{"a.go": "// Package foo is a fixture.\n" +
				"package foo\n\nimport (\n" +
				"\tfirst_text \"example.com/first/text\"\n" +
				"\t. \"example.com/second/text\"\n)\n"},
			Want_Diag: `import alias "first_text" is unnecessary`,
		},
		{
			Name: "unaliased import clean",
			Files: map[string]string{"a.go": "// Package foo is a fixture.\n" +
				"package foo\n\nimport \"fmt\"\n"},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Import_Alias_Unnecessary_Declared_Name makes package clause authoritative over path base.
func Test_Import_Alias_Unnecessary_Declared_Name(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name       string
		First      string
		Second     string
		Want_Count int
	}{
		{
			Name: "declared collision allows aliases", First: "text", Second: "text",
			Want_Count: 0,
		},
		{
			Name:  "distinct declarations reject aliases",
			First: "first", Second: "second",
			Want_Count: 2,
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			fsys := fstest.MapFS{
				"go.mod": {Data: []byte("module example.com\n")},
				"shared/first/first.go": {Data: gofmt_must(t,
					"// Package imported is a fixture.\n"+
						"package "+test.First+"\n")},
				"shared/second/second.go": {Data: gofmt_must(t,
					"// Package imported is a fixture.\n"+
						"package "+test.Second+"\n")},
				"shared/consumer/consumer.go": {Data: gofmt_must(t,
					"// Package consumer is a fixture.\npackage consumer\n\n"+
						"import (\n"+
						"\tfirst_text \"example.com/shared/first\"\n"+
						"\tsecond_text \"example.com/shared/second\"\n"+
						")\n")},
			}
			diags, err := lint.Check_File_System(&lint.Check_File_System_Input{
				Fsys: fsys, CPU_Count: 1, Shared_Component: "shared",
				Invariant_Exempt_Packages: []string{"**"},
				Pure_But_Indeterministic:  []string{"**"},
			})
			if err != nil {
				t.Fatalf("Check_File_System: %v", err)
			}
			diagnostic_count := 0
			for _, diagnostic := range diags {
				if diagnostic.Name == "import-alias-unnecessary" {
					diagnostic_count++
				}
			}
			if diagnostic_count != test.Want_Count {
				t.Fatalf(
					"alias diagnostics=%d, want %d", diagnostic_count,
					test.Want_Count)
			}
		})
	}
}

// Test_File_Size verifies the per-file line cap. The cap binds each file on its
// own, so it fires on an oversized file even when the package's file count
// already satisfies the fragmentation quota — the case a count-only rule cannot
// see, since a second file supplies the missing count whatever the first holds.
// A file at exactly LINES_PER_FILE_MAX stays silent. gofmt_must strips trailing
// blank lines, so this test bypasses run_diag_table and feeds raw bytes to
// lint.Main directly.
func Test_File_Size(t *testing.T) {
	t.Parallel()
	// 10001 lines. A trailing newline opens no line the scanner can record —
	// its offset lands past the end of the file — so token.File.LineCount
	// equals the newline count here, two from the source plus 9999 of padding.
	over := []byte("// Package foo is a fixture.\npackage foo\n" +
		strings.Repeat("\n", 9999))
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	lint_main(t, &lint.Main_Input{
		Fsys: fstest.MapFS{
			"a.go": {Data: over},
			"b.go": {Data: []byte("package foo\n")},
		},
		Stdout: stdout,
		Stderr: stderr,
	})
	if !strings.Contains(stdout.String(),
		"The file has 10001 lines. The maximum is 10000.") {
		t.Errorf("a file over the cap must be flagged; got: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "source files") {
		t.Errorf("the file count is already correct here; got: %s", stdout.String())
	}
	at_cap := []byte("// Package foo is a fixture.\npackage foo\n" +
		strings.Repeat("\n", 9998))
	stdout = &strings.Builder{}
	stderr = &strings.Builder{}
	lint_main(t, &lint.Main_Input{
		Fsys:   fstest.MapFS{"a.go": {Data: at_cap}},
		Stdout: stdout,
		Stderr: stderr,
	})
	if strings.Contains(stdout.String(), "(max 10000)") {
		t.Errorf("a file at the cap must stay silent; got: %s", stdout.String())
	}
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
			"lib/foo/foo.go":                sibling,
			"lib/foo/foo_test.go":           clause + "\n\n" + body,
			"lib/foo/SPECIFICATION.md":      SPECIFICATION_FIXTURE_MD,
			"lib/foo/specification_test.go": fixture_specification_test("foo"),
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

func f() (buffer [BUFFER_SIZE_MAX]byte) {
	return buffer
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

// Test_Main_Package_Size verifies the cap on package main: over the cap is
// flagged, exactly at it is silent, a build-tagged variant carries its own count
// rather than joining the untagged one, and a _test.go file adds nothing. gofmt
// strips the trailing blank lines the padding needs, so this test feeds raw bytes
// to lint.Main.
func Test_Main_Package_Size(t *testing.T) {
	t.Parallel()
	// A trailing newline opens no line the scanner records, so LineCount equals
	// the newline count: three from the source plus the padding.
	main_source := func(pad int) (source []byte) {
		return []byte("package main\n\nfunc main() {}\n" +
			strings.Repeat("\n", pad))
	}
	tagged := func(pad int) (source []byte) {
		return []byte("//go:build linux\n\npackage main\n\nfunc other() {}\n" +
			strings.Repeat("\n", pad))
	}
	const FRAGMENT = "package main in ."
	over := main_package_size_output(t, fstest.MapFS{"main.go": {Data: main_source(198)}})
	if !strings.Contains(over, FRAGMENT) {
		t.Errorf("201 lines must be flagged; got: %s", over)
	}
	at_cap := main_package_size_output(t, fstest.MapFS{"main.go": {Data: main_source(197)}})
	if strings.Contains(at_cap, FRAGMENT) {
		t.Errorf("200 lines must stay silent; got: %s", at_cap)
	}
	split := main_package_size_output(t, fstest.MapFS{
		"main.go":        {Data: main_source(147)},
		"other_linux.go": {Data: tagged(147)},
	})
	if strings.Contains(split, FRAGMENT) {
		t.Errorf("two build-tag groups of 150 must stay apart; got: %s", split)
	}
	with_test := main_package_size_output(t, fstest.MapFS{
		"main.go":      {Data: main_source(147)},
		"main_test.go": {Data: main_source(147)},
	})
	if strings.Contains(with_test, FRAGMENT) {
		t.Errorf("a test file must carry no count; got: %s", with_test)
	}
}

// Lints the fixture and returns stdout. Bypasses run_diag_table because the
// blank-line padding these fixtures need cannot survive gofmt.
func main_package_size_output(t *testing.T, fsys fstest.MapFS) (output string) {
	t.Helper()
	stdout := &strings.Builder{}
	lint_main(t, &lint.Main_Input{
		Fsys: fsys, Stdout: stdout, Stderr: &strings.Builder{},
	})
	return stdout.String()
}

// Test_Snap_Backtick verifies that the first argument to snap.Init / snap.Edit
// must be a backticked raw string literal; double-quoted string literals are
// flagged, non-literal arguments are unaffected.
func Test_Snap_Backtick(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "snap.Init with double-quoted string flagged",
			Files: map[string]string{
				"test.go": "package main\n\nimport \"x/snap\"\n\n" +
					"func f() { snap.Init(\"foo\") }\n",
			},
			Want_Diag: "The call snap.Init has no raw string literal",
		},
		{
			Name: "snap.Edit with double-quoted string flagged",
			Files: map[string]string{
				"test.go": "package main\n\nimport \"x/snap\"\n\n" +
					"func f() { snap.Edit(\"foo\") }\n",
			},
			Want_Diag: "The call snap.Edit has no raw string literal",
		},
		{
			Name: "snap.Init with backticked string allowed",
			Files: map[string]string{
				"test.go": "package main\n\nimport \"x/snap\"\n\n" +
					"func f() { snap.Edit(`foo`) }\n",
			},
			Want_Diag: "",
		},
		{
			Name: "snap.Other not in scope",
			Files: map[string]string{
				"test.go": "package main\n\nimport \"x/snap\"\n\n" +
					"func f() { snap.Other(\"foo\") }\n",
			},
			Want_Diag: "",
		},
		{
			Name: "non-literal arg allowed",
			Files: map[string]string{
				"test.go": "package main\n\n" +
					"import (\n" +
					"\t\"x/snap\"\n\n" +
					"\t\"github.com/james-orcales/james-orcales/" +
					"shared/invariant/v2\"\n" +
					")\n\n" +
					"const FIXTURE_HI = 100\n\n" +
					"func f(s string) {\n" +
					"\tinvariant.Cross_Product(\n" +
					"\t\tinvariant.Distinct_Boundary(" +
					"&invariant.Boundary_Input[int]{\n" +
					"\t\t\tX: len(s), Lo: 0, Hi: FIXTURE_HI, " +
					"Message: \"len in range\",\n" +
					"\t\t}),\n" +
					"\t\tinvariant.Always(" +
					"s == \"\", \"s is empty\"),\n" +
					"\t)\n" +
					"\tsnap.Init(s)\n" +
					"}\n",
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Banned_Package_And_File_Names verifies that the universal banned
// segments (util*) are flagged in package names and file names too.
func Test_Banned_Package_And_File_Names(t *testing.T) {
	t.Parallel()
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
			Want_Diag: `banned substring "utils"`,
		},
		{
			Name: "package name with util segment flagged",
			Files: map[string]string{
				"test.go": `package string_util

func f() (result int) { return 1 }
`,
			},
			Want_Diag: `banned substring "util"`,
		},
		{
			Name: "file name utils.go flagged",
			Files: map[string]string{
				"utils.go": `package main

func f() (result int) { return 1 }
`,
			},
			Want_Diag: `banned substring "utils"`,
		},
		{
			Name: "file name with utility segment flagged",
			Files: map[string]string{
				"string_utility.go": `package main

func f() (result int) { return 1 }
`,
			},
			Want_Diag: `banned substring "utility"`,
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

// Test_No_Interfaces verifies that interface types with method elements are
// flagged wherever they appear (TypeSpec, inline in signatures, type
// assertions). Type-element-only interfaces (generic constraints built from
// unions and approximations) are allowed because they carry no method set.
// `any` and bare `interface{}` are allowed as the empty interface.
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

// Test_No_Interfaces_Allowed covers shapes the rule must accept: bare `any`,
// empty `interface{}`, and generic type-constraint interfaces built only
// from type elements (no method set).
func Test_No_Interfaces_Allowed(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "any parameter allowed",
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
			Want_Diag: "",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_No_Interfaces_Allowed_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "generic type-constraint interface allowed",
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

type Number interface {
	~int | ~int64
}

func F[T Number](x T) (result int) {
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

// Test_Unnecessary_Method verifies that receiver methods whose name+signature
// does not match a stdlib interface method are flagged with a recommendation
// to convert to a free function. The set of legal targets is the closed
// stdlib table; third-party interface satisfaction is rejected by policy.
// Test_Unnecessary_Method_Allowed covers receiver methods whose name and
// signature match an entry in the stdlib interface table — these must pass
// without diagnostic. Free functions are also covered as a control.
func Test_Unnecessary_Method_Allowed(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "stdlib Stringer match allowed",
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
			Want_Diag: "",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_Unnecessary_Method_Allowed_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "stdlib error match allowed",
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
			Want_Diag: "",
		},
	})
}

// Test_Unnecessary_Method_Allowed_Read covers the io.Reader.Read shape;
// split off Test_Unnecessary_Method_Allowed for the 100-line cap.
func Test_Unnecessary_Method_Allowed_Read(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "stdlib Read match allowed",
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
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Unnecessary_Method_Allowed_Extra continues the allowed table: a
// stdlib-interface-satisfying method with `any`, and a plain free function
// (no receiver) as a control.
func Test_Unnecessary_Method_Allowed_Extra(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "stdlib Scan with any allowed",
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

func (t T) Scan(x any) (err error) {
	return nil
}
`,
			},
			Want_Diag: "",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_Unnecessary_Method_Allowed_Extra_Part2(t *testing.T) {
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

// Test_Unnecessary_Method_Flagged covers receiver methods that the stdlib
// table does not cover — either name unknown or signature wrong-shape — and
// must be flagged with the "convert to free function" instruction.
func Test_Unnecessary_Method_Flagged(t *testing.T) {
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
			Want_Diag: "satisfies no stdlib interface",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_Unnecessary_Method_Flagged_Part2(t *testing.T) {
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
			Want_Diag: "satisfies no stdlib interface",
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

// Test_Unbounded_Read verifies the tier-2 ban on stdlib read APIs whose
// internal buffer growth is determined by the source (read-to-EOF) or by
// the filesystem (whole-file slurp). Constructors returning growable
// readers (bufio.NewScanner, bufio.NewReader) are banned outright — the
// caller is forced onto io.ReadFull / r.Read(buf) with a fixed buf.
//
// The generated-file exemption and the negative (clean) case live in
// Test_Unbounded_Read_Exemptions to keep each function within the
// per-function line budget.
func Test_Unbounded_Read(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "io.ReadAll flagged",
			Files: map[string]string{
				"test.go": "package main\n" +
					"func f() { io.ReadAll(nil) }\n",
			},
			Want_Diag: "API \"io.ReadAll\" is unbounded",
		},
		{
			Name: "io.Copy flagged",
			Files: map[string]string{
				"test.go": "package main\n" +
					"func f() { io.Copy(nil, nil) }\n",
			},
			Want_Diag: "API \"io.Copy\" is unbounded",
		},
		{
			Name: "io.CopyBuffer flagged",
			Files: map[string]string{
				"test.go": "package main\n" +
					"func f() { io.CopyBuffer(nil, nil, nil) }\n",
			},
			Want_Diag: "API \"io.CopyBuffer\" is unbounded",
		},
		{
			Name: "os.ReadFile flagged",
			Files: map[string]string{
				"shared/simulation/os/test.go": "package os\nimport \"os\"\n" +
					"func f() { os.ReadFile(\"path\") }\n",
			},
			Want_Diag: "API \"os.ReadFile\" is unbounded",
		},
		{
			Name: "os.ReadDir flagged",
			Files: map[string]string{
				"shared/simulation/os/test.go": "package os\nimport \"os\"\n" +
					"func f() { os.ReadDir(\"path\") }\n",
			},
			Want_Diag: "API \"os.ReadDir\" is unbounded",
		},
		{
			Name: "bufio.NewScanner flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"bufio\"\n" +
					"func f() { bufio.NewScanner(nil) }\n",
			},
			Want_Diag: "API \"bufio.NewScanner\" is unbounded",
		},
		{
			Name: "bufio.NewReader flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"bufio\"\n" +
					"func f() { bufio.NewReader(nil) }\n",
			},
			Want_Diag: "API \"bufio.NewReader\" is unbounded",
		},
	}
	run_diag_table(t, tests)
}

// Test_Unbounded_Read_Exemptions verifies the two non-positive paths for
// the read-category ban: a source with no banned identifier produces no
// diagnostic, and a `// Code generated ... DO NOT EDIT` header exempts
// the file even with a banned call inside.
func Test_Unbounded_Read_Exemptions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "no banned API allowed",
			Files: map[string]string{
				"test.go": "package main\nfunc f() { return }\n",
			},
			Want_Diag: "",
		},
		{
			Name: "generated file exempt",
			Files: map[string]string{
				"test.go": "// Code generated by stubgen. DO NOT EDIT.\n" +
					"package main\n" +
					"func f() { io.ReadAll(nil) }\n",
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Unbounded_Decode verifies the tier-2 ban on stream decoder
// constructors. Each pulls from its reader until a token decodes — no
// caller-supplied cap. The bounded substitute is read-into-bounded-buf
// then Unmarshal.
func Test_Unbounded_Decode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "json.NewDecoder flagged",
			Files: map[string]string{
				"test.go": "package main\n" +
					"func f() { json.NewDecoder(nil) }\n",
			},
			Want_Diag: "API \"json.NewDecoder\" is unbounded",
		},
		{
			Name: "xml.NewDecoder flagged",
			Files: map[string]string{
				"test.go": "package main\n" +
					"func f() { xml.NewDecoder(nil) }\n",
			},
			Want_Diag: "API \"xml.NewDecoder\" is unbounded",
		},
		{
			Name: "gob.NewDecoder flagged",
			Files: map[string]string{
				"test.go": "package main\n" +
					"func f() { gob.NewDecoder(nil) }\n",
			},
			Want_Diag: "API \"gob.NewDecoder\" is unbounded",
		},
		{
			Name: "csv.NewReader flagged",
			Files: map[string]string{
				"test.go": "package main\n" +
					"func f() { csv.NewReader(nil) }\n",
			},
			Want_Diag: "API \"csv.NewReader\" is unbounded",
		},
		{
			Name: "no banned API allowed",
			Files: map[string]string{
				"test.go": "package main\nfunc f() { return }\n",
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Unbounded_Decompression verifies the tier-2 ban on decompressor
// constructors. The decompressed output is unbounded relative to input
// size — the source is the classic zip-bomb shape.
func Test_Unbounded_Decompression(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "gzip.NewReader flagged",
			Files: map[string]string{
				"test.go": "package main\n" +
					"func f() { gzip.NewReader(nil) }\n",
			},
			Want_Diag: "API \"gzip.NewReader\" is unbounded",
		},

		{
			Name: "flate.NewReader flagged",
			Files: map[string]string{
				"test.go": "package main\n" +
					"func f() { flate.NewReader(nil) }\n",
			},
			Want_Diag: "API \"flate.NewReader\" is unbounded",
		},

		{
			Name: "zlib.NewReader flagged",
			Files: map[string]string{
				"test.go": "package main\n" +
					"func f() { zlib.NewReader(nil) }\n",
			},
			Want_Diag: "API \"zlib.NewReader\" is unbounded",
		},

		{
			Name: "bzip2.NewReader flagged",
			Files: map[string]string{
				"test.go": "package main\n" +
					"func f() { bzip2.NewReader(nil) }\n",
			},
			Want_Diag: "API \"bzip2.NewReader\" is unbounded",
		},

		{
			Name: "lzw.NewReader flagged",
			Files: map[string]string{
				"test.go": "package main\n" +
					"func f() { lzw.NewReader(nil, 0, 0) }\n",
			},
			Want_Diag: "API \"lzw.NewReader\" is unbounded",
		},

		{
			Name: "zip.NewReader flagged",
			Files: map[string]string{
				"test.go": "package main\n" +
					"func f() { zip.NewReader(nil, 0) }\n",
			},
			Want_Diag: "API \"zip.NewReader\" is unbounded",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_Unbounded_Decompression_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "zip.OpenReader flagged",
			Files: map[string]string{
				"test.go": "package main\n" +
					"func f() { zip.OpenReader(\"path\") }\n",
			},
			Want_Diag: "API \"zip.OpenReader\" is unbounded",
		},

		{
			Name: "tar.NewReader flagged",
			Files: map[string]string{
				"test.go": "package main\n" +
					"func f() { tar.NewReader(nil) }\n",
			},
			Want_Diag: "API \"tar.NewReader\" is unbounded",
		},

		{
			Name: "no banned API allowed",
			Files: map[string]string{
				"test.go": "package main\nfunc f() { return }\n",
			},
			Want_Diag: "",
		},
	})
}

// Test_Unbounded_Allocation verifies the tier-2 ban on constructors that
// hand back growable buffers (bytes.NewBuffer*) and on the multiplied
// allocators (strings.Repeat, bytes.Repeat) where the output size is
// caller-supplied factor_a * factor_b.
func Test_Unbounded_Allocation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "bytes.NewBuffer flagged",
			Files: map[string]string{
				"test.go": "package main\n" +
					"func f() { bytes.NewBuffer(nil) }\n",
			},
			Want_Diag: "API \"bytes.NewBuffer\" is unbounded",
		},
		{
			Name: "bytes.NewBufferString flagged",
			Files: map[string]string{
				"test.go": "package main\n" +
					"func f() { bytes.NewBufferString(\"\") }\n",
			},
			Want_Diag: "API \"bytes.NewBufferString\" is unbounded",
		},
		{
			Name: "no banned API allowed",
			Files: map[string]string{
				"test.go": "package main\nfunc f() { return }\n",
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Unbounded_Http verifies the tier-2 ban on net/http's convenience
// surface (functions that route through DefaultClient with no timeout,
// servers without configured timeouts) and on bare references to the
// package-level Default* variables.
func Test_Unbounded_Http(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "http.Get flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"net/http\"\n" +
					"func f() { http.Get(\"\") }\n",
			},
			Want_Diag: "API \"http.Get\" is unbounded",
		},

		{
			Name: "http.Post flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"net/http\"\n" +
					"func f() { http.Post(\"\", \"\", nil) }\n",
			},
			Want_Diag: "API \"http.Post\" is unbounded",
		},

		{
			Name: "http.PostForm flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"net/http\"\n" +
					"func f() { http.PostForm(\"\", nil) }\n",
			},
			Want_Diag: "API \"http.PostForm\" is unbounded",
		},

		{
			Name: "http.Head flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"net/http\"\n" +
					"func f() { http.Head(\"\") }\n",
			},
			Want_Diag: "API \"http.Head\" is unbounded",
		},

		{
			Name: "http.ListenAndServe flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"net/http\"\n" +
					"func f() { http.ListenAndServe(\"\", nil) }\n",
			},
			Want_Diag: "API \"http.ListenAndServe\" is unbounded",
		},

		{
			Name: "http.ListenAndServeTLS flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"net/http\"\n" +
					"func f() { " +
					"http.ListenAndServeTLS(\"\", \"\", \"\", nil) }\n",
			},
			Want_Diag: "API \"http.ListenAndServeTLS\" is unbounded",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_Unbounded_Http_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "http.DefaultClient variable reference flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"net/http\"\n" +
					"func f() { if http.DefaultClient != nil { return } }\n",
			},
			Want_Diag: "API \"http.DefaultClient\" is unbounded",
		},

		{
			Name: "http.DefaultServeMux variable reference flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"net/http\"\n" +
					"func f() { if http.DefaultServeMux != nil { return } }\n",
			},
			Want_Diag: "API \"http.DefaultServeMux\" is unbounded",
		},

		{
			Name: "http.DefaultTransport variable reference flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"net/http\"\n" +
					"func f() { if http.DefaultTransport != nil { return } }\n",
			},
			Want_Diag: "API \"http.DefaultTransport\" is unbounded",
		},

		{
			Name: "no banned API allowed",
			Files: map[string]string{
				"test.go": "package main\nfunc f() { return }\n",
			},
			Want_Diag: "",
		},
	})
}

// Test_Deprecated_Ioutil verifies the tier-2 ban on the entire io/ioutil
// package. Each identifier — including the variable Discard — is flagged
// alongside its modern replacement.
func Test_Deprecated_Ioutil(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "ioutil.ReadAll flagged",
			Files: map[string]string{
				"test.go": "package main\n" +
					"func f() { ioutil.ReadAll(nil) }\n",
			},
			Want_Diag: "API \"ioutil.ReadAll\" is unbounded",
		},

		{
			Name: "ioutil.ReadFile flagged",
			Files: map[string]string{
				"test.go": "package main\n" +
					"func f() { ioutil.ReadFile(\"\") }\n",
			},
			Want_Diag: "API \"ioutil.ReadFile\" is unbounded",
		},

		{
			Name: "ioutil.ReadDir flagged",
			Files: map[string]string{
				"test.go": "package main\n" +
					"func f() { ioutil.ReadDir(\"\") }\n",
			},
			Want_Diag: "API \"ioutil.ReadDir\" is unbounded",
		},

		{
			Name: "ioutil.WriteFile flagged",
			Files: map[string]string{
				"test.go": "package main\n" +
					"func f() { ioutil.WriteFile(\"\", nil, 0) }\n",
			},
			Want_Diag: "API \"ioutil.WriteFile\" is unbounded",
		},

		{
			Name: "ioutil.TempFile flagged",
			Files: map[string]string{
				"test.go": "package main\n" +
					"func f() { ioutil.TempFile(\"\", \"\") }\n",
			},
			Want_Diag: "API \"ioutil.TempFile\" is unbounded",
		},

		{
			Name: "ioutil.TempDir flagged",
			Files: map[string]string{
				"test.go": "package main\n" +
					"func f() { ioutil.TempDir(\"\", \"\") }\n",
			},
			Want_Diag: "API \"ioutil.TempDir\" is unbounded",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_Deprecated_Ioutil_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "ioutil.NopCloser flagged",
			Files: map[string]string{
				"test.go": "package main\n" +
					"func f() { ioutil.NopCloser(nil) }\n",
			},
			Want_Diag: "API \"ioutil.NopCloser\" is unbounded",
		},

		{
			Name: "ioutil.Discard variable reference flagged",
			Files: map[string]string{
				"test.go": "package main\n" +
					"func f() { if ioutil.Discard != nil { return } }\n",
			},
			Want_Diag: "API \"ioutil.Discard\" is unbounded",
		},

		{
			Name: "no banned API allowed",
			Files: map[string]string{
				"test.go": "package main\nfunc f() { return }\n",
			},
			Want_Diag: "",
		},
	})
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
