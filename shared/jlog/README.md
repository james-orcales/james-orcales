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
