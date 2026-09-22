package simulation_test

import (
	"testing"

	"local/james-orcales/shared/std/sim/aver/default"
)

// TestMain wires the coverage recorder over the internal tree ("../**"): every
// Always reached must hold and every Sometimes both fire, or the run fails.
func TestMain(m *testing.M) {
	aver.Run_Test_Main(m, "../**")
}
