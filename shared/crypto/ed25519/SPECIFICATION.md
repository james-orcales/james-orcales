
# Keys

Private keys derive from exact-width RFC 8032 seeds. Public keys occupy fixed caller storage.

# Signatures

Sign writes one deterministic fixed-width Ed25519 signature into caller storage.
Verify accepts bounded hostile signature bytes and returns false for malformed input.

# Bounds

Messages follow the repository byte-sequence bound. Oversized messages and signatures panic.

# Allocation

Every exported runtime operation performs zero heap allocation.
