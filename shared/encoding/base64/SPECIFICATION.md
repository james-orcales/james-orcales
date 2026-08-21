
# Configuration

New_Encoding validates one 64-byte alphabet and padding atomically. Standard, URL, and raw
factories return configuration by value. With_Padding and Strict_Encoding validate a copy. Invalid
configuration returns scalar status.

# Encode

Encoded_Size reports exact required storage. Encode_Into writes caller storage and returns
STATUS_OUTPUT_TOO_SMALL before mutation when storage is short. NO_PADDING omits final padding.

# Decode

Decoded_Size_Maximum reports safe caller capacity. Decode_Into ignores CR and LF, validates padding,
and returns decoded prefix plus scalar status. NO_PADDING admits only tails containing whole bytes.

# Strict Decode

Strict_Encoding requires unused terminal bits to be zero. Newlines remain ignored. Noncanonical
terminal bits return STATUS_INPUT_INVALID without publishing that quantum.

# Bounds

Encoded storage holds at most bytes.SLICE_SIZE_MAXIMUM bytes. Decoded source and output use the
largest complete three-byte groups fitting that boundary. Oversized slices panic. Harmful overlap
returns STATUS_STORAGE_INVALID.

# Allocation

Every exported operation performs zero heap allocation with assertion tracking enabled.
Configuration, input, and output remain caller-owned values or storage.
