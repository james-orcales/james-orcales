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
