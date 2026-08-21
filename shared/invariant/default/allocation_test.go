package invariant_test

import (
	"testing"

	core "local/james-orcales/shared/invariant"
	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/testify"
)

// Default_Allocation_Subject keeps Tree's type lookup real in each enforcing build.
type Default_Allocation_Subject int

// Global swap stays outside measurement because dependency injection is setup, not assertion work.
func Test_Runtime_Entry_Points_Have_Zero_Allocations(t *testing.T) {
	previous := invariant.Default
	invariant.Default = &core.Recorder{}
	defer func() { invariant.Default = previous }()
	testify.Zero_Allocation(t, func() {
		invariant.Always(true, "always")
		invariant.Sometimes(true, "sometimes")
		invariant.Range(5, 0, 10, "range")
		invariant.Range_Holed(5, 0, 10, 1, 2, 3, 4, "holed range")
		invariant.Enum(1, 1, 2, "enum")
		invariant.Tree(Default_Allocation_Subject(0), "builder").
			Sometimes(true, "sometimes").
			Range_Int(5, 0, 10).
			Range_Holed_Int(5, 0, 10, 1, 2, 3, 4).
			Enum_4_Int(3, 1, 2, 3, 4).
			Ensure()
	})
}
