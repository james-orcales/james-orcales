//go:build !invariant_noop

package invariant

import "unsafe"

// Assertion_Builder is the complete deferred runtime state. Fixed observations make escape and
// allocation unnecessary even at the largest supported chain.
type Assertion_Builder struct {
	// Context holds ordinary namespace bytes or a recording plan. State_A and State_B interpret
	// that pointer as a tagged union: namespace length plus failure for enforcement, or packed
	// observations, ordinal, and failure for recording. The states are mutually exclusive, so
	// carrying both representations through every fluent return would be pure overhead.
	Context unsafe.Pointer
	// State_A holds the namespace length or the first 64 observations selected by Context.
	State_A uintptr
	// State_B holds the union tag, deferred failure, ordinal, and remaining observations.
	State_B uintptr
}

// Recorder_Register_Packages_For_Analysis keeps source analysis outside noop benchmarks because
// registration cost belongs only to builds that can observe registered obligations.
func Recorder_Register_Packages_For_Analysis(recorder *Recorder, directories ...string) {
	recorder_register_packages_for_analysis(recorder, directories...)
}
