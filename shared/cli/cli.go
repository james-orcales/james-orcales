// Package cli provides a minimal command-line interface parser.
//
// It supports commands with positional arguments and optional flags.
// Arguments must appear before flags in the command line.
// Supported types: string, int, bool.
//
// Example:
//
//	program := cli.New(cli.New_Input{
//		Label:       "myapp",
//		Description: "does something useful",
//		Commands: []cli.Command{{
//			Label:     "add",
//			Arguments: []cli.Option{{Label: "name", Description: "item name"}},
//			Flags:     []cli.Option{{Label: "priority", Value: "low"}},
//		}},
//	})
//	command, err := cli.Program_Parse(&program, os.Args)
//	if err != nil {
//		cli.Print_Help(os.Stderr, program)
//		os.Exit(1)
//	}
//
// Parse with Program_Parse and render help with Print_Help.
//
// For a binary that does one thing, build the program with New_Single instead of
// New: it has no command selector, so the first token after the program name is the
// first positional argument (as in `sloc ./src`) rather than a command name.
//
// A command's last argument may be variadic (New_Variadic), collecting zero or more
// trailing positionals into a slice, as in `sloc ./a ./b ./c`.
package cli

import (
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"

	invariant "github.com/james-orcales/james-orcales/shared/invariant/default"
)

// Program represents a command-line application with one or more commands.
type Program struct {
	// Label is the program name shown in help output.
	Label string
	// Description is the one-line program summary shown in help output.
	Description string
	// Commands are the program's commands.
	Commands []Command
	// Global_Flags are flags accepted by every command.
	Global_Flags []Option
	// Single marks a program with no command selector: the first token after the
	// program name is the first positional argument, not a command name. New_Single
	// sets it; New leaves it false.
	Single bool
}

// Command represents a single command within a program.
// Commands have a label, optional arguments (required, ordered),
// and optional flags (optional, unordered).
type Command struct {
	// Label is the command name typed on the command line.
	Label string
	// Description is the one-line command summary shown in help output.
	Description string
	// Arguments are ordered and must ALL appear before flags. Each is required,
	// except a variadic last argument, which collects zero or more positionals.
	Arguments []Option
	// Flags are optional and unordered.
	Flags []Option
}

// Option represents either an argument or a flag for a command.
// The Value field determines the type: string or int for an argument, string, int,
// or bool for a flag, or a []string/[]int slice for a variadic argument.
// After parsing, Value contains the user-provided value.
type Option struct {
	// Label is the argument or flag name.
	Label string
	// Description is the one-line summary shown in help output.
	Description string
	// Value holds the default before parsing and the user value after.
	Value any
	// Is_Flag distinguishes a flag from a positional argument.
	Is_Flag bool
}

// New_Argument_Input is the input for New_Argument.
type New_Argument_Input struct {
	// Label is the argument name.
	Label string
	// Description is the one-line summary shown in help output.
	Description string
}

// New_Argument creates a required positional argument with the specified type.
// Arguments always have zero values; custom defaults are not allowed.
// Type parameter T must be string, int, or bool.
func New_Argument[T string | int | bool](input New_Argument_Input) (option Option) {
	var zero_value T
	return Option{
		Label:       input.Label,
		Description: input.Description,
		Value:       zero_value,
		Is_Flag:     false,
	}
}

// New_Variadic_Input is the input for New_Variadic.
type New_Variadic_Input struct {
	// Label is the argument name.
	Label string
	// Description is the one-line summary shown in help output.
	Description string
}

// New_Variadic creates a slice-valued argument: it collects the trailing positionals
// and appends each repeated -label=value. It must be the last argument; scalar
// arguments may precede it. A slice argument is always optional — an empty list is
// valid, not an error — so a command that needs at least one element checks the
// slice's length itself. Type parameter T must be string or int; the Value is a []T,
// and that slice type is what marks the option variadic.
func New_Variadic[T string | int](input New_Variadic_Input) (option Option) {
	return Option{
		Label:       input.Label,
		Description: input.Description,
		Value:       []T{},
		Is_Flag:     false,
	}
}

// New_Flag_Input is the input for New_Flag.
type New_Flag_Input[T string | int | bool] struct {
	// Label is the flag name.
	Label string
	// Value is the flag's default value.
	Value T
	// Description is the one-line summary shown in help output.
	Description string
}

// New_Flag creates an optional flag with a default value.
// Type parameter T must be string, int, or bool.
func New_Flag[T string | int | bool](input New_Flag_Input[T]) (option Option) {
	return Option{
		Label:       input.Label,
		Description: input.Description,
		Value:       input.Value,
		Is_Flag:     true,
	}
}

// New_Input is the input for New.
type New_Input struct {
	// Label is the program name.
	Label string
	// Description is the one-line program summary.
	Description string
	// Global_Flags are flags accepted by every command.
	Global_Flags []Option
	// Commands are the program's commands.
	Commands []Command
}

// New creates a new Program from the given label, description, global flags, and
// commands. It validates that all commands have labels and that arguments and
// flags are properly configured, panicking when validation fails.
func New(input New_Input) (program Program) {
	panic_when(len(input.Commands) == 0, "Program has zero commands specified.")

	for index, flag := range input.Global_Flags {
		panic_when(!flag.Is_Flag, "Global flags must be created with New_Flag.")
		validate_flag_label(fmt.Sprintf("Global flag #%d", index), flag)
	}

	program = Program{
		Label:        input.Label,
		Description:  input.Description,
		Commands:     input.Commands,
		Global_Flags: input.Global_Flags,
	}

	for index, command := range program.Commands {
		panic_when(command.Label == "", "Program.Commands[%d].Label is unset.", index)
		command_validate_options(command, input.Global_Flags)
	}
	return program
}

// New_Single_Input is the input for New_Single.
type New_Single_Input struct {
	// Label is the program name.
	Label string
	// Description is the one-line program summary.
	Description string
	// Arguments are the program's required, ordered positional arguments. They must
	// ALL appear before flags.
	Arguments []Option
	// Flags are the program's optional, unordered flags.
	Flags []Option
}

// New_Single creates a program with no command selector: the first token after the
// program name is the first positional argument, not a command name. Use it for a
// binary that does one thing, like `sloc ./src`. It validates arguments and flags
// the same way New does, panicking when validation fails.
//
// A single-command program has no command namespace, so its first positional can
// take any value. This is the only configuration where a defaulted command and its
// positional arguments do not collide on the first token: with sibling commands,
// that token must select a command, and a positional sharing a command's name would
// be unreachable. A program that needs sibling commands must use New.
func New_Single(input New_Single_Input) (program Program) {
	command := Command{
		Label:       input.Label,
		Description: input.Description,
		Arguments:   input.Arguments,
		Flags:       input.Flags,
	}
	command_validate_options(command, nil)
	return Program{
		Label:       input.Label,
		Description: input.Description,
		Commands:    []Command{command},
		Single:      true,
	}
}

// Panics when any of a command's options is malformed: a label that is empty, not
// flag-safe, or shared with another option (arguments and flags occupy one
// -label=value namespace), an unsupported value type, or a slice argument that is not
// last. The global flags are pre-seeded so an argument or flag colliding with one is
// caught here too.
func command_validate_options(command Command, global_flags []Option) {
	seen := map[string]bool{}
	for _, global_flag := range global_flags {
		seen[global_flag.Label] = true
	}
	for argument_index, argument := range command.Arguments {
		option_validate_label(
			fmt.Sprintf("Argument #%d for command %q", argument_index, command.Label),
			argument)
		// A slice argument absorbs every remaining positional, so anything declared
		// after it is unreachable; requiring it last also forbids a second slice.
		is_last := argument_index == len(command.Arguments)-1
		panic_when(option_is_slice(argument) && !is_last,
			"Slice argument %q must be the last argument.", argument.Label)
		switch argument.Value.(type) {
		default:
			panic_when(true, "Argument %q has unsupported type: %T",
				argument.Label, argument.Value)
		case string, int, []string, []int:
		}
		panic_when(seen[argument.Label],
			"Command %q has argument %q that collides with another option.",
			command.Label, argument.Label)
		seen[argument.Label] = true
	}
	for _, flag := range command.Flags {
		validate_flag_label(fmt.Sprintf("Flag for command %q", command.Label), flag)
		panic_when(seen[flag.Label],
			"Command %q has flag %q that collides with another option.",
			command.Label, flag.Label)
		seen[flag.Label] = true
	}
}

// Reports whether an option holds a slice value. The slice type is what makes an
// option variadic: an argument that collects positionals, and any option that appends
// on each repeated -label=value.
func option_is_slice(option Option) (is_slice bool) {
	switch option.Value.(type) {
	case []string, []int:
		return true
	}
	return false
}

// Panics when an option's label is empty or carries a character that cannot appear in
// a -label token. Arguments and flags share this rule because both are settable by
// name.
func option_validate_label(context string, option Option) {
	panic_when(option.Label == "", "%s has no label.", context)
	panic_when(strings.Contains(option.Label, "_"),
		"Option labels cannot contain underscores. Instead of %q, use %q",
		option.Label, strings.ReplaceAll(option.Label, "_", "-"))
	panic_when(strings.Contains(option.Label, " "),
		"Option labels cannot contain spaces: %q", option.Label)
}

// Panics when a flag's label is invalid or its value is not a supported scalar type.
func validate_flag_label(context string, flag Option) {
	option_validate_label(context, flag)
	switch flag.Value.(type) {
	default:
		panic_when(true, "Flag %q has unsupported type: %T", flag.Label, flag.Value)
	case string, bool, int:
	}
}

// Program_Parse parses command-line arguments and returns the active command with
// populated values. The operating_system_args slice should be os.Args from main. If
// no command is specified, the first declared command is used. Every option is
// settable by -label=value and arguments may also be given positionally; the two
// kinds of token may interleave freely. It returns an error for an unknown command,
// an unknown option, a scalar set more than once, a positional with no argument to
// fill, a missing required argument, or an invalid or absent value.
func Program_Parse(
	program *Program, operating_system_args []string,
) (active_command Command, err error) {
	panic_when(len(operating_system_args) == 0, "Program_Parse needs at least one os_arg.")

	// Deep copy global flags to avoid modifying the program's definitions.
	original_global_flags := program.Global_Flags
	program.Global_Flags = make([]Option, len(original_global_flags))
	copy(program.Global_Flags, original_global_flags)

	defer func() {
		argument_count := len(active_command.Arguments)
		flag_count := len(active_command.Flags)
		invariant.Dot_Product(
			invariant.Sometimes(argument_count > 0),
			invariant.Sometimes(flag_count > 0),
		)
	}()

	active_command, arguments_start, err := program_resolve_command(
		program, operating_system_args)
	if err != nil {
		return active_command, err
	}
	// Nothing follows the consumed prefix (the program name, plus the command name in
	// multi-command mode), so the default command stands as-is.
	if len(operating_system_args) < arguments_start {
		return active_command, nil
	}

	tokens := operating_system_args[arguments_start:]
	filled, positionals, slice_named, err := program_assign_named(
		program, active_command, tokens)
	if err != nil {
		return active_command, err
	}
	err = command_assign_positionals(&command_assign_positionals_input{
		Command:     active_command,
		Positionals: positionals,
		Slice_Named: slice_named,
		Filled:      filled,
	})
	if err != nil {
		return active_command, err
	}
	err = command_validate_required(active_command, filled)
	if err != nil {
		return active_command, err
	}
	return active_command, nil
}

// One command-line token paired with its position, so a slice argument can reassemble
// its positional and -label=value contributions in the order they were written.
type indexed_token struct {
	Index int
	Value string
}

// Reports whether a token sets an option by name. A lone "-" is a positional value.
func is_named_token(token string) (named bool) {
	if !strings.HasPrefix(token, "-") {
		return false
	}
	return token != "-"
}

// Splits a -label=value token into its parts, rejecting the double-dash form.
func parse_named_token(token string) (
	label string, value string, value_was_set bool, err error,
) {
	if strings.HasPrefix(token, "--") {
		if len(token) > len("--") {
			return "", "", false, fmt.Errorf(
				"Flags use a single dash, not double; use '-%s'", token[2:])
		}
	}
	// A lone "-" was already excluded by is_named_token.
	invariant.Dot_Product(invariant.Always(token != "-"))
	label, value, value_was_set = strings.Cut(token[1:], "=")
	return label, value, value_was_set, nil
}

// Applies every -label=value token to its option and returns the bare positionals
// plus, for the slice argument, the values given by name — both tagged with their
// position so the slice preserves command-line order. filled records the scalar
// options set by name: a scalar named twice is an error, and a named scalar argument
// is skipped by the positional pass.
func program_assign_named(program *Program, command Command, tokens []string) (
	filled map[string]bool, positionals []indexed_token, slice_named []indexed_token, err error,
) {
	filled = map[string]bool{}
	for index, token := range tokens {
		if !is_named_token(token) {
			positionals = append(positionals, indexed_token{Index: index, Value: token})
			continue
		}
		label, value, value_was_set, format_err := parse_named_token(token)
		if format_err != nil {
			return filled, positionals, slice_named, format_err
		}
		option, is_slice_argument, find_err := program_find_option(program, command, label)
		if find_err != nil {
			return filled, positionals, slice_named, find_err
		}
		if is_slice_argument {
			slice_named = append(slice_named, indexed_token{Index: index, Value: value})
			continue
		}
		if filled[label] {
			return filled, positionals, slice_named,
				fmt.Errorf("%q set more than once", label)
		}
		set_err := option_set_value(option_set_value_input{
			Option: option, Value: value, Value_Was_Set: value_was_set, Name: label,
		})
		if set_err != nil {
			return filled, positionals, slice_named, set_err
		}
		filled[label] = true
	}
	return filled, positionals, slice_named, nil
}

// Finds the option named by a -label token across the command's arguments, then its
// flags, then the program's global flags, returning a pointer into the parse-time copy
// so assignment lands in the right slot. is_slice_argument is true when the match is a
// slice-valued argument, which appends rather than sets. Errors on an unknown label.
func program_find_option(program *Program, command Command, label string) (
	option *Option, is_slice_argument bool, err error,
) {
	for index := range command.Arguments {
		argument := &command.Arguments[index]
		if argument.Label == label {
			return argument, option_is_slice(*argument), nil
		}
	}
	for index := range command.Flags {
		if command.Flags[index].Label == label {
			return &command.Flags[index], false, nil
		}
	}
	for index := range program.Global_Flags {
		if program.Global_Flags[index].Label == label {
			return &program.Global_Flags[index], false, nil
		}
	}
	return nil, false, fmt.Errorf("%q is an unknown option", label)
}

// Finds the active command named by the args (defaulting to the first command) and
// deep-copies its Arguments and Flags so parsing never mutates the program.
// arguments_start is the index at which positionals and flags begin: after the
// program name and the command name in multi-command mode, after only the program
// name in single-command mode, where the first token is already a positional.
func program_resolve_command(
	program *Program, operating_system_args []string,
) (active_command Command, arguments_start int, err error) {
	command_index := 0
	arguments_start = 2
	if program.Single {
		arguments_start = 1
	} else if len(operating_system_args) > 1 {
		found := false
		for index, command := range program.Commands {
			if command.Label == operating_system_args[1] {
				command_index = index
				found = true
				break
			}
		}
		if !found {
			return program.Commands[0], arguments_start, fmt.Errorf(
				"%q is an unknown command", operating_system_args[1])
		}
	}

	source := program.Commands[command_index]
	active_command = source
	active_command.Arguments = make([]Option, len(source.Arguments))
	copy(active_command.Arguments, source.Arguments)
	active_command.Flags = make([]Option, len(source.Flags))
	copy(active_command.Flags, source.Flags)
	return active_command, arguments_start, nil
}

// Input for command_assign_positionals.
type command_assign_positionals_input struct {
	// Command is the active command whose arguments are filled in place.
	Command Command
	// Positionals are the bare tokens, each tagged with its command-line position.
	Positionals []indexed_token
	// Slice_Named are the trailing slice argument's -label=value contributions, tagged
	// with position so they merge with the positionals in order.
	Slice_Named []indexed_token
	// Filled records the scalar arguments already set by name; it gains those set here.
	Filled map[string]bool
}

// Fills the command's scalar arguments from the positional tokens in declaration
// order, skipping any already set by name, then routes the rest into the trailing
// slice argument together with its named contributions, in command-line order. Errors
// when a positional has no argument to fill. Filled gains every scalar argument set
// here so the required check can see it.
func command_assign_positionals(input *command_assign_positionals_input) (err error) {
	command := input.Command
	fill_targets := []int{}
	for index := range command.Arguments {
		argument := command.Arguments[index]
		if option_is_slice(argument) {
			continue
		}
		if input.Filled[argument.Label] {
			continue
		}
		fill_targets = append(fill_targets, index)
	}

	slice_index := command_slice_argument_index(command)
	slice_contributions := append([]indexed_token{}, input.Slice_Named...)
	cursor := 0
	for _, positional := range input.Positionals {
		if cursor < len(fill_targets) {
			argument := &command.Arguments[fill_targets[cursor]]
			set_err := option_set_positional(argument, positional.Value)
			if set_err != nil {
				return set_err
			}
			input.Filled[argument.Label] = true
			cursor++
			continue
		}
		if slice_index < 0 {
			return fmt.Errorf("unexpected argument %q", positional.Value)
		}
		slice_contributions = append(slice_contributions, positional)
	}
	if slice_index < 0 {
		return nil
	}
	slices.SortFunc(slice_contributions, func(left, right indexed_token) (order int) {
		return left.Index - right.Index
	})
	return option_set_slice(&command.Arguments[slice_index], slice_contributions)
}

// Returns the index of the command's trailing slice argument, or -1 when the last
// argument is not a slice.
func command_slice_argument_index(command Command) (index int) {
	count := len(command.Arguments)
	if count == 0 {
		return -1
	}
	if option_is_slice(command.Arguments[count-1]) {
		return count - 1
	}
	return -1
}

// Returns an error naming the first scalar argument left unset by both name and
// position. The slice argument is optional, so it is never required.
func command_validate_required(command Command, filled map[string]bool) (err error) {
	for index := range command.Arguments {
		argument := command.Arguments[index]
		if option_is_slice(argument) {
			continue
		}
		if filled[argument.Label] {
			continue
		}
		return fmt.Errorf("%q is missing required argument %q",
			command.Label, argument.Label)
	}
	return nil
}

// Converts a positional token to the scalar argument's type and assigns it. Unlike a
// named value, a positional is taken verbatim: no quote trimming, and empty is allowed.
func option_set_positional(argument *Option, value string) (err error) {
	switch argument.Value.(type) {
	default:
		panic_when(true, "unreachable")
	case string:
		argument.Value = value
	case int:
		number, parse_err := strconv.Atoi(value)
		if parse_err != nil {
			return fmt.Errorf("%s is an invalid number", value)
		}
		argument.Value = number
	}
	return nil
}

// Builds the slice option's value from its contributions, already sorted into
// command-line order, converting each element to the slice's element type.
func option_set_slice(option *Option, contributions []indexed_token) (err error) {
	switch option.Value.(type) {
	default:
		panic_when(true, "unreachable")
	case []string:
		values := make([]string, 0, len(contributions))
		for _, contribution := range contributions {
			values = append(values, contribution.Value)
		}
		option.Value = values
	case []int:
		values := make([]int, 0, len(contributions))
		for _, contribution := range contributions {
			number, parse_err := strconv.Atoi(contribution.Value)
			if parse_err != nil {
				return fmt.Errorf("%s is an invalid number", contribution.Value)
			}
			values = append(values, number)
		}
		option.Value = values
	}
	return nil
}

// Input for option_set_value.
type option_set_value_input struct {
	// Option is the flag whose Value is assigned in place.
	Option *Option
	// Value is the raw value text following '='.
	Value string
	// Value_Was_Set reports whether '=' appeared in the flag.
	Value_Was_Set bool
	// Name is the flag name, for error messages.
	Name string
}

// Validates and assigns a parsed flag value by the option's declared type.
// Non-boolean flags require a non-empty value.
func option_set_value(input option_set_value_input) (err error) {
	_, is_boolean := input.Option.Value.(bool)
	if !is_boolean {
		absent := !input.Value_Was_Set
		if input.Value == "" {
			absent = true
		}
		if absent {
			return fmt.Errorf(
				"%q expects a value. You must set flag values "+
					"with this syntax: -foo-bar=baz.",
				input.Name,
			)
		}
	}
	switch input.Option.Value.(type) {
	case bool:
		input.Option.Value = true
	case string:
		input.Option.Value = trim_quotes(input.Value)
	case int:
		number, parse_err := strconv.Atoi(input.Value)
		if parse_err != nil {
			return fmt.Errorf("%s is an invalid number", input.Value)
		}
		input.Option.Value = number
	}
	return nil
}

// Removes a matching pair of surrounding double or single quotes from a string
// value, leaving other strings untouched.
func trim_quotes(text string) (output string) {
	if len(text) < 2 {
		return text
	}
	first := text[0]
	last := text[len(text)-1]
	if first == '"' {
		if last == '"' {
			return text[1 : len(text)-1]
		}
	}
	if first == '\'' {
		if last == '\'' {
			return text[1 : len(text)-1]
		}
	}
	return text
}

// Print_Help writes a formatted help message to output: the program header,
// usage syntax, global flags, and each command with its arguments and flags.
func Print_Help(output io.Writer, program Program) {
	fmt.Fprintf(output, "%s %s\n\n", program.Label, program.Description)
	if program.Single {
		print_help_single(output, program)
		return
	}
	fmt.Fprintf(output,
		"Usage:\n    %s <command> <arguments> [-flags[=value]]\n\n", program.Label)

	if len(program.Global_Flags) > 0 {
		fmt.Fprintln(output, "Global Flags:")
		writer := tabwriter.NewWriter(output, 0, 8, 0, ' ', 0)
		for _, flag := range program.Global_Flags {
			print_help_flag(writer, flag, "    ", true)
			fmt.Fprintf(writer, "\n")
		}
		writer.Flush()
		fmt.Fprintln(output, "")
	}

	fmt.Fprintln(output, "Available Commands:")
	for _, command := range program.Commands {
		// A tabwriter per command avoids tab alignment bleeding across commands.
		writer := tabwriter.NewWriter(output, 0, 8, 0, ' ', 0)
		signature := "\033[34m" + command.Label + "\033[0m" + " "
		for _, argument := range command.Arguments {
			signature += option_format_signature(argument) + " "
		}
		fmt.Fprintf(writer, "    %s\t%s\n", signature, command.Description)
		if len(command.Flags) > 0 {
			fmt.Fprintln(writer, "")
			for _, flag := range command.Flags {
				print_help_flag(writer, flag, "        ", false)
			}
		}
		writer.Flush()
		fmt.Fprintln(output, "")
	}
}

// Renders help for a single-command program: a usage line carrying the program's
// own positional arguments — there is no command selector to choose — followed by
// its flags.
func print_help_single(output io.Writer, program Program) {
	command := program.Commands[0]
	signature := ""
	for _, argument := range command.Arguments {
		signature += option_format_signature(argument) + " "
	}
	fmt.Fprintf(output, "Usage:\n    %s %s[-flags[=value]]\n", program.Label, signature)
	if len(command.Flags) > 0 {
		fmt.Fprintln(output, "")
		fmt.Fprintln(output, "Flags:")
		writer := tabwriter.NewWriter(output, 0, 8, 0, ' ', 0)
		for _, flag := range command.Flags {
			print_help_flag(writer, flag, "    ", true)
		}
		writer.Flush()
	}
}

// Formats a positional argument for a usage line: <label:type>, with a trailing
// ellipsis for a variadic. A variadic shows its element type, not its slice type, so
// a []string reads as <paths:string...> rather than the noisier <paths:[]string>.
func option_format_signature(argument Option) (signature string) {
	if option_is_slice(argument) {
		element := "string"
		if _, is_integer := argument.Value.([]int); is_integer {
			element = "int"
		}
		return fmt.Sprintf("<%s:%s...>", argument.Label, element)
	}
	return fmt.Sprintf("<%s:%T>", argument.Label, argument.Value)
}

// Renders one flag row: the type-annotated label, its default, and its
// description. The label is blue when color is set.
func print_help_flag(writer io.Writer, flag Option, indent string, color bool) {
	label := flag.Label
	if color {
		label = "\033[34m" + flag.Label + "\033[0m"
	}

	value_type := ""
	default_value := fmt.Sprintf("%v", flag.Value)
	is_boolean := false
	if _, is := flag.Value.(bool); is {
		is_boolean = true
	} else {
		value_type = fmt.Sprintf("=%T", flag.Value)
	}
	// An empty string default renders as the literal "".
	if text, is_string := flag.Value.(string); is_string {
		if text == "" {
			default_value = `""`
		}
	}

	if is_boolean {
		fmt.Fprintf(writer, "%s-%s\t  %s\n", indent, label, flag.Description)
		return
	}
	fmt.Fprintf(writer, "%s-%s%s\t  (default: %s)\t  %s\n",
		indent, label, value_type, default_value, flag.Description)
}

// Get_Option retrieves an option by label from a slice of options, panicking
// when the label is not found. It extracts argument or flag values from a
// parsed command, as in Get_Option(command.Arguments, "task").Value.(string).
func Get_Option(flags []Option, label string) (option Option) {
	for _, flag := range flags {
		if flag.Label == label {
			return flag
		}
	}
	panic_when(true, "%q is an unknown option", label)
	return Option{}
}

// Panics with the formatted message when condition holds.
func panic_when(condition bool, message string, data ...any) {
	if condition {
		panic(fmt.Sprintf(message, data...))
	}
}
