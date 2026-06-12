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
	// Is_Variadic marks a positional argument that collects zero or more trailing
	// positionals into a slice. Only the last argument may be variadic.
	Is_Variadic bool
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

// New_Variadic creates a positional argument that collects zero or more trailing
// positionals into a slice. It must be the last argument; fixed arguments may
// precede it. A variadic is always optional — an empty list is valid, not an error —
// so a command that needs at least one element checks the slice's length itself.
// Type parameter T must be string or int; the parsed Value is a []T.
func New_Variadic[T string | int](input New_Variadic_Input) (option Option) {
	return Option{
		Label:       input.Label,
		Description: input.Description,
		Value:       []T{},
		Is_Flag:     false,
		Is_Variadic: true,
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

// Panics when any of a command's arguments or flags is malformed: an argument with
// no label or an unsupported type, a flag with an invalid label or type, or a flag
// whose label collides with a global flag.
func command_validate_options(command Command, global_flags []Option) {
	for argument_index, argument := range command.Arguments {
		panic_when(argument.Label == "",
			"Argument #%d for command %q has no label.",
			argument_index, command.Label)
		// A variadic absorbs every remaining positional, so anything declared after
		// it is unreachable. Requiring it to be last also forbids a second variadic.
		is_last := argument_index == len(command.Arguments)-1
		panic_when(argument.Is_Variadic && !is_last,
			"Variadic argument %q must be the last argument.", argument.Label)
		if argument.Is_Variadic {
			switch argument.Value.(type) {
			default:
				panic_when(true,
					"Variadic argument %q has unsupported type: %T",
					argument.Label, argument.Value)
			case []string, []int:
			}
			continue
		}
		switch argument.Value.(type) {
		default:
			panic_when(true,
				"Argument %q has unsupported type: %T",
				argument.Label, argument.Value)
		case string, int:
		}
	}
	for _, flag := range command.Flags {
		validate_flag_label(fmt.Sprintf("Flag for command %q", command.Label), flag)
		for _, global_flag := range global_flags {
			panic_when(flag.Label == global_flag.Label,
				"Command %q has flag %q that conflicts with a global flag.",
				command.Label, flag.Label)
		}
	}
}

// Panics when the flag's label is empty, contains an underscore or space, or
// carries an unsupported value type.
func validate_flag_label(context string, flag Option) {
	panic_when(flag.Label == "", "%s has no label.", context)
	panic_when(
		strings.Contains(flag.Label, "_"),
		"Flags cannot contain underscores. Instead of %q, use %q",
		flag.Label,
		strings.ReplaceAll(flag.Label, "_", "-"),
	)
	panic_when(strings.Contains(flag.Label, " "), "Flags cannot contain spaces: %q", flag.Label)
	switch flag.Value.(type) {
	default:
		panic_when(true, "Flag %q has unsupported type: %T", flag.Label, flag.Value)
	case string, bool, int:
	}
}

// Program_Parse parses command-line arguments and returns the active command
// with populated values. The operating_system_args slice should be os.Args from main.
// If no command is specified, the first declared command is used. Global flags
// are parsed and stored in program.Global_Flags. It returns an error for an
// unknown command, the wrong argument count, a positional after a flag, an
// unknown flag, or an invalid or absent flag value.
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
	// Nothing follows the consumed prefix (the program name, plus the command name
	// in multi-command mode), so the default command stands as-is.
	if len(operating_system_args) < arguments_start {
		return active_command, nil
	}

	positional_arguments, flags, err := command_collect_args(
		active_command, operating_system_args, arguments_start)
	if err != nil {
		return active_command, err
	}
	invariant.Dot_Product(invariant.Sometimes(len(flags) == 0))
	invariant.Dot_Product(invariant.Sometimes(len(flags) < len(active_command.Flags)))
	invariant.Dot_Product(invariant.Sometimes(len(flags) == len(active_command.Flags)))

	err = command_validate_count(active_command, len(positional_arguments))
	if err != nil {
		return active_command, err
	}
	if len(flags) > len(active_command.Flags)+len(program.Global_Flags) {
		return active_command, fmt.Errorf(
			"%q supports %d command flags and %d global flags. Got %d",
			active_command.Label,
			len(active_command.Flags),
			len(program.Global_Flags),
			len(flags),
		)
	}

	err = parse_arguments(active_command.Arguments, positional_arguments)
	if err != nil {
		return active_command, err
	}
	err = program_parse_flags(program, active_command, flags)
	if err != nil {
		return active_command, err
	}
	return active_command, nil
}

// Reports the argument-count error for a command given its parsed positional count,
// or nil when the count is acceptable. A variadic last argument turns the fixed
// arguments before it into a minimum rather than an exact count.
func command_validate_count(command Command, positional_count int) (err error) {
	declared_count := len(command.Arguments)
	last_is_variadic := declared_count > 0 &&
		command.Arguments[declared_count-1].Is_Variadic
	if last_is_variadic {
		minimum := declared_count - 1
		if positional_count < minimum {
			return fmt.Errorf("%q expects at least %d arguments. Got %d",
				command.Label, minimum, positional_count)
		}
		return nil
	}
	if positional_count != declared_count {
		return fmt.Errorf("%q expects %d arguments. Got %d",
			command.Label, declared_count, positional_count)
	}
	return nil
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

// Splits the args after the consumed prefix into positional arguments (before the
// first flag) and flags, erroring on a positional that appears after a flag.
// arguments_start is where the split begins: past the program-and-command prefix in
// multi-command mode, past only the program name in single-command mode.
func command_collect_args(
	active_command Command, operating_system_args []string, arguments_start int,
) (
	positional_arguments []string, flags []string, err error,
) {
	positional_arguments = make([]string, 0, len(active_command.Arguments))
	flags = make([]string, 0, len(active_command.Flags))
	if len(operating_system_args) <= arguments_start {
		return positional_arguments, flags, nil
	}

	start_of_flags := -1
	for index, argument := range operating_system_args[arguments_start:] {
		if strings.HasPrefix(argument, "-") {
			if argument != "-" {
				start_of_flags = index + arguments_start
				break
			}
		}
		positional_arguments = append(positional_arguments, argument)
	}
	if start_of_flags < 0 {
		return positional_arguments, flags, nil
	}
	for _, flag := range operating_system_args[start_of_flags:] {
		if !strings.HasPrefix(flag, "-") {
			return positional_arguments, flags, fmt.Errorf(
				"Positional arguments cannot appear after flags. Got %q", flag)
		}
		flags = append(flags, flag)
	}
	return positional_arguments, flags, nil
}

// Converts each positional argument to its option's declared type, writing the
// result back into the options. A variadic last argument absorbs every remaining
// positional into its slice.
func parse_arguments(arguments []Option, positional_arguments []string) (err error) {
	for index, argument := range arguments {
		if argument.Is_Variadic {
			// The variadic is the last argument; everything from here on is its slice,
			// so this returns and no later argument is visited.
			return option_parse_variadic(
				&arguments[index], positional_arguments[index:])
		}
		positional_argument := positional_arguments[index]
		switch argument.Value.(type) {
		default:
			panic_when(true, "unreachable")
		case string:
			arguments[index].Value = positional_argument
		case int:
			number, parse_err := strconv.Atoi(positional_argument)
			if parse_err != nil {
				return fmt.Errorf("%s is an invalid number", positional_argument)
			}
			arguments[index].Value = number
		}
	}
	return nil
}

// Collects the remaining positionals into the variadic argument's slice, converting
// each element to the slice's declared element type. The collected slice is fresh so
// it does not alias the parser's scratch buffer.
func option_parse_variadic(argument *Option, positional_arguments []string) (err error) {
	switch argument.Value.(type) {
	default:
		panic_when(true, "unreachable")
	case []string:
		values := make([]string, len(positional_arguments))
		copy(values, positional_arguments)
		argument.Value = values
	case []int:
		values := make([]int, 0, len(positional_arguments))
		for _, positional_argument := range positional_arguments {
			number, parse_err := strconv.Atoi(positional_argument)
			if parse_err != nil {
				return fmt.Errorf("%s is an invalid number", positional_argument)
			}
			values = append(values, number)
		}
		argument.Value = values
	}
	return nil
}

// Resolves each raw flag against the program's global flags and the active
// command's flags, writing the converted values back.
func program_parse_flags(program *Program, active_command Command, flags []string) (err error) {
	for _, flag := range flags {
		if strings.HasPrefix(flag, "--") {
			if len(flag) > len("--") {
				return fmt.Errorf(
					"Flags are denoted by a single dash, not double. use '-%s'",
					flag[2:])
			}
		}
		// A lone "-" is routed to positional args by command_collect_args, so a flag
		// token is never "-".
		invariant.Dot_Product(invariant.Always(flag != "-"))
		flag = flag[1:]
		flag_name, value, value_was_set := strings.Cut(flag, "=")

		global_index := slices.IndexFunc(program.Global_Flags,
			func(option Option) (match bool) {
				return option.Label == flag_name
			})
		if global_index >= 0 {
			err = option_set_value(option_set_value_input{
				Option:        &program.Global_Flags[global_index],
				Value:         value,
				Value_Was_Set: value_was_set,
				Name:          flag_name,
			})
			if err != nil {
				return err
			}
			continue
		}

		command_index := slices.IndexFunc(active_command.Flags,
			func(option Option) (match bool) {
				return option.Label == flag_name
			})
		if command_index < 0 {
			return fmt.Errorf("%q is an unknown flag", flag_name)
		}
		err = option_set_value(option_set_value_input{
			Option:        &active_command.Flags[command_index],
			Value:         value,
			Value_Was_Set: value_was_set,
			Name:          flag_name,
		})
		if err != nil {
			return err
		}
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
	if argument.Is_Variadic {
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
