package time_test

import (
	"testing"
	"unsafe"

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/simulation/time"
	"local/james-orcales/shared/testify"
)

// Test_Civil_Calendar converts Unix-relative days across leap-century boundaries and returns each
// exact input through inverse conversion.
func Test_Civil_Calendar(t *testing.T) {
	testify.Zero(t, time.ZONE_OFFSET_SECONDS_UTC)
	for _, check := range []struct {
		Days  time.Calendar_Day_Count
		Year  time.Civil_Year
		Month time.Civil_Month
		Day   time.Civil_Day
	}{
		{Days: -25_508, Year: 1900, Month: 3, Day: 1},
		{Days: -1, Year: 1969, Month: 12, Day: 31},
		{Days: 0, Year: 1970, Month: 1, Day: 1},
		{Days: 3_652, Year: 1980, Month: 1, Day: 1},
		{Days: 11_016, Year: 2000, Month: 2, Day: 29},
		{Days: 11_017, Year: 2000, Month: 3, Day: 1},
		{Days: 47_541, Year: 2100, Month: 3, Day: 1},
	} {
		year, month, day := time.Civil_From_Days(check.Days)
		testify.Equal(t, check.Year, year)
		testify.Equal(t, check.Month, month)
		testify.Equal(t, check.Day, day)
		testify.Equal(t, check.Days, time.Days_From_Civil(year, month, day))
	}
	for _, check := range []struct {
		Seconds     time.Unix_Second_Count
		Days        time.Calendar_Day_Count
		Day_Seconds time.Day_Second_Count
	}{
		{Seconds: time.Unix_Second_Count(-time.SECOND_COUNT_PER_DAY - 1), Days: -2,
			Day_Seconds: time.Day_Second_Count(time.SECOND_COUNT_PER_DAY - 1)},
		{Seconds: -1, Days: -1,
			Day_Seconds: time.Day_Second_Count(time.SECOND_COUNT_PER_DAY - 1)},
		{Seconds: 0, Days: 0, Day_Seconds: 0},
		{Seconds: time.Unix_Second_Count(time.SECOND_COUNT_PER_DAY), Days: 1,
			Day_Seconds: 0},
	} {
		days, day_seconds := time.Unix_Second_Split(check.Seconds)
		testify.Equal(t, check.Days, days)
		testify.Equal(t, check.Day_Seconds, day_seconds)
	}
}

// Test_Virtual_Clock_Monotonic check deterministic clock advance exactly one resolution per
// Tick. Also check Now_Realtime is epoch plus elapsed monotonic span, when no skew.
func Test_Virtual_Clock_Monotonic(t *testing.T) {
	virtual := time.Virtual_Clock{Resolution: 50, Epoch: 1000}
	c := time.Virtual_Clock_To_Clock(&virtual)

	testify.Zero(t, time.Clock_Now_Monotonic(c))
	testify.Equal(t, time.Moment(1000), time.Clock_Now_Realtime(c))
	time.Virtual_Clock_Tick(&virtual)
	testify.Equal(t, time.Monotonic_Moment(50), time.Clock_Now_Monotonic(c))
	testify.Equal(t, time.Moment(1050), time.Clock_Now_Realtime(c))
}

// Test_Virtual_Clock_Skew check modeled skew bend Now_Realtime away from true elapsed time,
// and leave Now_Monotonic untouched. Clock here lose one nanosecond of realtime per tick.
func Test_Virtual_Clock_Skew(t *testing.T) {
	virtual := time.Virtual_Clock{
		Resolution: 1000,
		Epoch:      0,
		Skew:       time.Skew(time.SKEW_KIND_LINEAR, 1, 0),
	}
	c := time.Virtual_Clock_To_Clock(&virtual)
	time.Virtual_Clock_Tick(&virtual)
	time.Virtual_Clock_Tick(&virtual)
	// Monotonic = 2*1000 = 2000; skew = 2*1 = 2; realtime = 0 + 2000 - 2 = 1998.
	testify.Equal(t, time.Monotonic_Moment(2000), time.Clock_Now_Monotonic(c))
	testify.Equal(t, time.Moment(1998), time.Clock_Now_Realtime(c))

	// Periodic skew of amplitude 1000 over four-tick period peak at quarter turn. At tick 1,
	// sin(2pi/4) = 1, thus skew equal full amplitude and cancel elapsed span exactly.
	periodic_virtual := time.Virtual_Clock{
		Resolution: 1000,
		Epoch:      0,
		Skew:       time.Skew(time.SKEW_KIND_PERIODIC, 1000, 4),
	}
	periodic := time.Virtual_Clock_To_Clock(&periodic_virtual)
	time.Virtual_Clock_Tick(&periodic_virtual)
	testify.Zero(t, time.Clock_Now_Realtime(periodic))
}

// Test_Monotonic_Moment check uptime domain: zero at boot, one year of nanoseconds at maximum.
// Maximum computed here from own factors, thus change to unit ladder of package cannot move
// bound without this test.
func Test_Monotonic_Moment(t *testing.T) {
	t.Parallel()
	const ONE_YEAR_NANOSECONDS int64 = 365 * 24 * 60 * 60 * 1000 * 1000 * 1000
	testify.Zero(t, time.MONOTONIC_MOMENT_MINIMUM)
	testify.Equal(t, ONE_YEAR_NANOSECONDS, int64(time.MONOTONIC_MOMENT_MAXIMUM))
	// Bound hold at the reader, not at each consumer: backend that hand out a moment past one
	// year fail on the read, before any holder store it.
	past_bound := time.Clock{
		Now_Monotonic: past_bound_monotonic,
		Now_Realtime:  invariant_now_realtime,
	}
	testify.Panics(t, func() { time.Clock_Now_Monotonic(past_bound) })
}

func past_bound_monotonic(_ unsafe.Pointer) (moment time.Monotonic_Moment) {
	return time.MONOTONIC_MOMENT_MAXIMUM + 1
}

func invariant_now_realtime(_ unsafe.Pointer) (moment time.Moment) {
	return 0
}

// SPECIAL_VALUE_COUNT: how many special values full-width signed domain have. Two bounds, plus
// four interior sentinels framework expand.
const SPECIAL_VALUE_COUNT = 6

// Signed special values every full-width domain in this package state.
func special_values() (values [SPECIAL_VALUE_COUNT]int64) {
	return [...]int64{
		bits.INTEGER_64_MINIMUM,
		bits.INTEGER_64_MAXIMUM,
		0,
		1,
		2,
		-1,
	}
}

// Hit each special value at two scalars virtual clock take. Clock only built, never ticked.
// Extreme resolution or epoch is legal configuration. Product it would read is not what these
// domains state.
func verify_virtual_clock_domains() {
	for _, value := range special_values() {
		resolution := time.Virtual_Clock{Resolution: time.Duration(value)}
		time.Virtual_Clock_To_Clock(&resolution)
		epoch := time.Virtual_Clock{Epoch: time.Moment(value)}
		time.Virtual_Clock_To_Clock(&epoch)
	}
}

// Hit each skew model, and each special value at both coefficients.
func verify_skew_domains() {
	kinds := [...]time.Skew_Kind{
		time.SKEW_KIND_LINEAR,
		time.SKEW_KIND_PERIODIC,
		time.SKEW_KIND_STEP,
	}
	for _, kind := range kinds {
		time.Skew(kind, 0, 0)
	}
	for _, value := range special_values() {
		time.Skew(time.SKEW_KIND_LINEAR, time.Duration(value), 0)
		time.Skew(time.SKEW_KIND_LINEAR, 0, time.Tick_Count(value))
	}
}

// Hit uptime domain at each special value framework expand. Resolution of one year reach upper
// bound on first tick, thus sweep cost one tick, not year of them.
func verify_uptime_domains() {
	grain_virtual := time.Virtual_Clock{Resolution: time.NANOSECOND}
	grain := time.Virtual_Clock_To_Clock(&grain_virtual)
	time.Clock_Now_Monotonic(grain)
	time.Virtual_Clock_Tick(&grain_virtual)
	time.Clock_Now_Monotonic(grain)
	time.Virtual_Clock_Tick(&grain_virtual)
	time.Clock_Now_Monotonic(grain)

	bound := time.Virtual_Clock{Resolution: time.Duration(time.MONOTONIC_MOMENT_MAXIMUM)}
	one_year := time.Virtual_Clock_To_Clock(&bound)
	time.Virtual_Clock_Tick(&bound)
	time.Clock_Now_Monotonic(one_year)
}

// Conversion entry points must observe every scalar sentinel the invariant framework expands.
func verify_civil_calendar_domains() {
	for _, seconds := range [...]time.Unix_Second_Count{
		time.Unix_Second_Count(time.UNIX_SECOND_COUNT_MINIMUM),
		time.Unix_Second_Count(time.UNIX_SECOND_COUNT_MAXIMUM),
		0,
		1,
		2,
		-1,
		time.Unix_Second_Count(time.SECOND_COUNT_PER_DAY),
		time.Unix_Second_Count(2 * time.SECOND_COUNT_PER_DAY),
	} {
		time.Unix_Second_Split(seconds)
	}
	for _, days := range [...]time.Calendar_Day_Count{
		time.Calendar_Day_Count(time.CALENDAR_DAY_COUNT_MINIMUM),
		time.Calendar_Day_Count(time.CALENDAR_DAY_COUNT_MAXIMUM),
		0,
		1,
		2,
		-1,
	} {
		year, month, day := time.Civil_From_Days(days)
		time.Days_From_Civil(year, month, day)
	}
}

// Test_Invariant_Domains check special values of each admitted scalar domain.
func Test_Invariant_Domains(t *testing.T) {
	t.Parallel()
	verify_virtual_clock_domains()
	verify_skew_domains()
	verify_uptime_domains()
	verify_civil_calendar_domains()
}

// Escaped results expose heap ownership hidden by stack-only use.
func verify_time_constructor_allocations(t *testing.T) {
	t.Run("Skew", func(t *testing.T) {
		kinds := [...]time.Skew_Kind{
			time.SKEW_KIND_LINEAR,
			time.SKEW_KIND_PERIODIC,
			time.SKEW_KIND_STEP,
		}
		expected := [...]time.Duration{2, 0, 0}
		for index, kind := range kinds {
			var offset time.Offset
			testify.Zero_Allocation(t, func() { offset = time.Skew(kind, 1, 1) })
			testify.Equal(t, expected[index], time.Offset_Read(offset, 1))
		}
	})
	t.Run("Virtual_Clock_To_Clock", func(t *testing.T) {
		virtual := time.Virtual_Clock{Resolution: 1}
		var host time.Clock
		testify.Zero_Allocation(t, func() {
			host = time.Virtual_Clock_To_Clock(&virtual)
		})
		testify.Not_Nil(t, host.Now_Monotonic)
		testify.Not_Nil(t, host.State)
	})
}

func verify_time_reader_allocations(t *testing.T) {
	t.Run("Unix_Second_Split", func(t *testing.T) {
		var days time.Calendar_Day_Count
		var day_seconds time.Day_Second_Count
		testify.Zero_Allocation(t, func() {
			days, day_seconds = time.Unix_Second_Split(0)
		})
		testify.Zero(t, days)
		testify.Zero(t, day_seconds)
	})
	t.Run("Civil_From_Days", func(t *testing.T) {
		var year time.Civil_Year
		var month time.Civil_Month
		var day time.Civil_Day
		testify.Zero_Allocation(t, func() {
			year, month, day = time.Civil_From_Days(0)
		})
		testify.Equal(t, time.Civil_Year(time.CIVIL_UNIX_EPOCH_YEAR), year)
		testify.Equal(t, time.Civil_Month(time.CIVIL_MONTH_MINIMUM), month)
		testify.Equal(t, time.Civil_Day(time.CIVIL_DAY_MINIMUM), day)
	})
	t.Run("Days_From_Civil", func(t *testing.T) {
		var days time.Calendar_Day_Count
		testify.Zero_Allocation(t, func() {
			days = time.Days_From_Civil(
				time.Civil_Year(time.CIVIL_UNIX_EPOCH_YEAR),
				time.Civil_Month(time.CIVIL_MONTH_MINIMUM),
				time.Civil_Day(time.CIVIL_DAY_MINIMUM),
			)
		})
		testify.Zero(t, days)
	})
	t.Run("Offset_Read", func(t *testing.T) {
		offset := time.Skew(time.SKEW_KIND_LINEAR, 1, 1)
		var skew time.Duration
		testify.Zero_Allocation(t, func() { skew = time.Offset_Read(offset, 1) })
		testify.Equal(t, time.Duration(2), skew)
	})
	virtual := time.Virtual_Clock{Resolution: 1}
	host := time.Virtual_Clock_To_Clock(&virtual)
	time.Virtual_Clock_Tick(&virtual)
	var monotonic time.Monotonic_Moment
	var realtime time.Moment
	t.Run("Now_Monotonic", func(t *testing.T) {
		testify.Zero_Allocation(t, func() { monotonic = time.Clock_Now_Monotonic(host) })
		testify.Positive(t, monotonic)
	})
	t.Run("Now_Realtime", func(t *testing.T) {
		testify.Zero_Allocation(t, func() { realtime = time.Clock_Now_Realtime(host) })
		testify.Positive(t, realtime)
	})
	t.Run("Tick", func(t *testing.T) {
		testify.Zero_Allocation(t, func() { time.Virtual_Clock_Tick(&virtual) })
	})
}

// Every clock API path needs direct heap evidence; functional tests cannot establish it.
func Test_Clock_API_Heap_Allocation(t *testing.T) {
	verify_time_constructor_allocations(t)
	verify_time_reader_allocations(t)
}
