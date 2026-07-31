
# Load

Load finds the workspace, discovers first-party Go packages, and type-checks their source through
injected filesystem operations. The operating-system bindings stay in the composition root.

### Discovers Source Through Injections

Load walks from the nearest go.mod, reads package source through the injected bounded readers, and
returns the type-checked packages in deterministic import-path order.

# Chain

### Root To Leaf

Graph_Chain traces an injected field's value from the composition root that wired it, along the
value's flow edges, to the leaf that invokes it, rendering the ordered functions as an injection
chain. It reports the field absent when no wiring reaches it.

# Resolve

### Wires Invokes Into Calls

Graph_Resolve rewrites each invoke into a resolved call by binding it to the callable its field
was wired to, appending the derived edges to the graph's calls.

# Broken

### Unwired Invoke

Graph_Broken reports every field an invoke dispatches through that no wiring reaches — a severed
injection chain. The result is sorted and deduplicated.

# Diff

### Added And Removed Calls

Graph_Diff compares two resolved graphs and reports the calls present in one but not the other:
added holds calls new in the later graph, removed holds calls gone from the earlier one.
