package main

import (
	"testing"

	"local/james-orcales/shared/cli"
)

// Test_Color_Progress_Enums verifies the -color and -progress flags accept only their
// permitted values: a member parses and reads back, while an out-of-set value errors
// instead of silently falling through to the auto branch.
func Test_Color_Progress_Enums(t *testing.T) {
	program := main_program()
	command, err := cli.Program_Parse(&program,
		[]string{"maddox", "echo hi", "-color=never", "-progress=always"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := cli.Get_Option(command.Flags, "color").Value.(string); got != "never" {
		t.Errorf("expected color never, got %q", got)
	}
	if got := cli.Get_Option(command.Flags, "progress").Value.(string); got != "always" {
		t.Errorf("expected progress always, got %q", got)
	}

	program = main_program()
	_, err = cli.Program_Parse(&program, []string{"maddox", "echo hi", "-color=bogus"})
	if err == nil {
		t.Error("expected an error for an out-of-set -color value")
	}

	program = main_program()
	_, err = cli.Program_Parse(&program, []string{"maddox", "echo hi", "-progress=bogus"})
	if err == nil {
		t.Error("expected an error for an out-of-set -progress value")
	}
}
