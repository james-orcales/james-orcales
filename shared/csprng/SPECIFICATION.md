
# Seed Expands To State

New is deterministic: one seed always yields the same Generator stream, and two distinct
seeds yield Generators whose first draws differ.

# Block Matches Reference Vectors

The ChaCha20 core reproduces RFC 8439's official vectors: the quarter-round of section 2.1.1,
the block function of section 2.3.2, and the two block vectors of Appendix A.1. Matching a
published spec, not a self-snapshot, anchors the core before fast-key-erasure hides it.

# Known Sequence

A fixed seed produces a frozen sequence of Uint64 values, locking the fast-key-erasure
ChaCha20 stream against accidental change. It is not the raw RFC keystream, since the first
32 bytes of every block reseed the key; the contract is per-version reproducibility.

# Read Fills Fully

Read fills the whole of its buffer, reports the full byte count, and never errors, across
sizes that stay within one refill, exactly fill it, and cross into the next; the bytes it
delivers are real keystream, never a silent run of zeros. It satisfies io.Reader.

# Bytes Are Uniform

A filled buffer sets close to half of all its bits over a large sample, the balance a fair
keystream holds.

# Below Is Bounded

Below returns a value in the half-open range zero to bound, never the bound itself, across
many draws and across small and large bounds; a non-positive bound is a precondition
violation that exits.

# Seed Is Erased After Construction

New performs the first fast-key-erasure refill, so the caller's seed no longer lives in the
Generator's key when New returns; a later disclosure of the key cannot reconstruct the seed
or the output that refill already produced.

# Hot Path Is Zero Allocation

A steady-state Read performs no heap allocation, even with the draw path's invariant assertions
active under coverage recording.
