package myers_test

import (
	"testing"

	invariant "github.com/james-orcales/james-orcales/shared/invariant/default"
)

// TestMain runs the suite under the invariant framework so that Always and
// Sometimes coverage is enforced across every test.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}
