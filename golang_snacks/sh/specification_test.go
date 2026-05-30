package sh_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/james-orcales/james-orcales/golang_snacks/sh"
	snap "github.com/james-orcales/james-orcales/golang_snacks/snap/default"
)

// Test_Spawn_Raw_Plan_Valid pins the env/executable/argument partition over the
// cases that produce a valid plan.
func Test_Spawn_Raw_Plan_Valid(t *testing.T) {
	snap.Batch_Expect(t, render_plan, []snap.Entry[[]string]{
		{
			Name:     "env then executable and arguments",
			Input:    []string{"A=1", "B=2", "cmd", "arg"},
			Snapshot: snap.Init(`env=[A=1 B=2] path="cmd" args=[arg]`),
		},
		{
			Name:     "env value may contain equals",
			Input:    []string{"VAR=v=w=x", "sh", "-c", "echo $VAR"},
			Snapshot: snap.Init(`env=[VAR=v=w=x] path="sh" args=[-c echo $VAR]`),
		},
		{
			Name:     "env value may be empty",
			Input:    []string{"EMPTY=", "sh"},
			Snapshot: snap.Init(`env=[EMPTY=] path="sh" args=[]`),
		},
		{
			Name:     "equals after the executable is an ordinary argument",
			Input:    []string{"echo", "key=value"},
			Snapshot: snap.Init(`env=[] path="echo" args=[key=value]`),
		},
		{
			Name:     "executable only",
			Input:    []string{"echo"},
			Snapshot: snap.Init(`env=[] path="echo" args=[]`),
		},
	})
}

// Test_Spawn_Raw_Plan_Invalid verifies a plan with no executable is rejected.
func Test_Spawn_Raw_Plan_Invalid(t *testing.T) {
	for _, arguments := range [][]string{{}, {"A=1", "B=2"}} {
		if _, ok := sh.Spawn_Raw_Plan(arguments); ok {
			t.Errorf("Spawn_Raw_Plan(%v) ok = true, want false", arguments)
		}
	}
}

// Test_Shell_Run_Command_Environment verifies the base environment comes first
// and per-command entries last, so a duplicate key resolves to the override.
func Test_Shell_Run_Command_Environment(t *testing.T) {
	shell, recorded := test_shell(nil)
	sh.Shell_Spawn(shell, "FOO=bar", "cmd", "arg")
	got := fmt.Sprintf("%v", (*recorded)[0].Environment)
	if !snap.Snapshot_Is_Equal(snap.Init(`[BASE=1 FOO=bar]`), got) {
		t.Errorf("environment = %s", got)
	}
}

// Test_Shell_Run_Command_Streams verifies a Command with nil streams inherits
// the Shell's defaults.
func Test_Shell_Run_Command_Streams(t *testing.T) {
	shell, recorded := test_shell(nil)
	sh.Shell_Spawn(shell, "cmd")
	if (*recorded)[0].Stdout != shell.Stdout {
		t.Error("Stdout was not defaulted to the Shell's Stdout")
	}
	if (*recorded)[0].Stderr != shell.Stderr {
		t.Error("Stderr was not defaulted to the Shell's Stderr")
	}
}

// Test_Shell_Spawn_Exit verifies a zero exit reports success and any other exit
// reports failure.
func Test_Shell_Spawn_Exit(t *testing.T) {
	for _, c := range []struct {
		Exit int
		Want bool
	}{{0, true}, {1, false}, {-1, false}} {
		fake := func(sh.Command) (outcome sh.Outcome) { return sh.Outcome{Exit: c.Exit} }
		shell, _ := test_shell(fake)
		if got := sh.Shell_Spawn(shell, "cmd"); got != c.Want {
			t.Fatalf("exit %d: ok = %v, want %v", c.Exit, got, c.Want)
		}
	}
}

// Test_Shell_Pipe_Capture verifies captured stdout is returned trimmed.
func Test_Shell_Pipe_Capture(t *testing.T) {
	fake := func(command sh.Command) (outcome sh.Outcome) {
		fmt.Fprint(command.Stdout, "  hello\n")
		return sh.Outcome{}
	}
	shell, _ := test_shell(fake)
	if got := sh.Shell_Pipe(shell, "echo", "hello"); got != "hello" {
		t.Errorf("Shell_Pipe = %q, want hello", got)
	}
}

// Test_Shell_Working_Directory_Derivation verifies deriving a sub-shell sets the
// new directory on the copy and leaves the original untouched.
func Test_Shell_Working_Directory_Derivation(t *testing.T) {
	shell, _ := test_shell(nil)
	shell.Working_Directory = "/start"
	derived := sh.Shell_With_Working_Directory(shell, "/elsewhere")
	if sh.Shell_Working_Directory(derived) != "/elsewhere" {
		t.Errorf("derived = %q, want /elsewhere", sh.Shell_Working_Directory(derived))
	}
	if sh.Shell_Working_Directory(shell) != "/start" {
		t.Error("deriving a sub-shell mutated the original")
	}
}

// Renders the three fields Spawn_Raw_Plan sets, so a valid parse can be
// snapshotted as a single line.
func render_plan(arguments []string) (rendered any) {
	command, ok := sh.Spawn_Raw_Plan(arguments)
	if !ok {
		return "INVALID"
	}
	return fmt.Sprintf("env=%v path=%q args=%v",
		command.Environment, command.Path, command.Arguments)
}

// Builds a Shell whose Run records the resolved Command and returns a canned
// Outcome, with a fixed base environment. The recorded slice lets tests assert
// what the ops resolved before handing it to Run.
func test_shell(
	run func(sh.Command) (outcome sh.Outcome),
) (shell *sh.Shell, recorded *[]sh.Command) {
	commands := []sh.Command{}
	shell = &sh.Shell{
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
		Environ: func() (environment []string) { return []string{"BASE=1"} },
		Run: func(command sh.Command) (outcome sh.Outcome) {
			commands = append(commands, command)
			if run != nil {
				return run(command)
			}
			return sh.Outcome{}
		},
	}
	return shell, &commands
}
