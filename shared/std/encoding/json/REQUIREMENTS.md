
# Validate

Validate accepts one complete encoding/json lexical value and reports first invalid byte position.
Invalid UTF-8 inside strings remains syntax because replacement belongs value decoding. Nesting,
input bytes, and error position remain bounded by package formulas.

# Compact

Compact_Size reports exact whitespace-free size. Compact_Into writes caller storage.
Invalid input, short output, and overlap return scalar status before output mutation.

# Indent

Indent_Size reports exact formatted size for caller prefix and indent bytes.
Indent_Into preserves string bytes and formats structural punctuation into caller storage.

# Bounds

Source, destination, prefix, and indent slices allow 64 KiB each. Token storage allows 65,536
entries and nesting allows 32,768 levels, derived from the input bound.
Oversized slices panic. Formatted output beyond package bound returns scalar status.

# Decode

Decode_Size sizes tokens and decoded bytes. Decode_Into preserves nesting, member order, duplicate
names, and exact numbers; strings replace malformed UTF-8 and unpaired surrogates with U+FFFD.
Bounded caller storage changes only on success. Tokens retain no source references.

# Allocation

Every exported operation performs zero heap allocation with assertion tracking enabled.
Parser stack and output remain bounded value or caller-owned storage.
