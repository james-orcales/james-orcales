
# Bounded Decompression

Decode_Into decompresses one zlib stream into caller-owned destination storage. Destination length
is hard output bound. Missing output storage returns STATUS_OUTPUT_TOO_SMALL; no growth occurs.

# Allocation

Every exported operation performs zero heap allocations. API returns written Count and scalar
Status.
Compressed input and destination storage stay caller-owned.

# Untrusted Input

Truncated, malformed, dictionary-dependent, checksum-invalid, and trailing input returns
STATUS_INPUT_INVALID. Decoder never panics for compressed bytes. Destination and compressed input
do not overlap.
