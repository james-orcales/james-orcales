//go:build sim.virtual_time

// WARN: This package is pre-alpha.
package sim

import "context"

// There's two kinds of Time structs in this package, system and virtual time.
// They share the same:
//
// - fields but differ in order
// - functions but differ in implementation
//
// Only one of them can ever be declared depending on the build tags provided. This emulates static
// dispatch. This is file is the interface they adhere to.

var UniversalTime *Time[any] = NewTime[any](nil)

func Start(ctx context.Context) {
	UniversalTime.Start(ctx)
}

// You probably shouldn't be using this. Refer to sim.Cpu instead.
func Propel(d Duration) {
	UniversalTime.Propel(d)
}

func Coast(d Duration) {
	UniversalTime.Coast(d)
}

func Cpu(work int) {
	// TODO: add multiplier and jitter.
	UniversalTime.Cpu(work)
}

// Monotonic clock
//
//	A clock that produces monotonically increasing values and is unaffected by system time
//	changes. The OS exposes various monotonic clocks. This library always uses the one that
//	measures time elapsed since system boot, including suspension.
//
//	NOTE: Although monotonic clocks are intended to never decrease, hardware/kernel bugs may
//	produce such behavior. In production, this will cause a panic. In simulation, this is not
//	modeled or accounted for.
func NowMonotonic() Moment {
	return UniversalTime.NowMonotonic()
}

// System clock
//
//	The system's calendar time, which may jump forward or backward due to NTP adjustments or
//	manual clock changes.
func NowSystem() Moment {
	return UniversalTime.NowSystem()
}
