// Package clock gives time as an injected dependency: a read-only Clock vtable over explicit
// caller-owned state, the moment and duration arithmetic it read in, and the civil calendar.
// Production wires an OS clock (clock/default). A simulation wires a Virtual one, or a view
// over the loop counter in sim/nbio. The code between never knows which one it holds.
// Nothing here advance time; that is the loop's job, and the loop live in sim/nbio.
package time

import (
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/math/fixedpoint"
	"local/james-orcales/shared/sim/aver/default"
)

// Moment is a clock reading in nanoseconds. Its epoch is arbitrary and belongs to one clock.
// Only the difference between two Moments from the SAME clock has a meaning. A monotonic
// Moment and a realtime Moment do not compare.
type Moment int64

// Moment_Invariants state complete clock-reading domain.
func Moment_Invariants(moment Moment, namespace aver.Namespace) {
	aver.Tree(moment, namespace).
		Range_Int64(int64(moment), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Duration: span of nanoseconds.
type Duration int64

// Duration_Invariants state complete nanosecond-span domain.
func Duration_Invariants(duration Duration, namespace aver.Namespace) {
	aver.Tree(duration, namespace).
		Range_Int64(int64(duration), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// NANOSECOND: unit Duration count in.
const NANOSECOND Duration = 1

// MICROSECOND: thousand nanoseconds.
const MICROSECOND = NANOSECOND * 1000

// MILLISECOND: thousand microseconds.
const MILLISECOND = MICROSECOND * 1000

// SECOND: thousand milliseconds.
const SECOND = MILLISECOND * 1000

// MINUTE: sixty seconds.
const MINUTE = SECOND * 60

// HOUR: sixty minutes.
const HOUR = MINUTE * 60

// DAY: twenty-four hours.
const DAY = HOUR * 24

// WEEK: seven days.
const WEEK = DAY * 7

// NANOSECOND_COUNT_PER_SECOND stays beside clock units so timestamp packages do not recreate it.
const NANOSECOND_COUNT_PER_SECOND = int32(SECOND / NANOSECOND)

// NANOSECOND_COUNT_MINIMUM begins normalized fractional seconds.
const NANOSECOND_COUNT_MINIMUM int32 = 0

// NANOSECOND_COUNT_MAXIMUM closes normalized fractional seconds.
const NANOSECOND_COUNT_MAXIMUM = NANOSECOND_COUNT_PER_SECOND - 1

// SECOND_COUNT_PER_MINUTE derives civil arithmetic from clock units.
const SECOND_COUNT_PER_MINUTE = int64(MINUTE / SECOND)

// SECOND_COUNT_PER_HOUR derives civil arithmetic from clock units.
const SECOND_COUNT_PER_HOUR = int64(HOUR / SECOND)

// SECOND_COUNT_PER_DAY derives civil arithmetic from clock units.
const SECOND_COUNT_PER_DAY = int64(DAY / SECOND)

// CIVIL_COMMON_YEAR_DAY_COUNT anchors proleptic Gregorian arithmetic.
const CIVIL_COMMON_YEAR_DAY_COUNT int64 = 365

// CIVIL_LEAP_YEAR_INTERVAL states regular Gregorian leap cadence.
const CIVIL_LEAP_YEAR_INTERVAL int64 = 4

// CIVIL_CENTURY_YEAR_INTERVAL removes non-400th century leap days.
const CIVIL_CENTURY_YEAR_INTERVAL int64 = 100

// CIVIL_ERA_YEAR_INTERVAL restores each 400th-year leap day.
const CIVIL_ERA_YEAR_INTERVAL int64 = 400

// CIVIL_ERA_LEAP_DAY_COUNT derives leap days in one Gregorian era.
const CIVIL_ERA_LEAP_DAY_COUNT = CIVIL_ERA_YEAR_INTERVAL/CIVIL_LEAP_YEAR_INTERVAL -
	CIVIL_ERA_YEAR_INTERVAL/CIVIL_CENTURY_YEAR_INTERVAL + 1

// CIVIL_ERA_DAY_COUNT derives complete Gregorian era size.
const CIVIL_ERA_DAY_COUNT = CIVIL_ERA_YEAR_INTERVAL*CIVIL_COMMON_YEAR_DAY_COUNT +
	CIVIL_ERA_LEAP_DAY_COUNT

// CIVIL_ERA_FINAL_DAY_INDEX supports exact final-era year extraction.
const CIVIL_ERA_FINAL_DAY_INDEX = CIVIL_ERA_DAY_COUNT - 1

// CIVIL_QUADRENNIAL_COMMON_DAY_COUNT supports Gregorian year extraction.
const CIVIL_QUADRENNIAL_COMMON_DAY_COUNT = CIVIL_LEAP_YEAR_INTERVAL *
	CIVIL_COMMON_YEAR_DAY_COUNT

// CIVIL_CENTURY_LEAP_DAY_COUNT derives leap days before one century boundary.
const CIVIL_CENTURY_LEAP_DAY_COUNT = (CIVIL_CENTURY_YEAR_INTERVAL-1)/
	CIVIL_LEAP_YEAR_INTERVAL -
	(CIVIL_CENTURY_YEAR_INTERVAL-1)/CIVIL_CENTURY_YEAR_INTERVAL

// CIVIL_CENTURY_DAY_COUNT supports Gregorian year extraction.
const CIVIL_CENTURY_DAY_COUNT = CIVIL_CENTURY_YEAR_INTERVAL*CIVIL_COMMON_YEAR_DAY_COUNT +
	CIVIL_CENTURY_LEAP_DAY_COUNT

// CIVIL_MARCH_MONTH_GROUP_DAY_COUNT is exact numerator for March-based month extraction.
const CIVIL_MARCH_MONTH_GROUP_DAY_COUNT int64 = 153

// CIVIL_MARCH_MONTH_GROUP_COUNT is exact denominator for March-based month extraction.
const CIVIL_MARCH_MONTH_GROUP_COUNT int64 = 5

// CIVIL_MARCH_MONTH_GROUP_BIAS keeps integer month transform exact.
const CIVIL_MARCH_MONTH_GROUP_BIAS int64 = 2

// CIVIL_MARCH_EPOCH_MONTH moves leap day to arithmetic year end.
const CIVIL_MARCH_EPOCH_MONTH = 3

// CIVIL_UNIX_EPOCH_YEAR anchors Unix-relative calendar days.
const CIVIL_UNIX_EPOCH_YEAR int64 = 1970

// CIVIL_UNIX_EPOCH_PREVIOUS_YEAR closes leap counting before Unix epoch.
const CIVIL_UNIX_EPOCH_PREVIOUS_YEAR = CIVIL_UNIX_EPOCH_YEAR - 1

// CIVIL_UNIX_EPOCH_ADJUSTED_YEAR moves January into prior arithmetic year.
const CIVIL_UNIX_EPOCH_ADJUSTED_YEAR = CIVIL_UNIX_EPOCH_PREVIOUS_YEAR

// CIVIL_UNIX_EPOCH_ERA locates Unix epoch inside Gregorian eras.
const CIVIL_UNIX_EPOCH_ERA = CIVIL_UNIX_EPOCH_ADJUSTED_YEAR /
	CIVIL_ERA_YEAR_INTERVAL

// CIVIL_UNIX_EPOCH_YEAR_OF_ERA locates Unix epoch inside its era.
const CIVIL_UNIX_EPOCH_YEAR_OF_ERA = CIVIL_UNIX_EPOCH_ADJUSTED_YEAR -
	CIVIL_UNIX_EPOCH_ERA*CIVIL_ERA_YEAR_INTERVAL

// CIVIL_MONTH_MINIMUM begins Gregorian year.
const CIVIL_MONTH_MINIMUM = 1

// CIVIL_MONTH_MAXIMUM closes Gregorian year.
const CIVIL_MONTH_MAXIMUM = 12

// CIVIL_DAY_MINIMUM begins one month.
const CIVIL_DAY_MINIMUM = 1

// CIVIL_DAY_MAXIMUM admits longest Gregorian month.
const CIVIL_DAY_MAXIMUM = 31

// CIVIL_JANUARY_MARCH_INDEX locates January inside March-based year.
const CIVIL_JANUARY_MARCH_INDEX = CIVIL_MONTH_MAXIMUM - CIVIL_MARCH_EPOCH_MONTH + 1

// CIVIL_UNIX_EPOCH_DAY_OF_YEAR derives January 1 inside March-based year.
const CIVIL_UNIX_EPOCH_DAY_OF_YEAR = (CIVIL_MARCH_MONTH_GROUP_DAY_COUNT*
	CIVIL_JANUARY_MARCH_INDEX+CIVIL_MARCH_MONTH_GROUP_BIAS)/
	CIVIL_MARCH_MONTH_GROUP_COUNT + CIVIL_DAY_MINIMUM - 1

// CIVIL_UNIX_EPOCH_DAY_OFFSET converts March-based days to Unix-relative days.
const CIVIL_UNIX_EPOCH_DAY_OFFSET = CIVIL_UNIX_EPOCH_ERA*CIVIL_ERA_DAY_COUNT +
	CIVIL_UNIX_EPOCH_YEAR_OF_ERA*CIVIL_COMMON_YEAR_DAY_COUNT +
	CIVIL_UNIX_EPOCH_YEAR_OF_ERA/CIVIL_LEAP_YEAR_INTERVAL -
	CIVIL_UNIX_EPOCH_YEAR_OF_ERA/CIVIL_CENTURY_YEAR_INTERVAL +
	CIVIL_UNIX_EPOCH_DAY_OF_YEAR

// UNIX_SECOND_COUNT_MINIMUM keeps second conversion inside Moment storage.
const UNIX_SECOND_COUNT_MINIMUM = bits.INTEGER_64_MINIMUM / int64(SECOND)

// UNIX_SECOND_COUNT_MAXIMUM keeps second conversion inside Moment storage.
const UNIX_SECOND_COUNT_MAXIMUM = bits.INTEGER_64_MAXIMUM / int64(SECOND)

// CALENDAR_DAY_COUNT_MINIMUM includes negative remainder floor at minimum Unix second.
const CALENDAR_DAY_COUNT_MINIMUM = UNIX_SECOND_COUNT_MINIMUM/SECOND_COUNT_PER_DAY - 1

// CALENDAR_DAY_COUNT_MAXIMUM closes Moment-representable Unix days.
const CALENDAR_DAY_COUNT_MAXIMUM = UNIX_SECOND_COUNT_MAXIMUM / SECOND_COUNT_PER_DAY

// CIVIL_YEAR_MINIMUM includes partial first Moment-representable year.
const CIVIL_YEAR_MINIMUM = CIVIL_UNIX_EPOCH_YEAR +
	CALENDAR_DAY_COUNT_MINIMUM/CIVIL_COMMON_YEAR_DAY_COUNT - 1

// CIVIL_YEAR_MAXIMUM includes partial final Moment-representable year.
const CIVIL_YEAR_MAXIMUM = CIVIL_UNIX_EPOCH_YEAR +
	CALENDAR_DAY_COUNT_MAXIMUM/CIVIL_COMMON_YEAR_DAY_COUNT

// DAY_SECOND_COUNT_MINIMUM begins one civil day.
const DAY_SECOND_COUNT_MINIMUM int64 = 0

// DAY_SECOND_COUNT_MAXIMUM closes one civil day.
const DAY_SECOND_COUNT_MAXIMUM = SECOND_COUNT_PER_DAY - 1

// ZONE_OFFSET_SECONDS_MINIMUM includes westernmost current civil offset.
const ZONE_OFFSET_SECONDS_MINIMUM int32 = -12 * int32(SECOND_COUNT_PER_HOUR)

// ZONE_OFFSET_SECONDS_MAXIMUM includes easternmost current civil offset.
const ZONE_OFFSET_SECONDS_MAXIMUM int32 = 14 * int32(SECOND_COUNT_PER_HOUR)

// ZONE_OFFSET_SECONDS_UTC is the civil origin between west and east offsets.
const ZONE_OFFSET_SECONDS_UTC = ZONE_OFFSET_SECONDS_MINIMUM - ZONE_OFFSET_SECONDS_MINIMUM

// Nanosecond_Count is normalized fraction shared by calendar timestamps.
type Nanosecond_Count int32

// Nanosecond_Count_Invariants rejects fraction outside one second.
func Nanosecond_Count_Invariants(value Nanosecond_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), NANOSECOND_COUNT_MINIMUM, NANOSECOND_COUNT_MAXIMUM).
		Ensure()
}

// Zone_Offset_Seconds is caller-selected civil displacement from UTC.
type Zone_Offset_Seconds int32

// Zone_Offset_Seconds_Invariants rejects offsets outside current civil range.
func Zone_Offset_Seconds_Invariants(
	value Zone_Offset_Seconds, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), ZONE_OFFSET_SECONDS_MINIMUM, ZONE_OFFSET_SECONDS_MAXIMUM).
		Ensure()
}

// Unix_Second_Count keeps calendar conversion inside Moment storage.
type Unix_Second_Count int64

// Unix_Second_Count_Invariants prevents later nanosecond conversion overflow.
func Unix_Second_Count_Invariants(value Unix_Second_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), UNIX_SECOND_COUNT_MINIMUM, UNIX_SECOND_COUNT_MAXIMUM).
		Ensure()
}

// Calendar_Day_Count is one Unix-relative proleptic Gregorian day.
type Calendar_Day_Count int64

// Calendar_Day_Count_Invariants keeps Gregorian arithmetic inside Moment domain.
func Calendar_Day_Count_Invariants(value Calendar_Day_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), CALENDAR_DAY_COUNT_MINIMUM, CALENDAR_DAY_COUNT_MAXIMUM).
		Ensure()
}

// Civil_Year is one Moment-representable proleptic Gregorian year.
type Civil_Year int64

// Civil_Year_Invariants includes partial years at both Moment boundaries.
func Civil_Year_Invariants(value Civil_Year, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), CIVIL_YEAR_MINIMUM, CIVIL_YEAR_MAXIMUM).
		Ensure()
}

// Civil_Month is one Gregorian month.
type Civil_Month int

// Civil_Month_Invariants rejects non-calendar month numbers.
func Civil_Month_Invariants(value Civil_Month, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), CIVIL_MONTH_MINIMUM, CIVIL_MONTH_MAXIMUM).
		Ensure()
}

// Civil_Day admits each day number before month-specific normalization.
type Civil_Day int

// Civil_Day_Invariants rejects numbers no Gregorian month can contain.
func Civil_Day_Invariants(value Civil_Day, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), CIVIL_DAY_MINIMUM, CIVIL_DAY_MAXIMUM).
		Ensure()
}

// Day_Second_Count is normalized second offset inside one civil day.
type Day_Second_Count int64

// Day_Second_Count_Invariants rejects seconds outside one civil day.
func Day_Second_Count_Invariants(value Day_Second_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), DAY_SECOND_COUNT_MINIMUM, DAY_SECOND_COUNT_MAXIMUM).
		Ensure()
}

// Day_Split is one Unix second as a day count and the second inside that day. One type, thus
// one deferred assertion covers both halves.
type Day_Split struct {
	// Days is the Unix-relative Gregorian day.
	Days Calendar_Day_Count
	// Day_Seconds is the normalized second inside Days.
	Day_Seconds Day_Second_Count
}

// Day_Split_Invariants keeps both halves inside their own domains.
func Day_Split_Invariants(split Day_Split, namespace aver.Namespace) {
	Calendar_Day_Count_Invariants(split.Days, namespace)
	Day_Second_Count_Invariants(split.Day_Seconds, namespace)
}

// Civil_Date is one proleptic Gregorian date. One type, thus one deferred assertion covers
// all three fields.
type Civil_Date struct {
	// Year is the Moment-representable year.
	Year Civil_Year
	// Month is the calendar month.
	Month Civil_Month
	// Day is the day number before month-specific normalization.
	Day Civil_Day
}

// Civil_Date_Invariants keeps each field inside its own domain.
func Civil_Date_Invariants(date Civil_Date, namespace aver.Namespace) {
	Civil_Year_Invariants(date.Year, namespace)
	Civil_Month_Invariants(date.Month, namespace)
	Civil_Day_Invariants(date.Day, namespace)
}

// Unix_Second_Split floors negative values because truncation would produce negative day seconds.
func Unix_Second_Split(seconds Unix_Second_Count) (split Day_Split) {
	defer func() { Day_Split_Invariants(split, "unix_second_split.split") }()
	Unix_Second_Count_Invariants(seconds, "unix_second_split.seconds")
	days := int64(seconds) / SECOND_COUNT_PER_DAY
	remainder := int64(seconds) % SECOND_COUNT_PER_DAY
	if remainder < 0 {
		days--
		remainder += SECOND_COUNT_PER_DAY
	}
	return Day_Split{Days: Calendar_Day_Count(days), Day_Seconds: Day_Second_Count(remainder)}
}

// Civil_From_Days uses era arithmetic so leap rules need no ambient timezone or lookup table.
func Civil_From_Days(days Calendar_Day_Count) (date Civil_Date) {
	defer func() { Civil_Date_Invariants(date, "civil_from_days.date") }()
	Calendar_Day_Count_Invariants(days, "civil_from_days.days")
	shifted_days := int64(days) + CIVIL_UNIX_EPOCH_DAY_OFFSET
	era := shifted_days / CIVIL_ERA_DAY_COUNT
	if shifted_days < 0 {
		if shifted_days%CIVIL_ERA_DAY_COUNT != 0 {
			era--
		}
	}
	day_of_era := shifted_days - era*CIVIL_ERA_DAY_COUNT
	year_of_era := (day_of_era - day_of_era/CIVIL_QUADRENNIAL_COMMON_DAY_COUNT +
		day_of_era/CIVIL_CENTURY_DAY_COUNT -
		day_of_era/CIVIL_ERA_FINAL_DAY_INDEX) / CIVIL_COMMON_YEAR_DAY_COUNT
	year := Civil_Year(year_of_era + era*CIVIL_ERA_YEAR_INTERVAL)
	day_of_year := day_of_era -
		(CIVIL_COMMON_YEAR_DAY_COUNT*year_of_era +
			year_of_era/CIVIL_LEAP_YEAR_INTERVAL -
			year_of_era/CIVIL_CENTURY_YEAR_INTERVAL)
	month_prime := (CIVIL_MARCH_MONTH_GROUP_COUNT*day_of_year +
		CIVIL_MARCH_MONTH_GROUP_BIAS) / CIVIL_MARCH_MONTH_GROUP_DAY_COUNT
	day := Civil_Day(day_of_year-(CIVIL_MARCH_MONTH_GROUP_DAY_COUNT*month_prime+
		CIVIL_MARCH_MONTH_GROUP_BIAS)/CIVIL_MARCH_MONTH_GROUP_COUNT) + CIVIL_DAY_MINIMUM
	month := Civil_Month(month_prime + CIVIL_MARCH_EPOCH_MONTH)
	if month > CIVIL_MONTH_MAXIMUM {
		month -= CIVIL_MONTH_MAXIMUM
		year++
	}
	return Civil_Date{Year: year, Month: month, Day: day}
}

// Days_From_Civil uses same March epoch so it is exact inverse over representable dates.
func Days_From_Civil(date Civil_Date) (days Calendar_Day_Count) {
	defer func() { Calendar_Day_Count_Invariants(days, "days_from_civil.days") }()
	Civil_Date_Invariants(date, "days_from_civil.date")
	year, month, day := date.Year, date.Month, date.Day
	adjusted_year := int64(year)
	if month < CIVIL_MARCH_EPOCH_MONTH {
		adjusted_year--
	}
	era := adjusted_year / CIVIL_ERA_YEAR_INTERVAL
	if adjusted_year < 0 {
		if adjusted_year%CIVIL_ERA_YEAR_INTERVAL != 0 {
			era--
		}
	}
	year_of_era := adjusted_year - era*CIVIL_ERA_YEAR_INTERVAL
	month_prime := int64(month) - CIVIL_MARCH_EPOCH_MONTH
	if month_prime < 0 {
		month_prime += CIVIL_MONTH_MAXIMUM
	}
	day_of_year := (CIVIL_MARCH_MONTH_GROUP_DAY_COUNT*month_prime+
		CIVIL_MARCH_MONTH_GROUP_BIAS)/CIVIL_MARCH_MONTH_GROUP_COUNT +
		int64(day) - CIVIL_DAY_MINIMUM
	day_of_era := year_of_era*CIVIL_COMMON_YEAR_DAY_COUNT +
		year_of_era/CIVIL_LEAP_YEAR_INTERVAL -
		year_of_era/CIVIL_CENTURY_YEAR_INTERVAL + day_of_year
	return Calendar_Day_Count(era*CIVIL_ERA_DAY_COUNT + day_of_era -
		CIVIL_UNIX_EPOCH_DAY_OFFSET)
}

// MONOTONIC_MOMENT_MINIMUM: zero. Clock restart at zero on each boot. Reading below zero mean
// clock go backward.
const MONOTONIC_MOMENT_MINIMUM Monotonic_Moment = 0

// MONOTONIC_MOMENT_MAXIMUM: one year of uptime. Many thing stop machine before one year.
// Chaos engineering kill it on purpose. Operating system update reboot it. More uptime than
// this mean defect, not data point.
const MONOTONIC_MOMENT_MAXIMUM Monotonic_Moment = Monotonic_Moment(DAY * 365)

// Monotonic_Moment: clock reading in nanoseconds. Count from machine boot (Linux
// CLOCK_BOOTTIME). Zero mean boot. Reading mean uptime. Bound can hold uptime. Plain Moment
// have no origin. Plain Moment take full signed range.
type Monotonic_Moment int64

// Monotonic_Moment_Invariants state complete uptime domain.
func Monotonic_Moment_Invariants(moment Monotonic_Moment, namespace aver.Namespace) {
	aver.Tree(moment, namespace).
		Range_Int64(
			int64(moment),
			int64(MONOTONIC_MOMENT_MINIMUM),
			int64(MONOTONIC_MOMENT_MAXIMUM)).
		Ensure()
}

// State carries caller-owned backend state. An interface, not an unsafe pointer: a pointer
// boxed in an interface sits in the data word, thus no allocation, and the reader gets a
// checked assertion instead of a blind cast.
type State interface{}

// Monotonic_Reader read monotonic clock over caller-owned state. Never go backward. Use for
// elapsed time, timeout, latency.
type Monotonic_Reader func(state State) (moment Monotonic_Moment)

// Realtime_Reader read wall clock as nanoseconds from Unix epoch over caller-owned state. Can
// jump. Use for calendar timestamp only, never for elapsed time.
type Realtime_Reader func(state State) (moment Moment)

// Clock: injected time source. Backend state stays explicit so the vtable never capture it.
type Clock struct {
	// State stays caller-owned because captured backend state would escape with this vtable.
	State State
	// Now_Monotonic is the monotonic slot of the vtable.
	Now_Monotonic Monotonic_Reader
	// Now_Realtime is the realtime slot of the vtable.
	Now_Realtime Realtime_Reader
}

// Clock_Invariants state both readers bound. Clock is vtable. One property only:
// every slot full. Zero Clock read as Clock, then panic on first use. Backend that
// fill one slot and forget other fail one call later.
func Clock_Invariants(clock Clock, namespace aver.Namespace) {
	aver.Always(
		clock.Now_Monotonic != nil, "A Clock has a monotonic reader.",
	)
	aver.Always(
		clock.Now_Realtime != nil, "A Clock has a realtime reader.",
	)
}

// Clock_Now_Monotonic passes state explicitly because a bound reader would allocate. Uptime
// bound hold here, at the one read every backend go through, thus OS clock past one year fail
// on the read and not at whichever holder happen to assert what it stored.
func Clock_Now_Monotonic(clock Clock) (moment Monotonic_Moment) {
	defer func() { Monotonic_Moment_Invariants(moment, "clock_now_monotonic.moment") }()
	Clock_Invariants(clock, "clock_now_monotonic.clock")
	return clock.Now_Monotonic(clock.State)
}

// Clock_Now_Realtime passes state explicitly because a bound reader would allocate.
func Clock_Now_Realtime(clock Clock) (moment Moment) {
	defer func() { Moment_Invariants(moment, "clock_now_realtime.moment") }()
	Clock_Invariants(clock, "clock_now_realtime.clock")
	return clock.Now_Realtime(clock.State)
}

// Tick_Count: how many time virtual clock advance. It is x in each skew formula.
type Tick_Count int64

// Tick_Count_Invariants state complete tick-count domain.
func Tick_Count_Invariants(ticks Tick_Count, namespace aver.Namespace) {
	aver.Tree(ticks, namespace).
		Range_Int64(int64(ticks), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Skew_Magnitude is coefficient A of one model: drift-per-tick, amplitude, or step size, in
// nanoseconds. Its own type, not Duration, because a Virtual_Clock chain already holds one
// Duration for Resolution and a chain holds each type one time.
type Skew_Magnitude int64

// Skew_Magnitude_Invariants states the Duration domain through the same constants.
func Skew_Magnitude_Invariants(magnitude Skew_Magnitude, namespace aver.Namespace) {
	aver.Tree(magnitude, namespace).
		Range_Int64(int64(magnitude), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Skew_Ticks is coefficient B of one model: linear initial offset, periodic period, or onset
// tick of a step. Its own type, not Tick_Count, for the same chain reason as Skew_Magnitude.
type Skew_Ticks int64

// Skew_Ticks_Invariants states the Tick_Count domain through the same constants.
func Skew_Ticks_Invariants(ticks Skew_Ticks, namespace aver.Namespace) {
	aver.Tree(ticks, namespace).
		Range_Int64(int64(ticks), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Offset keeps coefficients by value so a skew reader needs no separately owned state.
type Offset struct {
	// Kind permits static evaluator dispatch, avoiding captured procedure state.
	Kind Skew_Kind
	// A shares the lifetime of the model that owns it.
	A Skew_Magnitude
	// B shares the lifetime of the model that owns it.
	B Skew_Ticks
}

// Offset_Invariants keeps the model kind to the three evaluators and both coefficients inside
// their domains.
func Offset_Invariants(offset Offset, namespace aver.Namespace) {
	Skew_Kind_Invariants(offset.Kind, namespace)
	Skew_Magnitude_Invariants(offset.A, namespace)
	Skew_Ticks_Invariants(offset.B, namespace)
}

// Virtual_Clock configure deterministic clock that Virtual_Clock_To_Clock build. Time
// advance only when Tick run. Simulation reach future Moment by tick, never by wait.
type Virtual_Clock struct {
	// Resolution: how far monotonic clock advance on each Tick. Grain of simulated
	// oscillator.
	Resolution Duration
	// Epoch: wall-clock origin. Now_Realtime at tick zero, before skew.
	Epoch Moment
	// Skew bend Now_Realtime away from true elapsed time. Zero Skew mean perfect clock.
	Skew Offset
	// Ticks stays caller-owned because both readers and the root advance the same counter.
	Ticks Tick_Count
}

// Virtual_Clock_Invariants states every field, the skew through its own chain.
func Virtual_Clock_Invariants(virtual Virtual_Clock, namespace aver.Namespace) {
	Duration_Invariants(virtual.Resolution, namespace)
	Moment_Invariants(virtual.Epoch, namespace)
	Offset_Invariants(virtual.Skew, namespace)
	Tick_Count_Invariants(virtual.Ticks, namespace)
}

// Virtual_Clock_Pointer names caller-owned Virtual_Clock storage.
type Virtual_Clock_Pointer *Virtual_Clock

// Virtual_Clock_Pointer_Invariants admits absent storage; each holder asserts presence.
func Virtual_Clock_Pointer_Invariants(value Virtual_Clock_Pointer, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Virtual_Clock_Invariants(*value, namespace)
}

// Virtual_Clock_To_Clock binds caller-owned state without a captured function environment.
func Virtual_Clock_To_Clock(virtual Virtual_Clock_Pointer) (clock Clock) {
	defer func() { Clock_Invariants(clock, "virtual_clock_to_clock.clock") }()
	Virtual_Clock_Pointer_Invariants(virtual, "virtual_clock_to_clock.virtual")
	aver.Always(virtual != nil, "A virtual clock has caller-owned state.")
	virtual.Ticks = 0
	return Clock{
		State:         (*Virtual_Clock)(virtual),
		Now_Monotonic: virtual_clock_now_monotonic,
		Now_Realtime:  virtual_clock_now_realtime,
	}
}

// Virtual_Clock_Tick keeps advancement with the root that owns mutable clock state.
func Virtual_Clock_Tick(virtual Virtual_Clock_Pointer) {
	Virtual_Clock_Pointer_Invariants(virtual, "virtual_clock_tick.virtual")
	aver.Always(virtual != nil, "A tick advances caller-owned virtual-clock state.")
	virtual.Ticks++
}

func virtual_clock_now_monotonic(state State) (moment Monotonic_Moment) {
	defer func() { Monotonic_Moment_Invariants(moment, "virtual_clock_now_monotonic.moment") }()
	virtual := state.(*Virtual_Clock)
	return Monotonic_Moment(int64(virtual.Ticks) * int64(virtual.Resolution))
}

func virtual_clock_now_realtime(state State) (moment Moment) {
	defer func() { Moment_Invariants(moment, "virtual_clock_now_realtime.moment") }()
	virtual := state.(*Virtual_Clock)
	now := virtual.Epoch + Moment(int64(virtual.Ticks)*int64(virtual.Resolution))
	return now - Moment(Offset_Read(virtual.Skew, virtual.Ticks))
}

// SKEW_KIND_LINEAR model constant drift. A nanoseconds of skew per tick, plus initial B
// (A*x + B, x is tick count).
const SKEW_KIND_LINEAR Skew_Kind = 0

// SKEW_KIND_PERIODIC model sinusoidal wobble. Amplitude A over period of B ticks
// (A*sin(x*2pi/B)).
const SKEW_KIND_PERIODIC Skew_Kind = 1

// SKEW_KIND_STEP model jump of A after B ticks. NTP correction, or operator change clock.
const SKEW_KIND_STEP Skew_Kind = 2

// Skew_Kind pick which clock-deviation model Skew build.
type Skew_Kind uint8

// Skew_Kind_Invariants hold kind to three models that Skew build. Default arm of that switch
// is linear model. Unlisted kind would drift in silence, not fail.
func Skew_Kind_Invariants(kind Skew_Kind, namespace aver.Namespace) {
	aver.Tree(kind, namespace).
		Enum_3_Uint8(
			uint8(kind),
			uint8(SKEW_KIND_LINEAR),
			uint8(SKEW_KIND_PERIODIC),
			uint8(SKEW_KIND_STEP),
		).
		Ensure()
}

// Skew build Offset for one deviation model. Take kind plus two coefficients. Coefficient a
// is magnitude: drift-per-tick, amplitude, or step size. Coefficient b count ticks: linear
// initial offset, periodic period, or onset tick of step.
func Skew(kind Skew_Kind, a Skew_Magnitude, b Skew_Ticks) (offset Offset) {
	defer func() { Offset_Invariants(offset, "skew.offset") }()
	Skew_Kind_Invariants(kind, "skew.kind")
	Skew_Magnitude_Invariants(a, "skew.a")
	Skew_Ticks_Invariants(b, "skew.b")
	return Offset{Kind: kind, A: a, B: b}
}

// Offset_Read evaluates stored coefficients without closure state.
func Offset_Read(offset Offset, ticks Tick_Count) (skew Duration) {
	defer func() { Duration_Invariants(skew, "offset_read.skew") }()
	Offset_Invariants(offset, "offset_read.offset")
	Tick_Count_Invariants(ticks, "offset_read.ticks")
	switch offset.Kind {
	case SKEW_KIND_PERIODIC:
		// Zero period reports no skew because division would panic.
		if offset.B == 0 {
			return 0
		}
		// Phase reduction prevents scaled numerator overflow during long runs.
		phase := ticks % Tick_Count(offset.B)
		turns := fixedpoint.From_Ratio(
			fixedpoint.Numerator(phase),
			fixedpoint.Denominator(offset.B),
		)
		amplitude := fixedpoint.Number(fixedpoint.From_Integer(
			fixedpoint.Whole_Integer(offset.A),
		))
		wobble := fixedpoint.Multiply(
			fixedpoint.Multiplicand(amplitude),
			fixedpoint.Multiplier(fixedpoint.Sine_Turns(turns)),
		)
		return Duration(fixedpoint.Whole(wobble))
	case SKEW_KIND_STEP:
		if ticks > Tick_Count(offset.B) {
			return Duration(offset.A)
		}
		return 0
	default:
		return Duration(ticks)*Duration(offset.A) + Duration(offset.B)
	}
}
