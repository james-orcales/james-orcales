package io_test

import (
	"testing"

	invariant "local/james-orcales/g/shared/invariant/default"
)

// TestMain runs the suite through the invariant harness so the completion machine's
// coverage grid is registered and analyzed: an edge of the lifecycle machine this suite
// never witnesses fails the run instead of silently going uncovered.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}
