
# Cluster

These whole-cluster safety and liveness properties hold across many seeded runs of message loss,
reordering, duplication, delay, partition, and crash-restart, asserted after every delivery.

### Single Primary

No two replicas in Status_Normal ever act as the primary of the same (epoch, view); single-primary
is epoch-relative (§8.3), so the old and new groups' primaries coexisting across a handoff is
allowed while two primaries of one (epoch, view) is not.

### Agreement

Wherever two replicas have both committed an op they still retain, they hold the identical command
at that op — across the epoch boundary too, as op-numbers stay monotonic through a reconfiguration;
the prefix below a compacted Log_Start is instead checked through byte-identical Checkpoint_State.

### Liveness

Across the seeded runs every phase completes: commits advance, a primary fails over, a crashed
replica recovers, the efficient view change (§5.3) reports a bounded suffix and defers an install,
and a reconfiguration (§7) advances the epoch and brings up the new group.

### Exactly Once

A given client request executes to one result across the whole cluster: no (client, request-number)
pair ever maps to two different results, and no op-number executes twice on any one replica, even
through retries, crashes, and view changes.

### Linearizability

Every result a client receives equals a reference machine over the committed log in op order, and
every replica's live accumulator at commit C equals the reference's over the first C committed ops,
so the cluster, snapshot/restore included, is indistinguishable from a single correct machine.

### Clock Skew

Safety holds when every replica reads its own clock — a per-replica offset, a bounded drift, and
injected clock faults, so no two nodes share time — proving the protocol leans on consensus, not the
clock; the §4.4 prediction and the timeout-driven view change both run against divergent clocks.

### Reconfiguration

A reconfiguration grows, shrinks, and swaps the group across a run (§7): the epoch advances, the new
group catches up and serves, replaced replicas shut down, and exactly-once and agreement hold across
the epoch boundary — a committed op and its client result survive the move into the new epoch.
