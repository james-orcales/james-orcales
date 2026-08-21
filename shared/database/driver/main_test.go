package driver_test

import (
	"testing"

	"local/james-orcales/shared/sim/aver/default"
)

// TestMain registers driver invariant roots before specification run.
func TestMain(m *testing.M) {
	aver.Run_Test_Main(m)
}
