//go:build !invariant_disable_coverage && !prd && !prod && !production && !invariant_noop

package invariant_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"local/james-orcales/shared/invariant"
	default_invariant "local/james-orcales/shared/invariant/default"
)

// Test_Always_Violation prevents a false guard from returning.
func Test_Always_Violation(t *testing.T) {
	message := panic_text(func() {
		invariant.Recorder_Always(&invariant.Recorder{}, false, "guard")
	})
	if !strings.Contains(message, "guard") {
		t.Fatalf("panic = %q", message)
	}
}

// Test_Always_Eager prevents guard enforcement from moving to a later boundary.
func Test_Always_Eager(t *testing.T) {
	action := func() {
		invariant.Recorder_Always(&invariant.Recorder{}, false, "eager")
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
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	want := "🚨 1 coverage gaps 🚨\n\n" +
		"# Reachability gaps (1)\n\n" +
		"| Assertion | Source |\n" +
		"|-----------|--------|\n" +
		"| reachable | ok     |\n\n" +
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
		"/fixture/check.go:4  Always condition is constant true\n" +
		"/fixture/check.go:5  Always condition is constant true\n" +
		"/fixture/check.go:6  Always condition is constant true\n" +
		"/fixture/check.go:7  Always condition is constant true\n" +
		"/fixture/check.go:8  Always condition is constant true\n" +
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
		"/fixture/check.go:3  duplicate message: \"duplicate\"\n" +
		"🚨 1 duplicate messages 🚨\n"
	if output.String() != want {
		t.Fatalf("output=%q, want %q", output.String(), want)
	}
	if event_count(&recorder.Events) != 0 {
		t.Fatal("duplicate eager identities created partial events")
	}
}

// Test_Sometimes_Deferred protects Ensure as the only emission boundary.
func Test_Sometimes_Deferred(t *testing.T) {
	recorder := registered_single_axis(t, "deferred")
	metadata := chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "deferred", Ordinal: 0, Message: "axis",
	})
	builder := invariant.Recorder_Assertions(recorder, "deferred").Sometimes(true, "axis")
	if metadata.Frequency.Load() != 0 {
		t.Fatal("link credited eagerly")
	}
	builder.Ensure()
}

// Test_Sometimes_Coverage protects independent branch credit.
func Test_Sometimes_Coverage(t *testing.T) {
	recorder := registered_single_axis(t, "coverage")
	invariant.Recorder_Assertions(recorder, "coverage").Sometimes(true, "axis").Ensure()
	metadata := chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "coverage", Ordinal: 0, Message: "axis",
	})
	if metadata.Frequency.Load() != 1 {
		t.Fatal("Ensure did not credit the true branch")
	}
	if metadata.False_Frequency.Load() != 0 {
		t.Fatal("Ensure credited the wrong branch")
	}
}

// Test_Sometimes_Gap keeps both branches mandatory.
func Test_Sometimes_Gap(t *testing.T) {
	recorder, output, _ := registered_fixture(`package fixture
func check(value bool) { invariant.Assertions("gap").Sometimes(value, "axis").Ensure() }
`)
	invariant.Recorder_Assertions(recorder, "gap").Sometimes(true, "axis").Ensure()
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	want := "🚨 1 coverage gaps 🚨\n\n" +
		"# Branch gaps (1)\n\n" +
		"| Assertion | Link | Missing | Property | Source |\n" +
		"|-----------|-----:|---------|----------|--------|\n" +
		"| gap       |    0 | false   | axis     | value  |\n\n" +
		"🚨 1 coverage gaps 🚨\n"
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

// Test_Assertions_API pins the fluent surface shared by primitive presets.
func Test_Assertions_API(t *testing.T) {
	invariant.Recorder_Assertions(&invariant.Recorder{}, "api").
		Sometimes(true, "axis").Range_Int(1, 0, 2).Enum_Int(1, 1, 2).Ensure()
}

// Test_Assertions_Identity protects namespace, ordinal, and message identity.
func Test_Assertions_Identity(t *testing.T) {
	recorder, _, _ := registered_fixture(`package fixture
func check(a bool, b bool) {
	invariant.Assertions("identity").Sometimes(a, "same").Sometimes(b, "same").Ensure()
}
`)
	first := chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "identity", Ordinal: 0, Message: "same",
	})
	second := chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "identity", Ordinal: 1, Message: "same",
	})
	invariant.Recorder_Assertions(recorder, "identity").
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

// Test_Assertions_Atomic prevents partial coverage from a failing chain.
func Test_Assertions_Atomic(t *testing.T) {
	recorder, _, _ := registered_fixture(`package fixture
const Minimum = 0
const Maximum = 4
func check(value int) {
	invariant.Assertions("atomic").Sometimes(true, "axis").Range_Int(value, Minimum, Maximum).Ensure()
}
`)
	builder := invariant.Recorder_Assertions(recorder, "atomic").
		Sometimes(true, "axis").Range_Int(3, 0, 2)
	if panic_text(builder.Ensure) == "" {
		t.Fatal("invalid chain did not panic")
	}
	if count_covered(recorder) != 0 {
		t.Fatal("failed Ensure was not atomic")
	}
	plan := recorder.Assertion_Plans["atomic"]
	plan.Links[1].Entry.Metadata = nil
	unknown := invariant.Recorder_Assertions(recorder, "atomic").
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
	recorder := &invariant.Recorder{Is_Test: true}
	invariant.Recorder_Assertions(recorder, "foreign").Sometimes(true, "axis").Ensure()
	if event_count(&recorder.Events) != 0 {
		t.Fatal("foreign chain created coverage")
	}
	message := panic_text(func() {
		invariant.Recorder_Assertions(recorder, "foreign").Range_Int(3, 0, 2).Ensure()
	})
	if message == "" {
		t.Fatal("foreign preset was not enforced")
	}
}

// Test_Assertions_Allocation protects the zero-allocation runtime path.
func Test_Assertions_Allocation(t *testing.T) {
	recorder := &invariant.Recorder{}
	allocations := testing.AllocsPerRun(1000, func() {
		invariant.Recorder_Assertions(recorder, "allocation").
			Sometimes(true, "axis").Range_Int(1, 0, 2).Ensure()
	})
	if allocations != 0 {
		t.Fatalf("allocations = %f", allocations)
	}
}

// Test_Assertions_Persistence protects structural fuzz-key round trips.
func Test_Assertions_Persistence(t *testing.T) {
	key := assertion_key("persist", 0, "axis")
	recorder := registered_single_axis(t, "persist")
	invariant.Recorder_Merge_Fuzz_Coverage_From(
		recorder, strings.NewReader(invariant.Fuzz_Coverage_Line(key, true)))
	if recorder_event(t, recorder, key).Frequency.Load() != 1 {
		t.Fatal("persisted true branch did not merge")
	}
	recorder = registered_single_axis(t, "persist")
	invariant.Recorder_Merge_Fuzz_Coverage_From(
		recorder, strings.NewReader(invariant.Fuzz_Coverage_Line(key, false)))
	if recorder_event(t, recorder, key).False_Frequency.Load() != 1 {
		t.Fatal("persisted false branch did not merge")
	}
}

// Test_Assertions_Record_Validation prevents foreign or malformed fuzz records from crediting an
// exact registered identity.
func Test_Assertions_Record_Validation(t *testing.T) {
	key := assertion_key("persist", 0, "axis")
	encoded_key := base64.StdEncoding.EncodeToString([]byte(key))
	assert_persisted_record_ignored(t, "foreign namespace",
		invariant.Fuzz_Coverage_Line(assertion_key("foreign", 0, "axis"), true))
	assert_persisted_record_ignored(t, "foreign ordinal",
		invariant.Fuzz_Coverage_Line(assertion_key("persist", 1, "axis"), true))
	assert_persisted_record_ignored(t, "unknown key",
		invariant.Fuzz_Coverage_Line("unknown", true))
	assert_persisted_record_ignored(t, "invalid base64", "%%%\tT\n")
	assert_persisted_record_ignored(t, "missing separator", encoded_key+"\n")
	assert_persisted_record_ignored(t, "partial record", encoded_key+"\tT")
	assert_persisted_record_ignored(t, "invalid branch", encoded_key+"\tX\n")
}

// Test_Assertions_Registration_Packages keeps direct source registration independent of reach.
func Test_Assertions_Registration_Packages(t *testing.T) {
	recorder, output, code := registered_fixture(`package fixture
func check(value bool) {
	invariant.Always(value, "guard")
	invariant.Assertions("direct").Sometimes(value, "axis").Ensure()
}
`)
	if code != -1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	if event_count(&recorder.Events) != 2 {
		t.Fatalf("events = %d, want both direct source roots",
			event_count(&recorder.Events))
	}
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "direct", Ordinal: 0, Message: "axis",
	})
}

// Test_Assertions_Registration_Test_Source prevents test code from emitting production coverage.
func Test_Assertions_Registration_Test_Source(t *testing.T) {
	recorder, output, code := registered_fixture_with_test(`package fixture
func check(value bool) {
	invariant.Always(value, "guard")
	invariant.Assertions("production").Sometimes(value, "axis").Ensure()
}
`, `package fixture
func direct(value bool) {
	invariant.Always(value, "test guard")
	invariant.Assertions("test").Sometimes(value, "test axis").Ensure()
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
		"/fixture/check_test.go:3  test source calls Always\n" +
		"/fixture/check_test.go:4  test source calls Assertions\n" +
		"/fixture/check_test.go:5  test source calls Int_Invariants\n" +
		"/fixture/check_test.go:8  test source references Int_Invariants\n" +
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
		Namespace: "outer", Ordinal: 0, Message: "zero",
	})
}

// Test_Assertions_Registration_Walk keeps registration expansion aligned with runtime ordinals.
func Test_Assertions_Registration_Walk(t *testing.T) {
	recorder, _, code := registered_fixture(`package fixture
const Minimum = 0
const Maximum = 4
func check(value int, flag bool) {
	invariant.Assertions("walk").Sometimes(flag, "flag").Range_Int(value, Minimum, Maximum).Ensure()
}
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
	invariant.Assertions(namespace).Sometimes(value == 0, "zero").Ensure()
}
func check(value Number) { Number_Invariants(value, "number") }
`)
	if code != -1 {
		t.Fatalf("exit = %d", code)
	}
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "number", Ordinal: 0, Message: "zero",
	})
}

// Test_Assertions_Registration_Ensured rejects chains with no atomic boundary.
func Test_Assertions_Registration_Ensured(t *testing.T) {
	_, output, code := registered_fixture(`package fixture
func check(value bool) { invariant.Assertions("dangling").Sometimes(value, "axis") }
`)
	if code != 1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "not terminated by Ensure") {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	_, output, code = registered_fixture(`package fixture
func empty() { invariant.Assertions("empty").Ensure() }
`)
	if code != 1 {
		t.Fatalf("empty exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "has no links") {
		t.Fatalf("empty exit=%d output=%q", code, output.String())
	}
	_, output, code = registered_fixture(`package fixture
func build(value bool) invariant.Assertion_Builder {
	return invariant.Assertions("returned").Sometimes(value, "axis")
}
`)
	if code != 1 {
		t.Fatalf("returned exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "not terminated by Ensure") {
		t.Fatalf("returned exit=%d output=%q", code, output.String())
	}
}

// Test_Assertions_Registration_Literal keeps coverage identities statically knowable.
func Test_Assertions_Registration_Literal(t *testing.T) {
	_, output, code := registered_fixture(`package fixture
func check(value bool, message string) {
	invariant.Assertions("literal").Sometimes(value, message).Ensure()
}
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
	invariant.Assertions(namespace).Sometimes(value == 0, "zero").Ensure()
}
func first(value Number) { Number_Invariants(value, "number") }
func second(value Number) { Number_Invariants(value, "number") }
`)
	if code != 1 {
		t.Fatalf("reuse exit=%d output=%q", code, output.String())
	}
	want := "🚨 1 duplicate bundle namespaces 🚨\n" +
		"/fixture/check.go:7  banned: duplicate namespace \"number\"\n" +
		"🚨 1 duplicate bundle namespaces 🚨\n" +
		"🚨 1 duplicate messages 🚨\n" +
		"/fixture/check.go:7  duplicate namespace: \"number\"\n" +
		"🚨 1 duplicate messages 🚨\n"
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
func first(v bool) { invariant.Assertions("same").Sometimes(v, "a").Ensure() }
func second(v bool) { invariant.Assertions("same").Sometimes(v, "b").Ensure() }
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
	want = "🚨 1 duplicate messages 🚨\n" +
		"/b/b.go:2  duplicate namespace: \"global\"\n" +
		"🚨 1 duplicate messages 🚨\n"
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
	invariant.Assertions(namespace).Sometimes(value == 0, "zero").Ensure()
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
		Namespace: "first", Ordinal: 0, Message: "zero",
	})
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "second", Ordinal: 0, Message: "zero",
	})
}

// Test_Assertions_Registration_Source_Path prevents forwarded chains from merging coverage.
func Test_Assertions_Registration_Source_Path(t *testing.T) {
	recorder, output, code := registered_fixture(`package fixture
type First int
type Second int
type Parent struct { First First; Second Second }
func First_Invariants(value First, namespace invariant.Namespace) {
	invariant.Assertions(namespace).Sometimes(value == 0, "first").Ensure()
}
func Second_Invariants(value Second, namespace invariant.Namespace) {
	invariant.Assertions(namespace).Sometimes(value == 1, "second").Ensure()
}
func Parent_Invariants(value Parent, namespace invariant.Namespace) {
	First_Invariants(value.First, namespace)
	Second_Invariants(value.Second, namespace)
}
func check(value Parent) { Parent_Invariants(value, "same") }
`)
	if code != 1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	want := "🚨 1 duplicate messages 🚨\n" +
		"/fixture/check.go:13  duplicate namespace: \"same\"\n" +
		"🚨 1 duplicate messages 🚨\n"
	if output.String() != want {
		t.Fatalf("output=%q, want %q", output.String(), want)
	}
	if event_count(&recorder.Events) != 0 {
		t.Fatal("different forwarded chains published partial events")
	}
	if recorder.Assertion_Plans != nil {
		t.Fatal("different forwarded chains published a partial plan")
	}
}

// Test_Assertions_Registration_Caps keeps registration within the fixed runtime representation.
func Test_Assertions_Registration_Caps(t *testing.T) {
	var accepted strings.Builder
	accepted.WriteString("package fixture\n" +
		"func accepted(v bool) { invariant.Assertions(\"accepted\")")
	for ordinal_index := 0; ordinal_index < 70; ordinal_index++ {
		fmt.Fprintf(&accepted, ".Sometimes(v, %q)", fmt.Sprintf("axis %d", ordinal_index))
	}
	accepted.WriteString(".Ensure() }\n")
	recorder, output, code := registered_fixture(accepted.String())
	if code != -1 {
		t.Fatalf("70 links exit=%d output=%q", code, output.String())
	}
	if event_count(&recorder.Events) != 70 {
		t.Fatalf("70 links registered %d events", event_count(&recorder.Events))
	}
	var source strings.Builder
	source.WriteString("package fixture\nfunc check(v bool) { invariant.Assertions(\"cap\")")
	for ordinal_index := 0; ordinal_index <= 70; ordinal_index++ {
		fmt.Fprintf(&source, ".Sometimes(v, %q)", fmt.Sprintf("axis %d", ordinal_index))
	}
	source.WriteString(".Ensure() }\n")
	_, output, code = registered_fixture(source.String())
	if code != 1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "70 links") {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	var presets strings.Builder
	presets.WriteString(
		"package fixture\nfunc check(v int) { invariant.Assertions(\"presets\")")
	for preset_index := 0; preset_index < 15; preset_index++ {
		presets.WriteString(".Enum_4_Int(v, 0, 1, 2, 3)")
	}
	presets.WriteString(".Ensure() }\n")
	_, output, code = registered_fixture(presets.String())
	if code != 1 {
		t.Fatalf("75 expanded links exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "70 links") {
		t.Fatalf("75 expanded links exit=%d output=%q", code, output.String())
	}
}

// Test_Bundles_Static keeps template expansion independent of runtime control flow.
func Test_Bundles_Static(t *testing.T) {
	_, output, code := registered_fixture(`package fixture
type Number int
func Number_Invariants(value Number, namespace invariant.Namespace) {
	if value == 0 { invariant.Assertions(namespace).Sometimes(true, "zero").Ensure() }
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
	invariant.Assertions(namespace).Sometimes(value == 0, "zero").Ensure()
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
	invariant.Assertions(namespace).Sometimes(value == 0, "zero").Ensure()
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
		Namespace: "outer", Ordinal: 0, Message: "zero",
	})
}

// Test_Bundles_Composition keeps a composed chain free of a cross product.
func Test_Bundles_Composition(t *testing.T) {
	recorder := registered_nested_bundle(t)
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "outer", Ordinal: 0, Message: "zero",
	})
	if event_count(&recorder.Events) != 1 {
		t.Fatalf("events=%d, want the composed axis alone", event_count(&recorder.Events))
	}
}

// Test_Bundles_Casing accepts the repository's two invariant-helper casings.
func Test_Bundles_Casing(t *testing.T) {
	recorder, _, _ := registered_fixture(`package fixture
type number int
func number_invariants(value number, namespace invariant.Namespace) {
	invariant.Assertions(namespace).Sometimes(value == 0, "zero").Ensure()
}
func check(value number) { number_invariants(value, "number") }
`)
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "number", Ordinal: 0, Message: "zero",
	})
}

// Test_Bundles_Sugar protects unqualified roots only inside the configured sugar package.
func Test_Bundles_Sugar(t *testing.T) {
	recorder, _, code := registered_fixture_with_sugar(`package invariant
type Number int
func Number_Invariants(value Number, namespace Namespace) {
	Assertions(namespace).Sometimes(value == 0, "zero").Ensure()
}
func check(value Number) { Number_Invariants(value, "number") }
`)
	if code != -1 {
		t.Fatalf("exit = %d", code)
	}
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "number", Ordinal: 0, Message: "zero",
	})
}

// Test_Bundles_Cross_Package protects indexed template expansion across imports.
func Test_Bundles_Cross_Package(t *testing.T) {
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"go.mod": &fstest.MapFile{Data: []byte("module fixture\n")},
			"a/a.go": &fstest.MapFile{Data: []byte(`package a
type Number int
func Number_Invariants(value Number, namespace invariant.Namespace) {
	invariant.Assertions(namespace).Sometimes(value == 0, "zero").Ensure()
	Position_Invariants(Position(value), "position")
}
type Position int
func Position_Invariants(value Position, namespace invariant.Namespace) {
	invariant.Assertions(namespace).Sometimes(value > 0, "positive").Ensure()
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
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "number", Ordinal: 0, Message: "zero",
	})
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "position", Ordinal: 0, Message: "positive",
	})
}

// Test_Bundles_Callsite keeps each invocation's namespace independent.
func Test_Bundles_Callsite(t *testing.T) {
	recorder, output, _ := registered_fixture(`package fixture
type Number int
func Number_Invariants(value Number, namespace invariant.Namespace) {
	invariant.Assertions(namespace).Sometimes(value == 0, "zero").Ensure()
}
func first(value Number) { Number_Invariants(value, "first") }
func second(value Number) { Number_Invariants(value, "second") }
`)
	first := chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "first", Ordinal: 0, Message: "zero",
	})
	second := chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "second", Ordinal: 0, Message: "zero",
	})
	invariant.Recorder_Assertions(recorder, "first").Sometimes(true, "zero").Ensure()
	if first.Frequency.Load() != 1 {
		t.Fatal("first callsite did not receive its own credit")
	}
	if second.Frequency.Load() != 0 {
		t.Fatal("first callsite credited the second true branch")
	}
	if second.False_Frequency.Load() != 0 {
		t.Fatal("first callsite credited the second namespace")
	}
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	want := "🚨 3 coverage gaps 🚨\n\n" +
		"# Branch gaps (3)\n\n" +
		"| Assertion | Link | Missing | Property | Source     |\n" +
		"|-----------|-----:|---------|----------|------------|\n" +
		"| first     |    0 | false   | zero     | value == 0 |\n" +
		"| second    |    0 | false   | zero     | value == 0 |\n" +
		"| second    |    0 | true    | zero     | value == 0 |\n\n" +
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
	invariant.Assertions(namespace).Sometimes(value == 0, "zero").Ensure()
}
func check(value Number) { Number_Invariants(value, "number") }
`)
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	if !strings.Contains(output.String(), "| number    |    0 | false   | zero") {
		t.Fatalf("output = %q", output.String())
	}
}

// Test_Bundles_Custom_Types prevents primitive helpers from replacing typed domain helpers.
func Test_Bundles_Custom_Types(t *testing.T) {
	_, output, code := registered_fixture(`package fixture
func Int_Invariants(value int, namespace invariant.Namespace) {
	invariant.Assertions(namespace).Sometimes(value == 0, "zero").Ensure()
}
`)
	if code != 1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "primitive") {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
}

// Test_Bundles_Signed_Primitive_Mandates proves each signed helper's exact source creates both
// branches for every named witness.
func Test_Bundles_Signed_Primitive_Mandates(t *testing.T) {
	recorder, _, _ := registered_repository_fixture(t, "testdata/primitive_mandates")
	previous := default_invariant.Default
	default_invariant.Default = recorder
	defer func() { default_invariant.Default = previous }()
	exercise_signed_primitive_mandates()
	assert_primitive_mandates(t, recorder, signed_primitive_mandates())
}

// Test_Bundles_Unsigned_Primitive_Mandates proves each unsigned helper's exact source creates both
// branches for every named witness.
func Test_Bundles_Unsigned_Primitive_Mandates(t *testing.T) {
	recorder, _, _ := registered_repository_fixture(t, "testdata/primitive_mandates")
	previous := default_invariant.Default
	default_invariant.Default = recorder
	defer func() { default_invariant.Default = previous }()
	exercise_unsigned_primitive_mandates()
	assert_primitive_mandates(t, recorder, unsigned_primitive_mandates())
}

// Test_Bundles_Floating_Primitive_Mandates proves each float helper's exact source creates both
// branches for every non-finite witness.
func Test_Bundles_Floating_Primitive_Mandates(t *testing.T) {
	recorder, _, _ := registered_repository_fixture(t, "testdata/primitive_mandates")
	previous := default_invariant.Default
	default_invariant.Default = recorder
	defer func() { default_invariant.Default = previous }()
	exercise_float_primitive_mandates()
	assert_primitive_mandates(t, recorder, floating_primitive_mandates())
}

// Test_Bundles_Boolean_Primitive_Mandate proves the Boolean helper requires both domain values.
func Test_Bundles_Boolean_Primitive_Mandate(t *testing.T) {
	recorder, _, _ := registered_repository_fixture(t, "testdata/primitive_mandates")
	previous := default_invariant.Default
	default_invariant.Default = recorder
	defer func() { default_invariant.Default = previous }()
	default_invariant.Boolean_Invariants(false, "primitive.boolean")
	default_invariant.Boolean_Invariants(true, "primitive.boolean")
	assert_primitive_mandates(t, recorder, boolean_primitive_mandates())
}

// Test_Bundles_Primitive_Isolation proves complete coverage at one primitive callsite cannot fill
// gaps at another callsite that invokes the same helper.
func Test_Bundles_Primitive_Isolation(t *testing.T) {
	table := primitive_isolation_report(t, invariant.Coverage_Gap_Table_Write)
	want_table := "🚨 4 coverage gaps 🚨\n\n" +
		"# Branch gaps (4)\n\n" +
		"| Assertion        | Link | Missing | Property" +
		"                       | Source            |\n" +
		"|------------------|-----:|---------|---------" +
		"-----------------------|-------------------|\n" +
		"| primitive.second |    0 | false   | The value is one." +
		"              | n == 1            |\n" +
		"| primitive.second |    1 | true    | The value is negative one." +
		"     | n == -1           |\n" +
		"| primitive.second |    2 | true    | The value is the minimum int8." +
		" | n == math.MinInt8 |\n" +
		"| primitive.second |    3 | true    | The value is the maximum int8." +
		" | n == math.MaxInt8 |\n\n" +
		"🚨 4 coverage gaps 🚨\n"
	if table != want_table {
		t.Fatalf("table=%q, want %q", table, want_table)
	}
	json := primitive_isolation_report(t, default_invariant.Coverage_Gap_Json_Write)
	want_json := `[{"section":"branch","assertion":"primitive.second","link":0,` +
		`"missing":"false","property":"The value is one.","source":"n == 1"},` +
		`{"section":"branch","assertion":"primitive.second","link":1,` +
		`"missing":"true","property":"The value is negative one.","source":"n == -1"},` +
		`{"section":"branch","assertion":"primitive.second","link":2,` +
		`"missing":"true","property":"The value is the minimum int8.",` +
		`"source":"n == math.MinInt8"},{"section":"branch",` +
		`"assertion":"primitive.second","link":3,"missing":"true",` +
		`"property":"The value is the maximum int8.",` +
		`"source":"n == math.MaxInt8"}]` + "\n"
	if json != want_json {
		t.Fatalf("json=%q, want %q", json, want_json)
	}
}

// Test_Analysis_Gaps keeps uncovered branches fatal.
func Test_Analysis_Gaps(t *testing.T) {
	Test_Sometimes_Gap(t)
}

// Test_Analysis_Reachability_Identity keeps a builder's internal key out of public reports.
func Test_Analysis_Reachability_Identity(t *testing.T) {
	recorder, output, _ := registered_fixture(`package fixture
func check(value int) {
	invariant.Assertions("range").Range_Int(value, 0, 4).Ensure()
}
`)
	recorder.Events.Range(func(key any, value any) (continue_iteration bool) {
		metadata := value.(*invariant.Assertion_Metadata)
		if metadata.Kind == invariant.ASSERTION_KIND_SOMETIMES {
			metadata.Frequency.Store(1)
			metadata.False_Frequency.Store(1)
		}
		return true
	})
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	want := "🚨 2 coverage gaps 🚨\n\n" +
		"# Reachability gaps (2)\n\n" +
		"| Assertion | Source |\n" +
		"|-----------|--------|\n" +
		"| range     | value  |\n" +
		"| range     | value  |\n\n" +
		"🚨 2 coverage gaps 🚨\n"
	if output.String() != want {
		t.Fatalf("output=%q, want %q", output.String(), want)
	}
}

// Test_Analysis_Table_Order keeps repeated assertion names ordered by numeric link and polarity.
func Test_Analysis_Table_Order(t *testing.T) {
	recorder, output, _ := registered_fixture(`package fixture
func alpha(v bool) { invariant.Assertions("alpha").Sometimes(v, "axis").Ensure() }
func same(a bool, b bool) {
	invariant.Assertions("same").Sometimes(a, "second").Sometimes(b, "first").Ensure()
}
`)
	invariant.Recorder_Assertions(recorder, "same").
		Sometimes(false, "second").Sometimes(false, "first").Ensure()
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	want := "🚨 4 coverage gaps 🚨\n\n" +
		"# Branch gaps (4)\n\n" +
		"| Assertion | Link | Missing | Property | Source |\n" +
		"|-----------|-----:|---------|----------|--------|\n" +
		"| alpha     |    0 | false   | axis     | v      |\n" +
		"| alpha     |    0 | true    | axis     | v      |\n" +
		"| same      |    0 | true    | second   | a      |\n" +
		"| same      |    1 | true    | first    | b      |\n\n" +
		"🚨 4 coverage gaps 🚨\n"
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

// Test_Analysis_Table_Escape keeps Markdown structure outside every dynamic cell.
func Test_Analysis_Table_Escape(t *testing.T) {
	link := uint8(2)
	property := "P|Q\\R\nS"
	gaps := []invariant.Coverage_Gap{{
		Section: "branch", Assertion: "A|B\\C\nD", Link: &link,
		Absent: "true", Property: &property, Source: "x|y\\z\nw",
	}}
	output := &bytes.Buffer{}
	if err := invariant.Coverage_Gap_Table_Write(output, gaps); err != nil {
		t.Fatal(err)
	}
	want := "🚨 1 coverage gaps 🚨\n\n" +
		"# Branch gaps (1)\n\n" +
		"| Assertion    | Link | Missing | Property     | Source       |\n" +
		"|--------------|-----:|---------|--------------|--------------|\n" +
		"| A\\|B\\\\C<br>D |    2 | true    | P\\|Q\\\\R<br>S | x\\|y\\\\z<br>w |\n\n" +
		"🚨 1 coverage gaps 🚨\n"
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

// Test_Analysis_Output_Configuration keeps an invalid output mode fatal and diagnostic.
func Test_Analysis_Output_Configuration(t *testing.T) {
	output := &bytes.Buffer{}
	code := -1
	recorder := &invariant.Recorder{
		Output: output, Exit: func(status int) { code = status }, Is_Test: true,
		Output_Configuration_Diagnostic: "INVARIANT_OUTPUT has unknown value \"dense\"; " +
			"expected \"table\" or \"json\"",
	}
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
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
const Minimum = -2
const Maximum = 3
const Hole = 0
func check(v int, condition bool) {
	invariant.Always(condition, "guard")
	invariant.Assertions("summary").
		Sometimes(condition, "axis").
		Range_Holed_Int(v, Minimum, Maximum, Hole, Hole, Hole, Hole).
		Enum_Int(v, Minimum, Maximum).
		Ensure()
}
`)
	want := "✓ fixture: tested 21 properties (21 individual, of which 5 are panic-able)"
	if summary := invariant.Recorder_Assertion_Summary(recorder); summary != want {
		t.Fatalf("summary = %q, want %q", summary, want)
	}
}

// Test_Analysis_Clean prevents complete coverage from exiting as a failure.
func Test_Analysis_Clean(t *testing.T) {
	recorder := registered_single_axis(t, "clean")
	invariant.Recorder_Assertions(recorder, "clean").Sometimes(true, "axis").Ensure()
	invariant.Recorder_Assertions(recorder, "clean").Sometimes(false, "axis").Ensure()
	code := -1
	recorder.Exit = func(status int) { code = status }
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	if code != -1 {
		t.Fatalf("clean analysis exited %d", code)
	}
}

// Test_Coverage_Modes keeps benchmarks from contaminating coverage.
func Test_Coverage_Modes(t *testing.T) {
	recorder := registered_single_axis(t, "mode")
	recorder.Is_Benchmark = true
	invariant.Recorder_Assertions(recorder, "mode").Sometimes(true, "axis").Ensure()
	if count_covered(recorder) != 0 {
		t.Fatal("benchmark recorded coverage")
	}
}

// Test_Coverage_Enforcement prevents benchmarks from disabling guards.
func Test_Coverage_Enforcement(t *testing.T) {
	recorder := &invariant.Recorder{Is_Benchmark: true}
	message := panic_text(func() {
		invariant.Recorder_Assertions(recorder, "range").Range_Int(3, 0, 2).Ensure()
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
func check(v int) {
	invariant.Assertions("holed").Range_Holed_Int(v, -4, 6, -2, 3, 3, 3).Ensure()
}
`)
	if code != -1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	invariant.Recorder_Assertions(recorder, "holed").
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
	invariant.Recorder_Assertions(recorder, "range").Range_Int(1, 0, 4).Ensure()
	lower := chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "range", Ordinal: 0, Message: "The value is at least its minimum.",
	})
	if lower.Frequency.Load() != 1 {
		t.Fatal("lower guard was not credited")
	}
	upper := chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "range", Ordinal: 1, Message: "The value is at most its maximum.",
	})
	if upper.Frequency.Load() != 1 {
		t.Fatal("upper guard was not credited")
	}
}

// Test_Range_Coverage keeps both boundaries mandatory when distinct.
func Test_Range_Coverage(t *testing.T) {
	recorder, _, _ := registered_range(t, "range", -2, 3)
	for value := -2; value <= 3; value++ {
		invariant.Recorder_Assertions(recorder, "range").Range_Int(value, -2, 3).Ensure()
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
			Namespace: "range", Ordinal: uint8(message_index + 2), Message: message,
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
	recorder := &invariant.Recorder{}
	invariant.Recorder_Assertions(recorder, "runtime").Range_Int(1, 0, 1).Ensure()
	if panic_text(invariant.Recorder_Assertions(recorder, "runtime").
		Range_Int(2, 0, 1).Ensure) == "" {
		t.Fatal("an unregistered Range stopped enforcing its bounds")
	}
}

// Test_Range_Exclusions prevents canonical padding from weakening boundary witnesses or counting
// one hole repeatedly.
func Test_Range_Exclusions(t *testing.T) {
	recorder, output, code := registered_fixture(`package fixture
func check(v int) {
	invariant.Assertions("range").Range_Holed_Int(v, -4, 6, -2, 3, 3, 3).Ensure()
}
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
		invariant.Recorder_Assertions(recorder, "range").
			Range_Holed_Int(value, -4, 6, -2, 3, 3, 3).
			Ensure()
	}
	builder := invariant.Recorder_Assertions(&invariant.Recorder{}, "range").
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
		{"Range_Holed_Int(v, 0, 5, 0, 1, 2, 3)", "strictly inside"},
		{"Range_Holed_Int(v, 0, 5, 1, 2, 3, 5)", "strictly inside"},
		{"Range_Holed_Int(v, 0, 6, 1, 1, 2, 2)", "final-hole padding"},
		{"Range_Holed_Int(v, 0, 6, 2, 1, 2, 2)", "ascending"},
		{"Range_Holed_Int(v, 0, 6, 1, 2, 3)", "exactly four hole slots"},
		{"Range_Holed_Uint(v, 0, 6, 1, 2, 3, 3)", "exactly three hole slots"},
	}
	for _, fixture := range fixtures {
		source := "package fixture\nfunc check(v int) { invariant.Assertions(\"range\")." +
			fixture.Call + ".Ensure() }\n"
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
	recorder, _, _ := registered_fixture(`package fixture
func check(v int) { invariant.Assertions("enum").Enum_Int(v, 1, 2).Ensure() }
`)
	invariant.Recorder_Assertions(recorder, "enum").Enum_Int(1, 1, 2).Ensure()
	guard := chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "enum", Ordinal: 0, Message: "The value is an enum member.",
	})
	if guard.Frequency.Load() != 1 {
		t.Fatal("membership guard was not credited")
	}
}

// Test_Enum_Members keeps every canonical member mandatory and ordered by value.
func Test_Enum_Members(t *testing.T) {
	recorder, _, _ := registered_fixture(`package fixture
func pair(v int) { invariant.Assertions("enum.2").Enum_Int(v, -3, 9).Ensure() }
func triple(v int) { invariant.Assertions("enum.3").Enum_3_Int(v, -3, 2, 9).Ensure() }
func quartet(v int) { invariant.Assertions("enum.4").Enum_4_Int(v, -3, 0, 2, 9).Ensure() }
`)
	for _, value := range []int{-3, 9} {
		invariant.Recorder_Assertions(recorder, "enum.2").Enum_Int(value, -3, 9).Ensure()
	}
	for _, value := range []int{-3, 2, 9} {
		invariant.Recorder_Assertions(recorder, "enum.3").
			Enum_3_Int(value, -3, 2, 9).Ensure()
	}
	for _, value := range []int{-3, 0, 2, 9} {
		invariant.Recorder_Assertions(recorder, "enum.4").
			Enum_4_Int(value, -3, 0, 2, 9).Ensure()
	}
	assert_enum_members(t, recorder, "enum.2", []string{"-3", "9"})
	assert_enum_members(t, recorder, "enum.3", []string{"-3", "2", "9"})
	assert_enum_members(t, recorder, "enum.4", []string{"-3", "0", "2", "9"})
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
		{"Enum_4_Int(v, 1, 2, 3)", "exactly four members"},
		{"Enum_3_Int(v, 1, 1, 2)", "exactly distinct"},
		{"Enum_4_Int(v, 1, 3, 2, 4)", "ascending"},
	}
	for _, fixture := range fixtures {
		source := "package fixture\nfunc check(v int) { invariant.Assertions(\"enum\")." +
			fixture.Call + ".Ensure() }\n"
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
	source := "package fixture\nfunc check(v int) { invariant.Assertions(\"range\")." +
		call + ".Ensure() }\n"
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
	source := fmt.Sprintf(
		"package fixture\nfunc check(v %s) { "+
			"invariant.Assertions(\"range\").%s.Ensure() }\n",
		value_type, call)
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
	key := assertion_key("persist", 0, "axis")
	invariant.Recorder_Merge_Fuzz_Coverage_From(recorder, strings.NewReader(record))
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
	t *testing.T, recorder *invariant.Recorder, namespace invariant.Namespace, members []string,
) {
	t.Helper()
	guard := chain_metadata(t, recorder, chain_metadata_key{
		Namespace: namespace, Ordinal: 0, Message: "The value is an enum member.",
	})
	if guard.Frequency.Load() == 0 {
		t.Fatalf("%s membership guard was not witnessed", namespace)
	}
	for member_index, member := range members {
		message := "The value equals member " + member + "."
		metadata := chain_metadata(t, recorder, chain_metadata_key{
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

type primitive_mandate struct {
	Namespace  invariant.Namespace
	Messages   []string
	Conditions []string
}

func assert_primitive_mandates(
	t *testing.T, recorder *invariant.Recorder, mandates []primitive_mandate,
) {
	t.Helper()
	for _, mandate := range mandates {
		for message_index, message := range mandate.Messages {
			metadata := chain_metadata(t, recorder, chain_metadata_key{
				Namespace: mandate.Namespace,
				Ordinal:   uint8(message_index),
				Message:   message,
			})
			if metadata.Condition != mandate.Conditions[message_index] {
				t.Fatalf("%s %q condition=%q, want %q", mandate.Namespace, message,
					metadata.Condition, mandate.Conditions[message_index])
			}
			if metadata.Frequency.Load() == 0 {
				t.Fatalf("%s %q true branch was not witnessed",
					mandate.Namespace, message)
			}
			if metadata.False_Frequency.Load() == 0 {
				t.Fatalf("%s %q false branch was not witnessed",
					mandate.Namespace, message)
			}
		}
	}
}

func signed_primitive_mandates() (mandates []primitive_mandate) {
	return []primitive_mandate{
		{"primitive.int", []string{"The value is one.", "The value is negative one.",
			"The value is the minimum int.", "The value is the maximum int."},
			[]string{"n == 1", "n == -1", "n == math.MinInt64", "n == math.MaxInt64"}},
		{"primitive.int8", []string{"The value is one.", "The value is negative one.",
			"The value is the minimum int8.", "The value is the maximum int8."},
			[]string{"n == 1", "n == -1", "n == math.MinInt8", "n == math.MaxInt8"}},
		{"primitive.int16", []string{"The value is one.", "The value is negative one.",
			"The value is the minimum int16.", "The value is the maximum int16."},
			[]string{"n == 1", "n == -1", "n == math.MinInt16", "n == math.MaxInt16"}},
		{"primitive.int32", []string{"The value is one.", "The value is negative one.",
			"The value is the minimum int32.", "The value is the maximum int32."},
			[]string{"n == 1", "n == -1", "n == math.MinInt32", "n == math.MaxInt32"}},
		{"primitive.int64", []string{"The value is one.", "The value is negative one.",
			"The value is the minimum int64.", "The value is the maximum int64."},
			[]string{"n == 1", "n == -1", "n == math.MinInt64", "n == math.MaxInt64"}},
	}
}

func unsigned_primitive_mandates() (mandates []primitive_mandate) {
	return []primitive_mandate{
		{"primitive.uint", []string{"The value is zero.", "The value is one.",
			"The value is the maximum uint."},
			[]string{"n == 0", "n == 1", "n == math.MaxUint64"}},
		{"primitive.uint8", []string{"The value is zero.", "The value is one.",
			"The value is the maximum uint8."},
			[]string{"n == 0", "n == 1", "n == math.MaxUint8"}},
		{"primitive.uint16", []string{"The value is zero.", "The value is one.",
			"The value is the maximum uint16."},
			[]string{"n == 0", "n == 1", "n == math.MaxUint16"}},
		{"primitive.uint32", []string{"The value is zero.", "The value is one.",
			"The value is the maximum uint32."},
			[]string{"n == 0", "n == 1", "n == math.MaxUint32"}},
		{"primitive.uint64", []string{"The value is zero.", "The value is one.",
			"The value is the maximum uint64."},
			[]string{"n == 0", "n == 1", "n == math.MaxUint64"}},
	}
}

func floating_primitive_mandates() (mandates []primitive_mandate) {
	return []primitive_mandate{
		{"primitive.float32", []string{"The value is NaN.",
			"The value is negative infinity.", "The value is positive infinity."},
			[]string{"math.IsNaN(float64(f))", "float64(f) == math.Inf(-1)",
				"float64(f) == math.Inf(1)"}},
		{"primitive.float64", []string{"The value is NaN.",
			"The value is negative infinity.", "The value is positive infinity."},
			[]string{"math.IsNaN(f)", "f == math.Inf(-1)", "f == math.Inf(1)"}},
	}
}

func boolean_primitive_mandates() (mandates []primitive_mandate) {
	return []primitive_mandate{
		{"primitive.boolean", []string{"The value is true."}, []string{"b"}},
	}
}

func exercise_signed_primitive_mandates() {
	for _, value := range []int{0, 1, -1, math.MinInt64, math.MaxInt64} {
		default_invariant.Int_Invariants(value, "primitive.int")
	}
	for _, value := range []int8{0, 1, -1, math.MinInt8, math.MaxInt8} {
		default_invariant.Int8_Invariants(value, "primitive.int8")
	}
	for _, value := range []int16{0, 1, -1, math.MinInt16, math.MaxInt16} {
		default_invariant.Int16_Invariants(value, "primitive.int16")
	}
	for _, value := range []int32{0, 1, -1, math.MinInt32, math.MaxInt32} {
		default_invariant.Int32_Invariants(value, "primitive.int32")
	}
	for _, value := range []int64{0, 1, -1, math.MinInt64, math.MaxInt64} {
		default_invariant.Int64_Invariants(value, "primitive.int64")
	}
}

func exercise_unsigned_primitive_mandates() {
	for _, value := range []uint{0, 1, 2, math.MaxUint64} {
		default_invariant.Uint_Invariants(value, "primitive.uint")
	}
	for _, value := range []uint8{0, 1, 2, math.MaxUint8} {
		default_invariant.Uint8_Invariants(value, "primitive.uint8")
	}
	for _, value := range []uint16{0, 1, 2, math.MaxUint16} {
		default_invariant.Uint16_Invariants(value, "primitive.uint16")
	}
	for _, value := range []uint32{0, 1, 2, math.MaxUint32} {
		default_invariant.Uint32_Invariants(value, "primitive.uint32")
	}
	for _, value := range []uint64{0, 1, 2, math.MaxUint64} {
		default_invariant.Uint64_Invariants(value, "primitive.uint64")
	}
}

func exercise_float_primitive_mandates() {
	for _, value := range []float32{0, float32(math.NaN()),
		float32(math.Inf(-1)), float32(math.Inf(1))} {
		default_invariant.Float32_Invariants(value, "primitive.float32")
	}
	for _, value := range []float64{0, math.NaN(), math.Inf(-1), math.Inf(1)} {
		default_invariant.Float64_Invariants(value, "primitive.float64")
	}
}

func primitive_isolation_report(
	t *testing.T, reporter func(io.Writer, []invariant.Coverage_Gap) (err error),
) (report string) {
	t.Helper()
	recorder, output, code := registered_repository_fixture(t, "testdata/primitive_isolation")
	recorder.Report_Coverage_Gaps = reporter
	previous := default_invariant.Default
	default_invariant.Default = recorder
	defer func() { default_invariant.Default = previous }()
	for _, value := range []int8{0, 1, -1, math.MinInt8, math.MaxInt8} {
		default_invariant.Int8_Invariants(value, "primitive.first")
	}
	default_invariant.Int8_Invariants(1, "primitive.second")
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	if *code != 1 {
		t.Fatalf("exit=%d output=%q", *code, output.String())
	}
	return output.String()
}

func registered_repository_fixture(
	t *testing.T, directory string,
) (recorder *invariant.Recorder, output *bytes.Buffer, code *int) {
	t.Helper()
	working_directory, working_error := os.Getwd()
	if working_error != nil {
		t.Fatal(working_error)
	}
	output = &bytes.Buffer{}
	status := -1
	recorder = &invariant.Recorder{
		File_System: os.DirFS("/"), Working_Directory: working_directory,
		Packages_To_Analyze: []string{directory}, Output: output,
		Exit: func(exit_status int) { status = exit_status }, Is_Test: true,
		Sugar_Package: reflect.TypeOf(default_invariant.Sugar_Package_Marker{}).PkgPath(),
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	if status != -1 {
		t.Fatalf("registration exit=%d output=%q", status, output.String())
	}
	return recorder, output, &status
}

func registered_fixture(source string) (
	recorder *invariant.Recorder, output *bytes.Buffer, code int,
) {
	return registered_fixture_options(source, "")
}

func registered_global_namespace_fixture() (
	recorder *invariant.Recorder, output *bytes.Buffer, code int,
) {
	output = &bytes.Buffer{}
	code = -1
	recorder = &invariant.Recorder{
		File_System: fstest.MapFS{
			"go.mod": &fstest.MapFile{Data: []byte("module fixture\n")},
			"a/a.go": &fstest.MapFile{Data: []byte(`package a
func check(value bool) { invariant.Assertions("global").Sometimes(value, "a").Ensure() }
`)},
			"b/b.go": &fstest.MapFile{Data: []byte(`package b
func check(value bool) { invariant.Assertions("global").Sometimes(value, "b").Ensure() }
`)},
		},
		Packages_To_Analyze: []string{"/a", "/b"}, Output: output,
		Exit: func(status int) { code = status }, Is_Test: true,
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	return recorder, output, code
}

func registered_fixture_with_sugar(source string) (
	recorder *invariant.Recorder, output *bytes.Buffer, code int,
) {
	return registered_fixture_options(source, "fixture/fixture")
}

func registered_fixture_with_test(source string, test_source string) (
	recorder *invariant.Recorder, output *bytes.Buffer, code int,
) {
	output = &bytes.Buffer{}
	code = -1
	recorder = &invariant.Recorder{
		File_System: fstest.MapFS{
			"go.mod":                &fstest.MapFile{Data: []byte("module fixture\n")},
			"fixture/check.go":      &fstest.MapFile{Data: []byte(source)},
			"fixture/check_test.go": &fstest.MapFile{Data: []byte(test_source)},
		},
		Packages_To_Analyze: []string{"/fixture"}, Output: output,
		Exit: func(status int) { code = status }, Is_Test: true,
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	return recorder, output, code
}

func registered_fixture_options(source string, sugar string) (
	recorder *invariant.Recorder, output *bytes.Buffer, code int,
) {
	output = &bytes.Buffer{}
	code = -1
	recorder = &invariant.Recorder{
		File_System: fstest.MapFS{
			"go.mod":           &fstest.MapFile{Data: []byte("module fixture\n")},
			"fixture/check.go": &fstest.MapFile{Data: []byte(source)},
		},
		Packages_To_Analyze: []string{"/fixture"}, Output: output,
		Exit: func(status int) { code = status }, Is_Test: true,
		Sugar_Package: sugar,
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	return recorder, output, code
}

type chain_metadata_key struct {
	Namespace invariant.Namespace
	Ordinal   uint8
	Message   string
}

func assertion_key(namespace invariant.Namespace, ordinal uint8, message string) (key string) {
	return string(namespace) + invariant.ELEMENT_MESSAGE_SEPARATOR + fmt.Sprint(ordinal) +
		invariant.ELEMENT_MESSAGE_SEPARATOR + message
}

func chain_metadata(
	t *testing.T, recorder *invariant.Recorder, key chain_metadata_key,
) (metadata *invariant.Assertion_Metadata) {
	t.Helper()
	return recorder_event(t, recorder, assertion_key(key.Namespace, key.Ordinal, key.Message))
}

func recorder_event(
	t *testing.T, recorder *invariant.Recorder, key string,
) (metadata *invariant.Assertion_Metadata) {
	t.Helper()
	value, exists := recorder.Events.Load(key)
	if !exists {
		t.Fatalf("missing event %q", key)
	}
	return value.(*invariant.Assertion_Metadata)
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

func registered_single_axis(t *testing.T, namespace string) (recorder *invariant.Recorder) {
	t.Helper()
	source := fmt.Sprintf(
		"package fixture\n"+
			"func check(v bool) { invariant.Assertions(%q)."+
			"Sometimes(v, \"axis\").Ensure() }\n",
		namespace)
	recorder, output, code := registered_fixture(source)
	if code != -1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	return recorder
}

func registered_range(
	t *testing.T, namespace string, minimum int, maximum int,
) (recorder *invariant.Recorder, output *bytes.Buffer, code int) {
	t.Helper()
	source := fmt.Sprintf(
		"package fixture\n"+
			"func check(v int) { invariant.Assertions(%q)."+
			"Range_Int(v, %d, %d).Ensure() }\n",
		namespace, minimum, maximum)
	return registered_fixture(source)
}

func registered_nested_bundle(t *testing.T) (recorder *invariant.Recorder) {
	t.Helper()
	recorder, output, code := registered_fixture(`package fixture
type Inner int
func Inner_Invariants(value Inner, namespace invariant.Namespace) {
	invariant.Assertions(namespace).Sometimes(value == 0, "zero").Ensure()
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
