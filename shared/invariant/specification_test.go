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

// Test_Always_Violation: a Dot_Product given an Always observed false panics, naming
// the offending axis by its site.
func Test_Always_Violation(t *testing.T) {
	recorder := new_test_recorder()
	element := invariant.Recorder_Always(recorder, false)
	message := recover_message(func() {
		invariant.Recorder_Dot_Product(recorder, element)
	})
	if message == "" {
		t.Fatal("a Dot_Product given an Always(false) must panic")
	}
	if !strings.Contains(message, element.Site) {
		t.Fatalf("the panic must name the offending axis site %q, got: %s",
			element.Site, message)
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
		Kind: invariant.Assertion_Kind_Always, Site: "f.go:9", Condition: "x > 0",
	}
	recorder.Assertions.Store("f.go:9", metadata)

	invariant.Recorder_Analyze_Assertion_Frequency(recorder)

	if exit_code != 1 {
		t.Fatalf("an unreached Always must exit 1, got %d", exit_code)
	}
	if !strings.Contains(output.String(), "f.go:9") {
		t.Errorf("report must name the unreached Always site, got: %s", output.String())
	}
}

// Test_Sometimes_Coverage: a consumed Sometimes credits the branch it fired on.
func Test_Sometimes_Coverage(t *testing.T) {
	recorder := new_test_recorder()
	recorder.Is_Test = true
	element := invariant.Recorder_Sometimes(recorder, true)
	metadata := &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Sometimes, Site: element.Site,
	}
	recorder.Assertions.Store(element.Site, metadata)

	invariant.Recorder_Dot_Product(recorder, element)

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
			Kind: invariant.Assertion_Kind_Sometimes,
			Site: "f.go:3", Condition: "n == 0",
		}
		if one.Seen_True {
			metadata.Frequency.Add(1)
		} else {
			metadata.False_Frequency.Add(1)
		}
		recorder.Assertions.Store("f.go:3", metadata)

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

// Test_Distinct_Boundary_Endpoints: Hi credits true, Lo credits false, interior
// credits neither.
func Test_Distinct_Boundary_Endpoints(t *testing.T) {
	cases := []struct {
		Name      string
		X         int
		Frequency int64
		False     int64
	}{
		{"Hi endpoint", 3, 1, 0},
		{"Lo endpoint", 0, 0, 1},
		{"interior", 2, 0, 0},
	}
	for _, one := range cases {
		recorder := new_test_recorder()
		recorder.Is_Test = true
		element := invariant.Recorder_Distinct_Boundary(
			recorder, &invariant.Boundary_Input[int]{X: one.X, Lo: 0, Hi: 3})
		metadata := &invariant.Assertion_Metadata{
			Kind: invariant.Assertion_Kind_Boundary, Site: element.Site,
		}
		recorder.Assertions.Store(element.Site, metadata)

		invariant.Recorder_Dot_Product(recorder, element)

		if metadata.Frequency.Load() != one.Frequency {
			t.Errorf("%s: Frequency = %d, want %d",
				one.Name, metadata.Frequency.Load(), one.Frequency)
		}
		if metadata.False_Frequency.Load() != one.False {
			t.Errorf("%s: False_Frequency = %d, want %d",
				one.Name, metadata.False_Frequency.Load(), one.False)
		}
	}
}

// Test_Distinct_Boundary_Outside: a value beyond [Lo, Hi] panics at the Dot_Product,
// naming the boundary's site.
func Test_Distinct_Boundary_Outside(t *testing.T) {
	recorder := new_test_recorder()
	boundary := invariant.Recorder_Distinct_Boundary(
		recorder, &invariant.Boundary_Input[int]{X: 5, Lo: 0, Hi: 3})
	message := recover_message(func() {
		invariant.Recorder_Dot_Product(recorder, boundary)
	})
	if message == "" {
		t.Fatal("a value outside [Lo, Hi] must panic at the Dot_Product")
	}
	if !strings.Contains(message, boundary.Site) {
		t.Fatalf("the panic must name the boundary site %q, got: %s",
			boundary.Site, message)
	}
}

// Test_Distinct_Boundary_Bad_Bounds: endpoints that are not distinct panic, naming the
// boundary's site.
func Test_Distinct_Boundary_Bad_Bounds(t *testing.T) {
	recorder := new_test_recorder()
	boundary := invariant.Recorder_Distinct_Boundary(
		recorder, &invariant.Boundary_Input[int]{X: 3, Lo: 3, Hi: 3})
	message := recover_message(func() {
		invariant.Recorder_Dot_Product(recorder, boundary)
	})
	if message == "" {
		t.Fatal("endpoints that are not distinct must panic at the Dot_Product")
	}
	if !strings.Contains(message, boundary.Site) {
		t.Fatalf("the panic must name the boundary site %q, got: %s",
			boundary.Site, message)
	}
}

// Test_Impossible_Violation: when the forbidden combination occurs, Dot_Product panics,
// naming each co-occurring axis by its site.
func Test_Impossible_Violation(t *testing.T) {
	recorder := new_test_recorder()
	first := invariant.Recorder_Sometimes(recorder, true)
	second := invariant.Recorder_Sometimes(recorder, true)
	forbidden := invariant.Impossible(
		invariant.Event_True(first), invariant.Event_True(second))
	message := recover_message(func() {
		invariant.Recorder_Dot_Product(recorder, first, second, forbidden)
	})
	if message == "" {
		t.Fatal("an Impossible whose combination occurs must panic")
	}
	if !strings.Contains(message, first.Site) {
		t.Fatalf("the panic must name the first co-occurring axis %q, got: %s",
			first.Site, message)
	}
	if !strings.Contains(message, second.Site) {
		t.Fatalf("the panic must name the second co-occurring axis %q, got: %s",
			second.Site, message)
	}
}

// Test_Impossible_Absent: when the combination is not fully present, no panic.
func Test_Impossible_Absent(t *testing.T) {
	recorder := new_test_recorder()
	first := invariant.Recorder_Sometimes(recorder, true)
	second := invariant.Recorder_Sometimes(recorder, false)
	forbidden := invariant.Impossible(
		invariant.Event_True(first), invariant.Event_True(second))
	if did_panic(func() {
		invariant.Recorder_Dot_Product(recorder, first, second, forbidden)
	}) {
		t.Fatal("an Impossible whose combination is absent must not panic")
	}
}

// Test_Impossible_Glob: naming a subset of axes carves every cell matching the named
// events across all values of the unnamed axes.
func Test_Impossible_Glob(t *testing.T) {
	const source = `package fixture

func check(n int) {
	a := invariant.Sometimes(n == 0)
	b := invariant.Sometimes(n == 1)
	c := invariant.Sometimes(n == 2)
	invariant.Dot_Product(
		a, b, c,
		invariant.Impossible(invariant.Event_True(a), invariant.Event_True(b)),
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
	if _, ok := recorder.Assertions.Load("/fixture/check.go:7:tuple=(1,1,0)"); ok {
		t.Error("the (1,1,0) cell must be carved across the unnamed c axis")
	}
	if _, ok := recorder.Assertions.Load("/fixture/check.go:7:tuple=(1,1,1)"); ok {
		t.Error("the (1,1,1) cell must be carved across the unnamed c axis")
	}
	// A cell that does not match the named events fully survives — only a and b
	// both true is forbidden.
	if _, ok := recorder.Assertions.Load("/fixture/check.go:7:tuple=(1,0,1)"); !ok {
		t.Error("a cell not matching both named events must survive")
	}
}

// Test_Dot_Product_Inert: constructing an element on its own enforces nothing.
func Test_Dot_Product_Inert(t *testing.T) {
	recorder := new_test_recorder()
	if did_panic(func() { invariant.Recorder_Always(recorder, false) }) {
		t.Fatal("constructing an Always(false) must not panic on its own")
	}
}

// Test_Dot_Product_Identity: an element is keyed by its caller location.
func Test_Dot_Product_Identity(t *testing.T) {
	recorder := &invariant.Recorder{
		Get_Caller: func(skip int) (file string, line int) { return "fixture.go", 42 },
	}
	element := invariant.Recorder_Sometimes(recorder, true)
	if element.Site != "fixture.go:42" {
		t.Fatalf("element Site = %q, want fixture.go:42", element.Site)
	}
}

// Test_Dot_Product_Grid: registration seeds the surviving cells and drops the carve.
func Test_Dot_Product_Grid(t *testing.T) {
	const source = `package fixture

func check(n int) {
	zero := invariant.Sometimes(n == 0)
	one := invariant.Sometimes(n == 1)
	invariant.Dot_Product(
		zero, one,
		invariant.Impossible(invariant.Event_True(zero), invariant.Event_True(one)),
	)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/check.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if _, ok := recorder.Assertions.Load("/fixture/check.go:6:tuple=(0,0)"); !ok {
		t.Error("the surviving (0,0) cell must be seeded")
	}
	if _, ok := recorder.Assertions.Load("/fixture/check.go:6:tuple=(1,1)"); ok {
		t.Error("the (1,1) cell carved by the Impossible must not be seeded")
	}
}

// Test_Dot_Product_Attribution: a panic names every axis violated on the call, not only
// the first.
func Test_Dot_Product_Attribution(t *testing.T) {
	recorder := new_test_recorder()
	first := invariant.Recorder_Always(recorder, false)
	second := invariant.Recorder_Always(recorder, false)
	message := recover_message(func() {
		invariant.Recorder_Dot_Product(recorder, first, second)
	})
	if !strings.Contains(message, first.Site) {
		t.Fatalf("the panic must name the first violated axis %q, got: %s",
			first.Site, message)
	}
	if !strings.Contains(message, second.Site) {
		t.Fatalf("the panic must name the second violated axis %q, got: %s",
			second.Site, message)
	}
}

// Test_Bundles_Descent: a bundle spread into a Dot_Product is followed and seeded.
func Test_Bundles_Descent(t *testing.T) {
	const source = `package fixture

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, invariant.Sometimes(n < 0))
}

func check(n int) {
	invariant.Dot_Product(Pair_Invariants(n)...)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/pair.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if _, ok := recorder.Assertions.Load(
		"/fixture/pair.go:8::from=/fixture/pair.go:4"); !ok {
		t.Error("the bundle element must be seeded under the Dot_Product callsite")
	}
}

// Test_Bundles_Composition: a nested bundle's element rolls up to the top-level
// Dot_Product callsite, not the inner bundle's call site.
func Test_Bundles_Composition(t *testing.T) {
	const source = `package fixture

func Inner_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, invariant.Sometimes(n < 0))
}

func Outer_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	dot_elements = append(dot_elements, invariant.Sometimes(n > 0))
	return append(dot_elements, Inner_Invariants(n)...)
}

func check(n int) {
	invariant.Dot_Product(Outer_Invariants(n)...)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/nested.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if _, ok := recorder.Assertions.Load(
		"/fixture/nested.go:13::from=/fixture/nested.go:4"); !ok {
		t.Error("the deeply nested element must attribute to the top-level callsite")
	}
}

// Test_Bundles_Binding: a bundle reached through a local binding, not a direct
// spread, is still descended.
func Test_Bundles_Binding(t *testing.T) {
	const source = `package fixture

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, invariant.Sometimes(n < 0))
}

func check(n int) {
	elements := Pair_Invariants(n)
	invariant.Dot_Product(elements...)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/bound.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if _, ok := recorder.Assertions.Load(
		"/fixture/bound.go:9::from=/fixture/bound.go:4"); !ok {
		t.Error("a bundle reached through a binding must be descended")
	}
}

// Test_Bundles_Casing: a snake_case _invariants bundle is recognized like the
// Ada_Case _Invariants form.
func Test_Bundles_Casing(t *testing.T) {
	const source = `package fixture

func pair_invariants(n int) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, invariant.Sometimes(n < 0))
}

func check(n int) {
	invariant.Dot_Product(pair_invariants(n)...)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/lower.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if _, ok := recorder.Assertions.Load(
		"/fixture/lower.go:8::from=/fixture/lower.go:4"); !ok {
		t.Error("a snake_case bundle must be recognized like the Ada_Case form")
	}
}

// Test_Bundles_Recorder_Form: a bundle element built with the explicit Recorder_
// constructor — which leads with the recorder argument — is descended like the bare
// sugar form, its condition read past that leading recorder.
func Test_Bundles_Recorder_Form(t *testing.T) {
	const source = `package fixture

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, invariant.Recorder_Sometimes(Default, n < 0))
}

func check(n int) {
	invariant.Dot_Product(Pair_Invariants(n)...)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/recorder_form.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	value, ok := recorder.Assertions.Load(
		"/fixture/recorder_form.go:8::from=/fixture/recorder_form.go:4")
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
	return append(dot_elements, Sometimes(n < 0))
}
`
	const application = `package app

import (
	invariant "example.com/m/invariant"
	"example.com/m/sugar"
)

func check(n int) {
	invariant.Dot_Product(sugar.Pair_Invariants(n)...)
}
`
	files := fstest.MapFS{
		"m/go.mod":         &fstest.MapFile{Data: []byte("module example.com/m\n")},
		"m/sugar/sugar.go": &fstest.MapFile{Data: []byte(sugar)},
		"m/app/app.go":     &fstest.MapFile{Data: []byte(application)},
	}
	const key = "/m/app/app.go:9::from=/m/sugar/sugar.go:4"

	recognized := &invariant.Recorder{File_System: files, Sugar_Package: "example.com/m/sugar"}
	invariant.Recorder_Register_Packages_For_Analysis(recognized, "/m/app")
	if _, ok := recognized.Assertions.Load(key); !ok {
		t.Error("a sugar-package bundle's unqualified Sometimes must be recognized")
	}

	ignored := &invariant.Recorder{File_System: files}
	invariant.Recorder_Register_Packages_For_Analysis(ignored, "/m/app")
	if _, ok := ignored.Assertions.Load(key); ok {
		t.Error("without Sugar_Package a bare call is not a primitive; nothing seeds")
	}
}

// Test_Bundles_Cross_Package: a bundle in a sibling package of the same module is
// resolved through the module path.
func Test_Bundles_Cross_Package(t *testing.T) {
	const package_a = `package a

import invariant "example.com/m/invariant"

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, invariant.Sometimes(n < 0))
}
`
	const package_b = `package b

import (
	invariant "example.com/m/invariant"
	"example.com/m/a"
)

func check(n int) {
	invariant.Dot_Product(a.Pair_Invariants(n)...)
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

	if _, ok := recorder.Assertions.Load("/m/b/b.go:9::from=/m/a/a.go:6"); !ok {
		t.Error("a bundle in a sibling package of the module must be resolved")
	}
}

// Test_Bundles_Workspace: a bundle in a sibling module joined by a go.work workspace
// is resolved through the workspace.
func Test_Bundles_Workspace(t *testing.T) {
	recorder := &invariant.Recorder{File_System: workspace_bundle_fixture()}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/work/b")

	if _, ok := recorder.Assertions.Load("b/b.go:9::from=a/a.go:6"); !ok {
		t.Error("a bundle in a sibling workspace module must be resolved")
	}
}

// Test_Bundles_Callsite: two callsites of one bundle are independent entries.
func Test_Bundles_Callsite(t *testing.T) {
	const source = `package fixture

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, invariant.Sometimes(n < 0))
}

func check_a(n int) {
	invariant.Dot_Product(Pair_Invariants(n)...)
}

func check_b(n int) {
	invariant.Dot_Product(Pair_Invariants(n)...)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/two.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if _, ok := recorder.Assertions.Load("/fixture/two.go:8::from=/fixture/two.go:4"); !ok {
		t.Error("callsite A must have its own coverage entry")
	}
	if _, ok := recorder.Assertions.Load("/fixture/two.go:12::from=/fixture/two.go:4"); !ok {
		t.Error("callsite B must have its own coverage entry")
	}
}

// Test_Bundles_Gap_Location: a composed bundle's coverage gap is named at the compound
// callsite::from=site — the top-level Dot_Product (compose.go:12) joined to the deepest
// nested element's site (compose.go:4) — while the cross-product gap names the callsite
// alone. The snapshot captures the whole report so both locations are pinned.
func Test_Bundles_Gap_Location(t *testing.T) {
	const source = `package fixture

func Inner_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, invariant.Always(n > 0))
}

func Outer_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, Inner_Invariants(n)...)
}

func check(n int) {
	invariant.Dot_Product(Outer_Invariants(n)...)
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

	snap.Expect(t, snap.Init(`🚨 2 coverage gaps 🚨

# Cross-product gaps
/fixture/compose.go:12  tuple (0) never observed

# Reachability gaps
/fixture/compose.go:12::from=/fixture/compose.go:4  Always — never reached: "n > 0"
🚨 2 coverage gaps 🚨
`), output.String())
}

// Test_Bundles_Failure_Location: a bundle element's assertion failure names only its own
// in-bundle site (here transfer.go:9), never the consuming callsite — yet the call site is
// not lost. The panic's Go stack still unwinds through Recorder_Dot_Product and the frame
// that spread the bundle, so the snapshot pins both the site-only message and the stack
// frames that carry the call site the message omits.
func Test_Bundles_Failure_Location(t *testing.T) {
	recorder := &invariant.Recorder{
		Get_Caller: func(skip int) (file string, line int) { return "transfer.go", 9 },
	}
	element := invariant.Recorder_Always(recorder, false)
	message, stack := recover_with_stack(func() {
		dot_product_callsite(recorder, element)
	})

	snap.Expect(
		t,
		snap.Init(`🚨 Assertion Failure 🚨: transfer.go:9  Always — condition was false

github.com/james-orcales/james-orcales/shared/invariant.Recorder_Dot_Product (invariant.go)
github.com/james-orcales/james-orcales/shared/invariant_test.dot_product_callsite (specification_test.go)`),
		message+"\n\n"+stack,
	)
}

// Test_Bundles_Ban: a Dot_Product inside a bundle fails registration.
func Test_Bundles_Ban(t *testing.T) {
	const source = `package fixture

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	invariant.Dot_Product(invariant.Sometimes(n < 0))
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

// Test_Bundles_Orphan: an axis that never reaches a Dot_Product fails registration.
func Test_Bundles_Orphan(t *testing.T) {
	const source = `package fixture

func check(n int) {
	invariant.Sometimes(n == 0)
}
`
	var output bytes.Buffer
	exit_code := -1
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/o.go": &fstest.MapFile{Data: []byte(source)},
		},
		Output: &output,
		Exit:   func(code int) { exit_code = code },
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if exit_code != 1 {
		t.Fatalf("an orphan axis must exit 1, got %d", exit_code)
	}
	if !strings.Contains(output.String(), "o.go:4") {
		t.Errorf("the orphan report must name the axis site, got: %s", output.String())
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
		Kind: invariant.Assertion_Kind_Sometimes, Site: "f.go:3", Condition: "n == 0",
	}
	gap.Frequency.Add(1) // true seen, false never: a gap
	recorder.Assertions.Store("f.go:3", gap)
	fired := &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Always, Site: "f.go:9", Condition: "x > 0",
	}
	fired.Frequency.Add(1) // reached: not a gap
	recorder.Assertions.Store("f.go:9", fired)

	invariant.Recorder_Analyze_Assertion_Frequency(recorder)

	if exit_code != 1 {
		t.Fatalf("a gap must exit 1, got %d", exit_code)
	}
	report := output.String()
	if !strings.Contains(report, "f.go:3") {
		t.Errorf("the gap must be named by its site, got: %s", report)
	}
	if !strings.Contains(report, "n == 0") {
		t.Errorf("the gap must be named by its condition, got: %s", report)
	}
	if strings.Contains(report, "f.go:9") {
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
		Kind: invariant.Assertion_Kind_Tuple, Site: "f.go:7", Tuple_Indices: []int{1, 0},
	}
	recorder.Assertions.Store("f.go:7:tuple=(1,0)", tuple)

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

// Test_Analysis_Boundary: a Distinct_Boundary endpoint never reached is reported as a
// boundary gap, named by the value it bounds.
func Test_Analysis_Boundary(t *testing.T) {
	var output bytes.Buffer
	exit_code := -1
	recorder := &invariant.Recorder{
		Is_Test: true, Output: &output, Exit: func(code int) { exit_code = code },
	}
	boundary := &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Boundary, Site: "b.go:5", Condition: "age",
	}
	boundary.Frequency.Add(1) // Hi endpoint seen; Lo (False_Frequency) never
	recorder.Assertions.Store("b.go:5", boundary)

	invariant.Recorder_Analyze_Assertion_Frequency(recorder)

	if exit_code != 1 {
		t.Fatalf("an unreached endpoint must exit 1, got %d", exit_code)
	}
	report := output.String()
	if !strings.Contains(report, "Lo endpoint never observed") {
		t.Errorf("the unreached Lo endpoint must be reported, got: %s", report)
	}
	if !strings.Contains(report, "age") {
		t.Errorf("the boundary must be named by the value it bounds, got: %s", report)
	}
}

// Test_Analysis_Summary: a clean run reports how many properties it tested, split into
// individual and combination with the panic-able subset counted.
func Test_Analysis_Summary(t *testing.T) {
	recorder := &invariant.Recorder{}
	store := func(key string, kind invariant.Assertion_Kind) {
		recorder.Assertions.Store(key, &invariant.Assertion_Metadata{Kind: kind})
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
		Kind: invariant.Assertion_Kind_Sometimes, Site: "f.go:3", Condition: "n == 0",
	}
	metadata.Frequency.Add(1)
	metadata.False_Frequency.Add(1)
	recorder.Assertions.Store("f.go:3", metadata)

	invariant.Recorder_Analyze_Assertion_Frequency(recorder)

	if exited {
		t.Error("a fully exercised run must not exit")
	}
	if output.Len() != 0 {
		t.Errorf("a fully exercised run must print nothing, got %q", output.String())
	}
}

// Test_Coverage_Modes: a fuzz run records no coverage, even with a seeded entry.
func Test_Coverage_Modes(t *testing.T) {
	recorder := new_test_recorder()
	recorder.Is_Test = true
	recorder.Is_Fuzz = true
	element := invariant.Recorder_Sometimes(recorder, true)
	metadata := &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Sometimes, Site: element.Site,
	}
	recorder.Assertions.Store(element.Site, metadata)

	invariant.Recorder_Dot_Product(recorder, element)

	if metadata.Frequency.Load() != 0 {
		t.Fatalf("a fuzz run must record no coverage, got Frequency %d",
			metadata.Frequency.Load())
	}
}

// Test_Coverage_Enforcement: a Dot_Product enforces assertions even when coverage is
// not being recorded, like a production binary.
func Test_Coverage_Enforcement(t *testing.T) {
	recorder := &invariant.Recorder{} // Is_Test false: not a coverage run
	if !did_panic(func() {
		invariant.Recorder_Dot_Product(recorder, invariant.Recorder_Always(recorder, false))
	}) {
		t.Fatal("a Dot_Product must enforce assertions even when coverage is not recorded")
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
	invariant.Dot_Product(x.Pair_Invariants(n)...)
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
func dot_product_callsite(recorder *invariant.Recorder, element invariant.Dot_Element) {
	invariant.Recorder_Dot_Product(recorder, element)
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
