
# Injected Entropy

Read consumes only the caller-owned CSPRNG. Operating-system seeding remains in the existing
composition tier, so tests and simulations reproduce a seed exactly.

# Caller Owned Output

Read fills the complete destination and reports its length. Empty destination is valid.

# Bounds

Destination size follows the injected CSPRNG sink limit. Missing state or oversized storage causes
panic before a draw.

# Invariant Domains

Runtime calls reach every output and count boundary.

# Allocation

Every exported runtime operation performs zero heap allocation.
