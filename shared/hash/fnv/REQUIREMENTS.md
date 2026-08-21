
# Package Owned State

Each width uses package-owned bounded types and caller-owned state.

# Reference Values

All supported widths reproduce standard FNV-1 and FNV-1a answers.

# Offsets And Reset

Reset retains Kind and restores width-specific offset basis.

# Caller Owned Output

Sum, marshal, and clone results use only caller storage.

# State Compatibility

Width-specific state validates exact size and Kind identity before mutation.

# Clone

Clone copies initialized state into caller storage without aliasing.

# Bounds

Every source and destination call observes shared byte bounds.

# Invariant Domains

Tests exercise each scalar boundary and validation outcome.

# Allocation

Every exported runtime operation performs zero heap allocation.
