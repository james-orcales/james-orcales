
# Allocation

Every exported operation performs zero heap allocation. Append forms use existing caller capacity.
Structured stream forms use explicit caller scratch storage. Insufficient append or scratch storage
causes panic before partial output escapes.

# Bounds

Bytes and structured values hold at most 4096 encoded bytes. Inputs beyond this boundary cause
panic. Fixed-width reads and writes require their complete width.

# Byte Order

LITTLE_ENDIAN places least-significant byte first. BIG_ENDIAN places most-significant byte first.
NATIVE_ENDIAN selects target machine order. Byte_Order_String and Byte_Order_Go_String name each
order without interface methods.

# Fixed Width

Uint_16, Uint_32, and Uint_64 decode unsigned integers. Put_Uint_16 through Put_Uint_64 encode into
caller storage. Append_Uint_16 through Append_Uint_64 extend one caller-owned buffer.

# Structured Values

Size reports fixed-size grammar of bool and fixed-width integers. Arrays, slices, structs, and one
outer pointer compose those leaves. Blank fields reserve zero bytes. Encode, Decode, and Append use
caller storage; invalid grammar returns Error_Invalid_Type or VALUE_SIZE_INVALID.

# Stream IO

Reader and Writer bind injected nbio.Stream transport and caller scratch. Operations retire through
caller-owned completions without driving timeline. Read joins partial transfers before decode and
requires pointer destination. Empty source is Stream_EOF; truncated source is Stream_Unexpected_EOF.

# Varints

Unsigned varints encode seven payload bits per byte. Signed varints use zig-zag mapping. Put forms
write caller storage. Append forms extend caller capacity. Decode forms return zero count for
incomplete input and negative consumed count for overflow. Read forms consume at most ten bytes.

# Domain Errors

Error_Buffer_Too_Small, Error_Invalid_Type, and Error_Varint_Overflow are stable sentinels.
Oversized values, invalid byte order, short fixed-width storage, short scratch storage, or
insufficient append capacity cause panic before partial output escapes.
