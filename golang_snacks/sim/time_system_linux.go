//go:build linux && !sim.virtual_time

// WARN: This package is pre-alpha.
// WARN: I have not tested this yet!
package sim

import (
	"sync"
	"sync/atomic"
	"syscall"
	stdtime "time"
)

type Time struct {
	/*
		Reference: https://github.com/tigerbeetle/tigerbeetle/blob/fff8abc12593e72629c95f3dfd3809ba18f4667f/src/time.zig

			pub const TimeOS = struct {
			    /// Hardware and/or software bugs can mean that the monotonic t may regress.
			    /// One example (of many): https://bugzilla.redhat.com/show_bug.cgi?id=448449
			    /// We crash the process for safety if this ever happens, to protect against infinite loops.
			    /// It's better to crash and come back with a valid monotonic t than get stuck forever.
			    monotonic_guard: u64 = 0,
			    ...
	*/
	MonotonicGuard atomic.Int64

	// === Unused ===

	Mutex         sync.Mutex
	GoroutinePool *spawn.Pool

	NowAbsolute Moment

	Yields      []Yield
	YieldsCount int

	FrozenThreshold     int
	FrozenCheckInterval stdtime.Duration

	Battery func()

	EpochNowAbsolute  Moment
	EpochNowMonotonic Moment

	MonotonicClockResolution Duration
	SystemClockResolution    Duration

	Jump       Duration
	JumpMin    Duration
	JumpMax    Duration
	JumpChance float32

	NTPInterval Duration
	NTPNext     Moment

	SleepLatencyMin Duration
	SleepLatencyMax Duration
}

type Yield struct {
	Resume     Moment
	Goroutines []chan<- struct{}
}

func NewTime(_ *Time) *Time {
	result := &Time{}
	result.NowMonotonic()
	return result
}

func (t *Time) Main(main func()) {
	main()
}

/*
Reference: https://github.com/tigerbeetle/tigerbeetle/blob/fff8abc12593e72629c95f3dfd3809ba18f4667f/src/time.zig

	fn monotonic_linux() u64 {
	    assert(is_linux);
	    // The true monotonic t on Linux is not in fact CLOCK_MONOTONIC:
	    //
	    // CLOCK_MONOTONIC excludes elapsed time while the system is suspended (e.g. VM migration).
	    //
	    // CLOCK_BOOTTIME is the same as CLOCK_MONOTONIC but includes elapsed time during a suspend.
	    //
	    // For more detail and why CLOCK_MONOTONIC_RAW is even worse than CLOCK_MONOTONIC, see
	    // https://github.com/ziglang/zig/pull/933#discussion_r656021295.
	    const ts: posix.timespec = posix.t_gettime(posix.CLOCK.BOOTTIME) catch {
	        @panic("CLOCK_BOOTTIME required");
	    };
	    return @as(u64, @intCast(ts.sec)) * std.time.ns_per_s + @as(u64, @intCast(ts.nsec));
	}
*/
func (t *Time) NowMonotonic() Moment {
	var ts syscall.Timespec
	syscall.ClockGettime(0x7, &ts) // CLOCK_BOOTTIME = 0x7
	now := int64(ts.Sec*Second + ts.Nsec)
	before := t.MonotonicGuard.Swap(now)
	if now < before {
		panic("a hardware/kernel bug regressed the hardware time")
	}
	return Moment(now)
}

func (t *Time) NowSystem() Moment {
	return Moment(stdtime.Now().UnixNano())
}

func (t *Time) Propel(_ Duration) {}
func (t *Time) Coast(_ Duration)  {}
