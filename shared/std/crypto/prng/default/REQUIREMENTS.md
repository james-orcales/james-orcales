
# Entropy

Receive the entropy reader and seed storage from the caller's composition root.
Require a non-nil reader that fills the complete seed without an error.

# Initialization

Initialize the ChaCha generator from that seed at the caller's bounded cursor position.
Deterministic injected seed readers must produce reproducible generator output.

# Allocation

Keep generator and seed storage caller-owned. Initialization must allocate no memory
when the injected reader allocates no memory.
