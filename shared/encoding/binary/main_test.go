package binary_test

import (
	"testing"

	"local/james-orcales/shared/invariant/default"
)

// TestMain registers package invariant roots before suites run.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}
