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

A registered Range contains at least five legal values after distinct holes are removed. Use one
direct `Always` equality for one legal value. Use `Enum`, `Enum_3`, or `Enum_4` for two through four
legal values. Registration reports the legal count and the exact replacement. This rule does not
change unregistered runtime enforcement, production enforcement, or the public Range methods.

In the full build, Enum, Enum_3, and Enum_4 enforce their fixed member capacities at `Ensure`.
Members are statically resolvable, strictly ascending, and distinct. Registration seeds one
successful membership guard and gives each member an independent true/false axis.

## Type-owned helpers

A defined type keeps its contract beside the type in a trailing-namespace helper:

```go
const TOKEN_LENGTH = 16

type Token string

func Token_Invariants(token Token, namespace invariant.Namespace) {
	invariant.Always(len(token) == TOKEN_LENGTH, "The token length is valid.")
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

## Gap reports

Canonical `TestMain` callsites and assertion callsites import `shared/invariant/default`. Unset
`INVARIANT_OUTPUT` and `table` render dynamically aligned Markdown tables with section counts:

```text
🚨 1 coverage gaps 🚨

# Branch gaps (1)

| Assertion | Link | Missing | Property | Source |
|-----------|-----:|---------|----------|--------|
| queue     |    0 | false   | empty    | value  |

🚨 1 coverage gaps 🚨
```

`INVARIANT_OUTPUT=json` replaces the complete human report with one compact flat array and a
newline. Reachability records use `null` for `link` and `property`:

```json
[{"section":"reachability","assertion":"ready","link":null,"missing":"reachability",
"property":null,"source":"ok"}]
```

Any other environment value is a fatal configuration error. Both modes retain the same gap
collection, deterministic order, failure exit, clean summary, and fuzz-coverage merge behavior.

## Build modes

An untagged build retains the full deferred and recording-capable behavior above. Any of
`invariant_disable_coverage`, `prd`, `prod`, or `production` selects eager production enforcement.
Production aliases may be combined. `Always`, Range, holed Range, and every Enum capacity panic at
the violating call; `Sometimes` and `Ensure` are inert. Production builders carry no namespace,
plan, observation, allocation, or recording delegate, and panic text uses only the assertion prefix
and fixed property identity plus the offending value.

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

Benchmark 1 (20 runs, 67.0s): .local/tmp/sloc-full third_party
  measurement      mean ± σ              min ... max        outliers
  wall_time       3.35s ± 36.7ms       3.29s ... 3.42s        0 (0%)
  peak_rss      70.6MiB ± 3.05MiB    66.8MiB ... 80.3MiB      1 (5%)
  cpu_cycles      81.7G ± 780M         80.3G ... 83.0G        0 (0%)
  instructions     386G ± 191M          386G ... 387G        2 (10%)
  cpu_user        534ms ± 5.51ms       524ms ... 543ms        0 (0%)
  cpu_system     57.7ms ± 980us       56.2ms ... 60.0ms      2 (10%)

Benchmark 2 (20 runs, 42.1s): .local/tmp/sloc-production third_party
  measurement      mean ± σ              min ... max        outliers  delta
  wall_time       2.10s ± 83.1ms       2.03s ... 2.38s        1 (5%)  - 37.2% ±  1.2%
  peak_rss      69.6MiB ± 2.64MiB    64.7MiB ... 74.4MiB      0 (0%)  -  1.4% ±  2.6%
  cpu_cycles      44.0G ± 340M         43.5G ... 45.2G        1 (5%)  - 46.1% ±  0.5%
  instructions     171G ± 342M          171G ... 172G        2 (10%)  - 55.8% ±  0.0%
  cpu_user        259ms ± 1.43ms       253ms ... 260ms        1 (5%)  - 51.6% ±  0.5%
  cpu_system     58.2ms ± 2.28ms      55.2ms ... 66.3ms      2 (10%)  +  0.8% ±  2.0%

Benchmark 3 (20 runs, 37.7s): .local/tmp/sloc-noop third_party
  measurement      mean ± σ              min ... max        outliers  delta
  wall_time       1.89s ± 91.5ms       1.77s ... 2.06s        0 (0%)  - 43.6% ±  1.3%
  peak_rss      70.4MiB ± 2.36MiB    66.4MiB ... 73.8MiB      0 (0%)  -  0.2% ±  2.5%
  cpu_cycles      36.2G ± 717M         34.7G ... 37.6G       3 (15%)  - 55.7% ±  0.6%
  instructions     125G ± 453M          125G ... 126G         0 (0%)  - 67.5% ±  0.0%
  cpu_user        201ms ± 1.96ms       195ms ... 203ms       2 (10%)  - 62.4% ±  0.5%
  cpu_system     59.4ms ± 4.72ms      51.2ms ... 68.9ms      3 (15%)  +  2.9% ±  3.8%
```
