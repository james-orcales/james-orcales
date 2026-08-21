//go:build linux && amd64

package nbio_test

import (
	"testing"

	"local/james-orcales/shared/sim/nbio"
	"local/james-orcales/shared/testify"
)

// Test_Simulated_Statx_Heap_Allocation keeps Linux-only surface inside allocation proof.
func Test_Simulated_Statx_Heap_Allocation(t *testing.T) {
	harness := sim_allocation_harness{}
	result := nbio.Statx{}
	testify.Zero_Allocation(t, func() {
		sim_allocation_reset(&harness)
		result = nbio.Statx{}
		nbio.Platform_Statx(
			harness.Loop,
			&harness.Completion,
			nbio.DIRECTORY_CURRENT,
			"/",
			0,
			nbio.STATX_BASIC_STATS,
			&result,
			sim_allocation_callback,
		)
		sim_allocation_drive(&harness)
	})
	testify.No_Error(t, harness.Error)
	testify.No_Error(t, harness.Completion.Error)
	testify.Not_Nil(t, harness.Completion.Self)
	testify.False(t, harness.Completion.Armed)
	testify.Equal(t, nbio.STATX_BASIC_STATS, result.Mask)
}
