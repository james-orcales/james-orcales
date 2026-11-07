---
name: inject_io
description: >
  Load BEFORE you give a package an `io.IO`, write or review a composition root, or move code
  that blocks to the loop. Only `package main` or a test can hold the `io.Driver`. Gives the
  search that finds a violation, the `Runner` shape that replaces it, and the six defects to
  check after the conversion.
---

# Inject `io.IO`

## The point

The timeline is an injected dependency. One `io.IO` surface runs on the operating system in
production and on `New_Sim(seed)` under test, thus one seed reproduces any bug.

The timeline holds the absolute order of all events in the universe, and that order needs one
holder of the `io.Driver`. A library that drives passes its own tests, because its binary owns
the whole process. It fails when the universe package composes it. There its one call site
delivers every other application's completions from inside its own stack, and the order stops
being a property of the seed.

## Find the violation

```
rg -n "Run_Until|Run_For|driver\.Run|io\.Driver" --type go
```

Each hit must sit in a `main.go` file or a `_test.go` file. A hit in any other file is the bug.

`lint` reads names, not shapes. These five constructions pass it:

1. A wrapper copies the `io.IO` value and replaces its members. The `Driver` hides in the closure.
2. A helper takes a `done func() (finished bool)` parameter and drives in the body.
3. A library holds a field with the type `func()` or `func() (err error)`.
4. A test fake calls its callback before its `Read` returns.
5. A `Deinit` or a `Close` pumps until the queue is empty.

## The shape that replaces it

The callback records one continuation. The root runs it.

```go
// setup/internal/setup.go — library. The callback stores. It never continues.
system.Spawn(completion, func(
    _ *sysio.Completion, result sysio.Process_Result, operation_err error,
) {
    runner_complete_io(state, func() { continuation(result, operation_err) })
}, request, PROCESS_DURATION_MAX)
```

```go
// setup/main.go — root. Run each continuation, then make one pump pass.
for !runner.Stopped() {
    for runner.Rearm() {
    }
    if runner.Stopped() {
        break
    }
    completed, drive_err := driver.Run_Until(func() (finished bool) {
        if runner.Stopped() {
            return true
        }
        return bool(runner.Work_Queued())
    }, setup.PROCESS_DURATION_MAX)
```

`Runner` carries `Rearm`, `Work_Queued`, `Stopped`, and `Status`. It never carries a pump.

## Check these six after the conversion

1. **`Deinit` on the failure path.** The root stops with an operation still in flight, and
`Deinit` asserts that every operation retired. Do not call `Deinit` there.
2. **One constant for the pump and the operation.** Equal deadlines race. Make the root guard
strictly larger. The operation then retires with `Deadline_Exceeded`, and the failure path
becomes unreachable.
3. **The submitted `Completion` is not the caller's.** `Completion.Self` and
`Completion_Transition` never see the caller value, thus a second submission passes.
4. **One half of the surface.** Check `Fsync`, `Timeout`, `Next_Tick`, `Open_At`, and the socket
members. A member that keeps the loop shape has nothing to pump it.
5. **The inline assumption in the caller.** Search the callers for `retired` and for "did not
retire before return". That text says that library code requires delivery before the call
returns.
6. **A guard that one delivery order reaches.** A fake retires twice inside one submission. A
real loop delivers on two passes with a `Rearm` between them, which clears what the guard reads.

Defect 1 and defect 6 need a test with `sysio.New_Sim` and a real `Driver`. See
`Test_Main_Drive_Runner_Advances_Original_IO`.

## The simulation must defer

A hand-written `sysio.IO` that retires inline runs no driver, thus it covers one delivery order
only. Build the sweep on `sysio.New_Sim(seed)`. The harness holds the `Driver`.
