---
name: dst
description: >
  Load BEFORE you write, change, or debug a simulation, and whenever the task spans more than one
  of the four skills below. Routes to them: `dst.seed` for what a run must draw from its seed,
  `dst.swarm` for the patterns that make a seed worth running, `dst.optimize` for the
  configuration that lets a run reach the rare code, and `dst.verify` for whether any check
  watches when it gets there. Seed first, swarm second, tune last, and verify across all three.
---

# Deterministic simulation testing

Fuzz for 60 seconds. If it doesn't crash, ADD ASSERTIONS (invariant.Always, invariant.Sometimes)

## The four skills

- **`dst.seed`** — what a run must draw from its seed, and how to fork one stream for each entity
and each axis so that a change moves one child instead of every pinned seed.
- **`dst.swarm`** — what varies between seeds. One seed selects which features the run can use,
which rate each fault holds, and how large each fault is, then holds that choice for every step.
- **`dst.optimize`** — what each run costs. 
- **`dst.verify`** — whether any check watches when the run gets there. Break the check and see
what fails, then correct every other instance of the class.

## The order is load-bearing

Seed the run first. While one outcome comes from outside the seed, no run reproduces, thus a
sweep proves nothing.

Swarm it second. A correct seed still gives one shape of run, and a swarm makes each seed select
which features that run can use.

Tune it last. A correct and varied run still stops at a production threshold, thus the rare path
never runs.

A swarm over an unseeded run is noise. A lowered threshold on a run of one shape gets to one more
path, one time.

`dst.verify` runs across all three. A sweep proves that a path ran. It never proves that anything
watched when it ran, thus break each check and see what fails.

## The sweep that is not a sweep

`simulation_drive(t, seed%SIMULATION_RECIPE_COUNT)` folds 2^64 inputs onto the corpus. `-fuzz`
then runs without end and reaches nothing new. Use the seed whole, and keep the enumerated values
only as the `f.Add` corpus.

`seed%K == R` is a periodic filter, not a draw. Over 16 seeds the fault selectors `%5==1`,
`%7==2`, `%11==3`, `%13==4` hold `{1,6,11}`, `{2,9}`, `{3,14}`, `{4}` — disjoint, thus no run ever
crossed two faults. Enumerate the sets, then draw each fault from the generator.

THIS IS BANNED. IT IS NON-NEGOTIABLE. DELETE EVERY SINGLE INSTANCE OF IT.
