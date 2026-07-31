//go:build !invariant_disable_coverage && !prd && !prod && !production && !invariant_noop

package invariant

import (
	"fmt"
	"strings"
	"testing"
	"unsafe"
)

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
	builder := Recorder_Assertions(&Recorder{}, "ordinary").
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
		{Recorder_Assertions(&Recorder{}, "lower").Range_Int(-1, 0, 3), "below min"},
		{Recorder_Assertions(&Recorder{}, "upper").Range_Int(4, 0, 3), "exceeds max"},
		{Recorder_Assertions(&Recorder{}, "hole").
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
	builder := Recorder_Assertions(&Recorder{}, "enum").Enum_4_Int(5, 1, 2, 3, 4)
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
		Recorder_Assertions(recorder, "benchmark").Range_Int(5, 0, 10).Ensure()
	}
}

// Benchmark_Assertions_Immediate_Panic_Control measures the test-only timing alternative.
func Benchmark_Assertions_Immediate_Panic_Control(benchmark *testing.B) {
	recorder := &Recorder{}
	for benchmark.Loop() {
		builder := Recorder_Assertions(recorder, "benchmark")
		builder = assertion_range_int_immediate_control(builder, 5, 0, 10)
		assertion_ensure_immediate_control(builder)
	}
}
