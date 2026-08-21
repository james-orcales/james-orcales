// Package time gives time as an injected dependency. The backend is a vtable of procedures
// over explicit caller-owned state.
// Production wires an OS clock (time/default). A simulation wires a Virtual one. The code
// between never knows which one it holds. The Virtual backend lives here because it is pure
// arithmetic. It makes no operating-system call.
package time

import (
	"errors"
	"unsafe"

	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/fixedpoint"
	"local/james-orcales/shared/slices"
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

// Clock: injected time source. Backend state stays explicit so the vtable never capture it.
type Clock struct {
	// State stays caller-owned because captured backend state would escape with this vtable.
	State unsafe.Pointer
	// Now_Monotonic read monotonic clock. Never go backward. Use for elapsed time, timeout,
	// latency.
	Now_Monotonic func(state unsafe.Pointer) (moment Monotonic_Moment)
	// Now_Realtime read wall clock as nanoseconds from Unix epoch. Can jump. Use for calendar
	// timestamp only, never for elapsed time.
	Now_Realtime func(state unsafe.Pointer) (moment Moment)
}

// Clock_Now_Monotonic passes state explicitly because a bound reader would allocate.
func Clock_Now_Monotonic(clock Clock) (moment Monotonic_Moment) {
	return clock.Now_Monotonic(clock.State)
}

// Clock_Now_Realtime passes state explicitly because a bound reader would allocate.
func Clock_Now_Realtime(clock Clock) (moment Moment) {
	return clock.Now_Realtime(clock.State)
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

// Offset keeps coefficients by value so a skew reader needs no separately owned state.
type Offset struct {
	// Kind permits static evaluator dispatch, avoiding captured procedure state.
	Kind Skew_Kind
	// A shares the lifetime of the model that owns it.
	A Duration
	// B shares the lifetime of the model that owns it.
	B Tick_Count
}

// Virtual_Clock configure deterministic clock that Virtual_Clock_To_Clock build. Time
// advance only when Tick run. Simulation reach future Moment by tick, never by wait.
type Virtual_Clock struct {
	// Resolution: how far monotonic clock advance on each Tick. Grain of simulated
	// oscillator.
	Resolution Duration
	// Epoch: wall-clock origin. Now_Realtime at tick zero, before skew.
	Epoch Moment
	// Skew bend Now_Realtime away from true elapsed time. Zero Skew mean perfect clock.
	Skew Offset
	// Ticks stays caller-owned because both readers and the root advance the same counter.
	Ticks Tick_Count
}

// Virtual_Clock_Invariants leaves skew coefficients to their own typed invariant chain.
func Virtual_Clock_Invariants(virtual Virtual_Clock, namespace invariant.Namespace) {
	Duration_Invariants(virtual.Resolution, namespace)
	Moment_Invariants(virtual.Epoch, namespace)
}

// Virtual_Clock_To_Clock binds caller-owned state without a captured function environment.
func Virtual_Clock_To_Clock(virtual *Virtual_Clock) (clock Clock) {
	invariant.Always(virtual != nil, "A virtual clock has caller-owned state.")
	Virtual_Clock_Invariants(*virtual, "virtual_clock_to_clock.virtual")
	virtual.Ticks = 0
	clock = Clock{
		State:         unsafe.Pointer(virtual),
		Now_Monotonic: virtual_clock_now_monotonic,
		Now_Realtime:  virtual_clock_now_realtime,
	}
	Clock_Invariants(clock, "virtual_clock_to_clock.clock")
	return clock
}

// Virtual_Clock_Tick keeps advancement with the root that owns mutable clock state.
func Virtual_Clock_Tick(virtual *Virtual_Clock) {
	invariant.Always(virtual != nil, "A tick advances caller-owned virtual-clock state.")
	virtual.Ticks++
}

func virtual_clock_now_monotonic(state unsafe.Pointer) (moment Monotonic_Moment) {
	virtual := (*Virtual_Clock)(state)
	uptime := Monotonic_Moment(int64(virtual.Ticks) * int64(virtual.Resolution))
	Monotonic_Moment_Invariants(uptime, "virtual_clock_to_clock.uptime")
	return uptime
}

func virtual_clock_now_realtime(state unsafe.Pointer) (moment Moment) {
	virtual := (*Virtual_Clock)(state)
	now := virtual.Epoch + Moment(int64(virtual.Ticks)*int64(virtual.Resolution))
	return now - Moment(Offset_Read(virtual.Skew, virtual.Ticks))
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
	return Offset{Kind: kind, A: a, B: b}
}

// Offset_Read evaluates stored coefficients without closure state.
func Offset_Read(offset Offset, ticks Tick_Count) (skew Duration) {
	switch offset.Kind {
	case SKEW_KIND_PERIODIC:
		// Zero period reports no skew because division would panic.
		if offset.B == 0 {
			return 0
		}
		// Phase reduction prevents scaled numerator overflow during long runs.
		phase := ticks % offset.B
		turns := fixedpoint.From_Ratio(
			fixedpoint.Numerator(phase),
			fixedpoint.Denominator(offset.B),
		)
		amplitude := fixedpoint.Number(fixedpoint.From_Integer(
			fixedpoint.Whole_Integer(offset.A),
		))
		wobble := fixedpoint.Multiply(
			fixedpoint.Multiplicand(amplitude),
			fixedpoint.Multiplier(fixedpoint.Sine_Turns(turns)),
		)
		return Duration(fixedpoint.Whole(wobble))
	case SKEW_KIND_STEP:
		if ticks > offset.B {
			return offset.A
		}
		return 0
	default:
		return Duration(ticks)*offset.A + Duration(offset.B)
	}
}

// Callback receives the caller-owned completion after the backend retires its operation.
type Callback func(completion *Completion)

// Retired_Twice report backend retire one completion more than one time. Derived function
// deliver it, never hide it. Caller own completion. Caller must learn lifecycle broke.
var Retired_Twice = errors.New("time: the completion retired more than once")

// Deadline_Exceeded come back when finite operation retire without its external event.
var Deadline_Exceeded = errors.New("time: deadline exceeded")

// Virtual_Event_Capacity_Exceeded reports no free entry in caller-owned event storage.
var Virtual_Event_Capacity_Exceeded = errors.New("time: virtual event capacity exceeded")

// Completion: caller-owned storage for one in-flight operation. Caller allocate it, thus loop
// never allocate. Caller keep it alive until callback fire.
type Completion struct {
	// Data is an opaque int whose submitting operation defines. A transfer stores its byte
	// count, while an open or accept stores its descriptor.
	Data int
	// Error is nil on success and otherwise stores the operation failure.
	Error error
	// Callback stays specialized so retirement needs no captured adapter closure.
	Callback Callback
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
	// Event lets simulated retirement clear listener state without a captured callback.
	Event Event
	// Backend lets an outer backend correlate specialized result state without a captured
	// adapter. Backend clears it before delivering callback.
	Backend unsafe.Pointer
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
	// State stays caller-owned because every backend operation shares one loop.
	State unsafe.Pointer
	// Submit arm completion to retire one delay from now, in Ready_At order. Every other
	// backend surface schedule through it: read, socket accept, spawn. One queue thus hold
	// full order.
	Submit func(
		state unsafe.Pointer, completion *Completion, delay Duration, callback Callback,
	)
	// Timeout fire callback after duration on clock, off same queue every other completion
	// use. Duration must be positive.
	Timeout func(
		state unsafe.Pointer, completion *Completion, duration Duration, callback Callback,
	)
	// Open_Event make platform Event primitive.
	Open_Event func(state unsafe.Pointer) (event Event, err error)
	// Event_Listen arm completion for one Event notification.
	Event_Listen func(
		state unsafe.Pointer, event Event, completion *Completion, callback Callback,
	)
	// Event_Trigger make armed Event completion ready. Only operation safe to call from other
	// thread.
	Event_Trigger func(state unsafe.Pointer, event Event, completion *Completion)
	// Close_Event release Event after listener drain.
	Close_Event func(state unsafe.Pointer, event Event)
}

// Timeline_Submit passes loop state explicitly because a bound submitter would allocate.
func Timeline_Submit(
	loop Timeline, completion *Completion, delay Duration, callback Callback,
) {
	loop.Submit(loop.State, completion, delay, callback)
}

// Timeline_Timeout passes loop state explicitly because a bound timer would allocate.
func Timeline_Timeout(
	loop Timeline, completion *Completion, duration Duration, callback Callback,
) {
	loop.Timeout(loop.State, completion, duration, callback)
}

// Timeline_Open_Event keeps event ownership with the backend state that opened it.
func Timeline_Open_Event(loop Timeline) (event Event, err error) {
	return loop.Open_Event(loop.State)
}

// Timeline_Event_Listen keeps listener state on its owning backend.
func Timeline_Event_Listen(
	loop Timeline, event Event, completion *Completion, callback Callback,
) {
	loop.Event_Listen(loop.State, event, completion, callback)
}

// Timeline_Event_Trigger keeps trigger state on its owning backend.
func Timeline_Event_Trigger(loop Timeline, event Event, completion *Completion) {
	loop.Event_Trigger(loop.State, event, completion)
}

// Timeline_Close_Event keeps release on the backend that opened the event.
func Timeline_Close_Event(loop Timeline, event Event) {
	loop.Close_Event(loop.State, event)
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
	// State stays caller-owned because binding it into each drive operation would allocate.
	State unsafe.Pointer
	// Run drain every ready completion without block, then advance clock one tick. ROOT ONLY:
	// never hand to library, never call from library.
	Run func(state unsafe.Pointer) (err error)
	// Run_For drive loop until duration elapse on clock. Deliver each completion as it come
	// due. Time is GOAL here. Advance exactly duration, drain as it go, whatever complete.
	// Use to let span of time pass, not to wait for one operation.
	// ROOT ONLY: never hand to library, never call from library.
	Run_For func(state unsafe.Pointer, duration Duration) (err error)
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
		state unsafe.Pointer, timeout Duration, done func() (finished bool),
	) (completed bool, err error)
	// Deinit release kernel resources of backend, after every submitted operation join.
	Deinit func(state unsafe.Pointer)
}

// Driver_Run passes loop state explicitly because a bound driver would allocate.
func Driver_Run(driver Driver) (err error) {
	return driver.Run(driver.State)
}

// Driver_Run_For passes loop state explicitly because a bound driver would allocate.
func Driver_Run_For(driver Driver, duration Duration) (err error) {
	return driver.Run_For(driver.State, duration)
}

// Driver_Run_Until passes loop state explicitly because a bound driver would allocate.
func Driver_Run_Until(
	driver Driver, timeout Duration, done func() (finished bool),
) (completed bool, err error) {
	return driver.Run_Until(driver.State, timeout, done)
}

// Driver_Deinit releases resources through their owning backend state.
func Driver_Deinit(driver Driver) {
	driver.Deinit(driver.State)
}

// Virtual_Timeline_Memory gives the simulator bounded storage without owning an allocation.
type Virtual_Timeline_Memory struct {
	// Queue is caller-owned capacity for simultaneously queued completions.
	Queue []*Completion
	// Events is caller-owned capacity for simultaneously open events.
	Events []Virtual_Event
}

// Virtual_Timeline: deterministic loop backend. One ready-time queue, one virtual clock, no
// kernel. New_Virtual_Timeline build it, never hand it out. Run thus reproduce from clock
// alone. Nothing can script order.
type Virtual_Timeline struct {
	// Virtual: clock configuration. Grain each tick advance, wall-clock origin, modeled skew.
	// Timeline own clock state direct, hold no injected Clock. That indirection is for
	// code outside this package. Timeline is source injected readers build over.
	Virtual Virtual_Clock
	// Queue hold armed completions in Ready_At order, earliest first.
	Queue []*Completion
	// Queue_Count separates occupied entries from caller-owned capacity.
	Queue_Count int
	// Events hold cross-thread event entries indexed by handle minus one.
	Events []Virtual_Event
	// Drive_Active true while Run drive. Run called from inside completion callback thus
	// panic, never re-enter driver.
	Drive_Active bool
}

// Virtual_Event: one simulated cross-thread event.
type Virtual_Event struct {
	// Open prevents a closed slot from accepting listener or trigger operations.
	Open bool
	// Armed report listener attached and wait for next trigger.
	Armed bool
	// Triggered coalesces pending wakeups because one listener retirement is one notification.
	Triggered bool
	// Ready prevents repeated trigger from enqueueing one completion more than once.
	Ready bool
	// Listener keeps ownership direct so no listener map or captured callback is needed.
	Listener *Completion
}

// New_Virtual_Timeline binds caller-owned state and capacity so construction cannot allocate.
func New_Virtual_Timeline(
	state *Virtual_Timeline, virtual Virtual_Clock, memory Virtual_Timeline_Memory,
) (loop Timeline, driver Driver, clock Clock) {
	invariant.Always(state != nil, "A virtual timeline has caller-owned state.")
	invariant.Always(len(memory.Queue) > 0, "A virtual timeline has queue capacity.")
	invariant.Always(len(memory.Events) > 0, "A virtual timeline has event capacity.")
	invariant.Always(len(memory.Queue) <= slices.SLICE_COUNT_MAXIMUM,
		"A virtual timeline queue stays within the repository slice boundary.")
	invariant.Always(len(memory.Events) <= slices.SLICE_COUNT_MAXIMUM,
		"Virtual timeline events stay within the repository slice boundary.")
	Virtual_Clock_Invariants(virtual, "new_virtual_timeline.virtual")
	for index := range memory.Queue {
		memory.Queue[index] = nil
	}
	for index := range memory.Events {
		memory.Events[index] = Virtual_Event{}
	}
	state.Virtual = virtual
	state.Virtual.Ticks = 0
	state.Queue = memory.Queue
	state.Queue_Count = 0
	state.Events = memory.Events
	state.Drive_Active = false
	loop = virtual_timeline_to_timeline(state)
	Timeline_Invariants(loop, "new_virtual_timeline.loop")
	return loop, virtual_timeline_to_driver(state), virtual_timeline_to_clock(state)
}

// Build read-only Clock over own counter of timeline. Holder thus read exactly what driver
// advance. One counter, not second one that drift beside it.
func virtual_timeline_to_clock(state *Virtual_Timeline) (clock Clock) {
	clock = Clock{
		State:         unsafe.Pointer(state),
		Now_Monotonic: virtual_timeline_now_monotonic,
		Now_Realtime:  virtual_timeline_now_realtime,
	}
	Clock_Invariants(clock, "virtual_timeline_to_clock.clock")
	return clock
}

func virtual_timeline_now_monotonic(state unsafe.Pointer) (moment Monotonic_Moment) {
	return virtual_now((*Virtual_Timeline)(state))
}

func virtual_timeline_now_realtime(state unsafe.Pointer) (moment Moment) {
	timeline := (*Virtual_Timeline)(state)
	now := timeline.Virtual.Epoch + Moment(virtual_now(timeline))
	return now - Moment(Offset_Read(timeline.Virtual.Skew, timeline.Virtual.Ticks))
}

// Wire control plane onto vtable every backend and every caller hold.
func virtual_timeline_to_timeline(state *Virtual_Timeline) (loop Timeline) {
	return Timeline{
		State:         unsafe.Pointer(state),
		Submit:        virtual_timeline_submit,
		Timeout:       virtual_timeline_timeout,
		Open_Event:    virtual_timeline_open_event,
		Event_Listen:  virtual_timeline_event_listen,
		Event_Trigger: virtual_timeline_event_trigger,
		Close_Event:   virtual_timeline_close_event,
	}
}

func virtual_timeline_submit(
	state unsafe.Pointer, completion *Completion, delay Duration, callback Callback,
) {
	virtual_submit((*Virtual_Timeline)(state), completion, delay, callback)
}

func virtual_timeline_timeout(
	state unsafe.Pointer, completion *Completion, duration Duration, callback Callback,
) {
	virtual_timeout((*Virtual_Timeline)(state), completion, duration, callback)
}

func virtual_timeline_open_event(state unsafe.Pointer) (event Event, err error) {
	timeline := (*Virtual_Timeline)(state)
	for index := range timeline.Events {
		if !timeline.Events[index].Open {
			timeline.Events[index] = Virtual_Event{Open: true}
			return Event(index + 1), nil
		}
	}
	return 0, Virtual_Event_Capacity_Exceeded
}

func virtual_timeline_event_listen(
	state unsafe.Pointer, event Event, completion *Completion, callback Callback,
) {
	virtual_event_listen((*Virtual_Timeline)(state), event, completion, callback)
}

func virtual_timeline_event_trigger(
	state unsafe.Pointer, event Event, completion *Completion,
) {
	virtual_event_trigger((*Virtual_Timeline)(state), event, completion)
}

func virtual_timeline_close_event(state unsafe.Pointer, event Event) {
	timeline := (*Virtual_Timeline)(state)
	entry := virtual_event_entry(timeline, event)
	invariant.Always(!entry.Armed, "An event listener is drained before close.")
	*entry = Virtual_Event{}
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
	entry := virtual_event_entry(state, event)
	invariant.Always(!entry.Armed, "An event has at most one armed listener.")
	if entry.Triggered {
		virtual_queue_has_capacity(state)
	}
	virtual_arm(completion, callback)
	completion.Event = event
	entry.Armed = true
	entry.Listener = completion
	if entry.Triggered {
		entry.Triggered = false
		entry.Ready = true
		virtual_enqueue_now(state, completion)
	}
}

// Make armed event listener ready, or record trigger for later listener.
func virtual_event_trigger(state *Virtual_Timeline, event Event, completion *Completion) {
	entry := virtual_event_entry(state, event)
	if !entry.Armed {
		entry.Triggered = true
		return
	}
	invariant.Always(entry.Listener == completion,
		"A trigger names the completion its event armed.")
	if entry.Ready {
		entry.Triggered = true
		return
	}
	virtual_queue_has_capacity(state)
	entry.Ready = true
	virtual_enqueue_now(state, completion)
}

func virtual_event_entry(state *Virtual_Timeline, event Event) (entry *Virtual_Event) {
	invariant.Always(event > 0, "A virtual event handle is never zero.")
	invariant.Always(event <= Event(len(state.Events)),
		"A virtual event handle names caller-owned storage.")
	entry = &state.Events[int(event)-1]
	invariant.Always(entry.Open, "A virtual event operation names an open event.")
	return entry
}

// Read simulated moment every Ready_At measure against.
func virtual_now(state *Virtual_Timeline) (now Monotonic_Moment) {
	now = Monotonic_Moment(int64(state.Virtual.Ticks) * int64(state.Virtual.Resolution))
	Monotonic_Moment_Invariants(now, "virtual_now.now")
	return now
}

// Schedule completion to fire at now plus delay. Insert it in Ready_At order.
func virtual_submit(
	state *Virtual_Timeline, completion *Completion, delay Duration, callback Callback,
) {
	virtual_queue_has_capacity(state)
	virtual_arm(completion, callback)
	completion.Ready_At = virtual_now(state) + Monotonic_Moment(delay)
	virtual_enqueue(state, completion)
}

// Arm completion, but never put it on ready-time queue. Event listener use this. Assert
// completion is own original, not by-value copy. Then move it along lifecycle machine.
// Completion armed while armed panic on armed-to-armed edge.
func virtual_arm(completion *Completion, callback Callback) {
	original := completion.Self == nil || completion.Self == completion
	invariant.Always(original,
		"A submitted completion is its own original, never a by-value copy.")
	completion.Self = completion
	invariant.Always(!completion.Armed, "An armed completion is never armed a second time.")
	completion.Data = 0
	completion.Error = nil
	completion.Armed = true
	completion.Callback = callback
	completion.Event = 0
}

// Put already armed completion on queue as due now.
func virtual_enqueue_now(state *Virtual_Timeline, completion *Completion) {
	completion.Ready_At = virtual_now(state)
	virtual_enqueue(state, completion)
}

// Insert completion into queue in Ready_At order, earliest first.
func virtual_enqueue(state *Virtual_Timeline, completion *Completion) {
	index := 0
	for index < state.Queue_Count && state.Queue[index].Ready_At <= completion.Ready_At {
		index++
	}
	copy(state.Queue[index+1:state.Queue_Count+1], state.Queue[index:state.Queue_Count])
	state.Queue[index] = completion
	state.Queue_Count++
}

// Capacity rejects new ownership before any caller or event state changes.
func virtual_queue_has_capacity(state *Virtual_Timeline) {
	invariant.Always(state.Queue_Count < len(state.Queue),
		"A virtual timeline never exceed caller-owned queue capacity.")
}

// Fire earliest completion when due as of now. Report whether it fire.
func virtual_step(state *Virtual_Timeline) (advanced bool) {
	if state.Queue_Count == 0 {
		return false
	}
	if state.Queue[0].Ready_At > virtual_now(state) {
		return false
	}
	completion := state.Queue[0]
	last := state.Queue_Count - 1
	copy(state.Queue[:last], state.Queue[1:state.Queue_Count])
	state.Queue[last] = nil
	state.Queue_Count--
	// Go back to idle before callback run. Callback can then submit own completion again.
	// Repeating-timer pattern.
	invariant.Always(completion.Armed, "A delivered completion was armed.")
	completion.Armed = false
	callback := completion.Callback
	completion.Callback = nil
	event := completion.Event
	completion.Event = 0
	if event != 0 {
		entry := virtual_event_entry(state, event)
		entry.Armed = false
		entry.Ready = false
		entry.Listener = nil
	}
	callback(completion)
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
	state.Virtual.Ticks++
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

// Drive begin rejects reentrancy before any queue state can change.
func virtual_drive_begin(state *Virtual_Timeline) {
	invariant.Always(!state.Drive_Active,
		"A drive begins at top level, never from within a completion callback.")
	state.Drive_Active = true
}

// Drive end remains deferred so callback panic cannot leave the driver permanently active.
func virtual_drive_end(state *Virtual_Timeline) {
	state.Drive_Active = false
}

// Build driver over state. Capability that advance time. Only main or test hold it.
func virtual_timeline_to_driver(state *Virtual_Timeline) (driver Driver) {
	return Driver{
		State:     unsafe.Pointer(state),
		Run:       virtual_driver_run,
		Run_For:   virtual_driver_run_for,
		Run_Until: virtual_driver_run_until,
		Deinit:    virtual_driver_deinit,
	}
}

func virtual_driver_run(state unsafe.Pointer) (err error) {
	timeline := (*Virtual_Timeline)(state)
	virtual_drive_begin(timeline)
	defer virtual_drive_end(timeline)
	virtual_run(timeline)
	return nil
}

func virtual_driver_run_for(state unsafe.Pointer, duration Duration) (err error) {
	timeline := (*Virtual_Timeline)(state)
	virtual_drive_begin(timeline)
	defer virtual_drive_end(timeline)
	virtual_run_for(timeline, duration)
	return nil
}

func virtual_driver_run_until(
	state unsafe.Pointer, timeout Duration, done func() (finished bool),
) (completed bool, err error) {
	timeline := (*Virtual_Timeline)(state)
	virtual_drive_begin(timeline)
	defer virtual_drive_end(timeline)
	return virtual_run_until(timeline, timeout, done), nil
}

func virtual_driver_deinit(state unsafe.Pointer) {
	invariant.Always(state != nil, "A virtual driver deinitializes caller-owned state.")
}
