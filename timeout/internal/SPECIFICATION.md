
# Duration

### Units

A magnitude carries one of ns, us, ms, s, or m and scales to that many nanoseconds. A
two-letter unit is read whole, so 5ms is five milliseconds rather than five minutes
trailed by a stray letter.

### Malformed

An empty, unitless, non-numeric, unknown-unit, non-positive, or overflowing deadline is an
error, so a mistyped span never silently becomes a different one.

# Command

### Split

The first argument is the deadline and every argument behind it is the command, taken
verbatim: a child's own flags stay the child's and are never read as this program's.

### Absent

A command line carrying no deadline, or a deadline with no command behind it, is a usage
error.

# Run

### Exit Code

A child that finishes inside its deadline yields its own exit code, so the wrapper is
transparent to the command it fences.

### Timeout

A child still alive at its deadline is terminated and the run reports 124 — GNU timeout's
code — with a line on standard error, so a killed child is never read as a failed one.

### Start Failure

A command that cannot be started at all reports 125 and writes the reason, keeping the
wrapper's own failure distinct from the child's.
