
# Normal Operation

A replica is a pure step function: Replica_Receive folds one Message in, Replica_Tick folds
one clock tick in, and each returns Messages, newly Committed entries, and the Timer to
re-arm. The primary, the replica whose Identifier equals View mod Cluster_Size, alone appends.

### Append

Replica_Receive of a client command on the primary appends one Log_Entry tagged with the
current view, advancing Op by one so it equals the log length.

### Prepare

Appending on the primary returns a Prepare addressed to every other replica, carrying the
new op, the entry, and the primary's current commit number.

### Prepare Ok

A backup that receives a Prepare for the op immediately after its own last op appends the
entry and returns a single Prepare_Ok to the primary; a Prepare that skips an op is not
applied.

### Commit

The primary commits an op once it has Prepare_Ok from a quorum — f+1 distinct replicas
counting itself — advancing its commit number and surfacing the entry in Committed.

### Commit Broadcast

A primary with no client work to send tells the backups its commit number through a Commit
message when Replica_Tick passes its idle deadline.

### Commit Apply

A backup that learns of a higher commit number advances its own and surfaces every
newly-committed entry in Committed, in op order.

### Duplicate Request

A Request whose request-number is not greater than the client-table's recorded number is not
re-appended; if it equals the most recent executed request, the cached result is re-sent as a
Reply, supporting a client that lost its reply (§4.5) without re-executing the command.

### Inflight Request

A Request equal to the client-table's current not-yet-executed request is dropped: no new entry
and no Reply, since the inflight op will reply when it commits.

# Execution

A committed op is executed against the injected state machine exactly once per replica, as the
commit number crosses it. The state machine is optional: with none injected the core stays a
log-only agreement engine. The primary returns the result to the requesting client as a Reply.

### Up Call

Advancing the commit number calls the state machine's Execute once per op, in op order, and never
twice for the same op — the monotonic commit walk is the exactly-once guarantee.

### Reply

The primary, on committing a client's op, emits a Reply addressed to the client carrying the
current view, the request-number, and the Execute result; a backup committing the same op emits
no Reply.

### Client Table

Committing a client's op records {request-number, result, executed} in the client-table on every
replica, so any replica can answer a duplicate request and a new primary inherits the dedup state.

# Prediction

A non-deterministic operation's value is predetermined once by the primary and propagated in the
log entry, so every replica executes the identical value rather than each recomputing it (§4.4). The
primary computes it locally (local-predict) or combines f backups' values with its own (pre-step).

### Local Predict

With only Predict configured, the primary stamps each new entry's Prediction from Predict before it
broadcasts the Prepare; a backup applying the Prepare adopts the entry's Prediction unchanged and
never calls Predict.

### Predicted Execution

Execute receives the entry's stored Prediction, so every replica — primary and backups alike —
executes the op against the identical predetermined value rather than one each computed itself.

### Pre Step Request

With Combine configured, a client Request does not append immediately: the primary broadcasts a
Predict_Request carrying the command and the client/request-number, and buffers the pending request
until the backups answer.

### Pre Step Combine

The primary, once it holds Predict_Response from f backups, calls Combine over those responses and
its own local prediction and only then appends the entry and broadcasts the Prepare, the entry
carrying the combined Prediction.

# Checkpoint

A long-lived log cannot grow without bound (§5.1). A replica periodically snapshots the application
state at a committed op and garbage-collects the log prefix up to it, keeping a suffix so an
in-flight transfer is not stranded. A state machine with no Snapshot disables it: no compaction.

### Take

When the commit number crosses a multiple of the checkpoint interval, the replica records that op as
its Checkpoint_Op and captures the application state through the state machine's Snapshot.

### Compact

After a checkpoint the replica drops every log entry at or below Checkpoint_Op minus the retained
suffix, advancing Log_Start to the op before the first surviving entry, never discarding an op above
the commit number.

### Offset Index

Once a prefix is compacted, op k is stored at Log index k minus Log_Start minus one, and the op
number always equals Log_Start plus the log length.

# View Change

A view change replaces a primary the backups can no longer hear. It advances the view, and
because the next primary is fixed by View mod Cluster_Size, the protocol need only agree to
move to the new view and transfer the most up-to-date log to that predetermined primary.

### Timeout

A backup whose view-change deadline passes in Replica_Tick without contact from the primary
enters Status_View_Change and broadcasts a Start_View_Change for the next view.

### Start View Change

A replica that sees a Start_View_Change for a view above its own advances to it, enters
Status_View_Change, and broadcasts its own Start_View_Change for that view.

### Quorum

A replica that has collected Start_View_Change from a quorum of distinct replicas sends a
Do_View_Change, carrying a bounded log suffix and the last view in which it was normal, to the
new view's primary.

### Do View Change

The new primary collects Do_View_Change from a quorum of distinct replicas before it
installs the new view.

### Log Selection

The new primary adopts, from its collected Do_View_Change messages, the log with the
largest last-normal-view, breaking ties by the largest op — never merely the longest log.

### Suffix Report

A Do_View_Change carries only a bounded log suffix together with the reporter's op,
last-normal-view, and commit — not its full log; the suffix is the last few entries, so a
reporter whose op exceeds the suffix length reports fewer entries than it holds.

### Suffix Fetch

When the selected reporter's op exceeds what the new primary can reconstruct from its own committed
prefix plus the received suffix, the new primary fetches the rest by state transfer — using the
reporter's checkpoint when the prefix was garbage-collected — before it installs.

### Deferred Install

The new primary stays in Status_View_Change and broadcasts no Start_View until it holds the
complete selected log; only once the awaited state arrives does it install, return to normal, and
broadcast Start_View, which it emits exactly once.

### Start View

The new primary installs the selected log, returns to Status_Normal in the new view, and
broadcasts a Start_View carrying that log so the backups converge on it.

### Adoption

A backup that receives a Start_View adopts the message's log, view, op, and commit number
and returns to Status_Normal.

### Commit Preservation

The log the new primary installs contains every entry that was committed in any earlier
view, so no acknowledged command is ever lost across a view change.

# Recovery

A replica that has lost its volatile state rejoins the cluster through recovery rather than
participating with stale or empty state. It learns the current view and log from a quorum
of normal replicas before it resumes.

### Nonce

A recovering replica enters Status_Recovering, draws a fresh nonce, and broadcasts a
Recovery carrying that nonce.

### Response

A normal replica that receives a Recovery returns a Recovery_Response echoing the nonce and
reporting its view, and — if it is the primary of that view — its log, op, and commit.

### Stale Rejection

A Recovery_Response whose nonce differs from the recovering replica's in-flight nonce is
ignored, so a delayed response from an earlier attempt cannot complete the current one.

### Quorum

A recovering replica completes once it holds Recovery_Response from a quorum of distinct
replicas that includes the primary of the highest view among them.

### Rejoin

A recovered replica adopts the primary's log, op, and commit, returns to Status_Normal in
that view, and resumes normal operation.

### Quiescence

A recovering replica ignores Start_View_Change and Start_View, rejoining only through
Recovery_Response, so its wiped log never enters a view change and drops a committed entry.

### Checkpoint Advertise

A recovering replica that kept its checkpoint on disk advertises its Checkpoint_Op in the Recovery
it broadcasts, so the answering primary knows which prefix the replica already holds.

### Checkpoint Restore

A replica recovering warm — its checkpoint survived the restart — restores the application state
from that checkpoint, keeps its Log_Start at the checkpoint's op, and replays only the
un-checkpointed suffix the primary ships, rather than re-executing the whole log from empty.

# State Transfer

A replica that has fallen behind but not crashed catches up without a view change by fetching log
from a more current peer, rather than dropping the messages that reveal it is behind.

### Behind View

A normal replica that receives a Prepare or Commit from a later view drops its uncommitted log
tail, adopts the view, and sends a Get_State to the sender to fetch the rest.

### Gap

A normal replica that receives a Prepare for an op beyond its next one applies what it can and
sends a Get_State for the missing entries, rather than dropping the message.

### Response

A normal replica at least as current as a Get_State's view answers it with a New_State carrying
its log, op, and commit.

### Apply

A replica that receives a New_State no older than its own view adopts that log, op, commit, and
view, and resumes normal operation.

### Checkpoint Gap

A Get_State for an op the responder has already garbage-collected is answered with a New_State
carrying the responder's checkpoint — its Checkpoint_Op and Checkpoint_State — and the log from the
checkpoint forward, since the prefix the requester asked for no longer exists to ship.

### Checkpoint Apply

A replica that receives a New_State carrying a checkpoint restores the application state from it,
sets Log_Start to the checkpoint's op, adopts the message's log, op, commit, and the view in which
the checkpoint was taken, and resumes normal operation.

# Reconfiguration

A reconfiguration changes the group's membership and fault threshold f (§7). The old group commits
an administrator's request through normal-case agreement, advancing to a new epoch the new group
serves once caught up. Epoch dominates view: a replica processes only messages matching its epoch.

### Request

The primary accepts a Reconfiguration only at its own epoch, with a fresh request-number and a new
configuration of at least three members; it then appends the request, broadcasts a Prepare, and
stops accepting further client requests, the reconfiguration being the epoch's last op.

### Threshold

The new threshold is the largest f' with 2f'+1 at most the new configuration's size, so a quorum is
recomputed for the grown or shrunk group rather than inherited from the old one.

### Epoch Increment

When the primary holds a quorum of Prepare_Ok for the reconfiguration op, it increments its epoch,
sends Commit to the other old replicas, sends Start_Epoch to the added nodes, executes every client
op before the reconfiguration, and sets its status Transitioning.

### Start Epoch

A replica learning the new epoch through a Start_Epoch records the old and new configurations, the
new epoch and op-number, sets view 0, and enters Transitioning, state-transferring from the old
replicas when its log is missing the epoch's ops.

### New Group Catch Up

A new replica with an empty log catches up to the start of the epoch through state transfer — the
checkpoint path included, restoring a checkpoint rather than replaying from zero — and only then
becomes Normal.

### Epoch Started

Once a new replica is caught up to the start of the epoch it returns to Normal, executes any
un-executed ops, accepts requests if it is the new epoch's primary, and sends Epoch_Started to the
replaced replicas; a duplicate Start_Epoch after completion is answered with Epoch_Started.

### Old Group Shutdown

A replaced replica, on learning the new epoch, moves its configuration to Old_Configuration and the
new configuration in, enters Transitioning, and serves state transfer until it holds f'+1
Epoch_Started — the new group's quorum — then enters Shutdown.

### Epoch Precedence

A replica processes a message only at its own epoch: a lower-epoch one is discarded and its sender
sent a New_Epoch, a higher-epoch one is adopted even at a lower view, and only at an equal epoch
does the view comparison apply — epoch dominates view.

### View Change Across Epoch

A new primary whose topmost log entry is a committed Reconfiguration sends Start_Epoch and stops
accepting client requests; a non-topmost Reconfiguration is ignored, the reconfiguration having
already happened.

### Client Redirect

An old replica that receives a client request stamped with a stale epoch replies New_Epoch carrying
the current view and the new configuration, so the client can find the group that moved.
