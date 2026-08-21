
# Configuration

New_Encoding validates one 32-byte alphabet and padding atomically. Standard_Encoding and
Hexadecimal_Encoding return immutable-by-copy RFC 4648 configurations. With_Padding validates a
replacement without changing its input. Invalid configuration returns scalar Status.

# Encode

Encoded_Size reports exact required storage. Encode_Into writes into caller storage and returns
STATUS_OUTPUT_TOO_SMALL before mutation when storage is short. Padded output uses complete
eight-symbol groups. NO_PADDING omits only final padding.

# Decode

Decoded_Size_Maximum reports safe caller capacity without reading input. Decode_Into ignores CR and
LF, validates alphabet and padding, then returns decoded prefix and scalar Status. Padded input ends
at its final quantum. NO_PADDING admits only tails containing a whole decoded byte.

# Bounds

Encoded storage holds at most bytes.SLICE_SIZE_MAXIMUM bytes. Decoded source and output use the
largest complete five-byte groups fitting that encoded boundary. Source and destination cannot
overlap. Oversized slices panic. Harmful overlap returns STATUS_STORAGE_INVALID.

# Allocation

Every exported operation performs zero heap allocation with assertion tracking enabled.
Configuration, input, and output remain caller-owned values or storage.
