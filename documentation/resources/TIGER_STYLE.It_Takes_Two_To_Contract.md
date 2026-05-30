# It Takes Two to Contract

**Author:** matklad
**Published:** Dec 27, 2023
**Source:** https://tigerbeetle.com/blog/2023-12-27-it-takes-two-to-contract/

## Introduction

The post explores design by contract (DbC), initially presented as skepticism about the concept. The
author argues that DbC fundamentally relies on two key mechanisms: types and assertions.

## Core Argument: Types and Assertions Suffice

Most DbC benefits can be achieved through straightforward language features:

- **Preconditions:** Standard `assert` statements work effectively
- **Postconditions:** Use `defer` with assertions

```zig
assert(!grid.free_set.opened);
defer assert(grid.free_set.opened);
```

The only additional need might be naming function results for cleaner postconditions, though moving
assertions before return statements typically suffices.

## Where DbC Needs Language Support

First-class DbC support becomes necessary in inheritance-based OOP,
where derived classes must weaken preconditions and strengthen postconditions. This suggests
avoiding inheritance-based polymorphism as an alternative.

The Nickel configuration language demonstrates strong synergy between gradual typing, contracts, and
lazy evaluation — an exceptional case where DbC shines.

## The Central Insight: Contracts Require Two Parties

"Contracts always involve two parties." Meaningful assertions appear at both the call site and the
function definition site.

### Example from TigerBeetle

**Call site:** ```zig const filled = compaction.fill_immutable_values(target);
assert(filled <= target.len);
```

**Definition site:**
```zig
fn fill_immutable_values(compaction: *Compaction, target: []Value) usize {
    assert(target_count <= source_count);
    ...
}
```

## Benefits of Paired Assertions

### Readability and Clarity

Assertions at call sites immediately clarify semantics. The paired assertions form an "airlock" —
you can verify compatibility between assertions without keeping both definition and call site in
mind simultaneously.

### Robustness Through Defense in Depth

- Preconditions strengthen or relax as code evolves
- Separate assertion pairs at each site prevent refactoring errors
- Assertions using different local variable vocabularies increase bug detection likelihood

### Property-Based Testing Principles

The technique mirrors property-based testing: performing identical computations through different
code paths and verifying identical results.

## Advanced Applications

### Consensus Protocols and Hash-Chaining

In distributed systems like TigerBeetle, replicas assert the same invariants but with different
local knowledge of cluster state. Through cryptographic hash-chaining, replicas can verify
consistency without complete state synchronization.

### Multiple State Paths

States can be reached through different mechanisms:

- Processing events from initial empty state
- Loading from snapshots after crashes

Recording checksums in snapshots ensures equivalence across different
paths.

## Recommendations

1. **Think in pairs:** When writing assertions, consider which distant
   code depends on them
2. **Pair trivial cases:** Add equivalent assertions even for simple
   function-caller relationships
3. **Hunt for interesting patterns:** Look for assertion pairs
   separated by different implementations, process boundaries, or time
   delays

## Conclusion

Paired assertions represent a practical, defense-in-depth approach to
correctness that extends beyond simple function contracts into
distributed systems verification.
