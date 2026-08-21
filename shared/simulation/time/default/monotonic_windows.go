//go:build windows

package time

import "local/james-orcales/shared/simulation/time"

// Panic: system clock not yet supported on Windows. Faithful backend is
// QueryPerformanceCounter, which count across suspend. This stub keep package building on
// Windows until that land.
func new_monotonic_reader() (read func() (moment time.Monotonic_Moment)) {
	return func() (moment time.Monotonic_Moment) {
		panic("time: the system clock is not yet supported on Windows")
	}
}
