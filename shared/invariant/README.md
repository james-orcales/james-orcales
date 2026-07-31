# invariant

`shared/invariant` is the pure assertion engine. `shared/invariant/default` supplies the
OS-backed recorder and the package-level sugar used by application code.

## Eager guards

`Always(condition, message)` is deliberately bare and eager. A false condition panics at
its callsite; registration seeds its literal message so an unreached guard is also visible as a
coverage gap.

## Deferred assertion chains

An assertion set is one fluent expression:

```go
invariant.Assertions("queue").
    Sometimes(queue.Empty(), "The queue is empty.").
    Sometimes(queue.Full(), "The queue is full.").
    Ensure()
```

`Sometimes` records neither coverage nor failure itself. It captures one boolean by fluent
ordinal; `Ensure` resolves the registration-owned plan and credits exactly one of the axis's
independent true or false obligations. Repeated messages are safe because identity is namespace,
ordinal, and message.

There is no cross product. The suite witnesses every individual branch, not every combination, and
there are no tuple cells, carves, polarity references, combination gaps, or legends.

## Integer domains

Every concrete integer width has chain methods for bounded and enumerated domains:

```go
invariant.Assertions(namespace).
    Range_Int(value, Minimum, Maximum).
    Ensure()

invariant.Assertions(namespace).
    Range_Holed_Int(value, Minimum, Maximum, Hole_A, Hole_B, Hole_B, Hole_B).
    Ensure()

invariant.Assertions(namespace).
    Enum_3_Uint8(value, State_New, State_Ready, State_Done).
    Ensure()
```

In the full build, Range enforces both bounds at `Ensure`; Range_Holed also enforces fixed
strictly-interior hole slots. Signed methods carry four slots and unsigned methods three. Distinct
holes are ascending, with unused slots repeating the final hole. Coverage includes successful lower
and upper guards, both interval boundaries when distinct, and eligible interior 0, 1, 2, and -1
witnesses. A boundary cannot be excluded.

In the full build, Enum, Enum_3, and Enum_4 enforce their fixed member capacities at `Ensure`.
Members are statically resolvable, strictly ascending, and distinct. Registration seeds one
successful membership guard and gives each member an independent true/false axis.

## Type-owned helpers

A defined type keeps its contract beside the type in a trailing-namespace helper:

```go
type Token string

func Token_Invariants(token Token, namespace invariant.Namespace) {
    invariant.Assertions(namespace).
        Sometimes(len(token) == 0, "The token is empty.").
        Ensure()
}
```

Registration instantiates that template at each literal callsite namespace. A chain is one unsplit,
nonempty call nest ending in `Ensure`; returning a half-built builder is unsupported.

## Coverage lifecycle

Registration parses the selected packages before the suite and seeds every individual obligation.
It also publishes the immutable ordered handles runtime emission consumes; runtime never rebuilds an
identity from a message. A failed `Ensure` preflights the whole plan and credits nothing.

Plain tests and fuzz processes record. Benchmarks and ordinary binaries enforce without recording,
and ordinary binaries do not consult registration plans.

## Build modes

An untagged build retains the full deferred and recording-capable behavior above. Any of
`invariant_disable_coverage`, `prd`, `prod`, or `production` selects eager production enforcement.
Production aliases may be combined. `Always`, Range, holed Range, and every Enum capacity panic at
the violating call; `Sometimes` and `Ensure` are inert. Production builders carry no namespace,
plan, observation, allocation, or recording delegate, and panic text uses only the assertion prefix
and fixed property identity.

`invariant_noop` compiles the complete API as inert bodies solely for measuring assertion overhead.
Application code depends on assertion panics aborting control flow, so running any application in
this mode is undefined behavior. Combining `invariant_noop` with a production alias intentionally
fails compilation. Removed legacy assertion tags have no special meaning and select the ordinary
full implementation.

## Measured overhead

```text
Machine: Apple M4 (arm64)
  cores: 4 P + 6 E = 10 logical   freq: ?   ram: 16GiB   storage: 460GiB
  L1: 128KiB   L2: 16MiB   OS: macOS 26.2   kernel: Darwin 25.2.0

Benchmark 1 (20 runs, 65.4s): .local/tmp/sloc-full third_party
  measurement      mean ± σ              min ... max        outliers
  wall_time       3.27s ± 30.6ms       3.23s ... 3.32s        0 (0%)
  peak_rss      70.2MiB ± 1.80MiB    66.3MiB ... 73.2MiB      0 (0%)
  cpu_cycles      80.9G ± 851M         79.3G ... 82.2G        0 (0%)
  instructions     386G ± 119M          386G ... 386G         1 (5%)
  cpu_user        529ms ± 6.25ms       516ms ... 538ms        0 (0%)
  cpu_system     58.1ms ± 1.04ms      54.8ms ... 59.5ms       1 (5%)

Benchmark 2 (20 runs, 43.1s): .local/tmp/sloc-production third_party
  measurement      mean ± σ              min ... max        outliers  delta
  wall_time       2.15s ± 144ms        2.02s ... 2.69s       2 (10%)  - 34.2% ±  2.1%
  peak_rss      69.9MiB ± 2.20MiB    66.7MiB ... 73.8MiB      0 (0%)  -  0.4% ±  1.9%
  cpu_cycles      44.1G ± 715M         42.8G ... 46.3G       5 (25%)  - 45.5% ±  0.6%
  instructions     171G ± 432M          170G ... 172G         1 (5%)  - 55.7% ±  0.0%
  cpu_user        259ms ± 2.21ms       254ms ... 264ms        1 (5%)  - 50.9% ±  0.6%
  cpu_system     60.3ms ± 5.20ms      52.5ms ... 77.3ms      5 (25%)  +  3.7% ±  4.2%

Benchmark 3 (20 runs, 37.2s): .local/tmp/sloc-noop third_party
  measurement      mean ± σ              min ... max        outliers  delta
  wall_time       1.86s ± 82.0ms       1.77s ... 2.10s        1 (5%)  - 43.2% ±  1.2%
  peak_rss      70.2MiB ± 1.97MiB    66.0MiB ... 72.8MiB      0 (0%)  -  0.0% ±  1.7%
  cpu_cycles      36.3G ± 537M         35.1G ... 37.2G       5 (25%)  - 55.1% ±  0.6%
  instructions     125G ± 255M          125G ... 126G        3 (15%)  - 67.6% ±  0.0%
  cpu_user        201ms ± 2.04ms       197ms ... 204ms        0 (0%)  - 62.0% ±  0.6%
  cpu_system     59.1ms ± 3.23ms      52.1ms ... 65.7ms      3 (15%)  +  1.6% ±  2.7%
```
