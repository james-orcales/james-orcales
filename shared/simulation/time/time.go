// Package time gives time as an injected dependency. The backend is a vtable of closures.
// Production wires an OS clock (time/default). A simulation wires a Virtual one. The code
// between never knows which one it holds. The Virtual backend lives here because it is pure
// arithmetic. It makes no operating-system call.
package time

import (
	"errors"

	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/fixedpoint"
)

// INTEGER_64_MINIMUM is the smallest signed 64-bit integer. A Moment, a Duration, and a
// Tick_Count each fill the whole signed 64-bit range. A virtual clock reads its epoch, its
// skew, and its tick count as the caller sets them. No one of the three has a bound more
// narrow than its storage.
const INTEGER_64_MINIMUM int64 = -9223372036854775808

// INTEGER_64_MAXIMUM is the largest signed 64-bit integer.
const INTEGER_64_MAXIMUM int64 = 9223372036854775807

// Moment is a clock reading in nanoseconds. Its epoch is arbitrary and belongs to one clock.
// Only the difference between two Moments from the SAME clock has a meaning. A monotonic
// Moment and a realtime Moment do not compare.
type Moment int64

// Moment_Invariants state complete clock-reading domain.
func Moment_Invariants(moment Moment, namespace invariant.Namespace) {
	invariant.Tree(moment, namespace).
		Range_Int64(int64(moment), INTEGER_64_MINIMUM, INTEGER_64_MAXIMUM).
		Ensure()
}

// Duration: span of nanoseconds.
type Duration int64

// Duration_Invariants state complete nanosecond-span domain.
func Duration_Invariants(duration Duration, namespace invariant.Namespace) {
	invariant.Tree(duration, namespace).
		Range_Int64(int64(duration), INTEGER_64_MINIMUM, INTEGER_64_MAXIMUM).
		Ensure()
}

// NANOSECOND: unit Duration count in.
const NANOSECOND Duration = 1

// MICROSECOND: thousand nanoseconds.
const MICROSECOND = NANOSECOND * 1000

// MILLISECOND: thousand microseconds.
const MILLISECOND = MICROSECOND * 1000

// SECOND: thousand milliseconds.
const SECOND = MILLISECOND * 1000

// MINUTE: sixty seconds.
const MINUTE = SECOND * 60

// HOUR: sixty minutes.
const HOUR = MINUTE * 60

// DAY: twenty-four hours.
const DAY = HOUR * 24

// WEEK: seven days.
const WEEK = DAY * 7

// Monotonic_Moment: clock reading in nanoseconds. Count from machine boot (Linux
// CLOCK_BOOTTIME). Zero mean boot. Reading mean uptime. Bound can hold uptime. Plain Moment
// have no origin. Plain Moment take full signed range.
type Monotonic_Moment int64

// MONOTONIC_MOMENT_MINIMUM: zero. Clock restart at zero on each boot. Reading below zero mean
// clock go backward.
const MONOTONIC_MOMENT_MINIMUM Monotonic_Moment = 0

// MONOTONIC_MOMENT_MAXIMUM: one year of uptime. Many thing stop machine before one year.
// Chaos engineering kill it on purpose. Operating system update reboot it. More uptime than
// this mean defect, not data point.
const MONOTONIC_MOMENT_MAXIMUM Monotonic_Moment = Monotonic_Moment(DAY * 365)

// Monotonic_Moment_Invariants state complete uptime domain.
func Monotonic_Moment_Invariants(moment Monotonic_Moment, namespace invariant.Namespace) {
	invariant.Tree(moment, namespace).
		Range_Int64(
			int64(moment),
			int64(MONOTONIC_MOMENT_MINIMUM),
			int64(MONOTONIC_MOMENT_MAXIMUM)).
		Ensure()
}

// Clock: injected time source. Vtable of closures. Caller pick backend by value.
// Read-only. Driver move time, with tick returned beside clock. Holder read current Moment.
// Holder never move time.
type Clock struct {
	// Now_Monotonic read monotonic clock. Never go backward. Use for elapsed time, timeout,
	// latency.
	Now_Monotonic func() (moment Monotonic_Moment)
	// Now_Realtime read wall clock as nanoseconds from Unix epoch. Can jump. Use for calendar
	// timestamp only, never for elapsed time.
	Now_Realtime func() (moment Moment)
}

// Clock_Invariants state both readers bound. Clock is vtable. One property only:
// every slot full. Zero Clock read as Clock, then panic on first use. Backend that
// fill one slot and forget other fail one call later.
func Clock_Invariants(clock Clock, namespace invariant.Namespace) {
	invariant.Always(
		clock.Now_Monotonic != nil, "A Clock has a monotonic reader.",
	)
	invariant.Always(
		clock.Now_Realtime != nil, "A Clock has a realtime reader.",
	)
}

// Tick_Count: how many time virtual clock advance. It is x in each skew formula.
type Tick_Count int64

// Tick_Count_Invariants state complete tick-count domain.
func Tick_Count_Invariants(ticks Tick_Count, namespace invariant.Namespace) {
	invariant.Tree(ticks, namespace).
		Range_Int64(int64(ticks), INTEGER_64_MINIMUM, INTEGER_64_MAXIMUM).
		Ensure()
}

// Offset model how simulated wall clock drift from true elapsed time. Offset make
// Now_Realtime differ from Now_Monotonic. Nil Offset mean perfect clock.
type Offset func(ticks Tick_Count) (skew Duration)

// Virtual_Clock configure deterministic clock that Virtual_Clock_To_Clock build. Time
// advance only when Tick run. Simulation reach future Moment by tick, never by wait.
type Virtual_Clock struct {
	// Resolution: how far monotonic clock advance on each Tick. Grain of simulated
	// oscillator.
	Resolution Duration
	// Epoch: wall-clock origin. Now_Realtime at tick zero, before skew.
	Epoch Moment
	// Skew bend Now_Realtime away from true elapsed time. Nil Skew mean perfect clock.
	Skew Offset
}

// Virtual_Clock_Invariants state two scalars of virtual clock. Skew is closure. Closure
// arithmetic have no domain to state here. Skew state own coefficients where Skew build them.
func Virtual_Clock_Invariants(virtual Virtual_Clock, namespace invariant.Namespace) {
	Duration_Invariants(virtual.Resolution, namespace)
	Moment_Invariants(virtual.Epoch, namespace)
}

// Virtual_Clock_To_Clock return read-only Clock plus tick that advance it. Clock
// behind is deterministic. Make no operating-system call. Closures and tick share one counter,
// thus tick move what next Now_Monotonic read. Only driver hold tick: package main, or test
// harness. Pure code hold Clock alone. Pure code read time, never move it.
func Virtual_Clock_To_Clock(virtual Virtual_Clock) (clock Clock, tick func()) {
	defer func() { Clock_Invariants(clock, "virtual_clock_to_clock.clock") }()
	Virtual_Clock_Invariants(virtual, "virtual_clock_to_clock.virtual")
	ticks := Tick_Count(0)
	clock = Clock{
		Now_Monotonic: func() (moment Monotonic_Moment) {
			uptime := Monotonic_Moment(int64(ticks) * int64(virtual.Resolution))
			Monotonic_Moment_Invariants(uptime, "virtual_clock_to_clock.uptime")
			return uptime
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

// SKEW_KIND_LINEAR model constant drift. A nanoseconds of skew per tick, plus initial B
// (A*x + B, x is tick count).
const SKEW_KIND_LINEAR Skew_Kind = 0

// SKEW_KIND_PERIODIC model sinusoidal wobble. Amplitude A over period of B ticks
// (A*sin(x*2pi/B)).
const SKEW_KIND_PERIODIC Skew_Kind = 1

// SKEW_KIND_STEP model jump of A after B ticks. NTP correction, or operator change clock.
const SKEW_KIND_STEP Skew_Kind = 2

// Skew_Kind pick which clock-deviation model Skew build.
type Skew_Kind uint8

// Skew_Kind_Invariants hold kind to three models that Skew build. Default arm of that switch
// is linear model. Unlisted kind would drift in silence, not fail.
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

// Skew build Offset for one deviation model. Take kind plus two coefficients. Coefficient a
// is magnitude: drift-per-tick, amplitude, or step size. Coefficient b count ticks: linear
// initial offset, periodic period, or onset tick of step.
func Skew(kind Skew_Kind, a Duration, b Tick_Count) (offset Offset) {
	Skew_Kind_Invariants(kind, "skew.kind")
	Duration_Invariants(a, "skew.a")
	Tick_Count_Invariants(b, "skew.b")
	switch kind {
	case SKEW_KIND_PERIODIC:
		return func(ticks Tick_Count) (skew Duration) {
			// Zero period mean degenerate sinusoid. Report no skew. Division or
			// remainder by zero panic.
			if b == 0 {
				return 0
			}
			// Cut phase to one period before lift into fixed-point. Long run else
			// overflow scaled numerator.
			phase := ticks % b
			turns := fixedpoint.From_Ratio(
				fixedpoint.Numerator(phase),
				fixedpoint.Denominator(b),
			)
			amplitude := fixedpoint.Number(fixedpoint.From_Integer(
				fixedpoint.Whole_Integer(a),
			))
			wobble := fixedpoint.Multiply(
				fixedpoint.Multiplicand(amplitude),
				fixedpoint.Multiplier(fixedpoint.Sine_Turns(turns)),
			)
			return Duration(fixedpoint.Whole(wobble))
		}
	case SKEW_KIND_STEP:
		return func(ticks Tick_Count) (skew Duration) {
			if ticks > b {
				return a
			}
			return 0
		}
	default:
		return func(ticks Tick_Count) (skew Duration) {
			return Duration(ticks)*a + Duration(b)
		}
	}
}

// Callback receives the caller-owned completion after the backend retires its operation.
type Callback func(completion *Completion)

// Retired_Twice report backend retire one completion more than one time. Derived function
// deliver it, never hide it. Caller own completion. Caller must learn lifecycle broke.
var Retired_Twice = errors.New("time: the completion retired more than once")

// Deadline_Exceeded come back when finite operation retire without its external event.
var Deadline_Exceeded = errors.New("time: deadline exceeded")

// Completion: caller-owned storage for one in-flight operation. Caller allocate it, thus loop
// never allocate. Caller keep it alive until callback fire.
type Completion struct {
	// Data is an opaque int whose submitting operation defines. A transfer stores its byte
	// count, while an open or accept stores its descriptor.
	Data int
	// Error is nil on success and otherwise stores the operation failure.
	Error error
	// Callback closes over backend retirement work that must run before the public callback.
	Callback func()
	// Ready_At: uptime this operation complete at. Sit on monotonic timeline. Realtime jump
	// must not retire operation early, or hold it late.
	Ready_At Monotonic_Moment
	// Armed: completion is in flight. False mean never submitted, or delivered and free
	// again. Backend own it: it flip true on submit, false before delivery. Application never
	// read it, never write it. Show own state, not state of completion.
	Armed bool
	// Self: address of completion. First submit stamp it. Nothing clear it. Timeline track
	// in-flight operation by pointer, thus by-value copy carry this original address. Submit
	// of copy trip backend assertion. Without it, view of timeline and view of caller split
	// in silence. Only backend touch it.
	Self *Completion
	// Kernel_Identifier: generation token in kqueue udata or io_uring user_data. Backend own
	// it. Event_Trigger read it only after Event_Listen arm completion.
	Kernel_Identifier uint64
}

// Event: cross-thread wakeup handle of backend. kqueue EVFILT_USER ident, or eventfd
// descriptor. Primitive of loop. Carry no bytes. Name no endpoint. One purpose: make armed
// completion ready from other thread.
type Event uintptr

// Timeline: submit surface of event loop. Control plane every backend fill, every operation
// retire through. Timer and cross-thread wakeup live here, not on IO or OS surface. Neither
// one move bytes with endpoint. Each one decide WHEN completion run. When is subject of this
// package.
//
// Backend fill this vtable and return Driver beside it: deterministic simulator, kqueue, or
// io_uring. Code that hold Timeline arm work, never advance it.
type Timeline struct {
	// Submit arm completion to retire one delay from now, in Ready_At order. Every other
	// backend surface schedule through it: read, socket accept, spawn. One queue thus hold
	// full order.
	Submit func(completion *Completion, delay Duration, callback Callback)
	// Timeout fire callback after duration on clock, off same queue every other completion
	// use. Duration must be positive.
	Timeout func(completion *Completion, duration Duration, callback Callback)
	// Open_Event make platform Event primitive.
	Open_Event func() (event Event, err error)
	// Event_Listen arm completion for one Event notification.
	Event_Listen func(
		event Event, completion *Completion, callback Callback,
	)
	// Event_Trigger make armed Event completion ready. Only operation safe to call from other
	// thread.
	Event_Trigger func(event Event, completion *Completion)
	// Close_Event release Event after listener drain.
	Close_Event func(event Event)
}

// Timeline_Invariants state every slot full. Timeline is vtable. Zero Timeline read as
// Timeline, then panic on first use. Backend that fill five slots and forget sixth fail one
// call later.
func Timeline_Invariants(loop Timeline, namespace invariant.Namespace) {
	invariant.Always(loop.Submit != nil, "A Timeline arms a completion.")
	invariant.Always(loop.Timeout != nil, "A Timeline fires a timer.")
	invariant.Always(loop.Open_Event != nil, "A Timeline opens a cross-thread event.")
	invariant.Always(loop.Event_Listen != nil, "A Timeline listens for that event.")
	invariant.Always(loop.Event_Trigger != nil, "A Timeline triggers that event.")
	invariant.Always(loop.Close_Event != nil, "A Timeline closes that event.")
}

// Driver advance loop. Only capability that move time and deliver completions.
//
// ===========================================================================
// ONLY PACKAGE MAIN OR A TEST MAY DRIVE, RUN, OR TICK THE EVENT LOOP.
// NOT A LIBRARY. NOT A HELPER. NOT AN INJECTED FUNC VALUE. NOT ONCE.
// A VIOLATION IS AN ARCHITECTURAL BUG EVEN IF EVERY TEST PASSES.
// ===========================================================================
//
// Only code that build Driver hold it or call it: package main in production, or test harness
// in simulation. Library that pump work while its binary own full process. Library fail where
// it compose. Put together with others, it deliver completions of every other application
// from inside own call stack. That destroy absolute order assembly exist to hold.
type Driver struct {
	// Run drain every ready completion without block, then advance clock one tick. ROOT ONLY:
	// never hand to library, never call from library.
	Run func() (err error)
	// Run_For drive loop until duration elapse on clock. Deliver each completion as it come
	// due. Time is GOAL here. Advance exactly duration, drain as it go, whatever complete.
	// Use to let span of time pass, not to wait for one operation.
	// ROOT ONLY: never hand to library, never call from library.
	Run_For func(duration Duration) (err error)
	// Run_Until drive loop until done report true. Run-until-complete pump. Straight-line
	// code wait for own operation inline with it. Completion is GOAL here. Time is GUARD.
	// Stop instant done hold. Timeout only cap wait, thus stalled operation cannot hang
	// caller. Run_For put time first, thus two stay separate operations.
	//
	// timeout < 0 panic: unbounded pump put no cap on stalled operation.
	// timeout == 0 check done one time, return without drive. Poll.
	// timeout > 0 pump until done, or until clock pass now+timeout. completed report which
	// win: done (true), or timeout (false).
	//
	// Top-level and single-loop only: never call from inside completion callback.
	// ROOT ONLY: never inject it into library, and never inject func value of its shape.
	// Library that write done predicate and timeout is driving loop.
	Run_Until func(
		timeout Duration, done func() (finished bool),
	) (completed bool, err error)
	// Deinit release kernel resources of backend, after every submitted operation join.
	Deinit func()
}

// Virtual_Timeline: deterministic loop backend. One ready-time queue, one virtual clock, no
// kernel. New_Virtual_Timeline build it, never hand it out. Run thus reproduce from clock
// alone. Nothing can script order.
type Virtual_Timeline struct {
	// Virtual: clock configuration. Grain each tick advance, wall-clock origin, modeled skew.
	// Timeline own clock state direct, hold no injected Clock. That indirection is for
	// code outside this package. Timeline is source injected readers build over.
	Virtual Virtual_Clock
	// Ticks count how many grains driver advance. Now is Ticks times resolution. Live here,
	// on driver side, thus code that hold Timeline never advance time.
	Ticks Tick_Count
	// Queue hold armed completions in Ready_At order, earliest first.
	Queue []*Completion
	// Events hold cross-thread event entries, keyed by handle.
	Events map[Event]*Virtual_Event
	// Listeners hold completion armed for each event, keyed by handle. Map, not field on
	// entry: completion belong to submitter, entry describe event.
	Listeners map[Event]*Completion
	// Next_Event count handles handed out, thus each Open_Event return different handle,
	// never zero.
	Next_Event uint64
	// Drive_Active true while Run drive. Run called from inside completion callback thus
	// panic, never re-enter driver.
	Drive_Active bool
}

// Virtual_Event: one simulated cross-thread event. Hold whether listener armed, plus triggers
// that arrive before one attach. Armed completion live in Listeners map of loop, not here:
// entry describe event, completion belong to submitter.
type Virtual_Event struct {
	// Armed report listener attached and wait for next trigger.
	Armed bool
	// Triggered count notifications piled up before listener attach.
	Triggered int
}

// New_Virtual_Timeline return three thing: deterministic loop, driver that advance it, clock
// its completions measure against. Driver stay with root that build it. Program under test
// get loop and clock, NEVER pump.
func New_Virtual_Timeline(virtual Virtual_Clock) (loop Timeline, driver Driver, clock Clock) {
	Virtual_Clock_Invariants(virtual, "new_virtual_timeline.virtual")
	state := &Virtual_Timeline{
		Virtual:   virtual,
		Events:    map[Event]*Virtual_Event{},
		Listeners: map[Event]*Completion{},
	}
	loop = virtual_timeline_to_timeline(state)
	Timeline_Invariants(loop, "new_virtual_timeline.loop")
	return loop, virtual_timeline_to_driver(state), virtual_timeline_to_clock(state)
}

// Build read-only Clock over own counter of timeline. Holder thus read exactly what driver
// advance. One counter, not second one that drift beside it.
func virtual_timeline_to_clock(state *Virtual_Timeline) (clock Clock) {
	defer func() { Clock_Invariants(clock, "virtual_timeline_to_clock.clock") }()
	return Clock{
		Now_Monotonic: func() (moment Monotonic_Moment) {
			return virtual_now(state)
		},
		Now_Realtime: func() (moment Moment) {
			now := state.Virtual.Epoch + Moment(virtual_now(state))
			if state.Virtual.Skew == nil {
				return now
			}
			return now - Moment(state.Virtual.Skew(state.Ticks))
		},
	}
}

// Wire control plane onto vtable every backend and every caller hold.
func virtual_timeline_to_timeline(state *Virtual_Timeline) (loop Timeline) {
	return Timeline{
		Submit: func(completion *Completion, delay Duration, callback Callback) {
			virtual_submit(state, completion, delay, callback)
		},
		Timeout: func(
			completion *Completion, duration Duration, callback Callback,
		) {
			virtual_timeout(state, completion, duration, callback)
		},
		Open_Event: func() (event Event, err error) {
			state.Next_Event++
			handle := Event(state.Next_Event)
			state.Events[handle] = &Virtual_Event{}
			return handle, nil
		},
		Event_Listen: func(
			event Event, completion *Completion, callback Callback,
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

// Arm one simulated timer. Live beside vtable, not inside it: vtable literal is map of
// control plane, and body there hide that shape.
func virtual_timeout(
	state *Virtual_Timeline, completion *Completion, duration Duration,
	callback Callback,
) {
	invariant.Always(duration > 0, "A timeout duration is positive.")
	virtual_submit(state, completion, duration, callback)
}

// Arm event listener. Deliver at once when trigger already arrive.
func virtual_event_listen(
	state *Virtual_Timeline, event Event, completion *Completion,
	callback Callback,
) {
	entry := state.Events[event]
	invariant.Always(entry != nil, "A listened event was opened by this loop.")
	invariant.Always(!entry.Armed, "An event has at most one armed listener.")
	virtual_arm(state, completion, func(completion *Completion) {
		entry.Armed = false
		delete(state.Listeners, event)
		callback(completion)
	})
	entry.Armed = true
	state.Listeners[event] = completion
	if entry.Triggered > 0 {
		entry.Triggered--
		virtual_enqueue_now(state, completion)
	}
}

// Make armed event listener ready, or record trigger for later listener.
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

// Read simulated moment every Ready_At measure against.
func virtual_now(state *Virtual_Timeline) (now Monotonic_Moment) {
	now = Monotonic_Moment(int64(state.Ticks) * int64(state.Virtual.Resolution))
	Monotonic_Moment_Invariants(now, "virtual_now.now")
	return now
}

// Schedule completion to fire at now plus delay. Insert it in Ready_At order.
func virtual_submit(
	state *Virtual_Timeline, completion *Completion, delay Duration, callback Callback,
) {
	virtual_arm(state, completion, callback)
	completion.Ready_At = virtual_now(state) + Monotonic_Moment(delay)
	virtual_enqueue(state, completion)
}

// Arm completion, but never put it on ready-time queue. Event listener use this. Assert
// completion is own original, not by-value copy. Then move it along lifecycle machine.
// Completion armed while armed panic on armed-to-armed edge.
func virtual_arm(state *Virtual_Timeline, completion *Completion, callback Callback) {
	original := completion.Self == nil || completion.Self == completion
	invariant.Always(original,
		"A submitted completion is its own original, never a by-value copy.")
	completion.Self = completion
	invariant.Always(!completion.Armed, "An armed completion is never armed a second time.")
	completion.Data = 0
	completion.Error = nil
	completion.Armed = true
	completion.Callback = func() { callback(completion) }
}

// Put already armed completion on queue as due now.
func virtual_enqueue_now(state *Virtual_Timeline, completion *Completion) {
	completion.Ready_At = virtual_now(state)
	virtual_enqueue(state, completion)
}

// Insert completion into queue in Ready_At order, earliest first.
func virtual_enqueue(state *Virtual_Timeline, completion *Completion) {
	index := 0
	for index < len(state.Queue) && state.Queue[index].Ready_At <= completion.Ready_At {
		index++
	}
	state.Queue = append(state.Queue, nil)
	copy(state.Queue[index+1:], state.Queue[index:])
	state.Queue[index] = completion
}

// Fire earliest completion when due as of now. Report whether it fire.
func virtual_step(state *Virtual_Timeline) (advanced bool) {
	if len(state.Queue) == 0 {
		return false
	}
	if state.Queue[0].Ready_At > virtual_now(state) {
		return false
	}
	completion := state.Queue[0]
	state.Queue = state.Queue[1:]
	// Go back to idle before callback run. Callback can then submit own completion again.
	// Repeating-timer pattern.
	invariant.Always(completion.Armed, "A delivered completion was armed.")
	completion.Armed = false
	completion.Callback()
	return true
}

// Drain every completion due as of now, in Ready_At order. Never advance time. That is job of
// driver, thus queue stay passive.
func virtual_drain(state *Virtual_Timeline) {
	for virtual_step(state) {
	}
}

// Step of driver: drain what is due, then advance clock one grain.
//
// Driver advance one grain per tick. Never jump ahead to next Ready_At, even when queue idle
// until then. Jump would skip grains where time-triggered fault adversary act. Those faults
// crash or partition quiet node. They matter most: they strike while nothing scheduled.
func virtual_run(state *Virtual_Timeline) {
	virtual_drain(state)
	state.Ticks++
}

// Drive until duration elapse. Deliver completions as they come due.
func virtual_run_for(state *Virtual_Timeline, duration Duration) {
	deadline := virtual_now(state) + Monotonic_Moment(duration)
	for virtual_now(state) < deadline {
		virtual_run(state)
	}
}

// Drive until done report true, or until timeout of virtual time elapse. Run-until-complete
// pump. Cap stop stalled operation from spin without end. Negative timeout is that uncapped
// pump, thus it panic. Never hand caller drive with no bound.
func virtual_run_until(
	state *Virtual_Timeline, timeout Duration, done func() (finished bool),
) (completed bool) {
	invariant.Always(timeout >= 0, "A Run_Until timeout is never negative.")
	deadline := virtual_now(state) + Monotonic_Moment(timeout)
	for !done() {
		if virtual_now(state) >= deadline {
			return false
		}
		virtual_run(state)
	}
	return true
}

// Run pump as top-level drive. Assert no drive already in progress. Run called from inside
// completion callback thus panic, never re-enter driver.
func virtual_drive(state *Virtual_Timeline, pump func()) {
	invariant.Always(!state.Drive_Active,
		"A drive begins at top level, never from within a completion callback.")
	state.Drive_Active = true
	defer func() { state.Drive_Active = false }()
	pump()
}

// Build driver over state. Capability that advance time. Only main or test hold it.
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
			timeout Duration, done func() (finished bool),
		) (completed bool, err error) {
			virtual_drive(state, func() {
				completed = virtual_run_until(state, timeout, done)
			})
			return completed, nil
		},
		Deinit: func() {},
	}
}
