
# Always

`Recorder_Always` is an eager guard outside an `Assertions` builder. Its literal message is its
global coverage identity.

### Violation

In each enforcement mode, a false `Always` causes a panic at its callsite. The panic contains the
message and the observed false value. The benchmark-only noop build does not enforce the guard.

### Eager

`Always` enforces and records the condition when the call occurs. Builder deferral does not change
this behavior.

### Reachability

Registration records each literal message one time. An `Always` call that the suite does not get to
is a coverage gap.

### Constant

Registration rejects a true literal, its parenthesized form, a true Boolean constant alias, or the
negation of a false Boolean constant. The rule applies to `Always` and `Recorder_Always`.
A constant condition cannot enforce a property. Registration publishes no event after this error.

### Uniqueness

An `Always` message identifies one eager source root in the full registration set. Registration
fails if two roots have the same message. The recorder publishes no event for these roots.

# Sometimes

`Sometimes` is only an `Assertion_Builder` link. Its true and false branches are separate
obligations.

### Deferred

A link in record mode stores its condition and advances the value builder. Without a plan, the link
does not advance or record. Only `Ensure` can cause a panic, resolve a handle, or record coverage.
Production and benchmark-only noop builds make `Sometimes` and `Ensure` inert.

### Coverage

`Ensure` records exactly one registered branch. Namespace, ordinal, and literal message identify
that branch. Different ordinals keep duplicate messages separate.

### Gap

If an axis records only one Boolean value, the other branch is one coverage gap.

# Assertions

`Recorder_Assertions(recorder, namespace)` returns an `Assertion_Builder`.
`Assertions(namespace)` uses the default recorder. Value links return advanced copies. `Ensure`
ends the chain.

### API

Links include `Sometimes`, each concrete Range method, and each concrete Enum method. The API does
not have a bare `Sometimes`, cross product, `Impossible`, polarity reference, variadic preset, or
compatibility API.

### Identity

One literal namespace identifies one chain. Namespace, expanded ordinal, and message identify each
axis. Only registration makes this identity and its resolved coverage handle.

### Atomic

`Ensure` examines each deferred failure and selected handle before it records coverage. A failed
chain records nothing. A value panic contains the incorrect value. A full build also contains the
namespace. A production build contains only the fixed property identity.

### Foreign

A chain outside the analyzed packages enforces its presets at `Ensure` and records no coverage.
Ordinary binaries do not read registration plans or shape caches.

### Allocation

A warm record path and a warm enforcement path allocate nothing. Fixed observations exist only with
a plan. Ordinary full enforcement keeps a deferred result without record state. Production and noop
use zero builders. Production formats `Assertion_Failure` only for a panic.

### Persistence

An axis key has `namespace NUL ordinal NUL message`. `Ensure` writes the exact cached key from the
plan. A fuzz merge resolves this key directly. It does not calculate identity from runtime messages.
A fuzz worker writes only its first branch change. The coordinator combines these records.

### Record Validation

A stored record has one Base64 key, one tab, one branch marker, and a final newline. `T` means true.
`F` means false. The merge rejects invalid Base64, absent separators, partial records, unknown keys,
and all other markers. A rejected record changes no branch.

# Assertions Registration

Registration finds one fluent call nest that ends in `Ensure`. It walks to the `Assertions` root.
Then, it publishes an immutable ordered plan of resolved handles.

### Packages

After glob expansion, `Packages_To_Analyze` registers each recognized assertion in the selected
non-test source. Runtime reachability does not control direct registration. Registration makes a
helper chain instance only when it finds a callsite for that helper.

### Test Source

Registration rejects assertion writers and invariant helper calls in the selected test source.
It also rejects references that can put these functions in a function value.
Tests must use a registered production entry point. A violation publishes no event.

### Transitive

Registration follows each helper call in registered source or a called bundle. It follows the call
through the current module without a runtime condition. It parses a necessary package on demand.
Unrelated assertions stay unregistered. An unresolved called bundle causes a fatal error.

### Walk

The walk expands links in fluent order. It records each guard one time and each axis two times. It
does not make tuples, masks, carves, constraints, combinations, or legends.

### Template

If a root uses its final `Namespace` parameter, registration expands it at each literal helper
callsite. A function that returns an incomplete builder is not a template.

### Ensured

Each found root must be one nonempty call expression that ends in `Ensure`. A split chain, a bare
root, or a returned incomplete chain causes a fatal error.

### Literal

A namespace and an axis message must be a compile-time string literal without NUL. A final template
namespace parameter is the only alternative. Registration resolves that parameter at each literal
callsite.

### Namespace

One namespace names exactly one registered source root. A second discovery of the same root does
not change registration. A different root with that namespace causes a fatal error, even if its
links are the same.

### Caps

A chain can have 70 expanded links. Preset guards and generated axes use the same capacity.
Registration rejects link 71 before the suite starts. Ordinary runtime enforcement does not count
the statically expanded chain again.

# Bundles

A `_Invariants(value, namespace)` function owns the assertion chain for one type. It also calls the
corresponding helper for each field.

### Static

A bundle body has straight-line code. A branch or a loop makes its assertion set conditional. Thus,
registration rejects the bundle.

### Template

The `_Invariants` or `_invariants` name identifies a bundle. Its final parameter is `Namespace`.
Registration makes a chain instance for each literal callsite namespace.

### Descent

Registration follows each called bundle through the module. Direct package selection does not stop
this action. Registration records each called chain below the callsite namespace. Static-body and
cycle validation apply to the full call graph.

### Composition

A bundle can call other bundles. Each chain stays independent. Registration does not make a cross
product from their observations.

### Casing

An exported type helper uses `_Invariants`. An unexported type helper uses `_invariants`.

### Sugar

The configured sugar package can call assertion functions without a package qualifier. In all other
packages, registration does not identify an unqualified call as an assertion function.

### Cross Package

For a called helper, bundle resolution parses an unregistered package on demand. It also follows
the calls from that helper. It does not register unrelated assertions. A helper outside the current
module causes a fatal error. An unresolved helper also causes a fatal error.

### Callsite

A call to one bundle with different literal namespaces makes independent obligations. A different
chain cannot use one of these namespaces. Registration rejects that use.

### Gap Location

A bundle gap has separate report fields for its namespace, ordinal, property, absent polarity, and
source. An eager `Always` gap has only its assertion identity and source.

### Custom Types

Bundles belong to custom defined types. Use primitive presets directly or through the framework
primitive helpers. Do not put a primitive preset in a primitive helper bundle.

### Signed Primitive Mandates

`Int_Invariants` evidence must include `1`, `-1`, `math.MinInt64`, and `math.MaxInt64`.
Each fixed-width helper must include `1`, `-1`, and its `math.MinIntN` and `math.MaxIntN`.
Each axis must have the two branches. Other signed values are legal. A narrow platform cannot build.

### Unsigned Primitive Mandates

Evidence for `Uint_Invariants` must include `0`, `1`, and `math.MaxUint64`. Evidence for each
fixed-width unsigned helper must include `0`, `1`, and its corresponding `math.MaxUintN`. Each axis
must have the two branches. Other unsigned values are legal.

### Floating Primitive Mandates

Evidence for `Float32_Invariants` and `Float64_Invariants` must include NaN, negative infinity, and
positive infinity. Each axis must have the two branches. Other finite values are legal.

### Boolean Primitive Mandate

Evidence for `Boolean_Invariants` must include true and false through the two branches of its
true-value axis.

### Primitive Isolation

Registration expands the real primitive helper source at each literal callsite namespace. The same
helper at a different namespace has separate links. Coverage at one namespace does not change raw
metadata, table data, JSON data, or fuzz data for a different namespace.

# Analysis

After the suite, analysis reports each obligation that has no evidence. If there is a gap, analysis
exits with a nonzero status.

### Gaps

A guard with no call is in the reachability table. An absent axis polarity is in the branch table.
Each branch row has a namespace, numeric link, polarity, property, and unquoted source expression.

### Reachability Identity

A builder reachability row uses its public namespace. It does not expose the internal link key.

### Table Order

Each section has a count. Branch rows use assertion, numeric link, and polarity as the sort keys.
Reachability rows use assertion as the sort key. The gap banner occurs before and after the report.

### Table Escape

A table cell escapes pipes and backslashes. It writes a physical line break as `<br>`. These changes
keep the Markdown structure and do not change the registered value.

### Output Configuration

The default package accepts `INVARIANT_OUTPUT=table` or `INVARIANT_OUTPUT=json`.
An unset value means `table`. All other values cause a configuration diagnostic and a nonzero exit.
The suite does not start.

### Summary

A clean run reports each expanded property and the subset that can cause a panic. Each `Always` and
preset guard counts one time. Each axis counts two times. Each Range hole counts one time in the
panic subset. A helper does not combine its links. The builder has no combination total.

### Clean

When each obligation has evidence, analysis writes no gap report and does not call `Exit`.

# Coverage

Each registered builder link resolves through its pre-seeded plan. If resolution fails, `Ensure`
causes a panic before it records coverage. `Sometimes` cannot be outside a builder.

### Modes

Untagged tests and fuzz processes record coverage. Benchmarks and binaries only enforce assertions.
The production tags are `invariant_disable_coverage`, `prd`, `prod`, and `production`. You can
combine them. `invariant_noop` selects inert benchmarks. A mix with production fails compilation.

### Enforcement

A full build defers Range and Enum enforcement to `Ensure`. Production enforces bounds, holes, and
membership at each link. A production panic has the assertion prefix and fixed identity. It does
not have the namespace. `Always` stays eager. `Sometimes` and `Ensure` are inert in production.

### Uniqueness

A bare `Always` message is globally unique. A builder namespace is globally unique. An axis message
can occur again because the expanded ordinal is part of its identity.

### Literal

Each registered namespace and message is a compile-time string literal. A dynamic foreign chain can
enforce presets. It cannot make a coverage identity.

# Range

`Assertion_Builder.Range_TYPE(value, minimum, maximum)` exists for each signed and unsigned integer
width other than `uintptr`. Convert a defined integer explicitly at the call. A contiguous Range
accepts exactly these three arguments.

### Holed

`Range_Holed_TYPE` adds holes to the contiguous Range. A signed method takes four hole slots. An
unsigned method takes three. Put distinct holes in ascending order. Fill unused slots with the final
hole. Only this fill operation can duplicate a hole.

### Guard

The two Range families add lower and upper reachability guards. Registration rejects an incorrect
argument count, domain, hole order, fill value, or hole position. A full build defers a failure to
`Ensure`. Production causes a panic at the incorrect Range link.

### Coverage

A Range with multiple values has axes for its bounds. It also has axes for eligible interior values
`0`, `1`, `2`, and `-1`. Each axis has two branches. Evidence from a different namespace or ordinal
does not satisfy the axis.

### Cardinality

A registered Range domain must have at least five legal values after hole removal.
For one legal value, use direct `Always` equality. For two to four, use the corresponding Enum.
After validation, registration reports the count and replacement. Runtime ignores this rule.

### Exclusions

Each distinct hole must be strictly inside the bounds. A hole cannot equal a bound. A hole removes
the equal sentinel axis. Duplicate final-hole fill removes and counts the hole one time.
Registration rejects an incorrect hole before it records an entry.

### Registration

Registration resolves constants and arithmetic. It expands two guards before the separate axes. It
rejects an unresolved domain, incorrect slot sequence, small domain, incorrect argument count, or
expansion above 70 links.

# Enum

`Assertion_Builder.Enum_TYPE(value, first, second)`, `Enum_3_TYPE`, and `Enum_4_TYPE` exist for the
same integer types as Range. Their names specify two, three, and four members. The API has no
variadic compatibility method.

### Guard

Enum adds one successful membership guard. Registration rejects an incorrect argument count or
domain. A nonmember causes an assertion failure. A full build defers this failure to `Ensure`. A
production build causes a panic at the incorrect Enum link.

### Members

Registration must resolve each member statically. Members must be distinct and in ascending order.
Each member is a separate Boolean axis in that order. A full evidence set calls the helper with each
member and supplies the two branches for each member axis.

### Registration

Registration resolves the member constants. It records the membership guard and member axes. It
rejects an unresolved member, duplicate member, incorrect order, incorrect argument count, or
expansion above 70 links.
