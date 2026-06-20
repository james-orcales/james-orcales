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

// Registration parses an invariant.Dot_Product over inline elements and seeds
// one entry per element plus the full bucket grid minus the tuples an Impossible
// carves out. zero and one are Sometimes; the Dot_Product's prefix "check" forms
// each element key and the tuple keys; the Impossible forbids the (true, true) cell
// = tuple (1,1).
func Test_Register_Inline_Dot_Product_Seeds_Grid_Minus_Carves(t *testing.T) {
	const source = `package fixture

func check(n int) {
	invariant.Dot_Product("check",
		invariant.Sometimes(n == 0, "zero"),
		invariant.Sometimes(n == 1, "one"),
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

// Registration descends a *_Invariants bundle invoked inside a Dot_Product: each
// bundle element is keyed by the Dot_Product's prefix joined to the element's own
// message ("field␀lo"), so each prefix tracks the bundle separately; the tuple grid
// is keyed by the prefix ("field"), and the bundle's Impossible carves the (true,
// true) tuple (1,1).
func Test_Register_Descends_Bundle(t *testing.T) {
	const source = `package fixture

func Pair_Invariants(n int, namespace string) {
	invariant.Dot_Product(namespace,
		invariant.Sometimes(n < 0, "lo"),
		invariant.Sometimes(n > 0, "hi"),
		invariant.Impossible(invariant.Event_True("lo"), invariant.Event_True("hi")),
	)
}

func check(n int) {
	Pair_Invariants(n, "field")
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
		t.Error("expected lo keyed by namespace joined to its own message (field␀lo)")
	}
	if _, ok := recorder.Events.Load("lo"); ok {
		t.Error("lo must not be seeded under its bare message; the namespace qualifies it")
	}
	if _, ok := recorder.Events.Load("field:tuple=(0,0)"); !ok {
		t.Error("expected the surviving tuple (0,0) under the namespace")
	}
	if _, ok := recorder.Events.Load("field:tuple=(1,1)"); ok {
		t.Error("tuple (1,1) is carved by the bundle's Impossible; must not be seeded")
	}
}

// A composing _Invariants self-emits its own grid under its namespace and calls the sub-_Invariants
// with a LITERAL sub-namespace, registering each as its own self-contained grid — never flattened
// into the parent. Outer's axis keys under "field"; Inner's keys under the "field.inner"
// sub-namespace, not under "field". There is no joint cross-product across the two types.
func Test_Register_Nested_Invariants_Register_Separate_Grids(t *testing.T) {
	const source = `package fixture

func Inner_Invariants(n int, namespace string) {
	invariant.Dot_Product(namespace, invariant.Sometimes(n < 0, "inner"))
}

func Outer_Invariants(n int, namespace string) {
	invariant.Dot_Product(namespace, invariant.Sometimes(n > 0, "outer"))
	Inner_Invariants(n, "field.inner")
}

func check(n int) {
	Outer_Invariants(n, "field")
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/nested.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if _, ok := recorder.Events.Load(
		"field" + invariant.Element_Message_Separator + "outer"); !ok {
		t.Error("Outer Sometimes must key under its own namespace (field␀outer)")
	}
	if _, ok := recorder.Events.Load(
		"field.inner" + invariant.Element_Message_Separator + "inner"); !ok {
		t.Error("Inner Sometimes must key under its own sub-namespace (field.inner␀inner)")
	}
	if _, ok := recorder.Events.Load(
		"field" + invariant.Element_Message_Separator + "inner"); ok {
		t.Error("Inner's axis must not flatten into the parent grid (no field␀inner)")
	}
}

// One _Invariants called at two callsites with DISTINCT namespaces ("a" and "b") seeds two distinct
// per-axis entries, one per namespace — so a namespace covering only the true branch cannot mask a
// sibling that covered only the false branch. The lone Sometimes ("lo") yields "a␀lo" and "b␀lo",
// and no shared bare entry.
func Test_Register_Two_Namespaces_Of_One_Invariants_Yield_Distinct_Entries(t *testing.T) {
	const source = `package fixture

func Pair_Invariants(n int, namespace string) {
	invariant.Dot_Product(namespace, invariant.Sometimes(n < 0, "lo"))
}

func check_a(n int) {
	Pair_Invariants(n, "a")
}

func check_b(n int) {
	Pair_Invariants(n, "b")
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
		t.Error("expected a distinct entry for namespace a (a␀lo)")
	}
	if _, ok := recorder.Events.Load(
		"b" + invariant.Element_Message_Separator + "lo"); !ok {
		t.Error("expected a distinct entry for namespace b (b␀lo)")
	}
	if _, ok := recorder.Events.Load("lo"); ok {
		t.Error("the two namespaces must not share a bare per-axis entry")
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

func Pair_Invariants(n int, namespace string) {
	invariant.Dot_Product(namespace,
		invariant.Sometimes(n < 0, "lo"),
		invariant.Sometimes(n > 0, "hi"),
		invariant.Impossible(invariant.Event_True("lo"), invariant.Event_True("hi")))
}
`
	const package_b = `package b

import (
	invariant "example.com/m/invariant"
	"example.com/m/a"
)

func check(n int) {
	a.Pair_Invariants(n, "field")
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

// Renders an Assertion_Kind as the short name the registration snapshot reads by.
func kind_name(kind invariant.Assertion_Kind) (name string) {
	switch kind {
	case invariant.Assertion_Kind_Always:
		return "Always"
	case invariant.Assertion_Kind_Sometimes:
		return "Sometimes"
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
// (n == 0/1/2) — exactly as the preset builds it, with the explicit Recorder_Sometimes
// constructor whose condition rides the second argument and whose message rides the argument
// past it. The snapshot pins the whole footprint: one per-element entry per axis (keyed
// prefix · own-message) plus the full 2^3 tuple grid (no Impossible carves), every entry
// attributed to the caller's single Dot_Product prefix "field".
func Test_Register_Snapshots_Whole_Number_Invariants(t *testing.T) {
	const package_a = `package a

import invariant "example.com/m/invariant"

func Whole_Number_Invariants[I ~int | ~int64](n I, namespace string) {
	invariant.Dot_Product(namespace,
		invariant.Recorder_Sometimes(Default, n == 0, "zero"),
		invariant.Recorder_Sometimes(Default, n == 1, "one"),
		invariant.Recorder_Sometimes(Default, n == 2, "two"),
	)
}
`
	const package_b = `package b

import (
	invariant "example.com/m/invariant"
	"example.com/m/a"
)

func check(n int) {
	a.Whole_Number_Invariants(n, "field")
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

	snap.Expect(t, snap.Init(`Sometimes field · one n == 1
Sometimes field · two n == 2
Sometimes field · zero n == 0
Tuple field:tuple=(0,0,0) [0 0 0]
Tuple field:tuple=(0,0,1) [0 0 1]
Tuple field:tuple=(0,1,0) [0 1 0]
Tuple field:tuple=(0,1,1) [0 1 1]
Tuple field:tuple=(1,0,0) [1 0 0]
Tuple field:tuple=(1,0,1) [1 0 1]
Tuple field:tuple=(1,1,0) [1 1 0]
Tuple field:tuple=(1,1,1) [1 1 1]`), snapshot_registered(recorder))
}

// Registration descends the Float_Invariants bundle — three Sometimes axes (NaN,
// negative, positive), each bound to a local and built with the unqualified sugar
// primitive carrying its own message, plus three pairwise Impossibles. The snapshot pins
// the footprint: one per-element entry per axis (keyed prefix · own-message) and only the
// tuples the three mutual-exclusions leave standing — every co-true pair is carved, so of
// the 2^3 grid just the four with at most one axis true survive.
func Test_Register_Snapshots_Float_Invariants(t *testing.T) {
	const sugar = `package sugar

func Float_Invariants[F ~float32 | ~float64](f F, namespace string) {
	value := float64(f)
	Dot_Product(namespace,
		Sometimes(math.IsNaN(value), "nan"),
		Sometimes(value < 0, "negative"),
		Sometimes(value > 0, "positive"),
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
	sugar.Float_Invariants(f, "field")
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
const string_invariants = `Sometimes field · The value has a NUL byte. string_has_nul(s)
Sometimes field · The value has a control character. string_has_control(s)
Sometimes field · The value has a line break. string_has_line_break(s)
Sometimes field · The value has a multi-byte rune. string_has_multibyte_rune(s)
Sometimes field · The value has edge whitespace. string_has_edge_whitespace(s)
Sometimes field · The value has interior whitespace. string_has_interior_whitespace(s)
Sometimes field · The value has invalid UTF-8. string_has_invalid_utf8(s)
Sometimes field · The value is empty. len(s) == 0
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
// axis plus seven content axes over string-content predicates) and nine Impossibles. The
// snapshot pins the footprint: one per-element entry per axis and the 81 tuples surviving the
// carves — empty excludes every content axis (so with empty true only the all-false-content
// cell stands), and a NUL or a line break is itself a control character.
func Test_Register_Snapshots_String_Invariants(t *testing.T) {
	const sugar = `package sugar

func String_Invariants(s string, namespace string) {
	Dot_Product(namespace,
		Sometimes(len(s) == 0, "The value is empty."),
		Sometimes(string_has_edge_whitespace(s), "The value has edge whitespace."),
		Sometimes(string_has_interior_whitespace(s), "The value has interior whitespace."),
		Sometimes(string_has_invalid_utf8(s), "The value has invalid UTF-8."),
		Sometimes(string_has_nul(s), "The value has a NUL byte."),
		Sometimes(string_has_multibyte_rune(s), "The value has a multi-byte rune."),
		Sometimes(string_has_control(s), "The value has a control character."),
		Sometimes(string_has_line_break(s), "The value has a line break."),
		Impossible(Event_True("The value is empty."), Event_True("The value has edge whitespace.")),
		Impossible(Event_True("The value is empty."), Event_True("The value has interior whitespace.")),
		Impossible(Event_True("The value is empty."), Event_True("The value has invalid UTF-8.")),
		Impossible(Event_True("The value is empty."), Event_True("The value has a NUL byte.")),
		Impossible(Event_True("The value is empty."), Event_True("The value has a multi-byte rune.")),
		Impossible(Event_True("The value is empty."), Event_True("The value has a control character.")),
		Impossible(Event_True("The value is empty."), Event_True("The value has a line break.")),
		Impossible(Event_True("The value has a NUL byte."), Event_False("The value has a control character.")),
		Impossible(Event_True("The value has a line break."), Event_False("The value has a control character.")),
	)
}
`
	const application = `package app

import (
	invariant "example.com/m/invariant"
	"example.com/m/sugar"
)

func check(s string) {
	sugar.String_Invariants(s, "field")
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

func Account_Invariants(a Account, namespace string) {
	invariant.Dot_Product(namespace,
		invariant.Sometimes(a.Balance < 0, "overdrawn"),
		invariant.Sometimes(a.Balance == 0, "empty"),
		invariant.Impossible(invariant.Event_True("overdrawn"), invariant.Event_True("empty")),
	)
}

func check(a Account) {
	Account_Invariants(a, "account")
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

// Registration of a composed _Invariants: Transfer_Invariants composes Amount_Invariants (which
// itself composes Sign_Invariants) and Memo_Invariants, each called with its own LITERAL
// sub-namespace. The snapshot pins that each composes into its OWN self-contained grid — three
// independent one-axis grids under "transfer.amount", "transfer.amount.sign", "transfer.memo" — not
// one joint cross-product. Transfer itself has no direct axes, so it seeds no grid of its own.
func Test_Register_Snapshots_Composed_Bundle(t *testing.T) {
	const source = `package fixture

type Transfer struct {
	Amount int
	Memo   string
}

func Sign_Invariants(n int, namespace string) {
	invariant.Dot_Product(namespace, invariant.Sometimes(n < 0, "sign"))
}

func Amount_Invariants(n int, namespace string) {
	invariant.Dot_Product(namespace, invariant.Sometimes(n == 0, "amount"))
	Sign_Invariants(n, "transfer.amount.sign")
}

func Memo_Invariants(s string, namespace string) {
	invariant.Dot_Product(namespace, invariant.Sometimes(len(s) == 0, "memo"))
}

func Transfer_Invariants(x Transfer, namespace string) {
	Amount_Invariants(x.Amount, "transfer.amount")
	Memo_Invariants(x.Memo, "transfer.memo")
}

func check(x Transfer) {
	Transfer_Invariants(x, "transfer")
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/transfer.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	snap.Expect(t, snap.Init(`Sometimes transfer.amount · amount n == 0
Sometimes transfer.amount.sign · sign n < 0
Sometimes transfer.memo · memo len(s) == 0
Tuple transfer.amount.sign:tuple=(0) [0]
Tuple transfer.amount.sign:tuple=(1) [1]
Tuple transfer.amount:tuple=(0) [0]
Tuple transfer.amount:tuple=(1) [1]
Tuple transfer.memo:tuple=(0) [0]
Tuple transfer.memo:tuple=(1) [1]`), snapshot_registered(recorder))
}

// A bundle whose package is outside the analyzed module — its import path does not sit under the
// go.mod module path — cannot be resolved. Its elements would be enforced at runtime yet seed no
// coverage, so registration must fail rather than drop them silently: it reports the bundle by site
// and exits non-zero, and nothing is seeded.
func Test_Register_Fatal_On_Unresolvable_Cross_Module_Bundle(t *testing.T) {
	const package_b = `package b

import (
	_ "example.com/m/invariant"
	"other.com/x"
)

func check(n int) {
	x.Pair_Invariants(n, "field")
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

func Pair_Invariants(n int, namespace string) {
	Dot_Product(namespace,
		Sometimes(n < 0, "lo"),
		Sometimes(n > 0, "hi"),
		Impossible(Event_True("lo"), Event_True("hi")))
}
`
	const application = `package app

import (
	_ "example.com/m/invariant"
	"example.com/m/sugar"
)

func check(n int) {
	sugar.Pair_Invariants(n, "field")
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

// Without Sugar_Package, bare Sometimes / Impossible inside a resolved bundle are
// NOT the invariant primitives (they could be the user's own functions), so the
// bundle registers nothing — guarding the recognition against false positives.
func Test_Register_Unqualified_Sugar_Ignored_Outside_Sugar_Package(t *testing.T) {
	const sugar = `package sugar

func Pair_Invariants(n int, namespace string) {
	Dot_Product(namespace,
		Sometimes(n < 0, "lo"),
		Sometimes(n > 0, "hi"),
		Impossible(Event_True("lo"), Event_True("hi")))
}
`
	const application = `package app

import (
	_ "example.com/m/invariant"
	"example.com/m/sugar"
)

func check(n int) {
	sugar.Pair_Invariants(n, "field")
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

// A bundle named in snake_case (the project rule for an unexported type's bundle,
// e.g. pair_invariants) is recognized just like the Ada_Case _Invariants form:
// registration descends it, keys its elements under the Dot_Product prefix joined to
// the element's own message (field␀lo), and honors its Impossible carve of the
// (true, true) tuple.
func Test_Register_Recognizes_Lowercase_Invariants_Bundle(t *testing.T) {
	const source = `package fixture

func pair_invariants(n int, namespace string) {
	invariant.Dot_Product(namespace,
		invariant.Sometimes(n < 0, "lo"),
		invariant.Sometimes(n > 0, "hi"),
		invariant.Impossible(invariant.Event_True("lo"), invariant.Event_True("hi")))
}

func check(n int) {
	pair_invariants(n, "field")
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

// Benchmark_Dot_Product_Enforcement measures the shipped-binary hot path: a Dot_Product consuming
// two Sometimes axes with recording off (Is_Test false), so only enforcement runs. This is the path
// a production binary pays on every assertion and must not allocate.
func Benchmark_Dot_Product_Enforcement(b *testing.B) {
	recorder := &invariant.Recorder{}
	b.ReportAllocs()
	for range b.N {
		invariant.Recorder_Dot_Product(recorder, "bench",
			invariant.Recorder_Sometimes(recorder, true, "a"),
			invariant.Recorder_Sometimes(recorder, false, "b"))
	}
}

// Benchmark_Dot_Product_Recording measures the test / fuzz-worker hot path: recording on, the grid
// pre-seeded as registration would, so each call credits its elements and tuple. The per-callsite
// handle cache is warmed first, so the loop measures steady state — the cost a fuzz worker pays per
// input — which must not allocate.
func Benchmark_Dot_Product_Recording(b *testing.B) {
	recorder := &invariant.Recorder{Is_Test: true}
	for _, key := range []string{
		"bench" + invariant.Element_Message_Separator + "a",
		"bench" + invariant.Element_Message_Separator + "b",
		"bench:tuple=(1,0)",
	} {
		recorder.Events.Store(key, &invariant.Assertion_Metadata{
			Kind: invariant.Assertion_Kind_Sometimes, Message: key,
		})
	}
	// Warm the per-callsite handle cache so the loop measures steady state, not the
	// one-time key construction the first call pays.
	invariant.Recorder_Dot_Product(recorder, "bench",
		invariant.Recorder_Sometimes(recorder, true, "a"),
		invariant.Recorder_Sometimes(recorder, false, "b"))
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		invariant.Recorder_Dot_Product(recorder, "bench",
			invariant.Recorder_Sometimes(recorder, true, "a"),
			invariant.Recorder_Sometimes(recorder, false, "b"))
	}
}
