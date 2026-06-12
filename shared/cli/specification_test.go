package cli_test

import (
	"slices"
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

// Test_Parse_Single_Command verifies a single-command program reads its positionals
// and flags directly after the program name, with no command selector in slot 1.
func Test_Parse_Single_Command(t *testing.T) {
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

	// A token that would select a sibling command in a multi-command program is just
	// a positional here: a single-command program has no selector namespace.
	command, err = cli.Program_Parse(&program, []string{"sloc", "help"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Get_Option(command.Arguments, "path").Value.(string) != "help" {
		t.Errorf("expected path help, got %q",
			cli.Get_Option(command.Arguments, "path").Value)
	}

	// The exact argument-count rule still applies; single-command mode only changes
	// where positionals start, not their arity.
	_, err = cli.Program_Parse(&program, []string{"sloc"})
	if err == nil {
		t.Error("expected error for a missing positional argument")
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

// Test_Parse_Variadic verifies a variadic last argument collects zero or more
// trailing positionals into a slice, after any fixed positionals.
func Test_Parse_Variadic(t *testing.T) {
	single := cli.New_Single(cli.New_Single_Input{
		Label: "sloc", Description: "count lines of code",
		Arguments: []cli.Option{
			cli.New_Variadic[string](cli.New_Variadic_Input{Label: "path"}),
		},
		Flags: []cli.Option{{Label: "hidden", Value: false}},
	})
	assert_variadic(t, single, []string{"sloc", "a", "b", "c"}, "path", "a", "b", "c")
	// Zero trailing positionals yields an empty slice, not an error.
	assert_variadic(t, single, []string{"sloc"}, "path")
	// The variadic stops at the first flag.
	assert_variadic(t, single, []string{"sloc", "a", "b", "-hidden"}, "path", "a", "b")

	// Fixed positionals may precede the variadic: cp <dest> <source...>.
	multi := cli.New(cli.New_Input{
		Label: "fileutil", Description: "file utilities",
		Commands: []cli.Command{{
			Label: "cp", Description: "copy files",
			Arguments: []cli.Option{
				cli.New_Argument[string](cli.New_Argument_Input{Label: "dest"}),
				cli.New_Variadic[string](cli.New_Variadic_Input{Label: "source"}),
			},
		}},
	})
	assert_variadic(t, multi,
		[]string{"fileutil", "cp", "d", "s1", "s2"}, "source", "s1", "s2")

	// The fixed argument before the variadic is still required.
	_, err := cli.Program_Parse(&multi, []string{"fileutil", "cp"})
	if err == nil {
		t.Error("expected error for the missing fixed argument")
	}

	// A variadic int converts each element and reports a bad one.
	numbers := cli.New_Single(cli.New_Single_Input{
		Label: "sum", Description: "add numbers",
		Arguments: []cli.Option{cli.New_Variadic[int](cli.New_Variadic_Input{Label: "n"})},
	})
	command, err := cli.Program_Parse(&numbers, []string{"sum", "1", "2", "3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(cli.Get_Option(command.Arguments, "n").Value.([]int), []int{1, 2, 3}) {
		t.Errorf("expected [1 2 3], got %v", cli.Get_Option(command.Arguments, "n").Value)
	}
	_, err = cli.Program_Parse(&numbers, []string{"sum", "1", "x"})
	if err == nil {
		t.Error("expected error for a non-numeric variadic element")
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

// Test_New_Validation verifies New and New_Single panic on a malformed program.
func Test_New_Validation(t *testing.T) {
	// New rejects a command without a label.
	assert_panics(t, "command without a label", func() {
		cli.New(cli.New_Input{
			Label: "prog", Description: "test",
			Commands: []cli.Command{{Label: ""}},
		})
	})
	// New_Single rejects a malformed flag the same way New does.
	assert_panics(t, "flag with an underscore", func() {
		cli.New_Single(cli.New_Single_Input{
			Label: "sloc",
			Flags: []cli.Option{{Label: "no_ignore", Value: false}},
		})
	})
	// A variadic must be the last argument, which also forbids a second variadic.
	assert_panics(t, "non-terminal variadic", func() {
		cli.New_Single(cli.New_Single_Input{
			Label: "x",
			Arguments: []cli.Option{
				cli.New_Variadic[string](cli.New_Variadic_Input{Label: "a"}),
				cli.New_Argument[string](cli.New_Argument_Input{Label: "b"}),
			},
		})
	})
}

// Parses arguments against a copy of the program and asserts the named variadic
// argument equals want. The program is taken by value so callers can reuse it.
func assert_variadic(
	t *testing.T, program cli.Program, arguments []string, label string, want ...string,
) {
	t.Helper()
	command, err := cli.Program_Parse(&program, arguments)
	if err != nil {
		t.Fatalf("%v: unexpected error: %v", arguments, err)
	}
	got := cli.Get_Option(command.Arguments, label).Value.([]string)
	if !slices.Equal(got, want) {
		t.Errorf("%v: expected %v, got %v", arguments, want, got)
	}
}

// Asserts that calling action panics; reason labels the case in the failure.
func assert_panics(t *testing.T, reason string, action func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("expected panic: %s", reason)
		}
	}()
	action()
}
