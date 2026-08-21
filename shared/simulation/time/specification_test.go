package time_test

import (
	"testing"

	"local/james-orcales/shared/simulation/time"
)

// Test_Virtual_Clock_Monotonic check deterministic clock advance exactly one resolution per
// Tick. Also check Now_Realtime is epoch plus elapsed monotonic span, when no skew.
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

// Test_Virtual_Clock_Skew check modeled skew bend Now_Realtime away from true elapsed time,
// and leave Now_Monotonic untouched. Clock here lose one nanosecond of realtime per tick.
func Test_Virtual_Clock_Skew(t *testing.T) {
	c, tick := time.Virtual_Clock_To_Clock(time.Virtual_Clock{
		Resolution: 1000,
		Epoch:      0,
		Skew:       time.Skew(time.SKEW_KIND_LINEAR, 1, 0),
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

	// Periodic skew of amplitude 1000 over four-tick period peak at quarter turn. At tick 1,
	// sin(2pi/4) = 1, thus skew equal full amplitude and cancel elapsed span exactly.
	periodic, periodic_tick := time.Virtual_Clock_To_Clock(time.Virtual_Clock{
		Resolution: 1000,
		Epoch:      0,
		Skew:       time.Skew(time.SKEW_KIND_PERIODIC, 1000, 4),
	})
	periodic_tick()
	if got := periodic.Now_Realtime(); got != 0 {
		t.Fatalf("periodic realtime at quarter turn = %d, want 0", got)
	}
}

// Test_Timeline_Timeout check timeout fire exactly when virtual clock reach its deadline. No
// real wait. Fully deterministic.
func Test_Timeline_Timeout(t *testing.T) {
	loop, driver, clock := sim_loop(0)

	fired_at := time.Monotonic_Moment(-1)
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

// Test_Timeline_Event check Event primitive retire its listener before callback run. Same
// completion can then arm again for later trigger.
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

// Test_Timeline_Run_Until check driver pump loop until predicate report true, and report
// completed. Also check predicate that never trip return false at deadline. That is point of
// timeout: sim never spin without end.
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

// Test_Timeline_Reuse check submit of completion still in flight panic. Reused completion thus
// fail loud, never corrupt queue.
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

// Test_Timeline_Copy check submit of by-value copy panic. Copied completion thus fail loud,
// never split view of loop from view of caller. Fire original first, thus copy is unarmed.
// That isolate copy guard from reuse guard.
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

// Test_Timeline_Reentrancy check drive of loop from inside completion callback panic.
// Re-entrant Run* thus fail loud, never corrupt queue mid-drain.
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

// Test_Timeline_Unbounded check negative Run_Until timeout panic. Unbounded pump have no cap,
// thus stalled operation spin loop as long as process live. Timeline refuse request, never
// hang run.
func Test_Timeline_Unbounded(t *testing.T) {
	_, driver, _ := sim_loop(0)
	defer func() {
		if recover() == nil {
			t.Fatal("a negative Run_Until timeout must panic")
		}
	}()
	driver.Run_Until(func() (finished bool) { return false }, -1*time.NANOSECOND)
}

// Test_Monotonic_Moment check uptime domain: zero at boot, one year of nanoseconds at maximum.
// Maximum computed here from own factors, thus change to unit ladder of package cannot move
// bound without this test.
func Test_Monotonic_Moment(t *testing.T) {
	t.Parallel()
	const ONE_YEAR_NANOSECONDS int64 = 365 * 24 * 60 * 60 * 1000 * 1000 * 1000
	if time.MONOTONIC_MOMENT_MINIMUM != 0 {
		t.Fatalf("uptime minimum = %d, want 0", time.MONOTONIC_MOMENT_MINIMUM)
	}
	if int64(time.MONOTONIC_MOMENT_MAXIMUM) != ONE_YEAR_NANOSECONDS {
		t.Fatalf(
			"uptime maximum = %d, want %d",
			int64(time.MONOTONIC_MOMENT_MAXIMUM), ONE_YEAR_NANOSECONDS)
	}
}

// Run_Until cap for loop tests. Generous: completion give pump back instant it fire. This
// bound bite real stall only.
const SIM_DEADLINE = 4096 * time.NANOSECOND

// SPECIAL_VALUE_COUNT: how many special values full-width signed domain have. Two bounds, plus
// four interior sentinels framework expand.
const SPECIAL_VALUE_COUNT = 6

// Signed special values every full-width domain in this package state.
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

// Hit each special value at two scalars virtual clock take. Clock only built, never ticked.
// Extreme resolution or epoch is legal configuration. Product it would read is not what these
// domains state.
func verify_virtual_clock_domains() {
	for _, value := range special_values() {
		time.Virtual_Clock_To_Clock(
			time.Virtual_Clock{Resolution: time.Duration(value)})
		time.Virtual_Clock_To_Clock(time.Virtual_Clock{Epoch: time.Moment(value)})
		time.New_Virtual_Timeline(time.Virtual_Clock{Resolution: time.Duration(value)})
		time.New_Virtual_Timeline(time.Virtual_Clock{Epoch: time.Moment(value)})
	}
}

// Hit each skew model, and each special value at both coefficients.
func verify_skew_domains() {
	kinds := [...]time.Skew_Kind{
		time.SKEW_KIND_LINEAR,
		time.SKEW_KIND_PERIODIC,
		time.SKEW_KIND_STEP,
	}
	for _, kind := range kinds {
		time.Skew(kind, 0, 0)
	}
	for _, value := range special_values() {
		time.Skew(time.SKEW_KIND_LINEAR, time.Duration(value), 0)
		time.Skew(time.SKEW_KIND_LINEAR, 0, time.Tick_Count(value))
	}
}

// Hit uptime domain at each special value framework expand, through two clocks package build.
// Resolution of one year reach upper bound on first tick, thus sweep cost one tick, not year
// of them.
func verify_uptime_domains() {
	grain, grain_tick := time.Virtual_Clock_To_Clock(
		time.Virtual_Clock{Resolution: time.NANOSECOND})
	grain.Now_Monotonic()
	grain_tick()
	grain.Now_Monotonic()
	grain_tick()
	grain.Now_Monotonic()

	bound := time.Virtual_Clock{Resolution: time.Duration(time.MONOTONIC_MOMENT_MAXIMUM)}
	one_year, one_year_tick := time.Virtual_Clock_To_Clock(bound)
	one_year_tick()
	one_year.Now_Monotonic()

	_, driver, timeline := time.New_Virtual_Timeline(bound)
	driver.Run()
	timeline.Now_Monotonic()
}

// Build deterministic loop plus its driver, for one seed of test.
func sim_loop(_ uint64) (loop time.Timeline, driver time.Driver, clock time.Clock) {
	return time.New_Virtual_Timeline(time.Virtual_Clock{Resolution: time.NANOSECOND})
}

// Test_Invariant_Domains check special values of each admitted scalar domain.
func Test_Invariant_Domains(t *testing.T) {
	t.Parallel()
	verify_virtual_clock_domains()
	verify_skew_domains()
	verify_uptime_domains()
}
