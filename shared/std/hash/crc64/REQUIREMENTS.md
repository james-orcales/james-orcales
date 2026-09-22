
# Package Owned State

Digest operations use package-owned bounded types and caller-owned state.

# Reference Values

Checksum reproduces standard ISO and ECMA CRC-64 values.

# Table

Table_Make_Into accepts every reflected 64-bit polynomial in caller storage.

# Streaming And Update

Output depends on bytes and polynomial, not write division.

# Caller Owned Output

Sum, marshal, table, and clone results use only caller storage.

# State Compatibility

Serialized state validates identity, size, and table checksum before mutation.

# Clone

Clone copies initialized state into caller storage without aliasing.

# Bounds

Every source and destination call observes shared byte bounds.

# Invariant Domains

Tests exercise each scalar boundary and validation outcome.

# Allocation

Every exported runtime operation performs zero heap allocation.
