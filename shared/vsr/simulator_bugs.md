# Bugs the simulator found

Every bug logged here was found by the deterministic, seeded simulator in `shared/vsr/simulation`
— each a real protocol bug in `vsr.go`, not a test artifact. They are recorded because the
discovery *method* repeats, and because a bug is a pure function of its seed, so every entry below
reproduces exactly on demand.

Why the simulator catches them: the core is a pure deterministic step function (no goroutines,
I/O, or ambient clock — the linter enforces it); one seeded PRNG drives every fault (drop, delay,
duplicate, reorder, partition, crash-restart, client retry and recovery); `invariant.Sometimes`
coverage forces the buggy paths (view change, recovery, state transfer) to actually run; and the
cluster-wide safety assertions run after every single delivery, pinning a violation to the step
that caused it. Each fix changes `vsr.go`, never the assertion.

## Bug 1 — committed entry lost across a view change (agreement)

- **Symptom** (seed 1, N=3): `op 2 diverges: 1="cmd-22" 2="cmd-51"` — two replicas committed
  different commands at the same op, the cardinal consensus violation.
- **Root cause**: `Replica_Recover` wipes a crashed replica's log to empty, but a recovering
  replica still processed `Start_View_Change` — so it was dragged into a view change and its empty
  log won the merge, dropping a committed entry, which was then re-committed under a new command.
- **Fix**: a recovering replica is quiescent — it ignores `Start_View_Change`/`Start_View` and
  rejoins only via the nonce-protected `Recovery_Response` (VSR §4.3, §8), asserted locally by
  `Always(status != recovering)` in `replica_start_view_change`.
- **Guard**: `Test_Recovery_Quiescence`; seed 1 is a permanent regression seed.

## Bug 2 — in-flight request re-executed across a view change (exactly-once)

- **Symptom** (seed 70, N=3): client request `req-4` committed at ops 10, 11, and 16 and executed
  twice — a violation of exactly-once (one client request must execute once cluster-wide).
- **Root cause**: a request appended on one primary but not yet executed when a view change
  occurred survived into the new view's log. The new primary's client-table had only ever recorded
  *executed* requests, so it did not recognize the surviving entry as a duplicate; when the client
  retried, the new primary appended it again at a fresh op.
- **Fix**: `replica_rebuild_client_table` reconciles the client-table with the adopted log on every
  wholesale log replacement (view-change install, `Start_View` adoption, state-transfer apply,
  recovery rejoin), recording each client's highest request-number present, so a retry of an
  in-flight request is deduped instead of re-appended.
- **Guard**: the simulator's `### Exactly Once` check, run after every delivery across all seeds.

## Bug 3 — adopted committed prefix left the state machine stale (linearizability)

- **Symptom** (seed 120, N=5): `replica 4 accumulator 0 at commit 6 != reference c35bb...` — a
  replica reported commit 6 while its application accumulator was still empty, so its state machine
  did not reflect the prefix it claimed committed. Surfaced once the simulator's state machine
  became stateful (a rolling hash folded over every executed op) and the accumulator was checked
  against the reference for the first `Commit` ops after every delivery.
- **Root cause**: every path that adopted a log wholesale — view-change install, `Start_View`
  adoption, recovery rejoin, and state-transfer apply — set `Commit` directly without executing the
  newly-committed ops. With a `(command, prediction)`-pure state machine this was invisible (no
  state to go stale), but a stateful application would diverge: its accumulator never folded the
  adopted prefix, so it no longer matched the committed log.
- **Fix**: a single execution spine, `replica_execute_to_commit`, drives a separate `Executed`
  marker up to `Commit` on every commit advance, normal or adopted; the wholesale-adoption spine
  `replica_adopt_log` restores from the carrier's checkpoint when it is behind it, then executes the
  committed ops after it, so after any adoption the application state reflects exactly `Commit`.
- **Guard**: the simulator's `### Cluster` accumulator check (replica accumulator at commit C equals
  the reference of the first C committed ops) and checkpoint-agreement check, run after every
  delivery; seed 120 reproduces it with execution-on-adopt removed.

## Bug 4 — executing a lower op forgot a higher in-flight request (exactly-once)

- **Symptom** (seed 63, N=3): `replica 1 executed command "client-1000-req-5" twice` — client
  1000's request 5 committed at op 15 (view 4) and again at op 16 (view 7), executing twice.
- **Root cause**: at a view-change install the client-table rebuild recorded the highest request in
  the adopted log (req 5 at the uncommitted op 15), but the install then executed the committed
  prefix, and executing a *lower* op for the same client (req 4 at op 11) overwrote the table back
  to req 4 — the execution path set the record to the executed entry's number unconditionally. The
  new primary thus no longer knew req 5 was already in its log, so the client's retry of req 5
  passed the dedup check and was appended again at op 16.
- **Fix**: `replica_execute_to_commit` never lowers a client's recorded request-number: when the
  executed entry's number is below the table's, the higher (in-flight) number is kept with
  `Executed` cleared, while the reply still carries the op actually executed. The highest request a
  replica holds — committed or in-flight — stays recorded, so a retry is always deduped.
- **Guard**: the simulator's `### Exactly Once` op-level double-execution check; seed 63 is a
  permanent regression seed.

## Bug 5 — checkpoint captured the wrong op's state (agreement)

- **Symptom** (seed 105, N=3): `checkpoint at op 4 diverges: replica 0=0126... replica 2=6c12...`
  — two replicas checkpointed op 4 but stored different application state for it.
- **Root cause**: the checkpoint was taken after the execution loop, capturing the state machine as
  of `Commit`, but labelled with the highest interval *boundary* at or below `Commit`. When `Commit`
  jumped past a boundary in one step (a state-transfer catch-up), one replica's op-4 checkpoint held
  the state of op 4 while another's held the state of op 6, both filed under op 4.
- **Fix**: `replica_maybe_checkpoint` runs inside the execution loop and snapshots at the exact op a
  boundary falls on, while the application state reflects precisely that op; compaction is deferred
  to after the loop so advancing `Log_Start` cannot disturb the loop's indexing.
- **Guard**: the simulator's checkpoint-agreement check (two replicas at the same `Checkpoint_Op`
  must hold byte-identical `Checkpoint_State`), run after every delivery; seed 105 reproduces it
  with the after-loop snapshot restored.

## Bug 6 — a deferred view-change install re-ran on a late report (§8 safety)

- **Symptom** (seed 160, N=5): `Always(replica.Op >= selected_op)` failed in
  `replica_complete_install` — the new primary installed a log shorter than the selected reporter's
  op, the exact committed-op loss §5.3's suffix-only reports put at risk.
- **Root cause**: after the new primary deferred an install to fetch the selected log (§5.3), a
  later `Do_View_Change` re-reached quorum and re-ran `replica_install_new_view`. Selection re-read
  the now-stale self-report against a live log the deferred fetch had since shortened, so the
  reconstructed carrier installed fewer ops than the winner reported.
- **Fix**: `replica_receive_do_view_change` keeps tallying late reports but does not re-run
  selection while `View_Change_Deferred` is set; the in-flight fetch owns the install, and only its
  awaited `New_State` (or a fresh view change) completes or abandons it.
- **Guard**: the `Always(replica.Op >= selected_op)` install assertion plus the cluster-wide
  agreement and accumulator checks; seed 160 is a permanent regression seed.

## Bug 7 — a fetched install left the log starting above its checkpoint (agreement)

- **Symptom** (seed 160, N=5, surfaced once Bug 6 was fixed): `replica_log_entry` asserted
  `op > Log_Start` false while a backup adopted the new primary's `Start_View` — the backup
  restored to a checkpoint op but the `Start_View` suffix began above it, leaving a gap the backup
  could not fill and walking execution into a garbage-collected op.
- **Root cause**: the deferred fetch requested state from the new primary's *commit* number, so the
  adopted carrier's suffix began at commit+1 and set `Log_Start` to commit — above the primary's
  own (older) `Checkpoint_Op`. The `Start_View` the primary then shipped carried a suffix from that
  higher `Log_Start` but a checkpoint from the lower `Checkpoint_Op`, so a fresh backup had no
  source for the ops between them.
- **Fix**: `replica_begin_view_change_fetch` requests state from the new primary's `Log_Start`, not
  its commit, so the carrier's suffix stays aligned with a checkpoint the primary already holds (or,
  on the §5.2 gap, the reporter's checkpoint), keeping `Log_Start <= Checkpoint_Op` and the shipped
  `Start_View` self-consistent.
- **Guard**: the cluster-wide agreement and checkpoint-agreement checks plus the `replica_log_entry`
  bounds assertion, run after every delivery; seed 160 reproduces it with the commit-based fetch op
  restored.

## Bug 8 — a transitioning replica pulled into a view change kept its handoff state (§7 safety)

- **Symptom** (seed 5449, N=5): `Always(old_configuration_within_handoff)` failed in
  `replica_assert_safety` — replica 4 held a non-empty `Old_Configuration` while its status was
  `Status_View_Change`, the contradictory state of carrying epoch-handoff bookkeeping while no
  handoff is underway.
- **Root cause**: a replica being replaced (transitioning, serving state transfer to the new group)
  still processed `Start_View_Change`/`Start_View`. A higher-view `Start_View_Change` pulled it into
  `replica_start_view_change`, flipping its status to `Status_View_Change` without clearing the
  `Old_Configuration` the handoff had set — so it sat mid view change with stale handoff state, and
  (worse) a transitioning replica's view-change role has no defined meaning in §7.
- **Fix**: a transitioning replica is quiescent in the view-change protocol, mirroring the
  recovery-quiescence rule (Bug 1): `replica_receive_start_view_change` and
  `replica_receive_start_view` return early on `Status_Transition`
  (`replica_receive_do_view_change` already required
  `Status_View_Change`). It catches up only through the §7.1.1 epoch state-transfer path and
  rejoins normal operation through `replica_complete_epoch`, which clears `Old_Configuration`. And
  `Replica_Recover` now discards the handoff scratch (`Old_Configuration`, `Epoch_Start_Op`,
  `Epoch_Handoff_Due`, `Epoch_Started_From`) and shuts a recovering replica down when it is no
  longer a member of its configuration (§7.2), so a crash mid-handoff cannot resurrect stale
  handoff state.
- **Guard**: the `Always(old_configuration_within_handoff)` per-replica safety assertion, run after
  every step; seed 5449 reproduces it with the transitioning-quiescence guards removed.

## Bug 9 — epoch redirect re-executed (and could strand) committed ops

- **Symptom** (reconfiguration scan over thousands of seeds): three faces of one cause. Seed 1166
  `replica 1 executed command "client-1000-req-1" twice` — an exactly-once double-execution. Seed
  1045 `commit 2 exceeds op 1` — a replica left with commit past its op. Seeds 6605/7683
  `op 1 diverges: "reconfigure-1" vs "client-1000-req-1"` — two committed values at one op, an
  agreement violation; 7642 the matching double-execution. None fell in the committed seed ranges.
- **Root cause**: `replica_receive_new_epoch` — the redirect that pulls a replica stranded at a
  stale epoch forward (§7.2/§7.4) — wiped the replica to a full cold recovery (`Executed = 0`,
  `Op = 0`, `Log = nil`) and re-broadcast Recovery. But the replica had NOT lost state: a New_Epoch
  is a redirect, not a crash. Recovery then re-executed the already-executed committed prefix —
  invisible to a pure log, but a double-apply to a stateful state machine (the exactly-once face).
  An interim fix that instead fetched from a single peer was worse: that peer could be behind the
  replica's own commit, so adoption shortened the log below the commit number (`adopt_log` only
  raises commit, never lowers it — the commit-past-op face). And because the wiped replica re-joined
  through a path that did not guarantee an authority holding every committed op, the old group could
  commit a conflicting op at the reconfiguration's op-number (the agreement face).
- **Fix**: a redirected member re-learns the new epoch through the RECOVERY protocol — whose quorum
  guarantees the answering authority holds every committed op — while PRESERVING its committed,
  executed state: it does not wipe op, commit, log, checkpoint, or the Executed marker. Committed
  ops survive a reconfiguration, so its committed prefix is a valid prefix of the new epoch's log;
  recovery adoption restores only when behind and executes only beyond Executed, so the prefix is
  never re-executed (no double-apply) and never adopted from a peer behind it (no commit-past-op).
  A recovering replica stays quiescent (Bug 1), so a redirected member cannot vote in or be counted
  by the stale-epoch group, closing the agreement face.
- **Guard**: seeds 1045, 1166, 1841, 6605, 7683, 7642 are permanent regression seeds in
  `Test_Cluster_Reconfiguration`; the per-delivery agreement, exactly-once, and commit ≤ op checks
  run on every reconfiguration seed.

## Bug 10 — a pre-step prediction round completed after its request's ordering point

- **Symptom** (Stage 6a batching; the per-delivery exactly-once check on odd, pre-step seeds): seed
  93 `replica 0 executed command "client-1002-req-7" twice`, with the log showing the request at two
  consecutive ops straddling a committed `reconfigure-1` op; seed 13 likewise. A single client
  request committed at two op-numbers.
- **Root cause**: the §4.4 pre-step path opens a prediction round when a request arrives
  (`replica_begin_pre_step`) and appends the request only later, when f backups' predictions arrive
  (`replica_receive_predict_response`). The append at completion did not re-check that the request
  was still acceptable. While the round was open, the epoch's reconfiguration was appended (closing
  the primary to new requests, §7.1) and the epoch advanced. The round then completed and appended
  the request anyway — after the reconfiguration, in the wrong epoch — and the client, retried,
  committed it again. The synchronous local-predict path checks `replica_open_to_requests` and the
  client-table before it appends; the deferred pre-step path skipped that check.
- **Fix**: `replica_receive_predict_response` re-validates at completion — it deletes the round,
  then drops it without appending if the primary is no longer open to requests or the request-number
  is no longer fresh — so a round that opened optimistically cannot append a request whose ordering
  point has passed. The client retries and the new primary runs it afresh.
- **Guard**: the per-delivery exactly-once and linearizability checks on every odd (pre-step) seed;
  a 1200-seed scan is clean.

## Bug 11 — a matchIndex commit model counted ops a backup never acknowledged

- **Symptom** (Stage 6a batching): seed 180 (linearizability) and seed 210 (reconfiguration)
  reported one client request mapping to two different results; a request committed at op 40 (and
  replied) then reappeared at op 43 after a view change — a committed op was not preserved.
- **Root cause**: batching needs a backup to acknowledge a multi-entry Prepare in one round, which
  first prompted a Raft-style matchIndex model — record each backup's highest acknowledged op and
  credit it for every op through that high. It over-commits: a backup that acquires ops by STATE
  TRANSFER never acknowledges them, yet once it acks any later op its recorded high credits the
  transferred ops too. Those ops are not durably anchored on that backup — it can truncate a
  state-transferred, never-locally-committed op on a later view change — so a quorum resting on the
  inferred credit could commit an op the group does not actually hold, and a subsequent view change
  drops it.
- **Fix**: keep the explicit per-op acknowledgement tally (`Prepare_Ok_From[op][backup]`); a backup
  is counted toward an op's quorum only by a Prepare_Ok it sent for THAT op, which it sends only
  after appending the op from a Prepare. A batched Prepare is acknowledged op by op — the backup
  emits one Prepare_Ok per newly-appended entry — so batching is supported without inferring
  acknowledgements a backup never gave, and a state-transferred op is never counted until the backup
  re-receives it via a Prepare.
- **Guard**: the per-delivery agreement, exactly-once, and linearizability checks across the seed
  sweep; a 1200-seed scan is clean. The batch path is exercised every run (`Batch_Max` > 1 in the
  simulator, with a coverage assertion that a multi-entry Prepare is flushed).

## Bug 12 — a standby checkpointed empty application state

- **Symptom** (Stage 6b standbys): the checkpoint-agreement check failed across many seeds, e.g.
  `checkpoint op 4 diverges: r0=ec72... r2=0000000000000000` — a standby holding a checkpoint whose
  state blob was all zeros, the snapshot of an empty accumulator, where the active replicas held the
  real state. A replica that then state-transferred that checkpoint adopted the empty state and its
  own accumulator diverged.
- **Root cause**: a standby never executes (§6.1), so its application state is empty, but
  `replica_execute_reconfiguration` still called `replica_maybe_checkpoint`, and the standby's
  injected `Snapshot` happily captured its empty accumulator — writing a bogus checkpoint at the
  reconfiguration's op that disagreed with the active replicas and corrupted any replica that later
  fetched it.
- **Fix**: `replica_maybe_checkpoint` returns immediately for a standby — a replica with no
  application state has nothing to snapshot. A standby still bounds its log: it compacts by adopting
  a REAL checkpoint from an active replica during state transfer, never by taking one of its own.
- **Guard**: the per-delivery checkpoint-agreement check (equal Checkpoint_Op ⇒ equal
  Checkpoint_State) over the standby-enabled sweep; a 1200-seed scan is clean.

## Bug 13 — a promoted standby served with no application state

- **Symptom** (Stage 6b standbys): the accumulator oracle failed, e.g. `replica 2 accumulator 0 at
  commit 9 != reference c584...` — a replica reporting its commit number advanced but its
  application state empty, after a reconfiguration grew the group and moved it into the voting set.
- **Root cause**: a standby advances its Executed marker without executing (so it does not re-walk
  the log every step), so on promotion to active it believed it had executed up to its commit when
  it had built no application state at all. It then served reads and took checkpoints from an empty
  state.
- **Fix**: following TigerBeetle's state sync — acquire the materialized checkpoint, replay only the
  un-checkpointed suffix, never re-execute the whole log — a promoted standby materializes its state
  once when it returns to normal: `replica_materialize_application_state` restores the checkpoint it
  carries (a real snapshot adopted from an active replica) and replays the committed ops after it. A
  one-shot `Standby_Promotion_Due` flag, set at the role flip and consumed at the return to normal,
  makes the rebuild fire exactly once across every epoch-completion path.
- **Guard**: the per-delivery accumulator oracle (each active replica's live state equals the
  reference at its commit) plus a `Test_Standby_Promotion` unit test; the 1200-seed scan is clean.

## Bug 14 — recovery picked the authority by the wrong group size

- **Symptom** (Stage 6b standbys, a 1200-seed scan): agreement divergence across an epoch boundary,
  e.g. seed 356 `op 1 diverges: r0="reconfigure-1" r1="client-1002-req-1"`, two active replicas
  disagreeing on a committed op after a reconfiguration where standbys were present.
- **Root cause**: a recovering replica picks its authority as the primary of the highest reported
  view, computed inline as `Configuration[highest mod len(Configuration)]`. With standbys the
  primary is `Configuration[View mod Active_Count]` — over the voting prefix, not the whole — so the
  recovering replica adopted the log of the WRONG member (a standby, or a different active), then
  rejoined with a divergent log that a later commit propagated.
- **Fix**: compute the authority index over the active count, `highest mod Active_Count`, the same
  voting-aware leader rule the rest of the core uses (replica_primary_identifier). With no standbys
  (Active_Count zero, the active count is the whole group) this is unchanged.
- **Guard**: the per-delivery agreement check over the standby-enabled, reconfiguring sweep; the
  1200-seed scan is clean. Standbys are exercised every run (the voting set is fixed at three, so a
  five-member configuration carries two standbys; a coverage axis witnesses a standby following).

## Bug 15 — a checkpoint restore left a stale Op, overwriting a committed op (agreement)

- **Symptom** (seed 1266, N=5; surfaced by a wider seed scan beyond the committed ranges):
  `op 17 diverges: 1="read-client-1000-req-6" 3="read-client-1001-req-6"` — two replicas committed
  different commands at op 17, an agreement violation. The committed 0–239 ranges never reached it;
  a 400–1399 scan did, so the bug existed in the shipped core, latent.
- **Root cause**: `replica_restore_checkpoint` set `Log_Start` and `Executed` to the checkpoint op
  but left `replica.Op` at its stale pre-restore value, and `replica_apply_new_state` nilled the log
  without fixing `Op` either — so after restoring a state-transfer checkpoint the replica violated
  `Op == Log_Start + len(Log)`. The following `replica_splice_suffix` read the stale higher `Op`,
  treated the suffix's first (committed) op as one it already held, dropped it, and shifted the
  remaining entry down one op — overwriting the committed op 17 with op 18's command. A later
  delivery committed past it, and two replicas disagreed on a committed op.
- **Fix**: `replica_restore_checkpoint` now leaves a self-consistent state — `Op = Log_Start =
  checkpoint_op`, `Log` empty — upholding `Op == Log_Start + len(Log)` itself; the redundant
  log-nil in `replica_apply_new_state` is removed. The splice then sees the true post-restore op and
  appends the whole suffix without dropping its committed head.
- **Guard**: seed 1266 is a permanent regression seed in `Test_Cluster_Agreement`; the per-delivery
  agreement and accumulator oracles, run after every delivery.

## Bug 16 — a deferred view-change fetch accepted a stale, non-attaching New_State (gap)

- **Symptom** (seed 1378, N=5, with per-replica clock skew enabled): `log entry op is above log
  start` (`Always`) tripped in `replica_log_entry`, reached from a deferred view-change fetch
  completing → `replica_adopt_log` → execution walking into a garbage-collected op. Skew-specific:
  the same seed without clock skew is clean; the timing skew produces is what delivers the stale
  answer at the wrong moment.
- **Root cause**: the deferred view-change fetch (§5.3) asks the reporter for state from the
  new primary's `Log_Start`, but `replica_complete_view_change_fetch` accepted ANY `New_State`
  matching `(from, view, op >= fetch_op)`. A stale answer to an earlier state-transfer `Get_State` —
  one this replica sent while a normal backup — satisfied that check. Its suffix attached ABOVE the
  new primary's commit (`carrier_start > Commit`) and carried no checkpoint, so `replica_adopt_log`
  took its behind-branch and restored from a non-existent checkpoint (`Checkpoint_Op` 0), leaving
  `Log_Start` above `Executed` — a checkpoint-to-log gap the commit walk then indexed into.
- **Fix**: `replica_complete_view_change_fetch` now ignores a `New_State` whose suffix sits above
  the committed prefix (`carrier_start > Commit`) unless it carries a checkpoint. A faithful answer
  to the `Log_Start` fetch attaches at or below the commit (or ships a checkpoint), so only stale or
  mismatched answers are dropped, keeping the fetch outstanding for the real one.
- **Guard**: seed 1378 (clock skew on) is a permanent regression seed in `Test_Cluster_Clock_Skew`;
  the `replica_log_entry` bounds assertion and the cluster agreement oracle under the clock-skew
  sweep.

## Bug 17 — a lost Prepare_Ok was never recovered (liveness)

- **Symptom**: the fault-free tail wedged with an op uncommitted under a stable primary, the
  client's retries dropped as in flight, no view change to rescue it.
- **Root cause**: the primary's heartbeat re-drives an uncommitted op to a backup that already holds
  it; `replica_receive_prepare` returned early without re-acking, so the dropped `Prepare_Ok` was
  never regenerated.
- **Fix**: `replica_receive_prepare` re-acks the still-uncommitted ops a re-driven Prepare carries
  (`replica_redriven_prepare`), re-acking ONLY entries it holds identically to the primary's and
  reconciling a divergent one rather than re-acking a stale copy.
- **Guard**: the fault-free-tail convergence oracle (`simulator_run_tail`).

## Bug 18 — the primary stayed closed to requests after a reconfiguration (liveness)

- **Symptom**: the cluster never accepted client requests again after any reconfiguration.
- **Root cause**: `replica_open_to_requests` returned false whenever the topmost log entry was a
  reconfiguration; after it committed and the epoch advanced, the new epoch's primary still saw it
  on top and could never append its first op.
- **Fix**: a topmost reconfiguration closes requests only while UNCOMMITTED; once committed the
  epoch has advanced, so `replica_open_to_requests` returns `Op <= Commit` for that case.
- **Guard**: `Test_Cluster_Reconfiguration` cleanup (reconfigurations complete, epoch advances) and
  the tail oracle.

## Bug 19 — op-number reuse across a reconfiguration via a stale epoch-start op

- **Symptom** (seeds 222 op 220, 415 op 141, 64 op 126): `op N diverges` and one client request
  `mapped to two results`.
- **Root cause**: a replica reached epoch N+1 WITHOUT executing N+1's reconfiguration op — recovery
  and redirect JUMPED `replica.Epoch = message.Epoch` and copied `Epoch_Start_Op` from a (possibly
  stale) message. With a stale `Epoch_Start_Op` the new epoch's primary re-used op-numbers the old
  epoch had committed: two values at one op (fork) and one request at two ops (exactly-once). This
  is a §7.3 violation: a replica must CATCH UP to the new epoch, and the epoch advances by
  EXECUTING the reconfiguration op.
- **Fix**: `replica_execute_reconfiguration` triggers the epoch handoff only when the
  reconfiguration CHANGES the configuration (`configurations_equal`). A replica catching up to or
  recovering into the epoch its own reconfiguration op created executes that op (same configuration)
  and learns its epoch-start op from it WITHOUT advancing again or overshooting — so
  `Epoch_Start_Op` always comes from the executed op, not a message, so op-numbers never reuse.
- **Rejected** (measured net-negative, do not retry): joint consensus (a Raft mechanism, not VSR);
  a recovery "most-committed authority"; `Start_View` op-rejects; commit clamps. Each traded a fork
  for a wedge or a `commit > op` panic.
- **Guard**: the agreement and exactly-once oracles across `Test_Cluster_Reconfiguration` /
  `_Agreement` / `_Clock_Skew`.

## Bug 20 — a client-table record outliving its log entry stranded or duplicated a request

- **Symptom**: a request `mapped to two results` (seed 78 op 184/186), or the fault-free tail wedged
  with one request unanswered though the quorum was healthy (seeds 20, 105) — the primary held a
  client-table record for the request but no log entry, so it neither re-accepted nor committed it.
- **Root cause**: the client-table record is written at flush, but a view change can drop the
  uncommitted entry while the record survives. (a) On a checkpoint restore the record was lost for
  COMPACTED requests, so a new primary re-appended an already-executed one. (b) The flush-written
  record made `replica_receive_request` and the deferred prediction round treat a dropped request as
  in flight, so it was never re-accepted.
- **Fix**: ship the client-table with the checkpoint (`Checkpoint_Client_Table`, merged in
  `replica_restore_checkpoint`) so compacted exactly-once records survive a restore; and gate the
  in-flight dedup on whether the request is actually in the log (`replica_request_logged`) in both
  `replica_receive_request` and `replica_pre_step_appendable`, re-accepting a request the log no
  longer holds. The prediction-round freshness check abandons only on a STRICTLY newer or
  already-executed request, never on the same not-yet-executed number.
- **Guard**: the exactly-once oracle and the tail convergence oracle across the sweep.

## Bug 21 — a reconfiguration-added or re-added replica was never brought up (liveness)

- **Symptom** (seeds 205, 218, 86): the tail wedged with a voting member of the new configuration
  never active — stuck at the old epoch, or shut down and unable to rejoin.
- **Root cause**: only the replicas being replaced re-drove `Start_Epoch` to new members, and they
  retire after f'+1 `Epoch_Started` (a quorum, satisfied by the CONTINUING members) before a newly
  added member caught up — leaving it stranded with nobody to prod it. A member that legitimately
  shut down when an earlier epoch dropped it could never rejoin when a later epoch re-added its
  identifier, because every message to a shut-down replica was dropped.
- **Fix**: the new primary re-drives `Start_Epoch` to any configured member not yet confirmed active
  in the epoch (`replica_primary_resend_start_epoch`, tracked by `Epoch_Up_From`), per §7.2; and the
  epoch-precedence gate admits a `Start_Epoch` that names a shut-down replica in its new
  configuration, reviving it as a fresh member via the adopt path. The harness models a re-added
  identifier as a fresh node (§7.5) by delivering that revival `Start_Epoch`.
- **Guard**: the tail convergence oracle, which requires every voting member active and normal.

## Bug 22 — a view-changing replica answered Get_State with its stale uncommitted suffix

- **Symptom** (seed 3470, skew off): op 109 committed as two commands — `reconfigure-2` on the
  cluster (which advanced to the new epoch) and `client-1002-req-49` on one replica stuck in the old
  epoch. Surfaced once the simulator's main stream moved to the unbiased `shared/prng`; it was the
  dominant agreement-fork class the weaker generator had hidden.
- **Root cause**: `replica_receive_get_state` answered any Get_State except from a recovering
  replica. A replica that had advanced its view-number for a view change but not yet adopted the new
  view's log still held its old uncommitted suffix; answering a catch-up Get_State, it shipped that
  stale suffix stamped with the NEW view. The requester — already normal in the new view — accepted
  it, kept op 109 = `client-1002-req-49` (which the view change was about to supersede with
  `reconfigure-2`), and then committed it on a bare Commit advertising commit 109. Two committed
  values at one op. This violates VSR-Revisited §5.2: "a replica responds to a GETSTATE message only
  if its status is normal."
- **Fix**: a `Status_View_Change` replica no longer answers a plain catch-up Get_State (§5.2). The
  §5.3 view-change merge fetch is the lone exception — the new primary fetching a selected
  reporter's reported Do_View_Change log — so a `View_Change_Fetch` flag on that Get_State lets a
  view-changing reporter answer it (and only it). A `Status_Transition` replica still answers normal
  catch-ups, so the §7.1.1 epoch catch-up is not starved into a wedge (a blanket Normal-only gate
  wedged the epoch handoff). Across 0–5000 × skew this took the sweep from 36 to 31 failures (forks
  11→6, no liveness regression).
  The epoch-handoff analogue (a `Status_Transition` replica leaking its uncommitted suffix during
  §7.1.1 catch-up) is closed the same way: a transitioning replica still answers state transfer
  (so peers catching up are not starved) but caps its answer at its commit, shipping only the
  committed prefix the catch-up needs — fixing seeds 248 and 3072.
- **Guard**: pinned seed 3470 in `Test_Cluster_Agreement`. Reverting the view-change gate in
  `replica_receive_get_state` makes it fork `op 109 diverges`; the fix makes it pass.
- **Residual**: seeds 2268, 460, 3678, 26 still fork — the deposed-primary-late-commit, the hardest
  class. A view change ABANDONS an uncommitted op (op 75 = read-client-1002-req-42 at view 0), which
  the merge correctly drops and the new view reuses; a deposed/partitioned primary then commits the
  old value LATE on persisted acks, and a replica that still holds the old copy commits it on a bare
  Commit. A per-op "nack merge" was tried and REJECTED: keeping every op a quorum holds resurrects
  ops a later view legitimately abandoned, breaking `Test_View_Change_Log_Selection` (the §4.2
  viewstamp rule — the latest view's log prevails entirely, length included). Hash-chained commit
  does not help either: the deposed primary's commit is locally consistent (its own view-0 op + a
  view-0 Commit). The op is abandoned precisely because it never reached a real quorum, so the true
  fix is preventing the deposed primary from committing on acks from replicas that have since moved
  to a later view and overwritten the op — i.e. acks that no longer reflect a current quorum. That
  needs the full TigerBeetle DVC machinery (per-op headers + nacks to determine the canonical log,
  distinguishing committed from abandoned) done coherently, not a point fix. Open.
