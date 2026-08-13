package time_test

import (
	"testing"

	invariant "local/james-orcales/shared/invariant/default"
	timeos "local/james-orcales/shared/simulation/time/default"
)

// TestMain registers the package invariant roots before the smoke test runs.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}

// Test_Operating_System_Smoke verifies the host clock never reads monotonic
// backwards and reports a positive wall-clock time. Real time is non-deterministic,
// so this is a smoke test, not a snapshot.
func Test_Operating_System_Smoke(t *testing.T) {
	host, tick := timeos.New_Operating_System_Any_Clock()
	first := host.Now_Monotonic()
	second := host.Now_Monotonic()
	if second < first {
		t.Errorf("monotonic regressed: %d then %d", first, second)
	}
	tick() // No-op on a real clock.
	if host.Now_Realtime() <= 0 {
		t.Error("realtime must be positive nanoseconds since the Unix epoch")
	}
}
