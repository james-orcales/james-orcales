package resolution_test

import (
	"go/parser"
	"go/token"
	"testing"

	"local/james-orcales/lint/internal/diagnostic"
	"local/james-orcales/lint/internal/resolution"
	"local/james-orcales/lint/internal/source"
	"local/james-orcales/lint/internal/strings"
)

// Test_Type_Resolution keeps plain syntax from hiding a deleted bare type.
func Test_Type_Resolution(t *testing.T) {
	t.Parallel()
	field := check(t, "package fixture\n\ntype Output struct {\n\tValue Builder_Handle\n}\n")
	if !diagnosed(field, "Builder_Handle does not resolve to type") {
		t.Fatal("unresolved field type must be flagged")
	}
	parameter := check(t, "package fixture\n\nfunc Use(value Builder_Handle) {}\n")
	if !diagnosed(parameter, "Builder_Handle does not resolve to type") {
		t.Fatal("unresolved parameter type must be flagged")
	}
	declared := check(t, "package fixture\n\ntype Builder_Handle struct{}\n"+
		"\nfunc Use(value Builder_Handle) {}\n")
	if diagnosed(declared, "Builder_Handle does not resolve to type") {
		t.Fatal("package type declaration must resolve")
	}
	qualified := check(t, "package fixture\n\ntype Output struct {\n\tValue remote.Handle\n}\n")
	if len(qualified) != 0 {
		t.Fatal("qualified type outside parsed scope must remain unjudged")
	}
	generic := check(t, "package fixture\n\n"+
		"func Identity[Value any](value Value) Value { return value }\n")
	if len(generic) != 0 {
		t.Fatal("generic ban must own declarations with local type parameters")
	}
}

func check(t *testing.T, text string) (diagnostics []diagnostic.Diagnostic) {
	t.Helper()
	file_set := token.NewFileSet()
	file, err := parser.ParseFile(
		file_set, "fixture/fixture.go", text, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	parsed := []source.Parsed_File{{
		Path: "fixture/fixture.go", File_Set: file_set, File: file, Source: []byte(text),
	}}
	components := source.Build_Component_Index(nil, parsed, "")
	index := source.Build_Declaration_Index(parsed, components)
	return resolution.Make(index)(file_set, file, []byte(text))
}

func diagnosed(diagnostics []diagnostic.Diagnostic, fragment string) (found bool) {
	for _, value := range diagnostics {
		if strings.Contains(value.Message, fragment) {
			return true
		}
	}
	return false
}
