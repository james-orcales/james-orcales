// Package vsr is a Viewstamped Replication (Liskov & Cowling, 2012) protocol core: the agreement
// engine a replicated system embeds to keep a cluster's command log consistent through crashes and
// leader change. It is the VSR counterpart to a Raft library, chosen because its view change is
// deterministic — the primary of a view is fixed by View mod Cluster_Size — which is what a
// simulation-tested system wants. The core is a pure step function: it spawns nothing, performs no
// input or output, and reads no clock of its own. Replica_Receive folds one incoming Message into a
// replica, Replica_Tick folds the passage of time in, and each returns the Step_Output the caller
// must act on — messages to send, entries that became committed, the timer to re-arm. Time
// arrives as an injected Moment; delivery, randomness, and persistence live in the caller (in v1,
// the deterministic simulator in the test suite). The core guarantees agreement on a single ordered
// log of opaque commands; applying those commands to a state machine is the caller's concern.
package vsr

import (
	invariant "github.com/james-orcales/james-orcales/shared/invariant/default"
	"github.com/james-orcales/james-orcales/shared/time"
)

// View is a monotonically increasing epoch with one primary; the primary of a view is the replica
// whose Identifier is View mod Cluster_Size (VSR's deterministic leader rotation).
type View uint64

// Op is the 1-based index of a log entry — VSR's op-number. Zero means "no op".
type Op uint64

// Commit is the highest op a replica knows to be committed — VSR's commit-number.
type Commit uint64

// Replica_Identifier is a replica's fixed index in the cluster, 0..Cluster_Size-1.
type Replica_Identifier uint8

// Nonce is a recovery liveness token: a recovering replica stamps its Recovery with one and ignores
// any Recovery_Response that does not echo it, so a delayed response from an earlier attempt cannot
// complete the current one.
type Nonce uint64

// Epoch is a monotonically increasing reconfiguration epoch; it is 0 until the first
// reconfiguration.
type Epoch uint64

// View_Change_Suffix is how many trailing log entries a Do_View_Change carries (§5.3). The new
// primary ranks reporters by (last-normal-view, op) — numbers, not the log — and reconstructs the
// winner's log from its own committed prefix plus this suffix, fetching anything still missing. Two
// is the paper's default: enough that the common case (the new primary already holds the committed
// prefix the suffix attaches onto) needs no fetch, while the message stays bounded regardless of
// log length.
const View_Change_Suffix Op = 2

// Client_Identifier names a client of the cluster. It is not a Replica_Identifier: clients are not
// in the Configuration, so a reply is addressed by Client, never by To.
type Client_Identifier uint64

// Request_Number is a client's own monotonically increasing sequence number for its requests; it is
// what the client-table dedups on to give exactly-once semantics across retries.
type Request_Number uint64

// State_Machine is the application the committed log drives — the deterministic up-call VSR makes
// as the commit number advances. It is injected so the core stays a pure agreement engine that
// knows nothing of the application's meaning. It is OPTIONAL: a zero State_Machine (a nil Execute)
// runs the core in log-only mode, committing without executing or replying, which is exactly the
// behavior before execution existed.
type State_Machine struct {
	// Execute applies one committed command, against the prediction the primary predetermined
	// for it, and returns its client-visible result. The core calls it exactly once per op, in
	// op order, as the commit number crosses the op; a nil Execute means log-only mode. It must
	// be deterministic given (command, prediction) — every replica executes the same
	// committed log with the same stored prediction and must reach the same result — so it
	// may not read a clock or randomness; any non-determinism the op needs arrives through
	// prediction (§4.4).
	Execute func(command []byte, prediction []byte) (result []byte)
	// Predict computes the predetermined value for a non-deterministic op, called only on the
	// primary as it accepts a new request, never on a backup (which copies the stored value).
	// It is where a clock or randomness legitimately enters: the primary fixes the value once
	// and propagates it in the log entry. Nil means ordinary deterministic ops — no
	// prediction is made and the stored value stays empty.
	Predict func(command []byte) (prediction []byte)
	// Combine folds f backups' predictions and the primary's own into the value the op executes
	// against, for the pre-step variant where the value must reflect the cluster rather than
	// the primary alone (§4.4). It must be deterministic in its inputs. Nil selects
	// local-predict: the primary uses Predict alone with no pre-step round. Non-nil selects the
	// pre-step protocol: the primary gathers a Predict_Response from each of f backups first.
	Combine func(command []byte, responses [][]byte, own []byte) (prediction []byte)
	// Snapshot captures the application state at the current commit as an opaque blob, for
	// checkpointing and log compaction (§5.1). The core calls it when a checkpoint is taken,
	// after executing the committed prefix, so the returned bytes describe the state as of the
	// checkpoint's op. It must be a pure function of the executed prefix — every replica that
	// checkpoints the same op must produce byte-identical bytes — so it may read no clock or
	// randomness. Nil disables checkpointing: the log is never compacted, which is the behavior
	// before checkpoints existed, and is the correct fallback for a stateless log-only core.
	Snapshot func() (state []byte)
	// Restore re-installs the application state from a Snapshot blob, the inverse of Snapshot.
	// The core calls it when it adopts a checkpoint — through state transfer, recovery, or a
	// New_State gap response — before replaying the committed ops after the checkpoint, so
	// the state machine ends reflecting exactly the committed prefix. It must be deterministic
	// in its input. Nil is paired with a nil Snapshot, since a core that never checkpoints
	// never restores.
	Restore func(state []byte)
}

// Client_Record is the client-table entry for one client: the highest request the cluster has
// accepted from it, and — once that request has executed — its cached result. It is what makes
// a retried request idempotent: a duplicate is answered from here rather than re-executed.
type Client_Record struct {
	// Request_Number is the highest request-number this replica has accepted from the client.
	Request_Number Request_Number
	// Result is the cached Execute result for Request_Number, valid only once Executed is true.
	Result []byte
	// Executed reports whether Request_Number's op has committed and executed, so its Result is
	// the authoritative reply; false means the request is still in flight.
	Executed bool
}

// Configuration is the ordered group membership; the primary of a view is Configuration[View mod
// len(Configuration)].
type Configuration []Replica_Identifier

// Prediction_Round_Key identifies a request the primary is running the pre-step prediction round
// for, before it has assigned the request an op (§4.4). The op is not yet known, so the round is
// keyed by the client and its request-number — the same pair the eventual Predict_Response
// echoes.
type Prediction_Round_Key struct {
	// Client is the client whose request this round serves.
	Client Client_Identifier
	// Request_Number is that client's request-number for the buffered request.
	Request_Number Request_Number
}

// Prediction_Round is one buffered request's in-progress pre-step round: the command to append once
// the round completes, and each distinct backup's prediction collected so far (filed by sender so a
// duplicated or late Predict_Response cannot be counted twice).
type Prediction_Round struct {
	// Command is the buffered request's payload, appended once f responses arrive.
	Command []byte
	// Responses holds each distinct backup's prediction, keyed by sender for dedup.
	Responses map[Replica_Identifier][]byte
}

// Status_Normal is the steady state: the replica serves Prepare and Commit and, if primary, drives
// replication.
const Status_Normal Status = 0

// Status_View_Change means the replica has stopped normal work and is collecting the votes that
// install a new view.
const Status_View_Change Status = 1

// Status_Recovery means the replica lost its volatile state and is awaiting a quorum of
// Recovery_Response before it may participate again.
const Status_Recovery Status = 2

// Status_Transition means the replica is moving between epochs (§7): a new-group member catching up
// to the start of the epoch, or a replaced replica serving state transfer until the new group is
// up. It is the status a replica holds from the moment it learns the new epoch until it is either
// caught up (back to Status_Normal) or done serving the new group (Status_Shutdown).
const Status_Transition Status = 3

// Status_Shutdown means a replaced replica has handed its state to the new group — it has seen
// f'+1 Epoch_Started — and is done (§7.1.2). It no longer participates in any protocol.
const Status_Shutdown Status = 4

// Status selects which phase of the protocol a replica is in.
type Status uint8

// Log_Entry is one opaque client command together with the view it was first prepared in.
type Log_Entry struct {
	// View is the view this op was first prepared in. VSR tags each entry with its birth view.
	// This is metadata only: the merge ranks replicas by last-normal-view, never by per-entry
	// views, which crash-recovery can leave non-monotonic.
	View View
	// Command is the opaque client payload; the core never interprets it.
	Command []byte
	// Client is the client whose request this op serves; it travels with the entry so every
	// replica, executing the entry on commit, knows which client-table record to update.
	Client Client_Identifier
	// Request_Number is the client's request-number for this op, carried for the same reason as
	// Client: a committed entry must be self-describing so any replica can update its table.
	Request_Number Request_Number
	// Prediction is the value the primary predetermined for a non-deterministic op (§4.4),
	// stored in the entry so every replica executes the op against the identical value rather
	// than each recomputing it. It is empty for an ordinary deterministic op. Backups copy it
	// verbatim from the Prepare and never recompute it, keeping replicas' executions agreed.
	Prediction []byte
	// New_Configuration is non-empty only for a Reconfiguration entry (§7): it is the
	// membership the group becomes when this op commits. It rides in the log entry so a new
	// primary selected by a view change, reading its topmost entry, can see the reconfiguration
	// and re-drive the epoch handoff. An entry executes as a Reconfiguration — affecting only
	// VR state, never the State_Machine — exactly when this is non-empty.
	New_Configuration Configuration
	// New_Active_Count is the active-replica count the new epoch adopts (§6.1), carried
	// alongside New_Configuration in a Reconfiguration entry so the standby layout travels with
	// the membership change; meaningful only when New_Configuration is non-empty.
	New_Active_Count uint8
}

// Replica is one node's complete protocol state. Identity and the two timeout durations are set by
// the caller and never mutated by the core; everything else is protocol state the step functions
// evolve. The tally maps are keyed by Replica_Identifier so a duplicated message — which the
// network will produce — cannot be counted toward a quorum twice.
type Replica struct {
	// Identifier is this replica's fixed cluster index; it is the primary of View when
	// Identifier equals View mod Cluster_Size.
	Identifier Replica_Identifier
	// Configuration is the ordered group membership this replica belongs to. The leading
	// Active_Count members are the active replicas — they store the application state and
	// execute operations — and any after them are standbys (§6.1). The primary of a view is
	// Configuration[View mod Active_Count], always an active replica.
	Configuration Configuration
	// Active_Count is how many leading members of Configuration are active replicas; the rest
	// are standbys (§6.1). A standby replicates the log but never votes, executes, replies, or
	// becomes primary — TigerBeetle's non-voting-standby design (see replica_quorum), so the
	// voting set runs the base protocol unchanged. Zero means every member is active — no
	// standbys, the behavior before standbys existed. Set by the caller, never mutated.
	Active_Count uint8
	// Old_Configuration is the prior epoch's membership during a handoff (§7); empty unless
	// this replica is Status_Transition or Status_Shutdown. A replaced replica serves state
	// transfer to the new group out of it, and a quorum over the OLD group (the
	// decision-relevant configuration while the old group still owns the reconfiguration op) is
	// computed from it.
	Old_Configuration Configuration
	// Epoch is the reconfiguration epoch this replica is in; 0 until the first reconfiguration.
	Epoch Epoch
	// Heartbeat is how often a primary with no client work broadcasts its commit number.
	Heartbeat time.Duration
	// Timeout is how long a backup waits without contact before starting a view change, and how
	// long a recovering replica waits before re-broadcasting Recovery.
	Timeout time.Duration
	// State_Machine is the application the committed log drives; a zero value runs the core in
	// log-only mode. It is set by the caller and never mutated by the core.
	State_Machine State_Machine
	// Checkpoint_Interval is how many committed ops pass between checkpoints; 0 disables
	// checkpointing entirely (the log is never compacted). Set by the caller, never mutated.
	Checkpoint_Interval uint64
	// Log_Retain is how many ops of suffix to keep in Log after the checkpoint op when
	// compacting, so an in-flight recovery or transfer that needs a recently-GC'd op is not
	// stranded. Set by the caller, never mutated.
	Log_Retain uint64
	// Batch_Max is the most ops a busy primary collects into one Prepare before flushing the
	// batch (§6.2). 0 or 1 disables batching: every accepted request flushes immediately, the
	// behavior before batching existed. Set by the caller, never mutated.
	Batch_Max uint64

	// Status is the protocol phase this replica is in.
	Status Status
	// View is the epoch this replica currently believes is active.
	View View
	// Op is the highest op in Log; it always equals Log_Start plus the log length.
	Op Op
	// Commit is the highest op this replica knows to be committed.
	Commit Commit
	// Executed is the highest op the state machine has executed; it tracks Commit in steady
	// state but is recorded separately so a path that adopts a committed prefix wholesale
	// (view-change install, Start_View, recovery rejoin, state-transfer apply) knows which ops
	// it must still execute or restore-then-execute, keeping the application state reflecting
	// exactly Commit. With no state machine injected it stays a bookkeeping counter the
	// log-only core ignores.
	Executed Op
	// Log is the replica's ordered command log, holding only the suffix after Log_Start once a
	// prefix has been compacted; op's entry is Log[op - Log_Start - 1].
	Log []Log_Entry
	// Log_Start is the op-number just before the first entry still in Log; it is 0 until the
	// first compaction, then advances as the prefix is garbage-collected. An op at or below it
	// lives only in the checkpoint, not in Log.
	Log_Start Op
	// Checkpoint_Op is the op-number the last checkpoint covers — the highest op whose
	// execution Checkpoint_State reflects; 0 means no checkpoint has been taken.
	Checkpoint_Op Op
	// Checkpoint_State is the application state Snapshot captured at Checkpoint_Op; nil until
	// the first checkpoint. It is shipped to a replica that has GC'd past a peer's needs, and
	// restored on warm recovery.
	Checkpoint_State []byte

	// Last_Normal_View is the highest view in which this replica was Status_Normal; the
	// view-change merge ranks Do_View_Change logs by it.
	Last_Normal_View View
	// Timer_Deadline is when the replica's active timer next fires, in the injected clock's
	// frame; Replica_Tick does nothing until Now reaches it.
	Timer_Deadline time.Moment

	// Start_View_Change_From tallies the distinct senders of a Start_View_Change for the view
	// this replica is changing to.
	Start_View_Change_From map[Replica_Identifier]bool
	// Do_View_Change_From holds, at the new primary, the Do_View_Change report from each
	// distinct replica that has reported.
	Do_View_Change_From map[Replica_Identifier]Message
	// View_Change_Deferred reports that the new primary selected a winning report it cannot
	// reconstruct from its own log plus the received suffix and is awaiting a New_State before
	// it installs (§5.3 deferred install). While it is set the primary stays in
	// Status_View_Change and emits no Start_View; only a New_State from View_Change_Fetch_From
	// at op at least View_Change_Fetch_Op completes the install, so a stray New_State cannot
	// install early and the Start_View is emitted exactly once.
	View_Change_Deferred bool
	// View_Change_Fetch_From is the selected reporter the deferred install fetches the complete
	// log from; meaningful only while View_Change_Deferred.
	View_Change_Fetch_From Replica_Identifier
	// View_Change_Fetch_Op is the selected reporter's op, the floor the awaited New_State must
	// reach so the installed log holds every op the winner reported; meaningful only while
	// View_Change_Deferred.
	View_Change_Fetch_Op Op
	// Prepare_Ok_From tallies, per op, the distinct backups that have explicitly acknowledged
	// it. A backup is recorded for an op only by a Prepare_Ok it sent for that op — which it
	// sends only after appending the op from a Prepare — so an op a backup acquired by state
	// transfer, never acknowledging it, is not counted toward that op's quorum. Counting such
	// an op would let the primary commit it on a set that does not durably hold it: a backup
	// may truncate a state-transferred, never-locally-committed op on a later view change, and
	// a quorum resting on it could then lose the op (the over-commit the simulator surfaced). A
	// batched Prepare is acknowledged op by op, so each op in it is tallied independently here.
	Prepare_Ok_From map[Op]map[Replica_Identifier]bool
	// Request_Buffer holds accepted requests the primary has predetermined but not yet flushed
	// as a Prepare, the staging area for load-adaptive batching (§6.2): a busy primary collects
	// arrivals here and flushes them as one batch, while an idle primary flushes each
	// immediately. Each entry already carries its prediction and has its in-flight client-table
	// record, so flushing only appends to the log and broadcasts.
	Request_Buffer []Log_Entry
	// Recovery_Nonce is the nonce of the in-flight recovery attempt.
	Recovery_Nonce Nonce
	// Recovery_From holds the Recovery_Response from each distinct replica matching the nonce.
	Recovery_From map[Replica_Identifier]Message
	// Client_Table is the per-client dedup state, updated on every replica as ops execute, so a
	// replica can answer a retried request and a new primary inherits the exactly-once history.
	Client_Table map[Client_Identifier]Client_Record
	// Predict_From holds, on the primary running the pre-step variant, the in-progress
	// prediction round for each buffered request, keyed by client and request-number since the
	// op is not yet assigned. A round completes — its request appended — once it holds f
	// distinct backups' predictions, then its entry is removed.
	Predict_From map[Prediction_Round_Key]Prediction_Round
	// Epoch_Start_Op is the op-number at which the current epoch begins (§7.1): the op-number
	// of the reconfiguration that opened it. A transitioning new-group member must learn the
	// log up to here before it may serve; 0 when no epoch handoff is in progress.
	Epoch_Start_Op Op
	// Epoch_Handoff_Due reports that the reconfiguration op has just executed on this replica
	// and the role-specific epoch transition (config swap, status, Commit/Start_Epoch emission)
	// has not yet run. The execution spine sets it; the top-level step drains it once execution
	// settles, so the transition's outgoing messages flow out of the step that committed the
	// op.
	Epoch_Handoff_Due bool
	// Epoch_New_Configuration is the membership the pending handoff installs, captured from the
	// reconfiguration entry as it executed. The drain reads it here rather than re-reading the
	// entry, because compaction can garbage-collect the reconfiguration op (a checkpoint
	// landing on it) in the same step before the drain runs, leaving the entry unreadable.
	// Meaningful only while Epoch_Handoff_Due.
	Epoch_New_Configuration Configuration
	// Epoch_New_Active_Count is the active-replica count the pending handoff installs (§6.1),
	// captured from the reconfiguration entry alongside Epoch_New_Configuration for the same
	// reason; meaningful only while Epoch_Handoff_Due.
	Epoch_New_Active_Count uint8
	// Standby_Promotion_Due reports that a reconfiguration promoted this replica from
	// standby to active (§6.1) and it has not yet materialized the application state it never
	// built as a standby. It is set at the role flip and consumed when the replica next returns
	// to Status_Normal, which restores its carried checkpoint and replays the un-checkpointed
	// suffix. It makes the rebuild fire exactly once regardless of which epoch-completion path
	// runs.
	Standby_Promotion_Due bool
	// Epoch_Started_From tallies, at a replaced replica, the distinct new-group members that
	// have reported their epoch started (§7.1.2). Once it holds f'+1 — the new group's quorum
	// — the replaced replica's state is safely transferred and it shuts down.
	Epoch_Started_From map[Replica_Identifier]bool
	// Epoch_Up_From tallies, at the new primary, the configured members confirmed active in
	// the current epoch (they reported Epoch_Started, or were continuing members at the
	// handoff). The primary re-drives Start_Epoch to a configured member NOT in this set, so a
	// replica added by a reconfiguration is still brought up after replaced ones retire (§7.2).
	Epoch_Up_From map[Replica_Identifier]bool
	// Handoff_Commit_Redrive counts the heartbeats the new primary keeps re-driving the
	// reconfiguration commit to the old epoch. That commit is sent once at handoff drain; if
	// lost, an old member that acked the reconfiguration op never learns it committed and
	// strands at the old epoch, re-learning via quorum-blocking recovery. Several stranded at
	// once deadlock beyond the fault model, so the primary re-sends for a window; an advanced
	// member sees a stale commit and drops it (Bug 19).
	Handoff_Commit_Redrive int
}

// Message_Kind_Request is a client command arriving at the primary; it carries Command.
const Message_Kind_Request Message_Kind = 0

// Message_Kind_Prepare is the primary asking backups to replicate a batch of one or more ops
// (§6.2); it carries Entries, the highest Op in the batch, and the primary's Commit. A batch of one
// is the common case; a busy primary collects several requests into one Prepare.
const Message_Kind_Prepare Message_Kind = 1

// Message_Kind_Prepare_Ok is a backup acknowledging a prepared op; it carries Op, the highest op
// the backup has appended, which acknowledges every op through it since the log is gap-free.
const Message_Kind_Prepare_Ok Message_Kind = 2

// Message_Kind_Commit is the primary's heartbeat advertising its commit number.
const Message_Kind_Commit Message_Kind = 3

// Message_Kind_Start_View_Change proposes moving to View.
const Message_Kind_Start_View_Change Message_Kind = 4

// Message_Kind_Do_View_Change reports a replica's state to the new primary; it carries Op,
// Last_Normal_View, Commit, and a bounded Log_Suffix (§5.3), not the full log.
const Message_Kind_Do_View_Change Message_Kind = 5

// Message_Kind_Start_View installs a new view's log on the backups; it carries Log and Commit.
const Message_Kind_Start_View Message_Kind = 6

// Message_Kind_Recovery asks the cluster for current state; it carries Nonce.
const Message_Kind_Recovery Message_Kind = 7

// Message_Kind_Recovery_Response answers a Recovery; it echoes Nonce and reports View, and from the
// primary also Log, Op, and Commit.
const Message_Kind_Recovery_Response Message_Kind = 8

// Message_Kind_Get_State asks a more current replica for its state; it carries the requester's View
// and Op so a behind replica can catch up without a full view change.
const Message_Kind_Get_State Message_Kind = 9

// Message_Kind_New_State answers a Get_State with the responder's View, Log, Op, and Commit, and
// — when the requested prefix has been garbage-collected — the responder's Checkpoint_Op and
// Checkpoint_State, with Log carrying only the suffix after the checkpoint.
const Message_Kind_New_State Message_Kind = 10

// Message_Kind_Reply is the primary's answer to a client, addressed by Client rather than To; it
// carries the Request_Number it answers and the Execute Result.
const Message_Kind_Reply Message_Kind = 11

// Message_Kind_Predict_Request is the primary asking backups for their prediction of a pending
// non-deterministic op (§4.4 pre-step); it carries the Command and the Client/Request_Number that
// matches the response back to the pending request.
const Message_Kind_Predict_Request Message_Kind = 12

// Message_Kind_Predict_Response is a backup's prediction answering a Predict_Request; it echoes the
// Client/Request_Number and carries the backup's predicted value in Result.
const Message_Kind_Predict_Response Message_Kind = 13

// Message_Kind_Reconfiguration is the administrator client's request to change the group (§7); it
// carries the client's Epoch and Request_Number for the freshness check and the New_Configuration
// the group is to become.
const Message_Kind_Reconfiguration Message_Kind = 14

// Message_Kind_Start_Epoch tells a member of the new group the epoch has begun (§7.1): it carries
// the new Epoch, the epoch-start Op, the old configuration (in Log, repurposed), and the
// New_Configuration, so a new node knows where to fetch state and how far to catch up.
const Message_Kind_Start_Epoch Message_Kind = 15

// Message_Kind_Epoch_Started is a caught-up new replica's acknowledgement to the replaced replicas
// (§7.1.1): it carries the new Epoch and the sender's identifier (From), counting toward the f'+1 a
// replaced replica needs before it may shut down.
const Message_Kind_Epoch_Started Message_Kind = 16

// Message_Kind_New_Epoch is an old replica's redirect to a stale-epoch client (§7.4): it carries
// the current View and the New_Configuration so the client can find the group that moved on.
const Message_Kind_New_Epoch Message_Kind = 17

// Message_Kind_Check_Epoch is the administrator's completion probe (§7.3): a normal-case request in
// the new epoch whose reply tells the administrator the reconfiguration has finished. It carries
// the client's Epoch and Request_Number like any client request.
const Message_Kind_Check_Epoch Message_Kind = 18

// Message_Kind selects which fields of a Message carry meaning.
type Message_Kind uint8

// Message is the single discriminated record exchanged in the protocol, and the carrier of a client
// Request. Kind selects which fields are meaningful — the same tagged-struct shape the invariant
// package uses for Dot_Element — so a Message stays a plain copyable value the simulator can
// freely reorder, duplicate, and store.
type Message struct {
	// Kind selects which other fields carry meaning.
	Kind Message_Kind
	// From is the sender's identifier.
	From Replica_Identifier
	// To is the delivery target's identifier.
	To Replica_Identifier
	// View is the sender's view; every protocol message carries it.
	View View
	// Epoch is the sender's epoch; every message carries it.
	Epoch Epoch

	// Op is the op-number this message concerns. For a Prepare carrying a batch it is the
	// highest op in the batch; the receiver derives the batch's first op as Op minus
	// len(Entries) plus one.
	Op Op
	// Commit is the sender's commit number.
	Commit Commit
	// Entries are the log entries a Prepare replicates — a batch of one or more consecutive ops
	// (§6.2), in op order ending at Op.
	Entries []Log_Entry

	// Command carries a client Request's payload before it becomes a log entry.
	Command []byte

	// Client is the client a Request comes from and a Reply is addressed to; replies leave To
	// zero, since a client is not a member of the Configuration.
	Client Client_Identifier
	// Request_Number is the client's request-number, carried by a Request and echoed by a Reply
	// so the client can match a reply to the request it sent.
	Request_Number Request_Number
	// Result is the Execute result a Reply carries back to the client.
	Result []byte

	// Log is the sender's full log, carried by Start_View, a primary's Recovery_Response, and a
	// New_State. When a New_State, Start_View, or Recovery_Response carries a checkpoint, Log
	// holds only the suffix after Checkpoint_Op. Do_View_Change does not use it — it carries a
	// bounded Log_Suffix instead (§5.3).
	Log []Log_Entry
	// Log_Suffix is the bounded trailing slice a Do_View_Change carries instead of the full log
	// (§5.3): the last View_Change_Suffix entries, ending at Op. The receiver derives its first
	// op as Op minus the suffix length, and ranks reporters by (Last_Normal_View, Op) without
	// needing the rest of the log.
	Log_Suffix []Log_Entry
	// Last_Normal_View is the sender's last view in Status_Normal, the merge key.
	Last_Normal_View View

	// Nonce ties a Recovery to its Recovery_Response.
	Nonce Nonce

	// Checkpoint_Op is the op a shipped checkpoint covers: a recovering replica advertises its
	// own in a Recovery, and a New_State or Recovery_Response carrying a checkpoint reports the
	// op its accompanying Checkpoint_State reflects. Zero means no checkpoint travels with the
	// message.
	Checkpoint_Op Op
	// Checkpoint_State is the application-state blob a New_State or Recovery_Response ships
	// when the requester needs a prefix the responder has already garbage-collected; nil means
	// none.
	Checkpoint_State []byte
	// Checkpoint_Client_Table ships the responder's client-table alongside Checkpoint_State.
	// The exactly-once records for requests the checkpoint compacted live only in the table,
	// not in the shipped log suffix; without them a restoring replica that later becomes
	// primary would re-append an already-executed request at a fresh op (an exactly-once
	// violation). It travels exactly when Checkpoint_State does.
	Checkpoint_Client_Table map[Client_Identifier]Client_Record

	// New_Configuration is the membership a Reconfiguration requests, a Start_Epoch installs,
	// and a New_Epoch redirect advertises (§7); empty in every other message.
	New_Configuration Configuration
	// New_Active_Count is the new epoch's active-replica count (§6.1), carried with
	// New_Configuration by a Reconfiguration, Start_Epoch, and New_Epoch so every replica
	// adopts the same standby layout; meaningful only where New_Configuration is.
	New_Active_Count uint8
	// Old_Configuration is the prior epoch's membership a Start_Epoch carries (§7.1) so a brand
	// new node knows which replicas to fetch state from; empty in every other message.
	Old_Configuration Configuration
}

// Timer_Kind_None means the step armed no timer.
const Timer_Kind_None Timer_Kind = 0

// Timer_Kind_Commit is the primary's heartbeat timer.
const Timer_Kind_Commit Timer_Kind = 1

// Timer_Kind_View_Change is a backup's failure-detector and view-change-retry timer.
const Timer_Kind_View_Change Timer_Kind = 2

// Timer_Kind_Recovery is a recovering replica's Recovery-retry timer.
const Timer_Kind_Recovery Timer_Kind = 3

// Timer_Kind selects which timer a Timer_Reset re-arms.
type Timer_Kind uint8

// Timer_Reset tells the caller which timer to re-arm and when it should fire. The replica also
// stores this, so a caller that prefers to tick every replica every grain may ignore it; a real
// transport would use it to schedule the next wake-up.
type Timer_Reset struct {
	// Kind is which timer was armed.
	Kind Timer_Kind
	// Deadline is when the armed timer should fire, in the injected clock's frame.
	Deadline time.Moment
}

// Step_Output is everything one step produces.
type Step_Output struct {
	// Messages are the messages to deliver, each addressed via its To field.
	Messages []Message
	// Committed are the entries that became committed this step, in op order.
	Committed []Log_Entry
	// Replies are the client-addressed Replies this step produced, each routed by its Client
	// field; only the primary produces them, and only with a State_Machine injected.
	Replies []Message
	// Timer is the timer to re-arm, if any.
	Timer Timer_Reset
}

// New_Replica_Input configures a fresh replica.
type New_Replica_Input struct {
	// Identifier is the replica's fixed cluster index.
	Identifier Replica_Identifier
	// Configuration is the ordered group membership the replica belongs to; active replicas
	// lead, standbys (if any) follow.
	Configuration Configuration
	// Active_Count is how many leading members of Configuration are active replicas (§6.1); 0
	// means all are active (no standbys).
	Active_Count uint8
	// Epoch is the reconfiguration epoch the replica starts in.
	Epoch Epoch
	// Heartbeat is the primary's commit-broadcast interval.
	Heartbeat time.Duration
	// Timeout is the view-change and recovery timeout.
	Timeout time.Duration
	// State_Machine is the application the committed log drives; a zero value runs log-only.
	State_Machine State_Machine
	// Checkpoint_Interval is how many committed ops pass between checkpoints; 0 never
	// checkpoints.
	Checkpoint_Interval uint64
	// Log_Retain is the suffix kept after the checkpoint op when compacting; when it is zero
	// but checkpointing is enabled it defaults to Checkpoint_Interval, so the log always keeps
	// at least an interval's worth of suffix past each checkpoint.
	Log_Retain uint64
	// Batch_Max is the most ops a busy primary collects into one Prepare (§6.2); 0 or 1
	// disables batching.
	Batch_Max uint64
	// Now is the current clock reading, used to arm the first timer.
	Now time.Moment
}

// New_Replica builds a normal replica in view zero with its tallies initialized and its first timer
// armed relative to Now. A bare struct literal also works for tests that never tick; the
// constructor exists so the failure detector has a real initial deadline rather than zero.
func New_Replica(input *New_Replica_Input) (replica Replica) {
	// Default the retained suffix to one checkpoint interval when checkpointing is on but the
	// caller left it zero: a checkpoint must keep enough suffix that an in-flight transfer
	// needing a recently-committed op is not stranded, and an interval's worth is the natural
	// minimum.
	retain := input.Log_Retain
	if input.Checkpoint_Interval > 0 {
		if retain == 0 {
			retain = input.Checkpoint_Interval
		}
	}
	replica = Replica{
		Identifier:          input.Identifier,
		Configuration:       input.Configuration,
		Active_Count:        input.Active_Count,
		Epoch:               input.Epoch,
		Heartbeat:           input.Heartbeat,
		Timeout:             input.Timeout,
		State_Machine:       input.State_Machine,
		Checkpoint_Interval: input.Checkpoint_Interval,
		Log_Retain:          retain,
		Batch_Max:           input.Batch_Max,
		Status:              Status_Normal,
	}
	replica_ensure_scratch(&replica)
	replica_arm_timer(&replica, input.Now)
	return replica
}

// Replica_Tick_Input folds the passage of time up to Now into Replica.
type Replica_Tick_Input struct {
	// Replica is the replica to advance.
	Replica *Replica
	// Now is the current clock reading.
	Now time.Moment
}

// Replica_Tick fires the replica's timer if Now has reached its deadline: a recovering replica
// retries Recovery, a primary heartbeats, and a backup or stalled view-change advances the view.
// Before the deadline it does nothing.
func Replica_Tick(input *Replica_Tick_Input) (output Step_Output) {
	replica := input.Replica
	replica_ensure_scratch(replica)
	output = replica_tick_fire(replica, input.Now)
	step_output_fold(&output, replica_drain_epoch_handoff(replica, input.Now))
	replica_assert_safety(replica)
	return output
}

// Fires the due timer, if any, and returns what it emits; Replica_Tick wraps it with the post-step
// invariant check.
func replica_tick_fire(replica *Replica, now time.Moment) (output Step_Output) {
	if now < replica.Timer_Deadline {
		return output
	}
	if replica.Status == Status_Recovery {
		// Retry recovery with the same nonce in case the first Recovery, or the responses,
		// were lost; the nonce stays fixed so earlier responses still count.
		output.Messages = replica_broadcast(replica, Message{
			Kind:  Message_Kind_Recovery,
			Nonce: replica.Recovery_Nonce,
		})
		output.Timer = replica_arm_timer(replica, now)
		return output
	}
	if replica.Status == Status_Shutdown {
		return output // Out of the protocol; a shutdown replica does nothing.
	}
	if replica.Status == Status_Transition {
		return replica_tick_transition(replica, now)
	}
	if replica.Status == Status_Normal {
		if replica.Identifier == replica_primary_identifier(replica) {
			output.Messages = replica_primary_heartbeat(replica)
			output.Timer = replica_arm_timer(replica, now)
			return output
		}
	}
	// A standby never drives a view change (TigerBeetle's non-voting-standby design, see
	// replica_quorum): it is not responsible for liveness and casts no view-change vote, so on
	// a timeout it simply re-arms and keeps following. It learns of a completed view change
	// from the Start_View the new primary broadcasts, or catches up by state transfer on a
	// later-view message. The voting backups alone fail a primary over.
	if replica_is_standby(replica) {
		output.Timer = replica_arm_timer(replica, now)
		return output
	}
	// A backup that has lost the primary, or a replica whose in-progress view change has itself
	// stalled, advances to the next view — rotating the primary off a dead node.
	return replica_start_view_change(replica, replica.View+1, now)
}

// Fires a transitioning replica's timer (§7.1). A replica being replaced that has not yet heard
// f'+1 Epoch_Started resends Start_Epoch to the new replicas, prodding them to come up so it can
// shut down (§7.1.2 step 3). A new-group member still catching up re-issues its state-transfer
// request, in case the first Get_State or its answer was lost.
func replica_tick_transition(replica *Replica, now time.Moment) (output Step_Output) {
	being_replaced := !configuration_contains(replica.Configuration, replica.Identifier)
	if being_replaced {
		output.Messages = replica_resend_start_epoch(replica)
		output.Timer = replica_arm_timer(replica, now)
		return output
	}
	output.Messages = replica_epoch_state_transfer(replica)
	output.Timer = replica_arm_timer(replica, now)
	return output
}

// A replaced replica's Start_Epoch resend to the new replicas it has not yet heard an Epoch_Started
// from (§7.1.2 step 3): it carries the new epoch, the epoch-start op, and both configurations, the
// same as the original, so a new node that missed the first can still learn the epoch. A new node
// already active answers with Epoch_Started, advancing this replica toward shutdown.
func replica_resend_start_epoch(replica *Replica) (messages []Message) {
	for _, identifier := range replica.Configuration {
		if replica.Epoch_Started_From[identifier] {
			continue // Already heard from this new replica; no need to prod it.
		}
		messages = append(messages, Message{
			Kind:              Message_Kind_Start_Epoch,
			From:              replica.Identifier,
			To:                identifier,
			View:              0,
			Epoch:             replica.Epoch,
			Op:                replica.Epoch_Start_Op,
			Old_Configuration: replica.Old_Configuration,
			New_Configuration: replica.Configuration,
			New_Active_Count:  replica.Active_Count,
		})
	}
	return messages
}

// Emits what a primary sends each Heartbeat: it advertises its commit number to everyone, and
// re-drives each not-yet-committed op — one op per message — to just the backups that have not
// acknowledged it. This mirrors TigerBeetle: the commit timer carries the commit number, and an
// unacknowledged prepare is re-sent, one prepare per message (per-op), only to the replicas that
// have not responded. A standby never acknowledges (§6.1), so it keeps receiving the op — it needs
// the log to follow and to be promotable. A backup missing a committed op below the commit number
// catches up by ranged state transfer (our pull-repair path), as TigerBeetle's backups pull missing
// prepares. TigerBeetle splits this across a separate per-prepare timer and the commit timer; we
// fold both onto the one heartbeat tick — a v1 simplification with the same per-op,
// send-to-non-responders semantics.
func replica_primary_heartbeat(replica *Replica) (messages []Message) {
	for op := replica.Commit + 1; op <= Commit(replica.Op); op++ {
		acked := replica.Prepare_Ok_From[Op(op)]
		entry := replica_log_entry(replica, Op(op))
		for _, identifier := range replica.Configuration {
			if identifier == replica.Identifier {
				continue
			}
			if acked[identifier] {
				continue // This backup already holds and acknowledged the op.
			}
			messages = append(messages, Message{
				Kind:    Message_Kind_Prepare,
				From:    replica.Identifier,
				To:      identifier,
				View:    replica.View,
				Epoch:   replica.Epoch,
				Op:      Op(op),
				Entries: []Log_Entry{entry},
				Commit:  replica.Commit,
			})
		}
	}
	messages = append(messages, replica_broadcast(replica, Message{
		Kind:   Message_Kind_Commit,
		Commit: replica.Commit,
	})...)
	// Re-drive the reconfiguration commit at the OLD epoch so an old member that missed the
	// one-shot handoff commit advances rather than stranding recovery (Bug 19). An advanced
	// member sees a stale-epoch commit and just redirects, which the primary ignores at its own
	// epoch; the window closes once it has been re-sent enough times.
	if replica.Handoff_Commit_Redrive > 0 {
		replica.Handoff_Commit_Redrive--
		for _, identifier := range replica.Configuration {
			if identifier == replica.Identifier {
				continue
			}
			messages = append(messages, Message{
				Kind:   Message_Kind_Commit,
				From:   replica.Identifier,
				To:     identifier,
				View:   replica.View,
				Epoch:  replica.Epoch - 1,
				Commit: replica.Commit,
			})
		}
	}
	messages = append(messages, replica_primary_resend_start_epoch(replica)...)
	return messages
}

// The new primary re-drives Start_Epoch to any configured member not yet confirmed active in this
// epoch (§7.2): a replica ADDED by a reconfiguration whose Start_Epoch was lost, and whose replaced
// senders have since retired at f'+1, is otherwise stranded at the old epoch forever. Only past the
// founding epoch, where a reconfiguration can have added a member; an up member re-acks
// Epoch_Started and is then skipped, so this self-terminates once the new group is whole.
func replica_primary_resend_start_epoch(replica *Replica) (messages []Message) {
	if replica.Epoch == 0 {
		return nil
	}
	for _, identifier := range replica.Configuration {
		if identifier == replica.Identifier {
			continue
		}
		if replica.Epoch_Up_From[identifier] {
			continue
		}
		messages = append(messages, Message{
			Kind:              Message_Kind_Start_Epoch,
			From:              replica.Identifier,
			To:                identifier,
			View:              0,
			Epoch:             replica.Epoch,
			Op:                replica.Epoch_Start_Op,
			New_Configuration: replica.Configuration,
			New_Active_Count:  replica.Active_Count,
		})
	}
	return messages
}

// Replica_Recover_Input restarts Replica into recovery with a fresh Nonce at time Now.
type Replica_Recover_Input struct {
	// Replica is the replica that has restarted.
	Replica *Replica
	// Nonce is the fresh, caller-supplied recovery token.
	Nonce Nonce
	// Keep_Checkpoint models a checkpoint that survived the crash on disk: when true the
	// replica keeps Checkpoint_Op, Checkpoint_State, and Log_Start and rejoins by restoring
	// from the checkpoint and replaying only the un-checkpointed suffix (warm recovery, §5.1);
	// when false it wipes everything and re-learns the whole log from the cluster (cold
	// recovery).
	Keep_Checkpoint bool
	// Now is the current clock reading.
	Now time.Moment
}

// Replica_Recover models a replica restarting with its volatile state lost: it keeps only its
// identity and configuration, discards view, op, commit, and log, enters Status_Recovery, and
// broadcasts a Recovery stamped with the caller-supplied nonce. The nonce is injected rather than
// drawn internally so the core stays pure and deterministic; the caller owns randomness. A warm
// recovery (Keep_Checkpoint) preserves the on-disk checkpoint and advertises it, so the primary
// need ship only the suffix the checkpoint does not already cover.
func Replica_Recover(input *Replica_Recover_Input) (output Step_Output) {
	replica := input.Replica
	replica_ensure_scratch(replica)
	// A replica being replaced (already holding the new configuration it is not a member of,
	// §7.1.2) that crashes has been superseded: it shuts down rather than recovering, the §7.2
	// rule that an old replica not in the new group does not rejoin. Its state was already
	// handed to the new group, or will be served by the other old replicas.
	if !configuration_contains(replica.Configuration, replica.Identifier) {
		replica.Status = Status_Shutdown
		replica.Old_Configuration = nil
		replica_assert_safety(replica)
		return output
	}
	replica.Status = Status_Recovery
	replica.View = 0
	replica.Op = 0
	replica.Commit = 0
	replica.Log = nil
	replica.Last_Normal_View = 0
	// Cold recovery wipes the checkpoint too — nothing survived the crash. Warm recovery
	// keeps it, so the replica still holds the committed prefix the checkpoint covers and only
	// the suffix must come from the cluster; Op begins at the checkpoint op so the suffix
	// appends contiguously without a gap below Log_Start that the offset index could not
	// address.
	if !input.Keep_Checkpoint {
		replica.Log_Start = 0
		replica.Checkpoint_Op = 0
		replica.Checkpoint_State = nil
		replica.Executed = 0
	} else {
		// The checkpoint survived on disk while the volatile state machine did not: restore
		// it now so the in-memory application state reflects the checkpoint op, and begin
		// Op, Commit, and Executed there. On rejoin replica_adopt_log sees Executed already
		// at the checkpoint and replays only the un-checkpointed suffix the primary ships,
		// never the whole log.
		replica.Log_Start = replica.Checkpoint_Op
		replica.Op = replica.Checkpoint_Op
		replica.Commit = Commit(replica.Checkpoint_Op)
		replica.Executed = replica.Checkpoint_Op
		if replica.State_Machine.Restore != nil {
			replica.State_Machine.Restore(replica.Checkpoint_State)
		}
	}
	// Discard the volatile epoch-handoff bookkeeping: a crashed replica re-learns the current
	// epoch from the cluster (a Recovery_Response, or a higher-epoch Start_Epoch per §7.2), so
	// a half-finished handoff does not survive the restart. Leaving Old_Configuration set would
	// put the recovering replica in the contradictory state of holding handoff scratch while
	// not transitioning — the simulator surfaced exactly this (a transitioning replica that
	// crashed), see simulator_bugs.md. Epoch itself is kept: it is the replica's last known
	// epoch, the floor the epoch gate compares against, and recovery only ever moves it
	// forward.
	replica.Old_Configuration = nil
	replica.Epoch_Start_Op = 0
	replica.Epoch_Handoff_Due = false
	replica.Epoch_Started_From = map[Replica_Identifier]bool{}
	// Recovery materializes application state through the authority's checkpoint when it adopts
	// the log, so drop any pending standby-promotion rebuild; recovery subsumes it.
	replica.Standby_Promotion_Due = false
	replica_reset_primary_scratch(replica)
	replica.Recovery_Nonce = input.Nonce
	replica.Recovery_From = map[Replica_Identifier]Message{}
	output.Messages = replica_broadcast(replica, Message{
		Kind:          Message_Kind_Recovery,
		Nonce:         input.Nonce,
		Checkpoint_Op: replica.Checkpoint_Op,
	})
	output.Timer = replica_arm_timer(replica, input.Now)
	replica_assert_safety(replica)
	return output
}

// Replica_Receive_Input folds one incoming Message into Replica at time Now.
type Replica_Receive_Input struct {
	// Replica is the replica receiving the message.
	Replica *Replica
	// Message is the incoming message.
	Message Message
	// Now is the current clock reading.
	Now time.Moment
}

// Replica_Receive applies one incoming message to the replica, mutating it in place and returning
// the messages to send, entries newly committed, and the timer to re-arm.
func Replica_Receive(input *Replica_Receive_Input) (output Step_Output) {
	replica := input.Replica
	replica_ensure_scratch(replica)
	output = replica_dispatch(replica, input.Message, input.Now)
	step_output_fold(&output, replica_drain_epoch_handoff(replica, input.Now))
	replica_assert_safety(replica)
	return output
}

// Routes a message to the handler for its kind, after the epoch-precedence gate (§7.2): epoch
// dominates view, so the gate runs first on every message and only an epoch-matching message
// reaches a handler. A stale-epoch message is turned away with a New_Epoch redirect, a future-epoch
// one is adopted, and the gate owns those two cases entirely; a matching one falls through to its
// handler, where the existing view comparison applies.
func replica_dispatch(replica *Replica, message Message, now time.Moment) (output Step_Output) {
	gate, proceed := replica_message_epoch_check(replica, message, now)
	if !proceed {
		return gate
	}
	switch message.Kind {
	case Message_Kind_Request:
		return replica_receive_request(replica, message)
	case Message_Kind_Reconfiguration:
		return replica_receive_reconfiguration(replica, message)
	case Message_Kind_Check_Epoch:
		// A Check_Epoch (§7.3) is an ordinary client request in the new epoch whose reply
		// signals completion; the request path handles it like any client command.
		return replica_receive_request(replica, message)
	case Message_Kind_Prepare:
		return replica_receive_prepare(replica, message, now)
	case Message_Kind_Prepare_Ok:
		return replica_receive_prepare_ok(replica, message)
	case Message_Kind_Commit:
		return replica_receive_commit(replica, message, now)
	case Message_Kind_Start_View_Change:
		return replica_receive_start_view_change(replica, message, now)
	case Message_Kind_Do_View_Change:
		return replica_receive_do_view_change(replica, message, now)
	case Message_Kind_Start_View:
		return replica_receive_start_view(replica, message, now)
	case Message_Kind_Recovery:
		return replica_receive_recovery(replica, message)
	case Message_Kind_Recovery_Response:
		return replica_receive_recovery_response(replica, message, now)
	case Message_Kind_Get_State:
		return replica_receive_get_state(replica, message)
	case Message_Kind_New_State:
		return replica_receive_new_state(replica, message, now)
	case Message_Kind_Predict_Request:
		return replica_receive_predict_request(replica, message)
	case Message_Kind_Predict_Response:
		return replica_receive_predict_response(replica, message)
	case Message_Kind_Start_Epoch:
		return replica_receive_start_epoch(replica, message, now)
	case Message_Kind_Epoch_Started:
		return replica_receive_epoch_started(replica, message)
	}
	return output
}

// Folds one Step_Output into another, concatenating messages, committed entries, and replies and
// keeping the later timer when it armed one. It lets a step run a follow-on action (the epoch
// handoff) after its primary handler and merge what the follow-on emits into the same output.
func step_output_fold(into *Step_Output, more Step_Output) {
	into.Messages = append(into.Messages, more.Messages...)
	into.Committed = append(into.Committed, more.Committed...)
	into.Replies = append(into.Replies, more.Replies...)
	if more.Timer.Kind != Timer_Kind_None {
		into.Timer = more.Timer
	}
}

// Performs the epoch handoff once the reconfiguration op has executed on this replica (§7.1),
// draining the pending flag the execution spine set. Every replica advances to the new epoch; its
// role decides the rest. The OLD primary first tells the other old replicas the reconfiguration
// committed (a Commit at the old epoch, so they too execute it and self-advance) and sends
// Start_Epoch to the added nodes. Then this replica applies its role: a member of the new group
// resets to view 0 in the new configuration and, being already caught up, returns to normal and
// announces Epoch_Started to the replaced replicas; a replica being replaced moves its
// configuration into Old_Configuration, adopts the new one, and stays transitioning to serve state
// transfer until the new group is up. Returns the messages and timer the transition produced.
func replica_drain_epoch_handoff(replica *Replica, now time.Moment) (output Step_Output) {
	if !replica.Epoch_Handoff_Due {
		return output
	}
	replica.Epoch_Handoff_Due = false
	new_configuration := replica.Epoch_New_Configuration
	new_active_count := replica.Epoch_New_Active_Count
	is_old_primary := replica.Identifier == replica_primary_identifier(replica)
	if is_old_primary {
		// Tell the other OLD replicas the reconfiguration committed, at the old epoch and
		// view (still held here, before the swap below) so they process it normally, commit
		// the op, and self-advance; and announce the new epoch to the nodes being added,
		// which learn it only through Start_Epoch. Both read the old configuration from
		// replica.Configuration.
		output.Messages = append(output.Messages, replica_old_group_commit(replica)...)
		output.Messages = append(output.Messages,
			replica_start_epoch_messages(
				replica, new_configuration, new_active_count)...)
	}
	step_output_fold(&output,
		replica_enter_new_epoch(replica, new_configuration, new_active_count, now))
	return output
}

// The OLD primary's Commit to the other old replicas at handoff (§7.1 step 3): a heartbeat-shaped
// Commit stamped with the still-held old epoch and view (not the new ones the swap below adopts),
// carrying the commit number that now includes the reconfiguration op, so each old replica commits
// and executes it and self-advances into the new epoch. The old group is replica.Configuration,
// which has not yet been swapped for the new one.
func replica_old_group_commit(replica *Replica) (messages []Message) {
	for _, identifier := range replica.Configuration {
		if identifier == replica.Identifier {
			continue
		}
		messages = append(messages, Message{
			Kind:   Message_Kind_Commit,
			From:   replica.Identifier,
			To:     identifier,
			View:   replica.View,
			Epoch:  replica.Epoch,
			Commit: replica.Commit,
		})
	}
	return messages
}

// The OLD primary's Start_Epoch to the ADDED nodes (§7.1 step 3): those in the new configuration
// but not the old one (replica.Configuration, not yet swapped). Each carries the new epoch, the
// epoch-start op-number, and both configurations, so a brand-new node knows where to fetch state
// (the old group) and how far to catch up.
func replica_start_epoch_messages(
	replica *Replica, new_configuration Configuration, new_active_count uint8,
) (messages []Message) {
	for _, identifier := range new_configuration {
		if configuration_contains(replica.Configuration, identifier) {
			continue // Already a member; it learns the new epoch by executing the op.
		}
		messages = append(messages, Message{
			Kind:              Message_Kind_Start_Epoch,
			From:              replica.Identifier,
			To:                identifier,
			View:              0,
			Epoch:             replica.Epoch + 1,
			Op:                replica.Epoch_Start_Op,
			Old_Configuration: replica.Configuration,
			New_Configuration: new_configuration,
			New_Active_Count:  new_active_count,
		})
	}
	return messages
}

// How many heartbeats the new epoch's primary re-drives the reconfiguration commit to the old epoch
// (§7.1.1), so a member that missed the one-shot commit catches up rather than stranding recovery
// (simulator_bugs.md, Bug 19).
const handoff_commit_redrive = 16

// Applies a replica's role in the new epoch after the reconfiguration executed (§7.1.1, §7.1.2). It
// advances the epoch and resets to view 0 (the new epoch must start in view 0, §8.3, so Start_Epoch
// messages from old replicas at differing views cannot install two primaries). A member of the new
// group adopts the new configuration and, being already caught up to the epoch start, returns to
// normal and announces Epoch_Started to the replaced replicas. A replica being replaced keeps the
// old configuration (its current Configuration) in Old_Configuration, adopts the new one, and stays
// transitioning to serve the new group its state until f'+1 Epoch_Started arrive.
func replica_enter_new_epoch(
	replica *Replica, new_configuration Configuration, new_active_count uint8, now time.Moment,
) (output Step_Output) {
	old_configuration := replica.Configuration
	// Whether this replica was a standby in the OLD configuration, read before the swap below:
	// a standby promoted to active by the reconfiguration must materialize the application
	// state it never built (§6.1).
	was_standby := replica_is_standby(replica)
	replica.Epoch++
	replica.View = 0
	replica.Last_Normal_View = 0
	replica.Start_View_Change_From = map[Replica_Identifier]bool{}
	replica.Do_View_Change_From = map[Replica_Identifier]Message{}
	replica.View_Change_Deferred = false
	replica.Epoch_Started_From = map[Replica_Identifier]bool{}
	replica_reset_primary_scratch(replica)
	replica.Configuration = new_configuration
	replica.Active_Count = new_active_count
	// A standby promoted to active by this reconfiguration must materialize application state
	// it never built (§6.1); flag it so the rebuild fires once, when it returns to normal
	// below.
	if was_standby {
		if configuration_contains(new_configuration, replica.Identifier) {
			if !replica_is_standby(replica) {
				replica.Standby_Promotion_Due = true
			}
		}
	}
	if configuration_contains(new_configuration, replica.Identifier) {
		// A member of the new group, already holding the log to the epoch start: return to
		// normal in view 0 and tell the replaced replicas the epoch has started so they may
		// eventually shut down. Arm the commit re-drive so the primary keeps telling the
		// old group the reconfiguration committed, in case the commit was lost (Bug 19).
		replica.Old_Configuration = nil
		replica.Status = Status_Normal
		replica.Handoff_Commit_Redrive = handoff_commit_redrive
		// Mark the CONTINUING members up: present in both groups, they already hold the
		// epoch's log and need no Start_Epoch. Only a member ADDED by this reconfiguration
		// is left unconfirmed, so the heartbeat re-drives Start_Epoch to just those (§7.2).
		for _, identifier := range old_configuration {
			if configuration_contains(new_configuration, identifier) {
				replica.Epoch_Up_From[identifier] = true
			}
		}
		committed, replies := replica_consume_standby_promotion(replica)
		output.Committed = append(output.Committed, committed...)
		output.Replies = append(output.Replies, replies...)
		output.Messages = replica_epoch_started_messages(replica, old_configuration)
		output.Timer = replica_arm_timer(replica, now)
		return output
	}
	// A replica being replaced: keep the old configuration to serve state transfer and stay
	// transitioning until f'+1 Epoch_Started.
	replica.Old_Configuration = old_configuration
	replica.Status = Status_Transition
	output.Timer = replica_arm_timer(replica, now)
	return output
}

// A caught-up new-group member's Epoch_Started to every replica being replaced (§7.1.1): those in
// the old configuration but not the new one. Each carries the new epoch and this replica's
// identifier, a vote toward the f'+1 a replaced replica needs before it shuts down.
func replica_epoch_started_messages(
	replica *Replica, old_configuration Configuration,
) (messages []Message) {
	for _, identifier := range old_configuration {
		if configuration_contains(replica.Configuration, identifier) {
			continue // Still a member of the new group; not being replaced.
		}
		messages = append(messages, Message{
			Kind:  Message_Kind_Epoch_Started,
			From:  replica.Identifier,
			To:    identifier,
			View:  replica.View,
			Epoch: replica.Epoch,
		})
	}
	return messages
}

// The epoch-precedence gate (§7.2): epoch dominates view. It runs first on every message and
// returns whether the caller should proceed to the kind handler, plus any output the gate itself
// produced. A message at the replica's own epoch proceeds. A message at a LOWER epoch is discarded
// and the sender informed with a New_Epoch redirect (a client request's redirect addresses the
// client, §7.4; a replica message's addresses the sender). A message at a HIGHER epoch is adopted
// when it carries what adoption needs (a Start_Epoch, which carries the configurations), and
// otherwise dropped — a replica cannot move to an epoch whose membership it has not learned. A
// shutdown replica processes nothing. An Epoch_Started rides at the replica's own (new) epoch, so
// it passes the equal-epoch gate like any other message.
func replica_message_epoch_check(
	replica *Replica, message Message, now time.Moment,
) (output Step_Output, proceed bool) {
	// A replaced replica that has shut down is done — EXCEPT a later reconfiguration may re-add
	// its identifier, in which case a Start_Epoch naming it in the new configuration revives it
	// as a fresh member (§7.2 brings up the new group; the adopt path catches it up). Any other
	// message to a shut-down replica is still ignored.
	if replica.Status == Status_Shutdown {
		if message.Kind == Message_Kind_Start_Epoch {
			if configuration_contains(message.New_Configuration, replica.Identifier) {
				return replica_receive_start_epoch(replica, message, now), false
			}
		}
		return output, false
	}
	// Encode epoch precedence at the gate: a message below the replica's epoch must never be
	// processed normally. The Sometimes axes witness the {stale, processed-normally} outcomes
	// and the Impossible forbids their co-occurrence — a stale message slipping past the gate
	// would trip it.
	stale := invariant.Sometimes(message.Epoch < replica.Epoch,
		"message epoch below replica epoch")
	processed_normally := invariant.Sometimes(message.Epoch == replica.Epoch,
		"message epoch matches replica epoch")
	invariant.Dot_Product(
		"vsr.epoch_gate.precedence",
		stale, processed_normally,
		invariant.Impossible(
			invariant.Event_True("message epoch below replica epoch"),
			invariant.Event_True("message epoch matches replica epoch")),
	)
	if message.Epoch == replica.Epoch {
		return output, true
	}
	if message.Epoch < replica.Epoch {
		return replica_redirect_stale_epoch(replica, message), false
	}
	// A higher epoch: adopt it from a Start_Epoch (carries both configurations) or a New_Epoch
	// redirect (carries the new configuration, §7.2/§7.4). A New_Epoch is how an advanced
	// replica tells a stranded one — notably one recovering at a stale epoch — that the group
	// has moved, so it can stop being stuck behind and recover against the new group rather
	// than rejoining an emptied old-epoch remnant and breaking quorum intersection. Any OTHER
	// higher-epoch message leaves this replica behind until a Start_Epoch or New_Epoch reaches
	// it; dropping it is safe, since it is not stale — there is no sender to redirect.
	if message.Kind == Message_Kind_Start_Epoch {
		return replica_receive_start_epoch(replica, message, now), false
	}
	if message.Kind == Message_Kind_New_Epoch {
		return replica_receive_new_epoch(replica, message, now), false
	}
	return output, false
}

// Receives a New_Epoch redirect (§7.2, §7.4): an advanced replica's notice that the group has
// moved to a later epoch. A replica not in the new configuration has been replaced and shuts down.
// One that is a member adopts the new epoch and configuration; if it was recovering, it
// re-broadcasts its Recovery at the new epoch so the new group — which alone holds the epoch's
// committed state — answers it, rather than the replica staying stuck at the old epoch and
// rejoining an emptied remnant (the split the simulator surfaced, see simulator_bugs.md). A
// non-recovering member adopts the new epoch and fetches the epoch's state by transfer.
func replica_receive_new_epoch(
	replica *Replica, message Message, now time.Moment,
) (output Step_Output) {
	if message.Epoch <= replica.Epoch {
		// Already at this epoch: redundant redirect, ignore (Bug 19, re-drive).
		return output
	}
	// A replica one epoch behind whose topmost entry is the reconfiguration holds that op
	// (acked it) and only missed the commit. It advances the normal way: commit the op, which
	// drains the handoff into the new epoch, not recovery. This caps the recovering replicas
	// at the f that never acked the op, within fault tolerance, not a deadlock of many
	// recovering at once (Bug 19). The execute_reconfiguration overshoot guard stops a
	// double advance.
	if message.Epoch == replica.Epoch+1 {
		// Already executing the reconfiguration into this epoch: the next tick's drain
		// advances us. A redirect must not preempt that into recovery, stranding a replica
		// mid-handoff with no quorum (Bug 19).
		if replica.Epoch_Handoff_Due {
			output.Timer = replica_arm_timer(replica, now)
			return output
		}
		if replica.Op > replica.Log_Start {
			if len(replica_log_entry(replica, replica.Op).New_Configuration) > 0 {
				output.Committed, output.Replies = replica_apply_commit(
					replica, Commit(replica.Op))
				output.Timer = replica_arm_timer(replica, now)
				return output
			}
		}
	}
	if !configuration_contains(message.New_Configuration, replica.Identifier) {
		replica.Status = Status_Shutdown
		replica.Old_Configuration = nil
		return output
	}
	replica.Epoch = message.Epoch
	replica.Configuration = message.New_Configuration
	replica.Active_Count = message.New_Active_Count
	replica.Old_Configuration = nil
	replica.Standby_Promotion_Due = false
	// A replica that never crashed still holds its committed prefix, so it rejoins by catching
	// up, not quorum-blocking recovery: several behind replicas redirected together would
	// deadlock with no Normal quorum to answer them (Bug 19). A crashed replica lost its state
	// and still needs recovery.
	if replica.Status != Status_Recovery {
		// A never-crashed replica catches up as a Transition member, not via recovery
		// (which blocks on a quorum): it keeps its committed prefix, re-fetches the rest.
		// Only a crash that lost state recovers (Bug 19).
		return replica_rejoin_new_epoch(replica, message.Op, now)
	}
	return replica_recover_into_new_epoch(replica, now)
}

// Rejoins a never-crashed, redirected replica into the new epoch by catching up as a TRANSITION
// member, the path a new group member takes. A Transition member does NOT commit or vote until
// caught up, so a Commit cannot commit a stale entry mid-catch-up (the fork rejoining as Normal
// caused). Recovery is reserved for a crash that lost state; a replica that still holds its state
// must not block on a recovery quorum (Bug 19).
func replica_rejoin_new_epoch(
	replica *Replica, catch_up_target Op, now time.Moment,
) (output Step_Output) {
	// Drop EVERYTHING above this replica's own commit. The committed prefix is agreed by
	// safety, so keeping it is correct; the suffix above is uncommitted and may be a bypassed
	// primary's divergent copy of a differently-committed op. Keeping and later committing that
	// copy forks the log (Bug 19). The catch-up below re-fetches the authoritative ops, so
	// intersection holds without trusting our suffix.
	keep := int(replica.Commit) - int(replica.Log_Start)
	if keep < 0 {
		keep = 0
	}
	if keep < len(replica.Log) {
		replica.Log = replica.Log[:keep]
		replica.Op = Op(replica.Commit)
	}
	// Catch up to the redirector's commit (every committed op) before serving, so this replica
	// never votes in a view change missing one. Epoch_Start_Op drives catch_up_to_epoch and
	// gates the overshoot trigger; the redirector's commit is at or above the epoch-start op.
	replica.Epoch_Start_Op = catch_up_target
	replica.View = 0
	replica.Last_Normal_View = 0
	replica.Status = Status_Transition
	replica.Start_View_Change_From = map[Replica_Identifier]bool{}
	replica.Do_View_Change_From = map[Replica_Identifier]Message{}
	replica.View_Change_Deferred = false
	replica.Epoch_Started_From = map[Replica_Identifier]bool{}
	replica_reset_primary_scratch(replica)
	return replica_catch_up_to_epoch(replica, now)
}

// Enters recovery for a crash-restarted replica redirected into the new epoch. Recovery's quorum
// guarantees the authority holds every committed op; this does NOT wipe op, commit, log, or the
// checkpoint/Executed marker (a warm restart's prefix is valid and recovery executes only beyond
// Executed; re-executing it would double-apply, Bug 9). It keeps its nonce so earlier responses
// still count.
func replica_recover_into_new_epoch(replica *Replica, now time.Moment) (output Step_Output) {
	replica.Status = Status_Recovery
	replica.Recovery_From = map[Replica_Identifier]Message{}
	replica_reset_primary_scratch(replica)
	output.Messages = replica_broadcast(replica, Message{
		Kind:          Message_Kind_Recovery,
		Nonce:         replica.Recovery_Nonce,
		Checkpoint_Op: replica.Checkpoint_Op,
	})
	output.Timer = replica_arm_timer(replica, now)
	return output
}

// Redirects a sender at a stale epoch to the current one (§7.2, §7.4). A client request is answered
// to the client with the new configuration so the client can find the group that moved (§7.4); any
// other sender is a replica, informed with a New_Epoch addressed to it. The redirect carries the
// replica's current view and the new (now-current) configuration.
func replica_redirect_stale_epoch(replica *Replica, message Message) (output Step_Output) {
	redirect := Message{
		Kind:              Message_Kind_New_Epoch,
		From:              replica.Identifier,
		View:              replica.View,
		Epoch:             replica.Epoch,
		New_Configuration: replica.Configuration,
		New_Active_Count:  replica.Active_Count,
		// Op carries the new epoch's start op (the reconfiguration op): a never-crashed
		// replica redirected here catches up to it as a Transition member before serving.
		Op:     replica.Epoch_Start_Op,
		Commit: replica.Commit,
	}
	is_client := message.Kind == Message_Kind_Request ||
		message.Kind == Message_Kind_Reconfiguration ||
		message.Kind == Message_Kind_Check_Epoch
	if is_client {
		redirect.Client = message.Client
		output.Replies = []Message{redirect}
		return output
	}
	redirect.To = message.From
	output.Messages = []Message{redirect}
	return output
}

// Turns an administrator's Reconfiguration into the epoch's final log entry (§7.1). The primary
// accepts it only at its own epoch (the gate already ensured that), when the request-number is
// fresh per the client-table, and when the new configuration has at least three members — VR's
// minimum.
// It appends the entry carrying the new configuration, broadcasts the Prepare, and thereafter
// refuses client requests: replica_open_to_requests reads the topmost entry and sees the
// reconfiguration, so the request path turns new commands away without any extra flag.
func replica_receive_reconfiguration(replica *Replica, message Message) (output Step_Output) {
	if replica.Status != Status_Normal {
		return output
	}
	if replica.Identifier != replica_primary_identifier(replica) {
		return output
	}
	if len(message.New_Configuration) < 3 {
		return output // Below VR's minimum group size; the reconfiguration is rejected.
	}
	record, ok := replica.Client_Table[message.Client]
	if ok {
		if message.Request_Number <= record.Request_Number {
			return output // Stale or in-flight per the client-table; not re-appended.
		}
	}
	// Flush any client requests buffered for batching first, so they are ordered ahead of the
	// reconfiguration; the reconfiguration is then appended alone as the epoch's last op
	// (§7.1), never inside a batch, and broadcast immediately. Thereafter
	// replica_open_to_requests reads the topmost entry, sees the reconfiguration, and turns new
	// client requests away.
	output = replica_flush_batch(replica)
	replica.Client_Table[message.Client] = Client_Record{
		Request_Number: message.Request_Number,
		Executed:       false,
	}
	replica.Request_Buffer = []Log_Entry{{
		View:              replica.View,
		Command:           message.Command,
		Client:            message.Client,
		Request_Number:    message.Request_Number,
		New_Configuration: message.New_Configuration,
		New_Active_Count:  message.New_Active_Count,
	}}
	// Drop any in-flight prediction rounds opened before the reconfiguration: they would
	// otherwise complete after it and append a client op above it, an op at the old epoch past
	// its final op, a false commit the new epoch never holds (Bug 18/Bug 19). Those clients
	// retry and the new epoch's primary runs them afresh.
	replica.Predict_From = map[Prediction_Round_Key]Prediction_Round{}
	step_output_fold(&output, replica_flush_batch(replica))
	return output
}

// Receives a Start_Epoch (§7.1.1): the new group's notice that an epoch has begun. A replica
// already active in this epoch answers a (duplicate) Start_Epoch with Epoch_Started, so the sender
// — a replaced replica resending because it has not yet shut down (§7.1.2) — counts it. Otherwise
// the replica records the old and new configurations, the new epoch and start op, resets to view 0,
// and enters Transitioning, then either catches up by state transfer (its log is short of the epoch
// start) or, already holding the log, completes the epoch immediately.
func replica_receive_start_epoch(
	replica *Replica, message Message, now time.Moment,
) (output Step_Output) {
	if replica.Epoch == message.Epoch {
		if replica.Status == Status_Normal {
			// Already caught up and serving in this epoch: re-acknowledge so the
			// resending replaced replica can reach its f'+1 and shut down.
			output.Messages = []Message{{
				Kind:  Message_Kind_Epoch_Started,
				From:  replica.Identifier,
				To:    message.From,
				View:  replica.View,
				Epoch: replica.Epoch,
			}}
			return output
		}
		// Still transitioning in this epoch; the in-flight catch-up will finish.
		return output
	}
	if message.Epoch < replica.Epoch {
		return output // The gate already filters these; defensive.
	}
	// A recovering old replica informed of the next epoch (§7.2): if it is not a member of the
	// new group it shuts down — it is being replaced and its volatile state is gone anyway;
	// otherwise it adopts the new epoch's configuration and continues recovery against the new
	// group, the nonce-protected rejoin staying the single way its wiped log is restored
	// (recovery quiescence, Bug 1).
	if replica.Status == Status_Recovery {
		if !configuration_contains(message.New_Configuration, replica.Identifier) {
			replica.Status = Status_Shutdown
			return output
		}
		replica.Epoch = message.Epoch
		replica.Configuration = message.New_Configuration
		replica.Active_Count = message.New_Active_Count
		return output
	}
	// Whether this replica was a standby in the OLD configuration, read before the swap below.
	was_standby := replica_is_standby(replica)
	replica.Epoch = message.Epoch
	replica.Epoch_Start_Op = message.Op
	replica.Configuration = message.New_Configuration
	replica.Active_Count = message.New_Active_Count
	replica.Old_Configuration = message.Old_Configuration
	replica.View = 0
	replica.Last_Normal_View = 0
	replica.Status = Status_Transition
	replica.Start_View_Change_From = map[Replica_Identifier]bool{}
	replica.Do_View_Change_From = map[Replica_Identifier]Message{}
	replica.View_Change_Deferred = false
	replica.Epoch_Started_From = map[Replica_Identifier]bool{}
	replica_reset_primary_scratch(replica)
	// A standby promoted to active here must materialize the application state it never built
	// (§6.1); flag it so the rebuild fires once when it returns to normal in complete-epoch,
	// whether it completes locally or after a state-transfer catch-up.
	if was_standby {
		if !replica_is_standby(replica) {
			replica.Standby_Promotion_Due = true
		}
	}
	return replica_catch_up_to_epoch(replica, now)
}

// Brings a transitioning new-group member up to the start of the epoch (§7.1.1 step 2-3). When its
// log already reaches the epoch-start op it completes the epoch now; otherwise it fetches the
// missing log — checkpoint path included for a brand-new node with an empty log — from the old
// replicas (which serve the new group during the handoff) and the other new replicas, staying
// transitioning until a New_State carries it to the epoch start.
func replica_catch_up_to_epoch(replica *Replica, now time.Moment) (output Step_Output) {
	// Catch up to the epoch start AND to every committed op known so far (Commit tracks the
	// responder's commit). Completing at only the epoch-start op leaves the replica serving and
	// voting while behind the frontier; if a quorum of such short replicas then runs a view
	// change, it merges to the short log and drops a committed op longer ones held (Bug 19).
	target := replica.Epoch_Start_Op
	if Op(replica.Commit) > target {
		target = Op(replica.Commit)
	}
	if replica.Op >= target {
		return replica_complete_epoch(replica, now)
	}
	output.Messages = replica_epoch_state_transfer(replica)
	output.Timer = replica_arm_timer(replica, now)
	return output
}

// Sends a Get_State to every replica that can supply the epoch's log — the old replicas (named by
// Old_Configuration, which serve the new group during the handoff) and the other new replicas —
// asking from this replica's own op so the responder ships only what it lacks. A brand-new node
// asks from op 0, so the responder falls back to its checkpoint when the prefix was
// garbage-collected.
func replica_epoch_state_transfer(replica *Replica) (messages []Message) {
	sources := map[Replica_Identifier]bool{}
	for _, identifier := range replica.Old_Configuration {
		sources[identifier] = true
	}
	for _, identifier := range replica.Configuration {
		sources[identifier] = true
	}
	delete(sources, replica.Identifier)
	// Iterate the configurations in order, not the map, so the messages are emitted
	// deterministically; the set above only deduplicates an identifier in both groups.
	emitted := map[Replica_Identifier]bool{}
	for _, identifier := range append(
		append(Configuration{}, replica.Old_Configuration...), replica.Configuration...) {
		if identifier == replica.Identifier {
			continue
		}
		if emitted[identifier] {
			continue
		}
		emitted[identifier] = true
		messages = append(messages, Message{
			Kind:  Message_Kind_Get_State,
			From:  replica.Identifier,
			To:    identifier,
			View:  replica.View,
			Epoch: replica.Epoch,
			Op:    replica.Op,
		})
	}
	return messages
}

// Completes the epoch on a caught-up new-group member (§7.1.1 step 3): it returns to normal in view
// 0, executes any committed-but-unexecuted ops, and announces Epoch_Started to the replaced
// replicas. Whether it is the new epoch's primary follows from the new configuration and view 0, so
// no extra flag gates request acceptance — replica_receive_request already routes only to the
// current primary.
func replica_complete_epoch(replica *Replica, now time.Moment) (output Step_Output) {
	old_configuration := replica.Old_Configuration
	replica.Status = Status_Normal
	replica.Old_Configuration = nil
	// A standby promoted to active by the reconfiguration materializes its application state
	// here, once, before serving (§6.1); a replica that was already active just executes the
	// catch-up.
	if replica.Standby_Promotion_Due {
		output.Committed, output.Replies = replica_consume_standby_promotion(replica)
	} else {
		output.Committed, output.Replies = replica_execute_to_commit(replica)
	}
	output.Messages = replica_epoch_started_messages(replica, old_configuration)
	output.Timer = replica_arm_timer(replica, now)
	return output
}

// Receives an Epoch_Started from a caught-up new-group member (§7.1.2). A replica being replaced
// tallies these by sender and, once it holds f'+1 — the new group's quorum, so the group can
// process requests without it — shuts down: its state is safely transferred and it is no longer
// needed. A replica that is not being replaced (a retained member, or one not transitioning) has
// nothing to do with it.
func replica_receive_epoch_started(replica *Replica, message Message) (output Step_Output) {
	// A member already normal in this epoch — the new primary in particular — records that the
	// sender is active, which ends the heartbeat's Start_Epoch re-drive to it (§7.2). The
	// epoch-precedence gate has already discarded any stale-epoch Epoch_Started.
	if replica.Status == Status_Normal {
		replica.Epoch_Up_From[message.From] = true
		return output
	}
	if replica.Status != Status_Transition {
		return output
	}
	// Only a replica being replaced — handing off, so not a member of the new group — counts
	// Epoch_Started toward shutdown. A transitioning new-group member is itself catching up.
	if configuration_contains(replica.Configuration, replica.Identifier) {
		return output
	}
	replica.Epoch_Started_From[message.From] = true
	if len(replica.Epoch_Started_From) < replica_new_group_quorum(replica) {
		return output
	}
	// The new group's quorum is up: it can serve without this one, so the handoff is complete
	// and the replaced replica shuts down.
	replica.Status = Status_Shutdown
	return output
}

// Checks the per-replica safety invariants that must hold after every step. These are local — a
// replica verifies them from its own state — unlike agreement and single-primary, which span the
// cluster and so live in the simulator. A violation is a Dot_Product panic, failing fast in any
// embedding, not only under test.
func replica_assert_safety(replica *Replica) {
	// A replica being replaced adopts the new configuration it is handing off to (§7.1.2), so
	// it is legitimately NOT a member of its own Configuration while it serves the new group.
	// The membership invariant therefore holds only for a replica still serving its own group
	// — one that is not in a replaced-and-handing-off state. A shutdown replica is out of the
	// protocol entirely.
	serves_own_group := configuration_contains(replica.Configuration, replica.Identifier)
	// Old_Configuration is populated only across an epoch handoff, where the status is
	// Transitioning or Shutdown; a non-empty Old_Configuration outside those is a leak.
	handoff := replica.Status == Status_Transition || replica.Status == Status_Shutdown
	old_configuration_within_handoff := len(replica.Old_Configuration) == 0 || handoff
	// Log_Start must never sit above the checkpoint op: the ops between would be in neither the
	// log nor the checkpoint, a gap that walks execution into an absent op. A 0 Log_Start means
	// no compaction yet (no checkpoint either).
	log_start_below_checkpoint := replica.Log_Start == 0 ||
		replica.Log_Start <= replica.Checkpoint_Op
	// The active prefix must fit inside the configuration: the primary is Configuration[View
	// mod Active_Count], so an Active_Count above the membership would index out of range and
	// could select a non-existent primary. Zero is the no-standby default and is always valid.
	active_count_within_configuration := int(replica.Active_Count) <= len(replica.Configuration)
	// Eager guards: in the hot path (all hold) each is a bare boolean test with no
	// caller-frame lookup — the site is computed only on a violation — so this runs cheaply
	// on every step.
	invariant.Always(int(replica.Op) == int(replica.Log_Start)+len(replica.Log),
		"op equals log start plus log length")
	invariant.Always(replica.Commit <= Commit(replica.Op), "commit does not exceed op")
	invariant.Always(replica.Last_Normal_View <= replica.View,
		"last normal view does not exceed current view")
	// A 2f+1 group needs at least three members to tolerate one fault.
	invariant.Always(len(replica.Configuration) >= 3,
		"configuration has at least three members")
	// A replica must be a member of the configuration it serves, or it could never be a
	// primary — unless it is a replaced replica handing off, exempted above.
	invariant.Always(serves_own_group || handoff,
		"replica serves its own group unless handing off")
	invariant.Always(old_configuration_within_handoff,
		"old configuration set only during handoff")
	invariant.Always(log_start_below_checkpoint, "log start does not exceed checkpoint op")
	invariant.Always(active_count_within_configuration,
		"active count fits within configuration")
}

// Reports whether identifier is a member of configuration.
func configuration_contains(
	configuration Configuration, identifier Replica_Identifier,
) (yes bool) {
	for _, member := range configuration {
		if member == identifier {
			return true
		}
	}
	return false
}

// Configurations_Equal_Input is the pair of configurations a membership comparison checks.
type configurations_equal_input struct {
	A Configuration
	B Configuration
}

// Reports whether two configurations are the identical membership in the same order.
func configurations_equal(input *configurations_equal_input) (yes bool) {
	if len(input.A) != len(input.B) {
		return false
	}
	for index := range input.A {
		if input.A[index] != input.B[index] {
			return false
		}
	}
	return true
}

// Turns a client command into a new log entry on the primary, deduplicated by the client-table for
// exactly-once semantics. Only the primary of the current view, and only while normal, may append.
// A request the client-table has already seen is never re-appended: a stale one is dropped, an
// in-flight one is dropped (its op will reply when it commits), and a duplicate of an
// already-executed request is answered from the cache (§4.5, a client that lost its reply).
func replica_receive_request(replica *Replica, message Message) (output Step_Output) {
	if replica.Status != Status_Normal {
		return output
	}
	if replica.Identifier != replica_primary_identifier(replica) {
		return output
	}
	if !replica_open_to_requests(replica) {
		return output // The epoch's reconfiguration is its last op; no new client requests.
	}
	record, ok := replica.Client_Table[message.Client]
	if ok {
		if message.Request_Number < record.Request_Number {
			return output // Stale: a newer request already superseded it.
		}
		if message.Request_Number == record.Request_Number {
			if record.Executed {
				reply := replica_build_reply(replica, message.Client, record)
				output.Replies = []Message{reply}
				return output // Cached reply re-sent for an executed request.
			}
			// Recorded as latest but not executed: in flight only while still in
			// the log. A view change can drop the uncommitted entry but keep the
			// flush-written record, stranding it: recorded so never re-accepted,
			// unlogged so never committed. Gone from the log, re-accept it below
			// (Bug 20); the buffer/prediction checks still dedup a live retry.
			if replica_request_logged(replica, message.Client, message.Request_Number) {
				return output
			}
		}
	}
	// The client-table records a request only once it reaches the log (at flush). One accepted
	// into the batch buffer or a prediction round but not yet logged is in flight without a
	// record, so dedup it here; else re-buffering commits one request at two ops. A request the
	// buffer was cleared of on a view change is in neither, so its retry is re-accepted afresh:
	// never logged, it cannot have committed (Bug 20).
	if replica_request_buffered(replica, message.Client, message.Request_Number) {
		return output
	}
	// An accepted request never regresses the client's number: the early stale return turned
	// away anything below the record, leaving a strictly newer request or a re-accept of the
	// client's latest one that a view change dropped from the log (equal number, Bug 20).
	does_not_regress := !ok || message.Request_Number >= record.Request_Number
	invariant.Always(does_not_regress, "accepted request does not regress request number")
	// The pre-step variant cannot append yet: the executed value must combine the cluster's
	// predictions, so it first gathers f backups' predictions (§4.4). The request is buffered
	// and a Predict_Request broadcast; the append happens when f responses arrive.
	if replica.State_Machine.Combine != nil {
		return replica_begin_pre_step(replica, message)
	}
	// Local-predict (the default): the primary computes the value itself, stamps it on the
	// entry, and accepts it for batching now. With no Predict the prediction stays empty — a
	// deterministic op.
	return replica_accept_entry(replica, Log_Entry{
		View:           replica.View,
		Command:        message.Command,
		Client:         message.Client,
		Request_Number: message.Request_Number,
		Prediction:     replica_local_prediction(replica, message.Command),
	})
}

// Reports whether the client's request-number is already in this primary's batch buffer: accepted
// but not yet logged, so it has no client-table record yet (written at flush). Dedups a retry so it
// is not buffered and committed twice. A request the buffer was cleared of on a view change is not
// here, so its retry is re-accepted (Bug 20). A request still in a prediction round is deliberately
// NOT reported: its retry must fall through to replica_begin_pre_step, which re-broadcasts the
// Predict_Request so a round whose request or responses were lost still completes rather than
// wedging the client (begin_pre_step is idempotent and never opens a second round).
func replica_request_buffered(
	replica *Replica, client Client_Identifier, number Request_Number,
) (buffered bool) {
	for _, entry := range replica.Request_Buffer {
		if entry.Client != client {
			continue
		}
		if entry.Request_Number == number {
			return true
		}
	}
	return false
}

// Reports whether the request (client, number) is present in this replica's live log. A request
// the client-table records as the client's latest-but-unexecuted may have been dropped from the
// log by a view change (the record, written at flush, outlives the entry); only checking the log
// tells an in-flight request from one the primary must re-accept (Bug 20).
func replica_request_logged(
	replica *Replica, client Client_Identifier, number Request_Number,
) (logged bool) {
	for _, entry := range replica.Log {
		if entry.Client != client {
			continue
		}
		if entry.Request_Number == number {
			return true
		}
	}
	return false
}

// Computes the primary's local prediction for a command, or empty when no Predict is injected (an
// ordinary deterministic op needs none).
func replica_local_prediction(replica *Replica, command []byte) (prediction []byte) {
	if replica.State_Machine.Predict == nil {
		return nil
	}
	return replica.State_Machine.Predict(command)
}

// Accepts one request's entry — already carrying its predetermined prediction — buffering it for
// batching (§6.2), then flushing the batch when load-adaptive batching says to. Shared by the
// local-predict path and the pre-step path, which differ only in how the entry's prediction was
// computed; both arrive here with the entry built. The in-flight client-table record is NOT written
// here: a buffered request the buffer is later cleared of on a view change would otherwise orphan
// its record and wedge the client's retries (Bug 20). The record is written at flush, when the
// request reaches the log; until then a retry is deduped by replica_request_buffered.
func replica_accept_entry(replica *Replica, entry Log_Entry) (output Step_Output) {
	replica.Request_Buffer = append(replica.Request_Buffer, entry)
	return replica_maybe_flush_batch(replica)
}

// Flushes the buffered batch when load-adaptive batching calls for it (§6.2): when the primary is
// idle — its commit number has caught its op, so no round is in flight and latency, not throughput,
// dominates — or when the buffer has reached Batch_Max. A zero or one Batch_Max means no batching:
// every accepted request flushes at once (a batch of one), the behavior before batching existed.
func replica_maybe_flush_batch(replica *Replica) (output Step_Output) {
	if len(replica.Request_Buffer) == 0 {
		return output
	}
	limit := replica.Batch_Max
	if limit == 0 {
		limit = 1
	}
	idle := replica.Op == Op(replica.Commit)
	if idle {
		return replica_flush_batch(replica)
	}
	if uint64(len(replica.Request_Buffer)) >= limit {
		return replica_flush_batch(replica)
	}
	return output
}

// Appends the buffered batch to the log as consecutive ops and broadcasts one Prepare carrying the
// whole batch (§6.2). The Prepare's Op is the batch's highest op and Entries its entries in op
// order; a receiver derives the batch's first op as Op minus the entry count plus one. The
// in-flight client-table records were written as each entry was accepted, so flushing only appends
// and sends.
func replica_flush_batch(replica *Replica) (output Step_Output) {
	if len(replica.Request_Buffer) == 0 {
		return output
	}
	batch := replica.Request_Buffer
	replica.Request_Buffer = nil
	replica.Log = append(replica.Log, batch...)
	replica.Op = replica.Log_Start + Op(len(replica.Log))
	// Record each request as in-flight now that it is in the log, not at acceptance, so a batch
	// dropped on a view change before reaching here leaves no orphan record to wedge its client
	// (Bug 20). The commit-time execute later marks the record executed with its result.
	for _, entry := range batch {
		replica.Client_Table[entry.Client] = Client_Record{
			Request_Number: entry.Request_Number,
			Executed:       false,
		}
	}
	// To every backup, standbys included (§6.1): a standby replicates the log so it stays
	// current and can be promoted, though it never votes or executes. This is TigerBeetle's
	// design — it replicates to its non-voting standbys; the paper's "witnesses uninvolved in
	// the normal case" message-reduction is the cost we deliberately trade away for the
	// simpler, safer non-voting standby.
	output.Messages = replica_broadcast(replica, Message{
		Kind:    Message_Kind_Prepare,
		Op:      replica.Op,
		Entries: batch,
		Commit:  replica.Commit,
	})
	return output
}

// Opens the pre-step prediction round for a buffered request: it broadcasts a Predict_Request to
// the backups and records an empty tally keyed by client and request-number. A re-sent request for
// a round already open just re-broadcasts, leaving the tally intact, so a lost Predict_Request can
// be retried without losing collected responses.
func replica_begin_pre_step(replica *Replica, message Message) (output Step_Output) {
	key := Prediction_Round_Key{
		Client:         message.Client,
		Request_Number: message.Request_Number,
	}
	_, ok := replica.Predict_From[key]
	if !ok {
		replica.Predict_From[key] = Prediction_Round{
			Command:   message.Command,
			Responses: map[Replica_Identifier][]byte{},
		}
	}
	output.Messages = replica_broadcast(replica, Message{
		Kind:           Message_Kind_Predict_Request,
		Command:        message.Command,
		Client:         message.Client,
		Request_Number: message.Request_Number,
	})
	return output
}

// Answers a primary's Predict_Request with this backup's prediction for the command (§4.4
// pre-step). Only a normal replica in the primary's view answers; a non-normal or behind replica
// stays silent, and the pre-step round waits for f replies from the backups that are current. With
// no Predict injected the response is empty, which Combine must tolerate.
func replica_receive_predict_request(replica *Replica, message Message) (output Step_Output) {
	if replica.Status != Status_Normal {
		return output
	}
	if message.View != replica.View {
		return output
	}
	output.Messages = []Message{{
		Kind:           Message_Kind_Predict_Response,
		From:           replica.Identifier,
		To:             message.From,
		View:           replica.View,
		Epoch:          replica.Epoch,
		Client:         message.Client,
		Request_Number: message.Request_Number,
		Result:         replica_local_prediction(replica, message.Command),
	}}
	return output
}

// Tallies one backup's Predict_Response on the primary and, once f distinct backups have answered a
// pending request, combines their predictions with the primary's own, appends the request, and
// broadcasts the Prepare (§4.4 pre-step). A response for a round already completed, or for one
// this replica is not running, is dropped; a duplicate from the same sender is idempotent, filed by
// sender so it cannot inflate the count.
// Reports whether a completed prediction round for (client, number) may still be appended. The
// round opened optimistically and the cluster can move on while predictions are gathered, so the
// deferred append re-checks what replica_receive_request checks synchronously: the epoch still
// accepts requests, no strictly newer request superseded this one, this exact request is not
// already executed, and it is not already in the log (appending it then would be a second op for
// one request). A record at the same number but NOT executed is this very in-flight request — its
// flush-written record can outlive an entry a view change dropped — so it must still be appended,
// else the round completes forever without logging the request (a liveness gap).
func replica_pre_step_appendable(
	replica *Replica, client Client_Identifier, number Request_Number,
) (appendable bool) {
	if !replica_open_to_requests(replica) {
		return false
	}
	record, fresh := replica.Client_Table[client]
	if fresh {
		if number < record.Request_Number {
			return false
		}
		if number == record.Request_Number {
			if record.Executed {
				return false
			}
		}
	}
	return !replica_request_logged(replica, client, number)
}

func replica_receive_predict_response(replica *Replica, message Message) (output Step_Output) {
	if replica.Status != Status_Normal {
		return output
	}
	if message.View != replica.View {
		return output
	}
	if replica.Identifier != replica_primary_identifier(replica) {
		return output
	}
	key := Prediction_Round_Key{
		Client:         message.Client,
		Request_Number: message.Request_Number,
	}
	round, ok := replica.Predict_From[key]
	if !ok {
		return output // No open round: already completed, or never ours to run.
	}
	round.Responses[message.From] = message.Result
	// The pre-step round needs f backups' predictions; f = quorum - 1, since the primary adds
	// its own. Below that, keep collecting.
	if len(round.Responses) < replica_quorum(replica)-1 {
		return output
	}
	// The round is complete: a duplicate response after this point finds no open round and is
	// dropped. Re-validate before appending — the round opened optimistically, and the cluster
	// may have moved on while predictions were gathered. If the epoch's reconfiguration has
	// since been appended (no longer open to requests) or the request is no longer fresh,
	// abandon the round without appending; otherwise the pending request would land after the
	// reconfiguration, then commit a second time across the epoch boundary — the duplicate the
	// simulator surfaced. The synchronous local-predict path checks the same conditions in
	// replica_receive_request before it appends; the pre-step path must re-check them here
	// because its append is deferred. The client retries and the new primary runs the request
	// afresh.
	delete(replica.Predict_From, key)
	if !replica_pre_step_appendable(replica, message.Client, message.Request_Number) {
		return output
	}
	// Combine over the collected backups' predictions and the primary's own, then add. Iterate
	// the configuration, not the map, so the responses reach Combine in a deterministic order
	// — a map range would not, and Combine must be a pure function of its inputs.
	responses := make([][]byte, 0, len(round.Responses))
	for _, member := range replica.Configuration {
		response, answered := round.Responses[member]
		if answered {
			responses = append(responses, response)
		}
	}
	own := replica_local_prediction(replica, round.Command)
	prediction := replica.State_Machine.Combine(round.Command, responses, own)
	return replica_accept_entry(replica, Log_Entry{
		View:           replica.View,
		Command:        round.Command,
		Client:         message.Client,
		Request_Number: message.Request_Number,
		Prediction:     prediction,
	})
}

// Builds the primary's cached Reply to a client, addressed by Client (clients are not in the
// Configuration, so To stays zero) and carrying the current view, the request-number, and the
// record's cached result.
func replica_build_reply(
	replica *Replica, client Client_Identifier, record Client_Record,
) (reply Message) {
	return Message{
		Kind:           Message_Kind_Reply,
		View:           replica.View,
		Epoch:          replica.Epoch,
		Client:         client,
		Request_Number: record.Request_Number,
		Result:         record.Result,
	}
}

// Returns the first op of a re-driven Prepare (ops this replica already holds) whose held entry
// differs from the primary's authoritative entry, and whether any such op exists. A difference
// means this replica holds a stale entry (a bypassed primary's, or one a bad state transfer spliced
// in) that the primary's current-view log replaces. A compacted op is committed and cannot differ.
func replica_redriven_divergence(replica *Replica, message Message) (op Op, found bool) {
	base_op := message.Op - Op(len(message.Entries)) + 1
	for candidate := base_op; candidate <= message.Op; candidate++ {
		if candidate <= replica.Log_Start {
			continue
		}
		held := replica_log_entry(replica, candidate)
		offered := message.Entries[candidate-base_op]
		if string(held.Command) != string(offered.Command) {
			return candidate, true
		}
		if string(held.Prediction) != string(offered.Prediction) {
			return candidate, true
		}
	}
	return 0, false
}

// Re-acknowledges a re-driven Prepare whose ops this replica already holds: the primary re-sent it
// because this op's Prepare_Ok was lost, so without a fresh ack the op wedges uncommitted with no
// view change to rescue it (simulator_bugs.md, Bug 17). It re-acks ONLY ops held from the CURRENT
// view (entry view == replica view): a stale entry carried in by state transfer from a bypassed
// primary differs from the primary's entry there, and acking it would let the primary commit on a
// false quorum while this replica commits its own divergent entry — forking the log (the agreement
// violation the simulator surfaced). A standby never votes; ops at or below Log_Start are compacted
// (committed) and need no re-ack, and message.Commit can sit below this replica's Log_Start.
func replica_reack_redriven_prepare(replica *Replica, message Message) (messages []Message) {
	if replica_is_standby(replica) {
		return nil
	}
	first := Op(message.Commit) + 1
	base_op := message.Op - Op(len(message.Entries)) + 1
	if base_op > first {
		first = base_op
	}
	if first <= replica.Log_Start {
		first = replica.Log_Start + 1
	}
	for op := first; op <= message.Op; op++ {
		// The caller reconciled any divergence, so every op here holds the primary's
		// authoritative entry and is safe to acknowledge.
		messages = append(messages, Message{
			Kind:  Message_Kind_Prepare_Ok,
			From:  replica.Identifier,
			To:    replica_primary_identifier(replica),
			View:  replica.View,
			Epoch: replica.Epoch,
			Op:    op,
		})
	}
	return messages
}

// Handles a re-driven Prepare whose ops this replica already holds. If every held op matches the
// primary's authoritative entry it returns the re-acks, so a lost Prepare_Ok cannot wedge the
// primary (Bug 17). If one diverges (a bypassed primary's entry, or one a bad state transfer
// spliced in) it truncates from the divergence and returns fall_through, so the caller appends the
// primary's entries instead — committing the stale entry would fork the log (seed-162).
func replica_redriven_prepare(
	replica *Replica, message Message,
) (acks []Message, fall_through bool) {
	diverge, found := replica_redriven_divergence(replica, message)
	if !found {
		return replica_reack_redriven_prepare(replica, message), false
	}
	if diverge <= Op(replica.Commit) {
		// The Prepare's entry differs from one this replica COMMITTED. Committed ops
		// are immutable, so the sender is a behind primary reusing the op-number for a new
		// value (across a reconfiguration). REJECT it: re-acking would let it commit on a
		// false quorum and fork the log (Bug 19).
		return nil, false
	}
	keep := int(diverge) - 1 - int(replica.Log_Start)
	if keep < 0 {
		keep = 0
	}
	replica.Log = replica.Log[:keep]
	replica.Op = diverge - 1
	return nil, true
}

// Applies a backup's next op and acks it. A Prepare from a later view, or one that skips ahead of
// the next op, means the backup is behind, so it catches up by state transfer rather than dropping
// the message; one that skips an op the log must stay gap-free.
func replica_receive_prepare(
	replica *Replica, message Message, now time.Moment,
) (output Step_Output) {
	if replica.Status != Status_Normal {
		return output
	}
	if message.View < replica.View {
		return output
	}
	if message.View > replica.View {
		return replica_begin_state_transfer(replica, message.View, message.From, now)
	}
	// Same view: hearing from the primary resets the failure detector and commits what we hold.
	output.Timer = replica_arm_timer(replica, now)
	output.Committed, output.Replies = replica_apply_commit(replica, message.Commit)
	if message.Op <= replica.Op {
		acks, fall_through := replica_redriven_prepare(replica, message)
		if !fall_through {
			output.Messages = acks
			return output
		}
		// Divergence reconciled (stale suffix truncated); append the primary's entries.
	}
	// The batch covers ops base_op..message.Op; base_op is the highest op minus the entry count
	// plus one. A batch whose first op is beyond this backup's next op leaves a gap, so the
	// backup catches up by state transfer rather than appending a discontiguous run.
	base_op := message.Op - Op(len(message.Entries)) + 1
	if base_op > replica.Op+1 {
		transfer := replica_begin_state_transfer(replica, replica.View, message.From, now)
		output.Messages = transfer.Messages
		output.Timer = transfer.Timer
		return output
	}
	// Skip the leading entries the backup already holds — a re-driven or overlapping batch can
	// repeat ops at or below the current op — and append only the rest, keeping the log
	// gap-free.
	skip := int(replica.Op + 1 - base_op)
	first_appended := replica.Op + 1
	replica.Log = append(replica.Log, message.Entries[skip:]...)
	replica.Op = replica.Log_Start + Op(len(replica.Log))
	// A backup adopts each Prepare entry verbatim — including its predetermined prediction —
	// and never recomputes it; that verbatim copy is what keeps every replica executing the op
	// against the identical value (§4.4). It holds by construction; pin it on the batch's last
	// entry so a future change that mutates an entry between receipt and append cannot silently
	// break agreement.
	appended_matches := string(replica_log_entry(replica, replica.Op).Prediction) ==
		string(message.Entries[len(message.Entries)-1].Prediction)
	invariant.Always(appended_matches, "appended entry prediction matches the prepare")
	// A standby never votes (TigerBeetle's non-voting-standby design, see replica_quorum): it
	// appends the entries to stay current and serve as a promotable spare, but sends no
	// Prepare_Ok, so it is never counted toward a commit quorum. An active backup acknowledges
	// every op newly appended from this Prepare, one Prepare_Ok each, so the primary tallies an
	// explicit acknowledgement per op — never crediting an op a backup did not append from a
	// Prepare. A batched Prepare yields one ack per entry; a single-op Prepare yields one.
	if replica_is_standby(replica) {
		return output
	}
	for op := first_appended; op <= replica.Op; op++ {
		output.Messages = append(output.Messages, Message{
			Kind:  Message_Kind_Prepare_Ok,
			From:  replica.Identifier,
			To:    replica_primary_identifier(replica),
			View:  replica.View,
			Epoch: replica.Epoch,
			Op:    op,
		})
	}
	return output
}

// Applies the primary's advertised commit number on a backup and resets the failure detector, since
// a Commit is the primary's heartbeat. A Commit from a later view, or one advertising ops the
// backup is missing, triggers state transfer to catch up.
func replica_receive_commit(
	replica *Replica, message Message, now time.Moment,
) (output Step_Output) {
	if replica.Status != Status_Normal {
		return output
	}
	if message.View < replica.View {
		return output
	}
	if message.View > replica.View {
		return replica_begin_state_transfer(replica, message.View, message.From, now)
	}
	output.Timer = replica_arm_timer(replica, now)
	output.Committed, output.Replies = replica_apply_commit(replica, message.Commit)
	if message.Commit > Commit(replica.Op) {
		transfer := replica_begin_state_transfer(replica, replica.View, message.From, now)
		output.Messages = transfer.Messages
		output.Timer = transfer.Timer
	}
	return output
}

// Begins catching a behind-but-not-crashed replica up by asking a more current peer for state. A
// later view means the uncommitted tail may have been reordered by a view change, so it is dropped
// to the committed prefix and the new view adopted before requesting; a same-view gap keeps the log
// and just fetches forward. Truncation never goes below the commit number, so no committed op is
// lost.
func replica_begin_state_transfer(
	replica *Replica, target_view View, source Replica_Identifier, now time.Moment,
) (output Step_Output) {
	if target_view > replica.View {
		replica.View = target_view
		// Drop the uncommitted tail back to the commit number. The committed prefix that
		// remains in Log runs from Log_Start+1 through Commit, so the entries to keep
		// number Commit - Log_Start; the compacted prefix below Log_Start is untouched.
		replica.Log = replica.Log[:int(replica.Commit)-int(replica.Log_Start)]
		replica.Op = Op(replica.Commit)
		replica.Last_Normal_View = target_view
	}
	output.Messages = []Message{{
		Kind:  Message_Kind_Get_State,
		From:  replica.Identifier,
		To:    source,
		View:  replica.View,
		Epoch: replica.Epoch,
		Op:    replica.Op,
	}}
	output.Timer = replica_arm_timer(replica, now)
	return output
}

// Answers a Get_State from a replica that is at least as current, sending the log it lacks so it
// can catch up. When the requester's op is below this replica's Log_Start — the prefix it needs
// has been garbage-collected (§5.2) — the answer instead carries the checkpoint and only the
// suffix after the checkpoint op, since the GC'd prefix no longer exists to ship. A replica still
// changing views answers too: it holds its complete last-normal log, and a new primary deferring an
// install fetches the selected log from it precisely while it sits in view-change (§5.3), so
// refusing here would deadlock that fetch. A recovering replica, whose log was wiped, has nothing
// authoritative to offer and stays silent.
func replica_receive_get_state(replica *Replica, message Message) (output Step_Output) {
	if replica.Status == Status_Recovery {
		return output
	}
	if replica.View < message.View {
		return output
	}
	response := Message{
		Kind:   Message_Kind_New_State,
		From:   replica.Identifier,
		To:     message.From,
		View:   replica.View,
		Epoch:  replica.Epoch,
		Op:     replica.Op,
		Commit: replica.Commit,
	}
	if message.Op < replica.Log_Start {
		// The requester is behind the compacted prefix: ship the checkpoint and the suffix
		// after it. The suffix begins at Checkpoint_Op+1, which is at or above Log_Start
		// since the checkpoint never GCs above Checkpoint_Op - Log_Retain.
		response.Checkpoint_Op = replica.Checkpoint_Op
		response.Checkpoint_State = replica.Checkpoint_State
		response.Checkpoint_Client_Table = clone_client_table(replica.Client_Table)
		response.Log = replica_log_slice_from(replica, replica.Checkpoint_Op+1)
	} else {
		// The requester holds everything up to its op; ship only the entries beyond it. A
		// requester already at or past this replica's op gets an empty suffix — it is no
		// longer behind us, and its own apply guard will keep its longer log.
		from := message.Op + 1
		if from > replica.Op+1 {
			from = replica.Op + 1
		}
		response.Log = replica_log_slice_from(replica, from)
	}
	output.Messages = []Message{response}
	return output
}

// Adopts the state a peer sent in answer to this replica's Get_State, provided it is no older. The
// answer carries a suffix — the entries from message.Op back over its length — which this
// replica splices onto the prefix it already holds. A later view's log already preserves every
// committed op (the view change guaranteed it); a same-view answer is adopted only when it carries
// a strictly higher op, so it never shortens the log. When the answer carries a checkpoint (§5.2
// gap), the replica first Restores from it, sets Log_Start to the checkpoint op, and adopts the
// view in which the checkpoint was taken; then the committed prefix is executed so the application
// state matches.
func replica_receive_new_state(
	replica *Replica, message Message, now time.Moment,
) (output Step_Output) {
	// A new primary that deferred its view-change install awaits exactly this: the selected
	// reporter's complete log (§5.3). Route it to the deferred-install path, which adopts the
	// fetched log wholesale and finally broadcasts the Start_View it withheld.
	if replica.View_Change_Deferred {
		if replica.Status == Status_View_Change {
			return replica_complete_view_change_fetch(replica, message, now)
		}
	}
	// A transitioning new-group member is catching up to the start of the epoch (§7.1.1): it
	// applies the shipped state the same way, then completes the epoch once its log reaches the
	// epoch-start op. This is the path a brand-new node takes from an empty log — checkpoint
	// included — to serving in the new epoch.
	if replica.Status == Status_Transition {
		if message.Op <= replica.Op {
			return output // Nothing newer to adopt; keep waiting for a later answer.
		}
		replica_apply_new_state(replica, message)
		// Take the commit number the responder reported (capped to the ops just adopted),
		// so completing the epoch executes the caught-up committed prefix. Everything up to
		// the epoch start is committed — the reconfiguration op committed in the old group,
		// and every op before it with it.
		commit := message.Commit
		if Commit(replica.Op) < commit {
			commit = Commit(replica.Op)
		}
		if commit > replica.Commit {
			replica.Commit = commit
		}
		return replica_catch_up_to_epoch(replica, now)
	}
	if replica.Status != Status_Normal {
		return output
	}
	if message.View < replica.View {
		return output
	}
	if message.View == replica.View {
		if message.Op <= replica.Op {
			return output
		}
	}
	replica.View = message.View
	replica.Last_Normal_View = message.View
	replica_apply_new_state(replica, message)
	output.Committed, output.Replies = replica_apply_commit(replica, message.Commit)
	output.Timer = replica_arm_timer(replica, now)
	return output
}

// Adopts the log a New_State carries onto this replica: when it ships a checkpoint (the §5.2 gap)
// and the replica is behind it, restore from the checkpoint and rebuild the log from the shipped
// suffix; then splice the suffix onto the held prefix and reconcile the client-table. It leaves
// Commit to the caller, which differs by path (the normal path caps at the message's commit; the
// transitioning path commits up to the epoch start). It is the shared apply body of both
// state-transfer paths.
func replica_apply_new_state(replica *Replica, message Message) {
	if message.Checkpoint_State != nil {
		if message.Checkpoint_Op > replica.Executed {
			replica_restore_checkpoint(
				replica, message.Checkpoint_Op, message.Checkpoint_State,
				message.Checkpoint_Client_Table)
		}
	}
	// Splice the shipped suffix onto the prefix this replica holds: state transfer extends a
	// catch-up log rather than replacing it, since the requester's committed prefix already
	// matches the responder's by agreement.
	replica_splice_suffix(replica, message.Op, message.Log)
	replica_rebuild_client_table(replica)
}

// Completes a deferred view-change install when the awaited New_State arrives (§5.3). It accepts
// only the New_State the deferred fetch is waiting for — from the selected reporter, in this view,
// reaching at least its reported op — so a stray or stale New_State neither installs early nor
// drops a committed op. The carrier is the reporter's complete log (suffix above the committed
// floor the fetch requested, with the checkpoint when even that prefix was GC'd); installing it
// wholesale replaces the new primary's own uncommitted tail with the winner's, then broadcasts the
// Start_View that was withheld while fetching. A non-matching New_State is ignored, leaving the
// fetch outstanding.
func replica_complete_view_change_fetch(
	replica *Replica, message Message, now time.Moment,
) (output Step_Output) {
	if message.From != replica.View_Change_Fetch_From {
		return output
	}
	if message.View != replica.View {
		return output
	}
	if message.Op < replica.View_Change_Fetch_Op {
		return output // The fetched log is short of the winner's op; keep waiting.
	}
	// The fetch asked from this replica's Log_Start, so a faithful answer attaches at or
	// below the committed prefix (carrier_start <= Commit) or ships a checkpoint covering the
	// gap. A New_State attaching ABOVE the commit with no checkpoint is stale or mismatched —
	// typically a reply to an earlier state-transfer Get_State this replica sent as a backup —
	// whose omitted prefix lands on this replica's own UNCOMMITTED entries, which may diverge
	// from the selected log and which adopt cannot restore. Ignoring it keeps the fetch
	// outstanding for the faithful answer rather than installing a gap over a committed op
	// (simulator_bugs.md, Bug 16).
	carrier_start := message.Op - Op(len(message.Log))
	if message.Checkpoint_State == nil {
		if carrier_start > Op(replica.Commit) {
			return output
		}
	}
	return replica_complete_install(replica, message, replica.View_Change_Fetch_Op, now)
}

// Splices a suffix onto the log: suffix is the entries ending at final_op, so they begin at op
// final_op minus the length plus one. The replica keeps the entries it already holds below that
// start and appends the suffix, leaving Op at final_op. The kept count is clamped into the live
// log: a replica more compacted than the carrier (its Log_Start above the suffix start) keeps its
// whole suffix, and only the suffix entries above its current op are appended, so neither end
// indexes out of range.
func replica_splice_suffix(replica *Replica, final_op Op, suffix []Log_Entry) {
	suffix_start := final_op - Op(len(suffix)) + 1
	// Drop suffix entries the replica already holds — those at or below its current op — so
	// a more compacted replica does not re-append ops its checkpoint and log already cover.
	if suffix_start <= replica.Op {
		skip := int(replica.Op - suffix_start + 1)
		if skip > len(suffix) {
			skip = len(suffix)
		}
		suffix = suffix[skip:]
		suffix_start = replica.Op + 1
	}
	keep := int(suffix_start - replica.Log_Start - 1)
	if keep < 0 {
		keep = 0
	}
	kept := replica.Log
	if keep < len(kept) {
		kept = kept[:keep]
	}
	combined := make([]Log_Entry, 0, keep+len(suffix))
	combined = append(combined, kept...)
	combined = append(combined, suffix...)
	replica.Log = combined
	replica.Op = replica.Log_Start + Op(len(replica.Log))
}

// Replaces the log above the shared committed prefix with an authoritative suffix, discarding every
// entry this replica holds beyond that prefix — the wholesale-adoption counterpart to
// replica_splice_suffix. Where splice EXTENDS a catch-up log (state transfer, where the requester's
// own suffix above the carrier is trusted), replace REPLACES it: the suffix comes from the agreed
// log of a view (a Do_View_Change winner, a Start_View, or a Recovery_Response), so it supersedes
// this replica's tail entirely. carrier_start is the op just below the suffix's first entry; the
// shared committed prefix the replica keeps runs through floor, the higher of carrier_start and
// the replica's own Log_Start, since a replica more compacted than the carrier already covers its
// lower entries in its checkpoint. Suffix entries at or below floor are dropped (the
// replica's checkpoint already covers them, so re-appending them would misalign the offset index),
// and the replica's own live entries above floor are discarded, so a stale uncommitted entry from a
// bypassed primary cannot persist into the adopted view (simulator_bugs.md, Bug 9). The kept prefix
// is committed: the caller restricts this path to carrier_start <= Executed, and Log_Start <=
// Checkpoint_Op <= Commit, so floor <= Commit either way.
func replica_replace_from(replica *Replica, carrier_start Op, suffix []Log_Entry) {
	floor := carrier_start
	if replica.Log_Start > floor {
		floor = replica.Log_Start
	}
	// Drop suffix entries at or below floor: the replica holds those in its retained log or
	// checkpoint already, and the appended suffix must begin exactly at floor+1 so the kept
	// prefix and the suffix stay contiguous under the Log_Start offset index.
	suffix_start := carrier_start + 1
	if suffix_start <= floor {
		drop := int(floor - suffix_start + 1)
		if drop > len(suffix) {
			drop = len(suffix)
		}
		suffix = suffix[drop:]
	}
	keep := int(floor - replica.Log_Start)
	if keep < 0 {
		keep = 0
	}
	kept := replica.Log
	if keep < len(kept) {
		kept = kept[:keep]
	}
	combined := make([]Log_Entry, 0, keep+len(suffix))
	combined = append(combined, kept...)
	combined = append(combined, suffix...)
	replica.Log = combined
	replica.Op = replica.Log_Start + Op(len(replica.Log))
}

// Records a backup's acknowledgement and commits as far as the acknowledgements allow. The ack is
// filed by sender so a duplicate cannot inflate the count.
func replica_receive_prepare_ok(replica *Replica, message Message) (output Step_Output) {
	if replica.Status != Status_Normal {
		return output
	}
	if message.View != replica.View {
		return output
	}
	if replica.Identifier != replica_primary_identifier(replica) {
		return output
	}
	if replica.Prepare_Ok_From[message.Op] == nil {
		replica.Prepare_Ok_From[message.Op] = map[Replica_Identifier]bool{}
	}
	replica.Prepare_Ok_From[message.Op][message.From] = true
	replica_advance_commit_as_primary(replica)
	output.Committed, output.Replies = replica_execute_to_commit(replica)
	// The round may have committed, leaving the primary idle; drain any batch buffered while it
	// was busy so requests that arrived mid-round go out as one Prepare now (§6.2).
	step_output_fold(&output, replica_maybe_flush_batch(replica))
	return output
}

// Walks the commit number forward over every contiguous op that has reached a quorum. Commit
// advances in order — op k is committed only once k-1 is — so a quorum on a later op cannot
// commit it ahead of an earlier gap. Execution is a separate step (replica_execute_to_commit) so
// commit advance and state-machine execution share one path on every commit, normal or adopted.
func replica_advance_commit_as_primary(replica *Replica) {
	for op := replica.Commit + 1; op <= Commit(replica.Op); op++ {
		// The primary holds every op it has assigned, so it counts toward each op's quorum
		// itself, in addition to the distinct backups that explicitly acknowledged this op.
		distinct_count := len(replica.Prepare_Ok_From[Op(op)]) + 1
		if distinct_count < replica_quorum(replica) {
			break
		}
		// A commit is only safe behind a quorum; assert it when the op commits.
		invariant.Always(distinct_count >= replica_quorum(replica),
			"commit advances only behind a quorum")
		replica.Commit = op
	}
}

// Advances a replica's commit number toward commit_number, but never past the ops it actually holds
// — a backup cannot commit an entry it has not yet prepared — then executes up to the new
// commit. Surfaces the entries that became committed, in op order, and the Replies executing them
// produced. On a backup Replies is empty; only the primary, on its own commit path, replies.
func replica_apply_commit(
	replica *Replica, commit_number Commit,
) (committed []Log_Entry, replies []Message) {
	target := commit_number
	if Commit(replica.Op) < target {
		target = Commit(replica.Op)
	}
	if target > replica.Commit {
		replica.Commit = target
	}
	return replica_execute_to_commit(replica)
}

// Executes a Reconfiguration op (§7.1): a VR-state-only op, never an up-call to the State_Machine.
// It records the admin client's request-number so a retry is deduped (exactly-once applies to the
// admin too) but caches no result, flags the epoch handoff for the top-level step to perform once
// execution settles, and records the epoch-start op. The checkpoint at this op is still taken, so
// the boundary the reconfiguration lands on is captured like any other.
func replica_execute_reconfiguration(replica *Replica, entry Log_Entry, op Op) {
	replica.Client_Table[entry.Client] = Client_Record{
		Request_Number: entry.Request_Number,
		Executed:       true,
	}
	// Record this reconfiguration op as the epoch-start op if it is the newest executed, and
	// trigger the handoff only to a DIFFERENT configuration. A replica that jumped to this
	// epoch (redirect or recovery) and is catching up executes the reconfiguration op that
	// CREATED its current epoch (New_Configuration equals the current configuration); it must
	// learn its epoch-start op from it without advancing the epoch again and overshooting
	// (Bug 19). Capture the new configuration/active count now: the maybe-checkpoint below and
	// end-of-loop compaction can GC the op before the drain reads it.
	if op > replica.Epoch_Start_Op {
		replica.Epoch_Start_Op = op
		differs := !configurations_equal(&configurations_equal_input{
			A: entry.New_Configuration,
			B: replica.Configuration,
		})
		if differs {
			replica.Epoch_Handoff_Due = true
			replica.Epoch_New_Configuration = entry.New_Configuration
			replica.Epoch_New_Active_Count = entry.New_Active_Count
		}
	}
	replica_maybe_checkpoint(replica, op)
}

// Brings the state machine up to the commit number: it executes every committed op the state
// machine has not yet executed, in op order, exactly once — the walk from Executed+1 to Commit is
// the exactly-once guarantee, since Executed advances monotonically and an op is executed only as
// it first crosses. It is the single execution site, shared by the normal commit path and every
// path that adopts a committed prefix wholesale, so the application state always reflects exactly
// Commit. With no state machine injected it only advances Executed (a no-op bookkeeping counter)
// and returns nothing, leaving the core a pure agreement engine. An op below Log_Start lives only
// in the checkpoint, never in Log; a caller that adopts a checkpoint must Restore and set Executed
// to Checkpoint_Op (replica_restore_checkpoint) before calling this, so the walk here only ever
// touches ops that are present in Log.
func replica_execute_to_commit(
	replica *Replica,
) (committed []Log_Entry, replies []Message) {
	primary := replica.Identifier == replica_primary_identifier(replica)
	for op := replica.Executed + 1; op <= Op(replica.Commit); op++ {
		entry := replica_log_entry(replica, op)
		committed = append(committed, entry)
		replica.Executed = op
		if log_entry_is_reconfiguration(entry) {
			replica_execute_reconfiguration(replica, entry, op)
			continue
		}
		// A standby never runs the service (§6.1): it advances its commit and holds the log
		// so it stays current and can be promoted, but it makes no Execute up-call, caches
		// no result, emits no Reply, and takes no checkpoint — only the active replicas
		// store the application state. It still processed the reconfiguration above, since
		// that is VR state, not service state, and a standby must follow the epoch.
		if replica_is_standby(replica) {
			continue
		}
		if replica.State_Machine.Execute == nil {
			continue
		}
		// A Reconfiguration is never an up-call (§7.1): the branch above handled it and
		// continued, so any entry reaching Execute must not be one. Pin it at the call site
		// so a future change that lets a reconfiguration fall through here fails fast.
		invariant.Always(!log_entry_is_reconfiguration(entry),
			"executed entry is not a reconfiguration")
		result := replica.State_Machine.Execute(entry.Command, entry.Prediction)
		// Record the executed request without lowering the client's highest-seen number.
		// Adoption can put a higher in-flight request in the log (recorded by the
		// client-table rebuild) above a still-committing lower op; executing the lower op
		// must not forget the higher, or the primary would re-append it at a fresh op —
		// one client request committed twice, the exactly-once violation the simulator
		// surfaced (see simulator_bugs.md, Bug 4). The cached reply in the table belongs to
		// the highest request only when this op IS the highest seen.
		current, seen := replica.Client_Table[entry.Client]
		record := Client_Record{
			Request_Number: entry.Request_Number,
			Result:         result,
			Executed:       true,
		}
		if seen {
			if current.Request_Number > entry.Request_Number {
				record.Request_Number = current.Request_Number
				record.Result = nil
				record.Executed = false
			}
		}
		replica.Client_Table[entry.Client] = record
		// The reply always carries the op actually executed, even when the table now
		// records a higher pending request: the client matches it by request-number and
		// ignores any that does not answer its outstanding one.
		if primary {
			reply := replica_build_reply(replica, entry.Client, Client_Record{
				Request_Number: entry.Request_Number,
				Result:         result,
			})
			replies = append(replies, reply)
		}
		// Snapshot at the exact op a checkpoint boundary falls on, while the application
		// state reflects precisely op ops — not after the loop, where it would reflect
		// Commit and a checkpoint labelled with a lower boundary would carry a higher op's
		// state, diverging from a peer that checkpointed at the boundary (the divergence
		// the simulator surfaced, Bug 5).
		replica_maybe_checkpoint(replica, op)
	}
	replica_compact(replica)
	return committed, replies
}

// Materializes the application state of a replica just promoted from standby to active across a
// reconfiguration (§6.1, §7). A standby holds the committed log but ran no Execute up-calls, so it
// has no application state; once active it must build it. Following TigerBeetle's state sync —
// acquire the materialized checkpoint, then replay only the ops after it, never the whole log from
// zero — it restores the checkpoint it already carries (a real snapshot it adopted from an active
// replica while a standby) and re-executes the un-checkpointed suffix. Executed is reset to the
// checkpoint op so the replay runs; for a replica with no checkpoint (Checkpoint_Op zero, full log
// retained) it replays from the first op. The caller invokes this only on a standby-to-active
// promotion, so the suffix it replays was never executed here and no op runs twice.
func replica_materialize_application_state(
	replica *Replica,
) (committed []Log_Entry, replies []Message) {
	replica.Executed = replica.Checkpoint_Op
	if replica.State_Machine.Restore != nil {
		replica.State_Machine.Restore(replica.Checkpoint_State)
	}
	return replica_execute_to_commit(replica)
}

// Materializes the application state of a just-promoted standby when it returns to Status_Normal,
// then clears the pending flag, so the rebuild fires exactly once whichever epoch-completion path
// (direct via enter-new-epoch, or after a catch-up via complete-epoch) brought it back to normal. A
// replica with no pending promotion does nothing.
func replica_consume_standby_promotion(
	replica *Replica,
) (committed []Log_Entry, replies []Message) {
	if !replica.Standby_Promotion_Due {
		return committed, replies
	}
	replica.Standby_Promotion_Due = false
	return replica_materialize_application_state(replica)
}

// Records a checkpoint at op when op is the next interval boundary and the application state, just
// executed up to op, is exactly what Snapshot must capture (§5.1). It only snapshots —
// compaction is deferred to after the execution loop so advancing Log_Start cannot disturb the
// loop's indexing. A nil Snapshot or a zero interval disables checkpointing entirely, keeping a
// log-only or non-checkpointing core behaving exactly as before.
func replica_maybe_checkpoint(replica *Replica, op Op) {
	// A standby never snapshots (§6.1): it holds no application state, so it has nothing to
	// capture. Snapshotting its empty state would write a bogus checkpoint that diverges from
	// the active replicas and corrupts any replica that later state-transfers it. A standby
	// still compacts — it adopts a real checkpoint from an active replica during state transfer
	// — so its log stays bounded without ever taking one of its own.
	if replica_is_standby(replica) {
		return
	}
	if replica.State_Machine.Snapshot == nil {
		return
	}
	if replica.Checkpoint_Interval == 0 {
		return
	}
	if uint64(op)%replica.Checkpoint_Interval != 0 {
		return
	}
	if op <= replica.Checkpoint_Op {
		return
	}
	// Snapshot must reflect exactly the checkpoint op, so op must be both committed and
	// executed at the capture point — pin it.
	invariant.Always(op <= Op(replica.Commit), "checkpoint op is committed")
	invariant.Always(op == replica.Executed, "checkpoint op equals executed op")
	replica.Checkpoint_Op = op
	replica.Checkpoint_State = replica.State_Machine.Snapshot()
}

// Garbage-collects the log prefix the latest checkpoint has subsumed, keeping Log_Retain ops of
// suffix past the checkpoint op so an in-flight recovery or transfer needing a recently-committed
// op is not stranded (§5.1). It advances Log_Start to the op just before the first surviving
// entry. It never drops above Checkpoint_Op - Log_Retain, and the commit number is always at or
// above the checkpoint op, so a committed-but-unexecuted op is never discarded.
func replica_compact(replica *Replica) {
	// Keep Log_Retain ops of suffix after the checkpoint: the new Log_Start is the checkpoint
	// op minus the retained suffix, clamped so it never moves backward.
	new_start := replica.Checkpoint_Op
	if Op(replica.Log_Retain) < new_start {
		new_start = replica.Checkpoint_Op - Op(replica.Log_Retain)
	} else {
		new_start = 0
	}
	if new_start <= replica.Log_Start {
		return // Nothing new to drop.
	}
	// The entries to drop are those with op in (Log_Start, new_start]; the survivors begin at
	// op new_start+1, slice index new_start - Log_Start.
	drop := int(new_start - replica.Log_Start)
	survivors := make([]Log_Entry, len(replica.Log)-drop)
	copy(survivors, replica.Log[drop:])
	replica.Log = survivors
	replica.Log_Start = new_start
}

// Restores the application state from an adopted checkpoint and marks the checkpoint's op executed,
// so a following replica_execute_to_commit replays only the committed ops after the checkpoint
// rather than re-running the prefix the checkpoint already covers (the §5.1/§5.2 catch-up). It is
// the inverse half of compaction: Snapshot froze the prefix, Restore thaws it on the adopting
// replica. With no Restore injected it only advances the executed and Log_Start markers.
func replica_restore_checkpoint(
	replica *Replica, checkpoint_op Op, state []byte,
	client_table map[Client_Identifier]Client_Record,
) {
	replica.Checkpoint_Op = checkpoint_op
	replica.Checkpoint_State = state
	replica.Log_Start = checkpoint_op
	replica.Executed = checkpoint_op
	// Merge the checkpoint's client-table: it carries the exactly-once record for every request
	// the checkpoint compacted, which the shipped log suffix omits. Keep the higher request
	// number per client so a record this replica holds for an op above the checkpoint is not
	// lowered. Without this the rebuild below, scanning only the suffix, never re-learns a
	// compacted request and the replica re-appends it on becoming primary (exactly-once bug).
	for client, record := range client_table {
		current, seen := replica.Client_Table[client]
		if seen {
			if current.Request_Number >= record.Request_Number {
				continue
			}
		}
		replica.Client_Table[client] = record
	}
	// A restore materializes the state through checkpoint_op and holds no log beyond it yet:
	// the caller rebuilds the log from the incoming suffix. Op and Log must reflect that empty
	// state, keeping Op == Log_Start + len(Log). A stale higher Op here makes the following
	// splice treat the suffix's first (committed) op as already held and drop it, shifting the
	// rest down and overwriting a committed op (simulator_bugs.md, Bug 15).
	replica.Op = checkpoint_op
	replica.Log = nil
	if replica.State_Machine.Restore == nil {
		return
	}
	replica.State_Machine.Restore(state)
}

// Returns an independent copy of a client-table for shipping inside a checkpoint-bearing message,
// so a later mutation of the sender's live table cannot alter a message already in flight.
func clone_client_table(
	table map[Client_Identifier]Client_Record,
) (copied map[Client_Identifier]Client_Record) {
	copied = make(map[Client_Identifier]Client_Record, len(table))
	for client, record := range table {
		copied[client] = record
	}
	return copied
}

// Reconciles the client-table with the log after the log is replaced wholesale by a view change,
// recovery, or state transfer. Each client's highest request-number present in the adopted log is
// recorded, so a retry of an in-flight request — one appended on a prior primary but not yet
// executed when the view changed — is recognized as a duplicate rather than appended a second
// time. Without this the new primary, whose client-table only ever learned executed requests, would
// re-append the surviving uncommitted entry, and the same request would commit at two ops: an
// exactly-once violation the simulator surfaces (see simulator_bugs.md, Bug 2). An already-executed
// record is preserved; a request only seen here stays unexecuted, its result filled in when the op
// executes.
func replica_rebuild_client_table(replica *Replica) {
	for _, entry := range replica.Log {
		record, ok := replica.Client_Table[entry.Client]
		if ok {
			if entry.Request_Number <= record.Request_Number {
				continue // Not newer than what the table already holds.
			}
		}
		replica.Client_Table[entry.Client] = Client_Record{
			Request_Number: entry.Request_Number,
			Executed:       false,
		}
	}
}

// Tallies a vote for a view change and, once a quorum of distinct replicas wants the new view,
// reports this replica's log to the new view's primary. A vote for a higher view first pulls this
// replica up into that view.
func replica_receive_start_view_change(
	replica *Replica, message Message, now time.Moment,
) (output Step_Output) {
	// A recovering replica's log was wiped by the restart, so it must not be pulled into a view
	// change: its empty log could win the merge and drop a committed entry. It stays quiescent
	// until Recovery_Response restores authoritative state. A transitioning replica is mid
	// epoch handoff (§7): a new-group member still catching up has not finished installing the
	// epoch's log, and a replaced replica is only serving transfers — neither participates in a
	// view change until it is back to normal, so both stay quiescent here too.
	if replica.Status == Status_Recovery {
		return output
	}
	if replica.Status == Status_Transition {
		return output
	}
	// A standby casts no view-change vote and is never the new primary, so it does not join the
	// view-change protocol (TigerBeetle's non-voting-standby design, see replica_quorum). It
	// stays in its current view and follows the outcome through the Start_View the new primary
	// broadcasts, or catches up by state transfer on a later-view message. Pulling it in would
	// have it tally toward a quorum it must not be part of.
	if replica_is_standby(replica) {
		return output
	}
	if message.View < replica.View {
		return output
	}
	if message.View > replica.View {
		output = replica_start_view_change(replica, message.View, now)
	}
	if replica.Status != Status_View_Change {
		return output
	}
	before_count := len(replica.Start_View_Change_From)
	replica.Start_View_Change_From[message.From] = true
	if len(replica.Start_View_Change_From) < replica_quorum(replica) {
		return output
	}
	if before_count >= replica_quorum(replica) {
		return output
	}
	// The quorum was just reached. Report our log to the new view's primary; if we are that
	// primary, fold the report straight into our own tally rather than mailing it to ourselves.
	report := replica_build_do_view_change(replica)
	if report.To == replica.Identifier {
		install := replica_receive_do_view_change(replica, report, now)
		output.Messages = append(output.Messages, install.Messages...)
		output.Committed = append(output.Committed, install.Committed...)
		return output
	}
	output.Messages = append(output.Messages, report)
	return output
}

// Builds this replica's report to the new primary (§5.3): its op, commit, and last-normal-view —
// the numbers the primary ranks reporters by — plus a BOUNDED suffix of at most View_Change_Suffix
// trailing entries rather than the whole log, which can be arbitrarily large. The checkpoint
// travels too, so when the winner is selected and a fetch is needed the receiver knows the winner's
// compacted prefix; the receiver derives the suffix's first op as Op minus the suffix length.
func replica_build_do_view_change(replica *Replica) (message Message) {
	// The suffix is the last View_Change_Suffix entries, clamped to the first live op
	// (Log_Start+1) so a compacted prefix is never indexed. With Op below the bound the suffix
	// is the whole live log; with an empty log it is empty.
	suffix_start := replica.Log_Start + 1
	if replica.Op >= View_Change_Suffix {
		candidate := replica.Op - View_Change_Suffix + 1
		if candidate > suffix_start {
			suffix_start = candidate
		}
	}
	return Message{
		Kind:             Message_Kind_Do_View_Change,
		From:             replica.Identifier,
		To:               replica_primary_identifier(replica),
		View:             replica.View,
		Epoch:            replica.Epoch,
		Op:               replica.Op,
		Commit:           replica.Commit,
		Log_Suffix:       replica_log_slice_from(replica, suffix_start),
		Last_Normal_View: replica.Last_Normal_View,
		Checkpoint_Op:    replica.Checkpoint_Op,
		Checkpoint_State: replica.Checkpoint_State,

		Checkpoint_Client_Table: clone_client_table(replica.Client_Table),
	}
}

// Processed only by the new view's primary: it tallies reports by sender and installs the new view
// once a quorum has reported.
func replica_receive_do_view_change(
	replica *Replica, message Message, now time.Moment,
) (output Step_Output) {
	if replica.Status != Status_View_Change {
		return output
	}
	if message.View != replica.View {
		return output
	}
	if replica.Identifier != replica_primary_identifier(replica) {
		return output
	}
	replica.Do_View_Change_From[message.From] = message
	if len(replica.Do_View_Change_From) < replica_quorum(replica) {
		return output
	}
	// A deferred install already chose a winner to fetch the selected log from (§5.3): keep
	// tallying late reports, but do not re-run selection. Re-running it against the now-stale
	// self-report could pick a shorter log and drop a committed op (simulator_bugs.md, Bug 6);
	// the in-flight fetch owns the install until its New_State lands or a new view change
	// abandons it.
	if replica.View_Change_Deferred {
		return output
	}
	return replica_install_new_view(replica, now)
}

// Selects the surviving log and either installs it now or defers until it can be fetched (§5.3).
// The selection — largest last-normal-view, then largest op — is what guarantees every committed
// entry survives: the reporting quorum intersects every quorum that ever committed, so the most
// up-to-date reporter holds all committed entries. Because a report now carries only a bounded
// suffix, the new primary installs directly only when it can reconstruct the winner's exact log
// from its own committed prefix plus that suffix; otherwise it fetches the missing entries by state
// transfer and stays in view-change until the New_State arrives.
func replica_install_new_view(replica *Replica, now time.Moment) (output Step_Output) {
	// The new view is installed only on a quorum of reports; that quorum is what guarantees the
	// selected log holds every committed entry.
	report_count := len(replica.Do_View_Change_From)
	invariant.Always(report_count >= replica_quorum(replica),
		"new view installs only behind a quorum of reports")
	best := replica_select_log(replica.Do_View_Change_From)
	carrier, reconstructable := replica_reconstruct_selected_log(replica, best)
	if !reconstructable {
		// The winner reported more ops than its suffix supplies and the new primary cannot
		// rebuild them from its own committed prefix: fetch the complete log and DEFER the
		// install (§5.3) — no Start_View until the awaited New_State lands.
		return replica_begin_view_change_fetch(replica, best, now)
	}
	return replica_complete_install(replica, carrier, best.Op, now)
}

// Reconstructs the selected reporter's exact log from what the new primary already holds plus the
// reported suffix, returning the carrier to install and whether reconstruction was possible (§5.3).
// It is possible in three cases: the new primary IS the winner (it holds the log already); the
// winner's log is empty; or the winner's suffix attaches directly onto the new primary's committed
// prefix — every op below the suffix is at or under the new primary's commit, so by agreement the
// new primary holds an identical copy, and the suffix's first op is within its live log. Otherwise
// the entries between the new primary's trustworthy prefix and the suffix are missing or possibly
// divergent, and only a fetch can supply them safely.
func replica_reconstruct_selected_log(
	replica *Replica, best Message,
) (carrier Message, reconstructable bool) {
	// The carrier always ships the new primary's own checkpoint, which covers the prefix below
	// the reconstructed log; adopt restores from it only if the new primary is behind it.
	carrier.Checkpoint_Op = replica.Checkpoint_Op
	carrier.Checkpoint_State = replica.Checkpoint_State
	carrier.Checkpoint_Client_Table = clone_client_table(replica.Client_Table)
	if best.From == replica.Identifier {
		// The new primary is the winner: its own live log already IS the selected log.
		carrier.Op = replica.Op
		carrier.Log = replica_log_slice_from(replica, replica.Log_Start+1)
		return carrier, true
	}
	if best.Op == 0 {
		return carrier, true // The winner's log is empty; nothing to reconstruct.
	}
	suffix_start := best.Op - Op(len(best.Log_Suffix)) + 1
	if suffix_start < replica.Log_Start+1 {
		return carrier, false // The prefix below the suffix was GC'd here; fetch it.
	}
	if suffix_start-1 > Op(replica.Commit) {
		return carrier, false // Entries below the suffix are uncommitted here; may diverge.
	}
	// Keep the new primary's committed entries below the suffix — identical to the winner's by
	// agreement — and append the winner's suffix, yielding the winner's exact log.
	keep := int(suffix_start - 1 - replica.Log_Start)
	reconstructed := make([]Log_Entry, 0, keep+len(best.Log_Suffix))
	reconstructed = append(reconstructed, replica.Log[:keep]...)
	reconstructed = append(reconstructed, best.Log_Suffix...)
	carrier.Op = best.Op
	carrier.Log = reconstructed
	return carrier, true
}

// Sends a Get_State to the selected reporter for the complete log and records the deferred install
// (§5.3): the new primary stays in Status_View_Change with no Start_View until the awaited
// New_State arrives, then adopts that log wholesale (replacing its own divergent uncommitted tail).
// The fetch asks from the new primary's Log_Start, so the reporter ships everything from there
// forward — keeping the carrier's suffix aligned with a checkpoint the new primary can then ship
// in its own Start_View. When the reporter has garbage-collected even that prefix, its Get_State
// answer falls back to its checkpoint, and the adopted log starts at that checkpoint instead.
func replica_begin_view_change_fetch(
	replica *Replica, best Message, now time.Moment,
) (output Step_Output) {
	replica.View_Change_Deferred = true
	replica.View_Change_Fetch_From = best.From
	replica.View_Change_Fetch_Op = best.Op
	output.Messages = []Message{{
		Kind:  Message_Kind_Get_State,
		From:  replica.Identifier,
		To:    best.From,
		View:  replica.View,
		Epoch: replica.Epoch,
		Op:    replica.Log_Start,
	}}
	output.Timer = replica_arm_timer(replica, now)
	return output
}

// Installs a reconstructed-or-fetched selected log: it takes commit to the highest any reporter
// knew (capped at the adopted op), adopts the carrier, returns to normal in the new view, and
// broadcasts a Start_View so the backups converge. Adoption executes the committed prefix, so the
// new primary's application state reflects exactly its commit number — not stale, the bug a
// stateful adopt without execution would hide (see simulator_bugs.md, Bug 3). selected_op is the
// winning reporter's op: the installed op must reach it, or a committed op a too-short suffix
// omitted would be silently dropped (§8 safety) — pinned here.
func replica_complete_install(
	replica *Replica, carrier Message, selected_op Op, now time.Moment,
) (output Step_Output) {
	commit := replica.Commit
	for _, candidate := range replica.Do_View_Change_From {
		if candidate.Commit > commit {
			commit = candidate.Commit
		}
	}
	output.Committed, output.Replies = replica_adopt_log(replica, carrier, commit)
	// The selected reporter held every committed op (the reporting quorum intersects every
	// commit quorum); the installed log must not fall short of its op, or a committed op the
	// bounded suffix omitted would be lost.
	invariant.Always(replica.Op >= selected_op,
		"installed log reaches the selected reporter op")
	replica.Status = Status_Normal
	replica.View_Change_Deferred = false
	replica.Last_Normal_View = replica.View
	output.Timer = replica_arm_timer(replica, now)
	suffix := replica_log_slice_from(replica, replica.Log_Start+1)
	output.Messages = append(output.Messages, replica_broadcast(replica, Message{
		Kind:             Message_Kind_Start_View,
		Op:               replica.Op,
		Commit:           replica.Commit,
		Log:              suffix,
		Checkpoint_Op:    replica.Checkpoint_Op,
		Checkpoint_State: replica.Checkpoint_State,

		Checkpoint_Client_Table: clone_client_table(replica.Client_Table),
	})...)
	return output
}

// Installs a log carried by a Do_View_Change winner, Start_View, or Recovery_Response onto this
// replica — the shared spine of every WHOLESALE adoption, where the carrier's log supersedes this
// replica's above the shared committed prefix (the winner's uncommitted tail may differ, so the
// replica's own tail must be discarded, not merged). The carrier's Log is the suffix from
// carrier_start forward (carrier.Op minus the suffix length); its checkpoint covers the prefix
// below. The replica restores from the checkpoint when it is behind it, replaces its log with the
// carrier's suffix, reconciles the client-table, takes commit to the cap (clamped to the adopted
// op), and executes the committed prefix so the application state reflects exactly the commit
// number — never stale, the bug a stateful adopt without execution would hide (simulator_bugs.md,
// Bug 4). Returns what executing committed. A committed op is never dropped: VSR's quorum
// intersection guarantees the carrier holds every op committed in any prior view, so replacing the
// suffix wholesale preserves the committed prefix even though it discards this replica's own tail.
func replica_adopt_log(
	replica *Replica, carrier Message, commit_cap Commit,
) (committed []Log_Entry, replies []Message) {
	carrier_start := carrier.Op - Op(len(carrier.Log))
	if replica.Executed < carrier_start {
		// The replica is behind the carrier's suffix start: it has no other source for the
		// prefix the suffix omits, so restore from the carrier's checkpoint and replace the
		// log wholesale from carrier_start. The prefix below now lives in the checkpoint.
		replica_restore_checkpoint(replica, carrier.Checkpoint_Op, carrier.Checkpoint_State,
			carrier.Checkpoint_Client_Table)
		replica.Log_Start = carrier_start
		suffix := make([]Log_Entry, len(carrier.Log))
		copy(suffix, carrier.Log)
		replica.Log = suffix
	} else {
		// The replica already holds (executed) a prefix at or past carrier_start, so it
		// keeps its own consistent Log_Start and checkpoint and REPLACES its log from
		// carrier_start forward with the carrier's authoritative suffix. Setting Log_Start
		// to carrier_start here would be a bug: when the replica's own Log_Start is below
		// carrier_start it would strand the ops between, leaving Log_Start above
		// Checkpoint_Op with no source for the gap (the divergence the simulator surfaced,
		// see simulator_bugs.md).
		//
		// The replacement is AUTHORITATIVE, not a splice: the carrier (a Do_View_Change
		// winner, a Start_View, or a Recovery_Response) is the agreed log of its view, so
		// every entry above carrier_start comes from it and any entry this replica holds
		// above carrier_start is discarded, even one at an op the carrier also covers. A
		// replica left Normal with an uncommitted entry from a bypassed primary would
		// otherwise keep that stale entry, and the commit number advancing over it would
		// commit a value the quorum never chose: the agreement violation the simulator
		// surfaced (simulator_bugs.md, Bug 9). Keeping only the carrier-omitted committed
		// prefix below carrier_start is safe: carrier_start <= Executed <= Commit, so those
		// entries are committed and match the carrier's prefix by intersection.
		replica_replace_from(replica, carrier_start, carrier.Log)
	}
	replica.Op = replica.Log_Start + Op(len(replica.Log))
	replica_rebuild_client_table(replica)
	commit := commit_cap
	if Commit(replica.Op) < commit {
		commit = Commit(replica.Op)
	}
	if commit > replica.Commit {
		replica.Commit = commit
	}
	return replica_execute_to_commit(replica)
}

// Ranks the reports and returns the most up-to-date: the largest last-normal-view, breaking ties by
// the largest op. It reads only those numbers — never a full log — which is what lets a report
// carry just a bounded suffix (§5.3). Ranking by op rather than log length is what keeps it correct
// even when the longest reported suffix belongs to an earlier-view reporter whose tail a later view
// already discarded.
func replica_select_log(candidates map[Replica_Identifier]Message) (best Message) {
	chosen := false
	for _, candidate := range candidates {
		if !chosen {
			best = candidate
			chosen = true
			continue
		}
		if candidate.Last_Normal_View > best.Last_Normal_View {
			best = candidate
			continue
		}
		if candidate.Last_Normal_View < best.Last_Normal_View {
			continue
		}
		if candidate.Op > best.Op {
			best = candidate
			continue
		}
		// Equal (Last_Normal_View, Op): break the tie on the smaller identifier so the pick
		// is deterministic across the map's unstable order. The tied logs match (same view,
		// op), so either is correct; only the determinism matters.
		if candidate.Op == best.Op {
			if candidate.From < best.From {
				best = candidate
			}
		}
	}
	return best
}

// Adopts the new view's installed log on a backup and returns it to normal. A Start_View for a view
// it already serves normally is ignored.
func replica_receive_start_view(
	replica *Replica, message Message, now time.Moment,
) (output Step_Output) {
	// A recovering replica rejoins only through Recovery_Response, never via Start_View, so the
	// nonce-protected path stays the single way its wiped state is restored. A transitioning
	// replica catches up through the epoch state-transfer path (§7.1.1), not a Start_View, so
	// it stays quiescent here — adopting a Start_View would flip it to normal with its handoff
	// bookkeeping (Old_Configuration) still set.
	if replica.Status == Status_Recovery {
		return output
	}
	if replica.Status == Status_Transition {
		return output
	}
	if message.View < replica.View {
		return output
	}
	if message.View == replica.View {
		if replica.Status == Status_Normal {
			return output
		}
	}
	replica.View = message.View
	// Adopt the installed log with offset indexing and execute the committed prefix, so a
	// backup's application state reflects exactly its commit number rather than going stale on
	// adoption.
	output.Committed, output.Replies = replica_adopt_log(replica, message, message.Commit)
	replica.Status = Status_Normal
	replica.Last_Normal_View = replica.View
	output.Timer = replica_arm_timer(replica, now)
	return output
}

// Answers a recovering peer. Only a normal replica may answer — a replica that is itself
// recovering or changing views has nothing authoritative to report. The primary of the view reports
// its full log; a backup reports only its view, since the recovering replica must take its log from
// the primary.
func replica_receive_recovery(replica *Replica, message Message) (output Step_Output) {
	if replica.Status != Status_Normal {
		return output
	}
	// A standby does not answer recoveries (TigerBeetle's non-voting-standby design, see
	// replica_quorum): recovery completes on a quorum of responses over the voting set, so a
	// standby's response must not be one of them, or it would let recovery finish without a
	// true voting quorum. The recovering replica relearns its state from the voting replicas.
	if replica_is_standby(replica) {
		return output
	}
	response := Message{
		Kind:  Message_Kind_Recovery_Response,
		From:  replica.Identifier,
		To:    message.From,
		View:  replica.View,
		Epoch: replica.Epoch,
		Nonce: message.Nonce,
	}
	if replica.Identifier == replica_primary_identifier(replica) {
		response.Op = replica.Op
		response.Commit = replica.Commit
		// Ship the log suffix and the checkpoint. A warm-recovering replica that advertised
		// a checkpoint at or above ours already holds the prefix and will keep it; one that
		// did not needs the checkpoint to restore the prefix the suffix omits. Either way
		// the suffix from Log_Start forward plus the checkpoint is the primary's complete
		// authoritative state.
		response.Log = replica_log_slice_from(replica, replica.Log_Start+1)
		response.Checkpoint_Op = replica.Checkpoint_Op
		response.Checkpoint_State = replica.Checkpoint_State
		response.Checkpoint_Client_Table = clone_client_table(replica.Client_Table)
	}
	output.Messages = []Message{response}
	return output
}

// Tallies a response and completes recovery once a quorum has answered and the primary of the
// highest reported view is among them — only that primary's response carries the log the
// recovering replica adopts. A response whose nonce does not match the in-flight one is dropped, so
// a straggler from a previous attempt cannot complete this one.
func replica_receive_recovery_response(
	replica *Replica, message Message, now time.Moment,
) (output Step_Output) {
	if replica.Status != Status_Recovery {
		return output
	}
	if message.Nonce != replica.Recovery_Nonce {
		return output
	}
	replica.Recovery_From[message.From] = message
	if len(replica.Recovery_From) < replica_quorum(replica) {
		return output
	}
	// Recovery completes only behind a quorum of responses, the same intersection guarantee the
	// other phases rely on.
	response_count := len(replica.Recovery_From)
	invariant.Always(response_count >= replica_quorum(replica),
		"recovery completes only behind a quorum of responses")
	highest := View(0)
	for _, response := range replica.Recovery_From {
		if response.View > highest {
			highest = response.View
		}
	}
	// The authority is the primary of the highest reported view, and the primary is chosen from
	// the active replicas alone (§6.1): Configuration[View mod Active_Count], never a standby.
	// The whole configuration's size here would pick the wrong member when standbys exist.
	authority_index := uint64(highest) % uint64(replica_active_count(replica))
	authority_identifier := replica.Configuration[authority_index]
	authority, ok := replica.Recovery_From[authority_identifier]
	if !ok {
		return output
	}
	if authority.View != highest {
		return output
	}
	replica.View = highest
	// Adopt the authority's log with offset indexing and checkpoint-aware execution. A warm
	// recovery keeps its on-disk checkpoint, so replica_adopt_log restores from it only if the
	// authority's checkpoint is newer and replays only the un-checkpointed suffix; a cold
	// recovery, whose checkpoint was wiped, restores from the authority's and replays from
	// there.
	output.Committed, output.Replies = replica_adopt_log(replica, authority, authority.Commit)
	replica.Status = Status_Normal
	replica.Last_Normal_View = replica.View
	output.Timer = replica_arm_timer(replica, now)
	return output
}

// Clears the primary-only deferred-request scratch — the unflushed batch buffer and the
// in-progress pre-step prediction rounds — for a replica leaving its current view or epoch. Both
// hold requests the primary accepted but has not yet placed in the log: a buffered batch awaiting
// flush (§6.2), and pre-step rounds awaiting predictions (§4.4). A replica that is no longer the
// primary of the view it accepted them in must not later flush or complete them — doing so would
// append a request the new view's log may already hold, committing one client request at two ops
// (the duplicate the simulator surfaced). The client whose request is dropped here retries, and the
// new primary runs it afresh. The acknowledgement tally Prepare_Ok_From is deliberately NOT
// cleared: its entries are per-op, the commit walk only reads ops above the commit number, and a
// new primary relies on acknowledgements gathered before the view change to commit the tail it
// installs (Start_View does not re-acknowledge).
func replica_reset_primary_scratch(replica *Replica) {
	replica.Request_Buffer = nil
	replica.Predict_From = map[Prediction_Round_Key]Prediction_Round{}
	// Epoch_Up_From is epoch-relative: every caller is an epoch transition (enter/rejoin/start
	// epoch, recovery) or the constructor, never a plain view change, so clearing it here keeps
	// a view-change-elected primary's accumulated confirmations while a new epoch starts fresh.
	replica.Epoch_Up_From = map[Replica_Identifier]bool{}
}

// Moves the replica into Status_View_Change for view, resets the per-view tallies, counts its own
// vote, and broadcasts a Start_View_Change. Last_Normal_View is deliberately left untouched: it
// must keep naming the last view this replica was normal in, which log-selection ranks by.
func replica_start_view_change(replica *Replica, view View, now time.Moment) (output Step_Output) {
	// VSR-Revisited (sections 4.3, 8): a recovering replica must never enter a view change; its
	// wiped log could win the merge and drop a committed op, the unsafety recovery exists to
	// prevent. And a view change always advances the view. These pin the quiescence and view
	// monotonicity guarantees at the mutation point, in any embedding.
	invariant.Always(replica.Status != Status_Recovery,
		"recovering replica never enters a view change")
	invariant.Always(view > replica.View, "view change advances the view")
	replica.Status = Status_View_Change
	replica.View = view
	replica.Start_View_Change_From = map[Replica_Identifier]bool{replica.Identifier: true}
	replica.Do_View_Change_From = map[Replica_Identifier]Message{}
	replica_reset_primary_scratch(replica)
	// Abandon any deferred install from a prior view change: its fetch targets the old view, so
	// a late New_State for it must not install into this fresh one (§5.3).
	replica.View_Change_Deferred = false
	output.Messages = replica_broadcast(replica, Message{Kind: Message_Kind_Start_View_Change})
	output.Timer = replica_arm_timer(replica, now)
	return output
}

// Sets the replica's next timer deadline from its current role and status — a primary heartbeats
// every Heartbeat, everyone else waits Timeout — and reports what it armed.
func replica_arm_timer(replica *Replica, now time.Moment) (timer Timer_Reset) {
	kind := Timer_Kind_View_Change
	interval := replica.Timeout
	switch {
	case replica.Status == Status_Recovery:
		kind = Timer_Kind_Recovery
	case replica.Status == Status_View_Change:
		kind = Timer_Kind_View_Change
	case replica.Identifier == replica_primary_identifier(replica):
		kind = Timer_Kind_Commit
		interval = replica.Heartbeat
	}
	replica.Timer_Deadline = now + time.Moment(interval)
	return Timer_Reset{Kind: kind, Deadline: replica.Timer_Deadline}
}

// Stamps message with this replica's identity, current view, and epoch and addresses one copy to
// every other member of the configuration, returning them in configuration order.
func replica_broadcast(replica *Replica, message Message) (messages []Message) {
	message.From = replica.Identifier
	message.View = replica.View
	message.Epoch = replica.Epoch
	for _, identifier := range replica.Configuration {
		if identifier == replica.Identifier {
			continue
		}
		addressed := message
		addressed.To = identifier
		messages = append(messages, addressed)
	}
	return messages
}

// VSR's deterministic leader rule, narrowed to the active replicas (§6.1): the primary of a view is
// the member at index View mod Active_Count, so view rotation never lands on a standby — a standby
// can never be primary.
func replica_primary_identifier(replica *Replica) (identifier Replica_Identifier) {
	return replica.Configuration[uint64(replica.View)%uint64(replica_active_count(replica))]
}

// The number of active replicas: the configured Active_Count, or the whole configuration when it is
// zero (a group with no standbys, the behavior before standbys existed). The active replicas are
// the leading Active_Count members of Configuration; the rest are standbys.
func replica_active_count(replica *Replica) (count int) {
	if replica.Active_Count == 0 {
		return len(replica.Configuration)
	}
	return int(replica.Active_Count)
}

// Reports whether identifier is a standby in this configuration (§6.1): a member at or beyond the
// active prefix. A standby replicates the log but never votes, executes, replies, or becomes
// primary (TigerBeetle's non-voting standby, see replica_quorum). A non-member is not a standby
// here — membership is checked separately.
func replica_identifier_is_standby(replica *Replica, identifier Replica_Identifier) (yes bool) {
	active := replica_active_count(replica)
	for index, member := range replica.Configuration {
		if member == identifier {
			return index >= active
		}
	}
	return false
}

// Reports whether this replica is a standby (§6.1).
func replica_is_standby(replica *Replica) (yes bool) {
	return replica_identifier_is_standby(replica, replica.Identifier)
}

// Computes f+1 for a 2f+1 cluster — a simple majority, the smallest set that must intersect any
// other quorum.
//
// The quorum is over the VOTING replicas only — the active prefix; standbys never vote. This is a
// DELIBERATE COPY OF TIGERBEETLE'S DESIGN, NOT the VSR paper's §6.1: the paper's witnesses are
// voting members of the 2f+1 (only f+1 of which execute), which makes a witness part of commit
// quorums and forces its committed ops to survive view changes — a subtle safety property the
// simulator showed is easy to get wrong. TigerBeetle instead makes its extra replicas NON-VOTING
// standbys: they replicate the log and can be promoted, but never count toward any quorum, so the
// voting set runs the unmodified base protocol and inherits its proven safety. We follow
// TigerBeetle. With no standbys (Active_Count zero) this is the plain majority of the whole group,
// exactly as before standbys existed.
//
// It is EPOCH-RELATIVE during a handoff (§7): while the old group still owns the reconfiguration op
// the deciding quorum is the OLD group's. With standbys the voting count is fixed across the
// handoff (a reconfiguration preserves the voting-set size, TigerBeetle promoting a standby into a
// fixed slot), so the current Active_Count is the old group's voting count too; without standbys
// the old group's size is Old_Configuration's, preserved below.
func replica_quorum(replica *Replica) (size int) {
	if replica.Active_Count > 0 {
		return int(replica.Active_Count)/2 + 1
	}
	configuration := replica.Configuration
	if len(replica.Old_Configuration) > 0 {
		configuration = replica.Old_Configuration
	}
	return len(configuration)/2 + 1
}

// The new group's quorum, f'+1 over the new group's VOTING replicas (§7.1.2): the count of
// Epoch_Started a replaced replica needs before it may shut down. Over the voting prefix, not the
// whole Configuration, because standbys do not vote (TigerBeetle's non-voting-standby design, see
// replica_quorum); with no standbys it is the plain majority of the new Configuration.
func replica_new_group_quorum(replica *Replica) (size int) {
	if replica.Active_Count > 0 {
		return int(replica.Active_Count)/2 + 1
	}
	return len(replica.Configuration)/2 + 1
}

// Reports whether the primary is open to client requests. It is not once the epoch's
// reconfiguration has been appended: that request is the epoch's last op (§7.1), recognized as the
// topmost log entry carrying a New_Configuration. Reading the log rather than a separate flag keeps
// the rule correct across a view change, where a new primary inherits the topmost entry but not any
// volatile flag.
func replica_open_to_requests(replica *Replica) (open bool) {
	if replica.Op == 0 {
		return true
	}
	if replica.Op <= replica.Log_Start {
		// The whole log is compacted; a reconfiguration would never be GC'd here.
		return true
	}
	if len(replica_log_entry(replica, replica.Op).New_Configuration) == 0 {
		return true
	}
	// The topmost entry is a reconfiguration. It closes the primary to new requests only while
	// uncommitted (the reconfiguration is the epoch's last op). Once it commits, the epoch has
	// advanced and the new epoch's primary serves on top of it; without this the new primary
	// could never append its first op, wedging the cluster after a reconfiguration (Bug 18).
	return replica.Op <= Op(replica.Commit)
}

// Reports whether op's entry is a Reconfiguration (§7): one carrying a New_Configuration. A
// reconfiguration entry affects only VR state and is never passed to the State_Machine.
func log_entry_is_reconfiguration(entry Log_Entry) (yes bool) {
	return len(entry.New_Configuration) > 0
}

// Returns op's entry from the log, translating the op-number into the slice index that compaction
// offsets by Log_Start. It is the single site that knows op k lives at Log[k - Log_Start - 1], so
// every caller routes through it rather than open-coding the arithmetic. An op at or below
// Log_Start has been garbage-collected — it lives only in the checkpoint — and asking for it is
// a caller bug (the caller must take the checkpoint path), pinned here so it fails fast rather than
// reading the wrong entry.
func replica_log_entry(replica *Replica, op Op) (entry Log_Entry) {
	invariant.Always(op > replica.Log_Start, "log entry op is above log start")
	invariant.Always(op <= replica.Op, "log entry op is at or below the highest op")
	return replica.Log[op-replica.Log_Start-1]
}

// Returns the log entries from op forward, the suffix a heartbeat re-drive or a state-transfer
// response ships. op must be the op just after a held op (Log_Start+1 at the earliest), since an op
// at or below Log_Start has been compacted away; the slice is a copy so a caller may stash it in a
// message without aliasing the live log.
func replica_log_slice_from(replica *Replica, op Op) (entries []Log_Entry) {
	invariant.Always(op > replica.Log_Start, "log slice start op is above log start")
	start := int(op - replica.Log_Start - 1)
	entries = make([]Log_Entry, len(replica.Log)-start)
	copy(entries, replica.Log[start:])
	return entries
}

// Lazily initializes the per-phase tally maps so a Replica built as a bare struct literal (the
// common case in tests) is usable without the constructor.
func replica_ensure_scratch(replica *Replica) {
	if replica.Start_View_Change_From == nil {
		replica.Start_View_Change_From = map[Replica_Identifier]bool{}
	}
	if replica.Do_View_Change_From == nil {
		replica.Do_View_Change_From = map[Replica_Identifier]Message{}
	}
	if replica.Prepare_Ok_From == nil {
		replica.Prepare_Ok_From = map[Op]map[Replica_Identifier]bool{}
	}
	if replica.Recovery_From == nil {
		replica.Recovery_From = map[Replica_Identifier]Message{}
	}
	if replica.Client_Table == nil {
		replica.Client_Table = map[Client_Identifier]Client_Record{}
	}
	if replica.Predict_From == nil {
		replica.Predict_From = map[Prediction_Round_Key]Prediction_Round{}
	}
	if replica.Epoch_Started_From == nil {
		replica.Epoch_Started_From = map[Replica_Identifier]bool{}
	}
	if replica.Epoch_Up_From == nil {
		replica.Epoch_Up_From = map[Replica_Identifier]bool{}
	}
}
