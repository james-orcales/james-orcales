package specification_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"local/james-orcales/lint/internal/specification"
)

// This file dogfoods the doctrine it enforces: each leaf test builds a Package,
// knocks one rule out of true, and asserts Check reports it. The leaf tests are
// the file's leading declarations, in leaf order; the helpers follow them so the
// Order rule (tests at the top) holds for this very file.

// Test_Coverage_Presence verifies an in-scope pure package whose spec is absent
// is flagged.
func Test_Coverage_Presence(t *testing.T) {
	t.Parallel()
	target := baseline(t)
	target.Markdown = nil
	if !flagged(target, "missing SPECIFICATION.md") {
		t.Fatal("an in-scope package missing the spec must be flagged")
	}
}

// Test_Coverage_Exemptions verifies an impure package is never required to carry
// the file.
func Test_Coverage_Exemptions(t *testing.T) {
	t.Parallel()
	target := baseline(t)
	target.Markdown = nil
	target.Impure = true
	if flagged(target, "missing SPECIFICATION.md") {
		t.Fatal("an impure package must be exempt from the mandate")
	}
}

// Test_Format_Preamble verifies content before the first heading is flagged.
func Test_Format_Preamble(t *testing.T) {
	t.Parallel()
	target := baseline(t)
	target.Markdown = append([]byte("Stray prose.\n"), target.Markdown...)
	if !flagged(target, "content precedes the first heading") {
		t.Fatal("preamble content must be flagged")
	}
}

// Test_Format_Leaf verifies a # that gains a ### child becomes a branch whose
// leaf is the ###, requiring Test_<Parent>_<Child>.
func Test_Format_Leaf(t *testing.T) {
	t.Parallel()
	target := baseline(t)
	target.Markdown = append(target.Markdown,
		[]byte("\n# Extra\n\n### Child\n\nA child leaf.\n")...)
	if !flagged(target, "Test_Extra_Child") {
		t.Fatal("a new ### child must require its leaf test")
	}
}

// Test_Format_Heading_Blank_Lines verifies an unfenced heading is flagged.
func Test_Format_Heading_Blank_Lines(t *testing.T) {
	t.Parallel()
	target := baseline(t)
	target.Markdown = append(target.Markdown,
		[]byte("# No Fence\n\nIt lacks a leading blank.\n")...)
	if !flagged(target, "not preceded by a blank line") {
		t.Fatal("an unfenced heading must be flagged")
	}
}

// Test_Format_Heading_Level verifies a heading at a forbidden level (##) is
// flagged; only # and ### are permitted.
func Test_Format_Heading_Level(t *testing.T) {
	t.Parallel()
	target := baseline(t)
	target.Markdown = append(target.Markdown, []byte("\n## Mid Level\n\nLevel two.\n")...)
	if !flagged(target, "not level # or ###") {
		t.Fatal("a level-two heading must be flagged")
	}
}

// Test_Format_Heading_Characters verifies a heading word with a non-letter/digit
// rune is flagged, since it could not form a legal test name.
func Test_Format_Heading_Characters(t *testing.T) {
	t.Parallel()
	target := baseline(t)
	target.Markdown = append(target.Markdown, []byte("\n# Bad-Word\n\nIt has a hyphen.\n")...)
	if !flagged(target, "must use only letters and digits") {
		t.Fatal("a punctuated heading must be flagged")
	}
}

// Test_Format_Heading_Uniqueness verifies two headings sharing a name are
// flagged.
func Test_Format_Heading_Uniqueness(t *testing.T) {
	t.Parallel()
	target := baseline(t)
	target.Markdown = append(target.Markdown,
		[]byte("\n# Twin\n\nFirst.\n\n# Twin\n\nSecond.\n")...)
	if !flagged(target, "is duplicated") {
		t.Fatal("a duplicate heading must be flagged")
	}
}

// Test_Format_Section_Not_Empty verifies a heading with no body line is flagged.
func Test_Format_Section_Not_Empty(t *testing.T) {
	t.Parallel()
	target := baseline(t)
	target.Markdown = append(target.Markdown, []byte("\n# Empty\n")...)
	if !flagged(target, "has no body line") {
		t.Fatal("a bodyless section must be flagged")
	}
}

// Test_Format_Section_Size verifies a section over three lines is flagged.
func Test_Format_Section_Size(t *testing.T) {
	t.Parallel()
	target := baseline(t)
	target.Markdown = append(target.Markdown, []byte("\n# Long\n\none\ntwo\nthree\nfour\n")...)
	if !flagged(target, "exceeds three lines") {
		t.Fatal("an oversized section must be flagged")
	}
}

// Test_Format_Section_Contiguity verifies a blank line between body lines is
// flagged.
func Test_Format_Section_Contiguity(t *testing.T) {
	t.Parallel()
	target := baseline(t)
	target.Markdown = append(target.Markdown, []byte("\n# Gapped\n\none\n\ntwo\n")...)
	if !flagged(target, "blank line between body lines") {
		t.Fatal("a gapped section body must be flagged")
	}
}

// Test_Tests_Presence verifies a package whose test file is absent is flagged.
func Test_Tests_Presence(t *testing.T) {
	t.Parallel()
	target := baseline(t)
	target.Test = nil
	if !flagged(target, "missing specification_test.go") {
		t.Fatal("a package missing the test file must be flagged")
	}
}

// Test_Tests_Per_Heading verifies a heading with no matching test is flagged.
func Test_Tests_Per_Heading(t *testing.T) {
	t.Parallel()
	target := baseline(t)
	target.Markdown = append(target.Markdown, []byte("\n# Phantom\n\nIt has no test.\n")...)
	if !flagged(target, "Test_Phantom") {
		t.Fatal("a heading without a test must be flagged")
	}
}

// Test_Tests_Name_Normalization verifies a test not named for its heading's
// Ada_Case form is flagged.
func Test_Tests_Name_Normalization(t *testing.T) {
	t.Parallel()
	target := baseline(t)
	target.Test = parse_test(t, strings.Replace(BASELINE_MARKDOWN_TEST,
		"func Test_Sole_Rule(", "func Test_Misnamed(", 1))
	if !flagged(target, "needs Test_Sole_Rule") {
		t.Fatal("a misnamed heading test must be flagged")
	}
}

// Test_Tests_Order verifies a leaf whose test is not in order is flagged.
func Test_Tests_Order(t *testing.T) {
	t.Parallel()
	target := baseline(t)
	target.Markdown = append([]byte("\n# Alpha\n\nIt jumps the order.\n"), target.Markdown...)
	if !flagged(target, "in order") {
		t.Fatal("an out-of-order leaf test must be flagged")
	}
}

// The clean spec markdown every Format and Tests test mutates: one # leaf and
// its single matching test, clean under every rule. The leading blank satisfies
// the blank-before-heading rule for the first heading, and a two-word leaf keeps
// the name-normalization test honest (Sole Rule -> Sole_Rule -> Test_Sole_Rule).
const BASELINE_MARKDOWN = `
# Sole Rule

The sole rule.
`

const BASELINE_MARKDOWN_TEST = `package fixture_test

import "testing"

// Test_Sole_Rule checks the sole rule.
func Test_Sole_Rule(t *testing.T) {
	t.Parallel()
}
`

// Parses a specification_test.go source into the AST Check consumes, mirroring
// the caller's single parse of the real file.
func parse_test(t *testing.T, source string) (file *ast.File) {
	t.Helper()
	file_set := token.NewFileSet()
	parsed, err := parser.ParseFile(
		file_set, "pkg/specification_test.go", source, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return parsed
}

// Builds the clean Package: a pure in-scope module package carrying the baseline
// spec and its matching test.
func baseline(t *testing.T) (target specification.Package) {
	t.Helper()
	return specification.Package{
		Path: "pkg", Has_Module: true, Impure: false,
		Markdown: []byte(BASELINE_MARKDOWN), Test: parse_test(t, BASELINE_MARKDOWN_TEST),
	}
}

// Reports whether Check emits a diagnostic whose message contains fragment for
// the single package target, run scoped to it.
func flagged(target specification.Package, fragment string) (found bool) {
	diags := specification.Check(&specification.Check_Input{
		Packages: []specification.Package{target}, Scope: "pkg"})
	for _, d := range diags {
		if strings.Contains(d.Message, fragment) {
			return true
		}
	}
	return false
}
