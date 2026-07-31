package assertion_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"local/james-orcales/lint/internal/assertion"
	"local/james-orcales/lint/internal/diagnostic"
	"local/james-orcales/lint/internal/source"
)

// Test_Invariants_Presence verifies an in-scope type with no bundle function
// directly below it is flagged.
func Test_Invariants_Presence(t *testing.T) {
	t.Parallel()
	pf := parse(t, &parse_input{
		Path: "pkg/rule.go",
		Source_Text: "package fixture\n\n" +
			"// Widget is a fixture.\n" +
			"type Widget struct {\n\t// X is a fixture.\n\tX int\n}\n"})
	diags := assertion.Check_Type(pf.File_Set, pf.File, nil)
	if !diagnosed(diags, "directly below Widget") {
		t.Fatal("a type without its invariant must be flagged")
	}
}

// Test_Invariants_Casing verifies a wrongly-cased bundle name does not satisfy an
// exported type, which still wants the _Invariants suffix.
func Test_Invariants_Casing(t *testing.T) {
	t.Parallel()
	pf := parse(t, &parse_input{
		Path: "pkg/rule.go",
		Source_Text: "package fixture\n\n" +
			"// Widget is a fixture.\ntype Widget struct {\n" +
			"\t// X is a fixture.\n\tX int\n}\n\n" +
			"// Widget_invariants is wrongly cased.\n" +
			"func Widget_invariants(w Widget) {\n\tprintln(0)\n}\n"})
	diags := assertion.Check_Type(pf.File_Set, pf.File, nil)
	if !diagnosed(diags, "declare Widget_Invariants") {
		t.Fatal("an exported type wants the _Invariants suffix")
	}
}

// Test_Invariants_Signature verifies a bundle missing its invariant.Namespace
// parameter is flagged.
func Test_Invariants_Signature(t *testing.T) {
	t.Parallel()
	pf := parse(t, &parse_input{
		Path: "pkg/rule.go",
		Source_Text: "package fixture\n\n" +
			"// Widget is a fixture.\ntype Widget struct {\n" +
			"\t// X is a fixture.\n\tX int\n}\n\n" +
			"// Widget_Invariants is a fixture.\n" +
			"func Widget_Invariants(w Widget) {\n\tprintln(0)\n}\n"})
	diags := assertion.Check_Type(pf.File_Set, pf.File, nil)
	if !diagnosed(diags, "must take") {
		t.Fatal("a bundle without the namespace parameter must be flagged")
	}
}

// Test_Invariants_Orphan verifies a bundle-named function not declared below its
// type is flagged.
func Test_Invariants_Orphan(t *testing.T) {
	t.Parallel()
	pf := parse(t, &parse_input{
		Path: "pkg/rule.go",
		Source_Text: "package fixture\n\n" +
			"// Stray_Invariants is a fixture.\n" +
			"func Stray_Invariants() {\n\tprintln(0)\n}\n"})
	diags := assertion.Check_Type(pf.File_Set, pf.File, nil)
	if !diagnosed(diags, "directly below its type") {
		t.Fatal("an orphan invariant function must be flagged")
	}
}

// Test_Invariants_Scope verifies a defined scalar type, not just a struct, is in
// scope and flagged when it lacks a bundle.
func Test_Invariants_Scope(t *testing.T) {
	t.Parallel()
	pf := parse(t, &parse_input{
		Path: "pkg/rule.go",
		Source_Text: "package fixture\n\n" +
			"// Count is a fixture.\ntype Count uint64\n"})
	diags := assertion.Check_Type(pf.File_Set, pf.File, nil)
	if !diagnosed(diags, "directly below Count") {
		t.Fatal("a defined scalar type is in scope")
	}
	if name := assertion_input_struct(t); name != "" {
		t.Fatalf("argument-bundling struct remains: %s", name)
	}
}

// Test_Invariants_Scalar_Helper verifies raw assertions cannot replace the canonical primitive,
// Range, or Enum helper required by a defined scalar.
func Test_Invariants_Scalar_Helper(t *testing.T) {
	t.Parallel()
	raw := integer_helper_source("\tinvariant.Always(value >= Value_Min, \"min\")\n" +
		"\tinvariant.Always(value <= Value_Max, \"max\")\n" +
		"\tinvariant.Assertions(namespace).Sometimes(value == Value_Min, \"min\")." +
		"Sometimes(value == Value_Max, \"max\").Ensure()")
	if !diagnosed(check_fixture(t, raw), "must call a canonical helper") {
		t.Fatal("individual scalar assertions must not satisfy the helper mandate")
	}
	range_body := "\tinvariant.Assertions(namespace)." +
		"Range_Int(int(value), int(Value_Min), int(Value_Max)).Ensure()"
	if diagnosed(check_fixture(t, integer_helper_source(range_body)),
		"must call a canonical helper") {
		t.Fatal("the exact Range helper must satisfy the scalar mandate")
	}
	scalar_range_holed_helper(t)
	enum_body := "\tinvariant.Assertions(namespace)." +
		"Enum_Int(int(value), int(Value_Min), int(Value_Max)).Ensure()"
	if diagnosed(check_fixture(t, integer_helper_source(enum_body)),
		"must call a canonical helper") {
		t.Fatal("the exact Enum helper must satisfy the scalar mandate")
	}
	enum_3_body := "\tinvariant.Assertions(namespace)." +
		"Enum_3_Int(int(value), int(Value_Min), int(Value_Third), int(Value_Max)).Ensure()"
	if diagnosed(check_fixture(t, integer_helper_source(enum_3_body)),
		"must call a canonical helper") {
		t.Fatal("the exact Enum_3 helper must satisfy the scalar mandate")
	}
	enum_4_body := "\tinvariant.Assertions(namespace)." +
		"Enum_4_Int(int(value), int(Value_Min), int(Value_Third), " +
		"int(Value_Fourth), int(Value_Max)).Ensure()"
	if diagnosed(check_fixture(t, integer_helper_source(enum_4_body)),
		"must call a canonical helper") {
		t.Fatal("the exact Enum_4 helper must satisfy the scalar mandate")
	}
	preset := "\tinvariant.Int_Invariants(int(value), namespace)"
	if diagnosed(check_fixture(t, integer_helper_source(preset)),
		"must call a canonical helper") {
		t.Fatal("the exact primitive preset must satisfy the scalar mandate")
	}
	singleton := "\tinvariant.Always(int(value) == int(Value_Min), " +
		"\"The value is the only member.\")"
	if diagnosed(check_fixture(t, integer_helper_source(singleton)),
		"must call a canonical helper") {
		t.Fatal("a direct singleton Always must satisfy the integer mandate")
	}
	float_preset := "\tinvariant.Float64_Invariants(float64(value), namespace)"
	if diagnosed(check_fixture(t, float_helper_source(float_preset)),
		"must call a canonical helper") {
		t.Fatal("the exact float preset must satisfy the scalar mandate")
	}
	boolean := "\tinvariant.Boolean_Invariants(bool(value), namespace)"
	if diagnosed(check_fixture(t, boolean_helper_source(boolean)),
		"must call a canonical helper") {
		t.Fatal("the Boolean helper must satisfy the scalar mandate")
	}
	float_singleton := "\tinvariant.Always(float64(value) == float64(Value_Min), \"only\")"
	if !diagnosed(check_fixture(t, float_helper_source(float_singleton)),
		"must call a canonical helper") {
		t.Fatal("a float singleton must use its primitive preset")
	}
	boolean_singleton := "\tinvariant.Always(bool(value) == true, \"only\")"
	if !diagnosed(check_fixture(t, boolean_helper_source(boolean_singleton)),
		"must call a canonical helper") {
		t.Fatal("a Boolean singleton must use its primitive preset")
	}
}

// Test_Invariants_Count_Helper verifies a counted type requires an ensured Range_Int or Enum_Int
// over its own length, regardless of equivalent individual assertions.
func Test_Invariants_Count_Helper(t *testing.T) {
	t.Parallel()
	raw := "\tinvariant.Always(len(value) >= Value_Min, \"min\")\n" +
		"\tinvariant.Always(len(value) <= Value_Max, \"max\")\n" +
		"\tinvariant.Assertions(namespace).Sometimes(len(value) == Value_Min, \"min\")." +
		"Sometimes(len(value) == Value_Max, \"max\").Ensure()"
	if !diagnosed(check_fixture(t, count_helper_source(raw)),
		"must call Range_Int or Enum_Int") {
		t.Fatal("individual count assertions must not satisfy the helper mandate")
	}
	valid := "\tinvariant.Assertions(namespace)." +
		"Range_Holed_Int(len(value), Value_Min, Value_Max, 1, 2, 2, 2).Ensure()"
	if diagnosed(check_fixture(t, count_helper_source(valid)),
		"must call Range_Int or Enum_Int") {
		t.Fatal("Range_Holed_Int over the counted value must satisfy the mandate")
	}
	wrong_suffix := "\tinvariant.Assertions(namespace)." +
		"Range_Int64(int64(len(value)), int64(Value_Min), int64(Value_Max)).Ensure()"
	if !diagnosed(check_fixture(t, count_helper_source(wrong_suffix)),
		"must call Range_Int or Enum_Int") {
		t.Fatal("a differently typed Range helper must not substitute")
	}
	wrong_subject := "\tinvariant.Assertions(namespace)." +
		"Range_Int(len(value[:0]), Value_Min, Value_Max).Ensure()"
	if !diagnosed(check_fixture(t, count_helper_source(wrong_subject)),
		"must call Range_Int or Enum_Int") {
		t.Fatal("a Range_Int over another subject must not satisfy the mandate")
	}
	split := "\tassertions := invariant.Assertions(namespace)\n" +
		"\tassertions.Range_Int(len(value), Value_Min, Value_Max).Ensure()"
	if !diagnosed(check_fixture(t, count_helper_source(split)),
		"must call Range_Int or Enum_Int") {
		t.Fatal("a split builder must not satisfy the helper mandate")
	}
	singleton := "\tinvariant.Always(len(value) == Value_Min, " +
		"\"The count is the only member.\")"
	if diagnosed(check_fixture(t, count_helper_source(singleton)),
		"must call Range_Int or Enum_Int") {
		t.Fatal("a direct singleton Always must satisfy the count mandate")
	}
}

// Test_Invariants_Helper_Constants verifies canonical helpers retain package-level constant
// identity for Range edges and Enum members.
func Test_Invariants_Helper_Constants(t *testing.T) {
	t.Parallel()
	inline_range := "\tinvariant.Assertions(namespace)." +
		"Range_Int(int(value), int(Value_Min), int(7)).Ensure()"
	if !diagnosed(check_fixture(t, integer_helper_source(inline_range)),
		"arguments must be package-level constants") {
		t.Fatal("an inline Range edge must not satisfy the helper mandate")
	}
	inline_enum := "\tinvariant.Assertions(namespace)." +
		"Enum_Int(int(value), int(Value_Min), int(7)).Ensure()"
	if !diagnosed(check_fixture(t, integer_helper_source(inline_enum)),
		"arguments must be package-level constants") {
		t.Fatal("an inline Enum member must not satisfy the helper mandate")
	}
	converted := "\tinvariant.Assertions(namespace)." +
		"Range_Holed_Int(int(value), int(Value_Min), int(Value_Max), 1, 2, 2, 2).Ensure()"
	if diagnosed(check_fixture(t, integer_helper_source(converted)),
		"arguments must be package-level constants") {
		t.Fatal("exactly converted package constants must satisfy the mandate")
	}
	shadowed := "\tValue_Min := 0\n" +
		"\tinvariant.Assertions(namespace)." +
		"Range_Int(int(value), Value_Min, Value_Max).Ensure()"
	if !diagnosed(check_fixture(t, integer_helper_source(shadowed)),
		"arguments must be package-level constants") {
		t.Fatal("a local shadow of a package constant must not satisfy the mandate")
	}
	inline_singleton := "\tinvariant.Always(int(value) == 1, \"only\")"
	if !diagnosed(check_fixture(t, integer_helper_source(inline_singleton)),
		"arguments must be package-level constants") {
		t.Fatal("an inline singleton member must not satisfy the helper mandate")
	}
}

// Test_Invariants_Cross_Package_Identity proves same-spelled foreign constants and helpers cannot
// satisfy a local helper mandate.
func Test_Invariants_Cross_Package_Identity(t *testing.T) {
	t.Parallel()
	cross_package_constant_isolation(t)
	cross_package_helper_isolation(t)
}

// Test_Invariants_Helper_Identity verifies only a direct builder rooted at the actual invariant
// package and the helper's namespace parameter satisfies the body mandate.
func Test_Invariants_Helper_Identity(t *testing.T) {
	t.Parallel()
	literal := "\tinvariant.Assertions(\"manual\")." +
		"Range_Int(int(value), int(Value_Min), int(Value_Max)).Ensure()"
	if !diagnosed(check_fixture(t, integer_helper_source(literal)),
		"must call a canonical helper") {
		t.Fatal("a literal namespace must not satisfy a helper template")
	}
	nested := "\tif true {\n\t\tinvariant.Assertions(namespace)." +
		"Range_Int(int(value), int(Value_Min), int(Value_Max)).Ensure()\n\t}"
	if !diagnosed(check_fixture(t, integer_helper_source(nested)),
		"must call a canonical helper") {
		t.Fatal("a nested builder must not satisfy the direct helper mandate")
	}
	foreign := foreign_scalar_helper_source()
	if !diagnosed(check_fixture(t, foreign), "must call a canonical helper") {
		t.Fatal("a foreign Assertions lookalike must not satisfy the mandate")
	}
	aliased := aliased_scalar_helper_source()
	if !diagnosed(check_fixture(t, aliased), "must call a canonical helper") {
		t.Fatal("an import alias must not impersonate the literal invariant qualifier")
	}
	parent := strings.Replace(integer_helper_source(
		"\tinvariant.Assertions(namespace).Range_Int("+
			"int(value), int(Value_Min), int(Value_Max)).Ensure()"), "/default", "", 1)
	if !diagnosed(check_fixture(t, parent), "must call a canonical helper") {
		t.Fatal("the pure invariant package must not impersonate its default builder")
	}
	recorder := "\tinvariant.Recorder_Assertions(namespace)." +
		"Range_Int(int(value), int(Value_Min), int(Value_Max)).Ensure()"
	if !diagnosed(check_fixture(t, integer_helper_source(recorder)),
		"must call a canonical helper") {
		t.Fatal("Recorder_Assertions must not satisfy the helper mandate")
	}
	legacy := "\tinvariant.Dot_Product(namespace)." +
		"Range_Int(int(value), int(Value_Min), int(Value_Max)).Ensure()"
	if !diagnosed(check_fixture(t, integer_helper_source(legacy)),
		"must call a canonical helper") {
		t.Fatal("the old Dot_Product root must not satisfy the helper mandate")
	}
	conversion := "\tint := func(Value) int { return 0 }\n" +
		"\tinvariant.Assertions(namespace)." +
		"Range_Int(int(value), Value_Min, Value_Max).Ensure()"
	if !diagnosed(check_fixture(t, integer_helper_source(conversion)),
		"must call a canonical helper") {
		t.Fatal("a local function shadowing the primitive conversion must not substitute")
	}
	count := "\tlen := func(Value) int { return 0 }\n" +
		"\tinvariant.Assertions(namespace)." +
		"Range_Int(len(value), Value_Min, Value_Max).Ensure()"
	if !diagnosed(check_fixture(t, count_helper_source(count)),
		"must call Range_Int or Enum_Int") {
		t.Fatal("a local function shadowing len must not substitute")
	}
	assert_invalid_singleton_helper_identity(t)
}

// Test_Invariants_Builder_Walk verifies the static walk admits its entire supported builder and
// rejects the next expanded link instead of silently letting an oversized builder evade analysis.
func Test_Invariants_Builder_Walk(t *testing.T) {
	t.Parallel()
	at_limit := assertions_builder_body(70)
	if diagnosed(check_fixture(t, integer_helper_source(at_limit)),
		"must call a canonical helper") {
		t.Fatal("a builder with exactly 70 expanded links must satisfy the mandate")
	}
	over_limit := assertions_builder_body(71)
	if !diagnosed(check_fixture(t, integer_helper_source(over_limit)),
		"must call a canonical helper") {
		t.Fatal("a builder with 71 expanded links must exceed the static walk limit")
	}
}

// Test_Invariants_Field_Composition verifies a struct bundle that fails to call a
// field type's _Invariants is flagged, whether the field is a value or a pointer.
func Test_Invariants_Field_Composition(t *testing.T) {
	t.Parallel()
	pf := parse(t, &parse_input{
		Path: "pkg/rule.go",
		Source_Text: "package fixture\n\n" +
			"import (\n\tinvariant \"fixture/shared/invariant/default\"\n" +
			"\tforeign \"fixture/foreign\"\n)\n\n" +
			"const Token_Min = 0\n\nconst Token_Max = 8\n\n" +
			"// Token is a fixture.\ntype Token string\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(v Token, namespace invariant.Namespace) {\n" +
			"\tinvariant.Assertions(namespace)." +
			"Range_Int(len(v), Token_Min, Token_Max).Ensure()\n}\n\n" +
			"// Lexeme is a fixture.\ntype Lexeme struct {\n" +
			"\t// Tok is a fixture.\n\tTok Token\n}\n\n" +
			"// Lexeme_Invariants is a fixture.\n" +
			"func Lexeme_Invariants(v Lexeme, namespace invariant.Namespace) {\n" +
			"\tforeign.Token_Invariants(v.Tok, \"wrong\")\n" +
			"\tif false { Token_Invariants(v.Tok, \"nested\") }\n" +
			"\tToken_Invariants := func(Token, invariant.Namespace) {}\n" +
			"\tToken_Invariants(v.Tok, namespace)\n" +
			"\tinvariant.Assertions(namespace)." +
			"Sometimes(true, \"x\").Ensure()\n}\n\n" +
			"// Phrase is a fixture.\ntype Phrase struct {\n" +
			"\t// Tok is a fixture.\n\tTok *Token\n}\n\n" +
			"// Phrase_Invariants is a fixture.\n" +
			"func Phrase_Invariants(v Phrase, namespace invariant.Namespace) {\n" +
			"\tinvariant.Assertions(namespace)." +
			"Sometimes(true, \"y\").Ensure()\n}\n"})
	diags := check_source(pf)
	if !diagnosed(diags, "Lexeme_Invariants must call Token_Invariants") {
		t.Fatal("a struct that does not compose a value field's invariant must be flagged")
	}
	// A pointer field composes its pointee, the same as a value field of that type,
	// so a bundle that omits it is flagged too.
	if !diagnosed(diags, "Phrase_Invariants must call Token_Invariants") {
		t.Fatal("a struct that does not compose a pointer field's pointee must be flagged")
	}
	external := parse(t, &parse_input{
		Path: "other/token.go",
		Source_Text: "package other\n\n" +
			"import invariant \"fixture/shared/invariant/default\"\n\n" +
			"const Token_Min = 0\n\nconst Token_Max = 8\n\n" +
			"// Token is a fixture.\ntype Token string\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(v Token, namespace invariant.Namespace) {\n" +
			"\tinvariant.Assertions(namespace)." +
			"Range_Int(len(v), Token_Min, Token_Max).Ensure()\n}\n"})
	composed := parse(t, &parse_input{
		Path: "pkg/composed.go",
		Source_Text: "package fixture\n\n" +
			"import (\n\tinvariant \"fixture/shared/invariant/default\"\n" +
			"\texternal \"fixture/other\"\n)\n\n" +
			"// Holder is a fixture.\ntype Holder struct {\n" +
			"\t// Token is external.\n\tToken external.Token\n" +
			"\t// Count is primitive.\n\tCount int\n}\n\n" +
			"// Holder_Invariants is a fixture.\n" +
			"func Holder_Invariants(v Holder, namespace invariant.Namespace) {\n" +
			"\texternal.Token_Invariants(v.Token, \"token\")\n" +
			"\tinvariant.Int_Invariants(v.Count, \"count\")\n}\n"})
	if diagnosed(check_sources([]source.Parsed_File{external, composed}), "must call") {
		t.Fatal("aliased exact cross-package and primitive helpers must compose")
	}
}

// Test_Invariants_Parameter_Helper verifies only the input type's exact helper
// satisfies the leading helper requirement.
func Test_Invariants_Parameter_Helper(t *testing.T) {
	t.Parallel()
	pf := parse(t, &parse_input{
		Path: "pkg/rule.go",
		Source_Text: "package fixture\n\n" +
			"import (\n\tinvariant \"fixture/shared/invariant/default\"\n" +
			"\tforeign \"fixture/foreign\"\n)\n\n" +
			"const Token_Min = 0\n\nconst Token_Max = 8\n\n" +
			"// Token is a fixture.\ntype Token string\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(v Token, namespace invariant.Namespace) {\n" +
			"\tinvariant.Assertions(namespace)." +
			"Range_Int(len(v), Token_Min, Token_Max).Ensure()\n}\n\n" +
			"// Consume does.\nfunc Consume(tok Token) {\n" +
			"\tforeign.Token_Invariants(tok, \"wrong\")\n" +
			"\tinvariant.Always(true, \"raw guard\")\n" +
			"\tinvariant.Assertions(\"raw builder\")." +
			"Sometimes(true, \"raw axis\").Ensure()\n" +
			"\tprintln(0)\n}\n"})
	if !diagnosed(check_source(pf), "must call helper for tok") {
		t.Fatal("foreign and direct assertions must not satisfy the input helper")
	}
	parameter_helper_correct(t)
	parameter_helper_shadowed(t)
	parameter_helper_external(t)
}

// Test_Invariants_Output_Helper verifies only the return type's exact helper
// satisfies the first-statement output defer.
func Test_Invariants_Output_Helper(t *testing.T) {
	t.Parallel()
	pf := parse(t, &parse_input{
		Path: "pkg/rule.go",
		Source_Text: "package fixture\n\n" +
			"import (\n\tinvariant \"fixture/shared/invariant/default\"\n" +
			"\tforeign \"fixture/foreign\"\n)\n\n" +
			"const Token_Min = 0\n\nconst Token_Max = 8\n\n" +
			"// Token is a fixture.\ntype Token string\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(v Token, namespace invariant.Namespace) {\n" +
			"\tinvariant.Assertions(namespace)." +
			"Range_Int(len(v), Token_Min, Token_Max).Ensure()\n}\n\n" +
			"// Make does.\nfunc Make() (tok Token) {\n\tdefer func() {\n" +
			"\t\tforeign.Token_Invariants(tok, \"wrong\")\n" +
			"\t\tif false { Token_Invariants(tok, \"nested\") }\n" +
			"\t\tToken_Invariants := func(Token, invariant.Namespace) {}\n" +
			"\t\tToken_Invariants(tok, \"shadowed\")\n" +
			"\t\tinvariant.Always(true, \"raw guard\")\n\t}()\n\treturn \"\"\n}\n"})
	if !diagnosed(check_source(pf), "must call helper for tok in the output defer") {
		t.Fatal("foreign and direct assertions must not satisfy the output helper")
	}
	correct := parse(t, &parse_input{
		Path: "pkg/correct_output.go",
		Source_Text: "package fixture\n\n" +
			"import invariant \"fixture/shared/invariant/default\"\n\n" +
			"const Token_Min = 0\n\nconst Token_Max = 8\n\n" +
			"// Token is a fixture.\ntype Token string\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(v Token, namespace invariant.Namespace) {\n" +
			"\tinvariant.Assertions(namespace)." +
			"Range_Int(len(v), Token_Min, Token_Max).Ensure()\n}\n\n" +
			"// Make uses the exact output helper.\nfunc Make() (tok Token) {\n" +
			"\tdefer func() {\n\t\tToken_Invariants(tok, \"token\")\n" +
			"\t}()\n\treturn \"\"\n}\n"})
	if diagnosed(check_source(correct), "must call helper") {
		t.Fatal("the exact helper in the first output defer must satisfy the mandate")
	}
}

// Test_Invariants_Subject_Isolation proves one exact call cannot satisfy another same-typed field,
// input, or output subject.
func Test_Invariants_Subject_Isolation(t *testing.T) {
	t.Parallel()
	field_subject_isolation(t)
	parameter_subject_isolation(t)
	output_subject_isolation(t)
}

// Test_Invariants_Recorder_Registration verifies a non-exempt shared-library
// package whose test file declares no TestMain is flagged for missing wiring.
func Test_Invariants_Recorder_Registration(t *testing.T) {
	t.Parallel()
	files := []source.Parsed_File{
		parse(t, &parse_input{
			Path:        "pkg/rule.go",
			Source_Text: "// Package fixture is a fixture.\npackage fixture\n"}),
		parse(t, &parse_input{
			Path: "pkg/rule_test.go",
			Source_Text: "package fixture_test\n\nimport \"testing\"\n\n" +
				"func Test_Widget(t *testing.T) {}\n"}),
	}
	if !diagnosed(recorder_diagnostics(files), "must wire invariant.Run_Test_Main") {
		t.Fatal("a package with no TestMain must be flagged")
	}
}

// Test_Invariants_Primitive_Types verifies a raw string, slice, or map in a
// function signature or struct field is flagged: wrap it in a defined type.
func Test_Invariants_Primitive_Types(t *testing.T) {
	t.Parallel()
	pf := parse(t, &parse_input{
		Path: "pkg/rule.go",
		Source_Text: "package fixture\n\n" +
			"// Greet does.\n" +
			"func Greet(name string) (greeting string) {\n\treturn \"\"\n}\n"})
	diags := check_source(pf)
	if !diagnosed(diags, "raw string parameter") {
		t.Fatal("a raw string parameter must be flagged")
	}
	if !diagnosed(diags, "raw string result") {
		t.Fatal("a raw string result must be flagged")
	}
}

// Test_Simulation_Presence verifies a binary component with a non-exempt internal
// package but no simulation_test package is flagged.
func Test_Simulation_Presence(t *testing.T) {
	t.Parallel()
	if !diagnosed(simulation_diagnostics(simulation_base(t)),
		"must declare an internal/simulation_test") {
		t.Fatal("a binary component without a simulation package must be flagged")
	}
}

// Test_Simulation_Contents verifies a simulation package with no Fuzz function is
// flagged: the fuzz driver must be present.
func Test_Simulation_Contents(t *testing.T) {
	t.Parallel()
	sim := "package simulation_test\n\nimport (\n\t\"testing\"\n\n" +
		"\tinvariant \"fixture/shared/invariant/default\"\n)\n\n" +
		"func TestMain(m *testing.M) {\n" +
		"\tinvariant.Run_Test_Main(m, \"../**\")\n}\n"
	if !diagnosed(simulation_diagnostics(simulation_files(t, sim)),
		"must declare a fuzz function") {
		t.Fatal("a simulation package without a fuzz function must be flagged")
	}
}

// Test_Simulation_Test_Main verifies a simulation TestMain that is not exactly
// invariant.Run_Test_Main(m, <dirs>) is flagged.
func Test_Simulation_Test_Main(t *testing.T) {
	t.Parallel()
	sim := simulation_fixture_source("invariant.Run_Test_Main(m)")
	if !diagnosed(simulation_diagnostics(simulation_files(t, sim)),
		"simulation TestMain must be exactly") {
		t.Fatal("a simulation TestMain without directory arguments must be flagged")
	}
}

// Test_Simulation_Coverage verifies a simulation TestMain whose directory arguments
// do not register every internal package is flagged.
func Test_Simulation_Coverage(t *testing.T) {
	t.Parallel()
	sim := simulation_fixture_source("invariant.Run_Test_Main(m, \"../*\")")
	if !diagnosed(simulation_diagnostics(simulation_files(t, sim)),
		"simulation TestMain must be exactly") {
		t.Fatal("a narrower glob that omits nested internal packages must be flagged")
	}
}

// Test_Simulation_Entry verifies the simulation is flagged when it references an
// internal function other than Main.
func Test_Simulation_Entry(t *testing.T) {
	t.Parallel()
	sim := "package simulation_test\n\nimport (\n\t\"testing\"\n\n" +
		"\t\"github.com/james-orcales/james-orcales/pkg/internal\"\n" +
		"\tinvariant \"fixture/shared/invariant/default\"\n)\n\n" +
		"func TestMain(m *testing.M) {\n" +
		"\tinvariant.Run_Test_Main(m, \"../**\")\n}\n\n" +
		"func Fuzz_Main(f *testing.F) {\n\tinternal.Extra()\n}\n"
	files := []source.Parsed_File{
		parse(t, &parse_input{
			Path: "pkg/main.go",
			Source_Text: "// Package main is a fixture.\n" +
				"package main\n\nfunc main() {}\n"}),
		parse(t, &parse_input{
			Path: "pkg/internal/entry.go",
			Source_Text: "// Package internal is a fixture.\npackage internal\n\n" +
				"// Main is the entry point.\nfunc Main() {}\n\n" +
				"// Extra is a fixture.\nfunc Extra() {}\n"}),
		parse(t, &parse_input{
			Path:        "pkg/internal/simulation_test/sim_test.go",
			Source_Text: sim}),
	}
	if !diagnosed(simulation_diagnostics(files), "simulation may reference only Main") {
		t.Fatal("referencing a non-Main internal function must be flagged")
	}
}

// Test_Simulation_Blackbox verifies a simulation package that is not an external
// _test package is flagged.
func Test_Simulation_Blackbox(t *testing.T) {
	t.Parallel()
	sim := "package simulation\n\nimport (\n\t\"testing\"\n\n" +
		"\tinvariant \"fixture/shared/invariant/default\"\n)\n\n" +
		"func TestMain(m *testing.M) {\n" +
		"\tinvariant.Run_Test_Main(m, \"../**\")\n}\n\n" +
		"func Fuzz_Main(f *testing.F) {\n\tf.Fuzz(func(t *testing.T, data []byte) {})\n}\n"
	if !diagnosed(simulation_diagnostics(simulation_files(t, sim)), "must be blackbox") {
		t.Fatal("a whitebox simulation package must be flagged")
	}
}

func scalar_range_holed_helper(t *testing.T) {
	t.Helper()
	body := "\tinvariant.Assertions(namespace)." +
		"Range_Holed_Int(int(value), int(Value_Min), int(Value_Max), " +
		"int(Value_Third), int(Value_Fourth), int(Value_Fourth), " +
		"int(Value_Fourth)).Ensure()"
	if diagnosed(check_fixture(t, integer_helper_source(body)),
		"must call a canonical helper") {
		t.Fatal("the exact Range_Holed helper must satisfy the scalar mandate")
	}
}

func assert_invalid_singleton_helper_identity(t *testing.T) {
	t.Helper()
	invalid_singletons := []string{
		"\tinvariant.Always(int(Value_Min) == int(value), \"reversed\")",
		"\tinvariant.Always(int(value) != int(Value_Min), \"not equal\")",
		"\tmessage := \"dynamic\"\n" +
			"\tinvariant.Always(int(value) == int(Value_Min), message)",
		"\tinvariant.Always(int(value + 1) == int(Value_Min), \"wrong subject\")",
		"\tif true { invariant.Always(int(value) == int(Value_Min), \"nested\") }",
	}
	for _, body := range invalid_singletons {
		if !diagnosed(check_fixture(t, integer_helper_source(body)),
			"must call a canonical helper") {
			t.Fatalf("noncanonical singleton satisfied the helper mandate: %s", body)
		}
	}
	parent_singleton := strings.Replace(integer_helper_source(
		"\tinvariant.Always(int(value) == int(Value_Min), \"only\")"),
		"/default", "", 1)
	if !diagnosed(check_fixture(t, parent_singleton), "must call a canonical helper") {
		t.Fatal("the pure invariant package must not provide the singleton helper")
	}
	aliased_singleton := strings.ReplaceAll(integer_helper_source(
		"\tinvariant.Always(int(value) == int(Value_Min), \"only\")"),
		"invariant", "contract")
	if !diagnosed(check_fixture(t, aliased_singleton), "must call a canonical helper") {
		t.Fatal("an import alias must not provide the singleton helper")
	}
}

// The passing case stays outside the leaf so each topology remains independently legible while
// the specification leaf retains one mandate and the linter's bounded-function contract.
func parameter_helper_correct(t *testing.T) {
	t.Helper()
	correct := parse(t, &parse_input{
		Path: "pkg/correct.go",
		Source_Text: "package fixture\n\n" +
			"import invariant \"fixture/shared/invariant/default\"\n\n" +
			"const Token_Min = 0\n\nconst Token_Max = 8\n\n" +
			"// Token is a fixture.\ntype Token string\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(v Token, namespace invariant.Namespace) {\n" +
			"\tinvariant.Assertions(namespace)." +
			"Range_Int(len(v), Token_Min, Token_Max).Ensure()\n}\n\n" +
			"// Consume uses exact helpers.\nfunc Consume(tok Token, count int) {\n" +
			"\tToken_Invariants(tok, \"token\")\n" +
			"\tinvariant.Int_Invariants(count, \"count\")\n}\n"})
	if diagnosed(check_source(correct), "must call helper") {
		t.Fatal("exact local and primitive input helpers must satisfy the mandate")
	}
}

// A same-named parameter must not become an escape hatch from exact helper identity.
func parameter_helper_shadowed(t *testing.T) {
	t.Helper()
	shadowed := parse(t, &parse_input{
		Path: "pkg/shadowed.go",
		Source_Text: "package fixture\n\n" +
			"import invariant \"fixture/shared/invariant/default\"\n\n" +
			"const Token_Min = 0\n\nconst Token_Max = 8\n\n" +
			"// Token is a fixture.\ntype Token string\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(v Token, namespace invariant.Namespace) {\n" +
			"\tinvariant.Assertions(namespace)." +
			"Range_Int(len(v), Token_Min, Token_Max).Ensure()\n}\n\n" +
			"// Consume shadows the helper.\n" +
			"func Consume(tok Token, Token_Invariants " +
			"func(Token, invariant.Namespace)) {\n" +
			"\tToken_Invariants(tok, \"shadowed\")\n}\n"})
	if !diagnosed(check_source(shadowed), "must call helper for tok") {
		t.Fatal("a parameter shadowing the exact helper must not satisfy the mandate")
	}
}

// Cross-package identity is proved separately because aliases change spelling without changing
// the helper that owns the external type.
func parameter_helper_external(t *testing.T) {
	t.Helper()
	external := parse(t, &parse_input{
		Path: "other/input.go",
		Source_Text: "package other\n\n" +
			"import invariant \"fixture/shared/invariant/default\"\n\n" +
			"const Input_Min = -4\n\nconst Input_Max = 4\n\n" +
			"// Input is a fixture.\ntype Input int\n\n" +
			"// Input_Invariants is a fixture.\n" +
			"func Input_Invariants(v Input, namespace invariant.Namespace) {\n" +
			"\tinvariant.Assertions(namespace).Range_Int(" +
			"int(v), int(Input_Min), int(Input_Max)).Ensure()\n}\n"})
	consumer := parse(t, &parse_input{
		Path: "pkg/external.go",
		Source_Text: "package fixture\n\n" +
			"import external \"fixture/other\"\n\n" +
			"// Consume uses an aliased external helper.\n" +
			"func Consume(input external.Input) {\n" +
			"\texternal.Input_Invariants(input, \"input\")\n}\n"})
	if diagnosed(check_sources([]source.Parsed_File{external, consumer}), "must call helper") {
		t.Fatal("an aliased exact cross-package input helper must satisfy the mandate")
	}
}

func field_subject_isolation(t *testing.T) {
	t.Helper()
	pf := parse(t, &parse_input{
		Path: "pkg/field_isolation.go",
		Source_Text: "package fixture\n\n" +
			"import invariant \"fixture/shared/invariant/default\"\n\n" +
			"// Token is a fixture.\ntype Token int\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(value Token, namespace invariant.Namespace) {\n" +
			"\tinvariant.Int_Invariants(int(value), namespace)\n}\n\n" +
			"// Pair is a fixture.\ntype Pair struct {\n" +
			"\t// First is a fixture.\n\tFirst Token\n" +
			"\t// Second is a fixture.\n\tSecond Token\n}\n\n" +
			"// Pair_Invariants is a fixture.\n" +
			"func Pair_Invariants(value Pair, namespace invariant.Namespace) {\n" +
			"\tToken_Invariants(value.First, \"first\")\n}\n"})
	diags := check_source(pf)
	if !diagnosed(diags, "Pair_Invariants must call Token_Invariants(value.Second, ...)") {
		t.Fatalf("first field call satisfied the second field: %v", diags)
	}
	if diagnosed(diags, "Token_Invariants(value.First, ...)") {
		t.Fatalf("the exact first field call was rejected: %v", diags)
	}
}

func parameter_subject_isolation(t *testing.T) {
	t.Helper()
	pf := parse(t, &parse_input{
		Path: "pkg/parameter_isolation.go",
		Source_Text: "package fixture\n\n" +
			"import invariant \"fixture/shared/invariant/default\"\n\n" +
			"// Token is a fixture.\ntype Token int\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(value Token, namespace invariant.Namespace) {\n" +
			"\tinvariant.Int_Invariants(int(value), namespace)\n}\n\n" +
			"// Consume is a fixture.\n" +
			"func Consume(first Token, second Token) {\n" +
			"\tToken_Invariants(first, \"first\")\n\tprintln(0)\n}\n"})
	diags := check_source(pf)
	want := "Consume must call helper for second via Token_Invariants(second, ...)"
	if !diagnosed(diags, want) {
		t.Fatalf("first input call satisfied the second input: %v", diags)
	}
	if diagnosed(diags, "helper for first") {
		t.Fatalf("the exact first input call was rejected: %v", diags)
	}
}

func output_subject_isolation(t *testing.T) {
	t.Helper()
	pf := parse(t, &parse_input{
		Path: "pkg/output_isolation.go",
		Source_Text: "package fixture\n\n" +
			"import invariant \"fixture/shared/invariant/default\"\n\n" +
			"// Token is a fixture.\ntype Token int\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(value Token, namespace invariant.Namespace) {\n" +
			"\tinvariant.Int_Invariants(int(value), namespace)\n}\n\n" +
			"// Make is a fixture.\nfunc Make() (first Token, second Token) {\n" +
			"\tdefer func() { Token_Invariants(first, \"first\") }()\n" +
			"\treturn 0, 0\n}\n"})
	diags := check_source(pf)
	if !diagnosed(diags, "Make must call helper for second in the output defer") {
		t.Fatalf("first output call satisfied the second output: %v", diags)
	}
	if diagnosed(diags, "helper for first") {
		t.Fatalf("the exact first output call was rejected: %v", diags)
	}
}

func cross_package_constant_isolation(t *testing.T) {
	t.Helper()
	foreign := parse(t, &parse_input{
		Path:        "other/constants.go",
		Source_Text: "package other\n\nconst Value_Min = -4\n\nconst Value_Max = 4\n"})
	local := parse(t, &parse_input{
		Path: "pkg/local_constants.go",
		Source_Text: "package fixture\n\n" +
			"import invariant \"fixture/shared/invariant/default\"\n\n" +
			"// Value is a fixture.\ntype Value int\n\n" +
			"// Value_Invariants is a fixture.\n" +
			"func Value_Invariants(value Value, namespace invariant.Namespace) {\n" +
			"\tinvariant.Assertions(namespace).Range_Int(" +
			"int(value), Value_Min, Value_Max).Ensure()\n}\n"})
	diags := check_sources([]source.Parsed_File{foreign, local})
	if !diagnosed(diags, "arguments must be package-level constants") {
		t.Fatalf("foreign same-spelled constants satisfied the local helper: %v", diags)
	}
}

func cross_package_helper_isolation(t *testing.T) {
	t.Helper()
	foreign := parse(t, &parse_input{
		Path: "other/token.go",
		Source_Text: "package other\n\n" +
			"import invariant \"fixture/shared/invariant/default\"\n\n" +
			"// Token is a fixture.\ntype Token int\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(value Token, namespace invariant.Namespace) {\n" +
			"\tinvariant.Int_Invariants(int(value), namespace)\n}\n"})
	local := parse(t, &parse_input{
		Path: "pkg/local_helper.go",
		Source_Text: "package fixture\n\n" +
			"import (\n\tinvariant \"fixture/shared/invariant/default\"\n" +
			"\tforeign \"fixture/other\"\n)\n\n" +
			"// Token is a fixture.\ntype Token int\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(value Token, namespace invariant.Namespace) {\n" +
			"\tinvariant.Int_Invariants(int(value), namespace)\n}\n\n" +
			"// Consume is a fixture.\nfunc Consume(value Token) {\n" +
			"\tforeign.Token_Invariants(foreign.Token(value), \"foreign\")\n}\n"})
	diags := check_sources([]source.Parsed_File{foreign, local})
	want := "Consume must call helper for value via Token_Invariants(value, ...)"
	if !diagnosed(diags, want) {
		t.Fatalf("foreign same-named helper satisfied the local subject: %v", diags)
	}
}

// Reading the analyzer itself pins the deliberately flat API without coupling the production
// package to a self-reflection mechanism needed only by this regression.
func assertion_input_struct(t *testing.T) (name string) {
	t.Helper()
	file, parse_error := parser.ParseFile(token.NewFileSet(), "assertion.go", nil, 0)
	if parse_error != nil {
		t.Fatal(parse_error)
	}
	for _, declaration := range file.Decls {
		general, is_general := declaration.(*ast.GenDecl)
		if !is_general {
			continue
		}
		for _, specification := range general.Specs {
			type_specification, is_type := specification.(*ast.TypeSpec)
			if !is_type {
				continue
			}
			if strings.HasSuffix(type_specification.Name.Name, "_Input") {
				return type_specification.Name.Name
			}
		}
	}
	return ""
}

// The fixture a parse turns into a Parsed_File.
type parse_input struct {
	// Path is the repo-relative path the fixture stands in for.
	Path string
	// Source_Text is the Go source of the fixture.
	Source_Text string
}

// A fixture becomes a Parsed_File the checks read; a parse error is fatal because a
// malformed fixture is a test bug, not a rule violation the checks should judge.
func parse(t *testing.T, input *parse_input) (pf source.Parsed_File) {
	t.Helper()
	file_set := token.NewFileSet()
	file, parse_error := parser.ParseFile(
		file_set, input.Path, input.Source_Text, parser.SkipObjectResolution)
	if parse_error != nil {
		t.Fatalf("parse %s: %v", input.Path, parse_error)
	}
	return source.Parsed_File{
		Path: input.Path, File_Set: file_set, File: file,
		Source: []byte(input.Source_Text),
	}
}

// Reports whether some diagnostic's message contains the fragment — the substring
// match every leaf uses to confirm the rule it documents actually fired.
func diagnosed(diags []diagnostic.Diagnostic, fragment string) (found bool) {
	for _, diag := range diags {
		if strings.Contains(diag.Message, fragment) {
			return true
		}
	}
	return false
}

// Parses and checks one body-mandate fixture at the package path used by the component index.
func check_fixture(t *testing.T, code string) (diags []diagnostic.Diagnostic) {
	t.Helper()
	return check_source(parse(t, &parse_input{Path: "pkg/rule.go", Source_Text: code}))
}

// Keeping the underlying type fixed makes each integer fixture vary only the helper shape under
// judgment, without introducing an argument bundle merely to share fixture text.
func integer_helper_source(body string) (code string) {
	return "package fixture\n\n" +
		"import invariant \"fixture/shared/invariant/default\"\n\n" +
		"const Value_Min = -4\n\nconst Value_Third = -1\n\n" +
		"const Value_Fourth = 1\n\nconst Value_Max = 4\n\n" +
		"// Value is a fixture.\ntype Value int\n\n" +
		"// Value_Invariants is a fixture.\n" +
		"func Value_Invariants(value Value, namespace invariant.Namespace) {\n" +
		body + "\n}\n"
}

// The float fixture is intentionally separate because type text is test data, not a runtime
// input whose bundling would improve the API.
func float_helper_source(body string) (code string) {
	return "package fixture\n\n" +
		"import invariant \"fixture/shared/invariant/default\"\n\n" +
		"const Value_Min = -4\n\nconst Value_Max = 4\n\n" +
		"// Value is a fixture.\ntype Value float64\n\n" +
		"// Value_Invariants is a fixture.\n" +
		"func Value_Invariants(value Value, namespace invariant.Namespace) {\n" +
		body + "\n}\n"
}

// The Boolean fixture remains explicit so no generic fixture input structure can conceal which
// primitive preset the leaf is proving.
func boolean_helper_source(body string) (code string) {
	return "package fixture\n\n" +
		"import invariant \"fixture/shared/invariant/default\"\n\n" +
		"const Value_Min = -4\n\nconst Value_Max = 4\n\n" +
		"// Value is a fixture.\ntype Value bool\n\n" +
		"// Value_Invariants is a fixture.\n" +
		"func Value_Invariants(value Value, namespace invariant.Namespace) {\n" +
		body + "\n}\n"
}

// Builds a defined byte-slice helper, the same counted shape as Report_Invariants.
func count_helper_source(body string) (code string) {
	return "package fixture\n\n" +
		"import invariant \"fixture/shared/invariant/default\"\n\n" +
		"const Value_Min = 0\n\nconst Value_Max = 32\n\n" +
		"// Value is a fixture.\ntype Value []byte\n\n" +
		"// Value_Invariants is a fixture.\n" +
		"func Value_Invariants(value Value, namespace invariant.Namespace) {\n" +
		body + "\n}\n"
}

// A same-shaped fluent builder owned by another package cannot impersonate invariant.Assertions.
func foreign_scalar_helper_source() (code string) {
	return "package fixture\n\n" +
		"import (\n\tinvariant \"fixture/shared/invariant/default\"\n" +
		"\tforeign \"fixture/other\"\n)\n\n" +
		"const Value_Min = -4\n\nconst Value_Max = 4\n\n" +
		"// Value is a fixture.\ntype Value int\n\n" +
		"// Value_Invariants is a fixture.\n" +
		"func Value_Invariants(value Value, namespace invariant.Namespace) {\n" +
		"\tforeign.Assertions(namespace)." +
		"Range_Int(int(value), int(Value_Min), int(Value_Max)).Ensure()\n}\n"
}

// The mandated literal qualifier keeps helper bodies visually and statically canonical.
func aliased_scalar_helper_source() (code string) {
	return "package fixture\n\n" +
		"import contract \"fixture/shared/invariant/default\"\n\n" +
		"const Value_Min = -4\n\nconst Value_Max = 4\n\n" +
		"// Value is a fixture.\ntype Value int\n\n" +
		"// Value_Invariants is a fixture.\n" +
		"func Value_Invariants(value Value, namespace contract.Namespace) {\n" +
		"\tcontract.Assertions(namespace)." +
		"Range_Int(int(value), int(Value_Min), int(Value_Max)).Ensure()\n}\n"
}

// Builds one Range assertion after enough neutral links to pin the expanded-link boundary exactly.
func assertions_builder_body(link_count int) (body string) {
	return "\tinvariant.Assertions(namespace)." +
		strings.Repeat("Sometimes(true, \"axis\").", link_count-1) +
		"Range_Int(int(value), int(Value_Min), int(Value_Max)).Ensure()"
}

// Runs the cross-file checks over one package plus the framework component. Exact helper identity
// needs both import roots; the package remains a binary so recorder checks stay inert.
func check_source(pf source.Parsed_File) (diags []diagnostic.Diagnostic) {
	return check_sources([]source.Parsed_File{pf})
}

func check_sources(files []source.Parsed_File) (diags []diagnostic.Diagnostic) {
	file_to_component := map[string]int{}
	for _, file := range files {
		component := 0
		if strings.HasPrefix(file.Path, "other/") {
			component = 1
		}
		file_to_component[file.Path] = component
	}
	components := &source.Component_Index{
		Components: []source.Component{
			{Root: "pkg", Import_Path: "fixture/pkg"},
			{Root: "other", Import_Path: "fixture/other"},
			{
				Root: "shared", Import_Path: "fixture/shared",
				Is_Shared_Library: true,
			},
		},
		File_To_Component: file_to_component,
	}
	return assertion.Check(files, components, nil)
}

// Runs Check with the fixture treated as one shared-library component, the only tier
// the recorder rule binds; every file maps to it so the TestMain wiring is judged.
func recorder_diagnostics(files []source.Parsed_File) (diags []diagnostic.Diagnostic) {
	mapping := map[string]int{}
	for _, pf := range files {
		mapping[pf.Path] = 0
	}
	components := &source.Component_Index{
		Components: []source.Component{{
			Root: "pkg", Import_Path: "fixture/pkg", Is_Shared_Library: true,
		}},
		File_To_Component: mapping,
	}
	return assertion.Check(files, components, nil)
}

// Runs Check with the fixture treated as one binary component rooted at pkg, so the
// simulation checks judge its internal/simulation_test package. Import_Path is the
// real module path the entry check compares the simulation's selector imports against.
func simulation_diagnostics(files []source.Parsed_File) (diags []diagnostic.Diagnostic) {
	mapping := map[string]int{}
	for _, pf := range files {
		mapping[pf.Path] = 0
	}
	components := &source.Component_Index{
		Components: []source.Component{{
			Root:        "pkg",
			Import_Path: "github.com/james-orcales/james-orcales/pkg",
		}},
		File_To_Component: mapping,
	}
	return assertion.Check(files, components, nil)
}

// The binary-component fixture without a simulation package — a thin main and an
// internal entry point — the base each simulation leaf builds on.
func simulation_base(t *testing.T) (files []source.Parsed_File) {
	t.Helper()
	return []source.Parsed_File{
		parse(t, &parse_input{
			Path: "pkg/main.go",
			Source_Text: "// Package main is a fixture.\n" +
				"package main\n\nfunc main() {}\n"}),
		parse(t, &parse_input{
			Path: "pkg/internal/entry.go",
			Source_Text: "// Package internal is a fixture.\npackage internal\n\n" +
				"// Main is the entry point.\nfunc Main() {}\n"}),
	}
}

// The base fixture plus the given simulation package source at the conventional
// internal/simulation_test path, so a leaf varies only the part it exercises.
func simulation_files(t *testing.T, sim string) (files []source.Parsed_File) {
	t.Helper()
	files = simulation_base(t)
	return append(files, parse(t, &parse_input{
		Path:        "pkg/internal/simulation_test/sim_test.go",
		Source_Text: sim}))
}

// A simulation package source with the given TestMain call and a valid Fuzz driver,
// so the TestMain and coverage leaves vary only that one call.
func simulation_fixture_source(call string) (code string) {
	return "package simulation_test\n\n" +
		"import (\n\t\"testing\"\n\n" +
		"\tinvariant \"fixture/shared/invariant/default\"\n)\n\n" +
		"func TestMain(m *testing.M) {\n\t" + call + "\n}\n\n" +
		"func Fuzz_Main(f *testing.F) {\n\t" +
		"f.Fuzz(func(t *testing.T, data []byte) {})\n}\n"
}
