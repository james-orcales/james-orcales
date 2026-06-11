package time_test

import (
	"testing"

	"github.com/james-orcales/james-orcales/shared/time"
)

// Test_Virtual_Clock_Monotonic verifies the deterministic clock advances exactly one
// resolution per Tick, and that Now_Realtime is the epoch plus the elapsed monotonic
// span when there is no skew.
func Test_Virtual_Clock_Monotonic(t *testing.T) {
	c := time.Virtual_Clock_To_Clock(time.Virtual_Clock{Resolution: 50, Epoch: 1000})

	if got := c.Now_Monotonic(); got != 0 {
		t.Fatalf("monotonic at tick 0 = %d, want 0", got)
	}
	if got := c.Now_Realtime(); got != 1000 {
		t.Fatalf("realtime at tick 0 = %d, want 1000", got)
	}
	c.Tick()
	if got := c.Now_Monotonic(); got != 50 {
		t.Fatalf("monotonic after 1 tick = %d, want 50", got)
	}
	if got := c.Now_Realtime(); got != 1050 {
		t.Fatalf("realtime after 1 tick = %d, want 1050", got)
	}
}

// Test_Virtual_Clock_Skew verifies a modeled skew bends Now_Realtime away from true
// elapsed time while leaving Now_Monotonic untouched: a clock losing one nanosecond
// of realtime per tick.
func Test_Virtual_Clock_Skew(t *testing.T) {
	c := time.Virtual_Clock_To_Clock(time.Virtual_Clock{
		Resolution: 1000,
		Epoch:      0,
		Skew:       time.Skew(time.Skew_Input{Kind: time.Skew_Kind_Linear, A: 1, B: 0}),
	})
	c.Tick()
	c.Tick()
	// Monotonic = 2*1000 = 2000; skew = 2*1 = 2; realtime = 0 + 2000 - 2 = 1998.
	if got := c.Now_Monotonic(); got != 2000 {
		t.Fatalf("monotonic = %d, want 2000", got)
	}
	if got := c.Now_Realtime(); got != 1998 {
		t.Fatalf("realtime = %d, want 1998 (skewed)", got)
	}
}

// Test_Virtual_Clock_Sleep verifies sleeping advances the virtual clock by the
// slept duration rather than waiting, so Now_Monotonic moves forward by that span.
func Test_Virtual_Clock_Sleep(t *testing.T) {
	c := time.Virtual_Clock_To_Clock(time.Virtual_Clock{Resolution: 50})
	c.Sleep(150)
	if got := c.Now_Monotonic(); got != 150 {
		t.Fatalf("monotonic after Sleep(150) = %d, want 150", got)
	}
}
