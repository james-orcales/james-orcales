package hex_test

import (
	"testing"

	"local/james-orcales/shared/sim/aver/default"
)

// TestMain registers production paths so generated inputs cannot bypass public contracts.
func TestMain(m *testing.M) {
	aver.Run_Test_Main(m)
}
