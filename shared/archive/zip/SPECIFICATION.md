
# Bounded Entry

Decode_Into reads shared/bytes Slice, finds first classic ZIP member, and writes caller Slice.
Declared output size must fit destination. Stored and DEFLATE members decode locally.
Encrypted, multi-disk, ZIP64, unsupported, corrupt, or mismatched input returns Status, never panic.

# Allocation

Every exported operation performs zero heap allocations. Decoder uses fixed stack tables and
destination history. No reader, error, filename, table, or decompressed-output object escapes.
