

# Hash Matches Reference Vectors

The one-shot Hash reproduces the published XXH64 known-answer values across empty, sub-word,
word-sized, and 63-byte inputs, and across several seeds — matching the algorithm's reference,
not a self-snapshot.

# Digest Matches Reference Vectors

The streaming Digest reproduces those same vectors when the input is fed in chunks of every size,
so the partial-stripe buffering is correct across Write boundaries.

# Digest Equals One Shot

Streaming and one-shot agree on a longer input split at many different chunk sizes, the case the
fixed vectors underweight.

# Write Reports Full Count

Digest_Write consumes and reports every byte.

# Reset Restores Initial State

Digest_Reset returns a used Digest to the state of a fresh one with the same seed.

# Hot Path Is Zero Allocation

A one-shot Hash of a preallocated slice performs no heap allocation.
