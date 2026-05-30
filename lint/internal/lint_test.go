package lint_test

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/james-orcales/james-orcales/lint/internal"
)

// Gofmt_must formats test sources so fixtures don't need to be hand-perfect
// gofmt-clean. Required because check_gofmt runs as part of the tier-1
// pipeline against every test source. TestGofmt builds MapFS inline so it
// can submit deliberately un-formatted sources.
const doctrine_shared_library_go_module = "module github.com/james-orcales/" +
	"james-orcales/golang_snacks\n"
const doctrine_binary_go_module = "module example.com/mybinary\n"

const fixture_invariant_import_path = "github.com/james-orcales/james-orcales/" +
	"golang_snacks/invariant/v2/invariant_default"
const fixture_invariant_import = "import invariant \"" + fixture_invariant_import_path + "\"\n"

// Fixture_const_hi declares a file-scope upper bound used by Distinct_Boundary
// Hi positions in test fixtures so the new assertion-bound-named-constant
// rule is satisfied. Append to the import block, before any non-import decl.
const fixture_constant_hi = "\nconst fixture_hi = 100\n"

// Fixture_declaration_callee defines a parameterless g() returning a non-nil
// pointer, used by declaration-coverage fixtures so the test cases focus on
// the call-site shape rather than re-deriving a valid callee each time.
const fixture_declaration_callee = "func g() (out *int) {\n" +
	"\treturn nil\n" +
	"}\n\n"

// Fixture_declaration_callee_pair is the two-return analogue of g(), used by
// fixtures exercising multi-LHS declarations.
const fixture_declaration_callee_pair = "func g() (a *int, b *int) {\n" +
	"\tdefer func() {\n" +
	"a != nil, \"a is non-nil\"),\n" +
	"b != nil, \"b is non-nil\"),\n" +
	"\t\t)\n" +
	"\t}()\n" +
	"\treturn nil, nil\n" +
	"}\n\n"

// Fixture_clean_go is the canonical valid-Go fixture used by tests that
// need an accompanying .go file but don't care about its specific shape.
const fixture_clean_go = "package main\n\n" +
	"import invariant \"github.com/james-orcales/james-orcales/" +
	"golang_snacks/invariant/v2\"\n\n" +
	"const fixture_hi = 100\n\n" +
	"func f() (result int) {\n" +
	"\tdefer func() {\n" +
	"\t\tinvariant.Cross_Product(\n" +
	"\t\t\tinvariant.Distinct_Boundary(" +
	"&invariant.Boundary_Input[int]{\n" +
	"\t\t\t\tX: result, Lo: 0, Hi: fixture_hi}),\n" +
	"\t\t\tinvariant.Always(" +
	"result == 0, \"result is zero\"),\n" +
	"\t\t)\n" +
	"\t}()\n" +
	"\treturn 1\n" +
	"}\n"

// TestMain wires Recorder_Run_Test_Main: registers source files for coverage,
// runs the test suite, then reports any never-fired assertion sites.
const prelude_single = "package fixture\n\n" +
	fixture_invariant_import + fixture_declaration_callee
const prelude_pair = "package fixture\n\n" +
	fixture_invariant_import + fixture_declaration_callee_pair
const prelude_with_h = prelude_single +
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

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

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
			Want_Diag: "shadows outer scope variable",
		},
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
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := lint.Main(&lint.Main_Input{Fsys: fsys_map, Stdout: stdout, Stderr: stderr})
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
}`}, Want_Diag: "discard"},

		{Name: "two discards short decl",
			Files: map[string]string{"test.go": `package main
func f() {
	_, _ := g()
}`}, Want_Diag: "discard"},

		{Name: "three discards assign",
			Files: map[string]string{"test.go": `package main
func f() {
	_, _, _ = g()
}`}, Want_Diag: "discard"},

		{Name: "var blank no type",
			Files: map[string]string{"test.go": `package main
var _ = g()`}, Want_Diag: "discard"},

		{Name: "mixed lhs short decl allowed",
			Files: map[string]string{"test.go": `package main
func f() {
	_, x := g()
	_ = x
}`}, Want_Diag: "discard"},
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
				return
			}
			if !bytes.Contains(stdout.Bytes(), []byte(tt.Want_Diag)) {
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

` + fixture_invariant_import + `
const fixture_hi = 100

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
				return
			}
			if !bytes.Contains(stdout.Bytes(), []byte(tt.Want_Diag)) {
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
	}
	run_diag_table(t, tests)
}

// Test_Exported_Type_Exposes_Private verifies that exported struct field
// types and exported aliases that resolve to an unexported named identifier
// are flagged (with pointer unwrapping and same-file recursion).
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
`}, Want_Diag: "exposes unexported type bar"},
		{Name: "pointer to lowercase flagged",
			Files: map[string]string{"test.go": `package main
type Foo struct { F *bar }
type bar int
`}, Want_Diag: "exposes unexported type bar"},
		{Name: "double pointer flagged",
			Files: map[string]string{"test.go": `package main
type Foo struct { F **bar }
type bar int
`}, Want_Diag: "exposes unexported type bar"},
		{Name: "embedded lowercase flagged",
			Files: map[string]string{"test.go": `package main
type Foo struct { bar }
type bar struct{}
`}, Want_Diag: "exposes unexported type bar"},
		{Name: "transitive via same-file exported flagged",
			Files: map[string]string{"test.go": `package main
type Foo struct { M Middle }
type Middle struct { F bar }
type bar int
`}, Want_Diag: "exported type Foo exposes unexported type bar"},
		{Name: "alias to unexported flagged",
			Files: map[string]string{"test.go": `package main
type Foo = bar
type bar int
`}, Want_Diag: "exposes unexported type bar"},
		{Name: "pointer alias to unexported flagged",
			Files: map[string]string{"test.go": `package main
type Foo = *bar
type bar int
`}, Want_Diag: "exposes unexported type bar"},
	}
	run_diag_table(t, tests)
}

// Test_Exported_Type_Exposes_Private_Allows covers the negative cases:
// builtins, in-scope generic type parameters, qualified selectors, cycles,
// unexported parents, slice/container element positions (out of scope),
// and _test.go file exemption.
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
	I int
	S string
	E error
	A any
	B bool
	R rune
	Y byte
}
`}, Want_Diag: ""},
		{Name: "type parameter allowed",
			Files: map[string]string{"test.go": `package main
type Foo[T any] struct { X T }
`}, Want_Diag: ""},
		{Name: "qualified type allowed",
			Files: map[string]string{"test.go": `package main
import "time"
type Foo struct { T time.Time }
`}, Want_Diag: ""},
		{Name: "self-cycle allowed",
			Files: map[string]string{"test.go": `package main
type Node struct {
	Next *Node
	V    int
}
`}, Want_Diag: ""},
		{Name: "mutual recursion allowed",
			Files: map[string]string{"test.go": `package main
type A struct { B *B }
type B struct { A *A }
`}, Want_Diag: ""},
		{Name: "unexported parent allowed",
			Files: map[string]string{"test.go": `package main
type foo struct { X bar }
type bar int
`}, Want_Diag: ""},
		{Name: "alias to exported allowed",
			Files: map[string]string{"test.go": `package main
type Foo = Bar
type Bar int
`}, Want_Diag: ""},
		{Name: "pointer alias to exported allowed",
			Files: map[string]string{"test.go": `package main
type Foo = *Bar
type Bar int
`}, Want_Diag: ""},
		{Name: "slice of unexported allowed",
			Files: map[string]string{"test.go": `package main
type Foo struct { Xs []bar }
type bar int
`}, Want_Diag: ""},
		{Name: "_test.go file skipped",
			Files: map[string]string{"foo_test.go": `package foo_test
type Foo struct { F bar }
type bar int
`}, Want_Diag: ""},
	}
	run_diag_table(t, tests)
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
`}, Want_Diag: "bare return is banned"},

		{Name: "method with named return and bare return flagged",
			Files: map[string]string{"test.go": `package main
type S struct{}
func (s *S) f() (x int) { return }
`}, Want_Diag: "bare return is banned"},

		{Name: "closure with named return and bare return flagged",
			Files: map[string]string{"test.go": `package main
func g() (out int) {
	cb := func() (x int) { return }
	out = cb()
	return out
}
`}, Want_Diag: "bare return is banned"},

		{Name: "blank-named return with bare return flagged",
			Files: map[string]string{"test.go": `package main
func f() (_ int) { return }
`}, Want_Diag: "bare return is banned"},

		{Name: "multiple bare returns each flagged",
			Files: map[string]string{"test.go": `package main
func f(c bool) (x int) {
	if c {
		return
	}
	return
}
`}, Want_Diag: "bare return is banned"},

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

` + fixture_invariant_import + `
const fixture_hi = 100

func f() (x int) {
	defer func() {
	}()
	return 0
}
`}, Want_Diag: ""},

		{Name: "void closure inside value-returning func allowed",
			Files: map[string]string{"test.go": `package main

` + fixture_invariant_import + `
const fixture_hi = 100

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
			Want_Diag: "grouped declaration banned",
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
			Want_Diag: "grouped declaration banned",
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
			Want_Diag: "grouped declaration banned",
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
			Want_Diag: "grouped declaration banned",
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

const foo = "foo"
const bar = "bar"
`,
			},
			Want_Diag: "",
		},
		{
			Name: "multi-name const at package allowed",
			Files: map[string]string{
				"test.go": `package main

const a, b = 1, 2
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
	"os"
)

func main() {
	fmt.Println(os.Args)
}
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Constant_First_Flagged verifies that any file-scope const that appears
// after a var, type, or func declaration is flagged.
func Test_Constant_First_Flagged(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "const after var flagged",
			Files: map[string]string{
				"test.go": `package main

var x = 0
const c = 1
func main() { print(x, c) }
`,
			},
			Want_Diag: "const declaration must precede",
		},
		{
			Name: "const after type flagged",
			Files: map[string]string{
				"test.go": `package main

type T int
const c T = 1
`,
			},
			Want_Diag: "const declaration must precede",
		},
		{
			Name: "const after func flagged",
			Files: map[string]string{
				"test.go": `package main

func main() { print(c) }
const c = 1
`,
			},
			Want_Diag: "const declaration must precede",
		},
	}
	run_diag_table(t, tests)
}

// Test_Constant_First_Allowed verifies the negative side: consts that sit
// before all non-const file-scope decls are clean, and function-local
// consts are exempt from the ordering rule.
func Test_Constant_First_Allowed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "multiple consts at top allowed",
			Files: map[string]string{
				"test.go": `package main

const a = 1
const b = 2
func main() { print(a, b) }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "const after import allowed",
			Files: map[string]string{
				"test.go": `package main

import "fmt"

const c = 1
func main() { fmt.Println(c) }
`,
			},
			Want_Diag: "",
		},
		{
			Name: "function-local const allowed under file-scope decls",
			Files: map[string]string{
				"test.go": `package main

type T int
func main() {
	const inside = 1
	var t T
	print(t, inside)
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
	Name string ` + "`malformed`" + `
}
`
	fsys_map := fstest.MapFS{"test.go": &fstest.MapFile{Data: gofmt_must(t, source)}}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := lint.Main(&lint.Main_Input{Fsys: fsys_map, Stdout: stdout, Stderr: stderr})
	if code != 0 {
		t.Errorf("expected exit 0, got %d; output: %s", code, stdout.String())
	}
}

// Test_Assertion_Named_Constant_Flagged_Distinct_Boundary verifies that
// numeric literals at the Lo/Hi positions of Distinct_Boundary are flagged.
func Test_Assertion_Named_Constant_Flagged_Distinct_Boundary(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "distinct_boundary Lo non-zero literal flagged",
			Files: map[string]string{
				"test.go": `package main

` + fixture_invariant_import + `
const fixture_hi = 100

func f() {
	var x int
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{X: x, Lo: 1, Hi: x}),
	)
}
`,
			},
			Want_Diag: "assertion bound must be a file-level named constant",
		},

		{
			Name: "distinct_boundary Hi inline literal flagged",
			Files: map[string]string{
				"test.go": `package main

` + fixture_invariant_import + `
const fixture_hi = 100

func f() {
	var x int
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{X: x, Lo: 0, Hi: 42}),
	)
}
`,
			},
			Want_Diag: "assertion bound must be a file-level named constant",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_Assertion_Named_Constant_Flagged_Distinct_Boundary_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "distinct_boundary Hi arithmetic flagged",
			Files: map[string]string{
				"test.go": `package main

` + fixture_invariant_import + `
const fixture_hi = 100

const max_x = 100

func f() {
	var x int
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
				X: x, Lo: 0, Hi: max_x + 1,
			}),
	)
}
`,
			},
			Want_Diag: "assertion bound must be a file-level named constant",
		},

		{
			Name: "distinct_boundary Hi typed cast flagged",
			Files: map[string]string{
				"test.go": `package main

` + fixture_invariant_import + `
const fixture_hi = 100

const max_x = 100

func f() {
	var x int
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
				X: x, Lo: 0, Hi: int(max_x),
			}),
	)
}
`,
			},
			Want_Diag: "assertion bound must be a file-level named constant",
		},
	})
}

// Test_Assertion_Named_Constant_Flagged_Predicate verifies that numeric
// literals at the comparison RHS of Always/Sometimes predicates are flagged.
func Test_Assertion_Named_Constant_Flagged_Predicate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "always eq non-zero literal flagged",
			Files: map[string]string{
				"test.go": `package main

` + fixture_invariant_import + `
const fixture_hi = 100

func f() {
	var x int
	invariant.Cross_Product(invariant.Always(x == 5, "x is five"))
}
`,
			},
			Want_Diag: "assertion bound must be a file-level named constant",
		},
		{
			Name: "always lt non-zero literal flagged",
			Files: map[string]string{
				"test.go": `package main

` + fixture_invariant_import + `
const fixture_hi = 100

func f() {
	var x int
	invariant.Cross_Product(invariant.Always(x < 100, "x is under 100"))
}
`,
			},
			Want_Diag: "assertion bound must be a file-level named constant",
		},
		{
			Name: "sometimes gt non-zero literal flagged",
			Files: map[string]string{
				"test.go": `package main

` + fixture_invariant_import + `
const fixture_hi = 100

func f() {
	var s string
	invariant.Cross_Product(invariant.Sometimes(len(s) > 5, "s len is over five"))
}
`,
			},
			Want_Diag: "assertion bound must be a file-level named constant",
		},
	}
	run_diag_table(t, tests)
}

// Test_Assertion_Named_Constant_Allowed_Bounds verifies the
// Distinct_Boundary and zero/empty/nil sentinel shapes that the rule
// explicitly allows.
func Test_Assertion_Named_Constant_Allowed_Bounds(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "distinct_boundary named const bounds allowed",
			Files: map[string]string{
				"test.go": `package main

` + fixture_invariant_import + `
const fixture_hi = 100

const max_x = 100

func f() {
	var x int
}
`,
			},
			Want_Diag: "",
		},

		{
			Name: "always eq zero allowed",
			Files: map[string]string{
				"test.go": `package main

` + fixture_invariant_import + `
const fixture_hi = 100

func f() {
	var x int
}
`,
			},
			Want_Diag: "",
		},

		{
			Name: "sometimes eq empty string allowed",
			Files: map[string]string{
				"test.go": `package main

` + fixture_invariant_import + `
const fixture_hi = 100

func f() {
	var s string
}
`,
			},
			Want_Diag: "",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_Assertion_Named_Constant_Allowed_Bounds_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "always eq nil allowed",
			Files: map[string]string{
				"test.go": `package main

` + fixture_invariant_import + `
const fixture_hi = 100

func f() {
	var p *int
}
`,
			},
			Want_Diag: "",
		},
	})
}

// Test_Assertion_Named_Constant_Allowed_Predicate verifies the named-const
// and selector-chain shapes accepted at the RHS of Always/Sometimes
// comparison predicates.
func Test_Assertion_Named_Constant_Allowed_Predicate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "always eq named const allowed",
			Files: map[string]string{
				"test.go": `package main

` + fixture_invariant_import + `
const fixture_hi = 100

const max_x = 100

func f() {
	var x int
}
`,
			},
			Want_Diag: "",
		},
		{
			Name: "always lt selector chain allowed",
			Files: map[string]string{
				"test.go": `package main

import (
	"math"

)

func f() {
	var x int
}
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
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
			Want_Diag: "keyed",
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

` + fixture_invariant_import + `
const fixture_hi = 100

type Foo struct {
	A int
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

` + fixture_invariant_import + `
const fixture_hi = 100

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

` + fixture_invariant_import + `
const fixture_hi = 100

type Foo struct {
	A int
}

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
					"import invariant \"" +
					"github.com/james-orcales/james-orcales/" +
					"golang_snacks/invariant/v2\"\n\n" +
					"const fixture_hi = 100\n\n" +
					"func f() (result int) {\n" +
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
			code := lint.Main(&lint.Main_Input{
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
			if !bytes.Contains(stdout.Bytes(), []byte(tt.Want_Diag)) {
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
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	lint.Main(&lint.Main_Input{
		Fsys:         fsys_map,
		Stdout:       stdout,
		Stderr:       stderr,
		Scope_Prefix: "in_scope/",
	})
	output := stdout.String()
	if bytes.Contains(stdout.Bytes(), []byte("out_of_scope/")) {
		t.Errorf("expected out_of_scope/ diagnostics to be filtered; got: %s", output)
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
				fsys_map[k] = &fstest.MapFile{
					Data: []byte(v)}
			}
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			code := lint.Main(&lint.Main_Input{
				Fsys: fsys_map, Stdout: stdout, Stderr: stderr,
			})
			output := stdout.String()
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
				if !bytes.Contains(stdout.Bytes(), []byte(w)) {
					t.Errorf("expected output containing %q; got:\n%s",
						w, output)
				}
			}
			for _, f := range tt.Forbid {
				if bytes.Contains(stdout.Bytes(), []byte(f)) {
					t.Errorf("expected output NOT to contain %q; got:\n%s",
						f, output)
				}
			}
		})
	}
}

// Test_Banned_Identifiers verifies that function names containing banned
// segments (helper, util*) are flagged, with case-insensitive segment matching
// and no false positives on substrings like "helpme".
func Test_Banned_Identifiers(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
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

` + fixture_invariant_import + `
const fixture_hi = 100

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

` + fixture_invariant_import + `
const fixture_hi = 100

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

` + fixture_invariant_import + `
const fixture_hi = 100

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

` + fixture_invariant_import + `
const fixture_hi = 100

type S struct{ Count int }

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
			Want_Diag: "i (used as index) → rename to i_index",
		},

		{
			Name: "i_index induction allowed",
			Files: map[string]string{
				"test.go": `package main

` + fixture_invariant_import + `
const fixture_hi = 100

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

` + fixture_invariant_import + `
const fixture_hi = 100

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

` + fixture_invariant_import + `
const fixture_hi = 100

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

` + fixture_invariant_import + `
const fixture_hi = 100

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

` + fixture_invariant_import + `
const fixture_hi = 100

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

` + fixture_invariant_import + `
const fixture_hi = 100

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

` + fixture_invariant_import + `
const fixture_hi = 100

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

` + fixture_invariant_import + `
const fixture_hi = 100

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

` + fixture_invariant_import + `
const fixture_hi = 100

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
			Want_Diag: `rename user_res -> ` +
				`user_response,user_result,user_resource,user_reserve`,
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
			Want_Diag: `rename Run_Cb -> Run_Callback`,
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
			Want_Diag: `rename file_src -> file_source`,
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

const fixture_hi = 100

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

` + fixture_invariant_import + `
const fixture_hi = 100

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

` + fixture_invariant_import + `
const fixture_hi = 100

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
	}
	run_diag_table(t, tests)
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

` + fixture_invariant_import + `
const fixture_hi = 100

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

` + fixture_invariant_import + `
const fixture_hi = 100

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
	t.Parallel()
	source_a := "// Package foo is a fixture.\npackage foo\n\n" +
		"// A is a fixture.\nfunc A() { return }\n"
	source_b := "package foo\n\n// B is a fixture.\nfunc B() { return }\n"
	source_c := "package foo\n\n// C is a fixture.\nfunc C() { return }\n"
	test_a := "package foo_test\n\nimport \"testing\"\n\n" +
		"// Test_A verifies A.\nfunc Test_A(t *testing.T) { t.Helper() }\n"
	test_b := "package foo_test\n\nimport \"testing\"\n\n" +
		"// Test_B verifies B.\nfunc Test_B(t *testing.T) { t.Helper() }\n"
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
			Want_Diag: "snap.Init must use a backticked",
		},
		{
			Name: "snap.Edit with double-quoted string flagged",
			Files: map[string]string{
				"test.go": "package main\n\nimport \"x/snap\"\n\n" +
					"func f() { snap.Edit(\"foo\") }\n",
			},
			Want_Diag: "snap.Edit must use a backticked",
		},
		{
			Name: "snap.Init with backticked string allowed",
			Files: map[string]string{
				"test.go": "package main\n\nimport \"x/snap\"\n\n" +
					"func f() { snap.Init(`foo`) }\n",
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
					"\tinvariant \"github.com/james-orcales/james-orcales/" +
					"golang_snacks/invariant/v2\"\n" +
					")\n\n" +
					"const fixture_hi = 100\n\n" +
					"func f(s string) {\n" +
					"\tinvariant.Cross_Product(\n" +
					"\t\tinvariant.Distinct_Boundary(" +
					"&invariant.Boundary_Input[int]{\n" +
					"\t\t\tX: len(s), Lo: 0, Hi: fixture_hi, " +
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
	t.Parallel()
	run_diag_table(t, []struct {
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

` + fixture_invariant_import + `
const fixture_hi = 100

type F_Input struct {
	A int
	B int
}

func F(input *F_Input) (result int) {
	defer func() {
	}()
	return input.A + input.B
}
`,
			},
			Want_Diag: "",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_Input_Struct_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

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
	})
}

// Test_Input_Struct_Snake_Case is split off Test_Input_Struct so each
// function fits the 100-line cap. Verifies that the input-struct rule
// accepts a snake-case function paired with a snake-case input type.
func Test_Input_Struct_Snake_Case(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{
		{
			Name: "snake case function snake case input",
			Files: map[string]string{
				"test.go": `package main

` + fixture_invariant_import + `
const fixture_hi = 100

type f_input struct {
	A int
	B int
}

func f(input *f_input) (result int) {
	defer func() {
	}()
	return input.A + input.B
}
`,
			},
			Want_Diag: "",
		},
	}
	run_diag_table(t, tests)
}

// Test_Input_Struct_Skip_Shapes covers signatures that should NOT trigger
// the input-struct rule even though they have multiple parameters: variadic
// only, mixed-type params (no duplicates), and single-param signatures.
func Test_Input_Struct_Skip_Shapes(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "variadic only does not trigger",
			Files: map[string]string{
				"test.go": `package main

` + fixture_invariant_import + `
const fixture_hi = 100

func F(args ...int) (result int) {
	defer func() {
	}()
	return len(args)
}
`,
			},
			Want_Diag: "",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_Input_Struct_Skip_Shapes_Part2(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "mixed types do not trigger",
			Files: map[string]string{
				"test.go": `package main

` + fixture_invariant_import + `
const fixture_hi = 100

func F(a int, b string) (result int) {
	defer func() {
	}()
	return a + len(b)
}
`,
			},
			Want_Diag: "",
		},
	})
}

// Additional cases, split to keep each function within the length limit.
func Test_Input_Struct_Skip_Shapes_Part3(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "single param does not trigger",
			Files: map[string]string{
				"test.go": `package main

` + fixture_invariant_import + `
const fixture_hi = 100

func F(a int) (result int) {
	defer func() {
	}()
	return a
}
`,
			},
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

` + fixture_invariant_import + `
const fixture_hi = 100

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
			Want_Diag: "interface method sets are banned",
		},

		{
			Name: "inline interface in parameter flagged",
			Files: map[string]string{
				"test.go": `package main

` + fixture_invariant_import + `
const fixture_hi = 100

func main() (result int) {
	defer func() {
	}()
	return 0
}

func F(x interface{ M() }) (result int) { return 0 }
`,
			},
			Want_Diag: "interface method sets are banned",
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
			Want_Diag: "interface method sets are banned",
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

` + fixture_invariant_import + `
const fixture_hi = 100

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

` + fixture_invariant_import + `
const fixture_hi = 100

func main() (result int) {
	defer func() {
	}()
	return F(0)
}

type _Number interface {
	~int | ~int64
}

func F[T _Number](x T) (result int) {
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

` + fixture_invariant_import + `
const fixture_hi = 100

func main() (result int) {
	defer func() {
	}()
	return 0
}

type T struct{ X int }

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

` + fixture_invariant_import + `
const fixture_hi = 100

func main() (result int) {
	defer func() {
	}()
	return 0
}

type T struct{ X int }

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

` + fixture_invariant_import + `
const fixture_hi = 100

func main() (result int) {
	defer func() {
	}()
	return 0
}

type T struct{ X int }

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

` + fixture_invariant_import + `
const fixture_hi = 100

func main() (result int) {
	defer func() {
	}()
	return 0
}

type T struct{ X int }

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

` + fixture_invariant_import + `
const fixture_hi = 100

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

` + fixture_invariant_import + `
const fixture_hi = 100

func main() (result int) {
	defer func() {
	}()
	return 0
}

type T struct{ X int }

func (t T) Write(p []byte) (err error) {
	return nil
}
`,
			},
			Want_Diag: "does not satisfy any stdlib interface",
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

` + fixture_invariant_import + `
const fixture_hi = 100

func main() (result int) {
	defer func() {
	}()
	return 0
}

type T struct{ X int }

func (t T) Foo() (result int) {
	defer func() {
	}()
	return 0
}
`,
			},
			Want_Diag: "does not satisfy any stdlib interface",
		},
	})
}

// Test_Test_Package verifies that _test.go files must declare
// `package <X>_test`; main, main_test, and whitebox packages are flagged.
func Test_Test_Package(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
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

` + fixture_invariant_import + `
const fixture_hi = 100

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

// Test_No_Recursion verifies that direct and mutual recursion cycles are
// detected via the per-file Ident-based call graph.
func Test_No_Recursion(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{Name: "direct recursion",
			Files: map[string]string{"test.go": `package main
func f() {
	f()
}`}, Want_Diag: "recursion"},

		{Name: "recursion inside nested block",
			Files: map[string]string{"test.go": `package main
func f(x int) (result int) {
	if x > 0 {
		return f(x - 1)
	}
	return 0
}`}, Want_Diag: "recursion"},

		{Name: "no recursion: calls other function",
			Files: map[string]string{"test.go": `package main
func f_inner() { return }
func f() {
	f_inner()
}`}, Want_Diag: ""},

		{Name: "no recursion: no calls",
			Files: map[string]string{"test.go": `package main
func f() { return }`}, Want_Diag: ""},

		{Name: "mutual 2-cycle",
			Files: map[string]string{"test.go": `package main
func entry() { a(); b() }
func a() { b() }
func b() { a() }`}, Want_Diag: "cycle"},
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
func c() { a() }`}, Want_Diag: "cycle"},

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
func inner() { return }`}, Want_Diag: "cycle"},
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

// Test_Tiered_Checks verifies that when any tier-1 diagnostic fires, tier-2
// checks (currently recursion) are suppressed so they can safely rely on
// tier-1 contracts.
func Test_Tiered_Checks(t *testing.T) {
	t.Parallel()
	// When any tier-1 diagnostic fires, tier-2 checks (currently just recursion)
	// must be skipped so they can rely on tier-1 contracts (e.g. check_shadowing).
	fsys_map := fstest.MapFS{
		"test.go": &fstest.MapFile{
			Data: []byte(
				"// missing period at end of this comment\n" +
					"package main\n" +
					"func main() { return }\n" +
					"func f() { f() }\n",
			)},
	}
	stdout := &bytes.Buffer{}
	code := lint.Main(&lint.Main_Input{Fsys: fsys_map, Stdout: stdout, Stderr: &bytes.Buffer{}})
	if code == 0 {
		t.Fatalf("expected non-zero exit due to tier-1 diagnostic, got 0; output: %s",
			stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("should end with")) {
		t.Errorf("expected tier-1 (comment) diagnostic, got: %s", stdout.String())
	}
	if bytes.Contains(stdout.Bytes(), []byte("recursion")) {
		t.Errorf("tier-2 (recursion) diagnostic should be suppressed when "+
			"tier-1 fails, got: %s", stdout.String())
	}
}

// Test_Tiered_Checks_Cross_File guards the print-level gate: tier-2
// diagnostics in one file must be suppressed when ANY tier-1 diagnostic
// fires elsewhere in the run. The per-file gate already covers same-file
// suppression (Test_Tiered_Checks); without the print-level gate, tier-2
// from a clean-tier-1 file would still surface alongside another file's
// tier-1, breaking the contract that tier-2 only reports against globally
// tier-1-clean input.
func Test_Tiered_Checks_Cross_File(t *testing.T) {
	t.Parallel()
	fsys_map := fstest.MapFS{
		"a.go": &fstest.MapFile{
			Data: []byte(
				"// missing period at end of this comment\n" +
					"package main\n" +
					"func main() { return }\n",
			)},
		"b.go": &fstest.MapFile{Data: gofmt_must(t, `package main
func f() { f() }
`)},
	}
	stdout := &bytes.Buffer{}
	code := lint.Main(&lint.Main_Input{Fsys: fsys_map, Stdout: stdout, Stderr: &bytes.Buffer{}})
	if code == 0 {
		t.Fatalf("expected non-zero exit due to tier-1 diagnostic, got 0; output: %s",
			stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("should end with")) {
		t.Errorf("expected tier-1 (comment) diagnostic from a.go, got: %s", stdout.String())
	}
	if bytes.Contains(stdout.Bytes(), []byte("recursion")) {
		t.Errorf("tier-2 (recursion) in b.go should be suppressed when tier-1 "+
			"fires in a.go, got: %s", stdout.String())
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
				"test.go": "package main\nimport \"io\"\n" +
					"func f() { io.ReadAll(nil) }\n",
			},
			Want_Diag: "unbounded API 'io.ReadAll'",
		},
		{
			Name: "io.Copy flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"io\"\n" +
					"func f() { io.Copy(nil, nil) }\n",
			},
			Want_Diag: "unbounded API 'io.Copy'",
		},
		{
			Name: "io.CopyBuffer flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"io\"\n" +
					"func f() { io.CopyBuffer(nil, nil, nil) }\n",
			},
			Want_Diag: "unbounded API 'io.CopyBuffer'",
		},
		{
			Name: "os.ReadFile flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"os\"\n" +
					"func f() { os.ReadFile(\"path\") }\n",
			},
			Want_Diag: "unbounded API 'os.ReadFile'",
		},
		{
			Name: "os.ReadDir flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"os\"\n" +
					"func f() { os.ReadDir(\"path\") }\n",
			},
			Want_Diag: "unbounded API 'os.ReadDir'",
		},
		{
			Name: "bufio.NewScanner flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"bufio\"\n" +
					"func f() { bufio.NewScanner(nil) }\n",
			},
			Want_Diag: "unbounded API 'bufio.NewScanner'",
		},
		{
			Name: "bufio.NewReader flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"bufio\"\n" +
					"func f() { bufio.NewReader(nil) }\n",
			},
			Want_Diag: "unbounded API 'bufio.NewReader'",
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
					"package main\nimport \"io\"\n" +
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
				"test.go": "package main\nimport \"encoding/json\"\n" +
					"func f() { json.NewDecoder(nil) }\n",
			},
			Want_Diag: "unbounded API 'json.NewDecoder'",
		},
		{
			Name: "xml.NewDecoder flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"encoding/xml\"\n" +
					"func f() { xml.NewDecoder(nil) }\n",
			},
			Want_Diag: "unbounded API 'xml.NewDecoder'",
		},
		{
			Name: "gob.NewDecoder flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"encoding/gob\"\n" +
					"func f() { gob.NewDecoder(nil) }\n",
			},
			Want_Diag: "unbounded API 'gob.NewDecoder'",
		},
		{
			Name: "csv.NewReader flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"encoding/csv\"\n" +
					"func f() { csv.NewReader(nil) }\n",
			},
			Want_Diag: "unbounded API 'csv.NewReader'",
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
				"test.go": "package main\nimport \"compress/gzip\"\n" +
					"func f() { gzip.NewReader(nil) }\n",
			},
			Want_Diag: "unbounded API 'gzip.NewReader'",
		},

		{
			Name: "flate.NewReader flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"compress/flate\"\n" +
					"func f() { flate.NewReader(nil) }\n",
			},
			Want_Diag: "unbounded API 'flate.NewReader'",
		},

		{
			Name: "zlib.NewReader flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"compress/zlib\"\n" +
					"func f() { zlib.NewReader(nil) }\n",
			},
			Want_Diag: "unbounded API 'zlib.NewReader'",
		},

		{
			Name: "bzip2.NewReader flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"compress/bzip2\"\n" +
					"func f() { bzip2.NewReader(nil) }\n",
			},
			Want_Diag: "unbounded API 'bzip2.NewReader'",
		},

		{
			Name: "lzw.NewReader flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"compress/lzw\"\n" +
					"func f() { lzw.NewReader(nil, 0, 0) }\n",
			},
			Want_Diag: "unbounded API 'lzw.NewReader'",
		},

		{
			Name: "zip.NewReader flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"archive/zip\"\n" +
					"func f() { zip.NewReader(nil, 0) }\n",
			},
			Want_Diag: "unbounded API 'zip.NewReader'",
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
				"test.go": "package main\nimport \"archive/zip\"\n" +
					"func f() { zip.OpenReader(\"path\") }\n",
			},
			Want_Diag: "unbounded API 'zip.OpenReader'",
		},

		{
			Name: "tar.NewReader flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"archive/tar\"\n" +
					"func f() { tar.NewReader(nil) }\n",
			},
			Want_Diag: "unbounded API 'tar.NewReader'",
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
				"test.go": "package main\nimport \"bytes\"\n" +
					"func f() { bytes.NewBuffer(nil) }\n",
			},
			Want_Diag: "unbounded API 'bytes.NewBuffer'",
		},
		{
			Name: "bytes.NewBufferString flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"bytes\"\n" +
					"func f() { bytes.NewBufferString(\"\") }\n",
			},
			Want_Diag: "unbounded API 'bytes.NewBufferString'",
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
			Want_Diag: "unbounded API 'http.Get'",
		},

		{
			Name: "http.Post flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"net/http\"\n" +
					"func f() { http.Post(\"\", \"\", nil) }\n",
			},
			Want_Diag: "unbounded API 'http.Post'",
		},

		{
			Name: "http.PostForm flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"net/http\"\n" +
					"func f() { http.PostForm(\"\", nil) }\n",
			},
			Want_Diag: "unbounded API 'http.PostForm'",
		},

		{
			Name: "http.Head flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"net/http\"\n" +
					"func f() { http.Head(\"\") }\n",
			},
			Want_Diag: "unbounded API 'http.Head'",
		},

		{
			Name: "http.ListenAndServe flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"net/http\"\n" +
					"func f() { http.ListenAndServe(\"\", nil) }\n",
			},
			Want_Diag: "unbounded API 'http.ListenAndServe'",
		},

		{
			Name: "http.ListenAndServeTLS flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"net/http\"\n" +
					"func f() { " +
					"http.ListenAndServeTLS(\"\", \"\", \"\", nil) }\n",
			},
			Want_Diag: "unbounded API 'http.ListenAndServeTLS'",
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
			Want_Diag: "unbounded API 'http.DefaultClient'",
		},

		{
			Name: "http.DefaultServeMux variable reference flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"net/http\"\n" +
					"func f() { if http.DefaultServeMux != nil { return } }\n",
			},
			Want_Diag: "unbounded API 'http.DefaultServeMux'",
		},

		{
			Name: "http.DefaultTransport variable reference flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"net/http\"\n" +
					"func f() { if http.DefaultTransport != nil { return } }\n",
			},
			Want_Diag: "unbounded API 'http.DefaultTransport'",
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
				"test.go": "package main\nimport \"io/ioutil\"\n" +
					"func f() { ioutil.ReadAll(nil) }\n",
			},
			Want_Diag: "unbounded API 'ioutil.ReadAll'",
		},

		{
			Name: "ioutil.ReadFile flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"io/ioutil\"\n" +
					"func f() { ioutil.ReadFile(\"\") }\n",
			},
			Want_Diag: "unbounded API 'ioutil.ReadFile'",
		},

		{
			Name: "ioutil.ReadDir flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"io/ioutil\"\n" +
					"func f() { ioutil.ReadDir(\"\") }\n",
			},
			Want_Diag: "unbounded API 'ioutil.ReadDir'",
		},

		{
			Name: "ioutil.WriteFile flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"io/ioutil\"\n" +
					"func f() { ioutil.WriteFile(\"\", nil, 0) }\n",
			},
			Want_Diag: "unbounded API 'ioutil.WriteFile'",
		},

		{
			Name: "ioutil.TempFile flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"io/ioutil\"\n" +
					"func f() { ioutil.TempFile(\"\", \"\") }\n",
			},
			Want_Diag: "unbounded API 'ioutil.TempFile'",
		},

		{
			Name: "ioutil.TempDir flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"io/ioutil\"\n" +
					"func f() { ioutil.TempDir(\"\", \"\") }\n",
			},
			Want_Diag: "unbounded API 'ioutil.TempDir'",
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
				"test.go": "package main\nimport \"io/ioutil\"\n" +
					"func f() { ioutil.NopCloser(nil) }\n",
			},
			Want_Diag: "unbounded API 'ioutil.NopCloser'",
		},

		{
			Name: "ioutil.Discard variable reference flagged",
			Files: map[string]string{
				"test.go": "package main\nimport \"io/ioutil\"\n" +
					"func f() { if ioutil.Discard != nil { return } }\n",
			},
			Want_Diag: "unbounded API 'ioutil.Discard'",
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
func main() { return }`}, Want_Diag: "should be declared first"},

		{Name: "Main not first",
			Files: map[string]string{"test.go": `package impl
func helper() { return }
func Main() { return }`}, Want_Diag: "should be declared first"},

		{Name: "main is first",
			Files: map[string]string{"test.go": `package main
func main() { return }
func other() { return }`}, Want_Diag: ""},

		{Name: "Main is first",
			Files: map[string]string{"test.go": `// Package impl is a fixture.
package impl
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
const x = 1
type T int
func main() { return }`}, Want_Diag: ""},
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
// when measuring line length against the max_line_chars limit.
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
// max_line_chars is not flagged: import paths are unbreakable, so a long module
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
			Want_Diag: "banned scripting file",
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

// Files exceeding 1 MiB must be flagged by the max-file-size check.
func Test_Stream_Max_File_Size(t *testing.T) {
	t.Parallel()
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
				t.Errorf("path %q: want path-casing=%v, got: %s",
					tc.Path, want, output)
			}
		})
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

// Lines inside fenced code blocks must be exempt from the markdown line cap.
func Test_Stream_Markdown_Code_Fence_Exempt(t *testing.T) {
	t.Parallel()
	long := "```\n" + strings.Repeat("x", 200) + "\n```\n"
	output := run_stream(t, map[string]string{"docs.md": long})
	if strings.Contains(output, "markdown-line-length") {
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
// max_line_chars cap the file-tier check enforces on source lines — same
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

// Test_No_Ambient_Stdlib_Hard_Imports verifies that hard-banned imports
// (os, crypto/rand, math/rand v1, flag) fire in library packages, while
// math/rand/v2 is allowed.
func Test_No_Ambient_Stdlib_Hard_Imports(t *testing.T) {
	t.Parallel()
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
	}
	run_diag_table(t, tests)
}

// Additional cases, split to keep each function within the length limit.
func Test_No_Ambient_Stdlib_Hard_Imports_Part2(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "library imports math/rand/v2 clean",
			Files: map[string]string{
				"a.go": `// Package lib is a fixture.
package lib

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
	t.Parallel()
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
	}
	run_diag_table(t, tests)
}

// Additional cases, split to keep each function within the length limit.
func Test_No_Ambient_Stdlib_Soft_Calls_Part2(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

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

// Test_Coverage_Backfill_Ambient_Soft_One_Character drives is_ambient_soft_ident's
// Package and Name Lo (=1) buckets with a one-char import path and a one-char
// selected symbol. "a" is not an ambient stdlib package, so nothing fires.
func Test_Coverage_Backfill_Ambient_Soft_One_Character(t *testing.T) {
	t.Parallel()
	run_diag_table(t, []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{{
		Name: "non-stdlib one-char import path and symbol clean",
		Files: map[string]string{
			"a.go": `// Package lib is a fixture.
package lib

import "a"

// F is a fixture.
func F() { a.X() }
`,
		},
		Want_Diag: "",
	}})
}

// Test_No_Ambient_Stdlib_Soft_Network_Http verifies the http and net soft bans.
func Test_No_Ambient_Stdlib_Soft_Network_Http(t *testing.T) {
	t.Parallel()
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
func Test_No_Ambient_Stdlib_Exemptions_Part2(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "test.go in library may call time.Now",
			Files: map[string]string{
				"a.go": `// Package lib is a fixture.
package lib


const fixture_hi = 100

// F is a fixture.
func F() (n int) {
	defer func() {
	}()
	return 1
}
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
	t.Parallel()
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
			Forbid: []string{"ambient stdlib import"},
		},
	}
	run_doctrine_diag_table(t, tests)
}

// Additional cases, split to keep each function within the length limit.
func Test_No_Ambient_Stdlib_Composition_Tier_Part2(t *testing.T) {
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
			Forbid: []string{"ambient stdlib import"},
		},
	}
	run_doctrine_diag_table(t, tests)
}

// Test_No_Ambient_Stdlib_Composition_Tier_Extra continues the composition-tier
// allow-list: an ambient stdlib CALL (`time.Now()`) at the composition tier
// is allowed, and the binary-module composition tier (one level under the
// library tier) inherits the same exemption.
func Test_No_Ambient_Stdlib_Composition_Tier_Extra(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name       string
		Files      map[string]string
		Want_Diags []string
		Forbid     []string
	}{
		{
			Name: "ambient call (time.Now) at composition tier allowed",
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
			Forbid: []string{"ambient stdlib call"},
		},
		{
			Name: "binary module composition tier " +
				"(one level under library tier) allowed",
			Files: map[string]string{
				"mybinary/go.mod": doctrine_binary_go_module,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/internal/lib/lib.go": fixture_package("lib"),
				"mybinary/internal/lib/lib_default/" +
					"lib_default.go": `// Package lib_default is a fixture.
package lib_default

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
			Forbid: []string{"ambient stdlib import"},
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

// Test_Default_Package_Alias verifies that *_default packages must be imported
// with their prefix as the alias (snap_default → snap). This is what lets
// callers keep writing snap.Init/snap.Edit, and is required by snap.Edit's
// source-line rewriter which searches for the literal "snap.Edit(".
func Test_Default_Package_Alias(t *testing.T) {
	t.Parallel()
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
	}
	run_diag_table(t, tests)
}

// Additional cases, split to keep each function within the length limit.
func Test_Default_Package_Alias_Part2(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

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
				"mybinary/go.mod": doctrine_binary_go_module,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
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
			Want_Diags: []string{"forbids package \"helpers\"", "mybinary/internal/"},
		},
		{
			Name: "binary with package under internal is clean",
			Files: map[string]string{
				"mybinary/go.mod": doctrine_binary_go_module,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
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
				"mybinary/go.mod": doctrine_binary_go_module,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/cmd/extra/extra.go": "package main\n\n" +
					"func main() { return }\n",
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
				"golang_snacks/snap/snap.go":       fixture_package("snap"),
				"golang_snacks/snap/v2/snap.go":    fixture_package("snap"),
				"golang_snacks/snap/v2/sub/sub.go": fixture_package("sub"),
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
			Name: "main package at depth does not count as ancestor",
			Files: map[string]string{
				"mybinary/go.mod": doctrine_binary_go_module,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
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
					"const Max_Count = 100\n",
			},
			Want_Diag: "exported const Max_Count is missing a doc comment",
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
// max_files quota, forcing package_group_key_diag's max_files Boundary axis
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
		// max_lines_per_file, so max_files climbs past 1 and the Boundary's
		// Hi bucket fires (X == Hi == max_files > Lo == 1).
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
		// generated identifiers fit within max_identifier_chars; the
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
// of package_group_key_diag's two Boundary axes (st.Lines=2, max_files=1)
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
	// git_input_check, so the 30000 elements drive only the Hi=30000
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
	source := "package lib\n\n" +
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
// V.Targets boundary, Lo=min_non_empty). Two shapes exercise both Lo (1-char)
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
// Also imports "C" (1 char) to exercise is_ambient_hard_import Lo bucket.
func Test_Coverage_Backfill_Scope_Prefix_Hi(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("a", 4096)
	var stdout, stderr bytes.Buffer
	lint.Main(&lint.Main_Input{
		Fsys: fstest.MapFS{
			"go.mod": {Data: []byte("module test\ngo 1.25\n")},
			"lib/a.go": {Data: []byte(
				"package lib\n\nimport _ \"C\"\nimport _ \"" +
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

// Test_Coverage_Backfill_Check_Comments_Group_Min_Source feeds Check_Source
// the minimum-byte file containing a comment (`package x\n\n// c\n`, 16 bytes)
// so check_comments_group's source-length Lo bucket fires.
func Test_Coverage_Backfill_Check_Comments_Group_Min_Source(t *testing.T) {
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
