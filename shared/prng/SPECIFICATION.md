
# Seed Expands To State

New is deterministic: one seed always yields the same Generator, and two distinct
seeds yield Generators whose first draws differ.

# Known Sequence

A fixed seed produces a frozen sequence of Next values, locking the xoshiro256++
stream (seeded by splitmix64) against accidental change.

# Below Is Bounded

Below returns a value in the half-open range zero to bound, never the bound itself,
across many draws and across small and large bounds.

# Element Comes From Slice

Element returns one of the slice's own values; an empty slice is a precondition
violation that panics.

# Boolean Is Even

Boolean returns true and false, each over a large sample roughly half the time.

# Chance Matches Ratio

Chance is true at a frequency tracking the integer Ratio, exactly zero at a zero
numerator and always at a full one.

# Sample Matches Weights

Sample returns each outcome at a frequency tracking its integer weight, and never an
outcome of zero weight.

# Shuffle Permutes

Shuffle reorders a slice in place, preserving its multiset of elements and, over many
runs, reaching every permutation.

# Split Is Independent

Split returns a child Generator whose stream differs from the parent's continuation,
so a draw in one cannot perturb the other.

# Types Hold No Floating Point

The Generator, Ratio, and Distribution types hold only integer fields, so a run is
reproducible bit-for-bit across machines.

# Hot Path Is Zero Allocation

A steady-state Next draw performs no heap allocation.

# Bimodal Distribution Has Two Modes

Bimodal_Distribution returns a two-mode table from a fast value, a slow value, and the slow mode's
chance; every draw is one of the two values, never between.

# Percentile Distribution Hits Percentiles

Percentile_Distribution builds a table from the values at p25, p50, p75, p95, p99, and p100,
weighted by the mass between them, so a draw reproduces those percentiles.
