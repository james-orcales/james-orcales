# invariant

`shared/invariant` contains the pure assertion engine. `shared/invariant/default` supplies the
OS-backed recorder. It also supplies the package-level API for application code.

## Eager guards

`Always(condition, message)` evaluates its condition immediately. A false condition causes a panic
at the callsite. Registration records the literal message. Thus, an unreached guard is a coverage
gap.

## Deferred assertion chains

Use one fluent expression for an assertion set:

```go
invariant.Assertions("queue").
    Sometimes(queue.Empty(), "The queue is empty.").
    Sometimes(queue.Full(), "The queue is full.").
    Ensure()
```

`Sometimes` stores one Boolean condition and one fluent ordinal. It does not record coverage or a
failure. `Ensure` gets the registered plan. Then, `Ensure` records one true or false obligation.
Namespace, ordinal, and message form the identity. Thus, duplicate messages do not conflict.

The framework does not make a cross product. The suite supplies evidence for each branch. It does
not supply evidence for each combination. There are no tuple cells, carves, polarity references,
combination gaps, or legends.

## Integer domains

Each integer width has chain methods for bounded and enumerated domains:

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

A full build enforces the two Range bounds at `Ensure`. Range_Holed also enforces fixed holes.
The holes must be strictly inside the bounds. A signed method has four hole slots. An unsigned
method has three hole slots. Put distinct holes in ascending order. Fill unused slots with the final
hole. A boundary cannot be a hole.

Coverage includes the successful lower and upper guards. It also includes the two distinct bounds.
Eligible interior witnesses are `0`, `1`, `2`, and `-1`.

A registered Range must contain at least five legal values after the removal of distinct holes. Use
one direct `Always` equality for one legal value. Use `Enum`, `Enum_3`, or `Enum_4` for two to four
legal values. Registration reports the legal count and the correct replacement.

This cardinality rule applies only during registration. It does not change unregistered runtime
enforcement, production enforcement, or the public Range methods.

A full build enforces Enum membership at `Ensure`. Enum, Enum_3, and Enum_4 have fixed capacities.
Each member must be statically resolvable, distinct, and in ascending order. Registration records
one successful membership guard. Registration also gives each member a separate Boolean axis.

## Type-owned helpers

A defined type keeps its contract beside the type. The helper has a final namespace parameter:

```go
const TOKEN_LENGTH = 16

type Token string

func Token_Invariants(token Token, namespace invariant.Namespace) {
	invariant.Always(len(token) == TOKEN_LENGTH, "The token length is valid.")
}
```

Registration makes one template instance for each literal callsite namespace. A chain is one
nonempty call nest that ends in `Ensure`. Do not split the chain. Do not return an incomplete
builder.

## Coverage lifecycle

Registration parses the selected packages before the suite starts. It records each obligation.
Registration also publishes immutable handles in their source order. Runtime code uses these
handles. Runtime code does not calculate an identity from a message.

`Ensure` examines the full plan before it records coverage. If `Ensure` finds a failure, it
records no coverage.

Tests and fuzz processes record coverage. Benchmarks and ordinary binaries only enforce assertions.
Ordinary binaries do not read registration plans.

## Gap reports

Canonical `TestMain` and assertion callsites import `shared/invariant/default`. An unset
`INVARIANT_OUTPUT` value selects `table`. The `table` value writes aligned Markdown tables and
section counts:

```text
🚨 1 coverage gaps 🚨

# Branch gaps (1)

| Assertion | Link | Missing | Property | Source |
|-----------|-----:|---------|----------|--------|
| queue     |    0 | false   | empty    | value  |

🚨 1 coverage gaps 🚨
```

`INVARIANT_OUTPUT=json` writes one compact flat array and one newline. It does not write the human
report. A reachability record uses `null` for `link` and `property`:

```json
[{"section":"reachability","assertion":"ready","link":null,"missing":"reachability",
"property":null,"source":"ok"}]
```

All other environment values cause a fatal configuration error. The two modes use the same gap
collection and deterministic order. They also use the same failure exit, clean summary, and fuzz
coverage merge.

## Build modes

An untagged build uses deferred enforcement and can record coverage. Each production tag selects
eager enforcement. The tags are `invariant_disable_coverage`, `prd`, `prod`, and `production`. You
can combine production tags.

In a production build, `Always`, Range, Range_Holed, and each Enum method panic at the incorrect
link. `Sometimes` and `Ensure` are inert. A production builder has no namespace, plan, observation,
allocation, or recorder function. Panic text contains the assertion prefix, property, and incorrect
value.

`invariant_noop` compiles the full API as inert functions. Use this mode only to measure assertion
overhead. Application code can depend on assertion panics to stop control flow. Thus, application
behavior in this mode is undefined.

The combination of `invariant_noop` and a production tag fails compilation. Removed legacy tags
have no special meaning. They select the ordinary full implementation.

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
