
# Invariants

Every type states its properties in a companion function beside it, the shared/invariant bundle
convention made mandatory.

### Presence

An in-scope type declaration is immediately followed by its invariant function; only that
function's own doc comment and blank lines may sit between them.

### Casing

The function takes the type name with an _Invariants suffix when the type is exported, an
_invariants suffix when it is unexported.

### Signature

The function takes the type, by value or pointer, as its first parameter and an
invariant.Namespace as its last.

### Orphan

A function whose name ends in _Invariants or _invariants is itself declared directly below its
matching type, never adrift.

### Scope

Structs with fields, defined non-alias types, and generic types are in scope. Aliases, function
and interface types, empty structs, function-local types, _test.go files, and the packages in
lint.json's opt_out_assertion_mandate_packages are exempt.

### Numeric Bounds

A numeric type's value-parameter bundle guards both ends with the value on the left:
Always(v <= MAX) and Always(v >= MIN).

### Numeric Bound Constant

MAX and MIN are each a package-level constant, named so the bound reads as a deliberate limit —
never an inline literal nor an imported selector.

### Numeric Coverage

The bundle claims 0, 1, 2, and -1 for signed by a Dot_Product chain's Sometimes(v == V) or by
Always(v ==/!= V); a float claims NaN and both infinities instead. Both bound edges are always in
range, so each must be witnessed by chain links, never merely guarded.

### Numeric Range Preset

A terminated `Range_TYPE(value, minimum, maximum, holes...).Ensure()` chain satisfies Numeric Bounds
and Numeric Coverage at once. A defined integer explicitly converts each argument to TYPE;
conversion never relaxes the bound-constant rule or requires a local declaration.

### Numeric Enum Preset

A terminated `invariant.Dot_Product(namespace).Enum_TYPE(v, members…).Ensure()` chain satisfies the
same rules for a discrete domain. Every member, including one under its exact TYPE conversion, must
remain a package-level constant rather than an inline value.

### Count Bounds

A string, slice, or map type's bundle guards its length with the length on the left:
Always(len(v) <= MAX) and Always(len(v) >= MIN).

### Count Bound Constant

MAX and MIN for a length are each a package-level constant, never an inline literal nor an imported
selector.

### Count Coverage

The bundle claims 0, 1, and 2 by a Dot_Product chain's Sometimes(len(v) == V) or by
Always(len(v) ==/!= V), and witnesses both length edges with chain links, never merely guarding
them.

### Field Composition

A struct type's bundle calls the _Invariants of every field whose type has one — a preset for a
primitive, the type's own bundle otherwise. A pointer field composes its pointee. An immediate
sync.Mutex or sync.RWMutex field is exempt.

### Parameter Assertion

A named free function asserts every input parameter whose type has an _Invariants, in the leading
block right after the output defer: a flat call, or a range loop over a slice's elements. A bundle
or a method is exempt.

### Output Assertion

A named free function whose return values include one with an _Invariants asserts each in a defer
that is the first statement of the body.

### Recorder Registration

A non-exempt shared-library package's TestMain body is exactly invariant.Run_Test_Main(m), so its
suite registers with the coverage recorder; a binary component is witnessed via its simulation.
Any other body, or no TestMain, is banned.

### Primitive Types

A raw string, slice, or map may not be a function parameter, result, or struct field; it has no
preset and no bundle of its own. Wrap it in a defined type. A stdlib-interface method, a _test.go
file, and a package in opt_out_assertion_mandate_packages are exempt.

# Simulation

A binary component's invariants are witnessed only by a simulation package under its internal
directory, whose fuzz test drives internal.Main; its coverage is judged tier two.

### Presence

A binary component with a non-exempt internal package declares an internal/simulation_test package;
absent one, there is no blackbox witness for its invariants. A wholly exempt internal tree needs
none.

### Contents

The simulation package declares a Fuzz function that drives internal.Main; that fuzz, in its own
isolated test binary, witnesses the component's invariants. Other declarations are allowed.

### Test Main

The simulation TestMain body is exactly invariant.Run_Test_Main(m, "../**"); any other body, or
a missing TestMain, is banned.

### Coverage

The "../**" glob registers the internal package and every package beneath it, so all are
witnessed in one pattern; a narrower argument that omits some is banned.

### Entry

The simulation references only Main among the internal tree's functions; any other internal
function it names is a second entry point and is banned. Internal's exports are otherwise free.

### Blackbox

The simulation directory holds one blackbox test package and no source package: every file is a
_test.go declaring package <name>_test.
