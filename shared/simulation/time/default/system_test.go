package time_test

import (
	"testing"

	invariant "local/james-orcales/shared/invariant/default"
	timeos "local/james-orcales/shared/simulation/time/default"
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
	host := timeos.New_Operating_System_Clock()
	first := host.Now_Monotonic()
	second := host.Now_Monotonic()
	testify.True(t, second >= first, first, second)
	testify.Positive(t, host.Now_Realtime())
}
