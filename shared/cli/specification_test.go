package cli_test

import (
	"errors"
	"path"
	"slices"
	"strings"
	"testing"

	"local/james-orcales/shared/cli"
	sharedio "local/james-orcales/shared/io"
)

// Test_Parse_Commands verifies named, default, and unknown command resolution.
func Test_Parse_Commands(t *testing.T) {
	fixture := new_cli_fixture()
	command, err := parse_program(&fixture.Program, []string{"todoctl", "list"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if command.Label != "list" {
		t.Errorf("expected list, got %q", command.Label)
	}

	command, err = parse_program(&fixture.Program, []string{"todoctl"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if command.Label != "help" {
		t.Errorf("expected help, got %q", command.Label)
	}

	_, err = parse_program(&fixture.Program, []string{"todoctl", "bogus"})
	if err == nil {
		t.Error("expected error for unknown command")
	}

	// A near-miss command yields a suggestion; a wild miss does not.
	_, err = parse_program(&fixture.Program, []string{"todoctl", "lst"})
	if err == nil {
		t.Fatal("expected an error for unknown command lst")
	}
	if !strings.Contains(err.Error(), `did you mean "list"`) {
		t.Errorf("expected a suggestion of list, got %v", err)
	}
	_, err = parse_program(&fixture.Program, []string{"todoctl", "zzzzzzzz"})
	if err == nil {
		t.Fatal("expected an error for unknown command zzzzzzzz")
	}
	if strings.Contains(err.Error(), "did you mean") {
		t.Errorf("expected no suggestion for a wild miss, got %v", err)
	}
}

// Test_Parse_Single_Command verifies a single-command program reads its positionals
// and flags directly after the program name, with no command selector in slot 1.
func Test_Parse_Single_Command(t *testing.T) {
	program := new_single_fixture()

	command, err := parse_program(&program, []string{"sloc", "./src"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Get_Option(command.Arguments, "path").Value.(string) != "./src" {
		t.Errorf("expected path ./src, got %q",
			cli.Get_Option(command.Arguments, "path").Value)
	}

	command, err = parse_program(&program, []string{"sloc", "./src", "-hidden"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cli.Get_Option(command.Flags, "hidden").Value.(bool) {
		t.Error("expected hidden true")
	}

	// A token that would select a sibling command in a multi-command program is just
	// a positional here: a single-command program has no selector namespace.
	command, err = parse_program(&program, []string{"sloc", "help"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Get_Option(command.Arguments, "path").Value.(string) != "help" {
		t.Errorf("expected path help, got %q",
			cli.Get_Option(command.Arguments, "path").Value)
	}

	// The exact argument-count rule still applies; single-command mode only changes
	// where positionals start, not their arity.
	_, err = parse_program(&program, []string{"sloc"})
	if err == nil {
		t.Error("expected error for a missing positional argument")
	}
}

// Test_Parse_Multicall verifies a multicall program selects its command from the
// binary name in argv[0], not a token in slot 1, so a symlinked verb dispatches on
// its own name and everything after argv[0] is that command's arguments.
func Test_Parse_Multicall(t *testing.T) {
	program := new_multicall_fixture()

	// The command is the basename of argv[0]; the rest of argv is its arguments.
	command, err := parse_program(&program, []string{"/usr/local/bin/add", "milk"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if command.Label != "add" {
		t.Errorf("expected add, got %q", command.Label)
	}
	if cli.Get_Option(command.Arguments, "task").Value.(string) != "milk" {
		t.Errorf("expected task milk, got %v",
			cli.Get_Option(command.Arguments, "task").Value)
	}

	// An unknown binary name suggests the closest command.
	_, err = parse_program(&program, []string{"ad"})
	if err == nil {
		t.Fatal("expected an error for the unknown multicall name ad")
	}
	if !strings.Contains(err.Error(), `did you mean "add"`) {
		t.Errorf("expected a suggestion of add, got %v", err)
	}
}

// Test_Parse_Arguments verifies the argument count and integer conversion.
func Test_Parse_Arguments(t *testing.T) {
	fixture := new_cli_fixture()
	_, err := parse_program(&fixture.Program, []string{"todoctl", "add"})
	if err == nil {
		t.Error("expected error for missing argument")
	}

	command, err := parse_program(&fixture.Program, []string{"todoctl", "delete", "3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Get_Option(command.Arguments, "id").Value.(int) != 3 {
		t.Error("expected id 3")
	}

	_, err = parse_program(&fixture.Program, []string{"todoctl", "delete", "abc"})
	if err == nil {
		t.Error("expected error for non-numeric int argument")
	}
}

// Test_Parse_Named verifies every option is settable by -label=value, that named and
// positional tokens interleave freely, and that a repeated or unknown option errors.
func Test_Parse_Named(t *testing.T) {
	fixture := new_cli_fixture()

	// An argument can be set by name instead of by position.
	command, err := parse_program(&fixture.Program,
		[]string{"todoctl", "add", "-task=hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Get_Option(command.Arguments, "task").Value.(string) != "hello" {
		t.Errorf("expected task hello, got %v",
			cli.Get_Option(command.Arguments, "task").Value)
	}

	// Named and positional tokens may appear in any order.
	command, err = parse_program(&fixture.Program,
		[]string{"todoctl", "add", "-priority=high", "world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Get_Option(command.Arguments, "task").Value.(string) != "world" {
		t.Error("expected task world from a positional after a flag")
	}
	if cli.Get_Option(command.Flags, "priority").Value.(string) != "high" {
		t.Error("expected priority high")
	}

	// A positional skips an argument already set by name and fills the next free one.
	pair := cli.New_Single(cli.New_Single_Input{
		Label: "pair", Description: "two values",
		Arguments: []cli.Option{
			cli.New_Argument[string](cli.New_Argument_Input{Label: "first"}),
			cli.New_Argument[string](cli.New_Argument_Input{Label: "second"}),
		},
	})
	command, err = parse_program(&pair, []string{"pair", "-first=x", "y"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Get_Option(command.Arguments, "first").Value.(string) != "x" {
		t.Error("expected first x")
	}
	if cli.Get_Option(command.Arguments, "second").Value.(string) != "y" {
		t.Error("expected second y")
	}

	// Setting a scalar option twice, or naming an unknown option, is an error.
	_, err = parse_program(&fixture.Program,
		[]string{"todoctl", "add", "task", "-priority=high", "-priority=low"})
	if err == nil {
		t.Error("expected error for a scalar set more than once")
	}
	_, err = parse_program(&fixture.Program, []string{"todoctl", "add", "task", "-zzz=1"})
	if err == nil {
		t.Error("expected error for an unknown option")
	}

	// A near-miss option name yields a suggestion.
	_, err = parse_program(&fixture.Program,
		[]string{"todoctl", "add", "task", "-priorty=high"})
	if err == nil {
		t.Fatal("expected an error for unknown option -priorty")
	}
	if !strings.Contains(err.Error(), "did you mean -priority") {
		t.Errorf("expected a suggestion of -priority, got %v", err)
	}
}

// Test_Parse_Variadic verifies a slice argument collects trailing positionals, accepts
// repeated -label=value that append, and merges both kinds in token order.
func Test_Parse_Variadic(t *testing.T) {
	single := cli.New_Single(cli.New_Single_Input{
		Label: "sloc", Description: "count lines of code",
		Arguments: []cli.Option{
			cli.New_Variadic[string](cli.New_Variadic_Input{Label: "path"}),
		},
		Flags: []cli.Option{{Label: "hidden", Value: false}},
	})
	assert_variadic(t, single, []string{"sloc", "a", "b", "c"}, "path", "a", "b", "c")
	// Zero positionals yields an empty slice, not an error.
	assert_variadic(t, single, []string{"sloc"}, "path")
	// The slice stops at a flag it does not own.
	assert_variadic(t, single, []string{"sloc", "a", "b", "-hidden"}, "path", "a", "b")
	// Repeated -label=value append to the slice.
	assert_variadic(t, single, []string{"sloc", "-path=a", "-path=b"}, "path", "a", "b")
	// Positional and named contributions merge in token order.
	assert_variadic(t, single, []string{"sloc", "x", "-path=a"}, "path", "x", "a")

	// Scalar positionals may precede the slice: cp <dest> <source...>.
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
	// Naming the scalar frees its slot; the positionals overflow into the slice.
	assert_variadic(t, multi,
		[]string{"fileutil", "cp", "a", "b", "-dest=/tmp"}, "source", "a", "b")
	// The scalar before the slice is still required.
	_, err := parse_program(&multi, []string{"fileutil", "cp"})
	if err == nil {
		t.Error("expected error for the missing scalar argument")
	}

	// A slice int converts each element and reports a bad one.
	numbers := cli.New_Single(cli.New_Single_Input{
		Label: "sum", Description: "add numbers",
		Arguments: []cli.Option{cli.New_Variadic[int](cli.New_Variadic_Input{Label: "n"})},
	})
	command, err := parse_program(&numbers, []string{"sum", "1", "2", "3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(cli.Get_Option(command.Arguments, "n").Value.([]int), []int{1, 2, 3}) {
		t.Errorf("expected [1 2 3], got %v", cli.Get_Option(command.Arguments, "n").Value)
	}
	_, err = parse_program(&numbers, []string{"sum", "1", "x"})
	if err == nil {
		t.Error("expected error for a non-numeric slice element")
	}
}

// Test_Parse_Flags verifies flag assignment and the flag error cases.
func Test_Parse_Flags(t *testing.T) {
	fixture := new_cli_fixture()
	command, err := parse_program(&fixture.Program,
		[]string{"todoctl", "add", "task", "-priority=high"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Get_Option(command.Flags, "priority").Value.(string) != "high" {
		t.Error("expected priority high")
	}

	_, err = parse_program(&fixture.Program,
		[]string{"todoctl", "add", "task", "-bogus=1"})
	if err == nil {
		t.Error("expected error for unknown flag")
	}

	_, err = parse_program(&fixture.Program,
		[]string{"todoctl", "add", "task", "--priority=high"})
	if err == nil {
		t.Error("expected error for double-dash flag")
	}

	_, err = parse_program(&fixture.Program,
		[]string{"todoctl", "add", "task", "-priority"})
	if err == nil {
		t.Error("expected error for non-boolean flag without value")
	}
}

// Test_Parse_Enum verifies a flag-form enum: a permitted value is accepted, an omitted
// enum falls back to its default, and an out-of-set value is rejected — with a
// levenshtein suggestion for a near miss and the full allowed list otherwise. The int
// instantiation rejects a non-member with the same allowed-list message.
func Test_Parse_Enum(t *testing.T) {
	program := cli.New_Single(cli.New_Single_Input{
		Label: "prog", Description: "enum flags",
		Flags: []cli.Option{
			cli.New_Enum_Flag(cli.New_Enum_Flag_Input[string]{
				Label: "color", Enum: []string{"auto", "never", "always"},
				Value: "auto", Description: "when to colorize",
			}),
			cli.New_Enum_Flag(cli.New_Enum_Flag_Input[int]{
				Label: "level", Enum: []int{1, 2, 4, 8}, Value: 1,
				Description: "compression level",
			}),
		},
	})

	// A permitted value is accepted.
	command, err := parse_program(&program, []string{"prog", "-color=never"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Get_Option(command.Flags, "color").Value.(string) != "never" {
		t.Errorf("expected never, got %q", cli.Get_Option(command.Flags, "color").Value)
	}

	// An omitted enum keeps its default.
	command, err = parse_program(&program, []string{"prog"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Get_Option(command.Flags, "color").Value.(string) != "auto" {
		t.Errorf("expected default auto, got %q",
			cli.Get_Option(command.Flags, "color").Value)
	}

	// A near miss suggests the closest member.
	_, err = parse_program(&program, []string{"prog", "-color=nevr"})
	if err == nil {
		t.Fatal("expected an error for an out-of-set value")
	}
	if !strings.Contains(err.Error(), `did you mean "never"`) {
		t.Errorf("expected a suggestion of never, got %v", err)
	}

	// A wild miss lists the whole set instead of guessing.
	_, err = parse_program(&program, []string{"prog", "-color=purple"})
	if err == nil {
		t.Fatal("expected an error for an out-of-set value")
	}
	if !strings.Contains(err.Error(), "allowed: auto, never, always") {
		t.Errorf("expected the allowed list, got %v", err)
	}

	// An int enum accepts a member and rejects a non-member with the allowed list.
	command, err = parse_program(&program, []string{"prog", "-level=4"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Get_Option(command.Flags, "level").Value.(int) != 4 {
		t.Errorf("expected 4, got %v", cli.Get_Option(command.Flags, "level").Value)
	}
	_, err = parse_program(&program, []string{"prog", "-level=3"})
	if err == nil {
		t.Fatal("expected an error for an out-of-set int value")
	}
	if !strings.Contains(err.Error(), "allowed: 1, 2, 4, 8") {
		t.Errorf("expected the allowed int list, got %v", err)
	}
}

// Test_Parse_Help verifies that -h and -help select help before required argument checks.
func Test_Parse_Help(t *testing.T) {
	single := cli.New_Single(cli.New_Single_Input{
		Label: "tool", Description: "does a thing",
		Arguments: []cli.Option{
			cli.New_Argument[string](cli.New_Argument_Input{Label: "target"}),
		},
	})
	_, err := parse_program(&single, []string{"tool", "-help"})
	if !errors.Is(err, cli.Help_Requested) {
		t.Fatalf("expected Help_Requested, got %v", err)
	}
	short_context, err := parse_program(&single, []string{"tool", "-h"})
	if !errors.Is(err, cli.Help_Requested) {
		t.Fatalf("expected Help_Requested for -h, got %v", err)
	}
	short_help := strings.Builder{}
	cli.Print_Requested_Help(&short_help, single, short_context)
	if !strings.Contains(short_help.String(), "does a thing") {
		t.Fatalf("-h did not select the program help:\n%s", short_help.String())
	}

	// Multi-command: a command then -help resolves that command as the context.
	fixture := new_cli_fixture()
	command, err := parse_program(&fixture.Program, []string{"todoctl", "list", "-help"})
	if !errors.Is(err, cli.Help_Requested) {
		t.Fatalf("expected Help_Requested, got %v", err)
	}
	if command.Label != "list" {
		t.Errorf("expected list context, got %q", command.Label)
	}

	// Multi-command with -help but no command selected → root context (empty label).
	fixture = new_cli_fixture()
	command, err = parse_program(&fixture.Program, []string{"todoctl", "-help"})
	if !errors.Is(err, cli.Help_Requested) {
		t.Fatalf("expected Help_Requested, got %v", err)
	}
	if command.Label != "" {
		t.Errorf("expected root context (empty label), got %q", command.Label)
	}
}

// Test_Parse_Environment_Variables verifies injected typed values and environment defaults.
func Test_Parse_Environment_Variables(t *testing.T) {
	assert_parse_environment_variables(t)
}

// Test_Parse_Secrets verifies fallback paths and asynchronous secret conversion.
func Test_Parse_Secrets(t *testing.T) {
	assert_parse_secrets(t)
}

// Test_Parse_External_Errors verifies that external failures keep declaration order.
func Test_Parse_External_Errors(t *testing.T) {
	assert_parse_external_errors(t)
}

// Test_Completion verifies Complete offers a command's option labels for a leading dash
// and that Handle_Completion serves the reserved __complete invocation.
func Test_Completion(t *testing.T) {
	program := new_multicall_fixture()

	// A leading dash supplies each valid named token.
	dashed := cli.Complete(program, []string{"add", "-"})
	for _, expected := range []string{"-h", "-help", "-task"} {
		if !slices.Contains(dashed, expected) {
			t.Errorf("expected %s among %v", expected, dashed)
		}
	}

	// Handle_Completion serves __complete, printing candidates one per line.
	output := strings.Builder{}
	args := []string{"toolbox", "__complete", "add", "-"}
	if !cli.Handle_Completion(program, args, &output) {
		t.Fatal("expected __complete to be handled")
	}
	if !strings.Contains(output.String(), "-task") {
		t.Errorf("expected -task in completion output, got %q", output.String())
	}
}

// Test_Visibility_Hidden verifies a hidden flag and a hidden command still parse and
// resolve, yet appear in neither help output nor completion candidates.
func Test_Visibility_Hidden(t *testing.T) {
	program := cli.New(cli.New_Input{
		Label: "tool", Description: "a tool",
		Commands: []cli.Command{
			{Label: "run", Description: "run it", Flags: []cli.Option{
				cli.New_Flag(cli.New_Flag_Input[bool]{Label: "verbose"}),
				cli.New_Flag(cli.New_Flag_Input[bool]{
					Label: "secret", Hidden: true,
				}),
			}},
			{Label: "ghost", Description: "internal command", Hidden: true},
		},
	})

	// A hidden flag still parses, and a hidden command still resolves.
	command, err := parse_program(&program, []string{"tool", "run", "-secret"})
	if err != nil {
		t.Fatalf("hidden flag should parse: %v", err)
	}
	if !cli.Get_Option(command.Flags, "secret").Value.(bool) {
		t.Error("expected secret true")
	}
	if _, err = parse_program(&program, []string{"tool", "ghost"}); err != nil {
		t.Fatalf("hidden command should resolve: %v", err)
	}

	// Help shows the visible names and omits the hidden ones.
	help := strings.Builder{}
	cli.Print_Help(&help, program)
	if !strings.Contains(help.String(), "verbose") {
		t.Errorf("help should show a visible flag:\n%s", help.String())
	}
	if strings.Contains(help.String(), "secret") {
		t.Errorf("help must not show a hidden flag:\n%s", help.String())
	}
	if strings.Contains(help.String(), "ghost") {
		t.Errorf("help must not show a hidden command:\n%s", help.String())
	}

	// Completion omits both.
	if slices.Contains(cli.Complete(program, []string{"tool", ""}), "ghost") {
		t.Error("completion must not offer a hidden command")
	}
	if slices.Contains(cli.Complete(program, []string{"tool", "run", "-"}), "-secret") {
		t.Error("completion must not offer a hidden flag")
	}
}

// Test_Visibility_Deprecated verifies a deprecated flag still parses, is hidden like a
// hidden flag, and records a warning that Print_Deprecations emits only when it is used.
func Test_Visibility_Deprecated(t *testing.T) {
	program := cli.New_Single(cli.New_Single_Input{
		Label: "tool", Description: "a tool",
		Flags: []cli.Option{
			cli.New_Flag(cli.New_Flag_Input[string]{Label: "name"}),
			cli.New_Flag(cli.New_Flag_Input[string]{
				Label: "old-name", Deprecated: "use -name",
			}),
		},
	})

	// It still parses, and using it records a warning.
	command, err := parse_program(&program, []string{"tool", "-old-name=ada"})
	if err != nil {
		t.Fatalf("deprecated flag should parse: %v", err)
	}
	if cli.Get_Option(command.Flags, "old-name").Value.(string) != "ada" {
		t.Error("expected old-name ada")
	}
	warnings := strings.Builder{}
	cli.Print_Deprecations(&warnings, command)
	if !strings.Contains(warnings.String(), "use -name") {
		t.Errorf("expected a deprecation warning, got %q", warnings.String())
	}

	// Not using it records nothing.
	quiet, _ := parse_program(&program, []string{"tool", "-name=bob"})
	silence := strings.Builder{}
	cli.Print_Deprecations(&silence, quiet)
	if silence.String() != "" {
		t.Errorf("expected no warning when unused, got %q", silence.String())
	}

	// Help omits it.
	help := strings.Builder{}
	cli.Print_Help(&help, program)
	if strings.Contains(help.String(), "old-name") {
		t.Errorf("help must not show a deprecated flag:\n%s", help.String())
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
	check_case := func(name, raw, want string) {
		t.Helper()
		command, err := parse_program(&program, []string{"prog", "add", "task", raw})
		if err != nil {
			t.Fatalf("%s: parse failed: %v", name, err)
		}
		got := cli.Get_Option(command.Flags, "flag").Value.(string)
		if got != want {
			t.Errorf("%s: expected %q, got %q", name, want, got)
		}
	}
	check_case("double", `-flag="value"`, "value")
	check_case("single", `-flag='value'`, "value")
	check_case("none", `-flag=value`, "value")
	check_case("empty double", `-flag=""`, "")
	check_case("mismatched", `-flag="value'`, `"value'`)
	check_case("spaces", `-flag="hello world"`, "hello world")
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

// Test_Get_Environment_Lookup verifies that an undeclared environment key panics.
func Test_Get_Environment_Lookup(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("an undeclared environment key did not panic")
		}
	}()
	cli.Get_Environment(nil, "UNKNOWN")
}

// Test_Get_Secret_Lookup verifies that an undeclared secret key panics.
func Test_Get_Secret_Lookup(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("an undeclared secret key did not panic")
		}
	}()
	cli.Get_Secret(nil, "UNKNOWN")
}

// Test_New_Validation verifies New and New_Single panic on a malformed program: a
// command or option label that is empty, not flag-safe, or colliding, and a slice
// argument that is not last. The enum and reserved-label cases are asserted alongside.
func Test_New_Validation(t *testing.T) {
	// New rejects a command without a label.
	assert_panics(t, "command without a label", func() {
		cli.New(cli.New_Input{
			Label: "prog", Description: "test",
			Commands: []cli.Command{{Label: ""}},
		})
	})
	// New_Single rejects a malformed flag label the same way New does.
	assert_panics(t, "flag with an underscore", func() {
		cli.New_Single(cli.New_Single_Input{
			Label: "sloc",
			Flags: []cli.Option{{Label: "no_ignore", Value: false}},
		})
	})
	// An argument label must be flag-safe now that arguments are settable by name.
	assert_panics(t, "argument with an underscore", func() {
		cli.New_Single(cli.New_Single_Input{
			Label:     "sloc",
			Arguments: []cli.Option{{Label: "bad_label", Value: ""}},
		})
	})
	// An argument label may not collide with a flag in the -key=value namespace.
	assert_panics(t, "argument colliding with a flag", func() {
		cli.New_Single(cli.New_Single_Input{
			Label:     "sloc",
			Arguments: []cli.Option{{Label: "dup", Value: ""}},
			Flags:     []cli.Option{{Label: "dup", Value: false}},
		})
	})
	// A slice argument must be the last argument.
	assert_panics(t, "non-terminal slice argument", func() {
		cli.New_Single(cli.New_Single_Input{
			Label: "x",
			Arguments: []cli.Option{
				cli.New_Variadic[string](cli.New_Variadic_Input{Label: "a"}),
				cli.New_Argument[string](cli.New_Argument_Input{Label: "b"}),
			},
		})
	})
	assert_new_enum_validation(t)
	assert_external_validation(t)
}

// Test_Parse_Multicall_Self_Invocation verifies that a multicall binary run by its own
// name — not a verb link — selects the command from the first token, as `busybox ls`
// does. This is what lets a bootstrap verb run before the links exist.
func Test_Parse_Multicall_Self_Invocation(t *testing.T) {
	program := new_multicall_fixture()
	command, err := parse_program(&program, []string{"toolbox", "add", "milk"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if command.Label != "add" {
		t.Errorf("expected add, got %q", command.Label)
	}
	if cli.Get_Option(command.Arguments, "task").Value.(string) != "milk" {
		t.Errorf("expected task milk, got %v",
			cli.Get_Option(command.Arguments, "task").Value)
	}
}

// Test_Parse_Enum_Argument verifies an argument-form enum: it is settable by position
// and by name, an omitted value yields the existing missing-required error, and an
// out-of-set value is rejected with the allowed list.
func Test_Parse_Enum_Argument(t *testing.T) {
	program := cli.New_Single(cli.New_Single_Input{
		Label: "prog", Description: "enum argument",
		Arguments: []cli.Option{
			cli.New_Enum_Argument(cli.New_Enum_Argument_Input[string]{
				Label: "format", Enum: []string{"json", "yaml", "toml"},
				Description: "output format",
			}),
		},
	})

	// Accepted by position.
	command, err := parse_program(&program, []string{"prog", "yaml"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Get_Option(command.Arguments, "format").Value.(string) != "yaml" {
		t.Errorf("expected yaml, got %q",
			cli.Get_Option(command.Arguments, "format").Value)
	}

	// Accepted by name.
	command, err = parse_program(&program, []string{"prog", "-format=toml"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Get_Option(command.Arguments, "format").Value.(string) != "toml" {
		t.Errorf("expected toml, got %q",
			cli.Get_Option(command.Arguments, "format").Value)
	}

	// Omitted → the existing missing-required-argument error.
	_, err = parse_program(&program, []string{"prog"})
	if err == nil {
		t.Fatal("expected an error for a missing required enum argument")
	}
	if !strings.Contains(err.Error(), "missing required argument") {
		t.Errorf("expected the missing-required message, got %v", err)
	}

	// A non-member is rejected with the allowed list.
	_, err = parse_program(&program, []string{"prog", "xml"})
	if err == nil {
		t.Fatal("expected an error for an out-of-set argument")
	}
	if !strings.Contains(err.Error(), "allowed: json, yaml, toml") {
		t.Errorf("expected the allowed list, got %v", err)
	}
}

// Verifies that one injected snapshot supplies typed declarations and
// leaves optional declarations at their public defaults when the source is absent.
func assert_parse_environment_variables(t *testing.T) {
	t.Helper()
	program := external_program([]cli.Environment_Variable{
		cli.New_Environment_Variable(cli.New_Environment_Variable_Input[string]{
			Key: "NAME", Required: true,
		}),
		cli.New_Environment_Variable(cli.New_Environment_Variable_Input[int]{
			Key: "PORT", Value: 80,
		}),
		cli.New_Environment_Variable(cli.New_Environment_Variable_Input[bool]{
			Key: "ENABLED",
		}),
		cli.New_Enum_Environment_Variable(cli.New_Enum_Environment_Variable_Input[string]{
			Key: "MODE", Value: "safe", Enum: []string{"safe", "fast"},
		}),
		cli.New_Environment_Variable(cli.New_Environment_Variable_Input[string]{
			Key: "EMPTY", Value: "fallback", Allow_Empty: true,
		}),
	}, nil)
	parser := cli.Program_Parse(&program, cli.Program_Parse_Input{
		Arguments: []string{"external"},
		Environment: []string{
			"UNDECLARED", "NAME=service", "ENABLED=TRUE", "MODE=fast", "EMPTY=",
		},
	})
	result := cli.Parser_Done(parser)
	if result == nil {
		t.Fatal("a parser without secrets did not complete immediately")
	}
	if result.Error != nil {
		t.Fatalf("parse environment: %v", result.Error)
	}
	if cli.Get_Environment(result.Command.Environment, "NAME").Value != "service" {
		t.Fatal("NAME did not resolve")
	}
	if cli.Get_Environment(result.Command.Environment, "PORT").Value != 80 {
		t.Fatal("PORT did not keep its default")
	}
	if cli.Get_Environment(result.Command.Environment, "ENABLED").Value != true {
		t.Fatal("ENABLED did not use strconv.ParseBool")
	}
	if cli.Get_Environment(result.Command.Environment, "MODE").Value != "fast" {
		t.Fatal("MODE did not resolve its enum member")
	}
	if cli.Get_Environment(result.Command.Environment, "EMPTY").Value != "" {
		t.Fatal("EMPTY did not preserve its permitted empty value")
	}
}

// Verifies declaration-order joined errors without errors for
// unrelated environment entries.
func assert_parse_external_errors(t *testing.T) {
	t.Helper()
	program := external_program([]cli.Environment_Variable{
		cli.New_Environment_Variable(cli.New_Environment_Variable_Input[string]{
			Key: "FIRST", Required: true,
		}),
		cli.New_Environment_Variable(cli.New_Environment_Variable_Input[int]{
			Key: "SECOND", Required: true,
		}),
		cli.New_Enum_Environment_Variable(cli.New_Enum_Environment_Variable_Input[string]{
			Key: "THIRD", Required: true, Enum: []string{"yes", "no"},
		}),
	}, nil)
	result := cli.Parser_Done(cli.Program_Parse(&program, cli.Program_Parse_Input{
		Arguments: []string{"external"},
		Environment: []string{
			"UNDECLARED", "FIRST", "SECOND=x", "SECOND=2", "THIRD=maybe",
		},
	}))
	if result == nil {
		t.Fatal("the environment parser did not complete")
	}
	if result.Error == nil {
		t.Fatal("malformed, duplicate, and invalid entries did not fail")
	}
	message := result.Error.Error()
	first_offset := strings.Index(message, "FIRST")
	second_offset := strings.Index(message, "SECOND")
	third_offset := strings.Index(message, "THIRD")
	if first_offset < 0 {
		t.Fatalf("FIRST error is absent: %v", result.Error)
	}
	if second_offset < first_offset {
		t.Fatalf("SECOND error is out of order: %v", result.Error)
	}
	if third_offset < second_offset {
		t.Fatalf("environment errors are not in declaration order: %v", result.Error)
	}
}

// Verifies ordered fallback, terminal-newline removal, typed conversion,
// optional absence, parallel parser completion, and descriptor retirement.
func assert_parse_secrets(t *testing.T) {
	t.Helper()
	loop, driver, _ := sharedio.New_Sim(6)
	seed_secret_files(t, loop, driver, []secret_file{
		{Path: "/second/TOKEN", Content: "value\r\n"},
		{Path: "/secrets/COUNT", Content: "42\n"},
		{Path: "/secrets/ENABLED", Content: "1"},
		{Path: "/secrets/MODE", Content: "fast"},
	})
	program := external_program(nil, []cli.Secret{
		cli.New_Secret[string](cli.New_Secret_Input{
			Paths: []string{"/first/TOKEN", "/second/TOKEN"}, Required: true,
		}),
		cli.New_Secret[int](cli.New_Secret_Input{
			Paths: []string{"/secrets/COUNT"}, Required: true,
		}),
		cli.New_Secret[bool](cli.New_Secret_Input{
			Paths: []string{"/secrets/ENABLED"}, Required: true,
		}),
		cli.New_Enum_Secret(cli.New_Enum_Secret_Input[string]{
			Paths: []string{"/secrets/MODE"}, Required: true,
			Enum: []string{"safe", "fast"},
		}),
		cli.New_Secret[string](cli.New_Secret_Input{
			Paths: []string{"/secrets/OPTIONAL"},
		}),
	})
	parser := cli.Program_Parse(&program, cli.Program_Parse_Input{
		Arguments: []string{"external"}, Loop: loop,
	})
	if cli.Parser_Done(parser) != nil {
		t.Fatal("a parser with available secrets completed before the loop ran")
	}
	result := drive_parser(t, driver, parser)
	if result.Error != nil {
		t.Fatalf("parse secrets: %v", result.Error)
	}
	if cli.Get_Secret(result.Command.Secrets, "TOKEN").Value != "value" {
		t.Fatal("TOKEN did not use its fallback path or remove CRLF")
	}
	if cli.Get_Secret(result.Command.Secrets, "COUNT").Value != 42 {
		t.Fatal("COUNT did not convert")
	}
	if cli.Get_Secret(result.Command.Secrets, "ENABLED").Value != true {
		t.Fatal("ENABLED did not convert")
	}
	if cli.Get_Secret(result.Command.Secrets, "MODE").Value != "fast" {
		t.Fatal("MODE did not validate")
	}
	if cli.Get_Secret(result.Command.Secrets, "OPTIONAL").Value != "" {
		t.Fatal("an absent optional secret did not expose its zero value")
	}
	if driver.Introspect().Raw_Open != 0 {
		t.Fatal("the parser did not close all secret files")
	}
}

// Test_Parse_Secret_Bounds_And_Errors verifies the raw size boundary, regular-file rule, joined
// declaration order, and secret-value redaction.
func Test_Parse_Secret_Bounds_And_Errors(t *testing.T) {
	loop, driver, _ := sharedio.New_Sim(9)
	if make_err := loop.Make_Directory("/secrets/DIRECTORY"); make_err != nil {
		t.Fatalf("make directory: %v", make_err)
	}
	seed_secret_files(t, loop, driver, []secret_file{
		{Path: "/secrets/BOUNDARY", Content: strings.Repeat("x", cli.SECRET_BYTES_MAX)},
		{Path: "/secrets/OVERFLOW", Content: strings.Repeat("y", cli.SECRET_BYTES_MAX+1)},
		{Path: "/secrets/NUMBER", Content: "private-non-number"},
	})
	program := external_program(nil, []cli.Secret{
		cli.New_Secret[string](cli.New_Secret_Input{
			Paths: []string{"/secrets/BOUNDARY"}, Required: true,
		}),
		cli.New_Secret[string](cli.New_Secret_Input{
			Paths: []string{"/missing/OVERFLOW", "/secrets/OVERFLOW"}, Required: true,
		}),
		cli.New_Secret[string](cli.New_Secret_Input{
			Paths: []string{"/secrets/DIRECTORY"}, Required: true,
		}),
		cli.New_Secret[int](cli.New_Secret_Input{
			Paths: []string{"/secrets/NUMBER"}, Required: true,
		}),
	})
	result := drive_parser(t, driver, cli.Program_Parse(&program, cli.Program_Parse_Input{
		Arguments: []string{"external"}, Loop: loop,
	}))
	if result.Error == nil {
		t.Fatal("invalid secret paths did not fail")
	}
	if cli.Get_Secret(result.Command.Secrets, "BOUNDARY").Value !=
		strings.Repeat("x", cli.SECRET_BYTES_MAX) {
		t.Fatal("the exact size boundary did not resolve")
	}
	message := result.Error.Error()
	overflow_offset := strings.Index(message, "OVERFLOW")
	directory_offset := strings.Index(message, "DIRECTORY")
	number_offset := strings.Index(message, "NUMBER")
	if overflow_offset < 0 {
		t.Fatalf("OVERFLOW error is absent: %v", result.Error)
	}
	if directory_offset < overflow_offset {
		t.Fatalf("DIRECTORY error is out of order: %v", result.Error)
	}
	if number_offset < directory_offset {
		t.Fatalf("secret errors are not in declaration order: %v", result.Error)
	}
	if !strings.Contains(message, "/missing/OVERFLOW") {
		t.Fatalf("the missing fallback cause is absent: %v", result.Error)
	}
	if !strings.Contains(message, "/secrets/OVERFLOW") {
		t.Fatalf("fallback causes are absent: %v", result.Error)
	}
	if strings.Contains(message, "private-non-number") {
		t.Fatal("a secret error disclosed secret content")
	}
}

// Test_Parse_External_Short_Circuit verifies that help and argument errors do not need a loop or
// an external source.
func Test_Parse_External_Short_Circuit(t *testing.T) {
	program := cli.New_Single(cli.New_Single_Input{
		Label: "external",
		Arguments: []cli.Option{
			cli.New_Argument[string](cli.New_Argument_Input{Label: "target"}),
		},
		Secrets: []cli.Secret{
			cli.New_Secret[string](cli.New_Secret_Input{
				Paths: []string{"/secrets/TOKEN"}, Required: true,
			}),
		},
	})
	help := cli.Parser_Done(cli.Program_Parse(&program, cli.Program_Parse_Input{
		Arguments: []string{"external", "-help"},
	}))
	if help == nil {
		t.Fatal("help did not complete synchronously")
	}
	if !errors.Is(help.Error, cli.Help_Requested) {
		t.Fatalf("help did not complete synchronously: %+v", help)
	}
	invalid := cli.Parser_Done(cli.Program_Parse(&program, cli.Program_Parse_Input{
		Arguments: []string{"external"},
	}))
	if invalid == nil {
		t.Fatal("the CLI error did not complete synchronously")
	}
	if invalid.Error == nil {
		t.Fatal("the CLI error did not complete synchronously")
	}
}

// Test_External_Help_And_Deprecation verifies public defaults and paths in help, omission rules,
// and source-triggered warnings without secret values.
func Test_External_Help_And_Deprecation(t *testing.T) {
	loop, driver, _ := sharedio.New_Sim(2)
	seed_secret_files(t, loop, driver, []secret_file{
		{Path: "/secrets/OLD_SECRET", Content: "do-not-print"},
	})
	program := cli.New_Single(cli.New_Single_Input{
		Label: "external",
		Environment_Variables: []cli.Environment_Variable{
			cli.New_Environment_Variable(cli.New_Environment_Variable_Input[string]{
				Key: "PUBLIC", Value: "default", Description: "public value",
			}),
			cli.New_Environment_Variable(cli.New_Environment_Variable_Input[string]{
				Key: "HIDDEN_ENV", Hidden: true,
			}),
			cli.New_Environment_Variable(cli.New_Environment_Variable_Input[string]{
				Key: "OLD_ENV", Deprecated: "use PUBLIC",
			}),
			cli.New_Environment_Variable(cli.New_Environment_Variable_Input[string]{
				Key: "EMPTY_OLD", Deprecated: "use PUBLIC",
			}),
		},
		Secrets: []cli.Secret{
			cli.New_Secret[string](cli.New_Secret_Input{
				Paths: []string{"/secrets/TOKEN"}, Description: "token",
			}),
			cli.New_Secret[string](cli.New_Secret_Input{
				Paths: []string{"/secrets/HIDDEN_SECRET"}, Hidden: true,
			}),
			cli.New_Secret[string](cli.New_Secret_Input{
				Paths: []string{"/secrets/OLD_SECRET"}, Deprecated: "use TOKEN",
			}),
		},
	})
	text := assert_external_help(t, program)
	result := drive_parser(t, driver, cli.Program_Parse(&program, cli.Program_Parse_Input{
		Arguments:   []string{"external"},
		Environment: []string{"OLD_ENV=value", "EMPTY_OLD="},
		Loop:        loop,
	}))
	warnings := strings.Join(result.Command.Deprecation_Warnings, "\n")
	if !strings.Contains(warnings, "OLD_ENV") {
		t.Fatalf("the environment warning is absent: %q", warnings)
	}
	if !strings.Contains(warnings, "OLD_SECRET") {
		t.Fatalf("external deprecation warnings are absent: %q", warnings)
	}
	if strings.Contains(warnings, "EMPTY_OLD") {
		t.Fatalf("an absent value added a deprecation warning: %q", warnings)
	}
	if strings.Contains(warnings, "do-not-print") {
		t.Fatal("a warning disclosed secret content")
	}
	if strings.Contains(text, "do-not-print") {
		t.Fatal("help or a warning disclosed secret content")
	}
}

// Verifies the public external declarations and all help omission rules.
func assert_external_help(t *testing.T, program cli.Program) (text string) {
	t.Helper()
	help := strings.Builder{}
	cli.Print_Help(&help, program)
	text = help.String()
	for _, expected := range []string{
		"Environment Variables:", "PUBLIC", "default", "Secrets:", "/secrets/TOKEN",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("help does not contain %q:\n%s", expected, text)
		}
	}
	hidden_declarations := []string{
		"HIDDEN_ENV", "OLD_ENV", "EMPTY_OLD", "HIDDEN_SECRET", "OLD_SECRET",
	}
	for _, hidden := range hidden_declarations {
		if strings.Contains(text, hidden) {
			t.Fatalf("help contains hidden declaration %q:\n%s", hidden, text)
		}
	}
	return text
}

// Verifies external declaration validation at the program construction boundary.
func assert_external_validation(t *testing.T) {
	t.Helper()
	invalid_programs := []func(){
		func() {
			external_program([]cli.Environment_Variable{
				cli.New_Environment_Variable(
					cli.New_Environment_Variable_Input[string]{
						Key: "lower",
					},
				),
			}, nil)
		},
		func() {
			external_program([]cli.Environment_Variable{
				cli.New_Environment_Variable(
					cli.New_Environment_Variable_Input[string]{
						Key: "DUPLICATE",
					},
				),
				cli.New_Environment_Variable(
					cli.New_Environment_Variable_Input[string]{
						Key: "DUPLICATE",
					},
				),
			}, nil)
		},
		func() {
			external_program([]cli.Environment_Variable{
				cli.New_Environment_Variable(
					cli.New_Environment_Variable_Input[string]{
						Key: "COLLISION",
					},
				),
			}, []cli.Secret{
				cli.New_Secret[string](cli.New_Secret_Input{
					Paths: []string{"/secrets/COLLISION"},
				}),
			})
		},
		func() {
			external_program([]cli.Environment_Variable{
				cli.New_Environment_Variable(
					cli.New_Environment_Variable_Input[string]{
						Key: "REQUIRED", Required: true, Value: "default",
					},
				),
			}, nil)
		},
	}
	for index, construct := range invalid_programs {
		reason := "invalid external declaration " + string(rune('A'+index))
		assert_panics(t, reason, construct)
	}
	assert_invalid_secret_declarations(t)
}

// Verifies path validation separately so the external validation check stays small.
func assert_invalid_secret_declarations(t *testing.T) {
	t.Helper()
	invalid_secrets := []func(){
		func() {
			external_program(nil, []cli.Secret{
				cli.New_Secret[string](cli.New_Secret_Input{}),
			})
		},
		func() {
			external_program(nil, []cli.Secret{
				cli.New_Secret[string](cli.New_Secret_Input{
					Paths: []string{"relative/TOKEN"},
				}),
			})
		},
		func() {
			external_program(nil, []cli.Secret{
				cli.New_Secret[string](cli.New_Secret_Input{
					Paths: []string{"/one/TOKEN", "/two/OTHER"},
				}),
			})
		},
	}
	for index, construct := range invalid_secrets {
		reason := "invalid secret declaration " + string(rune('A'+index))
		assert_panics(t, reason, construct)
	}
}

// One small program supplies an external declaration fixture without a command selector.
func external_program(
	environment []cli.Environment_Variable,
	secrets []cli.Secret,
) (program cli.Program) {
	return cli.New_Single(cli.New_Single_Input{
		Label: "external", Environment_Variables: environment, Secrets: secrets,
	})
}

// A test file contains one path and the bytes that public filesystem operations must write.
type secret_file struct {
	Path    string
	Content string
}

// The simulator has no scripted file surface, so the test owns each descriptor lifecycle.
func seed_secret_files(
	t *testing.T,
	loop sharedio.IO,
	driver sharedio.Driver,
	files []secret_file,
) {
	t.Helper()
	for _, source := range files {
		if make_err := loop.Make_Directory(path.Dir(source.Path)); make_err != nil {
			t.Fatalf("make parent directory: %v", make_err)
		}
		file, create_err := loop.Create(source.Path)
		if create_err != nil {
			t.Fatalf("create secret file: %v", create_err)
		}
		completed := false
		var completion sharedio.Completion
		loop.Write(&completion, func(_ *sharedio.Completion, count int, err error) {
			if err != nil {
				t.Errorf("write secret file: %v", err)
			}
			if count != len(source.Content) {
				t.Errorf(
					"write secret file count = %d, want %d",
					count,
					len(source.Content),
				)
			}
			completed = true
		}, file, []byte(source.Content), 0)
		drive_sim_operation(t, driver, func() (finished bool) { return completed })
		completed = false
		loop.Close(&completion, func(_ *sharedio.Completion, err error) {
			if err != nil {
				t.Errorf("close secret file: %v", err)
			}
			completed = true
		}, file)
		drive_sim_operation(t, driver, func() (finished bool) { return completed })
	}
}

// The test root drives the simulator until one public operation retires.
func drive_sim_operation(
	t *testing.T,
	driver sharedio.Driver,
	done func() (finished bool),
) {
	t.Helper()
	completed, drive_err := driver.Run_Until(done, sharedio.FOREVER)
	if drive_err != nil {
		t.Fatalf("drive simulator: %v", drive_err)
	}
	if !completed {
		t.Fatal("simulator operation did not complete")
	}
}

// The test root drives the parser because shared/cli cannot own the application timeline.
func drive_parser(
	t *testing.T,
	driver sharedio.Driver,
	parser *cli.Parser,
) (result *cli.Parse_Result) {
	t.Helper()
	drive_sim_operation(t, driver, func() (finished bool) {
		return cli.Parser_Done(parser) != nil
	})
	return cli.Parser_Done(parser)
}

// Asserts New rejects the enum-specific malformations and the reserved -help label.
func assert_new_enum_validation(t *testing.T) {
	// An enum flag's default must be one of its permitted values.
	assert_panics(t, "enum flag default outside its set", func() {
		cli.New_Single(cli.New_Single_Input{
			Label: "prog",
			Flags: []cli.Option{
				cli.New_Enum_Flag(cli.New_Enum_Flag_Input[string]{
					Label: "color", Enum: []string{"auto", "never"},
					Value: "rainbow",
				}),
			},
		})
	})
	// An enum with no permitted values is malformed.
	assert_panics(t, "empty enum set", func() {
		cli.New_Single(cli.New_Single_Input{
			Label: "prog",
			Arguments: []cli.Option{
				cli.New_Enum_Argument(cli.New_Enum_Argument_Input[string]{
					Label: "format", Enum: []string{},
				}),
			},
		})
	})
	// The enum's element type must match the option's value type.
	assert_panics(t, "enum element type mismatch", func() {
		cli.New_Single(cli.New_Single_Input{
			Label:     "prog",
			Arguments: []cli.Option{{Label: "format", Value: "", Enum: []int{1, 2}}},
		})
	})
	// A variadic argument cannot also carry an enum: enums are single-valued.
	assert_panics(t, "enum on a variadic argument", func() {
		cli.New_Single(cli.New_Single_Input{
			Label: "prog",
			Arguments: []cli.Option{
				{Label: "path", Value: []string{}, Enum: []string{"a", "b"}},
			},
		})
	})
	// User declarations cannot change the function of a default help flag.
	assert_panics(t, "user option named help", func() {
		cli.New_Single(cli.New_Single_Input{
			Label: "tool",
			Flags: []cli.Option{{Label: "help", Value: false}},
		})
	})
	assert_panics(t, "user option named h", func() {
		cli.New_Single(cli.New_Single_Input{
			Label: "tool",
			Flags: []cli.Option{{Label: "h", Value: false}},
		})
	})
}

// Parses arguments against a copy of the program and asserts the named variadic
// argument equals want. The program is taken by value so callers can reuse it.
func assert_variadic(
	t *testing.T, program cli.Program, arguments []string, label string, want ...string,
) {
	t.Helper()
	command, err := parse_program(&program, arguments)
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
