//go:build !invariant_noop

package aver

import "local/james-orcales/shared/sim/aver"

// Always is an eager guard, so its enforcement and reachability stay independent of any chain.
func Always[T ~bool](condition T, message string) {
	aver.Recorder_Always(Default, condition, message)
}

// Sometimes states one inline two-branch axis on Default, for a body that owns no bundle.
func Sometimes[T ~bool](condition T, message string) {
	aver.Recorder_Sometimes(Default, condition, message)
}

// Range bounds one inline value to a contiguous interval on Default.
func Range[Value aver.Integer](
	value Value, minimum Value, maximum Value, message string,
) {
	aver.Recorder_Range(Default, value, minimum, maximum, message)
}

// Enum holds one inline value to two members on Default.
func Enum[Value aver.Integer](
	value Value, first Value, second Value, message string,
) {
	aver.Recorder_Enum(Default, value, first, second, message)
}

// Range_Holed removes four fixed exclusions from an inline interval on Default. A duplicate final
// hole stands for an unused slot, so one arity serves every width.
func Range_Holed[Value aver.Integer](
	value Value, minimum Value, maximum Value,
	hole_1 Value, hole_2 Value, hole_3 Value, hole_4 Value, message string,
) {
	aver.Recorder_Range_Holed(
		Default, value, minimum, maximum, hole_1, hole_2, hole_3, hole_4, message)
}

// Tree starts one deferred builder on Default. The subject supplies the chain type.
func Tree[Subject any](
	subject Subject, namespace Namespace,
) (builder Assertion_Builder) {
	return aver.Recorder_Tree(Default, subject, namespace)
}

// Assertion_Builder re-exports the fluent value for explicit APIs without adding an adapter.
type Assertion_Builder = aver.Assertion_Builder

// Go writes a function's inline body into a package's export data only when that package itself
// inlined the function. This package aliases Assertion_Builder and calls no link of its own, thus
// a consumer that imports this package alone reads a signature with no body and pays one call for
// every link a bundle states. Each function here makes this package inline one link, which puts
// that link's body where a consumer can reach it. Measured on the Go scanner: 689 to 509
// microseconds.
//
// A bare reference does not do this. The call is what makes the body travel.

// Carries the Sometimes body into this package's export data.
func inline_sometimes(
	builder Assertion_Builder, condition bool, message string,
) (next Assertion_Builder) {
	return builder.Sometimes(condition, message)
}

// Carries the Range_Int body into this package's export data.
func inline_range_int(
	builder Assertion_Builder, value int, minimum int, maximum int,
) (next Assertion_Builder) {
	return builder.Range_Int(value, minimum, maximum)
}

// Carries the Range_Int8 body into this package's export data.
func inline_range_int8(
	builder Assertion_Builder, value int8, minimum int8, maximum int8,
) (next Assertion_Builder) {
	return builder.Range_Int8(value, minimum, maximum)
}

// Carries the Range_Int16 body into this package's export data.
func inline_range_int16(
	builder Assertion_Builder, value int16, minimum int16, maximum int16,
) (next Assertion_Builder) {
	return builder.Range_Int16(value, minimum, maximum)
}

// Carries the Range_Int32 body into this package's export data.
func inline_range_int32(
	builder Assertion_Builder, value int32, minimum int32, maximum int32,
) (next Assertion_Builder) {
	return builder.Range_Int32(value, minimum, maximum)
}

// Carries the Range_Int64 body into this package's export data.
func inline_range_int64(
	builder Assertion_Builder, value int64, minimum int64, maximum int64,
) (next Assertion_Builder) {
	return builder.Range_Int64(value, minimum, maximum)
}

// Carries the Range_Uint body into this package's export data.
func inline_range_uint(
	builder Assertion_Builder, value uint, minimum uint, maximum uint,
) (next Assertion_Builder) {
	return builder.Range_Uint(value, minimum, maximum)
}

// Carries the Range_Uint8 body into this package's export data.
func inline_range_uint8(
	builder Assertion_Builder, value uint8, minimum uint8, maximum uint8,
) (next Assertion_Builder) {
	return builder.Range_Uint8(value, minimum, maximum)
}

// Carries the Range_Uint16 body into this package's export data.
func inline_range_uint16(
	builder Assertion_Builder, value uint16, minimum uint16, maximum uint16,
) (next Assertion_Builder) {
	return builder.Range_Uint16(value, minimum, maximum)
}

// Carries the Range_Uint32 body into this package's export data.
func inline_range_uint32(
	builder Assertion_Builder, value uint32, minimum uint32, maximum uint32,
) (next Assertion_Builder) {
	return builder.Range_Uint32(value, minimum, maximum)
}

// Carries the Range_Uint64 body into this package's export data.
func inline_range_uint64(
	builder Assertion_Builder, value uint64, minimum uint64, maximum uint64,
) (next Assertion_Builder) {
	return builder.Range_Uint64(value, minimum, maximum)
}

// Carries the Range_Holed_Int body into this package's export data.
func inline_range_holed_int(
	builder Assertion_Builder, value int, minimum int, maximum int, hole_1 int, hole_2 int,
	hole_3 int, hole_4 int,
) (next Assertion_Builder) {
	return builder.Range_Holed_Int(value, minimum, maximum, hole_1, hole_2, hole_3, hole_4)
}

// Carries the Range_Holed_Int8 body into this package's export data.
func inline_range_holed_int8(
	builder Assertion_Builder, value int8, minimum int8, maximum int8, hole_1 int8,
	hole_2 int8, hole_3 int8, hole_4 int8,
) (next Assertion_Builder) {
	return builder.Range_Holed_Int8(value, minimum, maximum, hole_1, hole_2, hole_3, hole_4)
}

// Carries the Range_Holed_Int16 body into this package's export data.
func inline_range_holed_int16(
	builder Assertion_Builder, value int16, minimum int16, maximum int16, hole_1 int16,
	hole_2 int16, hole_3 int16, hole_4 int16,
) (next Assertion_Builder) {
	return builder.Range_Holed_Int16(value, minimum, maximum, hole_1, hole_2, hole_3, hole_4)
}

// Carries the Range_Holed_Int32 body into this package's export data.
func inline_range_holed_int32(
	builder Assertion_Builder, value int32, minimum int32, maximum int32, hole_1 int32,
	hole_2 int32, hole_3 int32, hole_4 int32,
) (next Assertion_Builder) {
	return builder.Range_Holed_Int32(value, minimum, maximum, hole_1, hole_2, hole_3, hole_4)
}

// Carries the Range_Holed_Int64 body into this package's export data.
func inline_range_holed_int64(
	builder Assertion_Builder, value int64, minimum int64, maximum int64, hole_1 int64,
	hole_2 int64, hole_3 int64, hole_4 int64,
) (next Assertion_Builder) {
	return builder.Range_Holed_Int64(value, minimum, maximum, hole_1, hole_2, hole_3, hole_4)
}

// Carries the Range_Holed_Uint body into this package's export data.
func inline_range_holed_uint(
	builder Assertion_Builder, value uint, minimum uint, maximum uint, hole_1 uint,
	hole_2 uint, hole_3 uint,
) (next Assertion_Builder) {
	return builder.Range_Holed_Uint(value, minimum, maximum, hole_1, hole_2, hole_3)
}

// Carries the Range_Holed_Uint8 body into this package's export data.
func inline_range_holed_uint8(
	builder Assertion_Builder, value uint8, minimum uint8, maximum uint8, hole_1 uint8,
	hole_2 uint8, hole_3 uint8,
) (next Assertion_Builder) {
	return builder.Range_Holed_Uint8(value, minimum, maximum, hole_1, hole_2, hole_3)
}

// Carries the Range_Holed_Uint16 body into this package's export data.
func inline_range_holed_uint16(
	builder Assertion_Builder, value uint16, minimum uint16, maximum uint16, hole_1 uint16,
	hole_2 uint16, hole_3 uint16,
) (next Assertion_Builder) {
	return builder.Range_Holed_Uint16(value, minimum, maximum, hole_1, hole_2, hole_3)
}

// Carries the Range_Holed_Uint32 body into this package's export data.
func inline_range_holed_uint32(
	builder Assertion_Builder, value uint32, minimum uint32, maximum uint32, hole_1 uint32,
	hole_2 uint32, hole_3 uint32,
) (next Assertion_Builder) {
	return builder.Range_Holed_Uint32(value, minimum, maximum, hole_1, hole_2, hole_3)
}

// Carries the Range_Holed_Uint64 body into this package's export data.
func inline_range_holed_uint64(
	builder Assertion_Builder, value uint64, minimum uint64, maximum uint64, hole_1 uint64,
	hole_2 uint64, hole_3 uint64,
) (next Assertion_Builder) {
	return builder.Range_Holed_Uint64(value, minimum, maximum, hole_1, hole_2, hole_3)
}

// Carries the Enum_Int body into this package's export data.
func inline_enum_int(
	builder Assertion_Builder, value int, first int, second int,
) (next Assertion_Builder) {
	return builder.Enum_Int(value, first, second)
}

// Carries the Enum_Int8 body into this package's export data.
func inline_enum_int8(
	builder Assertion_Builder, value int8, first int8, second int8,
) (next Assertion_Builder) {
	return builder.Enum_Int8(value, first, second)
}

// Carries the Enum_Int16 body into this package's export data.
func inline_enum_int16(
	builder Assertion_Builder, value int16, first int16, second int16,
) (next Assertion_Builder) {
	return builder.Enum_Int16(value, first, second)
}

// Carries the Enum_Int32 body into this package's export data.
func inline_enum_int32(
	builder Assertion_Builder, value int32, first int32, second int32,
) (next Assertion_Builder) {
	return builder.Enum_Int32(value, first, second)
}

// Carries the Enum_Int64 body into this package's export data.
func inline_enum_int64(
	builder Assertion_Builder, value int64, first int64, second int64,
) (next Assertion_Builder) {
	return builder.Enum_Int64(value, first, second)
}

// Carries the Enum_Uint body into this package's export data.
func inline_enum_uint(
	builder Assertion_Builder, value uint, first uint, second uint,
) (next Assertion_Builder) {
	return builder.Enum_Uint(value, first, second)
}

// Carries the Enum_Uint8 body into this package's export data.
func inline_enum_uint8(
	builder Assertion_Builder, value uint8, first uint8, second uint8,
) (next Assertion_Builder) {
	return builder.Enum_Uint8(value, first, second)
}

// Carries the Enum_Uint16 body into this package's export data.
func inline_enum_uint16(
	builder Assertion_Builder, value uint16, first uint16, second uint16,
) (next Assertion_Builder) {
	return builder.Enum_Uint16(value, first, second)
}

// Carries the Enum_Uint32 body into this package's export data.
func inline_enum_uint32(
	builder Assertion_Builder, value uint32, first uint32, second uint32,
) (next Assertion_Builder) {
	return builder.Enum_Uint32(value, first, second)
}

// Carries the Enum_Uint64 body into this package's export data.
func inline_enum_uint64(
	builder Assertion_Builder, value uint64, first uint64, second uint64,
) (next Assertion_Builder) {
	return builder.Enum_Uint64(value, first, second)
}

// Carries the Enum_3_Int body into this package's export data.
func inline_enum_3_int(
	builder Assertion_Builder, value int, first int, second int, third int,
) (next Assertion_Builder) {
	return builder.Enum_3_Int(value, first, second, third)
}

// Carries the Enum_3_Int8 body into this package's export data.
func inline_enum_3_int8(
	builder Assertion_Builder, value int8, first int8, second int8, third int8,
) (next Assertion_Builder) {
	return builder.Enum_3_Int8(value, first, second, third)
}

// Carries the Enum_3_Int16 body into this package's export data.
func inline_enum_3_int16(
	builder Assertion_Builder, value int16, first int16, second int16, third int16,
) (next Assertion_Builder) {
	return builder.Enum_3_Int16(value, first, second, third)
}

// Carries the Enum_3_Int32 body into this package's export data.
func inline_enum_3_int32(
	builder Assertion_Builder, value int32, first int32, second int32, third int32,
) (next Assertion_Builder) {
	return builder.Enum_3_Int32(value, first, second, third)
}

// Carries the Enum_3_Int64 body into this package's export data.
func inline_enum_3_int64(
	builder Assertion_Builder, value int64, first int64, second int64, third int64,
) (next Assertion_Builder) {
	return builder.Enum_3_Int64(value, first, second, third)
}

// Carries the Enum_3_Uint body into this package's export data.
func inline_enum_3_uint(
	builder Assertion_Builder, value uint, first uint, second uint, third uint,
) (next Assertion_Builder) {
	return builder.Enum_3_Uint(value, first, second, third)
}

// Carries the Enum_3_Uint8 body into this package's export data.
func inline_enum_3_uint8(
	builder Assertion_Builder, value uint8, first uint8, second uint8, third uint8,
) (next Assertion_Builder) {
	return builder.Enum_3_Uint8(value, first, second, third)
}

// Carries the Enum_3_Uint16 body into this package's export data.
func inline_enum_3_uint16(
	builder Assertion_Builder, value uint16, first uint16, second uint16, third uint16,
) (next Assertion_Builder) {
	return builder.Enum_3_Uint16(value, first, second, third)
}

// Carries the Enum_3_Uint32 body into this package's export data.
func inline_enum_3_uint32(
	builder Assertion_Builder, value uint32, first uint32, second uint32, third uint32,
) (next Assertion_Builder) {
	return builder.Enum_3_Uint32(value, first, second, third)
}

// Carries the Enum_3_Uint64 body into this package's export data.
func inline_enum_3_uint64(
	builder Assertion_Builder, value uint64, first uint64, second uint64, third uint64,
) (next Assertion_Builder) {
	return builder.Enum_3_Uint64(value, first, second, third)
}

// Carries the Enum_4_Int body into this package's export data.
func inline_enum_4_int(
	builder Assertion_Builder, value int, first int, second int, third int, fourth int,
) (next Assertion_Builder) {
	return builder.Enum_4_Int(value, first, second, third, fourth)
}

// Carries the Enum_4_Int8 body into this package's export data.
func inline_enum_4_int8(
	builder Assertion_Builder, value int8, first int8, second int8, third int8, fourth int8,
) (next Assertion_Builder) {
	return builder.Enum_4_Int8(value, first, second, third, fourth)
}

// Carries the Enum_4_Int16 body into this package's export data.
func inline_enum_4_int16(
	builder Assertion_Builder, value int16, first int16, second int16, third int16,
	fourth int16,
) (next Assertion_Builder) {
	return builder.Enum_4_Int16(value, first, second, third, fourth)
}

// Carries the Enum_4_Int32 body into this package's export data.
func inline_enum_4_int32(
	builder Assertion_Builder, value int32, first int32, second int32, third int32,
	fourth int32,
) (next Assertion_Builder) {
	return builder.Enum_4_Int32(value, first, second, third, fourth)
}

// Carries the Enum_4_Int64 body into this package's export data.
func inline_enum_4_int64(
	builder Assertion_Builder, value int64, first int64, second int64, third int64,
	fourth int64,
) (next Assertion_Builder) {
	return builder.Enum_4_Int64(value, first, second, third, fourth)
}

// Carries the Enum_4_Uint body into this package's export data.
func inline_enum_4_uint(
	builder Assertion_Builder, value uint, first uint, second uint, third uint, fourth uint,
) (next Assertion_Builder) {
	return builder.Enum_4_Uint(value, first, second, third, fourth)
}

// Carries the Enum_4_Uint8 body into this package's export data.
func inline_enum_4_uint8(
	builder Assertion_Builder, value uint8, first uint8, second uint8, third uint8,
	fourth uint8,
) (next Assertion_Builder) {
	return builder.Enum_4_Uint8(value, first, second, third, fourth)
}

// Carries the Enum_4_Uint16 body into this package's export data.
func inline_enum_4_uint16(
	builder Assertion_Builder, value uint16, first uint16, second uint16, third uint16,
	fourth uint16,
) (next Assertion_Builder) {
	return builder.Enum_4_Uint16(value, first, second, third, fourth)
}

// Carries the Enum_4_Uint32 body into this package's export data.
func inline_enum_4_uint32(
	builder Assertion_Builder, value uint32, first uint32, second uint32, third uint32,
	fourth uint32,
) (next Assertion_Builder) {
	return builder.Enum_4_Uint32(value, first, second, third, fourth)
}

// Carries the Enum_4_Uint64 body into this package's export data.
func inline_enum_4_uint64(
	builder Assertion_Builder, value uint64, first uint64, second uint64, third uint64,
	fourth uint64,
) (next Assertion_Builder) {
	return builder.Enum_4_Uint64(value, first, second, third, fourth)
}

// Carries the Ensure body into this package's export data.
func inline_ensure(
	builder Assertion_Builder,
) {
	builder.Ensure()
}
