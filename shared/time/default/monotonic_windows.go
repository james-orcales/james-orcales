//go:build windows

package time

import "local/james-orcales/shared/time"

// Panics: the system clock is not yet supported on Windows. The faithful backend
// is QueryPerformanceCounter (it counts across suspend); this stub keeps the
// package building on Windows until that lands.
func new_monotonic_reader() (read func() (moment time.Moment)) {
	return func() (moment time.Moment) {
		panic("time: the system clock is not yet supported on Windows")
	}
}
