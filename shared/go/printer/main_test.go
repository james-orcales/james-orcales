package printer_test

import (
	"testing"

	"local/james-orcales/shared/simulation/aver/default"
)

// TestMain lets production entry points prove every registered printer domain.
func TestMain(m *testing.M) {
	aver.Run_Test_Main(m)
}
