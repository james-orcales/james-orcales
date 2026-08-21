package aver_flatjson_test

import (
	"testing"

	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/sim/aver/flatjson"
	"local/james-orcales/shared/testify"
)

// TestMain runs this assertion seam through the default test entrypoint.
func TestMain(m *testing.M) {
	aver.Run_Test_Main(m)
}

// Successful encoder guards stay on each encoded field's hot path.
func Test_Always_Has_Zero_Allocations(t *testing.T) {
	testify.Zero_Allocation(t, func() {
		aver_flatjson.Always(true, "guard")
	})
}
