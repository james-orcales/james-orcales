
# Gzip Members

Encode_Into writes one RFC 1952 member. Decode_Into reads concatenated members and returns first
header. Decode_Member_Into reports exact successful member boundary, so caller can retain following
bytes. Rejected member header reports rejected input span.

# Header Metadata

Extra remains bytes. Name and Comment use UTF-8 in API and ISO 8859-1 on wire. Encoder rejects NUL,
invalid UTF-8, characters above U+00FF, oversized fields, and more than 511 wire text bytes. Decoder
writes first header into caller storage and verifies header CRC.

# Compression Levels

NO_COMPRESSION, BEST_SPEED through BEST_COMPRESSION, DEFAULT_COMPRESSION, and HUFFMAN_ONLY retain
standard library meanings. Invalid levels return STATUS_LEVEL_INVALID.

# Bounds

Source, compressed input, and decoded destination each hold at most 64 MiB. Destination length is
hard output bound. Header and workspace stay caller-owned; no operation grows storage. Short byte
or metadata destination returns corresponding scalar Status.

# Allocation

Every exported operation performs zero heap allocations. Input, output, header metadata, and
encoder workspace stay caller-owned.

# Untrusted Input

Truncated, malformed, checksum-invalid, size-invalid, and trailing input returns scalar Status.
Decoder never panics for compressed bytes. Harmful storage overlap returns STATUS_STORAGE_INVALID.
