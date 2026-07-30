
# Always

`Recorder_Always` is an eager, bare guard whose condition must hold on every call. It is
message-keyed reachability.

### Violation

A false `Always` panics at its own callsite in every run mode, naming itself by its message.

### Eager

`Always` enforces and credits when called; it is never deferred.

### Reachability

An `Always` the suite never reaches is a coverage gap. Registration discovers the bare call and
seeds its literal message.

# Sometimes

`Recorder_Sometimes(recorder, identifier, condition, message)` is the bare observation: it claims
its condition is witnessed both true and false across the suite, never that it holds now.

### Silence

No polarity of a `Sometimes` panics, on any recorder in any run mode.

### Recording

Under a test run that is not a benchmark, a `Sometimes` credits the observed branch of its seeded
entry — true into `Frequency`, false into `False_Frequency`. The key is always
`identifier + NUL + message`; an unseeded key records nothing.

# Range

`Recorder_Range(recorder, identifier, value, minimum, maximum, excluded...)` is the bare bounds
guard, generic over every defined integer width except `uintptr`.

### Bounds

A value below the minimum, above the maximum, or equal to an exclusion panics eagerly, naming the
identifier, the value, and the violated bound; a value inside the interval and beside every hole
passes.

### Exclusions

An exclusion is a hole strictly inside `(minimum, maximum)`. A hole at or outside a bound is a
malformed guard: registration refuses it and the assert build panics in every mode.

### Boundaries

A root seeds two witnesses — the value at the minimum and at the maximum — and a test run credits
both polarities of each on every passing call.

# Enum

`Recorder_Enum(recorder, identifier, value, members...)` is the bare membership guard, generic
over the same integer widths as `Range`.

### Membership

A value equal to no member panics eagerly, naming the identifier, the value, and the member set;
a member passes.

### Distinct

Fewer than two distinct members is a malformed guard: registration refuses it and the assert
build panics in every mode. Repetition of a legal member is allowed.

### Members

A root seeds one witness per distinct member, and a test run credits each member's equality
branch on every passing call.

# Bundles

A `_Invariants` function bundles a type's assertions; its leading `identifier string` parameter
names the boundary demanding them. Composition is calling other `_Invariants`.

### Static

A `_Invariants` body must be straight-line: a branching or looping statement (`if`, `switch`,
`for`, `select`) fails registration.

### Parameter

A bundle's first parameter is exactly `identifier string`; any other shape fails registration.

### Relay

Inside a bundle, every primitive and nested bundle call receives the bundle's own identifier
parameter verbatim. A literal or any derived expression there fails registration.

### Primitive

A bundle whose subject is a primitive type fails registration: a primitive carries no
semantics of its own to bundle, so code states a primitive's assertions inline at a root or
wraps it in a custom type.

### Composition

A composite bundle calls its parts' bundles; every reached assertion seeds under the root
identifier alone, so the call graph never enters a key. A message duplicated across the
composition is a fatal collision.

### Repetition

A composition that reaches the same part twice under one root duplicates every key the
part seeds and fails registration, however many transparent bundles relay the calls. The
refusal is permanent — duplicated streams never merge into one entry. The remedy is
distinctness: separate the duplicates into distinct types whose own `_Invariants` state
distinctly-texted properties, or root each stream at its call site under its own
identifier.

### Cycles

A bundle composition that recurses into itself fails registration.

# Registration

Before the suite runs, a source scan of the analyzed packages finds every root — a primitive or
bundle call whose identifier argument is a bare string literal — and seeds the expected-coverage
space. A non-literal identifier is legal only as a bundle's forwarded parameter.

### Roots

A literal-identifier `Sometimes`, `Range`, or `Enum` seeds its witnesses under the identifier;
the sugar form and the `Recorder_*` form count alike.

### Descent

A literal-identifier bundle call seeds every assertion the composition reaches — transitively —
under the root identifier. Each root earns its own entries.

### Unresolved

A bundle recognised by name but not resolvable to a declaration fails registration — the
analyzer descends a bundle or refuses it.

### Atomicity

A failed registration seeds nothing: diagnostics are collected, every banner prints, the process
exits 1, and Events stays empty.

### Bounds

Range bounds and exclusions and Enum members must resolve statically: int literals, parens,
unary minus, `+ - * / <<`, and same-package single-name constants. A shift whose result does
not fit the resolver's signed word is unresolvable. Anything else fails registration.

# Coverage

### Uniqueness

An identifier is one layer of namespacing and globally unique: two roots sharing one literal, or
two assertions sharing one key, fail registration.

### Literal

An identifier or message that is not a bare string literal fails registration; so does an empty
identifier. Constants are runtime values to the static side.

### Separator

The NUL key separator is refused inside every identifier and message.

### Modes

A benchmark records nothing; outside a test run nothing records. Enforcement fires in every mode
of the assert build.

### Persistence

The first observation of each key and branch fires `Coverage_Sink`; a persisted key — NUL and
all — round-trips the fuzz merge back into its seeded entry.

# Analysis

### Gaps

After the suite, every seeded branch never observed is reported — the key rendered with `·` —
and the run exits 1.

### Summary

A clean run prints one line tallying the tested properties; an `Always` counts one panic-able
individual, a `Sometimes` two.

### Clean

A run whose every seeded branch was observed reports nothing.
