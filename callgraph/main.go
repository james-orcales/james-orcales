// Usage: callgraph [dir] [-filter=substring]   (defaults to current directory).
package main

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"local/james-orcales/callgraph/internal"
)

// Caps the go.mod read; real module files are far smaller.
const MODULE_FILE_BYTES_MAX = 1048576

// Caps a single directory read, bounding memory against a pathological directory.
const DIRECTORY_ENTRIES_MAX = 65536

// Caps how far up the tree the workspace search walks.
const WORKSPACE_SEARCH_DEPTH_MAX = 64

func main() {
	os.Exit(callgraph.Main(&callgraph.Main_Input{
		Arguments:    os.Args,
		Output:       os.Stdout,
		Error_Output: os.Stderr,
		Load:         load,
	}))
}

// Finds the workspace, type-checks every first-party package under it from
// source, and returns them in a deterministic order alongside the module path.
func load() (packages []*callgraph.Package, module string) {
	root, module := find_workspace()
	file_set := token.NewFileSet()
	resolver := importer.ForCompiler(file_set, "gc", export_lookup(export_data(root)))
	discovered := discover_packages(&discover_packages_input{Root: root, Module: module})
	slices.Sort(discovered)
	for _, path := range discovered {
		checked := load_package(&load_package_input{
			Path:     path,
			Root:     root,
			Module:   module,
			File_Set: file_set,
			Importer: resolver,
		})
		if checked == nil {
			continue
		}
		packages = append(packages, checked)
	}
	return packages, module
}

// Finds the workspace root by walking up from the working directory to the
// nearest go.mod, returning its directory and module path.
func find_workspace() (root string, module string) {
	directory, getwd_err := os.Getwd()
	if getwd_err != nil {
		return ".", ""
	}
	for range WORKSPACE_SEARCH_DEPTH_MAX {
		found := module_path(directory)
		if found != "" {
			return directory, found
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return ".", ""
		}
		directory = parent
	}
	return ".", ""
}

// Reads the module path from the workspace go.mod.
func module_path(root string) (path string) {
	content := read_file_bounded(filepath.Join(root, "go.mod"), MODULE_FILE_BYTES_MAX)
	for _, line := range strings.Split(string(content), "\n") {
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

// The input for discover_packages.
type discover_packages_input struct {
	// Root is the workspace root directory to walk.
	Root string
	// Module is the module path prefixing first-party import paths.
	Module string
}

// Walks the root for directories holding first-party Go source, returning their
// import paths; vendored, temporary, and hidden trees are skipped.
func discover_packages(input *discover_packages_input) (paths []string) {
	seen := map[string]bool{}
	walk := func(path string, entry fs.DirEntry, walk_err error) (next error) {
		if walk_err != nil {
			return nil
		}
		if entry.IsDir() {
			if path == input.Root {
				return nil
			}
			if skip_directory(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		import_path, is_source := package_import_path(&package_import_path_input{
			Root: input.Root, Module: input.Module, Path: path,
		})
		if !is_source {
			return nil
		}
		if seen[import_path] {
			return nil
		}
		seen[import_path] = true
		paths = append(paths, import_path)
		return nil
	}
	filepath.WalkDir(input.Root, walk)
	return paths
}

// The input for package_import_path.
type package_import_path_input struct {
	// Root is the workspace root the path is relative to.
	Root string
	// Module is the module path prefixing first-party import paths.
	Module string
	// Path is the source file path being classified.
	Path string
}

// Returns the import path of the package holding a Go source file, or false when
// the file is not first-party source.
func package_import_path(input *package_import_path_input) (import_path string, source bool) {
	name := filepath.Base(input.Path)
	if !strings.HasSuffix(name, ".go") {
		return "", false
	}
	if strings.HasSuffix(name, "_test.go") {
		return "", false
	}
	relative, relative_err := filepath.Rel(input.Root, filepath.Dir(input.Path))
	if relative_err != nil {
		return "", false
	}
	if relative == "." {
		return input.Module, true
	}
	return input.Module + "/" + filepath.ToSlash(relative), true
}

// Maps every package in the workspace's dependency closure to its compiled
// export-data file, asking the go toolchain — the only authority on where the
// build cache keeps them. Resolving an import from this file is a single read;
// the alternative, re-type-checking the package from source, is why loading was
// slow. A toolchain failure yields an empty map, leaving imports unresolved
// rather than crashing.
func export_data(root string) (exports map[string]string) {
	exports = map[string]string{}
	// -e keeps a single unbuildable package from failing the whole list, so its
	// broken siblings still yield export data instead of the map coming back empty.
	// The tab-separated template emits one "import-path<TAB>export-file" line per
	// package, sidestepping a streaming JSON decoder for a bounded buffer split.
	command := exec.Command(
		"go", "list", "-e", "-deps", "-export",
		"-f", "{{.ImportPath}}\t{{.Export}}", "./...")
	command.Dir = root
	output, list_err := command.Output()
	if list_err != nil {
		return exports
	}
	for _, line := range strings.Split(string(output), "\n") {
		path, export, split := strings.Cut(line, "\t")
		if !split {
			continue
		}
		if export == "" {
			continue
		}
		exports[path] = export
	}
	return exports
}

// Adapts the export-data map into the lookup the gc importer reads through,
// opening a package's export file on demand. An import absent from the map — a
// package with no export data — surfaces as an error the type checker swallows.
func export_lookup(
	exports map[string]string,
) (lookup func(path string) (reader io.ReadCloser, err error)) {
	return func(path string) (reader io.ReadCloser, err error) {
		export, found := exports[path]
		if !found {
			return nil, fmt.Errorf("no export data for %s", path)
		}
		return os.Open(export)
	}
}

// The input for load_package.
type load_package_input struct {
	// Path is the package's import path.
	Path string
	// Root is the workspace root directory.
	Root string
	// Module is the module path prefixing first-party import paths.
	Module string
	// File_Set resolves parsed positions, shared across the whole load.
	File_Set *token.FileSet
	// Importer resolves the package's imports from source.
	Importer types.Importer
}

// Parses and type-checks one first-party package from source, returning its
// extractable form, or nil when the directory holds no parseable source. Type
// errors are swallowed so a package with unresolved imports still yields the
// facts its resolvable code supports.
func load_package(input *load_package_input) (checked *callgraph.Package) {
	relative := strings.TrimPrefix(strings.TrimPrefix(input.Path, input.Module), "/")
	directory := filepath.Join(input.Root, relative)
	files := parse_directory(input.File_Set, directory)
	if len(files) == 0 {
		return nil
	}
	info := new_type_info()
	var swallowed []error
	configuration := &types.Config{Importer: input.Importer}
	configuration.Error = func(type_error error) {
		swallowed = append(swallowed, type_error)
	}
	types_package, _ := configuration.Check(input.Path, input.File_Set, files, info)
	return &callgraph.Package{
		Path:     input.Path,
		Is_Root:  path_is_root(directory, types_package),
		File_Set: input.File_Set,
		Files:    files,
		Info:     info,
		Types:    types_package,
	}
}

// Reports whether a directory name should be skipped while walking the workspace.
func skip_directory(name string) (skip bool) {
	if strings.HasPrefix(name, ".") {
		return true
	}
	switch name {
	case "third_party", "tmp", "vendor", "home", "node_modules":
		return true
	default:
		return false
	}
}

// Parses every non-test Go file in a directory, dropping any that fail to parse.
func parse_directory(file_set *token.FileSet, directory string) (files []*ast.File) {
	for _, entry := range read_directory_bounded(directory, DIRECTORY_ENTRIES_MAX) {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, parse_err := parser.ParseFile(
			file_set, filepath.Join(directory, name), nil, parser.SkipObjectResolution)
		if parse_err != nil {
			continue
		}
		files = append(files, file)
	}
	return files
}

// Reports whether a package is a composition root: a main package or a directory
// named default, where the doctrine allows impure bindings to enter.
func path_is_root(directory string, checked *types.Package) (root bool) {
	if filepath.Base(directory) == "default" {
		return true
	}
	if checked == nil {
		return false
	}
	if checked.Name() == "main" {
		return true
	}
	return false
}

// Returns a package's checked type info, freshly allocated for one Check call.
func new_type_info() (info *types.Info) {
	return &types.Info{
		Defs:       map[*ast.Ident]types.Object{},
		Uses:       map[*ast.Ident]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
		Types:      map[ast.Expr]types.TypeAndValue{},
	}
}

// Reads at most limit bytes of a file, returning what it read; a missing file
// yields nil.
func read_file_bounded(path string, limit_size int64) (content []byte) {
	file, open_err := os.Open(path)
	if open_err != nil {
		return nil
	}
	defer file.Close()
	buffer := make([]byte, limit_size)
	count, _ := io.ReadFull(io.LimitReader(file, limit_size), buffer)
	return buffer[:count]
}

// Reads at most limit entries of a directory, returning what it read; a missing
// directory yields nil.
func read_directory_bounded(directory string, limit int) (entries []fs.DirEntry) {
	handle, open_err := os.Open(directory)
	if open_err != nil {
		return nil
	}
	defer handle.Close()
	result, _ := handle.ReadDir(limit)
	return result
}
