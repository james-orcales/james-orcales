# shared/io — working in the event-loop model

This package is a **single-threaded, completion-based async IO loop**, modeled on
TigerBeetle's `io.zig`. It is deliberately *un-idiomatic* Go. If you write it the way Go
teaches you to — a goroutine per connection, a blocking `n, err := conn.Read(buf)`,
channels carrying results — you will fight it the whole way and lose.

This document is not an API reference (read `io.go` and `SPECIFICATION.md` for the ops).
It is the mental model and the list of ways people cut themselves.

---

## The whole idea in one paragraph

You are handed an `IO` — a **submit surface**. You hand it an operation, a caller-owned
`Completion`, and a callback, and it returns immediately. **Nothing has happened yet.**
The operation sits in a queue. Separately, whoever holds the `Driver` turns the crank
(`Run` / `Run_For` / `Run_Until`). *Only while the crank turns* do operations complete and
callbacks fire — one at a time, on the loop thread, in the order they come due. When the
crank stops, everything freezes. That's the entire model.

> Idiomatic Go blocks a goroutine until IO finishes. Here, submitting and completing are
> torn apart in time, and one thread runs everything. Retrain that instinct first.

---

## The four nouns — keep them straight

| Noun | What it is | Who holds it |
|------|-----------|--------------|
| `IO` | the submit surface (a struct of closures) | pure/library code |
| `Driver` | the crank: `Run` / `Run_For` / `Run_Until` | **only** `package main` or a test harness |
| `Completion` | caller-owned storage for **one** in-flight op | the caller of an op |
| `time.Clock` | read-only time source, *beneath* the loop | anyone (read-only) |

You do not construct `IO`/`Driver` yourself except at a composition root. You are *handed*
an `IO`. See "Getting one" below.

---

## The capability split — why you can't "just run the loop"

`IO` can submit but **cannot drive**. `Driver` drives. They are separate types on purpose,
and the separation is enforced twice:

- **Compiler:** a package that only has an `IO` has no method that advances time.
- **Linter:** the `driver-gateway` rules forbid *constructing* a loop or even *naming the
  `io.Driver` type* outside `package main` and tests.

Why go to that trouble? Because the whole point of a simulated loop is that a **test
harness controls time** — it decides when things complete and injects faults on the
timeline. If library code could grab the crank and turn it, it would steal that control.
So the rule is absolute: **`internal.Main` takes an `IO`; `main` and the fuzz harness hold
the `Driver` and drive.** If you feel the urge to give a library the `Driver`, that urge is
the bug.

---

## Getting one

Two backends, chosen *by value* — the same `IO` surface, a different thing behind it.

```go
// Real OS backend — package main only.
clock, _   := timeos.New_Operating_System_Clock()
loop, driver := iodefault.New_Operating_System_IO(clock)

// Deterministic simulator — main or a test. Seed is the ONLY input.
loop, driver, clock := io.New_Sim(seed)
```

`main` (or the harness) keeps `driver` and drives. It passes `loop` (the `IO`) down into
the library. The library never sees `driver`.

---

## How you actually get work done

### 1. Straight-line / sequential program — the synchronous pump

If your program does one thing after another (like `setup`: build this, then that), wrap
"submit + drive until it finishes" into an ordinary blocking call. This is the bridge from
a synchronous world into the async loop:

```go
func run(loop io.IO, driver io.Driver, request io.Process_Request) (result io.Process_Result) {
	var completion io.Completion
	done := false
	loop.Spawn(&completion, func(_ *io.Completion, r io.Process_Result, err error) {
		result = r
		done = true
	}, request)
	driver.Run_Until(func() (finished bool) { return done }) // crank until this op is done
	return result
}
```

The `Driver` stays in `main` (this pump lives in `main`); the library just calls `run`.
This is safe **only** because a sequential program never enters the pump while already
inside it (see pitfall #3).

### 2. Event-driven / concurrent program — submit many, react in callbacks

Submit several operations up front, let the harness drive `Run_For` / `Run_Until`, and do
your sequencing *inside the callbacks*: when one op completes, its callback submits the
next. Your program becomes a **state machine** whose transitions are completions. The
harness owns the crank the whole time.

---

## The pitfalls — this is the part that bites

**1. Nothing runs until you drive.** The single most common confusion. You submit an op,
it "does nothing," and you assume it's broken. It isn't — you never turned the crank. A
submitted op that never sees a `Run*` call just sits there forever. If something "hangs,"
the first question is: *who was supposed to drive, and did they?*

**2. Give each in-flight op its own `Completion`, and never copy it.** When you submit,
you pass `&completion` — a *pointer*. The loop remembers that address, writes the result
there later, and calls the callback from a `Run*`. Go's GC keeps the memory alive as long
as the loop holds the pointer, so you do *not* have to worry about lifetime — a plain
`var completion io.Completion` is fine even if the enclosing function returns before the
loop is driven. What you must worry about is **identity**: the loop and you have to be
looking at the same memory. Copying the value (`snapshot := completion`) or keeping
completions in a `[]io.Completion` you `append` to (a grow moves the backing array, so the
loop's pointers point at the abandoned copy) leaves the loop writing one place while you
read another. Keep each op's `Completion` as its own value and pass its address; if you
keep many, use `[]*io.Completion`, not `[]io.Completion`.

**3. One `Completion` backs one in-flight op — reuse panics.** Submitting a `Completion`
that is still in flight trips a loud assertion (`Armed`) rather than silently corrupting
the queue. Want two ops at once? Two `Completion`s. Reusing one is only fine *after* its
callback has fired.

**4. Never drive from inside a callback.** `Run` / `Run_For` / `Run_Until` are top-level,
single-loop only — **never** call one from within a completion callback. Doing so
re-enters the driver and corrupts the loop's state. This is the sharpest edge in the whole
model. If you're in a callback and need another op, *submit* it (and return); don't drive.

**5. Never block the loop thread.** Callbacks run on the one thread that runs everything.
A callback that makes a blocking syscall, sleeps, or spins stalls *every other operation*.
Heavy CPU work → hand it to `Compute` (runs off-thread, completes back on the loop). A
subprocess → `Spawn`. A raw blocking call in a callback → you've frozen the loop.

**6. Buffers must stay valid and untouched until the callback fires.** `Read` / `Write` /
`Receive` / `Send` complete *later*. The buffer you passed is read or written by the loop
in the meantime. Reusing, resizing, or freeing it before the callback is a data race with
the loop. Same rule for `Compute`'s `work`: it may only touch memory the loop leaves alone
until it completes.

**7. Some ops are synchronous — don't expect them to fault like async ones.** `Open`,
`Create`, `Listen`, `Peer_Address` return immediately with no `Completion`, because
opening/binding/getpeername don't block. The interesting, faultable, latency-bearing
behavior lives on the *later* async op (the `Read` after the `Open`, the `Accept` after the
`Listen`). Don't design around `Open` failing the way a `Read` can.

**8. Every submission resolves — handle the error path.** Cancelling an op still fires its
callback exactly once, with the `Cancelled` error instead of a result. Callbacks that
assume success-only will mishandle a cancel. Always read the `err` / check `Cancelled`.

**9. Don't reach for goroutines/channels for IO.** The loop *is* your concurrency. Adding a
goroutine that touches loop state, or a channel to "await" a result, reintroduces exactly
the multi-threading the single loop exists to avoid. The completion *is* the await.

---

## The simulator and the seed rule — read this before touching the Sim

`New_Sim(seed)` is a **pure function of its seed.** Every outcome — how long an op takes,
which connect fails, the bytes a receive delivers, the grain a signal lands on — is drawn
from the seed. There is **no scripting API**, and `io.go` carries a long, angry comment
explaining why you must never add one. The short version:

- **You do not feed data into the sim.** No "make *this* connect fail," no "deliver *these*
  bytes." If a test needs a different outcome, **change the seed** — never inject the
  outcome. A scripted sim finds only the bugs the author imagined, stops reproducing from
  its seed, and judges hand-picked inputs instead of the whole space.
- **Correctness is asserted by invariants across a seed sweep**, not by pinning specific
  outputs. Drive your program's entry point over many seeds and assert properties that must
  hold for *any* run. (See `maddox/internal/simulation_test` for the shape.)
- **The driver ticks one grain at a time and never jumps** to the next scheduled event,
  even when the queue is idle until then — so a time-triggered fault (crash/partition a
  quiescent node) can strike *any* grain, not just the ones where something was scheduled.

If you find yourself wanting to export the `sim` type, or add a `New_Sim` parameter that
isn't the seed, stop. That is the scripting API trying to come back.

---

## Where the raw IO lives

- **`shared/io`** — the `IO`/`Driver` surface and the deterministic simulator. No real
  syscalls. This is what library code depends on.
- **`shared/io/default`** — the real OS backend (kqueue/epoll, TLS goroutines, a worker
  pool, a self-pipe wake). The **only** place raw IO stdlib (`net`, `syscall`, `os/exec`,
  `crypto/tls`, …) is allowed. The `io-gateway` lint rule keeps it that way: everyone else
  routes through `shared/io`, and `package main` is the exempt wiring shell that binds the
  two together.
