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

// Return reader for CLOCK_BOOTTIME. Go syscall package ship no ClockGettime wrapper, thus
// raw clock_gettime syscall go direct. x/sys/unix would add dependency this module do not
// carry.
//
// Reading leave through closure, never through function result. Same reason csprng give raw
// entropy draw that way: host counter hold whatever it hold at instant of read, thus carry no
// domain bundle could state, and no bound test could put value on. Only caller-set values of
// pure tier carry one.
func new_monotonic_reader() (read func() (moment time.Monotonic_Moment)) {
	return func() (moment time.Monotonic_Moment) {
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
}

func wallclock_now_nanoseconds() (nanoseconds int64) {
	value := syscall.Timeval{}
	if err := syscall.Gettimeofday(&value); err != nil {
		panic("time: gettimeofday failed")
	}
	return int64(value.Sec)*NANOSECONDS_PER_SECOND +
		int64(value.Usec)*NANOSECONDS_PER_MICROSECOND
}
