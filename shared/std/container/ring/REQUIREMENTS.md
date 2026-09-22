
# Construction

Pool holds caller-owned Nodes with exactly NODE_COUNT_MAXIMUM positions. Initialize readies storage.
New takes node count from pool and closes nodes into one ring. Zero Pool has no storage and is not
ready.

# Handles

An operation names one ring by the handle of any one of its nodes, because a ring has no first
node. POSITION_NONE is the handle of an empty ring, and Value_At and Set_Value read and write the
value of one node.

# Traversal

Next and Previous walk one node in each direction and never leave the ring. Element_Count counts
the nodes of one ring, and it costs one lap.

# Movement

Move walks one offset of nodes, backward for a negative offset and forward for a positive one. An
offset of zero returns its own handle.

# Link

Link joins the ring of one handle to the ring of another handle, and it returns the handle that
followed the first one. Two handles of one ring divide that ring, and the result is the handle of
the divided part.

# Unlink

Unlink takes one count of nodes out of a ring, starting after the handle, and returns the handle
of the taken part. A count of zero changes nothing and returns POSITION_NONE.

# Iteration

For_Each calls one visitor with the value of each node, in forward order, starting at the handle.
The visitor must not change the ring.

# Release

Release returns the nodes of one ring to the pool, so a later New reuses them. The standard
library has no such operation, because a bounded pool needs one where a collector does not.

# Size Limits

A pool holds at most ELEMENT_COUNT_MAXIMUM nodes. A New that asks for more nodes than the pool
holds free causes a panic.

# Allocation

Every operation allocates zero heap storage. Caller owns fixed node storage. Release returns nodes
to storage for later New calls.

# Domain Errors

A handle that no live node holds causes a panic. This includes the free-chain sentinel, a handle
outside the pool, and a handle that Release already took. An Unlink count that is not below the
ring count also causes a panic, where the standard library folds that count into one lap.

# Invariant Domains

Tests drive each operation over empty pool, small ring, largest admitted ring, both ends of handle
domain and offset domain, and every concrete value sentinel.
