//go:build sim.virtual_time

// WARN: This package is pre-alpha.
package sim

import (
	"context"
	"slices"
	"sort"
	"sync"
	"sync/atomic"
	stdtime "time"

	"github.com/james-orcales/james-orcales/shared/invariant"
)

type Time[T any] struct {
	Mutex sync.Mutex

	// The real value of time. The simulation models absolute time as CPU-driven. Time only moves
	// when there is work to do. This has a couple of implications:
	//
	// 1) If all goroutines are blocked, time will freeze.
	// 2) If an operation is scheduled to complete in the future, the simulation must have
	// enough active tasks to reach that point in time.
	NowAbsolute Moment

	// Suspended is a monotonic slice of moments indicating when suspended goroutines are
	// ready to resume. The initial capacity of suspends is the max SuspendedCount possible.
	// If invoking Battery would exceed this, the simulator would instead fast forward
	// to the next scheduled Yield, effectively bounding goroutines to avoid OOM.
	Suspended      []Yield
	SuspendedCount int

	// The simulator is deemed frozen if Time.NowAbsolute did not increase every FrozenCheckInterval
	// nanoseconds for FrozenThreshold consecutive checks. There's two ways for this to occur:
	//
	// 1) The program is doing work but neglects propelling time.
	// 2) All goroutines are blocked.
	//
	// #1 is detected with a background goroutine checking whether time is progressing.
	// #2 is prevented by the battery
	FrozenThreshold int // immutable
	// FrozenCheckInterval refers to how often in real world time, as in the actual system time
	// on your machine, to check whether the simulator is frozen.
	FrozenCheckInterval stdtime.Duration // immutable

	// "The hardest part of building a perpetual motion machine is hiding the batteries"
	//
	// Given that any operation can finish in the future and we need enough CPU-work to reach
	// said future, this requires us to have an infinite amount of work to do. Battery is a
	// user-callback function that MUST propel time forward. It is executed in another goroutine
	// when the simulator detects that a Coast would freeze the system. If the system cannot
	// create any more goroutines, it is fast forwarded to the nearest Suspended and resumes
	// those goroutines.
	Battery chan T
	Charge  func() T
	Drain   func(T)

	// Epoch represents the reference moments captured upon process initialization.
	EpochNowAbsolute  Moment // immutable
	EpochNowMonotonic Moment // immutable

	//  Physical clocks (quartz crystal oscillators) tick at fixed intervals. Time
	//  resolutions describe the smallest discrete increment that a physical clock can measure.
	//  For example, if ClockResolutionMonotonic = 100ns and Propel(30), a sequence of
	//  NowAbsolute:NowMonotonic() updates would be:
	//
	//  	0:0 -> 30:0 -> 60:0 -> 90:0 -> 120:100 -> 150:100 -> 180:100 -> 210:200.
	ClockResolutionMonotonic Duration // immutable
	// The smallest measurable increment of the system clock. Typically coarser than monotonic
	// clocks, e.g., microseconds or milliseconds.
	ClockResolutionSystem Duration // immutable

	// Jump simulates the clock jumping backwards or forwards, misrepresenting the true
	// progression of time. This can occur during manual system time modification.
	//
	// DESIGN#1: Jump models the simplest form of NowSystem desynchronization. While clocks can
	// also drift or slew, a jump suffices to prevent users from relying on NowSystem moments for
	// duration calculations or event ordering.
	//
	// DESIGN#2: Technically, this is fault injection. However, we embed it inside the time struct
	// itself because this is what distinguishes NowSystem from Monotonic.
	//
	// DESIGN#3: We avoid the term Leap because Leap commonly refers to inserting extra time at
	// fixed boundaries (e.g. leap seconds, leap years) to adjust the calendar, which does not
	// affect the causal progression of time.
	Jump       Duration
	JumpMin    Duration // immutable, inclusive
	JumpMax    Duration // immutable, inclusive
	JumpChance float32  // immutable

	// Interval after which the realtime clock is corrected, simulating NTP polling.
	// Should be seconds in powers of 2 as per the NTP standard (e.g. 64 * sim.Second, 128 * sim.Second)
	// Reference: https://www.ntp.org/documentation/4.2.8-series/poll/
	NTPInterval Duration // immutable
	NTPNext     Moment   // Monotonic

	// DESIGN: Despite assuming that the CPU is infinitely fast, latency is still accounted for
	// to prevent the user from relying on exact timings of sleep calls.
	SleepLatencyMin Duration // immutable
	SleepLatencyMax Duration // immutable

	// Unused
	NowMonotonicGuard atomic.Int64
}

// Yield is a moment where one or more suspended goroutines become ready to resume. This is
// primarily used to artificially inflate completion time of IO operations. Example:
//
//	// Suspend for 5 virtual seconds before continuing.
//	sim.Coast(5 * sim.Second)
//	// In simulation, IO operations complete instantly because everything is in-memory.
//	// Semantically, this represents receiving data from the connection after 5 seconds.
//	conn.Read(buf)
type Yield struct {
	Resume     Moment
	Goroutines []chan<- struct{}
}

func NewTime[T any](dst *Time[T]) *Time[T] {
	mysteryTimestamp := Moment(stdtime.Date(2020, stdtime.April, 9, 16, 15, 0, 0, stdtime.UTC).UnixNano())
	if dst == nil {
		dst = &Time[T]{
			FrozenThreshold:          10,
			FrozenCheckInterval:      stdtime.Second / 2,
			Suspended:                make([]Yield, 0, 1000),
			EpochNowAbsolute:         mysteryTimestamp,
			EpochNowMonotonic:        10 * Minute,
			ClockResolutionMonotonic: 50 * Nanosecond,
			ClockResolutionSystem:    5 * Microsecond,
			NTPInterval:              64 * Second,
			JumpMin:                  1 * Hour,
			JumpMax:                  1 * Day,
			JumpChance:               0.001,
			SleepLatencyMin:          10 * Microsecond,
			SleepLatencyMax:          100 * Millisecond,
		}
	}
	invariant.Sometimes(dst.EpochNowAbsolute < dst.EpochNowMonotonic, "Initial EpochNowSystem is before Unix epoch")
	invariant.Always(dst.ClockResolutionMonotonic > 0, "Time.ClockResolutionMonotonic is a positive integer")
	invariant.Always(dst.ClockResolutionSystem > 0, "Time.ClockResolutionSystem is a positive integer")
	invariant.Always(dst.NTPInterval > 0, "Time.NTPInterval is a positive integer")
	invariant.Always(dst.JumpMin > 0, "Time.JumpMin is a positive integer")
	invariant.Always(dst.JumpMax >= dst.JumpMin, "Time.JumpMax is greater than or equal to JumpMin")
	invariant.Always(dst.JumpChance >= 0 && dst.JumpChance <= 1.0, "Time.JumpChance is 0.0-1.0")

	return dst
}

func (vtime *Time[T]) Start(ctx context.Context) {
	invariant.Always(vtime.Battery != nil, "Battery is initialized")
	invariant.Always(cap(vtime.Battery) > 0, "Battery must be buffered")
	invariant.Always(vtime.Charge != nil, "Charge is initialized")
	invariant.Always(vtime.Drain != nil, "Drain is initialized")

	go func() {
		prev := Moment(-1)
		count := 0
		for {
			select {
			case <-ctx.Done():
				return
			default:
				stdtime.Sleep(stdtime.Duration(vtime.FrozenCheckInterval))
				vtime.Mutex.Lock()
				now := vtime.NowAbsolute
				vtime.Mutex.Unlock()

				invariant.Always(now.IsAfterOrAt(prev), "Absolute time is monotonically increasing")
				if prev == now {
					count++
				} else {
					count = 0
				}
				invariant.Always(count < vtime.FrozenThreshold, "Simulation is never frozen")
				prev = now
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				vtime.Battery <- vtime.Charge()
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			vtime.Drain(<-vtime.Battery)
		}
	}

	for {
		vtime.Mutex.Lock()
		if vtime.SuspendedCount == 0 {
			break
		}
		next := vtime.Suspended[0]
		invariant.Always(len(next.Goroutines) > 0, "A suspend has a goroutine associated with it")
		vtime.Suspended = vtime.Suspended[1:]
		vtime.SuspendedCount -= len(next.Goroutines)

		elapse := next.Resume.Since(vtime.NowAbsolute)
		vtime.NowAbsolute = vtime.NowAbsolute.After(elapse)

		for _, notify := range next.Goroutines {
			close(notify)
		}
		vtime.Mutex.Unlock()
	}
}

// Propel actively advances simulation time by the specified duration. All pending Suspended scheduled
// within this timeframe are resumed. The specified duration is guaranteed to elapse in the
// simulation; time advanced by other goroutines does not count toward this duration.
//
// WARN: Propel is greedy. If a Yield is scheduled for +1µs and Propel advances by 1 year, the
// resumed goroutine possibly would not run until the full year has been consumed (determined by the
// scheduler). This is acceptable in practice: Propel is intended for very small, causal increments,
// while macroscopic progress should be achieved via Coast.
func (t *Time[T]) Propel(propel Duration) {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()

	if len(t.Suspended) == 0 || t.Suspended[0].Resume.Since(t.NowAbsolute) > propel {
		t.NowAbsolute = t.NowAbsolute.After(propel)
		return
	}

	for range invariant.Until(propel + 1) {
		if len(t.Suspended) == 0 || t.Suspended[0].Resume.Since(t.NowAbsolute) > propel {
			t.NowAbsolute = t.NowAbsolute.After(propel)
			break
		}
		// broadcast:start
		next := t.Suspended[0]
		invariant.Always(len(next.Goroutines) > 0, "A suspend has a goroutine associated with it")
		t.Suspended = t.Suspended[1:]
		t.SuspendedCount -= len(next.Goroutines)

		elapse := next.Resume.Since(t.NowAbsolute)
		t.NowAbsolute = t.NowAbsolute.After(elapse)
		propel -= elapse

		for _, notify := range next.Goroutines {
			close(notify)
		}
		// broadcast:end
	}
}

// Coast passively waits for other goroutines to propel time.
//
// If coasting would cause the simulation to freeze, the simulator invokes Battery. If no additional
// goroutines can be spawned, the simulator fast-forwards to the nearest scheduled Yield and resumes
// the associated goroutines.
func (t *Time[T]) Coast(duration Duration) {
	invariant.Always(duration > 0, "Time.Coast duration is positive")
	notify := make(chan struct{})

	t.Mutex.Lock()
	defer func() {
		t.Mutex.Unlock()
		<-notify
	}()

	resume := t.NowAbsolute.After(duration)
	if t.SuspendedCount == cap(t.Suspended) {
		// broadcast:start
		next := t.Suspended[0]
		invariant.Always(len(next.Goroutines) > 0, "A suspend has a goroutine associated with it")
		t.Suspended = t.Suspended[1:]
		t.SuspendedCount -= len(next.Goroutines)

		elapse := next.Resume.Since(t.NowAbsolute)
		t.NowAbsolute = t.NowAbsolute.After(elapse)

		for _, notify := range next.Goroutines {
			close(notify)
		}
		// broadcast:end
	}

	insert := sort.Search(len(t.Suspended), func(offset int) bool {
		return t.Suspended[offset].Resume.IsAfterOrAt(resume)
	})
	if len(t.Suspended) > insert && t.Suspended[insert].Resume.IsAt(resume) {
		t.Suspended[insert].Goroutines = append(t.Suspended[insert].Goroutines, (chan<- struct{})(notify))
	} else {
		t.Suspended = slices.Insert(t.Suspended, insert, Yield{resume, []chan<- struct{}{notify}})
	}
	t.SuspendedCount++
}

// In this simulation, the CPU is assumed to be infinitely fast. Nevertheless, no event can occur
// truly instantaneously, because even in an idealized model, each action must follow some prior
// event. To capture this, every operation is treated as taking at least one unit of causal time,
// ensuring that time advances with each step in the sequence of events.
//
// Now, you only need to decide which operations to count as "work" in the simulator. Should you
// account for CPU instructions, cache misses, syscalls, function calls, inlining, etc.? In
// practice, it’s simplest to reason about the source code itself. For example, if you sum 10
// integers or search a linked list 10 elements deep, just call Cpu(10) for both. The goal is to
// introduce a source, besides IO, of timing variations that roughly scale with the workload. If you
// are still unsure remember that this function locks the simulation mutex.
//
// Usage:
//
//	func sum(ints []int) {
//		defer sim.Cpu(len(ints))
//		// ...
//	}
func (t *Time[T]) Cpu(work int) {
	t.Propel(Duration(work))
}

func (t *Time[T]) NowMonotonic() (now Moment) {
	t.Cpu(1) // Query clock

	now = truncate(now, t.ClockResolutionMonotonic)
	elapsed := now.Since(t.EpochNowAbsolute)
	return t.EpochNowMonotonic.After(elapsed)
}

func (t *Time[T]) NowSystem() (now Moment) {
	var jump Duration
	if determine(t.JumpChance) {
		jump = random(t.JumpMin, t.JumpMax)
		if determine(0.5) {
			jump *= -1
		}
	}

	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	t.NowAbsolute = t.NowAbsolute.After(1) // Query clock
	now = truncate(t.NowAbsolute, t.ClockResolutionSystem)
	if shouldSync := now.IsAfterOrAt(t.NTPNext); shouldSync {
		t.Jump = 0
		t.NTPNext = t.NTPNext.After(t.NTPInterval)
	} else {
		t.Jump += jump
		now += Moment(t.Jump)
	}
	return now
}

func (vtime *Time[T]) WithParamFrozen(threshold int, interval stdtime.Duration) *Time[T] {
	vtime.FrozenThreshold = threshold
	vtime.FrozenCheckInterval = interval
	return vtime
}
