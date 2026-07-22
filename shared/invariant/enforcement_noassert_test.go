//go:build noassert

// The noassert build is a production mode, not a test mode: the rest of this suite asserts that
// enforcement fires, so it is meaningful only in the default build. These tests are the dual's
// own contract — pure silence: nothing panics and nothing records, whatever the inputs — and run
// under `go test -tags noassert -run Test_Noassert`.

package invariant_test

import (
	"testing"

	"local/james-orcales/shared/invariant"
)

func Test_Noassert_Always_Is_Silent(t *testing.T) {
	recorder := &invariant.Recorder{Is_Test: true}
	invariant.Recorder_Always(recorder, false, "violated")
}

func Test_Noassert_Range_Is_Silent(t *testing.T) {
	recorder := &invariant.Recorder{Is_Test: true}
	invariant.Recorder_Range(recorder, "id", 5, 0, 3)
	invariant.Recorder_Range(recorder, "id", 5, 0, 10, 5)
	invariant.Recorder_Range(recorder, "id", 1, 0, 10, 20)
}

func Test_Noassert_Enum_Is_Silent(t *testing.T) {
	recorder := &invariant.Recorder{Is_Test: true}
	invariant.Recorder_Enum(recorder, "id", 7, 1, 2)
	invariant.Recorder_Enum(recorder, "id", 1, 1)
}

func Test_Noassert_Records_Nothing(t *testing.T) {
	recorder := &invariant.Recorder{Is_Test: true}
	recorder.Events.Store("m", &invariant.Assertion_Metadata{})
	invariant.Recorder_Always(recorder, true, "m")
	invariant.Recorder_Sometimes(recorder, "id", true, "m")
	value, _ := recorder.Events.Load("m")
	metadata := value.(*invariant.Assertion_Metadata)
	if metadata.Frequency.Load() != 0 {
		t.Fatal("the noassert build must not record")
	}
}
