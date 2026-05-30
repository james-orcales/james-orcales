//go:build linux

package time

import (
	"syscall"
	"unsafe"
)

// CLOCK_BOOTTIME differs from CLOCK_MONOTONIC by counting time spent in system
// suspend (a VM migration or a laptop sleep), which is what a monotonic clock
// measuring real elapsed time must do.
const clock_boottime = 7

// Converts the timespec seconds field to nanoseconds.
const nanoseconds_per_second = 1_000_000_000

// Reads CLOCK_BOOTTIME as nanoseconds. The Go syscall package ships no
// ClockGettime wrapper, so the raw clock_gettime syscall is issued directly —
// x/sys/unix would add a dependency this module does not carry.
func monotonic_nanoseconds() (nanoseconds int64) {
	var timestamp syscall.Timespec
	pointer := uintptr(unsafe.Pointer(&timestamp))
	_, _, errno := syscall.Syscall(syscall.SYS_CLOCK_GETTIME, clock_boottime, pointer, 0)
	if errno != 0 {
		panic("time: CLOCK_BOOTTIME is required but clock_gettime failed")
	}
	return int64(timestamp.Sec)*nanoseconds_per_second + int64(timestamp.Nsec)
}
