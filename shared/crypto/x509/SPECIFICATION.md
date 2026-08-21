
# Certificate

Parse_Certificate validates one canonical DER certificate and borrows its complete encoding,
TBSCertificate, issuer, subject, and signature. It accepts RSA-2048, P-256, and Ed25519 public
keys with SHA-256 RSA, SHA-256 ECDSA, or Ed25519 certificate signatures.

# Verification

Verify_Signature_From verifies the borrowed TBSCertificate against one parsed issuer key.
Unsupported algorithm and key combinations return false.

# Bounds

Certificate input follows the shared DER bound. Oversized input panics. Bounded malformed input
returns a scalar refusal without changing caller storage.

# Allocation

Parsing and verification perform zero heap allocation. Every variable-size field aliases caller
input; key material occupies fixed caller-owned storage.
