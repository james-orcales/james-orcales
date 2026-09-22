package types_test

import "testing"

import "local/james-orcales/shared/sim/aver/default"

// TestMain lets production entry points prove every registered type domain.
func TestMain(m *testing.M) {
  aver.Run_Test_Main(m)
}
