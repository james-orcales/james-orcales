//go:build noassert

package invariant_test

import (
	"testing"

	"local/james-orcales/shared/invariant"
)

// The noassert build is a production mode, not a test mode: the rest of this suite asserts that
// enforcement fires, so it is meaningful only in the default build. These tests are the dual's own
// contract and run under `go test -tags noassert -run Test_Noassert`.

// Test_Noassert_Always_Is_Silent prevents the eager guard from surviving the dual: a false Always
// is the one failure that fires without a chain, so it is the first thing that must go quiet.
func Test_Noassert_Always_Is_Silent(t *testing.T) {
	recorder := &invariant.Recorder{}
	if message := panic_text(func() {
		invariant.Recorder_Always(recorder, false, "guard")
	}); message != "" {
		t.Fatalf("panic = %q, want none", message)
	}
}

// Test_Noassert_Constraint_Is_Silent prevents a carved cell from still panicking. A matching
// Impossible is the chain's fatal path, so it stands in for every constraint the dual drops.
func Test_Noassert_Constraint_Is_Silent(t *testing.T) {
	recorder := &invariant.Recorder{}
	if message := panic_text(func() {
		invariant.Recorder_Dot_Product(recorder, "noassert.constraint").
			Sometimes(true, "left").
			Sometimes(true, "right").
			Impossible("both cannot hold",
				invariant.Event_True("left"),
				invariant.Event_True("right")).
			Ensure()
	}); message != "" {
		t.Fatalf("panic = %q, want none", message)
	}
}

// Test_Noassert_Malformed_Chain_Is_Silent prevents the dual from retaining shape validation. An
// axis-less chain and a reference resolving to no sibling are both Ensure-time panics in the
// default build; neither may survive a build that promises to check nothing.
func Test_Noassert_Malformed_Chain_Is_Silent(t *testing.T) {
	recorder := &invariant.Recorder{}
	if message := panic_text(func() {
		invariant.Recorder_Dot_Product(recorder, "noassert.empty").Ensure()
	}); message != "" {
		t.Fatalf("axis-less chain: panic = %q, want none", message)
	}
	if message := panic_text(func() {
		invariant.Recorder_Dot_Product(recorder, "noassert.dangling").
			Sometimes(true, "present").
			Impossible("names nothing", invariant.Event_True("absent")).
			Ensure()
	}); message != "" {
		t.Fatalf("dangling reference: panic = %q, want none", message)
	}
}

// Test_Noassert_Preset_Guard_Is_Silent prevents the typed presets from keeping their bound checks.
// The presets expand into ordinary guards, so an out-of-range value and a non-member cover the
// whole Range_TYPE / Enum_TYPE family.
func Test_Noassert_Preset_Guard_Is_Silent(t *testing.T) {
	recorder := &invariant.Recorder{}
	if message := panic_text(func() {
		invariant.Recorder_Dot_Product(recorder, "noassert.range").
			Range_Int(99, 0, 10).
			Ensure()
	}); message != "" {
		t.Fatalf("out-of-range: panic = %q, want none", message)
	}
	if message := panic_text(func() {
		invariant.Recorder_Dot_Product(recorder, "noassert.enum").
			Enum_Int(7, 1, 2).
			Ensure()
	}); message != "" {
		t.Fatalf("non-member: panic = %q, want none", message)
	}
}

// Test_Noassert_Records_Nothing prevents the dual from paying the cost it exists to remove. A
// silent build that still walked the coverage map would keep the per-call map traffic that
// dominates the framework's runtime price, so recording nothing is the point, not a side effect.
func Test_Noassert_Records_Nothing(t *testing.T) {
	recorder := &invariant.Recorder{Is_Test: true}
	invariant.Recorder_Always(recorder, true, "reached")
	invariant.Recorder_Dot_Product(recorder, "noassert.records").
		Sometimes(true, "axis").
		Ensure()
	recorder.Events.Range(func(key any, value any) (keep_going bool) {
		t.Fatalf("recorded %v, want no coverage entries", key)
		return false
	})
}
