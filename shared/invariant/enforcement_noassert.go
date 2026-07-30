//go:build noassert

// The disabled build is signature-identical and behavior-free by construction.
package invariant

func Recorder_Always[T ~bool](recorder *Recorder, condition T, message string) {}

func Recorder_Assertions(
	recorder *Recorder, namespace Namespace,
) (builder Assertion_Builder) {
	return Assertion_Builder{}
}

func (builder Assertion_Builder) Sometimes(
	condition bool, message string,
) (next Assertion_Builder) {
	return builder
}

func (builder Assertion_Builder) Range_Int(
	value int, minimum int, maximum int, excluded ...int,
) (next Assertion_Builder) {
	return builder
}

func (builder Assertion_Builder) Range_Int8(
	value int8, minimum int8, maximum int8, excluded ...int8,
) (next Assertion_Builder) {
	return builder
}

func (builder Assertion_Builder) Range_Int16(
	value int16, minimum int16, maximum int16, excluded ...int16,
) (next Assertion_Builder) {
	return builder
}

func (builder Assertion_Builder) Range_Int32(
	value int32, minimum int32, maximum int32, excluded ...int32,
) (next Assertion_Builder) {
	return builder
}

func (builder Assertion_Builder) Range_Int64(
	value int64, minimum int64, maximum int64, excluded ...int64,
) (next Assertion_Builder) {
	return builder
}

func (builder Assertion_Builder) Range_Uint(
	value uint, minimum uint, maximum uint, excluded ...uint,
) (next Assertion_Builder) {
	return builder
}

func (builder Assertion_Builder) Range_Uint8(
	value uint8, minimum uint8, maximum uint8, excluded ...uint8,
) (next Assertion_Builder) {
	return builder
}

func (builder Assertion_Builder) Range_Uint16(
	value uint16, minimum uint16, maximum uint16, excluded ...uint16,
) (next Assertion_Builder) {
	return builder
}

func (builder Assertion_Builder) Range_Uint32(
	value uint32, minimum uint32, maximum uint32, excluded ...uint32,
) (next Assertion_Builder) {
	return builder
}

func (builder Assertion_Builder) Range_Uint64(
	value uint64, minimum uint64, maximum uint64, excluded ...uint64,
) (next Assertion_Builder) {
	return builder
}

func (builder Assertion_Builder) Enum_Int(
	value int, members ...int,
) (next Assertion_Builder) {
	return builder
}

func (builder Assertion_Builder) Enum_Int8(
	value int8, members ...int8,
) (next Assertion_Builder) {
	return builder
}

func (builder Assertion_Builder) Enum_Int16(
	value int16, members ...int16,
) (next Assertion_Builder) {
	return builder
}

func (builder Assertion_Builder) Enum_Int32(
	value int32, members ...int32,
) (next Assertion_Builder) {
	return builder
}

func (builder Assertion_Builder) Enum_Int64(
	value int64, members ...int64,
) (next Assertion_Builder) {
	return builder
}

func (builder Assertion_Builder) Enum_Uint(
	value uint, members ...uint,
) (next Assertion_Builder) {
	return builder
}

func (builder Assertion_Builder) Enum_Uint8(
	value uint8, members ...uint8,
) (next Assertion_Builder) {
	return builder
}

func (builder Assertion_Builder) Enum_Uint16(
	value uint16, members ...uint16,
) (next Assertion_Builder) {
	return builder
}

func (builder Assertion_Builder) Enum_Uint32(
	value uint32, members ...uint32,
) (next Assertion_Builder) {
	return builder
}

func (builder Assertion_Builder) Enum_Uint64(
	value uint64, members ...uint64,
) (next Assertion_Builder) {
	return builder
}

func (builder Assertion_Builder) Ensure() {}
