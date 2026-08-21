package ascii85_test

import (
	"testing"

	"local/james-orcales/shared/invariant/default"
)

// TestMain registers production paths so generated inputs cannot bypass public contracts.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}
