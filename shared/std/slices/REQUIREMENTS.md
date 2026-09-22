
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

Insert_Into adds values. Delete_Into removes one half-open range. Replace_Into exchanges it.
Delete_Function_Into removes selected values. Caller storage receives results and count.
Same-start destination supports overlap. Separate destination must not overlap source or values.

# Copy And Capacity

Clone_Into makes shallow copy. Compact_Into forms keep first value from each equal run.
Grow_Into copies source into storage holding requested additional slots. Clip returns input view.
Caller storage receives results. Each writing operation returns populated count.

# Order Changes

Reverse exchanges elements in place. Concatenate_Into joins slices inside separate caller storage.
Repeat_Into writes source copies. Sort forms order in place; stable form preserves equal order.
Is_Sorted and Is_Sorted_Function report applicable order.

# Extrema

Minimum and Maximum return the smallest and largest ordered values. Each Function form uses an
injected comparison function and keeps the first equal extreme.

# Binary Search

Binary_Search finds the first ordered match or its insertion index. Binary_Search_Function applies
the same rule to an injected comparison function that can compare different element and target
types.

# Iteration

All synchronously yields index-value pairs in forward order. Backward yields pairs in reverse order.
Values yields only values. Injected callbacks stop traversal. Each operation returns yielded count.
No operation returns closure.

# Collection

Append_Sequence_Into writes prefix followed by source sequence into caller storage. Collect_Into
copies source sequence. Sorted_Into, Sorted_Function_Into, and Sorted_Stable_Function_Into copy then
apply applicable order. Each operation returns populated destination count.

# Chunks

Chunk synchronously yields consecutive source views of at most requested count. Each view has no
unused capacity. Injected callback stops traversal. Chunk returns yielded view count.

# Nil Preservation

Clip preserves nil input. Caller owns every output destination and nil policy.

# Allocation

Every exported operation performs zero heap allocations, measured separately by Zero_Allocation.
Owned returns, iterator closures, append growth, and hidden scratch allocation stay absent.
Output operations accept caller storage and return only populated element count.

# Size Limits

Each input and output slice holds at most SLICE_COUNT_MAXIMUM elements. An operation rejects a
larger input or result before it returns the result.

# Domain Errors

Invalid index range, negative growth or repeat count, short or harmful overlapping destination,
empty extrema input, or chunk count below one causes panic.

# Invariant Domains

The tests reach both Boolean results, each normalized order, each comparison sentinel, each count
sentinel, and each position sentinel through public operations.
