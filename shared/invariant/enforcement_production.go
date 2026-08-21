//go:build (invariant_disable_coverage || prd || prod || production) && !invariant_noop

// The production build removes recording and deferred failure so successful checks collapse into
// their callers while violations still stop execution.
package invariant

import "unsafe"

// Recorder_Always remains eager because production removes every recording concern. The guard
// holds one cold call so it stays inside the inline budget; a failure body here costs more than
// the budget and every Always would stay a call.
func Recorder_Always[T ~bool](recorder *Recorder, condition T, message string) {
	if !condition {
		always_violation(recorder, bool(condition), message)
	}
}

// The condition arrives as a plain bool. A generic callee costs the guard four more and an any
// argument two more, and either puts a one-Always bundle over the inline budget. The value is
// false in every failure, thus the declared type carries nothing the message does not.
//
//go:noinline
func always_violation(recorder *Recorder, condition bool, message string) {
	failure := Assertion_Failure{
		Identity: message,
		Reason:   "  Always — condition was false",
		Value:    condition,
	}
	recorder_fatal_hook(recorder, failure.Error())
	panic(failure)
}

// Carries a recorder only when a composition hook is active. The usual production builder stays
// the zero value and retains its three-word, allocation-free representation.
func production_builder(recorder *Recorder) (builder Assertion_Builder) {
	if recorder == nil {
		return builder
	}
	if recorder.On_Fatal == nil {
		return builder
	}
	builder.Context = unsafe.Pointer(recorder)
	return builder
}

// Delivers one production assertion failure before the assertion site panics.
func production_fatal(builder Assertion_Builder, failure Assertion_Failure) {
	var recorder *Recorder
	if builder.Context != nil {
		recorder = (*Recorder)(builder.Context)
	}
	recorder_fatal_hook(recorder, failure.Error())
}

// Recorder_Sometimes keeps only its two-branch coverage duty, which production drops, so it has
// nothing left to enforce. A caller-chosen condition cannot violate a typed domain.
func Recorder_Sometimes[T ~bool](recorder *Recorder, condition T, message string) {
	return
}

// Recorder_Range keeps only value-dependent enforcement in the production build.
func Recorder_Range[Value Integer](
	recorder *Recorder, value Value, minimum Value, maximum Value, message string,
) {
	production_range(production_builder(recorder), value, minimum, maximum)
}

// Recorder_Enum keeps only membership enforcement in the production build.
func Recorder_Enum[Value Integer](
	recorder *Recorder, value Value, first Value, second Value, message string,
) {
	production_enum_2(production_builder(recorder), value, first, second)
}

// Recorder_Range_Holed keeps only value-dependent enforcement in the production build.
func Recorder_Range_Holed[Value Integer](
	recorder *Recorder, value Value, minimum Value, maximum Value,
	hole_1 Value, hole_2 Value, hole_3 Value, hole_4 Value, message string,
) {
	production_range_holed(
		production_builder(recorder), value, minimum, maximum,
		hole_1, hole_2, hole_3, hole_4)
}

// Recorder_Tree discards recorder state so production carries enforcement alone. It repeats the
// production_builder body instead of calling it: a consumer that reaches this function through
// the sugar package gets no body for an unexported callee, thus a call here is a real call at
// every bundle root.
func Recorder_Tree[Subject any](
	recorder *Recorder, subject Subject, namespace Namespace,
) (builder Assertion_Builder) {
	if recorder == nil {
		return builder
	}
	if recorder.On_Fatal == nil {
		return builder
	}
	builder.Context = unsafe.Pointer(recorder)
	return builder
}

// Sometimes disappears because a caller-chosen condition cannot violate a typed domain.
func (builder Assertion_Builder) Sometimes(
	condition bool, message string,
) (next Assertion_Builder) {
	return builder
}

// Range_Int keeps only value-dependent enforcement in the production build.
func (builder Assertion_Builder) Range_Int(
	value int, minimum int, maximum int,
) (next Assertion_Builder) {
	switch {
	case value < minimum, value > maximum:
		range_violation(builder, value, minimum)
	}
	return builder
}

// Range_Int8 keeps only value-dependent enforcement in the production build.
func (builder Assertion_Builder) Range_Int8(
	value int8, minimum int8, maximum int8,
) (next Assertion_Builder) {
	switch {
	case value < minimum, value > maximum:
		range_violation(builder, value, minimum)
	}
	return builder
}

// Range_Int16 keeps only value-dependent enforcement in the production build.
func (builder Assertion_Builder) Range_Int16(
	value int16, minimum int16, maximum int16,
) (next Assertion_Builder) {
	switch {
	case value < minimum, value > maximum:
		range_violation(builder, value, minimum)
	}
	return builder
}

// Range_Int32 keeps only value-dependent enforcement in the production build.
func (builder Assertion_Builder) Range_Int32(
	value int32, minimum int32, maximum int32,
) (next Assertion_Builder) {
	switch {
	case value < minimum, value > maximum:
		range_violation(builder, value, minimum)
	}
	return builder
}

// Range_Int64 keeps only value-dependent enforcement in the production build.
func (builder Assertion_Builder) Range_Int64(
	value int64, minimum int64, maximum int64,
) (next Assertion_Builder) {
	switch {
	case value < minimum, value > maximum:
		range_violation(builder, value, minimum)
	}
	return builder
}

// Range_Uint keeps only value-dependent enforcement in the production build.
func (builder Assertion_Builder) Range_Uint(
	value uint, minimum uint, maximum uint,
) (next Assertion_Builder) {
	switch {
	case value < minimum, value > maximum:
		range_violation(builder, value, minimum)
	}
	return builder
}

// Range_Uint8 keeps only value-dependent enforcement in the production build.
func (builder Assertion_Builder) Range_Uint8(
	value uint8, minimum uint8, maximum uint8,
) (next Assertion_Builder) {
	switch {
	case value < minimum, value > maximum:
		range_violation(builder, value, minimum)
	}
	return builder
}

// Range_Uint16 keeps only value-dependent enforcement in the production build.
func (builder Assertion_Builder) Range_Uint16(
	value uint16, minimum uint16, maximum uint16,
) (next Assertion_Builder) {
	switch {
	case value < minimum, value > maximum:
		range_violation(builder, value, minimum)
	}
	return builder
}

// Range_Uint32 keeps only value-dependent enforcement in the production build.
func (builder Assertion_Builder) Range_Uint32(
	value uint32, minimum uint32, maximum uint32,
) (next Assertion_Builder) {
	switch {
	case value < minimum, value > maximum:
		range_violation(builder, value, minimum)
	}
	return builder
}

// Range_Uint64 keeps only value-dependent enforcement in the production build.
func (builder Assertion_Builder) Range_Uint64(
	value uint64, minimum uint64, maximum uint64,
) (next Assertion_Builder) {
	switch {
	case value < minimum, value > maximum:
		range_violation(builder, value, minimum)
	}
	return builder
}

// Range_Holed_Int isolates four fixed signed exclusions without constructing a slice.
func (builder Assertion_Builder) Range_Holed_Int(
	value int, minimum int, maximum int,
	hole_1 int, hole_2 int, hole_3 int, hole_4 int,
) (next Assertion_Builder) {
	switch {
	case value < minimum,
		value > maximum,
		value == hole_1,
		value == hole_2,
		value == hole_3,
		value == hole_4:
		range_holed_violation(builder, value, minimum, maximum)
	}
	return builder
}

// Range_Holed_Int8 isolates four fixed signed exclusions without constructing a slice.
func (builder Assertion_Builder) Range_Holed_Int8(
	value int8, minimum int8, maximum int8,
	hole_1 int8, hole_2 int8, hole_3 int8, hole_4 int8,
) (next Assertion_Builder) {
	switch {
	case value < minimum,
		value > maximum,
		value == hole_1,
		value == hole_2,
		value == hole_3,
		value == hole_4:
		range_holed_violation(builder, value, minimum, maximum)
	}
	return builder
}

// Range_Holed_Int16 isolates four fixed signed exclusions without constructing a slice.
func (builder Assertion_Builder) Range_Holed_Int16(
	value int16, minimum int16, maximum int16,
	hole_1 int16, hole_2 int16, hole_3 int16, hole_4 int16,
) (next Assertion_Builder) {
	switch {
	case value < minimum,
		value > maximum,
		value == hole_1,
		value == hole_2,
		value == hole_3,
		value == hole_4:
		range_holed_violation(builder, value, minimum, maximum)
	}
	return builder
}

// Range_Holed_Int32 isolates four fixed signed exclusions without constructing a slice.
func (builder Assertion_Builder) Range_Holed_Int32(
	value int32, minimum int32, maximum int32,
	hole_1 int32, hole_2 int32, hole_3 int32, hole_4 int32,
) (next Assertion_Builder) {
	switch {
	case value < minimum,
		value > maximum,
		value == hole_1,
		value == hole_2,
		value == hole_3,
		value == hole_4:
		range_holed_violation(builder, value, minimum, maximum)
	}
	return builder
}

// Range_Holed_Int64 isolates four fixed signed exclusions without constructing a slice.
func (builder Assertion_Builder) Range_Holed_Int64(
	value int64, minimum int64, maximum int64,
	hole_1 int64, hole_2 int64, hole_3 int64, hole_4 int64,
) (next Assertion_Builder) {
	switch {
	case value < minimum,
		value > maximum,
		value == hole_1,
		value == hole_2,
		value == hole_3,
		value == hole_4:
		range_holed_violation(builder, value, minimum, maximum)
	}
	return builder
}

// Range_Holed_Uint isolates three fixed unsigned exclusions without constructing a slice.
func (builder Assertion_Builder) Range_Holed_Uint(
	value uint, minimum uint, maximum uint, hole_1 uint, hole_2 uint, hole_3 uint,
) (next Assertion_Builder) {
	switch {
	case value < minimum,
		value > maximum,
		value == hole_1,
		value == hole_2,
		value == hole_3:
		range_holed_violation(builder, value, minimum, maximum)
	}
	return builder
}

// Range_Holed_Uint8 isolates three fixed unsigned exclusions without constructing a slice.
func (builder Assertion_Builder) Range_Holed_Uint8(
	value uint8, minimum uint8, maximum uint8,
	hole_1 uint8, hole_2 uint8, hole_3 uint8,
) (next Assertion_Builder) {
	switch {
	case value < minimum,
		value > maximum,
		value == hole_1,
		value == hole_2,
		value == hole_3:
		range_holed_violation(builder, value, minimum, maximum)
	}
	return builder
}

// Range_Holed_Uint16 isolates three fixed unsigned exclusions without constructing a slice.
func (builder Assertion_Builder) Range_Holed_Uint16(
	value uint16, minimum uint16, maximum uint16,
	hole_1 uint16, hole_2 uint16, hole_3 uint16,
) (next Assertion_Builder) {
	switch {
	case value < minimum,
		value > maximum,
		value == hole_1,
		value == hole_2,
		value == hole_3:
		range_holed_violation(builder, value, minimum, maximum)
	}
	return builder
}

// Range_Holed_Uint32 isolates three fixed unsigned exclusions without constructing a slice.
func (builder Assertion_Builder) Range_Holed_Uint32(
	value uint32, minimum uint32, maximum uint32,
	hole_1 uint32, hole_2 uint32, hole_3 uint32,
) (next Assertion_Builder) {
	switch {
	case value < minimum,
		value > maximum,
		value == hole_1,
		value == hole_2,
		value == hole_3:
		range_holed_violation(builder, value, minimum, maximum)
	}
	return builder
}

// Range_Holed_Uint64 isolates three fixed unsigned exclusions without constructing a slice.
func (builder Assertion_Builder) Range_Holed_Uint64(
	value uint64, minimum uint64, maximum uint64,
	hole_1 uint64, hole_2 uint64, hole_3 uint64,
) (next Assertion_Builder) {
	switch {
	case value < minimum,
		value > maximum,
		value == hole_1,
		value == hole_2,
		value == hole_3:
		range_holed_violation(builder, value, minimum, maximum)
	}
	return builder
}

// Enum_Int isolates fixed two-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_Int(
	value int, first int, second int,
) (next Assertion_Builder) {
	switch value {
	case first, second:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_Int8 isolates fixed two-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_Int8(
	value int8, first int8, second int8,
) (next Assertion_Builder) {
	switch value {
	case first, second:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_Int16 isolates fixed two-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_Int16(
	value int16, first int16, second int16,
) (next Assertion_Builder) {
	switch value {
	case first, second:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_Int32 isolates fixed two-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_Int32(
	value int32, first int32, second int32,
) (next Assertion_Builder) {
	switch value {
	case first, second:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_Int64 isolates fixed two-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_Int64(
	value int64, first int64, second int64,
) (next Assertion_Builder) {
	switch value {
	case first, second:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_Uint isolates fixed two-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_Uint(
	value uint, first uint, second uint,
) (next Assertion_Builder) {
	switch value {
	case first, second:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_Uint8 isolates fixed two-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_Uint8(
	value uint8, first uint8, second uint8,
) (next Assertion_Builder) {
	switch value {
	case first, second:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_Uint16 isolates fixed two-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_Uint16(
	value uint16, first uint16, second uint16,
) (next Assertion_Builder) {
	switch value {
	case first, second:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_Uint32 isolates fixed two-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_Uint32(
	value uint32, first uint32, second uint32,
) (next Assertion_Builder) {
	switch value {
	case first, second:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_Uint64 isolates fixed two-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_Uint64(
	value uint64, first uint64, second uint64,
) (next Assertion_Builder) {
	switch value {
	case first, second:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_3_Int isolates fixed three-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_3_Int(
	value int, first int, second int, third int,
) (next Assertion_Builder) {
	switch value {
	case first, second, third:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_3_Int8 isolates fixed three-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_3_Int8(
	value int8, first int8, second int8, third int8,
) (next Assertion_Builder) {
	switch value {
	case first, second, third:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_3_Int16 isolates fixed three-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_3_Int16(
	value int16, first int16, second int16, third int16,
) (next Assertion_Builder) {
	switch value {
	case first, second, third:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_3_Int32 isolates fixed three-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_3_Int32(
	value int32, first int32, second int32, third int32,
) (next Assertion_Builder) {
	switch value {
	case first, second, third:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_3_Int64 isolates fixed three-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_3_Int64(
	value int64, first int64, second int64, third int64,
) (next Assertion_Builder) {
	switch value {
	case first, second, third:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_3_Uint isolates fixed three-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_3_Uint(
	value uint, first uint, second uint, third uint,
) (next Assertion_Builder) {
	switch value {
	case first, second, third:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_3_Uint8 isolates fixed three-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_3_Uint8(
	value uint8, first uint8, second uint8, third uint8,
) (next Assertion_Builder) {
	switch value {
	case first, second, third:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_3_Uint16 isolates fixed three-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_3_Uint16(
	value uint16, first uint16, second uint16, third uint16,
) (next Assertion_Builder) {
	switch value {
	case first, second, third:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_3_Uint32 isolates fixed three-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_3_Uint32(
	value uint32, first uint32, second uint32, third uint32,
) (next Assertion_Builder) {
	switch value {
	case first, second, third:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_3_Uint64 isolates fixed three-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_3_Uint64(
	value uint64, first uint64, second uint64, third uint64,
) (next Assertion_Builder) {
	switch value {
	case first, second, third:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_4_Int isolates fixed four-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_4_Int(
	value int, first int, second int, third int, fourth int,
) (next Assertion_Builder) {
	switch value {
	case first, second, third, fourth:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_4_Int8 isolates fixed four-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_4_Int8(
	value int8, first int8, second int8, third int8, fourth int8,
) (next Assertion_Builder) {
	switch value {
	case first, second, third, fourth:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_4_Int16 isolates fixed four-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_4_Int16(
	value int16, first int16, second int16, third int16, fourth int16,
) (next Assertion_Builder) {
	switch value {
	case first, second, third, fourth:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_4_Int32 isolates fixed four-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_4_Int32(
	value int32, first int32, second int32, third int32, fourth int32,
) (next Assertion_Builder) {
	switch value {
	case first, second, third, fourth:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_4_Int64 isolates fixed four-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_4_Int64(
	value int64, first int64, second int64, third int64, fourth int64,
) (next Assertion_Builder) {
	switch value {
	case first, second, third, fourth:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_4_Uint isolates fixed four-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_4_Uint(
	value uint, first uint, second uint, third uint, fourth uint,
) (next Assertion_Builder) {
	switch value {
	case first, second, third, fourth:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_4_Uint8 isolates fixed four-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_4_Uint8(
	value uint8, first uint8, second uint8, third uint8, fourth uint8,
) (next Assertion_Builder) {
	switch value {
	case first, second, third, fourth:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_4_Uint16 isolates fixed four-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_4_Uint16(
	value uint16, first uint16, second uint16, third uint16, fourth uint16,
) (next Assertion_Builder) {
	switch value {
	case first, second, third, fourth:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_4_Uint32 isolates fixed four-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_4_Uint32(
	value uint32, first uint32, second uint32, third uint32, fourth uint32,
) (next Assertion_Builder) {
	switch value {
	case first, second, third, fourth:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Enum_4_Uint64 isolates fixed four-member enforcement from recording and deferred diagnostics.
func (builder Assertion_Builder) Enum_4_Uint64(
	value uint64, first uint64, second uint64, third uint64, fourth uint64,
) (next Assertion_Builder) {
	switch value {
	case first, second, third, fourth:
		return builder
	}
	enum_violation(builder, value)
	return builder
}

// Ensure is inert because every enforcing link panics eagerly in production.
func (builder Assertion_Builder) Ensure() {
	return
}

func production_range[Value Integer](
	builder Assertion_Builder, value Value, minimum Value, maximum Value,
) (next Assertion_Builder) {
	if value < minimum {
		failure := Assertion_Failure{
			Identity: RANGE_GUARD_MINIMUM,
			Reason:   "  value below min",
			Value:    value,
		}
		production_fatal(builder, failure)
		panic(failure)
	}
	if value > maximum {
		failure := Assertion_Failure{
			Identity: RANGE_GUARD_MAXIMUM,
			Reason:   "  value exceeds max",
			Value:    value,
		}
		production_fatal(builder, failure)
		panic(failure)
	}
	return builder
}

func production_range_holed[Value Integer](
	builder Assertion_Builder, value Value, minimum Value, maximum Value,
	hole_1 Value, hole_2 Value, hole_3 Value, hole_4 Value,
) (next Assertion_Builder) {
	if value < minimum {
		failure := Assertion_Failure{
			Identity: RANGE_GUARD_MINIMUM,
			Reason:   "  value below min",
			Value:    value,
		}
		production_fatal(builder, failure)
		panic(failure)
	}
	if value > maximum {
		failure := Assertion_Failure{
			Identity: RANGE_GUARD_MAXIMUM,
			Reason:   "  value exceeds max",
			Value:    value,
		}
		production_fatal(builder, failure)
		panic(failure)
	}
	switch value {
	case hole_1, hole_2, hole_3, hole_4:
		failure := Assertion_Failure{
			Identity: "Range value is excluded",
			Value:    value,
		}
		production_fatal(builder, failure)
		panic(failure)
	}
	return builder
}

func production_enum_2[Value Integer](
	builder Assertion_Builder, value Value, first Value, second Value,
) (next Assertion_Builder) {
	switch value {
	case first, second:
		return builder
	}
	failure := Assertion_Failure{
		Identity: ENUM_GUARD_MEMBER,
		Reason:   "  value is not a member",
		Value:    value,
	}
	production_fatal(builder, failure)
	panic(failure)
}

func production_enum_3[Value Integer](
	builder Assertion_Builder, value Value, first Value, second Value, third Value,
) (next Assertion_Builder) {
	switch value {
	case first, second, third:
		return builder
	}
	failure := Assertion_Failure{
		Identity: ENUM_GUARD_MEMBER,
		Reason:   "  value is not a member",
		Value:    value,
	}
	production_fatal(builder, failure)
	panic(failure)
}

func production_enum_4[Value Integer](
	builder Assertion_Builder,
	value Value, first Value, second Value, third Value, fourth Value,
) (next Assertion_Builder) {
	switch value {
	case first, second, third, fourth:
		return builder
	}
	failure := Assertion_Failure{
		Identity: ENUM_GUARD_MEMBER,
		Reason:   "  value is not a member",
		Value:    value,
	}
	production_fatal(builder, failure)
	panic(failure)
}

// A guard body must stay inside the inline budget, thus it holds one call site and the cold body
// lives here. Without go:noinline the compiler folds these back into the guard and the guard
// exceeds the budget again. The cold functions are generic so a failure carries the value at its
// declared width; a generic call costs the guard nothing more than a widened one.

//go:noinline
func range_violation[Value Integer](builder Assertion_Builder, value Value, minimum Value) {
	identity, reason := RANGE_GUARD_MAXIMUM, "  value exceeds max"
	if value < minimum {
		identity, reason = RANGE_GUARD_MINIMUM, "  value below min"
	}
	failure := Assertion_Failure{Identity: identity, Reason: reason, Value: value}
	production_fatal(builder, failure)
	panic(failure)
}

//go:noinline
func range_holed_violation[Value Integer](
	builder Assertion_Builder, value Value, minimum Value, maximum Value,
) {
	if value < minimum {
		range_violation(builder, value, minimum)
	}
	if value > maximum {
		range_violation(builder, value, minimum)
	}
	failure := Assertion_Failure{Identity: "Range value is excluded", Value: value}
	production_fatal(builder, failure)
	panic(failure)
}

//go:noinline
func enum_violation[Value Integer](builder Assertion_Builder, value Value) {
	failure := Assertion_Failure{
		Identity: ENUM_GUARD_MEMBER,
		Reason:   "  value is not a member",
		Value:    value,
	}
	production_fatal(builder, failure)
	panic(failure)
}
