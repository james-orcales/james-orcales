package source_test

import (
	"go/ast"
	"go/token"
	"testing"

	"github.com/james-orcales/james-orcales/lint/internal/source"
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
