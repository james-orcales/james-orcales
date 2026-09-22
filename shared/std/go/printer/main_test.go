package printer_test

import "testing"

import "local/james-orcales/shared/std/sim/aver/default"

// TestMain lets production entry points prove every registered printer domain.
func TestMain(m *testing.M) {
  aver.Run_Test_Main(m)
}
