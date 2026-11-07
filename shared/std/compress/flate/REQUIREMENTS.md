
# Raw DEFLATE

Encode_Into writes one RFC 1951 stream. Decode_Into reads exactly one RFC 1951 stream.
Decode_Prefix_Into reports consumed compressed bytes for framing footer retention.
Dictionary forms retain at most DICTIONARY_SIZE_MAXIMUM bytes.

# Compression Levels

NO_COMPRESSION, BEST_SPEED through BEST_COMPRESSION, DEFAULT_COMPRESSION, and HUFFMAN_ONLY retain
standard library meanings. Invalid levels return STATUS_LEVEL_INVALID.

# Bounds

Source, compressed input, and destination each hold at most BYTE_COUNT_MAXIMUM bytes. Destination
length is hard output bound. Encoder workspace is caller-owned and fixed-size. No operation grows
storage. Short destination returns STATUS_OUTPUT_TOO_SMALL with count of complete bytes written.

# Allocation

Every exported operation performs zero heap allocations. Input, output, dictionary, and encoder
workspace stay caller-owned.

# Dictionaries

Dictionary forms preserve standard library preset-history behavior without owning history storage.

# Untrusted Input

Truncated, malformed, reserved, trailing, and impossible-distance compressed input returns
STATUS_INPUT_INVALID. Decoder never panics for compressed bytes. Overlapping input and output or
missing encoder workspace returns STATUS_STORAGE_INVALID.
