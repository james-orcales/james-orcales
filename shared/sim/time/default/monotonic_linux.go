//go:build linux

package time

import (
	"syscall"
	"unsafe"

	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/sim/time"
)

// CLOCK_BOOTTIME differ from CLOCK_MONOTONIC: it count time spent in system suspend (VM
// migration, or laptop sleep). Monotonic clock that measure real elapsed time must do that.
const CLOCK_BOOTTIME = 7

// Go syscall package ships no ClockGettime wrapper, and a direct syscall avoids another
// dependency for one primitive.
func monotonic_now() (moment time.Monotonic_Moment) {
	defer func() { time.Monotonic_Moment_Invariants(moment, "monotonic_now.moment") }()
	var timestamp syscall.Timespec
	pointer := uintptr(unsafe.Pointer(&timestamp))
	_, _, errno := syscall.Syscall(
		syscall.SYS_CLOCK_GETTIME, CLOCK_BOOTTIME, pointer, 0,
	)
	aver.Always(errno == 0, "The host reports CLOCK_BOOTTIME.")
	seconds := int64(timestamp.Sec) * NANOSECONDS_PER_SECOND
	return time.Monotonic_Moment(seconds + int64(timestamp.Nsec))
}
