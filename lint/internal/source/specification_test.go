package source_test

import (
	"go/ast"
	"go/token"
	"testing"

	"local/james-orcales/lint/internal/source"
)

// Test_Parsed_File_Path verifies Path carries the repo-relative file path.
func Test_Parsed_File_Path(t *testing.T) {
	t.Parallel()
	f := source.Parsed_File{Path: "lint/lint.go"}
	if f.Path != "lint/lint.go" {
		t.Fatal("Path must carry the file path")
	}
}

// Test_Parsed_File_File_Set verifies File Set carries the position fileset.
func Test_Parsed_File_File_Set(t *testing.T) {
	t.Parallel()
	f := source.Parsed_File{File_Set: token.NewFileSet()}
	if f.File_Set == nil {
		t.Fatal("File_Set must carry the fileset")
	}
}

// Test_Parsed_File_File verifies File carries the parsed syntax tree.
func Test_Parsed_File_File(t *testing.T) {
	t.Parallel()
	f := source.Parsed_File{File: &ast.File{Name: ast.NewIdent("p")}}
	if f.File.Name.Name != "p" {
		t.Fatal("File must carry the syntax tree")
	}
}

// Test_Parsed_File_Source verifies Source carries the raw bytes.
func Test_Parsed_File_Source(t *testing.T) {
	t.Parallel()
	f := source.Parsed_File{Source: []byte("package p\n")}
	if string(f.Source) != "package p\n" {
		t.Fatal("Source must carry the raw bytes")
	}
}

// Test_Component_Root verifies Root carries the top-level directory.
func Test_Component_Root(t *testing.T) {
	t.Parallel()
	c := source.Component{Root: "shared/cli"}
	if c.Root != "shared/cli" {
		t.Fatal("Root must carry the top-level directory")
	}
}

// Test_Component_Import_Path verifies Import Path carries the import path.
func Test_Component_Import_Path(t *testing.T) {
	t.Parallel()
	c := source.Component{Import_Path: "example.com/shared/cli"}
	if c.Import_Path != "example.com/shared/cli" {
		t.Fatal("Import_Path must carry the import path")
	}
}

// Test_Component_Shared_Library verifies Is Shared Library marks the shared tier.
func Test_Component_Shared_Library(t *testing.T) {
	t.Parallel()
	c := source.Component{Is_Shared_Library: true}
	if !c.Is_Shared_Library {
		t.Fatal("Is_Shared_Library must mark a shared component")
	}
}

// Test_Component_Directory_Package verifies Directory Package maps dirs to names.
func Test_Component_Directory_Package(t *testing.T) {
	t.Parallel()
	c := source.Component{Directory_Package: map[string]string{"pkg": "pkg"}}
	if c.Directory_Package["pkg"] != "pkg" {
		t.Fatal("Directory_Package must map a directory to its package")
	}
}

// Test_Component_Index_Components verifies Components holds the component list.
func Test_Component_Index_Components(t *testing.T) {
	t.Parallel()
	index := source.Component_Index{Components: []source.Component{{Root: "lint"}}}
	if index.Components[0].Root != "lint" {
		t.Fatal("Components must hold the component list")
	}
}

// Test_Component_Index_File_To_Component verifies the -1 sentinel marks a file
// belonging to no component.
func Test_Component_Index_File_To_Component(t *testing.T) {
	t.Parallel()
	index := source.Component_Index{File_To_Component: map[string]int{"x.go": -1}}
	if index.File_To_Component["x.go"] != -1 {
		t.Fatal("File_To_Component must carry the -1 no-component sentinel")
	}
}

// Test_Declaration_Index_Declarations verifies Declarations keys a declaration by
// directory, package clause, and name together.
func Test_Declaration_Index_Declarations(t *testing.T) {
	t.Parallel()
	index := declaration_workspace(t)
	key := source.Package_Symbol{
		Directory: "shared/alpha", Package: "alpha", Name: "Widget"}
	if index.Declarations[key].Kind != source.DECLARATION_KIND_TYPE {
		t.Fatal("Declarations must key a type by directory, package, and name")
	}
}

// Test_Declaration_Index_Imports verifies Imports maps a file's qualifier to the
// package it names, and omits an import the workspace does not own.
func Test_Declaration_Index_Imports(t *testing.T) {
	t.Parallel()
	index := declaration_workspace(t)
	key := source.File_Qualifier{Path: "shared/beta/beta.go", Qualifier: "alpha"}
	if index.Imports[key].Directory != "shared/alpha" {
		t.Fatal("Imports must map a qualifier to its package directory")
	}
	stdlib := source.File_Qualifier{Path: "shared/beta/beta.go", Qualifier: "fmt"}
	if _, present := index.Imports[stdlib]; present {
		t.Fatal("Imports must omit a package the workspace does not own")
	}
}

// Test_Declaration_Index_File_Package verifies File Package carries each file's
// own directory and package clause.
func Test_Declaration_Index_File_Package(t *testing.T) {
	t.Parallel()
	index := declaration_workspace(t)
	home := index.File_Package["shared/alpha/alpha_test.go"]
	if home.Package != "alpha_test" {
		t.Fatalf("File_Package must carry the package clause: got %q", home.Package)
	}
	if home.Directory != "shared/alpha" {
		t.Fatalf("File_Package must carry the directory: got %q", home.Directory)
	}
}

// Test_Declaration_Index_Declaration_Kind verifies the three kinds are told
// apart, and that a func, a type, and a const each carry their own node.
func Test_Declaration_Index_Declaration_Kind(t *testing.T) {
	t.Parallel()
	index := declaration_workspace(t)
	kinds := map[string]source.Declaration_Kind{
		"Make":     source.DECLARATION_KIND_FUNCTION,
		"Widget":   source.DECLARATION_KIND_TYPE,
		"SIZE_MAX": source.DECLARATION_KIND_CONSTANT,
	}
	for name, want := range kinds {
		declaration, found := source.Resolve(&source.Resolve_Input{
			Index: index, Path: "shared/alpha/alpha.go", Name: name})
		if !found {
			t.Fatalf("%s must resolve", name)
		}
		if declaration.Kind != want {
			t.Errorf("%s: got kind %d want %d", name, declaration.Kind, want)
		}
	}
}

// Test_Declaration_Index_Ambiguity verifies a name declared twice in one package
// resolves to nothing rather than to one of its declarations.
func Test_Declaration_Index_Ambiguity(t *testing.T) {
	t.Parallel()
	parsed_files := []source.Parsed_File{
		declaration_fixture(t, &declaration_fixture_input{
			Path: "shared/alpha/darwin.go",
			Text: "package alpha\n\nfunc Sample() (value int) { return 1 }\n",
		}),
		declaration_fixture(t, &declaration_fixture_input{
			Path: "shared/alpha/linux.go",
			Text: "package alpha\n\nfunc Sample() (value int) { return 2 }\n",
		}),
	}
	components := source.Build_Component_Index([]source.Component{{
		Root:              "shared",
		Import_Path:       "example.com/shared",
		Directory_Package: map[string]string{},
	}}, parsed_files, "shared")
	index := source.Build_Declaration_Index(parsed_files, components)
	if _, found := source.Resolve(&source.Resolve_Input{
		Index: index, Path: "shared/alpha/darwin.go", Name: "Sample",
	}); found {
		t.Fatal("a name declared in two build-tag variants must not resolve")
	}
}
