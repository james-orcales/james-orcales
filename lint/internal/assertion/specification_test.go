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
			"type Widget struct {\n\t// X is a fixture.\n\tX Part\n}\n"})
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
			"\t// X is a fixture.\n\tX Part\n}\n\n" +
			"// Widget_invariants is wrongly cased.\n" +
			"func Widget_invariants(identifier string, w Widget) {\n\tprintln(0)\n}\n"})
	diags := assertion.Check_Type(pf.File_Set, pf.File, nil)
	if !diagnosed(diags, "declare Widget_Invariants") {
		t.Fatal("an exported type wants the _Invariants suffix")
	}
}

// Test_Invariants_Signature verifies a bundle missing its leading identifier
// parameter is flagged.
func Test_Invariants_Signature(t *testing.T) {
	t.Parallel()
	pf := parse(t, &parse_input{
		Path: "pkg/rule.go",
		Source_Text: "package fixture\n\n" +
			"// Widget is a fixture.\ntype Widget struct {\n" +
			"\t// X is a fixture.\n\tX Part\n}\n\n" +
			"// Widget_Invariants is a fixture.\n" +
			"func Widget_Invariants(w Widget) {\n\tprintln(0)\n}\n"})
	diags := assertion.Check_Type(pf.File_Set, pf.File, nil)
	if !diagnosed(diags, "must take") {
		t.Fatal("a bundle without the leading identifier parameter must be flagged")
	}
	swapped := parse(t, &parse_input{
		Path: "pkg/rule.go",
		Source_Text: "package fixture\n\n" +
			"// Widget is a fixture.\ntype Widget struct {\n" +
			"\t// X is a fixture.\n\tX Part\n}\n\n" +
			"// Widget_Invariants is a fixture.\n" +
			"func Widget_Invariants(w Widget, identifier string) {\n\tprintln(0)\n}\n"})
	if !diagnosed(assertion.Check_Type(swapped.File_Set, swapped.File, nil), "must take") {
		t.Fatal("a bundle with the identifier trailing must be flagged")
	}
	correct := parse(t, &parse_input{
		Path: "pkg/rule.go",
		Source_Text: "package fixture\n\n" +
			"// Widget is a fixture.\ntype Widget struct {\n" +
			"\t// X is a fixture.\n\tX Part\n}\n\n" +
			"// Widget_Invariants is a fixture.\n" +
			"func Widget_Invariants(identifier string, w Widget) {\n\tprintln(0)\n}\n"})
	if diagnosed(assertion.Check_Type(correct.File_Set, correct.File, nil), "must take") {
		t.Fatal("the (identifier string, type) shape must satisfy the signature")
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

// Test_Invariants_Scalar_Helper verifies a defined integer states its domain with the
// bare Range or Enum guard, except a composed type — one some struct declares as a
// field — which may state hand-written assertions instead, since the bare guard's
// fixed witness texts collide under a shared root and only composition creates one.
// Floats and booleans state hand-written assertions, and an empty body satisfies
// nothing.
func Test_Invariants_Scalar_Helper(t *testing.T) {
	t.Parallel()
	hand := "\tinvariant.Always(value >= Value_Min, \"min\")\n" +
		"\tinvariant.Sometimes(identifier, value == Value_Min, \"at the floor\")"
	if !diagnosed(check_fixture(t, integer_helper_source(hand)),
		"must state invariant.Range or invariant.Enum") {
		t.Fatal("a root-only integer helper must state the bare guard")
	}
	if diagnosed(check_fixture(t, composed_integer_helper_source(hand)),
		"must state invariant.Range or invariant.Enum") {
		t.Fatal("hand-written assertions must satisfy a composed type's mandate")
	}
	empty := integer_helper_source("\tprintln(0)")
	if !diagnosed(check_fixture(t, empty), "must state invariant.Range or invariant.Enum") {
		t.Fatal("an integer helper stating nothing must be flagged")
	}
	literal := composed_integer_helper_source(
		"\tinvariant.Sometimes(\"literal\", value == Value_Min, \"at the floor\")")
	if !diagnosed(check_fixture(t, literal), "must state invariant.Range or invariant.Enum") {
		t.Fatal("a literal-identifier Sometimes must not satisfy the composable mandate")
	}
	mixed := integer_helper_source("\tinvariant.Range(identifier, int(value), Value_Min, 7)\n" +
		"\tinvariant.Sometimes(identifier, value == Value_Min, \"at the floor\")")
	if !diagnosed(check_fixture(t, mixed), "arguments must be package-level constants") {
		t.Fatal("a stated bare guard must still pin its domain in package constants")
	}
	range_body := "\tinvariant.Range(identifier, int(value), Value_Min, Value_Max)"
	if diagnosed(check_fixture(t, integer_helper_source(range_body)),
		"must state invariant.Range or invariant.Enum") {
		t.Fatal("the bare Range guard must satisfy the scalar mandate")
	}
	enum_body := "\tinvariant.Enum(identifier, int(value), Value_Min, Value_Max)"
	if diagnosed(check_fixture(t, integer_helper_source(enum_body)),
		"must state invariant.Range or invariant.Enum") {
		t.Fatal("the bare Enum guard must satisfy the scalar mandate")
	}
	float_body := "\tinvariant.Sometimes(identifier, float64(value) == 0, " +
		"\"The value is zero.\")"
	if diagnosed(check_fixture(t, float_helper_source(float_body)),
		"must state invariant.Always or invariant.Sometimes") {
		t.Fatal("a hand-written Sometimes must satisfy the float mandate")
	}
	if !diagnosed(check_fixture(t, float_helper_source("\tprintln(0)")),
		"must state invariant.Always or invariant.Sometimes") {
		t.Fatal("a float helper stating nothing must be flagged")
	}
	boolean_body := "\tinvariant.Sometimes(identifier, bool(value), \"The value is true.\")"
	if diagnosed(check_fixture(t, boolean_helper_source(boolean_body)),
		"must state invariant.Always or invariant.Sometimes") {
		t.Fatal("a hand-written Sometimes must satisfy the boolean mandate")
	}
}

// Test_Invariants_Count_Helper verifies a counted type states its domain with a bare
// Range or Enum over its own length — hand-written assertions substitute only for a
// composed type — and a guard over another subject alone satisfies neither shape.
func Test_Invariants_Count_Helper(t *testing.T) {
	t.Parallel()
	hand := "\tinvariant.Always(len(value) >= Value_Min, \"min\")\n" +
		"\tinvariant.Sometimes(identifier, len(value) == Value_Min, \"at the floor\")"
	if !diagnosed(check_fixture(t, count_helper_source(hand)),
		"must state invariant.Range or invariant.Enum") {
		t.Fatal("a root-only count helper must state the bare guard")
	}
	if diagnosed(check_fixture(t, composed_count_helper_source(hand)),
		"must state invariant.Range or invariant.Enum") {
		t.Fatal("hand-written assertions must satisfy a composed count type's mandate")
	}
	valid := "\tinvariant.Range(identifier, len(value), Value_Min, Value_Max, 1, 2)"
	if diagnosed(check_fixture(t, count_helper_source(valid)),
		"must state invariant.Range or invariant.Enum") {
		t.Fatal("Range over the counted value must satisfy the mandate")
	}
	wrong_subject := "\tinvariant.Range(identifier, Value_Min, Value_Min, Value_Max)"
	if !diagnosed(check_fixture(t, count_helper_source(wrong_subject)),
		"must state invariant.Range or invariant.Enum") {
		t.Fatal("a guard over another subject must not substitute")
	}
}

// Test_Invariants_Helper_Constants verifies bare guards retain package-level constant
// identity for Range edges and Enum members.
func Test_Invariants_Helper_Constants(t *testing.T) {
	t.Parallel()
	inline_range := "\tinvariant.Range(identifier, int(value), Value_Min, 7)"
	if !diagnosed(check_fixture(t, integer_helper_source(inline_range)),
		"arguments must be package-level constants") {
		t.Fatal("an inline Range edge must not satisfy the helper mandate")
	}
	inline_enum := "\tinvariant.Enum(identifier, int(value), Value_Min, 7)"
	if !diagnosed(check_fixture(t, integer_helper_source(inline_enum)),
		"arguments must be package-level constants") {
		t.Fatal("an inline Enum member must not satisfy the helper mandate")
	}
	converted := "\tinvariant.Range(identifier, int(value), int(Value_Min), int(Value_Max))"
	if diagnosed(check_fixture(t, integer_helper_source(converted)),
		"arguments must be package-level constants") {
		t.Fatal("exactly converted package constants must satisfy the mandate")
	}
	shadowed := "\tValue_Min := 0\n" +
		"\tinvariant.Range(identifier, int(value), Value_Min, Value_Max)"
	if !diagnosed(check_fixture(t, integer_helper_source(shadowed)),
		"arguments must be package-level constants") {
		t.Fatal("a local shadow of a package constant must not satisfy the mandate")
	}
}

// Test_Invariants_Helper_Identity verifies only a direct guard resolving to the actual
// invariant package and forwarding the helper's identifier parameter satisfies the
// body mandate.
func Test_Invariants_Helper_Identity(t *testing.T) {
	t.Parallel()
	literal := "\tinvariant.Range(\"manual\", int(value), Value_Min, Value_Max)"
	if !diagnosed(check_fixture(t, integer_helper_source(literal)),
		"must state invariant.Range or invariant.Enum") {
		t.Fatal("a literal identifier must not satisfy a helper body")
	}
	nested := "\tif true {\n" +
		"\t\tinvariant.Range(identifier, int(value), Value_Min, Value_Max)\n\t}"
	if !diagnosed(check_fixture(t, integer_helper_source(nested)),
		"must state invariant.Range or invariant.Enum") {
		t.Fatal("a nested guard must not satisfy the direct helper mandate")
	}
	foreign := foreign_scalar_helper_source()
	if !diagnosed(check_fixture(t, foreign), "must state invariant.Range or invariant.Enum") {
		t.Fatal("a foreign Range lookalike must not satisfy the mandate")
	}
	aliased := aliased_scalar_helper_source()
	if diagnosed(check_fixture(t, aliased), "must state invariant.Range or invariant.Enum") {
		t.Fatal("an aliased import of the real invariant package must satisfy the mandate")
	}
	conversion := "\tint := func(Value) int { return 0 }\n" +
		"\tinvariant.Range(identifier, int(value), Value_Min, Value_Max)"
	if !diagnosed(check_fixture(t, integer_helper_source(conversion)),
		"must state invariant.Range or invariant.Enum") {
		t.Fatal("a local function shadowing the primitive conversion must not substitute")
	}
	count := "\tlen := func(Value) int { return 0 }\n" +
		"\tinvariant.Range(identifier, len(value), Value_Min, Value_Max)"
	if !diagnosed(check_fixture(t, count_helper_source(count)),
		"must state invariant.Range or invariant.Enum") {
		t.Fatal("a local function shadowing len must not substitute")
	}
}

// Test_Invariants_Field_Composition verifies a struct bundle that fails to call a
// field type's _Invariants — identifier forwarded, field second — is flagged, whether
// the field is a value or a pointer.
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
			"func Token_Invariants(identifier string, v Token) {\n" +
			"\tinvariant.Range(identifier, len(v), Token_Min, Token_Max)\n}\n\n" +
			"// Lexeme is a fixture.\ntype Lexeme struct {\n" +
			"\t// Tok is a fixture.\n\tTok Token\n}\n\n" +
			"// Lexeme_Invariants is a fixture.\n" +
			"func Lexeme_Invariants(identifier string, v Lexeme) {\n" +
			"\tforeign.Token_Invariants(identifier, v.Tok)\n" +
			"\tif false { Token_Invariants(identifier, v.Tok) }\n" +
			"\tToken_Invariants := func(string, Token) {}\n" +
			"\tToken_Invariants(identifier, v.Tok)\n" +
			"\tinvariant.Sometimes(identifier, true, \"x\")\n}\n\n" +
			"// Phrase is a fixture.\ntype Phrase struct {\n" +
			"\t// Tok is a fixture.\n\tTok *Token\n}\n\n" +
			"// Phrase_Invariants is a fixture.\n" +
			"func Phrase_Invariants(identifier string, v Phrase) {\n" +
			"\tToken_Invariants(v.Tok, identifier)\n" +
			"\tinvariant.Sometimes(identifier, true, \"y\")\n}\n"})
	diags := check_source(pf)
	if !diagnosed(diags, "Lexeme_Invariants must call Token_Invariants") {
		t.Fatal("a struct that does not compose a value field's invariant must be flagged")
	}
	// A pointer field composes its pointee, and the field travels second behind the
	// forwarded identifier — a swapped-argument call must not substitute.
	if !diagnosed(diags, "Phrase_Invariants must call Token_Invariants") {
		t.Fatal("a swapped-argument composition must be flagged")
	}
	external := parse(t, &parse_input{
		Path: "other/token.go",
		Source_Text: "package other\n\n" +
			"import invariant \"fixture/shared/invariant/default\"\n\n" +
			"const Token_Min = 0\n\nconst Token_Max = 8\n\n" +
			"// Token is a fixture.\ntype Token string\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(identifier string, v Token) {\n" +
			"\tinvariant.Range(identifier, len(v), Token_Min, Token_Max)\n}\n"})
	composed := parse(t, &parse_input{
		Path: "pkg/composed.go",
		Source_Text: "package fixture\n\n" +
			"import external \"fixture/other\"\n\n" +
			"// Holder is a fixture.\ntype Holder struct {\n" +
			"\t// Token is external.\n\tToken external.Token\n}\n\n" +
			"// Holder_Invariants is a fixture.\n" +
			"func Holder_Invariants(identifier string, v Holder) {\n" +
			"\texternal.Token_Invariants(identifier, v.Token)\n}\n"})
	if diagnosed(check_sources([]source.Parsed_File{external, composed}), "must call") {
		t.Fatal("an aliased exact cross-package composition must satisfy the mandate")
	}
}

// Test_Invariants_Parameter_Helper verifies only the input type's exact helper —
// subject second, behind the root identifier — satisfies the leading helper
// requirement.
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
			"func Token_Invariants(identifier string, v Token) {\n" +
			"\tinvariant.Range(identifier, len(v), Token_Min, Token_Max)\n}\n\n" +
			"// Consume does.\nfunc Consume(tok Token) {\n" +
			"\tforeign.Token_Invariants(\"wrong\", tok)\n" +
			"\tToken_Invariants(tok, \"wrong-order\")\n" +
			"\tinvariant.Always(true, \"raw guard\")\n" +
			"\tprintln(0)\n}\n"})
	if !diagnosed(check_source(pf), "must call helper for tok") {
		t.Fatal("foreign, swapped, and direct assertions must not satisfy the input helper")
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
			"func Token_Invariants(identifier string, v Token) {\n" +
			"\tinvariant.Range(identifier, len(v), Token_Min, Token_Max)\n}\n\n" +
			"// Make does.\nfunc Make() (tok Token) {\n\tdefer func() {\n" +
			"\t\tforeign.Token_Invariants(\"wrong\", tok)\n" +
			"\t\tif false { Token_Invariants(\"nested\", tok) }\n" +
			"\t\tToken_Invariants := func(string, Token) {}\n" +
			"\t\tToken_Invariants(\"shadowed\", tok)\n" +
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
			"func Token_Invariants(identifier string, v Token) {\n" +
			"\tinvariant.Range(identifier, len(v), Token_Min, Token_Max)\n}\n\n" +
			"// Make uses the exact output helper.\nfunc Make() (tok Token) {\n" +
			"\tdefer func() {\n\t\tToken_Invariants(\"Make.tok\", tok)\n" +
			"\t}()\n\treturn \"\"\n}\n"})
	if diagnosed(check_source(correct), "must call helper") {
		t.Fatal("the exact helper in the first output defer must satisfy the mandate")
	}
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

// Test_Invariants_Primitive_Types verifies a primitive used as a function
// parameter, result, or struct field is flagged: all types are semantic.
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
	numeric := parse(t, &parse_input{
		Path: "pkg/rule.go",
		Source_Text: "package fixture\n\n" +
			"// Pair is a fixture.\ntype Pair struct {\n" +
			"\t// Flag is a fixture.\n\tFlag bool\n}\n\n" +
			"// Pair_Invariants is a fixture.\n" +
			"func Pair_Invariants(identifier string, p Pair) {\n\tprintln(0)\n}\n\n" +
			"// Add does.\nfunc Add(count int) {\n\tprintln(count)\n}\n"})
	numeric_diags := check_source(numeric)
	if !diagnosed(numeric_diags, "raw int parameter") {
		t.Fatal("a raw int parameter must be flagged")
	}
	if !diagnosed(numeric_diags, "raw bool field") {
		t.Fatal("a raw bool field must be flagged")
	}
	if diagnosed(numeric_diags, "raw string parameter identifier") {
		t.Fatal("a bundle's mandated leading identifier parameter must not be flagged")
	}
	misnamed := parse(t, &parse_input{
		Path: "pkg/rule.go",
		Source_Text: "package fixture\n\n" +
			"// Pair is a fixture.\ntype Pair struct {\n" +
			"\t// Flag is a fixture.\n\tFlag Part\n}\n\n" +
			"// Pair_Invariants is a fixture.\n" +
			"func Pair_Invariants(name string, p Pair) {\n\tprintln(0)\n}\n\n" +
			"// Describe does.\nfunc Describe(identifier string) {\n\tprintln(0)\n}\n"})
	misnamed_diags := check_source(misnamed)
	if !diagnosed(misnamed_diags, "raw string parameter name") {
		t.Fatal("the exemption covers only the mandated identifier spelling")
	}
	if !diagnosed(misnamed_diags, "raw string parameter identifier") {
		t.Fatal("a plain function's identifier parameter stays banned")
	}
}

// Test_Invariants_Semantic_Declarations verifies a type declared over another
// defined type is flagged: declarations sit directly on the primitive.
func Test_Invariants_Semantic_Declarations(t *testing.T) {
	t.Parallel()
	pf := parse(t, &parse_input{
		Path: "pkg/rule.go",
		Source_Text: "package fixture\n\n" +
			"import other \"fixture/other\"\n\n" +
			"// Tally is a fixture.\ntype Tally int\n\n" +
			"// Tally_Invariants is a fixture.\n" +
			"func Tally_Invariants(identifier string, t Tally) {\n\tprintln(0)\n}\n\n" +
			"// Kept is a fixture.\ntype Kept Tally\n\n" +
			"// Kept_Invariants is a fixture.\n" +
			"func Kept_Invariants(identifier string, k Kept) {\n\tprintln(0)\n}\n\n" +
			"// Reference is a fixture.\ntype Reference other.Measurements\n\n" +
			"// Reference_Invariants is a fixture.\n" +
			"func Reference_Invariants(identifier string, r Reference) {\n\tprintln(0)\n}\n\n" +
			"// Samples is a fixture.\ntype Samples []Tally\n\n" +
			"// Samples_Invariants is a fixture.\n" +
			"func Samples_Invariants(identifier string, s Samples) {\n\tprintln(0)\n}\n\n" +
			"// Alias is a fixture.\ntype Alias = Tally\n"})
	diags := check_source(pf)
	if !diagnosed(diags, "Kept declares on defined type Tally") {
		t.Fatal("a declaration over a same-package defined type must be flagged")
	}
	if !diagnosed(diags, "Reference declares on defined type other.Measurements") {
		t.Fatal("a declaration over a cross-package defined type must be flagged")
	}
	if diagnosed(diags, "Tally declares") {
		t.Fatal("a declaration on the primitive must pass")
	}
	if diagnosed(diags, "Samples declares") {
		t.Fatal("a slice declaration over a semantic type is composition, not aliasing")
	}
	if diagnosed(diags, "Alias declares") {
		t.Fatal("an alias renames, it does not declare")
	}
}

// Test_Invariants_Distinct_Fields verifies a struct with two fields of one type is
// flagged, whether declared in one field entry or across several.
func Test_Invariants_Distinct_Fields(t *testing.T) {
	t.Parallel()
	pf := parse(t, &parse_input{
		Path: "pkg/rule.go",
		Source_Text: "package fixture\n\n" +
			"// Metric is a fixture.\ntype Metric int64\n\n" +
			"// Metric_Invariants is a fixture.\n" +
			"func Metric_Invariants(identifier string, m Metric) {\n\tprintln(0)\n}\n\n" +
			"// Sample is a fixture.\ntype Sample struct {\n" +
			"\t// First is a fixture.\n\tFirst Metric\n" +
			"\t// Second is a fixture.\n\tSecond Metric\n}\n\n" +
			"// Sample_Invariants is a fixture.\n" +
			"func Sample_Invariants(identifier string, s Sample) {\n\tprintln(0)\n}\n\n" +
			"// Pair is a fixture.\ntype Pair struct {\n" +
			"\t// Left and Right share one entry.\n\tLeft, Right Metric\n}\n\n" +
			"// Pair_Invariants is a fixture.\n" +
			"func Pair_Invariants(identifier string, p Pair) {\n\tprintln(0)\n}\n\n" +
			"// Mixed is a fixture.\ntype Mixed struct {\n" +
			"\t// Count is a fixture.\n\tCount Metric\n" +
			"\t// Elapsed is a fixture.\n\tElapsed Sample\n}\n\n" +
			"// Mixed_Invariants is a fixture.\n" +
			"func Mixed_Invariants(identifier string, m Mixed) {\n\tprintln(0)\n}\n"})
	diags := check_source(pf)
	if !diagnosed(diags, "Sample declares two fields of type Metric") {
		t.Fatal("two same-typed fields across entries must be flagged")
	}
	if !diagnosed(diags, "Pair declares two fields of type Metric") {
		t.Fatal("two same-typed fields in one entry must be flagged")
	}
	if diagnosed(diags, "Mixed declares two fields") {
		t.Fatal("distinct field types must pass")
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
		"\tinvariant \"fixture/shared/invariant\"\n)\n\n" +
		"func TestMain(m *testing.M) {\n\tinvariant.Run_Test_Main(m, \"../**\")\n}\n"
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
		"\tinvariant \"fixture/shared/invariant\"\n)\n\n" +
		"func TestMain(m *testing.M) {\n\tinvariant.Run_Test_Main(m, \"../**\")\n}\n\n" +
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
		"\tinvariant \"fixture/shared/invariant\"\n)\n\n" +
		"func TestMain(m *testing.M) {\n\tinvariant.Run_Test_Main(m, \"../**\")\n}\n\n" +
		"func Fuzz_Main(f *testing.F) {\n\tf.Fuzz(func(t *testing.T, data []byte) {})\n}\n"
	if !diagnosed(simulation_diagnostics(simulation_files(t, sim)), "must be blackbox") {
		t.Fatal("a whitebox simulation package must be flagged")
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
			"func Token_Invariants(identifier string, v Token) {\n" +
			"\tinvariant.Range(identifier, len(v), Token_Min, Token_Max)\n}\n\n" +
			"// Consume uses the exact helper.\nfunc Consume(tok Token) {\n" +
			"\tToken_Invariants(\"Consume.tok\", tok)\n}\n"})
	if diagnosed(check_source(correct), "must call helper") {
		t.Fatal("the exact identifier-first input helper must satisfy the mandate")
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
			"func Token_Invariants(identifier string, v Token) {\n" +
			"\tinvariant.Range(identifier, len(v), Token_Min, Token_Max)\n}\n\n" +
			"// Consume shadows the helper.\n" +
			"func Consume(tok Token, Token_Invariants " +
			"func(string, Token)) {\n" +
			"\tToken_Invariants(\"shadowed\", tok)\n}\n"})
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
			"func Input_Invariants(identifier string, v Input) {\n" +
			"\tinvariant.Range(identifier, int(v), Input_Min, Input_Max)\n}\n"})
	consumer := parse(t, &parse_input{
		Path: "pkg/external.go",
		Source_Text: "package fixture\n\n" +
			"import external \"fixture/other\"\n\n" +
			"// Consume uses an aliased external helper.\n" +
			"func Consume(input external.Input) {\n" +
			"\texternal.Input_Invariants(\"Consume.input\", input)\n}\n"})
	if diagnosed(check_sources([]source.Parsed_File{external, consumer}), "must call helper") {
		t.Fatal("an aliased exact cross-package input helper must satisfy the mandate")
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
		"const Value_Min = -4\n\nconst Value_Max = 4\n\n" +
		"// Value is a fixture.\ntype Value int\n\n" +
		"// Value_Invariants is a fixture.\n" +
		"func Value_Invariants(identifier string, value Value) {\n" +
		body + "\n}\n"
}

// The composed variant adds a struct declaring the type as a field: composition is what
// makes the bare guard's fixed texts collide, so it alone unlocks the hand-written shape.
func composed_integer_helper_source(body string) (code string) {
	return integer_helper_source(body) +
		"\n// Holder is a fixture.\ntype Holder struct {\n" +
		"\t// Field is a fixture.\n\tField Value\n}\n\n" +
		"// Holder_Invariants is a fixture.\n" +
		"func Holder_Invariants(identifier string, h Holder) {\n" +
		"\tValue_Invariants(identifier, h.Field)\n}\n"
}

// The float fixture is intentionally separate because type text is test data, not a runtime
// input whose bundling would improve the API.
func float_helper_source(body string) (code string) {
	return "package fixture\n\n" +
		"import invariant \"fixture/shared/invariant/default\"\n\n" +
		"const Value_Min = -4\n\nconst Value_Max = 4\n\n" +
		"// Value is a fixture.\ntype Value float64\n\n" +
		"// Value_Invariants is a fixture.\n" +
		"func Value_Invariants(identifier string, value Value) {\n" +
		body + "\n}\n"
}

// The Boolean fixture remains explicit so no generic fixture input structure can conceal which
// hand-written witness the leaf is proving.
func boolean_helper_source(body string) (code string) {
	return "package fixture\n\n" +
		"import invariant \"fixture/shared/invariant/default\"\n\n" +
		"const Value_Min = -4\n\nconst Value_Max = 4\n\n" +
		"// Value is a fixture.\ntype Value bool\n\n" +
		"// Value_Invariants is a fixture.\n" +
		"func Value_Invariants(identifier string, value Value) {\n" +
		body + "\n}\n"
}

// Builds a defined byte-slice helper, the same counted shape as Report_Invariants.
func count_helper_source(body string) (code string) {
	return "package fixture\n\n" +
		"import invariant \"fixture/shared/invariant/default\"\n\n" +
		"const Value_Min = 0\n\nconst Value_Max = 32\n\n" +
		"// Value is a fixture.\ntype Value []byte\n\n" +
		"// Value_Invariants is a fixture.\n" +
		"func Value_Invariants(identifier string, value Value) {\n" +
		body + "\n}\n"
}

// The composed counted variant mirrors composed_integer_helper_source for len-guarded types.
func composed_count_helper_source(body string) (code string) {
	return count_helper_source(body) +
		"\n// Holder is a fixture.\ntype Holder struct {\n" +
		"\t// Field is a fixture.\n\tField Value\n}\n\n" +
		"// Holder_Invariants is a fixture.\n" +
		"func Holder_Invariants(identifier string, h Holder) {\n" +
		"\tValue_Invariants(identifier, h.Field)\n}\n"
}

// A same-shaped bare guard owned by another package cannot impersonate invariant.Range.
func foreign_scalar_helper_source() (code string) {
	return "package fixture\n\n" +
		"import (\n\tinvariant \"fixture/shared/invariant/default\"\n" +
		"\tforeign \"fixture/other\"\n)\n\n" +
		"const Value_Min = -4\n\nconst Value_Max = 4\n\n" +
		"// Value is a fixture.\ntype Value int\n\n" +
		"// Value_Invariants is a fixture.\n" +
		"func Value_Invariants(identifier string, value Value) {\n" +
		"\tforeign.Range(identifier, int(value), Value_Min, Value_Max)\n}\n"
}

// Import aliases change spelling, not the identity of the canonical invariant package.
func aliased_scalar_helper_source() (code string) {
	return "package fixture\n\n" +
		"import contract \"fixture/shared/invariant/default\"\n\n" +
		"const Value_Min = -4\n\nconst Value_Max = 4\n\n" +
		"// Value is a fixture.\ntype Value int\n\n" +
		"// Value_Invariants is a fixture.\n" +
		"func Value_Invariants(identifier string, value Value) {\n" +
		"\tcontract.Range(identifier, int(value), Value_Min, Value_Max)\n}\n"
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
		"import (\n\t\"testing\"\n\n\tinvariant \"fixture/shared/invariant\"\n)\n\n" +
		"func TestMain(m *testing.M) {\n\t" + call + "\n}\n\n" +
		"func Fuzz_Main(f *testing.F) {\n\t" +
		"f.Fuzz(func(t *testing.T, data []byte) {})\n}\n"
}
