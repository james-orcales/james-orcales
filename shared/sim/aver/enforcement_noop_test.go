//go:build invariant_noop && !invariant_disable_coverage && !prd && !prod && !production

package aver_test

import (
	"fmt"
	"io/fs"
	"testing"

	"local/james-orcales/shared/sim/aver"
	"local/james-orcales/shared/testify"
)

// Fixture_Subject stands in for a bundle subject where the test drives the builder directly.
// This build resolves no plan, so the chain type only has to compile.
type Fixture_Subject int

// Any read fails so scan-and-discard cannot masquerade as disabled registration.
type noop_registration_file_system struct{}

func (noop_registration_file_system) Open(string) (fs.File, error) {
	panic("noop registration read source")
}

// Opens a chain on a plan-free recorder over the fixture subject, keeping call sites short.
func fixture_assertions(namespace aver.Namespace) (builder aver.Assertion_Builder) {
	return aver.Recorder_Tree(&aver.Recorder{}, Fixture_Subject(0), namespace)
}

// Test_Optimized_Noop_Enforcement_Matches_Literal_Reference prevents the benchmark lower bound
// from hiding any assertion behavior behind its signature-identical surface.
func Test_Optimized_Noop_Enforcement_Matches_Literal_Reference(t *testing.T) {
	noop_pair(
		t, "always",
		func() { aver.Recorder_Always(&aver.Recorder{}, false, "ignored") },
		func() { noop_reference_always(false, "ignored") },
	)
	builder := fixture_assertions("ignored")
	for value := -2; value <= 6; value++ {
		noop_pair(
			t, fmt.Sprintf("value=%d", value),
			func() {
				builder.Sometimes(false, "ignored").
					Range_Int(value, 0, 4).
					Range_Holed_Int(value, 0, 4, 2, 2, 2, 2).
					Enum_Int(value, 1, 3).
					Enum_3_Int(value, 1, 3, 5).
					Enum_4_Int(value, 1, 2, 3, 5).
					Ensure()
			},
			func() {
				noop_reference_sometimes(false, "ignored")
				noop_reference_range_int(value, 0, 4)
				noop_reference_range_holed_int(value, 0, 4, 2, 2, 2, 2)
				noop_reference_enum_int(value, 1, 3)
				noop_reference_enum_3_int(value, 1, 3, 5)
				noop_reference_enum_4_int(value, 1, 2, 3, 5)
				noop_reference_ensure()
			},
		)
	}
}

func noop_pair(t *testing.T, name string, optimized func(), reference func()) {
	t.Helper()
	optimized_panic := noop_panic_text(optimized)
	reference_panic := noop_panic_text(reference)
	if optimized_panic != reference_panic {
		t.Fatalf("%s: optimized panic = %q, reference panic = %q",
			name, optimized_panic, reference_panic)
	}
}

func noop_reference_always(condition bool, message string) {
	return
}

func noop_reference_sometimes(condition bool, message string) {
	return
}

func noop_reference_range_int(value int, minimum int, maximum int) {
	return
}

func noop_reference_range_holed_int(
	value int, minimum int, maximum int, hole_1 int, hole_2 int, hole_3 int, hole_4 int,
) {
	return
}

func noop_reference_enum_int(value int, first int, second int) {
	return
}

func noop_reference_enum_3_int(value int, first int, second int, third int) {
	return
}

func noop_reference_enum_4_int(value int, first int, second int, third int, fourth int) {
	return
}

func noop_reference_ensure() {
	return
}

func noop_panic_text(action func()) (message string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			message = fmt.Sprint(recovered)
		}
	}()
	action()
	return ""
}

// Test_Noop_Assertions_Are_Completely_Inert exercises every invalid link because a partial noop
// would make benchmark comparisons depend on which primitive happens to reach a hot path.
func Test_Noop_Assertions_Are_Completely_Inert(t *testing.T) {
	recorder := &aver.Recorder{Is_Test: true}
	aver.Recorder_Always(recorder, false, "ignored")
	builder := aver.Recorder_Tree(recorder, Fixture_Subject(0), "ignored")
	if builder != (aver.Assertion_Builder{}) {
		t.Fatalf("builder = %+v, want zero value", builder)
	}
	builder = builder.Sometimes(false, "ignored")
	builder = builder.Range_Int(-1, 0, 1)
	builder = builder.Range_Int8(-1, 0, 1)
	builder = builder.Range_Int16(-1, 0, 1)
	builder = builder.Range_Int32(-1, 0, 1)
	builder = builder.Range_Int64(-1, 0, 1)
	builder = builder.Range_Uint(2, 0, 1)
	builder = builder.Range_Uint8(2, 0, 1)
	builder = builder.Range_Uint16(2, 0, 1)
	builder = builder.Range_Uint32(2, 0, 1)
	builder = builder.Range_Uint64(2, 0, 1)
	builder = builder.Range_Holed_Int(1, 0, 2, 1, 1, 1, 1)
	builder = builder.Range_Holed_Int8(1, 0, 2, 1, 1, 1, 1)
	builder = builder.Range_Holed_Int16(1, 0, 2, 1, 1, 1, 1)
	builder = builder.Range_Holed_Int32(1, 0, 2, 1, 1, 1, 1)
	builder = builder.Range_Holed_Int64(1, 0, 2, 1, 1, 1, 1)
	builder = builder.Range_Holed_Uint(1, 0, 2, 1, 1, 1)
	builder = builder.Range_Holed_Uint8(1, 0, 2, 1, 1, 1)
	builder = builder.Range_Holed_Uint16(1, 0, 2, 1, 1, 1)
	builder = builder.Range_Holed_Uint32(1, 0, 2, 1, 1, 1)
	builder = builder.Range_Holed_Uint64(1, 0, 2, 1, 1, 1)
	builder = builder.Enum_Int(3, 1, 2)
	builder = builder.Enum_Int8(3, 1, 2)
	builder = builder.Enum_Int16(3, 1, 2)
	builder = builder.Enum_Int32(3, 1, 2)
	builder = builder.Enum_Int64(3, 1, 2)
	builder = builder.Enum_Uint(3, 1, 2)
	builder = builder.Enum_Uint8(3, 1, 2)
	builder = builder.Enum_Uint16(3, 1, 2)
	builder = builder.Enum_Uint32(3, 1, 2)
	builder = builder.Enum_Uint64(3, 1, 2)
	builder = builder.Enum_3_Int(4, 1, 2, 3)
	builder = builder.Enum_3_Int8(4, 1, 2, 3)
	builder = builder.Enum_3_Int16(4, 1, 2, 3)
	builder = builder.Enum_3_Int32(4, 1, 2, 3)
	builder = builder.Enum_3_Int64(4, 1, 2, 3)
	builder = builder.Enum_3_Uint(4, 1, 2, 3)
	builder = builder.Enum_3_Uint8(4, 1, 2, 3)
	builder = builder.Enum_3_Uint16(4, 1, 2, 3)
	builder = builder.Enum_3_Uint32(4, 1, 2, 3)
	builder = builder.Enum_3_Uint64(4, 1, 2, 3)
	builder = builder.Enum_4_Int(5, 1, 2, 3, 4)
	builder = builder.Enum_4_Int8(5, 1, 2, 3, 4)
	builder = builder.Enum_4_Int16(5, 1, 2, 3, 4)
	builder = builder.Enum_4_Int32(5, 1, 2, 3, 4)
	builder = builder.Enum_4_Int64(5, 1, 2, 3, 4)
	builder = builder.Enum_4_Uint(5, 1, 2, 3, 4)
	builder = builder.Enum_4_Uint8(5, 1, 2, 3, 4)
	builder = builder.Enum_4_Uint16(5, 1, 2, 3, 4)
	builder = builder.Enum_4_Uint32(5, 1, 2, 3, 4)
	builder = builder.Enum_4_Uint64(5, 1, 2, 3, 4)
	builder.Ensure()
	if count := noop_event_count(&recorder.Events); count != 0 {
		t.Fatalf("events = %d, want none", count)
	}
}

// Registration must disappear with enforcement, or benchmark startup still pays source-analysis
// cost for obligations this build can never observe.
func Test_Noop_Assertion_Registration_Is_Inert(t *testing.T) {
	recorder := &aver.Recorder{
		File_System:         noop_registration_file_system{},
		Exit:                func(int) {},
		Is_Test:             true,
		Packages_To_Analyze: []string{"/existing"},
	}
	testify.Not_Panics(t, func() {
		aver.Recorder_Register_Packages_For_Analysis(recorder, "/replacement")
	})
	testify.Equal(t, 0, noop_event_count(&recorder.Events))
	testify.Nil(t, recorder.Assertion_Plans)
	testify.Equal(t, []string{"/existing"}, recorder.Packages_To_Analyze)
}

func noop_event_count(events interface{ Range(func(any, any) bool) }) (count int) {
	events.Range(func(_, _ any) (continue_iteration bool) {
		count++
		return true
	})
	return count
}
