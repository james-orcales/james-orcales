package tar_test

import (
	"testing"

	"local/james-orcales/shared/simulation/aver/default"
)

// TestMain registers TAR invariant roots before specification runs.
func TestMain(m *testing.M) {
	aver.Run_Test_Main(m)
}
