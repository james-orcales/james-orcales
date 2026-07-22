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

// Test_Always_Violation prevents its specification contract from regressing.
func Test_Always_Violation(t *testing.T) {
	recorder := &invariant.Recorder{}
	message := panic_text(func() { invariant.Recorder_Always(recorder, false, "guard") })
	if !strings.Contains(message, "guard") {
		t.Fatalf("panic = %q, want guard", message)
	}
}

// Test_Always_Eager prevents its specification contract from regressing.
func Test_Always_Eager(t *testing.T) {
	recorder := &invariant.Recorder{}
	if panic_text(func() { invariant.Recorder_Always(recorder, false, "eager") }) == "" {
		t.Fatal("Always must panic at its own call")
	}
}

// Test_Always_Reachability prevents its specification contract from regressing.
func Test_Always_Reachability(t *testing.T) {
	recorder, output, _ := registered_fixture(`package fixture
func check(ok bool) { invariant.Always(ok, "reachable") }
`)
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	if !strings.Contains(output.String(), "reachable") {
		t.Fatalf("gap report = %q, want reachable", output.String())
	}
}

// Test_Sometimes_Coverage prevents its specification contract from regressing.
func Test_Sometimes_Coverage(t *testing.T) {
	recorder, _, _ := registered_chain_fixture()
	product := invariant.Recorder_Dot_Product(recorder, "check").
		Sometimes(true, "zero").
		Sometimes(false, "one").
		Impossible("exclusive", invariant.Event_True("zero"), invariant.Event_True("one"))
	metadata := chain_metadata(&chain_metadata_input{
		Test: t, Recorder: recorder,
		Key: invariant.Chain_Key{Namespace: "check", Ordinal: 0, Message: "zero"},
	})
	if metadata.Frequency.Load() != 0 {
		t.Fatal("Sometimes recorded before Ensure")
	}
	product.Ensure()
	if metadata.Frequency.Load() != 1 {
		t.Fatalf("true frequency = %d, want 1", metadata.Frequency.Load())
	}
}

// Test_Sometimes_Gap prevents its specification contract from regressing.
func Test_Sometimes_Gap(t *testing.T) {
	recorder, output, _ := registered_chain_fixture()
	invariant.Recorder_Dot_Product(recorder, "check").
		Sometimes(true, "zero").
		Sometimes(false, "one").
		Impossible("exclusive", invariant.Event_True("zero"), invariant.Event_True("one")).
		Ensure()
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	if !strings.Contains(output.String(), "zero") {
		t.Fatalf("gap report = %q, want zero", output.String())
	}
}

// Test_Dot_Product_Links prevents its specification contract from regressing.
func Test_Dot_Product_Links(t *testing.T) {
	recorder := &invariant.Recorder{}
	invariant.Recorder_Dot_Product(recorder, "check").
		Sometimes(true, "present").
		Impossible("present cannot be false", invariant.Event_False("present")).
		Ensure()
}

// Test_Dot_Product_Identity prevents its specification contract from regressing.
func Test_Dot_Product_Identity(t *testing.T) {
	const SOURCE = `package fixture
func check(a bool, b bool) {
	invariant.Dot_Product("check").Sometimes(a, "same").Sometimes(b, "same").Ensure()
}
`
	recorder, _, _ := registered_fixture(SOURCE)
	chain_metadata(&chain_metadata_input{Test: t, Recorder: recorder,
		Key: invariant.Chain_Key{Namespace: "check", Ordinal: 0, Message: "same"}})
	chain_metadata(&chain_metadata_input{Test: t, Recorder: recorder,
		Key: invariant.Chain_Key{Namespace: "check", Ordinal: 1, Message: "same"}})
}

// Test_Dot_Product_Constraint prevents its specification contract from regressing.
func Test_Dot_Product_Constraint(t *testing.T) {
	recorder := &invariant.Recorder{}
	message := panic_text(func() {
		invariant.Recorder_Dot_Product(recorder, "check").
			Sometimes(false, "empty").
			Impossible("empty is required", invariant.Event_False("empty")).
			Ensure()
	})
	if !strings.Contains(message, "empty is required") {
		t.Fatalf("panic = %q, want rule message", message)
	}
	recorder, _, _ = registered_chain_fixture()
	axis := chain_metadata(&chain_metadata_input{
		Test: t, Recorder: recorder,
		Key: invariant.Chain_Key{Namespace: "check", Ordinal: 0, Message: "zero"},
	})
	message = panic_text(func() {
		invariant.Recorder_Dot_Product(recorder, "check").
			Sometimes(true, "zero").Sometimes(true, "one").
			Impossible("exclusive",
				invariant.Event_True("zero"), invariant.Event_True("one")).
			Ensure()
	})
	if !strings.Contains(message, "exclusive") {
		t.Fatalf("registered carve panic = %q", message)
	}
	if axis.Frequency.Load() != 0 {
		t.Fatal("a rejected Ensure partially credited an axis")
	}
}

// Test_Dot_Product_Sibling prevents its specification contract from regressing.
func Test_Dot_Product_Sibling(t *testing.T) {
	recorder := &invariant.Recorder{}
	var product invariant.Product
	message := panic_text(func() {
		product = invariant.Recorder_Dot_Product(recorder, "check").
			Sometimes(true, "present").
			Impossible("typo", invariant.Event_True("missing"))
	})
	if message != "" {
		t.Fatalf("Impossible panicked before Ensure: %q", message)
	}
	first := panic_text(product.Ensure)
	second := panic_text(func() {
		invariant.Recorder_Dot_Product(recorder, "check").
			Sometimes(true, "present").
			Impossible("typo", invariant.Event_True("missing")).
			Ensure()
	})
	if first == "" {
		t.Fatal("a non-sibling reference must panic")
	}
	if second != first {
		t.Fatalf("panics differ: %q != %q", first, second)
	}
	registered, _, _ := registered_chain_fixture()
	message = panic_text(func() {
		product = invariant.Recorder_Dot_Product(registered, "check").
			Sometimes(true, "zero").Sometimes(false, "one").
			Sometimes(true, "extra").
			Impossible("extra rule", invariant.Event_True("extra"))
	})
	if message != "" {
		t.Fatalf("malformed shape panicked before Ensure: %q", message)
	}
	if panic_text(product.Ensure) == "" {
		t.Fatal("malformed shape must panic at Ensure")
	}
}

// Test_Dot_Product_Shared prevents its specification contract from regressing.
func Test_Dot_Product_Shared(t *testing.T) {
	recorder := &invariant.Recorder{}
	invariant.Recorder_Dot_Product(recorder, "shared").Sometimes(true, "a").Ensure()
	message := panic_text(func() {
		invariant.Recorder_Dot_Product(recorder, "shared").Sometimes(true, "b").Ensure()
	})
	if !strings.Contains(message, "shape differs") {
		t.Fatalf("panic = %q, want shape mismatch", message)
	}
	invariant.Recorder_Dot_Product(recorder, "deferred").
		Sometimes(true, "a").Sometimes(true, "b").Ensure()
	message = panic_text(func() {
		invariant.Recorder_Dot_Product(recorder, "deferred").
			Sometimes(true, "different").Sometimes(true, "b").Ensure()
	})
	if !strings.Contains(message, "shape differs") {
		t.Fatalf("later matching link erased mismatch: %q", message)
	}
}

// Test_Dot_Product_Unknown prevents its specification contract from regressing.
func Test_Dot_Product_Unknown(t *testing.T) {
	recorder, _, _ := registered_chain_fixture()
	var product invariant.Product
	message := panic_text(func() {
		product = invariant.Recorder_Dot_Product(recorder, "check").
			Sometimes(true, "unknown")
	})
	if message != "" {
		t.Fatalf("Sometimes panicked before Ensure: %q", message)
	}
	message = panic_text(product.Ensure)
	if !strings.Contains(message, "unknown axis") {
		t.Fatalf("panic = %q, want unknown axis", message)
	}
	shape := recorder.Chain_Shapes["check"]
	shape.Tuples[invariant.Chain_Mask{1}] = invariant.Handle_Entry{}
	message = panic_text(func() {
		invariant.Recorder_Dot_Product(recorder, "check").
			Sometimes(true, "zero").Sometimes(false, "one").
			Impossible("exclusive",
				invariant.Event_True("zero"), invariant.Event_True("one")).
			Ensure()
	})
	if !strings.Contains(message, "unknown tuple") {
		t.Fatalf("panic = %q, want unknown tuple", message)
	}
}

// Test_Dot_Product_Allocation prevents its specification contract from regressing.
func Test_Dot_Product_Allocation(t *testing.T) {
	recorder, _, _ := registered_chain_fixture()
	call := func() {
		invariant.Recorder_Dot_Product(recorder, "check").
			Sometimes(true, "zero").
			Sometimes(false, "one").
			Impossible("exclusive",
				invariant.Event_True("zero"), invariant.Event_True("one")).
			Ensure()
	}
	call()
	if allocations := testing.AllocsPerRun(100, call); allocations != 0 {
		t.Fatalf("allocations = %v, want 0", allocations)
	}
}

// Test_Dot_Product_Persistence prevents its specification contract from regressing.
func Test_Dot_Product_Persistence(t *testing.T) {
	recorder, _, _ := registered_chain_fixture()
	var lines []string
	recorder.Coverage_Sink = func(key string, fired_true bool) {
		lines = append(lines, invariant.Fuzz_Coverage_Line(key, fired_true))
	}
	invariant.Recorder_Dot_Product(recorder, "check").
		Sometimes(true, "zero").
		Sometimes(false, "one").
		Impossible("exclusive", invariant.Event_True("zero"), invariant.Event_True("one")).
		Ensure()
	if len(lines) != 3 {
		t.Fatalf("first-coverage lines = %d, want 3", len(lines))
	}
	key := "check" + invariant.ELEMENT_MESSAGE_SEPARATOR + "0" +
		invariant.ELEMENT_MESSAGE_SEPARATOR + "zero"
	axis := chain_metadata(&chain_metadata_input{
		Test: t, Recorder: recorder,
		Key: invariant.Chain_Key{Namespace: "check", Ordinal: 0, Message: "zero"},
	})
	axis.Frequency.Store(0)
	recorder.Events.Delete(key)
	line := invariant.Fuzz_Coverage_Line(key, true)
	invariant.Recorder_Merge_Fuzz_Coverage_From(recorder, strings.NewReader(line))
	if axis.Frequency.Load() != 1 {
		t.Fatal("a serialized chain axis must merge through its structural key")
	}
}

// Test_Dot_Product_Registration_Walk prevents its specification contract from regressing.
func Test_Dot_Product_Registration_Walk(t *testing.T) {
	recorder, _, _ := registered_chain_fixture()
	if event_count(&recorder.Events) != 5 {
		count := event_count(&recorder.Events)
		t.Fatalf("events = %d, want two axes plus three tuples", count)
	}
	if _, exists := recorder.Events.Load("check:tuple=(1,1)"); exists {
		t.Fatal("the carved cell must not survive registration")
	}
	const GLOB_SOURCE = `package fixture
func check(a bool, b bool, c bool) {
	invariant.Dot_Product("glob").Sometimes(a, "a").Sometimes(b, "b").Sometimes(c, "c").
		Impossible("a and not b", invariant.Event_True("a"), invariant.Event_False("b")).Ensure()
}
`
	recorder, _, _ = registered_fixture(GLOB_SOURCE)
	for _, key := range []string{"glob:tuple=(1,0,0)", "glob:tuple=(1,0,1)"} {
		if _, exists := recorder.Events.Load(key); exists {
			t.Fatalf("unnamed c axis must be globbed from %s", key)
		}
	}
	if _, exists := recorder.Events.Load("glob:tuple=(1,1,0)"); !exists {
		t.Fatal("opposite b polarity must remain demanded")
	}
}

// Test_Dot_Product_Registration_Template prevents its specification contract from regressing.
func Test_Dot_Product_Registration_Template(t *testing.T) {
	const SOURCE = `package fixture
func Number_Invariants(n int, namespace invariant.Namespace) {
	invariant.Dot_Product(namespace).Sometimes(n == 0, "zero").Ensure()
}
func Number_Product(n int, namespace invariant.Namespace) invariant.Product {
	return invariant.Dot_Product(namespace).Sometimes(n < 0, "negative")
}
func check(n int) {
	Number_Invariants(n, "number")
	Number_Product(n, "prefix").Sometimes(n > 0, "positive").Ensure()
}
`
	recorder, _, _ := registered_fixture(SOURCE)
	chain_metadata(&chain_metadata_input{Test: t, Recorder: recorder,
		Key: invariant.Chain_Key{Namespace: "number", Ordinal: 0, Message: "zero"}})
	chain_metadata(&chain_metadata_input{Test: t, Recorder: recorder,
		Key: invariant.Chain_Key{Namespace: "prefix", Ordinal: 0, Message: "negative"}})
	chain_metadata(&chain_metadata_input{Test: t, Recorder: recorder,
		Key: invariant.Chain_Key{Namespace: "prefix", Ordinal: 1, Message: "positive"}})
}

// Test_Dot_Product_Registration_Ensured prevents its specification contract from regressing.
func Test_Dot_Product_Registration_Ensured(t *testing.T) {
	const SOURCE = `package fixture
func check(n int) { invariant.Dot_Product("check").Sometimes(n == 0, "zero") }
`
	_, output, code := registered_fixture(SOURCE)
	if code != 1 {
		t.Fatalf("exit = %d, want 1; output=%q", code, output.String())
	}
	const SUGAR_SOURCE = `package invariant
func check(n int) { Dot_Product("check").Sometimes(n == 0, "zero") }
`
	_, output, code = registered_fixture_with_sugar(SUGAR_SOURCE)
	if code != 1 {
		t.Fatalf("bare sugar exit = %d, want 1; output=%q", code, output.String())
	}
}

// Test_Dot_Product_Registration_Depth prevents its specification contract from regressing.
func Test_Dot_Product_Registration_Depth(t *testing.T) {
	const SOURCE = `package fixture
func Prefix(n int, namespace invariant.Namespace) invariant.Product {
	return invariant.Dot_Product(namespace).Sometimes(n == 0, "zero")
}
func Wrapped(n int, namespace invariant.Namespace) invariant.Product {
	return Prefix(n, namespace).Sometimes(n == 1, "one")
}
func check(n int) { Wrapped(n, "number").Ensure() }
`
	_, output, code := registered_fixture(SOURCE)
	if code != 1 {
		t.Fatalf("exit = %d, want 1; output=%q", code, output.String())
	}
}

// Test_Dot_Product_Registration_Reference prevents its specification contract from regressing.
func Test_Dot_Product_Registration_Reference(t *testing.T) {
	const SOURCE = `package fixture
func check(n int) {
	invariant.Dot_Product("check").Sometimes(n == 0, "same").Sometimes(n == 1, "same").
		Impossible("ambiguous", invariant.Event_True("same")).Ensure()
}
`
	_, output, code := registered_fixture(SOURCE)
	if code != 1 {
		t.Fatalf("exit = %d, want 1; output=%q", code, output.String())
	}
	const NUL_NAMESPACE_SOURCE = `package fixture
func check(n int) {
	invariant.Dot_Product("bad\x00namespace").Sometimes(n == 0, "zero").Ensure()
}
`
	_, output, code = registered_fixture(NUL_NAMESPACE_SOURCE)
	if code != 1 {
		t.Fatalf("NUL namespace exit = %d, want 1; output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "NUL") {
		t.Fatalf("NUL namespace diagnostic = %q", output.String())
	}
}

// Test_Dot_Product_Registration_Caps prevents its specification contract from regressing.
func Test_Dot_Product_Registration_Caps(t *testing.T) {
	var source strings.Builder
	source.WriteString("package fixture\nfunc check(v bool) { invariant.Dot_Product(\"check\")")
	for i_index := 0; i_index < 13; i_index++ {
		fmt.Fprintf(&source, ".Sometimes(v, %q)", fmt.Sprintf("axis %d", i_index))
	}
	source.WriteString(".Ensure() }\n")
	recorder, output, code := registered_fixture(source.String())
	if code != -1 {
		t.Fatalf("13-axis exit = %d, want no exit; output=%q", code, output.String())
	}
	if count := event_count(&recorder.Events); count != 13+(1<<13) {
		t.Fatalf("13-axis event count = %d, want %d", count, 13+(1<<13))
	}
	registered_product := invariant.Recorder_Dot_Product(recorder, "check")
	for i_index := 0; i_index < 13; i_index++ {
		message := fmt.Sprintf("axis %d", i_index)
		registered_product = registered_product.Sometimes(false, message)
	}
	if message := panic_text(registered_product.Ensure); message != "" {
		t.Fatalf("13-axis registered Ensure panic = %q", message)
	}
	source.Reset()
	source.WriteString("package fixture\nfunc check(v bool) { invariant.Dot_Product(\"links\")")
	source.WriteString(".Sometimes(v, \"axis\")")
	for i_index := 0; i_index < invariant.CHAIN_LINKS_MAX; i_index++ {
		fmt.Fprintf(&source, ".Impossible(%q, invariant.Event_True(\"axis\"))",
			fmt.Sprintf("rule %d", i_index))
	}
	source.WriteString(".Ensure() }\n")
	_, output, code = registered_fixture(source.String())
	if code != 1 {
		t.Fatalf("link-cap exit = %d, want 1; output=%q", code, output.String())
	}
	runtime_recorder := &invariant.Recorder{}
	product := invariant.Recorder_Dot_Product(runtime_recorder, "runtime links").
		Sometimes(true, "axis")
	for i_index := 1; i_index < invariant.CHAIN_LINKS_MAX; i_index++ {
		product = product.Impossible(
			fmt.Sprintf("rule %d", i_index), invariant.Event_True("axis"))
	}
	message := panic_text(func() {
		product = product.Impossible("excess", invariant.Event_True("axis"))
	})
	if message != "" {
		t.Fatalf("excess link panicked before Ensure: %q", message)
	}
	message = panic_text(product.Ensure)
	if !strings.Contains(message, "255 links") {
		t.Fatalf("runtime link cap panic = %q", message)
	}
	product = invariant.Recorder_Dot_Product(&invariant.Recorder{}, "runtime axes")
	for i_index := 0; i_index < invariant.CHAIN_LINKS_MAX; i_index++ {
		product = product.Sometimes(true, fmt.Sprintf("axis %d", i_index))
	}
	if message = panic_text(product.Ensure); message != "" {
		t.Fatalf("255-axis Ensure panic = %q", message)
	}
	product = invariant.Recorder_Dot_Product(&invariant.Recorder{}, "high axis")
	for i_index := 0; i_index < invariant.CHAIN_LINKS_MAX-1; i_index++ {
		product = product.Sometimes(i_index == invariant.CHAIN_LINKS_MAX-2,
			fmt.Sprintf("axis %d", i_index))
	}
	product = product.Impossible("high axis is forbidden",
		invariant.Event_True(fmt.Sprintf("axis %d", invariant.CHAIN_LINKS_MAX-2)))
	message = panic_text(product.Ensure)
	if !strings.Contains(message, "high axis is forbidden") {
		t.Fatalf("high-axis rule panic = %q", message)
	}
}

// Test_Bundles_Template prevents its specification contract from regressing.
func Test_Bundles_Template(t *testing.T) {
	const SOURCE = `package fixture
func Number_Invariants(n int, namespace invariant.Namespace) {
	invariant.Dot_Product(namespace).Sometimes(n == 0, "zero").Ensure()
}
func check(n int) { Number_Invariants(n, "number") }
`
	recorder, _, _ := registered_fixture(SOURCE)
	chain_metadata(&chain_metadata_input{Test: t, Recorder: recorder,
		Key: invariant.Chain_Key{Namespace: "number", Ordinal: 0, Message: "zero"}})
}

// Test_Bundles_Range_Template prevents its specification contract from regressing.
func Test_Bundles_Range_Template(t *testing.T) {
	const SOURCE = `package fixture
const MIN = -2
const MAX = 2
type Number int
func Number_Invariants(n Number, namespace invariant.Namespace) {
	invariant.Range_Invariants(n, MIN, MAX, namespace)
}
func check(n Number) { Number_Invariants(n, "number") }
`
	recorder, _, _ := registered_fixture(SOURCE)
	if event_count(&recorder.Events) == 0 {
		t.Fatal("a Range template must seed its callsite")
	}
}

// Test_Bundles_Descent prevents its specification contract from regressing.
func Test_Bundles_Descent(t *testing.T) {
	Test_Bundles_Template(t)
}

// Test_Bundles_Composition prevents its specification contract from regressing.
func Test_Bundles_Composition(t *testing.T) {
	const SOURCE = `package fixture
type Number int
func Number_Invariants(n Number, namespace invariant.Namespace) {
	invariant.Dot_Product(namespace).Sometimes(n == 0, "zero").Ensure()
}
func Pair_Invariants(n Number, namespace invariant.Namespace) {
	Number_Invariants(n, "pair.number")
	invariant.Dot_Product(namespace).Sometimes(n == 1, "one").Ensure()
}
func check(n Number) { Pair_Invariants(n, "pair") }
`
	recorder, _, _ := registered_fixture(SOURCE)
	chain_metadata(&chain_metadata_input{Test: t, Recorder: recorder,
		Key: invariant.Chain_Key{Namespace: "pair", Ordinal: 0, Message: "one"}})
	chain_metadata(&chain_metadata_input{Test: t, Recorder: recorder,
		Key: invariant.Chain_Key{Namespace: "pair.number", Ordinal: 0, Message: "zero"}})
}

// Test_Bundles_Casing prevents its specification contract from regressing.
func Test_Bundles_Casing(t *testing.T) {
	const SOURCE = `package fixture
type number int
func number_invariants(n number, namespace invariant.Namespace) {
	invariant.Dot_Product(namespace).Sometimes(n == 0, "zero").Ensure()
}
func check(n number) { number_invariants(n, "number") }
`
	recorder, _, _ := registered_fixture(SOURCE)
	chain_metadata(&chain_metadata_input{Test: t, Recorder: recorder,
		Key: invariant.Chain_Key{Namespace: "number", Ordinal: 0, Message: "zero"}})
}

// Test_Bundles_Sugar prevents its specification contract from regressing.
func Test_Bundles_Sugar(t *testing.T) {
	const SOURCE = `package invariant
func Number_Invariants(n int, namespace Namespace) {
	Dot_Product(namespace).Sometimes(n == 0, "zero").Ensure()
}
func check(n int) { Number_Invariants(n, "number") }
`
	recorder, _, _ := registered_fixture_with_sugar(SOURCE)
	chain_metadata(&chain_metadata_input{Test: t, Recorder: recorder,
		Key: invariant.Chain_Key{Namespace: "number", Ordinal: 0, Message: "zero"}})
}

// Test_Bundles_Cross_Package prevents its specification contract from regressing.
func Test_Bundles_Cross_Package(t *testing.T) {
	const SOURCE = `package fixture
func Number_Invariants(n int, namespace invariant.Namespace) {
	invariant.Dot_Product(namespace).Sometimes(n == 0, "zero").Ensure()
}
func check(n int) { Number_Invariants(n, "number") }
`
	recorder, _, _ := registered_fixture(SOURCE)
	chain_metadata(&chain_metadata_input{Test: t, Recorder: recorder,
		Key: invariant.Chain_Key{Namespace: "number", Ordinal: 0, Message: "zero"}})
}

// Test_Bundles_Callsite prevents its specification contract from regressing.
func Test_Bundles_Callsite(t *testing.T) {
	const SOURCE = `package fixture
func Number_Invariants(n int, namespace invariant.Namespace) {
	invariant.Dot_Product(namespace).Sometimes(n == 0, "zero").Ensure()
}
func first(n int) { Number_Invariants(n, "first") }
func second(n int) { Number_Invariants(n, "second") }
`
	recorder, _, _ := registered_fixture(SOURCE)
	chain_metadata(&chain_metadata_input{Test: t, Recorder: recorder,
		Key: invariant.Chain_Key{Namespace: "first", Ordinal: 0, Message: "zero"}})
	chain_metadata(&chain_metadata_input{Test: t, Recorder: recorder,
		Key: invariant.Chain_Key{Namespace: "second", Ordinal: 0, Message: "zero"}})
}

// Test_Bundles_Gap_Location prevents its specification contract from regressing.
func Test_Bundles_Gap_Location(t *testing.T) {
	const SOURCE = `package fixture
func Number_Invariants(n int, namespace invariant.Namespace) {
	invariant.Dot_Product(namespace).Sometimes(n == 0, "zero").Ensure()
}
func check(n int) { Number_Invariants(n, "number") }
`
	recorder, output, _ := registered_fixture(SOURCE)
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	if !strings.Contains(output.String(), "number · 0 · zero") {
		t.Fatalf("gap = %q, want chain identity", output.String())
	}
}

// Test_Bundles_Failure_Location prevents its specification contract from regressing.
func Test_Bundles_Failure_Location(t *testing.T) {
	recorder := &invariant.Recorder{}
	message := panic_text(func() {
		invariant.Recorder_Dot_Product(recorder, "number").
			Sometimes(true, "zero").
			Impossible("forbidden number", invariant.Event_True("zero")).
			Ensure()
	})
	if !strings.Contains(message, "forbidden number") {
		t.Fatalf("panic = %q, want rule name", message)
	}
}

// Test_Bundles_Static prevents its specification contract from regressing.
func Test_Bundles_Static(t *testing.T) {
	const SOURCE = `package fixture
type Number int
func Number_Invariants(n Number, namespace invariant.Namespace) {
	if n == 0 { invariant.Dot_Product(namespace).Sometimes(true, "zero").Ensure() }
}
func check(n Number) { Number_Invariants(n, "number") }
`
	_, output, code := registered_fixture(SOURCE)
	if code != 1 {
		t.Fatalf("exit = %d, want 1; output=%q", code, output.String())
	}
}

// Test_Bundles_Custom_Types prevents its specification contract from regressing.
func Test_Bundles_Custom_Types(t *testing.T) {
	const SOURCE = `package fixture
func Int_Invariants(n int, namespace invariant.Namespace) {
	invariant.Dot_Product(namespace).Sometimes(n == 0, "zero").Ensure()
}
func check(n int) { Int_Invariants(n, "number") }
`
	_, output, code := registered_fixture(SOURCE)
	if code != 1 {
		t.Fatalf("exit = %d, want 1; output=%q", code, output.String())
	}
}

// Test_Analysis_Gaps prevents its specification contract from regressing.
func Test_Analysis_Gaps(t *testing.T) {
	recorder, output, _ := registered_chain_fixture()
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	if output.Len() == 0 {
		t.Fatal("unobserved obligations must be reported")
	}
}

// Test_Analysis_Combination prevents its specification contract from regressing.
func Test_Analysis_Combination(t *testing.T) {
	recorder, output, _ := registered_chain_fixture()
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	if !strings.Contains(output.String(), "Cross-product") {
		t.Fatalf("report = %q, want Cross-product", output.String())
	}
}

// Test_Analysis_Legend prevents its specification contract from regressing.
func Test_Analysis_Legend(t *testing.T) {
	recorder, output, _ := registered_chain_fixture()
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	if !strings.Contains(output.String(), "zero") {
		t.Fatalf("report = %q, want axis legend", output.String())
	}
}

// Test_Analysis_Summary prevents its specification contract from regressing.
func Test_Analysis_Summary(t *testing.T) {
	recorder, _, _ := registered_chain_fixture()
	cover_chain_grid(recorder)
	summary := invariant.Recorder_Assertion_Summary(recorder)
	if !strings.Contains(summary, "properties") {
		t.Fatalf("summary = %q, want property tally", summary)
	}
}

// Test_Analysis_Tally prevents its specification contract from regressing.
func Test_Analysis_Tally(t *testing.T) {
	recorder, _, _ := registered_chain_fixture()
	cover_chain_grid(recorder)
	summary := invariant.Recorder_Assertion_Summary(recorder)
	if !strings.Contains(summary, "combinations") {
		t.Fatalf("summary = %q, want combinations", summary)
	}
}

// Test_Analysis_Summary_Names_Package prevents its specification contract from regressing.
func Test_Analysis_Summary_Names_Package(t *testing.T) {
	recorder, _, _ := registered_chain_fixture()
	recorder.Package_Label = "fixture"
	cover_chain_grid(recorder)
	summary := invariant.Recorder_Assertion_Summary(recorder)
	if !strings.Contains(summary, "fixture") {
		t.Fatalf("summary = %q, want package", summary)
	}
}

// Test_Analysis_Clean prevents its specification contract from regressing.
func Test_Analysis_Clean(t *testing.T) {
	recorder, output, _ := registered_chain_fixture()
	cover_chain_grid(recorder)
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	if output.Len() != 0 {
		t.Fatalf("clean output = %q, want empty", output.String())
	}
}

// Test_Coverage_Modes prevents its specification contract from regressing.
func Test_Coverage_Modes(t *testing.T) {
	recorder, _, _ := registered_chain_fixture()
	recorder.Is_Benchmark = true
	invariant.Recorder_Dot_Product(recorder, "check").
		Sometimes(true, "zero").
		Sometimes(false, "one").
		Impossible("exclusive", invariant.Event_True("zero"), invariant.Event_True("one")).
		Ensure()
	metadata := chain_metadata(&chain_metadata_input{Test: t, Recorder: recorder,
		Key: invariant.Chain_Key{Namespace: "check", Ordinal: 0, Message: "zero"}})
	if metadata.Frequency.Load() != 0 {
		t.Fatal("benchmarks must not record")
	}
}

// Test_Coverage_Enforcement prevents its specification contract from regressing.
func Test_Coverage_Enforcement(t *testing.T) {
	recorder := &invariant.Recorder{Is_Benchmark: true}
	message := panic_text(func() {
		invariant.Recorder_Dot_Product(recorder, "check").
			Sometimes(true, "zero").
			Impossible("forbidden", invariant.Event_True("zero")).Ensure()
	})
	if message == "" {
		t.Fatal("benchmark mode must enforce")
	}
}

// Test_Coverage_Uniqueness prevents its specification contract from regressing.
func Test_Coverage_Uniqueness(t *testing.T) {
	const SOURCE = `package fixture
func first(v bool) { invariant.Dot_Product("same").Sometimes(v, "a").Ensure() }
func second(v bool) { invariant.Dot_Product("same").Sometimes(v, "b").Ensure() }
`
	_, output, code := registered_fixture(SOURCE)
	if code != 1 {
		t.Fatalf("exit = %d, want duplicate failure; output=%q", code, output.String())
	}
	const TEMPLATE_SOURCE = `package fixture
type Number int
func Number_Invariants(n Number, namespace invariant.Namespace) {
	invariant.Dot_Product(namespace).Sometimes(n == 0, "zero").Ensure()
	invariant.Dot_Product(namespace).Sometimes(n == 1, "one").Ensure()
}
func check(n Number) { Number_Invariants(n, "number") }
`
	_, output, code = registered_fixture(TEMPLATE_SOURCE)
	if code != 1 {
		t.Fatalf("template duplicate exit = %d, want 1; output=%q", code, output.String())
	}
}

// Test_Coverage_Literal prevents its specification contract from regressing.
func Test_Coverage_Literal(t *testing.T) {
	const SOURCE = `package fixture
func check(v bool, message string) {
	invariant.Dot_Product("check").Sometimes(v, message).Ensure()
}
`
	_, output, code := registered_fixture(SOURCE)
	if code != 1 {
		t.Fatalf("exit = %d, want literal failure; output=%q", code, output.String())
	}
}

// Test_Coverage_Unresolved prevents its specification contract from regressing.
func Test_Coverage_Unresolved(t *testing.T) {
	const SOURCE = `package fixture
func check(v bool) { Missing(v, "check").Sometimes(v, "value").Ensure() }
`
	_, output, code := registered_fixture(SOURCE)
	if code != 1 {
		t.Fatalf("exit = %d, want unresolved failure; output=%q", code, output.String())
	}
}

// Test_Range_Guard prevents its specification contract from regressing.
func Test_Range_Guard(t *testing.T) {
	recorder := &invariant.Recorder{}
	if panic_text(func() { invariant.Recorder_Range(recorder, 3, 0, 2, "range") }) == "" {
		t.Fatal("out-of-range value must panic")
	}
}

// Test_Range_Coverage prevents its specification contract from regressing.
func Test_Range_Coverage(t *testing.T) {
	recorder, _, _ := registered_fixture(`package fixture
func check(v int) { invariant.Range_Invariants(v, -2, 2, "range") }
`)
	if event_count(&recorder.Events) == 0 {
		t.Fatal("Range must seed its boundaries and tuples")
	}
}

// Test_Range_Saturation prevents its specification contract from regressing.
func Test_Range_Saturation(t *testing.T) {
	recorder, _, _ := registered_fixture(`package fixture
func check(v int) { invariant.Range_Invariants(v, 0, 1, "range") }
`)
	if _, exists := recorder.Events.Load("range:tuple=(0,0)"); exists {
		t.Fatal("saturated all-false tuple must be carved")
	}
}

// Test_Range_Exclusions prevents its specification contract from regressing.
func Test_Range_Exclusions(t *testing.T) {
	recorder := &invariant.Recorder{}
	if panic_text(func() { invariant.Recorder_Range(recorder, 1, 0, 2, "range", 1) }) == "" {
		t.Fatal("excluded value must panic")
	}
}

// Test_Range_Enum prevents its specification contract from regressing.
func Test_Range_Enum(t *testing.T) {
	recorder := &invariant.Recorder{}
	if panic_text(func() { invariant.Recorder_Enum(recorder, 2, "enum", 0, 1) }) == "" {
		t.Fatal("non-member must panic")
	}
}

// Test_Range_Registration prevents its specification contract from regressing.
func Test_Range_Registration(t *testing.T) {
	recorder, _, _ := registered_fixture(`package fixture
const MIN = -2
const MAX = 2
func check(v int) { invariant.Range_Invariants(v, MIN, MAX, "range") }
`)
	if event_count(&recorder.Events) == 0 {
		t.Fatal("constant Range bounds must register")
	}
}

func registered_chain_fixture() (
	recorder *invariant.Recorder, output *bytes.Buffer, code int,
) {
	return registered_fixture(`package fixture
func check(n int) {
	invariant.Dot_Product("check").
		Sometimes(n == 0, "zero").
		Sometimes(n == 1, "one").
		Impossible("exclusive", invariant.Event_True("zero"), invariant.Event_True("one")).
		Ensure()
}
`)
}

func registered_fixture(source string) (
	recorder *invariant.Recorder, output *bytes.Buffer, code int,
) {
	return registered_fixture_options(&registered_fixture_options_input{Source: source})
}

func registered_fixture_with_sugar(source string) (
	recorder *invariant.Recorder, output *bytes.Buffer, code int,
) {
	return registered_fixture_options(&registered_fixture_options_input{
		Source: source, Sugar: "fixture/fixture",
	})
}

type registered_fixture_options_input struct {
	Source string
	Sugar  string
}

func registered_fixture_options(input *registered_fixture_options_input) (
	recorder *invariant.Recorder, output *bytes.Buffer, code int,
) {
	output = &bytes.Buffer{}
	tty := &bytes.Buffer{}
	code = -1
	recorder = &invariant.Recorder{
		File_System: fstest.MapFS{
			"go.mod":           &fstest.MapFile{Data: []byte("module fixture\n")},
			"fixture/check.go": &fstest.MapFile{Data: []byte(input.Source)},
		},
		Packages_To_Analyze: []string{"/fixture"},
		Output:              output,
		Tty:                 tty,
		Exit:                func(status int) { code = status },
		Is_Test:             true,
		Sugar_Package:       input.Sugar,
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	return recorder, output, code
}

type chain_metadata_input struct {
	Test     *testing.T
	Recorder *invariant.Recorder
	Key      invariant.Chain_Key
}

func chain_metadata(input *chain_metadata_input) (metadata *invariant.Assertion_Metadata) {
	input.Test.Helper()
	key := string(input.Key.Namespace) + invariant.ELEMENT_MESSAGE_SEPARATOR
	key += fmt.Sprint(input.Key.Ordinal) + invariant.ELEMENT_MESSAGE_SEPARATOR
	key += input.Key.Message
	value, exists := input.Recorder.Events.Load(key)
	if !exists {
		input.Test.Fatalf("missing chain key %q", key)
	}
	return value.(*invariant.Assertion_Metadata)
}

func event_count(events *sync.Map) (count int) {
	events.Range(func(_, _ any) (more bool) {
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

func cover_chain_grid(recorder *invariant.Recorder) {
	values := [3][2]bool{{false, false}, {true, false}, {false, true}}
	for _, value := range values {
		invariant.Recorder_Dot_Product(recorder, "check").
			Sometimes(value[0], "zero").
			Sometimes(value[1], "one").
			Impossible("exclusive",
				invariant.Event_True("zero"), invariant.Event_True("one")).
			Ensure()
	}
}
