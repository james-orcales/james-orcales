
# Package Owned State

Digest operations use caller-owned bounded state. Kind selects SHA-224 or SHA-256.

# Formula

Digest, block, padding, state, schedule, and round geometry derive from shared primitive widths and
FIPS structure.

# Reference Values

Checksum and Digest reproduce FIPS 180-4 SHA-224 and SHA-256 values.

# Stream

Writes preserve selected SHA-2 state across every block and call boundary.

# Caller Owned Output

Digest output uses fixed values or caller storage. Short storage stays untouched and reports
selected width.

# Bounds

Every source and destination call observes shared byte bounds. Invalid state is rejected.

# Invariant Domains

Runtime calls reach each valid state and caller-storage boundary.

# Allocation

Every exported runtime operation performs zero heap allocation.
