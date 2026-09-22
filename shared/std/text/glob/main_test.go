package glob_test

import "testing"

import "local/james-orcales/shared/sim/aver/default"

// TestMain keeps allocation proof on enforcing runtime paths.
func TestMain(m *testing.M) {
  aver.Run_Test_Main(m)
}
