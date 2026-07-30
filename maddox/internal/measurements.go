package maddox

import (
	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/fixedpoint"
)

// Wall_Time_Mean is the arithmetic mean of the elapsed-time distribution, fixedpoint-scaled; a
// distinct type gives the bare invariant its own per-metric root.
type Wall_Time_Mean int64

// Wall_Time_Mean_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Wall_Time_Mean_Invariants(identifier string, value Wall_Time_Mean) {
	invariant.Always(int64(value) >= 0, "A wall time mean is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Wall_Time_Mean) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Wall_Time_Mean) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Wall_Time_Standard_Deviation is the sample standard deviation of the elapsed-time distribution,
// fixedpoint-scaled; a distinct type gives the bare invariant its own per-metric root.
type Wall_Time_Standard_Deviation int64

// Wall_Time_Standard_Deviation_Invariants pins non-negativity, the honest bound for a statistic
// over a non-negative metric; a coverage witness here would be unfunded debt.
func Wall_Time_Standard_Deviation_Invariants(
	identifier string, value Wall_Time_Standard_Deviation,
) {
	invariant.Always(int64(value) >= 0, "A wall time standard deviation is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Wall_Time_Standard_Deviation) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Wall_Time_Standard_Deviation) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Wall_Time_Min is the smallest observed value of the elapsed-time distribution, fixedpoint-
// scaled; a distinct type gives the bare invariant its own per-metric root.
type Wall_Time_Min int64

// Wall_Time_Min_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Wall_Time_Min_Invariants(identifier string, value Wall_Time_Min) {
	invariant.Always(int64(value) >= 0, "A wall time min is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Wall_Time_Min) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Wall_Time_Min) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Wall_Time_Max is the largest observed value of the elapsed-time distribution, fixedpoint-scaled;
// a distinct type gives the bare invariant its own per-metric root.
type Wall_Time_Max int64

// Wall_Time_Max_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Wall_Time_Max_Invariants(identifier string, value Wall_Time_Max) {
	invariant.Always(int64(value) >= 0, "A wall time max is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Wall_Time_Max) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Wall_Time_Max) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Wall_Time_Median is the middle value of the elapsed-time distribution, fixedpoint-scaled; a
// distinct type gives the bare invariant its own per-metric root.
type Wall_Time_Median int64

// Wall_Time_Median_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Wall_Time_Median_Invariants(identifier string, value Wall_Time_Median) {
	invariant.Always(int64(value) >= 0, "A wall time median is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Wall_Time_Median) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Wall_Time_Median) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Wall_Time_Q1 is the first quartile of the elapsed-time distribution, fixedpoint-scaled; a
// distinct type gives the bare invariant its own per-metric root.
type Wall_Time_Q1 int64

// Wall_Time_Q1_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Wall_Time_Q1_Invariants(identifier string, value Wall_Time_Q1) {
	invariant.Always(int64(value) >= 0, "A wall time q1 is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Wall_Time_Q1) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Wall_Time_Q1) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Wall_Time_Q3 is the third quartile of the elapsed-time distribution, fixedpoint-scaled; a
// distinct type gives the bare invariant its own per-metric root.
type Wall_Time_Q3 int64

// Wall_Time_Q3_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Wall_Time_Q3_Invariants(identifier string, value Wall_Time_Q3) {
	invariant.Always(int64(value) >= 0, "A wall time q3 is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Wall_Time_Q3) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Wall_Time_Q3) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Wall_Time_Outliers counts the elapsed-time values beyond Tukey's fences; a distinct type gives
// the stray bounds their own per-metric root.
type Wall_Time_Outliers int

// Wall_Time_Outliers_Invariants bounds the tally to the stray range; the old framework funded no
// witnesses at this position, so none are planted.
func Wall_Time_Outliers_Invariants(identifier string, value Wall_Time_Outliers) {
	invariant.Always(
		int(value) >= STRAYS_MIN, "A wall time outliers tally never falls below the stray floor.")
	invariant.Always(
		int(value) <= STRAYS_MAX, "A wall time outliers tally never exceeds the stray ceiling.")
}

// Wall_Time_Count counts the kept elapsed-time runs; a distinct type gives the quorum bounds their
// own per-metric root.
type Wall_Time_Count int

// Wall_Time_Count_Invariants bounds the tally to the kept quorum range; the old framework funded
// no witnesses at this position, so none are planted.
func Wall_Time_Count_Invariants(identifier string, value Wall_Time_Count) {
	invariant.Always(int(value) >= KEPT_MIN, "A wall time count never falls below the kept quorum.")
	invariant.Always(int(value) <= KEPT_MAX, "A wall time count never exceeds the kept ceiling.")
}

// Wall_Time_Unit names the elapsed-time dimension; the unit is constant per metric, so one exact
// width is pinnable.
type Wall_Time_Unit string

// Wall_Time_Unit_Invariants pins the single nanoseconds width; a both-polarity width witness would
// be unsatisfiable for a constant unit.
func Wall_Time_Unit_Invariants(identifier string, value Wall_Time_Unit) {
	invariant.Always(
		len(value) == UNIT_BYTES_MAX, "A wall time unit always spans the eleven-byte nanoseconds width.")
}

// Wall_Time_Measurement is the elapsed-time distribution, each leaf typed per metric so the bare
// invariants seed under their own roots.
type Wall_Time_Measurement struct {
	// Mean is the arithmetic mean of the metric's values.
	Mean Wall_Time_Mean `json:"mean"`
	// Standard_Deviation is the sample standard deviation, with an n-1 denominator.
	Standard_Deviation Wall_Time_Standard_Deviation `json:"stddev"`
	// Min is the smallest value observed.
	Min Wall_Time_Min `json:"min"`
	// Max is the largest value observed.
	Max Wall_Time_Max `json:"max"`
	// Median is the middle value of the sorted values.
	Median Wall_Time_Median `json:"median"`
	// Q1 is the first quartile by poop's index math.
	Q1 Wall_Time_Q1 `json:"q1"`
	// Q3 is the third quartile by poop's index math.
	Q3 Wall_Time_Q3 `json:"q3"`
	// Outlier_Count is how many values fall beyond Tukey's fences.
	Outlier_Count Wall_Time_Outliers `json:"outliers"`
	// Sample_Count is how many values the distribution was computed from.
	Sample_Count Wall_Time_Count `json:"count"`
	// Unit names the unit the raw values are in.
	Unit Wall_Time_Unit `json:"unit"`
}

// Wall_Time_Measurement_Invariants forwards every field to its leaf helper so each bound is stated
// at the type that owns it.
func Wall_Time_Measurement_Invariants(identifier string, value Wall_Time_Measurement) {
	Wall_Time_Mean_Invariants(identifier, value.Mean)
	Wall_Time_Standard_Deviation_Invariants(identifier, value.Standard_Deviation)
	Wall_Time_Min_Invariants(identifier, value.Min)
	Wall_Time_Max_Invariants(identifier, value.Max)
	Wall_Time_Median_Invariants(identifier, value.Median)
	Wall_Time_Q1_Invariants(identifier, value.Q1)
	Wall_Time_Q3_Invariants(identifier, value.Q3)
	Wall_Time_Outliers_Invariants(identifier, value.Outlier_Count)
	Wall_Time_Count_Invariants(identifier, value.Sample_Count)
	Wall_Time_Unit_Invariants(identifier, value.Unit)
}

// As_Measurement converts field-by-field to the plain Measurement; a method rather than a free
// function so the conversion plants no invariant roots.
func (value Wall_Time_Measurement) As_Measurement() (measurement Measurement) {
	return Measurement{
		Mean:               Mean(value.Mean),
		Standard_Deviation: Standard_Deviation(value.Standard_Deviation),
		Min:                Min(value.Min),
		Max:                Max(value.Max),
		Median:             Median(value.Median),
		Q1:                 Q1(value.Q1),
		Q3:                 Q3(value.Q3),
		Outlier_Count:      Strays(value.Outlier_Count),
		Sample_Count:       Kept(value.Sample_Count),
		Unit:               Unit(value.Unit),
	}
}

// Peak_RSS_Mean is the arithmetic mean of the peak-memory distribution, fixedpoint-scaled; a
// distinct type gives the bare invariant its own per-metric root.
type Peak_RSS_Mean int64

// Peak_RSS_Mean_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Peak_RSS_Mean_Invariants(identifier string, value Peak_RSS_Mean) {
	invariant.Always(int64(value) >= 0, "A peak rss mean is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Peak_RSS_Mean) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Peak_RSS_Mean) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Peak_RSS_Standard_Deviation is the sample standard deviation of the peak-memory distribution,
// fixedpoint-scaled; a distinct type gives the bare invariant its own per-metric root.
type Peak_RSS_Standard_Deviation int64

// Peak_RSS_Standard_Deviation_Invariants pins non-negativity, the honest bound for a statistic
// over a non-negative metric; a coverage witness here would be unfunded debt.
func Peak_RSS_Standard_Deviation_Invariants(identifier string, value Peak_RSS_Standard_Deviation) {
	invariant.Always(int64(value) >= 0, "A peak rss standard deviation is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Peak_RSS_Standard_Deviation) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Peak_RSS_Standard_Deviation) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Peak_RSS_Min is the smallest observed value of the peak-memory distribution, fixedpoint-scaled;
// a distinct type gives the bare invariant its own per-metric root.
type Peak_RSS_Min int64

// Peak_RSS_Min_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Peak_RSS_Min_Invariants(identifier string, value Peak_RSS_Min) {
	invariant.Always(int64(value) >= 0, "A peak rss min is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Peak_RSS_Min) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Peak_RSS_Min) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Peak_RSS_Max is the largest observed value of the peak-memory distribution, fixedpoint-scaled; a
// distinct type gives the bare invariant its own per-metric root.
type Peak_RSS_Max int64

// Peak_RSS_Max_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Peak_RSS_Max_Invariants(identifier string, value Peak_RSS_Max) {
	invariant.Always(int64(value) >= 0, "A peak rss max is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Peak_RSS_Max) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Peak_RSS_Max) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Peak_RSS_Median is the middle value of the peak-memory distribution, fixedpoint-scaled; a
// distinct type gives the bare invariant its own per-metric root.
type Peak_RSS_Median int64

// Peak_RSS_Median_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Peak_RSS_Median_Invariants(identifier string, value Peak_RSS_Median) {
	invariant.Always(int64(value) >= 0, "A peak rss median is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Peak_RSS_Median) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Peak_RSS_Median) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Peak_RSS_Q1 is the first quartile of the peak-memory distribution, fixedpoint-scaled; a distinct
// type gives the bare invariant its own per-metric root.
type Peak_RSS_Q1 int64

// Peak_RSS_Q1_Invariants pins non-negativity, the honest bound for a statistic over a non-negative
// metric; a coverage witness here would be unfunded debt.
func Peak_RSS_Q1_Invariants(identifier string, value Peak_RSS_Q1) {
	invariant.Always(int64(value) >= 0, "A peak rss q1 is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Peak_RSS_Q1) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Peak_RSS_Q1) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Peak_RSS_Q3 is the third quartile of the peak-memory distribution, fixedpoint-scaled; a distinct
// type gives the bare invariant its own per-metric root.
type Peak_RSS_Q3 int64

// Peak_RSS_Q3_Invariants pins non-negativity, the honest bound for a statistic over a non-negative
// metric; a coverage witness here would be unfunded debt.
func Peak_RSS_Q3_Invariants(identifier string, value Peak_RSS_Q3) {
	invariant.Always(int64(value) >= 0, "A peak rss q3 is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Peak_RSS_Q3) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Peak_RSS_Q3) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Peak_RSS_Outliers counts the peak-memory values beyond Tukey's fences; a distinct type gives the
// stray bounds their own per-metric root.
type Peak_RSS_Outliers int

// Peak_RSS_Outliers_Invariants bounds the tally to the stray range; the old framework funded no
// witnesses at this position, so none are planted.
func Peak_RSS_Outliers_Invariants(identifier string, value Peak_RSS_Outliers) {
	invariant.Always(
		int(value) >= STRAYS_MIN, "A peak rss outliers tally never falls below the stray floor.")
	invariant.Always(
		int(value) <= STRAYS_MAX, "A peak rss outliers tally never exceeds the stray ceiling.")
}

// Peak_RSS_Count counts the kept peak-memory runs; a distinct type gives the quorum bounds their
// own per-metric root.
type Peak_RSS_Count int

// Peak_RSS_Count_Invariants bounds the tally to the kept quorum range; the old framework funded no
// witnesses at this position, so none are planted.
func Peak_RSS_Count_Invariants(identifier string, value Peak_RSS_Count) {
	invariant.Always(int(value) >= KEPT_MIN, "A peak rss count never falls below the kept quorum.")
	invariant.Always(int(value) <= KEPT_MAX, "A peak rss count never exceeds the kept ceiling.")
}

// Peak_RSS_Unit names the peak-memory dimension; the unit is constant per metric, so one exact
// width is pinnable.
type Peak_RSS_Unit string

// Peak_RSS_Unit_Invariants pins the single bytes width; a both-polarity width witness would be
// unsatisfiable for a constant unit.
func Peak_RSS_Unit_Invariants(identifier string, value Peak_RSS_Unit) {
	invariant.Always(
		len(value) == UNIT_BYTES_MIN, "A peak rss unit always spans the five-byte bytes width.")
}

// Peak_RSS_Measurement is the peak-memory distribution, each leaf typed per metric so the bare
// invariants seed under their own roots.
type Peak_RSS_Measurement struct {
	// Mean is the arithmetic mean of the metric's values.
	Mean Peak_RSS_Mean `json:"mean"`
	// Standard_Deviation is the sample standard deviation, with an n-1 denominator.
	Standard_Deviation Peak_RSS_Standard_Deviation `json:"stddev"`
	// Min is the smallest value observed.
	Min Peak_RSS_Min `json:"min"`
	// Max is the largest value observed.
	Max Peak_RSS_Max `json:"max"`
	// Median is the middle value of the sorted values.
	Median Peak_RSS_Median `json:"median"`
	// Q1 is the first quartile by poop's index math.
	Q1 Peak_RSS_Q1 `json:"q1"`
	// Q3 is the third quartile by poop's index math.
	Q3 Peak_RSS_Q3 `json:"q3"`
	// Outlier_Count is how many values fall beyond Tukey's fences.
	Outlier_Count Peak_RSS_Outliers `json:"outliers"`
	// Sample_Count is how many values the distribution was computed from.
	Sample_Count Peak_RSS_Count `json:"count"`
	// Unit names the unit the raw values are in.
	Unit Peak_RSS_Unit `json:"unit"`
}

// Peak_RSS_Measurement_Invariants forwards every field to its leaf helper so each bound is stated
// at the type that owns it.
func Peak_RSS_Measurement_Invariants(identifier string, value Peak_RSS_Measurement) {
	Peak_RSS_Mean_Invariants(identifier, value.Mean)
	Peak_RSS_Standard_Deviation_Invariants(identifier, value.Standard_Deviation)
	Peak_RSS_Min_Invariants(identifier, value.Min)
	Peak_RSS_Max_Invariants(identifier, value.Max)
	Peak_RSS_Median_Invariants(identifier, value.Median)
	Peak_RSS_Q1_Invariants(identifier, value.Q1)
	Peak_RSS_Q3_Invariants(identifier, value.Q3)
	Peak_RSS_Outliers_Invariants(identifier, value.Outlier_Count)
	Peak_RSS_Count_Invariants(identifier, value.Sample_Count)
	Peak_RSS_Unit_Invariants(identifier, value.Unit)
}

// As_Measurement converts field-by-field to the plain Measurement; a method rather than a free
// function so the conversion plants no invariant roots.
func (value Peak_RSS_Measurement) As_Measurement() (measurement Measurement) {
	return Measurement{
		Mean:               Mean(value.Mean),
		Standard_Deviation: Standard_Deviation(value.Standard_Deviation),
		Min:                Min(value.Min),
		Max:                Max(value.Max),
		Median:             Median(value.Median),
		Q1:                 Q1(value.Q1),
		Q3:                 Q3(value.Q3),
		Outlier_Count:      Strays(value.Outlier_Count),
		Sample_Count:       Kept(value.Sample_Count),
		Unit:               Unit(value.Unit),
	}
}

// CPU_Cycles_Mean is the arithmetic mean of the CPU-cycle distribution, fixedpoint-scaled; a
// distinct type gives the bare invariant its own per-metric root.
type CPU_Cycles_Mean int64

// CPU_Cycles_Mean_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func CPU_Cycles_Mean_Invariants(identifier string, value CPU_Cycles_Mean) {
	invariant.Always(int64(value) >= 0, "A cpu cycles mean is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value CPU_Cycles_Mean) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *CPU_Cycles_Mean) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_Cycles_Standard_Deviation is the sample standard deviation of the CPU-cycle distribution,
// fixedpoint-scaled; a distinct type gives the bare invariant its own per-metric root.
type CPU_Cycles_Standard_Deviation int64

// CPU_Cycles_Standard_Deviation_Invariants pins non-negativity, the honest bound for a statistic
// over a non-negative metric; a coverage witness here would be unfunded debt.
func CPU_Cycles_Standard_Deviation_Invariants(
	identifier string, value CPU_Cycles_Standard_Deviation,
) {
	invariant.Always(int64(value) >= 0, "A cpu cycles standard deviation is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value CPU_Cycles_Standard_Deviation) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *CPU_Cycles_Standard_Deviation) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_Cycles_Min is the smallest observed value of the CPU-cycle distribution, fixedpoint-scaled;
// a distinct type gives the bare invariant its own per-metric root.
type CPU_Cycles_Min int64

// CPU_Cycles_Min_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func CPU_Cycles_Min_Invariants(identifier string, value CPU_Cycles_Min) {
	invariant.Always(int64(value) >= 0, "A cpu cycles min is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value CPU_Cycles_Min) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *CPU_Cycles_Min) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_Cycles_Max is the largest observed value of the CPU-cycle distribution, fixedpoint-scaled; a
// distinct type gives the bare invariant its own per-metric root.
type CPU_Cycles_Max int64

// CPU_Cycles_Max_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func CPU_Cycles_Max_Invariants(identifier string, value CPU_Cycles_Max) {
	invariant.Always(int64(value) >= 0, "A cpu cycles max is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value CPU_Cycles_Max) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *CPU_Cycles_Max) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_Cycles_Median is the middle value of the CPU-cycle distribution, fixedpoint-scaled; a
// distinct type gives the bare invariant its own per-metric root.
type CPU_Cycles_Median int64

// CPU_Cycles_Median_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func CPU_Cycles_Median_Invariants(identifier string, value CPU_Cycles_Median) {
	invariant.Always(int64(value) >= 0, "A cpu cycles median is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value CPU_Cycles_Median) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *CPU_Cycles_Median) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_Cycles_Q1 is the first quartile of the CPU-cycle distribution, fixedpoint-scaled; a distinct
// type gives the bare invariant its own per-metric root.
type CPU_Cycles_Q1 int64

// CPU_Cycles_Q1_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func CPU_Cycles_Q1_Invariants(identifier string, value CPU_Cycles_Q1) {
	invariant.Always(int64(value) >= 0, "A cpu cycles q1 is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value CPU_Cycles_Q1) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *CPU_Cycles_Q1) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_Cycles_Q3 is the third quartile of the CPU-cycle distribution, fixedpoint-scaled; a distinct
// type gives the bare invariant its own per-metric root.
type CPU_Cycles_Q3 int64

// CPU_Cycles_Q3_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func CPU_Cycles_Q3_Invariants(identifier string, value CPU_Cycles_Q3) {
	invariant.Always(int64(value) >= 0, "A cpu cycles q3 is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value CPU_Cycles_Q3) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *CPU_Cycles_Q3) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_Cycles_Outliers counts the CPU-cycle values beyond Tukey's fences; a distinct type gives the
// stray bounds their own per-metric root.
type CPU_Cycles_Outliers int

// CPU_Cycles_Outliers_Invariants bounds the tally to the stray range; the old framework funded no
// witnesses at this position, so none are planted.
func CPU_Cycles_Outliers_Invariants(identifier string, value CPU_Cycles_Outliers) {
	invariant.Always(
		int(value) >= STRAYS_MIN, "A cpu cycles outliers tally never falls below the stray floor.")
	invariant.Always(
		int(value) <= STRAYS_MAX, "A cpu cycles outliers tally never exceeds the stray ceiling.")
}

// CPU_Cycles_Count counts the kept CPU-cycle runs; a distinct type gives the quorum bounds their
// own per-metric root.
type CPU_Cycles_Count int

// CPU_Cycles_Count_Invariants bounds the tally to the kept quorum range; the old framework funded
// no witnesses at this position, so none are planted.
func CPU_Cycles_Count_Invariants(identifier string, value CPU_Cycles_Count) {
	invariant.Always(int(value) >= KEPT_MIN, "A cpu cycles count never falls below the kept quorum.")
	invariant.Always(int(value) <= KEPT_MAX, "A cpu cycles count never exceeds the kept ceiling.")
}

// CPU_Cycles_Unit names the CPU-cycle dimension; the unit is constant per metric, so one exact
// width is pinnable.
type CPU_Cycles_Unit string

// CPU_Cycles_Unit_Invariants pins the single count width; a both-polarity width witness would be
// unsatisfiable for a constant unit.
func CPU_Cycles_Unit_Invariants(identifier string, value CPU_Cycles_Unit) {
	invariant.Always(
		len(value) == UNIT_BYTES_MIN, "A cpu cycles unit always spans the five-byte count width.")
}

// CPU_Cycles_Measurement is the CPU-cycle distribution, each leaf typed per metric so the bare
// invariants seed under their own roots.
type CPU_Cycles_Measurement struct {
	// Mean is the arithmetic mean of the metric's values.
	Mean CPU_Cycles_Mean `json:"mean"`
	// Standard_Deviation is the sample standard deviation, with an n-1 denominator.
	Standard_Deviation CPU_Cycles_Standard_Deviation `json:"stddev"`
	// Min is the smallest value observed.
	Min CPU_Cycles_Min `json:"min"`
	// Max is the largest value observed.
	Max CPU_Cycles_Max `json:"max"`
	// Median is the middle value of the sorted values.
	Median CPU_Cycles_Median `json:"median"`
	// Q1 is the first quartile by poop's index math.
	Q1 CPU_Cycles_Q1 `json:"q1"`
	// Q3 is the third quartile by poop's index math.
	Q3 CPU_Cycles_Q3 `json:"q3"`
	// Outlier_Count is how many values fall beyond Tukey's fences.
	Outlier_Count CPU_Cycles_Outliers `json:"outliers"`
	// Sample_Count is how many values the distribution was computed from.
	Sample_Count CPU_Cycles_Count `json:"count"`
	// Unit names the unit the raw values are in.
	Unit CPU_Cycles_Unit `json:"unit"`
}

// CPU_Cycles_Measurement_Invariants forwards every field to its leaf helper so each bound is
// stated at the type that owns it.
func CPU_Cycles_Measurement_Invariants(identifier string, value CPU_Cycles_Measurement) {
	CPU_Cycles_Mean_Invariants(identifier, value.Mean)
	CPU_Cycles_Standard_Deviation_Invariants(identifier, value.Standard_Deviation)
	CPU_Cycles_Min_Invariants(identifier, value.Min)
	CPU_Cycles_Max_Invariants(identifier, value.Max)
	CPU_Cycles_Median_Invariants(identifier, value.Median)
	CPU_Cycles_Q1_Invariants(identifier, value.Q1)
	CPU_Cycles_Q3_Invariants(identifier, value.Q3)
	CPU_Cycles_Outliers_Invariants(identifier, value.Outlier_Count)
	CPU_Cycles_Count_Invariants(identifier, value.Sample_Count)
	CPU_Cycles_Unit_Invariants(identifier, value.Unit)
}

// As_Measurement converts field-by-field to the plain Measurement; a method rather than a free
// function so the conversion plants no invariant roots.
func (value CPU_Cycles_Measurement) As_Measurement() (measurement Measurement) {
	return Measurement{
		Mean:               Mean(value.Mean),
		Standard_Deviation: Standard_Deviation(value.Standard_Deviation),
		Min:                Min(value.Min),
		Max:                Max(value.Max),
		Median:             Median(value.Median),
		Q1:                 Q1(value.Q1),
		Q3:                 Q3(value.Q3),
		Outlier_Count:      Strays(value.Outlier_Count),
		Sample_Count:       Kept(value.Sample_Count),
		Unit:               Unit(value.Unit),
	}
}

// Instructions_Mean is the arithmetic mean of the retired-instruction distribution, fixedpoint-
// scaled; a distinct type gives the bare invariant its own per-metric root.
type Instructions_Mean int64

// Instructions_Mean_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Instructions_Mean_Invariants(identifier string, value Instructions_Mean) {
	invariant.Always(int64(value) >= 0, "A instructions mean is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Instructions_Mean) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Instructions_Mean) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Instructions_Standard_Deviation is the sample standard deviation of the retired-instruction
// distribution, fixedpoint-scaled; a distinct type gives the bare invariant its own per-metric
// root.
type Instructions_Standard_Deviation int64

// Instructions_Standard_Deviation_Invariants pins non-negativity, the honest bound for a statistic
// over a non-negative metric; a coverage witness here would be unfunded debt.
func Instructions_Standard_Deviation_Invariants(
	identifier string, value Instructions_Standard_Deviation,
) {
	invariant.Always(int64(value) >= 0, "A instructions standard deviation is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Instructions_Standard_Deviation) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Instructions_Standard_Deviation) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Instructions_Min is the smallest observed value of the retired-instruction distribution,
// fixedpoint-scaled; a distinct type gives the bare invariant its own per-metric root.
type Instructions_Min int64

// Instructions_Min_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Instructions_Min_Invariants(identifier string, value Instructions_Min) {
	invariant.Always(int64(value) >= 0, "A instructions min is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Instructions_Min) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Instructions_Min) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Instructions_Max is the largest observed value of the retired-instruction distribution,
// fixedpoint-scaled; a distinct type gives the bare invariant its own per-metric root.
type Instructions_Max int64

// Instructions_Max_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Instructions_Max_Invariants(identifier string, value Instructions_Max) {
	invariant.Always(int64(value) >= 0, "A instructions max is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Instructions_Max) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Instructions_Max) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Instructions_Median is the middle value of the retired-instruction distribution, fixedpoint-
// scaled; a distinct type gives the bare invariant its own per-metric root.
type Instructions_Median int64

// Instructions_Median_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Instructions_Median_Invariants(identifier string, value Instructions_Median) {
	invariant.Always(int64(value) >= 0, "A instructions median is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Instructions_Median) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Instructions_Median) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Instructions_Q1 is the first quartile of the retired-instruction distribution, fixedpoint-
// scaled; a distinct type gives the bare invariant its own per-metric root.
type Instructions_Q1 int64

// Instructions_Q1_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Instructions_Q1_Invariants(identifier string, value Instructions_Q1) {
	invariant.Always(int64(value) >= 0, "A instructions q1 is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Instructions_Q1) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Instructions_Q1) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Instructions_Q3 is the third quartile of the retired-instruction distribution, fixedpoint-
// scaled; a distinct type gives the bare invariant its own per-metric root.
type Instructions_Q3 int64

// Instructions_Q3_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Instructions_Q3_Invariants(identifier string, value Instructions_Q3) {
	invariant.Always(int64(value) >= 0, "A instructions q3 is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Instructions_Q3) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Instructions_Q3) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Instructions_Outliers counts the retired-instruction values beyond Tukey's fences; a distinct
// type gives the stray bounds their own per-metric root.
type Instructions_Outliers int

// Instructions_Outliers_Invariants bounds the tally to the stray range; the old framework funded
// no witnesses at this position, so none are planted.
func Instructions_Outliers_Invariants(identifier string, value Instructions_Outliers) {
	invariant.Always(
		int(value) >= STRAYS_MIN, "A instructions outliers tally never falls below the stray floor.")
	invariant.Always(
		int(value) <= STRAYS_MAX, "A instructions outliers tally never exceeds the stray ceiling.")
}

// Instructions_Count counts the kept retired-instruction runs; a distinct type gives the quorum
// bounds their own per-metric root.
type Instructions_Count int

// Instructions_Count_Invariants bounds the tally to the kept quorum range; the old framework
// funded no witnesses at this position, so none are planted.
func Instructions_Count_Invariants(identifier string, value Instructions_Count) {
	invariant.Always(int(value) >= KEPT_MIN, "A instructions count never falls below the kept quorum.")
	invariant.Always(int(value) <= KEPT_MAX, "A instructions count never exceeds the kept ceiling.")
}

// Instructions_Unit names the retired-instruction dimension; the unit is constant per metric, so
// one exact width is pinnable.
type Instructions_Unit string

// Instructions_Unit_Invariants pins the single count width; a both-polarity width witness would be
// unsatisfiable for a constant unit.
func Instructions_Unit_Invariants(identifier string, value Instructions_Unit) {
	invariant.Always(
		len(value) == UNIT_BYTES_MIN, "A instructions unit always spans the five-byte count width.")
}

// Instructions_Measurement is the retired-instruction distribution, each leaf typed per metric so
// the bare invariants seed under their own roots.
type Instructions_Measurement struct {
	// Mean is the arithmetic mean of the metric's values.
	Mean Instructions_Mean `json:"mean"`
	// Standard_Deviation is the sample standard deviation, with an n-1 denominator.
	Standard_Deviation Instructions_Standard_Deviation `json:"stddev"`
	// Min is the smallest value observed.
	Min Instructions_Min `json:"min"`
	// Max is the largest value observed.
	Max Instructions_Max `json:"max"`
	// Median is the middle value of the sorted values.
	Median Instructions_Median `json:"median"`
	// Q1 is the first quartile by poop's index math.
	Q1 Instructions_Q1 `json:"q1"`
	// Q3 is the third quartile by poop's index math.
	Q3 Instructions_Q3 `json:"q3"`
	// Outlier_Count is how many values fall beyond Tukey's fences.
	Outlier_Count Instructions_Outliers `json:"outliers"`
	// Sample_Count is how many values the distribution was computed from.
	Sample_Count Instructions_Count `json:"count"`
	// Unit names the unit the raw values are in.
	Unit Instructions_Unit `json:"unit"`
}

// Instructions_Measurement_Invariants forwards every field to its leaf helper so each bound is
// stated at the type that owns it.
func Instructions_Measurement_Invariants(identifier string, value Instructions_Measurement) {
	Instructions_Mean_Invariants(identifier, value.Mean)
	Instructions_Standard_Deviation_Invariants(identifier, value.Standard_Deviation)
	Instructions_Min_Invariants(identifier, value.Min)
	Instructions_Max_Invariants(identifier, value.Max)
	Instructions_Median_Invariants(identifier, value.Median)
	Instructions_Q1_Invariants(identifier, value.Q1)
	Instructions_Q3_Invariants(identifier, value.Q3)
	Instructions_Outliers_Invariants(identifier, value.Outlier_Count)
	Instructions_Count_Invariants(identifier, value.Sample_Count)
	Instructions_Unit_Invariants(identifier, value.Unit)
}

// As_Measurement converts field-by-field to the plain Measurement; a method rather than a free
// function so the conversion plants no invariant roots.
func (value Instructions_Measurement) As_Measurement() (measurement Measurement) {
	return Measurement{
		Mean:               Mean(value.Mean),
		Standard_Deviation: Standard_Deviation(value.Standard_Deviation),
		Min:                Min(value.Min),
		Max:                Max(value.Max),
		Median:             Median(value.Median),
		Q1:                 Q1(value.Q1),
		Q3:                 Q3(value.Q3),
		Outlier_Count:      Strays(value.Outlier_Count),
		Sample_Count:       Kept(value.Sample_Count),
		Unit:               Unit(value.Unit),
	}
}

// Cache_References_Mean is the arithmetic mean of the cache-reference (Linux only) distribution,
// fixedpoint-scaled; a distinct type gives the bare invariant its own per-metric root.
type Cache_References_Mean int64

// Cache_References_Mean_Invariants pins non-negativity, the honest bound for a statistic over a
// non-negative metric; a coverage witness here would be unfunded debt.
func Cache_References_Mean_Invariants(identifier string, value Cache_References_Mean) {
	invariant.Always(int64(value) >= 0, "A cache references mean is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Cache_References_Mean) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Cache_References_Mean) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Cache_References_Standard_Deviation is the sample standard deviation of the cache-reference
// (Linux only) distribution, fixedpoint-scaled; a distinct type gives the bare invariant its own
// per-metric root.
type Cache_References_Standard_Deviation int64

// Cache_References_Standard_Deviation_Invariants pins non-negativity, the honest bound for a
// statistic over a non-negative metric; a coverage witness here would be unfunded debt.
func Cache_References_Standard_Deviation_Invariants(
	identifier string, value Cache_References_Standard_Deviation,
) {
	invariant.Always(int64(value) >= 0, "A cache references standard deviation is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Cache_References_Standard_Deviation) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Cache_References_Standard_Deviation) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Cache_References_Min is the smallest observed value of the cache-reference (Linux only)
// distribution, fixedpoint-scaled; a distinct type gives the bare invariant its own per-metric
// root.
type Cache_References_Min int64

// Cache_References_Min_Invariants pins non-negativity, the honest bound for a statistic over a
// non-negative metric; a coverage witness here would be unfunded debt.
func Cache_References_Min_Invariants(identifier string, value Cache_References_Min) {
	invariant.Always(int64(value) >= 0, "A cache references min is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Cache_References_Min) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Cache_References_Min) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Cache_References_Max is the largest observed value of the cache-reference (Linux only)
// distribution, fixedpoint-scaled; a distinct type gives the bare invariant its own per-metric
// root.
type Cache_References_Max int64

// Cache_References_Max_Invariants pins non-negativity, the honest bound for a statistic over a
// non-negative metric; a coverage witness here would be unfunded debt.
func Cache_References_Max_Invariants(identifier string, value Cache_References_Max) {
	invariant.Always(int64(value) >= 0, "A cache references max is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Cache_References_Max) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Cache_References_Max) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Cache_References_Median is the middle value of the cache-reference (Linux only) distribution,
// fixedpoint-scaled; a distinct type gives the bare invariant its own per-metric root.
type Cache_References_Median int64

// Cache_References_Median_Invariants pins non-negativity, the honest bound for a statistic over a
// non-negative metric; a coverage witness here would be unfunded debt.
func Cache_References_Median_Invariants(identifier string, value Cache_References_Median) {
	invariant.Always(int64(value) >= 0, "A cache references median is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Cache_References_Median) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Cache_References_Median) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Cache_References_Q1 is the first quartile of the cache-reference (Linux only) distribution,
// fixedpoint-scaled; a distinct type gives the bare invariant its own per-metric root.
type Cache_References_Q1 int64

// Cache_References_Q1_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Cache_References_Q1_Invariants(identifier string, value Cache_References_Q1) {
	invariant.Always(int64(value) >= 0, "A cache references q1 is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Cache_References_Q1) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Cache_References_Q1) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Cache_References_Q3 is the third quartile of the cache-reference (Linux only) distribution,
// fixedpoint-scaled; a distinct type gives the bare invariant its own per-metric root.
type Cache_References_Q3 int64

// Cache_References_Q3_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Cache_References_Q3_Invariants(identifier string, value Cache_References_Q3) {
	invariant.Always(int64(value) >= 0, "A cache references q3 is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Cache_References_Q3) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Cache_References_Q3) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Cache_References_Outliers counts the cache-reference (Linux only) values beyond Tukey's fences;
// a distinct type gives the stray bounds their own per-metric root.
type Cache_References_Outliers int

// Cache_References_Outliers_Invariants bounds the tally to the stray range; the old framework
// funded no witnesses at this position, so none are planted.
func Cache_References_Outliers_Invariants(identifier string, value Cache_References_Outliers) {
	invariant.Always(
		int(value) >= STRAYS_MIN, "A cache references outliers tally never falls below the stray floor.")
	invariant.Always(
		int(value) <= STRAYS_MAX, "A cache references outliers tally never exceeds the stray ceiling.")
}

// Cache_References_Count counts the kept cache-reference (Linux only) runs; a distinct type gives
// the quorum bounds their own per-metric root.
type Cache_References_Count int

// Cache_References_Count_Invariants bounds the tally to the kept quorum range; the old framework
// funded no witnesses at this position, so none are planted.
func Cache_References_Count_Invariants(identifier string, value Cache_References_Count) {
	invariant.Always(
		int(value) >= KEPT_MIN, "A cache references count never falls below the kept quorum.")
	invariant.Always(
		int(value) <= KEPT_MAX, "A cache references count never exceeds the kept ceiling.")
}

// Cache_References_Unit names the cache-reference (Linux only) dimension; the unit is constant per
// metric, so one exact width is pinnable.
type Cache_References_Unit string

// Cache_References_Unit_Invariants pins the single count width; a both-polarity width witness
// would be unsatisfiable for a constant unit.
func Cache_References_Unit_Invariants(identifier string, value Cache_References_Unit) {
	invariant.Always(
		len(value) == UNIT_BYTES_MIN, "A cache references unit always spans the five-byte count width.")
}

// Cache_References_Measurement is the cache-reference (Linux only) distribution, each leaf typed
// per metric so the bare invariants seed under their own roots.
type Cache_References_Measurement struct {
	// Mean is the arithmetic mean of the metric's values.
	Mean Cache_References_Mean `json:"mean"`
	// Standard_Deviation is the sample standard deviation, with an n-1 denominator.
	Standard_Deviation Cache_References_Standard_Deviation `json:"stddev"`
	// Min is the smallest value observed.
	Min Cache_References_Min `json:"min"`
	// Max is the largest value observed.
	Max Cache_References_Max `json:"max"`
	// Median is the middle value of the sorted values.
	Median Cache_References_Median `json:"median"`
	// Q1 is the first quartile by poop's index math.
	Q1 Cache_References_Q1 `json:"q1"`
	// Q3 is the third quartile by poop's index math.
	Q3 Cache_References_Q3 `json:"q3"`
	// Outlier_Count is how many values fall beyond Tukey's fences.
	Outlier_Count Cache_References_Outliers `json:"outliers"`
	// Sample_Count is how many values the distribution was computed from.
	Sample_Count Cache_References_Count `json:"count"`
	// Unit names the unit the raw values are in.
	Unit Cache_References_Unit `json:"unit"`
}

// Cache_References_Measurement_Invariants forwards every field to its leaf helper so each bound is
// stated at the type that owns it.
func Cache_References_Measurement_Invariants(
	identifier string, value Cache_References_Measurement,
) {
	Cache_References_Mean_Invariants(identifier, value.Mean)
	Cache_References_Standard_Deviation_Invariants(identifier, value.Standard_Deviation)
	Cache_References_Min_Invariants(identifier, value.Min)
	Cache_References_Max_Invariants(identifier, value.Max)
	Cache_References_Median_Invariants(identifier, value.Median)
	Cache_References_Q1_Invariants(identifier, value.Q1)
	Cache_References_Q3_Invariants(identifier, value.Q3)
	Cache_References_Outliers_Invariants(identifier, value.Outlier_Count)
	Cache_References_Count_Invariants(identifier, value.Sample_Count)
	Cache_References_Unit_Invariants(identifier, value.Unit)
}

// As_Measurement converts field-by-field to the plain Measurement; a method rather than a free
// function so the conversion plants no invariant roots.
func (value Cache_References_Measurement) As_Measurement() (measurement Measurement) {
	return Measurement{
		Mean:               Mean(value.Mean),
		Standard_Deviation: Standard_Deviation(value.Standard_Deviation),
		Min:                Min(value.Min),
		Max:                Max(value.Max),
		Median:             Median(value.Median),
		Q1:                 Q1(value.Q1),
		Q3:                 Q3(value.Q3),
		Outlier_Count:      Strays(value.Outlier_Count),
		Sample_Count:       Kept(value.Sample_Count),
		Unit:               Unit(value.Unit),
	}
}

// Cache_Misses_Mean is the arithmetic mean of the cache-miss (Linux only) distribution,
// fixedpoint-scaled; a distinct type gives the bare invariant its own per-metric root.
type Cache_Misses_Mean int64

// Cache_Misses_Mean_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Cache_Misses_Mean_Invariants(identifier string, value Cache_Misses_Mean) {
	invariant.Always(int64(value) >= 0, "A cache misses mean is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Cache_Misses_Mean) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Cache_Misses_Mean) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Cache_Misses_Standard_Deviation is the sample standard deviation of the cache-miss (Linux only)
// distribution, fixedpoint-scaled; a distinct type gives the bare invariant its own per-metric
// root.
type Cache_Misses_Standard_Deviation int64

// Cache_Misses_Standard_Deviation_Invariants pins non-negativity, the honest bound for a statistic
// over a non-negative metric; a coverage witness here would be unfunded debt.
func Cache_Misses_Standard_Deviation_Invariants(
	identifier string, value Cache_Misses_Standard_Deviation,
) {
	invariant.Always(int64(value) >= 0, "A cache misses standard deviation is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Cache_Misses_Standard_Deviation) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Cache_Misses_Standard_Deviation) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Cache_Misses_Min is the smallest observed value of the cache-miss (Linux only) distribution,
// fixedpoint-scaled; a distinct type gives the bare invariant its own per-metric root.
type Cache_Misses_Min int64

// Cache_Misses_Min_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Cache_Misses_Min_Invariants(identifier string, value Cache_Misses_Min) {
	invariant.Always(int64(value) >= 0, "A cache misses min is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Cache_Misses_Min) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Cache_Misses_Min) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Cache_Misses_Max is the largest observed value of the cache-miss (Linux only) distribution,
// fixedpoint-scaled; a distinct type gives the bare invariant its own per-metric root.
type Cache_Misses_Max int64

// Cache_Misses_Max_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Cache_Misses_Max_Invariants(identifier string, value Cache_Misses_Max) {
	invariant.Always(int64(value) >= 0, "A cache misses max is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Cache_Misses_Max) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Cache_Misses_Max) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Cache_Misses_Median is the middle value of the cache-miss (Linux only) distribution, fixedpoint-
// scaled; a distinct type gives the bare invariant its own per-metric root.
type Cache_Misses_Median int64

// Cache_Misses_Median_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Cache_Misses_Median_Invariants(identifier string, value Cache_Misses_Median) {
	invariant.Always(int64(value) >= 0, "A cache misses median is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Cache_Misses_Median) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Cache_Misses_Median) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Cache_Misses_Q1 is the first quartile of the cache-miss (Linux only) distribution, fixedpoint-
// scaled; a distinct type gives the bare invariant its own per-metric root.
type Cache_Misses_Q1 int64

// Cache_Misses_Q1_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Cache_Misses_Q1_Invariants(identifier string, value Cache_Misses_Q1) {
	invariant.Always(int64(value) >= 0, "A cache misses q1 is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Cache_Misses_Q1) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Cache_Misses_Q1) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Cache_Misses_Q3 is the third quartile of the cache-miss (Linux only) distribution, fixedpoint-
// scaled; a distinct type gives the bare invariant its own per-metric root.
type Cache_Misses_Q3 int64

// Cache_Misses_Q3_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Cache_Misses_Q3_Invariants(identifier string, value Cache_Misses_Q3) {
	invariant.Always(int64(value) >= 0, "A cache misses q3 is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Cache_Misses_Q3) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Cache_Misses_Q3) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Cache_Misses_Outliers counts the cache-miss (Linux only) values beyond Tukey's fences; a
// distinct type gives the stray bounds their own per-metric root.
type Cache_Misses_Outliers int

// Cache_Misses_Outliers_Invariants bounds the tally to the stray range; the old framework funded
// no witnesses at this position, so none are planted.
func Cache_Misses_Outliers_Invariants(identifier string, value Cache_Misses_Outliers) {
	invariant.Always(
		int(value) >= STRAYS_MIN, "A cache misses outliers tally never falls below the stray floor.")
	invariant.Always(
		int(value) <= STRAYS_MAX, "A cache misses outliers tally never exceeds the stray ceiling.")
}

// Cache_Misses_Count counts the kept cache-miss (Linux only) runs; a distinct type gives the
// quorum bounds their own per-metric root.
type Cache_Misses_Count int

// Cache_Misses_Count_Invariants bounds the tally to the kept quorum range; the old framework
// funded no witnesses at this position, so none are planted.
func Cache_Misses_Count_Invariants(identifier string, value Cache_Misses_Count) {
	invariant.Always(int(value) >= KEPT_MIN, "A cache misses count never falls below the kept quorum.")
	invariant.Always(int(value) <= KEPT_MAX, "A cache misses count never exceeds the kept ceiling.")
}

// Cache_Misses_Unit names the cache-miss (Linux only) dimension; the unit is constant per metric,
// so one exact width is pinnable.
type Cache_Misses_Unit string

// Cache_Misses_Unit_Invariants pins the single count width; a both-polarity width witness would be
// unsatisfiable for a constant unit.
func Cache_Misses_Unit_Invariants(identifier string, value Cache_Misses_Unit) {
	invariant.Always(
		len(value) == UNIT_BYTES_MIN, "A cache misses unit always spans the five-byte count width.")
}

// Cache_Misses_Measurement is the cache-miss (Linux only) distribution, each leaf typed per metric
// so the bare invariants seed under their own roots.
type Cache_Misses_Measurement struct {
	// Mean is the arithmetic mean of the metric's values.
	Mean Cache_Misses_Mean `json:"mean"`
	// Standard_Deviation is the sample standard deviation, with an n-1 denominator.
	Standard_Deviation Cache_Misses_Standard_Deviation `json:"stddev"`
	// Min is the smallest value observed.
	Min Cache_Misses_Min `json:"min"`
	// Max is the largest value observed.
	Max Cache_Misses_Max `json:"max"`
	// Median is the middle value of the sorted values.
	Median Cache_Misses_Median `json:"median"`
	// Q1 is the first quartile by poop's index math.
	Q1 Cache_Misses_Q1 `json:"q1"`
	// Q3 is the third quartile by poop's index math.
	Q3 Cache_Misses_Q3 `json:"q3"`
	// Outlier_Count is how many values fall beyond Tukey's fences.
	Outlier_Count Cache_Misses_Outliers `json:"outliers"`
	// Sample_Count is how many values the distribution was computed from.
	Sample_Count Cache_Misses_Count `json:"count"`
	// Unit names the unit the raw values are in.
	Unit Cache_Misses_Unit `json:"unit"`
}

// Cache_Misses_Measurement_Invariants forwards every field to its leaf helper so each bound is
// stated at the type that owns it.
func Cache_Misses_Measurement_Invariants(identifier string, value Cache_Misses_Measurement) {
	Cache_Misses_Mean_Invariants(identifier, value.Mean)
	Cache_Misses_Standard_Deviation_Invariants(identifier, value.Standard_Deviation)
	Cache_Misses_Min_Invariants(identifier, value.Min)
	Cache_Misses_Max_Invariants(identifier, value.Max)
	Cache_Misses_Median_Invariants(identifier, value.Median)
	Cache_Misses_Q1_Invariants(identifier, value.Q1)
	Cache_Misses_Q3_Invariants(identifier, value.Q3)
	Cache_Misses_Outliers_Invariants(identifier, value.Outlier_Count)
	Cache_Misses_Count_Invariants(identifier, value.Sample_Count)
	Cache_Misses_Unit_Invariants(identifier, value.Unit)
}

// As_Measurement converts field-by-field to the plain Measurement; a method rather than a free
// function so the conversion plants no invariant roots.
func (value Cache_Misses_Measurement) As_Measurement() (measurement Measurement) {
	return Measurement{
		Mean:               Mean(value.Mean),
		Standard_Deviation: Standard_Deviation(value.Standard_Deviation),
		Min:                Min(value.Min),
		Max:                Max(value.Max),
		Median:             Median(value.Median),
		Q1:                 Q1(value.Q1),
		Q3:                 Q3(value.Q3),
		Outlier_Count:      Strays(value.Outlier_Count),
		Sample_Count:       Kept(value.Sample_Count),
		Unit:               Unit(value.Unit),
	}
}

// Branch_Misses_Mean is the arithmetic mean of the branch-miss (Linux only) distribution,
// fixedpoint-scaled; a distinct type gives the bare invariant its own per-metric root.
type Branch_Misses_Mean int64

// Branch_Misses_Mean_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Branch_Misses_Mean_Invariants(identifier string, value Branch_Misses_Mean) {
	invariant.Always(int64(value) >= 0, "A branch misses mean is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Branch_Misses_Mean) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Branch_Misses_Mean) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Branch_Misses_Standard_Deviation is the sample standard deviation of the branch-miss (Linux
// only) distribution, fixedpoint-scaled; a distinct type gives the bare invariant its own per-
// metric root.
type Branch_Misses_Standard_Deviation int64

// Branch_Misses_Standard_Deviation_Invariants pins non-negativity, the honest bound for a
// statistic over a non-negative metric; a coverage witness here would be unfunded debt.
func Branch_Misses_Standard_Deviation_Invariants(
	identifier string, value Branch_Misses_Standard_Deviation,
) {
	invariant.Always(int64(value) >= 0, "A branch misses standard deviation is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Branch_Misses_Standard_Deviation) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Branch_Misses_Standard_Deviation) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Branch_Misses_Min is the smallest observed value of the branch-miss (Linux only) distribution,
// fixedpoint-scaled; a distinct type gives the bare invariant its own per-metric root.
type Branch_Misses_Min int64

// Branch_Misses_Min_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Branch_Misses_Min_Invariants(identifier string, value Branch_Misses_Min) {
	invariant.Always(int64(value) >= 0, "A branch misses min is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Branch_Misses_Min) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Branch_Misses_Min) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Branch_Misses_Max is the largest observed value of the branch-miss (Linux only) distribution,
// fixedpoint-scaled; a distinct type gives the bare invariant its own per-metric root.
type Branch_Misses_Max int64

// Branch_Misses_Max_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Branch_Misses_Max_Invariants(identifier string, value Branch_Misses_Max) {
	invariant.Always(int64(value) >= 0, "A branch misses max is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Branch_Misses_Max) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Branch_Misses_Max) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Branch_Misses_Median is the middle value of the branch-miss (Linux only) distribution,
// fixedpoint-scaled; a distinct type gives the bare invariant its own per-metric root.
type Branch_Misses_Median int64

// Branch_Misses_Median_Invariants pins non-negativity, the honest bound for a statistic over a
// non-negative metric; a coverage witness here would be unfunded debt.
func Branch_Misses_Median_Invariants(identifier string, value Branch_Misses_Median) {
	invariant.Always(int64(value) >= 0, "A branch misses median is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Branch_Misses_Median) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Branch_Misses_Median) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Branch_Misses_Q1 is the first quartile of the branch-miss (Linux only) distribution, fixedpoint-
// scaled; a distinct type gives the bare invariant its own per-metric root.
type Branch_Misses_Q1 int64

// Branch_Misses_Q1_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Branch_Misses_Q1_Invariants(identifier string, value Branch_Misses_Q1) {
	invariant.Always(int64(value) >= 0, "A branch misses q1 is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Branch_Misses_Q1) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Branch_Misses_Q1) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Branch_Misses_Q3 is the third quartile of the branch-miss (Linux only) distribution, fixedpoint-
// scaled; a distinct type gives the bare invariant its own per-metric root.
type Branch_Misses_Q3 int64

// Branch_Misses_Q3_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func Branch_Misses_Q3_Invariants(identifier string, value Branch_Misses_Q3) {
	invariant.Always(int64(value) >= 0, "A branch misses q3 is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value Branch_Misses_Q3) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *Branch_Misses_Q3) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Branch_Misses_Outliers counts the branch-miss (Linux only) values beyond Tukey's fences; a
// distinct type gives the stray bounds their own per-metric root.
type Branch_Misses_Outliers int

// Branch_Misses_Outliers_Invariants bounds the tally to the stray range; the old framework funded
// no witnesses at this position, so none are planted.
func Branch_Misses_Outliers_Invariants(identifier string, value Branch_Misses_Outliers) {
	invariant.Always(
		int(value) >= STRAYS_MIN, "A branch misses outliers tally never falls below the stray floor.")
	invariant.Always(
		int(value) <= STRAYS_MAX, "A branch misses outliers tally never exceeds the stray ceiling.")
}

// Branch_Misses_Count counts the kept branch-miss (Linux only) runs; a distinct type gives the
// quorum bounds their own per-metric root.
type Branch_Misses_Count int

// Branch_Misses_Count_Invariants bounds the tally to the kept quorum range; the old framework
// funded no witnesses at this position, so none are planted.
func Branch_Misses_Count_Invariants(identifier string, value Branch_Misses_Count) {
	invariant.Always(
		int(value) >= KEPT_MIN, "A branch misses count never falls below the kept quorum.")
	invariant.Always(int(value) <= KEPT_MAX, "A branch misses count never exceeds the kept ceiling.")
}

// Branch_Misses_Unit names the branch-miss (Linux only) dimension; the unit is constant per
// metric, so one exact width is pinnable.
type Branch_Misses_Unit string

// Branch_Misses_Unit_Invariants pins the single count width; a both-polarity width witness would
// be unsatisfiable for a constant unit.
func Branch_Misses_Unit_Invariants(identifier string, value Branch_Misses_Unit) {
	invariant.Always(
		len(value) == UNIT_BYTES_MIN, "A branch misses unit always spans the five-byte count width.")
}

// Branch_Misses_Measurement is the branch-miss (Linux only) distribution, each leaf typed per
// metric so the bare invariants seed under their own roots.
type Branch_Misses_Measurement struct {
	// Mean is the arithmetic mean of the metric's values.
	Mean Branch_Misses_Mean `json:"mean"`
	// Standard_Deviation is the sample standard deviation, with an n-1 denominator.
	Standard_Deviation Branch_Misses_Standard_Deviation `json:"stddev"`
	// Min is the smallest value observed.
	Min Branch_Misses_Min `json:"min"`
	// Max is the largest value observed.
	Max Branch_Misses_Max `json:"max"`
	// Median is the middle value of the sorted values.
	Median Branch_Misses_Median `json:"median"`
	// Q1 is the first quartile by poop's index math.
	Q1 Branch_Misses_Q1 `json:"q1"`
	// Q3 is the third quartile by poop's index math.
	Q3 Branch_Misses_Q3 `json:"q3"`
	// Outlier_Count is how many values fall beyond Tukey's fences.
	Outlier_Count Branch_Misses_Outliers `json:"outliers"`
	// Sample_Count is how many values the distribution was computed from.
	Sample_Count Branch_Misses_Count `json:"count"`
	// Unit names the unit the raw values are in.
	Unit Branch_Misses_Unit `json:"unit"`
}

// Branch_Misses_Measurement_Invariants forwards every field to its leaf helper so each bound is
// stated at the type that owns it.
func Branch_Misses_Measurement_Invariants(identifier string, value Branch_Misses_Measurement) {
	Branch_Misses_Mean_Invariants(identifier, value.Mean)
	Branch_Misses_Standard_Deviation_Invariants(identifier, value.Standard_Deviation)
	Branch_Misses_Min_Invariants(identifier, value.Min)
	Branch_Misses_Max_Invariants(identifier, value.Max)
	Branch_Misses_Median_Invariants(identifier, value.Median)
	Branch_Misses_Q1_Invariants(identifier, value.Q1)
	Branch_Misses_Q3_Invariants(identifier, value.Q3)
	Branch_Misses_Outliers_Invariants(identifier, value.Outlier_Count)
	Branch_Misses_Count_Invariants(identifier, value.Sample_Count)
	Branch_Misses_Unit_Invariants(identifier, value.Unit)
}

// As_Measurement converts field-by-field to the plain Measurement; a method rather than a free
// function so the conversion plants no invariant roots.
func (value Branch_Misses_Measurement) As_Measurement() (measurement Measurement) {
	return Measurement{
		Mean:               Mean(value.Mean),
		Standard_Deviation: Standard_Deviation(value.Standard_Deviation),
		Min:                Min(value.Min),
		Max:                Max(value.Max),
		Median:             Median(value.Median),
		Q1:                 Q1(value.Q1),
		Q3:                 Q3(value.Q3),
		Outlier_Count:      Strays(value.Outlier_Count),
		Sample_Count:       Kept(value.Sample_Count),
		Unit:               Unit(value.Unit),
	}
}

// CPU_User_Mean is the arithmetic mean of the user-CPU-time distribution, fixedpoint-scaled; a
// distinct type gives the bare invariant its own per-metric root.
type CPU_User_Mean int64

// CPU_User_Mean_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func CPU_User_Mean_Invariants(identifier string, value CPU_User_Mean) {
	invariant.Always(int64(value) >= 0, "A cpu user mean is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value CPU_User_Mean) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *CPU_User_Mean) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_User_Standard_Deviation is the sample standard deviation of the user-CPU-time distribution,
// fixedpoint-scaled; a distinct type gives the bare invariant its own per-metric root.
type CPU_User_Standard_Deviation int64

// CPU_User_Standard_Deviation_Invariants pins non-negativity, the honest bound for a statistic
// over a non-negative metric; a coverage witness here would be unfunded debt.
func CPU_User_Standard_Deviation_Invariants(identifier string, value CPU_User_Standard_Deviation) {
	invariant.Always(int64(value) >= 0, "A cpu user standard deviation is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value CPU_User_Standard_Deviation) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *CPU_User_Standard_Deviation) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_User_Min is the smallest observed value of the user-CPU-time distribution, fixedpoint-
// scaled; a distinct type gives the bare invariant its own per-metric root.
type CPU_User_Min int64

// CPU_User_Min_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func CPU_User_Min_Invariants(identifier string, value CPU_User_Min) {
	invariant.Always(int64(value) >= 0, "A cpu user min is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value CPU_User_Min) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *CPU_User_Min) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_User_Max is the largest observed value of the user-CPU-time distribution, fixedpoint-scaled;
// a distinct type gives the bare invariant its own per-metric root.
type CPU_User_Max int64

// CPU_User_Max_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func CPU_User_Max_Invariants(identifier string, value CPU_User_Max) {
	invariant.Always(int64(value) >= 0, "A cpu user max is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value CPU_User_Max) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *CPU_User_Max) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_User_Median is the middle value of the user-CPU-time distribution, fixedpoint-scaled; a
// distinct type gives the bare invariant its own per-metric root.
type CPU_User_Median int64

// CPU_User_Median_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func CPU_User_Median_Invariants(identifier string, value CPU_User_Median) {
	invariant.Always(int64(value) >= 0, "A cpu user median is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value CPU_User_Median) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *CPU_User_Median) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_User_Q1 is the first quartile of the user-CPU-time distribution, fixedpoint-scaled; a
// distinct type gives the bare invariant its own per-metric root.
type CPU_User_Q1 int64

// CPU_User_Q1_Invariants pins non-negativity, the honest bound for a statistic over a non-negative
// metric; a coverage witness here would be unfunded debt.
func CPU_User_Q1_Invariants(identifier string, value CPU_User_Q1) {
	invariant.Always(int64(value) >= 0, "A cpu user q1 is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value CPU_User_Q1) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *CPU_User_Q1) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_User_Q3 is the third quartile of the user-CPU-time distribution, fixedpoint-scaled; a
// distinct type gives the bare invariant its own per-metric root.
type CPU_User_Q3 int64

// CPU_User_Q3_Invariants pins non-negativity, the honest bound for a statistic over a non-negative
// metric; a coverage witness here would be unfunded debt.
func CPU_User_Q3_Invariants(identifier string, value CPU_User_Q3) {
	invariant.Always(int64(value) >= 0, "A cpu user q3 is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value CPU_User_Q3) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *CPU_User_Q3) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_User_Outliers counts the user-CPU-time values beyond Tukey's fences; a distinct type gives
// the stray bounds their own per-metric root.
type CPU_User_Outliers int

// CPU_User_Outliers_Invariants bounds the tally to the stray range; the old framework funded no
// witnesses at this position, so none are planted.
func CPU_User_Outliers_Invariants(identifier string, value CPU_User_Outliers) {
	invariant.Always(
		int(value) >= STRAYS_MIN, "A cpu user outliers tally never falls below the stray floor.")
	invariant.Always(
		int(value) <= STRAYS_MAX, "A cpu user outliers tally never exceeds the stray ceiling.")
}

// CPU_User_Count counts the kept user-CPU-time runs; a distinct type gives the quorum bounds their
// own per-metric root.
type CPU_User_Count int

// CPU_User_Count_Invariants bounds the tally to the kept quorum range; the old framework funded no
// witnesses at this position, so none are planted.
func CPU_User_Count_Invariants(identifier string, value CPU_User_Count) {
	invariant.Always(int(value) >= KEPT_MIN, "A cpu user count never falls below the kept quorum.")
	invariant.Always(int(value) <= KEPT_MAX, "A cpu user count never exceeds the kept ceiling.")
}

// CPU_User_Unit names the user-CPU-time dimension; the unit is constant per metric, so one exact
// width is pinnable.
type CPU_User_Unit string

// CPU_User_Unit_Invariants pins the single nanoseconds width; a both-polarity width witness would
// be unsatisfiable for a constant unit.
func CPU_User_Unit_Invariants(identifier string, value CPU_User_Unit) {
	invariant.Always(
		len(value) == UNIT_BYTES_MAX, "A cpu user unit always spans the eleven-byte nanoseconds width.")
}

// CPU_User_Measurement is the user-CPU-time distribution, each leaf typed per metric so the bare
// invariants seed under their own roots.
type CPU_User_Measurement struct {
	// Mean is the arithmetic mean of the metric's values.
	Mean CPU_User_Mean `json:"mean"`
	// Standard_Deviation is the sample standard deviation, with an n-1 denominator.
	Standard_Deviation CPU_User_Standard_Deviation `json:"stddev"`
	// Min is the smallest value observed.
	Min CPU_User_Min `json:"min"`
	// Max is the largest value observed.
	Max CPU_User_Max `json:"max"`
	// Median is the middle value of the sorted values.
	Median CPU_User_Median `json:"median"`
	// Q1 is the first quartile by poop's index math.
	Q1 CPU_User_Q1 `json:"q1"`
	// Q3 is the third quartile by poop's index math.
	Q3 CPU_User_Q3 `json:"q3"`
	// Outlier_Count is how many values fall beyond Tukey's fences.
	Outlier_Count CPU_User_Outliers `json:"outliers"`
	// Sample_Count is how many values the distribution was computed from.
	Sample_Count CPU_User_Count `json:"count"`
	// Unit names the unit the raw values are in.
	Unit CPU_User_Unit `json:"unit"`
}

// CPU_User_Measurement_Invariants forwards every field to its leaf helper so each bound is stated
// at the type that owns it.
func CPU_User_Measurement_Invariants(identifier string, value CPU_User_Measurement) {
	CPU_User_Mean_Invariants(identifier, value.Mean)
	CPU_User_Standard_Deviation_Invariants(identifier, value.Standard_Deviation)
	CPU_User_Min_Invariants(identifier, value.Min)
	CPU_User_Max_Invariants(identifier, value.Max)
	CPU_User_Median_Invariants(identifier, value.Median)
	CPU_User_Q1_Invariants(identifier, value.Q1)
	CPU_User_Q3_Invariants(identifier, value.Q3)
	CPU_User_Outliers_Invariants(identifier, value.Outlier_Count)
	CPU_User_Count_Invariants(identifier, value.Sample_Count)
	CPU_User_Unit_Invariants(identifier, value.Unit)
}

// As_Measurement converts field-by-field to the plain Measurement; a method rather than a free
// function so the conversion plants no invariant roots.
func (value CPU_User_Measurement) As_Measurement() (measurement Measurement) {
	return Measurement{
		Mean:               Mean(value.Mean),
		Standard_Deviation: Standard_Deviation(value.Standard_Deviation),
		Min:                Min(value.Min),
		Max:                Max(value.Max),
		Median:             Median(value.Median),
		Q1:                 Q1(value.Q1),
		Q3:                 Q3(value.Q3),
		Outlier_Count:      Strays(value.Outlier_Count),
		Sample_Count:       Kept(value.Sample_Count),
		Unit:               Unit(value.Unit),
	}
}

// CPU_System_Mean is the arithmetic mean of the system-CPU-time distribution, fixedpoint-scaled; a
// distinct type gives the bare invariant its own per-metric root.
type CPU_System_Mean int64

// CPU_System_Mean_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func CPU_System_Mean_Invariants(identifier string, value CPU_System_Mean) {
	invariant.Always(int64(value) >= 0, "A cpu system mean is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value CPU_System_Mean) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *CPU_System_Mean) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_System_Standard_Deviation is the sample standard deviation of the system-CPU-time
// distribution, fixedpoint-scaled; a distinct type gives the bare invariant its own per-metric
// root.
type CPU_System_Standard_Deviation int64

// CPU_System_Standard_Deviation_Invariants pins non-negativity, the honest bound for a statistic
// over a non-negative metric; a coverage witness here would be unfunded debt.
func CPU_System_Standard_Deviation_Invariants(
	identifier string, value CPU_System_Standard_Deviation,
) {
	invariant.Always(int64(value) >= 0, "A cpu system standard deviation is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value CPU_System_Standard_Deviation) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *CPU_System_Standard_Deviation) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_System_Min is the smallest observed value of the system-CPU-time distribution, fixedpoint-
// scaled; a distinct type gives the bare invariant its own per-metric root.
type CPU_System_Min int64

// CPU_System_Min_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func CPU_System_Min_Invariants(identifier string, value CPU_System_Min) {
	invariant.Always(int64(value) >= 0, "A cpu system min is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value CPU_System_Min) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *CPU_System_Min) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_System_Max is the largest observed value of the system-CPU-time distribution, fixedpoint-
// scaled; a distinct type gives the bare invariant its own per-metric root.
type CPU_System_Max int64

// CPU_System_Max_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func CPU_System_Max_Invariants(identifier string, value CPU_System_Max) {
	invariant.Always(int64(value) >= 0, "A cpu system max is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value CPU_System_Max) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *CPU_System_Max) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_System_Median is the middle value of the system-CPU-time distribution, fixedpoint-scaled; a
// distinct type gives the bare invariant its own per-metric root.
type CPU_System_Median int64

// CPU_System_Median_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func CPU_System_Median_Invariants(identifier string, value CPU_System_Median) {
	invariant.Always(int64(value) >= 0, "A cpu system median is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value CPU_System_Median) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *CPU_System_Median) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_System_Q1 is the first quartile of the system-CPU-time distribution, fixedpoint-scaled; a
// distinct type gives the bare invariant its own per-metric root.
type CPU_System_Q1 int64

// CPU_System_Q1_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func CPU_System_Q1_Invariants(identifier string, value CPU_System_Q1) {
	invariant.Always(int64(value) >= 0, "A cpu system q1 is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value CPU_System_Q1) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *CPU_System_Q1) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_System_Q3 is the third quartile of the system-CPU-time distribution, fixedpoint-scaled; a
// distinct type gives the bare invariant its own per-metric root.
type CPU_System_Q3 int64

// CPU_System_Q3_Invariants pins non-negativity, the honest bound for a statistic over a non-
// negative metric; a coverage witness here would be unfunded debt.
func CPU_System_Q3_Invariants(identifier string, value CPU_System_Q3) {
	invariant.Always(int64(value) >= 0, "A cpu system q3 is never negative.")
}

// MarshalJSON delegates to fixedpoint.Number because the field is a fixedpoint-scaled value and
// the report must keep its bare-decimal rendering.
func (value CPU_System_Q3) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint.Number so the decimal the old report emitted round-trips
// onto the fixed-point grid.
func (value *CPU_System_Q3) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_System_Outliers counts the system-CPU-time values beyond Tukey's fences; a distinct type
// gives the stray bounds their own per-metric root.
type CPU_System_Outliers int

// CPU_System_Outliers_Invariants bounds the tally to the stray range; the old framework funded no
// witnesses at this position, so none are planted.
func CPU_System_Outliers_Invariants(identifier string, value CPU_System_Outliers) {
	invariant.Always(
		int(value) >= STRAYS_MIN, "A cpu system outliers tally never falls below the stray floor.")
	invariant.Always(
		int(value) <= STRAYS_MAX, "A cpu system outliers tally never exceeds the stray ceiling.")
}

// CPU_System_Count counts the kept system-CPU-time runs; a distinct type gives the quorum bounds
// their own per-metric root.
type CPU_System_Count int

// CPU_System_Count_Invariants bounds the tally to the kept quorum range; the old framework funded
// no witnesses at this position, so none are planted.
func CPU_System_Count_Invariants(identifier string, value CPU_System_Count) {
	invariant.Always(int(value) >= KEPT_MIN, "A cpu system count never falls below the kept quorum.")
	invariant.Always(int(value) <= KEPT_MAX, "A cpu system count never exceeds the kept ceiling.")
}

// CPU_System_Unit names the system-CPU-time dimension; the unit is constant per metric, so one
// exact width is pinnable.
type CPU_System_Unit string

// CPU_System_Unit_Invariants pins the single nanoseconds width; a both-polarity width witness
// would be unsatisfiable for a constant unit.
func CPU_System_Unit_Invariants(identifier string, value CPU_System_Unit) {
	invariant.Always(
		len(value) == UNIT_BYTES_MAX, "A cpu system unit always spans the eleven-byte nanoseconds width.")
}

// CPU_System_Measurement is the system-CPU-time distribution, each leaf typed per metric so the
// bare invariants seed under their own roots.
type CPU_System_Measurement struct {
	// Mean is the arithmetic mean of the metric's values.
	Mean CPU_System_Mean `json:"mean"`
	// Standard_Deviation is the sample standard deviation, with an n-1 denominator.
	Standard_Deviation CPU_System_Standard_Deviation `json:"stddev"`
	// Min is the smallest value observed.
	Min CPU_System_Min `json:"min"`
	// Max is the largest value observed.
	Max CPU_System_Max `json:"max"`
	// Median is the middle value of the sorted values.
	Median CPU_System_Median `json:"median"`
	// Q1 is the first quartile by poop's index math.
	Q1 CPU_System_Q1 `json:"q1"`
	// Q3 is the third quartile by poop's index math.
	Q3 CPU_System_Q3 `json:"q3"`
	// Outlier_Count is how many values fall beyond Tukey's fences.
	Outlier_Count CPU_System_Outliers `json:"outliers"`
	// Sample_Count is how many values the distribution was computed from.
	Sample_Count CPU_System_Count `json:"count"`
	// Unit names the unit the raw values are in.
	Unit CPU_System_Unit `json:"unit"`
}

// CPU_System_Measurement_Invariants forwards every field to its leaf helper so each bound is
// stated at the type that owns it.
func CPU_System_Measurement_Invariants(identifier string, value CPU_System_Measurement) {
	CPU_System_Mean_Invariants(identifier, value.Mean)
	CPU_System_Standard_Deviation_Invariants(identifier, value.Standard_Deviation)
	CPU_System_Min_Invariants(identifier, value.Min)
	CPU_System_Max_Invariants(identifier, value.Max)
	CPU_System_Median_Invariants(identifier, value.Median)
	CPU_System_Q1_Invariants(identifier, value.Q1)
	CPU_System_Q3_Invariants(identifier, value.Q3)
	CPU_System_Outliers_Invariants(identifier, value.Outlier_Count)
	CPU_System_Count_Invariants(identifier, value.Sample_Count)
	CPU_System_Unit_Invariants(identifier, value.Unit)
}

// As_Measurement converts field-by-field to the plain Measurement; a method rather than a free
// function so the conversion plants no invariant roots.
func (value CPU_System_Measurement) As_Measurement() (measurement Measurement) {
	return Measurement{
		Mean:               Mean(value.Mean),
		Standard_Deviation: Standard_Deviation(value.Standard_Deviation),
		Min:                Min(value.Min),
		Max:                Max(value.Max),
		Median:             Median(value.Median),
		Q1:                 Q1(value.Q1),
		Q3:                 Q3(value.Q3),
		Outlier_Count:      Strays(value.Outlier_Count),
		Sample_Count:       Kept(value.Sample_Count),
		Unit:               Unit(value.Unit),
	}
}

// Measurements is the distribution of every metric for one command, in report
// order. The field order is the JSON order, so the report needs no custom encoder.
type Measurements struct {
	// Wall_Time is the elapsed-time distribution.
	Wall_Time Wall_Time_Measurement `json:"wall_time"`
	// Peak_RSS is the peak-memory distribution.
	Peak_RSS Peak_RSS_Measurement `json:"peak_rss"`
	// CPU_Cycles is the CPU-cycle distribution.
	CPU_Cycles CPU_Cycles_Measurement `json:"cpu_cycles"`
	// Instructions is the retired-instruction distribution.
	Instructions Instructions_Measurement `json:"instructions"`
	// Cache_References is the cache-reference distribution (Linux only).
	Cache_References Cache_References_Measurement `json:"cache_references"`
	// Cache_Misses is the cache-miss distribution (Linux only).
	Cache_Misses Cache_Misses_Measurement `json:"cache_misses"`
	// Branch_Misses is the branch-miss distribution (Linux only).
	Branch_Misses Branch_Misses_Measurement `json:"branch_misses"`
	// CPU_User is the user-CPU-time distribution.
	CPU_User CPU_User_Measurement `json:"cpu_user"`
	// CPU_System is the system-CPU-time distribution.
	CPU_System CPU_System_Measurement `json:"cpu_system"`
}

// Measurements_Invariants forwards every metric to its variant helper so each metric's bounds seed
// under its own root.
func Measurements_Invariants(identifier string, measurements Measurements) {
	Wall_Time_Measurement_Invariants(identifier, measurements.Wall_Time)
	Peak_RSS_Measurement_Invariants(identifier, measurements.Peak_RSS)
	CPU_Cycles_Measurement_Invariants(identifier, measurements.CPU_Cycles)
	Instructions_Measurement_Invariants(identifier, measurements.Instructions)
	Cache_References_Measurement_Invariants(identifier, measurements.Cache_References)
	Cache_Misses_Measurement_Invariants(identifier, measurements.Cache_Misses)
	Branch_Misses_Measurement_Invariants(identifier, measurements.Branch_Misses)
	CPU_User_Measurement_Invariants(identifier, measurements.CPU_User)
	CPU_System_Measurement_Invariants(identifier, measurements.CPU_System)
}

// As_Wall_Time converts field-by-field to the elapsed-time variant; a method rather than a free
// function so the conversion plants no invariant roots.
func (measurement Measurement) As_Wall_Time() (value Wall_Time_Measurement) {
	return Wall_Time_Measurement{
		Mean:               Wall_Time_Mean(measurement.Mean),
		Standard_Deviation: Wall_Time_Standard_Deviation(measurement.Standard_Deviation),
		Min:                Wall_Time_Min(measurement.Min),
		Max:                Wall_Time_Max(measurement.Max),
		Median:             Wall_Time_Median(measurement.Median),
		Q1:                 Wall_Time_Q1(measurement.Q1),
		Q3:                 Wall_Time_Q3(measurement.Q3),
		Outlier_Count:      Wall_Time_Outliers(measurement.Outlier_Count),
		Sample_Count:       Wall_Time_Count(measurement.Sample_Count),
		Unit:               Wall_Time_Unit(measurement.Unit),
	}
}

// As_Peak_RSS converts field-by-field to the peak-memory variant; a method rather than a free
// function so the conversion plants no invariant roots.
func (measurement Measurement) As_Peak_RSS() (value Peak_RSS_Measurement) {
	return Peak_RSS_Measurement{
		Mean:               Peak_RSS_Mean(measurement.Mean),
		Standard_Deviation: Peak_RSS_Standard_Deviation(measurement.Standard_Deviation),
		Min:                Peak_RSS_Min(measurement.Min),
		Max:                Peak_RSS_Max(measurement.Max),
		Median:             Peak_RSS_Median(measurement.Median),
		Q1:                 Peak_RSS_Q1(measurement.Q1),
		Q3:                 Peak_RSS_Q3(measurement.Q3),
		Outlier_Count:      Peak_RSS_Outliers(measurement.Outlier_Count),
		Sample_Count:       Peak_RSS_Count(measurement.Sample_Count),
		Unit:               Peak_RSS_Unit(measurement.Unit),
	}
}

// As_CPU_Cycles converts field-by-field to the CPU-cycle variant; a method rather than a free
// function so the conversion plants no invariant roots.
func (measurement Measurement) As_CPU_Cycles() (value CPU_Cycles_Measurement) {
	return CPU_Cycles_Measurement{
		Mean:               CPU_Cycles_Mean(measurement.Mean),
		Standard_Deviation: CPU_Cycles_Standard_Deviation(measurement.Standard_Deviation),
		Min:                CPU_Cycles_Min(measurement.Min),
		Max:                CPU_Cycles_Max(measurement.Max),
		Median:             CPU_Cycles_Median(measurement.Median),
		Q1:                 CPU_Cycles_Q1(measurement.Q1),
		Q3:                 CPU_Cycles_Q3(measurement.Q3),
		Outlier_Count:      CPU_Cycles_Outliers(measurement.Outlier_Count),
		Sample_Count:       CPU_Cycles_Count(measurement.Sample_Count),
		Unit:               CPU_Cycles_Unit(measurement.Unit),
	}
}

// As_Instructions converts field-by-field to the retired-instruction variant; a method rather than
// a free function so the conversion plants no invariant roots.
func (measurement Measurement) As_Instructions() (value Instructions_Measurement) {
	return Instructions_Measurement{
		Mean:               Instructions_Mean(measurement.Mean),
		Standard_Deviation: Instructions_Standard_Deviation(measurement.Standard_Deviation),
		Min:                Instructions_Min(measurement.Min),
		Max:                Instructions_Max(measurement.Max),
		Median:             Instructions_Median(measurement.Median),
		Q1:                 Instructions_Q1(measurement.Q1),
		Q3:                 Instructions_Q3(measurement.Q3),
		Outlier_Count:      Instructions_Outliers(measurement.Outlier_Count),
		Sample_Count:       Instructions_Count(measurement.Sample_Count),
		Unit:               Instructions_Unit(measurement.Unit),
	}
}

// As_Cache_References converts field-by-field to the cache-reference (Linux only) variant; a
// method rather than a free function so the conversion plants no invariant roots.
func (measurement Measurement) As_Cache_References() (value Cache_References_Measurement) {
	return Cache_References_Measurement{
		Mean:               Cache_References_Mean(measurement.Mean),
		Standard_Deviation: Cache_References_Standard_Deviation(measurement.Standard_Deviation),
		Min:                Cache_References_Min(measurement.Min),
		Max:                Cache_References_Max(measurement.Max),
		Median:             Cache_References_Median(measurement.Median),
		Q1:                 Cache_References_Q1(measurement.Q1),
		Q3:                 Cache_References_Q3(measurement.Q3),
		Outlier_Count:      Cache_References_Outliers(measurement.Outlier_Count),
		Sample_Count:       Cache_References_Count(measurement.Sample_Count),
		Unit:               Cache_References_Unit(measurement.Unit),
	}
}

// As_Cache_Misses converts field-by-field to the cache-miss (Linux only) variant; a method rather
// than a free function so the conversion plants no invariant roots.
func (measurement Measurement) As_Cache_Misses() (value Cache_Misses_Measurement) {
	return Cache_Misses_Measurement{
		Mean:               Cache_Misses_Mean(measurement.Mean),
		Standard_Deviation: Cache_Misses_Standard_Deviation(measurement.Standard_Deviation),
		Min:                Cache_Misses_Min(measurement.Min),
		Max:                Cache_Misses_Max(measurement.Max),
		Median:             Cache_Misses_Median(measurement.Median),
		Q1:                 Cache_Misses_Q1(measurement.Q1),
		Q3:                 Cache_Misses_Q3(measurement.Q3),
		Outlier_Count:      Cache_Misses_Outliers(measurement.Outlier_Count),
		Sample_Count:       Cache_Misses_Count(measurement.Sample_Count),
		Unit:               Cache_Misses_Unit(measurement.Unit),
	}
}

// As_Branch_Misses converts field-by-field to the branch-miss (Linux only) variant; a method
// rather than a free function so the conversion plants no invariant roots.
func (measurement Measurement) As_Branch_Misses() (value Branch_Misses_Measurement) {
	return Branch_Misses_Measurement{
		Mean:               Branch_Misses_Mean(measurement.Mean),
		Standard_Deviation: Branch_Misses_Standard_Deviation(measurement.Standard_Deviation),
		Min:                Branch_Misses_Min(measurement.Min),
		Max:                Branch_Misses_Max(measurement.Max),
		Median:             Branch_Misses_Median(measurement.Median),
		Q1:                 Branch_Misses_Q1(measurement.Q1),
		Q3:                 Branch_Misses_Q3(measurement.Q3),
		Outlier_Count:      Branch_Misses_Outliers(measurement.Outlier_Count),
		Sample_Count:       Branch_Misses_Count(measurement.Sample_Count),
		Unit:               Branch_Misses_Unit(measurement.Unit),
	}
}

// As_CPU_User converts field-by-field to the user-CPU-time variant; a method rather than a free
// function so the conversion plants no invariant roots.
func (measurement Measurement) As_CPU_User() (value CPU_User_Measurement) {
	return CPU_User_Measurement{
		Mean:               CPU_User_Mean(measurement.Mean),
		Standard_Deviation: CPU_User_Standard_Deviation(measurement.Standard_Deviation),
		Min:                CPU_User_Min(measurement.Min),
		Max:                CPU_User_Max(measurement.Max),
		Median:             CPU_User_Median(measurement.Median),
		Q1:                 CPU_User_Q1(measurement.Q1),
		Q3:                 CPU_User_Q3(measurement.Q3),
		Outlier_Count:      CPU_User_Outliers(measurement.Outlier_Count),
		Sample_Count:       CPU_User_Count(measurement.Sample_Count),
		Unit:               CPU_User_Unit(measurement.Unit),
	}
}

// As_CPU_System converts field-by-field to the system-CPU-time variant; a method rather than a
// free function so the conversion plants no invariant roots.
func (measurement Measurement) As_CPU_System() (value CPU_System_Measurement) {
	return CPU_System_Measurement{
		Mean:               CPU_System_Mean(measurement.Mean),
		Standard_Deviation: CPU_System_Standard_Deviation(measurement.Standard_Deviation),
		Min:                CPU_System_Min(measurement.Min),
		Max:                CPU_System_Max(measurement.Max),
		Median:             CPU_System_Median(measurement.Median),
		Q1:                 CPU_System_Q1(measurement.Q1),
		Q3:                 CPU_System_Q3(measurement.Q3),
		Outlier_Count:      CPU_System_Outliers(measurement.Outlier_Count),
		Sample_Count:       CPU_System_Count(measurement.Sample_Count),
		Unit:               CPU_System_Unit(measurement.Unit),
	}
}
