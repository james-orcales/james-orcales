
# Reference Value

Key_Into reproduces RFC 6070 and `crypto/pbkdf2` for every supported HMAC kind.

# Caller Owned Output

Derived bytes use caller storage. Refused work leaves that storage untouched. Empty output performs
no pseudorandom-function evaluation and succeeds.

# Work Bound

One call admits at most PRF_EVALUATION_COUNT_MAXIMUM HMAC evaluations. Required work is selected
digest block count multiplied by iteration count. Wider output therefore reduces the admitted
iteration count proportionally.

# Bounds

Password, salt, and output observe the shared byte bound. Iteration count is positive and cannot
exceed the one-block work limit.

# Invariant Domains

Runtime calls reach every owned scalar, collection boundary, and output status.

# Allocation

Every exported runtime operation performs zero heap allocation.
