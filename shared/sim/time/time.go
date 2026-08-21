// Package clock gives time as an injected dependency: a read-only Clock vtable over explicit
// caller-owned state, the moment and duration arithmetic it read in, and the civil calendar.
// Production wires an OS clock (clock/default). A simulation wires a Virtual one, or a view
// over the loop counter in sim/nbio. The code between never knows which one it holds.
// Nothing here advance time; that is the loop's job, and the loop live in sim/nbio.
package time

import (
	"unsafe"

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

// Unix_Second_Split floors negative values because truncation would produce negative day seconds.
func Unix_Second_Split(
	seconds Unix_Second_Count,
) (days Calendar_Day_Count, day_seconds Day_Second_Count) {
	defer func() {
		Calendar_Day_Count_Invariants(days, "unix_second_split.days")
		Day_Second_Count_Invariants(day_seconds, "unix_second_split.day_seconds")
	}()
	Unix_Second_Count_Invariants(seconds, "unix_second_split.seconds")
	days = Calendar_Day_Count(int64(seconds) / SECOND_COUNT_PER_DAY)
	remainder := int64(seconds) % SECOND_COUNT_PER_DAY
	if remainder < 0 {
		days--
		remainder += SECOND_COUNT_PER_DAY
	}
	return days, Day_Second_Count(remainder)
}

// Civil_From_Days uses era arithmetic so leap rules need no ambient timezone or lookup table.
func Civil_From_Days(
	days Calendar_Day_Count,
) (year Civil_Year, month Civil_Month, day Civil_Day) {
	defer func() {
		Civil_Year_Invariants(year, "civil_from_days.year")
		Civil_Month_Invariants(month, "civil_from_days.month")
		Civil_Day_Invariants(day, "civil_from_days.day")
	}()
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
	year = Civil_Year(year_of_era + era*CIVIL_ERA_YEAR_INTERVAL)
	day_of_year := day_of_era -
		(CIVIL_COMMON_YEAR_DAY_COUNT*year_of_era +
			year_of_era/CIVIL_LEAP_YEAR_INTERVAL -
			year_of_era/CIVIL_CENTURY_YEAR_INTERVAL)
	month_prime := (CIVIL_MARCH_MONTH_GROUP_COUNT*day_of_year +
		CIVIL_MARCH_MONTH_GROUP_BIAS) / CIVIL_MARCH_MONTH_GROUP_DAY_COUNT
	day = Civil_Day(day_of_year-(CIVIL_MARCH_MONTH_GROUP_DAY_COUNT*month_prime+
		CIVIL_MARCH_MONTH_GROUP_BIAS)/CIVIL_MARCH_MONTH_GROUP_COUNT) + CIVIL_DAY_MINIMUM
	month = Civil_Month(month_prime + CIVIL_MARCH_EPOCH_MONTH)
	if month > CIVIL_MONTH_MAXIMUM {
		month -= CIVIL_MONTH_MAXIMUM
		year++
	}
	return year, month, day
}

// Days_From_Civil uses same March epoch so it is exact inverse over representable dates.
func Days_From_Civil(
	year Civil_Year, month Civil_Month, day Civil_Day,
) (days Calendar_Day_Count) {
	defer func() { Calendar_Day_Count_Invariants(days, "days_from_civil.days") }()
	Civil_Year_Invariants(year, "days_from_civil.year")
	Civil_Month_Invariants(month, "days_from_civil.month")
	Civil_Day_Invariants(day, "days_from_civil.day")
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

// Monotonic_Moment: clock reading in nanoseconds. Count from machine boot (Linux
// CLOCK_BOOTTIME). Zero mean boot. Reading mean uptime. Bound can hold uptime. Plain Moment
// have no origin. Plain Moment take full signed range.
type Monotonic_Moment int64

// MONOTONIC_MOMENT_MINIMUM: zero. Clock restart at zero on each boot. Reading below zero mean
// clock go backward.
const MONOTONIC_MOMENT_MINIMUM Monotonic_Moment = 0

// MONOTONIC_MOMENT_MAXIMUM: one year of uptime. Many thing stop machine before one year.
// Chaos engineering kill it on purpose. Operating system update reboot it. More uptime than
// this mean defect, not data point.
const MONOTONIC_MOMENT_MAXIMUM Monotonic_Moment = Monotonic_Moment(DAY * 365)

// Monotonic_Moment_Invariants state complete uptime domain.
func Monotonic_Moment_Invariants(moment Monotonic_Moment, namespace aver.Namespace) {
	aver.Tree(moment, namespace).
		Range_Int64(
			int64(moment),
			int64(MONOTONIC_MOMENT_MINIMUM),
			int64(MONOTONIC_MOMENT_MAXIMUM)).
		Ensure()
}

// Clock: injected time source. Backend state stays explicit so the vtable never capture it.
type Clock struct {
	// State stays caller-owned because captured backend state would escape with this vtable.
	State unsafe.Pointer
	// Now_Monotonic read monotonic clock. Never go backward. Use for elapsed time, timeout,
	// latency.
	Now_Monotonic func(state unsafe.Pointer) (moment Monotonic_Moment)
	// Now_Realtime read wall clock as nanoseconds from Unix epoch. Can jump. Use for calendar
	// timestamp only, never for elapsed time.
	Now_Realtime func(state unsafe.Pointer) (moment Moment)
}

// Clock_Now_Monotonic passes state explicitly because a bound reader would allocate. Uptime
// bound hold here, at the one read every backend go through, thus OS clock past one year fail
// on the read and not at whichever holder happen to assert what it stored.
func Clock_Now_Monotonic(clock Clock) (moment Monotonic_Moment) {
	defer func() { Monotonic_Moment_Invariants(moment, "clock_now_monotonic.moment") }()
	return clock.Now_Monotonic(clock.State)
}

// Clock_Now_Realtime passes state explicitly because a bound reader would allocate.
func Clock_Now_Realtime(clock Clock) (moment Moment) {
	return clock.Now_Realtime(clock.State)
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

// Tick_Count: how many time virtual clock advance. It is x in each skew formula.
type Tick_Count int64

// Tick_Count_Invariants state complete tick-count domain.
func Tick_Count_Invariants(ticks Tick_Count, namespace aver.Namespace) {
	aver.Tree(ticks, namespace).
		Range_Int64(int64(ticks), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Offset keeps coefficients by value so a skew reader needs no separately owned state.
type Offset struct {
	// Kind permits static evaluator dispatch, avoiding captured procedure state.
	Kind Skew_Kind
	// A shares the lifetime of the model that owns it.
	A Duration
	// B shares the lifetime of the model that owns it.
	B Tick_Count
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

// Virtual_Clock_Invariants leaves skew coefficients to their own typed invariant chain.
func Virtual_Clock_Invariants(virtual Virtual_Clock, namespace aver.Namespace) {
	Duration_Invariants(virtual.Resolution, namespace)
	Moment_Invariants(virtual.Epoch, namespace)
}

// Virtual_Clock_To_Clock binds caller-owned state without a captured function environment.
func Virtual_Clock_To_Clock(virtual *Virtual_Clock) (clock Clock) {
	aver.Always(virtual != nil, "A virtual clock has caller-owned state.")
	Virtual_Clock_Invariants(*virtual, "virtual_clock_to_clock.virtual")
	virtual.Ticks = 0
	clock = Clock{
		State:         unsafe.Pointer(virtual),
		Now_Monotonic: virtual_clock_now_monotonic,
		Now_Realtime:  virtual_clock_now_realtime,
	}
	Clock_Invariants(clock, "virtual_clock_to_clock.clock")
	return clock
}

// Virtual_Clock_Tick keeps advancement with the root that owns mutable clock state.
func Virtual_Clock_Tick(virtual *Virtual_Clock) {
	aver.Always(virtual != nil, "A tick advances caller-owned virtual-clock state.")
	virtual.Ticks++
}

func virtual_clock_now_monotonic(state unsafe.Pointer) (moment Monotonic_Moment) {
	virtual := (*Virtual_Clock)(state)
	uptime := Monotonic_Moment(int64(virtual.Ticks) * int64(virtual.Resolution))
	Monotonic_Moment_Invariants(uptime, "virtual_clock_to_clock.uptime")
	return uptime
}

func virtual_clock_now_realtime(state unsafe.Pointer) (moment Moment) {
	virtual := (*Virtual_Clock)(state)
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
func Skew(kind Skew_Kind, a Duration, b Tick_Count) (offset Offset) {
	Skew_Kind_Invariants(kind, "skew.kind")
	Duration_Invariants(a, "skew.a")
	Tick_Count_Invariants(b, "skew.b")
	return Offset{Kind: kind, A: a, B: b}
}

// Offset_Read evaluates stored coefficients without closure state.
func Offset_Read(offset Offset, ticks Tick_Count) (skew Duration) {
	switch offset.Kind {
	case SKEW_KIND_PERIODIC:
		// Zero period reports no skew because division would panic.
		if offset.B == 0 {
			return 0
		}
		// Phase reduction prevents scaled numerator overflow during long runs.
		phase := ticks % offset.B
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
		if ticks > offset.B {
			return offset.A
		}
		return 0
	default:
		return Duration(ticks)*offset.A + Duration(offset.B)
	}
}
