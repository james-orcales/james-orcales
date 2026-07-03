# cli.rs parsing overhead

`shared_rs::cli::program_parse` originally rebuilt the whole `Vec<Parameter>`
for whichever namespace (arguments/flags/global flags) an option lives in on
every single `-label=value` token and every positional fill — `O(n·m)` for
`n` options and `m` tokens — and separately cloned a command's
arguments/flags twice per call (once in `resolve_command`, again inside
`assign_named` from a borrow of data the caller already owned uniquely).

Two independent fixes for the `O(n·m)` rebuild were tried, both keeping the
redundant-clone fix (moving the already-owned `Command`'s fields into
`assign_named`/`assign_positionals` instead of re-cloning them) and the
public API unchanged:

- **batched**: named/positional assignments collect into a pending list
  during the token walk, applied to each namespace in one `HashMap`-driven
  pass at the end (`O(n+m)`).
- **arena**: a new `gen_arena::update` primitive (added to
  `shared_rs/src/gen_arena.rs`, whitelisted in `lint_rs`'s `MUT_ALLOWED`)
  lets `Named_State` hold each namespace as a `gen_arena::Arena<Parameter>`
  instead of a `Vec<Parameter>`; a named assignment calls `update` — an
  `O(1)` in-place overwrite through a stable handle — immediately, no
  deferred pending list at all. `Command`/`Program`'s public
  `Vec<Parameter>` fields are unchanged; the arena is purely an internal
  detail of `assign_named`, seeded via `insert` and drained back to a
  `Vec<Parameter>` at the boundary.

`shared_rs/examples/cli_bench.rs` is the harness: a synthetic command with
40 string flags, parsed 20,000 times per process against a token list
setting every flag by name plus one positional argument. Three release
binaries were built — original, batched, arena — and compared with
`maddox`, this repo's whole-binary benchmarking tool (see
`sloc_rs/BENCHMARK.md` for the same approach applied elsewhere).

```
$ maddox cli_bench_before cli_bench_after cli_bench_arena -warmup=3 -runs=30 -duration=20
Machine: Apple M4 (arm64)
  cores: 4 P + 6 E = 10 logical   freq: ?   ram: 16GiB   storage: 460GiB
  L1: 128KiB   L2: 16MiB   OS: macOS 26.2   kernel: Darwin 25.2.0

Benchmark 1 (30 runs, 13.9s): cli_bench_before (original)
  measurement      mean ± σ              min ... max        outliers
  wall_time       462ms ± 9.99ms       445ms ... 495ms        2 (7%)
  peak_rss      1.58MiB ± 19.9KiB    1.55MiB ... 1.63MiB      0 (0%)
  cpu_cycles      1.89G ± 23.1M        1.84G ... 1.97G        2 (7%)
  instructions    8.62G ± 2.19M        8.61G ... 8.62G        0 (0%)
  cpu_user       11.0ms ± 223us       10.6ms ... 11.8ms       2 (7%)
  cpu_system     33.8us ± 11.1us      22.1us ... 73.4us       2 (7%)

Benchmark 2 (30 runs, 10.5s): cli_bench_after (batched, HashMap)
  measurement      mean ± σ              min ... max        outliers  delta
  wall_time       351ms ± 3.35ms       346ms ... 359ms        0 (0%)  - 24.2% ±  0.8%
  peak_rss      2.12MiB ± 147KiB     1.97MiB ... 2.33MiB      0 (0%)  + 34.5% ±  3.4%
  cpu_cycles      1.41G ± 12.6M        1.39G ... 1.44G        0 (0%)  - 25.4% ±  0.5%
  instructions    7.48G ± 7.69M        7.47G ... 7.49G        0 (0%)  - 13.2% ±  0.0%
  cpu_user       8.37ms ± 83.0us      8.26ms ... 8.57ms       0 (0%)  - 24.2% ±  0.8%
  cpu_system     30.2us ± 5.51us      18.3us ... 41.8us       0 (0%)  - 10.6% ± 13.5%

Benchmark 3 (30 runs, 10.5s): cli_bench_arena (gen_arena::update, eager)
  measurement      mean ± σ              min ... max        outliers  delta
  wall_time       349ms ± 3.90ms       344ms ... 357ms        0 (0%)  - 24.4% ±  0.8%
  peak_rss      1.67MiB ± 35.8KiB    1.61MiB ... 1.75MiB      0 (0%)  +  6.0% ±  0.9%
  cpu_cycles      1.42G ± 16.0M        1.40G ... 1.46G        0 (0%)  - 24.6% ±  0.5%
  instructions    8.11G ± 3.32M        8.10G ... 8.11G        0 (0%)  -  5.9% ±  0.0%
  cpu_user       8.35ms ± 95.3us      8.22ms ... 8.53ms       0 (0%)  - 24.3% ±  0.8%
  cpu_system     23.2us ± 8.75us      16.0us ... 41.0us       0 (0%)  - 31.3% ± 15.4%
```

## Conclusion

Both fixes cut wall time and CPU cycles by roughly a quarter versus the
original — every one of those deltas clears maddox's 1% significance
threshold. Batched and arena are statistically **tied** on wall time (351ms
vs 349ms) and CPU cycles (1.41G vs 1.42G): eliminating the `O(n·m)` rebuild
is what mattered: *how* it's eliminated (deferred batch vs. eager
per-token arena update) made no measurable difference to speed here.

Where they differ is memory and instruction count:

- **Arena uses much less memory**: `peak_rss` rises only `+6.0%` over
  baseline, versus batched's `+34.5%`. The batched approach builds a
  `HashMap<usize, Parameter_Value>` per parse call; the arena is a plain
  `Vec<Slot<Parameter>>`, more compact and without hashing overhead.
- **Arena executes more instructions** (`8.11G` vs `7.48G`, `-5.9%` vs
  `-13.2%` off baseline) yet matches batched's wall time — those extra
  instructions are evidently cheaper per cycle (better cache locality/branch
  prediction than hash-bucket lookups), not a real cost.

Net: the arena route is at least as fast, uses less memory, and — as a
bonus not visible in these numbers — reads simpler in the source, since
`assign_named` goes back to the original one-`try_fold` shape with no
separate "apply pending updates" pass at all. The cost paid for that is
infrastructural: a new `gen_arena::update` primitive and a 4th entry in
`lint_rs`'s hardcoded `MUT_ALLOWED` whitelist — a small, one-time, reusable
addition to shared code, not a per-call runtime cost.

Caveat: parsing a real command line (a handful of tokens against a handful
of options, once per process) never spends measurably long in
`program_parse` regardless of which fix is used — the synthetic workload
above (40 flags × 20,000 iterations) exists to surface the algorithmic
difference above process-noise floor, not to model a realistic invocation.
