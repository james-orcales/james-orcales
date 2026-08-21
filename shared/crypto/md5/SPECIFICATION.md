
# Package Owned State

Digest operations use caller-owned bounded state. Package owns no mutable global state.

# Formula

Digest, block, padding, state, and round geometry derive from shared primitive widths and RFC
structure.

# Reference Values

Checksum and Digest reproduce RFC 1321 values.

# Stream

Writes preserve MD5 state across every block and call boundary.

# Caller Owned Output

Digest output uses fixed values or caller storage. Short storage stays untouched and reports
required width.

# Bounds

Every source and destination call observes shared byte bounds. Uninitialized state is rejected.

# Invariant Domains

Runtime calls reach each valid state and caller-storage boundary.

# Allocation

Every exported runtime operation performs zero heap allocation.
