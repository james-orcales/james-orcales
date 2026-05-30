package time_test

import (
	"testing"

	timeos "github.com/james-orcales/james-orcales/golang_snacks/time/default"
)

// Test_Operating_System_Smoke verifies the host clock never reads monotonic
// backwards and reports a positive wall-clock time. Real time is non-deterministic,
// so this is a smoke test, not a snapshot.
func Test_Operating_System_Smoke(t *testing.T) {
	host := timeos.New_Operating_System_Clock()
	first := host.Now_Monotonic()
	second := host.Now_Monotonic()
	if second < first {
		t.Errorf("monotonic regressed: %d then %d", first, second)
	}
	host.Tick() // No-op on a real clock.
	if host.Now_Realtime() <= 0 {
		t.Error("realtime must be positive nanoseconds since the Unix epoch")
	}
}
