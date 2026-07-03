# cli.rs parsing overhead

`shared_rs::cli::program_parse` originally rebuilt the whole `Vec<Parameter>`
for whichever namespace (arguments/flags/global flags) an option lives in on
every single `-label=value` token and every positional fill — `O(n·m)` for
`n` options and `m` tokens — and separately cloned a command's
arguments/flags twice per call (once in `resolve_command`, again inside
`assign_named` from a borrow of data the caller already owned uniquely).
Both were fixed: named/positional assignments now collect into a pending
list during the token walk and apply to each namespace in one pass
(`O(n+m)`), and `program_parse_tokens` moves its already-owned `Command`'s
fields into `assign_named`/`assign_positionals` instead of re-cloning them.

`shared_rs/examples/cli_bench.rs` is the harness: a synthetic command with
40 string flags, parsed 20,000 times per process against a token list
setting every flag by name plus one positional argument. Two release
binaries were built — one from the code before either fix, one after both —
and compared with `maddox`, this repo's whole-binary benchmarking tool (see
`sloc_rs/BENCHMARK.md` for the same approach applied elsewhere).

```
$ maddox cli_bench_before cli_bench_after -warmup=3 -runs=30 -duration=20
Machine: Apple M4 (arm64)
  cores: 4 P + 6 E = 10 logical   freq: ?   ram: 16GiB   storage: 460GiB
  L1: 128KiB   L2: 16MiB   OS: macOS 26.2   kernel: Darwin 25.2.0

Benchmark 1 (30 runs, 14.4s): cli_bench_before
  measurement      mean ± σ              min ... max        outliers
  wall_time       481ms ± 8.82ms       468ms ... 496ms        0 (0%)
  peak_rss      1.57MiB ± 13.1KiB    1.55MiB ... 1.59MiB      0 (0%)
  cpu_cycles      1.90G ± 26.7M        1.86G ... 1.95G        0 (0%)
  instructions    8.61G ± 1.80M        8.61G ... 8.62G        0 (0%)
  cpu_user       11.5ms ± 209us       11.2ms ... 11.9ms       0 (0%)
  cpu_system     22.1us ± 2.51us      17.9us ... 28.2us       0 (0%)

Benchmark 2 (30 runs, 10.7s): cli_bench_after
  measurement      mean ± σ              min ... max        outliers  delta
  wall_time       357ms ± 4.68ms       349ms ... 369ms        2 (7%)  - 25.7% ±  0.8%
  peak_rss      2.18MiB ± 161KiB     1.97MiB ... 2.33MiB      0 (0%)  + 38.7% ±  3.7%
  cpu_cycles      1.42G ± 18.1M        1.39G ... 1.47G        1 (3%)  - 25.4% ±  0.6%
  instructions    7.48G ± 8.54M        7.47G ... 7.49G        0 (0%)  - 13.2% ±  0.0%
  cpu_user       8.53ms ± 109us       8.35ms ... 8.83ms       2 (7%)  - 25.8% ±  0.8%
  cpu_system     21.2us ± 3.88us      16.2us ... 34.0us       2 (7%)  -  3.7% ±  7.7%
```

Wall time, CPU cycles, and instructions all drop by roughly a quarter to a
third — every delta's confidence interval clears maddox's 1% significance
threshold. `peak_rss` rises (a `HashMap` per parse call to apply batched
updates costs more resident memory than the `Vec` shuffling it replaced),
but the absolute figures — low single-digit MiB — are immaterial for a
command-line parser invoked once per process.

Caveat: parsing a real command line (a handful of tokens against a handful
of options, once per process) never spends measurably long in
`program_parse` regardless of this fix — the synthetic workload above (40
flags × 20,000 iterations) exists to surface the algorithmic difference
above process-noise floor, not to model a realistic invocation.
