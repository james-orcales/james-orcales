package scanner_test

import (
	"testing"

	"local/james-orcales/shared/sim/aver/default"
)

// TestMain keeps allocation probes on production assertion paths.
func TestMain(m *testing.M) {
	aver.Run_Test_Main(m)
}
