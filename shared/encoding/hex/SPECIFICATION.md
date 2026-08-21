
# Encode

Encoded_Size reports exact required storage. Encode_Into writes lowercase hexadecimal into caller
storage. Short or overlapping storage returns scalar status before mutation.

# Decode

Decoded_Size_Maximum reports safe caller capacity. Decode_Into accepts upper- and lowercase
digits, returns the decoded prefix, and distinguishes invalid digits from odd length. Short or
overlapping storage returns scalar status.

# Dump

Dump_Size reports exact hexdump storage. Dump_Into emits the hexdump -C layout into caller storage.
Source and output are bounded by formulas derived from line width and repository byte capacity.

# Bounds

Every byte slice is bounded by bytes.SLICE_SIZE_MAXIMUM or the largest source whose formula fits
that storage. Oversized slices panic. Source and destination cannot overlap.

# Allocation

Every exported operation performs zero heap allocation with assertion tracking enabled. Input and
output storage remain caller-owned.
