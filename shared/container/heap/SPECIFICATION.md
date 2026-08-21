
# Initialization

Initialize orders an arbitrary element slice into a minimum heap. It is idempotent, thus a caller
can repeat it after an unspecified change to the elements.

# Insertion

Push adds one element into caller-owned spare capacity and returns the grown slice. Push rejects
storage with no remaining position. Result keeps minimum element at position zero.

# Extraction

Pop takes the minimum element and returns the shrunk slice together with that element. Pop and
Remove at position zero give the same result.

# Removal

Remove takes the element at one position and returns the shrunk slice together with that element.
The remaining elements keep the heap order.

# Repair

Fix restores the heap order after the element at one position changes. It costs less than a Remove
and a Push of the new value.

# Injected Order

Each operation takes a comparison function, and that function alone selects the minimum element. A
comparison that exchanges its two operands gives a maximum heap.

# Size Limits

Each element slice holds at most ELEMENT_COUNT_MAXIMUM elements. Push rejects a full slice before
it adds the new element, even when backing storage has more capacity.

# Allocation

Every operation allocates zero heap storage. Caller owns element storage and Push uses only spare
capacity already present in that storage.

# Domain Errors

An operation on an empty slice, a position that the slice does not hold, or a slice above the size
limit causes a panic.

# Invariant Domains

The tests reach both position ends, both interior position sentinels, and both report branches
through public operations.
