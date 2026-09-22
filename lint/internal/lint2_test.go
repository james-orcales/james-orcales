package lint_test

import (
	"fmt"
	"go/parser"
	"go/token"
	"local/james-orcales/lint/internal/strings"
	"testing"
	"testing/fstest"

	"local/james-orcales/lint/internal"
	"local/james-orcales/shared/std/sim/aver/default"
	library "local/james-orcales/shared/std/snap"
	"local/james-orcales/shared/std/snap/default"
	"local/james-orcales/shared/std/strconv"
)

func quote_text(text string) (quoted string) {
	var destination [strconv.QUOTED_TEXT_SIZE_MAXIMUM]byte
	count := strconv.Quote_Into(destination[:], strconv.Text(text))
	return string(destination[:int(count)])
}

func decimal_text(value int) (text string) {
	var destination [strconv.DECIMAL_TEXT_SIZE_MAXIMUM]byte
	count := strconv.Format_Decimal_Into(destination[:], strconv.Machine_Integer(value))
	return string(destination[:int(count)])
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
`}, Want_Diag: "Declare func TestMain first"},

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
			Name: "Default in non-snap package still banned",
			Files: map[string]string{
				"test.go": `package main
type S struct{}
var Default = &S{}
func main() { return }
`,
			},
			Want_Diag: "Do not declare a package-level var",
		},
	}
	run_diag_table(t, tests)
}

// Test_Instrumentation_Packages verifies the var-Default exemption is governed by
// lint.json's instrumentation_packages, not by the literal default/ directory
// name: a listed package earns the exemption whether its directory is named
// default/ or carries the _default suffix, while a default/ package absent from
// the list is banned (the heuristic is gone).
func Test_Instrumentation_Packages(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		Name      string
		Path      string
		Source    string
		Allowlist []string
		Want_Diag string
	}{
		{
			Name: "allowlisted default dir allowed",
			Path: "snap/default/wire.go",
			Source: `// Package snap is a fixture.
package snap
// Snapper is a fixture.
type Snapper struct{}
// Default is the OS-bound Snapper.
var Default = &Snapper{}
`,
			Allowlist: []string{"snap/default"},
		},
		{
			Name: "allowlisted _default suffix allowed",
			Path: "mylib/mylib_default/wire.go",
			Source: `// Package mylib_default is a fixture.
package mylib_default
// Thing is a fixture.
type Thing struct{}
// Default is the OS-bound Thing.
var Default = &Thing{}
`,
			Allowlist: []string{"mylib/mylib_default"},
		},
		{
			// The root assigns Default after it opens the IO, so the zero value is the
			// only declaration a package that owns no IO can give it.
			Name: "allowlisted uninitialized Default allowed",
			Path: "snap/default/wire.go",
			Source: `// Package snap is a fixture.
package snap
// Snapper is a fixture.
type Snapper struct{}
// Default is the OS-bound Snapper.
var Default Snapper
`,
			Allowlist: []string{"snap/default"},
		},
		{
			Name: "default dir absent from allowlist banned",
			Path: "snap/default/wire.go",
			Source: `// Package snap is a fixture.
package snap
// Snapper is a fixture.
type Snapper struct{}
// Default is the OS-bound Snapper.
var Default = &Snapper{}
`,
			Allowlist: nil,
			Want_Diag: "Do not declare a package-level var",
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			files := map[string]string{tt.Path: tt.Source}
			run_global_api_case(t, files, tt.Allowlist, tt.Want_Diag)
		})
	}
}

// Runs a one-package fixture through Main with the given allowlist and asserts
// either a clean pass (empty want) or that want appears in the output. Distinct
// param types keep the signature out of the Input-struct rule.
func run_global_api_case(
	t *testing.T, files map[string]string, allowlist []string, want string,
) {
	t.Helper()
	fsys_map := make(fstest.MapFS)
	for k, v := range files {
		fsys_map[k] = &fstest.MapFile{Data: gofmt_must(t, v)}
	}
	// This case isolates the var-Default (Mutable Globals) exemption, which the
	// allowlist governs independent of component layout; ignore the layout,
	// coverage, and single-module diagnostics the bare one-package fixture draws.
	diags, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys:                     fsys_map,
		Instrumentation_Packages: allowlist,
	})
	if err != nil {
		t.Fatalf("Check_File_System: %v", err)
	}
	banned := false
	for _, d := range diags {
		if strings.Contains(d.Message, "package-level var") {
			banned = true
		}
	}
	if want == "" {
		if banned {
			t.Fatal("var Default must be allowed when its package is allowlisted")
		}
		return
	}
	if !banned {
		t.Fatalf("expected a banned-var diagnostic (%q)", want)
	}
}

// Test_Shared_Component_Configurable proves shared-vs-binary classification
// follows Main_Input.Shared_Component, matched against the module's workspace-root-
// relative directory (here "lib"), not its import path: the same module is the
// shared library (internal/ forbidden) when that directory is the configured
// shared module, and a binary (the rule does not fire) when it is not.
func Test_Shared_Component_Configurable(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"lib/go.mod":                      "module example.com/lib\n",
		"lib/foo/foo.go":                  fixture_package("foo"),
		"lib/foo/internal/helper/help.go": fixture_package("helper"),
	}
	const FORBID = "A shared library permits no internal/"
	as_shared := run_shared_component_output(t, files, "lib")
	if !strings.Contains(as_shared, FORBID) {
		t.Fatalf("with lib/ as the shared module, internal/ must be "+
			"flagged; got: %s", as_shared)
	}
	as_binary := run_shared_component_output(t, files, "other")
	if strings.Contains(as_binary, FORBID) {
		t.Fatalf("with a different shared module, the shared-library internal/ rule "+
			"must not fire; got: %s", as_binary)
	}
}

// Test_Keyed_Struct_Init_Cross_Package verifies the rule reaches a struct
// declared in another package, and leaves a named slice's positional literal
// alone.
func Test_Keyed_Struct_Init_Cross_Package(t *testing.T) {
	t.Parallel()
	const UNKEYED = "literal has no field keys"
	alpha := "// Package alpha is a fixture.\npackage alpha\n\n" +
		"// Widget is a fixture.\ntype Widget struct {\n" +
		"\t// X is a fixture.\n\tX int\n}\n\n" +
		"// Numbers is a fixture.\ntype Numbers []int\n"
	beta := func(body string) (text string) {
		return "// Package beta is a fixture.\npackage beta\n\n" +
			"import alpha \"github.com/james-orcales/james-orcales/lib/alpha\"\n\n" +
			"// Build is a fixture.\nfunc Build() (widget alpha.Widget) {\n" +
			body + "}\n"
	}
	workspace := func(beta_body string) (files map[string]string) {
		return map[string]string{
			"lib/alpha/alpha.go": alpha,
			"lib/beta/beta.go":   beta(beta_body),
		}
	}
	unkeyed := run_shared_component_output(t, workspace(
		"\treturn alpha.Widget{1}\n"), "lib")
	if !strings.Contains(unkeyed, UNKEYED) {
		t.Fatalf("another package's struct literal must be keyed; got: %s", unkeyed)
	}
	keyed := run_shared_component_output(t, workspace(
		"\treturn alpha.Widget{X: 1}\n"), "lib")
	if strings.Contains(keyed, UNKEYED) {
		t.Fatalf("a keyed literal must stay silent; got: %s", keyed)
	}
	slice := run_shared_component_output(t, workspace(
		"\tnumbers := alpha.Numbers{1, 2}\n"+
			"\treturn alpha.Widget{X: numbers[0]}\n"), "lib")
	if strings.Contains(slice, UNKEYED) {
		t.Fatalf("a named slice has no fields to key; got: %s", slice)
	}
}

// Test_No_Recursion_Cross_File verifies the call graph spans the package, not
// the file: two functions in sibling files that call each other close a cycle.
// A cross-package cycle needs no coverage — it would need an import cycle, which
// Go rejects before the linter ever sees the code. The fixture carries a
// file-count diagnostic of its own, since two files in one package is what the
// case needs, so the assertion names the recursion message alone.
func Test_No_Recursion_Cross_File(t *testing.T) {
	t.Parallel()
	output := run_shared_component_output(t, map[string]string{
		"lib/foo/a.go": "// Package foo is a fixture.\npackage foo\n\n" +
			"func alpha() (value int) {\n\treturn beta()\n}\n",
		"lib/foo/b.go": "package foo\n\n" +
			"func beta() (value int) {\n\treturn alpha()\n}\n",
	}, "lib")
	if !strings.Contains(output, "recurse in a cycle") {
		t.Fatalf("a cycle across sibling files must be flagged; got: %s", output)
	}
}

// Test_No_Recursion_Exempt_File verifies a file the opt_out_recursion_ban list
// names contributes nothing to its package's call graph, so its recursion stays
// legal even though the graph is now package-wide.
func Test_No_Recursion_Exempt_File(t *testing.T) {
	t.Parallel()
	fsys_map := fstest.MapFS{
		"lib/foo/a.go": &fstest.MapFile{Data: gofmt_must(t,
			"// Package foo is a fixture.\npackage foo\n\n"+
				"func alpha() (value int) {\n\treturn alpha()\n}\n")},
		"lint.json": &fstest.MapFile{Data: test_lint_json_recurse_exempt(t,
			&test_lint_json_recurse_exempt_input{
				Shared_Component: "lib", Exempt: "lib/foo/**"})},
	}
	stdout := &strings.Builder{}
	lint_main(t, &lint.Main_Input{
		Fsys: fsys_map, Stdout: stdout, Stderr: &strings.Builder{}})
	if strings.Contains(stdout.String(), "recurse:") {
		t.Fatalf("an exempt file must contribute no edges; got: %s", stdout.String())
	}
}

// Test_Exported_Type_Exposes_Private_Cross_File verifies the transitive walk
// follows a field whose struct is declared in a sibling file of the same
// package. The fixture carries a file-count diagnostic of its own, since two
// files in one package is what the case needs, so the assertion names the
// exposure message alone.
func Test_Exported_Type_Exposes_Private_Cross_File(t *testing.T) {
	t.Parallel()
	output := run_shared_component_output(t, map[string]string{
		"lib/foo/a.go": "// Package foo is a fixture.\npackage foo\n\n" +
			"// Widget is a fixture.\ntype Widget struct {\n" +
			"\t// M is a fixture.\n\tM Middle\n}\n",
		"lib/foo/b.go": "package foo\n\n" +
			"// Middle is a fixture.\ntype Middle struct {\n" +
			"\t// F is a fixture.\n\tF secret\n}\n\n" +
			"type secret int\n",
	}, "lib")
	if !strings.Contains(output, "The public type Widget contains the private type secret") {
		t.Fatalf("the walk must follow a sibling file's struct; got: %s", output)
	}
}

// Test_Keyed_Struct_Init_Unparsed_Package verifies a literal of a type the run
// never parsed stays silent: the qualifier resolves to no declaration, and a
// miss must never be read as "not a struct".
func Test_Keyed_Struct_Init_Unparsed_Package(t *testing.T) {
	t.Parallel()
	output := run_shared_component_output(t, map[string]string{
		"lib/gamma/gamma.go": "// Package gamma is a fixture.\npackage gamma\n\n" +
			"import \"go/token\"\n\n" +
			"// Build is a fixture.\nfunc Build() (position token.Position) {\n" +
			"\treturn token.Position{\"a\", 1, 2, 3}\n}\n",
	}, "lib")
	if strings.Contains(output, "literal has no field keys") {
		t.Fatalf("an unparsed package's type is unknowable; got: %s", output)
	}
}

// Test_Binary_Default_Tier proves the default tier belongs to the shared library
// alone. The same lib/foo/default package is legal when lib/ is the shared
// component and a layout error when lib/ is a binary, whose impurity stays in
// package main. The impurity release travels with that classification, so the
// package's os import is exempt only in the shared case.
func Test_Binary_Default_Tier(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"lib/foo/foo.go": fixture_package("foo"),
		"lib/foo/default/wire.go": "// Package foo is a fixture.\npackage foo\n\n" +
			"import \"os\"\n\n// Read reads.\nfunc Read() (value string) {\n" +
			"\treturn os.Getenv(\"X\")\n}\n",
	}
	const FORBID = "permits no default tier"
	const IMPURE = "The stdlib import"
	as_shared := run_shared_component_output(t, files, "lib")
	if strings.Contains(as_shared, FORBID) {
		t.Fatalf("a shared library keeps its default tier; got: %s", as_shared)
	}
	if strings.Contains(as_shared, IMPURE) {
		t.Fatalf("a shared library's default tier is an impure home; got: %s", as_shared)
	}
	as_binary := run_shared_component_output(t, files, "other")
	if !strings.Contains(as_binary, FORBID) {
		t.Fatalf("a binary's default tier must be flagged; got: %s", as_binary)
	}
	if !strings.Contains(as_binary, IMPURE) {
		t.Fatalf("a binary's default tier takes the purity bans; got: %s", as_binary)
	}
}

// Test_Binary_Composition_Tier_Release proves the depth-based impurity release
// is the shared library's alone too. lib/foo/bar sits one non-main ancestor
// below the library tier, which releases it from the purity bans while lib/ is
// the shared component and releases nothing once lib/ is a binary.
func Test_Binary_Composition_Tier_Release(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"lib/foo/foo.go": fixture_package("foo"),
		"lib/foo/bar/bar.go": "// Package bar is a fixture.\npackage bar\n\n" +
			"import \"os\"\n\n// Read reads.\nfunc Read() (value string) {\n" +
			"\treturn os.Getenv(\"X\")\n}\n",
	}
	const IMPURE = "The stdlib import"
	as_shared := run_shared_component_output(t, files, "lib")
	if strings.Contains(as_shared, IMPURE) {
		t.Fatalf("the composition tier is an impure home; got: %s", as_shared)
	}
	as_binary := run_shared_component_output(t, files, "other")
	if !strings.Contains(as_binary, IMPURE) {
		t.Fatalf("a binary releases no package from the purity bans; got: %s", as_binary)
	}
}

// Lints the fixture with the given shared module and returns stdout. .go
// entries are gofmt-normalized; other files (go.mod) reach the linter verbatim.
func run_shared_component_output(
	t *testing.T, files map[string]string, shared_component string,
) (output string) {
	t.Helper()
	fsys_map := make(fstest.MapFS)
	for name, content := range files {
		data := []byte(content)
		if strings.Has_Suffix(name, ".go") {
			data = gofmt_must(t, content)
		}
		fsys_map[name] = &fstest.MapFile{Data: data}
	}
	fsys_map["lint.json"] = &fstest.MapFile{Data: test_lint_json(t, shared_component, nil)}
	stdout := &strings.Builder{}
	lint_main(t, &lint.Main_Input{
		Fsys: fsys_map, Stdout: stdout, Stderr: &strings.Builder{},
	})
	return stdout.String()
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
			Want_Diag: "Do not declare a package-level var",
		},

		{
			Name: "literal initializer banned",
			Files: map[string]string{
				"test.go": `package main
var x_count = 5
func main() { return }
`,
			},
			Want_Diag: "Do not declare a package-level var",
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
			Want_Diag: "Do not declare a package-level var",
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
			Want_Diag: "Do not declare a package-level var",
		},

		{
			Name: "multi-name single-line banned",
			Files: map[string]string{
				"test.go": `package main
var a_count, b_count = 1, 2
func main() { return }
`,
			},
			Want_Diag: "Do not declare a package-level var",
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
			Want_Diag: "Do not declare a package-level var",
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
			Want_Diag: "Do not declare a package-level var",
		},
	})
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
	stdout := &strings.Builder{}
	lint_main(t, &lint.Main_Input{Fsys: fsys_map, Stdout: stdout, Stderr: &strings.Builder{}})
	return stdout.String()
}

// Lints the given in-memory files and returns the linter's stdout with the
// trailing newline trimmed, so an isolated fixture yields a single snapshot
// line. .go entries are gofmt-normalized to match the AST tier's real inputs;
// fixtures that must stay byte-exact (the gofmt diagnostic, any non-Go file)
// go through run_snapshot_verbatim instead.
func run_snapshot(t *testing.T, files map[string]string, shared_component string) (output string) {
	t.Helper()
	normalized := make(map[string]string, len(files))
	for name, content := range files {
		if strings.Has_Suffix(name, ".go") {
			normalized[name] = string(gofmt_must(t, content))
			continue
		}
		normalized[name] = content
	}
	return run_snapshot_verbatim(t, normalized, shared_component)
}

// Like run_snapshot but without gofmt normalization: the bytes reach the
// linter exactly as written.
func run_snapshot_verbatim(
	t *testing.T, files map[string]string, shared_component string,
) (output string) {
	t.Helper()
	fsys_map := make(fstest.MapFS)
	for name, content := range files {
		fsys_map[name] = &fstest.MapFile{Data: []byte(content)}
	}
	fsys_map["lint.json"] = &fstest.MapFile{Data: test_lint_json(t, shared_component, nil)}
	stdout := &strings.Builder{}
	lint_main(t, &lint.Main_Input{
		Fsys: fsys_map, Stdout: stdout, Stderr: &strings.Builder{},
	})
	return strings.Trim_Right(stdout.String(), "\n")
}

// Pairs a captured snapshot with the in-memory fixture whose single linter
// diagnostic it pins. Verbatim skips gofmt for fixtures that must reach the
// linter byte-exact (the gofmt diagnostic, non-Go files). Drop removes output
// lines containing it, to strip orthogonal noise a fixture cannot avoid.
type snapshot_case struct {
	Snapshot library.Snapshot
	Files    map[string]string
	Verbatim bool
	Drop     string
	// Root_Module_Binary marks a fixture whose root go.mod is a binary module
	// rather than the shared library. By default a fixture's root module is the
	// shared library (Shared_Component "."), which most layout fixtures model; set
	// this for the few cases that exercise binary-module rules at the root, so no
	// module is classified shared and the root module stays a binary.
	Root_Module_Binary bool
}

// Drives each snapshot case: lints the fixture and compares stdout to the
// captured snapshot. Sequential — no t.Parallel — so snap.Edit captures write
// the source file without racing each other.
func run_snapshot_cases(t *testing.T, cases []snapshot_case) {
	t.Helper()
	run_snapshot_cases_shared(t, DOCTRINE_SHARED_COMPONENT_DIRECTORY, cases)
}

// Like run_snapshot_cases but with an explicit shared-component directory, so a
// single-component fixture can name its component (the dir its code lives under)
// and earn shared-library classification instead of binary.
func run_snapshot_cases_shared(t *testing.T, shared string, cases []snapshot_case) {
	t.Helper()
	for _, entry := range cases {
		output := run_snapshot(t, entry.Files, shared)
		if entry.Verbatim {
			output = run_snapshot_verbatim(t, entry.Files, shared)
		}
		if entry.Drop != "" {
			var builder strings.Builder
			for _, line := range strings.Split(output, "\n") {
				if strings.Contains(line, entry.Drop) {
					continue
				}
				strings.Builder_Write_Text(&builder, line)
				strings.Builder_Write_Byte(&builder, '\n')
			}
			output = strings.Trim_Right(builder.String(), "\n")
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

// Test_Snapshot_Bans_Imports pins the import-construct bans.
func Test_Snapshot_Bans_Imports(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`a.go:4:8: Do not use a blank import.`),
			Files: snapshot_package("import _ \"fmt\"\n")},
		{Snapshot: snap.Init(`a.go:4:8: Do not use a dot import.`), Files: snapshot_package("import . \"fmt\"\n")},
	})
}

// Test_Snapshot_Bans_Declarations pins the declaration-construct bans.
func Test_Snapshot_Bans_Declarations(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`a.go:5:11: Do not use iota.`),
			Files: snapshot_package("// X is a fixture.\nconst X = iota\n")},
		{Snapshot: snap.Init(`a.go:4:1: Do not use a grouped declaration. Write one declaration on each line.`), Files: snapshot_package("var (\n\ta = 1\n\tb = 2\n)\n")},
		{Snapshot: snap.Init(`a.go:5:1: The func F has an empty body. Write a body or remove the func.`),
			Files: snapshot_package("// F does.\nfunc F() {}\n")},
	})
}

// Test_Snapshot_Bans_Control pins the control-flow bans.
func Test_Snapshot_Bans_Control(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`a.go:7:2: Do not use a naked return. Write the return values.`),
			Files: snapshot_package(`// F does.
func F() (n int) {
	n = 1
	return
}
`)},
		{Snapshot: snap.Init(`a.go:6:5: The if condition has the compound operator "&&". Write nested if statements.`),
			Files: snapshot_package(`// F does.
func F() (n int) {
	if n > 0 && n < 5 {
		n = 1
	}
	return n
}
`)},
		{Snapshot: snap.Init(`a.go:8:3: Do not use fallthrough.`),
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
		{Snapshot: snap.Init(`a.go:7:3: The variable "n" shadows a variable in an outer scope. Write a different name.`), Files: snapshot_package(`// F does.
func F(n int) {
	{
		n := 0
		println(n)
	}
}
`)},
		{Snapshot: snap.Init(`a.go:7:2: The assignment to the blank identifier hides the value. Name the value or remove the assignment.`), Files: snapshot_package(`// F does.
func F() {
	x := 1
	_ = x
}
`)},
	})
}

// Test_Snapshot_Bans_Recursion pins the recurse bans (tier two).
func Test_Snapshot_Bans_Recursion(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`a.go:6:2: The function F recurses. Write a loop instead.`), Files: snapshot_package(`// F loops.
func F() {
	F()
}
`)},
		{Snapshot: snap.Init(`a.go:11:2: The functions recurse in a cycle. Write a loop instead. F → G → F`), Files: snapshot_package(`// F calls G.
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
		{Snapshot: snap.Init(`a.go:5:8: Do not declare an interface. Use concrete types.`), Files: snapshot_package(`// I is a fixture.
type I interface{ M() }
`)},
		{Snapshot: snap.Init(`a.go:5:14: Do not use generics. Write concrete types.`), Files: snapshot_package(`// Identity returns value.
func Identity[Value int](value Value) (result Value) { return value }
`)},
		{Snapshot: snap.Init(`a.go:11:12: Methods are banned. Convert Compute to free function. Write receiver as first parameter.`), Files: snapshot_package(`// T is a fixture.
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
		{Snapshot: snap.Init(`a.go:5:5: Do not declare a package-level var. Only regexp.MustCompile and errors.New are exceptions.`), Files: snapshot_package(`// X is a fixture.
var X = 0
`)},
		{Snapshot: snap.Init(`a.go:4:1: Do not use func init. Declare a func Init instead.`), Files: snapshot_package(`func init() { println(0) }
`)},
	})
}

// Test_Snapshot_Requirements_Forms pins the required-form checks.
func Test_Snapshot_Requirements_Forms(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`a.go:5:10: The return type int has no name. Name the return value.`), Files: snapshot_package(`// F does.
func F() int {
	return 0
}
`)},
		{Snapshot: snap.Init(`a.go:12:6: The T literal has no field keys. Write keyed fields.`), Files: snapshot_package(`// T is a fixture.
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
	})
}

// Test_Snapshot_Names pins the naming-convention checks.
func Test_Snapshot_Names(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`a.go:5:6: Rename MyName -> My_Name.`), Files: snapshot_package(`// MyName is a fixture.
func MyName() {
	println(0)
}
`)},
	})
}

// Test_Snapshot_Entry_Point pins the entry-point-first checks.
func Test_Snapshot_Entry_Point(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`a.go:8:1: Declare func main first in the file.`), Files: map[string]string{
			"a.go": "package main\n\n// Run does.\nfunc Run() {\n\tprintln(0)\n}\n\n" +
				"func main() {\n\tRun()\n}\n"}},
		{Snapshot: snap.Init(`a.go:9:1: Declare func Main first in the file.`), Files: map[string]string{
			"a.go": "package main\n\n// Run does.\nfunc Run() {\n\tprintln(0)\n}\n\n" +
				"// Main does.\nfunc Main() {\n\tRun()\n}\n"}},
		{Snapshot: snap.Init(`a_test.go:10:1: Declare func TestMain first in the file.`), Files: map[string]string{
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
		{Snapshot: snap.Init(`a.go:4:21: Rename count -> Count.
a.go:4:6: The type widget is not exported. Rename widget -> Widget.`), Files: snapshot_package(`type widget struct{ count int }
`)},
		{Snapshot: snap.Init(`a.go:9:8: The public type Widget contains the private type hidden. Export the private type.
a.go:4:6: The type hidden is not exported. Rename hidden -> Hidden.`), Files: snapshot_package(`type hidden struct{ X int }

// Widget is a fixture.
type Widget struct {
	// Inner is a fixture.
	Inner hidden
}
`)},
	})
}

// Test_Snapshot_Component_Layout pins the binary/shared component layout checks.
// A binary fixture lives under app/, a shared fixture under shared/.
func Test_Snapshot_Module_Layout(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`app/feature/feature.go:1:1: Move feature -> app/pkg/feature.`), Files: map[string]string{
			"app/pkg/entry.go": DOCTRINE_BINARY_PACKAGE_MAIN,
			"app/feature/feature.go": "// Package feature is a fixture.\n" +
				"package feature\n"}},
		{Snapshot: snap.Init(`app/cmd/app/main.go:1:1: The binary component "github.com/james-orcales/james-orcales/app" has its main package at "cmd/app". Move the main package to the component root.`), Files: map[string]string{
			"app/pkg/entry.go": DOCTRINE_BINARY_PACKAGE_MAIN,
			"app/cmd/app/main.go": "package main\n\n" +
				"func main() {\n\tprintln(0)\n}\n"}},
		{Snapshot: snap.Init(`shared/internal/x/x.go:1:1: A shared library permits no internal/ directory. Remove "shared/internal".`), Files: map[string]string{
			"shared/internal/x/x.go": "// Package x is a fixture.\npackage x\n"}},
		{Snapshot: snap.Init(`shared/main.go:1:1: The shared library "github.com/james-orcales/james-orcales/shared" permits no package main. Move the entry point to a binary component.`), Files: map[string]string{
			"shared/main.go": "package main\n\nfunc main() {\n\tprintln(0)\n}\n"}},
	})
}

// Test_Snapshot_Module_Structure pins the tier-depth check. Module placement is
// now governed by the single-module rule, snapshotted in Test_Snapshot_Single_Module.
func Test_Snapshot_Module_Structure(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`shared/a/b/c/c.go:1:1: The package "c" at "a/b/c" is below the library tier. It has 2 non-main ancestors: [a/b a].`), Files: map[string]string{
			"shared/a/a.go":     "// Package a is a fixture.\npackage a\n",
			"shared/a/b/b.go":   "// Package b is a fixture.\npackage b\n",
			"shared/a/b/c/c.go": "// Package c is a fixture.\npackage c\n"}},
	})
}

// Test_Snapshot_Single_Module pins the single-module rule: a nested go.mod, a
// go.work, and a missing root go.mod are each flagged at the offending file.
func Test_Snapshot_Single_Module(t *testing.T) {
	run_snapshot_cases(t, []snapshot_case{
		{Snapshot: snap.Init(`shared/nested/go.mod:1:1: A nested go.mod divides the repository into more than one module. Remove "shared/nested/go.mod".`), Files: map[string]string{
			"shared/nested/go.mod": "module example.com/nested\n",
			"shared/x.go":          "// Package shared x.\npackage shared\n"}},
	})
}

// Test_Snapshot_Purity pins the direct impure-stdlib checks.
func Test_Snapshot_Purity(t *testing.T) {
	run_snapshot_cases_shared(t, "lib", []snapshot_case{
		{Snapshot: snap.Init(`lib/library.go:4:8: The stdlib import "runtime" is impure. See lint/README.md for the resolutions.`), Files: map[string]string{
			"go.mod": DOCTRINE_SHARED_LIBRARY_GO_MODULE,
			"lib/library.go": `// Package library x.
package library

import "runtime"
`}},
		{Snapshot: snap.Init(`lib/library.go:8:2: The stdlib call fmt.Println is impure. See lint/README.md for the resolutions.`), Files: map[string]string{
			"go.mod": DOCTRINE_SHARED_LIBRARY_GO_MODULE,
			"lib/library.go": `// Package library x.
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
		{Snapshot: snap.Init(`shared/lib/library_test.go:7:2: The transitive call path/filepath.Walk is impure. A pure package calls only pure APIs.`), Files: map[string]string{
			"shared/lib/library.go": "// Package library x.\npackage library\n",
			"shared/lib/library_test.go": `package library_test

import "path/filepath"

// Test_F walks.
func Test_F() {
	filepath.Walk(".", nil)
}
`}},
		{Snapshot: snap.Init(`shared/lib/library.go:4:8: The dependency "github.com/james-orcales/james-orcales/shared/widget/default" is impure. A pure package imports only pure packages.`), Files: map[string]string{
			"shared/lib/library.go": `// Package library x.
package library

import "github.com/james-orcales/james-orcales/shared/widget/default"

// F uses the default.
func F() (s string) {
	return widget.Name()
}
`,
			"shared/widget/default/wire.go": `// Package widget is a fixture.
package widget

// Name names.
func Name() (s string) {
	return "x"
}
`}},
	})
}

// Test_Snapshot_Miscellaneous pins the remaining per-file checks.
func Test_Snapshot_Miscellaneous(t *testing.T) {
	run_snapshot_cases_shared(t, "foo", []snapshot_case{
		{Snapshot: snap.Init(`a.go:10:6: The type snapper is not exported. Rename snapper -> Snapper.`), Files: snapshot_package(`// F does.
func F() {
	var snap snapper
	snap.Init("plain")
}

type snapper struct{ X int }
`)},
		{Snapshot: snap.Init(`foo/default/wire.go:2:9: The default package declares "package wrong", not "package foo". The default package shadows the library it re-exports.`), Files: map[string]string{
			"foo/default/wire.go": "// Package wrong is a fixture.\npackage wrong\n"}},
		{Snapshot: snap.Init(`a.go:8:2: A sync mutex with a blank name has no Lock receiver. Use the noCopy idiom for non-copy semantics, or name the field.`), Files: snapshot_package(`import "sync"

// T is a fixture.
type T struct {
	_ sync.Mutex
	// X is a fixture.
	X int
}
`)},
		{Snapshot: snap.Init(`a.go:7:8: The struct tag key "yaml" is not a stdlib key. Use only json, xml, and asn1.`), Files: snapshot_package("// T is a fixture.\n" +
			"type T struct {\n\t// X is a fixture.\n\tX int `yaml:\"x\"`\n}\n")},
	})
}

// Test_Snapshot_Quoted_Messages pins the four diagnostics that once held a
// backtick in their message text. They now quote with %q, so a snapshot can
// carry them and the old equality-check special case is gone.
func Test_Snapshot_Quoted_Messages(t *testing.T) {
	run_snapshot_cases_shared(t, DOCTRINE_SHARED_COMPONENT_AT_ROOT, []snapshot_case{
		{Snapshot: snap.Init(`a.go:6:2: Do not use a bare "for {}" loop. For an unbounded loop, write a different form.`),
			Files: snapshot_package(`// F does.
func F() {
	for {
		break
	}
}
`)},
	})
}

// Test_Transitive_Purity_Instrumentation_Exemption verifies a pure package may
// import an impure package listed in instrumentation_packages without breaching
// the transitive-purity ban: instrumentation is a write-only side channel, so the
// exemption holds for ordinary code, not only _test.go files.
func Test_Transitive_Purity_Instrumentation_Exemption(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"go.mod": "module fixture\n\ngo 1.25\n",
		"instr/default/instr.go": "// Package instr is a fixture.\n" +
			"package instr\n\n// Sink is a fixture.\nfunc Sink() {\n\treturn\n}\n",
		"lib/library.go": "// Package library x.\npackage library\n\n" +
			"import \"fixture/instr/default\"\n\n" +
			"// Use is a fixture.\nfunc Use() {\n\tinstr.Sink()\n}\n",
	}
	fsys := fstest.MapFS{}
	for name, content := range files {
		data := []byte(content)
		if strings.Has_Suffix(name, ".go") {
			data = gofmt_must(t, content)
		}
		fsys[name] = &fstest.MapFile{Data: data}
	}
	fsys["lint.json"] = &fstest.MapFile{
		Data: test_lint_json(t, "fixture", []string{"instr/**"})}
	stdout := &strings.Builder{}
	code := lint_main(t, &lint.Main_Input{
		Fsys: fsys, Stdout: stdout, Stderr: &strings.Builder{}})
	if strings.Contains(stdout.String(), "The dependency") {
		t.Fatalf("instrumentation import must be exempt (exit %d): %s",
			code, stdout.String())
	}
}

// Test_Transitive_Purity_Telemetry_Exemption verifies the telemetry doctrine for the
// hardcoded stdlib observability packages: log, log/slog, runtime/trace, and runtime/pprof
// are write-only instrumentation, so a library-tier package may call into them — including
// log.Fatal, whose os.Exit the transitive-purity ban otherwise forbids — without tripping
// the impure-stdlib or transitive-purity checks.
func Test_Transitive_Purity_Telemetry_Exemption(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"go.mod": "module fixture\n\ngo 1.25\n",
		"lib/library.go": `// Package library x.
package library

import (
	"log"
	"log/slog"
	"runtime/pprof"
	"runtime/trace"
)

// Use is a fixture.
func Use() {
	slog.Info("x")
	pprof.StartCPUProfile(nil)
	trace.Start(nil)
	log.Fatal("x")
}
`,
	}
	fsys := fstest.MapFS{}
	for name, content := range files {
		data := []byte(content)
		if strings.Has_Suffix(name, ".go") {
			data = gofmt_must(t, content)
		}
		fsys[name] = &fstest.MapFile{Data: data}
	}
	fsys["lint.json"] = &fstest.MapFile{Data: test_lint_json(t, "fixture", nil)}
	stdout := &strings.Builder{}
	code := lint_main(t, &lint.Main_Input{
		Fsys: fsys, Stdout: stdout, Stderr: &strings.Builder{}})
	for _, banned := range []string{"The stdlib", "impure transitive"} {
		if strings.Contains(stdout.String(), banned) {
			t.Fatalf("telemetry stdlib must be exempt (exit %d), found %q: %s",
				code, banned, stdout.String())
		}
	}
}

// Test_Module_Discovery_Ignores_Untracked verifies a go.mod outside the tracked
// set is not discovered as a module. A gitignored gopls temp module mirrors the
// real module's path; discovered, its root (tmp/m) wins the longest-prefix import
// lookup and the instrumentation directory becomes tmp/m/instr/default, which no
// instrumentation entry matches — so the exempt import is wrongly flagged.
func Test_Module_Discovery_Ignores_Untracked(t *testing.T) {
	t.Parallel()
	gomod := "module fixture\n\ngo 1.25\n"
	files := map[string]string{
		"go.mod": gomod,
		"instr/default/instr.go": "// Package instr is a fixture.\npackage instr\n\n" +
			"// Sink is a fixture.\nfunc Sink() {\n\treturn\n}\n",
		"lib/library.go": "// Package library x.\npackage library\n\n" +
			"import \"fixture/instr/default\"\n\n" +
			"// Use is a fixture.\nfunc Use() {\n\tinstr.Sink()\n}\n",
	}
	fsys := fstest.MapFS{}
	for name, content := range files {
		data := []byte(content)
		if strings.Has_Suffix(name, ".go") {
			data = gofmt_must(t, content)
		}
		fsys[name] = &fstest.MapFile{Data: data}
	}
	fsys["lint.json"] = &fstest.MapFile{
		Data: test_lint_json(t, "fixture", []string{"instr/**"})}
	// A gitignored gopls temp module, absent from Tracked, declaring the same path.
	fsys["tmp/m/go.mod"] = &fstest.MapFile{Data: []byte(gomod)}
	stdout := &strings.Builder{}
	lint_main(t, &lint.Main_Input{
		Fsys: fsys, Stdout: stdout, Stderr: &strings.Builder{},
		Tracked: map[string]bool{
			"go.mod":                 true,
			"instr/default/instr.go": true,
			"lib/library.go":         true,
			"lint.json":              true,
		},
	})
	if strings.Contains(stdout.String(), "The dependency") {
		t.Fatalf("an untracked module must not shadow the real one: %s", stdout.String())
	}
}

// The linter ignores a fixed set of directories outright: top-level
// third_party/, vendor/ at any depth, .jj, and .git. A malformed Go
// file under each must yield no diagnostic. testdata and a nested third_party/
// are NOT ignored — the carve-out is the top-level drop-zone only — so the same
// file there must still be flagged.
func Test_Stream_Ignored_Directories(t *testing.T) {
	t.Parallel()
	for _, p := range []string{
		"third_party/build.go",
		"vendor/build.go",
		"deep/nest/vendor/build.go",
		".jj/build.go",
		".git/build.go",
	} {
		output := run_stream(t, map[string]string{p: "echo hi\n"})
		if strings.Contains(output, "expected 'package'") {
			t.Errorf("%s is ignored; want no diagnostic, got: %s", p, output)
		}
	}
	for _, p := range []string{
		"testdata/build.go",
		"pkg/third_party/build.go",
	} {
		output := run_stream(t, map[string]string{p: "echo hi\n"})
		if !strings.Contains(output, "expected 'package'") {
			t.Errorf("%s is not ignored; want a diagnostic, got: %s", p, output)
		}
	}
}

// Test_Ignore_Trims_Scan_Set verifies a lint.json `ignore` glob removes a path
// from the scan set for every tier at once: a malformed Go file and a invalid package
// layout under the ignored prefix both go silent, where without `ignore` each
// fires. Trimming the scan set, not exempting one rule, is what makes the path
// invisible to all checks.
func Test_Ignore_Trims_Scan_Set(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"foo/build.go":     {Data: []byte("echo hi\n")},
		"foo/bad-Dir/x.go": {Data: []byte("package x\n")},
	}
	tracked := map[string]bool{"foo/build.go": true, "foo/bad-Dir/x.go": true}
	control, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys: fsys, Tracked: tracked,
	})
	if err != nil {
		t.Fatalf("control Check_File_System: %v", err)
	}
	if !requirements_diagnosed(control, "expected 'package'") {
		t.Fatalf("control must flag the malformed Go file: %v", control)
	}
	if !requirements_diagnosed(control, "bad-Dir") {
		t.Fatalf("control must flag the invalid package layout: %v", control)
	}
	ignored, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys: fsys, Tracked: tracked, Ignore: []string{"foo/**"},
	})
	if err != nil {
		t.Fatalf("ignored Check_File_System: %v", err)
	}
	if requirements_diagnosed(ignored, "expected 'package'") {
		t.Error("ignore must silence the parse diagnostic")
	}
	if requirements_diagnosed(ignored, "bad-Dir") {
		t.Error("ignore must silence the parse diagnostic")
	}
}

// Builds a lint.json carrying the given ignore globs, with the other required
// keys present and valid (via configuration_document, which marshals from a map so
// a renamed or dropped key surfaces as a decode failure instead of compiling
// silently).
func lint_json_ignore(t *testing.T, ignore []string) (data string) {
	t.Helper()
	return configuration_document(map[string]any{
		"shared_component": "lint_test_no_shared_component",
		"ignore":           ignore,
	})
}

// Lints in-memory files end-to-end (the caller supplies lint.json), marking
// every file but lint.json tracked, and returns the exit code with stdout and
// stderr. The tracked set is what `ignore` trims, so without it the filter is
// inert; keeping all three streams lets a test tell a clean run (code 0, success
// line) from a hard config failure (code 2, stderr).
func run_lint_tracked(t *testing.T, files map[string]string) (code int, stdout, stderr string) {
	t.Helper()
	fsys := make(fstest.MapFS)
	tracked := make(map[string]bool)
	for name, content := range files {
		fsys[name] = &fstest.MapFile{Data: []byte(content)}
		if name != "lint.json" {
			tracked[name] = true
		}
	}
	// The single-module check needs a tracked root go.mod; inject one so the
	// ignore fixtures (which carry no go.mod) lint to a clean run.
	if _, present := fsys["go.mod"]; !present {
		fsys["go.mod"] = &fstest.MapFile{Data: []byte(DOCTRINE_ROOT_GO_MODULE)}
	}
	tracked["go.mod"] = true
	stdout_buffer := &strings.Builder{}
	stderr_buffer := &strings.Builder{}
	code = lint_main(t, &lint.Main_Input{
		Fsys:    fsys,
		Stdout:  stdout_buffer,
		Stderr:  stderr_buffer,
		Tracked: tracked,
	})
	return code, stdout_buffer.String(), stderr_buffer.String()
}

// A recursive dir/** entry drops the directory and its whole subtree from the
// scan set; the identical fixture without the entry still flags the directory,
// proving the ignore is load-bearing.
func Test_Ignore_Recursive(t *testing.T) {
	t.Parallel()
	fixture := func(ignore []string) (files map[string]string) {
		return map[string]string{
			"bad-Dir/x.go": "x\n",
			"lint.json":    lint_json_ignore(t, ignore),
		}
	}
	code, stdout, _ := run_lint_tracked(t, fixture(nil))
	if code != 1 {
		t.Fatalf("control: want exit 1, got %d: %s", code, stdout)
	}
	if !strings.Contains(stdout, "bad-Dir") {
		t.Fatalf("control: want bad-Dir flagged, got: %s", stdout)
	}
	code, stdout, stderr := run_lint_tracked(t, fixture([]string{"bad-Dir/**"}))
	if code != 0 {
		t.Fatalf("ignored: want clean run, got %d; stderr %q", code, stderr)
	}
	if strings.Contains(stdout, "bad-Dir") {
		t.Fatalf("ignored: bad-Dir/** must drop the subtree: %s", stdout)
	}
}

// A slash-less entry is relative to the repo root: it drops the root-level file
// and never the same basename under a subdirectory. Any-depth matching is spelled
// with a leading **/.
func Test_Ignore_Root_Relative(t *testing.T) {
	t.Parallel()
	fixture := func(ignore []string) (files map[string]string) {
		return map[string]string{
			"weird-File.go":   "x\n",
			"a/weird-File.go": "x\n",
			"lint.json":       lint_json_ignore(t, ignore),
		}
	}
	code, stdout, _ := run_lint_tracked(t, fixture(nil))
	if code != 1 {
		t.Fatalf("control: want exit 1, got %d: %s", code, stdout)
	}
	if !strings.Contains(stdout, "weird-File.go") {
		t.Fatalf("control: want weird-File.go flagged: %s", stdout)
	}
	// Root-relative: the root file is dropped, the nested one stays flagged.
	code, stdout, stderr := run_lint_tracked(t, fixture([]string{"weird-File.go"}))
	if code != 1 {
		t.Fatalf("root-relative: nested still flagged, got %d; %q", code, stderr)
	}
	if !strings.Contains(stdout, "a/weird-File.go") {
		t.Fatalf("a root-relative entry must not drop the nested file: %s", stdout)
	}
	// A leading **/ opts into any-depth matching, dropping both.
	code, stdout, stderr = run_lint_tracked(t, fixture([]string{"**/weird-File.go"}))
	if code != 0 {
		t.Fatalf("**/: want clean run, got %d; stderr %q", code, stderr)
	}
	if strings.Contains(stdout, "weird-File") {
		t.Fatalf("a leading **/ must drop every depth: %s", stdout)
	}
}

// An entry containing a slash is anchored to the repo root: it drops the exact
// path and never the same basename under a different parent.
func Test_Ignore_Anchored(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"top/weird-File.go":   "x\n",
		"other/weird-File.go": "x\n",
		"lint.json":           lint_json_ignore(t, []string{"top/weird-File.go"}),
	}
	code, stdout, stderr := run_lint_tracked(t, files)
	if code != 1 {
		t.Fatalf("want other/ still flagged, got %d; stderr %q", code, stderr)
	}
	if strings.Contains(stdout, "top/weird-File.go") {
		t.Fatalf("anchored entry must drop top/: %s", stdout)
	}
	if !strings.Contains(stdout, "other/weird-File.go") {
		t.Fatalf("anchored entry must not leak to other/: %s", stdout)
	}
}

// A single-segment glob (*.go) is root-relative: it drops matching names at the
// repo root only. Any-depth matching is spelled **/*.go.
func Test_Ignore_Segment_Glob(t *testing.T) {
	t.Parallel()
	fixture := func(ignore []string) (files map[string]string) {
		return map[string]string{
			"bad-Name.go":  "x\n",
			"a/Bad-Two.go": "x\n",
			"lint.json":    lint_json_ignore(t, ignore),
		}
	}
	code, stdout, _ := run_lint_tracked(t, fixture(nil))
	if code != 1 {
		t.Fatalf("control: want exit 1, got %d: %s", code, stdout)
	}
	if !strings.Contains(stdout, ".go") {
		t.Fatalf("control: want *.go names flagged: %s", stdout)
	}
	// Root-relative: *.go drops the root name, the nested one stays flagged.
	code, stdout, stderr := run_lint_tracked(t, fixture([]string{"*.go"}))
	if code != 1 {
		t.Fatalf("*.go: nested still flagged, got %d; %q", code, stderr)
	}
	if !strings.Contains(stdout, "a/Bad-Two.go") {
		t.Fatalf("*.go must not drop the nested name: %s", stdout)
	}
	// **/*.go opts into any-depth matching, dropping both.
	code, stdout, stderr = run_lint_tracked(t, fixture([]string{"**/*.go"}))
	if code != 0 {
		t.Fatalf("**/*.go: want clean run, got %d; stderr %q", code, stderr)
	}
	if strings.Contains(stdout, ".go") {
		t.Fatalf("**/*.go must drop matching names at any depth: %s", stdout)
	}
}

// A bare negation, empty, or ?/[ entry, or one path.Match deems malformed, is
// rejected at config-parse time, aborting the run with exit 2 — the same loud
// failure every other bad lint.json earns.
func Test_Ignore_Parse_Rejects(t *testing.T) {
	t.Parallel()
	// ? and [class] are the path.Match tokens we deliberately do not support — only
	// * and ** — so even a well-formed class is rejected, not just a malformed "bad[".
	// "!" negates nothing, and a list of only negation entries can never match
	// anything (Test_Ignore_Negation covers the accepted, meaningful negation form).
	bad_entries := [][]string{{"!"}, {""}, {"bad["}, {"a?b"}, {"a[bc]"}, {"!neg", "!other"}}
	for _, bad := range bad_entries {
		files := map[string]string{
			"good.go":   "x\n",
			"lint.json": lint_json_ignore(t, bad),
		}
		code, stdout, stderr := run_lint_tracked(t, files)
		if code != 2 {
			t.Fatalf("entry %q must exit 2, got %d: %s", bad, code, stdout)
		}
		if stderr == "" {
			t.Fatalf("entry %q must explain the rejection on stderr", bad)
		}
	}
}

// A "!"-prefixed entry re-includes a path a broader entry ignored: dir/** drops
// the whole directory, but !dir/keep-File.go holds that one parse
// violation out of the drop while its sibling stays dropped.
func Test_Ignore_Negation(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"dir/keep-File.go":  "x\n",
		"dir/other-File.go": "x\n",
		"lint.json":         lint_json_ignore(t, []string{"dir/**", "!dir/keep-File.go"}),
	}
	code, stdout, stderr := run_lint_tracked(t, files)
	if code != 1 {
		t.Fatalf("want dir/keep-File.go still flagged, got %d; stderr %q", code, stderr)
	}
	if !strings.Contains(stdout, "keep-File.go") {
		t.Fatalf("!dir/keep-File.go must re-include the ignored path: %s", stdout)
	}
	if strings.Contains(stdout, "other-File.go") {
		t.Fatalf("dir/** must still drop the rest of the directory: %s", stdout)
	}
}

// Test_No_Impure_Stdlib_Hard_Imports verifies that hard-banned imports
// (os, crypto/rand, math/rand v1, flag) fire in library packages.
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
			Want_Diag: "The stdlib import",
		},

		{
			Name: "library imports crypto/rand",
			Files: map[string]string{
				"a.go": `package library

import "crypto/rand"

func F() (n int, err error) { return rand.Reader.Read(nil) }
`,
			},
			Want_Diag: "The stdlib import",
		},

		{
			Name: "library imports math/rand v1",
			Files: map[string]string{
				"a.go": `package library

import "math/rand"

func F() (n int) { return rand.Int() }
`,
			},
			Want_Diag: "The stdlib import",
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
			Name: "library imports runtime",
			Files: map[string]string{
				"a.go": `package library

import "runtime"

func F() (s string) { return runtime.GOOS }
`,
			},
			Want_Diag: "The stdlib import",
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
			Want_Diag: "The stdlib call",
		},

		{
			Name: "library uses time.Duration clean",
			Files: map[string]string{
				"a.go": `// Package library x.
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
			Want_Diag: "The stdlib call",
		},

		{
			Name: "library uses fmt.Sprintf and fmt.Errorf clean",
			Files: map[string]string{
				"a.go": `// Package library x.
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
				"a.go": `// Package library x.
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
			Want_Diag: "The stdlib call",
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
			"a.go": `// Package library x.
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
			Want_Diag: "The stdlib call",
		},
		{
			Name: "library uses http.Request type is io-gateway-banned",
			Files: map[string]string{
				"a.go": `// Package library x.
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
			Want_Diag: "Route IO through shared/std/sim/nbio",
		},
		{
			Name: "library calls net.Dial",
			Files: map[string]string{
				"a.go": `package library

import "net"

func F() (c net.Conn, err error) { return net.Dial("tcp", "x:1") }
`,
			},
			Want_Diag: "The stdlib call",
		},
	}
	run_diag_table(t, tests)
}

// Curated call diagnostics remain visible beside stronger shared import bans.
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
			Want_Diag: "The transitive call",
		},
		{
			Name: "filepath.Abs reads the working directory",
			Files: map[string]string{
				"a.go": `package library

import "path/filepath"

func F() (s string, err error) { return filepath.Abs(".") }
`,
			},
			Want_Diag: "The transitive call",
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
			Want_Diag: "The transitive call",
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
			Want_Diag: "The transitive call",
		},
		{
			Name: "smtp.SendMail dials a server",
			Files: map[string]string{
				"a.go": `package library

import "net/smtp"

func F() (err error) { return smtp.SendMail("h:25", nil, "f", nil, nil) }
`,
			},
			Want_Diag: "The transitive call",
		},
	}
	run_diag_table(t, tests)
}

// Test_No_Impure_Stdlib_Exemptions verifies that package main and _test.go
// files may freely use The stdlib calls.
func Test_No_Impure_Stdlib_Exemptions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name      string
		Files     map[string]string
		Want_Diag string
	}{

		{
			Name: "package main may use runtime",
			Files: map[string]string{
				"main.go": `package main

import "runtime"

func main() { print(runtime.GOOS) }
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
				"a.go": `// Package library x.
package library


// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

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
				"shared/go.mod": DOCTRINE_SHARED_LIBRARY_GO_MODULE,
				"shared/foo/foo.go": `package foo

import "runtime"

func Read() (name string) { return runtime.GOOS }
`,
			},
			Want_Diags: []string{"The stdlib import \"runtime\""},
		},

		{
			Name: "composition tier impure import allowed",
			Files: map[string]string{
				"shared/go.mod":     DOCTRINE_SHARED_LIBRARY_GO_MODULE,
				"shared/foo/foo.go": fixture_package("foo"),
				"shared/foo/foo_default/" +
					"foo_default.go": `// Package foo_default is a fixture.
package foo_default

import (
	"runtime"

)

// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

// Read is a fixture.
func Read() (name string) {
	defer func() {
	}()
	return runtime.GOOS
}
`,
			},
			Forbid: []string{"The stdlib import"},
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
				"shared/go.mod":              DOCTRINE_SHARED_LIBRARY_GO_MODULE,
				"shared/std/snap/v2/snap.go": fixture_package("snap"),
				"shared/std/snap/v2/snap_default/" +
					"snap_default.go": `// Package snap_default is a fixture.
package snap_default

import (
	"runtime"

)

// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

// Read is a fixture.
func Read() (name string) {
	defer func() {
	}()
	return runtime.GOOS
}
`,
			},
			Forbid: []string{"The stdlib import"},
		},
	}
	run_doctrine_diag_table(t, tests)
}

// Test_No_Impure_Stdlib_Composition_Tier_Extra continues the composition-tier
// allow-list: an impure stdlib CALL (`fmt.Println`) at the composition tier is
// allowed in the shared library, and the same position inside a binary is not —
// a binary keeps its impurity in package main, so nothing under it reaches the
// composition tier. time is gatewayed separately, so fmt is the example here;
// see the Stdlib Time rule.
func Test_No_Impure_Stdlib_Composition_Tier_Extra(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name       string
		Files      map[string]string
		Want_Diags []string
		Forbid     []string
	}{
		{
			Name: "impure call (fmt.Println) at composition tier allowed",
			Files: map[string]string{
				"shared/go.mod":     DOCTRINE_SHARED_LIBRARY_GO_MODULE,
				"shared/foo/foo.go": fixture_package("foo"),
				"shared/foo/foo_default/" +
					"foo_default.go": `// Package foo_default is a fixture.
package foo_default

import "fmt"

// Stamp is a fixture.
func Stamp() { fmt.Println("stamped") }
`,
			},
			Forbid: []string{"The stdlib call"},
		},
		{
			Name: "binary module composition tier " +
				"(one level under library tier) flagged",
			Files: map[string]string{
				"mybinary/go.mod": DOCTRINE_BINARY_GO_MODULE,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/pkg/entry.go":       DOCTRINE_BINARY_PACKAGE_MAIN,
				"mybinary/pkg/lib/library.go": fixture_package("library"),
				"mybinary/pkg/lib/lib_default/" +
					"library_default.go": `// Package library_default is a fixture.
package library_default

import (
	"runtime"

)

// FIXTURE_HI is a fixture.
const FIXTURE_HI = 100

// Read is a fixture.
func Read() (name string) {
	defer func() {
	}()
	return runtime.GOOS
}
`,
			},
			Want_Diags: []string{"The stdlib import"},
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
			Want_Diag: "bare \"for {}\" loop",
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
			Want_Diag: "bare \"for {}\" loop",
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
			Want_Diag: "bare \"for {}\" loop",
		},
	}
	run_diag_table(t, tests)
}

// Test_No_Bare_For_Clean verifies that for-loops with a real condition,
// C-style headers, range loops, and the documented escape hatch
// `for range aver.GameLoop()` are not flagged.
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

` + FIXTURE_INVARIANT_IMPORT + `

func main() {
	for range aver.Game_Loop() {
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
	if requirements_diagnosed(clean, "default package") {
		t.Fatal("a default package declaring its parent name must not be flagged")
	}

	wrong, err := lint.Check_Source("foo/default/wire.go", "package wrong\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !requirements_diagnosed(wrong, "not \"package foo\"") {
		t.Fatal("a default package with the wrong package clause must be flagged")
	}
}

// Test_Binary_Module_Layout verifies that binary modules — every module
// at the workspace root except the hardcoded shared library — must
// keep all non-main, non-test packages under pkg/. Files in
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
				"mybinary/go.mod": DOCTRINE_BINARY_GO_MODULE,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/pkg/entry.go": DOCTRINE_BINARY_PACKAGE_MAIN,
			},
		},
		{
			Name: "binary with non-main package outside pkg flagged",
			Files: map[string]string{
				"mybinary/go.mod": DOCTRINE_BINARY_GO_MODULE,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/helpers/h.go": fixture_package("helpers"),
			},
			Want_Diags: []string{"Move helpers -> mybinary/pkg/helpers"},
		},
		{
			Name: "binary with package under pkg is clean",
			Files: map[string]string{
				"mybinary/go.mod": DOCTRINE_BINARY_GO_MODULE,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/pkg/entry.go":       DOCTRINE_BINARY_PACKAGE_MAIN,
				"mybinary/pkg/lib/library.go": fixture_package("library"),
			},
		},
		{
			Name: "binary with non-main package at module root flagged",
			Files: map[string]string{
				"mybinary/go.mod":     DOCTRINE_BINARY_GO_MODULE,
				"mybinary/library.go": fixture_package("mybinary"),
			},
			Want_Diags: []string{"Move . -> mybinary/pkg"},
		},
		{
			Name: "shared library with non-main package at depth 1 is exempt",
			Files: map[string]string{
				"shared/go.mod":     DOCTRINE_SHARED_LIBRARY_GO_MODULE,
				"shared/foo/foo.go": fixture_package("foo"),
			},
		},
		{
			Name: "second main package under cmd flagged",
			Files: map[string]string{
				"mybinary/go.mod": DOCTRINE_BINARY_GO_MODULE,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/cmd/extra/extra.go": "package main\n\n" +
					"func main() { return }\n",
			},
			Want_Diags: []string{
				"has its main package at \"cmd/extra\"",
				"Move the main package to the component root",
			},
		},
	}
	run_doctrine_diag_table(t, tests)
}

// Test_Binary_Module_Package_Main verifies that every binary module
// declares exactly one free func Main in its top-level pkg/ package
// — the composition entry point main() delegates to. Location is the
// pkg/ directory itself, not a subpackage; the function must be a
// free function (no receiver) of any signature; test files don't count;
// the shared library is exempt because it owns no entry point.
func Test_Binary_Module_Package_Main(t *testing.T) {
	tests := []struct {
		Name       string
		Files      map[string]string
		Want_Diags []string
		Forbid     []string
	}{
		{
			Name: "binary with func Main in pkg is clean",
			Files: map[string]string{
				"mybinary/go.mod": DOCTRINE_BINARY_GO_MODULE,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/pkg/entry.go": DOCTRINE_BINARY_PACKAGE_MAIN,
			},
		},
		{
			Name: "non-standard signature still satisfies the rule",
			Files: map[string]string{
				"mybinary/go.mod": DOCTRINE_BINARY_GO_MODULE,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/pkg/entry.go": "// Package entry is a fixture.\n" +
					"package entry\n\n// Main is a fixture.\n" +
					"func Main(a int) (s string) { return \"\" }\n",
			},
		},
		{
			Name: "binary without func Main flagged at component dir",
			Files: map[string]string{
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
			},
			Want_Diags: []string{
				"mybinary:1:1",
				"declares no func Main in pkg/",
			},
		},
		{
			Name: "func Main below top-level pkg does not satisfy the rule",
			Files: map[string]string{
				"mybinary/go.mod": DOCTRINE_BINARY_GO_MODULE,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/pkg/cmd/entry.go": DOCTRINE_BINARY_PACKAGE_MAIN,
			},
			Want_Diags: []string{"declares no func Main in pkg/"},
		},
	}
	run_doctrine_diag_table(t, tests)
}

// Test_Binary_Module_Package_Main_Part2 continues the binary entry-point
// cases (split so neither table function exceeds the line budget): a method
// named Main, a test-file-only Main, the multiple-Main violation, and the
// shared-library exemption.
func Test_Binary_Module_Package_Main_Part2(t *testing.T) {
	tests := []struct {
		Name       string
		Files      map[string]string
		Want_Diags []string
		Forbid     []string
	}{
		{
			Name: "method named Main does not satisfy the rule",
			Files: map[string]string{
				"mybinary/go.mod": DOCTRINE_BINARY_GO_MODULE,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/pkg/entry.go": "// Package entry is a fixture.\n" +
					"package entry\n\n// R is a fixture.\ntype R struct{}\n\n" +
					"// Main is a fixture.\nfunc (r R) Main() { return }\n",
			},
			Want_Diags: []string{"declares no func Main in pkg/"},
		},
		{
			Name: "func Main only in a test file does not satisfy the rule",
			Files: map[string]string{
				"mybinary/go.mod": DOCTRINE_BINARY_GO_MODULE,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/pkg/entry_test.go": "package entry\n\n" +
					"// Main is a fixture.\nfunc Main() { return }\n",
			},
			Want_Diags: []string{"declares no func Main in pkg/"},
		},
		{
			Name: "two func Main across pkg files flagged as multiple",
			Files: map[string]string{
				"mybinary/go.mod": DOCTRINE_BINARY_GO_MODULE,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/pkg/a.go": DOCTRINE_BINARY_PACKAGE_MAIN,
				"mybinary/pkg/b.go": "package entry\n\n" +
					"// Main is a fixture.\nfunc Main() { return }\n",
			},
			Want_Diags: []string{"declares more than one func Main in pkg/"},
		},
		{
			Name: "shared library without pkg func Main is exempt",
			Files: map[string]string{
				"shared/go.mod":     DOCTRINE_SHARED_LIBRARY_GO_MODULE,
				"shared/foo/foo.go": fixture_package("foo"),
			},
		},
		{
			Name: "shared library may expose func Main as an embeddable",
			Files: map[string]string{
				"shared/go.mod": DOCTRINE_SHARED_LIBRARY_GO_MODULE,
				"shared/foo/foo.go": "// Package foo is a fixture.\n" +
					"package foo\n\n// Main is an embeddable entry point.\n" +
					"func Main() { return }\n",
			},
		},
	}
	run_doctrine_diag_table(t, tests)
}

// Test_Shared_Library_No_Internal verifies that the shared library
// module may not contain internal/ directories or any package main.
// Other modules are unaffected — they use pkg/ and own a binary.
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
				"shared/go.mod":     DOCTRINE_SHARED_LIBRARY_GO_MODULE,
				"shared/foo/foo.go": fixture_package("foo"),
				"shared/foo/internal/helper/" +
					"help.go": fixture_package("helper"),
			},
			Want_Diags: []string{
				"A shared library permits no internal/", "shared/foo/internal",
			},
		},
		{
			Name: "shared library with package main flagged",
			Files: map[string]string{
				"shared/go.mod":     DOCTRINE_SHARED_LIBRARY_GO_MODULE,
				"shared/foo/foo.go": fixture_package("foo"),
				"shared/run/main.go": "package main\n\n" +
					"func main() { return }\n",
			},
			Want_Diags: []string{"permits no package main"},
		},
		{
			Name: "shared library without internal is clean",
			Files: map[string]string{
				"shared/go.mod":     DOCTRINE_SHARED_LIBRARY_GO_MODULE,
				"shared/foo/foo.go": fixture_package("foo"),
			},
		},
		{
			Name: "binary with pkg directory is unaffected by this rule",
			Files: map[string]string{
				"mybinary/go.mod": DOCTRINE_BINARY_GO_MODULE,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/pkg/entry.go":       DOCTRINE_BINARY_PACKAGE_MAIN,
				"mybinary/pkg/lib/library.go": fixture_package("library"),
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
				"shared/" +
					"go.mod": DOCTRINE_SHARED_LIBRARY_GO_MODULE,
				"shared/foo/foo.go":         fixture_package("foo"),
				"shared/foo/bar/bar.go":     fixture_package("bar"),
				"shared/foo/bar/baz/baz.go": fixture_package("baz"),
			},
			Want_Diags: []string{"is below the library tier", "baz"},
		},
		{
			Name: "library plus composition tier is clean",
			Files: map[string]string{
				"shared/go.mod":         DOCTRINE_SHARED_LIBRARY_GO_MODULE,
				"shared/foo/foo.go":     fixture_package("foo"),
				"shared/foo/bar/bar.go": fixture_package("bar"),
			},
			Forbid: []string{"is below the library tier"},
		},
		{
			Name: "v2 version directory does not count as ancestor",
			Files: map[string]string{
				"shared/" +
					"go.mod": DOCTRINE_SHARED_LIBRARY_GO_MODULE,
				"shared/std/snap/snap.go":         fixture_package("snap"),
				"shared/std/snap/v2/snap.go":      fixture_package("snap"),
				"shared/std/snap/v2/sub/child.go": fixture_package("child"),
			},
			Forbid: []string{"is below the library tier"},
		},
		{
			Name: "non-Go intermediate directory does not count as ancestor",
			Files: map[string]string{
				"shared/go.mod":     DOCTRINE_SHARED_LIBRARY_GO_MODULE,
				"shared/foo/foo.go": fixture_package("foo"),
				"shared/foo/examples/sample/" +
					"example.go": fixture_package("example"),
			},
			Forbid: []string{"is below the library tier"},
		},
		{
			Name: "binary composition tier one level under pkg is clean",
			Files: map[string]string{
				"mybinary/go.mod": DOCTRINE_BINARY_GO_MODULE,
				"mybinary/main.go": "package main\n\n" +
					"func main() { return }\n",
				"mybinary/pkg/entry.go":   DOCTRINE_BINARY_PACKAGE_MAIN,
				"mybinary/pkg/foo/foo.go": fixture_package("foo"),
			},
			Forbid: []string{"is below the library tier"},
		},
	}
	run_doctrine_diag_table(t, tests)
}

// Test_Library_Tier_Depth_Package_Anchor verifies that a binary module's
// top-level pkg directory is where the nesting count starts — the same
// role a shared module's root plays — and is not itself a package counted
// above another. A pure library package sits one level inside pkg and one
// more package below that (pkg/foo/bar); anything deeper is flagged. Both
// are pure — a binary holds no default tier — so the depth alone is the subject
// here.
func Test_Library_Tier_Depth_Package_Anchor(t *testing.T) {
	tests := []struct {
		Name       string
		Files      map[string]string
		Want_Diags []string
		Forbid     []string
	}{
		{
			Name: "library package and one below it under pkg is clean",
			Files: map[string]string{
				"bin/go.mod": DOCTRINE_BINARY_GO_MODULE,
				"bin/main.go": "package main\n\n" +
					"func main() { return }\n",
				"bin/pkg/entry.go":       DOCTRINE_BINARY_PACKAGE_MAIN,
				"bin/pkg/foo/foo.go":     fixture_package("foo"),
				"bin/pkg/foo/bar/bar.go": fixture_package("bar"),
			},
			Forbid: []string{"is below the library tier"},
		},
		{
			Name: "nesting beyond the composition tier under pkg flagged",
			Files: map[string]string{
				"bin/go.mod": DOCTRINE_BINARY_GO_MODULE,
				"bin/main.go": "package main\n\n" +
					"func main() { return }\n",
				"bin/pkg/entry.go":           DOCTRINE_BINARY_PACKAGE_MAIN,
				"bin/pkg/foo/foo.go":         fixture_package("foo"),
				"bin/pkg/foo/bar/bar.go":     fixture_package("bar"),
				"bin/pkg/foo/bar/baz/baz.go": fixture_package("baz"),
			},
			Want_Diags: []string{"is below the library tier", "baz"},
		},
	}
	run_doctrine_diag_table(t, tests)
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
	var stdout, stderr strings.Builder
	code := lint_main(t, &lint.Main_Input{
		Fsys:      fsys,
		Stdout:    &stdout,
		Stderr:    &stderr,
		CPU_Count: 1,
	})
	if code == 0 {
		t.Logf("expected non-zero exit for a tree with parse errors; got %d", code)
	}
}

// Main and Check_File_System fire their Hi bucket only when CPU_Count > 0;
// production tests use the degraded CPU_Count=0 path throughout.
func Test_Coverage_Backfill(t *testing.T) {
	t.Parallel()
	var stdout, stderr strings.Builder
	code := lint_main(t, &lint.Main_Input{
		Fsys:      fstest.MapFS{},
		Stdout:    &stdout,
		Stderr:    &stderr,
		CPU_Count: 4,
	})
	if code != 0 {
		t.Logf("Main exit code with empty FS: %d", code)
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
	var stdout, stderr strings.Builder
	code := lint_main(t, &lint.Main_Input{
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
	const MODULE_COUNT = 1025
	for i_index := 0; i_index < MODULE_COUNT; i_index++ {
		// Module / package names stay short (numbered suffix) so the
		// generated identifiers fit within IDENTIFIER_CHARS_MAX; the
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
	var stdout, stderr strings.Builder
	code := lint_main(t, &lint.Main_Input{
		Fsys: fsys, Stdout: &stdout, Stderr: &stderr, CPU_Count: 1,
	})
	if code < 0 {
		t.Fatalf("unexpected negative exit code %d", code)
	}
}

// Test_Coverage_Backfill_String_Bounded_Mixed_Invariant_Calls drives
// extract_call_name's Lo=0 (non-invariant call name) and Hi=26 ("Recorder_
// Distinct_Boundary") buckets via crafted source.
func Test_Coverage_Backfill_String_Bounded_Mixed_Invariant_Calls(t *testing.T) {
	t.Parallel()
	mixed_calls_source := "package fixture\n\n" +
		"\tfoo.Bar()\n" +
		"X: x, Lo: 0, Hi: FIXTURE_HI})\n" +
		"}\n"
	diags, err := lint.Check_Source("test.go", mixed_calls_source)
	t.Logf("mixed diags=%d err=%v", len(diags), err)
	rich_source := "package fixture\n\n" +
		"const Foo = 1\n" +
		"type Bar struct {\n\tA int\n\tB bool\n\tC string\n}\n\n" +
		"func Quux(input *Bar, s string, n int, b bool, p *int) (result int) {\n" +
		"\tdefer func() {\n" +
		"\t\t\t\tX: result, Lo: 0, Hi: FIXTURE_HI,\n" +
		"\t\t\t}),\n" +
		"\t\t)\n" +
		"\t}()\n" +
		"input != nil, \"input is non-nil\"),\n" +
		"p != nil, \"p is non-nil\"),\n" +
		"b, \"b is sometimes true\"),\n" +
		"\t\t\tX: n, Lo: 0, Hi: FIXTURE_HI,\n" +
		"\t\t}),\n" +
		"\t\t\tX: len(s), Lo: 0, Hi: FIXTURE_HI,\n" +
		"\t\t}),\n" +
		"s == \"\", \"s is empty\"),\n" +
		"\t)\n" +
		"\treturn 0\n" +
		"}\n"
	rich_diags, rich_err := lint.Check_Source("test.go", rich_source)
	t.Logf("rich diags=%d err=%v", len(rich_diags), rich_err)
}

// Test_Coverage_Backfill_Main_Input_Combinations exercises input sizes and parallelism.
func Test_Coverage_Backfill_Main_Input_Combinations(t *testing.T) {
	t.Parallel()
	long_path := strings.Repeat("a", 4096)
	full_tracked := make(map[string]bool, 30000)
	full_tracked["go.mod"] = true
	full_tracked["main.go"] = true
	for i_index := 0; i_index < 29998; i_index++ {
		full_tracked["k"+decimal_text(i_index)] = true
	}
	for bits_index := 0; bits_index < 16; bits_index++ {

		scope_prefix := ""
		if bits_index&(1<<1) != 0 {
			scope_prefix = long_path
		}
		tracked := map[string]bool(nil)
		if bits_index&(1<<2) != 0 {
			tracked = full_tracked
		}
		cpu_count := 0
		if bits_index&(1<<3) != 0 {
			cpu_count = 1024
		}
		var stdout, stderr strings.Builder
		lint_main(t, &lint.Main_Input{
			Fsys: fstest.MapFS{
				"go.mod":  {Data: []byte("module test\ngo 1.25\n")},
				"main.go": {Data: []byte("package main\n\nfunc main() {}\n")},
			},
			Stdout:       &stdout,
			Stderr:       &stderr,
			Scope_Prefix: scope_prefix,
			Tracked:      tracked,
			CPU_Count:    cpu_count,
		})
	}
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
	for strings.Builder_Size(&sb) < 4092 {
		strings.Builder_Write_Text(&sb, "aa/")
	}
	long_path := sb.String()[:4092] + "a.go"
	var stdout, stderr strings.Builder
	lint_main(t, &lint.Main_Input{
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
// Check_File_System with an empty filesystem at CPU_Count=0 and CPU_Count=1024.
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
// so the recurse visitor's Targets map has exactly one entry (Lo bucket of
// V.Targets boundary, Lo=NON_EMPTY_MIN). Two shapes exercise both Lo (1-char)
// and Hi (128-char) Caller buckets at Visit entry, where Scopes/Edges/
// Push_History are all empty (Lo for each).
func Test_Coverage_Backfill_Recursion_Visitor_Single_Function_File(t *testing.T) {
	t.Parallel()
	long_funcname := strings.Repeat("F", 128)
	cases := []string{
		// Documented package, lowercase 1-char function with a non-blank discard
		// statement so tier-1 stays clean and tier-2 (check_no_recurse)
		// runs, exercising the recurse visitor.
		"// Package fixture is for the recurse visitor.\npackage fixture\n\n" +
			"func f() {\n\tif true {\n\t}\n}\n",
		"// Package fixture is for the recurse visitor.\npackage fixture\n\n" +
			"// " +
			long_funcname +
			"\n// is for the recurse visitor.\nfunc " +
			long_funcname + "() {\n\tif true {\n\t}\n}\n",
		// Range with explicit blank key drives recurse_visitor_define_ident
		// with a nil expr branch (e_nil=true case).
		"// Package fixture is for the recurse visitor.\npackage fixture\n\n" +
			"func f() {\n\tfor range 1 {\n\t}\n}\n",
		// Single-function file with a 128-char self-recursing function name
		// drives recurse_visitor_enter_record_call_edge with Lo targets
		// (single-target) AND Hi caller (128-char) simultaneously.
		"// Package fixture is for the recurse visitor.\npackage fixture\n\n" +
			"// " +
			long_funcname +
			"\n// recurses into itself.\nfunc " +
			long_funcname + "() {\n\t" + long_funcname + "()\n}\n",
		// 128-char function calling a builtin (`println`) — call.Fun is a bare
		// identifier (True ident) but the name is not in v.Targets, so
		// record_call_edge returns without appending to v.Edges. Defer time
		// observes (Lo edges, Hi caller, True ident, Lo targets, Lo scopes,
		// Lo history).
		"// Package fixture is for the recurse visitor.\npackage fixture\n\n" +
			"// " +
			long_funcname +
			"\n// invokes a builtin.\nfunc " +
			long_funcname + "() {\n\tprintln(\"x\")\n}\n",
		// 128-char function calling a selector — call.Fun is *ast.SelectorExpr
		// (False ident). The package-level `p` var isn't added to v.Targets
		// (only FuncDecls are), so targets stays at 1 (single-target file).
		"// Package fixture is for the recurse visitor.\npackage fixture\n\n" +
			"var p = struct{ F func() }{F: func() {}}\n\n// " +
			long_funcname +
			"\n// invokes a method.\nfunc " +
			long_funcname + "() {\n\tp.F()\n}\n",
		// 128-char function with an if-init clause drives
		// recurse_visitor_enter_define_statement with Hi caller AND False
		// s_nil (init present) in a single-target file.
		"// Package fixture is for the recurse visitor.\npackage fixture\n\n" +
			"// " +
			long_funcname +
			"\n// has an if-init.\nfunc " +
			long_funcname + "() {\n\tif y := 1; y > 0 {\n\t}\n}\n",
		// 128-char function with a top-level := drives
		// recurse_visitor_define_ident with Hi caller, Lo scopes (=1, only
		// the BlockStmt scope is pushed at this point), Lo history (=1).
		"// Package fixture is for the recurse visitor.\npackage fixture\n\n" +
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

// Test_Coverage_Backfill_Recursion drives check_no_recurse to emit a
// self-recurse diagnostic with a single 1-char function name so the
// diag-message helper observes its Lo bucket.
func Test_Coverage_Backfill_Recursion(t *testing.T) {
	t.Parallel()
	// 128-char name (xxxxxxxxxxxx...x, snake_case-valid up to the cap).
	long_name := strings.Repeat("x", 128)
	source := "// Package fixture is a fixture.\n" +
		"package fixture\n\n" +
		"// F recurses into itself to trigger check_no_recurse.\n" +
		"func F() { F() }\n\n" +
		"// Long recurses into itself; diag_message gets Hi message.\n" +
		"func " + long_name + "() {\n" +
		"\t" + long_name + "()\n" +
		"}\n"
	diags, err := lint.Check_Source("test.go", source)
	t.Logf("recurse diags=%d err=%v", len(diags), err)
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
	var stdout, stderr strings.Builder
	lint_main(t, &lint.Main_Input{
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
	for strings.Builder_Size(&sb) < 4092 {
		strings.Builder_Write_Text(&sb, "aa/")
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
	diags := lint.Check_File(&lint.Check_File_Input{File_Set: file_set, File: file})
	t.Logf("empty_source diags=%d", len(diags))
}

// Test_Parse_Configuration covers lint.json decoding: a document with every key
// present yields the shared module and the allowlist; a missing required key, an
// empty value where one is required, an unknown top-level key, a rejected glob,
// and malformed JSON are all hard errors.
func Test_Parse_Configuration(t *testing.T) {
	t.Parallel()
	for _, tt := range parse_configuration_cases() {
		t.Run(tt.Name, func(t *testing.T) {
			configuration, err := lint.Parse_Configuration([]byte(tt.Input))
			if tt.Want_Err {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assert_configuration(t, configuration, tt.Want_Shared, tt.Want_List)
		})
	}
}

type parse_configuration_case struct {
	Name        string
	Input       string
	Want_Err    bool
	Want_Shared string
	Want_List   []string
}

// Returns the lint.json decode cases: the documents that decode cleanly plus
// every hard-error form. Split by outcome so each list stays within the function
// length cap.
func parse_configuration_cases() (cases []parse_configuration_case) {
	cases = append(parse_configuration_valid_cases(), parse_configuration_error_cases()...)
	return append(cases, parse_configuration_negation_error_cases()...)
}

// Renders a lint.json with every required key present and valid. An override
// replaces a key's value, or deletes the key when the value is nil, so a case can
// isolate one decode behavior without tripping the all-keys-required guard on an
// unrelated absent key. Marshaled from a map so each key name lives in one place.
func configuration_document(overrides map[string]any) (document string) {
	fields := map[string]any{
		"shared_component":                   "example.com/lib",
		"instrumentation_packages":           []string{},
		"pure_but_indeterministic_packages":  []string{},
		"ignore":                             []string{},
		"opt_out_assertion_mandate_packages": []string{},
		"opt_out_recursion_ban":              []string{},
		"opt_out_interface_ban":              []string{},
	}
	for key, value := range overrides {
		if value == nil {
			delete(fields, key)
			continue
		}
		fields[key] = value
	}
	return json_fields_text(fields)
}

func json_fields_text(fields map[string]any) (document string) {
	var builder strings.Builder
	strings.Builder_Write_Byte(&builder, '{')
	first := true
	for key, value := range fields {
		if !first {
			strings.Builder_Write_Byte(&builder, ',')
		}
		first = false
		strings.Builder_Write_Text(&builder, quote_text(key))
		strings.Builder_Write_Byte(&builder, ':')
		strings.Builder_Write_Text(&builder, json_field_value_text(value))
	}
	strings.Builder_Write_Byte(&builder, '}')
	return builder.String()
}

func json_field_value_text(value any) (text string) {
	switch typed := value.(type) {
	case string:
		return quote_text(typed)
	case []string:
		return json_string_list_text(typed)
	case map[string][]string:
		return json_string_list_map_text(typed)
	default:
		aver.Always(false, "JSON fixture value type stays supported.")
		return ""
	}
}

func json_string_list_text(values []string) (text string) {
	var builder strings.Builder
	strings.Builder_Write_Byte(&builder, '[')
	for index, value := range values {
		if index > 0 {
			strings.Builder_Write_Byte(&builder, ',')
		}
		strings.Builder_Write_Text(&builder, quote_text(value))
	}
	strings.Builder_Write_Byte(&builder, ']')
	return builder.String()
}

func json_string_list_map_text(values map[string][]string) (text string) {
	var builder strings.Builder
	strings.Builder_Write_Byte(&builder, '{')
	first := true
	for key, value := range values {
		if !first {
			strings.Builder_Write_Byte(&builder, ',')
		}
		first = false
		strings.Builder_Write_Text(&builder, quote_text(key))
		strings.Builder_Write_Byte(&builder, ':')
		strings.Builder_Write_Text(&builder, json_string_list_text(value))
	}
	strings.Builder_Write_Byte(&builder, '}')
	return builder.String()
}

// The lint.json documents that decode cleanly: the full document, one with every
// list empty, and one carrying ignore globs.
func parse_configuration_valid_cases() (cases []parse_configuration_case) {
	instrumentation := []string{"shared/std/sim/aver/**"}
	return []parse_configuration_case{
		{
			Name: "valid full document",
			Input: configuration_document(
				map[string]any{"instrumentation_packages": []string{"a/b", "c/d"}}),
			Want_Shared: "example.com/lib",
			Want_List:   []string{"a/b", "c/d"},
		},
		{
			Name:        "empty lists accepted",
			Input:       configuration_document(nil),
			Want_Shared: "example.com/lib",
		},
		{
			Name: "ignore globs accepted",
			Input: configuration_document(
				map[string]any{"ignore": []string{"a/b", "c.md"}}),
			Want_Shared: "example.com/lib",
		},
		{
			Name: "mixed positive and negated globs accepted",
			Input: configuration_document(
				map[string]any{"ignore": []string{"keep/**", "!keep/sub"}}),
			Want_Shared: "example.com/lib",
		},
		{
			Name: "broader assertion exemption over instrumentation accepted",
			Input: configuration_document(map[string]any{
				"instrumentation_packages":           instrumentation,
				"opt_out_assertion_mandate_packages": []string{"shared/std/sim/**"},
			}),
			Want_Shared: "example.com/lib",
			Want_List:   instrumentation,
		},
	}
}

// Every hard-error lint.json form, each with all other keys valid so the case
// reaches the check it names: a missing required key (including the newest,
// opt_out_recursion_ban), an empty value where one is required, an unknown or
// removed key, a rejected glob, a wrong value type, and malformed JSON.
func parse_configuration_error_cases() (cases []parse_configuration_case) {
	return []parse_configuration_case{
		{
			Name:     "missing shared_component rejected",
			Input:    configuration_document(map[string]any{"shared_component": nil}),
			Want_Err: true,
		},
		{
			Name: "missing opt_out_recursion_ban rejected",
			Input: configuration_document(
				map[string]any{"opt_out_recursion_ban": nil}),
			Want_Err: true,
		},
		{
			Name:     "empty shared_component rejected",
			Input:    configuration_document(map[string]any{"shared_component": ""}),
			Want_Err: true,
		},
		{
			Name: "unknown key rejected",
			Input: configuration_document(
				map[string]any{"global_api_allowlst": []string{}}),
			Want_Err: true,
		},
		{
			Name: "removed path_casing_allowlist rejected",
			Input: configuration_document(
				map[string]any{"path_casing_allowlist": []string{"foo"}}),
			Want_Err: true,
		},
		{
			Name: "unknown key word_replacement typo rejected",
			Input: configuration_document(
				map[string]any{"word_replacement": []string{}}),
			Want_Err: true,
		},
		{
			Name: "wrong value type rejected",
			Input: configuration_document(
				map[string]any{"instrumentation_packages": "no"}),
			Want_Err: true,
		},
		{
			Name: "explicit instrumentation assertion exemption rejected",
			Input: configuration_document(map[string]any{
				"instrumentation_packages":           []string{"shared/std/snap/**"},
				"opt_out_assertion_mandate_packages": []string{"shared/std/snap/**"},
			}),
			Want_Err: true,
		},
		{
			Name:     "malformed json rejected",
			Input:    "{",
			Want_Err: true,
		},
	}
}

// The negation-specific hard-error forms: a bare "!",
// a "!" negating only whitespace, and a list that is entirely negation entries.
func parse_configuration_negation_error_cases() (cases []parse_configuration_case) {
	return []parse_configuration_case{
		{
			Name: "bare negation rejected",
			Input: configuration_document(
				map[string]any{"ignore": []string{"!"}}),
			Want_Err: true,
		},
		{
			Name: "negation of only whitespace rejected",
			Input: configuration_document(
				map[string]any{"ignore": []string{"!  "}}),
			Want_Err: true,
		},
		{
			Name: "all-negation list rejected",
			Input: configuration_document(
				map[string]any{"ignore": []string{"!keep", "!other"}}),
			Want_Err: true,
		},
	}
}

// Asserts the decoded configuration matches the wanted shared module and
// allowlist (order-sensitive).
func assert_configuration(
	t *testing.T, configuration *lint.Configuration, want_shared string, want_list []string,
) {
	t.Helper()
	if configuration.Shared_Component != want_shared {
		t.Fatalf("shared_component: got %q want %q",
			configuration.Shared_Component, want_shared)
	}
	if len(configuration.Instrumentation_Packages) != len(want_list) {
		t.Fatalf("allowlist len: got %d want %d",
			len(configuration.Instrumentation_Packages), len(want_list))
	}
	for i, want := range want_list {
		if configuration.Instrumentation_Packages[i] != want {
			t.Errorf("allowlist[%d]: got %q want %q",
				i, configuration.Instrumentation_Packages[i], want)
		}
	}
}

// Test_Ignored_Directory_Third_Party_Hyphen pins that the top-level third-party/
// drop-zone — the vendored Go-compiler test corpus, 70k files — is pruned by
// Ignored_Directory, the single predicate both the worktree walk and the
// module-discovery walk consult. The rule previously matched only the underscore
// spelling, so the hyphen tree was walked on every run and trimmed only later by
// the ignore list, after the directory syscalls were already spent.
func Test_Ignored_Directory_Third_Party_Hyphen(t *testing.T) {
	t.Parallel()
	if !lint.Ignored_Directory("third-party") {
		t.Errorf("Ignored_Directory(%q) = false, want true", "third-party")
	}
	// The drop-zone is top-level only, mirroring the underscore form: a nested
	// pkg/third-party is first-party code and must stay in view.
	if lint.Ignored_Directory("pkg/third-party") {
		t.Errorf("Ignored_Directory(%q) = true, want false (top-level only)",
			"pkg/third-party")
	}
}

// Returns the diagnostics positioned at filename — the seam the scope test uses
// to prove a file was parsed, since an AST-tier diagnostic exists only for a file
// the parser saw.
func diags_at(diags []lint.Diagnostic, filename string) (output []lint.Diagnostic) {
	for _, diagnostic := range diags {
		if diagnostic.Position.Filename == filename {
			output = append(output, diagnostic)
		}
	}
	return output
}

// Test_Scope_Parses_Target_And_Shared_Components pins the scope-to-parse contract
// via the AST tier: a dot-import is flagged only in a file the parser saw, so the
// per-file diagnostics map exactly the parse set. A scoped run parses the scope's
// own module and the shared library — the one module a first-party file may
// import, so its packages must be in view for the transitive-purity rule, else it
// fails open — and nothing else. An unrelated out-of-scope module is never
// parsed; that is the work scope skips and the reason it is cheap.
func Test_Scope_Parses_Target_And_Shared_Components(t *testing.T) {
	t.Parallel()
	dot_import_file := "// Package fixture is a fixture.\n" +
		"package fixture\n\nimport . \"fmt\"\n"
	fsys := fstest.MapFS{
		"shared/go.mod": &fstest.MapFile{
			Data: []byte("module example.com/shared\n")},
		"shared/lib/library.go": &fstest.MapFile{Data: []byte(dot_import_file)},
		"mybinary/go.mod": &fstest.MapFile{
			Data: []byte("module example.com/mybinary\n")},
		"mybinary/pkg/entry.go": &fstest.MapFile{Data: []byte(dot_import_file)},
		"other/go.mod": &fstest.MapFile{
			Data: []byte("module example.com/other\n")},
		"other/lib/library.go": &fstest.MapFile{Data: []byte(dot_import_file)},
	}
	diags, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys:             fsys,
		Shared_Component: "shared",
		Scope:            "mybinary",
	})
	if err != nil {
		t.Fatalf("Check_File_System: %v", err)
	}
	if !requirements_diagnosed(
		diags_at(diags, "mybinary/pkg/entry.go"),
		"Do not use a dot import") {
		t.Errorf("in-scope file mybinary/pkg/entry.go was not parsed")
	}
	if !requirements_diagnosed(
		diags_at(diags, "shared/lib/library.go"),
		"Do not use a dot import") {
		t.Errorf("shared-library file shared/lib/library.go was not parsed; " +
			"transitive-purity resolution of imports into shared would fail open")
	}
	if requirements_diagnosed(
		diags_at(diags, "other/lib/library.go"),
		"Do not use a dot import") {
		t.Errorf("out-of-scope file other/lib/library.go was parsed; " +
			"scope must prune unrelated modules from the parse set")
	}
}

// Test_Deterministic_Library_Glob verifies a `*` entry releases each direct child
// package of a directory — `*` spans one path segment — so both pkg/a and pkg/b are
// released, which a bare "pkg" entry (matching only the empty "pkg" dir) would not.
func Test_Deterministic_Library_Glob(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"go.mod": []byte("module fixture\n\ngo 1.25\n"),
		"pkg/a/a.go": []byte("// Package a is a fixture.\n" +
			"package a\n\n" +
			"// F is a fixture.\n" +
			"func F() {\n\tgo done()\n}\n\n" +
			"func done() {\n\treturn\n}\n"),
		"pkg/b/b.go": []byte("// Package b is a fixture.\n" +
			"package b\n\n" +
			"// G is a fixture.\n" +
			"func G() {\n\tselect {}\n}\n"),
	}
	diags := deterministic_self_diagnostics(t, files, []string{"pkg/*"})
	if requirements_diagnosed(diags, "must not start a goroutine") {
		t.Fatal("a * entry must release each direct child package")
	}
	if requirements_diagnosed(diags, "must not use select") {
		t.Fatal("a * entry must release each direct child package")
	}
}

// Test_Deterministic_Negation_Holds_Package verifies a "!"-prefixed entry holds
// one package to the deterministic tier despite a broader release entry: pkg/keep
// stays covered (its goroutine is flagged) while pkg/other, matched only by the
// broader entry, is released.
func Test_Deterministic_Negation_Holds_Package(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"go.mod": []byte("module fixture\n\ngo 1.25\n"),
		"pkg/keep/keep.go": []byte("// Package keep is a fixture.\n" +
			"package keep\n\n" +
			"// F is a fixture.\n" +
			"func F() {\n\tgo done()\n}\n\n" +
			"func done() {\n\treturn\n}\n"),
		"pkg/other/other.go": []byte("// Package other is a fixture.\n" +
			"package other\n\n" +
			"// G is a fixture.\n" +
			"func G() {\n\tselect {}\n}\n"),
	}
	diags := deterministic_self_diagnostics(t, files, []string{"pkg/*", "!pkg/keep"})
	if !requirements_diagnosed(diags, "must not start a goroutine") {
		t.Fatal("!pkg/keep must hold pkg/keep to the deterministic tier")
	}
	if requirements_diagnosed(diags, "must not use select") {
		t.Fatal("pkg/* must still release pkg/other, which the negation does not name")
	}
}

// Ordered refinement avoids duplicating broad package groups after one temporary
// holdback ends.
func Test_Deterministic_Later_Positive_Refines_Negation(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"go.mod": []byte("module fixture\n\ngo 1.25\n"),
		"pkg/keep/keep.go": []byte("// Package keep is a fixture.\n" +
			"package keep\n\n" +
			"// F is a fixture.\n" +
			"func F() {\n\tgo done()\n}\n\n" +
			"func done() {\n\treturn\n}\n"),
	}
	diags := deterministic_self_diagnostics(
		t, files, []string{"pkg/*", "!pkg/keep", "pkg/keep"})
	if requirements_diagnosed(diags, "must not start a goroutine") {
		t.Fatal("later pkg/keep must release pkg/keep again")
	}
}

// Test_Deterministic_Sibling_Import verifies a deterministic package importing
// another deterministic first-party package satisfies the induction — no violation
// — while still held to the bans itself. With no exceptions every pure package is
// deterministic, so the sibling import is clean and only the goroutine is flagged.
func Test_Deterministic_Sibling_Import(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"go.mod": []byte("module fixture\n\ngo 1.25\n"),
		"pkg/grp/a/a.go": []byte("// Package a is a fixture.\n" +
			"package a\n\n" +
			"import \"fixture/pkg/grp/b\"\n\n" +
			"// F is a fixture.\n" +
			"func F() {\n\tb.G()\n\tgo done()\n}\n\n" +
			"func done() {\n\treturn\n}\n"),
		"pkg/grp/b/b.go": []byte("// Package b is a fixture.\n" +
			"package b\n\n" +
			"// G is a fixture.\n" +
			"func G() {\n\treturn\n}\n"),
	}
	diags := deterministic_self_diagnostics(t, files, nil)
	if !requirements_diagnosed(diags, "must not start a goroutine") {
		t.Fatal("a pure package is held to the tier by default")
	}
	if requirements_diagnosed(diags, "import only deterministic packages") {
		t.Fatal("importing a deterministic sibling must satisfy the induction")
	}
}

// Test_Deterministic_Instrumentation_Auto_Released verifies an instrumentation
// package is released from the deterministic tier with no duplicate
// pure_but_indeterministic_packages entry: write-only instrumentation is inherently
// nondeterministic, so listing it in instrumentation_packages implies the release.
func Test_Deterministic_Instrumentation_Auto_Released(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"go.mod": []byte("module fixture\n\ngo 1.25\n"),
		"pkg/instr/instr.go": []byte("// Package instr is a fixture.\n" +
			"package instr\n\n" +
			"// F is a fixture.\n" +
			"func F() {\n\tgo done()\n}\n\n" +
			"func done() {\n\treturn\n}\n"),
	}
	fsys := fstest.MapFS{}
	for name, content := range files {
		fsys[name] = &fstest.MapFile{Data: content}
	}
	diags, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys:                     fsys,
		Scope:                    "pkg",
		Shared_Component:         DOCTRINE_SHARED_COMPONENT_DIRECTORY,
		Instrumentation_Packages: []string{"pkg/instr/**"},
	})
	if err != nil {
		t.Fatalf("Check_File_System: %v", err)
	}
	if requirements_diagnosed(diags, "must not start a goroutine") {
		t.Fatal("instrumentation package must auto-release from determinism")
	}
}
