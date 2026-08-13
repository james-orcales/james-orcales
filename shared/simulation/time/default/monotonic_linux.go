//go:build linux

package time

import (
	"syscall"
	"unsafe"

	"local/james-orcales/shared/simulation/time"
)

// CLOCK_BOOTTIME differs from CLOCK_MONOTONIC by counting time spent in system
// suspend (a VM migration or a laptop sleep), which is what a monotonic clock
// measuring real elapsed time must do.
const CLOCK_BOOTTIME = 7

// Converts the timespec seconds field to nanoseconds.
const NANOSECONDS_PER_SECOND = 1_000_000_000

// Returns the reader for CLOCK_BOOTTIME. The Go syscall package ships no ClockGettime
// wrapper, so the raw clock_gettime syscall is issued directly — x/sys/unix would add a
// dependency this module does not carry.
//
// The reading leaves through a closure, never through a function result, for the reason
// csprng gives its raw entropy draw: a host counter holds whatever it holds at the
// instant it is read, so it carries no domain a bundle could state and no bound a test
// could put a value on. Only the caller-set values of the pure tier carry one.
func new_monotonic_reader() (read func() (moment time.Moment)) {
	return func() (moment time.Moment) {
		var timestamp syscall.Timespec
		pointer := uintptr(unsafe.Pointer(&timestamp))
		_, _, errno := syscall.Syscall(
			syscall.SYS_CLOCK_GETTIME, CLOCK_BOOTTIME, pointer, 0,
		)
		if errno != 0 {
			panic("time: CLOCK_BOOTTIME is required but clock_gettime failed")
		}
		seconds := int64(timestamp.Sec) * NANOSECONDS_PER_SECOND
		return time.Moment(seconds + int64(timestamp.Nsec))
	}
}
