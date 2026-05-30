package cli_test

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/james-orcales/james-orcales/shared/cli"
	"github.com/james-orcales/james-orcales/shared/invariant"
	"github.com/james-orcales/james-orcales/shared/snap/default"
)

func TestMain(m *testing.M) {
	invariant.RegisterPackagesForAnalysis()
	code := m.Run()
	invariant.AnalyzeAssertionFrequency()
	os.Exit(code)
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

Available Commands:
    [34mhelp[0m print help message

    [34madd[0m <task:string> 

        -deadline=string  (default: Jan 02 Mon)  deadline date
        -priority=string  (default: low)         set priority
        -noop             random flag that does nothing

    [34mlist[0m 

        -columns=string  (default: all)  comma-separated (all, deadline, priority, description)
        -count=int       (default: 0)    how many tasks to display

    [34mdelete[0m <id:int> remove a task by ID


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

Available Commands:
    [34mhelp[0m print help message

    [34madd[0m <task:string> 

        -deadline=string  (default: Jan 02 Mon)  deadline date
        -priority=string  (default: low)         set priority
        -noop             random flag that does nothing

    [34mlist[0m 

        -columns=string  (default: all)  comma-separated (all, deadline, priority, description)
        -count=int       (default: 0)    how many tasks to display

    [34mdelete[0m <id:int> remove a task by ID


Stderr:
"arsotitnaroisen" is an unknown command
"add" expects 1 arguments. Got 0
"deadline" expects a value. You must set flag values with this syntax: -foo-bar=baz.
"deadline" expects a value. You must set flag values with this syntax: -foo-bar=baz.
"unknown" is an unknown flag
Positional arguments cannot appear after flags. Got "commit to github"
"list" supports 2 command flags and 0 global flags. Got 3

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
