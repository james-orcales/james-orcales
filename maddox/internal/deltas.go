package maddox

import (
	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/fixedpoint"
)

// DIFF_PERCENT_MIN floors every diff percent: a relative difference taken against a
// non-negative reference cannot fall below negative one hundred percent.
const DIFF_PERCENT_MIN = -100 * fixedpoint.SCALE

// Wall_Time_Diff_Percent is the elapsed-time relative difference, fixedpoint-scaled.
type Wall_Time_Diff_Percent int64

// Wall_Time_Diff_Percent_Invariants states the floor alone: the old framework demanded no
// witness of this position, so none is funded.
func Wall_Time_Diff_Percent_Invariants(identifier string, value Wall_Time_Diff_Percent) {
	invariant.Always(int64(value) >= DIFF_PERCENT_MIN,
		"The wall time diff percent never falls below negative one hundred percent.")
}

// MarshalJSON delegates to fixedpoint so the report keeps its bare-decimal rendering.
func (value Wall_Time_Diff_Percent) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint so the report's bare decimal round-trips.
func (value *Wall_Time_Diff_Percent) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Wall_Time_Half_Percent is the elapsed-time confidence half-interval, fixedpoint-scaled.
type Wall_Time_Half_Percent int64

// Wall_Time_Half_Percent_Invariants states non-negativity alone: the old framework demanded
// no witness of this position, so none is funded.
func Wall_Time_Half_Percent_Invariants(identifier string, value Wall_Time_Half_Percent) {
	invariant.Always(int64(value) >= 0,
		"The wall time half percent half interval is never negative.")
}

// MarshalJSON delegates to fixedpoint so the report keeps its bare-decimal rendering.
func (value Wall_Time_Half_Percent) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint so the report's bare decimal round-trips.
func (value *Wall_Time_Half_Percent) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Wall_Time_Significant reports whether the elapsed-time interval clears the ±1% band.
type Wall_Time_Significant bool

// Wall_Time_Significant_Invariants witnesses both verdicts of significance: the
// per-metric verdicts are the family's only funded coverage.
func Wall_Time_Significant_Invariants(identifier string, value Wall_Time_Significant) {
	invariant.Sometimes(identifier, bool(value),
		"The wall time delta clears the significance band.")
}

// Wall_Time_Faster reports whether the elapsed-time mean sits below the reference's.
type Wall_Time_Faster bool

// Wall_Time_Faster_Invariants witnesses both directions of the mean comparison: the
// per-metric verdicts are the family's only funded coverage.
func Wall_Time_Faster_Invariants(identifier string, value Wall_Time_Faster) {
	invariant.Sometimes(identifier, bool(value),
		"The wall time candidate runs faster than the reference.")
}

// Wall_Time_Delta is the elapsed-time change against the reference.
type Wall_Time_Delta struct {
	// Diff_Percent is the candidate's mean as a signed percentage of the reference's.
	Diff_Percent Wall_Time_Diff_Percent `json:"diff_percent"`
	// Half_Percent is the half-width of the 95% confidence interval on Diff_Percent.
	Half_Percent Wall_Time_Half_Percent `json:"half_percent"`
	// Significant is true only when the interval clears the ±1% band.
	Significant Wall_Time_Significant `json:"significant"`
	// Faster is true when the candidate's mean is below the reference's.
	Faster Wall_Time_Faster `json:"faster"`
}

// Wall_Time_Delta_Invariants composes the field invariants of the elapsed-time change.
func Wall_Time_Delta_Invariants(identifier string, value Wall_Time_Delta) {
	Wall_Time_Diff_Percent_Invariants(identifier, value.Diff_Percent)
	Wall_Time_Half_Percent_Invariants(identifier, value.Half_Percent)
	Wall_Time_Significant_Invariants(identifier, value.Significant)
	Wall_Time_Faster_Invariants(identifier, value.Faster)
}

// As_Delta converts field-by-field to the plain Delta.
// A method rather than a free function so the conversion plants no invariant roots.
func (value Wall_Time_Delta) As_Delta() (delta Delta) {
	delta.Diff_Percent = Diff_Percent(value.Diff_Percent)
	delta.Half_Percent = Half_Percent(value.Half_Percent)
	delta.Significant = Significant(value.Significant)
	delta.Faster = Faster(value.Faster)
	return delta
}

// Peak_RSS_Diff_Percent is the peak-memory relative difference, fixedpoint-scaled.
type Peak_RSS_Diff_Percent int64

// Peak_RSS_Diff_Percent_Invariants states the floor alone: the old framework demanded no
// witness of this position, so none is funded.
func Peak_RSS_Diff_Percent_Invariants(identifier string, value Peak_RSS_Diff_Percent) {
	invariant.Always(int64(value) >= DIFF_PERCENT_MIN,
		"The peak rss diff percent never falls below negative one hundred percent.")
}

// MarshalJSON delegates to fixedpoint so the report keeps its bare-decimal rendering.
func (value Peak_RSS_Diff_Percent) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint so the report's bare decimal round-trips.
func (value *Peak_RSS_Diff_Percent) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Peak_RSS_Half_Percent is the peak-memory confidence half-interval, fixedpoint-scaled.
type Peak_RSS_Half_Percent int64

// Peak_RSS_Half_Percent_Invariants states non-negativity alone: the old framework demanded
// no witness of this position, so none is funded.
func Peak_RSS_Half_Percent_Invariants(identifier string, value Peak_RSS_Half_Percent) {
	invariant.Always(int64(value) >= 0,
		"The peak rss half percent half interval is never negative.")
}

// MarshalJSON delegates to fixedpoint so the report keeps its bare-decimal rendering.
func (value Peak_RSS_Half_Percent) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint so the report's bare decimal round-trips.
func (value *Peak_RSS_Half_Percent) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Peak_RSS_Significant reports whether the peak-memory interval clears the ±1% band.
type Peak_RSS_Significant bool

// Peak_RSS_Significant_Invariants witnesses both verdicts of significance: the
// per-metric verdicts are the family's only funded coverage.
func Peak_RSS_Significant_Invariants(identifier string, value Peak_RSS_Significant) {
	invariant.Sometimes(identifier, bool(value),
		"The peak rss delta clears the significance band.")
}

// Peak_RSS_Faster reports whether the peak-memory mean sits below the reference's.
type Peak_RSS_Faster bool

// Peak_RSS_Faster_Invariants witnesses both directions of the mean comparison: the
// per-metric verdicts are the family's only funded coverage.
func Peak_RSS_Faster_Invariants(identifier string, value Peak_RSS_Faster) {
	invariant.Sometimes(identifier, bool(value),
		"The peak rss candidate peaks lower than the reference.")
}

// Peak_RSS_Delta is the peak-memory change against the reference.
type Peak_RSS_Delta struct {
	// Diff_Percent is the candidate's mean as a signed percentage of the reference's.
	Diff_Percent Peak_RSS_Diff_Percent `json:"diff_percent"`
	// Half_Percent is the half-width of the 95% confidence interval on Diff_Percent.
	Half_Percent Peak_RSS_Half_Percent `json:"half_percent"`
	// Significant is true only when the interval clears the ±1% band.
	Significant Peak_RSS_Significant `json:"significant"`
	// Faster is true when the candidate's mean is below the reference's.
	Faster Peak_RSS_Faster `json:"faster"`
}

// Peak_RSS_Delta_Invariants composes the field invariants of the peak-memory change.
func Peak_RSS_Delta_Invariants(identifier string, value Peak_RSS_Delta) {
	Peak_RSS_Diff_Percent_Invariants(identifier, value.Diff_Percent)
	Peak_RSS_Half_Percent_Invariants(identifier, value.Half_Percent)
	Peak_RSS_Significant_Invariants(identifier, value.Significant)
	Peak_RSS_Faster_Invariants(identifier, value.Faster)
}

// As_Delta converts field-by-field to the plain Delta.
// A method rather than a free function so the conversion plants no invariant roots.
func (value Peak_RSS_Delta) As_Delta() (delta Delta) {
	delta.Diff_Percent = Diff_Percent(value.Diff_Percent)
	delta.Half_Percent = Half_Percent(value.Half_Percent)
	delta.Significant = Significant(value.Significant)
	delta.Faster = Faster(value.Faster)
	return delta
}

// CPU_Cycles_Diff_Percent is the CPU-cycle relative difference, fixedpoint-scaled.
type CPU_Cycles_Diff_Percent int64

// CPU_Cycles_Diff_Percent_Invariants states the floor alone: the old framework demanded no
// witness of this position, so none is funded.
func CPU_Cycles_Diff_Percent_Invariants(identifier string, value CPU_Cycles_Diff_Percent) {
	invariant.Always(int64(value) >= DIFF_PERCENT_MIN,
		"The cpu cycles diff percent never falls below negative one hundred percent.")
}

// MarshalJSON delegates to fixedpoint so the report keeps its bare-decimal rendering.
func (value CPU_Cycles_Diff_Percent) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint so the report's bare decimal round-trips.
func (value *CPU_Cycles_Diff_Percent) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_Cycles_Half_Percent is the CPU-cycle confidence half-interval, fixedpoint-scaled.
type CPU_Cycles_Half_Percent int64

// CPU_Cycles_Half_Percent_Invariants states non-negativity alone: the old framework demanded
// no witness of this position, so none is funded.
func CPU_Cycles_Half_Percent_Invariants(identifier string, value CPU_Cycles_Half_Percent) {
	invariant.Always(int64(value) >= 0,
		"The cpu cycles half percent half interval is never negative.")
}

// MarshalJSON delegates to fixedpoint so the report keeps its bare-decimal rendering.
func (value CPU_Cycles_Half_Percent) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint so the report's bare decimal round-trips.
func (value *CPU_Cycles_Half_Percent) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_Cycles_Significant reports whether the CPU-cycle interval clears the ±1% band.
type CPU_Cycles_Significant bool

// CPU_Cycles_Significant_Invariants witnesses both verdicts of significance: the
// per-metric verdicts are the family's only funded coverage.
func CPU_Cycles_Significant_Invariants(identifier string, value CPU_Cycles_Significant) {
	invariant.Sometimes(identifier, bool(value),
		"The cpu cycles delta clears the significance band.")
}

// CPU_Cycles_Faster reports whether the CPU-cycle mean sits below the reference's.
type CPU_Cycles_Faster bool

// CPU_Cycles_Faster_Invariants witnesses both directions of the mean comparison: the
// per-metric verdicts are the family's only funded coverage.
func CPU_Cycles_Faster_Invariants(identifier string, value CPU_Cycles_Faster) {
	invariant.Sometimes(identifier, bool(value),
		"The cpu cycles candidate burns fewer cycles than the reference.")
}

// CPU_Cycles_Delta is the CPU-cycle change against the reference.
type CPU_Cycles_Delta struct {
	// Diff_Percent is the candidate's mean as a signed percentage of the reference's.
	Diff_Percent CPU_Cycles_Diff_Percent `json:"diff_percent"`
	// Half_Percent is the half-width of the 95% confidence interval on Diff_Percent.
	Half_Percent CPU_Cycles_Half_Percent `json:"half_percent"`
	// Significant is true only when the interval clears the ±1% band.
	Significant CPU_Cycles_Significant `json:"significant"`
	// Faster is true when the candidate's mean is below the reference's.
	Faster CPU_Cycles_Faster `json:"faster"`
}

// CPU_Cycles_Delta_Invariants composes the field invariants of the CPU-cycle change.
func CPU_Cycles_Delta_Invariants(identifier string, value CPU_Cycles_Delta) {
	CPU_Cycles_Diff_Percent_Invariants(identifier, value.Diff_Percent)
	CPU_Cycles_Half_Percent_Invariants(identifier, value.Half_Percent)
	CPU_Cycles_Significant_Invariants(identifier, value.Significant)
	CPU_Cycles_Faster_Invariants(identifier, value.Faster)
}

// As_Delta converts field-by-field to the plain Delta.
// A method rather than a free function so the conversion plants no invariant roots.
func (value CPU_Cycles_Delta) As_Delta() (delta Delta) {
	delta.Diff_Percent = Diff_Percent(value.Diff_Percent)
	delta.Half_Percent = Half_Percent(value.Half_Percent)
	delta.Significant = Significant(value.Significant)
	delta.Faster = Faster(value.Faster)
	return delta
}

// Instructions_Diff_Percent is the retired-instruction relative difference, fixedpoint-scaled.
type Instructions_Diff_Percent int64

// Instructions_Diff_Percent_Invariants states the floor alone: the old framework demanded no
// witness of this position, so none is funded.
func Instructions_Diff_Percent_Invariants(identifier string, value Instructions_Diff_Percent) {
	invariant.Always(int64(value) >= DIFF_PERCENT_MIN,
		"The instructions diff percent never falls below negative one hundred percent.")
}

// MarshalJSON delegates to fixedpoint so the report keeps its bare-decimal rendering.
func (value Instructions_Diff_Percent) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint so the report's bare decimal round-trips.
func (value *Instructions_Diff_Percent) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Instructions_Half_Percent is the retired-instruction confidence half-interval, fixedpoint-scaled.
type Instructions_Half_Percent int64

// Instructions_Half_Percent_Invariants states non-negativity alone: the old framework demanded
// no witness of this position, so none is funded.
func Instructions_Half_Percent_Invariants(identifier string, value Instructions_Half_Percent) {
	invariant.Always(int64(value) >= 0,
		"The instructions half percent half interval is never negative.")
}

// MarshalJSON delegates to fixedpoint so the report keeps its bare-decimal rendering.
func (value Instructions_Half_Percent) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint so the report's bare decimal round-trips.
func (value *Instructions_Half_Percent) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Instructions_Significant reports whether the retired-instruction interval clears the ±1% band.
type Instructions_Significant bool

// Instructions_Significant_Invariants witnesses both verdicts of significance: the
// per-metric verdicts are the family's only funded coverage.
func Instructions_Significant_Invariants(identifier string, value Instructions_Significant) {
	invariant.Sometimes(identifier, bool(value),
		"The instructions delta clears the significance band.")
}

// Instructions_Faster reports whether the retired-instruction mean sits below the reference's.
type Instructions_Faster bool

// Instructions_Faster_Invariants witnesses both directions of the mean comparison: the
// per-metric verdicts are the family's only funded coverage.
func Instructions_Faster_Invariants(identifier string, value Instructions_Faster) {
	invariant.Sometimes(identifier, bool(value),
		"The instructions candidate retires fewer instructions than the reference.")
}

// Instructions_Delta is the retired-instruction change against the reference.
type Instructions_Delta struct {
	// Diff_Percent is the candidate's mean as a signed percentage of the reference's.
	Diff_Percent Instructions_Diff_Percent `json:"diff_percent"`
	// Half_Percent is the half-width of the 95% confidence interval on Diff_Percent.
	Half_Percent Instructions_Half_Percent `json:"half_percent"`
	// Significant is true only when the interval clears the ±1% band.
	Significant Instructions_Significant `json:"significant"`
	// Faster is true when the candidate's mean is below the reference's.
	Faster Instructions_Faster `json:"faster"`
}

// Instructions_Delta_Invariants composes the field invariants of the retired-instruction change.
func Instructions_Delta_Invariants(identifier string, value Instructions_Delta) {
	Instructions_Diff_Percent_Invariants(identifier, value.Diff_Percent)
	Instructions_Half_Percent_Invariants(identifier, value.Half_Percent)
	Instructions_Significant_Invariants(identifier, value.Significant)
	Instructions_Faster_Invariants(identifier, value.Faster)
}

// As_Delta converts field-by-field to the plain Delta.
// A method rather than a free function so the conversion plants no invariant roots.
func (value Instructions_Delta) As_Delta() (delta Delta) {
	delta.Diff_Percent = Diff_Percent(value.Diff_Percent)
	delta.Half_Percent = Half_Percent(value.Half_Percent)
	delta.Significant = Significant(value.Significant)
	delta.Faster = Faster(value.Faster)
	return delta
}

// Cache_References_Diff_Percent is the cache-reference relative difference, fixedpoint-scaled.
type Cache_References_Diff_Percent int64

// Cache_References_Diff_Percent_Invariants states the floor alone: the old framework demanded no
// witness of this position, so none is funded.
func Cache_References_Diff_Percent_Invariants(
	identifier string, value Cache_References_Diff_Percent,
) {
	invariant.Always(int64(value) >= DIFF_PERCENT_MIN,
		"The cache references diff percent never falls below negative one hundred percent.")
}

// MarshalJSON delegates to fixedpoint so the report keeps its bare-decimal rendering.
func (value Cache_References_Diff_Percent) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint so the report's bare decimal round-trips.
func (value *Cache_References_Diff_Percent) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Cache_References_Half_Percent is the cache-reference confidence half-interval, fixedpoint-scaled.
type Cache_References_Half_Percent int64

// Cache_References_Half_Percent_Invariants states non-negativity alone: the old framework demanded
// no witness of this position, so none is funded.
func Cache_References_Half_Percent_Invariants(
	identifier string, value Cache_References_Half_Percent,
) {
	invariant.Always(int64(value) >= 0,
		"The cache references half percent half interval is never negative.")
}

// MarshalJSON delegates to fixedpoint so the report keeps its bare-decimal rendering.
func (value Cache_References_Half_Percent) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint so the report's bare decimal round-trips.
func (value *Cache_References_Half_Percent) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Cache_References_Significant reports whether the cache-reference interval clears the ±1% band.
type Cache_References_Significant bool

// Cache_References_Significant_Invariants witnesses both verdicts of significance: the
// per-metric verdicts are the family's only funded coverage.
func Cache_References_Significant_Invariants(
	identifier string, value Cache_References_Significant,
) {
	invariant.Sometimes(identifier, bool(value),
		"The cache references delta clears the significance band.")
}

// Cache_References_Faster reports whether the cache-reference mean sits below the reference's.
type Cache_References_Faster bool

// Cache_References_Faster_Invariants witnesses both directions of the mean comparison: the
// per-metric verdicts are the family's only funded coverage.
func Cache_References_Faster_Invariants(identifier string, value Cache_References_Faster) {
	invariant.Sometimes(identifier, bool(value),
		"The cache references candidate touches the cache less than the reference.")
}

// Cache_References_Delta is the cache-reference change against the reference.
type Cache_References_Delta struct {
	// Diff_Percent is the candidate's mean as a signed percentage of the reference's.
	Diff_Percent Cache_References_Diff_Percent `json:"diff_percent"`
	// Half_Percent is the half-width of the 95% confidence interval on Diff_Percent.
	Half_Percent Cache_References_Half_Percent `json:"half_percent"`
	// Significant is true only when the interval clears the ±1% band.
	Significant Cache_References_Significant `json:"significant"`
	// Faster is true when the candidate's mean is below the reference's.
	Faster Cache_References_Faster `json:"faster"`
}

// Cache_References_Delta_Invariants composes the field invariants of the cache-reference change.
func Cache_References_Delta_Invariants(identifier string, value Cache_References_Delta) {
	Cache_References_Diff_Percent_Invariants(identifier, value.Diff_Percent)
	Cache_References_Half_Percent_Invariants(identifier, value.Half_Percent)
	Cache_References_Significant_Invariants(identifier, value.Significant)
	Cache_References_Faster_Invariants(identifier, value.Faster)
}

// As_Delta converts field-by-field to the plain Delta.
// A method rather than a free function so the conversion plants no invariant roots.
func (value Cache_References_Delta) As_Delta() (delta Delta) {
	delta.Diff_Percent = Diff_Percent(value.Diff_Percent)
	delta.Half_Percent = Half_Percent(value.Half_Percent)
	delta.Significant = Significant(value.Significant)
	delta.Faster = Faster(value.Faster)
	return delta
}

// Cache_Misses_Diff_Percent is the cache-miss relative difference, fixedpoint-scaled.
type Cache_Misses_Diff_Percent int64

// Cache_Misses_Diff_Percent_Invariants states the floor alone: the old framework demanded no
// witness of this position, so none is funded.
func Cache_Misses_Diff_Percent_Invariants(identifier string, value Cache_Misses_Diff_Percent) {
	invariant.Always(int64(value) >= DIFF_PERCENT_MIN,
		"The cache misses diff percent never falls below negative one hundred percent.")
}

// MarshalJSON delegates to fixedpoint so the report keeps its bare-decimal rendering.
func (value Cache_Misses_Diff_Percent) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint so the report's bare decimal round-trips.
func (value *Cache_Misses_Diff_Percent) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Cache_Misses_Half_Percent is the cache-miss confidence half-interval, fixedpoint-scaled.
type Cache_Misses_Half_Percent int64

// Cache_Misses_Half_Percent_Invariants states non-negativity alone: the old framework demanded
// no witness of this position, so none is funded.
func Cache_Misses_Half_Percent_Invariants(identifier string, value Cache_Misses_Half_Percent) {
	invariant.Always(int64(value) >= 0,
		"The cache misses half percent half interval is never negative.")
}

// MarshalJSON delegates to fixedpoint so the report keeps its bare-decimal rendering.
func (value Cache_Misses_Half_Percent) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint so the report's bare decimal round-trips.
func (value *Cache_Misses_Half_Percent) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Cache_Misses_Significant reports whether the cache-miss interval clears the ±1% band.
type Cache_Misses_Significant bool

// Cache_Misses_Significant_Invariants witnesses both verdicts of significance: the
// per-metric verdicts are the family's only funded coverage.
func Cache_Misses_Significant_Invariants(identifier string, value Cache_Misses_Significant) {
	invariant.Sometimes(identifier, bool(value),
		"The cache misses delta clears the significance band.")
}

// Cache_Misses_Faster reports whether the cache-miss mean sits below the reference's.
type Cache_Misses_Faster bool

// Cache_Misses_Faster_Invariants witnesses both directions of the mean comparison: the
// per-metric verdicts are the family's only funded coverage.
func Cache_Misses_Faster_Invariants(identifier string, value Cache_Misses_Faster) {
	invariant.Sometimes(identifier, bool(value),
		"The cache misses candidate misses the cache less than the reference.")
}

// Cache_Misses_Delta is the cache-miss change against the reference.
type Cache_Misses_Delta struct {
	// Diff_Percent is the candidate's mean as a signed percentage of the reference's.
	Diff_Percent Cache_Misses_Diff_Percent `json:"diff_percent"`
	// Half_Percent is the half-width of the 95% confidence interval on Diff_Percent.
	Half_Percent Cache_Misses_Half_Percent `json:"half_percent"`
	// Significant is true only when the interval clears the ±1% band.
	Significant Cache_Misses_Significant `json:"significant"`
	// Faster is true when the candidate's mean is below the reference's.
	Faster Cache_Misses_Faster `json:"faster"`
}

// Cache_Misses_Delta_Invariants composes the field invariants of the cache-miss change.
func Cache_Misses_Delta_Invariants(identifier string, value Cache_Misses_Delta) {
	Cache_Misses_Diff_Percent_Invariants(identifier, value.Diff_Percent)
	Cache_Misses_Half_Percent_Invariants(identifier, value.Half_Percent)
	Cache_Misses_Significant_Invariants(identifier, value.Significant)
	Cache_Misses_Faster_Invariants(identifier, value.Faster)
}

// As_Delta converts field-by-field to the plain Delta.
// A method rather than a free function so the conversion plants no invariant roots.
func (value Cache_Misses_Delta) As_Delta() (delta Delta) {
	delta.Diff_Percent = Diff_Percent(value.Diff_Percent)
	delta.Half_Percent = Half_Percent(value.Half_Percent)
	delta.Significant = Significant(value.Significant)
	delta.Faster = Faster(value.Faster)
	return delta
}

// Branch_Misses_Diff_Percent is the branch-miss relative difference, fixedpoint-scaled.
type Branch_Misses_Diff_Percent int64

// Branch_Misses_Diff_Percent_Invariants states the floor alone: the old framework demanded no
// witness of this position, so none is funded.
func Branch_Misses_Diff_Percent_Invariants(identifier string, value Branch_Misses_Diff_Percent) {
	invariant.Always(int64(value) >= DIFF_PERCENT_MIN,
		"The branch misses diff percent never falls below negative one hundred percent.")
}

// MarshalJSON delegates to fixedpoint so the report keeps its bare-decimal rendering.
func (value Branch_Misses_Diff_Percent) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint so the report's bare decimal round-trips.
func (value *Branch_Misses_Diff_Percent) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Branch_Misses_Half_Percent is the branch-miss confidence half-interval, fixedpoint-scaled.
type Branch_Misses_Half_Percent int64

// Branch_Misses_Half_Percent_Invariants states non-negativity alone: the old framework demanded
// no witness of this position, so none is funded.
func Branch_Misses_Half_Percent_Invariants(identifier string, value Branch_Misses_Half_Percent) {
	invariant.Always(int64(value) >= 0,
		"The branch misses half percent half interval is never negative.")
}

// MarshalJSON delegates to fixedpoint so the report keeps its bare-decimal rendering.
func (value Branch_Misses_Half_Percent) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint so the report's bare decimal round-trips.
func (value *Branch_Misses_Half_Percent) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// Branch_Misses_Significant reports whether the branch-miss interval clears the ±1% band.
type Branch_Misses_Significant bool

// Branch_Misses_Significant_Invariants witnesses both verdicts of significance: the
// per-metric verdicts are the family's only funded coverage.
func Branch_Misses_Significant_Invariants(identifier string, value Branch_Misses_Significant) {
	invariant.Sometimes(identifier, bool(value),
		"The branch misses delta clears the significance band.")
}

// Branch_Misses_Faster reports whether the branch-miss mean sits below the reference's.
type Branch_Misses_Faster bool

// Branch_Misses_Faster_Invariants witnesses both directions of the mean comparison: the
// per-metric verdicts are the family's only funded coverage.
func Branch_Misses_Faster_Invariants(identifier string, value Branch_Misses_Faster) {
	invariant.Sometimes(identifier, bool(value),
		"The branch misses candidate mispredicts less than the reference.")
}

// Branch_Misses_Delta is the branch-miss change against the reference.
type Branch_Misses_Delta struct {
	// Diff_Percent is the candidate's mean as a signed percentage of the reference's.
	Diff_Percent Branch_Misses_Diff_Percent `json:"diff_percent"`
	// Half_Percent is the half-width of the 95% confidence interval on Diff_Percent.
	Half_Percent Branch_Misses_Half_Percent `json:"half_percent"`
	// Significant is true only when the interval clears the ±1% band.
	Significant Branch_Misses_Significant `json:"significant"`
	// Faster is true when the candidate's mean is below the reference's.
	Faster Branch_Misses_Faster `json:"faster"`
}

// Branch_Misses_Delta_Invariants composes the field invariants of the branch-miss change.
func Branch_Misses_Delta_Invariants(identifier string, value Branch_Misses_Delta) {
	Branch_Misses_Diff_Percent_Invariants(identifier, value.Diff_Percent)
	Branch_Misses_Half_Percent_Invariants(identifier, value.Half_Percent)
	Branch_Misses_Significant_Invariants(identifier, value.Significant)
	Branch_Misses_Faster_Invariants(identifier, value.Faster)
}

// As_Delta converts field-by-field to the plain Delta.
// A method rather than a free function so the conversion plants no invariant roots.
func (value Branch_Misses_Delta) As_Delta() (delta Delta) {
	delta.Diff_Percent = Diff_Percent(value.Diff_Percent)
	delta.Half_Percent = Half_Percent(value.Half_Percent)
	delta.Significant = Significant(value.Significant)
	delta.Faster = Faster(value.Faster)
	return delta
}

// CPU_User_Diff_Percent is the user-CPU-time relative difference, fixedpoint-scaled.
type CPU_User_Diff_Percent int64

// CPU_User_Diff_Percent_Invariants states the floor alone: the old framework demanded no
// witness of this position, so none is funded.
func CPU_User_Diff_Percent_Invariants(identifier string, value CPU_User_Diff_Percent) {
	invariant.Always(int64(value) >= DIFF_PERCENT_MIN,
		"The cpu user diff percent never falls below negative one hundred percent.")
}

// MarshalJSON delegates to fixedpoint so the report keeps its bare-decimal rendering.
func (value CPU_User_Diff_Percent) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint so the report's bare decimal round-trips.
func (value *CPU_User_Diff_Percent) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_User_Half_Percent is the user-CPU-time confidence half-interval, fixedpoint-scaled.
type CPU_User_Half_Percent int64

// CPU_User_Half_Percent_Invariants states non-negativity alone: the old framework demanded
// no witness of this position, so none is funded.
func CPU_User_Half_Percent_Invariants(identifier string, value CPU_User_Half_Percent) {
	invariant.Always(int64(value) >= 0,
		"The cpu user half percent half interval is never negative.")
}

// MarshalJSON delegates to fixedpoint so the report keeps its bare-decimal rendering.
func (value CPU_User_Half_Percent) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint so the report's bare decimal round-trips.
func (value *CPU_User_Half_Percent) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_User_Significant reports whether the user-CPU-time interval clears the ±1% band.
type CPU_User_Significant bool

// CPU_User_Significant_Invariants witnesses both verdicts of significance: the
// per-metric verdicts are the family's only funded coverage.
func CPU_User_Significant_Invariants(identifier string, value CPU_User_Significant) {
	invariant.Sometimes(identifier, bool(value),
		"The cpu user delta clears the significance band.")
}

// CPU_User_Faster reports whether the user-CPU-time mean sits below the reference's.
type CPU_User_Faster bool

// CPU_User_Faster_Invariants witnesses both directions of the mean comparison: the
// per-metric verdicts are the family's only funded coverage.
func CPU_User_Faster_Invariants(identifier string, value CPU_User_Faster) {
	invariant.Sometimes(identifier, bool(value),
		"The cpu user candidate spends less user time than the reference.")
}

// CPU_User_Delta is the user-CPU-time change against the reference.
type CPU_User_Delta struct {
	// Diff_Percent is the candidate's mean as a signed percentage of the reference's.
	Diff_Percent CPU_User_Diff_Percent `json:"diff_percent"`
	// Half_Percent is the half-width of the 95% confidence interval on Diff_Percent.
	Half_Percent CPU_User_Half_Percent `json:"half_percent"`
	// Significant is true only when the interval clears the ±1% band.
	Significant CPU_User_Significant `json:"significant"`
	// Faster is true when the candidate's mean is below the reference's.
	Faster CPU_User_Faster `json:"faster"`
}

// CPU_User_Delta_Invariants composes the field invariants of the user-CPU-time change.
func CPU_User_Delta_Invariants(identifier string, value CPU_User_Delta) {
	CPU_User_Diff_Percent_Invariants(identifier, value.Diff_Percent)
	CPU_User_Half_Percent_Invariants(identifier, value.Half_Percent)
	CPU_User_Significant_Invariants(identifier, value.Significant)
	CPU_User_Faster_Invariants(identifier, value.Faster)
}

// As_Delta converts field-by-field to the plain Delta.
// A method rather than a free function so the conversion plants no invariant roots.
func (value CPU_User_Delta) As_Delta() (delta Delta) {
	delta.Diff_Percent = Diff_Percent(value.Diff_Percent)
	delta.Half_Percent = Half_Percent(value.Half_Percent)
	delta.Significant = Significant(value.Significant)
	delta.Faster = Faster(value.Faster)
	return delta
}

// CPU_System_Diff_Percent is the system-CPU-time relative difference, fixedpoint-scaled.
type CPU_System_Diff_Percent int64

// CPU_System_Diff_Percent_Invariants states the floor alone: the old framework demanded no
// witness of this position, so none is funded.
func CPU_System_Diff_Percent_Invariants(identifier string, value CPU_System_Diff_Percent) {
	invariant.Always(int64(value) >= DIFF_PERCENT_MIN,
		"The cpu system diff percent never falls below negative one hundred percent.")
}

// MarshalJSON delegates to fixedpoint so the report keeps its bare-decimal rendering.
func (value CPU_System_Diff_Percent) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint so the report's bare decimal round-trips.
func (value *CPU_System_Diff_Percent) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_System_Half_Percent is the system-CPU-time confidence half-interval, fixedpoint-scaled.
type CPU_System_Half_Percent int64

// CPU_System_Half_Percent_Invariants states non-negativity alone: the old framework demanded
// no witness of this position, so none is funded.
func CPU_System_Half_Percent_Invariants(identifier string, value CPU_System_Half_Percent) {
	invariant.Always(int64(value) >= 0,
		"The cpu system half percent half interval is never negative.")
}

// MarshalJSON delegates to fixedpoint so the report keeps its bare-decimal rendering.
func (value CPU_System_Half_Percent) MarshalJSON() (data []byte, err error) {
	return fixedpoint.Number(value).MarshalJSON()
}

// UnmarshalJSON delegates to fixedpoint so the report's bare decimal round-trips.
func (value *CPU_System_Half_Percent) UnmarshalJSON(data []byte) (err error) {
	return (*fixedpoint.Number)(value).UnmarshalJSON(data)
}

// CPU_System_Significant reports whether the system-CPU-time interval clears the ±1% band.
type CPU_System_Significant bool

// CPU_System_Significant_Invariants witnesses both verdicts of significance: the
// per-metric verdicts are the family's only funded coverage.
func CPU_System_Significant_Invariants(identifier string, value CPU_System_Significant) {
	invariant.Sometimes(identifier, bool(value),
		"The cpu system delta clears the significance band.")
}

// CPU_System_Faster reports whether the system-CPU-time mean sits below the reference's.
type CPU_System_Faster bool

// CPU_System_Faster_Invariants witnesses both directions of the mean comparison: the
// per-metric verdicts are the family's only funded coverage.
func CPU_System_Faster_Invariants(identifier string, value CPU_System_Faster) {
	invariant.Sometimes(identifier, bool(value),
		"The cpu system candidate spends less kernel time than the reference.")
}

// CPU_System_Delta is the system-CPU-time change against the reference.
type CPU_System_Delta struct {
	// Diff_Percent is the candidate's mean as a signed percentage of the reference's.
	Diff_Percent CPU_System_Diff_Percent `json:"diff_percent"`
	// Half_Percent is the half-width of the 95% confidence interval on Diff_Percent.
	Half_Percent CPU_System_Half_Percent `json:"half_percent"`
	// Significant is true only when the interval clears the ±1% band.
	Significant CPU_System_Significant `json:"significant"`
	// Faster is true when the candidate's mean is below the reference's.
	Faster CPU_System_Faster `json:"faster"`
}

// CPU_System_Delta_Invariants composes the field invariants of the system-CPU-time change.
func CPU_System_Delta_Invariants(identifier string, value CPU_System_Delta) {
	CPU_System_Diff_Percent_Invariants(identifier, value.Diff_Percent)
	CPU_System_Half_Percent_Invariants(identifier, value.Half_Percent)
	CPU_System_Significant_Invariants(identifier, value.Significant)
	CPU_System_Faster_Invariants(identifier, value.Faster)
}

// As_Delta converts field-by-field to the plain Delta.
// A method rather than a free function so the conversion plants no invariant roots.
func (value CPU_System_Delta) As_Delta() (delta Delta) {
	delta.Diff_Percent = Diff_Percent(value.Diff_Percent)
	delta.Half_Percent = Half_Percent(value.Half_Percent)
	delta.Significant = Significant(value.Significant)
	delta.Faster = Faster(value.Faster)
	return delta
}

// Deltas is every metric's change for one command relative to the reference, in the
// same order as Measurements.
type Deltas struct {
	// Wall_Time is the elapsed-time delta.
	Wall_Time Wall_Time_Delta `json:"wall_time"`
	// Peak_RSS is the peak-memory delta.
	Peak_RSS Peak_RSS_Delta `json:"peak_rss"`
	// CPU_Cycles is the CPU-cycle delta.
	CPU_Cycles CPU_Cycles_Delta `json:"cpu_cycles"`
	// Instructions is the retired-instruction delta.
	Instructions Instructions_Delta `json:"instructions"`
	// Cache_References is the cache-reference delta (Linux only).
	Cache_References Cache_References_Delta `json:"cache_references"`
	// Cache_Misses is the cache-miss delta (Linux only).
	Cache_Misses Cache_Misses_Delta `json:"cache_misses"`
	// Branch_Misses is the branch-miss delta (Linux only).
	Branch_Misses Branch_Misses_Delta `json:"branch_misses"`
	// CPU_User is the user-CPU-time delta.
	CPU_User CPU_User_Delta `json:"cpu_user"`
	// CPU_System is the system-CPU-time delta.
	CPU_System CPU_System_Delta `json:"cpu_system"`
}

// Deltas_Invariants composes the change of every metric field.
func Deltas_Invariants(identifier string, deltas Deltas) {
	Wall_Time_Delta_Invariants(identifier, deltas.Wall_Time)
	Peak_RSS_Delta_Invariants(identifier, deltas.Peak_RSS)
	CPU_Cycles_Delta_Invariants(identifier, deltas.CPU_Cycles)
	Instructions_Delta_Invariants(identifier, deltas.Instructions)
	Cache_References_Delta_Invariants(identifier, deltas.Cache_References)
	Cache_Misses_Delta_Invariants(identifier, deltas.Cache_Misses)
	Branch_Misses_Delta_Invariants(identifier, deltas.Branch_Misses)
	CPU_User_Delta_Invariants(identifier, deltas.CPU_User)
	CPU_System_Delta_Invariants(identifier, deltas.CPU_System)
}

// As_Wall_Time_Delta converts field-by-field to the elapsed-time variant.
// A method rather than a free function so the conversion plants no invariant roots.
func (delta Delta) As_Wall_Time_Delta() (value Wall_Time_Delta) {
	value.Diff_Percent = Wall_Time_Diff_Percent(delta.Diff_Percent)
	value.Half_Percent = Wall_Time_Half_Percent(delta.Half_Percent)
	value.Significant = Wall_Time_Significant(delta.Significant)
	value.Faster = Wall_Time_Faster(delta.Faster)
	return value
}

// As_Peak_RSS_Delta converts field-by-field to the peak-memory variant.
// A method rather than a free function so the conversion plants no invariant roots.
func (delta Delta) As_Peak_RSS_Delta() (value Peak_RSS_Delta) {
	value.Diff_Percent = Peak_RSS_Diff_Percent(delta.Diff_Percent)
	value.Half_Percent = Peak_RSS_Half_Percent(delta.Half_Percent)
	value.Significant = Peak_RSS_Significant(delta.Significant)
	value.Faster = Peak_RSS_Faster(delta.Faster)
	return value
}

// As_CPU_Cycles_Delta converts field-by-field to the CPU-cycle variant.
// A method rather than a free function so the conversion plants no invariant roots.
func (delta Delta) As_CPU_Cycles_Delta() (value CPU_Cycles_Delta) {
	value.Diff_Percent = CPU_Cycles_Diff_Percent(delta.Diff_Percent)
	value.Half_Percent = CPU_Cycles_Half_Percent(delta.Half_Percent)
	value.Significant = CPU_Cycles_Significant(delta.Significant)
	value.Faster = CPU_Cycles_Faster(delta.Faster)
	return value
}

// As_Instructions_Delta converts field-by-field to the retired-instruction variant.
// A method rather than a free function so the conversion plants no invariant roots.
func (delta Delta) As_Instructions_Delta() (value Instructions_Delta) {
	value.Diff_Percent = Instructions_Diff_Percent(delta.Diff_Percent)
	value.Half_Percent = Instructions_Half_Percent(delta.Half_Percent)
	value.Significant = Instructions_Significant(delta.Significant)
	value.Faster = Instructions_Faster(delta.Faster)
	return value
}

// As_Cache_References_Delta converts field-by-field to the cache-reference variant.
// A method rather than a free function so the conversion plants no invariant roots.
func (delta Delta) As_Cache_References_Delta() (value Cache_References_Delta) {
	value.Diff_Percent = Cache_References_Diff_Percent(delta.Diff_Percent)
	value.Half_Percent = Cache_References_Half_Percent(delta.Half_Percent)
	value.Significant = Cache_References_Significant(delta.Significant)
	value.Faster = Cache_References_Faster(delta.Faster)
	return value
}

// As_Cache_Misses_Delta converts field-by-field to the cache-miss variant.
// A method rather than a free function so the conversion plants no invariant roots.
func (delta Delta) As_Cache_Misses_Delta() (value Cache_Misses_Delta) {
	value.Diff_Percent = Cache_Misses_Diff_Percent(delta.Diff_Percent)
	value.Half_Percent = Cache_Misses_Half_Percent(delta.Half_Percent)
	value.Significant = Cache_Misses_Significant(delta.Significant)
	value.Faster = Cache_Misses_Faster(delta.Faster)
	return value
}

// As_Branch_Misses_Delta converts field-by-field to the branch-miss variant.
// A method rather than a free function so the conversion plants no invariant roots.
func (delta Delta) As_Branch_Misses_Delta() (value Branch_Misses_Delta) {
	value.Diff_Percent = Branch_Misses_Diff_Percent(delta.Diff_Percent)
	value.Half_Percent = Branch_Misses_Half_Percent(delta.Half_Percent)
	value.Significant = Branch_Misses_Significant(delta.Significant)
	value.Faster = Branch_Misses_Faster(delta.Faster)
	return value
}

// As_CPU_User_Delta converts field-by-field to the user-CPU-time variant.
// A method rather than a free function so the conversion plants no invariant roots.
func (delta Delta) As_CPU_User_Delta() (value CPU_User_Delta) {
	value.Diff_Percent = CPU_User_Diff_Percent(delta.Diff_Percent)
	value.Half_Percent = CPU_User_Half_Percent(delta.Half_Percent)
	value.Significant = CPU_User_Significant(delta.Significant)
	value.Faster = CPU_User_Faster(delta.Faster)
	return value
}

// As_CPU_System_Delta converts field-by-field to the system-CPU-time variant.
// A method rather than a free function so the conversion plants no invariant roots.
func (delta Delta) As_CPU_System_Delta() (value CPU_System_Delta) {
	value.Diff_Percent = CPU_System_Diff_Percent(delta.Diff_Percent)
	value.Half_Percent = CPU_System_Half_Percent(delta.Half_Percent)
	value.Significant = CPU_System_Significant(delta.Significant)
	value.Faster = CPU_System_Faster(delta.Faster)
	return value
}
