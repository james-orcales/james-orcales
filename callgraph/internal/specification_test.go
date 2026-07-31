package callgraph_test

import (
	"io/fs"
	"path/filepath"
	"slices"
	"testing"
	"testing/fstest"

	"local/james-orcales/callgraph/internal"
	"local/james-orcales/shared/snap/default"
)

// Test_Load_Discovers_Source_Through_Injections verifies that Load finds the
// nearest module and type-checks source through only its injected file operations.
func Test_Load_Discovers_Source_Through_Injections(t *testing.T) {
	file_system := fstest.MapFS{
		"go.mod": &fstest.MapFile{Data: []byte("module example.com/workspace\n")},
		"sample/sample.go": &fstest.MapFile{Data: []byte(
			"package sample\n\nfunc Value() int { return 1 }\n")},
	}
	workspace := filepath.Join(string(filepath.Separator), "workspace")
	packages, module := callgraph.Load(&callgraph.Load_Input{
		Working_Directory: filepath.Join(workspace, "sample"),
		Read_File: func(path string, limit_size int64) (content []byte) {
			relative, relative_err := filepath.Rel(workspace, path)
			if relative_err != nil {
				return nil
			}
			content, _ = fs.ReadFile(file_system, filepath.ToSlash(relative))
			return content
		},
		Read_Directory: func(directory string, limit int) (entries []fs.DirEntry) {
			relative, relative_err := filepath.Rel(workspace, directory)
			if relative_err != nil {
				return nil
			}
			entries, _ = fs.ReadDir(file_system, filepath.ToSlash(relative))
			return entries
		},
		Walk_Directory: func(
			root string, walk fs.WalkDirFunc,
		) (err error) {
			return fs.WalkDir(file_system, ".", func(
				path string, entry fs.DirEntry, walk_err error,
			) (next error) {
				absolute := filepath.Join(root, filepath.FromSlash(path))
				return walk(absolute, entry, walk_err)
			})
		},
		Export_Data: func(root string) (exports map[string]string) {
			return map[string]string{}
		},
	})
	if module != "example.com/workspace" {
		t.Fatalf("module = %q, want example.com/workspace", module)
	}
	paths := []string{}
	for _, checked := range packages {
		paths = append(paths, checked.Path)
	}
	if !slices.Equal(paths, []string{"example.com/workspace/sample"}) {
		t.Fatalf("packages = %v, want the sample package", paths)
	}
}

// Test_Chain_Root_To_Leaf verifies Graph_Chain traces an injected field back
// through the functions its value flows across — from the composition root that
// wired it to the leaf that invokes it — using the doctrine's own worked example
// of a clock created in main and threaded into Stamper.Now.
func Test_Chain_Root_To_Leaf(t *testing.T) {
	graph := example_clock_graph()
	chain, found := callgraph.Graph_Chain(graph, "Stamper.Now")
	if !found {
		t.Fatal("Chain not found for Stamper.Now")
	}
	if !snap.Snapshot_Is_Equal(
		snap.Init(`main → Server_New → Handler_New → Stamper.Run`), chain.String()) {
		t.Error("Snapshot mismatch")
	}
}

// Test_Resolve_Wires_Invokes_Into_Calls verifies Graph_Resolve rewrites an
// invoke through a field into a concrete call to the callable that field was
// wired to.
func Test_Resolve_Wires_Invokes_Into_Calls(t *testing.T) {
	graph := &callgraph.Graph{
		Invokes: []callgraph.Invoke{
			{From: "Stamper.Run", Through: "Stamper.Now"},
		},
		Wires: []callgraph.Wire{
			{Root: "main", Field: "Stamper.Now", Callable: "time.Now"},
		},
	}
	callgraph.Graph_Resolve(graph)
	if len(graph.Calls) != 1 {
		t.Fatalf("expected 1 resolved call, got %d", len(graph.Calls))
	}
	want := callgraph.Call{From: "Stamper.Run", To: "time.Now"}
	if graph.Calls[0] != want {
		t.Errorf("resolved call = %+v, want %+v", graph.Calls[0], want)
	}
}

// Test_Broken_Unwired_Invoke verifies Graph_Broken reports a field that an
// invoke dispatches through but no wiring reaches, and leaves a wired field out.
func Test_Broken_Unwired_Invoke(t *testing.T) {
	graph := &callgraph.Graph{
		Invokes: []callgraph.Invoke{
			{From: "Stamper.Run", Through: "Stamper.Now"},
			{From: "Logger.Log", Through: "Logger.Caller"},
		},
		Wires: []callgraph.Wire{
			{Root: "main", Field: "Stamper.Now", Callable: "time.Now"},
		},
	}
	broken := callgraph.Graph_Broken(graph)
	if len(broken) != 1 {
		t.Fatalf("expected 1 broken field, got %v", broken)
	}
	if broken[0] != "Logger.Caller" {
		t.Errorf("broken = %v, want [Logger.Caller]", broken)
	}
}

// Test_Diff_Added_And_Removed_Calls verifies Graph_Diff reports the calls
// present in one graph but not the other, such as a new reach to the network.
func Test_Diff_Added_And_Removed_Calls(t *testing.T) {
	before := &callgraph.Graph{Calls: []callgraph.Call{
		{From: "main", To: "Server_New"},
		{From: "Handler.Serve", To: "os.ReadFile"},
	}}
	after := &callgraph.Graph{Calls: []callgraph.Call{
		{From: "main", To: "Server_New"},
		{From: "Handler.Serve", To: "net.Dial"},
	}}
	added, removed := callgraph.Graph_Diff(
		&callgraph.Graph_Diff_Input{Before: before, After: after})
	if len(added) != 1 {
		t.Fatalf("expected 1 added call, got %v", added)
	}
	if added[0] != (callgraph.Call{From: "Handler.Serve", To: "net.Dial"}) {
		t.Errorf("added = %v", added)
	}
	if len(removed) != 1 {
		t.Fatalf("expected 1 removed call, got %v", removed)
	}
	if removed[0] != (callgraph.Call{From: "Handler.Serve", To: "os.ReadFile"}) {
		t.Errorf("removed = %v", removed)
	}
}

// Builds the README's worked example: a clock created in main and threaded
// through two constructors into Stamper.Now, which Stamper.Run invokes.
func example_clock_graph() (graph *callgraph.Graph) {
	return &callgraph.Graph{
		Functions: []callgraph.Function{
			{Key: "main", Is_Root: true},
			{Key: "Server_New", Is_Root: false},
			{Key: "Handler_New", Is_Root: false},
			{Key: "Stamper.Run", Is_Root: false},
		},
		Fields: []callgraph.Field{
			{Key: "Stamper.Now"},
		},
		Wires: []callgraph.Wire{
			{Root: "main", Field: "Stamper.Now", Callable: "time.Now"},
		},
		Flows: []callgraph.Flow{
			{From: "main", To: "Server_New"},
			{From: "Server_New", To: "Handler_New"},
		},
		Invokes: []callgraph.Invoke{
			{From: "Stamper.Run", Through: "Stamper.Now"},
		},
	}
}
