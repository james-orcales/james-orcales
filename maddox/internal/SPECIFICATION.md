
# Statistics

Measurement_Compute reduces one metric's per-run values to a distribution, the way
poop summarizes a benchmark.

### Distribution

The mean, sample standard deviation, extrema, median, and quartiles are computed
over the values, the quartiles by position.

### Outliers

A value past Tukey's fences, one and a half interquartile ranges beyond a quartile,
is counted as an outlier.

# Comparison

Compare reports how a candidate command's metric differs from the reference's, with
a confidence interval.

### Reference

A measurement compared against itself yields a zero difference that is never
significant.

### Significance

A difference is significant only when its confidence interval clears one percent
with a single sign.

# Sampling

Main runs each command repeatedly until the time budget or the run cap is reached,
whichever comes first.

### Budget

A non-zero time budget stops sampling once it is spent; a zero budget disables it,
leaving the run cap in charge.

### Runs

A non-zero run cap stops sampling once that many runs are kept; a zero cap disables
it, leaving the time budget in charge.

### Minimum

A command is sampled at least three times regardless of the limits, so the
statistics keep a quorum.

### Warmup

The configured warmup runs are taken and discarded before sampling, absent from the
reported run count.

# Output

Main writes a JSON report, one entry per command in the order given.

### Document

The first command is the reference and carries no deltas; every later command
carries deltas against it.

### Failure

A command that exits non-zero aborts the run with a non-zero status and its stderr
surfaced, unless failures are allowed.

# Table

Render_Table renders the report as poop's aligned, optionally colored comparison
table — the default human-readable output.

### Header

Each benchmark is named with its position, its kept-run count, its elapsed time, and
its command words.

### Units

A raw value is scaled to a human unit by magnitude — nanoseconds to milliseconds,
bytes to binary KiB and MiB, counts to millions.

### Delta

A non-reference benchmark's row carries its signed percentage change against the
reference.

### Color

ANSI color marks only the delta column, and only when color is enabled — never in
piped output.

### Sparse

A metric with no data — every value zero, like the Linux-only counters on macOS — is
left out of the table.

# Machine

Machine_Specs carries the host hardware and OS taken once at startup, so benchmark
results are reproducible across machines. The specs are injected through Main_Input,
keeping the library free of ambient OS reads.

### Document

The JSON document carries Machine_Specs as a top-level "machine" field alongside
"benchmarks". Every field round-trips through JSON; optional fields (P/E cores,
cache sizes) are omitted when zero.

### Table

Render_Table writes the machine header block before Benchmark 1. Sparse fields
(zero values) are omitted from the header, matching the sparse-metric convention.

# Progress

Throughout warmup and sampling, Main writes an in-place counter and elapsed time to
the diagnostic sink — only when progress is enabled, so piped output stays clean.
