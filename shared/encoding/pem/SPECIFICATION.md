
# Encode

Encoded_Size reports exact PEM storage. Encode_Into writes caller-ordered headers, wrapped standard
base64, and matching boundary lines into caller storage. Invalid blocks, oversized output, short
output, and overlap return scalar status before mutation.

# Decode

Decode_Into finds the first line-bound PEM block, borrows metadata from input, decodes its body into
caller storage and reports consumed input. Both line endings and horizontal base64 space work.
Missing, malformed, short, and overlapping input return scalar status.

# Bounds

Every collection follows a formula derived from bytes.SLICE_SIZE_MAXIMUM, PEM marker widths, base64
quantum width, and base64 line width. Oversized collections panic. Exact aggregate size remains a
status because individually bounded block fields can still exceed bounded encoded output together.

# Allocation

Every exported operation performs zero heap allocation with assertion tracking enabled. Input,
output, decoded bytes, and header slots remain caller-owned.
