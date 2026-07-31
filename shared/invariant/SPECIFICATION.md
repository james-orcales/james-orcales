
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

An `Always` message identifies one eager source root in the full registration set. An inline helper
message shares that one global set. Registration fails if two roots have the same message, and the
recorder publishes no event for these roots.

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

# Inline

`Sometimes`, `Range`, `Enum`, and `Range_Holed` are generic free functions for a body that owns no
bundle. They take no namespace, and their `message` argument carries the identity. Go forbids a type
parameter on a method, thus only these free forms are generic.

### Identity

One free message identifies one inline assertion in the full registration set. An inline axis key
has the shape of a chain axis key, with the message in the namespace position and no subject.
Registration rejects a message that `Always` or another inline helper already owns.

### Record

An inline helper enforces its condition, then records like `Always`. Registration resolves its
bounds and members statically, and seeds every expanded axis. An absent branch is a coverage gap.

### Plan

Registration pre-seeds the resolved handles under the bare message. Thus the record path reads one
map with the message alone and never joins the axis key. Without a plan, an inline helper only
enforces, which is what a binary and a benchmark get.

# Assertions

`Recorder_Tree(recorder, subject, namespace)` returns an `Assertion_Builder`.
`Tree(subject, namespace)` uses the default recorder. Value links return advanced copies. `Ensure`
ends the chain.

### API

Links include `Sometimes`, each concrete Range method, and each concrete Enum method. The API does
not have a bare `Sometimes`, cross product, `Impossible`, polarity reference, variadic preset, or
compatibility API.

### Identity

One literal namespace identifies one root. One subject type identifies one chain under that root.
Namespace, subject package, subject type, expanded ordinal, and message identify each axis. Only
registration makes this identity and its resolved coverage handle.

### Subject

`Assertions(subject, namespace)` takes the subject before the namespace, and the subject supplies
the chain type. Registration reads that type from the source. The runtime reads the same package
path and type name through `reflect`. A subject must be a defined package-level type.

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

Every axis key has `namespace NUL package NUL type NUL ordinal NUL message`, and an inline helper
puts its message in the namespace position. `Ensure` writes the exact cached key, thus a fuzz merge
resolves it without calculating identity from runtime messages.

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

Registration follows each helper call through the current module without a runtime condition, and
parses a necessary package on demand. A descent registers every assertion of the body it enters, not
only its chain. An unrelated assertion stays unregistered, and an unresolved bundle is fatal.

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

One namespace has exactly one source owner in the complete registration. Registration rejects the
same namespace at a second source callsite, including a call to the same helper declaration.
A collision publishes no event or plan.

### Source Owner

A nested bundle callsite gets the namespace of its parent. Two parent callsites make two
namespaces and two registrations. One parent callsite does not change the plan of the other.

### Source Path

A forwarded namespace keeps its literal source owner. Each forwarding callsite and assertion root
is part of its chain identity. Two chains with different subject types share one namespace. Two
chains with one subject type cannot.

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

### Namespace Source

A bundle call in a bundle body uses the `Namespace` parameter of that body. A string literal at
that position causes a fatal error. Only a callsite outside a bundle gives a namespace.

### Duplicate Namespace

One namespace literal occurs one time in the registered source. A second bundle callsite that uses
the same literal causes a fatal error.

### Template

The `_Invariants` or `_invariants` name identifies a bundle. Its final parameter is `Namespace`.
Registration makes a chain instance for each literal callsite namespace.

### Descent

Registration follows each called bundle through the module. Direct package selection does not stop
this action. Registration records each called chain at the callsite namespace. Static-body and
cycle validation apply to the full call graph.

### Composition

A bundle can call other bundles. The called chain keeps the callsite namespace. Registration does
not make a cross product from their observations.

### Tree

One subject type occurs one time in the expansion of one root. The type stands in for the call path,
thus a second occurrence gives two paths one identity. A diamond, a repeated sibling, and a cycle
are each a fatal error that names both callsites. One type can be a node of many roots.

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

### Boolean

A bundle for a defined Boolean type states exactly one `Sometimes`. A Boolean has two values and
both are obligations, thus one axis states the whole type. Any other link, a second link, or no
chain at all is a fatal error.

### Custom Types

A bundle belongs to one custom defined type. A primitive field has no bundle of its own, thus it
takes a defined type first. The framework supplies no primitive helper.

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

A bare `Always` message is globally unique. A builder namespace is globally unique as a root. Under
one root, one subject type identifies one chain. An axis message can occur again because the subject
type and the expanded ordinal are part of its identity.

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
