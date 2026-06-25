package lint_test

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/james-orcales/james-orcales/lint/internal"
)

// This file is the doctrine, enforced by the lint tool, which dogfoods on this
// very package whenever the tool lints lint/internal. Each Test_<Heading> builds
// an in-memory fixture, violates exactly the rule its heading names, and asserts
// the tool reports it — so there is no duplicated checking logic. The
// SPECIFICATION.md-structure rules now live in the specification subpackage and
// are tested there; specification_baseline remains here for the markdown and
// file-count rules that build on a minimal, self-consistent spec fixture.

// Test_Diagnostics_Tier_One verifies a tier-one diagnostic always prints, and
// any tier-one anywhere in scope suppresses every tier-two diagnostic.
func Test_Diagnostics_Tier_One(t *testing.T) {
	t.Parallel()
	var stdout strings.Builder
	lint_main(t, &lint.Main_Input{
		Fsys: fstest.MapFS{
			"a.go": &fstest.MapFile{Data: []byte(
				"// missing period at end of this comment\npackage p\n")},
			"b.go": &fstest.MapFile{Data: gofmt_must(t, "// Package p is a fixture.\n"+
				"package p\n\nfunc f() { f() }\n")},
		},
		Stdout: &stdout,
		Stderr: &strings.Builder{},
	})
	if !strings.Contains(stdout.String(), "should end with") {
		t.Fatalf("the tier-one diagnostic must print; got: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "recursion") {
		t.Fatalf("a tier-two diagnostic must be suppressed when tier one fires "+
			"anywhere in scope; got: %s", stdout.String())
	}
}

// Test_Diagnostics_Tier_Two verifies a tier-two diagnostic prints when no
// tier-one diagnostic fired anywhere in scope.
func Test_Diagnostics_Tier_Two(t *testing.T) {
	t.Parallel()
	var stdout strings.Builder
	lint_main(t, &lint.Main_Input{
		Fsys: fstest.MapFS{"a.go": &fstest.MapFile{Data: gofmt_must(t,
			"// Package p is a fixture.\npackage p\n\nfunc f() { f() }\n")}},
		Stdout: &stdout,
		Stderr: &strings.Builder{},
	})
	if !strings.Contains(stdout.String(), "recursion") {
		t.Fatalf("a tier-two diagnostic must print with no tier one; got: %s",
			stdout.String())
	}
}

// Test_Repository_Ignored_Directories verifies a banned script that would be
// flagged anywhere else is silent under an ignored directory.
func Test_Repository_Ignored_Directories(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"vendor/build.sh": []byte("echo hi\n"),
	}
	if specification_flags(t, files, "as a go script") {
		t.Fatal("a script under an ignored directory must not be flagged")
	}
}

// Test_Repository_Path_Casing verifies a hyphenated directory name is flagged.
func Test_Repository_Path_Casing(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"bad-dir/x.go": []byte("// Package x is a fixture.\npackage x\n"),
	}
	if !specification_flags(t, files, "rename bad-dir -> bad_dir") {
		t.Fatal("a hyphenated directory name must be flagged")
	}
}

// Test_Repository_Symlinks pins the tracked-symlink rule: a tracked symlink must
// resolve to a tracked target. An untracked symlink, and every symlink when there
// is no tracked set, are exempt.
func Test_Repository_Symlinks(t *testing.T) {
	t.Parallel()
	// A target outside the tracked set is flagged whether it is missing on disk,
	// escapes the repo, or merely sits outside the set.
	if !specification_diagnosed(
		symlink_lint(t, "missing", nil, map[string]bool{"link": true}),
		"untracked symlink target") {
		t.Fatal("an untracked symlink target must be flagged")
	}
	// A target that is a tracked file is clean.
	if specification_diagnosed(symlink_lint(t, "real.txt", nil,
		map[string]bool{"link": true, "real.txt": true}), "symlink") {
		t.Fatal("a tracked file target must not be flagged")
	}
	// A target directory that holds tracked files is clean.
	if specification_diagnosed(symlink_lint(t, "dir", nil,
		map[string]bool{"link": true, "dir/x": true}), "symlink") {
		t.Fatal("a tracked directory target must not be flagged")
	}
	// Only tracked symlinks are checked, so one absent from the set is skipped.
	if specification_diagnosed(symlink_lint(t, "missing", nil,
		map[string]bool{"other": true}), "symlink") {
		t.Fatal("an untracked symlink must not be checked")
	}
	// With no tracked set (a non-git tree) the rule self-disables.
	if specification_diagnosed(symlink_lint(t, "missing", nil, nil), "symlink") {
		t.Fatal("a nil tracked set must disable the check")
	}
	// A symlink to its own path is flagged though its own path is tracked.
	if !specification_diagnosed(
		symlink_lint(t, "link", nil, map[string]bool{"link": true}),
		"symlink targets itself") {
		t.Fatal("a self-referential symlink must be flagged")
	}
	// An unreadable target cannot be verified, so it is flagged.
	if !specification_diagnosed(
		symlink_lint(t, "x", fs.ErrInvalid, map[string]bool{"link": true}),
		"unreadable symlink target") {
		t.Fatal("an unreadable symlink target must be flagged")
	}
}

// Test_Repository_Banned_Script_Extensions verifies a file flagged by its
// scripting extension is caught.
func Test_Repository_Banned_Script_Extensions(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"build.sh": []byte("echo hi\n"),
	}
	if !specification_flags(t, files, "as a go script") {
		t.Fatal("a shell script must be flagged")
	}
}

// Test_Repository_Banned_Build_Files verifies a file flagged by its base name
// is caught.
func Test_Repository_Banned_Build_Files(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"Makefile": []byte("all:\n\techo hi\n"),
	}
	if !specification_flags(t, files, "as a go script") {
		t.Fatal("a makefile base name must be flagged")
	}
}

// Test_Repository_Banned_Archives verifies an xz-compressed file is flagged.
func Test_Repository_Banned_Archives(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"backup.tar.xz": []byte("\xfd7zXZ\x00"),
	}
	if !specification_flags(t, files, ".xz files are banned") {
		t.Fatal("an xz file must be flagged")
	}
}

// Test_Repository_Conflict_Markers verifies a merge-conflict marker line is flagged.
func Test_Repository_Conflict_Markers(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"notes.txt": []byte("ours\n<<<<<<< HEAD\ntheirs\n"),
	}
	if !specification_flags(t, files, "resolve the conflict") {
		t.Fatal("a conflict marker must be flagged")
	}
}

// Test_Repository_Github_Actions verifies a uses: line in a workflow is flagged.
func Test_Repository_Github_Actions(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		".github/workflows/ci.yml": []byte("steps:\n  - uses: actions/checkout@v4\n"),
	}
	if !repository_flags(t, files, "third-party github action banned") {
		t.Fatal("a uses: line must be flagged")
	}
}

// Test_Markdown_Line_Width verifies a line beyond the display-column cap is
// flagged in Markdown prose, in fenced code, and in Go source, while a table row
// and a Go raw string literal — neither of which can wrap — stay exempt.
func Test_Markdown_Line_Width(t *testing.T) {
	t.Parallel()
	files := specification_baseline(t)
	specification_markdown_append(files, "\n# Wide\n\n"+strings.Repeat("x", 120)+"\n")
	if !specification_diagnosed(specification_self_diagnostics(t, files),
		"visual limit") {
		t.Fatal("an over-wide markdown line must be flagged")
	}
	// A table row over the cap stays exempt: pipe-delimited cells cannot wrap to
	// the next line, so the column rule cannot apply to them.
	table := specification_baseline(t)
	specification_markdown_append(table, "\n| "+strings.Repeat("x", 120)+" |\n")
	if specification_diagnosed(specification_self_diagnostics(t, table),
		"visual limit") {
		t.Fatal("an over-wide table row must be exempt")
	}
	// A long line inside a fenced code block is NOT exempt: code is held to the
	// same visual cap as prose.
	fenced := specification_baseline(t)
	specification_markdown_append(fenced, "\n```\n"+strings.Repeat("x", 120)+"\n```\n")
	if !specification_diagnosed(specification_self_diagnostics(t, fenced),
		"visual limit") {
		t.Fatal("an over-wide line inside a fenced code block must be flagged")
	}
	// The table-row and URL exemptions are markdown-rendering allowances; inside a
	// fence the line is literal code, where that rationale evaporates, so a
	// table-shaped or URL-bearing code line is held to the cap like any other.
	fenced_table := specification_baseline(t)
	specification_markdown_append(fenced_table,
		"\n```\n| "+strings.Repeat("x", 120)+" |\n```\n")
	if !specification_diagnosed(specification_self_diagnostics(t, fenced_table),
		"visual limit") {
		t.Fatal("an over-wide table-shaped line inside a fence must be flagged")
	}
	fenced_url := specification_baseline(t)
	specification_markdown_append(fenced_url,
		"\n```\nhttps://"+strings.Repeat("x", 120)+"\n```\n")
	if !specification_diagnosed(specification_self_diagnostics(t, fenced_url),
		"visual limit") {
		t.Fatal("an over-wide URL line inside a fence must be flagged")
	}
	// In prose the URL exemption holds: an unbreakable URL has no fold point.
	prose_url := specification_baseline(t)
	specification_markdown_append(prose_url,
		"\nSee https://"+strings.Repeat("x", 120)+"\n")
	if specification_diagnosed(specification_self_diagnostics(t, prose_url),
		"visual limit") {
		t.Fatal("a prose line dominated by a URL must be exempt")
	}
	source := specification_one_file("package fixture\n\n// " + strings.Repeat("x", 110) +
		"\n// F does.\nfunc F() (n int) {\n\treturn 0\n}\n")
	if !specification_flags(t, source, "chars (max 100)") {
		t.Fatal("an over-wide source line must be flagged")
	}
	// A long line inside a backtick raw string literal is exempt: the bytes are
	// data, not wrappable source, so the column cap cannot apply.
	raw := specification_one_file("package fixture\n\n// X is a fixture.\nconst X = `" +
		strings.Repeat("x", 120) + "`\n")
	if specification_flags(t, raw, "chars (max 100)") {
		t.Fatal("a long line inside a raw string literal must be exempt")
	}
}

// Test_Markdown_Whitespace verifies a Markdown prose line with trailing
// whitespace is flagged.
func Test_Markdown_Whitespace(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"notes.md": []byte("# Title\n\nIt trails.   \n"),
	}
	if !specification_flags(t, files, "markdown line has trailing whitespace") {
		t.Fatal("a trailing-whitespace markdown line must be flagged")
	}
}

// Test_Markdown_Agent_Documentation_Size verifies an agent doc beyond the line
// cap is flagged.
func Test_Markdown_Agent_Documentation_Size(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"CLAUDE.md": []byte(strings.Repeat("line\n", 101)),
	}
	if !specification_flags(t, files, "split or trim it under") {
		t.Fatal("an over-long agent doc must be flagged")
	}
}

// Test_Markdown_Agent_Documentation_Pairing verifies a CLAUDE.md without its
// AGENTS.md sibling is flagged.
func Test_Markdown_Agent_Documentation_Pairing(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"top/CLAUDE.md": []byte("Shared instructions.\n"),
	}
	if !repository_flags(t, files, "AGENTS.md is missing") {
		t.Fatal("an unpaired CLAUDE.md must be flagged")
	}
}

// Test_Component_Layout_Single_Module verifies a nested go.mod — a second
// module inside the one-module repo — is flagged.
func Test_Component_Layout_Single_Module(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"go.mod":     []byte(doctrine_root_go_module),
		"pkg/go.mod": []byte("module example.com/nested\n\ngo 1.25\n"),
	}
	if !specification_flags(t, files, "nested go.mod") {
		t.Fatal("a nested go.mod must be flagged")
	}
}

// Test_Component_Layout_Shared_Component verifies the shared library's internal/ subtree
// and any package main are flagged: the first hides surface, the second is an entry
// point a fully-importable library has no business owning.
func Test_Component_Layout_Shared_Component(t *testing.T) {
	t.Parallel()
	// The shared library lives in shared/ so its Root matches the
	// shared-component directory the spec helper passes; binary fixtures live in
	// other top-level dirs and so stay binaries.
	files := map[string][]byte{
		"shared/internal/x/x.go": []byte("// Package x is a fixture.\npackage x\n"),
	}
	if !specification_flags(t, files, "forbids internal/ directories") {
		t.Fatal("a shared-library internal/ tree must be flagged")
	}
	main_files := map[string][]byte{
		"shared/main.go": []byte("package main\n\nfunc main() {\n\tprintln(0)\n}\n"),
	}
	if !specification_flags(t, main_files, "forbids package main") {
		t.Fatal("a shared-library package main must be flagged")
	}
}

// Test_Component_Layout_Binary_Component verifies a binary component's non-main
// package outside internal/ is flagged.
func Test_Component_Layout_Binary_Component(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"pkg/feature/feature.go": []byte(
			"// Package feature is a fixture.\npackage feature\n"),
	}
	if !specification_flags(t, files, "move feature -> pkg/internal/feature") {
		t.Fatal("a non-main package outside internal/ must be flagged")
	}
}

// Test_Component_Layout_Main_Package verifies a binary module's main package outside
// the module root — here under a cmd/ directory — is flagged.
func Test_Component_Layout_Main_Package(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"pkg/cmd/app/main.go": []byte("package main\n\nfunc main() {\n\tprintln(0)\n}\n"),
	}
	if !specification_flags(t, files, "main package must sit at the component root") {
		t.Fatal("a main package outside the module root must be flagged")
	}
}

// Test_Component_Layout_Internal_Entry_Point verifies a binary module without a
// func Main in its top-level internal/ package is flagged.
func Test_Component_Layout_Internal_Entry_Point(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"pkg/main.go": []byte("package main\n\nfunc main() {\n\tprintln(0)\n}\n"),
	}
	if !specification_flags(t, files, "declares no func Main in internal/") {
		t.Fatal("a binary module without an internal func Main must be flagged")
	}
}

// Test_Component_Layout_Tier_Depth verifies a package nested beyond one non-main ancestor
// is flagged.
func Test_Component_Layout_Tier_Depth(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"shared/a/a.go":     []byte("// Package a is a fixture.\npackage a\n"),
		"shared/a/b/b.go":   []byte("// Package b is a fixture.\npackage b\n"),
		"shared/a/b/c/c.go": []byte("// Package c is a fixture.\npackage c\n"),
	}
	if !specification_flags(t, files, "exceeds library tier") {
		t.Fatal("an over-nested package must be flagged")
	}
}

// Test_Component_Layout_Impure_Imports verifies a pure package importing a
// denylisted impure stdlib package is flagged.
func Test_Component_Layout_Impure_Imports(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"shared/lib/library.go": []byte("// Package library x.\npackage library\n\n" +
			"import \"os\"\n\n// F reads.\nfunc F() (s string) {\n" +
			"\treturn os.Getenv(\"X\")\n}\n"),
	}
	if !specification_flags(t, files, "impure stdlib import") {
		t.Fatal("an impure stdlib import must be flagged")
	}
}

// Test_Component_Layout_Impure_Calls verifies a pure package making a denylisted
// impure stdlib call is flagged even when the package may be imported.
func Test_Component_Layout_Impure_Calls(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"shared/lib/library.go": []byte("// Package library x.\npackage library\n\n" +
			"import \"fmt\"\n\n// F prints.\nfunc F() {\n" +
			"\tfmt.Println(\"x\")\n}\n"),
	}
	if !specification_flags(t, files, "impure stdlib call") {
		t.Fatal("an impure stdlib call must be flagged")
	}
}

// Test_Component_Layout_Transitive_Purity verifies a pure package importing an
// impure `default` package is flagged.
func Test_Component_Layout_Transitive_Purity(t *testing.T) {
	t.Parallel()
	const module = "github.com/james-orcales/james-orcales/shared"
	files := map[string][]byte{
		"shared/lib/library.go": []byte("// Package library x.\npackage library\n\n" +
			"import \"" + module + "/widget/default\"\n\n" +
			"// F uses the default.\nfunc F() (s string) {\n" +
			"\treturn widget.Name()\n}\n"),
		"shared/widget/default/wire.go": []byte(
			"// Package widget is a fixture.\npackage widget\n\n" +
				"// Name names.\nfunc Name() (s string) {\n\treturn \"x\"\n}\n"),
	}
	if !specification_flags(t, files, "impure dependency") {
		t.Fatal("a pure package importing an impure package must be flagged")
	}
}

// Test_Component_Layout_Transitive_Stdlib verifies the ban reaches a pure package's
// test file: a curated stdlib API that is impure only transitively
// (filepath.Walk -> the filesystem) is caught even in a _test.go file, where the
// transitive-purity ban still binds. log itself is exempt telemetry, so it cannot
// stand in for the curated impure case here.
func Test_Component_Layout_Transitive_Stdlib(t *testing.T) {
	t.Parallel()
	test_files := map[string][]byte{
		"shared/lib/library.go": []byte("// Package library x.\npackage library\n"),
		"shared/lib/library_test.go": []byte(
			"package library_test\n\nimport \"path/filepath\"\n\n" +
				"// Test_F walks.\nfunc Test_F() {\n" +
				"\tfilepath.Walk(\".\", nil)\n}\n"),
	}
	if !specification_flags(t, test_files, "impure transitive call") {
		t.Fatal("a pure package's test calling a curated stdlib API must be flagged")
	}
}

// Test_Component_Layout_Binary_Purity verifies package main is an impure home: a
// binary's main package using impure stdlib is not flagged.
func Test_Component_Layout_Binary_Purity(t *testing.T) {
	t.Parallel()
	main_impure := map[string][]byte{
		"pkg/main.go": []byte("package main\n\nimport \"os\"\n\n" +
			"func main() {\n\t_ = os.Getenv(\"X\")\n}\n"),
	}
	if specification_flags(t, main_impure, "impure stdlib") {
		t.Fatal("package main is an impure home; it must not be flagged")
	}
}

// Test_Component_Layout_Library_Purity verifies a shared library's pure package
// using impure stdlib is flagged.
func Test_Component_Layout_Library_Purity(t *testing.T) {
	t.Parallel()
	pure_impure := map[string][]byte{
		"shared/lib/library.go": []byte("// Package library x.\npackage library\n\n" +
			"import \"os\"\n\n// F reads.\nfunc F() (s string) {\n" +
			"\treturn os.Getenv(\"X\")\n}\n"),
	}
	if !specification_flags(t, pure_impure, "impure stdlib") {
		t.Fatal("a pure library package using impure stdlib must be flagged")
	}
}

// Test_Component_Layout_Default_Package_Impurity verifies a shared library's
// optional `default` package may be impure: it is not flagged.
func Test_Component_Layout_Default_Package_Impurity(t *testing.T) {
	t.Parallel()
	default_impure := map[string][]byte{
		"shared/foo/foo.go": []byte("// Package foo is a fixture.\npackage foo\n"),
		"shared/foo/default/wire.go": []byte(
			"// Package foo is a fixture.\npackage foo\n\n" +
				"import \"os\"\n\n// F reads.\nfunc F() (s string) {\n" +
				"\treturn os.Getenv(\"X\")\n}\n"),
	}
	if specification_flags(t, default_impure, "impure stdlib") {
		t.Fatal("an optional `default` package may be impure; it must not be flagged")
	}
}

// Test_Source_And_Test_Bans_Dot_Imports verifies a dot import is flagged.
func Test_Source_And_Test_Bans_Dot_Imports(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\nimport . \"strings\"\n")
	if !specification_flags(t, files, "dot import") {
		t.Fatal("a dot import must be flagged")
	}
}

// Test_Source_And_Test_Bans_Blank_Imports verifies a blank import is flagged.
func Test_Source_And_Test_Bans_Blank_Imports(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\nimport _ \"strings\"\n")
	if !specification_flags(t, files, "blank import is banned") {
		t.Fatal("a blank import must be flagged")
	}
}

// Test_Source_And_Test_Bans_Grouped_Declarations verifies a grouped declaration is flagged.
func Test_Source_And_Test_Bans_Grouped_Declarations(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\nconst (\n\tA = 1\n\tB = 2\n)\n")
	if !specification_flags(t, files, "grouped declaration banned") {
		t.Fatal("a grouped declaration must be flagged")
	}
}

// Test_Source_And_Test_Bans_Iota verifies iota is flagged.
func Test_Source_And_Test_Bans_Iota(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\nconst X = iota\n")
	if !specification_flags(t, files, "iota is banned") {
		t.Fatal("iota must be flagged")
	}
}

// Test_Source_And_Test_Bans_Interface_Declarations verifies an interface declaration is flagged.
func Test_Source_And_Test_Bans_Interface_Declarations(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n// I is a fixture.\n" +
		"type I interface{ M() int }\n")
	want := "interface declarations are banned (except for generics)"
	if !specification_flags(t, files, want) {
		t.Fatal("an interface declaration must be flagged")
	}
}

// Test_Source_And_Test_Bans_Variable_Shadows verifies a variable shadowing an
// outer scope is flagged.
func Test_Source_And_Test_Bans_Variable_Shadows(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n// F does.\n" +
		"func F() (n int) {\n\tn = 1\n\t{\n\t\tn := 2\n\t\t_ = n\n\t}\n\treturn n\n}\n")
	if !specification_flags(t, files, "shadows") {
		t.Fatal("a shadowed variable must be flagged")
	}
}

// Test_Source_And_Test_Bans_Mutable_Globals verifies a disallowed package-level var is flagged.
func Test_Source_And_Test_Bans_Mutable_Globals(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n// X is a fixture.\nvar X = 0\n")
	if !specification_flags(t, files, "package-level var is banned") {
		t.Fatal("a disallowed package-level var must be flagged")
	}
}

// Test_Source_And_Test_Bans_Discards verifies a bare _ = value discard is flagged.
func Test_Source_And_Test_Bans_Discards(t *testing.T) {
	t.Parallel()
	files := specification_one_file(
		"package fixture\n\n// F does.\nfunc F() {\n\tx := 1\n\t_ = x\n}\n")
	if !specification_flags(t, files, "discard") {
		t.Fatal("a bare discard must be flagged")
	}
}

// Test_Source_And_Test_Bans_Init_Functions verifies a func init is flagged.
func Test_Source_And_Test_Bans_Init_Functions(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\nfunc init() { println(0) }\n")
	if !specification_flags(t, files, "func init") {
		t.Fatal("a func init must be flagged")
	}
}

// Test_Source_And_Test_Bans_Empty_Bodies verifies an empty function body is flagged.
func Test_Source_And_Test_Bans_Empty_Bodies(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n// F does.\nfunc F() {}\n")
	if !specification_flags(t, files, "has an empty body") {
		t.Fatal("an empty function body must be flagged")
	}
}

// Test_Source_And_Test_Bans_Methods verifies a method outside the
// stdlib-interface exceptions is flagged.
func Test_Source_And_Test_Bans_Methods(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n// T is a fixture.\n" +
		"type T struct {\n\t// X is a fixture.\n\tX int\n}\n\n// Compute does.\n" +
		"func (t T) Compute() (n int) {\n\treturn t.X\n}\n")
	// Exempt the type-invariant rule: it is tier one and would otherwise suppress
	// the tier-two method diagnostic this test isolates.
	diags := invariant_exempt_self_diagnostics(t, files, []string{"pkg/**"})
	if !specification_diagnosed(diags, "does not satisfy any stdlib interface") {
		t.Fatal("a non-interface method must be flagged")
	}
}

// Test_Source_And_Test_Bans_Self_Recursion verifies a function that calls
// itself by bare name is flagged.
func Test_Source_And_Test_Bans_Self_Recursion(t *testing.T) {
	t.Parallel()
	files := specification_one_file(
		"package fixture\n\n// F loops.\nfunc F() {\n\tF()\n}\n")
	if !specification_flags(t, files, "calls itself") {
		t.Fatal("a self-recursive function must be flagged")
	}
}

// Test_Source_And_Test_Bans_Mutual_Recursion verifies a cycle of bare-name
// same-file calls is flagged.
func Test_Source_And_Test_Bans_Mutual_Recursion(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n// F calls G.\n" +
		"func F() {\n\tG()\n}\n\n// G calls F.\nfunc G() {\n\tF()\n}\n")
	if !specification_flags(t, files, "cycle") {
		t.Fatal("a mutually recursive cycle must be flagged")
	}
}

// Test_Source_And_Test_Bans_Compound_Conditions verifies a compound if condition is flagged.
func Test_Source_And_Test_Bans_Compound_Conditions(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n// F does.\n" +
		"func F() (n int) {\n\tif n > 0 && n < 5 {\n\t\tn = 1\n\t}\n\treturn n\n}\n")
	if !specification_flags(t, files, "compound if") {
		t.Fatal("a compound if condition must be flagged")
	}
}

// Test_Source_And_Test_Bans_Naked_Returns verifies a naked return is flagged.
func Test_Source_And_Test_Bans_Naked_Returns(t *testing.T) {
	t.Parallel()
	files := specification_one_file(
		"package fixture\n\n// F does.\nfunc F() (n int) {\n\tn = 1\n\treturn\n}\n")
	if !specification_flags(t, files, "naked return is banned") {
		t.Fatal("a naked return must be flagged")
	}
}

// Test_Source_And_Test_Bans_Fallthrough verifies a fallthrough is flagged.
func Test_Source_And_Test_Bans_Fallthrough(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n// F does.\n" +
		"func F() (n int) {\n\tswitch n {\n\tcase 1:\n\t\tfallthrough\n\tcase 2:\n" +
		"\t\tn = 3\n\t}\n\treturn n\n}\n")
	if !specification_flags(t, files, "fallthrough is banned") {
		t.Fatal("a fallthrough must be flagged")
	}
}

// Test_Source_And_Test_Bans_Bare_Loops verifies a bare infinite for loop is flagged.
func Test_Source_And_Test_Bans_Bare_Loops(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n// F does.\n" +
		"func F() {\n\tfor {\n\t\tbreak\n\t}\n}\n")
	if !specification_flags(t, files, "bare `for {}` is banned") {
		t.Fatal("a bare for loop must be flagged")
	}
}

// Test_Source_And_Test_Bans_Struct_Tags verifies a non-stdlib struct tag key is flagged.
func Test_Source_And_Test_Bans_Struct_Tags(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n// T is a fixture.\n" +
		"type T struct {\n\t// X is a fixture.\n\tX int `yaml:\"x\"`\n}\n")
	// Exempt the type-invariant rule: it is tier one and would otherwise suppress
	// the tier-two struct-tag diagnostic this test isolates.
	diags := invariant_exempt_self_diagnostics(t, files, []string{"pkg/**"})
	if !specification_diagnosed(diags, "is not stdlib") {
		t.Fatal("a non-stdlib struct tag must be flagged")
	}
}

// Test_Source_And_Test_Bans_Blank_Mutexes verifies a blank-named mutex field is flagged.
func Test_Source_And_Test_Bans_Blank_Mutexes(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\nimport \"sync\"\n\n" +
		"// T is a fixture.\ntype T struct {\n\t_ sync.Mutex\n\t" +
		"// X is a fixture.\n\tX int\n}\n")
	if !specification_flags(t, files, "blank-named sync mutex") {
		t.Fatal("a blank-named mutex must be flagged")
	}
}

// Test_Source_And_Test_Bans_Unbounded_Read verifies an unbounded read call is flagged.
func Test_Source_And_Test_Bans_Unbounded_Read(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\nimport \"io\"\n\n" +
		"// F reads.\nfunc F(r io.Reader) (b []byte) {\n" +
		"\tb, _ = io.ReadAll(r)\n\treturn b\n}\n")
	if !specification_flags(t, files, "unbounded-read") {
		t.Fatal("an unbounded read must be flagged")
	}
}

// Test_Source_And_Test_Bans_Unbounded_Decode verifies an unbounded decode is flagged.
func Test_Source_And_Test_Bans_Unbounded_Decode(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\nimport (\n\t\"encoding/json\"\n" +
		"\t\"io\"\n)\n\n// F decodes.\nfunc F(r io.Reader) (d *json.Decoder) {\n" +
		"\treturn json.NewDecoder(r)\n}\n")
	if !specification_flags(t, files, "unbounded-decode") {
		t.Fatal("an unbounded decode must be flagged")
	}
}

// Test_Source_And_Test_Bans_Unbounded_Decompression verifies an unbounded
// decompression is flagged.
func Test_Source_And_Test_Bans_Unbounded_Decompression(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\nimport (\n\t\"compress/gzip\"\n" +
		"\t\"io\"\n)\n\n// F wraps.\nfunc F(r io.Reader) (z *gzip.Reader, err error) {\n" +
		"\treturn gzip.NewReader(r)\n}\n")
	if !specification_flags(t, files, "unbounded-decompression") {
		t.Fatal("an unbounded decompression must be flagged")
	}
}

// Test_Source_And_Test_Bans_Unbounded_Allocation verifies an unbounded
// allocation is flagged.
func Test_Source_And_Test_Bans_Unbounded_Allocation(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\nimport \"bytes\"\n\n" +
		"// F buffers.\nfunc F() {\n" +
		"\tbytes.NewBuffer(nil)\n}\n")
	if !specification_flags(t, files, "unbounded-allocation") {
		t.Fatal("an unbounded allocation must be flagged")
	}
}

// Test_Source_And_Test_Bans_Unbounded_Http verifies an unbounded net/http call
// is flagged.
func Test_Source_And_Test_Bans_Unbounded_Http(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\nimport \"net/http\"\n\n" +
		"// F fetches.\nfunc F(url string) (resp *http.Response, err error) {\n" +
		"\treturn http.Get(url)\n}\n")
	if !specification_flags(t, files, "unbounded-http") {
		t.Fatal("an unbounded net/http call must be flagged")
	}
}

// Test_Source_And_Test_Bans_Deprecated_Ioutil verifies a deprecated ioutil call
// is flagged.
func Test_Source_And_Test_Bans_Deprecated_Ioutil(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\nimport (\n\t\"io\"\n" +
		"\t\"io/ioutil\"\n)\n\n// F reads.\nfunc F(r io.Reader) (b []byte, err error) {\n" +
		"\treturn ioutil.ReadAll(r)\n}\n")
	if !specification_flags(t, files, "deprecated-ioutil") {
		t.Fatal("a deprecated ioutil call must be flagged")
	}
}

// Test_Source_And_Test_Bans_Banned_Words verifies a banned word in any
// identifier is flagged.
func Test_Source_And_Test_Bans_Banned_Words(t *testing.T) {
	t.Parallel()
	files := specification_one_file(
		"package fixture\n\n// Length is a fixture.\nconst Length = 0\n")
	if !specification_flags(t, files, "banned substring \"length\"") {
		t.Fatal("a banned word must be flagged")
	}
}

// Test_Source_And_Test_Bans_Banned_Function_Words verifies the function-name-only
// word ban: helper in a function name is flagged, the word other identifiers may carry.
func Test_Source_And_Test_Bans_Banned_Function_Words(t *testing.T) {
	t.Parallel()
	files := specification_one_file(
		"package fixture\n\n// Helper helps.\nfunc Helper() (n int) {\n\treturn 0\n}\n")
	if !specification_flags(t, files, "banned substring \"helper\"") {
		t.Fatal("helper in a function name must be flagged")
	}
}

// Test_Source_And_Test_Requirements_Goimports verifies non-goimports formatting is flagged.
func Test_Source_And_Test_Requirements_Goimports(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\nimport  \"strings\"\n\n" +
		"// F does.\nfunc F() (s string) { return strings.TrimSpace(\"x\") }\n")
	if !specification_flags(t, files, "gofmt") {
		t.Fatal("non-goimports formatting must be flagged")
	}
}

// Test_Source_And_Test_Requirements_Default_Package_Name verifies a package in a
// `default/` directory that does not declare its parent's package clause is flagged.
func Test_Source_And_Test_Requirements_Default_Package_Name(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"pkg/default/wire.go": []byte("package wrong\n"),
	}
	if !specification_flags(t, files, "must declare 'package pkg'") {
		t.Fatal("a default package not declaring its parent's name must be flagged")
	}
}

// Test_Source_And_Test_Requirements_Entry_Point_First verifies a main declared after
// another function is flagged.
func Test_Source_And_Test_Requirements_Entry_Point_First(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package main\n\n// Run does.\n" +
		"func Run() {\n\tprintln(0)\n}\n\nfunc main() {\n\tRun()\n}\n")
	if !specification_flags(t, files, "func main should be declared first") {
		t.Fatal("a late main must be flagged")
	}
}

// Test_Source_And_Test_Requirements_Function_Size verifies a function over
// seventy lines is flagged.
func Test_Source_And_Test_Requirements_Function_Size(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n// F does.\nfunc F() {\n" +
		strings.Repeat("\tprintln(0)\n", 71) + "}\n")
	if !specification_flags(t, files, "max 70") {
		t.Fatal("an oversized function must be flagged")
	}
}

// Test_Source_And_Test_Requirements_Input_Structs verifies a function repeating
// a parameter type is flagged.
func Test_Source_And_Test_Requirements_Input_Structs(t *testing.T) {
	t.Parallel()
	files := specification_one_file(
		"package fixture\n\n// F does.\n" +
			"func F(a int, b int) (n int) {\n\treturn a + b\n}\n")
	if !specification_flags(t, files, "convert to") {
		t.Fatal("a repeated parameter type must be flagged")
	}
}

// Test_Source_And_Test_Requirements_Named_Returns verifies an unnamed return is flagged.
func Test_Source_And_Test_Requirements_Named_Returns(t *testing.T) {
	t.Parallel()
	files := specification_one_file(
		"package fixture\n\n// F does.\nfunc F() int {\n\treturn 0\n}\n")
	if !specification_flags(t, files, "unnamed return") {
		t.Fatal("an unnamed return must be flagged")
	}
}

// Test_Source_And_Test_Requirements_Keyed_Struct_Literals verifies an unkeyed
// struct literal is flagged.
func Test_Source_And_Test_Requirements_Keyed_Struct_Literals(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n// T is a fixture.\n" +
		"type T struct{ X int }\n\n// F builds T.\n" +
		"func F() (t T) {\n\tt = T{1}\n\treturn t\n}\n")
	if !specification_flags(t, files, "keyed") {
		t.Fatal("an unkeyed struct literal must be flagged")
	}
}

// Test_Source_And_Test_Requirements_Exported_Struct_Fields verifies an exported struct
// exposing an unexported-typed field is flagged.
func Test_Source_And_Test_Requirements_Exported_Struct_Fields(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n// private is a fixture.\n" +
		"type private struct{ X int }\n\n// T is a fixture.\n" +
		"type T struct{ P private }\n")
	if !specification_flags(t, files, "private") {
		t.Fatal("an exported field of an unexported type must be flagged")
	}
}

// Test_Source_And_Test_Requirements_Struct_Field_Public_Identifier verifies a
// lowercase struct field is flagged.
func Test_Source_And_Test_Requirements_Struct_Field_Public_Identifier(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n// T is a fixture.\n" +
		"type T struct{ count int }\n")
	if !specification_flags(t, files, "rename count -> Count") {
		t.Fatal("a lowercase struct field must be flagged")
	}
}

// Test_Source_And_Test_Requirements_Struct_Field_Type_Encapsulation verifies an
// exported struct whose field carries an unexported type is flagged.
func Test_Source_And_Test_Requirements_Struct_Field_Type_Encapsulation(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n// hidden is a fixture.\n" +
		"type hidden struct{ X int }\n\n// Widget is a fixture.\n" +
		"type Widget struct{ Inner hidden }\n")
	if !specification_flags(t, files, "public type Widget contains private type hidden") {
		t.Fatal("an exported struct field of an unexported type must be flagged")
	}
}

// Test_Source_And_Test_Requirements_Comment_Opening verifies a comment that does
// not open with a capital letter is flagged.
func Test_Source_And_Test_Requirements_Comment_Opening(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n// f does something.\nfunc F() {}\n")
	if !specification_flags(t, files, "should start with capital letter") {
		t.Fatal("a comment without a leading capital must be flagged")
	}
}

// Test_Source_And_Test_Requirements_Comment_Ending verifies a comment without a
// closing punctuation mark is flagged.
func Test_Source_And_Test_Requirements_Comment_Ending(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n// F does\nfunc F() {}\n")
	if !specification_flags(t, files, "should end with") {
		t.Fatal("a comment without closing punctuation must be flagged")
	}
}

// Test_Source_And_Test_Requirements_Package_Documentation_Comments verifies a missing
// package-clause doc comment is flagged. The documented F keeps the exported-declaration
// check silent so only the package-clause message can satisfy the assertion.
func Test_Source_And_Test_Requirements_Package_Documentation_Comments(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n// F does.\nfunc F() {}\n")
	if !specification_flags(t, files, "package \"fixture\" is missing a doc comment") {
		t.Fatal("a missing package-clause doc comment must be flagged")
	}
}

// Test_Source_And_Test_Requirements_Exported_Documentation_Comments verifies an
// undocumented exported declaration is flagged. The documented package clause keeps the
// package-clause check silent so only the exported-declaration message can satisfy it.
func Test_Source_And_Test_Requirements_Exported_Documentation_Comments(t *testing.T) {
	t.Parallel()
	files := specification_one_file(
		"// Package fixture is a fixture.\npackage fixture\n\nfunc F() {}\n")
	if !specification_flags(t, files, "exported func F is missing a doc comment") {
		t.Fatal("an undocumented exported declaration must be flagged")
	}
	// A lone Go directive is not documentation: CommentGroup.Text strips it,
	// leaving the declaration undocumented despite the comment being present.
	directive_only := specification_one_file(
		"// Package fixture is a fixture.\npackage fixture\n\n//go:noinline\nfunc G() {}\n")
	if !specification_flags(t, directive_only, "exported func G is missing a doc comment") {
		t.Fatal("an exported declaration documented only by a directive must be flagged")
	}
}

// Test_Source_And_Test_Requirements_Struct_Field_Documentation_Comments verifies an
// undocumented field of an exported struct is flagged.
func Test_Source_And_Test_Requirements_Struct_Field_Documentation_Comments(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n// Widget is a fixture.\n" +
		"type Widget struct {\n\tCount int\n}\n")
	if !specification_flags(t, files, "lacks a doc comment") {
		t.Fatal("an undocumented exported-struct field must be flagged")
	}
}

// Test_Source_And_Test_Requirements_Name_Style verifies an exported identifier
// not in Ada_Case is flagged. The rule governs identifier casing, not file
// names; a hyphenated path is the Path Casing rule's domain, so a file-name
// fixture would pass on that rule alone and never exercise this one.
func Test_Source_And_Test_Requirements_Name_Style(t *testing.T) {
	t.Parallel()
	files := specification_one_file(
		"package fixture\n\n// Bad_Name is a fixture.\nconst BadName = 0\n")
	if !specification_flags(t, files, "BadName -> Bad_Name") {
		t.Fatal("an exported identifier not in Ada_Case must be flagged")
	}
}

// Test_Source_And_Test_Requirements_Full_Words verifies an abbreviated name is flagged.
func Test_Source_And_Test_Requirements_Full_Words(t *testing.T) {
	t.Parallel()
	files := specification_one_file(
		"package fixture\n\n// Widget_Id is a fixture.\nconst Widget_Id = 0\n")
	if !specification_flags(t, files, "Widget_Identifier") {
		t.Fatal("an abbreviated name must be flagged")
	}
}

// Test_Source_And_Test_Requirements_Noun_Names verifies a present-participle name is flagged.
func Test_Source_And_Test_Requirements_Noun_Names(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n// Parsing is a fixture.\n" +
		"type Parsing struct{ X int }\n")
	if !specification_flags(t, files, "present participle") {
		t.Fatal("a present-participle name must be flagged")
	}
}

// Test_Source_And_Test_Requirements_Quantity_Suffixes verifies a len-bound name without
// the required suffix is flagged.
func Test_Source_And_Test_Requirements_Quantity_Suffixes(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n// F does.\n" +
		"func F(s string) (n int) {\n\ttotal := len(s)\n\treturn total\n}\n")
	if !specification_flags(t, files, "naming convention") {
		t.Fatal("a quantity name without its suffix must be flagged")
	}
}

// Test_Source_And_Test_Requirements_Arithmetic_Suffixes verifies an incoherent
// suffix arithmetic combination is flagged.
func Test_Source_And_Test_Requirements_Arithmetic_Suffixes(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n// F does.\n" +
		"func F() (n int) {\n\ta_index := 0\n\tb_size := 0\n" +
		"\tn = a_index + b_size\n\treturn n\n}\n")
	if !specification_flags(t, files, "is incoherent") {
		t.Fatal("an incoherent suffix arithmetic must be flagged")
	}
}

// Test_Source_And_Test_Requirements_Extremum_Suffixes verifies a leading max
// word in a declared name is flagged.
func Test_Source_And_Test_Requirements_Extremum_Suffixes(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n// Max count.\n" +
		"const max_count = 1\n")
	if !specification_flags(t, files, "rename max_count") {
		t.Fatal("a leading max in a declared name must be flagged")
	}
}

// Test_Source_And_Test_Requirements_Test_Documentation verifies an undocumented Test
// function is flagged.
func Test_Source_And_Test_Requirements_Test_Documentation(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"pkg/thing_test.go": []byte("package fixture_test\n\nimport \"testing\"\n\n" +
			"func Test_X(t *testing.T) { _ = t }\n"),
	}
	if !specification_flags(t, files, "is missing a doc comment") {
		t.Fatal("an undocumented test must be flagged")
	}
}

// Test_Source_And_Test_Requirements_Snap_Literals verifies a non-backticked snap literal
// is flagged.
func Test_Source_And_Test_Requirements_Snap_Literals(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n// F does.\n" +
		"func F(snap snapper) {\n\tsnap.Init(\"plain\")\n}\n\n" +
		"// snapper is a fixture.\ntype snapper struct{ X int }\n")
	if !specification_flags(t, files, "backticked raw string literal") {
		t.Fatal("a non-backticked snap literal must be flagged")
	}
}

// Test_Source_And_Test_Requirements_File_Count_Source verifies the per-package source file cap.
func Test_Source_And_Test_Requirements_File_Count_Source(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"pkg/a.go": []byte("// Package fixture is a fixture.\npackage fixture\n"),
		"pkg/b.go": []byte("package fixture\n"),
	}
	if !specification_flags(t, files, "has 2 source files") {
		t.Fatal("two small source files must be flagged")
	}
}

// Test_Source_And_Test_Requirements_File_Count_Tests verifies test files are
// counted independently from source files.
func Test_Source_And_Test_Requirements_File_Count_Tests(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"pkg/fixture.go": []byte("// Package fixture is a fixture.\npackage fixture\n"),
		"pkg/a_test.go":  []byte("package fixture_test\n"),
		"pkg/b_test.go":  []byte("package fixture_test\n"),
	}
	if !specification_flags(t, files, "has 2 test files") {
		t.Fatal("test files must be counted independently of source files")
	}
}

// Test_Source_And_Test_Requirements_File_Count_Specification_Test verifies
// specification_test.go is counted in its own independent group.
func Test_Source_And_Test_Requirements_File_Count_Specification_Test(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"pkg/fixture.go": []byte("// Package fixture is a fixture.\npackage fixture\n"),
		"pkg/specification_test.go": []byte("package fixture_test\n\n" +
			strings.Repeat("\n", 10001)),
	}
	if !specification_flags(t, files, "specification_test files") {
		t.Fatal("specification_test.go must be counted in its own independent group")
	}
}

// Test_Source_And_Test_Requirements_File_Count_Build_Tags verifies files
// sharing a build-tag constraint form an independent group.
func Test_Source_And_Test_Requirements_File_Count_Build_Tags(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"pkg/fixture.go": []byte("// Package fixture is a fixture.\npackage fixture\n"),
		"pkg/a_lin.go":   []byte("//go:build linux\n\npackage fixture\n"),
		"pkg/b_lin.go":   []byte("//go:build linux\n\npackage fixture\n"),
	}
	if !specification_flags(t, files, "under build constraint") {
		t.Fatal("files sharing a build tag must form an independent group")
	}
}

// Test_Deterministic_Entry_Format verifies an entry is an exact-path glob: a bare
// package path releases that one package and not a child — the goroutine in pkg is
// released while the select in the nested pkg/sub is still flagged — and a `**`
// entry releases the whole subtree.
func Test_Deterministic_Entry_Format(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"go.mod": []byte("module fixture\n\ngo 1.25\n"),
		"pkg/p.go": []byte("// Package p is a fixture.\n" +
			"package p\n\n" +
			"// F is a fixture.\n" +
			"func F() {\n\tgo done()\n}\n\n" +
			"func done() {\n\treturn\n}\n"),
		"pkg/sub/s.go": []byte("// Package sub is a fixture.\n" +
			"package sub\n\n" +
			"// G is a fixture.\n" +
			"func G() {\n\tselect {}\n}\n"),
	}
	exact := deterministic_self_diagnostics(t, files, []string{"pkg"})
	if specification_diagnosed(exact, "must not start a goroutine") {
		t.Fatal("an exact-path entry must release the named package")
	}
	if !specification_diagnosed(exact, "must not use select") {
		t.Fatal("an exact-path entry must not release a child package")
	}
	subtree := deterministic_self_diagnostics(t, files, []string{"pkg/**"})
	if specification_diagnosed(subtree, "must not use select") {
		t.Fatal("a ** entry must release the whole subtree")
	}
}

// Test_Deterministic_Goroutines verifies a go statement in a pure package is flagged.
func Test_Deterministic_Goroutines(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"go.mod": []byte("module fixture\n\ngo 1.25\n"),
		"pkg/p.go": []byte("// Package p is a fixture.\n" +
			"package p\n\n" +
			"// F is a fixture.\n" +
			"func F() {\n\tgo done()\n}\n\n" +
			"func done() {\n\treturn\n}\n"),
	}
	if !specification_diagnosed(deterministic_self_diagnostics(t, files, nil),
		"must not start a goroutine") {
		t.Fatal("a go statement in a deterministic package must be flagged")
	}
}

// Test_Deterministic_Channels verifies a channel in a pure package is flagged.
func Test_Deterministic_Channels(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"go.mod": []byte("module fixture\n\ngo 1.25\n"),
		"pkg/p.go": []byte("// Package p is a fixture.\n" +
			"package p\n\n" +
			"// F is a fixture.\n" +
			"func F() {\n\tc := make(chan int)\n\tclose(c)\n}\n"),
	}
	if !specification_diagnosed(deterministic_self_diagnostics(t, files, nil),
		"must not use a channel") {
		t.Fatal("a channel in a deterministic package must be flagged")
	}
}

// Test_Deterministic_Select verifies a select in a pure package is flagged.
func Test_Deterministic_Select(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"go.mod": []byte("module fixture\n\ngo 1.25\n"),
		"pkg/p.go": []byte("// Package p is a fixture.\n" +
			"package p\n\n" +
			"// F is a fixture.\n" +
			"func F() {\n\tselect {}\n}\n"),
	}
	if !specification_diagnosed(deterministic_self_diagnostics(t, files, nil),
		"must not use select") {
		t.Fatal("a select in a deterministic package must be flagged")
	}
}

// Test_Deterministic_Floats verifies a float type in a pure package is flagged.
func Test_Deterministic_Floats(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"go.mod": []byte("module fixture\n\ngo 1.25\n"),
		"pkg/p.go": []byte("// Package p is a fixture.\n" +
			"package p\n\n" +
			"// F is a fixture.\n" +
			"func F() (f float32) {\n\treturn 0\n}\n"),
	}
	if !specification_diagnosed(deterministic_self_diagnostics(t, files, nil),
		"must not use float") {
		t.Fatal("a float in a deterministic package must be flagged")
	}
}

// Test_Deterministic_Banned_Imports verifies a time import in a pure package is flagged.
func Test_Deterministic_Banned_Imports(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"go.mod": []byte("module fixture\n\ngo 1.25\n"),
		"pkg/p.go": []byte("// Package p is a fixture.\n" +
			"package p\n\n" +
			"import \"time\"\n\n" +
			"// F is a fixture.\n" +
			"func F() (d time.Duration) {\n\treturn 0\n}\n"),
	}
	if !specification_diagnosed(deterministic_self_diagnostics(t, files, nil),
		"must not import") {
		t.Fatal("a time import in a deterministic package must be flagged")
	}
}

// Test_Deterministic_Import_Induction verifies a deterministic package importing a
// pure_but_indeterministic_packages first-party package is flagged — that import is no longer
// deterministic — while a package also listed in instrumentation_packages is exempt:
// instrumentation is a write-only side channel the induction does not reach.
func Test_Deterministic_Import_Induction(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"go.mod": []byte("module fixture\n\ngo 1.25\n"),
		"pkg/p.go": []byte("// Package p is a fixture.\n" +
			"package p\n\n" +
			"import \"fixture/other\"\n\n" +
			"// F is a fixture.\n" +
			"func F() {\n\tother.G()\n}\n"),
		"other/o.go": []byte("// Package other is a fixture.\n" +
			"package other\n\n" +
			"// G is a fixture.\n" +
			"func G() {\n\treturn\n}\n"),
	}
	if !specification_diagnosed(deterministic_self_diagnostics(t, files, []string{"other"}),
		"import only deterministic packages") {
		t.Fatal("importing an opted-out first-party package must be flagged")
	}
	fsys := fstest.MapFS{}
	for name, content := range files {
		fsys[name] = &fstest.MapFile{Data: content}
	}
	exempt, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys:                     fsys,
		Scope:                    "pkg",
		Shared_Component:         doctrine_shared_component_directory,
		Pure_But_Indeterministic: []string{"other"},
		Instrumentation_Packages: []string{"other/**"},
	})
	if err != nil {
		t.Fatalf("Check_File_System: %v", err)
	}
	if specification_diagnosed(exempt, "import only deterministic packages") {
		t.Fatalf("a listed instrumentation import must be exempt; got: %v", exempt)
	}
}

// Test_Deterministic_Impurity verifies a pure_but_indeterministic_packages entry naming an
// all-impure directory — here a default tier — matches no pure package and is
// reported as a coverage gap. An impure package is never deterministic, so listing
// it releases nothing; the dead entry must fail loudly rather than pass silently.
func Test_Deterministic_Impurity(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"go.mod": []byte("module fixture\n\ngo 1.25\n"),
		"pkg/default/d.go": []byte("// Package pkg is a fixture.\n" +
			"package pkg\n"),
	}
	if !specification_diagnosed(
		deterministic_self_diagnostics(t, files, []string{"pkg/default"}),
		"no pure package found") {
		t.Fatal("an entry matching no pure package must be reported")
	}
}

// Test_Deterministic_Coverage verifies a pure_but_indeterministic_packages entry matching no
// pure package is reported.
func Test_Deterministic_Coverage(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"go.mod": []byte("module fixture\n\ngo 1.25\n"),
		"pkg/p.go": []byte("// Package p is a fixture.\n" +
			"package p\n"),
	}
	if !specification_diagnosed(
		deterministic_self_diagnostics(t, files, []string{"pkg/missing"}),
		"no pure package found") {
		t.Fatal("an entry matching no pure package must be reported")
	}
}

// Test_Deterministic_Coverage_Scope verifies the coverage check is bounded by the
// parse scope: an entry for a module outside the scope — whose files a scoped run
// never parses — is not reported as a gap, while an entry inside the scope that
// covers no pure package still is. So a scoped run flags its own typos and leaves
// other modules' entries to the whole-workspace run, which alone parses them all.
func Test_Deterministic_Coverage_Scope(t *testing.T) {
	t.Parallel()
	pure_file := "// Package fixture is a fixture.\npackage fixture\n"
	fsys := fstest.MapFS{
		"shared/go.mod": &fstest.MapFile{
			Data: []byte("module example.com/shared\n")},
		"mybinary/go.mod": &fstest.MapFile{
			Data: []byte("module example.com/mybinary\n")},
		"mybinary/internal/entry.go": &fstest.MapFile{Data: []byte(pure_file)},
		"other/go.mod": &fstest.MapFile{
			Data: []byte("module example.com/other\n")},
		"other/internal/x.go": &fstest.MapFile{Data: []byte(pure_file)},
	}
	diags, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys:             fsys,
		Shared_Component: "shared",
		Scope:            "mybinary",
		Pure_But_Indeterministic: []string{
			"mybinary/internal", "other/internal", "mybinary/internal/nonexistent"},
	})
	if err != nil {
		t.Fatalf("Check_File_System: %v", err)
	}
	if specification_diagnosed(diags, "no pure package found at \"other/internal\"") {
		t.Fatal("an out-of-scope module entry must not be reported as a coverage gap")
	}
	if !specification_diagnosed(diags,
		"no pure package found at \"mybinary/internal/nonexistent\"") {
		t.Fatal("an in-scope entry covering no pure package must still be reported")
	}
}

// Test_Stdlib_Time verifies a shared-module package importing stdlib time outside
// the time/default gateway is reported.
func Test_Stdlib_Time(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"go.mod": &fstest.MapFile{Data: []byte("module fixture\n\ngo 1.25\n")},
		"pkg/p.go": &fstest.MapFile{Data: []byte("// Package p is a fixture.\n" +
			"package p\n\n" +
			"import \"time\"\n\n" +
			"// F is a fixture.\n" +
			"func F() (d time.Duration) {\n\treturn 0\n}\n")},
	}
	diags, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys: fsys, Scope: "pkg", Shared_Component: "pkg",
	})
	if err != nil {
		t.Fatalf("Check_File_System: %v", err)
	}
	if !specification_diagnosed(diags, "may be imported only by") {
		t.Fatal("a non-gateway stdlib time import must be reported")
	}
}

// Test_Event_Loop_Driver verifies a non-main, non-test package that calls an IO loop
// constructor is flagged.
func Test_Event_Loop_Driver(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"import io \"fixture/shared/io\"\n\n" +
		"// Build makes a loop.\nfunc Build() (loop io.IO) {\n" +
		"\tl, _ := io.Sim_To_IO(nil)\n\treturn l\n}\n")
	if !specification_flags(t, files, "mints a loop driver") {
		t.Fatal("a library call to an IO loop constructor must be flagged")
	}
}

// Test_Event_Loop_Gateway verifies a non-gateway package that imports a raw IO stdlib
// package is flagged.
func Test_Event_Loop_Gateway(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\nimport \"net\"\n\n" +
		"// Dial does.\nfunc Dial() (connection net.Conn) {\n\treturn nil\n}\n")
	if !specification_flags(t, files, "route IO through shared/io") {
		t.Fatal("importing raw IO stdlib outside the gateway must be flagged")
	}
}

// Test_Event_Loop_Seed verifies a New_Sim whose parameter carries outcomes, not a seed,
// is flagged.
func Test_Event_Loop_Seed(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"// New_Sim builds a sim.\nfunc New_Sim(payloads [][]byte) (count int) {\n" +
		"\treturn len(payloads)\n}\n")
	if !specification_flags(t, files, "New_Sim takes only the seed") {
		t.Fatal("a New_Sim parameter that carries outcomes must be flagged")
	}
}

// Test_Configuration_Directory_Slash verifies a wildcard-free lint.json entry that
// names a directory must end in a slash, and one that names a file must not.
func Test_Configuration_Directory_Slash(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"go.mod":   {Data: []byte(doctrine_root_go_module)},
		"pkg/p.go": {Data: []byte("// Package p is a fixture.\npackage p\n")},
	}
	tracked := map[string]bool{"go.mod": true, "pkg/p.go": true}
	run := func(entry string) (diags []lint.Diagnostic) {
		diags, err := lint.Check_File_System(&lint.Check_File_System_Input{
			Fsys: fsys, Tracked: tracked,
			Shared_Component:         doctrine_shared_component_directory,
			Pure_But_Indeterministic: []string{entry},
		})
		if err != nil {
			t.Fatalf("Check_File_System: %v", err)
		}
		return diags
	}
	if !specification_diagnosed(run("pkg"), "names a directory") {
		t.Fatal("a directory entry without a trailing slash must be flagged")
	}
	if specification_diagnosed(run("pkg/"), "names a directory") {
		t.Fatal("a directory entry with a trailing slash must pass")
	}
	if !specification_diagnosed(run("pkg/p.go/"), "names a file") {
		t.Fatal("a file entry with a trailing slash must be flagged")
	}
}

// Test_Configuration_Packages_Only verifies an exact-file entry is flagged in every
// lint.json glob list except ignore and opt_out_recursion_ban, and that a wildcard-free
// entry matching neither a tracked file nor a directory is flagged as a coverage gap.
func Test_Configuration_Packages_Only(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"go.mod":   {Data: []byte(doctrine_root_go_module)},
		"pkg/p.go": {Data: []byte("// Package p is a fixture.\npackage p\n")},
	}
	tracked := map[string]bool{"go.mod": true, "pkg/p.go": true}
	run := func(input *lint.Check_File_System_Input) (diags []lint.Diagnostic) {
		input.Fsys = fsys
		input.Tracked = tracked
		input.Shared_Component = doctrine_shared_component_directory
		diags, err := lint.Check_File_System(input)
		if err != nil {
			t.Fatalf("Check_File_System: %v", err)
		}
		return diags
	}
	if !specification_diagnosed(
		run(&lint.Check_File_System_Input{Pure_But_Indeterministic: []string{"pkg/p.go"}}),
		"may name a file") {
		t.Fatal("an exact-file entry in pure_but_indeterministic_packages must be flagged")
	}
	if !specification_diagnosed(
		run(&lint.Check_File_System_Input{Instrumentation_Packages: []string{"pkg/p.go"}}),
		"may name a file") {
		t.Fatal("an exact-file entry in instrumentation_packages must be flagged")
	}
	if !specification_diagnosed(
		run(&lint.Check_File_System_Input{Invariant_Exempt_Packages: []string{"pkg/p.go"}}),
		"may name a file") {
		t.Fatal("an exact-file entry in opt_out_assertion_mandate_packages must be flagged")
	}
	if specification_diagnosed(
		run(&lint.Check_File_System_Input{Ignore: []string{"pkg/p.go"}}),
		"may name a file") {
		t.Fatal("an exact-file ignore entry must be allowed")
	}
	if specification_diagnosed(
		run(&lint.Check_File_System_Input{Recursion_Exempt: []string{"pkg/p.go"}}),
		"may name a file") {
		t.Fatal("an exact-file opt_out_recursion_ban entry must be allowed")
	}
	if !specification_diagnosed(
		run(&lint.Check_File_System_Input{Ignore: []string{"pkg/missing"}}),
		"matches no tracked file or directory") {
		t.Fatal("an entry matching neither a file nor a directory must be a coverage gap")
	}
}

// Test_Driver_Gateway_Main_Allowed verifies package main may construct a loop.
func Test_Driver_Gateway_Main_Allowed(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"pkg/main.go": []byte("// Package main is a fixture.\npackage main\n\n" +
			"import io \"fixture/shared/io\"\n\n" +
			"func main() {\n\tio.Sim_To_IO(nil)\n}\n"),
	}
	if specification_flags(t, files, "mints a loop driver") {
		t.Fatal("package main must be allowed to construct a loop driver")
	}
}

// Test_Driver_Gateway_Test_Allowed verifies a test may construct a loop.
func Test_Driver_Gateway_Test_Allowed(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"pkg/rule.go": []byte("// Package fixture is a fixture.\npackage fixture\n"),
		"pkg/rule_test.go": []byte("package fixture_test\n\n" +
			"import (\n\t\"testing\"\n\n" +
			"\tiodefault \"fixture/shared/io/default\"\n)\n\n" +
			"// Test_X is a fixture.\nfunc Test_X(t *testing.T) {\n" +
			"\tiodefault.New_Operating_System_IO(nil)\n}\n"),
	}
	if specification_flags(t, files, "mints a loop driver") {
		t.Fatal("a test may construct a loop driver")
	}
}

// Test_Driver_Gateway_Logical_Clock_Allowed verifies constructing the virtual (logical)
// clock is not gated: the clock is read-only, and its tick drives a loop only through a
// Driver, which is gated separately.
func Test_Driver_Gateway_Logical_Clock_Allowed(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"import time \"fixture/shared/time\"\n\n" +
		"// Build makes a clock.\nfunc Build() (clock time.Clock) {\n" +
		"\tc, _ := time.Virtual_Clock_To_Clock(time.Virtual_Clock{})\n\treturn c\n}\n")
	if specification_flags(t, files, "mints a loop driver") {
		t.Fatal("the read-only logical clock constructor must not be gated")
	}
}

// Test_Driver_Gateway_Type_Flagged verifies a non-main package that names the io.Driver
// type is flagged — internal takes io.IO, and only the harness holds the Driver.
func Test_Driver_Gateway_Type_Flagged(t *testing.T) {
	t.Parallel()
	library := "// Package io is a fixture.\npackage io\n\n" +
		"// Driver drives.\ntype Driver struct{}\n"
	consumer := "// Package fixture is a fixture.\npackage fixture\n\n" +
		"import io \"github.com/james-orcales/james-orcales/shared/io\"\n\n" +
		"// Main drives.\nfunc Main(driver io.Driver) {}\n"
	fsys := fstest.MapFS{
		"go.mod":          &fstest.MapFile{Data: []byte(doctrine_root_go_module)},
		"shared/io/io.go": &fstest.MapFile{Data: []byte(library)},
		"pkg/rule.go":     &fstest.MapFile{Data: []byte(consumer)},
	}
	diags, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys:             fsys,
		Scope:            "pkg",
		Shared_Component: doctrine_shared_component_directory,
	})
	if err != nil {
		t.Fatalf("Check_File_System: %v", err)
	}
	if !specification_diagnosed(diags, "io.Driver may be held only") {
		t.Fatal("naming io.Driver outside main or a test must be flagged")
	}
}

// Test_Driver_Gateway_Type_Main_Allowed verifies package main may hold the io.Driver type.
func Test_Driver_Gateway_Type_Main_Allowed(t *testing.T) {
	t.Parallel()
	library := "// Package io is a fixture.\npackage io\n\n" +
		"// Driver drives.\ntype Driver struct{}\n"
	consumer := "// Package main is a fixture.\npackage main\n\n" +
		"import io \"github.com/james-orcales/james-orcales/shared/io\"\n\n" +
		"// hold takes a driver.\nfunc hold(driver io.Driver) {}\n\nfunc main() {}\n"
	fsys := fstest.MapFS{
		"go.mod":          &fstest.MapFile{Data: []byte(doctrine_root_go_module)},
		"shared/io/io.go": &fstest.MapFile{Data: []byte(library)},
		"pkg/main.go":     &fstest.MapFile{Data: []byte(consumer)},
	}
	diags, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys:             fsys,
		Scope:            "pkg",
		Shared_Component: doctrine_shared_component_directory,
	})
	if err != nil {
		t.Fatalf("Check_File_System: %v", err)
	}
	if specification_diagnosed(diags, "io.Driver may be held only") {
		t.Fatal("package main may hold io.Driver")
	}
}

// Test_IO_Gateway_Call verifies an os file-operation call outside the gateway is flagged.
func Test_IO_Gateway_Call(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\nimport \"os\"\n\n" +
		"// F opens.\nfunc F() (file *os.File, err error) {\n\treturn os.Open(\"x\")\n}\n")
	if !specification_flags(t, files, "os.Open does raw IO") {
		t.Fatal("os.Open outside the gateway must be flagged")
	}
}

// Test_IO_Gateway_Process_Args verifies os.Args, not raw IO, is left alone.
func Test_IO_Gateway_Process_Args(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\nimport \"os\"\n\n" +
		"// F counts args.\nfunc F() (count int) {\n\treturn len(os.Args)\n}\n")
	if specification_flags(t, files, "does raw IO") {
		t.Fatal("os.Args is not raw IO and must be allowed")
	}
}

// Test_IO_Gateway_Test_Exempt verifies a test may import raw IO stdlib.
func Test_IO_Gateway_Test_Exempt(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"pkg/rule.go": []byte("// Package fixture is a fixture.\npackage fixture\n"),
		"pkg/rule_test.go": []byte("package fixture_test\n\n" +
			"import (\n\t\"net\"\n\t\"testing\"\n)\n\n// Test_X is a fixture.\n" +
			"func Test_X(t *testing.T) {\n\tnet.ParseIP(\"\")\n}\n"),
	}
	if specification_flags(t, files, "route IO through shared/io") {
		t.Fatal("a test may import raw IO stdlib")
	}
}

// Test_IO_Gateway_Instrumentation_Exempt verifies an instrumentation package may do
// raw IO — it is the diagnostics side channel.
func Test_IO_Gateway_Instrumentation_Exempt(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"go.mod": &fstest.MapFile{Data: []byte(doctrine_root_go_module)},
		"pkg/rule.go": &fstest.MapFile{Data: []byte(
			"// Package fixture is a fixture.\npackage fixture\n\nimport \"net\"\n\n" +
				"// Dial does.\nfunc Dial() (connection net.Conn) {\n" +
				"\treturn nil\n}\n")},
	}
	diags, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys:                     fsys,
		Scope:                    "pkg",
		Shared_Component:         doctrine_shared_component_directory,
		Instrumentation_Packages: []string{"pkg/**"},
	})
	if err != nil {
		t.Fatalf("Check_File_System: %v", err)
	}
	if specification_diagnosed(diags, "route IO through shared/io") {
		t.Fatal("an instrumentation package may do raw IO")
	}
}

// Test_IO_Gateway_Main_Exempt verifies package main may do raw IO — the un-simulated
// wiring shell the framework never witnesses.
func Test_IO_Gateway_Main_Exempt(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"pkg/main.go": []byte("// Package main is a fixture.\npackage main\n\n" +
			"import \"net\"\n\n" +
			"// main dials.\nfunc main() {\n\tnet.ParseIP(\"\")\n}\n"),
	}
	if specification_flags(t, files, "route IO through shared/io") {
		t.Fatal("package main may do raw IO")
	}
}

// Test_IO_Gateway_Time_Exempt verifies the time/default clock gateway may import
// syscall — the clock is the one capability the loop is built on, not IO it carries.
func Test_IO_Gateway_Time_Exempt(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"go.mod": &fstest.MapFile{Data: []byte(doctrine_root_go_module)},
		"shared/time/default/system.go": &fstest.MapFile{Data: []byte(
			"// Package time is a fixture.\npackage time\n\nimport \"syscall\"\n\n" +
				"// Now reads the clock.\nfunc Now() (n int64) {\n" +
				"\tstamp := syscall.Timespec{}\n\treturn stamp.Sec\n}\n")},
	}
	diags, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys:             fsys,
		Scope:            "shared/time/default",
		Shared_Component: doctrine_shared_component_directory,
	})
	if err != nil {
		t.Fatalf("Check_File_System: %v", err)
	}
	if specification_diagnosed(diags, "route IO through shared/io") {
		t.Fatal("the time/default clock gateway may import syscall")
	}
}

// Test_Sim_Script_Seed_Allowed verifies New_Sim taking only a seed is not flagged.
func Test_Sim_Script_Seed_Allowed(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"// New_Sim builds a sim.\nfunc New_Sim(seed uint64) (count int) { return 0 }\n")
	if specification_flags(t, files, "New_Sim takes only the seed") {
		t.Fatal("New_Sim taking only a seed must be allowed")
	}
}

// Test_Sim_Script_Export_Flagged verifies an exported Sim_ function is flagged.
func Test_Sim_Script_Export_Flagged(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"// New_Sim builds a sim.\nfunc New_Sim(seed uint64) (count int) { return 0 }\n\n" +
		"// Sim_Raise scripts a signal.\nfunc Sim_Raise(signal int) {}\n")
	if !specification_flags(t, files, "is a scripting entry") {
		t.Fatal("an exported Sim_ function must be flagged")
	}
}

// Test_Primitive_Field_Flagged verifies a raw slice struct field is flagged.
func Test_Primitive_Field_Flagged(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"// Bag is a fixture.\ntype Bag struct {\n" +
		"\t// Items is a fixture.\n\tItems []int\n}\n")
	if !specification_flags(t, files, "raw slice field") {
		t.Fatal("a raw slice field must be flagged")
	}
}

// Test_Primitive_Defined_Type_Passes verifies a defined wrapper over a primitive
// is allowed in a signature.
func Test_Primitive_Defined_Type_Passes(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"// Name is a fixture.\ntype Name string\n\n" +
		"// Greet does.\nfunc Greet(who Name) {\n}\n")
	if specification_flags(t, files, "raw ") {
		t.Fatal("a defined type wrapping a primitive must not be flagged")
	}
}

// Test_Primitive_Stdlib_Method_Exempt verifies a method satisfying a stdlib
// interface keeps its dictated primitive signature.
func Test_Primitive_Stdlib_Method_Exempt(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"// Level is a fixture.\ntype Level int\n\n" +
		"// String is a fixture.\n" +
		"func (l Level) String() (name string) {\n\treturn \"\"\n}\n")
	if specification_flags(t, files, "raw ") {
		t.Fatal("a stdlib-interface method must keep its primitive signature")
	}
}

// Test_Specification_Baseline pins that the unmutated package is clean, so every
// other test isolates the single rule it violates.
func Test_Specification_Baseline(t *testing.T) {
	t.Parallel()
	for _, diagnostic := range specification_self_diagnostics(t, specification_baseline(t)) {
		if diagnostic.Name == "specification" {
			t.Errorf("baseline has a specification diagnostic: %s", diagnostic.Message)
		}
		if strings.HasSuffix(diagnostic.Position.Filename, "SPECIFICATION.md") {
			t.Errorf("baseline flags the spec file: %s", diagnostic.Message)
		}
	}
}

// Runs the linter over a lone in-memory symlink and returns its diagnostics.
// Readlink is injected because fstest.MapFS cannot represent a symlink's target,
// and Tracked drives the tracked-membership the rule checks — the target need
// not exist in the fixture, only in Tracked.
func symlink_lint(
	t *testing.T, target string, read_err error, tracked map[string]bool,
) (diags []lint.Diagnostic) {
	t.Helper()
	diags, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys: fstest.MapFS{
			// Data is a fixed placeholder, not the target under test: fstest.MapFS
			// resolves a self-referential or cyclic symlink into an infinite loop,
			// and the rule reads the target through the injected Readlink below, not
			// from the fixture's bytes.
			"link": &fstest.MapFile{Mode: fs.ModeSymlink, Data: []byte("placeholder")},
		},
		Root_Directory: "/root",
		Tracked:        tracked,
		Readlink: func(_ string) (link_target string, link_err error) {
			return target, read_err
		},
	})
	if err != nil {
		t.Fatalf("Check_File_System: %v", err)
	}
	return diags
}

func specification_baseline(t *testing.T) (files map[string][]byte) {
	t.Helper()
	// One # leaf and its single matching test — clean under every rule, so a
	// Specification-section test can knock exactly one rule out of true and watch
	// the tool catch it. The leading blank line satisfies the blank-before-heading
	// rule for the first heading. A two-word leaf keeps the name-normalization
	// test honest (Sole Rule -> Sole_Rule -> Test_Sole_Rule).
	const markdown = "\n# Sole Rule\n\nThe sole rule.\n"
	const test_source = "package fixture_test\n\n" +
		"import \"testing\"\n\n" +
		"// Test_Sole_Rule checks the sole rule.\n" +
		"func Test_Sole_Rule(t *testing.T) {\n\tt.Parallel()\n}\n"
	return map[string][]byte{
		"go.mod": []byte("module fixture\n\ngo 1.25\n"),
		"pkg/fixture.go": []byte("// Package fixture is a fixture.\n" +
			"package fixture\n"),
		"pkg/SPECIFICATION.md":      []byte(markdown),
		"pkg/specification_test.go": []byte(test_source),
	}
}

// Builds an in-memory filesystem from the fixture files, runs the linter scoped
// to the fixture package, and returns its diagnostics.
func specification_self_diagnostics(
	t *testing.T, files map[string][]byte,
) (diags []lint.Diagnostic) {
	t.Helper()
	fsys := fstest.MapFS{}
	for name, content := range files {
		fsys[name] = &fstest.MapFile{Data: content}
	}
	if _, present := fsys["go.mod"]; !present {
		fsys["go.mod"] = &fstest.MapFile{Data: []byte(doctrine_root_go_module)}
	}
	diags, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys: fsys, Scope: "pkg", Shared_Component: doctrine_shared_component_directory,
		Word_Replacements: test_word_replacements()})
	if err != nil {
		t.Fatalf("Check_File_System: %v", err)
	}
	return diags
}

func specification_diagnosed(diags []lint.Diagnostic, fragment string) (found bool) {
	for _, diagnostic := range diags {
		if strings.Contains(diagnostic.Message, fragment) {
			return true
		}
	}
	return false
}

// Reports whether linting the given fixture files produces a diagnostic whose
// message contains the fragment — the delegation the Go-code rule tests use to
// confirm the tool enforces the rule the heading documents.
func specification_flags(
	t *testing.T, files map[string][]byte, fragment string,
) (found bool) {
	t.Helper()
	return specification_diagnosed(specification_self_diagnostics(t, files), fragment)
}

// Lints the fixture at whole-workspace scope and reports whether some diagnostic
// message contains the fragment. The repository-wide stream checks (banned files,
// conflict markers, workflow pins, agent-doc pairing) are exercised whole-tree —
// the way a whole-repo run reaches them — since a scoped run, by design, examines
// only its own subtree and so would never visit a fixture placed elsewhere.
func repository_flags(
	t *testing.T, files map[string][]byte, fragment string,
) (found bool) {
	t.Helper()
	fsys := fstest.MapFS{}
	for name, content := range files {
		fsys[name] = &fstest.MapFile{Data: content}
	}
	diags, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys: fsys, Shared_Component: doctrine_shared_component_directory,
		Word_Replacements: test_word_replacements()})
	if err != nil {
		t.Fatalf("Check_File_System: %v", err)
	}
	return specification_diagnosed(diags, fragment)
}

// Wraps one Go source string as the sole file of a fixture package. Kept
// separate from the fragment assertion so no helper takes two string parameters
// (which the input-struct rule would reject).
func specification_one_file(source string) (files map[string][]byte) {
	return map[string][]byte{"pkg/rule.go": []byte(source)}
}

func specification_markdown_append(files map[string][]byte, text string) {
	files["pkg/SPECIFICATION.md"] = append(files["pkg/SPECIFICATION.md"], text...)
}

func specification_markdown_prepend(files map[string][]byte, text string) {
	files["pkg/SPECIFICATION.md"] = append([]byte(text), files["pkg/SPECIFICATION.md"]...)
}

// Test_Source_And_Test_Bans_Recursion_Exempt verifies a recursive function in a
// package listed in opt_out_recursion_ban is not flagged.
func Test_Source_And_Test_Bans_Recursion_Exempt(t *testing.T) {
	t.Parallel()
	files := specification_one_file(
		"package fixture\n\n// F loops.\nfunc F() {\n\tF()\n}\n")
	diags := recursion_exempt_self_diagnostics(t, files, []string{"pkg/**"})
	if specification_diagnosed(diags, "calls itself") {
		t.Fatal("recursion in an exempt package must not be flagged")
	}
}

// Test_Source_And_Test_Bans_Recursion_Exempt_File verifies a recursive function in a
// file listed in opt_out_recursion_ban by its exact path is not flagged, proving the
// exemption works at file granularity, not just package granularity.
func Test_Source_And_Test_Bans_Recursion_Exempt_File(t *testing.T) {
	t.Parallel()
	files := specification_one_file(
		"package fixture\n\n// F loops.\nfunc F() {\n\tF()\n}\n")
	diags := recursion_exempt_self_diagnostics(t, files, []string{"pkg/rule.go"})
	if specification_diagnosed(diags, "calls itself") {
		t.Fatal("recursion in an exempt file must not be flagged")
	}
}

// Mirrors specification_self_diagnostics but threads opt_out_recursion_ban, the
// recursion ban's escape hatch for a recursive-descent parser package.
func recursion_exempt_self_diagnostics(
	t *testing.T, files map[string][]byte, exempt []string,
) (diags []lint.Diagnostic) {
	t.Helper()
	fsys := fstest.MapFS{}
	for name, content := range files {
		fsys[name] = &fstest.MapFile{Data: content}
	}
	diags, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys:             fsys,
		Scope:            "pkg",
		Shared_Component: doctrine_shared_component_directory,
		Recursion_Exempt: exempt,
	})
	if err != nil {
		t.Fatalf("Check_File_System: %v", err)
	}
	return diags
}

// Runs the linter over the fixture with the given package directories opted into
// the deterministic tier, returning its diagnostics. Mirrors
// specification_self_diagnostics but threads Pure_But_Indeterministic (the
// deterministic tier's opt-out list), which specification_flags does not carry.
func deterministic_self_diagnostics(
	t *testing.T, files map[string][]byte, exempt []string,
) (diags []lint.Diagnostic) {
	t.Helper()
	fsys := fstest.MapFS{}
	for name, content := range files {
		fsys[name] = &fstest.MapFile{Data: content}
	}
	diags, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys:                     fsys,
		Scope:                    "pkg",
		Shared_Component:         doctrine_shared_component_directory,
		Pure_But_Indeterministic: exempt,
	})
	if err != nil {
		t.Fatalf("Check_File_System: %v", err)
	}
	return diags
}

// Mirrors specification_self_diagnostics but threads opt_out_assertion_mandate_packages,
// the type-invariant rule's opt-out.
func invariant_exempt_self_diagnostics(
	t *testing.T, files map[string][]byte, exempt []string,
) (diags []lint.Diagnostic) {
	t.Helper()
	fsys := fstest.MapFS{}
	for name, content := range files {
		fsys[name] = &fstest.MapFile{Data: content}
	}
	diags, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys:                      fsys,
		Scope:                     "pkg",
		Shared_Component:          doctrine_shared_component_directory,
		Word_Replacements:         test_word_replacements(),
		Invariant_Exempt_Packages: exempt,
	})
	if err != nil {
		t.Fatalf("Check_File_System: %v", err)
	}
	return diags
}

// Test_Type_Invariant_Exempt_List_Skips_Package verifies a package listed in
// opt_out_assertion_mandate_packages is skipped by the type-invariant rule.
func Test_Type_Invariant_Exempt_List_Skips_Package(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"// Widget is a fixture.\ntype Widget struct {\n\t// X is a fixture.\n\tX int\n}\n")
	diags := invariant_exempt_self_diagnostics(t, files, []string{"pkg/**"})
	if specification_diagnosed(diags, "directly below Widget") {
		t.Fatal("a type in an exempt package must not be flagged")
	}
}

// Test_Type_Invariant_Clean_Pair_Passes verifies a type immediately followed by a
// correctly-signed bundle is not flagged.
func Test_Type_Invariant_Clean_Pair_Passes(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"import \"fixture/shared/invariant\"\n\n" +
		"// Widget is a fixture.\ntype Widget struct {\n" +
		"\t// X is a fixture.\n\tX int\n}\n\n" +
		"// Widget_Invariants is a fixture.\n" +
		"func Widget_Invariants(w Widget, namespace invariant.Namespace) {\n" +
		"\tprintln(0)\n}\n")
	if specification_flags(t, files, "directly below Widget") {
		t.Fatal("a well-formed pair must not be flagged")
	}
	if specification_flags(t, files, "must take") {
		t.Fatal("a well-formed signature must not be flagged")
	}
}

// Test_Type_Invariant_Pointer_First_Parameter_Passes verifies a pointer-typed
// first parameter satisfies the signature.
func Test_Type_Invariant_Pointer_First_Parameter_Passes(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"import \"fixture/shared/invariant\"\n\n" +
		"// Widget is a fixture.\ntype Widget struct {\n" +
		"\t// X is a fixture.\n\tX int\n}\n\n" +
		"// Widget_Invariants is a fixture.\n" +
		"func Widget_Invariants(w *Widget, namespace invariant.Namespace) {\n" +
		"\tprintln(0)\n}\n")
	if specification_flags(t, files, "must take") {
		t.Fatal("a pointer first parameter must satisfy the signature")
	}
}

// Test_Type_Invariant_Generic_Passes verifies a generic type whose bundle's type
// parameters match is not flagged.
func Test_Type_Invariant_Generic_Passes(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"import \"fixture/shared/invariant\"\n\n" +
		"// Box is a fixture.\ntype Box[T any] struct {\n" +
		"\t// Item is a fixture.\n\tItem T\n}\n\n" +
		"// Box_Invariants is a fixture.\n" +
		"func Box_Invariants[T any](b Box[T], namespace invariant.Namespace) {\n" +
		"\tprintln(0)\n}\n")
	if specification_flags(t, files, "directly below Box") {
		t.Fatal("a matching generic bundle must not be flagged")
	}
	if specification_flags(t, files, "must take") {
		t.Fatal("a matching generic signature must not be flagged")
	}
}

// Test_Type_Invariant_Exempt_Kinds_Pass verifies aliases, function types, and
// empty structs need no bundle.
func Test_Type_Invariant_Exempt_Kinds_Pass(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"// Alias is a fixture.\ntype Alias = int\n\n" +
		"// Callback is a fixture.\ntype Callback func()\n\n" +
		"// Marker is a fixture.\ntype Marker struct{}\n")
	if specification_flags(t, files, "directly below") {
		t.Fatal("aliases, function types, and empty structs are exempt")
	}
}

// Test_Type_Invariant_Between_Input_Struct_And_Function verifies the amended
// input-struct rule: a bundle may sit between an input struct and the function it
// feeds, satisfying both the locality rule and the type-invariant rule.
func Test_Type_Invariant_Between_Input_Struct_And_Function(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"import \"fixture/shared/invariant\"\n\n" +
		"// Foo_Input is a fixture.\ntype Foo_Input struct {\n" +
		"\t// A is a fixture.\n\tA int\n\t// B is a fixture.\n\tB int\n}\n\n" +
		"// Foo_Input_Invariants is a fixture.\n" +
		"func Foo_Input_Invariants(input Foo_Input, namespace invariant.Namespace) {\n" +
		"\tprintln(0)\n}\n\n" +
		"// Foo does.\nfunc Foo(input *Foo_Input) (n int) {\n" +
		"\treturn input.A + input.B\n}\n")
	if specification_flags(t, files, "directly above") {
		t.Fatal("the input struct may be parted from its function by its invariant")
	}
	if specification_flags(t, files, "directly below Foo_Input") {
		t.Fatal("the bundle directly below the input struct satisfies the rule")
	}
}

// Test_Type_Invariant_Numeric_Signed_Passes verifies a complete signed numeric
// bundle (const bounds, all six boundary claims) is not flagged.
func Test_Type_Invariant_Numeric_Signed_Passes(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"import \"fixture/shared/invariant\"\n\n" +
		"const Level_Max Level = 7\n\nconst Level_Min Level = -1\n\n" +
		"// Level is a fixture.\ntype Level int8\n\n" +
		"// Level_Invariants is a fixture.\n" +
		"func Level_Invariants(v Level, namespace invariant.Namespace) {\n" +
		"\tinvariant.Always(v <= Level_Max, \"max bound\")\n" +
		"\tinvariant.Always(v >= Level_Min, \"min bound\")\n" +
		"\tinvariant.Dot_Product(namespace,\n" +
		"\t\tinvariant.Sometimes(v == Level_Max, \"max\"),\n" +
		"\t\tinvariant.Sometimes(v == Level_Min, \"min\"),\n" +
		"\t\tinvariant.Sometimes(v == 0, \"zero\"),\n" +
		"\t\tinvariant.Sometimes(v == 1, \"one\"),\n" +
		"\t\tinvariant.Sometimes(v == -1, \"neg one\"),\n" +
		"\t\tinvariant.Sometimes(v == 2, \"two\"),\n" +
		"\t)\n}\n")
	if specification_flags(t, files, "must guard both ends") {
		t.Fatal("a complete signed bundle must not be flagged for bounds")
	}
	if specification_flags(t, files, "must be a package-level constant") {
		t.Fatal("named const bounds must not be flagged")
	}
	if specification_flags(t, files, "must claim") {
		t.Fatal("a complete signed bundle must not be flagged for coverage")
	}
}

// Test_Type_Invariant_Numeric_Unsigned_Passes verifies a complete unsigned bundle
// (no -1 claim, MIN a named const) is not flagged.
func Test_Type_Invariant_Numeric_Unsigned_Passes(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"import \"fixture/shared/invariant\"\n\n" +
		"const Count_Max Count = 7\n\nconst Count_Min Count = 0\n\n" +
		"// Count is a fixture.\ntype Count uint8\n\n" +
		"// Count_Invariants is a fixture.\n" +
		"func Count_Invariants(v Count, namespace invariant.Namespace) {\n" +
		"\tinvariant.Always(v <= Count_Max, \"max bound\")\n" +
		"\tinvariant.Always(v >= Count_Min, \"min bound\")\n" +
		"\tinvariant.Dot_Product(namespace,\n" +
		"\t\tinvariant.Sometimes(v == Count_Max, \"max\"),\n" +
		"\t\tinvariant.Sometimes(v == Count_Min, \"min\"),\n" +
		"\t\tinvariant.Sometimes(v == 0, \"zero\"),\n" +
		"\t\tinvariant.Sometimes(v == 1, \"one\"),\n" +
		"\t\tinvariant.Sometimes(v == 2, \"two\"),\n" +
		"\t)\n}\n")
	if specification_flags(t, files, "must claim") {
		t.Fatal("a complete unsigned bundle (no -1) must not be flagged for coverage")
	}
	if specification_flags(t, files, "must guard both ends") {
		t.Fatal("unsigned bounds must not be flagged")
	}
}

// Test_Type_Invariant_Numeric_Float_Passes verifies a float bundle with NaN/+-Inf
// claims and const Always bounds is not flagged.
func Test_Type_Invariant_Numeric_Float_Passes(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"import \"fixture/shared/invariant\"\n\nimport \"math\"\n\n" +
		"const Scale_Max Scale = 100\n\nconst Scale_Min Scale = -100\n\n" +
		"// Scale is a fixture.\ntype Scale float64\n\n" +
		"// Scale_Invariants is a fixture.\n" +
		"func Scale_Invariants(v Scale, namespace invariant.Namespace) {\n" +
		"\tinvariant.Always(v <= Scale_Max, \"max bound\")\n" +
		"\tinvariant.Always(v >= Scale_Min, \"min bound\")\n" +
		"\tinvariant.Dot_Product(namespace,\n" +
		"\t\tinvariant.Sometimes(math.IsNaN(float64(v)), \"nan\"),\n" +
		"\t\tinvariant.Sometimes(float64(v) == math.Inf(-1), \"neg inf\"),\n" +
		"\t\tinvariant.Sometimes(float64(v) == math.Inf(1), \"pos inf\"),\n" +
		"\t)\n}\n")
	if specification_flags(t, files, "must claim") {
		t.Fatal("a complete float bundle (NaN/+-Inf) must not be flagged for coverage")
	}
	if specification_flags(t, files, "must guard both ends") {
		t.Fatal("float const bounds must not be flagged")
	}
}

// Test_Type_Invariant_Count_String_Passes verifies a complete string length
// bundle is not flagged.
func Test_Type_Invariant_Count_String_Passes(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"import \"fixture/shared/invariant\"\n\n" +
		"const Name_Max = 32\n\nconst Name_Min = 0\n\n" +
		"// Name is a fixture.\ntype Name string\n\n" +
		"// Name_Invariants is a fixture.\n" +
		"func Name_Invariants(v Name, namespace invariant.Namespace) {\n" +
		"\tinvariant.Always(len(v) <= Name_Max, \"max bound\")\n" +
		"\tinvariant.Always(len(v) >= Name_Min, \"min bound\")\n" +
		"\tinvariant.Dot_Product(namespace,\n" +
		"\t\tinvariant.Sometimes(len(v) == Name_Max, \"max\"),\n" +
		"\t\tinvariant.Sometimes(len(v) == Name_Min, \"min\"),\n" +
		"\t\tinvariant.Sometimes(len(v) == 0, \"zero\"),\n" +
		"\t\tinvariant.Sometimes(len(v) == 1, \"one\"),\n" +
		"\t\tinvariant.Sometimes(len(v) == 2, \"two\"),\n" +
		"\t)\n}\n")
	if specification_flags(t, files, "must guard both ends") {
		t.Fatal("a complete string length bundle must not be flagged for bounds")
	}
	if specification_flags(t, files, "must claim") {
		t.Fatal("a complete string length bundle must not be flagged for coverage")
	}
}

// Test_Type_Invariant_Count_Slice_Passes verifies a complete slice length bundle
// is not flagged.
func Test_Type_Invariant_Count_Slice_Passes(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"import \"fixture/shared/invariant\"\n\n" +
		"const Buffer_Max = 64\n\nconst Buffer_Min = 0\n\n" +
		"// Buffer is a fixture.\ntype Buffer []byte\n\n" +
		"// Buffer_Invariants is a fixture.\n" +
		"func Buffer_Invariants(v Buffer, namespace invariant.Namespace) {\n" +
		"\tinvariant.Always(len(v) <= Buffer_Max, \"max bound\")\n" +
		"\tinvariant.Always(len(v) >= Buffer_Min, \"min bound\")\n" +
		"\tinvariant.Dot_Product(namespace,\n" +
		"\t\tinvariant.Sometimes(len(v) == Buffer_Max, \"max\"),\n" +
		"\t\tinvariant.Sometimes(len(v) == Buffer_Min, \"min\"),\n" +
		"\t\tinvariant.Sometimes(len(v) == 0, \"zero\"),\n" +
		"\t\tinvariant.Sometimes(len(v) == 1, \"one\"),\n" +
		"\t\tinvariant.Sometimes(len(v) == 2, \"two\"),\n" +
		"\t)\n}\n")
	if specification_flags(t, files, "must guard both ends") {
		t.Fatal("a complete slice length bundle must not be flagged for bounds")
	}
	if specification_flags(t, files, "must claim") {
		t.Fatal("a complete slice length bundle must not be flagged for coverage")
	}
}

// Test_Type_Invariant_Count_Bound_Not_Claimed verifies a length bundle whose
// MAX/MIN appear only in the Always guards — with no boundary claim for them — is
// not flagged: the guard is the bound, not a claimed value.
func Test_Type_Invariant_Count_Bound_Not_Claimed(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"import \"fixture/shared/invariant\"\n\n" +
		"const Buffer_Max = 64\n\nconst Buffer_Min = 0\n\n" +
		"// Buffer is a fixture.\ntype Buffer []byte\n\n" +
		"// Buffer_Invariants is a fixture.\n" +
		"func Buffer_Invariants(v Buffer, namespace invariant.Namespace) {\n" +
		"\tinvariant.Always(len(v) <= Buffer_Max, \"max bound\")\n" +
		"\tinvariant.Always(len(v) >= Buffer_Min, \"min bound\")\n" +
		"\tinvariant.Dot_Product(namespace,\n" +
		"\t\tinvariant.Sometimes(len(v) == 0, \"zero\"),\n" +
		"\t\tinvariant.Sometimes(len(v) == 1, \"one\"),\n" +
		"\t\tinvariant.Sometimes(len(v) == 2, \"two\"),\n" +
		"\t)\n}\n")
	if specification_flags(t, files, "must claim") {
		t.Fatal("a length bundle need not claim its MAX/MIN — the guard is the bound")
	}
}

// Test_Type_Invariant_Count_Map_Passes verifies a complete map length bundle is
// not flagged.
func Test_Type_Invariant_Count_Map_Passes(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"import \"fixture/shared/invariant\"\n\n" +
		"const Registry_Max = 16\n\nconst Registry_Min = 0\n\n" +
		"// Registry is a fixture.\ntype Registry map[string]int\n\n" +
		"// Registry_Invariants is a fixture.\n" +
		"func Registry_Invariants(v Registry, namespace invariant.Namespace) {\n" +
		"\tinvariant.Always(len(v) <= Registry_Max, \"max bound\")\n" +
		"\tinvariant.Always(len(v) >= Registry_Min, \"min bound\")\n" +
		"\tinvariant.Dot_Product(namespace,\n" +
		"\t\tinvariant.Sometimes(len(v) == Registry_Max, \"max\"),\n" +
		"\t\tinvariant.Sometimes(len(v) == Registry_Min, \"min\"),\n" +
		"\t\tinvariant.Sometimes(len(v) == 0, \"zero\"),\n" +
		"\t\tinvariant.Sometimes(len(v) == 1, \"one\"),\n" +
		"\t\tinvariant.Sometimes(len(v) == 2, \"two\"),\n" +
		"\t)\n}\n")
	if specification_flags(t, files, "must guard both ends") {
		t.Fatal("a complete map length bundle must not be flagged for bounds")
	}
	if specification_flags(t, files, "must claim") {
		t.Fatal("a complete map length bundle must not be flagged for coverage")
	}
}

// Test_Type_Invariant_Count_Generic_Passes verifies a complete generic-container
// length bundle (a generic slice) is not flagged.
func Test_Type_Invariant_Count_Generic_Passes(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"import \"fixture/shared/invariant\"\n\n" +
		"const Stack_Max = 8\n\nconst Stack_Min = 0\n\n" +
		"// Stack is a fixture.\ntype Stack[T any] []T\n\n" +
		"// Stack_Invariants is a fixture.\n" +
		"func Stack_Invariants[T any](v Stack[T], namespace invariant.Namespace) {\n" +
		"\tinvariant.Always(len(v) <= Stack_Max, \"max bound\")\n" +
		"\tinvariant.Always(len(v) >= Stack_Min, \"min bound\")\n" +
		"\tinvariant.Dot_Product(namespace,\n" +
		"\t\tinvariant.Sometimes(len(v) == Stack_Max, \"max\"),\n" +
		"\t\tinvariant.Sometimes(len(v) == Stack_Min, \"min\"),\n" +
		"\t\tinvariant.Sometimes(len(v) == 0, \"zero\"),\n" +
		"\t\tinvariant.Sometimes(len(v) == 1, \"one\"),\n" +
		"\t\tinvariant.Sometimes(len(v) == 2, \"two\"),\n" +
		"\t)\n}\n")
	if specification_flags(t, files, "must guard both ends") {
		t.Fatal("a complete generic-container length bundle must not be flagged for bounds")
	}
	if specification_flags(t, files, "must claim") {
		t.Fatal("a complete generic count bundle must not be flagged for coverage")
	}
}

// Test_Type_Invariant_Struct_Composition_Passes verifies a struct that composes
// every coverable field's invariant is not flagged.
func Test_Type_Invariant_Struct_Composition_Passes(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"import \"fixture/shared/invariant\"\n\n" +
		"// Token is a fixture.\ntype Token string\n\n" +
		"// Token_Invariants is a fixture.\n" +
		"func Token_Invariants(v Token, namespace invariant.Namespace) {\n" +
		"\tinvariant.Dot_Product(namespace, " +
		"invariant.Sometimes(len(v) == 0, \"x\"))\n}\n\n" +
		"// Pair is a fixture.\ntype Pair struct {\n" +
		"\t// Tok is a fixture.\n\tTok Token\n" +
		"\t// Count is a fixture.\n\tCount int\n}\n\n" +
		"// Pair_Invariants is a fixture.\n" +
		"func Pair_Invariants(v Pair, namespace invariant.Namespace) {\n" +
		"\tToken_Invariants(v.Tok, \"Pair.Tok\")\n" +
		"\tinvariant.Int_Invariants(v.Count, \"Pair.Count\")\n" +
		"\tinvariant.Dot_Product(namespace, invariant.Sometimes(true, \"x\"))\n}\n")
	if specification_flags(t, files, "must call") {
		t.Fatal("a struct composing all field invariants must not be flagged")
	}
}

// Test_Type_Invariant_Struct_Mutex_Skipped verifies a struct with a sync.Mutex
// field is skipped entirely.
func Test_Type_Invariant_Struct_Mutex_Skipped(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"import \"fixture/shared/invariant\"\n\nimport \"sync\"\n\n" +
		"// Guarded is a fixture.\ntype Guarded struct {\n" +
		"\t// Mu is a fixture.\n\tMu sync.Mutex\n\t// N is a fixture.\n\tN int\n}\n\n" +
		"// Guarded_Invariants is a fixture.\n" +
		"func Guarded_Invariants(v Guarded, namespace invariant.Namespace) {\n" +
		"\tinvariant.Dot_Product(namespace, invariant.Sometimes(true, \"x\"))\n}\n")
	if specification_flags(t, files, "must call") {
		t.Fatal("a struct with a sync.Mutex field must be skipped entirely")
	}
}

// Test_Type_Invariant_Struct_Function_Fields_Exempt verifies a func-typed field,
// which has no invariant, is exempt.
func Test_Type_Invariant_Struct_Function_Fields_Exempt(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"import \"fixture/shared/invariant\"\n\n" +
		"// Ops is a fixture.\ntype Ops struct {\n" +
		"\t// Run is a fixture.\n\tRun func()\n}\n\n" +
		"// Ops_Invariants is a fixture.\n" +
		"func Ops_Invariants(v Ops, namespace invariant.Namespace) {\n" +
		"\tinvariant.Dot_Product(namespace, invariant.Sometimes(true, \"x\"))\n}\n")
	if specification_flags(t, files, "must call") {
		t.Fatal("a func-typed field has no invariant and must be exempt")
	}
}

// Test_Type_Invariant_Struct_Boolean_Field_Required verifies a bool field requires
// the Boolean_Invariants preset.
func Test_Type_Invariant_Struct_Boolean_Field_Required(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"import \"fixture/shared/invariant\"\n\n" +
		"// Flag is a fixture.\ntype Flag struct {\n" +
		"\t// On is a fixture.\n\tOn bool\n}\n\n" +
		"// Flag_Invariants is a fixture.\n" +
		"func Flag_Invariants(v Flag, namespace invariant.Namespace) {\n" +
		"\tinvariant.Dot_Product(namespace, invariant.Sometimes(true, \"x\"))\n}\n")
	if !specification_flags(t, files, "must call Boolean_Invariants(v.On") {
		t.Fatal("a bool field must require the Boolean_Invariants preset")
	}
}

// Test_Type_Invariant_Struct_Pointer_Field_Composed verifies a pointer field is
// composed through its pointee: a bundle that omits its pointee invariant is
// flagged, the same as a value field of that type.
func Test_Type_Invariant_Struct_Pointer_Field_Composed(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"import \"fixture/shared/invariant\"\n\n" +
		"// Token is a fixture.\ntype Token string\n\n" +
		"// Token_Invariants is a fixture.\n" +
		"func Token_Invariants(v Token, namespace invariant.Namespace) {\n" +
		"\tinvariant.Dot_Product(namespace, " +
		"invariant.Sometimes(len(v) == 0, \"x\"))\n}\n\n" +
		"// Holder is a fixture.\ntype Holder struct {\n" +
		"\t// Tok is a fixture.\n\tTok *Token\n}\n\n" +
		"// Holder_Invariants is a fixture.\n" +
		"func Holder_Invariants(v Holder, namespace invariant.Namespace) {\n" +
		"\tinvariant.Dot_Product(namespace, invariant.Sometimes(true, \"x\"))\n}\n")
	if !specification_flags(t, files, "must call Token_Invariants") {
		t.Fatal("a pointer field whose pointee invariant is omitted must be flagged")
	}
}

// Test_Function_Assertion_Complete_Passes verifies a function that asserts its
// input after a first-statement output defer is not flagged.
func Test_Function_Assertion_Complete_Passes(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"import \"fixture/shared/invariant\"\n\n" +
		"// Token is a fixture.\ntype Token string\n\n" +
		"// Token_Invariants is a fixture.\n" +
		"func Token_Invariants(v Token, namespace invariant.Namespace) {\n" +
		"\tinvariant.Dot_Product(namespace, " +
		"invariant.Sometimes(len(v) == 0, \"x\"))\n}\n\n" +
		"// Count is a fixture.\ntype Count int\n\n" +
		"// Count_Invariants is a fixture.\n" +
		"func Count_Invariants(v Count, namespace invariant.Namespace) {\n" +
		"\tinvariant.Dot_Product(namespace, invariant.Sometimes(v == 0, \"x\"))\n}\n\n" +
		"// Process does.\nfunc Process(tok Token) (n Count) {\n" +
		"\tdefer func() {\n\t\tCount_Invariants(n, \"Process.n\")\n\t}()\n" +
		"\tToken_Invariants(tok, \"Process.tok\")\n\treturn 0\n}\n")
	if specification_flags(t, files, "must assert") {
		t.Fatal("a fully-asserting function must not be flagged")
	}
}

// Test_Function_Assertion_Exempt_Subjects verifies func/error params and returns,
// which have no invariant, need no assertion.
func Test_Function_Assertion_Exempt_Subjects(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"// Run does.\nfunc Run(action func(), failure error) (result error) {\n" +
		"\tprintln(0)\n\treturn nil\n}\n")
	if specification_flags(t, files, "must assert") {
		t.Fatal("func/error subjects have no invariant and need no assertion")
	}
}

// Test_Function_Assertion_Slice_Loop_Passes verifies a slice parameter asserted by
// an element-wise range loop is not flagged.
func Test_Function_Assertion_Slice_Loop_Passes(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"import \"fixture/shared/invariant\"\n\n" +
		"// Token is a fixture.\ntype Token string\n\n" +
		"// Token_Invariants is a fixture.\n" +
		"func Token_Invariants(v Token, namespace invariant.Namespace) {\n" +
		"\tinvariant.Dot_Product(namespace, " +
		"invariant.Sometimes(len(v) == 0, \"x\"))\n}\n\n" +
		"// Scan does.\nfunc Scan(toks []Token) {\n" +
		"\tfor _, t := range toks {\n\t\tToken_Invariants(t, \"Scan.tok\")\n\t}\n" +
		"\tprintln(0)\n}\n")
	if specification_flags(t, files, "must assert") {
		t.Fatal("a slice parameter asserted element-wise must not be flagged")
	}
}

// Test_Function_Assertion_Bundle_Exempt verifies a _Invariants bundle is exempt
// from the function-assertion rule (else it would assert its own value).
func Test_Function_Assertion_Bundle_Exempt(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n" +
		"import \"fixture/shared/invariant\"\n\n" +
		"// Token is a fixture.\ntype Token string\n\n" +
		"// Token_Invariants is a fixture.\n" +
		"func Token_Invariants(v Token, namespace invariant.Namespace) {\n" +
		"\tinvariant.Dot_Product(namespace, " +
		"invariant.Sometimes(len(v) == 0, \"x\"))\n}\n")
	if specification_flags(t, files, "must assert") {
		t.Fatal("a _Invariants bundle is exempt from the function-assertion rule")
	}
}

// One non-exempt fixture package: the shared source plus the given test file, so
// the recorder rule has a real package to judge. A single string parameter keeps
// it clear of the input-struct rule.
func recorder_test_files(test string) (files map[string][]byte) {
	return map[string][]byte{
		"pkg/rule.go":      []byte(recorder_fixture_source),
		"pkg/rule_test.go": []byte(test),
	}
}

const recorder_fixture_source = "// Package fixture is a fixture.\npackage fixture\n"

// Lints the recorder fixture with pkg treated as the shared library, since the
// recorder rule now binds shared libraries only — a binary component's packages
// are witnessed through its simulation instead, so pkg must be shared to be judged.
func recorder_self_diagnostics(
	t *testing.T, files map[string][]byte, exempt []string,
) (diags []lint.Diagnostic) {
	t.Helper()
	fsys := fstest.MapFS{}
	for name, content := range files {
		fsys[name] = &fstest.MapFile{Data: content}
	}
	if _, present := fsys["go.mod"]; !present {
		fsys["go.mod"] = &fstest.MapFile{Data: []byte(doctrine_root_go_module)}
	}
	diags, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys: fsys, Scope: "pkg", Shared_Component: "pkg",
		Word_Replacements:         test_word_replacements(),
		Invariant_Exempt_Packages: exempt,
	})
	if err != nil {
		t.Fatalf("Check_File_System: %v", err)
	}
	return diags
}

// Reports whether the recorder fixture produces a diagnostic containing fragment.
func recorder_flags(t *testing.T, files map[string][]byte, fragment string) (found bool) {
	t.Helper()
	return specification_diagnosed(recorder_self_diagnostics(t, files, nil), fragment)
}

// Reports whether some diagnostic carries the given rule name — used where a
// fragment match would be fooled by another rule mentioning the same word.
func specification_named(diags []lint.Diagnostic, name string) (found bool) {
	for _, diagnostic := range diags {
		if diagnostic.Name == name {
			return true
		}
	}
	return false
}

// The binary-component fixture without its simulation package — a thin main and an
// internal entry point — the base each simulation test builds on.
func simulation_component_files() (files map[string][]byte) {
	return map[string][]byte{
		"pkg/main.go": []byte(
			"// Package main is a fixture.\npackage main\n\nfunc main() {}\n"),
		"pkg/internal/entry.go": []byte(
			"// Package internal is a fixture.\npackage internal\n\n" +
				"// Main is the entry point.\nfunc Main() {}\n"),
	}
}

// The base fixture plus the given simulation package source, so the simulation rule
// has a real component whose internal/simulation_test package it can judge.
func simulation_test_files(sim string) (files map[string][]byte) {
	files = simulation_component_files()
	files["pkg/internal/simulation_test/sim_test.go"] = []byte(sim)
	return files
}

// A simulation package source with the given TestMain call and any extra trailing
// declarations, so each simulation test varies only the part it exercises.
func simulation_fixture_source(call string, extra ...string) (source string) {
	tail := ""
	for _, piece := range extra {
		tail += piece
	}
	return "package simulation_test\n\n" +
		"import (\n\t\"testing\"\n\n\tinvariant \"fixture/shared/invariant\"\n)\n\n" +
		"func TestMain(m *testing.M) {\n\t" + call + "\n}\n\n" +
		"func Fuzz_Main(f *testing.F) {\n\t" +
		"f.Fuzz(func(t *testing.T, data []byte) {})\n}\n" +
		tail
}

// A simulation package source that imports pkg/internal and runs the given statement
// in its fuzz body, for exercising the entry restriction to internal.Main.
func simulation_entry_source(body string) (source string) {
	return "package simulation_test\n\nimport (\n\t\"testing\"\n\n" +
		"\t\"github.com/james-orcales/james-orcales/pkg/internal\"\n" +
		"\tinvariant \"fixture/shared/invariant\"\n)\n\n" +
		"func TestMain(m *testing.M) {\n\tinvariant.Run_Test_Main(m, \"../**\")\n}\n\n" +
		"func Fuzz_Main(f *testing.F) {\n\t" + body + "\n}\n"
}

// Test_Simulation_Wired_Passes verifies a simulation package with the canonical
// TestMain and a lone Fuzz function satisfies the rule.
func Test_Simulation_Wired_Passes(t *testing.T) {
	t.Parallel()
	files := simulation_test_files(
		simulation_fixture_source("invariant.Run_Test_Main(m, \"../**\")"))
	if specification_named(specification_self_diagnostics(t, files), "simulation") {
		t.Fatal("a canonical simulation package must not be flagged")
	}
}

// Test_Simulation_Exempt_Passes verifies a binary component whose internal packages
// are all exempt needs no simulation package.
func Test_Simulation_Exempt_Passes(t *testing.T) {
	t.Parallel()
	files := simulation_component_files()
	diags := invariant_exempt_self_diagnostics(t, files, []string{"pkg/internal/**"})
	if specification_named(diags, "simulation") {
		t.Fatal("a wholly exempt internal tree must not require a simulation package")
	}
}

// Test_Simulation_Helpers_Allowed verifies a simulation package may declare helper
// functions and types alongside its fuzz driver and TestMain.
func Test_Simulation_Helpers_Allowed(t *testing.T) {
	t.Parallel()
	files := simulation_test_files(simulation_fixture_source(
		"invariant.Run_Test_Main(m, \"../**\")",
		"\n// Extra is a fixture.\nfunc extra() (n int) { return 0 }\n"))
	if specification_named(specification_self_diagnostics(t, files), "simulation") {
		t.Fatal("a helper declaration must not be flagged")
	}
}

// Test_Simulation_Entry_Main_Allowed verifies a simulation that references only
// internal.Main is not flagged.
func Test_Simulation_Entry_Main_Allowed(t *testing.T) {
	t.Parallel()
	files := simulation_test_files(simulation_entry_source("internal.Main()"))
	if specification_named(specification_self_diagnostics(t, files), "simulation") {
		t.Fatal("referencing only internal.Main must not be flagged")
	}
}

// Test_Simulation_No_Source verifies a source (non-test) file in the simulation
// directory is flagged: the directory holds only the blackbox test package.
func Test_Simulation_No_Source(t *testing.T) {
	t.Parallel()
	files := simulation_test_files(
		simulation_fixture_source("invariant.Run_Test_Main(m, \"../**\")"))
	files["pkg/internal/simulation_test/source.go"] =
		[]byte("// Package simulation is a fixture.\npackage simulation\n")
	if !specification_flags(t, files, "no source file") {
		t.Fatal("a source file in the simulation directory must be flagged")
	}
}

// Test_File_Count_Blackbox_Whitebox_Separate verifies external (foo_test) and
// internal (foo) test files are counted as separate groups, not lumped together.
func Test_File_Count_Blackbox_Whitebox_Separate(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"pkg/fixture.go":    []byte("// Package fixture is a fixture.\npackage fixture\n"),
		"pkg/white_test.go": []byte("package fixture\n"),
		"pkg/black_test.go": []byte("package fixture_test\n"),
	}
	if specification_flags(t, files, "has 2 test files") {
		t.Fatal("blackbox and whitebox test files must count as separate groups")
	}
}

// Test_File_Count_Whitebox verifies whitebox (internal) test files carry their own
// independent count, flagged when they exceed the per-file cap.
func Test_File_Count_Whitebox(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"pkg/fixture.go": []byte("// Package fixture is a fixture.\npackage fixture\n"),
		"pkg/a_test.go":  []byte("package fixture\n"),
		"pkg/b_test.go":  []byte("package fixture\n"),
	}
	if !specification_flags(t, files, "has 2 whitebox test files") {
		t.Fatal("two whitebox test files must be flagged as their own group")
	}
}

// Test_Recorder_Registration_No_Tests verifies a non-exempt package with no test
// file at all is flagged: it can never verify its coverage.
func Test_Recorder_Registration_No_Tests(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{"pkg/rule.go": []byte(recorder_fixture_source)}
	if !recorder_flags(t, files, "must wire invariant.Run_Test_Main") {
		t.Fatal("a package with no test file must be flagged")
	}
}

// Test_Recorder_Registration_Unwired verifies a hand-rolled TestMain that runs
// the suite itself without Run_Test_Main is flagged.
func Test_Recorder_Registration_Unwired(t *testing.T) {
	t.Parallel()
	files := recorder_test_files(
		"package fixture_test\n\nimport (\n\t\"os\"\n\t\"testing\"\n)\n\n" +
			"func TestMain(m *testing.M) {\n\tos.Exit(m.Run())\n}\n")
	if !recorder_flags(t, files, "TestMain must be exactly") {
		t.Fatal("a TestMain that never calls Run_Test_Main must be flagged")
	}
}

// Test_Recorder_Registration_Extra_Statements verifies a TestMain carrying any
// statement beyond the one canonical call is flagged: the body must be exactly it.
func Test_Recorder_Registration_Extra_Statements(t *testing.T) {
	t.Parallel()
	files := recorder_test_files(
		"package fixture_test\n\nimport (\n\t\"testing\"\n\n" +
			"\tinvariant \"fixture/shared/invariant\"\n)\n\n" +
			"func TestMain(m *testing.M) {\n\tinvariant.Run_Test_Main(m)\n" +
			"\tinvariant.Run_Test_Main(m)\n}\n")
	if !recorder_flags(t, files, "TestMain must be exactly") {
		t.Fatal("a TestMain with extra statements must be flagged")
	}
}

// Test_Recorder_Registration_Wired_Passes verifies the canonical TestMain wiring
// satisfies the rule.
func Test_Recorder_Registration_Wired_Passes(t *testing.T) {
	t.Parallel()
	files := recorder_test_files(
		"package fixture_test\n\nimport (\n\t\"testing\"\n\n" +
			"\tinvariant \"fixture/shared/invariant\"\n)\n\n" +
			"func TestMain(m *testing.M) {\n\tinvariant.Run_Test_Main(m)\n}\n")
	if recorder_flags(t, files, "Run_Test_Main") {
		t.Fatal("a TestMain wiring Run_Test_Main must not be flagged")
	}
}

// Test_Recorder_Registration_Main_Exempt verifies a main package, which holds no
// testable invariant logic, is exempt even without tests.
func Test_Recorder_Registration_Main_Exempt(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"pkg/main.go": []byte("// Package main is a fixture.\npackage main\n\n" +
			"func main() {}\n"),
	}
	if specification_flags(t, files, "Run_Test_Main") {
		t.Fatal("a main package must be exempt from the recorder rule")
	}
}

// Test_Recorder_Registration_Exempt_Passes verifies a package listed in
// opt_out_assertion_mandate_packages is skipped even with no TestMain.
func Test_Recorder_Registration_Exempt_Passes(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{"pkg/rule.go": []byte(recorder_fixture_source)}
	diags := recorder_self_diagnostics(t, files, []string{"pkg"})
	if specification_diagnosed(diags, "Run_Test_Main") {
		t.Fatal("an exempt package must not be flagged for missing wiring")
	}
}
