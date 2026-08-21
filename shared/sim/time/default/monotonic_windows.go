//go:build windows

package time

import (
	"syscall"

	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/sim/time"
)

// Stub: system clock not yet supported on Windows. Faithful backend is
// QueryPerformanceCounter, which count across suspend. This stub keep package building on
// Windows until that land, and fails the first read the way the backend would on a refused
// call.
func monotonic_now() (moment time.Monotonic_Moment) {
	defer func() { time.Monotonic_Moment_Invariants(moment, "monotonic_now.moment") }()
	errno := syscall.ENOSYS
	aver.Always(errno == 0, "The host reports a monotonic clock.")
	return 0
}
