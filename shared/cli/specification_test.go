package cli_test

import (
	"errors"
	"testing"

	"local/james-orcales/shared/cli"
	"local/james-orcales/shared/filepath"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/nbio"
	"local/james-orcales/shared/sim/time"
	"local/james-orcales/shared/slices"
	"local/james-orcales/shared/strconv"
	"local/james-orcales/shared/strings"
	"local/james-orcales/shared/testify"
)

// Test_Parse_Commands verifies named, default, and unknown command resolution.
func Test_Parse_Commands(t *testing.T) {
	fixture := new_cli_fixture()
	command, err := parse_program(&fixture.Program, []string{"todoctl", "list"})
	if err != "" {
		t.Fatalf("unexpected error: %v", err)
	}
	if command.Label != "list" {
		t.Errorf("expected list, got %q", command.Label)
	}

	command, err = parse_program(&fixture.Program, []string{"todoctl"})
	if err != "" {
		t.Fatalf("unexpected error: %v", err)
	}
	if command.Label != "help" {
		t.Errorf("expected help, got %q", command.Label)
	}

	_, err = parse_program(&fixture.Program, []string{"todoctl", "bogus"})
	if err == "" {
		t.Error("expected error for unknown command")
	}

	// A near-miss command yields a suggestion; a wild miss does not.
	_, err = parse_program(&fixture.Program, []string{"todoctl", "lst"})
	if err == "" {
		t.Fatal("expected an error for unknown command lst")
	}
	if !text_contains(err, `did you mean "list"`) {
		t.Errorf("expected a suggestion of list, got %v", err)
	}
	_, err = parse_program(&fixture.Program, []string{"todoctl", "zzzzzzzz"})
	if err == "" {
		t.Fatal("expected an error for unknown command zzzzzzzz")
	}
	if text_contains(err, "did you mean") {
		t.Errorf("expected no suggestion for a wild miss, got %v", err)
	}
}

// Test_Parse_Single_Command verifies a single-command program reads its positionals
// and flags directly after the program name, with no command selector in slot 1.
func Test_Parse_Single_Command(t *testing.T) {
	program := new_single_fixture()

	command, err := parse_program(&program, []string{"sloc", "./src"})
	if err != "" {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Option_String(cli.Resolved_Options(command.Arguments), "path") != "./src" {
		t.Errorf("expected path ./src, got %q",
			cli.Option_String(cli.Resolved_Options(command.Arguments), "path"))
	}

	command, err = parse_program(&program, []string{"sloc", "./src", "-hidden"})
	if err != "" {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cli.Option_Boolean(cli.Resolved_Options(command.Flags), "hidden") {
		t.Error("expected hidden true")
	}

	// A token that would select a sibling command in a multi-command program is just
	// a positional here: a single-command program has no selector namespace.
	command, err = parse_program(&program, []string{"sloc", "help"})
	if err != "" {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Option_String(cli.Resolved_Options(command.Arguments), "path") != "help" {
		t.Errorf("expected path help, got %q",
			cli.Option_String(cli.Resolved_Options(command.Arguments), "path"))
	}

	// The exact argument-count rule still applies; single-command mode only changes
	// where positionals start, not their arity.
	_, err = parse_program(&program, []string{"sloc"})
	if err == "" {
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
	if err != "" {
		t.Fatalf("unexpected error: %v", err)
	}
	if command.Label != "add" {
		t.Errorf("expected add, got %q", command.Label)
	}
	if cli.Option_String(cli.Resolved_Options(command.Arguments), "task") != "milk" {
		t.Errorf("expected task milk, got %v",
			cli.Option_String(cli.Resolved_Options(command.Arguments), "task"))
	}

	// An unknown binary name suggests the closest command.
	_, err = parse_program(&program, []string{"ad"})
	if err == "" {
		t.Fatal("expected an error for the unknown multicall name ad")
	}
	if !text_contains(err, `did you mean "add"`) {
		t.Errorf("expected a suggestion of add, got %v", err)
	}
}

// Test_Parse_Arguments verifies the argument count and integer conversion.
func Test_Parse_Arguments(t *testing.T) {
	fixture := new_cli_fixture()
	_, err := parse_program(&fixture.Program, []string{"todoctl", "add"})
	if err == "" {
		t.Error("expected error for missing argument")
	}

	command, err := parse_program(&fixture.Program, []string{"todoctl", "delete", "3"})
	if err != "" {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Option_Integer(cli.Resolved_Options(command.Arguments), "id") != 3 {
		t.Error("expected id 3")
	}

	_, err = parse_program(&fixture.Program, []string{"todoctl", "delete", "abc"})
	if err == "" {
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
	if err != "" {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Option_String(cli.Resolved_Options(command.Arguments), "task") != "hello" {
		t.Errorf("expected task hello, got %v",
			cli.Option_String(cli.Resolved_Options(command.Arguments), "task"))
	}

	// Named and positional tokens may appear in any order.
	command, err = parse_program(&fixture.Program,
		[]string{"todoctl", "add", "-priority=high", "world"})
	if err != "" {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Option_String(cli.Resolved_Options(command.Arguments), "task") != "world" {
		t.Error("expected task world from a positional after a flag")
	}
	if cli.Option_String(cli.Resolved_Options(command.Flags), "priority") != "high" {
		t.Error("expected priority high")
	}

	// A positional skips an argument already set by name and fills the next free one.
	pair := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "pair", Description: "two values",
		Arguments: []cli.Option{
			cli.New_Option(cli.New_Option_Input{Label: "first"}),
			cli.New_Option(cli.New_Option_Input{Label: "second"}),
		},
	})
	command, err = parse_program(&pair, []string{"pair", "-first=x", "y"})
	if err != "" {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Option_String(cli.Resolved_Options(command.Arguments), "first") != "x" {
		t.Error("expected first x")
	}
	if cli.Option_String(cli.Resolved_Options(command.Arguments), "second") != "y" {
		t.Error("expected second y")
	}

	// Setting a scalar option twice, or naming an unknown option, is an error.
	_, err = parse_program(&fixture.Program,
		[]string{"todoctl", "add", "task", "-priority=high", "-priority=low"})
	if err == "" {
		t.Error("expected error for a scalar set more than once")
	}
	_, err = parse_program(&fixture.Program, []string{"todoctl", "add", "task", "-zzz=1"})
	if err == "" {
		t.Error("expected error for an unknown option")
	}

	// A near-miss option name yields a suggestion.
	_, err = parse_program(&fixture.Program,
		[]string{"todoctl", "add", "task", "-priorty=high"})
	if err == "" {
		t.Fatal("expected an error for unknown option -priorty")
	}
	if !text_contains(err, "did you mean -priority") {
		t.Errorf("expected a suggestion of -priority, got %v", err)
	}
}

// Test_Parse_Variadic verifies a slice argument collects trailing positionals, accepts
// repeated -label=value that append, and merges both kinds in token order.
func Test_Parse_Variadic(t *testing.T) {
	single := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "sloc", Description: "count lines of code",
		Arguments: []cli.Option{
			cli.New_Option(cli.New_Option_Input{
				Label: "path", Type: cli.OPTION_TYPE_STRINGS,
			}),
		},
		Flags: []cli.Option{cli.New_Option(cli.New_Option_Input{
			Label: "hidden", Type: cli.OPTION_TYPE_BOOLEAN, Is_Flag: true,
		})},
	})
	assert_variadic(
		t, single, []string{"sloc", "a", "b", "c"}, "path", "a", "b", "c",
	)
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
				cli.New_Option(cli.New_Option_Input{Label: "dest"}),
				cli.New_Option(cli.New_Option_Input{
					Label: "source", Type: cli.OPTION_TYPE_STRINGS,
				}),
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
	if err == "" {
		t.Error("expected error for the missing scalar argument")
	}

	cli_parse_variadic_integers(t)
}

// Test_Parse_Flags verifies flag assignment and the flag error cases.
func Test_Parse_Flags(t *testing.T) {
	fixture := new_cli_fixture()
	command, err := parse_program(&fixture.Program,
		[]string{"todoctl", "add", "task", "-priority=high"})
	if err != "" {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Option_String(cli.Resolved_Options(command.Flags), "priority") != "high" {
		t.Error("expected priority high")
	}

	_, err = parse_program(&fixture.Program,
		[]string{"todoctl", "add", "task", "-bogus=1"})
	if err == "" {
		t.Error("expected error for unknown flag")
	}

	_, err = parse_program(&fixture.Program,
		[]string{"todoctl", "add", "task", "--priority=high"})
	if err == "" {
		t.Error("expected error for double-dash flag")
	}

	_, err = parse_program(&fixture.Program,
		[]string{"todoctl", "add", "task", "-priority"})
	if err == "" {
		t.Error("expected error for non-boolean flag without value")
	}
}

// Test_Parse_Enum verifies a flag-form enum: a permitted value is accepted, an omitted
// enum falls back to its default, and an out-of-set value is rejected — with a
// levenshtein suggestion for a near miss and the full allowed list otherwise. The int
// instantiation rejects a non-member with the same allowed-list message.
func Test_Parse_Enum(t *testing.T) {
	program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "prog", Description: "enum flags",
		Flags: []cli.Option{
			cli.New_Option(cli.New_Option_Input{
				Label: "color", String_Enum: []string{"auto", "never", "always"},
				String: "auto", Is_Flag: true, Description: "when to colorize",
			}),
			cli.New_Option(cli.New_Option_Input{
				Label: "level", Integer: 1, Integer_Enum: []int{1, 2, 4, 8},
				Type: cli.OPTION_TYPE_INTEGER, Is_Flag: true,
				Description: "compression level",
			}),
		},
	})
	// A permitted value is accepted.
	command, err := parse_program(&program, []string{"prog", "-color=never"})
	if err != "" {
		t.Fatalf("unexpected error: %v", err)
	}
	color := cli.Option_String(cli.Resolved_Options(command.Flags), "color")
	if color != "never" {
		t.Errorf("expected never, got %q", color)
	}

	// An omitted enum keeps its default.
	command, err = parse_program(&program, []string{"prog"})
	if err != "" {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Option_String(cli.Resolved_Options(command.Flags), "color") != "auto" {
		t.Errorf("expected default auto, got %q",
			cli.Option_String(cli.Resolved_Options(command.Flags), "color"))
	}

	// A near miss suggests the closest member.
	_, err = parse_program(&program, []string{"prog", "-color=nevr"})
	if err == "" {
		t.Fatal("expected an error for an out-of-set value")
	}
	if !text_contains(err, `did you mean "never"`) {
		t.Errorf("expected a suggestion of never, got %v", err)
	}

	// A wild miss lists the whole set instead of guessing.
	_, err = parse_program(&program, []string{"prog", "-color=purple"})
	if err == "" {
		t.Fatal("expected an error for an out-of-set value")
	}
	if !text_contains(err, "allowed: auto, never, always") {
		t.Errorf("expected the allowed list, got %v", err)
	}

	// An int enum accepts a member and rejects a non-member with the allowed list.
	command, err = parse_program(&program, []string{"prog", "-level=4"})
	if err != "" {
		t.Fatalf("unexpected error: %v", err)
	}
	level := cli.Option_Integer(cli.Resolved_Options(command.Flags), "level")
	if level != 4 {
		t.Errorf("expected 4, got %v", level)
	}
	_, err = parse_program(&program, []string{"prog", "-level=3"})
	if err == "" {
		t.Fatal("expected an error for an out-of-set int value")
	}
	if !text_contains(err, "allowed: 1, 2, 4, 8") {
		t.Errorf("expected the allowed int list, got %v", err)
	}
}

// Test_Parse_Help verifies that -h and -help select help before required argument checks.
func Test_Parse_Help(t *testing.T) {
	single := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "tool", Description: "does a thing",
		Arguments: []cli.Option{
			cli.New_Option(cli.New_Option_Input{Label: "target"}),
		},
	})
	_, err := parse_program(&single, []string{"tool", "-help"})
	testify.True(t, bool(err == cli.HELP_REQUESTED), "-help")
	short_context, err := parse_program(&single, []string{"tool", "-h"})
	testify.True(t, bool(err == cli.HELP_REQUESTED), "-h")
	short_help := new_output_buffer()
	cli.Print_Requested_Help(output_cli(&short_help), single, short_context.Label)
	testify.Contains_Any(
		t, output_text(&short_help), "does a thing", "-h program help",
	)

	// Multi-command: a command then -help resolves that command as the context.
	fixture := new_cli_fixture()
	command, err := parse_program(&fixture.Program, []string{"todoctl", "list", "-help"})
	testify.True(
		t, bool(err == cli.HELP_REQUESTED), "command help",
	)
	testify.Equal(t, "list", string(command.Label), "command help context")

	// Multi-command with -help but no command selected → root context (empty label).
	fixture = new_cli_fixture()
	command, err = parse_program(&fixture.Program, []string{"todoctl", "-help"})
	testify.True(t, bool(err == cli.HELP_REQUESTED), "root help")
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
	output := new_output_buffer()
	args := []string{"toolbox", "__complete", "add", "-"}
	testify.True(
		t, handle_completion(program, args, output_cli(&output)), "__complete",
	)
	testify.Contains_Any(t, output_text(&output), "-task", "completion output")
}

// Test_Visibility_Hidden verifies a hidden flag and a hidden command still parse and
// resolve, yet appear in neither help output nor completion candidates.
func Test_Visibility_Hidden(t *testing.T) {
	program := cli.New(cli.New_Input{
		Label: "tool", Description: "a tool",
		Commands: []cli.Command{
			{Label: "run", Description: "run it", Flags: []cli.Option{
				cli.New_Option(cli.New_Option_Input{
					Label: "verbose", Type: cli.OPTION_TYPE_BOOLEAN,
					Is_Flag: true,
				}),
				cli.New_Option(cli.New_Option_Input{
					Label: "secret", Type: cli.OPTION_TYPE_BOOLEAN,
					Is_Flag: true, Hidden: true,
				}),
			}},
			{Label: "ghost", Description: "internal command", Hidden: true},
		},
	})

	// A hidden flag still parses, and a hidden command still resolves.
	command, err := parse_program(&program, []string{"tool", "run", "-secret"})
	if err != "" {
		t.Fatalf("hidden flag should parse: %v", err)
	}
	if !cli.Option_Boolean(cli.Resolved_Options(command.Flags), "secret") {
		t.Error("expected secret true")
	}
	if _, err = parse_program(
		&program, []string{"tool", "ghost"},
	); err != "" {
		t.Fatalf("hidden command should resolve: %v", err)
	}

	// Help shows the visible names and omits the hidden ones.
	help := new_output_buffer()
	cli.Print_Help(output_cli(&help), program)
	if !text_contains(output_text(&help), "verbose") {
		t.Errorf("help should show a visible flag:\n%s", output_text(&help))
	}
	if text_contains(output_text(&help), "secret") {
		t.Errorf("help must not show a hidden flag:\n%s", output_text(&help))
	}
	if text_contains(output_text(&help), "ghost") {
		t.Errorf("help must not show a hidden command:\n%s", output_text(&help))
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
	program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "tool", Description: "a tool",
		Flags: []cli.Option{
			cli.New_Option(cli.New_Option_Input{Label: "name", Is_Flag: true}),
			cli.New_Option(cli.New_Option_Input{
				Label: "old-name", Is_Flag: true, Deprecated: "use -name",
			}),
		},
	})

	// It still parses, and using it records a warning.
	command, err := parse_program(&program, []string{"tool", "-old-name=ada"})
	if err != "" {
		t.Fatalf("deprecated flag should parse: %v", err)
	}
	if cli.Option_String(cli.Resolved_Options(command.Flags), "old-name") != "ada" {
		t.Error("expected old-name ada")
	}
	warnings := new_output_buffer()
	cli.Print_Deprecations(output_cli(&warnings), command.Deprecation_Warnings)
	if !text_contains(output_text(&warnings), "use -name") {
		t.Errorf(
			"expected a deprecation warning, got %q", output_text(&warnings),
		)
	}

	// Not using it records nothing.
	quiet, _ := parse_program(&program, []string{"tool", "-name=bob"})
	silence := new_output_buffer()
	cli.Print_Deprecations(output_cli(&silence), quiet.Deprecation_Warnings)
	if output_text(&silence) != "" {
		t.Errorf("expected no warning when unused, got %q", output_text(&silence))
	}

	// Help omits it.
	help := new_output_buffer()
	cli.Print_Help(output_cli(&help), program)
	if text_contains(output_text(&help), "old-name") {
		t.Errorf("help must not show a deprecated flag:\n%s", output_text(&help))
	}
}

// Test_Trim_Quotes_Cases verifies quoted flag values are unquoted by quote style.
func Test_Trim_Quotes_Cases(t *testing.T) {
	program := cli.New(cli.New_Input{
		Label:       "prog",
		Description: "test program",
		Commands: []cli.Command{{
			Label: "add",
			Arguments: []cli.Option{cli.New_Option(cli.New_Option_Input{
				Label: "task",
			})},
			Flags: []cli.Option{cli.New_Option(cli.New_Option_Input{
				Label: "flag", Is_Flag: true,
			})},
		}},
	})
	check_case := func(name, raw, want string) {
		t.Helper()
		command, err := parse_program(&program, []string{"prog", "add", "task", raw})
		if err != "" {
			t.Fatalf("%s: parse failed: %v", name, err)
		}
		got := string(cli.Option_String(cli.Resolved_Options(command.Flags), "flag"))
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
	options := cli.Resolved_Options{
		{
			Label: "a", Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_STRING},
			State: cli.Resolved_Option_State{String: "x"},
		},
		{
			Label: "b", Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_STRING},
			State: cli.Resolved_Option_State{String: "y"},
		},
	}
	if cli.Option_String(options, "b") != "y" {
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
		cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
			Label: "sloc",
			Flags: []cli.Option{{
				Label: "no_ignore",
				Type:  cli.Option_Type_State{Value: cli.OPTION_TYPE_BOOLEAN},
				State: cli.Option_State{Is_Flag: true},
			}},
		})
	})
	// An argument label must be flag-safe now that arguments are settable by name.
	assert_panics(t, "argument with an underscore", func() {
		cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
			Label:     "sloc",
			Arguments: []cli.Option{{Label: "bad_label"}},
		})
	})
	// An argument label may not collide with a flag in the -key=value namespace.
	assert_panics(t, "argument colliding with a flag", func() {
		cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
			Label:     "sloc",
			Arguments: []cli.Option{{Label: "dup"}},
			Flags: []cli.Option{{
				Label: "dup",
				Type:  cli.Option_Type_State{Value: cli.OPTION_TYPE_BOOLEAN},
				State: cli.Option_State{Is_Flag: true},
			}},
		})
	})
	// A slice argument must be the last argument.
	assert_panics(t, "non-terminal slice argument", func() {
		cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
			Label: "x",
			Arguments: []cli.Option{
				cli.New_Option(cli.New_Option_Input{
					Label: "a", Type: cli.OPTION_TYPE_STRINGS,
				}),
				cli.New_Option(cli.New_Option_Input{Label: "b"}),
			},
		})
	})
	assert_new_enum_validation(t)
	assert_external_validation(t)
}

// Integer conversion must preserve every element boundary inside caller storage.
func cli_parse_variadic_integers(t *testing.T) {
	t.Helper()
	numbers := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "sum", Description: "add numbers",
		Arguments: []cli.Option{
			cli.New_Option(cli.New_Option_Input{
				Label: "n", Type: cli.OPTION_TYPE_INTEGERS,
			}),
		},
	})
	command, err := parse_program(&numbers, []string{"sum", "1", "2", "3"})
	if err != "" {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(
		cli.Option_Integers(cli.Resolved_Options(command.Arguments), "n"),
		[]int{1, 2, 3},
	) {
		t.Errorf(
			"expected [1 2 3], got %v",
			cli.Option_Integers(cli.Resolved_Options(command.Arguments), "n"),
		)
	}
	_, err = parse_program(&numbers, []string{"sum", "1", "x"})
	if err == "" {
		t.Error("expected error for a non-numeric slice element")
	}
}

// Test_Parse_Multicall_Self_Invocation verifies that a multicall binary run by its own
// name — not a verb link — selects the command from the first token, as `busybox ls`
// does. This is what lets a bootstrap verb run before the links exist.
func Test_Parse_Multicall_Self_Invocation(t *testing.T) {
	program := new_multicall_fixture()
	command, err := parse_program(&program, []string{"toolbox", "add", "milk"})
	if err != "" {
		t.Fatalf("unexpected error: %v", err)
	}
	if command.Label != "add" {
		t.Errorf("expected add, got %q", command.Label)
	}
	if cli.Option_String(cli.Resolved_Options(command.Arguments), "task") != "milk" {
		t.Errorf("expected task milk, got %v",
			cli.Option_String(cli.Resolved_Options(command.Arguments), "task"))
	}
}

// Test_Parse_Enum_Argument verifies an argument-form enum: it is settable by position
// and by name, an omitted value yields the existing missing-required error, and an
// out-of-set value is rejected with the allowed list.
func Test_Parse_Enum_Argument(t *testing.T) {
	program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "prog", Description: "enum argument",
		Arguments: []cli.Option{
			cli.New_Option(cli.New_Option_Input{
				Label: "format", String_Enum: []string{"json", "yaml", "toml"},
				Description: "output format",
			}),
		},
	})

	// Accepted by position.
	command, err := parse_program(&program, []string{"prog", "yaml"})
	if err != "" {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Option_String(cli.Resolved_Options(command.Arguments), "format") != "yaml" {
		t.Errorf("expected yaml, got %q",
			cli.Option_String(cli.Resolved_Options(command.Arguments), "format"))
	}

	// Accepted by name.
	command, err = parse_program(&program, []string{"prog", "-format=toml"})
	if err != "" {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Option_String(cli.Resolved_Options(command.Arguments), "format") != "toml" {
		t.Errorf("expected toml, got %q",
			cli.Option_String(cli.Resolved_Options(command.Arguments), "format"))
	}

	// Omitted → the existing missing-required-argument error.
	_, err = parse_program(&program, []string{"prog"})
	if err == "" {
		t.Fatal("expected an error for a missing required enum argument")
	}
	if !text_contains(err, "missing required argument") {
		t.Errorf("expected the missing-required message, got %v", err)
	}

	// A non-member is rejected with the allowed list.
	_, err = parse_program(&program, []string{"prog", "xml"})
	if err == "" {
		t.Fatal("expected an error for an out-of-set argument")
	}
	if !text_contains(err, "allowed: json, yaml, toml") {
		t.Errorf("expected the allowed list, got %v", err)
	}
}

// Verifies that one injected snapshot supplies typed declarations and
// leaves optional declarations at their public defaults when the source is absent.
func assert_parse_environment_variables(t *testing.T) {
	t.Helper()
	program := external_program([]cli.Environment_Variable{
		cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
			Key: "NAME", Required: true,
		}),
		cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
			Key: "PORT", Type: cli.OPTION_TYPE_INTEGER, Integer: 80,
		}),
		cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
			Key: "ENABLED", Type: cli.OPTION_TYPE_BOOLEAN,
		}),
		cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
			Key: "MODE", String: "safe", String_Enum: []string{"safe", "fast"},
		}),
		cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
			Key: "LEVEL", Type: cli.OPTION_TYPE_INTEGER,
			Integer: 80, Integer_Enum: []int{80, 81},
		}),
		cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
			Key: "EMPTY", String: "fallback", Allow_Empty: true,
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
	if cli.Parse_Error_Present(result.Error) {
		t.Fatalf("parse environment: %v", result.Error)
	}
	assert_environment_values(t, result)
}

func assert_environment_values(t *testing.T, result cli.Parse_Result) {
	t.Helper()
	if cli.Environment_String(result.Command.Environment, "NAME") != "service" {
		t.Fatal("NAME did not resolve")
	}
	if cli.Environment_Integer(result.Command.Environment, "PORT") != 80 {
		t.Fatal("PORT did not keep its default")
	}
	if !cli.Environment_Boolean(result.Command.Environment, "ENABLED") {
		t.Fatal("ENABLED did not use strconv.ParseBool")
	}
	if cli.Environment_String(result.Command.Environment, "MODE") != "fast" {
		t.Fatal("MODE did not resolve its enum member")
	}
	if cli.Environment_Integer(result.Command.Environment, "LEVEL") != 81 {
		t.Fatal("LEVEL did not resolve its enum member")
	}
	if cli.Environment_String(result.Command.Environment, "EMPTY") != "" {
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
		cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
			Key: "FIRST", Required: true,
		}),
		cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
			Key: "SECOND", Type: cli.OPTION_TYPE_INTEGER, Required: true,
		}),
		cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
			Key: "THIRD", Required: true, String_Enum: []string{"yes", "no"},
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
	if !cli.Parse_Error_Present(result.Error) {
		t.Fatal("malformed, duplicate, and invalid entries did not fail")
	}
	message := string(cli.Parse_Error_Bytes(result.Error))
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
			cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
				Key: cli.External_Key(key), Type: cli.OPTION_TYPE_INTEGER,
				Integer: 3, Integer_Enum: []int{3},
			}),
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
			if !cli.Parse_Error_Present(result.Error) {
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
			cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
				Key: cli.External_Key(profile.Key), String: "allowed",
				String_Enum: []string{"allowed"}, Allow_Empty: true,
			}),
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
		if !cli.Parse_Error_Present(result.Error) {
			t.Fatalf("string enum accepted %q outside its set", profile.Raw)
		}
	}
}

// Test_Bounded_External_Boolean_Error keeps the Boolean conversion member observable.
func Test_Bounded_External_Boolean_Error(t *testing.T) {
	program := external_program([]cli.Environment_Variable{
		cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
			Key: "B", Type: cli.OPTION_TYPE_BOOLEAN,
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
	if !cli.Parse_Error_Present(result.Error) {
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
	cli_bounded_secret_enum_members(t)
}

func cli_bounded_secret_enum_members(t *testing.T) {
	t.Helper()
	strings_enum := make([]string, slices.COUNT_MAXIMUM)
	integers := make([]int, slices.COUNT_MAXIMUM)
	for index := range strings_enum {
		strings_enum[index] = "private-non-number"
		integers[index] = 1
	}
	program := external_program(nil, []cli.Secret{
		cli.New_Secret(cli.New_Secret_Input{
			Paths: []string{"/S"}, String_Enum: strings_enum, Required: true,
		}),
		cli.New_Secret(cli.New_Secret_Input{
			Paths: []string{"/secrets/INTEGER_ONE"}, Type: cli.OPTION_TYPE_INTEGER,
			Integer_Enum: integers, Required: true,
		}),
	})
	var parser cli.Parser
	cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
		Arguments: []string{"external"}, Loop: cli_secret_boundary_loop(),
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatalf("maximum secret enum parse incomplete: error=%v", result.Error)
	}
	if cli.Parse_Error_Present(result.Error) {
		t.Fatalf("maximum secret enum parse error: %v", result.Error)
	}
}

func cli_bounded_integer_enum_members(t *testing.T, members []int) {
	t.Helper()
	program := external_program([]cli.Environment_Variable{
		cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
			Key: "I", Type: cli.OPTION_TYPE_INTEGER,
			Integer: 3, Integer_Enum: members,
		}),
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
		cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
			Key: "S", String: "allowed", String_Enum: members,
		}),
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
			cli.New_Secret(cli.New_Secret_Input{
				Paths: []string{secret_path}, Type: cli.OPTION_TYPE_INTEGER,
				Required: true, Allow_Empty: true,
			}),
		})
		var parser cli.Parser
		cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
			Arguments: []string{"external"}, Loop: cli.Secret_IO_Of(loop),
		})
		result := drive_parser(t, driver, &parser)
		if !cli.Parse_Error_Present(result.Error) {
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
		cli.New_Secret(cli.New_Secret_Input{
			Paths: []string{"/first/TOKEN", "/second/TOKEN"}, Required: true,
		}),
		cli.New_Secret(cli.New_Secret_Input{
			Paths: []string{"/secrets/COUNT"}, Type: cli.OPTION_TYPE_INTEGER,
			Required: true,
		}),
		cli.New_Secret(cli.New_Secret_Input{
			Paths: []string{"/secrets/ENABLED"}, Type: cli.OPTION_TYPE_BOOLEAN,
			Required: true,
		}),
		cli.New_Secret(cli.New_Secret_Input{
			Paths: []string{"/secrets/MODE"}, Required: true,
			String_Enum: []string{"safe", "fast"},
		}),
		cli.New_Secret(cli.New_Secret_Input{
			Paths: []string{"/secrets/LEVEL"}, Type: cli.OPTION_TYPE_INTEGER,
			Integer_Enum: []int{80, 81}, Required: true,
		}),
		cli.New_Secret(cli.New_Secret_Input{
			Paths: []string{"/secrets/OPTIONAL"},
		}),
	})
	var parser cli.Parser
	cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
		Arguments: []string{"external"}, Loop: cli.Secret_IO_Of(loop),
	})
	_, done := cli.Parser_Done(&parser)
	if done {
		t.Fatal("a parser with available secrets completed before the loop ran")
	}
	result := drive_parser(t, driver, &parser)
	if cli.Parse_Error_Present(result.Error) {
		t.Fatalf("parse secrets: %v", result.Error)
	}
	if string(cli.Secret_String(result.Command.Secrets, "TOKEN")) != "value" {
		t.Fatal("TOKEN did not use its fallback path or remove CRLF")
	}
	if cli.Secret_Integer(result.Command.Secrets, "COUNT") != 42 {
		t.Fatal("COUNT did not convert")
	}
	if !cli.Secret_Boolean(result.Command.Secrets, "ENABLED") {
		t.Fatal("ENABLED did not convert")
	}
	if string(cli.Secret_String(result.Command.Secrets, "MODE")) != "fast" {
		t.Fatal("MODE did not validate")
	}
	if cli.Secret_Integer(result.Command.Secrets, "LEVEL") != 81 {
		t.Fatal("LEVEL did not validate")
	}
	if len(cli.Secret_String(result.Command.Secrets, "OPTIONAL")) != 0 {
		t.Fatal("an absent optional secret did not expose its zero value")
	}
	nbio.IO_Deinit(loop)
}

// Test_Parse_Secret_Bounds_And_Errors verifies the raw size boundary, regular-file rule, joined
// declaration order, and secret-value redaction.
func Test_Parse_Secret_Bounds_And_Errors(t *testing.T) {
	loop := cli_secret_boundary_loop()
	program := external_program(nil, []cli.Secret{
		cli.New_Secret(cli.New_Secret_Input{
			Paths: []string{"/secrets/BOUNDARY"}, Required: true,
		}),
		cli.New_Secret(cli.New_Secret_Input{
			Paths: []string{"/missing/OVERFLOW", "/secrets/OVERFLOW"}, Required: true,
		}),
		cli.New_Secret(cli.New_Secret_Input{
			Paths: []string{"/secrets/DIRECTORY"}, Required: true,
		}),
		cli.New_Secret(cli.New_Secret_Input{
			Paths: []string{"/secrets/NUMBER"}, Type: cli.OPTION_TYPE_INTEGER,
			Required: true,
		}),
		cli.New_Secret(cli.New_Secret_Input{
			Paths: []string{"/secrets/NEGATIVE"}, Required: true,
		}),
		cli.New_Secret(cli.New_Secret_Input{
			Paths: []string{"/secrets/READ_OVERFLOW"}, Required: true,
		}),
		{
			Key: "BOUNDARY_ENUM", Paths: []string{"/secrets/BOUNDARY_ENUM"},
			Enumeration: cli.External_Enumeration{String: []string{"allowed"}},
			Required:    true,
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
	if !cli.Parse_Error_Present(result.Error) {
		t.Fatal("invalid secret paths did not fail")
	}
	cli_assert_secret_boundary_result(t, result)
}

func cli_assert_secret_boundary_result(t *testing.T, result cli.Parse_Result) {
	t.Helper()
	boundary := cli.Secret_String(result.Command.Secrets, "BOUNDARY")
	if len(boundary) != cli.SECRET_BYTES_MAX {
		t.Fatal("the exact size boundary did not resolve")
	}
	for _, character := range boundary {
		if character != 'x' {
			t.Fatal("the exact size boundary changed content")
		}
	}
	message := string(cli.Parse_Error_Bytes(result.Error))
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
	integer_key := cli_boundary_text(cli.SECRET_KEY_SIZE_MAXIMUM, 'I')
	string_key := cli_boundary_text(cli.SECRET_KEY_SIZE_MAXIMUM, 'S')
	program := external_program(nil, []cli.Secret{
		{
			Key: cli.Secret_Key(integer_key), Paths: []string{"/" + integer_key},
			Type:        cli.External_Type_State{Value: cli.OPTION_TYPE_INTEGER},
			Enumeration: cli.External_Enumeration{Integers: []int{3}}, Required: true,
		},
		{
			Key: cli.Secret_Key(string_key), Paths: []string{"/" + string_key},
			Enumeration: cli.External_Enumeration{String: []string{"allowed"}},
			Required:    true,
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
	if !cli.Parse_Error_Present(result.Error) {
		t.Fatal("full enum keys accepted values outside their sets")
	}
}

// Maximum path reaches narrow production I/O adapter without oversized components.
func Test_Secret_IO_Maximum_Path(t *testing.T) {
	file_path := cli_maximum_secret_path()
	loop, driver := cli_sim_loop(19)
	seed_secret_files(t, loop, driver, []secret_file{
		{Path: file_path, Content: "value"},
	})
	program := external_program(nil, []cli.Secret{
		cli.New_Secret(cli.New_Secret_Input{Paths: []string{file_path}, Required: true}),
	})
	var parser cli.Parser
	cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
		Arguments: []string{"external"}, Loop: cli.Secret_IO_Of(loop),
	})
	result := drive_parser(t, driver, &parser)
	if cli.Parse_Error_Present(result.Error) {
		t.Fatal("maximum secret path failed")
	}
	nbio.IO_Deinit(loop)
}

func cli_maximum_secret_path() (file_path string) {
	file_path = "/"
	byte_budget := filepath.PATH_SIZE_MAXIMUM - len(file_path) - len("X")
	for byte_budget > 0 {
		component_size := byte_budget - len("/")
		if component_size > nbio.SIM_PATH_COMPONENT_BYTES_MAXIMUM {
			component_size = nbio.SIM_PATH_COMPONENT_BYTES_MAXIMUM
		}
		file_path += cli_boundary_text(component_size, 'd') + "/"
		byte_budget -= component_size + len("/")
	}
	return file_path + "X"
}

// Host path bound must reject malicious secret identity before file submission.
func Test_Bounded_Secret_Path_Host_Limit(t *testing.T) {
	oversized_key := cli_boundary_text(cli.SECRET_KEY_SIZE_MAXIMUM+1, 'X')
	assert_panics(t, "secret path exceeds host bound", func() {
		cli.New_Secret(cli.New_Secret_Input{Paths: []string{"/" + oversized_key}})
	})
}

// Test_Bounded_Secret_Conversion_Bytes keeps byte parsing at empty, interior, and full bounds.
func Test_Bounded_Secret_Conversion_Bytes(t *testing.T) {
	maximum := cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'x')
	program := external_program(nil, []cli.Secret{
		{Key: "EMPTY", Paths: []string{"/secrets/EMPTY"},
			Enumeration: cli.External_Enumeration{String: []string{""}},
			Allow_Empty: true},
		{Key: "TWO", Paths: []string{"/secrets/TWO"},
			Enumeration: cli.External_Enumeration{String: []string{"xx"}}},
		{Key: "TEXT_MAX", Paths: []string{"/secrets/TEXT_MAX"},
			Enumeration: cli.External_Enumeration{String: []string{maximum}}},
		{Key: "BOOL_EMPTY", Paths: []string{"/secrets/BOOL_EMPTY"},
			Type:        cli.External_Type_State{Value: cli.OPTION_TYPE_BOOLEAN},
			Allow_Empty: true},
		{Key: "BOOL_TWO", Paths: []string{"/secrets/BOOL_TWO"},
			Type: cli.External_Type_State{Value: cli.OPTION_TYPE_BOOLEAN}},
		{Key: "BOOL_MAX", Paths: []string{"/secrets/BOOL_MAX"},
			Type: cli.External_Type_State{Value: cli.OPTION_TYPE_BOOLEAN}},
		{Key: "NUMBER_TEXT", Paths: []string{"/secrets/NUMBER_TEXT"},
			Type: cli.External_Type_State{Value: cli.OPTION_TYPE_INTEGER}},
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
			Key:      cli.Secret_Key(filepath.Base(filepath.Text(file_path))),
			Paths:    []string{file_path},
			Type:     cli.External_Type_State{Value: cli.OPTION_TYPE_INTEGER},
			Required: true,
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
	if cli.Parse_Error_Present(result.Error) {
		t.Fatalf("integer edge secrets: %v", result.Error)
	}
	for _, secret := range result.Command.Secrets {
		cli.Secret_Integer(result.Command.Secrets, secret.Key)
	}
}

const CLI_SECRET_BOUNDARY_FILE_DEFAULT nbio.File = 1
const CLI_SECRET_BOUNDARY_FILE_NEGATIVE = CLI_SECRET_BOUNDARY_FILE_DEFAULT + 1
const CLI_SECRET_BOUNDARY_FILE_READ_OVERFLOW = CLI_SECRET_BOUNDARY_FILE_NEGATIVE + 1
const CLI_SECRET_BOUNDARY_FILE_INTEGER_MINIMUM = CLI_SECRET_BOUNDARY_FILE_READ_OVERFLOW + 1
const CLI_SECRET_BOUNDARY_FILE_INTEGER_MAXIMUM = CLI_SECRET_BOUNDARY_FILE_INTEGER_MINIMUM + 1
const CLI_SECRET_BOUNDARY_FILE_INTEGER_NEGATIVE_ONE = CLI_SECRET_BOUNDARY_FILE_INTEGER_MAXIMUM + 1
const CLI_SECRET_BOUNDARY_FILE_INTEGER_ONE = CLI_SECRET_BOUNDARY_FILE_INTEGER_NEGATIVE_ONE + 1
const CLI_SECRET_BOUNDARY_FILE_INTEGER_TWO = CLI_SECRET_BOUNDARY_FILE_INTEGER_ONE + 1
const CLI_SECRET_BOUNDARY_FILE_MAXIMUM = CLI_SECRET_BOUNDARY_FILE_INTEGER_TWO + 1
const CLI_SECRET_BOUNDARY_FILE_MAXIMUM_TEXT = CLI_SECRET_BOUNDARY_FILE_MAXIMUM + 1
const CLI_SECRET_BOUNDARY_FILE_EMPTY = CLI_SECRET_BOUNDARY_FILE_MAXIMUM_TEXT + 1
const CLI_SECRET_BOUNDARY_FILE_TWO = CLI_SECRET_BOUNDARY_FILE_EMPTY + 1
const CLI_SECRET_BOUNDARY_FILE_PATH_MAXIMUM = CLI_SECRET_BOUNDARY_FILE_TWO + 1

func cli_secret_boundary_file(path cli.Resolved_Secret_Path) (file nbio.File) {
	file_path := string(path)
	switch file_path {
	case "/secrets/NEGATIVE":
		return CLI_SECRET_BOUNDARY_FILE_NEGATIVE
	case "/secrets/READ_OVERFLOW":
		return CLI_SECRET_BOUNDARY_FILE_READ_OVERFLOW
	case "/secrets/INTEGER_MIN":
		return CLI_SECRET_BOUNDARY_FILE_INTEGER_MINIMUM
	case "/secrets/INTEGER_MAX":
		return CLI_SECRET_BOUNDARY_FILE_INTEGER_MAXIMUM
	case "/secrets/INTEGER_NEGATIVE_ONE":
		return CLI_SECRET_BOUNDARY_FILE_INTEGER_NEGATIVE_ONE
	case "/secrets/INTEGER_ONE":
		return CLI_SECRET_BOUNDARY_FILE_INTEGER_ONE
	case "/secrets/INTEGER_TWO":
		return CLI_SECRET_BOUNDARY_FILE_INTEGER_TWO
	case "/secrets/BOUNDARY", "/secrets/BOOL_MAX", "/secrets/NUMBER",
		"/secrets/BOUNDARY_ENUM":
		return CLI_SECRET_BOUNDARY_FILE_MAXIMUM
	case "/secrets/NUMBER_TEXT", "/secrets/TEXT_MAX":
		return CLI_SECRET_BOUNDARY_FILE_MAXIMUM_TEXT
	case "/secrets/EMPTY", "/secrets/BOOL_EMPTY":
		return CLI_SECRET_BOUNDARY_FILE_EMPTY
	case "/secrets/TWO", "/secrets/BOOL_TWO":
		return CLI_SECRET_BOUNDARY_FILE_TWO
	}
	if len(file_path) == filepath.PATH_SIZE_MAXIMUM {
		return CLI_SECRET_BOUNDARY_FILE_PATH_MAXIMUM
	}
	return CLI_SECRET_BOUNDARY_FILE_DEFAULT
}

func cli_secret_boundary_loop() (loop cli.Secret_IO) {
	loop.Status_Procedure = cli_secret_boundary_status
	loop.Open_Procedure = cli_secret_boundary_open
	loop.Read_Procedure = cli_secret_boundary_read
	loop.Close_Procedure = cli_secret_boundary_close
	return loop
}

func cli_secret_boundary_status(
	_ nbio.IO, path cli.Resolved_Secret_Path,
) (status nbio.File_Status, operation_err error) {
	file_path := string(path)
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
	if len(file_path) == filepath.PATH_SIZE_MAXIMUM {
		status.Size = int64(len("4"))
	}
	return status, nil
}

func cli_secret_boundary_open(
	_ nbio.IO, completion nbio.Completion_Handle,
	path cli.Resolved_Secret_Path, callback nbio.Callback,
) {
	completion.Data = int(cli_secret_boundary_file(path))
	completion.Error = nil
	callback(completion)
}

func cli_secret_boundary_read(
	_ nbio.IO, completion nbio.Completion_Handle, file nbio.File,
	buffer cli.Secret_Buffer, callback nbio.Callback,
) {
	if file == CLI_SECRET_BOUNDARY_FILE_NEGATIVE {
		completion.Data = cli.SECRET_READ_COUNT_MINIMUM
		completion.Error = nil
		callback(completion)
		return
	}
	if file == CLI_SECRET_BOUNDARY_FILE_READ_OVERFLOW {
		completion.Data = cli.SECRET_READ_COUNT_MAXIMUM
		completion.Error = nil
		callback(completion)
		return
	}
	integer_text := ""
	switch file {
	case CLI_SECRET_BOUNDARY_FILE_INTEGER_MINIMUM:
		integer_text = "-9223372036854775808"
	case CLI_SECRET_BOUNDARY_FILE_INTEGER_MAXIMUM:
		integer_text = "9223372036854775807"
	case CLI_SECRET_BOUNDARY_FILE_INTEGER_NEGATIVE_ONE:
		integer_text = "-1"
	case CLI_SECRET_BOUNDARY_FILE_INTEGER_ONE:
		integer_text = "1"
	case CLI_SECRET_BOUNDARY_FILE_INTEGER_TWO:
		integer_text = "2"
	}
	if integer_text != "" {
		completion.Data = copy(buffer, integer_text)
		completion.Error = nil
		callback(completion)
		return
	}
	maximum := file == CLI_SECRET_BOUNDARY_FILE_MAXIMUM
	if file == CLI_SECRET_BOUNDARY_FILE_MAXIMUM_TEXT {
		maximum = true
	}
	if maximum {
		for index := range buffer[:cli_secret_boundary_size(file)] {
			buffer[index] = 'x'
		}
		completion.Data = cli_secret_boundary_size(file)
	} else if file == CLI_SECRET_BOUNDARY_FILE_EMPTY {
		completion.Data = 0
	} else if file == CLI_SECRET_BOUNDARY_FILE_TWO {
		completion.Data = copy(buffer, "xx")
	} else if file == CLI_SECRET_BOUNDARY_FILE_PATH_MAXIMUM {
		completion.Data = copy(buffer, "4")
	} else {
		completion.Data = copy(buffer, "private-non-number")
	}
	completion.Error = nil
	callback(completion)
}

func cli_secret_boundary_size(file nbio.File) (size int) {
	if file == CLI_SECRET_BOUNDARY_FILE_MAXIMUM_TEXT {
		return strings.TEXT_SIZE_MAXIMUM
	}
	return cli.SECRET_BYTES_MAX
}

func cli_secret_boundary_close(
	_ nbio.IO, completion nbio.Completion_Handle,
	_ nbio.File, callback nbio.Callback,
) {
	completion.Error = nil
	callback(completion)
}

// Test_Parse_External_Short_Circuit verifies that help and argument errors do not need a loop or
// an external source.
func Test_Parse_External_Short_Circuit(t *testing.T) {
	program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "external",
		Arguments: []cli.Option{
			cli.New_Option(cli.New_Option_Input{Label: "target"}),
		},
		Secrets: []cli.Secret{
			cli.New_Secret(cli.New_Secret_Input{
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
	if !cli.Parse_Error_Equals(help.Error, cli.HELP_REQUESTED) {
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
	if !cli.Parse_Error_Present(invalid.Error) {
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
	program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "external",
		Environment_Variables: []cli.Environment_Variable{
			cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
				Key: "PUBLIC", String: "default", Description: "public value",
			}),
			cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
				Key: "HIDDEN_ENV", Hidden: true,
			}),
			cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
				Key: "OLD_ENV", Deprecated: "use PUBLIC",
			}),
			cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
				Key: "EMPTY_OLD", Deprecated: "use PUBLIC",
			}),
		},
		Secrets: []cli.Secret{
			cli.New_Secret(cli.New_Secret_Input{
				Paths: []string{"/secrets/TOKEN"}, Description: "token",
			}),
			cli.New_Secret(cli.New_Secret_Input{
				Paths: []string{"/secrets/HIDDEN_SECRET"}, Hidden: true,
			}),
			cli.New_Secret(cli.New_Secret_Input{
				Paths: []string{"/secrets/OLD_SECRET"}, Deprecated: "use TOKEN",
			}),
		},
	})
	text := assert_external_help(t, program)
	var parser cli.Parser
	cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
		Arguments:   []string{"external"},
		Environment: []string{"OLD_ENV=value", "EMPTY_OLD="},
		Loop:        cli.Secret_IO_Of(loop),
	})
	result := drive_parser(t, driver, &parser)
	warning_output := new_cli_output()
	cli.Print_Deprecations(
		cli_output_reference(&warning_output), result.Command.Deprecation_Warnings,
	)
	warnings := string(cli.Output_Bytes(cli_output_reference(&warning_output)))
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
	help := new_output_buffer()
	cli.Print_Help(output_cli(&help), program)
	text = output_text(&help)
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
					cli.New_Environment_Variable_Input{
						Key: "lower",
					},
				),
			}, nil)
		},
		func() {
			external_program([]cli.Environment_Variable{
				cli.New_Environment_Variable(
					cli.New_Environment_Variable_Input{
						Key: "DUPLICATE",
					},
				),
				cli.New_Environment_Variable(
					cli.New_Environment_Variable_Input{
						Key: "DUPLICATE",
					},
				),
			}, nil)
		},
		func() {
			external_program([]cli.Environment_Variable{
				cli.New_Environment_Variable(
					cli.New_Environment_Variable_Input{
						Key: "COLLISION",
					},
				),
			}, []cli.Secret{
				cli.New_Secret(cli.New_Secret_Input{
					Paths: []string{"/secrets/COLLISION"},
				}),
			})
		},
		func() {
			external_program([]cli.Environment_Variable{
				cli.New_Environment_Variable(
					cli.New_Environment_Variable_Input{
						Key: "REQUIRED", Required: true, String: "default",
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
				cli.New_Secret(cli.New_Secret_Input{}),
			})
		},
		func() {
			external_program(nil, []cli.Secret{
				cli.New_Secret(cli.New_Secret_Input{
					Paths: []string{"relative/TOKEN"},
				}),
			})
		},
		func() {
			external_program(nil, []cli.Secret{
				cli.New_Secret(cli.New_Secret_Input{
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
	return cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
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
		seed_secret_directories(t, loop, driver, source.Path)
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
			CLI_SIM_DEADLINE, func(completed_write nbio.Completion_Handle) {
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
		nbio.IO_Close(loop, &completion, file, func(
			completed_close nbio.Completion_Handle,
		) {
			if completed_close.Error != nil {
				t.Errorf("close secret file: %v", completed_close.Error)
			}
			completed = true
		})
		drive_sim_operation(t, driver, func() (finished bool) { return completed })
	}
}

func seed_secret_directories(
	t *testing.T, loop nbio.IO, driver nbio.Driver, file_path string,
) {
	t.Helper()
	for index := 1; index < len(file_path); index++ {
		if file_path[index] != '/' {
			continue
		}
		make_err := cli_sim_make_directory(t, loop, driver, file_path[:index])
		if make_err == nil {
			continue
		}
		if !errors.Is(make_err, nbio.Path_Exists) {
			t.Fatalf("make parent directory: %v", make_err)
		}
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
		func(completed nbio.Completion_Handle) {
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
		0o700, func(completed nbio.Completion_Handle) {
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
		cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
			Label: "prog",
			Flags: []cli.Option{
				cli.New_Option(cli.New_Option_Input{
					Label: "color", String_Enum: []string{"auto", "never"},
					String: "rainbow", Is_Flag: true,
				}),
			},
		})
	})
	// An enum with no permitted values is malformed.
	assert_panics(t, "empty enum set", func() {
		cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
			Label: "prog",
			Arguments: []cli.Option{
				cli.New_Option(cli.New_Option_Input{
					Label: "format", String_Enum: []string{},
				}),
			},
		})
	})
	// The enum's element type must match the option's value type.
	assert_panics(t, "enum element type mismatch", func() {
		cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
			Label: "prog",
			Arguments: []cli.Option{{
				Label:       "format",
				Enumeration: cli.Option_Enumeration{Integers: []int{1, 2}},
			}},
		})
	})
	// A variadic argument cannot also carry an enum: enums are single-valued.
	assert_panics(t, "enum on a variadic argument", func() {
		cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
			Label: "prog",
			Arguments: []cli.Option{
				{
					Label: "path",
					Type: cli.Option_Type_State{
						Value: cli.OPTION_TYPE_STRINGS,
					},
					Enumeration: cli.Option_Enumeration{
						String: []string{"a", "b"},
					},
				},
			},
		})
	})
	assert_reserved_help_validation(t)
}

// Reserved labels must retain parser-owned help behavior.
func assert_reserved_help_validation(t *testing.T) {
	t.Helper()
	testify.Panics(t, func() {
		cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
			Label: "tool",
			Flags: []cli.Option{{
				Label: "help",
				Type:  cli.Option_Type_State{Value: cli.OPTION_TYPE_BOOLEAN},
				State: cli.Option_State{Is_Flag: true},
			}},
		})
	}, "user option named help")
	testify.Panics(t, func() {
		cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
			Label: "tool",
			Flags: []cli.Option{{
				Label: "h",
				Type:  cli.Option_Type_State{Value: cli.OPTION_TYPE_BOOLEAN},
				State: cli.Option_State{Is_Flag: true},
			}},
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
	if err != "" {
		t.Fatalf(
			"%v: unexpected error: %s", arguments, err,
		)
	}
	got := cli.Option_Strings(
		cli.Resolved_Options(command.Arguments), cli.Option_Label(label),
	)
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
	Environment           cli.Resolved_Environment
	Secrets               []cli.Secret
	Resolved_Secrets      cli.Resolved_Secrets
	Command               cli.Resolved_Command
	Warning_Count_Maximum int
	Commands              []cli.Command
}

func cli_maximum_aggregate_fixture() (fixture cli_maximum_aggregate) {
	maximum_text := cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'a')
	maximum_options, maximum_resolved_options := cli_maximum_options()
	maximum_environment, maximum_resolved_environment := cli_maximum_environment()
	maximum_secrets, maximum_resolved_secrets := cli_maximum_secrets()
	maximum_commands := make([]cli.Command, slices.COUNT_MAXIMUM)
	for index := range maximum_commands {
		maximum_commands[index].Hidden = true
	}
	maximum_commands[0] = cli.Command{Label: "c"}
	maximum_warning_count := strings.TEXT_SIZE_MAXIMUM / len("warning: \n")
	maximum_warnings := make([]cli.Warning, maximum_warning_count)
	maximum_declaration := cli.Command{
		Label: cli.Label(maximum_text), Description: cli.Description(maximum_text),
		Arguments: cli.Arguments(maximum_options), Flags: cli.Flags(maximum_options),
		Hidden: true, Deprecated: cli.Deprecation(maximum_text),
	}
	maximum_command := cli.Resolved_Command{
		Parsed_Command: cli.Parsed_Command{
			Selected_Command: cli.Selected_Command{
				Label:       cli.Label(maximum_text),
				Description: cli.Description(maximum_text),
				Arguments:   cli.Resolved_Arguments(maximum_resolved_options),
				Flags:       cli.Resolved_Flags(maximum_resolved_options),
				Hidden:      true,
				Deprecated:  cli.Deprecation(maximum_text),
			},
			Deprecation_Warnings: maximum_warnings,
		},
		Environment: maximum_resolved_environment,
		Secrets:     maximum_resolved_secrets,
	}
	fixture.Program = cli.Program{
		Label:       "p",
		Description: "d",
		Selection: cli.Program_Selection{
			Commands:        maximum_commands,
			Single_Commands: cli.Single_Commands{Command: maximum_declaration},
			Global_Flags:    maximum_options,
			Help_Flags: cli.Help_Flags{
				Hidden: 1,
			},
		},
		Environment_Variables: maximum_environment,
		Secrets:               maximum_secrets,
	}
	fixture.Options = maximum_options
	fixture.Environment = maximum_resolved_environment
	fixture.Secrets = maximum_secrets
	fixture.Resolved_Secrets = maximum_resolved_secrets
	fixture.Command = maximum_command
	fixture.Warning_Count_Maximum = maximum_warning_count
	fixture.Commands = maximum_commands
	return fixture
}

func cli_maximum_options() (
	declarations []cli.Option, resolved cli.Resolved_Options,
) {
	declarations = make([]cli.Option, slices.COUNT_MAXIMUM)
	resolved = make(cli.Resolved_Options, slices.COUNT_MAXIMUM)
	for index := range declarations {
		declarations[index] = cli.Option{
			Label: "x", Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_BOOLEAN},
			State: cli.Option_State{Hidden: true, Is_Flag: true},
		}
	}
	declarations[0].State.Deprecated = "d"
	for index := range declarations {
		resolved[index] = cli_resolved_option(
			declarations[index], cli.Resolved_Option_State{
				Hidden:     declarations[index].State.Hidden,
				Deprecated: declarations[index].State.Deprecated,
			},
		)
	}
	return declarations, resolved
}

func cli_maximum_secrets() (
	declarations []cli.Secret, resolved cli.Resolved_Secrets,
) {
	declarations = make([]cli.Secret, slices.COUNT_MAXIMUM)
	resolved = make(cli.Resolved_Secrets, slices.COUNT_MAXIMUM)
	for index := range declarations {
		declarations[index] = cli.Secret{
			Key: "X", Paths: []string{"/X"}, Hidden: true,
		}
		resolved[index] = cli.Resolved_Secret{Secret: declarations[index]}
	}
	return declarations, resolved
}

func cli_maximum_environment() (
	declarations []cli.Environment_Variable, resolved cli.Resolved_Environment,
) {
	declarations = make([]cli.Environment_Variable, slices.COUNT_MAXIMUM)
	resolved = make(cli.Resolved_Environment, slices.COUNT_MAXIMUM)
	for index := range declarations {
		declarations[index] = cli.Environment_Variable{
			Key: "X", Hidden: true,
		}
		resolved[index] = cli_resolved_environment(
			declarations[index], cli.Environment_State{},
		)
	}
	return declarations, resolved
}

func cli_bounded_aggregate_render(t *testing.T, fixture *cli_maximum_aggregate) {
	t.Helper()
	output := new_cli_output()
	cli.Print_Help(cli_output_reference(&output), fixture.Program)
	cli.Output_Reset(cli_output_reference(&output))
	cli.Print_Requested_Help(cli_output_reference(&output), fixture.Program, "")
	cli.Output_Reset(cli_output_reference(&output))
	cli.Print_Command(cli_output_reference(&output), fixture.Program, cli.Command{Label: "p"})
	cli_complete(fixture.Program, []string{"p", ""})
	base_command := fixture.Commands[0]
	fixture.Commands[0] = cli.Command{
		Label: "c",
		Flags: []cli.Option{{
			Label: "e", Enumeration: cli.Option_Enumeration{String: []string{"v"}},
			State: cli.Option_State{String: "v", Is_Flag: true},
		}},
	}
	cli_complete(fixture.Program, []string{"p", "c", "-"})
	cli_complete(fixture.Program, []string{"p", "c", "-e="})
	fixture.Commands[0] = base_command
	cli.Output_Reset(cli_output_reference(&output))
	if !handle_completion(
		fixture.Program,
		[]string{"p", "__complete", "p", ""},
		cli_output_reference(&output),
	) {
		t.Fatal("maximum aggregate completion was not handled")
	}
	if _, script_err := completion_script(
		fixture.Program, "bash",
	); script_err != nil {
		t.Fatalf("maximum aggregate completion script: %v", script_err)
	}

	single := fixture.Program
	single.Selection.Mode.Value = cli.PROGRAM_MODE_SINGLE
	cli_complete(single, []string{"p", ""})
	cli.Output_Reset(cli_output_reference(&output))
	cli.Print_Deprecations(
		cli_output_reference(&output), fixture.Command.Deprecation_Warnings,
	)
}

func cli_bounded_aggregate_parse(t *testing.T, fixture *cli_maximum_aggregate) {
	t.Helper()
	global_flags := make(cli.Global_Flag_Storage, slices.COUNT_MAXIMUM)
	var parser cli.Parser
	assert_panics(t, "secret declarations without an I/O loop", func() {
		cli.Program_Parse(fixture.Program, &parser, cli.Program_Parse_Input{
			Arguments:         []string{"p", "c", "-x"},
			Command_Arguments: make(cli.Command_Argument_Storage, slices.COUNT_MAXIMUM),
			Command_Flags:     make(cli.Command_Flag_Storage, slices.COUNT_MAXIMUM),
			Global_Flags:      global_flags,
			Filled:            make([]bool, slices.COUNT_MAXIMUM),
			Positionals:       make([]cli.Indexed_Token, slices.COUNT_MAXIMUM),
			Slice_Named:       make([]cli.Indexed_Token, slices.COUNT_MAXIMUM),
			String_Values:     make([]string, slices.COUNT_MAXIMUM),
			Integer_Values:    make([]int, slices.COUNT_MAXIMUM),
			Failure_Storage:   make([]byte, cli.FAILURE_SIZE_MAXIMUM),
		})
	})
	cli.Parser_Done(&parser)

	parser = cli.Parser{
		Publication: cli.Publication{
			Result:               cli.Parse_Result{Command: fixture.Command},
			Completion:           cli.Parser_Completion{Value: true},
			Secret_Errors:        make([]cli.Secret_Failure, slices.COUNT_MAXIMUM),
			Environment_Warnings: make([]cli.Warning, fixture.Warning_Count_Maximum),
			Secret_Warnings:      make([]cli.Warning, slices.COUNT_MAXIMUM),
			Secret_Count:         slices.COUNT_MAXIMUM,
			Failure:              cli.Failure{},
		},
		Workspace: cli.Workspace{
			Command_Arguments: make(cli.Workspace_Arguments, slices.COUNT_MAXIMUM),
			Command_Flags:     make(cli.Workspace_Flags, slices.COUNT_MAXIMUM),
			Filled:            make([]bool, slices.COUNT_MAXIMUM),
			Positionals:       make([]cli.Indexed_Token, slices.COUNT_MAXIMUM),
			Slice_Named:       make([]cli.Indexed_Token, slices.COUNT_MAXIMUM),
			String_Values:     make([]string, slices.COUNT_MAXIMUM),
			Integer_Values:    make([]int, slices.COUNT_MAXIMUM),
		},
	}
	cli.Program_Parse(fixture.Program, &parser, cli.Program_Parse_Input{
		Arguments:         []string{"p", "c", "-help"},
		Command_Arguments: make(cli.Command_Argument_Storage, slices.COUNT_MAXIMUM),
		Command_Flags:     make(cli.Command_Flag_Storage, slices.COUNT_MAXIMUM),
		Global_Flags:      global_flags,
		Filled:            make([]bool, slices.COUNT_MAXIMUM),
		Positionals:       make([]cli.Indexed_Token, slices.COUNT_MAXIMUM),
		Slice_Named:       make([]cli.Indexed_Token, slices.COUNT_MAXIMUM),
		String_Values:     make([]string, slices.COUNT_MAXIMUM),
		Integer_Values:    make([]int, slices.COUNT_MAXIMUM),
		Failure_Storage:   make([]byte, cli.FAILURE_SIZE_MAXIMUM),
	})
	cli.Parser_Done(&parser)
}

func cli_bounded_aggregate_lookup(t *testing.T, fixture *cli_maximum_aggregate) {
	t.Helper()
	option := cli.Get_Option(cli.Resolved_Options(fixture.Command.Arguments), "x")
	if option.Label != "x" {
		t.Fatal("maximum option lookup changed the first declaration")
	}
	variable := cli.Get_Environment(fixture.Environment, "X")
	if variable.Key != "X" {
		t.Fatal("maximum environment lookup changed the first declaration")
	}
	secret := cli.Get_Secret(fixture.Resolved_Secrets, "X")
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
	cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_MULTICALL,
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
	if secret_key_size > cli.SECRET_KEY_SIZE_MAXIMUM {
		secret_key_size = cli.SECRET_KEY_SIZE_MAXIMUM
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
	argument = cli.New_Option(cli.New_Option_Input{
		Label:       cli.Option_Label(fixture.Argument_Label),
		Description: cli.Description(fixture.Description),
	})
	cli.New_Option(cli.New_Option_Input{Type: cli.OPTION_TYPE_STRINGS,
		Label:       cli.Option_Label(fixture.Argument_Label),
		Description: cli.Description(fixture.Description),
	})
	cli.New_Option(cli.New_Option_Input{
		Label:       cli.Option_Label(fixture.Flag_Label),
		String:      cli.Value_Text(fixture.String_Value),
		Is_Flag:     true,
		Description: cli.Description(fixture.Description), Hidden: true,
		Deprecated: cli.Deprecation(fixture.Description),
	})
	cli.New_Option(cli.New_Option_Input{
		Label:       cli.Option_Label(fixture.Flag_Label),
		String:      cli.Value_Text(fixture.String_Value),
		String_Enum: []string{fixture.String_Value}, Is_Flag: true,
		Description: cli.Description(fixture.Description), Hidden: true,
		Deprecated: cli.Deprecation(fixture.Description),
	})
	cli.New_Option(cli.New_Option_Input{
		Label:       cli.Option_Label(fixture.Argument_Label),
		String_Enum: []string{fixture.String_Value},
		Description: cli.Description(fixture.Description),
	})
	variable = cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
		Key:         cli.External_Key(fixture.External_Key),
		Description: cli.Description(fixture.Description),
		String:      cli.Value_Text(fixture.String_Value),
		String_Enum: []string{fixture.String_Value}, Hidden: true,
		Deprecated: cli.Deprecation(fixture.Description),
	})
	secret = cli.New_Secret(cli.New_Secret_Input{
		Paths:       []string{"/" + fixture.Secret_Key},
		Description: cli.Description(fixture.Description),
		String_Enum: []string{fixture.String_Value}, Hidden: true,
		Deprecated: cli.Deprecation(fixture.Description),
	})
	return argument, variable, secret
}

func cli_bounded_text_runtime(
	t *testing.T, fixture cli_text_fixture, argument cli.Option,
) {
	t.Helper()
	program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label:       cli.Label(fixture.Program_Label),
		Description: cli.Description(fixture.Description),
		Arguments:   []cli.Option{argument},
	})
	var parser cli.Parser
	cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
		Arguments:         []string{fixture.Program_Label, "value"},
		Command_Arguments: make(cli.Command_Argument_Storage, 1),
		Filled:            make([]bool, 1),
		Positionals:       make([]cli.Indexed_Token, 1),
		Slice_Named:       make([]cli.Indexed_Token, 1),
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("bounded text parser did not complete")
	}
	if cli.Parse_Error_Present(result.Error) {
		t.Fatalf("bounded text parse: %v", result.Error)
	}
	if cli.Get_Option(
		cli.Resolved_Options(result.Command.Arguments),
		cli.Option_Label(fixture.Argument_Label),
	).Label == "" {
		t.Fatal("bounded text option lookup failed")
	}
	cli_complete(program, []string{fixture.Program_Label, ""})
	output := new_cli_output()
	cli.Print_Command(cli_output_reference(&output), program, cli.Command{Label: "x"})
	cli.Output_Reset(cli_output_reference(&output))
	selected := program.Selection.Single_Commands.Command.Label
	if len(selected) == strings.TEXT_SIZE_MAXIMUM {
		assert_panics(t, "maximum requested command help", func() {
			cli.Print_Requested_Help(cli_output_reference(&output), program, selected)
		})
	} else {
		cli.Print_Requested_Help(cli_output_reference(&output), program, selected)
	}
	cli.Output_Reset(cli_output_reference(&output))
	handle_completion(program, []string{
		fixture.Program_Label, "__complete", fixture.Program_Label, "",
	}, cli_output_reference(&output))
}

func cli_bounded_integer_text() {
	integer_edges := [...]int{bits.INTEGER_MINIMUM, bits.INTEGER_MAXIMUM}
	for _, integer := range integer_edges {
		cli.New_Option(cli.New_Option_Input{
			Label: "n", Integer: cli.Integer(integer),
			Type: cli.OPTION_TYPE_INTEGER, Is_Flag: true,
		})
		cli.New_Option(cli.New_Option_Input{
			Label: "n", Integer: cli.Integer(integer),
			Integer_Enum: []int{integer}, Type: cli.OPTION_TYPE_INTEGER, Is_Flag: true,
		})
		cli.New_Option(cli.New_Option_Input{
			Label: "n", Integer_Enum: []int{integer}, Type: cli.OPTION_TYPE_INTEGER,
		})
		cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
			Key: "NUMBER", Type: cli.OPTION_TYPE_INTEGER, Integer: cli.Integer(integer),
			Integer_Enum: []int{integer},
		})
		cli.New_Secret(cli.New_Secret_Input{
			Paths: []string{"/NUMBER"}, Type: cli.OPTION_TYPE_INTEGER,
			Integer_Enum: []int{integer},
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
				Key: "", Description: "d", Deprecated: "d",
				Allow_Empty: true,
			},
			{
				Key: "XX", Paths: []string{"/missing/XX"},
				Description: "dd", Deprecated: "dd", Allow_Empty: true,
			},
		},
	), 2)
	counts := [...]int{1, 2, slices.COUNT_MAXIMUM}
	for _, count := range counts {
		secrets := make([]cli.Secret, count)
		for index := range secrets {
			secrets[index] = cli.Secret{
				Key: "X", Paths: []string{"/missing/X"},
			}
		}
		environment := make([]cli.Environment_Variable, count)
		for index := range environment {
			environment[index] = cli.Environment_Variable{Key: "E"}
		}
		flags := make([]cli.Option, count)
		for index := range flags {
			flags[index] = cli.Option{
				Label: "f",
				Type:  cli.Option_Type_State{Value: cli.OPTION_TYPE_BOOLEAN},
				State: cli.Option_State{Is_Flag: true},
			}
		}
		cli_parse_absent_secrets(t, cli_raw_single_program(
			"p", cli.Command{Label: "p", Flags: flags}, environment, secrets,
		), count)

		present := make([]cli.Secret, count)
		for index := range present {
			present[index] = cli.Secret{
				Key: "X", Paths: []string{"/X"},
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
			[]cli.Secret{{Key: "X", Paths: paths}},
		), count)
	}
	maximum_paths := make([]string, slices.COUNT_MAXIMUM)
	for index := range maximum_paths[:len(maximum_paths)-1] {
		maximum_paths[index] = "/missing/X"
	}
	maximum_paths[len(maximum_paths)-1] = "/X"
	cli_parse_present_secrets(t, cli_raw_single_program(
		"p", cli.Command{Label: "p"}, nil,
		[]cli.Secret{{Key: "X", Paths: maximum_paths}},
	), slices.COUNT_MAXIMUM)
}

func cli_bounded_secret_aggregate(t *testing.T) {
	t.Helper()
	arguments := make([]cli.Option, slices.COUNT_MAXIMUM)
	for index := range arguments {
		arguments[index] = cli.Option{
			Label: "a", Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_STRINGS},
		}
	}
	cli_parse_present_secrets(t, cli_raw_single_program(
		"p", cli.Command{Label: "p", Arguments: arguments}, nil,
		[]cli.Secret{{
			Key: "X", Paths: []string{"/X"},
		}},
	), slices.COUNT_MAXIMUM)

	maximum_text := cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'p')
	maximum_key := cli_boundary_text(cli.SECRET_KEY_SIZE_MAXIMUM, 'X')
	cli_parse_present_secrets(t, cli_raw_single_program(
		cli.Program_Label(maximum_text), cli.Command{
			Label: cli.Label(maximum_text), Description: cli.Description(maximum_text),
			Deprecated: cli.Deprecation(maximum_text),
		}, nil, []cli.Secret{{
			Key: cli.Secret_Key(maximum_key), Paths: []string{"/" + maximum_key},
			Description: cli.Description(maximum_text),
			Deprecated:  cli.Deprecation(maximum_text),
		}},
	), 1)
}

func cli_raw_single_program(
	label cli.Program_Label, command cli.Command,
	environment []cli.Environment_Variable, secrets []cli.Secret,
) (program cli.Program) {
	return cli.Program{
		Label: label,
		Selection: cli.Program_Selection{
			Single_Commands: cli.Single_Commands{Command: command},
			Help_Flags:      cli.Help_Flags{},
			Mode:            cli.Program_Mode{Value: cli.PROGRAM_MODE_SINGLE},
		},
		Environment_Variables: environment, Secrets: secrets,
	}
}

func cli_resolved_environment(
	declaration cli.Environment_Variable, state cli.Environment_State,
) (resolved cli.Resolved_Environment_Variable) {
	return cli.Resolved_Environment_Variable{
		Key: declaration.Key, Description: declaration.Description,
		Type: declaration.Type, Enumeration: declaration.Enumeration,
		Required: declaration.Required, Allow_Empty: declaration.Allow_Empty,
		Hidden: declaration.Hidden, Deprecated: declaration.Deprecated,
		State: state,
	}
}

func cli_resolved_option(
	declaration cli.Option, state cli.Resolved_Option_State,
) (resolved cli.Resolved_Option) {
	return cli.Resolved_Option{
		Label: declaration.Label, Description: declaration.Description,
		Type: declaration.Type, Enumeration: declaration.Enumeration,
		State: state,
	}
}

func cli_parse_absent_secrets(t *testing.T, program cli.Program, storage_count int) {
	t.Helper()
	var parser cli.Parser
	cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
		Arguments:         []string{"p"},
		Loop:              cli_secret_boundary_loop(),
		Command_Arguments: make(cli.Command_Argument_Storage, storage_count),
		Command_Flags:     make(cli.Command_Flag_Storage, storage_count),
		Global_Flags:      make(cli.Global_Flag_Storage, storage_count),
		Filled:            make([]bool, storage_count),
		Positionals:       make([]cli.Indexed_Token, storage_count),
		Slice_Named:       make([]cli.Indexed_Token, storage_count),
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("synchronous absent secrets did not publish")
	}
	if cli.Parse_Error_Present(result.Error) {
		t.Fatalf("optional absent secret: %v", result.Error)
	}
}

func cli_parse_present_secrets(t *testing.T, program cli.Program, storage_count int) {
	t.Helper()
	var parser cli.Parser
	cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
		Arguments:         []string{string(program.Label)},
		Loop:              cli_secret_boundary_loop(),
		Command_Arguments: make(cli.Command_Argument_Storage, storage_count),
		Command_Flags:     make(cli.Command_Flag_Storage, storage_count),
		Global_Flags:      make(cli.Global_Flag_Storage, storage_count),
		Filled:            make([]bool, storage_count),
		Positionals:       make([]cli.Indexed_Token, storage_count),
		Slice_Named:       make([]cli.Indexed_Token, storage_count),
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("synchronous present secrets did not publish")
	}
	if cli.Parse_Error_Present(result.Error) {
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
	cli_bounded_parser_failure_maximum(t)
}

func cli_bounded_parser_failure_maximum(t *testing.T) {
	t.Helper()
	command := cli.Command{Label: "p"}
	parser := cli_boundary_parser(command, 0, cli.FAILURE_SIZE_MAXIMUM)
	parser.Completion.Value = true
	if _, complete := cli.Parser_Done(&parser); !complete {
		t.Fatal("maximum failure parser did not publish")
	}
	parser.Completion.Value = false
	program := cli_raw_single_program("p", command, nil, nil)
	cli.Program_Parse(program, &parser, cli.Program_Parse_Input{
		Arguments:       []string{"p"},
		Failure_Storage: make([]byte, cli.FAILURE_SIZE_MAXIMUM),
	})
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
	cli_bounded_program_command_metadata()
	cli_bounded_external_defaults()
}

func cli_bounded_program_command_metadata() {
	for _, size := range [...]int{1, 2, strings.TEXT_SIZE_MAXIMUM} {
		text := cli_boundary_text(size, 'c')
		cli.New(cli.New_Input{
			Label: "p",
			Commands: []cli.Command{{
				Label: cli.Label(text), Description: cli.Description(text),
				Hidden: true, Deprecated: cli.Deprecation(text),
			}},
		})
	}
}

func cli_bounded_external_defaults() {
	maximum := cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'v')
	variable := cli.New_Environment_Variable(
		cli.New_Environment_Variable_Input{Key: "E", String: cli.Value_Text(maximum)},
	)
	cli.New(cli.New_Input{
		Mode: cli.PROGRAM_MODE_SINGLE, Label: "p",
		Environment_Variables: []cli.Environment_Variable{variable},
	})
	for _, value := range [...]int{
		bits.INTEGER_MINIMUM, bits.INTEGER_MAXIMUM, -1, 2,
	} {
		variable = cli.New_Environment_Variable(
			cli.New_Environment_Variable_Input{
				Key: "E", Type: cli.OPTION_TYPE_INTEGER,
				Integer: cli.Integer(value),
			},
		)
		cli.New(cli.New_Input{
			Mode: cli.PROGRAM_MODE_SINGLE, Label: "p",
			Environment_Variables: []cli.Environment_Variable{variable},
		})
	}
}

// Test_Bounded_Enum_Display_Name keeps the maximum positional identity observable.
func Test_Bounded_Enum_Display_Name(t *testing.T) {
	label := cli_boundary_text(cli.OPTION_LABEL_SIZE_MAXIMUM, 'a')
	program := cli_raw_single_program("p", cli.Command{
		Label: "p",
		Arguments: []cli.Option{cli.New_Option(
			cli.New_Option_Input{
				Label: cli.Option_Label(label), String_Enum: []string{"x"},
			},
		)},
	}, nil, nil)
	command, err := parse_program(&program, []string{"p", "x"})
	if err != "" {
		t.Fatalf("maximum enum display name: %s", err)
	}
	if cli.Option_String(
		cli.Resolved_Options(command.Arguments), cli.Option_Label(label),
	) != "x" {
		t.Fatal("maximum enum display name discarded its value")
	}
}

// Test_Bounded_Enum_Validation keeps complete defaults and permitted sets at validation.
func Test_Bounded_Enum_Validation(t *testing.T) {
	maximum_text := cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'x')
	for _, value := range [...]string{"x", "xx", maximum_text} {
		option := cli.New_Option(cli.New_Option_Input{
			Label: "value", String: cli.Value_Text(value), String_Enum: []string{value},
		})
		cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
			Label: "p", Arguments: []cli.Option{option},
		})
	}
	string_members := make([]string, slices.COUNT_MAXIMUM)
	integer_members := make([]int, slices.COUNT_MAXIMUM)
	for index := range string_members {
		string_members[index] = "x"
	}
	cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "p", Arguments: []cli.Option{cli.New_Option(cli.New_Option_Input{
			Label: "value", String: "x", String_Enum: string_members,
		})},
	})
	for _, value := range [...]int{bits.INTEGER_MINIMUM, bits.INTEGER_MAXIMUM, -1, 2} {
		cli_bounded_integer_enum_validation(value, []int{value})
	}
	cli_bounded_integer_enum_validation(0, integer_members)
}

func cli_bounded_integer_enum_validation(value int, members []int) {
	option := cli.New_Option(cli.New_Option_Input{
		Label: "value", Integer: cli.Integer(value), Integer_Enum: members,
		Type: cli.OPTION_TYPE_INTEGER,
	})
	cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "p", Arguments: []cli.Option{option},
	})
}

func cli_bounded_empty_option_constructors() {
	cli.New_Option(cli.New_Option_Input{})
	cli.New_Option(cli.New_Option_Input{Type: cli.OPTION_TYPE_STRINGS})
	cli.New_Option(cli.New_Option_Input{Is_Flag: true})
	cli.New_Option(cli.New_Option_Input{})
	cli.New_Option(cli.New_Option_Input{Type: cli.OPTION_TYPE_INTEGER})
	cli.New_Option(cli.New_Option_Input{Is_Flag: true})
	cli.New_Option(cli.New_Option_Input{Type: cli.OPTION_TYPE_INTEGER, Is_Flag: true})
	cli.New_Option(cli.New_Option_Input{
		Type: cli.OPTION_TYPE_BOOLEAN, Boolean: true, Is_Flag: true,
	})
	cli.New_Option(cli.New_Option_Input{
		Integer: -1, Type: cli.OPTION_TYPE_INTEGER, Is_Flag: true,
	})
}

func cli_bounded_empty_external_constructors() {
	cli.New_Environment_Variable(cli.New_Environment_Variable_Input{})
	cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
		String_Enum: []string{},
	})
	cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
		Type: cli.OPTION_TYPE_INTEGER, Integer_Enum: []int{},
	})
	cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
		Type: cli.OPTION_TYPE_INTEGER, Integer: -1, Integer_Enum: []int{},
	})
	cli.New_Secret(cli.New_Secret_Input{})
	cli.New_Secret(cli.New_Secret_Input{Type: cli.OPTION_TYPE_INTEGER})
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
				Type:  cli.Option_Type_State{Value: cli.OPTION_TYPE_BOOLEAN},
				State: cli.Option_State{Is_Flag: true},
			}
		}
		program := cli.New(cli.New_Input{
			Label: "p", Global_Flags: global_flags,
			Commands: []cli.Command{{
				Label: "c", Flags: []cli.Option{{
					Label: "local",
					Type: cli.Option_Type_State{
						Value: cli.OPTION_TYPE_BOOLEAN,
					},
					State: cli.Option_State{Is_Flag: true},
				}},
			}},
		})
		got_count := len(program.Selection.Global_Flags)
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
			Label: cli.Option_Label("g" + cli_decimal_text(index)),
			Type:  cli.Option_Type_State{Value: cli.OPTION_TYPE_BOOLEAN},
			State: cli.Option_State{Is_Flag: true, Deprecated: "d"},
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
	cli.Program_Parse(program, &parser, cli.Program_Parse_Input{
		Arguments: tokens, Global_Flags: make(cli.Global_Flag_Storage, len(flags)),
		Filled:               make([]bool, len(flags)),
		Failure_Storage:      make([]byte, cli.FAILURE_SIZE_MAXIMUM),
		Deprecation_Warnings: make([]cli.Warning, cli.DEPRECATION_WARNING_COUNT_MAXIMUM),
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("global flag parser did not complete")
	}
	if cli.Parse_Error_Present(result.Error) {
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
	for case_index, text_size := range text_sizes {
		cli_bounded_external_accessor_case(
			case_index, text_size, byte_sizes[case_index],
		)
	}
	var environment_bytes [strings.TEXT_SIZE_MAXIMUM]byte
	maximum_string := cli_resolved_environment(
		cli.Environment_Variable{
			Key: "E",
			Type: cli.External_Type_State{
				Value: cli.Scalar_Option_Type(cli.OPTION_TYPE_STRING),
			},
		},
		cli.Environment_State{
			String: cli.Value_Text(string(environment_bytes[:])),
		},
	)
	cli.Environment_String(cli.Resolved_Environment{maximum_string}, maximum_string.Key)
	cli_bounded_external_integers()
}

// Environment accessors search every admitted resolved collection size.
func Test_Environment_Accessor_Collection_Bounds(t *testing.T) {
	for _, count := range [...]int{0, 1, 2, slices.COUNT_MAXIMUM} {
		string_enum := make([]string, count)
		integer_enum := make([]int, count)
		for index := range string_enum {
			string_enum[index] = "x"
			integer_enum[index] = 1
		}
		text := cli.Resolved_Environment_Variable{
			Key: "E", Type: cli.External_Type_State{
				Value: cli.Scalar_Option_Type(cli.OPTION_TYPE_STRING),
			}, Enumeration: cli.External_Enumeration{String: string_enum},
			State: cli.Environment_State{String: "x"},
		}
		integer := cli.Resolved_Environment_Variable{
			Key: "E", Type: cli.External_Type_State{
				Value: cli.Scalar_Option_Type(cli.OPTION_TYPE_INTEGER),
			}, Enumeration: cli.External_Enumeration{Integers: integer_enum},
			State: cli.Environment_State{Integer: 1},
		}
		boolean := cli.Resolved_Environment_Variable{
			Key: "E", Type: cli.External_Type_State{
				Value: cli.Scalar_Option_Type(cli.OPTION_TYPE_BOOLEAN),
			},
		}
		cli_call_environment_accessors(t, count, text, integer, boolean)
	}
}

func cli_call_environment_accessors(
	t *testing.T, count int, text cli.Resolved_Environment_Variable,
	integer cli.Resolved_Environment_Variable,
	boolean cli.Resolved_Environment_Variable,
) {
	t.Helper()
	operations := [...]func(){
		func() { cli.Environment_String(cli_repeated_environment(count, text), "E") },
		func() { cli.Environment_Integer(cli_repeated_environment(count, integer), "E") },
		func() { cli.Environment_Boolean(cli_repeated_environment(count, boolean), "E") },
	}
	for _, operation := range operations {
		if count == 0 {
			assert_panics(t, "empty typed environment lookup", operation)
			continue
		}
		operation()
	}
}

func cli_repeated_environment(
	count int, variable cli.Resolved_Environment_Variable,
) (environment cli.Resolved_Environment) {
	environment = make(cli.Resolved_Environment, count)
	for index := range environment {
		environment[index] = variable
	}
	return environment
}

// Secret accessors search every admitted resolved collection size.
func Test_Secret_Accessor_Collection_Bounds(t *testing.T) {
	for _, count := range [...]int{0, 1, 2, slices.COUNT_MAXIMUM} {
		string_enum := make([]string, count)
		integer_enum := make([]int, count)
		for index := range string_enum {
			string_enum[index] = "x"
			integer_enum[index] = 1
		}
		text := cli.Resolved_Secret{
			Secret: cli.Secret{Key: "S", Type: cli.External_Type_State{
				Value: cli.Scalar_Option_Type(cli.OPTION_TYPE_STRING),
			}, Enumeration: cli.External_Enumeration{String: string_enum}},
		}
		integer := cli.Resolved_Secret{
			Secret: cli.Secret{Key: "S", Type: cli.External_Type_State{
				Value: cli.Scalar_Option_Type(cli.OPTION_TYPE_INTEGER),
			}, Enumeration: cli.External_Enumeration{Integers: integer_enum}},
			State: cli.Accepted_Secret_Value{Integer: 1},
		}
		boolean := cli.Resolved_Secret{Secret: cli.Secret{
			Key: "S", Type: cli.External_Type_State{
				Value: cli.Scalar_Option_Type(cli.OPTION_TYPE_BOOLEAN),
			},
		}}
		cli_call_secret_accessors(t, count, text, integer, boolean)
	}
}

func cli_call_secret_accessors(
	t *testing.T, count int, text cli.Resolved_Secret,
	integer cli.Resolved_Secret, boolean cli.Resolved_Secret,
) {
	t.Helper()
	operations := [...]func(){
		func() { cli.Secret_String(cli_repeated_secrets(count, text), "S") },
		func() { cli.Secret_Integer(cli_repeated_secrets(count, integer), "S") },
		func() { cli.Secret_Boolean(cli_repeated_secrets(count, boolean), "S") },
	}
	for _, operation := range operations {
		if count == 0 {
			assert_panics(t, "empty typed secret lookup", operation)
			continue
		}
		operation()
	}
}

func cli_repeated_secrets(
	count int, secret cli.Resolved_Secret,
) (secrets cli.Resolved_Secrets) {
	secrets = make(cli.Resolved_Secrets, count)
	for index := range secrets {
		secrets[index] = secret
	}
	return secrets
}

func cli_bounded_external_accessor_case(case_index int, text_size int, byte_size int) {
	var secret_bytes [cli.SECRET_BYTES_MAX]byte
	secret_paths := make([]string, slices.COUNT_MAXIMUM)
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
	environment := cli_resolved_environment(
		cli.Environment_Variable{
			Key: cli.External_Key(key), Description: cli.Description(text),
			Type: cli.External_Type_State{
				Value: cli.Scalar_Option_Type(cli.OPTION_TYPE_STRING),
			},
			Required: cli.Required(set), Allow_Empty: cli.Allow_Empty(set),
			Hidden: cli.Hidden(set), Deprecated: cli.Deprecation(text),
		},
		cli.Environment_State{
			String:  cli.Value_Text(text),
			Integer: cli.Integer(text_size),
			Boolean: cli.Boolean(set), Parsed: cli.Parsed(set),
		},
	)
	environment_values := cli.Resolved_Environment{environment}
	cli.Environment_String(environment_values, environment.Key)
	environment.Type.Value = cli.Scalar_Option_Type(cli.OPTION_TYPE_INTEGER)
	environment_values[0] = environment
	cli.Environment_Integer(environment_values, environment.Key)
	environment.Type.Value = cli.Scalar_Option_Type(cli.OPTION_TYPE_BOOLEAN)
	environment_values[0] = environment
	cli.Environment_Boolean(environment_values, environment.Key)

	secret := cli.Resolved_Secret{
		Secret: cli.Secret{
			Key: cli.Secret_Key(secret_key), Paths: secret_paths[:text_size],
			Description: cli.Description(text), Required: cli.Required(set),
			Allow_Empty: cli.Allow_Empty(set), Hidden: cli.Hidden(set),
			Deprecated: cli.Deprecation(text),
		},
		State: cli.Accepted_Secret_Value{
			Bytes:   cli.Secret_Value_Bytes(secret_bytes[:byte_size]),
			Integer: cli.Integer(text_size), Boolean: cli.Boolean(set),
		},
	}
	secret.Type.Value = cli.OPTION_TYPE_STRING
	secret_values := cli.Resolved_Secrets{secret}
	cli.Secret_String(secret_values, secret.Key)
	secret.Type.Value = cli.OPTION_TYPE_INTEGER
	secret_values[0] = secret
	cli.Secret_Integer(secret_values, secret.Key)
	secret.Type.Value = cli.OPTION_TYPE_BOOLEAN
	secret_values[0] = secret
	cli.Secret_Boolean(secret_values, secret.Key)
}

func cli_bounded_external_integers() {
	edges := [...]int{bits.INTEGER_MINIMUM, bits.INTEGER_MAXIMUM, -1}
	for _, edge := range edges {
		environment := cli_resolved_environment(
			cli.Environment_Variable{
				Key: "E", Type: cli.External_Type_State{
					Value: cli.Scalar_Option_Type(cli.OPTION_TYPE_INTEGER),
				},
			},
			cli.Environment_State{Integer: cli.Integer(edge)},
		)
		cli.Environment_Integer(cli.Resolved_Environment{environment}, environment.Key)
		secret := cli.Resolved_Secret{
			Secret: cli.Secret{
				Key:  "S",
				Type: cli.External_Type_State{Value: cli.OPTION_TYPE_INTEGER},
			},
			State: cli.Accepted_Secret_Value{Integer: cli.Integer(edge)},
		}
		cli.Secret_Integer(cli.Resolved_Secrets{secret}, secret.Key)
	}
}

// Test_Bounded_Environment_Default_Output keeps every typed default at help output.
func Test_Bounded_Environment_Default_Output(t *testing.T) {
	for _, value := range [...]string{
		"xx", cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'x'),
	} {
		cli_render_environment_default(t, cli.Environment_Variable{
			Key:   "E",
			State: cli.Environment_Default{String: cli.Value_Text(value)},
		}, len(value) == strings.TEXT_SIZE_MAXIMUM)
	}
	for _, value := range [...]int{
		bits.INTEGER_MINIMUM, bits.INTEGER_MAXIMUM, -1, 0, 1, 2,
	} {
		cli_render_environment_default(t, cli.Environment_Variable{
			Key: "E", Type: cli.External_Type_State{
				Value: cli.Scalar_Option_Type(cli.OPTION_TYPE_INTEGER),
			}, State: cli.Environment_Default{Integer: cli.Integer(value)},
		}, false)
	}
	cli_render_environment_default(t, cli.Environment_Variable{
		Key: "E", Type: cli.External_Type_State{
			Value: cli.Scalar_Option_Type(cli.OPTION_TYPE_BOOLEAN),
		}, State: cli.Environment_Default{Boolean: true},
	}, false)
}

func cli_render_environment_default(
	t *testing.T, variable cli.Environment_Variable, exceeds_output bool,
) {
	t.Helper()
	command := cli.Command{Label: "p"}
	program := cli_raw_single_program(
		"p", command, []cli.Environment_Variable{variable}, nil,
	)
	output := new_cli_output()
	if exceeds_output {
		assert_panics(t, "maximum environment default exceeds fixed output", func() {
			cli.Print_Command(cli_output_reference(&output), program, command)
		})
		return
	}
	cli.Print_Command(cli_output_reference(&output), program, command)
}

// Test_Bounded_External_Metadata keeps omission, type, and failure metadata observable.
func Test_Bounded_External_Metadata(t *testing.T) {
	for _, size := range []int{1, 2, strings.TEXT_SIZE_MAXIMUM} {
		deprecated := cli_boundary_text(size, 'd')
		program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
			Label: "p",
			Environment_Variables: []cli.Environment_Variable{{
				Key: "E", Deprecated: cli.Deprecation(deprecated),
			}},
		})
		output := new_cli_output()
		cli.Print_Help(cli_output_reference(&output), program)
	}

	maximum := cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'd')
	cli_parse_boundary_environment(t, []cli.Environment_Variable{{
		Key: "E", Required: true,
		Description: cli.Description(maximum), Deprecated: cli.Deprecation(maximum),
	}}, nil)
	cli_bounded_external_types(t)
	cli_bounded_external_conversion_key(t)
	paths := make([]string, slices.COUNT_MAXIMUM)
	for index := range paths {
		paths[index] = "/X"
	}
	cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "p", Secrets: []cli.Secret{{Key: "X", Paths: paths}},
	})
	_, _, environment, _ := cli_constructor_declarations(slices.COUNT_MAXIMUM)
	cli_parse_boundary_environment(t, environment, nil)
	cli_parse_boundary_environment(t, []cli.Environment_Variable{
		{Key: "E", Deprecated: "d"},
		{Key: "F", Deprecated: "d"},
	}, []string{"E=v", "F=v"})
}

func cli_bounded_external_types(t *testing.T) {
	t.Helper()
	integer_program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "p",
		Environment_Variables: []cli.Environment_Variable{{
			Key: "E", Type: cli.External_Type_State{Value: cli.OPTION_TYPE_INTEGER},
		}},
	})
	integer_output := new_cli_output()
	cli.Print_Help(cli_output_reference(&integer_output), integer_program)

	member_size := strings.TEXT_SIZE_MAXIMUM - len("string()")
	member := cli_boundary_text(member_size, 'm')
	maximum_program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "p",
		Environment_Variables: []cli.Environment_Variable{{
			Key: "E", State: cli.Environment_Default{String: cli.Value_Text(member)},
			Enumeration: cli.External_Enumeration{String: []string{member}},
		}},
	})
	maximum_output := new_cli_output()
	assert_panics(t, "maximum external type exceeds help output", func() {
		cli.Print_Help(cli_output_reference(&maximum_output), maximum_program)
	})
}

func cli_bounded_external_conversion_key(t *testing.T) {
	t.Helper()
	key := cli_boundary_text(cli.EXTERNAL_KEY_SIZE_MAXIMUM, 'E')
	program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "p",
		Environment_Variables: []cli.Environment_Variable{{
			Key:         cli.External_Key(key),
			Type:        cli.External_Type_State{Value: cli.OPTION_TYPE_INTEGER},
			Allow_Empty: true,
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
	if !cli.Parse_Error_Present(result.Error) {
		t.Fatal("maximum conversion key accepted empty integer")
	}
	cli_bounded_external_conversion_enums(t, key, &parser)
	cli_bounded_external_conversion_values(t, &parser)
}

func cli_bounded_external_conversion_enums(
	t *testing.T, key string, parser *cli.Parser,
) {
	t.Helper()
	integer_enum_key := cli_boundary_text(cli.EXTERNAL_KEY_SIZE_MAXIMUM-len("0"), 'I')
	integer_enum := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "p",
		Environment_Variables: []cli.Environment_Variable{{
			Key:         cli.External_Key(integer_enum_key),
			Type:        cli.External_Type_State{Value: cli.OPTION_TYPE_INTEGER},
			Enumeration: cli.External_Enumeration{Integers: []int{0, 1}},
		}},
	})
	cli_test_program_parse(&integer_enum, parser, cli.Program_Parse_Input{
		Arguments: []string{"p"}, Environment: []string{integer_enum_key + "=2"},
	})
	string_enum := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "p",
		Environment_Variables: []cli.Environment_Variable{{
			Key: cli.External_Key(key), State: cli.Environment_Default{String: "x"},
			Enumeration: cli.External_Enumeration{String: []string{"x"}},
			Allow_Empty: true,
		}},
	})
	cli_test_program_parse(&string_enum, parser, cli.Program_Parse_Input{
		Arguments: []string{"p"}, Environment: []string{key + "="},
	})
}

func cli_bounded_external_conversion_values(t *testing.T, parser *cli.Parser) {
	t.Helper()
	for _, value_size := range [...]int{1, 2, cli.ENVIRONMENT_VALUE_SIZE_MAXIMUM} {
		value := cli_boundary_text(value_size, 'x')
		invalid := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
			Label: "p",
			Environment_Variables: []cli.Environment_Variable{{
				Key: "E",
				Type: cli.External_Type_State{
					Value: cli.OPTION_TYPE_BOOLEAN,
				},
			}},
		})
		cli_test_program_parse(&invalid, parser, cli.Program_Parse_Input{
			Arguments: []string{"p"}, Environment: []string{"E=" + value},
		})
	}
	two_key := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "p", Environment_Variables: []cli.Environment_Variable{{
			Key: "EE", Type: cli.External_Type_State{Value: cli.OPTION_TYPE_BOOLEAN},
		}},
	})
	cli_test_program_parse(&two_key, parser, cli.Program_Parse_Input{
		Arguments: []string{"p"}, Environment: []string{"EE=x"},
	})
	maximum_enum_value := cli_boundary_text(cli.ENVIRONMENT_VALUE_SIZE_MAXIMUM, 'x')
	maximum_enum := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "p",
		Environment_Variables: []cli.Environment_Variable{{
			Key: "E", State: cli.Environment_Default{String: "allowed"},
			Enumeration: cli.External_Enumeration{String: []string{"allowed"}},
		}},
	})
	cli_test_program_parse(&maximum_enum, parser, cli.Program_Parse_Input{
		Arguments: []string{"p"}, Environment: []string{"E=" + maximum_enum_value},
	})
}

// Test_Bounded_Environment_State separates declaration width from source multiplicity.
func Test_Bounded_Environment_State(t *testing.T) {
	text_sizes := [...]int{1, 2, cli.EXTERNAL_KEY_SIZE_MAXIMUM}
	for _, size := range text_sizes {
		text := cli_boundary_text(size, 'E')
		variable := cli.Environment_Variable{
			Key: cli.External_Key(text), Description: cli.Description(text),
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
	for _, value := range [...]int{
		bits.INTEGER_MINIMUM, bits.INTEGER_MAXIMUM, -1, 1, 2,
	} {
		variable := cli.New_Environment_Variable(
			cli.New_Environment_Variable_Input{
				Key: "E", Type: cli.OPTION_TYPE_INTEGER,
			},
		)
		cli_parse_boundary_environment(
			t, []cli.Environment_Variable{variable},
			[]string{"E=" + cli_decimal_text(value)},
		)
	}
}

// Test_Bounded_Environment_Sources keeps empty and maximum source maps observable.
func Test_Bounded_Environment_Sources(t *testing.T) {
	empty_program := cli_raw_single_program(
		"p", cli.Command{Label: "p"},
		[]cli.Environment_Variable{{Key: ""}}, nil,
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
	if !cli.Parse_Error_Present(empty_result.Error) {
		t.Fatal("empty environment key did not publish its source failure")
	}

	variables := make([]cli.Environment_Variable, slices.COUNT_MAXIMUM)
	raw := make([]string, slices.COUNT_MAXIMUM)
	sources := make(cli.Environment_Sources, slices.COUNT_MAXIMUM)
	for index := range variables {
		key := "E" + cli_decimal_text(index)
		variables[index] = cli.Environment_Variable{
			Key: cli.External_Key(key),
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
	if cli.Parse_Error_Present(result.Error) {
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
			cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
				Label: "p",
				Environment_Variables: []cli.Environment_Variable{{
					Key: cli.External_Key(key),
				}},
			})
		})
	}
}

func cli_bounded_environment_sources(t *testing.T) {
	t.Helper()
	value := cli_boundary_text(strings.TEXT_SIZE_MAXIMUM-len("E="), 'v')
	variable := cli.Environment_Variable{
		Key: "E", Required: true, Deprecated: "d",
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
			Key: "E", Allow_Empty: true, Deprecated: "d",
		}
	}
	cli_parse_boundary_environment(t, variables, []string{"E=v"})

	deprecation_size := strings.TEXT_SIZE_MAXIMUM - len(cli.ENVIRONMENT_WARNING_PREFIX) -
		len("E") - len(cli.ENVIRONMENT_WARNING_SUFFIX)
	cli_parse_boundary_environment(t, []cli.Environment_Variable{{
		Key: "E", Deprecated: cli.Deprecation(
			cli_boundary_text(deprecation_size, 'd'),
		),
	}}, []string{"E=v"})
	cli_parse_boundary_environment(t, []cli.Environment_Variable{{
		Key: "E", Deprecated: cli.Deprecation(
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
		Key: cli.External_Key(text), Description: cli.Description(metadata),
		State:    cli.Environment_Default{String: cli.Value_Text(text)},
		Required: true, Allow_Empty: true, Hidden: true,
		Deprecated: cli.Deprecation(metadata),
	}
	variables := make(cli.Resolved_Environment, count)
	for index := range variables {
		variables[index] = cli_resolved_environment(variable, cli.Environment_State{})
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
	if text_size > cli.SECRET_KEY_SIZE_MAXIMUM {
		text_size = cli.SECRET_KEY_SIZE_MAXIMUM
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
		Required: true, Allow_Empty: true, Hidden: true,
		Deprecated: cli.Deprecation(metadata),
	}
	secrets := make(cli.Resolved_Secrets, count)
	for index := range secrets {
		secrets[index] = cli.Resolved_Secret{Secret: secret}
	}
	if count == 0 {
		assert_panics(t, "empty secret lookup", func() {
			cli.Get_Secret(secrets, "")
		})
		return
	}
	if cli.Get_Secret(secrets, cli.Secret_Key(text)).Key != secret.Key {
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
	input := cli.New_Input{Mode: cli.PROGRAM_MODE_MULTICALL,
		Label: cli.Label(text), Description: cli.Description(text),
		Commands: commands, Global_Flags: flags,
		Environment_Variables: environment, Secrets: secrets,
	}
	if count == 0 {
		assert_panics(t, "a multicall program needs one command", func() {
			cli.New(input)
		})
		return
	}
	program := cli.New(input)
	if len(program.Selection.Commands) != count {
		t.Fatalf(
			"bounded multicall command count = %d, want %d",
			len(program.Selection.Commands), count,
		)
	}
}

func cli_bounded_single_external_constructor(t *testing.T, count int) {
	t.Helper()
	_, flags, environment, secrets := cli_constructor_declarations(count)
	arguments := make([]cli.Option, count)
	for index := range arguments {
		arguments[index] = cli.Option{
			Label: cli.Option_Label("a" + cli_decimal_text(index)),
		}
	}
	program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "p", Arguments: arguments, Flags: flags,
		Environment_Variables: environment, Secrets: secrets,
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
		options[index] = cli.Option{Label: "duplicate"}
	}
	text := cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'p')
	assert_panics(t, "duplicate maximum option declarations", func() {
		cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
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
			Label: cli.Option_Label("g" + suffix),
			Type:  cli.Option_Type_State{Value: cli.OPTION_TYPE_BOOLEAN},
			State: cli.Option_State{Is_Flag: true, Hidden: true},
		}
		environment[index] = cli.Environment_Variable{
			Key: cli.External_Key("E" + suffix), Hidden: true,
		}
		secrets[index] = cli.Secret{
			Key: cli.Secret_Key("S" + suffix), Paths: []string{"/S" + suffix},
			Hidden: true,
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
	command := program.Selection.Single_Commands.Command
	for _, size := range sizes {
		output := new_cli_output()
		cli_fill_output(&output, size)
		written := cli.Output_Write(cli_output_reference(&output), nil)
		if written != 0 {
			t.Fatalf("empty output write = %d", written)
		}
		cli.Output_Write_Text(cli_output_reference(&output), "")
		if len(cli.Output_Bytes(cli_output_reference(&output))) != size {
			t.Fatalf(
				"output size changed: got %d, want %d",
				len(cli.Output_Bytes(cli_output_reference(&output))), size,
			)
		}
		cli.Print_Deprecations(cli_output_reference(&output), nil)

		requested_output := new_cli_output()
		cli_fill_output(&requested_output, size)
		command_output := new_cli_output()
		cli_fill_output(&command_output, size)
		if size == strings.TEXT_SIZE_MAXIMUM {
			assert_panics(t, "requested help exceeds full output", func() {
				cli.Print_Requested_Help(
					cli_output_reference(&requested_output),
					program, command.Label,
				)
			})
			assert_panics(t, "command help exceeds full output", func() {
				cli.Print_Command(
					cli_output_reference(&command_output), program, command,
				)
			})
			continue
		}
		cli.Print_Requested_Help(
			cli_output_reference(&requested_output), program, command.Label,
		)
		cli.Print_Command(cli_output_reference(&command_output), program, command)
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
			Completion:           cli.Parser_Completion{Value: true},
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
	empty_program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE, Label: "p"})
	cli_bounded_slice_index(t)
	assert_panics(t, "empty process arguments", func() {
		var empty_parser cli.Parser
		cli.Program_Parse(empty_program, &empty_parser, cli.Program_Parse_Input{
			Failure_Storage: make([]byte, cli.FAILURE_SIZE_MAXIMUM),
		})
	})
	cli_parse_zero_option_command(t, empty_program)
	assert_panics(t, "zero positional storage", func() {
		var parser cli.Parser
		cli.Program_Parse(empty_program, &parser, cli.Program_Parse_Input{
			Arguments:       []string{"p", "unexpected"},
			Failure_Storage: make([]byte, cli.FAILURE_SIZE_MAXIMUM),
		})
	})

	variadic := cli.New_Option(cli.New_Option_Input{
		Label: "value", Type: cli.OPTION_TYPE_STRINGS,
	})
	program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
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
	cli.Program_Parse(empty_program, &help_parser, cli.Program_Parse_Input{
		Arguments:       help_arguments,
		Failure_Storage: make([]byte, cli.FAILURE_SIZE_MAXIMUM),
	})
	help_result, help_complete := cli.Parser_Done(&help_parser)
	if !help_complete {
		t.Fatal("maximum help arguments did not complete")
	}
	if !cli.Parse_Error_Equals(help_result.Error, cli.HELP_REQUESTED) {
		t.Fatalf("maximum help arguments: %v", help_result.Error)
	}
}

func cli_bounded_slice_index(t *testing.T) {
	t.Helper()
	program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "p", Arguments: []cli.Option{
			cli.New_Option(cli.New_Option_Input{Label: "a"}),
			cli.New_Option(cli.New_Option_Input{Label: "b"}),
			cli.New_Option(cli.New_Option_Input{
				Label: "values", Type: cli.OPTION_TYPE_STRINGS,
			}),
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
	if !cli.Parse_Error_Present(result.Error) {
		t.Fatal("three-argument command accepted missing scalars")
	}
}

// Test_Bounded_Positional_Values keeps empty, two-byte, and full scalar text observable.
func Test_Bounded_Positional_Values(t *testing.T) {
	values := [...]string{
		"", "xx", cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'x'),
	}
	for _, value := range values {
		argument := cli.New_Option(cli.New_Option_Input{Label: "value"})
		program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
			Label: "p", Arguments: []cli.Option{argument},
		})
		var parser cli.Parser
		cli.Program_Parse(program, &parser, cli.Program_Parse_Input{
			Arguments:         []string{"p", value},
			Command_Arguments: make(cli.Command_Argument_Storage, 1),
			Filled:            make([]bool, 1),
			Positionals:       make([]cli.Indexed_Token, 1),
			Failure_Storage:   make([]byte, cli.FAILURE_SIZE_MAXIMUM),
		})
		result, complete := cli.Parser_Done(&parser)
		if !complete {
			t.Fatal("bounded positional value did not complete")
		}
		if cli.Parse_Error_Present(result.Error) {
			t.Fatalf("bounded positional value: %v", result.Error)
		}
	}
}

func cli_parse_zero_option_command(t *testing.T, program cli.Program) {
	t.Helper()
	var parser cli.Parser
	cli.Program_Parse(program, &parser, cli.Program_Parse_Input{
		Arguments:       []string{"p"},
		Failure_Storage: make([]byte, cli.FAILURE_SIZE_MAXIMUM),
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("zero-option command did not complete")
	}
	if cli.Parse_Error_Present(result.Error) {
		t.Fatalf("zero-option command: %v", result.Error)
	}
}

func cli_parse_variadic_boundary(t *testing.T, program cli.Program, arguments []string) {
	t.Helper()
	var parser cli.Parser
	cli.Program_Parse(program, &parser, cli.Program_Parse_Input{
		Arguments:         arguments,
		Command_Arguments: make(cli.Command_Argument_Storage, 1),
		Filled:            make([]bool, 1),
		Positionals:       make([]cli.Indexed_Token, cli.PARSED_TOKEN_COUNT_MAXIMUM),
		Slice_Named:       make([]cli.Indexed_Token, cli.PARSED_TOKEN_COUNT_MAXIMUM),
		String_Values:     make([]string, cli.PARSED_TOKEN_COUNT_MAXIMUM),
		Integer_Values:    make([]int, cli.PARSED_TOKEN_COUNT_MAXIMUM),
		Failure_Storage:   make([]byte, cli.FAILURE_SIZE_MAXIMUM),
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("maximum variadic command did not complete")
	}
	if cli.Parse_Error_Present(result.Error) {
		t.Fatalf("maximum variadic command: %v", result.Error)
	}
	values := cli.Option_Strings(cli.Resolved_Options(result.Command.Arguments), "value")
	if len(values) != len(arguments)-1 {
		t.Fatalf("variadic value count = %d", len(values))
	}
}

func cli_fill_output(output *cli.Output, size int) {
	cli.Output_Write_Text(
		cli_output_reference(output),
		cli.Value_Text(cli_boundary_text(size, 'x')),
	)
}

func cli_bounded_hidden_flag_section(t *testing.T, program cli.Program) {
	t.Helper()
	flags := make([]cli.Option, slices.COUNT_MAXIMUM)
	for index := range flags {
		flags[index] = cli.Option{
			Label: "f", Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_BOOLEAN},
			State: cli.Option_State{Is_Flag: true, Hidden: true},
		}
	}
	command := cli.Command{Label: "p", Flags: flags}
	output := new_cli_output()
	cli.Print_Command(cli_output_reference(&output), program, command)
}

func cli_bounded_empty_flag_row(t *testing.T, program cli.Program) {
	t.Helper()
	command := cli.Command{
		Label: "p",
		Flags: []cli.Option{{
			Type:  cli.Option_Type_State{Value: cli.OPTION_TYPE_BOOLEAN},
			State: cli.Option_State{Is_Flag: true},
		}},
	}
	output := new_cli_output()
	cli.Print_Command(cli_output_reference(&output), program, command)
}

func cli_bounded_option_signatures(t *testing.T, program cli.Program) {
	t.Helper()
	empty_enum_output := new_cli_output()
	cli.Print_Command(cli_output_reference(&empty_enum_output), program, cli.Command{
		Label: "p", Arguments: []cli.Option{{
			Label: "a", Enumeration: cli.Option_Enumeration{String: []string{}},
		}},
	})
	minimum_output := new_cli_output()
	cli.Print_Command(cli_output_reference(&minimum_output), program, cli.Command{
		Label: "p", Arguments: []cli.Option{{
			Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_INTEGER},
		}},
	})

	maximum_label_size := strings.TEXT_SIZE_MAXIMUM - cli.OPTION_SIGNATURE_SIZE_MINIMUM
	maximum_label := cli_boundary_text(maximum_label_size, 'a')
	maximum_output := new_cli_output()
	assert_panics(t, "maximum option signature exceeds usage output", func() {
		cli.Print_Command(cli_output_reference(&maximum_output), program, cli.Command{
			Label: "p", Arguments: []cli.Option{{
				Label: cli.Option_Label(maximum_label),
				Type:  cli.Option_Type_State{Value: cli.OPTION_TYPE_INTEGER},
			}},
		})
	})

	two_output := new_cli_output()
	cli.Print_Command(cli_output_reference(&two_output), program, cli.Command{
		Label: "p", Arguments: []cli.Option{{
			Label: "a", Enumeration: cli.Option_Enumeration{String: []string{"xx"}},
		}},
	})
	maximum_member := cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'm')
	enum_output := new_cli_output()
	assert_panics(t, "maximum enum signature exceeds usage output", func() {
		cli.Print_Command(cli_output_reference(&enum_output), program, cli.Command{
			Label: "p", Arguments: []cli.Option{{
				Label: "a",
				Enumeration: cli.Option_Enumeration{
					String: []string{maximum_member},
				},
			}},
		})
	})
}

func cli_bounded_visible_flag_section(t *testing.T, program cli.Program) {
	t.Helper()
	flags := make([]cli.Option, slices.COUNT_MAXIMUM)
	for index := range flags {
		flags[index] = cli.Option{
			Label: "f", Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_BOOLEAN},
			State: cli.Option_State{Is_Flag: true},
		}
	}
	output := new_cli_output()
	assert_panics(t, "maximum visible flags exceed fixed output", func() {
		cli.Print_Command(
			cli_output_reference(&output), program,
			cli.Command{Label: "p", Flags: flags},
		)
	})
}

// Test_Bounded_Completion_Scope keeps every collection and index result observable.
func Test_Bounded_Completion_Scope(t *testing.T) {
	indices := [...]int{0, 1, 2, slices.FOUND_INDEX_MAXIMUM}
	for _, index := range indices {
		options := make([]cli.Option, index+1)
		options[index] = cli.Option{
			Label: "a", Enumeration: cli.Option_Enumeration{String: []string{"member"}},
		}
		cli_expect_completion(t, cli_completion_program(options, nil, nil), "a")
		cli_expect_completion(t, cli_completion_program(nil, options, nil), "a")
		cli_expect_completion(t, cli_completion_program(nil, nil, options), "a")
	}
	labels := [...]string{
		"a", "aa", cli_boundary_text(cli.COMPLETION_OPTION_LABEL_SIZE_MAXIMUM, 'a'),
	}
	for _, label := range labels {
		option := cli.Option{
			Label:       cli.Option_Label(label),
			Enumeration: cli.Option_Enumeration{String: []string{"member"}},
		}
		cli_expect_completion(
			t, cli_completion_program([]cli.Option{option}, nil, nil), label,
		)
	}
	program := cli_completion_program(nil, nil, nil)
	if candidates := cli_complete(program, []string{"p", "-="}); len(candidates) != 0 {
		t.Fatalf("empty completion label returned %v", candidates)
	}
	cli_bounded_completion_candidates(t, 1)
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
	option := cli.Option{
		Label: "a", Enumeration: cli.Option_Enumeration{String: members},
	}
	program := cli_completion_program([]cli.Option{option}, nil, nil)
	candidates := cli_complete(program, []string{"p", "-a="})
	if len(candidates) != count {
		t.Fatalf("completion candidate count = %d, want %d", len(candidates), count)
	}
	integers := make([]int, count)
	for index := range integers {
		integers[index] = 1
	}
	integer := cli.Option{
		Label: "a", Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_INTEGER},
		Enumeration: cli.Option_Enumeration{Integers: integers},
	}
	program = cli_completion_program([]cli.Option{integer}, nil, nil)
	candidates = cli_complete(program, []string{"p", "-a="})
	if len(candidates) != count {
		t.Fatalf("integer completion candidate count = %d, want %d", len(candidates), count)
	}
}

// Test_Completion_Caller_Bounds proves caller storage owns every completion result.
func Test_Completion_Caller_Bounds(t *testing.T) {
	var storage [cli.CANDIDATE_COUNT_MAXIMUM]cli.Candidate
	hidden_flag := cli.Option{
		Label: "h", Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_BOOLEAN},
		State: cli.Option_State{Is_Flag: true, Hidden: true},
	}
	single := cli_completion_program(nil, []cli.Option{hidden_flag}, nil)
	selector := cli.Program{
		Label: "p",
		Selection: cli.Program_Selection{
			Commands:   []cli.Command{{Label: "c", Hidden: true}},
			Help_Flags: cli.Help_Flags{},
		},
	}
	enum := cli_completion_program(
		[]cli.Option{{Enumeration: cli.Option_Enumeration{String: []string{"x"}}}},
		[]cli.Option{{
			Label: "a", Enumeration: cli.Option_Enumeration{String: []string{"x"}},
		}}, nil,
	)
	counts := [...]int{0, 1, 2, cli.CANDIDATE_COUNT_MAXIMUM}
	for _, count := range counts {
		cells := storage[:count]
		cli.Complete(selector, []string{"p", ""}, cells)
		empty_output := new_cli_output()
		cli.Handle_Completion(
			selector, []string{"p"}, cli_output_reference(&empty_output), cells,
		)
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
			populated_output := new_cli_output()
			cli.Handle_Completion(
				single, []string{"p"},
				cli_output_reference(&populated_output), cells,
			)
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
		Label: "p", Selection: cli.Program_Selection{
			Commands: commands, Help_Flags: cli.Help_Flags{},
		},
	}
	command_candidates := cli.Complete(selector, []string{"p", ""}, storage)
	if len(command_candidates) != slices.COUNT_MAXIMUM {
		t.Fatalf("command completion candidate count = %d", len(command_candidates))
	}
	arguments := make([]cli.Option, slices.COUNT_MAXIMUM)
	flags := make([]cli.Option, slices.COUNT_MAXIMUM)
	global_flags := make([]cli.Option, slices.COUNT_MAXIMUM)
	for index := range arguments {
		arguments[index] = cli.Option{Label: "a"}
		flags[index] = cli.Option{
			Label: "f", Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_BOOLEAN},
			State: cli.Option_State{Is_Flag: true},
		}
		global_flags[index] = cli.Option{
			Label: "g", Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_BOOLEAN},
			State: cli.Option_State{Is_Flag: true},
		}
	}
	program := cli.Program{
		Label: "p",
		Selection: cli.Program_Selection{
			Single_Commands: cli.Single_Commands{Command: cli.Command{
				Label: "p", Arguments: arguments, Flags: flags,
			}},
			Global_Flags: global_flags,
			Help_Flags: cli.Help_Flags{
				Short_Label: "h",
				Long_Label:  "help",
			},
			Mode: cli.Program_Mode{Value: cli.PROGRAM_MODE_SINGLE},
		},
	}
	candidates := cli.Complete(program, []string{"p", "-"}, storage)
	if len(candidates) != cli.CANDIDATE_COUNT_MAXIMUM {
		t.Fatalf("all completion candidate count = %d", len(candidates))
	}
}

// Test_Candidate_Render_Bounds keeps segmented size and output edges observable.
func Test_Candidate_Render_Bounds(t *testing.T) {
	texts := [...]string{
		"", "x", "xx", cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'x'),
	}
	for _, expected := range texts {
		candidate := cli.Candidate{Segments: cli.Candidate_Segments{
			Dash:   cli.Candidate_Dash(expected),
			Label:  cli.Candidate_Label(expected),
			Equals: cli.Candidate_Equals(expected),
			Value:  cli.Candidate_Value(expected),
		}}
		output := new_cli_output()
		if len(expected) == strings.TEXT_SIZE_MAXIMUM {
			cli.Candidate_Equal(candidate, "")
			assert_panics(t, "maximum candidate exceeds fixed output", func() {
				cli.Candidate_Write(cli_output_reference(&output), candidate)
			})
			continue
		}
		joined := expected + expected + expected + expected
		if !cli.Candidate_Equal(candidate, cli.Value_Text(joined)) {
			t.Fatalf("candidate did not equal %d-byte text", len(expected))
		}
		cli.Candidate_Write(cli_output_reference(&output), candidate)
	}
	maximum := cli.Candidate{}
	maximum.Segments = cli.Candidate_Segments{
		Dash:   cli.Candidate_Dash(texts[len(texts)-1]),
		Label:  cli.Candidate_Label(texts[len(texts)-1]),
		Equals: cli.Candidate_Equals(texts[len(texts)-1]),
		Value:  cli.Candidate_Value(texts[len(texts)-1]),
	}
	maximum.State.Integer = cli.Integer(bits.INTEGER_MINIMUM)
	maximum.State.Has_Integer = true
	cli.Candidate_Equal(maximum, "")
	for _, value := range [...]int{bits.INTEGER_MAXIMUM, -1, 1, 2} {
		candidate := cli.Candidate{State: cli.Candidate_State{
			Integer: cli.Integer(value), Has_Integer: true,
		}}
		cli.Candidate_Equal(candidate, "")
		output := new_cli_output()
		cli.Candidate_Write(cli_output_reference(&output), candidate)
	}
	for _, size := range [...]int{0, 1, 2, strings.TEXT_SIZE_MAXIMUM} {
		output := new_cli_output()
		cli_fill_output(&output, size)
		cli.Candidate_Write(cli_output_reference(&output), cli.Candidate{})
	}
}

// Test_Completion_Script_Output_Bounds proves scripts append only to caller output.
func Test_Completion_Script_Output_Bounds(t *testing.T) {
	program := cli_completion_program(nil, nil, nil)
	for _, size := range [...]int{0, 1, 2, strings.TEXT_SIZE_MAXIMUM} {
		output := new_cli_output()
		cli_fill_output(&output, size)
		if size == strings.TEXT_SIZE_MAXIMUM {
			assert_panics(t, "full completion script output", func() {
				cli.Completion_Script(
					program, "fish", cli_output_reference(&output),
				)
			})
			continue
		}
		err := cli.Completion_Script(
			program, "fish", cli_output_reference(&output),
		)
		if err != nil {
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
			arguments[index] = cli.Option{Label: "aaa"}
			flags[index] = cli.Option{
				Label: "aaa",
				Type:  cli.Option_Type_State{Value: cli.OPTION_TYPE_BOOLEAN},
				State: cli.Option_State{Is_Flag: true},
			}
			global_flags[index] = flags[index]
		}
		program := cli_completion_program(arguments, nil, nil)
		cli_parse_unknown_option(t, &program, "aab")
		program = cli_completion_program(nil, flags, nil)
		cli_parse_unknown_option(t, &program, "aab")
		program = cli_completion_program(nil, nil, global_flags)
		cli_parse_unknown_option(t, &program, "aab")
	}
	for _, target := range []string{
		"x", "xx", cli_boundary_text(cli.OPTION_LABEL_SIZE_MAXIMUM, 'x'),
	} {
		program := cli_completion_program(
			[]cli.Option{{Label: "aa"}}, nil, nil,
		)
		cli_parse_unknown_option(t, &program, target)
	}
	cli_bounded_command_suggestion()
}

func cli_parse_unknown_option(t *testing.T, program *cli.Program, label string) {
	t.Helper()
	var parser cli.Parser
	cli_test_program_parse(program, &parser, cli.Program_Parse_Input{
		Arguments: []string{"p", "-" + label},
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("unknown option parser did not publish")
	}
	if !cli.Parse_Error_Present(result.Error) {
		t.Fatal("unknown option parser accepted its label")
	}
}

func cli_bounded_command_suggestion() {
	maximum := cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'a')
	target := maximum[:len(maximum)-len("a")] + "b"
	program := cli.Program{
		Label: "p",
		Selection: cli.Program_Selection{
			Commands:   []cli.Command{{Label: cli.Label(maximum)}},
			Help_Flags: cli.Help_Flags{},
		},
	}
	var parser cli.Parser
	cli_test_program_parse(&program, &parser, cli.Program_Parse_Input{
		Arguments: []string{"p", target},
	})
	short := program
	short.Selection.Commands[0].Label = "aa"
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
	two_arguments := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "p", Arguments: []cli.Option{
			cli.New_Option(cli.New_Option_Input{Label: "a"}),
			cli.New_Option(cli.New_Option_Input{Label: "b"}),
		},
	})
	cli_complete(two_arguments, []string{"p", "x", ""})

	selected := cli.Program{
		Label: "p",
		Selection: cli.Program_Selection{
			Commands:   []cli.Command{{Label: "c"}},
			Help_Flags: cli.Help_Flags{},
		},
	}
	cli_complete(selected, []string{"p", "c", ""})
}

func cli_completion_program(
	arguments []cli.Option, flags []cli.Option, global_flags []cli.Option,
) (program cli.Program) {
	return cli.Program{
		Label: "p",
		Selection: cli.Program_Selection{
			Single_Commands: cli.Single_Commands{Command: cli.Command{
				Label: "p", Arguments: arguments, Flags: flags,
			}},
			Global_Flags: global_flags,
			Help_Flags:   cli.Help_Flags{},
			Mode:         cli.Program_Mode{Value: cli.PROGRAM_MODE_SINGLE},
		},
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
	option := cli_boundary_option(count, cli.OPTION_TYPE_BOOLEAN)
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
		Selection: cli.Program_Selection{
			Commands:        commands,
			Single_Commands: cli.Single_Commands{Command: command},
			Global_Flags:    options,
			Help_Flags: cli.Help_Flags{
				Hidden: 1,
			},
		},
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
	output := new_cli_output()
	cli.Output_Write_Text(
		cli_output_reference(&output),
		cli.Value_Text(cli_boundary_text(count, 'x')),
	)
	handle_completion(program, words, cli_output_reference(&output))
	cli.Output_Reset(cli_output_reference(&output))
	handle_completion(program, []string{
		string(program.Label), "__complete", string(program.Label), "",
	}, cli_output_reference(&output))
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
		Selection: cli.Program_Selection{
			Commands:   []cli.Command{{Label: "c"}},
			Help_Flags: cli.Help_Flags{},
			Mode:       cli.Program_Mode{Value: cli.PROGRAM_MODE_MULTICALL},
		},
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
		Selection: cli.Program_Selection{
			Commands:   commands,
			Help_Flags: cli.Help_Flags{},
			Mode:       cli.Program_Mode{Value: cli.PROGRAM_MODE_MULTICALL},
		},
	}
	assert_panics(t, "maximum completion targets exceed fixed script", func() {
		completion_script(maximum_targets, "fish")
	})
}

func cli_bounded_render_entrypoints(
	t *testing.T, program cli.Program, command cli.Command, count int,
) {
	t.Helper()
	output := new_cli_output()
	cli.Output_Write_Text(
		cli_output_reference(&output),
		cli.Value_Text(cli_boundary_text(count, 'x')),
	)
	if count == slices.COUNT_MAXIMUM {
		assert_panics(t, "maximum program header exceeds fixed output", func() {
			cli.Print_Help(cli_output_reference(&output), program)
		})
		cli.Output_Reset(cli_output_reference(&output))
		assert_panics(t, "maximum command signature exceeds fixed output", func() {
			cli.Print_Command(cli_output_reference(&output), program, command)
		})
		return
	}
	cli.Print_Help(cli_output_reference(&output), program)
	cli.Output_Reset(cli_output_reference(&output))
	cli.Print_Requested_Help(cli_output_reference(&output), program, "")
	cli.Output_Reset(cli_output_reference(&output))
	cli.Print_Requested_Help(cli_output_reference(&output), program, command.Label)
	cli.Output_Reset(cli_output_reference(&output))
	cli.Print_Command(cli_output_reference(&output), program, command)
}

// Test_Bounded_Flag_Metadata keeps validation and rendering on every text edge.
func Test_Bounded_Flag_Metadata(t *testing.T) {
	profiles := [...]struct{ Label_Size, Description_Size int }{
		{1, 1}, {2, 2}, {1, strings.TEXT_SIZE_MAXIMUM},
		{cli.OPTION_LABEL_SIZE_MAXIMUM, 1},
	}
	for _, profile := range profiles {
		label := cli_boundary_text(profile.Label_Size, 'f')
		description := cli_boundary_text(profile.Description_Size, 'd')
		flag := cli.New_Option(cli.New_Option_Input{
			Label: cli.Option_Label(label), Type: cli.OPTION_TYPE_BOOLEAN,
			Description: cli.Description(description), Is_Flag: true,
		})
		program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
			Label: "p", Flags: []cli.Option{flag},
		})
		command := program.Selection.Single_Commands.Command
		output := new_cli_output()
		exceeds_output := profile.Description_Size == strings.TEXT_SIZE_MAXIMUM
		if !exceeds_output {
			exceeds_output = profile.Label_Size == cli.OPTION_LABEL_SIZE_MAXIMUM
		}
		if exceeds_output {
			assert_panics(t, "maximum flag row exceeds fixed output", func() {
				cli.Print_Command(cli_output_reference(&output), program, command)
			})
			continue
		}
		cli.Print_Command(cli_output_reference(&output), program, command)
	}
}

// Test_Bounded_Flag_Rendered_State keeps each scalar and enum edge at the row renderer.
func Test_Bounded_Flag_Rendered_State(t *testing.T) {
	for _, value := range [...]int{
		bits.INTEGER_MINIMUM, bits.INTEGER_MAXIMUM, -1, 0, 1, 2,
	} {
		cli_render_flag_state(t, cli.Option{
			Label: "f", Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_INTEGER},
			State: cli.Option_State{Integer: cli.Integer(value), Is_Flag: true},
		}, false)
	}
	for _, value := range [...]string{
		"", "x", "xx", cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'v'),
	} {
		cli_render_flag_state(t, cli.Option{
			Label: "f", State: cli.Option_State{
				String: cli.Value_Text(value), Is_Flag: true,
			},
		}, len(value) == strings.TEXT_SIZE_MAXIMUM)
	}
	for _, count := range [...]int{0, 1, 2, slices.COUNT_MAXIMUM} {
		cli_render_flag_enumerations(t, count)
	}
}

func cli_render_flag_enumerations(t *testing.T, count int) {
	t.Helper()
	cli_render_flag_state(t, cli.Option{
		Label: "s", Enumeration: cli.Option_Enumeration{
			String: make([]string, count),
		}, State: cli.Option_State{Is_Flag: true},
	}, count == slices.COUNT_MAXIMUM)
	cli_render_flag_state(t, cli.Option{
		Label: "i", Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_INTEGER},
		Enumeration: cli.Option_Enumeration{Integers: make([]int, count)},
		State:       cli.Option_State{Is_Flag: true},
	}, count == slices.COUNT_MAXIMUM)
}

func cli_render_flag_state(t *testing.T, flag cli.Option, exceeds_output bool) {
	t.Helper()
	command := cli.Command{Label: "p", Flags: []cli.Option{flag}}
	program := cli_raw_single_program("p", command, nil, nil)
	output := new_cli_output()
	if exceeds_output {
		assert_panics(t, "bounded flag row exceeds fixed output", func() {
			cli.Print_Command(cli_output_reference(&output), program, command)
		})
		return
	}
	cli.Print_Command(cli_output_reference(&output), program, command)
}

// Test_Bounded_Underscored_Labels keeps every correction-text edge observable.
func Test_Bounded_Underscored_Labels(t *testing.T) {
	sizes := [...]int{1, 2, cli.OPTION_LABEL_SIZE_MAXIMUM}
	for _, size := range sizes {
		label := cli_boundary_text(size, '_')
		assert_panics(t, "underscored option label", func() {
			cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
				Label: "p",
				Flags: []cli.Option{{
					Label: cli.Option_Label(label),
					Type: cli.Option_Type_State{
						Value: cli.OPTION_TYPE_BOOLEAN,
					},
					State: cli.Option_State{Is_Flag: true},
				}},
			})
		})
	}
}

// Test_Bounded_Empty_Validation keeps constructor-only invalid minima observable.
func Test_Bounded_Empty_Validation(t *testing.T) {
	cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE})
	cli_bounded_empty_parse(t)
	assert_panics(t, "empty argument label", func() {
		cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
			Label: "p", Arguments: []cli.Option{{}},
		})
	})
	assert_panics(t, "empty flag label", func() {
		cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
			Label: "p", Flags: []cli.Option{{
				Type:  cli.Option_Type_State{Value: cli.OPTION_TYPE_BOOLEAN},
				State: cli.Option_State{Is_Flag: true},
			}},
		})
	})
	assert_panics(t, "empty command declarations", func() {
		cli.New(cli.New_Input{Label: "p"})
	})
	cli_bounded_empty_external_declarations(t)

	program := cli_raw_single_program("", cli.Command{Label: "p"}, nil, nil)
	var parser cli.Parser
	cli.Program_Parse(program, &parser, cli.Program_Parse_Input{
		Arguments:       []string{"p"},
		Failure_Storage: make([]byte, cli.FAILURE_SIZE_MAXIMUM),
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
	if !cli.Parse_Error_Present(unknown_result.Error) {
		t.Fatal("empty option label was accepted")
	}

	assert_panics(t, "empty option lookup", func() {
		cli.Get_Option(cli.Resolved_Options{}, "")
	})
	output := new_cli_output()
	if handle_completion(cli.Program{}, nil, cli_output_reference(&output)) {
		t.Fatal("empty completion request was handled")
	}
}

func cli_bounded_empty_external_declarations(t *testing.T) {
	t.Helper()
	assert_panics(t, "empty external key", func() {
		cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
			Label:                 "p",
			Environment_Variables: []cli.Environment_Variable{{Key: ""}},
		})
	})
	assert_panics(t, "empty string environment enum", func() {
		cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
			Label: "p",
			Environment_Variables: []cli.Environment_Variable{{
				Key: "E", Enumeration: cli.External_Enumeration{String: []string{}},
			}},
		})
	})
	assert_panics(t, "empty integer environment enum", func() {
		cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
			Label: "p",
			Environment_Variables: []cli.Environment_Variable{{
				Key: "E",
				Type: cli.External_Type_State{
					Value: cli.OPTION_TYPE_INTEGER,
				},
				Enumeration: cli.External_Enumeration{Integers: []int{}},
			}},
		})
	})
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
	if !cli.Parse_Error_Present(unexpected_result.Error) {
		t.Fatal("empty command accepted an unexpected argument")
	}

	deprecated_program := cli_raw_single_program(
		"", cli.Command{
			Deprecated: "d",
			Arguments: []cli.Option{{
				Label: "values",
				Type:  cli.Option_Type_State{Value: cli.OPTION_TYPE_STRINGS},
			}},
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
	if cli.Parse_Error_Present(deprecated_result.Error) {
		t.Fatalf("empty deprecated command: %v", deprecated_result.Error)
	}
}

// Test_Bounded_Named_Assignment keeps complete-token value and name maxima observable.
func Test_Bounded_Named_Assignment(t *testing.T) {
	value := cli_boundary_text(cli.NAMED_VALUE_SIZE_MAXIMUM, 'v')
	cli_parse_named_boundary(t, cli.New_Option(cli.New_Option_Input{
		Label: "x", Is_Flag: true,
	}), "-x="+value)
	for _, token := range []string{"-x=x", "-x=xx", `-x=""`, `-x="xx"`} {
		cli_parse_named_boundary(t, cli.New_Option(cli.New_Option_Input{
			Label: "x", Is_Flag: true,
		}), token)
	}
	label := cli_boundary_text(cli.OPTION_LABEL_SIZE_MAXIMUM, 'x')
	cli_parse_named_boundary(t, cli.New_Option(cli.New_Option_Input{
		Label: cli.Option_Label(label), Type: cli.OPTION_TYPE_BOOLEAN, Is_Flag: true,
	}), "-"+label)
	variadic := cli.New_Option(cli.New_Option_Input{
		Label: "value", Type: cli.OPTION_TYPE_STRINGS,
	})
	variadic_program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "p", Arguments: []cli.Option{variadic},
	})
	cli_parse_variadic_boundary(t, variadic_program, []string{"p", "-value=x"})
}

func cli_parse_named_boundary(t *testing.T, flag cli.Option, token string) {
	t.Helper()
	program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "p", Flags: []cli.Option{flag},
	})
	var parser cli.Parser
	cli.Program_Parse(program, &parser, cli.Program_Parse_Input{
		Arguments: []string{"p", token}, Command_Flags: make(cli.Command_Flag_Storage, 1),
		Filled:          make([]bool, 1),
		Failure_Storage: make([]byte, cli.FAILURE_SIZE_MAXIMUM),
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("named boundary parse did not complete")
	}
	if cli.Parse_Error_Present(result.Error) {
		t.Fatalf("named boundary parse: %v", result.Error)
	}
}

// Test_Bounded_Parse_Phases keeps maximum selection and named lookup indices observable.
func Test_Bounded_Parse_Phases(t *testing.T) {
	arguments := make([]cli.Option, slices.COUNT_MAXIMUM)
	for index := range arguments {
		arguments[index] = cli.Option{Label: "a"}
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
		Label: "p",
		Selection: cli.Program_Selection{
			Commands: commands, Help_Flags: cli.Help_Flags{},
		},
	}
	var selection_parser cli.Parser
	cli_test_program_parse(&selection_program, &selection_parser, cli.Program_Parse_Input{
		Arguments: []string{"p", "last"},
	})
	selection_result, selection_complete := cli.Parser_Done(&selection_parser)
	if !selection_complete {
		t.Fatal("maximum command selection did not complete")
	}
	if cli.Parse_Error_Present(selection_result.Error) {
		t.Fatalf("maximum command selection: %v", selection_result.Error)
	}

	empty_name_program := cli.Program{
		Label: "p",
		Selection: cli.Program_Selection{
			Commands:   []cli.Command{{Label: "a"}},
			Help_Flags: cli.Help_Flags{},
		},
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
			Label: cli.Option_Label("f" + cli_decimal_text(index)),
			Type:  cli.Option_Type_State{Value: cli.OPTION_TYPE_BOOLEAN},
			State: cli.Option_State{Is_Flag: true},
		}
	}
	cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE, Label: "p", Flags: flags})
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
				cli.New_Option(cli.New_Option_Input{
					Label: "f", Type: cli.OPTION_TYPE_BOOLEAN,
					Is_Flag: true, Deprecated: deprecated,
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
		if cli.Parse_Error_Present(result.Error) {
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
		flags[index] = cli.New_Option(cli.New_Option_Input{
			Label: cli.Option_Label(label), Type: cli.OPTION_TYPE_BOOLEAN,
			Is_Flag: true, Deprecated: "d",
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
	if cli.Parse_Error_Present(result.Error) {
		t.Fatalf("maximum deprecation warnings: %v", result.Error)
	}
}

// Deprecation rendering observes complete name and guidance text boundaries.
func Test_Bounded_Deprecation_Text(t *testing.T) {
	for _, size := range [...]int{1, 2, strings.TEXT_SIZE_MAXIMUM} {
		text := cli.Value_Text(cli_boundary_text(size, 'd'))
		warnings := cli.Resolved_Deprecation_Warnings{{
			Kind: cli.Warning_Kind{Value: cli.WARNING_KIND_COMMAND},
			Name: cli.Warning_Name{Value: text},
			Guidance: cli.Warning_Guidance{
				Value: cli.Deprecation(text),
			},
		}}
		output := new_cli_output()
		if size == strings.TEXT_SIZE_MAXIMUM {
			assert_panics(t, "maximum deprecation text", func() {
				cli.Print_Deprecations(cli_output_reference(&output), warnings)
			})
			continue
		}
		cli.Print_Deprecations(cli_output_reference(&output), warnings)
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
	flag := cli.New_Option(cli.New_Option_Input{
		Label: cli.Option_Label(label), String: cli.Value_Text(text),
		String_Enum: strings_enum, Description: cli.Description(text),
		Is_Flag: true, Hidden: true, Deprecated: cli.Deprecation(text),
	})
	cli.New_Option(cli.New_Option_Input{
		Label: cli.Option_Label(label), String_Enum: strings_enum,
		Description: cli.Description(text),
	})
	cli.New_Option(cli.New_Option_Input{
		Label: cli.Option_Label(label), Integer: cli.Integer(count),
		Integer_Enum: integers_enum, Description: cli.Description(text),
		Type: cli.OPTION_TYPE_INTEGER, Is_Flag: true, Hidden: true,
		Deprecated: cli.Deprecation(text),
	})
	cli.New_Option(cli.New_Option_Input{
		Label: cli.Option_Label(label), Integer_Enum: integers_enum,
		Type:        cli.OPTION_TYPE_INTEGER,
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
	if secret_key_size > cli.SECRET_KEY_SIZE_MAXIMUM {
		secret_key_size = cli.SECRET_KEY_SIZE_MAXIMUM
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
	variable := cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
		Key: cli.External_Key(key), Description: cli.Description(text),
		Type: cli.OPTION_TYPE_INTEGER, Integer: cli.Integer(count),
		Integer_Enum: integers_enum,
		Required:     true, Allow_Empty: true, Hidden: true,
		Deprecated: cli.Deprecation(text),
	})
	cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
		Key: cli.External_Key(key), Description: cli.Description(text),
		String: cli.Value_Text(text), String_Enum: strings_enum,
		Required: true, Allow_Empty: true, Hidden: true,
		Deprecated: cli.Deprecation(text),
	})
	cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
		Key: cli.External_Key(key), Description: cli.Description(text),
		String:   cli.Value_Text(text),
		Required: true, Allow_Empty: true, Hidden: true,
		Deprecated: cli.Deprecation(text),
	})
	cli.New_Secret(cli.New_Secret_Input{
		Paths: paths, Description: cli.Description(text), String_Enum: strings_enum,
		Required: true, Allow_Empty: true, Hidden: true,
		Deprecated: cli.Deprecation(text),
	})
	cli.New_Secret(cli.New_Secret_Input{
		Paths: paths, Description: cli.Description(text),
		Type: cli.OPTION_TYPE_INTEGER, Integer_Enum: integers_enum,
		Required: true, Allow_Empty: true, Hidden: true,
		Deprecated: cli.Deprecation(text),
	})
	cli.New_Secret(cli.New_Secret_Input{
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
	option := cli_boundary_option(count, cli.OPTION_TYPE_STRINGS)
	arguments := make([]cli.Option, count)
	resolved_arguments := make(cli.Resolved_Options, count)
	for index := range arguments {
		arguments[index] = option
		resolved_arguments[index] = cli_resolved_option(
			option, cli.Resolved_Option_State{},
		)
	}
	command := cli_boundary_command(count)
	command.Arguments = arguments
	program := cli_raw_single_program("p", command, nil, nil)
	program.Selection.Help_Flags = cli.Help_Flags{
		Hidden: 1,
	}
	cli_parse_boundary_command(t, &program, count, []string{"p"})
	cli_complete(program, []string{"p", "-"})
	cli_parse_help_context(t, command, count)
	if cli.Get_Option(resolved_arguments, option.Label).Label != option.Label {
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
	option := cli_boundary_option(count, cli.OPTION_TYPE_BOOLEAN)
	option.Label = cli.Option_Label(label)
	option.State.Is_Flag = true
	flags := make([]cli.Option, count)
	for index := range flags {
		flags[index] = option
	}
	command := cli_boundary_command(count)
	command.Arguments = nil
	command.Flags = flags
	program := cli_raw_single_program("p", command, nil, nil)
	program.Selection.Help_Flags = cli.Help_Flags{
		Hidden: 1,
	}
	original := flags[0]
	enum_label_size := label_size
	if enum_label_size == strings.TEXT_SIZE_MAXIMUM-len("-") {
		enum_label_size -= len("=")
	}
	enum_label := cli_boundary_text(enum_label_size, 'e')
	flags[0].Label = cli.Option_Label(enum_label)
	flags[0].Type.Value = cli.OPTION_TYPE_STRING
	flags[0].Enumeration.String = []string{"v"}
	flags[0].State.String = "v"
	cli_complete(program, []string{"p", "-" + enum_label + "="})
	flags[0] = original
	cli_parse_boundary_command(t, &program, count, []string{"p", "-" + label})
	cli_complete(program, []string{"p", "-"})
	cli_parse_help_context(t, command, count)
	cli_render_boundary_command(t, program, command)
}

func cli_boundary_option(count int, option_type cli.Option_Type) (option cli.Option) {
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
		Label: cli.Option_Label(label), Description: cli.Description(text),
		Type: cli.Option_Type_State{Value: option_type},
		State: cli.Option_State{
			String:  cli.Value_Text(text),
			Integer: cli.Integer(integer), Boolean: true,
			Is_Flag: true, Hidden: true, Deprecated: cli.Deprecation(text),
		},
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
		Deprecated: cli.Deprecation(text),
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
	command := program.Selection.Single_Commands.Command
	failure_size := count
	if failure_size > cli.FAILURE_SIZE_MAXIMUM {
		failure_size = cli.FAILURE_SIZE_MAXIMUM
	}
	parser := cli_boundary_parser(command, count, failure_size)
	done_parser := parser
	done_parser.Completion.Value = true
	if _, complete := cli.Parser_Done(&done_parser); !complete {
		t.Fatal("completed boundary parser did not publish")
	}
	cli.Program_Parse(*program, &parser, cli.Program_Parse_Input{
		Arguments:            arguments,
		Command_Arguments:    make(cli.Command_Argument_Storage, count),
		Command_Flags:        make(cli.Command_Flag_Storage, count),
		Filled:               make([]bool, count),
		Positionals:          make([]cli.Indexed_Token, count),
		Slice_Named:          make([]cli.Indexed_Token, count),
		String_Values:        make([]string, count),
		Integer_Values:       make([]int, count),
		Failure_Storage:      make([]byte, cli.FAILURE_SIZE_MAXIMUM),
		Deprecation_Warnings: make([]cli.Warning, cli.DEPRECATION_WARNING_COUNT_MAXIMUM),
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("bounded command parser did not complete")
	}
	if cli.Parse_Error_Present(result.Error) {
		t.Fatalf("bounded command parse: %v", result.Error)
	}
}

func cli_boundary_parser(
	command cli.Command, count int, failure_size int,
) (parser cli.Parser) {
	return cli.Parser{
		Publication: cli.Publication{
			Result: cli.Parse_Result{
				Command: cli.Resolved_Command{
					Parsed_Command: cli.Parsed_Command{
						Selected_Command: cli.Selected_Command{
							Label:       command.Label,
							Description: command.Description,
							Arguments: make(
								cli.Resolved_Arguments, count,
							),
							Flags:      make(cli.Resolved_Flags, count),
							Hidden:     command.Hidden,
							Deprecated: command.Deprecated,
						},
						Deprecation_Warnings: make(
							[]cli.Warning,
							cli_boundary_warning_count(count),
						),
					},
					Environment: make(cli.Resolved_Environment, count),
					Secrets:     make(cli.Resolved_Secrets, count),
				},
				Error: cli.Parse_Error{
					Storage: make(
						cli.Parse_Error_Storage, cli.FAILURE_SIZE_MAXIMUM,
					),
					Size: cli.Parse_Error_Size{
						Value: cli.Parse_Error_Size_Value(failure_size),
					},
				},
			},
			Secret_Errors: make([]cli.Secret_Failure, count),
			Environment_Warnings: make(
				[]cli.Warning, cli_boundary_warning_count(count),
			),
			Secret_Warnings: make([]cli.Warning, count),
			Secret_Count:    cli.Declaration_Count(count),
			Secret_Parsers:  make([]cli.Secret_Parser, count),
			Failure: cli.Failure{
				Storage: make([]byte, cli.FAILURE_SIZE_MAXIMUM),
				Size: cli.Failure_Size{
					Value: cli.Failure_Size_Value(failure_size),
				},
			},
		},
		Workspace: cli.Workspace{
			Command_Arguments: make(cli.Workspace_Arguments, count),
			Command_Flags:     make(cli.Workspace_Flags, count),
			Filled:            make([]bool, count),
			Positionals:       make([]cli.Indexed_Token, count),
			Slice_Named:       make([]cli.Indexed_Token, count),
		},
	}
}

func cli_parse_help_context(t *testing.T, command cli.Command, count int) {
	t.Helper()
	program := cli.Program{
		Label: "p",
		Selection: cli.Program_Selection{
			Commands: []cli.Command{command},
			Help_Flags: cli.Help_Flags{
				Hidden: 1,
			},
		},
	}
	cli_complete(program, []string{"p", ""})
	var parser cli.Parser
	cli.Program_Parse(program, &parser, cli.Program_Parse_Input{
		Arguments:         []string{"p", string(command.Label), "-help"},
		Command_Arguments: make(cli.Command_Argument_Storage, count),
		Command_Flags:     make(cli.Command_Flag_Storage, count),
		Filled:            make([]bool, count),
		Positionals:       make([]cli.Indexed_Token, count),
		Slice_Named:       make([]cli.Indexed_Token, count),
		Failure_Storage:   make([]byte, cli.FAILURE_SIZE_MAXIMUM),
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("bounded help context did not complete")
	}
	if !cli.Parse_Error_Equals(result.Error, cli.HELP_REQUESTED) {
		t.Fatalf("bounded help context: %v", result.Error)
	}
}

func cli_render_boundary_command(t *testing.T, program cli.Program, command cli.Command) {
	t.Helper()
	output := new_cli_output()
	cli.Print_Deprecations(cli_output_reference(&output), nil)
	cli.Output_Reset(cli_output_reference(&output))
	if len(command.Label)+len(command.Arguments) < strings.TEXT_SIZE_MAXIMUM {
		cli.Print_Requested_Help(cli_output_reference(&output), program, command.Label)
		cli.Output_Reset(cli_output_reference(&output))
		cli.Print_Command(cli_output_reference(&output), program, command)
		return
	}
	assert_panics(t, "bounded requested help exceeds fixed output", func() {
		cli.Print_Requested_Help(cli_output_reference(&output), program, command.Label)
	})
	cli.Output_Reset(cli_output_reference(&output))
	assert_panics(t, "bounded command exceeds fixed rendering output", func() {
		cli.Print_Command(cli_output_reference(&output), program, command)
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
	input := cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "program",
		Arguments: []cli.Option{
			cli.New_Option(cli.New_Option_Input{Label: "value"}),
		},
	}
	testify.Zero_Allocation(t, func() {
		program = cli.New(input)
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
		func() {
			program = cli.New(cli.New_Input{
				Mode: cli.PROGRAM_MODE_SINGLE, Label: "program",
			})
		},
		func() {
			program = cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_MULTICALL,
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
		func() { result = cli.New_Option(cli.New_Option_Input{Label: "x"}) },
		func() {
			result = cli.New_Option(cli.New_Option_Input{
				Label: "x", Type: cli.OPTION_TYPE_INTEGER,
			})
		},
		func() {
			result = cli.New_Option(cli.New_Option_Input{
				Label: "x", Type: cli.OPTION_TYPE_BOOLEAN,
			})
		},
		func() {
			result = cli.New_Option(cli.New_Option_Input{
				Label: "x", Type: cli.OPTION_TYPE_STRINGS,
			})
		},
		func() {
			result = cli.New_Option(cli.New_Option_Input{
				Label: "x", Type: cli.OPTION_TYPE_INTEGERS,
			})
		},
		func() { result = cli.New_Option(cli.New_Option_Input{Label: "x", Is_Flag: true}) },
		func() {
			result = cli.New_Option(cli.New_Option_Input{
				Label: "x", Type: cli.OPTION_TYPE_INTEGER, Is_Flag: true,
			})
		},
		func() {
			result = cli.New_Option(cli.New_Option_Input{
				Label: "x", Type: cli.OPTION_TYPE_BOOLEAN, Is_Flag: true,
			})
		},
		func() {
			result = cli.New_Option(cli.New_Option_Input{
				Label: "x", String: "x", String_Enum: strings, Is_Flag: true,
			})
		},
		func() {
			result = cli.New_Option(cli.New_Option_Input{
				Label: "x", Integer: 1, Integer_Enum: integers,
				Type: cli.OPTION_TYPE_INTEGER, Is_Flag: true,
			})
		},
		func() {
			result = cli.New_Option(cli.New_Option_Input{
				Label: "x", String_Enum: strings,
			})
		},
		func() {
			result = cli.New_Option(cli.New_Option_Input{
				Label: "x", Integer_Enum: integers, Type: cli.OPTION_TYPE_INTEGER,
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
				cli.New_Environment_Variable_Input{
					Key: "X", Type: cli.OPTION_TYPE_INTEGER,
				})
		},
		func() {
			environment = cli.New_Environment_Variable(
				cli.New_Environment_Variable_Input{
					Key: "X", Type: cli.OPTION_TYPE_BOOLEAN,
				})
		},
		func() {
			environment = cli.New_Environment_Variable(
				cli.New_Environment_Variable_Input{Key: "X"})
		},
		func() {
			environment = cli.New_Environment_Variable(
				cli.New_Environment_Variable_Input{
					Key: "X", String: "x", String_Enum: strings,
				})
		},
		func() {
			environment = cli.New_Environment_Variable(
				cli.New_Environment_Variable_Input{
					Key: "X", Type: cli.OPTION_TYPE_INTEGER,
					Integer: 1, Integer_Enum: integers,
				})
		},
		func() { secret = cli.New_Secret(cli.New_Secret_Input{Paths: paths}) },
		func() {
			secret = cli.New_Secret(cli.New_Secret_Input{
				Paths: paths, Type: cli.OPTION_TYPE_INTEGER,
			})
		},
		func() {
			secret = cli.New_Secret(cli.New_Secret_Input{
				Paths: paths, Type: cli.OPTION_TYPE_BOOLEAN,
			})
		},
		func() {
			secret = cli.New_Secret(cli.New_Secret_Input{
				Paths: paths, String_Enum: strings,
			})
		},
		func() {
			secret = cli.New_Secret(cli.New_Secret_Input{
				Paths: paths, Type: cli.OPTION_TYPE_INTEGER,
				Integer_Enum: integers,
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

// Secret I/O narrowing must not create closure state.
func Test_Secret_IO_Of_Zero_Allocation(t *testing.T) {
	var loop cli.Secret_IO
	testify.Zero_Allocation(t, func() { loop = cli.Secret_IO_Of(nbio.IO{}) })
	if loop.Status_Procedure == nil {
		t.Fatal("secret IO constructor result was discarded")
	}
}

// Rejection uses assertion diagnostics because malformed declarations are programmer errors.
func Test_Panic_Paths(t *testing.T) {
	invalid_secret := cli.New_Secret(cli.New_Secret_Input{
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
			cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
				Label: "program", Flags: []cli.Option{{Label: "bad_label"}},
			})
		}},
		{"option lookup", func() { cli.Get_Option(nil, "UNKNOWN") }},
		{"environment lookup", func() { cli.Get_Environment(nil, "UNKNOWN") }},
		{"secret lookup", func() { cli.Get_Secret(nil, "UNKNOWN") }},
		{"secret path", func() {
			cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
				Label: "program", Secrets: invalid_secrets,
			})
		}},
	}
	for _, one := range operations {
		t.Run(one.Name, func(t *testing.T) {
			panicked := cli_panic(one.Operation)
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
	var arguments [ALLOCATION_ARGUMENT_COUNT]cli.Resolved_Option
	var filled [ALLOCATION_OPTION_COUNT]bool
	var positionals [ALLOCATION_PARSE_TOKEN_COUNT]cli.Indexed_Token
	var slice_named [ALLOCATION_PARSE_TOKEN_COUNT]cli.Indexed_Token
	var failure_storage [cli.FAILURE_SIZE_MAXIMUM]byte
	program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "program",
		Arguments: []cli.Option{
			cli.New_Option(cli.New_Option_Input{Label: "value"}),
		},
	})
	input := cli.Program_Parse_Input{
		Arguments:         []string{"program", "content"},
		Command_Arguments: arguments[:],
		Filled:            filled[:],
		Positionals:       positionals[:],
		Slice_Named:       slice_named[:],
		Failure_Storage:   failure_storage[:],
	}
	testify.Zero_Allocation(t, func() {
		cli.Program_Parse(program, &parser, input)
	})
	if _, complete := cli.Parser_Done(&parser); !complete {
		t.Fatal("allocation parse did not publish")
	}
}

// Malicious syntax belongs inside allocation contract, not only successful parsing.
func Test_Program_Parse_Unknown_Option_Zero_Allocation(t *testing.T) {
	program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE, Label: "program"})
	var parser cli.Parser
	var result cli.Parse_Result
	var complete cli.Parser_Complete
	var global_flags [cli.HELP_FLAG_COUNT]cli.Resolved_Option
	var filled [cli.HELP_FLAG_COUNT]bool
	var positionals [ALLOCATION_PROGRAM_TOKEN_COUNT]cli.Indexed_Token
	var slice_named [ALLOCATION_PROGRAM_TOKEN_COUNT]cli.Indexed_Token
	var failure_storage [cli.FAILURE_SIZE_MAXIMUM]byte
	input := cli.Program_Parse_Input{
		Arguments: []string{"program", "-unknown"}, Global_Flags: global_flags[:],
		Filled: filled[:], Positionals: positionals[:], Slice_Named: slice_named[:],
		Failure_Storage: failure_storage[:],
	}
	testify.Zero_Allocation(t, func() {
		parser = cli.Parser{}
		cli.Program_Parse(program, &parser, input)
		result, complete = cli.Parser_Done(&parser)
	})
	if !complete {
		t.Fatal("unknown option parse did not publish")
	}
	if !cli.Parse_Error_Present(result.Error) {
		t.Fatal("unknown option parse lost its error")
	}
}

// Parse errors borrow caller storage so publication needs no byte-to-string conversion.
func Test_Parse_Error_Borrows_Failure_Storage(t *testing.T) {
	program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE, Label: "program"})
	input := cli_test_parse_input(program, []string{"program", "-unknown"})
	var parser cli.Parser
	cli.Program_Parse(program, &parser, input)
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("parse error did not publish")
	}
	if !cli.Parse_Error_Present(result.Error) {
		t.Fatal("parse error is absent")
	}
	if len(cli.Parse_Error_Bytes(result.Error)) == 0 {
		t.Fatal("parse error has no diagnostic bytes")
	}
}

type cli_parse_error_case struct {
	Name      string
	Program   cli.Program
	Arguments []string
}

func cli_parse_error_cases() (cases []cli_parse_error_case) {
	boolean_flag := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "program", Flags: []cli.Option{
			cli.New_Option(cli.New_Option_Input{
				Label: "flag", Type: cli.OPTION_TYPE_BOOLEAN, Is_Flag: true,
			}),
		},
	})
	string_flag := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "program", Flags: []cli.Option{
			cli.New_Option(cli.New_Option_Input{Label: "text", Is_Flag: true}),
		},
	})
	integer_flag := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "program", Flags: []cli.Option{
			cli.New_Option(cli.New_Option_Input{
				Label: "count", Type: cli.OPTION_TYPE_INTEGER, Is_Flag: true,
			}),
		},
	})
	string_enum, integer_enum := cli_parse_error_enums()
	missing_argument := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "program", Arguments: []cli.Option{
			cli.New_Option(cli.New_Option_Input{Label: "value"}),
		},
	})
	integer_argument := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "program", Arguments: []cli.Option{
			cli.New_Option(cli.New_Option_Input{
				Label: "count", Type: cli.OPTION_TYPE_INTEGER,
			}),
		},
	})
	integer_variadic := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "program", Arguments: []cli.Option{
			cli.New_Option(cli.New_Option_Input{
				Label: "count", Type: cli.OPTION_TYPE_INTEGERS,
			}),
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

func cli_parse_error_enums() (string_enum cli.Program, integer_enum cli.Program) {
	string_enum = cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "program", Flags: []cli.Option{
			cli.New_Option(cli.New_Option_Input{
				Label: "color", String: "always",
				String_Enum: []string{"always", "never"}, Is_Flag: true,
			}),
		},
	})
	integer_enum = cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "program", Flags: []cli.Option{
			cli.New_Option(cli.New_Option_Input{
				Label: "level", Integer: 1, Integer_Enum: []int{1, 2},
				Type: cli.OPTION_TYPE_INTEGER, Is_Flag: true,
			}),
		},
	})
	return string_enum, integer_enum
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
	testify.Zero_Allocation(t, func() {
		parser = cli.Parser{}
		cli.Program_Parse(program, &parser, input)
		result, complete = cli.Parser_Done(&parser)
	})
	if !complete {
		t.Fatal("rejected CLI parse did not publish")
	}
	if !cli.Parse_Error_Present(result.Error) {
		t.Fatal("rejected CLI parse lost its error")
	}
	if len(cli.Parse_Error_Bytes(result.Error)) == 0 {
		t.Fatal("rejected CLI parse lost its diagnostic")
	}
}

// Deprecated command and flag publication retains borrowed warning parts.
func Test_Program_Parse_Deprecation_Zero_Allocation(t *testing.T) {
	program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "program",
		Flags: []cli.Option{cli.New_Option(cli.New_Option_Input{
			Label: "old", Type: cli.OPTION_TYPE_BOOLEAN,
			Is_Flag: true, Deprecated: "use new",
		})},
	})
	program.Selection.Single_Commands.Command.Deprecated = "use next"
	var parser cli.Parser
	var flags [ALLOCATION_ARGUMENT_COUNT]cli.Resolved_Option
	var filled [ALLOCATION_OPTION_COUNT]bool
	var positionals [ALLOCATION_PARSE_TOKEN_COUNT]cli.Indexed_Token
	var slice_named [ALLOCATION_PARSE_TOKEN_COUNT]cli.Indexed_Token
	var warnings [ALLOCATION_PARSE_TOKEN_COUNT]cli.Warning
	var failure_storage [cli.FAILURE_SIZE_MAXIMUM]byte
	input := cli.Program_Parse_Input{
		Arguments: []string{"program", "-old"}, Command_Flags: flags[:],
		Filled: filled[:], Positionals: positionals[:], Slice_Named: slice_named[:],
		Deprecation_Warnings: warnings[:], Failure_Storage: failure_storage[:],
	}
	testify.Zero_Allocation(t, func() {
		cli.Program_Parse(program, &parser, input)
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
	program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE, Label: "program"})
	var parser cli.Parser
	var global_flags [cli.HELP_FLAG_COUNT]cli.Resolved_Option
	var filled [cli.HELP_FLAG_COUNT]bool
	var positionals [ALLOCATION_PROGRAM_TOKEN_COUNT]cli.Indexed_Token
	var slice_named [ALLOCATION_PROGRAM_TOKEN_COUNT]cli.Indexed_Token
	var string_values [ALLOCATION_PARSE_TOKEN_COUNT]string
	var integer_values [ALLOCATION_PARSE_TOKEN_COUNT]int
	var failure_storage [cli.FAILURE_SIZE_MAXIMUM]byte
	input := cli.Program_Parse_Input{
		Arguments:     []string{"program"},
		Global_Flags:  global_flags[:],
		Filled:        filled[:],
		Positionals:   positionals[:],
		Slice_Named:   slice_named[:],
		String_Values: string_values[:], Integer_Values: integer_values[:],
		Failure_Storage: failure_storage[:],
	}
	testify.Zero_Allocation(t, func() {
		cli.Program_Parse(program, &parser, input)
	})
	if _, complete := cli.Parser_Done(&parser); !complete {
		t.Fatal("empty allocation parse did not publish")
	}
}

// Environment parsing must use caller maps and publication slices.
func Test_Program_Parse_Environment_Zero_Allocation(t *testing.T) {
	program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "program",
		Environment_Variables: []cli.Environment_Variable{
			cli.New_Environment_Variable(
				cli.New_Environment_Variable_Input{
					Key: "E", Deprecated: "use NEW_E",
				},
			),
		},
	})
	var parser cli.Parser
	var environment_values [ALLOCATION_ARGUMENT_COUNT]cli.Resolved_Environment_Variable
	var environment_warnings [ALLOCATION_ARGUMENT_COUNT]cli.Warning
	var integer_values [ALLOCATION_ARGUMENT_COUNT]int
	var deprecation_warnings [ALLOCATION_ARGUMENT_COUNT]cli.Warning
	var failure_storage [cli.FAILURE_SIZE_MAXIMUM]byte
	environment_sources := make(cli.Environment_Sources, 1)
	input := cli.Program_Parse_Input{
		Arguments: []string{"program"}, Environment: []string{"E=value"},
		Environment_Values:   environment_values[:],
		Environment_Warnings: environment_warnings[:],
		Environment_Sources:  environment_sources,
		Integer_Values:       integer_values[:],
		Deprecation_Warnings: deprecation_warnings[:],
		Failure_Storage:      failure_storage[:],
	}
	testify.Zero_Allocation(t, func() {
		cli.Program_Parse(program, &parser, input)
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("environment allocation parse did not publish")
	}
	if cli.Parse_Error_Present(result.Error) {
		t.Fatalf("environment allocation parse: %v", result.Error)
	}
}

type cli_environment_error_case struct {
	Name        string
	Program     cli.Program
	Environment []string
}

func cli_environment_error_cases() (cases []cli_environment_error_case) {
	required := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "program", Environment_Variables: []cli.Environment_Variable{
			cli.New_Environment_Variable(
				cli.New_Environment_Variable_Input{
					Key: "VALUE", Required: true,
				},
			),
		},
	})
	integer := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "program", Environment_Variables: []cli.Environment_Variable{
			cli.New_Environment_Variable(
				cli.New_Environment_Variable_Input{
					Key: "VALUE", Type: cli.OPTION_TYPE_INTEGER,
				},
			),
		},
	})
	boolean := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "program", Environment_Variables: []cli.Environment_Variable{
			cli.New_Environment_Variable(
				cli.New_Environment_Variable_Input{
					Key: "VALUE", Type: cli.OPTION_TYPE_BOOLEAN,
				},
			),
		},
	})
	string_enum := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "program", Environment_Variables: []cli.Environment_Variable{
			cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
				Key: "VALUE", String: "always",
				String_Enum: []string{"always", "never"},
			}),
		},
	})
	integer_enum := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "program", Environment_Variables: []cli.Environment_Variable{
			cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
				Key: "VALUE", Type: cli.OPTION_TYPE_INTEGER,
				Integer: 1, Integer_Enum: []int{1, 2},
			}),
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
	return cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "program", Environment_Variables: []cli.Environment_Variable{
			cli.New_Environment_Variable(
				cli.New_Environment_Variable_Input{
					Key: "FIRST", Required: true,
				},
			),
			cli.New_Environment_Variable(
				cli.New_Environment_Variable_Input{
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
	testify.Zero_Allocation(t, func() {
		parser = cli.Parser{}
		cli.Program_Parse(program, &parser, input)
		result, complete = cli.Parser_Done(&parser)
	})
	if !complete {
		t.Fatal("rejected environment parse did not publish")
	}
	if !cli.Parse_Error_Present(result.Error) {
		t.Fatal("rejected environment parse lost its error")
	}
	if len(cli.Parse_Error_Bytes(result.Error)) == 0 {
		t.Fatal("rejected environment parse lost its diagnostic")
	}
}

// Variadic parsing must publish a view over caller element storage.
func Test_Program_Parse_Variadic_Zero_Allocation(t *testing.T) {
	program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "program",
		Arguments: []cli.Option{
			cli.New_Option(cli.New_Option_Input{
				Label: "value", Type: cli.OPTION_TYPE_STRINGS,
			}),
		},
	})
	var parser cli.Parser
	var arguments [ALLOCATION_ARGUMENT_COUNT]cli.Resolved_Option
	var filled [ALLOCATION_OPTION_COUNT]bool
	var positionals [ALLOCATION_ARGUMENT_COUNT]cli.Indexed_Token
	var slice_named [ALLOCATION_ARGUMENT_COUNT]cli.Indexed_Token
	var string_values [ALLOCATION_ARGUMENT_COUNT]string
	var deprecation_warnings [ALLOCATION_PARSE_TOKEN_COUNT]cli.Warning
	var failure_storage [cli.FAILURE_SIZE_MAXIMUM]byte
	input := cli.Program_Parse_Input{
		Arguments: []string{"program", "content"}, Command_Arguments: arguments[:],
		Filled: filled[:], Positionals: positionals[:], Slice_Named: slice_named[:],
		String_Values:        string_values[:],
		Deprecation_Warnings: deprecation_warnings[:],
		Failure_Storage:      failure_storage[:],
	}
	testify.Zero_Allocation(t, func() {
		cli.Program_Parse(program, &parser, input)
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("variadic allocation parse did not publish")
	}
	if cli.Parse_Error_Present(result.Error) {
		t.Fatalf("variadic allocation parse: %v", result.Error)
	}
}

// Secret parsing must submit and retire through stable caller runners and buffers.
func Test_Program_Parse_Secret_Zero_Allocation(t *testing.T) {
	program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "program",
		Secrets: []cli.Secret{
			cli.New_Secret(cli.New_Secret_Input{
				Paths: []string{"/TOKEN"}, Required: true,
				Deprecated: "use NEW_TOKEN",
			}),
		},
	})
	var parser cli.Parser
	var secret_values [ALLOCATION_ARGUMENT_COUNT]cli.Resolved_Secret
	var secret_errors [ALLOCATION_ARGUMENT_COUNT]cli.Secret_Failure
	var secret_warnings [ALLOCATION_ARGUMENT_COUNT]cli.Warning
	var deprecation_warnings [ALLOCATION_ARGUMENT_COUNT]cli.Warning
	var secret_parsers [ALLOCATION_ARGUMENT_COUNT]cli.Secret_Parser
	var secret_buffer [cli.SECRET_BUFFER_BYTES_MAX]byte
	secret_buffers := [ALLOCATION_ARGUMENT_COUNT]cli.Secret_Bytes{secret_buffer[:]}
	var path_failure [ALLOCATION_ARGUMENT_COUNT]cli.Path_Failure
	path_failures := [ALLOCATION_ARGUMENT_COUNT]cli.Path_Failures{path_failure[:]}
	var failure_storage [cli.FAILURE_SIZE_MAXIMUM]byte
	input := cli.Program_Parse_Input{
		Arguments: []string{"program"}, Loop: cli_secret_boundary_loop(),
		Secret_Values: secret_values[:], Secret_Errors: secret_errors[:],
		Secret_Warnings: secret_warnings[:], Secret_Parsers: secret_parsers[:],
		Secret_Buffers: secret_buffers[:], Secret_Path_Failures: path_failures[:],
		Deprecation_Warnings: deprecation_warnings[:],
		Failure_Storage:      failure_storage[:],
	}
	testify.Zero_Allocation(t, func() {
		cli.Program_Parse(program, &parser, input)
	})
	result, complete := cli.Parser_Done(&parser)
	if !complete {
		t.Fatal("secret allocation parse did not publish")
	}
	if cli.Parse_Error_Present(result.Error) {
		t.Fatalf("secret allocation parse: %v", result.Error)
	}
}

// Secret rejection must preserve ordered path causes without heap-backed error trees.
func Test_Program_Parse_Secret_Errors_Zero_Allocation(t *testing.T) {
	program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "program",
		Secrets: []cli.Secret{
			cli.New_Secret(cli.New_Secret_Input{
				Paths: []string{"/missing/MISSING"}, Required: true,
			}),
			cli.New_Secret(cli.New_Secret_Input{
				Paths: []string{"/secrets/DIRECTORY"}, Required: true,
			}),
			cli.New_Secret(cli.New_Secret_Input{
				Paths: []string{"/secrets/OVERFLOW"}, Required: true,
			}),
			cli.New_Secret(cli.New_Secret_Input{
				Paths: []string{"/secrets/NUMBER"}, Type: cli.OPTION_TYPE_INTEGER,
				Required: true,
			}),
			cli.New_Secret(cli.New_Secret_Input{
				Paths: []string{"/secrets/NEGATIVE"}, Required: true,
			}),
			cli.New_Secret(cli.New_Secret_Input{
				Paths: []string{"/secrets/READ_OVERFLOW"}, Required: true,
			}),
			cli.New_Secret(cli.New_Secret_Input{
				Paths:       []string{"/secrets/BOUNDARY_ENUM"},
				String_Enum: []string{"allowed"}, Required: true,
			}),
			cli.New_Secret(cli.New_Secret_Input{
				Paths: []string{"/X"}, Type: cli.OPTION_TYPE_INTEGER,
				Required: true,
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
	testify.Zero_Allocation(t, func() {
		parser = cli.Parser{}
		cli.Program_Parse(program, &parser, input)
		result, complete = cli.Parser_Done(&parser)
	})
	if !complete {
		t.Fatal("rejected secret parse did not publish")
	}
	if !cli.Parse_Error_Present(result.Error) {
		t.Fatal("rejected secret parse lost its error")
	}
	if len(cli.Parse_Error_Bytes(result.Error)) == 0 {
		t.Fatal("rejected secret parse lost its diagnostic")
	}
}

// Injected I/O failures must render without heap-backed wrapping.
func Test_Program_Parse_Secret_Injected_Errors_Zero_Allocation(t *testing.T) {
	program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "program", Secrets: []cli.Secret{
			cli.New_Secret(cli.New_Secret_Input{
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
			testify.Zero_Allocation(t, func() {
				parser = cli.Parser{}
				cli.Program_Parse(program, &parser, input)
				result, complete = cli.Parser_Done(&parser)
			})
			if !complete {
				t.Fatal("injected secret failure did not publish")
			}
			if !cli.Parse_Error_Present(result.Error) {
				t.Fatal("injected secret failure did not publish")
			}
			if len(cli.Parse_Error_Bytes(result.Error)) == 0 {
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

var cli_secret_injected_error = errors.New(
	cli_boundary_text(strings.TEXT_SIZE_MAXIMUM, 'e'),
)

func cli_secret_injected_name(stage cli_secret_injected_stage) (name string) {
	switch stage {
	case CLI_SECRET_INJECTED_STATUS:
		name = "status"
	case CLI_SECRET_INJECTED_OPEN:
		name = "open"
	case CLI_SECRET_INJECTED_READ:
		name = "read"
	case CLI_SECRET_INJECTED_CLOSE:
		name = "close"
	}
	if name == "" {
		return "unknown"
	}
	return name
}

func cli_secret_injected_loop(stage cli_secret_injected_stage) (loop cli.Secret_IO) {
	loop.Status_Procedure = cli_secret_injected_status_procedure
	loop.Open_Procedure = cli_secret_injected_open_procedure
	loop.Read_Procedure = cli_secret_injected_read_procedure
	loop.Close_Procedure = cli_secret_injected_close_procedure
	switch stage {
	case CLI_SECRET_INJECTED_STATUS:
		loop.Status_Procedure = cli_secret_injected_status_failure
	case CLI_SECRET_INJECTED_OPEN:
		loop.Open_Procedure = cli_secret_injected_open_failure
	case CLI_SECRET_INJECTED_READ:
		loop.Read_Procedure = cli_secret_injected_read_failure
	case CLI_SECRET_INJECTED_CLOSE:
		loop.Close_Procedure = cli_secret_injected_close_failure
	}
	return loop
}

func cli_secret_injected_status_procedure(
	_ nbio.IO, _ cli.Resolved_Secret_Path,
) (status nbio.File_Status, operation_err error) {
	status.Exists = true
	status.Size = 1
	return status, nil
}

func cli_secret_injected_status_failure(
	_ nbio.IO, _ cli.Resolved_Secret_Path,
) (status nbio.File_Status, operation_err error) {
	return status, cli_secret_injected_error
}

func cli_secret_injected_open_procedure(
	_ nbio.IO, completion nbio.Completion_Handle,
	_ cli.Resolved_Secret_Path, callback nbio.Callback,
) {
	completion.Data = 1
	completion.Error = nil
	callback(completion)
}

func cli_secret_injected_open_failure(
	_ nbio.IO, completion nbio.Completion_Handle,
	_ cli.Resolved_Secret_Path, callback nbio.Callback,
) {
	completion.Data = 1
	completion.Error = cli_secret_injected_error
	callback(completion)
}

func cli_secret_injected_read_procedure(
	_ nbio.IO, completion nbio.Completion_Handle, _ nbio.File,
	buffer cli.Secret_Buffer, callback nbio.Callback,
) {
	buffer[0] = 'x'
	completion.Data = 1
	completion.Error = nil
	callback(completion)
}

func cli_secret_injected_read_failure(
	_ nbio.IO, completion nbio.Completion_Handle, _ nbio.File,
	buffer cli.Secret_Buffer, callback nbio.Callback,
) {
	buffer[0] = 'x'
	completion.Data = 1
	completion.Error = cli_secret_injected_error
	callback(completion)
}

func cli_secret_injected_close_procedure(
	_ nbio.IO, completion nbio.Completion_Handle,
	_ nbio.File, callback nbio.Callback,
) {
	completion.Error = nil
	callback(completion)
}

func cli_secret_injected_close_failure(
	_ nbio.IO, completion nbio.Completion_Handle,
	_ nbio.File, callback nbio.Callback,
) {
	completion.Error = cli_secret_injected_error
	callback(completion)
}

// Completion keeps candidates and rendered bytes in caller-owned fixed storage.
func Test_Completion_Zero_Allocation(t *testing.T) {
	program := cli_completion_program(
		nil, []cli.Option{{
			Label: "a", Enumeration: cli.Option_Enumeration{String: []string{"x"}},
		}}, nil,
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
	output := new_cli_output()
	testify.Zero_Allocation(t, func() {
		cli.Output_Reset(cli_output_reference(&output))
		cli.Candidate_Write(cli_output_reference(&output), candidate)
	})
	if string(cli.Output_Bytes(cli_output_reference(&output))) != "-a=x" {
		t.Fatalf(
			"allocation candidate output = %q",
			cli.Output_Bytes(cli_output_reference(&output)),
		)
	}
	var script_error error
	testify.Zero_Allocation(t, func() {
		cli.Output_Reset(cli_output_reference(&output))
		script_error = cli.Completion_Script(program, "fish", cli_output_reference(&output))
	})
	if script_error != nil {
		t.Fatalf("allocation completion script: %v", script_error)
	}
	args := []string{"program", "__complete", "program", "-a="}
	var handled cli.Boolean
	testify.Zero_Allocation(t, func() {
		cli.Output_Reset(cli_output_reference(&output))
		handled = cli.Handle_Completion(
			program, args, cli_output_reference(&output), storage[:],
		)
	})
	if !handled {
		t.Fatal("allocation completion request was not handled")
	}
}

// Public output operations share one fixed caller buffer through every rendering form.
func Test_Output_Operations_Zero_Allocation(t *testing.T) {
	flag := cli.New_Option(cli.New_Option_Input{
		Label: "name", String: "value", Is_Flag: true,
	})
	string_members := []string{"x"}
	integer_members := []int{1}
	program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE,
		Label: "program", Flags: []cli.Option{flag},
		Environment_Variables: []cli.Environment_Variable{
			cli.New_Environment_Variable(cli.New_Environment_Variable_Input{
				Key: "ENV", String: "x", String_Enum: string_members,
			}),
		},
		Secrets: []cli.Secret{
			cli.New_Secret(cli.New_Secret_Input{
				Paths: []string{"/SECRET"}, Type: cli.OPTION_TYPE_INTEGER,
				Integer_Enum: integer_members,
			}),
		},
	})
	command := program.Selection.Single_Commands.Command
	warnings := cli.Resolved_Deprecation_Warnings{{
		Guidance: cli.Warning_Guidance{Value: "deprecated"},
	}}
	source := []byte("x")
	output := new_cli_output()
	var bytes strings.Bytes
	var written strings.Size_Value
	operations := [...]func(){
		func() { cli.Output_Reset(cli_output_reference(&output)) },
		func() {
			cli.Output_Reset(cli_output_reference(&output))
			cli.Output_Write_Text(cli_output_reference(&output), "x")
		},
		func() {
			cli.Output_Reset(cli_output_reference(&output))
			written = cli.Output_Write(
				cli_output_reference(&output), strings.Bytes(source),
			)
		},
		func() { bytes = cli.Output_Bytes(cli_output_reference(&output)) },
		func() {
			cli.Output_Reset(cli_output_reference(&output))
			cli.Print_Help(cli_output_reference(&output), program)
		},
		func() {
			cli.Output_Reset(cli_output_reference(&output))
			cli.Print_Command(cli_output_reference(&output), program, command)
		},
		func() {
			cli.Output_Reset(cli_output_reference(&output))
			cli.Print_Requested_Help(
				cli_output_reference(&output), program, command.Label,
			)
		},
		func() {
			cli.Output_Reset(cli_output_reference(&output))
			cli.Print_Deprecations(cli_output_reference(&output), warnings)
		},
	}
	for _, operation := range operations {
		testify.Zero_Allocation(t, operation)
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
	options := cli_lookup_options()
	environment := cli.New_Environment_Variable(
		cli.New_Environment_Variable_Input{Key: "ENV", String: "x"},
	)
	secret := cli.New_Secret(cli.New_Secret_Input{Paths: []string{"/SECRET"}})
	environment_values := cli.Resolved_Environment{cli_resolved_environment(
		environment, cli.Environment_State{
			String: environment.State.String,
		},
	)}
	secret_values := cli.Resolved_Secrets{{Secret: secret}}
	var option cli.Resolved_Option
	var variable cli.Resolved_Environment_Variable
	var found_secret cli.Resolved_Secret
	var text cli.Value_Text
	var integer cli.Integer
	var boolean cli.Boolean
	var texts cli.String_Values
	var integers cli.Integer_Values
	var external_text cli.Value_Text
	var secret_text cli.Secret_Value_Bytes
	operations := [...]func(){
		func() { option = cli.Get_Option(options, "s") },
		func() { text = cli.Option_String(options, "s") },
		func() { integer = cli.Option_Integer(options, "i") },
		func() { boolean = cli.Option_Boolean(options, "b") },
		func() { texts = cli.Option_Strings(options, "ss") },
		func() { integers = cli.Option_Integers(options, "ii") },
		func() { variable = cli.Get_Environment(environment_values, "ENV") },
		func() { external_text = cli.Environment_String(environment_values, "ENV") },
		func() { found_secret = cli.Get_Secret(secret_values, "SECRET") },
		func() { secret_text = cli.Secret_String(secret_values, "SECRET") },
	}
	for _, operation := range operations {
		testify.Zero_Allocation(t, operation)
	}
	switch {
	case option.Label == "":
		t.Fatal("lookup result was discarded")
	case variable.Key == "":
		t.Fatal("lookup result was discarded")
	case found_secret.Key == "":
		t.Fatal("lookup result was discarded")
	case text == "":
		t.Fatal("text accessor result was discarded")
	case integer == 0:
		t.Fatal("integer accessor result was discarded")
	case !bool(boolean):
		t.Fatal("Boolean accessor result was discarded")
	case len(texts) != 0:
		t.Fatal("text collection accessor changed empty state")
	case len(integers) != 0:
		t.Fatal("integer collection accessor changed empty state")
	case external_text == "":
		t.Fatal("environment accessor result was discarded")
	case len(secret_text) != 0:
		t.Fatal("secret accessor changed empty state")
	}
}

func cli_lookup_options() (options cli.Resolved_Options) {
	return cli.Resolved_Options{
		{
			Label: "s", Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_STRING},
			State: cli.Resolved_Option_State{String: "x", Is_Flag: true},
		},
		{
			Label: "i", Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_INTEGER},
			State: cli.Resolved_Option_State{Integer: 1, Is_Flag: true},
		},
		{
			Label: "b", Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_BOOLEAN},
			State: cli.Resolved_Option_State{Boolean: true, Is_Flag: true},
		},
		{Label: "ss", Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_STRINGS}},
		{Label: "ii", Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_INTEGERS}},
	}
}

// Each external scalar arm remains direct after declaration lookup and parse publication.
func Test_External_Accessors_Zero_Allocation(t *testing.T) {
	integer_environment := cli.New_Environment_Variable(
		cli.New_Environment_Variable_Input{
			Key: "INTEGER", Type: cli.OPTION_TYPE_INTEGER, Integer: 1,
		},
	)
	boolean_environment := cli.New_Environment_Variable(
		cli.New_Environment_Variable_Input{
			Key: "BOOLEAN", Type: cli.OPTION_TYPE_BOOLEAN, Boolean: true,
		},
	)
	integer_secret := cli.Resolved_Secret{
		Secret: cli.Secret{Key: "INTEGER", Type: cli.External_Type_State{
			Value: cli.Scalar_Option_Type(cli.OPTION_TYPE_INTEGER),
		}}, State: cli.Accepted_Secret_Value{Integer: 1},
	}
	boolean_secret := cli.Resolved_Secret{
		Secret: cli.Secret{Key: "BOOLEAN", Type: cli.External_Type_State{
			Value: cli.Scalar_Option_Type(cli.OPTION_TYPE_BOOLEAN),
		}}, State: cli.Accepted_Secret_Value{Boolean: true},
	}
	var integer cli.Integer
	var boolean cli.Boolean
	integer_environment_values := cli.Resolved_Environment{cli_resolved_environment(
		integer_environment, cli.Environment_State{
			Integer: integer_environment.State.Integer,
		},
	)}
	boolean_environment_values := cli.Resolved_Environment{cli_resolved_environment(
		boolean_environment, cli.Environment_State{
			Boolean: boolean_environment.State.Boolean,
		},
	)}
	integer_secret_values := cli.Resolved_Secrets{integer_secret}
	boolean_secret_values := cli.Resolved_Secrets{boolean_secret}
	operations := [...]func(){
		func() { integer = cli.Environment_Integer(integer_environment_values, "INTEGER") },
		func() { boolean = cli.Environment_Boolean(boolean_environment_values, "BOOLEAN") },
		func() { integer = cli.Secret_Integer(integer_secret_values, "INTEGER") },
		func() { boolean = cli.Secret_Boolean(boolean_secret_values, "BOOLEAN") },
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
		Completion: cli.Parser_Completion{Value: true},
	}}
	var result cli.Parse_Result
	var complete cli.Parser_Complete
	testify.Zero_Allocation(t, func() {
		result, complete = cli.Parser_Done(&parser)
	})
	if !complete {
		t.Fatal("parser completion result was discarded")
	}
	if cli.Parse_Error_Present(result.Error) {
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
		option := cli.Resolved_Option{
			Label:       cli.Option_Label(cli_boundary_text(label_size, 'l')),
			Description: cli.Description(cli_boundary_text(count, 'd')),
			Type:        cli.Option_Type_State{Value: cli.OPTION_TYPE_STRING},
			State: cli.Resolved_Option_State{String: cli.Value_Text(
				cli_boundary_text(count, 'v'),
			)},
		}
		options := cli.Resolved_Options{option}
		cli.Option_String(options, option.Label)
		option.Type.Value = cli.OPTION_TYPE_BOOLEAN
		option.State.Boolean = cli.Boolean(count != 0)
		options[0] = option
		cli.Option_Boolean(options, option.Label)
		option.Type.Value = cli.OPTION_TYPE_STRINGS
		option.State.Strings = make([]string, count)
		options[0] = option
		cli.Option_Strings(options, option.Label)
		option.Type.Value = cli.OPTION_TYPE_INTEGERS
		option.State.Integers = make([]int, count)
		options[0] = option
		cli.Option_Integers(options, option.Label)
		option.Type.Value = cli.OPTION_TYPE_INTEGER
		options[0] = option
		cli.Option_Integer(options, option.Label)
	}
	for _, value := range [...]int{bits.INTEGER_MINIMUM, bits.INTEGER_MAXIMUM} {
		option := cli.Resolved_Option{
			Label: "integer",
			Type:  cli.Option_Type_State{Value: cli.OPTION_TYPE_INTEGER},
			State: cli.Resolved_Option_State{Integer: cli.Integer(value)},
		}
		cli.Option_Integer(cli.Resolved_Options{option}, option.Label)
	}
	cli_option_accessor_type_bounds(t)
}

// Each typed option accessor searches the complete bounded resolved collection.
func Test_Option_Accessor_Collection_Bounds(t *testing.T) {
	for _, count := range [...]int{0, 1, 2, slices.COUNT_MAXIMUM} {
		text := cli_boundary_text(min(count, strings.TEXT_SIZE_MAXIMUM), 'd')
		string_enum := make([]string, count)
		integer_enum := make([]int, count)
		for index := range string_enum {
			string_enum[index] = "x"
			integer_enum[index] = 1
		}
		string_option := cli.Resolved_Option{
			Label: "value", Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_STRING},
			Enumeration: cli.Option_Enumeration{String: string_enum},
			State: cli.Resolved_Option_State{
				String: "x", Deprecated: cli.Deprecation(text),
			},
		}
		integer_option := cli.Resolved_Option{
			Label: "value", Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_INTEGER},
			Enumeration: cli.Option_Enumeration{Integers: integer_enum},
			State:       cli.Resolved_Option_State{Integer: 1},
		}
		boolean_option := cli.Resolved_Option{
			Label: "value", Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_BOOLEAN},
		}
		strings_option := cli.Resolved_Option{
			Label: "value", Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_STRINGS},
			State: cli.Resolved_Option_State{Strings: make([]string, count)},
		}
		integers_option := cli.Resolved_Option{
			Label: "value",
			Type:  cli.Option_Type_State{Value: cli.OPTION_TYPE_INTEGERS},
			State: cli.Resolved_Option_State{Integers: make([]int, count)},
		}
		cli_call_option_accessors(
			t, count, string_option, integer_option, boolean_option,
			strings_option, integers_option,
		)
	}
}

func cli_call_option_accessors(
	t *testing.T, count int, string_option cli.Resolved_Option,
	integer_option cli.Resolved_Option, boolean_option cli.Resolved_Option,
	strings_option cli.Resolved_Option, integers_option cli.Resolved_Option,
) {
	t.Helper()
	operations := [...]func(){
		func() { cli.Option_String(cli_repeated_options(count, string_option), "value") },
		func() { cli.Option_Integer(cli_repeated_options(count, integer_option), "value") },
		func() { cli.Option_Boolean(cli_repeated_options(count, boolean_option), "value") },
		func() { cli.Option_Strings(cli_repeated_options(count, strings_option), "value") },
		func() {
			cli.Option_Integers(cli_repeated_options(count, integers_option), "value")
		},
	}
	for _, operation := range operations {
		if count == 0 {
			assert_panics(t, "empty typed option lookup", operation)
			continue
		}
		operation()
	}
}

func cli_repeated_options(
	count int, option cli.Resolved_Option,
) (options cli.Resolved_Options) {
	options = make(cli.Resolved_Options, count)
	for index := range options {
		options[index] = option
	}
	return options
}

func cli_option_accessor_type_bounds(t *testing.T) {
	t.Helper()
	options := [...]cli.Resolved_Option{
		{Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_STRING}},
		{Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_INTEGER}},
		{Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_BOOLEAN}},
		{Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_STRINGS}},
		{Type: cli.Option_Type_State{Value: cli.OPTION_TYPE_INTEGERS}},
	}
	accessors := [...]func(cli.Resolved_Options, cli.Option_Label){
		func(options cli.Resolved_Options, label cli.Option_Label) {
			cli.Option_String(options, label)
		},
		func(options cli.Resolved_Options, label cli.Option_Label) {
			cli.Option_Integer(options, label)
		},
		func(options cli.Resolved_Options, label cli.Option_Label) {
			cli.Option_Boolean(options, label)
		},
		func(options cli.Resolved_Options, label cli.Option_Label) {
			cli.Option_Strings(options, label)
		},
		func(options cli.Resolved_Options, label cli.Option_Label) {
			cli.Option_Integers(options, label)
		},
	}
	for accessor_index, accessor := range accessors {
		for option_index, option := range options {
			values := cli.Resolved_Options{option}
			if accessor_index == option_index {
				accessor(values, option.Label)
				continue
			}
			assert_panics(t, "typed option accessor mismatch", func() {
				accessor(values, option.Label)
			})
		}
	}
}

// Benchmark_Program_Parse keeps byte and allocation evidence beside hard allocation checks.
func Benchmark_Program_Parse(b *testing.B) {
	program := cli.New(cli.New_Input{Mode: cli.PROGRAM_MODE_SINGLE, Label: "program"})
	var parser cli.Parser
	var global_flags [cli.HELP_FLAG_COUNT]cli.Resolved_Option
	var filled [cli.HELP_FLAG_COUNT]bool
	var positionals [ALLOCATION_PROGRAM_TOKEN_COUNT]cli.Indexed_Token
	var slice_named [ALLOCATION_PROGRAM_TOKEN_COUNT]cli.Indexed_Token
	var failure_storage [cli.FAILURE_SIZE_MAXIMUM]byte
	input := cli.Program_Parse_Input{
		Arguments:       []string{"program"},
		Global_Flags:    global_flags[:],
		Filled:          filled[:],
		Positionals:     positionals[:],
		Slice_Named:     slice_named[:],
		Failure_Storage: failure_storage[:],
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		cli.Program_Parse(program, &parser, input)
	}
}
