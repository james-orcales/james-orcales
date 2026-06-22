# sloc_ocaml

`sloc_ocaml` is a faithful, byte-for-byte OCaml port of the Go [`sloc`](../sloc) source-line
counter: a generic per-line scanner partitions every physical line into code / comment / blank; ~70
language configs drive the one scanner; a tree walk classifies recognized files across Domains; a
renderer prints an aligned table (or JSON). Same classification rules, same languages, same
detection, same table and JSON output down to the byte. It is a twin of the Go `sloc`, expressed in
OCaml using strictly the standard distribution.

It derives, like the Go original, from [tokei](https://github.com/XAMPPRocky/tokei) — see
`LICENSE.mit.tokei`.

## Two files

- **`sloc.ml`** — the whole program: the scanner, the language table, detection, `count`, `render`
  (table / `--files` / JSON), `main`, and the composition root that binds the real filesystem and
  `git ls-files`.
- **`sloc_test.ml`** — the black-box test suite. It links `sloc.ml` and runs as its own binary, so
  verifying the tests never bootstraps anything; its name keeps `sloc.ml`'s guarded entry from
  running `main`.

No interface files, no build script, no third-party packages: only `Stdlib` + the bundled `Unix`
library, plus OCaml 5's `Domain` (in the standard library) for the parallel walk.

## Design

The code mirrors the Go program's pure-library + composition-root split, as labelled sections in the
one file:

- **Scanner.** One mutable `scan_state` per file: the block-comment depth and the verbatim,
  long-comment, and heredoc closers cross line boundaries; the `line_kind` verdict resets each line.
  A generic `classify_line` scans `source` in place by offset (no per-line copy), driven by a
  language's comment/string tokens — nesting block comments, raw and hash-counted strings, heredocs,
  Lua long brackets, the char-vs-lifetime tick. The hot path is allocation-free: a bulk skip over
  ordinary code bytes, then openers returning `int option` (`Some next` / `None`), allocating only
  when an opener actually fires.
- **Count.** A `tree` seam (`read_dir` lexically sorted, `read_file`) replaces Go's `fs.FS`; the
  walk prunes hidden and ignored directories, and classification fans out across `Domain`s over
  disjoint result slots — byte-identical to a sequential run, since results are written by index and
  read in walk order, so no lock is needed.
- **Render.** Category grouping, source/test split, thousands separators, %Code shares, and a
  hand-written JSON encoder matching Go's `encoding/json` (2-space indent, field order, HTML
  escaping).
- **Composition root.** The one place that binds the real OS: a `Sys.readdir`/`Unix.lstat` tree, a
  bounded reader, and a `git ls-files` ignore predicate. `Unix.lstat` (not `stat`) leaves symlinked
  directories undescended, matching `fs.WalkDir`.

**Errors are values, never exceptions.** The pure tier — scanner through `render`/`count` — contains
no `raise`, `try`, or `exception`. The only exception handling is the OS boundary `os_result`, where
OCaml's file/process APIs report failure solely by raising, converted there into the `result` the
program consumes. Outcomes are typed, not signalled by sentinels: an opener that may or may not
apply returns `option` (`Some next` / `None`), and `result` is reserved for the boundary, where
there is an actual error to carry.

`SPECIFICATION.md` describes the behavior of each entry; `sloc_test.ml` verifies it as a black box
over the public API with in-memory trees, touching nothing real.

## Build, test, run

```
# Compile the program.
ocamlopt -I +unix unix.cmxa sloc.ml -o sloc_ocaml

# Compile and run the black-box suite (its own binary; in-memory only).
ocamlopt -I +unix -I . unix.cmxa sloc.ml sloc_test.ml -o sloc_test && ./sloc_test

# Count a tree (read-only).
./sloc_ocaml [paths...] [--files] [--no-ignore] [--hidden] [--json]
```

`sloc.ml`'s entry runs `main` only when the binary is *not* named `sloc_test`, so the test binary
links the same definitions without counting anything at startup.

This port was verified byte-for-byte against the Go `./sloc` across this entire repository (~44k
files) for every flag combination, including non-ASCII paths.

## Prerequisites

- OCaml ≥ 5 with the native compiler `ocamlopt`, the bundled `Unix` library, and `Domain` (standard
  in OCaml 5). The OCaml toolchain is assumed present rather than vendored — this is a faithful twin
  and exercise.
- `git` on `PATH` for the gitignore-aware default; without it (or outside a git tree) nothing is
  ignored.

## Benchmarking

  ~/code/james-orcales git:(9d2d1bea7)
  $ go run ./maddox "sloc" "./sloc_ocaml/sloc_ocaml.v2" "./sloc_ocaml/sloc_ocaml.v2.unsafe_flag"
  Machine: Apple M4 (arm64)
    cores: 4 P + 6 E = 10 logical   freq: ?   ram: 16GiB   storage: 460GiB
    L1: 128KiB   L2: 16MiB   OS: macOS 26.2   kernel: Darwin 25.2.0

  Benchmark 1 (27 runs, 31.1s): sloc
    measurement      mean ± σ              min ... max        outliers
    wall_time       1.15s ± 8.70ms       1.14s ... 1.18s        1 (4%)
    peak_rss      83.3MiB ± 8.02MiB    73.0MiB ... 104MiB       0 (0%)
    cpu_cycles      21.6G ± 458M         20.4G ... 22.2G        0 (0%)
    instructions    41.6G ± 136M         41.3G ... 41.8G        0 (0%)
    cpu_user       50.3ms ± 261us       49.6ms ... 50.9ms       1 (4%)
    cpu_system      109ms ± 3.51ms      99.9ms ... 114ms        0 (0%)

  Benchmark 2 (20 runs, 30.8s): ./sloc_ocaml/sloc_ocaml
    measurement      mean ± σ              min ... max        outliers  delta
    wall_time       1.54s ± 55.3ms       1.46s ... 1.66s        0 (0%)  + 33.9% ±  1.9%
    peak_rss      96.5MiB ± 3.70MiB    90.1MiB ... 105MiB      2 (10%)  + 15.9% ±  4.7%
    cpu_cycles      24.4G ± 1.43G        22.7G ... 28.2G        0 (0%)  + 13.4% ±  2.7%
    instructions     107G ± 7.74G        97.2G ... 127G         0 (0%)  +156.3% ±  7.2%
    cpu_user        131ms ± 11.1ms       118ms ... 160ms        0 (0%)  +161.5% ±  8.6%
    cpu_system     46.9ms ± 1.09ms      45.4ms ... 50.2ms       1 (5%)  - 56.8% ±  1.5%

  Benchmark 3 (22 runs, 31.6s): ./sloc_ocaml/sloc_ocaml.unsafe_flag
    measurement      mean ± σ              min ... max        outliers  delta
    wall_time       1.44s ± 46.0ms       1.36s ... 1.59s        1 (5%)  + 24.7% ±  1.6%
    peak_rss      96.4MiB ± 3.16MiB    91.6MiB ... 103MiB       0 (0%)  + 15.7% ±  4.4%
    cpu_cycles      22.2G ± 886M         20.4G ... 23.8G        0 (0%)  +  3.1% ±  1.8%
    instructions    85.8G ± 5.19G        75.0G ... 95.5G        0 (0%)  +106.4% ±  4.8%
    cpu_user        110ms ± 7.14ms      94.5ms ... 122ms        0 (0%)  +118.6% ±  5.5%
    cpu_system     52.6ms ± 1.25ms      49.6ms ... 55.0ms       1 (5%)  - 51.5% ±  1.5%
