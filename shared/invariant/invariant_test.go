package invariant_test

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/james-orcales/james-orcales/shared/invariant"
	"github.com/james-orcales/james-orcales/shared/snap/default"
)

// When an Impossible declares a combination of element events and that exact
// combination occurs in the call, Recorder_Dot_Product must fail.
func Test_Dot_Product_Impossible_Combination_Fails(t *testing.T) {
	recorder := new_test_recorder()
	a := invariant.Recorder_Sometimes(recorder, true, "a")
	b := invariant.Recorder_Sometimes(recorder, true, "b")
	forbidden := invariant.Impossible(
		invariant.Event_True("a"),
		invariant.Event_True("b"),
	)
	failed := false
	func() {
		defer func() {
			if recover() != nil {
				failed = true
			}
		}()
		invariant.Recorder_Dot_Product(recorder, "check", a, b, forbidden)
	}()
	if !failed {
		t.Fatal("Recorder_Dot_Product must fail when an Impossible combination occurs")
	}
}

// When the combination an Impossible forbids does NOT occur (one referenced
// event differs from what was observed), Recorder_Dot_Product must not fail.
func Test_Dot_Product_Impossible_Combination_Absent_Passes(t *testing.T) {
	recorder := new_test_recorder()
	a := invariant.Recorder_Sometimes(recorder, true, "a")
	b := invariant.Recorder_Sometimes(recorder, false, "b")
	forbidden := invariant.Impossible(
		invariant.Event_True("a"),
		invariant.Event_True("b"),
	)
	failed := false
	func() {
		defer func() {
			if recover() != nil {
				failed = true
			}
		}()
		invariant.Recorder_Dot_Product(recorder, "check", a, b, forbidden)
	}()
	if failed {
		t.Fatal("Recorder_Dot_Product must not fail when the " +
			"Impossible combination is absent")
	}
}

// A Distinct_Boundary whose X falls outside [Lo, Hi] fails when consumed by
// Recorder_Dot_Product — never at construction. Like every element producer, the
// constructor is inert on its own.
func Test_Dot_Product_Boundary_Outside_Bounds_Fails(t *testing.T) {
	recorder := new_test_recorder()
	outside_bounds := invariant.Recorder_Distinct_Boundary(
		recorder, &invariant.Boundary_Input[int]{X: 5, Lo: 0, Hi: 3}, "bounds",
	)
	failed := false
	func() {
		defer func() {
			if recover() != nil {
				failed = true
			}
		}()
		invariant.Recorder_Dot_Product(recorder, "check", outside_bounds)
	}()
	if !failed {
		t.Fatal("Recorder_Dot_Product must fail when a " +
			"Distinct_Boundary X is outside the bounds")
	}
}

// An interior X satisfies the bound, so Recorder_Dot_Product does not fail — even
// though an interior value contributes no endpoint coverage.
func Test_Dot_Product_Boundary_Within_Bounds_Passes(t *testing.T) {
	recorder := new_test_recorder()
	within_bounds := invariant.Recorder_Distinct_Boundary(
		recorder, &invariant.Boundary_Input[int]{X: 2, Lo: 0, Hi: 3}, "bounds",
	)
	failed := false
	func() {
		defer func() {
			if recover() != nil {
				failed = true
			}
		}()
		invariant.Recorder_Dot_Product(recorder, "check", within_bounds)
	}()
	if failed {
		t.Fatal("Recorder_Dot_Product must not fail when a " +
			"Distinct_Boundary X is within bounds")
	}
}

// Nonsensical bounds (Lo >= Hi) are a violation too, and like the rest they fire
// only at consumption — never from the bare constructor.
func Test_Dot_Product_Boundary_Bad_Bounds_Fails(t *testing.T) {
	recorder := new_test_recorder()
	bad_bounds := invariant.Recorder_Distinct_Boundary(
		recorder, &invariant.Boundary_Input[int]{X: 3, Lo: 3, Hi: 3}, "bounds",
	)
	failed := false
	func() {
		defer func() {
			if recover() != nil {
				failed = true
			}
		}()
		invariant.Recorder_Dot_Product(recorder, "check", bad_bounds)
	}()
	if !failed {
		t.Fatal("Recorder_Dot_Product must fail when a Distinct_Boundary has Lo >= Hi")
	}
}

// An element's identity is the author-supplied message it carries — the identity
// that static registration and the runtime rendezvous on, with no caller lookup.
func Test_Element_Message_Is_Identity(t *testing.T) {
	recorder := new_test_recorder()
	element := invariant.Recorder_Sometimes(recorder, true, "balance positive")
	if element.Message != "balance positive" {
		t.Fatalf("element Message = %q, want \"balance positive\"", element.Message)
	}
}

// Recorder_Dot_Product increments the seeded tracker entry for each observed
// element: Frequency on a true event, False_Frequency on false.
func Test_Dot_Product_Increments_Seeded_Element(t *testing.T) {
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
		t.Fatalf("Frequency = %d, want 1", metadata.Frequency.Load())
	}
	if metadata.False_Frequency.Load() != 0 {
		t.Fatalf("False_Frequency = %d, want 0", metadata.False_Frequency.Load())
	}
}

// An element consumed by a Dot_Product is credited under the compound key the
// prefix forms with the element's own message (prefix + separator + message), not
// under the bare message. The element names itself "zero"; consumed by the "check"
// Dot_Product its runtime key is "check␀zero", so the prefixed entry is credited and
// the bare "zero" entry — a different grid's identity — is left untouched.
func Test_Dot_Product_Credits_Element_Via_Prefixed_Key(t *testing.T) {
	recorder := new_test_recorder()
	recorder.Is_Test = true
	element := invariant.Recorder_Sometimes(recorder, true, "zero")
	prefixed_key := "check" + invariant.Element_Message_Separator + element.Message
	prefixed := &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Sometimes, Message: prefixed_key,
	}
	bare := &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Sometimes, Message: "zero",
	}
	recorder.Events.Store(prefixed_key, prefixed)
	recorder.Events.Store("zero", bare)
	invariant.Recorder_Dot_Product(recorder, "check", element)
	if prefixed.Frequency.Load() != 1 {
		t.Fatalf("prefixed Frequency = %d, want 1", prefixed.Frequency.Load())
	}
	if bare.Frequency.Load() != 0 {
		t.Fatalf("bare Frequency = %d, want 0 (the prefix namespaces the key)",
			bare.Frequency.Load())
	}
}

// Recorder_Dot_Product increments the seeded tuple entry for the observed
// combination of element events, keyed by the Dot_Product's message prefix. Both
// Sometimes elements fire true, so the observed tuple is (1, 1) under prefix "check".
func Test_Dot_Product_Increments_Seeded_Tuple(t *testing.T) {
	recorder := new_test_recorder()
	recorder.Is_Test = true
	a := invariant.Recorder_Sometimes(recorder, true, "a")
	b := invariant.Recorder_Sometimes(recorder, true, "b")
	tuple := &invariant.Assertion_Metadata{
		Kind:          invariant.Assertion_Kind_Tuple,
		Message:       "check",
		Tuple_Indices: []int{1, 1},
	}
	recorder.Events.Store("check:tuple=(1,1)", tuple)
	invariant.Recorder_Dot_Product(recorder, "check", a, b)
	if tuple.Frequency.Load() != 1 {
		t.Fatalf("tuple Frequency = %d, want 1", tuple.Frequency.Load())
	}
}

// A Distinct_Boundary increments its seeded endpoint counters: the Hi endpoint
// bumps Frequency, the Lo endpoint bumps False_Frequency, and an interior value
// (in range but at neither endpoint) bumps neither.
func Test_Dot_Product_Boundary_Tracks_Endpoints(t *testing.T) {
	cases := []struct {
		Name           string
		X              int
		Want_Frequency int64
		Want_False     int64
	}{
		{"Hi endpoint", 3, 1, 0},
		{"Lo endpoint", 0, 0, 1},
		{"interior", 2, 0, 0},
	}
	for _, c := range cases {
		recorder := new_test_recorder()
		recorder.Is_Test = true
		element := invariant.Recorder_Distinct_Boundary(
			recorder, &invariant.Boundary_Input[int]{X: c.X, Lo: 0, Hi: 3}, "range",
		)
		key := "check" + invariant.Element_Message_Separator + element.Message
		metadata := &invariant.Assertion_Metadata{
			Kind: invariant.Assertion_Kind_Boundary, Message: key,
		}
		recorder.Events.Store(key, metadata)
		invariant.Recorder_Dot_Product(recorder, "check", element)
		if metadata.Frequency.Load() != c.Want_Frequency {
			t.Errorf("%s: Frequency = %d, want %d",
				c.Name, metadata.Frequency.Load(), c.Want_Frequency)
		}
		if metadata.False_Frequency.Load() != c.Want_False {
			t.Errorf("%s: False_Frequency = %d, want %d",
				c.Name, metadata.False_Frequency.Load(), c.Want_False)
		}
	}
}

// Registration parses an invariant.Dot_Product over inline elements and seeds
// one entry per element plus the full bucket grid minus the tuples an Impossible
// carves out. zero and one are Sometimes; the Dot_Product's prefix "check" forms
// each element key and the tuple keys; the Impossible forbids the (true, true) cell
// = tuple (1,1).
func Test_Register_Inline_Dot_Product_Seeds_Grid_Minus_Carves(t *testing.T) {
	const source = `package fixture

func check(n int) {
	zero := invariant.Sometimes(n == 0, "zero")
	one := invariant.Sometimes(n == 1, "one")
	invariant.Dot_Product("check",
		zero, one,
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

	if _, ok := recorder.Events.Load(
		"check" + invariant.Element_Message_Separator + "zero"); !ok {
		t.Error("expected a per-element entry seeded for the zero Sometimes")
	}
	if _, ok := recorder.Events.Load("check:tuple=(0,0)"); !ok {
		t.Error("expected the surviving tuple (0,0) seeded under the Dot_Product prefix")
	}
	if _, ok := recorder.Events.Load("check:tuple=(1,1)"); ok {
		t.Error("tuple (1,1) is carved by the Impossible; it must not be seeded")
	}
}

// A Dot_Product's message prefix composes into each held axis's coverage key: the
// element names itself "zero", the Dot_Product carries prefix "p", and the seeded
// per-element key is the prefix joined to the local message by the separator ("p␀zero").
// This is the identity the runtime rebuilds, so registration and runtime rendezvous on it.
func Test_Register_Prefix_Composes_Into_Element_Keys(t *testing.T) {
	const source = `package fixture

func check(n int) {
	invariant.Dot_Product("p", invariant.Sometimes(n == 0, "zero"))
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/check.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if _, ok := recorder.Events.Load(
		"p" + invariant.Element_Message_Separator + "zero"); !ok {
		t.Error("expected element keyed by prefix joined to its local message (p␀zero)")
	}
	if _, ok := recorder.Events.Load("zero"); ok {
		t.Error("the element must not be seeded under its bare local message")
	}
}

// Two Dot_Products with DISTINCT prefixes ("a" and "b") spreading the same inline element
// keep independent coverage entries — the prefix namespaces each axis apart, so neither
// masks the other's gap. (Reusing one prefix across two Dot_Products is instead a fatal
// duplicate, covered by Test_Register_Fatal_On_Duplicate_Message.
func Test_Register_Two_Distinct_Prefixes_Keep_Independent_Entries(t *testing.T) {
	const source = `package fixture

func check_a(n int) {
	invariant.Dot_Product("a", invariant.Sometimes(n == 0, "zero"))
}

func check_b(n int) {
	invariant.Dot_Product("b", invariant.Sometimes(n == 0, "zero"))
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/check.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if _, ok := recorder.Events.Load(
		"a" + invariant.Element_Message_Separator + "zero"); !ok {
		t.Error("expected an independent entry under prefix a (a␀zero)")
	}
	if _, ok := recorder.Events.Load(
		"b" + invariant.Element_Message_Separator + "zero"); !ok {
		t.Error("expected an independent entry under prefix b (b␀zero)")
	}
}

// A message claimed by two distinct assertions is a global-identity collision and fails
// registration, never silently merging two obligations into one (which would mask a gap).
// Here two Dot_Products share the prefix "dup"; registration prints the duplicate banner
// and exits 1.
func Test_Register_Fatal_On_Duplicate_Message(t *testing.T) {
	const source = `package fixture

func check_a(n int) {
	invariant.Dot_Product("dup", invariant.Sometimes(n == 0, "zero"))
}

func check_b(n int) {
	invariant.Dot_Product("dup", invariant.Sometimes(n == 1, "one"))
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
		t.Fatalf("a duplicate message must exit 1, got %d", exit_code)
	}
	if !strings.Contains(output.String(), "duplicate messages") {
		t.Errorf("the report must carry the duplicate banner, got: %s", output.String())
	}
}

// A message that is not a compile-time string literal — a variable or a concatenation —
// cannot be statically keyed under the same key the runtime emits, so its coverage would
// vanish. Registration refuses it: it prints the non-literal banner and exits 1. Here the
// Dot_Product prefix is a variable rather than a literal.
func Test_Register_Fatal_On_Non_Literal_Message(t *testing.T) {
	const source = `package fixture

func check(n int, prefix string) {
	invariant.Dot_Product(prefix, invariant.Sometimes(n == 0, "zero"))
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
	if !strings.Contains(output.String(), "non-literal messages") {
		t.Errorf("the report must carry the non-literal banner, got: %s", output.String())
	}
}

// Registration recognizes a Distinct_Boundary and seeds a two-bucket grid (Lo=0,
// Hi=1) keyed by the Dot_Product prefix "age check", plus a per-element entry keyed
// by that prefix joined to the boundary's own message "age", whose Condition is the
// X expression source.
func Test_Register_Distinct_Boundary_Seeds_Endpoints(t *testing.T) {
	const source = `package fixture

func check(age int) {
	invariant.Dot_Product("age check",
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{X: age, Lo: 22, Hi: 34}, "age"),
	)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/b.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	value, found := recorder.Events.Load(
		"age check" + invariant.Element_Message_Separator + "age")
	if !found {
		t.Fatal("expected a per-element entry seeded for the Distinct_Boundary")
	}
	metadata := value.(*invariant.Assertion_Metadata)
	if metadata.Kind != invariant.Assertion_Kind_Boundary {
		t.Errorf("per-element Kind = %d, want Assertion_Kind_Boundary", metadata.Kind)
	}
	if metadata.Condition != "age" {
		t.Errorf("per-element Condition = %q, want age", metadata.Condition)
	}
	if _, ok := recorder.Events.Load("age check:tuple=(0)"); !ok {
		t.Error("expected the Lo endpoint tuple (0) seeded under the Dot_Product prefix")
	}
	if _, ok := recorder.Events.Load("age check:tuple=(1)"); !ok {
		t.Error("expected the Hi endpoint tuple (1) seeded under the Dot_Product prefix")
	}
}

// Registration descends a *_Invariants bundle invoked inside a Dot_Product: each
// bundle element is keyed by the Dot_Product's prefix joined to the element's own
// message ("field␀lo"), so each prefix tracks the bundle separately; the tuple grid
// is keyed by the prefix ("field"), and the bundle's Impossible carves the (true,
// true) tuple (1,1).
func Test_Register_Descends_Bundle(t *testing.T) {
	const source = `package fixture

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	lo := invariant.Sometimes(n < 0, "lo")
	hi := invariant.Sometimes(n > 0, "hi")
	dot_elements = append(
		dot_elements,
		lo, hi,
		invariant.Impossible(invariant.Event_True("lo"), invariant.Event_True("hi")),
	)
	return dot_elements
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
		t.Error("expected lo keyed by prefix joined to its own message (field␀lo)")
	}
	if _, ok := recorder.Events.Load("lo"); ok {
		t.Error("lo must not be seeded under its bare message; the prefix qualifies it")
	}
	if _, ok := recorder.Events.Load("field:tuple=(0,0)"); !ok {
		t.Error("expected the surviving tuple (0,0) under the Dot_Product prefix")
	}
	if _, ok := recorder.Events.Load("field:tuple=(1,1)"); ok {
		t.Error("tuple (1,1) is carved by the bundle's Impossible; must not be seeded")
	}
}

// A *_Invariants bundle that composes another bundle (Outer appends Inner's
// elements) attributes EVERY descended element — however deep — to the one
// top-level Dot_Product prefix ("field"), each keyed by its own message, not under
// a bare message. This is what lets types compose invariants while keeping each
// top-level prefix's credit independent.
func Test_Register_Nested_Bundle_Attributes_To_Top_Level_Caller(t *testing.T) {
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
		t.Error("Inner Sometimes must key under the top-level prefix (field␀inner)")
	}
	if _, ok := recorder.Events.Load(
		"field" + invariant.Element_Message_Separator + "outer"); !ok {
		t.Error("Outer Sometimes must key under the top-level prefix (field␀outer)")
	}
	if _, ok := recorder.Events.Load("inner"); ok {
		t.Error("Inner's element must not be seeded under its bare message")
	}
}

// One bundle spread into two Dot_Products with DISTINCT prefixes ("a" and "b") seeds
// two distinct per-element entries, one per prefix — so a prefix covering only the
// true branch cannot mask a sibling prefix that covered only the false branch. The
// per-field prefix is what keeps them apart; the bundle's lone Sometimes ("lo")
// yields "a␀lo" and "b␀lo", and no shared bare entry.
func Test_Register_Two_Prefixes_Of_One_Bundle_Yield_Distinct_Entries(t *testing.T) {
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
		t.Error("expected a distinct entry for prefix a (a␀lo)")
	}
	if _, ok := recorder.Events.Load(
		"b" + invariant.Element_Message_Separator + "lo"); !ok {
		t.Error("expected a distinct entry for prefix b (b␀lo)")
	}
	if _, ok := recorder.Events.Load("lo"); ok {
		t.Error("the two prefixes must not share a bare per-element entry")
	}
}

// Registration follows a bundle reached through a single-assignment binding before
// the spread (elems := Pair_Invariants(n); Dot_Product("field", elems...)), not only
// the direct Dot_Product("field", Pair_Invariants(n)...) form. The element keys under
// the Dot_Product prefix joined to its own message (field␀lo).
func Test_Register_Bound_Variable_Bundle_Is_Descended(t *testing.T) {
	const source = `package fixture

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, invariant.Sometimes(n < 0, "lo"))
}

func check(n int) {
	elems := Pair_Invariants(n)
	invariant.Dot_Product("field", elems...)
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
		t.Error("expected the bound bundle keyed under the Dot_Product prefix (field␀lo)")
	}
	if _, ok := recorder.Events.Load("lo"); ok {
		t.Error("the bound bundle's element must not be seeded under its bare message")
	}
}

// Registration follows a bundle appended into the spread variable
// (elems = append(elems, Pair_Invariants(n)...); Dot_Product("field", elems...)),
// mirroring how a bundle body's own appends are read. The element keys under the
// Dot_Product prefix joined to its own message (field␀lo).
func Test_Register_Appended_Bundle_Is_Descended(t *testing.T) {
	const source = `package fixture

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, invariant.Sometimes(n < 0, "lo"))
}

func check(n int) {
	var elems []invariant.Dot_Element
	elems = append(elems, Pair_Invariants(n)...)
	invariant.Dot_Product("field", elems...)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/append.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if _, ok := recorder.Events.Load(
		"field" + invariant.Element_Message_Separator + "lo"); !ok {
		t.Error("appended bundle must key under the Dot_Product prefix (field␀lo)")
	}
	if _, ok := recorder.Events.Load("lo"); ok {
		t.Error("the appended bundle's element must not be seeded under its bare message")
	}
}

// Recorder_Analyze_Assertion_Frequency reports every never-fired assertion and
// every Sometimes whose false branch never fired, naming each by site and
// condition source, then exits non-zero. A fully exercised assertion is silent.
func Test_Analyze_Reports_Never_Fired_And_Exits(t *testing.T) {
	var output bytes.Buffer
	exit_code := -1
	recorder := &invariant.Recorder{
		Is_Test: true,
		Output:  &output,
		Exit:    func(code int) { exit_code = code },
	}
	// A Sometimes seen only true (its false branch never observed) and a tuple
	// that never occurred are two gaps; a fully-fired Always is not a gap.
	sometimes := &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Sometimes, Message: "zero", Condition: "n == 0",
	}
	sometimes.Frequency.Add(1)
	recorder.Events.Store("zero", sometimes)
	tuple := &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Tuple, Message: "grid", Tuple_Indices: []int{1, 0},
	}
	recorder.Events.Store("grid:tuple=(1,0)", tuple)
	reached := &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Always, Message: "positive", Condition: "x > 0",
	}
	reached.Frequency.Add(1)
	recorder.Events.Store("positive", reached)

	invariant.Recorder_Analyze_Assertion_Frequency(recorder)

	if exit_code != 1 {
		t.Fatalf("Exit code = %d, want 1", exit_code)
	}
	report := output.String()
	if !strings.Contains(report, "zero") {
		t.Error("report must name the Sometimes false-branch gap by its message")
	}
	if !strings.Contains(report, "n == 0") {
		t.Error("report must include the Sometimes condition source")
	}
	if !strings.Contains(report, "grid") {
		t.Error("report must name the never-occurring tuple by its message")
	}
	if strings.Contains(report, "positive") {
		t.Error("a fully-fired Always must not be reported")
	}
}

// A Distinct_Boundary whose Lo endpoint was never observed is a coverage gap:
// Recorder_Analyze_Assertion_Frequency reports it by its X condition and exits.
func Test_Analyze_Reports_Boundary_Endpoint_Gap(t *testing.T) {
	var output bytes.Buffer
	exit_code := -1
	recorder := &invariant.Recorder{
		Is_Test: true,
		Output:  &output,
		Exit:    func(code int) { exit_code = code },
	}
	// Hi endpoint observed (Frequency), Lo never (False_Frequency == 0): one gap.
	boundary := &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Boundary, Message: "age", Condition: "age",
	}
	boundary.Frequency.Add(1)
	recorder.Events.Store("age", boundary)

	invariant.Recorder_Analyze_Assertion_Frequency(recorder)

	if exit_code != 1 {
		t.Fatalf("Exit code = %d, want 1", exit_code)
	}
	report := output.String()
	if !strings.Contains(report, "Lo endpoint never observed") {
		t.Errorf("report must flag the unobserved Lo endpoint, got: %s", report)
	}
	if !strings.Contains(report, "age") {
		t.Error("report must name the boundary by its X condition")
	}
}

// With every assertion fully exercised, Recorder_Analyze_Assertion_Frequency
// reports nothing and does not exit.
func Test_Analyze_Clean_Run_Is_Silent(t *testing.T) {
	var output bytes.Buffer
	exited := false
	recorder := &invariant.Recorder{
		Is_Test: true,
		Output:  &output,
		Exit:    func(code int) { exited = true },
	}
	sometimes := &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Sometimes, Message: "zero", Condition: "n == 0",
	}
	sometimes.Frequency.Add(1)
	sometimes.False_Frequency.Add(1)
	recorder.Events.Store("zero", sometimes)

	invariant.Recorder_Analyze_Assertion_Frequency(recorder)

	if exited {
		t.Error("a clean run must not call Exit")
	}
	if output.Len() != 0 {
		t.Errorf("a clean run must print nothing, got %q", output.String())
	}
}

// Two prefixes namespacing one bundle element ("a␀lo" and "b␀lo", same bundle
// Sometimes "lo") are independent coverage entries: prefix A saw only true and
// prefix B only false, so Recorder_Analyze_Assertion_Frequency reports A's missing
// false branch AND B's missing true branch — neither prefix masks the other's gap,
// the per-field prefix is what keeps them apart.
func Test_Analyze_Complementary_Coverage_Reports_Each_Prefix_Gap(t *testing.T) {
	var output bytes.Buffer
	exit_code := -1
	recorder := &invariant.Recorder{
		Is_Test: true,
		Output:  &output,
		Exit:    func(code int) { exit_code = code },
	}
	key_a := "a" + invariant.Element_Message_Separator + "lo"
	prefix_a := &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Sometimes, Message: key_a, Condition: "n < 0",
	}
	prefix_a.Frequency.Add(1) // saw true; false branch never observed
	recorder.Events.Store(key_a, prefix_a)
	key_b := "b" + invariant.Element_Message_Separator + "lo"
	prefix_b := &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Sometimes, Message: key_b, Condition: "n < 0",
	}
	prefix_b.False_Frequency.Add(1) // saw false; true branch never observed
	recorder.Events.Store(key_b, prefix_b)

	invariant.Recorder_Analyze_Assertion_Frequency(recorder)

	if exit_code != 1 {
		t.Fatalf("Exit code = %d, want 1", exit_code)
	}
	report := output.String()
	// Message_display renders the NUL separator as " · ", so the report shows "a · lo".
	if !strings.Contains(report, "a · lo") {
		t.Error("report must name prefix A's gap by its prefixed message")
	}
	if !strings.Contains(report, "b · lo") {
		t.Error("report must name prefix B's gap by its prefixed message")
	}
	if !strings.Contains(report, "false branch never observed") {
		t.Error("prefix A must report its unobserved false branch")
	}
	if !strings.Contains(report, "true branch never observed") {
		t.Error("prefix B must report its unobserved true branch")
	}
}

// Recorder_Assertion_Summary counts the seeded assertions for the clean-run
// banner: per-element entries (Always, Sometimes) are individual properties,
// Tuple entries are combinations, and the Always family is panic-able.
func Test_Assertion_Summary_Counts_Properties(t *testing.T) {
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

// Registration resolves a *_Invariants bundle defined in another same-module
// package: package b's Dot_Product spreads a.Pair_Invariants, and the module's
// go.mod maps the import path "example.com/m/a" to its directory so the bundle's
// declaration (in a.go) is found, descended, and its grid seeded.
func Test_Register_Resolves_Cross_Package_Bundle(t *testing.T) {
	const package_a = `package a

import invariant "example.com/m/invariant"

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	lo := invariant.Sometimes(n < 0, "lo")
	hi := invariant.Sometimes(n > 0, "hi")
	return append(dot_elements, lo, hi,
		invariant.Impossible(invariant.Event_True("lo"), invariant.Event_True("hi")))
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
		t.Error("expected lo keyed under prefix field (field␀lo)")
	}
	if _, ok := recorder.Events.Load("lo"); ok {
		t.Error("lo must not be seeded under its bare cross-package message")
	}
	if _, ok := recorder.Events.Load("field:tuple=(0,0)"); !ok {
		t.Error("expected the surviving tuple (0,0) under b.go's Dot_Product prefix")
	}
	if _, ok := recorder.Events.Load("field:tuple=(1,1)"); ok {
		t.Error("tuple (1,1) is carved by the bundle's Impossible; must not be seeded")
	}
}

// Registration descends a generic *_Invariants bundle that returns a single appended
// Distinct_Boundary over its value parameter — the shape Whole_Number_Invariants takes —
// recognizing it cross-package as a Boundary axis keyed by the prefix joined to its own
// message (field␀range), with the Lo/Hi tuple grid seeded under the caller's Dot_Product
// prefix. Guards that the Sometimes→Boundary preset rewrite stays statically recognized,
// generic Boundary_Input[I] and all.
func Test_Register_Resolves_Generic_Boundary_Bundle(t *testing.T) {
	const package_a = `package a

import invariant "example.com/m/invariant"

func Integer_Invariants[I invariant.Numeric](n I) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements,
		invariant.Distinct_Boundary(&invariant.Boundary_Input[I]{X: n, Lo: 0, Hi: 9}, "range"))
}
`
	const package_b = `package b

import (
	invariant "example.com/m/invariant"
	"example.com/m/a"
)

func check(n int) {
	invariant.Dot_Product("field", a.Integer_Invariants(n)...)
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

	value, found := recorder.Events.Load(
		"field" + invariant.Element_Message_Separator + "range")
	if !found {
		t.Fatal("expected the boundary keyed under prefix field (field␀range)")
	}
	metadata := value.(*invariant.Assertion_Metadata)
	if metadata.Kind != invariant.Assertion_Kind_Boundary {
		t.Errorf("Kind = %d, want Assertion_Kind_Boundary", metadata.Kind)
	}
	if metadata.Condition != "n" {
		t.Errorf("Condition = %q, want n", metadata.Condition)
	}
	if _, ok := recorder.Events.Load("field:tuple=(0)"); !ok {
		t.Error("expected the Lo endpoint tuple (0) under b.go's Dot_Product prefix")
	}
	if _, ok := recorder.Events.Load("field:tuple=(1)"); !ok {
		t.Error("expected the Hi endpoint tuple (1) under b.go's Dot_Product prefix")
	}
}

// Renders an Assertion_Kind as the short name the registration snapshot reads by.
func kind_name(kind invariant.Assertion_Kind) (name string) {
	switch kind {
	case invariant.Assertion_Kind_Always:
		return "Always"
	case invariant.Assertion_Kind_Sometimes:
		return "Sometimes"
	case invariant.Assertion_Kind_Boundary:
		return "Boundary"
	case invariant.Assertion_Kind_Tuple:
		return "Tuple"
	}
	return "Unknown"
}

// Renders every seeded assertion as one sorted "kind key detail" line so a bundle's whole
// registration footprint compares against a golden as a single string. The map key (not
// the metadata Message) is the identity: a bundle element's key is the prefix joined to
// its own message by Element_Message_Separator, a tuple's is the prefix plus its bucket
// combination. The NUL separator is rendered as " · " so the golden stays readable text.
func snapshot_registered(recorder *invariant.Recorder) (snapshot string) {
	var lines []string
	recorder.Events.Range(func(key, value any) (more bool) {
		metadata := value.(*invariant.Assertion_Metadata)
		detail := metadata.Condition
		if metadata.Kind == invariant.Assertion_Kind_Tuple {
			detail = fmt.Sprint(metadata.Tuple_Indices)
		}
		display := strings.ReplaceAll(
			key.(string), invariant.Element_Message_Separator, " · ")
		line := fmt.Sprintf("%s %s %s", kind_name(metadata.Kind), display, detail)
		lines = append(lines, line)
		return true
	})
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

// Registration descends the Whole_Number_Invariants bundle — three Sometimes axes
// (n == 0/1/2) and a Distinct_Boundary over the type's range — exactly as the preset
// builds it: with the explicit Recorder_Sometimes / Recorder_Distinct_Boundary
// constructors, whose condition rides the second argument and whose message rides the
// argument past it. The snapshot pins the whole footprint: one per-element entry per axis
// (keyed prefix · own-message) plus the full 2^4 tuple grid (no Impossible carves), every
// entry attributed to the caller's single Dot_Product prefix "field".
func Test_Register_Snapshots_Whole_Number_Invariants(t *testing.T) {
	const package_a = `package a

import invariant "example.com/m/invariant"

func Whole_Number_Invariants[I invariant.Numeric](n I) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements,
		invariant.Recorder_Sometimes(Default, n == 0, "zero"),
		invariant.Recorder_Sometimes(Default, n == 1, "one"),
		invariant.Recorder_Sometimes(Default, n == 2, "two"),
		invariant.Recorder_Distinct_Boundary(Default, &invariant.Boundary_Input[I]{
			X:  n,
			Lo: whole_number_min[I](),
			Hi: whole_number_max[I](),
		}, "range"),
	)
}
`
	const package_b = `package b

import (
	invariant "example.com/m/invariant"
	"example.com/m/a"
)

func check(n int) {
	invariant.Dot_Product("field", a.Whole_Number_Invariants(n)...)
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

	snap.Expect(t, snap.Init(`Boundary field · range n
Sometimes field · one n == 1
Sometimes field · two n == 2
Sometimes field · zero n == 0
Tuple field:tuple=(0,0,0,0) [0 0 0 0]
Tuple field:tuple=(0,0,0,1) [0 0 0 1]
Tuple field:tuple=(0,0,1,0) [0 0 1 0]
Tuple field:tuple=(0,0,1,1) [0 0 1 1]
Tuple field:tuple=(0,1,0,0) [0 1 0 0]
Tuple field:tuple=(0,1,0,1) [0 1 0 1]
Tuple field:tuple=(0,1,1,0) [0 1 1 0]
Tuple field:tuple=(0,1,1,1) [0 1 1 1]
Tuple field:tuple=(1,0,0,0) [1 0 0 0]
Tuple field:tuple=(1,0,0,1) [1 0 0 1]
Tuple field:tuple=(1,0,1,0) [1 0 1 0]
Tuple field:tuple=(1,0,1,1) [1 0 1 1]
Tuple field:tuple=(1,1,0,0) [1 1 0 0]
Tuple field:tuple=(1,1,0,1) [1 1 0 1]
Tuple field:tuple=(1,1,1,0) [1 1 1 0]
Tuple field:tuple=(1,1,1,1) [1 1 1 1]`), snapshot_registered(recorder))
}

// Registration descends the Float_Invariants bundle — three Sometimes axes (NaN,
// negative, positive), each bound to a local and built with the unqualified sugar
// primitive carrying its own message, plus three pairwise Impossibles. The snapshot pins
// the footprint: one per-element entry per axis (keyed prefix · own-message) and only the
// tuples the three mutual-exclusions leave standing — every co-true pair is carved, so of
// the 2^3 grid just the four with at most one axis true survive.
func Test_Register_Snapshots_Float_Invariants(t *testing.T) {
	const sugar = `package sugar

func Float_Invariants[F ~float32 | ~float64](f F) (dot_elements []invariant.Dot_Element) {
	value := float64(f)
	not_a_number := Sometimes(math.IsNaN(value), "nan")
	negative := Sometimes(value < 0, "negative")
	positive := Sometimes(value > 0, "positive")
	return append(dot_elements,
		not_a_number, negative, positive,
		Impossible(Event_True("nan"), Event_True("negative")),
		Impossible(Event_True("nan"), Event_True("positive")),
		Impossible(Event_True("negative"), Event_True("positive")),
	)
}
`
	const application = `package app

import (
	invariant "example.com/m/invariant"
	"example.com/m/sugar"
)

func check(f float64) {
	invariant.Dot_Product("field", sugar.Float_Invariants(f)...)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"m/go.mod":         &fstest.MapFile{Data: []byte("module example.com/m\n")},
			"m/sugar/sugar.go": &fstest.MapFile{Data: []byte(sugar)},
			"m/app/app.go":     &fstest.MapFile{Data: []byte(application)},
		},
		Output:        &bytes.Buffer{},
		Sugar_Package: "example.com/m/sugar",
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/m/app")

	snap.Expect(t, snap.Init(`Sometimes field · nan math.IsNaN(value)
Sometimes field · negative value < 0
Sometimes field · positive value > 0
Tuple field:tuple=(0,0,0) [0 0 0]
Tuple field:tuple=(0,0,1) [0 0 1]
Tuple field:tuple=(0,1,0) [0 1 0]
Tuple field:tuple=(1,0,0) [1 0 0]`), snapshot_registered(recorder))
}

// The String_Invariants footprint is hoisted out of its test so the test body stays
// within the function-length budget; the 8 axes x 9 carves leave 81 of the 2^8 cells.
const string_invariants = `Sometimes field · Sometimes_Has_Control Sometimes_Has_Control(s)
Sometimes field · Sometimes_Has_Edge_Whitespace Sometimes_Has_Edge_Whitespace(s)
Sometimes field · Sometimes_Has_Interior_Whitespace Sometimes_Has_Interior_Whitespace(s)
Sometimes field · Sometimes_Has_Invalid_UTF8 Sometimes_Has_Invalid_UTF8(s)
Sometimes field · Sometimes_Has_Line_Break Sometimes_Has_Line_Break(s)
Sometimes field · Sometimes_Has_Multibyte_Rune Sometimes_Has_Multibyte_Rune(s)
Sometimes field · Sometimes_Has_Nul Sometimes_Has_Nul(s)
Sometimes field · empty len(s) == 0
Tuple field:tuple=(0,0,0,0,0,0,0,0) [0 0 0 0 0 0 0 0]
Tuple field:tuple=(0,0,0,0,0,0,1,0) [0 0 0 0 0 0 1 0]
Tuple field:tuple=(0,0,0,0,0,0,1,1) [0 0 0 0 0 0 1 1]
Tuple field:tuple=(0,0,0,0,0,1,0,0) [0 0 0 0 0 1 0 0]
Tuple field:tuple=(0,0,0,0,0,1,1,0) [0 0 0 0 0 1 1 0]
Tuple field:tuple=(0,0,0,0,0,1,1,1) [0 0 0 0 0 1 1 1]
Tuple field:tuple=(0,0,0,0,1,0,1,0) [0 0 0 0 1 0 1 0]
Tuple field:tuple=(0,0,0,0,1,0,1,1) [0 0 0 0 1 0 1 1]
Tuple field:tuple=(0,0,0,0,1,1,1,0) [0 0 0 0 1 1 1 0]
Tuple field:tuple=(0,0,0,0,1,1,1,1) [0 0 0 0 1 1 1 1]
Tuple field:tuple=(0,0,0,1,0,0,0,0) [0 0 0 1 0 0 0 0]
Tuple field:tuple=(0,0,0,1,0,0,1,0) [0 0 0 1 0 0 1 0]
Tuple field:tuple=(0,0,0,1,0,0,1,1) [0 0 0 1 0 0 1 1]
Tuple field:tuple=(0,0,0,1,0,1,0,0) [0 0 0 1 0 1 0 0]
Tuple field:tuple=(0,0,0,1,0,1,1,0) [0 0 0 1 0 1 1 0]
Tuple field:tuple=(0,0,0,1,0,1,1,1) [0 0 0 1 0 1 1 1]
Tuple field:tuple=(0,0,0,1,1,0,1,0) [0 0 0 1 1 0 1 0]
Tuple field:tuple=(0,0,0,1,1,0,1,1) [0 0 0 1 1 0 1 1]
Tuple field:tuple=(0,0,0,1,1,1,1,0) [0 0 0 1 1 1 1 0]
Tuple field:tuple=(0,0,0,1,1,1,1,1) [0 0 0 1 1 1 1 1]
Tuple field:tuple=(0,0,1,0,0,0,0,0) [0 0 1 0 0 0 0 0]
Tuple field:tuple=(0,0,1,0,0,0,1,0) [0 0 1 0 0 0 1 0]
Tuple field:tuple=(0,0,1,0,0,0,1,1) [0 0 1 0 0 0 1 1]
Tuple field:tuple=(0,0,1,0,0,1,0,0) [0 0 1 0 0 1 0 0]
Tuple field:tuple=(0,0,1,0,0,1,1,0) [0 0 1 0 0 1 1 0]
Tuple field:tuple=(0,0,1,0,0,1,1,1) [0 0 1 0 0 1 1 1]
Tuple field:tuple=(0,0,1,0,1,0,1,0) [0 0 1 0 1 0 1 0]
Tuple field:tuple=(0,0,1,0,1,0,1,1) [0 0 1 0 1 0 1 1]
Tuple field:tuple=(0,0,1,0,1,1,1,0) [0 0 1 0 1 1 1 0]
Tuple field:tuple=(0,0,1,0,1,1,1,1) [0 0 1 0 1 1 1 1]
Tuple field:tuple=(0,0,1,1,0,0,0,0) [0 0 1 1 0 0 0 0]
Tuple field:tuple=(0,0,1,1,0,0,1,0) [0 0 1 1 0 0 1 0]
Tuple field:tuple=(0,0,1,1,0,0,1,1) [0 0 1 1 0 0 1 1]
Tuple field:tuple=(0,0,1,1,0,1,0,0) [0 0 1 1 0 1 0 0]
Tuple field:tuple=(0,0,1,1,0,1,1,0) [0 0 1 1 0 1 1 0]
Tuple field:tuple=(0,0,1,1,0,1,1,1) [0 0 1 1 0 1 1 1]
Tuple field:tuple=(0,0,1,1,1,0,1,0) [0 0 1 1 1 0 1 0]
Tuple field:tuple=(0,0,1,1,1,0,1,1) [0 0 1 1 1 0 1 1]
Tuple field:tuple=(0,0,1,1,1,1,1,0) [0 0 1 1 1 1 1 0]
Tuple field:tuple=(0,0,1,1,1,1,1,1) [0 0 1 1 1 1 1 1]
Tuple field:tuple=(0,1,0,0,0,0,0,0) [0 1 0 0 0 0 0 0]
Tuple field:tuple=(0,1,0,0,0,0,1,0) [0 1 0 0 0 0 1 0]
Tuple field:tuple=(0,1,0,0,0,0,1,1) [0 1 0 0 0 0 1 1]
Tuple field:tuple=(0,1,0,0,0,1,0,0) [0 1 0 0 0 1 0 0]
Tuple field:tuple=(0,1,0,0,0,1,1,0) [0 1 0 0 0 1 1 0]
Tuple field:tuple=(0,1,0,0,0,1,1,1) [0 1 0 0 0 1 1 1]
Tuple field:tuple=(0,1,0,0,1,0,1,0) [0 1 0 0 1 0 1 0]
Tuple field:tuple=(0,1,0,0,1,0,1,1) [0 1 0 0 1 0 1 1]
Tuple field:tuple=(0,1,0,0,1,1,1,0) [0 1 0 0 1 1 1 0]
Tuple field:tuple=(0,1,0,0,1,1,1,1) [0 1 0 0 1 1 1 1]
Tuple field:tuple=(0,1,0,1,0,0,0,0) [0 1 0 1 0 0 0 0]
Tuple field:tuple=(0,1,0,1,0,0,1,0) [0 1 0 1 0 0 1 0]
Tuple field:tuple=(0,1,0,1,0,0,1,1) [0 1 0 1 0 0 1 1]
Tuple field:tuple=(0,1,0,1,0,1,0,0) [0 1 0 1 0 1 0 0]
Tuple field:tuple=(0,1,0,1,0,1,1,0) [0 1 0 1 0 1 1 0]
Tuple field:tuple=(0,1,0,1,0,1,1,1) [0 1 0 1 0 1 1 1]
Tuple field:tuple=(0,1,0,1,1,0,1,0) [0 1 0 1 1 0 1 0]
Tuple field:tuple=(0,1,0,1,1,0,1,1) [0 1 0 1 1 0 1 1]
Tuple field:tuple=(0,1,0,1,1,1,1,0) [0 1 0 1 1 1 1 0]
Tuple field:tuple=(0,1,0,1,1,1,1,1) [0 1 0 1 1 1 1 1]
Tuple field:tuple=(0,1,1,0,0,0,0,0) [0 1 1 0 0 0 0 0]
Tuple field:tuple=(0,1,1,0,0,0,1,0) [0 1 1 0 0 0 1 0]
Tuple field:tuple=(0,1,1,0,0,0,1,1) [0 1 1 0 0 0 1 1]
Tuple field:tuple=(0,1,1,0,0,1,0,0) [0 1 1 0 0 1 0 0]
Tuple field:tuple=(0,1,1,0,0,1,1,0) [0 1 1 0 0 1 1 0]
Tuple field:tuple=(0,1,1,0,0,1,1,1) [0 1 1 0 0 1 1 1]
Tuple field:tuple=(0,1,1,0,1,0,1,0) [0 1 1 0 1 0 1 0]
Tuple field:tuple=(0,1,1,0,1,0,1,1) [0 1 1 0 1 0 1 1]
Tuple field:tuple=(0,1,1,0,1,1,1,0) [0 1 1 0 1 1 1 0]
Tuple field:tuple=(0,1,1,0,1,1,1,1) [0 1 1 0 1 1 1 1]
Tuple field:tuple=(0,1,1,1,0,0,0,0) [0 1 1 1 0 0 0 0]
Tuple field:tuple=(0,1,1,1,0,0,1,0) [0 1 1 1 0 0 1 0]
Tuple field:tuple=(0,1,1,1,0,0,1,1) [0 1 1 1 0 0 1 1]
Tuple field:tuple=(0,1,1,1,0,1,0,0) [0 1 1 1 0 1 0 0]
Tuple field:tuple=(0,1,1,1,0,1,1,0) [0 1 1 1 0 1 1 0]
Tuple field:tuple=(0,1,1,1,0,1,1,1) [0 1 1 1 0 1 1 1]
Tuple field:tuple=(0,1,1,1,1,0,1,0) [0 1 1 1 1 0 1 0]
Tuple field:tuple=(0,1,1,1,1,0,1,1) [0 1 1 1 1 0 1 1]
Tuple field:tuple=(0,1,1,1,1,1,1,0) [0 1 1 1 1 1 1 0]
Tuple field:tuple=(0,1,1,1,1,1,1,1) [0 1 1 1 1 1 1 1]
Tuple field:tuple=(1,0,0,0,0,0,0,0) [1 0 0 0 0 0 0 0]`

// Registration descends the String_Invariants bundle — eight Sometimes axes (the empty
// axis plus seven content axes built from the Sometimes_Has_ helpers) and nine
// Impossibles. The snapshot pins the footprint: one per-element entry per axis and the 81
// tuples surviving the carves — empty excludes every content axis (so with empty true
// only the all-false-content cell stands), and a NUL or a line break is itself a control
// character.
func Test_Register_Snapshots_String_Invariants(t *testing.T) {
	const sugar = `package sugar

func String_Invariants(s string) (dot_elements []invariant.Dot_Element) {
	empty := Sometimes(len(s) == 0, "empty")
	edge_whitespace := Sometimes_Has_Edge_Whitespace(s)
	interior_whitespace := Sometimes_Has_Interior_Whitespace(s)
	invalid_utf8 := Sometimes_Has_Invalid_UTF8(s)
	nul := Sometimes_Has_Nul(s)
	byte_rune_mismatch := Sometimes_Has_Multibyte_Rune(s)
	control := Sometimes_Has_Control(s)
	line_break := Sometimes_Has_Line_Break(s)
	return append(dot_elements,
		empty,
		edge_whitespace, interior_whitespace, invalid_utf8, nul,
		byte_rune_mismatch, control, line_break,
		Impossible(Event_True("empty"), Event_True("Sometimes_Has_Edge_Whitespace")),
		Impossible(Event_True("empty"), Event_True("Sometimes_Has_Interior_Whitespace")),
		Impossible(Event_True("empty"), Event_True("Sometimes_Has_Invalid_UTF8")),
		Impossible(Event_True("empty"), Event_True("Sometimes_Has_Nul")),
		Impossible(Event_True("empty"), Event_True("Sometimes_Has_Multibyte_Rune")),
		Impossible(Event_True("empty"), Event_True("Sometimes_Has_Control")),
		Impossible(Event_True("empty"), Event_True("Sometimes_Has_Line_Break")),
		Impossible(Event_True("Sometimes_Has_Nul"), Event_False("Sometimes_Has_Control")),
		Impossible(Event_True("Sometimes_Has_Line_Break"), Event_False("Sometimes_Has_Control")),
	)
}
`
	const application = `package app

import (
	invariant "example.com/m/invariant"
	"example.com/m/sugar"
)

func check(s string) {
	invariant.Dot_Product("field", sugar.String_Invariants(s)...)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"m/go.mod":         &fstest.MapFile{Data: []byte("module example.com/m\n")},
			"m/sugar/sugar.go": &fstest.MapFile{Data: []byte(sugar)},
			"m/app/app.go":     &fstest.MapFile{Data: []byte(application)},
		},
		Output:        &bytes.Buffer{},
		Sugar_Package: "example.com/m/sugar",
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/m/app")

	snap.Expect(t, snap.Init(string_invariants), snapshot_registered(recorder))
}

// Registration descends a user-defined bundle on a user-defined type exactly as it does a
// default preset — same prefix · own-message keying, same grid, same Impossible carve —
// the only difference being that the primitives are qualified (invariant.Sometimes /
// invariant.Impossible), since the bundle lives outside the sugar package and so no
// Sugar_Package is set. Two Sometimes axes over an Account with an Impossible forbidding
// overdrawn-and-empty leave three of the four tuples.
func Test_Register_Snapshots_User_Defined_Bundle(t *testing.T) {
	const source = `package fixture

type Account struct {
	Balance int
}

func Account_Invariants(a Account) (dot_elements []invariant.Dot_Element) {
	overdrawn := invariant.Sometimes(a.Balance < 0, "overdrawn")
	empty := invariant.Sometimes(a.Balance == 0, "empty")
	return append(dot_elements,
		overdrawn, empty,
		invariant.Impossible(invariant.Event_True("overdrawn"), invariant.Event_True("empty")),
	)
}

func check(a Account) {
	invariant.Dot_Product("account", Account_Invariants(a)...)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/account.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	snap.Expect(t, snap.Init(`Sometimes account · empty a.Balance == 0
Sometimes account · overdrawn a.Balance < 0
Tuple account:tuple=(0,0) [0 0]
Tuple account:tuple=(0,1) [0 1]
Tuple account:tuple=(1,0) [1 0]`), snapshot_registered(recorder))
}

// Registration descends a composed bundle: Transfer_Invariants spreads Amount_Invariants
// (which itself nests Sign_Invariants) and Memo_Invariants. The snapshot pins that every
// axis — however deep its bundle nests — attributes to the one top-level Dot_Product
// callsite (transfer.go:27), keyed by its own site in its own bundle, with the grid
// spanning all three combined axes.
func Test_Register_Snapshots_Composed_Bundle(t *testing.T) {
	const source = `package fixture

type Transfer struct {
	Amount int
	Memo   string
}

func Sign_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, invariant.Sometimes(n < 0, "sign"))
}

func Amount_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	dot_elements = append(dot_elements, invariant.Sometimes(n == 0, "amount"))
	return append(dot_elements, Sign_Invariants(n)...)
}

func Memo_Invariants(s string) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, invariant.Sometimes(len(s) == 0, "memo"))
}

func Transfer_Invariants(x Transfer) (dot_elements []invariant.Dot_Element) {
	dot_elements = append(dot_elements, Amount_Invariants(x.Amount)...)
	return append(dot_elements, Memo_Invariants(x.Memo)...)
}

func check(x Transfer) {
	invariant.Dot_Product("transfer", Transfer_Invariants(x)...)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/transfer.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	snap.Expect(t, snap.Init(`Sometimes transfer · amount n == 0
Sometimes transfer · memo len(s) == 0
Sometimes transfer · sign n < 0
Tuple transfer:tuple=(0,0,0) [0 0 0]
Tuple transfer:tuple=(0,0,1) [0 0 1]
Tuple transfer:tuple=(0,1,0) [0 1 0]
Tuple transfer:tuple=(0,1,1) [0 1 1]
Tuple transfer:tuple=(1,0,0) [1 0 0]
Tuple transfer:tuple=(1,0,1) [1 0 1]
Tuple transfer:tuple=(1,1,0) [1 1 0]
Tuple transfer:tuple=(1,1,1) [1 1 1]`), snapshot_registered(recorder))
}

// A bundle whose package is in neither the analyzed module nor a go.work sibling
// (here there is no go.work at all) cannot be resolved. Its elements would be
// enforced at runtime yet seed no coverage, so registration must fail rather than
// drop them silently: it reports the bundle by site and exits non-zero, and nothing
// is seeded.
func Test_Register_Fatal_On_Unresolvable_Cross_Module_Bundle(t *testing.T) {
	const package_b = `package b

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
			"m/b/b.go": &fstest.MapFile{Data: []byte(package_b)},
		},
		Output: &output,
		Exit:   func(code int) { exit_code = code },
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/m/b")

	if exit_code != 1 {
		t.Fatalf("an unresolvable cross-module bundle must exit 1, got %d", exit_code)
	}
	if !strings.Contains(output.String(), "Pair_Invariants") {
		t.Errorf("the report must name the unresolved bundle, got: %s", output.String())
	}
	seeded := 0
	recorder.Events.Range(func(key, value any) (more bool) {
		seeded++
		return true
	})
	if seeded != 0 {
		t.Errorf("an unresolvable bundle must seed nothing, got %d entries", seeded)
	}
}

// A bundle defined in the recorder-less sugar package calls the sugar
// unqualified (Sometimes / Impossible / Event_True). With Sugar_Package naming
// that package's import path, the descent recognizes those bare calls, so the
// preset's axes seed and its Impossible carves the (1,1) tuple.
func Test_Register_Recognizes_Unqualified_Sugar(t *testing.T) {
	const sugar = `package sugar

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	lo := Sometimes(n < 0, "lo")
	hi := Sometimes(n > 0, "hi")
	return append(dot_elements, lo, hi,
		Impossible(Event_True("lo"), Event_True("hi")))
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
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"m/go.mod":         &fstest.MapFile{Data: []byte("module example.com/m\n")},
			"m/sugar/sugar.go": &fstest.MapFile{Data: []byte(sugar)},
			"m/app/app.go":     &fstest.MapFile{Data: []byte(application)},
		},
		Output:        &bytes.Buffer{},
		Sugar_Package: "example.com/m/sugar",
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/m/app")

	if _, ok := recorder.Events.Load(
		"field" + invariant.Element_Message_Separator + "lo"); !ok {
		t.Error("expected the unqualified Sometimes keyed under prefix field (field␀lo)")
	}
	if _, ok := recorder.Events.Load("lo"); ok {
		t.Error("the unqualified Sometimes must not be seeded under its bare message")
	}
	if _, ok := recorder.Events.Load("field:tuple=(0,0)"); !ok {
		t.Error("expected the surviving tuple (0,0) under app.go's Dot_Product prefix")
	}
	if _, ok := recorder.Events.Load("field:tuple=(1,1)"); ok {
		t.Error("tuple (1,1) must be carved by the unqualified Impossible; not seeded")
	}
}

// Registration recognizes a dedicated Sometimes_Has_X helper consumed by a Dot_Product as
// a Sometimes axis, and a bare Always_Has_X / Never_Has_X statement as an eager Always axis
// (Never_Has_X is Always(!has_X)). These helpers take no message argument: their identity
// is the helper's own name (the call selector), so the Sometimes_Has_ axis keys under the
// Dot_Product prefix joined to "Sometimes_Has_Edge_Whitespace", and each bare Always keys
// by its own selector name; the whole call is the condition text.
func Test_Register_Recognizes_String_Axis_Helpers(t *testing.T) {
	const source = `package fixture

func check(s string) {
	invariant.Always_Has_Control(s)
	invariant.Never_Has_Line_Break(s)
	invariant.Dot_Product("field",
		invariant.Sometimes_Has_Edge_Whitespace(s),
	)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/check.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	value, found := recorder.Events.Load(
		"field" + invariant.Element_Message_Separator + "Sometimes_Has_Edge_Whitespace")
	if !found {
		t.Fatal("expected a Sometimes entry for Sometimes_Has_Edge_Whitespace")
	}
	metadata := value.(*invariant.Assertion_Metadata)
	if metadata.Kind != invariant.Assertion_Kind_Sometimes {
		t.Errorf("Kind = %d, want Assertion_Kind_Sometimes", metadata.Kind)
	}
	if metadata.Condition != "invariant.Sometimes_Has_Edge_Whitespace(s)" {
		t.Errorf("Condition = %q", metadata.Condition)
	}
	always, found := recorder.Events.Load("Always_Has_Control")
	if !found {
		t.Fatal("expected an entry for bare Always_Has_Control by its own name")
	}
	if always.(*invariant.Assertion_Metadata).Kind != invariant.Assertion_Kind_Always {
		t.Error("Always_Has_Control must register as an Always axis")
	}
	never, found := recorder.Events.Load("Never_Has_Line_Break")
	if !found {
		t.Fatal("expected an entry for bare Never_Has_Line_Break by its own name")
	}
	if never.(*invariant.Assertion_Metadata).Kind != invariant.Assertion_Kind_Always {
		t.Error("Never_Has_Line_Break must register as an Always axis")
	}
}

// A bundle in the sugar package composes a dedicated helper unqualified
// (Sometimes_Has_Edge_Whitespace); the descent recognizes it as a Sometimes axis and keys
// it under the Dot_Product prefix joined to the helper's own name (the call selector).
func Test_Register_Recognizes_Unqualified_String_Axis(t *testing.T) {
	const sugar = `package sugar

func Token_Invariants(s string) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, Sometimes_Has_Edge_Whitespace(s))
}
`
	const application = `package app

import (
	invariant "example.com/m/invariant"
	"example.com/m/sugar"
)

func check(s string) {
	invariant.Dot_Product("field", sugar.Token_Invariants(s)...)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"m/go.mod":         &fstest.MapFile{Data: []byte("module example.com/m\n")},
			"m/sugar/sugar.go": &fstest.MapFile{Data: []byte(sugar)},
			"m/app/app.go":     &fstest.MapFile{Data: []byte(application)},
		},
		Output:        &bytes.Buffer{},
		Sugar_Package: "example.com/m/sugar",
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/m/app")

	if _, ok := recorder.Events.Load(
		"field" + invariant.Element_Message_Separator +
			"Sometimes_Has_Edge_Whitespace"); !ok {
		t.Error("expected the unqualified helper keyed under prefix field")
	}
}

// Without Sugar_Package, bare Sometimes / Impossible inside a resolved bundle are
// NOT the invariant primitives (they could be the user's own functions), so the
// bundle registers nothing — guarding the recognition against false positives.
func Test_Register_Unqualified_Sugar_Ignored_Outside_Sugar_Package(t *testing.T) {
	const sugar = `package sugar

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	lo := Sometimes(n < 0, "lo")
	hi := Sometimes(n > 0, "hi")
	return append(dot_elements, lo, hi,
		Impossible(Event_True("lo"), Event_True("hi")))
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
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"m/go.mod":         &fstest.MapFile{Data: []byte("module example.com/m\n")},
			"m/sugar/sugar.go": &fstest.MapFile{Data: []byte(sugar)},
			"m/app/app.go":     &fstest.MapFile{Data: []byte(application)},
		},
		Output: &bytes.Buffer{},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/m/app")

	if _, ok := recorder.Events.Load(
		"field" + invariant.Element_Message_Separator + "lo"); ok {
		t.Error("bare Sometimes must not be recognized when Sugar_Package is unset")
	}
	if _, ok := recorder.Events.Load("lo"); ok {
		t.Error("the bare Sometimes must not register without Sugar_Package")
	}
	if _, ok := recorder.Events.Load("field:tuple=(0,0)"); ok {
		t.Error("no axes recognized means no tuple grid should be seeded")
	}
}

// Returns a go.work workspace joining modules a and b: module a defines a
// Pair_Invariants bundle, module b spreads it cross-module. Shared by the
// cross-workspace resolution tests.
func workspace_bundle_fixture() (file_system fstest.MapFS) {
	const package_a = `package a

import invariant "example.com/invariant"

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	lo := invariant.Sometimes(n < 0, "lo")
	hi := invariant.Sometimes(n > 0, "hi")
	return append(dot_elements, lo, hi,
		invariant.Impossible(invariant.Event_True("lo"), invariant.Event_True("hi")))
}
`
	const package_b = `package b

import (
	invariant "example.com/invariant"
	"example.com/a"
)

func check(n int) {
	invariant.Dot_Product("field", a.Pair_Invariants(n)...)
}
`
	const go_work = "go 1.25\n\nuse (\n\t./a\n\t./b\n)\n"
	return fstest.MapFS{
		"work/go.work":  &fstest.MapFile{Data: []byte(go_work)},
		"work/a/go.mod": &fstest.MapFile{Data: []byte("module example.com/a\n")},
		"work/a/a.go":   &fstest.MapFile{Data: []byte(package_a)},
		"work/b/go.mod": &fstest.MapFile{Data: []byte("module example.com/b\n")},
		"work/b/b.go":   &fstest.MapFile{Data: []byte(package_b)},
	}
}

// Registration resolves a *_Invariants bundle defined in a SIBLING workspace
// module: a go.work joins modules a and b, b's Dot_Product spreads
// a.Pair_Invariants, and the bundle is descended via the workspace use-list (no
// go/packages). The bundle element keys under b's Dot_Product prefix joined to its
// own message (field␀lo); the bundle's Impossible still carves the (true, true) tuple.
func Test_Register_Resolves_Cross_Workspace_Module_Bundle(t *testing.T) {
	recorder := &invariant.Recorder{File_System: workspace_bundle_fixture()}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/work/b")

	if _, ok := recorder.Events.Load(
		"field" + invariant.Element_Message_Separator + "lo"); !ok {
		t.Error("expected lo keyed under b's Dot_Product prefix (field␀lo)")
	}
	if _, ok := recorder.Events.Load("lo"); ok {
		t.Error("lo must not be seeded under its bare cross-module message")
	}
	if _, ok := recorder.Events.Load("field:tuple=(0,0)"); !ok {
		t.Error("expected the surviving tuple (0,0) under b's Dot_Product prefix")
	}
	if _, ok := recorder.Events.Load("field:tuple=(1,1)"); ok {
		t.Error("tuple (1,1) carved by the sibling bundle's Impossible; not seeded")
	}
}

// A *_Invariants bundle returns its elements for a caller to consume; calling
// invariant.Dot_Product inside the bundle is banned (it would key the assertions to
// the bundle's own site, defeating per-callsite identity). Registration reports the
// violation by site and exits non-zero.
func Test_Register_Bans_Dot_Product_Inside_Bundle(t *testing.T) {
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
		t.Fatalf("expected Exit(1) for Dot_Product in bundle, got code %d", exit_code)
	}
	if !strings.Contains(output.String(), "p.go:4") {
		t.Errorf("ban report must name Dot_Product site p.go:4, got: %s", output.String())
	}
}

// A bundle named in snake_case (the project rule for an unexported type's bundle,
// e.g. pair_invariants) is recognized just like the Ada_Case _Invariants form:
// registration descends it, keys its elements under the Dot_Product prefix joined to
// the element's own message (field␀lo), and honors its Impossible carve of the
// (true, true) tuple.
func Test_Register_Recognizes_Lowercase_Invariants_Bundle(t *testing.T) {
	const source = `package fixture

func pair_invariants(n int) (dot_elements []invariant.Dot_Element) {
	lo := invariant.Sometimes(n < 0, "lo")
	hi := invariant.Sometimes(n > 0, "hi")
	return append(dot_elements, lo, hi,
		invariant.Impossible(invariant.Event_True("lo"), invariant.Event_True("hi")))
}

func check(n int) {
	invariant.Dot_Product("field", pair_invariants(n)...)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/lower.go": &fstest.MapFile{Data: []byte(source)},
		},
		Output: &bytes.Buffer{},
		Exit:   func(code int) {},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if _, ok := recorder.Events.Load(
		"field" + invariant.Element_Message_Separator + "lo"); !ok {
		t.Error("expected the lowercase bundle's lo keyed under prefix field (field␀lo)")
	}
	if _, ok := recorder.Events.Load("field:tuple=(0,0)"); !ok {
		t.Error("expected the surviving tuple (0,0) under the Dot_Product prefix")
	}
	if _, ok := recorder.Events.Load("field:tuple=(1,1)"); ok {
		t.Error("tuple (1,1) is carved by the lowercase bundle's Impossible; not seeded")
	}
}

// Returns a plain Recorder for the runtime tests. Identity is now the
// author-supplied message, not a caller location, so the recorder carries no
// caller seam — each element names itself when constructed.
func new_test_recorder() (recorder *invariant.Recorder) {
	return &invariant.Recorder{}
}
