
# Generator

The host constructor combines injected entropy, caller-owned random state, and the operating
system clock. Its V4 and V7 generators stamp their versions and produce valid UUIDs.

# Allocation

Construction uses caller-owned seed and generator storage without Go heap allocation.

# Replay

Construction preserves the supplied clock sequence, timestamp, and V7 replay state at every
accepted boundary, while reinitializing the random source from the supplied entropy.
