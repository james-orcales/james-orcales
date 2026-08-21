package tabwriter_test

import (
	"testing"

	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/text/tabwriter"
)

// TestMain keeps allocation probes on production assertion paths.
func TestMain(m *testing.M) {
	aver.Run_Test_Main(m)
}

func configuration_input(
	minimum_width int,
	tab_width int,
	padding int,
	pad_character byte,
	flags tabwriter.Flags,
) (input tabwriter.Configuration_Input) {
	input.Minimum_Width = tabwriter.Minimum_Width(minimum_width)
	input.Tab_Width = tabwriter.Tab_Width(tab_width)
	input.Padding = tabwriter.Padding(padding)
	input.Pad_Character = tabwriter.Pad_Character(pad_character)
	input.Flags = flags
	return input
}
