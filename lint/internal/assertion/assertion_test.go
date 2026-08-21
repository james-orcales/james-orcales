package assertion_test

import (
	"testing"

	"local/james-orcales/lint/internal/assertion"
)

// Test_Invariants_Unnamed_Default_Import verifies default-tier package name binds Namespace and
// canonical builder calls when import has no explicit alias.
func Test_Invariants_Unnamed_Default_Import(t *testing.T) {
	t.Parallel()
	code := "package fixture\n\n" +
		"import \"fixture/shared/invariant/default\"\n\n" +
		"const Value_Min = -4\n\nconst Value_Max = 4\n\n" +
		"// Value is a fixture.\ntype Value int\n\n" +
		"// Value_Invariants is a fixture.\n" +
		"func Value_Invariants(value Value, namespace invariant.Namespace) {\n" +
		"\tinvariant.Tree(value, namespace).Range_Int(" +
		"int(value), Value_Min, Value_Max).Ensure()\n}\n"
	pf := parse(t, &parse_input{Path: "pkg/rule.go", Source_Text: code})
	type_diags := assertion.Check_Type(pf.File_Set, pf.File, nil)
	if diagnosed(type_diags, "incorrect parameters") {
		t.Fatalf("default-tier package name was rejected in signature: %v", type_diags)
	}
	diags := check_source(pf)
	if diagnosed(diags, "does not call a canonical helper") {
		t.Fatalf("default-tier package name was rejected in body: %v", diags)
	}
}
