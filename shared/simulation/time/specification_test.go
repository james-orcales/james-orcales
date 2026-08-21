package time_test

import (
	"testing"

	"local/james-orcales/shared/simulation/time"
	"local/james-orcales/shared/testify"
)

// Test_Virtual_Clock_Monotonic check deterministic clock advance exactly one resolution per
// Tick. Also check Now_Realtime is epoch plus elapsed monotonic span, when no skew.
func Test_Virtual_Clock_Monotonic(t *testing.T) {
	virtual := time.Virtual_Clock{Resolution: 50, Epoch: 1000}
	c := time.Virtual_Clock_To_Clock(&virtual)

	testify.Zero(t, time.Clock_Now_Monotonic(c))
	testify.Equal(t, time.Moment(1000), time.Clock_Now_Realtime(c))
	time.Virtual_Clock_Tick(&virtual)
	testify.Equal(t, time.Monotonic_Moment(50), time.Clock_Now_Monotonic(c))
	testify.Equal(t, time.Moment(1050), time.Clock_Now_Realtime(c))
}

// Test_Virtual_Clock_Skew check modeled skew bend Now_Realtime away from true elapsed time,
// and leave Now_Monotonic untouched. Clock here lose one nanosecond of realtime per tick.
func Test_Virtual_Clock_Skew(t *testing.T) {
	virtual := time.Virtual_Clock{
		Resolution: 1000,
		Epoch:      0,
		Skew:       time.Skew(time.SKEW_KIND_LINEAR, 1, 0),
	}
	c := time.Virtual_Clock_To_Clock(&virtual)
	time.Virtual_Clock_Tick(&virtual)
	time.Virtual_Clock_Tick(&virtual)
	// Monotonic = 2*1000 = 2000; skew = 2*1 = 2; realtime = 0 + 2000 - 2 = 1998.
	testify.Equal(t, time.Monotonic_Moment(2000), time.Clock_Now_Monotonic(c))
	testify.Equal(t, time.Moment(1998), time.Clock_Now_Realtime(c))

	// Periodic skew of amplitude 1000 over four-tick period peak at quarter turn. At tick 1,
	// sin(2pi/4) = 1, thus skew equal full amplitude and cancel elapsed span exactly.
	periodic_virtual := time.Virtual_Clock{
		Resolution: 1000,
		Epoch:      0,
		Skew:       time.Skew(time.SKEW_KIND_PERIODIC, 1000, 4),
	}
	periodic := time.Virtual_Clock_To_Clock(&periodic_virtual)
	time.Virtual_Clock_Tick(&periodic_virtual)
	testify.Zero(t, time.Clock_Now_Realtime(periodic))
}

// Test_Timeline_Timeout check timeout fire exactly when virtual clock reach its deadline. No
// real wait. Fully deterministic.
func Test_Timeline_Timeout(t *testing.T) {
	loop, driver, clock := sim_loop(0)

	fired_at := time.Monotonic_Moment(-1)
	var completion time.Completion
	time.Timeline_Timeout(
		loop, &completion, 5*time.NANOSECOND, func(completed *time.Completion) {
			testify.No_Error(t, completed.Error)
			fired_at = time.Clock_Now_Monotonic(clock)
		},
	)

	time.Driver_Run_For(driver, 10*time.NANOSECOND)

	testify.Equal(t, time.Monotonic_Moment(5), fired_at)
}

// Long-lived completion must not keep callback closure, or closure keep operation buffer live.
// Clear before call so callback can arm same completion again without new callback being erased.
func Test_Timeline_Callback_Released(t *testing.T) {
	loop, driver, _ := sim_loop(0)

	var once time.Completion
	time.Timeline_Timeout(loop, &once, time.NANOSECOND, func(_ *time.Completion) {})
	time.Driver_Run_For(driver, 2*time.NANOSECOND)
	testify.Nil(t, once.Callback)

	var repeated time.Completion
	fired := 0
	var arm func()
	arm = func() {
		time.Timeline_Timeout(loop, &repeated, time.NANOSECOND, func(_ *time.Completion) {
			fired++
			if fired < 3 {
				arm()
			}
		})
	}
	arm()
	time.Driver_Run_For(driver, 4*time.NANOSECOND)
	testify.Equal(t, 3, fired)
	testify.Nil(t, repeated.Callback)
}

// Test_Timeline_Event check Event primitive retire its listener before callback run. Same
// completion can then arm again for later trigger.
func Test_Timeline_Event(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	event, open_err := time.Timeline_Open_Event(loop)
	if !testify.No_Error(t, open_err) {
		return
	}
	fired := 0
	var completion time.Completion
	callback := func(_ *time.Completion) { fired++ }
	time.Timeline_Event_Listen(loop, event, &completion, callback)
	time.Timeline_Event_Trigger(loop, event, &completion)
	time.Driver_Run(driver)
	testify.Equal(t, 1, fired)
	time.Timeline_Event_Listen(loop, event, &completion, callback)
	time.Timeline_Event_Trigger(loop, event, &completion)
	time.Driver_Run(driver)
	testify.Equal(t, 2, fired)
	time.Timeline_Close_Event(loop, event)
}

// Test_Timeline_Capacity checks a full caller store rejects new work before it changes
// completion or event lifecycle state.
func Test_Timeline_Capacity(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	var first time.Completion
	time.Timeline_Submit(loop, &first, 2*time.NANOSECOND, allocation_callback)
	var overflow time.Completion
	testify.Panics(t, func() {
		time.Timeline_Submit(loop, &overflow, 2*time.NANOSECOND, allocation_callback)
	})
	testify.False(t, overflow.Armed)

	event, open_err := time.Timeline_Open_Event(loop)
	if !testify.No_Error(t, open_err) {
		return
	}
	_, capacity_err := time.Timeline_Open_Event(loop)
	testify.Error_Is(t, capacity_err, time.Virtual_Event_Capacity_Exceeded)
	fired := false
	var listener time.Completion
	time.Timeline_Event_Listen(loop, event, &listener, func(_ *time.Completion) {
		fired = true
	})
	testify.Panics(t, func() {
		time.Timeline_Event_Trigger(loop, event, &listener)
	})
	time.Driver_Run_For(driver, 3*time.NANOSECOND)
	time.Timeline_Event_Trigger(loop, event, &listener)
	time.Driver_Run(driver)
	if !testify.True(t, fired) {
		return
	}
	time.Timeline_Close_Event(loop, event)
	verify_pending_trigger_capacity(t)
}

// Test_Timeline_Run_Until check driver pump loop until predicate report true, and report
// completed. Also check predicate that never trip return false at deadline. That is point of
// timeout: sim never spin without end.
func Test_Timeline_Run_Until(t *testing.T) {
	loop, driver, _ := sim_loop(1)

	done := false
	var completion time.Completion
	time.Timeline_Timeout(loop, &completion, 3*time.NANOSECOND, func(_ *time.Completion) {
		done = true
	})

	completed, drive_err := time.Driver_Run_Until(driver,
		SIM_DEADLINE, func() (finished bool) { return done },
	)
	testify.No_Error(t, drive_err)
	testify.True(t, completed)
	testify.True(t, done)

	completed, drive_err = time.Driver_Run_Until(driver,
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
	time.Timeline_Timeout(loop, &completion, 5*time.NANOSECOND, func(_ *time.Completion) {})
	testify.Panics(t, func() {
		time.Timeline_Timeout(loop, &completion, 5*time.NANOSECOND,
			func(_ *time.Completion) {})
	})
}

// Test_Timeline_Copy check submit of by-value copy panic. Copied completion thus fail loud,
// never split view of loop from view of caller. Fire original first, thus copy is unarmed.
// That isolate copy guard from reuse guard.
func Test_Timeline_Copy(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	var completion time.Completion
	time.Timeline_Timeout(loop, &completion, 5*time.NANOSECOND, func(_ *time.Completion) {})
	time.Driver_Run_For(driver, 10*time.NANOSECOND)
	duplicate := completion
	testify.Panics(t, func() {
		time.Timeline_Timeout(loop, &duplicate, 5*time.NANOSECOND,
			func(_ *time.Completion) {})
	})
}

// Test_Timeline_Reentrancy check drive of loop from inside completion callback panic.
// Re-entrant Run* thus fail loud, never corrupt queue mid-drain.
func Test_Timeline_Reentrancy(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	var completion time.Completion
	time.Timeline_Timeout(loop, &completion, 5*time.NANOSECOND, func(_ *time.Completion) {
		time.Driver_Run(driver)
	})
	testify.Panics(t, func() {
		time.Driver_Run_For(driver, 10*time.NANOSECOND)
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
		time.Driver_Run_Until(driver, -1*time.NANOSECOND,
			func() (finished bool) { return false })
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

// SIM_OPERATION_CAPACITY is one because these unit tests arm operations serially.
const SIM_OPERATION_CAPACITY = 1

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
		resolution := time.Virtual_Clock{Resolution: time.Duration(value)}
		time.Virtual_Clock_To_Clock(&resolution)
		epoch := time.Virtual_Clock{Epoch: time.Moment(value)}
		time.Virtual_Clock_To_Clock(&epoch)
		verify_virtual_timeline_domain(resolution)
		verify_virtual_timeline_domain(epoch)
	}
}

func verify_virtual_timeline_domain(virtual time.Virtual_Clock) {
	state := time.Virtual_Timeline{}
	queue := [SIM_OPERATION_CAPACITY]*time.Completion{}
	events := [SIM_OPERATION_CAPACITY]time.Virtual_Event{}
	time.New_Virtual_Timeline(&state, virtual, time.Virtual_Timeline_Memory{
		Queue: queue[:], Events: events[:],
	})
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
	grain_virtual := time.Virtual_Clock{Resolution: time.NANOSECOND}
	grain := time.Virtual_Clock_To_Clock(&grain_virtual)
	time.Clock_Now_Monotonic(grain)
	time.Virtual_Clock_Tick(&grain_virtual)
	time.Clock_Now_Monotonic(grain)
	time.Virtual_Clock_Tick(&grain_virtual)
	time.Clock_Now_Monotonic(grain)

	bound := time.Virtual_Clock{Resolution: time.Duration(time.MONOTONIC_MOMENT_MAXIMUM)}
	one_year := time.Virtual_Clock_To_Clock(&bound)
	time.Virtual_Clock_Tick(&bound)
	time.Clock_Now_Monotonic(one_year)

	_, driver, timeline := sim_loop_with_virtual(bound)
	time.Driver_Run(driver)
	time.Clock_Now_Monotonic(timeline)
}

// Build deterministic loop plus its driver, for one seed of test.
func sim_loop(_ uint64) (loop time.Timeline, driver time.Driver, clock time.Clock) {
	return sim_loop_with_virtual(time.Virtual_Clock{Resolution: time.NANOSECOND})
}

func sim_loop_with_virtual(
	virtual time.Virtual_Clock,
) (loop time.Timeline, driver time.Driver, clock time.Clock) {
	state := time.Virtual_Timeline{}
	queue := [SIM_OPERATION_CAPACITY]*time.Completion{}
	events := [SIM_OPERATION_CAPACITY]time.Virtual_Event{}
	return time.New_Virtual_Timeline(&state, virtual, time.Virtual_Timeline_Memory{
		Queue: queue[:], Events: events[:],
	})
}

// Test_Invariant_Domains check special values of each admitted scalar domain.
func Test_Invariant_Domains(t *testing.T) {
	t.Parallel()
	verify_virtual_clock_domains()
	verify_skew_domains()
	verify_uptime_domains()
}

// Escaped results expose heap ownership hidden by stack-only use.
func verify_time_constructor_allocations(t *testing.T) {
	t.Run("Skew", func(t *testing.T) {
		kinds := [...]time.Skew_Kind{
			time.SKEW_KIND_LINEAR,
			time.SKEW_KIND_PERIODIC,
			time.SKEW_KIND_STEP,
		}
		expected := [...]time.Duration{2, 0, 0}
		for index, kind := range kinds {
			var offset time.Offset
			testify.Zero_Allocation(t, func() { offset = time.Skew(kind, 1, 1) })
			testify.Equal(t, expected[index], time.Offset_Read(offset, 1))
		}
	})
	t.Run("Virtual_Clock_To_Clock", func(t *testing.T) {
		virtual := time.Virtual_Clock{Resolution: 1}
		var clock time.Clock
		testify.Zero_Allocation(t, func() {
			clock = time.Virtual_Clock_To_Clock(&virtual)
		})
		testify.Not_Nil(t, clock.Now_Monotonic)
		testify.Not_Nil(t, clock.State)
	})
	t.Run("New_Virtual_Timeline", func(t *testing.T) {
		virtual := time.Virtual_Clock{Resolution: 1}
		state := time.Virtual_Timeline{}
		queue := [SIM_OPERATION_CAPACITY]*time.Completion{}
		events := [SIM_OPERATION_CAPACITY]time.Virtual_Event{}
		memory := time.Virtual_Timeline_Memory{Queue: queue[:], Events: events[:]}
		var loop time.Timeline
		var driver time.Driver
		var clock time.Clock
		testify.Zero_Allocation(t, func() {
			loop, driver, clock = time.New_Virtual_Timeline(&state, virtual, memory)
		})
		testify.Not_Nil(t, loop.Submit)
		testify.Not_Nil(t, driver.Run)
		testify.Not_Nil(t, clock.Now_Realtime)
	})
}

func verify_time_reader_allocations(t *testing.T) {
	t.Run("Offset_Read", func(t *testing.T) {
		offset := time.Skew(time.SKEW_KIND_LINEAR, 1, 1)
		var skew time.Duration
		testify.Zero_Allocation(t, func() { skew = time.Offset_Read(offset, 1) })
		testify.Equal(t, time.Duration(2), skew)
	})
	virtual := time.Virtual_Clock{Resolution: 1}
	clock := time.Virtual_Clock_To_Clock(&virtual)
	time.Virtual_Clock_Tick(&virtual)
	var monotonic time.Monotonic_Moment
	var realtime time.Moment
	t.Run("Now_Monotonic", func(t *testing.T) {
		testify.Zero_Allocation(t, func() { monotonic = time.Clock_Now_Monotonic(clock) })
		testify.Positive(t, monotonic)
	})
	t.Run("Now_Realtime", func(t *testing.T) {
		testify.Zero_Allocation(t, func() { realtime = time.Clock_Now_Realtime(clock) })
		testify.Positive(t, realtime)
	})
	t.Run("Tick", func(t *testing.T) {
		testify.Zero_Allocation(t, func() { time.Virtual_Clock_Tick(&virtual) })
	})
}

func verify_timeline_submission_allocations(t *testing.T) {
	t.Run("Callback", func(t *testing.T) {
		completion := time.Completion{}
		callback := time.Callback(allocation_callback)
		testify.Zero_Allocation(t, func() { callback(&completion) })
		testify.Positive(t, completion.Data)
	})
	t.Run("Submit", func(t *testing.T) {
		loop, driver, _ := sim_loop(0)
		completion := time.Completion{}
		var run_err error
		testify.Zero_Allocation(t, func() {
			time.Timeline_Submit(loop, &completion, 0, allocation_callback)
			run_err = time.Driver_Run(driver)
		})
		testify.No_Error(t, run_err)
	})
	t.Run("Timeout", func(t *testing.T) {
		loop, driver, _ := sim_loop(0)
		completion := time.Completion{}
		var run_err error
		testify.Zero_Allocation(t, func() {
			time.Timeline_Timeout(loop, &completion, 1, allocation_callback)
			run_err = time.Driver_Run_For(driver, 2)
		})
		testify.No_Error(t, run_err)
	})
}

func verify_timeline_event_allocations(t *testing.T) {
	t.Run("Listen_Trigger", func(t *testing.T) {
		loop, driver, _ := sim_loop(0)
		event, err := time.Timeline_Open_Event(loop)
		testify.No_Error(t, err)
		completion := time.Completion{}
		var run_err error
		testify.Zero_Allocation(t, func() {
			time.Timeline_Event_Listen(loop, event, &completion, allocation_callback)
			time.Timeline_Event_Trigger(loop, event, &completion)
			run_err = time.Driver_Run(driver)
		})
		testify.No_Error(t, run_err)
		time.Timeline_Close_Event(loop, event)
	})
	t.Run("Open_Close", func(t *testing.T) {
		loop, _, _ := sim_loop(0)
		var opened time.Event
		var open_err error
		testify.Zero_Allocation(t, func() {
			opened, open_err = time.Timeline_Open_Event(loop)
			time.Timeline_Close_Event(loop, opened)
		})
		testify.No_Error(t, open_err)
	})
}

func verify_driver_allocations(t *testing.T) {
	t.Run("Run", func(t *testing.T) {
		_, driver, _ := sim_loop(0)
		var run_err error
		testify.Zero_Allocation(t, func() { run_err = time.Driver_Run(driver) })
		testify.No_Error(t, run_err)
	})
	t.Run("Run_For", func(t *testing.T) {
		_, driver, _ := sim_loop(0)
		var run_err error
		testify.Zero_Allocation(t, func() { run_err = time.Driver_Run_For(driver, 1) })
		testify.No_Error(t, run_err)
	})
	t.Run("Run_Until", func(t *testing.T) {
		loop, driver, _ := sim_loop(0)
		completion := time.Completion{}
		done := func() (finished bool) { return !completion.Armed }
		var completed bool
		var run_err error
		testify.Zero_Allocation(t, func() {
			time.Timeline_Timeout(loop, &completion, 1, allocation_callback)
			completed, run_err = time.Driver_Run_Until(driver, 2, done)
		})
		testify.True(t, completed)
		testify.No_Error(t, run_err)
	})
	t.Run("Deinit", func(t *testing.T) {
		_, driver, _ := sim_loop(0)
		testify.Zero_Allocation(t, func() { time.Driver_Deinit(driver) })
	})
}

func allocation_callback(completion *time.Completion) {
	completion.Data++
}

// Every time API path needs direct heap evidence; functional tests cannot establish it.
func Test_Time_API_Heap_Allocation(t *testing.T) {
	verify_time_constructor_allocations(t)
	verify_time_reader_allocations(t)
	verify_timeline_submission_allocations(t)
	verify_timeline_event_allocations(t)
	verify_driver_allocations(t)
}

// A trigger can arrive before its listener, so that deferred enqueue needs the same bound.
func verify_pending_trigger_capacity(t *testing.T) {
	t.Helper()
	loop, driver, _ := sim_loop(0)
	var first time.Completion
	time.Timeline_Submit(loop, &first, 2*time.NANOSECOND, allocation_callback)
	event, open_err := time.Timeline_Open_Event(loop)
	if !testify.No_Error(t, open_err) {
		return
	}
	var listener time.Completion
	time.Timeline_Event_Trigger(loop, event, &listener)
	fired := false
	callback := func(_ *time.Completion) { fired = true }
	testify.Panics(t, func() {
		time.Timeline_Event_Listen(loop, event, &listener, callback)
	})
	testify.False(t, listener.Armed)
	time.Driver_Run_For(driver, 3*time.NANOSECOND)
	time.Timeline_Event_Listen(loop, event, &listener, callback)
	time.Driver_Run(driver)
	if !testify.True(t, fired) {
		return
	}
	time.Timeline_Close_Event(loop, event)
}
