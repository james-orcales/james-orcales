
# Reference Points

Point_Generator and Point_Identity match NIST P-256 and SEC 1.

# Encoding

Point_Set_Bytes accepts SEC 1 infinity, compressed, and uncompressed encodings. Invalid input leaves
the destination unchanged. Point_Bytes_Into writes only complete caller-owned encodings.

# Group Operations

Point_Add and Point_Double use complete formulas for the P-256 `a = -3` curve. Every public point
boundary validates canonical projective coordinates; the zero value denotes infinity.

# Scalar Multiplication

Scalar multiplication scans every bit, uses complete formulas, and selects without scalar-dependent
control flow. Scalars have exact P-256 width and reduce modulo the group order.

# Bounds

Encoding, output, and scalar storage have explicit SEC 1 or P-256 limits. Oversized storage causes
panic. Invalid bounded encodings and scalar lengths return status before destination mutation.

# Invariant Domains

Runtime calls reach every owned collection, enum, count, status, and decision value.

# Allocation

Every exported runtime operation performs zero heap allocation.
