package filepath_test

import "testing"

import "local/james-orcales/shared/std/sim/aver/default"

// TestMain exists because domain gaps must fail same gate as behavior.
func TestMain(m *testing.M) {
  aver.Run_Test_Main(m)
}
