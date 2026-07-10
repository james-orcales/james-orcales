package cli

import (
	"fmt"
	"io"
	"path"
	"strings"
)

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
			return complete_in_command(&program, program.Commands[index], words, 1, current)
		}
		// argv[0] is not a verb link: fall through to self-invocation, where slot 1
		// selects the command, exactly like a multi-command program.
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
	if strings.HasPrefix(current, "-") && strings.Contains(current, "=") {
		return complete_enum_value(program, command, current)
	}
	if strings.HasPrefix(current, "-") {
		labels := program_option_labels(program, command)
		dashed := make([]string, 0, len(labels))
		for _, label := range labels {
			dashed = append(dashed, "-"+label)
		}
		return filter_prefix(dashed, current)
	}
	position := positional_index(command, words, args_start)
	if position >= 0 && position < len(command.Arguments) {
		if members, is_enum := option_enum_members(command.Arguments[position]); is_enum {
			return filter_prefix(members, current)
		}
	}
	return nil
}

// Completes a -label=value token to that option's enum members, when label names an
// enum in scope. A non-enum or unknown label yields nil (the shell completes files).
func complete_enum_value(program *Program, command Command, current string) (candidates []string) {
	equals := strings.Index(current, "=")
	label := current[1:equals]
	partial := current[equals+1:]
	option, found := command_scope_option(program, command, label)
	if !found {
		return nil
	}
	members, is_enum := option_enum_members(*option)
	if !is_enum {
		return nil
	}
	out := []string{}
	for _, member := range members {
		if strings.HasPrefix(member, partial) {
			out = append(out, "-"+label+"="+member)
		}
	}
	return out
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
		block, block_err := completion_block(name, shell)
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

// Builds one shell's completion registration for a single command name.
func completion_block(name string, shell string) (block string, err error) {
	switch shell {
	case "bash":
		return fmt.Sprintf(""+
			"_%[1]s_complete() {\n"+
			"    local IFS=$'\\n'\n"+
			"    COMPREPLY=($(%[1]s __complete \"${COMP_WORDS[@]}\"))\n"+
			"}\n"+
			"complete -o default -F _%[1]s_complete %[1]s\n", name), nil
	case "zsh":
		return fmt.Sprintf(""+
			"#compdef %[1]s\n"+
			"_%[1]s_complete() {\n"+
			"    local -a completions\n"+
			"    completions=(\"${(@f)$(%[1]s __complete \"${words[@]}\")}\")\n"+
			"    compadd -- $completions\n"+
			"}\n"+
			"compdef _%[1]s_complete %[1]s\n", name), nil
	case "fish":
		return fmt.Sprintf(""+
			"function _%[1]s_complete\n"+
			"    %[1]s __complete (commandline -opc) (commandline -ct)\n"+
			"end\n"+
			"complete -c %[1]s -f -a '(_%[1]s_complete)'\n", name), nil
	}
	return "", fmt.Errorf("unsupported shell %q; use bash, zsh, or fish", shell)
}
