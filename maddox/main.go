// Package main is the maddox command: it benchmarks two or more commands on
// macOS/Apple Silicon and writes a JSON or table report comparing them. It parses
// the command line with shared/cli, wires the cgo measurer and the operating-system
// clock, and hands them to the pure maddox library, which does the sampling,
// statistics, and report.
package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"local/james-orcales/maddox/internal"
	"local/james-orcales/shared/cli"
	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/io"
	"local/james-orcales/shared/time"
)

// EXIT_USAGE marks a malformed command line, kept distinct from a benchmark
// failure so a caller can tell "you invoked me wrong" from "a command failed".
const EXIT_USAGE = 2

// DURATION_SECONDS_DEFAULT is the default per-command time budget: sample each
// command for up to five seconds, or until the run cap, whichever comes first.
const DURATION_SECONDS_DEFAULT = 30

// RUNS_DEFAULT is the default per-command run cap: thirty samples, the
// central-limit-theorem rule-of-thumb where the t-test is well-behaved, so a fast
// command stops there rather than burning the whole time budget.
const RUNS_DEFAULT = 1000

// WARMUP_DEFAULT is the default number of discarded warmup runs: three, to prime
// caches and the filesystem so the measured runs reflect warm steady state.
const WARMUP_DEFAULT = 5

func main() {
	program := main_program()
	if cli.Handle_Completion(program, os.Args, os.Stdout) {
		os.Exit(0)
	}
	command, parse_err := cli.Program_Parse(&program, os.Args)
	if errors.Is(parse_err, cli.Help_Requested) {
		cli.Print_Requested_Help(os.Stdout, program, command)
		os.Exit(0)
	}
	if parse_err != nil {
		fmt.Fprintln(os.Stderr, parse_err)
		cli.Print_Help(os.Stderr, program)
		os.Exit(EXIT_USAGE)
	}
	command_strings := cli.Get_Option(command.Arguments, "command").Value.([]string)
	if len(command_strings) == 0 {
		cli.Print_Help(os.Stderr, program)
		os.Exit(EXIT_USAGE)
	}
	commands, build_err := commands_from_strings(command_strings)
	if build_err != nil {
		fmt.Fprintln(os.Stderr, "maddox: "+build_err.Error())
		os.Exit(EXIT_USAGE)
	}

	format := maddox.OUTPUT_FORMAT_TABLE
	if cli.Get_Option(command.Flags, "json").Value.(bool) {
		format = maddox.OUTPUT_FORMAT_JSON
	}
	duration_seconds := cli.Get_Option(command.Flags, "duration").Value.(int)
	runs := cli.Get_Option(command.Flags, "runs").Value.(int)
	warmup := cli.Get_Option(command.Flags, "warmup").Value.(int)
	allow_failures := cli.Get_Option(command.Flags, "allow-failures").Value.(bool)
	color_mode := cli.Get_Option(command.Flags, "color").Value.(string)
	progress_mode := cli.Get_Option(command.Flags, "progress").Value.(string)

	input := &maddox.Main_Input{
		Commands:       commands,
		Sampler:        system_sampler(),
		Duration_Max:   time.Duration(duration_seconds) * time.SECOND,
		Runs_Max:       runs,
		Warmup_Count:   warmup,
		Allow_Failures: allow_failures,
		Format:         format,
		Color:          resolve_stream(Stream_Mode(color_mode), os.Stdout),
		Progress:       resolve_stream(Stream_Mode(progress_mode), os.Stderr),
		Output:         os.Stdout,
		Stderr:         os.Stderr,
		Machine:        acquire_machine_specs(),
	}
	os.Exit(int(maddox.Main(*input)))
}

// Main_program declares the maddox command line: a variadic list of commands to
// benchmark, plus the sampling and output flags.
func main_program() (program cli.Program) {
	return cli.New_Single(cli.New_Single_Input{
		Label:       "maddox",
		Description: "benchmark and compare commands on macOS",
		Arguments: []cli.Option{
			cli.New_Variadic[string](cli.New_Variadic_Input{
				Label:       "command",
				Description: "a command to benchmark (whitespace-split, no shell)",
			}),
		},
		Flags: []cli.Option{
			cli.New_Flag[int](cli.New_Flag_Input[int]{
				Label:       "duration",
				Value:       DURATION_SECONDS_DEFAULT,
				Description: "per-command time budget in seconds",
			}),
			cli.New_Flag[int](cli.New_Flag_Input[int]{
				Label:       "runs",
				Value:       RUNS_DEFAULT,
				Description: "stop after this many runs (0 = only -duration)",
			}),
			cli.New_Flag[int](cli.New_Flag_Input[int]{
				Label:       "warmup",
				Value:       WARMUP_DEFAULT,
				Description: "runs to discard before sampling",
			}),
			cli.New_Flag[bool](cli.New_Flag_Input[bool]{
				Label:       "allow-failures",
				Value:       false,
				Description: "keep benchmarking a command that exits non-zero",
			}),
			cli.New_Flag[bool](cli.New_Flag_Input[bool]{
				Label:       "json",
				Value:       false,
				Description: "emit JSON instead of the table",
			}),
			cli.New_Enum_Flag(cli.New_Enum_Flag_Input[string]{
				Label:       "color",
				Enum:        []string{"auto", "never", "always"},
				Value:       "auto",
				Description: "colorize the table",
			}),
			cli.New_Enum_Flag(cli.New_Enum_Flag_Input[string]{
				Label:       "progress",
				Enum:        []string{"auto", "never", "always"},
				Value:       "auto",
				Description: "live progress on stderr",
			}),
		},
	})
}

// BOUND_MIN and BOUND_MAX bound the command-tier defined types' lengths; min is vacuous
// and max is eager, so the bounds hold for any real command line without observation.
const BOUND_MIN = -1
const BOUND_MAX = 1 << 16

// Cli_Commands is the raw command strings from the command line, each one a command to
// benchmark before it is parsed into words.
type Cli_Commands []string

// Cli_Commands_Invariants bounds the command-string count.
func Cli_Commands_Invariants(commands Cli_Commands, namespace invariant.Namespace) {
	invariant.Dot_Product(namespace).Range_Int(len(commands), BOUND_MIN, BOUND_MAX).Ensure()
}

// Stream_Mode is a color/progress toggle from the command line: never, always, or auto.
type Stream_Mode string

// Stream_Mode_Invariants bounds the mode word's length.
func Stream_Mode_Invariants(mode Stream_Mode, namespace invariant.Namespace) {
	invariant.Dot_Product(namespace).Range_Int(len(mode), BOUND_MIN, BOUND_MAX).Ensure()
}

// Commands_from_strings turns each command string into an io.Process_Request,
// splitting it on whitespace and partitioning leading KEY=VALUE assignments off as the
// process environment. It errors on a string with no executable.
func commands_from_strings(command_strings Cli_Commands) (commands maddox.Commands, err error) {
	defer func() { maddox.Commands_Invariants(commands, "commands_from_strings.commands") }()
	Cli_Commands_Invariants(command_strings, "commands_from_strings.command_strings")
	commands = make(maddox.Commands, 0, len(command_strings))
	for _, text := range command_strings {
		fields := strings.Fields(text)
		// A leading KEY=VALUE is an environment assignment: its '=' sits past index 0 so
		// the key is non-empty (IndexByte returns -1 with no '=', also <= 0). The first
		// field failing this is the executable, so stop partitioning there.
		var environment []string
		cut := 0
		for _, field := range fields {
			if strings.IndexByte(field, '=') <= 0 {
				break
			}
			environment = append(environment, field)
			cut++
		}
		remainder := fields[cut:]
		if len(remainder) == 0 {
			return nil, errors.New("empty command: " + strconv.Quote(text))
		}
		if remainder[0] == "" {
			return nil, errors.New("empty command: " + strconv.Quote(text))
		}
		command := io.Process_Request{Environment: environment, Path: remainder[0]}
		if len(remainder) > 1 {
			command.Arguments = remainder[1:]
		}
		commands = append(commands, command)
	}
	return commands, nil
}

// Resolve_stream turns an auto/never/always mode into a decision: always or never as
// named, auto when the stream is a terminal.
func resolve_stream(mode Stream_Mode, file *os.File) (enabled bool) {
	defer func() { invariant.Boolean_Invariants(enabled, "resolve_stream.enabled") }()
	Stream_Mode_Invariants(mode, "resolve_stream.mode")
	if mode == "always" {
		return true
	}
	if mode == "never" {
		return false
	}
	return is_terminal(file)
}

// Is_terminal reports whether the file is a character device, so color and progress
// are suppressed when the stream is piped or redirected to a file.
func is_terminal(file *os.File) (terminal bool) {
	defer func() { invariant.Boolean_Invariants(terminal, "is_terminal.terminal") }()
	stat, stat_err := file.Stat()
	if stat_err != nil {
		return false
	}
	return stat.Mode()&os.ModeCharDevice != 0
}
