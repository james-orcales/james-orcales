package path_test

import (
	"testing"

	"local/james-orcales/shared/invariant/default"
)

// TestMain lets production entry points prove every registered path domain.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}
