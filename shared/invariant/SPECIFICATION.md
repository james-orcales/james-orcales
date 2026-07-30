
# Always

`Recorder_Always` is a bare eager guard. Its literal message is its global coverage identity, and
it is independent of an `Assertions` builder.

### Violation

A false `Always` panics at its own callsite in every enforcing run mode and names its message.

### Eager

`Always` enforces and credits when called; builder deferral does not change it.

### Reachability

Registration seeds each literal message once, so an `Always` the suite never reaches is a gap.

# Sometimes

`Sometimes` exists only as an `Assertion_Builder` link and demands both branches independently.

### Deferred

A link captures its condition and advances the value builder. It never panics, resolves coverage,
or credits an event; `Ensure` owns every visible operation.

### Coverage

`Ensure` credits exactly one registered branch under namespace, ordinal, and literal message.
Repeated messages remain distinct because their ordinals differ.

### Gap

An axis observed only true or only false reports the other branch as an individual coverage gap.

# Assertions

`Recorder_Assertions(recorder, namespace)` returns an `Assertion_Builder`; `Assertions(namespace)`
uses the default recorder. Value links return advanced copies and `Ensure` terminates the chain.

### API

Links are `Sometimes` and every concrete `Range_TYPE` and `Enum_TYPE` method. There is no bare
`Sometimes`, cross product, `Impossible`, polarity reference, or compatibility surface.

### Identity

One literal namespace identifies one chain. Each axis is keyed by namespace, expanded ordinal,
and message; registration alone constructs that identity and its resolved coverage handle.

### Atomic

`Ensure` validates every deferred failure and selected handle before crediting anything. A failed
chain panics without partial coverage mutation.

### Foreign

A chain outside analyzed packages enforces its presets at `Ensure` and credits nothing. Ordinary
binaries do not consult registration plans or shape caches.

### Allocation

Warmed recording and enforcement allocate nothing. The builder is a value carrying fixed
observations, counters, deferred verdicts, and an optional registration-owned plan.

### Persistence

Axes serialize as `namespace NUL ordinal NUL message`; `Ensure` emits the plan's exact cached key.
Fuzz merge resolves that key directly and never reconstructs identity from runtime messages.

# Assertions Registration

Registration recognizes one fluent call nest ending in `Ensure`, walks to its `Assertions` root,
and publishes an immutable ordered plan of pre-resolved individual handles.

### Walk

The walk expands links in fluent order and seeds every guard once and every axis twice. It creates
no tuples, masks, carves, constraints, combinations, or legends.

### Template

A root using its function's trailing `Namespace` parameter is expanded at every literal
`_Invariants` callsite. Functions returning half-built builders are never templates.

### Ensured

Every discovered root must be a nonempty, unsplit call expression terminated by `Ensure`; a bare
root or a returned half-chain is fatal.

### Literal

Namespaces and axis messages must be compile-time string literals without NUL, except for the
trailing template namespace parameter resolved at literal callsites.

### Namespace

One namespace names exactly one registered chain. Any second root using it is fatal even when the
two chains have the same links.

### Caps

A chain accepts exactly 70 expanded links. Preset guards and generated axes occupy that same space;
registration rejects a 71st, while foreign runtime execution defers the equivalent panic to Ensure.

# Bundles

A `_Invariants(value, namespace)` function owns a type's ensured assertion chain and composes the
corresponding helpers of its fields.

### Static

A bundle body is straight-line; branching and looping make its emitted assertion set conditional
and therefore fail registration.

### Template

A bundle is recognized by its `_Invariants` or `_invariants` name and trailing `Namespace`
parameter, and its chain is instantiated under each literal callsite namespace.

### Descent

Registration follows bundle calls across the current module and seeds each reached chain under the
callsite namespace rather than the template parameter.

### Composition

A bundle may call other bundles. Each ensured chain remains independent and no cross-product is
formed between their observations.

### Casing

Exported and unexported type helpers use `_Invariants` and `_invariants` respectively.

### Sugar

The configured sugar package may call assertion writers unqualified. Elsewhere an unqualified call
is not treated as an invariant writer.

### Cross Package

Bundle resolution uses the current module path. A recognized helper outside that module or one that
cannot be resolved is fatal rather than silently skipped.

### Callsite

Calling one bundle under distinct literal namespaces produces independent individual obligations;
reusing a namespace for another chain is fatal.

### Gap Location

A bundle axis gap names its callsite namespace, expanded ordinal, message, and missed polarity.
An eager `Always` remains keyed only by its own message.

### Custom Types

Bundles belong to custom defined types. Primitive presets are used inline or by the framework's own
primitive helpers rather than wrapped in primitive `_Invariants` bundles.

# Analysis

After the suite, every unexercised individual obligation is reported and the run exits nonzero.

### Gaps

Never-fired Always and preset guards report reachability gaps; each unobserved Sometimes polarity
reports its namespaced axis key and condition.

### Summary

A clean run reports one total: Always and successful preset guards count once, while every axis
counts twice. There are no combination or panic-able subtotals and no legend.

### Clean

With every individual obligation exercised, analysis reports nothing and does not exit.

# Coverage

Every registered builder link resolves through its pre-seeded plan or `Ensure` panics before any
credit. `Sometimes` cannot exist outside a builder.

### Modes

Plain tests, fuzz coordinators, and fuzz workers record; benchmarks and ordinary binaries do not.
Fuzz workers persist only a branch's first transition and the coordinator unions those records.

### Enforcement

`Ensure` enforces Range and Enum in every enforcing mode, including modes that do not record.
Bare `Always` remains independent and eager; `noassert` makes the complete surface inert.

### Uniqueness

Bare `Always` messages are globally unique. Builder namespaces are globally unique, while axis
messages may repeat because expanded ordinal is part of their identity.

### Literal

Every registered namespace and message is a compile-time string literal. Dynamic foreign chains
may enforce presets but never create coverage identities.

# Range

`Assertion_Builder.Range_TYPE(value, minimum, maximum, holes...)` is available for every signed and
unsigned primitive width except `uintptr`; defined integers convert explicitly at the call.

### Guard

Range contributes successful lower-bound and upper-bound reachability guards. Invalid domains,
bound violations, and observed holes are deferred until `Ensure` in every enforcing mode.

### Coverage

A non-singleton interval witnesses minimum and maximum plus eligible strictly-interior `0`, `1`,
`2`, and `-1`, each as an independent true/false axis. A singleton has only its two guards.

### Exclusions

Holes must be strictly inside the boundaries, may never exclude a boundary, and remove an equal
sentinel axis. Invalid and out-of-range holes fail before registration seeds any entry.

### Registration

Registration resolves constants and arithmetic, expands two guards followed by the distinct axes,
and rejects any unresolvable domain or expansion beyond 70 links.

# Enum

`Assertion_Builder.Enum_TYPE(value, members...)` is available for the same primitive integer types
and requires at least two distinct members.

### Guard

Enum contributes one successful membership reachability guard. An invalid domain or non-member is
deferred until `Ensure` in every enforcing mode.

### Members

Every distinct member is an independent true/false axis in argument order. Repeated members do not
change the coverage set or expanded ordinal sequence.

### Registration

Registration resolves and normalizes the member constants, seeds the membership guard and member
axes, and rejects any unresolvable domain or expansion beyond 70 links.
