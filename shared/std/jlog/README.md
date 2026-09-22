# jlog

A zero-allocation, dependency-injected, flat JSON logger written in the house
free-function style (no methods, no fluent builder). One call per line:

```go
configuration := jlog.Logger_Configuration{Stream: sink, Clock: clock}
var storage [4096]byte
logger := jlog.New(&configuration, storage[:], jlog.LEVEL_INFO)

jlog.Logger_Info(&logger, "request done",
    jlog.String("method", method),
    jlog.Integer("status", status),
    jlog.Duration("latency", elapsed),
)
// {"level":"info","method":"GET","status":200,"latency":1500000,"message":"request done"}
```

Output is flat (scalars and scalar slices only); see `SPECIFICATION.md` for the
behaviour contract and the package doc comment for the design rationale.

## Why this over zerolog

zerolog is broader and battle-tested; jlog is a deliberate subset for one job — a
zero-allocation JSON logger for an observability sink, in this repo's style. What being
narrow buys:

- **Flat by construction.** The field constructors build only scalars and scalar
  slices — there is no nested-object builder. Every line comes out flat, the shape log
  backends index and `jq` queries cleanly (rationale in the package doc and
  `documentation/resources/kellybrazil.*`). zerolog lets you nest objects; jlog makes
  the queryable shape the only shape.

- **Barebones.** Roughly 6× less code — ≈1.1k lines of Go source against zerolog's
  ≈6.5k — and zero third-party dependencies, only the standard library and
  `shared/std/sim/time`, and `shared/std/sim/nbio`. No hooks, sampling, or CBOR, and the
  zero-allocation core is just the encoder. A pretty console writer does exist —
  `Console` — but it is opt-in and reads *finished* JSON lines back as an `nbio.Stream`,
  so the hot path never carries console work.

- **Allocates nothing, ever.** The caller owns every byte: the `Logger_Configuration`,
  the line storage `New` takes, and the line and prefix storage `Logger_With` takes. A line
  that outgrows its storage is dropped whole and counted in `Logger.Dropped`. The price is
  ownership: a `Logger` belongs to one goroutine. Goroutines share a
  `Logger_Configuration` and each keep their own `Logger` (`Logger_With` makes one).

- **Never waits, on purpose.** This repo's linter treats instrumentation as
  fire-and-forget — a logger may be ambient precisely because it must never block the
  program. Each line is submitted with `nbio.Write` from the `Logger`'s own storage, which
  stays borrowed until the Stream retires it; an emit while it is borrowed drops and counts.
  jlog neither waits for nor drives that retirement — the root does, per `inject_io`.
  Buffering, ordering across goroutines, and dropping under a slow sink belong to the Stream
  below: put a `shared/std/sync/diode` in front of any sink that can fall behind, and have the
  root loop `diode.Writer_Rearm`. No drain goroutine and no sleep.

- **Pure, separable core.** The library tier (`shared/std/jlog`) is pure —
  the clock, caller lookup, and sink all arrive as fields, so it is directly testable
  and holds no globals. The one ambient binding — the `os.Stderr` default logger — lives
  in the composition tier (`shared/std/jlog/default`); a console logger takes whatever sink it
  is given, and conventionally that is `os.Stderr` too, so stdout stays clean for program
  output. zerolog wires `os.Stderr` and a package-global logger in directly.

## Benchmarks

Workloads are a 1:1 copy of `rs/zerolog`'s own benchmark suite (same fields, the
same `"Test logging, but use a somewhat realistic message length."` message, the
same `b.RunParallel`), so the two columns are directly comparable.

Hardware:

```
Apple M4 (arm64) — 4 P + 6 E = 10 logical cores
RAM 16 GiB   L1 128 KiB   L2 16 MiB
macOS 26.2 (Darwin 25.2.0)   Go test -bench, GOMAXPROCS=10
```

Versus `github.com/rs/zerolog v1.35.1`:

| Workload                         | jlog          | zerolog v1.35.1 |
| -------------------------------- | ------------- | --------------- |
| Empty line (`Log().Msg("")`)     | **2.5 ns**    | 4.3 ns          |
| Info + message                   | **9.1 ns**    | 13.3 ns         |
| Disabled logger                  | 0.59 ns       | **0.35 ns**     |
| 4 fields + message               | 23.9 ns       | **22.4 ns**     |
| Sub-logger (4 context fields)    | **9.3 ns**    | 13.5 ns         |

Every row is **0 B/op, 0 allocs/op** for both loggers.

jlog is faster on the common cases — empty lines, single-message lines, and
sub-logger lines are 1.5–1.7× quicker. zerolog's builder amortizes better on a
line carrying many fields at once, edging the 4-field case by ~6%; the disabled
case is a sub-nanosecond wash. Both never allocate.

Reproduce:

```
go test ./shared/std/jlog/ -run='^$' -bench=Benchmark_ -benchmem
```

(The zerolog column is produced by copying the matching `Benchmark*` workloads
into a throwaway module that imports the real `rs/zerolog`; it does not build
inside this repo.)

## Default stream

`shared/std/jlog/default`'s `Default` is one `jlog.Logger` the package-level functions (`Info`,
`Error`, ...) write through, with one owner like any other. It is the zero Logger until the
root assigns `jlog.Default = jlog.New_Default_Logger(&state, loop.Storage, stderr)` over a
`jlog.Default_Logger` state it owns, with its `Line` storage filled; from then on
each line goes to that descriptor with `nbio.Storage_Write`, and the root's loop retires it.
