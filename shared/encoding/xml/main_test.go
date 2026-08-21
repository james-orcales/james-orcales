package xml_test

import (
	"testing"

	"local/james-orcales/shared/invariant/default"
)

// TestMain keeps invariant coverage on production-reachable package paths.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}
