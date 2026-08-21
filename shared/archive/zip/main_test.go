package zip_test

import (
	"testing"

	"local/james-orcales/shared/invariant/default"
)

// TestMain keeps invariant coverage load-bearing for ZIP parsing.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}
