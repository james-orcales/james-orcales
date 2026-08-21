
# Keys

Keys use exact-width odd 2048-bit moduli and the fixed public exponent 65537.
Private exponents are exact-width, nonzero, and smaller than the modulus.

# Encryption

OAEP with SHA-256 accepts messages up to its modulus-derived maximum. Encryption uses injected
entropy. Decryption validates all padding before transactional caller-output commit.

# Signatures

PSS uses SHA-256 and a digest-width injected salt. PKCS1 v1.5 uses SHA-256 DigestInfo.
Verification accepts bounded hostile signatures and returns false for malformed input.

# Bounds

All keys, ciphertexts, signatures, messages, and outputs have formula-derived limits.
Oversized input panics; bounded invalid input returns status or false.

# Allocation

Every exported runtime operation performs zero heap allocation.
