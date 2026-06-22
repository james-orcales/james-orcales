---
sut_path: /Users/james-orcales/code/james-orcales/shared/vsr
commit: 235c868dd644eb28f9a46752017766d84d87207d
updated: 2026-06-16
external_references:
  - path: documentation/resources/Viewstamped_Replication_Revisited.pdf
    why: the protocol's claimed guarantees (Liskov & Cowling 2012) — source of most properties
  - path: /Users/james-orcales/code/tigerbeetle
    why: priority authority for deliberate decisions (non-voting standbys, state sync, no leases)
---

# `shared/vsr` property catalog

The testable correctness properties of the VSR core, from applying the *applicable* parts of the
`antithesis-research` methodology to a pure deterministic protocol core. This is not an
Antithesis-platform artifact — no containers, SDK, or deployment topology. The `shared/invariant`
framework is the assertion system; the VOPR simulator (`shared/vsr/simulation`) is the harness.

## How to read this

Assertion vocabulary maps the methodology's Antithesis SDK onto `shared/invariant`: `Always` →
`invariant.Always` (safety; must be *reached*); `Sometimes(cond)` → `invariant.Sometimes` (must be
observed *both ways* — stricter than Antithesis); `Reachable` → a meaningful `invariant.Sometimes`
(no one-shot form exists); `Unreachable` → `invariant.Impossible`. The cross-cluster safety oracles
(agreement, single-primary, exactly-once, linearizability, checkpoint-agreement) are direct
comparison checks in the simulator (`t.Fatalf` on violation), not `invariant.*` calls; they run in
`simulator_assert_safety` after every delivery.

Tiers say where a property lives: **Tier-1** is one replica's local state (an `invariant.*` in
`vsr.go`); **Tier-2** is a cross-cluster property (a simulator oracle or coverage axis).

Status is one of: `enforced-inline` (an `invariant.*` in vsr.go), `enforced-simulator` (a simulator
oracle / coverage axis), `GAP` (a real guarantee with no current runtime check), or `partial`
(behaviourally enforced but not asserted as a named property).

Reachability discipline: any `Sometimes`/`Distinct_Boundary` must have both branches / both
endpoints genuinely reachable in the sweep, or it gaps `Run_Test_Main`. Enum-range and size-bound
properties are therefore phrased as reach-only `Always`, never a full-type-range boundary
(`Whole_Number_Invariants` is banned here for exactly that reason — its `Hi` endpoint is unreachable
for a counter that never nears `MaxUint64`).

## Gap summary (surfaced for the follow-up encoding pass)

Real guarantees the code makes or relies on, with no current runtime check. None are coded in this
pass (the standing "enforce later"); they are the input to a reviewed encoding pass.

- `batch-size-bounded` — `1 <= len(Entries) <= Batch_Max`: spec'd (§Batch Cap), unit-tested,
  no sweep invariant.
- `batch-never-spans-reconfiguration` — spec'd (§Batch Reconfiguration Singleton), unit-tested, no
  sweep invariant.
- `view-change-suffix-bounded` — `View_Change_Suffix` is *used* (vsr.go:2452) but
  `len(Log_Suffix) <= View_Change_Suffix` is never asserted.
- `status-in-range` / `message-kind-in-range` — no defensive enum-membership check.
- `request-eventually-commits` / `view-change-eventually-completes` — **true liveness is untested**:
  `specification_test.go:55` checks only *aggregate* "commits make progress across seeds," not
  per-run eventual completion after faults cease.
- `primary-always-active` / `nonce-stale-rejection` — behaviourally enforced (`partial`), not
  asserted as named invariants.

---

## A. Log & state integrity

Structural invariants of one replica's log/commit/checkpoint, plus the cardinal cluster guarantee
that committed data is never lost or contradicted.

### op-equals-log-extent — Op is exactly the log extent
*Safety · Tier-1 vsr.go:1451 · enforced-inline*

**Property** — a replica's `Op` always equals `Log_Start + len(Log)`.
**Invariant** — `invariant.Always(int(Op) == int(Log_Start)+len(Log))`; a structural identity that
must hold on every step, so `Always` is the only fit.
**Angle** — truncation on a behind-view drop, compaction, and wholesale adoption all rewrite
`Log`/`Log_Start` together; a reorder interleaving them could desync the two.
**Why** — the offset index `Log[op-Log_Start-1]` reads garbage if this breaks (paper §5.1;
SPECIFICATION §Offset Index).

### commit-never-exceeds-op — Commit ≤ Op
Safety · Tier-1 vsr.go:1453 + Tier-2 simulation_test.go:1229 · enforced-inline +
enforced-simulator

**Property** — a replica's commit number never exceeds its op number.
**Invariant** — `invariant.Always(Commit <= Op)`; plus a cluster oracle (`commit %d exceeds op %d`).
**Angle** — adopting a peer's state that is *behind* this replica's commit (Bug 9's commit-past-op
face) would violate it.
**Why** — a commit past op means committing an op the replica does not hold.

### last-normal-view-bounded — Last_Normal_View ≤ View
*Safety · Tier-1 vsr.go:1454 · enforced-inline*

**Property** — the last view a replica was normal in never exceeds its current view.
**Invariant** — `invariant.Always(Last_Normal_View <= View)`.
**Angle** — view-change selection ranks by `(Last_Normal_View, Op)`; a stale higher value corrupts
selection.
**Why** — correct log selection across a view change (paper §5.3).

### log-start-below-checkpoint — Log_Start ≤ Checkpoint_Op
*Safety · Tier-1 vsr.go:1465 · enforced-inline (regression: Bug 7)*

**Property** — `Log_Start` never sits above `Checkpoint_Op` (ops between would be in neither log nor
checkpoint).
**Invariant** — `invariant.Always(Log_Start == 0 || Log_Start <= Checkpoint_Op)`.
**Angle** — a deferred fetch whose suffix began above the primary's checkpoint left this gap
(Bug 7).
**Why** — execution otherwise walks into a garbage-collected op.

### agreement — no two replicas commit different values at one op
Safety · Tier-2 simulation_test.go:1642/1671 · enforced-simulator (regression: Bugs 1, 5, 9, 11,
14)

**Property** — two replicas never hold different committed commands at the same op-number.
**Invariant** — simulator oracle `simulator_check_agreement` / `_pair` (`op %d diverges`). The
cardinal consensus invariant.
**Angle** — the whole point of the VOPR: drop/delay/reorder/partition plus crash during view change
or reconfiguration.
**Why** — divergent commits are consensus failure (paper §8).

### committed-op-never-lost — durability across view change / epoch
*Safety · Tier-1 vsr.go:2613 + Tier-2 agreement oracle · enforced (regression: Bugs 6, 7, 9)*

**Property** — an op committed in any view survives every later view change, state transfer, and
epoch boundary.
**Invariant** — implied by `agreement` plus the install-completeness `Always(Op >= selected_op)`.
**Angle** — a too-short Do_View_Change suffix, a re-run deferred install, or an epoch redirect could
drop it (Bugs 6, 7, 9).
**Why** — "acknowledged writes survive failover," the headline guarantee (SPECIFICATION §Commit
Preservation).

## B. Consensus & single-primary

### single-primary-per-view — at most one acting primary
*Safety · Tier-2 simulation_test.go:1602 · enforced-simulator*

**Property** — at most one replica acts as primary for a given (epoch, view).
**Invariant** — simulator oracle `simulator_check_single_primary`.
**Angle** — a partition healing mid-view-change, or a stale leader still serving as a new one
emerges.
**Why** — two primaries means split-brain and divergent appends.

### commit-behind-quorum — commit needs f+1 distinct acks
*Safety · Tier-1 vsr.go:2128 · enforced-inline (regression: Bug 11)*

**Property** — an op commits only behind `Prepare_Ok` from a quorum of distinct replicas (f+1,
counting the primary).
**Invariant** — `invariant.Always(distinct_count >= replica_quorum(replica))`.
**Angle** — a matchIndex-style model over-counted state-transferred ops a backup never acked
(Bug 11).
**Why** — quorum intersection is what makes a committed op durable across a view change.

### primary-always-active — the primary is never a standby
*Safety · Tier-1 (structural) · partial*

**Property** — the primary of a view is always a voting active replica, never a standby.
**Invariant** — would be `invariant.Always(primary_index < Active_Count)`. Structurally true:
`replica_primary_identifier` computes `View mod Active_Count`.
**Angle** — a reconfiguration changing `Active_Count` concurrent with a view change.
**Why** — a standby holds no application state and cannot serve (§6.1; SPECIFICATION §Primary Always
Active).
**Open Questions** — could a mid-reconfiguration `Active_Count` change transiently select a standby
before the role split settles? `(partial: structurally excluded by replica_primary_identifier; not
independently asserted)`

## C. Exactly-once & client semantics

### exactly-once-execution — one request executes once cluster-wide
*Safety · Tier-2 simulation_test.go:1126 · enforced-simulator (regression: Bugs 2, 4, 9, 10)*

**Property** — a single client request is executed at most once across the whole cluster.
**Invariant** — simulator op-level oracle (`executed command %q twice`).
**Angle** — an in-flight request surviving a view change; executing a lower op overwriting a higher
in-flight record; a pre-step round completing after its ordering point; an epoch redirect
re-executing the prefix.
**Why** — double execution is a double side effect (paper §4.1, §4.5).

### reply-matches-reference — replies are linearizable
*Safety · Tier-2 simulation_test.go:1142/1274 · enforced-simulator (regression: Bug 3)*

**Property** — a client reply equals the reference state machine's result for that op applied in
commit order.
**Invariant** — simulator reference oracle (`reply ... reference op ...`) plus
`simulator_check_accumulator`.
**Angle** — a replica that recomputed a non-deterministic value, or adopted a committed prefix
without executing it (Bug 3), diverges from the reference.
**Why** — linearizability: clients observe results consistent with one total order.
**Open Questions** — the oracle checks result-equals-reference in commit order; does it also enforce
*real-time* ordering (a request completing before another sees its effect)? `(partial: commit-order
linearizability confirmed; real-time edge not separately checked)`

### request-number-monotonic — accepted requests strictly advance
*Safety · Tier-1 vsr.go:1515 · enforced-inline*

**Property** — a request the primary accepts strictly advances that client's recorded
request-number.
**Invariant** — `invariant.Always(is_new_request)`.
**Angle** — a duplicated/reordered client retry must not be appended as a fresh op.
**Why** — the foundation of dedup and exactly-once (paper §4.5).

### dedup-duplicate-and-in-flight — retries are not re-appended
*Safety · Tier-2 coverage T10/T11 + exactly-once oracle · enforced-simulator (regression: Bug 2)*

**Property** — a duplicate (already-executed) request re-sends the cached reply; an in-flight or
stale request is dropped; neither is re-appended.
**Invariant** — behavioural (`replica_receive_request` + `replica_rebuild_client_table`); witnessed
by coverage axes `sim.result.cached_replies` and `sim.client.unanswered`.
**Angle** — a retry arriving after a view change whose new primary rebuilt its client-table (Bug 2).
**Why** — a lost-reply client (§4.5) must get its answer without re-execution.

## D. Recovery & state transfer

### recovery-quiescence — a recovering replica never view-changes
*Safety · Tier-1 vsr.go:2866 · enforced-inline (regression: Bug 1; seed 1 permanent)*

**Property** — a replica in `Status_Recovery` never enters or participates in a view change.
**Invariant** — `invariant.Always(Status != Status_Recovery)` in `replica_start_view_change`.
**Angle** — a crashed replica with a wiped log, dragged into a view change, wins the merge with its
empty log (Bug 1).
**Why** — an empty log winning a merge drops a committed entry (paper §4.3).

### recovery-behind-quorum — recovery needs a quorum incl. authority
*Safety · Tier-1 vsr.go:2809 · enforced-inline (regression: Bug 14)*

**Property** — recovery completes only behind `Recovery_Response` from a quorum including the
primary
of the highest reported view, with the authority selected over the active count.
**Invariant** — `invariant.Always(response_count >= replica_quorum)`; authority index
`highest mod Active_Count`.
**Angle** — with standbys, an inline `mod len(Configuration)` picked the wrong authority (Bug 14).
**Why** — adopting the wrong member's log rejoins with a divergent log (paper §4.3).

### adopted-state-reflects-commit — no stale state machine after adoption
*Safety · Tier-2 simulation_test.go:1274 · enforced-simulator (regression: Bug 3)*

**Property** — after any wholesale log adoption (view install, Start_View, recovery rejoin, state
transfer), the application state reflects exactly `Commit`.
**Invariant** — simulator accumulator oracle (`accumulator ... at commit C != reference`).
**Angle** — any adoption path that set `Commit` without executing/restoring the prefix (Bug 3).
**Why** — a replica claiming commit C with stale state violates linearizability the moment it serves
a read.

### nonce-stale-rejection — a delayed response can't complete a fresh attempt
*Safety · Tier-1 (behavioural guard) · partial*

**Property** — a `Recovery_Response` whose nonce differs from the in-flight nonce is ignored.
**Invariant** — would be `invariant.Always(response.Nonce == replica.Nonce)` at the accept point;
currently a behavioural guard.
**Angle** — a duplicated/delayed response from an earlier recovery attempt arriving during a new
one.
**Why** — a stale response completing a recovery installs an out-of-date log (paper §4.3;
SPECIFICATION §Stale Rejection).

## E. View change & reconfiguration lifecycle

### view-install-reaches-selected-op — install is never short
*Safety · Tier-1 vsr.go:2613 · enforced-inline (regression: Bug 6; seed 160)*

**Property** — the log the new primary installs reaches at least the selected reporter's op.
**Invariant** — `invariant.Always(Op >= selected_op)`.
**Angle** — a late `Do_View_Change` re-running selection against a log a deferred fetch had
shortened
(Bug 6).
**Why** — a short install drops a committed op (paper §5.3).

### view-monotonic — a view change advances the view
*Safety · Tier-1 vsr.go:2868 · enforced-inline*

**Property** — a view change always moves to a strictly higher view.
**Invariant** — `invariant.Always(view > replica.View)` in `replica_start_view_change`.
**Angle** — a replayed/duplicated Start_View_Change for an old view.
**Why** — view numbers must be monotone for `View mod Active_Count` to name a stable primary.

### epoch-dominates-view — a stale-epoch message is never processed normally
*Safety · Tier-1 vsr.go:1114 · enforced-inline*

**Property** — a message below the replica's epoch is never processed as a normal-view message.
**Invariant** — `invariant.Dot_Product` of two `Sometimes` (epoch `<` vs `==`) plus
`invariant.Impossible(stale ∧ processed-normally)`.
**Angle** — a reconfiguration overlapping a view change or recovery (coverage T21/T22).
**Why** — processing a stale-epoch message normally re-admits a replaced member (paper §7.2).
**Reachability** — both `Sometimes` branches witnessed: seeds deliver some below-epoch (redirected)
and some equal-epoch messages.

### reconfiguration-never-executed — a Reconfiguration is never an up-call
*Safety · Tier-1 vsr.go:2208 · enforced-inline*

**Property** — a Reconfiguration log entry is never passed to `State_Machine.Execute`.
**Invariant** — `invariant.Always(!log_entry_is_reconfiguration(entry))` in `replica_apply_commit`.
**Angle** — the commit walk crossing the epoch's last op.
**Why** — a reconfiguration is control-plane, not application data (paper §7.1).

### epoch-iff-reconfiguration-op-executed — epoch advances only by executing its reconfig op
*Safety · Tier-2 agreement + exactly-once oracles · enforced-simulator (Bug 19; seeds 222/415/64)*

**Property** — a replica reaches epoch N+1 only by EXECUTING the reconfiguration op that created it,
which sets `Epoch_Start_Op` to that op; it never advances the epoch by copying a number from a
message. So a replica normal in epoch N+1 holds N+1's reconfiguration op and the committed prefix
below it, and never re-uses an op-number the old epoch committed.
**Invariant** — `replica_execute_reconfiguration` triggers the handoff only when the reconfig
changes the configuration (`configurations_equal`); the agreement and exactly-once oracles witness
no divergence or duplicate across the boundary.
**Angle** — recovery or redirect jumping the epoch with a stale `Epoch_Start_Op`, then the new
primary re-using op-numbers (Bug 19).
**Why** — op-numbers must stay globally monotonic through a reconfiguration (paper §7.3); a stale
epoch-start op breaks that and forks the log.

### old-group-no-commit-after-epoch — replaced group can't commit a conflicting op
*Safety · Tier-2 agreement oracle · enforced-simulator (regression: Bug 9; seeds 6605/7683)*

**Property** — after the epoch advances, the old group cannot commit a conflicting op at the
reconfiguration's op-number.
**Invariant** — the cluster agreement oracle, on reconfiguration seeds.
**Angle** — a redirected member rejoining the stale group and being counted in its quorum (Bug 9's
agreement face).
**Why** — two committed values at the reconfiguration op is an agreement violation across the
boundary.

### handoff-state-only-in-transition — Old_Configuration is bounded to handoff
*Safety · Tier-1 vsr.go:1463 · enforced-inline (regression: Bug 8; seed 5449)*

**Property** — a non-empty `Old_Configuration` exists only while status is Transition or Shutdown.
**Invariant** — `invariant.Always(len(Old_Configuration) == 0 || handoff)`.
**Angle** — a transitioning replica pulled into a view change kept stale handoff state (Bug 8).
**Why** — carrying handoff bookkeeping with no handoff underway is an undefined §7 state.

### old-group-shutdown-after-quorum — replaced replica waits for f'+1
*Safety · Tier-2 coverage T5 · enforced-simulator (coverage); threshold behavioural*

**Property** — a replaced replica enters Shutdown only after holding f'+1 `Epoch_Started` from the
new group.
**Invariant** — behavioural (`replica_complete_epoch`); witnessed by coverage
`sim.replica.status_shutdown`.
**Angle** — the new group not yet caught up when the old group tries to retire.
**Why** — shutting down too early strands the new group without a state-transfer source (paper
§7.1.2).
**Open Questions** — is the f'+1 threshold asserted anywhere, or only the *reachability* of
Shutdown?
`(partial: reachability witnessed; threshold not asserted)`

## F. Prediction / non-determinism (§4.4)

### prediction-copied-verbatim — a backup never recomputes a prediction
*Safety · Tier-1 vsr.go:1787 · enforced-inline*

**Property** — a backup applying a Prepare adopts the entry's prediction unchanged.
**Invariant** — `invariant.Always(appended_matches)` in `replica_receive_prepare`.
**Angle** — any path where a backup might recompute rather than copy.
**Why** — if backups recomputed a clock/random-derived value they would diverge (paper §4.4).

### prediction-determinism — every replica executes the identical predetermined value
*Safety · Tier-2 reference oracle + coverage T13/T14 · enforced-simulator*

**Property** — every replica executes an op against a byte-identical prediction, yielding identical
results.
**Invariant** — the simulator accumulator/result oracles; coverage `sim.result.predictions_stamped`
(T13) and `pre_step_rounds` (T14).
**Angle** — replicas execute at different wall-times; a clock-derived predictor would diverge
without
predetermination.
**Why** — determinism of the replicated state machine (paper §4.4).
**Open Questions** — `pre_step_rounds > 0` fires only on odd (Combine-configured) seeds; is its
false
branch witnessed by even seeds in the same run? `(partial: both-branch witnessing assumed from the
seed sweep; verify against Run_Test_Main output)`

## G. Checkpoint / compaction (§5.1, §5.2)

### checkpoint-agreement — same op ⇒ same checkpoint state
*Safety · Tier-2 simulation_test.go:1245 · enforced-simulator (regression: Bugs 5, 7, 12)*

**Property** — two replicas at the same `Checkpoint_Op` hold byte-identical `Checkpoint_State`.
**Invariant** — simulator oracle `simulator_check_checkpoint_agreement`
(`checkpoint op %d diverges`).
**Angle** — a commit jumping past an interval boundary in one step (state-transfer catch-up); a
standby snapshotting empty state (Bugs 5, 12).
**Why** — a divergent checkpoint corrupts any replica that later fetches it.

### checkpoint-captures-exact-op — snapshot is taken at exactly its labelled op
*Safety · Tier-1 vsr.go:2314/2315 · enforced-inline (regression: Bug 5)*

**Property** — a checkpoint's op equals `Executed` and is ≤ `Commit` when the snapshot is captured.
**Invariant** — `invariant.Always(op <= Commit)` and `invariant.Always(op == Executed)` in
`replica_take_checkpoint`.
**Angle** — snapshotting after the execution loop captured a later op than the label (Bug 5).
**Why** — a snapshot labelled op 4 holding op 6's state diverges across replicas.

## H. Batching & standbys (§6)

### batch-acked-per-op — a backup is counted only for ops it acked
*Safety · Tier-2 oracles · enforced-simulator (regression: Bug 11)*

**Property** — a backup is counted toward an op's commit quorum only via a `Prepare_Ok` it sent for
that exact op.
**Invariant** — the per-op tally `Prepare_Ok_From[op][backup]`; witnessed by the
agreement/linearizability oracles.
**Angle** — a batched Prepare acked op-by-op vs. a state-transferred op that was never acked
(Bug 11).
**Why** — inferring acks a backup never gave commits an op the group may not hold.

### batch-size-bounded — a Prepare carries 1..Batch_Max entries
*Safety · Tier-1 (proposed) · GAP*

**Property** — a Prepare's `Entries` length is between 1 and `Batch_Max`.
**Invariant** — would be `invariant.Always(len(Entries) >= 1 && len(Entries) <= Batch_Max)`, a
reach-only Always (no boundary, so no unreachable endpoint).
**Angle** — buffer flush at the cap vs. at round-commit.
**Why** — an unbounded batch defeats the message-size bound (paper §6.2; SPECIFICATION §Batch Cap).
**Status note** — spec'd and unit-tested (`Test_Batching_Batch_Cap`), no sweep invariant.

### batch-never-spans-reconfiguration — a Reconfiguration commits alone
*Safety · Tier-1 (proposed) · GAP*

**Property** — a batch never mixes a Reconfiguration entry with others; the Reconfiguration is the
epoch's last op, appended alone.
**Invariant** — would be `invariant.Always(len(Entries) == 1 || no Entries entry is a
Reconfiguration)`.
**Angle** — a buffered batch flushing as a Reconfiguration arrives.
**Why** — a reconfiguration inside a batch breaks the §7 "last op of the epoch" invariant.
**Status note** — spec'd and unit-tested (`Test_Batching_Batch_Reconfiguration_Singleton`), no sweep
invariant.

### standby-never-executes — a standby runs no service code
*Safety · Tier-2 simulation_test.go:1119 · enforced-simulator (regression: Bugs 12, 13)*

**Property** — a standby makes no `Execute` up-call, caches no result, emits no Reply, and takes no
checkpoint of its own.
**Invariant** — simulator oracle (`standby replica %d executed command %q`) plus the
accumulator-check exemption.
**Angle** — a standby that self-checkpointed empty state corrupted fetchers (Bug 12); a promoted
standby served with no state (Bug 13).
**Why** — standbys hold no application state by design (§6.1; TigerBeetle non-voting standby).

### standby-promotion-materializes-state — a promoted standby builds state before serving
Safety · Tier-2 accumulator oracle + Test_Standby_Promotion · enforced-simulator (regression:
Bug 13)

**Property** — a standby moved into the voting set materializes its application state (restore
carried
checkpoint, replay suffix) before serving.
**Invariant** — the accumulator oracle plus the `Test_Standby_Promotion` unit test.
**Angle** — a reconfiguration growing the group and promoting a standby (Bug 13).
**Why** — serving reads/checkpoints from empty state diverges (TigerBeetle state sync).

## I. Boundary & enum validity (Tier-1)

### status-in-range / message-kind-in-range — defined enum members only
*Safety · Tier-1 (proposed) · GAP*

**Property** — a `Status` / `Message_Kind` value is always a defined enum member.
**Invariant** — would be `invariant.Always(s <= Status_Shutdown)` /
`invariant.Always(k <= Message_Kind_Check_Epoch)`, reach-only Always — deliberately not a
`Distinct_Boundary`, whose full-type-range form would gap.
**Angle** — a corrupted/uninitialised field surfacing during a malformed transition.
**Why** — an out-of-range discriminant routes a message to the wrong handler.

### view-change-suffix-bounded — Do_View_Change suffix is bounded
*Safety · Tier-1 (proposed) · GAP*

**Property** — a Do_View_Change's `Log_Suffix` never exceeds `View_Change_Suffix` entries.
**Invariant** — would be `invariant.Always(len(Log_Suffix) <= View_Change_Suffix)`.
**Angle** — a reporter with a long log emitting its suffix.
**Why** — an unbounded suffix defeats §5.3's bounded-report optimization.
**Status note** — `View_Change_Suffix` is used at vsr.go:2452, never asserted.

## J. Liveness (progress)

### commits-make-progress — the cluster commits across the sweep
*Liveness · Tier-2 specification_test.go:55 · enforced-simulator (aggregate)*

**Property** — across the seed sweep, committed ops advance (the cluster is not globally stuck).
**Invariant** — `Test_Cluster_Liveness` asserts aggregate progress (`expected commits to make
progress across seeds`).
**Angle** — faults injected throughout; checked in aggregate because one adversarial 3-node seed can
legitimately stall.
**Why** — a protocol that never commits is useless.

### request-eventually-commits — outstanding requests complete after quiescence
*Liveness · Tier-2 (proposed) · GAP*

**Property** — after faults cease for a tail window, every outstanding client request eventually
commits and is answered.
**Invariant** — would be an end-of-run check: with fault injection disabled for the last N ticks,
`Clients[*].Unanswered` drains to zero.
**Angle** — the decisive liveness test — does the cluster recover progress once the network
heals, or
can it wedge?
**Why** — the real liveness guarantee; aggregate-progress does not catch a single-seed permanent
wedge.
**Open Questions** — does the simulator have a notion of a fault-free tail window to make
"eventually"
checkable deterministically? `(partial: faults are PRNG-driven per tick; a tail window needs the
harness to stop injecting — a design question for the encoding pass)`

### view-change-eventually-completes — a primary re-emerges after quiescence
*Liveness · Tier-2 (proposed) · GAP*

**Property** — after faults cease, a started view change completes and the cluster returns to a
single
normal primary.
**Invariant** — would be an end-of-run check: in the quiescent tail, exactly one replica is a
`Status_Normal` primary and the rest converge.
**Angle** — repeated view changes / dueling candidates that fail to settle.
**Why** — a cluster stuck in perpetual view change serves nothing.

---

## Assumptions

- The simulator's reference model is the linearizability ground truth; "linearizable" here means
  result-equals-reference applied in commit order (real-time ordering is only partially checked).
- `shared/invariant`'s coverage model is stricter than Antithesis (every `Always` reached, every
  `Sometimes` both ways), so reach-only `Always` is preferred for membership/bound properties.
- Crash-only fault model: Byzantine/corruption faults are out of scope (the core is not BFT and has
  no checksums — that layer would live in the caller's transport/disk, as in TigerBeetle).

## Open Questions (catalog-wide)

- The liveness gaps need a simulator notion of a fault-free tail window to be deterministically
  checkable — a design decision for the encoding pass, not just an assertion to add.
- Real-time linearizability (vs. commit-order) is only partially checked; is the stronger property
  worth a dedicated oracle, or is commit-order sufficient for this core's contract?
  `(needs human input)`
- Whether the Tier-1 GAP properties should be encoded as inline `Always` axes or as selective
  `_Invariants` bundles on the owning aggregates is deferred to the reviewed encoding pass.
