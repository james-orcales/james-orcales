package cli_test

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"local/james-orcales/shared/cli"
)

// Builds a single-command program with an enum flag and an enum positional, for the
// value-completion cases. path is a plain string arg (file completion, no candidates).
func new_completion_fixture() (program cli.Program) {
	return cli.New_Single(cli.New_Single_Input{
		Label: "tool", Description: "a tool",
		Arguments: []cli.Option{
			cli.New_Enum_Argument(cli.New_Enum_Argument_Input[string]{
				Label: "format", Enum: []string{"json", "csv", "toml"},
			}),
		},
		Flags: []cli.Option{
			cli.New_Enum_Flag(cli.New_Enum_Flag_Input[string]{
				Label: "color", Enum: []string{"auto", "never", "always"}, Value: "auto",
			}),
			cli.New_Flag(cli.New_Flag_Input[bool]{Label: "verbose"}),
		},
	})
}

// Test_Complete_Commands verifies command-name completion in a multi-command program,
// filtered by the partial word.
func Test_Complete_Commands(t *testing.T) {
	fixture := new_cli_fixture()
	got := cli.Complete(fixture.Program, []string{"todoctl", "l"})
	if !slices.Contains(got, "list") {
		t.Errorf("expected list among %v", got)
	}
	if slices.Contains(got, "add") {
		t.Errorf("prefix l should exclude add, got %v", got)
	}
}

// Test_Complete_Multicall_Self verifies a multicall binary run by its own name completes
// verb names from the first token, matching busybox self-invocation.
func Test_Complete_Multicall_Self(t *testing.T) {
	program := new_multicall_fixture()
	got := cli.Complete(program, []string{"toolbox", "ad"})
	if !slices.Contains(got, "add") {
		t.Errorf("expected add among %v", got)
	}
}

// Test_Complete_Flags verifies flag-name completion within a command, including the
// auto-injected -help and the command's arguments (settable by name).
func Test_Complete_Flags(t *testing.T) {
	fixture := new_cli_fixture()
	got := cli.Complete(fixture.Program, []string{"todoctl", "add", "-"})
	for _, want := range []string{"-deadline", "-priority", "-help", "-task"} {
		if !slices.Contains(got, want) {
			t.Errorf("expected %q among %v", want, got)
		}
	}
}

// Test_Complete_Enum_Flag_Value verifies an enum flag's value completes to its members.
func Test_Complete_Enum_Flag_Value(t *testing.T) {
	program := new_completion_fixture()
	got := cli.Complete(program, []string{"tool", "-color="})
	for _, want := range []string{"-color=auto", "-color=never", "-color=always"} {
		if !slices.Contains(got, want) {
			t.Errorf("expected %q among %v", want, got)
		}
	}
	got = cli.Complete(program, []string{"tool", "-color=n"})
	if !slices.Equal(got, []string{"-color=never"}) {
		t.Errorf("expected only -color=never, got %v", got)
	}
}

// Test_Complete_Positional_Enum verifies a positional whose option is an enum completes
// to its members.
func Test_Complete_Positional_Enum(t *testing.T) {
	program := new_completion_fixture()
	got := cli.Complete(program, []string{"tool", ""})
	for _, want := range []string{"json", "csv", "toml"} {
		if !slices.Contains(got, want) {
			t.Errorf("expected %q among %v", want, got)
		}
	}
}

// Test_Complete_File_Position verifies a plain (non-enum) positional yields no
// candidates, leaving file completion to the shell.
func Test_Complete_File_Position(t *testing.T) {
	program := new_single_fixture() // sloc: variadic string path, no enum
	got := cli.Complete(program, []string{"sloc", ""})
	if len(got) != 0 {
		t.Errorf("expected no candidates for a path position, got %v", got)
	}
}

// Test_Completion_Script verifies each shell's script names the binary and calls back
// into __complete, and an unknown shell errors.
func Test_Completion_Script(t *testing.T) {
	program := new_cli_fixture().Program
	for _, shell := range []string{"bash", "zsh", "fish"} {
		script, err := cli.Completion_Script(program, shell)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", shell, err)
		}
		if !strings.Contains(script, "todoctl") {
			t.Errorf("%s script should name the binary, got:\n%s", shell, script)
		}
		if !strings.Contains(script, "__complete") {
			t.Errorf("%s script should call __complete, got:\n%s", shell, script)
		}
	}
	if _, err := cli.Completion_Script(program, "tcsh"); err == nil {
		t.Error("expected an error for an unknown shell")
	}
}

// Test_Handle_Completion verifies the pre-parse gate intercepts __complete and
// completion and passes everything else through.
func Test_Handle_Completion(t *testing.T) {
	program := new_cli_fixture().Program

	out := bytes.Buffer{}
	if !cli.Handle_Completion(program, []string{"todoctl", "__complete", "todoctl", "l"}, &out) {
		t.Fatal("expected __complete to be handled")
	}
	if !strings.Contains(out.String(), "list") {
		t.Errorf("expected list candidate, got %q", out.String())
	}

	out.Reset()
	if !cli.Handle_Completion(program, []string{"todoctl", "completion", "bash"}, &out) {
		t.Fatal("expected completion to be handled")
	}
	if !strings.Contains(out.String(), "__complete") {
		t.Errorf("expected a script, got %q", out.String())
	}

	out.Reset()
	if cli.Handle_Completion(program, []string{"todoctl", "list"}, &out) {
		t.Error("a normal invocation must not be handled as completion")
	}
}
