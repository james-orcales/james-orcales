package driver_test

import (
	"testing"

	"local/james-orcales/shared/invariant/default"
)

// TestMain registers driver invariant roots before specification run.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}
