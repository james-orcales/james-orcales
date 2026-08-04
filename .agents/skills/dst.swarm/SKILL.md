---
name: dst.swarm
description: >
  Load BEFORE you write or change which features a simulation run can use. Swarm testing
  increases the quality of a seed: the seed selects a random subset of the features, then holds
  that subset for every step. A feature that is always active suppresses the behavior below it
  and takes step count from the features that remain. Coverage then comes from the sweep, never
  from one run.
---

# Swarm testing

Swarm testing refers to seeding and fault-injection patterns that produce high-value simulations.

## Disabling features

First find the features that a run can deactivate. A run that can do less gets deeper than a run
that can do everything. The list of what to remove is worth more than the list of what to add.

One run is one timeline, and each happy path and each failure mode must use the same one. A step
that sends a normal operation is a step that adds no fault. Thus each feature that stays active
takes depth from every other feature.

The features first divide the steps. Each of 20 active features gets a twentieth of them, thus no
feature gets deep enough to reach the state that holds the bug.

A feature also prevents the behavior of another. A push and pop driver at 50/50 must do about
370,000 runs to find a 32-slot stack overflow, because `pop` is what prevents the overflow.
Remove `pop`, and about one run in three finds it.

A subset makes nothing new possible. Only the distribution changes.

## Giving the run a fault-free window

Set each fault probability to zero for a window, and let the system converge.
`transition_to_liveness_mode` heals the network and stops each fault. Safety holds while faults
land, but liveness needs a window without them, thus a model with no such window states only
half of the specification.

## Drawing the rates per run

Draw each fault rate per run inside a declared bound. A rate that a constant states gives every
seed one model, thus the sweep gets wide in its schedule and stays narrow in its faults.
TigerBeetle's `vopr.zig` draws each one. Its `packet_loss_probability` is
`ratio(prng.int_inclusive(u8, 30), 100)`, so one run loses no packet and another loses three in
ten. A zero draw deactivates that fault for the whole run, thus an active-and-inactive subset is
the special case of a drawn rate.

## Biasing the roll toward the damaging moment

Raise the probability of a fault when the state makes that fault hurt. TigerBeetle's
`tick_crash_up` multiplies the crash numerator by 10 when that replica has writes in flight. A
model that keeps one rate spends most of its faults on the quiet steps.

## Varying the granularity of a fault

Draw how large a fault is, not only where it lands. TigerBeetle corrupted whole sectors, thus the
checksum always failed and the repair path always ran, and the assertion below it never ran.
Jepsen found the bug with single-bit corruption. One granularity prevents whatever sits below it.

## Coverage comes from the sweep, never from one run

Each run is narrower, so one run covers less than it did. Do not answer a coverage gap with a
configuration that activates every feature — that is the distribution that kept the bug unseen.
Widen the sweep, or add a seed whose configuration names the absent feature.

# Examples

- **Clock** — the skew model: linear drift, periodic, step, and non-ideal. `shared/time` builds
`SKEW_KIND_LINEAR`, `SKEW_KIND_PERIODIC`, and `SKEW_KIND_STEP`, and it lacks non-ideal.
- **Buggify** — FoundationDB's term for a fault that the binary itself emits: an unnecessary
error from a call that usually succeeds, a delay in an operation that is usually fast, or an
unusual tuning value.
