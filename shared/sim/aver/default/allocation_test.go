package aver_test

import (
	"testing"

	core "local/james-orcales/shared/sim/aver"
	aver "local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/testify"
)

// Default_Allocation_Subject keeps Tree's type lookup real in each enforcing build.
type Default_Allocation_Subject int

// Global swap stays outside measurement because dependency injection is setup, not assertion work.
func Test_Runtime_Entry_Points_Have_Zero_Allocations(t *testing.T) {
	previous := aver.Default
	aver.Default = &core.Recorder{}
	defer func() { aver.Default = previous }()
	testify.Zero_Allocation(t, func() {
		aver.Always(true, "always")
		aver.Sometimes(true, "sometimes")
		aver.Range(5, 0, 10, "range")
		aver.Range_Holed(5, 0, 10, 1, 2, 3, 4, "holed range")
		aver.Enum(1, 1, 2, "enum")
		aver.Tree(Default_Allocation_Subject(0), "builder").
			Sometimes(true, "sometimes").
			Range_Int(5, 0, 10).
			Range_Holed_Int(5, 0, 10, 1, 2, 3, 4).
			Enum_4_Int(3, 1, 2, 3, 4).
			Ensure()
	})
}
