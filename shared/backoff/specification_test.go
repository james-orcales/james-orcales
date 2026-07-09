package backoff_test

import (
	"errors"
	"testing"

	"local/james-orcales/shared/backoff"
	"local/james-orcales/shared/prng"
	"local/james-orcales/shared/time"
)

// Test_Constant_Waits_Fixed checks Constant returns a fixed interval and Reset is inert.
func Test_Constant_Waits_Fixed(t *testing.T) {
	policy := backoff.Constant(5 * time.SECOND)
	for draw_index := 0; draw_index < 4; draw_index++ {
		if policy.Next() != 5*time.SECOND {
			t.Fatalf("draw %d was not the fixed interval", draw_index)
		}
	}
	policy.Reset()
	if policy.Next() != 5*time.SECOND {
		t.Fatalf("reset changed a constant policy")
	}
}

// Test_Zero_Never_Waits checks Zero always yields a zero delay.
func Test_Zero_Never_Waits(t *testing.T) {
	policy := backoff.Zero()
	if policy.Next() != 0 {
		t.Fatalf("zero policy returned a non-zero delay")
	}
}

// Test_Stopped_Never_Retries checks Stopped always yields STOP.
func Test_Stopped_Never_Retries(t *testing.T) {
	policy := backoff.Stopped()
	if policy.Next() != backoff.STOP {
		t.Fatalf("stopped policy did not return STOP")
	}
}

// Test_Exponential_Grows_And_Caps checks the interval doubles and clamps at the cap.
func Test_Exponential_Grows_And_Caps(t *testing.T) {
	generator := prng.New(1)
	policy := backoff.Exponential(&backoff.Exponential_Input{
		Initial_Interval: 1 * time.SECOND,
		Interval_Max:     10 * time.SECOND,
		Multiplier:       prng.Ratio{Numerator: 2, Denominator: 1},
		Jitter:           prng.Ratio{Numerator: 0, Denominator: 1},
		Generator:        &generator,
	})
	want := []time.Duration{
		1 * time.SECOND, 2 * time.SECOND, 4 * time.SECOND, 8 * time.SECOND,
		10 * time.SECOND, 10 * time.SECOND,
	}
	for index, expected := range want {
		got := policy.Next()
		if got != expected {
			t.Fatalf("Next %d = %d, want %d", index, got, expected)
		}
	}
}

// Test_Jitter_Stays_Within_Bounds checks a jittered delay never leaves its spread.
func Test_Jitter_Stays_Within_Bounds(t *testing.T) {
	generator := prng.New(2)
	interval := 1000 * time.NANOSECOND
	policy := backoff.Exponential(&backoff.Exponential_Input{
		Initial_Interval: interval,
		Interval_Max:     interval,
		Multiplier:       prng.Ratio{Numerator: 1, Denominator: 1},
		Jitter:           prng.Ratio{Numerator: 1, Denominator: 2},
		Generator:        &generator,
	})
	low := interval - interval/2
	high := interval + interval/2
	for draw_index := 0; draw_index < 1000; draw_index++ {
		got := policy.Next()
		if got < low {
			t.Fatalf("draw %d = %d, below %d", draw_index, got, low)
		}
		if got > high {
			t.Fatalf("draw %d = %d, above %d", draw_index, got, high)
		}
	}
}

// Test_Seed_Reproduces_Delays checks one seed replays and distinct seeds diverge.
func Test_Seed_Reproduces_Delays(t *testing.T) {
	first := prng.New(7)
	again := prng.New(7)
	policy_first := backoff.New_Exponential(&first)
	policy_again := backoff.New_Exponential(&again)
	for draw_index := 0; draw_index < 20; draw_index++ {
		if policy_first.Next() != policy_again.Next() {
			t.Fatalf("draw %d diverged for the same seed", draw_index)
		}
	}
	other := prng.New(8)
	repeat := prng.New(7)
	policy_other := backoff.New_Exponential(&other)
	policy_repeat := backoff.New_Exponential(&repeat)
	if policy_other.Next() == policy_repeat.Next() {
		t.Fatalf("distinct seeds produced the same first delay")
	}
}

// Test_Retry_Returns_First_Success checks a first-try success stops immediately.
func Test_Retry_Returns_First_Success(t *testing.T) {
	calls := 0
	operation := func() (value int, err error) {
		calls++
		return 42, nil
	}
	outcome := run_retry(0, backoff.Zero(), 3, operation)
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
	operation := func() (value int, err error) {
		calls++
		return 0, backoff.Permanent(cause)
	}
	outcome := run_retry(0, backoff.Zero(), 5, operation)
	if !errors.Is(outcome.Error, backoff.Error_Permanent) {
		t.Fatalf("error = %v, want a permanent error", outcome.Error)
	}
	if !errors.Is(outcome.Error, cause) {
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
	operation := func() (value int, err error) {
		calls++
		return 0, transient
	}
	outcome := run_retry(0, backoff.Constant(time.MILLISECOND), 3, operation)
	if !errors.Is(outcome.Error, backoff.Error_Exhausted) {
		t.Fatalf("error = %v, want exhausted", outcome.Error)
	}
	if !errors.Is(outcome.Error, transient) {
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
	operation := func() (value int, err error) {
		calls++
		if calls < 3 {
			return 0, errors.New("transient")
		}
		return 1, nil
	}
	outcome := run_retry(0, backoff.Constant(interval), 5, operation)
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
	operation := func() (value int, err error) {
		calls++
		if calls == 1 {
			return 0, backoff.Retry_After(override, errors.New("slow down"))
		}
		return 7, nil
	}
	// The policy delay is a millisecond, but Retry_After overrides the single wait.
	outcome := run_retry(0, backoff.Constant(time.MILLISECOND), 5, operation)
	if outcome.Error != nil {
		t.Fatalf("error = %v, want nil", outcome.Error)
	}
	if outcome.Elapsed != override {
		t.Fatalf("elapsed = %d, want %d", outcome.Elapsed, override)
	}
}
