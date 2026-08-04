// Package maddox compares the performance of commands on macOS, the way poop does
// on Linux. It is the pure tier of the maddox binary: the statistics, the
// command-to-command comparison, and the JSON report are computed here over
// injected Samples, so this package spawns nothing and reads no clock of its own —
// package main wires the cgo measurer and the operating-system clock and hands them
// in through Main_Input.
package maddox

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/bits"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"local/james-orcales/shared/cli"
	invariant "local/james-orcales/shared/invariant/default"
	sysio "local/james-orcales/shared/io"
	"local/james-orcales/shared/math/fixedpoint"
	"local/james-orcales/shared/time"
)

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

// Main parses the injected operating-system arguments, benchmarks each command, and
// writes the selected report.
func Main(input Main_Input) (code Exit_Code) {
	defer func() { Exit_Code_Invariants(code, "Main.exit_code") }()
	Main_Input_Invariants(input, "Main.input")
	program := main_program()
	if cli.Handle_Completion(program, input.Arguments, input.Output) {
		return EXIT_SUCCESS
	}
	parser := cli.Program_Parse(&program, cli.Program_Parse_Input{Arguments: input.Arguments})
	result := cli.Parser_Done(parser)
	if result == nil {
		panic("maddox parser did not complete synchronously")
	}
	command := result.Command
	parse_err := result.Error
	if errors.Is(parse_err, cli.Help_Requested) {
		cli.Print_Requested_Help(input.Output, program, command)
		return EXIT_SUCCESS
	}
	if parse_err != nil {
		fmt.Fprintf(input.Error_Output, "maddox: %v\n\n", parse_err)
		cli.Print_Help(input.Error_Output, program)
		return EXIT_USAGE
	}
	command_strings := Cli_Commands(
		cli.Get_Option(command.Arguments, "command").Value.([]string))
	Cli_Commands_Invariants(command_strings, "Main.command_strings")
	commands, build_err := commands_from_strings(command_strings)
	if build_err != nil {
		fmt.Fprintf(input.Error_Output, "maddox: %v\n", build_err)
		return EXIT_USAGE
	}
	if len(commands) == 0 {
		cli.Print_Help(input.Error_Output, program)
		return EXIT_USAGE
	}
	format := OUTPUT_FORMAT_TABLE
	if cli.Get_Option(command.Flags, "json").Value.(bool) {
		format = OUTPUT_FORMAT_JSON
	}
	duration_seconds := cli.Get_Option(command.Flags, "duration").Value.(int)
	runs := cli.Get_Option(command.Flags, "runs").Value.(int)
	warmup := cli.Get_Option(command.Flags, "warmup").Value.(int)
	allow_failures := cli.Get_Option(command.Flags, "allow-failures").Value.(bool)
	color_mode := Stream_Mode(cli.Get_Option(command.Flags, "color").Value.(string))
	progress_mode := Stream_Mode(cli.Get_Option(command.Flags, "progress").Value.(string))
	return Exit_Code(main_benchmark(&Benchmark_Input{
		Commands: Commands(commands),
		Sampler:  input.Sampler,
		Duration_Max: Duration_Limit(
			time.Duration(duration_seconds) * time.SECOND,
		),
		Runs_Max:       Run_Limit(runs),
		Warmup_Count:   Warmup_Limit(warmup),
		Allow_Failures: Failure_Allowance(allow_failures),
		Format:         format,
		Color:          Color(resolve_stream(color_mode, input.Output_Is_Terminal)),
		Progress: Progress(resolve_stream(
			progress_mode, input.Error_Output_Is_Terminal)),
		Output:       input.Output,
		Error_Output: input.Error_Output,
		Machine:      input.Machine,
	}))
}

// Main_benchmark samples parsed commands and writes their comparison report.
func main_benchmark(input *Benchmark_Input) (code Benchmark_Exit_Code) {
	defer func() { Benchmark_Exit_Code_Invariants(code, "main_benchmark.exit_code") }()
	Benchmark_Input_Invariants(*input, "main_benchmark.input")
	benchmarks := make([]Benchmark, 0, len(input.Commands))
	reference := Measurements{}
	have_reference := false
	collect_input := Collect_Samples_Input{
		Sampler:        input.Sampler,
		Duration_Max:   input.Duration_Max,
		Runs_Max:       input.Runs_Max,
		Warmup_Count:   input.Warmup_Count,
		Allow_Failures: input.Allow_Failures,
		Progress:       input.Progress,
		Stderr:         input.Error_Output,
	}
	for index, command := range input.Commands {
		samples, run_exit, child_stderr := main_input_collect_samples(
			&collect_input, command)
		if input.Progress {
			input.Error_Output.Write([]byte(PROGRESS_CLEAR))
		}
		if run_exit != 0 {
			write_failure(&Write_Failure_Input{
				Stderr:       input.Error_Output,
				Index:        Position(index),
				Exit:         Failure_Status(run_exit),
				Child_Stderr: child_stderr,
			})
			return Benchmark_Exit_Code(EXIT_FAILURE)
		}
		measurements := measurements_compute(Distribution(samples))
		benchmark := Benchmark{
			Command:      command_words(command),
			Runs:         Kept(len(samples)),
			Elapsed:      samples_elapsed(Distribution(samples)),
			Measurements: measurements,
		}
		if have_reference {
			benchmark.Deltas = deltas_compute(
				reference, Candidate_Measurements(measurements))
		} else {
			reference = measurements
			have_reference = true
		}
		benchmarks = append(benchmarks, benchmark)
	}
	document := Document{Machine: input.Machine, Benchmarks: benchmarks}
	if input.Format == OUTPUT_FORMAT_JSON {
		return write_report(input.Output, document)
	}
	return write_table(input.Output, &Render_Table_Input{
		Document: document,
		Color:    input.Color,
	})
}

// Main_program declares commands, sampling limits, and output controls.
func main_program() (program cli.Program) {
	return cli.New_Single(cli.New_Single_Input{
		Label:       "maddox",
		Description: "benchmark and compare commands",
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

// Commands_from_strings separates each command without invoking a shell.
func commands_from_strings(
	command_strings Cli_Commands,
) (commands Parsed_Commands, err error) {
	defer func() { Parsed_Commands_Invariants(commands, "commands_from_strings.commands") }()
	Cli_Commands_Invariants(command_strings, "commands_from_strings.command_strings")
	if len(command_strings) > COMMAND_SET_MAX {
		return nil, fmt.Errorf("command count exceeds %d", COMMAND_SET_MAX)
	}
	commands = make(Parsed_Commands, 0, len(command_strings))
	for _, text := range command_strings {
		fields := strings.Fields(text)
		if len(fields) > COMMAND_WORDS_MAX {
			return nil, fmt.Errorf("command word count exceeds %d", COMMAND_WORDS_MAX)
		}
		for _, field := range fields {
			if len(field) > COMMAND_WORD_BYTES_MAX {
				return nil, fmt.Errorf(
					"command word exceeds %d bytes", COMMAND_WORD_BYTES_MAX)
			}
		}
		environment := []string{}
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
		command := sysio.Process_Request{Environment: environment, Path: remainder[0]}
		if len(remainder) > 1 {
			command.Arguments = remainder[1:]
		}
		commands = append(commands, command)
	}
	return commands, nil
}

// Resolve_stream applies an explicit mode or the injected terminal state.
func resolve_stream[Status ~uint8](mode Stream_Mode, terminal Status) (enabled Stream_State) {
	defer func() { Stream_State_Invariants(enabled, "resolve_stream.enabled") }()
	Stream_Mode_Invariants(mode, "resolve_stream.mode")
	Terminal_Status_Invariants(Terminal_Status(terminal), "resolve_stream.terminal")
	if mode == "always" {
		return true
	}
	if mode == "never" {
		return false
	}
	return Stream_State(uint8(terminal) == TERMINAL_STATUS_TERMINAL)
}

// Continuation records whether sampling takes one more sample.
type Continuation bool

// Continuation_Invariants witnesses both sampling decisions.
func Continuation_Invariants(value Continuation, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "Sampling continues.").
		Ensure()
}

// Run_Limit is the maximum number of kept runs.
type Run_Limit int64

// Run_Limit_Invariants accepts the complete signed 64-bit input domain.
func Run_Limit_Invariants(value Run_Limit, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Warmup_Limit_Invariants(value Warmup_Limit, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Failure_Allowance_Invariants(value Failure_Allowance, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "Command failures are allowed.").
		Ensure()
}

// Color records whether a rendered report contains ANSI color.
type Color bool

// Color_Invariants witnesses both color modes.
func Color_Invariants(value Color, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "ANSI color is enabled.").
		Ensure()
}

// Progress records whether sampling writes progress text.
type Progress bool

// Progress_Invariants witnesses both progress modes.
func Progress_Invariants(value Progress, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "Progress output is enabled.").
		Ensure()
}

// Output_Format selects how Main renders the report.
type Output_Format uint8

// Output_Format_Invariants holds an Output_Format to its two declared rendering modes
// and witnesses each mode. The modes are an enumeration, not a span, so a member axis
// names the mode it claims.
func Output_Format_Invariants(format Output_Format, namespace invariant.Namespace) {
	invariant.Tree(format, namespace).
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
func Resident_Bytes_Invariants(value Resident_Bytes, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Cycle_Count_Invariants(value Cycle_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Instruction_Count_Invariants(value Instruction_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
	value Cache_Reference_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
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
func Cache_Miss_Count_Invariants(value Cache_Miss_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Branch_Miss_Count_Invariants(value Branch_Miss_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func User_Time_Invariants(value User_Time, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), METRIC_MIN, METRIC_MAX).
		Ensure()
}

// System_Time is one sample's kernel-space CPU time.
type System_Time time.Duration

// System_Time_Invariants bounds a kernel-space CPU-time sample as a metric.
func System_Time_Invariants(value System_Time, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Sample_Invariants(sample Sample, namespace invariant.Namespace) {
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
func Captured_Output_Invariants(output Captured_Output, namespace invariant.Namespace) {
	invariant.Tree(output, namespace).
		Range_Int(len(output), CAPTURE_BYTES_MIN, CAPTURE_BYTES_MAX).
		Ensure()
}

// Completion_Moment is the monotonic clock reading after one measured run.
type Completion_Moment time.Moment

// Completion_Moment_Invariants distinguishes an absent stamp from a measured stamp.
func Completion_Moment_Invariants(value Completion_Moment, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(value != 0, "A completion moment is set.").
		Ensure()
}

// Run_Result is everything one measured run reports: its Sample, the exit code,
// and the stderr captured for a failing run.
type Run_Result struct {
	// Sample is the run's measurements, including its own tight wall time.
	Sample Sample
	// Completed_At is the sampler's monotonic-clock reading when this run finished. It
	// drives the time budget as a real stopwatch — spanning the gaps between runs, which
	// the reported per-run wall does not — while the library itself reads no clock: it
	// only subtracts the moments the sampler hands it. Left zero when the budget is off.
	Completed_At Completion_Moment
	// Exit is the command's exit code; non-zero is a failure.
	Exit Exit_Status
	// Stderr is the command's captured stderr, surfaced on failure.
	Stderr Captured_Output
}

// Run_Result_Invariants states a Run_Result's fields.
func Run_Result_Invariants(result Run_Result, namespace invariant.Namespace) {
	Sample_Invariants(result.Sample, namespace)
	Completion_Moment_Invariants(result.Completed_At, namespace)
	Exit_Status_Invariants(result.Exit, namespace)
	Captured_Output_Invariants(result.Stderr, namespace)
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
func Samples_Invariants(samples Samples, namespace invariant.Namespace) {
	invariant.Tree(samples, namespace).
		Range_Holed_Int(len(samples), COLLECTION_MIN, SAMPLES_MAX, 1, 2, 2, 2).
		Ensure()
}

// Distribution is a quorum of kept runs the statistics reduce — always at least the 3-run
// minimum, never the empty, one, or two counts the reducers cannot summarize.
type Distribution []Sample

// Distribution_Invariants bounds the run count: the quorum floor and the kept-run ceiling
// are witnessed, every short boundary claimed unreachable.
func Distribution_Invariants(samples Distribution, namespace invariant.Namespace) {
	invariant.Tree(samples, namespace).Range_Int(len(samples), QUORUM_MIN, SAMPLES_MAX).Ensure()
}

// Values is one metric's value pulled from every sample, the int64 the statistics work in.
type Values []int64

// Values_Invariants bounds the value count. The values are pulled from a kept distribution,
// floored at the 3-run quorum, so the quorum floor and the kept-run ceiling are the witnessed
// shapes; the empty, one, and two counts a quorum never holds fall below the range and are guarded.
func Values_Invariants(values Values, namespace invariant.Namespace) {
	invariant.Tree(values, namespace).Range_Int(len(values), QUORUM_MIN, SAMPLES_MAX).Ensure()
}

// Series is one metric pulled from a distribution — always the quorum's worth, never the
// empty, one, or two counts a distribution cannot hold.
type Series []int64

// Series_Invariants bounds the extracted count: it mirrors a distribution's quorum floor
// and kept-run ceiling, claiming every short boundary unreachable.
func Series_Invariants(values Series, namespace invariant.Namespace) {
	invariant.Tree(values, namespace).Range_Int(len(values), QUORUM_MIN, SAMPLES_MAX).Ensure()
}

// Deviations is the values the standard deviation reduces — a kept distribution's, so
// always at least the 3-run quorum, never the empty, one, or two counts below it.
type Deviations []int64

// Deviations_Invariants bounds the count. The deviations are a kept distribution's, floored at
// the 3-run quorum, so the quorum floor and the kept-run ceiling are witnessed and the shorter
// counts are guarded out.
func Deviations_Invariants(values Deviations, namespace invariant.Namespace) {
	invariant.Tree(values, namespace).Range_Int(len(values), QUORUM_MIN, SAMPLES_MAX).Ensure()
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
func Command_Line_Invariants(words Command_Line, namespace invariant.Namespace) {
	invariant.Tree(words, namespace).Range_Int(len(words), WORD_MIN, COMMAND_WORDS_MAX).Ensure()
}

// Parsed_Commands is a parser result, including the empty or rejected shape.
type Parsed_Commands []sysio.Process_Request

// Parsed_Commands_Invariants bounds every result from positional command parsing.
func Parsed_Commands_Invariants(commands Parsed_Commands, namespace invariant.Namespace) {
	invariant.Tree(commands, namespace).
		Range_Int(len(commands), COLLECTION_MIN, COMMAND_SET_MAX).
		Ensure()
}

// Commands is the set of commands a run benchmarks, in invocation order.
type Commands []sysio.Process_Request

// Commands_Invariants bounds a parsed, nonempty command set.
func Commands_Invariants(commands Commands, namespace invariant.Namespace) {
	invariant.Tree(commands, namespace).
		Range_Int(len(commands), COMMAND_SET_MIN, COMMAND_SET_MAX).
		Ensure()
}

// Benchmarks is one entry per benchmarked command, in invocation order.
type Benchmarks []Benchmark

// Benchmarks_Invariants bounds the entries produced by a valid invocation.
func Benchmarks_Invariants(benchmarks Benchmarks, namespace invariant.Namespace) {
	invariant.Tree(benchmarks, namespace).
		Range_Int(len(benchmarks), COMMAND_SET_MIN, COMMAND_SET_MAX).
		Ensure()
}

// REPORT_MIN is the shortest valid table: one one-byte command and no metric rows.
const REPORT_MIN = 99

// REPORT_MAX bounds the rendered bytes at the widest document the pipeline produces: a full
// command set of ceiling-scale, outlier-saturated benchmarks, colored, under a full machine
// header. Every field is at its own ceiling, so no document renders wider.
const REPORT_MAX = 9925

// Report is the rendered output of a document — the table or the JSON bytes.
type Report []byte

// Report_Invariants bounds a rendered report from its shortest valid table.
func Report_Invariants(report Report, namespace invariant.Namespace) {
	invariant.Tree(report, namespace).
		Range_Int(len(report), REPORT_MIN, REPORT_MAX).
		Ensure()
}

// LADDER_SIZE is the fixed rung count of every scale ladder.
const LADDER_SIZE = 5

// Ladder is a scale's rungs from largest divisor to smallest, fixed per unit family. A
// fixed array, not a slice: the rung count is part of the type, so the length never varies
// and carries no boundary discipline — only the steady fixed size, asserted below.
type Ladder [LADDER_SIZE]Scale_Step

// Ladder_Invariants states the one thing a ladder's length can be: its fixed size. The array
// type pins the count at compile time, so this is a steady truth, not a witnessed boundary.
func Ladder_Invariants(ladder Ladder, namespace invariant.Namespace) {
	invariant.Always(len(ladder) == LADDER_SIZE, "A ladder is always its fixed size.")
}

// Sampler is the one capability Main reaches the world through: run a command once
// and report what happened.
type Sampler struct {
	// Measure runs the command once and reports the Run_Result.
	Measure func(command sysio.Process_Request) (result Run_Result)
}

// Sampler_Invariants states the one property the function field carries that a func
// type cannot: a sampler must be able to measure, so its Measure is never nil — a nil
// one would crash Main the moment it tried to run a command.
func Sampler_Invariants(sampler Sampler, namespace invariant.Namespace) {
	invariant.Always(sampler.Measure != nil, "A sampler can always measure.")
}

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
func Host_Text_Invariants(text Host_Text, namespace invariant.Namespace) {
	invariant.Tree(text, namespace).
		Range_Int(len(text), HOST_TEXT_BYTES_MIN, HOST_TEXT_BYTES_MAX).
		Ensure()
}

// Processor_Model is the processor brand text.
type Processor_Model string

// Processor_Model_Invariants bounds processor brand text.
func Processor_Model_Invariants(value Processor_Model, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), HOST_TEXT_BYTES_MIN, HOST_TEXT_BYTES_MAX).
		Ensure()
}

// Processor_Architecture is the processor instruction-set name.
type Processor_Architecture string

// Processor_Architecture_Invariants bounds an instruction-set name.
func Processor_Architecture_Invariants(
	value Processor_Architecture, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), HOST_TEXT_BYTES_MIN, HOST_TEXT_BYTES_MAX).
		Ensure()
}

// Physical_Core_Count is the machine's physical core count.
type Physical_Core_Count int

// Physical_Core_Count_Invariants bounds a physical core count.
func Physical_Core_Count_Invariants(
	value Physical_Core_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), CORES_COUNT_MIN, CORES_COUNT_MAX).
		Ensure()
}

// Logical_Core_Count is the machine's visible hardware-thread count.
type Logical_Core_Count int

// Logical_Core_Count_Invariants bounds a logical core count.
func Logical_Core_Count_Invariants(value Logical_Core_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), CORES_COUNT_MIN, CORES_COUNT_MAX).
		Ensure()
}

// Performance_Core_Count is the machine's performance-core count.
type Performance_Core_Count int

// Performance_Core_Count_Invariants bounds a performance-core count.
func Performance_Core_Count_Invariants(
	value Performance_Core_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), CORES_COUNT_MIN, CORES_COUNT_MAX).
		Ensure()
}

// Efficiency_Core_Count is the machine's efficiency-core count.
type Efficiency_Core_Count int

// Efficiency_Core_Count_Invariants bounds an efficiency-core count.
func Efficiency_Core_Count_Invariants(
	value Efficiency_Core_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), CORES_COUNT_MIN, CORES_COUNT_MAX).
		Ensure()
}

// Processor_Frequency is the maximum processor frequency in hertz.
type Processor_Frequency uint64

// Processor_Frequency_Invariants bounds a processor frequency.
func Processor_Frequency_Invariants(
	value Processor_Frequency, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), HERTZ_MIN, HERTZ_MAX).
		Ensure()
}

// Level_1_Cache_Size is the processor's L1 cache size in bytes.
type Level_1_Cache_Size uint64

// Level_1_Cache_Size_Invariants bounds an L1 cache size.
func Level_1_Cache_Size_Invariants(
	value Level_1_Cache_Size, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), BYTE_SIZE_MIN, BYTE_SIZE_MAX).
		Ensure()
}

// Level_2_Cache_Size is the processor's L2 cache size in bytes.
type Level_2_Cache_Size uint64

// Level_2_Cache_Size_Invariants bounds an L2 cache size.
func Level_2_Cache_Size_Invariants(
	value Level_2_Cache_Size, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), BYTE_SIZE_MIN, BYTE_SIZE_MAX).
		Ensure()
}

// Level_3_Cache_Size is the processor's L3 cache size in bytes.
type Level_3_Cache_Size uint64

// Level_3_Cache_Size_Invariants bounds an L3 cache size.
func Level_3_Cache_Size_Invariants(
	value Level_3_Cache_Size, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), BYTE_SIZE_MIN, BYTE_SIZE_MAX).
		Ensure()
}

// Memory_Size is the machine's physical memory size in bytes.
type Memory_Size uint64

// Memory_Size_Invariants bounds a physical memory size.
func Memory_Size_Invariants(value Memory_Size, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), BYTE_SIZE_MIN, BYTE_SIZE_MAX).
		Ensure()
}

// Storage_Size is the boot volume's storage size in bytes.
type Storage_Size uint64

// Storage_Size_Invariants bounds a storage size.
func Storage_Size_Invariants(value Storage_Size, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), BYTE_SIZE_MIN, BYTE_SIZE_MAX).
		Ensure()
}

// Operating_System_Name is the operating-system product name.
type Operating_System_Name string

// Operating_System_Name_Invariants bounds an operating-system name.
func Operating_System_Name_Invariants(
	value Operating_System_Name, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), HOST_TEXT_BYTES_MIN, HOST_TEXT_BYTES_MAX).
		Ensure()
}

// Operating_System_Version is the operating-system release text.
type Operating_System_Version string

// Operating_System_Version_Invariants bounds operating-system release text.
func Operating_System_Version_Invariants(
	value Operating_System_Version, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), HOST_TEXT_BYTES_MIN, HOST_TEXT_BYTES_MAX).
		Ensure()
}

// Kernel_Version is the operating-system kernel release text.
type Kernel_Version string

// Kernel_Version_Invariants bounds kernel release text.
func Kernel_Version_Invariants(value Kernel_Version, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Machine_Specs_Invariants(specs Machine_Specs, namespace invariant.Namespace) {
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
func Unit_Invariants(name Unit, namespace invariant.Namespace) {
	invariant.Tree(name, namespace).
		Enum_Int(len(name), UNIT_BYTES_MIN, UNIT_BYTES_MAX).
		Ensure()
}

// STANDARD_DEVIATION_MAXIMUM is the largest sample deviation for three or more
// values in the metric domain. Three samples at zero, zero, and METRIC_MAX produce it.
const STANDARD_DEVIATION_MAXIMUM int64 = 5078426674188

// Mean is the arithmetic mean of one metric.
type Mean fixedpoint.Number

// Mean_Invariants bounds a mean to the metric domain.
func Mean_Invariants(value Mean, namespace invariant.Namespace) {
	invariant.Always(value >= 0, "A mean cannot be negative.")
	invariant.Tree(value, namespace).
		Enum_Int64(min(max(int64(value)/fixedpoint.SCALE, 1), 2), 1, 2).
		Ensure()
}

// Standard_Deviation is the sample standard deviation of one metric.
type Standard_Deviation fixedpoint.Number

// Standard_Deviation_Invariants bounds a deviation to the metric domain.
func Standard_Deviation_Invariants(value Standard_Deviation, namespace invariant.Namespace) {
	invariant.Always(value >= 0, "A standard deviation cannot be negative.")
	invariant.Always(
		int64(value)/fixedpoint.SCALE <= STANDARD_DEVIATION_MAXIMUM,
		"A standard deviation cannot exceed the metric domain.",
	)
	invariant.Tree(value, namespace).
		Enum_3_Int64(min(int64(value)/fixedpoint.SCALE, 2), 0, 1, 2).
		Ensure()
}

// Minimum is the smallest value of one metric.
type Minimum fixedpoint.Number

// Minimum_Invariants bounds a minimum to the metric domain.
func Minimum_Invariants(value Minimum, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(
			int64(value)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Ensure()
}

// Maximum is the largest value of one metric.
type Maximum fixedpoint.Number

// Maximum_Invariants bounds a maximum to the metric domain.
func Maximum_Invariants(value Maximum, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Int64(min(max(int64(value)/fixedpoint.SCALE, 1), 2), 1, 2).
		Ensure()
}

// Median is the middle value of one metric.
type Median fixedpoint.Number

// Median_Invariants bounds a median to the metric domain.
func Median_Invariants(value Median, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(
			int64(value)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Ensure()
}

// First_Quartile is the lower quartile of one metric.
type First_Quartile fixedpoint.Number

// First_Quartile_Invariants bounds a first quartile to the metric domain.
func First_Quartile_Invariants(value First_Quartile, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(
			int64(value)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Ensure()
}

// Third_Quartile is the upper quartile of one metric.
type Third_Quartile fixedpoint.Number

// Third_Quartile_Invariants bounds a third quartile to the metric domain.
func Third_Quartile_Invariants(value Third_Quartile, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(
			int64(value)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
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
func Measurement_Invariants(measurement Measurement, namespace invariant.Namespace) {
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

// Difference_Percent is a signed percentage difference.
type Difference_Percent fixedpoint.Number

// Difference_Percent_Invariants states the fixed-point percentage domain.
func Difference_Percent_Invariants(value Difference_Percent, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Int64(min(max(int64(value), -1), 1), -1, 0, 1).
		Ensure()
}

// Half_Percent is the half-width of a percentage confidence interval.
type Half_Percent fixedpoint.Number

// Half_Percent_Invariants states the nonnegative fixed-point percentage domain.
func Half_Percent_Invariants(value Half_Percent, namespace invariant.Namespace) {
	invariant.Always(value >= 0, "A confidence half-interval cannot be negative.")
	invariant.Tree(value, namespace).
		Enum_Int64(min(int64(value), 1), 0, 1).
		Ensure()
}

// Relative_Deviation is a standard deviation divided by its reference mean.
type Relative_Deviation fixedpoint.Number

// Relative_Deviation_Invariants states whether the nonnegative ratio is zero or positive.
func Relative_Deviation_Invariants(value Relative_Deviation, namespace invariant.Namespace) {
	invariant.Always(value >= 0, "A relative deviation cannot be negative.")
	invariant.Tree(value, namespace).
		Enum_Int64(min(int64(value), 1), 0, 1).
		Ensure()
}

// Student_T_Score is a 95-percent Student-t critical score.
type Student_T_Score fixedpoint.Number

// Student_T_Score_Invariants states the two whole-number score regions in the table.
func Student_T_Score_Invariants(value Student_T_Score, namespace invariant.Namespace) {
	invariant.Always(
		fixedpoint.Number(value) >= fixedpoint.From_Ratio(1960, 1000),
		"A t score is at least 1.96.",
	)
	invariant.Always(
		fixedpoint.Number(value) <= fixedpoint.From_Ratio(12706, 1000),
		"A t score is at most 12.706.",
	)
	invariant.Tree(value, namespace).
		Enum_Int64(min(int64(value)/fixedpoint.SCALE, 2), 1, 2).
		Ensure()
}

// SIGNIFICANCE_SIGNIFICANT is the shared positive significance fact.
const SIGNIFICANCE_SIGNIFICANT = true

// Significance records whether a confidence interval clears the significance band.
type Significance bool

// Significance_Invariants witnesses both significance outcomes.
func Significance_Invariants(value Significance, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "The difference is significant.").
		Ensure()
}

// SPEED_ORDER_FASTER is the shared positive speed-order fact.
const SPEED_ORDER_FASTER = true

// Speed_Order records whether the candidate is faster than the reference.
type Speed_Order bool

// Speed_Order_Invariants witnesses both speed orders.
func Speed_Order_Invariants(value Speed_Order, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Delta_Invariants(delta Delta, namespace invariant.Namespace) {
	Difference_Percent_Invariants(delta.Diff_Percent, namespace)
	Half_Percent_Invariants(delta.Half_Percent, namespace)
	Significance_Invariants(delta.Significant, namespace)
	Speed_Order_Invariants(delta.Faster, namespace)
}

// Wall_Time_Measurement is the wall-time distribution.
type Wall_Time_Measurement Measurement

// Wall_Time_Measurement_Invariants states the wall-time distribution fields.
func Wall_Time_Measurement_Invariants(
	value Wall_Time_Measurement, namespace invariant.Namespace,
) {
	invariant.Always(value.Unit == UNIT_TIME, "Wall time always uses nanoseconds.")
	invariant.Always(
		int64(value.Standard_Deviation)/fixedpoint.SCALE <= STANDARD_DEVIATION_MAXIMUM,
		"A wall-time standard deviation cannot exceed the metric domain.",
	)
	invariant.Tree(value, namespace).
		Range_Int64(
			int64(value.Mean)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Enum_3_Int64(min(int64(value.Standard_Deviation)/fixedpoint.SCALE, 2), 0, 1, 2).
		Range_Int64(
			int64(value.Min)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Max)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Median)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Q1)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Q3)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int(int(value.Outlier_Count), STRAYS_MIN, STRAYS_MAX).
		Range_Int(int(value.Sample_Count), KEPT_MIN, KEPT_MAX).
		Ensure()
}

// Peak_Resident_Measurement is the peak resident-memory distribution.
type Peak_Resident_Measurement Measurement

// Peak_Resident_Measurement_Invariants states the resident-memory distribution fields.
func Peak_Resident_Measurement_Invariants(
	value Peak_Resident_Measurement, namespace invariant.Namespace,
) {
	invariant.Always(value.Unit == UNIT_SIZE, "Peak resident memory always uses bytes.")
	invariant.Always(
		int64(value.Standard_Deviation)/fixedpoint.SCALE <= STANDARD_DEVIATION_MAXIMUM,
		"A resident-memory standard deviation cannot exceed the metric domain.",
	)
	invariant.Tree(value, namespace).
		Enum_3_Int64(min(int64(value.Mean)/fixedpoint.SCALE, 2), 0, 1, 2).
		Enum_3_Int64(min(int64(value.Standard_Deviation)/fixedpoint.SCALE, 2), 0, 1, 2).
		Range_Int64(
			int64(value.Min)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Enum_3_Int64(min(int64(value.Max)/fixedpoint.SCALE, 2), 0, 1, 2).
		Enum_3_Int64(min(int64(value.Median)/fixedpoint.SCALE, 2), 0, 1, 2).
		Range_Int64(
			int64(value.Q1)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Enum_3_Int64(min(int64(value.Q3)/fixedpoint.SCALE, 2), 0, 1, 2).
		Range_Int(int(value.Outlier_Count), STRAYS_MIN, STRAYS_MAX).
		Range_Int(int(value.Sample_Count), KEPT_MIN, KEPT_MAX).
		Ensure()
}

// Cycle_Measurement is the processor-cycle distribution.
type Cycle_Measurement Measurement

// Cycle_Measurement_Invariants states the processor-cycle distribution fields.
func Cycle_Measurement_Invariants(value Cycle_Measurement, namespace invariant.Namespace) {
	invariant.Always(value.Unit == UNIT_COUNT, "Processor cycles always use a count.")
	invariant.Always(
		int64(value.Standard_Deviation)/fixedpoint.SCALE <= STANDARD_DEVIATION_MAXIMUM,
		"A processor-cycle standard deviation cannot exceed the metric domain.",
	)
	invariant.Tree(value, namespace).
		Range_Int64(
			int64(value.Mean)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Enum_3_Int64(min(int64(value.Standard_Deviation)/fixedpoint.SCALE, 2), 0, 1, 2).
		Range_Int64(
			int64(value.Min)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Max)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Median)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Q1)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Q3)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int(int(value.Outlier_Count), STRAYS_MIN, STRAYS_MAX).
		Range_Int(int(value.Sample_Count), KEPT_MIN, KEPT_MAX).
		Ensure()
}

// Instruction_Measurement is the retired-instruction distribution.
type Instruction_Measurement Measurement

// Instruction_Measurement_Invariants states the instruction distribution fields.
func Instruction_Measurement_Invariants(
	value Instruction_Measurement, namespace invariant.Namespace,
) {
	invariant.Always(value.Unit == UNIT_COUNT, "Instructions always use a count.")
	invariant.Always(
		int64(value.Standard_Deviation)/fixedpoint.SCALE <= STANDARD_DEVIATION_MAXIMUM,
		"An instruction standard deviation cannot exceed the metric domain.",
	)
	invariant.Tree(value, namespace).
		Range_Int64(
			int64(value.Mean)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Enum_3_Int64(min(int64(value.Standard_Deviation)/fixedpoint.SCALE, 2), 0, 1, 2).
		Range_Int64(
			int64(value.Min)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Max)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Median)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Q1)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Q3)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int(int(value.Outlier_Count), STRAYS_MIN, STRAYS_MAX).
		Range_Int(int(value.Sample_Count), KEPT_MIN, KEPT_MAX).
		Ensure()
}

// Cache_Reference_Measurement is the cache-reference distribution.
type Cache_Reference_Measurement Measurement

// Cache_Reference_Measurement_Invariants states the cache-reference distribution fields.
func Cache_Reference_Measurement_Invariants(
	value Cache_Reference_Measurement, namespace invariant.Namespace,
) {
	invariant.Always(value.Unit == UNIT_COUNT, "Cache references always use a count.")
	invariant.Always(
		int64(value.Standard_Deviation)/fixedpoint.SCALE <= STANDARD_DEVIATION_MAXIMUM,
		"A cache-reference standard deviation cannot exceed the metric domain.",
	)
	invariant.Tree(value, namespace).
		Range_Int64(
			int64(value.Mean)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Enum_3_Int64(min(int64(value.Standard_Deviation)/fixedpoint.SCALE, 2), 0, 1, 2).
		Range_Int64(
			int64(value.Min)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Max)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Median)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Q1)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Q3)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int(int(value.Outlier_Count), STRAYS_MIN, STRAYS_MAX).
		Range_Int(int(value.Sample_Count), KEPT_MIN, KEPT_MAX).
		Ensure()
}

// Cache_Miss_Measurement is the cache-miss distribution.
type Cache_Miss_Measurement Measurement

// Cache_Miss_Measurement_Invariants states the cache-miss distribution fields.
func Cache_Miss_Measurement_Invariants(
	value Cache_Miss_Measurement, namespace invariant.Namespace,
) {
	invariant.Always(value.Unit == UNIT_COUNT, "Cache misses always use a count.")
	invariant.Always(
		int64(value.Standard_Deviation)/fixedpoint.SCALE <= STANDARD_DEVIATION_MAXIMUM,
		"A cache-miss standard deviation cannot exceed the metric domain.",
	)
	invariant.Tree(value, namespace).
		Range_Int64(
			int64(value.Mean)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Enum_3_Int64(min(int64(value.Standard_Deviation)/fixedpoint.SCALE, 2), 0, 1, 2).
		Range_Int64(
			int64(value.Min)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Max)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Median)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Q1)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Q3)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int(int(value.Outlier_Count), STRAYS_MIN, STRAYS_MAX).
		Range_Int(int(value.Sample_Count), KEPT_MIN, KEPT_MAX).
		Ensure()
}

// Branch_Miss_Measurement is the branch-miss distribution.
type Branch_Miss_Measurement Measurement

// Branch_Miss_Measurement_Invariants states the branch-miss distribution fields.
func Branch_Miss_Measurement_Invariants(
	value Branch_Miss_Measurement, namespace invariant.Namespace,
) {
	invariant.Always(value.Unit == UNIT_COUNT, "Branch misses always use a count.")
	invariant.Always(
		int64(value.Standard_Deviation)/fixedpoint.SCALE <= STANDARD_DEVIATION_MAXIMUM,
		"A branch-miss standard deviation cannot exceed the metric domain.",
	)
	invariant.Tree(value, namespace).
		Range_Int64(
			int64(value.Mean)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Enum_3_Int64(min(int64(value.Standard_Deviation)/fixedpoint.SCALE, 2), 0, 1, 2).
		Range_Int64(
			int64(value.Min)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Max)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Median)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Q1)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Q3)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int(int(value.Outlier_Count), STRAYS_MIN, STRAYS_MAX).
		Range_Int(int(value.Sample_Count), KEPT_MIN, KEPT_MAX).
		Ensure()
}

// User_Time_Measurement is the user-CPU-time distribution.
type User_Time_Measurement Measurement

// User_Time_Measurement_Invariants states the user-time distribution fields.
func User_Time_Measurement_Invariants(
	value User_Time_Measurement, namespace invariant.Namespace,
) {
	invariant.Always(value.Unit == UNIT_TIME, "User time always uses nanoseconds.")
	invariant.Always(
		int64(value.Standard_Deviation)/fixedpoint.SCALE <= STANDARD_DEVIATION_MAXIMUM,
		"A user-time standard deviation cannot exceed the metric domain.",
	)
	invariant.Tree(value, namespace).
		Range_Int64(
			int64(value.Mean)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Enum_3_Int64(min(int64(value.Standard_Deviation)/fixedpoint.SCALE, 2), 0, 1, 2).
		Range_Int64(
			int64(value.Min)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Max)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Median)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Q1)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Q3)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int(int(value.Outlier_Count), STRAYS_MIN, STRAYS_MAX).
		Range_Int(int(value.Sample_Count), KEPT_MIN, KEPT_MAX).
		Ensure()
}

// System_Time_Measurement is the system-CPU-time distribution.
type System_Time_Measurement Measurement

// System_Time_Measurement_Invariants states the system-time distribution fields.
func System_Time_Measurement_Invariants(
	value System_Time_Measurement, namespace invariant.Namespace,
) {
	invariant.Always(value.Unit == UNIT_TIME, "System time always uses nanoseconds.")
	invariant.Always(
		int64(value.Standard_Deviation)/fixedpoint.SCALE <= STANDARD_DEVIATION_MAXIMUM,
		"A system-time standard deviation cannot exceed the metric domain.",
	)
	invariant.Tree(value, namespace).
		Range_Int64(
			int64(value.Mean)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Enum_3_Int64(min(int64(value.Standard_Deviation)/fixedpoint.SCALE, 2), 0, 1, 2).
		Range_Int64(
			int64(value.Min)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Max)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Median)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Q1)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Range_Int64(
			int64(value.Q3)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
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
func Measurements_Invariants(measurements Measurements, namespace invariant.Namespace) {
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
func Wall_Time_Delta_Invariants(value Wall_Time_Delta, namespace invariant.Namespace) {
	invariant.Always(
		value.Half_Percent >= 0,
		"A wall-time confidence half-interval cannot be negative.",
	)
	invariant.Tree(value, namespace).
		Enum_3_Int64(min(max(int64(value.Diff_Percent), -1), 1), -1, 0, 1).
		Enum_Int64(min(int64(value.Half_Percent), 1), 0, 1).
		Sometimes(
			bool(value.Significant) == SIGNIFICANCE_SIGNIFICANT,
			"The difference is significant.").
		Sometimes(bool(value.Faster) == SPEED_ORDER_FASTER, "The candidate is faster.").
		Ensure()
}

// Peak_Resident_Delta is the peak resident-memory comparison.
type Peak_Resident_Delta Delta

// Peak_Resident_Delta_Invariants states the resident-memory comparison outcomes.
func Peak_Resident_Delta_Invariants(value Peak_Resident_Delta, namespace invariant.Namespace) {
	invariant.Always(
		value.Half_Percent >= 0,
		"A resident-memory confidence half-interval cannot be negative.",
	)
	invariant.Tree(value, namespace).
		Enum_3_Int64(min(max(int64(value.Diff_Percent), -1), 1), -1, 0, 1).
		Enum_Int64(min(int64(value.Half_Percent), 1), 0, 1).
		Sometimes(
			bool(value.Significant) == SIGNIFICANCE_SIGNIFICANT,
			"The difference is significant.").
		Sometimes(bool(value.Faster) == SPEED_ORDER_FASTER, "The candidate is faster.").
		Ensure()
}

// Cycle_Delta is the processor-cycle comparison.
type Cycle_Delta Delta

// Cycle_Delta_Invariants states the processor-cycle comparison outcomes.
func Cycle_Delta_Invariants(value Cycle_Delta, namespace invariant.Namespace) {
	invariant.Always(
		value.Half_Percent >= 0,
		"A processor-cycle confidence half-interval cannot be negative.",
	)
	invariant.Tree(value, namespace).
		Enum_3_Int64(min(max(int64(value.Diff_Percent), -1), 1), -1, 0, 1).
		Enum_Int64(min(int64(value.Half_Percent), 1), 0, 1).
		Sometimes(
			bool(value.Significant) == SIGNIFICANCE_SIGNIFICANT,
			"The difference is significant.").
		Sometimes(bool(value.Faster) == SPEED_ORDER_FASTER, "The candidate is faster.").
		Ensure()
}

// Instruction_Delta is the retired-instruction comparison.
type Instruction_Delta Delta

// Instruction_Delta_Invariants states the instruction comparison outcomes.
func Instruction_Delta_Invariants(value Instruction_Delta, namespace invariant.Namespace) {
	invariant.Always(
		value.Half_Percent >= 0,
		"An instruction confidence half-interval cannot be negative.",
	)
	invariant.Tree(value, namespace).
		Enum_3_Int64(min(max(int64(value.Diff_Percent), -1), 1), -1, 0, 1).
		Enum_Int64(min(int64(value.Half_Percent), 1), 0, 1).
		Sometimes(
			bool(value.Significant) == SIGNIFICANCE_SIGNIFICANT,
			"The difference is significant.").
		Sometimes(bool(value.Faster) == SPEED_ORDER_FASTER, "The candidate is faster.").
		Ensure()
}

// Cache_Reference_Delta is the cache-reference comparison.
type Cache_Reference_Delta Delta

// Cache_Reference_Delta_Invariants states the cache-reference comparison outcomes.
func Cache_Reference_Delta_Invariants(
	value Cache_Reference_Delta, namespace invariant.Namespace,
) {
	invariant.Always(
		value.Half_Percent >= 0,
		"A cache-reference confidence half-interval cannot be negative.",
	)
	invariant.Tree(value, namespace).
		Enum_3_Int64(min(max(int64(value.Diff_Percent), -1), 1), -1, 0, 1).
		Enum_Int64(min(int64(value.Half_Percent), 1), 0, 1).
		Sometimes(
			bool(value.Significant) == SIGNIFICANCE_SIGNIFICANT,
			"The difference is significant.").
		Sometimes(bool(value.Faster) == SPEED_ORDER_FASTER, "The candidate is faster.").
		Ensure()
}

// Cache_Miss_Delta is the cache-miss comparison.
type Cache_Miss_Delta Delta

// Cache_Miss_Delta_Invariants states the cache-miss comparison outcomes.
func Cache_Miss_Delta_Invariants(value Cache_Miss_Delta, namespace invariant.Namespace) {
	invariant.Always(
		value.Half_Percent >= 0,
		"A cache-miss confidence half-interval cannot be negative.",
	)
	invariant.Tree(value, namespace).
		Enum_3_Int64(min(max(int64(value.Diff_Percent), -1), 1), -1, 0, 1).
		Enum_Int64(min(int64(value.Half_Percent), 1), 0, 1).
		Sometimes(
			bool(value.Significant) == SIGNIFICANCE_SIGNIFICANT,
			"The difference is significant.").
		Sometimes(bool(value.Faster) == SPEED_ORDER_FASTER, "The candidate is faster.").
		Ensure()
}

// Branch_Miss_Delta is the branch-miss comparison.
type Branch_Miss_Delta Delta

// Branch_Miss_Delta_Invariants states the branch-miss comparison outcomes.
func Branch_Miss_Delta_Invariants(value Branch_Miss_Delta, namespace invariant.Namespace) {
	invariant.Always(
		value.Half_Percent >= 0,
		"A branch-miss confidence half-interval cannot be negative.",
	)
	invariant.Tree(value, namespace).
		Enum_3_Int64(min(max(int64(value.Diff_Percent), -1), 1), -1, 0, 1).
		Enum_Int64(min(int64(value.Half_Percent), 1), 0, 1).
		Sometimes(
			bool(value.Significant) == SIGNIFICANCE_SIGNIFICANT,
			"The difference is significant.").
		Sometimes(bool(value.Faster) == SPEED_ORDER_FASTER, "The candidate is faster.").
		Ensure()
}

// User_Time_Delta is the user-CPU-time comparison.
type User_Time_Delta Delta

// User_Time_Delta_Invariants states the user-time comparison outcomes.
func User_Time_Delta_Invariants(value User_Time_Delta, namespace invariant.Namespace) {
	invariant.Always(
		value.Half_Percent >= 0,
		"A user-time confidence half-interval cannot be negative.",
	)
	invariant.Tree(value, namespace).
		Enum_3_Int64(min(max(int64(value.Diff_Percent), -1), 1), -1, 0, 1).
		Enum_Int64(min(int64(value.Half_Percent), 1), 0, 1).
		Sometimes(
			bool(value.Significant) == SIGNIFICANCE_SIGNIFICANT,
			"The difference is significant.").
		Sometimes(bool(value.Faster) == SPEED_ORDER_FASTER, "The candidate is faster.").
		Ensure()
}

// System_Time_Delta is the system-CPU-time comparison.
type System_Time_Delta Delta

// System_Time_Delta_Invariants states the system-time comparison outcomes.
func System_Time_Delta_Invariants(value System_Time_Delta, namespace invariant.Namespace) {
	invariant.Always(
		value.Half_Percent >= 0,
		"A system-time confidence half-interval cannot be negative.",
	)
	invariant.Tree(value, namespace).
		Enum_3_Int64(min(max(int64(value.Diff_Percent), -1), 1), -1, 0, 1).
		Enum_Int64(min(int64(value.Half_Percent), 1), 0, 1).
		Sometimes(
			bool(value.Significant) == SIGNIFICANCE_SIGNIFICANT,
			"The difference is significant.").
		Sometimes(bool(value.Faster) == SPEED_ORDER_FASTER, "The candidate is faster.").
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
func Deltas_Invariants(deltas Deltas, namespace invariant.Namespace) {
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

// Elapsed is accumulated sampling time.
type Elapsed time.Duration

// Elapsed_Invariants distinguishes an instant sample set from one that consumed time.
func Elapsed_Invariants(value Elapsed, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Benchmark_Invariants(benchmark Benchmark, namespace invariant.Namespace) {
	Command_Line_Invariants(benchmark.Command, namespace)
	Kept_Invariants(benchmark.Runs, namespace)
	Elapsed_Invariants(benchmark.Elapsed, namespace)
	Measurements_Invariants(benchmark.Measurements, namespace)
	Deltas_Invariants(benchmark.Deltas, namespace)
}

// Document is the whole report: one Benchmark per command, in the order given,
// preceded by the host machine specs.
type Document struct {
	// Machine is the host hardware and OS snapshot, taken once at startup.
	Machine Machine_Specs `json:"machine"`
	// Benchmarks is one entry per command, in invocation order.
	Benchmarks Benchmarks `json:"benchmarks"`
}

// Document_Invariants composes the machine snapshot and the benchmark slice.
func Document_Invariants(document Document, namespace invariant.Namespace) {
	Machine_Specs_Invariants(document.Machine, namespace)
	Benchmarks_Invariants(document.Benchmarks, namespace)
}

// ARGUMENTS_COUNT_MIN is the program name that every process invocation carries.
const ARGUMENTS_COUNT_MIN = 1

// ARGUMENTS_COUNT_MAX bounds a realistic process command line.
const ARGUMENTS_COUNT_MAX = 4096

// Arguments is an operating-system command line, including the program name.
type Arguments []string

// Arguments_Invariants bounds the operating-system argument count.
func Arguments_Invariants(arguments Arguments, namespace invariant.Namespace) {
	invariant.Tree(arguments, namespace).
		Range_Int(len(arguments), ARGUMENTS_COUNT_MIN, ARGUMENTS_COUNT_MAX).
		Ensure()
}

// CLI_COMMANDS_COUNT_MAX leaves one operating-system argument for the program name.
const CLI_COMMANDS_COUNT_MAX = ARGUMENTS_COUNT_MAX - 1

// Cli_Commands is the command text parsed from positional arguments.
type Cli_Commands []string

// Cli_Commands_Invariants bounds the positional command count.
func Cli_Commands_Invariants(commands Cli_Commands, namespace invariant.Namespace) {
	invariant.Tree(commands, namespace).
		Range_Int(len(commands), COLLECTION_MIN, CLI_COMMANDS_COUNT_MAX).
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
func Stream_Mode_Invariants(mode Stream_Mode, namespace invariant.Namespace) {
	invariant.Tree(mode, namespace).
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
func Terminal_Status_Invariants(status Terminal_Status, namespace invariant.Namespace) {
	invariant.Tree(status, namespace).
		Enum_Uint8(
			uint8(status), TERMINAL_STATUS_NOT_TERMINAL, TERMINAL_STATUS_TERMINAL).
		Ensure()
}

// Output_Terminal_Status states whether the report stream points at a terminal.
type Output_Terminal_Status Terminal_Status

// Output_Terminal_Status_Invariants witnesses both report-stream states.
func Output_Terminal_Status_Invariants(
	status Output_Terminal_Status, namespace invariant.Namespace,
) {
	invariant.Tree(status, namespace).
		Enum_Uint8(
			uint8(status), TERMINAL_STATUS_NOT_TERMINAL, TERMINAL_STATUS_TERMINAL).
		Ensure()
}

// Error_Output_Terminal_Status states whether the diagnostic stream is a terminal.
type Error_Output_Terminal_Status Terminal_Status

// Error_Output_Terminal_Status_Invariants witnesses both diagnostic-stream states.
func Error_Output_Terminal_Status_Invariants(
	status Error_Output_Terminal_Status, namespace invariant.Namespace,
) {
	invariant.Tree(status, namespace).
		Enum_Uint8(
			uint8(status), TERMINAL_STATUS_NOT_TERMINAL, TERMINAL_STATUS_TERMINAL).
		Ensure()
}

// Stream_State states whether Maddox enables one optional stream feature.
type Stream_State bool

// Stream_State_Invariants witnesses both feature states.
func Stream_State_Invariants(state Stream_State, namespace invariant.Namespace) {
	invariant.Tree(state, namespace).
		Sometimes(bool(state), "The stream feature is active.").
		Ensure()
}

// Main_Input carries the command line and host bindings that Main needs.
type Main_Input struct {
	// Arguments is the operating-system command line, including the program name.
	Arguments Arguments
	// Sampler runs and measures one command, reporting its wall time; production wires
	// the operating-system measurer.
	Sampler Sampler
	// Output is where the report is written.
	Output io.Writer
	// Error_Output receives command-line and benchmark diagnostics.
	Error_Output io.Writer
	// Output_Is_Terminal reports whether Output points at a terminal.
	Output_Is_Terminal Output_Terminal_Status
	// Error_Output_Is_Terminal reports whether Error_Output points at a terminal.
	Error_Output_Is_Terminal Error_Output_Terminal_Status
	// Machine is the host hardware and OS snapshot; injected so the library makes no
	// ambient OS reads.
	Machine Machine_Specs
}

// Main_Input_Invariants states the coverable command-line and host fields.
func Main_Input_Invariants(input Main_Input, namespace invariant.Namespace) {
	Arguments_Invariants(input.Arguments, namespace)
	Sampler_Invariants(input.Sampler, namespace)
	Output_Terminal_Status_Invariants(input.Output_Is_Terminal, namespace)
	Error_Output_Terminal_Status_Invariants(input.Error_Output_Is_Terminal, namespace)
	Machine_Specs_Invariants(input.Machine, namespace)
}

// Duration_Limit is the optional per-command sampling budget.
type Duration_Limit time.Duration

// Duration_Limit_Invariants distinguishes an active budget from a disabled budget.
func Duration_Limit_Invariants(value Duration_Limit, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(value > 0, "A duration limit is active.").
		Ensure()
}

// Benchmark_Input carries the configuration resolved from the command line.
type Benchmark_Input struct {
	// Commands are the commands to benchmark; the first is the reference.
	Commands Commands
	// Sampler runs and measures one command.
	Sampler Sampler
	// Duration_Max is the per-command time budget.
	Duration_Max Duration_Limit
	// Runs_Max is the per-command run cap.
	Runs_Max Run_Limit
	// Warmup_Count is how many runs Main discards before sampling.
	Warmup_Count Warmup_Limit
	// Allow_Failures keeps benchmarking a command that exits non-zero.
	Allow_Failures Failure_Allowance
	// Format selects the report rendering.
	Format Output_Format
	// Color enables ANSI color in the table.
	Color Color
	// Progress enables live progress diagnostics.
	Progress Progress
	// Output receives the report.
	Output io.Writer
	// Error_Output receives progress and failure diagnostics.
	Error_Output io.Writer
	// Machine is the injected host snapshot.
	Machine Machine_Specs
}

// Benchmark_Input_Invariants states the resolved benchmark configuration.
func Benchmark_Input_Invariants(input Benchmark_Input, namespace invariant.Namespace) {
	Commands_Invariants(input.Commands, namespace)
	Sampler_Invariants(input.Sampler, namespace)
	Duration_Limit_Invariants(input.Duration_Max, namespace)
	Run_Limit_Invariants(input.Runs_Max, namespace)
	Warmup_Limit_Invariants(input.Warmup_Count, namespace)
	Failure_Allowance_Invariants(input.Allow_Failures, namespace)
	Output_Format_Invariants(input.Format, namespace)
	Color_Invariants(input.Color, namespace)
	Progress_Invariants(input.Progress, namespace)
	Machine_Specs_Invariants(input.Machine, namespace)
}

// Collect_Samples_Input carries only the dependencies needed by one sampling loop.
type Collect_Samples_Input struct {
	// Sampler runs and measures one command.
	Sampler Sampler
	// Duration_Max is the time budget for the command.
	Duration_Max Duration_Limit
	// Runs_Max is the kept-run limit.
	Runs_Max Run_Limit
	// Warmup_Count is the discarded-run limit.
	Warmup_Count Warmup_Limit
	// Allow_Failures keeps a failed run in the distribution.
	Allow_Failures Failure_Allowance
	// Progress enables progress output.
	Progress Progress
	// Stderr receives progress output.
	Stderr io.Writer
}

// Collect_Samples_Input_Invariants states the coverable sampling configuration.
func Collect_Samples_Input_Invariants(
	input Collect_Samples_Input, namespace invariant.Namespace,
) {
	Sampler_Invariants(input.Sampler, namespace)
	Duration_Limit_Invariants(input.Duration_Max, namespace)
	Run_Limit_Invariants(input.Runs_Max, namespace)
	Warmup_Limit_Invariants(input.Warmup_Count, namespace)
	Failure_Allowance_Invariants(input.Allow_Failures, namespace)
	Progress_Invariants(input.Progress, namespace)
}

// Main_input_collect_samples runs command through the warmup discards and then the
// measured loop, timing each kept run with the injected clock. A non-zero exit
// returns that exit and its stderr to abort the run; with Allow_Failures the run is
// kept and a zero exit is returned so sampling continues.
func main_input_collect_samples(
	input *Collect_Samples_Input, command sysio.Process_Request,
) (samples Samples, exit Exit_Status, stderr Captured_Output) {
	defer func() {
		Samples_Invariants(samples, "main_input_collect_samples.samples")
		Exit_Status_Invariants(exit, "main_input_collect_samples.exit")
		Captured_Output_Invariants(stderr, "main_input_collect_samples.stderr")
	}()
	Collect_Samples_Input_Invariants(*input, "main_input_collect_samples.input")
	warmups := 0
	// Elapsed is the running sum of measured wall time, the budget's clock: the library
	// reads no ambient clock, so a run's cost is the wall the sampler reports, accrued.
	var warmup_elapsed Elapsed
	for Warmup_Limit(warmups) < input.Warmup_Count {
		// Warming up past the kept-sample cap is pointless and would carry the warmup
		// counter past a tally's range, so the cap bounds both.
		if warmups >= SAMPLES_MAX {
			break
		}
		warm := input.Sampler.Measure(command)
		if warm.Exit != 0 {
			if !input.Allow_Failures {
				return nil, warm.Exit, warm.Stderr
			}
		}
		warmups++
		warmup_elapsed += Elapsed(warm.Sample.Wall)
		if input.Progress {
			render_progress(input.Stderr, &Render_Progress_Input{
				Command: command,
				Elapsed: warmup_elapsed,
				Phase:   "warmup",
				Count:   Census(warmups),
				Total:   Progress_Total(input.Warmup_Count),
			})
		}
	}
	samples = make([]Sample, 0)
	var elapsed Elapsed
	var stopwatch_start time.Moment
	for sampling_should_continue(&Sampling_Should_Continue_Input{
		Duration_Max: input.Duration_Max,
		Runs_Max:     input.Runs_Max,
		Count:        Tally(len(samples)),
		Elapsed:      elapsed,
	}) {
		result := input.Sampler.Measure(command)
		Run_Result_Invariants(result, "main_input_collect_samples.result")
		if result.Exit != 0 {
			if !input.Allow_Failures {
				return nil, result.Exit, result.Stderr
			}
		}
		sample := result.Sample
		if len(samples) == 0 {
			// Anchor at the first run's start so between-run gaps count in the budget.
			stopwatch_start = time.Moment(result.Completed_At) -
				time.Moment(sample.Wall)
		}
		samples = append(samples, sample)
		elapsed = Elapsed(time.Moment(result.Completed_At) - stopwatch_start)
		if input.Progress {
			render_progress(input.Stderr, &Render_Progress_Input{
				Command: command,
				Elapsed: elapsed,
				Count:   Census(len(samples)),
				Total:   Progress_Total(input.Runs_Max),
			})
		}
	}
	return samples, 0, nil
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
	input Sampling_Should_Continue_Input, namespace invariant.Namespace,
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
func sampling_should_continue(input *Sampling_Should_Continue_Input) (yes Continuation) {
	defer func() {
		Continuation_Invariants(yes, "sampling_should_continue.yes")
	}()
	Sampling_Should_Continue_Input_Invariants(*input, "sampling_should_continue.input")
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
	if input.Duration_Max > 0 {
		if time.Duration(input.Elapsed) >= time.Duration(input.Duration_Max) {
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
	value Candidate_Measurements, namespace invariant.Namespace,
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

// Extract pulls one metric's value out of every sample as the int64 the statistics
// work in; every raw metric is already an integer count, so no float ever enters.
func extract[T ~int64](
	samples Distribution, selector func(sample Sample) (value T),
) (values Series) {
	defer func() { Series_Invariants(values, "extract.values") }()
	Distribution_Invariants(samples, "extract.samples")
	values = make([]int64, len(samples))
	for index, sample := range samples {
		values[index] = int64(selector(sample))
	}
	return values
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
func Command_Word_Invariants(word Command_Word, namespace invariant.Namespace) {
	invariant.Tree(word, namespace).
		Range_Int(len(word), COMMAND_WORD_BYTES_MIN, COMMAND_WORD_BYTES_MAX).
		Ensure()
}

// Command_words flattens a command back to the words a reader recognizes:
// environment assignments, the executable, then its arguments.
func command_words(command sysio.Process_Request) (words Command_Line) {
	defer func() {
		Command_Line_Invariants(words, "command_words.words")
		for _, x := range words {
			Command_Word_Invariants(x, "command_words.word")
		}
	}()
	words = make([]Command_Word, 0, len(command.Environment)+1+len(command.Arguments))
	for _, environment := range command.Environment {
		words = append(words, Command_Word(environment))
	}
	words = append(words, Command_Word(command.Path))
	for _, argument := range command.Arguments {
		words = append(words, Command_Word(argument))
	}
	return words
}

// Write_report marshals the document to indented JSON and writes it to output,
// returning EXIT_FAILURE if marshaling or writing fails.
func write_report(output io.Writer, document Document) (code Benchmark_Exit_Code) {
	defer func() { Benchmark_Exit_Code_Invariants(code, "write_report.exit_code") }()
	Document_Invariants(document, "write_report.document")
	payload, marshal_err := json.MarshalIndent(document, "", "  ")
	if marshal_err != nil {
		return Benchmark_Exit_Code(EXIT_FAILURE)
	}
	payload = append(payload, '\n')
	_, write_err := output.Write(payload)
	if write_err != nil {
		return Benchmark_Exit_Code(EXIT_FAILURE)
	}
	return Benchmark_Exit_Code(EXIT_SUCCESS)
}

// Write_Failure_Input carries what a benchmarked command's failure is reported from.
type Write_Failure_Input struct {
	// Stderr is where the failure report is written.
	Stderr io.Writer
	// Index is the failing command's position in the run.
	Index Position
	// Exit is the non-zero status the command exited with.
	Exit Failure_Status
	// Child_Stderr is the failed command's own stderr, surfaced to the user.
	Child_Stderr Captured_Output
}

// Write_Failure_Input_Invariants states the failure report's coverable fields; the
// writer has no preset of its own.
func Write_Failure_Input_Invariants(input Write_Failure_Input, namespace invariant.Namespace) {
	Position_Invariants(input.Index, namespace)
	Failure_Status_Invariants(input.Exit, namespace)
	Captured_Output_Invariants(input.Child_Stderr, namespace)
}

// Write_failure reports a benchmarked command's non-zero exit to the diagnostic
// sink: a one-line header naming the command's position and code, then the
// command's own stderr.
func write_failure(input *Write_Failure_Input) {
	Write_Failure_Input_Invariants(*input, "write_failure.input")
	header := "maddox: benchmark " + strconv.Itoa(int(input.Index)+1) +
		" exited " + strconv.Itoa(int(input.Exit)) + "\n"
	input.Stderr.Write([]byte(header))
	input.Stderr.Write(input.Child_Stderr)
}

// INT64_MAX is the largest signed 64-bit value, used to test whether a 128-bit variance
// still fits a word before the fixed-point square root.
const INT64_MAX = 1<<63 - 1

// Measurement_Compute reduces one metric's per-run values to its distribution. It
// sorts a copy (the caller's slice is left untouched), then takes the mean, the
// sample standard deviation (n-1 denominator, matching poop), the extrema, the
// median, the quartiles by poop's index math, and the Tukey's-fences outlier count.
func Measurement_Compute(values Values, unit Unit) (measurement Measurement) {
	defer func() {
		Measurement_Invariants(measurement, "Measurement_Compute.measurement")
	}()
	Values_Invariants(values, "Measurement_Compute.values")
	for _, x := range values {
		Metric_Invariants(Metric(x), "Measurement_Compute.value")
	}
	Unit_Invariants(unit, "Measurement_Compute.unit")
	// The values are a kept distribution's quorum (Values_Invariants guards the empty,
	// one, and two counts away), so the statistics below always have a count to divide by.
	count := len(values)
	sorted := make([]int64, count)
	copy(sorted, values)
	slices.Sort(sorted)

	total := int64(0)
	for _, value := range sorted {
		total += value
	}
	mean_integer := total / int64(count)

	// Quartiles by position, exactly as poop indexes them: q3 falls back to the
	// maximum when there are too few points to take the upper quarter. The raw metric at
	// each quartile is kept alongside its fixed-point lift: the outlier fences are computed
	// from the raw integers, where 1.5*IQR cannot overflow the way the scaled form does.
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
			Sorted:        Points(sorted),
			Low_Quartile:  Low_Quartile(low_quartile),
			High_Quartile: High_Quartile(high_quartile),
		}),
		Sample_Count: Kept(count),
		Unit:         unit,
	}
	return measurement
}

// Wide is a 128-bit unsigned accumulator, for a sum of squares that overflows int64.
type Wide struct {
	// High is the upper 64 bits.
	High Accumulator_High
	// Low is the lower 64 bits.
	Low Accumulator_Low
}

// Wide_Invariants states the two halves of the 128-bit accumulator.
func Wide_Invariants(value Wide, namespace invariant.Namespace) {
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
func Wide_Subtotal_Invariants(value Wide_Subtotal, namespace invariant.Namespace) {
	Accumulator_Subtotal_High_Invariants(value.High, namespace)
	Accumulator_Low_Invariants(value.Low, namespace)
}

// Wide_add_square adds value squared into a 128-bit accumulator.
func wide_add_square(subtotal Wide_Subtotal, value Gap) (sum Wide) {
	defer func() { Wide_Invariants(sum, "wide_add_square.sum") }()
	Wide_Subtotal_Invariants(subtotal, "wide_add_square.accumulator")
	Gap_Invariants(value, "wide_add_square.value")
	product_high, product_low := bits.Mul64(uint64(value), uint64(value))
	low, carry := bits.Add64(uint64(subtotal.Low), product_low, 0)
	high, _ := bits.Add64(uint64(subtotal.High), product_high, carry)
	return Wide{High: Accumulator_High(high), Low: Accumulator_Low(low)}
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
	input Root_Of_Quotient_Input, namespace invariant.Namespace,
) {
	Accumulator_High_Invariants(input.High, namespace)
	Accumulator_Low_Invariants(input.Low, namespace)
	Divisor_Invariants(input.Denominator, namespace)
}

// Root_of_quotient returns the fixed-point square root of a 128-bit numerator over a
// denominator — the shared tail of the sample and pooled deviations. A quotient that fits
// a signed word keeps full fractional precision; a larger one, a multi-second jitter far
// outside maddox's fast-command envelope, falls back to the integer root.
func root_of_quotient(input *Root_Of_Quotient_Input) (deviation Standard_Deviation) {
	defer func() { Standard_Deviation_Invariants(deviation, "root_of_quotient.deviation") }()
	Root_Of_Quotient_Input_Invariants(*input, "root_of_quotient.input")
	if int(input.Denominator) <= 0 {
		return 0
	}
	denominator := uint64(int(input.Denominator))
	high := uint64(input.High)
	low := uint64(input.Low)
	quotient_high := high / denominator
	quotient_low, _ := bits.Div64(high%denominator, low, denominator)
	fits := quotient_high == 0
	if fits {
		fits = quotient_low <= INT64_MAX
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

// Low_Quartile is the lower quartile in the raw metric domain.
type Low_Quartile int64

// Low_Quartile_Invariants bounds a lower quartile.
func Low_Quartile_Invariants(value Low_Quartile, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func High_Quartile_Invariants(value High_Quartile, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Outlier_Count_Input_Invariants(input Outlier_Count_Input, namespace invariant.Namespace) {
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
func outlier_count(input *Outlier_Count_Input) (count Strays) {
	defer func() { Strays_Invariants(count, "outlier_count.count") }()
	Outlier_Count_Input_Invariants(*input, "outlier_count.input")
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

// Candidate_Measurement is the candidate side of one metric comparison.
type Candidate_Measurement Measurement

// Candidate_Measurement_Invariants states the candidate distribution fields.
func Candidate_Measurement_Invariants(
	value Candidate_Measurement, namespace invariant.Namespace,
) {
	invariant.Always(
		int64(value.Standard_Deviation)/fixedpoint.SCALE <= STANDARD_DEVIATION_MAXIMUM,
		"A candidate standard deviation cannot exceed the metric domain.",
	)
	invariant.Tree(value, namespace).
		Enum_Int64(min(max(int64(value.Mean)/fixedpoint.SCALE, 1), 2), 1, 2).
		Enum_3_Int64(min(int64(value.Standard_Deviation)/fixedpoint.SCALE, 2), 0, 1, 2).
		Range_Int64(
			int64(value.Min)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Enum_Int64(min(max(int64(value.Max)/fixedpoint.SCALE, 1), 2), 1, 2).
		Enum_Int64(min(max(int64(value.Median)/fixedpoint.SCALE, 1), 2), 1, 2).
		Range_Int64(
			int64(value.Q1)/fixedpoint.SCALE,
			METRIC_MIN,
			METRIC_MAX,
		).
		Enum_Int64(min(max(int64(value.Q3)/fixedpoint.SCALE, 1), 2), 1, 2).
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
	delta.Half_Percent = Half_Percent(half_interval(reference, candidate, Degree(degrees)))
	delta.Significant = significant(&Significant_Input{
		Diff_Percent: delta.Diff_Percent,
		Half_Percent: delta.Half_Percent,
	})
	return delta
}

// Half_interval is the 95% confidence half-width on Diff_Percent, from a pooled-variance
// two-sample t-test — poop's score*pooled*normalizer*100/mean. The pooled deviation is
// taken relative to the reference mean, folding in that final divide, so the math never
// forms a raw variance — which, for a metric in the billions, overflows.
func half_interval(
	reference Measurement, candidate Candidate_Measurement, degrees Degree,
) (half Half_Percent) {
	defer func() { Half_Percent_Invariants(half, "half_interval.half") }()
	Measurement_Invariants(reference, "half_interval.reference")
	Candidate_Measurement_Invariants(candidate, "half_interval.candidate")
	Degree_Invariants(degrees, "half_interval.degrees")
	first := fixedpoint.From_Ratio(1, fixedpoint.Denominator(candidate.Sample_Count))
	second := fixedpoint.From_Ratio(1, fixedpoint.Denominator(reference.Sample_Count))
	normalizer := fixedpoint.Number(fixedpoint.Square_Root(first + second))
	pooled := pooled_deviation(reference, candidate, degrees)
	score := student_t_score(degrees)
	band := fixedpoint.Multiply(
		fixedpoint.Multiplicand(score), fixedpoint.Multiplier(pooled),
	)
	band = fixedpoint.Multiply(
		fixedpoint.Multiplicand(band), fixedpoint.Multiplier(normalizer),
	)
	return Half_Percent(band * 100)
}

// Pooled_deviation is the pooled standard deviation as a fraction of the reference mean:
// the root of the degrees-weighted mean of the two relative variances. Dividing each
// deviation by the mean before squaring keeps every value near one, so a metric in the
// billions and its enormous raw variance never overflow.
func pooled_deviation(
	reference Measurement, candidate Candidate_Measurement, degrees Degree,
) (pooled Relative_Deviation) {
	defer func() { Relative_Deviation_Invariants(pooled, "pooled_deviation.pooled") }()
	Measurement_Invariants(reference, "pooled_deviation.reference")
	Candidate_Measurement_Invariants(candidate, "pooled_deviation.candidate")
	Degree_Invariants(degrees, "pooled_deviation.degrees")
	mean := fixedpoint.Number(reference.Mean)
	candidate_deviation := fixedpoint.Divide(
		fixedpoint.Dividend(fixedpoint.Number(candidate.Standard_Deviation)),
		fixedpoint.Divisor(mean),
	)
	reference_deviation := fixedpoint.Divide(
		fixedpoint.Dividend(fixedpoint.Number(reference.Standard_Deviation)),
		fixedpoint.Divisor(mean),
	)
	candidate_variance := fixedpoint.Multiply(
		fixedpoint.Multiplicand(candidate_deviation),
		fixedpoint.Multiplier(candidate_deviation),
	)
	reference_variance := fixedpoint.Multiply(
		fixedpoint.Multiplicand(reference_deviation),
		fixedpoint.Multiplier(reference_deviation),
	)
	weighted := candidate_variance*fixedpoint.Number(candidate.Sample_Count-1) +
		reference_variance*fixedpoint.Number(reference.Sample_Count-1)
	return Relative_Deviation(
		fixedpoint.Square_Root(weighted / fixedpoint.Number(int(degrees))),
	)
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
func Significant_Input_Invariants(input Significant_Input, namespace invariant.Namespace) {
	Difference_Percent_Invariants(input.Diff_Percent, namespace)
	Half_Percent_Invariants(input.Half_Percent, namespace)
	invariant.Always(input.Half_Percent >= 0, "A confidence half-interval is never negative.")
}

// Significant decides whether a difference clears poop's ±1% band: the whole
// confidence interval must sit beyond ±1% with a single sign. The && and || of
// poop's check are split into nested single-term ifs to satisfy the linter.
func significant(input *Significant_Input) (is Significance) {
	defer func() { Significance_Invariants(is, "significant.is") }()
	Significant_Input_Invariants(*input, "significant.input")
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
func Ansi_Code_Invariants(code Ansi_Code, namespace invariant.Namespace) {
	invariant.Tree(code, namespace).
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

// Render_Table_Input carries the report and whether to color it.
type Render_Table_Input struct {
	// Document is the report to render.
	Document Document
	// Color enables ANSI color codes.
	Color Color
}

// Render_Table_Input_Invariants composes the report and states the color flag.
func Render_Table_Input_Invariants(input Render_Table_Input, namespace invariant.Namespace) {
	Document_Invariants(input.Document, namespace)
	Color_Invariants(input.Color, namespace)
}

// Render_Table renders the report as poop's aligned, optionally colored comparison
// table — the default human-readable output. The machine specs block is written
// first, before Benchmark 1.
func Render_Table(input *Render_Table_Input) (output Report) {
	// The rendered bytes are text — printable ASCII, newlines, and ANSI escapes — so a
	// per-byte numeric bundle would demand the null, one, and all-ones bytes a text report
	// never carries. The report's shape is witnessed by its length, not its every byte.
	defer func() { Report_Invariants(output, "Render_Table.output") }()
	Render_Table_Input_Invariants(*input, "Render_Table.input")
	builder := strings.Builder{}
	render_machine_header(&builder, input.Document.Machine)
	for index, benchmark := range input.Document.Benchmarks {
		render_benchmark(&builder, Position(index), benchmark, input.Color)
	}
	return []byte(builder.String())
}

// Render_machine_header writes the host hardware block before the first benchmark.
// Sparse fields (zero values) are omitted, matching the sparse-metric convention
// for table rows.
func render_machine_header(builder *strings.Builder, m Machine_Specs) {
	Machine_Specs_Invariants(m, "render_machine_header.m")
	// An empty CPU model with no architecture means specs were never acquired (e.g.
	// the stub platform); the header is skipped so output stays clean.
	if m.CPU_Model == "" {
		if m.CPU_Arch == "" {
			return
		}
	}
	builder.WriteString("Machine: " + string(m.CPU_Model) + " (" + string(m.CPU_Arch) + ")\n")
	topology := Core_Topology{
		Physical:    m.Physical_Cores,
		Logical:     m.Logical_Cores,
		Performance: m.Performance_Cores,
		Efficiency:  m.Efficiency_Cores,
	}
	builder.WriteString("  cores: " + string(machine_specs_cores(topology)) + "   " +
		"freq: " + string(format_hz(Hertz(m.CPU_Frequency_Hz_Max))) + "   " +
		"ram: " + string(format_bytes(Byte_Size(m.RAM_Total_Bytes))) + "   " +
		"storage: " + string(format_bytes(Byte_Size(m.Storage_Total_Bytes))) + "\n")
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
	if cache_line != "" {
		builder.WriteString("  " + cache_line + operating_system_part + "\n")
	} else {
		builder.WriteString("  " + operating_system_part + "\n")
	}
	builder.WriteString("\n")
}

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
func Cores_Line_Invariants(text Cores_Line, namespace invariant.Namespace) {
	invariant.Tree(text, namespace).Range_Int(len(text), CORES_MIN, CORES_MAX).Ensure()
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
func Core_Topology_Invariants(value Core_Topology, namespace invariant.Namespace) {
	Physical_Core_Count_Invariants(value.Physical, namespace)
	Logical_Core_Count_Invariants(value.Logical, namespace)
	Performance_Core_Count_Invariants(value.Performance, namespace)
	Efficiency_Core_Count_Invariants(value.Efficiency, namespace)
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
		return Frequency(fixedpoint.Format(gigahertz, 2) + " GHz")
	}
	megahertz := fixedpoint.From_Ratio(fixedpoint.Numerator(hz), 1_000_000)
	return Frequency(fixedpoint.Format(megahertz, 0) + " MHz")
}

// Format_bytes renders a byte count, drawn from the machine specs as an unsigned integer, with
// the binary suffix ladder. The raw count is scaled down its rung and only then lifted into
// fixed-point, dividing through a 128-bit ratio: lifting the whole value first would overflow
// the 2^20 scale for a multi-petabyte size — the byte-size ceiling reaches 2^53 — and render a
// garbage, over-width cell. Each rung's raw divisor is recovered from its fixed-point form; the
// suffix is that rung's own, already witnessed by byte_ladder's invariants.
func format_bytes(value Byte_Size) (text Cell) {
	defer func() { Cell_Invariants(text, "format_bytes.text") }()
	Byte_Size_Invariants(value, "format_bytes.value")
	raw := int64(value)
	for _, step := range byte_ladder() {
		rung := int64(fixedpoint.Whole(fixedpoint.Number(step.Divisor)))
		if raw >= rung {
			scaled := fixedpoint.From_Ratio(
				fixedpoint.Numerator(raw), fixedpoint.Denominator(rung),
			)
			return Cell(
				string(format_significant(Quantity(scaled))) + string(step.Suffix),
			)
		}
	}
	integer := fixedpoint.Number(fixedpoint.From_Integer(fixedpoint.Whole_Integer(raw)))
	return Cell(string(format_significant(Quantity(integer))))
}

// ELAPSED_DISPLAY_MAX caps the total sampling time the table header renders. Kiloseconds is the
// widest rung the time ladder reaches, and a span past this scales there to five digits, past the
// header's glyph. The sum of ten thousand ceiling-valued run walls runs to years — the header
// pins it to the widest renderable span while the JSON keeps the exact nanoseconds. The bound is
// four kilosecond digits: 9999 trillion nanoseconds.
const ELAPSED_DISPLAY_MAX = 9999 * 1_000_000_000_000

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
func Span_Invariants(text Span, namespace invariant.Namespace) {
	invariant.Tree(text, namespace).
		Range_Int(len(text), SPAN_BYTES_MIN, SPAN_BYTES_MAX).
		Ensure()
}

// Format_elapsed renders a total sampling duration for the benchmark header with the time
// ladder. Like format_bytes it scales the raw nanoseconds down their rung before lifting into
// fixed-point, so a duration past the 2^20 scale's ceiling — a sum of many large run walls —
// reduces without overflow; and it caps the magnitude first, so an absurd multi-year span still
// lands within the header's glyph rather than rendering a five-digit count. The suffix is the
// rung's own, witnessed by time_ladder.
func format_elapsed(elapsed Elapsed) (text Span) {
	defer func() { Span_Invariants(text, "format_elapsed.text") }()
	Elapsed_Invariants(elapsed, "format_elapsed.elapsed")
	raw := int64(elapsed)
	if raw > ELAPSED_DISPLAY_MAX {
		raw = ELAPSED_DISPLAY_MAX
	}
	for _, step := range time_ladder() {
		rung := int64(fixedpoint.Whole(fixedpoint.Number(step.Divisor)))
		if raw >= rung {
			scaled := fixedpoint.From_Ratio(
				fixedpoint.Numerator(raw), fixedpoint.Denominator(rung),
			)
			return Span(
				string(format_significant(Quantity(scaled))) + string(step.Suffix),
			)
		}
	}
	integer := fixedpoint.Number(fixedpoint.From_Integer(fixedpoint.Whole_Integer(raw)))
	return Span(string(format_significant(Quantity(integer))))
}

// Write_table renders the table and writes it to output, returning EXIT_FAILURE if
// the write fails.
func write_table(output io.Writer, input *Render_Table_Input) (code Benchmark_Exit_Code) {
	defer func() { Benchmark_Exit_Code_Invariants(code, "write_table.exit_code") }()
	Render_Table_Input_Invariants(*input, "write_table.input")
	_, write_err := output.Write(Render_Table(input))
	if write_err != nil {
		return Benchmark_Exit_Code(EXIT_FAILURE)
	}
	return Benchmark_Exit_Code(EXIT_SUCCESS)
}

// COLUMN_NAME_WIDTH is the metric-name column width.
const COLUMN_NAME_WIDTH Extent = 12

// COLUMN_VALUE_WIDTH is the width of each scaled-quantity column.
const COLUMN_VALUE_WIDTH Extent = 8

// COLUMN_OUTLIERS_WIDTH is the outlier-count column width.
const COLUMN_OUTLIERS_WIDTH Extent = 9

// Render_benchmark writes one benchmark's header, its column header, and its metric
// rows. The header and the rows go through render_cells, so labels align with data.
func render_benchmark(
	builder *strings.Builder, index Position, benchmark Benchmark, color Color,
) {
	Position_Invariants(index, "render_benchmark.index")
	Benchmark_Invariants(benchmark, "render_benchmark.benchmark")
	Color_Invariants(color, "render_benchmark.color")
	elapsed := format_elapsed(benchmark.Elapsed)
	words := make([]string, len(benchmark.Command))
	for slot := range benchmark.Command {
		words[slot] = string(benchmark.Command[slot])
	}
	header := fmt.Sprintf("Benchmark %d (%d runs, %s): %s",
		int(index)+1, benchmark.Runs, elapsed, strings.Join(words, " "))
	builder.WriteString(header)
	builder.WriteString("\n")

	columns := render_cells(&Render_Cells_Input{
		Name: "measurement", Mean: "mean", Sigma: "σ",
		Low: "min", High: "max", Outliers: "outliers",
	})
	// The reference is the first benchmark; it is compared against nothing, so its
	// row carries no delta column. Every later benchmark is a candidate.
	if index != 0 {
		columns = columns + "  delta"
	}
	builder.WriteString(string(columns))
	builder.WriteString("\n")

	render_metric_rows(builder, index, benchmark, color)
	builder.WriteString("\n")
}

// Render_metric_rows writes one aligned row per metric, in report order.
func render_metric_rows(
	builder *strings.Builder, index Position, benchmark Benchmark, color Color,
) {
	Position_Invariants(index, "render_metric_rows.index")
	Benchmark_Invariants(benchmark, "render_metric_rows.benchmark")
	Color_Invariants(color, "render_metric_rows.color")
	m := benchmark.Measurements
	d := benchmark.Deltas
	// The reference is first; only later benchmarks carry a comparison.
	has_delta := Delta_Presence(index != 0)
	rows := []Metric_Line_Input{
		{Name: "wall_time", Measurement: Measurement(m.Wall_Time),
			Delta: Delta(d.Wall_Time)},
		{Name: "peak_rss", Measurement: Measurement(m.Peak_RSS),
			Delta: Delta(d.Peak_RSS)},
		{Name: "cpu_cycles", Measurement: Measurement(m.CPU_Cycles),
			Delta: Delta(d.CPU_Cycles)},
		{Name: "instructions", Measurement: Measurement(m.Instructions),
			Delta: Delta(d.Instructions)},
		{
			Name:        "cache_references",
			Measurement: Measurement(m.Cache_References),
			Delta:       Delta(d.Cache_References),
		},
		{
			Name:        "cache_misses",
			Measurement: Measurement(m.Cache_Misses),
			Delta:       Delta(d.Cache_Misses),
		},
		{
			Name:        "branch_misses",
			Measurement: Measurement(m.Branch_Misses),
			Delta:       Delta(d.Branch_Misses),
		},
		{Name: "cpu_user", Measurement: Measurement(m.CPU_User), Delta: Delta(d.CPU_User)},
		{Name: "cpu_system", Measurement: Measurement(m.CPU_System),
			Delta: Delta(d.CPU_System)},
	}
	for row := range rows {
		// A metric with no data on this platform — every value zero, e.g. the
		// Linux-only cache and branch counters on macOS — is left out of the table.
		if rows[row].Measurement.Max == 0 {
			continue
		}
		rows[row].Has_Delta = has_delta
		rows[row].Color = color
		builder.WriteString(string(metric_line(&rows[row])))
		builder.WriteString("\n")
	}
}

// Delta_Presence records whether a table row includes its comparison column.
type Delta_Presence bool

// Delta_Presence_Invariants witnesses both table-row forms.
func Delta_Presence_Invariants(value Delta_Presence, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "The row includes a delta.").
		Ensure()
}

// Metric_Line_Input carries everything one metric row is rendered from.
type Metric_Line_Input struct {
	// Name is the metric's row label.
	Name Caption
	// Measurement is the metric's distribution summary.
	Measurement Measurement
	// Delta is the signed change against the reference.
	Delta Delta
	// Has_Delta is false for the reference row, which carries no delta.
	Has_Delta Delta_Presence
	// Color is whether the delta column is ANSI-colored.
	Color Color
}

// Metric_Line_Input_Invariants states the row's name, distribution, delta, and flags.
func Metric_Line_Input_Invariants(input Metric_Line_Input, namespace invariant.Namespace) {
	Caption_Invariants(input.Name, namespace)
	Measurement_Invariants(input.Measurement, namespace)
	Delta_Invariants(input.Delta, namespace)
	Delta_Presence_Invariants(input.Has_Delta, namespace)
	Color_Invariants(input.Color, namespace)
}

// Metric_line renders one metric row: its name, scaled mean ± σ, min … max, outlier
// count, and — when the benchmark is not the reference — its delta.
func metric_line(input *Metric_Line_Input) (text Full_Row) {
	defer func() { Full_Row_Invariants(text, "metric_line.line") }()
	Metric_Line_Input_Invariants(*input, "metric_line.input")
	measurement := input.Measurement
	unit := measurement.Unit
	outlier_percent := fixedpoint.Number(0)
	if measurement.Sample_Count > 0 {
		outlier_percent = fixedpoint.From_Ratio(
			fixedpoint.Numerator(measurement.Outlier_Count)*100,
			fixedpoint.Denominator(measurement.Sample_Count),
		)
	}
	outlier_text := strconv.Itoa(int(measurement.Outlier_Count)) +
		" (" + string(fixedpoint.Format(outlier_percent, 0)) + "%)"
	row := render_cells(&Render_Cells_Input{
		Name: input.Name,
		Mean: Mean_Cell(format_quantity(Quantity(measurement.Mean), unit)),
		Sigma: Deviation_Cell(format_quantity(
			Quantity(measurement.Standard_Deviation), unit,
		)),
		Low:      Minimum_Cell(format_quantity(Quantity(measurement.Min), unit)),
		High:     Maximum_Cell(format_quantity(Quantity(measurement.Max), unit)),
		Outliers: Outliers(outlier_text),
	})
	text = Full_Row(row)
	if input.Has_Delta {
		delta := delta_render(input.Delta, input.Color)
		text = Full_Row(string(text) + "  " + string(delta))
	}
	return text
}

// Mean_Cell is one rendered mean value.
type Mean_Cell string

// Mean_Cell_Invariants bounds a rendered mean value.
func Mean_Cell_Invariants(value Mean_Cell, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), CELL_BYTES_MIN, CELL_BYTES_MAX).
		Ensure()
}

// Deviation_Cell is one rendered standard-deviation value.
type Deviation_Cell string

// Deviation_Cell_Invariants bounds a rendered standard-deviation value.
func Deviation_Cell_Invariants(value Deviation_Cell, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), CELL_BYTES_MIN, CELL_BYTES_MAX).
		Ensure()
}

// Minimum_Cell is one rendered minimum value.
type Minimum_Cell string

// Minimum_Cell_Invariants bounds a rendered minimum value.
func Minimum_Cell_Invariants(value Minimum_Cell, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), CELL_BYTES_MIN, CELL_BYTES_MAX).
		Ensure()
}

// Maximum_Cell is one rendered maximum value.
type Maximum_Cell string

// Maximum_Cell_Invariants bounds a rendered maximum value.
func Maximum_Cell_Invariants(value Maximum_Cell, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Render_Cells_Input_Invariants(input Render_Cells_Input, namespace invariant.Namespace) {
	Caption_Invariants(input.Name, namespace)
	Mean_Cell_Invariants(input.Mean, namespace)
	Deviation_Cell_Invariants(input.Sigma, namespace)
	Minimum_Cell_Invariants(input.Low, namespace)
	Maximum_Cell_Invariants(input.High, namespace)
	Outliers_Invariants(input.Outliers, namespace)
}

// Render_cells lays one row — header or data — into aligned, uncolored columns.
// Header and data share this layout, so a label always sits above its column.
func render_cells(input *Render_Cells_Input) (text Bare_Row) {
	defer func() { Bare_Row_Invariants(text, "render_cells.line") }()
	Render_Cells_Input_Invariants(*input, "render_cells.input")
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
func Cell_Invariants(text Cell, namespace invariant.Namespace) {
	invariant.Tree(text, namespace).
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
func Glyph_Invariants(text Glyph, namespace invariant.Namespace) {
	invariant.Tree(text, namespace).
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
func Column_Invariants(text Column, namespace invariant.Namespace) {
	invariant.Tree(text, namespace).Range_Int(len(text), COLUMN_MIN, COLUMN_MAX).Ensure()
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
func Caption_Invariants(text Caption, namespace invariant.Namespace) {
	invariant.Tree(text, namespace).Range_Int(len(text), CAPTION_MIN, CAPTION_MAX).Ensure()
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
func Outliers_Invariants(text Outliers, namespace invariant.Namespace) {
	invariant.Tree(text, namespace).Range_Int(len(text), OUTLIERS_MIN, OUTLIERS_MAX).Ensure()
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
func Padded_Invariants(text Padded, namespace invariant.Namespace) {
	invariant.Tree(text, namespace).Range_Int(len(text), PADDED_MIN, PADDED_MAX).Ensure()
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
func Bare_Row_Invariants(text Bare_Row, namespace invariant.Namespace) {
	invariant.Tree(text, namespace).Range_Int(len(text), BARE_ROW_MIN, BARE_ROW_MAX).Ensure()
}

// FULL_ROW_MIN is the narrowest complete metric row, the reference's bare row with no delta.
const FULL_ROW_MIN = 69

// FULL_ROW_MAX bounds a complete metric row: the widest data row, two spacer bytes, and the
// widest colored delta text. The widest tally and widest delta come from different runs, so the
// maximum is a near-ceiling-scale change with color rather than the two summed.
const FULL_ROW_MAX = 104

// Full_Row is one complete metric row as written to the report — an assembled row plus its
// delta column when the benchmark is a candidate. A distinct type, wider than the bare row.
type Full_Row string

// Full_Row_Invariants bounds a complete row's length; always many bytes, so the empty, one,
// and two-byte boundaries are unreachable while the min and max are witnessed.
func Full_Row_Invariants(text Full_Row, namespace invariant.Namespace) {
	invariant.Tree(text, namespace).Range_Int(len(text), FULL_ROW_MIN, FULL_ROW_MAX).Ensure()
}

// DELTA_BODY_MIN is the narrowest delta body, a sign over two padded percentages.
const DELTA_BODY_MIN = 16

// DELTA_BODY_MAX bounds the unpainted delta body: the change percentage pinned to the display
// cap ("9999999.0") with the widest confidence half-interval a bounded t-test reaches, which is
// only a few hundred percent — the pooled deviation relative to the mean caps the interval well
// below the display clamp, so the second percentage never widens to the first's.
const DELTA_BODY_MAX = 20

// Delta_Body is the uncolored delta text — a sign, a percentage, its confidence interval —
// before any ANSI color wraps it. A distinct type from the painted form, which runs wider.
type Delta_Body string

// Delta_Body_Invariants bounds the body's length; always many bytes, so the empty, one, and
// two-byte boundaries are unreachable while the min and max are witnessed.
func Delta_Body_Invariants(text Delta_Body, namespace invariant.Namespace) {
	invariant.Tree(text, namespace).
		Range_Int(len(text), DELTA_BODY_MIN, DELTA_BODY_MAX).
		Ensure()
}

// DELTA_TEXT_MIN is the narrowest rendered delta, an uncolored body.
const DELTA_TEXT_MIN = 16

// DELTA_TEXT_MAX bounds the rendered delta: the widest body wrapped in a color code and its
// reset — the body's max plus the nine bytes of the color escape and reset sequence.
const DELTA_TEXT_MAX = 29

// Delta_Text is the rendered delta column as written — the body, optionally wrapped in an ANSI
// color and its reset. A distinct type, wider than the bare body by the escape sequences.
type Delta_Text string

// Delta_Text_Invariants bounds the rendered delta's length; always many bytes, so the empty,
// one, and two-byte boundaries are unreachable while the min and max are witnessed.
func Delta_Text_Invariants(text Delta_Text, namespace invariant.Namespace) {
	invariant.Tree(text, namespace).
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
func Frequency_Invariants(text Frequency, namespace invariant.Namespace) {
	invariant.Tree(text, namespace).
		Range_Holed_Int(
			len(text), FREQUENCY_BYTES_MIN, FREQUENCY_BYTES_MAX, 2, 2, 2, 2).
		Ensure()
}

// SUFFIX_BYTES_MIN is the empty base unit a suffix floors at.
const SUFFIX_BYTES_MIN = 0

// SUFFIX_BYTES_BARE is the length of the bare unit letters like "s" and "B".
const SUFFIX_BYTES_BARE = 1

// SUFFIX_BYTES_PREFIXED is the length of a scale prefix over a unit letter, like "ms".
const SUFFIX_BYTES_PREFIXED = 2

// SUFFIX_BYTES_MAX is the longest a unit suffix is: the binary byte suffixes like "MiB".
const SUFFIX_BYTES_MAX = 3

// Suffix is a unit suffix on a scaled quantity — "", "s", "ms", "MiB". A distinct type
// so the suffix ladder's trusted text carries a length invariant.
type Suffix string

// Suffix_Invariants holds a suffix's length to the four lengths a suffix has, and witnesses
// each length. A suffix spans the empty base unit through the three-byte binary suffixes, so
// the two interior lengths are members in their own right.
func Suffix_Invariants(unit Suffix, namespace invariant.Namespace) {
	invariant.Tree(unit, namespace).
		Enum_4_Int(
			len(unit), SUFFIX_BYTES_MIN, SUFFIX_BYTES_BARE, SUFFIX_BYTES_PREFIXED,
			SUFFIX_BYTES_MAX).
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
func Phase_Invariants(name Phase, namespace invariant.Namespace) {
	invariant.Tree(name, namespace).
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
func Extent_Invariants(value Extent, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Kept_Invariants(value Kept, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Degree_Invariants(value Degree, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Census_Invariants(value Census, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Divisor_Invariants(value Divisor, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Tally_Invariants(value Tally, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Strays_Invariants(value Strays, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Position_Invariants(value Position, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Exit_Code_Invariants(value Exit_Code, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Int(int(value), EXIT_CODE_MIN, int(EXIT_FAILURE), EXIT_CODE_MAX).
		Ensure()
}

// Benchmark_Exit_Code is the success or failure result after argument parsing.
type Benchmark_Exit_Code Exit_Code

// Benchmark_Exit_Code_Invariants holds benchmark work to success or failure.
func Benchmark_Exit_Code_Invariants(
	value Benchmark_Exit_Code, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
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
func Exit_Status_Invariants(value Exit_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Failure_Status_Invariants(value Failure_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Cores_Invariants(value Cores, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Hertz_Invariants(value Hertz, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Byte_Size_Invariants(value Byte_Size, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Gap_Invariants(value Gap, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Accumulator_High_Invariants(high Accumulator_High, namespace invariant.Namespace) {
	invariant.Tree(high, namespace).
		Range_Uint64(uint64(high), ACCUMULATOR_MIN, ACCUMULATOR_HIGH_MAX).
		Ensure()
}

// Accumulator_Subtotal_High is the high word before one more squared deviation is added.
type Accumulator_Subtotal_High uint64

// Accumulator_Subtotal_High_Invariants bounds the high word of a partial sum.
func Accumulator_Subtotal_High_Invariants(
	high Accumulator_Subtotal_High, namespace invariant.Namespace,
) {
	invariant.Tree(high, namespace).
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
func Accumulator_Low_Invariants(low Accumulator_Low, namespace invariant.Namespace) {
	invariant.Tree(low, namespace).
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
func Metric_Invariants(value Metric, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Average_Invariants(value Average, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Points_Invariants(values Points, namespace invariant.Namespace) {
	invariant.Tree(values, namespace).Range_Int(len(values), QUORUM_MIN, POINTS_MAX).Ensure()
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
func Label_Invariants(text Label, namespace invariant.Namespace) {
	invariant.Tree(text, namespace).
		Range_Int(len(text), LABEL_BYTES_MIN, LABEL_BYTES_MAX).
		Ensure()
}

// Right_Alignment records which side receives a column's padding.
type Right_Alignment bool

// Right_Alignment_Invariants witnesses both column alignments.
func Right_Alignment_Invariants(value Right_Alignment, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "The column is right-aligned.").
		Ensure()
}

// Pad aligns text to a visible width with spaces — on the left when right is set, so the
// text right-aligns, on the right otherwise. The width is the rune count, so a multibyte
// glyph like σ still counts as one column. One padder, so every column width passes one site
// and the width and padded-line invariants see the whole layout's range, not a per-side slice.
func pad(text Column, width Extent, right Right_Alignment) (result Padded) {
	defer func() { Padded_Invariants(result, "pad.padded") }()
	Column_Invariants(text, "pad.text")
	Extent_Invariants(width, "pad.width")
	Right_Alignment_Invariants(right, "pad.right")
	space := int(width) - utf8.RuneCountInString(string(text))
	if space < 0 {
		return Padded(text)
	}
	if right {
		return Padded(strings.Repeat(" ", space) + string(text))
	}
	return Padded(string(text) + strings.Repeat(" ", space))
}

// PERCENT_DISPLAY_MAX caps a rendered percentage's magnitude at the seven-digit ceiling the
// delta column was sized against. Format at one decimal yields at most "9999999.0" — nine
// bytes — so a sign, two such values, and the "% ± " and "%" glue land exactly at
// DELTA_BODY_MAX; a larger percentage is pinned here rather than overrunning the fixed layout.
// The bound is written as an integer times the scale so it stays a compile-time constant.
const PERCENT_DISPLAY_MAX fixedpoint.Number = 9_999_999 * fixedpoint.SCALE

// Delta_render formats one metric's change: a sign, the percentage, and its
// confidence half-interval. A significant change is colored — red slower, green
// faster — while an insignificant one stays faint.
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
	half_percent := fixedpoint.Number(delta.Half_Percent)
	// Pin both percentages to the column's widest value. A change past ten million percent —
	// a candidate a hundred-thousand-fold off the reference — cannot fit the fixed delta
	// layout the body invariant sizes for, and its exact magnitude past the bound is noise;
	// without the clamp its digits overrun DELTA_BODY_MAX and trip the guard.
	if difference > PERCENT_DISPLAY_MAX {
		difference = PERCENT_DISPLAY_MAX
	}
	if half_percent > PERCENT_DISPLAY_MAX {
		half_percent = PERCENT_DISPLAY_MAX
	}
	diff := pad(Column(fixedpoint.Format(difference, 1)), 5, true)
	half := pad(Column(fixedpoint.Format(half_percent, 1)), 4, true)
	body := Delta_Body(sign + string(diff) + "% ± " + string(half) + "%")
	return paint(body, code, color)
}

// Quantity is a nonnegative fixed-point value prepared for human-readable scaling.
type Quantity fixedpoint.Number

// Quantity_Invariants states the zero, unit, and larger rendering regions.
func Quantity_Invariants(value Quantity, namespace invariant.Namespace) {
	invariant.Always(value >= 0, "A rendered quantity cannot be negative.")
	invariant.Tree(value, namespace).
		Enum_3_Int64(min(int64(value)/fixedpoint.SCALE, 2), 0, 1, 2).
		Ensure()
}

// Scale_Step is one rung of a scaling ladder: the divisor above which the suffix applies.
type Scale_Step struct {
	// Divisor is the magnitude the value is divided by at this rung.
	Divisor Quantity
	// Suffix names the unit at this rung — empty at the base rung, so it is a suffix, not a
	// cell, whose length range admits the empty base unit.
	Suffix Suffix
}

// Scale_Step_Invariants states a rung's suffix; the divisor has no preset of its own.
func Scale_Step_Invariants(step Scale_Step, namespace invariant.Namespace) {
	Quantity_Invariants(step.Divisor, namespace)
	Suffix_Invariants(step.Suffix, namespace)
}

// Format_quantity renders a raw value scaled to a human unit with three significant
// figures — poop's printUnit, e.g. 14906807 nanoseconds becomes "14.9ms".
func format_quantity(value Quantity, unit Unit) (text Cell) {
	defer func() { Cell_Invariants(text, "format_quantity.text") }()
	Quantity_Invariants(value, "format_quantity.value")
	Unit_Invariants(unit, "format_quantity.unit")
	scaled, unit_suffix := scale_quantity(value, unit)
	return Cell(string(format_significant(scaled)) + string(unit_suffix))
}

// Scale_quantity divides a value down to its human magnitude and names the unit
// suffix, dispatching on the metric's unit.
func scale_quantity(
	value Quantity, unit Unit,
) (scaled Quantity, unit_suffix Suffix) {
	defer func() {
		Quantity_Invariants(scaled, "scale_quantity.scaled")
		Suffix_Invariants(unit_suffix, "scale_quantity.suffix")
	}()
	Quantity_Invariants(value, "scale_quantity.value")
	Unit_Invariants(unit, "scale_quantity.unit")
	switch unit {
	case "nanoseconds":
		return scale_ladder(value, time_ladder())
	case "bytes":
		return scale_ladder(value, byte_ladder())
	}
	return scale_ladder(value, count_ladder())
}

// Scale_ladder divides a value by the first rung it reaches or above, naming that rung's
// suffix; below the lowest rung it stays in the base unit.
func scale_ladder(
	value Quantity, ladder Ladder,
) (scaled Quantity, unit_suffix Suffix) {
	defer func() {
		Quantity_Invariants(scaled, "scale_ladder.scaled")
		Suffix_Invariants(unit_suffix, "scale_ladder.suffix")
	}()
	Quantity_Invariants(value, "scale_ladder.value")
	Ladder_Invariants(ladder, "scale_ladder.ladder")
	for _, step := range ladder {
		if value >= step.Divisor {
			scaled = Quantity(fixedpoint.Divide(
				fixedpoint.Dividend(value), fixedpoint.Divisor(step.Divisor),
			))
			return scaled, Suffix(step.Suffix)
		}
	}
	return value, ""
}

// Time_ladder is the nanosecond-to-kilosecond ladder; the base unit is ns.
func time_ladder() (ladder Ladder) {
	defer func() { Ladder_Invariants(ladder, "time_ladder.ladder") }()
	return Ladder{
		{
			Divisor: Quantity(fixedpoint.From_Integer(1_000_000_000_000)),
			Suffix:  "ks",
		},
		{Divisor: Quantity(fixedpoint.From_Integer(1_000_000_000)), Suffix: "s"},
		{Divisor: Quantity(fixedpoint.From_Integer(1_000_000)), Suffix: "ms"},
		{Divisor: Quantity(fixedpoint.From_Integer(1_000)), Suffix: "us"},
		{Divisor: Quantity(fixedpoint.From_Integer(1)), Suffix: "ns"},
	}
}

// Byte_ladder is the binary (1024) ladder with IEC suffixes, since memory is a
// power-of-two quantity; the base unit is B.
func byte_ladder() (ladder Ladder) {
	defer func() { Ladder_Invariants(ladder, "byte_ladder.ladder") }()
	return Ladder{
		{
			Divisor: Quantity(
				fixedpoint.From_Integer(1024 * 1024 * 1024 * 1024),
			),
			Suffix: "TiB",
		},
		{
			Divisor: Quantity(fixedpoint.From_Integer(1024 * 1024 * 1024)),
			Suffix:  "GiB",
		},
		{Divisor: Quantity(fixedpoint.From_Integer(1024 * 1024)), Suffix: "MiB"},
		{Divisor: Quantity(fixedpoint.From_Integer(1024)), Suffix: "KiB"},
		{Divisor: Quantity(fixedpoint.From_Integer(1)), Suffix: "B"},
	}
}

// Count_ladder is the metric-prefix ladder for a bare count; the base unit is unnamed.
func count_ladder() (ladder Ladder) {
	defer func() { Ladder_Invariants(ladder, "count_ladder.ladder") }()
	return Ladder{
		{
			Divisor: Quantity(fixedpoint.From_Integer(1_000_000_000_000)),
			Suffix:  "T",
		},
		{Divisor: Quantity(fixedpoint.From_Integer(1_000_000_000)), Suffix: "G"},
		{Divisor: Quantity(fixedpoint.From_Integer(1_000_000)), Suffix: "M"},
		{Divisor: Quantity(fixedpoint.From_Integer(1_000)), Suffix: "K"},
	}
}

// Format_significant renders a scaled value to three significant figures: whole
// numbers and hundreds with no decimals, tens with one, units with two.
func format_significant(value Quantity) (text Glyph) {
	defer func() { Glyph_Invariants(text, "format_significant.text") }()
	Quantity_Invariants(value, "format_significant.value")
	if value >= Quantity(fixedpoint.From_Integer(1000)) {
		return Glyph(fixedpoint.Format(fixedpoint.Number(value), 0))
	}
	if fixedpoint.Is_Integer(fixedpoint.Number(value)) {
		return Glyph(fixedpoint.Format(fixedpoint.Number(value), 0))
	}
	if value >= Quantity(fixedpoint.From_Integer(100)) {
		return Glyph(fixedpoint.Format(fixedpoint.Number(value), 0))
	}
	if value >= Quantity(fixedpoint.From_Integer(10)) {
		return Glyph(fixedpoint.Format(fixedpoint.Number(value), 1))
	}
	return Glyph(fixedpoint.Format(fixedpoint.Number(value), 2))
}

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
func Progress_Total_Invariants(value Progress_Total, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(
			int64(value),
			LIMIT_MIN,
			LIMIT_MAX,
		).
		Ensure()
}

// Render_Progress_Input is one progress update: the command being sampled, how long
// it has been sampling, and how many runs are done against the cap.
type Render_Progress_Input struct {
	// Command is the command being sampled.
	Command sysio.Process_Request
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
func Render_Progress_Input_Invariants(input Render_Progress_Input, namespace invariant.Namespace) {
	Elapsed_Invariants(input.Elapsed, namespace)
	Phase_Invariants(input.Phase, namespace)
	Census_Invariants(input.Count, namespace)
	Progress_Total_Invariants(input.Total, namespace)
}

// Render_progress writes an in-place progress line to stderr: elapsed seconds, an
// optional phase word (warmup), the counter (count over its total, or just the count
// when the total is disabled), and the command. Gated by Progress at the call site.
func render_progress(stderr io.Writer, input *Render_Progress_Input) {
	Render_Progress_Input_Invariants(*input, "render_progress.input")
	seconds := fixedpoint.From_Ratio(
		fixedpoint.Numerator(input.Elapsed), fixedpoint.Denominator(time.SECOND),
	)
	counter := strconv.Itoa(int(input.Count))
	if input.Total > 0 {
		counter = counter + "/" + strconv.FormatInt(int64(input.Total), 10)
	}
	phase_text := ""
	if input.Phase != "" {
		phase_text = string(input.Phase) + " "
	}
	command_label := progress_label(command_words(input.Command))
	output := PROGRESS_CLEAR + string(fixedpoint.Format(seconds, 1)) + "s  " +
		phase_text + counter + "  " + string(command_label)
	stderr.Write([]byte(output))
}

// Progress_label joins the command words and truncates them to keep the progress
// line on one terminal row.
func progress_label(words Command_Line) (text Label) {
	defer func() { Label_Invariants(text, "progress_label.label") }()
	Command_Line_Invariants(words, "progress_label.words")
	parts := make([]string, len(words))
	for index, word := range words {
		Command_Word_Invariants(word, "progress_label.word")
		parts[index] = string(word)
	}
	joined := strings.Join(parts, " ")
	runes := []rune(joined)
	if len(runes) <= PROGRESS_LABEL_RUNES_MAX {
		return Label(joined)
	}
	return Label(string(runes[:PROGRESS_LABEL_RUNES_MAX-1]) + "…")
}
