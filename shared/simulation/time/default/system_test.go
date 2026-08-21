package time_test

import (
	"testing"

	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/simulation/time/default"
	"local/james-orcales/shared/testify"
)

// TestMain register package invariant roots before smoke test run.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}

// Test_Operating_System_Smoke check host clock never read monotonic backward, and report
// positive wall-clock time. Real time is not deterministic, thus this is smoke test, not
// snapshot.
func Test_Operating_System_Smoke(t *testing.T) {
	host := time.New_Operating_System_Clock()
	first := time.Clock_Now_Monotonic(host)
	second := time.Clock_Now_Monotonic(host)
	testify.True(t, second >= first, first, second)
	testify.Positive(t, time.Clock_Now_Realtime(host))
}

// Host-backed readers need the same heap proof as deterministic readers.
func Test_Operating_System_API_Heap_Allocation(t *testing.T) {
	host := time.New_Operating_System_Clock()
	t.Run("New_Operating_System_Clock", func(t *testing.T) {
		testify.Zero_Allocation(t, func() { host = time.New_Operating_System_Clock() })
	})
	t.Run("Now_Monotonic", func(t *testing.T) {
		var moment int64
		testify.Zero_Allocation(t, func() {
			moment = int64(time.Clock_Now_Monotonic(host))
		})
		testify.Positive(t, moment)
	})
	t.Run("Now_Realtime", func(t *testing.T) {
		var moment int64
		testify.Zero_Allocation(t, func() {
			moment = int64(time.Clock_Now_Realtime(host))
		})
		testify.Positive(t, moment)
	})
}
