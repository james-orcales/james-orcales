package filepath_test

import (
	"testing"

	"local/james-orcales/shared/simulation/aver/default"
)

// TestMain exists because domain gaps must fail same gate as behavior.
func TestMain(m *testing.M) {
	aver.Run_Test_Main(m)
}
