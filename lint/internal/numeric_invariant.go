package lint

import (
	"go/ast"
	"go/token"
	"path"
	"strings"
)

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
		if type_invariants_path_exempt(pf.Path, exempt) {
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
	if candidate.Name.Name != type_invariant_name(input.Type.Name.Name) {
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
	Has_Upper      bool
	Has_Lower      bool
	Upper_Name     string
	Lower_Name     string
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
// required value, plus the MAX/MIN claims tied to the bound constants (non-float).
func numeric_coverage_diagnostics(
	facts numeric_facts, input *numeric_bundle_input,
) (diags []Diagnostic) {

	for _, label := range numeric_required_labels(input.Kind) {
		if facts.Claimed_Values[label] {
			continue
		}
		diags = append(diags, numeric_missing_claim(label, input))
	}
	if input.Kind == "float" {
		return diags
	}
	diags = append(diags, numeric_limit_claim(facts, facts.Upper_Name, input)...)
	diags = append(diags, numeric_limit_claim(facts, facts.Lower_Name, input)...)
	return diags
}

// Reports the missing-claim diagnostic for a bound constant the bundle never
// claims; silent when the bound is not a known package constant.
func numeric_limit_claim(
	facts numeric_facts, name string, input *numeric_bundle_input,
) (diags []Diagnostic) {

	if !numeric_is_package_constant(name, input.Constants) {
		return nil
	}
	if facts.Claimed_Values[name] {
		return nil
	}
	return []Diagnostic{numeric_missing_claim(name, input)}
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
		if type_invariants_path_exempt(pf.Path, exempt) {
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
	if bundle.Name.Name != type_invariant_name(input.Type.Name.Name) {
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

	// A pointer field is optional: it may be nil, so a straight-line bundle cannot
	// unconditionally compose it (the invariant recorder bans the guarding if). Its
	// present-only properties belong in an Imply, not a mandatory composition call.
	_, is_pointer := field_type.(*ast.StarExpr)
	if is_pointer {
		return "", false
	}
	switch typed := field_type.(type) {
	case *ast.ArrayType:
		if typed.Len != nil {
			return "", false
		}
		return "Slice_Invariants", true
	case *ast.MapType:
		return "Map_Invariants", true
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
	mapped := struct_primitive_preset(name)
	if mapped != "" {
		return mapped, true
	}
	if struct_is_builtin(name) {
		return "", false
	}
	return type_invariant_name(name), false
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
	case "string":
		return "String_Invariants"
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
		if type_invariants_path_exempt(pf.Path, exempt) {
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

// Derives one subject's requirement: a slice/variadic asks for the element type's
// _Invariants in a loop, a map for Map_Invariants, anything else for a flat call.
func function_requirement(
	field_type ast.Expr, scope *function_scope,
) (expected string, loop bool, required bool) {

	core := field_type
	star, is_star := core.(*ast.StarExpr)
	if is_star {
		core = star.X
	}
	array, is_array := core.(*ast.ArrayType)
	if is_array {
		element, element_required := function_named_invariant(array.Elt, scope)
		return element, true, element_required
	}
	ellipsis, is_ellipsis := core.(*ast.Ellipsis)
	if is_ellipsis {
		element, element_required := function_named_invariant(ellipsis.Elt, scope)
		return element, true, element_required
	}
	_, is_map := core.(*ast.MapType)
	if is_map {
		return "Map_Invariants", false, true
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
	preset := struct_primitive_preset(identifier.Name)
	if preset != "" {
		return preset, true
	}
	if struct_is_builtin(identifier.Name) {
		return "", false
	}
	name := type_invariant_name(identifier.Name)
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
func check_recorder_test_main(parsed_files []parsed_file, exempt []string) (diags []Diagnostic) {
	for _, group := range recorder_test_main_groups(parsed_files) {
		if type_invariants_path_exempt(group.Directory, exempt) {
			continue
		}
		// A main package holds the binary's wiring, not testable invariant logic.
		if group.Is_Main {
			continue
		}
		diags = append(diags, recorder_group_diagnostics(group)...)
	}
	return diags
}

// One directory's recorder-relevant facts, gathered across all its files.
type recorder_group struct {
	Directory           string
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
