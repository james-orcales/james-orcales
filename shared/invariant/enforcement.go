//go:build !invariant_disable_coverage && !prd && !prod && !production && !invariant_noop

// The enforcing build keeps fluent links observationally silent: they only advance a value and
// latch raw verdicts. Ensure is the single boundary that can panic or mutate coverage.
package invariant

import (
	"fmt"
	"unsafe"
)

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
		recorder_always_failure(bool(condition), message)
	}
	if !recorder.Is_Test {
		return
	}
	if recorder.Is_Benchmark {
		return
	}
	recorder_increment(recorder, message, true)
}

// Keeping diagnostic formatting out of the eager guard leaves its passing branch inlineable.
//
//go:noinline
func recorder_always_failure(condition bool, message string) {
	panic(ASSERTION_FAILURE_MESSAGE_PREFIX + message +
		"  Always — condition was false: " + fmt.Sprint(condition))
}

// Recorder_Assertions starts one deferred chain. Only recording modes consult registration;
// shipped enforcement therefore pays no map, cache, lock, or identity-validation cost.
func Recorder_Assertions(recorder *Recorder, namespace Namespace) (builder Assertion_Builder) {
	builder.Context = unsafe.Pointer(unsafe.StringData(string(namespace)))
	builder.State_A = uintptr(len(namespace))
	if recorder.Assertion_Plans == nil {
		return builder
	}
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
	value int, minimum int, maximum int,
) (next Assertion_Builder) {
	return assertion_range_head(builder, value, minimum, maximum)
}

// Range_Int8 keeps only value-dependent enforcement in ordinary binaries.
func (builder Assertion_Builder) Range_Int8(
	value int8, minimum int8, maximum int8,
) (next Assertion_Builder) {
	return assertion_range_head(builder, value, minimum, maximum)
}

// Range_Int16 keeps only value-dependent enforcement in ordinary binaries.
func (builder Assertion_Builder) Range_Int16(
	value int16, minimum int16, maximum int16,
) (next Assertion_Builder) {
	return assertion_range_head(builder, value, minimum, maximum)
}

// Range_Int32 keeps only value-dependent enforcement in ordinary binaries.
func (builder Assertion_Builder) Range_Int32(
	value int32, minimum int32, maximum int32,
) (next Assertion_Builder) {
	return assertion_range_head(builder, value, minimum, maximum)
}

// Range_Int64 keeps only value-dependent enforcement in ordinary binaries.
func (builder Assertion_Builder) Range_Int64(
	value int64, minimum int64, maximum int64,
) (next Assertion_Builder) {
	return assertion_range_head(builder, value, minimum, maximum)
}

// Range_Uint keeps only value-dependent enforcement in ordinary binaries.
func (builder Assertion_Builder) Range_Uint(
	value uint, minimum uint, maximum uint,
) (next Assertion_Builder) {
	return assertion_range_head(builder, value, minimum, maximum)
}

// Range_Uint8 keeps only value-dependent enforcement in ordinary binaries.
func (builder Assertion_Builder) Range_Uint8(
	value uint8, minimum uint8, maximum uint8,
) (next Assertion_Builder) {
	return assertion_range_head(builder, value, minimum, maximum)
}

// Range_Uint16 keeps only value-dependent enforcement in ordinary binaries.
func (builder Assertion_Builder) Range_Uint16(
	value uint16, minimum uint16, maximum uint16,
) (next Assertion_Builder) {
	return assertion_range_head(builder, value, minimum, maximum)
}

// Range_Uint32 keeps only value-dependent enforcement in ordinary binaries.
func (builder Assertion_Builder) Range_Uint32(
	value uint32, minimum uint32, maximum uint32,
) (next Assertion_Builder) {
	return assertion_range_head(builder, value, minimum, maximum)
}

// Range_Uint64 keeps only value-dependent enforcement in ordinary binaries.
func (builder Assertion_Builder) Range_Uint64(
	value uint64, minimum uint64, maximum uint64,
) (next Assertion_Builder) {
	return assertion_range_head(builder, value, minimum, maximum)
}

// Range_Holed_Int excludes up to four canonical signed holes without constructing a slice.
func (builder Assertion_Builder) Range_Holed_Int(
	value int, minimum int, maximum int,
	hole_1 int, hole_2 int, hole_3 int, hole_4 int,
) (next Assertion_Builder) {
	return assertion_range_holed_head(
		builder, value, minimum, maximum, hole_1, hole_2, hole_3, hole_4)
}

// Range_Holed_Int8 excludes up to four canonical signed holes without constructing a slice.
func (builder Assertion_Builder) Range_Holed_Int8(
	value int8, minimum int8, maximum int8,
	hole_1 int8, hole_2 int8, hole_3 int8, hole_4 int8,
) (next Assertion_Builder) {
	return assertion_range_holed_head(
		builder, value, minimum, maximum, hole_1, hole_2, hole_3, hole_4)
}

// Range_Holed_Int16 excludes up to four canonical signed holes without constructing a slice.
func (builder Assertion_Builder) Range_Holed_Int16(
	value int16, minimum int16, maximum int16,
	hole_1 int16, hole_2 int16, hole_3 int16, hole_4 int16,
) (next Assertion_Builder) {
	return assertion_range_holed_head(
		builder, value, minimum, maximum, hole_1, hole_2, hole_3, hole_4)
}

// Range_Holed_Int32 excludes up to four canonical signed holes without constructing a slice.
func (builder Assertion_Builder) Range_Holed_Int32(
	value int32, minimum int32, maximum int32,
	hole_1 int32, hole_2 int32, hole_3 int32, hole_4 int32,
) (next Assertion_Builder) {
	return assertion_range_holed_head(
		builder, value, minimum, maximum, hole_1, hole_2, hole_3, hole_4)
}

// Range_Holed_Int64 excludes up to four canonical signed holes without constructing a slice.
func (builder Assertion_Builder) Range_Holed_Int64(
	value int64, minimum int64, maximum int64,
	hole_1 int64, hole_2 int64, hole_3 int64, hole_4 int64,
) (next Assertion_Builder) {
	return assertion_range_holed_head(
		builder, value, minimum, maximum, hole_1, hole_2, hole_3, hole_4)
}

// Range_Holed_Uint excludes up to three canonical unsigned holes without constructing a slice.
func (builder Assertion_Builder) Range_Holed_Uint(
	value uint, minimum uint, maximum uint, hole_1 uint, hole_2 uint, hole_3 uint,
) (next Assertion_Builder) {
	return assertion_range_holed_head(
		builder, value, minimum, maximum, hole_1, hole_2, hole_3, hole_3)
}

// Range_Holed_Uint8 excludes up to three canonical unsigned holes without constructing a slice.
func (builder Assertion_Builder) Range_Holed_Uint8(
	value uint8, minimum uint8, maximum uint8,
	hole_1 uint8, hole_2 uint8, hole_3 uint8,
) (next Assertion_Builder) {
	return assertion_range_holed_head(
		builder, value, minimum, maximum, hole_1, hole_2, hole_3, hole_3)
}

// Range_Holed_Uint16 excludes up to three canonical unsigned holes without constructing a slice.
func (builder Assertion_Builder) Range_Holed_Uint16(
	value uint16, minimum uint16, maximum uint16,
	hole_1 uint16, hole_2 uint16, hole_3 uint16,
) (next Assertion_Builder) {
	return assertion_range_holed_head(
		builder, value, minimum, maximum, hole_1, hole_2, hole_3, hole_3)
}

// Range_Holed_Uint32 excludes up to three canonical unsigned holes without constructing a slice.
func (builder Assertion_Builder) Range_Holed_Uint32(
	value uint32, minimum uint32, maximum uint32,
	hole_1 uint32, hole_2 uint32, hole_3 uint32,
) (next Assertion_Builder) {
	return assertion_range_holed_head(
		builder, value, minimum, maximum, hole_1, hole_2, hole_3, hole_3)
}

// Range_Holed_Uint64 excludes up to three canonical unsigned holes without constructing a slice.
func (builder Assertion_Builder) Range_Holed_Uint64(
	value uint64, minimum uint64, maximum uint64,
	hole_1 uint64, hole_2 uint64, hole_3 uint64,
) (next Assertion_Builder) {
	return assertion_range_holed_head(
		builder, value, minimum, maximum, hole_1, hole_2, hole_3, hole_3)
}

// Enum_Int keeps fixed two-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_Int(
	value int, first int, second int,
) (next Assertion_Builder) {
	return assertion_enum_2_head(builder, value, first, second)
}

// Enum_Int8 keeps fixed two-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_Int8(
	value int8, first int8, second int8,
) (next Assertion_Builder) {
	return assertion_enum_2_head(builder, value, first, second)
}

// Enum_Int16 keeps fixed two-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_Int16(
	value int16, first int16, second int16,
) (next Assertion_Builder) {
	return assertion_enum_2_head(builder, value, first, second)
}

// Enum_Int32 keeps fixed two-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_Int32(
	value int32, first int32, second int32,
) (next Assertion_Builder) {
	return assertion_enum_2_head(builder, value, first, second)
}

// Enum_Int64 keeps fixed two-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_Int64(
	value int64, first int64, second int64,
) (next Assertion_Builder) {
	return assertion_enum_2_head(builder, value, first, second)
}

// Enum_Uint keeps fixed two-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_Uint(
	value uint, first uint, second uint,
) (next Assertion_Builder) {
	return assertion_enum_2_head(builder, value, first, second)
}

// Enum_Uint8 keeps fixed two-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_Uint8(
	value uint8, first uint8, second uint8,
) (next Assertion_Builder) {
	return assertion_enum_2_head(builder, value, first, second)
}

// Enum_Uint16 keeps fixed two-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_Uint16(
	value uint16, first uint16, second uint16,
) (next Assertion_Builder) {
	return assertion_enum_2_head(builder, value, first, second)
}

// Enum_Uint32 keeps fixed two-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_Uint32(
	value uint32, first uint32, second uint32,
) (next Assertion_Builder) {
	return assertion_enum_2_head(builder, value, first, second)
}

// Enum_Uint64 keeps fixed two-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_Uint64(
	value uint64, first uint64, second uint64,
) (next Assertion_Builder) {
	return assertion_enum_2_head(builder, value, first, second)
}

// Enum_3_Int keeps fixed three-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_3_Int(
	value int, first int, second int, third int,
) (next Assertion_Builder) {
	return assertion_enum_3_head(builder, value, first, second, third)
}

// Enum_3_Int8 keeps fixed three-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_3_Int8(
	value int8, first int8, second int8, third int8,
) (next Assertion_Builder) {
	return assertion_enum_3_head(builder, value, first, second, third)
}

// Enum_3_Int16 keeps fixed three-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_3_Int16(
	value int16, first int16, second int16, third int16,
) (next Assertion_Builder) {
	return assertion_enum_3_head(builder, value, first, second, third)
}

// Enum_3_Int32 keeps fixed three-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_3_Int32(
	value int32, first int32, second int32, third int32,
) (next Assertion_Builder) {
	return assertion_enum_3_head(builder, value, first, second, third)
}

// Enum_3_Int64 keeps fixed three-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_3_Int64(
	value int64, first int64, second int64, third int64,
) (next Assertion_Builder) {
	return assertion_enum_3_head(builder, value, first, second, third)
}

// Enum_3_Uint keeps fixed three-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_3_Uint(
	value uint, first uint, second uint, third uint,
) (next Assertion_Builder) {
	return assertion_enum_3_head(builder, value, first, second, third)
}

// Enum_3_Uint8 keeps fixed three-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_3_Uint8(
	value uint8, first uint8, second uint8, third uint8,
) (next Assertion_Builder) {
	return assertion_enum_3_head(builder, value, first, second, third)
}

// Enum_3_Uint16 keeps fixed three-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_3_Uint16(
	value uint16, first uint16, second uint16, third uint16,
) (next Assertion_Builder) {
	return assertion_enum_3_head(builder, value, first, second, third)
}

// Enum_3_Uint32 keeps fixed three-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_3_Uint32(
	value uint32, first uint32, second uint32, third uint32,
) (next Assertion_Builder) {
	return assertion_enum_3_head(builder, value, first, second, third)
}

// Enum_3_Uint64 keeps fixed three-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_3_Uint64(
	value uint64, first uint64, second uint64, third uint64,
) (next Assertion_Builder) {
	return assertion_enum_3_head(builder, value, first, second, third)
}

// Enum_4_Int keeps fixed four-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_4_Int(
	value int, first int, second int, third int, fourth int,
) (next Assertion_Builder) {
	return assertion_enum_4_head(builder, value, first, second, third, fourth)
}

// Enum_4_Int8 keeps fixed four-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_4_Int8(
	value int8, first int8, second int8, third int8, fourth int8,
) (next Assertion_Builder) {
	return assertion_enum_4_head(builder, value, first, second, third, fourth)
}

// Enum_4_Int16 keeps fixed four-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_4_Int16(
	value int16, first int16, second int16, third int16, fourth int16,
) (next Assertion_Builder) {
	return assertion_enum_4_head(builder, value, first, second, third, fourth)
}

// Enum_4_Int32 keeps fixed four-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_4_Int32(
	value int32, first int32, second int32, third int32, fourth int32,
) (next Assertion_Builder) {
	return assertion_enum_4_head(builder, value, first, second, third, fourth)
}

// Enum_4_Int64 keeps fixed four-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_4_Int64(
	value int64, first int64, second int64, third int64, fourth int64,
) (next Assertion_Builder) {
	return assertion_enum_4_head(builder, value, first, second, third, fourth)
}

// Enum_4_Uint keeps fixed four-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_4_Uint(
	value uint, first uint, second uint, third uint, fourth uint,
) (next Assertion_Builder) {
	return assertion_enum_4_head(builder, value, first, second, third, fourth)
}

// Enum_4_Uint8 keeps fixed four-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_4_Uint8(
	value uint8, first uint8, second uint8, third uint8, fourth uint8,
) (next Assertion_Builder) {
	return assertion_enum_4_head(builder, value, first, second, third, fourth)
}

// Enum_4_Uint16 keeps fixed four-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_4_Uint16(
	value uint16, first uint16, second uint16, third uint16, fourth uint16,
) (next Assertion_Builder) {
	return assertion_enum_4_head(builder, value, first, second, third, fourth)
}

// Enum_4_Uint32 keeps fixed four-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_4_Uint32(
	value uint32, first uint32, second uint32, third uint32, fourth uint32,
) (next Assertion_Builder) {
	return assertion_enum_4_head(builder, value, first, second, third, fourth)
}

// Enum_4_Uint64 keeps fixed four-member enforcement inlineable in ordinary binaries.
func (builder Assertion_Builder) Enum_4_Uint64(
	value uint64, first uint64, second uint64, third uint64, fourth uint64,
) (next Assertion_Builder) {
	return assertion_enum_4_head(builder, value, first, second, third, fourth)
}

// Ensure keeps the successful ordinary path small enough to inline at every assertion site.
func (builder Assertion_Builder) Ensure() {
	if builder.State_B&(ASSERTION_RECORDING_MASK|ASSERTION_FAILURE_MASK) == 0 {
		return
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

// Replacing recording state after a failure keeps diagnostics rich without enlarging every
// builder or retaining state that Ensure must never credit.
//
//go:noinline
func (builder Assertion_Builder) assertion_fail(
	failure uint8, value any,
) (next Assertion_Builder) {
	if builder.assertion_failure() == ASSERTION_FAILURE_NONE {
		message := assertion_failure_text(
			failure, builder.assertion_namespace(), fmt.Sprint(value))
		builder.Context = unsafe.Pointer(&message)
		builder.State_A = 0
		builder.State_B = uintptr(failure) << ASSERTION_FAILURE_SHIFT
	}
	return builder
}

func (builder *Assertion_Builder) assertion_failure_message() (message string) {
	return *(*string)(builder.Context)
}

func assertion_failure_text(failure uint8, namespace Namespace, value string) (message string) {
	prefix := string(namespace) + " · "
	switch failure {
	case ASSERTION_FAILURE_LINKS:
		return "Assertions exceeds 70 links"
	case ASSERTION_FAILURE_RANGE_DOMAIN:
		return prefix + "Range minimum exceeds maximum: " + value
	case ASSERTION_FAILURE_RANGE_LOWER:
		return prefix + RANGE_GUARD_MINIMUM + "  value below min: " + value
	case ASSERTION_FAILURE_RANGE_UPPER:
		return prefix + RANGE_GUARD_MAXIMUM + "  value exceeds max: " + value
	case ASSERTION_FAILURE_RANGE_EXCLUSION:
		return prefix + "Range exclusion is not strictly inside the interval: " + value
	case ASSERTION_FAILURE_RANGE_EXCLUDED:
		return prefix + "Range value is excluded: " + value
	case ASSERTION_FAILURE_ENUM_DOMAIN:
		return prefix + "Enum requires at least two distinct members: " + value
	case ASSERTION_FAILURE_ENUM_MEMBER:
		return prefix + ENUM_GUARD_MEMBER + "  value is not a member: " + value
	}
	return "Assertions failed"
}

func assertion_range_head[Value Integer](
	builder Assertion_Builder, value Value, minimum Value, maximum Value,
) (next Assertion_Builder) {
	if value >= minimum {
		if value <= maximum {
			if !builder.assertion_recording() {
				return builder
			}
		}
	}
	return assertion_range_slow(builder, value, minimum, maximum)
}

//go:noinline
func assertion_range_slow[Value Integer](
	builder Assertion_Builder, value Value, minimum Value, maximum Value,
) (next Assertion_Builder) {
	if value < minimum {
		builder = builder.assertion_fail(ASSERTION_FAILURE_RANGE_LOWER, value)
	}
	if value > maximum {
		builder = builder.assertion_fail(ASSERTION_FAILURE_RANGE_UPPER, value)
	}
	if builder.assertion_recording() {
		builder = assertion_range_recording(
			builder, value, minimum, maximum, false,
			Value(0), Value(0), Value(0), Value(0))
	}
	return builder
}

func assertion_range_holed_head[Value Integer](
	builder Assertion_Builder, value Value, minimum Value, maximum Value,
	hole_1 Value, hole_2 Value, hole_3 Value, hole_4 Value,
) (next Assertion_Builder) {
	valid := value >= minimum && value <= maximum
	if value == hole_1 {
		valid = false
	}
	if value == hole_2 {
		valid = false
	}
	if value == hole_3 {
		valid = false
	}
	if value == hole_4 {
		valid = false
	}
	if valid {
		if !builder.assertion_recording() {
			return builder
		}
	}
	return assertion_range_holed_slow(
		builder, value, minimum, maximum, hole_1, hole_2, hole_3, hole_4)
}

//go:noinline
func assertion_range_holed_slow[Value Integer](
	builder Assertion_Builder, value Value, minimum Value, maximum Value,
	hole_1 Value, hole_2 Value, hole_3 Value, hole_4 Value,
) (next Assertion_Builder) {
	if value < minimum {
		builder = builder.assertion_fail(ASSERTION_FAILURE_RANGE_LOWER, value)
	}
	if value > maximum {
		builder = builder.assertion_fail(ASSERTION_FAILURE_RANGE_UPPER, value)
	}
	excluded := value == hole_1
	if value == hole_2 {
		excluded = true
	}
	if value == hole_3 {
		excluded = true
	}
	if value == hole_4 {
		excluded = true
	}
	if excluded {
		builder = builder.assertion_fail(ASSERTION_FAILURE_RANGE_EXCLUDED, value)
	}
	if builder.assertion_recording() {
		builder = assertion_range_recording(
			builder, value, minimum, maximum, true,
			hole_1, hole_2, hole_3, hole_4)
	}
	return builder
}

// Only a registered test run needs boundary and sentinel observations.
//
//go:noinline
func assertion_range_recording[Value Integer](
	builder Assertion_Builder, value Value, minimum Value, maximum Value,
	holed bool, hole_1 Value, hole_2 Value, hole_3 Value, hole_4 Value,
) (next Assertion_Builder) {
	builder = builder.assertion_guard()
	builder = builder.assertion_guard()
	if minimum == maximum {
		return builder
	}
	builder = builder.assertion_axis(value == minimum)
	builder = builder.assertion_axis(value == maximum)
	builder = assertion_range_candidate_recording(
		builder, value, minimum, maximum, Value(0), holed,
		hole_1, hole_2, hole_3, hole_4)
	builder = assertion_range_candidate_recording(
		builder, value, minimum, maximum, Value(1), holed,
		hole_1, hole_2, hole_3, hole_4)
	builder = assertion_range_candidate_recording(
		builder, value, minimum, maximum, Value(2), holed,
		hole_1, hole_2, hole_3, hole_4)
	zero := Value(0)
	negative_one := zero - Value(1)
	if negative_one < zero {
		builder = assertion_range_candidate_recording(
			builder, value, minimum, maximum, negative_one, holed,
			hole_1, hole_2, hole_3, hole_4)
	}
	return builder
}

func assertion_range_candidate_recording[Value Integer](
	builder Assertion_Builder, value Value, minimum Value, maximum Value,
	candidate Value, holed bool, hole_1 Value, hole_2 Value, hole_3 Value, hole_4 Value,
) (next Assertion_Builder) {
	outside := candidate <= minimum
	if candidate >= maximum {
		outside = true
	}
	if outside {
		return builder
	}
	if holed {
		excluded := candidate == hole_1
		if candidate == hole_2 {
			excluded = true
		}
		if candidate == hole_3 {
			excluded = true
		}
		if candidate == hole_4 {
			excluded = true
		}
		if excluded {
			return builder
		}
	}
	return builder.assertion_axis(value == candidate)
}

func assertion_enum_2_head[Value Integer](
	builder Assertion_Builder, value Value, first Value, second Value,
) (next Assertion_Builder) {
	matched := value == first
	if value == second {
		matched = true
	}
	if matched {
		if !builder.assertion_recording() {
			return builder
		}
	}
	return assertion_enum_2_slow(builder, value, first, second)
}

//go:noinline
func assertion_enum_2_slow[Value Integer](
	builder Assertion_Builder, value Value, first Value, second Value,
) (next Assertion_Builder) {
	matched := value == first
	if value == second {
		matched = true
	}
	if !matched {
		builder = builder.assertion_fail(ASSERTION_FAILURE_ENUM_MEMBER, value)
	}
	if builder.assertion_recording() {
		builder = builder.assertion_guard()
		builder = builder.assertion_axis(value == first)
		builder = builder.assertion_axis(value == second)
	}
	return builder
}

func assertion_enum_3_head[Value Integer](
	builder Assertion_Builder, value Value, first Value, second Value, third Value,
) (next Assertion_Builder) {
	matched := value == first
	if value == second {
		matched = true
	}
	if value == third {
		matched = true
	}
	if matched {
		if !builder.assertion_recording() {
			return builder
		}
	}
	return assertion_enum_3_slow(builder, value, first, second, third)
}

//go:noinline
func assertion_enum_3_slow[Value Integer](
	builder Assertion_Builder, value Value, first Value, second Value, third Value,
) (next Assertion_Builder) {
	matched := value == first
	if value == second {
		matched = true
	}
	if value == third {
		matched = true
	}
	if !matched {
		builder = builder.assertion_fail(ASSERTION_FAILURE_ENUM_MEMBER, value)
	}
	if builder.assertion_recording() {
		builder = builder.assertion_guard()
		builder = builder.assertion_axis(value == first)
		builder = builder.assertion_axis(value == second)
		builder = builder.assertion_axis(value == third)
	}
	return builder
}

func assertion_enum_4_head[Value Integer](
	builder Assertion_Builder, value Value,
	first Value, second Value, third Value, fourth Value,
) (next Assertion_Builder) {
	matched := value == first
	if value == second {
		matched = true
	}
	if value == third {
		matched = true
	}
	if value == fourth {
		matched = true
	}
	if matched {
		if !builder.assertion_recording() {
			return builder
		}
	}
	return assertion_enum_4_slow(builder, value, first, second, third, fourth)
}

//go:noinline
func assertion_enum_4_slow[Value Integer](
	builder Assertion_Builder, value Value,
	first Value, second Value, third Value, fourth Value,
) (next Assertion_Builder) {
	matched := value == first
	if value == second {
		matched = true
	}
	if value == third {
		matched = true
	}
	if value == fourth {
		matched = true
	}
	if !matched {
		builder = builder.assertion_fail(ASSERTION_FAILURE_ENUM_MEMBER, value)
	}
	if builder.assertion_recording() {
		builder = builder.assertion_guard()
		builder = builder.assertion_axis(value == first)
		builder = builder.assertion_axis(value == second)
		builder = builder.assertion_axis(value == third)
		builder = builder.assertion_axis(value == fourth)
	}
	return builder
}
