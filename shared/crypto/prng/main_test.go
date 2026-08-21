package prng

import (
	"testing"

	"local/james-orcales/shared/invariant/default"
)

// TestMain registers this package's assertions with the invariant coverage recorder and reports any
// gaps after the suite runs.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}
