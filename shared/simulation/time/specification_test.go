package time_test

import (
	"testing"

	"local/james-orcales/shared/simulation/time"
)

// Test_Virtual_Clock_Monotonic verifies the deterministic clock advances exactly one
// resolution per Tick, and that Now_Realtime is the epoch plus the elapsed monotonic
// span when there is no skew.
func Test_Virtual_Clock_Monotonic(t *testing.T) {
	c, tick := time.Virtual_Clock_To_Any_Clock(time.Virtual_Clock{Resolution: 50, Epoch: 1000})

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
	c, tick := time.Virtual_Clock_To_Any_Clock(time.Virtual_Clock{
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
	periodic, periodic_tick := time.Virtual_Clock_To_Any_Clock(time.Virtual_Clock{
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

// Test_Timeline_Timeout verifies a timeout fires exactly when the virtual clock reaches
// its deadline — no real waiting, fully deterministic.
func Test_Timeline_Timeout(t *testing.T) {
	loop, driver, clock := sim_loop(0)

	fired_at := time.Moment(-1)
	var completion time.Completion
	loop.Timeout(&completion, func(_ *time.Completion, err error) {
		if err != nil {
			t.Fatalf("timeout error: %v", err)
		}
		fired_at = clock.Now_Monotonic()
	}, 5*time.NANOSECOND)

	driver.Run_For(10 * time.NANOSECOND)

	if fired_at != 5 {
		t.Fatalf("timeout fired at %d, want 5", fired_at)
	}
}

// Test_Timeline_Next_Tick verifies next-tick callbacks use the completed queue and reset
// removes every queued callback for a source without firing it.
func Test_Timeline_Next_Tick(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	fired := 0
	var first time.Completion
	var second time.Completion
	loop.Next_Tick(&first, func(_ *time.Completion) { fired++ }, time.NEXT_TICK_VSR)
	loop.Next_Tick(&second, func(_ *time.Completion) { fired++ }, time.NEXT_TICK_VSR)
	loop.Reset_Next_Tick(time.NEXT_TICK_VSR)
	driver.Run()
	if fired != 0 {
		t.Fatalf("reset next tick fired %d callbacks, want 0", fired)
	}
	loop.Next_Tick(&first, func(_ *time.Completion) { fired++ }, time.NEXT_TICK_VSR)
	driver.Run()
	if fired != 1 {
		t.Fatalf("next tick fired %d callbacks, want 1", fired)
	}
}

// Test_Timeline_Event verifies the TigerBeetle Event primitive retires its listener before
// invoking the callback, allowing the same completion to be re-armed for a later trigger.
func Test_Timeline_Event(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	event, open_err := loop.Open_Event()
	if open_err != nil {
		t.Fatalf("open event: %v", open_err)
	}
	fired := 0
	var completion time.Completion
	callback := func(_ *time.Completion) { fired++ }
	loop.Event_Listen(event, &completion, callback)
	loop.Event_Trigger(event, &completion)
	driver.Run()
	if fired != 1 {
		t.Fatalf("event fired %d times, want 1", fired)
	}
	loop.Event_Listen(event, &completion, callback)
	loop.Event_Trigger(event, &completion)
	driver.Run()
	if fired != 2 {
		t.Fatalf("event fired %d times, want 2", fired)
	}
	loop.Close_Event(event)
}

// Test_Timeline_Run_Until verifies the driver pumps the loop until the predicate reports true,
// reporting completed, and — the point of the timeout — that a never-satisfied predicate
// returns false at the deadline instead of spinning the sim forever.
func Test_Timeline_Run_Until(t *testing.T) {
	loop, driver, _ := sim_loop(1)

	done := false
	var completion time.Completion
	loop.Timeout(&completion, func(_ *time.Completion, _ error) {
		done = true
	}, 3*time.NANOSECOND)

	completed, drive_err := driver.Run_Until(
		func() (finished bool) { return done }, SIM_DEADLINE,
	)
	if drive_err != nil {
		t.Fatalf("Run_Until error: %v", drive_err)
	}
	if !completed {
		t.Fatal("Run_Until reported the read did not complete")
	}
	if !done {
		t.Fatal("Run_Until returned before the read completed")
	}

	completed, drive_err = driver.Run_Until(
		func() (finished bool) { return false }, SIM_DEADLINE,
	)
	if drive_err != nil {
		t.Fatalf("Run_Until deadline error: %v", drive_err)
	}
	if completed {
		t.Fatal("Run_Until reported completion for a predicate that never trips")
	}
}

// Test_Timeline_Reuse verifies submitting a completion that is still in flight panics, so a
// reused completion fails loudly instead of corrupting the queue.
func Test_Timeline_Reuse(t *testing.T) {
	loop, _, _ := sim_loop(0)
	var completion time.Completion
	loop.Timeout(&completion, func(_ *time.Completion, err error) {}, 5*time.NANOSECOND)
	defer func() {
		if recover() == nil {
			t.Fatal("reusing an in-flight completion must panic")
		}
	}()
	loop.Timeout(&completion, func(_ *time.Completion, err error) {}, 5*time.NANOSECOND)
}

// Test_Timeline_Copy verifies submitting a by-value copy of a completion panics, so a copied
// completion fails loudly instead of splitting the loop's view from the caller's. It fires
// the original first so the copy is unarmed — isolating the copy guard from the reuse one.
func Test_Timeline_Copy(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	var completion time.Completion
	loop.Timeout(&completion, func(_ *time.Completion, err error) {}, 5*time.NANOSECOND)
	driver.Run_For(10 * time.NANOSECOND)
	duplicate := completion
	defer func() {
		if recover() == nil {
			t.Fatal("submitting a copied completion must panic")
		}
	}()
	loop.Timeout(&duplicate, func(_ *time.Completion, err error) {}, 5*time.NANOSECOND)
}

// Test_Timeline_Reentrancy verifies driving the loop from within a completion callback panics,
// so a re-entrant Run* fails loudly instead of corrupting the queue mid-drain.
func Test_Timeline_Reentrancy(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	var completion time.Completion
	loop.Timeout(&completion, func(_ *time.Completion, err error) {
		driver.Run()
	}, 5*time.NANOSECOND)
	defer func() {
		if recover() == nil {
			t.Fatal("driving from within a callback must panic")
		}
	}()
	driver.Run_For(10 * time.NANOSECOND)
}

// Test_Timeline_Introspect verifies the census reports the queue by class, so a stall shows
// as the class that will not drain rather than as one opaque depth.
func Test_Timeline_Introspect(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	var timer, deferred time.Completion
	loop.Timeout(&timer, func(_ *time.Completion, _ error) {}, time.MICROSECOND)
	loop.Next_Tick(&deferred, func(_ *time.Completion) {}, time.NEXT_TICK_LSM)
	counts := driver.Introspect()
	if counts.Timeouts != 1 {
		t.Fatalf("timer count = %d, want 1", counts.Timeouts)
	}
	if counts.Raw_Open != 0 {
		t.Fatalf(
			"descriptor count = %d, want 0 with no reporter", counts.Raw_Open)
	}
}

// The Run_Until cap for the loop tests: generous, since a completion returns the pump the
// instant it fires — this bound only bites a genuine stall.
const SIM_DEADLINE = 4096 * time.NANOSECOND

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
		time.Virtual_Clock_To_Any_Clock(
			time.Virtual_Clock{Resolution: time.Duration(value)})
		time.Virtual_Clock_To_Any_Clock(time.Virtual_Clock{Epoch: time.Moment(value)})
		time.New_Virtual_Timeline(time.Virtual_Clock{Resolution: time.Duration(value)})
		time.New_Virtual_Timeline(time.Virtual_Clock{Epoch: time.Moment(value)})
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

// Builds a deterministic loop and its driver for one seed of test.
func sim_loop(_ uint64) (loop time.Timeline, driver time.Driver, clock time.Any_Clock) {
	return time.New_Virtual_Timeline(time.Virtual_Clock{Resolution: time.NANOSECOND})
}

// Test_Invariant_Domains verifies the special values of each admitted scalar domain.
func Test_Invariant_Domains(t *testing.T) {
	t.Parallel()
	verify_virtual_clock_domains()
	verify_skew_domains()
}
