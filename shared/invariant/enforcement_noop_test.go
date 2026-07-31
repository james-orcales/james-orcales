//go:build invariant_noop && !invariant_disable_coverage && !prd && !prod && !production

package invariant_test

import (
	"testing"

	"local/james-orcales/shared/invariant"
)

// Test_Noop_Assertions_Are_Completely_Inert exercises every invalid link because a partial noop
// would make benchmark comparisons depend on which primitive happens to reach a hot path.
func Test_Noop_Assertions_Are_Completely_Inert(t *testing.T) {
	recorder := &invariant.Recorder{Is_Test: true}
	invariant.Recorder_Always(recorder, false, "ignored")
	builder := invariant.Recorder_Assertions(recorder, "ignored")
	if builder != (invariant.Assertion_Builder{}) {
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

func noop_event_count(events interface{ Range(func(any, any) bool) }) (count int) {
	events.Range(func(_, _ any) (continue_iteration bool) {
		count++
		return true
	})
	return count
}
