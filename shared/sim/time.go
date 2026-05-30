// WARN: This package is pre-alpha.
// TODO: Wrap the standard library to integrate with virtual time.
package sim

import (
	"math/rand/v2"
	stdtime "time"

	"github.com/james-orcales/james-orcales/shared/invariant"
)

const (
	Nanosecond  = 1
	Microsecond = Nanosecond * 1000
	Millisecond = Microsecond * 1000
	Second      = Millisecond * 1000
	Minute      = Second * 60
	Hour        = Minute * 60
	Day         = Hour * 24
	Week        = Day * 7
)

// TODO: Have a default seed based on the current git commit hash.

// Moment describes nanoseconds elapsed since an arbitrary, clock-specific start. Only
// differences between two `Moment`s produced by the same clock are meaningful, yielding a
// `Duration`.
//
// Unit: Nanosecond
type Moment int64
type Duration int64

func (moment Moment) After(duration Duration) Moment {
	invariant.Always(duration >= 0, "Moment.Advance duration is non-negative")
	return moment + Moment(duration)
}

func (moment Moment) Until(later Moment) Duration {
	invariant.Always(later >= moment, "Moment.Since argument is later than or at receiver")
	return Duration(later - moment)
}

func (moment Moment) Since(earlier Moment) Duration {
	invariant.Always(earlier <= moment, "Moment.Since argument is earlier than or at receiver")
	return Duration(moment - earlier)
}

func (moment Moment) IsAt(m Moment) bool {
	return moment == m
}

func (moment Moment) IsBefore(m Moment) bool {
	return moment < m
}

func (moment Moment) IsAfter(m Moment) bool {
	return moment > m
}

func (moment Moment) IsBeforeOrAt(m Moment) bool {
	return moment <= m
}

func (moment Moment) IsAfterOrAt(m Moment) bool {
	return moment >= m
}

// WARN: Moment objects are not guaranteed to be nanoseconds elapsed since the UNIX epoch. This is
// simply a convenience function. Consider Monotonic moments describing system uptime versus
// SystemTime moments describing nanoseconds since UNIX epoch.
func (moment Moment) Stdtime() stdtime.Time {
	return stdtime.Unix(0, int64(moment)).UTC()
}

func (d Duration) IsLonger(other Duration) bool {
	return d > other
}

func (d Duration) IsShorter(other Duration) bool {
	return d < other
}

func (d Duration) IsEqual(other Duration) bool {
	return d == other
}

// === HELPERS =================================================================================================================================================

type _Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32
}

func random[T _Number](lo, hi T) T {
	invariant.Always(lo <= hi, "random lo is less than or equal to hi")
	invariant.Always(hi > 0, "random hi is positive")
	return lo + T(rand.Int64N(int64(hi-lo)))
}

func determine(chance float32) bool {
	invariant.Always(chance >= 0.0, "determine.chance is greater than or equal to 0")
	invariant.Always(chance <= 1.0, "determine.chance is less than or equal to 1")
	return rand.Float32() < chance
}

func truncate(moment Moment, interval Duration) Moment {
	return moment - moment%Moment(interval)
}
