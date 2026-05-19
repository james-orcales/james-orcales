# Linter

This linter is a best-effort programmatic enforcement of TigerStyle from TigerBeetle. It also draws
inspiration from gingerBill, the Odin programming language, Antithesis, and Resonate. It's nitpicky
and is designed for AI-centric workflows. The idea is to heavily constrain the design space, letting
AI arrive on a select few architectural decisions without human intervention; paving the way for
reliable eyes-closed vibecoding.

## Blanket Bans

These are rules that apply to both source and test files.

- Variable shadowing
- Naked return
- func init()
- Compound statements in if/else if clauses (&& and ||)
- iota
- Dot imports
- Interace declaration
- Grouped declarations using `const/var/type ( ... )` blocks.
- Switch cases using `fallthrough`
- Blank import e.g. `import _ "pkg"` (init-side-effect smuggling)
- Whitebox tests (including main_test)

## Blanket Requirements

These are rules that apply to both source and test files.

- goimports compliance
- Functions are a max of 70 lines
- Named return
- Public struct fields. Exported structs cannot have fields of private types.
- Keyed struct literals e.g. `Coordinate{0, 1} -> Coordinate{X: 0, Y: 1}`
- Comments start with a space, followed by a capital letter ending in a period or colon.
- Package-level declarations have doc comments.

## File splitting

A package may contain up to <total_sloc>/30,000 files. Source, test, and unique build tag
combinations have independent `total_sloc` counts.

## Global mutable variables

Global mutable variables are only permitted when initialized with calls to `regexp.MustCompile()`
or `errors.New()`. Compile-time interface assertions via `var _ Interface = (*Type)(nil)` are also
permitted.

In packages named `*_default`, a single declaration `var Default = <expr>` is permitted with any
initializer. The name must be exactly "Default" and the declaration must be a single-name,
single-value spec. Some packages are better suited to expose global APIs, typically for Telemetry
and observability.

## Free functions only

All methods are banned with a preset list of exceptions that satisfy stdlib interfaces.

## Naming

Ada_Case for exported identifiers and snake_case for unexported identifiers. With the exception of
`func TestMain(m *testing.M)` in `*_test.go` files. Files and directories must use FQDN-style naming
but with underscores instead of hyphens.

## Line length

Line length is capped at 100 columns. Exceptions: lines that are SOLELY a URL, or lines within
triple backticks (code blocks).

## Variable discards

Discards in the style `_ = func` are banned. In the case of multi-variable declarations, it is
permitted if at least variable is named e.g. `_, err := foo()`

## Input structs

A package-level function's parameters and return values are evaluated independently. If any type
appears twice or more across the parameters, the function MUST accept a single input struct.
Exception: variadics may live as a separate parameter to an input struct. If any type appears twice
or more across the return values, it MUST return a single output struct. Struct names must be
prefixed with the function name and follow the same export convention (input struct first, output
struct second, declared immediately above the function). The function MUST accept the input struct
as a pointer as it's often bigger than 16 bytes.

## Declaration Order

Package-level variables and constants MUST be declared at the top of the file. Variables first and
then constants. Type declarations may appear anywhere with the exception of [Input
structs](#input-structs)

## Assertions

Assertions encode invariants that must hold in production, not observations about what the tests
happen to exercise. Respect this carefully — the linter only softly enforces it. If you find
yourself writing many `invariant.Excluding()` assertions and straining against the function-length
and line-length caps, that pressure is a signal: revisit whether better test inputs could eliminate
those Excluding assertions rather than compressing them — each is deliberately boilerplate,
occupying a whole line.

Remember that test inputs can be generated programmatically — especially useful for proving
reachability of the `Hi` threshold in `invariant.Distinct_Boundary()` assertions.

On meeting the function-length and line-length caps: another source of tension is high-cardinality
`invariant.Cross_Product()` calls. One lead is trimming the function signature; another is making
the caller guarantee that a conditional collapses to a constant — e.g. promoting
`Sometimes(slice == nil)` to `Always(slice == nil)`. In short: push ifs up and fors down.

The following mandated assertions do not apply to *_test.go files.

### Post-conditions

Every function that has named returns with at least one trackable type (see
[Tracked types](#tracked-types)) must open with `defer func() { ... }()` as its very first
statement. The defer body is a no-arg, no-return FuncLit and must contain exactly one
`invariant.Cross_Product(...)` call covering every named return. Functions with no named returns,
or whose named returns are all untracked, are exempt.

The defer must not be preceded by any other statement. Cleanup defers must come after the assertion
defer (LIFO ordering means the assertion defer fires last on return).

### Pre-conditions

Immediately after the post-condition defer (or as the first statement for functions that are defer-
exempt), the function body must contain a second `invariant.Cross_Product(...)` call — the prologue
— covering every receiver and every parameter.

Only axis-builder bindings (`x := invariant.Distinct_Boundary(...)`) and nil-guarded `if X != nil
{ ... }` blocks that dispatch into an independent assertion chain may appear between the defer and
the prologue `Cross_Product`. The prologue scan terminates at the first non-matching statement.

### Tracked types

The requirement walker emits one obligation per tracked leaf reachable from a parameter, receiver,
or named return. Tracking rules by type:

- **Integers** (`int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`, `uint16`, `uint32`,
  `uint64`, `uintptr`, `byte`, `rune`): requires `boundary_int` + `zero_int`.
- **Floats** (`float32`, `float64`): requires `boundary_float` + `zero_float` + `nan_float` +
  `inf_float`.
- **Strings**: requires `boundary_int` (on length) + `zero_string`.
- **Booleans**: requires `bool`.
- **Pointers**: requires a nil-check (`pointer` kind). If the pointee is a same-file struct, the
  walker recurses into its `int`, `string`, `bool`, `slice`, `map`, `chan`, and pointer fields
  (one level; self-referential structs are unrolled at most twice via a visited-set guard,
  `check_invariant_assertions_recursion_unroll_threshold = 2`).
- **Slices / variadics**: requires `boundary_int` + `zero_slice`. Must be covered in BOTH the
  prologue and the post-condition defer (length is mutable inside the body).
- **Maps**: requires `boundary_int` + `zero_map`. Must be covered in BOTH the prologue and the
  post-condition defer.
- **Channels** (`chan T`, `<-chan T`, `chan<- T`): requires a nil-check (`pointer` kind) on the
  channel itself, plus `boundary_int` + `zero_int` on `cap(<chan>)`.
- **Anonymous / inline structs**: fields are walked unconditionally (no name to check against
  the mutex-locked index).
- **Fixed-size arrays** (`[N]T`), function types, interface types: not tracked; no obligation
  emitted.

### Mid-function declarations

Any statement that introduces a new identifier (`:=` or `var` in a block body, or an init clause
of `if`/`for`/`switch`) must be immediately followed by an `invariant` assertion covering every
non-discard LHS identifier. For `if`/`for`/`switch` init clauses, the assertion must lead every
reachable branch body. `defer` and `go` function-literal scopes are not walked for declarations.

### Predicates

Predicate arguments to `invariant.Always` / `invariant.Sometimes` (and their `Is_` /
`Recorder_` variants) may not use compound boolean operators (`&&`, `||`). Each conjunct must be
its own axis inside `Cross_Product`. Guard inner-field access with `if X != nil { ... }` instead.

### Boundary literals

`Lo` and `Hi` fields of `invariant.Boundary_Input` must reference named constants, not inline
literals. The constant's sign must match the bound it occupies — a constant used as `Lo` must be
non-negative-anchored; a constant used as `Hi` must be positive-anchored.

### Obligation-skip machinery

Two per-file / per-function indexes narrow the set of obligations:

**Mutex-locked structs** (per file). The walker scans every package-level `type T struct { ... }`
declaration. If any field's type (named or unnamed embed) resolves to `<sync-alias>.Mutex` or
`<sync-alias>.RWMutex` (the alias is looked up against the file's local import names), the struct
name is recorded as locked. When a pointer-to-locked-struct or value-of-locked-struct parameter is
encountered during requirement emission, the walker skips descending into the struct's fields —
the struct is treated as opaque. The top-level pointer nil-check obligation still applies.

**Always-nil pointers** (per function). Built during prologue scanning. When the prologue contains
`invariant.Always(X == nil, ...)` (or `Is_Always` / `Recorder_Always` / `Recorder_Is_Always`
variants), the LHS ident-chain `X` is recorded. `Cross_Product` / `Recorder_Cross_Product` calls
are unwrapped so each inner `Always(X == nil, ...)` argument contributes its path. When a
parameter path appears in this set, field recursion into the pointed-to struct is suppressed — the
pointer is provably nil at entry so its fields can never be dereferenced. The pointer's own nil-
check obligation is still satisfied by the same `Always(X == nil)` call.
