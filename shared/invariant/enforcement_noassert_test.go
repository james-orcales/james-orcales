//go:build noassert

package invariant_test

import (
	"testing"

	"local/james-orcales/shared/invariant"
)

// Test_Noassert_Builder_Is_Inert prevents the disabled surface from retaining side effects.
func Test_Noassert_Builder_Is_Inert(t *testing.T) {
	recorder := &invariant.Recorder{Is_Test: true}
	invariant.Recorder_Always(recorder, false, "ignored")
	invariant.Recorder_Assertions(recorder, "ignored").
		Sometimes(true, "axis").
		Range_Int(99, 0, 1).
		Range_Holed_Int(99, 0, 1, 0, 0, 0, 0).
		Enum_Int(99, 1, 2).
		Enum_3_Int(99, 1, 2, 3).
		Enum_4_Int(99, 1, 2, 3, 4).
		Ensure()
	if count := event_count_noassert(&recorder.Events); count != 0 {
		t.Fatalf("events = %d", count)
	}
}

func event_count_noassert(events interface{ Range(func(any, any) bool) }) (count int) {
	events.Range(func(_, _ any) (continue_iteration bool) {
		count++
		return true
	})
	return count
}
