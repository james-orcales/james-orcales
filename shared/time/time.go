// Package time is a dependency-injected time source modeled on TigerBeetle's
// vsr.Time, whose backend is a struct of function pointers (a vtable). Here that is
// a struct of closures: production wires an OS clock (time/default), a simulation
// wires a Virtual one, and the code between never knows which it holds. The Virtual
// backend lives here because it is pure arithmetic with no operating-system call.
package time

import "local/james-orcales/shared/fixedpoint"

// Moment is a clock reading in nanoseconds since an arbitrary, clock-specific epoch
// (TigerBeetle's stdx.Instant). Only the difference between two Moments from the
// SAME clock is meaningful; a monotonic Moment and a realtime Moment are not
// comparable.
type Moment int64

// Duration is a span of nanoseconds (TigerBeetle's stdx.Duration).
type Duration int64

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

// Offset models how a simulated wall clock deviates from true elapsed time —
// TigerBeetle's TimeSim.offset. It is what makes Now_Realtime diverge from
// Now_Monotonic. A nil Offset is a perfect clock.
type Offset func(ticks int64) (skew Duration)

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

// Virtual_Clock_To_Clock returns a read-only Clock backed by a deterministic, OS-free
// virtual clock, plus the tick that advances it. The clock's closures and tick share
// one counter, so tick advances what the next Now_Monotonic reads. Only the driver —
// package main or a test harness — holds tick; pure code holds only the Clock and so
// can read time but never move it.
func Virtual_Clock_To_Clock(virtual Virtual_Clock) (clock Clock, tick func()) {
	ticks := int64(0)
	clock = Clock{
		Now_Monotonic: func() (moment Moment) {
			return Moment(ticks * int64(virtual.Resolution))
		},
		Now_Realtime: func() (moment Moment) {
			now := virtual.Epoch + Moment(ticks*int64(virtual.Resolution))
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

// Skew_Input is the model and its coefficients, mirroring TimeSim's offset_type plus
// offset_coefficient_A and offset_coefficient_B.
type Skew_Input struct {
	// Kind selects the deviation model.
	Kind Skew_Kind
	// A is the magnitude coefficient: drift-per-tick, amplitude, or step size.
	A Duration
	// B is the tick coefficient: the linear initial offset, the periodic period, or
	// the step's onset tick.
	B int64
}

// Skew builds the Offset described by input.
func Skew(input Skew_Input) (offset Offset) {
	switch input.Kind {
	case SKEW_KIND_PERIODIC:
		return func(ticks int64) (skew Duration) {
			// A zero period is a degenerate sinusoid; report no skew rather than divide
			// (or take a remainder) by zero.
			if input.B == 0 {
				return 0
			}
			// Reduce the phase to one period before lifting it into fixed-point, so a
			// long-running tick count cannot overflow the scaled numerator.
			phase := ticks % input.B
			turns := fixedpoint.Divide(&fixedpoint.Divide_Input{
				Dividend: fixedpoint.From_Integer(phase),
				Divisor:  fixedpoint.From_Integer(input.B),
			})
			wobble := fixedpoint.Multiply(&fixedpoint.Multiply_Input{
				A: fixedpoint.From_Integer(int64(input.A)),
				B: fixedpoint.Sine_Turns(turns),
			})
			return Duration(fixedpoint.Whole(wobble))
		}
	case SKEW_KIND_STEP:
		return func(ticks int64) (skew Duration) {
			if ticks > input.B {
				return input.A
			}
			return 0
		}
	default:
		return func(ticks int64) (skew Duration) {
			return Duration(ticks)*input.A + Duration(input.B)
		}
	}
}
