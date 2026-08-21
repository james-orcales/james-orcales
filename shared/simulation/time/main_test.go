package time_test

import (
	"testing"

	invariant "local/james-orcales/shared/invariant/default"
)

// TestMain register package invariant roots before specification run.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}
