package assertion_test

import (
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
}

// Test_Invariants_Numeric_Bounds verifies a numeric bundle lacking the Always
// upper/lower bound guards is flagged.
func Test_Invariants_Numeric_Bounds(t *testing.T) {
	t.Parallel()
	pf := parse(t, &parse_input{
		Path: "pkg/rule.go",
		Source_Text: "package fixture\n\n" +
			"import \"fixture/shared/invariant\"\n\n" +
			"// Tiny is a fixture.\ntype Tiny uint8\n\n" +
			"// Tiny_Invariants is a fixture.\n" +
			"func Tiny_Invariants(v Tiny, namespace invariant.Namespace) {\n" +
			"\tinvariant.Dot_Product(namespace)." +
			"Sometimes(v == 0, \"zero\").Ensure()\n}\n"})
	if !diagnosed(check_source(pf), "must guard both ends") {
		t.Fatal("a numeric bundle without Always bounds must be flagged")
	}
}

// Test_Invariants_Numeric_Bound_Constant verifies an inline-literal bound is flagged.
func Test_Invariants_Numeric_Bound_Constant(t *testing.T) {
	t.Parallel()
	pf := parse(t, &parse_input{
		Path: "pkg/rule.go",
		Source_Text: "package fixture\n\n" +
			"import \"fixture/shared/invariant\"\n\n" +
			"// Tiny is a fixture.\ntype Tiny uint8\n\n" +
			"// Tiny_Invariants is a fixture.\n" +
			"func Tiny_Invariants(v Tiny, namespace invariant.Namespace) {\n" +
			"\tinvariant.Always(v <= 7, \"max\")\n" +
			"\tinvariant.Always(v >= 0, \"min\")\n" +
			"\tinvariant.Dot_Product(namespace)." +
			"Sometimes(v == 0, \"zero\").Ensure()\n}\n"})
	if !diagnosed(check_source(pf), "must be a package-level constant") {
		t.Fatal("an inline-literal numeric bound must be flagged")
	}
}

// Test_Invariants_Numeric_Coverage verifies a signed numeric bundle missing the
// -1 boundary claim is flagged.
func Test_Invariants_Numeric_Coverage(t *testing.T) {
	t.Parallel()
	// A signed bundle missing the -1 boundary claim is flagged.
	missing_claim := check_source(parse(t, &parse_input{Path: "pkg/rule.go",
		Source_Text: sig_bundle_source(
			"\tinvariant.Always(v <= Sig_Max, \"max\")\n" +
				"\tinvariant.Always(v >= Sig_Min, \"min\")\n" +
				"\tinvariant.Dot_Product(namespace).\n" +
				"\t\tSometimes(v == Sig_Max, \"max\").\n" +
				"\t\tSometimes(v == Sig_Min, \"min\").\n" +
				"\t\tSometimes(v == 0, \"zero\").\n" +
				"\t\tSometimes(v == 1, \"one\").\n" +
				"\t\tSometimes(v == 2, \"two\").Ensure()")}))
	if !diagnosed(missing_claim, "must claim -1") {
		t.Fatal("a signed numeric bundle missing the -1 claim must be flagged")
	}
	// A bundle that claims every sentinel and witnesses its minimum but never its maximum
	// is flagged: both bound edges must be witnessed.
	missing_max := check_source(parse(t, &parse_input{Path: "pkg/rule.go",
		Source_Text: sig_bundle_source(
			"\tinvariant.Always(v <= Sig_Max, \"max\")\n" +
				"\tinvariant.Always(v >= Sig_Min, \"min\")\n" +
				"\tinvariant.Dot_Product(namespace).\n" +
				"\t\tSometimes(v == Sig_Min, \"min\").\n" +
				"\t\tSometimes(v == 0, \"zero\").\n" +
				"\t\tSometimes(v == 1, \"one\").\n" +
				"\t\tSometimes(v == 2, \"two\").\n" +
				"\t\tSometimes(v == -1, \"neg\").Ensure()")}))
	if !diagnosed(missing_max, "must witness its maximum") {
		t.Fatal("a numeric bundle that never witnesses its maximum must be flagged")
	}
}

// Test_Invariants_Numeric_Range_Preset verifies a bundle whose body is one Range_Invariants call
// satisfies the bound and coverage rules, while an inline-literal bound to it is still flagged.
func Test_Invariants_Numeric_Range_Preset(t *testing.T) {
	t.Parallel()
	preset := check_source(parse(t, &parse_input{Path: "pkg/rule.go",
		Source_Text: sig_bundle_source(
			"\tinvariant.Range_Invariants(v, Sig_Min, Sig_Max, namespace)")}))
	verbose := check_source(parse(t, &parse_input{Path: "pkg/rule.go",
		Source_Text: sig_bundle_source(
			"\tinvariant.Always(v <= Sig_Max, \"max\")\n" +
				"\tinvariant.Always(v >= Sig_Min, \"min\")\n" +
				"\tinvariant.Dot_Product(namespace).\n" +
				"\t\tSometimes(v == Sig_Min, \"min\").\n" +
				"\t\tSometimes(v == Sig_Max, \"max\").\n" +
				"\t\tSometimes(v == 0, \"zero\").\n" +
				"\t\tSometimes(v == 1, \"one\").\n" +
				"\t\tSometimes(v == 2, \"two\").\n" +
				"\t\tSometimes(v == -1, \"neg\").Ensure()")}))
	// Either form satisfies the bound and coverage mandate — neither is flagged.
	clean := func(form string, diags []diagnostic.Diagnostic) {
		if diagnosed(diags, "must guard both ends") {
			t.Errorf("%s form must not be flagged for its bounds", form)
		}
		if diagnosed(diags, "must claim") {
			t.Errorf("%s form must not be flagged for its coverage", form)
		}
		if diagnosed(diags, "must witness") {
			t.Errorf("%s form must not be flagged for its edges", form)
		}
		if diagnosed(diags, "must be a package-level constant") {
			t.Errorf("%s form must not be flagged for its constants", form)
		}
	}
	clean("preset", preset)
	clean("verbose", verbose)
	// The Numeric Bound Constant rule still holds: an inline bound to the preset is flagged.
	inline := check_source(parse(t, &parse_input{Path: "pkg/rule.go",
		Source_Text: sig_bundle_source(
			"\tinvariant.Range_Invariants(v, Sig_Min, 7, namespace)")}))
	if !diagnosed(inline, "must be a package-level constant") {
		t.Fatal("an inline-literal preset bound must be flagged")
	}
}

// Test_Invariants_Numeric_Enum_Preset verifies an Enum_Invariants body satisfies the bound and
// coverage mandate at once, while each member is still held to the package-level-constant rule.
func Test_Invariants_Numeric_Enum_Preset(t *testing.T) {
	t.Parallel()
	preset := check_source(parse(t, &parse_input{Path: "pkg/rule.go",
		Source_Text: sig_bundle_source(
			"\tinvariant.Enum_Invariants(v, namespace, Sig_Min, Sig_Max)")}))
	if diagnosed(preset, "must guard both ends") {
		t.Error("an enum preset must not be flagged for its bounds")
	}
	if diagnosed(preset, "must claim") {
		t.Error("an enum preset must not be flagged for its coverage")
	}
	if diagnosed(preset, "must witness") {
		t.Error("an enum preset must not be flagged for its edges")
	}
	if diagnosed(preset, "must be a package-level constant") {
		t.Error("an all-constant enum preset must not be flagged for its members")
	}
	// The Numeric Bound Constant rule still holds: an inline-literal member is flagged.
	inline := check_source(parse(t, &parse_input{Path: "pkg/rule.go",
		Source_Text: sig_bundle_source(
			"\tinvariant.Enum_Invariants(v, namespace, Sig_Min, 7)")}))
	if !diagnosed(inline, "must be a package-level constant") {
		t.Fatal("an inline-literal enum member must be flagged")
	}
}

// Test_Invariants_Count_Bounds verifies a string/slice/map bundle lacking the
// Always len bounds is flagged.
func Test_Invariants_Count_Bounds(t *testing.T) {
	t.Parallel()
	pf := parse(t, &parse_input{
		Path: "pkg/rule.go",
		Source_Text: "package fixture\n\n" +
			"import \"fixture/shared/invariant\"\n\n" +
			"// Name is a fixture.\ntype Name string\n\n" +
			"// Name_Invariants is a fixture.\n" +
			"func Name_Invariants(v Name, namespace invariant.Namespace) {\n" +
			"\tinvariant.Dot_Product(namespace)." +
			"Sometimes(len(v) == 0, \"empty\").Ensure()\n}\n"})
	if !diagnosed(check_source(pf), "Always(len(v) <= MAX)") {
		t.Fatal("a length bundle without Always len bounds must be flagged")
	}
}

// Test_Invariants_Count_Bound_Constant verifies an inline-literal len bound is flagged.
func Test_Invariants_Count_Bound_Constant(t *testing.T) {
	t.Parallel()
	pf := parse(t, &parse_input{
		Path: "pkg/rule.go",
		Source_Text: "package fixture\n\n" +
			"import \"fixture/shared/invariant\"\n\n" +
			"// Name is a fixture.\ntype Name string\n\n" +
			"// Name_Invariants is a fixture.\n" +
			"func Name_Invariants(v Name, namespace invariant.Namespace) {\n" +
			"\tinvariant.Always(len(v) <= 32, \"max\")\n" +
			"\tinvariant.Always(len(v) >= 0, \"min\")\n" +
			"\tinvariant.Dot_Product(namespace)." +
			"Sometimes(len(v) == 0, \"empty\").Ensure()\n}\n"})
	if !diagnosed(check_source(pf), "must be a package-level constant") {
		t.Fatal("an inline-literal len bound must be flagged")
	}
}

// Test_Invariants_Count_Coverage verifies a length bundle missing the 2 claim is flagged.
func Test_Invariants_Count_Coverage(t *testing.T) {
	t.Parallel()
	pf := parse(t, &parse_input{
		Path: "pkg/rule.go",
		Source_Text: "package fixture\n\n" +
			"import \"fixture/shared/invariant\"\n\n" +
			"const Name_Max = 32\n\nconst Name_Min = 0\n\n" +
			"// Name is a fixture.\ntype Name string\n\n" +
			"// Name_Invariants is a fixture.\n" +
			"func Name_Invariants(v Name, namespace invariant.Namespace) {\n" +
			"\tinvariant.Always(len(v) <= Name_Max, \"max bound\")\n" +
			"\tinvariant.Always(len(v) >= Name_Min, \"min bound\")\n" +
			"\tinvariant.Dot_Product(namespace).\n" +
			"\t\tSometimes(len(v) == Name_Max, \"max\").\n" +
			"\t\tSometimes(len(v) == Name_Min, \"min\").\n" +
			"\t\tSometimes(len(v) == 0, \"zero\").\n" +
			"\t\tSometimes(len(v) == 1, \"one\").Ensure()\n}\n"})
	if !diagnosed(check_source(pf), "must claim 2") {
		t.Fatal("a length bundle missing the 2 claim must be flagged")
	}
	// A count bundle whose body is one Range_Invariants over len(v) is accepted.
	preset_source := "package fixture\n\n" +
		"import \"fixture/shared/invariant\"\n\n" +
		"const Name_Max = 32\n\nconst Name_Min = 0\n\n" +
		"// Name is a fixture.\ntype Name string\n\n" +
		"// Name_Invariants is a fixture.\n" +
		"func Name_Invariants(v Name, namespace invariant.Namespace) {\n" +
		"\tinvariant.Range_Invariants(len(v), Name_Min, Name_Max, namespace)\n}\n"
	preset := check_source(parse(t, &parse_input{
		Path: "pkg/rule.go", Source_Text: preset_source}))
	if diagnosed(preset, "must guard") {
		t.Error("a count len-preset bundle must not be flagged for bounds")
	}
	if diagnosed(preset, "must claim") {
		t.Error("a count len-preset bundle must not be flagged for coverage")
	}
	if diagnosed(preset, "must witness") {
		t.Error("a count len-preset bundle must not be flagged for edges")
	}
}

// Test_Invariants_Field_Composition verifies a struct bundle that fails to call a
// field type's _Invariants is flagged, whether the field is a value or a pointer.
func Test_Invariants_Field_Composition(t *testing.T) {
	t.Parallel()
	pf := parse(t, &parse_input{
		Path: "pkg/rule.go",
		Source_Text: "package fixture\n\n" +
			"import \"fixture/shared/invariant\"\n\n" +
			"// Token is a fixture.\ntype Token string\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(v Token, namespace invariant.Namespace) {\n" +
			"\tinvariant.Dot_Product(namespace)." +
			"Sometimes(len(v) == 0, \"x\").Ensure()\n}\n\n" +
			"// Lexeme is a fixture.\ntype Lexeme struct {\n" +
			"\t// Tok is a fixture.\n\tTok Token\n}\n\n" +
			"// Lexeme_Invariants is a fixture.\n" +
			"func Lexeme_Invariants(v Lexeme, namespace invariant.Namespace) {\n" +
			"\tinvariant.Dot_Product(namespace)." +
			"Sometimes(true, \"x\").Ensure()\n}\n\n" +
			"// Phrase is a fixture.\ntype Phrase struct {\n" +
			"\t// Tok is a fixture.\n\tTok *Token\n}\n\n" +
			"// Phrase_Invariants is a fixture.\n" +
			"func Phrase_Invariants(v Phrase, namespace invariant.Namespace) {\n" +
			"\tinvariant.Dot_Product(namespace)." +
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
}

// Test_Invariants_Parameter_Assertion verifies a function that does not assert an
// input parameter is flagged.
func Test_Invariants_Parameter_Assertion(t *testing.T) {
	t.Parallel()
	pf := parse(t, &parse_input{
		Path: "pkg/rule.go",
		Source_Text: "package fixture\n\n" +
			"import \"fixture/shared/invariant\"\n\n" +
			"// Token is a fixture.\ntype Token string\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(v Token, namespace invariant.Namespace) {\n" +
			"\tinvariant.Dot_Product(namespace)." +
			"Sometimes(len(v) == 0, \"x\").Ensure()\n}\n\n" +
			"// Consume does.\nfunc Consume(tok Token) {\n\tprintln(0)\n}\n"})
	if !diagnosed(check_source(pf), "must assert tok") {
		t.Fatal("a function that does not assert an input parameter must be flagged")
	}
}

// Test_Invariants_Output_Assertion verifies a function that does not assert a
// named return in a first-statement defer is flagged.
func Test_Invariants_Output_Assertion(t *testing.T) {
	t.Parallel()
	pf := parse(t, &parse_input{
		Path: "pkg/rule.go",
		Source_Text: "package fixture\n\n" +
			"import \"fixture/shared/invariant\"\n\n" +
			"// Token is a fixture.\ntype Token string\n\n" +
			"// Token_Invariants is a fixture.\n" +
			"func Token_Invariants(v Token, namespace invariant.Namespace) {\n" +
			"\tinvariant.Dot_Product(namespace)." +
			"Sometimes(len(v) == 0, \"x\").Ensure()\n}\n\n" +
			"// Make does.\nfunc Make() (tok Token) {\n\treturn \"\"\n}\n"})
	if !diagnosed(check_source(pf), "must assert tok in a first-statement defer") {
		t.Fatal("a function that does not assert its return in a defer must be flagged")
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

// Runs the cross-file checks over a single source fixture. The component graph maps
// the file to no component (-1) so the recorder and simulation checks Check also runs
// stay inert rather than dereferencing an absent graph and panicking.
func check_source(pf source.Parsed_File) (diags []diagnostic.Diagnostic) {
	return assertion.Check(&assertion.Check_Input{
		Parsed_Files: []source.Parsed_File{pf},
		Components: &source.Component_Index{
			File_To_Component: map[string]int{pf.Path: -1},
		},
		Exempt: nil,
	})
}

// Runs Check with the fixture treated as one shared-library component, the only tier
// the recorder rule binds; every file maps to it so the TestMain wiring is judged.
func recorder_diagnostics(files []source.Parsed_File) (diags []diagnostic.Diagnostic) {
	mapping := map[string]int{}
	for _, pf := range files {
		mapping[pf.Path] = 0
	}
	return assertion.Check(&assertion.Check_Input{
		Parsed_Files: files,
		Components: &source.Component_Index{
			Components: []source.Component{{
				Root: "pkg", Import_Path: "fixture/pkg", Is_Shared_Library: true,
			}},
			File_To_Component: mapping,
		},
		Exempt: nil,
	})
}

// Runs Check with the fixture treated as one binary component rooted at pkg, so the
// simulation checks judge its internal/simulation_test package. Import_Path is the
// real module path the entry check compares the simulation's selector imports against.
func simulation_diagnostics(files []source.Parsed_File) (diags []diagnostic.Diagnostic) {
	mapping := map[string]int{}
	for _, pf := range files {
		mapping[pf.Path] = 0
	}
	return assertion.Check(&assertion.Check_Input{
		Parsed_Files: files,
		Components: &source.Component_Index{
			Components: []source.Component{{
				Root:        "pkg",
				Import_Path: "github.com/james-orcales/james-orcales/pkg",
			}},
			File_To_Component: mapping,
		},
		Exempt: nil,
	})
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

// Wraps a Sig bundle body (a signed int8 with bounds Sig_Min = -8 and Sig_Max = 7) in the fixture
// boilerplate, so a numeric-mandate test varies only the body.
func sig_bundle_source(body string) (source string) {
	return "package fixture\n\n" +
		"import \"fixture/shared/invariant\"\n\n" +
		"const Sig_Max Sig = 7\n\nconst Sig_Min Sig = -8\n\n" +
		"// Sig is a fixture.\ntype Sig int8\n\n" +
		"// Sig_Invariants is a fixture.\n" +
		"func Sig_Invariants(v Sig, namespace invariant.Namespace) {\n" + body + "\n}\n"
}
