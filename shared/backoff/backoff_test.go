package backoff_test

import (
	"testing"

	"local/james-orcales/shared/backoff"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/sim/nbio"
	"local/james-orcales/shared/sim/time"
)

// TestMain makes public operations prove their invariant paths.
func TestMain(m *testing.M) {
	aver.Run_Test_Main(m)
}

// Retry_Result captures what a driven Retry delivered plus the virtual time it took.
type Retry_Result struct {
	// Value is the result the operation ultimately returned.
	Value backoff.Result
	// Error is the error Retry delivered, or nil on success.
	Error error
	// Elapsed is virtual time retries consumed on timeline clock.
	Elapsed time.Duration
}

// Every other simulator resource is one slot: retry touches no endpoint and reads one clock.
const RETRY_SLOT_CAPACITY = 1

// Simulated loop whose timeline hold timer_count timers at once, on a grain of resolution.
// Test-side slices allocate; the constructor under test still must not.
func retry_loop(
	resolution time.Duration, timer_count int,
) (loop nbio.Timeline, driver nbio.Driver, host time.Clock) {
	state := &nbio.Sim{}
	surface, driver := nbio.New_Simulated_IO(state, 0, resolution, nbio.Sim_Memory{
		Nodes: []nbio.Sim_Node{{
			Name:     make([]byte, nbio.SIM_PATH_COMPONENT_BYTES_MAXIMUM),
			Contents: make([]byte, nbio.SIM_FILE_BYTES_MAXIMUM),
		}},
		Descriptors: []nbio.Sim_Descriptor{{
			Address_IP: make([]byte, nbio.IPV6_ADDRESS_BYTES),
			Peer_IP:    make([]byte, nbio.IPV6_ADDRESS_BYTES),
		}},
		Operations: []nbio.Sim_Operation{{
			Address_IP: make([]byte, nbio.IPV6_ADDRESS_BYTES),
		}},
		Queue:  make([]*nbio.Completion, timer_count),
		Events: make([]nbio.Virtual_Event, RETRY_SLOT_CAPACITY),
		Clocks: make([]nbio.Sim_Clock, RETRY_SLOT_CAPACITY),
	})
	return surface.Timeline, driver, nbio.Sim_Clock_To_Clock(&state.Clocks[0])
}

// Drives Retry on a fresh seeded sim loop, pumping until it finishes, and reports the
// delivered outcome with the virtual time the waits consumed.
func run_retry(
	policy backoff.Policy, tries_count backoff.Try_Count,
	operation backoff.Operation,
) (outcome Retry_Result) {
	loop, driver, host := retry_loop(time.MILLISECOND, int(tries_count))
	started := time.Clock_Now_Monotonic(host)
	retry_state := backoff.Retry_State{Input: backoff.Retry_Input{
		Timer: loop, Policy: policy, Tries_Max: tries_count, Clock: host,
	}}
	backoff.Retry(&retry_state, operation)
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
		nbio.Driver_Run_Until(
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
