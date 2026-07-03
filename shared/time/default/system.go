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
// monotonic clock behind a guard that panics on regression; Now_Realtime reads the
// wall clock. The OS clock advances on its own, so tick is a no-op.
func New_Operating_System_Clock() (host time.Clock, tick func()) {
	// The guard holds the highest monotonic read handed out, so a real regression is caught
	// even when the IO backend's off-loop goroutines read the clock alongside the loop.
	guard := &atomic.Int64{}
	host = time.Clock{
		Now_Monotonic: func() (moment time.Moment) { return read_monotonic(guard) },
		Now_Realtime:  func() (moment time.Moment) { return read_realtime() },
	}
	return host, func() {}
}

// Sleep blocks the calling goroutine for duration against the host clock — the real
// counterpart of a virtual clock's tick, held by the composition root and injected into
// code that must wait (a diode drain, say) so the pure tier still touches no wall clock.
func Sleep(duration time.Duration) {
	wallclock.Sleep(wallclock.Duration(int64(duration)))
}

// Reads the per-OS monotonic clock and panics if it genuinely ran backwards, staying
// correct when several goroutines read at once (the IO backend times a spawn off the loop
// thread). guard holds the highest value handed out. A read below it is either a reorder (a
// concurrent reader recorded a later time) or a real regression; the re-read tells them
// apart. Because the re-read happens after this goroutine observed previous, it is causally
// after the read that set previous, so on a healthy clock it returns at least previous —
// only a clock that is truly behind stays below it. A false ordering from a racing store
// resolves itself; a regressing clock does not.
func read_monotonic(guard *atomic.Int64) (now time.Moment) {
	raw := monotonic_nanoseconds()
	previous := guard.Load()
	for raw >= previous && !guard.CompareAndSwap(previous, raw) {
		previous = guard.Load()
	}
	if raw < previous {
		if monotonic_nanoseconds() < previous {
			panic("time: the monotonic clock regressed (a hardware or kernel bug)")
		}
		return time.Moment(previous)
	}
	return time.Moment(raw)
}

// Reads the wall clock as nanoseconds since the Unix epoch.
func read_realtime() (now time.Moment) {
	return time.Moment(wallclock.Now().UnixNano())
}
