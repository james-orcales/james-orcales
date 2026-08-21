package adler32_test

import (
	"testing"

	"local/james-orcales/shared/invariant/default"
)

// TestMain makes public operations prove their invariant paths.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}
