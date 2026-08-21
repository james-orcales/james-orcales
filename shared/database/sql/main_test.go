package sql_test

import (
	"testing"

	"local/james-orcales/shared/sim/aver/default"
)

// TestMain registers SQL invariant roots before specification run.
func TestMain(m *testing.M) {
	aver.Run_Test_Main(m)
}
