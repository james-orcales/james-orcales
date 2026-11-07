
# Ordering

Sort orders caller-owned elements in place and need not preserve equal-element order. Stable
keeps equal-element input order. Is_Sorted reports whether elements already have injected order.
Exchanging comparison operands gives reverse order without adapter state.

# Search

Search returns first position where injected predicate becomes true, or count when it never becomes
true. Find returns first nonpositive comparison position and reports equality there. Explicit state
lets both injected functions avoid captured closures.

# Size Limits

Each element slice and search count holds at most ELEMENT_COUNT_MAXIMUM elements. Any larger input
causes panic before package calls injected code.

# Allocation

Every operation allocates zero heap storage. Caller owns element and search state storage. Stable
uses in-place symmetric merge and both orderings use bounded fixed stack storage.

# Domain Errors

Negative count, count above ELEMENT_COUNT_MAXIMUM, nil injected function, or oversized elements
cause panic.

# Invariant Domains

Tests reach both Boolean results, both comparison signs and equality, and count and position
boundaries through public operations.
