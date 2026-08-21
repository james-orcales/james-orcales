package template_test

import (
	"testing"

	"local/james-orcales/shared/invariant/default"
)

// TestMain keeps allocation proof on enforcing runtime paths.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}
