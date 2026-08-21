// Package assertion enforces the _Invariants doctrine — every in-scope type
// states its properties in a companion helper beside it, and every named
// function calls the helpers for its typed inputs and outputs — and the sibling simulation
// doctrine that witnesses those invariants. The caller hands over the
// already-parsed files and the component graph, so this package reads and parses
// nothing: it is a pure, deterministic function of its input.
package assertion

import (
	"fmt"
	"go/ast"
	"go/token"
	"local/james-orcales/lint/internal/strings"
	"path"
	"sort"

	"local/james-orcales/lint/internal/diagnostic"
	"local/james-orcales/lint/internal/source"
	"local/james-orcales/shared/strconv"
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
	diags = append(diags, check_raw_types(parsed_files, exempt)...)
	diags = append(diags, check_collection_types(parsed_files, components)...)
	diags = append(diags, check_always_condition(parsed_files, components, exempt)...)
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
	base_kind := base_kind_declaration_index(parsed_files, components, false)
	for _, file := range parsed_files {
		if strings.Has_Suffix(file.Path, "_test.go") {
			continue
		}
		if source.Path_Matches_Glob(file.Path, exempt) {
			continue
		}
		directory := path.Dir(file.Path)
		diags = append(diags, invariant_file_diagnostics(
			file, components, constants[directory], base_kind)...)
	}
	return diags
}

// Maps each package-qualified type name to the type expression it stands over, after every chain of
// defined types is followed to its end. A name of its own hides no kind, thus a helper over a named
// integer still owes the scalar mandate and one over a named slice still owes the count mandate.
// The mandate releases test files, so it leaves them out; the blanket collection ban reads them.
func base_kind_declaration_index(
	parsed_files []Parsed_File, components *Component_Index, with_tests bool,
) (base_kind map[string]ast.Expr) {
	base_kind = map[string]ast.Expr{}
	named := map[string]string{}
	for _, pf := range parsed_files {
		if !with_tests {
			if strings.Has_Suffix(pf.Path, "_test.go") {
				continue
			}
		}
		package_path := helper_package_path(pf, components)
		// A qualified base names its package by the local name this file gave it, thus the
		// edge it declares resolves only against this file's own import table.
		imports := helper_import_paths(pf.File)
		for _, declaration := range pf.File.Decls {
			general, is_general := declaration.(*ast.GenDecl)
			if !is_general {
				continue
			}
			if general.Tok != token.TYPE {
				continue
			}
			base_kind_declaration_specs(
				general.Specs, package_path, imports, base_kind, named)
		}
	}
	base_kind_resolve_named(base_kind, named)
	return base_kind
}

// Records a type that stands directly over a kind this pass reads, and the name every other defined
// type stands over, which a later walk resolves.
func base_kind_declaration_specs(
	specifications []ast.Spec, package_path string, imports map[string]string,
	base_kind map[string]ast.Expr, named map[string]string,
) {
	for _, specification := range specifications {
		type_specification, is_type := specification.(*ast.TypeSpec)
		if !is_type {
			continue
		}
		if type_specification.Assign.IsValid() {
			continue
		}
		identity := package_path + "\x00" + type_specification.Name.Name
		foreign, qualified := base_kind_foreign_identity(
			type_specification.Type, imports)
		if qualified {
			named[identity] = foreign
			continue
		}
		identifier, is_identifier := type_specification.Type.(*ast.Ident)
		if !is_identifier {
			base_kind[identity] = type_specification.Type
			continue
		}
		if suffix, _, _ := invariant_identifier_kind(identifier.Name); suffix != "" {
			base_kind[identity] = type_specification.Type
			continue
		}
		named[identity] = package_path + "\x00" + identifier.Name
	}
}

// Names the declaration a qualified base stands over. A base outside this module, or under a
// qualifier the file never imported, resolves to nothing and stays where the caller puts it.
func base_kind_foreign_identity(
	base ast.Expr, imports map[string]string,
) (identity string, qualified bool) {

	selector, is_selector := base.(*ast.SelectorExpr)
	if !is_selector {
		return "", false
	}
	qualifier, is_qualifier := selector.X.(*ast.Ident)
	if !is_qualifier {
		return "", false
	}
	import_path := imports[qualifier.Name]
	if import_path == "" {
		return "", false
	}
	return import_path + "\x00" + selector.Sel.Name, true
}

// Follows each named edge to the kind at its end until no more resolve.
func base_kind_resolve_named(base_kind map[string]ast.Expr, named map[string]string) {
	for settled := false; !settled; {
		settled = true
		for identity, base := range named {
			if _, found := base_kind[identity]; found {
				continue
			}
			resolved, found := base_kind[base]
			if !found {
				continue
			}
			base_kind[identity] = resolved
			settled = false
		}
	}
}

// Gives the kind a type owes its mandate to, following the name it stands over when its own
// declaration spells another type rather than a kind this pass reads.
func invariant_resolved_kind(
	type_specification *ast.TypeSpec, package_path string, base_kind map[string]ast.Expr,
) (suffix string, primitive string, count bool) {
	suffix, primitive, count = invariant_type_kind(type_specification)
	if suffix != "" {
		return suffix, primitive, count
	}
	if type_specification.Assign.IsValid() {
		return "", "", false
	}
	resolved, found := base_kind[package_path+"\x00"+type_specification.Name.Name]
	if !found {
		return "", "", false
	}
	return invariant_type_kind(&ast.TypeSpec{
		Name: type_specification.Name, Type: resolved,
	})
}

// Constants are indexed per package because an argument in one package must not borrow a
// same-named declaration from another package to satisfy the deliberate-boundary rule.
func invariant_package_constants(
	parsed_files []Parsed_File,
) (constants map[string]map[string]bool) {
	constants = map[string]map[string]bool{}
	for _, file := range parsed_files {
		if strings.Has_Suffix(file.Path, "_test.go") {
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
	base_kind map[string]ast.Expr,
) (diags []Diagnostic) {
	package_path := helper_package_path(file, components)
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
		suffix, _, _ := invariant_resolved_kind(
			type_specification, package_path, base_kind)
		if suffix == "" {
			continue
		}
		diags = append(diags, invariant_type_diagnostics(
			file, components, constants, base_kind, index)...)
	}
	return diags
}

func invariant_type_diagnostics(
	file Parsed_File, components *Component_Index, constants map[string]bool,
	base_kind map[string]ast.Expr, index int,
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
		Base_Kind:       base_kind,
	}
	found, constant := invariant_body_helper(helper, type_specification, scope)
	if found {
		if constant {
			return nil
		}
		return invariant_constant_diagnostic(file, helper)
	}
	return invariant_missing_helper_diagnostic(file, helper, type_specification, scope)
}

// Gives the kind a bundle's own subject owes its mandate to.
func invariant_scope_kind(
	type_specification *ast.TypeSpec, scope *Invariant_Scope,
) (suffix string, primitive string, count bool) {
	return invariant_resolved_kind(
		type_specification, scope.Current_Package, scope.Base_Kind)
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
		return strings.To_Upper(name[:1]) + name[1:], name, false
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return strings.To_Upper(name[:1]) + name[1:], name, false
	case "byte":
		return "Uint8", "uint8", false
	case "rune":
		return "Int32", "int32", false
	case "float32", "float64":
		return strings.To_Upper(name[:1]) + name[1:], name, false
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
			matched, valid := invariant_direct_singleton(
				call, helper, type_specification, &statement_scope)
			if matched {
				if valid {
					return true, true
				}
				found = true
			}
			matched, valid = invariant_builder_preset(
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

// Reports whether primitive is one of the two floating-point widths.
func invariant_float_primitive(primitive string) (yes bool) {
	switch primitive {
	case "float32", "float64":
		return true
	}
	return false
}

func invariant_direct_singleton(
	call *ast.CallExpr, helper *ast.FuncDecl,
	type_specification *ast.TypeSpec, scope *Invariant_Scope,
) (matched bool, constant bool) {
	_, primitive, count := invariant_scope_kind(type_specification, scope)
	if !count {
		// A singleton Always states one legal value. A Boolean has two, thus the library
		// requires its bundle to ensure a Tree whose one link is a Sometimes, and an
		// Always here would name a shape the library rejects.
		if primitive == "bool" {
			return false, false
		}
		// A float lost its preset and has no builder preset, thus a singleton Always is
		// what remains for it, the same as for an integer.
		if !invariant_integer_primitive(primitive) {
			if !invariant_float_primitive(primitive) {
				return false, false
			}
		}
	}
	selector, is_selector := call.Fun.(*ast.SelectorExpr)
	if !is_selector {
		return false, false
	}
	qualifier, is_qualifier := selector.X.(*ast.Ident)
	if !is_qualifier {
		return false, false
	}
	if qualifier.Name != "aver" {
		return false, false
	}
	if selector.Sel.Name != "Always" {
		return false, false
	}
	identity := helper_callee_identity(
		call.Fun, scope.Current_Package, scope.Imports, scope.Shadowed)
	if identity != scope.Default_Package+"\x00Always" {
		return false, false
	}
	if len(call.Args) != 2 {
		return false, false
	}
	if !invariant_string_literal(call.Args[1]) {
		return false, false
	}
	equality, is_equality := invariant_unparen(call.Args[0]).(*ast.BinaryExpr)
	if !is_equality {
		return false, false
	}
	if equality.Op != token.EQL {
		return false, false
	}
	if !invariant_subject(equality.X, helper, type_specification, scope) {
		return false, false
	}
	return true, invariant_argument_constant(equality.Y, primitive, scope)
}

func invariant_integer_primitive(primitive string) (integer bool) {
	if strings.Has_Prefix(primitive, "int") {
		return true
	}
	return strings.Has_Prefix(primitive, "uint")
}

func unquote_literal(text string) (unquoted string, err error) {
	if len(text) > strconv.TEXT_SIZE_MAXIMUM {
		return "", fmt.Errorf("quoted text exceeds %d bytes", strconv.TEXT_SIZE_MAXIMUM)
	}
	var destination [strconv.UNQUOTED_TEXT_SIZE_MAXIMUM]byte
	count, err := strconv.Unquote_Into(destination[:], strconv.Text(text))
	if err != nil {
		return "", err
	}
	return string(destination[:int(count)]), nil
}

func invariant_string_literal(expression ast.Expr) (literal bool) {
	expression = invariant_unparen(expression)
	value, is_literal := expression.(*ast.BasicLit)
	if !is_literal {
		return false
	}
	if value.Kind != token.STRING {
		return false
	}
	_, unquote_error := unquote_literal(value.Value)
	return unquote_error == nil
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
	if !invariant_builder_root(current, helper, type_specification, scope) {
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
	suffix, primitive, _ := invariant_scope_kind(type_specification, scope)
	range_name := "Range_" + suffix
	range_holed_name := "Range_Holed_" + suffix
	enum_name := "Enum_" + suffix
	enum_3_name := "Enum_3_" + suffix
	enum_4_name := "Enum_4_" + suffix
	// A Boolean has no Range or Enum, thus Sometimes is its only link. Its two values are two
	// obligations, so one axis states the whole type.
	if primitive == "bool" {
		if method != "Sometimes" {
			return false, false
		}
		if len(call.Args) != 2 {
			return false, false
		}
		if !invariant_subject(call.Args[0], helper, type_specification, scope) {
			return false, false
		}
		// A Sometimes carries no domain operand, thus it has no constant to check.
		return true, true
	}
	switch method {
	case range_name, range_holed_name:
		argument_count := 3
		if method == range_holed_name {
			argument_count = 7
			if strings.Has_Prefix(suffix, "Uint") {
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

// Reports whether expression names the helper's own value parameter. A Tree root carries the whole
// subject, not a measurement of it, because the subject is what supplies the chain type. A count
// helper still writes len(value) at its link, and that is a separate predicate.
func invariant_root_subject(
	expression ast.Expr, helper *ast.FuncDecl, type_specification *ast.TypeSpec,
) (matched bool) {
	name, _ := invariant_value_parameter(helper, type_specification.Name.Name)
	if name == "" {
		return false
	}
	expression = invariant_unparen(expression)
	star, is_star := expression.(*ast.StarExpr)
	if is_star {
		expression = invariant_unparen(star.X)
	}
	return invariant_identifier(expression, name)
}

// A Tree root takes the helper's own subject before its namespace. The subject supplies the chain
// type, thus a root over another value would key its plan on a type the helper does not own.
func invariant_builder_root(
	call *ast.CallExpr, helper *ast.FuncDecl, type_specification *ast.TypeSpec,
	scope *Invariant_Scope,
) (matched bool) {
	if len(call.Args) != 2 {
		return false
	}
	if !invariant_root_subject(call.Args[0], helper, type_specification) {
		return false
	}
	if !invariant_identifier(call.Args[1], invariant_namespace_parameter(helper)) {
		return false
	}
	selector, is_selector := call.Fun.(*ast.SelectorExpr)
	if !is_selector {
		return false
	}
	if selector.Sel.Name != "Tree" {
		return false
	}
	qualifier, is_qualifier := selector.X.(*ast.Ident)
	if !is_qualifier {
		return false
	}
	if qualifier.Name != "aver" {
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
	_, primitive, count := invariant_scope_kind(type_specification, scope)
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
	if selector, qualified := expression.(*ast.SelectorExpr); qualified {
		return invariant_qualified_constant(selector, scope)
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

// Reports whether a qualified operand names a constant of a package this file imports. A bare name
// resolves only in its own package, thus the qualifier is what states the boundary a shared bound
// crosses. This pass reads one package at a time and never the imported one, so the name itself
// stays unchecked here. Registration parses that package and rejects a name it does not declare.
func invariant_qualified_constant(
	selector *ast.SelectorExpr, scope *Invariant_Scope,
) (valid bool) {
	qualifier, is_qualifier := selector.X.(*ast.Ident)
	if !is_qualifier {
		return false
	}
	if scope.Shadowed[qualifier.Name] {
		return false
	}
	return scope.Imports[qualifier.Name] != ""
}

func invariant_constant_diagnostic(
	file Parsed_File, helper *ast.FuncDecl,
) (diags []Diagnostic) {
	return []Diagnostic{{
		Position: file.File_Set.Position(helper.Name.Pos()),
		Message: fmt.Sprintf(
			"The function %s gives a canonical helper an argument that is not "+
				"a package-level constant. Write a package-level constant.",
			helper.Name.Name),
	}}
}

func invariant_missing_helper_diagnostic(
	file Parsed_File, helper *ast.FuncDecl, type_specification *ast.TypeSpec,
	scope *Invariant_Scope,
) (diags []Diagnostic) {
	_, _, count := invariant_scope_kind(type_specification, scope)
	value, _ := invariant_value_parameter(helper, type_specification.Name.Name)
	message := fmt.Sprintf(
		"The function %s does not call a canonical helper for %s. Write %s.",
		helper.Name.Name, value, invariant_remedy_text(type_specification, scope))
	if count {
		message = fmt.Sprintf(
			"The function %s does not call a canonical helper for len(%s). "+
				"Write a Range_Int family, an Enum_Int family, or a direct "+
				"Always equality.",
			helper.Name.Name, value)
	}
	return []Diagnostic{{
		Position: file.File_Set.Position(helper.Name.Pos()), Message: message,
	}}
}

// Names the forms a scalar can actually take. Only an integer has a Range and an Enum family, and a
// Boolean rejects a singleton Always because its two values are two obligations, thus one list
// pasted from the suffix would name a form that does not exist.
func invariant_remedy_text(
	type_specification *ast.TypeSpec, scope *Invariant_Scope,
) (remedy string) {
	suffix, primitive, _ := invariant_scope_kind(type_specification, scope)
	if primitive == "bool" {
		return "a Tree whose one link is a Sometimes, and then an Ensure"
	}
	if invariant_float_primitive(primitive) {
		return "a direct Always equality to a package constant"
	}
	return "an Always(<var> == <const>), a Range_" + suffix +
		" family, or an Enum_" + suffix + " family"
}

// Check_Type enforces the per-file type-invariant rules (Presence, Casing,
// Signature, Orphan, Scope): every in-scope type is followed directly by its
// correctly-named, correctly-signed bundle, and every bundle sits below its type.
// Test files and files under an exempt package are skipped.
func Check_Type(
	file_set *token.FileSet, file *ast.File, exempt []string,
) (diags []diagnostic.Diagnostic) {
	filename := file_set.Position(file.Pos()).Filename
	if strings.Has_Suffix(filename, "_test.go") {
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
	separator := strings.Last_Index_Byte(identity, '\x00')
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
			relative := strings.Trim_Prefix(directory, component.Root+"/")
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
		import_path, unquote_error := unquote_literal(specification.Path.Value)
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
		return shared + "/sim/aver/default"
	}
	for _, candidate := range imports {
		if strings.Has_Suffix(candidate, "/sim/aver/default") {
			return candidate
		}
		if strings.Has_Suffix(candidate, "/sim/aver") {
			import_path = candidate + "/default"
		}
	}
	if import_path != "" {
		return import_path
	}
	return "<sim/aver/default>"
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
	declarations := index_declarations(parsed_files, components)
	for _, pf := range parsed_files {
		if strings.Has_Suffix(pf.Path, "_test.go") {
			continue
		}
		if source.Path_Matches_Glob(pf.Path, exempt) {
			continue
		}
		diags = append(diags,
			struct_file_diagnostics(pf, defined, declarations, components)...)
		diags = append(diags, defined_pointer_file_diagnostics(pf, defined, components)...)
	}
	return diags
}

// Pointer helper owes one thing: pointee helper on dereferenced value. Nil exit comes first,
// because dereference of nil storage panics before any assertion runs. Any further statement
// restates pointee properties at a second position, thus body is exactly two statements.
//
// NO EXEMPTIONS. NONE AUTHORIZED. NONE WILL BE AUTHORIZED. EVERY defined pointer type gets this
// exact body. Struct pointee, non-struct pointee, destination, handle: ALL of them. Pointee
// without helper is a diagnostic, not a skip. Unnamed pointee is a diagnostic, not a skip. Owner
// has repaired this rule repeatedly after agents carved out a case. Last carve-out skipped struct
// pointees and pushed them into the per-field inherited check, which then demanded each field
// twice. Do NOT add a skip here. Do NOT add a skip upstream. Fix the code under lint instead.
func defined_pointer_file_diagnostics(
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
		if type_specification.Assign.IsValid() {
			continue
		}
		pointer, is_pointer := type_specification.Type.(*ast.StarExpr)
		if !is_pointer {
			continue
		}
		// Absent, misnamed, or mis-signed helper is the Layout, Orphan, and Signature
		// rules' report. Same defect twice hides nothing and helps nobody.
		helper := type_invariants_following_function(file.File, index)
		if helper == nil {
			continue
		}
		if helper.Name.Name != source.Invariant_Name(type_specification.Name.Name) {
			continue
		}
		parameter := struct_parameter_name(helper, type_specification.Name.Name)
		if parameter == "" {
			continue
		}
		scope := &Invariant_Scope{
			Type_Parameters: struct_type_parameter_set(type_specification),
			Defined:         defined,
			Current_Package: helper_package_path(file, components),
			Imports:         helper_import_paths(file.File),
			Shadowed:        function_value_names(helper),
		}
		gap := defined_pointer_gap(helper, pointer, parameter, scope)
		if gap == "" {
			continue
		}
		diags = append(diags, Diagnostic{
			Position: file.File_Set.Position(helper.Name.Pos()),
			Message:  "Function " + helper.Name.Name + " " + gap,
		})
	}
	return diags
}

// One helper, one gap. Pointee first: a body cannot be judged against a helper that has no name
// or does not exist, and each of those is its own defect.
func defined_pointer_gap(
	helper *ast.FuncDecl, pointer *ast.StarExpr, parameter string, scope *Invariant_Scope,
) (gap string) {
	expected := struct_named_invariant(pointer.X, scope)
	if expected == "" {
		return "points at an unnamed type. Name the pointee type, then call its " +
			"_Invariants on *" + parameter + "."
	}
	if !scope.Defined[expected] {
		return "points at " + type_base_name(pointer.X) + " without " +
			helper_identity_name(expected) + ". Declare " +
			helper_identity_name(expected) + "."
	}
	namespace := helper_namespace_parameter(helper)
	if defined_pointer_body_exact(helper, parameter, namespace, expected, scope) {
		return ""
	}
	return "body is not exactly `if " + parameter + " == nil { return }` then `" +
		defined_pointer_callee_text(pointer.X, expected) +
		"(*" + parameter + ", " + namespace + ")`."
}

// Remedy text is what the reader types. A foreign pointee's helper is reached through the
// file's own import name, thus the message carries that qualifier.
func defined_pointer_callee_text(pointee ast.Expr, expected string) (text string) {
	text = helper_identity_name(expected)
	selector, is_selector := pointee.(*ast.SelectorExpr)
	if !is_selector {
		return text
	}
	qualifier, is_identifier := selector.X.(*ast.Ident)
	if !is_identifier {
		return text
	}
	return qualifier.Name + "." + text
}

// Body's own Namespace parameter is the only namespace a nested bundle call may forward. A blank
// name cannot be forwarded, thus the message names the conventional one.
func helper_namespace_parameter(helper *ast.FuncDecl) (name string) {
	if helper.Type.Params == nil {
		return "namespace"
	}
	if len(helper.Type.Params.List) == 0 {
		return "namespace"
	}
	last := helper.Type.Params.List[len(helper.Type.Params.List)-1]
	if len(last.Names) == 0 {
		return "namespace"
	}
	name = last.Names[len(last.Names)-1].Name
	if name == "_" {
		return "namespace"
	}
	return name
}

// Exactly nil exit then pointee helper. Callee identity is package-qualified, thus a same-named
// foreign or shadowed function never substitutes.
func defined_pointer_body_exact(
	helper *ast.FuncDecl, parameter string, namespace string, expected string,
	scope *Invariant_Scope,
) (exact bool) {
	if len(helper.Body.List) != 2 {
		return false
	}
	if !defined_pointer_nil_guard(helper.Body.List[0], parameter) {
		return false
	}
	call := statement_call(helper.Body.List[1])
	if call == nil {
		return false
	}
	callee := helper_callee_identity(
		call.Fun, scope.Current_Package, scope.Imports, scope.Shadowed)
	if callee != expected {
		return false
	}
	if len(call.Args) != 2 {
		return false
	}
	dereference, is_dereference := call.Args[0].(*ast.StarExpr)
	if !is_dereference {
		return false
	}
	subject, is_identifier := dereference.X.(*ast.Ident)
	if !is_identifier {
		return false
	}
	if subject.Name != parameter {
		return false
	}
	forwarded, is_identifier := call.Args[1].(*ast.Ident)
	if !is_identifier {
		return false
	}
	return forwarded.Name == namespace
}

// Only `if parameter == nil { return }`: no init, no else, one bare return. Same shape sim/aver
// permits in a pointer bundle, thus the two rules agree on one body.
func defined_pointer_nil_guard(statement ast.Stmt, parameter string) (guards bool) {
	guard, is_if := statement.(*ast.IfStmt)
	if !is_if {
		return false
	}
	if guard.Init != nil {
		return false
	}
	if guard.Else != nil {
		return false
	}
	condition, is_binary := guard.Cond.(*ast.BinaryExpr)
	if !is_binary {
		return false
	}
	if condition.Op != token.EQL {
		return false
	}
	subject, is_identifier := condition.X.(*ast.Ident)
	if !is_identifier {
		return false
	}
	if subject.Name != parameter {
		return false
	}
	nil_identifier, is_identifier := condition.Y.(*ast.Ident)
	if !is_identifier {
		return false
	}
	if nil_identifier.Name != "nil" {
		return false
	}
	if len(guard.Body.List) != 1 {
		return false
	}
	exit, is_return := guard.Body.List[0].(*ast.ReturnStmt)
	if !is_return {
		return false
	}
	return len(exit.Results) == 0
}

// Checks every literal struct type in one file, and every defined type that stands over a
// same-package struct and thus inherits its fields. A defined pointer is not here: it has its
// own exact-body rule in defined_pointer_file_diagnostics.
func struct_file_diagnostics(
	file Parsed_File, defined map[string]bool,
	declarations *Declaration_Index, components *Component_Index,
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
		struct_type, is_struct, inherits := struct_declared_fields(
			type_specification, file, declarations.Structs, components)
		if !is_struct {
			continue
		}
		if len(struct_type.Fields.List) == 0 {
			continue
		}
		if struct_has_mutex(struct_type) {
			continue
		}
		diags = append(diags, struct_type_diagnostics(&Struct_Type_Input{
			File:               file,
			Index:              index,
			Type_Specification: type_specification,
			Struct_Type:        struct_type,
			Defined:            defined,
			Declarations:       declarations,
			Inherits:           inherits,
			Components:         components,
		})...)
	}
	return diags
}

// One struct declaration under check, with what its bundle is judged against.
type Struct_Type_Input struct {
	// File holds the declaration.
	File Parsed_File
	// Index is the declaration's position in File, because the bundle follows it directly.
	Index int
	// Type_Specification is the declaration.
	Type_Specification *ast.TypeSpec
	// Struct_Type holds the fields the bundle owes, own or inherited.
	Struct_Type *ast.StructType
	// Defined is the module-wide helper index.
	Defined map[string]bool
	// Declarations classifies an inherited field's type.
	Declarations *Declaration_Index
	// Inherits selects the inherited-field rule over the own-field rule.
	Inherits bool
	// Components resolves package paths.
	Components *Component_Index
}

// Checks that one struct's bundle composes every coverable field.
func struct_type_diagnostics(input *Struct_Type_Input) (diags []Diagnostic) {
	file := input.File
	index := input.Index
	type_specification := input.Type_Specification
	struct_type := input.Struct_Type
	defined := input.Defined
	components := input.Components
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
	if input.Inherits {
		scope.Declarations = input.Declarations
		scope.Constants = input.Declarations.Constants[path.Dir(file.Path)]
		return struct_inherited_diagnostics(
			bundle, struct_type, scope, present, parameter, position)
	}
	for _, field := range struct_type.Fields.List {
		for _, gap := range struct_field_missing_calls(field, scope, present, parameter) {
			diags = append(diags, Diagnostic{
				Position: position,
				Message:  "The function " + bundle.Name.Name + " " + gap,
			})
		}
	}
	return diags
}

// Flags an Always whose condition joins terms with && or ||. Such a condition collapses a preset
// into one guard: `a == A || a == B` is an Enum, which owes a membership guard and one axis for
// each member, and `x >= MIN && x <= MAX` is a Range, which owes two guards and its bound axes. As
// one Always each owes one guard, thus a suite that never reaches the second member or either
// boundary still runs clean.
func check_always_condition(
	parsed_files []Parsed_File, components *Component_Index, exempt []string,
) (diags []Diagnostic) {
	for _, pf := range parsed_files {
		if strings.Has_Suffix(pf.Path, "_test.go") {
			continue
		}
		if source.Path_Matches_Glob(pf.Path, exempt) {
			continue
		}
		diags = append(diags, always_condition_diagnostics(pf, components)...)
	}
	return diags
}

// Walks one file for an Always call whose condition is a logical join.
func always_condition_diagnostics(
	file Parsed_File, components *Component_Index,
) (diags []Diagnostic) {
	imports := helper_import_paths(file.File)
	default_package := helper_default_package(components, imports)
	ast.Inspect(file.File, func(node ast.Node) (descend bool) {
		return always_condition_node(file, imports, default_package, node, &diags)
	})
	return diags
}

func always_condition_node(
	file Parsed_File, imports map[string]string, default_package string,
	node ast.Node, diags *[]Diagnostic,
) (descend bool) {
	call, is_call := node.(*ast.CallExpr)
	if !is_call {
		return true
	}
	if !always_named_call(call, imports, default_package) {
		return true
	}
	if len(call.Args) == 0 {
		return true
	}
	join, is_join := invariant_unparen(call.Args[0]).(*ast.BinaryExpr)
	if !is_join {
		return true
	}
	if join.Op != token.LAND {
		if join.Op != token.LOR {
			return true
		}
	}
	*diags = append(*diags, Diagnostic{
		Position: file.File_Set.Position(call.Args[0].Pos()),
		Message: fmt.Sprintf(
			"The Always condition has the compound operator %q. "+
				"Write a Range for a bound, an Enum for a membership, "+
				"or separate Always calls.", join.Op.String()),
	})
	return true
}

// Reports whether call names the composition tier's Always or Recorder_Always.
func always_named_call(
	call *ast.CallExpr, imports map[string]string, default_package string,
) (matched bool) {
	selector, is_selector := call.Fun.(*ast.SelectorExpr)
	if !is_selector {
		return false
	}
	if selector.Sel.Name != "Always" {
		if selector.Sel.Name != "Recorder_Always" {
			return false
		}
	}
	qualifier, is_qualifier := selector.X.(*ast.Ident)
	if !is_qualifier {
		return false
	}
	return imports[qualifier.Name] == default_package
}

// Declaration_Index names what each defined type of the module stands over, because an inherited
// field is judged by the kind at the end of its type's chain and not by the name it carries.
type Declaration_Index struct {
	// Structs maps a package-qualified type name to the struct it declares or stands over.
	Structs map[string]*ast.StructType
	// Booleans holds each package-qualified type name that stands over bool.
	Booleans map[string]bool
	// Constants holds each package's constant names by directory, for singleton Always
	// operands.
	Constants map[string]map[string]bool
}

// One walk over every declaration serves both indexes. A defined type over a same-package name
// is followed to its end, thus type A B, type B struct{} puts A's fields in reach.
func index_declarations(
	parsed_files []Parsed_File, components *Component_Index,
) (index *Declaration_Index) {
	index = &Declaration_Index{
		Structs:   map[string]*ast.StructType{},
		Booleans:  map[string]bool{},
		Constants: invariant_package_constants(parsed_files),
	}
	chains := map[string]string{}
	for _, file := range parsed_files {
		if strings.Has_Suffix(file.Path, "_test.go") {
			continue
		}
		package_path := helper_package_path(file, components)
		for _, declaration := range file.File.Decls {
			general, is_general := declaration.(*ast.GenDecl)
			if !is_general {
				continue
			}
			if general.Tok != token.TYPE {
				continue
			}
			for _, specification := range general.Specs {
				index_declaration_specification(
					specification, package_path, index, chains)
			}
		}
	}
	declaration_resolve_chains(index, chains)
	return index
}

// An alias adds a name and no identity, thus it is never indexed. A pointer to a struct still
// reaches every field, thus a field typed by a defined pointer composes like a struct field.
func index_declaration_specification(
	specification ast.Spec, package_path string,
	index *Declaration_Index, chains map[string]string,
) {
	type_specification, is_type := specification.(*ast.TypeSpec)
	if !is_type {
		return
	}
	if type_specification.Assign.IsValid() {
		return
	}
	identity := package_path + "\x00" + type_specification.Name.Name
	if struct_type, literal := type_specification.Type.(*ast.StructType); literal {
		index.Structs[identity] = struct_type
		return
	}
	base, named := struct_declared_base(type_specification.Type)
	if !named {
		return
	}
	if base == "bool" {
		index.Booleans[identity] = true
		return
	}
	chains[identity] = package_path + "\x00" + base
}

// Every chain ends within the declaration count, because a cycle does not compile. The bound
// keeps the walk finite on input the compiler never saw.
func declaration_resolve_chains(index *Declaration_Index, chains map[string]string) {
	for identity := range chains {
		current := identity
		for step := 0; step <= len(chains); step++ {
			if struct_type, found := index.Structs[current]; found {
				index.Structs[identity] = struct_type
				break
			}
			if index.Booleans[current] {
				index.Booleans[identity] = true
				break
			}
			next, chained := chains[current]
			if !chained {
				break
			}
			current = next
		}
	}
}

// Gives the name a type declaration stands over. A leading star is unwrapped because Go selects a
// field through a pointer, thus a field typed by a defined pointer holds a struct's fields too.
func struct_declared_base(expression ast.Expr) (name string, named bool) {
	star, is_star := expression.(*ast.StarExpr)
	if is_star {
		expression = star.X
	}
	identifier, is_identifier := expression.(*ast.Ident)
	if !is_identifier {
		return "", false
	}
	return identifier.Name, true
}

// Gives the struct whose fields a type declaration owns, and whether it inherits them. A literal
// struct owns its own. A defined type over a same-package struct inherits that struct's fields
// and states them under the inherited rule. A defined pointer never reaches here: its body is
// exactly the nil exit and the pointee helper, see defined_pointer_file_diagnostics. An alias is
// out of scope.
func struct_declared_fields(
	type_specification *ast.TypeSpec, file Parsed_File,
	struct_index map[string]*ast.StructType, components *Component_Index,
) (struct_type *ast.StructType, is_struct bool, inherits bool) {
	if declared, literal := type_specification.Type.(*ast.StructType); literal {
		return declared, true, false
	}
	if type_specification.Assign.IsValid() {
		return nil, false, false
	}
	identifier, is_identifier := type_specification.Type.(*ast.Ident)
	if !is_identifier {
		return nil, false, false
	}
	identity := helper_package_path(file, components) + "\x00" + identifier.Name
	inherited, found := struct_index[identity]
	return inherited, found, found
}

// Checks that a defined type's bundle states every field it inherits. The inline and the
// converted names are collected one time so the per-field walk stays out of the bundle body.
func struct_inherited_diagnostics(
	bundle *ast.FuncDecl,
	struct_type *ast.StructType,
	scope *Invariant_Scope,
	present map[string]bool,
	parameter string,
	position token.Position,
) (diags []Diagnostic) {
	gaps_input := &Struct_Inherited_Field_Gaps_Input{
		Scope:     scope,
		Present:   present,
		Converted: struct_converted_fields(bundle, parameter, scope),
		Inline:    struct_inline_fields(bundle, struct_type, parameter, scope),
		Parameter: parameter,
	}
	for _, field := range struct_type.Fields.List {
		gaps_input.Field = field
		for _, gap := range struct_inherited_field_gaps(gaps_input) {
			diags = append(diags, Diagnostic{
				Position: position,
				Message:  "The function " + bundle.Name.Name + " " + gap,
			})
		}
	}
	return diags
}

// The three name sets one inherited field is checked against.
type Struct_Inherited_Field_Gaps_Input struct {
	// Field is the inherited field under check.
	Field *ast.Field
	// Scope resolves the field type and the bundle it must carry.
	Scope *Invariant_Scope
	// Present holds helper\x00field for each direct helper call in the bundle.
	Present map[string]bool
	// Converted holds each field the bundle composes through a defined type of its own.
	Converted map[string]bool
	// Inline holds each field a Range, Enum, Sometimes, or singleton Always states.
	Inline map[string]bool
	// Parameter is the bundle's value name.
	Parameter string
}

// A scalar inherited field is stated inline, because the struct it comes from already holds its
// type under any shared root and a second helper call would place it twice. A struct field has no
// inline form, thus it is composed, directly or through a defined type of its own.
func struct_inherited_field_gaps(
	gaps_input *Struct_Inherited_Field_Gaps_Input,
) (gaps []string) {
	field := gaps_input.Field
	names := struct_field_names(field)
	if len(names) == 0 {
		return nil
	}
	expected := struct_field_invariant(field.Type, gaps_input.Scope)
	if expected == "" {
		return nil
	}
	if !gaps_input.Scope.Defined[expected] {
		return nil
	}
	is_struct := struct_field_is_struct(field.Type, gaps_input.Scope)
	for _, name := range names {
		subject := gaps_input.Parameter + "." + name
		if !is_struct {
			if !gaps_input.Inline[name] {
				gaps = append(gaps, "does not assert the inherited field "+
					subject+" inline. Write an inline assertion for "+
					subject+".")
			}
			continue
		}
		if gaps_input.Present[expected+"\x00"+name] {
			continue
		}
		if gaps_input.Converted[name] {
			continue
		}
		gaps = append(gaps, "does not call a helper for the inherited field "+
			subject+". Call "+helper_identity_name(expected)+"("+subject+", ...).")
	}
	return gaps
}

// Reports whether a field's type is a struct. Only a struct keeps its composition duty under a
// defined type, because no single link states a struct. A foreign type is opaque to this pass, thus
// it keeps that duty too rather than owe an inline form this pass cannot read.
func struct_field_is_struct(field_type ast.Expr, scope *Invariant_Scope) (yes bool) {
	star, is_star := field_type.(*ast.StarExpr)
	if is_star {
		field_type = star.X
	}
	if _, literal := field_type.(*ast.StructType); literal {
		return true
	}
	if _, foreign := field_type.(*ast.SelectorExpr); foreign {
		return true
	}
	identifier, is_identifier := field_type.(*ast.Ident)
	if !is_identifier {
		return false
	}
	_, found := scope.Declarations.Structs[scope.Current_Package+"\x00"+identifier.Name]
	return found
}

// Gives the inherited fields a bundle states inline. A Range or an Enum link states a domain, a
// direct Always states a single value, and a Sometimes states only a Boolean.
func struct_inline_fields(
	bundle *ast.FuncDecl, struct_type *ast.StructType,
	parameter string, scope *Invariant_Scope,
) (inline map[string]bool) {
	inline = map[string]bool{}
	boolean_fields := struct_boolean_fields(struct_type, scope)
	for _, statement := range bundle.Body.List {
		call := statement_call(statement)
		if call == nil {
			continue
		}
		for _, name := range struct_chain_fields(call, parameter, boolean_fields) {
			inline[name] = true
		}
		if !always_named_call(call, scope.Imports, scope.Default_Package) {
			continue
		}
		name := struct_always_field(call, parameter, scope)
		if name != "" {
			inline[name] = true
		}
	}
	return inline
}

// A singleton is the one domain with no Range and no Enum, thus a direct equality against a
// package constant is the only inline form that states it. A literal names no shared fact and a
// comparison holds one side, thus neither counts.
func struct_always_field(
	call *ast.CallExpr, parameter string, scope *Invariant_Scope,
) (name string) {
	if len(call.Args) == 0 {
		return ""
	}
	equality, is_binary := invariant_unparen(call.Args[0]).(*ast.BinaryExpr)
	if !is_binary {
		return ""
	}
	if equality.Op != token.EQL {
		return ""
	}
	if !struct_constant_operand(equality.Y, scope) {
		return ""
	}
	return struct_expression_field(equality.X, parameter)
}

// A constant operand is a package constant of this package or a qualified one of an imported
// package, under at most one conversion. A shadowed name denotes a local value, never the
// constant.
func struct_constant_operand(expression ast.Expr, scope *Invariant_Scope) (constant bool) {
	expression = struct_unwrap_conversion(expression, scope.Shadowed)
	if selector, qualified := expression.(*ast.SelectorExpr); qualified {
		return invariant_qualified_constant(selector, scope)
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

// Strips one conversion, because an inline link takes a primitive and a value of a defined kind
// reaches it through one. A shadowed conversion name is a local function, not a conversion.
func struct_unwrap_conversion(
	expression ast.Expr, shadowed map[string]bool,
) (unwrapped ast.Expr) {
	expression = invariant_unparen(expression)
	call, is_call := expression.(*ast.CallExpr)
	if !is_call {
		return expression
	}
	if len(call.Args) != 1 {
		return expression
	}
	conversion, is_identifier := call.Fun.(*ast.Ident)
	if !is_identifier {
		return expression
	}
	if shadowed[conversion.Name] {
		return expression
	}
	return invariant_unparen(call.Args[0])
}

// Gives the inherited fields whose type stands over bool, the one kind a Sometimes states in
// full.
func struct_boolean_fields(
	struct_type *ast.StructType, scope *Invariant_Scope,
) (booleans map[string]bool) {
	booleans = map[string]bool{}
	for _, field := range struct_type.Fields.List {
		identifier, is_identifier := field.Type.(*ast.Ident)
		if !is_identifier {
			continue
		}
		identity := scope.Current_Package + "\x00" + identifier.Name
		if !scope.Declarations.Booleans[identity] {
			continue
		}
		for _, name := range struct_field_names(field) {
			booleans[name] = true
		}
	}
	return booleans
}

// Walks one ensured chain from Ensure back to its root and names the field each stating link
// takes. The root takes the whole value and names no field, thus the walk ends there.
func struct_chain_fields(
	call *ast.CallExpr, parameter string, boolean_fields map[string]bool,
) (names []string) {
	current, ensured := invariant_ensure_receiver(call)
	if !ensured {
		return nil
	}
	for current != nil {
		method, receiver, matched := invariant_builder_method(current)
		if !matched {
			break
		}
		name := struct_link_field(current, parameter)
		if name != "" {
			if struct_link_states(method, boolean_fields[name]) {
				names = append(names, name)
			}
		}
		current = receiver
	}
	return names
}

// The link's first argument is its subject.
func struct_link_field(link *ast.CallExpr, parameter string) (name string) {
	if len(link.Args) == 0 {
		return ""
	}
	return struct_expression_field(link.Args[0], parameter)
}

// A Range or an Enum states a domain. A Sometimes states two values and nothing more, thus it
// states a Boolean and no other kind.
func struct_link_states(method string, boolean bool) (states bool) {
	if strings.Has_Prefix(method, "Range_") {
		return true
	}
	if strings.Has_Prefix(method, "Enum_") {
		return true
	}
	if method != "Sometimes" {
		return false
	}
	return boolean
}

// Reads parameter.Field under at most one conversion and one dereference, because an inline link
// takes a primitive and a field of a defined kind reaches it through a conversion.
func struct_expression_field(expression ast.Expr, parameter string) (field string) {
	expression = struct_unwrap_conversion(expression, map[string]bool{})
	if star, is_star := expression.(*ast.StarExpr); is_star {
		expression = invariant_unparen(star.X)
	}
	selector, is_selector := expression.(*ast.SelectorExpr)
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

// Gives the inherited struct fields a bundle composes through a defined type of its own. The Go
// compiler already proves the conversion shares the field's underlying type, thus this pass only
// checks that the called bundle belongs to the converted type.
func struct_converted_fields(
	bundle *ast.FuncDecl, parameter string, scope *Invariant_Scope,
) (converted map[string]bool) {
	converted = map[string]bool{}
	shadowed := function_shadow_copy(scope.Shadowed)
	for _, statement := range bundle.Body.List {
		call := statement_call(statement)
		if call != nil {
			defined, field := struct_conversion_argument(call, parameter)
			callee := helper_callee_identity(
				call.Fun, scope.Current_Package, scope.Imports, shadowed)
			owner := scope.Current_Package + "\x00" + source.Invariant_Name(defined)
			if field != "" {
				if callee == owner {
					converted[field] = true
				}
			}
		}
		function_statement_shadows(statement, shadowed)
	}
	return converted
}

// Reads a first argument of the form Defined(parameter.Field).
func struct_conversion_argument(
	call *ast.CallExpr, parameter string,
) (defined string, field string) {
	if len(call.Args) == 0 {
		return "", ""
	}
	conversion, is_call := invariant_unparen(call.Args[0]).(*ast.CallExpr)
	if !is_call {
		return "", ""
	}
	if len(conversion.Args) != 1 {
		return "", ""
	}
	identifier, is_identifier := conversion.Fun.(*ast.Ident)
	if !is_identifier {
		return "", ""
	}
	selector, is_selector := invariant_unparen(conversion.Args[0]).(*ast.SelectorExpr)
	if !is_selector {
		return "", ""
	}
	base, is_base := selector.X.(*ast.Ident)
	if !is_base {
		return "", ""
	}
	if base.Name != parameter {
		return "", ""
	}
	return identifier.Name, selector.Sel.Name
}

// Package-qualified keys ensure adding Foo_Invariants in one package cannot silently impose or
// satisfy a helper requirement for an unrelated Foo in another package.
func struct_helper_index(
	parsed_files []Parsed_File, components *Component_Index,
) (defined map[string]bool) {
	defined = map[string]bool{}
	for _, pf := range parsed_files {
		if strings.Has_Suffix(pf.Path, "_test.go") {
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

// Separating field discovery from diagnostic ownership keeps bundle identity and position out of
// an argument-bundling struct.
func struct_field_missing_calls(
	field *ast.Field,
	scope *Invariant_Scope,
	present map[string]bool,
	parameter string,
) (gaps []string) {
	names := struct_field_names(field)
	if len(names) == 0 {
		return nil
	}
	expected := struct_field_invariant(field.Type, scope)
	if expected == "" {
		return nil
	}
	if !scope.Defined[expected] {
		return nil
	}
	for _, name := range names {
		if present[expected+"\x00"+name] {
			continue
		}
		subject := parameter + "." + name
		gaps = append(gaps, "does not call a helper for the field "+subject+
			". Call "+helper_identity_name(expected)+"("+subject+", ...).")
	}
	return gaps
}

// Gives the names a field is selected by. An embedded field declares none, and Go selects it by the
// type it embeds, thus anonymity hides it from a reader and never from the mandate.
func struct_field_names(field *ast.Field) (names []string) {
	if len(field.Names) != 0 {
		for _, name := range field.Names {
			names = append(names, name.Name)
		}
		return names
	}
	embedded := field.Type
	star, is_star := embedded.(*ast.StarExpr)
	if is_star {
		embedded = star.X
	}
	switch typed := embedded.(type) {
	case *ast.Ident:
		return []string{typed.Name}
	case *ast.SelectorExpr:
		return []string{typed.Sel.Name}
	default:
		return nil
	}
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
) (identity string) {

	// A pointer field composes its pointee: *Token requires Token_Invariants, the
	// same as a Token field. The bundle passes the field, or its dereference, as the
	// value — struct_present_calls accepts both.
	star, is_pointer := field_type.(*ast.StarExpr)
	if is_pointer {
		field_type = star.X
	}
	switch typed := field_type.(type) {
	case *ast.ArrayType:
		// Raw slice field is banned by check_raw_types, not composed here.
		return ""
	case *ast.MapType:
		// Raw map field is banned by check_raw_types, not composed here.
		return ""
	case *ast.SelectorExpr:
		package_path := struct_selector_package(typed, scope.Imports)
		if package_path == "" {
			return ""
		}
		return package_path + "\x00" + typed.Sel.Name + "_Invariants"
	case *ast.IndexExpr:
		return struct_named_invariant(typed.X, scope)
	case *ast.IndexListExpr:
		return struct_named_invariant(typed.X, scope)
	case *ast.Ident:
		return struct_field_ident_invariant(typed.Name, scope)
	default:
		return ""
	}
}

// Returns the bundle name for a generic instantiation's base (an ident or a
// cross-package selector), or "" otherwise.
func struct_named_invariant(
	base ast.Expr, scope *Invariant_Scope,
) (identity string) {

	selector, is_selector := base.(*ast.SelectorExpr)
	if is_selector {
		package_path := struct_selector_package(selector, scope.Imports)
		if package_path == "" {
			return ""
		}
		return package_path + "\x00" + selector.Sel.Name + "_Invariants"
	}
	identifier, is_identifier := base.(*ast.Ident)
	if is_identifier {
		return struct_field_ident_invariant(identifier.Name, scope)
	}
	return ""
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

// Predeclared identifiers own no package bundle; user-declared identifiers may own one.
func struct_field_ident_invariant(
	name string, scope *Invariant_Scope,
) (identity string) {

	if scope.Type_Parameters[name] {
		return ""
	}
	if struct_is_builtin(name) {
		return ""
	}
	return scope.Current_Package + "\x00" + source.Invariant_Name(name)
}

// Reports whether name is a predeclared type that has no preset, so a field of it
// is exempt rather than mistaken for a defined type.
func struct_is_builtin(name string) (yes bool) {
	if suffix, _, _ := invariant_identifier_kind(name); suffix != "" {
		return true
	}
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
		if strings.Has_Suffix(pf.Path, "_test.go") {
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
	// Base_Kind resolves a type that stands over another name to the kind at the end of that
	// chain, thus a name of its own hides no mandate.
	Base_Kind map[string]ast.Expr
	// Declarations tells an inherited field that owns fields from one a link can state, and
	// names the Boolean types, the one kind a Sometimes states in full.
	Declarations *Declaration_Index
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
				Message: fmt.Sprintf(
					"The function %s does not call a helper for the "+
						"output %s. Call %s in a defer that is the "+
						"first statement.",
					name, requirement.Subject, function_form(requirement))})
			continue
		}
		if !function_requirement_met(defer_body, requirement, scope) {
			diags = append(diags, Diagnostic{Position: position,
				Message: fmt.Sprintf(
					"The function %s does not call a helper for the "+
						"output %s. Call %s in the output defer.",
					name, requirement.Subject, function_form(requirement))})
		}
	}
	for _, requirement := range inputs {
		if function_requirement_met(lead, requirement, scope) {
			continue
		}
		diags = append(diags, Diagnostic{Position: position,
			Message: fmt.Sprintf(
				"The function %s does not call a helper for the input %s. "+
					"Call %s at the start of the body.",
				name, requirement.Subject, function_form(requirement))})
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

// Constructed collection types own no helper name; check_raw_types diagnoses boundary instead.
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
// invariant coverage recorder. aver.Run_Test_Main is the canonical TestMain
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
	if strings.Has_Suffix(file.Path, "_test.go") {
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
// func TestMain(m *testing.M) { aver.Run_Test_Main(m) } — its parameter
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

// Reports whether the call is exactly aver.Run_Test_Main(m).
func recorder_canonical_call(call *ast.CallExpr) (canonical bool) {
	selector, is_selector := call.Fun.(*ast.SelectorExpr)
	if !is_selector {
		return false
	}
	qualifier, is_qualifier := selector.X.(*ast.Ident)
	if !is_qualifier {
		return false
	}
	if qualifier.Name != "aver" {
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
			Message: fmt.Sprintf(
				"The directory %s has no TestMain that calls "+
					"aver.Run_Test_Main. Write a TestMain that calls "+
					"aver.Run_Test_Main.", group.Directory),
		}}
	}
	if !group.Test_Main_Canonical {
		return []Diagnostic{{
			Position: group.Test_Main_Position,
			Message: "The body of TestMain is not aver.Run_Test_Main(m). " +
				"Write aver.Run_Test_Main(m).",
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
		return simulation_diagnostic(position, fmt.Sprintf(
			"The binary component %q has no internal/%s package. "+
				"Declare an internal/%s package that calls internal.Main.",
			component.Import_Path, SIMULATION_DIRECTORY, SIMULATION_DIRECTORY))
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
		if strings.Has_Suffix(pf.Path, "_test.go") {
			continue
		}
		if components.File_To_Component[pf.Path] != component_index_number {
			continue
		}
		directory := path.Dir(pf.Path)
		under_internal := directory == internal_root
		if strings.Has_Prefix(directory, internal_root+"/") {
			under_internal = true
		}
		if !under_internal {
			continue
		}
		if directory == sim_directory {
			continue
		}
		if strings.Has_Prefix(directory, sim_directory+"/") {
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
		if !strings.Has_Suffix(pf.Path, "_test.go") {
			return simulation_diagnostic(position, fmt.Sprintf(
				"The simulation directory has the source file %s. "+
					"Remove the source file.", pf.Path))
		}
		if !strings.Has_Suffix(pf.File.Name.Name, "_test") {
			return simulation_diagnostic(position, fmt.Sprintf(
				"The simulation package %s is not an external test package. "+
					"Add the suffix %q to the package name.",
				pf.File.Name.Name, "_test"))
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
		"The simulation package has no fuzz function for internal.Main. "+
			"Declare a fuzz function that calls internal.Main.")
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
	if !strings.Has_Prefix(function.Name.Name, "Fuzz") {
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
		if strings.Has_Suffix(pf.Path, "_test.go") {
			continue
		}
		if components.File_To_Component[pf.Path] != component_index_number {
			continue
		}
		directory := path.Dir(pf.Path)
		under := directory == internal_root
		if strings.Has_Prefix(directory, internal_root+"/") {
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
		if strings.Has_Prefix(import_path, internal_import_path+"/") {
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
			return simulation_entry_node(
				locals, internal_functions, position, node, &diags)
		})
	}
	return diags
}

func simulation_entry_node(
	locals map[string]bool, internal_functions map[string]bool,
	position token.Position, node ast.Node, diags *[]Diagnostic,
) (descend bool) {
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
	*diags = append(*diags, simulation_diagnostic(position, fmt.Sprintf(
		"The simulation package refers to %s.%s. Refer only to Main.",
		identifier.Name, selector.Sel.Name))...)
	return true
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

// The simulation's TestMain body must be exactly aver.Run_Test_Main(m, "../**"):
// that one glob registers the internal package and every package beneath it, so the
// isolated simulation binary seeds and judges them all without enumerating each.
func simulation_test_main_diagnostics(
	files []Parsed_File, position token.Position,
) (diags []Diagnostic) {
	function := simulation_find_test_main(files)
	if function == nil {
		return simulation_diagnostic(position,
			"The simulation package has no TestMain. "+
				"Write a TestMain that calls aver.Run_Test_Main.")
	}
	if simulation_test_main_canonical(function) {
		return nil
	}
	return simulation_diagnostic(position, fmt.Sprintf(
		"The body of the simulation TestMain is not "+
			"aver.Run_Test_Main(m, %q). "+
			"Write aver.Run_Test_Main(m, %q).",
		SIMULATION_GLOB, SIMULATION_GLOB))
}

// Reports whether the TestMain body is exactly aver.Run_Test_Main(m, "../**").
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
// that statement is exactly aver.Run_Test_Main(m, <string literals>).
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
// aver.Run_Test_Main(m, <one or more string literals>).
func simulation_call_directories(call *ast.CallExpr) (directories []string, canonical bool) {
	selector, is_selector := call.Fun.(*ast.SelectorExpr)
	if !is_selector {
		return nil, false
	}
	qualifier, is_qualifier := selector.X.(*ast.Ident)
	if !is_qualifier {
		return nil, false
	}
	if qualifier.Name != "aver" {
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
	unquoted, unquote_error := unquote_literal(literal.Value)
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

// Boundary types need names because only names can own invariant bundles. Shares type-invariant
// exemptions.
func check_raw_types(parsed_files []Parsed_File, exempt []string) (diags []Diagnostic) {
	for _, pf := range parsed_files {
		if strings.Has_Suffix(pf.Path, "_test.go") {
			continue
		}
		if source.Path_Matches_Glob(pf.Path, exempt) {
			continue
		}
		diags = append(diags, raw_type_file_diagnostics(pf)...)
	}
	return diags
}

// One file walk keeps function and field enforcement under same syntax rule.
func raw_type_file_diagnostics(file Parsed_File) (diags []Diagnostic) {
	for _, declaration := range file.File.Decls {
		switch typed := declaration.(type) {
		case *ast.FuncDecl:
			diags = append(diags, raw_type_function_diagnostics(file, typed)...)
		case *ast.GenDecl:
			diags = append(diags, raw_type_struct_diagnostics(file, typed)...)
		}
	}
	return diags
}

func raw_type_function_diagnostics(
	file Parsed_File, function *ast.FuncDecl,
) (diags []Diagnostic) {
	position := file.File_Set.Position(function.Name.Pos())
	parameter_gaps := raw_type_field_gaps(function.Type.Params, "parameter")
	diags = append(diags,
		boundary_owner_diagnostics(parameter_gaps, function.Name.Name, position)...)
	result_gaps := raw_type_field_gaps(function.Type.Results, "result")
	diags = append(diags,
		boundary_owner_diagnostics(result_gaps, function.Name.Name, position)...)
	return diags
}

// Only named struct declarations create field boundaries; inline structs get caught at owner
// boundary that contains them.
func raw_type_struct_diagnostics(file Parsed_File, general *ast.GenDecl) (diags []Diagnostic) {
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
		gaps := raw_type_field_gaps(struct_type.Fields, "field")
		position := file.File_Set.Position(type_specification.Name.Pos())
		diags = append(diags,
			boundary_owner_diagnostics(
				gaps, type_specification.Name.Name, position)...)
	}
	return diags
}

// Same AST field list represents every boundary role, keeping syntax judgment identical.
func raw_type_field_gaps(fields *ast.FieldList, role string) (gaps []string) {
	if fields == nil {
		return nil
	}
	for _, field := range fields.List {
		if type_is_identifier(field.Type) {
			continue
		}
		for _, identifier := range field_declaration_names(field) {
			// Embedded field owns no explicit identifier, so diagnostic omits subject decoration.
			named := ""
			if identifier != "" {
				named = " (" + identifier + ")"
			}
			gaps = append(gaps, "has a raw type "+role+named+
				". Declare a defined type for the "+role+".")
		}
	}
	return gaps
}

// Ownership stays separate because collection rule uses same declaration-shaped diagnostics.
func boundary_owner_diagnostics(
	gaps []string, owner string, position token.Position,
) (diags []Diagnostic) {
	for _, gap := range gaps {
		diags = append(diags, Diagnostic{
			Position: position,
			Message:  "The declaration " + owner + " " + gap,
		})
	}
	return diags
}

// Anonymous field still needs diagnostic even though no declared name can decorate message.
func field_declaration_names(field *ast.Field) (names []string) {
	if len(field.Names) == 0 {
		return []string{""}
	}
	for _, name := range field.Names {
		names = append(names, name.Name)
	}
	return names
}

// Selector must start at package identifier; arbitrary selector expression names no type.
// Predeclared identifier is raw: int, string, and kin carry no _Invariants and never can, thus a
// boundary typed by one has no mandate at all. Every parameter, result, and field is a
// user-defined type. Sole pass: error. An interface has no length and no domain, thus no
// assertion could state one. Owner allowed error and nothing else.
func type_is_identifier(expression ast.Expr) (yes bool) {
	if identifier, is_identifier := expression.(*ast.Ident); is_identifier {
		if identifier.Name == "error" {
			return true
		}
		return !struct_is_builtin(identifier.Name)
	}
	selector, is_selector := expression.(*ast.SelectorExpr)
	if !is_selector {
		return false
	}
	_, is_qualifier := selector.X.(*ast.Ident)
	return is_qualifier
}

// SMALL_SLICE_COUNT_MAX is the largest len a slice may be bounded to and stay a slice. At or below
// it, the members are few enough to name, and a struct with one field per member states the shape
// a slice only implies through a loop.
const SMALL_SLICE_COUNT_MAX = 8

// CONSTANT_CHAIN_DEPTH_MAX bounds how many constant aliases a bound may be followed through. A
// cycle between constants does not compile, so the cap trips only on a pathological chain, which
// is then left unjudged rather than walked forever.
const CONSTANT_CHAIN_DEPTH_MAX = 16

// Constant_Value is one package-level constant's initializer beside the import table of its file,
// because a bound may cross a package boundary more than once and each hop resolves qualifiers
// through the imports of the file that spelled it.
type Constant_Value struct {
	// Expression is the constant's initializer.
	Expression ast.Expr
	// Package qualifies the bare names the initializer uses.
	Package string
	// Imports resolves the qualified names the initializer uses.
	Imports map[string]string
}

// Collection_Index holds what the small-slice and fixed-array bans need of every type: whether its
// chain of defined types ends at a fixed array, and the largest len its helper states for a slice.
type Collection_Index struct {
	// Array marks a package-qualified type whose base kind is a fixed [N]T.
	Array map[string]bool
	// Slice_Bound gives a package-qualified slice type the len upper bound its helper resolves to.
	Slice_Bound map[string]int64
}

// Flags a fixed array or a small bounded slice used as a struct field, parameter, or result. A
// slice bounded to SMALL_SLICE_COUNT_MAX or fewer is a struct wearing a loop. A fixed array instead
// loses caller control over storage and length, so its remedy is a slice. The ban is blanket: no
// helper, stdlib method, test file, or opted-out package is released. A bound no chain of constants
// resolves is left unjudged.
func check_collection_types(
	parsed_files []Parsed_File, components *Component_Index,
) (diags []Diagnostic) {
	index := collection_index(parsed_files, components)
	for _, pf := range parsed_files {
		scope := &Invariant_Scope{
			Current_Package: helper_package_path(pf, components),
			Imports:         helper_import_paths(pf.File),
		}
		for _, declaration := range pf.File.Decls {
			switch typed := declaration.(type) {
			case *ast.FuncDecl:
				diags = append(diags,
					collection_function_diagnostics(pf, typed, scope, index)...)
			case *ast.GenDecl:
				diags = append(diags,
					collection_struct_diagnostics(pf, typed, scope, index)...)
			}
		}
	}
	return diags
}

// Builds the array set from the base-kind index and the slice bounds from every count helper.
func collection_index(
	parsed_files []Parsed_File, components *Component_Index,
) (index *Collection_Index) {
	index = &Collection_Index{Array: map[string]bool{}, Slice_Bound: map[string]int64{}}
	base_kind := base_kind_declaration_index(parsed_files, components, true)
	for identity, base := range base_kind {
		array, is_array := base.(*ast.ArrayType)
		if !is_array {
			continue
		}
		if array.Len == nil {
			continue
		}
		index.Array[identity] = true
	}
	values := constant_value_index(parsed_files, components)
	for _, pf := range parsed_files {
		collection_file_bounds(pf, components, base_kind, values, index)
	}
	return index
}

// Records the len upper bound of every slice type in one file whose helper states one.
func collection_file_bounds(
	pf Parsed_File, components *Component_Index, base_kind map[string]ast.Expr,
	values map[string]Constant_Value, index *Collection_Index,
) {
	package_path := helper_package_path(pf, components)
	imports := helper_import_paths(pf.File)
	scope := &Invariant_Scope{
		Current_Package: package_path,
		Imports:         imports,
		Default_Package: helper_default_package(components, imports),
		Base_Kind:       base_kind,
	}
	for declaration_index, declaration := range pf.File.Decls {
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
		if !collection_is_slice(type_specification, scope) {
			continue
		}
		helper := type_invariants_following_function(pf.File, declaration_index)
		if helper == nil {
			continue
		}
		if helper.Name.Name != source.Invariant_Name(type_specification.Name.Name) {
			continue
		}
		helper_scope := *scope
		helper_scope.Shadowed = function_value_names(helper)
		bound, resolved := collection_helper_bound(
			helper, type_specification, &helper_scope, values)
		if !resolved {
			continue
		}
		index.Slice_Bound[package_path+"\x00"+type_specification.Name.Name] = bound
	}
}

// Reports whether the type's chain of defined types ends at a slice, not a string or a map, which
// share the count mandate but have no members to name.
func collection_is_slice(
	type_specification *ast.TypeSpec, scope *Invariant_Scope,
) (yes bool) {
	if type_specification.Assign.IsValid() {
		return false
	}
	resolved := type_specification.Type
	if _, is_identifier := resolved.(*ast.Ident); is_identifier {
		base, found := scope.Base_Kind[scope.Current_Package+"\x00"+type_specification.Name.Name]
		if !found {
			return false
		}
		resolved = base
	}
	array, is_array := resolved.(*ast.ArrayType)
	if !is_array {
		return false
	}
	return array.Len == nil
}

// Reads the largest len the helper allows: a Range upper bound, the largest Enum member, or the
// Always singleton. The helper is a count helper by the Count Helper mandate, so each canonical
// link states len(value) first and its bound operands after.
func collection_helper_bound(
	helper *ast.FuncDecl, type_specification *ast.TypeSpec, scope *Invariant_Scope,
	values map[string]Constant_Value,
) (bound int64, resolved bool) {
	shadowed := function_shadow_copy(scope.Shadowed)
	for _, statement := range helper.Body.List {
		call := statement_call(statement)
		if call != nil {
			statement_scope := *scope
			statement_scope.Shadowed = shadowed
			operands := collection_singleton_operands(call, helper, type_specification, &statement_scope)
			if len(operands) == 0 {
				operands = collection_builder_operands(
					call, helper, type_specification, &statement_scope)
			}
			if len(operands) > 0 {
				return collection_operands_bound(operands, &statement_scope, values)
			}
		}
		function_statement_shadows(statement, shadowed)
	}
	return 0, false
}

// Returns the singleton operand of an aver.Always(len(value) == CONSTANT, "...") over the subject.
func collection_singleton_operands(
	call *ast.CallExpr, helper *ast.FuncDecl, type_specification *ast.TypeSpec,
	scope *Invariant_Scope,
) (operands []ast.Expr) {
	identity := helper_callee_identity(
		call.Fun, scope.Current_Package, scope.Imports, scope.Shadowed)
	if identity != scope.Default_Package+"\x00Always" {
		return nil
	}
	if len(call.Args) != 2 {
		return nil
	}
	equality, is_equality := invariant_unparen(call.Args[0]).(*ast.BinaryExpr)
	if !is_equality {
		return nil
	}
	if equality.Op != token.EQL {
		return nil
	}
	if !collection_subject(equality.X, helper, type_specification, scope) {
		return nil
	}
	return []ast.Expr{equality.Y}
}

// Returns the bound operands of the first Range_Int or Enum_Int link over the subject in an
// ensured Tree builder: a Range yields its upper edge, an Enum every member.
func collection_builder_operands(
	ensure *ast.CallExpr, helper *ast.FuncDecl, type_specification *ast.TypeSpec,
	scope *Invariant_Scope,
) (operands []ast.Expr) {
	current, ensured := invariant_ensure_receiver(ensure)
	if !ensured {
		return nil
	}
	for step_index := 0; step_index < ASSERTIONS_BUILDER_LINKS_MAX; step_index++ {
		method, receiver, linked := invariant_builder_method(current)
		if !linked {
			return nil
		}
		operands = collection_link_operands(method, current, helper, type_specification, scope)
		if len(operands) > 0 {
			return operands
		}
		current = receiver
	}
	return nil
}

// Picks the bound operands out of one link when it is a count link over the subject.
func collection_link_operands(
	method string, call *ast.CallExpr, helper *ast.FuncDecl,
	type_specification *ast.TypeSpec, scope *Invariant_Scope,
) (operands []ast.Expr) {
	if len(call.Args) < 2 {
		return nil
	}
	if !collection_subject(call.Args[0], helper, type_specification, scope) {
		return nil
	}
	switch method {
	case "Range_Int", "Range_Holed_Int":
		// The upper edge is the second bound operand; a holed range's holes sit below it.
		if len(call.Args) < 3 {
			return nil
		}
		return []ast.Expr{call.Args[2]}
	case "Enum_Int", "Enum_3_Int", "Enum_4_Int":
		return call.Args[1:]
	}
	return nil
}

// Reports whether expression is len(value) over the helper's own subject, the one form a count
// link's first operand takes.
func collection_subject(
	expression ast.Expr, helper *ast.FuncDecl, type_specification *ast.TypeSpec,
	scope *Invariant_Scope,
) (matched bool) {
	value, pointer := invariant_value_parameter(helper, type_specification.Name.Name)
	if value == "" {
		return false
	}
	call, is_call := invariant_unparen(expression).(*ast.CallExpr)
	if !is_call {
		return false
	}
	if !invariant_identifier(call.Fun, "len") {
		return false
	}
	if scope.Shadowed["len"] {
		return false
	}
	if len(call.Args) != 1 {
		return false
	}
	return invariant_value(call.Args[0], value, pointer)
}

// Resolves every operand and keeps the largest; one unresolved operand leaves the bound unjudged,
// because a member that cannot be read may be the one that exceeds the cap.
func collection_operands_bound(
	operands []ast.Expr, scope *Invariant_Scope, values map[string]Constant_Value,
) (bound int64, resolved bool) {
	for operand_index, operand := range operands {
		value, ok := constant_integer(operand, scope.Current_Package, scope.Imports, values, 0)
		if !ok {
			return 0, false
		}
		if operand_index == 0 {
			bound = value
		}
		if value > bound {
			bound = value
		}
	}
	return bound, true
}

// Maps each package-qualified constant name to its initializer, so a bound spelled as another
// constant, in this package or an imported one, can be followed to a literal.
func constant_value_index(
	parsed_files []Parsed_File, components *Component_Index,
) (values map[string]Constant_Value) {
	values = map[string]Constant_Value{}
	for _, pf := range parsed_files {
		package_path := helper_package_path(pf, components)
		imports := helper_import_paths(pf.File)
		for _, declaration := range pf.File.Decls {
			general, is_general := declaration.(*ast.GenDecl)
			if !is_general {
				continue
			}
			if general.Tok != token.CONST {
				continue
			}
			constant_value_specs(general.Specs, package_path, imports, values)
		}
	}
	return values
}

// Records each named constant with an initializer of its own. Grouped Declarations bans const
// groups and Iota bans iota, so a spec without its own value has nothing to record.
func constant_value_specs(
	specifications []ast.Spec, package_path string, imports map[string]string,
	values map[string]Constant_Value,
) {
	for _, specification := range specifications {
		value, is_value := specification.(*ast.ValueSpec)
		if !is_value {
			continue
		}
		for name_index, name := range value.Names {
			if name_index >= len(value.Values) {
				continue
			}
			values[package_path+"\x00"+name.Name] = Constant_Value{
				Expression: value.Values[name_index],
				Package:    package_path,
				Imports:    imports,
			}
		}
	}
}

// Evaluates a constant expression to an integer: a literal, a bare or qualified constant name
// followed to its initializer, a conversion, a negation, or a + - * / << over two such operands.
// Anything else, an overflow, or a chain past CONSTANT_CHAIN_DEPTH_MAX resolves to nothing.
func constant_integer(
	expression ast.Expr, package_path string, imports map[string]string,
	values map[string]Constant_Value, depth int,
) (value int64, resolved bool) {
	if depth > CONSTANT_CHAIN_DEPTH_MAX {
		return 0, false
	}
	expression = invariant_unparen(expression)
	switch typed := expression.(type) {
	case *ast.BasicLit:
		return constant_literal(typed)
	case *ast.Ident:
		return constant_named(package_path+"\x00"+typed.Name, values, depth)
	case *ast.SelectorExpr:
		qualifier, is_qualifier := typed.X.(*ast.Ident)
		if !is_qualifier {
			return 0, false
		}
		import_path := imports[qualifier.Name]
		if import_path == "" {
			return 0, false
		}
		return constant_named(import_path+"\x00"+typed.Sel.Name, values, depth)
	case *ast.CallExpr:
		// A one-argument call in a constant initializer is a conversion, which changes the
		// type and never the value.
		if len(typed.Args) != 1 {
			return 0, false
		}
		return constant_integer(typed.Args[0], package_path, imports, values, depth+1)
	case *ast.UnaryExpr:
		if typed.Op != token.SUB {
			return 0, false
		}
		value, resolved = constant_integer(typed.X, package_path, imports, values, depth+1)
		if !resolved {
			return 0, false
		}
		if value == strconv.INTEGER_64_MINIMUM {
			return 0, false
		}
		return -value, true
	case *ast.BinaryExpr:
		return constant_binary(typed, package_path, imports, values, depth)
	}
	return 0, false
}

// Follows a constant name to its initializer, evaluated in the package and imports that spelled it.
func constant_named(
	identity string, values map[string]Constant_Value, depth int,
) (value int64, resolved bool) {
	constant, found := values[identity]
	if !found {
		return 0, false
	}
	return constant_integer(
		constant.Expression, constant.Package, constant.Imports, values, depth+1)
}

// Reads an integer literal in any Go base with any digit separators; a string, rune, or float
// literal is not a len bound.
func constant_literal(literal *ast.BasicLit) (value int64, resolved bool) {
	if literal.Kind != token.INT {
		return 0, false
	}
	if len(literal.Value) > strconv.TEXT_SIZE_MAXIMUM {
		return 0, false
	}
	parsed, parse_error := strconv.Parse_Integer(
		strconv.Text(literal.Value), strconv.IMPLIED_BASE_MINIMUM, strconv.BIT_SIZE_MAXIMUM)
	if parse_error != nil {
		return 0, false
	}
	return int64(parsed), true
}

// Evaluates one arithmetic step with an overflow check on each, because a wrapped product or shift
// could land below the cap and ban a slice whose real bound is huge.
func constant_binary(
	binary *ast.BinaryExpr, package_path string, imports map[string]string,
	values map[string]Constant_Value, depth int,
) (value int64, resolved bool) {
	left, left_ok := constant_integer(binary.X, package_path, imports, values, depth+1)
	if !left_ok {
		return 0, false
	}
	right, right_ok := constant_integer(binary.Y, package_path, imports, values, depth+1)
	if !right_ok {
		return 0, false
	}
	switch binary.Op {
	case token.ADD:
		value = left + right
		return value, (value > left) == (right > 0)
	case token.SUB:
		value = left - right
		return value, (value < left) == (right > 0)
	case token.MUL:
		if left == 0 {
			return 0, true
		}
		value = left * right
		if value/left != right {
			return 0, false
		}
		return value, value != strconv.INTEGER_64_MINIMUM
	case token.QUO:
		if right == 0 {
			return 0, false
		}
		return left / right, true
	case token.SHL:
		if right < 0 {
			return 0, false
		}
		if right > 62 {
			return 0, false
		}
		if left < 0 {
			return 0, false
		}
		value = left << right
		return value, value>>right == left
	}
	return 0, false
}

// Flags every function's fixed-array and small-slice parameters and results, a helper and a
// stdlib method included: the type itself is what the ban removes, so its helper goes with it.
func collection_function_diagnostics(
	file Parsed_File, function *ast.FuncDecl, scope *Invariant_Scope, index *Collection_Index,
) (diags []Diagnostic) {
	position := file.File_Set.Position(function.Name.Pos())
	gaps := collection_field_gaps(function.Type.Params, "parameter", scope, index)
	gaps = append(gaps, collection_field_gaps(function.Type.Results, "result", scope, index)...)
	return boundary_owner_diagnostics(gaps, function.Name.Name, position)
}

// Flags each struct type's fixed-array and small-slice fields.
func collection_struct_diagnostics(
	file Parsed_File, general *ast.GenDecl, scope *Invariant_Scope, index *Collection_Index,
) (diags []Diagnostic) {
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
		gaps := collection_field_gaps(struct_type.Fields, "field", scope, index)
		position := file.File_Set.Position(type_specification.Name.Pos())
		diags = append(diags,
			boundary_owner_diagnostics(gaps, type_specification.Name.Name, position)...)
	}
	return diags
}

// Describes each field whose type is a fixed array or a small bounded slice, in the same shape
// raw_type_field_gaps uses so both bans read alike.
func collection_field_gaps(
	fields *ast.FieldList, role string, scope *Invariant_Scope, index *Collection_Index,
) (gaps []string) {
	if fields == nil {
		return nil
	}
	for _, field := range fields.List {
		gap := collection_type_gap(field.Type, role, scope, index)
		if gap == "" {
			continue
		}
		for _, identifier := range field_declaration_names(field) {
			named := ""
			if identifier != "" {
				named = " (" + identifier + ")"
			}
			gaps = append(gaps, strings.Replace(gap, "\x00", named, 1))
		}
	}
	return gaps
}

// Judges one field type, a leading * unwrapped: a raw [N]T or a defined type over one is a fixed
// array, a defined slice type whose bound resolves at or under the cap is a small slice. The
// placeholder marks where the field's own name goes once it is known.
func collection_type_gap(
	expression ast.Expr, role string, scope *Invariant_Scope, index *Collection_Index,
) (gap string) {
	core := expression
	if star, is_star := core.(*ast.StarExpr); is_star {
		core = star.X
	}
	if array, is_array := core.(*ast.ArrayType); is_array {
		if array.Len == nil {
			return ""
		}
		return "has a fixed array " + role + "\x00. " +
			"Convert fixed arrays to slices instead."
	}
	identity := collection_type_identity(core, scope)
	if identity == "" {
		return ""
	}
	if index.Array[identity] {
		return "has a fixed array " + role + "\x00. " +
			"Convert fixed arrays to slices instead."
	}
	bound, bounded := index.Slice_Bound[identity]
	if !bounded {
		return ""
	}
	if bound > SMALL_SLICE_COUNT_MAX {
		return ""
	}
	return "has a " + helper_identity_name(identity) + " " + role + "\x00 bounded to " +
		fmt.Sprintf("%d", bound) + " members. Declare a struct with one field per member instead."
}

// Names the package-qualified type a bare or qualified type expression refers to.
func collection_type_identity(expression ast.Expr, scope *Invariant_Scope) (identity string) {
	switch typed := expression.(type) {
	case *ast.Ident:
		return scope.Current_Package + "\x00" + typed.Name
	case *ast.SelectorExpr:
		foreign, qualified := base_kind_foreign_identity(typed, scope.Imports)
		if !qualified {
			return ""
		}
		return foreign
	}
	return ""
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
		Message: fmt.Sprintf(
			"The type %s has no invariants function. "+
				"Declare %s(%s, aver.Namespace) directly below the type %s.",
			type_name, want, type_name, type_name),
	}
}

// Builds the diagnostic for a bundle whose parameters
// are not the type, by value or pointer, first and an aver.Namespace last.
func type_invariants_bad_signature(
	file_set *token.FileSet, function *ast.FuncDecl, type_specification *ast.TypeSpec,
) (diag Diagnostic) {

	type_name := type_specification.Name.Name
	return Diagnostic{
		Position: file_set.Position(function.Name.Pos()),
		Message: fmt.Sprintf(
			"The function %s has incorrect parameters. "+
				"Write the parameters (%s or *%s, aver.Namespace).",
			function.Name.Name, type_name, type_name),
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
			Message: fmt.Sprintf(
				"The function %s is not directly below the type of the same "+
					"name. Declare the function %s directly below the type.",
				function.Name.Name, function.Name.Name),
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
			Message: fmt.Sprintf(
				"There is a comment between %s and %s. Remove the comment.",
				type_specification.Name.Name, function.Name.Name),
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
	if strings.Has_Suffix(name, "_Invariants") {
		return true
	}
	return strings.Has_Suffix(name, "_invariants")
}

// Reports whether the bundle takes its type, by
// value or pointer, first and an aver.Namespace last.
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
	if len(want) != len(got) {
		return false
	}
	for index, name := range want {
		if name != got[index] {
			return false
		}
	}
	return true
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
		unquoted, unquote_err := unquote_literal(specification.Path.Value)
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
		names[source.Import_Local_Name(specification, unquoted)] = true
	}
	return names
}

// Reports whether an import path has a segment
// named aver — the framework package, wherever it sits in the module.
func type_invariants_path_is_invariant(import_path string) (yes bool) {
	for _, segment := range strings.Split(import_path, "/") {
		if segment == "aver" {
			return true
		}
	}
	return false
}
