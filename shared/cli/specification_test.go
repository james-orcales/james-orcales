package cli_test

import (
	"slices"
	"strings"
	"testing"

	"local/james-orcales/shared/cli"
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

	// A near-miss command yields a suggestion; a wild miss does not.
	_, err = cli.Program_Parse(&fixture.Program, []string{"todoctl", "lst"})
	if err == nil {
		t.Fatal("expected an error for unknown command lst")
	}
	if !strings.Contains(err.Error(), `did you mean "list"`) {
		t.Errorf("expected a suggestion of list, got %v", err)
	}
	_, err = cli.Program_Parse(&fixture.Program, []string{"todoctl", "zzzzzzzz"})
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

// Test_Parse_Multicall verifies a multicall program selects its command from the
// binary name in argv[0], not a token in slot 1, so a symlinked verb dispatches on
// its own name and everything after argv[0] is that command's arguments.
func Test_Parse_Multicall(t *testing.T) {
	program := new_multicall_fixture()

	// The command is the basename of argv[0]; the rest of argv is its arguments.
	command, err := cli.Program_Parse(&program, []string{"/usr/local/bin/add", "milk"})
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
	_, err = cli.Program_Parse(&program, []string{"ad"})
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

// Test_Parse_Named verifies every option is settable by -label=value, that named and
// positional tokens interleave freely, and that a repeated or unknown option errors.
func Test_Parse_Named(t *testing.T) {
	fixture := new_cli_fixture()

	// An argument can be set by name instead of by position.
	command, err := cli.Program_Parse(&fixture.Program,
		[]string{"todoctl", "add", "-task=hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Get_Option(command.Arguments, "task").Value.(string) != "hello" {
		t.Errorf("expected task hello, got %v",
			cli.Get_Option(command.Arguments, "task").Value)
	}

	// Named and positional tokens may appear in any order.
	command, err = cli.Program_Parse(&fixture.Program,
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
	command, err = cli.Program_Parse(&pair, []string{"pair", "-first=x", "y"})
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
	_, err = cli.Program_Parse(&fixture.Program,
		[]string{"todoctl", "add", "task", "-priority=high", "-priority=low"})
	if err == nil {
		t.Error("expected error for a scalar set more than once")
	}
	_, err = cli.Program_Parse(&fixture.Program, []string{"todoctl", "add", "task", "-zzz=1"})
	if err == nil {
		t.Error("expected error for an unknown option")
	}

	// A near-miss option name yields a suggestion.
	_, err = cli.Program_Parse(&fixture.Program,
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
	_, err := cli.Program_Parse(&multi, []string{"fileutil", "cp"})
	if err == nil {
		t.Error("expected error for the missing scalar argument")
	}

	// A slice int converts each element and reports a bad one.
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
		t.Error("expected error for a non-numeric slice element")
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
	// An enum flag's default must be one of its permitted values.
	assert_panics(t, "enum flag default outside its set", func() {
		cli.New_Single(cli.New_Single_Input{
			Label: "prog",
			Flags: []cli.Option{
				cli.New_Enum_Flag(cli.New_Enum_Flag_Input[string]{
					Label: "color", Enum: []string{"auto", "never"}, Value: "rainbow",
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
			Label:     "prog",
			Arguments: []cli.Option{{Label: "path", Value: []string{}, Enum: []string{"a", "b"}}},
		})
	})
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
	command, err := cli.Program_Parse(&program, []string{"prog", "-color=never"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Get_Option(command.Flags, "color").Value.(string) != "never" {
		t.Errorf("expected never, got %q", cli.Get_Option(command.Flags, "color").Value)
	}

	// An omitted enum keeps its default.
	command, err = cli.Program_Parse(&program, []string{"prog"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Get_Option(command.Flags, "color").Value.(string) != "auto" {
		t.Errorf("expected default auto, got %q",
			cli.Get_Option(command.Flags, "color").Value)
	}

	// A near miss suggests the closest member.
	_, err = cli.Program_Parse(&program, []string{"prog", "-color=nevr"})
	if err == nil {
		t.Fatal("expected an error for an out-of-set value")
	}
	if !strings.Contains(err.Error(), `did you mean "never"`) {
		t.Errorf("expected a suggestion of never, got %v", err)
	}

	// A wild miss lists the whole set instead of guessing.
	_, err = cli.Program_Parse(&program, []string{"prog", "-color=purple"})
	if err == nil {
		t.Fatal("expected an error for an out-of-set value")
	}
	if !strings.Contains(err.Error(), "allowed: auto, never, always") {
		t.Errorf("expected the allowed list, got %v", err)
	}

	// An int enum accepts a member and rejects a non-member with the allowed list.
	command, err = cli.Program_Parse(&program, []string{"prog", "-level=4"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Get_Option(command.Flags, "level").Value.(int) != 4 {
		t.Errorf("expected 4, got %v", cli.Get_Option(command.Flags, "level").Value)
	}
	_, err = cli.Program_Parse(&program, []string{"prog", "-level=3"})
	if err == nil {
		t.Fatal("expected an error for an out-of-set int value")
	}
	if !strings.Contains(err.Error(), "allowed: 1, 2, 4, 8") {
		t.Errorf("expected the allowed int list, got %v", err)
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
	command, err := cli.Program_Parse(&program, []string{"prog", "yaml"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Get_Option(command.Arguments, "format").Value.(string) != "yaml" {
		t.Errorf("expected yaml, got %q",
			cli.Get_Option(command.Arguments, "format").Value)
	}

	// Accepted by name.
	command, err = cli.Program_Parse(&program, []string{"prog", "-format=toml"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.Get_Option(command.Arguments, "format").Value.(string) != "toml" {
		t.Errorf("expected toml, got %q",
			cli.Get_Option(command.Arguments, "format").Value)
	}

	// Omitted → the existing missing-required-argument error.
	_, err = cli.Program_Parse(&program, []string{"prog"})
	if err == nil {
		t.Fatal("expected an error for a missing required enum argument")
	}
	if !strings.Contains(err.Error(), "missing required argument") {
		t.Errorf("expected the missing-required message, got %v", err)
	}

	// A non-member is rejected with the allowed list.
	_, err = cli.Program_Parse(&program, []string{"prog", "xml"})
	if err == nil {
		t.Fatal("expected an error for an out-of-set argument")
	}
	if !strings.Contains(err.Error(), "allowed: json, yaml, toml") {
		t.Errorf("expected the allowed list, got %v", err)
	}
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
