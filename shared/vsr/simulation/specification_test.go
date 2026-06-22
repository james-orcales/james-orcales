package simulation_test

import (
	"fmt"
	"testing"
)

// Test_Cluster_Single_Primary drives many seeds in parallel, asserting no view ever has two acting
// primaries. Seed 1 is the regression guard for the recovery-quiescence bug: before the fix it
// drove the cluster into an agreement violation (op 2 committed as two different commands).
func Test_Cluster_Single_Primary(t *testing.T) {
	t.Parallel()
	for seed := int64(0); seed < 40; seed++ {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			t.Parallel()
			run_simulation(t, seed)
		})
	}
}

// Test_Cluster_Agreement drives many seeds in parallel, asserting committed entries never diverge.
func Test_Cluster_Agreement(t *testing.T) {
	t.Parallel()
	seeds := []int64{}
	for seed := int64(40); seed < 80; seed++ {
		seeds = append(seeds, seed)
	}
	// Permanent regression seed for restore_checkpoint leaving a stale Op, which made a
	// state-transfer splice overwrite a committed op — two replicas disagreeing on a committed
	// op (simulator_bugs.md, Bug 15). A wider 400..1399 scan found it; this seed pins it.
	seeds = append(seeds, 1266)
	// Regression seed: a view-changing replica answered a catch-up Get_State with its stale
	// uncommitted suffix (§5.2), so the requester kept op 109 = client-1002-req-49 past the
	// view change that reused it for reconfigure-2, then committed it on a Commit (Bug 22).
	// Reverting the view-change gate in replica_receive_get_state makes this seed fork; the fix
	// makes it pass.
	seeds = append(seeds, 3470)
	for _, seed := range seeds {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			t.Parallel()
			run_simulation(t, seed)
		})
	}
}

// Test_Cluster_Liveness drives many seeds in parallel and asserts every phase is exercised to
// completion: commits advance, primaries fail over, and crashed replicas recover and rejoin. Each
// seed writes its own slot, and the totals are checked in a cleanup once the subtests finish.
func Test_Cluster_Liveness(t *testing.T) {
	t.Parallel()
	const first = 80
	const limit = 120
	results := make([]simulation_result, limit-first)
	for seed := int64(first); seed < limit; seed++ {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			t.Parallel()
			results[seed-first] = run_simulation(t, seed)
		})
	}
	t.Cleanup(func() {
		total := simulation_result{}
		for index := range results {
			total.Commits += results[index].Commits
			total.View_Changes_Completed += results[index].View_Changes_Completed
			total.Recoveries_Completed += results[index].Recoveries_Completed
			total.State_Transfers += results[index].State_Transfers
		}
		if total.Commits == 0 {
			t.Error("expected commits to make progress across seeds")
		}
		if total.View_Changes_Completed == 0 {
			t.Error("expected at least one view change to complete across seeds")
		}
		if total.Recoveries_Completed == 0 {
			t.Error("expected at least one recovery to complete across seeds")
		}
		if total.State_Transfers == 0 {
			t.Error("expected at least one state transfer across seeds")
		}
	})
}

// Test_Cluster_Exactly_Once drives many seeds in parallel, asserting after every delivery that no
// (client, request-number) pair ever maps to two different results and no op executes twice on a
// replica — the exactly-once guarantee under retries, crashes, and view changes. Those assertions
// live inside run_simulation and fail fast on any seed; here the cleanup confirms the sweep
// actually exercised the path, so a silently inert run cannot pass. The total is checked rather
// than per-seed because an adversarial schedule can legitimately stall one 3-node seed.
func Test_Cluster_Exactly_Once(t *testing.T) {
	t.Parallel()
	const first = 120
	const limit = 160
	results := make([]simulation_result, limit-first)
	for seed := int64(first); seed < limit; seed++ {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			t.Parallel()
			results[seed-first] = run_simulation(t, seed)
		})
	}
	t.Cleanup(func() {
		replies := 0
		batches := 0
		reads := 0
		for index := range results {
			replies += results[index].Replies
			batches += results[index].Batches_Flushed
			reads += results[index].Reads_Committed
		}
		if replies == 0 {
			t.Error("expected the cluster to reply to clients across seeds")
		}
		if batches == 0 {
			t.Error("expected the primary to flush a " +
				"multi-request batch across seeds (§6.2)")
		}
		if reads == 0 {
			t.Error("expected read-only commands to commit " +
				"through consensus across seeds (§6.3)")
		}
	})
}

// Test_Cluster_Linearizability drives many seeds in parallel, asserting every result a client
// receives equals a reference state machine applied to the committed log in op order, so the
// cluster is indistinguishable from a single correct machine. The match is checked per reply
// inside run_simulation; the cleanup confirms clients did receive results across the sweep.
func Test_Cluster_Linearizability(t *testing.T) {
	t.Parallel()
	const first = 160
	const limit = 200
	results := make([]simulation_result, limit-first)
	for seed := int64(first); seed < limit; seed++ {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			t.Parallel()
			results[seed-first] = run_simulation(t, seed)
		})
	}
	t.Cleanup(func() {
		client_results := 0
		for index := range results {
			client_results += results[index].Client_Results
		}
		if client_results == 0 {
			t.Error("expected clients to receive results across seeds")
		}
	})
}

// Test_Cluster_Clock_Skew drives many seeds with per-replica clock skew and drift, asserting every
// safety oracle inside run_simulation still holds when no two nodes share a clock — the proof that
// VSR safety rests on consensus, not on time (§4.4 prediction and the timeout-driven view
// change both run against divergent clocks here). Clocks bend only in this sweep; the others keep
// identical clocks so their regression seeds reproduce exactly.
func Test_Cluster_Clock_Skew(t *testing.T) {
	t.Parallel()
	seeds := []int64{}
	for seed := int64(400); seed < 440; seed++ {
		seeds = append(seeds, seed)
	}
	// Permanent regression seed for the deferred view-change fetch accepting a stale,
	// non-attaching New_State under clock skew, which left a checkpoint-to-log gap the commit
	// walk indexed into (simulator_bugs.md, Bug 16). A wider 400..1399 skew-on scan found it.
	seeds = append(seeds, 1378)
	// Permanent regression seeds for the skew-on liveness bugs the PRNG sweep surfaced
	// (simulator_bugs.md, Bug 22): view-thrash from arming a delivery timer off the true clock
	// while ticks read the perceived one (281); a stuck cluster that could not retry into the
	// next view before the tail ended (639, 4171); a recovery that counted a replaced replica's
	// stale response (997); and a reconfiguration grow that built a duplicate config (4405).
	seeds = append(seeds, 281, 639, 4171, 997, 4405)
	for _, seed := range seeds {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			t.Parallel()
			run_simulation_with(t, seed, true)
		})
	}
}

// Test_Cluster_Reconfiguration drives many seeds in parallel, asserting the §7 reconfiguration runs
// end to end: the safety checks (epoch-aware single-primary, agreement across the epoch boundary,
// exactly-once, linearizability) hold after every delivery inside run_simulation, and the cleanup
// confirms reconfigurations actually completed and the epoch advanced across the sweep — so a run
// that never reconfigured cannot pass.
func Test_Cluster_Reconfiguration(t *testing.T) {
	t.Parallel()
	seeds := []int64{}
	for seed := int64(200); seed < 240; seed++ {
		seeds = append(seeds, seed)
	}
	// Permanent regression seeds for the reconfiguration bugs the simulator found across the
	// epoch boundary (simulator_bugs.md, Bug 9): exactly-once double-apply and commit-past-op.
	seeds = append(seeds, 1045, 1166, 1841, 6605, 7683, 7642)
	// Permanent regression seeds for the §7 deposed-primary / new-epoch-primary fork family the
	// PRNG sweep surfaced (simulator_bugs.md, Bug 22): a stale old-epoch member bare-committing
	// a divergent tail (2268, 460); a committed op lost when a recovering replica's op=0
	// redirect made the new-epoch primary truncate it (3678); and an adopted-not-executed
	// client-table record kept over the checkpoint's executed one, re-committing it (680).
	seeds = append(seeds, 2268, 460, 3678, 680)
	results := make([]simulation_result, len(seeds))
	for index, seed := range seeds {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			t.Parallel()
			results[index] = run_simulation(t, seed)
		})
	}
	t.Cleanup(func() {
		completed := 0
		epoch_max := 0
		for index := range results {
			completed += results[index].Reconfigurations_Completed
			if int(results[index].Epoch_Max) > epoch_max {
				epoch_max = int(results[index].Epoch_Max)
			}
		}
		if completed == 0 {
			t.Error("expected reconfigurations to complete across seeds")
		}
		if epoch_max == 0 {
			t.Error("expected the epoch to advance past 0 across seeds")
		}
	})
}
