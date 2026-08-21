//go:build invariant_noop

package invariant

// Assertion_Builder carries nothing in the noop build. A link returns the value it was handed, so
// an empty builder makes every fluent return free and lets the inliner fold a whole chain away.
// That is the point of this build: it measures what a body costs without its assertions.
type Assertion_Builder struct{}
