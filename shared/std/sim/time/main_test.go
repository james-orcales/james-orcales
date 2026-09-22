package time_test

import "testing"

import "local/james-orcales/shared/std/sim/aver/default"

// TestMain register package invariant roots before specification run.
func TestMain(m *testing.M) {
  aver.Run_Test_Main(m)
}
