//go:build !invariant_disable_coverage && !prd && !prod && !production && !invariant_noop

package invariant

import (
	"fmt"
	"strings"
	"testing"
	"unsafe"
)

// Fixture_Subject stands in for a bundle subject where the test drives the builder directly. These
// tests use a plan-free Recorder, so the chain type never reaches a lookup.
type Fixture_Subject int

// Opens a chain on a plan-free recorder over the fixture subject, keeping call sites short.
func fixture_assertions(namespace Namespace) (builder Assertion_Builder) {
	return Recorder_Tree(&Recorder{}, Fixture_Subject(0), namespace)
}

// Test_Optimized_Full_Enforcement_Matches_Literal_Reference keeps packed deferred state honest by
// comparing its externally visible result with direct, field-by-field test code.
func Test_Optimized_Full_Enforcement_Matches_Literal_Reference(t *testing.T) {
	const NAMESPACE = Namespace("pair")
	for _, condition := range []bool{false, true} {
		assertion_pair(
			t, fmt.Sprintf("always=%t", condition),
			func() { Recorder_Always(&Recorder{}, condition, "identity") },
			func() { full_reference_always(condition, "identity") },
		)
	}
	assertion_pair(
		t, "sometimes",
		func() {
			fixture_assertions(NAMESPACE).
				Sometimes(false, "axis").Ensure()
		},
		func() {
			full_reference_sometimes(false, "axis")
			full_reference_ensure()
		},
	)
	for value := -2; value <= 6; value++ {
		assertion_pair(
			t, fmt.Sprintf("range=%d", value),
			func() {
				fixture_assertions(NAMESPACE).
					Range_Int(value, 0, 4).Ensure()
			},
			func() { full_reference_range_int(NAMESPACE, value, 0, 4) },
		)
		assertion_pair(
			t, fmt.Sprintf("range-holed=%d", value),
			func() {
				fixture_assertions(NAMESPACE).
					Range_Holed_Int(value, 0, 4, 2, 2, 2, 2).Ensure()
			},
			func() {
				full_reference_range_holed_int(NAMESPACE, value, 0, 4, 2, 2, 2, 2)
			},
		)
		assertion_pair(
			t, fmt.Sprintf("enum=%d", value),
			func() {
				fixture_assertions(NAMESPACE).
					Enum_Int(value, 1, 3).Ensure()
			},
			func() { full_reference_enum_int(NAMESPACE, value, 1, 3) },
		)
		assertion_pair(
			t, fmt.Sprintf("enum-3=%d", value),
			func() {
				fixture_assertions(NAMESPACE).
					Enum_3_Int(value, 1, 3, 5).Ensure()
			},
			func() { full_reference_enum_3_int(NAMESPACE, value, 1, 3, 5) },
		)
		assertion_pair(
			t, fmt.Sprintf("enum-4=%d", value),
			func() {
				fixture_assertions(NAMESPACE).
					Enum_4_Int(value, 1, 2, 3, 5).Ensure()
			},
			func() { full_reference_enum_4_int(NAMESPACE, value, 1, 2, 3, 5) },
		)
	}
}

func assertion_pair(t *testing.T, name string, optimized func(), reference func()) {
	t.Helper()
	optimized_panic := panic_message(optimized)
	reference_panic := panic_message(reference)
	if optimized_panic != reference_panic {
		t.Fatalf("%s: optimized panic = %q, reference panic = %q",
			name, optimized_panic, reference_panic)
	}
}

func full_reference_always(condition bool, message string) {
	if !condition {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX + message +
			"  Always — condition was false: " + fmt.Sprint(condition))
	}
}

func full_reference_sometimes(condition bool, message string) {
	return
}

func full_reference_ensure() {
	return
}

func full_reference_range_int(namespace Namespace, value int, minimum int, maximum int) {
	prefix := ASSERTION_FAILURE_MESSAGE_PREFIX + string(namespace) + " · "
	if value < minimum {
		panic(prefix + RANGE_GUARD_MINIMUM + "  value below min: " + fmt.Sprint(value))
	}
	if value > maximum {
		panic(prefix + RANGE_GUARD_MAXIMUM + "  value exceeds max: " + fmt.Sprint(value))
	}
}

func full_reference_range_holed_int(
	namespace Namespace, value int, minimum int, maximum int,
	hole_1 int, hole_2 int, hole_3 int, hole_4 int,
) {
	full_reference_range_int(namespace, value, minimum, maximum)
	switch value {
	case hole_1, hole_2, hole_3, hole_4:
		prefix := ASSERTION_FAILURE_MESSAGE_PREFIX + string(namespace) + " · "
		panic(prefix + "Range value is excluded: " + fmt.Sprint(value))
	}
}

func full_reference_enum_int(namespace Namespace, value int, first int, second int) {
	switch value {
	case first, second:
		return
	}
	prefix := ASSERTION_FAILURE_MESSAGE_PREFIX + string(namespace) + " · "
	panic(prefix + ENUM_GUARD_MEMBER + "  value is not a member: " + fmt.Sprint(value))
}

func full_reference_enum_3_int(
	namespace Namespace, value int, first int, second int, third int,
) {
	switch value {
	case first, second, third:
		return
	}
	prefix := ASSERTION_FAILURE_MESSAGE_PREFIX + string(namespace) + " · "
	panic(prefix + ENUM_GUARD_MEMBER + "  value is not a member: " + fmt.Sprint(value))
}

func full_reference_enum_4_int(
	namespace Namespace, value int, first int, second int, third int, fourth int,
) {
	switch value {
	case first, second, third, fourth:
		return
	}
	prefix := ASSERTION_FAILURE_MESSAGE_PREFIX + string(namespace) + " · "
	panic(prefix + ENUM_GUARD_MEMBER + "  value is not a member: " + fmt.Sprint(value))
}

// Test_Assertion_Builder_Fits_Three_Words prevents recording-only state from making every fluent
// return copy more than the ordinary enforcement path can justify.
func Test_Assertion_Builder_Fits_Three_Words(t *testing.T) {
	size := unsafe.Sizeof(Assertion_Builder{})
	if size > 3*unsafe.Sizeof(uintptr(0)) {
		t.Fatalf("builder size = %d bytes, want at most three words", size)
	}
}

// Test_Assertions_Non_Recording_Success_Does_Not_Construct_Observations protects the ordinary full
// build from paying for the registration-owned representation.
func Test_Assertions_Non_Recording_Success_Does_Not_Construct_Observations(t *testing.T) {
	builder := fixture_assertions("ordinary").
		Sometimes(true, "axis").
		Range_Holed_Int(3, 0, 10, 4, 4, 4, 4).
		Enum_3_Int(3, 1, 2, 3)
	if ordinal := builder.assertion_ordinal(); ordinal != 0 {
		t.Fatalf("ordinal = %d, want no recording progress", ordinal)
	}
	for ordinal := uint8(0); ordinal < ASSERTION_LINKS_MAX; ordinal++ {
		if builder.assertion_observed(ordinal) {
			t.Fatalf("observation %d was constructed", ordinal)
		}
	}
	builder.Ensure()
}

// Test_Assertions_Non_Recording_Range_Failures_Are_Deferred keeps the cheap full-build head from
// changing the fluent chain's atomic failure boundary.
func Test_Assertions_Non_Recording_Range_Failures_Are_Deferred(t *testing.T) {
	failures := []struct {
		Builder Assertion_Builder
		Want    string
	}{
		{fixture_assertions("lower").Range_Int(-1, 0, 3), "below min"},
		{fixture_assertions("upper").Range_Int(4, 0, 3), "exceeds max"},
		{fixture_assertions("hole").
			Range_Holed_Int(2, 0, 3, 2, 2, 2, 2), "excluded"},
	}
	for _, failure := range failures {
		message := panic_message(failure.Builder.Ensure)
		if !strings.Contains(message, failure.Want) {
			t.Fatalf("panic = %q, want %q", message, failure.Want)
		}
	}
}

// Test_Assertions_Non_Recording_Enum_Failure_Is_Deferred keeps membership on the enforcement path
// while registration alone owns distinct-member expansion.
func Test_Assertions_Non_Recording_Enum_Failure_Is_Deferred(t *testing.T) {
	builder := fixture_assertions("enum").Enum_4_Int(5, 1, 2, 3, 4)
	if message := panic_message(builder.Ensure); !strings.Contains(message, "not a member") {
		t.Fatalf("panic = %q", message)
	}
}

func panic_message(action func()) (message string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			message = fmt.Sprint(recovered)
		}
	}()
	action()
	return ""
}

// Moving the violation boundary stays isolated here because deferred failure remains the contract.
func assertion_range_int_immediate_control(
	builder Assertion_Builder, value int, minimum int, maximum int, excluded ...int,
) (next Assertion_Builder) {
	if value < minimum {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX + "immediate value below min")
	}
	if value > maximum {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX + "immediate value exceeds max")
	}
	for _, hole := range excluded {
		if value == hole {
			panic(ASSERTION_FAILURE_MESSAGE_PREFIX + "immediate value is excluded")
		}
	}
	if builder.assertion_recording() {
		return assertion_range_slow(builder, value, minimum, maximum)
	}
	return builder
}

func assertion_ensure_immediate_control(builder Assertion_Builder) {
	if builder.assertion_recording() {
		assertion_ensure(&builder)
	}
}

// Benchmark_Assertions_Deferred_Panic_Control measures the retained contract.
func Benchmark_Assertions_Deferred_Panic_Control(benchmark *testing.B) {
	recorder := &Recorder{}
	for benchmark.Loop() {
		Recorder_Tree(recorder, Fixture_Subject(0), "benchmark").Range_Int(5, 0, 10).
			Ensure()
	}
}

// Benchmark_Assertions_Immediate_Panic_Control measures the test-only timing alternative.
func Benchmark_Assertions_Immediate_Panic_Control(benchmark *testing.B) {
	recorder := &Recorder{}
	for benchmark.Loop() {
		builder := Recorder_Tree(recorder, Fixture_Subject(0), "benchmark")
		builder = assertion_range_int_immediate_control(builder, 5, 0, 10)
		assertion_ensure_immediate_control(builder)
	}
}
