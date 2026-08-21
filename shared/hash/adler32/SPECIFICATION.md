
# Package Owned State

Digest operations use package-owned bounded types and caller-owned state.

# Reference Values

Checksum and Digest reproduce Adler-32 reference values.

# Reduction Boundary

Bounded writes preserve RFC 1950 reduction across call boundaries.

# Caller Owned Output

Sum, marshal, and clone results use only caller storage.

# State Compatibility

Serialized state matches standard eight-byte Adler-32 format and rejects hostile input.

# Clone

Clone copies initialized state into caller storage without aliasing.

# Bounds

Every source and destination call observes shared byte bounds.

# Invariant Domains

Tests exercise each scalar boundary and validation outcome.

# Allocation

Every exported runtime operation performs zero heap allocation.
