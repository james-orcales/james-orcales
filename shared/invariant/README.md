# invariant

`shared/invariant` contains the pure assertion engine. `shared/invariant/default` supplies the
OS-backed recorder. It also supplies the package-level API for application code.

Application code and canonical `TestMain` functions import `shared/invariant/default`.
The complete API, registration rules, coverage rules, build modes, and output contract are in
[`SPECIFICATION.md`](SPECIFICATION.md).

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
