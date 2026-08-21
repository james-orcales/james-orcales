
# Diff Into

Diff Into renders minimal rune edit script into caller-owned output. Caller supplies decoded-rune
and dynamic-programming matrix storage. Oversized text, short workspace, and short output return
distinct scalar statuses. Operation performs zero allocations.

### Cases

Identity, insertion, deletion, substitution, quote escaping, Unicode, bounds, status paths, and
allocation proof preserve script semantics.

# Line Diff Into

Line Diff Into renders minimal line edit script into caller-owned output. Each emitted line starts
with space, plus, or minus. Operation uses same caller-owned workspace and performs zero
allocations.

### Cases

Empty, identity, insertion, deletion, replacement, trailing newline, bounds, status paths, and
allocation proof preserve line semantics.

# Diff Boundaries

Diff Boundaries proves every accepted character output and workspace boundary.

### Cases

Zero, one, two, maximum, and first rejected bounds reach public status paths.

# Line Diff Boundaries

Line Diff Boundaries proves every accepted line output and workspace boundary.

### Cases

Zero, one, two, maximum, and first rejected bounds reach public status paths.

# Find Common Prefix

Find Common Prefix returns longest shared leading rune run without allocation.

### Cases

Empty, disjoint, partial, Unicode, and maximum-size inputs return bounded borrowed results.

# Find Common Suffix

Find Common Suffix returns longest shared trailing rune run without allocation.

### Cases

Empty, disjoint, partial, Unicode, and maximum-size inputs return bounded borrowed results.

# Find Common Run

Find Common Run returns longest shared run only when run spans at least half longer input.

### Cases

Empty, qualifying, too-short, Unicode, and maximum-size inputs return bounded borrowed results.

# Runes Have Prefix

Runes Have Prefix reports whether nonempty expected run begins input.

### Predicate

Matching, empty, oversized, and different expected runs cover both results.

# Runes Have Suffix

Runes Have Suffix reports whether nonempty expected run ends input.

### Predicate

Matching, empty, oversized, and different expected runs cover both results.
