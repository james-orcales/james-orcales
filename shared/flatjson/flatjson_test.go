package flatjson_test

import (
	"testing"

	invariant "github.com/james-orcales/james-orcales/shared/invariant/default"
)

// TestMain runs the suite under the invariant framework so the Always assertions in
// shared/flatjson are coverage-checked: any the tests never reach fails the run. It lives
// in its own file so specification_test.go keeps its spec-leaf tests at the very top.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}
