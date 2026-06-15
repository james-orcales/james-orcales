
# Always

`Recorder_Always` builds a guard whose condition must hold on every call; a false
one fails only when a `Dot_Product` consumes it.

### Violation

A `Dot_Product` given an `Always` that was observed false panics, naming the
offending axis by its site.

### Reachability

An `Always` the suite never reaches is reported as a coverage gap.

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
naming the boundary's site.

### Bad Bounds

Endpoints that are not distinct, a reversed or equal pair, panic at the
`Dot_Product`, naming the boundary's site.

# Impossible

`Impossible` declares element events that must never all occur together on one
call.

### Violation

When every named event is observed on the same call, the `Dot_Product` panics,
naming each co-occurring axis by its site and observed event.

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

Constructing a false `Always` enforces nothing until a `Dot_Product` consumes it.

### Identity

An element is identified by its caller location, the site both registration and the
runtime rendezvous on.

### Grid

Registration seeds one demanded tuple per surviving combination of the axes'
buckets, dropping the cells an `Impossible` carves.

### Attribution

A panic names every axis it found violated on the call, each by its site, not only
the first, so one run surfaces them all.

# Bundles

A `_Invariants` function returns a type's elements for a caller to spread into a
`Dot_Product`, so a type's properties travel and compose with it.

### Descent

Registration follows a bundle spread into a `Dot_Product` and seeds each of its
elements under the consuming callsite.

### Composition

A bundle that composes other bundles attributes every element, however deeply
nested, to the one top-level `Dot_Product` callsite that spread it.

### Binding

A bundle reached through a local binding or an append into the spread slice is
descended, not only the direct spread form.

### Casing

A bundle is recognized whether its type is exported (`_Invariants`) or unexported
(`_invariants`), matching the type's casing.

### Recorder Form

A bundle element is recognized whether built with the bare sugar primitive (`Sometimes`)
or the explicit `Recorder_Sometimes` form that leads with the recorder argument; the
condition is then read past that leading recorder.

### Sugar

A bundle in the sugar package may call the primitives unqualified; the descent recognizes
them because Sugar_Package names that package. Elsewhere a bare call is the caller's own
function, not a primitive, and seeds nothing unless qualified.

### Cross Package

A bundle defined in another package of the same module is resolved through the
module path and descended.

### Workspace

A bundle defined in a sibling module joined by a `go.work` workspace is resolved
through the workspace and descended.

### Callsite

Two callsites spreading one bundle become independent coverage entries, so neither
masks the other's gap.

### Gap Location

A bundle element's coverage gap is named at the compound `callsite::from=site` — the
consuming Dot_Product joined to the element's site in its bundle — so a composed bundle's
gap names the top-level callsite and the deepest nested site together.

### Failure Location

A bundle element's assertion failure message names only its own in-bundle site — for a
composed bundle the deepest nested one — never the consuming callsite. The callsite is not
lost: the panic's stack still unwinds through the Dot_Product, carrying it.

### Ban

A `Dot_Product` called inside a bundle is reported and fails registration.

### Orphan

An axis that never reaches a `Dot_Product` is reported and fails registration.

# Analysis

After the suite, every unexercised obligation is reported under its kind and the run
exits non-zero.

### Gaps

A never-fired obligation is named by its site and condition, while a fully exercised
one is left unreported.

### Combination

A grid cell the run never witnessed is reported as a cross-product gap, named by its
tuple of buckets.

### Boundary

A `Distinct_Boundary` endpoint the run never reached is reported as a boundary gap,
named by the value it bounds.

### Summary

A clean run reports how many properties it tested, splitting individual from
combination and counting the panic-able subset.

### Clean

With every obligation exercised, the analysis reports nothing and does not exit.

# Coverage

Coverage is never silently dropped: the analyzer either accounts for a construct or
fails on it.

### Modes

Coverage is recorded and checked only under a plain test run; a fuzz or benchmark
run records and checks nothing.

### Enforcement

A `Dot_Product` enforces its assertions on every call, in every run, even when no
coverage is being recorded.

### Unresolved

A bundle consumed by a `Dot_Product` that the analyzer cannot resolve is fatal, not
skipped.
