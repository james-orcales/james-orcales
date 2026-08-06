
# Construction

New makes an empty list, and the zero List value is an empty list that is ready to use.
Initialize empties a list that already holds elements.

# Handles

An operation names one element by its Position, and that handle stays correct until Remove takes
it. POSITION_NONE is the handle that no element holds. A handle belongs to the list that made
it, thus an operation that reads a handle of a different list is a caller error.

# Traversal

Front and Back return the first and the final position, or POSITION_NONE for an empty list. Next
and Previous walk to a neighbor position, and each one returns POSITION_NONE at its end of the
list. Value_At returns the value that one position holds.

# Insertion

Push_Front and Push_Back add one value at each end and return its new position. Insert_Before and
Insert_After add one value beside a mark of the same list.

# Removal

Remove takes one position out of its list and returns the value that the position held. A later
insertion reuses the storage of the removed node.

# Moves

Move_To_Front and Move_To_Back move one position to each end. Move_Before and Move_After move one
position beside a mark. Neither operation makes a new node.

# Copies

Push_Back_List and Push_Front_List add a copy of another list at each end. The other list can be
the same list, and it keeps its own elements.

# Size Limits

A list holds at most ELEMENT_COUNT_MAXIMUM elements. An insertion above that limit causes a
panic.

# Domain Errors

A position that no live element holds causes a panic. This includes POSITION_NONE, both sentinel
positions, a position outside the pool, and a position that Remove already took.

# Invariant Domains

The tests drive each operation over an empty list, a small list, and the largest admitted list,
and over both ends of the handle domain.
