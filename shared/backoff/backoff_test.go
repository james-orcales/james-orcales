package backoff_test

import (
	"testing"

	"local/james-orcales/shared/backoff"
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/simulation/time"
)

// TestMain makes public operations prove their invariant paths.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}

// Retry_Result captures what a driven Retry delivered plus the virtual time it took.
type Retry_Result struct {
	// Value is the result the operation ultimately returned.
	Value int
	// Error is the error Retry delivered, or nil on success.
	Error error
	// Elapsed is virtual time retries consumed on timeline clock.
	Elapsed time.Duration
}

// Drives Retry on a fresh seeded sim loop, pumping until it finishes, and reports the
// delivered outcome with the virtual time the waits consumed.
func run_retry(
	policy backoff.Policy, tries_count backoff.Try_Count,
	operation backoff.Operation[int],
) (outcome Retry_Result) {
	var timeline time.Virtual_Timeline
	queue := make([]*time.Completion, int(tries_count))
	events := make([]time.Virtual_Event, 1)
	loop, driver, clock := time.New_Virtual_Timeline(
		&timeline,
		time.Virtual_Clock{Resolution: time.MILLISECOND},
		time.Virtual_Timeline_Memory{Queue: queue, Events: events},
	)
	started := time.Clock_Now_Monotonic(clock)
	retry_state := backoff.Retry_State[int]{Input: backoff.Retry_Input{
		Timer: loop, Policy: policy, Tries_Max: tries_count, Clock: clock,
	}}
	backoff.Retry[int](&retry_state, operation)
	attempt_max := backoff.Attempt_Count(tries_count)
	for cycle_count := backoff.Attempt_Count(0); cycle_count < attempt_max; cycle_count++ {
		for rearm_count := backoff.Try_Count(0); rearm_count < tries_count; rearm_count++ {
			if !bool(backoff.Retry_Rearm(&retry_state)) {
				break
			}
		}
		if bool(backoff.Retry_Stopped(&retry_state)) {
			break
		}
		time.Driver_Run_Until(
			driver, time.Duration(tries_count)*time.MINUTE,
			func() (finished bool) {
				return bool(backoff.Retry_Work_Queued(&retry_state))
			},
		)
	}
	_, outcome.Value, outcome.Error = backoff.Retry_Status(&retry_state)
	// Completion remembers its exact deadline after driver advances past delivery.
	outcome.Elapsed = time.Duration(retry_state.Completion.Ready_At - started)
	return outcome
}
