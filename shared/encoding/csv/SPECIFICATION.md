
# Configuration

New_Configuration validates delimiter, comment, width, quote, trim, and line-ending policy.
Standard_Configuration selects comma, inferred width, strict quotes, no comment, and line feeds.
Decoder_Init binds validated policy to caller-owned cross-record state.

# Encode

Encoded_Size reports exact record storage. Encode_Into writes one quoted record to caller storage.
Invalid configuration, oversized records, short output, and overlap return status.

# Decode

Decode_Record_Into skips blank and comment lines, normalizes CRLF, and parses one record.
Multiline values and positions write into caller storage. Caller owns decoder state.
Results include field count, consumed input, parse position, and scalar status.

# Bounds

Collections follow bytes.SLICE_SIZE_MAXIMUM or separator-derived field-count formula.
Oversized collections panic. Bounded fields can combine into oversized-record status.

# Allocation

Every exported operation performs zero heap allocation with assertion tracking enabled.
Encoded, decoded, field, and decoder storage remain caller-owned.
