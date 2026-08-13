//go:build darwin

package time

/*
#include <mach/mach_time.h>
*/
import "C"

import "local/james-orcales/shared/simulation/time"

// Returns the reader for mach_continuous_time — the monotonic clock that, unlike
// mach_absolute_time, keeps counting across system suspend. The timebase converts mach
// ticks to nanoseconds; mach_timebase_info is cached inside libc, so querying it per
// call is cheap and avoids a forbidden package-level cache.
//
// The reading leaves through a closure, never through a function result, for the reason
// csprng gives its raw entropy draw: a host counter holds whatever it holds at the
// instant it is read, so it carries no domain a bundle could state and no bound a test
// could put a value on. Only the caller-set values of the pure tier carry one.
func new_monotonic_reader() (read func() (moment time.Moment)) {
	return func() (moment time.Moment) {
		var timebase C.mach_timebase_info_data_t
		if C.mach_timebase_info(&timebase) != 0 {
			panic("time: mach_timebase_info failed")
		}
		if timebase.denom == 0 {
			panic("time: mach_timebase_info returned a zero denominator")
		}
		ticks := uint64(C.mach_continuous_time())
		return time.Moment(ticks * uint64(timebase.numer) / uint64(timebase.denom))
	}
}
