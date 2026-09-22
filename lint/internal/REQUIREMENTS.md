
# Diagnostics

A diagnostic is tier one or tier two. Tier one always prints; any tier-one anywhere in scope
suppresses tier two, which prints only when no tier-one fired and so may rely on tier-one
contracts.

### Tier One

Every per-file rule not listed under Tier Two.

### Tier Two

Self Recursion, Mutual Recursion, Init Functions, Mutable Globals, Methods,
Struct Tags.

# Repository

These rules govern every tracked file, whatever its language.

### Ignored Directories

The linter never scans the top-level `third_party/` drop-zone, any `vendor/`, `.jj`, `.git`, or a
gitignored path. A nested `third_party/` below the repository root is first-party code, scanned
like anything else.

# Component Layout

The repo is one Go module; each top-level directory of Go code is a component. The layout splits
pure from impure. Shared component: `shared/foo (pure) -> shared/foo/default (impure)`. Binary
component: `main (impure) -> pkg (pure)`.

### Single Module

The repo is one Go module: exactly one `go.mod`, at the root, with no `go.work` and no nested
`go.mod`. A component is a top-level directory of Go source, not a module of its own.

### Shared Component

The shared component, `shared`, has no `internal` directory at any depth and declares no
`package main`. It is fully importable, never executed. Its root-relative directory is
lint.json's `shared_component`.

### Binary Component

In a binary component, every non-main package lives under a top-level pkg directory; code at the
component root or any non-pkg directory is banned.

### Binary Isolation

A binary component imports no package of another binary component. Go's `internal` directory
barred this; `pkg` does not, so the rule does. Code two binaries share belongs in `shared`.

### Main Package

A binary component holds exactly one main package, and it sits at the component root; a `cmd`
directory is banned.

### Entry Point

A binary component declares exactly one free func Main in its top-level pkg directory; it is
the entry point its thin package main calls, and zero or several Main functions are banned. A shared
component may also expose a func Main, representing an embeddable entry point.

### Tier Depth

A non-main package has at most one non-main package above it, not counting `v{N}` dirs or, in a
binary, the top-level pkg dir. So the deepest nest is `shared/foo/[v{N}]/default`, a library
package then its impure default tier, or `binary/pkg/foo/[v{N}]/bar`, both pure.

### Impure Imports

A pure package may not import os, os/exec, os/user, os/signal, flag, runtime, math/rand, or
crypto/rand; inject these instead.

### Impure Calls

A pure package may not call the clock, stdout, or network via time, fmt, net, or net/http
(time.Now, fmt.Print, http.Get, net.Dial); inject these instead.

### Transitive Purity

A pure package imports only pure packages, never an impure one; purity then holds across the whole
import closure by induction, with no graph walk. It may import a package listed in lint.json's
`instrumentation_packages` — a write-only side channel the ban exempts.

### Transitive Stdlib

A pure package calls no curated stdlib API that transitively reaches the impure set; this binds its
test files too, so an impure reach is caught even in a _test.go file.

### Binary Purity

A binary module keeps its impurity in package main and its pkg packages pure. Tests may be
impure.

### Binary Default Tier

A binary component declares no default package. Only its package main is impure, so a package
under pkg gets no release from the purity bans, whatever its depth.

### Library Purity

A shared library keeps its packages pure; each package is wholly pure or wholly impure, never
mixed.

### Default Package Impurity

A shared library may add one `default` package — a `default/` directory declaring its parent's
package clause — as the home for any impure global API; it is the only impure package allowed.

# Source And Test Bans

These constructs are forbidden in source and test files alike.

### Dot Imports

An import binds a package name; dot imports are banned. Dot imports pollute the global namespace
and complicates type resolution.

### Blank Imports

An import names a package a caller uses; blank imports are banned. Blank imports indicate that
a library is being imported solely for it's global side-effects, an implicit control flow.

### Grouped Declarations

Each const, var, and type stands alone; parenthesized groups are banned. Simple reduction of the
language surface area. It also helps grepping for constants with the regex `^const foo =`

### Iota

A constant spells out its value; iota is banned. Make the value explicit instead of relying on
declaration order.

### Panic

Only `shared/sim/aver/**` calls `panic`; assertion enforcement owns that mechanism. Every other
package uses `aver.Always` so assertion reachability stays observable.

### Generics

Outside packages listed in lint.json's `instrumentation_packages` and exact packages
`shared/testify`, `shared/slices`, and `shared/sort`, type and func declarations have no type
parameters. Shared exceptions cover source and tests, but not child packages. Instrumentation may
use type parameters so one adapter can observe arbitrary types without production dependencies.

### Interface Declarations

A package declares no interface. The method-free forms `interface{}` and `any` are banned too. A
`_test.go` file is exempt, as is a package listed in `opt_out_interface_ban`.

### Variable Shadows

A name declared in an inner scope must not shadow one from an outer scope. The outer scope is the
whole package, sibling files included; an external test package is a separate name space.

### Mutable Globals

A global mutable var is permitted only via `regexp.MustCompile` or `errors.New`, as a
`var _ Interface = (*Type)(nil) assertion`, or as a single `Default` var in a package at or under
a directory listed in lint.json's `instrumentation_packages`.

### Discards

A bare `_ = value` discard is banned; a multi-value assignment may discard when at least one
variable is named, as in `_, err := foo()`.

### Init Functions

A package declares no func init; wire setup explicitly instead. Libraries should expose a
`func Init()` instead.

### Empty Bodies

A function body holds at least one statement; an empty body is banned.

### Closure Bodies

Outside test files, a closure body is empty or holds only direct function calls; the last may be
returned. This keeps callback logic in named functions while permitting bounded no-op callbacks.

### Methods

All methods banned except `Assertion_Builder` methods in `shared/sim/aver` and its `default` tier.
Make every other method free function with receiver as first parameter.

### Self Recursion

A function never calls itself by bare name, including inside a closure or a go statement — the call
graph must be acyclic regardless of stack. A method call does not count; a file in
opt_out_recursion_ban contributes nothing to its package's graph.

### Mutual Recursion

A function never reaches itself through a cycle of bare-name calls anywhere in its package,
closures and go statements included — the call graph is a DAG. A cycle across packages cannot
compile, so the graph stops at the package boundary.

### Compound Conditions

An if condition holds a single term; split && and || into nested ifs.

### Naked Returns

A return statement names its values; a bare return is banned.

### Fallthrough

A switch case does not fall through to the next.

### Bare Loops

A for loop carries a condition, range, or post clause; a bare for that loops forever is banned.

### Struct Tags

A struct tag uses only the stdlib keys json, xml, and asn1; any other key is banned.

### Blank Mutexes

A sync.Mutex or sync.RWMutex struct field is named so it can be locked; a blank-named one is banned.

# Source And Test Requirements

These forms are required in source and test files alike.

### Type Resolution

Every bare type in a package declaration or function signature resolves to a predeclared type or a
type declared by that package. Qualified types outside parsed scope remain unjudged.

### Default Package Name

A package in a directory named `default` declares the package clause of its parent directory,
shadowing the library it re-exports.

### Entry Point First

A file's main, Main, or TestMain is its first function declaration.

### Named Returns

A function with results names every one of them.

### Array Capacity

An array type's capacity is a named constant, never a literal, as in [BUFFER_SIZE_MAX]byte. The
`[...]T{…}` form declares no capacity and is exempt.

### Keyed Struct Literals

A struct literal names each field, as in Coordinate{X: 0, Y: 1}. A type the run never parsed is
exempt, since nothing says whether it is a struct.

### Exported Struct Fields

An exported struct exposes no field of an unexported type, through any struct its own package
declares.

### Exported Types

A package-level type declaration or alias is exported. Function-local types and _test.go files
are exempt.

### Struct Field Public Identifier

Every struct field name begins with a capital letter; an unexported field is banned.

### Struct Field Type Encapsulation

A struct field's type must be exported if the struct is exported.

### Name Style

Exported identifiers use Ada_Case, unexported use snake_case, TestMain aside; every const uses
SCREAMING_SNAKE_CASE instead, whatever its scope or export status. A numeric capacity may occupy
one underscore-delimited Ada_Case segment, as in Enum_3_Int.

# Deterministic

Every pure package is held, atop purity, to bans making it reproducible; a package
in lint.json's pure_but_indeterministic_packages opts out, and the bans bind _test.go too.

### Entry Format

A pure_but_indeterministic_packages entry is an exact-path glob naming a pure package
to opt out: `shared/io` releases that package, `shared/io/**` its subtree, `*` and
`**` spanning one path segment or many; a `!`-prefixed entry negates, always winning.

### Goroutines

A deterministic package starts no goroutine; the kernel decides goroutine
interleaving, the root of nondeterminism.

### Channels

A deterministic package uses no channel — no channel type, send, or receive —
since cross-goroutine handoff order is not the program's to decide.

### Select

A deterministic package uses no select; the runtime randomizes the choice among
the ready cases.

### Floats

A deterministic package uses no float32 or float64; IEEE-754 results diverge across
platforms through fused-multiply-add contraction and extended-precision intermediates.

### Banned Imports

A deterministic package imports none of time, context, sync, or sync/atomic; it
injects the clock and holds no concurrency to guard.

### Import Induction

A deterministic package's first-party imports are themselves deterministic, save
those listed in lint.json's `instrumentation_packages` — write-only side channels
the induction exempts.

### Impurity

Determinism is stricter than purity, so only pure packages are held; an impure
package (the main package, a default tier) is never deterministic and needs no
pure_but_indeterministic_packages entry to be excused.

### Coverage

A pure_but_indeterministic_packages entry that names a concrete path matching no pure
package is reported — a typo or stale path releasing nothing; a root-anchored
wildcard, naming no path, is exempt.

### Coverage Scope

A scoped run judges coverage only for entries within the scope it parsed; an entry
for a module outside the scope is left to the whole-workspace run, which alone
parses every module.

# Stdlib Time

Stdlib time may be imported only by the shared module's sim/time/default gateway;
every other shared-module package injects the Clock instead.

# Event Loop

Blocking and non-blocking IO runs on one event loop; pure code submits to it and never drives or
mints it.

### Driver

The Driver advances time and drives the loop. Only package main or a test can create it through
New_Virtual_Timeline or New_Operating_System_IO. Only those packages can name time.Driver.
Internal code takes time.Timeline, and the harness drives it.

### Gateway

Raw IO stdlib lives only in sim/nbio/default and sim/time/default. Instrumentation,
tests, generated files, and package main are exempt. os/default may import syscall alone. Route
other raw IO through shared/sim/nbio.

### Seed

A simulated backend takes only a seed. New_Sim has one integer seed parameter, and no Sim_* helper
lets a caller script outcomes. Thus, a run is a pure function of its seed.
