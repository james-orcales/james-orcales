//go:build windows

package time

import (
	"syscall"

	"local/james-orcales/shared/sim/time"
)

// Panic: system clock not yet supported on Windows. Faithful backend is
// QueryPerformanceCounter, which count across suspend. This stub keep package building on
// Windows until that land.
func monotonic_now() (moment time.Monotonic_Moment) {
	panic("time: the system clock is not yet supported on Windows")
}

func wallclock_now_nanoseconds() (nanoseconds int64) {
	value := syscall.Timeval{}
	if err := syscall.Gettimeofday(&value); err != nil {
		panic("time: gettimeofday failed")
	}
	return value.Nanoseconds()
}
