package prng

import "testing"

import "local/james-orcales/shared/std/sim/aver/default"

// TestMain registers this package's assertions with the invariant coverage recorder and reports any
// gaps after the suite runs.
func TestMain(m *testing.M) {
  aver.Run_Test_Main(m)
}
