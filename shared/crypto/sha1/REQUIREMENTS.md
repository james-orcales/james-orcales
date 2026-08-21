
# Package Owned State

Digest operations use caller-owned bounded state. Package owns no mutable global state.

# Formula

Digest, block, padding, state, schedule, and round geometry derive from shared primitive widths and
FIPS structure.

# Reference Values

Checksum_Into and Digest_Sum_Into reproduce FIPS 180-4 SHA-1 values.

# Stream

Writes preserve SHA-1 state across every block and call boundary.

# Caller Owned Output

Digest output uses caller storage. Short storage stays untouched and reports required width.

# Bounds

Every source and destination call observes shared byte bounds. Uninitialized state is rejected.

# Invariant Domains

Runtime calls reach each valid state and caller-storage boundary.

# Allocation

Every exported runtime operation performs zero heap allocation.
