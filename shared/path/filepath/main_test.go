package filepath_test

import (
	"testing"

	"local/james-orcales/shared/invariant/default"
)

// TestMain exists because domain gaps must fail same gate as behavior.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}
