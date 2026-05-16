package cli_test

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/james-orcales/james-orcales/golang_snacks/cli"
	"github.com/james-orcales/james-orcales/golang_snacks/invariant"
	"github.com/james-orcales/james-orcales/golang_snacks/snap"
)

func TestMain(m *testing.M) {
	database.Grow(1024 * 256)
	defer stdout.Reset()
	defer stderr.Reset()

	invariant.RegisterPackagesForAnalysis()
	initGlobal()
	code := m.Run()
	invariant.AnalyzeAssertionFrequency()
	os.Exit(code)
}

// initializing program AFTER registering packages so that cli.New assertions are registered
func initGlobal() {
	program = cli.New(cli.NewInput{
		Label:       "todoctl",
		Description: "is a todo list manager",
		Commands: []cli.Command{
			{Label: "help", Description: "print help message"},
			{
				Label: "add",
				Arguments: []cli.Option{
					{Label: "task", Value: "", Description: "describes what you need to do"},
				},
				Flags: []cli.Option{
					{Label: "deadline", Value: "Jan 02 Mon", Description: "deadline date"},
					{Label: "priority", Value: "low", Description: "set priority"},
					{Label: "noop", Value: false, Description: "random flag that does nothing"},
				},
			},
			{
				Label: "list",
				Flags: []cli.Option{
					{
						Label:       "columns",
						Value:       "all",
						Description: "comma-separated (all, deadline, priority, description)",
					},
					{Label: "count", Value: 0, Description: "how many tasks to display"},
				},
			},
			{
				Label:       "delete",
				Description: "remove a task by ID",
				Arguments: []cli.Option{
					// Setting Value to 0 ensures the parser treats this as an int
					{Label: "id", Value: 0, Description: "the integer index of the task to remove"},
				},
			},
		},
	})
}

var (
	database = &bytes.Buffer{}
	stdout   = &bytes.Buffer{}
	stderr   = &bytes.Buffer{}
	program  cli.Program
	// Usually, you'd inline this in func main(). Here, we extracted it out into separate
	// function since each test is a separate entrypoint.
	command_handler = func(command cli.Command) {
		switch command.Label {
		case "help":
			cli.PrintHelp(stdout, program)
		case "add":
			database.WriteString(fmt.Sprintf(
				"%s | %s | %s\n",
				cli.GetOption(command.Flags, "deadline").Value,
				cli.GetOption(command.Flags, "priority").Value,
				cli.GetOption(command.Arguments, "task").Value,
			))
		case "delete":
			id := cli.GetOption(command.Arguments, "id").Value.(int)
			content := strings.TrimSpace(database.String())
			if content == "" {
				fmt.Fprintln(stdout, "List is already empty")
				return
			}

			lines := strings.Split(content, "\n")
			if id < 0 || id >= len(lines) {
				fmt.Fprintf(stdout, "Error: ID %d is out of range (0 to %d)\n", id, len(lines)-1)
				return
			}

			database.Reset()
			for i, line := range lines {
				if i == id {
					continue
				}
				database.WriteString(line + "\n")
			}

			fmt.Fprintf(stdout, "Deleted task %d\n", id)
		case "list":
			columns := cli.GetOption(command.Flags, "columns").Value.(string)
			count := cli.GetOption(command.Flags, "count").Value.(int)
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
				fmt.Fprintln(stdout, database.String())
			} else {
				iteration := 0
				for line := range strings.Lines(database.String()) {
					if iteration >= count {
						break
					}
					cols := strings.Split(line, " | ")
					out := []string{}
					if deadline {
						out = append(out, cols[0])
					}
					if priority {
						out = append(out, cols[1])
					}
					if description {
						out = append(out, string(cols[2][:len(cols[2])-1]))
					}

					fmt.Fprintln(stdout, strings.Join(out, "::"))
					iteration++
				}

			}
		}
	}
)

type checkInput struct {
	T        *testing.T
	Out      *bytes.Buffer
	Err      *bytes.Buffer
	Snapshot snap.Snapshot
}

func check(in checkInput) {
	in.T.Helper()
	defer in.Out.Reset()
	defer in.Err.Reset()
	defer database.Reset()
	expect := fmt.Sprintf("Stdout:\n%s\nStderr:\n%s\nDatabase:\n%s\n", in.Out.String(), in.Err.String(), database.String())
	if !in.Snapshot.IsEqual(expect) {
		in.T.Fatal("Snapshot mismatch")
	}
}

func TestHelpMessage(t *testing.T) {
	cli.PrintHelp(stdout, program)
	check(checkInput{T: t, Out: stdout, Err: stderr, Snapshot: snap.Init(`Stdout:
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

func TestDemo(t *testing.T) {
	commands := [][]string{
		{"todoctl", "add", "commit to github", "-deadline=Nov 21 Fri"},
		{"todoctl", "add", "something important", "-noop"},
		{"todoctl", "add", "foo bar baz"},
		{"todoctl", "list", "-count=2"},
		{"todoctl", "delete", "1"},
		{"todoctl", "list", "-columns=priority,description"},
	}
	for _, command := range commands {
		command, err := cli.Parse(&program, command)
		if err != nil {
			panic(err)
		}
		command_handler(command)
	}
	check(checkInput{T: t, Out: stdout, Err: stderr, Snapshot: snap.Init(`Stdout:
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

func TestUserError(t *testing.T) {
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
		command, err := cli.Parse(&program, command)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			continue
		}
		command_handler(command)
	}
	check(checkInput{T: t, Out: stdout, Err: stderr, Snapshot: snap.Init(`Stdout:
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

func TestQuoteTrimming(t *testing.T) {
	commands := [][]string{
		{"todoctl", "add", "task with double quotes", `-deadline="Nov 25 Tue"`},
		{"todoctl", "add", "task with single quotes", `-priority='high'`},
		{"todoctl", "add", "task with mixed text", `-deadline="Dec 01 Mon"`, `-priority='medium'`},
		{"todoctl", "list"},
	}
	for _, command := range commands {
		command, err := cli.Parse(&program, command)
		if err != nil {
			panic(err)
		}
		command_handler(command)
	}
	check(checkInput{T: t, Out: stdout, Err: stderr, Snapshot: snap.Init(`Stdout:
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

func TestTrimQuotesUnit(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "double quotes",
			args:     []string{"prog", "add", "task", `-flag="value"`},
			expected: "value",
		},
		{
			name:     "single quotes",
			args:     []string{"prog", "add", "task", `-flag='value'`},
			expected: "value",
		},
		{
			name:     "no quotes",
			args:     []string{"prog", "add", "task", `-flag=value`},
			expected: "value",
		},
		{
			name:     "empty double quotes",
			args:     []string{"prog", "add", "task", `-flag=""`},
			expected: "",
		},
		{
			name:     "empty single quotes",
			args:     []string{"prog", "add", "task", `-flag=''`},
			expected: "",
		},
		{
			name:     "mismatched quotes",
			args:     []string{"prog", "add", "task", `-flag="value'`},
			expected: `"value'`,
		},
		{
			name:     "value with spaces in double quotes",
			args:     []string{"prog", "add", "task", `-flag="hello world"`},
			expected: "hello world",
		},
		{
			name:     "value with spaces in single quotes",
			args:     []string{"prog", "add", "task", `-flag='hello world'`},
			expected: "hello world",
		},
	}

	program := cli.New(cli.NewInput{
		Label:       "prog",
		Description: "test program",
		Commands: []cli.Command{
			{
				Label: "add",
				Arguments: []cli.Option{
					{Label: "task", Value: "", Description: "task name"},
				},
				Flags: []cli.Option{
					{Label: "flag", Value: "", Description: "test flag"},
				},
			},
		},
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := cli.Parse(&program, tt.args)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			actual := cli.GetOption(cmd.Flags, "flag").Value.(string)
			if actual != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, actual)
			}
		})
	}
}
