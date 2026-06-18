package invariant_test

import (
	"bytes"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/james-orcales/james-orcales/shared/invariant"
	snap "github.com/james-orcales/james-orcales/shared/snap/default"
)

// Test_Always_Violation: a false Always panics on its own, in every run mode, naming
// itself by its message — there is no Dot_Product to defer to.
func Test_Always_Violation(t *testing.T) {
	recorder := new_test_recorder()
	message := recover_message(func() {
		invariant.Recorder_Always(recorder, false, "balance non-negative")
	})
	if message == "" {
		t.Fatal("a false Always must panic on its own")
	}
	if !strings.Contains(message, "balance non-negative") {
		t.Fatalf("the panic must name the Always by its message, got: %s", message)
	}
	if !strings.Contains(message, "Always — condition was false") {
		t.Fatalf("the panic must describe the Always violation, got: %s", message)
	}
}

// Test_Always_Eager: a false Always panics at its own call, before the next statement runs
// — it is never inert and never waits for a Dot_Product to consume it.
func Test_Always_Eager(t *testing.T) {
	recorder := new_test_recorder()
	reached_next := false
	did_panic(func() {
		invariant.Recorder_Always(recorder, false, "guard")
		reached_next = true
	})
	if reached_next {
		t.Fatal("a false Always must panic at its own call, before the following statement")
	}
}

// Test_Always_Reachability: an Always the suite never reaches is a coverage gap.
func Test_Always_Reachability(t *testing.T) {
	var output bytes.Buffer
	exit_code := -1
	recorder := &invariant.Recorder{
		Is_Test: true, Output: &output, Exit: func(code int) { exit_code = code },
	}
	metadata := &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Always, Message: "positive", Condition: "x > 0",
	}
	recorder.Events.Store("positive", metadata)

	invariant.Recorder_Analyze_Assertion_Frequency(recorder)

	if exit_code != 1 {
		t.Fatalf("an unreached Always must exit 1, got %d", exit_code)
	}
	if !strings.Contains(output.String(), "positive") {
		t.Errorf("report must name the unreached Always by its message, got: %s",
			output.String())
	}
}

// Test_Sometimes_Coverage: a consumed Sometimes credits the branch it fired on.
func Test_Sometimes_Coverage(t *testing.T) {
	recorder := new_test_recorder()
	recorder.Is_Test = true
	element := invariant.Recorder_Sometimes(recorder, true, "zero")
	key := "check" + invariant.Element_Message_Separator + element.Message
	metadata := &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Sometimes, Message: key,
	}
	recorder.Events.Store(key, metadata)

	invariant.Recorder_Dot_Product(recorder, "check", element)

	if metadata.Frequency.Load() != 1 {
		t.Fatalf("a true Sometimes must credit the true branch, got %d",
			metadata.Frequency.Load())
	}
	if metadata.False_Frequency.Load() != 0 {
		t.Fatalf("a true Sometimes must not credit the false branch, got %d",
			metadata.False_Frequency.Load())
	}
}

// Test_Sometimes_Gap: a Sometimes seen only one way reports the branch it missed,
// whichever it was.
func Test_Sometimes_Gap(t *testing.T) {
	cases := []struct {
		Name      string
		Seen_True bool
		Reason    string
	}{
		{"only true observed", true, "false branch never observed"},
		{"only false observed", false, "true branch never observed"},
	}
	for _, one := range cases {
		var output bytes.Buffer
		exit_code := -1
		recorder := &invariant.Recorder{
			Is_Test: true, Output: &output, Exit: func(code int) { exit_code = code },
		}
		metadata := &invariant.Assertion_Metadata{
			Kind:    invariant.Assertion_Kind_Sometimes,
			Message: "zero", Condition: "n == 0",
		}
		if one.Seen_True {
			metadata.Frequency.Add(1)
		} else {
			metadata.False_Frequency.Add(1)
		}
		recorder.Events.Store("zero", metadata)

		invariant.Recorder_Analyze_Assertion_Frequency(recorder)

		if exit_code != 1 {
			t.Fatalf("%s: a one-sided Sometimes must exit 1, got %d",
				one.Name, exit_code)
		}
		if !strings.Contains(output.String(), one.Reason) {
			t.Errorf("%s: report must contain %q, got: %s",
				one.Name, one.Reason, output.String())
		}
	}
}

// Test_Impossible_Violation: when the forbidden combination occurs, Dot_Product panics,
// naming each co-occurring axis by its message.
func Test_Impossible_Violation(t *testing.T) {
	recorder := new_test_recorder()
	first := invariant.Recorder_Sometimes(recorder, true, "first")
	second := invariant.Recorder_Sometimes(recorder, true, "second")
	forbidden := invariant.Impossible(
		invariant.Event_True("first"), invariant.Event_True("second"))
	message := recover_message(func() {
		invariant.Recorder_Dot_Product(recorder, "check", first, second, forbidden)
	})
	if message == "" {
		t.Fatal("an Impossible whose combination occurs must panic")
	}
	if !strings.Contains(message, first.Message) {
		t.Fatalf("the panic must name the first co-occurring axis %q, got: %s",
			first.Message, message)
	}
	if !strings.Contains(message, second.Message) {
		t.Fatalf("the panic must name the second co-occurring axis %q, got: %s",
			second.Message, message)
	}
}

// Test_Impossible_Absent: when the combination is not fully present, no panic.
func Test_Impossible_Absent(t *testing.T) {
	recorder := new_test_recorder()
	first := invariant.Recorder_Sometimes(recorder, true, "first")
	second := invariant.Recorder_Sometimes(recorder, false, "second")
	forbidden := invariant.Impossible(
		invariant.Event_True("first"), invariant.Event_True("second"))
	if did_panic(func() {
		invariant.Recorder_Dot_Product(recorder, "check", first, second, forbidden)
	}) {
		t.Fatal("an Impossible whose combination is absent must not panic")
	}
}

// Test_Impossible_Glob: naming a subset of axes carves every cell matching the named
// events across all values of the unnamed axes.
func Test_Impossible_Glob(t *testing.T) {
	const source = `package fixture

func check(n int) {
	a := invariant.Sometimes(n == 0, "a")
	b := invariant.Sometimes(n == 1, "b")
	c := invariant.Sometimes(n == 2, "c")
	invariant.Dot_Product("check",
		a, b, c,
		invariant.Impossible(invariant.Event_True("a"), invariant.Event_True("b")),
	)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/check.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	// Both a and b true is carved for either value of the unnamed c, so neither
	// (1,1,0) nor (1,1,1) survives.
	if _, ok := recorder.Events.Load("check:tuple=(1,1,0)"); ok {
		t.Error("the (1,1,0) cell must be carved across the unnamed c axis")
	}
	if _, ok := recorder.Events.Load("check:tuple=(1,1,1)"); ok {
		t.Error("the (1,1,1) cell must be carved across the unnamed c axis")
	}
	// A cell that does not match the named events fully survives — only a and b
	// both true is forbidden.
	if _, ok := recorder.Events.Load("check:tuple=(1,0,1)"); !ok {
		t.Error("a cell not matching both named events must survive")
	}
}

// Test_Impossible_Sibling: an Impossible may reference only axes of its own Dot_Product. Naming a
// message that is not a sibling panics at the Dot_Product on every call — a structural precondition
// checked before recording, independent of whether the forbidden combination can occur (here the
// named axis is not even present in the product).
func Test_Impossible_Sibling(t *testing.T) {
	recorder := new_test_recorder()
	present := invariant.Recorder_Sometimes(recorder, true, "present")
	// "orphan" is not an axis of the product below — only "present" is — so the Impossible
	// names a message that is not a sibling.
	if !did_panic(func() {
		invariant.Recorder_Dot_Product(recorder, "check", present,
			invariant.Impossible(invariant.Event_True("orphan")))
	}) {
		t.Fatal("an Impossible naming a non-sibling must panic at the Dot_Product")
	}
}

// Test_Dot_Product_Inert: constructing a Sometimes enforces and records nothing until a
// Dot_Product consumes it. An Always, by contrast, is eager, as Test_Always_Eager shows.
func Test_Dot_Product_Inert(t *testing.T) {
	recorder := new_test_recorder()
	if did_panic(func() { invariant.Recorder_Sometimes(recorder, false, "zero") }) {
		t.Fatal("constructing a Sometimes must not panic on its own")
	}
}

// Test_Dot_Product_Identity: an element is keyed by the author-supplied message it carries.
func Test_Dot_Product_Identity(t *testing.T) {
	recorder := new_test_recorder()
	element := invariant.Recorder_Sometimes(recorder, true, "balance positive")
	if element.Message != "balance positive" {
		t.Fatalf("element Message = %q, want \"balance positive\"", element.Message)
	}
}

// Test_Dot_Product_Grid: registration seeds the surviving cells and drops the carve. The
// grid is over the varying Sometimes axes alone; the bare Always is not an element and
// occupies no coordinate — it seeds only its own reachability entry under its message.
func Test_Dot_Product_Grid(t *testing.T) {
	const source = `package fixture

func check(n int) {
	invariant.Always(n >= 0, "non-negative")
	zero := invariant.Sometimes(n == 0, "zero")
	one := invariant.Sometimes(n == 1, "one")
	invariant.Dot_Product("check",
		zero,
		one,
		invariant.Impossible(invariant.Event_True("zero"), invariant.Event_True("one")),
	)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/check.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if _, ok := recorder.Events.Load("check:tuple=(0,0)"); !ok {
		t.Error("the surviving (0,0) cell must be seeded")
	}
	if _, ok := recorder.Events.Load("check:tuple=(1,1)"); ok {
		t.Error("the (1,1) cell carved by the Impossible must not be seeded")
	}
	always, ok := recorder.Events.Load("non-negative")
	if !ok {
		t.Fatal("the bare Always must seed a reachability entry under its message")
	}
	if always.(*invariant.Assertion_Metadata).Kind != invariant.Assertion_Kind_Always {
		t.Error("the bare Always entry must be an Always axis")
	}
}

// Test_Dot_Product_Attribution: a panic names every element violated on the call — each
// triggered Impossible, not only the first. An eager Always is not part of this; a false one
// short-circuits before the Dot_Product runs.
func Test_Dot_Product_Attribution(t *testing.T) {
	recorder := new_test_recorder()
	a := invariant.Recorder_Sometimes(recorder, true, "a")
	b := invariant.Recorder_Sometimes(recorder, true, "b")
	c := invariant.Recorder_Sometimes(recorder, true, "c")
	message := recover_message(func() {
		invariant.Recorder_Dot_Product(recorder, "check", a, b, c,
			invariant.Impossible(invariant.Event_True("a"), invariant.Event_True("b")),
			invariant.Impossible(invariant.Event_True("a"), invariant.Event_True("c")),
		)
	})
	if strings.Count(message, "forbidden combination occurred") != 2 {
		t.Fatalf("panic must name each violated element, not only the first: %s", message)
	}
}

// Test_Bundles_Descent: a bundle spread into a Dot_Product is followed and seeded.
func Test_Bundles_Descent(t *testing.T) {
	const source = `package fixture

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, invariant.Sometimes(n < 0, "lo"))
}

func check(n int) {
	invariant.Dot_Product("field", Pair_Invariants(n)...)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/pair.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if _, ok := recorder.Events.Load(
		"field" + invariant.Element_Message_Separator + "lo"); !ok {
		t.Error("the bundle element must be seeded under the Dot_Product prefix")
	}
}

// Test_Bundles_Composition: a nested bundle's element rolls up to the top-level
// Dot_Product callsite, not the inner bundle's call site.
func Test_Bundles_Composition(t *testing.T) {
	const source = `package fixture

func Inner_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, invariant.Sometimes(n < 0, "inner"))
}

func Outer_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	dot_elements = append(dot_elements, invariant.Sometimes(n > 0, "outer"))
	return append(dot_elements, Inner_Invariants(n)...)
}

func check(n int) {
	invariant.Dot_Product("field", Outer_Invariants(n)...)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/nested.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if _, ok := recorder.Events.Load(
		"field" + invariant.Element_Message_Separator + "inner"); !ok {
		t.Error("the deeply nested element must attribute to the top-level prefix")
	}
}

// Test_Bundles_Binding: a bundle reached through a local binding, not a direct
// spread, is still descended.
func Test_Bundles_Binding(t *testing.T) {
	const source = `package fixture

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, invariant.Sometimes(n < 0, "lo"))
}

func check(n int) {
	elements := Pair_Invariants(n)
	invariant.Dot_Product("field", elements...)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/bound.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if _, ok := recorder.Events.Load(
		"field" + invariant.Element_Message_Separator + "lo"); !ok {
		t.Error("a bundle reached through a binding must be descended")
	}
}

// Test_Bundles_Casing: a snake_case _invariants bundle is recognized like the
// Ada_Case _Invariants form.
func Test_Bundles_Casing(t *testing.T) {
	const source = `package fixture

func pair_invariants(n int) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, invariant.Sometimes(n < 0, "lo"))
}

func check(n int) {
	invariant.Dot_Product("field", pair_invariants(n)...)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/lower.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if _, ok := recorder.Events.Load(
		"field" + invariant.Element_Message_Separator + "lo"); !ok {
		t.Error("a snake_case bundle must be recognized like the Ada_Case form")
	}
}

// Test_Bundles_Recorder_Form: a bundle element built with the explicit Recorder_
// constructor — which leads with the recorder argument — is descended like the bare
// sugar form, its condition read past that leading recorder.
func Test_Bundles_Recorder_Form(t *testing.T) {
	const source = `package fixture

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, invariant.Recorder_Sometimes(Default, n < 0, "lo"))
}

func check(n int) {
	invariant.Dot_Product("field", Pair_Invariants(n)...)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/recorder_form.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	value, ok := recorder.Events.Load(
		"field" + invariant.Element_Message_Separator + "lo")
	if !ok {
		t.Fatal("a Recorder_ element must be descended like the bare sugar form")
	}
	if condition := value.(*invariant.Assertion_Metadata).Condition; condition != "n < 0" {
		t.Errorf("Condition = %q, want \"n < 0\" past the recorder", condition)
	}
}

// Test_Bundles_Sugar: a bundle in the sugar package may call the primitives unqualified;
// the descent recognizes the bare call only because Sugar_Package names that package. The
// same fixture without Sugar_Package treats the bare call as the caller's own function and
// seeds nothing.
func Test_Bundles_Sugar(t *testing.T) {
	const sugar = `package sugar

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, Sometimes(n < 0, "lo"))
}
`
	const application = `package app

import (
	invariant "example.com/m/invariant"
	"example.com/m/sugar"
)

func check(n int) {
	invariant.Dot_Product("field", sugar.Pair_Invariants(n)...)
}
`
	files := fstest.MapFS{
		"m/go.mod":         &fstest.MapFile{Data: []byte("module example.com/m\n")},
		"m/sugar/sugar.go": &fstest.MapFile{Data: []byte(sugar)},
		"m/app/app.go":     &fstest.MapFile{Data: []byte(application)},
	}
	key := "field" + invariant.Element_Message_Separator + "lo"

	recognized := &invariant.Recorder{File_System: files, Sugar_Package: "example.com/m/sugar"}
	invariant.Recorder_Register_Packages_For_Analysis(recognized, "/m/app")
	if _, ok := recognized.Events.Load(key); !ok {
		t.Error("a sugar-package bundle's unqualified Sometimes must be recognized")
	}

	ignored := &invariant.Recorder{File_System: files}
	invariant.Recorder_Register_Packages_For_Analysis(ignored, "/m/app")
	if _, ok := ignored.Events.Load(key); ok {
		t.Error("without Sugar_Package a bare call is not a primitive; nothing seeds")
	}
}

// Test_Bundles_Cross_Package: a bundle in a sibling package of the same module is
// resolved through the module path.
func Test_Bundles_Cross_Package(t *testing.T) {
	const package_a = `package a

import invariant "example.com/m/invariant"

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, invariant.Sometimes(n < 0, "lo"))
}
`
	const package_b = `package b

import (
	invariant "example.com/m/invariant"
	"example.com/m/a"
)

func check(n int) {
	invariant.Dot_Product("field", a.Pair_Invariants(n)...)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"m/go.mod": &fstest.MapFile{Data: []byte("module example.com/m\n")},
			"m/a/a.go": &fstest.MapFile{Data: []byte(package_a)},
			"m/b/b.go": &fstest.MapFile{Data: []byte(package_b)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/m/b")

	if _, ok := recorder.Events.Load(
		"field" + invariant.Element_Message_Separator + "lo"); !ok {
		t.Error("a bundle in a sibling package of the module must be resolved")
	}
}

// Test_Bundles_Workspace: a bundle in a sibling module joined by a go.work workspace
// is resolved through the workspace.
func Test_Bundles_Workspace(t *testing.T) {
	recorder := &invariant.Recorder{File_System: workspace_bundle_fixture()}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/work/b")

	if _, ok := recorder.Events.Load(
		"field" + invariant.Element_Message_Separator + "lo"); !ok {
		t.Error("a bundle in a sibling workspace module must be resolved")
	}
}

// Test_Bundles_Callsite: spreading one bundle into two Dot_Products with distinct
// prefixes ("a" and "b") yields independent coverage entries — the per-field prefix is
// what keeps them apart — reusing one prefix is instead a fatal duplicate.
func Test_Bundles_Callsite(t *testing.T) {
	const source = `package fixture

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, invariant.Sometimes(n < 0, "lo"))
}

func check_a(n int) {
	invariant.Dot_Product("a", Pair_Invariants(n)...)
}

func check_b(n int) {
	invariant.Dot_Product("b", Pair_Invariants(n)...)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/two.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if _, ok := recorder.Events.Load(
		"a" + invariant.Element_Message_Separator + "lo"); !ok {
		t.Error("prefix A must have its own coverage entry")
	}
	if _, ok := recorder.Events.Load(
		"b" + invariant.Element_Message_Separator + "lo"); !ok {
		t.Error("prefix B must have its own coverage entry")
	}
}

// Test_Bundles_Gap_Location: a composed bundle's element gap is named by the consuming
// Dot_Product's prefix joined to the element's own message (covered by
// Test_Analysis_Legend). An eager Always written inside a bundle body is not an element;
// it fires when the bundle is built, and its reachability gap is named by its own bare
// message ("positive"), never a prefixed key.
func Test_Bundles_Gap_Location(t *testing.T) {
	const source = `package fixture

func Inner_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	invariant.Always(n > 0, "positive")
	return dot_elements
}

func Outer_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, Inner_Invariants(n)...)
}

func check(n int) {
	invariant.Dot_Product("field", Outer_Invariants(n)...)
}
`
	var output bytes.Buffer
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/compose.go": &fstest.MapFile{Data: []byte(source)},
		},
		Is_Test: true, Output: &output, Exit: func(code int) {},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)

	snap.Expect(t, snap.Init(`🚨 1 coverage gaps 🚨

# Reachability gaps
positive  Always — never reached: "n > 0"
🚨 1 coverage gaps 🚨
`), output.String())
}

// Test_Bundles_Failure_Location: a deferred violation (here a triggered Impossible) names its
// axes by their own message, never the consuming "field" prefix — yet the call site is not lost.
// The panic's Go stack still unwinds through Recorder_Dot_Product and the frame that spread the
// elements, so the snapshot pins both the message-only failure line and the stack frames.
func Test_Bundles_Failure_Location(t *testing.T) {
	recorder := new_test_recorder()
	a := invariant.Recorder_Sometimes(recorder, true, "a")
	b := invariant.Recorder_Sometimes(recorder, true, "b")
	forbidden := invariant.Impossible(invariant.Event_True("a"), invariant.Event_True("b"))
	message, stack := recover_with_stack(func() {
		dot_product_callsite(recorder, a, b, forbidden)
	})

	snap.Expect(
		t,
		snap.Init(`🚨 Assertion Failure 🚨: Impossible — forbidden combination occurred:
  a  true
  b  true

github.com/james-orcales/james-orcales/shared/invariant.Recorder_Dot_Product (invariant.go)
github.com/james-orcales/james-orcales/shared/invariant_test.dot_product_callsite (specification_test.go)`),
		message+"\n\n"+stack,
	)
}

// Test_Bundles_Ban: a Dot_Product inside a bundle fails registration.
func Test_Bundles_Ban(t *testing.T) {
	const source = `package fixture

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	invariant.Dot_Product("field", invariant.Sometimes(n < 0, "lo"))
	return dot_elements
}
`
	var output bytes.Buffer
	exit_code := -1
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/p.go": &fstest.MapFile{Data: []byte(source)},
		},
		Output: &output,
		Exit:   func(code int) { exit_code = code },
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if exit_code != 1 {
		t.Fatalf("a Dot_Product inside a bundle must exit 1, got %d", exit_code)
	}
	if !strings.Contains(output.String(), "p.go:4") {
		t.Errorf("the ban report must name the Dot_Product site, got: %s", output.String())
	}
}

// Test_Analysis_Gaps: a never-fired obligation is named by site and condition, while
// a fully exercised one is left unreported.
func Test_Analysis_Gaps(t *testing.T) {
	var output bytes.Buffer
	exit_code := -1
	recorder := &invariant.Recorder{
		Is_Test: true, Output: &output, Exit: func(code int) { exit_code = code },
	}
	gap := &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Sometimes, Message: "zero", Condition: "n == 0",
	}
	gap.Frequency.Add(1) // true seen, false never: a gap
	recorder.Events.Store("zero", gap)
	fired := &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Always, Message: "positive", Condition: "x > 0",
	}
	fired.Frequency.Add(1) // reached: not a gap
	recorder.Events.Store("positive", fired)

	invariant.Recorder_Analyze_Assertion_Frequency(recorder)

	if exit_code != 1 {
		t.Fatalf("a gap must exit 1, got %d", exit_code)
	}
	report := output.String()
	if !strings.Contains(report, "zero") {
		t.Errorf("the gap must be named by its message, got: %s", report)
	}
	if !strings.Contains(report, "n == 0") {
		t.Errorf("the gap must be named by its condition, got: %s", report)
	}
	if strings.Contains(report, "positive") {
		t.Errorf("a fully exercised obligation must not be reported, got: %s", report)
	}
}

// Test_Analysis_Combination: a never-witnessed grid cell is reported under
// cross-product gaps, named by its tuple of buckets.
func Test_Analysis_Combination(t *testing.T) {
	var output bytes.Buffer
	exit_code := -1
	recorder := &invariant.Recorder{
		Is_Test: true, Output: &output, Exit: func(code int) { exit_code = code },
	}
	tuple := &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Tuple, Message: "grid", Tuple_Indices: []int{1, 0},
	}
	recorder.Events.Store("grid:tuple=(1,0)", tuple)

	invariant.Recorder_Analyze_Assertion_Frequency(recorder)

	if exit_code != 1 {
		t.Fatalf("an unwitnessed cell must exit 1, got %d", exit_code)
	}
	report := output.String()
	if !strings.Contains(report, "Cross-product") {
		t.Errorf("the cell must be reported under cross-product gaps, got: %s", report)
	}
	if !strings.Contains(report, "(1,0)") {
		t.Errorf("the cell must be named by its tuple, got: %s", report)
	}
}

// Test_Analysis_Legend: across a nested bundle, a never-observed grid cell must be
// debuggable from the report alone. The grid prints its axis legend once — each position
// named by kind, condition, and the axis's own site (the deepest one for a composed
// bundle) — then each cell decodes its bucket back to the axis's event. Without it a bare
// "(1)" gives no way to learn which axis the position is or what bucket 1 means there. The
// constant Always is absent from the coordinate; its coverage is its reachability gap alone.
func Test_Analysis_Legend(t *testing.T) {
	const source = `package fixture

func Inner_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, invariant.Sometimes(n < 0, "negative"))
}

func Outer_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	invariant.Always(n != 0, "nonzero")
	return append(dot_elements, Inner_Invariants(n)...)
}

func check(n int) {
	invariant.Dot_Product("field", Outer_Invariants(n)...)
}
`
	var output bytes.Buffer
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/nested.go": &fstest.MapFile{Data: []byte(source)},
		},
		Is_Test: true, Output: &output, Exit: func(code int) {},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)

	snap.Expect(t, snap.Init(`🚨 5 coverage gaps 🚨

# Cross-product gaps
field  grid axes:
  [0] Sometimes "n < 0" from negative
field  tuple (0) never observed  ->  [0]=false
field  tuple (1) never observed  ->  [0]=true

# Branch gaps
field · negative  Sometimes — false branch never observed: "n < 0"
field · negative  Sometimes — true branch never observed: "n < 0"

# Reachability gaps
nonzero  Always — never reached: "n != 0"
🚨 5 coverage gaps 🚨
`), output.String())
}

// Test_Analysis_Summary: a clean run reports how many properties it tested, split into
// individual and combination with the panic-able subset counted.
func Test_Analysis_Summary(t *testing.T) {
	recorder := &invariant.Recorder{}
	store := func(key string, kind invariant.Assertion_Kind) {
		recorder.Events.Store(key, &invariant.Assertion_Metadata{Kind: kind})
	}
	store("a", invariant.Assertion_Kind_Always)
	store("b", invariant.Assertion_Kind_Sometimes)
	store("c:tuple=(0)", invariant.Assertion_Kind_Tuple)
	store("c:tuple=(1)", invariant.Assertion_Kind_Tuple)

	summary := invariant.Recorder_Assertion_Summary(recorder)

	want := "✓ tested 4 properties (2 individual + 2 combinations, of which 1 are panic-able)"
	if summary != want {
		t.Fatalf("summary = %q, want %q", summary, want)
	}
}

// Test_Analysis_Clean: a fully exercised run reports nothing and does not exit.
func Test_Analysis_Clean(t *testing.T) {
	var output bytes.Buffer
	exited := false
	recorder := &invariant.Recorder{
		Is_Test: true, Output: &output, Exit: func(code int) { exited = true },
	}
	metadata := &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Sometimes, Message: "zero", Condition: "n == 0",
	}
	metadata.Frequency.Add(1)
	metadata.False_Frequency.Add(1)
	recorder.Events.Store("zero", metadata)

	invariant.Recorder_Analyze_Assertion_Frequency(recorder)

	if exited {
		t.Error("a fully exercised run must not exit")
	}
	if output.Len() != 0 {
		t.Errorf("a fully exercised run must print nothing, got %q", output.String())
	}
}

// Test_Coverage_Modes: coverage is recorded in every mode but a benchmark — a plain test, a
// fuzzing coordinator, and a fuzz worker all credit observations (the worker runs the fuzzed
// body). The analysis runs in a plain test and the coordinator, never in a worker. Enforcement
// fires everywhere (see Coverage/Enforcement).
func Test_Coverage_Modes(t *testing.T) {
	// Records reports whether a Dot_Product observation credits a pre-seeded entry in the
	// given mode — i.e. whether coverage is recorded.
	records := func(is_fuzz, is_fuzz_worker, is_benchmark bool) (recorded bool) {
		recorder := &invariant.Recorder{
			Is_Test:        true,
			Is_Fuzz:        is_fuzz,
			Is_Fuzz_Worker: is_fuzz_worker,
			Is_Benchmark:   is_benchmark,
		}
		element := invariant.Recorder_Sometimes(recorder, true, "zero")
		key := "check" + invariant.Element_Message_Separator + element.Message
		metadata := &invariant.Assertion_Metadata{
			Kind: invariant.Assertion_Kind_Sometimes, Message: key,
		}
		recorder.Events.Store(key, metadata)
		invariant.Recorder_Dot_Product(recorder, "check", element)
		return metadata.Frequency.Load() == 1
	}
	if !records(false, false, false) {
		t.Error("a plain test run must record coverage")
	}
	if !records(true, false, false) {
		t.Error("a fuzz coordinator must record coverage")
	}
	if !records(true, true, false) {
		t.Error("a fuzz worker subprocess must record coverage (it runs the fuzzed body)")
	}
	if records(false, false, true) {
		t.Error("a benchmark must record no coverage")
	}

	// The analysis follows the same gate: a fuzz coordinator checks a seeded gap
	// (fatal), a fuzz worker checks nothing.
	analyzes := func(is_fuzz_worker bool) (exit int, reported bool) {
		var output bytes.Buffer
		exit = -1
		recorder := &invariant.Recorder{
			Is_Test: true, Is_Fuzz: true, Is_Fuzz_Worker: is_fuzz_worker,
			Output: &output, Exit: func(code int) { exit = code },
		}
		// A Sometimes that never fired either way: a gap.
		recorder.Events.Store("g", &invariant.Assertion_Metadata{
			Kind: invariant.Assertion_Kind_Sometimes, Message: "g",
		})
		invariant.Recorder_Analyze_Assertion_Frequency(recorder)
		return exit, output.Len() > 0
	}
	exit, reported := analyzes(false)
	if exit != 1 {
		t.Errorf("fuzz coordinator must exit=1 and report, got exit=%d reported=%v",
			exit, reported)
	}
	if !reported {
		t.Errorf("fuzz coordinator must exit=1 and report, got exit=%d reported=%v",
			exit, reported)
	}
	exit, reported = analyzes(true)
	if exit != -1 {
		t.Errorf("a fuzz worker must not analyze, got exit=%d reported=%v", exit, reported)
	}
	if reported {
		t.Errorf("a fuzz worker must not analyze, got exit=%d reported=%v", exit, reported)
	}
}

// Test_Coverage_Enforcement: enforcement runs in every mode, even when coverage is not
// being recorded (Is_Test false), like a production binary — both the eager Always and the
// Dot_Product element kinds.
func Test_Coverage_Enforcement(t *testing.T) {
	recorder := &invariant.Recorder{} // Is_Test false: not a coverage run
	if !did_panic(func() { invariant.Recorder_Always(recorder, false, "guard") }) {
		t.Fatal("an eager Always must enforce even when coverage is not recorded")
	}
	if !did_panic(func() {
		invariant.Recorder_Dot_Product(recorder, "check",
			invariant.Recorder_Sometimes(recorder, true, "a"),
			invariant.Recorder_Sometimes(recorder, true, "b"),
			invariant.Impossible(invariant.Event_True("a"), invariant.Event_True("b")))
	}) {
		t.Fatal("a Dot_Product must enforce its elements even without coverage recording")
	}
}

// Test_Coverage_Uniqueness: two Dot_Products sharing a message fail registration — a
// duplicate would silently merge two obligations and mask a gap.
func Test_Coverage_Uniqueness(t *testing.T) {
	const source = `package fixture

func check_a(n int) {
	invariant.Dot_Product("field", invariant.Sometimes(n < 0, "lo"))
}

func check_b(n int) {
	invariant.Dot_Product("field", invariant.Sometimes(n > 0, "hi"))
}
`
	var output bytes.Buffer
	exit_code := -1
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/check.go": &fstest.MapFile{Data: []byte(source)},
		},
		Output: &output,
		Exit:   func(code int) { exit_code = code },
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if exit_code != 1 {
		t.Fatalf("duplicate messages must exit 1, got %d", exit_code)
	}
	if !strings.Contains(output.String(), "duplicate") {
		t.Errorf("the report must name the collision, got: %s", output.String())
	}
}

// Test_Coverage_Literal: a non-literal message fails registration — the static side cannot
// key it, so the coverage would vanish if it were allowed through.
func Test_Coverage_Literal(t *testing.T) {
	const source = `package fixture

func check(n int) {
	msg := "zero"
	invariant.Dot_Product("field", invariant.Sometimes(n == 0, msg))
}
`
	var output bytes.Buffer
	exit_code := -1
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/check.go": &fstest.MapFile{Data: []byte(source)},
		},
		Output: &output,
		Exit:   func(code int) { exit_code = code },
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if exit_code != 1 {
		t.Fatalf("a non-literal message must exit 1, got %d", exit_code)
	}
	if !strings.Contains(output.String(), "non-literal") {
		t.Errorf("the report must say non-literal messages, got: %s", output.String())
	}
}

// Test_Coverage_Unresolved: a bundle the analyzer cannot resolve is fatal, never
// silently skipped.
func Test_Coverage_Unresolved(t *testing.T) {
	const source = `package b

import (
	invariant "example.com/m/invariant"
	"other.com/x"
)

func check(n int) {
	invariant.Dot_Product("field", x.Pair_Invariants(n)...)
}
`
	var output bytes.Buffer
	exit_code := -1
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"m/go.mod": &fstest.MapFile{Data: []byte("module example.com/m\n")},
			"m/b/b.go": &fstest.MapFile{Data: []byte(source)},
		},
		Output: &output,
		Exit:   func(code int) { exit_code = code },
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/m/b")

	if exit_code != 1 {
		t.Fatalf("an unresolvable bundle must exit 1, got %d", exit_code)
	}
	if !strings.Contains(output.String(), "Pair_Invariants") {
		t.Errorf("the report must name the unresolved bundle, got: %s", output.String())
	}
}

// Test_Coverage_Literal_Reference: an Impossible reference message must also be a string literal,
// like every message — a non-literal reference fails registration, since the static side cannot
// match it to a sibling axis to carve.
func Test_Coverage_Literal_Reference(t *testing.T) {
	const source = `package fixture

func check(n int) {
	a := invariant.Sometimes(n == 0, "a")
	b := invariant.Sometimes(n == 1, "b")
	axis := "a"
	invariant.Dot_Product("field", a, b,
		invariant.Impossible(invariant.Event_True(axis), invariant.Event_True("b")))
}
`
	var output bytes.Buffer
	exit_code := -1
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/check.go": &fstest.MapFile{Data: []byte(source)},
		},
		Output: &output,
		Exit:   func(code int) { exit_code = code },
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if exit_code != 1 {
		t.Fatalf("a non-literal Impossible reference must exit 1, got %d", exit_code)
	}
	if !strings.Contains(output.String(), "non-literal") {
		t.Errorf("the report must say non-literal, got: %s", output.String())
	}
}

// Test_Coverage_Sink_Fires_On_First_Coverage: a fuzz worker persists each cell the first
// time it is covered. Coverage_Sink fires on the 0→1 transition of a branch and not on
// later observations, so a long run appends at most one line per branch — the bound that
// keeps it off the syscall-per-assertion path.
func Test_Coverage_Sink_Fires_On_First_Coverage(t *testing.T) {
	type event struct {
		Key   string
		Fired bool
	}
	var sunk []event
	recorder := &invariant.Recorder{
		Is_Test:        true,
		Is_Fuzz:        true,
		Is_Fuzz_Worker: true,
		Coverage_Sink: func(key string, fired_true bool) {
			sunk = append(sunk, event{key, fired_true})
		},
	}
	key := "check" + invariant.Element_Message_Separator + "zero"
	recorder.Events.Store(key, &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Sometimes, Message: key,
	})

	for range 3 {
		invariant.Recorder_Dot_Product(
			recorder, "check", invariant.Recorder_Sometimes(recorder, true, "zero"))
	}
	for range 2 {
		invariant.Recorder_Dot_Product(
			recorder, "check", invariant.Recorder_Sometimes(recorder, false, "zero"))
	}

	if len(sunk) != 2 {
		t.Fatalf("sink must fire once per branch on first coverage, got %d events: %v",
			len(sunk), sunk)
	}
	seen_true, seen_false := false, false
	for _, e := range sunk {
		if e.Key != key {
			t.Errorf("sink key = %q, want %q", e.Key, key)
		}
		seen_true = seen_true || e.Fired
		seen_false = seen_false || !e.Fired
	}
	if !seen_true {
		t.Errorf("sink must fire for both branches, got %v", sunk)
	}
	if !seen_false {
		t.Errorf("sink must fire for both branches, got %v", sunk)
	}
}

// Test_Fuzz_Coverage_Line_Round_Trip: the worker's Fuzz_Coverage_Line and the coordinator's
// Recorder_Merge_Fuzz_Coverage_From are inverses — encoding cells (including a NUL-bearing
// key) and merging them back unions exactly those branches into a seeded grid, leaving the
// other branch alone and skipping a key with no seeded entry.
func Test_Fuzz_Coverage_Line_Round_Trip(t *testing.T) {
	nul_key := "field" + invariant.Element_Message_Separator + "empty"
	var file bytes.Buffer
	file.WriteString(invariant.Fuzz_Coverage_Line(nul_key, true))
	file.WriteString(invariant.Fuzz_Coverage_Line("plain", false))
	file.WriteString(invariant.Fuzz_Coverage_Line("unseeded", true)) // no entry: skipped

	recorder := &invariant.Recorder{Is_Test: true}
	for _, key := range []string{nul_key, "plain"} {
		recorder.Events.Store(key, &invariant.Assertion_Metadata{
			Kind: invariant.Assertion_Kind_Sometimes, Message: key,
		})
	}

	invariant.Recorder_Merge_Fuzz_Coverage_From(recorder, &file)

	covered, _ := recorder.Events.Load(nul_key)
	if covered.(*invariant.Assertion_Metadata).Frequency.Load() == 0 {
		t.Errorf("merge must mark %q covered on its true branch", nul_key)
	}
	if covered.(*invariant.Assertion_Metadata).False_Frequency.Load() != 0 {
		t.Errorf("merge must not touch %q's false branch", nul_key)
	}
	plain, _ := recorder.Events.Load("plain")
	if plain.(*invariant.Assertion_Metadata).False_Frequency.Load() == 0 {
		t.Error(`merge must mark "plain" covered on its false branch`)
	}
	if _, seeded := recorder.Events.Load("unseeded"); seeded {
		t.Error("merge must not create entries for unseeded keys")
	}
}

// Reports whether calling action panics, recovering so the test can assert on it.
func did_panic(action func()) (panicked bool) {
	defer func() {
		if recover() != nil {
			panicked = true
		}
	}()
	action()
	return false
}

// Returns the message action panics with, or "" when it does not panic, so a test can
// assert on which axis the panic names.
func recover_message(action func()) (message string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			message = fmt.Sprint(recovered)
		}
	}()
	action()
	return ""
}

// Spreads element into a Dot_Product from a stable, named frame, so the panic stack
// carries a recognizable call site for recover_with_stack to capture. The noinline keeps
// it a frame of its own rather than folding into the caller.
//
//go:noinline
func dot_product_callsite(recorder *invariant.Recorder, elements ...invariant.Dot_Element) {
	invariant.Recorder_Dot_Product(recorder, "field", elements...)
}

// Runs action, recovers the panic it raises, and returns the panic message together with
// the call stack at the panic point — kept to the invariant and call-site frames and
// rendered "func (file)", dropping line numbers, addresses, and the runtime/testing
// scaffolding — so a snapshot pins which frames the failure unwound through without
// machine-specific or line-shifting noise. Captured inside the defer, where the panicking
// frames below the recover are still live on the goroutine's stack.
func recover_with_stack(action func()) (message string, stack string) {
	defer func() {
		message = fmt.Sprint(recover())
		program_counters := make([]uintptr, 64)
		count := runtime.Callers(0, program_counters)
		frames := runtime.CallersFrames(program_counters[:count])
		var lines []string
		for range count {
			frame, more := frames.Next()
			is_core := strings.Contains(frame.Function, "/shared/invariant.")
			is_callsite := strings.HasSuffix(frame.Function, ".dot_product_callsite")
			keep := is_core || is_callsite
			if keep {
				rendered := frame.Function + " (" + filepath.Base(frame.File) + ")"
				lines = append(lines, rendered)
			}
			if !more {
				break
			}
		}
		stack = strings.Join(lines, "\n")
	}()
	action()
	return message, stack
}
