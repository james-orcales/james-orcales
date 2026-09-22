// Package maddox compares the performance of commands, the way poop does on Linux. It is the
// pure tier of the maddox binary: every run is a spawn submitted to the injected IO loop, and the
// statistics, the command-to-command comparison, and the report are computed here. Package main
// binds the real loop, the clock, and the host snapshot, then pumps.
package maddox

import (
	"fmt"
	"unsafe"

	"local/james-orcales/shared/std/cli"
	"local/james-orcales/shared/std/math/bits"
	"local/james-orcales/shared/std/math/fixedpoint"
	"local/james-orcales/shared/std/math/scalar"
	"local/james-orcales/shared/std/sim/aver/default"
	"local/james-orcales/shared/std/sim/nbio"
	"local/james-orcales/shared/std/sim/time"
	"local/james-orcales/shared/std/strings"
)

// RUN_DEADLINE bounds one run. A benchmark states no deadline of its own, thus this sits far past
// any command worth measuring while the monotonic clock plus it still fits a signed word.
const RUN_DEADLINE = 1 << 62

// STAGE_START is a parsed start Rearm has yet to act on; it is the zero stage, thus a runner the
// root builds around Main's start begins there.
const STAGE_START Stage = 0

// STAGE_READY is a run, a command's end, or the report that Rearm has yet to act on.
const STAGE_READY Stage = 1

// STAGE_SUBMITTED is a run in flight on the loop.
const STAGE_SUBMITTED Stage = 2

// STAGE_RETIRED is a run whose result the callback recorded and Rearm has yet to judge.
const STAGE_RETIRED Stage = 3

// STAGE_WRITE is a write Rearm has yet to submit.
const STAGE_WRITE Stage = 4

// STAGE_WRITE_FLIGHT is a write in flight on the loop.
const STAGE_WRITE_FLIGHT Stage = 5

// STAGE_STOPPED is a finished run whose Status is final.
const STAGE_STOPPED Stage = 6

// HELP_BYTES_MAX is the cli output store: help, a usage error, or completion candidates.
const HELP_BYTES_MAX = 1 << 12

// Main parses the command line into the benchmark setup, or into the help, usage diagnostic, or
// completion candidates a run writes instead. It touches no loop: the root copies the start into
// its runner, then repeats Rearm and pumps.
func Main(input Main_Input) (start Main_Start) {
	defer func() { Main_Start_Invariants(start, "Main.start") }()
	Main_Input_Invariants(input, "Main.input")
	// The program is injected as its builder, never passed or returned as a value: its shape is
	// fixed, thus no run could witness the bounds its invariants state.
	return main_parse(input, func() (program cli.Program) {
		return cli.New(cli.New_Input{
			Mode:        cli.PROGRAM_MODE_SINGLE,
			Label:       "maddox",
			Description: "benchmark and compare commands",
			Arguments: []cli.Option{cli.New_Option(cli.New_Option_Input{
				Label:       "command",
				Type:        cli.OPTION_TYPE_STRINGS,
				Description: "a command to benchmark (whitespace-split, no shell)",
			})},
			Flags: []cli.Option{cli.New_Option(cli.New_Option_Input{
				Label: "duration", Type: cli.OPTION_TYPE_INTEGER, Is_Flag: true,
				Integer:     DURATION_SECONDS_DEFAULT,
				Description: "per-command time budget in seconds",
			}), cli.New_Option(cli.New_Option_Input{
				Label: "runs", Type: cli.OPTION_TYPE_INTEGER, Is_Flag: true,
				Integer:     RUNS_DEFAULT,
				Description: "stop after this many runs (0 = only -duration)",
			}), cli.New_Option(cli.New_Option_Input{
				Label: "warmup", Type: cli.OPTION_TYPE_INTEGER, Is_Flag: true,
				Integer:     WARMUP_DEFAULT,
				Description: "runs to discard before sampling",
			}), cli.New_Option(cli.New_Option_Input{
				Label: "allow-failures", Type: cli.OPTION_TYPE_BOOLEAN,
				Is_Flag:     true,
				Description: "keep benchmarking a command that exits non-zero",
			}), cli.New_Option(cli.New_Option_Input{
				Label: "json", Type: cli.OPTION_TYPE_BOOLEAN, Is_Flag: true,
				Description: "emit JSON instead of the table",
			}), cli.New_Option(cli.New_Option_Input{
				Label: "color", String_Enum: []string{"auto", "never", "always"},
				String: "auto", Is_Flag: true, Description: "colorize the table",
			}), cli.New_Option(cli.New_Option_Input{
				Label: "progress", String_Enum: []string{"auto", "never", "always"},
				String: "auto", Is_Flag: true,
				Description: "live progress on stderr",
			})},
		})
	})
}

// Program_Build builds the command-line program.
type Program_Build func() (program cli.Program)

// Parses the command line against the built program. Help, completion, and usage render here,
// where the program is in hand; everything else reads plain values out of the parse.
func main_parse(input Main_Input, build Program_Build) (start Main_Start) {
	defer func() { Main_Start_Invariants(start, "main_parse.start") }()
	Main_Input_Invariants(input, "main_parse.input")
	program := build()
	output := cli.Output{Builder: cli.Output_Builder{Storage: make([]byte, HELP_BYTES_MAX)}}
	help := cli.Output_Reference_Of(&output)
	arguments := cli.Process_Arguments(input.Arguments)
	candidates := make(cli.Candidates, cli.CANDIDATE_COUNT_MAXIMUM)
	if cli.Handle_Completion(program, arguments, help, candidates) {
		text := cli.Output_Bytes(help)
		return Main_Start{Write: nbio.Stream_Memory{Memory: text, Cursor: int64(len(text))}}
	}
	parser := cli.Parser{}
	command := program.Selection.Single_Commands.Command
	token_count := len(arguments) - 1
	cli.Program_Parse(program, &parser, cli.Program_Parse_Input{
		Arguments:         arguments,
		Command_Arguments: make(cli.Command_Argument_Storage, len(command.Arguments)),
		Command_Flags:     make(cli.Command_Flag_Storage, len(command.Flags)),
		Global_Flags:      make(cli.Global_Flag_Storage, 0),
		Filled: make(cli.Filled_Options,
			len(command.Arguments)+len(command.Flags)),
		Positionals:     make(cli.Positionals, token_count),
		Slice_Named:     make(cli.Named_Variadic_Tokens, token_count),
		String_Values:   make(cli.String_Values, token_count),
		Integer_Values:  make(cli.Integer_Values, token_count),
		Failure_Storage: make(cli.Parse_Failure_Storage, cli.FAILURE_SIZE_MAXIMUM),
		Deprecation_Warnings: make(
			cli.Deprecation_Warnings, cli.DEPRECATION_WARNING_COUNT_MAXIMUM),
	})
	result, complete := cli.Parser_Done(&parser)
	aver.Always(complete, "A program without secrets parses synchronously.")
	failure := string(cli.Parse_Error_Bytes(result.Error))
	commands := Cli_Commands{}
	if failure == "" {
		commands = Cli_Commands(cli.Option_Strings(
			cli.Resolved_Options(result.Command.Arguments), "command"))
	}
	if len(commands) > 0 {
		flags := cli.Resolved_Options(result.Command.Flags)
		resolved := Main_Flags{
			Json:     Json_Flag(cli.Option_Boolean(flags, "json")),
			Duration: Duration_Limit(cli.Option_Integer(flags, "duration")),
			Runs:     Run_Limit(cli.Option_Integer(flags, "runs")),
			Warmup:   Warmup_Limit(cli.Option_Integer(flags, "warmup")),
			Allow_Failures: Failure_Allowance(
				cli.Option_Boolean(flags, "allow-failures")),
			Color:    Color_Mode(cli.Option_String(flags, "color")),
			Progress: Progress_Mode(cli.Option_String(flags, "progress")),
		}
		resolved.Output_Is_Terminal = input.Output_Is_Terminal
		resolved.Error_Output_Is_Terminal = input.Error_Output_Is_Terminal
		return main_start(input.Environment, commands, resolved)
	}
	start = Main_Start{Status: MAIN_STATUS_USAGE}
	switch failure {
	case cli.HELP_REQUESTED:
		cli.Print_Requested_Help(help, program, result.Command.Label)
		start = Main_Start{}
	case "":
		cli.Print_Help(help, program)
	default:
		cli.Output_Write_Text(help, cli.Value_Text("maddox: "+failure+"\n\n"))
		cli.Print_Help(help, program)
	}
	text := cli.Output_Bytes(help)
	start.Write = nbio.Stream_Memory{Memory: text, Cursor: int64(len(text))}
	return start
}

// Main_Flags is every flag the parse resolved, with the terminal facts the automatic modes read.
type Main_Flags struct {
	// Json selects the JSON report.
	Json Json_Flag
	// Duration is the per-command time budget in seconds.
	Duration Duration_Limit
	// Runs is the per-command run cap.
	Runs Run_Limit
	// Warmup is how many runs are discarded before sampling.
	Warmup Warmup_Limit
	// Allow_Failures keeps benchmarking a command that exits non-zero.
	Allow_Failures Failure_Allowance
	// Color is the table color mode.
	Color Color_Mode
	// Progress is the progress diagnostic mode.
	Progress Progress_Mode
	// Output_Is_Terminal reports whether the report stream is a terminal.
	Output_Is_Terminal Output_Terminal_Status
	// Error_Output_Is_Terminal reports whether the diagnostic stream is a terminal.
	Error_Output_Is_Terminal Error_Output_Terminal_Status
}

// Main_Flags_Invariants states every resolved flag.
func Main_Flags_Invariants(value Main_Flags, namespace aver.Namespace) {
	Json_Flag_Invariants(value.Json, namespace)
	Duration_Limit_Invariants(value.Duration, namespace)
	Run_Limit_Invariants(value.Runs, namespace)
	Warmup_Limit_Invariants(value.Warmup, namespace)
	Failure_Allowance_Invariants(value.Allow_Failures, namespace)
	Color_Mode_Invariants(value.Color, namespace)
	Progress_Mode_Invariants(value.Progress, namespace)
	Output_Terminal_Status_Invariants(value.Output_Is_Terminal, namespace)
	Error_Output_Terminal_Status_Invariants(value.Error_Output_Is_Terminal, namespace)
}

// Json_Flag reports the JSON report selected.
type Json_Flag bool

// Json_Flag_Invariants witnesses both report forms.
func Json_Flag_Invariants(value Json_Flag, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The JSON report is selected.").
		Ensure()
}

// Color_Mode is the table color setting.
type Color_Mode Stream_Mode

// Color_Mode_Invariants bounds the mode vocabulary by its three lengths.
func Color_Mode_Invariants(value Color_Mode, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Int(
			len(value), STREAM_MODE_BYTES_MIN,
			STREAM_MODE_BYTES_MIDDLE, STREAM_MODE_BYTES_MAX).
		Ensure()
}

// Progress_Mode is the progress diagnostic setting.
type Progress_Mode Stream_Mode

// Progress_Mode_Invariants bounds the mode vocabulary by its three lengths.
func Progress_Mode_Invariants(value Progress_Mode, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Int(
			len(value), STREAM_MODE_BYTES_MIN,
			STREAM_MODE_BYTES_MIDDLE, STREAM_MODE_BYTES_MAX).
		Ensure()
}

// Builds the commands and resolves the flags into the benchmark setup.
func main_start(
	environment Inherited_Environment, command_strings Cli_Commands, flags Main_Flags,
) (start Main_Start) {
	defer func() { Main_Start_Invariants(start, "main_start.start") }()
	Inherited_Environment_Invariants(environment, "main_start.environment")
	Cli_Commands_Invariants(command_strings, "main_start.command_strings")
	Main_Flags_Invariants(flags, "main_start.flags")
	commands, failure, empty := commands_from_strings(command_strings, environment)
	message := ""
	switch failure {
	case COMMAND_FAILURE_COUNT:
		message = fmt.Sprintf("command count exceeds %d", COMMAND_SET_MAX)
	case COMMAND_FAILURE_WORDS:
		message = fmt.Sprintf("command word count exceeds %d", COMMAND_WORDS_MAX)
	case COMMAND_FAILURE_BYTES:
		message = fmt.Sprintf("command word exceeds %d bytes", COMMAND_WORD_BYTES_MAX)
	case COMMAND_FAILURE_EMPTY:
		message = fmt.Sprintf("empty command: %q", string(empty))
	}
	if failure != COMMAND_FAILURE_NONE {
		text := []byte("maddox: " + message + "\n")
		return Main_Start{
			Write:  nbio.Stream_Memory{Memory: text, Cursor: int64(len(text))},
			Status: MAIN_STATUS_USAGE,
		}
	}
	configuration := Configuration{
		Commands:       Command_Set(commands),
		Inherited:      Inherited_Count(len(environment)),
		Duration_Max:   flags.Duration,
		Runs_Max:       flags.Runs,
		Warmup_Count:   flags.Warmup,
		Allow_Failures: flags.Allow_Failures,
		Format:         OUTPUT_FORMAT_TABLE,
		Color: Color(resolve_stream(Stream_Mode(flags.Color),
			Terminal_Status(flags.Output_Is_Terminal))),
		Progress: Progress(resolve_stream(Stream_Mode(flags.Progress),
			Terminal_Status(flags.Error_Output_Is_Terminal))),
	}
	if flags.Json {
		configuration.Format = OUTPUT_FORMAT_JSON
	}
	start.Configuration = &configuration
	return start
}

// Main_Input carries the command line and the host facts the setup reads.
type Main_Input struct {
	// Arguments is the operating-system command line, including the program name.
	Arguments Arguments
	// Environment is the process environment every command inherits.
	Environment Inherited_Environment
	// Output_Is_Terminal reports whether the report stream is a terminal.
	Output_Is_Terminal Output_Terminal_Status
	// Error_Output_Is_Terminal reports whether the diagnostic stream is a terminal.
	Error_Output_Is_Terminal Error_Output_Terminal_Status
}

// Main_Input_Invariants states the command line and host facts.
func Main_Input_Invariants(value Main_Input, namespace aver.Namespace) {
	Arguments_Invariants(value.Arguments, namespace)
	Inherited_Environment_Invariants(value.Environment, namespace)
	Output_Terminal_Status_Invariants(value.Output_Is_Terminal, namespace)
	Error_Output_Terminal_Status_Invariants(value.Error_Output_Is_Terminal, namespace)
}

// MAIN_STATUS_SUCCESS is a setup that benchmarks, or help that was asked for.
const MAIN_STATUS_SUCCESS Main_Status = Main_Status(EXIT_SUCCESS)

// MAIN_STATUS_USAGE is a malformed command line.
const MAIN_STATUS_USAGE Main_Status = Main_Status(EXIT_USAGE)

// Main_Status is the exit code parsing alone can settle.
type Main_Status int

// Main_Status_Invariants holds the status to success or usage.
func Main_Status_Invariants(value Main_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(int(value), int(MAIN_STATUS_SUCCESS), int(MAIN_STATUS_USAGE)).
		Ensure()
}

// WRITE_TARGET_OUTPUT is the report stream.
const WRITE_TARGET_OUTPUT Write_Target = 0

// WRITE_TARGET_ERROR is the diagnostic stream.
const WRITE_TARGET_ERROR Write_Target = 1

// Write_Target names the stream a write goes to.
type Write_Target uint8

// Write_Target_Invariants holds the target to the two streams.
func Write_Target_Invariants(value Write_Target, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(WRITE_TARGET_OUTPUT), uint8(WRITE_TARGET_ERROR)).
		Ensure()
}

// Main_Start is what parsing settles: the setup to benchmark, or the text to write and stop.
type Main_Start struct {
	// Configuration is the benchmark setup; nil when the run only writes.
	Configuration Configuration_Handle
	// Write is the help, diagnostic, or candidates to write; empty when the run benchmarks. It
	// goes to the report stream on success and to the diagnostic stream on a usage failure.
	Write nbio.Stream_Memory
	// Status is the exit code once Write lands, or success for a run that benchmarks.
	Status Main_Status
}

// Main_Start_Invariants states the settled start.
func Main_Start_Invariants(value Main_Start, namespace aver.Namespace) {
	Configuration_Handle_Invariants(value.Configuration, namespace)
	Main_Status_Invariants(value.Status, namespace)
}

// Configuration is the benchmark setup resolved from the command line.
type Configuration struct {
	// Commands are the commands to benchmark; the first is the reference.
	Commands Command_Set
	// Inherited is how many environment entries every command inherits before its own.
	Inherited Inherited_Count
	// Duration_Max is the per-command time budget.
	Duration_Max Duration_Limit
	// Runs_Max is the per-command run cap.
	Runs_Max Run_Limit
	// Warmup_Count is how many runs are discarded before sampling.
	Warmup_Count Warmup_Limit
	// Allow_Failures keeps benchmarking a command that exits non-zero.
	Allow_Failures Failure_Allowance
	// Format selects the report rendering.
	Format Output_Format
	// Color enables ANSI color in the table.
	Color Color
	// Progress enables live progress diagnostics.
	Progress Progress
}

// Configuration_Invariants states the resolved benchmark setup.
func Configuration_Invariants(value Configuration, namespace aver.Namespace) {
	Command_Set_Invariants(value.Commands, namespace)
	Inherited_Count_Invariants(value.Inherited, namespace)
	Duration_Limit_Invariants(value.Duration_Max, namespace)
	Run_Limit_Invariants(value.Runs_Max, namespace)
	Warmup_Limit_Invariants(value.Warmup_Count, namespace)
	Failure_Allowance_Invariants(value.Allow_Failures, namespace)
	Output_Format_Invariants(value.Format, namespace)
	Color_Invariants(value.Color, namespace)
	Progress_Invariants(value.Progress, namespace)
}

// Configuration_Handle borrows a benchmark setup.
type Configuration_Handle *Configuration

// Configuration_Handle_Invariants composes the borrowed setup.
func Configuration_Handle_Invariants(value Configuration_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Configuration_Invariants(*value, namespace)
}

// Stage is where the run stands; callbacks only move it forward, Rearm acts on it.
type Stage int

// Stage_Invariants holds a stage to the declared ones.
func Stage_Invariants(value Stage, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), int(STAGE_START), int(STAGE_STOPPED)).
		Ensure()
}

// Final_Write reports a write that ends the run once it lands; any other leaves the runner ready.
type Final_Write bool

// Final_Write_Invariants witnesses both write endings.
func Final_Write_Invariants(value Final_Write, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The write ends the run.").
		Ensure()
}

// Progress_Flight reports a progress line in flight.
type Progress_Flight bool

// Progress_Flight_Invariants witnesses both write states.
func Progress_Flight_Invariants(value Progress_Flight, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A progress line is in flight.").
		Ensure()
}

// Report_Due reports every command sampled.
type Report_Due bool

// Report_Due_Invariants witnesses both sampling states.
func Report_Due_Invariants(value Report_Due, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The report is due.").
		Ensure()
}

// Measurements_Handle borrows the reference command's stored measurements.
type Measurements_Handle *Measurements

// Measurements_Handle_Invariants composes stored measurements.
func Measurements_Handle_Invariants(value Measurements_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Measurements_Invariants(*value, namespace)
}

// Sample_Store holds the kept samples of the command in flight, growing one run at a time.
type Sample_Store []Sample

// Sample_Store_Invariants bounds the store to the kept-run ceiling.
func Sample_Store_Invariants(value Sample_Store, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COLLECTION_MIN, SAMPLES_MAX).
		Ensure()
}

// Sampler is the progress of the command in flight: which command, and how far it sampled.
type Sampler struct {
	// Command_Index is the command being sampled.
	Command_Index Position
	// Warmups counts discarded runs of the command being sampled.
	Warmups Tally
	// Warmup_Elapsed sums the discarded runs' wall time.
	Warmup_Elapsed Warmup_Elapsed
	// Stopwatch_Start is the first kept run's start, from which the budget runs.
	Stopwatch_Start Stopwatch_Start
	// Elapsed is the budget stopwatch.
	Elapsed Stopwatch_Elapsed
	// Samples holds the kept samples of the command being sampled.
	Samples Sample_Store
}

// Sampler_Invariants states the benchmark's progress.
func Sampler_Invariants(value Sampler, namespace aver.Namespace) {
	Position_Invariants(value.Command_Index, namespace)
	Tally_Invariants(value.Warmups, namespace)
	Warmup_Elapsed_Invariants(value.Warmup_Elapsed, namespace)
	Stopwatch_Start_Invariants(value.Stopwatch_Start, namespace)
	Stopwatch_Elapsed_Invariants(value.Elapsed, namespace)
	Sample_Store_Invariants(value.Samples, namespace)
}

// Open_Sampler is a sampler a run is judged against, before the run joins it: its sample store
// still has room.
type Open_Sampler Sampler

// Open_Sampler_Invariants states the progress with a free sample slot.
func Open_Sampler_Invariants(value Open_Sampler, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Command_Index), POSITION_MIN, POSITION_MAX).
		Range_Int(int(value.Warmups), TALLY_MIN, TALLY_MAX).
		Range_Int64(int64(value.Warmup_Elapsed), ELAPSED_MINIMUM,
			int64(time.MONOTONIC_MOMENT_MAXIMUM)).
		Range_Int64(int64(value.Stopwatch_Start), int64(time.MONOTONIC_MOMENT_MINIMUM),
			int64(time.MONOTONIC_MOMENT_MAXIMUM)).
		Range_Int64(int64(value.Elapsed), ELAPSED_MINIMUM,
			int64(time.MONOTONIC_MOMENT_MAXIMUM)).
		Range_Int(len(value.Samples), COLLECTION_MIN, OPEN_SAMPLES_MAX).
		Ensure()
}

// OPEN_SAMPLES_MAX is one short of the sample cap.
const OPEN_SAMPLES_MAX = SAMPLES_MAX - 1

// Open_Sampler_Handle borrows a sampler a run is judged against.
type Open_Sampler_Handle *Open_Sampler

// Open_Sampler_Handle_Invariants composes the borrowed sampler.
func Open_Sampler_Handle_Invariants(value Open_Sampler_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Open_Sampler_Invariants(*value, namespace)
}

// Sampler_Handle borrows the runner's benchmark progress.
type Sampler_Handle *Sampler

// Sampler_Handle_Invariants composes the borrowed progress.
func Sampler_Handle_Invariants(value Sampler_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Sampler_Invariants(*value, namespace)
}

// Warmup_Elapsed sums the discarded runs' wall time.
type Warmup_Elapsed time.Duration

// Warmup_Elapsed_Invariants bounds the warmup time.
func Warmup_Elapsed_Invariants(value Warmup_Elapsed, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), ELAPSED_MINIMUM, int64(time.MONOTONIC_MOMENT_MAXIMUM)).
		Ensure()
}

// Stopwatch_Elapsed is the budget stopwatch.
type Stopwatch_Elapsed time.Duration

// Stopwatch_Elapsed_Invariants bounds the stopwatch.
func Stopwatch_Elapsed_Invariants(value Stopwatch_Elapsed, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), ELAPSED_MINIMUM, int64(time.MONOTONIC_MOMENT_MAXIMUM)).
		Ensure()
}

// Stopwatch_Start is the monotonic moment the first kept run of a command started.
type Stopwatch_Start time.Monotonic_Moment

// Stopwatch_Start_Invariants bounds the moment to the clock's range.
func Stopwatch_Start_Invariants(value Stopwatch_Start, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), int64(time.MONOTONIC_MOMENT_MINIMUM),
			int64(time.MONOTONIC_MOMENT_MAXIMUM)).
		Ensure()
}

// Runner is one whole run's caller-owned state. Callbacks record into it; the root calls Rearm to
// turn what they recorded into the next submission, then pumps.
type Runner struct {
	// IO is the injected loop every run and write goes through.
	IO nbio.IO
	// Clock stamps each run's completion for the time budget.
	Clock time.Clock
	// Output receives the report.
	Output nbio.Stream
	// Error_Output receives progress and diagnostics.
	Error_Output nbio.Stream
	// Discard swallows every command's standard output.
	Discard nbio.Stream_Discard
	// Capture receives a run's standard error.
	Capture nbio.Stream_Memory
	// Machine is the host snapshot the report carries.
	Machine Machine_Specs
	// Start is what parsing settled: the setup, or the text to write and stop.
	Start Main_Start
	// Sampler is the command in flight's progress.
	Sampler Sampler
	// Record is what the finished commands leave behind.
	Record Record
	// Stage is where the run stands.
	Stage Stage
	// Status is the exit code, final once Stage is STAGE_STOPPED.
	Status Exit_Code
	// Write is the write Rearm submits next.
	Write nbio.Stream_Memory
	// Write_Target is the stream Write goes to.
	Write_Target Write_Target
	// Final_Write reports a write that ends the run once it lands.
	Final_Write Final_Write
	// Progress_Flight reports a progress line in flight.
	Progress_Flight Progress_Flight
	// Progress holds the progress line in flight.
	Progress nbio.Stream_Memory
	// Result is the recorded outcome of the last run.
	Result nbio.Process_Result
	// Result_Error is the recorded failure of the last run.
	Result_Error error
	// Process_Completion is the spawn submission. A callback recovers the runner from it by its
	// offset, thus no runner holds a pointer back to itself.
	Process_Completion nbio.Completion
	// Write_Completion is the report and diagnostic write submission.
	Write_Completion nbio.Completion
	// Progress_Completion is the progress-line write submission.
	Progress_Completion nbio.Completion
}

// Runner_Invariants composes the runner's state.
func Runner_Invariants(value Runner, namespace aver.Namespace) {
	nbio.IO_Invariants(value.IO, namespace)
	time.Clock_Invariants(&value.Clock, namespace)
	Machine_Specs_Invariants(value.Machine, namespace)
	Main_Start_Invariants(value.Start, namespace)
	Sampler_Invariants(value.Sampler, namespace)
	Record_Invariants(value.Record, namespace)
	Stage_Invariants(value.Stage, namespace)
	Exit_Code_Invariants(value.Status, namespace)
	Write_Target_Invariants(value.Write_Target, namespace)
	Final_Write_Invariants(value.Final_Write, namespace)
	Progress_Flight_Invariants(value.Progress_Flight, namespace)
}

// Runner_Handle borrows the caller-owned runner.
type Runner_Handle *Runner

// Runner_Handle_Invariants composes the borrowed runner.
func Runner_Handle_Invariants(value Runner_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Runner_Invariants(*value, namespace)
}

// Rearm turns what a callback recorded into the next submission and reports whether it did
// anything, so the root repeats it until the run needs the loop again.
func Rearm(runner Runner_Handle) (progressed Progression) {
	defer func() { Progression_Invariants(progressed, "Rearm.progressed") }()
	Runner_Handle_Invariants(runner, "Rearm.runner")
	step := Ready_Step{}
	switch runner.Stage {
	case STAGE_START:
		// Allocated once here, so no run allocates a capture or a sample.
		runner.Capture = nbio.Stream_Memory{Memory: make([]byte, CAPTURE_BYTES_MAX)}
		runner.Sampler.Samples = make(Sample_Store, 0, SAMPLES_MAX)
		runner.Status = Exit_Code(runner.Start.Status)
		runner.Stage = STAGE_READY
		step = start_step(runner.Start)
	case STAGE_READY:
		step = ready_step(*runner.Start.Configuration, &runner.Sampler, &runner.Record,
			runner.Machine)
		if step.Kind == STEP_KIND_WRITE {
			break
		}
		step.Request.Stdout = nbio.Discard_To_Stream(&runner.Discard)
		step.Request.Error_Output_Storage = runner.Capture.Memory
		runner.Stage = STAGE_SUBMITTED
		nbio.IO_Spawn(runner.IO, &runner.Process_Completion, step.Request, RUN_DEADLINE,
			func(retired *nbio.Completion, result nbio.Process_Result, err error) {
				runner_process_retired(nbio.Completion_Handle(retired), result, err)
			})
	case STAGE_RETIRED:
		runner.Stage = STAGE_READY
		moment := Completion_Moment(time.Clock_Now_Monotonic(runner.Clock))
		result := run_result(runner.Result, Spawn_Failure(runner.Result_Error != nil &&
			runner.Result_Error != nbio.Process_Output_Too_Large), moment)
		judged := judge_step(
			*runner.Start.Configuration, (*Open_Sampler)(&runner.Sampler), result)
		if judged.Kind == STEP_KIND_FAIL {
			runner.Status = EXIT_FAILURE
			step = Ready_Step{Kind: STEP_KIND_WRITE, Write: judged.Write,
				Target: WRITE_TARGET_ERROR, Final: true}
		}
		// A line skipped while an earlier one is in flight is superseded by the next.
		if judged.Progress.Cursor > 0 {
			if !runner.Progress_Flight {
				runner.Progress = judged.Progress
				runner.Progress_Flight = true
				nbio.Write(runner.Error_Output, &runner.Progress_Completion,
					runner.Progress.Memory[:runner.Progress.Cursor],
					runner_progress_retired)
			}
		}
	case STAGE_WRITE:
		// A write waits for a progress line in flight, thus the lines stay in order.
		if runner.Progress_Flight {
			return false
		}
		stream := runner.Output
		if runner.Write_Target == WRITE_TARGET_ERROR {
			stream = runner.Error_Output
		}
		runner.Stage = STAGE_WRITE_FLIGHT
		nbio.Write(stream, &runner.Write_Completion,
			runner.Write.Memory[:runner.Write.Cursor], runner_write_retired)
	default:
		return false
	}
	if step.Kind == STEP_KIND_WRITE {
		runner.Write = step.Write
		runner.Write_Target = step.Target
		runner.Final_Write = step.Final
		runner.Stage = STAGE_WRITE
	}
	return true
}

// Decides what a parsed start submits: its help, diagnostic, or candidates, as the run's only
// write, or nothing when it benchmarks.
func start_step(start Main_Start) (step Ready_Step) {
	defer func() { Ready_Step_Invariants(step, "start_step.step") }()
	Main_Start_Invariants(start, "start_step.start")
	if start.Write.Cursor == 0 {
		return step
	}
	step = Ready_Step{Kind: STEP_KIND_WRITE, Write: start.Write, Final: true}
	if start.Status != MAIN_STATUS_SUCCESS {
		step.Target = WRITE_TARGET_ERROR
	}
	return step
}

// Work_Queued reports whether a callback recorded work only Rearm can act on.
func Work_Queued(runner Runner_Handle) (queued Queue_State) {
	defer func() { Queue_State_Invariants(queued, "Work_Queued.queued") }()
	Runner_Handle_Invariants(runner, "Work_Queued.runner")
	switch runner.Stage {
	case STAGE_START, STAGE_READY, STAGE_RETIRED:
		return true
	case STAGE_WRITE:
		return Queue_State(!runner.Progress_Flight)
	}
	return false
}

// Stopped reports whether the run is over and Status is final.
func Stopped(runner Runner_Handle) (stopped Stop_State) {
	defer func() { Stop_State_Invariants(stopped, "Stopped.stopped") }()
	Runner_Handle_Invariants(runner, "Stopped.runner")
	return runner.Stage == STAGE_STOPPED
}

// STEP_KIND_SPAWN is a run to submit.
const STEP_KIND_SPAWN Step_Kind = 0

// STEP_KIND_WRITE is a write to submit.
const STEP_KIND_WRITE Step_Kind = 1

// Step_Kind is what Rearm submits next.
type Step_Kind uint8

// Step_Kind_Invariants holds the kind to a run or a write.
func Step_Kind_Invariants(value Step_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(STEP_KIND_SPAWN), uint8(STEP_KIND_WRITE)).
		Ensure()
}

// Ready_Step is what a ready runner submits: the next run, a progress clear, or the report.
type Ready_Step struct {
	// Kind is the submission.
	Kind Step_Kind
	// Request is the run to spawn.
	Request nbio.Process_Request
	// Write is the bytes to write.
	Write nbio.Stream_Memory
	// Target is the stream Write goes to.
	Target Write_Target
	// Final reports a write that ends the run once it lands.
	Final Final_Write
}

// Ready_Step_Invariants states the submission.
func Ready_Step_Invariants(value Ready_Step, namespace aver.Namespace) {
	Step_Kind_Invariants(value.Kind, namespace)
	Write_Target_Invariants(value.Target, namespace)
	Final_Write_Invariants(value.Final, namespace)
}

// Decides what a ready runner submits: the report once due, the next warmup or kept run, or,
// once sampling stops, the command's benchmark appended to the report and then the next command.
func ready_step(
	configuration Configuration, sampler Sampler_Handle, record Record_Handle,
	machine Machine_Specs,
) (step Ready_Step) {
	defer func() { Ready_Step_Invariants(step, "ready_step.step") }()
	Configuration_Invariants(configuration, "ready_step.configuration")
	Sampler_Handle_Invariants(sampler, "ready_step.sampler")
	Record_Handle_Invariants(record, "ready_step.record")
	Machine_Specs_Invariants(machine, "ready_step.machine")
	if !record.Report_Due {
		if sampling_warms(configuration, sampler.Warmups) {
			step.Request = commands_at(configuration.Commands, sampler.Command_Index)
			return step
		}
		if sampling_should_continue(&Sampling_Should_Continue_Input{
			Duration_Max: configuration.Duration_Max,
			Runs_Max:     configuration.Runs_Max,
			Count:        Tally(len(sampler.Samples)),
			Elapsed:      Elapsed(sampler.Elapsed),
		}) {
			step.Request = commands_at(configuration.Commands, sampler.Command_Index)
			return step
		}
		finished := command_finish(configuration, sampler.Command_Index,
			Distribution(sampler.Samples), record.Reference, record.Report, machine)
		sampler.Samples = sampler.Samples[:0]
		sampler.Warmups = 0
		sampler.Warmup_Elapsed = 0
		sampler.Stopwatch_Start = 0
		sampler.Elapsed = 0
		if !finished.Report_Due {
			sampler.Command_Index++
		}
		if record.Reference == nil {
			record.Reference = &finished.Measurements
		}
		record.Report = finished.Report
		record.Report_Due = finished.Report_Due
		step.Kind = STEP_KIND_WRITE
		step.Target = WRITE_TARGET_ERROR
		if configuration.Progress {
			clear := []byte(PROGRESS_CLEAR)
			step.Write = nbio.Stream_Memory{Memory: clear, Cursor: int64(len(clear))}
			return step
		}
		if !record.Report_Due {
			step.Kind = STEP_KIND_SPAWN
			step.Request = commands_at(configuration.Commands, sampler.Command_Index)
			return step
		}
	}
	step.Kind = STEP_KIND_WRITE
	step.Target = WRITE_TARGET_OUTPUT
	step.Final = true
	step.Write = record.Report
	if configuration.Format == OUTPUT_FORMAT_JSON {
		step.Write = Json_Tail_Render(record.Report)
	}
	return step
}

// Record is what the finished commands leave behind: the reference to compare against and the
// report so far.
type Record struct {
	// Reference is the first command's measurements, nil until the first command finishes.
	Reference Measurements_Handle
	// Report holds the report rendered so far, one benchmark appended as each command finishes.
	Report nbio.Stream_Memory
	// Report_Due reports every command sampled, thus the next advance writes the report.
	Report_Due Report_Due
}

// Record_Invariants states what the finished commands left.
func Record_Invariants(value Record, namespace aver.Namespace) {
	Measurements_Handle_Invariants(value.Reference, namespace)
	Report_Due_Invariants(value.Report_Due, namespace)
}

// Record_Handle borrows the runner's record.
type Record_Handle *Record

// Record_Handle_Invariants composes the borrowed record.
func Record_Handle_Invariants(value Record_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Record_Invariants(*value, namespace)
}

// Reports whether the command being sampled still owes a discarded warmup run. Warming up past
// the kept-sample cap is pointless and would carry the warmup counter past a tally's range, so
// the cap bounds both.
func sampling_warms(configuration Configuration, warmups Tally) (warms Continuation) {
	defer func() { Continuation_Invariants(warms, "sampling_warms.warms") }()
	Configuration_Invariants(configuration, "sampling_warms.configuration")
	Tally_Invariants(warmups, "sampling_warms.warmups")
	if Warmup_Limit(warmups) >= configuration.Warmup_Count {
		return false
	}
	return warmups < SAMPLES_MAX
}

// Finished_Command is a sampled command reduced: its measurements, the report with its benchmark
// appended, and whether it was the last.
type Finished_Command struct {
	// Measurements is the command's distribution summaries.
	Measurements Measurements
	// Report is the report with the command's benchmark appended.
	Report nbio.Stream_Memory
	// Report_Due reports the command was the last, thus the report is due.
	Report_Due Report_Due
}

// Finished_Command_Invariants states the reduced command.
func Finished_Command_Invariants(value Finished_Command, namespace aver.Namespace) {
	Measurements_Invariants(value.Measurements, namespace)
	Report_Due_Invariants(value.Report_Due, namespace)
}

// Reduces the finished command's samples to its benchmark and appends it to report.
func command_finish(
	configuration Configuration, index Position, samples Distribution,
	reference Measurements_Handle, report nbio.Stream_Memory, machine Machine_Specs,
) (finished Finished_Command) {
	defer func() { Finished_Command_Invariants(finished, "command_finish.finished") }()
	Configuration_Invariants(configuration, "command_finish.configuration")
	Position_Invariants(index, "command_finish.index")
	Distribution_Invariants(samples, "command_finish.samples")
	Measurements_Handle_Invariants(reference, "command_finish.reference")
	Machine_Specs_Invariants(machine, "command_finish.machine")
	finished.Measurements = measurements_compute(samples)
	benchmark := Benchmark{
		Command: command_words(commands_at(configuration.Commands, index),
			configuration.Inherited),
		Runs:         Kept(len(samples)),
		Elapsed:      samples_elapsed(samples),
		Measurements: finished.Measurements,
	}
	if reference == nil {
		report = Table_Head_Render(machine)
		if configuration.Format == OUTPUT_FORMAT_JSON {
			report = Json_Head_Render(machine)
		}
	} else {
		benchmark.Deltas = deltas_compute(
			*reference, Candidate_Measurements(finished.Measurements))
	}
	if configuration.Format == OUTPUT_FORMAT_JSON {
		finished.Report = Json_Benchmark_Render(report, index, benchmark)
	} else {
		finished.Report = Table_Benchmark_Render(
			report, index, benchmark, configuration.Color)
	}
	finished.Report_Due = int(index)+1 == int(configuration.Commands.Count)
	return finished
}

// Spawn_Failure reports a run the loop could not start at all.
type Spawn_Failure bool

// Spawn_Failure_Invariants witnesses both spawn outcomes.
func Spawn_Failure_Invariants(value Spawn_Failure, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The run could not start.").
		Ensure()
}

// SPAWN_FAILURE_EXIT is the exit code reported when the child could not be spawned at all —
// distinct from any code the child itself could return.
const SPAWN_FAILURE_EXIT = 127

// Converts a recorded spawn outcome into a Run_Result stamped at moment. A run that could not
// start reports the exit code a shell gives a command it cannot execute.
func run_result(
	outcome nbio.Process_Result, failure Spawn_Failure, moment Completion_Moment,
) (result Run_Result) {
	defer func() { Run_Result_Invariants(result, "run_result.result") }()
	Spawn_Failure_Invariants(failure, "run_result.failure")
	Completion_Moment_Invariants(moment, "run_result.moment")
	result.Completed_At = moment
	result.Spawn_Failure = failure
	if failure {
		result.Exit = SPAWN_FAILURE_EXIT
		return result
	}
	usage := outcome.Usage
	result.Sample = Sample{
		Wall:             Metric(usage.Wall),
		RSS_Bytes_Max:    Resident_Bytes(usage.RSS_Bytes_Max),
		CPU_Cycles:       Cycle_Count(usage.Cycle_Count),
		Instructions:     Instruction_Count(usage.Instruction_Count),
		Cache_References: Cache_Reference_Count(usage.Cache_Reference_Count),
		Cache_Misses:     Cache_Miss_Count(usage.Cache_Miss_Count),
		Branch_Misses:    Branch_Miss_Count(usage.Branch_Miss_Count),
		CPU_User:         User_Time(usage.CPU_User),
		CPU_System:       System_Time(usage.CPU_System),
	}
	result.Exit = Exit_Status(outcome.Exit)
	if result.Exit != 0 {
		result.Stderr = Captured_Output(outcome.Error_Output)
	}
	return result
}

// STEP_KIND_CONTINUE is a judged run that leaves the runner ready.
const STEP_KIND_CONTINUE Judge_Kind = 0

// STEP_KIND_FAIL is a judged run whose failure stops the benchmark.
const STEP_KIND_FAIL Judge_Kind = 1

// Judge_Kind is what a judged run leads to.
type Judge_Kind uint8

// Judge_Kind_Invariants holds the kind to continuing or failing.
func Judge_Kind_Invariants(value Judge_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(STEP_KIND_CONTINUE), uint8(STEP_KIND_FAIL)).
		Ensure()
}

// Judge_Step is what a judged run submits: a failure diagnostic, or an optional progress line.
type Judge_Step struct {
	// Kind is whether the benchmark continues.
	Kind Judge_Kind
	// Write is the failure diagnostic.
	Write nbio.Stream_Memory
	// Progress is the progress line to write, empty when progress is off.
	Progress nbio.Stream_Memory
}

// Judge_Step_Invariants states the judged outcome.
func Judge_Step_Invariants(value Judge_Step, namespace aver.Namespace) {
	Judge_Kind_Invariants(value.Kind, namespace)
}

// Judges one run: a failure aborts unless allowed, a warmup is discarded, and a kept run joins
// the samples.
func judge_step(
	configuration Configuration, sampler Open_Sampler_Handle, result Run_Result,
) (step Judge_Step) {
	defer func() { Judge_Step_Invariants(step, "judge_step.step") }()
	Configuration_Invariants(configuration, "judge_step.configuration")
	Open_Sampler_Handle_Invariants(sampler, "judge_step.sampler")
	Run_Result_Invariants(result, "judge_step.result")
	command := commands_at(configuration.Commands, sampler.Command_Index)
	if result.Exit != 0 {
		if !configuration.Allow_Failures {
			step.Kind = STEP_KIND_FAIL
			step.Write = failure_message(&Failure_Message_Input{
				Progress: configuration.Progress, Index: sampler.Command_Index,
				Path: Command_Word(command.Path), Exit: Failure_Status(result.Exit),
				Stderr: result.Stderr, Spawn_Failure: result.Spawn_Failure,
			})
			return step
		}
	}
	progress := Render_Progress_Input{
		Words: command_words(command, configuration.Inherited),
		Total: Progress_Total(configuration.Runs_Max),
	}
	if sampling_warms(configuration, sampler.Warmups) {
		sampler.Warmups++
		sampler.Warmup_Elapsed += Warmup_Elapsed(result.Sample.Wall)
		progress.Elapsed = Elapsed(sampler.Warmup_Elapsed)
		progress.Phase = "warmup"
		progress.Count = Census(sampler.Warmups)
		progress.Total = Progress_Total(configuration.Warmup_Count)
	} else {
		if len(sampler.Samples) == 0 {
			// Anchor at the first run's start so between-run gaps count in the budget.
			sampler.Stopwatch_Start = Stopwatch_Start(int64(result.Completed_At) -
				int64(result.Sample.Wall))
		}
		sampler.Samples = append(sampler.Samples, result.Sample)
		sampler.Elapsed = Stopwatch_Elapsed(int64(result.Completed_At) -
			int64(sampler.Stopwatch_Start))
		progress.Elapsed = Elapsed(sampler.Elapsed)
		progress.Count = Census(len(sampler.Samples))
	}
	if configuration.Progress {
		step.Progress = render_progress(&progress)
	}
	return step
}

// Failure_Message_Input is a benchmarked command's failure as its diagnostic reports it.
type Failure_Message_Input struct {
	// Progress reports a progress line to clear first.
	Progress Progress
	// Index is the failing command's position.
	Index Position
	// Path is the failing command's executable.
	Path Command_Word
	// Exit is the failing run's exit code.
	Exit Failure_Status
	// Stderr is the failing run's captured stderr.
	Stderr Captured_Output
	// Spawn_Failure reports a run the loop could not start at all.
	Spawn_Failure Spawn_Failure
}

// Failure_Message_Input_Invariants states the failure.
func Failure_Message_Input_Invariants(value Failure_Message_Input, namespace aver.Namespace) {
	Progress_Invariants(value.Progress, namespace)
	Position_Invariants(value.Index, namespace)
	Command_Word_Invariants(value.Path, namespace)
	Failure_Status_Invariants(value.Exit, namespace)
	Captured_Output_Invariants(value.Stderr, namespace)
	Spawn_Failure_Invariants(value.Spawn_Failure, namespace)
}

// Failure_Message_Input_Handle borrows a failure to report.
type Failure_Message_Input_Handle *Failure_Message_Input

// Failure_Message_Input_Handle_Invariants composes the borrowed failure.
func Failure_Message_Input_Handle_Invariants(
	value Failure_Message_Input_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Failure_Message_Input_Invariants(*value, namespace)
}

// Renders a benchmarked command's non-zero exit for the diagnostic sink: the progress clear, a
// one-line header naming the command's position and code, then the command's own stderr, or the
// spawn diagnostic when the command never started.
func failure_message(input Failure_Message_Input_Handle) (message nbio.Stream_Memory) {
	Failure_Message_Input_Handle_Invariants(input, "failure_message.input")
	text := []byte{}
	if input.Progress {
		text = append(text, PROGRESS_CLEAR...)
	}
	text = fmt.Appendf(text, "maddox: benchmark %d exited %d\n", int(input.Index)+1,
		int(input.Exit))
	text = append(text, input.Stderr...)
	if input.Spawn_Failure {
		text = append(text, "maddox: cannot spawn "+string(input.Path)+"\n"...)
	}
	return nbio.Stream_Memory{Memory: text, Cursor: int64(len(text))}
}

// Records only; Rearm judges, so no continuation runs inside the loop's own stack.
func runner_process_retired(
	completion nbio.Completion_Handle, result nbio.Process_Result, err error,
) {
	runner := (*Runner)(unsafe.Add(unsafe.Pointer(completion),
		-int(unsafe.Offsetof(Runner{}.Process_Completion))))
	runner.Result = result
	runner.Result_Error = err
	runner.Stage = STAGE_RETIRED
}

// A report or diagnostic that fails to land fails the run.
func runner_write_retired(completion nbio.Completion_Handle) {
	runner := (*Runner)(unsafe.Add(unsafe.Pointer(completion),
		-int(unsafe.Offsetof(Runner{}.Write_Completion))))
	if completion.Error != nil {
		runner.Status = EXIT_FAILURE
		runner.Stage = STAGE_STOPPED
		return
	}
	runner.Stage = STAGE_READY
	if runner.Final_Write {
		runner.Stage = STAGE_STOPPED
	}
}

// A progress line that fails to land changes nothing: the next one replaces it.
func runner_progress_retired(completion nbio.Completion_Handle) {
	runner := (*Runner)(unsafe.Add(unsafe.Pointer(completion),
		-int(unsafe.Offsetof(Runner{}.Progress_Completion))))
	runner.Progress_Flight = false
}

// Commands_from_strings separates each command without invoking a shell. Leading NAME=VALUE words
// are the command's environment assignments, appended after the inherited environment.
func commands_from_strings(
	command_strings Cli_Commands, inherited Inherited_Environment,
) (commands Commands, failure Command_Failure, empty Command_Text) {
	defer func() {
		Commands_Invariants(commands, "commands_from_strings.commands")
		Command_Failure_Invariants(failure, "commands_from_strings.failure")
		Command_Text_Invariants(empty, "commands_from_strings.empty")
	}()
	Inherited_Environment_Invariants(inherited, "commands_from_strings.inherited")
	Cli_Commands_Invariants(command_strings, "commands_from_strings.command_strings")
	if len(command_strings) > COMMAND_SET_MAX {
		return Commands{}, COMMAND_FAILURE_COUNT, ""
	}
	for _, text := range command_strings {
		fields := text_fields(Command_Text(text))
		if len(fields) > COMMAND_WORDS_MAX {
			return Commands{}, COMMAND_FAILURE_WORDS, ""
		}
		for _, field := range fields {
			if len(field) > COMMAND_WORD_BYTES_MAX {
				return Commands{}, COMMAND_FAILURE_BYTES, ""
			}
		}
		assignment_count := 0
		for _, field := range fields {
			if !text_assignment(Command_Word(field)) {
				break
			}
			assignment_count++
		}
		if assignment_count == len(fields) {
			return Commands{}, COMMAND_FAILURE_EMPTY, Command_Text(text)
		}
		environment := make([]string, 0, len(inherited)+assignment_count)
		environment = append(environment, inherited...)
		environment = append(environment, fields[:assignment_count]...)
		command := nbio.Process_Request{
			Environment: environment,
			Path:        fields[assignment_count],
			Arguments:   fields[assignment_count+1:],
		}
		commands = Commands(commands_append(Partial_Commands(commands), command))
	}
	return commands, COMMAND_FAILURE_NONE, ""
}

// Splits text at runs of ASCII whitespace, as strings.Fields does for the text a shell passes.
func text_fields(text Command_Text) (fields Command_Fields) {
	defer func() { Command_Fields_Invariants(fields, "text_fields.fields") }()
	Command_Text_Invariants(text, "text_fields.text")
	start_index := -1
	for index := 0; index <= len(text); index++ {
		space := index == len(text)
		if !space {
			switch text[index] {
			case ' ', '\t', '\n', '\v', '\f', '\r':
				space = true
			}
		}
		if !space {
			if start_index < 0 {
				start_index = index
			}
			continue
		}
		if start_index >= 0 {
			fields = append(fields, string(text[start_index:index]))
			start_index = -1
		}
	}
	return fields
}

// Reports whether a word is a NAME=VALUE assignment: an equals sign past the first byte.
func text_assignment(word Command_Word) (assignment Assignment) {
	defer func() { Assignment_Invariants(assignment, "text_assignment.assignment") }()
	Command_Word_Invariants(word, "text_assignment.word")
	for index := 1; index < len(word); index++ {
		if word[index] == '=' {
			return true
		}
	}
	return false
}

// Resolve_stream applies an explicit mode or the injected terminal state.
func resolve_stream(mode Stream_Mode, terminal Terminal_Status) (enabled Stream_State) {
	defer func() { Stream_State_Invariants(enabled, "resolve_stream.enabled") }()
	Stream_Mode_Invariants(mode, "resolve_stream.mode")
	Terminal_Status_Invariants(terminal, "resolve_stream.terminal")
	if mode == "always" {
		return true
	}
	if mode == "never" {
		return false
	}
	return Stream_State(uint8(terminal) == TERMINAL_STATUS_TERMINAL)
}

// Inherited_Environment is the process environment every command inherits.
type Inherited_Environment []string

// Inherited_Environment_Invariants bounds the value.
func Inherited_Environment_Invariants(value Inherited_Environment, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COLLECTION_MIN, ENVIRONMENT_COUNT_MAX).
		Ensure()
}

// Command_Failure is why the command-line commands do not parse, if they do not.
type Command_Failure uint8

// Command_Failure_Invariants holds a failure to the declared reasons.
func Command_Failure_Invariants(value Command_Failure, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(COMMAND_FAILURE_NONE), uint8(COMMAND_FAILURE_EMPTY),
		).
		Ensure()
}

// COMMAND_FAILURE_NONE is commands that parse.
const COMMAND_FAILURE_NONE Command_Failure = 0

// COMMAND_FAILURE_COUNT is more commands than a benchmark compares.
const COMMAND_FAILURE_COUNT Command_Failure = 1

// COMMAND_FAILURE_WORDS is a command with more words than a request carries.
const COMMAND_FAILURE_WORDS Command_Failure = 2

// COMMAND_FAILURE_BYTES is a command word longer than a request carries.
const COMMAND_FAILURE_BYTES Command_Failure = 3

// COMMAND_FAILURE_EMPTY is a command of nothing but environment assignments.
const COMMAND_FAILURE_EMPTY Command_Failure = 4

// Command_Text is one positional command as the user typed it.
type Command_Text string

// Command_Text_Invariants bounds the value.
func Command_Text_Invariants(value Command_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COLLECTION_MIN, COMMAND_TEXT_BYTES_MAX).
		Ensure()
}

// Command_Fields is one command's whitespace-separated words.
type Command_Fields []string

// Command_Fields_Invariants bounds the value.
func Command_Fields_Invariants(value Command_Fields, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COLLECTION_MIN, COMMAND_FIELDS_MAX).
		Ensure()
}

// Assignment reports whether a word assigns an environment variable.
type Assignment bool

// Assignment_Invariants bounds the value.
func Assignment_Invariants(value Assignment, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The word is an assignment.").
		Ensure()
}

// Progression reports whether Rearm acted.
type Progression bool

// Progression_Invariants bounds the value.
func Progression_Invariants(value Progression, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Rearm acted.").
		Ensure()
}

// Queue_State reports whether work waits on Rearm.
type Queue_State bool

// Queue_State_Invariants bounds the value.
func Queue_State_Invariants(value Queue_State, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Work is queued.").
		Ensure()
}

// Stop_State reports whether the run stopped.
type Stop_State bool

// Stop_State_Invariants bounds the value.
func Stop_State_Invariants(value Stop_State, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The run stopped.").
		Ensure()
}

// Render_Progress_Input_Handle borrows caller-owned render progress input state.
type Render_Progress_Input_Handle *Render_Progress_Input

// Render_Progress_Input_Handle_Invariants bounds the value.
func Render_Progress_Input_Handle_Invariants(value Render_Progress_Input_Handle,
	namespace aver.Namespace) {
	if value == nil {
		return
	}
	Render_Progress_Input_Invariants(*value, namespace)
}

// Sampling_Should_Continue_Input_Handle borrows caller-owned sampling should continue input state.
type Sampling_Should_Continue_Input_Handle *Sampling_Should_Continue_Input

// Sampling_Should_Continue_Input_Handle_Invariants bounds the value.
func Sampling_Should_Continue_Input_Handle_Invariants(value Sampling_Should_Continue_Input_Handle,
	namespace aver.Namespace) {
	if value == nil {
		return
	}
	Sampling_Should_Continue_Input_Invariants(*value, namespace)
}

// Outlier_Count_Input_Handle borrows caller-owned outlier count input state.
type Outlier_Count_Input_Handle *Outlier_Count_Input

// Outlier_Count_Input_Handle_Invariants bounds the value.
func Outlier_Count_Input_Handle_Invariants(value Outlier_Count_Input_Handle,
	namespace aver.Namespace) {
	if value == nil {
		return
	}
	Outlier_Count_Input_Invariants(*value, namespace)
}

// Significant_Input_Handle borrows caller-owned significant input state.
type Significant_Input_Handle *Significant_Input

// Significant_Input_Handle_Invariants bounds the value.
func Significant_Input_Handle_Invariants(value Significant_Input_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Significant_Input_Invariants(*value, namespace)
}

// Render_Cells_Input_Handle borrows caller-owned render cells input state.
type Render_Cells_Input_Handle *Render_Cells_Input

// Render_Cells_Input_Handle_Invariants bounds the value.
func Render_Cells_Input_Handle_Invariants(value Render_Cells_Input_Handle,
	namespace aver.Namespace) {
	if value == nil {
		return
	}
	Render_Cells_Input_Invariants(*value, namespace)
}

// Root_Of_Quotient_Input_Handle borrows caller-owned root of quotient input state.
type Root_Of_Quotient_Input_Handle *Root_Of_Quotient_Input

// Root_Of_Quotient_Input_Handle_Invariants bounds the value.
func Root_Of_Quotient_Input_Handle_Invariants(value Root_Of_Quotient_Input_Handle,
	namespace aver.Namespace) {
	if value == nil {
		return
	}
	Root_Of_Quotient_Input_Invariants(*value, namespace)
}

// Metric_Line_Input_Handle borrows caller-owned metric line input state.
type Metric_Line_Input_Handle *Metric_Line_Input

// Metric_Line_Input_Handle_Invariants bounds the value.
func Metric_Line_Input_Handle_Invariants(value Metric_Line_Input_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Metric_Line_Input_Invariants(*value, namespace)
}

// Root_Index is the heap position a sift starts from: a parent, thus in the first half.
type Root_Index int

// Root_Index_Invariants bounds a root to the parents of the largest heap.
func Root_Index_Invariants(value Root_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COLLECTION_MIN, ROOT_INDEX_MAX).
		Ensure()
}

// ROOT_INDEX_MAX is the last parent of the largest heap.
const ROOT_INDEX_MAX = POINTS_MAX/2 - 1

// End_Index is the length of the heap a sift works within.
type End_Index int

// End_Index_Invariants bounds a heap to at least one point and at most the largest run.
func End_Index_Invariants(value End_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), END_INDEX_MIN, POINTS_MAX).
		Ensure()
}

// END_INDEX_MIN is a heap of the one point the last swap leaves.
const END_INDEX_MIN = 1

// Fraction_Digits is how many decimals a rendered number keeps.
type Fraction_Digits int

// Fraction_Digits_Invariants bounds the value.
func Fraction_Digits_Invariants(value Fraction_Digits, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Int(int(value), FRACTION_DIGITS_NONE, FRACTION_DIGITS_ONE,
			FRACTION_DIGITS_TWO).
		Ensure()
}

// Fixed_Text is one rendered fixed-point number.
type Fixed_Text string

// Fixed_Text_Invariants bounds the value.
func Fixed_Text_Invariants(value Fixed_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), FIXED_TEXT_BYTES_MIN, FIXED_TEXT_BYTES_MAX).
		Ensure()
}

// FRACTION_DIGITS_NONE renders a whole number.
const FRACTION_DIGITS_NONE = 0

// FRACTION_DIGITS_ONE renders tenths.
const FRACTION_DIGITS_ONE = 1

// FRACTION_DIGITS_TWO renders hundredths.
const FRACTION_DIGITS_TWO = 2

// COMMAND_TEXT_BYTES_MAX is the longest positional value the command line parser keeps.
const COMMAND_TEXT_BYTES_MAX = strings.TEXT_SIZE_MAXIMUM

// COMMAND_FIELDS_MAX is the most words text of that length splits into.
const COMMAND_FIELDS_MAX = (COMMAND_TEXT_BYTES_MAX + 1) / 2

// FIXED_TEXT_BYTES_MIN is one digit.
const FIXED_TEXT_BYTES_MIN = 1

// FIXED_TEXT_BYTES_MAX is the longest rendered value: a year of progress in tenths of a second.
const FIXED_TEXT_BYTES_MAX = len("31536000.0")

// Fixed_Value is a nonnegative fixed-point value a cell or a progress line renders: a scaled
// quantity, a percentage pinned to its column, a frequency, or the progress seconds.
type Fixed_Value fixedpoint.Number

// Fixed_Value_Invariants bounds a value to the longest span progress reports.
func Fixed_Value_Invariants(value Fixed_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), PERCENT_MINIMUM, FIXED_VALUE_MAX).
		Ensure()
}

// FIXED_VALUE_MAX is the progress seconds of a stopwatch run to the monotonic clock's end.
const FIXED_VALUE_MAX = PROGRESS_SECONDS_MAX * fixedpoint.SCALE

// PROGRESS_SECONDS_MAX is the monotonic clock's end in whole seconds.
const PROGRESS_SECONDS_MAX = int64(time.MONOTONIC_MOMENT_MAXIMUM) / int64(time.SECOND)

// EXIT_SUCCESS is the status Main returns when every command was benchmarked and
// the report was written.
const EXIT_SUCCESS Exit_Code = 0

// EXIT_FAILURE is the status Main returns when a command failed or the report could
// not be written.
const EXIT_FAILURE Exit_Code = 1

// EXIT_USAGE marks a malformed command line, distinct from a benchmark failure.
const EXIT_USAGE Exit_Code = 2

// DURATION_SECONDS_DEFAULT is the default per-command time budget.
const DURATION_SECONDS_DEFAULT = 30

// RUNS_DEFAULT caps fast commands before they consume the complete time budget.
const RUNS_DEFAULT = 1000

// WARMUP_DEFAULT primes caches and files before Maddox keeps measurements.
const WARMUP_DEFAULT = 5

// RUNS_MIN is the smallest number of samples a command is run, so a spent budget
// still leaves a quorum for the statistics — poop's min_samples.
const RUNS_MIN = 3

// SAMPLES_MAX caps the samples held for one command, bounding memory against a
// command fast enough to run unboundedly within the budget — poop's MAX_SAMPLES. Exported so
// the blackbox simulation sizes its scenarios against the real ceiling rather than a duplicate.
const SAMPLES_MAX = 10000

// LIMIT_MIN is the smallest limit accepted from a command line or a fuzz scenario.
const LIMIT_MIN int64 = -1 << 63

// LIMIT_MAX is the largest limit accepted from a command line or a fuzz scenario.
const LIMIT_MAX int64 = 1<<63 - 1

// Continuation records whether sampling takes one more sample.
type Continuation bool

// Continuation_Invariants witnesses both sampling decisions.
func Continuation_Invariants(value Continuation, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Sampling continues.").
		Ensure()
}

// Run_Limit is the maximum number of kept runs.
type Run_Limit int64

// Run_Limit_Invariants accepts the complete signed 64-bit input domain.
func Run_Limit_Invariants(value Run_Limit, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value),
			int64(LIMIT_MIN),
			int64(LIMIT_MAX),
		).
		Ensure()
}

// Warmup_Limit is the maximum number of discarded warmup runs.
type Warmup_Limit int64

// Warmup_Limit_Invariants accepts the complete signed 64-bit input domain.
func Warmup_Limit_Invariants(value Warmup_Limit, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value),
			int64(LIMIT_MIN),
			int64(LIMIT_MAX),
		).
		Ensure()
}

// Failure_Allowance records whether a failed command remains measurable.
type Failure_Allowance bool

// Failure_Allowance_Invariants witnesses both failure policies.
func Failure_Allowance_Invariants(value Failure_Allowance, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Command failures are allowed.").
		Ensure()
}

// Color records whether a rendered report contains ANSI color.
type Color bool

// Color_Invariants witnesses both color modes.
func Color_Invariants(value Color, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "ANSI color is enabled.").
		Ensure()
}

// Progress records whether sampling writes progress text.
type Progress bool

// Progress_Invariants witnesses both progress modes.
func Progress_Invariants(value Progress, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Progress output is enabled.").
		Ensure()
}

// Output_Format selects how Main renders the report.
type Output_Format uint8

// Output_Format_Invariants holds an Output_Format to its two declared rendering modes
// and witnesses each mode. The modes are an enumeration, not a span, so a member axis
// names the mode it claims.
func Output_Format_Invariants(format Output_Format, namespace aver.Namespace) {
	aver.Tree(format, namespace).
		Enum_Uint8(
			uint8(format), uint8(OUTPUT_FORMAT_TABLE), uint8(OUTPUT_FORMAT_JSON)).
		Ensure()
}

// OUTPUT_FORMAT_TABLE renders the human-readable comparison table; the default.
const OUTPUT_FORMAT_TABLE Output_Format = 0

// OUTPUT_FORMAT_JSON renders the machine-readable JSON document.
const OUTPUT_FORMAT_JSON Output_Format = 1

// Resident_Bytes is one sample's peak resident memory size.
type Resident_Bytes int64

// Resident_Bytes_Invariants bounds a resident-memory sample.
func Resident_Bytes_Invariants(value Resident_Bytes, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value),
			METRIC_MIN,
			METRIC_MAX,
		).
		Ensure()
}

// Cycle_Count is one sample's processor-cycle count.
type Cycle_Count int64

// Cycle_Count_Invariants bounds a processor-cycle sample.
func Cycle_Count_Invariants(value Cycle_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value),
			METRIC_MIN,
			METRIC_MAX,
		).
		Ensure()
}

// Instruction_Count is one sample's retired-instruction count.
type Instruction_Count int64

// Instruction_Count_Invariants bounds a retired-instruction sample.
func Instruction_Count_Invariants(value Instruction_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value),
			METRIC_MIN,
			METRIC_MAX,
		).
		Ensure()
}

// Cache_Reference_Count is one sample's cache-reference count.
type Cache_Reference_Count int64

// Cache_Reference_Count_Invariants bounds a cache-reference sample.
func Cache_Reference_Count_Invariants(
	value Cache_Reference_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value),
			METRIC_MIN,
			METRIC_MAX,
		).
		Ensure()
}

// Cache_Miss_Count is one sample's cache-miss count.
type Cache_Miss_Count int64

// Cache_Miss_Count_Invariants bounds a cache-miss sample.
func Cache_Miss_Count_Invariants(value Cache_Miss_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value),
			METRIC_MIN,
			METRIC_MAX,
		).
		Ensure()
}

// Branch_Miss_Count is one sample's branch-miss count.
type Branch_Miss_Count int64

// Branch_Miss_Count_Invariants bounds a branch-miss sample.
func Branch_Miss_Count_Invariants(value Branch_Miss_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value),
			METRIC_MIN,
			METRIC_MAX,
		).
		Ensure()
}

// User_Time is one sample's user-space CPU time.
type User_Time time.Duration

// User_Time_Invariants bounds a user-space CPU-time sample as a metric.
func User_Time_Invariants(value User_Time, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), METRIC_MIN, METRIC_MAX).
		Ensure()
}

// System_Time is one sample's kernel-space CPU time.
type System_Time time.Duration

// System_Time_Invariants bounds a kernel-space CPU-time sample as a metric.
func System_Time_Invariants(value System_Time, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), METRIC_MIN, METRIC_MAX).
		Ensure()
}

// Sample is one run's measurements.
type Sample struct {
	// Wall is the run's elapsed time, measured and reported by the sampler that ran it.
	Wall Metric
	// RSS_Bytes_Max is the run's peak physical memory footprint, in bytes.
	RSS_Bytes_Max Resident_Bytes
	// CPU_Cycles is the run's CPU cycle count from the hardware counters.
	CPU_Cycles Cycle_Count
	// Instructions is the run's retired-instruction count from the counters.
	Instructions Instruction_Count
	// Cache_References is the run's last-level cache reference count (Linux only).
	Cache_References Cache_Reference_Count
	// Cache_Misses is the run's last-level cache miss count (Linux only).
	Cache_Misses Cache_Miss_Count
	// Branch_Misses is the run's mispredicted-branch count (Linux only).
	Branch_Misses Branch_Miss_Count
	// CPU_User is the run's user-space CPU time.
	CPU_User User_Time
	// CPU_System is the run's kernel-space CPU time.
	CPU_System System_Time
}

// Sample_Invariants states a Sample's metric fields.
func Sample_Invariants(sample Sample, namespace aver.Namespace) {
	Metric_Invariants(sample.Wall, namespace)
	Resident_Bytes_Invariants(sample.RSS_Bytes_Max, namespace)
	Cycle_Count_Invariants(sample.CPU_Cycles, namespace)
	Instruction_Count_Invariants(sample.Instructions, namespace)
	Cache_Reference_Count_Invariants(sample.Cache_References, namespace)
	Cache_Miss_Count_Invariants(sample.Cache_Misses, namespace)
	Branch_Miss_Count_Invariants(sample.Branch_Misses, namespace)
	User_Time_Invariants(sample.CPU_User, namespace)
	System_Time_Invariants(sample.CPU_System, namespace)
}

// CAPTURE_BYTES_MIN is the empty capture: a command that wrote nothing to stderr.
const CAPTURE_BYTES_MIN = 0

// CAPTURE_BYTES_MAX is the sampler's stderr capture cap; a verbose failure fills it.
const CAPTURE_BYTES_MAX = 1 << 16

// Captured_Output is a failing command's stderr, read back into a bounded buffer — the
// bytes the command wrote, never trusted, never unbounded.
type Captured_Output []byte

// Captured_Output_Invariants bounds the captured stderr and witnesses its boundaries: the
// empty min, the full-buffer max, and the one- and two-byte shapes between.
func Captured_Output_Invariants(output Captured_Output, namespace aver.Namespace) {
	aver.Tree(output, namespace).
		Range_Int(len(output), CAPTURE_BYTES_MIN, CAPTURE_BYTES_MAX).
		Ensure()
}

// COLLECTION_MIN is the empty count: the floor a failure or an empty input leaves a
// sample or value collection at. QUORUM_MIN is the non-empty floor of a distribution;
// SAMPLES_MAX is the kept-run ceiling they share.
const COLLECTION_MIN = 0

// QUORUM_MIN is 3, the floor below which a distribution is too thin to trust.
const QUORUM_MIN = 3

// Samples is what one command's collection yields: empty when the first run fails before
// the minimum, or a full distribution. The one- and two-run counts between never occur,
// since the 3-run minimum is reached in one uninterrupted stretch or not at all.
type Samples []Sample

// Samples_Invariants states the collected count. The exclusions preserve the gap between an
// early failure and the uninterrupted three-run quorum while Range keeps both reachable edges
// demanded rather than reducing the contract to bound guards.
func Samples_Invariants(samples Samples, namespace aver.Namespace) {
	aver.Tree(samples, namespace).
		Range_Holed_Int(len(samples), COLLECTION_MIN, SAMPLES_MAX, 1, 2, 2, 2).
		Ensure()
}

// Distribution is a quorum of kept runs the statistics reduce — always at least the 3-run
// minimum, never the empty, one, or two counts the reducers cannot summarize.
type Distribution []Sample

// Distribution_Invariants bounds the run count: the quorum floor and the kept-run ceiling
// are witnessed, every short boundary claimed unreachable.
func Distribution_Invariants(samples Distribution, namespace aver.Namespace) {
	aver.Tree(samples, namespace).Range_Int(len(samples), QUORUM_MIN, SAMPLES_MAX).Ensure()
}

// Values is one metric's value pulled from every sample, the int64 the statistics work in.
type Values []int64

// Values_Invariants bounds the value count. The values are pulled from a kept distribution,
// floored at the 3-run quorum, so the quorum floor and the kept-run ceiling are the witnessed
// shapes; the empty, one, and two counts a quorum never holds fall below the range and are guarded.
func Values_Invariants(values Values, namespace aver.Namespace) {
	aver.Tree(values, namespace).Range_Int(len(values), QUORUM_MIN, SAMPLES_MAX).Ensure()
}

// Series is one metric pulled from a distribution — always the quorum's worth, never the
// empty, one, or two counts a distribution cannot hold.
type Series []int64

// Series_Invariants bounds the extracted count: it mirrors a distribution's quorum floor
// and kept-run ceiling, claiming every short boundary unreachable.
func Series_Invariants(values Series, namespace aver.Namespace) {
	aver.Tree(values, namespace).Range_Int(len(values), QUORUM_MIN, SAMPLES_MAX).Ensure()
}

// Deviations is the values the standard deviation reduces — a kept distribution's, so
// always at least the 3-run quorum, never the empty, one, or two counts below it.
type Deviations []int64

// Deviations_Invariants bounds the count. The deviations are a kept distribution's, floored at
// the 3-run quorum, so the quorum floor and the kept-run ceiling are witnessed and the shorter
// counts are guarded out.
func Deviations_Invariants(values Deviations, namespace aver.Namespace) {
	aver.Tree(values, namespace).Range_Int(len(values), QUORUM_MIN, SAMPLES_MAX).Ensure()
}

// WORD_MIN is the lone executable a command line always carries.
const WORD_MIN = 1

// COMMAND_WORDS_MAX bounds a command line's token count (executable + env + arguments).
const COMMAND_WORDS_MAX = 32

// COMMAND_SET_MAX bounds how many commands/benchmarks one run compares.
const COMMAND_SET_MAX = 8

// COMMAND_SET_MIN is the first command that a valid invocation benchmarks.
const COMMAND_SET_MIN = 1

// Command_Line is one command flattened to its words — the executable and its arguments.
type Command_Line []Command_Word

// Command_Line_Invariants bounds the word count; a command always has its executable, so
// the empty count is unreachable while the one-word min, two-word shape, and max are witnessed.
func Command_Line_Invariants(words Command_Line, namespace aver.Namespace) {
	aver.Tree(words, namespace).Range_Int(len(words), WORD_MIN, COMMAND_WORDS_MAX).Ensure()
}

// Command_Count is how many commands a run holds.
type Command_Count int

// Command_Count_Invariants bounds the command count to the command set.
func Command_Count_Invariants(value Command_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COLLECTION_MIN, COMMAND_SET_MAX).
		Ensure()
}

// Commands is the set of commands a run benchmarks, in invocation order. Slots past Count
// hold no command.
type Commands struct {
	// Count is how many slots hold a command.
	Count Command_Count
	// Command_1 is the command at position 0.
	Command_1 nbio.Process_Request
	// Command_2 is the command at position 1.
	Command_2 nbio.Process_Request
	// Command_3 is the command at position 2.
	Command_3 nbio.Process_Request
	// Command_4 is the command at position 3.
	Command_4 nbio.Process_Request
	// Command_5 is the command at position 4.
	Command_5 nbio.Process_Request
	// Command_6 is the command at position 5.
	Command_6 nbio.Process_Request
	// Command_7 is the command at position 6.
	Command_7 nbio.Process_Request
	// Command_8 is the command at position 7.
	Command_8 nbio.Process_Request
}

// Commands_Invariants bounds the command count.
func Commands_Invariants(value Commands, namespace aver.Namespace) {
	Command_Count_Invariants(value.Count, namespace)
}

// Command_Set is the commands a benchmark compares: at least one.
type Command_Set Commands

// Command_Set_Invariants bounds the set to one command through a full set.
func Command_Set_Invariants(value Command_Set, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), COMMAND_COUNT_MIN, COMMAND_SET_MAX).
		Ensure()
}

// Partial_Commands is a set still being filled: a free slot remains.
type Partial_Commands Commands

// Partial_Commands_Invariants bounds the set to an empty one through one short of full.
func Partial_Commands_Invariants(value Partial_Commands, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), COLLECTION_MIN, PARTIAL_COMMANDS_MAX).
		Ensure()
}

// PARTIAL_COMMANDS_MAX is one short of a full command set.
const PARTIAL_COMMANDS_MAX = COMMAND_SET_MAX - 1

// Returns the command at position.
func commands_at(commands Command_Set, position Position) (command nbio.Process_Request) {
	Command_Set_Invariants(commands, "commands_at.commands")
	Position_Invariants(position, "commands_at.position")
	aver.Always(int(position) < int(commands.Count), "A command position is inside the set.")
	switch position {
	case 0:
		return commands.Command_1
	case 1:
		return commands.Command_2
	case 2:
		return commands.Command_3
	case 3:
		return commands.Command_4
	case 4:
		return commands.Command_5
	case 5:
		return commands.Command_6
	case 6:
		return commands.Command_7
	}
	return commands.Command_8
}

// Stores command in the first free slot.
func commands_append(
	commands Partial_Commands, command nbio.Process_Request,
) (extended Command_Set) {
	defer func() { Command_Set_Invariants(extended, "commands_append.extended") }()
	Partial_Commands_Invariants(commands, "commands_append.commands")
	switch commands.Count {
	case 0:
		commands.Command_1 = command
	case 1:
		commands.Command_2 = command
	case 2:
		commands.Command_3 = command
	case 3:
		commands.Command_4 = command
	case 4:
		commands.Command_5 = command
	case 5:
		commands.Command_6 = command
	case 6:
		commands.Command_7 = command
	default:
		commands.Command_8 = command
	}
	commands.Count++
	return Command_Set(commands)
}

// REPORT_MIN is the shortest valid table: one one-byte command and no metric rows.
const REPORT_MIN = 99

// REPORT_MAX bounds the rendered bytes at the widest document the pipeline produces: a full
// command set of ceiling-scale, outlier-saturated benchmarks, colored, under a full machine
// header. Every field is at its own ceiling, so no document renders wider.
const REPORT_MAX = 9925

// HOST_TEXT_BYTES_MIN is the empty spec: a sysctl or uname read that returned nothing.
const HOST_TEXT_BYTES_MIN = 0

// HOST_TEXT_BYTES_MAX bounds a host spec field. A CPU brand or kernel string is short;
// a longer value is a malformed probe to reject, not a spec to carry in the report.
const HOST_TEXT_BYTES_MAX = 1 << 8

// Host_Text is one field of the host snapshot — a CPU brand, architecture, OS name, or
// kernel string from sysctl or uname. Short and bounded, never an arbitrary blob.
type Host_Text string

// Host_Text_Invariants bounds a host spec field's length, witnessing the empty min, the
// capped max, and the one- and two-byte shapes between.
func Host_Text_Invariants(text Host_Text, namespace aver.Namespace) {
	aver.Tree(text, namespace).
		Range_Int(len(text), HOST_TEXT_BYTES_MIN, HOST_TEXT_BYTES_MAX).
		Ensure()
}

// Processor_Model is the processor brand text.
type Processor_Model string

// Processor_Model_Invariants bounds processor brand text.
func Processor_Model_Invariants(value Processor_Model, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), HOST_TEXT_BYTES_MIN, HOST_TEXT_BYTES_MAX).
		Ensure()
}

// Processor_Architecture is the processor instruction-set name.
type Processor_Architecture string

// Processor_Architecture_Invariants bounds an instruction-set name.
func Processor_Architecture_Invariants(
	value Processor_Architecture, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), HOST_TEXT_BYTES_MIN, HOST_TEXT_BYTES_MAX).
		Ensure()
}

// Physical_Core_Count is the machine's physical core count.
type Physical_Core_Count int

// Physical_Core_Count_Invariants bounds a physical core count.
func Physical_Core_Count_Invariants(
	value Physical_Core_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), CORES_COUNT_MIN, CORES_COUNT_MAX).
		Ensure()
}

// Logical_Core_Count is the machine's visible hardware-thread count.
type Logical_Core_Count int

// Logical_Core_Count_Invariants bounds a logical core count.
func Logical_Core_Count_Invariants(value Logical_Core_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), CORES_COUNT_MIN, CORES_COUNT_MAX).
		Ensure()
}

// Performance_Core_Count is the machine's performance-core count.
type Performance_Core_Count int

// Performance_Core_Count_Invariants bounds a performance-core count.
func Performance_Core_Count_Invariants(
	value Performance_Core_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), CORES_COUNT_MIN, CORES_COUNT_MAX).
		Ensure()
}

// Efficiency_Core_Count is the machine's efficiency-core count.
type Efficiency_Core_Count int

// Efficiency_Core_Count_Invariants bounds an efficiency-core count.
func Efficiency_Core_Count_Invariants(
	value Efficiency_Core_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), CORES_COUNT_MIN, CORES_COUNT_MAX).
		Ensure()
}

// Processor_Frequency is the maximum processor frequency in hertz.
type Processor_Frequency uint64

// Processor_Frequency_Invariants bounds a processor frequency.
func Processor_Frequency_Invariants(
	value Processor_Frequency, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), HERTZ_MIN, HERTZ_MAX).
		Ensure()
}

// Level_1_Cache_Size is the processor's L1 cache size in bytes.
type Level_1_Cache_Size uint64

// Level_1_Cache_Size_Invariants bounds an L1 cache size.
func Level_1_Cache_Size_Invariants(
	value Level_1_Cache_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), BYTE_SIZE_MIN, BYTE_SIZE_MAX).
		Ensure()
}

// Level_2_Cache_Size is the processor's L2 cache size in bytes.
type Level_2_Cache_Size uint64

// Level_2_Cache_Size_Invariants bounds an L2 cache size.
func Level_2_Cache_Size_Invariants(
	value Level_2_Cache_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), BYTE_SIZE_MIN, BYTE_SIZE_MAX).
		Ensure()
}

// Level_3_Cache_Size is the processor's L3 cache size in bytes.
type Level_3_Cache_Size uint64

// Level_3_Cache_Size_Invariants bounds an L3 cache size.
func Level_3_Cache_Size_Invariants(
	value Level_3_Cache_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), BYTE_SIZE_MIN, BYTE_SIZE_MAX).
		Ensure()
}

// Memory_Size is the machine's physical memory size in bytes.
type Memory_Size uint64

// Memory_Size_Invariants bounds a physical memory size.
func Memory_Size_Invariants(value Memory_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), BYTE_SIZE_MIN, BYTE_SIZE_MAX).
		Ensure()
}

// Storage_Size is the boot volume's storage size in bytes.
type Storage_Size uint64

// Storage_Size_Invariants bounds a storage size.
func Storage_Size_Invariants(value Storage_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), BYTE_SIZE_MIN, BYTE_SIZE_MAX).
		Ensure()
}

// Operating_System_Name is the operating-system product name.
type Operating_System_Name string

// Operating_System_Name_Invariants bounds an operating-system name.
func Operating_System_Name_Invariants(
	value Operating_System_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), HOST_TEXT_BYTES_MIN, HOST_TEXT_BYTES_MAX).
		Ensure()
}

// Operating_System_Version is the operating-system release text.
type Operating_System_Version string

// Operating_System_Version_Invariants bounds operating-system release text.
func Operating_System_Version_Invariants(
	value Operating_System_Version, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), HOST_TEXT_BYTES_MIN, HOST_TEXT_BYTES_MAX).
		Ensure()
}

// Kernel_Version is the operating-system kernel release text.
type Kernel_Version string

// Kernel_Version_Invariants bounds kernel release text.
func Kernel_Version_Invariants(value Kernel_Version, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), HOST_TEXT_BYTES_MIN, HOST_TEXT_BYTES_MAX).
		Ensure()
}

// Machine_Specs is a snapshot of the host hardware and OS taken once at startup,
// carried in the report so benchmark results are reproducible across machines.
type Machine_Specs struct {
	// CPU_Model is the CPU's brand string, e.g. "Apple M4 Max".
	CPU_Model Processor_Model `json:"cpu_model"`
	// CPU_Arch is the instruction-set architecture, e.g. "arm64".
	CPU_Arch Processor_Architecture `json:"cpu_arch"`
	// Physical_Cores is the total physical core count across all performance levels.
	Physical_Cores Physical_Core_Count `json:"physical_cores"`
	// Logical_Cores is the OS-visible thread count, which may exceed Physical_Cores
	// when hyperthreading or SMT is active.
	Logical_Cores Logical_Core_Count `json:"logical_cores"`
	// Performance_Cores is the P-core count on hybrid CPUs (Apple Silicon, Alder Lake+).
	// Zero when the CPU does not expose a performance/efficiency split.
	Performance_Cores Performance_Core_Count `json:"performance_cores,omitempty"`
	// Efficiency_Cores is the E-core count on hybrid CPUs.
	Efficiency_Cores Efficiency_Core_Count `json:"efficiency_cores,omitempty"`
	// CPU_Frequency_Hz_Max is the maximum rated CPU frequency in Hz; zero when the
	// kernel does not expose it (e.g. Apple Silicon with no cpufrequency_max sysctl).
	CPU_Frequency_Hz_Max Processor_Frequency `json:"cpu_frequency_hz_max,omitempty"`
	// Cache_L1_Bytes is the per-core L1 data cache size in bytes.
	Cache_L1_Bytes Level_1_Cache_Size `json:"cache_l1_bytes,omitempty"`
	// Cache_L2_Bytes is the per-core L2 cache size in bytes.
	Cache_L2_Bytes Level_2_Cache_Size `json:"cache_l2_bytes,omitempty"`
	// Cache_L3_Bytes is the shared L3 cache size in bytes.
	Cache_L3_Bytes Level_3_Cache_Size `json:"cache_l3_bytes,omitempty"`
	// RAM_Total_Bytes is the total installed physical memory in bytes.
	RAM_Total_Bytes Memory_Size `json:"ram_total_bytes"`
	// Storage_Total_Bytes is the total capacity of the boot filesystem in bytes. It
	// is a benchmark-relevant proxy for SSD throughput: on Apple Silicon a larger
	// drive spreads I/O across more NAND dies, so the 512GB model reads and writes
	// faster than the 256GB even on identical silicon.
	Storage_Total_Bytes Storage_Size `json:"storage_total_bytes"`
	// Operating_System_Name is the operating-system name, e.g. "macOS".
	Operating_System_Name Operating_System_Name `json:"operating_system_name"`
	// Operating_System_Version is the OS release, e.g. "15.2".
	Operating_System_Version Operating_System_Version `json:"operating_system_version"`
	// Kernel_Version is the kernel release string, e.g. "Darwin 25.2.0".
	Kernel_Version Kernel_Version `json:"kernel_version"`
}

// Machine_Specs_Invariants states every primitive field of the host snapshot.
func Machine_Specs_Invariants(specs Machine_Specs, namespace aver.Namespace) {
	Processor_Model_Invariants(specs.CPU_Model, namespace)
	Processor_Architecture_Invariants(specs.CPU_Arch, namespace)
	Physical_Core_Count_Invariants(specs.Physical_Cores, namespace)
	Logical_Core_Count_Invariants(specs.Logical_Cores, namespace)
	Performance_Core_Count_Invariants(specs.Performance_Cores, namespace)
	Efficiency_Core_Count_Invariants(specs.Efficiency_Cores, namespace)
	Processor_Frequency_Invariants(specs.CPU_Frequency_Hz_Max, namespace)
	Level_1_Cache_Size_Invariants(specs.Cache_L1_Bytes, namespace)
	Level_2_Cache_Size_Invariants(specs.Cache_L2_Bytes, namespace)
	Level_3_Cache_Size_Invariants(specs.Cache_L3_Bytes, namespace)
	Memory_Size_Invariants(specs.RAM_Total_Bytes, namespace)
	Storage_Size_Invariants(specs.Storage_Total_Bytes, namespace)
	Operating_System_Name_Invariants(specs.Operating_System_Name, namespace)
	Operating_System_Version_Invariants(specs.Operating_System_Version, namespace)
	Kernel_Version_Invariants(specs.Kernel_Version, namespace)
}

// UNIT_BYTES_MIN is the shortest unit in the vocabulary — "count" and "bytes", five bytes
// each. Every rendered unit is one of the closed vocabulary, so no shorter width ever occurs.
const UNIT_BYTES_MIN = 5

// UNIT_BYTES_MAX is the longest unit in the vocabulary — "nanoseconds", eleven bytes.
// Units are a closed set, so a longer value is a malformed probe to reject.
const UNIT_BYTES_MAX = 11

// UNIT_COUNT is the unit for hardware event counters.
const UNIT_COUNT = "count"

// UNIT_SIZE is the unit for memory measurements.
const UNIT_SIZE = "bytes"

// UNIT_TIME is the unit for elapsed and processor time.
const UNIT_TIME = "nanoseconds"

// Unit names the dimension a Measurement's raw values are in — a word from a closed vocabulary
// of exactly two widths: the five-byte "count"/"bytes" and the eleven-byte "nanoseconds".
type Unit string

// Unit_Invariants uses an Enum because the vocabulary admits two lengths and no width between;
// saturation makes the all-false cell impossible instead of demanding an invented third width.
func Unit_Invariants(name Unit, namespace aver.Namespace) {
	aver.Tree(name, namespace).
		Enum_Int(len(name), UNIT_BYTES_MIN, UNIT_BYTES_MAX).
		Ensure()
}

// STANDARD_DEVIATION_MAXIMUM is the largest sample deviation for three or more
// values in the metric domain. Three samples at zero, zero, and METRIC_MAX produce it.
const STANDARD_DEVIATION_MAXIMUM int64 = 5078426674188

// Mean is the arithmetic mean of one metric.
type Mean fixedpoint.Number

// Mean_Invariants bounds a mean to the metric domain.
func Mean_Invariants(value Mean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int64(int64(value), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Ensure()
}

// METRIC_SCALED_MAX is METRIC_MAX lifted onto the fixed-point grid.
const METRIC_SCALED_MAX = METRIC_MAX * fixedpoint.SCALE

// Standard_Deviation is the sample standard deviation of one metric.
type Standard_Deviation fixedpoint.Number

// Standard_Deviation_Invariants bounds a deviation to the metric domain.
func Standard_Deviation_Invariants(value Standard_Deviation, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int64(int64(value), METRIC_MIN, STANDARD_DEVIATION_SCALED_MAXIMUM,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Ensure()
}

// STANDARD_DEVIATION_SCALED_MAXIMUM is STANDARD_DEVIATION_MAXIMUM on the fixed-point grid.
const STANDARD_DEVIATION_SCALED_MAXIMUM = STANDARD_DEVIATION_MAXIMUM * fixedpoint.SCALE

// Minimum is the smallest value of one metric.
type Minimum fixedpoint.Number

// Minimum_Invariants bounds a minimum to the metric domain.
func Minimum_Invariants(value Minimum, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int64(int64(value), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Ensure()
}

// Maximum is the largest value of one metric.
type Maximum fixedpoint.Number

// Maximum_Invariants bounds a maximum to the metric domain.
func Maximum_Invariants(value Maximum, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int64(int64(value), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Ensure()
}

// Median is the middle value of one metric.
type Median fixedpoint.Number

// Median_Invariants bounds a median to the metric domain.
func Median_Invariants(value Median, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int64(int64(value), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Ensure()
}

// First_Quartile is the lower quartile of one metric.
type First_Quartile fixedpoint.Number

// First_Quartile_Invariants bounds a first quartile to the metric domain.
func First_Quartile_Invariants(value First_Quartile, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int64(int64(value), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Ensure()
}

// Third_Quartile is the upper quartile of one metric.
type Third_Quartile fixedpoint.Number

// Third_Quartile_Invariants bounds a third quartile to the metric domain.
func Third_Quartile_Invariants(value Third_Quartile, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int64(int64(value), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Ensure()
}

// Measurement is the distribution of one metric across a command's runs — poop's
// Measurement, reduced from the raw Samples. Every field is a number so the report
// marshals to JSON without custom encoders.
type Measurement struct {
	// Mean is the arithmetic mean of the metric's values.
	Mean Mean `json:"mean"`
	// Standard_Deviation is the sample standard deviation, with an n-1 denominator.
	Standard_Deviation Standard_Deviation `json:"stddev"`
	// Min is the smallest value observed.
	Min Minimum `json:"min"`
	// Max is the largest value observed.
	Max Maximum `json:"max"`
	// Median is the middle value of the sorted values.
	Median Median `json:"median"`
	// Q1 is the first quartile by poop's index math.
	Q1 First_Quartile `json:"q1"`
	// Q3 is the third quartile by poop's index math.
	Q3 Third_Quartile `json:"q3"`
	// Outlier_Count is how many values fall beyond Tukey's fences.
	Outlier_Count Strays `json:"outliers"`
	// Sample_Count is how many values the distribution was computed from.
	Sample_Count Kept `json:"count"`
	// Unit names the unit the raw values are in.
	Unit Unit `json:"unit"`
}

// Measurement_Invariants states the integer and unit fields of a distribution;
// the fixedpoint.Number fields have no preset of their own.
func Measurement_Invariants(measurement Measurement, namespace aver.Namespace) {
	Mean_Invariants(measurement.Mean, namespace)
	Standard_Deviation_Invariants(measurement.Standard_Deviation, namespace)
	Minimum_Invariants(measurement.Min, namespace)
	Maximum_Invariants(measurement.Max, namespace)
	Median_Invariants(measurement.Median, namespace)
	First_Quartile_Invariants(measurement.Q1, namespace)
	Third_Quartile_Invariants(measurement.Q3, namespace)
	Strays_Invariants(measurement.Outlier_Count, namespace)
	Kept_Invariants(measurement.Sample_Count, namespace)
	Unit_Invariants(measurement.Unit, namespace)
}

// Difference_Percent is a signed percentage difference: a fixed-point ratio times a hundred,
// wrapping as the signed word wraps once a candidate's mean dwarfs the reference's.
type Difference_Percent fixedpoint.Number

// Difference_Percent_Invariants states the fixed-point percentage domain. A hundred times any word
// is a multiple of four even once it wraps, thus the one and two units and their negation are
// unreachable, and the largest is four short of the signed word's top.
func Difference_Percent_Invariants(value Difference_Percent, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value), bits.INTEGER_64_MINIMUM, DIFFERENCE_PERCENT_MAXIMUM,
			scalar.FRACTION_UNIT_NEGATIVE_ONE, scalar.FRACTION_UNIT_ONE,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Ensure()
}

// DIFFERENCE_PERCENT_MAXIMUM is the largest multiple of four a signed word holds.
const DIFFERENCE_PERCENT_MAXIMUM = 1<<63 - 4

// HALF_PERCENT_MAXIMUM is the widest confidence half-interval: the pooled relative deviation at
// the root of a quarter of the signed word, scaled by the four-degree score and the normalizer of
// two three-run commands, then by a hundred. A hundred times a whole word unit, it never falls to
// one or two grid units.
const HALF_PERCENT_MAXIMUM = 352442397099400

// Half_Percent is the half-width of a percentage confidence interval.
type Half_Percent fixedpoint.Number

// Half_Percent_Invariants states the nonnegative fixed-point percentage domain.
func Half_Percent_Invariants(value Half_Percent, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value), PERCENT_MINIMUM, HALF_PERCENT_MAXIMUM,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Ensure()
}

// PERCENT_MINIMUM is the zero a nonnegative fixed-point ratio floors at.
const PERCENT_MINIMUM = 0

// STUDENT_T_SCORE_MINIMUM is 1.96, the normal score past the tabulated degrees.
const STUDENT_T_SCORE_MINIMUM = 1960 * fixedpoint.SCALE / 1000

// STUDENT_T_SCORE_MAXIMUM is 2.776, the score at four degrees of freedom: two commands of at
// least RUNS_MIN kept runs each leave no fewer.
const STUDENT_T_SCORE_MAXIMUM = 2776 * fixedpoint.SCALE / 1000

// Student_T_Score is a 95-percent Student-t critical score.
type Student_T_Score fixedpoint.Number

// Student_T_Score_Invariants states the two whole-number score regions in the table.
func Student_T_Score_Invariants(value Student_T_Score, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), STUDENT_T_SCORE_MINIMUM, STUDENT_T_SCORE_MAXIMUM).
		Ensure()
}

// SIGNIFICANCE_SIGNIFICANT is the shared positive significance fact.
const SIGNIFICANCE_SIGNIFICANT = true

// Significance records whether a confidence interval clears the significance band.
type Significance bool

// Significance_Invariants witnesses both significance outcomes.
func Significance_Invariants(value Significance, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The difference is significant.").
		Ensure()
}

// SPEED_ORDER_FASTER is the shared positive speed-order fact.
const SPEED_ORDER_FASTER = true

// Speed_Order records whether the candidate is faster than the reference.
type Speed_Order bool

// Speed_Order_Invariants witnesses both speed orders.
func Speed_Order_Invariants(value Speed_Order, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The candidate is faster.").
		Ensure()
}

// Delta is one metric's change in a candidate command relative to the reference —
// poop's colored ratio column, as data.
type Delta struct {
	// Diff_Percent is the candidate's mean as a signed percentage of the reference's.
	Diff_Percent Difference_Percent `json:"diff_percent"`
	// Half_Percent is the half-width of the 95% confidence interval on Diff_Percent.
	Half_Percent Half_Percent `json:"half_percent"`
	// Significant is true only when the interval clears the ±1% band.
	Significant Significance `json:"significant"`
	// Faster is true when the candidate's mean is below the reference's.
	Faster Speed_Order `json:"faster"`
}

// Delta_Invariants states the boolean fields of a metric's change; the
// fixedpoint.Number fields have no preset of their own.
func Delta_Invariants(delta Delta, namespace aver.Namespace) {
	Difference_Percent_Invariants(delta.Diff_Percent, namespace)
	Half_Percent_Invariants(delta.Half_Percent, namespace)
	Significance_Invariants(delta.Significant, namespace)
	Speed_Order_Invariants(delta.Faster, namespace)
}

// Wall_Time_Measurement is the wall-time distribution.
type Wall_Time_Measurement Measurement

// Wall_Time_Measurement_Invariants states the wall-time distribution fields.
func Wall_Time_Measurement_Invariants(
	value Wall_Time_Measurement, namespace aver.Namespace,
) {
	aver.Always(value.Unit == UNIT_TIME, "Wall time always uses nanoseconds.")
	aver.Always(
		int64(value.Standard_Deviation)/fixedpoint.SCALE <= STANDARD_DEVIATION_MAXIMUM,
		"A wall-time standard deviation cannot exceed the metric domain.",
	)
	aver.Tree(value, namespace).
		Range_Holed_Int64(int64(value.Mean), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(
			int64(value.Standard_Deviation),
			METRIC_MIN,
			STANDARD_DEVIATION_SCALED_MAXIMUM,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Min), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Max), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Median), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Q1), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Q3), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Int(int(value.Outlier_Count), STRAYS_MIN, STRAYS_MAX).
		Range_Int(int(value.Sample_Count), KEPT_MIN, KEPT_MAX).
		Ensure()
}

// Peak_Resident_Measurement is the peak resident-memory distribution.
type Peak_Resident_Measurement Measurement

// Peak_Resident_Measurement_Invariants states the resident-memory distribution fields.
func Peak_Resident_Measurement_Invariants(
	value Peak_Resident_Measurement, namespace aver.Namespace,
) {
	aver.Always(value.Unit == UNIT_SIZE, "Peak resident memory always uses bytes.")
	aver.Always(
		int64(value.Standard_Deviation)/fixedpoint.SCALE <= STANDARD_DEVIATION_MAXIMUM,
		"A resident-memory standard deviation cannot exceed the metric domain.",
	)
	aver.Tree(value, namespace).
		Range_Holed_Int64(int64(value.Mean), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(
			int64(value.Standard_Deviation),
			METRIC_MIN,
			STANDARD_DEVIATION_SCALED_MAXIMUM,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Min), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Max), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Median), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Q1), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Q3), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Int(int(value.Outlier_Count), STRAYS_MIN, STRAYS_MAX).
		Range_Int(int(value.Sample_Count), KEPT_MIN, KEPT_MAX).
		Ensure()
}

// Cycle_Measurement is the processor-cycle distribution.
type Cycle_Measurement Measurement

// Cycle_Measurement_Invariants states the processor-cycle distribution fields.
func Cycle_Measurement_Invariants(value Cycle_Measurement, namespace aver.Namespace) {
	aver.Always(value.Unit == UNIT_COUNT, "Processor cycles always use a count.")
	aver.Always(
		int64(value.Standard_Deviation)/fixedpoint.SCALE <= STANDARD_DEVIATION_MAXIMUM,
		"A processor-cycle standard deviation cannot exceed the metric domain.",
	)
	aver.Tree(value, namespace).
		Range_Holed_Int64(int64(value.Mean), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(
			int64(value.Standard_Deviation),
			METRIC_MIN,
			STANDARD_DEVIATION_SCALED_MAXIMUM,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Min), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Max), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Median), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Q1), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Q3), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Int(int(value.Outlier_Count), STRAYS_MIN, STRAYS_MAX).
		Range_Int(int(value.Sample_Count), KEPT_MIN, KEPT_MAX).
		Ensure()
}

// Instruction_Measurement is the retired-instruction distribution.
type Instruction_Measurement Measurement

// Instruction_Measurement_Invariants states the instruction distribution fields.
func Instruction_Measurement_Invariants(
	value Instruction_Measurement, namespace aver.Namespace,
) {
	aver.Always(value.Unit == UNIT_COUNT, "Instructions always use a count.")
	aver.Always(
		int64(value.Standard_Deviation)/fixedpoint.SCALE <= STANDARD_DEVIATION_MAXIMUM,
		"An instruction standard deviation cannot exceed the metric domain.",
	)
	aver.Tree(value, namespace).
		Range_Holed_Int64(int64(value.Mean), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(
			int64(value.Standard_Deviation),
			METRIC_MIN,
			STANDARD_DEVIATION_SCALED_MAXIMUM,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Min), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Max), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Median), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Q1), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Q3), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Int(int(value.Outlier_Count), STRAYS_MIN, STRAYS_MAX).
		Range_Int(int(value.Sample_Count), KEPT_MIN, KEPT_MAX).
		Ensure()
}

// Cache_Reference_Measurement is the cache-reference distribution.
type Cache_Reference_Measurement Measurement

// Cache_Reference_Measurement_Invariants states the cache-reference distribution fields.
func Cache_Reference_Measurement_Invariants(
	value Cache_Reference_Measurement, namespace aver.Namespace,
) {
	aver.Always(value.Unit == UNIT_COUNT, "Cache references always use a count.")
	aver.Always(
		int64(value.Standard_Deviation)/fixedpoint.SCALE <= STANDARD_DEVIATION_MAXIMUM,
		"A cache-reference standard deviation cannot exceed the metric domain.",
	)
	aver.Tree(value, namespace).
		Range_Holed_Int64(int64(value.Mean), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(
			int64(value.Standard_Deviation),
			METRIC_MIN,
			STANDARD_DEVIATION_SCALED_MAXIMUM,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Min), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Max), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Median), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Q1), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Q3), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Int(int(value.Outlier_Count), STRAYS_MIN, STRAYS_MAX).
		Range_Int(int(value.Sample_Count), KEPT_MIN, KEPT_MAX).
		Ensure()
}

// Cache_Miss_Measurement is the cache-miss distribution.
type Cache_Miss_Measurement Measurement

// Cache_Miss_Measurement_Invariants states the cache-miss distribution fields.
func Cache_Miss_Measurement_Invariants(
	value Cache_Miss_Measurement, namespace aver.Namespace,
) {
	aver.Always(value.Unit == UNIT_COUNT, "Cache misses always use a count.")
	aver.Always(
		int64(value.Standard_Deviation)/fixedpoint.SCALE <= STANDARD_DEVIATION_MAXIMUM,
		"A cache-miss standard deviation cannot exceed the metric domain.",
	)
	aver.Tree(value, namespace).
		Range_Holed_Int64(int64(value.Mean), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(
			int64(value.Standard_Deviation),
			METRIC_MIN,
			STANDARD_DEVIATION_SCALED_MAXIMUM,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Min), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Max), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Median), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Q1), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Q3), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Int(int(value.Outlier_Count), STRAYS_MIN, STRAYS_MAX).
		Range_Int(int(value.Sample_Count), KEPT_MIN, KEPT_MAX).
		Ensure()
}

// Branch_Miss_Measurement is the branch-miss distribution.
type Branch_Miss_Measurement Measurement

// Branch_Miss_Measurement_Invariants states the branch-miss distribution fields.
func Branch_Miss_Measurement_Invariants(
	value Branch_Miss_Measurement, namespace aver.Namespace,
) {
	aver.Always(value.Unit == UNIT_COUNT, "Branch misses always use a count.")
	aver.Always(
		int64(value.Standard_Deviation)/fixedpoint.SCALE <= STANDARD_DEVIATION_MAXIMUM,
		"A branch-miss standard deviation cannot exceed the metric domain.",
	)
	aver.Tree(value, namespace).
		Range_Holed_Int64(int64(value.Mean), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(
			int64(value.Standard_Deviation),
			METRIC_MIN,
			STANDARD_DEVIATION_SCALED_MAXIMUM,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Min), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Max), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Median), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Q1), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Q3), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Int(int(value.Outlier_Count), STRAYS_MIN, STRAYS_MAX).
		Range_Int(int(value.Sample_Count), KEPT_MIN, KEPT_MAX).
		Ensure()
}

// User_Time_Measurement is the user-CPU-time distribution.
type User_Time_Measurement Measurement

// User_Time_Measurement_Invariants states the user-time distribution fields.
func User_Time_Measurement_Invariants(
	value User_Time_Measurement, namespace aver.Namespace,
) {
	aver.Always(value.Unit == UNIT_TIME, "User time always uses nanoseconds.")
	aver.Always(
		int64(value.Standard_Deviation)/fixedpoint.SCALE <= STANDARD_DEVIATION_MAXIMUM,
		"A user-time standard deviation cannot exceed the metric domain.",
	)
	aver.Tree(value, namespace).
		Range_Holed_Int64(int64(value.Mean), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(
			int64(value.Standard_Deviation),
			METRIC_MIN,
			STANDARD_DEVIATION_SCALED_MAXIMUM,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Min), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Max), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Median), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Q1), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Q3), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Int(int(value.Outlier_Count), STRAYS_MIN, STRAYS_MAX).
		Range_Int(int(value.Sample_Count), KEPT_MIN, KEPT_MAX).
		Ensure()
}

// System_Time_Measurement is the system-CPU-time distribution.
type System_Time_Measurement Measurement

// System_Time_Measurement_Invariants states the system-time distribution fields.
func System_Time_Measurement_Invariants(
	value System_Time_Measurement, namespace aver.Namespace,
) {
	aver.Always(value.Unit == UNIT_TIME, "System time always uses nanoseconds.")
	aver.Always(
		int64(value.Standard_Deviation)/fixedpoint.SCALE <= STANDARD_DEVIATION_MAXIMUM,
		"A system-time standard deviation cannot exceed the metric domain.",
	)
	aver.Tree(value, namespace).
		Range_Holed_Int64(int64(value.Mean), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(
			int64(value.Standard_Deviation),
			METRIC_MIN,
			STANDARD_DEVIATION_SCALED_MAXIMUM,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Min), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Max), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Median), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Q1), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Q3), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Int(int(value.Outlier_Count), STRAYS_MIN, STRAYS_MAX).
		Range_Int(int(value.Sample_Count), KEPT_MIN, KEPT_MAX).
		Ensure()
}

// Measurements is the distribution of every metric for one command, in report
// order. The field order is the JSON order, so the report needs no custom encoder.
type Measurements struct {
	// Wall_Time is the elapsed-time distribution.
	Wall_Time Wall_Time_Measurement `json:"wall_time"`
	// Peak_RSS is the peak-memory distribution.
	Peak_RSS Peak_Resident_Measurement `json:"peak_rss"`
	// CPU_Cycles is the CPU-cycle distribution.
	CPU_Cycles Cycle_Measurement `json:"cpu_cycles"`
	// Instructions is the retired-instruction distribution.
	Instructions Instruction_Measurement `json:"instructions"`
	// Cache_References is the cache-reference distribution (Linux only).
	Cache_References Cache_Reference_Measurement `json:"cache_references"`
	// Cache_Misses is the cache-miss distribution (Linux only).
	Cache_Misses Cache_Miss_Measurement `json:"cache_misses"`
	// Branch_Misses is the branch-miss distribution (Linux only).
	Branch_Misses Branch_Miss_Measurement `json:"branch_misses"`
	// CPU_User is the user-CPU-time distribution.
	CPU_User User_Time_Measurement `json:"cpu_user"`
	// CPU_System is the system-CPU-time distribution.
	CPU_System System_Time_Measurement `json:"cpu_system"`
}

// Measurements_Invariants composes the per-metric distribution of every field.
func Measurements_Invariants(measurements Measurements, namespace aver.Namespace) {
	Wall_Time_Measurement_Invariants(measurements.Wall_Time, namespace)
	Peak_Resident_Measurement_Invariants(measurements.Peak_RSS, namespace)
	Cycle_Measurement_Invariants(measurements.CPU_Cycles, namespace)
	Instruction_Measurement_Invariants(measurements.Instructions, namespace)
	Cache_Reference_Measurement_Invariants(measurements.Cache_References, namespace)
	Cache_Miss_Measurement_Invariants(measurements.Cache_Misses, namespace)
	Branch_Miss_Measurement_Invariants(measurements.Branch_Misses, namespace)
	User_Time_Measurement_Invariants(measurements.CPU_User, namespace)
	System_Time_Measurement_Invariants(measurements.CPU_System, namespace)
}

// Wall_Time_Delta is the wall-time comparison.
type Wall_Time_Delta Delta

// Wall_Time_Delta_Invariants states the wall-time comparison outcomes.
func Wall_Time_Delta_Invariants(value Wall_Time_Delta, namespace aver.Namespace) {
	aver.Always(
		value.Half_Percent >= 0,
		"A wall-time confidence half-interval cannot be negative.",
	)
	aver.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value.Diff_Percent),
			bits.INTEGER_64_MINIMUM, DIFFERENCE_PERCENT_MAXIMUM,
			scalar.FRACTION_UNIT_NEGATIVE_ONE, scalar.FRACTION_UNIT_ONE,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(
			int64(value.Half_Percent), PERCENT_MINIMUM, HALF_PERCENT_MAXIMUM,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Sometimes(bool(value.Significant), "The difference is significant.").
		Sometimes(bool(value.Faster), "The candidate is faster.").
		Ensure()
}

// Peak_Resident_Delta is the peak resident-memory comparison.
type Peak_Resident_Delta Delta

// Peak_Resident_Delta_Invariants states the resident-memory comparison outcomes.
func Peak_Resident_Delta_Invariants(value Peak_Resident_Delta, namespace aver.Namespace) {
	aver.Always(
		value.Half_Percent >= 0,
		"A resident-memory confidence half-interval cannot be negative.",
	)
	aver.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value.Diff_Percent),
			bits.INTEGER_64_MINIMUM, DIFFERENCE_PERCENT_MAXIMUM,
			scalar.FRACTION_UNIT_NEGATIVE_ONE, scalar.FRACTION_UNIT_ONE,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(
			int64(value.Half_Percent), PERCENT_MINIMUM, HALF_PERCENT_MAXIMUM,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Sometimes(bool(value.Significant), "The difference is significant.").
		Sometimes(bool(value.Faster), "The candidate is faster.").
		Ensure()
}

// Cycle_Delta is the processor-cycle comparison.
type Cycle_Delta Delta

// Cycle_Delta_Invariants states the processor-cycle comparison outcomes.
func Cycle_Delta_Invariants(value Cycle_Delta, namespace aver.Namespace) {
	aver.Always(
		value.Half_Percent >= 0,
		"A processor-cycle confidence half-interval cannot be negative.",
	)
	aver.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value.Diff_Percent),
			bits.INTEGER_64_MINIMUM, DIFFERENCE_PERCENT_MAXIMUM,
			scalar.FRACTION_UNIT_NEGATIVE_ONE, scalar.FRACTION_UNIT_ONE,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(
			int64(value.Half_Percent), PERCENT_MINIMUM, HALF_PERCENT_MAXIMUM,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Sometimes(bool(value.Significant), "The difference is significant.").
		Sometimes(bool(value.Faster), "The candidate is faster.").
		Ensure()
}

// Instruction_Delta is the retired-instruction comparison.
type Instruction_Delta Delta

// Instruction_Delta_Invariants states the instruction comparison outcomes.
func Instruction_Delta_Invariants(value Instruction_Delta, namespace aver.Namespace) {
	aver.Always(
		value.Half_Percent >= 0,
		"An instruction confidence half-interval cannot be negative.",
	)
	aver.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value.Diff_Percent),
			bits.INTEGER_64_MINIMUM, DIFFERENCE_PERCENT_MAXIMUM,
			scalar.FRACTION_UNIT_NEGATIVE_ONE, scalar.FRACTION_UNIT_ONE,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(
			int64(value.Half_Percent), PERCENT_MINIMUM, HALF_PERCENT_MAXIMUM,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Sometimes(bool(value.Significant), "The difference is significant.").
		Sometimes(bool(value.Faster), "The candidate is faster.").
		Ensure()
}

// Cache_Reference_Delta is the cache-reference comparison.
type Cache_Reference_Delta Delta

// Cache_Reference_Delta_Invariants states the cache-reference comparison outcomes.
func Cache_Reference_Delta_Invariants(
	value Cache_Reference_Delta, namespace aver.Namespace,
) {
	aver.Always(
		value.Half_Percent >= 0,
		"A cache-reference confidence half-interval cannot be negative.",
	)
	aver.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value.Diff_Percent),
			bits.INTEGER_64_MINIMUM, DIFFERENCE_PERCENT_MAXIMUM,
			scalar.FRACTION_UNIT_NEGATIVE_ONE, scalar.FRACTION_UNIT_ONE,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(
			int64(value.Half_Percent), PERCENT_MINIMUM, HALF_PERCENT_MAXIMUM,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Sometimes(bool(value.Significant), "The difference is significant.").
		Sometimes(bool(value.Faster), "The candidate is faster.").
		Ensure()
}

// Cache_Miss_Delta is the cache-miss comparison.
type Cache_Miss_Delta Delta

// Cache_Miss_Delta_Invariants states the cache-miss comparison outcomes.
func Cache_Miss_Delta_Invariants(value Cache_Miss_Delta, namespace aver.Namespace) {
	aver.Always(
		value.Half_Percent >= 0,
		"A cache-miss confidence half-interval cannot be negative.",
	)
	aver.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value.Diff_Percent),
			bits.INTEGER_64_MINIMUM, DIFFERENCE_PERCENT_MAXIMUM,
			scalar.FRACTION_UNIT_NEGATIVE_ONE, scalar.FRACTION_UNIT_ONE,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(
			int64(value.Half_Percent), PERCENT_MINIMUM, HALF_PERCENT_MAXIMUM,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Sometimes(bool(value.Significant), "The difference is significant.").
		Sometimes(bool(value.Faster), "The candidate is faster.").
		Ensure()
}

// Branch_Miss_Delta is the branch-miss comparison.
type Branch_Miss_Delta Delta

// Branch_Miss_Delta_Invariants states the branch-miss comparison outcomes.
func Branch_Miss_Delta_Invariants(value Branch_Miss_Delta, namespace aver.Namespace) {
	aver.Always(
		value.Half_Percent >= 0,
		"A branch-miss confidence half-interval cannot be negative.",
	)
	aver.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value.Diff_Percent),
			bits.INTEGER_64_MINIMUM, DIFFERENCE_PERCENT_MAXIMUM,
			scalar.FRACTION_UNIT_NEGATIVE_ONE, scalar.FRACTION_UNIT_ONE,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(
			int64(value.Half_Percent), PERCENT_MINIMUM, HALF_PERCENT_MAXIMUM,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Sometimes(bool(value.Significant), "The difference is significant.").
		Sometimes(bool(value.Faster), "The candidate is faster.").
		Ensure()
}

// User_Time_Delta is the user-CPU-time comparison.
type User_Time_Delta Delta

// User_Time_Delta_Invariants states the user-time comparison outcomes.
func User_Time_Delta_Invariants(value User_Time_Delta, namespace aver.Namespace) {
	aver.Always(
		value.Half_Percent >= 0,
		"A user-time confidence half-interval cannot be negative.",
	)
	aver.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value.Diff_Percent),
			bits.INTEGER_64_MINIMUM, DIFFERENCE_PERCENT_MAXIMUM,
			scalar.FRACTION_UNIT_NEGATIVE_ONE, scalar.FRACTION_UNIT_ONE,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(
			int64(value.Half_Percent), PERCENT_MINIMUM, HALF_PERCENT_MAXIMUM,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Sometimes(bool(value.Significant), "The difference is significant.").
		Sometimes(bool(value.Faster), "The candidate is faster.").
		Ensure()
}

// System_Time_Delta is the system-CPU-time comparison.
type System_Time_Delta Delta

// System_Time_Delta_Invariants states the system-time comparison outcomes.
func System_Time_Delta_Invariants(value System_Time_Delta, namespace aver.Namespace) {
	aver.Always(
		value.Half_Percent >= 0,
		"A system-time confidence half-interval cannot be negative.",
	)
	aver.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value.Diff_Percent),
			bits.INTEGER_64_MINIMUM, DIFFERENCE_PERCENT_MAXIMUM,
			scalar.FRACTION_UNIT_NEGATIVE_ONE, scalar.FRACTION_UNIT_ONE,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(
			int64(value.Half_Percent), PERCENT_MINIMUM, HALF_PERCENT_MAXIMUM,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Sometimes(bool(value.Significant), "The difference is significant.").
		Sometimes(bool(value.Faster), "The candidate is faster.").
		Ensure()
}

// Deltas is every metric's change for one command relative to the reference, in the
// same order as Measurements.
type Deltas struct {
	// Wall_Time is the elapsed-time delta.
	Wall_Time Wall_Time_Delta `json:"wall_time"`
	// Peak_RSS is the peak-memory delta.
	Peak_RSS Peak_Resident_Delta `json:"peak_rss"`
	// CPU_Cycles is the CPU-cycle delta.
	CPU_Cycles Cycle_Delta `json:"cpu_cycles"`
	// Instructions is the retired-instruction delta.
	Instructions Instruction_Delta `json:"instructions"`
	// Cache_References is the cache-reference delta (Linux only).
	Cache_References Cache_Reference_Delta `json:"cache_references"`
	// Cache_Misses is the cache-miss delta (Linux only).
	Cache_Misses Cache_Miss_Delta `json:"cache_misses"`
	// Branch_Misses is the branch-miss delta (Linux only).
	Branch_Misses Branch_Miss_Delta `json:"branch_misses"`
	// CPU_User is the user-CPU-time delta.
	CPU_User User_Time_Delta `json:"cpu_user"`
	// CPU_System is the system-CPU-time delta.
	CPU_System System_Time_Delta `json:"cpu_system"`
}

// Deltas_Invariants composes the change of every metric field.
func Deltas_Invariants(deltas Deltas, namespace aver.Namespace) {
	Wall_Time_Delta_Invariants(deltas.Wall_Time, namespace)
	Peak_Resident_Delta_Invariants(deltas.Peak_RSS, namespace)
	Cycle_Delta_Invariants(deltas.CPU_Cycles, namespace)
	Instruction_Delta_Invariants(deltas.Instructions, namespace)
	Cache_Reference_Delta_Invariants(deltas.Cache_References, namespace)
	Cache_Miss_Delta_Invariants(deltas.Cache_Misses, namespace)
	Branch_Miss_Delta_Invariants(deltas.Branch_Misses, namespace)
	User_Time_Delta_Invariants(deltas.CPU_User, namespace)
	System_Time_Delta_Invariants(deltas.CPU_System, namespace)
}

// ELAPSED_MINIMUM is no sampling time at all.
const ELAPSED_MINIMUM = 0

// Elapsed is accumulated sampling time.
type Elapsed time.Duration

// Elapsed_Invariants distinguishes an instant sample set from one that consumed time.
func Elapsed_Invariants(value Elapsed, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), ELAPSED_MINIMUM, int64(time.MONOTONIC_MOMENT_MAXIMUM)).
		Sometimes(value > 0, "Sampling consumed time.").
		Ensure()
}

// Benchmark is one command's entry in the report.
type Benchmark struct {
	// Command is the command's words, as a reader recognizes them.
	Command Command_Line `json:"command"`
	// Runs is how many samples were kept, warmup excluded.
	Runs Kept `json:"runs"`
	// Elapsed is the total wall time of the kept runs.
	Elapsed Elapsed `json:"elapsed_ns"`
	// Measurements is the per-metric distribution.
	Measurements Measurements `json:"measurements"`
	// Deltas is the change against the reference; the reference compared against
	// itself is the zero delta. Which command is the reference is positional — the
	// first — not encoded here, so every benchmark carries a comparable delta.
	Deltas Deltas `json:"deltas"`
}

// Benchmark_Invariants states a Benchmark's fields, composing the delta distribution
// like any other value field.
func Benchmark_Invariants(benchmark Benchmark, namespace aver.Namespace) {
	Command_Line_Invariants(benchmark.Command, namespace)
	Kept_Invariants(benchmark.Runs, namespace)
	Elapsed_Invariants(benchmark.Elapsed, namespace)
	Measurements_Invariants(benchmark.Measurements, namespace)
	Deltas_Invariants(benchmark.Deltas, namespace)
}

// ARGUMENTS_COUNT_MIN is the program name that every process invocation carries.
const ARGUMENTS_COUNT_MIN = 1

// ARGUMENTS_COUNT_MAX bounds a realistic process command line.
const ARGUMENTS_COUNT_MAX = 4096

// Arguments is an operating-system command line, including the program name.
type Arguments []string

// Arguments_Invariants bounds the operating-system argument count.
func Arguments_Invariants(arguments Arguments, namespace aver.Namespace) {
	aver.Tree(arguments, namespace).
		Range_Int(len(arguments), ARGUMENTS_COUNT_MIN, ARGUMENTS_COUNT_MAX).
		Ensure()
}

// COMMAND_COUNT_MIN is the one command a benchmark needs.
const COMMAND_COUNT_MIN = 1

// CLI_COMMANDS_COUNT_MAX leaves one operating-system argument for the program name.
const CLI_COMMANDS_COUNT_MAX = ARGUMENTS_COUNT_MAX - 1

// Cli_Commands is the command text parsed from positional arguments.
type Cli_Commands []string

// Cli_Commands_Invariants bounds the positional command count.
func Cli_Commands_Invariants(commands Cli_Commands, namespace aver.Namespace) {
	aver.Tree(commands, namespace).
		Range_Int(len(commands), COMMAND_COUNT_MIN, CLI_COMMANDS_COUNT_MAX).
		Ensure()
}

// STREAM_MODE_BYTES_MIN is the byte length of "auto".
const STREAM_MODE_BYTES_MIN = 4

// STREAM_MODE_BYTES_MAX is the byte length of "always".
const STREAM_MODE_BYTES_MAX = 6

// STREAM_MODE_BYTES_MIDDLE is the byte length of "never".
const STREAM_MODE_BYTES_MIDDLE = 5

// Stream_Mode is an automatic, disabled, or enabled stream setting.
type Stream_Mode string

// Stream_Mode_Invariants bounds the closed mode vocabulary by its three lengths.
func Stream_Mode_Invariants(mode Stream_Mode, namespace aver.Namespace) {
	aver.Tree(mode, namespace).
		Enum_3_Int(
			len(mode), STREAM_MODE_BYTES_MIN,
			STREAM_MODE_BYTES_MIDDLE, STREAM_MODE_BYTES_MAX).
		Ensure()
}

// TERMINAL_STATUS_NOT_TERMINAL is the shared negative terminal state.
const TERMINAL_STATUS_NOT_TERMINAL uint8 = 0

// TERMINAL_STATUS_TERMINAL is the shared positive terminal state.
const TERMINAL_STATUS_TERMINAL uint8 = 1

// Terminal_Status states whether an injected stream points at a terminal.
type Terminal_Status uint8

// Terminal_Status_Invariants witnesses both terminal states.
func Terminal_Status_Invariants(status Terminal_Status, namespace aver.Namespace) {
	aver.Tree(status, namespace).
		Enum_Uint8(
			uint8(status), TERMINAL_STATUS_NOT_TERMINAL, TERMINAL_STATUS_TERMINAL).
		Ensure()
}

// Output_Terminal_Status states whether the report stream points at a terminal.
type Output_Terminal_Status Terminal_Status

// Output_Terminal_Status_Invariants witnesses both report-stream states.
func Output_Terminal_Status_Invariants(
	status Output_Terminal_Status, namespace aver.Namespace,
) {
	aver.Tree(status, namespace).
		Enum_Uint8(
			uint8(status), TERMINAL_STATUS_NOT_TERMINAL, TERMINAL_STATUS_TERMINAL).
		Ensure()
}

// Error_Output_Terminal_Status states whether the diagnostic stream is a terminal.
type Error_Output_Terminal_Status Terminal_Status

// Error_Output_Terminal_Status_Invariants witnesses both diagnostic-stream states.
func Error_Output_Terminal_Status_Invariants(
	status Error_Output_Terminal_Status, namespace aver.Namespace,
) {
	aver.Tree(status, namespace).
		Enum_Uint8(
			uint8(status), TERMINAL_STATUS_NOT_TERMINAL, TERMINAL_STATUS_TERMINAL).
		Ensure()
}

// Stream_State states whether Maddox enables one optional stream feature.
type Stream_State bool

// Stream_State_Invariants witnesses both feature states.
func Stream_State_Invariants(state Stream_State, namespace aver.Namespace) {
	aver.Tree(state, namespace).
		Sometimes(bool(state), "The stream feature is active.").
		Ensure()
}

// Duration_Limit is the optional per-command sampling budget, in whole seconds as the command line
// states it.
type Duration_Limit int64

// Duration_Limit_Invariants distinguishes an active budget from a disabled budget.
func Duration_Limit_Invariants(value Duration_Limit, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Sometimes(value > 0, "A duration limit is active.").
		Ensure()
}

// Sampling_Should_Continue_Input is the loop state sampling_should_continue judges.
type Sampling_Should_Continue_Input struct {
	// Elapsed is the wall time spent sampling so far.
	Elapsed Elapsed
	// Duration_Max is the time budget; zero disables it.
	Duration_Max Duration_Limit
	// Runs_Max is the run cap; zero disables it.
	Runs_Max Run_Limit
	// Count is how many runs have been kept so far.
	Count Tally
}

// Sampling_Should_Continue_Input_Invariants states the sampling loop state.
func Sampling_Should_Continue_Input_Invariants(
	input Sampling_Should_Continue_Input, namespace aver.Namespace,
) {
	Elapsed_Invariants(input.Elapsed, namespace)
	Duration_Limit_Invariants(input.Duration_Max, namespace)
	Run_Limit_Invariants(input.Runs_Max, namespace)
	Tally_Invariants(input.Count, namespace)
}

// Sampling_should_continue decides whether to take another sample. The 3-run minimum
// always wins first and the 10000-run cap always stops; between them, sampling stops
// when any active limit is met — the run cap or the time budget — and a limit of zero
// is inactive, so both zero leaves only the safety cap. The compound condition is
// split into nested single-term ifs for the linter.
func sampling_should_continue(input Sampling_Should_Continue_Input_Handle) (yes Continuation) {
	defer func() {
		Continuation_Invariants(yes, "sampling_should_continue.yes")
	}()
	Sampling_Should_Continue_Input_Handle_Invariants(input, "sampling_should_continue.input")
	if int(input.Count) < RUNS_MIN {
		return true
	}
	if int(input.Count) >= SAMPLES_MAX {
		return false
	}
	if input.Runs_Max > 0 {
		if Run_Limit(input.Count) >= input.Runs_Max {
			return false
		}
	}
	// The product wraps as the flag's nanoseconds always have, so an overflowing budget stays
	// the budget it was.
	budget := time.Duration(input.Duration_Max) * time.SECOND
	if budget > 0 {
		if time.Duration(input.Elapsed) >= budget {
			return false
		}
	}
	return true
}

// Samples_elapsed sums the wall time of the kept runs — how long the command's
// measured sampling took in total.
func samples_elapsed(samples Distribution) (elapsed Elapsed) {
	defer func() { Elapsed_Invariants(elapsed, "samples_elapsed.elapsed") }()
	Distribution_Invariants(samples, "samples_elapsed.samples")
	for _, sample := range samples {
		elapsed += Elapsed(sample.Wall)
	}
	return elapsed
}

// Measurements_compute reduces the samples to one Measurement per metric, tagging
// each with the unit its raw values are in.
func measurements_compute(samples Distribution) (measurements Measurements) {
	defer func() {
		Measurements_Invariants(measurements, "measurements_compute.measurements")
	}()
	Distribution_Invariants(samples, "measurements_compute.samples")
	measurements.Wall_Time = Wall_Time_Measurement(Measurement_Compute(
		Values(extract(samples, sample_wall)), UNIT_TIME))
	measurements.Peak_RSS = Peak_Resident_Measurement(Measurement_Compute(
		Values(extract(samples, sample_rss)), UNIT_SIZE))
	measurements.CPU_Cycles = Cycle_Measurement(Measurement_Compute(
		Values(extract(samples, sample_cycles)), UNIT_COUNT))
	measurements.Instructions = Instruction_Measurement(Measurement_Compute(
		Values(extract(samples, sample_instructions)), UNIT_COUNT))
	measurements.Cache_References = Cache_Reference_Measurement(Measurement_Compute(
		Values(extract(samples, sample_cache_references)), UNIT_COUNT))
	measurements.Cache_Misses = Cache_Miss_Measurement(Measurement_Compute(
		Values(extract(samples, sample_cache_misses)), UNIT_COUNT))
	measurements.Branch_Misses = Branch_Miss_Measurement(Measurement_Compute(
		Values(extract(samples, sample_branch_misses)), UNIT_COUNT))
	measurements.CPU_User = User_Time_Measurement(Measurement_Compute(
		Values(extract(samples, sample_user)), UNIT_TIME))
	measurements.CPU_System = System_Time_Measurement(Measurement_Compute(
		Values(extract(samples, sample_system)), UNIT_TIME))
	return measurements
}

// Candidate_Measurements is the candidate metric set for one report comparison.
type Candidate_Measurements Measurements

// Candidate_Measurements_Invariants states every candidate distribution field.
func Candidate_Measurements_Invariants(
	value Candidate_Measurements, namespace aver.Namespace,
) {
	Wall_Time_Measurement_Invariants(value.Wall_Time, namespace)
	Peak_Resident_Measurement_Invariants(value.Peak_RSS, namespace)
	Cycle_Measurement_Invariants(value.CPU_Cycles, namespace)
	Instruction_Measurement_Invariants(value.Instructions, namespace)
	Cache_Reference_Measurement_Invariants(value.Cache_References, namespace)
	Cache_Miss_Measurement_Invariants(value.Cache_Misses, namespace)
	Branch_Miss_Measurement_Invariants(value.Branch_Misses, namespace)
	User_Time_Measurement_Invariants(value.CPU_User, namespace)
	System_Time_Measurement_Invariants(value.CPU_System, namespace)
}

// Deltas_compute compares every metric of the candidate against the reference.
func deltas_compute(reference Measurements, candidate Candidate_Measurements) (deltas Deltas) {
	defer func() {
		Deltas_Invariants(deltas, "deltas_compute.deltas")
	}()
	Measurements_Invariants(reference, "deltas_compute.reference")
	Candidate_Measurements_Invariants(candidate, "deltas_compute.candidate")
	deltas.Wall_Time = Wall_Time_Delta(Compare(
		Measurement(reference.Wall_Time), Candidate_Measurement(candidate.Wall_Time)))
	deltas.Peak_RSS = Peak_Resident_Delta(Compare(
		Measurement(reference.Peak_RSS), Candidate_Measurement(candidate.Peak_RSS)))
	deltas.CPU_Cycles = Cycle_Delta(Compare(
		Measurement(reference.CPU_Cycles), Candidate_Measurement(candidate.CPU_Cycles)))
	deltas.Instructions = Instruction_Delta(Compare(
		Measurement(reference.Instructions), Candidate_Measurement(candidate.Instructions)))
	deltas.Cache_References = Cache_Reference_Delta(Compare(
		Measurement(reference.Cache_References),
		Candidate_Measurement(candidate.Cache_References)))
	deltas.Cache_Misses = Cache_Miss_Delta(Compare(
		Measurement(reference.Cache_Misses), Candidate_Measurement(candidate.Cache_Misses)))
	deltas.Branch_Misses = Branch_Miss_Delta(Compare(
		Measurement(reference.Branch_Misses),
		Candidate_Measurement(candidate.Branch_Misses)))
	deltas.CPU_User = User_Time_Delta(Compare(
		Measurement(reference.CPU_User), Candidate_Measurement(candidate.CPU_User)))
	deltas.CPU_System = System_Time_Delta(Compare(
		Measurement(reference.CPU_System), Candidate_Measurement(candidate.CPU_System)))
	return deltas
}

// Sample_wall reads a sample's wall time, a non-negative monotonic span.
func sample_wall(sample Sample) (value Metric) {
	defer func() { Metric_Invariants(value, "sample_wall.value") }()
	Sample_Invariants(sample, "sample_wall.sample")
	return Metric(sample.Wall)
}

// Sample_rss reads a sample's peak resident size as a metric.
func sample_rss(sample Sample) (value Metric) {
	defer func() { Metric_Invariants(value, "sample_rss.value") }()
	Sample_Invariants(sample, "sample_rss.sample")
	return Metric(sample.RSS_Bytes_Max)
}

// Sample_cycles reads a sample's CPU cycle count as a metric.
func sample_cycles(sample Sample) (value Metric) {
	defer func() { Metric_Invariants(value, "sample_cycles.value") }()
	Sample_Invariants(sample, "sample_cycles.sample")
	return Metric(sample.CPU_Cycles)
}

// Sample_instructions reads a sample's retired-instruction count as a metric.
func sample_instructions(sample Sample) (value Metric) {
	defer func() { Metric_Invariants(value, "sample_instructions.value") }()
	Sample_Invariants(sample, "sample_instructions.sample")
	return Metric(sample.Instructions)
}

// Sample_cache_references reads a sample's cache-reference count as a metric.
func sample_cache_references(sample Sample) (value Metric) {
	defer func() { Metric_Invariants(value, "sample_cache_references.value") }()
	Sample_Invariants(sample, "sample_cache_references.sample")
	return Metric(sample.Cache_References)
}

// Sample_cache_misses reads a sample's cache-miss count as a metric.
func sample_cache_misses(sample Sample) (value Metric) {
	defer func() { Metric_Invariants(value, "sample_cache_misses.value") }()
	Sample_Invariants(sample, "sample_cache_misses.sample")
	return Metric(sample.Cache_Misses)
}

// Sample_branch_misses reads a sample's branch-miss count as a metric.
func sample_branch_misses(sample Sample) (value Metric) {
	defer func() { Metric_Invariants(value, "sample_branch_misses.value") }()
	Sample_Invariants(sample, "sample_branch_misses.sample")
	return Metric(sample.Branch_Misses)
}

// Sample_user reads a sample's user CPU time as a metric.
func sample_user(sample Sample) (value Metric) {
	defer func() { Metric_Invariants(value, "sample_user.value") }()
	Sample_Invariants(sample, "sample_user.sample")
	return Metric(sample.CPU_User)
}

// Sample_system reads a sample's system CPU time as a metric.
func sample_system(sample Sample) (value Metric) {
	defer func() { Metric_Invariants(value, "sample_system.value") }()
	Sample_Invariants(sample, "sample_system.sample")
	return Metric(sample.CPU_System)
}

// COMMAND_WORD_BYTES_MIN is the one-byte executable in the shortest parsed command.
const COMMAND_WORD_BYTES_MIN = 1

// COMMAND_WORD_BYTES_MAX caps one argv word: a few kilobytes is already an unreasonable
// single token, and a longer one is a probe to reject rather than benchmark.
const COMMAND_WORD_BYTES_MAX = 1 << 12

// Command_Word is one word of a benchmarked command — an environment assignment, the
// executable, or an argument. It is what the user typed at the shell, bounded short.
type Command_Word string

// Command_Word_Invariants bounds each nonempty word from the positional command syntax.
func Command_Word_Invariants(word Command_Word, namespace aver.Namespace) {
	aver.Tree(word, namespace).
		Range_Int(len(word), COMMAND_WORD_BYTES_MIN, COMMAND_WORD_BYTES_MAX).
		Ensure()
}

// INT64_MAX is the largest signed 64-bit value, used to test whether a 128-bit variance
// still fits a word before the fixed-point square root.
const INT64_MAX = 1<<63 - 1

// Wide is a 128-bit unsigned accumulator, for a sum of squares that overflows int64.
type Wide struct {
	// High is the upper 64 bits.
	High Accumulator_High
	// Low is the lower 64 bits.
	Low Accumulator_Low
}

// Wide_Invariants states the two halves of the 128-bit accumulator.
func Wide_Invariants(value Wide, namespace aver.Namespace) {
	Accumulator_High_Invariants(value.High, namespace)
	Accumulator_Low_Invariants(value.Low, namespace)
}

// Wide_Subtotal is a 128-bit accumulator before one more deviation is added.
type Wide_Subtotal struct {
	// High is the upper 64 bits of the partial sum.
	High Accumulator_Subtotal_High
	// Low is the lower 64 bits of the partial sum.
	Low Accumulator_Low
}

// Wide_Subtotal_Invariants states both words of the partial sum.
func Wide_Subtotal_Invariants(value Wide_Subtotal, namespace aver.Namespace) {
	Accumulator_Subtotal_High_Invariants(value.High, namespace)
	Accumulator_Low_Invariants(value.Low, namespace)
}

// Standard_deviation is the sample standard deviation with an n-1 denominator. The sum
// of squared deviations is accumulated in 128 bits so a large metric's deviations cannot
// overflow before the divide and the fixed-point root. Its result does not depend on order.
func standard_deviation(
	sorted Deviations, center Average, count Kept,
) (deviation Standard_Deviation) {
	defer func() { Standard_Deviation_Invariants(deviation, "standard_deviation.deviation") }()
	Deviations_Invariants(sorted, "standard_deviation.sorted")
	Average_Invariants(center, "standard_deviation.mean")
	Kept_Invariants(count, "standard_deviation.count")
	if count <= 1 {
		return 0
	}
	sum := Wide{}
	for _, value := range sorted {
		distance := value - int64(center)
		if distance < 0 {
			distance = -distance
		}
		sum = wide_add_square(Wide_Subtotal{
			High: Accumulator_Subtotal_High(sum.High), Low: sum.Low,
		}, Gap(uint64(distance)))
	}
	return root_of_quotient(&Root_Of_Quotient_Input{
		High: sum.High, Low: sum.Low, Denominator: Divisor(int(count) - 1),
	})
}

// Root_Of_Quotient_Input bundles a 128-bit numerator with the Bessel-corrected divisor — the
// sample count less one — that divides it before the root.
type Root_Of_Quotient_Input struct {
	// High is the numerator's upper 64 bits.
	High Accumulator_High
	// Low is the numerator's lower 64 bits.
	Low Accumulator_Low
	// Denominator is the sample count less one, the Bessel correction.
	Denominator Divisor
}

// Root_Of_Quotient_Input_Invariants states the numerator halves and the divisor.
func Root_Of_Quotient_Input_Invariants(
	input Root_Of_Quotient_Input, namespace aver.Namespace,
) {
	Accumulator_High_Invariants(input.High, namespace)
	Accumulator_Low_Invariants(input.Low, namespace)
	Divisor_Invariants(input.Denominator, namespace)
}

// Low_Quartile is the lower quartile in the raw metric domain.
type Low_Quartile int64

// Low_Quartile_Invariants bounds a lower quartile.
func Low_Quartile_Invariants(value Low_Quartile, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value),
			METRIC_MIN,
			METRIC_MAX,
		).
		Ensure()
}

// High_Quartile is the upper quartile in the raw metric domain.
type High_Quartile int64

// High_Quartile_Invariants bounds an upper quartile.
func High_Quartile_Invariants(value High_Quartile, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value),
			METRIC_MIN,
			METRIC_MAX,
		).
		Ensure()
}

// Outlier_Count_Input bundles the sorted values with the raw quartiles the fences derive from.
type Outlier_Count_Input struct {
	// Sorted is the ascending metric values.
	Sorted Points
	// Low_Quartile is the metric at q1; a value below q1 minus 1.5*IQR is an outlier.
	Low_Quartile Low_Quartile
	// High_Quartile is the metric at q3; a value above q3 plus 1.5*IQR is an outlier.
	High_Quartile High_Quartile
}

// Outlier_Count_Input_Invariants states the sorted values and the two quartiles they bracket.
func Outlier_Count_Input_Invariants(input Outlier_Count_Input, namespace aver.Namespace) {
	Points_Invariants(input.Sorted, namespace)
	Low_Quartile_Invariants(input.Low_Quartile, namespace)
	High_Quartile_Invariants(input.High_Quartile, namespace)
}

// Outlier_count counts the values beyond Tukey's fences — a point more than 1.5 interquartile
// ranges past a quartile. The whole test runs in the raw metric domain, doubled so the 3/2 is
// an integer 3 over a factored-out 2: a value is an outlier when 2*value falls outside
// [2*q1 - 3*IQR, 2*q3 + 3*IQR]. Every term stays within int64 for metrics up to the
// representable ceiling — 2*value and 2*q are at most 2^44, 3*IQR at most ~2^45 — where the
// fixed-point form, each quartile first lifted by the 2^20 scale to near 2^63, overflowed on a
// wide spread and produced garbage fences that miscounted the whole run as outliers.
func outlier_count(input Outlier_Count_Input_Handle) (count Strays) {
	defer func() { Strays_Invariants(count, "outlier_count.count") }()
	Outlier_Count_Input_Handle_Invariants(input, "outlier_count.input")
	low_quartile := int64(input.Low_Quartile)
	high_quartile := int64(input.High_Quartile)
	inter_quartile := high_quartile - low_quartile
	low_fence := 2*low_quartile - 3*inter_quartile
	high_fence := 2*high_quartile + 3*inter_quartile
	for _, value := range input.Sorted {
		doubled := 2 * int64(value)
		if doubled < low_fence {
			count++
			continue
		}
		if doubled > high_fence {
			count++
		}
	}
	return count
}

// Row_Measurement is a distribution a table row renders: a metric with data, thus a maximum of
// at least one whole unit and a mean of at least one unit over the sample cap.
type Row_Measurement Measurement

// Row_Measurement_Invariants states the rendered distribution fields.
func Row_Measurement_Invariants(value Row_Measurement, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value.Mean), MEAN_NONZERO_MIN, METRIC_SCALED_MAX).
		Range_Holed_Int64(
			int64(value.Standard_Deviation),
			METRIC_MIN,
			STANDARD_DEVIATION_SCALED_MAXIMUM,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Min), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Int64(int64(value.Max), fixedpoint.SCALE, METRIC_SCALED_MAX).
		Range_Holed_Int64(int64(value.Median), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Q1), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Q3), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Int(int(value.Outlier_Count), STRAYS_MIN, STRAYS_MAX).
		Range_Int(int(value.Sample_Count), KEPT_MIN, KEPT_MAX).
		Enum_Int(len(value.Unit), UNIT_BYTES_MIN, UNIT_BYTES_MAX).
		Ensure()
}

// Candidate_Measurement is the candidate side of one metric comparison.
type Candidate_Measurement Measurement

// Candidate_Measurement_Invariants states the candidate distribution fields.
func Candidate_Measurement_Invariants(
	value Candidate_Measurement, namespace aver.Namespace,
) {
	aver.Always(
		int64(value.Standard_Deviation)/fixedpoint.SCALE <= STANDARD_DEVIATION_MAXIMUM,
		"A candidate standard deviation cannot exceed the metric domain.",
	)
	aver.Tree(value, namespace).
		Range_Holed_Int64(int64(value.Mean), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(
			int64(value.Standard_Deviation),
			METRIC_MIN,
			STANDARD_DEVIATION_SCALED_MAXIMUM,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Min), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Max), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Median), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Q1), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Holed_Int64(int64(value.Q3), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Range_Int(int(value.Outlier_Count), STRAYS_MIN, STRAYS_MAX).
		Range_Int(int(value.Sample_Count), KEPT_MIN, KEPT_MAX).
		Enum_Int(len(value.Unit), UNIT_BYTES_MIN, UNIT_BYTES_MAX).
		Ensure()
}

// Compare reports how the candidate's mean differs from the reference's, with the
// 95% confidence half-interval from a pooled-variance two-sample t-test — poop's
// ratio computation. A zero or degenerate reference yields a zero, non-significant
// delta rather than a divide-by-zero.
func Compare(reference Measurement, candidate Candidate_Measurement) (delta Delta) {
	defer func() { Delta_Invariants(delta, "Compare.delta") }()
	Measurement_Invariants(reference, "Compare.reference")
	Candidate_Measurement_Invariants(candidate, "Compare.candidate")
	candidate_measurement := Measurement(candidate)
	if reference.Mean == 0 {
		return delta
	}
	ratio := fixedpoint.Divide(
		fixedpoint.Dividend(
			fixedpoint.Number(candidate_measurement.Mean)-
				fixedpoint.Number(reference.Mean),
		),
		fixedpoint.Divisor(fixedpoint.Number(reference.Mean)),
	)
	delta.Diff_Percent = Difference_Percent(ratio * 100)
	delta.Faster = candidate_measurement.Mean < reference.Mean

	degrees := candidate_measurement.Sample_Count + reference.Sample_Count - 2
	if degrees < 1 {
		return delta
	}
	delta.Half_Percent = half_interval(&Half_Interval_Input{
		Mean:                Reference_Mean(reference.Mean),
		Reference_Deviation: reference.Standard_Deviation,
		Candidate_Deviation: Candidate_Deviation(candidate.Standard_Deviation),
		Reference_Count:     reference.Sample_Count,
		Candidate_Count:     Candidate_Count(candidate.Sample_Count),
		Degrees:             Degree(degrees),
	})
	delta.Significant = significant(&Significant_Input{
		Diff_Percent: delta.Diff_Percent,
		Half_Percent: delta.Half_Percent,
	})
	return delta
}

// Half_Interval_Input is what a confidence half-width reads of two distributions whose
// reference mean is not zero.
type Half_Interval_Input struct {
	// Mean is the reference mean the deviations are taken relative to.
	Mean Reference_Mean
	// Reference_Deviation is the reference standard deviation.
	Reference_Deviation Standard_Deviation
	// Candidate_Deviation is the candidate standard deviation.
	Candidate_Deviation Candidate_Deviation
	// Reference_Count is the reference's kept runs.
	Reference_Count Kept
	// Candidate_Count is the candidate's kept runs.
	Candidate_Count Candidate_Count
	// Degrees is the pooled degrees of freedom.
	Degrees Degree
}

// Half_Interval_Input_Invariants states the two distributions.
func Half_Interval_Input_Invariants(value Half_Interval_Input, namespace aver.Namespace) {
	Reference_Mean_Invariants(value.Mean, namespace)
	Standard_Deviation_Invariants(value.Reference_Deviation, namespace)
	Candidate_Deviation_Invariants(value.Candidate_Deviation, namespace)
	Kept_Invariants(value.Reference_Count, namespace)
	Candidate_Count_Invariants(value.Candidate_Count, namespace)
	Degree_Invariants(value.Degrees, namespace)
}

// Half_Interval_Input_Handle borrows the distributions a half-width reads.
type Half_Interval_Input_Handle *Half_Interval_Input

// Half_Interval_Input_Handle_Invariants composes the borrowed distributions.
func Half_Interval_Input_Handle_Invariants(
	value Half_Interval_Input_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Half_Interval_Input_Invariants(*value, namespace)
}

// Reference_Mean is a reference mean a comparison divides by, thus never zero.
type Reference_Mean fixedpoint.Number

// Reference_Mean_Invariants bounds a mean from the smallest one a whole sample leaves.
func Reference_Mean_Invariants(value Reference_Mean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), MEAN_NONZERO_MIN, METRIC_SCALED_MAX).
		Ensure()
}

// MEAN_NONZERO_MIN is the smallest mean that is not zero: one unit over the sample cap.
const MEAN_NONZERO_MIN = fixedpoint.SCALE / SAMPLES_MAX

// Candidate_Deviation is a candidate's standard deviation.
type Candidate_Deviation fixedpoint.Number

// Candidate_Deviation_Invariants bounds a deviation to the metric domain.
func Candidate_Deviation_Invariants(value Candidate_Deviation, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int64(int64(value), METRIC_MIN, STANDARD_DEVIATION_SCALED_MAXIMUM,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Ensure()
}

// Candidate_Count is a candidate's kept-run count.
type Candidate_Count int

// Candidate_Count_Invariants bounds the count to the quorum range.
func Candidate_Count_Invariants(value Candidate_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), KEPT_MIN, KEPT_MAX).
		Ensure()
}

// Half_interval is the 95% confidence half-width on Diff_Percent, from a pooled-variance
// two-sample t-test — poop's score*pooled*normalizer*100/mean. The pooled deviation is
// taken relative to the reference mean, folding in that final divide, so the math never
// forms a raw variance — which, for a metric in the billions, overflows. Its squares wrap as
// the signed word wraps, and a negative pool roots to zero.
func half_interval(input Half_Interval_Input_Handle) (half Half_Percent) {
	defer func() { Half_Percent_Invariants(half, "half_interval.half") }()
	Half_Interval_Input_Handle_Invariants(input, "half_interval.input")
	first := fixedpoint.From_Ratio(1, fixedpoint.Denominator(input.Candidate_Count))
	second := fixedpoint.From_Ratio(1, fixedpoint.Denominator(input.Reference_Count))
	normalizer := fixedpoint.Number(fixedpoint.Square_Root(first + second))
	mean := fixedpoint.Number(input.Mean)
	candidate_deviation := fixedpoint.Divide(
		fixedpoint.Dividend(input.Candidate_Deviation), fixedpoint.Divisor(mean),
	)
	reference_deviation := fixedpoint.Divide(
		fixedpoint.Dividend(input.Reference_Deviation), fixedpoint.Divisor(mean),
	)
	candidate_variance := fixedpoint.Multiply(
		fixedpoint.Multiplicand(candidate_deviation),
		fixedpoint.Multiplier(candidate_deviation),
	)
	reference_variance := fixedpoint.Multiply(
		fixedpoint.Multiplicand(reference_deviation),
		fixedpoint.Multiplier(reference_deviation),
	)
	weighted := candidate_variance*fixedpoint.Number(input.Candidate_Count-1) +
		reference_variance*fixedpoint.Number(input.Reference_Count-1)
	pooled := fixedpoint.Square_Root(weighted / fixedpoint.Number(int(input.Degrees)))
	score := student_t_score(input.Degrees)
	band := fixedpoint.Multiply(
		fixedpoint.Multiplicand(score), fixedpoint.Multiplier(pooled),
	)
	band = fixedpoint.Multiply(
		fixedpoint.Multiplicand(band), fixedpoint.Multiplier(normalizer),
	)
	return Half_Percent(band * 100)
}

// Student_t_score returns the Student-t critical value for 95% confidence at the given
// degrees of freedom as a fixed-point number, falling back to the normal-distribution
// 1.96 past the tabulated range — poop's getStatScore95. The tables hold thousandths so
// they read as the published constants, and From_Ratio puts them on the fixed-point grid.
// The tables are local, not package globals, so the package keeps no mutable state.
func student_t_score(degrees_of_freedom Degree) (score Student_T_Score) {
	defer func() { Student_T_Score_Invariants(score, "student_t_score.score") }()
	Degree_Invariants(degrees_of_freedom, "student_t_score.degrees_of_freedom")
	freedom := int(degrees_of_freedom)
	table_1to30 := []int64{
		12706, 4303, 3182, 2776, 2571, 2447, 2365, 2306, 2262, 2228,
		2201, 2179, 2160, 2145, 2131, 2120, 2110, 2101, 2093, 2086,
		2080, 2074, 2069, 2064, 2060, 2056, 2052, 2045, 2048, 2042,
	}
	table_10s := []int64{
		2228, 2086, 2042, 2021, 2009, 2000, 1994, 1990, 1987, 1984, 1982, 1980,
	}
	milli := int64(1960)
	if freedom >= 1 {
		if freedom <= 30 {
			milli = table_1to30[freedom-1]
		}
	}
	if freedom > 30 {
		if freedom <= 120 {
			milli = table_10s[freedom/10-1]
		}
	}
	return Student_T_Score(fixedpoint.From_Ratio(fixedpoint.Numerator(milli), 1000))
}

// Significant_Input carries the difference and its confidence half-interval, both as
// fixed-point percentages.
type Significant_Input struct {
	// Diff_Percent is the signed percentage difference under test.
	Diff_Percent Difference_Percent
	// Half_Percent is the confidence interval's half-width, as a percentage.
	Half_Percent Half_Percent
}

// Significant_Input_Invariants states the one property the fields carry that
// fixedpoint.Number cannot: the half-interval is a confidence half-width, so it is
// never negative. The difference may have either sign and is left unconstrained.
func Significant_Input_Invariants(input Significant_Input, namespace aver.Namespace) {
	Difference_Percent_Invariants(input.Diff_Percent, namespace)
	Half_Percent_Invariants(input.Half_Percent, namespace)
	aver.Always(input.Half_Percent >= 0, "A confidence half-interval is never negative.")
}

// Significant decides whether a difference clears poop's ±1% band: the whole
// confidence interval must sit beyond ±1% with a single sign. The && and || of
// poop's check are split into nested single-term ifs to satisfy the linter.
func significant(input Significant_Input_Handle) (is Significance) {
	defer func() { Significance_Invariants(is, "significant.is") }()
	Significant_Input_Handle_Invariants(input, "significant.input")
	difference := fixedpoint.Number(input.Diff_Percent)
	half := fixedpoint.Number(input.Half_Percent)
	one := fixedpoint.Number(fixedpoint.From_Integer(1))
	if difference >= one {
		if difference-half >= one {
			return true
		}
	}
	negative_one := fixedpoint.Number(fixedpoint.From_Integer(-1))
	if difference <= negative_one {
		if difference+half <= negative_one {
			return true
		}
	}
	return false
}

// ANSI_CODE_BYTES_MIN is the shortest an ansi_code in use is, in bytes — the
// two-attribute resets like "\x1b[0m"; the empty string is never one of the codes.
const ANSI_CODE_BYTES_MIN = 4

// ANSI_CODE_BYTES_MAX is the longest an ansi_code can be, in bytes — the four-digit
// SGR sequences like "\x1b[92m".
const ANSI_CODE_BYTES_MAX = 5

// Ansi_Code is an ANSI escape sequence; a distinct type so paint can take it
// without colliding with its string text argument under the same-type-param rule.
type Ansi_Code string

// Ansi_Code_Invariants holds an ansi_code's byte length to the two lengths an SGR code
// has, and witnesses each length. A one-digit code is four bytes and a two-digit code is
// five, so the domain has no interior value that a span could claim.
func Ansi_Code_Invariants(code Ansi_Code, namespace aver.Namespace) {
	aver.Tree(code, namespace).
		Enum_Int(len(code), ANSI_CODE_BYTES_MIN, ANSI_CODE_BYTES_MAX).
		Ensure()
}

// ANSI_RESET clears all set attributes.
const ANSI_RESET Ansi_Code = "\x1b[0m"

// ANSI_FAINT sets faint, for an insignificant delta.
const ANSI_FAINT Ansi_Code = "\x1b[2m"

// ANSI_BRIGHT_GREEN sets bright green, for a significant speedup.
const ANSI_BRIGHT_GREEN Ansi_Code = "\x1b[92m"

// ANSI_BRIGHT_RED sets bright red, for a significant slowdown.
const ANSI_BRIGHT_RED Ansi_Code = "\x1b[91m"

// Machine_specs_cores renders the core topology line, distinguishing P/E cores on
// CORES_MIN is the single-digit core count a layout floors at, like "0".
const CORES_MIN = 1

// CORES_MAX bounds the core-layout string: the widest hybrid form "1023 P + 1023 E = 1023
// logical" with every count at the render-safe core ceiling, thirty bytes.
const CORES_MAX = 30

// Cores_Line is the rendered core-layout line — "8", "4 P + 4 E = 8 logical". A distinct type
// from a cell: it is wider than a formatted value, so it carries its own length range.
type Cores_Line string

// Cores_Line_Invariants bounds the core-layout length; it is never empty, with the single-digit
// min, the widest hybrid max, and the one- and two-byte shapes between witnessed.
func Cores_Line_Invariants(text Cores_Line, namespace aver.Namespace) {
	aver.Tree(text, namespace).Range_Int(len(text), CORES_MIN, CORES_MAX).Ensure()
}

// Core_Topology is the processor core data needed by the topology renderer.
type Core_Topology struct {
	// Physical is the physical core count.
	Physical Physical_Core_Count
	// Logical is the visible hardware-thread count.
	Logical Logical_Core_Count
	// Performance is the performance-core count.
	Performance Performance_Core_Count
	// Efficiency is the efficiency-core count.
	Efficiency Efficiency_Core_Count
}

// Core_Topology_Invariants states each core count.
func Core_Topology_Invariants(value Core_Topology, namespace aver.Namespace) {
	Physical_Core_Count_Invariants(value.Physical, namespace)
	Logical_Core_Count_Invariants(value.Logical, namespace)
	Performance_Core_Count_Invariants(value.Performance, namespace)
	Efficiency_Core_Count_Invariants(value.Efficiency, namespace)
}

// ELAPSED_DISPLAY_MAX caps the total sampling time the table header renders. Kiloseconds is the
// widest rung_index the time ladder reaches, and a span past this scales there to five digits, past
// the header's glyph. The sum of ten thousand ceiling-valued run walls runs to years — the header
// pins it to the widest renderable span while the JSON keeps the exact nanoseconds. The bound is
// four kilosecond digits: 9999 trillion nanoseconds.
const ELAPSED_DISPLAY_MAX = ELAPSED_DISPLAY_KILOSECONDS * 1_000_000_000_000

// ELAPSED_DISPLAY_KILOSECONDS is the widest span the header renders, in kiloseconds.
const ELAPSED_DISPLAY_KILOSECONDS = 9999

// SPAN_BYTES_MIN is the shortest rendered span: the one-byte "0" for no elapsed time.
const SPAN_BYTES_MIN = 1

// SPAN_BYTES_MAX bounds a rendered span's length: a four-byte glyph and a two-byte time suffix,
// the widest the header's clamped duration reaches.
const SPAN_BYTES_MAX = 6

// Span is a rendered sampling duration in the benchmark header — "5ns", "9ks", "26.4ks". A
// distinct type from a table cell: the time suffixes stop two bytes short of the seven-byte
// byte-size cell, so a span carries its own, narrower length invariant.
type Span string

// Span_Invariants bounds a rendered span's length, witnessing the one-byte floor, the two-byte
// single-digit-seconds shape, and the widest glyph-and-suffix form.
func Span_Invariants(text Span, namespace aver.Namespace) {
	aver.Tree(text, namespace).
		Range_Int(len(text), SPAN_BYTES_MIN, SPAN_BYTES_MAX).
		Ensure()
}

// COLUMN_NAME_WIDTH is the metric-name column width.
const COLUMN_NAME_WIDTH Extent = 12

// COLUMN_VALUE_WIDTH is the width of each scaled-quantity column.
const COLUMN_VALUE_WIDTH Extent = 8

// COLUMN_OUTLIERS_WIDTH is the outlier-count column width.
const COLUMN_OUTLIERS_WIDTH Extent = 9

// Delta_Presence records whether a table row includes its comparison column.
type Delta_Presence bool

// Delta_Presence_Invariants witnesses both table-row forms.
func Delta_Presence_Invariants(value Delta_Presence, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The row includes a delta.").
		Ensure()
}

// Metric_Line_Input carries everything one metric row is rendered from.
type Metric_Line_Input struct {
	// Name is the metric's row label.
	Name Caption
	// Measurement is the metric's distribution summary.
	Measurement Row_Measurement
	// Delta is the signed change against the reference.
	Delta Delta
	// Has_Delta is false for the reference row, which carries no delta.
	Has_Delta Delta_Presence
	// Color is whether the delta column is ANSI-colored.
	Color Color
}

// Metric_Line_Input_Invariants states the row's name, distribution, delta, and flags.
func Metric_Line_Input_Invariants(input Metric_Line_Input, namespace aver.Namespace) {
	Caption_Invariants(input.Name, namespace)
	Row_Measurement_Invariants(input.Measurement, namespace)
	Delta_Invariants(input.Delta, namespace)
	Delta_Presence_Invariants(input.Has_Delta, namespace)
	Color_Invariants(input.Color, namespace)
}

// Mean_Cell is one rendered mean value.
type Mean_Cell string

// Mean_Cell_Invariants bounds a rendered mean value.
func Mean_Cell_Invariants(value Mean_Cell, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), CELL_BYTES_MIN, CELL_BYTES_MAX).
		Ensure()
}

// Deviation_Cell is one rendered standard-deviation value.
type Deviation_Cell string

// Deviation_Cell_Invariants bounds a rendered standard-deviation value.
func Deviation_Cell_Invariants(value Deviation_Cell, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), CELL_BYTES_MIN, CELL_BYTES_MAX).
		Ensure()
}

// Minimum_Cell is one rendered minimum value.
type Minimum_Cell string

// Minimum_Cell_Invariants bounds a rendered minimum value.
func Minimum_Cell_Invariants(value Minimum_Cell, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), CELL_BYTES_MIN, CELL_BYTES_MAX).
		Ensure()
}

// Maximum_Cell is one rendered maximum value.
type Maximum_Cell string

// Maximum_Cell_Invariants bounds a rendered maximum value.
func Maximum_Cell_Invariants(value Maximum_Cell, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), CELL_BYTES_MIN, CELL_BYTES_MAX).
		Ensure()
}

// Render_Cells_Input is one table row's cell texts.
type Render_Cells_Input struct {
	// Name is the row label.
	Name Caption
	// Mean is the mean cell.
	Mean Mean_Cell
	// Sigma is the standard-deviation cell.
	Sigma Deviation_Cell
	// Low is the minimum cell.
	Low Minimum_Cell
	// High is the maximum cell.
	High Maximum_Cell
	// Outliers is the outlier-count cell.
	Outliers Outliers
}

// Render_Cells_Input_Invariants states every cell text of one row.
func Render_Cells_Input_Invariants(input Render_Cells_Input, namespace aver.Namespace) {
	Caption_Invariants(input.Name, namespace)
	Mean_Cell_Invariants(input.Mean, namespace)
	Deviation_Cell_Invariants(input.Sigma, namespace)
	Minimum_Cell_Invariants(input.Low, namespace)
	Maximum_Cell_Invariants(input.High, namespace)
	Outliers_Invariants(input.Outliers, namespace)
}

// Render_cells lays one row — header or data — into aligned, uncolored columns.
// Header and data share this layout, so a label always sits above its column.
func render_cells(input Render_Cells_Input_Handle) (text Bare_Row) {
	defer func() { Bare_Row_Invariants(text, "render_cells.line") }()
	Render_Cells_Input_Handle_Invariants(input, "render_cells.input")
	return Bare_Row("  " +
		string(pad(Column(input.Name), COLUMN_NAME_WIDTH, false)) + " " +
		string(pad(Column(input.Mean), COLUMN_VALUE_WIDTH, true)) + " ± " +
		string(pad(Column(input.Sigma), COLUMN_VALUE_WIDTH, false)) + "  " +
		string(pad(Column(input.Low), COLUMN_VALUE_WIDTH, true)) + " ... " +
		string(pad(Column(input.High), COLUMN_VALUE_WIDTH, false)) + "  " +
		string(pad(Column(input.Outliers), COLUMN_OUTLIERS_WIDTH, true)))
}

// CELL_BYTES_MIN is the shortest cell: a single digit like "0".
const CELL_BYTES_MIN = 1

// CELL_BYTES_MAX bounds a cell's length: the widest scaled quantity the table holds — three
// significant figures and sign over a four-byte ceiling, plus a three-byte unit suffix.
const CELL_BYTES_MAX = 7

// Cell is one rendered, scaled metric value placed in a table column — a formatted number
// with its unit suffix. A distinct type so its trusted, machine-built text carries a length
// invariant, not the untrusted-string content preset whose adversarial axes it can never
// exhibit. Narrower roles — the bare significant figures, a byte size, a padded column, a
// core layout — carry their own width range as glyph, quantum, column, and cores.
type Cell string

// Cell_Invariants bounds a cell's length: never empty, the single-byte min and the widest
// max witnessed alongside the two-byte shape.
func Cell_Invariants(text Cell, namespace aver.Namespace) {
	aver.Tree(text, namespace).
		Range_Int(len(text), CELL_BYTES_MIN, CELL_BYTES_MAX).
		Ensure()
}

// GLYPH_MIN is the single digit a formatted figure floors at, like "0".
const GLYPH_MIN = 1

// GLYPH_TWO_BYTES is the two-digit figure, like "50".
const GLYPH_TWO_BYTES = 2

// GLYPH_THREE_BYTES is the three-digit figure like "500", and equally the two-digit figure
// over a decimal point like "9.9". The two shapes occupy the same three bytes.
const GLYPH_THREE_BYTES = 3

// GLYPH_MAX bounds the bare significant figures before a suffix: a sign over three figures
// with a decimal point.
const GLYPH_MAX = 4

// Glyph is the bare significant-figure text a scale produces before its unit suffix — "0",
// "9.99". A distinct type, narrower than a whole cell, since the suffix is appended after.
type Glyph string

// Glyph_Invariants holds the figure length to the four lengths a figure has, and witnesses
// each length. A figure is never empty, and it spans the single digit through the widest
// form, so the two interior lengths are members in their own right.
func Glyph_Invariants(text Glyph, namespace aver.Namespace) {
	aver.Tree(text, namespace).
		Enum_4_Int(len(text), GLYPH_MIN, GLYPH_TWO_BYTES, GLYPH_THREE_BYTES, GLYPH_MAX).
		Ensure()
}

// COLUMN_MIN is the single byte a padded column floors at before alignment.
const COLUMN_MIN = 1

// COLUMN_MAX bounds the unpadded text a column pads: a metric name, an outlier tally, or a
// scaled value, the widest of which is the sixteen-byte longest metric name.
const COLUMN_MAX = 16

// Column is the text handed to the padder for one table column — a value, a name, or an
// outlier count, before it is widened to the column. A distinct type covering that union,
// wider than a single scaled cell.
type Column string

// Column_Invariants bounds the column-text length; never empty, with the single-byte min,
// the widest text max, and the one- and two-byte shapes between witnessed.
func Column_Invariants(text Column, namespace aver.Namespace) {
	aver.Tree(text, namespace).Range_Int(len(text), COLUMN_MIN, COLUMN_MAX).Ensure()
}

// CAPTION_MIN is the shortest metric or column name, like "peak_rss".
const CAPTION_MIN = 8

// CAPTION_MAX bounds a metric or column name, the widest being "cache_references".
const CAPTION_MAX = 16

// Caption is a metric-row or column-header name. A distinct type from a whole line: a name is
// a narrow fixed band, never the empty-to-row span a line covers.
type Caption string

// Caption_Invariants bounds a caption's length; always several bytes, so the empty, one, and
// two-byte boundaries are unreachable while the min and max are witnessed.
func Caption_Invariants(text Caption, namespace aver.Namespace) {
	aver.Tree(text, namespace).Range_Int(len(text), CAPTION_MIN, CAPTION_MAX).Ensure()
}

// OUTLIERS_MIN is the shortest outlier tally, the six-byte "0 (0%)".
const OUTLIERS_MIN = 6

// OUTLIERS_MAX bounds the outlier tally: the four-digit strays ceiling with its two-digit
// percentage, "4999 (49%)", ten bytes — an outlier count never exceeds half its samples.
const OUTLIERS_MAX = 10

// Outliers is the rendered outlier count with its percentage — "0 (0%)", "3 (10%)". A
// distinct type from a line: a tally is always several bytes, never the full row a line is.
type Outliers string

// Outliers_Invariants bounds the tally's length; always several bytes, so the empty, one, and
// two-byte boundaries are unreachable while the min and max are witnessed.
func Outliers_Invariants(text Outliers, namespace aver.Namespace) {
	aver.Tree(text, namespace).Range_Int(len(text), OUTLIERS_MIN, OUTLIERS_MAX).Ensure()
}

// PADDED_MIN is the narrowest padded column, the four-wide delta interval.
const PADDED_MIN = 4

// PADDED_MAX bounds a padded column: the widest unpadded text a column ever holds, the
// sixteen-byte longest metric name, which already exceeds every column's pad width.
const PADDED_MAX = 16

// Padded is one column widened to its alignment width — the padder's output, between the
// narrowest delta column and the widest value. A distinct type from a whole row.
type Padded string

// Padded_Invariants bounds a padded column's length; always several bytes, so the empty, one,
// and two-byte boundaries are unreachable while the min and max are witnessed.
func Padded_Invariants(text Padded, namespace aver.Namespace) {
	aver.Tree(text, namespace).Range_Int(len(text), PADDED_MIN, PADDED_MAX).Ensure()
}

// BARE_ROW_MIN is the narrowest assembled data row, all columns at their minimum.
const BARE_ROW_MIN = 69

// BARE_ROW_MAX bounds an assembled data row before any delta column is appended: the widest
// name, four widest value cells, and the widest outlier tally, aligned and joined.
const BARE_ROW_MAX = 74

// Bare_Row is one assembled, aligned table row without its delta — the fixed columns joined.
// A distinct type from the full row, which carries the delta and so runs wider.
type Bare_Row string

// Bare_Row_Invariants bounds an assembled row's length; always many bytes, so the empty, one,
// and two-byte boundaries are unreachable while the min and max are witnessed.
func Bare_Row_Invariants(text Bare_Row, namespace aver.Namespace) {
	aver.Tree(text, namespace).Range_Int(len(text), BARE_ROW_MIN, BARE_ROW_MAX).Ensure()
}

// DELTA_BODY_MIN is the narrowest delta body, a sign over two padded percentages.
const DELTA_BODY_MIN = 16

// DELTA_BODY_MAX bounds the unpainted delta body: the change percentage and its confidence
// half-interval each pinned to the display cap ("9999999.0"). A reference mean near zero spreads
// the pooled relative deviation far past the cap, so the interval widens as far as the change.
const DELTA_BODY_MAX = len("+9999999.0% ± 9999999.0%")

// Delta_Body is the uncolored delta text — a sign, a percentage, its confidence interval —
// before any ANSI color wraps it. A distinct type from the painted form, which runs wider.
type Delta_Body string

// Delta_Body_Invariants bounds the body's length; always many bytes, so the empty, one, and
// two-byte boundaries are unreachable while the min and max are witnessed.
func Delta_Body_Invariants(text Delta_Body, namespace aver.Namespace) {
	aver.Tree(text, namespace).
		Range_Int(len(text), DELTA_BODY_MIN, DELTA_BODY_MAX).
		Ensure()
}

// DELTA_TEXT_MIN is the narrowest rendered delta, an uncolored body.
const DELTA_TEXT_MIN = 16

// DELTA_TEXT_MAX bounds the rendered delta: the widest body wrapped in a color code and its
// reset — the body's max plus the nine bytes of the color escape and reset sequence.
const DELTA_TEXT_MAX = DELTA_BODY_MAX + len(ANSI_BRIGHT_RED) + len(ANSI_RESET)

// Delta_Text is the rendered delta column as written — the body, optionally wrapped in an ANSI
// color and its reset. A distinct type, wider than the bare body by the escape sequences.
type Delta_Text string

// Delta_Text_Invariants bounds the rendered delta's length; always many bytes, so the empty,
// one, and two-byte boundaries are unreachable while the min and max are witnessed.
func Delta_Text_Invariants(text Delta_Text, namespace aver.Namespace) {
	aver.Tree(text, namespace).
		Range_Int(len(text), DELTA_TEXT_MIN, DELTA_TEXT_MAX).
		Ensure()
}

// FREQUENCY_BYTES_MIN is the shortest rendered frequency: the one-byte "?" placeholder.
const FREQUENCY_BYTES_MIN = 1

// FREQUENCY_BYTES_MAX bounds a rendered frequency's length, the widest "N.NN GHz" form.
const FREQUENCY_BYTES_MAX = 1 << 3

// Frequency is a rendered CPU frequency — "?", "3.60 GHz", "800 MHz". A distinct type:
// a frequency is one byte or at least five, never two, so it carries its own invariant.
type Frequency string

// Frequency_Invariants excludes the two-byte shape the formatter can never produce while Range
// keeps the placeholder floor, widest form, and ordinary interior lengths demanded.
func Frequency_Invariants(text Frequency, namespace aver.Namespace) {
	aver.Tree(text, namespace).
		Range_Holed_Int(
			len(text), FREQUENCY_BYTES_MIN, FREQUENCY_BYTES_MAX, 2, 2, 2, 2).
		Ensure()
}

// PHASE_BYTES_MIN is the empty sampling phase a phase word floors at.
const PHASE_BYTES_MIN = 0

// PHASE_BYTES_MAX is the longest a progress phase is: the six-byte "warmup".
const PHASE_BYTES_MAX = 6

// Phase is a progress-line phase word — "" while sampling, "warmup" while warming up. A
// distinct type so the phase word carries a length invariant.
type Phase string

// Phase_Invariants uses an Enum because only the empty sampling phase and the warmup word exist;
// saturation rejects every other width and demands both real states.
func Phase_Invariants(name Phase, namespace aver.Namespace) {
	aver.Tree(name, namespace).
		Enum_Int(len(name), PHASE_BYTES_MIN, PHASE_BYTES_MAX).
		Ensure()
}

// EXTENT_MIN is the narrowest column width the layout assembles: the four-wide delta
// half-interval column.
const EXTENT_MIN = 4

// EXTENT_MAX bounds a column width: the widest column the table lays out, the twelve-wide
// metric-name column.
const EXTENT_MAX = 12

// Extent is a fixed column width — always at least three, never zero, one, or two. A
// distinct type so the layout's trusted widths carry a deliberate range, not the full
// signed-integer span the preset would demand observing.
type Extent int

// Extent_Invariants bounds an extent; a column width is always at least three, so the
// zero, one, two, and negative boundaries are unreachable while the min and max witness.
func Extent_Invariants(value Extent, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), EXTENT_MIN, EXTENT_MAX).
		Ensure()
}

// KEPT_MIN is the quorum a kept distribution carries at least: the 3-run minimum.
const KEPT_MIN = QUORUM_MIN

// KEPT_MAX is the kept-run ceiling a distribution reaches.
const KEPT_MAX = SAMPLES_MAX

// Kept is the number of runs a computed distribution was reduced from — at least the
// quorum, up to the run cap. A distinct type from a census: a reduced distribution always
// carries a quorum, so the empty, one, and two counts are unreachable.
type Kept int

// Kept_Invariants bounds a kept-run count to the quorum range; the below-quorum counts are
// guarded away, and the quorum floor and the run-cap ceiling are witnessed.
func Kept_Invariants(value Kept, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), KEPT_MIN, KEPT_MAX).
		Ensure()
}

// DEGREE_MIN is the smallest pooled degrees-of-freedom a real comparison reaches: two
// quorum distributions less two. Both commands carry at least the 3-run quorum, so the
// pooled n1+n2-2 never dips below four — the one, two, and three shapes are unreachable.
const DEGREE_MIN = 2*QUORUM_MIN - 2

// DEGREE_MAX bounds a pooled degrees-of-freedom: two full kept runs less two, the most the
// two-sample t-test reaches when both commands fill the run cap.
const DEGREE_MAX = 2*SAMPLES_MAX - 2

// Degree is a Student-t pooled degrees-of-freedom — the two sample counts less two, each at
// least the quorum, so it is at least four. A distinct type carrying that range; single
// sample counts travel as a tally, not this.
type Degree int

// Degree_Invariants bounds a degree; a degree is at least one, so the zero and negative
// boundaries are unreachable while the one min, the two shape, and the max are witnessed.
func Degree_Invariants(value Degree, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), DEGREE_MIN, DEGREE_MAX).
		Ensure()
}

// CENSUS_MIN is the single sample a reduced distribution carries at least.
const CENSUS_MIN = 1

// CENSUS_MAX bounds a sample or run count: the kept-run ceiling.
const CENSUS_MAX = SAMPLES_MAX

// Census is a count of samples or runs — at least one, never zero or negative, up to the run
// cap. A distinct type for the single-distribution count, separate from the pooled degree.
type Census int

// Census_Invariants bounds a census; it is at least one, so the zero and negative boundaries
// are unreachable while the one min, the two shape, and the cap max are witnessed.
func Census_Invariants(value Census, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), CENSUS_MIN, CENSUS_MAX).
		Ensure()
}

// DIVISOR_MIN is the Bessel divisor a quorum floors at: the 3-run minimum less one. A
// distribution always carries the quorum, so the divisor never drops below two.
const DIVISOR_MIN = QUORUM_MIN - 1

// DIVISOR_MAX bounds the sample-variance divisor: a full kept run less one.
const DIVISOR_MAX = SAMPLES_MAX - 1

// Divisor is the count-less-one denominator of the sample variance — at least one, up to a
// full run less one. A distinct type so the off-by-one ceiling is its own witnessed bound.
type Divisor int

// Divisor_Invariants bounds a divisor; it is at least one, so the zero and negative
// boundaries are unreachable while the one min, the two shape, and the max are witnessed.
func Divisor_Invariants(value Divisor, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), DIVISOR_MIN, DIVISOR_MAX).
		Ensure()
}

// TALLY_MIN is the empty count or first index a tally floors at: zero.
const TALLY_MIN = 0

// TALLY_MAX bounds a tally: the kept-run ceiling a count can reach.
const TALLY_MAX = SAMPLES_MAX

// Tally is a non-negative count or index — runs kept, outliers found, a benchmark's
// position. A distinct type bounding it to the non-negative range it lives in.
type Tally int

// Tally_Invariants bounds a tally; it is never negative, with the zero min, the kept-run
// max, and the one- and two-count shapes between witnessed.
func Tally_Invariants(value Tally, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), TALLY_MIN, TALLY_MAX).
		Ensure()
}

// STRAYS_MIN is the empty outlier count a clean distribution floors at: zero.
const STRAYS_MIN = 0

// STRAYS_MAX bounds the outlier count: the middle half of the sorted run pins the quartiles,
// so at most the outer half less one point can fall beyond the fences.
const STRAYS_MAX = SAMPLES_MAX/2 - 1

// Strays is the count of outliers a Tukey scan finds — non-negative, bounded by the outer
// half of the run. A distinct type from a tally: an outlier count never fills the whole run.
type Strays int

// Strays_Invariants bounds the outlier count; never negative, with the zero min, the outer-half
// max, and the one- and two-count shapes between witnessed.
func Strays_Invariants(value Strays, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), STRAYS_MIN, STRAYS_MAX).
		Ensure()
}

// Index_min is the first position a zero-based index floors at: zero.
const POSITION_MIN = 0

// Index_max bounds a benchmark or command index: the last position in a full report, one
// less than the command-set cap.
const POSITION_MAX = COMMAND_SET_MAX - 1

// Position is a zero-based index into the benchmarks or commands — a distinct type from a
// tally, bounded by the structural cap rather than the run cap.
type Position int

// Position_Invariants bounds a position; it is never negative, with the zero min, the last
// slot max, and the one- and two-position shapes between witnessed.
func Position_Invariants(value Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), POSITION_MIN, POSITION_MAX).
		Ensure()
}

// EXIT_CODE_MIN is the success code zero a code floors at.
const EXIT_CODE_MIN = 0

// EXIT_CODE_MAX is the usage code that distinguishes invalid arguments from failures.
const EXIT_CODE_MAX = 2

// Exit_Code is the process exit code for success, failure, or invalid usage.
type Exit_Code int

// Exit_Code_Invariants holds an exit code to the three declared states.
func Exit_Code_Invariants(value Exit_Code, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Int(int(value), EXIT_CODE_MIN, int(EXIT_FAILURE), EXIT_CODE_MAX).
		Ensure()
}

// Benchmark_Exit_Code is the success or failure result after argument parsing.
type Benchmark_Exit_Code Exit_Code

// Benchmark_Exit_Code_Invariants holds benchmark work to success or failure.
func Benchmark_Exit_Code_Invariants(
	value Benchmark_Exit_Code, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Int(int(value), int(EXIT_SUCCESS), int(EXIT_FAILURE)).
		Ensure()
}

// EXIT_STATUS_MIN is the success code a command's exit floors at: zero.
const EXIT_STATUS_MIN = 0

// EXIT_STATUS_MAX is the largest a command's exit reaches: a POSIX wait status is one
// byte, so 255 is the ceiling — distinct from the binary's own two-valued exit_code.
const EXIT_STATUS_MAX = 255

// Exit_Status is the exit code of a benchmarked command, as the byte a POSIX wait status
// carries. A distinct type from the binary's exit_code: a child may exit any of 256 codes,
// where maddox itself returns only success or failure.
type Exit_Status int

// Exit_Status_Invariants bounds a command's exit to the byte range and witnesses the small
// codes and the extremes; a negative code is guarded away, never a child's real status.
func Exit_Status_Invariants(value Exit_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), EXIT_STATUS_MIN, EXIT_STATUS_MAX).
		Ensure()
}

// FAILURE_STATUS_MIN is the smallest a failing command's exit is: one. A failure is
// reported only for a non-zero exit, so its status never reaches zero.
const FAILURE_STATUS_MIN = 1

// FAILURE_STATUS_MAX is the largest a failing command's exit reaches: the byte ceiling.
const FAILURE_STATUS_MAX = 255

// Failure_Status is the exit of a command that failed — always non-zero, up to the byte a
// POSIX wait status carries. A distinct type from Exit_Status: the failure diagnostic is
// reported only on a non-zero exit, so its status is never success.
type Failure_Status int

// Failure_Status_Invariants bounds a failing command's exit to the non-zero byte range and
// witnesses the small codes and the ceiling; success and negatives are guarded away.
func Failure_Status_Invariants(value Failure_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), FAILURE_STATUS_MIN, FAILURE_STATUS_MAX).
		Ensure()
}

// CORES_COUNT_MIN is the zero a core count floors at: a CPU without a hybrid split
// reports no performance or efficiency cores.
const CORES_COUNT_MIN = 0

// CORES_COUNT_MAX bounds a core count to a realistic, render-safe topology: even the
// widest server fits, and the core-layout line stays within its length.
const CORES_COUNT_MAX = 1023

// Cores is a CPU core count — physical, logical, or a hybrid performance/efficiency
// tier. A distinct type bounded to a realistic topology, so its bundle witnesses the
// small counts and the ceiling a real machine reaches, not a signed word's extremes.
type Cores int

// Cores_Invariants bounds a core count to the realistic range; never negative, with the
// zero, one, two, and ceiling shapes witnessed.
func Cores_Invariants(value Cores, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), CORES_COUNT_MIN, CORES_COUNT_MAX).
		Ensure()
}

// HERTZ_MIN is the zero an unknown frequency reports: the kernel exposed no maximum.
const HERTZ_MIN = 0

// HERTZ_MAX bounds a CPU frequency to a realistic, render-safe ceiling: 8 GHz renders as
// a short "8.00 GHz", within the frequency field's length.
const HERTZ_MAX = 8_000_000_000

// Hertz is a CPU frequency in cycles per second. A distinct type bounded to a realistic
// ceiling so its bundle witnesses zero (unknown), the small values, and the ceiling a real
// machine reports, never a word's unreachable extreme.
type Hertz uint64

// Hertz_Invariants bounds a frequency to the realistic range; the zero (unknown), the one
// and two shapes, and the ceiling are witnessed.
func Hertz_Invariants(value Hertz, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), HERTZ_MIN, HERTZ_MAX).
		Ensure()
}

// BYTE_SIZE_MIN is the zero a byte size floors at: a cache or store the kernel did not
// report.
const BYTE_SIZE_MIN = 0

// BYTE_SIZE_MAX bounds a hardware byte size — a cache, memory, or storage capacity — to a
// realistic, render-safe ceiling of eight pebibytes.
const BYTE_SIZE_MAX = 1 << 53

// Byte_Size is a hardware capacity in bytes: a cache line, installed memory, or a store. A
// distinct type bounded to a realistic ceiling so its bundle witnesses zero (unreported),
// the small sizes, and the ceiling, never a word's unreachable extreme.
type Byte_Size uint64

// Byte_Size_Invariants bounds a byte size to the realistic range; the zero, one, two, and
// ceiling shapes are witnessed.
func Byte_Size_Invariants(value Byte_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), BYTE_SIZE_MIN, BYTE_SIZE_MAX).
		Ensure()
}

// GAP_MIN is the smallest a deviation magnitude is: zero, when a value equals the mean.
const GAP_MIN = 0

// GAP_MAX is the largest deviation magnitude a distribution reaches: a lone ceiling value among
// SAMPLES_MAX otherwise-zero samples pulls the mean only to METRIC_MAX/SAMPLES_MAX, so that
// value's distance from the mean is METRIC_MAX less that quotient. It is the maximum over every
// valid distribution — attained exactly at the sample ceiling — and a safe guard, since a
// shorter run lifts the subtracted mean further and so falls below it.
const GAP_MAX = METRIC_MAX - METRIC_MAX/SAMPLES_MAX

// Gap is the magnitude of one value's deviation from the mean, the unsigned the 128-bit
// accumulator squares. Bounded by the largest deviation a ceiling-valued sample reaches, so its
// bundle witnesses the small shapes and that attained extreme alike.
type Gap uint64

// Gap_Invariants bounds a gap. The zero floor and the ceiling — a lone extreme value's distance
// from the mean it barely shifts — are both reached, so it is an ordinary bounded range.
func Gap_Invariants(value Gap, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), GAP_MIN, GAP_MAX).
		Ensure()
}

// ACCUMULATOR_MIN is the smallest a 128-bit accumulator word is: zero, an empty sum.
const ACCUMULATOR_MIN = 0

// ACCUMULATOR_HIGH_MAX is the largest the high word reaches: the high half of the greatest sum
// of squared deviations. That sum is a ceiling-scale bimodal distribution — half the sample cap
// at zero, half at the metric ceiling — so every deviation is half the ceiling; a shorter or
// less-spread run yields a smaller sum, making this both the attained maximum and a safe guard.
const ACCUMULATOR_HIGH_MAX uint64 = SAMPLES_MAX / 2 *
	((METRIC_MAX-METRIC_MAX/2)*(METRIC_MAX-METRIC_MAX/2) + METRIC_MAX/2*(METRIC_MAX/2)) /
	(1 << 64)

// ACCUMULATOR_SUBTOTAL_HIGH_MAX is the greatest high word before the final value is added.
const ACCUMULATOR_SUBTOTAL_HIGH_MAX uint64 = (SAMPLES_MAX/2*
	((METRIC_MAX-METRIC_MAX/2)*(METRIC_MAX-METRIC_MAX/2)+METRIC_MAX/2*(METRIC_MAX/2)) -
	(METRIC_MAX/2)*(METRIC_MAX/2)) / (1 << 64)

// Accumulator_High is the upper 64-bit half of the 128-bit sum-of-squares accumulator — a small
// register value, bounded by the high half of the greatest sum a distribution reaches.
type Accumulator_High uint64

// Accumulator_High_Invariants bounds the high word to the high half of the greatest sum of
// squared deviations a distribution reaches.
func Accumulator_High_Invariants(high Accumulator_High, namespace aver.Namespace) {
	aver.Tree(high, namespace).
		Range_Uint64(uint64(high), ACCUMULATOR_MIN, ACCUMULATOR_HIGH_MAX).
		Ensure()
}

// Accumulator_Subtotal_High is the high word before one more squared deviation is added.
type Accumulator_Subtotal_High uint64

// Accumulator_Subtotal_High_Invariants bounds the high word of a partial sum.
func Accumulator_Subtotal_High_Invariants(
	high Accumulator_Subtotal_High, namespace aver.Namespace,
) {
	aver.Tree(high, namespace).
		Range_Uint64(uint64(high), ACCUMULATOR_MIN, ACCUMULATOR_SUBTOTAL_HIGH_MAX).
		Ensure()
}

// ACCUMULATOR_LOW_MAX is the low word's ceiling: the full word width. The low half is a modular
// residue of the running sum, so as the sum grows past 2^64 it ranges across the whole word.
const ACCUMULATOR_LOW_MAX uint64 = 1<<64 - 1

// Accumulator_Low is the lower 64-bit half of the 128-bit sum-of-squares accumulator — a
// full-width modular residue of the running sum.
type Accumulator_Low uint64

// Accumulator_Low_Invariants bounds the low word to the full word width it spans as a modular
// residue of the running sum.
func Accumulator_Low_Invariants(low Accumulator_Low, namespace aver.Namespace) {
	aver.Tree(low, namespace).
		Range_Uint64(uint64(low), ACCUMULATOR_MIN, ACCUMULATOR_LOW_MAX).
		Ensure()
}

// METRIC_MIN is the zero floor a measured metric sits at: a count or a monotonic span is
// never negative.
const METRIC_MIN = 0

// METRIC_MAX bounds a metric to the fixed-point representable ceiling. The statistics lift
// each value with From_Integer, a multiply by SCALE (2^20) that overflows int64 at 2^43, so a
// value the pipeline can reduce stays below this. It is the honest ceiling, not the signed
// word's max: a metric past it cannot be represented, so the sampler must not report one, and
// the guard surfaces such a value rather than letting the sum-of-squares silently overflow.
const METRIC_MAX = 1<<43 - 1

// Metric is one measured value the statistics reduce — a wall or CPU-time span in
// nanoseconds, a byte count, or a hardware counter — as the non-negative int64 they share. A
// distinct type bounded to the representable range: a benchmark metric is never negative and
// never larger than the fixed-point ceiling, so its bundle witnesses the small shapes and the
// representable max a real distribution reaches, not the signed word's unreachable extreme.
type Metric int64

// Metric_Invariants bounds a metric to the representable non-negative range; the negative
// boundary is guarded away, with the zero min, the representable max, and the one and two
// shapes witnessed.
func Metric_Invariants(value Metric, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value),
			METRIC_MIN,
			METRIC_MAX,
		).
		Ensure()
}

// AVERAGE_MIN is the zero floor an integer mean sits at: the metrics it averages are
// non-negative, so their mean is too — it never goes below zero.
const AVERAGE_MIN = 0

// AVERAGE_MAX bounds the integer mean to the largest value From_Integer can lift without
// overflowing — one short of 2^43, the fixed-point representable ceiling.
const AVERAGE_MAX = 1<<43 - 1

// Average is a distribution's integer mean as the deviation works in it, bounded to the
// fixed-point representable range the quartiles share. A distinct type so the mean does not
// travel as a bare int.
type Average int64

// Average_Invariants bounds the integer mean to the non-negative fixed-point range and
// witnesses the small values and the representable ceiling; the negative boundaries are
// guarded away, since a mean of non-negative metrics never goes below zero.
func Average_Invariants(value Average, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value),
			AVERAGE_MIN,
			AVERAGE_MAX,
		).
		Ensure()
}

// POINTS_MAX bounds a sorted run's length: the per-command sample cap, since the run is the
// kept samples sorted.
const POINTS_MAX = SAMPLES_MAX

// Points is an ascending run of metric values an outlier scan walks — the kept samples sorted,
// so always at least the 3-run quorum, never the empty, one, or two counts below it.
type Points []int64

// Points_Invariants bounds a sorted run's length. The run is a kept distribution's samples,
// floored at the 3-run quorum, so the quorum floor and the cap max are witnessed and the
// shorter counts are guarded out.
func Points_Invariants(values Points, namespace aver.Namespace) {
	aver.Tree(values, namespace).Range_Int(len(values), QUORUM_MIN, POINTS_MAX).Ensure()
}

// LABEL_BYTES_MIN is the one-byte command in the shortest progress label.
const LABEL_BYTES_MIN = 1

// LABEL_BYTES_MAX bounds a progress label: the command text is truncated to a rune cap, so
// at most that many runes survive, each at most four UTF-8 bytes.
const LABEL_BYTES_MAX = PROGRESS_LABEL_RUNES_MAX * 4

// Label is the truncated command text on a progress line — bounded to one terminal row,
// so a distinct type holds it to a length invariant, not the untrusted-string content
// preset whose megabyte axis a truncated label can never reach.
type Label string

// Label_Invariants bounds a nonempty progress label through its truncation ceiling.
func Label_Invariants(text Label, namespace aver.Namespace) {
	aver.Tree(text, namespace).
		Range_Int(len(text), LABEL_BYTES_MIN, LABEL_BYTES_MAX).
		Ensure()
}

// Right_Alignment records which side receives a column's padding.
type Right_Alignment bool

// Right_Alignment_Invariants witnesses both column alignments.
func Right_Alignment_Invariants(value Right_Alignment, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The column is right-aligned.").
		Ensure()
}

// PERCENT_DISPLAY_MAX caps a rendered percentage's magnitude at the seven-digit ceiling the
// delta column was sized against. Format at one decimal yields at most "9999999.0" — nine
// bytes — so a sign, two such values, and the "% ± " and "%" glue land exactly at
// DELTA_BODY_MAX; a larger percentage is pinned here rather than overrunning the fixed layout.
// The bound is written as an integer times the scale so it stays a compile-time constant.
const PERCENT_DISPLAY_MAX fixedpoint.Number = 9_999_999 * fixedpoint.SCALE

// Quantity is a scaled fixed-point value a cell renders to three significant figures. A value
// at or past one unit scales down its ladder to under a thousand of that unit, so the largest is
// the capped elapsed span at its top rung; a value under one unit is a fraction the scaling
// never reaches, and a fraction of whole samples never falls to one or two grid units.
type Quantity fixedpoint.Number

// Quantity_Invariants bounds a quantity to the widest a cell renders.
func Quantity_Invariants(value Quantity, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int64(int64(value), PERCENT_MINIMUM, QUANTITY_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Ensure()
}

// QUANTITY_MAX is the capped elapsed span on its kilosecond rung.
const QUANTITY_MAX = ELAPSED_DISPLAY_KILOSECONDS * fixedpoint.SCALE

// Magnitude is an unscaled distribution statistic on the fixed-point grid, as a row renders it.
type Magnitude fixedpoint.Number

// Magnitude_Invariants bounds a magnitude to the metric domain.
func Magnitude_Invariants(value Magnitude, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int64(int64(value), METRIC_MIN, METRIC_SCALED_MAX,
			scalar.FRACTION_UNIT_ONE, scalar.FRACTION_UNIT_TWO,
			scalar.FRACTION_UNIT_TWO, scalar.FRACTION_UNIT_TWO,
		).
		Ensure()
}

// Format_quantity renders a raw value scaled to a human unit with three significant figures —
// poop's printUnit, e.g. 14906807 nanoseconds becomes "14.9ms". The count ladder has no base
// rung: a count under a thousand divides by zero, which the fixed-point division reads as zero.
func format_quantity(value Magnitude, unit Unit) (text Cell) {
	defer func() { Cell_Invariants(text, "format_quantity.text") }()
	Magnitude_Invariants(value, "format_quantity.value")
	Unit_Invariants(unit, "format_quantity.unit")
	divisors := [LADDER_SIZE]int64{1_000_000_000_000, 1_000_000_000, 1_000_000, 1_000, 0}
	suffixes := [LADDER_SIZE]string{"T", "G", "M", "K", ""}
	switch unit {
	case UNIT_TIME:
		divisors[LADDER_SIZE-1] = 1
		suffixes = [LADDER_SIZE]string{"ks", "s", "ms", "us", "ns"}
	case "bytes":
		divisors = [LADDER_SIZE]int64{1 << 40, 1 << 30, 1 << 20, 1 << 10, 1}
		suffixes = [LADDER_SIZE]string{"TiB", "GiB", "MiB", "KiB", "B"}
	}
	for rung := range divisors {
		divisor := fixedpoint.From_Integer(fixedpoint.Whole_Integer(divisors[rung]))
		if int64(value) >= int64(divisor) {
			scaled := fixedpoint.Divide(
				fixedpoint.Dividend(value), fixedpoint.Divisor(divisor))
			return Cell(string(format_significant(Quantity(scaled))) + suffixes[rung])
		}
	}
	return Cell(format_significant(Quantity(value)))
}

// LADDER_SIZE is the rung count of every scale ladder.
const LADDER_SIZE = 5

// Paint wraps text in an ANSI color when color is enabled; the codes have zero
// visible width, so wrapping after padding leaves alignment intact.
func paint(text Delta_Body, code Ansi_Code, color Color) (painted Delta_Text) {
	defer func() { Delta_Text_Invariants(painted, "paint.painted") }()
	Delta_Body_Invariants(text, "paint.text")
	Ansi_Code_Invariants(code, "paint.code")
	Color_Invariants(color, "paint.color")
	if !color {
		return Delta_Text(text)
	}
	return Delta_Text(string(code) + string(text) + string(ANSI_RESET))
}

// PROGRESS_CLEAR returns to the start of the line and erases it, so the next progress
// update — or the report — overwrites the previous progress text cleanly.
const PROGRESS_CLEAR = "\r\x1b[K"

// PROGRESS_LABEL_RUNES_MAX caps the command text in the progress line so the
// carriage-return update never wraps and strands a stale partial line.
const PROGRESS_LABEL_RUNES_MAX = 50

// Progress_Total is the optional total count on one progress line.
type Progress_Total int64

// Progress_Total_Invariants accepts the complete signed 64-bit input domain.
func Progress_Total_Invariants(value Progress_Total, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value),
			LIMIT_MIN,
			LIMIT_MAX,
		).
		Ensure()
}

// Completion_Moment is the monotonic clock reading after one measured run.
type Completion_Moment time.Monotonic_Moment

// Completion_Moment_Invariants distinguishes an absent stamp from a measured stamp.
func Completion_Moment_Invariants(value Completion_Moment, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), int64(time.MONOTONIC_MOMENT_MINIMUM),
			int64(time.MONOTONIC_MOMENT_MAXIMUM)).
		Sometimes(value != 0, "A completion moment is set.").
		Ensure()
}

// Run_Result is everything one measured run reports: its Sample, the exit code, and the stderr
// captured for a failing run.
type Run_Result struct {
	// Sample is the run's measurements, including its own tight wall time.
	Sample Sample
	// Completed_At is the monotonic clock reading when this run was judged. It drives the time
	// budget as a real stopwatch — spanning the gaps between runs, which the reported per-run
	// wall does not.
	Completed_At Completion_Moment
	// Exit is the command's exit code; non-zero is a failure.
	Exit Exit_Status
	// Stderr is the command's captured stderr, surfaced on failure.
	Stderr Captured_Output
	// Spawn_Failure reports a run the loop could not start at all.
	Spawn_Failure Spawn_Failure
}

// Run_Result_Invariants states a Run_Result's fields.
func Run_Result_Invariants(result Run_Result, namespace aver.Namespace) {
	Sample_Invariants(result.Sample, namespace)
	Completion_Moment_Invariants(result.Completed_At, namespace)
	Exit_Status_Invariants(result.Exit, namespace)
	Captured_Output_Invariants(result.Stderr, namespace)
	Spawn_Failure_Invariants(result.Spawn_Failure, namespace)
}

// Sample_Selector reads one metric out of a sample.
type Sample_Selector func(sample Sample) (value Metric)

// Extract pulls one metric's value out of every sample as the int64 the statistics work in; every
// raw metric is already an integer count, so no float ever enters.
func extract(samples Distribution, selector Sample_Selector) (values Series) {
	defer func() { Series_Invariants(values, "extract.values") }()
	Distribution_Invariants(samples, "extract.samples")
	values = make(Series, len(samples))
	for index, sample := range samples {
		values[index] = int64(selector(sample))
	}
	return values
}

// Command_words flattens a command back to the words a reader recognizes: its own environment
// assignments, the executable, then its arguments. The first inherited entries of its
// environment are the process's, not the command's, thus they are left out.
func command_words(
	command nbio.Process_Request, inherited Inherited_Count,
) (words Command_Line) {
	defer func() { Command_Line_Invariants(words, "command_words.words") }()
	Inherited_Count_Invariants(inherited, "command_words.inherited")
	words = make(Command_Line, 0, COMMAND_WORDS_MAX)
	for _, assignment := range command.Environment[inherited:] {
		words = append(words, Command_Word(assignment))
	}
	words = append(words, Command_Word(command.Path))
	for _, argument := range command.Arguments {
		words = append(words, Command_Word(argument))
	}
	return words
}

// Inherited_Count is the process environment entries every command's environment starts with.
type Inherited_Count int

// Inherited_Count_Invariants bounds the inherited environment to a process environment.
func Inherited_Count_Invariants(value Inherited_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COLLECTION_MIN, ENVIRONMENT_COUNT_MAX).
		Ensure()
}

// ENVIRONMENT_COUNT_MAX bounds a process environment.
const ENVIRONMENT_COUNT_MAX = ARGUMENTS_COUNT_MAX

// Measurement_Compute reduces one metric's per-run values to its distribution. It sorts a copy
// (the caller's slice is left untouched), then takes the mean, the sample standard deviation (n-1
// denominator, matching poop), the extrema, the median, the quartiles by poop's index math, and
// the Tukey's-fences outlier count.
func Measurement_Compute(values Values, unit Unit) (measurement Measurement) {
	defer func() {
		Measurement_Invariants(measurement, "Measurement_Compute.measurement")
	}()
	Values_Invariants(values, "Measurement_Compute.values")
	for _, value := range values {
		Metric_Invariants(Metric(value), "Measurement_Compute.value")
	}
	Unit_Invariants(unit, "Measurement_Compute.unit")
	// The values are a kept distribution's quorum (Values_Invariants guards the empty, one, and
	// two counts away), so the statistics below always have a count to divide by.
	count := len(values)
	sorted := make(Points, count)
	copy(sorted, values)
	points_sort(sorted)

	total := int64(0)
	for _, value := range sorted {
		total += value
	}
	mean_integer := total / int64(count)

	// Quartiles by position, exactly as poop indexes them: q3 falls back to the maximum when
	// there are too few points to take the upper quarter. The raw metric at each quartile is
	// kept alongside its fixed-point lift: the outlier fences are computed from the raw
	// integers, where 1.5*IQR cannot overflow the way the scaled form does.
	low_quartile := sorted[count/4]
	high_quartile := sorted[count-1]
	if count >= 4 {
		high_quartile = sorted[count-count/4]
	}
	q1 := fixedpoint.Number(fixedpoint.From_Integer(fixedpoint.Whole_Integer(low_quartile)))
	q3 := fixedpoint.Number(fixedpoint.From_Integer(fixedpoint.Whole_Integer(high_quartile)))
	mean := fixedpoint.From_Ratio(
		fixedpoint.Numerator(total), fixedpoint.Denominator(count),
	)
	deviation := standard_deviation(Deviations(values), Average(mean_integer), Kept(count))

	measurement = Measurement{
		Mean:               Mean(mean),
		Standard_Deviation: Standard_Deviation(deviation),
		Min: Minimum(fixedpoint.Number(fixedpoint.From_Integer(
			fixedpoint.Whole_Integer(sorted[0]),
		))),
		Max: Maximum(fixedpoint.Number(fixedpoint.From_Integer(
			fixedpoint.Whole_Integer(sorted[count-1]),
		))),
		Median: Median(fixedpoint.Number(fixedpoint.From_Integer(
			fixedpoint.Whole_Integer(sorted[count/2]),
		))),
		Q1: First_Quartile(q1),
		Q3: Third_Quartile(q3),
		Outlier_Count: outlier_count(&Outlier_Count_Input{
			Sorted:        sorted,
			Low_Quartile:  Low_Quartile(low_quartile),
			High_Quartile: High_Quartile(high_quartile),
		}),
		Sample_Count: Kept(count),
		Unit:         unit,
	}
	return measurement
}

// Sorts points ascending in place by heapsort: the shared sort stops at a smaller collection than
// the sample cap, and heapsort needs no scratch store and no recursion.
func points_sort(points Points) {
	Points_Invariants(points, "points_sort.points")
	count := len(points)
	for root_index := count/2 - 1; root_index >= 0; root_index-- {
		points_sift(points, Root_Index(root_index), End_Index(count))
	}
	for end_index := count - 1; end_index > 0; end_index-- {
		points[0], points[end_index] = points[end_index], points[0]
		points_sift(points, 0, End_Index(end_index))
	}
}

// Moves the value at root_index down the heap held in the first end_index points.
func points_sift(points Points, root_index Root_Index, end_index End_Index) {
	Root_Index_Invariants(root_index, "points_sift.root_index")
	End_Index_Invariants(end_index, "points_sift.end_index")
	Points_Invariants(points, "points_sift.points")
	parent_index := int(root_index)
	for depth_index := 0; depth_index < POINTS_MAX; depth_index++ {
		child_index := 2*parent_index + 1
		if child_index >= int(end_index) {
			return
		}
		if child_index+1 < int(end_index) {
			if points[child_index+1] > points[child_index] {
				child_index++
			}
		}
		if points[parent_index] >= points[child_index] {
			return
		}
		points[parent_index], points[child_index] =
			points[child_index], points[parent_index]
		parent_index = child_index
	}
}

// Wide_add_square adds value squared into a 128-bit accumulator.
func wide_add_square(subtotal Wide_Subtotal, value Gap) (sum Wide) {
	defer func() { Wide_Invariants(sum, "wide_add_square.sum") }()
	Wide_Subtotal_Invariants(subtotal, "wide_add_square.accumulator")
	Gap_Invariants(value, "wide_add_square.value")
	product_high, product_low := bits.Multiply_64(
		bits.Word_64(value), bits.Multiplier_64(value))
	low, carry := bits.Add_64(bits.Word_64(subtotal.Low), bits.Addend_64(product_low), 0)
	high, _ := bits.Add_64(bits.Word_64(subtotal.High), bits.Addend_64(product_high),
		bits.Carry_In(carry))
	return Wide{High: Accumulator_High(high), Low: Accumulator_Low(low)}
}

// Root_of_quotient returns the fixed-point square root of a 128-bit numerator over a denominator —
// the shared tail of the sample and pooled deviations. A quotient that fits a signed word keeps
// full fractional precision; a larger one, a multi-second jitter far outside maddox's fast-command
// envelope, falls back to the integer root.
func root_of_quotient(input Root_Of_Quotient_Input_Handle) (deviation Standard_Deviation) {
	defer func() { Standard_Deviation_Invariants(deviation, "root_of_quotient.deviation") }()
	Root_Of_Quotient_Input_Handle_Invariants(input, "root_of_quotient.input")
	if int(input.Denominator) <= 0 {
		return 0
	}
	denominator := uint64(int(input.Denominator))
	high := uint64(input.High)
	low := uint64(input.Low)
	quotient_high := high / denominator
	quotient_low, _ := bits.Divide_64(bits.Dividend_High_64(high%denominator),
		bits.Dividend_Low_64(low), bits.Divisor_64(denominator))
	fits := quotient_high == 0
	if fits {
		fits = uint64(quotient_low) <= INT64_MAX
	}
	if fits {
		return Standard_Deviation(fixedpoint.Square_Root_Scaled(
			fixedpoint.Radicand(quotient_low),
		))
	}
	root := fixedpoint.Integer_Root(
		fixedpoint.High_Word(quotient_high), fixedpoint.Low_Word(quotient_low),
	)
	return Standard_Deviation(fixedpoint.From_Integer(fixedpoint.Whole_Integer(root)))
}

// Renders a fixed-point number with digits decimals, as the shared formatter writes it.
func fixed_text(value Fixed_Value, digits Fraction_Digits) (text Fixed_Text) {
	defer func() { Fixed_Text_Invariants(text, "fixed_text.text") }()
	Fraction_Digits_Invariants(digits, "fixed_text.digits")
	Fixed_Value_Invariants(value, "fixed_text.value")
	storage := [fixedpoint.TEXT_SIZE_MAXIMUM]byte{}
	count := fixedpoint.Into_Text(
		storage[:], fixedpoint.Number(value), fixedpoint.Digit_Count(digits))
	return Fixed_Text(string(storage[:count]))
}

// Table_Head_Render renders the host hardware block that opens the table. Sparse fields (zero
// values) are omitted, matching the sparse-metric convention for table rows.
func Table_Head_Render(m Machine_Specs) (report nbio.Stream_Memory) {
	Machine_Specs_Invariants(m, "Table_Head_Render.m")
	// An empty CPU model with no architecture means specs were never acquired; the header is
	// skipped so output stays clean.
	if m.CPU_Model == "" {
		if m.CPU_Arch == "" {
			return report
		}
	}
	text := []byte("Machine: " + string(m.CPU_Model) + " (" + string(m.CPU_Arch) + ")\n")
	topology := Core_Topology{
		Physical:    m.Physical_Cores,
		Logical:     m.Logical_Cores,
		Performance: m.Performance_Cores,
		Efficiency:  m.Efficiency_Cores,
	}
	text = append(text, "  cores: "+string(machine_specs_cores(topology))+"   "+
		"freq: "+string(format_hz(Hertz(m.CPU_Frequency_Hz_Max)))+"   "+
		"ram: "+string(format_bytes(Byte_Size(m.RAM_Total_Bytes)))+"   "+
		"storage: "+string(format_bytes(Byte_Size(m.Storage_Total_Bytes)))+"\n"...)
	cache_line := ""
	if m.Cache_L1_Bytes > 0 {
		cache_line += "L1: " + string(format_bytes(Byte_Size(m.Cache_L1_Bytes))) + "   "
	}
	if m.Cache_L2_Bytes > 0 {
		cache_line += "L2: " + string(format_bytes(Byte_Size(m.Cache_L2_Bytes))) + "   "
	}
	if m.Cache_L3_Bytes > 0 {
		cache_line += "L3: " + string(format_bytes(Byte_Size(m.Cache_L3_Bytes))) + "   "
	}
	operating_system_part := "OS: " + string(m.Operating_System_Name) + " " +
		string(m.Operating_System_Version) + "   kernel: " + string(m.Kernel_Version)
	text = append(text, "  "+cache_line+operating_system_part+"\n\n"...)
	return nbio.Stream_Memory{Memory: text, Cursor: int64(len(text))}
}

// Table_Benchmark_Render appends one benchmark's header, its column header, and its metric rows.
// The header and the rows go through render_cells, so labels align with data.
func Table_Benchmark_Render(
	report nbio.Stream_Memory, index Position, benchmark Benchmark, color Color,
) (extended nbio.Stream_Memory) {
	Position_Invariants(index, "Table_Benchmark_Render.index")
	Benchmark_Invariants(benchmark, "Table_Benchmark_Render.benchmark")
	Color_Invariants(color, "Table_Benchmark_Render.color")
	text := fmt.Appendf(report.Memory[:report.Cursor], "Benchmark %d (%d runs, %s): ",
		int(index)+1, benchmark.Runs, format_elapsed(benchmark.Elapsed))
	for word_index, word := range benchmark.Command {
		if word_index > 0 {
			text = append(text, ' ')
		}
		text = append(text, word...)
	}
	text = append(text, '\n')
	columns := render_cells(&Render_Cells_Input{
		Name: "measurement", Mean: "mean", Sigma: "σ",
		Low: "min", High: "max", Outliers: "outliers",
	})
	text = append(text, columns...)
	// The reference is the first benchmark; it is compared against nothing, so its row carries
	// no delta column. Every later benchmark is a candidate.
	if index != 0 {
		text = append(text, "  delta"...)
	}
	text = append(text, '\n')
	report = render_metric_rows(nbio.Stream_Memory{Memory: text, Cursor: int64(len(text))},
		index, benchmark, color)
	text = append(report.Memory[:report.Cursor], '\n')
	return nbio.Stream_Memory{Memory: text, Cursor: int64(len(text))}
}

// Machine_specs_cores renders the CPU core layout, naming performance and efficiency cores on
// hybrid CPUs and collapsing to a single count when physical equals logical.
func machine_specs_cores(m Core_Topology) (text Cores_Line) {
	defer func() { Cores_Line_Invariants(text, "machine_specs_cores.text") }()
	Core_Topology_Invariants(m, "machine_specs_cores.m")
	if m.Performance > 0 {
		if m.Efficiency > 0 {
			return Cores_Line(fmt.Sprintf("%d P + %d E = %d logical",
				m.Performance, m.Efficiency, m.Logical))
		}
	}
	if int(m.Physical) == int(m.Logical) {
		return Cores_Line(fmt.Sprintf("%d", m.Physical))
	}
	return Cores_Line(fmt.Sprintf("%d physical, %d logical", m.Physical, m.Logical))
}

// Format_hz renders a frequency in Hz to a human-readable GHz or MHz string.
func format_hz(hz Hertz) (text Frequency) {
	defer func() { Frequency_Invariants(text, "format_hz.text") }()
	Hertz_Invariants(hz, "format_hz.hz")
	if hz == 0 {
		return "?"
	}
	if hz >= 1_000_000_000 {
		gigahertz := fixedpoint.From_Ratio(fixedpoint.Numerator(hz), 1_000_000_000)
		return Frequency(fixed_text(Fixed_Value(gigahertz), 2) + " GHz")
	}
	megahertz := fixedpoint.From_Ratio(fixedpoint.Numerator(hz), 1_000_000)
	return Frequency(fixed_text(Fixed_Value(megahertz), 0) + " MHz")
}

// Format_bytes renders a byte count, drawn from the machine specs as an unsigned integer, with the
// binary suffix ladder. The raw count is scaled down its rung and only then lifted into
// fixed-point, dividing through a 128-bit ratio: lifting the whole value first would overflow the
// 2^20 scale for a multi-petabyte size — the byte-size ceiling reaches 2^53 — and render a
// garbage, over-width cell.
func format_bytes(value Byte_Size) (text Cell) {
	defer func() { Cell_Invariants(text, "format_bytes.text") }()
	Byte_Size_Invariants(value, "format_bytes.value")
	raw := int64(value)
	divisors := [LADDER_SIZE]int64{1 << 40, 1 << 30, 1 << 20, 1 << 10, 1}
	suffixes := [LADDER_SIZE]string{"TiB", "GiB", "MiB", "KiB", "B"}
	for rung := range divisors {
		if raw >= divisors[rung] {
			scaled := fixedpoint.From_Ratio(
				fixedpoint.Numerator(raw), fixedpoint.Denominator(divisors[rung]),
			)
			return Cell(string(format_significant(Quantity(scaled))) + suffixes[rung])
		}
	}
	return Cell(format_significant(Quantity(fixedpoint.From_Integer(0))))
}

// Format_elapsed renders a total sampling duration for the benchmark header with the time ladder.
// Like format_bytes it scales the raw nanoseconds down their rung before lifting into
// fixed-point, and it caps the magnitude first, so an absurd multi-year span still lands within
// the header's glyph rather than rendering a five-digit count.
func format_elapsed(elapsed Elapsed) (text Span) {
	defer func() { Span_Invariants(text, "format_elapsed.text") }()
	Elapsed_Invariants(elapsed, "format_elapsed.elapsed")
	raw := min(int64(elapsed), ELAPSED_DISPLAY_MAX)
	divisors := [LADDER_SIZE]int64{1_000_000_000_000, 1_000_000_000, 1_000_000, 1_000, 1}
	suffixes := [LADDER_SIZE]string{"ks", "s", "ms", "us", "ns"}
	for rung := range divisors {
		if raw >= divisors[rung] {
			scaled := fixedpoint.From_Ratio(
				fixedpoint.Numerator(raw), fixedpoint.Denominator(divisors[rung]),
			)
			return Span(string(format_significant(Quantity(scaled))) + suffixes[rung])
		}
	}
	return Span(format_significant(Quantity(fixedpoint.From_Integer(0))))
}

// Render_metric_rows writes one aligned row per metric, in report order.
func render_metric_rows(
	report nbio.Stream_Memory, index Position, benchmark Benchmark, color Color,
) (extended nbio.Stream_Memory) {
	Position_Invariants(index, "render_metric_rows.index")
	Benchmark_Invariants(benchmark, "render_metric_rows.benchmark")
	Color_Invariants(color, "render_metric_rows.color")
	m := benchmark.Measurements
	d := benchmark.Deltas
	// The reference is first; only later benchmarks carry a comparison.
	has_delta := Delta_Presence(index != 0)
	rows := [METRIC_ROW_COUNT]Metric_Line_Input{
		{
			Name: "wall_time", Measurement: Row_Measurement(m.Wall_Time),
			Delta: Delta(d.Wall_Time),
		},
		{
			Name: "peak_rss", Measurement: Row_Measurement(m.Peak_RSS),
			Delta: Delta(d.Peak_RSS),
		},
		{
			Name: "cpu_cycles", Measurement: Row_Measurement(m.CPU_Cycles),
			Delta: Delta(d.CPU_Cycles),
		},
		{
			Name: "instructions", Measurement: Row_Measurement(m.Instructions),
			Delta: Delta(d.Instructions),
		},
		{
			Name: "cache_references", Measurement: Row_Measurement(m.Cache_References),
			Delta: Delta(d.Cache_References),
		},
		{
			Name: "cache_misses", Measurement: Row_Measurement(m.Cache_Misses),
			Delta: Delta(d.Cache_Misses),
		},
		{
			Name: "branch_misses", Measurement: Row_Measurement(m.Branch_Misses),
			Delta: Delta(d.Branch_Misses),
		},
		{
			Name: "cpu_user", Measurement: Row_Measurement(m.CPU_User),
			Delta: Delta(d.CPU_User),
		},
		{
			Name: "cpu_system", Measurement: Row_Measurement(m.CPU_System),
			Delta: Delta(d.CPU_System),
		},
	}
	for row := range rows {
		// A metric with no data on this platform — every value zero, e.g. the Linux-only
		// cache and branch counters on macOS — is left out of the table.
		if rows[row].Measurement.Max == 0 {
			continue
		}
		rows[row].Has_Delta = has_delta
		rows[row].Color = color
		report = metric_line(report, &rows[row])
	}
	return report
}

// METRIC_ROW_COUNT is the metrics a benchmark reports, one table row each.
const METRIC_ROW_COUNT = 9

// Metric_line renders one metric row: its name, scaled mean ± σ, min … max, outlier count, and —
// when the benchmark is not the reference — its delta — and ends the line.
func metric_line(
	report nbio.Stream_Memory, input Metric_Line_Input_Handle,
) (extended nbio.Stream_Memory) {
	Metric_Line_Input_Handle_Invariants(input, "metric_line.input")
	measurement := input.Measurement
	unit := measurement.Unit
	outlier_percent := fixedpoint.Number(0)
	if measurement.Sample_Count > 0 {
		outlier_percent = fixedpoint.From_Ratio(
			fixedpoint.Numerator(measurement.Outlier_Count)*100,
			fixedpoint.Denominator(measurement.Sample_Count),
		)
	}
	outlier_text := fmt.Sprintf("%d (%s%%)", int(measurement.Outlier_Count),
		fixed_text(Fixed_Value(outlier_percent), 0))
	row := render_cells(&Render_Cells_Input{
		Name: input.Name,
		Mean: Mean_Cell(format_quantity(Magnitude(measurement.Mean), unit)),
		Sigma: Deviation_Cell(format_quantity(
			Magnitude(measurement.Standard_Deviation), unit,
		)),
		Low:      Minimum_Cell(format_quantity(Magnitude(measurement.Min), unit)),
		High:     Maximum_Cell(format_quantity(Magnitude(measurement.Max), unit)),
		Outliers: Outliers(outlier_text),
	})
	text := append(report.Memory[:report.Cursor], row...)
	if input.Has_Delta {
		text = append(append(text, "  "...), delta_render(input.Delta, input.Color)...)
	}
	text = append(text, '\n')
	return nbio.Stream_Memory{Memory: text, Cursor: int64(len(text))}
}

// Pad aligns text to a visible width with spaces — on the left when right is set, so the text
// right-aligns, on the right otherwise. The width is the rune count, so a multibyte glyph like σ
// still counts as one column.
func pad(text Column, width Extent, right Right_Alignment) (result Padded) {
	defer func() { Padded_Invariants(result, "pad.padded") }()
	Column_Invariants(text, "pad.text")
	Extent_Invariants(width, "pad.width")
	Right_Alignment_Invariants(right, "pad.right")
	rune_count := 0
	for range string(text) {
		rune_count++
	}
	space := int(width) - rune_count
	if space < 0 {
		return Padded(text)
	}
	spaces := "            "[:space]
	if right {
		return Padded(spaces + string(text))
	}
	return Padded(string(text) + spaces)
}

// Delta_render formats one metric's change: a sign, the percentage, and its confidence
// half-interval. A significant change is colored — red slower, green faster — while an
// insignificant one stays faint.
func delta_render(delta Delta, color Color) (text Delta_Text) {
	defer func() { Delta_Text_Invariants(text, "delta_render.text") }()
	Delta_Invariants(delta, "delta_render.delta")
	Color_Invariants(color, "delta_render.color")
	sign := "+"
	if delta.Faster {
		sign = "-"
	}
	code := ANSI_FAINT
	if delta.Significant {
		code = ANSI_BRIGHT_RED
		if delta.Faster {
			code = ANSI_BRIGHT_GREEN
		}
	}
	difference := fixedpoint.Number(delta.Diff_Percent)
	if difference < 0 {
		difference = -difference
	}
	// The most negative word negates to itself, a magnitude past any the column shows.
	if difference < 0 {
		difference = PERCENT_DISPLAY_MAX
	}
	half_percent := fixedpoint.Number(delta.Half_Percent)
	// Pin both percentages to the column's widest value. A change past ten million percent
	// cannot fit the fixed delta layout the body invariant sizes for, and its exact magnitude
	// past the bound is noise; without the clamp its digits overrun DELTA_BODY_MAX and trip the
	// guard.
	if difference > PERCENT_DISPLAY_MAX {
		difference = PERCENT_DISPLAY_MAX
	}
	if half_percent > PERCENT_DISPLAY_MAX {
		half_percent = PERCENT_DISPLAY_MAX
	}
	diff := pad(Column(fixed_text(Fixed_Value(difference), 1)), 5, true)
	half := pad(Column(fixed_text(Fixed_Value(half_percent), 1)), 4, true)
	body := Delta_Body(sign + string(diff) + "% ± " + string(half) + "%")
	return paint(body, code, color)
}

// Format_significant renders a scaled value to three significant figures: whole numbers and
// hundreds with no decimals, tens with one, units with two.
func format_significant(value Quantity) (text Glyph) {
	defer func() { Glyph_Invariants(text, "format_significant.text") }()
	Quantity_Invariants(value, "format_significant.value")
	if value >= Quantity(fixedpoint.From_Integer(1000)) {
		return Glyph(fixed_text(Fixed_Value(value), 0))
	}
	if fixedpoint.Is_Integer(fixedpoint.Number(value)) {
		return Glyph(fixed_text(Fixed_Value(value), 0))
	}
	if value >= Quantity(fixedpoint.From_Integer(100)) {
		return Glyph(fixed_text(Fixed_Value(value), 0))
	}
	if value >= Quantity(fixedpoint.From_Integer(10)) {
		return Glyph(fixed_text(Fixed_Value(value), 1))
	}
	return Glyph(fixed_text(Fixed_Value(value), 2))
}

// Render_Progress_Input is one progress update: the command being sampled, how long it has been
// sampling, and how many runs are done against the cap.
type Render_Progress_Input struct {
	// Words is the command being sampled.
	Words Command_Line
	// Elapsed is how long it has been sampling.
	Elapsed Elapsed
	// Phase is whether the run is warming up or sampling.
	Phase Phase
	// Count is how many runs have completed so far.
	Count Census
	// Total is the run cap the count is measured against.
	Total Progress_Total
}

// Render_Progress_Input_Invariants states the update's coverable fields.
func Render_Progress_Input_Invariants(input Render_Progress_Input, namespace aver.Namespace) {
	Elapsed_Invariants(input.Elapsed, namespace)
	Phase_Invariants(input.Phase, namespace)
	Census_Invariants(input.Count, namespace)
	Progress_Total_Invariants(input.Total, namespace)
	Command_Line_Invariants(input.Words, namespace)
}

// Render_progress renders an in-place progress line: elapsed seconds, an optional phase word
// (warmup), the counter (count over its total, or just the count when the total is disabled), and
// the command.
func render_progress(input Render_Progress_Input_Handle) (line nbio.Stream_Memory) {
	Render_Progress_Input_Handle_Invariants(input, "render_progress.input")
	seconds := fixedpoint.From_Ratio(
		fixedpoint.Numerator(input.Elapsed), fixedpoint.Denominator(time.SECOND),
	)
	text := []byte(PROGRESS_CLEAR + fixed_text(Fixed_Value(seconds), 1) + "s  ")
	if input.Phase != "" {
		text = append(text, string(input.Phase)+" "...)
	}
	text = fmt.Appendf(text, "%d", int(input.Count))
	if input.Total > 0 {
		text = fmt.Appendf(text, "/%d", int64(input.Total))
	}
	text = append(text, "  "...)
	text = append(text, progress_label(input.Words)...)
	return nbio.Stream_Memory{Memory: text, Cursor: int64(len(text))}
}

// Progress_label joins the command words and truncates them to keep the progress line on one
// terminal row.
func progress_label(words Command_Line) (text Label) {
	defer func() { Label_Invariants(text, "progress_label.label") }()
	Command_Line_Invariants(words, "progress_label.words")
	joined := ""
	for index, word := range words {
		Command_Word_Invariants(word, "progress_label.word")
		if index > 0 {
			joined += " "
		}
		joined += string(word)
	}
	runes := []rune(joined)
	if len(runes) <= PROGRESS_LABEL_RUNES_MAX {
		return Label(joined)
	}
	return Label(string(runes[:PROGRESS_LABEL_RUNES_MAX-1]) + "…")
}

// JSON_HEX_DIGITS spells a control character's escape.
const JSON_HEX_DIGITS = "0123456789abcdef"

// JSON_REPLACEMENT is the replacement rune, as three bytes, a valid one spans.
const JSON_REPLACEMENT = "�"

// Json_Text is one JSON string value: a command word or a host field.
type Json_Text string

// Json_Text_Invariants bounds the text to the longest command word.
func Json_Text_Invariants(value Json_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COLLECTION_MIN, COMMAND_WORD_BYTES_MAX).
		Ensure()
}

// Json_Head_Render renders the document up to its first benchmark: the machine object and the
// opening of the benchmark array. Members follow the order and form the reflective encoder gave
// them: machine fields that are optional are omitted when zero.
func Json_Head_Render(m Machine_Specs) (report nbio.Stream_Memory) {
	Machine_Specs_Invariants(m, "Json_Head_Render.m")
	text := []byte("{\n  \"machine\": {\n    \"cpu_model\": ")
	report = json_string(nbio.Stream_Memory{Memory: text, Cursor: int64(len(text))},
		Json_Text(m.CPU_Model))
	text = append(report.Memory[:report.Cursor], ",\n    \"cpu_arch\": "...)
	report = json_string(nbio.Stream_Memory{Memory: text, Cursor: int64(len(text))},
		Json_Text(m.CPU_Arch))
	text = fmt.Appendf(report.Memory[:report.Cursor],
		",\n    \"physical_cores\": %d,\n    \"logical_cores\": %d",
		int(m.Physical_Cores), int(m.Logical_Cores))
	if m.Performance_Cores != 0 {
		text = fmt.Appendf(text, ",\n    \"performance_cores\": %d",
			int(m.Performance_Cores))
	}
	if m.Efficiency_Cores != 0 {
		text = fmt.Appendf(text, ",\n    \"efficiency_cores\": %d", int(m.Efficiency_Cores))
	}
	if m.CPU_Frequency_Hz_Max != 0 {
		text = fmt.Appendf(text, ",\n    \"cpu_frequency_hz_max\": %d",
			uint64(m.CPU_Frequency_Hz_Max))
	}
	if m.Cache_L1_Bytes != 0 {
		text = fmt.Appendf(text, ",\n    \"cache_l1_bytes\": %d", uint64(m.Cache_L1_Bytes))
	}
	if m.Cache_L2_Bytes != 0 {
		text = fmt.Appendf(text, ",\n    \"cache_l2_bytes\": %d", uint64(m.Cache_L2_Bytes))
	}
	if m.Cache_L3_Bytes != 0 {
		text = fmt.Appendf(text, ",\n    \"cache_l3_bytes\": %d", uint64(m.Cache_L3_Bytes))
	}
	text = fmt.Appendf(text, ",\n    \"ram_total_bytes\": %d,\n    \"storage_total_bytes\": %d",
		uint64(m.RAM_Total_Bytes), uint64(m.Storage_Total_Bytes))
	text = append(text, ",\n    \"operating_system_name\": "...)
	report = json_string(nbio.Stream_Memory{Memory: text, Cursor: int64(len(text))},
		Json_Text(m.Operating_System_Name))
	text = append(report.Memory[:report.Cursor], ",\n    \"operating_system_version\": "...)
	report = json_string(nbio.Stream_Memory{Memory: text, Cursor: int64(len(text))},
		Json_Text(m.Operating_System_Version))
	text = append(report.Memory[:report.Cursor], ",\n    \"kernel_version\": "...)
	report = json_string(nbio.Stream_Memory{Memory: text, Cursor: int64(len(text))},
		Json_Text(m.Kernel_Version))
	text = append(report.Memory[:report.Cursor], "\n  },\n  \"benchmarks\": [\n"...)
	return nbio.Stream_Memory{Memory: text, Cursor: int64(len(text))}
}

// Json_Benchmark_Render appends one benchmark to the array, after a separator unless it is the
// first.
func Json_Benchmark_Render(
	report nbio.Stream_Memory, index Position, benchmark Benchmark,
) (extended nbio.Stream_Memory) {
	Position_Invariants(index, "Json_Benchmark_Render.index")
	Benchmark_Invariants(benchmark, "Json_Benchmark_Render.benchmark")
	text := report.Memory[:report.Cursor]
	if index > 0 {
		text = append(text, ",\n"...)
	}
	text = append(text, "    {\n      \"command\": [\n"...)
	for word_index, word := range benchmark.Command {
		if word_index > 0 {
			text = append(text, ",\n"...)
		}
		text = append(text, "        "...)
		report = json_string(nbio.Stream_Memory{Memory: text, Cursor: int64(len(text))},
			Json_Text(word))
		text = report.Memory[:report.Cursor]
	}
	text = fmt.Appendf(text, "\n      ],\n      \"runs\": %d,\n      \"elapsed_ns\": %d,\n",
		int(benchmark.Runs), int64(benchmark.Elapsed))
	return json_metrics(nbio.Stream_Memory{Memory: text, Cursor: int64(len(text))},
		benchmark.Measurements, benchmark.Deltas)
}

// Appends the measurements and deltas objects, closing the benchmark. Fixed-point numbers are
// their raw scaled integers.
func json_metrics(
	report nbio.Stream_Memory, m Measurements, d Deltas,
) (extended nbio.Stream_Memory) {
	Measurements_Invariants(m, "json_metrics.m")
	Deltas_Invariants(d, "json_metrics.d")
	names := [METRIC_ROW_COUNT]string{"wall_time", "peak_rss", "cpu_cycles", "instructions",
		"cache_references", "cache_misses", "branch_misses", "cpu_user", "cpu_system"}
	measurements := [METRIC_ROW_COUNT]Measurement{
		Measurement(m.Wall_Time), Measurement(m.Peak_RSS), Measurement(m.CPU_Cycles),
		Measurement(m.Instructions), Measurement(m.Cache_References),
		Measurement(m.Cache_Misses), Measurement(m.Branch_Misses), Measurement(m.CPU_User),
		Measurement(m.CPU_System),
	}
	deltas := [METRIC_ROW_COUNT]Delta{
		Delta(d.Wall_Time), Delta(d.Peak_RSS), Delta(d.CPU_Cycles), Delta(d.Instructions),
		Delta(d.Cache_References), Delta(d.Cache_Misses), Delta(d.Branch_Misses),
		Delta(d.CPU_User), Delta(d.CPU_System),
	}
	text := report.Memory[:report.Cursor]
	separator := "      \"measurements\": {\n"
	for row, value := range measurements {
		text = fmt.Appendf(text,
			"%s        %q: {\n          \"mean\": %d,\n          \"stddev\": %d,\n"+
				"          \"min\": %d,\n          \"max\": %d,\n"+
				"          \"median\": %d,\n          \"q1\": %d,\n"+
				"          \"q3\": %d,\n          \"outliers\": %d,\n"+
				"          \"count\": %d,\n          \"unit\": %q\n        }",
			separator, names[row], int64(value.Mean),
			int64(value.Standard_Deviation), int64(value.Min), int64(value.Max),
			int64(value.Median), int64(value.Q1), int64(value.Q3),
			int(value.Outlier_Count), int(value.Sample_Count), string(value.Unit))
		separator = ",\n"
	}
	separator = "\n      },\n      \"deltas\": {\n"
	for row, delta := range deltas {
		text = fmt.Appendf(text,
			"%s        %q: {\n          \"diff_percent\": %d,\n"+
				"          \"half_percent\": %d,\n          \"significant\": %t,\n"+
				"          \"faster\": %t\n        }",
			separator, names[row], int64(delta.Diff_Percent), int64(delta.Half_Percent),
			bool(delta.Significant), bool(delta.Faster))
		separator = ",\n"
	}
	text = append(text, "\n      }\n    }"...)
	return nbio.Stream_Memory{Memory: text, Cursor: int64(len(text))}
}

// Json_Tail_Render closes the array and the document with a trailing newline.
func Json_Tail_Render(report nbio.Stream_Memory) (extended nbio.Stream_Memory) {
	text := append(report.Memory[:report.Cursor], "\n  ]\n}\n"...)
	return nbio.Stream_Memory{Memory: text, Cursor: int64(len(text))}
}

// Appends value as a JSON string, escaped as the standard encoder escapes it: the quote, the
// backslash, the short control escapes, every other control and the HTML-significant <, >, & as
// \u00XX, the line and paragraph separators, and every invalid byte as the replacement rune. A
// range over text decodes an invalid byte as a replacement rune one byte wide, where a valid
// U+FFFD is three bytes wide.
func json_string(
	report nbio.Stream_Memory, value Json_Text,
) (extended nbio.Stream_Memory) {
	Json_Text_Invariants(value, "json_string.value")
	text := append(report.Memory[:report.Cursor], '"')
	for index, character := range string(value) {
		invalid := false
		if character == '�' {
			invalid = !bool(strings.Has_Prefix(
				strings.Text(value[index:]), strings.Text(JSON_REPLACEMENT)))
		}
		switch {
		case invalid:
			text = append(text, `�`...)
		case character == '"', character == '\\':
			text = append(text, '\\', byte(character))
		case character == '\b':
			text = append(text, `\b`...)
		case character == '\f':
			text = append(text, `\f`...)
		case character == '\n':
			text = append(text, `\n`...)
		case character == '\r':
			text = append(text, `\r`...)
		case character == '\t':
			text = append(text, `\t`...)
		case character < 0x20, character == '<', character == '>', character == '&':
			text = append(text, '\\', 'u', '0', '0',
				JSON_HEX_DIGITS[character>>4], JSON_HEX_DIGITS[character&0xf])
		case character == ' ', character == ' ':
			text = fmt.Appendf(text, `\u%04x`, character)
		default:
			text = append(text, string(character)...)
		}
	}
	text = append(text, '"')
	return nbio.Stream_Memory{Memory: text, Cursor: int64(len(text))}
}
