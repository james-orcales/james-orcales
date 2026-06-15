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

	"github.com/james-orcales/james-orcales/maddox/internal"
	"github.com/james-orcales/james-orcales/shared/cli"
	"github.com/james-orcales/james-orcales/shared/sh"
	"github.com/james-orcales/james-orcales/shared/time"
	time_default "github.com/james-orcales/james-orcales/shared/time/default"
)

// Exit_usage marks a malformed command line, kept distinct from a benchmark
// failure so a caller can tell "you invoked me wrong" from "a command failed".
const exit_usage = 2

// Duration_seconds_default is the default per-command time budget: sample each
// command for up to five seconds, or until the run cap, whichever comes first.
const duration_seconds_default = 30

// Runs_default is the default per-command run cap: thirty samples, the
// central-limit-theorem rule-of-thumb where the t-test is well-behaved, so a fast
// command stops there rather than burning the whole time budget.
const runs_default = 1000

// Warmup_default is the default number of discarded warmup runs: three, to prime
// caches and the filesystem so the measured runs reflect warm steady state.
const warmup_default = 5

func main() {
	program := main_program()
	command, parse_err := cli.Program_Parse(&program, os.Args)
	if parse_err != nil {
		fmt.Fprintln(os.Stderr, parse_err)
		cli.Print_Help(os.Stderr, program)
		os.Exit(exit_usage)
	}
	command_strings := cli.Get_Option(command.Arguments, "command").Value.([]string)
	if len(command_strings) == 0 {
		cli.Print_Help(os.Stderr, program)
		os.Exit(exit_usage)
	}
	commands, build_err := commands_from_strings(command_strings)
	if build_err != nil {
		fmt.Fprintln(os.Stderr, "maddox: "+build_err.Error())
		os.Exit(exit_usage)
	}

	format := maddox.Output_Format_Table
	if cli.Get_Option(command.Flags, "json").Value.(bool) {
		format = maddox.Output_Format_Json
	}
	duration_seconds := cli.Get_Option(command.Flags, "duration").Value.(int)
	runs := cli.Get_Option(command.Flags, "runs").Value.(int)
	warmup := cli.Get_Option(command.Flags, "warmup").Value.(int)
	allow_failures := cli.Get_Option(command.Flags, "allow-failures").Value.(bool)
	color_mode := cli.Get_Option(command.Flags, "color").Value.(string)
	progress_mode := cli.Get_Option(command.Flags, "progress").Value.(string)

	input := &maddox.Main_Input{
		Commands:       commands,
		Clock:          time_default.New_Operating_System_Clock(),
		Sampler:        system_sampler(),
		Duration_Max:   time.Duration(duration_seconds) * time.Second,
		Runs_Max:       runs,
		Warmup_Count:   warmup,
		Allow_Failures: allow_failures,
		Format:         format,
		Color:          resolve_stream(color_mode, os.Stdout),
		Progress:       resolve_stream(progress_mode, os.Stderr),
		Output:         os.Stdout,
		Stderr:         os.Stderr,
		Machine:        acquire_machine_specs(),
	}
	os.Exit(maddox.Main(input))
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
				Value:       duration_seconds_default,
				Description: "per-command time budget in seconds",
			}),
			cli.New_Flag[int](cli.New_Flag_Input[int]{
				Label:       "runs",
				Value:       runs_default,
				Description: "stop after this many runs (0 = only -duration)",
			}),
			cli.New_Flag[int](cli.New_Flag_Input[int]{
				Label:       "warmup",
				Value:       warmup_default,
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
			cli.New_Flag[string](cli.New_Flag_Input[string]{
				Label:       "color",
				Value:       "auto",
				Description: "auto, never, or always",
			}),
			cli.New_Flag[string](cli.New_Flag_Input[string]{
				Label:       "progress",
				Value:       "auto",
				Description: "live progress on stderr: auto, never, or always",
			}),
		},
	})
}

// Commands_from_strings turns each command string into an sh.Command, splitting it
// on whitespace and partitioning leading KEY=VALUE assignments off via the sh
// library's parser. It errors on a string with no executable.
func commands_from_strings(command_strings []string) (commands []sh.Command, err error) {
	commands = make([]sh.Command, 0, len(command_strings))
	for _, text := range command_strings {
		command, ok := sh.Spawn_Raw_Plan(strings.Fields(text))
		if !ok {
			return nil, errors.New("empty command: " + strconv.Quote(text))
		}
		commands = append(commands, command)
	}
	return commands, nil
}

// Resolve_stream turns an auto/never/always mode into a decision: always or never as
// named, auto when the stream is a terminal.
func resolve_stream(mode string, file *os.File) (enabled bool) {
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
	stat, stat_err := file.Stat()
	if stat_err != nil {
		return false
	}
	return stat.Mode()&os.ModeCharDevice != 0
}
