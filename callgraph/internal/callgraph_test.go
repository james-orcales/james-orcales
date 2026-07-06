package callgraph_test

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"slices"
	"testing"

	"local/james-orcales/callgraph/internal"
	invariant "local/james-orcales/shared/invariant/default"
)

// TestMain runs the suite under the invariant framework so Always and Sometimes
// coverage is enforced across every test.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}

// Test_Extract_Invoke_Through_Field verifies Extract mints an invoke fact for a
// call dispatched through a func-typed struct field, such as s.Now().
func Test_Extract_Invoke_Through_Field(t *testing.T) {
	checked := check_package(t, `package sample

type Stamper struct {
	Now func() int64
}

func Stamper_Run(s *Stamper) (moment int64) {
	return s.Now()
}
`)
	graph := callgraph.Extract([]*callgraph.Package{checked})
	want := callgraph.Invoke{From: "sample.Stamper_Run", Through: "sample.Stamper.Now"}
	if !slices.Contains(graph.Invokes, want) {
		t.Errorf("Invokes = %+v, want to contain %+v", graph.Invokes, want)
	}
}

// Test_Extract_Wire_At_Struct_Literal verifies Extract mints a wire fact for a
// struct literal that binds a func-typed field to a named callable.
func Test_Extract_Wire_At_Struct_Literal(t *testing.T) {
	checked := check_package(t, `package sample

type Stamper struct {
	Now func() int64
}

func real_now() (moment int64) {
	return 0
}

func New_Default_Stamper() (stamper *Stamper) {
	return &Stamper{Now: real_now}
}
`)
	graph := callgraph.Extract([]*callgraph.Package{checked})
	want := callgraph.Wire{
		Root:     "sample.New_Default_Stamper",
		Field:    "sample.Stamper.Now",
		Callable: "sample.real_now",
	}
	if !slices.Contains(graph.Wires, want) {
		t.Errorf("Wires = %+v, want to contain %+v", graph.Wires, want)
	}
}

// Test_Extract_Flow_Through_Argument verifies Extract mints a flow edge when a
// function hands a func-typed value on to another as a call argument.
func Test_Extract_Flow_Through_Argument(t *testing.T) {
	checked := check_package(t, `package sample

func consume(now func() int64) (moment int64) {
	return now()
}

func thread(now func() int64) (moment int64) {
	return consume(now)
}
`)
	graph := callgraph.Extract([]*callgraph.Package{checked})
	want := callgraph.Flow{From: "sample.thread", To: "sample.consume"}
	if !slices.Contains(graph.Flows, want) {
		t.Errorf("Flows = %+v, want to contain %+v", graph.Flows, want)
	}
}

// Type-checks src as a single-file package and returns it as a callgraph
// Package ready for Extract; the package clause supplies the import path.
func check_package(t *testing.T, source string) (checked *callgraph.Package) {
	t.Helper()
	file_set := token.NewFileSet()
	file, parse_err := parser.ParseFile(
		file_set, "sample.go", source, parser.SkipObjectResolution)
	if parse_err != nil {
		t.Fatalf("parse: %v", parse_err)
	}
	path := file.Name.Name
	info := &types.Info{
		Defs:       map[*ast.Ident]types.Object{},
		Uses:       map[*ast.Ident]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
		Types:      map[ast.Expr]types.TypeAndValue{},
	}
	configuration := &types.Config{Importer: importer.Default()}
	checked_types, check_err := configuration.Check(path, file_set, []*ast.File{file}, info)
	if check_err != nil {
		t.Fatalf("check: %v", check_err)
	}
	return &callgraph.Package{
		Path:     path,
		File_Set: file_set,
		Files:    []*ast.File{file},
		Info:     info,
		Types:    checked_types,
	}
}
