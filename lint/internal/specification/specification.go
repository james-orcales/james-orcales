// Package specification enforces the doctrine governing SPECIFICATION.md files:
// the markdown structure every pure package's spec must obey and the
// specification_test.go that must mirror its leaf headings. It reads and parses
// nothing — the caller pre-computes each Package from its single parse — so it
// is a pure, deterministic function of its input.
package specification

import (
	"fmt"
	"go/ast"
	"go/token"
	"local/james-orcales/lint/internal/strings"
	"path"

	"local/james-orcales/lint/internal/diagnostic"
	"local/james-orcales/shared/unicode/ucd"
)

// Package is one package directory under spec-check, with everything Check needs
// already gathered by the caller so this package performs no I/O of its own.
type Package struct {
	// Path is the package directory the diagnostics attach under.
	Path string
	// Has_Module is true when the directory belongs to a discovered module; the
	// coverage mandate no-ops otherwise, matching every other doctrine check.
	Has_Module bool
	// Impure is true for a package main or default tier, exempt from the mandate.
	Impure bool
	// Markdown is the exact-cased SPECIFICATION.md content, or nil when absent.
	Markdown []byte
	// Test is the specification_test.go parsed by the caller's single parse, or
	// nil when the file is absent.
	Test *ast.File
}

// Check_Input is the whole set of package directories to check plus the scope.
type Check_Input struct {
	// Packages are the directories to check, one entry per directory, in the
	// deterministic order the caller established.
	Packages []Package
	// Scope narrows the coverage mandate to a package argument; empty is a
	// whole-workspace run that demands the file of every pure package.
	Scope string
}

// Check enforces the SPECIFICATION.md doctrine over every package in the input.
func Check(input *Check_Input) (diags []diagnostic.Diagnostic) {
	for _, target := range input.Packages {
		diags = append(diags, check_package(target, input.Scope)...)
	}
	return diags
}

func check_package(target Package, scope string) (diags []diagnostic.Diagnostic) {
	if target.Markdown == nil {
		return check_coverage(target, scope)
	}
	markdown_path := path.Join(target.Path, "SPECIFICATION.md")
	lines := strings.Split(string(target.Markdown), "\n")
	leaves, format_diags := check_format(markdown_path, lines)
	diags = append(diags, format_diags...)
	return append(diags, check_tests(target.Path, target.Test, leaves)...)
}

// Applies the mandate to a package whose SPECIFICATION.md is absent. Coverage
// follows the package argument: an explicit scope demands the file within that
// subtree, an empty scope demands it everywhere. Module-less, vendored/example,
// and impure (package main or default) trees are never required to carry one.
func check_coverage(target Package, scope string) (diags []diagnostic.Diagnostic) {
	if !target.Has_Module {
		return nil
	}
	if directory_exempt(target.Path) {
		return nil
	}
	if target.Impure {
		return nil
	}
	covered := scope == ""
	if !covered {
		covered = target.Path == scope
	}
	if !covered {
		covered = strings.Has_Prefix(target.Path, scope+"/")
	}
	if !covered {
		return nil
	}
	return []diagnostic.Diagnostic{coverage_diag(target.Path)}
}

// Validates the structural rules a SPECIFICATION.md must obey — no preamble
// before the first heading, a single # / ### heading model, unique headings of
// letters-and-digits words, a blank line either side of every heading, a
// contiguous body of one to three lines per section. Line width is not checked
// here: the markdown line-max check enforces it for every .md file. Returns the
// leaf test-name bases in source order so the correspondence rules can use them.
func check_format(
	markdown_path string, lines []string,
) (leaves []string, diags []diagnostic.Diagnostic) {
	headings, scan_diags := scan_headings(markdown_path, lines)
	diags = append(diags, scan_diags...)
	leaf_lines, names, leaf_diags := scan_leaves(markdown_path, headings)
	diags = append(diags, leaf_diags...)
	body_diags := scan_bodies(markdown_path, lines, headings, leaf_lines)
	return names, append(diags, body_diags...)
}

// One heading found in a SPECIFICATION.md: its level (1 or 3), 1-based source
// line, the raw text after the marker, and its Ada_Case form.
type Heading struct {
	// Level is the heading level: 1 or 3.
	Level int
	// Line is the 1-based source line the heading sits on.
	Line int
	// Raw is the heading text following the marker.
	Raw string
	// Ada is the heading's Ada_Case form.
	Ada string
}

func position_at(path string, line int) (position token.Position) {
	return token.Position{Filename: path, Line: line}
}

// Reports a line's heading level: 3 for "### ", 1 for "# ", 0 otherwise.
func heading_parse(line string) (level int, raw string) {
	if strings.Has_Prefix(line, "### ") {
		return 3, strings.Trim_Prefix(line, "### ")
	}
	if strings.Has_Prefix(line, "# ") {
		return 1, strings.Trim_Prefix(line, "# ")
	}
	return 0, ""
}

// Pass one: collect every # / ### heading and emit the diagnostics that need
// only line context — bad heading levels, content before the first heading,
// blank-line fencing, and non-letter/digit heading words.
func scan_headings(
	markdown_path string, lines []string,
) (headings []Heading, diags []diagnostic.Diagnostic) {
	seen_heading := false
	for i, line := range lines {
		position := position_at(markdown_path, i+1)
		level, raw := heading_parse(line)
		if level == 0 {
			if strings.Has_Prefix(line, "#") {
				diags = append(diags, heading_level_diag(position))
				continue
			}
			if strings.Trim_Space(line) == "" {
				continue
			}
			if !seen_heading {
				diags = append(diags, preamble_diag(position))
			}
			continue
		}
		headings = append(headings, Heading{
			Level: level, Line: i + 1, Raw: raw, Ada: ada_case(raw)})
		diags = append(diags, heading_line_diags(position, lines, i, raw)...)
		seen_heading = true
	}
	return headings, diags
}

// The per-heading diagnostics for one heading line: non-letter/digit words and
// blank-line fencing.
func heading_line_diags(
	position token.Position, lines []string, i int, raw string,
) (diags []diagnostic.Diagnostic) {
	if heading_words_invalid(raw) {
		diags = append(diags, heading_words_diag(position, raw))
	}
	return append(diags, blank_lines(position, lines, i, raw)...)
}

// State for the walk that determines leaves: a # with no ### child is a leaf
// named Ada(#); each ### is a leaf named Ada(#)_Ada(###). # names are unique
// file-wide; ### names are unique within their parent #.
type Tree struct {
	// Path is the SPECIFICATION.md path being walked.
	Path string
	// Seen_H2 records the # names seen file-wide, for uniqueness.
	Seen_H2 map[string]bool
	// Seen_H3 records the ### names seen within the current #.
	Seen_H3 map[string]bool
	// Parent is the currently open # heading.
	Parent Heading
	// Has_Child records whether the open # has a ### child.
	Has_Child bool
	// Leaf_Lines maps each leaf heading's line to true.
	Leaf_Lines map[int]bool
	// Names is the ordered list of leaf test-name bases.
	Names []string
}

// Pass two: walk the headings into the tree, returning the lines that open a
// leaf section, the ordered leaf test-name bases, and the uniqueness diagnostics.
func scan_leaves(
	markdown_path string, headings []Heading,
) (leaf_lines map[int]bool, names []string, diags []diagnostic.Diagnostic) {
	state := &Tree{
		Path: markdown_path, Seen_H2: map[string]bool{},
		Seen_H3: map[string]bool{}, Leaf_Lines: map[int]bool{},
	}
	for _, entry := range headings {
		diags = append(diags, tree_add(state, entry)...)
	}
	tree_close(state)
	return state.Leaf_Lines, state.Names, diags
}

func tree_add(state *Tree, entry Heading) (diags []diagnostic.Diagnostic) {
	if entry.Level == 3 {
		return tree_child(state, entry)
	}
	tree_close(state)
	if state.Seen_H2[entry.Raw] {
		diags = append(diags, tree_duplicate(state, entry))
	}
	state.Seen_H2[entry.Raw] = true
	state.Parent = entry
	state.Has_Child = false
	state.Seen_H3 = map[string]bool{}
	return diags
}

func tree_child(state *Tree, entry Heading) (diags []diagnostic.Diagnostic) {
	position := position_at(state.Path, entry.Line)
	if state.Parent.Line == 0 {
		return []diagnostic.Diagnostic{orphan_diag(position, entry.Raw)}
	}
	state.Has_Child = true
	if state.Seen_H3[entry.Raw] {
		diags = append(diags, tree_duplicate(state, entry))
	}
	state.Seen_H3[entry.Raw] = true
	state.Leaf_Lines[entry.Line] = true
	state.Names = append(state.Names, state.Parent.Ada+"_"+entry.Ada)
	return diags
}

// Records the just-finished # as a leaf when it gained no ### child.
func tree_close(state *Tree) {
	if state.Parent.Line == 0 {
		return
	}
	if state.Has_Child {
		return
	}
	state.Leaf_Lines[state.Parent.Line] = true
	state.Names = append(state.Names, state.Parent.Ada)
}

func tree_duplicate(state *Tree, entry Heading) (diag diagnostic.Diagnostic) {
	position := position_at(state.Path, entry.Line)
	return heading_duplicate_diag(position, entry.Raw)
}

// State for the body pass: the currently open section, its accumulated body line
// count, and whether a blank line has already interrupted that body.
type Body struct {
	// Path is the SPECIFICATION.md path being walked.
	Path string
	// Leaf_Lines maps each leaf heading's line to true.
	Leaf_Lines map[int]bool
	// Open is the currently open section's heading.
	Open Heading
	// Body is the accumulated body-line count for the open section.
	Body int
	// Blank records whether a blank line has interrupted the body.
	Blank bool
}

// Pass three: attribute body lines to their opening heading, flagging oversized
// sections, gaps in a section body, and leaf sections with no body. A branch #
// intro is size- and gap-checked but, not being a leaf, may be empty.
func scan_bodies(
	markdown_path string, lines []string, headings []Heading, leaf_lines map[int]bool,
) (diags []diagnostic.Diagnostic) {
	at := map[int]Heading{}
	for _, entry := range headings {
		at[entry.Line] = entry
	}
	state := &Body{Path: markdown_path, Leaf_Lines: leaf_lines}
	for i, line := range lines {
		entry, is_heading := at[i+1]
		if is_heading {
			diags = append(diags, body_close(state)...)
			state.Open = entry
			state.Body = 0
			state.Blank = false
			continue
		}
		if strings.Has_Prefix(line, "#") {
			diags = append(diags, body_close(state)...)
			state.Open = Heading{}
			state.Body = 0
			state.Blank = false
			continue
		}
		if strings.Trim_Space(line) == "" {
			if state.Body > 0 {
				state.Blank = true
			}
			continue
		}
		if state.Open.Line == 0 {
			continue
		}
		diags = append(diags, body_line(state, i+1)...)
	}
	return append(diags, body_close(state)...)
}

func body_line(state *Body, line int) (diags []diagnostic.Diagnostic) {
	position := position_at(state.Path, line)
	raw := state.Open.Raw
	if state.Blank {
		diags = append(diags, section_contiguity_diag(position, raw))
		state.Blank = false
	}
	state.Body++
	if state.Body == 4 {
		diags = append(diags, section_diag(position, raw))
	}
	return diags
}

// Emits the body-required diagnostic when a leaf section closed with no body.
func body_close(state *Body) (diags []diagnostic.Diagnostic) {
	if state.Open.Line == 0 {
		return nil
	}
	if !state.Leaf_Lines[state.Open.Line] {
		return nil
	}
	if state.Body != 0 {
		return nil
	}
	position := position_at(state.Path, state.Open.Line)
	return []diagnostic.Diagnostic{section_body_diag(position, state.Open.Raw)}
}

// True when a heading carries a word with a rune that is neither a letter nor a
// digit. Such a rune survives into the normalized Test_<Heading> name and makes
// it an illegal Go identifier, so the test-correspondence rule could never be
// satisfied for that heading.
func heading_words_invalid(raw string) (invalid bool) {
	for _, word := range strings.Fields(raw) {
		for _, letter := range word {
			if ucd.Is_Letter(ucd.Character(letter)) {
				continue
			}
			if ucd.Is_Digit(ucd.Character(letter)) {
				continue
			}
			return true
		}
	}
	return false
}

func preamble_diag(position token.Position) (diag diagnostic.Diagnostic) {
	return diagnostic.Diagnostic{
		Position: position, Name: "specification",
		Want: "Start the file with a heading.",
		Message: fmt.Sprintf(
			"%s:%d The content is before the first heading. "+
				"Start the file with a heading.",
			position.Filename, position.Line),
	}
}

func section_body_diag(position token.Position, raw string) (diag diagnostic.Diagnostic) {
	return diagnostic.Diagnostic{
		Position: position, Name: "specification",
		Want: "Write a body line in the section.",
		Message: fmt.Sprintf(
			"%s:%d The section %q has no body line. Write a body line.",
			position.Filename, position.Line, raw),
	}
}

func section_contiguity_diag(position token.Position, raw string) (diag diagnostic.Diagnostic) {
	return diagnostic.Diagnostic{
		Position: position, Name: "specification",
		Want: "Remove the blank line from the section body.",
		Message: fmt.Sprintf(
			"%s:%d The section %q has a blank line between two body lines. "+
				"Remove the blank line.",
			position.Filename, position.Line, raw),
	}
}

func heading_duplicate_diag(position token.Position, raw string) (diag diagnostic.Diagnostic) {
	return diagnostic.Diagnostic{
		Position: position, Name: "specification",
		Want: "Write a different heading.",
		Message: fmt.Sprintf(
			"%s:%d The heading %q is not unique. Write a different heading.",
			position.Filename, position.Line, raw),
	}
}

func heading_words_diag(position token.Position, raw string) (diag diagnostic.Diagnostic) {
	return diagnostic.Diagnostic{
		Position: position, Name: "specification",
		Want: "Write only letters and digits in the heading.",
		Message: fmt.Sprintf(
			"%s:%d The heading %q has a character that is not a letter or a "+
				"digit. Write only letters and digits.",
			position.Filename, position.Line, raw),
	}
}

func heading_level_diag(position token.Position) (diag diagnostic.Diagnostic) {
	return diagnostic.Diagnostic{
		Position: position, Name: "specification",
		Want: "Write a \"#\" heading or a \"###\" heading.",
		Message: fmt.Sprintf(
			"%s:%d The heading level is not \"#\" or \"###\". "+
				"Write a \"#\" heading or a \"###\" heading.",
			position.Filename, position.Line),
	}
}

func orphan_diag(position token.Position, raw string) (diag diagnostic.Diagnostic) {
	return diagnostic.Diagnostic{
		Position: position, Name: "specification",
		Want: "Put the subheading below a \"#\" heading.",
		Message: fmt.Sprintf(
			"%s:%d The \"###\" heading %q has no parent \"#\" heading. "+
				"Put the heading below a \"#\" heading.",
			position.Filename, position.Line, raw),
	}
}

func section_diag(position token.Position, raw string) (diag diagnostic.Diagnostic) {
	return diagnostic.Diagnostic{
		Position: position, Name: "specification",
		Want: "Write a maximum of three lines in the section.",
		Message: fmt.Sprintf(
			"%s:%d The section %q has more than three lines. "+
				"Write a maximum of three lines.",
			position.Filename, position.Line, raw),
	}
}

// A directory is exempt from the coverage mandate when any path segment is
// `third_party` (vendored code in a separate module) or `examples`
// (illustrative, not a real package contract). An existing SPECIFICATION.md in
// such a tree is still format-validated; it just is never required to exist.
func directory_exempt(directory string) (exempt bool) {
	for _, segment := range strings.Split(directory, "/") {
		if segment == "third_party" {
			return true
		}
		if segment == "examples" {
			return true
		}
	}
	return false
}

func coverage_diag(directory string) (diag diagnostic.Diagnostic) {
	return diagnostic.Diagnostic{
		Position: token.Position{Filename: path.Join(directory, "SPECIFICATION.md")},
		Name:     "specification",
		Want:     "Add SPECIFICATION.md to the package.",
		Message: fmt.Sprintf(
			"The package %q has no SPECIFICATION.md. Add SPECIFICATION.md.",
			directory),
	}
}

func test_file_diag(directory string) (diag diagnostic.Diagnostic) {
	return diagnostic.Diagnostic{
		Position: token.Position{Filename: path.Join(directory, "specification_test.go")},
		Name:     "specification",
		Want:     "Add specification_test.go to the package.",
		Message: fmt.Sprintf(
			"The package %q has no specification_test.go. "+
				"Add specification_test.go.",
			directory),
	}
}

func blank_lines(
	position token.Position, lines []string, i int, raw string,
) (diags []diagnostic.Diagnostic) {
	preceded := i > 0
	if preceded {
		preceded = lines[i-1] == ""
	}
	if !preceded {
		diags = append(diags, diagnostic.Diagnostic{
			Position: position, Name: "specification",
			Want: "Write a blank line before the heading.",
			Message: fmt.Sprintf(
				"%s:%d There is no blank line before the heading %q. "+
					"Write a blank line.",
				position.Filename, i+1, raw),
		})
	}
	followed := i+1 < len(lines)
	if followed {
		followed = lines[i+1] == ""
	}
	if !followed {
		diags = append(diags, diagnostic.Diagnostic{
			Position: position, Name: "specification",
			Want: "Write a blank line after the heading.",
			Message: fmt.Sprintf(
				"%s:%d There is no blank line after the heading %q. "+
					"Write a blank line.",
				position.Filename, i+1, raw),
		})
	}
	return diags
}

// Verifies the specification_test.go declarations are exactly Test_<Heading> for
// each leaf, in leaf order, as the file's leading functions. Comparing by index
// enforces both the per-heading correspondence and the "tests at the very top,
// in order" rule in one pass: a helper or a misordered test shifts the sequence
// and surfaces as a mismatch at that position. A nil Test means the file is
// absent — the caller found no exact-cased specification_test.go.
func check_tests(
	directory string, test *ast.File, leaves []string,
) (diags []diagnostic.Diagnostic) {
	if test == nil {
		return []diagnostic.Diagnostic{test_file_diag(directory)}
	}
	test_path := path.Join(directory, "specification_test.go")
	functions := test_function_names(test)
	for i, leaf := range leaves {
		want := "Test_" + leaf
		matched := i < len(functions)
		if matched {
			matched = functions[i] == want
		}
		if matched {
			continue
		}
		diags = append(diags, diagnostic.Diagnostic{
			Position: token.Position{Filename: test_path},
			Name:     "specification",
			Want:     "Declare func " + want + ".",
			Message: fmt.Sprintf(
				"%s:%d The file needs %s for the leaf %q. "+
					"Declare the tests in leaf order at the top of the file.",
				test_path, i+1, want, leaf),
		})
	}
	return diags
}

// Lists the leading declaration names: a func by its name, a var/const/type by
// its keyword so any declaration before the leaf tests breaks the ordered match.
// Imports are skipped. The AST is the caller's single parse, so no reparse here.
func test_function_names(file *ast.File) (functions []string) {
	for _, declaration := range file.Decls {
		if generic, is_generic := declaration.(*ast.GenDecl); is_generic {
			if generic.Tok == token.IMPORT {
				continue
			}
			functions = append(functions, generic.Tok.String())
			continue
		}
		function, is_function := declaration.(*ast.FuncDecl)
		if !is_function {
			continue
		}
		functions = append(functions, function.Name.Name)
	}
	return functions
}

// Normalizes a heading to the Ada_Case form used for its test name: each
// space-separated word's first rune is upper-cased and the words are joined
// with underscores ("Test File Name" -> "Test_File_Name").
func ada_case(raw string) (name string) {
	words := strings.Fields(raw)
	for i, word := range words {
		runes := []rune(word)
		runes[0] = rune(ucd.To_Upper(ucd.Character(runes[0])))
		words[i] = string(runes)
	}
	return strings.Join(words, "_")
}
