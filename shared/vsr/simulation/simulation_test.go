package simulation_test

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"testing"

	invariant "github.com/james-orcales/james-orcales/shared/invariant/default"
	"github.com/james-orcales/james-orcales/shared/jlog"
	"github.com/james-orcales/james-orcales/shared/prng"
	"github.com/james-orcales/james-orcales/shared/time"
	"github.com/james-orcales/james-orcales/shared/vsr"
)

// TestMain runs the suite through the invariant harness so that, under a plain `go test`, every
// Always reached must hold and every Sometimes must be observed both true and false — a coverage
// gap or a violated assertion fails the run.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}

// How many virtual ticks one simulated run spans.
const sim_total_ticks = 1500

// How many virtual ticks the fault-free tail runs before declaring the cluster wedged. Generous: a
// view change, a recovery, and a few client retries must all complete once faults cease.
const sim_tail_ticks = 800

// The superset of replicas pre-allocated for a run (§7): membership grows, shrinks, and swaps
// within it as reconfigurations move the group, so the array must hold the largest group plus the
// fresh nodes a swap or grow brings in. Indices beyond the initial group start dormant and join
// only when a reconfiguration adds them.
const sim_superset = 7

// How often the administrator issues a reconfiguration, in ticks. Spaced so a reconfiguration has
// room to complete — the new group catching up, the old group shutting down — before the next, yet
// frequent enough that several epochs pass in a run.
const sim_reconfigure_every = 320

// How many clients issue requests against the cluster.
const sim_client_count = 3

// How many ticks a client waits for a reply before re-sending the same request to every replica.
const sim_client_timeout = 60

// The percent chance, each tick, that an idle client forgets its request-number knowledge and
// recovers it from the cluster's cached reply — modeling §4.5 client recovery.
const sim_client_recover_percent = 2

// State_machine_apply_input is the command and its predetermined prediction the application folds.
type state_machine_apply_input struct {
	// Command is the opaque op payload.
	Command []byte
	// Prediction is the value the primary predetermined for the op (§4.4); empty for a
	// deterministic op.
	Prediction []byte
}

// Applies one command against its predetermined prediction (§4.4): an 8-byte hash folding both the
// command bytes and the prediction bytes. The result depends on BOTH, so a divergent prediction
// would produce a divergent result — which is exactly what the prediction protocol must prevent
// by stamping the value once on the primary and copying it verbatim to every replica. The
// application is otherwise a pure function of (command, prediction), deliberately stateless,
// because Stage 1 has no state-machine restore: a replica that adopts a log via view change,
// recovery, or state transfer commits those ops without executing them, so a running accumulator
// would diverge across replicas. A (command, prediction)-pure result keeps every replica's reply to
// a given request identical, the exactly-once property under test; a later stage with
// Snapshot/Restore can make it stateful.
func state_machine_apply(input *state_machine_apply_input) (result []byte) {
	hash := uint64(0xcbf29ce484222325)
	for _, b := range input.Command {
		hash = (hash ^ uint64(b)) * 0x100000001b3
	}
	for _, b := range input.Prediction {
		hash = (hash ^ uint64(b)) * 0x100000001b3
	}
	result = make([]byte, 8)
	for index := 0; index < 8; index++ {
		result[index] = byte(hash >> (56 - 8*index))
	}
	return result
}

// A process outside the cluster issuing requests under exactly-once semantics. It holds one
// outstanding request at a time, re-sending the identical request on timeout, and occasionally
// forgets its request-number to model a client that restarted and must relearn it (§4.5).
type client struct {
	// Identifier is this client's cluster-wide identity, distinct from any Replica_Identifier.
	Identifier vsr.Client_Identifier
	// Request_Number is the highest request-number this client has issued.
	Request_Number vsr.Request_Number
	// Unanswered reports whether a request is awaiting a reply.
	Unanswered bool
	// Request_Command is the outstanding request's command, re-sent verbatim on a retry.
	Request_Command []byte
	// Request_Moment is when the outstanding request was first issued, for the retry timer.
	Request_Moment time.Moment
	// Recovery reports that the client has forgotten its request-number and must learn it from
	// a cached Reply before issuing anything new (§4.5).
	Recovery bool
}

// The tick interval between crash-restarts.
const sim_crash_every = 220

// The tick interval between partition onsets.
const sim_isolate_every = 180

// How many millisecond grains a partition lasts.
const sim_isolate_window = 150

// The tick interval between transient clock-fault onsets, under the clock-skew sweep.
const sim_clock_fault_every = 200

// How many millisecond grains a transient clock fault lasts before it heals.
const sim_clock_fault_window = 120

// The largest message delay, in millisecond grains.
const sim_delay_max = 3

// The percent chance a message is dropped by the network.
const sim_drop_percent = 8

// The percent chance a message is duplicated by the network.
const sim_duplicate_percent = 4

// Simulation_result aggregates the liveness outcomes of one run.
type simulation_result struct {
	View_Changes_Completed int
	Recoveries_Completed   int
	Commits                int
	State_Transfers        int
	View_Max               vsr.View
	// Replies counts the client-addressed Replies the cluster produced.
	Replies int
	// Client_Results counts the results clients accepted, checked against the reference.
	Client_Results int
	// Client_Recoveries counts the times a client relearned its request-number from a cache.
	Client_Recoveries int
	// Cached_Replies counts the duplicate-request cached replies the primary re-sent.
	Cached_Replies int
	// Predictions_Stamped counts the committed ops carrying a non-empty prediction, witnessing
	// that the §4.4 prediction path was exercised rather than idling on empty values.
	Predictions_Stamped int
	// Pre_Step_Rounds counts the Predict_Request broadcasts the primary opened, witnessing that
	// the pre-step variant ran on the seeds configured for it.
	Pre_Step_Rounds int
	// Checkpoint_Gaps counts the New_State responses that shipped a checkpoint because the
	// requester needed a garbage-collected prefix (§5.2), witnessing the gap path ran.
	Checkpoint_Gaps int
	// Warm_Recoveries_Started counts the crashes that recovered warm (checkpoint kept on disk).
	Warm_Recoveries_Started int
	// View_Change_Fetches counts the deliveries that found a new primary deferring its
	// view-change install while it fetches the selected log (§5.3), witnessing the fetch path
	// ran rather than every winner being reconstructable from the suffix alone.
	View_Change_Fetches int
	// Do_View_Change_Suffixes counts the Do_View_Change messages whose op exceeded their suffix
	// length — a genuinely bounded report that dropped entries the reporter still held (§5.3),
	// witnessing the suffix bound bit rather than every log fitting within it.
	Do_View_Change_Suffixes int
	// Reconfigurations_Completed counts the reconfigurations that ran to completion in a run:
	// the epoch advanced and a member of the new group reached Status_Normal in it (§7),
	// witnessing the whole handoff actually executed end to end.
	Reconfigurations_Completed int
	// Epoch_Max is the highest epoch any replica reached, witnessing reconfigurations advanced
	// the epoch beyond 0.
	Epoch_Max vsr.Epoch
	// Epoch_Checkpoint_Catch_Ups counts the brand-new nodes that restored a checkpoint to catch
	// up to the epoch start (§7.1.1), the empty-log path that fetches a GC'd prefix as a
	// checkpoint.
	Epoch_Checkpoint_Catch_Ups int
	// Reconfigure_Over_View_Change counts the ticks where a replica was transitioning between
	// epochs while another was changing views — the §7.2 overlap of a reconfiguration and a
	// view change.
	Reconfigure_Over_View_Change int
	// Reconfigure_Over_Recovery counts the ticks where a replica was transitioning between
	// epochs while another was recovering — the §7.2 overlap of a reconfiguration and recovery.
	Reconfigure_Over_Recovery int
	// Batches_Flushed counts the Prepares carrying more than one entry, witnessing that a busy
	// primary actually batched several requests into one round (§6.2) rather than every Prepare
	// being a batch of one.
	Batches_Flushed int
	// Reads_Committed counts the read-only commands that committed, witnessing that reads
	// actually ran through consensus (§6.3, TigerBeetle's reads-through-consensus) rather than
	// the workload being writes only.
	Reads_Committed int
	// Tail_Drained counts runs whose fault-free tail had open requests to drain, witnessing the
	// liveness check exercised real work, not an already-quiescent cluster.
	Tail_Drained int
}

// Scheduled_message is a message scheduled for delivery at a virtual moment.
type scheduled_message struct {
	Message    vsr.Message
	Deliver_At time.Moment
}

// Simulator holds one run's whole mutable world: the cluster, the in-flight network, the partition
// schedule, the fault generator, the clients, and the verification models. Its free functions
// evolve it; run_simulation owns its life.
type simulator struct {
	T         *testing.T
	Seed      int64
	Generator prng.Generator
	Clock     time.Clock
	// Replica_Clocks is one virtual clock per replica index, ticked in lockstep with Clock
	// but read independently so each replica perceives its own time — no two nodes share a
	// clock. Skew off makes each read like Clock; skew and drift bend them apart.
	Replica_Clocks []time.Clock
	// Clock_Generator draws all per-replica clock randomness (offsets, drifts, fault onsets)
	// from a stream SEPARATE from Generator, so adding clocks does not perturb the fault
	// schedule the documented regression seeds reproduce.
	Clock_Generator prng.Generator
	// Clock_Skew enables per-replica offset, drift, and injected clock faults. Off (the
	// default run) keeps every clock identical for exact traces; the sweep turns it on.
	Clock_Skew bool
	// Clock_Fault_Until and Clock_Fault_Offset model a transient clock fault (an NTP jump or
	// a bad oscillator): while the true clock is before Clock_Fault_Until[i], replica i's
	// perceived time is shifted by Clock_Fault_Offset[i]; past it the fault heals.
	Clock_Fault_Until  []time.Moment
	Clock_Fault_Offset []time.Duration
	// Faultless, set for the fault-free tail, stops message drops and new client requests so
	// the cluster can drain to a converged state for the liveness check.
	Faultless       bool
	Cluster_Count   int
	Replicas        []vsr.Replica
	Previous_Status []vsr.Status
	Network         []scheduled_message
	Isolated_Until  []time.Moment
	Next_Nonce      vsr.Nonce
	Next_Command    int
	Result          simulation_result
	Clients         []client
	// Executed records, per replica, the set of commands it has applied. Commands are globally
	// unique per request, so a command applied twice on one replica is a double-execution —
	// the reliable Stage-1 exactly-once check, since op-numbers are not visible inside Execute
	// and a replica that adopts a log via view change or recovery commits ops without
	// executing.
	Executed []map[string]bool
	// Executed_Result records, per replica, the result each command applied to. Because the
	// primary stamps a non-deterministic op's prediction once and every replica copies it
	// verbatim (§4.4), every replica that executes a given command must produce byte-identical
	// results; a divergence here would mean a replica recomputed the value, not reused it.
	Executed_Result []map[string][]byte
	// Accumulator is each replica's stateful application state: a rolling 64-bit hash folded
	// over every (command, prediction) it has executed in op order. Snapshot returns it and
	// Restore sets it, so a replica that adopts a checkpoint must restore-then-replay to reach
	// the same value — making snapshot/restore correctness observable. A replica that adopted
	// a committed prefix without executing or restoring it would diverge here from the
	// reference.
	Accumulator []uint64
	// Reference is the linearizability oracle: a single state machine fed the agreed committed
	// log in op order, holding the canonical result for each committed op.
	Reference reference_model
	// Outcomes is the exactly-once oracle: every (client, request-number) maps to one result,
	// fed from the replies clients receive and the reference's canonical results.
	Outcomes map[outcome_key][]byte
	// Active records, per replica index, whether it currently participates: a member of the
	// current epoch's group that has not shut down. A dormant pre-allocated node (not yet
	// added) and a replaced node that has shut down are inactive, excluded from delivery
	// (except a Start_Epoch, which activates a dormant node), ticking, the primary computation,
	// and the crash injector. It tracks the protocol's own membership view rather than
	// dictating it.
	Active []bool
	// Epoch is the epoch the simulator believes is current — the highest any active replica
	// serves normally — and Configuration is that epoch's membership. The admin reconfigures
	// relative to these, and the primary/agreement computations scope to them.
	Epoch         vsr.Epoch
	Configuration []vsr.Replica_Identifier
	// Admin is the administrator client that issues reconfigurations, with its own identity and
	// request-number so exactly-once applies to it like any client (§7.1).
	Admin client
	// Next_Fresh is the next pre-allocated index a swap brings in as a brand-new node, walking
	// up from the initial group size so each swap introduces a node that has never served.
	Next_Fresh int
	// Reconfigure_After is the earliest tick the next reconfiguration may issue. The injector
	// fires once the group is settled at or after it, then pushes it forward — so a
	// reconfiguration is not skipped merely because the group was mid-crash at an exact
	// interval boundary, which under constant faults would otherwise leave a run with no
	// reconfiguration at all.
	Reconfigure_After int
	// Trace is a synchronous jlog logger emitting one structured JSON line per simulation event
	// (delivery, fault, commit, the ending violation) into Trace_Sink, read offline with jq.
	// The zero Logger is a disabled no-op, so when VSR_TRACE does not select this seed every
	// trace call costs nothing and the sweep is unaffected.
	Trace jlog.Logger
	// Trace_Sink buffers the trace lines; nil when tracing off, flushed to a file at run end.
	Trace_Sink *bytes.Buffer
}

// How many committed ops pass between checkpoints in the simulated cluster — small, so compaction
// and the §5.2 gap path fire constantly across a run.
const sim_checkpoint_interval = 4

// How many ops of suffix each replica retains past a checkpoint; small, so a behind replica often
// needs a prefix a peer has already garbage-collected, exercising the gap response.
const sim_log_retain = 4

// The most ops a busy primary batches into one Prepare (§6.2). Above one, so a primary loaded by
// the clients' concurrent requests collects several into a batch, exercising the multi-entry
// Prepare and the per-op acknowledgement of a batch.
const sim_batch_max = 3

// Identifies one client request for the exactly-once map.
type outcome_key struct {
	Client  vsr.Client_Identifier
	Request vsr.Request_Number
}

// The single correct machine the cluster must be indistinguishable from: it applies the committed
// log in op order and remembers each op's canonical result and the request it serves, so a reply
// can be checked against the result that op should have produced.
type reference_model struct {
	// Applied is how many committed ops the reference has folded in; it only grows.
	Applied int
	// Result_Of_Op holds, per 1-based op, the canonical result the cluster must return for it.
	Result_Of_Op map[vsr.Op][]byte
	// Request_Of_Op holds, per 1-based op, the (client, request-number) the op serves.
	Request_Of_Op map[vsr.Op]outcome_key
	// Accumulator is the canonical rolling hash after the committed prefix the reference has
	// folded in; it is what a replica's live accumulator at commit C must equal.
	Accumulator uint64
	// Accumulator_Of_Commit holds the canonical accumulator after the first N committed ops, so
	// a replica reporting commit C can be checked against the reference value at exactly C.
	Accumulator_Of_Commit map[vsr.Op]uint64
}

// Accumulator_fold_input is the prior accumulator and the executed (command, prediction) to fold.
type accumulator_fold_input struct {
	// Previous is the accumulator before this op.
	Previous uint64
	// Command is the executed op's payload.
	Command []byte
	// Prediction is the executed op's predetermined value (§4.4); empty for a deterministic
	// op.
	Prediction []byte
}

// Folds one executed (command, prediction) into a rolling 64-bit FNV-1a hash — the stateful
// application's transition. It depends on the prior accumulator, so a replica that skips or
// re-orders an op, or executes one against a divergent prediction, lands on a different value.
func accumulator_fold(input *accumulator_fold_input) (next uint64) {
	next = input.Previous
	for _, b := range input.Command {
		next = (next ^ uint64(b)) * 0x100000001b3
	}
	for _, b := range input.Prediction {
		next = (next ^ uint64(b)) * 0x100000001b3
	}
	return next
}

// Drives a cluster through one seed's worth of client load, crashes, partitions, and message
// loss/reordering/duplication on a virtual clock, asserting the cluster safety invariants after
// every delivery and recording the liveness coverage axes. A safety violation fails the test naming
// the seed; the framework's Always/Sometimes do the rest.
func run_simulation(t *testing.T, seed int64) (result simulation_result) {
	return run_simulation_with(t, seed, false)
}

// Drives one seed, optionally with per-replica clock skew, drift, and injected clock faults. The
// default run_simulation leaves every clock identical so its traces — and the regression seeds
// pinned to them — are unchanged; Test_Cluster_Clock_Skew turns skew on to prove the safety
// oracles still hold when no two nodes share a clock.
func run_simulation_with(t *testing.T, seed int64, clock_skew bool) (result simulation_result) {
	t.Helper()
	state := new_simulator(t, seed, clock_skew)
	// Flush even when a safety Fatalf unwinds this goroutine, so the trace ends at the fork.
	defer simulator_flush_trace(state)
	simulator_run_main(state)
	// Fault-free tail: heal every fault, then drain. After faults cease the cluster must
	// converge (every open request commits, the voting set settles to Normal) or it has
	// wedged — a liveness violation the tail reports naming the seed.
	for index := range state.Isolated_Until {
		state.Isolated_Until[index] = 0
		state.Clock_Fault_Until[index] = 0
	}
	had_open_requests := false
	for index := range state.Clients {
		if state.Clients[index].Unanswered {
			had_open_requests = true
		}
	}
	if !simulator_run_tail(state) {
		t.Fatalf("seed %d: cluster did not converge in the fault-free tail (wedged)", seed)
	}
	if had_open_requests {
		state.Result.Tail_Drained++
	}
	return state.Result
}

// Builds one run's world: the superset of replicas, the clients, the verification models, and —
// when clock_skew is set — each replica's own skewing clock.
func new_simulator(t *testing.T, seed int64, clock_skew bool) (state *simulator) {
	cluster_count := 3
	if seed%2 == 0 {
		cluster_count = 5 // Exercise both quorum sizes.
	}
	clock := time.Virtual_Clock_To_Clock(time.Virtual_Clock{Resolution: time.Millisecond})
	state = &simulator{
		T:              t,
		Seed:           seed,
		Generator:      prng.New(uint64(seed)),
		Clock:          clock,
		Replica_Clocks: make([]time.Clock, sim_superset),
		// Clock stream seeded apart from Generator so per-replica clocks draw without
		// shifting the main fault schedule the regression seeds reproduce.
		Clock_Generator:    prng.New(uint64(seed) ^ 0xc10cc10cc10cc10c),
		Clock_Skew:         clock_skew,
		Clock_Fault_Until:  make([]time.Moment, sim_superset),
		Clock_Fault_Offset: make([]time.Duration, sim_superset),
		Cluster_Count:      cluster_count,
		Replicas:           make([]vsr.Replica, sim_superset),
		Previous_Status:    make([]vsr.Status, sim_superset),
		Isolated_Until:     make([]time.Moment, sim_superset),
		Active:             make([]bool, sim_superset),
		Next_Nonce:         vsr.Nonce(1),
		Executed:           make([]map[string]bool, sim_superset),
		Executed_Result:    make([]map[string][]byte, sim_superset),
		Accumulator:        make([]uint64, sim_superset),
		Reference: reference_model{
			Result_Of_Op:          map[vsr.Op][]byte{},
			Request_Of_Op:         map[vsr.Op]outcome_key{},
			Accumulator_Of_Commit: map[vsr.Op]uint64{0: 0},
		},
		Outcomes:      map[outcome_key][]byte{},
		Epoch:         0,
		Configuration: sorted_configuration(cluster_count),
		// The administrator's identity sits above the client identifiers, which sit above
		// the replica identifiers, so none aliases another.
		Admin:             client{Identifier: vsr.Client_Identifier(2000)},
		Next_Fresh:        cluster_count,
		Reconfigure_After: sim_reconfigure_every,
	}
	simulator_allocate(state, cluster_count)
	if trace_enabled(seed) {
		state.Trace_Sink = &bytes.Buffer{}
		state.Trace = jlog.New(jlog.New_Input{
			Writer: state.Trace_Sink,
			Floor:  jlog.Level_Trace,
		})
		jlog.Logger_Info(state.Trace, "seed",
			jlog.Int64("seed", seed),
			jlog.Boolean("skew", clock_skew),
			jlog.Integer("cluster", cluster_count))
	}
	return state
}

// Drives the faulty phase: client load, crashes, partitions, reconfigurations, and message
// loss/reorder/duplication on a virtual clock, asserting safety after every delivery and recording
// the coverage axes.
func simulator_run_main(state *simulator) {
	for tick_index := 0; tick_index < sim_total_ticks; tick_index++ {
		state.Clock.Tick()
		// Advance every replica's own clock with the true clock; a drifting clock falls
		// behind or races ahead within the same wall-clock tick.
		for index := range state.Replica_Clocks {
			state.Replica_Clocks[index].Tick()
		}
		now := state.Clock.Now_Monotonic()
		simulator_inject_faults(state, tick_index, now)
		simulator_inject_reconfiguration(state, tick_index, now)
		simulator_tick_clients(state, now)
		simulator_tick_replicas(state, now)
		simulator_assert_safety(state)
		simulator_record_coverage(state)
		simulator_deliver(state, now)
	}
}

// Runs the fault-free tail: it stops new faults and new client requests, lets timers and delivery
// drain the cluster, and returns whether it converged within sim_tail_ticks. The caller heals
// partitions and clock faults first; a caller that leaves a fault in place (the negative test)
// watches it fail to converge, proving the liveness check is not vacuous.
func simulator_run_tail(state *simulator) (converged bool) {
	state.Faultless = true
	for tick_index := 0; tick_index < sim_tail_ticks; tick_index++ {
		state.Clock.Tick()
		for index := range state.Replica_Clocks {
			state.Replica_Clocks[index].Tick()
		}
		now := state.Clock.Now_Monotonic()
		simulator_tick_clients(state, now)
		simulator_tick_replicas(state, now)
		simulator_assert_safety(state)
		simulator_deliver(state, now)
		if simulator_converged(state) {
			return true
		}
	}
	return false
}

// Reports whether the cluster has drained: every client request answered, and every voting member
// of the current configuration active and back to Status_Normal (no one stuck in view change or
// recovery). Standbys are not required to have caught up, only the voting set that serves requests.
func simulator_converged(state *simulator) (converged bool) {
	for index := range state.Clients {
		if state.Clients[index].Unanswered {
			return false
		}
	}
	voting_count := int(active_count_for(len(state.Configuration)))
	for position_index := 0; position_index < voting_count; position_index++ {
		if position_index >= len(state.Configuration) {
			break
		}
		identifier := state.Configuration[position_index]
		if !state.Active[identifier] {
			return false
		}
		if state.Replicas[identifier].Status != vsr.Status_Normal {
			return false
		}
	}
	return true
}

// Builds the superset of replicas and the clients for a run. The initial group's members start
// active in it; the pre-allocated nodes beyond it start dormant with a self-including placeholder
// configuration (so the per-replica safety assert holds if they are ever inspected) and activate
// only on a Start_Epoch that adds them.
func simulator_allocate(state *simulator, cluster_count int) {
	initial := sorted_configuration(cluster_count)
	for index := range state.Replicas {
		state.Executed[index] = map[string]bool{}
		state.Executed_Result[index] = map[string][]byte{}
		jitter := time.Duration(50 + prng.Generator_Below(&state.Generator, 40))
		configuration := initial
		active := index < cluster_count
		if !active {
			configuration = dormant_configuration(index)
		}
		state.Active[index] = active
		state.Replica_Clocks[index] = simulator_replica_clock(state)
		state.Replicas[index] = vsr.New_Replica(&vsr.New_Replica_Input{
			Identifier:          vsr.Replica_Identifier(index),
			Configuration:       configuration,
			Active_Count:        active_count_for(len(configuration)),
			Heartbeat:           10 * time.Millisecond,
			Timeout:             jitter * time.Millisecond,
			State_Machine:       simulator_state_machine(state, index),
			Checkpoint_Interval: sim_checkpoint_interval,
			Log_Retain:          sim_log_retain,
			Batch_Max:           sim_batch_max,
			Now:                 state.Replica_Clocks[index].Now_Realtime(),
		})
		state.Previous_Status[index] = state.Replicas[index].Status
	}
	state.Clients = make([]client, sim_client_count)
	for index := range state.Clients {
		// Client identifiers start above any replica identifier so they never alias.
		state.Clients[index] = client{Identifier: vsr.Client_Identifier(1000 + index)}
	}
}

// Builds one replica's clock. Skew off makes it read like the true clock; skew on gives it a
// per-replica offset (Epoch) and a bounded linear drift, so no two replicas share time: the
// offset differentiates the §4.4 timestamps each would stamp and the drift desynchronizes their
// timeouts — what VSR safety must survive by leaning on consensus, not the clock.
func simulator_replica_clock(state *simulator) (clock time.Clock) {
	virtual := time.Virtual_Clock{Resolution: time.Millisecond}
	if state.Clock_Skew {
		virtual.Epoch = time.Moment(prng.Generator_Below(&state.Clock_Generator, 50)) *
			time.Moment(time.Millisecond)
		// Drift A ns/tick shifts the effective rate to Resolution-A; |A| far below
		// Resolution keeps the clock monotonic while still drifting up to ~2%.
		rate := time.Duration(prng.Generator_Below(&state.Clock_Generator, 40001) - 20000)
		virtual.Skew = time.Skew(time.Skew_Input{Kind: time.Skew_Kind_Linear, A: rate})
	}
	return time.Virtual_Clock_To_Clock(virtual)
}

// A dormant pre-allocated node's placeholder configuration: three members including itself,
// wrapping within the superset, so the per-replica membership and minimum-size safety asserts hold
// before the node ever joins a real group. A Start_Epoch replaces it with the real new
// configuration.
func dormant_configuration(index int) (configuration vsr.Configuration) {
	return vsr.Configuration{
		vsr.Replica_Identifier(index),
		vsr.Replica_Identifier((index + 1) % sim_superset),
		vsr.Replica_Identifier((index + 2) % sim_superset),
	}
}

// Builds the state machine for one replica: a STATEFUL hash application that folds each executed
// (command, prediction) into a rolling accumulator, records the command and its result, and exposes
// the accumulator through Snapshot/Restore. The accumulator makes snapshot/restore correctness
// observable end-to-end: a replica that adopts a checkpoint must restore-then-replay to reach the
// same value the reference computes for that commit, and one that adopts a committed prefix without
// executing or restoring would diverge. The closure captures the replica's index, so each replica
// folds its own copy of the same committed log. Predict reads the virtual clock — a value that
// WOULD diverge if each replica recomputed it at execution time, since they execute at different
// moments. The protocol must prevent that divergence by stamping the value once on the primary and
// copying it verbatim (§4.4). Even seeds run local-predict (Combine nil); odd seeds run the
// pre-step variant (Combine set), folding the backups' clock readings with the primary's
// deterministically so the stored value still reflects the cluster.
func simulator_state_machine(state *simulator, index int) (machine vsr.State_Machine) {
	machine = vsr.State_Machine{
		Execute: func(command []byte, prediction []byte) (result []byte) {
			simulator_observe_execution(state, index, command)
			if command_is_read(command) {
				// A read returns the current application state without mutating it.
				// It still ran through the log and committed like any op (§6.3
				// reads-through-consensus, TigerBeetle's design); the result
				// reflects the accumulator at this op, which is identical on every
				// replica that executed the same committed prefix.
				result = uint64_to_bytes(state.Accumulator[index])
				state.Executed[index][string(command)] = true
				state.Executed_Result[index][string(command)] = result
				return result
			}
			state.Accumulator[index] = accumulator_fold(&accumulator_fold_input{
				Previous:   state.Accumulator[index],
				Command:    command,
				Prediction: prediction,
			})
			result = state_machine_apply(&state_machine_apply_input{
				Command:    command,
				Prediction: prediction,
			})
			state.Executed[index][string(command)] = true
			state.Executed_Result[index][string(command)] = result
			return result
		},
		Predict: func(command []byte) (prediction []byte) {
			return moment_to_bytes(simulator_replica_now(state, index))
		},
		Snapshot: func() (snapshot []byte) {
			return uint64_to_bytes(state.Accumulator[index])
		},
		Restore: func(snapshot []byte) {
			state.Accumulator[index] = bytes_to_uint64(snapshot)
		},
	}
	if state.Seed%2 != 0 {
		machine.Combine = func(
			command []byte, responses [][]byte, own []byte,
		) (prediction []byte) {
			folded := bytes_to_uint64(own)
			for _, response := range responses {
				folded += bytes_to_uint64(response)
			}
			return uint64_to_bytes(folded)
		}
	}
	return machine
}

// Encodes a virtual moment as 8 big-endian bytes — the form a clock-derived prediction takes.
func moment_to_bytes(moment time.Moment) (encoded []byte) {
	return uint64_to_bytes(uint64(moment))
}

// Encodes a uint64 as 8 big-endian bytes.
func uint64_to_bytes(value uint64) (encoded []byte) {
	encoded = make([]byte, 8)
	for index := 0; index < 8; index++ {
		encoded[index] = byte(value >> (56 - 8*index))
	}
	return encoded
}

// Decodes 8 big-endian bytes back to a uint64; a shorter slice reads as its leading bytes.
func bytes_to_uint64(encoded []byte) (value uint64) {
	for _, b := range encoded {
		value = (value << 8) | uint64(b)
	}
	return value
}

// Runs once per tick: a periodic crash-restart of an active normal replica and a periodic
// partition onset. Both scope to the ACTIVE group — the current epoch's members that have not shut
// down — so a crash never targets a dormant pre-allocated node or a node already gone, and never
// drops the active group below the quorum it needs to keep making progress. Client request
// injection is no longer here — explicit clients drive it from simulator_tick_clients so
// exactly-once is exercised by real retries.
func simulator_inject_faults(state *simulator, tick_index int, now time.Moment) {
	if tick_index%sim_crash_every == 0 {
		simulator_inject_crash(state, now)
	}
	if tick_index%sim_isolate_every == 0 {
		simulator_inject_isolation(state, now)
	}
	if state.Clock_Skew {
		if tick_index%sim_clock_fault_every == 0 {
			simulator_inject_clock_fault(state, now)
		}
	}
}

// Applies a transient clock fault to one active replica: a forward step in its perceived time for a
// window, modeling an NTP correction or a bad oscillator. Perceived time jumps forward while the
// fault is open and snaps back when it heals — both directions a clock can move that safety must
// survive and liveness must recover from. The draw is from the separate clock stream, and the fault
// fires only under the clock-skew sweep.
func simulator_inject_clock_fault(state *simulator, now time.Moment) {
	active := simulator_active_indices(state)
	if len(active) == 0 {
		return
	}
	victim := active[prng.Generator_Below(&state.Clock_Generator, len(active))]
	jump := time.Duration(20+prng.Generator_Below(&state.Clock_Generator, 40)) *
		time.Duration(time.Millisecond)
	state.Clock_Fault_Offset[victim] = jump
	state.Clock_Fault_Until[victim] = now +
		time.Moment(sim_clock_fault_window)*time.Moment(time.Millisecond)
}

// Partitions one active replica for a window. A voting replica is partitioned only when the group
// is fully settled: a reconfiguration's stranded member plus a freshly partitioned one would leave
// two voting replicas behind, deadlocking recovery (beyond the fault model). A standby never
// threatens the quorum.
func simulator_inject_isolation(state *simulator, now time.Moment) {
	active := simulator_active_indices(state)
	if len(active) == 0 {
		return
	}
	victim := active[prng.Generator_Below(&state.Generator, len(active))]
	if !is_standby(&state.Replicas[victim]) {
		if !simulator_group_settled(state) {
			return
		}
	}
	window := time.Moment(sim_isolate_window) * time.Moment(time.Millisecond)
	state.Isolated_Until[victim] = now + window
	jlog.Logger_Info(state.Trace, "fault",
		jlog.Int64("t", now),
		jlog.String("kind", "isolate"),
		jlog.Integer("replica", victim),
		jlog.Int64("until", state.Isolated_Until[victim]))
}

// Crash-restarts one active normal replica, respecting the fault model so the cluster stays
// recoverable. Half the crashes are warm (checkpoint survived on disk) and half cold (everything
// lost), so both recovery paths run across the sweep.
func simulator_inject_crash(state *simulator, now time.Moment) {
	active := simulator_active_indices(state)
	if len(active) == 0 {
		return
	}
	victim := active[prng.Generator_Below(&state.Generator, len(active))]
	if state.Replicas[victim].Status != vsr.Status_Normal {
		return
	}
	// Respect the fault model. The voting set tolerates one fault, but a reconfiguration can
	// strand one, so crash a voting replica only when the group is fully settled
	// (every voting member Normal at the frontier). Crashing mid-handoff strands two at once;
	// they recover with no quorum to answer them and deadlock. A standby is non-voting, so
	// crashing one never threatens the quorum, and a standby recovering during a handoff still
	// witnesses the §7.2 reconfigure-over-recovery overlap.
	if !is_standby(&state.Replicas[victim]) {
		if !simulator_group_settled(state) {
			return
		}
	}
	warm := prng.Generator_Below(&state.Generator, 2) == 0
	if warm {
		state.Result.Warm_Recoveries_Started++
	}
	output := vsr.Replica_Recover(&vsr.Replica_Recover_Input{
		Replica:         &state.Replicas[victim],
		Nonce:           state.Next_Nonce,
		Keep_Checkpoint: warm,
		Now:             now,
	})
	state.Next_Nonce++
	// A crash loses the volatile state machine. A cold recovery starts the accumulator from
	// zero and re-executes the whole committed log, so clear the executed-set and accumulator.
	// A warm recovery's checkpoint survived: Replica_Recover already Restored the accumulator
	// to the checkpoint op, and only the un-checkpointed suffix replays — clearing the
	// executed-set keeps that replay from tripping the op-level double-execution check, which
	// the cluster-facing oracles do not need.
	state.Executed[victim] = map[string]bool{}
	state.Executed_Result[victim] = map[string][]byte{}
	if !warm {
		state.Accumulator[victim] = 0
	}
	jlog.Logger_Info(state.Trace, "fault",
		jlog.Int64("t", now),
		jlog.String("kind", "crash"),
		jlog.Integer("replica", victim),
		jlog.Boolean("warm", warm))
	simulator_send(state, output.Messages, now)
}

// Issues the administrator's reconfigurations (§7), cycling grow, shrink, and swap across a run so
// the group's size and membership both change. A reconfiguration is sent only when the current
// group is settled (every member normal in the current epoch) so it does not pile onto an
// in-progress handoff, and only when the chosen target group is valid (at least three members). It
// carries the admin's own monotonic request-number, so exactly-once applies to it too.
func simulator_inject_reconfiguration(state *simulator, tick_index int, now time.Moment) {
	if tick_index < state.Reconfigure_After {
		return
	}
	if !simulator_group_settled(state) {
		// Not settled yet; retry next tick rather than skipping this reconfiguration.
		return
	}
	target, ok := simulator_next_configuration(state)
	if !ok {
		// The chosen change is not possible now (no fresh node, or a size floor); advance
		// the admin's request count so the cycle moves to a change that is, and try again
		// soon.
		state.Admin.Request_Number++
		state.Reconfigure_After = tick_index + 1
		return
	}
	state.Admin.Request_Number++
	state.Reconfigure_After = tick_index + sim_reconfigure_every
	primary := simulator_believed_primary(state)
	jlog.Logger_Info(state.Trace, "fault",
		jlog.Int64("t", now),
		jlog.String("kind", "reconfigure"),
		jlog.Uint64("request", state.Admin.Request_Number),
		jlog.String("config", fmt.Sprintf("%v", target)),
		jlog.Uint8("active", active_count_for(len(target))))
	simulator_send(state, []vsr.Message{{
		Kind:           vsr.Message_Kind_Reconfiguration,
		From:           primary,
		To:             primary,
		Epoch:          state.Epoch,
		Client:         state.Admin.Identifier,
		Request_Number: state.Admin.Request_Number,
		Command: []byte(fmt.Sprintf(
			"reconfigure-%d", state.Admin.Request_Number)),
		New_Configuration: target,
		New_Active_Count:  active_count_for(len(target)),
	}}, now)
}

// Reports whether the current group is settled: every member of the current configuration is
// active, normal, and in the current epoch, with no replica still transitioning. Issuing a
// reconfiguration only when settled keeps one handoff from overlapping the next, which would tangle
// the membership the simulator tracks (the protocol tolerates overlap; the harness's bookkeeping is
// simpler this way).
func simulator_group_settled(state *simulator) (settled bool) {
	for index := range state.Replicas {
		if state.Replicas[index].Status == vsr.Status_Transition {
			return false
		}
	}
	for _, identifier := range state.Configuration {
		replica := &state.Replicas[identifier]
		if !state.Active[identifier] {
			return false
		}
		if replica.Status != vsr.Status_Normal {
			return false
		}
		if replica.Epoch != state.Epoch {
			return false
		}
	}
	return true
}

// Chooses the next configuration the administrator moves the group to, cycling grow (3->5), shrink
// (back toward 3), and swap (replace one member with a fresh pre-allocated node) by the
// reconfigure count, so a run exercises all three membership changes. It returns false when the
// chosen change is not possible — no fresh node left for a swap, or a size change that would breach
// the three-member minimum.
func simulator_next_configuration(state *simulator) (target vsr.Configuration, ok bool) {
	current := append(vsr.Configuration{}, state.Configuration...)
	switch state.Admin.Request_Number % 3 {
	case 0:
		// Grow: add the next two DISTINCT pristine nodes (the second searched past the
		// first, so neither re-adds a current member — a duplicate breaks quorum math).
		first := simulator_next_fresh_from(state, 0)
		second := simulator_next_fresh_from(state, first+1)
		if second >= sim_superset {
			return target, false
		}
		target = append(current,
			vsr.Replica_Identifier(first), vsr.Replica_Identifier(second))
	case 1:
		// Shrink: drop the last two members if that keeps at least three.
		if len(current) < 5 {
			return target, false
		}
		target = current[:len(current)-2]
	default:
		// Swap: replace the last member with the next fresh node.
		if state.Next_Fresh >= sim_superset {
			return target, false
		}
		target = append(current[:len(current)-1],
			vsr.Replica_Identifier(state.Next_Fresh))
	}
	if len(target) < 3 {
		return target, false
	}
	return target, true
}

// The active replica indices: members of the current epoch participating now (not dormant, not shut
// down). It scopes the fault injector and is the simulator's working set.
func simulator_active_indices(state *simulator) (indices []int) {
	for index := range state.Replicas {
		if state.Active[index] {
			indices = append(indices, index)
		}
	}
	return indices
}

// Folds the current moment into every ACTIVE replica and ships what each emits, routing any replies
// to their clients and advancing the verification models. A dormant pre-allocated node and a
// shut-down replica are not ticked: they are out of the protocol. Membership is refreshed afterward
// so a replica that shut down or joined this tick is reflected before delivery.
// Returns a replica's perceived time: its own virtual clock's reading. Each replica reads its own
// clock rather than a shared global one, so no two nodes' clocks are assumed equal — the realistic
// model, and the adversarial test that VSR safety depends only on consensus, never on the clock.
// With skew off this equals the true clock; the skew step bends Epoch/drift per replica and an
// injected clock fault offsets it for a window.
func simulator_replica_now(state *simulator, index int) (now time.Moment) {
	now = state.Replica_Clocks[index].Now_Realtime()
	// A transient clock fault shifts perceived time until it heals at Clock_Fault_Until; a zero
	// (unset) deadline is always in the past, so a replica with no fault is unaffected.
	if state.Clock.Now_Monotonic() < state.Clock_Fault_Until[index] {
		now += time.Moment(state.Clock_Fault_Offset[index])
	}
	return now
}

func simulator_tick_replicas(state *simulator, now time.Moment) {
	for index := range state.Replicas {
		if !state.Active[index] {
			continue
		}
		output := vsr.Replica_Tick(&vsr.Replica_Tick_Input{
			Replica: &state.Replicas[index], Now: simulator_replica_now(state, index),
		})
		simulator_handle_output(state, output, now)
	}
	simulator_refresh_membership(state)
}

// Refreshes the simulator's membership view from the replicas' own state: a replica that has shut
// down leaves the active set, and the current epoch and configuration are taken from the
// highest-epoch active normal replica. This tracks the protocol's membership decisions rather than
// dictating them — the simulator follows where the replicas have actually moved.
func simulator_refresh_membership(state *simulator) {
	for index := range state.Replicas {
		if state.Replicas[index].Status == vsr.Status_Shutdown {
			state.Active[index] = false
		}
	}
	highest := state.Epoch
	for index := range state.Replicas {
		if !state.Active[index] {
			continue
		}
		replica := &state.Replicas[index]
		if replica.Status != vsr.Status_Normal {
			continue
		}
		if replica.Epoch < highest {
			continue
		}
		if replica.Epoch > highest {
			highest = replica.Epoch
		}
		if configuration_contains(replica.Configuration, replica.Identifier) {
			if replica.Epoch == highest {
				state.Configuration = append(
					vsr.Configuration{}, replica.Configuration...)
			}
		}
	}
	if highest > state.Epoch {
		state.Result.Reconfigurations_Completed++
		state.Next_Fresh = simulator_next_fresh(state)
	}
	state.Epoch = highest
}

// The next pre-allocated index a future swap or grow brings in as a brand-new node. It walks the
// superset for the first PRISTINE identifier — one not in the current group and never used before —
// so a reconfiguration never re-adds a node that is already a member, nor resurrects a spent one. A
// replaced node has shut down for good and advanced its epoch past zero; re-adding its identifier
// would plant a dead, un-catchable node in the voting set that no fault-free tail could ever bring
// to Normal (the wedge the liveness tail surfaced).
func simulator_next_fresh(state *simulator) (index int) {
	return simulator_next_fresh_from(state, 0)
}

// The first pristine identifier at or after start. Splitting out the start lets a grow pick TWO
// distinct fresh nodes: the second search begins past the first, so it never re-adds the first nor
// any current member (the duplicate-member config a naive Next_Fresh+1 produced, which broke quorum
// math and tripped the commit-not-past-op assertion).
func simulator_next_fresh_from(state *simulator, start int) (index int) {
	for index = start; index < sim_superset; index++ {
		if configuration_contains(state.Configuration, vsr.Replica_Identifier(index)) {
			continue
		}
		if state.Replicas[index].Epoch != 0 {
			continue
		}
		if state.Replicas[index].Status == vsr.Status_Shutdown {
			continue
		}
		return index
	}
	return sim_superset
}

// Issues each idle client's next request to the believed primary, re-sends a timed-out request to
// every replica (so a retry reaches whichever node is now primary), and occasionally forgets a
// client's request-number to model §4.5 client recovery.
func simulator_tick_clients(state *simulator, now time.Moment) {
	primary := simulator_believed_primary(state)
	timeout := time.Moment(sim_client_timeout) * time.Moment(time.Millisecond)
	for index := range state.Clients {
		this := &state.Clients[index]
		if this.Unanswered {
			if now-this.Request_Moment >= timeout {
				this.Request_Moment = now
				simulator_broadcast_request(state, this, now)
			}
			continue
		}
		if state.Faultless {
			continue // The tail drains open requests; it issues no new ones.
		}
		recover := prng.Generator_Below(&state.Generator, 100) < sim_client_recover_percent
		if recover {
			// Forget the request-number and re-send the last command to relearn the
			// number from the cached reply, the §4.5 recovery path. A client with no
			// history skips.
			if this.Request_Number == 0 {
				continue
			}
			this.Recovery = true
			this.Unanswered = true
			this.Request_Moment = now
			this.Request_Command = client_last_command(this)
			simulator_broadcast_request(state, this, now)
			continue
		}
		this.Request_Number++
		this.Unanswered = true
		this.Request_Moment = now
		this.Request_Command = client_command(this.Identifier, this.Request_Number)
		// The request carries the simulator's current epoch — a client that has learned the
		// latest configuration out of band (§7.4) — so the primary accepts it rather than
		// redirecting it as stale.
		simulator_send(state, []vsr.Message{{
			Kind:           vsr.Message_Kind_Request,
			From:           primary,
			To:             primary,
			Epoch:          state.Epoch,
			Client:         this.Identifier,
			Request_Number: this.Request_Number,
			Command:        this.Request_Command,
		}}, now)
	}
}

// The command for a client's request-number, a deterministic function of (identifier, number) so a
// §4.5-recovering client that kept only its number rebuilds the identical command — read or write
// alike. Every third request is a read ("read-" prefix), so the cluster runs a read/write mix; a
// read is an ordinary command the state machine answers through consensus, TigerBeetle's design (it
// has no lease fast-read path), so the core treats it no differently from a write.
func client_command(identifier vsr.Client_Identifier, number vsr.Request_Number) (command []byte) {
	if number%3 == 0 {
		return []byte(fmt.Sprintf("read-client-%d-req-%d", identifier, number))
	}
	return []byte(fmt.Sprintf("client-%d-req-%d", identifier, number))
}

// The command a recovering client re-sends: it rebuilds the command deterministically from its
// identifier and request-number, the only state a §4.5-recovering client keeps.
func client_last_command(this *client) (command []byte) {
	return client_command(this.Identifier, this.Request_Number)
}

// Reports whether a command is a read (§6.3): the state machine answers it from current state
// without mutating it. Reads run through consensus as ordinary ops, so this is only the
// application's own interpretation, never a protocol distinction.
func command_is_read(command []byte) (yes bool) {
	return len(command) >= 5 && string(command[:5]) == "read-"
}

// Re-sends a client's outstanding request to every active replica, so a retry after a view change
// or an epoch change still reaches the new primary; non-primaries ignore it, and the primary dedups
// it. The request carries the current epoch, so an old replica that has fallen behind redirects the
// client rather than the request being silently lost.
func simulator_broadcast_request(state *simulator, this *client, now time.Moment) {
	var messages []vsr.Message
	for index := range state.Replicas {
		if !state.Active[index] {
			continue
		}
		messages = append(messages, vsr.Message{
			Kind:           vsr.Message_Kind_Request,
			From:           vsr.Replica_Identifier(index),
			To:             vsr.Replica_Identifier(index),
			Epoch:          state.Epoch,
			Client:         this.Identifier,
			Request_Number: this.Request_Number,
			Command:        this.Request_Command,
		})
	}
	simulator_send(state, messages, now)
}

// Ships a step's outgoing messages, advances the reference over anything newly committed, and
// routes the step's replies to their clients.
func simulator_handle_output(state *simulator, output vsr.Step_Output, now time.Moment) {
	simulator_send(state, output.Messages, now)
	state.Result.Commits += len(output.Committed)
	simulator_advance_reference(state)
	for _, reply := range output.Replies {
		// A New_Epoch redirect (§7.4) rides in Replies addressed to a client, but it is not
		// a command result: the simulator's clients learn the epoch out of band (they stamp
		// state.Epoch), so the redirect is dropped here and kept out of the result and
		// exactly-once oracles, which only a true Reply feeds.
		if reply.Kind == vsr.Message_Kind_New_Epoch {
			continue
		}
		state.Result.Replies++
		simulator_observe_reply(state, reply)
		simulator_route_reply(state, reply)
	}
}

// Delivers one reply to its addressed client: the client accepts a result for its outstanding
// request-number, clears the in-flight flag, and — if it was recovering — relearns the
// request-number it had forgotten. The exactly-once result map already enforces that the number is
// never reused for a different command.
func simulator_route_reply(state *simulator, reply vsr.Message) {
	for index := range state.Clients {
		this := &state.Clients[index]
		if this.Identifier != reply.Client {
			continue
		}
		if !this.Unanswered {
			return
		}
		if reply.Request_Number != this.Request_Number {
			return // A stale or duplicate reply for an already-satisfied request.
		}
		if this.Recovery {
			state.Result.Client_Recoveries++
			this.Recovery = false
		}
		state.Result.Client_Results++
		this.Unanswered = false
		return
	}
}

// Simulator_count_delivery records the coverage a delivery exercises: a transitioning new-group
// member adopting a checkpoint-bearing New_State (§7.1.1 empty-log catch-up), and a Request
// answered from the client-table without committing (a §4.5 cached re-send).
func simulator_count_delivery(
	state *simulator, message vsr.Message, target_transition bool, output vsr.Step_Output,
) {
	if target_transition {
		if message.Kind == vsr.Message_Kind_New_State {
			if message.Checkpoint_State != nil {
				state.Result.Epoch_Checkpoint_Catch_Ups++
			}
		}
	}
	if message.Kind == vsr.Message_Kind_Request {
		if len(output.Replies) > 0 {
			if len(output.Committed) == 0 {
				state.Result.Cached_Replies++
			}
		}
	}
}

// Simulator_drop_reason reports why the network drops this message before delivery, or "" if it is
// deliverable: a partition on the sender, a partition on the target, or a shut-down/dormant target
// (§7.1). The string doubles as the trace event name.
func simulator_drop_reason(state *simulator, message vsr.Message, now time.Moment) (reason string) {
	if simulator_is_isolated(state, message.From, now) {
		return "drop:iso-from"
	}
	if simulator_is_isolated(state, message.To, now) {
		return "drop:iso-to"
	}
	if !simulator_deliverable(state, message) {
		return "drop:undeliverable"
	}
	return ""
}

// Delivers every message now due, in a shuffled order, dropping any to or from a partitioned
// replica, and asserting safety after each delivery.
func simulator_deliver(state *simulator, now time.Moment) {
	var held_back, due []scheduled_message
	for _, flight := range state.Network {
		if flight.Deliver_At <= now {
			due = append(due, flight)
		} else {
			held_back = append(held_back, flight)
		}
	}
	state.Network = held_back
	prng.Generator_Shuffle(&state.Generator, due)
	for _, flight := range due {
		message := flight.Message
		if reason := simulator_drop_reason(state, message, now); reason != "" {
			jlog.Logger_Info(state.Trace, reason, trace_message_fields(now, message)...)
			continue // Partition, or shut-down/dormant target; senders' timers retry.
		}
		// A Start_Epoch to a dormant pre-allocated node adds it to the group (§7.1):
		// activate it so it is ticked and delivered to from here on, then let it process
		// the message.
		if message.Kind == vsr.Message_Kind_Start_Epoch {
			state.Active[message.To] = true
		}
		target := &state.Replicas[message.To]
		target_transition := target.Status == vsr.Status_Transition
		if state.Trace_Sink != nil {
			jlog.Logger_Info(state.Trace, "deliver",
				trace_message_fields(now, message)...)
			jlog.Logger_Info(state.Trace, "before", trace_replica_fields(target)...)
		}
		// A replica processes a received message on its own clock, like a tick, not a
		// global one it cannot read. Arming the timer here off the true clock while ticks
		// read the perceived clock lets drift diverge them by ~a full Timeout, firing a
		// just-armed view-change timer almost at once, thrashing the view under skew.
		output := vsr.Replica_Receive(&vsr.Replica_Receive_Input{
			Replica: target,
			Message: message,
			Now:     simulator_replica_now(state, int(message.To)),
		})
		if state.Trace_Sink != nil {
			jlog.Logger_Info(state.Trace, "after", trace_replica_fields(target)...)
			for index := range output.Messages {
				jlog.Logger_Info(state.Trace, "send",
					trace_message_fields(now, output.Messages[index])...)
			}
			for index := range output.Committed {
				entry := output.Committed[index]
				jlog.Logger_Info(state.Trace, "commit", jlog.Int64("t", now),
					jlog.Uint64("birth_view", entry.View),
					jlog.String("cmd", string(entry.Command)))
			}
		}
		simulator_count_delivery(state, message, target_transition, output)
		simulator_handle_output(state, output, now)
		simulator_refresh_membership(state)
		simulator_assert_safety(state)
	}
}

// Reports whether a message may be delivered to its target. A shut-down replica receives nothing. A
// dormant pre-allocated node (not yet added to any group) receives only a Start_Epoch — the message
// that adds it; everything else to it is dropped, since it is not in play.
func simulator_deliverable(state *simulator, message vsr.Message) (deliverable bool) {
	if state.Replicas[message.To].Status == vsr.Status_Shutdown {
		// A shut-down node accepts only a Start_Epoch: a later reconfiguration that
		// re-adds its identifier revives it as a fresh member (the paper provisions a
		// fresh node for the new configuration). Without this a node that shut down when
		// an earlier epoch dropped it could never rejoin when a later epoch adds it back,
		// wedging the cluster below quorum; receive_start_epoch's adopt path catches it up.
		return message.Kind == vsr.Message_Kind_Start_Epoch
	}
	if state.Active[message.To] {
		return true
	}
	return message.Kind == vsr.Message_Kind_Start_Epoch
}

// Reports whether identifier is a member of configuration — the simulation-side mirror of the
// core's own membership test, used to scope the epoch-aware checks to a configuration's members.
func configuration_contains(
	configuration vsr.Configuration, identifier vsr.Replica_Identifier,
) (yes bool) {
	for _, member := range configuration {
		if member == identifier {
			return true
		}
	}
	return false
}

// Reports whether the replica is currently inside its partition window.
func simulator_is_isolated(
	state *simulator, identifier vsr.Replica_Identifier, now time.Moment,
) (yes bool) {
	return now < state.Isolated_Until[identifier]
}

// Advances the linearizability oracle over every op now committed cluster-wide that it has not yet
// folded in. It reads each op's command from a replica that holds it (agreement guarantees they
// agree), folds it into the reference machine in op order, and records the canonical result and the
// request that op serves — the ground truth every reply and execution is checked against.
func simulator_advance_reference(state *simulator) {
	highest := vsr.Commit(0)
	for index := range state.Replicas {
		if !state.Active[index] {
			continue
		}
		if state.Replicas[index].Commit > highest {
			highest = state.Replicas[index].Commit
		}
	}
	for op := vsr.Op(state.Reference.Applied) + 1; op <= vsr.Op(highest); op++ {
		entry, ok := simulator_committed_entry(state, op)
		if !ok {
			break // No replica exposes this op yet; resume when one does.
		}
		state.Reference.Applied = int(op)
		// A Reconfiguration op (§7) is committed and ordered in the log but never
		// executed by the State_Machine: it folds no accumulator and produces no client
		// result. The accumulator snapshot at this op is therefore the unchanged value,
		// so a replica that commits up to a reconfiguration still matches the reference.
		if len(entry.New_Configuration) > 0 {
			state.Reference.Accumulator_Of_Commit[op] = state.Reference.Accumulator
			continue
		}
		// A read (§6.3, reads-through-consensus) returns the current state without folding
		// it, so the accumulator is unchanged at this op and the canonical result is the
		// state the executing replicas saw. Counted so a Sometimes-style cleanup confirms
		// reads actually ran.
		if command_is_read(entry.Command) {
			result := uint64_to_bytes(state.Reference.Accumulator)
			state.Reference.Result_Of_Op[op] = result
			state.Reference.Accumulator_Of_Commit[op] = state.Reference.Accumulator
			key := outcome_key{Client: entry.Client, Request: entry.Request_Number}
			state.Reference.Request_Of_Op[op] = key
			simulator_record_outcome(state, key, result)
			state.Result.Reads_Committed++
			continue
		}
		// The committed entry's prediction is identical on every replica (log agreement
		// plus the verbatim-copy invariant), so applying it here reproduces the value every
		// replica's Execute saw — the linearizability ground truth for a §4.4 op.
		result := state_machine_apply(&state_machine_apply_input{
			Command:    entry.Command,
			Prediction: entry.Prediction,
		})
		state.Reference.Result_Of_Op[op] = result
		// Fold the same (command, prediction) into the reference accumulator and snapshot
		// it at this op, so a replica's live accumulator at commit op can be checked
		// against the canonical value for exactly that many committed ops — the
		// end-to-end snapshot/restore correctness oracle.
		state.Reference.Accumulator = accumulator_fold(&accumulator_fold_input{
			Previous:   state.Reference.Accumulator,
			Command:    entry.Command,
			Prediction: entry.Prediction,
		})
		state.Reference.Accumulator_Of_Commit[op] = state.Reference.Accumulator
		if len(entry.Prediction) > 0 {
			state.Result.Predictions_Stamped++
		}
		key := outcome_key{Client: entry.Client, Request: entry.Request_Number}
		state.Reference.Request_Of_Op[op] = key
		simulator_record_outcome(state, key, result)
	}
}

// Returns the committed entry at a 1-based op from any replica that both commits it and still
// retains it in its log (an op compacted below Log_Start lives only in that replica's checkpoint,
// so it is skipped in favor of one that still holds the entry). All such replicas agree (the
// agreement check enforces it), so the first found is canonical.
func simulator_committed_entry(state *simulator, op vsr.Op) (entry vsr.Log_Entry, ok bool) {
	for index := range state.Replicas {
		if !state.Active[index] {
			continue // A dormant or shut-down replica's log is not canonical.
		}
		replica := &state.Replicas[index]
		if vsr.Commit(op) > replica.Commit {
			continue
		}
		if op <= replica.Log_Start {
			continue // Compacted away on this replica; another still holds it.
		}
		return replica.Log[op-replica.Log_Start-1], true
	}
	return entry, false
}

// Checks one replica's execution against the exactly-once oracle: a command is globally unique to
// one client request, so a replica executing the same command twice is a double-execution — the
// core wrongly running a committed op more than once. The (client, request) result map is fed from
// replies and the reference, which carry reliable keys; execution feeds only this op-level check.
func simulator_observe_execution(state *simulator, index int, command []byte) {
	// A standby never runs the service (§6.1): if one ever makes an Execute up-call, the role
	// split has leaked. This is the headline standby safety property.
	if is_standby(&state.Replicas[index]) {
		state.T.Fatalf("seed %d: standby replica %d executed command %q",
			state.Seed, index, command)
	}
	if state.Executed[index][string(command)] {
		state.T.Fatalf("seed %d: replica %d executed command %q twice",
			state.Seed, index, command)
	}
}

// Checks a reply against the oracles: its result must equal the reference machine's result for the
// op serving that (client, request-number), and that pair must map to a single result everywhere.
func simulator_observe_reply(state *simulator, reply vsr.Message) {
	key := outcome_key{Client: reply.Client, Request: reply.Request_Number}
	simulator_record_outcome(state, key, reply.Result)
	for op, served := range state.Reference.Request_Of_Op {
		if served != key {
			continue
		}
		expected := state.Reference.Result_Of_Op[op]
		if string(expected) != string(reply.Result) {
			state.T.Fatalf("seed %d: reply for %+v = %q, reference op %d = %q",
				state.Seed, key, reply.Result, op, expected)
		}
	}
}

// Files a (client, request-number) result into the exactly-once map, failing the seed if the same
// request is ever seen mapping to a second, different result.
func simulator_record_outcome(state *simulator, key outcome_key, result []byte) {
	previous, seen := state.Outcomes[key]
	if seen {
		if string(previous) != string(result) {
			state.T.Fatalf("seed %d: request %+v mapped to two results: %q vs %q",
				state.Seed, key, previous, result)
		}
		return
	}
	stored := make([]byte, len(result))
	copy(stored, result)
	state.Outcomes[key] = stored
}

// Applies the lossy network to a step's outgoing messages: each may be dropped, delayed, and
// duplicated independently.
func simulator_send(state *simulator, messages []vsr.Message, now time.Moment) {
	for _, message := range messages {
		if message.Kind == vsr.Message_Kind_Get_State {
			// Count the catch-up even when the network later drops the request.
			state.Result.State_Transfers++
		}
		if message.Kind == vsr.Message_Kind_Predict_Request {
			// Count the pre-step round even when the network later drops the request.
			state.Result.Pre_Step_Rounds++
		}
		if message.Kind == vsr.Message_Kind_Prepare {
			// A Prepare carrying more than one entry is a flushed batch (§6.2): the
			// primary collected several requests into one round. Count it even if later
			// dropped; the per-recipient copies overcount the fan-out, but the axis
			// only witnesses that batching happened at all.
			if len(message.Entries) > 1 {
				state.Result.Batches_Flushed++
			}
		}
		if message.Kind == vsr.Message_Kind_New_State {
			// A New_State carrying a checkpoint is the §5.2 gap response: the
			// requester needed a prefix the responder had already garbage-collected.
			// Count it even if later dropped.
			if message.Checkpoint_State != nil {
				state.Result.Checkpoint_Gaps++
			}
		}
		if message.Kind == vsr.Message_Kind_Do_View_Change {
			// A report whose op exceeds its suffix length dropped entries the reporter
			// still held: the §5.3 bounded suffix genuinely bit. Count it even if the
			// network later drops the report.
			if message.Op > vsr.Op(len(message.Log_Suffix)) {
				state.Result.Do_View_Change_Suffixes++
			}
		}
		if !state.Faultless {
			// The fault-free tail delivers everything; only the faulty phase drops.
			if prng.Generator_Below(&state.Generator, 100) < sim_drop_percent {
				continue
			}
		}
		copies := 1
		if prng.Generator_Below(&state.Generator, 100) < sim_duplicate_percent {
			copies = 2
		}
		for copy_index := 0; copy_index < copies; copy_index++ {
			grains := prng.Generator_Below(&state.Generator, sim_delay_max+1)
			delay := time.Moment(grains) * time.Moment(time.Millisecond)
			state.Network = append(state.Network, scheduled_message{
				Message: message, Deliver_At: now + delay,
			})
		}
	}
}

// Runs after every delivery: cheap plain-Go checks that fail fast and name the seed. These are the
// real safety net, so they run at the finest granularity.
func simulator_assert_safety(state *simulator) {
	for index := range state.Replicas {
		replica := &state.Replicas[index]
		if int(replica.Op) != int(replica.Log_Start)+len(replica.Log) {
			state.T.Fatalf("seed %d: replica %d op %d != Log_Start %d + log length %d",
				state.Seed, replica.Identifier, replica.Op, replica.Log_Start,
				len(replica.Log))
		}
		if replica.Commit > vsr.Commit(replica.Op) {
			state.T.Fatalf("seed %d: replica %d commit %d exceeds op %d",
				state.Seed, replica.Identifier, replica.Commit, replica.Op)
		}
	}
	simulator_check_single_primary(state)
	simulator_check_agreement(state)
	simulator_check_result_agreement(state)
	simulator_check_checkpoint_agreement(state)
	simulator_check_accumulator(state)
	simulator_check_fault_model(state)
}

// Asserts the harness keeps the cluster within its fault model: the voting set tolerates
// f = (active_count-1)/2 faults, so at most f voting replicas may be recovering at once. Past that,
// fewer than a quorum are Normal, so no one can answer the recoveries and the cluster is
// unrecoverable — a state the fault-free tail could never drain. Checking after every delivery pins
// the exact step that broke the model, not a wedge 1500 ticks later.
func simulator_check_fault_model(state *simulator) {
	voting_count := int(active_count_for(len(state.Configuration)))
	if voting_count > len(state.Configuration) {
		voting_count = len(state.Configuration)
	}
	f := (voting_count - 1) / 2
	recovery_count := 0
	for position_index := 0; position_index < voting_count; position_index++ {
		identifier := state.Configuration[position_index]
		if !state.Active[identifier] {
			continue
		}
		if state.Replicas[identifier].Status == vsr.Status_Recovery {
			recovery_count++
		}
	}
	if recovery_count > f {
		state.T.Fatalf("seed %d: %d of %d voting recovering, over f=%d (fault model)",
			state.Seed, recovery_count, voting_count, f)
	}
}

// Asserts any two replicas that have taken a checkpoint at the same op hold byte-identical
// checkpoint state. A checkpoint is the snapshot of the committed prefix up to its op; since the
// committed prefix is identical cluster-wide, two checkpoints at the same op must capture the same
// application state. A divergence means a replica's state machine reached the checkpoint op along a
// different (buggy) execution path than its peers.
func simulator_check_checkpoint_agreement(state *simulator) {
	at_op := map[vsr.Op][]byte{}
	owner := map[vsr.Op]int{}
	for index := range state.Replicas {
		replica := &state.Replicas[index]
		if replica.Checkpoint_Op == 0 {
			continue
		}
		seen, ok := at_op[replica.Checkpoint_Op]
		if !ok {
			at_op[replica.Checkpoint_Op] = replica.Checkpoint_State
			owner[replica.Checkpoint_Op] = index
			continue
		}
		if string(seen) != string(replica.Checkpoint_State) {
			state.T.Fatalf(
				"seed %d: checkpoint op %d diverges: r%d=%x r%d=%x",
				state.Seed, replica.Checkpoint_Op, owner[replica.Checkpoint_Op],
				seen, index, replica.Checkpoint_State)
		}
	}
}

// Asserts each replica's live application accumulator equals the reference accumulator for the
// first Commit ops. A replica that adopted a committed prefix without executing or restoring it —
// the stale-state bug the CRITICAL design note warns of — would land on a different accumulator
// than the reference, which folds every committed op exactly once in order. The check runs only
// when the reference has folded at least as far as the replica's commit, so a replica momentarily
// ahead of the oracle is skipped until the oracle catches up.
func simulator_check_accumulator(state *simulator) {
	for index := range state.Replicas {
		if !state.Active[index] {
			continue // A dormant or shut-down replica is out of play.
		}
		replica := &state.Replicas[index]
		// A standby never executes (§6.1): it maintains no application state, so its
		// accumulator stays zero while its commit advances. The accumulator oracle is for
		// active replicas.
		if is_standby(replica) {
			continue
		}
		// A recovering replica's accumulator is mid-restore (its volatile state was lost
		// and the rejoin has not completed), so it is not yet meaningful to compare.
		if replica.Status == vsr.Status_Recovery {
			continue
		}
		// A transitioning replica is mid-catch-up to a new epoch (§7.1), its accumulator
		// not yet at its commit; compare it only once it is back to normal.
		if replica.Status == vsr.Status_Transition {
			continue
		}
		expected, ok := state.Reference.Accumulator_Of_Commit[vsr.Op(replica.Commit)]
		if !ok {
			continue // The oracle has not folded this far yet.
		}
		if state.Accumulator[index] != expected {
			state.T.Fatalf(
				"seed %d: replica %d accumulator %x at commit %d != reference %x",
				state.Seed, replica.Identifier, state.Accumulator[index],
				replica.Commit, expected)
		}
	}
}

// Asserts every replica that has executed a given command produced the byte-identical result. A
// command's result depends on its predetermined prediction (§4.4); because the primary stamps that
// prediction once and every replica copies it verbatim, the results must match. A divergence here
// would mean a replica executed the op against a different value than its peers — the very
// failure predetermination exists to prevent.
func simulator_check_result_agreement(state *simulator) {
	results := map[string][]byte{}
	owners := map[string]int{}
	for index := range state.Replicas {
		for command, result := range state.Executed_Result[index] {
			seen, ok := results[command]
			if !ok {
				results[command] = result
				owners[command] = index
				continue
			}
			if string(seen) != string(result) {
				state.T.Fatalf(
					"seed %d: command %q executed to two results: "+
						"replica %d=%q replica %d=%q",
					state.Seed, command, owners[command], seen, index, result)
			}
		}
	}
}

// Records the per-replica liveness coverage axes and tallies the phase-completion counters for one
// replica: its status and view/epoch axes, compaction, the §5.3 fetch, and the view-change and
// recovery completions detected against its previous status.
func simulator_record_replica_coverage(state *simulator, index int) {
	replica := &state.Replicas[index]
	invariant.Dot_Product("sim.replica.status_normal",
		invariant.Sometimes(replica.Status == vsr.Status_Normal, "replica is normal"))
	invariant.Dot_Product("sim.replica.status_view_change",
		invariant.Sometimes(
			replica.Status == vsr.Status_View_Change,
			"replica is view-changing"))
	invariant.Dot_Product("sim.replica.status_recovery",
		invariant.Sometimes(replica.Status == vsr.Status_Recovery, "replica is recovering"))
	// The §7 reconfiguration statuses must be witnessed across the sweep: a replica mid epoch
	// handoff (transitioning) and a replaced replica that has shut down.
	invariant.Dot_Product("sim.replica.status_transition",
		invariant.Sometimes(
			replica.Status == vsr.Status_Transition,
			"replica is transitioning"))
	invariant.Dot_Product("sim.replica.status_shutdown",
		invariant.Sometimes(replica.Status == vsr.Status_Shutdown, "replica is shut down"))
	invariant.Dot_Product("sim.replica.view_advanced",
		invariant.Sometimes(replica.View > 0, "replica view advanced past zero"))
	// The epoch must sometimes advance past 0, witnessing a reconfiguration ran.
	invariant.Dot_Product("sim.replica.epoch_advanced",
		invariant.Sometimes(replica.Epoch > 0, "replica epoch advanced past zero"))
	if replica.Epoch > state.Result.Epoch_Max {
		state.Result.Epoch_Max = replica.Epoch
	}
	// Compaction must actually run: a replica's Log_Start must sometimes advance past zero,
	// witnessing the log prefix was garbage-collected rather than the cluster idling below the
	// first checkpoint.
	invariant.Dot_Product("sim.replica.log_compacted",
		invariant.Sometimes(replica.Log_Start > 0, "replica log prefix was compacted"))
	// A standby must sometimes exist and follow — be a non-voting member (§6.1) holding
	// committed log — so the standby paths (no vote, no execute, follow) are genuinely
	// exercised rather than every member always being active.
	invariant.Dot_Product("sim.replica.standby_follows",
		invariant.Sometimes(
			is_standby(replica) && replica.Op > 0,
			"standby exists and holds log"))
	// A new primary deferring its install while it fetches the selected log is the §5.3 fetch
	// path running; record it so the Sometimes axis can witness it.
	if replica.View_Change_Deferred {
		state.Result.View_Change_Fetches++
	}
	previous := state.Previous_Status[index]
	if previous == vsr.Status_View_Change {
		if replica.Status == vsr.Status_Normal {
			state.Result.View_Changes_Completed++
		}
	}
	if previous == vsr.Status_Recovery {
		if replica.Status == vsr.Status_Normal {
			state.Result.Recoveries_Completed++
		}
	}
	state.Previous_Status[index] = replica.Status
	if replica.View > state.Result.View_Max {
		state.Result.View_Max = replica.View
	}
}

// Records the §7.2 overlap counters: a replica mid epoch transition while another is mid view
// change or mid recovery, the concurrent paths the reconfiguration must survive.
func simulator_record_overlap_coverage(state *simulator) {
	any_transition := false
	any_view_change := false
	any_recovery := false
	for index := range state.Replicas {
		switch state.Replicas[index].Status {
		case vsr.Status_Transition:
			any_transition = true
		case vsr.Status_View_Change:
			any_view_change = true
		case vsr.Status_Recovery:
			any_recovery = true
		}
	}
	if !any_transition {
		return
	}
	if any_view_change {
		state.Result.Reconfigure_Over_View_Change++
	}
	if any_recovery {
		state.Result.Reconfigure_Over_Recovery++
	}
}

// Runs once per tick: the liveness coverage axes, whose runtime.Callers cost makes per-delivery use
// prohibitive. Per-replica safety Always assertions live in vsr.go itself; these Sometimes axes
// demand every liveness branch be observed across the seed sweep.
func simulator_record_coverage(state *simulator) {
	for index := range state.Replicas {
		simulator_record_replica_coverage(state, index)
	}
	simulator_record_overlap_coverage(state)
	// A client must sometimes have a request unanswered and sometimes not, so the request/reply
	// path is genuinely exercised rather than idling.
	for index := range state.Clients {
		invariant.Dot_Product("sim.client.unanswered",
			invariant.Sometimes(
				state.Clients[index].Unanswered,
				"client has an unanswered request"))
	}
	// Clock skew must be witnessed both ways: the spread between the fastest and slowest
	// replica's perceived time sometimes exceeds five milliseconds (the skew sweep) and
	// sometimes does not (the default shared-clock runs), so the model is genuinely exercised.
	invariant.Dot_Product("sim.clock.skew_observed",
		invariant.Sometimes(
			simulator_clock_skew(state) > time.Moment(5)*time.Moment(time.Millisecond),
			"replica clocks skewed past five milliseconds"))
	// A transient clock fault must sometimes be active (the skew sweep injects them) and
	// sometimes not, so the fault path is witnessed both ways across the sweep.
	invariant.Dot_Product("sim.clock.fault_active",
		invariant.Sometimes(
			simulator_clock_fault_active(state),
			"a replica clock is faulted"))
	simulator_record_result_coverage(state)
}

// Reports whether any active replica is currently inside a transient clock-fault window.
func simulator_clock_fault_active(state *simulator) (active bool) {
	now := state.Clock.Now_Monotonic()
	for index := range state.Replicas {
		if !state.Active[index] {
			continue
		}
		if now < state.Clock_Fault_Until[index] {
			return true
		}
	}
	return false
}

// Returns the spread between the highest and lowest perceived time across active replicas — the
// cluster's clock skew this tick. Zero when fewer than two replicas are active.
func simulator_clock_skew(state *simulator) (spread time.Moment) {
	low := time.Moment(0)
	high := time.Moment(0)
	seen := false
	for index := range state.Replicas {
		if !state.Active[index] {
			continue
		}
		now := simulator_replica_now(state, index)
		if !seen {
			low, high, seen = now, now, true
			continue
		}
		if now < low {
			low = now
		}
		if now > high {
			high = now
		}
	}
	return high - low
}

func simulator_record_result_coverage(state *simulator) {
	// Both new exactly-once branches must be witnessed across the sweep: a duplicate's cached
	// reply re-sent, and a client recovering its forgotten request-number. Each reads false on
	// a run's early ticks and true once the event occurs, covering both outcomes.
	invariant.Dot_Product("sim.result.cached_replies",
		invariant.Sometimes(state.Result.Cached_Replies > 0, "a cached reply was re-sent"))
	invariant.Dot_Product("sim.result.client_recoveries",
		invariant.Sometimes(
			state.Result.Client_Recoveries > 0,
			"a client recovered its request number"))
	// The §4.4 prediction path must be genuinely exercised: a committed op must sometimes
	// carry a non-empty predetermined value, and the pre-step variant must sometimes open a
	// round (it runs only on odd seeds, so a sweep over both parities witnesses both true and
	// false).
	invariant.Dot_Product("sim.result.predictions_stamped",
		invariant.Sometimes(
			state.Result.Predictions_Stamped > 0,
			"a committed op carried a prediction"))
	invariant.Dot_Product("sim.result.pre_step_rounds",
		invariant.Sometimes(
			state.Result.Pre_Step_Rounds > 0,
			"a pre-step prediction round opened"))
	// The §5.1/§5.2 checkpoint paths must genuinely run across the sweep: the gap response
	// must sometimes ship a checkpoint to a requester behind a GC'd prefix, and a crash must
	// sometimes recover warm from a surviving checkpoint.
	invariant.Dot_Product("sim.result.checkpoint_gaps",
		invariant.Sometimes(
			state.Result.Checkpoint_Gaps > 0,
			"a gap response shipped a checkpoint"))
	invariant.Dot_Product("sim.result.warm_recoveries",
		invariant.Sometimes(
			state.Result.Warm_Recoveries_Started > 0,
			"a warm recovery started"))
	// The §5.3 efficient-view-change paths must run across the sweep: a Do_View_Change must
	// sometimes carry a bounded suffix shorter than its op (dropping held entries), and the
	// new primary must sometimes defer its install to fetch the selected log it could not
	// reconstruct from that suffix alone.
	invariant.Dot_Product("sim.result.do_view_change_suffixes",
		invariant.Sometimes(
			state.Result.Do_View_Change_Suffixes > 0,
			"a do-view-change carried a bounded suffix"))
	invariant.Dot_Product("sim.result.view_change_fetches",
		invariant.Sometimes(
			state.Result.View_Change_Fetches > 0,
			"a new primary deferred to fetch the selected log"))
	// A reconfiguration must run to completion across the sweep: the epoch advanced and a new
	// group came up.
	invariant.Dot_Product("sim.result.reconfigurations_completed",
		invariant.Sometimes(
			state.Result.Reconfigurations_Completed > 0,
			"a reconfiguration completed"))
	// The §7 edge cases must genuinely run across the sweep: a brand-new node restoring a
	// checkpoint to catch up to the epoch start, a reconfiguration overlapping a view change,
	// and one overlapping recovery — the concurrent paths reconfiguration must survive.
	invariant.Dot_Product("sim.result.epoch_checkpoint_catch_ups",
		invariant.Sometimes(
			state.Result.Epoch_Checkpoint_Catch_Ups > 0,
			"a new node restored a checkpoint to catch up to the epoch"))
	invariant.Dot_Product("sim.result.reconfigure_over_view_change",
		invariant.Sometimes(
			state.Result.Reconfigure_Over_View_Change > 0,
			"a reconfiguration overlapped a view change"))
	invariant.Dot_Product("sim.result.reconfigure_over_recovery",
		invariant.Sometimes(
			state.Result.Reconfigure_Over_Recovery > 0,
			"a reconfiguration overlapped a recovery"))
}

// The acting primary of the current epoch's highest view — the simulator's stand-in for a client
// that has learned the latest group and leader (§7.4). It scopes to active normal replicas at the
// current epoch and reads each candidate's OWN configuration to decide whether it is its view's
// primary, so a grown, shrunk, or swapped group routes to the right node. It falls back to the
// current configuration's view-0 primary when no replica is settled yet.
func simulator_believed_primary(state *simulator) (identifier vsr.Replica_Identifier) {
	highest := vsr.View(0)
	found := false
	for index := range state.Replicas {
		if !state.Active[index] {
			continue
		}
		replica := &state.Replicas[index]
		if replica.Status != vsr.Status_Normal {
			continue
		}
		if replica.Epoch != state.Epoch {
			continue
		}
		if replica.Identifier != acting_primary(replica) {
			continue
		}
		newer := !found
		if replica.View > highest {
			newer = true
		}
		if newer {
			highest = replica.View
			identifier = replica.Identifier
			found = true
		}
	}
	if found {
		return identifier
	}
	return state.Configuration[0]
}

// The identifier this replica computes as the primary of its current view: Configuration[View mod
// Active_Count], mirroring the core's standby-aware leader rule (§6.1), so the simulator's
// single-primary and believed-primary checks agree with the replicas about who leads. An
// Active_Count of zero means no standbys, so the whole configuration is active.
func acting_primary(replica *vsr.Replica) (identifier vsr.Replica_Identifier) {
	active := uint64(replica.Active_Count)
	if active == 0 {
		active = uint64(len(replica.Configuration))
	}
	return replica.Configuration[uint64(replica.View)%active]
}

// The active (voting) replica count for a configuration of the given size. The voting set is fixed
// at three — f=1, a 2f+1 quorum set — and any members beyond it are non-voting standbys
// (TigerBeetle's design, see vsr.go replica_quorum). So a three-member configuration is all voting
// and a five-member one is three voting plus two standbys, and the voting count stays three across
// the simulator's grow and shrink, keeping the per-handoff quorum well-defined without tracking a
// separate old voting count.
func active_count_for(size int) (count uint8) {
	return 3
}

// Reports whether a replica is a standby in its current configuration (§6.1): at or beyond the
// active prefix. Standbys never execute, so the simulator's application-state oracles skip them.
func is_standby(replica *vsr.Replica) (yes bool) {
	active := int(replica.Active_Count)
	if active == 0 {
		active = len(replica.Configuration)
	}
	for index, member := range replica.Configuration {
		if member == replica.Identifier {
			return index >= active
		}
	}
	return false
}

// Builds the identity configuration [0, 1, .., count-1], so a replica's identifier is also its
// index in the configuration and the primary of view V is identifier V mod count.
func sorted_configuration(count int) (configuration vsr.Configuration) {
	configuration = make(vsr.Configuration, count)
	for index := range configuration {
		configuration[index] = vsr.Replica_Identifier(index)
	}
	return configuration
}

// Asserts no two replicas are simultaneously the acting primary of one (epoch, view).
// Single-primary is EPOCH-RELATIVE (§8.3): two primaries in the same view but different epochs is
// allowed — the old group's primary may re-run the reconfiguration while the new group's primary
// serves — so the key is the (epoch, view) pair, and each candidate's own configuration decides
// whether it is that view's primary, since configurations differ across epochs.
func simulator_check_single_primary(state *simulator) {
	state.T.Helper()
	type epoch_view struct {
		Epoch vsr.Epoch
		View  vsr.View
	}
	primary_of := map[epoch_view]vsr.Replica_Identifier{}
	for index := range state.Replicas {
		if !state.Active[index] {
			continue
		}
		replica := &state.Replicas[index]
		if replica.Status != vsr.Status_Normal {
			continue
		}
		if replica.Identifier != acting_primary(replica) {
			continue
		}
		key := epoch_view{Epoch: replica.Epoch, View: replica.View}
		other, seen := primary_of[key]
		if seen {
			if other != replica.Identifier {
				state.T.Fatalf(
					"seed %d: two primaries in epoch %d view %d: %d and %d",
					state.Seed, replica.Epoch, replica.View, other,
					replica.Identifier)
			}
		}
		primary_of[key] = replica.Identifier
	}
}

// Asserts every pair of ACTIVE replicas holds identical commands at every op both have committed
// AND both still retain — the protocol's core safety guarantee, restricted to the ops still in both
// logs. Agreement holds across the epoch boundary: op-numbers stay globally monotonic through a
// reconfiguration (the new group continues from the epoch-start op), so two replicas in different
// epochs still agree on any committed op both retain — the §8.3 condition that every committed op
// survives the epoch change. Compaction garbage-collects a committed prefix into a checkpoint, so
// an op below either replica's Log_Start is checked through simulator_check_checkpoint_agreement
// instead.
func simulator_check_agreement(state *simulator) {
	state.T.Helper()
	for index := range state.Replicas {
		if !state.Active[index] {
			continue
		}
		for other := index + 1; other < len(state.Replicas); other++ {
			if !state.Active[other] {
				continue
			}
			simulator_check_agreement_pair(&simulator_check_agreement_pair_input{
				State: state,
				A:     &state.Replicas[index],
				B:     &state.Replicas[other],
			})
		}
	}
}

// Check_agreement_pair_input is the two active replicas a pairwise agreement check compares.
type simulator_check_agreement_pair_input struct {
	// State is the run, for the seed and the failure reporter.
	State *simulator
	// A and B are the two replicas whose overlapping committed, retained ops must match.
	A *vsr.Replica
	B *vsr.Replica
}

// Asserts two active replicas hold identical commands at every op both committed and both retain.
func simulator_check_agreement_pair(input *simulator_check_agreement_pair_input) {
	a, b := input.A, input.B
	limit := a.Commit
	if b.Commit < limit {
		limit = b.Commit
	}
	// Start above the higher of the two Log_Starts: an op at or below either is compacted out
	// of that replica's log, so only the retained suffix both hold is comparable.
	first := a.Log_Start
	if b.Log_Start > first {
		first = b.Log_Start
	}
	for op := vsr.Commit(first) + 1; op <= limit; op++ {
		ai := op - vsr.Commit(a.Log_Start) - 1
		bi := op - vsr.Commit(b.Log_Start) - 1
		if string(a.Log[ai].Command) != string(b.Log[bi].Command) {
			jlog.Logger_Info(input.State.Trace, "VIOLATION agreement",
				jlog.Uint64("op", op),
				jlog.Uint8("a", a.Identifier),
				jlog.String("a_cmd", string(a.Log[ai].Command)),
				jlog.Uint64("a_birth_view", a.Log[ai].View),
				jlog.Uint64("a_epoch", a.Epoch),
				jlog.Uint64("a_view", a.View),
				jlog.Uint8("b", b.Identifier),
				jlog.String("b_cmd", string(b.Log[bi].Command)),
				jlog.Uint64("b_birth_view", b.Log[bi].View),
				jlog.Uint64("b_epoch", b.Epoch),
				jlog.Uint64("b_view", b.View))
			input.State.T.Fatalf("seed %d: op %d diverges: %d=%q(bview=%d ep=%d "+
				"view=%d commit=%d) %d=%q(bview=%d ep=%d view=%d commit=%d)",
				input.State.Seed, op,
				a.Identifier, a.Log[ai].Command, a.Log[ai].View, a.Epoch, a.View,
				a.Commit, b.Identifier, b.Log[bi].Command, b.Log[bi].View, b.Epoch,
				b.View, b.Commit)
		}
	}
}

// The trace facility emits one structured jlog line per simulation event — every delivery with the
// target's state before and after the step, every message produced, every commit, every injected
// fault, and the safety violation that ends the run — into a per-seed buffer flushed to a file, so
// a fork can be read offline with jq instead of being chased with throwaway prints that perturb the
// schedule. It is inert unless VSR_TRACE names the running seed, so the sweep pays nothing for it.

// Trace_enabled reports whether VSR_TRACE selects this seed. Both the skew-off and skew-on runs of
// the seed trace, to separate files, because a fork often reproduces under only one of them.
func trace_enabled(seed int64) (enabled bool) {
	want := os.Getenv("VSR_TRACE")
	if want == "" {
		return false
	}
	parsed, err := strconv.ParseInt(want, 10, 64)
	if err != nil {
		return false
	}
	return parsed == seed
}

// Simulator_flush_trace writes the buffered trace to /tmp once the run ends — including when a
// safety Fatalf unwinds the goroutine, since the caller defers this. The filename carries the seed
// and skew so the off and on runs do not clobber each other.
func simulator_flush_trace(state *simulator) {
	if state.Trace_Sink == nil {
		return
	}
	name := fmt.Sprintf("/tmp/vsr_trace_%d_skew_%v.log", state.Seed, state.Clock_Skew)
	if err := os.WriteFile(name, state.Trace_Sink.Bytes(), 0o644); err != nil {
		state.T.Logf("trace flush to %s failed: %v", name, err)
	}
}

// Trace_message_fields builds the jlog fields for a message — its routing and consensus numbers
// plus whichever log slice it carries (Prepare in Entries, Start_View/New_State in Log,
// Do_View_Change in Log_Suffix) — for a caller to pass straight to jlog.Logger_Info.
func trace_message_fields(now time.Moment, message vsr.Message) (fields []jlog.Field) {
	fields = []jlog.Field{
		jlog.Int64("t", now),
		jlog.String("kind", message_kind_name(message.Kind)),
		jlog.Uint8("from", message.From),
		jlog.Uint8("to", message.To),
		jlog.Uint64("view", message.View),
		jlog.Uint64("epoch", message.Epoch),
		jlog.Uint64("op", message.Op),
		jlog.Uint64("commit", message.Commit),
	}
	if message.Nonce != 0 {
		fields = append(fields, jlog.Uint64("nonce", message.Nonce))
	}
	if entries := trace_entries(message.Entries); len(entries) > 0 {
		fields = append(fields, jlog.Strings("entries", entries))
	}
	if log := trace_entries(message.Log); len(log) > 0 {
		fields = append(fields, jlog.Strings("log", log))
	}
	if suffix := trace_entries(message.Log_Suffix); len(suffix) > 0 {
		fields = append(fields, jlog.Strings("suffix", suffix))
	}
	return fields
}

// Trace_replica_fields builds the jlog fields for a replica's consensus state and live log — the
// before/after snapshot around a step — for a caller to pass straight to jlog.Logger_Info.
func trace_replica_fields(replica *vsr.Replica) (fields []jlog.Field) {
	return []jlog.Field{
		jlog.Uint8("id", replica.Identifier),
		jlog.String("status", status_name(replica.Status)),
		jlog.Uint64("view", replica.View),
		jlog.Uint64("epoch", replica.Epoch),
		jlog.Uint64("op", replica.Op),
		jlog.Uint64("commit", replica.Commit),
		jlog.Uint64("log_start", replica.Log_Start),
		jlog.Uint64("epoch_start_op", replica.Epoch_Start_Op),
		jlog.String("config", fmt.Sprintf("%v", replica.Configuration)),
		jlog.Integer("active_count", int(replica.Active_Count)),
		jlog.Integer("recovery_responses", len(replica.Recovery_From)),
		jlog.Uint64("recovery_nonce", replica.Recovery_Nonce),
		jlog.Strings("log", trace_entries(replica.Log)),
	}
}

// Message_kind_name renders a kind as its name, falling back to the integer for an unknown value.
func message_kind_name(kind vsr.Message_Kind) (name string) {
	switch kind {
	case vsr.Message_Kind_Request:
		return "Request"
	case vsr.Message_Kind_Prepare:
		return "Prepare"
	case vsr.Message_Kind_Prepare_Ok:
		return "Prepare_Ok"
	case vsr.Message_Kind_Commit:
		return "Commit"
	case vsr.Message_Kind_Start_View_Change:
		return "Start_View_Change"
	case vsr.Message_Kind_Do_View_Change:
		return "Do_View_Change"
	case vsr.Message_Kind_Start_View:
		return "Start_View"
	case vsr.Message_Kind_Recovery:
		return "Recovery"
	case vsr.Message_Kind_Recovery_Response:
		return "Recovery_Response"
	case vsr.Message_Kind_Get_State:
		return "Get_State"
	case vsr.Message_Kind_New_State:
		return "New_State"
	case vsr.Message_Kind_Reply:
		return "Reply"
	case vsr.Message_Kind_Predict_Request:
		return "Predict_Request"
	case vsr.Message_Kind_Predict_Response:
		return "Predict_Response"
	case vsr.Message_Kind_Reconfiguration:
		return "Reconfiguration"
	case vsr.Message_Kind_Start_Epoch:
		return "Start_Epoch"
	case vsr.Message_Kind_Epoch_Started:
		return "Epoch_Started"
	case vsr.Message_Kind_New_Epoch:
		return "New_Epoch"
	case vsr.Message_Kind_Check_Epoch:
		return "Check_Epoch"
	}
	return fmt.Sprintf("Kind(%d)", int(kind))
}

// Status_name renders a status as its name, falling back to the integer for an unknown value.
func status_name(status vsr.Status) (name string) {
	switch status {
	case vsr.Status_Normal:
		return "Normal"
	case vsr.Status_View_Change:
		return "View_Change"
	case vsr.Status_Recovery:
		return "Recovery"
	case vsr.Status_Transition:
		return "Transition"
	case vsr.Status_Shutdown:
		return "Shutdown"
	}
	return fmt.Sprintf("Status(%d)", int(status))
}

// Trace_entries renders a log slice as one "vBIRTH:COMMAND" string per entry, so a reused or
// resurrected op-number is visible at a glance in the resulting JSON array.
func trace_entries(entries []vsr.Log_Entry) (rendered []string) {
	rendered = make([]string, 0, len(entries))
	for index := range entries {
		entry := entries[index]
		rendered = append(rendered, fmt.Sprintf("v%d:%s", entry.View, entry.Command))
	}
	return rendered
}
