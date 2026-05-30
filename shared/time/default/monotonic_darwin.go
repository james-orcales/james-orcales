//go:build darwin

package time

/*
#include <mach/mach_time.h>
*/
import "C"

// Reads mach_continuous_time — the monotonic clock that, unlike mach_absolute_time,
// keeps counting across system suspend. The timebase converts mach ticks to
// nanoseconds; mach_timebase_info is cached inside libc, so querying it per call is
// cheap and avoids a forbidden package-level cache.
func monotonic_nanoseconds() (nanoseconds int64) {
	var timebase C.mach_timebase_info_data_t
	if C.mach_timebase_info(&timebase) != 0 {
		panic("time: mach_timebase_info failed")
	}
	if timebase.denom == 0 {
		panic("time: mach_timebase_info returned a zero denominator")
	}
	ticks := uint64(C.mach_continuous_time())
	return int64(ticks * uint64(timebase.numer) / uint64(timebase.denom))
}
