package xxhash_test

import (
	"testing"

	"local/james-orcales/shared/invariant/default"
)

// TestMain makes public operations prove invariant paths.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}
