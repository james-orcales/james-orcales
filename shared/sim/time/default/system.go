// Package time is composition tier: operating-system-backed clock. Declare package time, thus
// caller import ".../time/default" and read it as library with no alias.
package time

import (
	"syscall"

	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/sim/time"
)

// NANOSECONDS_PER_SECOND converts the host timeval seconds field to repository duration units.
const NANOSECONDS_PER_SECOND = 1_000_000_000

// MICROSECONDS_PER_SECOND makes the timeval subsecond conversion visible as a formula.
const MICROSECONDS_PER_SECOND = 1_000_000

// NANOSECONDS_PER_MICROSECOND converts the host timeval microseconds field without a literal.
const NANOSECONDS_PER_MICROSECOND = NANOSECONDS_PER_SECOND / MICROSECONDS_PER_SECOND

// New_Operating_System_Clock return read-only Clock backed by host operating system. The
// platform primitive owns monotonicity; duplicating that state in a captured guard would make
// each injected clock allocate.
func New_Operating_System_Clock() (host time.Clock) {
	defer func() { time.Clock_Invariants(host, "new_operating_system_clock.host") }()
	host = time.Clock{
		Now_Monotonic: operating_system_monotonic,
		Now_Realtime:  operating_system_realtime,
	}
	return host
}

// Clock_Now_Monotonic keeps default-package callers on the pure clock operation.
func Clock_Now_Monotonic(host time.Clock) (moment time.Monotonic_Moment) {
	defer func() { time.Monotonic_Moment_Invariants(moment, "clock_now_monotonic.moment") }()
	time.Clock_Invariants(host, "clock_now_monotonic.host")
	return time.Clock_Now_Monotonic(host)
}

// Clock_Now_Realtime keeps default-package callers on the pure clock operation.
func Clock_Now_Realtime(host time.Clock) (moment time.Moment) {
	defer func() { time.Moment_Invariants(moment, "clock_now_realtime.moment") }()
	time.Clock_Invariants(host, "clock_now_realtime.host")
	return time.Clock_Now_Realtime(host)
}

func operating_system_monotonic(state time.State) (moment time.Monotonic_Moment) {
	defer func() {
		time.Monotonic_Moment_Invariants(moment, "operating_system_monotonic.moment")
	}()
	aver.Always(state == nil, "An operating-system monotonic read carries no state.")
	return monotonic_now()
}

// Gettimeofday exists on every platform, thus the wall clock needs no platform file.
func operating_system_realtime(state time.State) (moment time.Moment) {
	defer func() { time.Moment_Invariants(moment, "operating_system_realtime.moment") }()
	aver.Always(state == nil, "An operating-system realtime read carries no state.")
	value := syscall.Timeval{}
	aver.Always(syscall.Gettimeofday(&value) == nil, "The host reports its wall clock.")
	return time.Moment(int64(value.Sec)*NANOSECONDS_PER_SECOND +
		int64(value.Usec)*NANOSECONDS_PER_MICROSECOND)
}
