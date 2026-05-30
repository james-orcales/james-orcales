package myers_test

import (
	"testing"

	"github.com/james-orcales/james-orcales/shared/invariant"
)

// TestMain runs the suite under the invariant framework so that Always and
// Sometimes coverage is enforced across every test.
func TestMain(m *testing.M) {
	invariant.RunTestMain(m)
}
