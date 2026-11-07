
# Package Owned State

Hash operations use package-owned bounded types and caller-owned state.

# Reference Values

Bytes reproduces SipHash-2-4 vectors for injected key and zero output mask.

# Write Division

Bytes, text, single bytes, and chunks form identical byte streams.

# Seed Reset And Clone

Reset and clone retain explicit function identity in caller storage.

# Caller Owned Output

Hash_Sum_Into writes standard little-endian value into caller storage.

# Bounds

Per-call and logical-message bounds reject overflow without state mutation.

# Invariant Domains

Tests exercise each scalar boundary and validation outcome.

# Allocation

Every exported runtime operation performs zero heap allocation.
