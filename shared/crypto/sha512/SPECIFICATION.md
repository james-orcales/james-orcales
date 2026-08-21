
# Package Owned State

Digest operations use caller-owned bounded state. Kind selects four FIPS 180-4 functions.

# Formula

Digest, block, padding, state, schedule, and round geometry derive from shared primitive widths and
FIPS structure.

# Reference Values

Checksum_Into and Digest_Sum_Into reproduce SHA-384, SHA-512, SHA-512/224, and SHA-512/256 values.

# Stream

Writes preserve selected SHA-2 state across every block and call boundary.

# Caller Owned Output

Digest output uses caller storage. Short storage stays untouched and reports selected width.

# Bounds

Every source and destination call observes shared byte bounds. Invalid state is rejected.

# Invariant Domains

Runtime calls reach each valid state and caller-storage boundary.

# Allocation

Every exported runtime operation performs zero heap allocation.
