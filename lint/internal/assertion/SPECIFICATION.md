
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

The function takes `identifier string` as its first parameter and the type, by value or
pointer, as its second — the framework's bundle parameter doctrine.

### Orphan

A function whose name ends in _Invariants or _invariants is itself declared directly below its
matching type, never adrift.

### Scope

Structs with fields, defined non-alias types, and generic types are in scope; their helper bodies
carry the mandate and the pass defines no argument-bundling _Input structs. Aliases, function and
interface types, empty structs, local types, tests, and opted-out packages are exempt.

### Scalar Helper

A defined integer helper states its domain with a bare invariant.Range or invariant.Enum over
the exactly converted value, forwarding its identifier; hand-written Sometimes witnesses may
accompany the guard but individual assertions never substitute for it. A float or boolean
helper states hand-written invariant.Always or identifier-forwarding invariant.Sometimes
assertions — the primitive presets no longer exist.

### Count Helper

A defined string, slice, or map helper states a bare invariant.Range or invariant.Enum over
len(value), forwarding its identifier. Individual assertions, another subject, and a foreign
lookalike never substitute.

### Helper Constants

Each Range boundary and Enum member is a package-level constant in the type's package, optionally
wrapped in its exact primitive conversion. Inline literals, computed call operands, imported
selectors, and conversion to another primitive never satisfy the helper mandate.

### Helper Identity

The guard is a direct helper-body statement and resolves to the actual invariant package
without parameter, local, or import shadowing. Its identifier argument is the helper's own
leading parameter, forwarded verbatim; foreign lookalikes and literal identifiers never
substitute.

### Field Composition

A struct type's helper directly calls the exact package-qualified _Invariants helper of every
field whose type has one, forwarding its identifier first and passing the field second.
Foreign, nested, or shadowed calls never substitute. A pointer field composes its pointee; an
immediate mutex is exempt.

### Parameter Helper

A named free function calls the exact helper for every input parameter whose type has one, in the
leading block right after the output defer. Direct Always or Sometimes assertions and same-named
foreign or shadowed functions never substitute. A helper or method is exempt.

### Output Helper

A named free function whose return values include one with an _Invariants directly calls each exact
helper in a defer that is the first statement of the body. Nested, shadowed, and assertion calls
never substitute.

### Recorder Registration

A non-exempt shared-library package's TestMain body is exactly invariant.Run_Test_Main(m), so its
suite registers with the coverage recorder; a binary component is witnessed via its simulation.
Any other body, or no TestMain, is banned.

### Primitive Types

A primitive type — string, bool, any numeric width, byte, rune, uintptr, a complex, a raw
slice, or a map — may not be a function parameter, result, or struct field: all types are
semantic, and a primitive can carry no bundle of its own. Wrap it in a defined type.
Primitives appear only as the underlying type of a declaration. A stdlib-interface method, a
_test.go file, and a package in opt_out_assertion_mandate_packages are exempt.

### Semantic Declarations

A type declaration sits directly on a primitive: `type Metric int64`, never `type Kept Tally`
or `type Reference pkg.Measurements`. Declaring over another defined type would alias a
contract instead of owning one. Struct, slice, map, array, function, interface, and channel
declarations remain the composition mechanism and are exempt, as are aliases.

### Distinct Fields

No struct declares two fields of one type, in one field entry or across several. Two
same-typed fields under one root would seed identical coverage keys; the fields were never
the same thing — separate them into distinct semantic types with their own _Invariants.

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
