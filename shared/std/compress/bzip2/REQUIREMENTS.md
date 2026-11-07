
# Bounded Decompression

Decode_Into decompresses one bzip2 stream into caller-owned destination storage with hard bound.
Caller supplies uint32 transform storage fitting stream header's declared block item count.
Missing output or transform storage returns status; no growth occurs.

# Allocation

Every exported operation performs zero heap allocations. API returns written Count and scalar
Status.
Compressed input, destination, and transform storage stay caller-owned.

# Untrusted Input

Truncated, malformed, randomized, checksum-invalid, and trailing input returns STATUS_INPUT_INVALID.
Decoder never panics for compressed bytes. Destination and compressed input do not overlap.
