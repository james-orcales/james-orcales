package nbio_test

import (
	"testing"

	invariant "local/james-orcales/shared/invariant/default"
)

// TestMain register completion-machine assertions. Thus unused legal edge fail suite.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}
