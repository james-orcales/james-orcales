// Package source holds the parsed-input model the linter's rules consume — a
// parsed Go file and the workspace's component graph. It is the input
// counterpart to the diagnostic package (output), letting a pure rule subpackage
// name these types without importing the impure lint core.
package source

import (
	"go/ast"
	"go/token"
	"local/james-orcales/lint/internal/strings"
	"path"
	"regexp"
	"sort"
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
		if strings.Has_Suffix(pf.Path, "_test.go") {
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
			relative = strings.Trim_Prefix(pf.Path, root+"/")
		}
		canonical_directory := Canonicalize(path.Dir(relative))
		directory_package := index.Components[component_index_number].Directory_Package
		if _, has := directory_package[canonical_directory]; !has {
			directory_package[canonical_directory] = pf.File.Name.Name
		}
	}
	return index
}

// Declaration_Kind tells a resolved name apart. There is no fourth kind: a
// package-level var is banned outright, so every package-level name a file can
// reach is a func, a type, or a const.
type Declaration_Kind int

// DECLARATION_KIND_FUNCTION marks a func declaration.
const DECLARATION_KIND_FUNCTION Declaration_Kind = 1

// DECLARATION_KIND_TYPE marks a type declaration or alias.
const DECLARATION_KIND_TYPE Declaration_Kind = 2

// DECLARATION_KIND_CONSTANT marks a const declaration.
const DECLARATION_KIND_CONSTANT Declaration_Kind = 3

// Declaration is one package-level name and the node that declares it. Only the
// node matching Kind is set; the rest stay nil.
type Declaration struct {
	// Kind is which of the three package-level declaration forms this is.
	Kind Declaration_Kind
	// Path is the repo-relative path of the file that declares the name.
	Path string
	// Function is the declaration when Kind is DECLARATION_KIND_FUNCTION.
	Function *ast.FuncDecl
	// Type_Specification is the declaration when Kind is DECLARATION_KIND_TYPE.
	Type_Specification *ast.TypeSpec
	// Value is the declared expression when Kind is DECLARATION_KIND_CONSTANT.
	// The iota ban and the grouped-declaration ban together mean a const writes
	// its own value out, so this expression is the whole story.
	Value ast.Expr
	// Ambiguous marks a name declared more than once in its package — the
	// build-tag variants of one package declare the same name per platform.
	// Resolve refuses an ambiguous name rather than pick a variant at random.
	Ambiguous bool
}

// Package_Symbol names one package-level declaration: which directory holds it,
// which package clause declares it, and the name itself. The package clause is
// part of the key because a directory holds two packages whenever an external
// test package sits beside the package it tests, and their names must not mix.
type Package_Symbol struct {
	// Directory is the repo-relative directory holding the declaring file.
	Directory string
	// Package is the package clause of the declaring file.
	Package string
	// Name is the declared identifier.
	Name string
}

// File_Qualifier names one file's local qualifier for an imported package: the
// alias when the import declares one, the imported package's own name when it
// does not.
type File_Qualifier struct {
	// Path is the repo-relative path of the importing file.
	Path string
	// Qualifier is the local name that file calls the imported package by.
	Qualifier string
}

// File_Import names one imported path in one source file.
type File_Import struct {
	// Path is repo-relative path of importing file.
	Path string
	// Import_Path is path written by import specification.
	Import_Path string
}

// Declaration_Index resolves a name to its declaration anywhere in the parsed
// set, with no type checker and no second parse. The doctrine is what makes it
// exact: dot and blank imports are banned, so no name enters a file's scope
// invisibly; aliases are explicit, so one qualifier names one package; package
// vars and func init are banned, so a package-level name is a func, a type, or
// a const; and shadowing is banned, so one name means one thing inside a file.
type Declaration_Index struct {
	// Declarations maps each package-level name to its declaration.
	Declarations map[Package_Symbol]Declaration
	// Imports maps an importing file's local qualifier to the imported package.
	// The Name field of the value is empty: it names the package, not a member.
	Imports map[File_Qualifier]Package_Symbol
	// Import_Paths maps each first-party import path to its declared package.
	Import_Paths map[File_Import]Package_Symbol
	// File_Package maps a file path to its own package clause and directory, so
	// an unqualified reference resolves without re-reading the AST.
	File_Package map[string]Package_Symbol
}

// Build_Declaration_Index indexes every package-level declaration in the parsed
// set and every import each file resolves through. A scoped run passes only the
// parsed subset, so a name outside that subset simply does not resolve — which
// every consumer must read as "unknown", never as "absent".
func Build_Declaration_Index(
	parsed_files []Parsed_File, components *Component_Index,
) (index *Declaration_Index) {

	index = &Declaration_Index{
		Declarations: make(map[Package_Symbol]Declaration, len(parsed_files)),
		Imports:      make(map[File_Qualifier]Package_Symbol, len(parsed_files)),
		Import_Paths: make(map[File_Import]Package_Symbol, len(parsed_files)),
		File_Package: make(map[string]Package_Symbol, len(parsed_files)),
	}
	for _, pf := range parsed_files {
		home := Package_Symbol{
			Directory: path.Dir(pf.Path), Package: pf.File.Name.Name}
		index.File_Package[pf.Path] = home
		declaration_index_file(index, pf, home)
		declaration_index_imports(index, pf, components)
	}
	return index
}

// Records one file's package-level declarations. A name already present in the
// package is marked ambiguous rather than overwritten: the build-tag variants of
// one package each declare the same name, and no consumer may pick between them.
func declaration_index_file(
	index *Declaration_Index, pf Parsed_File, home Package_Symbol,
) {

	record := func(name string, declaration Declaration) {
		if name == "_" {
			return
		}
		key := Package_Symbol{
			Directory: home.Directory, Package: home.Package, Name: name}
		if _, present := index.Declarations[key]; present {
			declaration.Ambiguous = true
		}
		declaration.Path = pf.Path
		index.Declarations[key] = declaration
	}
	for _, declaration := range pf.File.Decls {
		function_declaration, is_function := declaration.(*ast.FuncDecl)
		if is_function {
			// A method belongs to its receiver, not to the package's name
			// space, so it is not a package-level declaration.
			if function_declaration.Recv == nil {
				record(function_declaration.Name.Name, Declaration{
					Kind:     DECLARATION_KIND_FUNCTION,
					Function: function_declaration,
				})
			}
			continue
		}
		generic_declaration, is_generic := declaration.(*ast.GenDecl)
		if !is_generic {
			continue
		}
		declaration_index_generic(generic_declaration, record)
	}
}

// Records the type and const specs of one package-level GenDecl. A var spec is
// skipped: check_no_package_vars bans the form, so indexing it would give a
// consumer a kind the doctrine says cannot exist.
func declaration_index_generic(
	generic_declaration *ast.GenDecl, record func(name string, d Declaration),
) {

	for _, specification := range generic_declaration.Specs {
		type_specification, is_type := specification.(*ast.TypeSpec)
		if is_type {
			record(type_specification.Name.Name, Declaration{
				Kind:               DECLARATION_KIND_TYPE,
				Type_Specification: type_specification,
			})
			continue
		}
		if generic_declaration.Tok != token.CONST {
			continue
		}
		value_specification, is_value := specification.(*ast.ValueSpec)
		if !is_value {
			continue
		}
		for name_index, name := range value_specification.Names {
			value := ast.Expr(nil)
			if name_index < len(value_specification.Values) {
				value = value_specification.Values[name_index]
			}
			record(name.Name, Declaration{
				Kind: DECLARATION_KIND_CONSTANT, Value: value})
		}
	}
}

// Records the package each of one file's qualifiers names. An import the
// workspace does not own — stdlib or third party — is left out, so a lookup
// through it misses and its consumer skips the reference.
func declaration_index_imports(
	index *Declaration_Index, pf Parsed_File, components *Component_Index,
) {

	for _, import_specification := range pf.File.Imports {
		import_path := strings.Trim(import_specification.Path.Value, `"`)
		directory, package_name, resolved :=
			import_path_directory(import_path, components)
		if !resolved {
			continue
		}
		qualifier := Import_Local_Name(import_specification, import_path)
		if import_specification.Name == nil {
			qualifier = package_name
		}
		if qualifier == "_" {
			continue
		}
		if qualifier == "." {
			continue
		}
		imported_package := Package_Symbol{
			Directory: directory, Package: package_name}
		index.Imports[File_Qualifier{
			Path: pf.Path, Qualifier: qualifier,
		}] = imported_package
		index.Import_Paths[File_Import{
			Path: pf.Path, Import_Path: import_path,
		}] = imported_package
	}
}

// Maps a first-party import path to the directory that holds it and the package
// name declared there. The component's Import_Path prefix and Root are what turn
// one into the other; Directory_Package supplies the declared name, which a
// default directory deliberately makes differ from its own last segment.
func import_path_directory(
	import_path string, components *Component_Index,
) (directory string, package_name string, resolved bool) {

	component_index_number := For_Import_Path(import_path, components)
	if component_index_number < 0 {
		return "", "", false
	}
	m := components.Components[component_index_number]
	relative := strings.Trim_Prefix(
		strings.Trim_Prefix(import_path, m.Import_Path), "/")
	directory = relative
	if m.Root != "." {
		directory = m.Root
		if relative != "" {
			directory = m.Root + "/" + relative
		}
	}
	if directory == "" {
		directory = "."
	}
	canonical := Canonicalize(relative)
	if relative == "" {
		canonical = "."
	}
	package_name = m.Directory_Package[canonical]
	if package_name == "" {
		// No parsed file declared the package, so the best available name is
		// path's last segment. Default tier is stronger: repository doctrine
		// requires parent package name even when scoped run did not parse it.
		package_name = import_path[strings.Last_Index(import_path, "/")+1:]
		if package_name == "default" {
			package_name = path.Base(path.Dir(import_path))
		}
	}
	return directory, package_name, true
}

// Package_Names returns every name declared by the package the given file
// belongs to, its own declarations included. An ambiguous name is listed: a name
// declared in two build-tag variants is still in scope, so a caller asking what
// occupies the package's name space must see it even though Resolve refuses to
// pick a declaration for it. A file the index does not hold yields nil.
func Package_Names(index *Declaration_Index, file_path string) (names map[string]bool) {

	home, known := index.File_Package[file_path]
	if !known {
		return nil
	}
	names = make(map[string]bool)
	for key := range index.Declarations {
		if key.Directory != home.Directory {
			continue
		}
		if key.Package != home.Package {
			continue
		}
		names[key.Name] = true
	}
	return names
}

// Resolve_Input carries one name lookup: the index, the file the reference sits
// in, the qualifier it is written with, and the name itself.
type Resolve_Input struct {
	// Index is the workspace's declaration index.
	Index *Declaration_Index
	// Path is the repo-relative path of the file holding the reference.
	Path string
	// Qualifier is the reference's package qualifier, empty when the reference
	// is a bare name in its own package.
	Qualifier string
	// Name is the referenced identifier.
	Name string
}

// Resolve returns the declaration a reference names. found is false when the
// name belongs to a package the run never parsed, when no declaration carries
// it, or when the package declares it more than once — the three cases a
// consumer must treat alike, as "unknown", never as "absent".
func Resolve(input *Resolve_Input) (declaration Declaration, found bool) {

	home, known := input.Index.File_Package[input.Path]
	if !known {
		return Declaration{}, false
	}
	key := Package_Symbol{
		Directory: home.Directory, Package: home.Package, Name: input.Name}
	if input.Qualifier != "" {
		imported, imported_known := input.Index.Imports[File_Qualifier{
			Path: input.Path, Qualifier: input.Qualifier}]
		if !imported_known {
			return Declaration{}, false
		}
		key = Package_Symbol{
			Directory: imported.Directory,
			Package:   imported.Package,
			Name:      input.Name,
		}
	}
	declaration, found = input.Index.Declarations[key]
	if !found {
		return Declaration{}, false
	}
	if declaration.Ambiguous {
		return Declaration{}, false
	}
	return declaration, true
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
		if strings.Has_Prefix(file_path, module.Root+"/") {
			return i
		}
	}
	return -1
}

// Returns ancestor directories of `directory` from nearest to module
// root, exclusive of "." itself. aver.GameLoop annotates the loop
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
	// A binary has one impure home, package main, so nothing under it reaches the
	// composition tier. Without this gate a binary's internal/foo/bar would be
	// released from the purity bans by depth alone.
	if !m.Is_Shared_Library {
		return false
	}
	relative := pf.Path
	if m.Root != "." {
		relative = strings.Trim_Prefix(pf.Path, m.Root+"/")
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

	base := strings.Trim_Suffix(pf.File.Name.Name, "_test")
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
		relative = strings.Trim_Prefix(pf.Path, m.Root+"/")
	}
	canonical := Canonicalize(path.Dir(relative))
	return directory_is_impure(canonical, m)
}

// True iff the module-relative directory holds an impure package: a `default`
// directory (the naming convention for an impure global binding, see
// check_default_package_name) or a package sitting exactly one non-main Go
// ancestor below the library tier.
func directory_is_impure(canonical string, m Component) (yes bool) {

	// The impure tier is the shared library's alone. A binary keeps its impurity
	// in package main — which Is_Impure_Package answers before reaching here — so
	// no directory under a binary earns the release.
	if !m.Is_Shared_Library {
		return false
	}
	if canonical == "." {
		return false
	}
	last := canonical
	slash_offset := strings.Last_Index(canonical, "/")
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
	relative := strings.Trim_Prefix(import_path, m.Import_Path)
	relative = strings.Trim_Prefix(relative, "/")
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
		under := &Import_Path_Under_Component_Input{
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

// Import_Path_Under_Component_Input pairs a candidate import path with a
// component's declared import prefix, for the containment test.
type Import_Path_Under_Component_Input struct {
	// Import_Path is the candidate package path under test.
	Import_Path string
	// Component_Path is the component's declared import prefix.
	Component_Path string
}

// True iff the import path names the component itself or a package within it.
func import_path_under_component(input *Import_Path_Under_Component_Input) (yes bool) {

	if input.Import_Path == input.Component_Path {
		return true
	}
	return strings.Has_Prefix(input.Import_Path, input.Component_Path+"/")
}

// Time_Gateway keeps the host clock in the simulation subtree.
func Time_Gateway(components *Component_Index) (gateway string) {
	return shared_library_directory(components, "simulation/time/default")
}

// Returns the workspace-relative directory named by relative within the shared library
// component, or "" when no module is the shared library. The gateways all sit at a fixed
// path below the shared root, thus one resolver keeps them from drifting apart.
func shared_library_directory(
	components *Component_Index, relative string,
) (directory string) {

	for _, m := range components.Components {
		if !m.Is_Shared_Library {
			continue
		}
		if m.Root == "." {
			return relative
		}
		return m.Root + "/" + relative
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

// IO_Gateway keeps the operating-system backend in the simulation subtree.
func IO_Gateway(components *Component_Index) (gateway string) {
	return shared_library_directory(components, "simulation/nbio/default")
}

// Syscall_Gateways permits the process bindings that simulation/nbio cannot supply.
// The raw-IO ban stays active for all other imports in these directories.
func Syscall_Gateways(components *Component_Index) (gateways []string) {
	directory := shared_library_directory(components, "os/default")
	if directory == "" {
		return nil
	}
	return []string{directory}
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
	return method_signature_matches(&Method_Signature{
		Name:    function_declaration.Name.Name,
		Params:  strings.Join(params, ","),
		Results: strings.Join(results, ","),
	})
}

// Method_Signature is a method's name and its parameter and result lists as
// source text, for matching against known interface signatures.
type Method_Signature struct {
	// Name is the method name.
	Name string
	// Params is the parameter list as source text.
	Params string
	// Results is the result list as source text.
	Results string
}

func method_signature_matches(input *Method_Signature) (yes bool) {
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

// Import_Local_Name is an explicit import name, or package name required by repository layout.
// Default tier declares parent package name so its directory remains an implementation detail.
func Import_Local_Name(implementation *ast.ImportSpec, import_path string) (name string) {
	if implementation.Name != nil {
		return implementation.Name.Name
	}
	name = path.Base(import_path)
	if name == "default" {
		return path.Base(path.Dir(import_path))
	}
	return name
}

// Invariant_Name is the _Invariants bundle-function name for a type: Name_Invariants
// when the type is exported, name_invariants when it is unexported.
func Invariant_Name(type_name string) (name string) {
	if ast.IsExported(type_name) {
		return type_name + "_Invariants"
	}
	return type_name + "_invariants"
}

// Path_Matches_Glob reports whether filename matches patterns as an exact-path
// glob list: "pkg/sub" matches only that path, "pkg/**" its whole subtree, and
// "**" everything. Later matches refine earlier broad policy, letting one list
// exempt a tree, bind a subtree, then exempt framework packages inside it. A
// "!"-prefixed match clears the decision; a later positive match restores it.
func Path_Matches_Glob(filename string, patterns []string) (yes bool) {
	for _, entry := range patterns {
		parsed := Parse_Glob_Pattern(entry)
		hit, _ := Glob_Match(&Glob_Match_Input{Pattern: parsed.Core, Path: filename})
		if !hit {
			continue
		}
		yes = !parsed.Negate
	}
	return yes
}

// A Glob_Pattern is a lint.json glob entry parsed into the two facts the matcher
// needs: Core, the pattern reduced to a form Glob_Match runs against a full path,
// and Negate, whether the entry vetoes rather than grants a match. Every entry is
// relative to the repository root; matching at any depth is spelled out with a
// leading **/.
type Glob_Pattern struct {
	// Core is the pattern reduced to the form Glob_Match runs against a full path.
	Core string
	// Negate marks a "!"-prefixed entry, clearing any earlier matching grant.
	Negate bool
}

// Parse_Glob_Pattern reduces a raw lint.json entry to a Glob_Pattern. A leading
// "!" is stripped into Negate before the rest of the reduction runs, so Core never
// carries it. Every entry is relative to the repository root: a leading slash is
// redundant and stripped, and a trailing slash is stripped, so "go.mod" and
// "/go.mod" both name the root go.mod and never a nested one. Match at any depth is
// requested explicitly with a leading "**/". Assumes the entry is non-empty once
// any leading "!" is stripped.
func Parse_Glob_Pattern(raw string) (parsed Glob_Pattern) {
	parsed.Negate = strings.Has_Prefix(raw, "!")
	raw = strings.Trim_Prefix(raw, "!")
	trimmed := strings.Trim_Suffix(raw, "/")
	trimmed = strings.Trim_Prefix(trimmed, "/")
	parsed.Core = trimmed
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
	return glob_match_segments(&Glob_Match_Segments_Input{
		Pattern: strings.Split(input.Pattern, "/"),
		Name:    strings.Split(input.Path, "/"),
	})
}

// Glob_Match_Segments_Input holds a glob pattern and a candidate path, each
// already split into path segments, for the segment-level match.
type Glob_Match_Segments_Input struct {
	// Pattern is the glob split into path segments.
	Pattern []string
	// Name is the candidate path split into segments.
	Name []string
}

// Matches the Pattern segments against the Name segments with a two-pointer scan
// that backtracks across **, the segment-level analogue of wildcard matching. A
// ** is remembered as a resume point and first tried as matching zero segments;
// on a later mismatch the scan returns to it and lets the ** swallow one more
// name segment, which is how a single ** spans an unknown depth. Trailing **s
// match the empty remainder, which is why dir/** also matches dir itself.
func glob_match_segments(input *Glob_Match_Segments_Input) (matched bool, err error) {
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
