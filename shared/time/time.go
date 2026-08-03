// Package time is a dependency-injected time source modeled on TigerBeetle's
// vsr.Time, whose backend is a struct of function pointers (a vtable). Here that is
// a struct of closures: production wires an OS clock (time/default), a simulation
// wires a Virtual one, and the code between never knows which it holds. The Virtual
// backend lives here because it is pure arithmetic with no operating-system call.
package time

import (
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

// Clock is the injected time source — the Go translation of TigerBeetle's `Time`
// vtable, expressed as closures so the backend is chosen by value. It is read-only:
// advancing time is the driver's job (the tick returned beside the clock at
// construction), so a holder can only read the current Moment, never move time.
type Clock struct {
	// Now_Monotonic reads the monotonic clock, which never regresses; use it to
	// measure elapsed time, timeouts, and latency.
	Now_Monotonic func() (moment Moment)
	// Now_Realtime reads wall-clock time as nanoseconds since the Unix epoch; it can
	// jump, so use it only for calendar timestamps, never for elapsed time.
	Now_Realtime func() (moment Moment)
}

// Clock_Invariants states that both readers are bound. A Clock is a vtable, so its
// only property is that every slot is filled: the zero Clock reads as a Clock but
// panics on first use, and a backend that fills one slot and forgets the other is the
// same failure one call later.
func Clock_Invariants(clock Clock, namespace invariant.Namespace) {
	invariant.Always(
		clock.Now_Monotonic != nil, "A Clock has a monotonic reader.",
	)
	invariant.Always(
		clock.Now_Realtime != nil, "A Clock has a realtime reader.",
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

// Virtual_Clock configures the deterministic clock Virtual_Clock_To_Clock builds —
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

// Virtual_Clock_To_Clock returns a read-only Clock backed by a deterministic, OS-free
// virtual clock, plus the tick that advances it. The clock's closures and tick share
// one counter, so tick advances what the next Now_Monotonic reads. Only the driver —
// package main or a test harness — holds tick; pure code holds only the Clock and so
// can read time but never move it.
func Virtual_Clock_To_Clock(virtual Virtual_Clock) (clock Clock, tick func()) {
	defer func() { Clock_Invariants(clock, "virtual_clock_to_clock.clock") }()
	Virtual_Clock_Invariants(virtual, "virtual_clock_to_clock.virtual")
	ticks := Tick_Count(0)
	clock = Clock{
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
