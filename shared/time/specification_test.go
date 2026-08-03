package time_test

import (
	"testing"

	"local/james-orcales/shared/time"
)

// Test_Virtual_Clock_Monotonic verifies the deterministic clock advances exactly one
// resolution per Tick, and that Now_Realtime is the epoch plus the elapsed monotonic
// span when there is no skew.
func Test_Virtual_Clock_Monotonic(t *testing.T) {
	c, tick := time.Virtual_Clock_To_Clock(time.Virtual_Clock{Resolution: 50, Epoch: 1000})

	if got := c.Now_Monotonic(); got != 0 {
		t.Fatalf("monotonic at tick 0 = %d, want 0", got)
	}
	if got := c.Now_Realtime(); got != 1000 {
		t.Fatalf("realtime at tick 0 = %d, want 1000", got)
	}
	tick()
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
	c, tick := time.Virtual_Clock_To_Clock(time.Virtual_Clock{
		Resolution: 1000,
		Epoch:      0,
		Skew:       time.Skew(time.Skew_Input{Kind: time.SKEW_KIND_LINEAR, A: 1, B: 0}),
	})
	tick()
	tick()
	// Monotonic = 2*1000 = 2000; skew = 2*1 = 2; realtime = 0 + 2000 - 2 = 1998.
	if got := c.Now_Monotonic(); got != 2000 {
		t.Fatalf("monotonic = %d, want 2000", got)
	}
	if got := c.Now_Realtime(); got != 1998 {
		t.Fatalf("realtime = %d, want 1998 (skewed)", got)
	}

	// A periodic skew of amplitude 1000 over a four-tick period peaks at a quarter
	// turn: at tick 1, sin(2pi/4) = 1, so the skew equals the full amplitude and
	// cancels the elapsed span exactly.
	periodic, periodic_tick := time.Virtual_Clock_To_Clock(time.Virtual_Clock{
		Resolution: 1000,
		Epoch:      0,
		Skew: time.Skew(time.Skew_Input{
			Kind: time.SKEW_KIND_PERIODIC, A: 1000, B: 4,
		}),
	})
	periodic_tick()
	if got := periodic.Now_Realtime(); got != 0 {
		t.Fatalf("periodic realtime at quarter turn = %d, want 0", got)
	}
}

// Test_Invariant_Domains verifies the special values of each admitted scalar domain.
func Test_Invariant_Domains(t *testing.T) {
	t.Parallel()
	verify_virtual_clock_domains()
	verify_skew_domains()
}

// SPECIAL_VALUE_COUNT is how many special values a full-width signed domain has: its
// two bounds and the four interior sentinels the framework expands.
const SPECIAL_VALUE_COUNT = 6

// The signed special values every full-width domain in this package states.
func special_values() (values [SPECIAL_VALUE_COUNT]int64) {
	return [...]int64{
		time.INTEGER_64_MINIMUM,
		time.INTEGER_64_MAXIMUM,
		0,
		1,
		2,
		-1,
	}
}

// Exercises each special value at the two scalars a virtual clock is configured with.
// The clock is only built, never ticked: an extreme resolution or epoch is a legal
// configuration, and the product it would read is not what these domains state.
func verify_virtual_clock_domains() {
	for _, value := range special_values() {
		time.Virtual_Clock_To_Clock(time.Virtual_Clock{Resolution: time.Duration(value)})
		time.Virtual_Clock_To_Clock(time.Virtual_Clock{Epoch: time.Moment(value)})
	}
}

// Exercises each skew model and each special value at both of its coefficients.
func verify_skew_domains() {
	kinds := [...]time.Skew_Kind{
		time.SKEW_KIND_LINEAR,
		time.SKEW_KIND_PERIODIC,
		time.SKEW_KIND_STEP,
	}
	for _, kind := range kinds {
		time.Skew(time.Skew_Input{Kind: kind})
	}
	for _, value := range special_values() {
		time.Skew(time.Skew_Input{A: time.Duration(value)})
		time.Skew(time.Skew_Input{B: time.Tick_Count(value)})
	}
}
