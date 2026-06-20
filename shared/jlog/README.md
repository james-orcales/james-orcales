# jlog

A zero-allocation, dependency-injected, flat JSON logger written in the house
free-function style (no methods, no fluent builder). One call per line:

```go
logger := jlog.New(jlog.New_Input{Writer: os.Stderr, Floor: jlog.Level_Info})

jlog.Logger_Info(logger, "request done",
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
  `shared/time`. No pretty console writer, hooks, sampling, or CBOR: just the encoder
  and the diode.

- **Faster, and faster under stress.** jlog wins most of the formatting benchmarks
  above, and its diode is ~2× quicker than zerolog's *and* allocation-free even while
  dropping. zerolog's diode allocates 3×/op and allocates *more* as the drop rate
  climbs — worsening exactly when you are already degraded. jlog's memory is bounded by
  a fixed ring (TigerStyle: no unbounded growth) and its drop path allocates nothing,
  so shedding load never adds GC pressure.

- **Async by default, on purpose.** This repo's linter treats instrumentation as
  fire-and-forget — a logger may be ambient precisely because it must never block the
  program. Synchronous logging is real harm here: a production sink is usually a disk
  or another server, so a synchronous write drops a syscall and its stalls onto the
  caller's hot path. The default logger is non-blocking; the syscall lives on the
  diode's drain.

- **Pure, separable core.** The library tier (`shared/jlog`, `shared/diode`) is pure —
  the clock, caller lookup, and sink all arrive as fields, so it is trivially testable
  and holds no globals. The single ambient binding lives in the composition tier
  (`shared/jlog/default`). zerolog wires `os.Stderr` and a package-global logger in
  directly.

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
go test ./shared/jlog/ -run='^$' -bench=Benchmark_ -benchmem
```

(The zerolog column is produced by copying the matching `Benchmark*` workloads
into a throwaway module that imports the real `rs/zerolog`; it does not build
inside this repo.)

## Non-blocking default

`shared/jlog/default`'s `Default` logger writes through a `shared/diode` — a lock-free
ring that hands finished lines to a background drain, so a producer never waits on the
sink. The table above writes to `io.Discard` (zero sink cost); the one below writes to
`/dev/null`, a *real* sink whose `write` syscall the caller would otherwise pay on its
own goroutine.

| Caller cost, one line to `/dev/null` | jlog                | zerolog v1.35.1  |
| ------------------------------------ | ------------------- | ---------------- |
| Synchronous (`New` → the sink)       | 485 ns, 0 allocs    | 498 ns, 0 allocs |
| Non-blocking diode                   | **84 ns, 0 allocs** | 171 ns, 3 allocs |

Both diodes use the same shape: a poller with a 100 ms interval over a 1000-slot ring.

The diode moves the `write` syscall off the caller, so logging a line costs ~5.8× less
and never blocks — and jlog's diode allocates nothing. `/dev/null` is the *conservative*
case: its syscall is fast and never stalls. A terminal, file, pipe, or network sink is
slower (microseconds to milliseconds) and can block, where the synchronous caller waits
the full time while the diode caller stays flat at ~84 ns.

Synchronously the two loggers cost about the same. Through a diode jlog is ~2× faster
and allocation-free, while zerolog's allocates 3 per write — a fresh ring bucket on every
store, an escaping slice header, and a pool miss for each dropped line's buffer (see
`../diode/SPECIFICATION.md`). zerolog's diode also writes `consider using a larger diode`
to stderr on every collision under sustained load (silenced for this measurement); jlog
drops silently.

The trade is that the diode drops the oldest lines under sustained overload (reported
via its alerter) and loses anything still buffered at an unflushed exit; see
`../diode/SPECIFICATION.md`.

Reproduce (the zerolog column comes from a throwaway module importing the real
`rs/zerolog`, same as the table above):

```
go test ./shared/jlog/default/ -run='^$' -bench=Benchmark_Caller -benchmem
```
