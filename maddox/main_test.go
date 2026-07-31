package main

import (
	"os"
	"testing"

	maddox "local/james-orcales/maddox/internal"
)

// Test_Is_Terminal_Regular_File verifies that the root reports redirected output as
// nonterminal before it injects that state into Main.
func Test_Is_Terminal_Regular_File(t *testing.T) {
	file, create_err := os.CreateTemp(t.TempDir(), "maddox-output-*")
	if create_err != nil {
		t.Fatal(create_err)
	}
	defer file.Close()
	if uint8(is_terminal(file)) != maddox.TERMINAL_STATUS_NOT_TERMINAL {
		t.Fatal("a regular file must not be a terminal")
	}
}

// Test_Terminal_Status_Constants verifies that each stream-specific status uses the
// shared terminal domain.
func Test_Terminal_Status_Constants(t *testing.T) {
	output := maddox.Output_Terminal_Status(maddox.TERMINAL_STATUS_NOT_TERMINAL)
	if uint8(output) != maddox.TERMINAL_STATUS_NOT_TERMINAL {
		t.Fatal("the output status must use the shared nonterminal member")
	}
	error_output := maddox.Error_Output_Terminal_Status(maddox.TERMINAL_STATUS_TERMINAL)
	if uint8(error_output) != maddox.TERMINAL_STATUS_TERMINAL {
		t.Fatal("the error-output status must use the shared terminal member")
	}
}
