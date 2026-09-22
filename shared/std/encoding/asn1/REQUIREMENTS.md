
# Element

Element is one borrowed DER tag-length-value. Class, tag, construction bit, and content stay
explicit. Decode consumes one element and leaves trailing input to the caller.

# Encode

Encoded_Size reports exact canonical DER storage. Encode_Into writes caller storage only after
element, aggregate size, capacity, and overlap checks succeed.

# Decode

Decode accepts canonical definite DER lengths and minimal high-tag identifiers. It reports the
first invalid byte without an owned error or hidden buffer.

# Bounds

Input, output, content, identifiers, lengths, and positions follow formulas from the shared byte
limit and DER bit fields. Oversized slices panic; oversized aggregates return scalar status.

# Allocation

Every exported operation performs zero heap allocation with assertion tracking enabled. Decoded
content borrows the encoded input; encoded output remains caller-owned.
