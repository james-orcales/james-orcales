//go:build !noassert

// This file is the enforcing half of a compile-time dual. It holds every entry point that runs in
// a shipped binary — the eager guard, the chain root, and the fluent links that validate, enforce,
// and credit — while enforcement_noassert.go holds a signature-identical set of no-ops selected by
// `-tags noassert`. The registration and analysis machinery stays in invariant.go untagged because
// it runs only under `go test`, where the tag is never set.
//
// The split is what makes the tag honest. A shared body guarded by a build-time constant would
// leave the real code in the binary and make elimination a question about the inliner's budget;
// here the disabled build compiles bodies that are literally empty, so "off" costs nothing by
// construction rather than by the compiler's discretion.

package invariant

import "strings"

// Recorder_Always stays outside Product because an eager guard has no second branch to widen a
// demanded grid. It panics immediately when condition is false in every run mode; under a plain
// test run it also credits reachability so an uncalled guard remains visible as a gap.
func Recorder_Always[T ~bool](recorder *Recorder, condition T, message string) {
	if !condition {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX + message + "  Always — condition was false")
	}
	// Enforcement (the panic above) runs in every mode; coverage is credited under a test run
	// or the fuzz coordinator (not a worker), matching Ensure's recording policy. The
	// reachability entry is seeded statically by recorder_register_eager_always;
	// recorder_increment no-ops when the bare Always was never registered (an Always in a
	// non-analyzed package).
	if !recorder.Is_Test {
		return
	}
	if recorder.Is_Benchmark {
		return
	}
	recorder_increment(recorder, message, true)
}

// Recorder_Dot_Product starts one demanded chain under namespace.
func Recorder_Dot_Product(recorder *Recorder, namespace Namespace) (product Product) {
	if strings.Contains(string(namespace), ELEMENT_MESSAGE_SEPARATOR) {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX + "Dot_Product namespace contains NUL")
	}
	return Product{
		Recorder:  recorder,
		Shape:     recorder_chain_shape(recorder, namespace),
		Namespace: namespace,
	}
}

// Sometimes only advances the builder; Ensure owns validation and coverage mutation.
func (product Product) Sometimes(condition bool, message string) (next Product) {
	if product.Failure != 0 {
		return product
	}
	if product.Ordinal == CHAIN_LINKS_MAX {
		return product.chain_defer_failure(PRODUCT_FAILURE_LINKS)
	}
	if strings.Contains(message, ELEMENT_MESSAGE_SEPARATOR) {
		return product.chain_defer_failure(PRODUCT_FAILURE_SOMETIMES)
	}
	mismatch := product.Shape.chain_axis(product.Ordinal, product.Axis_Count, message)
	if condition {
		product.Observations = chain_mask_with(product.Observations, product.Ordinal)
	}
	product.Ordinal++
	product.Axis_Count++
	if mismatch {
		product.Mismatch = true
	}
	return product
}

// Impossible only advances the builder; Ensure owns validation and enforcement.
func (product Product) Impossible(
	message string, references ...Dot_Element_Reference,
) (next Product) {
	if product.Failure != 0 {
		return product
	}
	if product.Ordinal == CHAIN_LINKS_MAX {
		return product.chain_defer_failure(PRODUCT_FAILURE_LINKS)
	}
	if len(references) == 0 {
		return product.chain_defer_failure(PRODUCT_FAILURE_IMPOSSIBLE)
	}
	if len(references) > CHAIN_LINKS_MAX {
		return product.chain_defer_failure(PRODUCT_FAILURE_IMPOSSIBLE)
	}
	if product.Shape.chain_replays() {
		rule, mismatch, failure := product.Shape.chain_rule_replay(
			product.Ordinal, product.Axis_Count, message, references)
		if failure != 0 {
			product = product.chain_defer_failure(failure)
		}
		return product.impossible_advance(rule, mismatch)
	}
	link := Chain_Link{
		Kind: DOT_ELEMENT_KIND_IMPOSSIBLE, Ordinal: product.Ordinal,
		Axis_Count: product.Axis_Count, Message: message,
	}
	rule, mismatch, failure := product.Shape.chain_rule(link, references)
	if failure != 0 {
		product = product.chain_defer_failure(failure)
	}
	return product.impossible_advance(rule, mismatch)
}

// Range_Int expands the int bounded preset inside this product.
func (product Product) Range_Int(
	value int, minimum int, maximum int, excluded ...int,
) (next Product) {
	return product_range(product, CHAIN_INTEGER_KIND_INT, value, minimum, maximum, excluded)
}

// Range_Int8 expands the int8 bounded preset inside this product.
func (product Product) Range_Int8(
	value int8, minimum int8, maximum int8, excluded ...int8,
) (next Product) {
	return product_range(product, CHAIN_INTEGER_KIND_INT8, value, minimum, maximum, excluded)
}

// Range_Int16 expands the int16 bounded preset inside this product.
func (product Product) Range_Int16(
	value int16, minimum int16, maximum int16, excluded ...int16,
) (next Product) {
	return product_range(product, CHAIN_INTEGER_KIND_INT16, value, minimum, maximum, excluded)
}

// Range_Int32 expands the int32 bounded preset inside this product.
func (product Product) Range_Int32(
	value int32, minimum int32, maximum int32, excluded ...int32,
) (next Product) {
	return product_range(product, CHAIN_INTEGER_KIND_INT32, value, minimum, maximum, excluded)
}

// Range_Int64 expands the int64 bounded preset inside this product.
func (product Product) Range_Int64(
	value int64, minimum int64, maximum int64, excluded ...int64,
) (next Product) {
	return product_range(product, CHAIN_INTEGER_KIND_INT64, value, minimum, maximum, excluded)
}

// Range_Uint expands the uint bounded preset inside this product.
func (product Product) Range_Uint(
	value uint, minimum uint, maximum uint, excluded ...uint,
) (next Product) {
	return product_range(product, CHAIN_INTEGER_KIND_UINT, value, minimum, maximum, excluded)
}

// Range_Uint8 expands the uint8 bounded preset inside this product.
func (product Product) Range_Uint8(
	value uint8, minimum uint8, maximum uint8, excluded ...uint8,
) (next Product) {
	return product_range(product, CHAIN_INTEGER_KIND_UINT8, value, minimum, maximum, excluded)
}

// Range_Uint16 expands the uint16 bounded preset inside this product.
func (product Product) Range_Uint16(
	value uint16, minimum uint16, maximum uint16, excluded ...uint16,
) (next Product) {
	return product_range(product, CHAIN_INTEGER_KIND_UINT16, value, minimum, maximum, excluded)
}

// Range_Uint32 expands the uint32 bounded preset inside this product.
func (product Product) Range_Uint32(
	value uint32, minimum uint32, maximum uint32, excluded ...uint32,
) (next Product) {
	return product_range(product, CHAIN_INTEGER_KIND_UINT32, value, minimum, maximum, excluded)
}

// Range_Uint64 expands the uint64 bounded preset inside this product.
func (product Product) Range_Uint64(
	value uint64, minimum uint64, maximum uint64, excluded ...uint64,
) (next Product) {
	return product_range(product, CHAIN_INTEGER_KIND_UINT64, value, minimum, maximum, excluded)
}

// Enum_Int expands the int member-set preset inside this product.
func (product Product) Enum_Int(value int, members ...int) (next Product) {
	if !chain_enum_domain_valid(members) {
		return product.chain_defer_failure(PRODUCT_FAILURE_ENUM_DOMAIN)
	}
	minimum, maximum := chain_integer_bounds(members)
	return product_preset(product, CHAIN_PRESET_KIND_ENUM, CHAIN_INTEGER_KIND_INT,
		value, minimum, maximum, members)
}

// Enum_Int8 expands the int8 member-set preset inside this product.
func (product Product) Enum_Int8(value int8, members ...int8) (next Product) {
	if !chain_enum_domain_valid(members) {
		return product.chain_defer_failure(PRODUCT_FAILURE_ENUM_DOMAIN)
	}
	minimum, maximum := chain_integer_bounds(members)
	return product_preset(product, CHAIN_PRESET_KIND_ENUM, CHAIN_INTEGER_KIND_INT8,
		value, minimum, maximum, members)
}

// Enum_Int16 expands the int16 member-set preset inside this product.
func (product Product) Enum_Int16(value int16, members ...int16) (next Product) {
	if !chain_enum_domain_valid(members) {
		return product.chain_defer_failure(PRODUCT_FAILURE_ENUM_DOMAIN)
	}
	minimum, maximum := chain_integer_bounds(members)
	return product_preset(product, CHAIN_PRESET_KIND_ENUM, CHAIN_INTEGER_KIND_INT16,
		value, minimum, maximum, members)
}

// Enum_Int32 expands the int32 member-set preset inside this product.
func (product Product) Enum_Int32(value int32, members ...int32) (next Product) {
	if !chain_enum_domain_valid(members) {
		return product.chain_defer_failure(PRODUCT_FAILURE_ENUM_DOMAIN)
	}
	minimum, maximum := chain_integer_bounds(members)
	return product_preset(product, CHAIN_PRESET_KIND_ENUM, CHAIN_INTEGER_KIND_INT32,
		value, minimum, maximum, members)
}

// Enum_Int64 expands the int64 member-set preset inside this product.
func (product Product) Enum_Int64(value int64, members ...int64) (next Product) {
	if !chain_enum_domain_valid(members) {
		return product.chain_defer_failure(PRODUCT_FAILURE_ENUM_DOMAIN)
	}
	minimum, maximum := chain_integer_bounds(members)
	return product_preset(product, CHAIN_PRESET_KIND_ENUM, CHAIN_INTEGER_KIND_INT64,
		value, minimum, maximum, members)
}

// Enum_Uint expands the uint member-set preset inside this product.
func (product Product) Enum_Uint(value uint, members ...uint) (next Product) {
	if !chain_enum_domain_valid(members) {
		return product.chain_defer_failure(PRODUCT_FAILURE_ENUM_DOMAIN)
	}
	minimum, maximum := chain_integer_bounds(members)
	return product_preset(product, CHAIN_PRESET_KIND_ENUM, CHAIN_INTEGER_KIND_UINT,
		value, minimum, maximum, members)
}

// Enum_Uint8 expands the uint8 member-set preset inside this product.
func (product Product) Enum_Uint8(value uint8, members ...uint8) (next Product) {
	if !chain_enum_domain_valid(members) {
		return product.chain_defer_failure(PRODUCT_FAILURE_ENUM_DOMAIN)
	}
	minimum, maximum := chain_integer_bounds(members)
	return product_preset(product, CHAIN_PRESET_KIND_ENUM, CHAIN_INTEGER_KIND_UINT8,
		value, minimum, maximum, members)
}

// Enum_Uint16 expands the uint16 member-set preset inside this product.
func (product Product) Enum_Uint16(value uint16, members ...uint16) (next Product) {
	if !chain_enum_domain_valid(members) {
		return product.chain_defer_failure(PRODUCT_FAILURE_ENUM_DOMAIN)
	}
	minimum, maximum := chain_integer_bounds(members)
	return product_preset(product, CHAIN_PRESET_KIND_ENUM, CHAIN_INTEGER_KIND_UINT16,
		value, minimum, maximum, members)
}

// Enum_Uint32 expands the uint32 member-set preset inside this product.
func (product Product) Enum_Uint32(value uint32, members ...uint32) (next Product) {
	if !chain_enum_domain_valid(members) {
		return product.chain_defer_failure(PRODUCT_FAILURE_ENUM_DOMAIN)
	}
	minimum, maximum := chain_integer_bounds(members)
	return product_preset(product, CHAIN_PRESET_KIND_ENUM, CHAIN_INTEGER_KIND_UINT32,
		value, minimum, maximum, members)
}

// Enum_Uint64 expands the uint64 member-set preset inside this product.
func (product Product) Enum_Uint64(value uint64, members ...uint64) (next Product) {
	if !chain_enum_domain_valid(members) {
		return product.chain_defer_failure(PRODUCT_FAILURE_ENUM_DOMAIN)
	}
	minimum, maximum := chain_integer_bounds(members)
	return product_preset(product, CHAIN_PRESET_KIND_ENUM, CHAIN_INTEGER_KIND_UINT64,
		value, minimum, maximum, members)
}

// Ensure validates the complete shape, enforces every carve, and credits the packed tuple.
func (product Product) Ensure() {
	product.ensure_shape()
	axis_count := product.Axis_Count
	records := recorder_chain_records(product.Recorder)
	if !product.Shape.Registered {
		records = false
	}
	if len(product.Shape.Rules) == 0 {
		if !records {
			return
		}
	}
	tuple_mask := product.Shape.chain_tuple(product.Observations)
	var violations []string
	for _, rule := range product.Shape.Rules {
		matches := true
		for i_index := range tuple_mask {
			if tuple_mask[i_index]&rule.Mask[i_index] != rule.Want[i_index] {
				matches = false
				break
			}
		}
		if matches {
			violations = append(violations, rule.Message)
		}
	}
	if len(violations) > 0 {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX + strings.Join(violations, "\n"))
	}
	if !records {
		return
	}
	tuple := product.chain_handle(tuple_mask)
	if axis_count > 0 {
		if tuple.Metadata == nil {
			panic(ASSERTION_FAILURE_MESSAGE_PREFIX +
				"registered Dot_Product credited unknown tuple")
		}
	}
	for _, guard := range product.Shape.Guards {
		recorder_increment_entry(product.Recorder, guard.Entry, true)
	}
	for i_index := uint8(0); i_index < axis_count; i_index++ {
		axis := product.Shape.Axes[i_index]
		condition := chain_mask_has(product.Observations, axis.Ordinal)
		recorder_increment_entry(product.Recorder, axis.Entry, condition)
	}
	if axis_count > 0 {
		recorder_increment_entry(product.Recorder, tuple, true)
	}
}
