// Package time is the composition tier: the operating-system-backed clock, the Go
// translation of TigerBeetle's TimeOS. It declares package time so callers import
// ".../time/default" and read it as the library with no alias.
package time

import (
	"sync/atomic"
	wallclock "time"

	"github.com/james-orcales/james-orcales/shared/time"
)

// New_Operating_System_Clock returns a read-only Clock backed by the host operating
// system — TigerBeetle's TimeOS — plus the driver's tick. Now_Monotonic reads the OS
// monotonic clock behind a guard that panics on regression; Now_Realtime reads the
// wall clock. The OS clock advances on its own, so tick is a no-op.
func New_Operating_System_Clock() (host time.Clock, tick func()) {
	// The guard remembers the last monotonic read so a regression — which a hardware
	// or kernel bug can cause — is caught instead of wedging callers.
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

// Reads the per-OS monotonic clock and panics if it ran backwards.
func read_monotonic(guard *atomic.Int64) (now time.Moment) {
	raw := monotonic_nanoseconds()
	previous := guard.Swap(raw)
	if raw < previous {
		panic("time: the monotonic clock regressed (a hardware or kernel bug)")
	}
	return time.Moment(raw)
}

// Reads the wall clock as nanoseconds since the Unix epoch.
func read_realtime() (now time.Moment) {
	return time.Moment(wallclock.Now().UnixNano())
}
