---
name: dst.optimize
description: >
  Load BEFORE you tune a simulation that runs too slow, finds too little, or reports a coverage
  gap that no seed can close. A production configuration makes a poor simulation, because it
  holds each rare operation behind a threshold that a test never reaches. Move the threshold,
  never grow the input. A `Sometimes` that never fires names the next threshold to move.
---

# Optimizing a simulation

Production is configured to stay stable. A simulation is configured to find bugs quickly, thus
the two configurations must differ.

## Move the threshold, never grow the input

A rare operation stays rare because a threshold holds it back. Lower the threshold instead of
growing the input that must climb it:

- Run each periodic task more often. A compaction that production does every 48 hours runs every
minute in a test.
- Lower each failure-detection value to the timescale of the injected faults, or the failover
never runs.
- Lower the volume that triggers a data move. Split a shard at one kilobyte, not one terabyte.
- Keep the data small. Never lift a production dataset into a test.

## Let a coverage gap name the next change

A `Sometimes` that never records its true branch names a threshold that no run reaches, and
`invariant.Run_Test_Main` reports each one at the end of the suite. Read that report as a list of
configuration changes, not only as a list of absent tests.

## Keep each run cheap

- Cap the memory. A tight cap makes an allocation error surface instead of hide.
- Keep the logs small. A verbose level in every run buries the one run that matters.
- Watch the simulated time against the wall time. A busy-wait loop stops the simulator from
advancing, thus the run spends wall time and gains no simulated time.
- Shrink the simulated durations to make a test faster. Never jump to the next event, because the
idle grains are where a fault lands.
- Bound each `Run_Until` with a finite timeout
