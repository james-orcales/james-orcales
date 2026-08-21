package ucd_test

import (
	"testing"

	"local/james-orcales/shared/simulation/aver/default"
)

// TestMain registers the package invariant roots before the specification runs.
func TestMain(m *testing.M) {
	aver.Run_Test_Main(m)
}
