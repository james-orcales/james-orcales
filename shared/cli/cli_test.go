package cli_test

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"testing"

	"local/james-orcales/shared/cli"
	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/snap/default"
)

func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}

// Per-test program plus its captured output buffers.
type cli_fixture struct {
	Program  cli.Program
	Stdout   *bytes.Buffer
	Stderr   *bytes.Buffer
	Database *bytes.Buffer
}

// Builds the todoctl program with fresh buffers. cli.New runs here, inside each
// test, so its assertions are registered before use.
func new_cli_fixture() (fixture cli_fixture) {
	fixture.Stdout = &bytes.Buffer{}
	fixture.Stderr = &bytes.Buffer{}
	fixture.Database = &bytes.Buffer{}
	fixture.Program = cli.New(cli.New_Input{
		Label:       "todoctl",
		Description: "is a todo list manager",
		Commands: []cli.Command{
			{Label: "help", Description: "print help message"},
			todoctl_add_command(),
			todoctl_list_command(),
			todoctl_delete_command(),
		},
	})
	return fixture
}

// Builds the todoctl "add" command.
func todoctl_add_command() (command cli.Command) {
	return cli.Command{
		Label: "add",
		Arguments: []cli.Option{
			{Label: "task", Value: "", Description: "describes what you need to do"},
		},
		Flags: []cli.Option{
			{Label: "deadline", Value: "Jan 02 Mon", Description: "deadline date"},
			{Label: "priority", Value: "low", Description: "set priority"},
			{Label: "noop", Value: false, Description: "random flag that does nothing"},
		},
	}
}

// Builds the todoctl "list" command.
func todoctl_list_command() (command cli.Command) {
	return cli.Command{
		Label: "list",
		Flags: []cli.Option{
			{
				Label: "columns",
				Value: "all",
				Description: "comma-separated (all, deadline, " +
					"priority, description)",
			},
			{Label: "count", Value: 0, Description: "how many tasks to display"},
		},
	}
}

// Builds the todoctl "delete" command.
func todoctl_delete_command() (command cli.Command) {
	return cli.Command{
		Label:       "delete",
		Description: "remove a task by ID",
		Arguments: []cli.Option{
			// Setting Value to 0 makes the parser treat this as an int.
			{
				Label:       "id",
				Value:       0,
				Description: "the integer index of the task to remove",
			},
		},
	}
}

// Dispatches a parsed command against the fixture's state, the way func main
// would; extracted so each test stays its own entry point.
func run_command(command cli.Command, fixture *cli_fixture) {
	switch command.Label {
	case "help":
		cli.Print_Help(fixture.Stdout, fixture.Program)
	case "add":
		run_add(command, fixture)
	case "delete":
		run_delete(command, fixture)
	case "list":
		run_list(command, fixture)
	}
}

// Appends a formatted task row to the fixture's database.
func run_add(command cli.Command, fixture *cli_fixture) {
	fixture.Database.WriteString(fmt.Sprintf(
		"%s | %s | %s\n",
		cli.Get_Option(command.Flags, "deadline").Value,
		cli.Get_Option(command.Flags, "priority").Value,
		cli.Get_Option(command.Arguments, "task").Value,
	))
}

// Removes the task at the given index from the fixture's database.
func run_delete(command cli.Command, fixture *cli_fixture) {
	identifier := cli.Get_Option(command.Arguments, "id").Value.(int)
	content := strings.TrimSpace(fixture.Database.String())
	if content == "" {
		fmt.Fprintln(fixture.Stdout, "List is already empty")
		return
	}
	lines := strings.Split(content, "\n")
	if identifier < 0 {
		fmt.Fprintf(fixture.Stdout, "Error: ID %d is out of range (0 to %d)\n",
			identifier, len(lines)-1)
		return
	}
	if identifier >= len(lines) {
		fmt.Fprintf(fixture.Stdout, "Error: ID %d is out of range (0 to %d)\n",
			identifier, len(lines)-1)
		return
	}
	fixture.Database.Reset()
	for index, line := range lines {
		if index == identifier {
			continue
		}
		fixture.Database.WriteString(line + "\n")
	}
	fmt.Fprintf(fixture.Stdout, "Deleted task %d\n", identifier)
}

// Prints the selected columns of up to count tasks from the database.
func run_list(command cli.Command, fixture *cli_fixture) {
	columns := cli.Get_Option(command.Flags, "columns").Value.(string)
	count := cli.Get_Option(command.Flags, "count").Value.(int)
	all, deadline, priority, description := false, false, false, false
	for column := range strings.SplitSeq(columns, ",") {
		if column == "all" {
			all = true
			break
		}
		if column == "deadline" {
			deadline = true
		}
		if column == "priority" {
			priority = true
		}
		if column == "description" {
			description = true
		}
	}
	if all {
		fmt.Fprintln(fixture.Stdout, fixture.Database.String())
		return
	}
	iteration := 0
	for line := range strings.Lines(fixture.Database.String()) {
		if iteration >= count {
			break
		}
		parts := strings.Split(line, " | ")
		output := []string{}
		if deadline {
			output = append(output, parts[0])
		}
		if priority {
			output = append(output, parts[1])
		}
		if description {
			output = append(output, string(parts[2][:len(parts[2])-1]))
		}
		fmt.Fprintln(fixture.Stdout, strings.Join(output, "::"))
		iteration++
	}
}

// Input for check.
type check_input struct {
	T        *testing.T
	Fixture  *cli_fixture
	Snapshot snap.Snapshot
}

// Asserts the fixture's combined buffers against the snapshot.
func check(input check_input) {
	input.T.Helper()
	expect := fmt.Sprintf("Stdout:\n%s\nStderr:\n%s\nDatabase:\n%s\n",
		input.Fixture.Stdout.String(), input.Fixture.Stderr.String(),
		input.Fixture.Database.String())
	if !snap.Snapshot_Is_Equal(input.Snapshot, expect) {
		input.T.Fatal("Snapshot mismatch")
	}
}

// Test_Help_Message verifies Print_Help renders the program, commands, and flags.
func Test_Help_Message(t *testing.T) {
	fixture := new_cli_fixture()
	cli.Print_Help(fixture.Stdout, fixture.Program)
	check(check_input{T: t, Fixture: &fixture, Snapshot: snap.Init(`Stdout:
todoctl is a todo list manager

Usage:
    todoctl <command> <arguments> [-flags[=value]]
    Positional arguments may also be supplied via -key=val syntax.

Global Flags:
    -[34mhelp[0m  show this help


Available Commands:
    [34mhelp[0m print help message

    [34madd[0m <task: string> 

        -deadline=string  (default: Jan 02 Mon)  deadline date
        -priority=string  (default: low)         set priority
        -noop             random flag that does nothing

    [34mlist[0m 

        -columns=string  (default: all)  comma-separated (all, deadline, priority, description)
        -count=int       (default: 0)    how many tasks to display

    [34mdelete[0m <id: int> remove a task by ID


Stderr:

Database:

`)})
}

// Test_Demo verifies a sequence of add, list, and delete commands end to end.
func Test_Demo(t *testing.T) {
	fixture := new_cli_fixture()
	commands := [][]string{
		{"todoctl", "add", "commit to github", "-deadline=Nov 21 Fri"},
		{"todoctl", "add", "something important", "-noop"},
		{"todoctl", "add", "foo bar baz"},
		{"todoctl", "list", "-count=2"},
		{"todoctl", "delete", "1"},
		{"todoctl", "list", "-columns=priority,description"},
	}
	for _, command := range commands {
		command, err := cli.Program_Parse(&fixture.Program, command)
		if err != nil {
			panic(err)
		}
		run_command(command, &fixture)
	}
	check(check_input{T: t, Fixture: &fixture, Snapshot: snap.Init(`Stdout:
Nov 21 Fri | low | commit to github
Jan 02 Mon | low | something important
Jan 02 Mon | low | foo bar baz

Deleted task 1

Stderr:

Database:
Nov 21 Fri | low | commit to github
Jan 02 Mon | low | foo bar baz

`)})
}

// Test_User_Error verifies parse errors are reported for malformed input.
func Test_User_Error(t *testing.T) {
	fixture := new_cli_fixture()
	commands := [][]string{
		{"different_label"},
		{"todoctl", "arsotitnaroisen"},
		{"todoctl", "add"},
		{"todoctl", "add", "without flags"},
		{"todoctl", "add", "missing flag value", "-deadline="},
		{"todoctl", "add", "another missing flag value", "-deadline"},
		{"todoctl", "add", "unknown flag", "-unknown"},
		{"todoctl", "add", "-out_of_place", "commit to github"},
		{"todoctl", "list", "-columns=priority,description", "-count=-1"},
		{"todoctl", "list", "-count=0", "-count=-1", "-count=-2"},
	}
	for _, command := range commands {
		command, err := cli.Program_Parse(&fixture.Program, command)
		if err != nil {
			fmt.Fprintln(fixture.Stderr, err.Error())
			continue
		}
		run_command(command, &fixture)
	}
	check(check_input{T: t, Fixture: &fixture, Snapshot: snap.Init(`Stdout:
todoctl is a todo list manager

Usage:
    todoctl <command> <arguments> [-flags[=value]]
    Positional arguments may also be supplied via -key=val syntax.

Global Flags:
    -[34mhelp[0m  show this help


Available Commands:
    [34mhelp[0m print help message

    [34madd[0m <task: string> 

        -deadline=string  (default: Jan 02 Mon)  deadline date
        -priority=string  (default: low)         set priority
        -noop             random flag that does nothing

    [34mlist[0m 

        -columns=string  (default: all)  comma-separated (all, deadline, priority, description)
        -count=int       (default: 0)    how many tasks to display

    [34mdelete[0m <id: int> remove a task by ID


Stderr:
unknown command "arsotitnaroisen"
missing required argument "task"; pass it by position or as -task=value
-deadline needs a value, e.g. -deadline=value
-deadline needs a value, e.g. -deadline=value
unknown option -unknown
unknown option -out_of_place
-count may only be given once

Database:
Jan 02 Mon | low | without flags

`)})
}

// Test_Quotes verifies quoted flag values are unquoted during parsing.
func Test_Quotes(t *testing.T) {
	fixture := new_cli_fixture()
	commands := [][]string{
		{"todoctl", "add", "task with double quotes", `-deadline="Nov 25 Tue"`},
		{"todoctl", "add", "task with single quotes", `-priority='high'`},
		{"todoctl", "add", "task with mixed text", `-deadline="Dec 01 Mon"`, `-priority='medium'`},
		{"todoctl", "list"},
	}
	for _, command := range commands {
		command, err := cli.Program_Parse(&fixture.Program, command)
		if err != nil {
			panic(err)
		}
		run_command(command, &fixture)
	}
	check(check_input{T: t, Fixture: &fixture, Snapshot: snap.Init(`Stdout:
Nov 25 Tue | low | task with double quotes
Jan 02 Mon | high | task with single quotes
Dec 01 Mon | medium | task with mixed text


Stderr:

Database:
Nov 25 Tue | low | task with double quotes
Jan 02 Mon | high | task with single quotes
Dec 01 Mon | medium | task with mixed text

`)})
}

// Test_Single_Help verifies single-command help drops the <command> selector and
// renders the program's own positionals in the usage line.
func Test_Single_Help(t *testing.T) {
	program := new_single_fixture()
	output := bytes.Buffer{}
	cli.Print_Help(&output, program)
	help := output.String()
	if strings.Contains(help, "<command>") {
		t.Errorf("single-command help must not mention <command>:\n%s", help)
	}
	if !strings.Contains(help, "sloc <path: string>") {
		t.Errorf("expected usage with the positional, got:\n%s", help)
	}
}

// Test_Variadic_Help verifies a variadic renders with an ellipsis and its element
// type, not the raw slice type.
func Test_Variadic_Help(t *testing.T) {
	program := cli.New_Single(cli.New_Single_Input{
		Label: "sloc", Description: "count lines of code",
		Arguments: []cli.Option{
			cli.New_Variadic[string](cli.New_Variadic_Input{Label: "path"}),
		},
	})
	output := bytes.Buffer{}
	cli.Print_Help(&output, program)
	help := output.String()
	if !strings.Contains(help, "<path: string...>") {
		t.Errorf("expected <path: string...>, got:\n%s", help)
	}
	if strings.Contains(help, "[]string") {
		t.Errorf("help must not show the raw slice type:\n%s", help)
	}
}

// Test_Enum_Flag_Help verifies an enum flag renders its permitted set in place of the
// bare type annotation, keeping its default.
func Test_Enum_Flag_Help(t *testing.T) {
	program := cli.New_Single(cli.New_Single_Input{
		Label: "prog", Description: "enum flag help",
		Flags: []cli.Option{
			cli.New_Enum_Flag(cli.New_Enum_Flag_Input[string]{
				Label: "color", Enum: []string{"auto", "never", "always"},
				Value: "auto", Description: "when to colorize",
			}),
		},
	})
	output := bytes.Buffer{}
	cli.Print_Help(&output, program)
	help := output.String()
	// The label is ANSI-colored in flag rows, so assert on the uncolored value part —
	// the enum set — rather than the "-color" prefix.
	if !strings.Contains(help, "=(auto|never|always)") {
		t.Errorf("expected the enum set in the flag row, got:\n%s", help)
	}
	if !strings.Contains(help, "default: auto") {
		t.Errorf("expected the default, got:\n%s", help)
	}
}

// Test_Enum_Argument_Help verifies an enum positional shows its permitted set in the
// usage signature instead of the bare type.
func Test_Enum_Argument_Help(t *testing.T) {
	program := cli.New_Single(cli.New_Single_Input{
		Label: "prog", Description: "enum argument help",
		Arguments: []cli.Option{
			cli.New_Enum_Argument(cli.New_Enum_Argument_Input[int]{
				Label: "level", Enum: []int{1, 2, 4, 8},
				Description: "compression level",
			}),
		},
	})
	output := bytes.Buffer{}
	cli.Print_Help(&output, program)
	help := output.String()
	if !strings.Contains(help, "<level: (1|2|4|8)>") {
		t.Errorf("expected the enum set in the signature, got:\n%s", help)
	}
}

// Test_Print_Requested_Help verifies the render helper picks catalog help for the root
// context (empty-label command) and per-command help for a selected command.
func Test_Print_Requested_Help(t *testing.T) {
	fixture := new_cli_fixture()
	root := bytes.Buffer{}
	cli.Print_Requested_Help(&root, fixture.Program, cli.Command{})
	if !strings.Contains(root.String(), "Available Commands") {
		t.Errorf("root help should list commands, got:\n%s", root.String())
	}

	command, _ := cli.Program_Parse(&fixture.Program, []string{"todoctl", "list", "-help"})
	one := bytes.Buffer{}
	cli.Print_Requested_Help(&one, fixture.Program, command)
	if !strings.Contains(one.String(), "list") {
		t.Errorf("command help should name the command, got:\n%s", one.String())
	}
	if strings.Contains(one.String(), "Available Commands") {
		t.Errorf("command help should not list every command, got:\n%s", one.String())
	}
}

// Builds a single-command program: one positional path and one flag, no selector.
func new_single_fixture() (program cli.Program) {
	return cli.New_Single(cli.New_Single_Input{
		Label:       "sloc",
		Description: "count lines of code",
		Arguments: []cli.Option{
			{Label: "path", Value: "", Description: "directory to scan"},
		},
		Flags: []cli.Option{
			{Label: "hidden", Value: false, Description: "include hidden dot-files"},
		},
	})
}

// Test_Command_Help verifies Print_Command renders one command's own usage line and
// flags, the per-verb help a multicall binary shows for the invoked name.
func Test_Command_Help(t *testing.T) {
	program := new_multicall_fixture()
	command, err := cli.Program_Parse(&program, []string{"add", "milk"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := bytes.Buffer{}
	cli.Print_Command(&output, program, command)
	help := output.String()
	if !strings.Contains(help, "add <task: string>") {
		t.Errorf("expected the command signature, got:\n%s", help)
	}
	if !strings.Contains(help, "priority") {
		t.Errorf("expected the command's flags, got:\n%s", help)
	}
}

// Builds a multicall program: each command is selected by the binary name in argv[0],
// as a busybox-style symlinked binary is.
func new_multicall_fixture() (program cli.Program) {
	return cli.New_Multicall(cli.New_Multicall_Input{
		Label:       "toolbox",
		Description: "a multicall binary",
		Commands: []cli.Command{
			todoctl_add_command(),
			todoctl_list_command(),
			todoctl_delete_command(),
		},
	})
}

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
				Label: "color", Enum: []string{"auto", "never", "always"},
				Value: "auto",
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

	output := bytes.Buffer{}
	args := []string{"todoctl", "__complete", "todoctl", "l"}
	if !cli.Handle_Completion(program, args, &output) {
		t.Fatal("expected __complete to be handled")
	}
	if !strings.Contains(output.String(), "list") {
		t.Errorf("expected list candidate, got %q", output.String())
	}

	output.Reset()
	if !cli.Handle_Completion(program, []string{"todoctl", "completion", "bash"}, &output) {
		t.Fatal("expected completion to be handled")
	}
	if !strings.Contains(output.String(), "__complete") {
		t.Errorf("expected a script, got %q", output.String())
	}

	output.Reset()
	if cli.Handle_Completion(program, []string{"todoctl", "list"}, &output) {
		t.Error("a normal invocation must not be handled as completion")
	}
}
