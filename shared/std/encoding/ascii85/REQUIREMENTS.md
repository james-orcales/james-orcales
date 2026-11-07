
# Encode

Encoded_Size_Maximum reports caller storage needed without reading source. Encoded_Size reports
exact storage after zero-block compression. Encode_Into writes into caller storage and returns
STATUS_OUTPUT_TOO_SMALL before one byte changes when destination cannot hold complete output.

# Decode

Decode_Into ignores bytes through ASCII space, expands z only between groups, and returns decoded
and consumed counts. Flush processes final partial group. Invalid digit, z inside one group, or one
trailing digit returns STATUS_INPUT_INVALID and publishes zero counts.

# Bounds

Each encoded or decoded slice holds at most bytes.SLICE_SIZE_MAXIMUM bytes. Encode source maximum is
largest whole group whose worst encoding fits. Source and destination never overlap. Oversize or
overlap causes panic. Valid input beyond destination returns STATUS_OUTPUT_TOO_SMALL.

# Allocation

Every exported operation performs zero heap allocation with assertion tracking enabled. Input and
output storage remain caller-owned.
