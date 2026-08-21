
# Keys

Private keys are exact-width canonical nonzero P-256 scalars. Public keys are finite validated
P-256 points. Setters reject bounded invalid encodings before destination mutation.

# Signatures

Sign writes fixed-width IEEE P1363 `r || s` into caller storage for one SHA-256 digest.
An injected generator supplies nonce candidates; scalar width bounds rejection. Output uses low S.
Verify accepts bounded hostile bytes and rejects malformed, zero, noncanonical, or invalid values.

# Bounds

Every byte input has an explicit P-256-derived maximum. Oversized input causes panic. Bounded
invalid input returns status or false.

# Allocation

Every exported runtime operation performs zero heap allocation.
