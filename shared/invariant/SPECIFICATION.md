
# Always

`Recorder_Always` is a bare eager guard. Its literal message is its global coverage identity, and
it is independent of an `Assertions` builder.

### Violation

A false `Always` panics at its own callsite in every enforcing build and names its message. The
panic includes the observed false value. The benchmark-only noop build is non-enforcing.

### Eager

`Always` enforces and credits when called; builder deferral does not change it.

### Reachability

Registration seeds each literal message once, so an `Always` the suite never reaches is a gap.

# Sometimes

`Sometimes` exists only as an `Assertion_Builder` link and demands both branches independently.

### Deferred

A recording link captures its condition and advances the value builder. Without a plan it does not
advance or observe; only `Ensure` may panic, resolve, or credit. Production and benchmark-only noop
builds compile `Sometimes` and `Ensure` as inert links.

### Coverage

`Ensure` credits exactly one registered branch under namespace, ordinal, and literal message.
Repeated messages remain distinct because their ordinals differ.

### Gap

An axis observed only true or only false reports the other branch as an individual coverage gap.

# Assertions

`Recorder_Assertions(recorder, namespace)` returns an `Assertion_Builder`; `Assertions(namespace)`
uses the default recorder. Value links return advanced copies and `Ensure` terminates the chain.

### API

Links are `Sometimes`, every concrete `Range_TYPE` and `Range_Holed_TYPE` method, and the
`Enum_TYPE`, `Enum_3_TYPE`, and `Enum_4_TYPE` families. There is no bare `Sometimes`, cross
product, `Impossible`, polarity reference, variadic preset, or compatibility surface.

### Identity

One literal namespace identifies one chain. Each axis is keyed by namespace, expanded ordinal,
and message; registration alone constructs that identity and its resolved coverage handle.

### Atomic

`Ensure` validates every deferred failure and selected handle before crediting; a failed chain
credits nothing. Value panics include the offending value; full builds separate namespace and
property with ` · `, while production retains only the fixed property.

### Foreign

A chain outside analyzed packages enforces its presets at `Ensure` and credits nothing. Ordinary
binaries do not consult registration plans or shape caches.

### Allocation

Warmed recording and enforcement allocate nothing; fixed observations exist only with a plan.
Ordinary full enforcement carries a deferred verdict without constructing recording state.
Production and noop use zero builders; production formats `Assertion_Failure` only when rendered.

### Persistence

Axes serialize as `namespace NUL ordinal NUL message`; `Ensure` emits the plan's exact cached key.
Fuzz merge resolves that key directly and never reconstructs identity from runtime messages. Fuzz
workers persist only their first branch transition; the coordinator unions those records.

# Assertions Registration

Registration recognizes one fluent call nest ending in `Ensure`, walks to its `Assertions` root,
and publishes an immutable ordered plan of pre-resolved individual handles.

### Packages

`Packages_To_Analyze`, after glob expansion, selects the packages whose non-test Go source is
registered directly. Every recognized direct `Always` call and ensured `Assertions` chain in that
source is registered regardless of whether its enclosing function is reachable at runtime. An
ensured chain inside an `_Invariants` or `_invariants` declaration remains a template and is
instantiated only through a reached bundle callsite.

### Transitive

Every `_Invariants` and `_invariants` call found in registered source or a reached bundle is
registered transitively and unconditionally within the current module. This applies to bare and
qualified calls even when the declaration package is not selected by `Packages_To_Analyze`.
Registration lazily parses only the packages needed to resolve those reached bundles; unrelated
direct assertions in an unregistered package are not registered. An unresolved reached bundle is
fatal rather than silently skipped.

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

One namespace names exactly one registered source root. Rediscovering that root through bundle
composition is idempotent; a distinct root using the namespace is fatal even with identical links.

### Caps

A chain accepts exactly 70 expanded links. Preset guards and generated axes occupy that same space,
and registration rejects a 71st before the suite runs. Ordinary runtime enforcement never recounts
the statically expanded chain.

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

Registration follows every reached bundle call across the current module, whether or not the
declaration package was selected for direct registration, and seeds each reached chain under the
callsite namespace rather than the template parameter. Static-body and cycle validation still
apply to the complete reached graph.

### Composition

A bundle may call other bundles. Each ensured chain remains independent and no cross-product is
formed between their observations.

### Casing

Exported and unexported type helpers use `_Invariants` and `_invariants` respectively.

### Sugar

The configured sugar package may call assertion writers unqualified. Elsewhere an unqualified call
is not treated as an invariant writer.

### Cross Package

Bundle resolution uses the current module path. Reaching a helper in an unregistered package
lazily parses that package for the helper and its transitive bundle calls without registering the
package's unrelated direct assertions. A recognized helper outside the current module or one that
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

A clean run reports every expanded property and its panic-able subset. Each Always and preset guard
counts once, each axis twice, and each Range hole once as panic-able. A helper never collapses its
links into one property; the builder has no combination total.

### Clean

With every individual obligation exercised, analysis reports nothing and does not exit.

# Coverage

Every registered builder link resolves through its pre-seeded plan or `Ensure` panics before any
credit. `Sometimes` cannot exist outside a builder.

### Modes

Untagged tests and fuzz processes record; benchmarks and binaries only enforce. Production is any
combination of `invariant_disable_coverage`, `prd`, `prod`, and `production`. Only `invariant_noop`
selects inert benchmarks; mixing it with production fails compilation. Legacy tags remain ordinary.

### Enforcement

The full build defers Range and Enum to `Ensure`. Production checks only bounds, holes, and
membership at each link and panics with the assertion prefix and fixed identity, never namespace.
Always stays eager; Sometimes and Ensure are inert. Noop use is undefined; APIs are shared.

### Uniqueness

Bare `Always` messages are globally unique. Builder namespaces are globally unique, while axis
messages may repeat because expanded ordinal is part of their identity.

### Literal

Every registered namespace and message is a compile-time string literal. Dynamic foreign chains
may enforce presets but never create coverage identities.

# Range

`Assertion_Builder.Range_TYPE(value, minimum, maximum)` is available for every signed and unsigned
primitive width except `uintptr`; defined integers convert explicitly at the call. A contiguous
Range accepts exactly those three arguments.

### Holed

`Range_Holed_TYPE` separates excluded domains from the ordinary contiguous path. A signed method
takes four hole slots and an unsigned method takes three. Distinct holes are strictly ascending;
unused slots repeat the final hole, and only that final-hole padding may duplicate a value.

### Guard

Both families contribute lower and upper reachability guards. Wrong arities, malformed domains, and
invalid hole order, padding, or placement fail registration. Full builds defer observed failures
until `Ensure`; production panics at the violating Range link.

### Coverage

A non-singleton interval witnesses minimum and maximum plus eligible strictly-interior `0`, `1`,
`2`, and `-1`, each as an independent true/false axis. A singleton has only its two guards.

### Exclusions

Distinct holes must be strictly inside the boundaries, may never exclude a boundary, and remove an
equal sentinel axis. Repeated final-hole padding is idempotent: it removes and counts that hole only
once. Invalid and out-of-range holes fail before registration seeds any entry.

### Registration

Registration resolves constants and arithmetic, expands two guards followed by the distinct axes,
and rejects any unresolvable domain, noncanonical slot sequence, wrong arity, or expansion beyond
70 links.

# Enum

`Assertion_Builder.Enum_TYPE(value, first, second)`, `Enum_3_TYPE`, and `Enum_4_TYPE` are available
for the same primitive integer types. Their names declare exactly two, three, and four members; no
variadic compatibility surface remains.

### Guard

Enum contributes one successful membership reachability guard. A wrong arity, invalid domain, or
non-member is a registration failure. The full build defers an observed non-member until `Ensure`;
a production build panics at the violating Enum link.

### Members

Members are statically resolvable, exactly distinct, and strictly ascending. Every member is an
independent true/false axis in canonical ascending order.

### Registration

Registration resolves the member constants, seeds the membership guard and canonical member axes,
and rejects any unresolved, duplicate, nonascending, wrong-arity, or beyond-70-link domain.
