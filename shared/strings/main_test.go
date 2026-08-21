package strings_test

import (
	"testing"

	"local/james-orcales/shared/invariant/default"
)

// TestMain lets invariant coverage reject unobserved branches.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}
