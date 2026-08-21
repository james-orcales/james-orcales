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
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/sim/time"
)

func monotonic_now() (moment time.Monotonic_Moment) {
	defer func() { time.Monotonic_Moment_Invariants(moment, "monotonic_now.moment") }()
	timebase := C.mach_timebase_info_data_t{}
	aver.Always(C.mach_timebase_info(&timebase) == 0, "The host reports its timebase.")
	aver.Always(timebase.denom != 0, "A timebase has a nonzero denominator.")
	ticks := uint64(C.mach_continuous_time())
	denominator := uint64(timebase.denom)
	numerator := uint64(timebase.numer)
	whole := ticks / denominator * numerator
	remainder := ticks % denominator * numerator / denominator
	return time.Monotonic_Moment(whole + remainder)
}
