package io_test

import (
	"testing"

	invariant "local/james-orcales/shared/invariant/default"
)

// TestMain registers the completion-machine assertions. Thus, an unused legal edge fails the suite.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}
