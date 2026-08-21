//go:build invariant_noop

package aver

import "local/james-orcales/shared/simulation/aver"

// The noop build states each entry point with an empty body rather than a call the library
// answers with an empty body, and it states the builder and every link of its own. A caller that
// imports this tier alone then meets the whole chain in one package, thus the inliner folds a
// guard away instead of leaving one call for each link. That is the point of this build: it
// measures the body a caller wrote and never the guard around it.

// Assertion_Builder carries nothing in this build, thus every fluent return is free.
type Assertion_Builder struct{}

// Always states nothing in this build.
func Always[T ~bool](condition T, message string) { return }

// Sometimes states nothing in this build.
func Sometimes[T ~bool](condition T, message string) { return }

// Range states nothing in this build.
func Range[Value aver.Integer](value Value, minimum Value, maximum Value, message string) {
	return
}

// Enum states nothing in this build.
func Enum[Value aver.Integer](value Value, first Value, second Value, message string) {
	return
}

// Range_Holed states nothing in this build.
func Range_Holed[Value aver.Integer](
	value Value, minimum Value, maximum Value,
	hole_1 Value, hole_2 Value, hole_3 Value, hole_4 Value, message string,
) {
	return
}

// Tree hands back the empty builder, thus every link the chain states folds away.
func Tree[Subject any](
	subject Subject, namespace Namespace,
) (builder Assertion_Builder) {
	return Assertion_Builder{}
}

// Sometimes preserves signature-identical benchmark source while dropping branch coverage.
func (builder Assertion_Builder) Sometimes(
	condition bool, message string,
) (next Assertion_Builder) {
	return builder
}

// Range_Int preserves signature-identical benchmark source while dropping range enforcement.
func (builder Assertion_Builder) Range_Int(
	value int, minimum int, maximum int,
) (next Assertion_Builder) {
	return builder
}

// Range_Int8 preserves signature-identical benchmark source while dropping range enforcement.
func (builder Assertion_Builder) Range_Int8(
	value int8, minimum int8, maximum int8,
) (next Assertion_Builder) {
	return builder
}

// Range_Int16 preserves signature-identical benchmark source while dropping range enforcement.
func (builder Assertion_Builder) Range_Int16(
	value int16, minimum int16, maximum int16,
) (next Assertion_Builder) {
	return builder
}

// Range_Int32 preserves signature-identical benchmark source while dropping range enforcement.
func (builder Assertion_Builder) Range_Int32(
	value int32, minimum int32, maximum int32,
) (next Assertion_Builder) {
	return builder
}

// Range_Int64 preserves signature-identical benchmark source while dropping range enforcement.
func (builder Assertion_Builder) Range_Int64(
	value int64, minimum int64, maximum int64,
) (next Assertion_Builder) {
	return builder
}

// Range_Uint preserves signature-identical benchmark source while dropping range enforcement.
func (builder Assertion_Builder) Range_Uint(
	value uint, minimum uint, maximum uint,
) (next Assertion_Builder) {
	return builder
}

// Range_Uint8 preserves signature-identical benchmark source while dropping range enforcement.
func (builder Assertion_Builder) Range_Uint8(
	value uint8, minimum uint8, maximum uint8,
) (next Assertion_Builder) {
	return builder
}

// Range_Uint16 preserves signature-identical benchmark source while dropping range enforcement.
func (builder Assertion_Builder) Range_Uint16(
	value uint16, minimum uint16, maximum uint16,
) (next Assertion_Builder) {
	return builder
}

// Range_Uint32 preserves signature-identical benchmark source while dropping range enforcement.
func (builder Assertion_Builder) Range_Uint32(
	value uint32, minimum uint32, maximum uint32,
) (next Assertion_Builder) {
	return builder
}

// Range_Uint64 preserves signature-identical benchmark source while dropping range enforcement.
func (builder Assertion_Builder) Range_Uint64(
	value uint64, minimum uint64, maximum uint64,
) (next Assertion_Builder) {
	return builder
}

// Range_Holed_Int preserves comparable benchmark source while dropping range enforcement.
func (builder Assertion_Builder) Range_Holed_Int(
	value int, minimum int, maximum int,
	hole_1 int, hole_2 int, hole_3 int, hole_4 int,
) (next Assertion_Builder) {
	return builder
}

// Range_Holed_Int8 preserves comparable benchmark source while dropping range enforcement.
func (builder Assertion_Builder) Range_Holed_Int8(
	value int8, minimum int8, maximum int8,
	hole_1 int8, hole_2 int8, hole_3 int8, hole_4 int8,
) (next Assertion_Builder) {
	return builder
}

// Range_Holed_Int16 preserves comparable benchmark source while dropping range enforcement.
func (builder Assertion_Builder) Range_Holed_Int16(
	value int16, minimum int16, maximum int16,
	hole_1 int16, hole_2 int16, hole_3 int16, hole_4 int16,
) (next Assertion_Builder) {
	return builder
}

// Range_Holed_Int32 preserves comparable benchmark source while dropping range enforcement.
func (builder Assertion_Builder) Range_Holed_Int32(
	value int32, minimum int32, maximum int32,
	hole_1 int32, hole_2 int32, hole_3 int32, hole_4 int32,
) (next Assertion_Builder) {
	return builder
}

// Range_Holed_Int64 preserves comparable benchmark source while dropping range enforcement.
func (builder Assertion_Builder) Range_Holed_Int64(
	value int64, minimum int64, maximum int64,
	hole_1 int64, hole_2 int64, hole_3 int64, hole_4 int64,
) (next Assertion_Builder) {
	return builder
}

// Range_Holed_Uint preserves comparable benchmark source while dropping range enforcement.
func (builder Assertion_Builder) Range_Holed_Uint(
	value uint, minimum uint, maximum uint, hole_1 uint, hole_2 uint, hole_3 uint,
) (next Assertion_Builder) {
	return builder
}

// Range_Holed_Uint8 preserves comparable benchmark source while dropping range enforcement.
func (builder Assertion_Builder) Range_Holed_Uint8(
	value uint8, minimum uint8, maximum uint8,
	hole_1 uint8, hole_2 uint8, hole_3 uint8,
) (next Assertion_Builder) {
	return builder
}

// Range_Holed_Uint16 preserves comparable benchmark source while dropping range enforcement.
func (builder Assertion_Builder) Range_Holed_Uint16(
	value uint16, minimum uint16, maximum uint16,
	hole_1 uint16, hole_2 uint16, hole_3 uint16,
) (next Assertion_Builder) {
	return builder
}

// Range_Holed_Uint32 preserves comparable benchmark source while dropping range enforcement.
func (builder Assertion_Builder) Range_Holed_Uint32(
	value uint32, minimum uint32, maximum uint32,
	hole_1 uint32, hole_2 uint32, hole_3 uint32,
) (next Assertion_Builder) {
	return builder
}

// Range_Holed_Uint64 preserves comparable benchmark source while dropping range enforcement.
func (builder Assertion_Builder) Range_Holed_Uint64(
	value uint64, minimum uint64, maximum uint64,
	hole_1 uint64, hole_2 uint64, hole_3 uint64,
) (next Assertion_Builder) {
	return builder
}

// Enum_Int preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_Int(
	value int, first int, second int,
) (next Assertion_Builder) {
	return builder
}

// Enum_Int8 preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_Int8(
	value int8, first int8, second int8,
) (next Assertion_Builder) {
	return builder
}

// Enum_Int16 preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_Int16(
	value int16, first int16, second int16,
) (next Assertion_Builder) {
	return builder
}

// Enum_Int32 preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_Int32(
	value int32, first int32, second int32,
) (next Assertion_Builder) {
	return builder
}

// Enum_Int64 preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_Int64(
	value int64, first int64, second int64,
) (next Assertion_Builder) {
	return builder
}

// Enum_Uint preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_Uint(
	value uint, first uint, second uint,
) (next Assertion_Builder) {
	return builder
}

// Enum_Uint8 preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_Uint8(
	value uint8, first uint8, second uint8,
) (next Assertion_Builder) {
	return builder
}

// Enum_Uint16 preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_Uint16(
	value uint16, first uint16, second uint16,
) (next Assertion_Builder) {
	return builder
}

// Enum_Uint32 preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_Uint32(
	value uint32, first uint32, second uint32,
) (next Assertion_Builder) {
	return builder
}

// Enum_Uint64 preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_Uint64(
	value uint64, first uint64, second uint64,
) (next Assertion_Builder) {
	return builder
}

// Enum_3_Int preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_3_Int(
	value int, first int, second int, third int,
) (next Assertion_Builder) {
	return builder
}

// Enum_3_Int8 preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_3_Int8(
	value int8, first int8, second int8, third int8,
) (next Assertion_Builder) {
	return builder
}

// Enum_3_Int16 preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_3_Int16(
	value int16, first int16, second int16, third int16,
) (next Assertion_Builder) {
	return builder
}

// Enum_3_Int32 preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_3_Int32(
	value int32, first int32, second int32, third int32,
) (next Assertion_Builder) {
	return builder
}

// Enum_3_Int64 preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_3_Int64(
	value int64, first int64, second int64, third int64,
) (next Assertion_Builder) {
	return builder
}

// Enum_3_Uint preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_3_Uint(
	value uint, first uint, second uint, third uint,
) (next Assertion_Builder) {
	return builder
}

// Enum_3_Uint8 preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_3_Uint8(
	value uint8, first uint8, second uint8, third uint8,
) (next Assertion_Builder) {
	return builder
}

// Enum_3_Uint16 preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_3_Uint16(
	value uint16, first uint16, second uint16, third uint16,
) (next Assertion_Builder) {
	return builder
}

// Enum_3_Uint32 preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_3_Uint32(
	value uint32, first uint32, second uint32, third uint32,
) (next Assertion_Builder) {
	return builder
}

// Enum_3_Uint64 preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_3_Uint64(
	value uint64, first uint64, second uint64, third uint64,
) (next Assertion_Builder) {
	return builder
}

// Enum_4_Int preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_4_Int(
	value int, first int, second int, third int, fourth int,
) (next Assertion_Builder) {
	return builder
}

// Enum_4_Int8 preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_4_Int8(
	value int8, first int8, second int8, third int8, fourth int8,
) (next Assertion_Builder) {
	return builder
}

// Enum_4_Int16 preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_4_Int16(
	value int16, first int16, second int16, third int16, fourth int16,
) (next Assertion_Builder) {
	return builder
}

// Enum_4_Int32 preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_4_Int32(
	value int32, first int32, second int32, third int32, fourth int32,
) (next Assertion_Builder) {
	return builder
}

// Enum_4_Int64 preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_4_Int64(
	value int64, first int64, second int64, third int64, fourth int64,
) (next Assertion_Builder) {
	return builder
}

// Enum_4_Uint preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_4_Uint(
	value uint, first uint, second uint, third uint, fourth uint,
) (next Assertion_Builder) {
	return builder
}

// Enum_4_Uint8 preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_4_Uint8(
	value uint8, first uint8, second uint8, third uint8, fourth uint8,
) (next Assertion_Builder) {
	return builder
}

// Enum_4_Uint16 preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_4_Uint16(
	value uint16, first uint16, second uint16, third uint16, fourth uint16,
) (next Assertion_Builder) {
	return builder
}

// Enum_4_Uint32 preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_4_Uint32(
	value uint32, first uint32, second uint32, third uint32, fourth uint32,
) (next Assertion_Builder) {
	return builder
}

// Enum_4_Uint64 preserves comparable benchmark source while dropping membership enforcement.
func (builder Assertion_Builder) Enum_4_Uint64(
	value uint64, first uint64, second uint64, third uint64, fourth uint64,
) (next Assertion_Builder) {
	return builder
}

// Ensure preserves chain terminator while noop links retain no deferred state.
func (builder Assertion_Builder) Ensure() { return }
