
# Package Owned State

Digest operations use caller-owned bounded state. Kind selects every hash available under
`shared/crypto`.

# Reference Values

Digest reproduces standard HMAC results for MD5, SHA-1, SHA-224, SHA-256, SHA-384, SHA-512/224,
SHA-512/256, and SHA-512.

# Stream

Writes preserve keyed inner state across every bounded call.

# Key Reduction

Keys larger than selected hash block reduce through selected hash before pad construction.

# Caller Owned Output

Digest output uses fixed values or caller storage. Short storage stays untouched and reports
selected width.

# Reset And Clone

Reset preserves selected key. Clone copies live state without aliasing.

# Constant Time Equality

Equal compares same-size tags without content-dependent exit.

# Bounds

Every key, source, destination, and tag observes shared byte bounds. Invalid state is rejected.

# Invariant Domains

Runtime calls reach each kind, state, collection boundary, and storage outcome.

# Allocation

Every exported runtime operation performs zero heap allocation.
