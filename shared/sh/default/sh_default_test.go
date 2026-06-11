package sh_test

import (
	"fmt"
	"os"
	"testing"

	sh "github.com/james-orcales/james-orcales/shared/sh/default"
	snap "github.com/james-orcales/james-orcales/shared/snap/default"
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

// Test_Default_Pipe captures stdout from a real command through the ambient
// Default shell.
func Test_Default_Pipe(t *testing.T) {
	if got := sh.Pipe("echo", "scripted"); got != "scripted" {
		t.Errorf("Pipe = %q, want scripted", got)
	}
}

// Test_Default_Which finds an executable on PATH and misses an absent one.
func Test_Default_Which(t *testing.T) {
	if sh.Which("go") == "" {
		t.Error("Which should find go on PATH")
	}
	if sh.Which("nonexistent-command-xyz-123") != "" {
		t.Error("Which should return empty for an absent command")
	}
}

// Test_Default_File_Exists distinguishes an existing path from an absent one.
func Test_Default_File_Exists(t *testing.T) {
	directory := t.TempDir()
	if !sh.File_Exists(directory) {
		t.Error("File_Exists should report the temp dir exists")
	}
	if sh.File_Exists(directory + "/absent") {
		t.Error("File_Exists should report an absent path missing")
	}
}

// Test_Default_Make_Directory_All creates a nested directory.
func Test_Default_Make_Directory_All(t *testing.T) {
	nested := t.TempDir() + "/a/b/c"
	if err := sh.Make_Directory_All(nested); err != nil {
		t.Fatalf("Make_Directory_All: %v", err)
	}
	if !sh.File_Exists(nested) {
		t.Error("Make_Directory_All did not create the directory")
	}
}

// Test_Default_Push_Directory changes into a directory and restores it on pop.
func Test_Default_Push_Directory(t *testing.T) {
	before, _ := os.Getwd()
	pop := sh.Push_Directory(t.TempDir())
	if moved, _ := os.Getwd(); moved == before {
		t.Error("Push_Directory did not change the working directory")
	}
	pop()
	if after, _ := os.Getwd(); after != before {
		t.Error("pop did not restore the working directory")
	}
}

// Test_Default_Lines splits output with positive, negative, and zero counts.
func Test_Default_Lines(t *testing.T) {
	if got := fmt.Sprintf("%v", sh.Lines("a\nb\nc\nd", -2)); got != "[c d]" {
		t.Errorf("Lines tail = %s, want [c d]", got)
	}
	if got := sh.Lines("", 0); got != nil {
		t.Errorf("Lines zero = %v, want nil", got)
	}
}
