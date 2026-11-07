package simulation_test

import "testing"

import "local/james-orcales/shared/std/sim/aver/default"

// TestMain registers the pkg tree, so every Always reached must hold and every Sometimes fire.
func TestMain(m *testing.M) {
  aver.Run_Test_Main(m, "../**")
}
