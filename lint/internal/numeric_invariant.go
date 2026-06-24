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
