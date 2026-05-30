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
// Specification-section tests start from specification_baseline, a minimal but
// self-consistent spec and test pair built inline (no os, no embed: the real
// files are dogfooded separately), and knock one rule out of true.
// Test_Specification_Baseline pins that the unmutated pair is clean.

// Test_Specification_File_Name verifies a spec present only under a different
// case is treated as missing.
func Test_Specification_File_Name(t *testing.T) {
	t.Parallel()
	files := specification_baseline(t)
	files["pkg/specification.md"] = files["pkg/SPECIFICATION.md"]
	delete(files, "pkg/SPECIFICATION.md")
	if !specification_diagnosed(specification_self_diagnostics(t, files),
		"missing SPECIFICATION.md") {
		t.Fatal("a wrong-case spec file must be flagged as missing")
	}
}

// Test_Specification_Coverage verifies an in-scope package without the file is
// flagged.
func Test_Specification_Coverage(t *testing.T) {
	t.Parallel()
	files := specification_baseline(t)
	delete(files, "pkg/SPECIFICATION.md")
	if !specification_diagnosed(specification_self_diagnostics(t, files),
		"missing SPECIFICATION.md") {
		t.Fatal("a package missing the spec must be flagged")
	}
}

// Test_Specification_Preamble verifies content before the first heading is flagged.
func Test_Specification_Preamble(t *testing.T) {
	t.Parallel()
	files := specification_baseline(t)
	specification_markdown_prepend(files, "Stray prose.\n")
	if !specification_diagnosed(specification_self_diagnostics(t, files),
		"content precedes the first heading") {
		t.Fatal("preamble content must be flagged")
	}
}

// Test_Specification_Leaf verifies a # that gains a ### child becomes a branch
// whose leaf is the ###, requiring Test_<Parent>_<Child>.
func Test_Specification_Leaf(t *testing.T) {
	t.Parallel()
	files := specification_baseline(t)
	// A fresh top-level section owns the child, so the expected parent prefix
	// stays Test_Extra_Child no matter how the real sections above are ordered.
	specification_markdown_append(files, "\n# Extra\n\n### Child\n\nA child leaf.\n")
	if !specification_diagnosed(specification_self_diagnostics(t, files),
		"Test_Extra_Child") {
		t.Fatal("a new ### child must require its leaf test")
	}
}

// Test_Specification_Heading_Blank_Lines verifies an unfenced heading is flagged.
func Test_Specification_Heading_Blank_Lines(t *testing.T) {
	t.Parallel()
	files := specification_baseline(t)
	specification_markdown_append(files, "# No Fence\n\nIt lacks a leading blank.\n")
	if !specification_diagnosed(specification_self_diagnostics(t, files),
		"not preceded by a blank line") {
		t.Fatal("an unfenced heading must be flagged")
	}
}

// Test_Specification_Heading_Level verifies a heading at a forbidden level (##)
// is flagged; only # and ### are permitted.
func Test_Specification_Heading_Level(t *testing.T) {
	t.Parallel()
	files := specification_baseline(t)
	specification_markdown_append(files, "\n## Mid Level\n\nLevel two.\n")
	if !specification_diagnosed(specification_self_diagnostics(t, files),
		"not level # or ###") {
		t.Fatal("a level-two heading must be flagged")
	}
}

// Test_Specification_Heading_Characters verifies a heading word with a non-letter/digit rune is
// flagged, since it could not form a legal test name.
func Test_Specification_Heading_Characters(t *testing.T) {
	t.Parallel()
	files := specification_baseline(t)
	specification_markdown_append(files, "\n# Bad-Word\n\nIt has a hyphen.\n")
	if !specification_diagnosed(specification_self_diagnostics(t, files),
		"must use only letters and digits") {
		t.Fatal("a punctuated heading must be flagged")
	}
}

// Test_Specification_Heading_Uniqueness verifies two headings sharing a name are flagged.
func Test_Specification_Heading_Uniqueness(t *testing.T) {
	t.Parallel()
	files := specification_baseline(t)
	specification_markdown_append(files, "\n# Twin\n\nFirst.\n\n# Twin\n\nSecond.\n")
	if !specification_diagnosed(specification_self_diagnostics(t, files),
		"is duplicated") {
		t.Fatal("a duplicate heading must be flagged")
	}
}

// Test_Specification_Section_Not_Empty verifies a heading with no body line is flagged.
func Test_Specification_Section_Not_Empty(t *testing.T) {
	t.Parallel()
	files := specification_baseline(t)
	specification_markdown_append(files, "\n# Empty\n")
	if !specification_diagnosed(specification_self_diagnostics(t, files),
		"has no body line") {
		t.Fatal("a bodyless section must be flagged")
	}
}

// Test_Specification_Section_Size verifies a section over three lines is flagged.
func Test_Specification_Section_Size(t *testing.T) {
	t.Parallel()
	files := specification_baseline(t)
	specification_markdown_append(files, "\n# Long\n\none\ntwo\nthree\nfour\n")
	if !specification_diagnosed(specification_self_diagnostics(t, files),
		"exceeds three lines") {
		t.Fatal("an oversized section must be flagged")
	}
}

// Test_Specification_Section_Contiguity verifies a blank line between body lines is flagged.
func Test_Specification_Section_Contiguity(t *testing.T) {
	t.Parallel()
	files := specification_baseline(t)
	specification_markdown_append(files, "\n# Gapped\n\none\n\ntwo\n")
	if !specification_diagnosed(specification_self_diagnostics(t, files),
		"blank line between body lines") {
		t.Fatal("a gapped section body must be flagged")
	}
}

// Test_Specification_Test_File_Name verifies a test file under a different name is missing.
func Test_Specification_Test_File_Name(t *testing.T) {
	t.Parallel()
	files := specification_baseline(t)
	files["pkg/spec_test.go"] = files["pkg/specification_test.go"]
	delete(files, "pkg/specification_test.go")
	if !specification_diagnosed(specification_self_diagnostics(t, files),
		"missing specification_test.go") {
		t.Fatal("a wrong-name test file must be flagged as missing")
	}
}

// Test_Specification_Test_Per_Heading verifies a heading with no matching test is flagged.
func Test_Specification_Test_Per_Heading(t *testing.T) {
	t.Parallel()
	files := specification_baseline(t)
	specification_markdown_append(files, "\n# Phantom\n\nIt has no test.\n")
	if !specification_diagnosed(specification_self_diagnostics(t, files),
		"Test_Phantom") {
		t.Fatal("a heading without a test must be flagged")
	}
}

// Test_Specification_Test_Name_Normalization verifies a test not named for its heading's
// Ada_Case form is flagged.
func Test_Specification_Test_Name_Normalization(t *testing.T) {
	t.Parallel()
	files := specification_baseline(t)
	renamed := strings.Replace(string(files["pkg/specification_test.go"]),
		"func Test_Sole_Rule(", "func Test_Misnamed(", 1)
	files["pkg/specification_test.go"] = []byte(renamed)
	if !specification_diagnosed(specification_self_diagnostics(t, files),
		"needs Test_Sole_Rule") {
		t.Fatal("a misnamed heading test must be flagged")
	}
}

// Test_Specification_Test_Order verifies a heading whose test is not in order is flagged.
func Test_Specification_Test_Order(t *testing.T) {
	t.Parallel()
	files := specification_baseline(t)
	specification_markdown_prepend(files, "\n# Alpha\n\nIt jumps the order.\n")
	if !specification_diagnosed(specification_self_diagnostics(t, files),
		"in order") {
		t.Fatal("an out-of-order test must be flagged")
	}
}

// Test_Diagnostics_Tier_One verifies a tier-one diagnostic always prints, and
// any tier-one anywhere in scope suppresses every tier-two diagnostic.
func Test_Diagnostics_Tier_One(t *testing.T) {
	t.Parallel()
	var stdout strings.Builder
	lint.Main(&lint.Main_Input{
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
	lint.Main(&lint.Main_Input{
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

// Test_Repository_Symlinks verifies a symlink whose target does not resolve is flagged.
func Test_Repository_Symlinks(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"link": &fstest.MapFile{Mode: fs.ModeSymlink, Data: []byte("missing")},
	}
	diags, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys:           fsys,
		Root_Directory: "/root",
		Readlink: func(name string) (target string, err error) {
			return "missing", nil
		},
		Stat: func(name string) (info fs.FileInfo, err error) {
			return nil, fs.ErrNotExist
		},
	})
	if err != nil {
		t.Fatalf("Check_File_System: %v", err)
	}
	if !specification_diagnosed(diags, "dangling symlink") {
		t.Fatal("a dangling symlink must be flagged")
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
	if !specification_flags(t, files, "third-party github action banned") {
		t.Fatal("a uses: line must be flagged")
	}
}

// Test_Repository_Copyleft_Licenses verifies a GPL license text is flagged.
func Test_Repository_Copyleft_Licenses(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"LICENSE": []byte("GNU GENERAL PUBLIC LICENSE\n\n" +
			"This program is free software.\n"),
	}
	if !specification_flags(t, files, "permissive license") {
		t.Fatal("a copyleft license must be flagged")
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
	if !specification_flags(t, files, "AGENTS.md is missing") {
		t.Fatal("an unpaired CLAUDE.md must be flagged")
	}
}

// Test_Commits_Subject_Size verifies a subject over the character cap is flagged.
func Test_Commits_Subject_Size(t *testing.T) {
	t.Parallel()
	diags := lint.Git_Input_Check(lint.Git_Input{
		Enabled: true,
		Non_Merge_Commits: []lint.Git_Commit{
			{Hash: "abc", Subject: "feat: " + strings.Repeat("x", 200)},
		},
	})
	if !specification_diagnosed(diags, "commit subject is") {
		t.Fatal("an over-long subject must be flagged")
	}
}

// Test_Commits_Conventional_Subjects verifies a non-conventional subject is flagged.
func Test_Commits_Conventional_Subjects(t *testing.T) {
	t.Parallel()
	diags := lint.Git_Input_Check(lint.Git_Input{
		Enabled: true,
		Non_Merge_Commits: []lint.Git_Commit{
			{Hash: "abc", Subject: "did some stuff"},
		},
	})
	if !specification_diagnosed(diags, "non-conventional commit subject") {
		t.Fatal("a non-conventional subject must be flagged")
	}
}

// Test_Commits_Fixup_Commits verifies a fixup! subject is flagged.
func Test_Commits_Fixup_Commits(t *testing.T) {
	t.Parallel()
	diags := lint.Git_Input_Check(lint.Git_Input{
		Enabled: true,
		Non_Merge_Commits: []lint.Git_Commit{
			{Hash: "abc", Subject: "fixup! feat: thing"},
		},
	})
	if !specification_diagnosed(diags, "fixup commit on branch") {
		t.Fatal("a fixup commit must be flagged")
	}
}

// Test_Commits_Merge_Commits verifies a non-subtree merge commit is flagged.
func Test_Commits_Merge_Commits(t *testing.T) {
	t.Parallel()
	diags := lint.Git_Input_Check(lint.Git_Input{
		Enabled: true,
		Merge_Commits: []lint.Git_Commit{
			{Hash: "abc", Subject: "Merge branch 'feature' into main"},
		},
	})
	if !specification_diagnosed(diags, "merge commit on branch") {
		t.Fatal("a merge commit must be flagged")
	}
}

// Test_Module_Layout_Module_Definition verifies a go.mod whose directory is
// absent from go.work is flagged.
func Test_Module_Layout_Module_Definition(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"go.work":           []byte("go 1.25\n\nuse ./registered\n"),
		"registered/go.mod": []byte("module example.com/registered\n\ngo 1.25\n"),
		"orphan/go.mod":     []byte("module example.com/orphan\n\ngo 1.25\n"),
	}
	if !specification_flags(t, files, "not registered in go.work") {
		t.Fatal("a module absent from go.work must be flagged")
	}
}

// Test_Module_Layout_Module_Location verifies a go.mod nested below the repo
// root is flagged.
func Test_Module_Layout_Module_Location(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"nested/inner/go.mod": []byte("module example.com/inner\n\ngo 1.25\n"),
	}
	if !specification_flags(t, files, "located at the repository root") {
		t.Fatal("a module below the repo root must be flagged")
	}
}

// Test_Module_Layout_Shared_Module verifies the shared library's internal/ subtree
// and any package main are flagged: the first hides surface, the second is an entry
// point a fully-importable library has no business owning.
func Test_Module_Layout_Shared_Module(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"go.mod": []byte("module github.com/james-orcales/" +
			"james-orcales/golang_snacks\n\ngo 1.25\n"),
		"internal/x/x.go": []byte("// Package x is a fixture.\npackage x\n"),
	}
	if !specification_flags(t, files, "forbids internal/ directories") {
		t.Fatal("a shared-library internal/ tree must be flagged")
	}
	main_files := map[string][]byte{
		"go.mod": []byte("module github.com/james-orcales/" +
			"james-orcales/golang_snacks\n\ngo 1.25\n"),
		"main.go": []byte("package main\n\nfunc main() {\n\tprintln(0)\n}\n"),
	}
	if !specification_flags(t, main_files, "forbids package main") {
		t.Fatal("a shared-library package main must be flagged")
	}
}

// Test_Module_Layout_Binary_Module verifies a binary module's non-main package
// outside internal/ is flagged.
func Test_Module_Layout_Binary_Module(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"go.mod":             []byte("module example.com/bin\n\ngo 1.25\n"),
		"feature/feature.go": []byte("// Package feature is a fixture.\npackage feature\n"),
	}
	if !specification_flags(t, files, "move feature -> ./internal/feature") {
		t.Fatal("a non-main package outside internal/ must be flagged")
	}
}

// Test_Module_Layout_Main_Package verifies a binary module's main package outside
// the module root — here under a cmd/ directory — is flagged.
func Test_Module_Layout_Main_Package(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"go.mod":          []byte("module example.com/bin\n\ngo 1.25\n"),
		"cmd/app/main.go": []byte("package main\n\nfunc main() {\n\tprintln(0)\n}\n"),
	}
	if !specification_flags(t, files, "main package must sit at the module root") {
		t.Fatal("a main package outside the module root must be flagged")
	}
}

// Test_Module_Layout_Internal_Entry_Point verifies a binary module without a
// func Main in its top-level internal/ package is flagged.
func Test_Module_Layout_Internal_Entry_Point(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"go.mod":  []byte("module example.com/bin\n\ngo 1.25\n"),
		"main.go": []byte("package main\n\nfunc main() {\n\tprintln(0)\n}\n"),
	}
	if !specification_flags(t, files, "declares no func Main in internal/") {
		t.Fatal("a binary module without an internal func Main must be flagged")
	}
}

// Test_Module_Layout_Tier_Depth verifies a package nested beyond one non-main ancestor
// is flagged.
func Test_Module_Layout_Tier_Depth(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"go.mod": []byte("module github.com/james-orcales/" +
			"james-orcales/golang_snacks\n\ngo 1.25\n"),
		"a/a.go":     []byte("// Package a is a fixture.\npackage a\n"),
		"a/b/b.go":   []byte("// Package b is a fixture.\npackage b\n"),
		"a/b/c/c.go": []byte("// Package c is a fixture.\npackage c\n"),
	}
	if !specification_flags(t, files, "exceeds library tier") {
		t.Fatal("an over-nested package must be flagged")
	}
}

// Test_Module_Layout_Impure_Imports verifies a pure package importing a
// denylisted impure stdlib package is flagged.
func Test_Module_Layout_Impure_Imports(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"go.mod": []byte("module github.com/james-orcales/" +
			"james-orcales/golang_snacks\n\ngo 1.25\n"),
		"lib/library.go": []byte("// Package library is a fixture.\npackage library\n\n" +
			"import \"os\"\n\n// F reads.\nfunc F() (s string) {\n" +
			"\treturn os.Getenv(\"X\")\n}\n"),
	}
	if !specification_flags(t, files, "impure stdlib import") {
		t.Fatal("an impure stdlib import must be flagged")
	}
}

// Test_Module_Layout_Impure_Calls verifies a pure package making a denylisted
// impure stdlib call is flagged even when the package may be imported.
func Test_Module_Layout_Impure_Calls(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"go.mod": []byte("module github.com/james-orcales/" +
			"james-orcales/golang_snacks\n\ngo 1.25\n"),
		"lib/library.go": []byte("// Package library is a fixture.\npackage library\n\n" +
			"import \"fmt\"\n\n// F prints.\nfunc F() {\n" +
			"\tfmt.Println(\"x\")\n}\n"),
	}
	if !specification_flags(t, files, "impure stdlib call") {
		t.Fatal("an impure stdlib call must be flagged")
	}
}

// Test_Module_Layout_Transitive_Purity verifies a pure package importing an
// impure `default` package is flagged.
func Test_Module_Layout_Transitive_Purity(t *testing.T) {
	t.Parallel()
	const module = "github.com/james-orcales/james-orcales/golang_snacks"
	files := map[string][]byte{
		"go.mod": []byte("module " + module + "\n\ngo 1.25\n"),
		"lib/library.go": []byte("// Package library is a fixture.\npackage library\n\n" +
			"import \"" + module + "/widget/default\"\n\n" +
			"// F uses the default.\nfunc F() (s string) {\n" +
			"\treturn widget.Name()\n}\n"),
		"widget/default/wire.go": []byte(
			"// Package widget is a fixture.\npackage widget\n\n" +
				"// Name names.\nfunc Name() (s string) {\n\treturn \"x\"\n}\n"),
	}
	if !specification_flags(t, files, "impure dependency") {
		t.Fatal("a pure package importing an impure package must be flagged")
	}
}

// Test_Module_Layout_Transitive_Stdlib verifies the ban reaches a pure package's
// test file: a curated stdlib API that is impure only transitively
// (log.Fatal -> os.Exit) is caught even though importing log and calling
// log.Print stay allowed, and even in _test.go.
func Test_Module_Layout_Transitive_Stdlib(t *testing.T) {
	t.Parallel()
	const module = "github.com/james-orcales/james-orcales/golang_snacks"
	test_files := map[string][]byte{
		"go.mod":         []byte("module " + module + "\n\ngo 1.25\n"),
		"lib/library.go": []byte("// Package library is a fixture.\npackage library\n"),
		"lib/library_test.go": []byte("package library_test\n\nimport \"log\"\n\n" +
			"// Test_F exits.\nfunc Test_F() {\n\tlog.Fatal(\"x\")\n}\n"),
	}
	if !specification_flags(t, test_files, "impure transitive call") {
		t.Fatal("a pure package's test calling a curated stdlib API must be flagged")
	}
}

// Test_Module_Layout_Binary_Purity verifies package main is an impure home: a
// binary's main package using impure stdlib is not flagged.
func Test_Module_Layout_Binary_Purity(t *testing.T) {
	t.Parallel()
	main_impure := map[string][]byte{
		"go.mod": []byte("module example.com/app\n\ngo 1.25\n"),
		"main.go": []byte("package main\n\nimport \"os\"\n\n" +
			"func main() {\n\t_ = os.Getenv(\"X\")\n}\n"),
	}
	if specification_flags(t, main_impure, "impure stdlib") {
		t.Fatal("package main is an impure home; it must not be flagged")
	}
}

// Test_Module_Layout_Library_Purity verifies a shared library's pure package
// using impure stdlib is flagged.
func Test_Module_Layout_Library_Purity(t *testing.T) {
	t.Parallel()
	const shared = "github.com/james-orcales/james-orcales/golang_snacks"
	pure_impure := map[string][]byte{
		"go.mod": []byte("module " + shared + "\n\ngo 1.25\n"),
		"lib/library.go": []byte("// Package library is a fixture.\npackage library\n\n" +
			"import \"os\"\n\n// F reads.\nfunc F() (s string) {\n" +
			"\treturn os.Getenv(\"X\")\n}\n"),
	}
	if !specification_flags(t, pure_impure, "impure stdlib") {
		t.Fatal("a pure library package using impure stdlib must be flagged")
	}
}

// Test_Module_Layout_Default_Package_Impurity verifies a shared library's
// optional `default` package may be impure: it is not flagged.
func Test_Module_Layout_Default_Package_Impurity(t *testing.T) {
	t.Parallel()
	const shared = "github.com/james-orcales/james-orcales/golang_snacks"
	default_impure := map[string][]byte{
		"go.mod":     []byte("module " + shared + "\n\ngo 1.25\n"),
		"foo/foo.go": []byte("// Package foo is a fixture.\npackage foo\n"),
		"foo/default/wire.go": []byte(
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
	if !specification_flags(t, files, "does not satisfy any stdlib interface") {
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
	if !specification_flags(t, files, "is not stdlib") {
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

// Test_Source_And_Test_Bans_Whitebox_Tests verifies a whitebox test package is flagged.
func Test_Source_And_Test_Bans_Whitebox_Tests(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"pkg/thing_test.go": []byte("package fixture\n\nimport \"testing\"\n\n" +
			"// Test_X is a fixture.\nfunc Test_X(t *testing.T) { _ = t }\n"),
	}
	if !specification_flags(t, files, "test file must declare") {
		t.Fatal("a whitebox test package must be flagged")
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

// Test_Source_And_Test_Requirements_Method_Prefix verifies a free function over a local
// type that lacks the type-name prefix is flagged.
func Test_Source_And_Test_Requirements_Method_Prefix(t *testing.T) {
	t.Parallel()
	files := specification_one_file("package fixture\n\n// Widget is a fixture.\n" +
		"type Widget struct{ X int }\n\n// Process does.\n" +
		"func Process(w Widget) (n int) {\n\treturn w.X\n}\n")
	if !specification_flags(t, files, "rename Process -> Widget_<verb>") {
		t.Fatal("a method without a type prefix must be flagged")
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
	diags, err := lint.Check_File_System(&lint.Check_File_System_Input{
		Fsys: fsys, Scope: "pkg"})
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
