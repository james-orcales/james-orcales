package tar_test

import (
	"testing"

	"local/james-orcales/shared/invariant/default"
)

// TestMain registers TAR invariant roots before specification runs.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}
