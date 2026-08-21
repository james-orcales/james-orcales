package myers_test

import (
	"testing"

	"local/james-orcales/shared/simulation/aver/default"
)

// TestMain runs the suite under the invariant framework so that Always and
// Sometimes coverage is enforced across every test.
func TestMain(m *testing.M) {
	aver.Run_Test_Main(m)
}
