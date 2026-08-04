
# Equality

Equal compares comparable elements in index order. Equal_Function uses an injected equality
function and can compare two different element types. A nil slice and an empty slice are equal.

# Ordering

Compare orders two ordered slices lexicographically. Compare_Function uses an injected comparison
function and returns its first nonzero result. A common prefix puts the shorter slice first.

# Search

Index and Index_Function return the first matching index, or INDEX_NOT_FOUND. Contains and
Contains_Function report the same searches as Boolean facts.

# Edits

Insert adds a values slice at an index. Delete removes a half-open index range. Replace exchanges a
half-open range for a values slice. Delete_Function removes each value that an injected predicate
selects. The operations reuse available capacity and preserve overlapping inputs.

# Copy And Capacity

Clone makes a shallow copy. Compact and Compact_Function keep the first value from each consecutive
equal run. Grow reserves capacity, and Clip removes unused capacity.

# Order Changes

Reverse exchanges elements in place. Concatenate joins a slice of slices. Repeat copies one slice a
specified count of times. Sort and Sort_Function order in place, and Sort_Stable_Function preserves
the input order of equal elements. Is_Sorted and Is_Sorted_Function report the applicable order.

# Extrema

Minimum and Maximum return the smallest and largest ordered values. Each Function form uses an
injected comparison function and keeps the first equal extreme.

# Binary Search

Binary_Search finds the first ordered match or its insertion index. Binary_Search_Function applies
the same rule to an injected comparison function that can compare different element and target
types.

# Iteration

All yields index-value pairs in forward order. Backward yields the pairs in reverse order. Values
yields only the values. Each iterator stops when its consumer stops.

# Collection

Append_Sequence appends an iterator to a slice. Collect makes a new slice. Sorted, Sorted_Function,
and Sorted_Stable_Function collect and then apply their applicable order.

# Chunks

Chunk yields consecutive slices of at most one requested count. Each chunk has no unused capacity,
thus an append to a chunk cannot change the source slice.

# Nil Preservation

An operation that can return its input preserves a nil input. Collect and the Sorted forms return
nil for an empty sequence. Repeat always returns a nonnil slice.

# Size Limits

Each input and output slice holds at most SLICE_COUNT_MAXIMUM elements. An operation rejects a
larger input or result before it returns the result.

# Domain Errors

An invalid index range, a negative growth or repeat count, an empty extrema input, or a chunk count
below one causes a panic.

# Invariant Domains

The tests reach both Boolean results, each normalized order, each comparison sentinel, each count
sentinel, and each position sentinel through public operations.
