package time_test

import (
	"testing"

	"local/james-orcales/shared/simulation/time"
	"local/james-orcales/shared/testify"
)

// Test_Virtual_Clock_Monotonic check deterministic clock advance exactly one resolution per
// Tick. Also check Now_Realtime is epoch plus elapsed monotonic span, when no skew.
func Test_Virtual_Clock_Monotonic(t *testing.T) {
	c, tick := time.Virtual_Clock_To_Clock(time.Virtual_Clock{Resolution: 50, Epoch: 1000})

	testify.Zero(t, c.Now_Monotonic())
	testify.Equal(t, time.Moment(1000), c.Now_Realtime())
	tick()
	testify.Equal(t, time.Monotonic_Moment(50), c.Now_Monotonic())
	testify.Equal(t, time.Moment(1050), c.Now_Realtime())
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
	testify.Equal(t, time.Monotonic_Moment(2000), c.Now_Monotonic())
	testify.Equal(t, time.Moment(1998), c.Now_Realtime())

	// Periodic skew of amplitude 1000 over four-tick period peak at quarter turn. At tick 1,
	// sin(2pi/4) = 1, thus skew equal full amplitude and cancel elapsed span exactly.
	periodic, periodic_tick := time.Virtual_Clock_To_Clock(time.Virtual_Clock{
		Resolution: 1000,
		Epoch:      0,
		Skew:       time.Skew(time.SKEW_KIND_PERIODIC, 1000, 4),
	})
	periodic_tick()
	testify.Zero(t, periodic.Now_Realtime())
}

// Test_Timeline_Timeout check timeout fire exactly when virtual clock reach its deadline. No
// real wait. Fully deterministic.
func Test_Timeline_Timeout(t *testing.T) {
	loop, driver, clock := sim_loop(0)

	fired_at := time.Monotonic_Moment(-1)
	var completion time.Completion
	loop.Timeout(&completion, 5*time.NANOSECOND, func(completed *time.Completion) {
		testify.No_Error(t, completed.Error)
		fired_at = clock.Now_Monotonic()
	})

	driver.Run_For(10 * time.NANOSECOND)

	testify.Equal(t, time.Monotonic_Moment(5), fired_at)
}

// Test_Timeline_Event check Event primitive retire its listener before callback run. Same
// completion can then arm again for later trigger.
func Test_Timeline_Event(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	event, open_err := loop.Open_Event()
	if !testify.No_Error(t, open_err) {
		return
	}
	fired := 0
	var completion time.Completion
	callback := func(_ *time.Completion) { fired++ }
	loop.Event_Listen(event, &completion, callback)
	loop.Event_Trigger(event, &completion)
	driver.Run()
	testify.Equal(t, 1, fired)
	loop.Event_Listen(event, &completion, callback)
	loop.Event_Trigger(event, &completion)
	driver.Run()
	testify.Equal(t, 2, fired)
	loop.Close_Event(event)
}

// Test_Timeline_Run_Until check driver pump loop until predicate report true, and report
// completed. Also check predicate that never trip return false at deadline. That is point of
// timeout: sim never spin without end.
func Test_Timeline_Run_Until(t *testing.T) {
	loop, driver, _ := sim_loop(1)

	done := false
	var completion time.Completion
	loop.Timeout(&completion, 3*time.NANOSECOND, func(_ *time.Completion) {
		done = true
	})

	completed, drive_err := driver.Run_Until(
		SIM_DEADLINE, func() (finished bool) { return done },
	)
	testify.No_Error(t, drive_err)
	testify.True(t, completed)
	testify.True(t, done)

	completed, drive_err = driver.Run_Until(
		SIM_DEADLINE, func() (finished bool) { return false },
	)
	testify.No_Error(t, drive_err)
	testify.False(t, completed)
}

// Test_Timeline_Reuse check submit of completion still in flight panic. Reused completion thus
// fail loud, never corrupt queue.
func Test_Timeline_Reuse(t *testing.T) {
	loop, _, _ := sim_loop(0)
	var completion time.Completion
	loop.Timeout(&completion, 5*time.NANOSECOND, func(_ *time.Completion) {})
	testify.Panics(t, func() {
		loop.Timeout(&completion, 5*time.NANOSECOND, func(_ *time.Completion) {})
	})
}

// Test_Timeline_Copy check submit of by-value copy panic. Copied completion thus fail loud,
// never split view of loop from view of caller. Fire original first, thus copy is unarmed.
// That isolate copy guard from reuse guard.
func Test_Timeline_Copy(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	var completion time.Completion
	loop.Timeout(&completion, 5*time.NANOSECOND, func(_ *time.Completion) {})
	driver.Run_For(10 * time.NANOSECOND)
	duplicate := completion
	testify.Panics(t, func() {
		loop.Timeout(&duplicate, 5*time.NANOSECOND, func(_ *time.Completion) {})
	})
}

// Test_Timeline_Reentrancy check drive of loop from inside completion callback panic.
// Re-entrant Run* thus fail loud, never corrupt queue mid-drain.
func Test_Timeline_Reentrancy(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	var completion time.Completion
	loop.Timeout(&completion, 5*time.NANOSECOND, func(_ *time.Completion) {
		driver.Run()
	})
	testify.Panics(t, func() {
		driver.Run_For(10 * time.NANOSECOND)
	})
}

// Test_Callback_Result keeps operation meaning outside the shared completion: Data is opaque to
// the timeline, while each operation interprets it as its descriptor, count, or status.
func Test_Callback_Result(t *testing.T) {
	called := false
	completion := time.Completion{Data: 42, Error: time.Deadline_Exceeded}
	callback := time.Callback(func(got *time.Completion) {
		testify.True(t, got == &completion)
		testify.Equal(t, 42, got.Data)
		testify.Error_Is(t, got.Error, time.Deadline_Exceeded)
		called = true
	})
	callback(&completion)
	testify.True(t, called)
}

// Test_Timeline_Unbounded check negative Run_Until timeout panic. Unbounded pump have no cap,
// thus stalled operation spin loop as long as process live. Timeline refuse request, never
// hang run.
func Test_Timeline_Unbounded(t *testing.T) {
	_, driver, _ := sim_loop(0)
	testify.Panics(t, func() {
		driver.Run_Until(-1*time.NANOSECOND, func() (finished bool) { return false })
	})
}

// Test_Monotonic_Moment check uptime domain: zero at boot, one year of nanoseconds at maximum.
// Maximum computed here from own factors, thus change to unit ladder of package cannot move
// bound without this test.
func Test_Monotonic_Moment(t *testing.T) {
	t.Parallel()
	const ONE_YEAR_NANOSECONDS int64 = 365 * 24 * 60 * 60 * 1000 * 1000 * 1000
	testify.Zero(t, time.MONOTONIC_MOMENT_MINIMUM)
	testify.Equal(t, ONE_YEAR_NANOSECONDS, int64(time.MONOTONIC_MOMENT_MAXIMUM))
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
