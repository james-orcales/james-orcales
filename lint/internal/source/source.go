// Package source holds the parsed-input model the linter's rules consume — a
// parsed Go file and the workspace's component graph. It is the input
// counterpart to the diagnostic package (output), letting a pure rule subpackage
// name these types without importing the impure lint core.
package source

import (
	"go/ast"
	"go/token"
	"path"
	"regexp"
	"sort"
	"strings"
)

// Parsed_File is one tracked Go file, parsed once and reused by every AST-tier
// rule so nothing re-reads or re-parses the tree.
type Parsed_File struct {
	// Path is the repo-relative path of the file.
	Path string
	// File_Set is the token.FileSet that resolves the AST's positions to
	// file:line:col.
	File_Set *token.FileSet
	// File is the parsed syntax tree.
	File *ast.File
	// Source is the raw bytes the file was parsed from.
	Source []byte
}

// Component is one component's identity: a top-level directory of Go source in
// the single-module repo. The layout rules need which module owns a file,
// whether that module is the shared library, and which directories hold non-main
// Go packages (the Go-ancestor set the tier-depth rule uses).
type Component struct {
	// Root is the component's top-level directory, e.g. "shared/cli" or "lint".
	Root string
	// Import_Path is the component's full import path.
	Import_Path string
	// Is_Shared_Library marks a shared component; false is a binary component.
	Is_Shared_Library bool
	// Directory_Package maps each directory to the package name declared there.
	Directory_Package map[string]string
}

// Component_Index is the whole workspace's component graph, built once after
// parsing and threaded through the directory-level checks. Components is sorted
// longest-Root first so File_To_Component resolution is a linear scan with a
// longest-prefix-wins guarantee.
type Component_Index struct {
	// Components is every component, sorted longest-Root first.
	Components []Component
	// File_To_Component maps a file path to its component's index in Components,
	// or -1 when the file belongs to no discovered component.
	File_To_Component map[string]int
}

// Classifies, orders, and binds parsed files to the components discovered by
// discover_components. Components is sorted longest-Root first so File_To_Component
// resolution is a linear longest-prefix scan. A scoped run passes only the
// parsed subset; the resulting index still covers every module's Root (for
// import resolution) but its File_To_Component and Directory_Package describe only
// the files actually parsed — which is all the in-scope checks consult.
func Build_Component_Index(
	components []Component, parsed_files []Parsed_File, shared_component string,
) (index *Component_Index) {

	index = &Component_Index{
		Components: components, File_To_Component: make(map[string]int, len(parsed_files))}
	// Classify the shared library by its workspace-root-relative directory (the
	// module Root, e.g. "shared"), matching the slash-relative form used
	// by the rest of lint.json; every other module is a binary. An empty
	// shared_component (e.g. a test that doesn't set one) leaves every module a binary.
	// path.Clean so "./shared/" matches the cleaned module Root; guard the
	// empty case, since path.Clean("") is "." and would wrongly match a root module.
	shared_root := shared_component
	if shared_root != "" {
		shared_root = path.Clean(shared_root)
	}
	for i := range index.Components {
		index.Components[i].Is_Shared_Library = index.Components[i].Root == shared_root
	}
	sort.Slice(index.Components, func(i, j int) (less bool) {
		return len(index.Components[i].Root) > len(index.Components[j].Root)
	})
	for _, pf := range parsed_files {
		index.File_To_Component[pf.Path] =
			component_index_resolve(pf.Path, index.Components)
	}
	// Directory_Package excludes test/main files (component-tier-depth rule).
	for _, pf := range parsed_files {
		if strings.HasSuffix(pf.Path, "_test.go") {
			continue
		}
		if pf.File.Name.Name == "main" {
			continue
		}
		component_index_number := index.File_To_Component[pf.Path]
		if component_index_number < 0 {
			continue
		}
		root := index.Components[component_index_number].Root
		relative := pf.Path
		if root != "." {
			relative = strings.TrimPrefix(pf.Path, root+"/")
		}
		canonical_directory := Canonicalize(path.Dir(relative))
		directory_package := index.Components[component_index_number].Directory_Package
		if _, has := directory_package[canonical_directory]; !has {
			directory_package[canonical_directory] = pf.File.Name.Name
		}
	}
	return index
}

// Strips ^v[0-9]+$ segments from a slash-separated directory path so
// snap/v2/X is treated identically to snap/X. Major-version segments
// are Go module-versioning convention rather than real package tiers,
// and the doctrine's depth rules must see through them.
func Canonicalize(directory string) (canonical string) {

	if directory == "." {
		return "."
	}
	segments := strings.Split(directory, "/")
	filtered := make([]string, 0, len(segments))
	for _, s := range segments {
		if component_index_version_re.MatchString(s) {
			continue
		}
		filtered = append(filtered, s)
	}
	if len(filtered) == 0 {
		return "."
	}
	return strings.Join(filtered, "/")
}

var component_index_version_re = regexp.MustCompile(`^v[0-9]+$`)

func component_index_resolve(file_path string, components []Component) (index int) {

	for i, module := range components {
		if module.Root == "." {
			return i
		}
		if file_path == module.Root {
			return i
		}
		if strings.HasPrefix(file_path, module.Root+"/") {
			return i
		}
	}
	return -1
}

// Returns ancestor directories of `directory` from nearest to module
// root, exclusive of "." itself. invariant.GameLoop annotates the loop
// as intentionally unbounded — path.Dir's fixed point on "." provides
// the real termination.
func check_component_tier_depth_ancestors(directory string) (ancestors []string) {

	current := directory
	for step := 0; ; step++ {
		parent := path.Dir(current)
		if parent == "." {
			break
		}
		if parent == current {
			break
		}
		ancestors = append(ancestors, parent)
		current = parent
	}
	return ancestors
}

// Returns the non-main Go ancestor packages of canonical that count toward
// tier depth. A binary component's top-level internal directory is excluded: all
// its code sits under internal and func Main lives there, so internal is the
// directory the count starts from — the same role a shared module's root
// plays — not a package nested above another. Without the exclusion
// internal/foo/default would count internal as a second ancestor and read as
// nested too deep. Shared components have no internal directory, so the exclusion
// never affects them.
func Library_Ancestors(
	m Component, canonical string,
) (ancestors []string) {
	for _, a := range check_component_tier_depth_ancestors(canonical) {
		if a == "internal" {
			continue
		}
		if _, has := m.Directory_Package[a]; !has {
			continue
		}
		ancestors = append(ancestors, a)
	}
	return ancestors
}

// True iff the file sits exactly one non-main Go ancestor below the
// library tier in its module. Mirrors check_component_tier_depth's
// counting logic but inverts the threshold: tier-depth fires when
// count > 1, the composition-tier exemption fires when count == 1.
func Is_Composition_Tier(pf Parsed_File, components *Component_Index) (yes bool) {
	component_index_number := components.File_To_Component[pf.Path]
	if component_index_number < 0 {
		return false
	}
	m := components.Components[component_index_number]
	relative := pf.Path
	if m.Root != "." {
		relative = strings.TrimPrefix(pf.Path, m.Root+"/")
	}
	canonical := Canonicalize(path.Dir(relative))
	if canonical == "." {
		return false
	}
	return len(Library_Ancestors(m, canonical)) == 1
}

// True iff the file belongs to an impure package: package main, or a `default`
// package or one a Go ancestor below the library tier. Those are the very
// packages a pure package may not depend on, so they are exempt as callers too
// — including their _test.go files, classified by directory since tests carry
// no entry in Directory_Package. A file owned by no module (index -1) is left
// to the downstream no-op convention every other doctrine check follows.
func Is_Impure_Package(pf Parsed_File, components *Component_Index) (yes bool) {

	base := strings.TrimSuffix(pf.File.Name.Name, "_test")
	if base == "main" {
		return true
	}
	component_index_number := components.File_To_Component[pf.Path]
	if component_index_number < 0 {
		return false
	}
	m := components.Components[component_index_number]
	relative := pf.Path
	if m.Root != "." {
		relative = strings.TrimPrefix(pf.Path, m.Root+"/")
	}
	canonical := Canonicalize(path.Dir(relative))
	return directory_is_impure(canonical, m)
}

// True iff the module-relative directory holds an impure package: a `default`
// directory (the naming convention for an impure global binding, see
// check_default_package_name) or a package sitting exactly one non-main Go
// ancestor below the library tier.
func directory_is_impure(canonical string, m Component) (yes bool) {

	if canonical == "." {
		return false
	}
	last := canonical
	slash_offset := strings.LastIndex(canonical, "/")
	if slash_offset >= 0 {
		last = canonical[slash_offset+1:]
	}
	if last == "default" {
		return true
	}
	return len(Library_Ancestors(m, canonical)) == 1
}

// True iff the import path resolves to a first-party package that is itself
// impure (a `default` package, or one a Go ancestor below the library tier). A stdlib
// or third-party path is owned by no module and so is never first-party here.
func Import_Path_Is_Impure(import_path string, components *Component_Index) (yes bool) {

	component_index_number := For_Import_Path(import_path, components)
	if component_index_number < 0 {
		return false
	}
	m := components.Components[component_index_number]
	relative := strings.TrimPrefix(import_path, m.Import_Path)
	relative = strings.TrimPrefix(relative, "/")
	if relative == "" {
		relative = "."
	}
	canonical := Canonicalize(relative)
	return directory_is_impure(canonical, m)
}

// Returns the index of the module whose path is the longest prefix of the
// import path, or -1 for a stdlib/third-party path owned by no module.
func For_Import_Path(import_path string, components *Component_Index) (index int) {

	index = -1
	for i := range components.Components {
		m := components.Components[i]
		if m.Import_Path == "" {
			continue
		}
		under := &import_path_under_component_input{
			Import_Path: import_path, Component_Path: m.Import_Path}
		if !import_path_under_component(under) {
			continue
		}
		if index < 0 {
			index = i
			continue
		}
		if len(m.Import_Path) > len(components.Components[index].Import_Path) {
			index = i
		}
	}
	return index
}

type import_path_under_component_input struct {
	// Import_Path is the candidate package path under test.
	Import_Path string
	// Component_Path is the component's declared import prefix.
	Component_Path string
}

// True iff the import path names the component itself or a package within it.
func import_path_under_component(input *import_path_under_component_input) (yes bool) {

	if input.Import_Path == input.Component_Path {
		return true
	}
	return strings.HasPrefix(input.Import_Path, input.Component_Path+"/")
}

// Returns the workspace-relative directory of the shared module's stdlib-time
// gateway (its time/default), or "" when no module is the shared library.
func Time_Gateway(components *Component_Index) (gateway string) {

	for _, m := range components.Components {
		if !m.Is_Shared_Library {
			continue
		}
		if m.Root == "." {
			return "time/default"
		}
		return m.Root + "/time/default"
	}
	return ""
}

// Returns the shared library component's import path, or "" when no module is the
// shared library.
func Shared_Import(components *Component_Index) (import_path string) {
	for _, m := range components.Components {
		if m.Is_Shared_Library {
			return m.Import_Path
		}
	}
	return ""
}

// Returns the workspace-relative directory of the shared module's raw-IO gateway (its
// io/default), or "" when no module is the shared library.
func IO_Gateway(components *Component_Index) (gateway string) {
	for _, m := range components.Components {
		if !m.Is_Shared_Library {
			continue
		}
		if m.Root == "." {
			return "io/default"
		}
		return m.Root + "/io/default"
	}
	return ""
}

// Method_Satisfies_Stdlib reports whether the method's signature implements a
// standard-library interface (e.g. Read([]byte) (int, error)). The rules that
// exempt such a method — the Methods ban and the Primitive Types rule — share it.
func Method_Satisfies_Stdlib(function_declaration *ast.FuncDecl) (yes bool) {
	if function_declaration.Recv == nil {
		return false
	}
	params := method_field_types(function_declaration.Type.Params)
	results := method_field_types(function_declaration.Type.Results)
	return method_signature_matches(&method_signature{
		Name:    function_declaration.Name.Name,
		Params:  strings.Join(params, ","),
		Results: strings.Join(results, ","),
	})
}

type method_signature struct {
	Name    string
	Params  string
	Results string
}

func method_signature_matches(input *method_signature) (yes bool) {
	switch input.Name {
	case "Error", "String", "GoString":
		return input.Params == "" && input.Results == "string"
	case "Read", "Write":
		return input.Params == "[]byte" && input.Results == "int,error"
	case "Close":
		return input.Params == "" && input.Results == "error"
	case "Seek":
		return input.Params == "int64,int" && input.Results == "int64,error"
	case "WriteTo":
		return input.Params == "io.Writer" && input.Results == "int64,error"
	case "ReadFrom":
		return input.Params == "io.Reader" && input.Results == "int64,error"
	case "Len":
		return input.Params == "" && input.Results == "int"
	case "Less":
		return input.Params == "int,int" && input.Results == "bool"
	case "Swap":
		return input.Params == "int,int" && input.Results == ""
	case "MarshalJSON", "MarshalText", "MarshalBinary":
		return input.Params == "" && input.Results == "[]byte,error"
	case "UnmarshalJSON", "UnmarshalText", "UnmarshalBinary":
		return input.Params == "[]byte" && input.Results == "error"
	case "Format":
		return input.Params == "fmt.State,rune" && input.Results == ""
	case "Set":
		return input.Params == "string" && input.Results == "error"
	case "Scan":
		return input.Params == "any" && input.Results == "error"
	case "Visit":
		return input.Params == "ast.Node" && input.Results == "ast.Visitor"
	case "Open":
		return input.Params == "string" && input.Results == "fs.File,error"
	case "ReadFile":
		return input.Params == "string" && input.Results == "[]byte,error"
	case "ReadDir":
		return input.Params == "string" && input.Results == "[]fs.DirEntry,error"
	case "Stat":
		switch input.Params {
		case "":
			return input.Results == "fs.FileInfo,error"
		case "string":
			return input.Results == "fs.FileInfo,error"
		}
		return false
	case "Name":
		return input.Params == "" && input.Results == "string"
	case "Size":
		return input.Params == "" && input.Results == "int64"
	case "Mode":
		return input.Params == "" && input.Results == "fs.FileMode"
	case "ModTime":
		return input.Params == "" && input.Results == "time.Time"
	case "IsDir":
		return input.Params == "" && input.Results == "bool"
	case "Sys":
		return input.Params == "" && input.Results == "any"
	case "Type":
		return input.Params == "" && input.Results == "fs.FileMode"
	case "Info":
		return input.Params == "" && input.Results == "fs.FileInfo,error"
	}
	return false
}

func method_field_types(fl *ast.FieldList) (output_list []string) {
	if fl == nil {
		return nil
	}
	for _, f := range fl.List {
		rendered := method_render_type(f.Type)
		count := len(f.Names)
		if count == 0 {
			count = 1
		}
		for range count {
			output_list = append(output_list, rendered)
		}
	}
	return output_list
}

func method_render_type(expression ast.Expr) (output_string string) {
	prefix := ""
	for step := 0; ; step++ {
		stripped := false
		switch e := expression.(type) {
		case *ast.StarExpr:
			prefix += "*"
			expression = e.X
			stripped = true
		case *ast.ArrayType:
			if e.Len != nil {
				return "<unknown>"
			}
			prefix += "[]"
			expression = e.Elt
			stripped = true
		case *ast.Ellipsis:
			prefix += "..."
			expression = e.Elt
			stripped = true
		}
		if !stripped {
			break
		}
	}
	switch e := expression.(type) {
	case *ast.Ident:
		return prefix + e.Name
	case *ast.SelectorExpr:
		package_identifier, ok := e.X.(*ast.Ident)
		if !ok {
			return "<unknown>"
		}
		return prefix + package_identifier.Name + "." + e.Sel.Name
	case *ast.InterfaceType:
		if e.Methods == nil {
			return prefix + "any"
		}
		if len(e.Methods.List) == 0 {
			return prefix + "any"
		}
	}
	return "<unknown>"
}

// Import_Local_Name is a named import's local name, or the last path segment of
// an unnamed one.
func Import_Local_Name(implementation *ast.ImportSpec, import_path string) (name string) {
	if implementation.Name != nil {
		return implementation.Name.Name
	}
	slash_offset := strings.LastIndex(import_path, "/")
	return import_path[slash_offset+1:]
}

// Invariant_Name is the _Invariants bundle-function name for a type: Name_Invariants
// when the type is exported, name_invariants when it is unexported.
func Invariant_Name(type_name string) (name string) {
	if ast.IsExported(type_name) {
		return type_name + "_Invariants"
	}
	return type_name + "_invariants"
}

// Path_Matches_Glob reports whether filename matches any of patterns as an
// exact-path glob: "pkg/sub" matches only that path, "pkg/**" its whole subtree,
// and "**" everything. These are the same * / ** globs the deterministic tier's
// exceptions use, so the invariant and recursion exemption lists — both
// user-written lint.json globs — read alike.
func Path_Matches_Glob(filename string, patterns []string) (yes bool) {
	for _, entry := range patterns {
		matched, _ := Glob_Match(&Glob_Match_Input{
			Pattern: Parse_Glob_Pattern(entry).Core, Path: filename})
		if matched {
			return true
		}
	}
	return false
}

// A Glob_Pattern is a lint.json glob entry parsed into the one fact the matcher
// needs: Core, the pattern reduced to a form Glob_Match runs against a full path.
// An unanchored, slash-less entry is rewritten with a leading **/ so it matches at
// any depth.
type Glob_Pattern struct {
	// Core is the pattern reduced to the form Glob_Match runs against a full path.
	Core string
}

// Parse_Glob_Pattern reduces a raw lint.json entry to a Glob_Pattern. A trailing
// slash is stripped; a leading or interior slash anchors the entry to the root; a
// slash-less entry floats, modeled as **/ + entry so one matcher serves both.
// Assumes the entry is non-empty and un-negated.
func Parse_Glob_Pattern(raw string) (parsed Glob_Pattern) {
	trimmed := strings.TrimSuffix(raw, "/")
	had_leading_slash := strings.HasPrefix(trimmed, "/")
	trimmed = strings.TrimPrefix(trimmed, "/")
	anchored := had_leading_slash || strings.Contains(trimmed, "/")
	parsed.Core = trimmed
	if !anchored {
		parsed.Core = "**/" + trimmed
	}
	return parsed
}

// Glob_Match_Input pairs a doublestar pattern with the path tested against it.
type Glob_Match_Input struct {
	// Pattern is the doublestar pattern: ** spans whole segments, * spans one.
	Pattern string
	// Path is the slash-separated path tested against Pattern.
	Path string
}

// Glob_Match reports whether Path matches the doublestar Pattern. A ** segment
// matches zero or more whole path segments; every other segment is matched against
// the corresponding path segment by path.Match, so *, ?, and [...] keep their
// single-segment meaning (none crosses a slash) and a malformed segment surfaces
// as path.Match's ErrBadPattern.
func Glob_Match(input *Glob_Match_Input) (matched bool, err error) {
	return glob_match_segments(&glob_match_segments_input{
		Pattern: strings.Split(input.Pattern, "/"),
		Name:    strings.Split(input.Path, "/"),
	})
}

type glob_match_segments_input struct {
	Pattern []string
	Name    []string
}

// Matches the Pattern segments against the Name segments with a two-pointer scan
// that backtracks across **, the segment-level analogue of wildcard matching. A
// ** is remembered as a resume point and first tried as matching zero segments;
// on a later mismatch the scan returns to it and lets the ** swallow one more
// name segment, which is how a single ** spans an unknown depth. Trailing **s
// match the empty remainder, which is why dir/** also matches dir itself.
func glob_match_segments(input *glob_match_segments_input) (matched bool, err error) {
	pattern := input.Pattern
	name := input.Name
	pattern_index := 0
	name_index := 0
	// The resume index sits just after the most recent **; -1 means no ** is
	// available to backtrack to. star_name_index records how much of name that **
	// has been charged with so far.
	star_pattern_index := -1
	star_name_index := 0
	for name_index < len(name) {
		if pattern_index < len(pattern) {
			if pattern[pattern_index] == "**" {
				star_pattern_index = pattern_index + 1
				star_name_index = name_index
				pattern_index++
				continue
			}
			ok, match_err := path.Match(pattern[pattern_index], name[name_index])
			if match_err != nil {
				return false, match_err
			}
			if ok {
				pattern_index++
				name_index++
				continue
			}
		}
		// No literal segment matched here, so the only way forward is to charge
		// the last ** with one more name segment; absent a **, the match fails.
		if star_pattern_index < 0 {
			return false, nil
		}
		pattern_index = star_pattern_index
		star_name_index++
		name_index = star_name_index
	}
	// Name is exhausted; the match holds only if every leftover pattern segment
	// is a ** standing for the empty remainder.
	for pattern_index < len(pattern) {
		if pattern[pattern_index] != "**" {
			return false, nil
		}
		pattern_index++
	}
	return true, nil
}
