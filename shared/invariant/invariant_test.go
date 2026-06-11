package invariant_test

import (
	"bytes"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/james-orcales/james-orcales/shared/invariant"
)

// A Dot_Product containing an Always(false) element must fail: the asserted
// condition did not hold. Mirrors v2's Always semantics, adapted to v3's
// collect-then-check shape — Recorder_Always returns a False element rather
// than failing at construction, so Recorder_Dot_Product is where the violation
// is caught.
func Test_Dot_Product_Always_False_Fails(t *testing.T) {
	recorder := &invariant.Recorder{}
	failed := false
	func() {
		defer func() {
			if recover() != nil {
				failed = true
			}
		}()
		invariant.Recorder_Dot_Product(recorder, invariant.Recorder_Always(recorder, false))
	}()
	if !failed {
		t.Fatal("Recorder_Dot_Product must fail when given an Always(false) element")
	}
}

// When an Impossible declares a combination of element events and that exact
// combination occurs in the call, Recorder_Dot_Product must fail.
func Test_Dot_Product_Impossible_Combination_Fails(t *testing.T) {
	recorder := new_test_recorder()
	a := invariant.Recorder_Sometimes(recorder, true)
	b := invariant.Recorder_Sometimes(recorder, true)
	forbidden := invariant.Impossible(
		invariant.Event_True(a),
		invariant.Event_True(b),
	)
	failed := false
	func() {
		defer func() {
			if recover() != nil {
				failed = true
			}
		}()
		invariant.Recorder_Dot_Product(recorder, a, b, forbidden)
	}()
	if !failed {
		t.Fatal("Recorder_Dot_Product must fail when an Impossible combination occurs")
	}
}

// When the combination an Impossible forbids does NOT occur (one referenced
// event differs from what was observed), Recorder_Dot_Product must not fail.
func Test_Dot_Product_Impossible_Combination_Absent_Passes(t *testing.T) {
	recorder := new_test_recorder()
	a := invariant.Recorder_Sometimes(recorder, true)
	b := invariant.Recorder_Sometimes(recorder, false)
	forbidden := invariant.Impossible(
		invariant.Event_True(a),
		invariant.Event_True(b),
	)
	failed := false
	func() {
		defer func() {
			if recover() != nil {
				failed = true
			}
		}()
		invariant.Recorder_Dot_Product(recorder, a, b, forbidden)
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
		recorder, &invariant.Boundary_Input[int]{X: 5, Lo: 0, Hi: 3},
	)
	failed := false
	func() {
		defer func() {
			if recover() != nil {
				failed = true
			}
		}()
		invariant.Recorder_Dot_Product(recorder, outside_bounds)
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
		recorder, &invariant.Boundary_Input[int]{X: 2, Lo: 0, Hi: 3},
	)
	failed := false
	func() {
		defer func() {
			if recover() != nil {
				failed = true
			}
		}()
		invariant.Recorder_Dot_Product(recorder, within_bounds)
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
		recorder, &invariant.Boundary_Input[int]{X: 3, Lo: 3, Hi: 3},
	)
	failed := false
	func() {
		defer func() {
			if recover() != nil {
				failed = true
			}
		}()
		invariant.Recorder_Dot_Product(recorder, bad_bounds)
	}()
	if !failed {
		t.Fatal("Recorder_Dot_Product must fail when a Distinct_Boundary has Lo >= Hi")
	}
}

// An element's Site is its caller location (file:line) — the identity that static
// registration and the runtime rendezvous on.
func Test_Element_Site_Is_Caller_Location(t *testing.T) {
	recorder := &invariant.Recorder{
		Get_Caller: func(skip int) (file string, line int) { return "fixture.go", 42 },
	}
	element := invariant.Recorder_Sometimes(recorder, true)
	if element.Site != "fixture.go:42" {
		t.Fatalf("element Site = %q, want fixture.go:42", element.Site)
	}
}

// With Site_Root set, an element's Site is reported relative to that workspace
// root rather than as the absolute path Get_Caller returns.
func Test_Element_Site_Relative_At_Runtime(t *testing.T) {
	recorder := &invariant.Recorder{
		Site_Root:  "/work",
		Get_Caller: func(skip int) (file string, line int) { return "/work/pkg/x.go", 7 },
	}
	element := invariant.Recorder_Sometimes(recorder, true)
	if element.Site != "pkg/x.go:7" {
		t.Fatalf("element Site = %q, want pkg/x.go:7", element.Site)
	}
}

// Recorder_Dot_Product increments the seeded tracker entry for each observed
// element: Frequency on a true event, False_Frequency on false.
func Test_Dot_Product_Increments_Seeded_Element(t *testing.T) {
	recorder := new_test_recorder()
	recorder.Is_Test = true
	element := invariant.Recorder_Sometimes(recorder, true)
	metadata := &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Sometimes, Site: element.Site,
	}
	recorder.Assertions.Store(element.Site, metadata)
	invariant.Recorder_Dot_Product(recorder, element)
	if metadata.Frequency.Load() != 1 {
		t.Fatalf("Frequency = %d, want 1", metadata.Frequency.Load())
	}
	if metadata.False_Frequency.Load() != 0 {
		t.Fatalf("False_Frequency = %d, want 0", metadata.False_Frequency.Load())
	}
}

// A bundle element is credited at runtime under its caller-qualified key
// (callsite::from=site), not its bare site: the Dot_Product callsite the runtime
// computes for the tuple also qualifies each element. The counter stub hands the
// element site fixture.go:1 and the Dot_Product callsite fixture.go:2; with both a
// combined and a bare entry seeded, the combined (bundle) key wins and the bare one
// is left untouched.
func Test_Dot_Product_Credits_Bundle_Element_Via_Combined_Key(t *testing.T) {
	recorder := new_test_recorder()
	recorder.Is_Test = true
	element := invariant.Recorder_Sometimes(recorder, true) // Site fixture.go:1
	combined := &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Sometimes, Site: "fixture.go:2::from=fixture.go:1",
	}
	bare := &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Sometimes, Site: "fixture.go:1",
	}
	recorder.Assertions.Store("fixture.go:2::from=fixture.go:1", combined)
	recorder.Assertions.Store("fixture.go:1", bare)
	invariant.Recorder_Dot_Product(recorder, element)
	if combined.Frequency.Load() != 1 {
		t.Fatalf("combined Frequency = %d, want 1", combined.Frequency.Load())
	}
	if bare.Frequency.Load() != 0 {
		t.Fatalf("bare Frequency = %d, want 0 (the combined key takes precedence)",
			bare.Frequency.Load())
	}
}

// Recorder_Dot_Product increments the seeded tuple entry for the observed
// combination of element events, keyed by the Dot_Product call site. Both
// Sometimes elements fire true, so the observed tuple is (1, 1); the call site
// consumes the counter stub's third value, fixture.go:3.
func Test_Dot_Product_Increments_Seeded_Tuple(t *testing.T) {
	recorder := new_test_recorder()
	recorder.Is_Test = true
	a := invariant.Recorder_Sometimes(recorder, true) // Site fixture.go:1
	b := invariant.Recorder_Sometimes(recorder, true) // Site fixture.go:2
	tuple := &invariant.Assertion_Metadata{
		Kind:          invariant.Assertion_Kind_Tuple,
		Site:          "fixture.go:3",
		Tuple_Indices: []int{1, 1},
	}
	recorder.Assertions.Store("fixture.go:3:tuple=(1,1)", tuple)
	invariant.Recorder_Dot_Product(recorder, a, b)
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
			recorder, &invariant.Boundary_Input[int]{X: c.X, Lo: 0, Hi: 3},
		)
		metadata := &invariant.Assertion_Metadata{
			Kind: invariant.Assertion_Kind_Boundary, Site: element.Site,
		}
		recorder.Assertions.Store(element.Site, metadata)
		invariant.Recorder_Dot_Product(recorder, element)
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
// carves out. zero and one are Sometimes (lines 4 and 5); the Dot_Product is at
// line 6; the Impossible forbids the (true, true) cell = tuple (1,1).
func Test_Register_Inline_Dot_Product_Seeds_Grid_Minus_Carves(t *testing.T) {
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

	if _, ok := recorder.Assertions.Load("/fixture/check.go:4"); !ok {
		t.Error("expected a per-element entry seeded at the zero Sometimes (line 4)")
	}
	if _, ok := recorder.Assertions.Load("/fixture/check.go:6:tuple=(0,0)"); !ok {
		t.Error("expected the surviving tuple (0,0) seeded at the Dot_Product (line 6)")
	}
	if _, ok := recorder.Assertions.Load("/fixture/check.go:6:tuple=(1,1)"); ok {
		t.Error("tuple (1,1) is carved by the Impossible; it must not be seeded")
	}
}

// Registration recognizes a Distinct_Boundary and seeds a two-bucket grid (Lo=0,
// Hi=1) keyed by the Dot_Product callsite (line 4), plus a per-element entry at
// the boundary's own site (line 5) whose Condition is the X expression source.
func Test_Register_Distinct_Boundary_Seeds_Endpoints(t *testing.T) {
	const source = `package fixture

func check(age int) {
	invariant.Dot_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{X: age, Lo: 22, Hi: 34}),
	)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/b.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	value, found := recorder.Assertions.Load("/fixture/b.go:5")
	if !found {
		t.Fatal("expected a per-element entry seeded at the Distinct_Boundary (line 5)")
	}
	metadata := value.(*invariant.Assertion_Metadata)
	if metadata.Kind != invariant.Assertion_Kind_Boundary {
		t.Errorf("per-element Kind = %d, want Assertion_Kind_Boundary", metadata.Kind)
	}
	if metadata.Condition != "age" {
		t.Errorf("per-element Condition = %q, want age", metadata.Condition)
	}
	if _, ok := recorder.Assertions.Load("/fixture/b.go:4:tuple=(0)"); !ok {
		t.Error("expected the Lo endpoint tuple (0) seeded at the Dot_Product (line 4)")
	}
	if _, ok := recorder.Assertions.Load("/fixture/b.go:4:tuple=(1)"); !ok {
		t.Error("expected the Hi endpoint tuple (1) seeded at the Dot_Product (line 4)")
	}
}

// Registration descends a *_Invariants bundle invoked inside a Dot_Product: each
// bundle element is keyed by the Dot_Product callsite plus its own site inside the
// bundle (caller::from=site, here line 15 + line 4), so every callsite of the
// bundle is tracked separately; the tuple grid is keyed by the Dot_Product callsite
// (line 15), and the bundle's Impossible carves the (true, true) tuple (1,1).
func Test_Register_Descends_Bundle(t *testing.T) {
	const source = `package fixture

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	lo := invariant.Sometimes(n < 0)
	hi := invariant.Sometimes(n > 0)
	dot_elements = append(
		dot_elements,
		lo, hi,
		invariant.Impossible(invariant.Event_True(lo), invariant.Event_True(hi)),
	)
	return dot_elements
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

	if _, ok := recorder.Assertions.Load("/fixture/pair.go:15::from=/fixture/pair.go:4"); !ok {
		t.Error("expected lo keyed by caller::from=site (Dot_Product line 15, lo line 4)")
	}
	if _, ok := recorder.Assertions.Load("/fixture/pair.go:4"); ok {
		t.Error("lo must no longer be seeded under its bare site; the caller qualifies it")
	}
	if _, ok := recorder.Assertions.Load("/fixture/pair.go:15:tuple=(0,0)"); !ok {
		t.Error("expected the surviving tuple (0,0) at the Dot_Product callsite (line 15)")
	}
	if _, ok := recorder.Assertions.Load("/fixture/pair.go:15:tuple=(1,1)"); ok {
		t.Error("tuple (1,1) is carved by the bundle's Impossible; must not be seeded")
	}
}

// A *_Invariants bundle that composes another bundle (Outer appends Inner's
// elements) attributes EVERY descended element — however deep — to the one
// top-level Dot_Product callsite (line 13), not to the inner bundle's call site
// (line 9) and not to a bare site. This is what lets types compose invariants
// while keeping each top-level callsite's credit independent.
func Test_Register_Nested_Bundle_Attributes_To_Top_Level_Caller(t *testing.T) {
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
		t.Error("Inner Sometimes (line 4) must key to the top-level caller (line 13)")
	}
	if _, ok := recorder.Assertions.Load(
		"/fixture/nested.go:13::from=/fixture/nested.go:8"); !ok {
		t.Error("Outer Sometimes (line 8) must key to the top-level caller (line 13)")
	}
	if _, ok := recorder.Assertions.Load(
		"/fixture/nested.go:9::from=/fixture/nested.go:4"); ok {
		t.Error("Inner's element must not attribute to the inner bundle call site (line 9)")
	}
	if _, ok := recorder.Assertions.Load("/fixture/nested.go:4"); ok {
		t.Error("Inner's element must not be seeded under its bare site")
	}
}

// One bundle spread at two distinct Dot_Product callsites seeds two distinct
// per-element entries, one per callsite — so a callsite covering only the true
// branch cannot mask a sibling callsite that covered only the false branch. The
// bundle's lone Sometimes (line 4) yields callsite-A (line 8) and callsite-B
// (line 12) entries, and no shared bare entry.
func Test_Register_Two_Callsites_Of_One_Bundle_Yield_Distinct_Entries(t *testing.T) {
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
		t.Error("expected a distinct entry for callsite A (line 8)")
	}
	if _, ok := recorder.Assertions.Load("/fixture/two.go:12::from=/fixture/two.go:4"); !ok {
		t.Error("expected a distinct entry for callsite B (line 12)")
	}
	if _, ok := recorder.Assertions.Load("/fixture/two.go:4"); ok {
		t.Error("the two callsites must not share a bare per-element entry")
	}
}

// Registration follows a bundle reached through a single-assignment binding before
// the spread (elems := Pair_Invariants(n); Dot_Product(elems...)), not only the
// direct Dot_Product(Pair_Invariants(n)...) form. The element keys to the
// Dot_Product callsite (line 9), not the binding line.
func Test_Register_Bound_Variable_Bundle_Is_Descended(t *testing.T) {
	const source = `package fixture

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, invariant.Sometimes(n < 0))
}

func check(n int) {
	elems := Pair_Invariants(n)
	invariant.Dot_Product(elems...)
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
		t.Error("expected the bound bundle keyed by Dot_Product callsite (line 9)")
	}
	if _, ok := recorder.Assertions.Load("/fixture/bound.go:4"); ok {
		t.Error("the bound bundle's element must not be seeded under its bare site")
	}
}

// Registration follows a bundle appended into the spread variable
// (elems = append(elems, Pair_Invariants(n)...); Dot_Product(elems...)), mirroring
// how a bundle body's own appends are read. The element keys to the Dot_Product
// callsite (line 10).
func Test_Register_Appended_Bundle_Is_Descended(t *testing.T) {
	const source = `package fixture

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, invariant.Sometimes(n < 0))
}

func check(n int) {
	var elems []invariant.Dot_Element
	elems = append(elems, Pair_Invariants(n)...)
	invariant.Dot_Product(elems...)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/append.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if _, ok := recorder.Assertions.Load(
		"/fixture/append.go:10::from=/fixture/append.go:4"); !ok {
		t.Error("expected the appended bundle keyed by Dot_Product callsite (line 10)")
	}
	if _, ok := recorder.Assertions.Load("/fixture/append.go:4"); ok {
		t.Error("the appended bundle's element must not be seeded under its bare site")
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
		Kind: invariant.Assertion_Kind_Sometimes, Site: "f.go:3", Condition: "n == 0",
	}
	sometimes.Frequency.Add(1)
	recorder.Assertions.Store("f.go:3", sometimes)
	tuple := &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Tuple, Site: "f.go:7", Tuple_Indices: []int{1, 0},
	}
	recorder.Assertions.Store("f.go:7:tuple=(1,0)", tuple)
	reached := &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Always, Site: "f.go:9", Condition: "x > 0",
	}
	reached.Frequency.Add(1)
	recorder.Assertions.Store("f.go:9", reached)

	invariant.Recorder_Analyze_Assertion_Frequency(recorder)

	if exit_code != 1 {
		t.Fatalf("Exit code = %d, want 1", exit_code)
	}
	report := output.String()
	if !strings.Contains(report, "f.go:3") {
		t.Error("report must name the Sometimes false-branch gap site f.go:3")
	}
	if !strings.Contains(report, "n == 0") {
		t.Error("report must include the Sometimes condition source")
	}
	if !strings.Contains(report, "f.go:7") {
		t.Error("report must name the never-occurring tuple site f.go:7")
	}
	if strings.Contains(report, "f.go:9") {
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
		Kind: invariant.Assertion_Kind_Boundary, Site: "b.go:5", Condition: "age",
	}
	boundary.Frequency.Add(1)
	recorder.Assertions.Store("b.go:5", boundary)

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
		Kind: invariant.Assertion_Kind_Sometimes, Site: "f.go:3", Condition: "n == 0",
	}
	sometimes.Frequency.Add(1)
	sometimes.False_Frequency.Add(1)
	recorder.Assertions.Store("f.go:3", sometimes)

	invariant.Recorder_Analyze_Assertion_Frequency(recorder)

	if exited {
		t.Error("a clean run must not call Exit")
	}
	if output.Len() != 0 {
		t.Errorf("a clean run must print nothing, got %q", output.String())
	}
}

// Two callsites of one bundle (a.go:9 and b.go:9, same bundle Sometimes at lib.go:4)
// are independent coverage entries: callsite A saw only true and callsite B only
// false, so Recorder_Analyze_Assertion_Frequency reports A's missing false branch
// AND B's missing true branch — neither callsite masks the other's gap, the
// regression this change fixes.
func Test_Analyze_Complementary_Coverage_Reports_Each_Callsite_Gap(t *testing.T) {
	var output bytes.Buffer
	exit_code := -1
	recorder := &invariant.Recorder{
		Is_Test: true,
		Output:  &output,
		Exit:    func(code int) { exit_code = code },
	}
	callsite_a := &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Sometimes,
		Site: "a.go:9::from=lib.go:4", Condition: "n < 0",
	}
	callsite_a.Frequency.Add(1) // saw true; false branch never observed
	recorder.Assertions.Store("a.go:9::from=lib.go:4", callsite_a)
	callsite_b := &invariant.Assertion_Metadata{
		Kind: invariant.Assertion_Kind_Sometimes,
		Site: "b.go:9::from=lib.go:4", Condition: "n < 0",
	}
	callsite_b.False_Frequency.Add(1) // saw false; true branch never observed
	recorder.Assertions.Store("b.go:9::from=lib.go:4", callsite_b)

	invariant.Recorder_Analyze_Assertion_Frequency(recorder)

	if exit_code != 1 {
		t.Fatalf("Exit code = %d, want 1", exit_code)
	}
	report := output.String()
	if !strings.Contains(report, "a.go:9::from=lib.go:4") {
		t.Error("report must name callsite A's gap by its caller-qualified site")
	}
	if !strings.Contains(report, "b.go:9::from=lib.go:4") {
		t.Error("report must name callsite B's gap by its caller-qualified site")
	}
	if !strings.Contains(report, "false branch never observed") {
		t.Error("callsite A must report its unobserved false branch")
	}
	if !strings.Contains(report, "true branch never observed") {
		t.Error("callsite B must report its unobserved true branch")
	}
}

// Recorder_Assertion_Summary counts the seeded assertions for the clean-run
// banner: per-element entries (Always, Sometimes) are individual properties,
// Tuple entries are combinations, and the Always family is panic-able.
func Test_Assertion_Summary_Counts_Properties(t *testing.T) {
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

// Registration resolves a *_Invariants bundle defined in another same-module
// package: package b's Dot_Product spreads a.Pair_Invariants, and the module's
// go.mod maps the import path "example.com/m/a" to its directory so the bundle's
// declaration (in a.go) is found, descended, and its grid seeded.
func Test_Register_Resolves_Cross_Package_Bundle(t *testing.T) {
	const package_a = `package a

import invariant "example.com/m/invariant"

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	lo := invariant.Sometimes(n < 0)
	hi := invariant.Sometimes(n > 0)
	return append(dot_elements, lo, hi,
		invariant.Impossible(invariant.Event_True(lo), invariant.Event_True(hi)))
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
		t.Error("expected lo keyed by callsite b.go:9 and bundle site a.go:6")
	}
	if _, ok := recorder.Assertions.Load("/m/a/a.go:6"); ok {
		t.Error("lo must not be seeded under its bare cross-package site")
	}
	if _, ok := recorder.Assertions.Load("/m/b/b.go:9:tuple=(0,0)"); !ok {
		t.Error("expected the surviving tuple (0,0) at b.go's Dot_Product (line 9)")
	}
	if _, ok := recorder.Assertions.Load("/m/b/b.go:9:tuple=(1,1)"); ok {
		t.Error("tuple (1,1) is carved by the bundle's Impossible; must not be seeded")
	}
}

// Registration descends a generic *_Invariants bundle that returns a single appended
// Distinct_Boundary over its value parameter — the shape Whole_Number_Invariants takes —
// recognizing it cross-package as a Boundary axis keyed
// caller::from=site (b.go:9 + a.go:7), with the Lo/Hi tuple grid seeded at the caller's
// Dot_Product. Guards that the Sometimes→Boundary preset rewrite stays statically
// recognized, generic Boundary_Input[I] and all.
func Test_Register_Resolves_Generic_Boundary_Bundle(t *testing.T) {
	const package_a = `package a

import invariant "example.com/m/invariant"

func Integer_Invariants[I invariant.Numeric](n I) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements,
		invariant.Distinct_Boundary(&invariant.Boundary_Input[I]{X: n, Lo: 0, Hi: 9}))
}
`
	const package_b = `package b

import (
	invariant "example.com/m/invariant"
	"example.com/m/a"
)

func check(n int) {
	invariant.Dot_Product(a.Integer_Invariants(n)...)
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

	value, found := recorder.Assertions.Load("/m/b/b.go:9::from=/m/a/a.go:7")
	if !found {
		t.Fatal("expected the boundary keyed by callsite b.go:9 and bundle site a.go:7")
	}
	metadata := value.(*invariant.Assertion_Metadata)
	if metadata.Kind != invariant.Assertion_Kind_Boundary {
		t.Errorf("Kind = %d, want Assertion_Kind_Boundary", metadata.Kind)
	}
	if metadata.Condition != "n" {
		t.Errorf("Condition = %q, want n", metadata.Condition)
	}
	if _, ok := recorder.Assertions.Load("/m/b/b.go:9:tuple=(0)"); !ok {
		t.Error("expected the Lo endpoint tuple (0) at b.go's Dot_Product (line 9)")
	}
	if _, ok := recorder.Assertions.Load("/m/b/b.go:9:tuple=(1)"); !ok {
		t.Error("expected the Hi endpoint tuple (1) at b.go's Dot_Product (line 9)")
	}
}

// A bundle whose package is in neither the analyzed module nor a go.work sibling
// (here there is no go.work at all) cannot be resolved, so registration skips it
// gracefully — nothing is seeded and no panic occurs.
func Test_Register_Skips_Cross_Module_Bundle(t *testing.T) {
	const package_b = `package b

import (
	invariant "example.com/m/invariant"
	"other.com/x"
)

func check(n int) {
	invariant.Dot_Product(x.Pair_Invariants(n)...)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"m/go.mod": &fstest.MapFile{Data: []byte("module example.com/m\n")},
			"m/b/b.go": &fstest.MapFile{Data: []byte(package_b)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/m/b")

	seeded := 0
	recorder.Assertions.Range(func(key, value any) (more bool) {
		seeded++
		return true
	})
	if seeded != 0 {
		t.Errorf("expected nothing seeded for an unresolvable cross-module bundle, got %d",
			seeded)
	}
}

// A bundle defined in the recorder-less sugar package calls the sugar
// unqualified (Sometimes / Impossible / Event_True). With Sugar_Package naming
// that package's import path, the descent recognizes those bare calls, so the
// preset's axes seed and its Impossible carves the (1,1) tuple.
func Test_Register_Recognizes_Unqualified_Sugar(t *testing.T) {
	const sugar = `package sugar

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	lo := Sometimes(n < 0)
	hi := Sometimes(n > 0)
	return append(dot_elements, lo, hi,
		Impossible(Event_True(lo), Event_True(hi)))
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

	if _, ok := recorder.Assertions.Load("/m/app/app.go:9::from=/m/sugar/sugar.go:4"); !ok {
		t.Error("expected the unqualified Sometimes keyed by app.go:9 and sugar.go:4")
	}
	if _, ok := recorder.Assertions.Load("/m/sugar/sugar.go:4"); ok {
		t.Error("the unqualified Sometimes must not be seeded under its bare site")
	}
	if _, ok := recorder.Assertions.Load("/m/app/app.go:9:tuple=(0,0)"); !ok {
		t.Error("expected the surviving tuple (0,0) at app.go's Dot_Product (line 9)")
	}
	if _, ok := recorder.Assertions.Load("/m/app/app.go:9:tuple=(1,1)"); ok {
		t.Error("tuple (1,1) must be carved by the unqualified Impossible; not seeded")
	}
}

// Registration recognizes a dedicated Sometimes_Has_X / Always_Has_X / Never_Has_X
// helper called directly in a Dot_Product as a Sometimes / Always / Always axis
// (Never_Has_X is Always(!has_X)), sited at the helper's call, with the whole call as
// the condition text.
func Test_Register_Recognizes_String_Axis_Helpers(t *testing.T) {
	const source = `package fixture

func check(s string) {
	invariant.Dot_Product(
		invariant.Sometimes_Has_Edge_Whitespace(s),
		invariant.Always_Has_Control(s),
		invariant.Never_Has_Line_Break(s),
	)
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/check.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	value, found := recorder.Assertions.Load("/fixture/check.go:5")
	if !found {
		t.Fatal("expected an entry at Sometimes_Has_Edge_Whitespace (line 5)")
	}
	metadata := value.(*invariant.Assertion_Metadata)
	if metadata.Kind != invariant.Assertion_Kind_Sometimes {
		t.Errorf("Kind = %d, want Assertion_Kind_Sometimes", metadata.Kind)
	}
	if metadata.Condition != "invariant.Sometimes_Has_Edge_Whitespace(s)" {
		t.Errorf("Condition = %q", metadata.Condition)
	}
	always, found := recorder.Assertions.Load("/fixture/check.go:6")
	if !found {
		t.Fatal("expected an entry at Always_Has_Control (line 6)")
	}
	if always.(*invariant.Assertion_Metadata).Kind != invariant.Assertion_Kind_Always {
		t.Error("Always_Has_Control must register as an Always axis")
	}
	never, found := recorder.Assertions.Load("/fixture/check.go:7")
	if !found {
		t.Fatal("expected an entry at Never_Has_Line_Break (line 7)")
	}
	if never.(*invariant.Assertion_Metadata).Kind != invariant.Assertion_Kind_Always {
		t.Error("Never_Has_Line_Break must register as an Always axis")
	}
}

// A bundle in the sugar package composes a dedicated helper unqualified
// (Sometimes_Has_Edge_Whitespace); the descent recognizes it as a Sometimes axis and
// keys it by the Dot_Product callsite plus its site in the bundle (caller::from=site).
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
	invariant.Dot_Product(sugar.Token_Invariants(s)...)
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

	if _, ok := recorder.Assertions.Load(
		"/m/app/app.go:9::from=/m/sugar/sugar.go:4"); !ok {
		t.Error("expected the unqualified helper keyed by app.go:9 and sugar.go:4")
	}
}

// Without Sugar_Package, bare Sometimes / Impossible inside a resolved bundle are
// NOT the invariant primitives (they could be the user's own functions), so the
// bundle registers nothing — guarding the recognition against false positives.
func Test_Register_Unqualified_Sugar_Ignored_Outside_Sugar_Package(t *testing.T) {
	const sugar = `package sugar

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	lo := Sometimes(n < 0)
	hi := Sometimes(n > 0)
	return append(dot_elements, lo, hi,
		Impossible(Event_True(lo), Event_True(hi)))
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
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"m/go.mod":         &fstest.MapFile{Data: []byte("module example.com/m\n")},
			"m/sugar/sugar.go": &fstest.MapFile{Data: []byte(sugar)},
			"m/app/app.go":     &fstest.MapFile{Data: []byte(application)},
		},
		Output: &bytes.Buffer{},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/m/app")

	if _, ok := recorder.Assertions.Load("/m/sugar/sugar.go:4"); ok {
		t.Error("bare Sometimes must not be recognized when Sugar_Package is unset")
	}
	if _, ok := recorder.Assertions.Load("/m/app/app.go:9::from=/m/sugar/sugar.go:4"); ok {
		t.Error("caller-qualified Sometimes must not register without Sugar_Package")
	}
	if _, ok := recorder.Assertions.Load("/m/app/app.go:9:tuple=(0,0)"); ok {
		t.Error("no axes recognized means no tuple grid should be seeded")
	}
}

// With a go.work in an ancestor directory, registration reports Sites relative to
// that workspace root: the Sometimes on line 4 is seeded as "pkg/x.go:4", not the
// absolute "/proj/pkg/x.go:4".
func Test_Register_Sites_Relative_To_Workspace(t *testing.T) {
	const source = `package pkg

func check(n int) {
	invariant.Dot_Product(invariant.Sometimes(n == 0))
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"proj/go.work":  &fstest.MapFile{Data: []byte("go 1.25\n")},
			"proj/pkg/x.go": &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/proj/pkg")

	if _, ok := recorder.Assertions.Load("pkg/x.go:4"); !ok {
		t.Error("expected the Sometimes seeded at the workspace-relative pkg/x.go:4")
	}
	if _, ok := recorder.Assertions.Load("/proj/pkg/x.go:4"); ok {
		t.Error("Site must be workspace-relative, not the absolute /proj/pkg/x.go:4")
	}
}

// With no go.work but a .git directory in an ancestor, registration falls back to
// the git root for relative Sites.
func Test_Register_Sites_Relative_To_Git_Root(t *testing.T) {
	const source = `package pkg

func check(n int) {
	invariant.Dot_Product(invariant.Sometimes(n == 0))
}
`
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"repo/.git/HEAD": &fstest.MapFile{Data: []byte("ref: refs/heads/main\n")},
			"repo/pkg/x.go":  &fstest.MapFile{Data: []byte(source)},
		},
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/repo/pkg")

	if _, ok := recorder.Assertions.Load("pkg/x.go:4"); !ok {
		t.Error("expected the Sometimes seeded at the git-root-relative pkg/x.go:4")
	}
}

// Returns a go.work workspace joining modules a and b: module a defines a
// Pair_Invariants bundle, module b spreads it cross-module. Shared by the
// cross-workspace resolution tests.
func workspace_bundle_fixture() (file_system fstest.MapFS) {
	const package_a = `package a

import invariant "example.com/invariant"

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	lo := invariant.Sometimes(n < 0)
	hi := invariant.Sometimes(n > 0)
	return append(dot_elements, lo, hi,
		invariant.Impossible(invariant.Event_True(lo), invariant.Event_True(hi)))
}
`
	const package_b = `package b

import (
	invariant "example.com/invariant"
	"example.com/a"
)

func check(n int) {
	invariant.Dot_Product(a.Pair_Invariants(n)...)
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
// go/packages). Sites are workspace-relative, so the bundle element keys by the b
// callsite (line 9) and its own site inside a (line 6); the bundle's Impossible
// still carves the (true, true) tuple.
func Test_Register_Resolves_Cross_Workspace_Module_Bundle(t *testing.T) {
	recorder := &invariant.Recorder{File_System: workspace_bundle_fixture()}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/work/b")

	if _, ok := recorder.Assertions.Load("b/b.go:9::from=a/a.go:6"); !ok {
		t.Error("expected lo keyed by b's callsite (line 9) and a's bundle site (line 6)")
	}
	if _, ok := recorder.Assertions.Load("a/a.go:6"); ok {
		t.Error("lo must not be seeded under its bare cross-module site")
	}
	if _, ok := recorder.Assertions.Load("b/b.go:9:tuple=(0,0)"); !ok {
		t.Error("expected the surviving tuple (0,0) at b's Dot_Product (line 9)")
	}
	if _, ok := recorder.Assertions.Load("b/b.go:9:tuple=(1,1)"); ok {
		t.Error("tuple (1,1) carved by the sibling bundle's Impossible; not seeded")
	}
}

// Every seeded key for a cross-workspace-module bundle is workspace-relative: none
// retains the absolute /work/ prefix. This confirms the sibling module is parsed
// with the same Site_Root (the go.work dir) as the primary module, so its static
// element site matches the runtime site.
func Test_Register_Sites_Relative_Across_Workspace_Modules(t *testing.T) {
	recorder := &invariant.Recorder{File_System: workspace_bundle_fixture()}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/work/b")

	seeded := 0
	recorder.Assertions.Range(func(key, value any) (more bool) {
		seeded++
		if strings.Contains(key.(string), "/work/") {
			t.Errorf("seeded key must be workspace-relative, got absolute: %q", key)
		}
		return true
	})
	if seeded == 0 {
		t.Error("expected the cross-workspace bundle to seed entries")
	}
}

// An axis constructor that never reaches a Dot_Product is an orphan: it enforces
// nothing and seeds nothing. Registration reports it by site and exits non-zero.
func Test_Register_Fatal_On_Orphan_Axis(t *testing.T) {
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
		t.Fatalf("expected Exit(1) for an orphan axis, got exit code %d", exit_code)
	}
	if !strings.Contains(output.String(), "o.go:4") {
		t.Errorf("orphan report must name the axis site o.go:4, got: %s", output.String())
	}
}

// A *_Invariants bundle returns its elements for a caller to consume; calling
// invariant.Dot_Product inside the bundle is banned (it would key the assertions to
// the bundle's own site, defeating per-callsite identity). Registration reports the
// violation by site and exits non-zero.
func Test_Register_Bans_Dot_Product_Inside_Bundle(t *testing.T) {
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
		t.Fatalf("expected Exit(1) for Dot_Product in bundle, got code %d", exit_code)
	}
	if !strings.Contains(output.String(), "p.go:4") {
		t.Errorf("ban report must name Dot_Product site p.go:4, got: %s", output.String())
	}
}

// An axis appended inside a *_Invariants bundle flows into the returned slice (and
// thence a Dot_Product when the bundle is spread), so it is not an orphan even
// though no Dot_Product appears in the bundle itself.
func Test_Register_Bundle_Axis_Not_Orphan(t *testing.T) {
	const source = `package fixture

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	lo := invariant.Sometimes(n < 0)
	return append(dot_elements, lo)
}
`
	exited := false
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/b.go": &fstest.MapFile{Data: []byte(source)},
		},
		Output: &bytes.Buffer{},
		Exit:   func(code int) { exited = true },
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if exited {
		t.Error("an axis appended in a *_Invariants bundle must not be flagged as orphan")
	}
}

// An axis appended into a Dot_Product's own argument list — the post-condition
// shape for a function whose bool return is composed beside another return's bundle
// — is consumed by that Dot_Product, so it is not an orphan. The analyzer must
// recurse append() reaching a Dot_Product just as it does inside a bundle.
func Test_Register_Dot_Product_Append_Not_Orphan(t *testing.T) {
	const source = `package fixture

func Pair_Invariants(n int) (dot_elements []invariant.Dot_Element) {
	return append(dot_elements, invariant.Sometimes(n < 0))
}

func check(n int) (size int, ok bool) {
	defer func() {
		invariant.Dot_Product(append(
			Pair_Invariants(size), invariant.Sometimes(ok))...)
	}()
	return n, n > 0
}
`
	exited := false
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/d.go": &fstest.MapFile{Data: []byte(source)},
		},
		Output: &bytes.Buffer{},
		Exit:   func(code int) { exited = true },
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if exited {
		t.Error("an axis appended into a Dot_Product must not be orphan")
	}
}

// An axis bound to a name that appears as a Dot_Product argument is consumed, so
// it is not an orphan.
func Test_Register_Consumed_Axis_Not_Orphan(t *testing.T) {
	const source = `package fixture

func check(n int) {
	zero := invariant.Sometimes(n == 0)
	invariant.Dot_Product(zero)
}
`
	exited := false
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/c.go": &fstest.MapFile{Data: []byte(source)},
		},
		Output: &bytes.Buffer{},
		Exit:   func(code int) { exited = true },
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if exited {
		t.Error("an axis bound to a name used as a Dot_Product argument must not be orphan")
	}
}

// A bundle named in snake_case (the project rule for an unexported type's bundle,
// e.g. pair_invariants) is recognized just like the Ada_Case _Invariants form:
// registration descends it, keys its elements by caller::from=site (line 11 + line
// 4), and honors its Impossible carve of the (true, true) tuple.
func Test_Register_Recognizes_Lowercase_Invariants_Bundle(t *testing.T) {
	const source = `package fixture

func pair_invariants(n int) (dot_elements []invariant.Dot_Element) {
	lo := invariant.Sometimes(n < 0)
	hi := invariant.Sometimes(n > 0)
	return append(dot_elements, lo, hi,
		invariant.Impossible(invariant.Event_True(lo), invariant.Event_True(hi)))
}

func check(n int) {
	invariant.Dot_Product(pair_invariants(n)...)
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

	if _, ok := recorder.Assertions.Load(
		"/fixture/lower.go:11::from=/fixture/lower.go:4"); !ok {
		t.Error("expected the lowercase bundle's lo keyed by caller::from=site")
	}
	if _, ok := recorder.Assertions.Load("/fixture/lower.go:11:tuple=(0,0)"); !ok {
		t.Error("expected the surviving tuple (0,0) at the Dot_Product callsite (line 11)")
	}
	if _, ok := recorder.Assertions.Load("/fixture/lower.go:11:tuple=(1,1)"); ok {
		t.Error("tuple (1,1) is carved by the lowercase bundle's Impossible; not seeded")
	}
}

// An axis appended inside a snake_case bundle (pair_invariants) is consumed by the
// bundle's returned slice just like in an Ada_Case bundle, so it is not an orphan.
func Test_Register_Lowercase_Bundle_Axis_Not_Orphan(t *testing.T) {
	const source = `package fixture

func pair_invariants(n int) (dot_elements []invariant.Dot_Element) {
	lo := invariant.Sometimes(n < 0)
	return append(dot_elements, lo)
}
`
	exited := false
	recorder := &invariant.Recorder{
		File_System: fstest.MapFS{
			"fixture/b.go": &fstest.MapFile{Data: []byte(source)},
		},
		Output: &bytes.Buffer{},
		Exit:   func(code int) { exited = true },
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")

	if exited {
		t.Error("an axis appended in a snake_case bundle must not be flagged as orphan")
	}
}

// Returns a Recorder whose Get_Caller hands out a distinct file:line per
// element, so Site-based identity is unique within a test.
func new_test_recorder() (recorder *invariant.Recorder) {
	n := 0
	return &invariant.Recorder{
		Get_Caller: func(skip int) (file string, line int) {
			n++
			return "fixture.go", n
		},
	}
}
