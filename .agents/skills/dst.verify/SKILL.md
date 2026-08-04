---
name: dst.verify
description: >
  Load BEFORE you trust a check, bank a seed, or close a bug found by a fuzzer. A fuzzer proves
  that a path ran. A mutation proves that something watched when it ran. Break the check, run the
  suite, and put the check back — a suite that still passes says that nothing observes it. Then
  name the class the bug belongs to and correct every other instance of it.
---

# Verifying that a check is worth something

## Golden snapshots

A pinned seed dies in silence, because a moved stream still passes the suite. `snap.Expect` the
first values of each named stream, for two or three fixed seeds. Keep one snapshot for each
stream, never one for the whole run, because the stream that fails states the blast radius.

`prng.Test_Known_Sequence` already freezes the generator itself. Nothing freezes the topology
above it. A changed snapshot is not a file to update. It is the instruction to sweep again.

## The probe loop

Break the check, run the suite, then put the check back. A suite that still passes says that
nothing observes the check, thus nothing stops the next person from removing it.

## The seven ways a check is worth nothing

1. **It reads data that nothing fills.** The gate tests a field that the query never selected.
2. **It bounds the part, not the whole.** Each step meets its own deadline, and a flow that
cycles between steps never completes.
3. **It exercises the callee, not the caller.** The encoder works when it has space, and nothing
shows that the caller supplies that space.
4. **It passes for the wrong reason.** Remove the guard, and something else rejects the input the
same way.
5. **It reads the constant that it guards.** `delay <= max_delay` passes at any `max_delay`.
Write the number again in the test, on purpose.
6. **It samples where the limit is out of reach.** "No run kept more than 2048 clients" passes
with the cap removed. Drive the capping path `2 * max` times.
7. **It pins a seed, not a fix.** Determinism has the scope `(seed, build)`. Make a unit test the
durable artifact, and keep the seed as a bonus.

Number 5 and number 6 survive a review, because each reads as exactly the property you wanted.

## Audit the full class

When the fuzzer shows a bug, name the class it belongs to, then grep for every other instance.
One empty-binding bypass became thirteen corrected sites in one pass, under the class "a security
decision made on data that silently takes a default value".

A speculative audit finds nothing, because you hold no filter for what counts. An audit anchored
to one demonstrated instance finds the rest, because the proven bug gives you the exact grep. The
same holds for any assertion that fires, not only for a fuzzer finding.
