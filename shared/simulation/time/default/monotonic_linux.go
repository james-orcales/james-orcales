//go:build linux

package time

import (
	"syscall"
	"unsafe"

	"local/james-orcales/shared/simulation/time"
)

// CLOCK_BOOTTIME differ from CLOCK_MONOTONIC: it count time spent in system suspend (VM
// migration, or laptop sleep). Monotonic clock that measure real elapsed time must do that.
const CLOCK_BOOTTIME = 7

// Go syscall package ships no ClockGettime wrapper, and a direct syscall avoids another
// dependency for one primitive.
func monotonic_now() (moment time.Monotonic_Moment) {
	var timestamp syscall.Timespec
	pointer := uintptr(unsafe.Pointer(&timestamp))
	_, _, errno := syscall.Syscall(
		syscall.SYS_CLOCK_GETTIME, CLOCK_BOOTTIME, pointer, 0,
	)
	if errno != 0 {
		panic("time: CLOCK_BOOTTIME is required but clock_gettime failed")
	}
	seconds := int64(timestamp.Sec) * NANOSECONDS_PER_SECOND
	return time.Monotonic_Moment(seconds + int64(timestamp.Nsec))
}

func wallclock_now_nanoseconds() (nanoseconds int64) {
	value := syscall.Timeval{}
	if err := syscall.Gettimeofday(&value); err != nil {
		panic("time: gettimeofday failed")
	}
	return int64(value.Sec)*NANOSECONDS_PER_SECOND +
		int64(value.Usec)*NANOSECONDS_PER_MICROSECOND
}
