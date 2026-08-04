//go:build !invariant_disable_coverage && !prd && !prod && !production && !invariant_noop

package invariant_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	core "local/james-orcales/shared/invariant"
	invariant "local/james-orcales/shared/invariant/default"
)

// Test_Always_Violation prevents a false guard from returning.
func Test_Always_Violation(t *testing.T) {
	message := panic_text(func() {
		core.Recorder_Always(&core.Recorder{}, false, "guard")
	})
	if !strings.Contains(message, "guard") {
		t.Fatalf("panic = %q", message)
	}
}

// Test_Always_Eager prevents guard enforcement from moving to a later boundary.
func Test_Always_Eager(t *testing.T) {
	action := func() {
		core.Recorder_Always(&core.Recorder{}, false, "eager")
	}
	if panic_text(action) == "" {
		t.Fatal("Always did not panic at its call")
	}
}

// Test_Always_Reachability keeps successful guards in the coverage mandate.
func Test_Always_Reachability(t *testing.T) {
	recorder, output, _ := registered_fixture(`package fixture
func check(ok bool) { invariant.Always(ok, "reachable") }
`)
	core.Recorder_Analyze_Assertion_Frequency(recorder)
	want := "🚨 1 coverage gaps 🚨\n\n" +
		"# Gaps by namespace (1)\n\n" +
		"| Namespace | Gaps | Unreached |\n" +
		"|-----------|-----:|----------:|\n" +
		"| reachable |    1 |         1 |\n\n" +
		"# Reachability gaps (1)\n\n" +
		"| Assertion | Type | Source |\n" +
		"|-----------|------|--------|\n" +
		"| reachable |      | ok     |\n\n" +
		"🚨 1 coverage gaps 🚨\n"
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

// Test_Always_Constant prevents a constant-true guard from becoming a coverage obligation.
func Test_Always_Constant(t *testing.T) {
	recorder, output, code := registered_fixture(`package fixture
const PRESENT = true
const ABSENT = false
func literal() { invariant.Always(true, "literal") }
func parenthesized() { invariant.Always((true), "parenthesized") }
func named() { invariant.Always(PRESENT, "named") }
func negated() { invariant.Always(!ABSENT, "negated") }
func recorded(recorder *invariant.Recorder) { invariant.Recorder_Always(recorder, true, "recorded") }
func variable(value bool) { invariant.Always(value, "variable") }
`)
	if code != 1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	want := "🚨 5 constant Always conditions 🚨\n" +
		"/invariant_test/check.go:4  Always condition is constant true\n" +
		"/invariant_test/check.go:5  Always condition is constant true\n" +
		"/invariant_test/check.go:6  Always condition is constant true\n" +
		"/invariant_test/check.go:7  Always condition is constant true\n" +
		"/invariant_test/check.go:8  Always condition is constant true\n" +
		"🚨 5 constant Always conditions 🚨\n"
	if output.String() != want {
		t.Fatalf("output=%q, want %q", output.String(), want)
	}
	if event_count(&recorder.Events) != 0 {
		t.Fatal("a constant Always condition created partial events")
	}
}

// Test_Always_Uniqueness prevents two eager roots from sharing one global coverage identity.
func Test_Always_Uniqueness(t *testing.T) {
	recorder, output, code := registered_fixture(`package fixture
func first(ok bool) { invariant.Always(ok, "duplicate") }
func second(ok bool) { invariant.Always(ok, "duplicate") }
`)
	if code != 1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	want := "🚨 1 duplicate messages 🚨\n" +
		"/invariant_test/check.go:3  duplicate message: \"duplicate\"\n" +
		"🚨 1 duplicate messages 🚨\n"
	if output.String() != want {
		t.Fatalf("output=%q, want %q", output.String(), want)
	}
	if event_count(&recorder.Events) != 0 {
		t.Fatal("duplicate eager identities created partial events")
	}
}

// Test_Fatal_Hook keeps a composition root's last word ahead of the panic, so an asynchronous log
// egress can drain before the process stops.
func Test_Fatal_Hook(t *testing.T) {
	recorder := &core.Recorder{}
	hook_message := ""
	hook_called := false
	recorder.On_Fatal = func(message string) {
		hook_called = true
		hook_message = message
	}
	panic_message := panic_text(func() {
		core.Recorder_Always(recorder, false, "fatal hook guard")
	})
	if !hook_called {
		t.Fatal("fatal hook did not run before the panic")
	}
	if hook_message != panic_message {
		t.Fatalf("hook message = %q, panic = %q", hook_message, panic_message)
	}
}

// Test_Sometimes_Deferred protects Ensure as the only emission boundary.
func Test_Sometimes_Deferred(t *testing.T) {
	recorder := registered_single_axis(t, "deferred")
	metadata := chain_metadata(t, recorder, fixture_chain_key("deferred", 0, "axis"))
	builder := fixture_assertions(recorder, "deferred").
		Sometimes(true, "axis")
	if metadata.Frequency.Load() != 0 {
		t.Fatal("link credited eagerly")
	}
	builder.Ensure()
}

// Test_Sometimes_Coverage protects independent branch credit.
func Test_Sometimes_Coverage(t *testing.T) {
	recorder := registered_single_axis(t, "coverage")
	fixture_assertions(recorder, "coverage").Sometimes(true, "axis").
		Ensure()
	metadata := chain_metadata(t, recorder, fixture_chain_key("coverage", 0, "axis"))
	if metadata.Frequency.Load() != 1 {
		t.Fatal("Ensure did not credit the true branch")
	}
	if metadata.False_Frequency.Load() != 0 {
		t.Fatal("Ensure credited the wrong branch")
	}
}

// Test_Sometimes_Gap keeps both branches mandatory.
func Test_Sometimes_Gap(t *testing.T) {
	recorder, output, _ := registered_fixture(
		bundle_fixture("gap", ".Sometimes(value == 0, \"axis\")"))
	fixture_assertions(recorder, "gap").Sometimes(true, "axis").Ensure()
	core.Recorder_Analyze_Assertion_Frequency(recorder)
	want := "🚨 1 coverage gaps 🚨\n\n" +
		"# Gaps by namespace (1)\n\n" +
		"| Namespace | Gaps | Unreached |\n" +
		"|-----------|-----:|----------:|\n" +
		"| gap       |    1 |         0 |\n\n" +
		"# Branch gaps (1)\n\n" +
		"| Assertion | Type            | Link | Missing | Reached | Property | " +
		"Source     |\n" +
		"|-----------|-----------------|-----:|---------|---------|----------|" +
		"------------|\n" +
		"| gap       | Fixture_Subject |    0 | false   | yes     | axis     | " +
		"value == 0 |\n\n" +
		"🚨 1 coverage gaps 🚨\n"
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

// Test_Inline_Identity keeps one inline message to one assertion across the whole registration.
func Test_Inline_Identity(t *testing.T) {
	recorder, _, code := registered_fixture(`package fixture
func check(value bool, count int) {
	invariant.Sometimes(value, "inline axis")
	invariant.Range(count, 0, 4, "inline range")
}
`)
	if code != -1 {
		t.Fatalf("exit = %d", code)
	}
	inline_metadata(t, recorder, "inline axis", 0, "inline axis")
	inline_metadata(t, recorder, "inline range", 0, core.RANGE_GUARD_MINIMUM)
	inline_metadata(t, recorder, "inline range", 2, core.RANGE_MESSAGE_MINIMUM)
	_, output, code := registered_fixture(`package fixture
func check(value bool) {
	invariant.Always(value, "shared")
	invariant.Sometimes(value, "shared")
}
`)
	if code != 1 {
		t.Fatalf("duplicate exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "duplicate message") {
		t.Fatalf("duplicate output=%q, want a duplicate message diagnostic",
			output.String())
	}
}

// Test_Inline_Record credits an inline helper's branches the way Always credits its guard.
func Test_Inline_Record(t *testing.T) {
	recorder, _, code := registered_fixture(`package fixture
func check(value bool) { invariant.Sometimes(value, "recorded") }
`)
	if code != -1 {
		t.Fatalf("exit = %d", code)
	}
	metadata := inline_metadata(t, recorder, "recorded", 0, "recorded")
	core.Recorder_Sometimes(recorder, true, "recorded")
	if metadata.Frequency.Load() != 1 {
		t.Fatal("inline Sometimes did not credit the true branch")
	}
	if metadata.False_Frequency.Load() != 0 {
		t.Fatal("inline Sometimes credited the wrong branch")
	}
	core.Recorder_Sometimes(recorder, false, "recorded")
	if metadata.False_Frequency.Load() != 1 {
		t.Fatal("inline Sometimes did not credit the false branch")
	}
	gapped, output, _ := registered_fixture(`package fixture
func check(count int) { invariant.Range(count, 0, 4, "gapped") }
`)
	core.Recorder_Range(gapped, 0, 0, 4, "gapped")
	core.Recorder_Analyze_Assertion_Frequency(gapped)
	// An inline helper keys on its own message, thus it owns no subject type and its cell is
	// empty.
	want := "| gapped    |      |    3 | true    | yes     | " +
		"The value equals the maximum. | count  |"
	if !strings.Contains(output.String(), want) {
		t.Fatalf("output = %q, want a row containing %q", output.String(), want)
	}
	if strings.Contains(output.String(), core.ELEMENT_MESSAGE_SEPARATOR) {
		t.Fatalf("output = %q, want no raw separator in the table", output.String())
	}
}

// Test_Inline_Plan keeps the record path off the joined key, so it stays allocation free.
func Test_Inline_Plan(t *testing.T) {
	recorder, _, code := registered_fixture(`package fixture
func check(count int) { invariant.Range(count, 0, 4, "planned") }
`)
	if code != -1 {
		t.Fatalf("exit = %d", code)
	}
	allocations := testing.AllocsPerRun(1000, func() {
		core.Recorder_Range(recorder, 2, 0, 4, "planned")
	})
	if allocations != 0 {
		t.Fatalf("recording allocations = %f, want 0", allocations)
	}
	guard := inline_metadata(t, recorder, "planned", 0, core.RANGE_GUARD_MINIMUM)
	if guard.Frequency.Load() == 0 {
		t.Fatal("inline Range did not credit its lower guard")
	}
	unplanned := &core.Recorder{}
	allocations = testing.AllocsPerRun(1000, func() {
		core.Recorder_Range(unplanned, 2, 0, 4, "planned")
	})
	if allocations != 0 {
		t.Fatalf("plan-free allocations = %f, want 0", allocations)
	}
}

// Test_Assertions_API pins the fluent surface shared by primitive presets.
func Test_Assertions_API(t *testing.T) {
	core.Recorder_Tree(&core.Recorder{}, Fixture_Subject(0), "api").
		Sometimes(true, "axis").Range_Int(1, 0, 2).Enum_Int(1, 1, 2).Ensure()
}

// Test_Assertions_Identity protects namespace, ordinal, and message identity.
func Test_Assertions_Identity(t *testing.T) {
	recorder, _, _ := registered_fixture(bundle_fixture("identity",
		".Sometimes(value == 0, \"same\").Sometimes(value == 1, \"same\")"))
	first := chain_metadata(t, recorder, fixture_chain_key("identity", 0, "same"))
	second := chain_metadata(t, recorder, fixture_chain_key("identity", 1, "same"))
	fixture_assertions(recorder, "identity").
		Sometimes(true, "same").Sometimes(false, "same").Ensure()
	if first.Frequency.Load() != 1 {
		t.Fatalf("first true frequency=%d, want 1", first.Frequency.Load())
	}
	if first.False_Frequency.Load() != 0 {
		t.Fatalf("first frequencies=(%d,%d), want (1,0)",
			first.Frequency.Load(), first.False_Frequency.Load())
	}
	if second.Frequency.Load() != 0 {
		t.Fatalf("second true frequency=%d, want 0", second.Frequency.Load())
	}
	if second.False_Frequency.Load() != 1 {
		t.Fatalf("second frequencies=(%d,%d), want (0,1)",
			second.Frequency.Load(), second.False_Frequency.Load())
	}
}

// Test_Assertions_Subject requires a defined package-level type as the chain subject.
func Test_Assertions_Subject(t *testing.T) {
	_, output, code := registered_fixture("package fixture\n" +
		"type Value int\n" +
		"func Value_Invariants(value Value, namespace invariant.Namespace) {\n" +
		"\tinvariant.Tree(int(value), namespace)." +
		"Sometimes(value == 0, \"zero\").Ensure()\n" +
		"}\n" +
		"func check(value Value) { Value_Invariants(value, \"unnamed\") }\n")
	if code != 1 {
		t.Fatalf("unnamed exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "invalid assertion subjects") {
		t.Fatalf("unnamed output=%q, want an invalid subject diagnostic", output.String())
	}
	// The local type is declared inside the bundle, so the chain has an owner and the subject
	// rule is what has to reject it.
	_, output, code = registered_fixture("package fixture\n" +
		"type Value int\n" +
		"func Value_Invariants(value Value, namespace invariant.Namespace) {\n" +
		"\ttype Local int\n" +
		"\tinvariant.Tree(Local(0), namespace)." +
		"Sometimes(value == 0, \"zero\").Ensure()\n" +
		"}\n" +
		"func check(value Value) { Value_Invariants(value, \"local\") }\n")
	if code != 1 {
		t.Fatalf("local exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "invalid assertion subjects") {
		t.Fatalf("local output=%q, want an invalid subject diagnostic", output.String())
	}
}

// Test_Assertions_Atomic prevents partial coverage from a failing chain.
func Test_Assertions_Atomic(t *testing.T) {
	recorder, _, _ := registered_fixture(`package invariant_test
const Minimum = 0
const Maximum = 4
type Fixture_Subject int
func Fixture_Subject_Invariants(value Fixture_Subject, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(true, "axis").
		Range_Int(int(value), Minimum, Maximum).Ensure()
}
func check(value Fixture_Subject) { Fixture_Subject_Invariants(value, "atomic") }
`)
	builder := fixture_assertions(recorder, "atomic").
		Sometimes(true, "axis").Range_Int(3, 0, 2)
	if panic_text(builder.Ensure) == "" {
		t.Fatal("invalid chain did not panic")
	}
	if count_covered(recorder) != 0 {
		t.Fatal("failed Ensure was not atomic")
	}
	plan := recorder.Assertion_Plans[core.Plan_Key{
		Namespace: "atomic", Package: FIXTURE_PACKAGE, Type: "Fixture_Subject",
	}]
	plan.Links[1].Entry.Metadata = nil
	unknown := fixture_assertions(recorder, "atomic").
		Sometimes(true, "axis").Range_Int(1, 0, 4)
	if !strings.Contains(panic_text(unknown.Ensure), "unknown coverage handle") {
		t.Fatal("Ensure accepted an unresolved registration handle")
	}
	if count_covered(recorder) != 0 {
		t.Fatal("unknown handle partially credited the plan")
	}
}

// Test_Assertions_Foreign keeps unanalyzed packages enforcing without inventing coverage.
func Test_Assertions_Foreign(t *testing.T) {
	recorder := &core.Recorder{Is_Test: true}
	fixture_assertions(recorder, "foreign").Sometimes(true, "axis").
		Ensure()
	if event_count(&recorder.Events) != 0 {
		t.Fatal("foreign chain created coverage")
	}
	message := panic_text(func() {
		fixture_assertions(recorder, "foreign").Range_Int(3, 0, 2).Ensure()
	})
	if message == "" {
		t.Fatal("foreign preset was not enforced")
	}
}

// Test_Assertions_Allocation protects the zero-allocation runtime path.
func Test_Assertions_Allocation(t *testing.T) {
	recorder := &core.Recorder{}
	allocations := testing.AllocsPerRun(1000, func() {
		fixture_assertions(recorder, "allocation").
			Sometimes(true, "axis").Range_Int(1, 0, 2).Ensure()
	})
	if allocations != 0 {
		t.Fatalf("allocations = %f", allocations)
	}
}

// Test_Assertions_Persistence protects structural fuzz-key round trips.
func Test_Assertions_Persistence(t *testing.T) {
	key := fixture_key("persist", 0, "axis")
	recorder := registered_single_axis(t, "persist")
	core.Recorder_Merge_Fuzz_Coverage_From(
		recorder, strings.NewReader(core.Fuzz_Coverage_Line(key, true)))
	if recorder_event(t, recorder, key).Frequency.Load() != 1 {
		t.Fatal("persisted true branch did not merge")
	}
	recorder = registered_single_axis(t, "persist")
	core.Recorder_Merge_Fuzz_Coverage_From(
		recorder, strings.NewReader(core.Fuzz_Coverage_Line(key, false)))
	if recorder_event(t, recorder, key).False_Frequency.Load() != 1 {
		t.Fatal("persisted false branch did not merge")
	}
}

// Test_Assertions_Record_Validation prevents foreign or malformed fuzz records from crediting an
// exact registered identity.
func Test_Assertions_Record_Validation(t *testing.T) {
	key := fixture_key("persist", 0, "axis")
	encoded_key := base64.StdEncoding.EncodeToString([]byte(key))
	assert_persisted_record_ignored(t, "foreign namespace",
		core.Fuzz_Coverage_Line(fixture_key("foreign", 0, "axis"), true))
	assert_persisted_record_ignored(t, "foreign ordinal",
		core.Fuzz_Coverage_Line(fixture_key("persist", 1, "axis"), true))
	assert_persisted_record_ignored(t, "unknown key",
		core.Fuzz_Coverage_Line("unknown", true))
	assert_persisted_record_ignored(t, "invalid base64", "%%%\tT\n")
	assert_persisted_record_ignored(t, "missing separator", encoded_key+"\n")
	assert_persisted_record_ignored(t, "partial record", encoded_key+"\tT")
	assert_persisted_record_ignored(t, "invalid branch", encoded_key+"\tX\n")
}

// Test_Assertions_Registration_Packages keeps direct source registration independent of reach.
func Test_Assertions_Registration_Packages(t *testing.T) {
	recorder, output, code := registered_fixture(`package fixture
type Fixture_Subject int
func Fixture_Subject_Invariants(value Fixture_Subject, namespace invariant.Namespace) {
	invariant.Always(value != 9, "guard")
	invariant.Tree(value, namespace).Sometimes(value == 0, "axis").Ensure()
}
func check(value Fixture_Subject) { Fixture_Subject_Invariants(value, "direct") }
`)
	if code != -1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	if event_count(&recorder.Events) != 2 {
		t.Fatalf("events = %d, want both direct source roots",
			event_count(&recorder.Events))
	}
	chain_metadata(t, recorder, fixture_chain_key("direct", 0, "axis"))
}

// Test_Assertions_Registration_Test_Source prevents test code from emitting production coverage.
func Test_Assertions_Registration_Test_Source(t *testing.T) {
	recorder, output, code := registered_fixture_with_test(`package fixture
type Fixture_Subject int
func Fixture_Subject_Invariants(value Fixture_Subject, namespace invariant.Namespace) {
	invariant.Always(value != 9, "guard")
	invariant.Tree(value, namespace).Sometimes(value == 0, "axis").Ensure()
}
func check(value Fixture_Subject) { Fixture_Subject_Invariants(value, "production") }
`, `package fixture
type Fixture_Subject int
func direct(value Fixture_Subject) {
	invariant.Always(value != 9, "test guard")
	invariant.Tree(value, "test").Sometimes(value == 0, "test axis").Ensure()
	invariant.Int_Invariants(1, "production")
}
func alias() {
	writer := invariant.Int_Invariants
	writer(1, "production")
}
`)
	if code != 1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	want := "🚨 4 test assertion callsites 🚨\n" +
		"/invariant_test/check_test.go:4  test source calls Always\n" +
		"/invariant_test/check_test.go:5  test source calls Tree\n" +
		"/invariant_test/check_test.go:6  test source calls Int_Invariants\n" +
		"/invariant_test/check_test.go:9  test source references Int_Invariants\n" +
		"🚨 4 test assertion callsites 🚨\n"
	if output.String() != want {
		t.Fatalf("output=%q, want %q", output.String(), want)
	}
	if event_count(&recorder.Events) != 0 {
		t.Fatal("a test assertion callsite created partial events")
	}
}

// Test_Assertions_Registration_Transitive keeps reached bundles unconditional across composition.
func Test_Assertions_Registration_Transitive(t *testing.T) {
	recorder := registered_nested_bundle(t)
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "outer", Package: FIXTURE_PACKAGE, Type: "Inner",
		Ordinal: 0, Message: "zero",
	})
	// A descent must register the whole body it enters. An Always or an inline helper in a
	// bundle of an unanalyzed package still runs, thus it still owes coverage.
	output := &bytes.Buffer{}
	across := &core.Recorder{
		File_System: fstest.MapFS{
			"go.mod": &fstest.MapFile{Data: []byte("module fixture\n")},
			"a/a.go": &fstest.MapFile{Data: []byte(`package a
type Leaf int
func Leaf_Invariants(value Leaf, namespace invariant.Namespace) {
	invariant.Always(value != 99, "leaf is not ninety nine")
	invariant.Sometimes(value == 7, "leaf is seven")
	invariant.Tree(value, namespace).Sometimes(value == 0, "zero").Ensure()
}
`)},
			"b/b.go": &fstest.MapFile{Data: []byte(`package b
import a "fixture/a"
func check(value a.Leaf) { a.Leaf_Invariants(value, "root") }
`)},
		},
		Packages_To_Analyze: []string{"/b"}, Output: output,
		Exit: func(int) {}, Is_Test: true,
	}
	core.Recorder_Register_Packages_For_Analysis(across)
	if output.String() != "" {
		t.Fatalf("cross-package output=%q, want no diagnostic", output.String())
	}
	recorder_event(t, across, "leaf is not ninety nine")
	inline_metadata(t, across, "leaf is seven", 0, "leaf is seven")
	chain_metadata(t, across, chain_metadata_key{
		Namespace: "root", Package: "fixture/a", Type: "Leaf",
		Ordinal: 0, Message: "zero",
	})
	// An analyzed bundle is reached two times, by the file walk and by the descent from its
	// callsite. Its eager guards must register one time, not report themselves as duplicates.
	twice, output, code := registered_fixture(`package fixture
type Leaf int
func Leaf_Invariants(value Leaf, namespace invariant.Namespace) {
	invariant.Always(value != 99, "twice guard")
	invariant.Sometimes(value == 7, "twice axis")
	invariant.Tree(value, namespace).Sometimes(value == 0, "zero").Ensure()
}
func check(value Leaf) { Leaf_Invariants(value, "twice") }
`)
	if code != -1 {
		t.Fatalf("twice exit=%d output=%q", code, output.String())
	}
	recorder_event(t, twice, "twice guard")
	inline_metadata(t, twice, "twice axis", 0, "twice axis")
}

// Test_Assertions_Registration_Cross_Package_Constants keeps one shared bound at one site. A type
// that cannot reuse a foreign bundle can still name that bundle's constants.
func Test_Assertions_Registration_Cross_Package_Constants(t *testing.T) {
	qualified := cross_package_constant_fixture(
		"bound.SPAN_MINIMUM, bound.SPAN_MAXIMUM")
	output := &bytes.Buffer{}
	across := &core.Recorder{
		File_System: qualified, Packages_To_Analyze: []string{"/b"}, Output: output,
		Exit: func(int) {}, Is_Test: true,
	}
	core.Recorder_Register_Packages_For_Analysis(across)
	if output.String() != "" {
		t.Fatalf("qualified bounds output=%q, want no diagnostic", output.String())
	}
	chain_metadata(t, across, chain_metadata_key{
		Namespace: "root", Package: "fixture/b", Type: "Span",
		Ordinal: 0, Message: "The value is at least its minimum.",
	})
	// A bare name states no boundary, thus it stays unresolvable even when a package this file
	// imports declares it.
	bare := cross_package_constant_fixture("SPAN_MINIMUM, SPAN_MAXIMUM")
	silent := &bytes.Buffer{}
	local := &core.Recorder{
		File_System: bare, Packages_To_Analyze: []string{"/b"}, Output: silent,
		Exit: func(int) {}, Is_Test: true,
	}
	core.Recorder_Register_Packages_For_Analysis(local)
	if !strings.Contains(silent.String(), "not statically resolvable") {
		t.Fatalf("bare bounds output=%q, want unresolvable", silent.String())
	}
	assert_constant_bytes_resolve(t)
	assert_composed_bytes_resolve(t)
	assert_foreign_declaration_scope(t)
}

// Test_Assertions_Registration_Constant_Expression keeps one evaluator behind every operand, so a
// form the language calls constant never depends on which caller happens to read it.
func Test_Assertions_Registration_Constant_Expression(t *testing.T) {
	bounds := []struct {
		Name       string
		Expression string
		Declared   string
	}{
		// "<a>" is 3, "body" is 4, and "</a>" is 4.
		{Name: "concatenated names", Declared: "0..11",
			Expression: "len(OPEN + BODY + CLOSE)"},
		{Name: "conversion", Declared: "0..6", Expression: "len(string(BODY)) + 2"},
		{Name: "remainder", Declared: "0..23", Expression: "123 % 100"},
		{Name: "exclusive or", Declared: "0..6", Expression: "5 ^ 3"},
		{Name: "right shift", Declared: "0..10", Expression: "80 >> 3"},
		{Name: "parenthesized product", Declared: "0..14",
			Expression: "(3 + 4) * 2"},
	}
	for _, bound := range bounds {
		assert_constant_expression_bound(t, bound.Expression, bound.Declared)
	}
	assert_constant_condition(t)
	assert_non_constant_operand(t)
}

// Test_Assertions_Registration_Word_Width keeps a conversion from vanishing. Reading a complement
// at unbounded precision answers minus one for every width, which is a wrong bound and not a
// refusal.
func Test_Assertions_Registration_Word_Width(t *testing.T) {
	widths := []struct {
		Expression string
		Declared   string
	}{
		{Expression: "^uint8(0)", Declared: "0..255"},
		{Expression: "^uint16(0)", Declared: "0..65535"},
		{Expression: "^uint64(0) >> 60", Declared: "0..15"},
		// A machine word is 64 bits. At 32 each of these would shift away to zero, which
		// no Range admits, thus the number itself is the proof of the width.
		{Expression: "^uint(0) >> 59", Declared: "0..31"},
		{Expression: "^uint(0) >> 40", Declared: "0..16777215"},
		{Expression: "int32(1) << 20", Declared: "0..1048576"},
		// A shift takes the type of its left operand, thus a typed count never narrows an
		// untyped left side into a width that cannot hold the result.
		{Expression: "1 << (32 << (^uint(0) >> 63)) - 1",
			Declared: "0..18446744073709551615"},
	}
	for _, width := range widths {
		assert_constant_expression_bound(t, width.Expression, width.Declared)
	}
	assert_unrepresentable_conversion(t)
}

// Test_Assertions_Registration_Walk keeps registration expansion aligned with runtime ordinals.
func Test_Assertions_Registration_Walk(t *testing.T) {
	recorder, _, code := registered_fixture(`package fixture
type Fixture_Subject int
const Minimum = 0
const Maximum = 4
func Fixture_Subject_Invariants(value Fixture_Subject, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 1, "flag").
		Range_Int(int(value), Minimum, Maximum).Ensure()
}
func check(value Fixture_Subject) { Fixture_Subject_Invariants(value, "walk") }
`)
	if code != -1 {
		t.Fatalf("exit = %d", code)
	}
	if event_count(&recorder.Events) != 7 {
		t.Fatalf("events = %d, want 7 individual entries", event_count(&recorder.Events))
	}
	recorder.Events.Range(func(key any, _ any) (continue_iteration bool) {
		if strings.Contains(key.(string), ":tuple=") {
			t.Fatalf("tuple survived: %q", key)
		}
		return true
	})
}

// Test_Assertions_Registration_Template protects callsite-owned template namespaces.
func Test_Assertions_Registration_Template(t *testing.T) {
	recorder, _, code := registered_fixture(`package fixture
type Number int
func Number_Invariants(value Number, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "zero").Ensure()
}
func check(value Number) { Number_Invariants(value, "number") }
`)
	if code != -1 {
		t.Fatalf("exit = %d", code)
	}
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "number", Package: FIXTURE_PACKAGE, Type: "Number",
		Ordinal: 0, Message: "zero",
	})
}

// Test_Assertions_Registration_Ensured rejects chains with no atomic boundary.
func Test_Assertions_Registration_Ensured(t *testing.T) {
	_, output, code := registered_fixture(`package fixture
type Fixture_Subject int
func Fixture_Subject_Invariants(value Fixture_Subject, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "axis")
}
func check(value Fixture_Subject) { Fixture_Subject_Invariants(value, "dangling") }
`)
	if code != 1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "not terminated by Ensure") {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	_, output, code = registered_fixture(`package fixture
type Fixture_Subject int
func Fixture_Subject_Invariants(value Fixture_Subject, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Ensure()
}
func check(value Fixture_Subject) { Fixture_Subject_Invariants(value, "empty") }
`)
	if code != 1 {
		t.Fatalf("empty exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "has no links") {
		t.Fatalf("empty exit=%d output=%q", code, output.String())
	}
	_, output, code = registered_fixture(`package fixture
type Fixture_Subject int
func Fixture_Subject_Invariants(
	value Fixture_Subject, namespace invariant.Namespace,
) invariant.Assertion_Builder {
	return invariant.Tree(value, namespace).Sometimes(value == 0, "axis")
}
func check(value Fixture_Subject) { Fixture_Subject_Invariants(value, "returned") }
`)
	if code != 1 {
		t.Fatalf("returned exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "not terminated by Ensure") {
		t.Fatalf("returned exit=%d output=%q", code, output.String())
	}
}

// Test_Assertions_Registration_Bundle_Only keeps a chain inside the bundle that owns its type.
func Test_Assertions_Registration_Bundle_Only(t *testing.T) {
	_, output, code := registered_fixture(`package fixture
type Value int
func check(value Value) {
	invariant.Tree(value, "loose").Sometimes(value == 0, "zero").Ensure()
}
`)
	if code != 1 {
		t.Fatalf("loose exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "outside an _Invariants bundle") {
		t.Fatalf("loose output=%q, want a bundle-only diagnostic", output.String())
	}
	recorder, output, code := registered_fixture(`package fixture
type Value int
func Value_Invariants(value Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "zero").Ensure()
}
func check(value Value) { Value_Invariants(value, "owned") }
`)
	if code != -1 {
		t.Fatalf("owned exit=%d output=%q", code, output.String())
	}
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "owned", Package: FIXTURE_PACKAGE, Type: "Value",
		Ordinal: 0, Message: "zero",
	})
}

// Test_Assertions_Registration_Alias keeps a second name from reaching one plan, which would leave
// an obligation nobody owns.
func Test_Assertions_Registration_Alias(t *testing.T) {
	_, output, code := registered_fixture(`package fixture
type Value = struct {
	Count int
}
func check(value Value) { println(value.Count) }
`)
	if code != 1 {
		t.Fatalf("literal alias exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "type alias") {
		t.Fatalf("literal alias output=%q, want an alias diagnostic", output.String())
	}
	// A named right side adds no identity either, thus it is refused on the same terms.
	_, output, code = registered_fixture(`package fixture
type Value int
func Value_Invariants(value Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "zero").Ensure()
}
type Named = Value
func check(value Named) { Value_Invariants(Value(value), "named") }
`)
	if code != 1 {
		t.Fatalf("named alias exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "type alias") {
		t.Fatalf("named alias output=%q, want an alias diagnostic", output.String())
	}
	// A defined type carries an identity of its own, thus it is what a second name must be.
	_, output, code = registered_fixture(`package fixture
type Value int
func Value_Invariants(value Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "zero").Ensure()
}
func check(value Value) { Value_Invariants(value, "defined") }
`)
	if code != -1 {
		t.Fatalf("defined type exit=%d output=%q", code, output.String())
	}
}

// Test_Assertions_Registration_Literal keeps coverage identities statically knowable.
func Test_Assertions_Registration_Literal(t *testing.T) {
	_, output, code := registered_fixture(`package fixture
type Fixture_Subject int
func Fixture_Subject_Invariants(
	value Fixture_Subject, message string, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).Sometimes(value == 0, message).Ensure()
}
func check(value Fixture_Subject) { Fixture_Subject_Invariants(value, "m", "literal") }
`)
	if code != 1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "literal") {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
}

// Test_Assertions_Registration_Namespace rejects ambiguous emission plans.
func Test_Assertions_Registration_Namespace(t *testing.T) {
	recorder, output, code := registered_fixture(`package fixture
type Number int
func Number_Invariants(value Number, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "zero").Ensure()
}
func first(value Number) { Number_Invariants(value, "number") }
func second(value Number) { Number_Invariants(value, "number") }
`)
	if code != 1 {
		t.Fatalf("reuse exit=%d output=%q", code, output.String())
	}
	want := "🚨 1 duplicate bundle namespaces 🚨\n" +
		"/invariant_test/check.go:7  banned: duplicate namespace \"number\"\n" +
		"🚨 1 duplicate bundle namespaces 🚨\n" +
		"🚨 1 repeated subject types 🚨\n" +
		"/invariant_test/check.go:7  repeated subject type \"Number\"" +
		" under namespace \"number\"\n" +
		"🚨 1 repeated subject types 🚨\n"
	if output.String() != want {
		t.Fatalf("reuse output=%q, want %q", output.String(), want)
	}
	if event_count(&recorder.Events) != 0 {
		t.Fatal("a reused namespace published partial events")
	}
	if recorder.Assertion_Plans != nil {
		t.Fatal("a reused namespace published a partial plan")
	}
	recorder, output, code = registered_fixture(`package fixture
type Fixture_Subject int
func Fixture_Subject_Invariants(value Fixture_Subject, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "a").Ensure()
}
func first(value Fixture_Subject) { Fixture_Subject_Invariants(value, "same") }
func second(value Fixture_Subject) { Fixture_Subject_Invariants(value, "same") }
`)
	if code != 1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "duplicate namespace") {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	if event_count(&recorder.Events) != 0 {
		t.Fatal("conflicting roots published partial events")
	}
	if recorder.Assertion_Plans != nil {
		t.Fatal("conflicting roots published a partial namespace")
	}
	recorder, output, code = registered_global_namespace_fixture()
	if code != 1 {
		t.Fatalf("global exit=%d output=%q", code, output.String())
	}
	// The namespace now names a bundle callsite, thus the bundle check reports it first.
	want = "🚨 1 duplicate bundle namespaces 🚨\n" +
		"/b/b.go:6  banned: duplicate namespace \"global\"\n" +
		"🚨 1 duplicate bundle namespaces 🚨\n"
	if output.String() != want {
		t.Fatalf("global output=%q, want %q", output.String(), want)
	}
	if event_count(&recorder.Events) != 0 {
		t.Fatal("a cross-package namespace collision published partial events")
	}
	if recorder.Assertion_Plans != nil {
		t.Fatal("a cross-package namespace collision published a partial plan")
	}
}

// Test_Assertions_Registration_Source_Owner keeps static descent idempotent for one owner.
func Test_Assertions_Registration_Source_Owner(t *testing.T) {
	recorder, output, code := registered_fixture(`package fixture
type Leaf int
func Leaf_Invariants(value Leaf, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "zero").Ensure()
}
type Parent struct { Value Leaf }
func Parent_Invariants(value Parent, namespace invariant.Namespace) {
	Leaf_Invariants(value.Value, namespace)
}
func first(value Parent) { Parent_Invariants(value, "first") }
func second(value Parent) { Parent_Invariants(value, "second") }
`)
	if code != -1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	if output.String() != "" {
		t.Fatalf("output=%q, want no diagnostic", output.String())
	}
	events := event_count(&recorder.Events)
	if events != 2 {
		t.Fatalf("events=%d, want one axis per parent callsite", events)
	}
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "first", Package: FIXTURE_PACKAGE, Type: "Leaf",
		Ordinal: 0, Message: "zero",
	})
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "second", Package: FIXTURE_PACKAGE, Type: "Leaf",
		Ordinal: 0, Message: "zero",
	})
}

// Test_Assertions_Registration_Source_Path keeps two sibling subjects apart under one namespace.
func Test_Assertions_Registration_Source_Path(t *testing.T) {
	recorder, output, code := registered_fixture(`package fixture
type First int
type Second int
type Parent struct { First First; Second Second }
func First_Invariants(value First, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "first").Ensure()
}
func Second_Invariants(value Second, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 1, "second").Ensure()
}
func Parent_Invariants(value Parent, namespace invariant.Namespace) {
	First_Invariants(value.First, namespace)
	Second_Invariants(value.Second, namespace)
}
func check(value Parent) { Parent_Invariants(value, "same") }
`)
	if code != -1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	if output.String() != "" {
		t.Fatalf("output=%q, want no diagnostic", output.String())
	}
	events := event_count(&recorder.Events)
	if events != 2 {
		t.Fatalf("events=%d, want one axis per sibling subject", events)
	}
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "same", Package: FIXTURE_PACKAGE, Type: "First",
		Ordinal: 0, Message: "first",
	})
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "same", Package: FIXTURE_PACKAGE, Type: "Second",
		Ordinal: 0, Message: "second",
	})
}

// Test_Assertions_Registration_Caps keeps registration within the fixed runtime representation.
func Test_Assertions_Registration_Caps(t *testing.T) {
	recorder, output, code := registered_fixture(
		axis_chain_fixture(core.ASSERTION_OBSERVATIONS_MAX, "accepted"))
	if code != -1 {
		t.Fatalf("109 axes exit=%d output=%q", code, output.String())
	}
	if event_count(&recorder.Events) != core.ASSERTION_OBSERVATIONS_MAX {
		t.Fatalf("109 axes registered %d events", event_count(&recorder.Events))
	}
	_, output, code = registered_fixture(
		axis_chain_fixture(core.ASSERTION_OBSERVATIONS_MAX+1, "axes"))
	if code != 1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "109 axes") {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	// A guard consumes a link and no observation, thus 26 four-member enums reach 130 links
	// while holding 104 axes. Only the larger cap can reject that chain.
	var presets strings.Builder
	presets.WriteString(BUNDLE_FIXTURE_HEAD)
	for preset_index := 0; preset_index < 26; preset_index++ {
		presets.WriteString(".Enum_4_Int(int(value), 0, 1, 2, 3)")
	}
	presets.WriteString(bundle_fixture_tail("presets"))
	_, output, code = registered_fixture(presets.String())
	if code != 1 {
		t.Fatalf("130 expanded links exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "127 links") {
		t.Fatalf("130 expanded links exit=%d output=%q", code, output.String())
	}
}

// Test_Bundles_Static keeps template expansion independent of runtime control flow.
func Test_Bundles_Static(t *testing.T) {
	_, output, code := registered_fixture(`package fixture
type Number int
func Number_Invariants(value Number, namespace invariant.Namespace) {
	if value == 0 { invariant.Tree(value, namespace).Sometimes(true, "zero").Ensure() }
}
`)
	if code != 1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "control flow") {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
}

// Test_Bundles_Namespace_Source keeps a nested bundle namespace at its callsite.
func Test_Bundles_Namespace_Source(t *testing.T) {
	recorder, output, code := registered_fixture(`package fixture
type Inner int
func Inner_Invariants(value Inner, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "zero").Ensure()
}
type Outer struct { Inner Inner }
func Outer_Invariants(value Outer, namespace invariant.Namespace) {
	Inner_Invariants(value.Inner, "outer.inner")
}
func check(value Outer) { Outer_Invariants(value, "outer") }
`)
	if code != 1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "bundle literal namespaces") {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	if event_count(&recorder.Events) != 0 {
		t.Fatal("a bundle literal namespace published events")
	}
	if recorder.Assertion_Plans != nil {
		t.Fatal("a bundle literal namespace published a plan")
	}
}

// Test_Bundles_Duplicate_Namespace keeps one namespace literal at one callsite.
func Test_Bundles_Duplicate_Namespace(t *testing.T) {
	recorder, output, code := registered_fixture(`package fixture
type Number int
func Number_Invariants(value Number, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "zero").Ensure()
}
func first(value Number) { Number_Invariants(value, "number") }
func second(value Number) { Number_Invariants(value, "number") }
`)
	if code != 1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "duplicate bundle namespaces") {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	if event_count(&recorder.Events) != 0 {
		t.Fatal("a duplicate namespace literal published events")
	}
	if recorder.Assertion_Plans != nil {
		t.Fatal("a duplicate namespace literal published a plan")
	}
}

// Test_Bundles_Template maps the legacy heading to the fluent template contract.
func Test_Bundles_Template(t *testing.T) {
	Test_Assertions_Registration_Template(t)
}

// Test_Bundles_Descent protects nested template expansion.
func Test_Bundles_Descent(t *testing.T) {
	recorder := registered_nested_bundle(t)
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "outer", Package: FIXTURE_PACKAGE, Type: "Inner",
		Ordinal: 0, Message: "zero",
	})
}

// Test_Bundles_Composition keeps a composed chain free of a cross product.
func Test_Bundles_Composition(t *testing.T) {
	recorder := registered_nested_bundle(t)
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "outer", Package: FIXTURE_PACKAGE, Type: "Inner",
		Ordinal: 0, Message: "zero",
	})
	if event_count(&recorder.Events) != 1 {
		t.Fatalf("events=%d, want the composed axis alone", event_count(&recorder.Events))
	}
}

// Test_Bundles_Tree rejects one subject type that occurs twice under one root.
func Test_Bundles_Tree(t *testing.T) {
	recorder, output, code := registered_fixture("package fixture\n" +
		"type Leaf int\n" +
		"type Left struct { Value Leaf }\n" +
		"type Right struct { Value Leaf }\n" +
		"type Root struct { Left Left; Right Right }\n" +
		"func Leaf_Invariants(value Leaf, namespace invariant.Namespace) {\n" +
		"\tinvariant.Tree(value, namespace)." +
		"Sometimes(value == 0, \"zero\").Ensure()\n" +
		"}\n" +
		"func Left_Invariants(value Left, namespace invariant.Namespace) {\n" +
		"\tLeaf_Invariants(value.Value, namespace)\n" +
		"}\n" +
		"func Right_Invariants(value Right, namespace invariant.Namespace) {\n" +
		"\tLeaf_Invariants(value.Value, namespace)\n" +
		"}\n" +
		"func Root_Invariants(value Root, namespace invariant.Namespace) {\n" +
		"\tLeft_Invariants(value.Left, namespace)\n" +
		"\tRight_Invariants(value.Right, namespace)\n" +
		"}\n" +
		"func check(value Root) { Root_Invariants(value, \"diamond\") }\n")
	if code != 1 {
		t.Fatalf("diamond exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "repeated subject type") {
		t.Fatalf("diamond output=%q, want a repeated subject diagnostic", output.String())
	}
	if event_count(&recorder.Events) != 0 {
		t.Fatal("a diamond published partial events")
	}
	if recorder.Assertion_Plans != nil {
		t.Fatal("a diamond published a partial plan")
	}
	_, output, code = registered_fixture("package fixture\n" +
		"type Leaf int\n" +
		"type Pair struct { Low Leaf; High Leaf }\n" +
		"func Leaf_Invariants(value Leaf, namespace invariant.Namespace) {\n" +
		"\tinvariant.Tree(value, namespace)." +
		"Sometimes(value == 0, \"zero\").Ensure()\n" +
		"}\n" +
		"func Pair_Invariants(value Pair, namespace invariant.Namespace) {\n" +
		"\tLeaf_Invariants(value.Low, namespace)\n" +
		"\tLeaf_Invariants(value.High, namespace)\n" +
		"}\n" +
		"func check(value Pair) { Pair_Invariants(value, \"sibling\") }\n")
	if code != 1 {
		t.Fatalf("sibling exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "repeated subject type") {
		t.Fatalf("sibling output=%q, want a repeated subject diagnostic", output.String())
	}
}

// Test_Bundles_Casing accepts the repository's two invariant-helper casings.
func Test_Bundles_Casing(t *testing.T) {
	recorder, _, _ := registered_fixture(`package fixture
type number int
func number_invariants(value number, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "zero").Ensure()
}
func check(value number) { number_invariants(value, "number") }
`)
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "number", Package: FIXTURE_PACKAGE, Type: "number",
		Ordinal: 0, Message: "zero",
	})
}

// Test_Bundles_Sugar protects unqualified roots only inside the configured sugar package.
func Test_Bundles_Sugar(t *testing.T) {
	recorder, _, code := registered_fixture_with_sugar(`package invariant
type Number int
func Number_Invariants(value Number, namespace Namespace) {
	Tree(value, namespace).Sometimes(value == 0, "zero").Ensure()
}
func check(value Number) { Number_Invariants(value, "number") }
`)
	if code != -1 {
		t.Fatalf("exit = %d", code)
	}
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "number", Package: FIXTURE_PACKAGE, Type: "Number",
		Ordinal: 0, Message: "zero",
	})
}

// Test_Bundles_Cross_Package protects indexed template expansion across imports.
func Test_Bundles_Cross_Package(t *testing.T) {
	recorder := &core.Recorder{
		File_System: fstest.MapFS{
			"go.mod": &fstest.MapFile{Data: []byte("module fixture\n")},
			"a/a.go": &fstest.MapFile{Data: []byte(`package a
type Number int
func Number_Invariants(value Number, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "zero").Ensure()
	Position_Invariants(Position(value), "position")
}
type Position int
func Position_Invariants(value Position, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value > 0, "positive").Ensure()
}
`)},
			"b/b.go": &fstest.MapFile{Data: []byte(`package b
import a "fixture/a"
func check(value a.Number) { a.Number_Invariants(value, "number") }
`)},
		},
		Packages_To_Analyze: []string{"/b"}, Output: &bytes.Buffer{},
		Exit: func(int) {}, Is_Test: true,
	}
	core.Recorder_Register_Packages_For_Analysis(recorder)
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "number", Package: "fixture/a", Type: "Number",
		Ordinal: 0, Message: "zero",
	})
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "position", Package: "fixture/a", Type: "Position",
		Ordinal: 0, Message: "positive",
	})
}

// Test_Bundles_Callsite keeps each invocation's namespace independent.
func Test_Bundles_Callsite(t *testing.T) {
	recorder, output, _ := registered_fixture(`package fixture
type Fixture_Subject int
func Fixture_Subject_Invariants(value Fixture_Subject, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "zero").Ensure()
}
func first(value Fixture_Subject) { Fixture_Subject_Invariants(value, "first") }
func second(value Fixture_Subject) { Fixture_Subject_Invariants(value, "second") }
`)
	first := chain_metadata(t, recorder, fixture_chain_key("first", 0, "zero"))
	second := chain_metadata(t, recorder, fixture_chain_key("second", 0, "zero"))
	fixture_assertions(recorder, "first").Sometimes(true, "zero").
		Ensure()
	if first.Frequency.Load() != 1 {
		t.Fatal("first callsite did not receive its own credit")
	}
	if second.Frequency.Load() != 0 {
		t.Fatal("first callsite credited the second true branch")
	}
	if second.False_Frequency.Load() != 0 {
		t.Fatal("first callsite credited the second namespace")
	}
	core.Recorder_Analyze_Assertion_Frequency(recorder)
	// The ranking separates the two at a glance: "second" was never driven, "first" was.
	want := "🚨 3 coverage gaps 🚨\n\n" +
		"# Gaps by namespace (2)\n\n" +
		"| Namespace | Gaps | Unreached |\n" +
		"|-----------|-----:|----------:|\n" +
		"| second    |    2 |         2 |\n" +
		"| first     |    1 |         0 |\n\n" +
		"# Branch gaps (3)\n\n" +
		"| Assertion | Type            | Link | Missing | Reached | Property | " +
		"Source     |\n" +
		"|-----------|-----------------|-----:|---------|---------|----------|" +
		"------------|\n" +
		"| first     | Fixture_Subject |    0 | false   | yes     | zero     | " +
		"value == 0 |\n" +
		"| second    | Fixture_Subject |    0 | false   | no      | zero     | " +
		"value == 0 |\n" +
		"| second    | Fixture_Subject |    0 | true    | no      | zero     | " +
		"value == 0 |\n\n" +
		"🚨 3 coverage gaps 🚨\n"
	if output.String() != want {
		t.Fatalf("output=%q, want %q", output.String(), want)
	}
}

// Test_Bundles_Gap_Location keeps diagnostics attached to the callsite identity.
func Test_Bundles_Gap_Location(t *testing.T) {
	recorder, output, _ := registered_fixture(`package fixture
type Number int
func Number_Invariants(value Number, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "zero").Ensure()
}
func check(value Number) { Number_Invariants(value, "number") }
`)
	core.Recorder_Analyze_Assertion_Frequency(recorder)
	if !strings.Contains(output.String(),
		"| number    | Number |    0 | false   | no      | zero") {
		t.Fatalf("output = %q", output.String())
	}
}

// Test_Bundles_Boolean requires a defined Boolean type to state exactly one Sometimes.
func Test_Bundles_Boolean(t *testing.T) {
	recorder, output, code := registered_fixture(`package fixture
type Flag bool
func Flag_Invariants(value Flag, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(bool(value), "set").Ensure()
}
func check(value Flag) { Flag_Invariants(value, "flag") }
`)
	if code != -1 {
		t.Fatalf("accepted exit=%d output=%q", code, output.String())
	}
	chain_metadata(t, recorder, fixture_chain_key_typed("flag", "Flag", 0, "set"))
	_, output, code = registered_fixture(`package fixture
type Flag bool
func Flag_Invariants(value Flag, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "set").
		Sometimes(!bool(value), "clear").
		Ensure()
}
func check(value Flag) { Flag_Invariants(value, "flag") }
`)
	if code != 1 {
		t.Fatalf("two links exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "Boolean bundle") {
		t.Fatalf("two links output=%q, want a Boolean bundle diagnostic", output.String())
	}
	_, output, code = registered_fixture(`package fixture
type Flag bool
func Flag_Invariants(value Flag, namespace invariant.Namespace) {
	invariant.Always(bool(value) || !bool(value), "flag is a flag")
}
func check(value Flag) { Flag_Invariants(value, "flag") }
`)
	if code != 1 {
		t.Fatalf("no chain exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "Boolean bundle") {
		t.Fatalf("no chain output=%q, want a Boolean bundle diagnostic", output.String())
	}
}

// Test_Bundles_Custom_Types prevents primitive helpers from replacing typed domain helpers.
func Test_Bundles_Custom_Types(t *testing.T) {
	_, output, code := registered_fixture(`package fixture
func Int_Invariants(value int, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "zero").Ensure()
}
`)
	if code != 1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "primitive") {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
}

// Test_Analysis_Gaps keeps uncovered branches fatal and keeps two duplicated bundles apart.
func Test_Analysis_Gaps(t *testing.T) {
	Test_Sometimes_Gap(t)
	analysis_gap_names_its_subject(t)
}

// Test_Analysis_Reached separates a wrong bound from an absent witness. An axis that ran and lacks
// one polarity needs a different value; one that never ran needs a different path.
func Test_Analysis_Reached(t *testing.T) {
	// One polarity witnessed, thus the axis ran and owes only its other branch.
	ran, output, _ := registered_fixture(
		bundle_fixture("ran", ".Sometimes(value == 0, \"axis\")"))
	fixture_assertions(ran, "ran").Sometimes(true, "axis").Ensure()
	core.Recorder_Analyze_Assertion_Frequency(ran)
	if !strings.Contains(output.String(), "| false   | yes     |") {
		t.Fatalf("output = %q, want a reached branch row", output.String())
	}
	// Nothing drove the chain, thus both polarities are absent and both rows say so.
	quiet, silent, _ := registered_fixture(
		bundle_fixture("quiet", ".Sometimes(value == 0, \"axis\")"))
	core.Recorder_Analyze_Assertion_Frequency(quiet)
	if strings.Contains(silent.String(), "| yes     |") {
		t.Fatalf("output = %q, want no reached row", silent.String())
	}
	if strings.Count(silent.String(), "| no      |") != 2 {
		t.Fatalf("output = %q, want both polarities unreached", silent.String())
	}
}

// Test_Analysis_Namespace_Summary puts the largest namespace first, so the reader starts where the
// gaps are rather than counting rows.
func Test_Analysis_Namespace_Summary(t *testing.T) {
	recorder, output, _ := registered_fixture(`package fixture
type Alpha int
func Alpha_Invariants(value Alpha, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "axis").Ensure()
}
type Fixture_Subject int
func Fixture_Subject_Invariants(value Fixture_Subject, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "first").
		Sometimes(value == 1, "second").Sometimes(value == 2, "third").Ensure()
}
func alpha(value Alpha) { Alpha_Invariants(value, "alpha") }
func wide(value Fixture_Subject) { Fixture_Subject_Invariants(value, "wide") }
`)
	// Each axis witnesses one polarity, thus "wide" holds three reached gaps against the two
	// unreached gaps of a namespace nothing drove.
	fixture_assertions(recorder, "wide").
		Sometimes(true, "first").Sometimes(false, "second").
		Sometimes(false, "third").Ensure()
	core.Recorder_Analyze_Assertion_Frequency(recorder)
	want := "# Gaps by namespace (2)\n\n" +
		"| Namespace | Gaps | Unreached |\n" +
		"|-----------|-----:|----------:|\n" +
		"| wide      |    3 |         0 |\n" +
		"| alpha     |    2 |         2 |\n"
	if !strings.Contains(output.String(), want) {
		t.Fatalf("output = %q, want a ranking containing %q", output.String(), want)
	}
}

// Test_Analysis_Domains keeps the values a Range actually saw beside the interval it declared, so a
// bound no real value approaches is visible without a second run.
func Test_Analysis_Domains(t *testing.T) {
	recorder, output, _ := registered_fixture(
		bundle_fixture("heading_size", ".Range_Int(int(value), -9, 9)"))
	for _, level := range []int{1, 6, 2} {
		fixture_assertions(recorder, "heading_size").Range_Int(level, -9, 9).Ensure()
	}
	core.Recorder_Analyze_Assertion_Frequency(recorder)
	want := "| heading_size | Fixture_Subject |    0 | -9..9    | 1..6     |     3 |"
	if !strings.Contains(output.String(), want) {
		t.Fatalf("output = %q, want a row containing %q", output.String(), want)
	}
	// A declared bound is not a verdict, thus a Range nobody drove still states its interval.
	quiet, silent, _ := registered_fixture(
		bundle_fixture("quiet", ".Range_Int(int(value), -9, 9)"))
	core.Recorder_Analyze_Assertion_Frequency(quiet)
	if !strings.Contains(silent.String(), "| -9..9    |          |     0 |") {
		t.Fatalf("output = %q, want an unobserved domain row", silent.String())
	}
}

// Test_Analysis_Reachability_Identity keeps a builder's internal key out of public reports.
func Test_Analysis_Reachability_Identity(t *testing.T) {
	recorder, output, _ := registered_fixture(`package fixture
type Fixture_Subject int
func Fixture_Subject_Invariants(value Fixture_Subject, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Range_Int(int(value), 0, 4).Ensure()
}
func check(value Fixture_Subject) { Fixture_Subject_Invariants(value, "range") }
`)
	recorder.Events.Range(func(key any, value any) (continue_iteration bool) {
		metadata := value.(*core.Assertion_Metadata)
		if metadata.Kind == core.ASSERTION_KIND_SOMETIMES {
			metadata.Frequency.Store(1)
			metadata.False_Frequency.Store(1)
		}
		return true
	})
	core.Recorder_Analyze_Assertion_Frequency(recorder)
	want := "🚨 2 coverage gaps 🚨\n\n" +
		"# Gaps by namespace (1)\n\n" +
		"| Namespace | Gaps | Unreached |\n" +
		"|-----------|-----:|----------:|\n" +
		"| range     |    2 |         2 |\n\n" +
		"# Reachability gaps (2)\n\n" +
		"| Assertion | Type            | Source     |\n" +
		"|-----------|-----------------|------------|\n" +
		"| range     | Fixture_Subject | int(value) |\n" +
		"| range     | Fixture_Subject | int(value) |\n\n" +
		"# Range domains (1)\n\n" +
		"| Assertion | Type            | Link | Declared | Observed | Count | " +
		"Source     |\n" +
		"|-----------|-----------------|-----:|----------|----------|------:|" +
		"------------|\n" +
		"| range     | Fixture_Subject |    0 | 0..4     |          |     0 | " +
		"int(value) |\n\n" +
		"🚨 2 coverage gaps 🚨\n"
	if output.String() != want {
		t.Fatalf("output=%q, want %q", output.String(), want)
	}
}

// Test_Analysis_Table_Order keeps repeated assertion names ordered by numeric link and polarity.
func Test_Analysis_Table_Order(t *testing.T) {
	recorder, output, _ := registered_fixture(`package fixture
type Alpha int
func Alpha_Invariants(value Alpha, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "axis").Ensure()
}
type Fixture_Subject int
func Fixture_Subject_Invariants(value Fixture_Subject, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "second").
		Sometimes(value == 1, "first").Ensure()
}
func alpha(value Alpha) { Alpha_Invariants(value, "alpha") }
func same(value Fixture_Subject) { Fixture_Subject_Invariants(value, "same") }
`)
	fixture_assertions(recorder, "same").
		Sometimes(false, "second").Sometimes(false, "first").Ensure()
	core.Recorder_Analyze_Assertion_Frequency(recorder)
	// Two namespaces tie on count, thus the ranking breaks the tie by name.
	want := "🚨 4 coverage gaps 🚨\n\n" +
		"# Gaps by namespace (2)\n\n" +
		"| Namespace | Gaps | Unreached |\n" +
		"|-----------|-----:|----------:|\n" +
		"| alpha     |    2 |         2 |\n" +
		"| same      |    2 |         0 |\n\n" +
		"# Branch gaps (4)\n\n" +
		"| Assertion | Type            | Link | Missing | Reached | Property | " +
		"Source     |\n" +
		"|-----------|-----------------|-----:|---------|---------|----------|" +
		"------------|\n" +
		"| alpha     | Alpha           |    0 | false   | no      | axis     | " +
		"value == 0 |\n" +
		"| alpha     | Alpha           |    0 | true    | no      | axis     | " +
		"value == 0 |\n" +
		"| same      | Fixture_Subject |    0 | true    | yes     | second   | " +
		"value == 0 |\n" +
		"| same      | Fixture_Subject |    1 | true    | yes     | first    | " +
		"value == 1 |\n\n" +
		"🚨 4 coverage gaps 🚨\n"
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

// Test_Analysis_Table_Escape keeps Markdown structure outside every dynamic cell.
func Test_Analysis_Table_Escape(t *testing.T) {
	link := uint8(2)
	property := "P|Q\\R\nS"
	gaps := []core.Coverage_Gap{{
		Section: "branch", Assertion: "A|B\\C\nD", Type: "T|U\\V\nW", Link: &link,
		Absent: "true", Property: &property, Source: "x|y\\z\nw",
	}}
	output := &bytes.Buffer{}
	if err := core.Coverage_Gap_Table_Write(output, gaps); err != nil {
		t.Fatal(err)
	}
	// The ranking escapes its namespace cell on the same terms as every other table.
	want := "🚨 1 coverage gaps 🚨\n\n" +
		"# Gaps by namespace (1)\n\n" +
		"| Namespace    | Gaps | Unreached |\n" +
		"|--------------|-----:|----------:|\n" +
		"| A\\|B\\\\C<br>D |    1 |         1 |\n\n" +
		"# Branch gaps (1)\n\n" +
		"| Assertion    | Type         | Link | Missing | Reached | Property     | " +
		"Source       |\n" +
		"|--------------|--------------|-----:|---------|---------|--------------|" +
		"--------------|\n" +
		"| A\\|B\\\\C<br>D | T\\|U\\\\V<br>W |    2 | true    | no      | " +
		"P\\|Q\\\\R<br>S | x\\|y\\\\z<br>w |\n\n" +
		"🚨 1 coverage gaps 🚨\n"
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

// Test_Analysis_Overflow keeps a report nobody can read off the terminal, and keeps the two facts
// that replace it exact: where the report went and how many gaps it holds.
func Test_Analysis_Overflow(t *testing.T) {
	// Twenty axes nobody drives owe both polarities, which is the threshold exactly.
	bounded, output, _ := registered_fixture(axis_chain_fixture(20, "bounded"))
	spilled := 0
	bounded.Report_Overflow = func(gaps []core.Coverage_Gap) (path string, err error) {
		spilled = len(gaps)
		return "/tmp/unused.json", nil
	}
	core.Recorder_Analyze_Assertion_Frequency(bounded)
	if spilled != 0 {
		t.Fatalf("40 gaps spilled %d records, want the whole report on the terminal",
			spilled)
	}
	if !strings.Contains(output.String(), "# Branch gaps (40)") {
		t.Fatalf("output = %q, want the branch table", output.String())
	}
	over, terminal, _ := registered_fixture(axis_chain_fixture(21, "over"))
	over.Report_Overflow = func(gaps []core.Coverage_Gap) (path string, err error) {
		spilled = len(gaps)
		return "/tmp/invariant-coverage-gaps-1.json", nil
	}
	core.Recorder_Analyze_Assertion_Frequency(over)
	if spilled != 42 {
		t.Fatalf("spilled %d records, want every gap the banner counts", spilled)
	}
	if strings.Contains(terminal.String(), "# Branch gaps") {
		t.Fatalf("output = %q, want no branch table", terminal.String())
	}
	want := "42 coverage gaps saved to /tmp/invariant-coverage-gaps-1.json\n"
	if !strings.Contains(terminal.String(), want) {
		t.Fatalf("output = %q, want it to end with %q", terminal.String(), want)
	}
	if !strings.Contains(terminal.String(), "# Gaps by namespace (1, showing 1)") {
		t.Fatalf("output = %q, want a truncated ranking", terminal.String())
	}
	assert_overflow_seam_absent(t)
}

// Test_Analysis_Output_Configuration keeps an invalid output mode fatal and diagnostic.
func Test_Analysis_Output_Configuration(t *testing.T) {
	output := &bytes.Buffer{}
	code := -1
	recorder := &core.Recorder{
		Output: output, Exit: func(status int) { code = status }, Is_Test: true,
		Output_Configuration_Diagnostic: "INVARIANT_OUTPUT has unknown value \"dense\"; " +
			"expected \"table\" or \"json\"",
	}
	core.Recorder_Analyze_Assertion_Frequency(recorder)
	want := "invariant: INVARIANT_OUTPUT has unknown value \"dense\"; " +
		"expected \"table\" or \"json\"\n"
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
}

// Test_Analysis_Summary protects each expanded obligation and the panic-able subset from being
// collapsed into one undifferentiated helper count.
func Test_Analysis_Summary(t *testing.T) {
	recorder, _, _ := registered_fixture(`package fixture
type Fixture_Subject int
const Minimum = -2
const Maximum = 3
const Hole = 0
func Fixture_Subject_Invariants(value Fixture_Subject, namespace invariant.Namespace) {
	invariant.Always(value != 99, "guard")
	invariant.Tree(value, namespace).
		Sometimes(value == 0, "axis").
		Range_Holed_Int(int(value), Minimum, Maximum, Hole, Hole, Hole, Hole).
		Enum_Int(int(value), Minimum, Maximum).
		Ensure()
}
func check(value Fixture_Subject) { Fixture_Subject_Invariants(value, "summary") }
`)
	want := "✓ invariant_test: tested \033[34m21\033[0m properties, " +
		"of which \033[33m5\033[0m are panic-able"
	if summary := core.Recorder_Assertion_Summary(recorder); summary != want {
		t.Fatalf("summary = %q, want %q", summary, want)
	}
}

// Test_Analysis_Clean prevents complete coverage from exiting as a failure.
func Test_Analysis_Clean(t *testing.T) {
	recorder := registered_single_axis(t, "clean")
	fixture_assertions(recorder, "clean").Sometimes(true, "axis").
		Ensure()
	fixture_assertions(recorder, "clean").Sometimes(false, "axis").
		Ensure()
	code := -1
	recorder.Exit = func(status int) { code = status }
	core.Recorder_Analyze_Assertion_Frequency(recorder)
	if code != -1 {
		t.Fatalf("clean analysis exited %d", code)
	}
}

// Test_Coverage_Modes keeps benchmarks from contaminating coverage.
func Test_Coverage_Modes(t *testing.T) {
	recorder := registered_single_axis(t, "mode")
	recorder.Is_Benchmark = true
	fixture_assertions(recorder, "mode").Sometimes(true, "axis").
		Ensure()
	if count_covered(recorder) != 0 {
		t.Fatal("benchmark recorded coverage")
	}
}

// Test_Coverage_Enforcement prevents benchmarks from disabling guards.
func Test_Coverage_Enforcement(t *testing.T) {
	recorder := &core.Recorder{Is_Benchmark: true}
	message := panic_text(func() {
		fixture_assertions(recorder, "range").Range_Int(3, 0, 2).Ensure()
	})
	if message == "" {
		t.Fatal("benchmark skipped enforcement")
	}
}

// Test_Coverage_Uniqueness maps coverage uniqueness to namespace uniqueness.
func Test_Coverage_Uniqueness(t *testing.T) {
	Test_Assertions_Registration_Namespace(t)
}

// Test_Coverage_Literal maps stable coverage identity to literal registration.
func Test_Coverage_Literal(t *testing.T) {
	Test_Assertions_Registration_Literal(t)
}

// Test_Range_Holed keeps arbitrary strict-interior holes canonical and recordable.
func Test_Range_Holed(t *testing.T) {
	recorder, output, code := registered_fixture(`package fixture
type Fixture_Subject int
func Fixture_Subject_Invariants(value Fixture_Subject, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Holed_Int(int(value), -4, 6, -2, 3, 3, 3).Ensure()
}
func check(value Fixture_Subject) { Fixture_Subject_Invariants(value, "holed") }
`)
	if code != -1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	fixture_assertions(recorder, "holed").
		Range_Holed_Int(0, -4, 6, -2, 3, 3, 3).
		Ensure()
	if event_count(&recorder.Events) != 8 {
		t.Fatalf("events = %d, want canonical guards, boundaries, and sentinels",
			event_count(&recorder.Events))
	}
}

// Test_Range_Guard keeps both successful bounds visible.
func Test_Range_Guard(t *testing.T) {
	recorder, _, _ := registered_range(t, "range", 0, 4)
	fixture_assertions(recorder, "range").Range_Int(1, 0, 4).Ensure()
	lower := chain_metadata(t, recorder,
		fixture_chain_key("range", 0, "The value is at least its minimum."))
	if lower.Frequency.Load() != 1 {
		t.Fatal("lower guard was not credited")
	}
	upper := chain_metadata(t, recorder,
		fixture_chain_key("range", 1, "The value is at most its maximum."))
	if upper.Frequency.Load() != 1 {
		t.Fatal("upper guard was not credited")
	}
}

// Test_Range_Coverage keeps both boundaries mandatory when distinct.
func Test_Range_Coverage(t *testing.T) {
	recorder, _, _ := registered_range(t, "range", -2, 3)
	for value := -2; value <= 3; value++ {
		fixture_assertions(recorder, "range").Range_Int(value, -2, 3).
			Ensure()
	}
	messages := []string{
		"The value equals the minimum.",
		"The value equals the maximum.",
		"The value is zero.",
		"The value is one.",
		"The value is two.",
		"The value is negative one.",
	}
	for message_index, message := range messages {
		metadata := chain_metadata(t, recorder, chain_metadata_key{
			Namespace: "range", Package: FIXTURE_PACKAGE, Type: "Fixture_Subject",
			Ordinal: uint8(message_index + 2), Message: message,
		})
		if metadata.Frequency.Load() == 0 {
			t.Fatalf("%q true branch was not witnessed", message)
		}
		if metadata.False_Frequency.Load() == 0 {
			t.Fatalf("%q false branch was not witnessed", message)
		}
	}
}

// Test_Range_Cardinality requires the smallest helper that states each registered integer
// domain exactly, while leaving unregistered runtime enforcement unchanged.
func Test_Range_Cardinality(t *testing.T) {
	methods := []struct {
		Suffix string
		Type   string
	}{
		{"Int", "int"}, {"Int8", "int8"}, {"Int16", "int16"},
		{"Int32", "int32"}, {"Int64", "int64"}, {"Uint", "uint"},
		{"Uint8", "uint8"}, {"Uint16", "uint16"},
		{"Uint32", "uint32"}, {"Uint64", "uint64"},
	}
	for _, method := range methods {
		assert_range_cardinality(t, method.Suffix, method.Type)
		assert_range_holed_cardinality(t, method.Suffix, method.Type)
	}
	assert_small_range_rejected(t, "Range_Int(v, -2, 1)", 4, "Enum_4_Int")
	assert_small_range_rejected(t,
		"Range_Uint64(v, 18446744073709551612, 18446744073709551615)",
		4, "Enum_4_Uint64")
	assert_large_range_registered(t,
		"Range_Uint64(v, 18446744073709551611, 18446744073709551615)", "uint64")
	assert_large_range_registered(t,
		"Range_Int64(v, -9223372036854775808, -9223372036854775804)", "int64")
	recorder := &core.Recorder{}
	fixture_assertions(recorder, "runtime").Range_Int(1, 0, 1).Ensure()
	if panic_text(fixture_assertions(recorder, "runtime").
		Range_Int(2, 0, 1).Ensure) == "" {
		t.Fatal("an unregistered Range stopped enforcing its bounds")
	}
}

// Test_Range_Exclusions prevents canonical padding from weakening boundary witnesses or counting
// one hole repeatedly.
func Test_Range_Exclusions(t *testing.T) {
	recorder, output, code := registered_fixture(`package fixture
type Fixture_Subject int
func Fixture_Subject_Invariants(value Fixture_Subject, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Holed_Int(int(value), -4, 6, -2, 3, 3, 3).Ensure()
}
func check(value Fixture_Subject) { Fixture_Subject_Invariants(value, "range") }
`)
	if code != -1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	for value := -4; value <= 6; value++ {
		excluded := value == -2
		if value == 3 {
			excluded = true
		}
		if excluded {
			continue
		}
		fixture_assertions(recorder, "range").
			Range_Holed_Int(value, -4, 6, -2, 3, 3, 3).
			Ensure()
	}
	builder := core.Recorder_Tree(
		&core.Recorder{}, Fixture_Subject(0), "range").
		Range_Holed_Int(3, -4, 6, -2, 3, 3, 3)
	if message := panic_text(builder.Ensure); !strings.Contains(message, "excluded") {
		t.Fatalf("panic = %q", message)
	}
	if event_count(&recorder.Events) != 8 {
		t.Fatalf("events = %d, want canonical guards, boundaries, and sentinels",
			event_count(&recorder.Events))
	}
}

// Test_Range_Registration rejects wrong arities and noncanonical static hole domains before the
// suite.
func Test_Range_Registration(t *testing.T) {
	fixtures := []struct {
		Call string
		Want string
	}{
		{"Range_Int(v, 0)", "exactly value, minimum, and maximum"},
		{"Range_Int(v, 0, 4, 1)", "exactly value, minimum, and maximum"},
		{"Range_Holed_Int(int(value), 0, 5, 0, 1, 2, 3)", "strictly inside"},
		{"Range_Holed_Int(int(value), 0, 5, 1, 2, 3, 5)", "strictly inside"},
		{"Range_Holed_Int(int(value), 0, 6, 1, 1, 2, 2)", "final-hole padding"},
		{"Range_Holed_Int(int(value), 0, 6, 2, 1, 2, 2)", "ascending"},
		{"Range_Holed_Int(int(value), 0, 6, 1, 2, 3)", "exactly four hole slots"},
		{"Range_Holed_Uint(v, 0, 6, 1, 2, 3, 3)", "exactly three hole slots"},
	}
	for _, fixture := range fixtures {
		source := BUNDLE_FIXTURE_HEAD + "." + fixture.Call +
			bundle_fixture_tail("range")
		_, output, code := registered_fixture(source)
		if code != 1 {
			t.Fatalf("%s exit=%d output=%q", fixture.Call, code, output.String())
		}
		if !strings.Contains(output.String(), fixture.Want) {
			t.Fatalf("%s exit=%d output=%q, want %q",
				fixture.Call, code, output.String(), fixture.Want)
		}
		if strings.Contains(output.String(), "legal value") {
			t.Fatalf("%s reported cardinality before its structural error: %q",
				fixture.Call, output.String())
		}
	}
}

// Test_Enum_Guard keeps successful membership visible.
func Test_Enum_Guard(t *testing.T) {
	recorder, _, _ := registered_fixture(
		bundle_fixture("enum", ".Enum_Int(int(value), 1, 2)"))
	fixture_assertions(recorder, "enum").Enum_Int(1, 1, 2).Ensure()
	guard := chain_metadata(t, recorder,
		fixture_chain_key("enum", 0, "The value is an enum member."))
	if guard.Frequency.Load() != 1 {
		t.Fatal("membership guard was not credited")
	}
}

// Test_Enum_Members keeps every canonical member mandatory and ordered by value.
func Test_Enum_Members(t *testing.T) {
	recorder, _, _ := registered_fixture(`package fixture
type Pair int
func Pair_Invariants(value Pair, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Enum_Int(int(value), -3, 9).Ensure()
}
type Triple int
func Triple_Invariants(value Triple, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Enum_3_Int(int(value), -3, 2, 9).Ensure()
}
type Quartet int
func Quartet_Invariants(value Quartet, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Enum_4_Int(int(value), -3, 0, 2, 9).Ensure()
}
func pair(value Pair) { Pair_Invariants(value, "enum.2") }
func triple(value Triple) { Triple_Invariants(value, "enum.3") }
func quartet(value Quartet) { Quartet_Invariants(value, "enum.4") }
`)
	for _, value := range []int{-3, 9} {
		core.Recorder_Tree(recorder, Pair(0), "enum.2").Enum_Int(value, -3, 9).
			Ensure()
	}
	for _, value := range []int{-3, 2, 9} {
		core.Recorder_Tree(recorder, Triple(0), "enum.3").
			Enum_3_Int(value, -3, 2, 9).Ensure()
	}
	for _, value := range []int{-3, 0, 2, 9} {
		core.Recorder_Tree(recorder, Quartet(0), "enum.4").
			Enum_4_Int(value, -3, 0, 2, 9).Ensure()
	}
	assert_enum_members(t, recorder, "enum.2", "Pair", []string{"-3", "9"})
	assert_enum_members(t, recorder, "enum.3", "Triple", []string{"-3", "2", "9"})
	assert_enum_members(t, recorder, "enum.4", "Quartet",
		[]string{"-3", "0", "2", "9"})
}

// Test_Enum_Registration rejects wrong capacities, duplicates, and nonascending domains.
func Test_Enum_Registration(t *testing.T) {
	fixtures := []struct {
		Call string
		Want string
	}{
		{"Enum_Int(v, 1)", "exactly two members"},
		{"Enum_Int(v, 1, 2, 3)", "exactly two members"},
		{"Enum_3_Int(v, 1, 2)", "exactly three members"},
		{"Enum_4_Int(int(value), 1, 2, 3)", "exactly four members"},
		{"Enum_3_Int(v, 1, 1, 2)", "exactly distinct"},
		{"Enum_4_Int(int(value), 1, 3, 2, 4)", "ascending"},
	}
	for _, fixture := range fixtures {
		source := BUNDLE_FIXTURE_HEAD + "." + fixture.Call +
			bundle_fixture_tail("enum")
		_, output, code := registered_fixture(source)
		if code != 1 {
			t.Fatalf("%s exit=%d output=%q", fixture.Call, code, output.String())
		}
		if !strings.Contains(output.String(), fixture.Want) {
			t.Fatalf("%s exit=%d output=%q, want %q",
				fixture.Call, code, output.String(), fixture.Want)
		}
	}
}

func assert_range_cardinality(t *testing.T, suffix string, value_type string) {
	t.Helper()
	for legal_count := 1; legal_count <= 5; legal_count++ {
		call := fmt.Sprintf("Range_%s(v, 0, %d)", suffix, legal_count-1)
		if legal_count == 5 {
			assert_large_range_registered(t, call, value_type)
			continue
		}
		assert_small_range_rejected(
			t, call, legal_count, range_cardinality_replacement(suffix, legal_count))
	}
}

func assert_range_holed_cardinality(t *testing.T, suffix string, value_type string) {
	t.Helper()
	holes := "1, 2, 3, 3"
	if strings.HasPrefix(suffix, "Uint") {
		holes = "1, 2, 3"
	}
	for legal_count := 2; legal_count <= 5; legal_count++ {
		call := fmt.Sprintf(
			"Range_Holed_%s(v, 0, %d, %s)", suffix, legal_count+2, holes)
		if legal_count == 5 {
			assert_large_range_registered(t, call, value_type)
			continue
		}
		assert_small_range_rejected(
			t, call, legal_count, range_cardinality_replacement(suffix, legal_count))
	}
}

func assert_small_range_rejected(
	t *testing.T, call string, legal_count int, replacement string,
) {
	t.Helper()
	source := BUNDLE_FIXTURE_HEAD + "." + call + bundle_fixture_tail("range")
	recorder, output, code := registered_fixture(source)
	want := fmt.Sprintf(
		"Range covers %d legal values; use %s instead", legal_count, replacement)
	if legal_count == 1 {
		want = "Range covers 1 legal value; use Always instead"
	}
	if code != 1 {
		t.Fatalf("%s exit=%d output=%q, want %q", call, code, output.String(), want)
	}
	if !strings.Contains(output.String(), "1 invalid bounds") {
		t.Fatalf("%s output=%q, want invalid bounds", call, output.String())
	}
	if !strings.Contains(output.String(), want) {
		t.Fatalf("%s output=%q, want %q", call, output.String(), want)
	}
	if event_count(&recorder.Events) != 0 {
		t.Fatalf("%s created partial events", call)
	}
	if recorder.Assertion_Plans != nil {
		t.Fatalf("%s created partial assertion plans", call)
	}
}

func assert_large_range_registered(t *testing.T, call string, value_type string) {
	t.Helper()
	source := "package fixture\ntype Fixture_Subject " + value_type + "\n" +
		"func Fixture_Subject_Invariants(value Fixture_Subject, " +
		"namespace invariant.Namespace) {\n\tinvariant.Tree(value, namespace)." +
		call + bundle_fixture_tail("range")
	_, output, code := registered_fixture(source)
	if code != -1 {
		t.Fatalf("%s exit=%d output=%q", call, code, output.String())
	}
}

func range_cardinality_replacement(suffix string, legal_count int) (replacement string) {
	if legal_count == 2 {
		return "Enum_" + suffix
	}
	return fmt.Sprintf("Enum_%d_%s", legal_count, suffix)
}

func assert_persisted_record_ignored(t *testing.T, name string, record string) {
	t.Helper()
	recorder := registered_single_axis(t, "persist")
	key := fixture_key("persist", 0, "axis")
	core.Recorder_Merge_Fuzz_Coverage_From(recorder, strings.NewReader(record))
	metadata := recorder_event(t, recorder, key)
	if metadata.Frequency.Load() != 0 {
		t.Fatalf("%s credited target true frequency=%d", name, metadata.Frequency.Load())
	}
	if metadata.False_Frequency.Load() != 0 {
		t.Fatalf("%s credited target frequencies=(%d,%d)", name,
			metadata.Frequency.Load(), metadata.False_Frequency.Load())
	}
}

func assert_enum_members(
	t *testing.T, recorder *core.Recorder, namespace core.Namespace,
	subject string, members []string,
) {
	t.Helper()
	guard := chain_metadata(t, recorder, chain_metadata_key{
		Package: FIXTURE_PACKAGE, Type: subject,
		Namespace: namespace, Ordinal: 0, Message: "The value is an enum member.",
	})
	if guard.Frequency.Load() == 0 {
		t.Fatalf("%s membership guard was not witnessed", namespace)
	}
	for member_index, member := range members {
		message := "The value equals member " + member + "."
		metadata := chain_metadata(t, recorder, chain_metadata_key{
			Package: FIXTURE_PACKAGE, Type: subject,
			Namespace: namespace, Ordinal: uint8(member_index + 1), Message: message,
		})
		if metadata.Frequency.Load() == 0 {
			t.Fatalf("%s %q true branch was not witnessed", namespace, message)
		}
		if metadata.False_Frequency.Load() == 0 {
			t.Fatalf("%s %q false branch was not witnessed", namespace, message)
		}
	}
}

func registered_repository_fixture(
	t *testing.T, directory string,
) (recorder *core.Recorder, output *bytes.Buffer, code *int) {
	t.Helper()
	working_directory, working_error := os.Getwd()
	if working_error != nil {
		t.Fatal(working_error)
	}
	output = &bytes.Buffer{}
	status := -1
	recorder = &core.Recorder{
		File_System: os.DirFS("/"), Working_Directory: working_directory,
		Packages_To_Analyze: []string{directory}, Output: output,
		Exit: func(exit_status int) { status = exit_status }, Is_Test: true,
		Sugar_Package: reflect.TypeOf(invariant.Sugar_Package_Marker{}).PkgPath(),
	}
	core.Recorder_Register_Packages_For_Analysis(recorder)
	if status != -1 {
		t.Fatalf("registration exit=%d output=%q", status, output.String())
	}
	return recorder, output, &status
}

// BUNDLE_FIXTURE_HEAD opens the bundle that owns Fixture_Subject, up to the point where a caller
// appends its own fluent links. bundle_fixture_tail closes it and adds the callsite.
const BUNDLE_FIXTURE_HEAD = "package fixture\ntype Fixture_Subject int\n" +
	"func Fixture_Subject_Invariants(value Fixture_Subject, " +
	"namespace invariant.Namespace) {\n\tinvariant.Tree(value, namespace)"

// Closes a bundle opened by BUNDLE_FIXTURE_HEAD and names it at one callsite.
func bundle_fixture_tail(namespace string) (source string) {
	return ".Ensure()\n}\nfunc check(value Fixture_Subject) { " +
		"Fixture_Subject_Invariants(value, \"" + namespace + "\") }\n"
}

// A pure caller wires no seam and cannot open a file, thus it keeps the whole report rather than
// naming a file that was never written.
func assert_overflow_seam_absent(t *testing.T) {
	t.Helper()
	recorder, output, _ := registered_fixture(axis_chain_fixture(21, "unwired"))
	core.Recorder_Analyze_Assertion_Frequency(recorder)
	if !strings.Contains(output.String(), "# Branch gaps (42)") {
		t.Fatalf("output = %q, want the whole report", output.String())
	}
	if strings.Contains(output.String(), "saved to") {
		t.Fatalf("output = %q, want no file named", output.String())
	}
}

// Builds a bundle whose chain is one Sometimes for each axis, so a fixture reaches the observation
// cap without a guard consuming a link beside it.
func axis_chain_fixture(axis_count int, namespace string) (source string) {
	var chain strings.Builder
	chain.WriteString(BUNDLE_FIXTURE_HEAD)
	for ordinal_index := 0; ordinal_index < axis_count; ordinal_index++ {
		fmt.Fprintf(&chain, ".Sometimes(value == 0, %q)",
			fmt.Sprintf("axis %d", ordinal_index))
	}
	chain.WriteString(bundle_fixture_tail(namespace))
	return chain.String()
}

// A bound built from two literals and one imported constant is still one static number. Each part
// is a measure over a string, thus the sum is a Go constant expression like any other.
func assert_composed_bytes_resolve(t *testing.T) {
	t.Helper()
	files := fstest.MapFS{
		"go.mod": &fstest.MapFile{Data: []byte("module fixture\n")},
		"a/a.go": &fstest.MapFile{Data: []byte(`package a
const MIDDLE = "middle"
`)},
		"b/b.go": &fstest.MapFile{Data: []byte(`package b
import part "fixture/a"
const LABEL_BYTES_MIN = 0
const LABEL_BYTES_MAX = len("prefix.") +
	len(part.MIDDLE) + len(".suffix")
type Label string
func Label_Invariants(value Label, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), LABEL_BYTES_MIN, LABEL_BYTES_MAX).Ensure()
}
func check(value Label) { Label_Invariants(value, "label") }
`)},
	}
	output := &bytes.Buffer{}
	recorder := &core.Recorder{
		File_System: files, Packages_To_Analyze: []string{"/b"}, Output: output,
		Exit: func(int) {}, Is_Test: true,
	}
	core.Recorder_Register_Packages_For_Analysis(recorder)
	if output.String() != "" {
		t.Fatalf("composed bound output=%q, want no diagnostic", output.String())
	}
	// The domain row prints the bound registration resolved, which pins the arithmetic:
	// "prefix." is 7, "middle" is 6, and ".suffix" is 7.
	core.Recorder_Analyze_Assertion_Frequency(recorder)
	if !strings.Contains(output.String(), "0..20") {
		t.Fatalf("composed bound = %q, want a 0..20 domain", output.String())
	}
}

// A constant string's length is a Go constant expression, thus a bound derived from one is static
// and a bundle should not restate the number the string already carries.
func assert_constant_bytes_resolve(t *testing.T) {
	t.Helper()
	_, output, code := registered_fixture(`package fixture
const LABEL = "a seeded label"
const LABEL_BYTES_MAX = len(LABEL)
const LABEL_BYTES_MIN = 0
type Label string
func Label_Invariants(value Label, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), LABEL_BYTES_MIN, LABEL_BYTES_MAX).Ensure()
}
func check(value Label) { Label_Invariants(value, "label") }
`)
	if code != -1 {
		t.Fatalf("len bound exit=%d output=%q", code, output.String())
	}
	assert_concatenated_bytes_resolve(t)
}

// A concatenation is one constant string, thus its measure is one static number. Spelling a wrapper
// out of its parts is how a bound stays tied to the text it bounds.
func assert_concatenated_bytes_resolve(t *testing.T) {
	t.Helper()
	recorder, output, code := registered_fixture(`package fixture
const ENVELOPE_BYTES_MAX = 32 - len(
	"<a><b>"+
		"</b></a>")
const ENVELOPE_BYTES_MIN = 0
type Envelope string
func Envelope_Invariants(value Envelope, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), ENVELOPE_BYTES_MIN, ENVELOPE_BYTES_MAX).Ensure()
}
func check(value Envelope) { Envelope_Invariants(value, "envelope") }
`)
	if code != -1 {
		t.Fatalf("concatenated bound exit=%d output=%q", code, output.String())
	}
	// The domain row prints the bound registration resolved. "<a><b>" is 6 bytes and
	// "</b></a>" is 8, thus the wrapper is 14 and the payload bound is 18.
	core.Recorder_Analyze_Assertion_Frequency(recorder)
	if !strings.Contains(output.String(), "0..18") {
		t.Fatalf("concatenated bound = %q, want a 0..18 domain", output.String())
	}
}

// CONSTANT_EXPRESSION_HEAD declares the strings each bound below measures or converts.
const CONSTANT_EXPRESSION_HEAD = "package fixture\n" +
	"const OPEN = \"<a>\"\n" +
	"const BODY = \"body\"\n" +
	"const CLOSE = \"</a>\"\n" +
	"const SPAN_MINIMUM = 0\n"

// Drives one bound through registration and reads the resolved number back out of the domain row,
// which is what proves the arithmetic rather than the absence of a diagnostic.
func assert_constant_expression_bound(t *testing.T, expression string, declared string) {
	t.Helper()
	recorder, output, code := registered_fixture(CONSTANT_EXPRESSION_HEAD +
		"const SPAN_MAXIMUM = " + expression + "\n" + `type Span string
func Span_Invariants(value Span, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SPAN_MINIMUM, SPAN_MAXIMUM).Ensure()
}
func check(value Span) { Span_Invariants(value, "span") }
`)
	if code != -1 {
		t.Fatalf("%s exit=%d output=%q", expression, code, output.String())
	}
	core.Recorder_Analyze_Assertion_Frequency(recorder)
	if !strings.Contains(output.String(), declared) {
		t.Fatalf("%s declared %q, want %q", expression, output.String(), declared)
	}
}

// A value its type cannot hold is not a constant, thus a bound built from one is refused rather
// than wrapped into a number the source never states.
func assert_unrepresentable_conversion(t *testing.T) {
	t.Helper()
	for _, operand := range []string{"uint8(300)", "int8(200)", "uint16(70000)"} {
		_, output, code := registered_fixture(CONSTANT_EXPRESSION_HEAD +
			"const SPAN_MAXIMUM = " + operand + "\n" + `type Span string
func Span_Invariants(value Span, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SPAN_MINIMUM, SPAN_MAXIMUM).Ensure()
}
func check(value Span) { Span_Invariants(value, "span") }
`)
		if code != 1 {
			t.Fatalf("%s exit=%d output=%q", operand, code, output.String())
		}
		if !strings.Contains(output.String(), "not statically resolvable") {
			t.Fatalf("%s output=%q, want unresolvable", operand, output.String())
		}
	}
}

// A comparison and a conjunction are constant expressions, thus a guard built from them states a
// fact registration can already settle and owes no runtime evidence.
func assert_constant_condition(t *testing.T) {
	t.Helper()
	conditions := []string{
		"len(OPEN) < len(CLOSE)",
		"len(OPEN) > 0 && len(CLOSE) > 0",
		"len(OPEN) > 99 || len(CLOSE) > 0",
	}
	for _, condition := range conditions {
		_, output, code := registered_fixture(CONSTANT_EXPRESSION_HEAD +
			"func check() { invariant.Always(" + condition + ", \"constant\") }\n")
		if code != 1 {
			t.Fatalf("%s exit=%d output=%q", condition, code, output.String())
		}
		if !strings.Contains(output.String(), "Always condition is constant true") {
			t.Fatalf("%s output=%q, want a constant diagnostic",
				condition, output.String())
		}
	}
}

// An operand the language does not call constant stays unresolved, thus the evaluator never guesses
// a bound from a value only the run can produce.
func assert_non_constant_operand(t *testing.T) {
	t.Helper()
	operands := []string{"measure()", "variable", "OPEN[0:1]"}
	for _, operand := range operands {
		_, output, code := registered_fixture(CONSTANT_EXPRESSION_HEAD +
			"var variable = 4\nfunc measure() (count int) { return 4 }\n" +
			`type Span string
func Span_Invariants(value Span, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SPAN_MINIMUM, ` + operand + `).Ensure()
}
func check(value Span) { Span_Invariants(value, "span") }
`)
		if code != 1 {
			t.Fatalf("%s exit=%d output=%q", operand, code, output.String())
		}
		if !strings.Contains(output.String(), "not statically resolvable") {
			t.Fatalf("%s output=%q, want unresolvable", operand, output.String())
		}
	}
}

// A foreign constant is read with the names its own file could see. The importing package declares
// the same bare name at a different value, thus a scope that failed to travel would resolve 512 in
// silence rather than the 16 the source states.
func assert_foreign_declaration_scope(t *testing.T) {
	t.Helper()
	files := fstest.MapFS{
		"go.mod": &fstest.MapFile{Data: []byte("module fixture\n")},
		"a/a.go": &fstest.MapFile{Data: []byte(`package a
const BITS = 4
const SCALE = 1 << BITS
`)},
		"b/b.go": &fstest.MapFile{Data: []byte(`package b
import width "fixture/a"
const BITS = 9
const SPAN_MINIMUM = 0
type Span string
func Span_Invariants(value Span, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SPAN_MINIMUM, width.SCALE).Ensure()
}
func check(value Span) { Span_Invariants(value, "span") }
`)},
	}
	output := &bytes.Buffer{}
	recorder := &core.Recorder{
		File_System: files, Packages_To_Analyze: []string{"/b"}, Output: output,
		Exit: func(int) {}, Is_Test: true,
	}
	core.Recorder_Register_Packages_For_Analysis(recorder)
	if output.String() != "" {
		t.Fatalf("foreign declaration output=%q, want no diagnostic", output.String())
	}
	core.Recorder_Analyze_Assertion_Frequency(recorder)
	if strings.Contains(output.String(), "0..512") {
		t.Fatalf("output = %q, want the importer's BITS to be invisible", output.String())
	}
	if !strings.Contains(output.String(), "0..16") {
		t.Fatalf("output = %q, want a 0..16 domain", output.String())
	}
}

// Builds two packages: one owning the shared bounds, one whose bundle names them as its Range.
func cross_package_constant_fixture(bounds string) (files fstest.MapFS) {
	return fstest.MapFS{
		"go.mod": &fstest.MapFile{Data: []byte("module fixture\n")},
		"a/a.go": &fstest.MapFile{Data: []byte(`package a
const SPAN_MINIMUM = -4
const SPAN_MAXIMUM = 4
`)},
		"b/b.go": &fstest.MapFile{Data: []byte(`package b
import bound "fixture/a"
type Span int
func Span_Invariants(value Span, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Range_Int(int(value), ` + bounds + `).Ensure()
}
func check(value Span) { Span_Invariants(value, "root") }
`)},
	}
}

// Wraps one fluent chain in the bundle that owns its subject type, plus the callsite that names it.
// A chain lives only in an _Invariants bundle, thus a fixture exercising a chain declares one. The
// links keep whatever the caller writes, because a fixture is parsed and never type-checked.
func bundle_fixture(namespace string, links string) (source string) {
	return "package fixture\n" +
		"type Fixture_Subject int\n" +
		"func Fixture_Subject_Invariants(" +
		"value Fixture_Subject, namespace invariant.Namespace) {\n" +
		"\tinvariant.Tree(value, namespace)" + links + ".Ensure()\n}\n" +
		"func check(value Fixture_Subject) { Fixture_Subject_Invariants(value, \"" +
		namespace + "\") }\n"
}

// Two types that share their constants state the same property under one namespace, which the
// design calls for. Their rows then agree on every other field, thus the subject type is the only
// handle a reader has on which of them lacks evidence.
func analysis_gap_names_its_subject(t *testing.T) {
	recorder, output, _ := registered_fixture(`package fixture
type Alpha int
func Alpha_Invariants(value Alpha, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "axis").Ensure()
}
type Beta int
func Beta_Invariants(value Beta, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "axis").Ensure()
}
type Fixture_Subject struct {
	A Alpha
	B Beta
}
func Fixture_Subject_Invariants(value Fixture_Subject, namespace invariant.Namespace) {
	Alpha_Invariants(value.A, namespace)
	Beta_Invariants(value.B, namespace)
}
func check(value Fixture_Subject) { Fixture_Subject_Invariants(value, "same") }
`)
	core.Recorder_Analyze_Assertion_Frequency(recorder)
	for _, subject := range []string{"| Alpha ", "| Beta "} {
		if !strings.Contains(output.String(), subject) {
			t.Fatalf("gap report has no %q cell: %q", subject, output.String())
		}
	}
}

func registered_fixture(source string) (
	recorder *core.Recorder, output *bytes.Buffer, code int,
) {
	return registered_fixture_options(source, "")
}

// FIXTURE_PACKAGE is the import path registered_fixture gives its one package. It suits a test
// that only registers, because no runtime chain has to agree with it.
const FIXTURE_PACKAGE = "local/james-orcales/shared/invariant_test"

// Pair mirrors the enum-width fixture's two-member subject, so a runtime chain resolves the same
// plan that fixture registered. Triple and Quartet do the same for three and four members.
type Pair int

// Triple is the three-member enum subject.
type Triple int

// Quartet is the four-member enum subject.
type Quartet int

func registered_global_namespace_fixture() (
	recorder *core.Recorder, output *bytes.Buffer, code int,
) {
	output = &bytes.Buffer{}
	code = -1
	recorder = &core.Recorder{
		File_System: fstest.MapFS{
			"go.mod": &fstest.MapFile{Data: []byte("module fixture\n")},
			"a/a.go": &fstest.MapFile{Data: []byte(`package a
type Fixture_Subject int
func Fixture_Subject_Invariants(value Fixture_Subject, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "a").Ensure()
}
func check(value Fixture_Subject) { Fixture_Subject_Invariants(value, "global") }
`)},
			"b/b.go": &fstest.MapFile{Data: []byte(`package b
type Other_Subject int
func Other_Subject_Invariants(value Other_Subject, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "b").Ensure()
}
func check(value Other_Subject) { Other_Subject_Invariants(value, "global") }
`)},
		},
		Packages_To_Analyze: []string{"/a", "/b"}, Output: output,
		Exit: func(status int) { code = status }, Is_Test: true,
	}
	core.Recorder_Register_Packages_For_Analysis(recorder)
	return recorder, output, code
}

func registered_fixture_with_sugar(source string) (
	recorder *core.Recorder, output *bytes.Buffer, code int,
) {
	return registered_fixture_options(source, "fixture/fixture")
}

func registered_fixture_with_test(source string, test_source string) (
	recorder *core.Recorder, output *bytes.Buffer, code int,
) {
	output = &bytes.Buffer{}
	code = -1
	recorder = &core.Recorder{
		File_System: fstest.MapFS{
			"go.mod": &fstest.MapFile{
				Data: []byte("module local/james-orcales/shared\n"),
			},
			"invariant_test/check.go":      &fstest.MapFile{Data: []byte(source)},
			"invariant_test/check_test.go": &fstest.MapFile{Data: []byte(test_source)},
		},
		Packages_To_Analyze: []string{"/invariant_test"}, Output: output,
		Exit: func(status int) { code = status }, Is_Test: true,
	}
	core.Recorder_Register_Packages_For_Analysis(recorder)
	return recorder, output, code
}

func registered_fixture_options(source string, sugar string) (
	recorder *core.Recorder, output *bytes.Buffer, code int,
) {
	output = &bytes.Buffer{}
	code = -1
	recorder = &core.Recorder{
		File_System: fstest.MapFS{
			"go.mod": &fstest.MapFile{
				Data: []byte("module local/james-orcales/shared\n"),
			},
			"invariant_test/check.go": &fstest.MapFile{Data: []byte(source)},
		},
		Packages_To_Analyze: []string{"/invariant_test"}, Output: output,
		Exit: func(status int) { code = status }, Is_Test: true,
		Sugar_Package: sugar,
	}
	core.Recorder_Register_Packages_For_Analysis(recorder)
	return recorder, output, code
}

type chain_metadata_key struct {
	Namespace core.Namespace
	Package   string
	Type      string
	Ordinal   uint8
	Message   string
}

func assertion_key(key chain_metadata_key) (encoded string) {
	separator := core.ELEMENT_MESSAGE_SEPARATOR
	return string(key.Namespace) + separator + key.Package + separator + key.Type +
		separator + fmt.Sprint(key.Ordinal) + separator + key.Message
}

func chain_metadata(
	t *testing.T, recorder *core.Recorder, key chain_metadata_key,
) (metadata *core.Assertion_Metadata) {
	t.Helper()
	return recorder_event(t, recorder, assertion_key(key))
}

// Gives the tracker entry of one expanded axis of an inline helper. An inline key has the same
// shape as a chain key, with the helper's own message in the namespace position and no subject.
func inline_metadata(
	t *testing.T, recorder *core.Recorder, message string, ordinal uint8, axis string,
) (metadata *core.Assertion_Metadata) {
	t.Helper()
	return recorder_event(t, recorder, assertion_key(chain_metadata_key{
		Namespace: core.Namespace(message), Ordinal: ordinal, Message: axis,
	}))
}

func recorder_event(
	t *testing.T, recorder *core.Recorder, key string,
) (metadata *core.Assertion_Metadata) {
	t.Helper()
	value, exists := recorder.Events.Load(key)
	if !exists {
		var registered []string
		recorder.Events.Range(func(other any, _ any) (continue_iteration bool) {
			registered = append(registered, fmt.Sprintf("%q", other))
			return true
		})
		sort.Strings(registered)
		t.Fatalf("missing event %q\nregistered:\n  %s",
			key, strings.Join(registered, "\n  "))
	}
	return value.(*core.Assertion_Metadata)
}

func event_count(events *sync.Map) (count int) {
	events.Range(func(_, _ any) (continue_iteration bool) {
		count++
		return true
	})
	return count
}

func panic_text(action func()) (message string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			message = fmt.Sprint(recovered)
		}
	}()
	action()
	return ""
}

func registered_single_axis(t *testing.T, namespace string) (recorder *core.Recorder) {
	t.Helper()
	source := bundle_fixture(namespace, ".Sometimes(value == 0, \"axis\")")
	recorder, output, code := registered_fixture(source)
	if code != -1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	return recorder
}

// Gives the expected tracker key of one link in a registration-only fixture chain. These tests
// never call Assertions at run time, so the fixture's own import path is the right one.
func fixture_key(namespace string, ordinal uint8, message string) (key string) {
	return assertion_key(fixture_chain_key(namespace, ordinal, message))
}

// Gives the expected chain key of one link in a fixture whose subject is Fixture_Subject. Every
// such fixture shares one package and one subject type, thus only the three varying fields remain
// at a callsite.
func fixture_chain_key(
	namespace string, ordinal uint8, message string,
) (key chain_metadata_key) {
	return fixture_chain_key_typed(namespace, "Fixture_Subject", ordinal, message)
}

// Gives the expected chain key of one link in a fixture whose subject type is not the default.
func fixture_chain_key_typed(
	namespace string, subject string, ordinal uint8, message string,
) (key chain_metadata_key) {
	return chain_metadata_key{
		Namespace: core.Namespace(namespace), Package: FIXTURE_PACKAGE,
		Type: subject, Ordinal: ordinal, Message: message,
	}
}

func registered_range(
	t *testing.T, namespace string, minimum int, maximum int,
) (recorder *core.Recorder, output *bytes.Buffer, code int) {
	t.Helper()
	source := BUNDLE_FIXTURE_HEAD +
		fmt.Sprintf(".Range_Int(int(value), %d, %d)", minimum, maximum) +
		bundle_fixture_tail(namespace)
	return registered_fixture(source)
}

func registered_nested_bundle(t *testing.T) (recorder *core.Recorder) {
	t.Helper()
	recorder, output, code := registered_fixture(`package fixture
type Inner int
func Inner_Invariants(value Inner, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "zero").Ensure()
}
type Outer struct { Inner Inner }
func Outer_Invariants(value Outer, namespace invariant.Namespace) {
	Inner_Invariants(value.Inner, namespace)
}
func check(value Outer) { Outer_Invariants(value, "outer") }
`)
	if code != -1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	return recorder
}
