package backoff_test

import (
	"errors"
	"testing"
	"unsafe"

	"local/james-orcales/shared/backoff"
	"local/james-orcales/shared/random/prng"
	"local/james-orcales/shared/simulation/time"
	"local/james-orcales/shared/testify"
)

// Test_Constant_Waits_Fixed checks Constant returns a fixed interval and Reset is inert.
func Test_Constant_Waits_Fixed(t *testing.T) {
	var state backoff.Policy_State
	policy := backoff.Constant(&state, backoff.Initial_Interval(5*time.SECOND))
	for draw_index := 0; draw_index < 4; draw_index++ {
		if backoff.Policy_Next(policy) != backoff.Delay(5*time.SECOND) {
			t.Fatalf("draw %d was not the fixed interval", draw_index)
		}
	}
	backoff.Policy_Reset(policy)
	if backoff.Policy_Next(policy) != backoff.Delay(5*time.SECOND) {
		t.Fatalf("reset changed a constant policy")
	}
}

// Test_Zero_Never_Waits checks Zero always yields a zero delay.
func Test_Zero_Never_Waits(t *testing.T) {
	var state backoff.Policy_State
	policy := backoff.Zero(&state)
	if backoff.Policy_Next(policy) != 0 {
		t.Fatalf("zero policy returned a non-zero delay")
	}
}

// Test_Stopped_Never_Retries checks Stopped always yields STOP.
func Test_Stopped_Never_Retries(t *testing.T) {
	var state backoff.Policy_State
	policy := backoff.Stopped(&state)
	if backoff.Policy_Next(policy) != backoff.STOP {
		t.Fatalf("stopped policy did not return STOP")
	}
}

// Test_Exponential_Grows_And_Caps checks the interval doubles and clamps at the cap.
func Test_Exponential_Grows_And_Caps(t *testing.T) {
	generator := prng.New(1)
	var state backoff.Policy_State
	policy := backoff.Exponential(&state, &backoff.Exponential_Input{
		Initial_Interval: backoff.Initial_Interval(1 * time.SECOND),
		Interval_Max:     backoff.Maximum_Interval(10 * time.SECOND),
		Multiplier:       prng.Ratio{Numerator: 2, Denominator: 1},
		Jitter:           prng.Ratio{Numerator: 0, Denominator: 1},
		Generator:        &generator,
	})
	want := []time.Duration{
		1 * time.SECOND, 2 * time.SECOND, 4 * time.SECOND, 8 * time.SECOND,
		10 * time.SECOND, 10 * time.SECOND,
	}
	for index, expected := range want {
		got := backoff.Policy_Next(policy)
		if got != backoff.Delay(expected) {
			t.Fatalf("Next %d = %d, want %d", index, got, expected)
		}
	}
}

// Test_Jitter_Stays_Within_Bounds checks a jittered delay never leaves its spread.
func Test_Jitter_Stays_Within_Bounds(t *testing.T) {
	generator := prng.New(2)
	interval := 1000 * time.NANOSECOND
	var state backoff.Policy_State
	policy := backoff.Exponential(&state, &backoff.Exponential_Input{
		Initial_Interval: backoff.Initial_Interval(interval),
		Interval_Max:     backoff.Maximum_Interval(interval),
		Multiplier:       prng.Ratio{Numerator: 1, Denominator: 1},
		Jitter:           prng.Ratio{Numerator: 1, Denominator: 2},
		Generator:        &generator,
	})
	low := interval - interval/2
	high := interval + interval/2
	for draw_index := 0; draw_index < 1000; draw_index++ {
		got := backoff.Policy_Next(policy)
		if got < backoff.Delay(low) {
			t.Fatalf("draw %d = %d, below %d", draw_index, got, low)
		}
		if got > backoff.Delay(high) {
			t.Fatalf("draw %d = %d, above %d", draw_index, got, high)
		}
	}
}

// Test_Exponential_Saturates_Wide_Product checks full-width terms cannot wrap interval math.
func Test_Exponential_Saturates_Wide_Product(t *testing.T) {
	generator := prng.New(2)
	input := backoff.Exponential_Input{
		Initial_Interval: backoff.INTERVAL_MAXIMUM,
		Interval_Max:     backoff.Maximum_Interval(backoff.INTERVAL_MAXIMUM),
		Multiplier:       prng.Ratio{Numerator: ^uint64(0), Denominator: 1},
		Jitter:           prng.Ratio{Denominator: 1},
		Generator:        &generator,
	}
	var state backoff.Policy_State
	policy := backoff.Exponential(&state, &input)
	if backoff.Policy_Next(policy) != backoff.Delay(backoff.INTERVAL_MAXIMUM) {
		t.Fatal("wide multiplier did not saturate")
	}
	input.Multiplier = prng.Ratio{Numerator: 1, Denominator: 1}
	input.Jitter = prng.Ratio{Numerator: ^uint64(0), Denominator: ^uint64(0)}
	policy = backoff.Exponential(&state, &input)
	delay := backoff.Policy_Next(policy)
	if delay < 0 {
		t.Fatal("wide jitter escaped interval domain")
	}
	if delay > backoff.Delay(backoff.INTERVAL_MAXIMUM) {
		t.Fatal("wide jitter escaped interval domain")
	}
}

// Test_Seed_Reproduces_Delays checks one seed replays and distinct seeds diverge.
func Test_Seed_Reproduces_Delays(t *testing.T) {
	first := prng.New(7)
	again := prng.New(7)
	var state_first backoff.Policy_State
	var state_again backoff.Policy_State
	policy_first := backoff.New_Exponential(&state_first, &first)
	policy_again := backoff.New_Exponential(&state_again, &again)
	for draw_index := 0; draw_index < 20; draw_index++ {
		if backoff.Policy_Next(policy_first) != backoff.Policy_Next(policy_again) {
			t.Fatalf("draw %d diverged for the same seed", draw_index)
		}
	}
	other := prng.New(8)
	repeat := prng.New(7)
	var state_other backoff.Policy_State
	var state_repeat backoff.Policy_State
	policy_other := backoff.New_Exponential(&state_other, &other)
	policy_repeat := backoff.New_Exponential(&state_repeat, &repeat)
	if backoff.Policy_Next(policy_other) == backoff.Policy_Next(policy_repeat) {
		t.Fatalf("distinct seeds produced the same first delay")
	}
}

// Test_Retry_Returns_First_Success checks a first-try success stops immediately.
func Test_Retry_Returns_First_Success(t *testing.T) {
	calls := 0
	operation := func(_ *backoff.Retry_State[int]) (value int, err error) {
		calls++
		return 42, nil
	}
	var state backoff.Policy_State
	outcome := run_retry(backoff.Zero(&state), 3, operation)
	if outcome.Error != nil {
		t.Fatalf("error = %v, want nil", outcome.Error)
	}
	if outcome.Value != 42 {
		t.Fatalf("value = %d, want 42", outcome.Value)
	}
	if calls != 1 {
		t.Fatalf("operation ran %d times, want 1", calls)
	}
}

// Test_Retry_Stops_On_Permanent checks a Permanent error ends Retry after one attempt.
func Test_Retry_Stops_On_Permanent(t *testing.T) {
	cause := errors.New("fatal")
	calls := 0
	var permanent backoff.Permanent_Error
	operation := func(_ *backoff.Retry_State[int]) (value int, err error) {
		calls++
		return 0, backoff.Permanent(&permanent, cause)
	}
	var state backoff.Policy_State
	outcome := run_retry(backoff.Zero(&state), 5, operation)
	if !bool(backoff.Error_Matches(outcome.Error, backoff.Error_Permanent)) {
		t.Fatalf("error = %v, want a permanent error", outcome.Error)
	}
	if !bool(backoff.Error_Matches(outcome.Error, cause)) {
		t.Fatalf("error = %v, want it to wrap the cause", outcome.Error)
	}
	if calls != 1 {
		t.Fatalf("operation ran %d times, want 1", calls)
	}
}

// Test_Retry_Exhausts_After_Tries checks Retry gives up after Tries_Max failures.
func Test_Retry_Exhausts_After_Tries(t *testing.T) {
	transient := errors.New("transient")
	calls := 0
	operation := func(_ *backoff.Retry_State[int]) (value int, err error) {
		calls++
		return 0, transient
	}
	var state backoff.Policy_State
	outcome := run_retry(backoff.Constant(
		&state, backoff.Initial_Interval(time.MILLISECOND),
	), 3, operation)
	if !bool(backoff.Error_Matches(outcome.Error, backoff.Error_Exhausted)) {
		t.Fatalf("error = %v, want exhausted", outcome.Error)
	}
	if !bool(backoff.Error_Matches(outcome.Error, transient)) {
		t.Fatalf("error = %v, want it to wrap the last failure", outcome.Error)
	}
	if calls != 3 {
		t.Fatalf("operation ran %d times, want 3", calls)
	}
}

// Test_Retry_Waits_On_Timeline checks the between-attempt waits advance the io clock.
func Test_Retry_Waits_On_Timeline(t *testing.T) {
	interval := 5 * time.MILLISECOND
	calls := 0
	operation := func(_ *backoff.Retry_State[int]) (value int, err error) {
		calls++
		if calls < 3 {
			return 0, errors.New("transient")
		}
		return 1, nil
	}
	var state backoff.Policy_State
	outcome := run_retry(
		backoff.Constant(&state, backoff.Initial_Interval(interval)), 5, operation,
	)
	if outcome.Error != nil {
		t.Fatalf("error = %v, want nil", outcome.Error)
	}
	// Two waits precede the third, successful attempt.
	if outcome.Elapsed != 2*interval {
		t.Fatalf("elapsed = %d, want %d", outcome.Elapsed, 2*interval)
	}
}

// Test_Retry_After_Overrides_Delay checks Retry_After replaces the policy delay once.
func Test_Retry_After_Overrides_Delay(t *testing.T) {
	override := 20 * time.MILLISECOND
	calls := 0
	var retry_after backoff.Retry_After_Error
	operation := func(_ *backoff.Retry_State[int]) (value int, err error) {
		calls++
		if calls == 1 {
			return 0, backoff.Retry_After(
				&retry_after, backoff.Retry_Delay(override),
				errors.New("slow down"),
			)
		}
		return 7, nil
	}
	// The policy delay is a millisecond, but Retry_After overrides the single wait.
	var state backoff.Policy_State
	outcome := run_retry(backoff.Constant(
		&state, backoff.Initial_Interval(time.MILLISECOND),
	), 5, operation)
	if outcome.Error != nil {
		t.Fatalf("error = %v, want nil", outcome.Error)
	}
	if outcome.Elapsed != override {
		t.Fatalf("elapsed = %d, want %d", outcome.Elapsed, override)
	}
}

// Test_Invariant_Domains reaches every bounded policy, retry, and error sentinel.
func Test_Invariant_Domains(t *testing.T) {
	policy_invariant_domains()
	retry_invariant_domains()
	error_invariant_domains(t)
}

func policy_invariant_domains() {
	generator := prng.New(11)
	intervals := invariant_intervals()
	states := invariant_policy_states(&generator)
	for index, interval := range intervals {
		state := states[index]
		policy := backoff.Constant(&state, interval)
		backoff.Policy_Next(policy)
		backoff.Policy_Reset(policy)

		state = states[index]
		policy = backoff.Zero(&state)
		backoff.Policy_Next(policy)
		backoff.Policy_Reset(policy)

		state = states[index]
		policy = backoff.Stopped(&state)
		backoff.Policy_Next(policy)
		backoff.Policy_Reset(policy)

		state = states[index]
		policy = backoff.Exponential(&state, &backoff.Exponential_Input{
			Initial_Interval: interval,
			Interval_Max:     backoff.Maximum_Interval(interval),
			Multiplier:       prng.Ratio{Numerator: 1, Denominator: 1},
			Jitter:           prng.Ratio{Numerator: 0, Denominator: 1},
			Generator:        &generator,
		})
		backoff.Policy_Next(policy)
		backoff.Policy_Reset(policy)

		state = states[index]
		policy = backoff.New_Exponential(&state, &generator)
		backoff.Policy_Next(policy)
		backoff.Policy_Reset(policy)
	}
}

func retry_invariant_domains() {
	generator := prng.New(12)
	intervals := invariant_intervals()
	states := invariant_policy_states(&generator)
	var timeline time.Virtual_Timeline
	queue := [ALLOCATION_TIMELINE_CAPACITY]*time.Completion{}
	events := [ALLOCATION_TIMELINE_CAPACITY]time.Virtual_Event{}
	loop, _, clock := time.New_Virtual_Timeline(
		&timeline,
		time.Virtual_Clock{Resolution: time.NANOSECOND},
		time.Virtual_Timeline_Memory{Queue: queue[:], Events: events[:]},
	)
	tries := [...]backoff.Try_Count{1, 2, 2, backoff.TRY_COUNT_MAXIMUM}
	attempts := [...]backoff.Attempt_Count{0, 1, 2, backoff.Attempt_Count(
		backoff.TRY_COUNT_MAXIMUM,
	)}
	for index, interval := range intervals {
		policy_storage := states[index]
		policy := backoff.Zero(&policy_storage)
		state := backoff.Retry_State[int]{
			Input: backoff.Retry_Input{
				Timer: loop, Policy: policy, Tries_Max: tries[index], Clock: clock,
				Elapsed_Time_Max: backoff.Elapsed_Limit(interval),
			},
			Try_Count: attempts[index],
			Started:   backoff.Started_Moment(interval),
			Stopped:   true,
		}
		backoff.Retry_Rearm(&state)
		backoff.Retry_Work_Queued(&state)
		backoff.Retry_Stopped(&state)
		backoff.Retry_Status(&state)
		state.Stopped = false
		backoff.Retry_Status(&state)
		state.Stopped = true
		backoff.Retry(&state, invariant_success)
	}
}

func invariant_success(_ *backoff.Retry_State[int]) (result int, err error) {
	return 1, nil
}

func error_invariant_domains(t *testing.T) {
	intervals := invariant_intervals()
	var retry_after backoff.Retry_After_Error
	for _, interval := range intervals {
		backoff.Retry_After(
			&retry_after, backoff.Retry_Delay(interval), backoff.Error_Exhausted,
		)
	}
	backoff.Retry_After(&retry_after, 0, backoff.Error_Exhausted)
	var permanent backoff.Permanent_Error
	marked := backoff.Permanent(&permanent, backoff.Error_Exhausted)
	if !bool(backoff.Error_Matches(marked, backoff.Error_Permanent)) {
		t.Fatal("permanent marker did not match")
	}
	if bool(backoff.Error_Matches(marked, backoff.Error_Elapsed_Max)) {
		t.Fatal("permanent marker matched unrelated sentinel")
	}
}

func invariant_intervals() (
	intervals [INVARIANT_SENTINEL_COUNT]backoff.Initial_Interval,
) {
	return [INVARIANT_SENTINEL_COUNT]backoff.Initial_Interval{
		0, 1, 2, backoff.INTERVAL_MAXIMUM,
	}
}

func invariant_policy_states(
	generator *prng.Generator,
) (states [INVARIANT_SENTINEL_COUNT]backoff.Policy_State) {
	return [INVARIANT_SENTINEL_COUNT]backoff.Policy_State{
		policy_state(backoff.POLICY_KIND_CONSTANT, 0, generator),
		policy_state(backoff.POLICY_KIND_STOPPED, 1, generator),
		policy_state(backoff.POLICY_KIND_EXPONENTIAL, 2, generator),
		policy_state(backoff.POLICY_KIND_CONSTANT, backoff.INTERVAL_MAXIMUM, generator),
	}
}

func policy_state(
	kind backoff.Policy_Kind, interval backoff.Initial_Interval,
	generator *prng.Generator,
) (state backoff.Policy_State) {
	return backoff.Policy_State{
		Kind:             kind,
		Initial_Interval: interval,
		Current_Interval: backoff.Current_Interval(interval),
		Interval_Max:     backoff.Maximum_Interval(interval),
		Multiplier:       prng.Ratio{Numerator: 1, Denominator: 1},
		Jitter:           prng.Ratio{Numerator: 0, Denominator: 1},
		Generator:        generator,
	}
}

type Allocation_Fixture struct {
	Generator          prng.Generator
	Input              backoff.Exponential_Input
	Constant           backoff.Policy_State
	Exponential        backoff.Policy_State
	Zero               backoff.Policy_State
	Stopped            backoff.Policy_State
	Policy_Constant    backoff.Policy
	Policy_Exponential backoff.Policy
	Policy_Zero        backoff.Policy
	Policy_Stopped     backoff.Policy
	Delay              backoff.Delay
	Match              backoff.Boolean
	Permanent          backoff.Permanent_Error
	After              backoff.Retry_After_Error
	Error              error
	Retry_State        backoff.Retry_State[int]
	Result             int
	Result_Error       error
	Operation_Error    error
	Operation_Kind     uint8
	Operation_Count    int
	Notify_Count       int
	Driver             time.Driver
}

const ALLOCATION_TIMELINE_CAPACITY = 1

const INVARIANT_SENTINEL_COUNT = 4

const ALLOCATION_OPERATION_SUCCESS uint8 = 0

const ALLOCATION_OPERATION_FAILURE uint8 = 1

const ALLOCATION_OPERATION_PERMANENT uint8 = 2

const ALLOCATION_OPERATION_RETRY_AFTER uint8 = 3

func allocation_operation(state *backoff.Retry_State[int]) (result int, err error) {
	fixture := (*Allocation_Fixture)(state.Context)
	fixture.Operation_Count++
	switch fixture.Operation_Kind {
	case ALLOCATION_OPERATION_FAILURE:
		return 0, fixture.Operation_Error
	case ALLOCATION_OPERATION_PERMANENT:
		return 0, backoff.Permanent(&fixture.Permanent, fixture.Operation_Error)
	case ALLOCATION_OPERATION_RETRY_AFTER:
		if fixture.Operation_Count == 1 {
			return 0, backoff.Retry_After(
				&fixture.After, backoff.Retry_Delay(time.NANOSECOND),
				fixture.Operation_Error,
			)
		}
	}
	return 1, nil
}

func allocation_notify(context unsafe.Pointer, _ error, _ backoff.Delay) {
	fixture := (*Allocation_Fixture)(context)
	fixture.Notify_Count++
}

// Test_Allocation proves every public runtime path owns no heap storage.
func Test_Allocation(t *testing.T) {
	fixture := Allocation_Fixture{Generator: prng.New(1)}
	var timeline time.Virtual_Timeline
	queue := [ALLOCATION_TIMELINE_CAPACITY]*time.Completion{}
	events := [ALLOCATION_TIMELINE_CAPACITY]time.Virtual_Event{}
	loop, driver, clock := time.New_Virtual_Timeline(
		&timeline,
		time.Virtual_Clock{Resolution: time.NANOSECOND},
		time.Virtual_Timeline_Memory{Queue: queue[:], Events: events[:]},
	)
	fixture.Input = backoff.Exponential_Input{
		Initial_Interval: backoff.Initial_Interval(time.SECOND),
		Interval_Max:     backoff.Maximum_Interval(10 * time.SECOND),
		Multiplier:       prng.Ratio{Numerator: 2, Denominator: 1},
		Jitter:           prng.Ratio{Numerator: 1, Denominator: 2},
		Generator:        &fixture.Generator,
	}
	fixture.Retry_State.Input = backoff.Retry_Input{
		Timer: loop, Policy: backoff.Zero(&fixture.Constant), Tries_Max: 1, Clock: clock,
		Notify: allocation_notify,
	}
	fixture.Retry_State.Context = unsafe.Pointer(&fixture)
	fixture.Driver = driver
	fixture.Operation_Error = backoff.Error_Exhausted
	allocation_policy_paths(t, &fixture)
	allocation_error_paths(t, &fixture)
	allocation_retry_paths(t, &fixture)
}

func allocation_policy_paths(t *testing.T, fixture *Allocation_Fixture) {
	testify.Zero_Allocation(t, func() {
		fixture.Policy_Constant = backoff.Constant(
			&fixture.Constant, backoff.Initial_Interval(time.SECOND),
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Policy_Zero = backoff.Zero(&fixture.Zero)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Policy_Stopped = backoff.Stopped(&fixture.Stopped)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Policy_Exponential = backoff.Exponential(
			&fixture.Exponential, &fixture.Input,
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Policy_Exponential = backoff.New_Exponential(
			&fixture.Exponential, &fixture.Generator,
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Delay = backoff.Policy_Next(fixture.Policy_Constant)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Delay = backoff.Policy_Next(fixture.Policy_Zero)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Delay = backoff.Policy_Next(fixture.Policy_Stopped)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Delay = backoff.Policy_Next(fixture.Policy_Exponential)
	})
	testify.Zero_Allocation(t, func() { backoff.Policy_Reset(fixture.Policy_Constant) })
	testify.Zero_Allocation(t, func() { backoff.Policy_Reset(fixture.Policy_Zero) })
	testify.Zero_Allocation(t, func() { backoff.Policy_Reset(fixture.Policy_Stopped) })
	testify.Zero_Allocation(t, func() {
		backoff.Policy_Reset(fixture.Policy_Exponential)
	})
}

func allocation_error_paths(t *testing.T, fixture *Allocation_Fixture) {
	testify.Zero_Allocation(t, func() {
		fixture.Error = backoff.Permanent(&fixture.Permanent, backoff.Error_Exhausted)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Error = backoff.Retry_After(
			&fixture.After, backoff.Retry_Delay(time.SECOND), backoff.Error_Exhausted,
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Error = backoff.Error_Exhausted
		fixture.Match = backoff.Error_Matches(
			fixture.Error, backoff.Error_Exhausted,
		)
		fixture.Match = backoff.Error_Matches(
			fixture.Error, backoff.Error_Elapsed_Max,
		)
	})
}

func allocation_retry_paths(t *testing.T, fixture *Allocation_Fixture) {
	fixture.Operation_Kind = ALLOCATION_OPERATION_SUCCESS
	testify.Zero_Allocation(t, func() {
		backoff.Retry(&fixture.Retry_State, allocation_operation)
		fixture.Match, fixture.Result, fixture.Result_Error = backoff.Retry_Status(
			&fixture.Retry_State,
		)
	})
	fixture.Operation_Kind = ALLOCATION_OPERATION_FAILURE
	fixture.Retry_State.Input.Tries_Max = 1
	testify.Zero_Allocation(t, func() {
		backoff.Retry(&fixture.Retry_State, allocation_operation)
		fixture.Match, fixture.Result, fixture.Result_Error = backoff.Retry_Status(
			&fixture.Retry_State,
		)
	})
	fixture.Operation_Kind = ALLOCATION_OPERATION_PERMANENT
	fixture.Retry_State.Input.Tries_Max = 2
	testify.Zero_Allocation(t, func() {
		backoff.Retry(&fixture.Retry_State, allocation_operation)
		fixture.Match, fixture.Result, fixture.Result_Error = backoff.Retry_Status(
			&fixture.Retry_State,
		)
	})
	fixture.Operation_Kind = ALLOCATION_OPERATION_FAILURE
	fixture.Retry_State.Input.Policy = fixture.Policy_Stopped
	testify.Zero_Allocation(t, func() {
		backoff.Retry(&fixture.Retry_State, allocation_operation)
		fixture.Match, fixture.Result, fixture.Result_Error = backoff.Retry_Status(
			&fixture.Retry_State,
		)
	})
	fixture.Retry_State.Input.Policy = fixture.Policy_Zero
	fixture.Retry_State.Input.Tries_Max = 2
	testify.Zero_Allocation(t, func() {
		backoff.Retry(&fixture.Retry_State, allocation_operation)
		fixture.Match = backoff.Retry_Rearm(&fixture.Retry_State)
		fixture.Match, fixture.Result, fixture.Result_Error = backoff.Retry_Status(
			&fixture.Retry_State,
		)
	})
	fixture.Retry_State.Input.Policy = fixture.Policy_Constant
	fixture.Retry_State.Input.Elapsed_Time_Max = 1
	testify.Zero_Allocation(t, func() {
		backoff.Retry(&fixture.Retry_State, allocation_operation)
		fixture.Match, fixture.Result, fixture.Result_Error = backoff.Retry_Status(
			&fixture.Retry_State,
		)
	})
	fixture.Operation_Kind = ALLOCATION_OPERATION_RETRY_AFTER
	fixture.Retry_State.Input.Elapsed_Time_Max = 0
	testify.Zero_Allocation(t, func() {
		fixture.Operation_Count = 0
		backoff.Retry(&fixture.Retry_State, allocation_operation)
		fixture.Match, fixture.Result, fixture.Result_Error = backoff.Retry_Status(
			&fixture.Retry_State,
		)
		time.Driver_Run(fixture.Driver)
		time.Driver_Run(fixture.Driver)
		fixture.Match = backoff.Retry_Work_Queued(&fixture.Retry_State)
		fixture.Match = backoff.Retry_Rearm(&fixture.Retry_State)
		fixture.Match = backoff.Retry_Stopped(&fixture.Retry_State)
		fixture.Match, fixture.Result, fixture.Result_Error = backoff.Retry_Status(
			&fixture.Retry_State,
		)
	})
}
