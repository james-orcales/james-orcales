package prng_test

import "testing"

import "local/james-orcales/shared/sim/aver/default"

// TestMain makes bounded random domains fail when public tests miss a witness.
func TestMain(m *testing.M) {
  aver.Run_Test_Main(m)
}
