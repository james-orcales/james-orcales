//go:build invariant_noop

package aver

// Assertion_Builder carries nothing in the noop build. A link returns the value it was handed, so
// an empty builder makes every fluent return free and lets the inliner fold a whole chain away.
// That is the point of this build: it measures what a body costs without its assertions.
type Assertion_Builder struct{}

// Recorder_Register_Packages_For_Analysis stays inert because this build can observe no registered
// obligation and exists only to measure code without assertions.
func Recorder_Register_Packages_For_Analysis(recorder *Recorder, directories ...string) { return }
