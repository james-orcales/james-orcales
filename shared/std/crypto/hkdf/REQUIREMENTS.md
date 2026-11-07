
# Reference Values

Extract, Expand, and Key reproduce RFC 5869 values and `crypto/hkdf` for every supported HMAC kind.

# Extract And Expand

Extract emits selected hash width. Expand chains every prior block and counter through selected
HMAC.

# Caller Owned Output

Derived bytes use caller storage. Refused output stays untouched.

# Output Limit

Expand accepts at most 255 selected-hash blocks.

# Bounds

Secret, salt, pseudorandom key, and info observe shared byte bounds. Output has formula-derived
RFC capacity.

# Invariant Domains

Runtime calls reach each kind, collection boundary, and output status.

# Allocation

Every exported runtime operation performs zero heap allocation.
