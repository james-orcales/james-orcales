//go:build (invariant_disable_coverage || prd || prod || production) && !invariant_noop

package invariant_test

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"local/james-orcales/shared/invariant"
)

// Fixture_Subject stands in for a bundle subject where the test drives the builder directly.
// This build resolves no plan, so the chain type only has to compile.
type Fixture_Subject int

// Test_Production_Fatal_Hook verifies that every production assertion entrypoint preserves the
// fatal egress before it panics.
func Test_Production_Fatal_Hook(t *testing.T) {
	tests := []struct {
		Name string
		Run  func(recorder *invariant.Recorder)
	}{
		{Name: "always", Run: func(recorder *invariant.Recorder) {
			invariant.Recorder_Always(recorder, false, "fatal hook guard")
		}},
		{Name: "inline range", Run: func(recorder *invariant.Recorder) {
			invariant.Recorder_Range(recorder, -1, 0, 1, "fatal hook range")
		}},
		{Name: "inline enum", Run: func(recorder *invariant.Recorder) {
			invariant.Recorder_Enum(recorder, 3, 1, 2, "fatal hook enum")
		}},
		{Name: "fluent range", Run: func(recorder *invariant.Recorder) {
			invariant.Recorder_Tree(recorder, Fixture_Subject(0), "fatal hook tree").
				Range_Int(-1, 0, 1).Ensure()
		}},
	}
	for _, test := range tests {
		recorder := &invariant.Recorder{}
		hook_count := 0
		hook_message := ""
		recorder.On_Fatal = func(message string) {
			hook_count++
			hook_message = message
		}
		panic_message := production_panic_text(func() { test.Run(recorder) })
		if hook_count != 1 {
			t.Fatalf("%s hook count = %d, want 1", test.Name, hook_count)
		}
		if hook_message != panic_message {
			t.Fatalf("%s hook message = %q, panic = %q",
				test.Name, hook_message, panic_message)
		}
	}
}

// Opens a chain on a plan-free recorder over the fixture subject, keeping call sites short.
func fixture_assertions(namespace invariant.Namespace) (builder invariant.Assertion_Builder) {
	return invariant.Recorder_Tree(&invariant.Recorder{}, Fixture_Subject(0), namespace)
}

// Test_Optimized_Production_Enforcement_Matches_Literal_Reference keeps the inlineable build tied
// to a direct test implementation whose branches can be inspected without packed state or helpers.
func Test_Optimized_Production_Enforcement_Matches_Literal_Reference(t *testing.T) {
	recorder := &invariant.Recorder{}
	for _, condition := range []bool{false, true} {
		production_pair(
			t, fmt.Sprintf("always=%t", condition),
			func() { invariant.Recorder_Always(recorder, condition, "identity") },
			func() { production_reference_always(condition, "identity") },
		)
	}
	builder := invariant.Recorder_Tree(recorder, Fixture_Subject(0), "ignored")
	production_pair(
		t, "sometimes",
		func() { builder.Sometimes(false, "axis").Ensure() },
		func() {
			production_reference_sometimes(false, "axis")
			production_reference_ensure()
		},
	)
	for value := -2; value <= 6; value++ {
		production_pair(
			t, fmt.Sprintf("range=%d", value),
			func() { builder.Range_Int(value, 0, 4) },
			func() { production_reference_range_int(value, 0, 4) },
		)
		production_pair(
			t, fmt.Sprintf("range-holed=%d", value),
			func() { builder.Range_Holed_Int(value, 0, 4, 2, 2, 2, 2) },
			func() { production_reference_range_holed_int(value, 0, 4, 2, 2, 2, 2) },
		)
		production_pair(
			t, fmt.Sprintf("enum=%d", value),
			func() { builder.Enum_Int(value, 1, 3) },
			func() { production_reference_enum_int(value, 1, 3) },
		)
		production_pair(
			t, fmt.Sprintf("enum-3=%d", value),
			func() { builder.Enum_3_Int(value, 1, 3, 5) },
			func() { production_reference_enum_3_int(value, 1, 3, 5) },
		)
		production_pair(
			t, fmt.Sprintf("enum-4=%d", value),
			func() { builder.Enum_4_Int(value, 1, 2, 3, 5) },
			func() { production_reference_enum_4_int(value, 1, 2, 3, 5) },
		)
	}
}

func production_pair(t *testing.T, name string, optimized func(), reference func()) {
	t.Helper()
	optimized_panic := production_panic_text(optimized)
	reference_panic := production_panic_text(reference)
	if optimized_panic != reference_panic {
		t.Fatalf("%s: optimized panic = %q, reference panic = %q",
			name, optimized_panic, reference_panic)
	}
}

func production_reference_always(condition bool, message string) {
	if !condition {
		panic(invariant.ASSERTION_FAILURE_MESSAGE_PREFIX + message +
			"  Always — condition was false: " + fmt.Sprint(condition))
	}
}

func production_reference_sometimes(condition bool, message string) {
	return
}

func production_reference_ensure() {
	return
}

func production_reference_range_int(value int, minimum int, maximum int) {
	if value < minimum {
		panic(invariant.ASSERTION_FAILURE_MESSAGE_PREFIX +
			invariant.RANGE_GUARD_MINIMUM + "  value below min: " + fmt.Sprint(value))
	}
	if value > maximum {
		panic(invariant.ASSERTION_FAILURE_MESSAGE_PREFIX +
			invariant.RANGE_GUARD_MAXIMUM + "  value exceeds max: " + fmt.Sprint(value))
	}
}

func production_reference_range_holed_int(
	value int, minimum int, maximum int, hole_1 int, hole_2 int, hole_3 int, hole_4 int,
) {
	production_reference_range_int(value, minimum, maximum)
	switch value {
	case hole_1, hole_2, hole_3, hole_4:
		panic(invariant.ASSERTION_FAILURE_MESSAGE_PREFIX +
			"Range value is excluded: " + fmt.Sprint(value))
	}
}

func production_reference_enum_int(value int, first int, second int) {
	switch value {
	case first, second:
		return
	}
	panic(invariant.ASSERTION_FAILURE_MESSAGE_PREFIX +
		invariant.ENUM_GUARD_MEMBER + "  value is not a member: " + fmt.Sprint(value))
}

func production_reference_enum_3_int(value int, first int, second int, third int) {
	switch value {
	case first, second, third:
		return
	}
	panic(invariant.ASSERTION_FAILURE_MESSAGE_PREFIX +
		invariant.ENUM_GUARD_MEMBER + "  value is not a member: " + fmt.Sprint(value))
}

func production_reference_enum_4_int(
	value int, first int, second int, third int, fourth int,
) {
	switch value {
	case first, second, third, fourth:
		return
	}
	panic(invariant.ASSERTION_FAILURE_MESSAGE_PREFIX +
		invariant.ENUM_GUARD_MEMBER + "  value is not a member: " + fmt.Sprint(value))
}

// Test_Production_Int_Assertions_Enforce_At_The_Violating_Link keeps every int family eager.
func Test_Production_Int_Assertions_Enforce_At_The_Violating_Link(t *testing.T) {
	builder := fixture_assertions("must-not-appear")
	production_assert_link(
		t, "Range_Int",
		func() { builder.Range_Int(1, 0, 2) },
		func() { builder.Range_Int(-1, 0, 2) },
	)
	production_assert_link(
		t, "Range_Holed_Int",
		func() { builder.Range_Holed_Int(1, 0, 3, 2, 2, 2, 2) },
		func() { builder.Range_Holed_Int(2, 0, 3, 2, 2, 2, 2) },
	)
	production_assert_link(
		t, "Enum_Int",
		func() { builder.Enum_Int(1, 1, 2) },
		func() { builder.Enum_Int(3, 1, 2) },
	)
	production_assert_link(
		t, "Enum_3_Int",
		func() { builder.Enum_3_Int(2, 1, 2, 3) },
		func() { builder.Enum_3_Int(4, 1, 2, 3) },
	)
	production_assert_link(
		t, "Enum_4_Int",
		func() { builder.Enum_4_Int(3, 1, 2, 3, 4) },
		func() { builder.Enum_4_Int(5, 1, 2, 3, 4) },
	)
}

// Test_Production_Int8_Assertions_Enforce_At_The_Violating_Link keeps every int8 family eager.
func Test_Production_Int8_Assertions_Enforce_At_The_Violating_Link(t *testing.T) {
	builder := fixture_assertions("must-not-appear")
	production_assert_link(
		t, "Range_Int8",
		func() { builder.Range_Int8(1, 0, 2) },
		func() { builder.Range_Int8(-1, 0, 2) },
	)
	production_assert_link(
		t, "Range_Holed_Int8",
		func() { builder.Range_Holed_Int8(1, 0, 3, 2, 2, 2, 2) },
		func() { builder.Range_Holed_Int8(2, 0, 3, 2, 2, 2, 2) },
	)
	production_assert_link(
		t, "Enum_Int8",
		func() { builder.Enum_Int8(1, 1, 2) },
		func() { builder.Enum_Int8(3, 1, 2) },
	)
	production_assert_link(
		t, "Enum_3_Int8",
		func() { builder.Enum_3_Int8(2, 1, 2, 3) },
		func() { builder.Enum_3_Int8(4, 1, 2, 3) },
	)
	production_assert_link(
		t, "Enum_4_Int8",
		func() { builder.Enum_4_Int8(3, 1, 2, 3, 4) },
		func() { builder.Enum_4_Int8(5, 1, 2, 3, 4) },
	)
}

// Test_Production_Int16_Assertions_Enforce_At_The_Violating_Link keeps every int16 family eager.
func Test_Production_Int16_Assertions_Enforce_At_The_Violating_Link(t *testing.T) {
	builder := fixture_assertions("must-not-appear")
	production_assert_link(
		t, "Range_Int16",
		func() { builder.Range_Int16(1, 0, 2) },
		func() { builder.Range_Int16(-1, 0, 2) },
	)
	production_assert_link(
		t, "Range_Holed_Int16",
		func() { builder.Range_Holed_Int16(1, 0, 3, 2, 2, 2, 2) },
		func() { builder.Range_Holed_Int16(2, 0, 3, 2, 2, 2, 2) },
	)
	production_assert_link(
		t, "Enum_Int16",
		func() { builder.Enum_Int16(1, 1, 2) },
		func() { builder.Enum_Int16(3, 1, 2) },
	)
	production_assert_link(
		t, "Enum_3_Int16",
		func() { builder.Enum_3_Int16(2, 1, 2, 3) },
		func() { builder.Enum_3_Int16(4, 1, 2, 3) },
	)
	production_assert_link(
		t, "Enum_4_Int16",
		func() { builder.Enum_4_Int16(3, 1, 2, 3, 4) },
		func() { builder.Enum_4_Int16(5, 1, 2, 3, 4) },
	)
}

// Test_Production_Int32_Assertions_Enforce_At_The_Violating_Link keeps every int32 family eager.
func Test_Production_Int32_Assertions_Enforce_At_The_Violating_Link(t *testing.T) {
	builder := fixture_assertions("must-not-appear")
	production_assert_link(
		t, "Range_Int32",
		func() { builder.Range_Int32(1, 0, 2) },
		func() { builder.Range_Int32(-1, 0, 2) },
	)
	production_assert_link(
		t, "Range_Holed_Int32",
		func() { builder.Range_Holed_Int32(1, 0, 3, 2, 2, 2, 2) },
		func() { builder.Range_Holed_Int32(2, 0, 3, 2, 2, 2, 2) },
	)
	production_assert_link(
		t, "Enum_Int32",
		func() { builder.Enum_Int32(1, 1, 2) },
		func() { builder.Enum_Int32(3, 1, 2) },
	)
	production_assert_link(
		t, "Enum_3_Int32",
		func() { builder.Enum_3_Int32(2, 1, 2, 3) },
		func() { builder.Enum_3_Int32(4, 1, 2, 3) },
	)
	production_assert_link(
		t, "Enum_4_Int32",
		func() { builder.Enum_4_Int32(3, 1, 2, 3, 4) },
		func() { builder.Enum_4_Int32(5, 1, 2, 3, 4) },
	)
}

// Test_Production_Int64_Assertions_Enforce_At_The_Violating_Link keeps every int64 family eager.
func Test_Production_Int64_Assertions_Enforce_At_The_Violating_Link(t *testing.T) {
	builder := fixture_assertions("must-not-appear")
	production_assert_link(
		t, "Range_Int64",
		func() { builder.Range_Int64(1, 0, 2) },
		func() { builder.Range_Int64(-1, 0, 2) },
	)
	production_assert_link(
		t, "Range_Holed_Int64",
		func() { builder.Range_Holed_Int64(1, 0, 3, 2, 2, 2, 2) },
		func() { builder.Range_Holed_Int64(2, 0, 3, 2, 2, 2, 2) },
	)
	production_assert_link(
		t, "Enum_Int64",
		func() { builder.Enum_Int64(1, 1, 2) },
		func() { builder.Enum_Int64(3, 1, 2) },
	)
	production_assert_link(
		t, "Enum_3_Int64",
		func() { builder.Enum_3_Int64(2, 1, 2, 3) },
		func() { builder.Enum_3_Int64(4, 1, 2, 3) },
	)
	production_assert_link(
		t, "Enum_4_Int64",
		func() { builder.Enum_4_Int64(3, 1, 2, 3, 4) },
		func() { builder.Enum_4_Int64(5, 1, 2, 3, 4) },
	)
}

// Test_Production_Uint_Assertions_Enforce_At_The_Violating_Link keeps every uint family eager.
func Test_Production_Uint_Assertions_Enforce_At_The_Violating_Link(t *testing.T) {
	builder := fixture_assertions("must-not-appear")
	production_assert_link(
		t, "Range_Uint",
		func() { builder.Range_Uint(1, 0, 2) },
		func() { builder.Range_Uint(3, 0, 2) },
	)
	production_assert_link(
		t, "Range_Holed_Uint",
		func() { builder.Range_Holed_Uint(1, 0, 3, 2, 2, 2) },
		func() { builder.Range_Holed_Uint(2, 0, 3, 2, 2, 2) },
	)
	production_assert_link(
		t, "Enum_Uint",
		func() { builder.Enum_Uint(1, 1, 2) },
		func() { builder.Enum_Uint(3, 1, 2) },
	)
	production_assert_link(
		t, "Enum_3_Uint",
		func() { builder.Enum_3_Uint(2, 1, 2, 3) },
		func() { builder.Enum_3_Uint(4, 1, 2, 3) },
	)
	production_assert_link(
		t, "Enum_4_Uint",
		func() { builder.Enum_4_Uint(3, 1, 2, 3, 4) },
		func() { builder.Enum_4_Uint(5, 1, 2, 3, 4) },
	)
}

// Test_Production_Uint8_Assertions_Enforce_At_The_Violating_Link keeps every uint8 family eager.
func Test_Production_Uint8_Assertions_Enforce_At_The_Violating_Link(t *testing.T) {
	builder := fixture_assertions("must-not-appear")
	production_assert_link(
		t, "Range_Uint8",
		func() { builder.Range_Uint8(1, 0, 2) },
		func() { builder.Range_Uint8(3, 0, 2) },
	)
	production_assert_link(
		t, "Range_Holed_Uint8",
		func() { builder.Range_Holed_Uint8(1, 0, 3, 2, 2, 2) },
		func() { builder.Range_Holed_Uint8(2, 0, 3, 2, 2, 2) },
	)
	production_assert_link(
		t, "Enum_Uint8",
		func() { builder.Enum_Uint8(1, 1, 2) },
		func() { builder.Enum_Uint8(3, 1, 2) },
	)
	production_assert_link(
		t, "Enum_3_Uint8",
		func() { builder.Enum_3_Uint8(2, 1, 2, 3) },
		func() { builder.Enum_3_Uint8(4, 1, 2, 3) },
	)
	production_assert_link(
		t, "Enum_4_Uint8",
		func() { builder.Enum_4_Uint8(3, 1, 2, 3, 4) },
		func() { builder.Enum_4_Uint8(5, 1, 2, 3, 4) },
	)
}

// Test_Production_Uint16_Assertions_Enforce_At_The_Violating_Link keeps every uint16 family eager.
func Test_Production_Uint16_Assertions_Enforce_At_The_Violating_Link(t *testing.T) {
	builder := fixture_assertions("must-not-appear")
	production_assert_link(
		t, "Range_Uint16",
		func() { builder.Range_Uint16(1, 0, 2) },
		func() { builder.Range_Uint16(3, 0, 2) },
	)
	production_assert_link(
		t, "Range_Holed_Uint16",
		func() { builder.Range_Holed_Uint16(1, 0, 3, 2, 2, 2) },
		func() { builder.Range_Holed_Uint16(2, 0, 3, 2, 2, 2) },
	)
	production_assert_link(
		t, "Enum_Uint16",
		func() { builder.Enum_Uint16(1, 1, 2) },
		func() { builder.Enum_Uint16(3, 1, 2) },
	)
	production_assert_link(
		t, "Enum_3_Uint16",
		func() { builder.Enum_3_Uint16(2, 1, 2, 3) },
		func() { builder.Enum_3_Uint16(4, 1, 2, 3) },
	)
	production_assert_link(
		t, "Enum_4_Uint16",
		func() { builder.Enum_4_Uint16(3, 1, 2, 3, 4) },
		func() { builder.Enum_4_Uint16(5, 1, 2, 3, 4) },
	)
}

// Test_Production_Uint32_Assertions_Enforce_At_The_Violating_Link keeps every uint32 family eager.
func Test_Production_Uint32_Assertions_Enforce_At_The_Violating_Link(t *testing.T) {
	builder := fixture_assertions("must-not-appear")
	production_assert_link(
		t, "Range_Uint32",
		func() { builder.Range_Uint32(1, 0, 2) },
		func() { builder.Range_Uint32(3, 0, 2) },
	)
	production_assert_link(
		t, "Range_Holed_Uint32",
		func() { builder.Range_Holed_Uint32(1, 0, 3, 2, 2, 2) },
		func() { builder.Range_Holed_Uint32(2, 0, 3, 2, 2, 2) },
	)
	production_assert_link(
		t, "Enum_Uint32",
		func() { builder.Enum_Uint32(1, 1, 2) },
		func() { builder.Enum_Uint32(3, 1, 2) },
	)
	production_assert_link(
		t, "Enum_3_Uint32",
		func() { builder.Enum_3_Uint32(2, 1, 2, 3) },
		func() { builder.Enum_3_Uint32(4, 1, 2, 3) },
	)
	production_assert_link(
		t, "Enum_4_Uint32",
		func() { builder.Enum_4_Uint32(3, 1, 2, 3, 4) },
		func() { builder.Enum_4_Uint32(5, 1, 2, 3, 4) },
	)
}

// Test_Production_Uint64_Assertions_Enforce_At_The_Violating_Link keeps every uint64 family eager.
func Test_Production_Uint64_Assertions_Enforce_At_The_Violating_Link(t *testing.T) {
	builder := fixture_assertions("must-not-appear")
	production_assert_link(
		t, "Range_Uint64",
		func() { builder.Range_Uint64(1, 0, 2) },
		func() { builder.Range_Uint64(3, 0, 2) },
	)
	production_assert_link(
		t, "Range_Holed_Uint64",
		func() { builder.Range_Holed_Uint64(1, 0, 3, 2, 2, 2) },
		func() { builder.Range_Holed_Uint64(2, 0, 3, 2, 2, 2) },
	)
	production_assert_link(
		t, "Enum_Uint64",
		func() { builder.Enum_Uint64(1, 1, 2) },
		func() { builder.Enum_Uint64(3, 1, 2) },
	)
	production_assert_link(
		t, "Enum_3_Uint64",
		func() { builder.Enum_3_Uint64(2, 1, 2, 3) },
		func() { builder.Enum_3_Uint64(4, 1, 2, 3) },
	)
	production_assert_link(
		t, "Enum_4_Uint64",
		func() { builder.Enum_4_Uint64(3, 1, 2, 3, 4) },
		func() { builder.Enum_4_Uint64(5, 1, 2, 3, 4) },
	)
}

// Test_Production_Assertions_Retain_Only_Enforcement protects the zero builder from regressing.
func Test_Production_Assertions_Retain_Only_Enforcement(t *testing.T) {
	recorder := &invariant.Recorder{Is_Test: true}
	builder := invariant.Recorder_Tree(recorder, Fixture_Subject(0), "must-not-appear")
	if builder != (invariant.Assertion_Builder{}) {
		t.Fatalf("builder = %+v, want zero value", builder)
	}
	builder.Sometimes(false, "ignored").Ensure()
	if count := production_event_count(&recorder.Events); count != 0 {
		t.Fatalf("events = %d, want none", count)
	}
}

// Test_Production_Always_Remains_Eager keeps the bare guard independent of fluent-link timing.
func Test_Production_Always_Remains_Eager(t *testing.T) {
	message := production_panic_text(func() {
		invariant.Recorder_Always(&invariant.Recorder{}, false, "always identity")
	})
	want := invariant.ASSERTION_FAILURE_MESSAGE_PREFIX + "always identity"
	if !strings.Contains(message, want) {
		t.Fatalf("panic = %q", message)
	}
}

// Test_Production_Panics_Use_Fixed_Identity_Without_Namespace keeps dynamic state out of panics.
func Test_Production_Panics_Use_Fixed_Identity_Without_Namespace(t *testing.T) {
	builder := fixture_assertions("must-not-appear")
	tests := []struct {
		Name     string
		Action   func()
		Identity string
	}{
		{
			"range minimum",
			func() { builder.Range_Int(-1, 0, 2) },
			invariant.RANGE_GUARD_MINIMUM,
		},
		{
			"range maximum",
			func() { builder.Range_Int(3, 0, 2) },
			invariant.RANGE_GUARD_MAXIMUM,
		},
		{
			"range hole",
			func() { builder.Range_Holed_Int(1, 0, 2, 1, 1, 1, 1) },
			"Range value is excluded",
		},
		{
			"enum member",
			func() { builder.Enum_Int(3, 1, 2) },
			invariant.ENUM_GUARD_MEMBER,
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			message := production_panic_text(test.Action)
			if !strings.HasPrefix(message, invariant.ASSERTION_FAILURE_MESSAGE_PREFIX) {
				t.Fatalf("panic = %q, want assertion prefix", message)
			}
			if !strings.Contains(message, test.Identity) {
				t.Fatalf("panic = %q, want identity %q", message, test.Identity)
			}
			if strings.Contains(message, "must-not-appear") {
				t.Fatalf("panic retained namespace state: %q", message)
			}
		})
	}
}

func production_assert_link(
	t *testing.T, name string, valid func(), invalid func(),
) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		if message := production_panic_text(valid); message != "" {
			t.Fatalf("valid value panicked: %q", message)
		}
		if message := production_panic_text(invalid); message == "" {
			t.Fatal("violating link did not panic eagerly")
		}
	})
}

func production_panic_text(action func()) (message string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			message = fmt.Sprint(recovered)
		}
	}()
	action()
	return ""
}

func production_event_count(events *sync.Map) (count int) {
	events.Range(func(_, _ any) (continue_iteration bool) {
		count++
		return true
	})
	return count
}
