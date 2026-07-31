# shared/io — the event-loop guide

A **single-threaded, completion-based async IO loop**. Deliberately *un-idiomatic* Go: no
goroutine-per-connection, no blocking reads, no channels. One thread runs everything;
concurrency is many operations in flight at once, not many threads.

---

## 1. What this library is for

One vocabulary, each term built from the ones before it:

- An **event** — one crossing of the boundary between an application and the world: an
  *emission* (an operation submitted: bytes to send, a file to write, a process to
  start) or an *observation* (a completion delivered: the bytes received, the exit
  code, the timer's expiry, the signal).
- The **universe** — every application in the stack that emits or observes events:
  services, daemons, cron jobs, scripts.
- The **timeline** — the absolute order of all events in the universe.
- A **`Completion`** — the caller-owned identity of one in-flight operation: submitting
  it is the emission, its callback firing is the observation.
- **`IO`** — the surface an application emits through and registers what it will
  observe: its entire io vocabulary, held whole by every tier beneath the root. It
  stays op-level so the simulator can sit under the program and fault it per
  operation; coarser injected verbs would put an untested translation layer between
  the two.
- **`Driver`** — the crank (`Run` / `Run_For` / `Run_Until`): delivering observations
  is advancing the timeline, so it is held **only by the code that constructed it**.
- **`time.Clock`** — an application's local, read-only view of time; never a source
  of order.

**This framework makes the timeline an injected dependency.** This works because
application code is deterministic between observations: given the same observations in
the same order, it makes the same emissions in the same order. Control the order and
outcome of every observation and you have controlled the entire run.

In production nobody has that control. The applications run on separate boxes, each
kernel schedules its own box's io, each box has its own clock, and no component ever
sees the absolute order of events. The worst bugs are ordering bugs — the retry that
lands after the failover, the poll that reads mid-deploy, two writers interleaving on
one file — and when one happens in production you get one bad ordering, once, with no
way to reproduce it. Hand-written mocks don't help: they replay only the orderings the
test author thought of.

The fix is to make the ordering come from somewhere controllable. Every binary is split
into a pure entry point — an `internal.Main` that receives its `io.IO` and `time.Clock`
as arguments and does io only through them — and a thin `main` that constructs the real
pair. Two backends can then sit behind the same `io.IO`: the OS backend, where the
kernel decides what completes when, and the simulator, where a seed decides. The
application cannot tell the difference, and each seed produces one specific,
reproducible ordering.

The **universe package** applies this to the whole product at once: one composition
root that imports every application's pure entry point and connects them all to one
simulated loop. Every event across every application then happens in one process, in
one order, decided by one seed — a fault can be injected between any two events, and
any failure reproduces by re-running its seed. Each application still gets its own
clock, because real boxes have separate, skewed clocks: a clock tells an application
what time it thinks it is; the loop decides what actually happens, in what order.

## 2. THE CRITICAL GUARANTEE

```
=====================================================================================
 ONLY PACKAGE MAIN OR A TEST MAY DRIVE, RUN, OR TICK THE EVENT LOOP.

 NOT A LIBRARY. NOT A SHARED PACKAGE. NOT A HELPER. NOT AN INJECTED FUNC VALUE.
 NOT "JUST THIS ONCE". IF CODE OUTSIDE PACKAGE MAIN OR A TEST ADVANCES THE LOOP,
 THAT IS AN ARCHITECTURAL BUG — EVEN IF IT WORKS, EVEN IF EVERY TEST PASSES.
=====================================================================================
```

Driving is any act that advances the timeline: calling `Run`, `Run_For`, or `Run_Until`,
or invoking the clock tick. The only code allowed to do it is the code that constructed
the driver — a binary's `package main` in production, a test harness in simulation (the
universe package is one). Everything else emits and observes events; it never orders
them.

Why "even if every test passes": a library that pumps the loop works fine while its
binary owns the whole process, so no test inside that binary will ever catch it. The bug
detonates later, when the binary is assembled into the universe package and the loop it
pumps is the whole product's — one library call site delivering every other
application's completions from inside its own stack, destroying the one thing the
assembly exists for: holding the absolute order. That is why a violation is an
architectural bug and not a runtime bug: it is invisible where it is written and fatal
where it composes.

Three layers enforce it, none of them optional:

- **Construction**: `IO` has no time-advancing method; the tick lives on the driver
  side, physically out of a library's reach.
- **Lint**: constructing a backend — or even *naming* `io.Driver` — is confined to
  `package main` and tests.
- **Runtime**: driving from inside a callback panics; an unbounded pump with nothing in
  flight panics.

The lint cannot see every disguise — a bare func value with `Run_Until`'s shape slips
through. That a violation compiles and passes does not make it legal. If you think you
need an exception, you need a different shape instead — section 5 has all of them.

**Corollary: there are no synchronous ops below the root.** "How does library code wait
for its op" is a category error — it doesn't. Pure code submits against `io.IO`, returns,
and exposes its progress as state; sequence is a completion chain; doneness is a
predicate; the root pumps. Section 5 shows the shapes.

## 3. Getting a loop

Two backends behind the same surface, constructed only at a composition root:

```go
// Real OS backend — a binary's package main.
clock, tick := timeos.New_Operating_System_Clock()
loop, driver := iodefault.New_Operating_System_IO(clock)

// Deterministic simulator — a test harness. The seed is the ONLY input.
loop, driver, clock := io.New_Sim(seed)
```

The constructor keeps `driver` and drives; it passes `loop` and the clock down. Library
code cannot tell which backend is behind its `loop` — that is the point: the same entry
point runs against the OS in production and inside a fabricated world under test. The
universe package is this recipe at stack scale: one `New_Sim`, every application's
entry point wired onto the one loop, per-application virtual clocks.

## 4. Driving — root code only

```go
driver.Run()                       // one pass: dispatch whatever is ready NOW, then return
driver.Run_For(duration)           // crank for a duration of (real | virtual) time
completed := driver.Run_Until(done, timeout)  // crank until done() — the root's pump
```

`Run_Until` semantics, exactly:

- It pumps, re-checking `done()` after every drain, and **returns the moment `done()`
  flips** — an op that completed inline returns immediately, and idle waits block until
  the genuine next event (nearest timer, socket readiness, worker wake), never on a
  polling interval.
- `timeout` is a **guard, not a goal**: `completed` reports whether `done()` tripped
  (`true`) or the deadline did (`false`). Use `Run_For` when elapsed time *is* the goal.
- `io.FOREVER` (`-1`) never expires — for a program that runs until an event, not a clock.
  If a `FOREVER` pump reaches a state where nothing in flight could ever flip `done()`,
  the loop **panics** rather than hanging: awaiting the impossible is a deadlock.
- `io.IMMEDIATE` (`0`) evaluates `done()` once without driving — a non-blocking poll.

## 5. Program shapes

### a. Linear chains — the next op is submitted inside the callback

Submitting from a callback is the model working as intended (only *driving* from a
callback is banned). A protocol step chains to the next:

```go
socket, err := loop.Open_Socket_TCP(io.FAMILY_IPV4, options)
if err != nil { ... }
state.Socket = socket
loop.Connect(&connect_completion, func(_ *io.Completion, err error) {
	if err != nil { ... ; return }
	loop.Send(&send_completion, func(_ *io.Completion, count int, err error) {
		if err != nil { ... ; return }
		receive_first(socket)   // submits the first Receive
	}, socket, request)
}, socket, address, 30*time.SECOND)
```

`Connect` borrows the descriptor and reports only an error. It never creates, transfers,
or closes the socket. Its positive finite deadline wins ties and reports
`io.Deadline_Exceeded`. Teardown records the terminal result, calls `Shutdown(BOTH)` to make an
armed socket operation resolve, waits for that callback, then submits asynchronous `Close`;
closing before the borrower drains violates the one-completion/one-operation ownership rule.

Re-arming the same descriptor from within its own completion is supported: the loop
retires a finished op *before* delivering its callback, so a newly armed op is not swept
by the old one's cleanup.

### b. Iteration — the trampoline

A chain that loops back on itself — the next `Receive` submitted from the last one's
completion, write *k+1* from write *k* — is a static call cycle, which the no-recursion
lint rejects. The cure: the completion only **records** the continuation in state; a
rearm function — called by the root, outside any callback — turns recorded state into
submissions. Callback writes state, rearm submits, the call graph stays acyclic:

```go
// State replaces the call stack: the cursor is the program counter.
type mirror struct {
	Writes           []write
	Index            int
	Write_Queued     bool
	Done             bool
	Status           int
	Write_Completion io.Completion
	Close_Completion io.Completion
}

// Called by the root only. Reports whether it armed anything.
func mirror_rearm(loop io.IO, state *mirror) (armed bool) {
	if !state.Write_Queued {
		return false
	}
	state.Write_Queued = false
	if state.Index >= len(state.Writes) {
		state.Done = true
		return false
	}
	next := state.Writes[state.Index]
	file, err := loop.Create(next.Path) // inline op: no completion, no timeline
	if err != nil {
		state.Status = 1
		state.Done = true
		return false
	}
	loop.Write(&state.Write_Completion, func(_ *io.Completion, _ int, write_err error) {
		mirror_written(loop, state, file, write_err)
	}, file, next.Contents, 0)
	return true
}

// Completion: record progress and queue the continuation — never submit the next
// iteration here; that is rearm's job, from the root.
func mirror_written(loop io.IO, state *mirror, file io.File, err error) {
	if err != nil {
		state.Status = 1
		state.Done = true
		return
	}
	loop.Close(&state.Close_Completion, func(_ *io.Completion, _ error) {
		state.Index++
		state.Write_Queued = true
	}, file)
}
```

A "mostly synchronous" program is exactly this: sequential means one op in flight at a time — the
degenerate state machine. Its planning, diffing, and parsing stay plain functions (pure computation
and inline ops never touch the timeline); only the few truly async sites (content reads, the write
loop, the spawn sequence) become chain links.

### c. What an application exposes, and how the root runs it

An application never runs its own loop. Its entry point submits the standing work
(listeners, signal watches, the first timer), then returns a handful of plain functions
for the root to call:

```go
type Runner struct {
	Cadence     func()                  // run periodic work whose interval has elapsed
	Rearm       func() (armed bool)     // submit any continuations recorded by callbacks
	Work_Queued func() (queued bool)    // is there a recorded continuation waiting?
	Stopped     func() (finished bool)  // is the application finished?
}
```

Whoever holds the driver — the binary's `main` in production, the harness or the
universe package in simulation — runs the same loop either way:

```go
for !runner.Stopped() {
	runner.Cadence()
	for runner.Rearm() {   // keep arming until no continuation is waiting
		driver.Run()
	}
	driver.Run_Until(func() (finished bool) {
		return runner.Stopped() || runner.Work_Queued()
	}, tick)               // sleep until a completion records new work, or the tick
}
```

This split is what makes the universe package possible: with twenty applications on one
loop, the root simply calls each one's `Cadence` and `Rearm` in turn — no application
can monopolize the loop, because none of them can run it.

`Work_Queued` must report exactly what `Rearm` would act on. If a continuation flag is
checked by `Rearm` but missing from `Work_Queued`, the loop sleeps through it and that
continuation waits out the full tick every time — slow, and silent. If `Work_Queued`
reports something `Rearm` never acts on, the loop wakes, arms nothing, sleeps, wakes —
a busy spin. Guard the pairing with an invariant in the fuzz harness: after `Rearm`
returns false, `Work_Queued` must be false too.

A sequential program exposes the same functions, just fewer of them — `mirror_rearm`
in section b *is* a `Rearm`, and its `Done` field *is* `Stopped`.

### d. Graceful shutdown

A signal is an event like any other: it arrives as a completion on the loop thread, not
on some separate goroutine. There is no `signal.Notify` channel, no `os.Exit` from a
handler, and nothing to synchronize — the signal callback runs between two other
completions, touching the same state they do.

Shutdown is therefore just another state transition. The application watches for the
signals at startup, and the first one to arrive starts a bounded drain:

```go
// At startup — armed once, alongside the listeners and timers.
loop.Watch_Signal(&state.Terminate_Completion,
	func(_ *io.Completion, _ io.Signal) { drain_begin(state) }, io.SIGNAL_TERMINATE)
loop.Watch_Signal(&state.Interrupt_Completion,
	func(_ *io.Completion, _ io.Signal) { drain_begin(state) }, io.SIGNAL_INTERRUPT)

// The first signal starts the drain; a second one changes nothing.
func drain_begin(state *daemon) {
	if state.Draining {
		return
	}
	state.Draining = true
	state.Drain_Deadline = state.Clock.Now_Monotonic() + time.Moment(drain_timeout)
}

// Called from Cadence every pass: stop when the work is gone or time is up.
func drain_tick(state *daemon) {
	if !state.Draining {
		return
	}
	if len(state.Connections) == 0 {
		state.Stop = true
		return
	}
	if state.Clock.Now_Monotonic() >= state.Drain_Deadline {
		state.Stop = true
	}
}
```

While draining, `Rearm` stops arming *new* work — no fresh accepts — but keeps servicing
the continuations of work already in flight, so open connections run to completion. The
deadline bounds a peer that never finishes. When `Stop` flips, `Stopped` reports it, the
root's loop exits, and the process ends — the root, not the application, owns the exit.

This is also why shutdown is modeled as an event instead of a process kill: in the
simulator the signal lands on a seed-drawn grain, so the seed sweep exercises shutdown
arriving in the middle of everything — mid-connect, mid-drain, mid-write — and the drain
invariants have to hold on every one of those timelines.

### e. Banned shapes

- **Injecting `Run_Until` into a library** — as a field, a parameter, any func value. A
  library authoring `done` predicates and timeouts decides when and how long time moves:
  that is driving with the type name filed off; assembled into the universe package it
  advances every other application's events from inside its own stack.
- **Injecting root-built blocking wrappers** (`Spawn func(request) result`) — the same
  crime by proxy: the closure is root code, but it pumps the shared timeline from under a
  library call site.
- **Replacing `io.IO` with injected domain verbs** (`Read_File func(path) ...`) — takes
  the binary off the op-level surface. The simulator can no longer sit under it, fault
  granularity collapses to whatever the verbs expose, and the untested wrappers become
  the actual io layer. That is mock-DI, the thing this library exists to kill.

### f. The flag smell — booleans sequencing io are a state machine in denial

The shapes above each add a boolean or two: a `*_Queued` flag for the trampoline, an
`Active` guard against overlap, a `Done`, a `Draining`. One is fine. But they
accumulate, and every new one doubles the representable combinations: five booleans is
thirty-two states, of which maybe six mean anything. Which combinations are valid, and
which moves between them are legal, exists only in the author's head. The tell is
always the same: correctness starts depending on the *order* the flags are checked —
a rearm that must test `Closed` before `Receive_Queued` is a transition table written
as an if-ladder, by accident.

```go
type mirror_state int
const mirror_idle         mirror_state = 0
const mirror_write_armed  mirror_state = 1
const mirror_write_queued mirror_state = 2
const mirror_done         mirror_state = 3

func mirror_transition_legal(from, to mirror_state) (legal bool) {
	if from == mirror_idle {
		return to == mirror_write_armed
	}
	if from == mirror_write_armed {
		if to == mirror_write_queued {
			return true
		}
		return to == mirror_done
	}
	if from == mirror_write_queued {
		if to == mirror_write_armed {
			return true
		}
		return to == mirror_done
	}
	return false
}

func mirror_transition(state *mirror, from, to mirror_state) {
	if state.State != from {
		panic("mirror: transition from a state the caller did not expect")
	}
	if !mirror_transition_legal(from, to) {
		panic("mirror: transition along an edge the machine does not have")
	}
	state.State = to
}
```

The live exemplar is the `Completion` machine in `shared/io/io.go` — `Completion_State`,
`Completion_Transition_Legal`, `Completion_Transition` — enforced identically by both
backends.

## 6. Rules of the loop

Each of these fails loudly where the runtime can make it:

1. **Nothing runs until the root drives.** A submitted op without a `Run*` call sits
   forever. If something "hangs", ask first: who was supposed to drive, and did they?
2. **Never drive from inside a callback** — re-entering the driver panics. In a callback,
   *submit* and return.
3. **One `Completion` per in-flight op** — resubmitting an armed completion panics. Two
   ops at once means two completions. Reuse is fine after the callback fires.
4. **Never copy a `Completion`** — the loop tracks the op by pointer; submitting a
   by-value copy panics. Keep each as its own value and pass `&completion`; store many as
   `[]*io.Completion`, never `[]io.Completion` (append moves the array under the loop).
5. **Never block the loop thread — or anywhere else.** Blocking io is an async op on
   the loop, a subprocess is `Spawn`, and `Compute` parallelizes compute with
   goroutines — no more, no less; anything that waits on the world is an op. `Compute`
   is **only** for compute-intensive work that parallelizes well — JSON parsing,
   high-traffic request processing — never an escape hatch for blocking io.
6. **Buffers belong to the loop until the callback fires.** Reusing or resizing a
   submitted buffer races the backend.
7. **Every submission resolves exactly once** — including operations retired by descriptor
   teardown (`io.Canceled`).
   Callbacks must handle the error path.
8. **No goroutines or channels around loop state.** The completion *is* the await; the
   loop *is* the concurrency.

## 7. The simulator and the seed rule

`New_Sim(seed)` is a **pure function of its seed**. Every outcome — latency, which connect
fails, delivered byte counts, which spawn exits non-zero, the grain a signal lands on — is
drawn from the seed's PRNG. There is **no scripting API**, deliberately:

- **Never inject an outcome.** No "make this connect fail", no "deliver these bytes". A
  test needing a different outcome changes the seed. A scripted sim finds only the bugs
  its author imagined and stops reproducing from its seed.
- **Assert invariants over a seed sweep**, not pinned outputs. Drive the real entry point
  over hundreds of seeds (`f.Fuzz` with a corpus loop) and assert properties that must
  hold for *any* run: convergence, idempotency, no double-fire, cursor monotonicity.
- **The test harness drives the same loop shape as `main`** — same canonical loop, same
  `Runner`. If prod and test drive differently, the suite exercises a loop nobody ships,
  and real-backend-only bugs hide in the gap.
- The sim driver advances one grain per tick and never jumps to the next scheduled event —
  the idle grains are exactly where a fault adversary crashes a box or drops a partition,
  so every grain stays a decision point.

The universe package is this recipe at full scale: every application's entry point on
one sim loop, per-application clocks skewed by the seed, the product's timelines
explored one seed at a time.

If you want to export the `sim` type or add a `New_Sim` parameter that is not the seed:
stop. That is the scripting API trying to come back.

## 8. Layout

- **`shared/io`** — the `IO`/`Driver` surface and the deterministic simulator. No
  syscalls. This is what every binary's pure tier depends on.
- **`shared/io/default`** — the real OS backend: kqueue/epoll readiness, inline file
  syscalls, a compute worker pool with a self-pipe wake, TLS goroutines, process
  spawning. The **only** place raw IO stdlib (`net`, `syscall`, `os/exec`, `crypto/tls`,
  …) is allowed; the `io-gateway` lint rule keeps everyone else routing through
  `shared/io`.
- **`shared/time/default`** — the clock gateway, the only importer of stdlib time.
