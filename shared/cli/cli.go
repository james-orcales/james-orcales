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
package cli

import (
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/james-orcales/james-orcales/shared/invariant"
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
}

// Command represents a single command within a program.
// Commands have a label, optional arguments (required, ordered),
// and optional flags (optional, unordered).
type Command struct {
	// Label is the command name typed on the command line.
	Label string
	// Description is the one-line command summary shown in help output.
	Description string
	// Arguments are required and ordered. They must ALL appear before flags.
	Arguments []Option
	// Flags are optional and unordered.
	Flags []Option
}

// Option represents either an argument or a flag for a command.
// The Value field determines the type (string, int, or bool for flags).
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
		for argument_index, argument := range command.Arguments {
			panic_when(argument.Label == "",
				"Argument #%d for command %q has no label.",
				argument_index, command.Label)
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
			for _, global_flag := range input.Global_Flags {
				panic_when(flag.Label == global_flag.Label,
					"Command %q has flag %q that conflicts with a global flag.",
					command.Label, flag.Label)
			}
		}
	}
	return program
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
		invariant.Sometimes(argument_count == 0, "Command does not take arguments.")
		invariant.Sometimes(argument_count > 0, "Command takes arguments.")
		invariant.Sometimes(flag_count == 0, "Command does not support flags.")
		invariant.Sometimes(flag_count > 0, "Command supports flags.")
	}()

	active_command, err = program_resolve_command(program, operating_system_args)
	if err != nil {
		return active_command, err
	}
	if len(operating_system_args) == 1 {
		return active_command, nil
	}

	positional_arguments, flags, err := command_collect_args(
		active_command, operating_system_args)
	if err != nil {
		return active_command, err
	}
	invariant.Sometimes(len(flags) == 0, "No flags were set.")
	invariant.Sometimes(len(flags) < len(active_command.Flags), "Some flags were set.")
	invariant.Sometimes(len(flags) == len(active_command.Flags), "All flags were set.")

	if len(positional_arguments) != len(active_command.Arguments) {
		return active_command, fmt.Errorf(
			"%q expects %d arguments. Got %d",
			active_command.Label,
			len(active_command.Arguments),
			len(positional_arguments),
		)
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

// Finds the active command named by the args (defaulting to the first command)
// and deep-copies its Arguments and Flags so parsing never mutates the program.
func program_resolve_command(
	program *Program, operating_system_args []string,
) (active_command Command, err error) {
	active_command = program.Commands[0]
	command_index := 0
	if len(operating_system_args) > 1 {
		found := false
		for index, command := range program.Commands {
			if command.Label == operating_system_args[1] {
				active_command = command
				command_index = index
				found = true
				break
			}
		}
		if !found {
			return active_command, fmt.Errorf(
				"%q is an unknown command", operating_system_args[1])
		}
	}

	source := program.Commands[command_index]
	active_command.Arguments = make([]Option, len(source.Arguments))
	copy(active_command.Arguments, source.Arguments)
	active_command.Flags = make([]Option, len(source.Flags))
	copy(active_command.Flags, source.Flags)
	return active_command, nil
}

// Splits the args after the command into positional arguments (before the first
// flag) and flags, erroring on a positional that appears after a flag.
func command_collect_args(active_command Command, operating_system_args []string) (
	positional_arguments []string, flags []string, err error,
) {
	positional_arguments = make([]string, 0, len(active_command.Arguments))
	flags = make([]string, 0, len(active_command.Flags))
	if len(operating_system_args) <= 2 {
		return positional_arguments, flags, nil
	}

	start_of_flags := -1
	for index, argument := range operating_system_args[2:] {
		if strings.HasPrefix(argument, "-") {
			if argument != "-" {
				start_of_flags = index + 2
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
// result back into the options.
func parse_arguments(arguments []Option, positional_arguments []string) (err error) {
	for index, positional_argument := range positional_arguments {
		switch arguments[index].Value.(type) {
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
		invariant.Always(flag != "-", "Lone dashes are treated as positional arguments.")
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
			signature += fmt.Sprintf("<%s:%T> ", argument.Label, argument.Value)
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
