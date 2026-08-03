// Package time is the composition tier: the operating-system-backed clock, the Go
// translation of TigerBeetle's TimeOS. It declares package time so callers import
// ".../time/default" and read it as the library with no alias.
package time

import (
	"sync/atomic"
	wallclock "time"

	"local/james-orcales/shared/time"
)

// New_Operating_System_Clock returns a read-only Clock backed by the host operating
// system — TigerBeetle's TimeOS — plus the driver's tick. Now_Monotonic reads the OS
// monotonic clock behind a guard that panics on regression; Now_Realtime reads the wall
// clock. The OS clock advances on its own, so tick is a no-op. Every host reading stays
// inside a closure and crosses no function of this package, because a value the machine
// supplies carries no domain a bundle could state.
func New_Operating_System_Clock() (host time.Clock, tick func()) {
	defer func() { time.Clock_Invariants(host, "new_operating_system_clock.host") }()
	read := new_monotonic_reader()
	// The guard holds the highest monotonic read handed out, so a real regression is caught
	// even when the IO backend's off-loop goroutines read the clock alongside the loop.
	guard := &atomic.Int64{}
	host = time.Clock{
		// Panics if the monotonic clock genuinely ran backwards, staying correct when
		// several goroutines read at once (the IO backend times a spawn off the loop
		// thread). A read below the guard is either a reorder (a concurrent reader recorded
		// a later time) or a real regression; the re-read tells them apart. Because the
		// re-read happens after this goroutine observed previous, it is causally after the
		// read that set previous, so on a healthy clock it returns at least previous — only
		// a clock that is truly behind stays below it. A false ordering from a racing store
		// resolves itself; a regressing clock does not.
		Now_Monotonic: func() (moment time.Moment) {
			raw := read()
			previous := time.Moment(guard.Load())
			for raw >= previous && !guard.CompareAndSwap(int64(previous), int64(raw)) {
				previous = time.Moment(guard.Load())
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
			return time.Moment(wallclock.Now().UnixNano())
		},
	}
	return host, func() {}
}

// New_Sleep returns the host sleeper: a closure that parks the calling goroutine for a
// span of the wall clock. It is the real counterpart of a virtual clock's tick, held by
// the composition root and injected into code that must wait (a diode drain, say) so
// the pure tier still touches no wall clock.
//
// !!! THIS IS A STOPGAP. THE WAIT BELONGS IN io.IO.Timeout. !!!
//
// A goroutine parked in wallclock.Sleep is a thread the completion loop cannot drive,
// cannot cancel, and cannot advance against a simulated timeline, so every seed that
// reaches this call stops reproducing. That is also why the span carries no assertion:
// its domain runs to the largest int64, and a test that put a value on that bound would
// have to wait out the sleep it names. io.IO.Timeout submits the wait as an operation
// the loop owns, where the simulator supplies the bound instead of the wall clock.
// Delete this constructor the moment the drain takes an io.IO.
func New_Sleep() (sleep func(duration time.Duration)) {
	return func(duration time.Duration) {
		wallclock.Sleep(wallclock.Duration(int64(duration)))
	}
}
