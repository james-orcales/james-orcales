package cli_test

import (
	"errors"
	"path"
	"testing"
	"unsafe"

	"local/james-orcales/shared/cli"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/simulation/nbio"
	"local/james-orcales/shared/simulation/time"
	"local/james-orcales/shared/slices"
	"local/james-orcales/shared/strconv"
	"local/james-orcales/shared/strings"
	"local/james-orcales/shared/testify"
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
	if !text_contains(err.Error(), `did you mean "list"`) {
		t.Errorf("expected a suggestion of list, got %v", err)
	}
	_, err = parse_program(&fixture.Program, []string{"todoctl", "zzzzzzzz"})
	if err == nil {
		t.Fatal("expected an error for unknown command zzzzzzzz")
	}
	if text_contains(err.Error(), "did you mean") {
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
	if cli.Option_String(cli.Get_Option(command.Arguments, "path")) != "./src" {
		t.Errorf("expected path ./src, got %q",
			cli.Option_String(cli.Get_Option(command.Arguments, "path")))
	}

	command, err = parse_program(&program, []string{"sloc", "./src", "-hidden"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cli.Option_Boolean(cli.Get_Option(command.Flags, "hidden")) {
		t.Error("expected hidden true")
	}

	// A token that would select a sibling command in a multi-command program is just
	// a positional here: a single-command program has no selector namespace.
	command, err = parse_program(&program, []string{"sloc", "help"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Option_String(cli.Get_Option(command.Arguments, "path")) != "help" {
		t.Errorf("expected path help, got %q",
			cli.Option_String(cli.Get_Option(command.Arguments, "path")))
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
	if cli.Option_String(cli.Get_Option(command.Arguments, "task")) != "milk" {
		t.Errorf("expected task milk, got %v",
			cli.Option_String(cli.Get_Option(command.Arguments, "task")))
	}

	// An unknown binary name suggests the closest command.
	_, err = parse_program(&program, []string{"ad"})
	if err == nil {
		t.Fatal("expected an error for the unknown multicall name ad")
	}
	if !text_contains(err.Error(), `did you mean "add"`) {
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
	if cli.Option_Integer(cli.Get_Option(command.Arguments, "id")) != 3 {
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
	if cli.Option_String(cli.Get_Option(command.Arguments, "task")) != "hello" {
		t.Errorf("expected task hello, got %v",
			cli.Option_String(cli.Get_Option(command.Arguments, "task")))
	}

	// Named and positional tokens may appear in any order.
	command, err = parse_program(&fixture.Program,
		[]string{"todoctl", "add", "-priority=high", "world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Option_String(cli.Get_Option(command.Arguments, "task")) != "world" {
		t.Error("expected task world from a positional after a flag")
	}
	if cli.Option_String(cli.Get_Option(command.Flags, "priority")) != "high" {
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
	if cli.Option_String(cli.Get_Option(command.Arguments, "first")) != "x" {
		t.Error("expected first x")
	}
	if cli.Option_String(cli.Get_Option(command.Arguments, "second")) != "y" {
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
	if !text_contains(err.Error(), "did you mean -priority") {
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
	if !slices.Equal(
		cli.Option_Integers(cli.Get_Option(command.Arguments, "n")),
		[]int{1, 2, 3},
	) {
		t.Errorf(
			"expected [1 2 3], got %v",
			cli.Option_Integers(cli.Get_Option(command.Arguments, "n")),
		)
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
	if cli.Option_String(cli.Get_Option(command.Flags, "priority")) != "high" {
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
			cli.New_String_Enum_Flag(cli.New_String_Enum_Flag_Input{
				Label: "color", Enum: []string{"auto", "never", "always"},
				Value: "auto", Description: "when to colorize",
			}),
			cli.New_Integer_Enum_Flag(cli.New_Integer_Enum_Flag_Input{
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
	color := cli.Option_String(cli.Get_Option(command.Flags, "color"))
	if color != "never" {
		t.Errorf("expected never, got %q", color)
	}

	// An omitted enum keeps its default.
	command, err = parse_program(&program, []string{"prog"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Option_String(cli.Get_Option(command.Flags, "color")) != "auto" {
		t.Errorf("expected default auto, got %q",
			cli.Option_String(cli.Get_Option(command.Flags, "color")))
	}

	// A near miss suggests the closest member.
	_, err = parse_program(&program, []string{"prog", "-color=nevr"})
	if err == nil {
		t.Fatal("expected an error for an out-of-set value")
	}
	if !text_contains(err.Error(), `did you mean "never"`) {
		t.Errorf("expected a suggestion of never, got %v", err)
	}

	// A wild miss lists the whole set instead of guessing.
	_, err = parse_program(&program, []string{"prog", "-color=purple"})
	if err == nil {
		t.Fatal("expected an error for an out-of-set value")
	}
	if !text_contains(err.Error(), "allowed: auto, never, always") {
		t.Errorf("expected the allowed list, got %v", err)
	}

	// An int enum accepts a member and rejects a non-member with the allowed list.
	command, err = parse_program(&program, []string{"prog", "-level=4"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	level := cli.Option_Integer(cli.Get_Option(command.Flags, "level"))
	if level != 4 {
		t.Errorf("expected 4, got %v", level)
	}
	_, err = parse_program(&program, []string{"prog", "-level=3"})
	if err == nil {
		t.Fatal("expected an error for an out-of-set int value")
	}
	if !text_contains(err.Error(), "allowed: 1, 2, 4, 8") {
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
	testify.Error_Is(t, err, cli.Help_Requested, "-help")
	short_context, err := parse_program(&single, []string{"tool", "-h"})
	testify.Error_Is(t, err, cli.Help_Requested, "-h")
	short_help := output_buffer{}
	cli.Print_Requested_Help(output_cli(&short_help), single, short_context)
	testify.Contains_Any(t, short_help.String(), "does a thing", "-h program help")

	// Multi-command: a command then -help resolves that command as the context.
	fixture := new_cli_fixture()
	command, err := parse_program(&fixture.Program, []string{"todoctl", "list", "-help"})
	testify.Error_Is(t, err, cli.Help_Requested, "command help")
	testify.Equal(t, "list", string(command.Label), "command help context")

	// Multi-command with -help but no command selected → root context (empty label).
	fixture = new_cli_fixture()
	command, err = parse_program(&fixture.Program, []string{"todoctl", "-help"})
	testify.Error_Is(t, err, cli.Help_Requested, "root help")
	testify.Empty(t, command.Label, "root help context")
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
	dashed := cli_complete(program, []string{"add", "-"})
	for _, expected := range []string{"-h", "-help", "-task"} {
		if !candidates_contain(dashed, expected) {
			t.Errorf("expected %q among %v", expected, dashed)
		}
	}

	// Handle_Completion serves __complete, printing candidates one per line.
	output := output_buffer{}
	args := []string{"toolbox", "__complete", "add", "-"}
	testify.True(
		t, handle_completion(program, args, output_cli(&output)), "__complete",
	)
	testify.Contains_Any(t, output.String(), "-task", "completion output")
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
	if !cli.Option_Boolean(cli.Get_Option(command.Flags, "secret")) {
		t.Error("expected secret true")
	}
	if _, err = parse_program(&program, []string{"tool", "ghost"}); err != nil {
		t.Fatalf("hidden command should resolve: %v", err)
	}

	// Help shows the visible names and omits the hidden ones.
	help := output_buffer{}
	cli.Print_Help(output_cli(&help), program)
	if !text_contains(help.String(), "verbose") {
		t.Errorf("help should show a visible flag:\n%s", help.String())
	}
	if text_contains(help.String(), "secret") {
		t.Errorf("help must not show a hidden flag:\n%s", help.String())
	}
	if text_contains(help.String(), "ghost") {
		t.Errorf("help must not show a hidden command:\n%s", help.String())
	}

	// Completion omits both.
	if candidates_contain(cli_complete(program, []string{"tool", ""}), "ghost") {
		t.Error("completion must not offer a hidden command")
	}
	if candidates_contain(cli_complete(program, []string{"tool", "run", "-"}), "-secret") {
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
	if cli.Option_String(cli.Get_Option(command.Flags, "old-name")) != "ada" {
		t.Error("expected old-name ada")
	}
	warnings := output_buffer{}
	cli.Print_Deprecations(output_cli(&warnings), command)
	if !text_contains(warnings.String(), "use -name") {
		t.Errorf("expected a deprecation warning, got %q", warnings.String())
	}

	// Not using it records nothing.
	quiet, _ := parse_program(&program, []string{"tool", "-name=bob"})
	silence := output_buffer{}
	cli.Print_Deprecations(output_cli(&silence), quiet)
	if silence.String() != "" {
		t.Errorf("expected no warning when unused, got %q", silence.String())
	}

	// Help omits it.
	help := output_buffer{}
	cli.Print_Help(output_cli(&help), program)
	if text_contains(help.String(), "old-name") {
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
		got := string(cli.Option_String(cli.Get_Option(command.Flags, "flag")))
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
	if cli.Option_String(cli.Get_Option(options, "b")) != "y" {
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
	if cli.Option_String(cli.Get_Option(command.Arguments, "task")) != "milk" {
		t.Errorf("expected task milk, got %v",
			cli.Option_String(cli.Get_Option(command.Arguments, "task")))
	}
}

// Test_Parse_Enum_Argument verifies an argument-form enum: it is settable by position
// and by name, an omitted value yields the existing missing-required error, and an
// out-of-set value is rejected with the allowed list.
func Test_Parse_Enum_Argument(t *testing.T) {
	program := cli.New_Single(cli.New_Single_Input{
		Label: "prog", Description: "enum argument",
		Arguments: []cli.Option{
			cli.New_String_Enum_Argument(cli.New_String_Enum_Argument_Input{
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
	if cli.Option_String(cli.Get_Option(command.Arguments, "format")) != "yaml" {
		t.Errorf("expected yaml, got %q",
			cli.Option_String(cli.Get_Option(command.Arguments, "format")))
	}

	// Accepted by name.
	command, err = parse_program(&program, []string{"prog", "-format=toml"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Option_String(cli.Get_Option(command.Arguments, "format")) != "toml" {
		t.Errorf("expected toml, got %q",
			cli.Option_String(cli.Get_Option(command.Arguments, "format")))
	}

	// Omitted → the existing missing-required-argument error.
	_, err = parse_program(&program, []string{"prog"})
	if err == nil {
		t.Fatal("expected an error for a missing required enum argument")
	}
	if !text_contains(err.Error(), "missing required argument") {
		t.Errorf("expected the missing-required message, got %v", err)
	}

	// A non-member is rejected with the allowed list.
	_, err = parse_program(&program, []string{"prog", "xml"})
	if err == nil {
		t.Fatal("expected an error for an out-of-set argument")
	}
	if !text_contains(err.Error(), "allowed: json, yaml, toml") {
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
		cli.New_String_Enum_Environment_Variable(
			cli.New_String_Enum_Environment_Variable_Input{
				Key: "MODE", Value: "safe", Enum: []string{"safe", "fast"},
			}),
		cli.New_Integer_Enum_Environment_Variable(
			cli.New_Integer_Enum_Environment_Variable_Input{
				Key: "LEVEL", Value: 80, Enum: []int{80, 81},
			}),
		cli.New_Environment_Variable(cli.New_Environment_Variable_Input[string]{
			Key: "EMPTY", Value: "fallback", Allow_Empty: true,
		}),
	}, nil)
	var parser cli.Parser
	cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
		Arguments: []string{"external"},
		Environment: []string{
			"UNDECLARED", "NAME=service", "ENABLED=TRUE", "MODE=fast", "LEVEL=81",
			"EMPTY=",
		},
	})
	result, done := cli.Parser_Done(&parser)
	if !done {
		t.Fatal("a parser without secrets did not complete immediately")
	}
	if result.Error != nil {
		t.Fatalf("parse environment: %v", result.Error)
	}
	assert_environment_values(t, result)
}

func assert_environment_values(t *testing.T, result cli.Parse_Result) {
	t.Helper()
	if cli.Environment_String(
		cli.Get_Environment(result.Command.Environment, "NAME"),
	) != "service" {
		t.Fatal("NAME did not resolve")
	}
	if cli.Environment_Integer(
		cli.Get_Environment(result.Command.Environment, "PORT"),
	) != 80 {
		t.Fatal("PORT did not keep its default")
	}
	if !cli.Environment_Boolean(
		cli.Get_Environment(result.Command.Environment, "ENABLED"),
	) {
		t.Fatal("ENABLED did not use strconv.ParseBool")
	}
	if cli.Environment_String(
		cli.Get_Environment(result.Command.Environment, "MODE"),
	) != "fast" {
		t.Fatal("MODE did not resolve its enum member")
	}
	if cli.Environment_Integer(
		cli.Get_Environment(result.Command.Environment, "LEVEL"),
	) != 81 {
		t.Fatal("LEVEL did not resolve its enum member")
	}
	if cli.Environment_String(
		cli.Get_Environment(result.Command.Environment, "EMPTY"),
	) != "" {
		t.Fatal("EMPTY did not preserve its permitted empty value")
	}
	if len(result.Command.Deprecation_Warnings) != 0 {
		t.Fatal("absent deprecations published empty warning slots")
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
		cli.New_String_Enum_Environment_Variable(
			cli.New_String_Enum_Environment_Variable_Input{
				Key: "THIRD", Required: true, Enum: []string{"yes", "no"},
			}),
	}, nil)
	var parser cli.Parser
	cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
		Arguments: []string{"external"},
		Environment: []string{
			"UNDECLARED", "FIRST", "SECOND=x", "SECOND=2", "THIRD=maybe",
		},
	})
	result, done := cli.Parser_Done(&parser)
	if !done {
		t.Fatal("the environment parser did not complete")
	}
	if result.Error == nil {
		t.Fatal("malformed, duplicate, and invalid entries did not fail")
	}
	message := result.Error.Error()
	first_offset := text_index(message, "FIRST")
	second_offset := text_index(message, "SECOND")
	third_offset := text_index(message, "THIRD")
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

// Test_Bounded_External_Integer_Errors keeps every scalar edge in enum diagnostics.
func Test_Bounded_External_Integer_Errors(t *testing.T) {
	values := [...]int{bits.INTEGER_MINIMUM, bits.INTEGER_MAXIMUM, -1, 0, 1, 2}
	key_sizes := [...]int{1, 2}
	for _, key_size := range key_sizes {
		key := cli_boundary_text(key_size, 'I')
		program := external_program([]cli.Environment_Variable{
			cli.New_Integer_Enum_Environment_Variable(
				cli.New_Integer_Enum_Environment_Variable_Input{
					Key: cli.External_Key(key), Value: 3, Enum: []int{3},
				},
			),
		}, nil)
		for _, value := range values {
			var parser cli.Parser
			cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
				Arguments:   []string{"external"},
				Environment: []string{key + "=" + cli_decimal_text(value)},
			})
			result, complete := cli.Parser_Done(&parser)
			if !complete {
				t.Fatal("integer enum environment parse did not complete")
			}
			if result.Error == nil {
				t.Fatalf("integer enum accepted %d outside its set", value)
			}
		}
	}
}

// Test_Bounded_External_String_Errors keeps short raw and key edges observable.
func Test_Bounded_External_String_Errors(t *testing.T) {
	profiles := [...]struct{ Key, Raw string }{
		{"S", ""}, {"SS", "x"}, {"S", "xx"},
	}
	for _, profile := range profiles {
		program := external_program([]cli.Environment_Variable{
			cli.New_String_Enum_Environment_Variable(
				cli.New_String_Enum_Environment_Variable_Input{
					Key: cli.External_Key(profile.Key), Value: "allowed",
					Enum: []string{"allowed"}, Allow_Empty: true,
				},
			),
		}, nil)
		var parser cli.Parser
		cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
			Arguments:   []string{"external"},
			Environment: []string{profile.Key + "=" + profile.Raw},
		})
		result, complete := cli.Parser_Done(&parser)
		if !complete {
			t.Fatal("string enum environment parse did not complete")
		}
		if result.Error == nil {
			t.Fatalf("string enum accepted %q outside its set", profile.Raw)
		}
	}
}

// Test_Bounded_External_Boolean_Error keeps the Boolean conversion member observable.
func Test_Bounded_External_Boolean_Error(t *testing.T) {
	program := external_program([]cli.Environment_Variable{
		cli.New_Environment_Variable[bool](cli.New_Environment_Variable_Input[bool]{
			Key: "B", Value: false,
		}),
	}, nil)
	var parser cli.Parser
	cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
		Arguments: []string{"external"}, Environment: []string{"B=maybe"},
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("Boolean conversion error did not complete")
	}
	if result.Error == nil {
		t.Fatal("Boolean conversion accepted invalid text")
	}
}

// Test_Bounded_External_Enum_Members keeps the maximum validated set observable.
func Test_Bounded_External_Enum_Members(t *testing.T) {
	integers := make([]int, slices.COUNT_MAXIMUM)
	strings_enum := make([]string, slices.COUNT_MAXIMUM)
	for index := range integers {
		integers[index] = 3
		strings_enum[index] = "allowed"
	}
	integer_edges := [...]int{bits.INTEGER_MINIMUM, bits.INTEGER_MAXIMUM, -1, 0, 1, 2}
	copy(integers, integer_edges[:])
	cli_bounded_integer_enum_members(t, integers)
	cli_bounded_string_enum_members(t, strings_enum)
}

func cli_bounded_integer_enum_members(t *testing.T, members []int) {
	t.Helper()
	program := external_program([]cli.Environment_Variable{
		cli.New_Integer_Enum_Environment_Variable(
			cli.New_Integer_Enum_Environment_Variable_Input{
				Key: "I", Value: 3, Enum: members,
			},
		),
	}, nil)
	assert_panics(t, "maximum integer enum exceeds diagnostic text", func() {
		var parser cli.Parser
		cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
			Arguments: []string{"external"}, Environment: []string{"I=4"},
		})
	})
}

func cli_bounded_string_enum_members(t *testing.T, members []string) {
	t.Helper()
	program := external_program([]cli.Environment_Variable{
		cli.New_String_Enum_Environment_Variable(
			cli.New_String_Enum_Environment_Variable_Input{
				Key: "S", Value: "allowed", Enum: members,
			},
		),
	}, nil)
	assert_panics(t, "maximum string enum exceeds diagnostic text", func() {
		var parser cli.Parser
		cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
			Arguments:   []string{"external"},
			Environment: []string{"S=zzzzzzzzzzzz"},
		})
	})
}

// Test_Bounded_Secret_Conversion_Errors keeps raw payload and key edges observable.
func Test_Bounded_Secret_Conversion_Errors(t *testing.T) {
	profiles := [...]struct{ Key_Size, Raw_Size int }{
		{1, 0}, {2, 1}, {1, 2},
	}
	for _, profile := range profiles {
		key := cli_boundary_text(profile.Key_Size, 'X')
		secret_path := "/" + key
		raw_bytes := make([]byte, profile.Raw_Size)
		for index := range raw_bytes {
			raw_bytes[index] = 'x'
		}
		raw := string(raw_bytes)
		loop, driver := cli_sim_loop(1)
		seed_secret_files(t, loop, driver, []secret_file{
			{Path: secret_path, Content: raw},
		})
		program := external_program(nil, []cli.Secret{
			cli.New_Secret[int](cli.New_Secret_Input{
				Paths: []string{secret_path}, Required: true, Allow_Empty: true,
			}),
		})
		var parser cli.Parser
		cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
			Arguments: []string{"external"}, Loop: loop,
		})
		result := drive_parser(t, driver, &parser)
		if result.Error == nil {
			t.Fatalf("integer secret accepted %d invalid bytes", profile.Raw_Size)
		}
		nbio.IO_Deinit(loop)
	}
}

// Verifies ordered fallback, terminal-newline removal, typed conversion,
// optional absence, parallel parser completion, and descriptor retirement.
func assert_parse_secrets(t *testing.T) {
	t.Helper()
	loop, driver := cli_sim_loop(6)
	seed_secret_files(t, loop, driver, []secret_file{
		{Path: "/second/TOKEN", Content: "value\r\n"},
		{Path: "/secrets/COUNT", Content: "42\n"},
		{Path: "/secrets/ENABLED", Content: "1"},
		{Path: "/secrets/MODE", Content: "fast"},
		{Path: "/secrets/LEVEL", Content: "81"},
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
		cli.New_String_Enum_Secret(cli.New_String_Enum_Secret_Input{
			Paths: []string{"/secrets/MODE"}, Required: true,
			Enum: []string{"safe", "fast"},
		}),
		cli.New_Integer_Enum_Secret(cli.New_Integer_Enum_Secret_Input{
			Paths: []string{"/secrets/LEVEL"}, Required: true,
			Enum: []int{80, 81},
		}),
		cli.New_Secret[string](cli.New_Secret_Input{
			Paths: []string{"/secrets/OPTIONAL"},
		}),
	})
	var parser cli.Parser
	cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
		Arguments: []string{"external"}, Loop: loop,
	})
	_, done := cli.Parser_Done(&parser)
	if done {
		t.Fatal("a parser with available secrets completed before the loop ran")
	}
	result := drive_parser(t, driver, &parser)
	if result.Error != nil {
		t.Fatalf("parse secrets: %v", result.Error)
	}
	if string(cli.Secret_String(cli.Get_Secret(result.Command.Secrets, "TOKEN"))) != "value" {
		t.Fatal("TOKEN did not use its fallback path or remove CRLF")
	}
	if cli.Secret_Integer(cli.Get_Secret(result.Command.Secrets, "COUNT")) != 42 {
		t.Fatal("COUNT did not convert")
	}
	if !cli.Secret_Boolean(cli.Get_Secret(result.Command.Secrets, "ENABLED")) {
		t.Fatal("ENABLED did not convert")
	}
	if string(cli.Secret_String(cli.Get_Secret(result.Command.Secrets, "MODE"))) != "fast" {
		t.Fatal("MODE did not validate")
	}
	if cli.Secret_Integer(cli.Get_Secret(result.Command.Secrets, "LEVEL")) != 81 {
		t.Fatal("LEVEL did not validate")
	}
	if len(cli.Secret_String(cli.Get_Secret(result.Command.Secrets, "OPTIONAL"))) != 0 {
		t.Fatal("an absent optional secret did not expose its zero value")
	}
	nbio.IO_Deinit(loop)
}

// Test_Parse_Secret_Bounds_And_Errors verifies the raw size boundary, regular-file rule, joined
// declaration order, and secret-value redaction.
func Test_Parse_Secret_Bounds_And_Errors(t *testing.T) {
	loop := cli_secret_boundary_loop()
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
		cli.New_Secret[string](cli.New_Secret_Input{
			Paths: []string{"/secrets/NEGATIVE"}, Required: true,
		}),
		cli.New_Secret[string](cli.New_Secret_Input{
			Paths: []string{"/secrets/READ_OVERFLOW"}, Required: true,
		}),
		{
			Key: "BOUNDARY_ENUM", Paths: []string{"/secrets/BOUNDARY_ENUM"},
			Value: "", Enum: []string{"allowed"}, Required: true,
		},
	})
	var parser cli.Parser
	cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
		Arguments: []string{"external"}, Loop: loop,
	})
	result, done := cli.Parser_Done(&parser)
	if !done {
		t.Fatal("boundary parser did not complete")
	}
	if result.Error == nil {
		t.Fatal("invalid secret paths did not fail")
	}
	cli_assert_secret_boundary_result(t, result)
}

func cli_assert_secret_boundary_result(t *testing.T, result cli.Parse_Result) {
	t.Helper()
	boundary := cli.Secret_String(cli.Get_Secret(result.Command.Secrets, "BOUNDARY"))
	if len(boundary) != cli.SECRET_BYTES_MAX {
		t.Fatal("the exact size boundary did not resolve")
	}
	for _, character := range boundary {
		if character != 'x' {
			t.Fatal("the exact size boundary changed content")
		}
	}
	message := result.Error.Error()
	overflow_offset := text_index(message, "OVERFLOW")
	directory_offset := text_index(message, "DIRECTORY")
	number_offset := text_index(message, "NUMBER")
	if overflow_offset < 0 {
		t.Fatalf("OVERFLOW error is absent: %v", result.Error)
	}
	if directory_offset < overflow_offset {
		t.Fatalf("DIRECTORY error is out of order: %v", result.Error)
	}
	if number_offset < directory_offset {
		t.Fatalf("secret errors are not in declaration order: %v", result.Error)
	}
	if !text_contains(message, "/missing/OVERFLOW") {
		t.Fatalf("the missing fallback cause is absent: %v", result.Error)
	}
	if !text_contains(message, "/secrets/OVERFLOW") {
		t.Fatalf("fallback causes are absent: %v", result.Error)
	}
	if text_contains(message, "private-non-number") {
		t.Fatal("a secret error disclosed secret content")
	}
}

// Test_Bounded_External_Enum_Keys keeps full resolved keys out of joined diagnostics.
func Test_Bounded_External_Enum_Keys(t *testing.T) {
	integer_key := cli_boundary_text(cli.EXTERNAL_KEY_SIZE_MAXIMUM, 'I')
	string_key := cli_boundary_text(cli.EXTERNAL_KEY_SIZE_MAXIMUM, 'S')
	program := external_program(nil, []cli.Secret{
		{
			Key: cli.Secret_Key(integer_key), Paths: []string{"/" + integer_key},
			Value: int(0), Enum: []int{3}, Required: true,
		},
		{
			Key: cli.Secret_Key(string_key), Paths: []string{"/" + string_key},
			Value: "", Enum: []string{"allowed"}, Required: true,
		},
	})
	var parser cli.Parser
	cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
		Arguments: []string{"external"}, Loop: cli_secret_boundary_loop(),
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("full enum key parser did not complete")
	}
	if result.Error == nil {
		t.Fatal("full enum keys accepted values outside their sets")
	}
}

// Test_Bounded_Secret_Conversion_Bytes keeps byte parsing at empty, interior, and full bounds.
func Test_Bounded_Secret_Conversion_Bytes(t *testing.T) {
	maximum := cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'x')
	program := external_program(nil, []cli.Secret{
		{Key: "EMPTY", Paths: []string{"/secrets/EMPTY"}, Value: "",
			Enum: []string{""}, Allow_Empty: true},
		{Key: "TWO", Paths: []string{"/secrets/TWO"}, Value: "",
			Enum: []string{"xx"}},
		{Key: "TEXT_MAX", Paths: []string{"/secrets/TEXT_MAX"}, Value: "",
			Enum: []string{maximum}},
		{Key: "BOOL_EMPTY", Paths: []string{"/secrets/BOOL_EMPTY"}, Value: false,
			Allow_Empty: true},
		{Key: "BOOL_TWO", Paths: []string{"/secrets/BOOL_TWO"}, Value: false},
		{Key: "BOOL_MAX", Paths: []string{"/secrets/BOOL_MAX"}, Value: false},
		{Key: "NUMBER_TEXT", Paths: []string{"/secrets/NUMBER_TEXT"}, Value: int(0)},
	})
	var parser cli.Parser
	cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
		Arguments: []string{"external"}, Loop: cli_secret_boundary_loop(),
	})
	if _, complete := cli.Parser_Done(&parser); !complete {
		t.Fatal("bounded secret conversion did not complete")
	}
}

// Test_Secret_Integer_Edges proves direct byte parsing reaches every machine witness.
func Test_Secret_Integer_Edges(t *testing.T) {
	paths := [...]string{
		"/secrets/INTEGER_MIN", "/secrets/INTEGER_MAX",
		"/secrets/INTEGER_NEGATIVE_ONE", "/secrets/INTEGER_ONE",
		"/secrets/INTEGER_TWO",
	}
	secrets := make([]cli.Secret, len(paths))
	for index, file_path := range paths {
		secrets[index] = cli.Secret{
			Key:   cli.Secret_Key(path.Base(file_path)),
			Paths: []string{file_path}, Value: int(0), Required: true,
		}
	}
	program := external_program(nil, secrets)
	var parser cli.Parser
	cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
		Arguments: []string{"external"}, Loop: cli_secret_boundary_loop(),
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("integer edge secrets did not complete")
	}
	if result.Error != nil {
		t.Fatalf("integer edge secrets: %v", result.Error)
	}
	for _, secret := range result.Command.Secrets {
		cli.Secret_Integer(secret)
	}
}

type cli_secret_boundary_state struct {
	Path string
}

func cli_secret_boundary_loop() (loop nbio.IO) {
	state := &cli_secret_boundary_state{}
	loop.Storage.State = unsafe.Pointer(state)
	loop.Storage.Status_Procedure = cli_secret_boundary_status
	loop.Storage.Open_At_Procedure = cli_secret_boundary_open
	loop.Storage.Read_Procedure = cli_secret_boundary_read
	loop.Close_Procedure = cli_secret_boundary_close
	return loop
}

func cli_secret_boundary_status(
	_ unsafe.Pointer, file_path string,
) (status nbio.File_Status, operation_err error) {
	status.Exists = !text_has_prefix(file_path, "/missing/")
	status.Size = int64(len("private-non-number"))
	switch file_path {
	case "/secrets/BOUNDARY":
		status.Size = cli.SECRET_BYTES_MAX
	case "/secrets/BOOL_MAX":
		status.Size = cli.SECRET_BYTES_MAX
	case "/secrets/NUMBER":
		status.Size = cli.SECRET_BYTES_MAX
	case "/secrets/NUMBER_TEXT", "/secrets/TEXT_MAX":
		status.Size = strings.TEXT_SIZE_MAXIMUM
	case "/secrets/BOUNDARY_ENUM":
		status.Size = cli.SECRET_BYTES_MAX
	case "/secrets/EMPTY", "/secrets/BOOL_EMPTY":
		status.Size = 0
	case "/secrets/TWO", "/secrets/BOOL_TWO":
		status.Size = int64(len("xx"))
	case "/secrets/INTEGER_MIN":
		status.Size = int64(len("-9223372036854775808"))
	case "/secrets/INTEGER_MAX":
		status.Size = int64(len("9223372036854775807"))
	case "/secrets/INTEGER_NEGATIVE_ONE":
		status.Size = int64(len("-1"))
	case "/secrets/INTEGER_ONE", "/secrets/INTEGER_TWO":
		status.Size = int64(len("1"))
	case "/secrets/OVERFLOW":
		status.Size = cli.SECRET_BYTES_MAX + 1
	case "/secrets/NEGATIVE":
		status.Size = 0
	case "/secrets/READ_OVERFLOW":
		status.Size = cli.SECRET_BYTES_MAX
	case "/secrets/DIRECTORY":
		status.Mode = nbio.FILE_MODE_DIRECTORY
	}
	if len(file_path) == strings.TEXT_SIZE_MAXIMUM {
		status.Size = int64(len("4"))
	}
	return status, nil
}

func cli_secret_boundary_open(
	state_pointer unsafe.Pointer, completion *nbio.Completion, _ nbio.File,
	file_path string, _ nbio.Open_At_Options, callback nbio.Callback,
) {
	state := (*cli_secret_boundary_state)(state_pointer)
	state.Path = file_path
	completion.Data = 1
	completion.Error = nil
	callback(completion)
}

func cli_secret_boundary_read(
	state_pointer unsafe.Pointer, completion *nbio.Completion, _ nbio.File,
	buffer []byte, _ int64, _ time.Duration, callback nbio.Callback,
) {
	state := (*cli_secret_boundary_state)(state_pointer)
	if state.Path == "/secrets/NEGATIVE" {
		completion.Data = cli.SECRET_READ_COUNT_MINIMUM
		completion.Error = nil
		callback(completion)
		return
	}
	if state.Path == "/secrets/READ_OVERFLOW" {
		completion.Data = cli.SECRET_READ_COUNT_MAXIMUM
		completion.Error = nil
		callback(completion)
		return
	}
	integer_text := ""
	switch state.Path {
	case "/secrets/INTEGER_MIN":
		integer_text = "-9223372036854775808"
	case "/secrets/INTEGER_MAX":
		integer_text = "9223372036854775807"
	case "/secrets/INTEGER_NEGATIVE_ONE":
		integer_text = "-1"
	case "/secrets/INTEGER_ONE":
		integer_text = "1"
	case "/secrets/INTEGER_TWO":
		integer_text = "2"
	}
	if integer_text != "" {
		completion.Data = copy(buffer, integer_text)
		completion.Error = nil
		callback(completion)
		return
	}
	maximum := false
	switch state.Path {
	case "/secrets/BOUNDARY", "/secrets/BOOL_MAX", "/secrets/NUMBER",
		"/secrets/NUMBER_TEXT", "/secrets/TEXT_MAX", "/secrets/BOUNDARY_ENUM":
		maximum = true
	}
	if maximum {
		for index := range buffer[:cli_secret_boundary_size(state.Path)] {
			buffer[index] = 'x'
		}
		completion.Data = cli_secret_boundary_size(state.Path)
	} else if state.Path == "/secrets/EMPTY" {
		completion.Data = 0
	} else if state.Path == "/secrets/BOOL_EMPTY" {
		completion.Data = 0
	} else if state.Path == "/secrets/TWO" {
		completion.Data = copy(buffer, "xx")
	} else if state.Path == "/secrets/BOOL_TWO" {
		completion.Data = copy(buffer, "xx")
	} else if len(state.Path) == strings.TEXT_SIZE_MAXIMUM {
		completion.Data = copy(buffer, "4")
	} else {
		completion.Data = copy(buffer, "private-non-number")
	}
	completion.Error = nil
	callback(completion)
}

func cli_secret_boundary_size(file_path string) (size int) {
	if file_path == "/secrets/NUMBER_TEXT" {
		return strings.TEXT_SIZE_MAXIMUM
	}
	if file_path == "/secrets/TEXT_MAX" {
		return strings.TEXT_SIZE_MAXIMUM
	}
	return cli.SECRET_BYTES_MAX
}

func cli_secret_boundary_close(
	_ unsafe.Pointer, completion *nbio.Completion, _ nbio.File,
	callback nbio.Callback,
) {
	completion.Error = nil
	callback(completion)
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
	var help_parser cli.Parser
	cli_test_program_parse(&program, &help_parser, cli.Program_Parse_Input{
		Arguments: []string{"external", "-help"},
	})
	help, help_done := cli.Parser_Done(&help_parser)
	if !help_done {
		t.Fatal("help did not complete synchronously")
	}
	if !errors.Is(help.Error, cli.Help_Requested) {
		t.Fatalf("help did not complete synchronously: %+v", help)
	}
	var invalid_parser cli.Parser
	cli_test_program_parse(&program, &invalid_parser, cli.Program_Parse_Input{
		Arguments: []string{"external"},
	})
	invalid, invalid_done := cli.Parser_Done(&invalid_parser)
	if !invalid_done {
		t.Fatal("the CLI error did not complete synchronously")
	}
	if invalid.Error == nil {
		t.Fatal("the CLI error did not complete synchronously")
	}
}

// Test_External_Help_And_Deprecation verifies public defaults and paths in help, omission rules,
// and source-triggered warnings without secret values.
func Test_External_Help_And_Deprecation(t *testing.T) {
	loop, driver := cli_sim_loop(2)
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
	var parser cli.Parser
	cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
		Arguments:   []string{"external"},
		Environment: []string{"OLD_ENV=value", "EMPTY_OLD="},
		Loop:        loop,
	})
	result := drive_parser(t, driver, &parser)
	var warning_output cli.Output
	cli.Print_Deprecations(&warning_output, result.Command)
	warnings := string(cli.Output_Bytes(&warning_output))
	if !text_contains(warnings, "OLD_ENV") {
		t.Fatalf("the environment warning is absent: %q", warnings)
	}
	if !text_contains(warnings, "OLD_SECRET") {
		t.Fatalf("external deprecation warnings are absent: %q", warnings)
	}
	if text_contains(warnings, "EMPTY_OLD") {
		t.Fatalf("an absent value added a deprecation warning: %q", warnings)
	}
	if text_contains(warnings, "do-not-print") {
		t.Fatal("a warning disclosed secret content")
	}
	if text_contains(text, "do-not-print") {
		t.Fatal("help or a warning disclosed secret content")
	}
}

// Verifies the public external declarations and all help omission rules.
func assert_external_help(t *testing.T, program cli.Program) (text string) {
	t.Helper()
	help := output_buffer{}
	cli.Print_Help(output_cli(&help), program)
	text = help.String()
	for _, expected := range []string{
		"Environment Variables:", "PUBLIC", "default", "Secrets:", "/secrets/TOKEN",
	} {
		if !text_contains(text, expected) {
			t.Fatalf("help does not contain %q:\n%s", expected, text)
		}
	}
	hidden_declarations := []string{
		"HIDDEN_ENV", "OLD_ENV", "EMPTY_OLD", "HIDDEN_SECRET", "OLD_SECRET",
	}
	for _, hidden := range hidden_declarations {
		if text_contains(text, hidden) {
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
	loop nbio.IO,
	driver nbio.Driver,
	files []secret_file,
) {
	t.Helper()
	for _, source := range files {
		make_err := cli_sim_make_directory(t, loop, driver, path.Dir(source.Path))
		if make_err != nil {
			if !errors.Is(make_err, nbio.Path_Exists) {
				t.Fatalf("make parent directory: %v", make_err)
			}
		}
		options := nbio.Open_At_Options{
			Access: nbio.OPEN_WRITE_ONLY, Create: true, Truncate: true,
			Permissions: 0o600,
		}
		file, create_err := cli_sim_open(t, loop, driver, source.Path, options)
		if create_err != nil {
			t.Fatalf("create secret file: %v", create_err)
		}
		completed := false
		var completion nbio.Completion
		nbio.Storage_Write(loop.Storage, &completion, file, []byte(source.Content), 0,
			CLI_SIM_DEADLINE, func(completed_write *nbio.Completion) {
				if completed_write.Error != nil {
					t.Errorf("write secret file: %v", completed_write.Error)
				}
				if completed_write.Data != len(source.Content) {
					t.Errorf(
						"write secret file count = %d, want %d",
						completed_write.Data,
						len(source.Content),
					)
				}
				completed = true
			})
		drive_sim_operation(t, driver, func() (finished bool) { return completed })
		completed = false
		nbio.IO_Close(loop, &completion, file, func(completed_close *nbio.Completion) {
			if completed_close.Error != nil {
				t.Errorf("close secret file: %v", completed_close.Error)
			}
			completed = true
		})
		drive_sim_operation(t, driver, func() (finished bool) { return completed })
	}
}

// The test root drives the simulator until one public operation retires.
func drive_sim_operation(
	t *testing.T,
	driver nbio.Driver,
	done func() (finished bool),
) {
	t.Helper()
	completed, drive_err := nbio.Driver_Run_Until(driver, CLI_SIM_DEADLINE, done)
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
	driver nbio.Driver,
	parser *cli.Parser,
) (result cli.Parse_Result) {
	t.Helper()
	drive_sim_operation(t, driver, func() (finished bool) {
		_, done := cli.Parser_Done(parser)
		return bool(done)
	})
	result, _ = cli.Parser_Done(parser)
	return result
}

const CLI_SIM_OPERATION_CAPACITY = 8
const CLI_SIM_COMPLETION_CAPACITY = CLI_SIM_OPERATION_CAPACITY * 2
const CLI_SIM_NODE_CAPACITY = 32
const CLI_SIM_DESCRIPTOR_CAPACITY = 16
const CLI_SIM_EVENT_CAPACITY = 1
const CLI_SIM_DEADLINE = time.MICROSECOND

// CLI reads no clock, so one view slot satisfies the simulator's bound.
const CLI_SIM_CLOCK_CAPACITY = 1

// CLI test owns the loop because library only submits secret I/O.
func cli_sim_loop(seed uint64) (loop nbio.IO, driver nbio.Driver) {
	var state nbio.Sim
	queue := [CLI_SIM_COMPLETION_CAPACITY]*nbio.Completion{}
	events := [CLI_SIM_EVENT_CAPACITY]nbio.Virtual_Event{}
	clocks := [CLI_SIM_CLOCK_CAPACITY]nbio.Sim_Clock{}
	nodes := [CLI_SIM_NODE_CAPACITY]nbio.Sim_Node{}
	descriptors := [CLI_SIM_DESCRIPTOR_CAPACITY]nbio.Sim_Descriptor{}
	operations := [CLI_SIM_OPERATION_CAPACITY]nbio.Sim_Operation{}
	return nbio.New_Simulated_IO(&state, seed, time.NANOSECOND, nbio.Sim_Memory{
		Nodes:       nodes[:],
		Descriptors: descriptors[:],
		Operations:  operations[:],
		Queue:       queue[:],
		Events:      events[:],
		Clocks:      clocks[:],
	})
}

func cli_sim_open(
	t *testing.T, loop nbio.IO, driver nbio.Driver, file_path string,
	options nbio.Open_At_Options,
) (file nbio.File, operation_err error) {
	t.Helper()
	done := false
	var completion nbio.Completion
	nbio.Storage_Open_At(loop.Storage, &completion, nbio.DIRECTORY_CURRENT, file_path, options,
		func(completed *nbio.Completion) {
			file = nbio.File(completed.Data)
			operation_err = completed.Error
			done = true
		})
	drive_sim_operation(t, driver, func() (finished bool) { return done })
	return file, operation_err
}

func cli_sim_make_directory(
	t *testing.T, loop nbio.IO, driver nbio.Driver, directory_path string,
) (operation_err error) {
	t.Helper()
	done := false
	var completion nbio.Completion
	nbio.Storage_Mkdir_At(loop.Storage, &completion, nbio.DIRECTORY_CURRENT, directory_path,
		0o700, func(completed *nbio.Completion) {
			operation_err = completed.Error
			done = true
		})
	drive_sim_operation(t, driver, func() (finished bool) { return done })
	return operation_err
}

// Asserts New rejects the enum-specific malformations and the reserved -help label.
func assert_new_enum_validation(t *testing.T) {
	// An enum flag's default must be one of its permitted values.
	assert_panics(t, "enum flag default outside its set", func() {
		cli.New_Single(cli.New_Single_Input{
			Label: "prog",
			Flags: []cli.Option{
				cli.New_String_Enum_Flag(cli.New_String_Enum_Flag_Input{
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
				cli.New_String_Enum_Argument(cli.New_String_Enum_Argument_Input{
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
	testify.Panics(t, func() {
		cli.New_Single(cli.New_Single_Input{
			Label: "tool",
			Flags: []cli.Option{{Label: "help", Value: false}},
		})
	}, "user option named help")
	testify.Panics(t, func() {
		cli.New_Single(cli.New_Single_Input{
			Label: "tool",
			Flags: []cli.Option{{Label: "h", Value: false}},
		})
	}, "user option named h")
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
	got := cli.Option_Strings(cli.Get_Option(command.Arguments, cli.Option_Label(label)))
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

// Test_Bounded_Aggregates keeps maximum admitted collections observable through public
// operations. Hidden declarations keep rendering inside its separate fixed text bound.
func Test_Bounded_Aggregates(t *testing.T) {
	fixture := cli_maximum_aggregate_fixture()
	cli_bounded_aggregate_render(t, &fixture)
	cli_bounded_aggregate_parse(t, &fixture)
	cli_bounded_aggregate_lookup(t, &fixture)
}

type cli_maximum_aggregate struct {
	Program               cli.Program
	Options               []cli.Option
	Environment           []cli.Environment_Variable
	Secrets               []cli.Secret
	Command               cli.Command
	Warning_Count_Maximum int
	Commands              []cli.Command
}

func cli_maximum_aggregate_fixture() (fixture cli_maximum_aggregate) {
	maximum_text := cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'a')
	maximum_options := make([]cli.Option, slices.COUNT_MAXIMUM)
	for index := range maximum_options {
		maximum_options[index] = cli.Option{
			Label: "x", Value: false, State: cli.Option_State{{Hidden: true}},
		}
	}
	maximum_options[0].State[0].Deprecated = "d"
	maximum_environment := make([]cli.Environment_Variable, slices.COUNT_MAXIMUM)
	for index := range maximum_environment {
		maximum_environment[index] = cli.Environment_Variable{
			Key: "X", Value: "", Hidden: true,
		}
	}
	maximum_secrets := make([]cli.Secret, slices.COUNT_MAXIMUM)
	for index := range maximum_secrets {
		maximum_secrets[index] = cli.Secret{
			Key: "X", Paths: []string{"/X"}, Value: "", Hidden: true,
		}
	}
	maximum_commands := make([]cli.Command, slices.COUNT_MAXIMUM)
	for index := range maximum_commands {
		maximum_commands[index].Hidden = true
	}
	maximum_commands[0] = cli.Command{Label: "c"}
	maximum_warning_count := strings.TEXT_SIZE_MAXIMUM / len("warning: \n")
	maximum_warnings := make([]cli.Warning, maximum_warning_count)
	maximum_command := cli.Command{
		Label:                cli.Label(maximum_text),
		Description:          cli.Description(maximum_text),
		Arguments:            cli.Arguments(maximum_options),
		Flags:                cli.Flags(maximum_options),
		Hidden:               true,
		Deprecated:           cli.Deprecation(maximum_text),
		Deprecation_Warnings: maximum_warnings,
		Environment:          maximum_environment,
		Secrets:              maximum_secrets,
	}
	fixture.Program = cli.Program{
		Label:       "p",
		Description: "d",
		Selection: cli.Program_Selection{{
			Commands:        maximum_commands,
			Single_Commands: cli.Single_Commands{maximum_command},
			Global_Flags:    maximum_options,
			Help_Flags: cli.Help_Flags{
				{State: cli.Option_State{{Hidden: true}}},
				{State: cli.Option_State{{Hidden: true}}},
			},
		}},
		Environment_Variables: maximum_environment,
		Secrets:               maximum_secrets,
	}
	fixture.Options = maximum_options
	fixture.Environment = maximum_environment
	fixture.Secrets = maximum_secrets
	fixture.Command = maximum_command
	fixture.Warning_Count_Maximum = maximum_warning_count
	fixture.Commands = maximum_commands
	return fixture
}

func cli_bounded_aggregate_render(t *testing.T, fixture *cli_maximum_aggregate) {
	t.Helper()
	var output cli.Output
	cli.Print_Help(&output, fixture.Program)
	cli.Output_Reset(&output)
	cli.Print_Requested_Help(&output, fixture.Program, cli.Command{})
	cli.Output_Reset(&output)
	cli.Print_Command(&output, fixture.Program, cli.Command{Label: "p"})
	cli_complete(fixture.Program, []string{"p", ""})
	base_command := fixture.Commands[0]
	fixture.Commands[0] = cli.Command{
		Label: "c",
		Flags: []cli.Option{{
			Label: "e", Value: "v", Enum: []string{"v"},
			State: cli.Option_State{{Is_Flag: true}},
		}},
	}
	cli_complete(fixture.Program, []string{"p", "c", "-"})
	cli_complete(fixture.Program, []string{"p", "c", "-e="})
	fixture.Commands[0] = base_command
	cli.Output_Reset(&output)
	if !handle_completion(
		fixture.Program, []string{"p", "__complete", "p", ""}, &output,
	) {
		t.Fatal("maximum aggregate completion was not handled")
	}
	if _, script_err := completion_script(
		fixture.Program, "bash",
	); script_err != nil {
		t.Fatalf("maximum aggregate completion script: %v", script_err)
	}

	single := fixture.Program
	single.Selection[0].Mode[0] = cli.PROGRAM_MODE_SINGLE
	cli_complete(single, []string{"p", ""})
	cli.Output_Reset(&output)
	cli.Print_Deprecations(&output, fixture.Command)
}

func cli_bounded_aggregate_parse(t *testing.T, fixture *cli_maximum_aggregate) {
	t.Helper()
	global_flags := make([]cli.Option, slices.COUNT_MAXIMUM)
	copy(global_flags, fixture.Options)
	var parser cli.Parser
	assert_panics(t, "secret declarations without an I/O loop", func() {
		cli.Program_Parse(&fixture.Program, &parser, cli.Program_Parse_Input{
			Arguments:         []string{"p", "c", "-x"},
			Command_Arguments: make([]cli.Option, slices.COUNT_MAXIMUM),
			Command_Flags:     make([]cli.Option, slices.COUNT_MAXIMUM),
			Global_Flags:      global_flags,
			Filled:            make([]bool, slices.COUNT_MAXIMUM),
			Positionals:       make([]cli.Indexed_Token, slices.COUNT_MAXIMUM),
			Slice_Named:       make([]cli.Indexed_Token, slices.COUNT_MAXIMUM),
			String_Values:     make([]string, slices.COUNT_MAXIMUM),
			Integer_Values:    make([]int, slices.COUNT_MAXIMUM),
			Failures:          make([]error, cli.FAILURE_COUNT_MAXIMUM),
		})
	})
	cli.Parser_Done(&parser)

	parser = cli.Parser{
		Publication: cli.Publication{
			Result:               cli.Parse_Result{Command: fixture.Command},
			Completion:           cli.Parser_Completion{true},
			Environment_Errors:   make([]error, slices.COUNT_MAXIMUM),
			Secret_Errors:        make([]cli.Secret_Failure, slices.COUNT_MAXIMUM),
			Environment_Warnings: make([]cli.Warning, fixture.Warning_Count_Maximum),
			Secret_Warnings:      make([]cli.Warning, slices.COUNT_MAXIMUM),
			Secret_Count:         slices.COUNT_MAXIMUM,
			Failure:              cli.Failure{},
		},
		Workspace: cli.Workspace{
			Command_Arguments: make([]cli.Option, slices.COUNT_MAXIMUM),
			Command_Flags:     make([]cli.Option, slices.COUNT_MAXIMUM),
			Filled:            make([]bool, slices.COUNT_MAXIMUM),
			Positionals:       make([]cli.Indexed_Token, slices.COUNT_MAXIMUM),
			Slice_Named:       make([]cli.Indexed_Token, slices.COUNT_MAXIMUM),
			String_Values:     make([]string, slices.COUNT_MAXIMUM),
			Integer_Values:    make([]int, slices.COUNT_MAXIMUM),
		},
	}
	cli.Program_Parse(&fixture.Program, &parser, cli.Program_Parse_Input{
		Arguments:         []string{"p", "c", "-help"},
		Command_Arguments: make([]cli.Option, slices.COUNT_MAXIMUM),
		Command_Flags:     make([]cli.Option, slices.COUNT_MAXIMUM),
		Global_Flags:      global_flags,
		Filled:            make([]bool, slices.COUNT_MAXIMUM),
		Positionals:       make([]cli.Indexed_Token, slices.COUNT_MAXIMUM),
		Slice_Named:       make([]cli.Indexed_Token, slices.COUNT_MAXIMUM),
		String_Values:     make([]string, slices.COUNT_MAXIMUM),
		Integer_Values:    make([]int, slices.COUNT_MAXIMUM),
		Failures:          make([]error, cli.FAILURE_COUNT_MAXIMUM),
	})
	cli.Parser_Done(&parser)
}

func cli_bounded_aggregate_lookup(t *testing.T, fixture *cli_maximum_aggregate) {
	t.Helper()
	option := cli.Get_Option(fixture.Options, "x")
	if option.Label != "x" {
		t.Fatal("maximum option lookup changed the first declaration")
	}
	variable := cli.Get_Environment(fixture.Environment, "X")
	if variable.Key != "X" {
		t.Fatal("maximum environment lookup changed the first declaration")
	}
	secret := cli.Get_Secret(fixture.Secrets, "X")
	if secret.Key != "X" {
		t.Fatal("maximum secret lookup changed the first declaration")
	}
}

// Test_Bounded_Text keeps every scalar edge legal through constructors and runtime lookup.
func Test_Bounded_Text(t *testing.T) {
	text_sizes := [...]int{1, 2, strings.TEXT_SIZE_MAXIMUM}
	for _, size := range text_sizes {
		cli_bounded_text(t, size)
	}
	cli_bounded_integer_text()
}

func cli_bounded_text(t *testing.T, size int) {
	t.Helper()
	fixture := cli_bounded_text_fixture(size)
	argument, variable, secret := cli_bounded_text_constructors(fixture)
	cli_bounded_text_runtime(t, fixture, argument)
	cli.New(cli.New_Input{
		Label:                 cli.Label(fixture.Program_Label),
		Description:           cli.Description(fixture.Description),
		Commands:              []cli.Command{{Label: "x"}},
		Environment_Variables: []cli.Environment_Variable{variable},
		Secrets:               []cli.Secret{secret},
	})
	cli.New_Multicall(cli.New_Multicall_Input{
		Label:       cli.Label(fixture.Program_Label),
		Description: cli.Description(fixture.Description),
		Commands:    []cli.Command{{Label: "x"}},
	})
}

type cli_text_fixture struct {
	Program_Label  string
	Argument_Label string
	Flag_Label     string
	Description    string
	External_Key   string
	Secret_Key     string
	String_Value   string
}

func cli_bounded_text_fixture(size int) (fixture cli_text_fixture) {
	option_label_size := size
	if option_label_size == strings.TEXT_SIZE_MAXIMUM {
		option_label_size -= len("-")
	}
	external_key_size := size
	if external_key_size > cli.EXTERNAL_KEY_SIZE_MAXIMUM {
		external_key_size = cli.EXTERNAL_KEY_SIZE_MAXIMUM
	}
	secret_key_size := size
	if secret_key_size == strings.TEXT_SIZE_MAXIMUM {
		secret_key_size -= len("/")
	}
	return cli_text_fixture{
		Program_Label:  cli_boundary_text(size, 'p'),
		Argument_Label: cli_boundary_text(option_label_size, 'a'),
		Flag_Label:     cli_boundary_text(option_label_size, 'b'),
		Description:    cli_boundary_text(size, 'd'),
		External_Key:   cli_boundary_text(external_key_size, 'X'),
		Secret_Key:     cli_boundary_text(secret_key_size, 'Y'),
		String_Value:   cli_boundary_text(size, 'v'),
	}
}

func cli_bounded_text_constructors(
	fixture cli_text_fixture,
) (argument cli.Option, variable cli.Environment_Variable, secret cli.Secret) {
	argument = cli.New_Argument[string](cli.New_Argument_Input{
		Label:       cli.Option_Label(fixture.Argument_Label),
		Description: cli.Description(fixture.Description),
	})
	cli.New_Variadic[string](cli.New_Variadic_Input{
		Label:       cli.Option_Label(fixture.Argument_Label),
		Description: cli.Description(fixture.Description),
	})
	cli.New_Flag(cli.New_Flag_Input[string]{
		Label: cli.Option_Label(fixture.Flag_Label), Value: fixture.String_Value,
		Description: cli.Description(fixture.Description), Hidden: true,
		Deprecated: cli.Deprecation(fixture.Description),
	})
	cli.New_String_Enum_Flag(cli.New_String_Enum_Flag_Input{
		Label:       cli.Option_Label(fixture.Flag_Label),
		Value:       cli.String_Enum_Default(fixture.String_Value),
		Enum:        []string{fixture.String_Value},
		Description: cli.Description(fixture.Description), Hidden: true,
		Deprecated: cli.Deprecation(fixture.Description),
	})
	cli.New_String_Enum_Argument(cli.New_String_Enum_Argument_Input{
		Label:       cli.Option_Label(fixture.Argument_Label),
		Enum:        []string{fixture.String_Value},
		Description: cli.Description(fixture.Description),
	})
	variable = cli.New_String_Enum_Environment_Variable(
		cli.New_String_Enum_Environment_Variable_Input{
			Key:         cli.External_Key(fixture.External_Key),
			Description: cli.Description(fixture.Description),
			Value:       cli.String_Enum_Default(fixture.String_Value),
			Enum:        []string{fixture.String_Value}, Hidden: true,
			Deprecated: cli.Deprecation(fixture.Description),
		},
	)
	secret = cli.New_String_Enum_Secret(cli.New_String_Enum_Secret_Input{
		Paths:       []string{"/" + fixture.Secret_Key},
		Description: cli.Description(fixture.Description),
		Enum:        []string{fixture.String_Value}, Hidden: true,
		Deprecated: cli.Deprecation(fixture.Description),
	})
	return argument, variable, secret
}

func cli_bounded_text_runtime(
	t *testing.T, fixture cli_text_fixture, argument cli.Option,
) {
	t.Helper()
	program := cli.New_Single(cli.New_Single_Input{
		Label:       cli.Label(fixture.Program_Label),
		Description: cli.Description(fixture.Description),
		Arguments:   []cli.Option{argument},
	})
	var parser cli.Parser
	cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
		Arguments:         []string{fixture.Program_Label, "value"},
		Command_Arguments: make([]cli.Option, 1), Filled: make([]bool, 1),
		Positionals: make([]cli.Indexed_Token, 1),
		Slice_Named: make([]cli.Indexed_Token, 1),
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("bounded text parser did not complete")
	}
	if result.Error != nil {
		t.Fatalf("bounded text parse: %v", result.Error)
	}
	if cli.Get_Option(
		result.Command.Arguments, cli.Option_Label(fixture.Argument_Label),
	).Label == "" {
		t.Fatal("bounded text option lookup failed")
	}
	cli_complete(program, []string{fixture.Program_Label, ""})
	var output cli.Output
	cli.Print_Command(&output, program, cli.Command{Label: "x"})
	cli.Output_Reset(&output)
	cli.Print_Requested_Help(&output, program, cli.Command{Label: "x"})
	cli.Output_Reset(&output)
	handle_completion(program, []string{
		fixture.Program_Label, "__complete", fixture.Program_Label, "",
	}, &output)
}

func cli_bounded_integer_text() {
	integer_edges := [...]int{bits.INTEGER_MINIMUM, bits.INTEGER_MAXIMUM}
	for _, integer := range integer_edges {
		cli.New_Flag(cli.New_Flag_Input[int]{Label: "n", Value: integer})
		cli.New_Integer_Enum_Flag(cli.New_Integer_Enum_Flag_Input{
			Label: "n", Value: cli.Integer_Enum_Default(integer), Enum: []int{integer},
		})
		cli.New_Integer_Enum_Argument(cli.New_Integer_Enum_Argument_Input{
			Label: "n", Enum: []int{integer},
		})
		cli.New_Integer_Enum_Environment_Variable(
			cli.New_Integer_Enum_Environment_Variable_Input{
				Key: "NUMBER", Value: cli.Integer_Enum_Default(integer),
				Enum: []int{integer},
			},
		)
		cli.New_Integer_Enum_Secret(cli.New_Integer_Enum_Secret_Input{
			Paths: []string{"/NUMBER"}, Enum: []int{integer},
		})
	}
}

// Test_Bounded_Secret_State drives declaration, parent-storage, and fallback edges while
// every path remains absent. No file payload obscures state-count coverage.
func Test_Bounded_Secret_State(t *testing.T) {
	cli_bounded_secret_declarations(t)
	cli_bounded_secret_paths(t)
	cli_bounded_secret_aggregate(t)
}

func cli_bounded_secret_declarations(t *testing.T) {
	t.Helper()
	cli_parse_absent_secrets(t, cli_raw_single_program(
		"p", cli.Command{Label: "p"}, nil,
		[]cli.Secret{
			{
				Key: "", Value: "", Description: "d", Deprecated: "d",
				Allow_Empty: true,
			},
			{
				Key: "XX", Paths: []string{"/missing/XX"}, Value: "",
				Description: "dd", Deprecated: "dd", Allow_Empty: true,
			},
		},
	), 2)
	counts := [...]int{1, 2, slices.COUNT_MAXIMUM}
	for _, count := range counts {
		secrets := make([]cli.Secret, count)
		for index := range secrets {
			secrets[index] = cli.Secret{
				Key: "X", Paths: []string{"/missing/X"}, Value: "",
			}
		}
		environment := make([]cli.Environment_Variable, count)
		for index := range environment {
			environment[index] = cli.Environment_Variable{Key: "E", Value: ""}
		}
		flags := make([]cli.Option, count)
		for index := range flags {
			flags[index] = cli.Option{
				Label: "f", Value: false, State: cli.Option_State{{Is_Flag: true}},
			}
		}
		cli_parse_absent_secrets(t, cli_raw_single_program(
			"p", cli.Command{Label: "p", Flags: flags}, environment, secrets,
		), count)

		present := make([]cli.Secret, count)
		for index := range present {
			present[index] = cli.Secret{
				Key: "X", Paths: []string{"/X"}, Value: "",
			}
		}
		present_command := cli_boundary_command(count)
		present_command.Flags = flags
		cli_parse_present_secrets(t, cli_raw_single_program(
			"p", present_command, environment, present,
		), count)
	}
}

func cli_bounded_secret_paths(t *testing.T) {
	t.Helper()
	path_counts := [...]int{2, slices.COUNT_MAXIMUM}
	for _, count := range path_counts {
		paths := make([]string, count)
		for index := range paths {
			paths[index] = "/missing/X"
		}
		cli_parse_absent_secrets(t, cli_raw_single_program(
			"p", cli.Command{Label: "p"}, nil,
			[]cli.Secret{{Key: "X", Paths: paths, Value: ""}},
		), count)
	}
	maximum_paths := make([]string, slices.COUNT_MAXIMUM)
	for index := range maximum_paths[:len(maximum_paths)-1] {
		maximum_paths[index] = "/missing/X"
	}
	maximum_paths[len(maximum_paths)-1] = "/X"
	cli_parse_present_secrets(t, cli_raw_single_program(
		"p", cli.Command{Label: "p"}, nil,
		[]cli.Secret{{Key: "X", Paths: maximum_paths, Value: ""}},
	), slices.COUNT_MAXIMUM)
}

func cli_bounded_secret_aggregate(t *testing.T) {
	t.Helper()
	arguments := make([]cli.Option, slices.COUNT_MAXIMUM)
	for index := range arguments {
		arguments[index] = cli.Option{Label: "a", Value: []string{}}
	}
	cli_parse_present_secrets(t, cli_raw_single_program(
		"p", cli.Command{Label: "p", Arguments: arguments}, nil,
		[]cli.Secret{{
			Key: "X", Paths: []string{"/X"}, Value: "",
		}},
	), slices.COUNT_MAXIMUM)

	maximum_text := cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'p')
	maximum_key := cli_boundary_text(strings.TEXT_SIZE_MAXIMUM-len("/"), 'X')
	cli_parse_present_secrets(t, cli_raw_single_program(
		cli.Program_Label(maximum_text), cli.Command{
			Label: cli.Label(maximum_text), Description: cli.Description(maximum_text),
			Deprecated: cli.Deprecation(maximum_text),
		}, nil, []cli.Secret{{
			Key: cli.Secret_Key(maximum_key), Paths: []string{"/" + maximum_key},
			Value: "", Description: cli.Description(maximum_text),
			Deprecated: cli.Deprecation(maximum_text),
		}},
	), 1)
}

func cli_raw_single_program(
	label cli.Program_Label, command cli.Command,
	environment []cli.Environment_Variable, secrets []cli.Secret,
) (program cli.Program) {
	return cli.Program{
		Label: label,
		Selection: cli.Program_Selection{{
			Single_Commands: cli.Single_Commands{command},
			Mode:            cli.Program_Mode{cli.PROGRAM_MODE_SINGLE},
		}},
		Environment_Variables: environment, Secrets: secrets,
	}
}

func cli_parse_absent_secrets(t *testing.T, program cli.Program, storage_count int) {
	t.Helper()
	var parser cli.Parser
	cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
		Arguments:         []string{"p"},
		Loop:              cli_secret_boundary_loop(),
		Command_Arguments: make([]cli.Option, storage_count),
		Command_Flags:     make([]cli.Option, storage_count),
		Global_Flags:      make([]cli.Option, storage_count),
		Filled:            make([]bool, storage_count),
		Positionals:       make([]cli.Indexed_Token, storage_count),
		Slice_Named:       make([]cli.Indexed_Token, storage_count),
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("synchronous absent secrets did not publish")
	}
	if result.Error != nil {
		t.Fatalf("optional absent secret: %v", result.Error)
	}
}

func cli_parse_present_secrets(t *testing.T, program cli.Program, storage_count int) {
	t.Helper()
	var parser cli.Parser
	cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
		Arguments:         []string{string(program.Label)},
		Loop:              cli_secret_boundary_loop(),
		Command_Arguments: make([]cli.Option, storage_count),
		Command_Flags:     make([]cli.Option, storage_count),
		Global_Flags:      make([]cli.Option, storage_count),
		Filled:            make([]bool, storage_count),
		Positionals:       make([]cli.Indexed_Token, storage_count),
		Slice_Named:       make([]cli.Indexed_Token, storage_count),
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("synchronous present secrets did not publish")
	}
	if result.Error != nil {
		t.Fatalf("present secret: %v", result.Error)
	}
}

// Test_Bounded_Command_State keeps each declared command collection edge observable.
func Test_Bounded_Command_State(t *testing.T) {
	counts := [...]int{1, 2, slices.COUNT_MAXIMUM}
	for _, count := range counts {
		cli_bounded_argument_command(t, count)
		cli_bounded_flag_command(t, count)
	}
}

// Test_Bounded_Constructors keeps constructor-only metadata and borrowed enum edges visible.
func Test_Bounded_Constructors(t *testing.T) {
	counts := [...]int{1, 2, slices.COUNT_MAXIMUM}
	for _, count := range counts {
		cli_bounded_option_constructors(t, count)
		cli_bounded_external_constructors(t, count)
	}
	cli_bounded_empty_option_constructors()
	cli_bounded_empty_external_constructors()
}

// Test_Bounded_Enum_Display_Name keeps the maximum positional identity observable.
func Test_Bounded_Enum_Display_Name(t *testing.T) {
	label := cli_boundary_text(cli.OPTION_LABEL_SIZE_MAXIMUM, 'a')
	program := cli_raw_single_program("p", cli.Command{
		Label: "p",
		Arguments: []cli.Option{cli.New_String_Enum_Argument(
			cli.New_String_Enum_Argument_Input{
				Label: cli.Option_Label(label), Enum: []string{"x"},
			},
		)},
	}, nil, nil)
	command, err := parse_program(&program, []string{"p", "x"})
	if err != nil {
		t.Fatalf("maximum enum display name: %v", err)
	}
	if cli.Option_String(cli.Get_Option(command.Arguments, cli.Option_Label(label))) != "x" {
		t.Fatal("maximum enum display name discarded its value")
	}
}

func cli_bounded_empty_option_constructors() {
	cli.New_Argument[string](cli.New_Argument_Input{})
	cli.New_Variadic[string](cli.New_Variadic_Input{})
	cli.New_Flag(cli.New_Flag_Input[string]{})
	cli.New_String_Enum_Argument(cli.New_String_Enum_Argument_Input{})
	cli.New_Integer_Enum_Argument(cli.New_Integer_Enum_Argument_Input{})
	cli.New_String_Enum_Flag(cli.New_String_Enum_Flag_Input{})
	cli.New_Integer_Enum_Flag(cli.New_Integer_Enum_Flag_Input{})
	cli.New_Integer_Enum_Flag(cli.New_Integer_Enum_Flag_Input{Value: -1})
}

func cli_bounded_empty_external_constructors() {
	cli.New_Environment_Variable(cli.New_Environment_Variable_Input[string]{})
	cli.New_String_Enum_Environment_Variable(
		cli.New_String_Enum_Environment_Variable_Input{},
	)
	cli.New_Integer_Enum_Environment_Variable(
		cli.New_Integer_Enum_Environment_Variable_Input{},
	)
	cli.New_Integer_Enum_Environment_Variable(
		cli.New_Integer_Enum_Environment_Variable_Input{Value: -1},
	)
	cli.New_String_Enum_Secret(cli.New_String_Enum_Secret_Input{})
	cli.New_Integer_Enum_Secret(cli.New_Integer_Enum_Secret_Input{})
}

// Test_Bounded_Program_Constructors keeps validation bounds separate from runtime shape.
func Test_Bounded_Program_Constructors(t *testing.T) {
	counts := [...]int{0, 1, 2, slices.COUNT_MAXIMUM}
	for _, count := range counts {
		cli_bounded_multicall_constructor(t, count)
		cli_bounded_single_external_constructor(t, count)
	}
	cli_bounded_single_option_constructor(t)
}

// Test_Bounded_Global_Validation keeps command collision search at every width.
func Test_Bounded_Global_Validation(t *testing.T) {
	counts := [...]int{0, 1, 2, slices.COUNT_MAXIMUM}
	for _, count := range counts {
		global_flags := make([]cli.Option, count)
		for index := range global_flags {
			global_flags[index] = cli.Option{
				Label: cli.Option_Label("g" + cli_decimal_text(index)),
				Value: false,
				State: cli.Option_State{{Is_Flag: true}},
			}
		}
		program := cli.New(cli.New_Input{
			Label: "p", Global_Flags: global_flags,
			Commands: []cli.Command{{
				Label: "c", Flags: []cli.Option{{Label: "local", Value: false}},
			}},
		})
		got_count := len(program.Selection[0].Global_Flags)
		if got_count != count {
			t.Fatalf("global flag count = %d, want %d", got_count, count)
		}
	}
}

// Test_Bounded_Global_Parse keeps present collection widths and warning counts observable.
func Test_Bounded_Global_Parse(t *testing.T) {
	for _, count := range []int{1, 2, slices.COUNT_MAXIMUM} {
		flags := cli_global_flags(count)
		tokens := []string{"p", "c", "-" + string(flags[len(flags)-1].Label)}
		cli_parse_global_flags(t, flags, tokens)
	}
	flags := cli_global_flags(cli.DEPRECATION_WARNING_COUNT_MAXIMUM)
	tokens := make([]string, len(flags)+2)
	tokens[0] = "p"
	tokens[1] = "c"
	for index := range flags {
		tokens[index+2] = "-" + string(flags[index].Label)
	}
	result := cli_parse_global_flags(t, flags, tokens)
	if len(result.Command.Deprecation_Warnings) != cli.DEPRECATION_WARNING_COUNT_MAXIMUM {
		t.Fatalf("deprecation warning count = %d", len(result.Command.Deprecation_Warnings))
	}
}

func cli_global_flags(count int) (flags []cli.Option) {
	flags = make([]cli.Option, count)
	for index := range flags {
		flags[index] = cli.Option{
			Label: cli.Option_Label("g" + cli_decimal_text(index)), Value: false,
			State: cli.Option_State{{Is_Flag: true, Deprecated: "d"}},
		}
	}
	return flags
}

func cli_parse_global_flags(
	t *testing.T, flags []cli.Option, tokens []string,
) (result cli.Parse_Result) {
	t.Helper()
	program := cli.New(cli.New_Input{
		Label: "p", Global_Flags: flags,
		Commands: []cli.Command{{Label: "c"}},
	})
	var parser cli.Parser
	cli.Program_Parse(&program, &parser, cli.Program_Parse_Input{
		Arguments: tokens, Global_Flags: make([]cli.Option, len(flags)),
		Filled:               make([]bool, len(flags)),
		Deprecation_Warnings: make([]cli.Warning, cli.DEPRECATION_WARNING_COUNT_MAXIMUM),
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("global flag parser did not complete")
	}
	if result.Error != nil {
		t.Fatalf("global flag parser: %v", result.Error)
	}
	return result
}

// Test_Bounded_External_Lookup keeps declaration metadata observable at lookup bounds.
func Test_Bounded_External_Lookup(t *testing.T) {
	counts := [...]int{0, 1, 2, slices.COUNT_MAXIMUM}
	for _, count := range counts {
		cli_bounded_environment_lookup(t, count)
		cli_bounded_secret_lookup(t, count)
	}
}

// Test_Bounded_External_Accessors keeps each typed scalar view's complete metadata domain.
func Test_Bounded_External_Accessors(t *testing.T) {
	text_sizes := [...]int{0, 1, 2, strings.TEXT_SIZE_MAXIMUM}
	byte_sizes := [...]int{0, 1, 2, cli.SECRET_BYTES_MAX}
	var secret_bytes [cli.SECRET_BYTES_MAX]byte
	secret_paths := make([]string, slices.COUNT_MAXIMUM)
	for case_index, text_size := range text_sizes {
		key_size := text_size
		if key_size > cli.EXTERNAL_KEY_SIZE_MAXIMUM {
			key_size = cli.EXTERNAL_KEY_SIZE_MAXIMUM
		}
		secret_key_size := text_size
		if secret_key_size > cli.SECRET_KEY_SIZE_MAXIMUM {
			secret_key_size = cli.SECRET_KEY_SIZE_MAXIMUM
		}
		text := cli_boundary_text(text_size, 'd')
		key := cli_boundary_text(key_size, 'K')
		secret_key := cli_boundary_text(secret_key_size, 'S')
		set := case_index == 1
		environment_state := cli.External_State{{
			String: cli.External_Value_Text(text), Integer: cli.Integer(text_size),
			Boolean: cli.Boolean(set), Parsed: cli.Parsed(set),
		}}
		environment := cli.Environment_Variable{
			Key: cli.External_Key(key), Description: cli.Description(text),
			Type: cli.External_Type_State{
				cli.Scalar_Option_Type(cli.OPTION_TYPE_STRING),
			},
			Required: cli.Required(set), Allow_Empty: cli.Allow_Empty(set),
			Hidden: cli.Hidden(set), Deprecated: cli.Deprecation(text),
			State: environment_state,
		}
		cli.Environment_String(environment)
		environment.Type[0] = cli.Scalar_Option_Type(cli.OPTION_TYPE_INTEGER)
		cli.Environment_Integer(environment)
		environment.Type[0] = cli.Scalar_Option_Type(cli.OPTION_TYPE_BOOLEAN)
		cli.Environment_Boolean(environment)

		secret_state := cli.External_State{{
			Bytes:   cli.Secret_Value_Bytes(secret_bytes[:byte_sizes[case_index]]),
			Integer: cli.Integer(text_size), Boolean: cli.Boolean(set),
			Parsed: cli.Parsed(set),
		}}
		secret := cli.Secret{
			Key: cli.Secret_Key(secret_key), Paths: secret_paths[:text_size],
			Description: cli.Description(text), Required: cli.Required(set),
			Allow_Empty: cli.Allow_Empty(set), Hidden: cli.Hidden(set),
			Deprecated: cli.Deprecation(text), State: secret_state,
		}
		secret.Value = ""
		cli.Secret_String(secret)
		secret.Value = int(0)
		cli.Secret_Integer(secret)
		secret.Value = false
		cli.Secret_Boolean(secret)
	}
	maximum_string := cli.Environment_Variable{
		Key: "E",
		Type: cli.External_Type_State{
			cli.Scalar_Option_Type(cli.OPTION_TYPE_STRING),
		},
		State: cli.External_State{{
			String: cli.External_Value_Text(string(secret_bytes[:])),
		}},
	}
	cli.Environment_String(maximum_string)
	cli_bounded_external_integers()
}

func cli_bounded_external_integers() {
	edges := [...]int{bits.INTEGER_MINIMUM, bits.INTEGER_MAXIMUM, -1}
	for _, edge := range edges {
		state := cli.External_State{{Integer: cli.Integer(edge)}}
		environment := cli.Environment_Variable{
			Key: "E", Type: cli.External_Type_State{
				cli.Scalar_Option_Type(cli.OPTION_TYPE_INTEGER),
			}, State: state,
		}
		cli.Environment_Integer(environment)
		secret := cli.Secret{Key: "S", Value: int(0), State: state}
		cli.Secret_Integer(secret)
	}
}

// Test_Bounded_External_Metadata keeps omission, type, and failure metadata observable.
func Test_Bounded_External_Metadata(t *testing.T) {
	for _, size := range []int{1, 2, strings.TEXT_SIZE_MAXIMUM} {
		deprecated := cli_boundary_text(size, 'd')
		program := cli.New_Single(cli.New_Single_Input{
			Label: "p",
			Environment_Variables: []cli.Environment_Variable{{
				Key: "E", Value: "", Deprecated: cli.Deprecation(deprecated),
			}},
		})
		var output cli.Output
		cli.Print_Help(&output, program)
	}

	maximum := cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'd')
	cli_parse_boundary_environment(t, []cli.Environment_Variable{{
		Key: "E", Value: "", Required: true,
		Description: cli.Description(maximum), Deprecated: cli.Deprecation(maximum),
	}}, nil)
	cli_bounded_external_types(t)
	cli_bounded_external_conversion_key(t)
	paths := make([]string, slices.COUNT_MAXIMUM)
	for index := range paths {
		paths[index] = "/X"
	}
	cli.New_Single(cli.New_Single_Input{
		Label: "p", Secrets: []cli.Secret{{Key: "X", Paths: paths, Value: ""}},
	})
	_, _, environment, _ := cli_constructor_declarations(slices.COUNT_MAXIMUM)
	cli_parse_boundary_environment(t, environment, nil)
	cli_parse_boundary_environment(t, []cli.Environment_Variable{
		{Key: "E", Value: "", Deprecated: "d"},
		{Key: "F", Value: "", Deprecated: "d"},
	}, []string{"E=v", "F=v"})
}

func cli_bounded_external_types(t *testing.T) {
	t.Helper()
	integer_program := cli.New_Single(cli.New_Single_Input{
		Label:                 "p",
		Environment_Variables: []cli.Environment_Variable{{Key: "E", Value: int(0)}},
	})
	var integer_output cli.Output
	cli.Print_Help(&integer_output, integer_program)

	member_size := strings.TEXT_SIZE_MAXIMUM - len("string()")
	member := cli_boundary_text(member_size, 'm')
	maximum_program := cli.New_Single(cli.New_Single_Input{
		Label: "p",
		Environment_Variables: []cli.Environment_Variable{{
			Key: "E", Value: member, Enum: []string{member},
		}},
	})
	var maximum_output cli.Output
	assert_panics(t, "maximum external type exceeds help output", func() {
		cli.Print_Help(&maximum_output, maximum_program)
	})
}

func cli_bounded_external_conversion_key(t *testing.T) {
	t.Helper()
	key := cli_boundary_text(cli.EXTERNAL_KEY_SIZE_MAXIMUM, 'E')
	program := cli.New_Single(cli.New_Single_Input{
		Label: "p",
		Environment_Variables: []cli.Environment_Variable{{
			Key: cli.External_Key(key), Value: int(0), Allow_Empty: true,
		}},
	})
	var parser cli.Parser
	cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
		Arguments: []string{"p"}, Environment: []string{key + "="},
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("maximum conversion key did not complete")
	}
	if result.Error == nil {
		t.Fatal("maximum conversion key accepted empty integer")
	}
	integer_enum_key := cli_boundary_text(cli.EXTERNAL_KEY_SIZE_MAXIMUM-len("0"), 'I')
	integer_enum := cli.New_Single(cli.New_Single_Input{
		Label: "p",
		Environment_Variables: []cli.Environment_Variable{{
			Key: cli.External_Key(integer_enum_key), Value: int(0), Enum: []int{0, 1},
		}},
	})
	cli_test_program_parse(&integer_enum, &parser, cli.Program_Parse_Input{
		Arguments: []string{"p"}, Environment: []string{integer_enum_key + "=2"},
	})
	string_enum := cli.New_Single(cli.New_Single_Input{
		Label: "p",
		Environment_Variables: []cli.Environment_Variable{{
			Key: cli.External_Key(key), Value: "x", Enum: []string{"x"},
			Allow_Empty: true,
		}},
	})
	cli_test_program_parse(&string_enum, &parser, cli.Program_Parse_Input{
		Arguments: []string{"p"}, Environment: []string{key + "="},
	})
	for _, value_size := range [...]int{1, 2, cli.ENVIRONMENT_VALUE_SIZE_MAXIMUM} {
		value := cli_boundary_text(value_size, 'x')
		invalid := cli.New_Single(cli.New_Single_Input{
			Label: "p",
			Environment_Variables: []cli.Environment_Variable{{
				Key: "E", Value: false,
			}},
		})
		cli_test_program_parse(&invalid, &parser, cli.Program_Parse_Input{
			Arguments: []string{"p"}, Environment: []string{"E=" + value},
		})
	}
	two_key := cli.New_Single(cli.New_Single_Input{
		Label:                 "p",
		Environment_Variables: []cli.Environment_Variable{{Key: "EE", Value: false}},
	})
	cli_test_program_parse(&two_key, &parser, cli.Program_Parse_Input{
		Arguments: []string{"p"}, Environment: []string{"EE=x"},
	})
	maximum_enum_value := cli_boundary_text(cli.ENVIRONMENT_VALUE_SIZE_MAXIMUM, 'x')
	maximum_enum := cli.New_Single(cli.New_Single_Input{
		Label: "p",
		Environment_Variables: []cli.Environment_Variable{{
			Key: "E", Value: "allowed", Enum: []string{"allowed"},
		}},
	})
	cli_test_program_parse(&maximum_enum, &parser, cli.Program_Parse_Input{
		Arguments: []string{"p"}, Environment: []string{"E=" + maximum_enum_value},
	})
}

// Test_Bounded_Environment_State separates declaration width from source multiplicity.
func Test_Bounded_Environment_State(t *testing.T) {
	text_sizes := [...]int{1, 2, cli.EXTERNAL_KEY_SIZE_MAXIMUM}
	for _, size := range text_sizes {
		text := cli_boundary_text(size, 'E')
		variable := cli.Environment_Variable{
			Key: cli.External_Key(text), Description: cli.Description(text), Value: "",
			Allow_Empty: true, Deprecated: cli.Deprecation(text),
		}
		cli_parse_boundary_environment(t, []cli.Environment_Variable{variable}, nil)
		if size > 0 {
			value_size := size
			available := strings.TEXT_SIZE_MAXIMUM - size - len("=")
			if value_size > available {
				value_size = available
			}
			value := cli_boundary_text(value_size, 'v')
			resolved := variable
			resolved.Deprecated = ""
			cli_parse_boundary_environment(t, []cli.Environment_Variable{resolved},
				[]string{text + "=" + value})
		}
	}
	cli_bounded_environment_sources(t)
	cli_bounded_environment_warnings(t)
}

// Test_Bounded_Environment_Sources keeps empty and maximum source maps observable.
func Test_Bounded_Environment_Sources(t *testing.T) {
	empty_program := cli_raw_single_program(
		"p", cli.Command{Label: "p"},
		[]cli.Environment_Variable{{Key: "", Value: ""}}, nil,
	)
	var empty_parser cli.Parser
	empty_sources := cli.Environment_Sources{"stale-a": {}, "stale-b": {}}
	cli_test_program_parse(&empty_program, &empty_parser, cli.Program_Parse_Input{
		Arguments: []string{"p"}, Environment: []string{"=x", "=x"},
		Environment_Sources: empty_sources,
	})
	empty_result, empty_complete := cli.Parser_Done(&empty_parser)
	if !empty_complete {
		t.Fatal("empty environment key parse did not complete")
	}
	if empty_result.Error == nil {
		t.Fatal("empty environment key did not publish its source failure")
	}

	variables := make([]cli.Environment_Variable, slices.COUNT_MAXIMUM)
	raw := make([]string, slices.COUNT_MAXIMUM)
	sources := make(cli.Environment_Sources, slices.COUNT_MAXIMUM)
	for index := range variables {
		key := "E" + cli_decimal_text(index)
		variables[index] = cli.Environment_Variable{
			Key: cli.External_Key(key), Value: "",
		}
		raw[index] = key + "=x"
		sources[cli.External_Key(key)] = cli.Environment_Source{}
	}
	program := cli_raw_single_program("p", cli.Command{Label: "p"}, variables, nil)
	var parser cli.Parser
	cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
		Arguments: []string{"p"}, Environment: raw, Environment_Sources: sources,
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("maximum environment source map did not complete")
	}
	if result.Error != nil {
		t.Fatalf("maximum environment source map: %v", result.Error)
	}
}

// Test_Bounded_External_Key_Code_Points keeps hostile Unicode inside rune reality.
func Test_Bounded_External_Key_Code_Points(t *testing.T) {
	characters := [...]rune{
		rune(cli.CHARACTER_MINIMUM), 1, 2, rune(cli.CHARACTER_MAXIMUM),
	}
	for _, character := range characters {
		key := "A" + string(character)
		assert_panics(t, "non-ASCII external key", func() {
			cli.New_Single(cli.New_Single_Input{
				Label: "p",
				Environment_Variables: []cli.Environment_Variable{{
					Key: cli.External_Key(key), Value: "",
				}},
			})
		})
	}
}

func cli_bounded_environment_sources(t *testing.T) {
	t.Helper()
	value := cli_boundary_text(strings.TEXT_SIZE_MAXIMUM-len("E="), 'v')
	variable := cli.Environment_Variable{
		Key: "E", Value: "", Required: true, Deprecated: "d",
	}
	cli_parse_boundary_environment(t, []cli.Environment_Variable{variable}, []string{
		"E=" + value,
	})
	raw := make([]string, slices.COUNT_MAXIMUM)
	for index := range raw {
		raw[index] = "E=" + value
	}
	cli_parse_boundary_environment(t, []cli.Environment_Variable{variable}, raw)
}

func cli_bounded_environment_warnings(t *testing.T) {
	t.Helper()
	maximum_count := strings.TEXT_SIZE_MAXIMUM / len("warning: \n")
	variables := make([]cli.Environment_Variable, maximum_count)
	for index := range variables {
		variables[index] = cli.Environment_Variable{
			Key: "E", Value: "", Allow_Empty: true, Deprecated: "d",
		}
	}
	cli_parse_boundary_environment(t, variables, []string{"E=v"})

	deprecation_size := strings.TEXT_SIZE_MAXIMUM - len(cli.ENVIRONMENT_WARNING_PREFIX) -
		len("E") - len(cli.ENVIRONMENT_WARNING_SUFFIX)
	cli_parse_boundary_environment(t, []cli.Environment_Variable{{
		Key: "E", Value: "", Deprecated: cli.Deprecation(
			cli_boundary_text(deprecation_size, 'd'),
		),
	}}, []string{"E=v"})
	cli_parse_boundary_environment(t, []cli.Environment_Variable{{
		Key: "E", Value: "", Deprecated: cli.Deprecation(
			cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'd'),
		),
	}}, nil)
}

func cli_parse_boundary_environment(
	t *testing.T, environment []cli.Environment_Variable, raw []string,
) {
	t.Helper()
	program := cli_raw_single_program(
		"p", cli.Command{Label: "p"}, environment, nil,
	)
	var parser cli.Parser
	cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
		Arguments: []string{"p"}, Environment: raw,
	})
	_, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("bounded environment parse did not publish")
	}
}

func cli_bounded_environment_lookup(t *testing.T, count int) {
	t.Helper()
	text_size := count
	if text_size > cli.EXTERNAL_KEY_SIZE_MAXIMUM {
		text_size = cli.EXTERNAL_KEY_SIZE_MAXIMUM
	}
	text := cli_boundary_text(text_size, 'E')
	metadata := text
	if count == slices.COUNT_MAXIMUM {
		metadata = cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'd')
	}
	variable := cli.Environment_Variable{
		Key: cli.External_Key(text), Description: cli.Description(metadata), Value: text,
		Required: true, Allow_Empty: true, Hidden: true,
		Deprecated: cli.Deprecation(metadata),
	}
	variables := make([]cli.Environment_Variable, count)
	for index := range variables {
		variables[index] = variable
	}
	if count == 0 {
		assert_panics(t, "empty environment lookup", func() {
			cli.Get_Environment(variables, "")
		})
		return
	}
	if cli.Get_Environment(variables, cli.External_Key(text)).Key != variable.Key {
		t.Fatal("bounded environment lookup changed the declaration")
	}
}

func cli_bounded_secret_lookup(t *testing.T, count int) {
	t.Helper()
	text_size := count
	if text_size > strings.TEXT_SIZE_MAXIMUM-len("/") {
		text_size = strings.TEXT_SIZE_MAXIMUM - len("/")
	}
	text := cli_boundary_text(text_size, 'S')
	metadata := text
	if count == slices.COUNT_MAXIMUM {
		metadata = cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'd')
	}
	paths := make([]string, count)
	for index := range paths {
		paths[index] = "/" + text
	}
	secret := cli.Secret{
		Key: cli.Secret_Key(text), Paths: paths, Description: cli.Description(metadata),
		Value: text, Required: true, Allow_Empty: true, Hidden: true,
		Deprecated: cli.Deprecation(metadata),
	}
	secrets := make([]cli.Secret, count)
	for index := range secrets {
		secrets[index] = secret
	}
	if count == 0 {
		assert_panics(t, "empty secret lookup", func() {
			cli.Get_Secret(secrets, "")
		})
		return
	}
	if cli.Get_Secret(secrets, cli.External_Key(text)).Key != secret.Key {
		t.Fatal("bounded secret lookup changed the declaration")
	}
}

func cli_bounded_multicall_constructor(t *testing.T, count int) {
	t.Helper()
	label_size := count
	if label_size > strings.TEXT_SIZE_MAXIMUM {
		label_size = strings.TEXT_SIZE_MAXIMUM
	}
	text := cli_boundary_text(label_size, 'p')
	commands, flags, environment, secrets := cli_constructor_declarations(count)
	input := cli.New_Multicall_Input{
		Label: cli.Label(text), Description: cli.Description(text),
		Commands: commands, Global_Flags: flags,
		Environment_Variables: environment, Secrets: secrets,
	}
	if count == 0 {
		assert_panics(t, "a multicall program needs one command", func() {
			cli.New_Multicall(input)
		})
		return
	}
	program := cli.New_Multicall(input)
	if len(program.Selection[0].Commands) != count {
		t.Fatalf(
			"bounded multicall command count = %d, want %d",
			len(program.Selection[0].Commands), count,
		)
	}
}

func cli_bounded_single_external_constructor(t *testing.T, count int) {
	t.Helper()
	_, _, environment, secrets := cli_constructor_declarations(count)
	program := cli.New_Single(cli.New_Single_Input{
		Label: "p", Environment_Variables: environment, Secrets: secrets,
	})
	if len(program.Environment_Variables) != count {
		t.Fatalf("bounded environment count = %d, want %d",
			len(program.Environment_Variables), count)
	}
	if len(program.Secrets) != count {
		t.Fatalf("bounded secret count = %d, want %d",
			len(program.Secrets), count)
	}
}

func cli_bounded_single_option_constructor(t *testing.T) {
	t.Helper()
	options := make([]cli.Option, slices.COUNT_MAXIMUM)
	for index := range options {
		options[index] = cli.Option{Label: "duplicate", Value: ""}
	}
	text := cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'p')
	assert_panics(t, "duplicate maximum option declarations", func() {
		cli.New_Single(cli.New_Single_Input{
			Label: cli.Label(text), Description: cli.Description(text),
			Arguments: options, Flags: options,
		})
	})
}

func cli_constructor_declarations(count int) (
	commands []cli.Command,
	flags []cli.Option,
	environment []cli.Environment_Variable,
	secrets []cli.Secret,
) {
	commands = make([]cli.Command, count)
	flags = make([]cli.Option, count)
	environment = make([]cli.Environment_Variable, count)
	secrets = make([]cli.Secret, count)
	for index := 0; index < count; index++ {
		suffix := cli_decimal_text(index)
		commands[index] = cli.Command{Label: cli.Label("c" + suffix), Hidden: true}
		flags[index] = cli.Option{
			Label: cli.Option_Label("g" + suffix), Value: false,
			State: cli.Option_State{{Is_Flag: true, Hidden: true}},
		}
		environment[index] = cli.Environment_Variable{
			Key: cli.External_Key("E" + suffix), Value: "", Hidden: true,
		}
		secrets[index] = cli.Secret{
			Key: cli.Secret_Key("S" + suffix), Paths: []string{"/S" + suffix},
			Value: "", Hidden: true,
		}
	}
	return commands, flags, environment, secrets
}

func cli_decimal_text(value int) (text string) {
	var storage [strconv.DECIMAL_TEXT_SIZE_MAXIMUM]byte
	count := strconv.Format_Decimal_Into(storage[:], strconv.Machine_Integer(value))
	return string(storage[:count])
}

// Test_Bounded_Program_Entrypoints prevents a narrow fixture from hiding public bounds.
func Test_Bounded_Program_Entrypoints(t *testing.T) {
	counts := [...]int{0, 1, 2, slices.COUNT_MAXIMUM}
	for _, count := range counts {
		program, command := cli_boundary_program(count)
		cli_bounded_completion_entrypoints(t, program, command, count)
		cli_bounded_render_entrypoints(t, program, command, count)
	}
}

// Test_Bounded_Output_States keeps every caller-owned size visible at public gates.
func Test_Bounded_Output_States(t *testing.T) {
	sizes := [...]int{0, 1, 2, strings.TEXT_SIZE_MAXIMUM}
	program := cli_raw_single_program("p", cli.Command{Label: "p"}, nil, nil)
	command := program.Selection[0].Single_Commands[0]
	for _, size := range sizes {
		var output cli.Output
		cli_fill_output(&output, size)
		written, write_err := output.Write(nil)
		if write_err != nil {
			t.Fatalf("empty output write = %d, %v", written, write_err)
		}
		if written != 0 {
			t.Fatalf("empty output write = %d, %v", written, write_err)
		}
		cli.Output_Write_Text(&output, "")
		if len(cli.Output_Bytes(&output)) != size {
			t.Fatalf(
				"output size changed: got %d, want %d",
				len(cli.Output_Bytes(&output)), size,
			)
		}
		cli.Print_Deprecations(&output, cli.Command{})

		var requested_output cli.Output
		cli_fill_output(&requested_output, size)
		var command_output cli.Output
		cli_fill_output(&command_output, size)
		if size == strings.TEXT_SIZE_MAXIMUM {
			assert_panics(t, "requested help exceeds full output", func() {
				cli.Print_Requested_Help(&requested_output, program, command)
			})
			assert_panics(t, "command help exceeds full output", func() {
				cli.Print_Command(&command_output, program, command)
			})
			continue
		}
		cli.Print_Requested_Help(&requested_output, program, command)
		cli.Print_Command(&command_output, program, command)
	}
	cli_bounded_hidden_flag_section(t, program)
	cli_bounded_visible_flag_section(t, program)
	cli_bounded_empty_flag_row(t, program)
	cli_bounded_option_signatures(t, program)
}

// Test_Bounded_Parser_Done_State keeps terminal-only parent edges observable.
func Test_Bounded_Parser_Done_State(t *testing.T) {
	parser := cli.Parser{
		Publication: cli.Publication{
			Completion:           cli.Parser_Completion{true},
			Environment_Warnings: make([]cli.Warning, 2),
			Secret_Count:         cli.Declaration_Count(slices.COUNT_MAXIMUM),
		},
	}
	_, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("completed boundary parser reported pending")
	}
}

// Test_Bounded_Process_Arguments keeps empty input and every post-program token visible.
func Test_Bounded_Process_Arguments(t *testing.T) {
	empty_program := cli.New_Single(cli.New_Single_Input{Label: "p"})
	cli_bounded_slice_index(t)
	assert_panics(t, "empty process arguments", func() {
		var empty_parser cli.Parser
		cli.Program_Parse(&empty_program, &empty_parser, cli.Program_Parse_Input{})
	})
	cli_parse_zero_option_command(t, empty_program)
	assert_panics(t, "zero positional storage", func() {
		var parser cli.Parser
		cli.Program_Parse(&empty_program, &parser, cli.Program_Parse_Input{
			Arguments: []string{"p", "unexpected"},
		})
	})

	variadic := cli.New_Variadic[string](cli.New_Variadic_Input{Label: "value"})
	program := cli.New_Single(cli.New_Single_Input{
		Label: "p", Arguments: []cli.Option{variadic},
	})
	positionals := make([]string, slices.COUNT_MAXIMUM)
	positionals[0] = "p"
	for index := 1; index < len(positionals); index++ {
		positionals[index] = "x"
	}
	cli_parse_variadic_boundary(t, program, positionals)

	named := make([]string, slices.COUNT_MAXIMUM)
	named[0] = "p"
	for index := 1; index < len(named); index++ {
		named[index] = "-value=x"
	}
	cli_parse_variadic_boundary(t, program, named)

	help_arguments := make([]string, slices.COUNT_MAXIMUM)
	help_arguments[0] = "p"
	help_arguments[1] = "-help"
	var help_parser cli.Parser
	cli.Program_Parse(&empty_program, &help_parser, cli.Program_Parse_Input{
		Arguments: help_arguments,
	})
	help_result, help_complete := cli.Parser_Done(&help_parser)
	if !help_complete {
		t.Fatal("maximum help arguments did not complete")
	}
	if !errors.Is(help_result.Error, cli.Help_Requested) {
		t.Fatalf("maximum help arguments: %v", help_result.Error)
	}
}

func cli_bounded_slice_index(t *testing.T) {
	t.Helper()
	program := cli.New_Single(cli.New_Single_Input{
		Label: "p", Arguments: []cli.Option{
			cli.New_Argument[string](cli.New_Argument_Input{Label: "a"}),
			cli.New_Argument[string](cli.New_Argument_Input{Label: "b"}),
			cli.New_Variadic[string](cli.New_Variadic_Input{Label: "values"}),
		},
	})
	var parser cli.Parser
	cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
		Arguments: []string{"p"},
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("three-argument command did not complete")
	}
	if result.Error == nil {
		t.Fatal("three-argument command accepted missing scalars")
	}
}

// Test_Bounded_Positional_Values keeps empty, two-byte, and full scalar text observable.
func Test_Bounded_Positional_Values(t *testing.T) {
	values := [...]string{
		"", "xx", cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'x'),
	}
	for _, value := range values {
		argument := cli.New_Argument[string](cli.New_Argument_Input{Label: "value"})
		program := cli.New_Single(cli.New_Single_Input{
			Label: "p", Arguments: []cli.Option{argument},
		})
		var parser cli.Parser
		cli.Program_Parse(&program, &parser, cli.Program_Parse_Input{
			Arguments:         []string{"p", value},
			Command_Arguments: make([]cli.Option, 1),
			Filled:            make([]bool, 1),
			Positionals:       make([]cli.Indexed_Token, 1),
		})
		result, complete := cli.Parser_Done(&parser)
		if !complete {
			t.Fatal("bounded positional value did not complete")
		}
		if result.Error != nil {
			t.Fatalf("bounded positional value: %v", result.Error)
		}
	}
}

func cli_parse_zero_option_command(t *testing.T, program cli.Program) {
	t.Helper()
	var parser cli.Parser
	cli.Program_Parse(&program, &parser, cli.Program_Parse_Input{
		Arguments: []string{"p"},
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("zero-option command did not complete")
	}
	if result.Error != nil {
		t.Fatalf("zero-option command: %v", result.Error)
	}
}

func cli_parse_variadic_boundary(t *testing.T, program cli.Program, arguments []string) {
	t.Helper()
	var parser cli.Parser
	cli.Program_Parse(&program, &parser, cli.Program_Parse_Input{
		Arguments:         arguments,
		Command_Arguments: make([]cli.Option, 1),
		Filled:            make([]bool, 1),
		Positionals:       make([]cli.Indexed_Token, cli.PARSED_TOKEN_COUNT_MAXIMUM),
		Slice_Named:       make([]cli.Indexed_Token, cli.PARSED_TOKEN_COUNT_MAXIMUM),
		String_Values:     make([]string, cli.PARSED_TOKEN_COUNT_MAXIMUM),
		Integer_Values:    make([]int, cli.PARSED_TOKEN_COUNT_MAXIMUM),
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("maximum variadic command did not complete")
	}
	if result.Error != nil {
		t.Fatalf("maximum variadic command: %v", result.Error)
	}
	values := cli.Option_Strings(cli.Get_Option(result.Command.Arguments, "value"))
	if len(values) != len(arguments)-1 {
		t.Fatalf("variadic value count = %d", len(values))
	}
}

func cli_fill_output(output *cli.Output, size int) {
	cli.Output_Write_Text(output, cli.Value_Text(cli_boundary_text(size, 'x')))
}

func cli_bounded_hidden_flag_section(t *testing.T, program cli.Program) {
	t.Helper()
	flags := make([]cli.Option, slices.COUNT_MAXIMUM)
	for index := range flags {
		flags[index] = cli.Option{
			Label: "f", Value: false,
			State: cli.Option_State{{Is_Flag: true, Hidden: true}},
		}
	}
	command := cli.Command{Label: "p", Flags: flags}
	var output cli.Output
	cli.Print_Command(&output, program, command)
}

func cli_bounded_empty_flag_row(t *testing.T, program cli.Program) {
	t.Helper()
	command := cli.Command{
		Label: "p",
		Flags: []cli.Option{{
			Value: false, State: cli.Option_State{{Is_Flag: true}},
		}},
	}
	var output cli.Output
	cli.Print_Command(&output, program, command)
}

func cli_bounded_option_signatures(t *testing.T, program cli.Program) {
	t.Helper()
	var empty_enum_output cli.Output
	cli.Print_Command(&empty_enum_output, program, cli.Command{
		Label: "p", Arguments: []cli.Option{{
			Label: "a", Value: "", Enum: []string{},
		}},
	})
	var minimum_output cli.Output
	cli.Print_Command(&minimum_output, program, cli.Command{
		Label: "p", Arguments: []cli.Option{{Value: int(0)}},
	})

	maximum_label_size := strings.TEXT_SIZE_MAXIMUM - cli.OPTION_SIGNATURE_SIZE_MINIMUM
	maximum_label := cli_boundary_text(maximum_label_size, 'a')
	var maximum_output cli.Output
	assert_panics(t, "maximum option signature exceeds usage output", func() {
		cli.Print_Command(&maximum_output, program, cli.Command{
			Label: "p", Arguments: []cli.Option{{
				Label: cli.Option_Label(maximum_label), Value: int(0),
			}},
		})
	})

	var two_output cli.Output
	cli.Print_Command(&two_output, program, cli.Command{
		Label: "p", Arguments: []cli.Option{{
			Label: "a", Value: "", Enum: []string{"xx"},
		}},
	})
	maximum_member := cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'm')
	var enum_output cli.Output
	assert_panics(t, "maximum enum signature exceeds usage output", func() {
		cli.Print_Command(&enum_output, program, cli.Command{
			Label: "p", Arguments: []cli.Option{{
				Label: "a", Value: "", Enum: []string{maximum_member},
			}},
		})
	})
}

func cli_bounded_visible_flag_section(t *testing.T, program cli.Program) {
	t.Helper()
	flags := make([]cli.Option, slices.COUNT_MAXIMUM)
	for index := range flags {
		flags[index] = cli.Option{
			Label: "f", Value: false, State: cli.Option_State{{Is_Flag: true}},
		}
	}
	var output cli.Output
	assert_panics(t, "maximum visible flags exceed fixed output", func() {
		cli.Print_Command(&output, program, cli.Command{Label: "p", Flags: flags})
	})
}

// Test_Bounded_Completion_Scope keeps every collection and index result observable.
func Test_Bounded_Completion_Scope(t *testing.T) {
	indices := [...]int{0, 1, 2, slices.FOUND_INDEX_MAXIMUM}
	for _, index := range indices {
		options := make([]cli.Option, index+1)
		options[index] = cli.Option{Label: "a", Value: "", Enum: []string{"member"}}
		cli_expect_completion(t, cli_completion_program(options, nil, nil), "a")
		cli_expect_completion(t, cli_completion_program(nil, options, nil), "a")
		cli_expect_completion(t, cli_completion_program(nil, nil, options), "a")
	}
	labels := [...]string{
		"a", "aa", cli_boundary_text(cli.COMPLETION_OPTION_LABEL_SIZE_MAXIMUM, 'a'),
	}
	for _, label := range labels {
		option := cli.Option{
			Label: cli.Option_Label(label), Value: "", Enum: []string{"member"},
		}
		cli_expect_completion(
			t, cli_completion_program([]cli.Option{option}, nil, nil), label,
		)
	}
	program := cli_completion_program(nil, nil, nil)
	if candidates := cli_complete(program, []string{"p", "-="}); len(candidates) != 0 {
		t.Fatalf("empty completion label returned %v", candidates)
	}
	cli_bounded_completion_candidates(t, 2)
	cli_bounded_completion_candidates(t, slices.COUNT_MAXIMUM)
	two_commands := cli.New(cli.New_Input{
		Label: "p", Commands: []cli.Command{{Label: "a"}, {Label: "b"}},
	})
	cli_complete(two_commands, []string{"p", ""})
	cli_complete(two_commands, []string{
		"p", cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'z'),
	})
}

func cli_bounded_completion_candidates(t *testing.T, count int) {
	t.Helper()
	members := make([]string, count)
	for index := range members {
		members[index] = "member"
	}
	option := cli.Option{Label: "a", Value: "", Enum: members}
	program := cli_completion_program([]cli.Option{option}, nil, nil)
	candidates := cli_complete(program, []string{"p", "-a="})
	if len(candidates) != count {
		t.Fatalf("completion candidate count = %d, want %d", len(candidates), count)
	}
}

// Test_Completion_Caller_Bounds proves caller storage owns every completion result.
func Test_Completion_Caller_Bounds(t *testing.T) {
	var storage [cli.CANDIDATE_COUNT_MAXIMUM]cli.Candidate
	hidden_flag := cli.Option{
		Label: "h", Value: false,
		State: cli.Option_State{{Is_Flag: true, Hidden: true}},
	}
	single := cli_completion_program(nil, []cli.Option{hidden_flag}, nil)
	selector := cli.Program{
		Label: "p",
		Selection: cli.Program_Selection{{
			Commands: []cli.Command{{Label: "c", Hidden: true}},
		}},
	}
	enum := cli_completion_program(
		[]cli.Option{{Value: "", Enum: []string{"x"}}},
		[]cli.Option{{Label: "a", Value: "", Enum: []string{"x"}}}, nil,
	)
	counts := [...]int{0, 1, 2, cli.CANDIDATE_COUNT_MAXIMUM}
	for _, count := range counts {
		cells := storage[:count]
		cli.Complete(selector, []string{"p", ""}, cells)
		var empty_output cli.Output
		cli.Handle_Completion(selector, []string{"p"}, &empty_output, cells)
		cli.Complete(single, []string{
			"p", cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, '-'),
		}, cells)
		cli.Complete(enum, []string{"p", "y"}, cells)
		cli.Complete(enum, []string{"p", "yy"}, cells)
		cli.Complete(enum, []string{"p", "-a=y"}, cells)
		if count > 0 {
			cli.Complete(enum, []string{"p", "-a="}, cells)
		}
		if count > 1 {
			cli.Complete(single, []string{"p", "-"}, cells)
			cli.Complete(single, []string{"p", "--"}, cells)
			var populated_output cli.Output
			cli.Handle_Completion(single, []string{"p"}, &populated_output, cells)
		}
	}
	assert_panics(t, "one completion cell cannot hold two help labels", func() {
		cli.Complete(single, []string{"p", "-"}, storage[:1])
	})
	cli_bounded_all_completion_candidates(t, storage[:])
}

func cli_bounded_all_completion_candidates(
	t *testing.T, storage cli.Candidates,
) {
	t.Helper()
	commands := make([]cli.Command, slices.COUNT_MAXIMUM)
	for index := range commands {
		commands[index] = cli.Command{Label: "c"}
	}
	selector := cli.Program{
		Label: "p", Selection: cli.Program_Selection{{Commands: commands}},
	}
	command_candidates := cli.Complete(selector, []string{"p", ""}, storage)
	if len(command_candidates) != slices.COUNT_MAXIMUM {
		t.Fatalf("command completion candidate count = %d", len(command_candidates))
	}
	arguments := make([]cli.Option, slices.COUNT_MAXIMUM)
	flags := make([]cli.Option, slices.COUNT_MAXIMUM)
	global_flags := make([]cli.Option, slices.COUNT_MAXIMUM)
	for index := range arguments {
		arguments[index] = cli.Option{Label: "a", Value: ""}
		flags[index] = cli.Option{Label: "f", Value: false}
		global_flags[index] = cli.Option{Label: "g", Value: false}
	}
	program := cli.Program{
		Label: "p",
		Selection: cli.Program_Selection{{
			Single_Commands: cli.Single_Commands{{
				Label: "p", Arguments: arguments, Flags: flags,
			}},
			Global_Flags: global_flags,
			Help_Flags: cli.Help_Flags{
				{Label: "h", Value: false}, {Label: "help", Value: false},
			},
			Mode: cli.Program_Mode{cli.PROGRAM_MODE_SINGLE},
		}},
	}
	candidates := cli.Complete(program, []string{"p", "-"}, storage)
	if len(candidates) != cli.CANDIDATE_COUNT_MAXIMUM {
		t.Fatalf("all completion candidate count = %d", len(candidates))
	}
}

// Test_Candidate_Render_Bounds keeps segmented size and output edges observable.
func Test_Candidate_Render_Bounds(t *testing.T) {
	texts := [...]string{
		"", "x", cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'x'),
	}
	for _, expected := range texts {
		candidate := cli.Candidate{}
		candidate.Segments[0] = expected
		if !cli.Candidate_Equal(candidate, cli.Value_Text(expected)) {
			t.Fatalf("candidate did not equal %d-byte text", len(expected))
		}
	}
	maximum := cli.Candidate{}
	for index := range maximum.Segments {
		maximum.Segments[index] = texts[len(texts)-1]
	}
	maximum.State[0].Integer = cli.Integer(bits.INTEGER_MINIMUM)
	maximum.State[0].Has_Integer = true
	cli.Candidate_Equal(maximum, "")
	for _, size := range [...]int{0, 1, 2, strings.TEXT_SIZE_MAXIMUM} {
		var output cli.Output
		cli_fill_output(&output, size)
		cli.Candidate_Write(&output, cli.Candidate{})
	}
}

// Test_Completion_Script_Output_Bounds proves scripts append only to caller output.
func Test_Completion_Script_Output_Bounds(t *testing.T) {
	program := cli_completion_program(nil, nil, nil)
	for _, size := range [...]int{0, 1, 2, strings.TEXT_SIZE_MAXIMUM} {
		var output cli.Output
		cli_fill_output(&output, size)
		if size == strings.TEXT_SIZE_MAXIMUM {
			assert_panics(t, "full completion script output", func() {
				cli.Completion_Script(program, "fish", &output)
			})
			continue
		}
		if err := cli.Completion_Script(program, "fish", &output); err != nil {
			t.Fatalf("%d-byte completion output: %v", size, err)
		}
	}
}

// Test_Closest_Option_Bounds keeps direct no-slice suggestion search observable.
func Test_Closest_Option_Bounds(t *testing.T) {
	for _, count := range [...]int{0, 1, 2, slices.COUNT_MAXIMUM} {
		arguments := make([]cli.Option, count)
		flags := make([]cli.Option, count)
		global_flags := make([]cli.Option, count)
		for index := range arguments {
			arguments[index] = cli.Option{Label: "aaa", Value: ""}
			flags[index] = cli.Option{Label: "aaa", Value: false}
			global_flags[index] = cli.Option{Label: "aaa", Value: false}
		}
		program := cli_completion_program(arguments, nil, nil)
		cli_parse_unknown_option(&program, "aab")
		program = cli_completion_program(nil, flags, nil)
		cli_parse_unknown_option(&program, "aab")
		program = cli_completion_program(nil, nil, global_flags)
		cli_parse_unknown_option(&program, "aab")
	}
	for _, target := range []string{
		"x", "xx", cli_boundary_text(cli.OPTION_LABEL_SIZE_MAXIMUM, 'x'),
	} {
		program := cli_completion_program(
			[]cli.Option{{Label: "aa", Value: ""}}, nil, nil,
		)
		cli_parse_unknown_option(&program, target)
	}
	cli_bounded_command_suggestion()
}

func cli_parse_unknown_option(program *cli.Program, label string) {
	var parser cli.Parser
	cli_test_program_parse(program, &parser, cli.Program_Parse_Input{
		Arguments: []string{"p", "-" + label},
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		panic("unknown option parser did not publish")
	}
	if result.Error == nil {
		panic("unknown option parser accepted its label")
	}
}

func cli_bounded_command_suggestion() {
	maximum := cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'a')
	target := maximum[:len(maximum)-len("a")] + "b"
	program := cli.Program{
		Label: "p",
		Selection: cli.Program_Selection{{
			Commands: []cli.Command{{Label: cli.Label(maximum)}},
		}},
	}
	var parser cli.Parser
	cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
		Arguments: []string{"p", target},
	})
	short := program
	short.Selection[0].Commands[0].Label = "aa"
	cli_test_program_parse(&short, &parser, cli.Program_Parse_Input{
		Arguments: []string{"p", "x"},
	})
}

// Test_Bounded_Positional_Completion keeps cursor and nonempty-word edges observable.
func Test_Bounded_Positional_Completion(t *testing.T) {
	program := cli_completion_program(nil, nil, nil)
	cli_complete(program, []string{"p"})
	cli_complete(program, []string{"p", "a", ""})
	cli_complete(program, []string{"p", "a", "b", ""})
	words := make([]string, slices.COUNT_MAXIMUM)
	words[0] = "p"
	cli_complete(program, words)
	two_arguments := cli.New_Single(cli.New_Single_Input{
		Label: "p", Arguments: []cli.Option{
			cli.New_Argument[string](cli.New_Argument_Input{Label: "a"}),
			cli.New_Argument[string](cli.New_Argument_Input{Label: "b"}),
		},
	})
	cli_complete(two_arguments, []string{"p", "x", ""})

	selected := cli.Program{
		Label: "p",
		Selection: cli.Program_Selection{{
			Commands: []cli.Command{{Label: "c"}},
		}},
	}
	cli_complete(selected, []string{"p", "c", ""})
}

func cli_completion_program(
	arguments []cli.Option, flags []cli.Option, global_flags []cli.Option,
) (program cli.Program) {
	return cli.Program{
		Label: "p",
		Selection: cli.Program_Selection{{
			Single_Commands: cli.Single_Commands{{
				Label: "p", Arguments: arguments, Flags: flags,
			}},
			Global_Flags: global_flags,
			Mode:         cli.Program_Mode{cli.PROGRAM_MODE_SINGLE},
		}},
	}
}

func cli_expect_completion(t *testing.T, program cli.Program, label string) {
	t.Helper()
	candidates := cli_complete(program, []string{"p", "-" + label + "="})
	if len(candidates) == 0 {
		t.Fatalf("%d-byte option label did not complete", len(label))
	}
}

func cli_boundary_program(count int) (program cli.Program, command cli.Command) {
	text_size := count
	if text_size > strings.TEXT_SIZE_MAXIMUM {
		text_size = strings.TEXT_SIZE_MAXIMUM
	}
	text := cli_boundary_text(text_size, 'p')
	option := cli_boundary_option(count, false)
	options := make([]cli.Option, count)
	for index := range options {
		options[index] = option
	}
	command = cli_boundary_command(count)
	command.Arguments = cli.Arguments(options)
	command.Flags = cli.Flags(options)
	commands := make([]cli.Command, count)
	for index := range commands {
		commands[index] = command
	}
	program = cli.Program{
		Label: cli.Program_Label(text), Description: cli.Program_Description(text),
		Selection: cli.Program_Selection{{
			Commands: commands, Single_Commands: cli.Single_Commands{command},
			Global_Flags: options,
			Help_Flags: cli.Help_Flags{
				{State: cli.Option_State{{Hidden: true}}},
				{State: cli.Option_State{{Hidden: true}}},
			},
		}},
		Environment_Variables: make([]cli.Environment_Variable, count),
		Secrets:               make([]cli.Secret, count),
	}
	return program, command
}

func cli_bounded_completion_entrypoints(
	t *testing.T, program cli.Program, command cli.Command, count int,
) {
	t.Helper()
	words := make([]string, count)
	if len(words) > 0 {
		words[0] = string(program.Label)
	}
	if len(words) > 1 {
		words[1] = string(command.Label)
	}
	if len(words) > 2 {
		words[len(words)-1] = "-"
	}
	cli_complete(program, nil)
	cli_complete(program, words)
	if count == 0 {
		assert_panics(t, "selector mode needs one command", func() {
			cli_complete(program, []string{"", "", "-"})
		})
		cli_bounded_completion_script(t, program, count)
		return
	}
	cli_complete(program, []string{
		string(program.Label), string(command.Label), "-",
	})
	var output cli.Output
	cli.Output_Write_Text(&output, cli.Value_Text(cli_boundary_text(count, 'x')))
	handle_completion(program, words, &output)
	cli.Output_Reset(&output)
	handle_completion(program, []string{
		string(program.Label), "__complete", string(program.Label), "",
	}, &output)
	cli_bounded_completion_script(t, program, count)
}

func cli_bounded_completion_script(t *testing.T, program cli.Program, count int) {
	t.Helper()
	shell_size := count
	if shell_size > strings.TEXT_SIZE_MAXIMUM {
		shell_size = strings.TEXT_SIZE_MAXIMUM
	}
	completion_script(program, cli.Shell(cli_boundary_text(shell_size, 's')))
	if count == slices.COUNT_MAXIMUM {
		assert_panics(t, "maximum completion registration exceeds fixed text", func() {
			completion_script(program, "bash")
		})
		return
	}
	if _, script_err := completion_script(program, "bash"); script_err != nil {
		t.Fatalf("bounded completion script: %v", script_err)
	}
}

// Test_Bounded_Completion_Script_Output keeps exact text and target maxima observable.
func Test_Bounded_Completion_Script_Output(t *testing.T) {
	minimum_program := cli_raw_single_program("", cli.Command{}, nil, nil)
	minimum_script, minimum_err := completion_script(minimum_program, "fish")
	if minimum_err != nil {
		t.Fatalf("minimum completion script: %v", minimum_err)
	}
	if len(minimum_script) != cli.SCRIPT_SIZE_MINIMUM {
		t.Fatalf("minimum completion script size = %d", len(minimum_script))
	}

	label := cli_boundary_text(cli.COMPLETION_ZSH_MAXIMUM_LABEL_SIZE, 'p')
	program := cli_raw_single_program(
		cli.Program_Label(label), cli.Command{Label: cli.Label(label)}, nil, nil,
	)
	script, script_err := completion_script(program, "zsh")
	if script_err != nil {
		t.Fatalf("maximum completion script: %v", script_err)
	}
	if len(script) != strings.TEXT_SIZE_MAXIMUM {
		t.Fatalf("completion script size = %d", len(script))
	}

	two_targets := cli.Program{
		Label: "p",
		Selection: cli.Program_Selection{{
			Commands: []cli.Command{{Label: "c"}},
			Mode:     cli.Program_Mode{cli.PROGRAM_MODE_MULTICALL},
		}},
	}
	if _, err := completion_script(two_targets, "fish"); err != nil {
		t.Fatalf("two completion targets: %v", err)
	}

	commands := make([]cli.Command, slices.COUNT_MAXIMUM)
	for index := range commands {
		commands[index] = cli.Command{Label: "c"}
	}
	maximum_targets := cli.Program{
		Label: "p",
		Selection: cli.Program_Selection{{
			Commands: commands, Mode: cli.Program_Mode{cli.PROGRAM_MODE_MULTICALL},
		}},
	}
	assert_panics(t, "maximum completion targets exceed fixed script", func() {
		completion_script(maximum_targets, "fish")
	})
}

func cli_bounded_render_entrypoints(
	t *testing.T, program cli.Program, command cli.Command, count int,
) {
	t.Helper()
	var output cli.Output
	cli.Output_Write_Text(&output, cli.Value_Text(cli_boundary_text(count, 'x')))
	if count == slices.COUNT_MAXIMUM {
		assert_panics(t, "maximum program header exceeds fixed output", func() {
			cli.Print_Help(&output, program)
		})
		cli.Output_Reset(&output)
		assert_panics(t, "maximum command signature exceeds fixed output", func() {
			cli.Print_Command(&output, program, command)
		})
		return
	}
	cli.Print_Help(&output, program)
	cli.Output_Reset(&output)
	cli.Print_Requested_Help(&output, program, cli.Command{})
	cli.Output_Reset(&output)
	cli.Print_Requested_Help(&output, program, command)
	cli.Output_Reset(&output)
	cli.Print_Command(&output, program, command)
}

// Test_Bounded_Flag_Metadata keeps validation and rendering on every text edge.
func Test_Bounded_Flag_Metadata(t *testing.T) {
	profiles := [...]struct{ Label_Size, Description_Size int }{
		{1, 1}, {2, 2}, {cli.OPTION_LABEL_SIZE_MAXIMUM, strings.TEXT_SIZE_MAXIMUM},
	}
	for _, profile := range profiles {
		label := cli_boundary_text(profile.Label_Size, 'f')
		description := cli_boundary_text(profile.Description_Size, 'd')
		flag := cli.New_Flag(cli.New_Flag_Input[bool]{
			Label: cli.Option_Label(label), Description: cli.Description(description),
		})
		program := cli.New_Single(cli.New_Single_Input{
			Label: "p", Flags: []cli.Option{flag},
		})
		command := program.Selection[0].Single_Commands[0]
		var output cli.Output
		if profile.Description_Size == strings.TEXT_SIZE_MAXIMUM {
			assert_panics(t, "maximum flag row exceeds fixed output", func() {
				cli.Print_Command(&output, program, command)
			})
			continue
		}
		cli.Print_Command(&output, program, command)
	}
}

// Test_Bounded_Underscored_Labels keeps every correction-text edge observable.
func Test_Bounded_Underscored_Labels(t *testing.T) {
	sizes := [...]int{1, 2, cli.OPTION_LABEL_SIZE_MAXIMUM}
	for _, size := range sizes {
		label := cli_boundary_text(size, '_')
		assert_panics(t, "underscored option label", func() {
			cli.New_Single(cli.New_Single_Input{
				Label: "p",
				Flags: []cli.Option{{Label: cli.Option_Label(label), Value: false}},
			})
		})
	}
}

// Test_Bounded_Empty_Validation keeps constructor-only invalid minima observable.
func Test_Bounded_Empty_Validation(t *testing.T) {
	cli.New_Single(cli.New_Single_Input{})
	cli_bounded_empty_parse(t)
	assert_panics(t, "empty argument label", func() {
		cli.New_Single(cli.New_Single_Input{
			Label: "p", Arguments: []cli.Option{{Value: ""}},
		})
	})
	assert_panics(t, "empty flag label", func() {
		cli.New_Single(cli.New_Single_Input{
			Label: "p", Flags: []cli.Option{{Value: false}},
		})
	})
	assert_panics(t, "empty command declarations", func() {
		cli.New(cli.New_Input{Label: "p"})
	})
	assert_panics(t, "empty external key", func() {
		cli.New_Single(cli.New_Single_Input{
			Label:                 "p",
			Environment_Variables: []cli.Environment_Variable{{Key: "", Value: ""}},
		})
	})
	assert_panics(t, "empty string environment enum", func() {
		cli.New_Single(cli.New_Single_Input{
			Label: "p",
			Environment_Variables: []cli.Environment_Variable{{
				Key: "E", Value: "", Enum: []string{},
			}},
		})
	})
	assert_panics(t, "empty integer environment enum", func() {
		cli.New_Single(cli.New_Single_Input{
			Label: "p",
			Environment_Variables: []cli.Environment_Variable{{
				Key: "E", Value: int(0), Enum: []int{},
			}},
		})
	})

	program := cli_raw_single_program("", cli.Command{Label: "p"}, nil, nil)
	var parser cli.Parser
	cli.Program_Parse(&program, &parser, cli.Program_Parse_Input{
		Arguments: []string{"p"},
	})
	if _, complete := cli.Parser_Done(&parser); !complete {
		t.Fatal("empty program label parse did not complete")
	}

	unknown_program := cli_raw_single_program("p", cli.Command{Label: "p"}, nil, nil)
	var unknown_parser cli.Parser
	cli_test_program_parse(&unknown_program, &unknown_parser, cli.Program_Parse_Input{
		Arguments: []string{"p", "-=x"},
	})
	unknown_result, unknown_complete := cli.Parser_Done(&unknown_parser)
	if !unknown_complete {
		t.Fatal("empty option label parse did not complete")
	}
	if unknown_result.Error == nil {
		t.Fatal("empty option label was accepted")
	}

	assert_panics(t, "empty option lookup", func() {
		cli.Get_Option([]cli.Option{}, "")
	})
	var output cli.Output
	if handle_completion(cli.Program{}, nil, &output) {
		t.Fatal("empty completion request was handled")
	}
}

func cli_bounded_empty_parse(t *testing.T) {
	t.Helper()
	empty_program := cli_raw_single_program("", cli.Command{}, nil, nil)
	cli_parse_zero_option_command(t, empty_program)
	var unexpected_parser cli.Parser
	cli_test_program_parse(&empty_program, &unexpected_parser, cli.Program_Parse_Input{
		Arguments: []string{"p", "unexpected"},
	})
	unexpected_result, unexpected_complete := cli.Parser_Done(&unexpected_parser)
	if !unexpected_complete {
		t.Fatal("empty command unexpected argument did not complete")
	}
	if unexpected_result.Error == nil {
		t.Fatal("empty command accepted an unexpected argument")
	}

	deprecated_program := cli_raw_single_program(
		"", cli.Command{
			Deprecated: "d",
			Arguments:  []cli.Option{{Label: "values", Value: []string{}}},
		}, nil, nil,
	)
	var deprecated_parser cli.Parser
	cli_test_program_parse(&deprecated_program, &deprecated_parser, cli.Program_Parse_Input{
		Arguments: []string{"p"},
	})
	deprecated_result, deprecated_complete := cli.Parser_Done(&deprecated_parser)
	if !deprecated_complete {
		t.Fatal("empty deprecated command did not complete")
	}
	if deprecated_result.Error != nil {
		t.Fatalf("empty deprecated command: %v", deprecated_result.Error)
	}
}

// Test_Bounded_Named_Assignment keeps complete-token value and name maxima observable.
func Test_Bounded_Named_Assignment(t *testing.T) {
	value := cli_boundary_text(cli.NAMED_VALUE_SIZE_MAXIMUM, 'v')
	cli_parse_named_boundary(t, cli.New_Flag(cli.New_Flag_Input[string]{
		Label: "x",
	}), "-x="+value)
	for _, token := range []string{"-x=x", "-x=xx", `-x=""`, `-x="xx"`} {
		cli_parse_named_boundary(t, cli.New_Flag(cli.New_Flag_Input[string]{
			Label: "x",
		}), token)
	}
	label := cli_boundary_text(cli.OPTION_LABEL_SIZE_MAXIMUM, 'x')
	cli_parse_named_boundary(t, cli.New_Flag(cli.New_Flag_Input[bool]{
		Label: cli.Option_Label(label),
	}), "-"+label)
	variadic := cli.New_Variadic[string](cli.New_Variadic_Input{Label: "value"})
	variadic_program := cli.New_Single(cli.New_Single_Input{
		Label: "p", Arguments: []cli.Option{variadic},
	})
	cli_parse_variadic_boundary(t, variadic_program, []string{"p", "-value=x"})
}

func cli_parse_named_boundary(t *testing.T, flag cli.Option, token string) {
	t.Helper()
	program := cli.New_Single(cli.New_Single_Input{
		Label: "p", Flags: []cli.Option{flag},
	})
	var parser cli.Parser
	cli.Program_Parse(&program, &parser, cli.Program_Parse_Input{
		Arguments: []string{"p", token}, Command_Flags: make([]cli.Option, 1),
		Filled: make([]bool, 1),
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("named boundary parse did not complete")
	}
	if result.Error != nil {
		t.Fatalf("named boundary parse: %v", result.Error)
	}
}

// Test_Bounded_Parse_Phases keeps maximum selection and named lookup indices observable.
func Test_Bounded_Parse_Phases(t *testing.T) {
	arguments := make([]cli.Option, slices.COUNT_MAXIMUM)
	for index := range arguments {
		arguments[index] = cli.Option{Label: "a", Value: ""}
	}
	arguments[len(arguments)-1].Label = "x"
	argument_program := cli_raw_single_program(
		"p", cli.Command{Label: "p", Arguments: arguments}, nil, nil,
	)
	var argument_parser cli.Parser
	cli_test_program_parse(&argument_program, &argument_parser, cli.Program_Parse_Input{
		Arguments: []string{"p", "-x=value"},
	})
	if _, complete := cli.Parser_Done(&argument_parser); !complete {
		t.Fatal("maximum named argument lookup did not complete")
	}

	commands := make([]cli.Command, slices.COUNT_MAXIMUM)
	for index := range commands {
		commands[index].Hidden = true
	}
	commands[len(commands)-1] = cli.Command{Label: "last", Hidden: true}
	selection_program := cli.Program{
		Label:     "p",
		Selection: cli.Program_Selection{{Commands: commands}},
	}
	var selection_parser cli.Parser
	cli_test_program_parse(&selection_program, &selection_parser, cli.Program_Parse_Input{
		Arguments: []string{"p", "last"},
	})
	selection_result, selection_complete := cli.Parser_Done(&selection_parser)
	if !selection_complete {
		t.Fatal("maximum command selection did not complete")
	}
	if selection_result.Error != nil {
		t.Fatalf("maximum command selection: %v", selection_result.Error)
	}

	empty_name_program := cli.Program{
		Label:     "p",
		Selection: cli.Program_Selection{{Commands: []cli.Command{{Label: "a"}}}},
	}
	var empty_name_parser cli.Parser
	cli_test_program_parse(&empty_name_program, &empty_name_parser, cli.Program_Parse_Input{
		Arguments: []string{"p", ""},
	})
	if _, complete := cli.Parser_Done(&empty_name_parser); !complete {
		t.Fatal("empty command selection did not complete")
	}
}

// Test_Bounded_Option_Index keeps the final admitted validation index observable.
func Test_Bounded_Option_Index(t *testing.T) {
	flags := make([]cli.Option, slices.COUNT_MAXIMUM)
	for index := range flags {
		flags[index] = cli.Option{
			Label: cli.Option_Label("f" + cli_decimal_text(index)), Value: false,
			State: cli.Option_State{{Is_Flag: true}},
		}
	}
	cli.New_Single(cli.New_Single_Input{Label: "p", Flags: flags})
}

// Test_Bounded_Deprecation_Count keeps the two-warning diagnostic width observable.
func Test_Bounded_Deprecation_Count(t *testing.T) {
	for warning_count := 1; warning_count <= 2; warning_count++ {
		deprecated := cli.Deprecation("")
		if warning_count == 2 {
			deprecated = "d"
		}
		program := cli_raw_single_program("p", cli.Command{
			Label: "p", Deprecated: "d",
			Flags: []cli.Option{
				cli.New_Flag(cli.New_Flag_Input[bool]{
					Label: "f", Deprecated: deprecated,
				}),
			},
		}, nil, nil)
		var parser cli.Parser
		cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
			Arguments:            []string{"p", "-f"},
			Deprecation_Warnings: make([]cli.Warning, warning_count),
		})
		result, complete := cli.Parser_Done(&parser)
		if !complete {
			t.Fatalf("%d deprecation warnings did not publish", warning_count)
		}
		if result.Error != nil {
			t.Fatalf("%d deprecation warnings: %v", warning_count, result.Error)
		}
		if len(result.Command.Deprecation_Warnings) != warning_count {
			t.Fatalf(
				"deprecation warning count = %d",
				len(result.Command.Deprecation_Warnings),
			)
		}
	}
	flag_count := cli.DEPRECATION_WARNING_COUNT_MAXIMUM - len("warning")/len("warning")
	flags := make([]cli.Option, flag_count)
	tokens := make([]string, flag_count+len("p"))
	tokens[0] = "p"
	for index := range flags {
		label := "f" + cli_decimal_text(index)
		flags[index] = cli.New_Flag(cli.New_Flag_Input[bool]{
			Label: cli.Option_Label(label), Deprecated: "d",
		})
		tokens[index+1] = "-" + label
	}
	program := cli_raw_single_program(
		"p", cli.Command{Label: "p", Deprecated: "d", Flags: flags}, nil, nil,
	)
	var parser cli.Parser
	cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
		Arguments:            tokens,
		Deprecation_Warnings: make([]cli.Warning, cli.DEPRECATION_WARNING_COUNT_MAXIMUM),
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("maximum deprecation warnings did not publish")
	}
	if result.Error != nil {
		t.Fatalf("maximum deprecation warnings: %v", result.Error)
	}
}

func cli_bounded_option_constructors(t *testing.T, count int) {
	t.Helper()
	text_size := count
	if text_size > strings.TEXT_SIZE_MAXIMUM {
		text_size = strings.TEXT_SIZE_MAXIMUM
	}
	label_size := text_size
	if label_size == strings.TEXT_SIZE_MAXIMUM {
		label_size -= len("-")
	}
	label := cli_boundary_text(label_size, 'o')
	text := cli_boundary_text(text_size, 'd')
	strings_enum := make([]string, count)
	integers_enum := make([]int, count)
	for index := range strings_enum {
		strings_enum[index] = text
		integers_enum[index] = count
	}
	flag := cli.New_String_Enum_Flag(cli.New_String_Enum_Flag_Input{
		Label: cli.Option_Label(label), Value: cli.String_Enum_Default(text),
		Enum: strings_enum, Description: cli.Description(text),
		Hidden: true, Deprecated: cli.Deprecation(text),
	})
	cli.New_String_Enum_Argument(cli.New_String_Enum_Argument_Input{
		Label: cli.Option_Label(label), Enum: strings_enum,
		Description: cli.Description(text),
	})
	cli.New_Integer_Enum_Flag(cli.New_Integer_Enum_Flag_Input{
		Label: cli.Option_Label(label), Value: cli.Integer_Enum_Default(count),
		Enum: integers_enum, Description: cli.Description(text),
		Hidden: true, Deprecated: cli.Deprecation(text),
	})
	cli.New_Integer_Enum_Argument(cli.New_Integer_Enum_Argument_Input{
		Label: cli.Option_Label(label), Enum: integers_enum,
		Description: cli.Description(text),
	})
	if flag.Label == "" {
		t.Fatal("bounded option constructor discarded its label")
	}
}

func cli_bounded_external_constructors(t *testing.T, count int) {
	t.Helper()
	text_size := count
	if text_size > strings.TEXT_SIZE_MAXIMUM {
		text_size = strings.TEXT_SIZE_MAXIMUM
	}
	key_size := text_size
	if key_size > cli.EXTERNAL_KEY_SIZE_MAXIMUM {
		key_size = cli.EXTERNAL_KEY_SIZE_MAXIMUM
	}
	key := cli_boundary_text(key_size, 'K')
	secret_key_size := text_size
	if secret_key_size == strings.TEXT_SIZE_MAXIMUM {
		secret_key_size -= len("/")
	}
	secret_key := cli_boundary_text(secret_key_size, 'S')
	text := cli_boundary_text(text_size, 'd')
	paths := make([]string, count)
	strings_enum := make([]string, count)
	integers_enum := make([]int, count)
	for index := range paths {
		paths[index] = "/" + secret_key
		strings_enum[index] = text
		integers_enum[index] = count
	}
	variable := cli.New_Integer_Enum_Environment_Variable(
		cli.New_Integer_Enum_Environment_Variable_Input{
			Key: cli.External_Key(key), Description: cli.Description(text),
			Value: cli.Integer_Enum_Default(count), Enum: integers_enum,
			Required: true, Allow_Empty: true, Hidden: true,
			Deprecated: cli.Deprecation(text),
		},
	)
	cli.New_String_Enum_Environment_Variable(
		cli.New_String_Enum_Environment_Variable_Input{
			Key: cli.External_Key(key), Description: cli.Description(text),
			Value: cli.String_Enum_Default(text), Enum: strings_enum,
			Required: true, Allow_Empty: true, Hidden: true,
			Deprecated: cli.Deprecation(text),
		},
	)
	cli.New_Environment_Variable(cli.New_Environment_Variable_Input[string]{
		Key: cli.External_Key(key), Description: cli.Description(text), Value: text,
		Required: true, Allow_Empty: true, Hidden: true,
		Deprecated: cli.Deprecation(text),
	})
	cli.New_String_Enum_Secret(cli.New_String_Enum_Secret_Input{
		Paths: paths, Description: cli.Description(text), Enum: strings_enum,
		Required: true, Allow_Empty: true, Hidden: true,
		Deprecated: cli.Deprecation(text),
	})
	cli.New_Integer_Enum_Secret(cli.New_Integer_Enum_Secret_Input{
		Paths: paths, Description: cli.Description(text), Enum: integers_enum,
		Required: true, Allow_Empty: true, Hidden: true,
		Deprecated: cli.Deprecation(text),
	})
	cli.New_Secret[string](cli.New_Secret_Input{
		Paths: paths, Description: cli.Description(text),
		Required: true, Allow_Empty: true, Hidden: true,
		Deprecated: cli.Deprecation(text),
	})
	if variable.Key == "" {
		t.Fatal("bounded external constructor discarded its key")
	}
}

func cli_bounded_argument_command(t *testing.T, count int) {
	t.Helper()
	option := cli_boundary_option(count, []string{})
	arguments := make([]cli.Option, count)
	for index := range arguments {
		arguments[index] = option
	}
	command := cli_boundary_command(count)
	command.Arguments = arguments
	program := cli_raw_single_program("p", command, nil, nil)
	program.Selection[0].Help_Flags = cli.Help_Flags{
		{State: cli.Option_State{{Hidden: true}}},
		{State: cli.Option_State{{Hidden: true}}},
	}
	cli_parse_boundary_command(t, &program, count, []string{"p"})
	cli_complete(program, []string{"p", "-"})
	cli_parse_help_context(t, command, count)
	if cli.Get_Option(arguments, option.Label).Label != option.Label {
		t.Fatal("bounded argument lookup changed the declaration")
	}
	cli_render_boundary_command(t, program, command)
}

func cli_bounded_flag_command(t *testing.T, count int) {
	t.Helper()
	label_size := count
	if label_size == strings.TEXT_SIZE_MAXIMUM {
		label_size -= len("-")
	}
	label := cli_boundary_text(label_size, 'f')
	option := cli_boundary_option(count, false)
	option.Label = cli.Option_Label(label)
	option.State[0].Is_Flag = true
	flags := make([]cli.Option, count)
	for index := range flags {
		flags[index] = option
	}
	command := cli_boundary_command(count)
	command.Arguments = nil
	command.Flags = flags
	program := cli_raw_single_program("p", command, nil, nil)
	program.Selection[0].Help_Flags = cli.Help_Flags{
		{State: cli.Option_State{{Hidden: true}}},
		{State: cli.Option_State{{Hidden: true}}},
	}
	original := flags[0]
	enum_label_size := label_size
	if enum_label_size == strings.TEXT_SIZE_MAXIMUM-len("-") {
		enum_label_size -= len("=")
	}
	enum_label := cli_boundary_text(enum_label_size, 'e')
	flags[0].Label = cli.Option_Label(enum_label)
	flags[0].Value = "v"
	flags[0].Enum = []string{"v"}
	flags[0].State[0].String = "v"
	cli_complete(program, []string{"p", "-" + enum_label + "="})
	flags[0] = original
	cli_parse_boundary_command(t, &program, count, []string{"p", "-" + label})
	cli_complete(program, []string{"p", "-"})
	cli_parse_help_context(t, command, count)
	cli_render_boundary_command(t, program, command)
}

func cli_boundary_option(count int, value any) (option cli.Option) {
	text_size := count
	if text_size > strings.TEXT_SIZE_MAXIMUM {
		text_size = strings.TEXT_SIZE_MAXIMUM
	}
	text := cli_boundary_text(text_size, 'o')
	label_size := text_size
	if label_size == strings.TEXT_SIZE_MAXIMUM {
		label_size -= len("-")
	}
	label := cli_boundary_text(label_size, 'l')
	integer := count
	if count == slices.COUNT_MAXIMUM {
		integer = bits.INTEGER_MAXIMUM
	}
	return cli.Option{
		Label: cli.Option_Label(label), Description: cli.Description(text), Value: value,
		State: cli.Option_State{{
			String: cli.Value_Text(text), Integer: cli.Integer(integer), Boolean: true,
			Strings: make([]string, count), Integers: make([]int, count), Parsed: true,
			Is_Flag: true, Hidden: true, Deprecated: cli.Deprecation(text),
		}},
	}
}

func cli_boundary_command(count int) (command cli.Command) {
	text_size := count
	if text_size > strings.TEXT_SIZE_MAXIMUM {
		text_size = strings.TEXT_SIZE_MAXIMUM
	}
	text := cli_boundary_text(text_size, 'c')
	return cli.Command{
		Label: cli.Label(text), Description: cli.Description(text), Hidden: true,
		Deprecated:           cli.Deprecation(text),
		Deprecation_Warnings: make([]cli.Warning, cli_boundary_warning_count(count)),
		Environment:          make([]cli.Environment_Variable, count),
		Secrets:              make([]cli.Secret, count),
	}
}

func cli_boundary_warning_count(count int) (warning_count int) {
	maximum_count := strings.TEXT_SIZE_MAXIMUM / len("warning: \n")
	if count > maximum_count {
		return maximum_count
	}
	return count
}

func cli_parse_boundary_command(
	t *testing.T, program *cli.Program, count int, arguments []string,
) {
	t.Helper()
	parser := cli.Parser{
		Publication: cli.Publication{
			Result: cli.Parse_Result{
				Command: program.Selection[0].Single_Commands[0],
			},
			Environment_Errors: make([]error, count),
			Secret_Errors:      make([]cli.Secret_Failure, count),
			Environment_Warnings: make(
				[]cli.Warning, cli_boundary_warning_count(count),
			),
			Secret_Warnings: make([]cli.Warning, count),
			Secret_Count:    cli.Declaration_Count(count),
		},
		Workspace: cli.Workspace{
			Command_Arguments: make([]cli.Option, count),
			Command_Flags:     make([]cli.Option, count),
			Filled:            make([]bool, count),
			Positionals:       make([]cli.Indexed_Token, count),
			Slice_Named:       make([]cli.Indexed_Token, count),
		},
	}
	cli.Program_Parse(program, &parser, cli.Program_Parse_Input{
		Arguments:            arguments,
		Command_Arguments:    make([]cli.Option, count),
		Command_Flags:        make([]cli.Option, count),
		Filled:               make([]bool, count),
		Positionals:          make([]cli.Indexed_Token, count),
		Slice_Named:          make([]cli.Indexed_Token, count),
		String_Values:        make([]string, count),
		Integer_Values:       make([]int, count),
		Deprecation_Warnings: make([]cli.Warning, cli.DEPRECATION_WARNING_COUNT_MAXIMUM),
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("bounded command parser did not complete")
	}
	if result.Error != nil {
		t.Fatalf("bounded command parse: %v", result.Error)
	}
}

func cli_parse_help_context(t *testing.T, command cli.Command, count int) {
	t.Helper()
	program := cli.Program{
		Label: "p",
		Selection: cli.Program_Selection{{
			Commands: []cli.Command{command},
			Help_Flags: cli.Help_Flags{
				{State: cli.Option_State{{Hidden: true}}},
				{State: cli.Option_State{{Hidden: true}}},
			},
		}},
	}
	cli_complete(program, []string{"p", ""})
	var parser cli.Parser
	cli.Program_Parse(&program, &parser, cli.Program_Parse_Input{
		Arguments:         []string{"p", string(command.Label), "-help"},
		Command_Arguments: make([]cli.Option, count),
		Command_Flags:     make([]cli.Option, count),
		Filled:            make([]bool, count),
		Positionals:       make([]cli.Indexed_Token, count),
		Slice_Named:       make([]cli.Indexed_Token, count),
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("bounded help context did not complete")
	}
	if !errors.Is(result.Error, cli.Help_Requested) {
		t.Fatalf("bounded help context: %v", result.Error)
	}
}

func cli_render_boundary_command(t *testing.T, program cli.Program, command cli.Command) {
	t.Helper()
	var output cli.Output
	cli.Print_Deprecations(&output, command)
	cli.Output_Reset(&output)
	if len(command.Label)+len(command.Arguments) < strings.TEXT_SIZE_MAXIMUM {
		cli.Print_Requested_Help(&output, program, command)
		cli.Output_Reset(&output)
		cli.Print_Command(&output, program, command)
		return
	}
	assert_panics(t, "bounded requested help exceeds fixed output", func() {
		cli.Print_Requested_Help(&output, program, command)
	})
	cli.Output_Reset(&output)
	assert_panics(t, "bounded command exceeds fixed rendering output", func() {
		cli.Print_Command(&output, program, command)
	})
}

func cli_boundary_text(size int, character byte) (text string) {
	var content [strings.TEXT_SIZE_MAXIMUM]byte
	for index := range content[:size] {
		content[index] = character
	}
	return string(content[:size])
}

const ALLOCATION_ARGUMENT_COUNT = len("x")
const ALLOCATION_PROGRAM_TOKEN_COUNT = ALLOCATION_ARGUMENT_COUNT
const ALLOCATION_PARSE_TOKEN_COUNT = ALLOCATION_PROGRAM_TOKEN_COUNT + ALLOCATION_ARGUMENT_COUNT
const ALLOCATION_OPTION_COUNT = ALLOCATION_ARGUMENT_COUNT

// Construction belongs inside allocation contract because every definition remains borrowed.
func Test_New_Single_Zero_Allocation(t *testing.T) {
	var program cli.Program
	input := cli.New_Single_Input{
		Label: "program",
		Arguments: []cli.Option{
			cli.New_Argument[string](cli.New_Argument_Input{Label: "value"}),
		},
	}
	testify.Zero_Allocation(t, func() {
		program = cli.New_Single(input)
	})
	if program.Label == "" {
		t.Fatal("construction result was discarded")
	}
}

// Selector choice changes fixed program state, never ownership of declarations.
func Test_Program_Constructors_Zero_Allocation(t *testing.T) {
	commands := []cli.Command{{Label: "run"}}
	var program cli.Program
	operations := [...]func(){
		func() { program = cli.New(cli.New_Input{Label: "program", Commands: commands}) },
		func() { program = cli.New_Single(cli.New_Single_Input{Label: "program"}) },
		func() {
			program = cli.New_Multicall(cli.New_Multicall_Input{
				Label: "program", Commands: commands,
			})
		},
	}
	for _, operation := range operations {
		testify.Zero_Allocation(t, operation)
	}
	if program.Label == "" {
		t.Fatal("program constructor result was discarded")
	}
}

// Every typed option constructor must preserve borrowed input without interface boxing.
func Test_Option_Constructors_Zero_Allocation(t *testing.T) {
	strings := []string{"x"}
	integers := []int{1}
	var result cli.Option
	operations := [...]func(){
		func() { result = cli.New_Argument[string](cli.New_Argument_Input{Label: "x"}) },
		func() { result = cli.New_Argument[int](cli.New_Argument_Input{Label: "x"}) },
		func() { result = cli.New_Argument[bool](cli.New_Argument_Input{Label: "x"}) },
		func() { result = cli.New_Variadic[string](cli.New_Variadic_Input{Label: "x"}) },
		func() { result = cli.New_Variadic[int](cli.New_Variadic_Input{Label: "x"}) },
		func() { result = cli.New_Flag(cli.New_Flag_Input[string]{Label: "x"}) },
		func() { result = cli.New_Flag(cli.New_Flag_Input[int]{Label: "x"}) },
		func() { result = cli.New_Flag(cli.New_Flag_Input[bool]{Label: "x"}) },
		func() {
			result = cli.New_String_Enum_Flag(cli.New_String_Enum_Flag_Input{
				Label: "x", Value: "x", Enum: strings,
			})
		},
		func() {
			result = cli.New_Integer_Enum_Flag(cli.New_Integer_Enum_Flag_Input{
				Label: "x", Value: 1, Enum: integers,
			})
		},
		func() {
			result = cli.New_String_Enum_Argument(cli.New_String_Enum_Argument_Input{
				Label: "x", Enum: strings,
			})
		},
		func() {
			result = cli.New_Integer_Enum_Argument(cli.New_Integer_Enum_Argument_Input{
				Label: "x", Enum: integers,
			})
		},
	}
	for _, operation := range operations {
		testify.Zero_Allocation(t, operation)
	}
	if result.Label == "" {
		t.Fatal("option constructor result was discarded")
	}
}

// External constructors must retain enum and path storage without boxing borrowed slices.
func Test_External_Constructors_Zero_Allocation(t *testing.T) {
	strings := []string{"x"}
	integers := []int{1}
	paths := []string{"/X"}
	var environment cli.Environment_Variable
	var secret cli.Secret
	operations := [...]func(){
		func() {
			environment = cli.New_Environment_Variable(
				cli.New_Environment_Variable_Input[string]{Key: "X"})
		},
		func() {
			environment = cli.New_Environment_Variable(
				cli.New_Environment_Variable_Input[int]{Key: "X"})
		},
		func() {
			environment = cli.New_Environment_Variable(
				cli.New_Environment_Variable_Input[bool]{Key: "X"})
		},
		func() {
			environment = cli.New_String_Enum_Environment_Variable(
				cli.New_String_Enum_Environment_Variable_Input{
					Key: "X", Value: "x", Enum: strings,
				})
		},
		func() {
			environment = cli.New_Integer_Enum_Environment_Variable(
				cli.New_Integer_Enum_Environment_Variable_Input{
					Key: "X", Value: 1, Enum: integers,
				})
		},
		func() { secret = cli.New_Secret[string](cli.New_Secret_Input{Paths: paths}) },
		func() { secret = cli.New_Secret[int](cli.New_Secret_Input{Paths: paths}) },
		func() { secret = cli.New_Secret[bool](cli.New_Secret_Input{Paths: paths}) },
		func() {
			secret = cli.New_String_Enum_Secret(cli.New_String_Enum_Secret_Input{
				Paths: paths, Enum: strings,
			})
		},
		func() {
			secret = cli.New_Integer_Enum_Secret(cli.New_Integer_Enum_Secret_Input{
				Paths: paths, Enum: integers,
			})
		},
	}
	for _, operation := range operations {
		testify.Zero_Allocation(t, operation)
	}
	if environment.Key == "" {
		t.Fatal("external constructor result was discarded")
	}
	if secret.Key == "" {
		t.Fatal("external constructor result was discarded")
	}
}

// Rejection stays allocation-free because malicious declarations are normal input.
func Test_Panic_Paths_Zero_Allocation(t *testing.T) {
	invalid_secret := cli.New_Secret[string](cli.New_Secret_Input{
		Paths: []string{"relative"},
	})
	invalid_secrets := []cli.Secret{invalid_secret}
	operations := [...]struct {
		Name      string
		Operation func()
	}{
		{"command", func() {
			cli.New(cli.New_Input{
				Label: "program", Commands: []cli.Command{{}},
			})
		}},
		{"option label", func() {
			cli.New_Single(cli.New_Single_Input{
				Label: "program", Flags: []cli.Option{{Label: "bad_label"}},
			})
		}},
		{"option lookup", func() { cli.Get_Option(cli.Options(nil), "UNKNOWN") }},
		{"environment lookup", func() { cli.Get_Environment(nil, "UNKNOWN") }},
		{"secret lookup", func() { cli.Get_Secret(nil, "UNKNOWN") }},
		{"secret path", func() {
			cli.New_Single(cli.New_Single_Input{
				Label: "program", Secrets: invalid_secrets,
			})
		}},
	}
	for _, one := range operations {
		t.Run(one.Name, func(t *testing.T) {
			panicked := false
			testify.Zero_Allocation(t, func() {
				panicked = cli_panic(one.Operation)
			})
			if !panicked {
				t.Fatal("rejected input did not panic")
			}
		})
	}
}

func cli_panic(operation func()) (panicked bool) {
	defer func() { panicked = recover() != nil }()
	operation()
	return false
}

// Parsing must write into bounded caller state instead of constructing hidden storage.
func Test_Program_Parse_Zero_Allocation(t *testing.T) {
	var parser cli.Parser
	var arguments [ALLOCATION_ARGUMENT_COUNT]cli.Option
	var filled [ALLOCATION_OPTION_COUNT]bool
	var positionals [ALLOCATION_PARSE_TOKEN_COUNT]cli.Indexed_Token
	var slice_named [ALLOCATION_PARSE_TOKEN_COUNT]cli.Indexed_Token
	program := cli.New_Single(cli.New_Single_Input{
		Label: "program",
		Arguments: []cli.Option{
			cli.New_Argument[string](cli.New_Argument_Input{Label: "value"}),
		},
	})
	input := cli.Program_Parse_Input{
		Arguments:         []string{"program", "content"},
		Command_Arguments: arguments[:],
		Filled:            filled[:],
		Positionals:       positionals[:],
		Slice_Named:       slice_named[:],
	}
	testify.Zero_Allocation(t, func() {
		cli.Program_Parse(&program, &parser, input)
	})
	if _, complete := cli.Parser_Done(&parser); !complete {
		t.Fatal("allocation parse did not publish")
	}
}

// Malicious syntax belongs inside allocation contract, not only successful parsing.
func Test_Program_Parse_Unknown_Option_Zero_Allocation(t *testing.T) {
	program := cli.New_Single(cli.New_Single_Input{Label: "program"})
	var parser cli.Parser
	var result cli.Parse_Result
	var complete cli.Parser_Complete
	var global_flags [cli.HELP_FLAG_COUNT]cli.Option
	var filled [cli.HELP_FLAG_COUNT]bool
	var positionals [ALLOCATION_PROGRAM_TOKEN_COUNT]cli.Indexed_Token
	var slice_named [ALLOCATION_PROGRAM_TOKEN_COUNT]cli.Indexed_Token
	var failures [ALLOCATION_ARGUMENT_COUNT]error
	input := cli.Program_Parse_Input{
		Arguments: []string{"program", "-unknown"}, Global_Flags: global_flags[:],
		Filled: filled[:], Positionals: positionals[:], Slice_Named: slice_named[:],
		Failures: failures[:],
	}
	testify.Zero_Allocation(t, func() {
		parser = cli.Parser{}
		cli.Program_Parse(&program, &parser, input)
		result, complete = cli.Parser_Done(&parser)
	})
	if !complete {
		t.Fatal("unknown option parse did not publish")
	}
	if result.Error == nil {
		t.Fatal("unknown option parse lost its error")
	}
}

type cli_parse_error_case struct {
	Name      string
	Program   cli.Program
	Arguments []string
}

func cli_parse_error_cases() (cases []cli_parse_error_case) {
	boolean_flag := cli.New_Single(cli.New_Single_Input{
		Label: "program", Flags: []cli.Option{
			cli.New_Flag(cli.New_Flag_Input[bool]{Label: "flag"}),
		},
	})
	string_flag := cli.New_Single(cli.New_Single_Input{
		Label: "program", Flags: []cli.Option{
			cli.New_Flag(cli.New_Flag_Input[string]{Label: "text"}),
		},
	})
	integer_flag := cli.New_Single(cli.New_Single_Input{
		Label: "program", Flags: []cli.Option{
			cli.New_Flag(cli.New_Flag_Input[int]{Label: "count"}),
		},
	})
	string_enum := cli.New_Single(cli.New_Single_Input{
		Label: "program", Flags: []cli.Option{
			cli.New_String_Enum_Flag(cli.New_String_Enum_Flag_Input{
				Label: "color", Value: "always", Enum: []string{"always", "never"},
			}),
		},
	})
	integer_enum := cli.New_Single(cli.New_Single_Input{
		Label: "program", Flags: []cli.Option{
			cli.New_Integer_Enum_Flag(cli.New_Integer_Enum_Flag_Input{
				Label: "level", Value: 1, Enum: []int{1, 2},
			}),
		},
	})
	missing_argument := cli.New_Single(cli.New_Single_Input{
		Label: "program", Arguments: []cli.Option{
			cli.New_Argument[string](cli.New_Argument_Input{Label: "value"}),
		},
	})
	integer_argument := cli.New_Single(cli.New_Single_Input{
		Label: "program", Arguments: []cli.Option{
			cli.New_Argument[int](cli.New_Argument_Input{Label: "count"}),
		},
	})
	integer_variadic := cli.New_Single(cli.New_Single_Input{
		Label: "program", Arguments: []cli.Option{
			cli.New_Variadic[int](cli.New_Variadic_Input{Label: "count"}),
		},
	})
	commands := cli.New(cli.New_Input{
		Label: "program", Commands: []cli.Command{{Label: "build"}},
	})
	return []cli_parse_error_case{
		{"double dash", boolean_flag, []string{"program", "--flag"}},
		{"repeated flag", boolean_flag, []string{"program", "-flag", "-flag"}},
		{"missing named value", string_flag, []string{"program", "-text"}},
		{"invalid named integer", integer_flag, []string{"program", "-count=no"}},
		{"near string enum", string_enum, []string{"program", "-color=alway"}},
		{"far string enum", string_enum, []string{"program", "-color=other"}},
		{"integer enum", integer_enum, []string{"program", "-level=3"}},
		{"too many arguments", boolean_flag, []string{"program", "extra"}},
		{"missing argument", missing_argument, []string{"program"}},
		{"invalid positional integer", integer_argument, []string{"program", "no"}},
		{"invalid variadic integer", integer_variadic, []string{"program", "no"}},
		{"near command", commands, []string{"program", "buil"}},
		{"far command", commands, []string{"program", "other"}},
	}
}

// Every rejected CLI form must retain its diagnostic in caller parser storage.
func Test_Program_Parse_CLI_Errors_Zero_Allocation(t *testing.T) {
	for _, one := range cli_parse_error_cases() {
		t.Run(one.Name, func(t *testing.T) {
			cli_assert_parse_error_zero_allocation(t, one.Program, one.Arguments)
		})
	}
}

func cli_assert_parse_error_zero_allocation(
	t *testing.T, program cli.Program, arguments []string,
) {
	t.Helper()
	input := cli_test_parse_input(program, arguments)
	var parser cli.Parser
	var result cli.Parse_Result
	var complete cli.Parser_Complete
	var message string
	testify.Zero_Allocation(t, func() {
		parser = cli.Parser{}
		cli.Program_Parse(&program, &parser, input)
		result, complete = cli.Parser_Done(&parser)
		if result.Error != nil {
			message = result.Error.Error()
		}
	})
	if !complete {
		t.Fatal("rejected CLI parse did not publish")
	}
	if result.Error == nil {
		t.Fatal("rejected CLI parse lost its error")
	}
	if message == "" {
		t.Fatal("rejected CLI parse lost its diagnostic")
	}
}

// Deprecated command and flag publication retains borrowed warning parts.
func Test_Program_Parse_Deprecation_Zero_Allocation(t *testing.T) {
	program := cli.New_Single(cli.New_Single_Input{
		Label: "program",
		Flags: []cli.Option{cli.New_Flag(cli.New_Flag_Input[bool]{
			Label: "old", Deprecated: "use new",
		})},
	})
	program.Selection[0].Single_Commands[0].Deprecated = "use next"
	var parser cli.Parser
	var flags [ALLOCATION_ARGUMENT_COUNT]cli.Option
	var filled [ALLOCATION_OPTION_COUNT]bool
	var positionals [ALLOCATION_PARSE_TOKEN_COUNT]cli.Indexed_Token
	var slice_named [ALLOCATION_PARSE_TOKEN_COUNT]cli.Indexed_Token
	var warnings [ALLOCATION_PARSE_TOKEN_COUNT]cli.Warning
	input := cli.Program_Parse_Input{
		Arguments: []string{"program", "-old"}, Command_Flags: flags[:],
		Filled: filled[:], Positionals: positionals[:], Slice_Named: slice_named[:],
		Deprecation_Warnings: warnings[:],
	}
	testify.Zero_Allocation(t, func() {
		cli.Program_Parse(&program, &parser, input)
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("deprecated allocation parse did not publish")
	}
	if len(result.Command.Deprecation_Warnings) != len(warnings) {
		t.Fatal("deprecated allocation parse lost warnings")
	}
}

// Empty parsing proves caller storage does not hide allocation behind scalar assignment.
func Test_Program_Parse_Empty_Zero_Allocation(t *testing.T) {
	program := cli.New_Single(cli.New_Single_Input{Label: "program"})
	var parser cli.Parser
	var global_flags [cli.HELP_FLAG_COUNT]cli.Option
	var filled [cli.HELP_FLAG_COUNT]bool
	var positionals [ALLOCATION_PROGRAM_TOKEN_COUNT]cli.Indexed_Token
	var slice_named [ALLOCATION_PROGRAM_TOKEN_COUNT]cli.Indexed_Token
	var string_values [ALLOCATION_PARSE_TOKEN_COUNT]string
	var integer_values [ALLOCATION_PARSE_TOKEN_COUNT]int
	var failures [ALLOCATION_PARSE_TOKEN_COUNT]error
	input := cli.Program_Parse_Input{
		Arguments:     []string{"program"},
		Global_Flags:  global_flags[:],
		Filled:        filled[:],
		Positionals:   positionals[:],
		Slice_Named:   slice_named[:],
		String_Values: string_values[:], Integer_Values: integer_values[:],
		Failures: failures[:],
	}
	testify.Zero_Allocation(t, func() {
		cli.Program_Parse(&program, &parser, input)
	})
	if _, complete := cli.Parser_Done(&parser); !complete {
		t.Fatal("empty allocation parse did not publish")
	}
}

// Environment parsing must use caller maps and publication slices.
func Test_Program_Parse_Environment_Zero_Allocation(t *testing.T) {
	program := cli.New_Single(cli.New_Single_Input{
		Label: "program",
		Environment_Variables: []cli.Environment_Variable{
			cli.New_Environment_Variable[string](
				cli.New_Environment_Variable_Input[string]{
					Key: "E", Deprecated: "use NEW_E",
				},
			),
		},
	})
	var parser cli.Parser
	var environment_values [ALLOCATION_ARGUMENT_COUNT]cli.Environment_Variable
	var environment_errors [ALLOCATION_ARGUMENT_COUNT]error
	var environment_warnings [ALLOCATION_ARGUMENT_COUNT]cli.Warning
	var failures [ALLOCATION_ARGUMENT_COUNT]error
	var integer_values [ALLOCATION_ARGUMENT_COUNT]int
	var deprecation_warnings [ALLOCATION_ARGUMENT_COUNT]cli.Warning
	environment_sources := make(cli.Environment_Sources, 1)
	input := cli.Program_Parse_Input{
		Arguments: []string{"program"}, Environment: []string{"E=value"},
		Environment_Values:   environment_values[:],
		Environment_Errors:   environment_errors[:],
		Environment_Warnings: environment_warnings[:],
		Environment_Sources:  environment_sources,
		Failures:             failures[:],
		Integer_Values:       integer_values[:],
		Deprecation_Warnings: deprecation_warnings[:],
	}
	testify.Zero_Allocation(t, func() {
		cli.Program_Parse(&program, &parser, input)
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("environment allocation parse did not publish")
	}
	if result.Error != nil {
		t.Fatalf("environment allocation parse: %v", result.Error)
	}
}

type cli_environment_error_case struct {
	Name        string
	Program     cli.Program
	Environment []string
}

func cli_environment_error_cases() (cases []cli_environment_error_case) {
	required := cli.New_Single(cli.New_Single_Input{
		Label: "program", Environment_Variables: []cli.Environment_Variable{
			cli.New_Environment_Variable[string](
				cli.New_Environment_Variable_Input[string]{
					Key: "VALUE", Required: true,
				},
			),
		},
	})
	integer := cli.New_Single(cli.New_Single_Input{
		Label: "program", Environment_Variables: []cli.Environment_Variable{
			cli.New_Environment_Variable[int](
				cli.New_Environment_Variable_Input[int]{Key: "VALUE"},
			),
		},
	})
	boolean := cli.New_Single(cli.New_Single_Input{
		Label: "program", Environment_Variables: []cli.Environment_Variable{
			cli.New_Environment_Variable[bool](
				cli.New_Environment_Variable_Input[bool]{Key: "VALUE"},
			),
		},
	})
	string_enum := cli.New_Single(cli.New_Single_Input{
		Label: "program", Environment_Variables: []cli.Environment_Variable{
			cli.New_String_Enum_Environment_Variable(
				cli.New_String_Enum_Environment_Variable_Input{
					Key: "VALUE", Value: "always",
					Enum: []string{"always", "never"},
				},
			),
		},
	})
	integer_enum := cli.New_Single(cli.New_Single_Input{
		Label: "program", Environment_Variables: []cli.Environment_Variable{
			cli.New_Integer_Enum_Environment_Variable(
				cli.New_Integer_Enum_Environment_Variable_Input{
					Key: "VALUE", Value: 1, Enum: []int{1, 2},
				},
			),
		},
	})
	two_required := cli_two_required_environment_program()
	return []cli_environment_error_case{
		{"required", required, nil},
		{"malformed", required, []string{"VALUE"}},
		{"duplicate", required, []string{"VALUE=x", "VALUE=y"}},
		{"malformed duplicate", required, []string{"VALUE", "VALUE=x"}},
		{"integer", integer, []string{"VALUE=no"}},
		{"Boolean", boolean, []string{"VALUE=no"}},
		{"near string enum", string_enum, []string{"VALUE=alway"}},
		{"far string enum", string_enum, []string{"VALUE=other"}},
		{"integer enum", integer_enum, []string{"VALUE=3"}},
		{"ordered", two_required, nil},
	}
}

func cli_two_required_environment_program() (program cli.Program) {
	return cli.New_Single(cli.New_Single_Input{
		Label: "program", Environment_Variables: []cli.Environment_Variable{
			cli.New_Environment_Variable[string](
				cli.New_Environment_Variable_Input[string]{
					Key: "FIRST", Required: true,
				},
			),
			cli.New_Environment_Variable[string](
				cli.New_Environment_Variable_Input[string]{
					Key: "SECOND", Required: true,
				},
			),
		},
	})
}

// Environment rejection must render every source and type failure into parser storage.
func Test_Program_Parse_Environment_Errors_Zero_Allocation(t *testing.T) {
	for _, one := range cli_environment_error_cases() {
		t.Run(one.Name, func(t *testing.T) {
			cli_assert_environment_error_zero_allocation(
				t, one.Program, one.Environment,
			)
		})
	}
}

func cli_assert_environment_error_zero_allocation(
	t *testing.T, program cli.Program, environment []string,
) {
	t.Helper()
	input := cli_test_parse_input(program, []string{"program"})
	input.Environment = environment
	var parser cli.Parser
	var result cli.Parse_Result
	var complete cli.Parser_Complete
	var message string
	testify.Zero_Allocation(t, func() {
		parser = cli.Parser{}
		cli.Program_Parse(&program, &parser, input)
		result, complete = cli.Parser_Done(&parser)
		if result.Error != nil {
			message = result.Error.Error()
		}
	})
	if !complete {
		t.Fatal("rejected environment parse did not publish")
	}
	if result.Error == nil {
		t.Fatal("rejected environment parse lost its error")
	}
	if message == "" {
		t.Fatal("rejected environment parse lost its diagnostic")
	}
}

// Variadic parsing must publish a view over caller element storage.
func Test_Program_Parse_Variadic_Zero_Allocation(t *testing.T) {
	program := cli.New_Single(cli.New_Single_Input{
		Label: "program",
		Arguments: []cli.Option{
			cli.New_Variadic[string](cli.New_Variadic_Input{Label: "value"}),
		},
	})
	var parser cli.Parser
	var arguments [ALLOCATION_ARGUMENT_COUNT]cli.Option
	var filled [ALLOCATION_OPTION_COUNT]bool
	var positionals [ALLOCATION_ARGUMENT_COUNT]cli.Indexed_Token
	var slice_named [ALLOCATION_ARGUMENT_COUNT]cli.Indexed_Token
	var string_values [ALLOCATION_ARGUMENT_COUNT]string
	var deprecation_warnings [ALLOCATION_PARSE_TOKEN_COUNT]cli.Warning
	input := cli.Program_Parse_Input{
		Arguments: []string{"program", "content"}, Command_Arguments: arguments[:],
		Filled: filled[:], Positionals: positionals[:], Slice_Named: slice_named[:],
		String_Values:        string_values[:],
		Deprecation_Warnings: deprecation_warnings[:],
	}
	testify.Zero_Allocation(t, func() {
		cli.Program_Parse(&program, &parser, input)
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("variadic allocation parse did not publish")
	}
	if result.Error != nil {
		t.Fatalf("variadic allocation parse: %v", result.Error)
	}
}

// Secret parsing must submit and retire through stable caller runners and buffers.
func Test_Program_Parse_Secret_Zero_Allocation(t *testing.T) {
	program := cli.New_Single(cli.New_Single_Input{
		Label: "program",
		Secrets: []cli.Secret{
			cli.New_Secret[string](cli.New_Secret_Input{
				Paths: []string{"/TOKEN"}, Required: true,
				Deprecated: "use NEW_TOKEN",
			}),
		},
	})
	var parser cli.Parser
	var secret_values [ALLOCATION_ARGUMENT_COUNT]cli.Secret
	var secret_errors [ALLOCATION_ARGUMENT_COUNT]cli.Secret_Failure
	var secret_warnings [ALLOCATION_ARGUMENT_COUNT]cli.Warning
	var deprecation_warnings [ALLOCATION_ARGUMENT_COUNT]cli.Warning
	var secret_parsers [ALLOCATION_ARGUMENT_COUNT]cli.Secret_Parser
	var secret_buffer [cli.SECRET_BUFFER_BYTES_MAX]byte
	secret_buffers := [ALLOCATION_ARGUMENT_COUNT]cli.Secret_Bytes{secret_buffer[:]}
	var path_failure [ALLOCATION_ARGUMENT_COUNT]cli.Path_Failure
	path_failures := [ALLOCATION_ARGUMENT_COUNT]cli.Path_Failures{path_failure[:]}
	var failures [ALLOCATION_ARGUMENT_COUNT]error
	input := cli.Program_Parse_Input{
		Arguments: []string{"program"}, Loop: cli_secret_boundary_loop(),
		Secret_Values: secret_values[:], Secret_Errors: secret_errors[:],
		Secret_Warnings: secret_warnings[:], Secret_Parsers: secret_parsers[:],
		Secret_Buffers: secret_buffers[:], Secret_Path_Failures: path_failures[:],
		Failures: failures[:], Deprecation_Warnings: deprecation_warnings[:],
	}
	testify.Zero_Allocation(t, func() {
		cli.Program_Parse(&program, &parser, input)
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("secret allocation parse did not publish")
	}
	if result.Error != nil {
		t.Fatalf("secret allocation parse: %v", result.Error)
	}
}

// Secret rejection must preserve ordered path causes without heap-backed error trees.
func Test_Program_Parse_Secret_Errors_Zero_Allocation(t *testing.T) {
	program := cli.New_Single(cli.New_Single_Input{
		Label: "program",
		Secrets: []cli.Secret{
			cli.New_Secret[string](cli.New_Secret_Input{
				Paths: []string{"/missing/MISSING"}, Required: true,
			}),
			cli.New_Secret[string](cli.New_Secret_Input{
				Paths: []string{"/secrets/DIRECTORY"}, Required: true,
			}),
			cli.New_Secret[string](cli.New_Secret_Input{
				Paths: []string{"/secrets/OVERFLOW"}, Required: true,
			}),
			cli.New_Secret[int](cli.New_Secret_Input{
				Paths: []string{"/secrets/NUMBER"}, Required: true,
			}),
			cli.New_Secret[string](cli.New_Secret_Input{
				Paths: []string{"/secrets/NEGATIVE"}, Required: true,
			}),
			cli.New_Secret[string](cli.New_Secret_Input{
				Paths: []string{"/secrets/READ_OVERFLOW"}, Required: true,
			}),
			cli.New_String_Enum_Secret(cli.New_String_Enum_Secret_Input{
				Paths: []string{"/secrets/BOUNDARY_ENUM"},
				Enum:  []string{"allowed"}, Required: true,
			}),
			cli.New_Secret[int](cli.New_Secret_Input{
				Paths: []string{"/X"}, Required: true,
			}),
		},
	})
	input := cli_test_parse_input(program, []string{"program"})
	input.Loop = cli_secret_boundary_loop()
	for index := range program.Secrets {
		input.Secret_Buffers[index] = make([]byte, cli.SECRET_BUFFER_BYTES_MAX)
		input.Secret_Path_Failures[index] = make(
			[]cli.Path_Failure, len(program.Secrets[index].Paths),
		)
	}
	var parser cli.Parser
	var result cli.Parse_Result
	var complete cli.Parser_Complete
	var message string
	testify.Zero_Allocation(t, func() {
		parser = cli.Parser{}
		cli.Program_Parse(&program, &parser, input)
		result, complete = cli.Parser_Done(&parser)
		if result.Error != nil {
			message = result.Error.Error()
		}
	})
	if !complete {
		t.Fatal("rejected secret parse did not publish")
	}
	if result.Error == nil {
		t.Fatal("rejected secret parse lost its error")
	}
	if message == "" {
		t.Fatal("rejected secret parse lost its diagnostic")
	}
}

// Injected I/O failures must render without heap-backed wrapping.
func Test_Program_Parse_Secret_Injected_Errors_Zero_Allocation(t *testing.T) {
	program := cli.New_Single(cli.New_Single_Input{
		Label: "program", Secrets: []cli.Secret{
			cli.New_Secret[string](cli.New_Secret_Input{
				Paths: []string{"/secrets/VALUE"}, Required: true,
			}),
		},
	})
	stages := [...]cli_secret_injected_stage{
		CLI_SECRET_INJECTED_STATUS,
		CLI_SECRET_INJECTED_OPEN,
		CLI_SECRET_INJECTED_READ,
		CLI_SECRET_INJECTED_CLOSE,
	}
	for _, stage := range stages {
		t.Run(cli_secret_injected_name(stage), func(t *testing.T) {
			input := cli_test_parse_input(program, []string{"program"})
			input.Loop = cli_secret_injected_loop(stage)
			input.Secret_Buffers[0] = make([]byte, cli.SECRET_BUFFER_BYTES_MAX)
			input.Secret_Path_Failures[0] = make([]cli.Path_Failure, 1)
			var parser cli.Parser
			var result cli.Parse_Result
			var complete cli.Parser_Complete
			var message string
			testify.Zero_Allocation(t, func() {
				parser = cli.Parser{}
				cli.Program_Parse(&program, &parser, input)
				result, complete = cli.Parser_Done(&parser)
				if result.Error != nil {
					message = result.Error.Error()
				}
			})
			if !complete {
				t.Fatal("injected secret failure did not publish")
			}
			if result.Error == nil {
				t.Fatal("injected secret failure did not publish")
			}
			if message == "" {
				t.Fatal("injected secret failure lost its diagnostic")
			}
		})
	}
}

type cli_secret_injected_stage uint8

const CLI_SECRET_INJECTED_STATUS cli_secret_injected_stage = cli_secret_injected_stage(len(""))
const CLI_SECRET_INJECTED_OPEN = CLI_SECRET_INJECTED_STATUS + 1
const CLI_SECRET_INJECTED_READ = CLI_SECRET_INJECTED_OPEN + 1
const CLI_SECRET_INJECTED_CLOSE = CLI_SECRET_INJECTED_READ + 1

var cli_secret_injected_error = errors.New("injected secret failure")

type cli_secret_injected_state struct {
	Stage cli_secret_injected_stage
}

func cli_secret_injected_name(stage cli_secret_injected_stage) (name string) {
	switch stage {
	case CLI_SECRET_INJECTED_STATUS:
		return "status"
	case CLI_SECRET_INJECTED_OPEN:
		return "open"
	case CLI_SECRET_INJECTED_READ:
		return "read"
	case CLI_SECRET_INJECTED_CLOSE:
		return "close"
	}
	panic("unknown injected secret stage")
}

func cli_secret_injected_loop(stage cli_secret_injected_stage) (loop nbio.IO) {
	state := &cli_secret_injected_state{Stage: stage}
	loop.Storage.State = unsafe.Pointer(state)
	loop.Storage.Status_Procedure = cli_secret_injected_status_procedure
	loop.Storage.Open_At_Procedure = cli_secret_injected_open_procedure
	loop.Storage.Read_Procedure = cli_secret_injected_read_procedure
	loop.Close_Procedure = cli_secret_injected_close_procedure
	return loop
}

func cli_secret_injected_status_procedure(
	state_pointer unsafe.Pointer, _ string,
) (status nbio.File_Status, operation_err error) {
	state := (*cli_secret_injected_state)(state_pointer)
	if state.Stage == CLI_SECRET_INJECTED_STATUS {
		return status, cli_secret_injected_error
	}
	status.Exists = true
	status.Size = 1
	return status, nil
}

func cli_secret_injected_open_procedure(
	state_pointer unsafe.Pointer, completion *nbio.Completion, _ nbio.File,
	_ string, _ nbio.Open_At_Options, callback nbio.Callback,
) {
	state := (*cli_secret_injected_state)(state_pointer)
	completion.Data = 1
	if state.Stage == CLI_SECRET_INJECTED_OPEN {
		completion.Error = cli_secret_injected_error
	}
	callback(completion)
}

func cli_secret_injected_read_procedure(
	state_pointer unsafe.Pointer, completion *nbio.Completion, _ nbio.File,
	buffer []byte, _ int64, _ time.Duration, callback nbio.Callback,
) {
	state := (*cli_secret_injected_state)(state_pointer)
	buffer[0] = 'x'
	completion.Data = 1
	completion.Error = nil
	if state.Stage == CLI_SECRET_INJECTED_READ {
		completion.Error = cli_secret_injected_error
	}
	callback(completion)
}

func cli_secret_injected_close_procedure(
	state_pointer unsafe.Pointer, completion *nbio.Completion, _ nbio.File,
	callback nbio.Callback,
) {
	state := (*cli_secret_injected_state)(state_pointer)
	completion.Error = nil
	if state.Stage == CLI_SECRET_INJECTED_CLOSE {
		completion.Error = cli_secret_injected_error
	}
	callback(completion)
}

// Completion keeps candidates and rendered bytes in caller-owned fixed storage.
func Test_Completion_Zero_Allocation(t *testing.T) {
	program := cli_completion_program(
		nil, []cli.Option{{Label: "a", Value: "", Enum: []string{"x"}}}, nil,
	)
	words := []string{"program", "-a="}
	var storage [cli.CANDIDATE_COUNT_MAXIMUM]cli.Candidate
	var candidates cli.Candidates
	testify.Zero_Allocation(t, func() {
		candidates = cli.Complete(program, words, storage[:])
	})
	if len(candidates) != 1 {
		t.Fatalf("allocation completion count = %d, want 1", len(candidates))
	}
	candidate := candidates[0]
	var equal cli.Boolean
	testify.Zero_Allocation(t, func() {
		equal = cli.Candidate_Equal(candidate, "-a=x")
	})
	if !equal {
		t.Fatal("allocation candidate comparison discarded its result")
	}
	var output cli.Output
	testify.Zero_Allocation(t, func() {
		cli.Output_Reset(&output)
		cli.Candidate_Write(&output, candidate)
	})
	if string(cli.Output_Bytes(&output)) != "-a=x" {
		t.Fatalf("allocation candidate output = %q", cli.Output_Bytes(&output))
	}
	var script_error error
	testify.Zero_Allocation(t, func() {
		cli.Output_Reset(&output)
		script_error = cli.Completion_Script(program, "fish", &output)
	})
	if script_error != nil {
		t.Fatalf("allocation completion script: %v", script_error)
	}
	args := []string{"program", "__complete", "program", "-a="}
	var handled cli.Boolean
	testify.Zero_Allocation(t, func() {
		cli.Output_Reset(&output)
		handled = cli.Handle_Completion(program, args, &output, storage[:])
	})
	if !handled {
		t.Fatal("allocation completion request was not handled")
	}
}

// Public output operations share one fixed caller buffer through every rendering form.
func Test_Output_Operations_Zero_Allocation(t *testing.T) {
	flag := cli.New_Flag(cli.New_Flag_Input[string]{Label: "name", Value: "value"})
	string_members := []string{"x"}
	integer_members := []int{1}
	program := cli.New_Single(cli.New_Single_Input{
		Label: "program", Flags: []cli.Option{flag},
		Environment_Variables: []cli.Environment_Variable{
			cli.New_String_Enum_Environment_Variable(
				cli.New_String_Enum_Environment_Variable_Input{
					Key: "ENV", Value: "x", Enum: string_members,
				}),
		},
		Secrets: []cli.Secret{
			cli.New_Integer_Enum_Secret(cli.New_Integer_Enum_Secret_Input{
				Paths: []string{"/SECRET"}, Enum: integer_members,
			}),
		},
	})
	command := program.Selection[0].Single_Commands[0]
	command.Deprecation_Warnings = []cli.Warning{{
		Guidance: cli.Warning_Guidance{"deprecated"},
	}}
	source := []byte("x")
	var output cli.Output
	var bytes strings.Bytes
	var written int
	var write_err error
	operations := [...]func(){
		func() { cli.Output_Reset(&output) },
		func() {
			cli.Output_Reset(&output)
			cli.Output_Write_Text(&output, "x")
		},
		func() {
			cli.Output_Reset(&output)
			written, write_err = output.Write(source)
		},
		func() { bytes = cli.Output_Bytes(&output) },
		func() {
			cli.Output_Reset(&output)
			cli.Print_Help(&output, program)
		},
		func() {
			cli.Output_Reset(&output)
			cli.Print_Command(&output, program, command)
		},
		func() {
			cli.Output_Reset(&output)
			cli.Print_Requested_Help(&output, program, command)
		},
		func() {
			cli.Output_Reset(&output)
			cli.Print_Deprecations(&output, command)
		},
	}
	for _, operation := range operations {
		testify.Zero_Allocation(t, operation)
	}
	if write_err != nil {
		t.Fatal("output operation result was discarded")
	}
	if written == 0 {
		t.Fatal("output operation result was discarded")
	}
	if bytes == nil {
		t.Fatal("output operation result was discarded")
	}
}

// Lookup and typed views return borrowed scalar state without rebuilding legacy interfaces.
func Test_Lookup_Operations_Zero_Allocation(t *testing.T) {
	options := []cli.Option{
		cli.New_Flag(cli.New_Flag_Input[string]{Label: "s", Value: "x"}),
		cli.New_Flag(cli.New_Flag_Input[int]{Label: "i", Value: 1}),
		cli.New_Flag(cli.New_Flag_Input[bool]{Label: "b", Value: true}),
		cli.New_Variadic[string](cli.New_Variadic_Input{Label: "ss"}),
		cli.New_Variadic[int](cli.New_Variadic_Input{Label: "ii"}),
	}
	environment := cli.New_Environment_Variable(
		cli.New_Environment_Variable_Input[string]{Key: "ENV", Value: "x"},
	)
	secret := cli.New_Secret[string](cli.New_Secret_Input{Paths: []string{"/SECRET"}})
	environment_values := cli.Resolved_Environment{environment}
	secret_values := cli.Resolved_Secrets{secret}
	var option cli.Option
	var variable cli.Environment_Variable
	var found_secret cli.Secret
	var text cli.Value_Text
	var integer cli.Integer
	var boolean cli.Boolean
	var texts cli.String_Values
	var integers cli.Integer_Values
	var external_text cli.External_Value_Text
	var secret_text cli.Secret_Value_Bytes
	operations := [...]func(){
		func() { option = cli.Get_Option(options, "s") },
		func() { text = cli.Option_String(options[0]) },
		func() { integer = cli.Option_Integer(options[1]) },
		func() { boolean = cli.Option_Boolean(options[2]) },
		func() { texts = cli.Option_Strings(options[3]) },
		func() { integers = cli.Option_Integers(options[4]) },
		func() { variable = cli.Get_Environment(environment_values, "ENV") },
		func() { external_text = cli.Environment_String(environment) },
		func() { found_secret = cli.Get_Secret(secret_values, "SECRET") },
		func() { secret_text = cli.Secret_String(secret) },
	}
	for _, operation := range operations {
		testify.Zero_Allocation(t, operation)
	}
	if option.Label == "" {
		t.Fatal("lookup result was discarded")
	}
	if variable.Key == "" {
		t.Fatal("lookup result was discarded")
	}
	if found_secret.Key == "" {
		t.Fatal("lookup result was discarded")
	}
	if text == "" {
		t.Fatal("text accessor result was discarded")
	}
	if integer == 0 {
		t.Fatal("integer accessor result was discarded")
	}
	if !boolean {
		t.Fatal("Boolean accessor result was discarded")
	}
	if len(texts) != 0 {
		t.Fatal("text collection accessor changed empty state")
	}
	if len(integers) != 0 {
		t.Fatal("integer collection accessor changed empty state")
	}
	if external_text == "" {
		t.Fatal("environment accessor result was discarded")
	}
	if len(secret_text) != 0 {
		t.Fatal("secret accessor changed empty state")
	}
}

// Each external scalar arm remains direct after declaration lookup and parse publication.
func Test_External_Accessors_Zero_Allocation(t *testing.T) {
	integer_environment := cli.New_Environment_Variable(
		cli.New_Environment_Variable_Input[int]{Key: "INTEGER", Value: 1},
	)
	boolean_environment := cli.New_Environment_Variable(
		cli.New_Environment_Variable_Input[bool]{Key: "BOOLEAN", Value: true},
	)
	integer_secret := cli.Secret{
		Key: "INTEGER", Type: cli.External_Type_State{
			cli.Scalar_Option_Type(cli.OPTION_TYPE_INTEGER),
		}, State: cli.External_State{{Integer: 1}},
	}
	boolean_secret := cli.Secret{
		Key: "BOOLEAN", Type: cli.External_Type_State{
			cli.Scalar_Option_Type(cli.OPTION_TYPE_BOOLEAN),
		}, State: cli.External_State{{Boolean: true}},
	}
	var integer cli.Integer
	var boolean cli.Boolean
	operations := [...]func(){
		func() { integer = cli.Environment_Integer(integer_environment) },
		func() { boolean = cli.Environment_Boolean(boolean_environment) },
		func() { integer = cli.Secret_Integer(integer_secret) },
		func() { boolean = cli.Secret_Boolean(boolean_secret) },
	}
	for _, operation := range operations {
		testify.Zero_Allocation(t, operation)
	}
	if integer == 0 {
		t.Fatal("external integer result was discarded")
	}
	if !boolean {
		t.Fatal("external Boolean result was discarded")
	}
}

// Terminal parser observation returns embedded result state without interface work.
func Test_Parser_Done_Zero_Allocation(t *testing.T) {
	parser := cli.Parser{Publication: cli.Publication{
		Completion: cli.Parser_Completion{true},
	}}
	var result cli.Parse_Result
	var complete cli.Parser_Complete
	testify.Zero_Allocation(t, func() {
		result, complete = cli.Parser_Done(&parser)
	})
	if !complete {
		t.Fatal("parser completion result was discarded")
	}
	if result.Error != nil {
		t.Fatal("zero parser gained an error")
	}
}

// Typed option accessors keep every scalar and caller-owned collection boundary direct.
func Test_Option_Accessor_Bounds(t *testing.T) {
	counts := [...]int{0, 1, 2, slices.COUNT_MAXIMUM}
	for _, count := range counts {
		label_size := count
		if label_size > cli.OPTION_LABEL_SIZE_MAXIMUM {
			label_size = cli.OPTION_LABEL_SIZE_MAXIMUM
		}
		option := cli.Option{
			Label:       cli.Option_Label(cli_boundary_text(label_size, 'l')),
			Description: cli.Description(cli_boundary_text(count, 'd')),
			Type:        cli.Option_Type_State{cli.OPTION_TYPE_STRING},
			State: cli.Option_State{{String: cli.Value_Text(
				cli_boundary_text(count, 'v'),
			)}},
		}
		cli.Option_String(option)
		option.Type[0] = cli.OPTION_TYPE_BOOLEAN
		option.State[0].Boolean = cli.Boolean(count != 0)
		cli.Option_Boolean(option)
		option.Type[0] = cli.OPTION_TYPE_STRINGS
		option.State[0].Strings = make([]string, count)
		cli.Option_Strings(option)
		option.Type[0] = cli.OPTION_TYPE_INTEGERS
		option.State[0].Integers = make([]int, count)
		cli.Option_Integers(option)
		option.Type[0] = cli.OPTION_TYPE_INTEGER
		cli.Option_Integer(option)
	}
	for _, value := range [...]int{bits.INTEGER_MINIMUM, bits.INTEGER_MAXIMUM} {
		cli.Option_Integer(cli.Option{
			Type:  cli.Option_Type_State{cli.OPTION_TYPE_INTEGER},
			State: cli.Option_State{{Integer: cli.Integer(value)}},
		})
	}
	cli_option_accessor_type_bounds(t)
}

func cli_option_accessor_type_bounds(t *testing.T) {
	t.Helper()
	options := [...]cli.Option{
		{Type: cli.Option_Type_State{cli.OPTION_TYPE_STRING}},
		{Type: cli.Option_Type_State{cli.OPTION_TYPE_INTEGER}},
		{Type: cli.Option_Type_State{cli.OPTION_TYPE_BOOLEAN}},
		{Type: cli.Option_Type_State{cli.OPTION_TYPE_STRINGS}},
		{Type: cli.Option_Type_State{cli.OPTION_TYPE_INTEGERS}},
	}
	accessors := [...]func(cli.Option){
		func(option cli.Option) { cli.Option_String(option) },
		func(option cli.Option) { cli.Option_Integer(option) },
		func(option cli.Option) { cli.Option_Boolean(option) },
		func(option cli.Option) { cli.Option_Strings(option) },
		func(option cli.Option) { cli.Option_Integers(option) },
	}
	for accessor_index, accessor := range accessors {
		for option_index, option := range options {
			if accessor_index == option_index {
				accessor(option)
				continue
			}
			assert_panics(t, "typed option accessor mismatch", func() {
				accessor(option)
			})
		}
	}
}

// Benchmark_Program_Parse keeps byte and allocation evidence beside hard allocation checks.
func Benchmark_Program_Parse(b *testing.B) {
	program := cli.New_Single(cli.New_Single_Input{Label: "program"})
	var parser cli.Parser
	var global_flags [cli.HELP_FLAG_COUNT]cli.Option
	var filled [cli.HELP_FLAG_COUNT]bool
	var positionals [ALLOCATION_PROGRAM_TOKEN_COUNT]cli.Indexed_Token
	var slice_named [ALLOCATION_PROGRAM_TOKEN_COUNT]cli.Indexed_Token
	input := cli.Program_Parse_Input{
		Arguments:    []string{"program"},
		Global_Flags: global_flags[:],
		Filled:       filled[:],
		Positionals:  positionals[:],
		Slice_Named:  slice_named[:],
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		cli.Program_Parse(&program, &parser, input)
	}
}
