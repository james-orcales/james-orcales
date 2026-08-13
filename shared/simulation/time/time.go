// Package time is a dependency-injected time source modeled on TigerBeetle's
// vsr.Time, whose backend is a struct of function pointers (a vtable). Here that is
// a struct of closures: production wires an OS clock (time/default), a simulation
// wires a Virtual one, and the code between never knows which it holds. The Virtual
// backend lives here because it is pure arithmetic with no operating-system call.
package time

import (
	"errors"

	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/fixedpoint"
)

// INTEGER_64_MINIMUM is the smallest signed 64-bit integer. A Moment, a Duration, and
// a Tick_Count each occupy the whole signed 64-bit range: a virtual clock reads its
// epoch, its skew, and its tick count as the caller sets them, and none of the three
// carries a narrower bound than its storage.
const INTEGER_64_MINIMUM int64 = -9223372036854775808

// INTEGER_64_MAXIMUM is the largest signed 64-bit integer.
const INTEGER_64_MAXIMUM int64 = 9223372036854775807

// Moment is a clock reading in nanoseconds since an arbitrary, clock-specific epoch
// (TigerBeetle's stdx.Instant). Only the difference between two Moments from the
// SAME clock is meaningful; a monotonic Moment and a realtime Moment are not
// comparable.
type Moment int64

// Moment_Invariants states the complete clock-reading domain.
func Moment_Invariants(moment Moment, namespace invariant.Namespace) {
	invariant.Tree(moment, namespace).
		Range_Int64(int64(moment), INTEGER_64_MINIMUM, INTEGER_64_MAXIMUM).
		Ensure()
}

// Duration is a span of nanoseconds (TigerBeetle's stdx.Duration).
type Duration int64

// Duration_Invariants states the complete nanosecond-span domain.
func Duration_Invariants(duration Duration, namespace invariant.Namespace) {
	invariant.Tree(duration, namespace).
		Range_Int64(int64(duration), INTEGER_64_MINIMUM, INTEGER_64_MAXIMUM).
		Ensure()
}

// NANOSECOND is the unit a Duration counts in.
const NANOSECOND Duration = 1

// MICROSECOND is a thousand nanoseconds.
const MICROSECOND = NANOSECOND * 1000

// MILLISECOND is a thousand microseconds.
const MILLISECOND = MICROSECOND * 1000

// SECOND is a thousand milliseconds.
const SECOND = MILLISECOND * 1000

// MINUTE is sixty seconds.
const MINUTE = SECOND * 60

// HOUR is sixty minutes.
const HOUR = MINUTE * 60

// DAY is twenty-four hours.
const DAY = HOUR * 24

// WEEK is seven days.
const WEEK = DAY * 7

// Any_Clock is the injected time source — the Go translation of TigerBeetle's `Time`
// vtable, expressed as closures so the backend is chosen by value. It is read-only:
// advancing time is the driver's job (the tick returned beside the clock at
// construction), so a holder can only read the current Moment, never move time.
type Any_Clock struct {
	// Now_Monotonic reads the monotonic clock, which never regresses; use it to
	// measure elapsed time, timeouts, and latency.
	Now_Monotonic func() (moment Moment)
	// Now_Realtime reads wall-clock time as nanoseconds since the Unix epoch; it can
	// jump, so use it only for calendar timestamps, never for elapsed time.
	Now_Realtime func() (moment Moment)
}

// Any_Clock_Invariants states that both readers are bound. A Any_Clock is a vtable, so its
// only property is that every slot is filled: the zero Any_Clock reads as a Any_Clock but
// panics on first use, and a backend that fills one slot and forgets the other is the
// same failure one call later.
func Any_Clock_Invariants(clock Any_Clock, namespace invariant.Namespace) {
	invariant.Always(
		clock.Now_Monotonic != nil, "A Any_Clock has a monotonic reader.",
	)
	invariant.Always(
		clock.Now_Realtime != nil, "A Any_Clock has a realtime reader.",
	)
}

// Tick_Count is how many times a virtual clock advanced — the abscissa every skew
// model reads (TimeSim's x).
type Tick_Count int64

// Tick_Count_Invariants states the complete tick-count domain.
func Tick_Count_Invariants(ticks Tick_Count, namespace invariant.Namespace) {
	invariant.Tree(ticks, namespace).
		Range_Int64(int64(ticks), INTEGER_64_MINIMUM, INTEGER_64_MAXIMUM).
		Ensure()
}

// Offset models how a simulated wall clock deviates from true elapsed time —
// TigerBeetle's TimeSim.offset. It is what makes Now_Realtime diverge from
// Now_Monotonic. A nil Offset is a perfect clock.
type Offset func(ticks Tick_Count) (skew Duration)

// Virtual_Clock configures the deterministic clock Virtual_Clock_To_Any_Clock builds —
// TigerBeetle's TimeSim. Time advances only when Tick is called, so a simulation
// reaches a future Moment by ticking rather than by waiting.
type Virtual_Clock struct {
	// Resolution is how far the monotonic clock advances on each Tick — the grain of
	// a simulated oscillator (TimeSim.resolution).
	Resolution Duration
	// Epoch is the wall-clock origin: Now_Realtime at tick zero, before any skew
	// (TimeSim.epoch).
	Epoch Moment
	// Skew bends Now_Realtime away from true elapsed time; nil is a perfect clock
	// (TimeSim.offset).
	Skew Offset
}

// Virtual_Clock_Invariants states the two scalars a virtual clock is configured with.
// Skew is a closure, so the arithmetic it stands for has no domain to state here; its
// coefficients are stated where Skew builds it.
func Virtual_Clock_Invariants(virtual Virtual_Clock, namespace invariant.Namespace) {
	Duration_Invariants(virtual.Resolution, namespace)
	Moment_Invariants(virtual.Epoch, namespace)
}

// Virtual_Clock_To_Any_Clock returns a read-only Any_Clock backed by a deterministic, OS-free
// virtual clock, plus the tick that advances it. The clock's closures and tick share
// one counter, so tick advances what the next Now_Monotonic reads. Only the driver —
// package main or a test harness — holds tick; pure code holds only the Any_Clock and so
// can read time but never move it.
func Virtual_Clock_To_Any_Clock(virtual Virtual_Clock) (clock Any_Clock, tick func()) {
	defer func() { Any_Clock_Invariants(clock, "virtual_clock_to_clock.clock") }()
	Virtual_Clock_Invariants(virtual, "virtual_clock_to_clock.virtual")
	ticks := Tick_Count(0)
	clock = Any_Clock{
		Now_Monotonic: func() (moment Moment) {
			return Moment(int64(ticks) * int64(virtual.Resolution))
		},
		Now_Realtime: func() (moment Moment) {
			now := virtual.Epoch + Moment(int64(ticks)*int64(virtual.Resolution))
			if virtual.Skew == nil {
				return now
			}
			return now - Moment(virtual.Skew(ticks))
		},
	}
	return clock, func() { ticks++ }
}

// SKEW_KIND_LINEAR models constant drift: A nanoseconds of skew per tick plus an
// initial B (TimeSim OffsetType.linear, A*x + B).
const SKEW_KIND_LINEAR Skew_Kind = 0

// SKEW_KIND_PERIODIC models a sinusoidal wobble of amplitude A over a period of B
// ticks (TimeSim OffsetType.periodic, A*sin(x*2pi/B)).
const SKEW_KIND_PERIODIC Skew_Kind = 1

// SKEW_KIND_STEP models a discontinuous jump of A after B ticks — an NTP correction
// or operator clock change (TimeSim OffsetType.step).
const SKEW_KIND_STEP Skew_Kind = 2

// Skew_Kind selects which clock-deviation model Skew builds.
type Skew_Kind uint8

// Skew_Kind_Invariants holds a kind to the three models Skew builds. The default arm
// of that switch is the linear model, so an unlisted kind would drift silently rather
// than fail.
func Skew_Kind_Invariants(kind Skew_Kind, namespace invariant.Namespace) {
	invariant.Tree(kind, namespace).
		Enum_3_Uint8(
			uint8(kind),
			uint8(SKEW_KIND_LINEAR),
			uint8(SKEW_KIND_PERIODIC),
			uint8(SKEW_KIND_STEP),
		).
		Ensure()
}

// Skew_Input is the model and its coefficients, mirroring TimeSim's offset_type plus
// offset_coefficient_A and offset_coefficient_B.
type Skew_Input struct {
	// Kind selects the deviation model.
	Kind Skew_Kind
	// A is the magnitude coefficient: drift-per-tick, amplitude, or step size.
	A Duration
	// B is the tick coefficient: the linear initial offset, the periodic period, or
	// the step's onset tick.
	B Tick_Count
}

// Skew_Input_Invariants states the model and each of its two coefficients.
func Skew_Input_Invariants(input Skew_Input, namespace invariant.Namespace) {
	Skew_Kind_Invariants(input.Kind, namespace)
	Duration_Invariants(input.A, namespace)
	Tick_Count_Invariants(input.B, namespace)
}

// Skew builds the Offset described by input.
func Skew(input Skew_Input) (offset Offset) {
	Skew_Input_Invariants(input, "skew.input")
	switch input.Kind {
	case SKEW_KIND_PERIODIC:
		return func(ticks Tick_Count) (skew Duration) {
			// A zero period is a degenerate sinusoid; report no skew rather than divide
			// (or take a remainder) by zero.
			if input.B == 0 {
				return 0
			}
			// Reduce the phase to one period before lifting it into fixed-point, so a
			// long-running tick count cannot overflow the scaled numerator.
			phase := ticks % input.B
			turns := fixedpoint.From_Ratio(
				fixedpoint.Numerator(phase),
				fixedpoint.Denominator(input.B),
			)
			amplitude := fixedpoint.Number(fixedpoint.From_Integer(
				fixedpoint.Whole_Integer(input.A),
			))
			wobble := fixedpoint.Multiply(
				fixedpoint.Multiplicand(amplitude),
				fixedpoint.Multiplier(fixedpoint.Sine_Turns(turns)),
			)
			return Duration(fixedpoint.Whole(wobble))
		}
	case SKEW_KIND_STEP:
		return func(ticks Tick_Count) (skew Duration) {
			if ticks > input.B {
				return input.A
			}
			return 0
		}
	default:
		return func(ticks Tick_Count) (skew Duration) {
			return Duration(ticks)*input.A + Duration(input.B)
		}
	}
}

// Next_Tick_Source groups deferred callbacks so Reset_Next_Tick can remove a whole source.
type Next_Tick_Source int

// NEXT_TICK_LSM is TigerBeetle's storage-origin next-tick source.
const NEXT_TICK_LSM Next_Tick_Source = 0

// NEXT_TICK_VSR is TigerBeetle's replication-origin next-tick source.
const NEXT_TICK_VSR Next_Tick_Source = 1

// Next_Tick_Callback receives a deferred next-tick completion.
type Next_Tick_Callback func(completion *Completion)

// Timeout_Callback receives a status-only result from a timeout, connect, close, or
// another operation that returns no value beyond its error.
type Timeout_Callback func(completion *Completion, err error)

// Retired_Twice reports that a backend retired one completion more than one time. A derived
// function delivers it rather than hiding it, because the caller owns the completion and must
// learn that its lifecycle broke.
var Retired_Twice = errors.New("time: the completion retired more than once")

// Deadline_Exceeded is returned after a finite operation retires without its external event.
var Deadline_Exceeded = errors.New("time: deadline exceeded")

// FOREVER is the Run_Until timeout that never expires: the loop pumps until done reports
// true, however long that takes — for a caller (a server) that runs until an event, not a
// clock.
const FOREVER Duration = -1

// IMMEDIATE is the Run_Until timeout that expires at once: done is evaluated a single time
// and the loop is not driven — a non-blocking poll of the predicate.
const IMMEDIATE Duration = 0

// Completion is the caller-owned storage for one in-flight operation —
// TigerBeetle's IO.Completion. The caller allocates it, so the loop never does, and
// must keep it alive until the callback fires.
type Completion struct {
	// Callback is the closure the backend runs on completion; it closes over the
	// typed callback and the computed result.
	Callback func()
	// Ready_At is the virtual Moment this operation completes, mirroring
	// TigerBeetle's Storage.Read.ready_at.
	Ready_At Moment
	// Next_Tick_Source identifies next-tick completions for Reset_Next_Tick. Other operations
	// leave it untouched; it is backend-owned metadata.
	Next_Tick_Source Next_Tick_Source
	// Next_Tick reports whether this armed completion is a next-tick operation.
	Next_Tick bool
	// State is the completion's position in its lifecycle machine, mutated only through
	// Completion_Transition. It is backend-owned: applications never read or write it —
	// expose your own state, not the completion's.
	State Completion_State
	// Self is the completion's own address, stamped on its first submit and never cleared.
	// The timeline tracks an in-flight op by pointer, so a by-value copy carries this
	// original address; submitting the copy trips the backend's assert instead of silently
	// splitting the timeline's view from the caller's. Only the backends touch it.
	Self *Completion
	// Kernel_Identifier is the generation token stored in kqueue udata or io_uring user_data.
	// Backends own it; Event_Trigger reads it only after Event_Listen has armed the completion.
	Kernel_Identifier uint64
}

// Completion_State is one position in a completion's lifecycle machine. The machine has
// exactly two legal edges: idle to armed on submit and armed to idle before delivery.
type Completion_State int

// COMPLETION_IDLE is the zero value: never submitted, or delivered and reusable. A
// delivery resets to idle before the callback runs, so a callback may resubmit its own
// completion — the repeating-timer pattern.
const COMPLETION_IDLE Completion_State = 0

// COMPLETION_ARMED marks an in-flight operation: submitted and owned until delivery.
const COMPLETION_ARMED Completion_State = 1

// Completion_Transition_Legal is the machine's transition table: it reports whether the
// edge from one state to another exists. A function rather than a table value because Go
// has no const maps and a package var is banned; the flat one-clause-per-edge shape is
// the point — the whole graph, readable in one place.
func Completion_Transition_Legal(from Completion_State, to Completion_State) (legal bool) {
	if from == COMPLETION_IDLE {
		return to == COMPLETION_ARMED
	}
	if from == COMPLETION_ARMED {
		return to == COMPLETION_IDLE
	}
	return false
}

// Completion_Transition moves a completion along one edge of its lifecycle machine. Its
// two Always guards fail loudly on a caller whose belief about the current state is
// stale — a reused or double-armed completion — and on an edge the machine does not
// have, so a lifecycle bug dies at the mutation instead of corrupting a queue.
// Each transition records both ends of its edge. Thus, the suite must use each legal edge.
// Backend code only. Applications never transition a completion.
func Completion_Transition(completion *Completion, from Completion_State, to Completion_State) {
	invariant.Always(completion.State == from,
		"A completion transitions from the state its caller expects.")
	invariant.Always(Completion_Transition_Legal(from, to),
		"A completion transitions along an edge its machine has.")
	completion.State = to
	invariant.Sometimes(from == COMPLETION_IDLE, "the edge leaves idle")
	invariant.Sometimes(to == COMPLETION_IDLE, "the edge enters idle")
}

// Event is the backend's cross-thread wakeup handle — TigerBeetle's kqueue EVFILT_USER ident
// or eventfd descriptor. It is the loop's own primitive: it carries no bytes and names no
// endpoint, and its only purpose is to make an armed completion ready from another thread.
type Event uintptr

// Operation classifies one armed completion for the census Introspect reports. The classes
// name what a completion is waiting for, not which backend armed it, so one queue reports a
// stall the same way whichever surface submitted the work.
type Operation int

// OPERATION_COMPLETED keeps a delivered-or-ready callback in one class.
const OPERATION_COMPLETED Operation = 0

// OPERATION_TIMEOUT is a pending timer.
const OPERATION_TIMEOUT Operation = 1

// OPERATION_READ_WAITER is a completion waiting for readability.
const OPERATION_READ_WAITER Operation = 2

// OPERATION_WRITE_WAITER is a completion waiting for writability.
const OPERATION_WRITE_WAITER Operation = 3

// OPERATION_SIGNAL is a registered signal watch.
const OPERATION_SIGNAL Operation = 4

// OPERATION_SPAWN is a started child the backend has not yet reaped.
const OPERATION_SPAWN Operation = 5

// OPERATION_NEXT_TICK is a deferred callback carrying no kernel work.
const OPERATION_NEXT_TICK Operation = 6

// OPERATION_EVENT is an armed cross-thread event listener.
const OPERATION_EVENT Operation = 7

// Timeline is the event loop's own submit surface — the control plane every backend fills and
// every operation retires through. Timers, deferred callbacks, and the cross-thread wakeup
// live here rather than on an IO or an OS surface, because none of the three transfers bytes
// with an endpoint: each one only decides WHEN a completion runs, and when is this package's
// subject.
//
// A backend — the deterministic simulator, kqueue, io_uring — fills this vtable and returns a
// Driver beside it. Code that holds a Timeline can arm work and can never advance it.
type Timeline struct {
	// Submit arms completion to retire delay from now, in Ready_At order. It is what every
	// other backend surface — a read, a socket accept, a spawn — schedules through, so one
	// queue holds the whole order.
	Submit func(completion *Completion, delay Duration, callback func())
	// Classify records what an armed completion waits for, so Introspect can report a stall
	// by class. A backend that submits through this vtable states its own class; a submit
	// that states none counts as ready work.
	Classify func(completion *Completion, operation Operation)
	// Report_Descriptors registers the reader of how many raw descriptors a submitter holds
	// open. The census is one struct, and a descriptor count is knowledge the loop does not
	// have, so the code that holds descriptors hands the loop a reader for them.
	Report_Descriptors func(reader func() (count int))
	// Timeout fires callback after the duration on the clock, off the same queue every other
	// completion uses. The duration must be positive; a caller that wants to yield uses
	// Next_Tick.
	Timeout func(completion *Completion, callback Timeout_Callback, duration Duration)
	// Next_Tick defers a callback without kernel IO, matching io/linux.zig:332-352 and
	// io/darwin.zig:757-781.
	Next_Tick func(
		completion *Completion, callback Next_Tick_Callback, source Next_Tick_Source,
	)
	// Reset_Next_Tick removes every queued next-tick completion for source without delivery.
	Reset_Next_Tick func(source Next_Tick_Source)
	// Open_Event creates TigerBeetle's platform Event primitive.
	Open_Event func() (event Event, err error)
	// Event_Listen arms completion for one Event notification.
	Event_Listen func(event Event, completion *Completion, callback Next_Tick_Callback)
	// Event_Trigger makes an armed Event completion ready. It is the only operation safe to
	// call from another thread.
	Event_Trigger func(event Event, completion *Completion)
	// Close_Event releases an Event after its listener has drained.
	Close_Event func(event Event)
}

// Timeline_Invariants states that every slot is filled. A Timeline is a vtable, so the zero
// Timeline reads as a Timeline and panics on first use, and a backend that fills nine slots
// and forgets the tenth is the same failure one call later.
func Timeline_Invariants(loop Timeline, namespace invariant.Namespace) {
	invariant.Always(loop.Submit != nil, "A Timeline arms a completion.")
	invariant.Always(loop.Classify != nil, "A Timeline classifies an armed completion.")
	invariant.Always(
		loop.Report_Descriptors != nil, "A Timeline takes a descriptor-count reader.")
	invariant.Always(loop.Timeout != nil, "A Timeline fires a timer.")
	invariant.Always(loop.Next_Tick != nil, "A Timeline defers a callback.")
	invariant.Always(loop.Reset_Next_Tick != nil, "A Timeline drops a next-tick source.")
	invariant.Always(loop.Open_Event != nil, "A Timeline opens a cross-thread event.")
	invariant.Always(loop.Event_Listen != nil, "A Timeline listens for that event.")
	invariant.Always(loop.Event_Trigger != nil, "A Timeline triggers that event.")
	invariant.Always(loop.Close_Event != nil, "A Timeline closes that event.")
}

// Driver advances the loop — the only capability that moves time and delivers completions.
//
// ===========================================================================
// ONLY PACKAGE MAIN OR A TEST MAY DRIVE, RUN, OR TICK THE EVENT LOOP.
// NOT A LIBRARY. NOT A HELPER. NOT AN INJECTED FUNC VALUE. NOT ONCE.
// A VIOLATION IS AN ARCHITECTURAL BUG EVEN IF EVERY TEST PASSES.
// ===========================================================================
//
// Only the code that constructed the Driver may hold or call it: a binary's package main in
// production, a test harness in simulation. A library that pumps works while its binary owns
// the whole process, and it fails where it composes: assembled with others, it delivers every
// other application's completions from inside its own call stack, destroying the absolute
// order the assembly exists to hold.
type Driver struct {
	// Run drains every ready completion without blocking, then advances the clock one
	// tick (TigerBeetle IO.run). ROOT ONLY: never handed to, or called from, a library.
	Run func() (err error)
	// Run_For drives the loop until the duration has elapsed on the clock, delivering
	// completions as they come due (TigerBeetle IO.run_for_ns). Here time is the GOAL: it
	// advances exactly duration, draining as it goes, regardless of what completes — reach
	// for it to let a span of time pass, not to wait for a particular op.
	// ROOT ONLY: never handed to, or called from, a library.
	Run_For func(duration Duration) (err error)
	// Run_Until drives the loop until done reports true — the run-until-complete pump that
	// lets straight-line code wait for its own op inline. Here completion is the GOAL and
	// time is the GUARD: it stops the instant done holds, and timeout only caps the wait so
	// a stalled op can't hang the caller. This opposite emphasis — completion-first with a
	// time bound, versus Run_For's time-first — is why the two stay separate ops.
	//
	// timeout < 0 (FOREVER) waits unbounded — a server pumping until a shutdown signal.
	// timeout == 0 (IMMEDIATE) evaluates done once and returns without driving — a poll.
	// timeout > 0 pumps until done or the clock passes now+timeout. completed reports which
	// won: done (true) or the timeout (false).
	//
	// Top-level and single-loop only: never call it from within a completion callback.
	// ROOT ONLY: never inject it — or a func value of its shape — into a library; a
	// library that authors done predicates and timeouts is driving the loop.
	Run_Until func(
		done func() (finished bool), timeout Duration,
	) (completed bool, err error)
	// Deinit releases the backend's kernel resources after every submitted operation is joined.
	Deinit func()
	// Introspect returns a point-in-time census of the loop's internal queues — the depths a
	// stall shows up in. Read-only, safe only on the loop thread, so like the rest of Driver
	// it is ROOT ONLY: the root may sample it (for an admin snapshot); a library may not.
	Introspect func() (counts Timeline_Counts)
}

// Timeline_Counts is a census of a loop backend's internal queues at one instant — how many
// completions are ready to run, how many sockets await readability or writability, how many
// timers and signal watchers are pending, how much cross-thread work is posted back, and how
// many raw descriptors are held open. It is the loop-internals view of an admin state
// snapshot: a stall is usually visible here as a queue that will not drain (a backed-up
// accept, a write that never completes).
type Timeline_Counts struct {
	// Completed is the number of completions whose callbacks are ready to run next drain.
	Completed int
	// Timeouts is the number of pending timer completions.
	Timeouts int
	// IO_Backlog is the number of Darwin operations waiting to enter kqueue.
	IO_Backlog int
	// IO_Inflight is the number of Darwin operations registered with kqueue.
	IO_Inflight int
	// IO_Queued is the number of Linux submissions not yet flushed to the kernel.
	IO_Queued int
	// IO_In_Kernel is the number of Linux submissions awaiting completion.
	IO_In_Kernel int
	// Signal_Waiters is the number of registered signal watchers.
	Signal_Waiters int
	// Spawns is the number of started children the backend has not yet reaped.
	Spawns int
	// Raw_Open is the number of raw descriptors the backend holds open.
	Raw_Open int
}

// Virtual_Timeline is the deterministic loop backend: one ready-time queue, a virtual clock,
// and no kernel at all. New_Virtual_Timeline builds it and never hands it out, so a run
// reproduces from its clock alone and nothing can be scripted into the order.
type Virtual_Timeline struct {
	// Virtual is the clock configuration: the grain each tick advances, the wall-clock
	// origin, and the modeled skew. The timeline owns its clock state directly rather than
	// holding an injected Any_Clock — that indirection is for code outside this package, and
	// the timeline is the source the injected readers are built over.
	Virtual Virtual_Clock
	// Ticks counts how many grains the driver has advanced; "now" is Ticks times the
	// resolution. It lives here, on the driver side, so code holding a Timeline can never
	// advance time.
	Ticks Tick_Count
	// Queue holds the armed completions in Ready_At order, earliest first.
	Queue []*Completion
	// Operations classifies each armed completion for Introspect.
	Operations map[*Completion]Operation
	// Events holds the cross-thread event entries, keyed by handle.
	Events map[Event]*Virtual_Event
	// Listeners holds the completion armed for each event, keyed by handle. It is a map
	// rather than a field on the entry, because the completion is the submitter's and the
	// entry describes the event.
	Listeners map[Event]*Completion
	// Next_Event counts the handles handed out, so each Open_Event returns a distinct
	// nonzero one.
	Next_Event uint64
	// Drive_Active is set while a Run is driving, so a Run called from within a completion
	// callback panics rather than re-entering the driver.
	Drive_Active bool
	// Raw_Open reads how many descriptors a submitter reports holding, for Introspect alone.
	Raw_Open func() (count int)
}

// Virtual_Event is one simulated cross-thread event: whether a listener is armed, and the
// triggers that arrived before one attached. The armed completion itself lives in the loop's
// Listeners map rather than here, because an entry describes the event and the completion
// belongs to whoever submitted it.
type Virtual_Event struct {
	// Armed reports that a listener is attached and waiting for the next trigger.
	Armed bool
	// Triggered counts notifications accumulated before a listener attaches.
	Triggered int
}

// New_Virtual_Timeline returns the deterministic loop, the driver that advances it, and the clock
// its completions are measured against. The driver stays with the root that built it: the
// program under test receives the loop and the clock, NEVER a pump.
func New_Virtual_Timeline(virtual Virtual_Clock) (loop Timeline, driver Driver, clock Any_Clock) {
	Virtual_Clock_Invariants(virtual, "new_virtual_timeline.virtual")
	state := &Virtual_Timeline{
		Virtual:    virtual,
		Operations: map[*Completion]Operation{},
		Events:     map[Event]*Virtual_Event{},
		Listeners:  map[Event]*Completion{},
	}
	loop = virtual_timeline_to_timeline(state)
	Timeline_Invariants(loop, "new_virtual_timeline.loop")
	return loop, virtual_timeline_to_driver(state), virtual_timeline_to_any_clock(state)
}

// Builds the read-only Any_Clock over the timeline's own counter, so what a holder reads is
// exactly what the driver has advanced — one counter, not a second one drifting beside it.
func virtual_timeline_to_any_clock(state *Virtual_Timeline) (clock Any_Clock) {
	defer func() { Any_Clock_Invariants(clock, "virtual_timeline_to_any_clock.clock") }()
	return Any_Clock{
		Now_Monotonic: func() (moment Moment) {
			return virtual_now(state)
		},
		Now_Realtime: func() (moment Moment) {
			now := state.Virtual.Epoch + virtual_now(state)
			if state.Virtual.Skew == nil {
				return now
			}
			return now - Moment(state.Virtual.Skew(state.Ticks))
		},
	}
}

// Wires the control plane onto the vtable every backend and every caller holds.
func virtual_timeline_to_timeline(state *Virtual_Timeline) (loop Timeline) {
	return Timeline{
		Submit: func(completion *Completion, delay Duration, callback func()) {
			virtual_submit(state, completion, delay, callback)
		},
		Classify: func(completion *Completion, operation Operation) {
			state.Operations[completion] = operation
		},
		Report_Descriptors: func(reader func() (count int)) {
			state.Raw_Open = reader
		},
		Timeout: func(
			completion *Completion, callback Timeout_Callback, duration Duration,
		) {
			virtual_timeout(state, completion, callback, duration)
		},
		Next_Tick: func(
			completion *Completion, callback Next_Tick_Callback,
			source Next_Tick_Source,
		) {
			virtual_submit(state, completion, 0, func() { callback(completion) })
			completion.Next_Tick_Source = source
			completion.Next_Tick = true
			state.Operations[completion] = OPERATION_NEXT_TICK
		},
		Reset_Next_Tick: func(source Next_Tick_Source) {
			virtual_reset_next_tick(state, source)
		},
		Open_Event: func() (event Event, err error) {
			state.Next_Event++
			handle := Event(state.Next_Event)
			state.Events[handle] = &Virtual_Event{}
			return handle, nil
		},
		Event_Listen: func(
			event Event, completion *Completion, callback Next_Tick_Callback,
		) {
			virtual_event_listen(state, event, completion, callback)
		},
		Event_Trigger: func(event Event, completion *Completion) {
			virtual_event_trigger(state, event, completion)
		},
		Close_Event: func(event Event) {
			entry := state.Events[event]
			invariant.Always(entry != nil, "A closed event was opened by this loop.")
			invariant.Always(
				!entry.Armed, "An event listener is drained before close.",
			)
			delete(state.Events, event)
		},
	}
}

// Arms one simulated timer. It lives beside the vtable rather than inside it, because the
// vtable literal is the map of the control plane and a body there hides the shape.
func virtual_timeout(
	state *Virtual_Timeline, completion *Completion, callback Timeout_Callback,
	duration Duration,
) {
	invariant.Always(duration > 0, "A timeout duration is positive; yields use Next_Tick.")
	virtual_submit(state, completion, duration, func() { callback(completion, nil) })
	state.Operations[completion] = OPERATION_TIMEOUT
}

// Arms an event listener, delivering at once when a trigger already arrived.
func virtual_event_listen(
	state *Virtual_Timeline, event Event, completion *Completion, callback Next_Tick_Callback,
) {
	entry := state.Events[event]
	invariant.Always(entry != nil, "A listened event was opened by this loop.")
	invariant.Always(!entry.Armed, "An event has at most one armed listener.")
	virtual_arm(state, completion, func() {
		entry.Armed = false
		delete(state.Listeners, event)
		callback(completion)
	})
	entry.Armed = true
	state.Listeners[event] = completion
	state.Operations[completion] = OPERATION_EVENT
	if entry.Triggered > 0 {
		entry.Triggered--
		virtual_enqueue_now(state, completion)
	}
}

// Makes an armed event listener ready, or records the trigger for a later listener.
func virtual_event_trigger(state *Virtual_Timeline, event Event, completion *Completion) {
	entry := state.Events[event]
	invariant.Always(entry != nil, "A triggered event was opened by this loop.")
	if !entry.Armed {
		entry.Triggered++
		return
	}
	invariant.Always(state.Listeners[event] == completion,
		"A trigger names the completion its event armed.")
	virtual_enqueue_now(state, completion)
}

// Reads the simulated moment every Ready_At is measured against.
func virtual_now(state *Virtual_Timeline) (now Moment) {
	return Moment(int64(state.Ticks) * int64(state.Virtual.Resolution))
}

// Schedules completion to fire at now plus delay and inserts it in Ready_At order.
func virtual_submit(
	state *Virtual_Timeline, completion *Completion, delay Duration, callback func(),
) {
	virtual_arm(state, completion, callback)
	completion.Ready_At = virtual_now(state) + Moment(delay)
	virtual_enqueue(state, completion)
}

// Arms completion without placing it on the ready-time queue, for an event listener. It
// asserts the completion is its own original, not a by-value copy, then moves it along the
// lifecycle machine — a completion armed while armed panics on the armed-to-armed edge.
func virtual_arm(state *Virtual_Timeline, completion *Completion, callback func()) {
	original := completion.Self == nil || completion.Self == completion
	invariant.Always(original,
		"A submitted completion is its own original, never a by-value copy.")
	completion.Self = completion
	Completion_Transition(completion, COMPLETION_IDLE, COMPLETION_ARMED)
	completion.Callback = callback
	completion.Next_Tick = false
	state.Operations[completion] = OPERATION_COMPLETED
}

// Places an already armed completion on the queue as due now.
func virtual_enqueue_now(state *Virtual_Timeline, completion *Completion) {
	completion.Ready_At = virtual_now(state)
	virtual_enqueue(state, completion)
}

// Inserts completion into the queue in Ready_At order, earliest first.
func virtual_enqueue(state *Virtual_Timeline, completion *Completion) {
	index := 0
	for index < len(state.Queue) && state.Queue[index].Ready_At <= completion.Ready_At {
		index++
	}
	state.Queue = append(state.Queue, nil)
	copy(state.Queue[index+1:], state.Queue[index:])
	state.Queue[index] = completion
}

// Removes every queued next-tick completion for source and retires it without delivery,
// matching io/linux.zig:354-367 and io/darwin.zig:783-796.
func virtual_reset_next_tick(state *Virtual_Timeline, source Next_Tick_Source) {
	kept := state.Queue[:0]
	for _, completion := range state.Queue {
		if state.Operations[completion] != OPERATION_NEXT_TICK {
			kept = append(kept, completion)
			continue
		}
		if completion.Next_Tick_Source != source {
			kept = append(kept, completion)
			continue
		}
		delete(state.Operations, completion)
		Completion_Transition(completion, COMPLETION_ARMED, COMPLETION_IDLE)
	}
	state.Queue = kept
}

// Fires the earliest completion if it is due as of now, reporting whether it did, mirroring
// TigerBeetle's Storage.step.
func virtual_step(state *Virtual_Timeline) (advanced bool) {
	if len(state.Queue) == 0 {
		return false
	}
	if state.Queue[0].Ready_At > virtual_now(state) {
		return false
	}
	completion := state.Queue[0]
	state.Queue = state.Queue[1:]
	delete(state.Operations, completion)
	// Return to idle before the callback runs — TigerBeetle's ordering — so a callback may
	// legally resubmit its own completion, the repeating-timer pattern.
	Completion_Transition(completion, COMPLETION_ARMED, COMPLETION_IDLE)
	completion.Callback()
	return true
}

// Drains every completion due as of now, in Ready_At order. It never advances time —
// advancing is the driver's job, so the queue itself stays passive.
func virtual_drain(state *Virtual_Timeline) {
	for virtual_step(state) {
	}
}

// The driver's step: drain what is due, then advance the clock one grain, mirroring
// TigerBeetle's Storage.run.
//
// The driver advances one grain per tick and never jumps ahead to the next Ready_At, even
// when the queue is idle until then. A jump would skip the grains where a time-triggered
// fault adversary — crash or partition a quiescent node, the faults that matter most because
// they strike while nothing is scheduled — would act.
func virtual_run(state *Virtual_Timeline) {
	virtual_drain(state)
	state.Ticks++
}

// Drives until the duration has elapsed, delivering completions as they come due.
func virtual_run_for(state *Virtual_Timeline, duration Duration) {
	deadline := virtual_now(state) + Moment(duration)
	for virtual_now(state) < deadline {
		virtual_run(state)
	}
}

// Drives until done reports true, or until timeout of virtual time has elapsed — the
// run-until-complete pump, capped so a stalled operation cannot spin the loop forever.
func virtual_run_until(
	state *Virtual_Timeline, done func() (finished bool), timeout Duration,
) (completed bool) {
	deadline := virtual_now(state) + Moment(timeout)
	for !done() {
		if timeout >= 0 {
			if virtual_now(state) >= deadline {
				return false
			}
		}
		virtual_run(state)
	}
	return true
}

// Runs pump as the top-level drive, asserting no drive is already in progress, so a Run
// called from within a completion callback panics instead of re-entering the driver.
func virtual_drive(state *Virtual_Timeline, pump func()) {
	invariant.Always(!state.Drive_Active,
		"A drive begins at top level, never from within a completion callback.")
	state.Drive_Active = true
	defer func() { state.Drive_Active = false }()
	pump()
}

// Samples every queue class without exposing loop state.
func virtual_introspect(state *Virtual_Timeline) (counts Timeline_Counts) {
	for _, operation := range state.Operations {
		if operation == OPERATION_COMPLETED {
			counts.Completed++
		}
		if operation == OPERATION_TIMEOUT {
			counts.Timeouts++
		}
		if operation == OPERATION_READ_WAITER {
			counts.IO_Backlog++
		}
		if operation == OPERATION_WRITE_WAITER {
			counts.IO_Backlog++
		}
		if operation == OPERATION_SIGNAL {
			counts.Signal_Waiters++
		}
		if operation == OPERATION_SPAWN {
			counts.Spawns++
		}
	}
	if state.Raw_Open != nil {
		counts.Raw_Open = state.Raw_Open()
	}
	return counts
}

// Builds the driver over state — the advancing capability, held only by main or a test.
func virtual_timeline_to_driver(state *Virtual_Timeline) (driver Driver) {
	return Driver{
		Run: func() (err error) {
			virtual_drive(state, func() { virtual_run(state) })
			return nil
		},
		Run_For: func(duration Duration) (err error) {
			virtual_drive(state, func() { virtual_run_for(state, duration) })
			return nil
		},
		Run_Until: func(
			done func() (finished bool), timeout Duration,
		) (completed bool, err error) {
			virtual_drive(state, func() {
				completed = virtual_run_until(state, done, timeout)
			})
			return completed, nil
		},
		Deinit:     func() {},
		Introspect: func() (counts Timeline_Counts) { return virtual_introspect(state) },
	}
}
