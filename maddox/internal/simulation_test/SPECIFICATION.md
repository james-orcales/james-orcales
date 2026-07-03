
# Witness

The simulation drives internal.Main end to end so every maddox invariant is exercised through
the one public entry point. These leaves anchor the blackbox suite; the fuzz target then explores
the same driver, and the coverage recorder fails the run on any unwitnessed boundary.

### Pipeline

A full benchmark run — sampling a distribution, reducing it to statistics, comparing candidate to
reference, and rendering the report — completes without tripping an invariant, so the sampling,
statistics, comparison, and rendering paths are all driven from one Main call.

### Failure

A command that exits non-zero aborts the run through Main rather than reporting statistics for it,
driving the failure and captured-stderr paths that a successful run never reaches.
