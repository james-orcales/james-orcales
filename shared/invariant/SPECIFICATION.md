
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
entry — true into `Frequency`, false into `False_Frequency`. A non-empty identifier scopes the
key so one property at two boundaries earns two entries; an unseeded key records nothing.

# Range

`Recorder_Range(recorder, identifier, value, minimum, maximum, excluded...)` is the bare bounds
guard, generic over every defined integer width except `uintptr`. Trailing exclusions are holes
inside the interval the value must also avoid.

### Bounds

A value below the minimum, above the maximum, or equal to an exclusion panics eagerly, naming the
identifier, the value, and the violated bound; a value inside the interval and beside every hole
passes.

# Enum

`Recorder_Enum(recorder, identifier, value, members...)` is the bare membership guard, generic
over the same integer widths as `Range`.

### Membership

A value equal to no member panics eagerly, naming the identifier, the value, and the member set;
a member passes.

# Bundles

A `_Invariants` function bundles a type's assertions, so they travel with the type; its trailing
`identifier` names the boundary demanding them and threads into each `Sometimes`. Composition is
calling other `_Invariants`.

### Static

A `_Invariants` body must be straight-line: a branching or looping statement (`if`, `switch`,
`for`, `select`) fails registration.
