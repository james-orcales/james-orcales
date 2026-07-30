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

// Test_Sometimes_Silence prevents its specification contract from regressing.
func Test_Sometimes_Silence(t *testing.T) {
	recorder := &invariant.Recorder{Is_Test: true}
	silent := panic_text(func() { invariant.Recorder_Sometimes(recorder, "id", true, "m") })
	if silent != "" {
		t.Fatalf("true panicked: %q", silent)
	}
	silent = panic_text(func() { invariant.Recorder_Sometimes(recorder, "id", false, "m") })
	if silent != "" {
		t.Fatalf("false panicked: %q", silent)
	}
}

// Test_Sometimes_Recording prevents its specification contract from regressing.
func Test_Sometimes_Recording(t *testing.T) {
	recorder := &invariant.Recorder{Is_Test: true}
	scoped := seed_sometimes_entry(recorder, scoped_key("id", "m"))
	bare := seed_sometimes_entry(recorder, scoped_key("", "m"))
	unscoped := seed_sometimes_entry(recorder, "m")
	invariant.Recorder_Sometimes(recorder, "id", true, "m")
	if scoped.Frequency.Load() != 1 {
		t.Fatal("a non-empty identifier must credit its scoped key")
	}
	invariant.Recorder_Sometimes(recorder, "", false, "m")
	if bare.False_Frequency.Load() != 1 {
		t.Fatal("an empty identifier must still key through the separator")
	}
	if unscoped.Frequency.Load() != 0 {
		t.Fatal("the bare message key must never be credited")
	}
	if unscoped.False_Frequency.Load() != 0 {
		t.Fatal("the bare message key must never be credited")
	}
}

// Test_Range_Bounds prevents its specification contract from regressing.
func Test_Range_Bounds(t *testing.T) {
	recorder := &invariant.Recorder{}
	message := panic_text(func() { invariant.Recorder_Range(recorder, "id", 11, 0, 10) })
	if !strings.Contains(message, "above the maximum 10") {
		t.Fatalf("panic = %q", message)
	}
	message = panic_text(func() { invariant.Recorder_Range(recorder, "id", -1, 0, 10) })
	if !strings.Contains(message, "below the minimum 0") {
		t.Fatalf("panic = %q", message)
	}
	if panic_text(func() { invariant.Recorder_Range(recorder, "id", 5, 0, 10) }) != "" {
		t.Fatal("an in-range value must pass")
	}
	message = panic_text(func() { invariant.Recorder_Range(recorder, "id", 5, 0, 10, 5) })
	if !strings.Contains(message, "excluded") {
		t.Fatalf("panic = %q", message)
	}
}

// Test_Range_Exclusions prevents its specification contract from regressing.
func Test_Range_Exclusions(t *testing.T) {
	modes := []*invariant.Recorder{
		{}, {Is_Test: true}, {Is_Benchmark: true}, {Is_Fuzz: true}, {Is_Fuzz_Worker: true},
	}
	for mode_index, mode := range modes {
		for _, hole := range []int{-1, 0, 10, 11} {
			message := panic_text(func() {
				invariant.Recorder_Range(mode, "id", 5, 0, 10, hole)
			})
			if !strings.Contains(message, "strictly inside") {
				t.Fatalf("mode %d hole %d panic = %q", mode_index, hole, message)
			}
		}
	}
	if panic_text(func() {
		invariant.Recorder_Range(&invariant.Recorder{}, "id", 5, 0, 10, 7)
	}) != "" {
		t.Fatal("a strictly-inside hole beside the value must pass")
	}
	invalid, output, code := registered_fixture(`package fixture
func check(v int) { invariant.Range("holed", v, 0, 2, 0) }
`)
	if code != 1 {
		t.Fatalf("static exclusion exit = %d, want 1; output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "strictly inside") {
		t.Fatalf("diagnostic = %q, want strictly inside", output.String())
	}
	if count := event_count(&invalid.Events); count != 0 {
		t.Fatalf("a refused exclusion seeded %d events", count)
	}
}

// Test_Range_Boundaries prevents its specification contract from regressing.
func Test_Range_Boundaries(t *testing.T) {
	recorder := &invariant.Recorder{Is_Test: true}
	minimum := seed_sometimes_entry(
		recorder, scoped_key("id", invariant.RANGE_MESSAGE_MINIMUM))
	maximum := seed_sometimes_entry(
		recorder, scoped_key("id", invariant.RANGE_MESSAGE_MAXIMUM))
	invariant.Recorder_Range(recorder, "id", 0, 0, 10)
	if minimum.Frequency.Load() != 1 {
		t.Fatal("a value at the minimum must credit the minimum witness")
	}
	if maximum.False_Frequency.Load() != 1 {
		t.Fatal("a value off the maximum must credit the maximum's false branch")
	}
	invariant.Recorder_Range(recorder, "id", 10, 0, 10)
	if maximum.Frequency.Load() != 1 {
		t.Fatal("a value at the maximum must credit the maximum witness")
	}
	benchmark := &invariant.Recorder{Is_Test: true, Is_Benchmark: true}
	silent := seed_sometimes_entry(
		benchmark, scoped_key("id", invariant.RANGE_MESSAGE_MINIMUM))
	invariant.Recorder_Range(benchmark, "id", 0, 0, 10)
	if silent.Frequency.Load() != 0 {
		t.Fatal("a benchmark must not record")
	}
}

// Test_Enum_Membership prevents its specification contract from regressing.
func Test_Enum_Membership(t *testing.T) {
	recorder := &invariant.Recorder{}
	message := panic_text(func() { invariant.Recorder_Enum(recorder, "id", 7, 1, 2) })
	if !strings.Contains(message, "not among the members 1 2") {
		t.Fatalf("panic = %q", message)
	}
	if panic_text(func() { invariant.Recorder_Enum(recorder, "id", 2, 1, 2) }) != "" {
		t.Fatal("a member must pass")
	}
}

// Test_Enum_Distinct prevents its specification contract from regressing.
func Test_Enum_Distinct(t *testing.T) {
	modes := []*invariant.Recorder{
		{}, {Is_Test: true}, {Is_Benchmark: true}, {Is_Fuzz: true}, {Is_Fuzz_Worker: true},
	}
	for mode_index, mode := range modes {
		for _, members := range [][]int{nil, {1}, {1, 1}} {
			message := panic_text(func() {
				invariant.Recorder_Enum(mode, "id", 1, members...)
			})
			if !strings.Contains(message, "at least two distinct members") {
				t.Fatalf("mode %d members %v panic = %q",
					mode_index, members, message)
			}
		}
	}
	if panic_text(func() {
		invariant.Recorder_Enum(&invariant.Recorder{}, "id", 1, 0, 0, 1)
	}) != "" {
		t.Fatal("two distinct members with repetition must pass")
	}
	invalid, output, code := registered_fixture(`package fixture
func check(v int) { invariant.Enum("mode", v, 1, 1) }
`)
	if code != 1 {
		t.Fatalf("static enum exit = %d, want 1; output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "at least two distinct members") {
		t.Fatalf("diagnostic = %q", output.String())
	}
	if count := event_count(&invalid.Events); count != 0 {
		t.Fatalf("a refused enum seeded %d events", count)
	}
}

// Test_Enum_Members prevents its specification contract from regressing.
func Test_Enum_Members(t *testing.T) {
	recorder := &invariant.Recorder{Is_Test: true}
	one := seed_sometimes_entry(
		recorder, scoped_key("id", invariant.Enum_Member_Message("1")))
	two := seed_sometimes_entry(
		recorder, scoped_key("id", invariant.Enum_Member_Message("2")))
	invariant.Recorder_Enum(recorder, "id", 1, 1, 2, 1)
	if one.Frequency.Load() != 1 {
		t.Fatal("the matched member must credit its witness")
	}
	if two.False_Frequency.Load() != 1 {
		t.Fatal("an unmatched member must credit its false branch")
	}
	registered, _, code := registered_fixture(`package fixture
func check(v int) { invariant.Enum("mode", v, 0, 0, 1) }
`)
	if code != -1 {
		t.Fatalf("exit = %d, want no exit", code)
	}
	recorder_event(t, registered, scoped_key("mode", invariant.Enum_Member_Message("0")))
	recorder_event(t, registered, scoped_key("mode", invariant.Enum_Member_Message("1")))
	if count := event_count(&registered.Events); count != 2 {
		t.Fatalf("events = %d, want one witness per distinct member", count)
	}
}

// Test_Bundles_Static prevents its specification contract from regressing.
func Test_Bundles_Static(t *testing.T) {
	invalid, output, code := registered_fixture(`package fixture
type Number int
func Number_Invariants(identifier string, n Number) {
	if n == 0 { invariant.Sometimes(identifier, true, "zero") }
}
func check(n Number) { Number_Invariants("number", n) }
`)
	if code != 1 {
		t.Fatalf("exit = %d, want 1; output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "control flow inside bundle") {
		t.Fatalf("output = %q", output.String())
	}
	if count := event_count(&invalid.Events); count != 0 {
		t.Fatalf("a refused bundle seeded %d events", count)
	}
}

// Test_Bundles_Parameter prevents its specification contract from regressing.
func Test_Bundles_Parameter(t *testing.T) {
	_, output, code := registered_fixture(`package fixture
type Number int
func Number_Invariants(n Number, identifier string) {
	invariant.Sometimes(identifier, n == 0, "zero")
}
`)
	if code != 1 {
		t.Fatalf("exit = %d, want 1; output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "identifier string") {
		t.Fatalf("output = %q, want the required shape", output.String())
	}
}

// Test_Bundles_Relay prevents its specification contract from regressing.
func Test_Bundles_Relay(t *testing.T) {
	_, output, code := registered_fixture(`package fixture
type Number int
func Number_Invariants(identifier string, n Number) {
	invariant.Sometimes("literal", n == 0, "zero")
}
`)
	if code != 1 {
		t.Fatalf("literal exit = %d, want 1; output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "forward") {
		t.Fatalf("output = %q, want forwarding diagnostic", output.String())
	}
	_, output, code = registered_fixture(`package fixture
type Number int
func Number_Invariants(identifier string, n Number) {
	invariant.Sometimes(identifier+".", n == 0, "zero")
}
`)
	if code != 1 {
		t.Fatalf("derived exit = %d, want 1; output=%q", code, output.String())
	}
}

// Test_Bundles_Primitive prevents its specification contract from regressing.
func Test_Bundles_Primitive(t *testing.T) {
	_, output, code := registered_fixture(`package fixture
func Int_Invariants(identifier string, n int) {
	invariant.Sometimes(identifier, n == 0, "zero")
}
`)
	if code != 1 {
		t.Fatalf("exit = %d, want 1; output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "primitive bundle") {
		t.Fatalf("output = %q", output.String())
	}
	_, sugar_output, sugar_code := registered_fixture_with_sugar(`package invariant
func Int_Invariants(identifier string, n int) {
	Sometimes(identifier, n == 0, "zero")
}
`)
	if sugar_code != -1 {
		t.Fatalf("sugar exit = %d, want no exit; output=%q", sugar_code,
			sugar_output.String())
	}
}

// Test_Bundles_Composition prevents its specification contract from regressing.
func Test_Bundles_Composition(t *testing.T) {
	recorder, _, code := registered_fixture(`package fixture
type Number int
type Pair int
func Number_Invariants(identifier string, n Number) {
	invariant.Sometimes(identifier, n == 0, "zero")
}
func Pair_Invariants(identifier string, p Pair) {
	invariant.Sometimes(identifier, p == 1, "one")
	Number_Invariants(identifier, Number(p))
}
func check(p Pair) { Pair_Invariants("pair", p) }
`)
	if code != -1 {
		t.Fatalf("exit = %d, want no exit", code)
	}
	recorder_event(t, recorder, scoped_key("pair", "one"))
	recorder_event(t, recorder, scoped_key("pair", "zero"))
	if count := event_count(&recorder.Events); count != 2 {
		t.Fatalf("events = %d, want 2 — keys must carry no callee names", count)
	}
	_, output, duplicate_code := registered_fixture(`package fixture
type Number int
type Pair int
func Number_Invariants(identifier string, n Number) {
	invariant.Sometimes(identifier, n == 0, "zero")
}
func Pair_Invariants(identifier string, p Pair) {
	invariant.Sometimes(identifier, p == 0, "zero")
	Number_Invariants(identifier, Number(p))
}
func check(p Pair) { Pair_Invariants("pair", p) }
`)
	if duplicate_code != 1 {
		t.Fatalf("duplicate exit = %d, want 1; output=%q", duplicate_code, output.String())
	}
	if !strings.Contains(output.String(), "duplicate message") {
		t.Fatalf("output = %q", output.String())
	}
}

// Test_Bundles_Cycles prevents its specification contract from regressing.
func Test_Bundles_Cycles(t *testing.T) {
	_, output, code := registered_fixture(`package fixture
type A int
type B int
func A_Invariants(identifier string, a A) {
	B_Invariants(identifier, B(a))
}
func B_Invariants(identifier string, b B) {
	A_Invariants(identifier, A(b))
}
func check(a A) { A_Invariants("cycle", a) }
`)
	if code != 1 {
		t.Fatalf("exit = %d, want 1; output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "bundle cycle") {
		t.Fatalf("output = %q", output.String())
	}
}

// Test_Registration_Roots prevents its specification contract from regressing.
func Test_Registration_Roots(t *testing.T) {
	recorder, output, code := registered_fixture(`package fixture
func check(v bool, r *invariant.Recorder) {
	invariant.Sometimes("io", v, "read observed")
	invariant.Recorder_Sometimes(r, "cli", v, "flag observed")
}
`)
	if code != -1 {
		t.Fatalf("exit = %d, want no exit; output=%q", code, output.String())
	}
	scoped := recorder_event(t, recorder, scoped_key("io", "read observed"))
	if scoped.Kind != invariant.ASSERTION_KIND_SOMETIMES {
		t.Fatalf("kind = %d, want Sometimes", scoped.Kind)
	}
	recorder_event(t, recorder, scoped_key("cli", "flag observed"))
}

// Test_Registration_Descent prevents its specification contract from regressing.
func Test_Registration_Descent(t *testing.T) {
	recorder, output, code := registered_fixture(`package fixture
type Number int
func Number_Invariants(identifier string, n Number) {
	invariant.Sometimes(identifier, n == 0, "zero")
	invariant.Range(identifier, int(n), 0, 9)
}
func first(n Number) { Number_Invariants("first", n) }
func second(n Number) { Number_Invariants("second", n) }
`)
	if code != -1 {
		t.Fatalf("exit = %d, want no exit; output=%q", code, output.String())
	}
	recorder_event(t, recorder, scoped_key("first", "zero"))
	recorder_event(t, recorder, scoped_key("second", "zero"))
	recorder_event(t, recorder, scoped_key("first", invariant.RANGE_MESSAGE_MINIMUM))
	recorder_event(t, recorder, scoped_key("second", invariant.RANGE_MESSAGE_MAXIMUM))
	if count := event_count(&recorder.Events); count != 6 {
		t.Fatalf("events = %d, want three per root", count)
	}
}

// Test_Registration_Sugar prevents its specification contract from regressing.
func Test_Registration_Sugar(t *testing.T) {
	const SUGAR_SOURCE = `package sugar
func Number_Invariants(identifier string, n int) {
	Sometimes(identifier, n == 0, "zero")
}
`
	const CONSUMER_SOURCE = `package consumer
import "fixture/sugar"
func check(n int) { sugar.Number_Invariants("number", n) }
`
	recorder, output, code := sugar_fixture(SUGAR_SOURCE, CONSUMER_SOURCE, "fixture/sugar")
	if code != -1 {
		t.Fatalf("exit = %d, want no exit; output=%q", code, output.String())
	}
	recorder_event(t, recorder, scoped_key("number", "zero"))
	outside, _, outside_code := sugar_fixture(SUGAR_SOURCE, CONSUMER_SOURCE, "")
	if outside_code != -1 {
		t.Fatalf("outside exit = %d, want no exit", outside_code)
	}
	if count := event_count(&outside.Events); count != 0 {
		t.Fatalf("unqualified calls outside the sugar package seeded %d events", count)
	}
}

// Test_Registration_Unresolved prevents its specification contract from regressing.
func Test_Registration_Unresolved(t *testing.T) {
	_, output, code := registered_fixture(`package fixture
func check(v int) { Missing_Invariants("io", v) }
`)
	if code != 1 {
		t.Fatalf("exit = %d, want 1; output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "unresolved bundle") {
		t.Fatalf("output = %q", output.String())
	}
}

// Test_Registration_Atomicity prevents its specification contract from regressing.
func Test_Registration_Atomicity(t *testing.T) {
	recorder, output, code := registered_fixture(`package fixture
func check(a bool, b bool) {
	invariant.Always(a, "held")
	invariant.Always(b, "held")
}
`)
	if code != 1 {
		t.Fatalf("exit = %d, want 1; output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), `duplicate message: "held"`) {
		t.Fatalf("output = %q, want the duplicate line", output.String())
	}
	if count := event_count(&recorder.Events); count != 0 {
		t.Fatalf("a failed registration seeded %d events", count)
	}
}

// Test_Registration_Bounds prevents its specification contract from regressing.
func Test_Registration_Bounds(t *testing.T) {
	recorder, output, code := registered_fixture(`package fixture
const MAXIMUM = 10/3 - 1
func check(v int) {
	invariant.Range("bounded", v, -(2), MAXIMUM+1)
}
`)
	if code != -1 {
		t.Fatalf("exit = %d, want no exit; output=%q", code, output.String())
	}
	recorder_event(t, recorder, scoped_key("bounded", invariant.RANGE_MESSAGE_MINIMUM))
	recorder_event(t, recorder, scoped_key("bounded", invariant.RANGE_MESSAGE_MAXIMUM))
	invalid, unresolved_output, unresolved_code := registered_fixture(`package fixture
func check(v int, limit int) { invariant.Range("bounded", v, 0, limit) }
`)
	if unresolved_code != 1 {
		t.Fatalf("unresolved exit = %d, want 1", unresolved_code)
	}
	if !strings.Contains(unresolved_output.String(), "unresolved preset bounds") {
		t.Fatalf("diagnostic = %q", unresolved_output.String())
	}
	if count := event_count(&invalid.Events); count != 0 {
		t.Fatalf("an unresolved bound seeded %d events", count)
	}
}

// Test_Coverage_Uniqueness prevents its specification contract from regressing.
func Test_Coverage_Uniqueness(t *testing.T) {
	_, output, code := registered_fixture(`package fixture
func first(v bool) { invariant.Sometimes("same", v, "a") }
func second(v bool) { invariant.Sometimes("same", v, "b") }
`)
	if code != 1 {
		t.Fatalf("exit = %d, want 1; output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "duplicate identifier") {
		t.Fatalf("output = %q, want duplicate identifier", output.String())
	}
}

// Test_Coverage_Literal prevents its specification contract from regressing.
func Test_Coverage_Literal(t *testing.T) {
	_, output, code := registered_fixture(`package fixture
func check(v bool, identifier string) { invariant.Sometimes(identifier, v, "m") }
`)
	if code != 1 {
		t.Fatalf("variable identifier exit = %d, want 1; output=%q", code, output.String())
	}
	_, output, code = registered_fixture(`package fixture
func check(v bool) { invariant.Sometimes("", v, "m") }
`)
	if code != 1 {
		t.Fatalf("empty identifier exit = %d, want 1; output=%q", code, output.String())
	}
	_, output, code = registered_fixture(`package fixture
func check(v bool, message string) { invariant.Sometimes("io", v, message) }
`)
	if code != 1 {
		t.Fatalf("variable message exit = %d, want 1; output=%q", code, output.String())
	}
	_, output, code = registered_fixture(`package fixture
func check(v bool, message string) { invariant.Always(v, message) }
`)
	if code != 1 {
		t.Fatalf("variable Always message exit = %d, want 1; output=%q",
			code, output.String())
	}
}

// Test_Coverage_Separator prevents its specification contract from regressing.
func Test_Coverage_Separator(t *testing.T) {
	_, output, code := registered_fixture(`package fixture
func check(v bool) { invariant.Sometimes("bad\x00id", v, "m") }
`)
	if code != 1 {
		t.Fatalf("NUL identifier exit = %d, want 1; output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), "NUL") {
		t.Fatalf("diagnostic = %q, want NUL", output.String())
	}
	_, output, code = registered_fixture(`package fixture
func check(v bool) { invariant.Sometimes("io", v, "bad\x00m") }
`)
	if code != 1 {
		t.Fatalf("NUL message exit = %d, want 1; output=%q", code, output.String())
	}
	_, output, code = registered_fixture(`package fixture
func check(v bool) { invariant.Always(v, "bad\x00m") }
`)
	if code != 1 {
		t.Fatalf("NUL Always message exit = %d, want 1; output=%q", code, output.String())
	}
}

// Test_Coverage_Modes prevents its specification contract from regressing.
func Test_Coverage_Modes(t *testing.T) {
	benchmark := &invariant.Recorder{Is_Test: true, Is_Benchmark: true}
	entry := seed_sometimes_entry(benchmark, scoped_key("id", "m"))
	invariant.Recorder_Sometimes(benchmark, "id", true, "m")
	if entry.Frequency.Load() != 0 {
		t.Fatal("a benchmark must not record")
	}
	production := &invariant.Recorder{}
	entry = seed_sometimes_entry(production, scoped_key("p", "m"))
	invariant.Recorder_Sometimes(production, "p", true, "m")
	if entry.Frequency.Load() != 0 {
		t.Fatal("outside a test run nothing records")
	}
}

// Test_Coverage_Persistence prevents its specification contract from regressing.
func Test_Coverage_Persistence(t *testing.T) {
	recorder, _, _ := registered_fixture(`package fixture
func check(v bool) { invariant.Sometimes("io", v, "read observed") }
`)
	var lines []string
	recorder.Coverage_Sink = func(key string, fired_true bool) {
		lines = append(lines, invariant.Fuzz_Coverage_Line(key, fired_true))
	}
	invariant.Recorder_Sometimes(recorder, "io", true, "read observed")
	if len(lines) != 1 {
		t.Fatalf("sink lines = %d, want 1", len(lines))
	}
	entry := recorder_event(t, recorder, scoped_key("io", "read observed"))
	entry.Frequency.Store(0)
	invariant.Recorder_Merge_Fuzz_Coverage_From(recorder, strings.NewReader(lines[0]))
	if entry.Frequency.Load() != 1 {
		t.Fatal("a NUL-carrying key must round-trip the fuzz merge")
	}
}

// Test_Analysis_Gaps prevents its specification contract from regressing.
func Test_Analysis_Gaps(t *testing.T) {
	recorder, output, _ := registered_fixture(`package fixture
func check(v bool) { invariant.Sometimes("io", v, "read observed") }
`)
	invariant.Recorder_Sometimes(recorder, "io", true, "read observed")
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	if !strings.Contains(output.String(), "false branch never observed") {
		t.Fatalf("report = %q", output.String())
	}
	if !strings.Contains(output.String(), "io · read observed") {
		t.Fatalf("report = %q, want the rendered key", output.String())
	}
}

// Test_Analysis_Summary prevents its specification contract from regressing.
func Test_Analysis_Summary(t *testing.T) {
	recorder, _, _ := registered_fixture(`package fixture
func check(v bool) {
	invariant.Always(v, "held")
	invariant.Sometimes("io", v, "read observed")
}
`)
	summary := invariant.Recorder_Assertion_Summary(recorder)
	want := "✓ fixture: tested 3 properties" +
		" (3 individual + 0 combinations, of which 1 are panic-able)"
	if summary != want {
		t.Fatalf("summary = %q, want %q", summary, want)
	}
}

// Test_Analysis_Clean prevents its specification contract from regressing.
func Test_Analysis_Clean(t *testing.T) {
	recorder, output, _ := registered_fixture(`package fixture
func check(v bool) { invariant.Sometimes("io", v, "read observed") }
`)
	invariant.Recorder_Sometimes(recorder, "io", true, "read observed")
	invariant.Recorder_Sometimes(recorder, "io", false, "read observed")
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	if output.Len() != 0 {
		t.Fatalf("clean output = %q, want empty", output.String())
	}
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
	tty := &bytes.Buffer{}
	code = -1
	recorder = &invariant.Recorder{
		File_System: fstest.MapFS{
			"go.mod":           &fstest.MapFile{Data: []byte("module fixture\n")},
			"fixture/check.go": &fstest.MapFile{Data: []byte(source)},
		},
		Packages_To_Analyze: []string{"/fixture"},
		Output:              output,
		Tty:                 tty,
		Exit:                func(status int) { code = status },
		Is_Test:             true,
		Sugar_Package:       sugar,
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	return recorder, output, code
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
	events.Range(func(_, _ any) (more bool) {
		count++
		return true
	})
	return count
}

// A two-package module: the analyzed consumer roots a bundle the sugar package declares, so
// cross-package resolution and the sugar-only unqualified recognition are both exercised.
func sugar_fixture(sugar_source string, consumer_source string, sugar_package string) (
	recorder *invariant.Recorder, output *bytes.Buffer, code int,
) {
	output = &bytes.Buffer{}
	code = -1
	recorder = &invariant.Recorder{
		File_System: fstest.MapFS{
			"go.mod":            &fstest.MapFile{Data: []byte("module fixture\n")},
			"sugar/sugar.go":    &fstest.MapFile{Data: []byte(sugar_source)},
			"consumer/check.go": &fstest.MapFile{Data: []byte(consumer_source)},
		},
		Packages_To_Analyze: []string{"/consumer"},
		Output:              output,
		Exit:                func(status int) { code = status },
		Is_Test:             true,
		Sugar_Package:       sugar_package,
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	return recorder, output, code
}

func scoped_key(identifier string, message string) (key string) {
	return identifier + invariant.ELEMENT_MESSAGE_SEPARATOR + message
}

func seed_sometimes_entry(
	recorder *invariant.Recorder, key string,
) (metadata *invariant.Assertion_Metadata) {
	metadata = &invariant.Assertion_Metadata{
		Kind:    invariant.ASSERTION_KIND_SOMETIMES,
		Message: key,
	}
	recorder.Events.Store(key, metadata)
	return metadata
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
