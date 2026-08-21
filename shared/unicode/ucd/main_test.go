package ucd_test

import (
	"testing"

	"local/james-orcales/shared/invariant/default"
)

// TestMain registers the package invariant roots before the specification runs.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}
