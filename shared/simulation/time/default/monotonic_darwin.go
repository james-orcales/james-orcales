//go:build darwin

package time

/*
#cgo noescape mach_timebase_info
#cgo nocallback mach_timebase_info
#cgo noescape mach_continuous_time
#cgo nocallback mach_continuous_time
#include <mach/mach_time.h>
*/
import "C"

import (
	"syscall"

	"local/james-orcales/shared/simulation/time"
)

func monotonic_now() (moment time.Monotonic_Moment) {
	timebase := C.mach_timebase_info_data_t{}
	if C.mach_timebase_info(&timebase) != 0 {
		panic("time: mach_timebase_info failed")
	}
	if timebase.denom == 0 {
		panic("time: mach_timebase_info returned a zero denominator")
	}
	ticks := uint64(C.mach_continuous_time())
	denominator := uint64(timebase.denom)
	numerator := uint64(timebase.numer)
	whole := ticks / denominator * numerator
	remainder := ticks % denominator * numerator / denominator
	return time.Monotonic_Moment(whole + remainder)
}

func wallclock_now_nanoseconds() (nanoseconds int64) {
	value := syscall.Timeval{}
	if err := syscall.Gettimeofday(&value); err != nil {
		panic("time: gettimeofday failed")
	}
	return int64(value.Sec)*NANOSECONDS_PER_SECOND +
		int64(value.Usec)*NANOSECONDS_PER_MICROSECOND
}
