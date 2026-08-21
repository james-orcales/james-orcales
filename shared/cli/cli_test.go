package cli_test

import (
	"fmt"
	"testing"

	"local/james-orcales/shared/cli"
	"local/james-orcales/shared/simulation/aver/default"
	"local/james-orcales/shared/snap/default"
	"local/james-orcales/shared/strings"
)

func TestMain(m *testing.M) {
	aver.Run_Test_Main(m)
}

type output_buffer struct {
	cli.Output
}

func output_cli(output *output_buffer) (destination *cli.Output) {
	return &output.Output
}

func (output *output_buffer) String() (text string) {
	return string(cli.Output_Bytes(&output.Output))
}

func output_write_string(output *output_buffer, source string) {
	cli.Output_Write_Text(&output.Output, cli.Value_Text(source))
}

func output_reset(output *output_buffer) {
	cli.Output_Reset(&output.Output)
}

func cli_complete(program cli.Program, words []string) (candidates cli.Candidates) {
	var storage [cli.CANDIDATE_COUNT_MAXIMUM]cli.Candidate
	return cli.Complete(program, words, storage[:])
}

func candidates_contain(candidates cli.Candidates, expected string) (contained bool) {
	for _, candidate := range candidates {
		if cli.Candidate_Equal(candidate, cli.Value_Text(expected)) {
			return true
		}
	}
	return false
}

func candidates_equal(candidates cli.Candidates, expected []string) (equal bool) {
	if len(candidates) != len(expected) {
		return false
	}
	for index := range candidates {
		if !cli.Candidate_Equal(candidates[index], cli.Value_Text(expected[index])) {
			return false
		}
	}
	return true
}

func handle_completion(
	program cli.Program, args []string, output *cli.Output,
) (handled bool) {
	var storage [cli.CANDIDATE_COUNT_MAXIMUM]cli.Candidate
	return bool(cli.Handle_Completion(program, args, output, storage[:]))
}

func completion_script(
	program cli.Program, shell cli.Shell,
) (script string, err error) {
	var output cli.Output
	err = cli.Completion_Script(program, shell, &output)
	return string(cli.Output_Bytes(&output)), err
}

func text_contains(source string, fragment string) (contained bool) {
	return bool(strings.Contains(
		strings.Text(source), strings.Text(fragment),
	))
}

func text_index(source string, fragment string) (index int) {
	return int(strings.Index(
		strings.Text(source), strings.Text(fragment),
	))
}

func text_has_prefix(source string, prefix string) (present bool) {
	return bool(strings.Has_Prefix(
		strings.Text(source), strings.Text(prefix),
	))
}

func text_trim_space(source string) (trimmed string) {
	return string(strings.Trim_Space(strings.Text(source)))
}

func text_split(source string, separator string) (parts []string) {
	var storage [strings.TEXT_COUNT_MAXIMUM]strings.Text
	count := strings.Split_Into(
		storage[:], strings.Text(source), strings.Text(separator),
	)
	output := make([]string, int(count))
	for index := range output {
		output[index] = string(storage[index])
	}
	return output
}

func text_lines(source string) (lines []string) {
	return text_split(source, "\n")
}

func text_join(parts []string, separator string) (joined string) {
	var output strings.Builder
	for index := range parts {
		if index > 0 {
			strings.Builder_Write_Text(&output, strings.Text(separator))
		}
		strings.Builder_Write_Text(&output, strings.Text(parts[index]))
	}
	return string(strings.Builder_Bytes(&output))
}

// Per-test program plus its captured output buffers.
type cli_fixture struct {
	Program  cli.Program
	Stdout   *output_buffer
	Stderr   *output_buffer
	Database *output_buffer
}

// Builds the todoctl program with fresh buffers. cli.New runs here, inside each
// test, so its assertions are registered before use.
func new_cli_fixture() (fixture cli_fixture) {
	fixture.Stdout = &output_buffer{}
	fixture.Stderr = &output_buffer{}
	fixture.Database = &output_buffer{}
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
		cli.Print_Help(output_cli(fixture.Stdout), fixture.Program)
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
	output_write_string(fixture.Database, fmt.Sprintf(
		"%s | %s | %s\n",
		cli.Option_String(cli.Get_Option(command.Flags, "deadline")),
		cli.Option_String(cli.Get_Option(command.Flags, "priority")),
		cli.Option_String(cli.Get_Option(command.Arguments, "task")),
	))
}

// Removes the task at the given index from the fixture's database.
func run_delete(command cli.Command, fixture *cli_fixture) {
	identifier := int(cli.Option_Integer(cli.Get_Option(command.Arguments, "id")))
	content := text_trim_space(fixture.Database.String())
	if content == "" {
		fmt.Fprintln(fixture.Stdout, "List is already empty")
		return
	}
	lines := text_split(content, "\n")
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
	output_reset(fixture.Database)
	for index, line := range lines {
		if index == identifier {
			continue
		}
		output_write_string(fixture.Database, line+"\n")
	}
	fmt.Fprintf(fixture.Stdout, "Deleted task %d\n", identifier)
}

// Prints the selected columns of up to count tasks from the database.
func run_list(command cli.Command, fixture *cli_fixture) {
	columns := string(cli.Option_String(cli.Get_Option(command.Flags, "columns")))
	count := int(cli.Option_Integer(cli.Get_Option(command.Flags, "count")))
	all, deadline, priority, description := false, false, false, false
	for _, column := range text_split(columns, ",") {
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
	for _, line := range text_lines(fixture.Database.String()) {
		if iteration >= count {
			break
		}
		parts := text_split(line, " | ")
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
		fmt.Fprintln(fixture.Stdout, text_join(output, "::"))
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
	cli.Print_Help(output_cli(fixture.Stdout), fixture.Program)
	check(check_input{T: t, Fixture: &fixture, Snapshot: snap.Init(`Stdout:
todoctl is a todo list manager

Usage:
    todoctl <command> <arguments> [-flags[=value]]
    Positional arguments may also be supplied via -key=val syntax.

Global Flags:
    -[34mh[0m     show this help
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
		command, err := parse_program(&fixture.Program, command)
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
		command, err := parse_program(&fixture.Program, command)
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
    -[34mh[0m     show this help
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
		command, err := parse_program(&fixture.Program, command)
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
	output := output_buffer{}
	cli.Print_Help(output_cli(&output), program)
	help := output.String()
	if text_contains(help, "<command>") {
		t.Errorf("single-command help must not mention <command>:\n%s", help)
	}
	if !text_contains(help, "sloc <path: string>") {
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
	output := output_buffer{}
	cli.Print_Help(output_cli(&output), program)
	help := output.String()
	if !text_contains(help, "<path: string...>") {
		t.Errorf("expected <path: string...>, got:\n%s", help)
	}
	if text_contains(help, "[]string") {
		t.Errorf("help must not show the raw slice type:\n%s", help)
	}
}

// Test_Enum_Flag_Help verifies an enum flag renders its permitted set in place of the
// bare type annotation, keeping its default.
func Test_Enum_Flag_Help(t *testing.T) {
	program := cli.New_Single(cli.New_Single_Input{
		Label: "prog", Description: "enum flag help",
		Flags: []cli.Option{
			cli.New_String_Enum_Flag(cli.New_String_Enum_Flag_Input{
				Label: "color", Enum: []string{"auto", "never", "always"},
				Value: "auto", Description: "when to colorize",
			}),
		},
	})
	output := output_buffer{}
	cli.Print_Help(output_cli(&output), program)
	help := output.String()
	// The label is ANSI-colored in flag rows, so assert on the uncolored value part —
	// the enum set — rather than the "-color" prefix.
	if !text_contains(help, "=(auto|never|always)") {
		t.Errorf("expected the enum set in the flag row, got:\n%s", help)
	}
	if !text_contains(help, "default: auto") {
		t.Errorf("expected the default, got:\n%s", help)
	}
}

// Test_Enum_Argument_Help verifies an enum positional shows its permitted set in the
// usage signature instead of the bare type.
func Test_Enum_Argument_Help(t *testing.T) {
	program := cli.New_Single(cli.New_Single_Input{
		Label: "prog", Description: "enum argument help",
		Arguments: []cli.Option{
			cli.New_Integer_Enum_Argument(cli.New_Integer_Enum_Argument_Input{
				Label: "level", Enum: []int{1, 2, 4, 8},
				Description: "compression level",
			}),
		},
	})
	output := output_buffer{}
	cli.Print_Help(output_cli(&output), program)
	help := output.String()
	if !text_contains(help, "<level: (1|2|4|8)>") {
		t.Errorf("expected the enum set in the signature, got:\n%s", help)
	}
}

// Test_Print_Requested_Help verifies the render helper picks catalog help for the root
// context (empty-label command) and per-command help for a selected command.
func Test_Print_Requested_Help(t *testing.T) {
	fixture := new_cli_fixture()
	root := output_buffer{}
	cli.Print_Requested_Help(output_cli(&root), fixture.Program, cli.Command{})
	if !text_contains(root.String(), "Available Commands") {
		t.Errorf("root help should list commands, got:\n%s", root.String())
	}

	command, _ := parse_program(&fixture.Program, []string{"todoctl", "list", "-help"})
	one := output_buffer{}
	cli.Print_Requested_Help(output_cli(&one), fixture.Program, command)
	if !text_contains(one.String(), "list") {
		t.Errorf("command help should name the command, got:\n%s", one.String())
	}
	if text_contains(one.String(), "Available Commands") {
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
	command, err := parse_program(&program, []string{"add", "milk"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := output_buffer{}
	cli.Print_Command(output_cli(&output), program, command)
	help := output.String()
	if !text_contains(help, "add <task: string>") {
		t.Errorf("expected the command signature, got:\n%s", help)
	}
	if !text_contains(help, "priority") {
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
			cli.New_String_Enum_Argument(cli.New_String_Enum_Argument_Input{
				Label: "format", Enum: []string{"json", "csv", "toml"},
			}),
		},
		Flags: []cli.Option{
			cli.New_String_Enum_Flag(cli.New_String_Enum_Flag_Input{
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
	got := cli_complete(fixture.Program, []string{"todoctl", "l"})
	if !candidates_contain(got, "list") {
		t.Errorf("expected list among %v", got)
	}
	if candidates_contain(got, "add") {
		t.Errorf("prefix l should exclude add, got %v", got)
	}
}

// Test_Complete_Multicall_Self verifies a multicall binary run by its own name completes
// verb names from the first token, matching busybox self-invocation.
func Test_Complete_Multicall_Self(t *testing.T) {
	program := new_multicall_fixture()
	got := cli_complete(program, []string{"toolbox", "ad"})
	if !candidates_contain(got, "add") {
		t.Errorf("expected add among %v", got)
	}
}

// Test_Complete_Flags verifies flag-name completion for default and declared options.
func Test_Complete_Flags(t *testing.T) {
	fixture := new_cli_fixture()
	got := cli_complete(fixture.Program, []string{"todoctl", "add", "-"})
	for _, want := range []string{"-deadline", "-priority", "-h", "-help", "-task"} {
		if !candidates_contain(got, want) {
			t.Errorf("expected %q among %v", want, got)
		}
	}
}

// Test_Complete_Enum_Flag_Value verifies an enum flag's value completes to its members.
func Test_Complete_Enum_Flag_Value(t *testing.T) {
	program := new_completion_fixture()
	got := cli_complete(program, []string{"tool", "-color="})
	for _, want := range []string{"-color=auto", "-color=never", "-color=always"} {
		if !candidates_contain(got, want) {
			t.Errorf("expected %q among %v", want, got)
		}
	}
	got = cli_complete(program, []string{"tool", "-color=n"})
	if !candidates_equal(got, []string{"-color=never"}) {
		t.Errorf("expected only -color=never, got %v", got)
	}
}

// Test_Complete_Positional_Enum verifies a positional whose option is an enum completes
// to its members.
func Test_Complete_Positional_Enum(t *testing.T) {
	program := new_completion_fixture()
	got := cli_complete(program, []string{"tool", ""})
	for _, want := range []string{"json", "csv", "toml"} {
		if !candidates_contain(got, want) {
			t.Errorf("expected %q among %v", want, got)
		}
	}
}

// Test_Complete_File_Position verifies a plain (non-enum) positional yields no
// candidates, leaving file completion to the shell.
func Test_Complete_File_Position(t *testing.T) {
	program := new_single_fixture() // sloc: variadic string path, no enum
	got := cli_complete(program, []string{"sloc", ""})
	if len(got) != 0 {
		t.Errorf("expected no candidates for a path position, got %v", got)
	}
}

// Test_Completion_Script verifies each shell's script names the binary and calls back
// into __complete, and an unknown shell errors.
func Test_Completion_Script(t *testing.T) {
	program := new_cli_fixture().Program
	for _, shell := range []string{"bash", "zsh", "fish"} {
		script, err := completion_script(program, cli.Shell(shell))
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", shell, err)
		}
		if !text_contains(string(script), "todoctl") {
			t.Errorf("%s script should name the binary, got:\n%s", shell, script)
		}
		if !text_contains(string(script), "__complete") {
			t.Errorf("%s script should call __complete, got:\n%s", shell, script)
		}
	}
	if _, err := completion_script(program, cli.Shell("tcsh")); err == nil {
		t.Error("expected an error for an unknown shell")
	}
}

// Test_Handle_Completion verifies the pre-parse gate intercepts __complete and
// completion and passes everything else through.
func Test_Handle_Completion(t *testing.T) {
	program := new_cli_fixture().Program

	output := output_buffer{}
	args := []string{"todoctl", "__complete", "todoctl", "l"}
	if !handle_completion(program, args, output_cli(&output)) {
		t.Fatal("expected __complete to be handled")
	}
	if !text_contains(output.String(), "list") {
		t.Errorf("expected list candidate, got %q", output.String())
	}

	output_reset(&output)
	if !handle_completion(
		program, []string{"todoctl", "completion", "bash"}, output_cli(&output),
	) {
		t.Fatal("expected completion to be handled")
	}
	if !text_contains(output.String(), "__complete") {
		t.Errorf("expected a script, got %q", output.String())
	}

	output_reset(&output)
	if handle_completion(program, []string{"todoctl", "list"}, output_cli(&output)) {
		t.Error("a normal invocation must not be handled as completion")
	}
}

// Existing fixtures have no secrets, so their parser must publish before a loop runs.
func parse_program(
	program *cli.Program,
	arguments []string,
) (command cli.Command, err error) {
	var parser cli.Parser
	input := cli_test_parse_input(*program, arguments)
	cli.Program_Parse(program, &parser, input)
	result, done := cli.Parser_Done(&parser)
	if !done {
		panic("a program without secrets did not complete synchronously")
	}
	return result.Command, result.Error
}

func cli_test_parse_input(
	program cli.Program, arguments []string,
) (input cli.Program_Parse_Input) {
	argument_count := 0
	flag_count := 0
	if program.Selection[0].Mode[0] == cli.PROGRAM_MODE_SINGLE {
		command := program.Selection[0].Single_Commands[0]
		argument_count = len(command.Arguments)
		flag_count = len(command.Flags)
	} else {
		for _, command := range program.Selection[0].Commands {
			if len(command.Arguments) > argument_count {
				argument_count = len(command.Arguments)
			}
			if len(command.Flags) > flag_count {
				flag_count = len(command.Flags)
			}
		}
	}
	global_count := len(program.Selection[0].Global_Flags)
	environment_count := len(program.Environment_Variables)
	secret_count := len(program.Secrets)
	environment_warning_count := environment_count
	if environment_warning_count > cli.DEPRECATION_WARNING_COUNT_MAXIMUM {
		environment_warning_count = cli.DEPRECATION_WARNING_COUNT_MAXIMUM
	}
	token_count := len(arguments) - 1
	if token_count < 0 {
		token_count = 0
	}
	return cli.Program_Parse_Input{
		Arguments:            arguments,
		Command_Arguments:    make([]cli.Option, argument_count),
		Command_Flags:        make([]cli.Option, flag_count),
		Global_Flags:         make([]cli.Option, global_count),
		Filled:               make([]bool, argument_count+flag_count+global_count),
		Positionals:          make([]cli.Indexed_Token, token_count),
		Slice_Named:          make([]cli.Indexed_Token, token_count),
		String_Values:        make([]string, token_count),
		Integer_Values:       make([]int, token_count),
		Deprecation_Warnings: make([]cli.Warning, cli.DEPRECATION_WARNING_COUNT_MAXIMUM),
		Environment_Values:   make([]cli.Environment_Variable, environment_count),
		Environment_Errors:   make([]error, environment_count),
		Environment_Warnings: make([]cli.Warning, environment_warning_count),
		Environment_Sources:  make(cli.Environment_Sources, environment_count),
		Secret_Values:        make([]cli.Secret, secret_count),
		Secret_Errors:        make([]cli.Secret_Failure, secret_count),
		Secret_Warnings:      make([]cli.Warning, secret_count),
		Secret_Parsers:       make([]cli.Secret_Parser, secret_count),
		Secret_Buffers:       make([]cli.Secret_Bytes, secret_count),
		Secret_Path_Failures: make([]cli.Path_Failures, secret_count),
		Failures:             make([]error, environment_count+secret_count),
	}
}

func cli_test_program_parse(
	program *cli.Program, parser *cli.Parser, input cli.Program_Parse_Input,
) {
	prepared := cli_test_parse_input(*program, input.Arguments)
	prepared.Environment = input.Environment
	prepared.Loop = input.Loop
	if input.Loop.Storage.Open_At_Procedure != nil {
		for index, secret := range program.Secrets {
			prepared.Secret_Buffers[index] = make([]byte, cli.SECRET_BUFFER_BYTES_MAX)
			prepared.Secret_Path_Failures[index] = make(
				[]cli.Path_Failure, len(secret.Paths),
			)
		}
	}
	cli_test_parse_override(&prepared, input)
	cli.Program_Parse(program, parser, prepared)
}

func cli_test_parse_override(
	prepared *cli.Program_Parse_Input, input cli.Program_Parse_Input,
) {
	if input.Command_Arguments != nil {
		prepared.Command_Arguments = input.Command_Arguments
	}
	if input.Command_Flags != nil {
		prepared.Command_Flags = input.Command_Flags
	}
	if input.Global_Flags != nil {
		prepared.Global_Flags = input.Global_Flags
	}
	if input.Filled != nil {
		prepared.Filled = input.Filled
	}
	if input.Positionals != nil {
		prepared.Positionals = input.Positionals
	}
	if input.Slice_Named != nil {
		prepared.Slice_Named = input.Slice_Named
	}
	if input.String_Values != nil {
		prepared.String_Values = input.String_Values
	}
	if input.Integer_Values != nil {
		prepared.Integer_Values = input.Integer_Values
	}
	if input.Deprecation_Warnings != nil {
		prepared.Deprecation_Warnings = input.Deprecation_Warnings
	}
	if input.Environment_Values != nil {
		prepared.Environment_Values = input.Environment_Values
	}
	if input.Environment_Errors != nil {
		prepared.Environment_Errors = input.Environment_Errors
	}
	if input.Environment_Warnings != nil {
		prepared.Environment_Warnings = input.Environment_Warnings
	}
	if input.Environment_Sources != nil {
		prepared.Environment_Sources = input.Environment_Sources
	}
	if input.Secret_Values != nil {
		prepared.Secret_Values = input.Secret_Values
	}
	if input.Secret_Errors != nil {
		prepared.Secret_Errors = input.Secret_Errors
	}
	if input.Secret_Warnings != nil {
		prepared.Secret_Warnings = input.Secret_Warnings
	}
	if input.Secret_Parsers != nil {
		prepared.Secret_Parsers = input.Secret_Parsers
	}
	if input.Secret_Buffers != nil {
		prepared.Secret_Buffers = input.Secret_Buffers
	}
	if input.Secret_Path_Failures != nil {
		prepared.Secret_Path_Failures = input.Secret_Path_Failures
	}
	if input.Failures != nil {
		prepared.Failures = input.Failures
	}
}
