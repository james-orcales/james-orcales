//go:build !invariant_noop

package invariant

import "local/james-orcales/shared/invariant"

// Always is an eager guard, so its enforcement and reachability stay independent of any chain.
func Always[T ~bool](condition T, message string) {
	invariant.Recorder_Always(Default, condition, message)
}

// Sometimes states one inline two-branch axis on Default, for a body that owns no bundle.
func Sometimes[T ~bool](condition T, message string) {
	invariant.Recorder_Sometimes(Default, condition, message)
}

// Range bounds one inline value to a contiguous interval on Default.
func Range[Value invariant.Integer](
	value Value, minimum Value, maximum Value, message string,
) {
	invariant.Recorder_Range(Default, value, minimum, maximum, message)
}

// Enum holds one inline value to two members on Default.
func Enum[Value invariant.Integer](
	value Value, first Value, second Value, message string,
) {
	invariant.Recorder_Enum(Default, value, first, second, message)
}

// Range_Holed removes four fixed exclusions from an inline interval on Default. A duplicate final
// hole stands for an unused slot, so one arity serves every width.
func Range_Holed[Value invariant.Integer](
	value Value, minimum Value, maximum Value,
	hole_1 Value, hole_2 Value, hole_3 Value, hole_4 Value, message string,
) {
	invariant.Recorder_Range_Holed(
		Default, value, minimum, maximum, hole_1, hole_2, hole_3, hole_4, message)
}

// Tree starts one deferred builder on Default. The subject supplies the chain type.
func Tree[Subject any](
	subject Subject, namespace Namespace,
) (builder Assertion_Builder) {
	return invariant.Recorder_Tree(Default, subject, namespace)
}

// Assertion_Builder re-exports the fluent value for explicit APIs without adding an adapter.
type Assertion_Builder = invariant.Assertion_Builder
