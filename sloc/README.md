# sloc

Count the code, comment, and blank lines of a source tree.

## Benchmark

Against [tokei](https://github.com/XAMPPRocky/tokei), counting this repo. Benchmark a
compiled binary, not `go run ./sloc` — the `go` wrapper runs sloc as a grandchild the
sampler can't attribute, so its rss/cpu/instruction columns would measure the wrapper,
not the work:

```
$ go build -o .local/bin/sloc ./sloc
$ go run ./maddox "sloc" "tokei"
Machine: Apple M4 (arm64)
  cores: 4 P + 6 E = 10 logical   freq: ?   ram: 16GiB   storage: 460GiB
  L1: 128KiB   L2: 16MiB   OS: macOS 26.2   kernel: Darwin 25.2.0

Benchmark 1 (24 runs, 30.2s): sloc
  measurement      mean ± σ              min ... max        outliers
  wall_time       1.26s ± 73.0ms       1.18s ... 1.43s        0 (0%)
  peak_rss      77.1MiB ± 2.34MiB    73.4MiB ... 81.2MiB      0 (0%)
  cpu_cycles      15.2G ± 1.19G        12.7G ... 16.2G        0 (0%)
  instructions    44.0G ± 254M         43.4G ... 44.3G        0 (0%)
  cpu_user       48.8ms ± 1.80ms      44.5ms ... 50.8ms       1 (4%)
  cpu_system     56.3ms ± 8.06ms      39.5ms ... 63.3ms       0 (0%)

Benchmark 2 (18 runs, 30.7s): tokei
  measurement      mean ± σ              min ... max        outliers  delta
  wall_time       1.70s ± 57.3ms       1.63s ... 1.80s        0 (0%)  + 35.2% ±  3.3%
  peak_rss      85.3MiB ± 3.09MiB    80.7MiB ... 92.6MiB      0 (0%)  + 10.5% ±  2.2%
  cpu_cycles      21.9G ± 613M         21.0G ... 23.4G        0 (0%)  + 43.7% ±  4.1%
  instructions    69.7G ± 401M         69.3G ... 70.8G       3 (17%)  + 58.5% ±  0.5%
  cpu_user       90.1ms ± 2.09ms      88.1ms ... 96.0ms       0 (0%)  + 84.7% ±  2.5%
  cpu_system     58.1ms ± 5.20ms      51.5ms ... 70.2ms       0 (0%)  +  3.2% ±  7.8%
```
