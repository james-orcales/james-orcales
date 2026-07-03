# cli.rs parsing overhead

`shared_rs::cli::program_parse` originally rebuilt the whole `Vec<Parameter>`
for whichever namespace (arguments/flags/global flags) an option lives in on
every single `-label=value` token and every positional fill — `O(n·m)` for
`n` options and `m` tokens — and separately cloned a command's
arguments/flags twice per call (once in `resolve_command`, again inside
`assign_named` from a borrow of data the caller already owned uniquely).

Three independent fixes for the `O(n·m)` rebuild were tried, all keeping the
redundant-clone fix (moving the already-owned `Command`'s fields into
`assign_named`/`assign_positionals` instead of re-cloning them) and the
public API unchanged:

- **batched**: named/positional assignments collect into a pending list
  during the token walk, applied to each namespace in one `HashMap`-driven
  pass at the end (`O(n+m)`).
- **arena (gen_arena)**: a new `gen_arena::update` primitive lets
  `Named_State` hold each namespace as a `gen_arena::Arena<Parameter>`
  instead of a `Vec<Parameter>`; a named assignment calls `update` — an
  `O(1)` in-place overwrite through a stable, generation-checked handle —
  immediately, no deferred pending list.
- **arena (plain)**: same idea, but `assign_named` never calls `remove`, so
  the generational safety `gen_arena` provides is never exercised — a new
  `arena::update` primitive was added to the plain, non-generational `arena`
  module instead (a bare `Vec<Parameter>`, no per-slot generation field, no
  `Entry::Free` variant to check).

All three keep `Command`/`Program`'s public `Vec<Parameter>` fields
unchanged; the arena (either kind) is purely an internal detail of
`assign_named`, seeded via `insert` and drained back to a `Vec<Parameter>`
at the boundary.

`shared_rs/examples/cli_bench.rs` is the harness: a synthetic command with
40 string flags, parsed 20,000 times per process against a token list
setting every flag by name plus one positional argument, compared with
`maddox`, this repo's whole-binary benchmarking tool (see
`sloc_rs/BENCHMARK.md` for the same approach applied elsewhere).

## Pass 1: all four against the original (30 runs each)

A first, low-sample pass — useful to confirm all three fixes beat the
original by a wide margin, but too noisy (±0.5–0.9 percentage points on the
deltas) to tell the three fixes apart from each other.

```
$ maddox cli_bench_before cli_bench_after cli_bench_arena cli_bench_arena_plain -warmup=3 -runs=30 -duration=25
Machine: Apple M4 (arm64)
  cores: 4 P + 6 E = 10 logical   freq: ?   ram: 16GiB   storage: 460GiB
  L1: 128KiB   L2: 16MiB   OS: macOS 26.2   kernel: Darwin 25.2.0

Benchmark 1 (30 runs, 13.4s): cli_bench_before (original)
  wall_time       447ms ± 5.64ms
  peak_rss      1.57MiB ± 18.3KiB
  cpu_cycles      1.89G ± 21.1M

Benchmark 2 (30 runs, 10.2s): cli_bench_after (batched, HashMap)
  wall_time       339ms ± 4.02ms   - 24.2% ±  0.6%
  peak_rss      2.18MiB ± 163KiB   + 38.2% ±  3.7%
  cpu_cycles      1.41G ± 12.2M    - 25.5% ±  0.5%

Benchmark 3 (30 runs, 10.4s): cli_bench_arena (gen_arena::update, eager)
  wall_time       347ms ± 4.45ms   - 22.4% ±  0.6%
  peak_rss      1.67MiB ± 27.6KiB  +  6.4% ±  0.8%
  cpu_cycles      1.43G ± 16.4M    - 24.3% ±  0.5%

Benchmark 4 (30 runs, 10.1s): cli_bench_arena_plain (arena::update, eager, no generation)
  wall_time       335ms ± 4.25ms   - 25.0% ±  0.6%
  peak_rss      1.69MiB ± 36.0KiB  +  7.6% ±  0.9%
  cpu_cycles      1.38G ± 15.6M    - 27.1% ±  0.5%
```

At this sample size, plain `arena` *looked* fastest outright (335ms vs
batched's 339ms and gen_arena's 347ms) — but those gaps are only a few ms on
means with several ms of σ each, i.e. within one another's noise. That read
doesn't survive more samples (below) — recorded here for the record, not as
the conclusion.

## Pass 2: direct pairwise comparisons (250 runs, ~85–90s budget each)

Two follow-up runs, sharing a reference command so `maddox` computes a
direct, confidence-interval-backed delta between the specific pair being
asked about, with ~8× the samples of pass 1.

**batched vs. both arena variants** (batched as reference):

```
$ maddox cli_bench_after cli_bench_arena cli_bench_arena_plain -warmup=5 -runs=250 -duration=90
Machine: Apple M4 (arm64)
  cores: 4 P + 6 E = 10 logical   freq: ?   ram: 16GiB   storage: 460GiB
  L1: 128KiB   L2: 16MiB   OS: macOS 26.2   kernel: Darwin 25.2.0

Benchmark 1 (250 runs, 85.3s): cli_bench_after (batched)
  wall_time       341ms ± 5.52ms
  peak_rss      2.14MiB ± 162KiB
  cpu_cycles      1.41G ± 19.3M

Benchmark 2 (250 runs, 87.3s): cli_bench_arena (gen_arena)
  wall_time       349ms ± 4.95ms   +  2.4% ±  0.3%
  peak_rss      1.67MiB ± 32.6KiB  - 22.1% ±  0.9%
  cpu_cycles      1.42G ± 19.9M    +  0.9% ±  0.2%

Benchmark 3 (250 runs, 85.1s): cli_bench_arena_plain (plain arena)
  wall_time       340ms ± 9.01ms   -  0.3% ±  0.4%
  peak_rss      1.70MiB ± 37.6KiB  - 20.8% ±  0.9%
  cpu_cycles      1.38G ± 27.1M    -  1.9% ±  0.3%
```

**gen_arena vs. plain arena directly** (gen_arena as reference):

```
$ maddox cli_bench_arena cli_bench_arena_plain -warmup=5 -runs=250 -duration=90
Machine: Apple M4 (arm64)
  cores: 4 P + 6 E = 10 logical   freq: ?   ram: 16GiB   storage: 460GiB
  L1: 128KiB   L2: 16MiB   OS: macOS 26.2   kernel: Darwin 25.2.0

Benchmark 1 (250 runs, 87.7s): cli_bench_arena (gen_arena)
  wall_time       351ms ± 6.01ms
  peak_rss      1.67MiB ± 31.8KiB
  cpu_cycles      1.43G ± 26.4M

Benchmark 2 (250 runs, 86.4s): cli_bench_arena_plain (plain arena)
  wall_time       346ms ± 5.17ms   -  1.4% ±  0.3%
  peak_rss      1.70MiB ± 39.8KiB  +  1.8% ±  0.4%
  cpu_cycles      1.38G ± 14.9M    -  3.2% ±  0.3%
  instructions    7.86G ± 2.70M    -  3.0% ±  0.0%
```

## Conclusion (superseding pass 1's read)

Every delta above whose confidence interval excludes zero is a real effect,
per maddox's own significance rule; the rest are noise, however large they
looked at 30 samples. With 250-sample confidence intervals:

- **All three fixes still beat the original by roughly a quarter** on wall
  time and CPU cycles (pass 1) — that headline result holds.
- **`gen_arena` is significantly *slower* than batched**: `+2.4% ± 0.3%`
  wall time, `+0.9% ± 0.2%` CPU cycles. Both intervals exclude zero. Pass 1's
  30-sample read called this "a close third" — that was noise; it's a real,
  if small, regression.
- **Plain `arena` is statistically *tied* with batched** on wall time
  (`-0.3% ± 0.4%`, interval includes zero) and `cpu_user` — not the outright
  win pass 1 suggested. It does show a small but real edge on raw
  `cpu_cycles` (`-1.9% ± 0.3%`).
- **Plain `arena` is significantly faster than `gen_arena`**, confirmed by
  the direct pairwise: `-1.4% ± 0.3%` wall time, `-3.2% ± 0.3%` CPU cycles,
  `-3.0% ± 0.0%` instructions — all intervals exclude zero. The generation
  check (`slot.generation == handle.generation`, `Entry::Free` branch) that
  `gen_arena` pays on every lookup is a real, measurable cost here, not a
  rounding error — `assign_named` never calls `remove`, so that check buys
  nothing in this code and can be skipped entirely.
- **Memory: both arena variants crush batched** (`gen_arena` `-22.1% ± 0.9%`,
  plain `arena` `-20.8% ± 0.9%` peak RSS vs. batched — both intervals
  exclude zero, both large). Between the two arena variants, `gen_arena`
  uses *very slightly* less memory than plain `arena` (`+1.8% ± 0.4%` for
  plain vs. gen_arena as reference — significant, but tiny, likely `Vec`
  growth/rounding rather than anything structural).

**Net, properly stated**: plain `arena` is at least as fast as batched
(tied on wall time, a hair better on CPU cycles), significantly faster than
`gen_arena`, and — tied with `gen_arena` — uses roughly a fifth less memory
than batched. It remains the best overall pick: not because it's the
outright fastest (it isn't, statistically, vs. batched), but because it
matches batched's speed while using much less memory and reading as simpler
code (`assign_named` is one plain `try_fold`, no `HashMap`, no generation
bookkeeping). The lesson from pass 1 vs. pass 2: a handful of runs can
produce a clean-looking ranking that a properly-powered comparison
overturns — worth the extra 3 minutes of wall-clock time before trusting a
"X is fastest" claim.

Caveat unchanged: parsing a real command line (a handful of tokens against a
handful of options, once per process) never spends measurably long in
`program_parse` regardless of which fix is used — the synthetic workload
above (40 flags × 20,000 iterations) exists to surface the algorithmic
difference above process-noise floor, not to model a realistic invocation.
