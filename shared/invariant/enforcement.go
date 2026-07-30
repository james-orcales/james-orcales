//go:build !noassert

// The enforcing build keeps fluent links observationally silent: they only advance a value and
// latch raw verdicts. Ensure is the single boundary that can panic or mutate coverage.
package invariant

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
	builder.Recorder = recorder
	builder.Namespace = namespace
	if !recorder.Is_Test {
		return builder
	}
	if recorder.Is_Benchmark {
		return builder
	}
	builder.Plan = recorder.Assertion_Plans[namespace]
	return builder
}

// Sometimes captures one branch by expanded ordinal. The message belongs exclusively to the
// registration plan and is intentionally unread here, preventing runtime identity reconstruction.
func (builder Assertion_Builder) Sometimes(
	condition bool, message string,
) (next Assertion_Builder) {
	return builder.assertion_axis(condition)
}

// Range_Int captures the int bounded-domain assertion for Ensure.
func (builder Assertion_Builder) Range_Int(
	value int, minimum int, maximum int, excluded ...int,
) (next Assertion_Builder) {
	return assertion_range(builder, value, minimum, maximum, excluded)
}

// Range_Int8 captures the int8 bounded-domain assertion for Ensure.
func (builder Assertion_Builder) Range_Int8(
	value int8, minimum int8, maximum int8, excluded ...int8,
) (next Assertion_Builder) {
	return assertion_range(builder, value, minimum, maximum, excluded)
}

// Range_Int16 captures the int16 bounded-domain assertion for Ensure.
func (builder Assertion_Builder) Range_Int16(
	value int16, minimum int16, maximum int16, excluded ...int16,
) (next Assertion_Builder) {
	return assertion_range(builder, value, minimum, maximum, excluded)
}

// Range_Int32 captures the int32 bounded-domain assertion for Ensure.
func (builder Assertion_Builder) Range_Int32(
	value int32, minimum int32, maximum int32, excluded ...int32,
) (next Assertion_Builder) {
	return assertion_range(builder, value, minimum, maximum, excluded)
}

// Range_Int64 captures the int64 bounded-domain assertion for Ensure.
func (builder Assertion_Builder) Range_Int64(
	value int64, minimum int64, maximum int64, excluded ...int64,
) (next Assertion_Builder) {
	return assertion_range(builder, value, minimum, maximum, excluded)
}

// Range_Uint captures the uint bounded-domain assertion for Ensure.
func (builder Assertion_Builder) Range_Uint(
	value uint, minimum uint, maximum uint, excluded ...uint,
) (next Assertion_Builder) {
	return assertion_range(builder, value, minimum, maximum, excluded)
}

// Range_Uint8 captures the uint8 bounded-domain assertion for Ensure.
func (builder Assertion_Builder) Range_Uint8(
	value uint8, minimum uint8, maximum uint8, excluded ...uint8,
) (next Assertion_Builder) {
	return assertion_range(builder, value, minimum, maximum, excluded)
}

// Range_Uint16 captures the uint16 bounded-domain assertion for Ensure.
func (builder Assertion_Builder) Range_Uint16(
	value uint16, minimum uint16, maximum uint16, excluded ...uint16,
) (next Assertion_Builder) {
	return assertion_range(builder, value, minimum, maximum, excluded)
}

// Range_Uint32 captures the uint32 bounded-domain assertion for Ensure.
func (builder Assertion_Builder) Range_Uint32(
	value uint32, minimum uint32, maximum uint32, excluded ...uint32,
) (next Assertion_Builder) {
	return assertion_range(builder, value, minimum, maximum, excluded)
}

// Range_Uint64 captures the uint64 bounded-domain assertion for Ensure.
func (builder Assertion_Builder) Range_Uint64(
	value uint64, minimum uint64, maximum uint64, excluded ...uint64,
) (next Assertion_Builder) {
	return assertion_range(builder, value, minimum, maximum, excluded)
}

// Enum_Int captures the int member-domain assertion for Ensure.
func (builder Assertion_Builder) Enum_Int(
	value int, members ...int,
) (next Assertion_Builder) {
	return assertion_enum(builder, value, members)
}

// Enum_Int8 captures the int8 member-domain assertion for Ensure.
func (builder Assertion_Builder) Enum_Int8(
	value int8, members ...int8,
) (next Assertion_Builder) {
	return assertion_enum(builder, value, members)
}

// Enum_Int16 captures the int16 member-domain assertion for Ensure.
func (builder Assertion_Builder) Enum_Int16(
	value int16, members ...int16,
) (next Assertion_Builder) {
	return assertion_enum(builder, value, members)
}

// Enum_Int32 captures the int32 member-domain assertion for Ensure.
func (builder Assertion_Builder) Enum_Int32(
	value int32, members ...int32,
) (next Assertion_Builder) {
	return assertion_enum(builder, value, members)
}

// Enum_Int64 captures the int64 member-domain assertion for Ensure.
func (builder Assertion_Builder) Enum_Int64(
	value int64, members ...int64,
) (next Assertion_Builder) {
	return assertion_enum(builder, value, members)
}

// Enum_Uint captures the uint member-domain assertion for Ensure.
func (builder Assertion_Builder) Enum_Uint(
	value uint, members ...uint,
) (next Assertion_Builder) {
	return assertion_enum(builder, value, members)
}

// Enum_Uint8 captures the uint8 member-domain assertion for Ensure.
func (builder Assertion_Builder) Enum_Uint8(
	value uint8, members ...uint8,
) (next Assertion_Builder) {
	return assertion_enum(builder, value, members)
}

// Enum_Uint16 captures the uint16 member-domain assertion for Ensure.
func (builder Assertion_Builder) Enum_Uint16(
	value uint16, members ...uint16,
) (next Assertion_Builder) {
	return assertion_enum(builder, value, members)
}

// Enum_Uint32 captures the uint32 member-domain assertion for Ensure.
func (builder Assertion_Builder) Enum_Uint32(
	value uint32, members ...uint32,
) (next Assertion_Builder) {
	return assertion_enum(builder, value, members)
}

// Enum_Uint64 captures the uint64 member-domain assertion for Ensure.
func (builder Assertion_Builder) Enum_Uint64(
	value uint64, members ...uint64,
) (next Assertion_Builder) {
	return assertion_enum(builder, value, members)
}

// Ensure is the only fluent operation allowed to panic or credit. It preflights the whole plan so
// an invalid execution can never leave a misleading partially-covered chain.
func (builder Assertion_Builder) Ensure() {
	if builder.Failure != ASSERTION_FAILURE_NONE {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX + builder.assertion_failure_message())
	}
	if builder.Plan == nil {
		return
	}
	if len(builder.Plan.Links) != int(builder.Ordinal) {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX +
			"registered Assertions chain differs from its registration plan")
	}
	for _, link := range builder.Plan.Links {
		if link.Entry.Metadata == nil {
			panic(ASSERTION_FAILURE_MESSAGE_PREFIX +
				"registered Assertions chain resolved an unknown coverage handle")
		}
	}
	for _, link := range builder.Plan.Links {
		condition := true
		if link.Kind == ASSERTION_KIND_SOMETIMES {
			condition = builder.assertion_observed(link.Ordinal)
		}
		recorder_increment_entry(builder.Recorder, link.Entry, condition)
	}
}

func (builder Assertion_Builder) assertion_axis(condition bool) (next Assertion_Builder) {
	if builder.Ordinal >= ASSERTION_LINKS_MAX {
		return builder.assertion_fail(ASSERTION_FAILURE_LINKS)
	}
	if condition {
		builder.Observations[builder.Ordinal/64] |= uint64(1) << (builder.Ordinal % 64)
	}
	builder.Ordinal++
	return builder
}

func (builder Assertion_Builder) assertion_guard() (next Assertion_Builder) {
	if builder.Ordinal >= ASSERTION_LINKS_MAX {
		return builder.assertion_fail(ASSERTION_FAILURE_LINKS)
	}
	builder.Ordinal++
	return builder
}

func (builder Assertion_Builder) assertion_observed(ordinal uint8) (observed bool) {
	return builder.Observations[ordinal/64]&(uint64(1)<<(ordinal%64)) != 0
}

func (builder Assertion_Builder) assertion_fail(failure uint8) (next Assertion_Builder) {
	if builder.Failure == ASSERTION_FAILURE_NONE {
		builder.Failure = failure
	}
	return builder
}

func (builder Assertion_Builder) assertion_failure_message() (message string) {
	prefix := string(builder.Namespace) + ELEMENT_MESSAGE_SEPARATOR
	switch builder.Failure {
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

func assertion_range[Value Integer](
	builder Assertion_Builder, value Value, minimum Value, maximum Value, excluded []Value,
) (next Assertion_Builder) {
	builder = builder.assertion_guard()
	builder = builder.assertion_guard()
	if minimum > maximum {
		builder = builder.assertion_fail(ASSERTION_FAILURE_RANGE_DOMAIN)
	}
	if value < minimum {
		builder = builder.assertion_fail(ASSERTION_FAILURE_RANGE_LOWER)
	}
	if value > maximum {
		builder = builder.assertion_fail(ASSERTION_FAILURE_RANGE_UPPER)
	}
	for _, hole := range excluded {
		outside := hole <= minimum
		if hole >= maximum {
			outside = true
		}
		if outside {
			builder = builder.assertion_fail(ASSERTION_FAILURE_RANGE_EXCLUSION)
		}
		if value == hole {
			builder = builder.assertion_fail(ASSERTION_FAILURE_RANGE_EXCLUDED)
		}
	}
	if minimum == maximum {
		return builder
	}
	builder = builder.assertion_axis(value == minimum)
	builder = builder.assertion_axis(value == maximum)
	builder = assertion_range_candidate(builder, value, minimum, maximum, excluded, Value(0))
	builder = assertion_range_candidate(builder, value, minimum, maximum, excluded, Value(1))
	builder = assertion_range_candidate(builder, value, minimum, maximum, excluded, Value(2))
	zero := Value(0)
	negative_one := zero - Value(1)
	if negative_one < zero {
		builder = assertion_range_candidate(
			builder, value, minimum, maximum, excluded, negative_one)
	}
	return builder
}

func assertion_range_candidate[Value Integer](
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

func assertion_enum[Value Integer](
	builder Assertion_Builder, value Value, members []Value,
) (next Assertion_Builder) {
	builder = builder.assertion_guard()
	distinct := 0
	matched := false
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
		distinct++
		condition := value == member
		if condition {
			matched = true
		}
		builder = builder.assertion_axis(condition)
	}
	if distinct < 2 {
		builder = builder.assertion_fail(ASSERTION_FAILURE_ENUM_DOMAIN)
	}
	if !matched {
		builder = builder.assertion_fail(ASSERTION_FAILURE_ENUM_MEMBER)
	}
	return builder
}
