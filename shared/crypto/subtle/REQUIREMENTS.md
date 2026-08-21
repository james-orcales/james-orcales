
# Constant Time

Secret byte and scalar values never control comparison, selection, or copy branches.

# Reference Behavior

Operations reproduce `crypto/subtle` for every defined input domain.

# Caller Owned Storage

Copy and XOR operations write only caller storage. Refused operations leave it untouched.

# Bounds

Every byte slice observes shared byte bounds. Selectors and comparison results admit only zero or
one. Less-or-equal operands stay inside its defined nonnegative 31-bit domain.

# Overlap

XOR accepts exact overlap and rejects partial overlap before writing.

# Invariant Domains

Runtime calls reach every scalar value, collection boundary, and storage outcome.

# Allocation

Every exported runtime operation performs zero heap allocation.
