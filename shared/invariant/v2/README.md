# invariant v2

Property-style assertions for Go that record which assertions ever fired during
a test suite, so untested branches surface as failures at the end of the run.

## Two tiers

- **`invariant/v2/`** — pure library tier. All dependencies (filesystem,
  `runtime.Callers`, stdout, `os.Exit`) arrive as fields on a `Recorder`
  struct. The package imports neither `os` nor `runtime`. Suitable for testing
  the package itself or for tests that need to redirect I/O.

- **`invariant/v2/invariant_default/`** — composition tier. Wires the pure
  library to real OS primitives, exports a `Default` `*Recorder` singleton, and
  re-exports the surface so callers only need one import.

Day-to-day usage is the composition tier:

```go
import invariant "github.com/james-orcales/james-orcales/shared/invariant/v2/invariant_default"

func TestMain(m *testing.M) {
    invariant.Run_Test_Main(m)
}

func F(x int) {
    invariant.Ensure(x >= 0, "x must be non-negative")
    invariant.Sometimes(x == 0, "x can be the boundary")
}
```

Testing the package itself uses the pure tier:

```go
import invariant "github.com/james-orcales/james-orcales/shared/invariant/v2"

func Test_Something(t *testing.T) {
    recorder := &invariant.Recorder{
        Output:      &bytes.Buffer{},
        File_System: fstest.MapFS{...},
        Get_Caller:  func(skip int) (invariant.Frame_Information, error) { ... },
        Exit:        func(code int) { /* record */ },
        Is_Test:     true,
    }
    invariant.Recorder_Ensure(recorder, true, "...")
}
```

## Assertion kinds

| Primitive          | Tracker?  | On failure                                           |
|--------------------|-----------|------------------------------------------------------|
| `Ensure`           | yes       | Emits message, panics (or `Exit(1)` if Fatal_Failures) |
| `Always`           | yes       | Emits message, no panic                              |
| `Sometimes`        | yes       | Records true vs false branches separately            |
| `Boundary`         | yes       | Enforces `Lo <= X <= Hi`; tracks endpoint hits as Cross_Product buckets |
| `Reachable`        | yes       | Increments tracker; no condition                     |
| `Unreachable`      | no        | Emits message, panics / `Exit(1)`                    |
| `Unimplemented`    | no        | Emits message, panics / `Exit(1)`                    |
| `*_When_Reachable` | no        | Same shape, never registers in the tracker           |
| `*_Err_Is_Nil`     | depends   | Err-aware variant: format embeds the error           |
| `X_*`              | as parent | Closure-form (lazy condition)                        |

## Coverage reporting

At test teardown (`Run_Test_Main` → `Analyze_Assertion_Frequency`):

- Every pre-registered assertion whose `Frequency == 0` is reported as
  *never true* and the process exits with code 1.
- Every `Sometimes` whose `False_Frequency == 0` is reported as
  *never false*.

The pre-registration step walks every `.go` file in the registered packages,
finds every `invariant.*` call site, and seeds a tracker entry per literal
call. Runtime firings increment those entries.

## Fatal_Failures

Setting `Recorder.Fatal_Failures = true` swaps the `panic` in every Ensure-class
primitive for `Exit(1)`. Use this in environments where a panic would be
swallowed or where a structured exit-code signal is preferred.

## What's not here yet

### TODO: Path coverage tracking (much later)

The Phase 2 layer designed in the planning discussion but explicitly deferred
to a future release. The mechanism:

- Use `go/ssa` + `golang.org/x/tools/go/pointer` to build a sound call graph
  across the analyzed packages, resolving function-typed struct field
  invocations (C-style interfaces) and other indirect calls.
- Enumerate every statically-reachable path from `main` to each `invariant.*`
  primitive. Register one tracker entry per distinct path, keyed by the full
  stack.
- At runtime, hash the stack of every primitive firing and look up the matching
  entry. After the test run, any path that exists statically but was never
  walked at runtime = coverage failure ("this code path was never exercised").

Why deferred:

- Significant new dependencies (`golang.org/x/tools/go/ssa`, `pointer`).
- Whole-program pointer analysis runs on every test invocation — adds seconds
  to startup.
- The MVP's contract-style coverage (never-fired / never-false) already
  surfaces most of the value at a fraction of the implementation cost.

Constraints already enforced by the linter that make Phase 2 tractable when we
get to it: no recursion, no Go-style interface method sets, no package
vars (except whitelist), single `main` entry. Under those constraints the
call graph is a DAG with no virtual dispatch slop, so pointer analysis gives
exact path enumeration rather than over-approximation.
