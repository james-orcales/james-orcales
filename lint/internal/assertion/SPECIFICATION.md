
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

Structs with fields, defined non-alias types, and generic types are in scope; their helper bodies
carry the mandate and the pass defines no argument-bundling _Input structs. Aliases, function and
interface types, empty structs, local types, tests, and opted-out packages are exempt.

### Scalar Helper

An integer helper uses direct singleton Always equality against a package constant, or an ensured
exact Range or Enum builder over the converted value. A float uses direct singleton Always equality.
A Boolean ensures a Tree whose one link is a Sometimes, because its two values are two obligations.

### Count Helper

A defined string, slice, or map helper uses direct singleton Always equality against a package
constant or ensures one exact Int Range or Enum family over len(value). Another subject, suffix,
unterminated or split builder, or unrelated Tree builder never substitutes.

### Helper Constants

Each Range boundary, Enum member, and singleton Always member is a package-level constant in the
type's package. An integer can use its exact primitive conversion. Inline literals, computed call
operands, imported selectors, and conversion to another primitive never satisfy the helper mandate.

### Cross Package Identity

A constant belongs only to its declaring package. A same-spelled constant from another package
cannot supply a local Range boundary, Enum member, or singleton member. A helper also belongs to its
declaring package. A foreign helper with the required name cannot satisfy a local typed subject.

### Helper Identity

All canonical calls are direct shared/invariant/default statements with qualifier invariant. A Tree
root takes its helper's own subject before the namespace, and Always uses `subject == CONSTANT` with
a literal message. Shadowing, an alias, a reversed equality, another root, and a split builder fail.

### Builder Walk

An ensured Tree builder expands to at most 70 fluent links. A longer builder is too costly for
static analysis and is banned.

### Field Composition

A struct type's helper directly calls the exact package-qualified _Invariants helper of every field.
Every field has a defined type, thus every field has one. Foreign, nested, or shadowed calls never
substitute. A pointer field composes its pointee, and an immediate mutex is exempt.

### Parameter Helper

A named free function calls the exact helper for every input parameter whose type has one, in the
leading block right after the output defer. Direct Always or Sometimes assertions and same-named
foreign or shadowed functions never substitute. A helper or method is exempt.

### Output Helper

A named free function whose return values include one with an _Invariants directly calls each exact
helper in a defer that is the first statement of the body. Nested, shadowed, and assertion calls
never substitute.

### Subject Isolation

Each field, input, and output is a separate helper requirement. One helper call satisfies only the
exact subject in its first argument. A call for one subject cannot satisfy another subject, even
when both subjects have the same type and require the same helper.

### Recorder Registration

A non-exempt shared-library package's TestMain body is exactly `invariant.Run_Test_Main(m)`. Thus,
the suite registers the coverage recorder. A binary component uses its simulation. Any other body,
or no TestMain, is banned.

### Primitive Types

No builtin may be a function parameter, result, or struct field. A builtin has no bundle of its own
and the framework supplies no preset, thus wrap it in a defined type. A stdlib-interface method, a
_test.go file, and a package in opt_out_assertion_mandate_packages are exempt.

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

The simulation TestMain body is exactly `invariant.Run_Test_Main(m, "../**")`. Any other body, or a
missing TestMain, is banned.

### Coverage

The "../**" glob registers the internal package and every package beneath it, so all are
witnessed in one pattern; a narrower argument that omits some is banned.

### Entry

The simulation references only Main among the internal tree's functions; any other internal
function it names is a second entry point and is banned. Internal's exports are otherwise free.

### Blackbox

The simulation directory holds one blackbox test package and no source package: every file is a
_test.go declaring package <name>_test.
