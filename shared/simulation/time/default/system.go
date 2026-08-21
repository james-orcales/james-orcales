// Package time is composition tier: operating-system-backed clock. Declare package time, thus
// caller import ".../time/default" and read it as library with no alias.
package time

import (
	"sync/atomic"

	"local/james-orcales/shared/simulation/time"
)

// NANOSECONDS_PER_SECOND converts the host timeval seconds field to repository duration units.
const NANOSECONDS_PER_SECOND = 1_000_000_000

// MICROSECONDS_PER_SECOND makes the timeval subsecond conversion visible as a formula.
const MICROSECONDS_PER_SECOND = 1_000_000

// NANOSECONDS_PER_MICROSECOND converts the host timeval microseconds field without a literal.
const NANOSECONDS_PER_MICROSECOND = NANOSECONDS_PER_SECOND / MICROSECONDS_PER_SECOND

// New_Operating_System_Clock return read-only Clock backed by host operating system.
// Now_Monotonic read OS monotonic clock behind guard that panic when clock go backward.
// Now_Realtime read wall clock. Host clock advance on its own, thus constructor hand back no
// tick: only virtual clock need one. Every host reading stay inside closure, cross no function
// of this package: value machine supply carry no domain bundle could state.
func New_Operating_System_Clock() (host time.Clock) {
	defer func() { time.Clock_Invariants(host, "new_operating_system_clock.host") }()
	read := new_monotonic_reader()
	// Guard hold highest monotonic read handed out. Real backward step thus caught, even when
	// off-loop goroutines of IO backend read clock beside loop.
	guard := &atomic.Int64{}
	host = time.Clock{
		// Panic when monotonic clock truly go backward. Stay correct when many goroutines
		// read at once (IO backend time spawn off loop thread). Read below guard is either
		// reorder, or real backward step. Reorder mean concurrent reader record later time.
		// Re-read tell two apart. Re-read happen after this goroutine see previous, thus it
		// is causally after read that set previous. Healthy clock thus return at least
		// previous. Only clock truly behind stay below it. False order from racing store
		// fix itself. Clock that go backward do not.
		Now_Monotonic: func() (moment time.Monotonic_Moment) {
			raw := read()
			previous := time.Monotonic_Moment(guard.Load())
			for raw >= previous && !guard.CompareAndSwap(int64(previous), int64(raw)) {
				previous = time.Monotonic_Moment(guard.Load())
			}
			if raw < previous {
				if read() < previous {
					panic("time: the monotonic clock regressed (a kernel bug)")
				}
				return previous
			}
			return raw
		},
		Now_Realtime: func() (moment time.Moment) {
			return time.Moment(wallclock_now_nanoseconds())
		},
	}
	return host
}
