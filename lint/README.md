# lint

Static checker for this monorepo. Run from the workspace root or any module root:

```
go run ./lint .
go run ./lint ./golang_snacks
```

Most rules are local style (snake_case, line length, no interfaces, no package vars, …)
and the diagnostic explains itself. This document covers the one set of rules whose
diagnostics would be illegible without context: the **workspace organization doctrine**.

## The principle: end-to-end dependency injection

The doctrine exists to enforce a single rule: **a function never reaches out to state
it didn't receive**. Every dependency travels through parameters or struct fields,
all the way down from the program's entry point (`main`) to the leaf call site.

The opposite of injection is **impure state** — state read implicitly from the
process, OS, or some global:

```go
// Impure. Reaches out to the OS clock without asking the caller.
func Stamp() string {
    return time.Now().Format(time.RFC3339)
}
```

```go
// Injected. The clock arrives as a field. Tests pass a fake one.
type Stamper struct{ Now func() time.Time }

func (s *Stamper) Stamp() string {
    return s.Now().Format(time.RFC3339)
}
```

Impure state is fine in `main()` — the program is allowed to bind to the real
world there. It's poison everywhere else because:

- **Testability dies at the offending line.** You can't substitute the clock
  without monkey-patching, build tags, or running tests serially around a global.
- **Configurability dies too.** Two callers can't run the same library with
  different time semantics — there's only one wall clock in the process.
- **The dependency is invisible.** The function's signature claims to take `()`
  but actually depends on the OS. Readers and reviewers can't see it.

"End-to-end" means the injection chain is unbroken from `main` to the leaf:

```
main()                          ← binds time.Now to the real OS clock
  → Server.New(clock)
    → Handler.New(clock)
      → Stamper{Now: clock}.Stamp()   ← uses the clock it was given
```

If any step reaches out to `time.Now()` directly, the chain breaks. Substituting
at `main()` no longer reaches the leaf, and the leaf becomes untestable in
isolation. The whole point of injection is to keep that chain intact through
every layer.

The workspace doctrine carves the codebase so the chain is *enforceable*: the
library tier must be free of impure state, and the composition tier (or `package main`)
is the only place where impure bindings are allowed to enter the chain.

### Exemption: telemetry

Telemetry is excluded. For example, logging. Yes, you may pass loggers around if they need to carry
context through call chains but it shouldn't be required. Otherwise, every function would take it as
a parameter. This goes through for other observability libraries such as the assertion library, or
datadog-type libraries.

To put it simply, observability is a layer on top of the program and is not part of the runtime
(technically it is but that's a shitty library if it harms your program in any tangible way).

(technically part 2: the assertion framework does affect runtime via panics/os.Exit but I see it
more as bringing a new primitive to the language. the test suite coverage that it enforces is the
observability part.)

## Workspace organization doctrine

The doctrine carves the monorepo into two kinds of modules and dictates where Go code
may live within each, so the end-to-end injection chain has a known shape.

### Two module kinds

```
james-orcales/
├── golang_snacks/      ← shared library (the only one)
│   └── go.mod          ← module github.com/james-orcales/james-orcales/golang_snacks
├── lint/               ← binary
│   └── go.mod
└── big_bang/           ← binary
    └── go.mod
```

- **Shared library.** Exactly one module: `golang_snacks`. It exists to be imported
  by every binary. The module path is hardcoded in the linter.
- **Binary modules.** Every other module at the workspace root produces one
  deployable executable and is not imported by anything else.

### Where code lives — shared library

```
golang_snacks/
├── go.mod
├── snap/                    ← library at depth 1
│   └── snap.go
├── invariant/               ← library at depth 1
│   └── invariant.go
└── sim/
    ├── sim.go               ← library at depth 1
    └── time/                ← composition tier of sim (one deeper, OK)
        └── time.go
```

- **No `package main` anywhere** — the shared library is imported, never run, so
  an entry point has no place in it.
- Non-main packages live at depth 1.
- **No `internal/` anywhere** — the shared library exists to be imported, so
  visibility-restricting it is forbidden.

### Where code lives — binaries

```
lint/
├── go.mod
├── main.go                  ← package main
└── internal/                ← all non-main code must be under here
    ├── lint.go              ← package lint (binary's own library tier)
    └── lint_test.go
```

- The single `package main` sits at the module root — no `cmd/` directory, no
  second binary.
- Non-main packages must live under `<module>/internal/`. They cannot sit at the
  module root or in any non-`internal/` subtree.
- Consequence: a binary module exposes nothing importable. Go's own `internal/`
  visibility enforces this on the import side; the linter enforces the layout.

Cross-module dependencies flow only **binary → shared library**. Never binary →
binary, never shared library → binary.

### Library tier and composition tier

Within a module, non-main packages may nest **at most one level deep**:

```
golang_snacks/snap/snap.go           ← library tier
golang_snacks/snap/default/x.go      ← composition tier (one deeper, OK)
golang_snacks/snap/default/y/y.go    ← too deep, flagged
```

Major-version directories (`v2`, `v3`, …) are invisible to this count — they're a
Go module-versioning convention, not a real package layer:

```
golang_snacks/snap/v2/snap.go           ← still library tier (v2 doesn't count)
golang_snacks/snap/v2/default/x.go      ← still composition tier
```

The two positions differ in what they're allowed to touch:

- **Library tier.** The spec. Must be free of impure state: no `import "os"`, no
  `time.Now()`, no `fmt.Println`, no `http.DefaultClient`. All outside-world
  dependencies arrive as parameters or struct fields.
- **Composition tier.** The one designated place where a library is permitted to
  bind itself to the real world: provide a `Default` instance wired to `os.Stderr`,
  expose a CLI built on the library, ship an example.

### The two tiers in code

Library tier — pure, every dependency is a field:

```go
// golang_snacks/snap/snap.go
package snap

type Snapper struct {
    Output     io.Writer
    Get_Caller func() string
}

func (s *Snapper) Run() { fmt.Fprintln(s.Output, s.Get_Caller()) }
```

Composition tier — the one place where the binding to the real world lives:

```go
// golang_snacks/snap/default/wire.go
package snap

import (
    "os"
    "runtime"

    "…/golang_snacks/snap"
)

var Default = &snap.Snapper{
    Output:     os.Stderr,
    Get_Caller: func() string { _, _, _, ok := runtime.Caller(1); _ = ok; return "" },
}
```

The library stays substitutable (tests pass a `bytes.Buffer` for `Output` and a
stub for `Get_Caller`), and the binding stays auditable (`grep` for `"os."` in
`snap/default/` enumerates every reach into impure state).

## Resolving diagnostics

### `impure stdlib import` / `impure stdlib call`

The file is in a library-tier package and touches impure stdlib state. Example:

```go
// golang_snacks/snap/snap.go
package snap

import "os"  // ← flagged: library tier may not bind impure state

var Default_Output = os.Stderr
```

Three fixes:

1. **Inject the dependency** — the library-tier example above:

   ```go
   type Snapper struct { Output io.Writer }  // caller supplies it
   ```

2. **Move the code to `package main`** — the composition root is allowed to bind
   anything.

3. **Move the binding to a composition-tier sub-package:**

   ```
   golang_snacks/snap/
   ├── snap.go              ← pure types
   └── default/
       └── wire.go          ← package snap; imports "os", wires Default
   ```

   The sub-package declares `package snap` (its parent's name) and re-exports the
   types, so callers `import "…/snap/default"` and read identically to before.

### `default package must declare 'package <X>'`

A package in a directory named `default` did not declare its parent library's
package clause. The composition-tier package re-exports its library and must
shadow the library's name so the API reads as if no split had happened.

```go
// flagged: golang_snacks/snap/default/wire.go
package wire

// clean: golang_snacks/snap/default/wire.go
package snap
```

The directory is named `default` — a Go keyword that cannot be a package name —
precisely so the parent's name is the natural declaration. Importing
`…/snap/default` then binds to `snap` with no alias. There's a load-bearing
reason beyond ergonomics: `snap.Edit`'s source-line rewriter searches for the
literal `snap.Edit(` in the file; declaring `package snap` keeps that string
correct so the snapshot update doesn't silently fail.

### `binary module forbids package … outside of internal/`

```
mybinary/
├── main.go
└── helpers/         ← flagged: binary's non-main code must be under internal/
    └── helpers.go
```

Fix: move under `internal/`.

```
mybinary/
├── main.go
└── internal/
    └── helpers/
        └── helpers.go
```

If `helpers` is meant to be imported by *other* modules, it doesn't belong in a
binary at all — promote it to `golang_snacks/helpers/`.

### `binary module … declares no func Main in internal/` / `… declares multiple func Main`

Every binary module exposes its entry point as a single free `func Main` living
directly in its top-level `internal/` package. `package main`'s `main()` is a
thin shell that delegates to it:

```
mybinary/
├── main.go          ← package main: func main() { os.Exit(internal.Main()) }
└── internal/
    └── lint.go      ← package …: func Main(…) int { … the real logic … }
```

Go bars `package main` from being imported, hence from being unit-tested, so any
logic left there is untestable. Pinning the entry point to `internal/` keeps
`main()` trivial and gives every binary one auditable seam. The signature is
free — `Main` may take and return whatever the binary needs.

Fix for *no* `func Main`: move the body of `main()` into a `func Main` in
`internal/` and have `main()` call it. A `func Main` deeper than the top level
(e.g. `internal/cmd/`) does not count — hoist it to `internal/` itself. Fix for
*multiple*: a binary has exactly one entry point; collapse the duplicates into
one. The shared library is exempt from the *requirement* — it is imported, not
run — but it *may* still expose a `func Main` of its own to represent an
embeddable entry point a host can call; the rule never flags it either way.

### `shared library forbids internal/ directories` / `shared library forbids package main`

```
golang_snacks/
├── snap/
│   └── internal/        ← flagged: shared library is fully exposed
│       └── helper.go
└── run/
    └── main.go          ← flagged: a fully-importable library owns no entry point
```

Fix for `internal/`: rename it (e.g. to `snap_internal/` if it's a
composition-tier helper), or promote the contents to a normal package. Fix for
`package main`: move the entry point to its own binary module.

### `package … exceeds library tier`

A package may have at most one non-main package above it. The count starts at the
module root for a shared library, and at the top-level `internal/` for a binary —
`internal/` is where all of a binary's non-main code lives, so it is the starting
point, not a counted level (the same role the module root plays for a shared
library). `v{N}` version dirs are not counted either.

Shared library — count from the module root:

```
golang_snacks/foo/foo.go              ← library tier
golang_snacks/foo/bar/bar.go          ← composition tier (OK)
golang_snacks/foo/bar/baz/baz.go      ← flagged: too deep
```

Binary — count from `internal/`, which does not itself count:

```
mybinary/internal/lint.go             ← library tier (holds func Main)
mybinary/internal/foo/foo.go          ← library tier
mybinary/internal/foo/default/x.go    ← composition tier (OK)
mybinary/internal/foo/bar/baz/x.go    ← flagged: too deep
```

So the deepest legal package is `shared/foo/[v{N}]/default` or
`binary/internal/foo/[v{N}]/default`: a pure library package, then its impure
`default` composition tier.

Fix: either hoist the too-deep package up (make it a sibling one level higher) or
fold its contents into its parent. If a package has genuinely grown enough
subdivision to need three layers, it should fission into separate sibling
packages instead.

## Hardcoded knowledge

- The shared library module path is `shared_library_module_path` in
  `lint/internal/lint.go`. Change it there if the shared library ever gets renamed.
- Major-version segment detection uses `^v[0-9]+$` (`module_index_version_re`).
