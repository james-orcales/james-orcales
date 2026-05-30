package sh_test

import (
	"testing"

	sh "github.com/james-orcales/james-orcales/golang_snacks/sh/default"
	snap "github.com/james-orcales/james-orcales/golang_snacks/snap/default"
)

// Test_Default_Shell_Pipe_Echo runs a real process end to end through the
// OS-wired Shell, confirming the composition tier spawns and captures stdout.
func Test_Default_Shell_Pipe_Echo(t *testing.T) {
	shell := sh.Init_Default_Shell()
	got := sh.Shell_Pipe(shell, "echo", "hello world")
	if !snap.Snapshot_Is_Equal(snap.Init(`hello world`), got) {
		t.Errorf("Shell_Pipe = %q", got)
	}
}

// Test_Default_Shell_Spawn_Exit_Status confirms the exit-code -> ok mapping
// against real processes.
func Test_Default_Shell_Spawn_Exit_Status(t *testing.T) {
	shell := sh.Init_Default_Shell()
	if !sh.Shell_Spawn(shell, "true") {
		t.Error("`true` should report success")
	}
	if sh.Shell_Spawn(shell, "false") {
		t.Error("`false` should report failure")
	}
}
