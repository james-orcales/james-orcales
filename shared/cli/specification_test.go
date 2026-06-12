package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/james-orcales/james-orcales/shared/cli"
)

// Test_Parse_Commands verifies named, default, and unknown command resolution.
func Test_Parse_Commands(t *testing.T) {
	fixture := new_cli_fixture()
	command, err := cli.Program_Parse(&fixture.Program, []string{"todoctl", "list"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if command.Label != "list" {
		t.Errorf("expected list, got %q", command.Label)
	}

	command, err = cli.Program_Parse(&fixture.Program, []string{"todoctl"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if command.Label != "help" {
		t.Errorf("expected help, got %q", command.Label)
	}

	_, err = cli.Program_Parse(&fixture.Program, []string{"todoctl", "bogus"})
	if err == nil {
		t.Error("expected error for unknown command")
	}
}

// Test_Parse_Arguments verifies the argument count and integer conversion.
func Test_Parse_Arguments(t *testing.T) {
	fixture := new_cli_fixture()
	_, err := cli.Program_Parse(&fixture.Program, []string{"todoctl", "add"})
	if err == nil {
		t.Error("expected error for missing argument")
	}

	command, err := cli.Program_Parse(&fixture.Program, []string{"todoctl", "delete", "3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Get_Option(command.Arguments, "id").Value.(int) != 3 {
		t.Error("expected id 3")
	}

	_, err = cli.Program_Parse(&fixture.Program, []string{"todoctl", "delete", "abc"})
	if err == nil {
		t.Error("expected error for non-numeric int argument")
	}
}

// Test_Parse_Flags verifies flag assignment and the flag error cases.
func Test_Parse_Flags(t *testing.T) {
	fixture := new_cli_fixture()
	command, err := cli.Program_Parse(&fixture.Program,
		[]string{"todoctl", "add", "task", "-priority=high"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Get_Option(command.Flags, "priority").Value.(string) != "high" {
		t.Error("expected priority high")
	}

	_, err = cli.Program_Parse(&fixture.Program,
		[]string{"todoctl", "add", "task", "-bogus=1"})
	if err == nil {
		t.Error("expected error for unknown flag")
	}

	_, err = cli.Program_Parse(&fixture.Program,
		[]string{"todoctl", "add", "task", "--priority=high"})
	if err == nil {
		t.Error("expected error for double-dash flag")
	}

	_, err = cli.Program_Parse(&fixture.Program,
		[]string{"todoctl", "add", "task", "-priority"})
	if err == nil {
		t.Error("expected error for non-boolean flag without value")
	}
}

// Test_Trim_Quotes_Cases verifies quoted flag values are unquoted by quote style.
func Test_Trim_Quotes_Cases(t *testing.T) {
	program := cli.New(cli.New_Input{
		Label:       "prog",
		Description: "test program",
		Commands: []cli.Command{{
			Label:     "add",
			Arguments: []cli.Option{{Label: "task", Value: ""}},
			Flags:     []cli.Option{{Label: "flag", Value: ""}},
		}},
	})
	check := func(name, raw, want string) {
		t.Helper()
		command, err := cli.Program_Parse(&program, []string{"prog", "add", "task", raw})
		if err != nil {
			t.Fatalf("%s: parse failed: %v", name, err)
		}
		got := cli.Get_Option(command.Flags, "flag").Value.(string)
		if got != want {
			t.Errorf("%s: expected %q, got %q", name, want, got)
		}
	}
	check("double", `-flag="value"`, "value")
	check("single", `-flag='value'`, "value")
	check("none", `-flag=value`, "value")
	check("empty double", `-flag=""`, "")
	check("mismatched", `-flag="value'`, `"value'`)
	check("spaces", `-flag="hello world"`, "hello world")
}

// Test_Get_Option_Lookup verifies a present label returns its option and an
// absent label panics.
func Test_Get_Option_Lookup(t *testing.T) {
	options := []cli.Option{
		{Label: "a", Value: "x"},
		{Label: "b", Value: "y"},
	}
	if cli.Get_Option(options, "b").Value.(string) != "y" {
		t.Error("expected y")
	}

	defer func() {
		if recover() == nil {
			t.Error("expected panic for an unknown option")
		}
	}()
	cli.Get_Option(options, "absent")
}

// Test_Parse_Single verifies a single-command program reads its positionals and
// flags directly after the program name, with no command selector in slot 1.
func Test_Parse_Single(t *testing.T) {
	program := new_single_fixture()

	command, err := cli.Program_Parse(&program, []string{"sloc", "./src"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Get_Option(command.Arguments, "path").Value.(string) != "./src" {
		t.Errorf("expected path ./src, got %q",
			cli.Get_Option(command.Arguments, "path").Value)
	}

	command, err = cli.Program_Parse(&program, []string{"sloc", "./src", "-hidden"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cli.Get_Option(command.Flags, "hidden").Value.(bool) {
		t.Error("expected hidden true")
	}

	// A token that would name a sibling command in a multi-command program is just
	// a positional here: a single-command program has no selector namespace for the
	// first positional to collide with.
	command, err = cli.Program_Parse(&program, []string{"sloc", "help"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Get_Option(command.Arguments, "path").Value.(string) != "help" {
		t.Errorf("expected path help, got %q",
			cli.Get_Option(command.Arguments, "path").Value)
	}

	// The existing exact argument-count rule still applies; single-command mode only
	// changes where positionals start, not their arity.
	_, err = cli.Program_Parse(&program, []string{"sloc"})
	if err == nil {
		t.Error("expected error for a missing positional argument")
	}
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
	if !strings.Contains(help, "sloc <path:string>") {
		t.Errorf("expected usage with the positional, got:\n%s", help)
	}
}

// Test_New_Single_Validation verifies New_Single rejects a malformed program the
// same way New does.
func Test_New_Single_Validation(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic for a flag with an underscore")
		}
	}()
	cli.New_Single(cli.New_Single_Input{
		Label: "sloc",
		Flags: []cli.Option{{Label: "no_ignore", Value: false}},
	})
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

// Test_New_Validation verifies New panics on a malformed program.
func Test_New_Validation(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic for a command without a label")
		}
	}()
	cli.New(cli.New_Input{
		Label:       "prog",
		Description: "test",
		Commands:    []cli.Command{{Label: ""}},
	})
}
