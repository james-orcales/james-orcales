
# Always

`Recorder_Always` is an eager guard whose condition must hold on every call. Unlike the
element kinds it is not a `Dot_Element` and never flows through a `Dot_Product`: it
returns nothing and enforces on the spot.

### Violation

A false `Always` panics at its own call site, in every run mode, naming itself by its
message. There is no deferred phase — see Eager.

### Eager

Constructing a false `Always` panics immediately; an `Always` is never inert and never
consumed. Contrast the element kinds, which stay inert until a `Dot_Product` (see Dot
Product / Inert).

### Reachability

An `Always` the suite never reaches is reported as a coverage gap, named by its message. A
bare `Always` is discovered by the same source scan that registers `Dot_Product` calls,
so its reachability is tracked without being consumed.

# Sometimes

`Recorder_Sometimes` builds an axis claiming the run observes its condition both
true and false; alone it never panics.

### Coverage

A consumed `Sometimes` credits its true branch on a true event and its false
branch on a false event.

### Gap

A `Sometimes` observed only one way, true without false or false without true, is
reported as a coverage gap.

# Impossible

`Impossible` declares element events that must never all occur together on one
call, naming each by the message of a sibling axis — `Event_True("m")` /
`Event_False("m")` — not by holding the axis value.

### Violation

When every named event is observed on the same call, the `Dot_Product` panics,
naming each co-occurring axis by its message and observed event.

### Absent

When the forbidden combination is not fully present, the `Dot_Product` does not
panic.

### Glob

An `Impossible` need not name every axis; the unnamed axes are wildcards, so it
carves every cell matching the named events across all their values.

### Sibling

A reference names an axis of its own `Dot_Product`. Naming a non-sibling panics at the `Dot_Product`
on every call — a structural precondition checked before recording — so a typo is caught at once,
not as an unfillable gap.

# Imply

`Imply` builds a gated `Sometimes`: an axis recorded only on a call where its prerequisite holds.
The message-less prerequisite is no axis. The condition is evaluated eagerly, so one safe only under
the prerequisite must self-guard (`p != nil && p.x`) — the prerequisite gates recording, not eval.

### Gated

A gated axis credits a branch only on a call where the prerequisite held; a call where it
failed credits neither branch, so a failing prerequisite never stands in for the gated false event.

### Excluded

A gated axis is per-axis coverage only — it joins no tuple of the grid, since the message-less
prerequisite is not an axis to cross with.

### Conjunction

An axis meaningful only under several prerequisites gates on their conjunction: it records only on a
call where every one holds.

# Dot Product

`Recorder_Dot_Product` is the only consumer: an element enforces and records
nothing until it is passed here.

### Inert

Constructing a `Sometimes` enforces and records nothing until a `Dot_Product` consumes it.
An `Always`, by contrast, is eager (see Always / Eager).

### Identity

Identity is the message, not a source location. A `Dot_Product` prefixes its message onto each
axis, keying it prefix + own message. The prefix is a literal: the `Dot_Product`'s own argument,
or the literal namespace at each `_Invariants` callsite when it self-emits under its parameter.

### Grid

Registration seeds one tuple per surviving combination of the ungated axes' buckets, dropping
cells an `Impossible` carves. The grid is over ungated `Sometimes` axes only — an `Always` is not an
element and a gated `Imply` axis is excluded (see Imply / Excluded). The grid is the prefix.

### Attribution

A panic names every element it found violated on the call — each triggered `Impossible`, not
only the first. A false eager `Always` is not part of this; it
panics at its own site, so consecutive `Always` guards short-circuit on the first failure.

# Bundles

A `_Invariants(v, namespace)` function self-emits its own `Dot_Product(namespace, …)` over a
type's axes, so a type's properties travel and compose with it.

### Descent

Registration follows a `_Invariants(v, "lit")` call, resolves the function, and seeds the grid its
body self-emits — keyed by the callsite's literal namespace prefixed onto each axis's own message.

### Composition

A `_Invariants` that calls other `_Invariants` registers each as its own self-contained grid under
its own namespace — never flattened into the parent. There is no joint cross-product across types.

### Casing

A bundle is recognized whether its type is exported (`_Invariants`) or unexported
(`_invariants`), matching the type's casing.

### Sugar

A bundle in the sugar package may call the writers unqualified; the scan recognizes them because
Sugar_Package names that package. Elsewhere a bare call is the caller's own function, not a writer,
and seeds nothing unless qualified.

### Cross Package

A `_Invariants` in another package of the same module is resolved through the module path. Import
paths resolve relative to the `go.mod` module path — a plain prefix match, so a non-URL `module
local` works too; a `_Invariants` in a module outside this `go.mod` is unresolvable and fatal.

### Callsite

Calling one `_Invariants` at two callsites with distinct namespaces yields independent grids — the
per-namespace prefix keeps them apart, so neither masks the other's gap. Reusing one namespace is a
duplicate and fails registration.

### Gap Location

A bundle axis's gap is named by the callsite namespace prefixed onto the axis's own message. An
eager `Always` in a bundle body is not an axis; its gap names its own bare message.

### Failure Location

A carve's violation names the co-occurring axes by their own message, never the namespace prefix —
yet the panic's stack still unwinds through the `Dot_Product`, carrying it. An eager `Always` panics
from its own frame.

### Static

A `_Invariants` body must be straight-line: a branching or looping statement (`if`, `switch`, `for`,
`select`) fails registration, since it would make the axes it self-emits depend on runtime values.

# Analysis

After the suite, every unexercised obligation is reported under its kind and the run
exits non-zero.

### Gaps

A never-fired obligation is named by its message and condition, while a fully exercised
one is left unreported.

### Combination

A grid cell the run never witnessed is reported as a cross-product gap, named by its
tuple of buckets.

### Legend

A cross-product gap prints its grid's axis legend once, each position named by kind,
condition, and message (the axis's own message), and decodes each cell's buckets
back to the events they stand for, so a bare coordinate is debuggable across nested bundles.

### Summary

A clean run reports how many properties it tested, splitting individual from
combination and counting the panic-able subset.

### Clean

With every obligation exercised, the analysis reports nothing and does not exit.

# Coverage

Coverage of a consumed construct is never silently dropped: the analyzer either accounts for
it or fails on it. A bare `Sometimes` never passed to a `Dot_Product` records nothing and is
not flagged — consuming an element is the author's responsibility.

### Modes

Coverage is recorded in every mode but a benchmark — plain test, fuzz coordinator, and fuzz
worker all credit observations. Under -fuzz each worker appends to a shared file; the
coordinator unions that file before analysis. Enforcement fires in every mode.

### Enforcement

A `Dot_Product` enforces its `Impossible` assertions on every call,
in every run, even when no coverage is being recorded. An eager `Always` enforces
independently of any `Dot_Product`, also on every call in every run.

### Uniqueness

Every assertion message is a global identity; registration fails when two distinct assertions
claim the same one — two `Dot_Product`s sharing a message, a repeated axis, or two `Always`
sharing a message. A duplicate would mask a gap, so it is fatal, not merged.

### Literal

An assertion's message must be a compile-time string literal, so registration can seed it under the
same key the runtime emits. A non-literal message — a variable or a concatenation — cannot be keyed
statically; its coverage would vanish, so it fails registration. `Impossible` references obey it.

### Unresolved

A bundle consumed by a `Dot_Product` that the analyzer cannot resolve is fatal, not
skipped.
