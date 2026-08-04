---
name: dst.seed
description: >
  Load BEFORE you write or change code that takes a seed — a simulation harness, a fuzz driver,
  or a fault model. Also load it when a pinned seed does not reproduce its run. Each value that
  the run does not receive as an input comes from one generator seeded off the run seed, and the
  harness is not exempt. Give each new axis its own stream. Do not add a draw inside an existing
  stream.
---

# Seeding

## What you must seed

Each value that the run does not receive as an input:

- **Payloads** — the bytes of each request, key, value, and message. Seed the content, not the
width alone. A run that seeds only the width gives the same shape of body each time.
- **Time** — each clock read, and each entity's own offset, drift, and skew.
- **IO outcomes** — the latency, which connect fails, each short read and write, each exit code.
- **Crash faults** — stop a process and start it again, at machine, rack, or datacenter
granularity. Also whether the restart reformats.
- **Network faults** — the partition, symmetric and asymmetric, the drop, the delivery out of
order, the duplicate, the replay, the latency, and the clog. Most models leave the duplicate out.
Here it found a remote panic on the first sweep.
- **Disk faults** — the corruption of an unsynced write at reboot, the misdirected write, the bit
flip, the read error, the write error, the full device, and the degradation.
- **Order** — which of two ready operations completes first, and each shuffle.
- **Workload** — which operation each step sends, and the fixture that the run walks.
- **Configuration** — the cluster size, the worker count, each tuning knob, the active features.
- **Identity** — each identifier, nonce, and token that the run makes.

Draw each probability per run. A probability that a constant states gives every seed one model.

The harness is not exempt. It draws which fixture to change, which corpus order to walk, and
which subset to activate.

## Where each value comes from

`prng.New(seed)` and `io.New_Sim(seed)` are the only sources. To change an outcome, change the
seed.

## Why you must fork a stream

One prng generator is one sequence, and each draw takes the next value in it. Thus a draw that you
add, remove, or move changes each value after it. One stream for the whole run couples every part of
the run to every other part, and a change anywhere then moves each pinned seed. (invalidating the
entire seed history)

`Generator_Split(&parent)` is `New(Generator_Next(&parent))`: one draw off the parent, made into
an independent child, thus a draw in a child moves no other stream. Fork one stream for each
entity and each axis at construction, and put each new fork after the last one
(`prng.New(seed ^ SALT)` for an axis that must never move). A draw added to the root, a fork put
before an existing fork, a changed salt, or a changed PRNG moves each pinned seed.

## The per-tick probabilistic fault

Roll each fault on each tick, outside the binary. The binary never asks for a fault. It sees a
slow disk, a corrupt sector, or a partitioned link, and it must react.

A per-tick probability is a rate, not a chance. The expected count is the tick count times the
probability, thus TigerBeetle's `replica_crash_probability` of `ratio(2, 10_000_000)` fires
constantly across millions of ticks. Read a small number as often, never as rare.

Give each fault a stability floor. `tick_crash` counts `replica_crash_stability` down before the
next roll, and the packet simulator holds `partition_stability` and `unpartition_stability`.
Without a floor, a per-tick roll gives a flood of events too short to do anything.
