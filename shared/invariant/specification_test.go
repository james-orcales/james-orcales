//go:build !invariant_disable_coverage && !prd && !prod && !production && !invariant_noop

package invariant_test

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"local/james-orcales/shared/invariant"
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
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "identity", Ordinal: 0, Message: "same",
	})
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "identity", Ordinal: 1, Message: "same",
	})
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
	recorder := registered_single_axis(t, "persist")
	key := assertion_key("persist", 0, "axis")
	invariant.Recorder_Merge_Fuzz_Coverage_From(
		recorder, strings.NewReader(invariant.Fuzz_Coverage_Line(key, false)))
	if recorder_event(t, recorder, key).False_Frequency.Load() != 1 {
		t.Fatal("persisted false branch did not merge")
	}
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

// Test_Assertions_Registration_Transitive keeps reached bundles unconditional across composition.
func Test_Assertions_Registration_Transitive(t *testing.T) {
	recorder := registered_nested_bundle(t)
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "outer.inner", Ordinal: 0, Message: "zero",
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
	if code != -1 {
		t.Fatalf("reuse exit=%d output=%q", code, output.String())
	}
	if len(recorder.Assertion_Plans) != 1 {
		t.Fatalf("plans=%d, want one idempotent source root", len(recorder.Assertion_Plans))
	}
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "number", Ordinal: 0, Message: "zero",
	})
	_, output, code = registered_fixture(`package fixture
func first(v bool) { invariant.Assertions("same").Sometimes(v, "a").Ensure() }
func second(v bool) { invariant.Assertions("same").Sometimes(v, "b").Ensure() }
`)
	if code != 1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "duplicate namespace") {
		t.Fatalf("exit=%d output=%q", code, output.String())
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

// Test_Bundles_Template maps the legacy heading to the fluent template contract.
func Test_Bundles_Template(t *testing.T) {
	Test_Assertions_Registration_Template(t)
}

// Test_Bundles_Descent protects nested template expansion.
func Test_Bundles_Descent(t *testing.T) {
	recorder := registered_nested_bundle(t)
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "outer.inner", Ordinal: 0, Message: "zero",
	})
}

// Test_Bundles_Composition keeps parent and child obligations independent.
func Test_Bundles_Composition(t *testing.T) {
	recorder := registered_nested_bundle(t)
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "outer", Ordinal: 0, Message: "positive",
	})
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "outer.inner", Ordinal: 0, Message: "zero",
	})
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
	recorder, _, _ := registered_fixture(`package fixture
type Number int
func Number_Invariants(value Number, namespace invariant.Namespace) {
	invariant.Assertions(namespace).Sometimes(value == 0, "zero").Ensure()
}
func first(value Number) { Number_Invariants(value, "first") }
func second(value Number) { Number_Invariants(value, "second") }
`)
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "first", Ordinal: 0, Message: "zero",
	})
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "second", Ordinal: 0, Message: "zero",
	})
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

// Test_Analysis_Gaps keeps uncovered branches fatal.
func Test_Analysis_Gaps(t *testing.T) {
	Test_Sometimes_Gap(t)
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
	invariant.Always(true, "guard")
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
	recorder, _, _ := registered_range(t, "range", 0, 4)
	for value := 0; value <= 4; value++ {
		invariant.Recorder_Assertions(recorder, "range").Range_Int(value, 0, 4).Ensure()
	}
	minimum := chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "range", Ordinal: 2, Message: "The value equals the minimum.",
	})
	maximum := chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "range", Ordinal: 3, Message: "The value equals the maximum.",
	})
	if minimum.Frequency.Load() == 0 {
		t.Fatal("range minimum true branch was not witnessed")
	}
	if minimum.False_Frequency.Load() == 0 {
		t.Fatal("range minimum false branch was not witnessed")
	}
	if maximum.Frequency.Load() == 0 {
		t.Fatal("range maximum true branch was not witnessed")
	}
	if maximum.False_Frequency.Load() == 0 {
		t.Fatal("range maximum false branch was not witnessed")
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
func check(v int) { invariant.Assertions("enum").Enum_4_Int(v, -3, 0, 2, 9).Ensure() }
`)
	if event_count(&recorder.Events) != 5 {
		t.Fatalf("events = %d, want guard plus four members", event_count(&recorder.Events))
	}
	invariant.Recorder_Assertions(recorder, "enum").Enum_4_Int(2, -3, 0, 2, 9).Ensure()
	member := chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "enum", Ordinal: 3, Message: "The value equals member 2.",
	})
	if member.Frequency.Load() != 1 {
		t.Fatal("member branch was not credited")
	}
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

func registered_fixture(source string) (
	recorder *invariant.Recorder, output *bytes.Buffer, code int,
) {
	return registered_fixture_options(source, "")
}

func registered_fixture_with_sugar(source string) (
	recorder *invariant.Recorder, output *bytes.Buffer, code int,
) {
	return registered_fixture_options(source, "fixture/fixture")
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
	invariant.Assertions(namespace).Sometimes(value.Inner > 0, "positive").Ensure()
	Inner_Invariants(value.Inner, "outer.inner")
}
func check(value Outer) { Outer_Invariants(value, "outer") }
`)
	if code != -1 {
		t.Fatalf("exit=%d output=%q", code, output.String())
	}
	return recorder
}
