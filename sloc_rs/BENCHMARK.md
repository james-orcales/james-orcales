~/code/james-orcales git:(acd82277c)
$ maddox "sloc" "sloc_ocaml" ".local/share/rust/target/release/sloc_rs" -warmup=3
Machine: Apple M4 (arm64)
  cores: 4 P + 6 E = 10 logical   freq: ?   ram: 16GiB   storage: 460GiB
  L1: 128KiB   L2: 16MiB   OS: macOS 26.2   kernel: Darwin 25.2.0

Benchmark 1 (26 runs, 30.4s): sloc
  measurement      mean ± σ              min ... max        outliers
  wall_time       1.17s ± 38.7ms       1.14s ... 1.35s        1 (4%)
  peak_rss      84.7MiB ± 7.09MiB    73.0MiB ... 111MiB       1 (4%)
  cpu_cycles      21.6G ± 955M         19.8G ... 25.3G       3 (12%)
  instructions    41.7G ± 217M         41.5G ... 42.7G        1 (4%)
  cpu_user       50.5ms ± 302us       49.6ms ... 50.9ms       1 (4%)
  cpu_system      109ms ± 7.22ms      94.5ms ... 136ms       3 (12%)

Benchmark 2 (20 runs, 31.3s): sloc_ocaml
  measurement      mean ± σ              min ... max        outliers  delta
  wall_time       1.57s ± 52.0ms       1.44s ... 1.67s       2 (10%)  + 34.1% ±  2.3%
  peak_rss      99.7MiB ± 2.78MiB    94.8MiB ... 104MiB       0 (0%)  + 17.7% ±  4.0%
  cpu_cycles      24.4G ± 1.05G        22.5G ... 26.5G        0 (0%)  + 12.8% ±  2.8%
  instructions     106G ± 5.97G        91.7G ... 119G        4 (20%)  +154.6% ±  5.7%
  cpu_user        129ms ± 8.76ms       105ms ... 146ms       4 (20%)  +155.2% ±  6.9%
  cpu_system     48.9ms ± 3.61ms      44.8ms ... 57.3ms      2 (10%)  - 55.0% ±  3.3%

Benchmark 3 (28 runs, 30.3s): .local/share/rust/target/release/sloc_rs
  measurement      mean ± σ              min ... max        outliers  delta
  wall_time       1.08s ± 19.5ms       1.06s ... 1.15s        2 (7%)  -  7.4% ±  1.4%
  peak_rss      93.1MiB ± 2.08MiB    88.9MiB ... 97.7MiB      0 (0%)  + 10.0% ±  3.3%
  cpu_cycles      19.6G ± 394M         18.9G ... 20.4G        0 (0%)  -  9.5% ±  1.8%
  instructions    39.2G ± 47.2M        39.1G ... 39.4G        2 (7%)  -  6.1% ±  0.2%
  cpu_user       66.2ms ± 172us       65.9ms ... 66.5ms       1 (4%)  + 31.3% ±  0.3%
  cpu_system     76.2ms ± 2.95ms      70.7ms ... 82.4ms       0 (0%)  - 29.9% ±  2.7%

## Binary size

Same program, release/native builds (arm64 macOS):

| binary             | size    | bytes     | notes                               |
| ------------------ | ------- | --------- | ----------------------------------- |
| `sloc` (Go)        | 3.4 MiB | 3,511,714 | default `go build`                  |
| `sloc` (Go)        | 2.3 MiB | 2,384,738 | stripped, `-ldflags="-s -w"`        |
| `sloc_ocaml`       | 1.4 MiB | 1,501,288 | `ocamlopt` native, unstripped       |
| `sloc_rs`          | 619 KiB |   633,424 | `cargo build --release` (`strip = true`) |
