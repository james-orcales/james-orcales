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
//
// An argument or flag may be an enum (New_Enum_Argument, New_Enum_Flag), restricting its
// value to a fixed set; parsing rejects anything outside it.
package cli

import (
	"errors"
	"fmt"
	"io"
	"path"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"

	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/levenshtein"
)

// Help_Requested is returned by Program_Parse when the command line carries the
// auto-injected -help flag. It short-circuits parsing — even a missing required argument
// does not preempt it — so the caller renders help and exits successfully. Match it with
// errors.Is, then call Print_Requested_Help with the returned command.
var Help_Requested = errors.New("help requested")

// HELP_LABEL is the reserved option label for the auto-injected help flag, so -help works
// on every program without being declared. A user option may not claim it.
const HELP_LABEL = "help"

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
	// Multicall marks a program whose command is selected by the binary name in
	// argv[0], as a busybox-style symlinked binary is. New_Multicall sets it.
	Multicall bool
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
	// Hidden omits the command from help and completion; it still resolves when named.
	Hidden bool
	// Deprecated, when non-empty, is the guidance shown when the command is used. A
	// deprecated command is hidden like a hidden one and warns on use.
	Deprecated string
	// Deprecation_Warnings is filled by Program_Parse on the returned command: one
	// ready-to-print message per deprecated command or flag the invocation actually
	// used. It is empty on a command held as a definition.
	Deprecation_Warnings []string
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
	// Enum, when non-nil, restricts Value to a permitted set; it holds a []string or
	// []int whose element type matches Value. New_Enum_Flag and New_Enum_Argument set
	// it, and parsing rejects any value outside it.
	Enum any
	// Is_Flag distinguishes a flag from a positional argument.
	Is_Flag bool
	// Hidden omits the flag from help and completion; it still parses when named.
	// Meaningful for flags, not positional arguments, which always appear in usage.
	Hidden bool
	// Deprecated, when non-empty, is the guidance shown when the flag is used. A
	// deprecated flag is hidden like a hidden one and warns on use.
	Deprecated string
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
	// Hidden omits the flag from help and completion; it still parses when named.
	Hidden bool
	// Deprecated, when non-empty, is the guidance shown when the flag is used.
	Deprecated string
}

// New_Flag creates an optional flag with a default value.
// Type parameter T must be string, int, or bool.
func New_Flag[T string | int | bool](input New_Flag_Input[T]) (option Option) {
	return Option{
		Label:       input.Label,
		Description: input.Description,
		Value:       input.Value,
		Is_Flag:     true,
		Hidden:      input.Hidden,
		Deprecated:  input.Deprecated,
	}
}

// New_Enum_Flag_Input is the input for New_Enum_Flag.
type New_Enum_Flag_Input[T string | int] struct {
	// Label is the flag name.
	Label string
	// Enum is the set of permitted values. It may not be empty, and Value must be one
	// of its members.
	Enum []T
	// Value is the flag's default value.
	Value T
	// Description is the one-line summary shown in help output.
	Description string
	// Hidden omits the flag from help and completion; it still parses when named.
	Hidden bool
	// Deprecated, when non-empty, is the guidance shown when the flag is used.
	Deprecated string
}

// New_Enum_Flag creates an optional flag whose value must be one of Enum, defaulting to
// Value. New panics when Enum is empty or Value is not a member. Type parameter T must
// be string or int; bool is excluded because a bool already enumerates its two values.
func New_Enum_Flag[T string | int](input New_Enum_Flag_Input[T]) (option Option) {
	return Option{
		Label:       input.Label,
		Description: input.Description,
		Value:       input.Value,
		Enum:        input.Enum,
		Is_Flag:     true,
		Hidden:      input.Hidden,
		Deprecated:  input.Deprecated,
	}
}

// New_Enum_Argument_Input is the input for New_Enum_Argument.
type New_Enum_Argument_Input[T string | int] struct {
	// Label is the argument name.
	Label string
	// Enum is the set of permitted values. It may not be empty.
	Enum []T
	// Description is the one-line summary shown in help output.
	Description string
}

// New_Enum_Argument creates a required positional argument whose value must be one of
// Enum. Like New_Argument it has no default; the required check enforces that it is
// supplied, and the enum check enforces membership. New panics when Enum is empty. Type
// parameter T must be string or int.
func New_Enum_Argument[T string | int](input New_Enum_Argument_Input[T]) (option Option) {
	var zero_value T
	return Option{
		Label:       input.Label,
		Description: input.Description,
		Value:       zero_value,
		Enum:        input.Enum,
		Is_Flag:     false,
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

	// -help is auto-injected as a global flag so every command carries it in help output
	// and its label is reserved; the injection precedes validation so a colliding user
	// option is caught, and reserve_help_label reports it with a clearer message first.
	reserve_help_label(input.Commands, input.Global_Flags)
	global_flags := append([]Option{help_flag()}, input.Global_Flags...)

	for index, flag := range global_flags {
		panic_when(!flag.Is_Flag, "Global flags must be created with New_Flag.")
		validate_flag_label(fmt.Sprintf("Global flag #%d", index), flag)
	}

	program = Program{
		Label:        input.Label,
		Description:  input.Description,
		Commands:     input.Commands,
		Global_Flags: global_flags,
	}

	for index, command := range program.Commands {
		panic_when(command.Label == "", "Program.Commands[%d].Label is unset.", index)
		command_validate_options(command, global_flags)
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
	// A single-command program still gets the auto-injected -help global flag.
	reserve_help_label([]Command{command}, nil)
	global_flags := []Option{help_flag()}
	command_validate_options(command, global_flags)
	return Program{
		Label:        input.Label,
		Description:  input.Description,
		Commands:     []Command{command},
		Global_Flags: global_flags,
		Single:       true,
	}
}

// The bool flag auto-injected into every program's Global_Flags: it documents -help
// in help output and reserves the label. The parser recognizes the token directly
// (see program_wants_help), so this flag's value is never read.
func help_flag() (option Option) {
	return New_Flag(New_Flag_Input[bool]{Label: HELP_LABEL, Description: "show this help"})
}

// Panics when a user option claims the reserved HELP_LABEL, so the auto-injected -help
// flag never collides with or shadows a real option.
func reserve_help_label(commands []Command, global_flags []Option) {
	for _, flag := range global_flags {
		panic_when(flag.Label == HELP_LABEL,
			"%q is reserved for the auto-injected -help flag", HELP_LABEL)
	}
	for _, command := range commands {
		for _, argument := range command.Arguments {
			panic_when(argument.Label == HELP_LABEL,
				"%q is reserved for the auto-injected -help flag", HELP_LABEL)
		}
		for _, flag := range command.Flags {
			panic_when(flag.Label == HELP_LABEL,
				"%q is reserved for the auto-injected -help flag", HELP_LABEL)
		}
	}
}

// New_Multicall_Input is the input for New_Multicall.
type New_Multicall_Input struct {
	// Label is the program name shown in help output.
	Label string
	// Description is the one-line program summary.
	Description string
	// Global_Flags are flags accepted by every command.
	Global_Flags []Option
	// Commands are the program's commands, each selected by its label matching the
	// binary name in argv[0].
	Commands []Command
}

// New_Multicall creates a program whose command is selected by the binary name in
// argv[0], not a token in slot 1 — the model of a busybox-style binary symlinked to
// each of its command names. It validates commands and global flags exactly as New
// does, panicking when validation fails.
func New_Multicall(input New_Multicall_Input) (program Program) {
	program = New(New_Input{
		Label:        input.Label,
		Description:  input.Description,
		Global_Flags: input.Global_Flags,
		Commands:     input.Commands,
	})
	program.Multicall = true
	return program
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
		validate_enum(
			fmt.Sprintf("Argument %q for command %q", argument.Label, command.Label),
			argument)
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
	validate_enum(context, flag)
}

// Panics when an option carries a malformed enum: an Enum whose element type is not
// []string/[]int or does not match Value's type, an empty Enum, an enum on a variadic
// option, or — for a flag, which has a real default — a default Value outside the set.
// A non-enum option (Enum nil) is left untouched. An argument's default is its zero
// value, deliberately not checked for membership: the required check enforces that the
// argument is supplied, and the parse-time enum check enforces membership then.
func validate_enum(context string, option Option) {
	if option.Enum == nil {
		return
	}
	// A variadic collects many values; an enum constrains one. Combining them has no
	// defined meaning, so it is rejected rather than silently ignoring one.
	panic_when(option_is_slice(option), "%s is variadic and cannot be an enum.", context)
	switch enum := option.Enum.(type) {
	default:
		panic_when(true, "%s has an unsupported enum type: %T", context, option.Enum)
	case []string:
		_, matches := option.Value.(string)
		panic_when(!matches,
			"%s has a []string enum but a %T value.", context, option.Value)
		panic_when(len(enum) == 0, "%s has an empty enum.", context)
		panic_when(option.Is_Flag && !slices.Contains(enum, option.Value.(string)),
			"%s default %q is not one of its enum values.", context, option.Value)
	case []int:
		_, matches := option.Value.(int)
		panic_when(!matches, "%s has a []int enum but a %T value.", context, option.Value)
		panic_when(len(enum) == 0, "%s has an empty enum.", context)
		panic_when(option.Is_Flag && !slices.Contains(enum, option.Value.(int)),
			"%s default %d is not one of its enum values.", context, option.Value)
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
			"cli.parse.command_shape",
			invariant.Sometimes(argument_count > 0, "command has positional arguments"),
			invariant.Sometimes(flag_count > 0, "command has flags"),
		)
	}()

	// -help short-circuits before command resolution, so it works even when the command
	// slot holds -help itself or a required argument is missing. The caller renders the
	// resolved context with Print_Requested_Help and exits successfully.
	if program_wants_help(operating_system_args[1:]) {
		return program_help_context(program, operating_system_args), Help_Requested
	}

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
	err = command_assign_positionals(&Command_Assign_Positionals_Input{
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
	active_command.Deprecation_Warnings = collect_deprecations(program, active_command, filled)
	return active_command, nil
}

// Gathers a warning for each deprecated command or flag the invocation used: the
// resolved command when it is itself deprecated (invoking it is using it), and every
// deprecated command flag or global flag that was set by name (present in filled).
func collect_deprecations(
	program *Program, command Command, filled map[string]bool,
) (warnings []string) {
	warnings = []string{}
	if command.Deprecated != "" {
		message := fmt.Sprintf("command %q is deprecated: %s",
			command.Label, command.Deprecated)
		warnings = append(warnings, message)
	}
	warnings = append(warnings, deprecated_flags_used(command.Flags, filled)...)
	warnings = append(warnings, deprecated_flags_used(program.Global_Flags, filled)...)
	return warnings
}

// Gathers a warning for each deprecated flag that was set by name.
func deprecated_flags_used(flags []Option, filled map[string]bool) (warnings []string) {
	warnings = []string{}
	for index := range flags {
		flag := flags[index]
		if flag.Deprecated == "" {
			continue
		}
		if !filled[flag.Label] {
			continue
		}
		warnings = append(warnings,
			fmt.Sprintf("-%s is deprecated: %s", flag.Label, flag.Deprecated))
	}
	return warnings
}

// Reports whether the arguments carry the auto-injected -help flag. Help is detected
// before command resolution so it works despite a missing or malformed command line.
func program_wants_help(tokens []string) (wants bool) {
	for _, token := range tokens {
		if token == "-"+HELP_LABEL {
			return true
		}
	}
	return false
}

// Resolves the command a -help request refers to, for Print_Requested_Help. A multicall
// program uses the argv[0] verb; a multi-command program uses the slot-1 command when it
// names one. Anything else — a single-command program, an unknown or absent command —
// yields the zero-value root context (empty Label), which renders the whole program.
func program_help_context(program *Program, operating_system_args []string) (context Command) {
	if program.Multicall {
		name := path.Base(operating_system_args[0])
		index, err := program_select_command(program, name)
		// A self-invoked multicall binary (run by its own name) names the verb in the
		// first token, so -help there resolves that verb, not the root.
		if err != nil {
			if len(operating_system_args) > 1 {
				token := operating_system_args[1]
				index, err = program_select_command(program, token)
			}
		}
		if err != nil {
			return Command{}
		}
		return program.Commands[index]
	}
	if program.Single {
		return Command{}
	}
	if len(operating_system_args) > 1 {
		index, err := program_select_command(program, operating_system_args[1])
		if err != nil {
			return Command{}
		}
		return program.Commands[index]
	}
	return Command{}
}

// One command-line token paired with its position, so a slice argument can reassemble
// its positional and -label=value contributions in the order they were written.
type Indexed_Token struct {
	// Index is the token's position in the original argument list.
	Index int
	// Value is the token's text as written on the command line.
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
				"use a single dash: -%s, not --%s", token[2:], token[2:])
		}
	}
	// A lone "-" was already excluded by is_named_token.
	invariant.Always(token != "-", "named token is not a lone dash")
	label, value, value_was_set = strings.Cut(token[1:], "=")
	return label, value, value_was_set, nil
}

// Applies every -label=value token to its option and returns the bare positionals
// plus, for the slice argument, the values given by name — both tagged with their
// position so the slice preserves command-line order. filled records the scalar
// options set by name: a scalar named twice is an error, and a named scalar argument
// is skipped by the positional pass.
func program_assign_named(program *Program, command Command, tokens []string) (
	filled map[string]bool, positionals []Indexed_Token, slice_named []Indexed_Token, err error,
) {
	filled = map[string]bool{}
	for index, token := range tokens {
		if !is_named_token(token) {
			positionals = append(positionals, Indexed_Token{Index: index, Value: token})
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
			slice_named = append(slice_named, Indexed_Token{Index: index, Value: value})
			continue
		}
		if filled[label] {
			return filled, positionals, slice_named,
				fmt.Errorf("-%s may only be given once", label)
		}
		set_err := option_set_value(Option_Set_Value_Input{
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
	match, ok := levenshtein.Closest(levenshtein.Closest_Input{
		Target: label, Candidates: program_option_labels(program, command),
	})
	if ok {
		return nil, false,
			fmt.Errorf("unknown option -%s, did you mean -%s?", label, match)
	}
	return nil, false, fmt.Errorf("unknown option -%s", label)
}

// Reports whether an option appears in help and completion. A hidden or deprecated
// flag still parses; it is only kept out of what the tool advertises.
func option_shown(option Option) (shown bool) {
	if option.Hidden {
		return false
	}
	return option.Deprecated == ""
}

// Reports whether a command appears in help and completion. A hidden or deprecated
// command still resolves; it is only kept out of what the tool advertises.
func command_shown(command Command) (shown bool) {
	if command.Hidden {
		return false
	}
	return command.Deprecated == ""
}

// Returns the shown commands' labels, the candidate set for a did-you-mean suggestion
// and for completion. Hidden and deprecated commands are omitted, so neither surface
// advertises them; they still resolve by exact name in program_select_command.
func program_command_labels(program *Program) (labels []string) {
	labels = []string{}
	for index := range program.Commands {
		if command_shown(program.Commands[index]) {
			labels = append(labels, program.Commands[index].Label)
		}
	}
	return labels
}

// Returns the option labels reachable by name — the command's arguments and its shown
// flags plus the shown global flags — the candidate set for a did-you-mean suggestion
// and for completion. Hidden and deprecated flags are omitted; arguments always show.
func program_option_labels(program *Program, command Command) (labels []string) {
	for index := range command.Arguments {
		labels = append(labels, command.Arguments[index].Label)
	}
	for index := range command.Flags {
		if option_shown(command.Flags[index]) {
			labels = append(labels, command.Flags[index].Label)
		}
	}
	for index := range program.Global_Flags {
		if option_shown(program.Global_Flags[index]) {
			labels = append(labels, program.Global_Flags[index].Label)
		}
	}
	return labels
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
	if program.Multicall {
		arguments_start = 1
		name := path.Base(operating_system_args[0])
		command_index, err = program_select_command(program, name)
		// Busybox self-invocation: run by its own name rather than a verb link, the
		// binary takes the command from the first token (as `busybox ls` does), so a
		// bootstrap verb can run before the links exist. The error from the slot-1 token
		// replaces the argv[0] one, naming what the user actually typed as the command.
		if err != nil {
			if len(operating_system_args) > 1 {
				arguments_start = 2
				token := operating_system_args[1]
				command_index, err = program_select_command(program, token)
			}
		}
	} else if program.Single {
		arguments_start = 1
	} else if len(operating_system_args) > 1 {
		command_index, err = program_select_command(program, operating_system_args[1])
	}
	if err != nil {
		return program.Commands[0], arguments_start, err
	}

	source := program.Commands[command_index]
	active_command = source
	active_command.Arguments = make([]Option, len(source.Arguments))
	copy(active_command.Arguments, source.Arguments)
	active_command.Flags = make([]Option, len(source.Flags))
	copy(active_command.Flags, source.Flags)
	return active_command, arguments_start, nil
}

// Resolves a command name to its index, suggesting the closest command when the name
// is an unrecognized near-miss. Shared by multicall selection (the argv[0] basename)
// and multi-command selection (the slot-1 token).
func program_select_command(program *Program, name string) (index int, err error) {
	for candidate_index, command := range program.Commands {
		if command.Label == name {
			return candidate_index, nil
		}
	}
	match, ok := levenshtein.Closest(levenshtein.Closest_Input{
		Target: name, Candidates: program_command_labels(program),
	})
	if ok {
		return 0, fmt.Errorf("unknown command %q, did you mean %q?", name, match)
	}
	return 0, fmt.Errorf("unknown command %q", name)
}

// Input for command_assign_positionals.
type Command_Assign_Positionals_Input struct {
	// Command is the active command whose arguments are filled in place.
	Command Command
	// Positionals are the bare tokens, each tagged with its command-line position.
	Positionals []Indexed_Token
	// Slice_Named are the trailing slice argument's -label=value contributions, tagged
	// with position so they merge with the positionals in order.
	Slice_Named []Indexed_Token
	// Filled records the scalar arguments already set by name; it gains those set here.
	Filled map[string]bool
}

// Fills the command's scalar arguments from the positional tokens in declaration
// order, skipping any already set by name, then routes the rest into the trailing
// slice argument together with its named contributions, in command-line order. Errors
// when a positional has no argument to fill. Filled gains every scalar argument set
// here so the required check can see it.
func command_assign_positionals(input *Command_Assign_Positionals_Input) (err error) {
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
	slice_contributions := append([]Indexed_Token{}, input.Slice_Named...)
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
			return fmt.Errorf(
				"too many arguments: %q was not expected", positional.Value)
		}
		slice_contributions = append(slice_contributions, positional)
	}
	if slice_index < 0 {
		return nil
	}
	slices.SortFunc(slice_contributions, func(left, right Indexed_Token) (order int) {
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
		return fmt.Errorf(
			"missing required argument %q; pass it by position or as -%s=value",
			argument.Label, argument.Label)
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
			return fmt.Errorf("%s expects a whole number, but got %q",
				argument.Label, value)
		}
		argument.Value = number
	}
	return option_check_enum(argument, argument.Label)
}

// Builds the slice option's value from its contributions, already sorted into
// command-line order, converting each element to the slice's element type.
func option_set_slice(option *Option, contributions []Indexed_Token) (err error) {
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
				return fmt.Errorf("%s expects a whole number, but got %q",
					option.Label, contribution.Value)
			}
			values = append(values, number)
		}
		option.Value = values
	}
	return nil
}

// Input for option_set_value.
type Option_Set_Value_Input struct {
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
func option_set_value(input Option_Set_Value_Input) (err error) {
	_, is_boolean := input.Option.Value.(bool)
	if !is_boolean {
		absent := !input.Value_Was_Set
		if input.Value == "" {
			absent = true
		}
		if absent {
			return fmt.Errorf(
				"-%s needs a value, e.g. -%s=value", input.Name, input.Name)
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
			return fmt.Errorf("%s expects a whole number, but got %q",
				input.Name, input.Value)
		}
		input.Option.Value = number
	}
	return option_check_enum(input.Option, "-"+input.Name)
}

// Returns an error when a parsed value falls outside an enum option's permitted set, and
// nil when the option is not an enum or the value is a member. For a string enum it
// suggests the closest member on a near miss, mirroring the did-you-mean the parser
// already gives for unknown options; otherwise, and for an int enum, it lists the whole
// set. display_name is the option as the user wrote it — a -flag or a bare argument
// label — for the message.
func option_check_enum(option *Option, display_name string) (err error) {
	switch enum := option.Enum.(type) {
	case nil:
		return nil
	case []string:
		value := option.Value.(string)
		if slices.Contains(enum, value) {
			return nil
		}
		match, ok := levenshtein.Closest(levenshtein.Closest_Input{
			Target: value, Candidates: enum,
		})
		if ok {
			return fmt.Errorf("invalid value %q for %s, did you mean %q?",
				value, display_name, match)
		}
		allowed, _ := enum_values_text(*option, ", ")
		return fmt.Errorf("invalid value %q for %s; allowed: %s",
			value, display_name, allowed)
	case []int:
		value := option.Value.(int)
		if slices.Contains(enum, value) {
			return nil
		}
		allowed, _ := enum_values_text(*option, ", ")
		return fmt.Errorf("invalid value %d for %s; allowed: %s",
			value, display_name, allowed)
	}
	panic_when(true, "unreachable enum type %T", option.Enum)
	return nil
}

// Returns an enum option's permitted values as strings (ints formatted in base 10), or
// is_enum false when the option carries no enum. The single source of an enum's members
// for help, error messages, and completion.
func option_enum_members(option Option) (members []string, is_enum bool) {
	switch enum := option.Enum.(type) {
	case []string:
		return enum, true
	case []int:
		members = make([]string, len(enum))
		for index, number := range enum {
			members[index] = strconv.Itoa(number)
		}
		return members, true
	}
	return nil, false
}

// Formats an enum option's permitted values joined by separator, e.g. "auto|never" for
// help or "auto, never" for an error. is_enum is false when the option carries no enum.
func enum_values_text(option Option, separator string) (text string, is_enum bool) {
	members, is_enum := option_enum_members(option)
	if !is_enum {
		return "", false
	}
	return strings.Join(members, separator), true
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
		Print_Command(output, program, program.Commands[0])
		return
	}
	fmt.Fprintf(output,
		"Usage:\n    %s <command> <arguments> [-flags[=value]]\n", program.Label)
	if program_has_arguments(program) {
		fmt.Fprintln(output, NAMED_ARGUMENT_LEGEND)
	}
	fmt.Fprintln(output, "")

	global_flags := shown_options(program.Global_Flags)
	if len(global_flags) > 0 {
		fmt.Fprintln(output, "Global Flags:")
		writer := tabwriter.NewWriter(output, 0, 8, 0, ' ', 0)
		for _, flag := range global_flags {
			print_help_flag(writer, flag, "    ", true)
			fmt.Fprintf(writer, "\n")
		}
		writer.Flush()
		fmt.Fprintln(output, "")
	}

	fmt.Fprintln(output, "Available Commands:")
	for _, command := range program.Commands {
		if !command_shown(command) {
			continue
		}
		// A tabwriter per command avoids tab alignment bleeding across commands.
		writer := tabwriter.NewWriter(output, 0, 8, 0, ' ', 0)
		signature := "\033[34m" + command.Label + "\033[0m" + " "
		for _, argument := range command.Arguments {
			signature += option_format_signature(argument) + " "
		}
		fmt.Fprintf(writer, "    %s\t%s\n", signature, command.Description)
		command_flags := shown_options(command.Flags)
		if len(command_flags) > 0 {
			fmt.Fprintln(writer, "")
			for _, flag := range command_flags {
				print_help_flag(writer, flag, "        ", false)
			}
		}
		writer.Flush()
		fmt.Fprintln(output, "")
	}
}

// Print_Requested_Help renders the help a Help_Requested parse asks for: the whole
// program — its command catalog, or a single-command program's own usage — when command
// is the zero-value root context, or one command's usage when a command was selected. It
// is the turnkey response to errors.Is(err, cli.Help_Requested).
func Print_Requested_Help(output io.Writer, program Program, command Command) {
	if command.Label == "" {
		Print_Help(output, program)
		return
	}
	Print_Command(output, program, command)
}

// Print_Command writes help for a single command: a usage line carrying the command's
// own name and positional arguments, then its flags and the program's global flags. It
// is the per-verb help a multicall binary shows for the name in argv[0], and the body
// of a single-command program's help.
func Print_Command(output io.Writer, program Program, command Command) {
	signature := ""
	for _, argument := range command.Arguments {
		signature += option_format_signature(argument) + " "
	}
	fmt.Fprintf(output, "Usage:\n    %s %s[-flags[=value]]\n", command.Label, signature)
	if len(command.Arguments) > 0 {
		fmt.Fprintln(output, NAMED_ARGUMENT_LEGEND)
	}
	print_help_flag_section(output, "Flags:", command.Flags)
	print_help_flag_section(output, "Global Flags:", program.Global_Flags)
}

// Print_Deprecations writes a warning line for each deprecated command or flag the
// parse used, gathered by Program_Parse into command.Deprecation_Warnings. A binary
// calls it after a successful parse, before running, so the user is nudged off a
// deprecated option without the option ceasing to work.
func Print_Deprecations(output io.Writer, command Command) {
	for _, warning := range command.Deprecation_Warnings {
		fmt.Fprintf(output, "warning: %s\n", warning)
	}
}

// The options that appear in help and completion, dropping hidden and deprecated ones.
func shown_options(options []Option) (shown []Option) {
	shown = []Option{}
	for index := range options {
		if option_shown(options[index]) {
			shown = append(shown, options[index])
		}
	}
	return shown
}

// Writes a titled section of flag rows, or nothing when no flag in it is shown.
func print_help_flag_section(output io.Writer, title string, flags []Option) {
	shown := shown_options(flags)
	if len(shown) == 0 {
		return
	}
	fmt.Fprintln(output, "")
	fmt.Fprintln(output, title)
	writer := tabwriter.NewWriter(output, 0, 8, 0, ' ', 0)
	for _, flag := range shown {
		print_help_flag(writer, flag, "    ", true)
	}
	writer.Flush()
}

// Formats a positional argument for a usage line: <label: type>, with a trailing
// ellipsis for a variadic. The space after the colon keeps it from reading like a
// key:value the user must type. A variadic shows its element type, not its slice type,
// so a []string reads as <paths: string...> rather than the noisier <paths: []string>.
func option_format_signature(argument Option) (signature string) {
	if option_is_slice(argument) {
		element := "string"
		if _, is_integer := argument.Value.([]int); is_integer {
			element = "int"
		}
		return fmt.Sprintf("<%s: %s...>", argument.Label, element)
	}
	// An enum argument shows its permitted set in place of the bare type, so a usage
	// line reads <format: (json|yaml|toml)> rather than <format: string>.
	if choices, is_enum := enum_values_text(argument, "|"); is_enum {
		return fmt.Sprintf("<%s: (%s)>", argument.Label, choices)
	}
	return fmt.Sprintf("<%s: %T>", argument.Label, argument.Value)
}

// The note printed under the usage line when a program has at least one positional
// argument: each may also be supplied by name, not only by position.
const NAMED_ARGUMENT_LEGEND = "    Positional arguments may also be supplied via -key=val syntax."

// Reports whether any of the program's commands declares a positional argument, the
// condition under which the named-argument legend is worth printing.
func program_has_arguments(program Program) (has bool) {
	for _, command := range program.Commands {
		if len(command.Arguments) > 0 {
			return true
		}
	}
	return false
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
	} else if choices, is_enum := enum_values_text(flag, "|"); is_enum {
		// An enum flag shows its permitted set in place of the bare type, so the reader
		// sees -color=(auto|never|always) rather than an unhelpful -color=string.
		value_type = "=(" + choices + ")"
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

// Complete returns the shell-completion candidates for a partially typed command line.
// words mirrors the argv being completed — words[0] is the program (or, for a multicall
// link, the verb) and the final element is the word under the cursor, possibly empty. It
// reads the live Program, so candidates track command, flag, and enum definitions with no
// separate registration. An empty result means "no cli candidate" — the shell falls back
// to file completion.
func Complete(program Program, words []string) (candidates []string) {
	if len(words) == 0 {
		return nil
	}
	current := words[len(words)-1]

	if program.Single {
		return complete_in_command(&program, program.Commands[0], words, 1, current)
	}
	if program.Multicall {
		index, err := program_select_command(&program, path.Base(words[0]))
		if err == nil {
			return complete_in_command(
				&program, program.Commands[index], words, 1, current)
		}
		// A non-verb argv[0] falls through to self-invocation, where slot 1 selects
		// the command, exactly like a multi-command program.
	}
	// Multi-command (or a self-invoked multicall binary): slot 1 selects the command.
	// While it is still being typed, the candidates are the command names themselves.
	if len(words) <= 2 {
		return filter_prefix(program_command_labels(&program), current)
	}
	index, err := program_select_command(&program, words[1])
	if err != nil {
		return nil
	}
	return complete_in_command(&program, program.Commands[index], words, 2, current)
}

// Completes the word under the cursor within a resolved command: an enum option's members
// after -label=, then flag and named-argument labels for a leading dash, then an enum
// positional's members. Anything else yields nil so the shell completes files.
func complete_in_command(
	program *Program, command Command, words []string, args_start int, current string,
) (candidates []string) {
	if strings.HasPrefix(current, "-") {
		if strings.Contains(current, "=") {
			return complete_enum_value(program, command, current)
		}
		labels := program_option_labels(program, command)
		dashed := make([]string, 0, len(labels))
		for _, label := range labels {
			dashed = append(dashed, "-"+label)
		}
		return filter_prefix(dashed, current)
	}
	position := positional_index(command, words, args_start)
	if position < 0 {
		return nil
	}
	if position >= len(command.Arguments) {
		return nil
	}
	if members, is_enum := option_enum_members(command.Arguments[position]); is_enum {
		return filter_prefix(members, current)
	}
	return nil
}

// Completes a -label=value token to that option's enum members, when label names an
// enum in scope. A non-enum or unknown label yields nil (the shell completes files).
func complete_enum_value(
	program *Program, command Command, current string,
) (candidates []string) {
	equals_offset := strings.Index(current, "=")
	label := current[1:equals_offset]
	partial := current[equals_offset+1:]
	option, found := command_scope_option(program, command, label)
	if !found {
		return nil
	}
	members, is_enum := option_enum_members(*option)
	if !is_enum {
		return nil
	}
	output := []string{}
	for _, member := range members {
		if strings.HasPrefix(member, partial) {
			output = append(output, "-"+label+"="+member)
		}
	}
	return output
}

// Reports the positional slot the cursor sits in: the count of bare (non-flag) words
// already typed after args_start, excluding the word under the cursor. A named argument
// (-label=value) is not counted, matching how completion offers members for the next bare
// slot.
func positional_index(command Command, words []string, args_start int) (position int) {
	for index := args_start; index < len(words)-1; index++ {
		if !strings.HasPrefix(words[index], "-") {
			position++
		}
	}
	return position
}

// Finds an option by label across a command's arguments, its flags, and the program's
// global flags — the same scope program_find_option searches, but read-only and without
// a did-you-mean.
func command_scope_option(
	program *Program, command Command, label string,
) (option *Option, found bool) {
	for index := range command.Arguments {
		if command.Arguments[index].Label == label {
			return &command.Arguments[index], true
		}
	}
	for index := range command.Flags {
		if command.Flags[index].Label == label {
			return &command.Flags[index], true
		}
	}
	for index := range program.Global_Flags {
		if program.Global_Flags[index].Label == label {
			return &program.Global_Flags[index], true
		}
	}
	return nil, false
}

// Returns the candidates that start with prefix, preserving order. Always non-nil so the
// caller can compare lengths.
func filter_prefix(candidates []string, prefix string) (matches []string) {
	matches = []string{}
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, prefix) {
			matches = append(matches, candidate)
		}
	}
	return matches
}

// Handle_Completion is the pre-parse gate a binary calls before Program_Parse: it serves
// the reserved `completion <shell>` (prints the shell script) and `__complete <words...>`
// (prints candidates, one per line) invocations, returning true when it handled one. It
// runs before parsing so it works for a single-command program, whose first token is
// otherwise a positional. IO goes to output, like Print_Help.
func Handle_Completion(program Program, args []string, output io.Writer) (handled bool) {
	if len(args) < 2 {
		return false
	}
	switch args[1] {
	case "__complete":
		for _, candidate := range Complete(program, args[2:]) {
			fmt.Fprintln(output, candidate)
		}
		return true
	case "completion":
		shell := ""
		if len(args) > 2 {
			shell = args[2]
		}
		script, err := Completion_Script(program, shell)
		if err != nil {
			fmt.Fprintln(output, err)
			return true
		}
		fmt.Fprint(output, script)
		return true
	}
	return false
}

// Completion_Script returns a bash, zsh, or fish script that wires the shell's completion
// to call back into `<name> __complete`, so candidates always come from the live Program.
// The script is tiny and static; all knowledge lives in the binary. A multicall program
// emits one registration per verb link. An unsupported shell is an error.
func Completion_Script(program Program, shell string) (script string, err error) {
	names := completion_target_names(program)
	builder := strings.Builder{}
	for _, name := range names {
		block, block_err := completion_block(
			&Completion_Block_Input{Name: name, Shell: shell})
		if block_err != nil {
			return "", block_err
		}
		builder.WriteString(block)
	}
	return builder.String(), nil
}

// The command names the completion script registers: for a multicall program, the
// program's own name (so self-invocation completes verb names) plus each verb link;
// otherwise just the program.
func completion_target_names(program Program) (names []string) {
	if program.Multicall {
		return append([]string{program.Label}, program_command_labels(&program)...)
	}
	return []string{program.Label}
}

// The command name and shell a completion registration is built for, bundled because
// the parameter types repeat.
type Completion_Block_Input struct {
	// Name is the command name the completion registration is built for.
	Name string
	// Shell is the target shell whose completion syntax is emitted.
	Shell string
}

// Builds one shell's completion registration for a single command name.
func completion_block(input *Completion_Block_Input) (block string, err error) {
	switch input.Shell {
	case "bash":
		return fmt.Sprintf(""+
			"_%[1]s_complete() {\n"+
			"    local IFS=$'\\n'\n"+
			"    COMPREPLY=($(%[1]s __complete \"${COMP_WORDS[@]}\"))\n"+
			"}\n"+
			"complete -o default -F _%[1]s_complete %[1]s\n", input.Name), nil
	case "zsh":
		return fmt.Sprintf(""+
			"#compdef %[1]s\n"+
			"_%[1]s_complete() {\n"+
			"    local -a completions\n"+
			"    completions=(\"${(@f)$(%[1]s __complete \"${words[@]}\")}\")\n"+
			"    compadd -- $completions\n"+
			"}\n"+
			"compdef _%[1]s_complete %[1]s\n", input.Name), nil
	case "fish":
		return fmt.Sprintf(""+
			"function _%[1]s_complete\n"+
			"    %[1]s __complete (commandline -opc) (commandline -ct)\n"+
			"end\n"+
			"complete -c %[1]s -f -a '(_%[1]s_complete)'\n", input.Name), nil
	}
	return "", fmt.Errorf("unsupported shell %q; use bash, zsh, or fish", input.Shell)
}
