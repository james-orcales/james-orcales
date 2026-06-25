// Package assertion enforces the _Invariants doctrine — every in-scope type
// states its properties in a companion bundle function beside it, and every named
// function asserts its typed inputs and outputs — and the sibling simulation
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

	"github.com/james-orcales/james-orcales/lint/internal/diagnostic"
	"github.com/james-orcales/james-orcales/lint/internal/source"
)

// Unexported aliases so the moved rule bodies name these types unqualified, as
// they did in package lint.
type parsed_file = source.Parsed_File
type component_index = source.Component_Index
type component_information = source.Component

// Diagnostic aliases the diagnostic package's type so the moved rule bodies name
// it unqualified.
type Diagnostic = diagnostic.Diagnostic

// Check_Input carries the parsed set, the component graph, and the exempt list.
type Check_Input struct {
	// Parsed_Files is the whole parsed tree.
	Parsed_Files []source.Parsed_File
	// Components is the workspace's component graph; recorder and simulation need it.
	Components *source.Component_Index
	// Exempt is lint.json's invariant_exempt_packages.
	Exempt []string
}

// Check runs the cross-file invariant and simulation checks over the parsed set,
// in the order the doctrine aggregator ran them.
func Check(input *Check_Input) (diags []diagnostic.Diagnostic) {
	diags = append(diags, check_numeric_invariants(input.Parsed_Files, input.Exempt)...)
	diags = append(diags, check_struct_invariants(input.Parsed_Files, input.Exempt)...)
	diags = append(diags, check_function_invariants(input.Parsed_Files, input.Exempt)...)
	diags = append(diags,
		check_recorder_test_main(input.Parsed_Files, input.Components, input.Exempt)...)
	diags = append(diags, check_primitive_types(input.Parsed_Files, input.Exempt)...)
	diags = append(diags,
		check_simulation(input.Parsed_Files, input.Components, input.Exempt)...)
	return diags
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

// Flags numeric defined types whose bundle omits a required bound guard, bound
// constant, or boundary claim. Cross-file because a bound's constant may live in
// a sibling file, so the package's const set is gathered first. Shares the
// type-invariant rule's opt-out and skips test files, and fires only when the
// bundle exists — absence is the presence rule's job.
func check_numeric_invariants(
	parsed_files []parsed_file, exempt []string,
) (diags []Diagnostic) {

	constants := numeric_package_constants(parsed_files)
	for _, pf := range parsed_files {
		if strings.HasSuffix(pf.Path, "_test.go") {
			continue
		}
		if source.Path_Matches_Glob(pf.Path, exempt) {
			continue
		}
		diags = append(diags,
			numeric_file_diagnostics(pf, constants[path.Dir(pf.Path)])...)
	}
	return diags
}

// Maps each package directory to the set of top-level const names declared in its
// non-test files — the names a numeric bound may legitimately reference.
func numeric_package_constants(
	parsed_files []parsed_file,
) (constants map[string]map[string]bool) {

	constants = map[string]map[string]bool{}
	for _, pf := range parsed_files {
		if strings.HasSuffix(pf.Path, "_test.go") {
			continue
		}
		directory := path.Dir(pf.Path)
		if constants[directory] == nil {
			constants[directory] = map[string]bool{}
		}
		numeric_collect_constants(pf.File, constants[directory])
	}
	return constants
}

// Adds every top-level const name in file to the set.
func numeric_collect_constants(file *ast.File, into map[string]bool) {
	for _, declaration := range file.Decls {
		general, is_general := declaration.(*ast.GenDecl)
		if !is_general {
			continue
		}
		if general.Tok != token.CONST {
			continue
		}
		for _, specification := range general.Specs {
			value_specification, is_value := specification.(*ast.ValueSpec)
			if !is_value {
				continue
			}
			for _, name := range value_specification.Names {
				into[name.Name] = true
			}
		}
	}
}

// Checks every numeric defined type + value-parameter bundle pair in one file.
func numeric_file_diagnostics(
	file parsed_file, constants map[string]bool,
) (diags []Diagnostic) {

	invariant_names := type_invariants_import_names(file.File)
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
		kind, count := numeric_subject_kind(type_specification)
		if kind == "" {
			continue
		}
		diags = append(diags, numeric_type_diagnostics(&numeric_type_input{
			File: file, Index: index, Type: type_specification, Kind: kind,
			Count: count, Constants: constants, Invariant_Names: invariant_names,
		})...)
	}
	return diags
}

// Groups one numeric type with the context its bundle check needs: it carries two
// maps, which loose parameters may not repeat.
type numeric_type_input struct {
	File            parsed_file
	Index           int
	Type            *ast.TypeSpec
	Kind            string
	Count           bool
	Constants       map[string]bool
	Invariant_Names map[string]bool
}

// Checks the bundle directly below one numeric type, or nothing when the type has
// no value-parameter bundle (a pointer-parameter or absent bundle is skipped).
func numeric_type_diagnostics(input *numeric_type_input) (diags []Diagnostic) {
	candidate := type_invariants_following_function(input.File.File, input.Index)
	if candidate == nil {
		return nil
	}
	if candidate.Name.Name != source.Invariant_Name(input.Type.Name.Name) {
		return nil
	}
	value := numeric_value_parameter_name(candidate, input.Type.Name.Name)
	if value == "" {
		return nil
	}
	return numeric_bundle_diagnostics(&numeric_bundle_input{
		File_Set: input.File.File_Set, Bundle: candidate, Value: value,
		Count: input.Count, Kind: input.Kind, Constants: input.Constants,
		Invariant_Names: input.Invariant_Names,
	})
}

// Classifies a type declaration's numeric kind ("signed", "unsigned", "float"),
// or "" when it is not a defined type over a direct builtin numeric.
func numeric_type_kind(type_specification *ast.TypeSpec) (kind string) {
	if type_specification.Assign.IsValid() {
		return ""
	}
	if type_specification.TypeParams != nil {
		return ""
	}
	base, is_identifier := type_specification.Type.(*ast.Ident)
	if !is_identifier {
		return ""
	}
	return numeric_kind(base.Name)
}

// Maps a builtin numeric type name to its signedness class, or "" for non-numeric.
func numeric_kind(name string) (kind string) {
	switch name {
	case "int", "int8", "int16", "int32", "int64", "rune":
		return "signed"
	case "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "byte":
		return "unsigned"
	case "float32", "float64":
		return "float"
	default:
		return ""
	}
}

// Returns the boundary-coverage kind and whether it applies to the value's
// count. A numeric defined type yields its signedness and false; a string,
// slice, or map type yields "unsigned" and true; anything else "".
func numeric_subject_kind(type_specification *ast.TypeSpec) (kind string, count bool) {
	kind = numeric_type_kind(type_specification)
	if kind != "" {
		return kind, false
	}
	if numeric_is_count_type(type_specification) {
		return "unsigned", true
	}
	return "", false
}

// Reports whether the type is a defined string, slice, or map — a container whose
// count carries the boundary discipline. Generics are in scope; an alias, a
// fixed array, or a channel is not.
func numeric_is_count_type(type_specification *ast.TypeSpec) (yes bool) {
	if type_specification.Assign.IsValid() {
		return false
	}
	switch base := type_specification.Type.(type) {
	case *ast.MapType:
		return true
	case *ast.ArrayType:
		return base.Len == nil
	case *ast.Ident:
		return base.Name == "string"
	default:
		return false
	}
}

// Renders the asserted subject for diagnostics: the value, or len(value).
func numeric_subject_text(input *numeric_bundle_input) (text string) {
	if input.Count {
		return "len(" + input.Value + ")"
	}
	return input.Value
}

// Returns the bundle's first parameter name when it is the type by value, or ""
// when there is no parameter or it is a pointer. The value may be a generic
// instantiation (v Stack[T]), so the base name is compared.
func numeric_value_parameter_name(
	bundle *ast.FuncDecl, type_name string,
) (name string) {

	if bundle.Type.Params == nil {
		return ""
	}
	if len(bundle.Type.Params.List) == 0 {
		return ""
	}
	first := bundle.Type.Params.List[0]
	if numeric_type_base_name(first.Type) != type_name {
		return ""
	}
	if len(first.Names) == 0 {
		return ""
	}
	return first.Names[0].Name
}

// Returns the base type name of a value parameter: a bare identifier, or the base
// of a generic instantiation Name[...]; "" for a pointer or anything else.
func numeric_type_base_name(expression ast.Expr) (name string) {
	identifier, is_identifier := expression.(*ast.Ident)
	if is_identifier {
		return identifier.Name
	}
	base, _ := type_invariants_instantiation(expression)
	if base == nil {
		return ""
	}
	return base.Name
}

// Carries the bundle and the facts its checks read: two maps keep it off loose
// parameters.
type numeric_bundle_input struct {
	File_Set        *token.FileSet
	Bundle          *ast.FuncDecl
	Value           string
	Count           bool
	Kind            string
	Constants       map[string]bool
	Invariant_Names map[string]bool
}

// The body summary one bundle yields: which bounds it guards, the named operands
// of those guards, and which boundary values it claims.
type numeric_facts struct {
	Has_Upper  bool
	Has_Lower  bool
	Upper_Name string
	Lower_Name string
	// Claimed_Values holds every boundary value the bundle names in any equality or
	// inequality claim, by either Always or Sometimes.
	Claimed_Values map[string]bool
}

// Collects bound and coverage diagnostics for one numeric bundle.
func numeric_bundle_diagnostics(input *numeric_bundle_input) (diags []Diagnostic) {
	facts := numeric_collect_facts(input)
	diags = append(diags, numeric_bound_diagnostics(facts, input)...)
	diags = append(diags, numeric_coverage_diagnostics(facts, input)...)
	return diags
}

// Walks the bundle body, summarizing every Always/Sometimes condition into facts.
func numeric_collect_facts(input *numeric_bundle_input) (facts numeric_facts) {
	facts.Claimed_Values = map[string]bool{}
	is_subject := numeric_subject_matcher(input)
	ast.Inspect(input.Bundle.Body, func(node ast.Node) (recurse bool) {
		call, is_call := node.(*ast.CallExpr)
		if !is_call {
			return true
		}
		is_always, matched := numeric_invariant_call(call, input.Invariant_Names)
		if !matched {
			return true
		}
		numeric_classify_condition(call.Args[0], is_subject, is_always, &facts)
		return true
	})
	return facts
}

// Reports whether an expression is the asserted subject — the value, or its count.
type numeric_subject func(expression ast.Expr) (matches bool)

// Builds the predicate that recognizes the asserted subject: the value itself, or
// its count when the type is a string, slice, or map.
func numeric_subject_matcher(input *numeric_bundle_input) (match numeric_subject) {
	if input.Count {
		return func(expression ast.Expr) (matches bool) {
			return numeric_is_count(expression, input.Value)
		}
	}
	return func(expression ast.Expr) (matches bool) {
		return numeric_is_value(expression, input.Value)
	}
}

// Reports whether expression is len(value).
func numeric_is_count(expression ast.Expr, value string) (yes bool) {
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
	if len(call.Args) != 1 {
		return false
	}
	return numeric_is_value(call.Args[0], value)
}

// Reports whether call is invariant.Always/Sometimes and which, by the local
// import name; matched is false for any other call or one with no arguments.
func numeric_invariant_call(
	call *ast.CallExpr, invariant_names map[string]bool,
) (is_always bool, matched bool) {

	if len(call.Args) == 0 {
		return false, false
	}
	selector, is_selector := call.Fun.(*ast.SelectorExpr)
	if !is_selector {
		return false, false
	}
	qualifier, is_identifier := selector.X.(*ast.Ident)
	if !is_identifier {
		return false, false
	}
	if !invariant_names[qualifier.Name] {
		return false, false
	}
	if selector.Sel.Name == "Always" {
		return true, true
	}
	if selector.Sel.Name == "Sometimes" {
		return false, true
	}
	return false, false
}

// Folds one condition into facts: bounds come only from Always; claims (equality,
// inequality, NaN, infinities) from either Always or Sometimes.
func numeric_classify_condition(
	condition ast.Expr, is_subject numeric_subject, is_always bool,
	facts *numeric_facts,
) {
	if numeric_nan_call(condition) {
		facts.Claimed_Values["NaN"] = true
		return
	}
	binary, is_binary := condition.(*ast.BinaryExpr)
	if !is_binary {
		return
	}
	if binary.Op == token.LEQ {
		numeric_record_upper(binary, is_subject, is_always, facts)
		return
	}
	if binary.Op == token.GEQ {
		numeric_record_lower(binary, is_subject, is_always, facts)
		return
	}
	if binary.Op == token.EQL {
		numeric_record_claim(binary, is_subject, facts)
		return
	}
	if binary.Op == token.NEQ {
		numeric_record_claim(binary, is_subject, facts)
		return
	}
}

// Records an Always(v <= C) upper bound guard and its operand name.
func numeric_record_upper(
	binary *ast.BinaryExpr, is_subject numeric_subject, is_always bool,
	facts *numeric_facts,
) {
	if !is_always {
		return
	}
	if !is_subject(binary.X) {
		return
	}
	facts.Has_Upper = true
	facts.Upper_Name = numeric_operand_name(binary.Y)
}

// Records an Always(v >= C) lower bound guard and its operand name.
func numeric_record_lower(
	binary *ast.BinaryExpr, is_subject numeric_subject, is_always bool,
	facts *numeric_facts,
) {
	if !is_always {
		return
	}
	if !is_subject(binary.X) {
		return
	}
	facts.Has_Lower = true
	facts.Lower_Name = numeric_operand_name(binary.Y)
}

// Records a boundary claim: an infinity (float) or an integer/const equality.
func numeric_record_claim(
	binary *ast.BinaryExpr, is_subject numeric_subject, facts *numeric_facts,
) {
	sign := numeric_infinity_sign(binary.X)
	if sign == 0 {
		sign = numeric_infinity_sign(binary.Y)
	}
	if sign < 0 {
		facts.Claimed_Values["-Inf"] = true
		return
	}
	if sign > 0 {
		facts.Claimed_Values["+Inf"] = true
		return
	}
	operand := numeric_other_operand(binary, is_subject)
	if operand == nil {
		return
	}
	label := numeric_value_label(operand)
	if label == "" {
		return
	}
	facts.Claimed_Values[label] = true
}

// Reports the bounds and bound-constant diagnostics for one bundle.
func numeric_bound_diagnostics(
	facts numeric_facts, input *numeric_bundle_input,
) (diags []Diagnostic) {

	position := input.File_Set.Position(input.Bundle.Name.Pos())
	name := input.Bundle.Name.Name
	subject := numeric_subject_text(input)
	incomplete := false
	if !facts.Has_Upper {
		incomplete = true
	}
	if !facts.Has_Lower {
		incomplete = true
	}
	if incomplete {
		diags = append(diags, Diagnostic{Position: position,
			Message: name + " must guard both ends: Always(" + subject +
				" <= MAX) and Always(" + subject + " >= MIN)"})
	}
	if facts.Has_Upper {
		if !numeric_is_package_constant(facts.Upper_Name, input.Constants) {
			diags = append(diags, Diagnostic{Position: position,
				Message: name + " upper bound must be a package-level constant"})
		}
	}
	if facts.Has_Lower {
		if !numeric_is_package_constant(facts.Lower_Name, input.Constants) {
			diags = append(diags, Diagnostic{Position: position,
				Message: name + " lower bound must be a package-level constant"})
		}
	}
	return diags
}

// Reports the boundary-coverage diagnostics for one bundle: a claim for each
// required interior value. The MAX/MIN bound itself is the Always(<=)/Always(>=)
// guard (see numeric_bound_diagnostics), not a boundary claim — claiming it as
// Always(!= bound) is a loophole and witnessing it as Sometimes(== bound) forces
// allocating the max, so neither is required here.
func numeric_coverage_diagnostics(
	facts numeric_facts, input *numeric_bundle_input,
) (diags []Diagnostic) {

	for _, label := range numeric_required_labels(input.Kind) {
		if facts.Claimed_Values[label] {
			continue
		}
		diags = append(diags, numeric_missing_claim(label, input))
	}
	return diags
}

// Builds the diagnostic for a boundary value the bundle never claims.
func numeric_missing_claim(label string, input *numeric_bundle_input) (diag Diagnostic) {
	subject := numeric_subject_text(input)
	return Diagnostic{
		Position: input.File_Set.Position(input.Bundle.Name.Pos()),
		Message: input.Bundle.Name.Name + " must claim " + label +
			" via Sometimes(" + subject + " == " + label + ") or Always(" +
			subject + " ==/!= " + label + ")",
	}
}

// Returns the boundary values a bundle of the given kind must claim.
func numeric_required_labels(kind string) (labels []string) {
	if kind == "float" {
		return []string{"NaN", "-Inf", "+Inf"}
	}
	if kind == "signed" {
		return []string{"0", "1", "-1", "2"}
	}
	return []string{"0", "1", "2"}
}

// Reports whether name is a known package-level constant.
func numeric_is_package_constant(name string, constants map[string]bool) (yes bool) {
	if name == "" {
		return false
	}
	return constants[name]
}

// Reports whether operand is the value identifier.
func numeric_is_value(operand ast.Expr, value string) (yes bool) {
	identifier, is_identifier := operand.(*ast.Ident)
	if !is_identifier {
		return false
	}
	return identifier.Name == value
}

// Returns operand's identifier name, or "" when it is not a bare identifier (a
// literal or selector is therefore never accepted as a bound constant).
func numeric_operand_name(operand ast.Expr) (name string) {
	identifier, is_identifier := operand.(*ast.Ident)
	if !is_identifier {
		return ""
	}
	return identifier.Name
}

// Returns the comparison operand that is not the subject, or nil when neither is.
func numeric_other_operand(
	binary *ast.BinaryExpr, is_subject numeric_subject,
) (operand ast.Expr) {
	if is_subject(binary.X) {
		return binary.Y
	}
	if is_subject(binary.Y) {
		return binary.X
	}
	return nil
}

// Maps a claimed operand to its canonical label: 0/1/2 literals, -1, or a bare
// identifier's own name (a const claim); "" for anything else.
func numeric_value_label(operand ast.Expr) (label string) {
	basic, is_basic := operand.(*ast.BasicLit)
	if is_basic {
		return numeric_basic_label(basic)
	}
	unary, is_unary := operand.(*ast.UnaryExpr)
	if is_unary {
		return numeric_unary_label(unary)
	}
	identifier, is_identifier := operand.(*ast.Ident)
	if is_identifier {
		return identifier.Name
	}
	return ""
}

// Maps an integer literal 0, 1, or 2 to its label; "" otherwise.
func numeric_basic_label(basic *ast.BasicLit) (label string) {
	if basic.Kind != token.INT {
		return ""
	}
	switch basic.Value {
	case "0":
		return "0"
	case "1":
		return "1"
	case "2":
		return "2"
	default:
		return ""
	}
}

// Maps the literal -1 (a unary minus over 1) to its label; "" otherwise.
func numeric_unary_label(unary *ast.UnaryExpr) (label string) {
	if unary.Op != token.SUB {
		return ""
	}
	basic, is_basic := unary.X.(*ast.BasicLit)
	if !is_basic {
		return ""
	}
	if basic.Kind != token.INT {
		return ""
	}
	if basic.Value != "1" {
		return ""
	}
	return "-1"
}

// Returns -1 for math.Inf(-1), +1 for math.Inf(1), 0 for anything else.
func numeric_infinity_sign(operand ast.Expr) (sign int) {
	call, is_call := operand.(*ast.CallExpr)
	if !is_call {
		return 0
	}
	if numeric_math_member(call.Fun) != "Inf" {
		return 0
	}
	if len(call.Args) != 1 {
		return 0
	}
	if numeric_value_label(call.Args[0]) == "-1" {
		return -1
	}
	if numeric_value_label(call.Args[0]) == "1" {
		return 1
	}
	return 0
}

// Reports whether condition is a math.IsNaN(...) call.
func numeric_nan_call(condition ast.Expr) (yes bool) {
	call, is_call := condition.(*ast.CallExpr)
	if !is_call {
		return false
	}
	return numeric_math_member(call.Fun) == "IsNaN"
}

// Returns the member name when fun is a math.<Member> selector; "" otherwise.
func numeric_math_member(function ast.Expr) (member string) {
	selector, is_selector := function.(*ast.SelectorExpr)
	if !is_selector {
		return ""
	}
	identifier, is_identifier := selector.X.(*ast.Ident)
	if !is_identifier {
		return ""
	}
	if identifier.Name != "math" {
		return ""
	}
	return selector.Sel.Name
}

// Flags a struct bundle that fails to call the _Invariants of a field whose type
// has one. Existence-driven: a field is required only when an _Invariants for its
// type exists in the module (presets always do), so adding one later auto-enables
// the field. A struct with an immediate sync.Mutex/RWMutex field is skipped whole.
// Cross-file because the bundle index spans the whole module.
func check_struct_invariants(parsed_files []parsed_file, exempt []string) (diags []Diagnostic) {
	defined := struct_bundle_index(parsed_files)
	for _, pf := range parsed_files {
		if strings.HasSuffix(pf.Path, "_test.go") {
			continue
		}
		if source.Path_Matches_Glob(pf.Path, exempt) {
			continue
		}
		diags = append(diags, struct_file_diagnostics(pf, defined)...)
	}
	return diags
}

// Collects the base names of every bundle defined in the module's non-test files,
// so a field whose type has gained one is recognized without cross-package resolution.
func struct_bundle_index(parsed_files []parsed_file) (defined map[string]bool) {
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
				defined[function.Name.Name] = true
			}
		}
	}
	return defined
}

// Checks every struct type + value/pointer-parameter bundle pair in one file.
func struct_file_diagnostics(file parsed_file, defined map[string]bool) (diags []Diagnostic) {
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
		diags = append(diags, struct_type_diagnostics(&struct_type_input{
			File: file, Index: index, Type: type_specification,
			Struct: struct_type, Defined: defined,
		})...)
	}
	return diags
}

// Carries one struct and the module bundle index; the parsed file and the map
// keep it off loose parameters.
type struct_type_input struct {
	File    parsed_file
	Index   int
	Type    *ast.TypeSpec
	Struct  *ast.StructType
	Defined map[string]bool
}

// Checks that one struct's bundle composes every coverable field.
func struct_type_diagnostics(input *struct_type_input) (diags []Diagnostic) {
	bundle := type_invariants_following_function(input.File.File, input.Index)
	if bundle == nil {
		return nil
	}
	if bundle.Name.Name != source.Invariant_Name(input.Type.Name.Name) {
		return nil
	}
	parameter := struct_parameter_name(bundle, input.Type.Name.Name)
	if parameter == "" {
		return nil
	}
	present := struct_present_calls(bundle, parameter)
	for _, field := range input.Struct.Fields.List {
		diags = append(diags, struct_field_diagnostics(&struct_field_input{
			Field:           field,
			Type_Parameters: struct_type_parameter_set(input.Type),
			Defined:         input.Defined,
			Present:         present,
			Parameter:       parameter,
			Bundle:          bundle.Name.Name,
			Position:        input.File.File_Set.Position(bundle.Name.Pos()),
		})...)
	}
	return diags
}

// Carries one field and everything its check reads; three maps keep it off loose
// parameters.
type struct_field_input struct {
	Field           *ast.Field
	Type_Parameters map[string]bool
	Defined         map[string]bool
	Present         map[string]bool
	Parameter       string
	Bundle          string
	Position        token.Position
}

// Reports the missing composition call for one field, per declared name.
func struct_field_diagnostics(input *struct_field_input) (diags []Diagnostic) {
	if len(input.Field.Names) == 0 {
		return nil
	}
	expected, preset := struct_field_invariant(input.Field.Type, input.Type_Parameters)
	if expected == "" {
		return nil
	}
	if !preset {
		if !input.Defined[expected] {
			return nil
		}
	}
	for _, name := range input.Field.Names {
		if input.Present[expected+"\x00"+name.Name] {
			continue
		}
		diags = append(diags, Diagnostic{
			Position: input.Position,
			Message: input.Bundle + " must call " + expected + "(" +
				input.Parameter + "." + name.Name + ", ...)",
		})
	}
	return diags
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
	if numeric_type_base_name(parameter_type) != type_name {
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
func struct_present_calls(bundle *ast.FuncDecl, parameter string) (present map[string]bool) {
	present = map[string]bool{}
	ast.Inspect(bundle.Body, func(node ast.Node) (recurse bool) {
		call, is_call := node.(*ast.CallExpr)
		if !is_call {
			return true
		}
		callee := struct_callee_name(call.Fun)
		if callee == "" {
			return true
		}
		field := struct_first_argument_field(call, parameter)
		if field == "" {
			return true
		}
		present[callee+"\x00"+field] = true
		return true
	})
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
	field_type ast.Expr, type_parameters map[string]bool,
) (name string, preset bool) {

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
		return typed.Sel.Name + "_Invariants", false
	case *ast.IndexExpr:
		return struct_named_invariant(typed.X, type_parameters)
	case *ast.IndexListExpr:
		return struct_named_invariant(typed.X, type_parameters)
	case *ast.Ident:
		return struct_field_ident_invariant(typed.Name, type_parameters)
	default:
		return "", false
	}
}

// Returns the bundle name for a generic instantiation's base (an ident or a
// cross-package selector), or "" otherwise.
func struct_named_invariant(
	base ast.Expr, type_parameters map[string]bool,
) (name string, preset bool) {

	selector, is_selector := base.(*ast.SelectorExpr)
	if is_selector {
		return selector.Sel.Name + "_Invariants", false
	}
	identifier, is_identifier := base.(*ast.Ident)
	if is_identifier {
		return struct_field_ident_invariant(identifier.Name, type_parameters)
	}
	return "", false
}

// Maps a field ident to its expected bundle: a struct type param is exempt, a
// primitive maps to its preset, a no-preset builtin is exempt, else it is a
// defined type whose own bundle (by casing) is expected.
func struct_field_ident_invariant(
	name string, type_parameters map[string]bool,
) (invariant_name string, preset bool) {

	if type_parameters[name] {
		return "", false
	}
	if name == "string" {
		// A raw string field is banned by check_primitive_types, not composed here.
		return "", false
	}
	mapped := struct_primitive_preset(name)
	if mapped != "" {
		return mapped, true
	}
	if struct_is_builtin(name) {
		return "", false
	}
	return source.Invariant_Name(name), false
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

// Flags an ordinary function that fails to assert an input parameter or a named
// return value. Each subject must have its type's _Invariants called on it
// (existence-driven, like the struct rule); named returns are asserted in a
// first-statement defer, inputs in the leading block right after it. Cross-file
// because the bundle index spans the whole module.
func check_function_invariants(parsed_files []parsed_file, exempt []string) (diags []Diagnostic) {
	defined := struct_bundle_index(parsed_files)
	for _, pf := range parsed_files {
		if strings.HasSuffix(pf.Path, "_test.go") {
			continue
		}
		if source.Path_Matches_Glob(pf.Path, exempt) {
			continue
		}
		diags = append(diags, function_file_diagnostics(pf, defined)...)
	}
	return diags
}

// Checks every named free function in one file.
func function_file_diagnostics(file parsed_file, defined map[string]bool) (diags []Diagnostic) {
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
		diags = append(diags, function_diagnostics(&function_input{
			File: file, Function: function, Defined: defined,
		})...)
	}
	return diags
}

// Carries one function and the module bundle index.
type function_input struct {
	File     parsed_file
	Function *ast.FuncDecl
	Defined  map[string]bool
}

// Carries the two name maps a requirement derivation reads, so they stay off
// loose parameters.
type function_scope struct {
	Type_Parameters map[string]bool
	Defined         map[string]bool
}

// One subject (param or named return) and the assertion it must carry: a flat
// call, or an element-wise range loop for a slice.
type assertion_requirement struct {
	Subject  string
	Expected string
	Loop     bool
}

// Collects the assertion gaps for one function's inputs and outputs.
func function_diagnostics(input *function_input) (diags []Diagnostic) {
	scope := &function_scope{
		Type_Parameters: function_type_parameter_set(input.Function),
		Defined:         input.Defined,
	}
	inputs := function_requirements(input.Function.Type.Params, scope)
	outputs := function_requirements(input.Function.Type.Results, scope)
	if len(inputs) == 0 {
		if len(outputs) == 0 {
			return nil
		}
	}
	position := input.File.File_Set.Position(input.Function.Name.Pos())
	name := input.Function.Name.Name
	lead, defer_body, has_defer := function_lead_and_defer(input.Function.Body.List)
	for _, requirement := range outputs {
		if !has_defer {
			diags = append(diags, Diagnostic{Position: position,
				Message: name + " must assert " + requirement.Subject +
					" in a first-statement defer"})
			continue
		}
		if !function_requirement_met(defer_body, requirement) {
			diags = append(diags, Diagnostic{Position: position,
				Message: name + " must assert " + requirement.Subject +
					" in the output defer"})
		}
	}
	for _, requirement := range inputs {
		if function_requirement_met(lead, requirement) {
			continue
		}
		diags = append(diags, Diagnostic{Position: position,
			Message: name + " must assert " + requirement.Subject + " via " +
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

// Builds the assertion requirements for a parameter or result list, skipping
// blank and exempt subjects.
func function_requirements(
	fields *ast.FieldList, scope *function_scope,
) (requirements []assertion_requirement) {

	if fields == nil {
		return nil
	}
	for _, field := range fields.List {
		expected, loop, required := function_requirement(field.Type, scope)
		if !required {
			continue
		}
		for _, name := range field.Names {
			if name.Name == "_" {
				continue
			}
			requirements = append(requirements, assertion_requirement{
				Subject: name.Name, Expected: expected, Loop: loop,
			})
		}
	}
	return requirements
}

// Derives one subject's requirement: a defined type asks for a flat _Invariants
// call. A raw slice, variadic, or map is banned by check_primitive_types rather
// than asserted here, so it carries no requirement.
func function_requirement(
	field_type ast.Expr, scope *function_scope,
) (expected string, loop bool, required bool) {

	core := field_type
	star, is_star := core.(*ast.StarExpr)
	if is_star {
		core = star.X
	}
	if _, is_array := core.(*ast.ArrayType); is_array {
		return "", false, false
	}
	if _, is_ellipsis := core.(*ast.Ellipsis); is_ellipsis {
		return "", false, false
	}
	if _, is_map := core.(*ast.MapType); is_map {
		return "", false, false
	}
	flat, flat_required := function_named_invariant(core, scope)
	return flat, false, flat_required
}

// Maps a named type expression (ident, selector, pointer, or generic
// instantiation) to its _Invariants name and whether one exists.
func function_named_invariant(
	type_expression ast.Expr, scope *function_scope,
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
		name := selector.Sel.Name + "_Invariants"
		return name, scope.Defined[name]
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
		return preset, true
	}
	if struct_is_builtin(identifier.Name) {
		return "", false
	}
	name := source.Invariant_Name(identifier.Name)
	return name, scope.Defined[name]
}

// Splits a body into the leading assertion block and the first-statement defer's
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
		if !function_is_assertion_statement(statement) {
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

// Reports whether a statement is an assertion: an _Invariants call, or a range
// loop whose body is only _Invariants calls.
func function_is_assertion_statement(statement ast.Stmt) (yes bool) {
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
	expression_statement, is_expression := statement.(*ast.ExprStmt)
	if !is_expression {
		return false
	}
	call, is_call := expression_statement.X.(*ast.CallExpr)
	if !is_call {
		return false
	}
	return type_invariants_is_bundle_name(struct_callee_name(call.Fun))
}

// Reports whether the statements satisfy one requirement.
func function_requirement_met(
	statements []ast.Stmt, requirement assertion_requirement,
) (met bool) {

	if requirement.Loop {
		return function_loop_asserts(statements, requirement)
	}
	return function_flat_asserts(statements, requirement)
}

// Reports whether some statement is a flat Expected(subject, …) call.
func function_flat_asserts(
	statements []ast.Stmt, requirement assertion_requirement,
) (met bool) {

	for _, statement := range statements {
		found := false
		ast.Inspect(statement, func(node ast.Node) (recurse bool) {
			call, is_call := node.(*ast.CallExpr)
			if !is_call {
				return true
			}
			if struct_callee_name(call.Fun) != requirement.Expected {
				return true
			}
			if function_first_argument_name(call) != requirement.Subject {
				return true
			}
			found = true
			return false
		})
		if found {
			return true
		}
	}
	return false
}

// Reports whether some statement is a `range subject` loop asserting each element
// with Expected.
func function_loop_asserts(
	statements []ast.Stmt, requirement assertion_requirement,
) (met bool) {

	for _, statement := range statements {
		range_statement, is_range := statement.(*ast.RangeStmt)
		if !is_range {
			continue
		}
		subject, is_subject := range_statement.X.(*ast.Ident)
		if !is_subject {
			continue
		}
		if subject.Name != requirement.Subject {
			continue
		}
		value, is_value := range_statement.Value.(*ast.Ident)
		if !is_value {
			continue
		}
		if function_block_asserts(range_statement.Body, requirement, value.Name) {
			return true
		}
	}
	return false
}

// Reports whether a range body calls the requirement's expected invariant on the
// loop's value identifier.
func function_block_asserts(
	block *ast.BlockStmt, requirement assertion_requirement, value string,
) (yes bool) {

	for _, statement := range block.List {
		expression_statement, is_expression := statement.(*ast.ExprStmt)
		if !is_expression {
			continue
		}
		call, is_call := expression_statement.X.(*ast.CallExpr)
		if !is_call {
			continue
		}
		if struct_callee_name(call.Fun) != requirement.Expected {
			continue
		}
		if function_first_argument_name(call) != value {
			continue
		}
		return true
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

// Renders the expected assertion form for a diagnostic.
func function_form(requirement assertion_requirement) (form string) {
	if requirement.Loop {
		return "for _, x := range " + requirement.Subject + " { " +
			requirement.Expected + "(x, ...) }"
	}
	return requirement.Expected + "(" + requirement.Subject + ", ...)"
}

// Flags every non-exempt, non-main package whose test binary fails to wire the
// invariant coverage recorder. invariant.Run_Test_Main is the canonical TestMain
// body; without it a package's mandated _Invariants bundles run but their
// Sometimes axes and Always reachability are never verified — the discipline
// silently evaporates. Package-level because the TestMain may live in any of the
// directory's test files, so the whole directory is judged together. Shares the
// type-invariant rule's opt-out.
func check_recorder_test_main(
	parsed_files []parsed_file, components *component_index, exempt []string,
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
type recorder_group struct {
	Directory           string
	Any_Path            string
	Is_Main             bool
	Has_Test            bool
	Source_Anchor       token.Position
	Test_Anchor         token.Position
	Test_Main_Found     bool
	Test_Main_Canonical bool
	Test_Main_Position  token.Position
}

// Buckets parsed files by directory, preserving first-seen order (parsed_files is
// path-sorted) so diagnostics are deterministic without a separate sort.
func recorder_test_main_groups(parsed_files []parsed_file) (groups []*recorder_group) {
	index := map[string]*recorder_group{}
	for _, pf := range parsed_files {
		directory := path.Dir(pf.Path)
		group := index[directory]
		if group == nil {
			group = &recorder_group{Directory: directory}
			index[directory] = group
			groups = append(groups, group)
		}
		recorder_group_absorb(group, pf)
	}
	return groups
}

// Folds one file's facts into its directory group: a test file may carry the
// TestMain and is the preferred anchor; a source file marks the main package.
func recorder_group_absorb(group *recorder_group, file parsed_file) {
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
func recorder_group_scan_test_main(group *recorder_group, file parsed_file) {
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
func recorder_group_diagnostics(group *recorder_group) (diags []Diagnostic) {
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

// Simulation_directory names the test-only package under a binary component's
// internal/ whose fuzz test drives internal.Main.
const simulation_directory = "simulation_test"

// Simulation_glob is the sole TestMain directory argument: the internal package and
// every package beneath it, registered in one pattern since the recorder recurses on **.
const simulation_glob = "../**"

// A binary component's invariants are witnessed only by a simulation package that
// drives internal.Main through a fuzz test, never by a per-package Run_Test_Main.
// This holds that package to its contract: it exists, declares nothing but the
// fuzz driver and its TestMain, and registers every non-exempt internal package
// for coverage. Every diagnostic is tier two, so a tier-one issue suppresses it.
func check_simulation(
	parsed_files []parsed_file, components *component_index, exempt []string,
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
	parsed_files []parsed_file, components *component_index,
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
	sim_directory := internal_root + "/" + simulation_directory
	sim_files := simulation_package_files(parsed_files, sim_directory)
	if len(sim_files) == 0 {
		return simulation_diagnostic(position, "binary component "+
			strconv.Quote(component.Import_Path)+" must declare an internal/"+
			simulation_directory+" package driving internal.Main")
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
	parsed_files []parsed_file, components *component_index,
	component_index_number int, exempt []string, internal_root string,
) (dirs []string) {
	sim_directory := internal_root + "/" + simulation_directory
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
	parsed_files []parsed_file, sim_directory string,
) (files []parsed_file) {
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
	sim_files []parsed_file, position token.Position,
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
	files []parsed_file, position token.Position,
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
	parsed_files []parsed_file, components *component_index,
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
	sim_files []parsed_file, internal_functions map[string]bool,
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
	files []parsed_file, position token.Position,
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
			strconv.Quote(simulation_glob)+")")
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
	return directories[0] == simulation_glob
}

// The first TestMain with a *testing.M parameter among the simulation files.
func simulation_find_test_main(files []parsed_file) (function *ast.FuncDecl) {
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
func check_primitive_types(parsed_files []parsed_file, exempt []string) (diags []Diagnostic) {
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
func primitive_file_diagnostics(file parsed_file) (diags []Diagnostic) {
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
	file parsed_file, function *ast.FuncDecl,
) (diags []Diagnostic) {

	if source.Method_Satisfies_Stdlib(function) {
		return nil
	}
	position := file.File_Set.Position(function.Name.Pos())
	diags = append(diags, primitive_field_diagnostics(&primitive_field_input{
		Fields: function.Type.Params, Role: "parameter",
		Owner: function.Name.Name, Position: position,
	})...)
	diags = append(diags, primitive_field_diagnostics(&primitive_field_input{
		Fields: function.Type.Results, Role: "result",
		Owner: function.Name.Name, Position: position,
	})...)
	return diags
}

// Flags each struct type's raw string/slice/map fields.
func primitive_struct_diagnostics(file parsed_file, general *ast.GenDecl) (diags []Diagnostic) {
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
		diags = append(diags, primitive_field_diagnostics(&primitive_field_input{
			Fields: struct_type.Fields, Role: "field",
			Owner:    type_specification.Name.Name,
			Position: file.File_Set.Position(type_specification.Name.Pos()),
		})...)
	}
	return diags
}

// Carries one field list and how to name its diagnostics; a struct keeps the role
// and owner off loose string parameters.
type primitive_field_input struct {
	Fields   *ast.FieldList
	Role     string
	Owner    string
	Position token.Position
}

// Flags each field in the list whose type is a raw string, slice, or map.
func primitive_field_diagnostics(input *primitive_field_input) (diags []Diagnostic) {
	if input.Fields == nil {
		return nil
	}
	for _, field := range input.Fields.List {
		kind := numeric_raw_primitive_kind(field.Type)
		if kind == "" {
			continue
		}
		for _, identifier := range primitive_field_names(field) {
			diags = append(diags, Diagnostic{
				Position: input.Position,
				Message: input.Owner + ": raw " + kind + " " + input.Role + " " +
					identifier + "; wrap it in a defined type",
			})
		}
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
