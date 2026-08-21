//go:build darwin

package time

/*
#include <mach/mach_time.h>
*/
import "C"

import (
	"syscall"

	"local/james-orcales/shared/simulation/time"
)

// Return reader for mach_continuous_time. That is monotonic clock that keep counting across
// system suspend. mach_absolute_time do not. Timebase turn mach ticks into nanoseconds. libc
// cache mach_timebase_info, thus per-call query is cheap and need no package-level cache,
// which is banned.
//
// Reading leave through closure, never through function result. Same reason csprng give raw
// entropy draw that way: host counter hold whatever it hold at instant of read, thus carry no
// domain bundle could state, and no bound test could put value on. Only caller-set values of
// pure tier carry one.
func new_monotonic_reader() (read func() (moment time.Monotonic_Moment)) {
	return func() (moment time.Monotonic_Moment) {
		var timebase C.mach_timebase_info_data_t
		if C.mach_timebase_info(&timebase) != 0 {
			panic("time: mach_timebase_info failed")
		}
		if timebase.denom == 0 {
			panic("time: mach_timebase_info returned a zero denominator")
		}
		ticks := uint64(C.mach_continuous_time())
		return time.Monotonic_Moment(
			ticks * uint64(timebase.numer) / uint64(timebase.denom))
	}
}

func wallclock_now_nanoseconds() (nanoseconds int64) {
	value := syscall.Timeval{}
	if err := syscall.Gettimeofday(&value); err != nil {
		panic("time: gettimeofday failed")
	}
	return int64(value.Sec)*NANOSECONDS_PER_SECOND +
		int64(value.Usec)*NANOSECONDS_PER_MICROSECOND
}
