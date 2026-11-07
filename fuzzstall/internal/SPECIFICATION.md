
# Rate

### Quota

A rate is a quota of new interesting inputs and the window they must arrive in, written as
90/30s. A window written alone asks for one input, thus 30s and 1/30s name the same rate.

### Seconds

A window is a whole number of seconds and carries the suffix s. Seconds are the one unit
there is, thus 60s is the only way to write a minute.

### Malformed

An empty, suffixless, non-numeric, non-positive, or overflowing window is an error, and so
is any other unit or a quota outside its own bounds, so a mistyped rate never silently
becomes a different one.

# Command

### Split

The first argument is the rate. Every argument behind it belongs to `go test` and is taken
verbatim behind a `test` of this program's own, so the caller names the fuzz target with
`-fuzz` and this program reads no flag of its own.

### Absent

A command line carrying no rate, or a rate with no `go test` argument behind it, is a
usage error.

# Progress

### Count

A `go test -fuzz` progress line reports how many new interesting inputs the run found. The
number behind `new interesting: ` is that count, and it is the one field this program
reads.

### Other

A line that carries no such field — a baseline coverage line, a failure, or anything else
the run writes — leaves the record unchanged.

### Malformed

A field with no digit behind it, or a run of digits longer than an int64 counts, is read as
no count at all rather than as some other number.

# Line

### Split

Output arrives in chunks that start and stop at any byte, thus a line is assembled across
chunks and is complete only at its newline.

### Overflow

A line longer than the buffer keeps its first bytes and discards the remaining ones. The
field this program reads is near the start of a progress line, thus a bound that truncates
a long line still reads every count.

# Run

### Exit Code

A run that stops by itself yields its own exit code, thus a fuzz target that fails is
reported as a failure and never as saturation.

### Saturation

When a full window passes without the quota of new interesting inputs arriving, the run is
terminated and reports 124 with a line on standard error. The window starts again each time
the quota is met, thus a run that holds its rate is never terminated.

### Baseline

The window is measured from the first progress line, not from the start of the run.
Baseline coverage over a large corpus reports no count, and a wait for it is not
saturation.

### Start Failure

A `go test` that does not start at all reports 125 and writes the reason, which keeps this
program's own failure distinct from the run's.