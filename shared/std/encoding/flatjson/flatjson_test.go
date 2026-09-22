package flatjson_test

import "testing"

import "local/james-orcales/shared/std/sim/aver/default"

// TestMain runs the suite under the invariant framework so the Always assertions in
// shared/std/encoding/flatjson are coverage-checked: any the tests never reach fails the run. It
// lives
// in its own file so specification_test.go keeps its spec-leaf tests at the very top.
func TestMain(m *testing.M) {
  aver.Run_Test_Main(m)
}
