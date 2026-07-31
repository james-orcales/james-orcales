// Package assertion enforces the _Invariants doctrine — every in-scope type
// states its properties in a companion helper beside it, and every named
// function calls the helpers for its typed inputs and outputs — and the sibling simulation
// doctrine that witnesses those invariants. The caller hands over the
// already-parsed files and the component graph, so this package reads and parses
// nothing: it is a pure, deterministic function of its input.
package assertion

import (
	"go/ast"
	"go/token"
	"path"
	"slices"
	"sort"
	"strconv"
	"strings"

	"local/james-orcales/lint/internal/diagnostic"
	"local/james-orcales/lint/internal/source"
)

// ASSERTIONS_BUILDER_LINKS_MAX bounds static expansion because a helper larger than a function's
// own line budget is already too diffuse to remain an auditable invariant boundary.
const ASSERTIONS_BUILDER_LINKS_MAX = 70

// PARENTHESES_DEPTH_MAX keeps expression unwrapping independently bounded so lowering the
// fluent-builder budget does not change which otherwise-identical subjects and constants resolve.
const PARENTHESES_DEPTH_MAX = 255

// Parsed_File aliases the source package's type so the moved rule bodies name it
// unqualified, as they did in package lint.
type Parsed_File = source.Parsed_File

// Component_Index aliases the source package's component graph so the moved rule
// bodies name it unqualified.
type Component_Index = source.Component_Index

// Diagnostic aliases the diagnostic package's type so the moved rule bodies name
// it unqualified.
type Diagnostic = diagnostic.Diagnostic

// Check runs the cross-file invariant and simulation checks over the parsed set,
// in the order the doctrine aggregator ran them.
func Check(
	parsed_files []source.Parsed_File,
	components *source.Component_Index,
	exempt []string,
) (diags []diagnostic.Diagnostic) {
	diags = append(diags,
		check_value_invariants(parsed_files, components, exempt)...)
	diags = append(diags,
		check_struct_invariants(parsed_files, components, exempt)...)
	diags = append(diags,
		check_function_invariants(parsed_files, components, exempt)...)
	diags = append(diags,
		check_recorder_test_main(parsed_files, components, exempt)...)
	diags = append(diags, check_primitive_types(parsed_files, exempt)...)
	diags = append(diags,
		check_simulation(parsed_files, components, exempt)...)
	return diags
}

// Canonical helpers, rather than a second interpretation of their individual assertions, are the
// stable contract the linter can mandate across invariant implementation changes.
func check_value_invariants(
	parsed_files []Parsed_File, components *Component_Index, exempt []string,
) (diags []Diagnostic) {
	constants := invariant_package_constants(parsed_files)
	for _, file := range parsed_files {
		if strings.HasSuffix(file.Path, "_test.go") {
			continue
		}
		if source.Path_Matches_Glob(file.Path, exempt) {
			continue
		}
		directory := path.Dir(file.Path)
		diags = append(diags, invariant_file_diagnostics(
			file, components, constants[directory])...)
	}
	return diags
}

// Constants are indexed per package because an argument in one package must not borrow a
// same-named declaration from another package to satisfy the deliberate-boundary rule.
func invariant_package_constants(
	parsed_files []Parsed_File,
) (constants map[string]map[string]bool) {
	constants = map[string]map[string]bool{}
	for _, file := range parsed_files {
		if strings.HasSuffix(file.Path, "_test.go") {
			continue
		}
		directory := path.Dir(file.Path)
		if constants[directory] == nil {
			constants[directory] = map[string]bool{}
		}
		invariant_collect_constants(file.File, constants[directory])
	}
	return constants
}

func invariant_collect_constants(file *ast.File, constants map[string]bool) {
	for _, declaration := range file.Decls {
		general, is_general := declaration.(*ast.GenDecl)
		if !is_general {
			continue
		}
		if general.Tok != token.CONST {
			continue
		}
		for _, specification := range general.Specs {
			value, is_value := specification.(*ast.ValueSpec)
			if !is_value {
				continue
			}
			for _, name := range value.Names {
				constants[name.Name] = true
			}
		}
	}
}

func invariant_file_diagnostics(
	file Parsed_File, components *Component_Index, constants map[string]bool,
) (diags []Diagnostic) {
	for index, declaration := range file.File.Decls {
		general, is_general := declaration.(*ast.GenDecl)
		if !is_general {
			continue
		}
		if general.Tok != token.TYPE {
			continue
		}
		type_specification, is_type := general.Specs[0].(*ast.TypeSpec)
		if !is_type {
			continue
		}
		suffix, _, _ := invariant_type_kind(type_specification)
		if suffix == "" {
			continue
		}
		diags = append(diags, invariant_type_diagnostics(
			file, components, constants, index)...)
	}
	return diags
}

func invariant_type_diagnostics(
	file Parsed_File, components *Component_Index, constants map[string]bool,
	index int,
) (diags []Diagnostic) {
	general := file.File.Decls[index].(*ast.GenDecl)
	type_specification := general.Specs[0].(*ast.TypeSpec)
	helper := type_invariants_following_function(file.File, index)
	if helper == nil {
		return nil
	}
	if helper.Name.Name != source.Invariant_Name(type_specification.Name.Name) {
		return nil
	}
	value, _ := invariant_value_parameter(helper, type_specification.Name.Name)
	if value == "" {
		return nil
	}
	namespace := invariant_namespace_parameter(helper)
	if namespace == "" {
		return nil
	}
	imports := helper_import_paths(file.File)
	scope := &Invariant_Scope{
		Current_Package: helper_package_path(file, components),
		Imports:         imports,
		Default_Package: helper_default_package(components, imports),
		Shadowed:        function_value_names(helper),
		Constants:       constants,
	}
	found, constant := invariant_body_helper(helper, type_specification, scope)
	if found {
		if constant {
			return nil
		}
		return invariant_constant_diagnostic(file, helper)
	}
	return invariant_missing_helper_diagnostic(file, helper, type_specification)
}

// Direct underlying types determine the concrete helper without go/types; aliases and uintptr
// have no Assertions preset and therefore remain outside this body-shape mandate.
func invariant_type_kind(
	type_specification *ast.TypeSpec,
) (suffix string, primitive string, count bool) {
	if type_specification.Assign.IsValid() {
		return "", "", false
	}
	switch typed := type_specification.Type.(type) {
	case *ast.ArrayType:
		if typed.Len == nil {
			return "Int", "int", true
		}
	case *ast.MapType:
		return "Int", "int", true
	case *ast.Ident:
		return invariant_identifier_kind(typed.Name)
	}
	return "", "", false
}

func invariant_identifier_kind(name string) (suffix string, primitive string, count bool) {
	switch name {
	case "string":
		return "Int", "int", true
	case "int", "int8", "int16", "int32", "int64":
		return strings.ToUpper(name[:1]) + name[1:], name, false
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return strings.ToUpper(name[:1]) + name[1:], name, false
	case "byte":
		return "Uint8", "uint8", false
	case "rune":
		return "Int32", "int32", false
	case "float32", "float64":
		return strings.ToUpper(name[:1]) + name[1:], name, false
	case "bool":
		return "Boolean", "bool", false
	}
	return "", "", false
}

func invariant_value_parameter(
	helper *ast.FuncDecl, type_name string,
) (name string, pointer bool) {
	if helper.Type.Params == nil {
		return "", false
	}
	if len(helper.Type.Params.List) == 0 {
		return "", false
	}
	first := helper.Type.Params.List[0]
	expression := first.Type
	star, is_star := expression.(*ast.StarExpr)
	if is_star {
		expression = star.X
		pointer = true
	}
	if type_base_name(expression) != type_name {
		return "", false
	}
	if len(first.Names) == 0 {
		return "", false
	}
	return first.Names[0].Name, pointer
}

func invariant_namespace_parameter(helper *ast.FuncDecl) (name string) {
	if helper.Type.Params == nil {
		return ""
	}
	parameters := helper.Type.Params.List
	if len(parameters) == 0 {
		return ""
	}
	last := parameters[len(parameters)-1]
	if len(last.Names) == 0 {
		return ""
	}
	return last.Names[0].Name
}

func invariant_body_helper(
	helper *ast.FuncDecl, type_specification *ast.TypeSpec, scope *Invariant_Scope,
) (found bool, constant bool) {
	shadowed := function_shadow_copy(scope.Shadowed)
	for _, statement := range helper.Body.List {
		call := statement_call(statement)
		if call != nil {
			statement_scope := *scope
			statement_scope.Shadowed = shadowed
			if invariant_direct_preset(
				call, helper, type_specification, &statement_scope) {
				return true, true
			}
			matched, valid := invariant_builder_preset(
				call, helper, type_specification, &statement_scope)
			if matched {
				if valid {
					return true, true
				}
				found = true
			}
		}
		function_statement_shadows(statement, shadowed)
	}
	return found, false
}

func invariant_direct_preset(
	call *ast.CallExpr, helper *ast.FuncDecl,
	type_specification *ast.TypeSpec, scope *Invariant_Scope,
) (matched bool) {
	suffix, _, count := invariant_type_kind(type_specification)
	if count {
		return false
	}
	want := scope.Default_Package + "\x00" + suffix + "_Invariants"
	identity := helper_callee_identity(
		call.Fun, scope.Current_Package, scope.Imports, scope.Shadowed)
	if identity != want {
		return false
	}
	if len(call.Args) != 2 {
		return false
	}
	if !invariant_subject(call.Args[0], helper, type_specification, scope) {
		return false
	}
	namespace := invariant_namespace_parameter(helper)
	return invariant_identifier(call.Args[1], namespace)
}

func invariant_builder_preset(
	ensure *ast.CallExpr, helper *ast.FuncDecl,
	type_specification *ast.TypeSpec, scope *Invariant_Scope,
) (matched bool, constant bool) {
	current, ensured := invariant_ensure_receiver(ensure)
	if !ensured {
		return false, false
	}
	valid := false
	for step_index := 0; step_index < ASSERTIONS_BUILDER_LINKS_MAX; step_index++ {
		_, receiver, linked := invariant_builder_method(current)
		if !linked {
			break
		}
		preset, arguments_valid := invariant_builder_link(
			current, helper, type_specification, scope)
		if preset {
			matched = true
			if arguments_valid {
				valid = true
			}
		}
		current = receiver
	}
	if !invariant_builder_root(current, helper, scope) {
		return false, false
	}
	return matched, valid
}

func invariant_ensure_receiver(
	ensure *ast.CallExpr,
) (receiver *ast.CallExpr, matched bool) {
	selector, is_selector := ensure.Fun.(*ast.SelectorExpr)
	if !is_selector {
		return nil, false
	}
	if selector.Sel.Name != "Ensure" {
		return nil, false
	}
	if len(ensure.Args) != 0 {
		return nil, false
	}
	receiver, matched = selector.X.(*ast.CallExpr)
	return receiver, matched
}

func invariant_builder_method(
	call *ast.CallExpr,
) (method string, receiver *ast.CallExpr, matched bool) {
	selector, is_selector := call.Fun.(*ast.SelectorExpr)
	if !is_selector {
		return "", nil, false
	}
	receiver, matched = selector.X.(*ast.CallExpr)
	return selector.Sel.Name, receiver, matched
}

func invariant_builder_link(
	call *ast.CallExpr, helper *ast.FuncDecl,
	type_specification *ast.TypeSpec, scope *Invariant_Scope,
) (matched bool, constant bool) {
	method, _, _ := invariant_builder_method(call)
	suffix, primitive, _ := invariant_type_kind(type_specification)
	range_name := "Range_" + suffix
	range_holed_name := "Range_Holed_" + suffix
	enum_name := "Enum_" + suffix
	enum_3_name := "Enum_3_" + suffix
	enum_4_name := "Enum_4_" + suffix
	switch method {
	case range_name, range_holed_name:
		argument_count := 3
		if method == range_holed_name {
			argument_count = 7
			if strings.HasPrefix(suffix, "Uint") {
				argument_count = 6
			}
		}
		if len(call.Args) != argument_count {
			return false, false
		}
		if !invariant_subject(call.Args[0], helper, type_specification, scope) {
			return false, false
		}
		return true, invariant_arguments_constant(
			call.Args[1:3], primitive, scope)
	}
	member_count := 0
	if method == enum_name {
		member_count = 2
	}
	if method == enum_3_name {
		member_count = 3
	}
	if method == enum_4_name {
		member_count = 4
	}
	if member_count == 0 {
		return false, false
	}
	if len(call.Args) != member_count+1 {
		return false, false
	}
	if !invariant_subject(call.Args[0], helper, type_specification, scope) {
		return false, false
	}
	return true, invariant_arguments_constant(call.Args[1:], primitive, scope)
}

func invariant_builder_root(
	call *ast.CallExpr, helper *ast.FuncDecl, scope *Invariant_Scope,
) (matched bool) {
	if len(call.Args) != 1 {
		return false
	}
	if !invariant_identifier(call.Args[0], invariant_namespace_parameter(helper)) {
		return false
	}
	selector, is_selector := call.Fun.(*ast.SelectorExpr)
	if !is_selector {
		return false
	}
	if selector.Sel.Name != "Assertions" {
		return false
	}
	qualifier, is_qualifier := selector.X.(*ast.Ident)
	if !is_qualifier {
		return false
	}
	if qualifier.Name != "invariant" {
		return false
	}
	if scope.Shadowed[qualifier.Name] {
		return false
	}
	package_path := scope.Imports[qualifier.Name]
	return package_path == scope.Default_Package
}

func invariant_subject(
	expression ast.Expr, helper *ast.FuncDecl, type_specification *ast.TypeSpec,
	scope *Invariant_Scope,
) (matched bool) {
	_, primitive, count := invariant_type_kind(type_specification)
	value, pointer := invariant_value_parameter(helper, type_specification.Name.Name)
	expression = invariant_unparen(expression)
	if count {
		call, is_call := expression.(*ast.CallExpr)
		if !is_call {
			return false
		}
		identifier, is_identifier := call.Fun.(*ast.Ident)
		if !is_identifier {
			return false
		}
		if identifier.Name != "len" {
			return false
		}
		if scope.Shadowed[identifier.Name] {
			return false
		}
		if len(call.Args) != 1 {
			return false
		}
		return invariant_value(call.Args[0], value, pointer)
	}
	call, is_call := expression.(*ast.CallExpr)
	if !is_call {
		return false
	}
	conversion, is_conversion := call.Fun.(*ast.Ident)
	if !is_conversion {
		return false
	}
	if conversion.Name != primitive {
		return false
	}
	if scope.Shadowed[conversion.Name] {
		return false
	}
	if len(call.Args) != 1 {
		return false
	}
	return invariant_value(call.Args[0], value, pointer)
}

func invariant_value(expression ast.Expr, value string, pointer bool) (matched bool) {
	expression = invariant_unparen(expression)
	if pointer {
		star, is_star := expression.(*ast.StarExpr)
		if !is_star {
			return false
		}
		expression = invariant_unparen(star.X)
	}
	return invariant_identifier(expression, value)
}

func invariant_identifier(expression ast.Expr, name string) (matched bool) {
	expression = invariant_unparen(expression)
	identifier, is_identifier := expression.(*ast.Ident)
	if !is_identifier {
		return false
	}
	return identifier.Name == name
}

func invariant_unparen(expression ast.Expr) (unwrapped ast.Expr) {
	for depth_index := 0; depth_index < PARENTHESES_DEPTH_MAX; depth_index++ {
		parenthesized, is_parenthesized := expression.(*ast.ParenExpr)
		if !is_parenthesized {
			return expression
		}
		expression = parenthesized.X
	}
	return expression
}

func invariant_arguments_constant(
	arguments []ast.Expr, primitive string, scope *Invariant_Scope,
) (valid bool) {
	for _, argument := range arguments {
		if !invariant_argument_constant(argument, primitive, scope) {
			return false
		}
	}
	return true
}

func invariant_argument_constant(
	expression ast.Expr, primitive string, scope *Invariant_Scope,
) (valid bool) {
	expression = invariant_unparen(expression)
	call, is_call := expression.(*ast.CallExpr)
	if is_call {
		if len(call.Args) != 1 {
			return false
		}
		conversion, is_conversion := call.Fun.(*ast.Ident)
		if !is_conversion {
			return false
		}
		if conversion.Name != primitive {
			return false
		}
		if scope.Shadowed[conversion.Name] {
			return false
		}
		expression = invariant_unparen(call.Args[0])
	}
	identifier, is_identifier := expression.(*ast.Ident)
	if !is_identifier {
		return false
	}
	if scope.Shadowed[identifier.Name] {
		return false
	}
	return scope.Constants[identifier.Name]
}

func invariant_constant_diagnostic(
	file Parsed_File, helper *ast.FuncDecl,
) (diags []Diagnostic) {
	return []Diagnostic{{
		Position: file.File_Set.Position(helper.Name.Pos()),
		Message: helper.Name.Name +
			" canonical helper arguments must be package-level constants",
	}}
}

func invariant_missing_helper_diagnostic(
	file Parsed_File, helper *ast.FuncDecl, type_specification *ast.TypeSpec,
) (diags []Diagnostic) {
	suffix, _, count := invariant_type_kind(type_specification)
	value, _ := invariant_value_parameter(helper, type_specification.Name.Name)
	message := helper.Name.Name + " must call a canonical helper for " + value +
		": " + suffix + "_Invariants, a Range_" + suffix +
		" family, or an Enum_" + suffix + " family"
	if count {
		message = helper.Name.Name + " must call Range_Int or Enum_Int family for len(" +
			value + ") in a direct invariant.Assertions(namespace) builder " +
			"ending in Ensure()"
	}
	return []Diagnostic{{
		Position: file.File_Set.Position(helper.Name.Pos()), Message: message,
	}}
}

// Check_Type enforces the per-file type-invariant rules (Presence, Casing,
// Signature, Orphan, Scope): every in-scope type is followed directly by its
// correctly-named, correctly-signed bundle, and every bundle sits below its type.
// Test files and files under an exempt package are skipped.
func Check_Type(
	file_set *token.FileSet, file *ast.File, exempt []string,
) (diags []diagnostic.Diagnostic) {
	filename := file_set.Position(file.Pos()).Filename
	if strings.HasSuffix(filename, "_test.go") {
		return nil
	}
	if source.Path_Matches_Glob(filename, exempt) {
		return nil
	}
	invariant_names := type_invariants_import_names(file)
	diags = append(diags,
		check_type_invariants_forward(file_set, file, invariant_names)...)
	return append(diags, check_type_invariants_orphan(file_set, file)...)
}

// Returns the declared base name beneath pointers and generic instantiations because companion
// helper identity follows the named type rather than its spelling at one use site.
func type_base_name(expression ast.Expr) (name string) {
	star, is_star := expression.(*ast.StarExpr)
	if is_star {
		expression = star.X
	}
	index, is_index := expression.(*ast.IndexExpr)
	if is_index {
		expression = index.X
	}
	index_list, is_index_list := expression.(*ast.IndexListExpr)
	if is_index_list {
		expression = index_list.X
	}
	identifier, is_identifier := expression.(*ast.Ident)
	if !is_identifier {
		return ""
	}
	return identifier.Name
}

func helper_identity_name(identity string) (name string) {
	separator := strings.LastIndexByte(identity, '\x00')
	if separator < 0 {
		return identity
	}
	return identity[separator+1:]
}

// The component root carries the import prefix while the file directory supplies the package
// suffix. The path fallback preserves exact same-directory identity for isolated lint fixtures.
func helper_package_path(file Parsed_File, components *Component_Index) (import_path string) {
	if components != nil {
		component_index_number, mapped := components.File_To_Component[file.Path]
		if mapped {
			if component_index_number < 0 {
				return path.Dir(file.Path)
			}
			if component_index_number >= len(components.Components) {
				return path.Dir(file.Path)
			}
			component := components.Components[component_index_number]
			directory := path.Dir(file.Path)
			if directory == component.Root {
				return component.Import_Path
			}
			relative := strings.TrimPrefix(directory, component.Root+"/")
			if relative == directory {
				return path.Dir(file.Path)
			}
			if relative == "." {
				return path.Dir(file.Path)
			}
			return component.Import_Path + "/" + relative
		}
	}
	return path.Dir(file.Path)
}

func helper_import_paths(file *ast.File) (imports map[string]string) {
	imports = map[string]string{}
	for _, specification := range file.Imports {
		import_path, unquote_error := strconv.Unquote(specification.Path.Value)
		if unquote_error != nil {
			continue
		}
		local := source.Import_Local_Name(specification, import_path)
		if local == "." {
			continue
		}
		if local == "_" {
			continue
		}
		imports[local] = import_path
	}
	return imports
}

func helper_default_package(
	components *Component_Index, imports map[string]string,
) (import_path string) {
	shared := source.Shared_Import(components)
	if shared != "" {
		return shared + "/invariant/default"
	}
	for _, candidate := range imports {
		if strings.HasSuffix(candidate, "/invariant/default") {
			return candidate
		}
		if strings.HasSuffix(candidate, "/invariant") {
			import_path = candidate + "/default"
		}
	}
	if import_path != "" {
		return import_path
	}
	return "<invariant/default>"
}

// A selector resolves only through the file's import table. A method whose receiver happens to
// share an import alias or helper suffix therefore cannot impersonate a package helper.
func helper_callee_identity(
	callee ast.Expr,
	current_package string,
	imports map[string]string,
	shadowed map[string]bool,
) (identity string) {
	index, is_index := callee.(*ast.IndexExpr)
	if is_index {
		callee = index.X
	}
	index_list, is_index_list := callee.(*ast.IndexListExpr)
	if is_index_list {
		callee = index_list.X
	}
	identifier, is_identifier := callee.(*ast.Ident)
	if is_identifier {
		if shadowed[identifier.Name] {
			return ""
		}
		return current_package + "\x00" + identifier.Name
	}
	selector, is_selector := callee.(*ast.SelectorExpr)
	if !is_selector {
		return ""
	}
	qualifier, is_qualifier := selector.X.(*ast.Ident)
	if !is_qualifier {
		return ""
	}
	if shadowed[qualifier.Name] {
		return ""
	}
	import_path := imports[qualifier.Name]
	if import_path == "" {
		return ""
	}
	return import_path + "\x00" + selector.Sel.Name
}

// Flags a struct helper that fails to call the exact _Invariants helper of a field whose type
// has one. Existence-driven: a field is required only when an _Invariants for its
// type exists in the module (presets always do), so adding one later auto-enables
// the field. A struct with an immediate sync.Mutex/RWMutex field is skipped whole.
// Cross-file because the bundle index spans the whole module.
func check_struct_invariants(
	parsed_files []Parsed_File, components *Component_Index, exempt []string,
) (diags []Diagnostic) {
	defined := struct_helper_index(parsed_files, components)
	for _, pf := range parsed_files {
		if strings.HasSuffix(pf.Path, "_test.go") {
			continue
		}
		if source.Path_Matches_Glob(pf.Path, exempt) {
			continue
		}
		diags = append(diags, struct_file_diagnostics(pf, defined, components)...)
	}
	return diags
}

// Package-qualified keys ensure adding Foo_Invariants in one package cannot silently impose or
// satisfy a helper requirement for an unrelated Foo in another package.
func struct_helper_index(
	parsed_files []Parsed_File, components *Component_Index,
) (defined map[string]bool) {
	defined = map[string]bool{}
	for _, pf := range parsed_files {
		if strings.HasSuffix(pf.Path, "_test.go") {
			continue
		}
		for _, declaration := range pf.File.Decls {
			function, is_function := declaration.(*ast.FuncDecl)
			if !is_function {
				continue
			}
			if function.Recv != nil {
				continue
			}
			if type_invariants_is_bundle_name(function.Name.Name) {
				identity := helper_package_path(pf, components) +
					"\x00" + function.Name.Name
				defined[identity] = true
			}
		}
	}
	return defined
}

// Checks every struct type + value/pointer-parameter bundle pair in one file.
func struct_file_diagnostics(
	file Parsed_File, defined map[string]bool, components *Component_Index,
) (diags []Diagnostic) {
	for index, declaration := range file.File.Decls {
		general, is_general := declaration.(*ast.GenDecl)
		if !is_general {
			continue
		}
		if general.Tok != token.TYPE {
			continue
		}
		type_specification, is_type := general.Specs[0].(*ast.TypeSpec)
		if !is_type {
			continue
		}
		struct_type, is_struct := type_specification.Type.(*ast.StructType)
		if !is_struct {
			continue
		}
		if len(struct_type.Fields.List) == 0 {
			continue
		}
		if struct_has_mutex(struct_type) {
			continue
		}
		diags = append(diags, struct_type_diagnostics(
			file,
			index,
			type_specification,
			struct_type,
			defined,
			components,
		)...)
	}
	return diags
}

// Checks that one struct's bundle composes every coverable field.
func struct_type_diagnostics(
	file Parsed_File,
	index int,
	type_specification *ast.TypeSpec,
	struct_type *ast.StructType,
	defined map[string]bool,
	components *Component_Index,
) (diags []Diagnostic) {
	bundle := type_invariants_following_function(file.File, index)
	if bundle == nil {
		return nil
	}
	if bundle.Name.Name != source.Invariant_Name(type_specification.Name.Name) {
		return nil
	}
	parameter := struct_parameter_name(bundle, type_specification.Name.Name)
	if parameter == "" {
		return nil
	}
	current_package := helper_package_path(file, components)
	imports := helper_import_paths(file.File)
	scope := &Invariant_Scope{
		Type_Parameters: struct_type_parameter_set(type_specification),
		Defined:         defined,
		Current_Package: current_package,
		Imports:         imports,
		Default_Package: helper_default_package(components, imports),
		Shadowed:        function_value_names(bundle),
	}
	present := struct_present_calls(bundle, parameter, scope)
	position := file.File_Set.Position(bundle.Name.Pos())
	for _, field := range struct_type.Fields.List {
		for _, call := range struct_field_missing_calls(field, scope, present, parameter) {
			diags = append(diags, Diagnostic{
				Position: position,
				Message:  bundle.Name.Name + " must call " + call,
			})
		}
	}
	return diags
}

// Separating field discovery from diagnostic ownership keeps bundle identity and position out of
// an argument-bundling struct.
func struct_field_missing_calls(
	field *ast.Field,
	scope *Invariant_Scope,
	present map[string]bool,
	parameter string,
) (calls []string) {
	if len(field.Names) == 0 {
		return nil
	}
	expected, preset := struct_field_invariant(field.Type, scope)
	if expected == "" {
		return nil
	}
	if !preset {
		if !scope.Defined[expected] {
			return nil
		}
	}
	for _, name := range field.Names {
		if present[expected+"\x00"+name.Name] {
			continue
		}
		calls = append(calls, helper_identity_name(expected)+"("+
			parameter+"."+name.Name+", ...)")
	}
	return calls
}

// Reports whether an immediate field is a sync.Mutex or sync.RWMutex.
func struct_has_mutex(struct_type *ast.StructType) (yes bool) {
	for _, field := range struct_type.Fields.List {
		if struct_is_mutex(field.Type) {
			return true
		}
	}
	return false
}

// Reports whether field_type is sync.Mutex or sync.RWMutex.
func struct_is_mutex(field_type ast.Expr) (yes bool) {
	selector, is_selector := field_type.(*ast.SelectorExpr)
	if !is_selector {
		return false
	}
	qualifier, is_identifier := selector.X.(*ast.Ident)
	if !is_identifier {
		return false
	}
	if qualifier.Name != "sync" {
		return false
	}
	if selector.Sel.Name == "Mutex" {
		return true
	}
	return selector.Sel.Name == "RWMutex"
}

// Returns the struct bundle's first parameter name when it is the struct by value
// or pointer (possibly a generic instantiation), or "" otherwise.
func struct_parameter_name(bundle *ast.FuncDecl, type_name string) (name string) {
	if bundle.Type.Params == nil {
		return ""
	}
	if len(bundle.Type.Params.List) == 0 {
		return ""
	}
	first := bundle.Type.Params.List[0]
	parameter_type := first.Type
	star, is_star := parameter_type.(*ast.StarExpr)
	if is_star {
		parameter_type = star.X
	}
	if type_base_name(parameter_type) != type_name {
		return ""
	}
	if len(first.Names) == 0 {
		return ""
	}
	return first.Names[0].Name
}

// Returns the struct's own type-parameter names, so a field typed as one is exempt.
func struct_type_parameter_set(type_specification *ast.TypeSpec) (parameters map[string]bool) {
	parameters = map[string]bool{}
	if type_specification.TypeParams == nil {
		return parameters
	}
	for _, field := range type_specification.TypeParams.List {
		for _, name := range field.Names {
			parameters[name.Name] = true
		}
	}
	return parameters
}

// Collects "callee\x00field" for every call in the bundle whose first argument is
// the parameter's field (value or deref), so a composition call can be looked up.
func struct_present_calls(
	bundle *ast.FuncDecl, parameter string, scope *Invariant_Scope,
) (present map[string]bool) {
	present = map[string]bool{}
	shadowed := function_shadow_copy(scope.Shadowed)
	for _, statement := range bundle.Body.List {
		call := statement_call(statement)
		if call != nil {
			callee := helper_callee_identity(
				call.Fun, scope.Current_Package, scope.Imports, shadowed)
			field := struct_first_argument_field(call, parameter)
			if callee != "" {
				if field != "" {
					present[callee+"\x00"+field] = true
				}
			}
		}
		function_statement_shadows(statement, shadowed)
	}
	return present
}

// Returns a call's final callee name: a bare ident, a selector's member, or the
// base of a generic call; "" otherwise.
func struct_callee_name(callee ast.Expr) (name string) {
	// A generic call Foo_Invariants[T](...) wraps the callee in an index; unwrap it.
	index, is_index := callee.(*ast.IndexExpr)
	if is_index {
		callee = index.X
	}
	index_list, is_index_list := callee.(*ast.IndexListExpr)
	if is_index_list {
		callee = index_list.X
	}
	switch typed := callee.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.SelectorExpr:
		return typed.Sel.Name
	default:
		return ""
	}
}

// Returns the field name when a call's first argument is parameter.Field or
// *parameter.Field; "" otherwise.
func struct_first_argument_field(call *ast.CallExpr, parameter string) (field string) {
	if len(call.Args) == 0 {
		return ""
	}
	argument := call.Args[0]
	star, is_star := argument.(*ast.StarExpr)
	if is_star {
		argument = star.X
	}
	selector, is_selector := argument.(*ast.SelectorExpr)
	if !is_selector {
		return ""
	}
	base, is_identifier := selector.X.(*ast.Ident)
	if !is_identifier {
		return ""
	}
	if base.Name != parameter {
		return ""
	}
	return selector.Sel.Name
}

// Returns the _Invariants name a field of the given type must call and whether it
// is a preset (always available), or "" when the field is exempt.
func struct_field_invariant(
	field_type ast.Expr, scope *Invariant_Scope,
) (identity string, preset bool) {

	// A pointer field composes its pointee: *Token requires Token_Invariants, the
	// same as a Token field. The bundle passes the field, or its dereference, as the
	// value — struct_present_calls accepts both.
	star, is_pointer := field_type.(*ast.StarExpr)
	if is_pointer {
		field_type = star.X
	}
	switch typed := field_type.(type) {
	case *ast.ArrayType:
		// A raw slice field is banned by check_primitive_types, not composed here.
		return "", false
	case *ast.MapType:
		// A raw map field is banned by check_primitive_types, not composed here.
		return "", false
	case *ast.SelectorExpr:
		package_path := struct_selector_package(typed, scope.Imports)
		if package_path == "" {
			return "", false
		}
		return package_path + "\x00" + typed.Sel.Name + "_Invariants", false
	case *ast.IndexExpr:
		return struct_named_invariant(typed.X, scope)
	case *ast.IndexListExpr:
		return struct_named_invariant(typed.X, scope)
	case *ast.Ident:
		return struct_field_ident_invariant(typed.Name, scope)
	default:
		return "", false
	}
}

// Returns the bundle name for a generic instantiation's base (an ident or a
// cross-package selector), or "" otherwise.
func struct_named_invariant(
	base ast.Expr, scope *Invariant_Scope,
) (identity string, preset bool) {

	selector, is_selector := base.(*ast.SelectorExpr)
	if is_selector {
		package_path := struct_selector_package(selector, scope.Imports)
		if package_path == "" {
			return "", false
		}
		return package_path + "\x00" + selector.Sel.Name + "_Invariants", false
	}
	identifier, is_identifier := base.(*ast.Ident)
	if is_identifier {
		return struct_field_ident_invariant(identifier.Name, scope)
	}
	return "", false
}

func struct_selector_package(
	selector *ast.SelectorExpr, imports map[string]string,
) (package_path string) {
	qualifier, is_qualifier := selector.X.(*ast.Ident)
	if !is_qualifier {
		return ""
	}
	return imports[qualifier.Name]
}

// Maps a field ident to its expected bundle: a struct type param is exempt, a
// primitive maps to its preset, a no-preset builtin is exempt, else it is a
// defined type whose own bundle (by casing) is expected.
func struct_field_ident_invariant(
	name string, scope *Invariant_Scope,
) (identity string, preset bool) {

	if scope.Type_Parameters[name] {
		return "", false
	}
	if name == "string" {
		// A raw string field is banned by check_primitive_types, not composed here.
		return "", false
	}
	mapped := struct_primitive_preset(name)
	if mapped != "" {
		return scope.Default_Package + "\x00" + mapped, true
	}
	if struct_is_builtin(name) {
		return "", false
	}
	return scope.Current_Package + "\x00" + source.Invariant_Name(name), false
}

// Maps a builtin primitive to its framework preset name, or "" when none.
func struct_primitive_preset(name string) (preset string) {
	switch name {
	case "int":
		return "Int_Invariants"
	case "int8":
		return "Int8_Invariants"
	case "int16":
		return "Int16_Invariants"
	case "int32", "rune":
		return "Int32_Invariants"
	case "int64":
		return "Int64_Invariants"
	case "uint":
		return "Uint_Invariants"
	case "uint8", "byte":
		return "Uint8_Invariants"
	case "uint16":
		return "Uint16_Invariants"
	case "uint32":
		return "Uint32_Invariants"
	case "uint64":
		return "Uint64_Invariants"
	case "float32":
		return "Float32_Invariants"
	case "float64":
		return "Float64_Invariants"
	case "bool":
		return "Boolean_Invariants"
	default:
		return ""
	}
}

// Reports whether name is a predeclared type that has no preset, so a field of it
// is exempt rather than mistaken for a defined type.
func struct_is_builtin(name string) (yes bool) {
	switch name {
	case "uintptr", "complex64", "complex128", "error", "any", "comparable":
		return true
	default:
		return false
	}
}

// Flags an ordinary function that omits the exact helper for an input or named return. Output
// helpers stay deferred and input helpers stay leading so every function has one visible boundary.
func check_function_invariants(
	parsed_files []Parsed_File, components *Component_Index, exempt []string,
) (diags []Diagnostic) {
	defined := struct_helper_index(parsed_files, components)
	for _, pf := range parsed_files {
		if strings.HasSuffix(pf.Path, "_test.go") {
			continue
		}
		if source.Path_Matches_Glob(pf.Path, exempt) {
			continue
		}
		diags = append(diags, function_file_diagnostics(pf, defined, components)...)
	}
	return diags
}

// Checks every named free function in one file.
func function_file_diagnostics(
	file Parsed_File, defined map[string]bool, components *Component_Index,
) (diags []Diagnostic) {
	for _, declaration := range file.File.Decls {
		function, is_function := declaration.(*ast.FuncDecl)
		if !is_function {
			continue
		}
		if function.Recv != nil {
			continue
		}
		if function.Body == nil {
			continue
		}
		// A bundle asserting its own value would be self-referential. Methods are
		// excluded above; main and init have no parameters or results to assert,
		// and TestMain lives in a _test.go file the whole check already skips.
		if type_invariants_is_bundle_name(function.Name.Name) {
			continue
		}
		diags = append(diags, function_diagnostics(
			file,
			function,
			defined,
			components,
		)...)
	}
	return diags
}

// Identity resolution must travel together; separating any map from this lexical scope lets
// shadowing or a same-spelled declaration satisfy only one side of the mandate.
type Invariant_Scope struct {
	// Type_Parameters is the set of the function's type-parameter names.
	Type_Parameters map[string]bool
	// Defined is the set of type names with an invariant bundle in the module.
	Defined map[string]bool
	// Current_Package qualifies bare helper calls.
	Current_Package string
	// Imports resolves selector types and calls.
	Imports map[string]string
	// Default_Package owns every primitive preset helper.
	Default_Package string
	// Shadowed prevents a local value from impersonating a package or helper identifier.
	Shadowed map[string]bool
	// Constants pins Range boundaries and Enum members to declarations in the type's package.
	Constants map[string]bool
}

// One subject and the exact package-qualified helper it must carry.
type Helper_Requirement struct {
	// Subject is the parameter or named-return name that needs its helper.
	Subject string
	// Expected is the exact helper identity.
	Expected string
}

// Collects the helper gaps for one function's inputs and outputs.
func function_diagnostics(
	file Parsed_File,
	function *ast.FuncDecl,
	defined map[string]bool,
	components *Component_Index,
) (diags []Diagnostic) {
	imports := helper_import_paths(file.File)
	scope := &Invariant_Scope{
		Type_Parameters: function_type_parameter_set(function),
		Defined:         defined,
		Current_Package: helper_package_path(file, components),
		Imports:         imports,
		Default_Package: helper_default_package(components, imports),
		Shadowed:        function_value_names(function),
	}
	inputs := function_requirements(function.Type.Params, scope)
	outputs := function_requirements(function.Type.Results, scope)
	if len(inputs) == 0 {
		if len(outputs) == 0 {
			return nil
		}
	}
	position := file.File_Set.Position(function.Name.Pos())
	name := function.Name.Name
	lead, defer_body, has_defer := function_lead_and_defer(function.Body.List)
	for _, requirement := range outputs {
		if !has_defer {
			diags = append(diags, Diagnostic{Position: position,
				Message: name + " must call helper for " + requirement.Subject +
					" in a first-statement defer"})
			continue
		}
		if !function_requirement_met(defer_body, requirement, scope) {
			diags = append(diags, Diagnostic{Position: position,
				Message: name + " must call helper for " + requirement.Subject +
					" in the output defer"})
		}
	}
	for _, requirement := range inputs {
		if function_requirement_met(lead, requirement, scope) {
			continue
		}
		diags = append(diags, Diagnostic{Position: position,
			Message: name + " must call helper for " + requirement.Subject + " via " +
				function_form(requirement)})
	}
	return diags
}

// Returns the function's own type-parameter names.
func function_type_parameter_set(function *ast.FuncDecl) (parameters map[string]bool) {
	parameters = map[string]bool{}
	if function.Type.TypeParams == nil {
		return parameters
	}
	for _, field := range function.Type.TypeParams.List {
		for _, name := range field.Names {
			parameters[name.Name] = true
		}
	}
	return parameters
}

// Package identifiers and package-level helpers stop denoting those declarations when a function
// parameter or named result shadows them for the whole body.
func function_value_names(function *ast.FuncDecl) (names map[string]bool) {
	names = map[string]bool{}
	function_add_field_names(function.Type.Params, names)
	function_add_field_names(function.Type.Results, names)
	return names
}

func function_add_field_names(fields *ast.FieldList, names map[string]bool) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		for _, name := range field.Names {
			names[name.Name] = true
		}
	}
}

func function_shadow_copy(names map[string]bool) (copy map[string]bool) {
	copy = map[string]bool{}
	for name := range names {
		copy[name] = true
	}
	return copy
}

// A declaration begins shadowing after its own statement, so callers resolve the statement first
// and extend this set second.
func function_statement_shadows(statement ast.Stmt, names map[string]bool) {
	assignment, is_assignment := statement.(*ast.AssignStmt)
	if is_assignment {
		if assignment.Tok != token.DEFINE {
			return
		}
		for _, expression := range assignment.Lhs {
			identifier, is_identifier := expression.(*ast.Ident)
			if is_identifier {
				names[identifier.Name] = true
			}
		}
		return
	}
	declaration, is_declaration := statement.(*ast.DeclStmt)
	if !is_declaration {
		return
	}
	general, is_general := declaration.Decl.(*ast.GenDecl)
	if !is_general {
		return
	}
	for _, specification := range general.Specs {
		switch typed := specification.(type) {
		case *ast.ValueSpec:
			for _, name := range typed.Names {
				names[name.Name] = true
			}
		case *ast.TypeSpec:
			names[typed.Name.Name] = true
		}
	}
}

// Builds the helper requirements for a parameter or result list, skipping
// blank and exempt subjects.
func function_requirements(
	fields *ast.FieldList, scope *Invariant_Scope,
) (requirements []Helper_Requirement) {

	if fields == nil {
		return nil
	}
	for _, field := range fields.List {
		expected, required := function_requirement(field.Type, scope)
		if !required {
			continue
		}
		for _, name := range field.Names {
			if name.Name == "_" {
				continue
			}
			requirements = append(requirements, Helper_Requirement{
				Subject: name.Name, Expected: expected,
			})
		}
	}
	return requirements
}

// Derives one subject's requirement: a defined type asks for a flat _Invariants
// call. A raw slice, variadic, or map is banned by check_primitive_types rather
// than asserted here, so it carries no requirement.
func function_requirement(
	field_type ast.Expr, scope *Invariant_Scope,
) (expected string, required bool) {

	core := field_type
	star, is_star := core.(*ast.StarExpr)
	if is_star {
		core = star.X
	}
	if _, is_array := core.(*ast.ArrayType); is_array {
		return "", false
	}
	if _, is_ellipsis := core.(*ast.Ellipsis); is_ellipsis {
		return "", false
	}
	if _, is_map := core.(*ast.MapType); is_map {
		return "", false
	}
	return function_named_invariant(core, scope)
}

// Maps a named type expression (ident, selector, pointer, or generic
// instantiation) to its _Invariants name and whether one exists.
func function_named_invariant(
	type_expression ast.Expr, scope *Invariant_Scope,
) (expected string, required bool) {

	core := type_expression
	star, is_star := core.(*ast.StarExpr)
	if is_star {
		core = star.X
	}
	index, is_index := core.(*ast.IndexExpr)
	if is_index {
		core = index.X
	}
	index_list, is_index_list := core.(*ast.IndexListExpr)
	if is_index_list {
		core = index_list.X
	}
	selector, is_selector := core.(*ast.SelectorExpr)
	if is_selector {
		package_path := struct_selector_package(selector, scope.Imports)
		if package_path == "" {
			return "", false
		}
		identity := package_path + "\x00" + selector.Sel.Name + "_Invariants"
		return identity, scope.Defined[identity]
	}
	identifier, is_identifier := core.(*ast.Ident)
	if !is_identifier {
		return "", false
	}
	if scope.Type_Parameters[identifier.Name] {
		return "", false
	}
	if identifier.Name == "string" {
		// A raw string subject is banned by check_primitive_types, not asserted here.
		return "", false
	}
	preset := struct_primitive_preset(identifier.Name)
	if preset != "" {
		return scope.Default_Package + "\x00" + preset, true
	}
	if struct_is_builtin(identifier.Name) {
		return "", false
	}
	identity := scope.Current_Package + "\x00" + source.Invariant_Name(identifier.Name)
	return identity, scope.Defined[identity]
}

// Splits a body into the leading helper block and the first-statement defer's
// body (when the first statement is a defer of a func literal).
func function_lead_and_defer(
	body []ast.Stmt,
) (lead []ast.Stmt, defer_body []ast.Stmt, has_defer bool) {

	start := 0
	if len(body) > 0 {
		literal := function_defer_literal(body[0])
		if literal != nil {
			defer_body = literal.Body.List
			has_defer = true
			start = 1
		}
	}
	for _, statement := range body[start:] {
		if !function_is_helper_statement(statement) {
			break
		}
		lead = append(lead, statement)
	}
	return lead, defer_body, has_defer
}

// Returns the func literal of a first-statement `defer func(){…}()`, or nil.
func function_defer_literal(statement ast.Stmt) (literal *ast.FuncLit) {
	defer_statement, is_defer := statement.(*ast.DeferStmt)
	if !is_defer {
		return nil
	}
	function_literal, is_literal := defer_statement.Call.Fun.(*ast.FuncLit)
	if !is_literal {
		return nil
	}
	return function_literal
}

// Reports whether a statement is a helper call, or a range
// loop whose body is only _Invariants calls.
func function_is_helper_statement(statement ast.Stmt) (yes bool) {
	if function_is_invariant_call(statement) {
		return true
	}
	range_statement, is_range := statement.(*ast.RangeStmt)
	if !is_range {
		return false
	}
	if len(range_statement.Body.List) == 0 {
		return false
	}
	for _, inner := range range_statement.Body.List {
		if !function_is_invariant_call(inner) {
			return false
		}
	}
	return true
}

// Reports whether a statement is a bare `X_Invariants(...)` call.
func function_is_invariant_call(statement ast.Stmt) (yes bool) {
	call := statement_call(statement)
	if call == nil {
		return false
	}
	return type_invariants_is_bundle_name(struct_callee_name(call.Fun))
}

// Restricting helper recognition to a call that is itself the statement prevents a nested or
// dead-code call from satisfying a topology mandate.
func statement_call(statement ast.Stmt) (call *ast.CallExpr) {
	expression_statement, is_expression := statement.(*ast.ExprStmt)
	if !is_expression {
		return nil
	}
	call, _ = expression_statement.X.(*ast.CallExpr)
	return call
}

// Reports whether the statements satisfy one requirement.
func function_requirement_met(
	statements []ast.Stmt, requirement Helper_Requirement, scope *Invariant_Scope,
) (met bool) {
	return function_calls_helper(statements, requirement, scope)
}

// Reports whether some statement calls the exact expected helper on the subject.
func function_calls_helper(
	statements []ast.Stmt, requirement Helper_Requirement, scope *Invariant_Scope,
) (met bool) {

	shadowed := function_shadow_copy(scope.Shadowed)
	for _, statement := range statements {
		call := statement_call(statement)
		if call != nil {
			identity := helper_callee_identity(
				call.Fun, scope.Current_Package, scope.Imports, shadowed)
			if identity == requirement.Expected {
				if function_first_argument_name(call) == requirement.Subject {
					return true
				}
			}
		}
		function_statement_shadows(statement, shadowed)
	}
	return false
}

// Returns a call's first-argument identifier name, dereferencing a leading
// pointer; "" when the first argument is not an identifier.
func function_first_argument_name(call *ast.CallExpr) (name string) {
	if len(call.Args) == 0 {
		return ""
	}
	argument := call.Args[0]
	star, is_star := argument.(*ast.StarExpr)
	if is_star {
		argument = star.X
	}
	identifier, is_identifier := argument.(*ast.Ident)
	if !is_identifier {
		return ""
	}
	return identifier.Name
}

// Renders the expected helper form for a diagnostic without exposing the identity separator.
func function_form(requirement Helper_Requirement) (form string) {
	return helper_identity_name(requirement.Expected) + "(" + requirement.Subject + ", ...)"
}

// Flags every non-exempt, non-main package whose test binary fails to wire the
// invariant coverage recorder. invariant.Run_Test_Main is the canonical TestMain
// body; without it a package's mandated _Invariants bundles run but their
// Sometimes axes and Always reachability are never verified — the discipline
// silently evaporates. Package-level because the TestMain may live in any of the
// directory's test files, so the whole directory is judged together. Shares the
// type-invariant rule's opt-out.
func check_recorder_test_main(
	parsed_files []Parsed_File, components *Component_Index, exempt []string,
) (diags []Diagnostic) {
	for _, group := range recorder_test_main_groups(parsed_files) {
		if source.Path_Matches_Glob(group.Directory, exempt) {
			continue
		}
		// A main package holds the binary's wiring, not testable invariant logic.
		if group.Is_Main {
			continue
		}
		// A binary component's invariants are witnessed through its simulation
		// package driving internal.Main, so only a shared library still registers
		// its own recorder here; a binary internal package is covered there instead.
		component_index_number := components.File_To_Component[group.Any_Path]
		if component_index_number < 0 {
			continue
		}
		if !components.Components[component_index_number].Is_Shared_Library {
			continue
		}
		diags = append(diags, recorder_group_diagnostics(group)...)
	}
	return diags
}

// One directory's recorder-relevant facts, gathered across all its files.
type Recorder_Group struct {
	// Directory is the package directory these facts describe.
	Directory string
	// Any_Path is one source path in the directory, for diagnostics.
	Any_Path string
	// Is_Main records whether the package is package main.
	Is_Main bool
	// Has_Test records whether the directory has a test file.
	Has_Test bool
	// Source_Anchor is a position in the package's non-test source.
	Source_Anchor token.Position
	// Test_Anchor is a position in the package's test source.
	Test_Anchor token.Position
	// Test_Main_Found records whether a TestMain was found.
	Test_Main_Found bool
	// Test_Main_Canonical records whether the TestMain matches the required form.
	Test_Main_Canonical bool
	// Test_Main_Position is the TestMain's source position.
	Test_Main_Position token.Position
}

// Buckets parsed files by directory, preserving first-seen order (parsed_files is
// path-sorted) so diagnostics are deterministic without a separate sort.
func recorder_test_main_groups(parsed_files []Parsed_File) (groups []*Recorder_Group) {
	index := map[string]*Recorder_Group{}
	for _, pf := range parsed_files {
		directory := path.Dir(pf.Path)
		group := index[directory]
		if group == nil {
			group = &Recorder_Group{Directory: directory}
			index[directory] = group
			groups = append(groups, group)
		}
		recorder_group_absorb(group, pf)
	}
	return groups
}

// Folds one file's facts into its directory group: a test file may carry the
// TestMain and is the preferred anchor; a source file marks the main package.
func recorder_group_absorb(group *Recorder_Group, file Parsed_File) {
	if group.Any_Path == "" {
		group.Any_Path = file.Path
	}
	position := file.File_Set.Position(file.File.Name.Pos())
	if strings.HasSuffix(file.Path, "_test.go") {
		group.Has_Test = true
		if group.Test_Anchor.Line == 0 {
			group.Test_Anchor = position
		}
		recorder_group_scan_test_main(group, file)
		return
	}
	if file.File.Name.Name == "main" {
		group.Is_Main = true
	}
	if group.Source_Anchor.Line == 0 {
		group.Source_Anchor = position
	}
}

// Records the directory's first real TestMain — name TestMain with a *testing.M
// parameter — and whether it is the exact canonical shape.
func recorder_group_scan_test_main(group *Recorder_Group, file Parsed_File) {
	if group.Test_Main_Found {
		return
	}
	for _, declaration := range file.File.Decls {
		function, is_function := declaration.(*ast.FuncDecl)
		if !is_function {
			continue
		}
		if function.Name.Name != "TestMain" {
			continue
		}
		parameter := recorder_test_main_parameter(function)
		if parameter == "" {
			continue
		}
		group.Test_Main_Found = true
		group.Test_Main_Position = file.File_Set.Position(function.Name.Pos())
		group.Test_Main_Canonical = recorder_test_main_canonical(function, parameter)
		return
	}
}

// Returns the name of TestMain's *testing.M parameter, or "" when it has none —
// a TestMain without one is not the suite entry point.
func recorder_test_main_parameter(function *ast.FuncDecl) (name string) {
	if function.Type.Params == nil {
		return ""
	}
	for _, field := range function.Type.Params.List {
		star, is_star := field.Type.(*ast.StarExpr)
		if !is_star {
			continue
		}
		selector, is_selector := star.X.(*ast.SelectorExpr)
		if !is_selector {
			continue
		}
		qualifier, is_qualifier := selector.X.(*ast.Ident)
		if !is_qualifier {
			continue
		}
		if qualifier.Name != "testing" {
			continue
		}
		if selector.Sel.Name != "M" {
			continue
		}
		if len(field.Names) == 0 {
			continue
		}
		return field.Names[0].Name
	}
	return ""
}

// Reports whether the function is the one allowed shape exactly:
// func TestMain(m *testing.M) { invariant.Run_Test_Main(m) } — its parameter
// named m and its body that sole call, nothing more.
func recorder_test_main_canonical(
	function *ast.FuncDecl, parameter string,
) (canonical bool) {

	if parameter != "m" {
		return false
	}
	if function.Body == nil {
		return false
	}
	if len(function.Body.List) != 1 {
		return false
	}
	expression, is_expression := function.Body.List[0].(*ast.ExprStmt)
	if !is_expression {
		return false
	}
	call, is_call := expression.X.(*ast.CallExpr)
	if !is_call {
		return false
	}
	return recorder_canonical_call(call)
}

// Reports whether the call is exactly invariant.Run_Test_Main(m).
func recorder_canonical_call(call *ast.CallExpr) (canonical bool) {
	selector, is_selector := call.Fun.(*ast.SelectorExpr)
	if !is_selector {
		return false
	}
	qualifier, is_qualifier := selector.X.(*ast.Ident)
	if !is_qualifier {
		return false
	}
	if qualifier.Name != "invariant" {
		return false
	}
	if selector.Sel.Name != "Run_Test_Main" {
		return false
	}
	if len(call.Args) != 1 {
		return false
	}
	argument, is_argument := call.Args[0].(*ast.Ident)
	if !is_argument {
		return false
	}
	return argument.Name == "m"
}

// Emits the one diagnostic a non-exempt package's wiring gap warrants: a missing
// TestMain, or a TestMain that is not the one allowed shape.
func recorder_group_diagnostics(group *Recorder_Group) (diags []Diagnostic) {
	if !group.Test_Main_Found {
		anchor := group.Source_Anchor
		if group.Has_Test {
			anchor = group.Test_Anchor
		}
		return []Diagnostic{{
			Position: anchor,
			Message: group.Directory +
				" must wire invariant.Run_Test_Main in a TestMain",
		}}
	}
	if !group.Test_Main_Canonical {
		return []Diagnostic{{
			Position: group.Test_Main_Position,
			Message:  "TestMain must be exactly: invariant.Run_Test_Main(m)",
		}}
	}
	return nil
}

// SIMULATION_DIRECTORY names the test-only package under a binary component's
// internal/ whose fuzz test drives internal.Main.
const SIMULATION_DIRECTORY = "simulation_test"

// SIMULATION_GLOB is the sole TestMain directory argument: the internal package and
// every package beneath it, registered in one pattern since the recorder recurses on **.
const SIMULATION_GLOB = "../**"

// A binary component's invariants are witnessed only by a simulation package that
// drives internal.Main through a fuzz test, never by a per-package Run_Test_Main.
// This holds that package to its contract: it exists, declares nothing but the
// fuzz driver and its TestMain, and registers every non-exempt internal package
// for coverage. Every diagnostic is tier two, so a tier-one issue suppresses it.
func check_simulation(
	parsed_files []Parsed_File, components *Component_Index, exempt []string,
) (diags []Diagnostic) {
	for i := range components.Components {
		if components.Components[i].Is_Shared_Library {
			continue
		}
		diags = append(diags,
			simulation_component_diagnostics(parsed_files, components, i, exempt)...)
	}
	for i := range diags {
		diags[i].Tier = 2
	}
	return diags
}

// The simulation diagnostics for one binary component: none when its internal tree
// carries no non-exempt package (nothing to witness), else the presence, contents,
// and TestMain checks against the package at internal/simulation_test.
func simulation_component_diagnostics(
	parsed_files []Parsed_File, components *Component_Index,
	component_index_number int, exempt []string,
) (diags []Diagnostic) {
	component := components.Components[component_index_number]
	internal_root := component.Root + "/internal"
	internal_dirs := simulation_internal_dirs(
		parsed_files, components, component_index_number, exempt, internal_root)
	if len(internal_dirs) == 0 {
		return nil
	}
	position := token.Position{Filename: internal_root, Line: 1, Column: 1}
	sim_directory := internal_root + "/" + SIMULATION_DIRECTORY
	sim_files := simulation_package_files(parsed_files, sim_directory)
	if len(sim_files) == 0 {
		return simulation_diagnostic(position, "binary component "+
			strconv.Quote(component.Import_Path)+" must declare an internal/"+
			SIMULATION_DIRECTORY+" package driving internal.Main")
	}
	diags = append(diags, simulation_package_diagnostics(sim_files, position)...)
	diags = append(diags, simulation_contents_diagnostics(sim_files, position)...)
	internal_functions := simulation_internal_functions(
		parsed_files, components, component_index_number, internal_root)
	diags = append(diags, simulation_entry_diagnostics(
		sim_files, internal_functions, component.Import_Path+"/internal", position)...)
	return append(diags, simulation_test_main_diagnostics(sim_files, position)...)
}

// The non-exempt internal package directories of the component, sorted, excluding
// the simulation package itself — the packages the TestMain must register so their
// invariants seed and judge in the simulation's isolated test binary.
func simulation_internal_dirs(
	parsed_files []Parsed_File, components *Component_Index,
	component_index_number int, exempt []string, internal_root string,
) (dirs []string) {
	sim_directory := internal_root + "/" + SIMULATION_DIRECTORY
	seen := map[string]bool{}
	for _, pf := range parsed_files {
		if strings.HasSuffix(pf.Path, "_test.go") {
			continue
		}
		if components.File_To_Component[pf.Path] != component_index_number {
			continue
		}
		directory := path.Dir(pf.Path)
		under_internal := directory == internal_root
		if strings.HasPrefix(directory, internal_root+"/") {
			under_internal = true
		}
		if !under_internal {
			continue
		}
		if directory == sim_directory {
			continue
		}
		if strings.HasPrefix(directory, sim_directory+"/") {
			continue
		}
		if source.Path_Matches_Glob(directory, exempt) {
			continue
		}
		if seen[directory] {
			continue
		}
		seen[directory] = true
		dirs = append(dirs, directory)
	}
	sort.Strings(dirs)
	return dirs
}

// The parsed test files that make up the simulation package at sim_directory.
func simulation_package_files(
	parsed_files []Parsed_File, sim_directory string,
) (files []Parsed_File) {
	for _, pf := range parsed_files {
		if path.Dir(pf.Path) != sim_directory {
			continue
		}
		files = append(files, pf)
	}
	return files
}

// The simulation directory holds one blackbox test package and no source package:
// every file is a _test.go whose clause ends in _test. A source file would compile
// into the same tree the fuzz drives, a back door around the isolated test binary.
func simulation_package_diagnostics(
	sim_files []Parsed_File, position token.Position,
) (diags []Diagnostic) {
	for _, pf := range sim_files {
		if !strings.HasSuffix(pf.Path, "_test.go") {
			return simulation_diagnostic(position,
				"simulation holds only a blackbox test package; no source file")
		}
		if !strings.HasSuffix(pf.File.Name.Name, "_test") {
			return simulation_diagnostic(position,
				"simulation package must be blackbox: package <name>_test")
		}
	}
	return nil
}

// The simulation package declares a fuzz function that drives internal.Main — the
// witness for the component's invariants. Any other declaration is allowed; the fuzz
// driver just has to be present.
func simulation_contents_diagnostics(
	files []Parsed_File, position token.Position,
) (diags []Diagnostic) {
	for _, pf := range files {
		for _, declaration := range pf.File.Decls {
			if simulation_is_fuzz(declaration) {
				return nil
			}
		}
	}
	return simulation_diagnostic(position,
		"simulation package must declare a fuzz function driving internal.Main")
}

// Reports whether the declaration is a free Fuzz function taking a *testing.F.
func simulation_is_fuzz(declaration ast.Decl) (fuzz bool) {
	function, is_function := declaration.(*ast.FuncDecl)
	if !is_function {
		return false
	}
	if function.Recv != nil {
		return false
	}
	if !strings.HasPrefix(function.Name.Name, "Fuzz") {
		return false
	}
	return simulation_fuzz_parameter(function) != ""
}

// The exported free-function names, Main aside, declared across the component's
// internal tree — the functions the simulation is forbidden to reference, since
// each is a second entry point that could witness an invariant without driving Main.
func simulation_internal_functions(
	parsed_files []Parsed_File, components *Component_Index,
	component_index_number int, internal_root string,
) (functions map[string]bool) {
	functions = map[string]bool{}
	for _, pf := range parsed_files {
		if strings.HasSuffix(pf.Path, "_test.go") {
			continue
		}
		if components.File_To_Component[pf.Path] != component_index_number {
			continue
		}
		directory := path.Dir(pf.Path)
		under := directory == internal_root
		if strings.HasPrefix(directory, internal_root+"/") {
			under = true
		}
		if !under {
			continue
		}
		for _, declaration := range pf.File.Decls {
			function, is_function := declaration.(*ast.FuncDecl)
			if !is_function {
				continue
			}
			if function.Recv != nil {
				continue
			}
			if function.Name.Name == "Main" {
				continue
			}
			if !token.IsExported(function.Name.Name) {
				continue
			}
			functions[function.Name.Name] = true
		}
	}
	return functions
}

// The local names the simulation file binds to internal-subtree imports, so a
// selector on one of them can be checked against the forbidden-function set.
func simulation_internal_import_locals(
	file *ast.File, internal_import_path string,
) (locals map[string]bool) {
	locals = map[string]bool{}
	for _, specification := range file.Imports {
		import_path := strings.Trim(specification.Path.Value, "\"")
		under := import_path == internal_import_path
		if strings.HasPrefix(import_path, internal_import_path+"/") {
			under = true
		}
		if !under {
			continue
		}
		locals[source.Import_Local_Name(specification, import_path)] = true
	}
	return locals
}

// The simulation may reference only Main among the internal tree's functions: any
// other exported internal function it names is a second entry point that could
// fabricate a witness without driving Main. Types, constants, and vars stay free.
func simulation_entry_diagnostics(
	sim_files []Parsed_File, internal_functions map[string]bool,
	internal_import_path string, position token.Position,
) (diags []Diagnostic) {
	for _, pf := range sim_files {
		locals := simulation_internal_import_locals(pf.File, internal_import_path)
		ast.Inspect(pf.File, func(node ast.Node) (descend bool) {
			selector, is_selector := node.(*ast.SelectorExpr)
			if !is_selector {
				return true
			}
			identifier, is_identifier := selector.X.(*ast.Ident)
			if !is_identifier {
				return true
			}
			if !locals[identifier.Name] {
				return true
			}
			if !internal_functions[selector.Sel.Name] {
				return true
			}
			diags = append(diags, simulation_diagnostic(position,
				"simulation may reference only Main; got "+
					identifier.Name+"."+selector.Sel.Name)...)
			return true
		})
	}
	return diags
}

// Returns the name of the function's *testing.F parameter, or "" when it has none —
// a Fuzz function without one is not a real fuzz target.
func simulation_fuzz_parameter(function *ast.FuncDecl) (name string) {
	if function.Type.Params == nil {
		return ""
	}
	for _, field := range function.Type.Params.List {
		star, is_star := field.Type.(*ast.StarExpr)
		if !is_star {
			continue
		}
		selector, is_selector := star.X.(*ast.SelectorExpr)
		if !is_selector {
			continue
		}
		qualifier, is_qualifier := selector.X.(*ast.Ident)
		if !is_qualifier {
			continue
		}
		if qualifier.Name != "testing" {
			continue
		}
		if selector.Sel.Name != "F" {
			continue
		}
		if len(field.Names) == 0 {
			continue
		}
		return field.Names[0].Name
	}
	return ""
}

// The simulation's TestMain body must be exactly invariant.Run_Test_Main(m, "../**"):
// that one glob registers the internal package and every package beneath it, so the
// isolated simulation binary seeds and judges them all without enumerating each.
func simulation_test_main_diagnostics(
	files []Parsed_File, position token.Position,
) (diags []Diagnostic) {
	function := simulation_find_test_main(files)
	if function == nil {
		return simulation_diagnostic(position,
			"simulation package must wire invariant.Run_Test_Main in a TestMain")
	}
	if simulation_test_main_canonical(function) {
		return nil
	}
	return simulation_diagnostic(position,
		"simulation TestMain must be exactly: invariant.Run_Test_Main(m, "+
			strconv.Quote(SIMULATION_GLOB)+")")
}

// Reports whether the TestMain body is exactly invariant.Run_Test_Main(m, "../**").
func simulation_test_main_canonical(function *ast.FuncDecl) (canonical bool) {
	directories, ok := simulation_test_main_directories(function)
	if !ok {
		return false
	}
	if len(directories) != 1 {
		return false
	}
	return directories[0] == SIMULATION_GLOB
}

// The first TestMain with a *testing.M parameter among the simulation files.
func simulation_find_test_main(files []Parsed_File) (function *ast.FuncDecl) {
	for _, pf := range files {
		for _, declaration := range pf.File.Decls {
			candidate, is_function := declaration.(*ast.FuncDecl)
			if !is_function {
				continue
			}
			if candidate.Name.Name != "TestMain" {
				continue
			}
			if recorder_test_main_parameter(candidate) == "" {
				continue
			}
			return candidate
		}
	}
	return nil
}

// The directory arguments of the simulation TestMain's sole statement, and whether
// that statement is exactly invariant.Run_Test_Main(m, <one or more string literals>).
func simulation_test_main_directories(
	function *ast.FuncDecl,
) (directories []string, canonical bool) {
	if recorder_test_main_parameter(function) != "m" {
		return nil, false
	}
	if function.Body == nil {
		return nil, false
	}
	if len(function.Body.List) != 1 {
		return nil, false
	}
	expression, is_expression := function.Body.List[0].(*ast.ExprStmt)
	if !is_expression {
		return nil, false
	}
	call, is_call := expression.X.(*ast.CallExpr)
	if !is_call {
		return nil, false
	}
	return simulation_call_directories(call)
}

// The string-literal directory arguments after m, and whether the call is exactly
// invariant.Run_Test_Main(m, <one or more string literals>).
func simulation_call_directories(call *ast.CallExpr) (directories []string, canonical bool) {
	selector, is_selector := call.Fun.(*ast.SelectorExpr)
	if !is_selector {
		return nil, false
	}
	qualifier, is_qualifier := selector.X.(*ast.Ident)
	if !is_qualifier {
		return nil, false
	}
	if qualifier.Name != "invariant" {
		return nil, false
	}
	if selector.Sel.Name != "Run_Test_Main" {
		return nil, false
	}
	if len(call.Args) < 2 {
		return nil, false
	}
	first, is_ident := call.Args[0].(*ast.Ident)
	if !is_ident {
		return nil, false
	}
	if first.Name != "m" {
		return nil, false
	}
	for _, argument := range call.Args[1:] {
		literal, ok := simulation_string_literal(argument)
		if !ok {
			return nil, false
		}
		directories = append(directories, literal)
	}
	return directories, true
}

// The unquoted value of a string-literal expression, or ok false when it is not one.
func simulation_string_literal(expression ast.Expr) (value string, ok bool) {
	literal, is_literal := expression.(*ast.BasicLit)
	if !is_literal {
		return "", false
	}
	if literal.Kind != token.STRING {
		return "", false
	}
	unquoted, unquote_error := strconv.Unquote(literal.Value)
	if unquote_error != nil {
		return "", false
	}
	return unquoted, true
}

// One simulation diagnostic at position with the given message.
func simulation_diagnostic(position token.Position, message string) (diags []Diagnostic) {
	return []Diagnostic{{
		Position: position,
		Name:     "simulation",
		Message:  message,
	}}
}

// Flags a raw string, slice, or map used as a function/method parameter or result,
// or as a struct field. Such a type has no preset and cannot carry its own bundle;
// a defined wrapper gives it well-defined coverage. A method satisfying a stdlib
// interface keeps its dictated signature. Shares the type-invariant opt-out.
func check_primitive_types(parsed_files []Parsed_File, exempt []string) (diags []Diagnostic) {
	for _, pf := range parsed_files {
		if strings.HasSuffix(pf.Path, "_test.go") {
			continue
		}
		if source.Path_Matches_Glob(pf.Path, exempt) {
			continue
		}
		diags = append(diags, primitive_file_diagnostics(pf)...)
	}
	return diags
}

// Checks every function signature and struct field in one file.
func primitive_file_diagnostics(file Parsed_File) (diags []Diagnostic) {
	for _, declaration := range file.File.Decls {
		switch typed := declaration.(type) {
		case *ast.FuncDecl:
			diags = append(diags, primitive_function_diagnostics(file, typed)...)
		case *ast.GenDecl:
			diags = append(diags, primitive_struct_diagnostics(file, typed)...)
		}
	}
	return diags
}

// Flags a non-stdlib function's raw string/slice/map parameters and results.
func primitive_function_diagnostics(
	file Parsed_File, function *ast.FuncDecl,
) (diags []Diagnostic) {

	if source.Method_Satisfies_Stdlib(function) {
		return nil
	}
	position := file.File_Set.Position(function.Name.Pos())
	parameter_gaps := primitive_field_gaps(function.Type.Params, "parameter")
	diags = append(diags,
		primitive_owner_diagnostics(parameter_gaps, function.Name.Name, position)...)
	result_gaps := primitive_field_gaps(function.Type.Results, "result")
	diags = append(diags,
		primitive_owner_diagnostics(result_gaps, function.Name.Name, position)...)
	return diags
}

// Flags each struct type's raw string/slice/map fields.
func primitive_struct_diagnostics(file Parsed_File, general *ast.GenDecl) (diags []Diagnostic) {
	if general.Tok != token.TYPE {
		return nil
	}
	for _, specification := range general.Specs {
		type_specification, is_type := specification.(*ast.TypeSpec)
		if !is_type {
			continue
		}
		struct_type, is_struct := type_specification.Type.(*ast.StructType)
		if !is_struct {
			continue
		}
		gaps := primitive_field_gaps(struct_type.Fields, "field")
		position := file.File_Set.Position(type_specification.Name.Pos())
		diags = append(diags,
			primitive_owner_diagnostics(
				gaps, type_specification.Name.Name, position)...)
	}
	return diags
}

// Separating raw-field discovery from declaration ownership keeps role, owner, and position out
// of an argument-bundling struct.
func primitive_field_gaps(fields *ast.FieldList, role string) (gaps []string) {
	if fields == nil {
		return nil
	}
	for _, field := range fields.List {
		kind := numeric_raw_primitive_kind(field.Type)
		if kind == "" {
			continue
		}
		for _, identifier := range primitive_field_names(field) {
			gaps = append(gaps, "raw "+kind+" "+role+" "+identifier+
				"; wrap it in a defined type")
		}
	}
	return gaps
}

// Adds the declaration identity and position only after the field scan, keeping scan inputs
// direct and diagnostics uniform between function and struct subjects.
func primitive_owner_diagnostics(
	gaps []string, owner string, position token.Position,
) (diags []Diagnostic) {
	for _, gap := range gaps {
		diags = append(diags, Diagnostic{
			Position: position,
			Message:  owner + ": " + gap,
		})
	}
	return diags
}

// Returns a field's declared names, or one empty name for an anonymous field so it
// still yields a diagnostic.
func primitive_field_names(field *ast.Field) (names []string) {
	if len(field.Names) == 0 {
		return []string{""}
	}
	for _, name := range field.Names {
		names = append(names, name.Name)
	}
	return names
}

// Returns "string", "slice", or "map" when the type is a raw primitive of that kind
// — a leading * unwrapped, a variadic counted as a slice — or "" otherwise. A fixed
// [N]T array is not a slice; a defined type that wraps a primitive is not raw.
func numeric_raw_primitive_kind(expression ast.Expr) (kind string) {
	core := expression
	if star, is_star := core.(*ast.StarExpr); is_star {
		core = star.X
	}
	if _, is_ellipsis := core.(*ast.Ellipsis); is_ellipsis {
		return "slice"
	}
	switch typed := core.(type) {
	case *ast.Ident:
		if typed.Name == "string" {
			return "string"
		}
		return ""
	case *ast.ArrayType:
		if typed.Len != nil {
			return ""
		}
		return "slice"
	case *ast.MapType:
		return "map"
	default:
		return ""
	}
}

// Flags every in-scope type whose next declaration
// is not its correctly-named, correctly-signed bundle function.
func check_type_invariants_forward(
	file_set *token.FileSet, file *ast.File, invariant_names map[string]bool,
) (diags []Diagnostic) {

	for index, declaration := range file.Decls {
		general, is_general := declaration.(*ast.GenDecl)
		if !is_general {
			continue
		}
		if general.Tok != token.TYPE {
			continue
		}
		// Grouped Declarations bans type (...) groups, so a type holds one spec.
		type_specification, is_type := general.Specs[0].(*ast.TypeSpec)
		if !is_type {
			continue
		}
		if !type_invariant_required(type_specification) {
			continue
		}
		diags = append(diags, check_type_invariants_one(
			file_set, file, index, type_specification, invariant_names)...)
	}
	return diags
}

// Judges the in-scope type at file.Decls[index]:
// presence and casing of the following bundle, then its signature and the gap.
func check_type_invariants_one(
	file_set *token.FileSet, file *ast.File, index int,
	type_specification *ast.TypeSpec, invariant_names map[string]bool,
) (diags []Diagnostic) {

	want := source.Invariant_Name(type_specification.Name.Name)
	bundle := type_invariants_following_function(file, index)
	if bundle == nil {
		return append(diags, type_invariants_absent(file_set, type_specification, want))
	}
	if bundle.Name.Name != want {
		return append(diags, type_invariants_absent(file_set, type_specification, want))
	}
	if !type_invariants_signature_ok(bundle, type_specification, invariant_names) {
		diags = append(diags,
			type_invariants_bad_signature(file_set, bundle, type_specification))
	}
	diags = append(diags,
		check_type_invariants_gap(file_set, file, type_specification, bundle)...)
	return diags
}

// Builds the diagnostic for a type with no bundle below it.
func type_invariants_absent(
	file_set *token.FileSet, type_specification *ast.TypeSpec, want string,
) (diag Diagnostic) {

	type_name := type_specification.Name.Name
	return Diagnostic{
		Position: file_set.Position(type_specification.Name.Pos()),
		Message: "declare " + want + "(" + type_name +
			", invariant.Namespace) directly below " + type_name,
	}
}

// Builds the diagnostic for a bundle whose parameters
// are not the type, by value or pointer, first and an invariant.Namespace last.
func type_invariants_bad_signature(
	file_set *token.FileSet, function *ast.FuncDecl, type_specification *ast.TypeSpec,
) (diag Diagnostic) {

	type_name := type_specification.Name.Name
	return Diagnostic{
		Position: file_set.Position(function.Name.Pos()),
		Message: function.Name.Name + " must take (" + type_name + " or *" +
			type_name + ", invariant.Namespace)",
	}
}

// Flags a bundle-named function adrift from its
// type: one whose immediately preceding declaration is not the type it names.
func check_type_invariants_orphan(
	file_set *token.FileSet, file *ast.File,
) (diags []Diagnostic) {

	for index, declaration := range file.Decls {
		function, is_function := declaration.(*ast.FuncDecl)
		if !is_function {
			continue
		}
		if function.Recv != nil {
			continue
		}
		if !type_invariants_is_bundle_name(function.Name.Name) {
			continue
		}
		if type_invariants_preceding_type(file, index, function.Name.Name) {
			continue
		}
		diags = append(diags, Diagnostic{
			Position: file_set.Position(function.Name.Pos()),
			Message:  function.Name.Name + " must be declared directly below its type",
		})
	}
	return diags
}

// Flags a comment between a type and its bundle that is
// not the bundle's own doc comment, so only blank lines and that doc may separate
// them.
func check_type_invariants_gap(
	file_set *token.FileSet, file *ast.File,
	type_specification *ast.TypeSpec, function *ast.FuncDecl,
) (diags []Diagnostic) {

	for _, group := range file.Comments {
		if group == function.Doc {
			continue
		}
		if group.Pos() <= type_specification.End() {
			continue
		}
		if group.End() >= function.Pos() {
			continue
		}
		diags = append(diags, Diagnostic{
			Position: file_set.Position(group.Pos()),
			Message: "remove the comment between " + type_specification.Name.Name +
				" and " + function.Name.Name,
		})
	}
	return diags
}

// Reports whether a type declaration must carry a bundle.
// Aliases, function and interface types, and empty structs state no properties
// worth a bundle; every other defined type is in scope.
func type_invariant_required(type_specification *ast.TypeSpec) (required bool) {
	if type_specification.Assign.IsValid() {
		return false
	}
	switch base := type_specification.Type.(type) {
	case *ast.FuncType:
		return false
	case *ast.InterfaceType:
		return false
	case *ast.StructType:
		return len(base.Fields.List) > 0
	default:
		return true
	}
}

// Returns the function declared immediately
// below the declaration at index, or nil when the next declaration is not one.
func type_invariants_following_function(
	file *ast.File, index int,
) (function *ast.FuncDecl) {

	if index+1 >= len(file.Decls) {
		return nil
	}
	next, is_function := file.Decls[index+1].(*ast.FuncDecl)
	if !is_function {
		return nil
	}
	return next
}

// Reports whether the declaration before index is
// the type whose bundle name is function_name.
func type_invariants_preceding_type(
	file *ast.File, index int, function_name string,
) (yes bool) {

	if index == 0 {
		return false
	}
	general, is_general := file.Decls[index-1].(*ast.GenDecl)
	if !is_general {
		return false
	}
	if general.Tok != token.TYPE {
		return false
	}
	type_specification, is_type := general.Specs[0].(*ast.TypeSpec)
	if !is_type {
		return false
	}
	return source.Invariant_Name(type_specification.Name.Name) == function_name
}

// Reports whether name ends in the bundle suffix.
func type_invariants_is_bundle_name(name string) (yes bool) {
	if strings.HasSuffix(name, "_Invariants") {
		return true
	}
	return strings.HasSuffix(name, "_invariants")
}

// Reports whether the bundle takes its type, by
// value or pointer, first and an invariant.Namespace last.
func type_invariants_signature_ok(
	function *ast.FuncDecl, type_specification *ast.TypeSpec,
	invariant_names map[string]bool,
) (ok bool) {

	if function.Type.Params == nil {
		return false
	}
	list := function.Type.Params.List
	if len(list) < 2 {
		return false
	}
	if !type_invariants_first_is_type(list[0].Type, type_specification) {
		return false
	}
	return type_invariants_last_is_namespace(list[len(list)-1].Type, invariant_names)
}

// Reports whether expression is the bundle's type,
// dereferencing a leading pointer and, for a generic type, requiring it be
// instantiated over its own parameters in order.
func type_invariants_first_is_type(
	expression ast.Expr, type_specification *ast.TypeSpec,
) (ok bool) {

	star, is_star := expression.(*ast.StarExpr)
	if is_star {
		expression = star.X
	}
	if type_specification.TypeParams == nil {
		identifier, is_identifier := expression.(*ast.Ident)
		if !is_identifier {
			return false
		}
		return identifier.Name == type_specification.Name.Name
	}
	return type_invariants_generic_matches(expression, type_specification)
}

// Reports whether expression is the generic type
// instantiated over its declared parameters in order: Box[T] for type Box[T any].
func type_invariants_generic_matches(
	expression ast.Expr, type_specification *ast.TypeSpec,
) (ok bool) {

	base, arguments := type_invariants_instantiation(expression)
	if base == nil {
		return false
	}
	if base.Name != type_specification.Name.Name {
		return false
	}
	want := type_invariants_field_names(type_specification.TypeParams)
	got := type_invariants_argument_names(arguments)
	return slices.Equal(want, got)
}

// Splits a generic instantiation into its base
// identifier and type arguments, handling the one- and many-argument AST forms.
func type_invariants_instantiation(
	expression ast.Expr,
) (base *ast.Ident, arguments []ast.Expr) {

	switch node := expression.(type) {
	case *ast.IndexExpr:
		identifier, is_identifier := node.X.(*ast.Ident)
		if !is_identifier {
			return nil, nil
		}
		return identifier, []ast.Expr{node.Index}
	case *ast.IndexListExpr:
		identifier, is_identifier := node.X.(*ast.Ident)
		if !is_identifier {
			return nil, nil
		}
		return identifier, node.Indices
	default:
		return nil, nil
	}
}

// Flattens a field list to its declared names.
func type_invariants_field_names(fields *ast.FieldList) (names []string) {
	for _, field := range fields.List {
		for _, name := range field.Names {
			names = append(names, name.Name)
		}
	}
	return names
}

// Returns the identifier names of type arguments,
// or nil when any argument is not a bare identifier so a mismatch is reported.
func type_invariants_argument_names(arguments []ast.Expr) (names []string) {
	for _, argument := range arguments {
		identifier, is_identifier := argument.(*ast.Ident)
		if !is_identifier {
			return nil
		}
		names = append(names, identifier.Name)
	}
	return names
}

// Reports whether expression is the selector
// <pkg>.Namespace for a local name bound to the invariant package.
func type_invariants_last_is_namespace(
	expression ast.Expr, invariant_names map[string]bool,
) (ok bool) {

	selector, is_selector := expression.(*ast.SelectorExpr)
	if !is_selector {
		return false
	}
	if selector.Sel.Name != "Namespace" {
		return false
	}
	qualifier, is_identifier := selector.X.(*ast.Ident)
	if !is_identifier {
		return false
	}
	return invariant_names[qualifier.Name]
}

// Returns the local names the invariant package is
// bound to in this file — its own package name, or an explicit import alias — so
// the namespace parameter is recognized however the package was imported.
func type_invariants_import_names(file *ast.File) (names map[string]bool) {
	names = map[string]bool{}
	for _, specification := range file.Imports {
		unquoted, unquote_err := strconv.Unquote(specification.Path.Value)
		if unquote_err != nil {
			continue
		}
		if !type_invariants_path_is_invariant(unquoted) {
			continue
		}
		if specification.Name != nil {
			names[specification.Name.Name] = true
			continue
		}
		names[path.Base(unquoted)] = true
	}
	return names
}

// Reports whether an import path has a segment
// named invariant — the framework package, wherever it sits in the module.
func type_invariants_path_is_invariant(import_path string) (yes bool) {
	for _, segment := range strings.Split(import_path, "/") {
		if segment == "invariant" {
			return true
		}
	}
	return false
}
