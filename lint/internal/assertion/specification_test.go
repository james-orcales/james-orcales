package assertion_test

import (
	"go/parser"
	"go/token"
	"local/james-orcales/lint/internal/strings"
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
	if !diagnosed(diags, "directly below the type Widget") {
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
	if !diagnosed(diags, "Declare Widget_Invariants") {
		t.Fatal("an exported type wants the _Invariants suffix")
	}
}

// Test_Invariants_Signature verifies a bundle missing its aver.Namespace
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
	if !diagnosed(diags, "Write the parameters (Widget or *Widget, aver.Namespace).") {
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
	if !diagnosed(diags, "directly below the type of the same name") {
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
	if !diagnosed(diags, "directly below the type Count") {
		t.Fatal("a defined scalar type is in scope")
	}
	exemptions := &assertion.Exemptions{
		Packages:                 []string{"**", "!pkg/**"},
		Instrumentation_Packages: []string{"pkg/"},
	}
	diags = assertion.Check_Type(pf.File_Set, pf.File, exemptions)
	if diagnosed(diags, "directly below the type Count") {
		t.Fatal("assertion opt-in must not restore instrumentation mandate")
	}
}

// Test_Invariants_Underlying_Kind verifies a chain of defined types cannot hide the kind it stands
// over, thus a named integer still owes the scalar mandate and a named slice the count mandate.
func Test_Invariants_Underlying_Kind(t *testing.T) {
	t.Parallel()
	loose := "\taver.Always(int(v) > Span_Min, \"loose\")"
	if !diagnosed(check_fixture(t, chained_helper_source(CHAINED_INTEGER, loose)),
		"does not call a canonical helper") {
		t.Fatal("a defined type over a named integer owes the scalar mandate")
	}
	exact := "\taver.Tree(v, namespace)." +
		"Range_Int(int(v), Span_Min, Span_Max).Ensure()"
	if diagnosed(check_fixture(t, chained_helper_source(CHAINED_INTEGER, exact)),
		"does not call a canonical helper") {
		t.Fatal("the exact Range helper must satisfy the resolved scalar mandate")
	}
	counted := "\taver.Always(len(v) > Span_Min, \"loose\")"
	if !diagnosed(check_fixture(t, chained_helper_source(CHAINED_SLICE, counted)),
		"Write a Range_Int family, an Enum_Int family") {
		t.Fatal("a defined type over a named slice owes the count mandate")
	}
	qualified_base_resolves(t)
}

// Test_Invariants_Scalar_Helper verifies raw assertions cannot replace the canonical primitive,
// Range, or Enum helper required by a defined scalar.
func Test_Invariants_Scalar_Helper(t *testing.T) {
	t.Parallel()
	raw := integer_helper_source("\taver.Always(value >= Value_Min, \"min\")\n" +
		"\taver.Always(value <= Value_Max, \"max\")\n" +
		"\taver.Tree(value, namespace).Sometimes(value == Value_Min, \"min\")." +
		"Sometimes(value == Value_Max, \"max\").Ensure()")
	if !diagnosed(check_fixture(t, raw), "does not call a canonical helper") {
		t.Fatal("individual scalar assertions must not satisfy the helper mandate")
	}
	range_body := "\taver.Tree(value, namespace)." +
		"Range_Int(int(value), int(Value_Min), int(Value_Max)).Ensure()"
	if diagnosed(check_fixture(t, integer_helper_source(range_body)),
		"does not call a canonical helper") {
		t.Fatal("the exact Range helper must satisfy the scalar mandate")
	}
	scalar_range_holed_helper(t)
	enum_body := "\taver.Tree(value, namespace)." +
		"Enum_Int(int(value), int(Value_Min), int(Value_Max)).Ensure()"
	if diagnosed(check_fixture(t, integer_helper_source(enum_body)),
		"does not call a canonical helper") {
		t.Fatal("the exact Enum helper must satisfy the scalar mandate")
	}
	enum_3_body := "\taver.Tree(value, namespace)." +
		"Enum_3_Int(int(value), int(Value_Min), int(Value_Third), int(Value_Max)).Ensure()"
	if diagnosed(check_fixture(t, integer_helper_source(enum_3_body)),
		"does not call a canonical helper") {
		t.Fatal("the exact Enum_3 helper must satisfy the scalar mandate")
	}
	enum_4_body := "\taver.Tree(value, namespace)." +
		"Enum_4_Int(int(value), int(Value_Min), int(Value_Third), " +
		"int(Value_Fourth), int(Value_Max)).Ensure()"
	if diagnosed(check_fixture(t, integer_helper_source(enum_4_body)),
		"does not call a canonical helper") {
		t.Fatal("the exact Enum_4 helper must satisfy the scalar mandate")
	}
	retired := "\taver.Int_Invariants(int(value), namespace)"
	if !diagnosed(check_fixture(t, integer_helper_source(retired)),
		"does not call a canonical helper") {
		t.Fatal("a deleted primitive preset must not satisfy the scalar mandate")
	}
	singleton := "\taver.Always(int(value) == int(Value_Min), " +
		"\"The value is the only member.\")"
	if diagnosed(check_fixture(t, integer_helper_source(singleton)),
		"does not call a canonical helper") {
		t.Fatal("a direct singleton Always must satisfy the integer mandate")
	}
	// A float has no builder preset, thus Always is all it can state.
	float_singleton := "\taver.Always(float64(value) == float64(Value_Min), \"only\")"
	if diagnosed(check_fixture(t, float_helper_source(float_singleton)),
		"does not call a canonical helper") {
		t.Fatal("a float singleton must satisfy the scalar mandate")
	}
	// A Boolean has two legal values, thus a Tree with one Sometimes states the whole type. A
	// singleton Always would claim the type is constant, which the library rejects outright.
	boolean := "\taver.Tree(value, namespace).Sometimes(bool(value), \"set\").Ensure()"
	if diagnosed(check_fixture(t, boolean_helper_source(boolean)),
		"does not call a canonical helper") {
		t.Fatal("one Sometimes must satisfy the Boolean mandate")
	}
	boolean_singleton := "\taver.Always(bool(value) == true, \"only\")"
	if !diagnosed(check_fixture(t, boolean_helper_source(boolean_singleton)),
		"does not call a canonical helper") {
		t.Fatal("a singleton Always must not satisfy the Boolean mandate")
	}
	scalar_helper_remedy_text(t)
}

// Test_Invariants_Count_Helper verifies a counted type requires an ensured Range_Int or Enum_Int
// over its own length, regardless of equivalent individual assertions.
func Test_Invariants_Count_Helper(t *testing.T) {
	t.Parallel()
	raw := "\taver.Always(len(value) >= Value_Min, \"min\")\n" +
		"\taver.Always(len(value) <= Value_Max, \"max\")\n" +
		"\taver.Tree(value, namespace).Sometimes(len(value) == Value_Min, \"min\")." +
		"Sometimes(len(value) == Value_Max, \"max\").Ensure()"
	if !diagnosed(check_fixture(t, count_helper_source(raw)),
		"Write a Range_Int family, an Enum_Int family") {
		t.Fatal("individual count assertions must not satisfy the helper mandate")
	}
	valid := "\taver.Tree(value, namespace)." +
		"Range_Holed_Int(len(value), Value_Min, Value_Max, 1, 2, 2, 2).Ensure()"
	if diagnosed(check_fixture(t, count_helper_source(valid)),
		"Write a Range_Int family, an Enum_Int family") {
		t.Fatal("Range_Holed_Int over the counted value must satisfy the mandate")
	}
	wrong_suffix := "\taver.Tree(value, namespace)." +
		"Range_Int64(int64(len(value)), int64(Value_Min), int64(Value_Max)).Ensure()"
	if !diagnosed(check_fixture(t, count_helper_source(wrong_suffix)),
		"Write a Range_Int family, an Enum_Int family") {
		t.Fatal("a differently typed Range helper must not substitute")
	}
	wrong_subject := "\taver.Tree(value, namespace)." +
		"Range_Int(len(value[:0]), Value_Min, Value_Max).Ensure()"
	if !diagnosed(check_fixture(t, count_helper_source(wrong_subject)),
		"Write a Range_Int family, an Enum_Int family") {
		t.Fatal("a Range_Int over another subject must not satisfy the mandate")
	}
	split := "\tassertions := aver.Tree(value, namespace)\n" +
		"\tassertions.Range_Int(len(value), Value_Min, Value_Max).Ensure()"
	if !diagnosed(check_fixture(t, count_helper_source(split)),
		"Write a Range_Int family, an Enum_Int family") {
		t.Fatal("a split builder must not satisfy the helper mandate")
	}
	singleton := "\taver.Always(len(value) == Value_Min, " +
		"\"The count is the only member.\")"
	if diagnosed(check_fixture(t, count_helper_source(singleton)),
		"Write a Range_Int family, an Enum_Int family") {
		t.Fatal("a direct singleton Always must satisfy the count mandate")
	}
}

// Test_Invariants_Helper_Constants verifies canonical helpers retain package-level constant
// identity for Range edges and Enum members.
func Test_Invariants_Helper_Constants(t *testing.T) {
	t.Parallel()
	inline_range := "\taver.Tree(value, namespace)." +
		"Range_Int(int(value), int(Value_Min), int(7)).Ensure()"
	if !diagnosed(check_fixture(t, integer_helper_source(inline_range)),
		"an argument that is not a package-level constant") {
		t.Fatal("an inline Range edge must not satisfy the helper mandate")
	}
	inline_enum := "\taver.Tree(value, namespace)." +
		"Enum_Int(int(value), int(Value_Min), int(7)).Ensure()"
	if !diagnosed(check_fixture(t, integer_helper_source(inline_enum)),
		"an argument that is not a package-level constant") {
		t.Fatal("an inline Enum member must not satisfy the helper mandate")
	}
	converted := "\taver.Tree(value, namespace)." +
		"Range_Holed_Int(int(value), int(Value_Min), int(Value_Max), 1, 2, 2, 2).Ensure()"
	if diagnosed(check_fixture(t, integer_helper_source(converted)),
		"an argument that is not a package-level constant") {
		t.Fatal("exactly converted package constants must satisfy the mandate")
	}
	shadowed := "\tValue_Min := 0\n" +
		"\taver.Tree(value, namespace)." +
		"Range_Int(int(value), Value_Min, Value_Max).Ensure()"
	if !diagnosed(check_fixture(t, integer_helper_source(shadowed)),
		"an argument that is not a package-level constant") {
		t.Fatal("a local shadow of a package constant must not satisfy the mandate")
	}
	inline_singleton := "\taver.Always(int(value) == 1, \"only\")"
	if !diagnosed(check_fixture(t, integer_helper_source(inline_singleton)),
		"an argument that is not a package-level constant") {
		t.Fatal("an inline singleton member must not satisfy the helper mandate")
	}
	// A qualified constant states the boundary it crosses, thus one shared bound serves every
	// type that names it and no type has to copy the number.
	qualified := "package fixture\n\n" +
		"import (\n\taver \"fixture/shared/sim/aver/default\"\n" +
		"\tbound \"fixture/other\"\n)\n\n" +
		"// Value is a fixture.\ntype Value int\n\n" +
		"// Value_Invariants is a fixture.\n" +
		"func Value_Invariants(value Value, namespace aver.Namespace) {\n" +
		"\taver.Tree(value, namespace).Range_Int(" +
		"int(value), bound.SPAN_MINIMUM, bound.SPAN_MAXIMUM).Ensure()\n}\n"
	if diagnosed(check_fixture(t, qualified),
		"an argument that is not a package-level constant") {
		t.Fatal("a qualified constant must satisfy the helper mandate")
	}
}

// Test_Invariants_Cross_Package_Identity proves same-spelled foreign constants and helpers cannot
// satisfy a local helper mandate.
func Test_Invariants_Cross_Package_Identity(t *testing.T) {
	t.Parallel()
	cross_package_constant_isolation(t)
	cross_package_helper_isolation(t)
}

// Test_Invariants_Helper_Identity verifies only a direct builder rooted at the actual aver
// package and the helper's namespace parameter satisfies the body mandate.
func Test_Invariants_Helper_Identity(t *testing.T) {
	t.Parallel()
	literal := "\taver.Assertions(\"manual\")." +
		"Range_Int(int(value), int(Value_Min), int(Value_Max)).Ensure()"
	if !diagnosed(check_fixture(t, integer_helper_source(literal)),
		"does not call a canonical helper") {
		t.Fatal("a literal namespace must not satisfy a helper template")
	}
	nested := "\tif true {\n\t\taver.Tree(value, namespace)." +
		"Range_Int(int(value), int(Value_Min), int(Value_Max)).Ensure()\n\t}"
	if !diagnosed(check_fixture(t, integer_helper_source(nested)),
		"does not call a canonical helper") {
		t.Fatal("a nested builder must not satisfy the direct helper mandate")
	}
	foreign := foreign_scalar_helper_source()
	if !diagnosed(check_fixture(t, foreign), "does not call a canonical helper") {
		t.Fatal("a foreign Assertions lookalike must not satisfy the mandate")
	}
	aliased := aliased_scalar_helper_source()
	if !diagnosed(check_fixture(t, aliased), "does not call a canonical helper") {
		t.Fatal("an import alias must not impersonate the literal aver qualifier")
	}
	parent := strings.Replace(integer_helper_source(
		"\taver.Tree(value, namespace).Range_Int("+
			"int(value), int(Value_Min), int(Value_Max)).Ensure()"), "/default", "", 1)
	if !diagnosed(check_fixture(t, parent), "does not call a canonical helper") {
		t.Fatal("the pure aver package must not impersonate its default builder")
	}
	recorder := "\taver.Recorder_Assertions(namespace)." +
		"Range_Int(int(value), int(Value_Min), int(Value_Max)).Ensure()"
	if !diagnosed(check_fixture(t, integer_helper_source(recorder)),
		"does not call a canonical helper") {
		t.Fatal("Recorder_Assertions must not satisfy the helper mandate")
	}
	legacy := "\taver.Dot_Product(namespace)." +
		"Range_Int(int(value), int(Value_Min), int(Value_Max)).Ensure()"
	if !diagnosed(check_fixture(t, integer_helper_source(legacy)),
		"does not call a canonical helper") {
		t.Fatal("the old Dot_Product root must not satisfy the helper mandate")
	}
	retired := "\taver.Assertions(namespace)." +
		"Range_Int(int(value), int(Value_Min), int(Value_Max)).Ensure()"
	if !diagnosed(check_fixture(t, integer_helper_source(retired)),
		"does not call a canonical helper") {
		t.Fatal("the old Assertions root must not satisfy the helper mandate")
	}
	conversion := "\tint := func(Value) int { return 0 }\n" +
		"\taver.Tree(value, namespace)." +
		"Range_Int(int(value), Value_Min, Value_Max).Ensure()"
	if !diagnosed(check_fixture(t, integer_helper_source(conversion)),
		"does not call a canonical helper") {
		t.Fatal("a local function shadowing the primitive conversion must not substitute")
	}
	count := "\tlen := func(Value) int { return 0 }\n" +
		"\taver.Tree(value, namespace)." +
		"Range_Int(len(value), Value_Min, Value_Max).Ensure()"
	if !diagnosed(check_fixture(t, count_helper_source(count)),
		"Write a Range_Int family, an Enum_Int family") {
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
		"does not call a canonical helper") {
		t.Fatal("a builder with exactly 70 expanded links must satisfy the mandate")
	}
	over_limit := assertions_builder_body(71)
	if !diagnosed(check_fixture(t, integer_helper_source(over_limit)),
		"does not call a canonical helper") {
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
			"import (\n\taver \"fixture/shared/sim/aver/default\"\n" +
			"\tforeign \"fixture/foreign\"\n)\n\n" +
			"const Token_Min = 0\n\nconst Token_Max = 8\n\n" +
			"// Token is a fixture.\ntype Token string\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(v Token, namespace aver.Namespace) {\n" +
			"\taver.Tree(v, namespace)." +
			"Range_Int(len(v), Token_Min, Token_Max).Ensure()\n}\n\n" +
			"// Lexeme is a fixture.\ntype Lexeme struct {\n" +
			"\t// Tok is a fixture.\n\tTok Token\n}\n\n" +
			"// Lexeme_Invariants is a fixture.\n" +
			"func Lexeme_Invariants(v Lexeme, namespace aver.Namespace) {\n" +
			"\tforeign.Token_Invariants(v.Tok, \"wrong\")\n" +
			"\tif false { Token_Invariants(v.Tok, \"nested\") }\n" +
			"\tToken_Invariants := func(Token, aver.Namespace) {}\n" +
			"\tToken_Invariants(v.Tok, namespace)\n" +
			"\taver.Tree(v, namespace).Sometimes(true, \"x\").Ensure()\n}\n\n" +
			"// Phrase is a fixture.\ntype Phrase struct {\n" +
			"\t// Tok is a fixture.\n\tTok *Token\n}\n\n" +
			"// Phrase_Invariants is a fixture.\n" +
			"func Phrase_Invariants(v Phrase, namespace aver.Namespace) {\n" +
			"\taver.Tree(v, namespace)." +
			"Sometimes(true, \"y\").Ensure()\n}\n"})
	diags := check_source(pf)
	if !diagnosed(diags, "Lexeme_Invariants does not call a helper for the field v.Tok") {
		t.Fatal("a struct that does not compose a value field's invariant must be flagged")
	}
	// A pointer field composes its pointee, the same as a value field of that type,
	// so a bundle that omits it is flagged too.
	if !diagnosed(diags, "Phrase_Invariants does not call a helper for the field v.Tok") {
		t.Fatal("a struct that does not compose a pointer field's pointee must be flagged")
	}
	external := parse(t, &parse_input{
		Path: "other/token.go",
		Source_Text: "package other\n\n" +
			"import aver \"fixture/shared/sim/aver/default\"\n\n" +
			"const Token_Min = 0\n\nconst Token_Max = 8\n\n" +
			"// Token is a fixture.\ntype Token string\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(v Token, namespace aver.Namespace) {\n" +
			"\taver.Tree(v, namespace)." +
			"Range_Int(len(v), Token_Min, Token_Max).Ensure()\n}\n"})
	composed := parse(t, &parse_input{
		Path: "pkg/composed.go",
		Source_Text: "package fixture\n\n" +
			"import (\n\taver \"fixture/shared/sim/aver/default\"\n" +
			"\texternal \"fixture/other\"\n)\n\n" +
			"const Count_Min = 0\n\nconst Count_Max = 4\n\n" +
			"// Count is a fixture.\ntype Count int\n\n" +
			"// Count_Invariants is a fixture.\n" +
			"func Count_Invariants(value Count, namespace aver.Namespace) {\n" +
			"\taver.Tree(value, namespace)." +
			"Range_Int(int(value), int(Count_Min), int(Count_Max)).Ensure()\n}\n\n" +
			"// Holder is a fixture.\ntype Holder struct {\n" +
			"\t// Token is external.\n\tToken external.Token\n" +
			"\t// Count is local.\n\tCount Count\n}\n\n" +
			"// Holder_Invariants is a fixture.\n" +
			"func Holder_Invariants(v Holder, namespace aver.Namespace) {\n" +
			"\texternal.Token_Invariants(v.Token, \"token\")\n" +
			"\tCount_Invariants(v.Count, \"count\")\n}\n"})
	if diagnosed(check_sources([]source.Parsed_File{external, composed}),
		"does not call a helper") {
		t.Fatal("an aliased cross-package helper and a local helper must compose")
	}
}

// Test_Invariants_Inherited_Fields verifies a defined type over a struct states an inherited scalar
// field inline and composes an inherited struct field, which has no inline form.
func Test_Invariants_Inherited_Fields(t *testing.T) {
	t.Parallel()
	assert_inherited_scalar_is_inline(t)
	assert_inherited_struct_is_composed(t)
}

// Test_Invariants_Inline_Form verifies a Sometimes states no domain, thus it counts for a Boolean
// field and for nothing else.
func Test_Invariants_Inline_Form(t *testing.T) {
	t.Parallel()
	loose := INHERITED_FIELD_HEAD + "// Kept_Invariants is a fixture.\n" +
		"func Kept_Invariants(v Kept, namespace aver.Namespace) {\n" +
		INHERITED_STRUCT_LINK + "\taver.Tree(v, namespace)." +
		"Sometimes(int(v.Mk) == Mark_Min, \"the mark is least\").Ensure()\n}\n"
	if !diagnosed(check_source(parse(t, &parse_input{
		Path: "pkg/rule.go", Source_Text: loose})),
		"does not assert the inherited field v.Mk inline") {
		t.Fatal("a Sometimes over a bounded field must not state it")
	}
	// A comparison holds one side of the domain, and a literal names no shared fact.
	for _, condition := range []string{
		"int(v.Mk) > Mark_Min", "int(v.Mk) == 0", "int(v.Mk) != Mark_Min"} {
		partial := INHERITED_FIELD_HEAD + "// Kept_Invariants is a fixture.\n" +
			"func Kept_Invariants(v Kept, namespace aver.Namespace) {\n" +
			"\taver.Always(" + condition + ", \"partial\")\n" +
			INHERITED_STRUCT_LINK + "}\n"
		if !diagnosed(check_source(parse(t, &parse_input{
			Path: "pkg/rule.go", Source_Text: partial})),
			"does not assert the inherited field v.Mk inline") {
			t.Fatalf("%q must not state the field", condition)
		}
	}
	boolean := "package fixture\n\n" +
		"import aver \"fixture/shared/sim/aver/default\"\n\n" +
		"// Flag is a fixture.\ntype Flag bool\n\n" +
		"// Flag_Invariants is a fixture.\n" +
		"func Flag_Invariants(v Flag, namespace aver.Namespace) {\n" +
		"\taver.Tree(v, namespace).Sometimes(bool(v), \"set\").Ensure()\n}\n\n" +
		"// Switchboard is a fixture.\ntype Switchboard struct {\n" +
		"\t// Flg is a fixture.\n\tFlg Flag\n}\n\n" +
		"// Switchboard_Invariants is a fixture.\n" +
		"func Switchboard_Invariants(v Switchboard, namespace aver.Namespace) {\n" +
		"\tFlag_Invariants(v.Flg, namespace)\n}\n\n" +
		"// Panel is a fixture.\ntype Panel Switchboard\n\n" +
		"// Panel_Invariants is a fixture.\n" +
		"func Panel_Invariants(v Panel, namespace aver.Namespace) {\n" +
		"\taver.Tree(v, namespace)." +
		"Sometimes(bool(v.Flg), \"set\").Ensure()\n}\n"
	if diagnosed(check_source(parse(t, &parse_input{
		Path: "pkg/rule.go", Source_Text: boolean})), "v.Flg") {
		t.Fatal("a Sometimes must state an inherited Boolean field")
	}
}

// Test_Invariants_Defined_Pointers verifies a defined pointer helper holds exactly the nil exit
// and the pointee helper on the dereferenced value, for struct and non-struct pointees alike.
func Test_Invariants_Defined_Pointers(t *testing.T) {
	t.Parallel()
	assert_defined_pointer_body(t)
	assert_defined_pointer_pointee(t)
	assert_defined_pointer_foreign(t)
}

// Test_Invariants_Embedded_Fields verifies an anonymous field still owes its helper, under the name
// of the type it embeds and through a pointer.
func Test_Invariants_Embedded_Fields(t *testing.T) {
	t.Parallel()
	head := "package fixture\n\n" +
		"import aver \"fixture/shared/sim/aver/default\"\n\n" +
		"const Mark_Min = 0\n\nconst Mark_Max = 8\n\n" +
		"// Mark is a fixture.\ntype Mark int\n\n" +
		"// Mark_Invariants is a fixture.\n" +
		"func Mark_Invariants(v Mark, namespace aver.Namespace) {\n" +
		"\taver.Tree(v, namespace)." +
		"Range_Int(int(v), Mark_Min, Mark_Max).Ensure()\n}\n\n" +
		"// Frame is a fixture.\ntype Frame struct {\n" +
		"\t// Mk is a fixture.\n\tMk Mark\n}\n\n" +
		"// Frame_Invariants is a fixture.\n" +
		"func Frame_Invariants(v Frame, namespace aver.Namespace) {\n" +
		"\tMark_Invariants(v.Mk, namespace)\n}\n\n" +
		"// Reference is a fixture.\ntype Reference struct {\n\t*Frame\n}\n\n"
	bare := head + "// Reference_Invariants is a fixture.\n" +
		"func Reference_Invariants(v Reference, namespace aver.Namespace) {\n" +
		"\taver.Always(v.Frame != nil, \"the frame is present\")\n}\n"
	want := "Call Frame_Invariants(v.Frame, ...)."
	if !diagnosed(check_source(parse(t, &parse_input{
		Path: "pkg/rule.go", Source_Text: bare})), want) {
		t.Fatal("an embedded pointer that states only its presence must be flagged")
	}
	composed := head + "// Reference_Invariants is a fixture.\n" +
		"func Reference_Invariants(v Reference, namespace aver.Namespace) {\n" +
		"\taver.Always(v.Frame != nil, \"the frame is present\")\n" +
		"\tFrame_Invariants(*v.Frame, namespace)\n}\n"
	if diagnosed(check_source(parse(t, &parse_input{
		Path: "pkg/rule.go", Source_Text: composed})), "Call Frame_Invariants") {
		t.Fatal("an embedded pointer composed by its pointee must be accepted")
	}
}

// Test_Invariants_Always_Condition verifies a compound Always condition is flagged, because it
// collapses a Range or an Enum into one guard and drops their branch obligations.
func Test_Invariants_Always_Condition(t *testing.T) {
	t.Parallel()
	bound := integer_helper_source(
		"\taver.Always(int(value) >= int(Value_Min) && " +
			"int(value) <= int(Value_Max), \"in range\")")
	if !diagnosed(check_fixture(t, bound), "The Always condition has the compound operator") {
		t.Fatal("a hand-written Range must be flagged")
	}
	membership := integer_helper_source(
		"\taver.Always(int(value) == int(Value_Min) || " +
			"int(value) == int(Value_Max), \"a member\")")
	if !diagnosed(check_fixture(t, membership),
		"The Always condition has the compound operator") {
		t.Fatal("a hand-written Enum must be flagged")
	}
	single := integer_helper_source(
		"\taver.Always(int(value) == int(Value_Min), \"the only member\")")
	if diagnosed(check_fixture(t, single), "The Always condition has the compound operator") {
		t.Fatal("a single-term Always must be accepted")
	}
}

// Test_Invariants_Parameter_Helper verifies only the input type's exact helper
// satisfies the leading helper requirement.
func Test_Invariants_Parameter_Helper(t *testing.T) {
	t.Parallel()
	pf := parse(t, &parse_input{
		Path: "pkg/rule.go",
		Source_Text: "package fixture\n\n" +
			"import (\n\taver \"fixture/shared/sim/aver/default\"\n" +
			"\tforeign \"fixture/foreign\"\n)\n\n" +
			"const Token_Min = 0\n\nconst Token_Max = 8\n\n" +
			"// Token is a fixture.\ntype Token string\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(v Token, namespace aver.Namespace) {\n" +
			"\taver.Tree(v, namespace)." +
			"Range_Int(len(v), Token_Min, Token_Max).Ensure()\n}\n\n" +
			"// Consume does.\nfunc Consume(tok Token) {\n" +
			"\tforeign.Token_Invariants(tok, \"wrong\")\n" +
			"\taver.Always(true, \"raw guard\")\n" +
			"\taver.Assertions(\"raw builder\")." +
			"Sometimes(true, \"raw axis\").Ensure()\n" +
			"\tprintln(0)\n}\n"})
	if !diagnosed(check_source(pf), "does not call a helper for the input tok") {
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
			"import (\n\taver \"fixture/shared/sim/aver/default\"\n" +
			"\tforeign \"fixture/foreign\"\n)\n\n" +
			"const Token_Min = 0\n\nconst Token_Max = 8\n\n" +
			"// Token is a fixture.\ntype Token string\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(v Token, namespace aver.Namespace) {\n" +
			"\taver.Tree(v, namespace)." +
			"Range_Int(len(v), Token_Min, Token_Max).Ensure()\n}\n\n" +
			"// Make does.\nfunc Make() (tok Token) {\n\tdefer func() {\n" +
			"\t\tforeign.Token_Invariants(tok, \"wrong\")\n" +
			"\t\tif false { Token_Invariants(tok, \"nested\") }\n" +
			"\t\tToken_Invariants := func(Token, aver.Namespace) {}\n" +
			"\t\tToken_Invariants(tok, \"shadowed\")\n" +
			"\t\taver.Always(true, \"raw guard\")\n\t}()\n\treturn \"\"\n}\n"})
	if !diagnosed(check_source(pf), "does not call a helper for the output tok. "+
		"Call Token_Invariants(tok, ...) in the output defer.") {
		t.Fatal("foreign and direct assertions must not satisfy the output helper")
	}
	correct := parse(t, &parse_input{
		Path: "pkg/correct_output.go",
		Source_Text: "package fixture\n\n" +
			"import aver \"fixture/shared/sim/aver/default\"\n\n" +
			"const Token_Min = 0\n\nconst Token_Max = 8\n\n" +
			"// Token is a fixture.\ntype Token string\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(v Token, namespace aver.Namespace) {\n" +
			"\taver.Tree(v, namespace)." +
			"Range_Int(len(v), Token_Min, Token_Max).Ensure()\n}\n\n" +
			"// Make uses the exact output helper.\nfunc Make() (tok Token) {\n" +
			"\tdefer func() {\n\t\tToken_Invariants(tok, \"token\")\n" +
			"\t}()\n\treturn \"\"\n}\n"})
	if diagnosed(check_source(correct), "does not call a helper") {
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
	if !diagnosed(recorder_diagnostics(files),
		"has no TestMain that calls aver.Run_Test_Main") {
		t.Fatal("a package with no TestMain must be flagged")
	}
}

// Test_Invariants_Raw_Types verifies boundary syntax alone decides ownership: identifiers name
// types, while every constructed expression needs declaration first.
func Test_Invariants_Raw_Types(t *testing.T) {
	t.Parallel()
	parsed := parse(t, &parse_input{
		Path: "pkg/rule.go",
		Source_Text: "package fixture\n\n" +
			"import outside \"fixture/outside\"\n\n" +
			"// Token names fixture type.\ntype Token int\n\n" +
			"// Holder names fixture owner.\ntype Holder struct {\n" +
			"\tLocal Token\n\tQualified outside.Token\n\tBuiltin int\n" +
			"\tPointer *Token\n\tInline struct { Value Token }\n}\n\n" +
			"// Convert exercises type boundaries.\n" +
			"func Convert(local Token, qualified outside.Token, builtin string, " +
			"raw *Token) " +
			"(local_result Token, qualified_result outside.Token, raw_result *Token, " +
			"builtin_result int32, failure error) {\n" +
			"\tprintln(local, qualified, builtin, raw)\n" +
			"\treturn local, qualified, raw, 0, nil\n}\n",
	})
	diags := check_source(parsed)
	if !diagnosed(diags, "The declaration Convert has a raw type parameter (raw). "+
		"Declare a defined type for the parameter.") {
		t.Fatal("constructed parameter type must be flagged")
	}
	if !diagnosed(diags, "The declaration Convert has a raw type result (raw_result). "+
		"Declare a defined type for the result.") {
		t.Fatal("constructed result type must be flagged")
	}
	if !diagnosed(diags, "The declaration Holder has a raw type field (Pointer). "+
		"Declare a defined type for the field.") {
		t.Fatal("constructed field type must be flagged")
	}
	if !diagnosed(diags, "The declaration Holder has a raw type field (Inline).") {
		t.Fatal("inline struct field must be flagged")
	}
	// Predeclared identifier is raw too: int carries no _Invariants and never can.
	if !diagnosed(diags, "The declaration Holder has a raw type field (Builtin). "+
		"Declare a defined type for the field.") {
		t.Fatal("predeclared field type must be flagged")
	}
	if !diagnosed(diags, "The declaration Convert has a raw type parameter (builtin). "+
		"Declare a defined type for the parameter.") {
		t.Fatal("predeclared parameter type must be flagged")
	}
	if !diagnosed(diags, "The declaration Convert has a raw type result (builtin_result). "+
		"Declare a defined type for the result.") {
		t.Fatal("predeclared result type must be flagged")
	}
	// An error is an interface with no length and no domain, thus no assertion could state it.
	for _, accepted := range []string{"Local", "Qualified", "local", "qualified",
		"local_result", "qualified_result", "failure"} {
		for _, entry := range diags {
			if !strings.Contains(entry.Message, "has a raw ") {
				continue
			}
			if strings.Contains(entry.Message, "("+accepted+")") {
				t.Fatalf("identifier type %s must pass", accepted)
			}
		}
	}
}

// Test_Invariants_Small_Slices verifies a defined slice type whose helper bounds len to at most 8
// is banned as a field, parameter, or result, whatever form the helper takes, and a larger or
// unresolved bound is not.
func Test_Invariants_Small_Slices(t *testing.T) {
	t.Parallel()
	ranged := "\taver.Tree(members, namespace).Range_Int(len(members), MEMBERS_MIN, MEMBERS_MAX).Ensure()"
	if !diagnosed(check_fixture(t, small_slice_source(ranged, "8")), small_slice_message("8")) {
		t.Fatal("a Range_Int upper bound of 8 must ban the slice field")
	}
	if diagnosed(check_fixture(t, small_slice_source(ranged, "9")), "Holder has a Members field") {
		t.Fatal("a Range_Int upper bound of 9 must keep the slice field")
	}
	chained := "\taver.Tree(members, namespace).Range_Int(len(members), MEMBERS_MIN, MEMBERS_CAP).Ensure()"
	chain := small_slice_source(chained, "4") + "\n// MEMBERS_CAP is a fixture.\nconst MEMBERS_CAP = MEMBERS_MAX\n"
	if !diagnosed(check_fixture(t, chain), small_slice_message("4")) {
		t.Fatal("a bound reached through a chain of constants must ban the slice field")
	}
	enum := "\taver.Tree(members, namespace).Enum_Int(len(members), MEMBERS_MIN, MEMBERS_MAX).Ensure()"
	if !diagnosed(check_fixture(t, small_slice_source(enum, "3")), small_slice_message("3")) {
		t.Fatal("an Enum_Int largest member of 3 must ban the slice field")
	}
	singleton := "\taver.Always(len(members) == MEMBERS_MAX, \"fixed\")"
	if !diagnosed(check_fixture(t, small_slice_source(singleton, "2")), small_slice_message("2")) {
		t.Fatal("an Always singleton of 2 must ban the slice field")
	}
	unresolved := small_slice_source(ranged, "int(len(\"abc\"))")
	if diagnosed(check_fixture(t, unresolved), "Holder has a Members field") {
		t.Fatal("a bound that resolves to no integer must leave the slice field unjudged")
	}
	diags := check_fixture(t, small_slice_source(ranged, "8"))
	if !diagnosed(diags, "The declaration Take has a Members parameter (members) bounded to "+
		"8 members. Declare a struct with one field per member instead.") {
		t.Fatal("a small slice parameter must be flagged")
	}
	if !diagnosed(diags, "The declaration Take has a Members result (out) bounded to "+
		"8 members. Declare a struct with one field per member instead.") {
		t.Fatal("a small slice result must be flagged")
	}
	if !diagnosed(diags, "The declaration Members_Invariants has a Members parameter (members) "+
		"bounded to 8 members.") {
		t.Fatal("the ban is blanket, thus the helper's own parameter is flagged")
	}
	test_file := parse(t, &parse_input{Path: "pkg/rule_test.go", Source_Text: small_slice_source(ranged, "8")})
	if !diagnosed(check_source(test_file), small_slice_message("8")) {
		t.Fatal("the ban is blanket, thus a _test.go file is flagged")
	}
}

// Test_Invariants_Fixed_Arrays verifies a fixed array, raw or through a defined type, is banned
// as a field, parameter, or result whatever its length and owner, while a local variable is
// untouched.
func Test_Invariants_Fixed_Arrays(t *testing.T) {
	t.Parallel()
	code := "package fixture\n\n" +
		"import aver \"fixture/shared/sim/aver/default\"\n\n" +
		"// WIDTH is a fixture.\nconst WIDTH = 64\n\n" +
		"// Cell is a fixture.\ntype Cell int\n\n" +
		"// Cell_Invariants is a fixture.\n" +
		"func Cell_Invariants(cell Cell, namespace aver.Namespace) {\n" +
		"\taver.Always(int(cell) == WIDTH, \"fixed\")\n}\n\n" +
		"// Row is a fixture.\ntype Row [WIDTH]Cell\n\n" +
		"// Row_Invariants is a fixture.\n" +
		"func Row_Invariants(row Row, namespace aver.Namespace) {\n" +
		"\taver.Always(len(row) == WIDTH, \"fixed\")\n}\n\n" +
		"// Grid is a fixture.\ntype Grid struct {\n" +
		"\t// Raw is a fixture.\n\tRaw [WIDTH]Cell\n" +
		"\t// Named is a fixture.\n\tNamed Row\n" +
		"\t// Pointed is a fixture.\n\tPointed *Row\n}\n\n" +
		"// Grid_Invariants is a fixture.\n" +
		"func Grid_Invariants(grid Grid, namespace aver.Namespace) {\n" +
		"\tRow_Invariants(grid.Named, namespace)\n" +
		"\tRow_Invariants(*grid.Pointed, namespace)\n}\n\n" +
		"// Fill is a fixture.\n" +
		"func Fill(row Row) (out [WIDTH]Cell) {\n" +
		"\tvar scratch [WIDTH]Cell\n\treturn scratch\n}\n"
	diags := check_fixture(t, code)
	remedy := " Convert fixed arrays to slices instead."
	if !diagnosed(diags, "The declaration Grid has a fixed array field (Raw)."+remedy) {
		t.Fatal("a raw fixed array field must be flagged")
	}
	if !diagnosed(diags, "The declaration Grid has a fixed array field (Named)."+remedy) {
		t.Fatal("a defined type over a fixed array must be flagged as a field")
	}
	if !diagnosed(diags, "The declaration Grid has a fixed array field (Pointed)."+remedy) {
		t.Fatal("a pointer to a fixed array type must be flagged as a field")
	}
	if !diagnosed(diags, "The declaration Fill has a fixed array parameter (row)."+remedy) {
		t.Fatal("a defined array parameter must be flagged")
	}
	if !diagnosed(diags, "The declaration Fill has a fixed array result (out)."+remedy) {
		t.Fatal("a raw fixed array result must be flagged")
	}
	if !diagnosed(diags, "The declaration Row_Invariants has a fixed array parameter (row)."+remedy) {
		t.Fatal("the ban is blanket, thus the helper's own array parameter is flagged")
	}
	stringer := "package fixture\n\n" +
		"// WIDTH is a fixture.\nconst WIDTH = 2\n\n" +
		"// Pair is a fixture.\ntype Pair [WIDTH]byte\n\n" +
		"// String is a fixture.\nfunc (pair Pair) String() (text string) {\n\treturn \"\"\n}\n\n" +
		"// Read is a fixture.\nfunc Read(destination Pair) (count int, err error) {\n\treturn 0, nil\n}\n"
	if !diagnosed(check_source(parse(t, &parse_input{Path: "pkg/rule_test.go", Source_Text: stringer})),
		"The declaration Read has a fixed array parameter (destination)."+remedy) {
		t.Fatal("the ban is blanket, thus a _test.go file is flagged")
	}
	if diagnosed(diags, "scratch") {
		t.Fatal("a local array variable is not a field, parameter, or result")
	}
}

// Test_Simulation_Presence verifies a binary component with a non-exempt internal
// package but no simulation_test package is flagged.
func Test_Simulation_Presence(t *testing.T) {
	t.Parallel()
	if !diagnosed(simulation_diagnostics(simulation_base(t)),
		"has no internal/simulation_test package") {
		t.Fatal("a binary component without a simulation package must be flagged")
	}
}

// Test_Simulation_Contents verifies a simulation package with no Fuzz function is
// flagged: the fuzz driver must be present.
func Test_Simulation_Contents(t *testing.T) {
	t.Parallel()
	sim := "package simulation_test\n\nimport (\n\t\"testing\"\n\n" +
		"\taver \"fixture/shared/sim/aver/default\"\n)\n\n" +
		"func TestMain(m *testing.M) {\n" +
		"\taver.Run_Test_Main(m, \"../**\")\n}\n"
	if !diagnosed(simulation_diagnostics(simulation_files(t, sim)),
		"The simulation package has no fuzz function for internal.Main.") {
		t.Fatal("a simulation package without a fuzz function must be flagged")
	}
}

// Test_Simulation_Test_Main verifies a simulation TestMain that is not exactly
// aver.Run_Test_Main(m, <dirs>) is flagged.
func Test_Simulation_Test_Main(t *testing.T) {
	t.Parallel()
	sim := simulation_fixture_source("aver.Run_Test_Main(m)")
	if !diagnosed(simulation_diagnostics(simulation_files(t, sim)),
		"The body of the simulation TestMain is not "+
			"aver.Run_Test_Main(m, \"../**\").") {
		t.Fatal("a simulation TestMain without directory arguments must be flagged")
	}
}

// Test_Simulation_Coverage verifies a simulation TestMain whose directory arguments
// do not register every internal package is flagged.
func Test_Simulation_Coverage(t *testing.T) {
	t.Parallel()
	sim := simulation_fixture_source("aver.Run_Test_Main(m, \"../*\")")
	if !diagnosed(simulation_diagnostics(simulation_files(t, sim)),
		"The body of the simulation TestMain is not "+
			"aver.Run_Test_Main(m, \"../**\").") {
		t.Fatal("a narrower glob that omits nested internal packages must be flagged")
	}
}

// Test_Simulation_Entry verifies the simulation is flagged when it references an
// internal function other than Main.
func Test_Simulation_Entry(t *testing.T) {
	t.Parallel()
	sim := "package simulation_test\n\nimport (\n\t\"testing\"\n\n" +
		"\t\"github.com/james-orcales/james-orcales/pkg/internal\"\n" +
		"\taver \"fixture/shared/sim/aver/default\"\n)\n\n" +
		"func TestMain(m *testing.M) {\n" +
		"\taver.Run_Test_Main(m, \"../**\")\n}\n\n" +
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
	if !diagnosed(simulation_diagnostics(files),
		"The simulation package refers to internal.Extra. Refer only to Main.") {
		t.Fatal("referencing a non-Main internal function must be flagged")
	}
}

// Test_Simulation_Blackbox verifies a simulation package that is not an external
// _test package is flagged.
func Test_Simulation_Blackbox(t *testing.T) {
	t.Parallel()
	sim := "package simulation\n\nimport (\n\t\"testing\"\n\n" +
		"\taver \"fixture/shared/sim/aver/default\"\n)\n\n" +
		"func TestMain(m *testing.M) {\n" +
		"\taver.Run_Test_Main(m, \"../**\")\n}\n\n" +
		"func Fuzz_Main(f *testing.F) {\n\tf.Fuzz(func(t *testing.T, data []byte) {})\n}\n"
	if !diagnosed(simulation_diagnostics(simulation_files(t, sim)),
		"The simulation package simulation is not an external test package.") {
		t.Fatal("a whitebox simulation package must be flagged")
	}
}

// The remedy a diagnostic names must exist. Every primitive preset is gone, and only an integer
// has a Range or an Enum family, thus one generated list cannot serve all three kinds.
func scalar_helper_remedy_text(t *testing.T) {
	t.Helper()
	empty := "\tprintln(0)"
	integer := check_fixture(t, integer_helper_source(empty))
	if diagnosed(integer, "Int_Invariants,") {
		t.Fatal("the integer remedy must not name a deleted preset")
	}
	if !diagnosed(integer,
		"Write an Always(<var> == <const>), a Range_Int family, or an Enum_Int family.") {
		t.Fatal("the integer remedy must show the Always equality shape")
	}
	if !diagnosed(integer, "Range_Int family") {
		t.Fatal("the integer remedy must name the Range_Int family")
	}
	if !diagnosed(integer, "Enum_Int family") {
		t.Fatal("the integer remedy must name the Enum_Int family")
	}
	float_diags := check_fixture(t, float_helper_source(empty))
	for _, absent := range []string{
		"Float64_Invariants", "Range_Float64", "Enum_Float64"} {
		if diagnosed(float_diags, absent) {
			t.Fatalf("the float remedy must not name %s", absent)
		}
	}
	if !diagnosed(float_diags, "Always") {
		t.Fatal("the float remedy must name direct Always equality")
	}
	boolean := check_fixture(t, boolean_helper_source(empty))
	for _, absent := range []string{
		"Boolean_Invariants", "Range_Boolean", "Enum_Boolean", "Always"} {
		if diagnosed(boolean, absent) {
			t.Fatalf("the Boolean remedy must not name %s", absent)
		}
	}
	if !diagnosed(boolean, "Sometimes") {
		t.Fatal("the Boolean remedy must name a Tree whose one link is a Sometimes")
	}
}

func scalar_range_holed_helper(t *testing.T) {
	t.Helper()
	body := "\taver.Tree(value, namespace)." +
		"Range_Holed_Int(int(value), int(Value_Min), int(Value_Max), " +
		"int(Value_Third), int(Value_Fourth), int(Value_Fourth), " +
		"int(Value_Fourth)).Ensure()"
	if diagnosed(check_fixture(t, integer_helper_source(body)),
		"does not call a canonical helper") {
		t.Fatal("the exact Range_Holed helper must satisfy the scalar mandate")
	}
}

func assert_invalid_singleton_helper_identity(t *testing.T) {
	t.Helper()
	invalid_singletons := []string{
		"\taver.Always(int(Value_Min) == int(value), \"reversed\")",
		"\taver.Always(int(value) != int(Value_Min), \"not equal\")",
		"\tmessage := \"dynamic\"\n" +
			"\taver.Always(int(value) == int(Value_Min), message)",
		"\taver.Always(int(value + 1) == int(Value_Min), \"wrong subject\")",
		"\tif true { aver.Always(int(value) == int(Value_Min), \"nested\") }",
	}
	for _, body := range invalid_singletons {
		if !diagnosed(check_fixture(t, integer_helper_source(body)),
			"does not call a canonical helper") {
			t.Fatalf("noncanonical singleton satisfied the helper mandate: %s", body)
		}
	}
	parent_singleton := strings.Replace(integer_helper_source(
		"\taver.Always(int(value) == int(Value_Min), \"only\")"),
		"/default", "", 1)
	if !diagnosed(check_fixture(t, parent_singleton), "does not call a canonical helper") {
		t.Fatal("the pure aver package must not provide the singleton helper")
	}
	aliased_singleton := strings.Replace_All(integer_helper_source(
		"\taver.Always(int(value) == int(Value_Min), \"only\")"),
		"aver", "contract")
	if !diagnosed(check_fixture(t, aliased_singleton), "does not call a canonical helper") {
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
			"import aver \"fixture/shared/sim/aver/default\"\n\n" +
			"const Token_Min = 0\n\nconst Token_Max = 8\n\n" +
			"// Token is a fixture.\ntype Token string\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(v Token, namespace aver.Namespace) {\n" +
			"\taver.Tree(v, namespace)." +
			"Range_Int(len(v), Token_Min, Token_Max).Ensure()\n}\n\n" +
			"// Consume uses exact helpers.\nfunc Consume(tok Token) {\n" +
			"\tToken_Invariants(tok, \"token\")\n}\n"})
	if diagnosed(check_source(correct), "does not call a helper") {
		t.Fatal("an exact local input helper must satisfy the mandate")
	}
}

// A same-named parameter must not become an escape hatch from exact helper identity.
func parameter_helper_shadowed(t *testing.T) {
	t.Helper()
	shadowed := parse(t, &parse_input{
		Path: "pkg/shadowed.go",
		Source_Text: "package fixture\n\n" +
			"import aver \"fixture/shared/sim/aver/default\"\n\n" +
			"const Token_Min = 0\n\nconst Token_Max = 8\n\n" +
			"// Token is a fixture.\ntype Token string\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(v Token, namespace aver.Namespace) {\n" +
			"\taver.Tree(v, namespace)." +
			"Range_Int(len(v), Token_Min, Token_Max).Ensure()\n}\n\n" +
			"// Consume shadows the helper.\n" +
			"func Consume(tok Token, Token_Invariants " +
			"func(Token, aver.Namespace)) {\n" +
			"\tToken_Invariants(tok, \"shadowed\")\n}\n"})
	if !diagnosed(check_source(shadowed), "does not call a helper for the input tok") {
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
			"import aver \"fixture/shared/sim/aver/default\"\n\n" +
			"const Input_Min = -4\n\nconst Input_Max = 4\n\n" +
			"// Input is a fixture.\ntype Input int\n\n" +
			"// Input_Invariants is a fixture.\n" +
			"func Input_Invariants(v Input, namespace aver.Namespace) {\n" +
			"\taver.Tree(v, namespace).Range_Int(" +
			"int(v), int(Input_Min), int(Input_Max)).Ensure()\n}\n"})
	consumer := parse(t, &parse_input{
		Path: "pkg/external.go",
		Source_Text: "package fixture\n\n" +
			"import external \"fixture/other\"\n\n" +
			"// Consume uses an aliased external helper.\n" +
			"func Consume(input external.Input) {\n" +
			"\texternal.Input_Invariants(input, \"input\")\n}\n"})
	if diagnosed(check_sources([]source.Parsed_File{external, consumer}),
		"does not call a helper") {
		t.Fatal("an aliased exact cross-package input helper must satisfy the mandate")
	}
}

func field_subject_isolation(t *testing.T) {
	t.Helper()
	pf := parse(t, &parse_input{
		Path: "pkg/field_isolation.go",
		Source_Text: "package fixture\n\n" +
			"import aver \"fixture/shared/sim/aver/default\"\n\n" +
			"const Token_Min = 0\n\nconst Token_Max = 8\n\n" +
			"// Token is a fixture.\ntype Token int\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(value Token, namespace aver.Namespace) {\n" +
			"\taver.Tree(value, namespace)." +
			"Range_Int(int(value), Token_Min, Token_Max).Ensure()\n}\n\n" +
			"// Pair is a fixture.\ntype Pair struct {\n" +
			"\t// First is a fixture.\n\tFirst Token\n" +
			"\t// Second is a fixture.\n\tSecond Token\n}\n\n" +
			"// Pair_Invariants is a fixture.\n" +
			"func Pair_Invariants(value Pair, namespace aver.Namespace) {\n" +
			"\tToken_Invariants(value.First, \"first\")\n}\n"})
	diags := check_source(pf)
	if !diagnosed(diags, "The function Pair_Invariants does not call a helper for "+
		"the field value.Second. Call Token_Invariants(value.Second, ...).") {
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
			"import aver \"fixture/shared/sim/aver/default\"\n\n" +
			"const Token_Min = 0\n\nconst Token_Max = 8\n\n" +
			"// Token is a fixture.\ntype Token int\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(value Token, namespace aver.Namespace) {\n" +
			"\taver.Tree(value, namespace)." +
			"Range_Int(int(value), Token_Min, Token_Max).Ensure()\n}\n\n" +
			"// Consume is a fixture.\n" +
			"func Consume(first Token, second Token) {\n" +
			"\tToken_Invariants(first, \"first\")\n\tprintln(0)\n}\n"})
	diags := check_source(pf)
	want := "The function Consume does not call a helper for the input second. " +
		"Call Token_Invariants(second, ...) at the start of the body."
	if !diagnosed(diags, want) {
		t.Fatalf("first input call satisfied the second input: %v", diags)
	}
	if diagnosed(diags, "helper for the input first") {
		t.Fatalf("the exact first input call was rejected: %v", diags)
	}
}

func output_subject_isolation(t *testing.T) {
	t.Helper()
	pf := parse(t, &parse_input{
		Path: "pkg/output_isolation.go",
		Source_Text: "package fixture\n\n" +
			"import aver \"fixture/shared/sim/aver/default\"\n\n" +
			"const Token_Min = 0\n\nconst Token_Max = 8\n\n" +
			"// Token is a fixture.\ntype Token int\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(value Token, namespace aver.Namespace) {\n" +
			"\taver.Tree(value, namespace)." +
			"Range_Int(int(value), Token_Min, Token_Max).Ensure()\n}\n\n" +
			"// Make is a fixture.\nfunc Make() (first Token, second Token) {\n" +
			"\tdefer func() { Token_Invariants(first, \"first\") }()\n" +
			"\treturn 0, 0\n}\n"})
	diags := check_source(pf)
	if !diagnosed(diags, "The function Make does not call a helper for the output "+
		"second. Call Token_Invariants(second, ...) in the output defer.") {
		t.Fatalf("first output call satisfied the second output: %v", diags)
	}
	if diagnosed(diags, "helper for the output first") {
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
			"import aver \"fixture/shared/sim/aver/default\"\n\n" +
			"// Value is a fixture.\ntype Value int\n\n" +
			"// Value_Invariants is a fixture.\n" +
			"func Value_Invariants(value Value, namespace aver.Namespace) {\n" +
			"\taver.Tree(value, namespace).Range_Int(" +
			"int(value), Value_Min, Value_Max).Ensure()\n}\n"})
	diags := check_sources([]source.Parsed_File{foreign, local})
	if !diagnosed(diags, "an argument that is not a package-level constant") {
		t.Fatalf("foreign same-spelled constants satisfied the local helper: %v", diags)
	}
}

// A qualified name states the boundary it crosses, thus it hides no type. A defined type over one
// stands over the same string its far side does, and it owes the same count mandate.
func qualified_base_resolves(t *testing.T) {
	t.Helper()
	foreign := parse(t, &parse_input{
		Path: "other/target.go",
		Source_Text: "package other\n\n" +
			"import aver \"fixture/shared/sim/aver/default\"\n\n" +
			"const Span_Min = 0\n\nconst Span_Max = 8\n\n" +
			"// Request_Target is a fixture.\ntype Request_Target string\n\n" +
			"// Request_Target_Invariants is a fixture.\n" +
			"func Request_Target_Invariants(" +
			"value Request_Target, namespace aver.Namespace) {\n" +
			"\taver.Tree(value, namespace)." +
			"Range_Int(len(value), Span_Min, Span_Max).Ensure()\n}\n"})
	local := parse(t, &parse_input{
		Path: "pkg/parsed.go",
		Source_Text: "package fixture\n\n" +
			"import (\n\taver \"fixture/shared/sim/aver/default\"\n" +
			"\tproxy \"fixture/other\"\n)\n\n" +
			"const Span_Min = 0\n\nconst Span_Max = 8\n\n" +
			"// Parsed_Request_Target is a fixture.\n" +
			"type Parsed_Request_Target proxy.Request_Target\n\n" +
			"// Parsed_Request_Target_Invariants is a fixture.\n" +
			"func Parsed_Request_Target_Invariants(" +
			"value Parsed_Request_Target, namespace aver.Namespace) {\n" +
			"\taver.Always(len(value) >= Span_Min, \"min\")\n" +
			"\taver.Always(len(value) <= Span_Max, \"max\")\n}\n"})
	diags := check_sources([]source.Parsed_File{foreign, local})
	if !diagnosed(diags, "Write a Range_Int family, an Enum_Int family") {
		t.Fatalf("a qualified base hid the string it stands over: %v", diags)
	}
}

func cross_package_helper_isolation(t *testing.T) {
	t.Helper()
	foreign := parse(t, &parse_input{
		Path: "other/token.go",
		Source_Text: "package other\n\n" +
			"import aver \"fixture/shared/sim/aver/default\"\n\n" +
			"const Token_Min = 0\n\nconst Token_Max = 8\n\n" +
			"// Token is a fixture.\ntype Token int\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(value Token, namespace aver.Namespace) {\n" +
			"\taver.Tree(value, namespace)." +
			"Range_Int(int(value), Token_Min, Token_Max).Ensure()\n}\n"})
	local := parse(t, &parse_input{
		Path: "pkg/local_helper.go",
		Source_Text: "package fixture\n\n" +
			"import (\n\taver \"fixture/shared/sim/aver/default\"\n" +
			"\tforeign \"fixture/other\"\n)\n\n" +
			"const Token_Min = 0\n\nconst Token_Max = 8\n\n" +
			"// Token is a fixture.\ntype Token int\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(value Token, namespace aver.Namespace) {\n" +
			"\taver.Tree(value, namespace)." +
			"Range_Int(int(value), Token_Min, Token_Max).Ensure()\n}\n\n" +
			"// Consume is a fixture.\nfunc Consume(value Token) {\n" +
			"\tforeign.Token_Invariants(foreign.Token(value), \"foreign\")\n}\n"})
	diags := check_sources([]source.Parsed_File{foreign, local})
	want := "The function Consume does not call a helper for the input value. " +
		"Call Token_Invariants(value, ...) at the start of the body."
	if !diagnosed(diags, want) {
		t.Fatalf("foreign same-named helper satisfied the local subject: %v", diags)
	}
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
		"import aver \"fixture/shared/sim/aver/default\"\n\n" +
		"const Value_Min = -4\n\nconst Value_Third = -1\n\n" +
		"const Value_Fourth = 1\n\nconst Value_Max = 4\n\n" +
		"// Value is a fixture.\ntype Value int\n\n" +
		"// Value_Invariants is a fixture.\n" +
		"func Value_Invariants(value Value, namespace aver.Namespace) {\n" +
		body + "\n}\n"
}

// The named type a chained fixture's subject stands over.
type chained_subject string

// CHAINED_INTEGER stands over Reading_Representation, thus its kind is two names away.
const CHAINED_INTEGER chained_subject = "Reading"

// CHAINED_SLICE stands over a slice of a type that is itself two names from its kind.
const CHAINED_SLICE chained_subject = "Readings"

// Builds a helper whose subject stands over a named type, which stands over another named type.
// Only a walk to the end of that chain can see the kind the subject owes its mandate to.
func chained_helper_source(subject chained_subject, body string) (code string) {
	return "package fixture\n\n" +
		"import aver \"fixture/shared/sim/aver/default\"\n\n" +
		"const Span_Min = 0\n\nconst Span_Max = 8\n\n" +
		"// Reading_Representation is a fixture.\ntype Reading_Representation int\n\n" +
		"// Reading_Representation_Invariants is a fixture.\n" +
		"func Reading_Representation_Invariants(" +
		"v Reading_Representation, namespace aver.Namespace) {\n" +
		"\taver.Tree(v, namespace)." +
		"Range_Int(int(v), Span_Min, Span_Max).Ensure()\n}\n\n" +
		"// Reading is a fixture.\ntype Reading Reading_Representation\n\n" +
		"// Reading_Invariants is a fixture.\n" +
		"func Reading_Invariants(v Reading, namespace aver.Namespace) {\n" +
		"\taver.Tree(v, namespace)." +
		"Range_Int(int(v), Span_Min, Span_Max).Ensure()\n}\n\n" +
		"// Readings is a fixture.\ntype Readings []Reading\n\n" +
		"// Readings_Invariants is a fixture.\n" +
		"func Readings_Invariants(v Readings, namespace aver.Namespace) {\n" +
		"\taver.Tree(v, namespace)." +
		"Range_Int(len(v), Span_Min, Span_Max).Ensure()\n}\n\n" +
		"// Value is a fixture.\ntype Value " + string(subject) + "\n\n" +
		"// Value_Invariants is a fixture.\n" +
		"func Value_Invariants(v Value, namespace aver.Namespace) {\n" +
		body + "\n}\n"
}

// The float fixture is intentionally separate because type text is test data, not a runtime
// input whose bundling would improve the API.
func float_helper_source(body string) (code string) {
	return "package fixture\n\n" +
		"import aver \"fixture/shared/sim/aver/default\"\n\n" +
		"const Value_Min = -4\n\nconst Value_Max = 4\n\n" +
		"// Value is a fixture.\ntype Value float64\n\n" +
		"// Value_Invariants is a fixture.\n" +
		"func Value_Invariants(value Value, namespace aver.Namespace) {\n" +
		body + "\n}\n"
}

// The Boolean fixture remains explicit so no generic fixture input structure can conceal which
// primitive preset the leaf is proving.
func boolean_helper_source(body string) (code string) {
	return "package fixture\n\n" +
		"import aver \"fixture/shared/sim/aver/default\"\n\n" +
		"const Value_Min = -4\n\nconst Value_Max = 4\n\n" +
		"// Value is a fixture.\ntype Value bool\n\n" +
		"// Value_Invariants is a fixture.\n" +
		"func Value_Invariants(value Value, namespace aver.Namespace) {\n" +
		body + "\n}\n"
}

// Builds a defined byte-slice helper, the same counted shape as Report_Invariants.
func count_helper_source(body string) (code string) {
	return "package fixture\n\n" +
		"import aver \"fixture/shared/sim/aver/default\"\n\n" +
		"const Value_Min = 0\n\nconst Value_Max = 32\n\n" +
		"// Value is a fixture.\ntype Value []byte\n\n" +
		"// Value_Invariants is a fixture.\n" +
		"func Value_Invariants(value Value, namespace aver.Namespace) {\n" +
		body + "\n}\n"
}

// The field diagnostic the slice fixture draws once its bound resolves at or under the cap.
func small_slice_message(bound string) (message string) {
	return "The declaration Holder has a Members field (Items) bounded to " + bound +
		" members. Declare a struct with one field per member instead."
}

// A slice fixture whose len bound is the one free variable, so each leaf varies only the helper
// form and the bound under judgment.
func small_slice_source(body string, bound string) (code string) {
	return "package fixture\n\n" +
		"import aver \"fixture/shared/sim/aver/default\"\n\n" +
		"// MEMBERS_MIN is a fixture.\nconst MEMBERS_MIN = 0\n\n" +
		"// MEMBERS_MAX is a fixture.\nconst MEMBERS_MAX = " + bound + "\n\n" +
		"// Member is a fixture.\ntype Member int\n\n" +
		"// Member_Invariants is a fixture.\n" +
		"func Member_Invariants(member Member, namespace aver.Namespace) {\n" +
		"\taver.Always(int(member) == MEMBERS_MIN, \"fixed\")\n}\n\n" +
		"// Members is a fixture.\ntype Members []Member\n\n" +
		"// Members_Invariants is a fixture.\n" +
		"func Members_Invariants(members Members, namespace aver.Namespace) {\n" +
		body + "\n}\n\n" +
		"// Holder is a fixture.\ntype Holder struct {\n" +
		"\t// Items is a fixture.\n\tItems Members\n}\n\n" +
		"// Holder_Invariants is a fixture.\n" +
		"func Holder_Invariants(holder Holder, namespace aver.Namespace) {\n" +
		"\tMembers_Invariants(holder.Items, namespace)\n}\n\n" +
		"// Take is a fixture.\n" +
		"func Take(members Members) (out Members) {\n" +
		"\tdefer Members_Invariants(out, aver.Namespace(\"out\"))\n" +
		"\tMembers_Invariants(members, aver.Namespace(\"members\"))\n" +
		"\treturn members\n}\n"
}

// A same-shaped fluent builder owned by another package cannot impersonate aver.Assertions.
func foreign_scalar_helper_source() (code string) {
	return "package fixture\n\n" +
		"import (\n\taver \"fixture/shared/sim/aver/default\"\n" +
		"\tforeign \"fixture/other\"\n)\n\n" +
		"const Value_Min = -4\n\nconst Value_Max = 4\n\n" +
		"// Value is a fixture.\ntype Value int\n\n" +
		"// Value_Invariants is a fixture.\n" +
		"func Value_Invariants(value Value, namespace aver.Namespace) {\n" +
		"\tforeign.Assertions(namespace)." +
		"Range_Int(int(value), int(Value_Min), int(Value_Max)).Ensure()\n}\n"
}

// The mandated literal qualifier keeps helper bodies visually and statically canonical.
func aliased_scalar_helper_source() (code string) {
	return "package fixture\n\n" +
		"import contract \"fixture/shared/sim/aver/default\"\n\n" +
		"const Value_Min = -4\n\nconst Value_Max = 4\n\n" +
		"// Value is a fixture.\ntype Value int\n\n" +
		"// Value_Invariants is a fixture.\n" +
		"func Value_Invariants(value Value, namespace contract.Namespace) {\n" +
		"\tcontract.Assertions(namespace)." +
		"Range_Int(int(value), int(Value_Min), int(Value_Max)).Ensure()\n}\n"
}

// Builds one Range assertion after enough neutral links to pin the expanded-link boundary exactly.
func assertions_builder_body(link_count int) (body string) {
	return "\taver.Tree(value, namespace)." +
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
		if strings.Has_Prefix(file.Path, "other/") {
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
		"\taver \"fixture/shared/sim/aver/default\"\n)\n\n" +
		"func TestMain(m *testing.M) {\n\t" + call + "\n}\n\n" +
		"func Fuzz_Main(f *testing.F) {\n\t" +
		"f.Fuzz(func(t *testing.T, data []byte) {})\n}\n"
}

// INHERITED_FIELD_HEAD declares a struct over a scalar and a struct field, its composing bundle,
// and a defined type over it. Each case appends that defined type's own bundle.
const INHERITED_FIELD_HEAD = "package fixture\n\n" +
	"import aver \"fixture/shared/sim/aver/default\"\n\n" +
	"const Mark_Min = 0\n\nconst Mark_Max = 8\n\n" +
	"// Mark is a fixture.\ntype Mark int\n\n" +
	"// Mark_Invariants is a fixture.\n" +
	"func Mark_Invariants(v Mark, namespace aver.Namespace) {\n" +
	"\taver.Tree(v, namespace).Range_Int(int(v), Mark_Min, Mark_Max).Ensure()\n}\n\n" +
	"// Token is a fixture.\ntype Token string\n\n" +
	"// Token_Invariants is a fixture.\n" +
	"func Token_Invariants(v Token, namespace aver.Namespace) {\n" +
	"\taver.Tree(v, namespace).Range_Int(len(v), Mark_Min, Mark_Max).Ensure()\n}\n\n" +
	"// Inner is a fixture.\ntype Inner struct {\n" +
	"\t// Tok is a fixture.\n\tTok Token\n}\n\n" +
	"// Inner_Invariants is a fixture.\n" +
	"func Inner_Invariants(v Inner, namespace aver.Namespace) {\n" +
	"\tToken_Invariants(v.Tok, namespace)\n}\n\n" +
	"// Holder is a fixture.\ntype Holder struct {\n" +
	"\t// Mk is a fixture.\n\tMk Mark\n\t// In is a fixture.\n\tIn Inner\n}\n\n" +
	"// Holder_Invariants is a fixture.\n" +
	"func Holder_Invariants(v Holder, namespace aver.Namespace) {\n" +
	"\tMark_Invariants(v.Mk, namespace)\n\tInner_Invariants(v.In, namespace)\n}\n\n" +
	"// Kept is a fixture.\ntype Kept Holder\n\n"

// INHERITED_STRUCT_LINK composes the inherited struct field, which every scalar case still owes.
const INHERITED_STRUCT_LINK = "\tInner_Invariants(v.In, namespace)\n"

// A defined type cannot compose the scalar field's helper, because the struct it inherits from
// already holds that type under any shared root.
func assert_inherited_scalar_is_inline(t *testing.T) {
	t.Helper()
	composed := INHERITED_FIELD_HEAD + "// Kept_Invariants is a fixture.\n" +
		"func Kept_Invariants(v Kept, namespace aver.Namespace) {\n" +
		"\tMark_Invariants(v.Mk, namespace)\n" + INHERITED_STRUCT_LINK + "}\n"
	if !diagnosed(check_source(parse(t, &parse_input{
		Path: "pkg/rule.go", Source_Text: composed})),
		"does not assert the inherited field v.Mk inline") {
		t.Fatal("a composed inherited scalar field must be flagged")
	}
	inline := INHERITED_FIELD_HEAD + "// Kept_Invariants is a fixture.\n" +
		"func Kept_Invariants(v Kept, namespace aver.Namespace) {\n" +
		INHERITED_STRUCT_LINK + "\taver.Tree(v, namespace)." +
		"Range_Int(int(v.Mk), Mark_Min, Mark_Max).Ensure()\n}\n"
	if diagnosed(check_source(parse(t, &parse_input{
		Path: "pkg/rule.go", Source_Text: inline})), "v.Mk") {
		t.Fatal("an inlined inherited scalar field must be accepted")
	}
	// A singleton value has no Range and no Enum, thus a direct Always is all that states it.
	singleton := INHERITED_FIELD_HEAD + "// Kept_Invariants is a fixture.\n" +
		"func Kept_Invariants(v Kept, namespace aver.Namespace) {\n" +
		"\taver.Always(int(v.Mk) == Mark_Min, \"the only mark\")\n" +
		INHERITED_STRUCT_LINK + "}\n"
	if diagnosed(check_source(parse(t, &parse_input{
		Path: "pkg/rule.go", Source_Text: singleton})), "v.Mk") {
		t.Fatal("a direct Always must state an inherited scalar field")
	}
}

// A struct has no single link that states it, thus the defined type composes it. A defined type of
// its own keeps that struct at one position when the field type is already occupied.
func assert_inherited_struct_is_composed(t *testing.T) {
	t.Helper()
	inline_only := INHERITED_FIELD_HEAD + "// Kept_Invariants is a fixture.\n" +
		"func Kept_Invariants(v Kept, namespace aver.Namespace) {\n" +
		"\taver.Tree(v, namespace)." +
		"Range_Int(int(v.Mk), Mark_Min, Mark_Max).Ensure()\n}\n"
	if !diagnosed(check_source(parse(t, &parse_input{
		Path: "pkg/rule.go", Source_Text: inline_only})),
		"Call Inner_Invariants(v.In, ...).") {
		t.Fatal("an omitted inherited struct field must be flagged")
	}
	converted := INHERITED_FIELD_HEAD + "// Kept_Invariants is a fixture.\n" +
		"func Kept_Invariants(v Kept, namespace aver.Namespace) {\n" +
		"\tSpare_Invariants(Spare(v.In), namespace)\n" +
		"\taver.Tree(v, namespace)." +
		"Range_Int(int(v.Mk), Mark_Min, Mark_Max).Ensure()\n}\n\n" +
		"// Spare is a fixture.\ntype Spare Inner\n\n" +
		"// Spare_Invariants is a fixture.\n" +
		"func Spare_Invariants(v Spare, namespace aver.Namespace) {\n" +
		"\tToken_Invariants(v.Tok, namespace)\n}\n"
	if diagnosed(check_source(parse(t, &parse_input{
		Path: "pkg/rule.go", Source_Text: converted})), "v.In") {
		t.Fatal("a defined type of its own must compose the inherited struct field")
	}
}

// DEFINED_POINTER_HEAD declares a scalar, a struct over it, and a defined pointer to that struct.
// Each case appends the pointer's own bundle body.
const DEFINED_POINTER_HEAD = "package fixture\n\n" +
	"import aver \"fixture/shared/sim/aver/default\"\n\n" +
	"const Mark_Min = 0\n\nconst Mark_Max = 8\n\n" +
	"// Mark is a fixture.\ntype Mark int\n\n" +
	"// Mark_Invariants is a fixture.\n" +
	"func Mark_Invariants(v Mark, namespace aver.Namespace) {\n" +
	"\taver.Tree(v, namespace)." +
	"Range_Int(int(v), Mark_Min, Mark_Max).Ensure()\n}\n\n" +
	"// Frame is a fixture.\ntype Frame struct {\n" +
	"\t// Mk is a fixture.\n\tMk Mark\n}\n\n" +
	"// Frame_Invariants is a fixture.\n" +
	"func Frame_Invariants(v Frame, namespace aver.Namespace) {\n" +
	"\tMark_Invariants(v.Mk, namespace)\n}\n\n" +
	"// Handle is a fixture.\ntype Handle *Frame\n\n" +
	"// Handle_Invariants is a fixture.\n" +
	"func Handle_Invariants(v Handle, namespace aver.Namespace) {\n"

// DEFINED_POINTER_GUARD is the one nil exit a pointer bundle holds.
const DEFINED_POINTER_GUARD = "\tif v == nil {\n\t\treturn\n\t}\n"

// DEFINED_POINTER_WANT names the exact body every off-shape case is told to write.
const DEFINED_POINTER_WANT = "Function Handle_Invariants body is not exactly " +
	"`if v == nil { return }` then `Frame_Invariants(*v, namespace)`."

// The exact body passes. Every other body, guard alone, aver guard, extra statement, late guard,
// inequality guard, field helper, literal namespace, or value argument, is flagged.
func assert_defined_pointer_body(t *testing.T) {
	t.Helper()
	guard := DEFINED_POINTER_GUARD
	exact := DEFINED_POINTER_HEAD + guard + "\tFrame_Invariants(*v, namespace)\n}\n"
	if diagnosed(check_source(parse(t, &parse_input{
		Path: "pkg/rule.go", Source_Text: exact})), "Handle_Invariants") {
		t.Fatal("exact pointer body must be accepted")
	}
	off_shape := map[string]string{
		"guard only": guard,
		"aver guard": "\taver.Always(v != nil, \"present\")\n" +
			"\tFrame_Invariants(*v, namespace)\n",
		"extra statement": guard + "\tFrame_Invariants(*v, namespace)\n" +
			"\tMark_Invariants(v.Mk, namespace)\n",
		"late guard":        "\tFrame_Invariants(*v, namespace)\n" + guard,
		"inequality guard":  "\tif v != nil {\n\t\tFrame_Invariants(*v, namespace)\n\t}\n",
		"field helper":      guard + "\tMark_Invariants(v.Mk, namespace)\n",
		"literal namespace": guard + "\tFrame_Invariants(*v, \"handle\")\n",
		"value argument":    guard + "\tFrame_Invariants(v, namespace)\n",
	}
	for name, body := range off_shape {
		if !diagnosed(check_source(parse(t, &parse_input{
			Path: "pkg/rule.go", Source_Text: DEFINED_POINTER_HEAD + body + "}\n"})),
			DEFINED_POINTER_WANT) {
			t.Fatalf("%s must be flagged", name)
		}
	}
}

// A non-struct pointee owes the same body. A pointee without a helper and an unnamed pointee are
// each their own diagnostic, never a skip.
func assert_defined_pointer_pointee(t *testing.T) {
	t.Helper()
	scalar := "package fixture\n\n" +
		"import aver \"fixture/shared/sim/aver/default\"\n\n" +
		"const Mark_Min = 0\n\nconst Mark_Max = 8\n\n" +
		"// Mark is a fixture.\ntype Mark int\n\n" +
		"// Mark_Invariants is a fixture.\n" +
		"func Mark_Invariants(v Mark, namespace aver.Namespace) {\n" +
		"\taver.Tree(v, namespace)." +
		"Range_Int(int(v), Mark_Min, Mark_Max).Ensure()\n}\n\n" +
		"// Mark_Pointer is a fixture.\ntype Mark_Pointer *Mark\n\n" +
		"// Mark_Pointer_Invariants is a fixture.\n" +
		"func Mark_Pointer_Invariants(v Mark_Pointer, _ aver.Namespace) {\n" +
		"\taver.Always(v != nil, \"present\")\n}\n"
	if !diagnosed(check_source(parse(t, &parse_input{
		Path: "pkg/rule.go", Source_Text: scalar})),
		"Function Mark_Pointer_Invariants body is not exactly `if v == nil { return }` "+
			"then `Mark_Invariants(*v, namespace)`.") {
		t.Fatal("non-struct pointee owes the same body")
	}
	orphan := "package fixture\n\n" +
		"import aver \"fixture/shared/sim/aver/default\"\n\n" +
		"// Raw is a fixture.\ntype Raw int\n\n" +
		"// Raw_Pointer is a fixture.\ntype Raw_Pointer *Raw\n\n" +
		"// Raw_Pointer_Invariants is a fixture.\n" +
		"func Raw_Pointer_Invariants(v Raw_Pointer, namespace aver.Namespace) {\n" +
		DEFINED_POINTER_GUARD + "}\n"
	if !diagnosed(check_source(parse(t, &parse_input{
		Path: "pkg/rule.go", Source_Text: orphan})),
		"Function Raw_Pointer_Invariants points at Raw without Raw_Invariants.") {
		t.Fatal("pointee without helper must be flagged on the pointer helper")
	}
	unnamed := "package fixture\n\n" +
		"import aver \"fixture/shared/sim/aver/default\"\n\n" +
		"// Blob is a fixture.\ntype Blob *[]byte\n\n" +
		"// Blob_Invariants is a fixture.\n" +
		"func Blob_Invariants(v Blob, namespace aver.Namespace) {\n" +
		DEFINED_POINTER_GUARD + "}\n"
	if !diagnosed(check_source(parse(t, &parse_input{
		Path: "pkg/rule.go", Source_Text: unnamed})),
		"Function Blob_Invariants points at an unnamed type.") {
		t.Fatal("unnamed pointee must be flagged")
	}
}

// A foreign pointee is reached through the file's import name, and the remedy names it that way.
func assert_defined_pointer_foreign(t *testing.T) {
	t.Helper()
	external := parse(t, &parse_input{
		Path: "other/token.go",
		Source_Text: "package other\n\n" +
			"import aver \"fixture/shared/sim/aver/default\"\n\n" +
			"const Token_Min = 0\n\nconst Token_Max = 8\n\n" +
			"// Token is a fixture.\ntype Token string\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(v Token, namespace aver.Namespace) {\n" +
			"\taver.Tree(v, namespace)." +
			"Range_Int(len(v), Token_Min, Token_Max).Ensure()\n}\n"})
	head := "package fixture\n\n" +
		"import (\n\taver \"fixture/shared/sim/aver/default\"\n" +
		"\texternal \"fixture/other\"\n)\n\n" +
		"// Token_Pointer is a fixture.\ntype Token_Pointer *external.Token\n\n" +
		"// Token_Pointer_Invariants is a fixture.\n" +
		"func Token_Pointer_Invariants(v Token_Pointer, namespace aver.Namespace) {\n"
	want := "Function Token_Pointer_Invariants body is not exactly " +
		"`if v == nil { return }` then `external.Token_Invariants(*v, namespace)`."
	guarded := parse(t, &parse_input{Path: "pkg/pointer.go",
		Source_Text: head + DEFINED_POINTER_GUARD + "}\n"})
	if !diagnosed(check_sources([]source.Parsed_File{external, guarded}), want) {
		t.Fatal("foreign pointee owes its package-qualified helper")
	}
	exact := parse(t, &parse_input{Path: "pkg/pointer.go",
		Source_Text: head + DEFINED_POINTER_GUARD +
			"\texternal.Token_Invariants(*v, namespace)\n}\n"})
	if diagnosed(check_sources([]source.Parsed_File{external, exact}),
		"Token_Pointer_Invariants") {
		t.Fatal("exact foreign pointer body must be accepted")
	}
}
