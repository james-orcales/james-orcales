
# Always

`Recorder_Always` is an eager, bare guard whose condition must hold on every call. It remains
message-keyed reachability and is neither a chain link nor part of a demanded grid.

### Violation

A false `Always` panics at its own callsite in every run mode, naming itself by its message.

### Eager

`Always` enforces and credits when called; it is never deferred until a `Dot_Product.Ensure`.

### Reachability

An `Always` the suite never reaches is a coverage gap. Registration discovers the bare call and
seeds its literal message without requiring a product namespace.

# Sometimes

`Sometimes` exists only as a `Product` chain link. Each link claims that its condition is witnessed
both true and false; there is no bare form and no caller-supplied global identity.

### Coverage

A link only captures its observed branch by fluent ordinal. `Ensure` projects that raw observation
through registration's axis plan and credits its pre-resolved handle; repeated messages remain
distinct because link ordinal is part of the registered identity.

### Gap

An axis observed only true or only false is reported under its chain key as a coverage gap.

# Dot Product

`Recorder_Dot_Product(recorder, namespace)` returns a value `Product`; `Dot_Product(namespace)`
uses the default recorder. The links return advanced copies and `Ensure` terminates.

### Links

The only links are `Sometimes(condition, message)` and `Impossible(message, references...)`.
They retain only the facts needed for deferred shape checks; `Ensure` alone exposes validation,
enforces constraints, and credits coverage. References use `Event_True` and `Event_False` polarity.

### Identity

One literal namespace identifies one chain. An axis is keyed by namespace, link ordinal, and its
literal message; registration constructs that identity and its coverage handle exactly once.

### Constraint

`Impossible` carves every grid cell matching its referenced polarities and globs over unnamed axes.
Registration binds each link directly to its compiled rule and resolved siblings; warmed replay uses
those positions without rebuilding masks or scanning axes. `Ensure` panics for every match in order.

### Sibling

Each reference must resolve to exactly one preceding axis in the same chain. Missing references,
ambiguous repeated-message references, repeated references, and empty rules panic only at `Ensure`.

### Shared

Executions sharing a namespace must traverse the same links, messages, references, and order.
`Ensure` panics on a different shape so one namespace can never combine unrelated coverage.

### Unknown

A registered chain's `Ensure` resolves every planned axis and tuple before crediting, then panics
if any handle is unseeded. A chain outside analyzed packages validates and enforces but credits
nothing.

### Allocation

The warmed recording and enforcement paths allocate nothing. `Product` is a small value carrying
only recorder/shape references, namespace, counters, and packed link observations; it never escapes.

### Persistence

Registration serializes an axis as `namespace NUL ordinal NUL message`; tuple keys retain their
flat form. Handles emit those exact strings, and fuzz merge resolves axes through registration's
exact-key map rather than decoding and reconstructing their identity.

# Dot Product Registration

Registration recognizes a product as one call nest ending in `Ensure`, walks backward to its root,
and publishes the identities, coverage handles, tuple positions, and carve masks runtime consumes.

### Walk

The walk assigns each `Sometimes` its tuple position, seeds both branch obligations, then the full
`2^axes` grid minus every globbed carve. Runtime retains outcomes by fluent ordinal and
`Ensure` projects them through this plan; tuple entries stay keyed `namespace:tuple=(...)`.

### Template

A root using its function's trailing `Namespace` parameter is expanded at each literal
`_Invariants` callsite. A `Product`-returning function resolves the same way as a reusable prefix.

### Ensured

A chain root not terminated by `Ensure` is fatal. Product-returning constructor definitions are the
only exemption; split local-variable chains are unsupported rather than silently unregistered.

### Depth

An invariant bundle owns one self-contained product. Registration rejects a chain bundle calling a
chain bundle, preventing recursive or flattened cross-products.

### Reference

Namespaces, axis messages, constraint messages, and references must be string literals without NUL.
Duplicate namespaces or constraint messages and references to no unique preceding axis are fatal.

### Caps

A chain admits all 255 links representable by its ordinal, including 255 `Sometimes` axes; there is
no smaller grid-policy axis cap. Registration and runtime reject only a 256th link at `Ensure`.

# Bundles

A `_Invariants(v, namespace)` function ensures one product over a type's axes, so the complete
demanded grid travels with the type without returning elements or asking callers to spread a bundle.

### Template

A `_Invariants` is recognized by its `_Invariants` or `_invariants` name and trailing `string` or
`Namespace` parameter; its chain is seeded at each callsite under that callsite's literal namespace.

### Range Template

A `_Invariants` whose body is a single `Range_Invariants(v, MIN, MAX, namespace)` under its own
namespace parameter is a bound-grid template, not a direct callsite. Its grid is seeded from `MIN`
and `MAX` at each `_Invariants` callsite's literal namespace, never under the bare parameter.

### Descent

Registration follows `_Invariants(v, "lit")`, resolves its ensured chain, and seeds the axes and
grid under the callsite's namespace rather than the template parameter.

### Composition

A `_Invariants` calling other `_Invariants` registers each ensured chain independently under its own
namespace. Products are never flattened into a joint cross-product.

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

Calling one `_Invariants` at distinct literal namespaces yields independent demanded grids. Reusing
one namespace for another chain is fatal even when the axes happen to match.

### Gap Location

A bundle axis gap names its callsite namespace, ordinal, and own message. An eager `Always` in the
body remains bare and names only its own message.

### Failure Location

A constraint violation uses the `Impossible` link's descriptive message and matching references;
the stack unwinds through `Ensure`. An eager `Always` still panics from its own frame.

### Static

A `_Invariants` body must be straight-line: a branching or looping statement (`if`, `switch`, `for`,
`select`) fails registration, since it would make the axes it self-emits depend on runtime values.

### Custom Types

A bundle's subject is a custom, defined type. A primitive subject — a builtin, an unnamed slice,
map, or composite — fails registration, except in the framework package that owns the presets.
Cover a primitive inline, or wrap it in a custom type that carries its own bundle.

# Analysis

After the suite, every unexercised axis branch and surviving tuple obligation is reported under its
existing kind, and the run exits non-zero.

### Gaps

A never-fired obligation is named by its chain key and condition; a fully exercised obligation is
left unreported.

### Combination

A grid cell the run never witnessed is reported as a cross-product gap, named by its
tuple of buckets.

### Legend

A cross-product gap prints its grid's axis legend once, each position named by kind,
condition, and message (the axis's own message), and decodes each cell's buckets
back to the events they stand for, so a bare coordinate is debuggable across nested bundles.

### Summary

A clean run reports the count as individual plus combination, of which a panic-able subset.
Individual counts each Always once and each Sometimes twice — true and false are two obligations;
combination is the surviving and carved cells; panic-able is the Always and the carved cells.

### Tally

An eager Always tallies once by literal message. Chain axes, surviving cells, and carved cells tally
per namespace; a carve's glob counts every cell it spans.

### Summary Names Package

When the analyzed package is resolvable from the module root, the summary prefixes
the package path so the line is identifiable when many packages print to the same terminal.

### Clean

With every obligation exercised, the analysis reports nothing and does not exit.

# Coverage

Coverage is never silently dropped: every registered chain link resolves to a pre-seeded entry or
panics. `Sometimes` cannot exist outside a chain.

### Modes

Coverage is recorded in every mode but a benchmark — plain test, fuzz coordinator, and fuzz
worker all credit observations. Under -fuzz each worker appends to a shared file; the
coordinator unions that file before analysis. Enforcement fires in every mode.

### Enforcement

`Ensure` enforces every `Impossible` in every run mode even when coverage recording is disabled.
`Always` enforces independently and eagerly in those same modes.

### Uniqueness

Bare `Always` messages remain globally unique. Chain namespaces are globally unique, while axis
messages may repeat because ordinal is part of their identity.

### Literal

Every registered namespace, link message, and reference must be a compile-time string literal. The
only namespace exception is the trailing template parameter resolved at literal callsites.

### Unresolved

A `_Invariants` or Product-returning constructor used by a registered chain must resolve; an
unresolved chain template is fatal, never skipped.

# Range

`Recorder_Range` is the bounded-integer preset. It collapses a bounded newtype's mandated bound
guards and its `0/1/2/-1` boundary claims into one call, keyed by the callsite namespace.

### Guard

`Range` enforces `value ∈ [min, max]` as two eager bound guards, each a reachability obligation
keyed by the callsite namespace so it never collides across callers. A violation panics in every
mode; reaching a guard credits it under a test run.

### Coverage

`Range` witnesses both interval edges — `Sometimes(v == min)` and `Sometimes(v == max)` — plus each
of `{0, 1, 2, -1}` strictly inside, all mutually exclusive. A single-value interval seeds no grid,
the guards alone carrying reachability; `-1` is dropped for an unsigned value.

### Saturation

When the interval's width is below its witnessed-axis count every value is a witnessed axis, so the
all-false cell (a value matching none) can never occur and is carved — else the grid would demand an
unreachable value forever; over `[0, 1]` every value is `min` or `max`, so "neither" is impossible.

### Exclusions

A callsite declares in-range values unreachable — `Range_Invariants(v, MIN, MAX, ns, holes…)`. Each
hole is enforced (reaching it panics) and drops its sentinel axis; the all-false cell is carved once
every reachable value is a witnessed axis — Saturation is that carve with no holes.

### Enum

`Enum_Invariants(v, ns, members…)` is the sugar for a discrete set: the span is `[min…max]` of the
members, every in-span non-member is an excluded hole, and reaching a non-member panics. A member
strictly inside the span fills the all-false cell; a two-member set carves it.

### Registration

The scan specialises a `Range_Invariants` callsite from its `MIN`/`MAX` arguments, evaluated as
integer constants — literals, sibling-const references, and constant arithmetic or shifts (no
`iota`). A non-literal namespace or an unevaluable bound is fatal.
