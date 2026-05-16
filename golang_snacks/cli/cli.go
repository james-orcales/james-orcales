// Package cli provides a minimal command-line interface parser.
//
// It supports commands with positional arguments and optional flags.
// Arguments must appear before flags in the command line.
// Supported types: string, int, bool.
//
// Example:
//
//	program := cli.New("myapp", "does something useful",
//		cli.Command{
//			Label: "add",
//			Arguments: []cli.Option{
//				{Label: "name", Value: "", Description: "item name"},
//			},
//			Flags: []cli.Option{
//				{Label: "priority", Value: "low", Description: "urgency level"},
//			},
//		},
//	)
//	command, err := program.Parse(os.Args)
//	if err != nil {
//		program.PrintHelp()
//		os.Exit(1)
//	}
package cli

import (
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/james-orcales/james-orcales/golang_snacks/invariant"
)

// Program represents a command-line application with one or more commands.
type Program struct {
	Label       string
	Description string
	Commands    []Command
	GlobalFlags []Option
}

// Command represents a single command within a program.
// Commands have a label, optional arguments (required, ordered),
// and optional flags (optional, unordered).
type Command struct {
	Label       string
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
	Label       string
	Description string
	Value       any
	IsFlag      bool
}

type NewArgInput struct {
	Label       string
	Description string
}

// NewArg creates a required positional argument with the specified type.
// Arguments always have zero values; custom defaults are not allowed.
// Type parameter T must be string, int, or bool.
func NewArg[T string | int | bool](input NewArgInput) (opt Option) {
	var zeroValue T
	return Option{
		Label:       input.Label,
		Description: input.Description,
		Value:       zeroValue,
		IsFlag:      false,
	}
}

type NewFlagInput[T string | int | bool] struct {
	Label       string
	Value       T
	Description string
}

// NewFlag creates an optional flag with a default value.
// Type parameter T must be string, int, or bool.
func NewFlag[T string | int | bool](input NewFlagInput[T]) (opt Option) {
	return Option{
		Label:       input.Label,
		Description: input.Description,
		Value:       input.Value,
		IsFlag:      true,
	}
}

type NewInput struct {
	Label       string
	Description string
	GlobalFlags []Option
	Commands    []Command
}

// New creates a new Program with the given label, description, global flags, and commands.
// It validates that all commands have labels and that arguments/flags are properly configured.
// Panics if validation fails (e.g., empty labels, unsupported types, invalid flag names).
//
// Supported argument types: string, int
// Supported flag types: string, int, bool
// Flag labels cannot contain dashes or spaces (use underscores instead).
func New(input NewInput) (prog Program) {
	panic_when(len(input.Commands) == 0, "Program has zero commands specified")

	// Validate global flags
	for i, flag := range input.GlobalFlags {
		panic_when(flag.Label == "", "Global flag #%d has no label", i)
		panic_when(!flag.IsFlag, "Global flags must be created with NewFlag")
		panic_when(
			strings.Contains(flag.Label, "_"),
			"Flags cannot contain underscores. Instead of %q, use %q",
			flag.Label,
			strings.ReplaceAll(flag.Label, "_", "-"),
		)
		panic_when(
			strings.Contains(flag.Label, " "),
			"Flags cannot contain spaces: %q",
			flag.Label,
		)
		switch flag.Value.(type) {
		default:
			panic_when(true, "Global flag %q has unsupported type: %T", flag.Label, flag.Value)
		case string, bool, int:
		}
	}

	program := Program{
		Label:       input.Label,
		Description: input.Description,
		Commands:    input.Commands,
		GlobalFlags: input.GlobalFlags,
	}

	for i, command := range program.Commands {
		panic_when(command.Label == "", "Program.Commands[%d].Label is unset", i)
		for argIdx, arg := range command.Arguments {
			panic_when(arg.Label == "", "Argument #%d for command %q has no label", argIdx, command.Label)
			switch arg.Value.(type) {
			default:
				panic_when(true, "Argument %q has unsupported type: %T", arg.Label, arg.Value)
			case string, int:
			}
		}
		for flagIdx, flag := range command.Flags {
			panic_when(flag.Label == "", "Flag #%d for command %q has no label", flagIdx, command.Label)
			panic_when(
				strings.Contains(flag.Label, "_"),
				"Flags cannot contain underscores. Instead of %q, use %q",
				flag.Label,
				strings.ReplaceAll(flag.Label, "_", "-"),
			)
			panic_when(
				strings.Contains(flag.Label, " "),
				"Flags cannot contain spaces: %q",
				flag.Label,
			)
			// Check for conflicts with global flags
			for _, globalFlag := range input.GlobalFlags {
				panic_when(
					flag.Label == globalFlag.Label,
					"Command %q has flag %q that conflicts with global flag",
					command.Label,
					flag.Label,
				)
			}
			switch flag.Value.(type) {
			default:
				panic_when(true, "Flag %q has unsupported type: %T", flag.Label, flag.Value)
			case string, bool, int:
			}
		}
	}
	return program
}

// Parse parses command-line arguments and returns the active command with populated values.
// The os_args slice should be os.Args from the main function.
// If no command is specified, the first declared command is used.
// Global flags are parsed and stored in program.GlobalFlags.
//
// Returns an error if:
//   - The command name is unrecognized
//   - Wrong number of arguments provided
//   - Positional arguments appear after flags
//   - Unknown flags are used
//   - Flag values are invalid or missing
//   - Type conversion fails (e.g., non-numeric value for int flag)
//
// After successful parsing, the returned Command has all Option.Value fields
// populated with user input, and program.GlobalFlags contains parsed global flag values.
func Parse(program *Program, os_args []string) (active_command Command, err error) {
	panic_when(len(os_args) == 0, "program.Parse needs at least one os_arg")

	// Deep copy global flags to avoid modifying the program's global flag definitions
	originalGlobalFlags := program.GlobalFlags
	program.GlobalFlags = make([]Option, len(originalGlobalFlags))
	copy(program.GlobalFlags, originalGlobalFlags)

	defer func() {
		invariant.Sometimes(len(active_command.Arguments) == 0, "Command does not take arguments")
		invariant.Sometimes(len(active_command.Arguments) > 0, "Command takes arguments")
		invariant.Sometimes(len(active_command.Flags) == 0, "Command does not support flags")
		invariant.Sometimes(len(active_command.Flags) > 0, "Command supports flags")
	}()

	// === Finding active_command command ===
	commandIndex := 0
	active_command = program.Commands[0]
	if len(os_args) == 1 {
		invariant.Reachable("Defaulted to first declared command")
		// Deep copy for early return
		active_command.Arguments = make([]Option, len(program.Commands[0].Arguments))
		copy(active_command.Arguments, program.Commands[0].Arguments)
		active_command.Flags = make([]Option, len(program.Commands[0].Flags))
		copy(active_command.Flags, program.Commands[0].Flags)
		return active_command, err
	} else {
		found := false
		for i, command := range program.Commands {
			if command.Label == os_args[1] {
				active_command = command
				commandIndex = i
				found = true
				break
			}
		}
		if !found {
			invariant.Reachable("User specified an unknown command")
			return active_command, fmt.Errorf("%q is an unknown command", os_args[1])
		}
	}

	// Deep copy Arguments and Flags to avoid modifying the program's command definitions
	active_command.Arguments = make([]Option, len(program.Commands[commandIndex].Arguments))
	copy(active_command.Arguments, program.Commands[commandIndex].Arguments)
	active_command.Flags = make([]Option, len(program.Commands[commandIndex].Flags))
	copy(active_command.Flags, program.Commands[commandIndex].Flags)

	// === Collecting ===
	positional_arguments := make([]string, 0, len(active_command.Arguments))
	flags := make([]string, 0, len(active_command.Flags))
	if len(os_args) > 2 {
		start_of_flags := -1
		for i, argument := range os_args[2:] {
			if strings.HasPrefix(argument, "-") && argument != "-" {
				start_of_flags = i + 2
				break
			}
			positional_arguments = append(positional_arguments, argument)
		}
		if start_of_flags >= 0 && start_of_flags < len(os_args) {
			for _, flag := range os_args[start_of_flags:] {
				if strings.HasPrefix(flag, "-") {
					flags = append(flags, flag)
				} else {
					invariant.Reachable("User provides flags before positional arguments")
					return active_command, fmt.Errorf("Positional arguments cannot appear after flags. Got %q", flag)
				}
			}
		}
	}
	invariant.Sometimes(len(flags) == 0, "No flags were set")
	invariant.Sometimes(len(flags) < len(active_command.Flags), "Some flags were set")
	invariant.Sometimes(len(flags) == len(active_command.Flags), "All flags were set")

	if len(positional_arguments) != len(active_command.Arguments) {
		invariant.Reachable("User provided inexact number of arguments")
		return active_command, fmt.Errorf(
			"%q expects %d arguments. Got %d",
			active_command.Label,
			len(active_command.Arguments),
			len(positional_arguments),
		)
	}
	if len(flags) > len(active_command.Flags)+len(program.GlobalFlags) {
		invariant.Reachable("User provided too many flags")
		return active_command, fmt.Errorf(
			"%q supports %d command flags and %d global flags. Got %d",
			active_command.Label,
			len(active_command.Flags),
			len(program.GlobalFlags),
			len(flags),
		)
	}

	// === Parsing ===
	for i, positional_argument := range positional_arguments {
		switch active_command.Arguments[i].Value.(type) {
		default:
			panic_when(true, "unreachable")
		case string:
			invariant.Reachable("User provided a string positional argument")
			active_command.Arguments[i].Value = positional_argument
		case int:
			invariant.Reachable("User provided an int positional argument")
			num, parseErr := strconv.Atoi(positional_argument)
			if parseErr != nil {
				return active_command, fmt.Errorf("%s is an invalid number", positional_argument)
			}
			active_command.Arguments[i].Value = num
		}
	}
	for _, flag := range flags {
		if strings.HasPrefix(flag, "--") && len(flag) > len("--") {
			return active_command, fmt.Errorf("Flags are denoted by a single dash, not double. use '-%s'", flag[2:])
		}
		invariant.Always(flag != "-", "Lone dashes are treated as postional arguments")
		flag = flag[1:]
		flagName, value, value_was_set := strings.Cut(flag, "=")

		// Try global flags first
		globalIndex := slices.IndexFunc(program.GlobalFlags, func(option Option) bool {
			return option.Label == flagName
		})
		if globalIndex >= 0 {
			if _, is_bool := program.GlobalFlags[globalIndex].Value.(bool); !is_bool && (!value_was_set || value == "") {
				return active_command, fmt.Errorf(
					"%q expects a value. You must set flag values with this syntax: -foo-bar=baz.",
					flagName,
				)
			}
			switch program.GlobalFlags[globalIndex].Value.(type) {
			case bool:
				program.GlobalFlags[globalIndex].Value = true
			case string:
				program.GlobalFlags[globalIndex].Value = trimQuotes(value)
			case int:
				num, parseErr := strconv.Atoi(value)
				if parseErr != nil {
					return active_command, fmt.Errorf("%s is an invalid number", value)
				}
				program.GlobalFlags[globalIndex].Value = num
			}
			continue
		}

		// Try command-specific flags
		i := slices.IndexFunc(active_command.Flags, func(option Option) bool {
			return option.Label == flagName
		})
		if i < 0 {
			invariant.Reachable("User provided unknown flag")
			return active_command, fmt.Errorf("%q is an unknown flag", flagName)
		} else if _, is_bool := active_command.Flags[i].Value.(bool); !is_bool && (!value_was_set || value == "") {
			invariant.Reachable("User did not set a value to a non-bool flag")
			return active_command, fmt.Errorf(
				"%q expects a value. You must set flag values with this syntax: -foo-bar=baz.",
				flagName,
			)
		}
		switch active_command.Flags[i].Value.(type) {
		case bool:
			invariant.Reachable("User set a boolean flag")
			active_command.Flags[i].Value = true
		case string:
			invariant.Reachable("User set a string flag")
			active_command.Flags[i].Value = trimQuotes(value)
		case int:
			invariant.Reachable("User set an int flag")
			num, parseErr := strconv.Atoi(value)
			if parseErr != nil {
				return active_command, fmt.Errorf("%s is an invalid number", value)
			}
			active_command.Flags[i].Value = num
		}
	}

	return active_command, nil
}

// trimQuotes removes surrounding quotes from a string value.
// Handles both double quotes (") and single quotes (').
// Only trims if the string starts and ends with matching quotes.
func trimQuotes(s string) (out string) {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// PrintHelp writes a formatted help message to out.
// The output includes the program description, optional preamble text, usage syntax,
// global flags, and details for each command with its arguments and flags.
func PrintHelp(out io.Writer, program Program) {
	// Header
	fmt.Fprintf(out, "%s %s\n\n", program.Label, program.Description)

	fmt.Fprintf(out, "Usage:\n    %s <command> <arguments> [-flags[=value]]\n\n", program.Label)

	// Global flags
	if len(program.GlobalFlags) > 0 {
		blue := func(s string) string { return "\033[34m" + s + "\033[0m" }
		fmt.Fprintln(out, "Global Flags:")
		w := tabwriter.NewWriter(out, 0, 8, 0, ' ', 0)
		for _, flag := range program.GlobalFlags {
			valType := ""
			defaultVal := fmt.Sprintf("%v", flag.Value)

			isBool := false
			if _, is := flag.Value.(bool); is {
				isBool = true
			} else {
				valType = fmt.Sprintf("=%T", flag.Value)
			}

			// Format empty strings as ""
			if str, isString := flag.Value.(string); isString && str == "" {
				defaultVal = `""`
			}

			if isBool {
				fmt.Fprintf(w, "    -%s\t  %s\n", blue(flag.Label), flag.Description)
			} else {
				fmt.Fprintf(
					w,
					"    -%s%s\t  (default: %s)\t  %s\n",
					blue(flag.Label),
					valType,
					defaultVal,
					flag.Description,
				)
			}
			fmt.Fprintf(w, "\n")
		}
		w.Flush()
		fmt.Fprintln(out, "")
	}

	fmt.Fprintln(out, "Available Commands:")

	for _, cmd := range program.Commands {
		// Use tabwriter per command to avoid tab alignment across commands
		w := tabwriter.NewWriter(out, 0, 8, 0, ' ', 0)

		blue := func(s string) string { return "\033[34m" + s + "\033[0m" }
		signature := blue(cmd.Label) + " "
		for _, arg := range cmd.Arguments {
			signature += fmt.Sprintf("<%s:%T> ", arg.Label, arg.Value)
		}
		fmt.Fprintf(w, "    %s\t%s\n", signature, cmd.Description)

		if len(cmd.Flags) > 0 {
			fmt.Fprintln(w, "")

			for _, flag := range cmd.Flags {
				valType := ""
				defaultVal := fmt.Sprintf("%v", flag.Value)

				isBool := false
				if _, is := flag.Value.(bool); is {
					isBool = true
				} else {
					valType = fmt.Sprintf("=%T", flag.Value)
				}

				// Format empty strings as ""
				if str, isString := flag.Value.(string); isString && str == "" {
					defaultVal = `""`
				}

				if isBool {
					fmt.Fprintf(w, "        -%s\t  %s\n", flag.Label, flag.Description)
				} else {
					fmt.Fprintf(
						w,
						"        -%s%s\t  (default: %s)\t  %s\n",
						flag.Label,
						valType,
						defaultVal,
						flag.Description,
					)
				}
			}
		}

		// Flush per command and add blank line between commands
		w.Flush()
		fmt.Fprintln(out, "")
	}
}

// GetOption retrieves an option by label from a slice of options.
// This is typically used to extract argument or flag values from a parsed command.
// Panics if the label is not found.
//
// Example:
//
//	task := cli.GetOption(command.Arguments, "task").Value.(string)
//	priority := cli.GetOption(command.Flags, "priority").Value.(string)
func GetOption(flags []Option, label string) (opt Option) {
	for _, flag := range flags {
		if flag.Label == label {
			return flag
		}
	}
	panic_when(true, "%q is an unknown option", label)
	return Option{}
}

func panic_when(cond bool, message string, data ...any) {
	if cond {
		panic(fmt.Sprintf(message, data...))
	}
}
