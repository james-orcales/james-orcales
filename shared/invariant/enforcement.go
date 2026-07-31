//go:build !noassert

// The enforcing build keeps fluent links observationally silent: they only advance a value and
// latch raw verdicts. Ensure is the single boundary that can panic or mutate coverage.
package invariant

import "unsafe"

// ASSERTION_FAILURE_NONE reserves zero so the builder's zero value has no deferred verdict.
const ASSERTION_FAILURE_NONE uint8 = 0

// ASSERTION_FAILURE_LINKS defers expanded-cap enforcement to Ensure.
const ASSERTION_FAILURE_LINKS uint8 = 1

// ASSERTION_FAILURE_RANGE_DOMAIN distinguishes a malformed interval from an observed violation.
const ASSERTION_FAILURE_RANGE_DOMAIN uint8 = 2

// ASSERTION_FAILURE_RANGE_LOWER identifies the lower guard without formatting on the passing path.
const ASSERTION_FAILURE_RANGE_LOWER uint8 = 3

// ASSERTION_FAILURE_RANGE_UPPER identifies the upper guard without formatting on the passing path.
const ASSERTION_FAILURE_RANGE_UPPER uint8 = 4

// ASSERTION_FAILURE_RANGE_EXCLUSION identifies a malformed boundary or exterior hole.
const ASSERTION_FAILURE_RANGE_EXCLUSION uint8 = 5

// ASSERTION_FAILURE_RANGE_EXCLUDED identifies an observed legal hole.
const ASSERTION_FAILURE_RANGE_EXCLUDED uint8 = 6

// ASSERTION_FAILURE_ENUM_DOMAIN distinguishes a malformed member set.
const ASSERTION_FAILURE_ENUM_DOMAIN uint8 = 7

// ASSERTION_FAILURE_ENUM_MEMBER identifies an observed non-member.
const ASSERTION_FAILURE_ENUM_MEMBER uint8 = 8

// ASSERTION_RECORDING_MASK tags the union without colliding with any packed recording field.
const ASSERTION_RECORDING_MASK = uintptr(1) << (unsafe.Sizeof(uintptr(0))*8 - 1)

// ASSERTION_ORDINAL_SHIFT leaves the low byte available for the six observations above ordinal 63.
const ASSERTION_ORDINAL_SHIFT = 8

// ASSERTION_ORDINAL_MASK reserves seven bits because registration caps ordinals at 70.
const ASSERTION_ORDINAL_MASK = uintptr(0x7f) << ASSERTION_ORDINAL_SHIFT

// ASSERTION_FAILURE_SHIFT keeps the deferred verdict independent of observations and ordinals.
const ASSERTION_FAILURE_SHIFT = 16

// ASSERTION_FAILURE_MASK reserves four bits for the nine failure states.
const ASSERTION_FAILURE_MASK = uintptr(0x0f) << ASSERTION_FAILURE_SHIFT

// Recorder_Always remains eager because it is deliberately outside the deferred builder.
func Recorder_Always[T ~bool](recorder *Recorder, condition T, message string) {
	if !condition {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX + message + "  Always — condition was false")
	}
	if !recorder.Is_Test {
		return
	}
	if recorder.Is_Benchmark {
		return
	}
	recorder_increment(recorder, message, true)
}

// Recorder_Assertions starts one deferred chain. Only recording modes consult registration;
// shipped enforcement therefore pays no map, cache, lock, or identity-validation cost.
func Recorder_Assertions(recorder *Recorder, namespace Namespace) (builder Assertion_Builder) {
	builder.Context = unsafe.Pointer(unsafe.StringData(string(namespace)))
	builder.State_A = uintptr(len(namespace))
	if !recorder.Is_Test {
		return builder
	}
	if recorder.Is_Benchmark {
		return builder
	}
	plan := recorder.Assertion_Plans[namespace]
	if plan != nil {
		builder.Context = unsafe.Pointer(plan)
		builder.State_A = 0
		builder.State_B = ASSERTION_RECORDING_MASK
	}
	return builder
}

// Sometimes captures one branch by expanded ordinal. The message belongs exclusively to the
// registration plan and is intentionally unread here, preventing runtime identity reconstruction.
func (builder Assertion_Builder) Sometimes(
	condition bool, message string,
) (next Assertion_Builder) {
	if !builder.assertion_recording() {
		return builder
	}
	return builder.assertion_axis(condition)
}

// Range_Int keeps only value-dependent enforcement in ordinary binaries.
func (builder Assertion_Builder) Range_Int(
	value int, minimum int, maximum int, excluded ...int,
) (next Assertion_Builder) {
	valid := value >= minimum && value <= maximum
	for _, hole := range excluded {
		if value == hole {
			valid = false
			break
		}
	}
	if valid {
		if !builder.assertion_recording() {
			return builder
		}
	}
	return assertion_range_slow[int](
		builder, value, minimum, maximum,
		unsafe.Pointer(unsafe.SliceData(excluded)), len(excluded))
}

// Range_Int8 keeps only value-dependent enforcement in ordinary binaries.
func (builder Assertion_Builder) Range_Int8(
	value int8, minimum int8, maximum int8, excluded ...int8,
) (next Assertion_Builder) {
	return assertion_range_head(builder, value, minimum, maximum, excluded)
}

// Range_Int16 keeps only value-dependent enforcement in ordinary binaries.
func (builder Assertion_Builder) Range_Int16(
	value int16, minimum int16, maximum int16, excluded ...int16,
) (next Assertion_Builder) {
	return assertion_range_head(builder, value, minimum, maximum, excluded)
}

// Range_Int32 keeps only value-dependent enforcement in ordinary binaries.
func (builder Assertion_Builder) Range_Int32(
	value int32, minimum int32, maximum int32, excluded ...int32,
) (next Assertion_Builder) {
	return assertion_range_head(builder, value, minimum, maximum, excluded)
}

// Range_Int64 keeps only value-dependent enforcement in ordinary binaries.
func (builder Assertion_Builder) Range_Int64(
	value int64, minimum int64, maximum int64, excluded ...int64,
) (next Assertion_Builder) {
	return assertion_range_head(builder, value, minimum, maximum, excluded)
}

// Range_Uint keeps only value-dependent enforcement in ordinary binaries.
func (builder Assertion_Builder) Range_Uint(
	value uint, minimum uint, maximum uint, excluded ...uint,
) (next Assertion_Builder) {
	return assertion_range_head(builder, value, minimum, maximum, excluded)
}

// Range_Uint8 keeps only value-dependent enforcement in ordinary binaries.
func (builder Assertion_Builder) Range_Uint8(
	value uint8, minimum uint8, maximum uint8, excluded ...uint8,
) (next Assertion_Builder) {
	valid := value >= minimum && value <= maximum
	for _, hole := range excluded {
		if value == hole {
			valid = false
			break
		}
	}
	if valid {
		if !builder.assertion_recording() {
			return builder
		}
	}
	return assertion_range_slow[uint8](
		builder, value, minimum, maximum,
		unsafe.Pointer(unsafe.SliceData(excluded)), len(excluded))
}

// Range_Uint16 keeps only value-dependent enforcement in ordinary binaries.
func (builder Assertion_Builder) Range_Uint16(
	value uint16, minimum uint16, maximum uint16, excluded ...uint16,
) (next Assertion_Builder) {
	return assertion_range_head(builder, value, minimum, maximum, excluded)
}

// Range_Uint32 keeps only value-dependent enforcement in ordinary binaries.
func (builder Assertion_Builder) Range_Uint32(
	value uint32, minimum uint32, maximum uint32, excluded ...uint32,
) (next Assertion_Builder) {
	return assertion_range_head(builder, value, minimum, maximum, excluded)
}

// Range_Uint64 keeps only value-dependent enforcement in ordinary binaries.
func (builder Assertion_Builder) Range_Uint64(
	value uint64, minimum uint64, maximum uint64, excluded ...uint64,
) (next Assertion_Builder) {
	return assertion_range_head(builder, value, minimum, maximum, excluded)
}

// Enum_Int keeps only membership enforcement in ordinary binaries.
//
//go:noinline
func (builder Assertion_Builder) Enum_Int(
	value int, members ...int,
) (next Assertion_Builder) {
	return assertion_enum_head(builder, value, members, assertion_enum_slow[int])
}

// Enum_Int8 keeps only membership enforcement in ordinary binaries.
//
//go:noinline
func (builder Assertion_Builder) Enum_Int8(
	value int8, members ...int8,
) (next Assertion_Builder) {
	return assertion_enum_head(builder, value, members, assertion_enum_slow[int8])
}

// Enum_Int16 keeps only membership enforcement in ordinary binaries.
//
//go:noinline
func (builder Assertion_Builder) Enum_Int16(
	value int16, members ...int16,
) (next Assertion_Builder) {
	return assertion_enum_head(builder, value, members, assertion_enum_slow[int16])
}

// Enum_Int32 keeps only membership enforcement in ordinary binaries.
//
//go:noinline
func (builder Assertion_Builder) Enum_Int32(
	value int32, members ...int32,
) (next Assertion_Builder) {
	return assertion_enum_head(builder, value, members, assertion_enum_slow[int32])
}

// Enum_Int64 keeps only membership enforcement in ordinary binaries.
//
//go:noinline
func (builder Assertion_Builder) Enum_Int64(
	value int64, members ...int64,
) (next Assertion_Builder) {
	return assertion_enum_head(builder, value, members, assertion_enum_slow[int64])
}

// Enum_Uint keeps only membership enforcement in ordinary binaries.
//
//go:noinline
func (builder Assertion_Builder) Enum_Uint(
	value uint, members ...uint,
) (next Assertion_Builder) {
	return assertion_enum_head(builder, value, members, assertion_enum_slow[uint])
}

// Enum_Uint8 keeps only membership enforcement in ordinary binaries.
//
//go:noinline
func (builder Assertion_Builder) Enum_Uint8(
	value uint8, members ...uint8,
) (next Assertion_Builder) {
	return assertion_enum_head(builder, value, members, assertion_enum_slow[uint8])
}

// Enum_Uint16 keeps only membership enforcement in ordinary binaries.
//
//go:noinline
func (builder Assertion_Builder) Enum_Uint16(
	value uint16, members ...uint16,
) (next Assertion_Builder) {
	return assertion_enum_head(builder, value, members, assertion_enum_slow[uint16])
}

// Enum_Uint32 keeps only membership enforcement in ordinary binaries.
//
//go:noinline
func (builder Assertion_Builder) Enum_Uint32(
	value uint32, members ...uint32,
) (next Assertion_Builder) {
	return assertion_enum_head(builder, value, members, assertion_enum_slow[uint32])
}

// Enum_Uint64 keeps only membership enforcement in ordinary binaries.
//
//go:noinline
func (builder Assertion_Builder) Enum_Uint64(
	value uint64, members ...uint64,
) (next Assertion_Builder) {
	return assertion_enum_head(builder, value, members, assertion_enum_slow[uint64])
}

// Ensure keeps the successful ordinary path small enough to inline at every assertion site.
func (builder Assertion_Builder) Ensure() {
	if builder.assertion_failure() == ASSERTION_FAILURE_NONE {
		if !builder.assertion_recording() {
			return
		}
	}
	assertion_ensure(&builder)
}

//go:noinline
func assertion_ensure(builder *Assertion_Builder) {
	if builder.assertion_failure() != ASSERTION_FAILURE_NONE {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX + builder.assertion_failure_message())
	}
	assertion_record(builder)
}

// Preflighting the whole plan in this cold boundary prevents an invalid execution from leaving
// misleading partial coverage.
//
//go:noinline
func assertion_record(builder *Assertion_Builder) {
	plan := builder.assertion_plan()
	if len(plan.Links) != int(builder.assertion_ordinal()) {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX +
			"registered Assertions chain differs from its registration plan")
	}
	for _, link := range plan.Links {
		if link.Entry.Metadata == nil {
			panic(ASSERTION_FAILURE_MESSAGE_PREFIX +
				"registered Assertions chain resolved an unknown coverage handle")
		}
	}
	for _, link := range plan.Links {
		condition := true
		if link.Kind == ASSERTION_KIND_SOMETIMES {
			condition = builder.assertion_observed(link.Ordinal)
		}
		recorder_increment_entry(plan.Recorder, link.Entry, condition)
	}
}

func (builder Assertion_Builder) assertion_axis(condition bool) (next Assertion_Builder) {
	ordinal := builder.assertion_ordinal()
	if condition {
		if ordinal < 64 {
			builder.State_A |= uintptr(1) << ordinal
		} else {
			builder.State_B |= uintptr(1) << (ordinal - 64)
		}
	}
	builder.State_B += uintptr(1) << ASSERTION_ORDINAL_SHIFT
	return builder
}

func (builder Assertion_Builder) assertion_guard() (next Assertion_Builder) {
	builder.State_B += uintptr(1) << ASSERTION_ORDINAL_SHIFT
	return builder
}

func (builder Assertion_Builder) assertion_observed(ordinal uint8) (observed bool) {
	if !builder.assertion_recording() {
		return false
	}
	if ordinal < 64 {
		return builder.State_A&(uintptr(1)<<ordinal) != 0
	}
	return builder.State_B&(uintptr(1)<<(ordinal-64)) != 0
}

func (builder Assertion_Builder) assertion_recording() (recording bool) {
	return builder.State_B&ASSERTION_RECORDING_MASK != 0
}

func (builder Assertion_Builder) assertion_ordinal() (ordinal uint8) {
	return uint8((builder.State_B & ASSERTION_ORDINAL_MASK) >> ASSERTION_ORDINAL_SHIFT)
}

func (builder Assertion_Builder) assertion_failure() (failure uint8) {
	return uint8((builder.State_B & ASSERTION_FAILURE_MASK) >> ASSERTION_FAILURE_SHIFT)
}

func (builder *Assertion_Builder) assertion_plan() (plan *Assertion_Plan) {
	return (*Assertion_Plan)(builder.Context)
}

func (builder *Assertion_Builder) assertion_namespace() (namespace Namespace) {
	if builder.assertion_recording() {
		return builder.assertion_plan().Namespace
	}
	if builder.State_A == 0 {
		return ""
	}
	return Namespace(unsafe.String((*byte)(builder.Context), int(builder.State_A)))
}

//go:noinline
func (builder Assertion_Builder) assertion_fail(failure uint8) (next Assertion_Builder) {
	if builder.assertion_failure() == ASSERTION_FAILURE_NONE {
		builder.State_B |= uintptr(failure) << ASSERTION_FAILURE_SHIFT
	}
	return builder
}

func (builder *Assertion_Builder) assertion_failure_message() (message string) {
	prefix := string(builder.assertion_namespace()) + ELEMENT_MESSAGE_SEPARATOR
	switch builder.assertion_failure() {
	case ASSERTION_FAILURE_LINKS:
		return "Assertions exceeds 70 links"
	case ASSERTION_FAILURE_RANGE_DOMAIN:
		return prefix + "Range minimum exceeds maximum"
	case ASSERTION_FAILURE_RANGE_LOWER:
		return prefix + RANGE_GUARD_MINIMUM + "  value below min"
	case ASSERTION_FAILURE_RANGE_UPPER:
		return prefix + RANGE_GUARD_MAXIMUM + "  value exceeds max"
	case ASSERTION_FAILURE_RANGE_EXCLUSION:
		return prefix + "Range exclusion is not strictly inside the interval"
	case ASSERTION_FAILURE_RANGE_EXCLUDED:
		return prefix + "Range value is excluded"
	case ASSERTION_FAILURE_ENUM_DOMAIN:
		return prefix + "Enum requires at least two distinct members"
	case ASSERTION_FAILURE_ENUM_MEMBER:
		return prefix + ENUM_GUARD_MEMBER + "  value is not a member"
	}
	return "Assertions failed"
}

func assertion_range_head[Value Integer](
	builder Assertion_Builder, value Value, minimum Value, maximum Value, excluded []Value,
) (next Assertion_Builder) {
	valid := value >= minimum && value <= maximum
	for _, hole := range excluded {
		if value == hole {
			valid = false
			break
		}
	}
	if valid {
		if !builder.assertion_recording() {
			return builder
		}
	}
	return assertion_range_slow(
		builder, value, minimum, maximum,
		unsafe.Pointer(unsafe.SliceData(excluded)), len(excluded))
}

//go:noinline
func assertion_range_slow[Value Integer](
	builder Assertion_Builder, value Value, minimum Value, maximum Value,
	excluded_data unsafe.Pointer, excluded_count int,
) (next Assertion_Builder) {
	excluded := unsafe.Slice((*Value)(excluded_data), excluded_count)
	if value < minimum {
		builder = builder.assertion_fail(ASSERTION_FAILURE_RANGE_LOWER)
	}
	if value > maximum {
		builder = builder.assertion_fail(ASSERTION_FAILURE_RANGE_UPPER)
	}
	for _, hole := range excluded {
		if value == hole {
			builder = builder.assertion_fail(ASSERTION_FAILURE_RANGE_EXCLUDED)
		}
	}
	if builder.assertion_recording() {
		builder = assertion_range_recording(builder, value, minimum, maximum, excluded)
	}
	return builder
}

// Only a registered test run needs boundary and sentinel observations.
//
//go:noinline
func assertion_range_recording[Value Integer](
	builder Assertion_Builder, value Value, minimum Value, maximum Value, excluded []Value,
) (next Assertion_Builder) {
	builder = builder.assertion_guard()
	builder = builder.assertion_guard()
	if minimum == maximum {
		return builder
	}
	builder = builder.assertion_axis(value == minimum)
	builder = builder.assertion_axis(value == maximum)
	builder = assertion_range_candidate_recording(
		builder, value, minimum, maximum, excluded, Value(0))
	builder = assertion_range_candidate_recording(
		builder, value, minimum, maximum, excluded, Value(1))
	builder = assertion_range_candidate_recording(
		builder, value, minimum, maximum, excluded, Value(2))
	zero := Value(0)
	negative_one := zero - Value(1)
	if negative_one < zero {
		builder = assertion_range_candidate_recording(
			builder, value, minimum, maximum, excluded, negative_one)
	}
	return builder
}

func assertion_range_candidate_recording[Value Integer](
	builder Assertion_Builder, value Value, minimum Value, maximum Value,
	excluded []Value, candidate Value,
) (next Assertion_Builder) {
	outside := candidate <= minimum
	if candidate >= maximum {
		outside = true
	}
	if outside {
		return builder
	}
	for _, hole := range excluded {
		if hole == candidate {
			return builder
		}
	}
	return builder.assertion_axis(value == candidate)
}

func assertion_enum_head[Value Integer](
	builder Assertion_Builder, value Value, members []Value,
	slow func(Assertion_Builder, Value, unsafe.Pointer, int) (next Assertion_Builder),
) (next Assertion_Builder) {
	matched := false
	for _, member := range members {
		if value == member {
			matched = true
			break
		}
	}
	if matched {
		if !builder.assertion_recording() {
			return builder
		}
	}
	return slow(
		builder, value, unsafe.Pointer(unsafe.SliceData(members)), len(members))
}

//go:noinline
func assertion_enum_slow[Value Integer](
	builder Assertion_Builder, value Value, member_data unsafe.Pointer, member_count int,
) (next Assertion_Builder) {
	members := unsafe.Slice((*Value)(member_data), member_count)
	matched := false
	for _, member := range members {
		if value == member {
			matched = true
			break
		}
	}
	if !matched {
		builder = builder.assertion_fail(ASSERTION_FAILURE_ENUM_MEMBER)
	}
	if builder.assertion_recording() {
		builder = assertion_enum_recording(builder, value, members)
	}
	return builder
}

// Distinct-member expansion is useful only to the registration-owned emission plan.
//
//go:noinline
func assertion_enum_recording[Value Integer](
	builder Assertion_Builder, value Value, members []Value,
) (next Assertion_Builder) {
	builder = builder.assertion_guard()
	for index, member := range members {
		duplicate := false
		for _, earlier := range members[:index] {
			if earlier == member {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}
		builder = builder.assertion_axis(value == member)
	}
	return builder
}
