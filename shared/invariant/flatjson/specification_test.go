//go:build !invariant_disable_coverage && !prd && !prod && !production && !invariant_noop

package invariant_flatjson

import "testing"

// Test_Always verifies that the encoder assertion seam retains failure enforcement.
func Test_Always(t *testing.T) {
	const MESSAGE = "The flat JSON encoder assertion failed."
	Always(true, MESSAGE)
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("a false encoder assertion did not panic")
		}
	}()
	Always(false, MESSAGE)
}
