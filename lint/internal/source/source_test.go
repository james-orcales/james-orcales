package source_test

import (
	"go/parser"
	"go/token"
	"testing"

	"local/james-orcales/lint/internal/source"
)

// Declaration_fixture_input pairs a fixture file's path with its source text.
type declaration_fixture_input struct {
	// Path is the repo-relative path the fixture is parsed under.
	Path string
	// Text is the fixture's Go source.
	Text string
}

// Parses one fixture file into the Parsed_File shape the index consumes.
func declaration_fixture(
	t *testing.T, input *declaration_fixture_input,
) (pf source.Parsed_File) {
	t.Helper()
	file_set := token.NewFileSet()
	file, err := parser.ParseFile(
		file_set, input.Path, input.Text,
		parser.SkipObjectResolution|parser.ParseComments)
	if err != nil {
		t.Fatalf("declaration_fixture %s: %v", input.Path, err)
	}
	return source.Parsed_File{
		Path:     input.Path,
		File_Set: file_set,
		File:     file,
		Source:   []byte(input.Text),
	}
}

// Builds a two-package workspace under one shared component, plus an external
// test package beside the first, so a test can resolve names in every direction.
func declaration_workspace(t *testing.T) (index *source.Declaration_Index) {
	t.Helper()
	parsed_files := []source.Parsed_File{
		declaration_fixture(t, &declaration_fixture_input{
			Path: "shared/alpha/alpha.go",
			Text: "package alpha\n\n" +
				"const SIZE_MAX = 4\n\n" +
				"type Widget struct{ X int }\n\n" +
				"func Make() (widget Widget) { return Widget{X: 0} }\n",
		}),
		declaration_fixture(t, &declaration_fixture_input{
			Path: "shared/alpha/alpha_test.go",
			Text: "package alpha_test\n\n" +
				"func helper() (value int) { return 0 }\n",
		}),
		declaration_fixture(t, &declaration_fixture_input{
			Path: "shared/beta/beta.go",
			Text: "package beta\n\n" +
				"import (\n\talpha \"example.com/shared/alpha\"\n" +
				"\tcomposed \"example.com/shared/alpha/default\"\n" +
				"\t\"fmt\"\n)\n\n" +
				"func Use() (widget alpha.Widget) {\n" +
				"\tfmt.Sprint()\n\treturn alpha.Make()\n}\n",
		}),
	}
	components := source.Build_Component_Index([]source.Component{{
		Root:              "shared",
		Import_Path:       "example.com/shared",
		Directory_Package: map[string]string{},
	}}, parsed_files, "shared")
	return source.Build_Declaration_Index(parsed_files, components)
}

// Test_Build_Declaration_Index_Resolves_Same_Package verifies a bare name
// resolves within its own package, and carries the kind of its declaration.
func Test_Build_Declaration_Index_Resolves_Same_Package(t *testing.T) {
	t.Parallel()
	index := declaration_workspace(t)
	function, found := source.Resolve(&source.Resolve_Input{
		Index: index, Path: "shared/alpha/alpha.go", Name: "Make"})
	if !found {
		t.Fatal("a bare name must resolve within its own package")
	}
	if function.Kind != source.DECLARATION_KIND_FUNCTION {
		t.Errorf("Make is a func: got kind %d", function.Kind)
	}
	constant, found := source.Resolve(&source.Resolve_Input{
		Index: index, Path: "shared/alpha/alpha.go", Name: "SIZE_MAX"})
	if !found {
		t.Fatal("a const must resolve")
	}
	if constant.Kind != source.DECLARATION_KIND_CONSTANT {
		t.Errorf("SIZE_MAX is a const: got kind %d", constant.Kind)
	}
	if constant.Value == nil {
		t.Error("a const carries the expression it is declared with")
	}
}

// Test_Build_Declaration_Index_Resolves_Across_Packages verifies a qualified
// name resolves through the referring file's own import list.
func Test_Build_Declaration_Index_Resolves_Across_Packages(t *testing.T) {
	t.Parallel()
	index := declaration_workspace(t)
	widget, found := source.Resolve(&source.Resolve_Input{
		Index: index, Path: "shared/beta/beta.go", Qualifier: "alpha", Name: "Widget"})
	if !found {
		t.Fatal("a qualified name must resolve through the file's imports")
	}
	if widget.Kind != source.DECLARATION_KIND_TYPE {
		t.Errorf("Widget is a type: got kind %d", widget.Kind)
	}
	if widget.Path != "shared/alpha/alpha.go" {
		t.Errorf("the declaring file is alpha.go: got %q", widget.Path)
	}
}

// Test_Build_Declaration_Index_Skips_Stdlib verifies a qualifier naming a
// package outside the workspace resolves to nothing, since no file was parsed
// for it and a miss must never be read as an absent declaration.
func Test_Build_Declaration_Index_Skips_Stdlib(t *testing.T) {
	t.Parallel()
	index := declaration_workspace(t)
	if _, found := source.Resolve(&source.Resolve_Input{
		Index: index, Path: "shared/beta/beta.go", Qualifier: "fmt", Name: "Sprint",
	}); found {
		t.Error("a stdlib qualifier must resolve to nothing")
	}
}

// Test_Build_Declaration_Index_Separates_Test_Package verifies an external test
// package's names stay out of the package it tests, though both sit in one
// directory.
func Test_Build_Declaration_Index_Separates_Test_Package(t *testing.T) {
	t.Parallel()
	index := declaration_workspace(t)
	if _, found := source.Resolve(&source.Resolve_Input{
		Index: index, Path: "shared/alpha/alpha.go", Name: "helper",
	}); found {
		t.Error("package alpha_test must not declare names into package alpha")
	}
	if _, found := source.Resolve(&source.Resolve_Input{
		Index: index, Path: "shared/alpha/alpha_test.go", Name: "helper",
	}); !found {
		t.Error("a name must resolve within its own external test package")
	}
}

// Test_Parse_Glob_Pattern_Negate verifies a leading "!" is stripped into Negate,
// leaving Core identical to the un-negated entry's reduction.
func Test_Parse_Glob_Pattern_Negate(t *testing.T) {
	t.Parallel()
	negated := source.Parse_Glob_Pattern("!pkg/keep")
	if !negated.Negate {
		t.Fatal("a leading ! must set Negate")
	}
	plain := source.Parse_Glob_Pattern("pkg/keep")
	if negated.Core != plain.Core {
		t.Fatalf("Core must ignore the !: got %q want %q", negated.Core, plain.Core)
	}
	if plain.Negate {
		t.Fatal("an entry with no leading ! must not set Negate")
	}
}

// Test_Path_Matches_Glob_Negation_Overrides_Broader_Match verifies a narrower
// negated entry holds one path out of a broader positive entry's release.
func Test_Path_Matches_Glob_Negation_Overrides_Broader_Match(t *testing.T) {
	t.Parallel()
	patterns := []string{"pkg/**", "!pkg/keep"}
	if source.Path_Matches_Glob("pkg/keep", patterns) {
		t.Error("!pkg/keep must veto the broader pkg/** match")
	}
	if !source.Path_Matches_Glob("pkg/other", patterns) {
		t.Error("pkg/** must still match a path the negation does not name")
	}
}

// Test_Path_Matches_Glob_Negation_Wins_Regardless_Of_Order verifies negation
// always wins whether the negated entry is listed before or after the positive
// entry it overrides.
func Test_Path_Matches_Glob_Negation_Wins_Regardless_Of_Order(t *testing.T) {
	t.Parallel()
	negation_first := []string{"!pkg/keep", "pkg/**"}
	negation_last := []string{"pkg/**", "!pkg/keep"}
	if source.Path_Matches_Glob("pkg/keep", negation_first) {
		t.Error("negation listed before the positive entry must still win")
	}
	if source.Path_Matches_Glob("pkg/keep", negation_last) {
		t.Error("negation listed after the positive entry must still win")
	}
}
