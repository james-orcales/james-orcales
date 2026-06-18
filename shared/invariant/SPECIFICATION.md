
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

# Distinct Boundary

`Recorder_Distinct_Boundary` asserts `Lo < Hi` and `Lo <= X <= Hi`, recording which
endpoint the value lands on.

### Endpoints

The `Hi` endpoint credits the true branch, the `Lo` endpoint the false branch, and
an interior value credits neither.

### Outside

A value beyond `[Lo, Hi]` panics when the boundary reaches a `Dot_Product`,
naming the boundary's message.

### Bad Bounds

Endpoints that are not distinct, a reversed or equal pair, panic at the
`Dot_Product`, naming the boundary's message.

# Impossible

`Impossible` declares element events that must never all occur together on one
call.

### Violation

When every named event is observed on the same call, the `Dot_Product` panics,
naming each co-occurring axis by its message and observed event.

### Absent

When the forbidden combination is not fully present, the `Dot_Product` does not
panic.

### Glob

An `Impossible` need not name every axis; the unnamed axes are wildcards, so it
carves every cell matching the named events across all their values.

# Dot Product

`Recorder_Dot_Product` is the only consumer: an element enforces and records
nothing until it is passed here.

### Inert

Constructing a `Sometimes` or `Distinct_Boundary` enforces and records nothing until a
`Dot_Product` consumes it. An `Always`, by contrast, is eager (see Always / Eager).

### Identity

Identity is the message, never a source location. A `Dot_Product` prefixes its own message
onto every axis it holds; an axis is identified by that compound prefix + own message, read
from literal arguments. Two `Dot_Product`s with distinct messages namespace their axes apart.

### Grid

Registration seeds one tuple per surviving combination of varying axes' buckets, dropping
cells an `Impossible` carves. The grid is over `Sometimes` and `Distinct_Boundary` only —
an `Always` is not an element. The grid is identified by the `Dot_Product`'s message.

### Attribution

A panic names every element it found violated on the call — each `Distinct_Boundary` or
`Impossible` by its message, not only the first. A false eager `Always` is not part of this; it
panics at its own site, so consecutive `Always` guards short-circuit on the first failure.

# Bundles

A `_Invariants` function returns a type's elements for a caller to spread into a
`Dot_Product`, so a type's properties travel and compose with it.

### Descent

Registration follows a bundle spread into a `Dot_Product` and seeds each of its
elements under the consuming `Dot_Product`'s message prefixed to the element's own message.

### Composition

A bundle that composes other bundles attributes every element, however deeply
nested, to the one top-level `Dot_Product`'s message prefix.

### Binding

A bundle reached through a local binding or an append into the spread slice is
descended, not only the direct spread form.

### Casing

A bundle is recognized whether its type is exported (`_Invariants`) or unexported
(`_invariants`), matching the type's casing.

### Recorder Form

A bundle element is recognized whether built with the bare sugar primitive (`Sometimes`)
or the explicit `Recorder_Sometimes` form that leads with the recorder argument; the
condition is then read past that leading recorder and the message past the condition.

### Sugar

A bundle in the sugar package may call the primitives unqualified; the descent recognizes
them because Sugar_Package names that package. Elsewhere a bare call is the caller's own
function, not a primitive, and seeds nothing unless qualified.

### Cross Package

A bundle defined in another package of the same module is resolved through the
module path and descended, so its elements' message literals are read.

### Workspace

A bundle defined in a sibling module joined by a `go.work` workspace is resolved
through the workspace and descended.

### Callsite

Spreading one bundle into two `Dot_Product`s with distinct messages yields independent
coverage entries — the per-field prefix keeps them apart, so neither masks the other's gap.
Reusing one message across two `Dot_Product`s is a duplicate and fails registration.

### Gap Location

A bundle element's (`Sometimes` / `Distinct_Boundary`) gap is named by the consuming
`Dot_Product`'s message prefixed to the element's own message. An eager `Always` in a bundle
body is not an element; its gap names its own bare message.

### Failure Location

A bundle element's deferred violation (a bad `Distinct_Boundary`, a triggered `Impossible`)
names only its own message, never the consuming prefix — yet the panic's stack still
unwinds through the Dot_Product, carrying it. An eager `Always` panics from its own frame.

### Ban

A `Dot_Product` called inside a bundle is reported and fails registration.

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

### Boundary

A `Distinct_Boundary` endpoint the run never reached is reported as a boundary gap,
named by the value it bounds.

### Summary

A clean run reports how many properties it tested, splitting individual from
combination and counting the panic-able subset.

### Clean

With every obligation exercised, the analysis reports nothing and does not exit.

# Coverage

Coverage of a consumed construct is never silently dropped: the analyzer either accounts for
it or fails on it. A bare `Sometimes` or `Distinct_Boundary` never passed to a `Dot_Product`
records nothing and is not flagged — consuming an element is the author's responsibility.

### Modes

Coverage is recorded in every mode but a benchmark — plain test, fuzz coordinator, and fuzz
worker all credit observations. Under -fuzz each worker appends to a shared file; the
coordinator unions that file before analysis. Enforcement fires in every mode.

### Enforcement

A `Dot_Product` enforces its `Distinct_Boundary` and `Impossible` assertions on every call,
in every run, even when no coverage is being recorded. An eager `Always` enforces
independently of any `Dot_Product`, also on every call in every run.

### Uniqueness

Every assertion message is a global identity; registration fails when two distinct assertions
claim the same one — two `Dot_Product`s sharing a message, a repeated axis, or two `Always`
sharing a message. A duplicate would mask a gap, so it is fatal, not merged.

### Literal

An assertion's message must be a compile-time string literal, so registration can seed it under
the same key the runtime emits. A non-literal message — a variable or a concatenation — cannot
be statically keyed, so its coverage would vanish; it fails registration, not silently.

### Unresolved

A bundle consumed by a `Dot_Product` that the analyzer cannot resolve is fatal, not
skipped.
