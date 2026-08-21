package binary_test

import (
	"testing"

	"local/james-orcales/shared/simulation/aver/default"
)

// TestMain registers package invariant roots before suites run.
func TestMain(m *testing.M) {
	aver.Run_Test_Main(m)
}
