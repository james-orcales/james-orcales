// Package simulation_test is maddox's blackbox witness: a single fuzz target
// drives internal.Main end to end — sampling, statistics, comparison, and
// rendering — so every production invariant in the maddox library is exercised
// through the one public entry point, never by reaching into a helper. Under a
// plain `go test` the seed corpus is replayed and the invariant recorder judges
// coverage; under `-fuzz` the same driver explores.
package simulation_test

import (
	"fmt"
	"testing"

	"local/james-orcales/maddox/internal"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/nbio"
	"local/james-orcales/shared/sim/time"
)

// The requirements tests mirror REQUIREMENTS.md: one leaf per heading, in heading
// order, each driving internal.Main through the shared scenario driver. They assert no
// invariant trips; the coverage recorder judges the boundaries the fuzz target reaches.

// Test_Witness_Pipeline drives a full benchmark run — sampling, statistics, comparison,
// and rendering — through Main from the base scenario.
func Test_Witness_Pipeline(t *testing.T) {
	drive(t, decode_scenario(build(base_scenario())))
}

// Test_Witness_Failure drives a command that exits non-zero, exercising the abort and
// captured-stderr paths a successful run never reaches.
func Test_Witness_Failure(t *testing.T) {
	drive(t, decode_scenario(with(func(s *scenario) {
		s.Exit = 1
		s.Fail_Every = 1
		s.Stderr_Bytes = 4
	})))
}

// SIM_DELTAS_MAX is the per-sample deviation pattern length — a driver construct, not a
// maddox bound, so it is the one ceiling defined here rather than imported from the library.
// Every other structural ceiling is maddox's own exported const, so the driver cannot drift
// from the bounds it is meant to exercise.
const SIM_DELTAS_MAX = 32

// SIM_TEXTS_MAX bounds the explicit positional texts a scenario passes verbatim.
const SIM_TEXTS_MAX = 4

// SIM_ENVIRONMENT_MAX is the inherited environment's ceiling.
const SIM_ENVIRONMENT_MAX = maddox.ENVIRONMENT_COUNT_MAX

// Fuzz_Main is the sole witness. It decodes the fuzz bytes into a benchmark
// scenario and drives internal.Main; the seed corpus is a battery of honest
// end-to-end runs, each reaching a real boundary of the pipeline.
func Fuzz_Main(f *testing.F) {
	for _, seed := range seed_corpus() {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		drive(t, decode_scenario(data))
	})
}

// Scenario is one benchmark run fully described: the commands, the sampling
// limits, and the per-sample values the injected Sampler will report. Every
// field is decoded totally from the fuzz bytes so any input is a valid run.
type scenario struct {
	// Commands is the command count, 0..COMMAND_SET_MAX.
	Commands int
	// Words is the word count in each command line, WORD_MIN..COMMAND_WORDS_MAX.
	Words int
	// Word_Bytes is the byte length of each argument word, 0..COMMAND_WORD_BYTES_MAX.
	Word_Bytes int
	// Path_Bytes, for a single-command run, sets the executable's byte length, so the
	// progress label reaches its short shapes; 0 keeps the default distinct path.
	Path_Bytes int
	// Runs is Runs_Max, the full int range (Main caps kept runs internally).
	Runs int
	// Warmup is Warmup_Count, the full int range.
	Warmup int
	// Duration is the parsed time budget, in seconds, the full signed range.
	Duration int64
	// Base is the center every metric's samples deviate around.
	Base int64
	// Deltas is the repeating deviation pattern added to Base.
	Deltas []int64
	// Deltas_Command is the 1-based command the deviation pattern applies to; 0 applies it to
	// every command.
	Deltas_Command int
	// Center_Count pins the first N runs to Base exactly — the tight median cluster whose zero
	// interquartile range turns every remaining spread sample into a Tukey outlier.
	Center_Count int
	// Sleep is the clock grains each run advances — the wall time.
	Sleep int64
	// Exit is the exit code the sampler reports.
	Exit int
	// Fail_Every makes every Nth run report Exit; 0 disables.
	Fail_Every int
	// Fail_Command is the 1-based command whose runs report Exit; 0 disables. It fails a
	// specific benchmark, so the failure diagnostic sees that benchmark's position.
	Fail_Command int
	// Divergence shifts each command's metrics by its index times this, so later
	// commands measure higher than the reference and the comparison turns significant.
	Divergence int64
	// Stderr_Bytes is the captured stderr a failing run carries.
	Stderr_Bytes int
	// Environment_Count is how many entries the inherited environment holds.
	Environment_Count int
	// Texts, when present, are the positional command texts passed verbatim.
	Texts []string
	// Allow_Fail keeps benchmarking a command that exits non-zero.
	Allow_Fail bool
	// Format selects the report rendering.
	Format uint8
	// Color enables ANSI color in the table.
	Color bool
	// Progress writes the run counter while sampling.
	Progress bool
	// Broken_Output makes the report writer fail, driving the render failure exit.
	Broken_Output bool
	// Wall_Pattern applies the sample deviation and command divergence to wall time.
	Wall_Pattern bool
	// Path_Wide makes a single command's executable four-byte runes, so the progress label
	// reaches its byte ceiling — the rune cap times four bytes.
	Path_Wide bool
	// Arguments_Max fills the operating-system argument list to its declared ceiling.
	Arguments_Max bool
	// Assignment prefixes each command with an environment assignment.
	Assignment bool
	// Progress_Never states the progress mode as never rather than leaving it automatic.
	Progress_Never bool
	// Spawn_Fail makes every spawn fail to start.
	Spawn_Fail bool
	// Machine is the injected host snapshot.
	Machine maddox.Machine_Specs
}

// DRIVE_PASS_COUNT bounds delivery passes past two full commands at the sample ceiling and their
// warmups.
const DRIVE_PASS_COUNT = 1 << 20

// DRIVE_OUTPUT_BYTES holds any report the scenarios render.
const DRIVE_OUTPUT_BYTES = 1 << 22

// SIMULATION_YEAR is the monotonic clock's end, which no simulated run passes.
const SIMULATION_YEAR = int64(time.MONOTONIC_MOMENT_MAXIMUM)

// The simulated loop holds every spawn and write until the driver delivers it, the most recent
// first, so a progress line stays in flight across runs. The clock is the sum of the delivered
// runs' walls, so the budget stopwatch advances without a real clock.
type simulation struct {
	Scenario      scenario
	Run           int
	Command_Index int
	Previous      string
	Now           time.Monotonic_Moment
	Submissions   []submission
}

// Submission is one submission the loop has yet to retire: a spawn, or a write to Stream.
type submission struct {
	Spawn      bool
	Completion *nbio.Completion
	Request    nbio.Process_Request
	Process    nbio.Process_Callback
	Stream     nbio.Stream
	Mode       nbio.Stream_Mode
	Buffer     []byte
	Offset     int64
	Whence     nbio.Seek_From
	Callback   nbio.Stream_Callback
}

// Held is a stream whose operations wait in the simulated loop until delivered.
type held struct {
	Loop  *simulation
	Inner nbio.Stream
}

// Drive runs the scenario end to end through Main and the loop, asking the runner at every point
// the root does. Only the invariants observed along the way matter; the report is kept only so a
// broken sink can fail its write.
func drive(t *testing.T, s scenario) {
	t.Helper()
	loop := &simulation{Scenario: s, Command_Index: -1}
	io := nbio.IO{}
	io.Spawn = simulation_spawn
	io.Storage.State = nbio.State(loop)
	output := nbio.Stream_Memory{Memory: make([]byte, DRIVE_OUTPUT_BYTES)}
	// A report sink with no room fails every write, driving the report failure exit.
	if s.Broken_Output {
		output.Memory = nil
	}
	error_output := nbio.Stream_Memory{Memory: make([]byte, DRIVE_OUTPUT_BYTES)}
	held_output := held{Loop: loop, Inner: nbio.Memory_To_Stream(&output)}
	held_error_output := held{Loop: loop, Inner: nbio.Memory_To_Stream(&error_output)}
	environment := make(maddox.Inherited_Environment, s.Environment_Count)
	for index := range environment {
		environment[index] = fmt.Sprintf("E%d=v", index)
	}
	start := maddox.Main(maddox.Main_Input{
		Arguments:                scenario_arguments(s),
		Environment:              environment,
		Output_Is_Terminal:       scenario_output_terminal(s),
		Error_Output_Is_Terminal: scenario_error_output_terminal(s),
	})
	runner := maddox.Runner{
		IO: io,
		Clock: time.Clock{
			State: time.State(loop), Now_Monotonic: simulation_now,
			Now_Realtime: simulation_realtime,
		},
		Output: nbio.Stream{State: nbio.State(&held_output),
			Procedure: held_procedure},
		Error_Output: nbio.Stream{State: nbio.State(&held_error_output),
			Procedure: held_procedure},
		Machine: s.Machine,
		Start:   start,
	}
	for pass_index := 0; pass_index < DRIVE_PASS_COUNT; pass_index++ {
		maddox.Work_Queued(&runner)
		if maddox.Stopped(&runner) {
			// A stopped run has nothing left to rearm.
			maddox.Rearm(&runner)
			return
		}
		// The first rearm judges a retired run, which a full sample store holds until the
		// next one reduces the command, so the root asks between them.
		if maddox.Rearm(&runner) {
			maddox.Work_Queued(&runner)
			maddox.Stopped(&runner)
			for maddox.Rearm(&runner) {
			}
		}
		maddox.Work_Queued(&runner)
		if maddox.Stopped(&runner) {
			continue
		}
		if !simulation_deliver(loop) {
			t.Fatal("the simulated run waits on nothing")
		}
	}
	t.Fatal("the simulated run never stopped")
}

func held_procedure(
	state nbio.State, completion *nbio.Completion, mode nbio.Stream_Mode, buffer []byte,
	offset int64, whence nbio.Seek_From, callback nbio.Stream_Callback,
) {
	stream := state.(*held)
	stream.Loop.Submissions = append(stream.Loop.Submissions, submission{
		Completion: completion, Stream: stream.Inner, Mode: mode, Buffer: buffer,
		Offset: offset, Whence: whence, Callback: callback,
	})
}

func simulation_spawn(
	state nbio.State, completion *nbio.Completion, request nbio.Process_Request,
	_ time.Duration, callback nbio.Process_Callback,
) {
	loop := state.(*simulation)
	loop.Submissions = append(loop.Submissions, submission{
		Spawn: true, Completion: completion, Request: request, Process: callback,
	})
}

// Retires the most recent submission. A write reaches its memory sink; a spawn synthesizes one
// run's measurements from the scenario. Commands are measured in sequence, each with a distinct
// path, so a path change marks the next benchmark and resets the run counter.
func simulation_deliver(loop *simulation) (delivered bool) {
	if len(loop.Submissions) == 0 {
		return false
	}
	next := loop.Submissions[len(loop.Submissions)-1]
	loop.Submissions = loop.Submissions[:len(loop.Submissions)-1]
	if !next.Spawn {
		next.Stream.Procedure(next.Stream.State, next.Completion, next.Mode, next.Buffer,
			next.Offset, next.Whence, next.Callback)
		return true
	}
	s := loop.Scenario
	command := next.Request
	if command.Path != loop.Previous {
		loop.Previous = command.Path
		loop.Command_Index++
		loop.Run = 0
	}
	if s.Spawn_Fail {
		loop.Run++
		next.Process(next.Completion, nbio.Process_Result{}, nbio.Canceled)
		return true
	}
	center := s.Base + int64(loop.Command_Index)*s.Divergence
	result := usage_from(sample_value(s, center, loop.Command_Index, loop.Run))
	// Wall is this run's tight cost, extracted as a Metric into the statistics, so it carries
	// the same representable-ceiling contract as every other metric and passes through the same
	// clamp. A run costs its wall on the virtual clock, which ends at a year of uptime.
	wall := s.Sleep
	if s.Wall_Pattern {
		wall_center := s.Sleep + int64(loop.Command_Index)*s.Divergence
		wall = sample_value(s, wall_center, loop.Command_Index, loop.Run)
	}
	wall = min(metric_clamp(wall), SIMULATION_YEAR-int64(loop.Now))
	result.Usage.Wall = time.Duration(wall)
	loop.Now += time.Monotonic_Moment(wall)
	fail := false
	if s.Fail_Command > 0 {
		if loop.Command_Index == s.Fail_Command-1 {
			fail = true
		}
	}
	if !fail {
		if s.Fail_Every > 0 {
			if loop.Run%s.Fail_Every == 0 {
				fail = true
			}
		}
	}
	if fail {
		result.Exit = s.Exit
		result.Error_Output = make([]byte, s.Stderr_Bytes)
	}
	loop.Run++
	next.Process(next.Completion, result, nil)
	return true
}

func simulation_now(state time.State) (moment time.Monotonic_Moment) {
	return state.(*simulation).Now
}

func simulation_realtime(state time.State) (moment time.Moment) {
	return time.Moment(state.(*simulation).Now)
}

// Scenario_output_terminal resolves the report stream's simulated terminal state.
func scenario_output_terminal(s scenario) (status maddox.Output_Terminal_Status) {
	if s.Color {
		return maddox.Output_Terminal_Status(maddox.TERMINAL_STATUS_TERMINAL)
	}
	return maddox.Output_Terminal_Status(maddox.TERMINAL_STATUS_NOT_TERMINAL)
}

// Scenario_error_output_terminal resolves the diagnostic stream's terminal state.
func scenario_error_output_terminal(s scenario) (status maddox.Error_Output_Terminal_Status) {
	if s.Progress {
		return maddox.Error_Output_Terminal_Status(maddox.TERMINAL_STATUS_TERMINAL)
	}
	return maddox.Error_Output_Terminal_Status(maddox.TERMINAL_STATUS_NOT_TERMINAL)
}

// Scenario_arguments renders the scenario through the public command-line boundary.
func scenario_arguments(s scenario) (arguments maddox.Arguments) {
	if s.Arguments_Max {
		arguments = make(maddox.Arguments, maddox.ARGUMENTS_COUNT_MAX)
		arguments[0] = "maddox"
		for argument_index := 1; argument_index < len(arguments); argument_index++ {
			arguments[argument_index] = "x"
		}
		return arguments
	}
	arguments = maddox.Arguments{"maddox"}
	for _, command := range command_set(s) {
		arguments = append(arguments, command_text(command))
	}
	arguments = append(arguments, s.Texts...)
	if s.Commands == 0 {
		if len(s.Texts) == 0 {
			return arguments
		}
	}
	if s.Commands == 1 {
		if s.Words == 1 {
			if s.Path_Bytes == 2 {
				return arguments
			}
		}
	}
	arguments = append(arguments,
		fmt.Sprintf("-duration=%d", s.Duration),
		fmt.Sprintf("-runs=%d", s.Runs),
		fmt.Sprintf("-warmup=%d", s.Warmup))
	if s.Allow_Fail {
		arguments = append(arguments, "-allow-failures")
	}
	if s.Format == 1 {
		arguments = append(arguments, "-json")
	}
	color_mode := "never"
	if s.Color {
		color_mode = "always"
	}
	arguments = append(arguments, "-color="+color_mode)
	if s.Progress {
		arguments = append(arguments, "-progress=always")
	}
	if s.Progress_Never {
		arguments = append(arguments, "-progress=never")
	}
	return arguments
}

// Command_text joins one injected process request into Maddox's positional syntax.
func command_text(command nbio.Process_Request) (text string) {
	words := append([]string{}, command.Environment...)
	words = append(words, command.Path)
	words = append(words, command.Arguments...)
	for index, word := range words {
		if index > 0 {
			text += " "
		}
		text += word
	}
	return text
}

// Repeats one text count times.
func text_repeat(unit string, count int) (text string) {
	for index := 0; index < count; index++ {
		text += unit
	}
	return text
}

// Sample_value is run's metric value about a given center: the center itself for the first
// Center_Count runs — a tight, zero-IQR median cluster — then the center plus the deviation
// pattern, so every later spread sample falls outside the collapsed Tukey fence as an outlier.
// A pattern bound to one command leaves every other command at its center.
func sample_value(s scenario, center int64, command_index int, run int) (value int64) {
	value = center
	if s.Deltas_Command > 0 {
		if command_index != s.Deltas_Command-1 {
			return value
		}
	}
	if run >= s.Center_Count {
		if len(s.Deltas) > 0 {
			value += s.Deltas[run%len(s.Deltas)]
		}
	}
	return value
}

// Command_set is the scenario's commands, each an executable plus Words-1 argument
// words of Word_Bytes bytes, so the command and command-word length invariants see
// the empty, one-, two-, and full-length shapes.
func command_set(s scenario) (commands []nbio.Process_Request) {
	commands = make([]nbio.Process_Request, s.Commands)
	for index := range commands {
		// A distinct path per command lets the sampler tell benchmarks apart; the argument
		// words carry the command-word length shapes. A single-command run may set a short
		// executable so the progress label reaches its short shapes.
		path := fmt.Sprintf("cmd%d", index)
		if s.Commands == 1 {
			path = single_command_path(s)
		}
		command := nbio.Process_Request{Path: path}
		if s.Assignment {
			command.Environment = []string{"A=b"}
		}
		// The command line parser keeps a positional of at most COMMAND_TEXT_BYTES_MAX
		// bytes, so each argument word shrinks until the whole command text fits.
		word_bytes := s.Word_Bytes
		if s.Words > 1 {
			fit := (maddox.COMMAND_TEXT_BYTES_MAX-len(path))/(s.Words-1) - 1
			word_bytes = max(0, min(word_bytes, fit))
		}
		argument := text_repeat("a", word_bytes)
		for word_index := 1; word_index < s.Words; word_index++ {
			command.Arguments = append(command.Arguments, argument)
		}
		commands[index] = command
	}
	return commands
}

// Single_command_path is a lone command's executable at the selected width.
func single_command_path(s scenario) (path string) {
	if s.Path_Wide {
		return text_repeat("𝕫", s.Path_Bytes)
	}
	if s.Path_Bytes > 0 {
		return text_repeat("z", s.Path_Bytes)
	}
	return "cmd0"
}

// Metric_clamp bounds a decoded value to the range a real sampler reports within:
// non-negative and no larger than the representable ceiling. Every metric a sample carries
// — the counters here and the wall time in drive — passes through it, so a full-range
// decoded field reaches maddox as an in-contract metric rather than one the fixed-point
// statistics cannot represent, which only an out-of-contract sampler would report.
func metric_clamp(value int64) (clamped int64) {
	if value < 0 {
		return 0
	}
	if value > maddox.METRIC_MAX {
		return maddox.METRIC_MAX
	}
	return value
}

// Usage_from builds a run whose every metric is the given value, clamped to the valid metric
// range. A real loop reports valid metrics: non-negative and within the representable ceiling, so
// the clamp keeps the simulated measurements in contract rather than tripping the metric guard,
// which only an out-of-contract loop would.
func usage_from(value int64) (result nbio.Process_Result) {
	value = metric_clamp(value)
	result.Usage.RSS_Bytes_Max = value
	result.Usage.Cycle_Count = uint64(value)
	result.Usage.Instruction_Count = uint64(value)
	result.Usage.Cache_Reference_Count = uint64(value)
	result.Usage.Cache_Miss_Count = uint64(value)
	result.Usage.Branch_Miss_Count = uint64(value)
	result.Usage.CPU_User = time.Duration(value)
	result.Usage.CPU_System = time.Duration(value)
	return result
}

// Cursor draws structured values out of the fuzz byte string. Every read is total
// — a slice past the end reads as zero — so any input, however short or malformed,
// decodes to a complete scenario and the fuzzer never wastes a mutation.
type cursor struct {
	// Data is the fuzz byte string being consumed.
	Data []byte
	// Position is the read offset; a read past the end yields zero.
	Position int
}

func cursor_u8(c *cursor) (value uint8) {
	if c.Position < len(c.Data) {
		value = c.Data[c.Position]
	}
	c.Position++
	return value
}

func cursor_u16(c *cursor) (value uint16) {
	return uint16(cursor_u8(c))<<8 | uint16(cursor_u8(c))
}

func cursor_u32(c *cursor) (value uint32) {
	return uint32(cursor_u16(c))<<16 | uint32(cursor_u16(c))
}

func cursor_u64(c *cursor) (value uint64) {
	return uint64(cursor_u32(c))<<32 | uint64(cursor_u32(c))
}

func cursor_i64(c *cursor) (value int64) {
	return int64(cursor_u64(c))
}

// Decode_scenario reads a scenario from the fuzz bytes in the same field order build
// writes. Counts are clamped to their structural ceilings; the sample base and
// deviation pattern take the full signed range so the fuzzer can drive the
// distribution from zero-variance to accumulator-saturating.
func decode_scenario(data []byte) (s scenario) {
	c := &cursor{Data: data}
	flags := cursor_u8(c)
	s.Format = flags & 1
	s.Color = flags&2 != 0
	s.Progress = flags&4 != 0
	s.Allow_Fail = flags&8 != 0
	s.Broken_Output = flags&16 != 0
	s.Path_Wide = flags&32 != 0
	s.Arguments_Max = flags&64 != 0
	s.Wall_Pattern = flags&128 != 0
	more := cursor_u8(c)
	s.Assignment = more&1 != 0
	s.Progress_Never = more&2 != 0
	s.Spawn_Fail = more&4 != 0
	// Clamp the command and word counts to maddox's own structural ceilings, and budget the
	// total argument bytes so the rendered report stays under its width ceiling. Importing the
	// library bounds lets the driver reach the real maxima (COMMAND_SET_MAX, COMMAND_WORDS_MAX)
	// instead of an arbitrary decode limit.
	s.Commands = min(int(cursor_u16(c)), maddox.COMMAND_SET_MAX)
	s.Words = max(maddox.WORD_MIN, min(int(cursor_u16(c)), maddox.COMMAND_WORDS_MAX))
	budget := 32768 / (s.Commands*s.Words + 1)
	s.Word_Bytes = min(min(int(cursor_u16(c)), maddox.COMMAND_WORD_BYTES_MAX), budget)
	s.Path_Bytes = min(int(cursor_u16(c)), maddox.COMMAND_WORD_BYTES_MAX)
	// Runs_Max and Warmup_Count take the full int range: a caller may set any value
	// (Main caps kept runs at samples_max and treats a non-positive limit as disabled).
	s.Runs = int(cursor_i64(c))
	s.Warmup = int(cursor_i64(c))
	s.Duration = cursor_i64(c)
	s.Base = cursor_i64(c)
	deltas := min(int(cursor_u8(c)), SIM_DELTAS_MAX)
	for index := 0; index < deltas; index++ {
		s.Deltas = append(s.Deltas, cursor_i64(c))
	}
	s.Deltas_Command = min(int(cursor_u8(c)), maddox.COMMAND_SET_MAX)
	s.Center_Count = min(int(cursor_u16(c)), maddox.SAMPLES_MAX)
	s.Sleep = cursor_i64(c)
	s.Exit = int(cursor_u8(c))
	s.Fail_Every = int(cursor_u8(c))
	s.Fail_Command = int(cursor_u8(c))
	s.Divergence = cursor_i64(c)
	s.Stderr_Bytes = min(int(cursor_u32(c)), maddox.CAPTURE_BYTES_MAX)
	s.Environment_Count = min(int(cursor_u16(c)), SIM_ENVIRONMENT_MAX)
	texts := min(int(cursor_u8(c)), SIM_TEXTS_MAX)
	for index := 0; index < texts; index++ {
		size := min(int(cursor_u16(c)), maddox.COMMAND_TEXT_BYTES_MAX)
		text := make([]byte, size)
		for byte_index := range text {
			text[byte_index] = cursor_u8(c)
		}
		s.Texts = append(s.Texts, string(text))
	}
	s.Machine = decode_machine(c)
	return s
}

// Decode_machine reads the host snapshot: five text fields by length, four core
// counts as full-range ints, and six sizes as full-range uint64s — the ranges the
// Machine_Specs invariants witness at their boundaries.
func decode_machine(c *cursor) (m maddox.Machine_Specs) {
	m.CPU_Model = maddox.Processor_Model(host_text(c))
	m.CPU_Arch = maddox.Processor_Architecture(host_text(c))
	m.Operating_System_Name = maddox.Operating_System_Name(host_text(c))
	m.Operating_System_Version = maddox.Operating_System_Version(host_text(c))
	m.Kernel_Version = maddox.Kernel_Version(host_text(c))
	// A real acquire_machine_specs reports realistic, render-safe values; clamp to the
	// domain ceilings so the simulated host honors that contract rather than tripping the
	// table's render guards, which only an out-of-contract probe would.
	m.Physical_Cores = maddox.Physical_Core_Count(
		min(int(cursor_u16(c)), maddox.CORES_COUNT_MAX))
	m.Logical_Cores = maddox.Logical_Core_Count(
		min(int(cursor_u16(c)), maddox.CORES_COUNT_MAX))
	m.Performance_Cores = maddox.Performance_Core_Count(
		min(int(cursor_u16(c)), maddox.CORES_COUNT_MAX))
	m.Efficiency_Cores = maddox.Efficiency_Core_Count(
		min(int(cursor_u16(c)), maddox.CORES_COUNT_MAX))
	m.CPU_Frequency_Hz_Max = maddox.Processor_Frequency(
		min(cursor_u64(c), maddox.HERTZ_MAX))
	m.Cache_L1_Bytes = maddox.Level_1_Cache_Size(
		min(cursor_u64(c), maddox.BYTE_SIZE_MAX))
	m.Cache_L2_Bytes = maddox.Level_2_Cache_Size(
		min(cursor_u64(c), maddox.BYTE_SIZE_MAX))
	m.Cache_L3_Bytes = maddox.Level_3_Cache_Size(
		min(cursor_u64(c), maddox.BYTE_SIZE_MAX))
	m.RAM_Total_Bytes = maddox.Memory_Size(min(cursor_u64(c), maddox.BYTE_SIZE_MAX))
	m.Storage_Total_Bytes = maddox.Storage_Size(min(cursor_u64(c), maddox.BYTE_SIZE_MAX))
	return m
}

// Host_text is a run of 'x' of the decoded length, clamped to the host-field
// ceiling — the invariants witness the field's length, not its bytes.
func host_text(c *cursor) (text maddox.Host_Text) {
	count := min(int(cursor_u16(c)), maddox.HOST_TEXT_BYTES_MAX)
	return maddox.Host_Text(text_repeat("x", count))
}

// Writer is the cursor's inverse: it lays down the same big-endian fields a seed
// scenario needs so build and decode_scenario round-trip. Fields clamped on decode
// round-trip for any value at or below their ceiling.
type writer struct {
	// Data is the accumulated byte string.
	Data []byte
}

func writer_u8(w *writer, value uint8) { w.Data = append(w.Data, value) }

func writer_u16(w *writer, value uint16) {
	writer_u8(w, uint8(value>>8))
	writer_u8(w, uint8(value))
}

func writer_u32(w *writer, value uint32) {
	writer_u16(w, uint16(value>>16))
	writer_u16(w, uint16(value))
}

func writer_u64(w *writer, value uint64) {
	writer_u32(w, uint32(value>>32))
	writer_u32(w, uint32(value))
}

func writer_i64(w *writer, value int64) { writer_u64(w, uint64(value)) }

// Build serializes a scenario to the byte form decode_scenario reads, so a seed is
// authored as a struct and replayed as bytes.
func build(s scenario) (data []byte) {
	w := &writer{}
	var flags uint8
	flags |= s.Format & 1
	if s.Color {
		flags |= 2
	}
	if s.Progress {
		flags |= 4
	}
	if s.Allow_Fail {
		flags |= 8
	}
	if s.Broken_Output {
		flags |= 16
	}
	if s.Path_Wide {
		flags |= 32
	}
	if s.Arguments_Max {
		flags |= 64
	}
	if s.Wall_Pattern {
		flags |= 128
	}
	writer_u8(w, flags)
	var more uint8
	if s.Assignment {
		more |= 1
	}
	if s.Progress_Never {
		more |= 2
	}
	if s.Spawn_Fail {
		more |= 4
	}
	writer_u8(w, more)
	writer_u16(w, uint16(s.Commands))
	writer_u16(w, uint16(s.Words))
	writer_u16(w, uint16(s.Word_Bytes))
	writer_u16(w, uint16(s.Path_Bytes))
	writer_i64(w, int64(s.Runs))
	writer_i64(w, int64(s.Warmup))
	writer_i64(w, s.Duration)
	writer_i64(w, s.Base)
	writer_u8(w, uint8(len(s.Deltas)))
	for _, delta := range s.Deltas {
		writer_i64(w, delta)
	}
	writer_u8(w, uint8(s.Deltas_Command))
	writer_u16(w, uint16(s.Center_Count))
	writer_i64(w, s.Sleep)
	writer_u8(w, uint8(s.Exit))
	writer_u8(w, uint8(s.Fail_Every))
	writer_u8(w, uint8(s.Fail_Command))
	writer_i64(w, s.Divergence)
	writer_u32(w, uint32(s.Stderr_Bytes))
	writer_u16(w, uint16(s.Environment_Count))
	writer_u8(w, uint8(len(s.Texts)))
	for _, text := range s.Texts {
		writer_u16(w, uint16(len(text)))
		w.Data = append(w.Data, text...)
	}
	build_machine(w, s.Machine)
	return w.Data
}

func build_machine(w *writer, m maddox.Machine_Specs) {
	writer_u16(w, uint16(len(m.CPU_Model)))
	writer_u16(w, uint16(len(m.CPU_Arch)))
	writer_u16(w, uint16(len(m.Operating_System_Name)))
	writer_u16(w, uint16(len(m.Operating_System_Version)))
	writer_u16(w, uint16(len(m.Kernel_Version)))
	writer_u16(w, uint16(m.Physical_Cores))
	writer_u16(w, uint16(m.Logical_Cores))
	writer_u16(w, uint16(m.Performance_Cores))
	writer_u16(w, uint16(m.Efficiency_Cores))
	writer_u64(w, uint64(m.CPU_Frequency_Hz_Max))
	writer_u64(w, uint64(m.Cache_L1_Bytes))
	writer_u64(w, uint64(m.Cache_L2_Bytes))
	writer_u64(w, uint64(m.Cache_L3_Bytes))
	writer_u64(w, uint64(m.RAM_Total_Bytes))
	writer_u64(w, uint64(m.Storage_Total_Bytes))
}

// Base_scenario is a sane two-command benchmark: five kept runs of a small
// {0,±1,±2} distribution, table output, a modest realistic machine. Seeds start
// here and vary the one axis they exercise.
func base_scenario() (s scenario) {
	return scenario{
		Commands:   2,
		Words:      2,
		Word_Bytes: 1,
		Runs:       5,
		Base:       1000,
		Deltas:     []int64{0, 1, -1, 2, -2},
		Sleep:      1,
		Format:     0,
		Machine: maddox.Machine_Specs{
			CPU_Model:             "x",
			CPU_Arch:              "x",
			Operating_System_Name: "x",
			Physical_Cores:        8,
			Logical_Cores:         8,
			RAM_Total_Bytes:       16,
			Storage_Total_Bytes:   512,
		},
	}
}

// With applies overrides to a fresh base scenario, keeping seeds terse.
func with(mutate func(s *scenario)) (data []byte) {
	s := base_scenario()
	mutate(&s)
	return build(s)
}

// Json marks a scenario for JSON output, the shape that carries extreme injected
// data without a rendering guard rejecting it.
func json(s *scenario) { s.Format = 1 }

// Seed_corpus is the honest battery: each seed is a real end-to-end Main run that
// reaches a distinct boundary of the pipeline, assembled from focused groups.
func seed_corpus() (seeds [][]byte) {
	seeds = append(seeds, build(base_scenario()))
	seeds = append(seeds, seeds_output()...)
	seeds = append(seeds, seeds_shape()...)
	seeds = append(seeds, seeds_variance()...)
	seeds = append(seeds, seeds_failure()...)
	seeds = append(seeds, seeds_configuration()...)
	seeds = append(seeds, seeds_machine()...)
	seeds = append(seeds, seeds_machine_values()...)
	seeds = append(seeds, seeds_render()...)
	seeds = append(seeds, seeds_progress()...)
	seeds = append(seeds, seeds_overflow()...)
	seeds = append(seeds, seeds_saturation()...)
	seeds = append(seeds, seeds_width()...)
	seeds = append(seeds, seeds_extremes()...)
	seeds = append(seeds, seeds_comparison()...)
	seeds = append(seeds, seeds_document()...)
	seeds = append(seeds, seeds_clock()...)
	seeds = append(seeds, seeds_command_line()...)
	return seeds
}

// Seeds_width drives the rendered cells to their width ceilings: ceiling-scale, sharply
// divergent metrics with color and a full machine render the metric name, mean, deviation, and
// delta columns each at their widest, so the full row, bare row, delta text, delta body, and
// padded cell reach their length maxima that a modest run never approaches.
func seeds_width() (seeds [][]byte) {
	return [][]byte{
		with(func(s *scenario) {
			s.Commands = 2
			s.Base = maddox.METRIC_MAX / 2
			s.Divergence = maddox.METRIC_MAX / 4
			s.Deltas = []int64{0, 1 << 30, -(1 << 30)}
			s.Color = true
			full_machine(s)
		}),
		with(func(s *scenario) {
			s.Commands = 2
			s.Base = maddox.METRIC_MAX / 2
			s.Divergence = -(maddox.METRIC_MAX / 4)
			s.Deltas = []int64{0, 1 << 30, -(1 << 30)}
			full_machine(s)
		}),
		// A near-zero reference of maximal coefficient of variation — a lone ceiling
		// spike among zeros — against a ceiling-scale candidate: the change overruns the
		// display cap and the widest confidence half-interval a bounded t-test reaches
		// widen the delta body and its colored text to their maxima.
		with(func(s *scenario) {
			s.Commands = 2
			s.Runs = maddox.SAMPLES_MAX
			s.Sleep = 0
			s.Center_Count = maddox.SAMPLES_MAX - 1
			s.Base = 0
			s.Deltas = []int64{maddox.METRIC_MAX}
			s.Divergence = maddox.METRIC_MAX
			s.Color = true
		}),
	}
}

// Seeds_saturation drives the pipeline to the structural ceilings a narrow corpus never reaches:
// a full two-command run at the sample cap — an instant clock and no time budget keep exactly
// SAMPLES_MAX samples per command — so the sample-count, distribution, series, and
// degrees-of-freedom invariants witness their maxima (two n=SAMPLES_MAX distributions give a
// t-test degrees-of-freedom of 2*(SAMPLES_MAX-1)); a full command set, so the benchmark
// collection reaches its size ceiling and the per-benchmark index its last position; and a
// full command line, so the command-line word count reaches COMMAND_WORDS_MAX in both the
// plain and the progress-rendered paths.
func seeds_saturation() (seeds [][]byte) {
	return [][]byte{
		with(func(s *scenario) { s.Commands = 8 }),
		with(func(s *scenario) {
			s.Words = maddox.COMMAND_WORDS_MAX
			s.Progress = true
			s.Warmup = 1
		}),
		// A full two-command run whose middle half is pinned to a ceiling-scale center and
		// whose other half splits just above and below it: the IQR collapses to zero, so
		// every spread sample is an outlier and the outlier count reaches its ceiling
		// (SAMPLES_MAX/2 - 1). Ceiling values, a sharp divergence, and color make one row
		// carry the widest name, cells, outlier tally, and delta at once, so the assembled
		// rows and their columns reach their true maxima.
		with(func(s *scenario) {
			s.Commands = 2
			s.Runs = maddox.SAMPLES_MAX
			s.Sleep = 0
			s.Center_Count = maddox.SAMPLES_MAX/2 + 1
			s.Base = maddox.METRIC_MAX / 2
			s.Deltas = []int64{100, -100}
			s.Divergence = maddox.METRIC_MAX / 4
			s.Color = true
			s.Wall_Pattern = true
		}),
		// Both commands use the same wall distribution, so each comparison side reaches
		// the maximum wall-time outlier count.
		with(func(s *scenario) {
			json(s)
			s.Commands = 2
			s.Runs = maddox.SAMPLES_MAX
			s.Sleep = 1000
			s.Center_Count = maddox.SAMPLES_MAX/2 + 1
			s.Deltas = []int64{100, -100}
			s.Divergence = 0
			s.Wall_Pattern = true
		}),
		// A full single-command bimodal distribution — half at zero, half at the metric
		// ceiling — maximizes the sum of squared deviations, so the accumulator's high word
		// reaches its largest value; the progress counter is on, so the progress render's
		// running count reaches the sample ceiling in the same run.
		with(func(s *scenario) {
			s.Commands = 1
			s.Runs = maddox.SAMPLES_MAX
			s.Sleep = 0
			s.Base = maddox.METRIC_MAX / 2
			s.Deltas = []int64{
				maddox.METRIC_MAX - maddox.METRIC_MAX/2,
				-(maddox.METRIC_MAX / 2),
			}
			s.Progress = true
			s.Warmup = 1
		}),
		low_word_congruence_seed(),
	}
}

// Low_word_congruence_seed is a crafted single-command distribution — seven symmetric pairs
// about a mid-range center plus a lone +1 spike — whose sum of squared deviations
// (2*sum(d^2) + 1) is congruent to 2^64-1 modulo 2^64, so the 128-bit accumulator's low word
// reaches its full-width ceiling. The deltas were found by a 2-adic square-root search for that
// congruence; no benchmark distribution reaches the modular ceiling by chance.
func low_word_congruence_seed() (seed []byte) {
	return with(func(s *scenario) {
		s.Commands = 1
		s.Runs = 16
		s.Sleep = 0
		s.Base = maddox.METRIC_MAX / 2
		s.Deltas = []int64{
			826094412318, -826094412318, 3456986167502, -3456986167502,
			3426655266655, -3426655266655, 779384028002, -779384028002,
			990395852196, -990395852196, 1437896511053, -1437896511053,
			512601834219, -512601834219, 1, 0,
		}
	})
}

// Seeds_overflow pins the fuzz-discovered inputs where a large but in-contract value overflowed
// maddox's fixed-point math or a fixed column width — each once tripped a boundary guard deep in
// the statistics or the renderer, and each is now handled. The metric magnitudes here sit near
// the representable ceiling the narrow seed corpus never reaches.
func seeds_overflow() (seeds [][]byte) {
	return [][]byte{
		// A distribution split between zero and the representable ceiling over
		// enough runs to clear the strays ceiling: the interquartile range spans
		// nearly the whole metric range, so the Tukey outlier fence (1.5*IQR lifted
		// by the fixed-point scale) overflowed int64 and miscounted every run as an
		// outlier past that ceiling. Base sits at the midpoint; the two-grain
		// deviation drives each run to one extreme or the other.
		with(func(s *scenario) {
			s.Commands = 1
			// Enough runs to clear the strays ceiling (half the sample cap), sized off
			// the exported cap so it tracks it rather than a magic number.
			s.Runs = maddox.SAMPLES_MAX/2 + 1
			s.Base = 1 << 42
			s.Deltas = []int64{1 << 42, -(1 << 42)}
			s.Sleep = 0
		}),
		// A machine memory size high on the byte-size range, where lifting the
		// raw count whole into the 2^20 fixed-point scale overflowed and rendered
		// a garbage, over-width cell. Eight tebibytes renders "8TiB" once the
		// scaling divides before it lifts.
		with(func(s *scenario) {
			full_machine(s)
			s.Machine.RAM_Total_Bytes = 1 << 43
		}),
		// A ceiling-valued second command against a tiny reference: the percentage
		// change runs past a trillion percent, thirteen digits, whose width overran
		// the fixed delta column before the display clamp pinned it to the column's
		// widest value.
		with(func(s *scenario) {
			s.Base = 500
			s.Divergence = maddox.METRIC_MAX
			s.Deltas = []int64{0}
			s.Runs = 3
		}),
		// Three runs each costing a three-kilosecond wall: the total sampling time sums
		// past the fixed-point scale's ceiling, where lifting it whole to render the
		// benchmark header overflowed and produced a garbage, over-width cell. Nine
		// kiloseconds renders "9ks" once the scaling divides before it lifts.
		with(func(s *scenario) {
			s.Sleep = 3_000_000_000_000
			s.Runs = 3
		}),
	}
}

// Seeds_progress exercises the progress counter with varied command shapes.
func seeds_progress() (seeds [][]byte) {
	progress := func(mutate func(s *scenario)) (data []byte) {
		return with(func(s *scenario) {
			s.Progress = true
			s.Warmup = 2
			mutate(s)
		})
	}
	// The progress total mirrors Runs_Max, injected config a caller may set to any int; a
	// short duration budget stops sampling after a few renders so the extreme is cheap.
	total := func(runs int) (data []byte) {
		return with(func(s *scenario) {
			s.Progress = true
			s.Runs = runs
			s.Duration = 20
			s.Sleep = 8_000_000_000
		})
	}
	return [][]byte{
		progress(func(s *scenario) { s.Words = 1 }),
		progress(func(s *scenario) { s.Words = 3 }),
		progress(func(s *scenario) { s.Word_Bytes = 0 }),
		progress(func(s *scenario) { s.Word_Bytes = 2 }),
		progress(func(s *scenario) { s.Word_Bytes = maddox.COMMAND_WORD_BYTES_MAX }),
		total(0), total(1), total(2), total(-1), total(int(bits.INTEGER_64_MINIMUM)), total(
			int(bits.INTEGER_64_MAXIMUM)),
		// Progress labels at their one- and two-byte shapes: a single short-path command.
		with(func(s *scenario) {
			s.Progress = true
			s.Commands = 1
			s.Words = 1
			s.Path_Bytes = 1
		}),
		with(func(s *scenario) {
			s.Progress = true
			s.Commands = 1
			s.Words = 1
			s.Path_Bytes = 2
		}),
		// A single command with a rune-cap-long four-byte-per-rune executable and no
		// arguments: the progress label is the untruncated path at its byte ceiling.
		with(func(s *scenario) {
			s.Progress = true
			s.Commands = 1
			s.Words = 1
			s.Path_Wide = true
			s.Path_Bytes = maddox.PROGRESS_LABEL_RUNES_MAX
		}),
		// A full operating-system argument list reaches the command-line input ceiling.
		with(func(s *scenario) {
			s.Arguments_Max = true
		}),
	}
}

// Seeds_machine_values drives the host's numeric fields to each boundary — zero, one,
// two, and the render-safe ceiling — through the table, so the core layout, the byte
// sizes, and the frequency all render and their invariants witness the shapes a real
// machine reports. Go's fuzzer guides on edges, not values, so these are seeded, not
// discovered.
func seeds_machine_values() (seeds [][]byte) {
	cores := func(value maddox.Cores) (mutate func(s *scenario)) {
		return func(s *scenario) { full_machine(s); set_all_cores(&s.Machine, value) }
	}
	size := func(value maddox.Byte_Size) (mutate func(s *scenario)) {
		return func(s *scenario) { full_machine(s); set_all_sizes(&s.Machine, value) }
	}
	frequency := func(value maddox.Hertz) (mutate func(s *scenario)) {
		return func(s *scenario) {
			full_machine(s)
			s.Machine.CPU_Frequency_Hz_Max = maddox.Processor_Frequency(value)
		}
	}
	json_value := func(mutate func(s *scenario)) (wrapped func(s *scenario)) {
		return func(s *scenario) {
			json(s)
			mutate(s)
		}
	}
	return [][]byte{
		with(cores(0)), with(cores(1)), with(cores(2)), with(cores(maddox.CORES_COUNT_MAX)),
		with(size(0)), with(size(1)), with(size(2)), with(size(maddox.BYTE_SIZE_MAX)),
		// 1023 GiB renders "1023GiB" — a full seven-byte cell, within the representable
		// range the fixed-point render can lift without overflow.
		with(size(1023 * (1 << 30))),
		with(frequency(0)), with(frequency(1)), with(frequency(2)),
		with(frequency(maddox.HERTZ_MAX)),
		with(json_value(cores(0))), with(json_value(cores(1))),
		with(json_value(cores(2))), with(json_value(cores(maddox.CORES_COUNT_MAX))),
		with(json_value(size(0))), with(json_value(size(1))),
		with(json_value(size(2))), with(json_value(size(maddox.BYTE_SIZE_MAX))),
		with(json_value(frequency(0))), with(json_value(frequency(1))),
		with(json_value(frequency(2))), with(json_value(frequency(maddox.HERTZ_MAX))),
	}
}

// Set_all_cores sets every core-count field to value.
func set_all_cores(m *maddox.Machine_Specs, value maddox.Cores) {
	m.Physical_Cores = maddox.Physical_Core_Count(value)
	m.Logical_Cores = maddox.Logical_Core_Count(value)
	m.Performance_Cores = maddox.Performance_Core_Count(value)
	m.Efficiency_Cores = maddox.Efficiency_Core_Count(value)
}

// Set_all_sizes sets every byte-size field to value.
func set_all_sizes(m *maddox.Machine_Specs, value maddox.Byte_Size) {
	m.Cache_L1_Bytes = maddox.Level_1_Cache_Size(value)
	m.Cache_L2_Bytes = maddox.Level_2_Cache_Size(value)
	m.Cache_L3_Bytes = maddox.Level_3_Cache_Size(value)
	m.RAM_Total_Bytes = maddox.Memory_Size(value)
	m.Storage_Total_Bytes = maddox.Storage_Size(value)
}

// Full_machine populates every host field with a realistic hybrid-CPU snapshot so the
// table header renders the core layout, the cache and memory sizes, and the frequency.
func full_machine(s *scenario) {
	s.Machine.Performance_Cores = 4
	s.Machine.Efficiency_Cores = 4
	s.Machine.CPU_Frequency_Hz_Max = 3_600_000_000
	s.Machine.Cache_L1_Bytes = 64 * 1024
	s.Machine.Cache_L2_Bytes = 4 * 1024 * 1024
	s.Machine.Cache_L3_Bytes = 32 * 1024 * 1024
	s.Machine.RAM_Total_Bytes = 16 * 1024 * 1024 * 1024
	s.Machine.Storage_Total_Bytes = 512 * 1024 * 1024 * 1024
}

// Seeds_render drives the table machine header — the core layout, byte sizes, and
// frequency — plus the colored and progress variants that reach the render paths a plain
// JSON run never touches.
func seeds_render() (seeds [][]byte) {
	return [][]byte{
		// One one-byte command with zero measurements is the shortest valid table.
		with(func(s *scenario) {
			s.Commands = 1
			s.Words = 1
			s.Path_Bytes = 1
			s.Runs = 3
			s.Base = 0
			s.Deltas = []int64{0}
			s.Sleep = 0
			s.Machine = maddox.Machine_Specs{}
		}),
		with(full_machine),
		with(func(s *scenario) { full_machine(s); s.Color = true }),
		with(func(s *scenario) { full_machine(s); s.Progress = true; s.Warmup = 1 }),
		// A two-digit core count with no hybrid split, so the cores line is two bytes.
		with(func(s *scenario) {
			full_machine(s)
			s.Machine.Performance_Cores = 0
			s.Machine.Efficiency_Cores = 0
			s.Machine.Physical_Cores = 88
			s.Machine.Logical_Cores = 88
		}),
		// An empty document with no machine, so the rendered report is empty.
		with(func(s *scenario) {
			s.Commands = 0
			s.Machine = maddox.Machine_Specs{}
		}),
		// Three one-second runs, so the header's total sampling span renders "3s" — the
		// two-byte, single-digit-seconds shape between the "0" floor and the wider forms.
		with(func(s *scenario) {
			s.Sleep = 1_000_000_000
			s.Runs = 3
		}),
	}
}

// Seeds_output varies the report rendering and its flags.
func seeds_output() (seeds [][]byte) {
	return [][]byte{
		with(json),
		with(func(s *scenario) { s.Color = true }),
		with(func(s *scenario) { s.Progress = true; s.Warmup = 2 }),
		with(func(s *scenario) { s.Broken_Output = true }),
		with(func(s *scenario) { json(s); s.Broken_Output = true }),
	}
}

// Seeds_shape varies the command-set, command-line, and sample-count structure.
func seeds_shape() (seeds [][]byte) {
	return [][]byte{
		with(func(s *scenario) { s.Commands = 0 }),
		with(func(s *scenario) { s.Commands = 1 }),
		with(func(s *scenario) { s.Commands = 4 }),
		with(func(s *scenario) { json(s); s.Commands = 0 }),
		with(func(s *scenario) { json(s); s.Commands = 1 }),
		with(func(s *scenario) { json(s); s.Commands = maddox.COMMAND_SET_MAX }),
		with(func(s *scenario) { s.Words = 1 }),
		with(func(s *scenario) { s.Words = 3 }),
		with(func(s *scenario) { s.Word_Bytes = 0 }),
		with(func(s *scenario) { s.Word_Bytes = 2 }),
		with(func(s *scenario) { s.Word_Bytes = maddox.COMMAND_WORD_BYTES_MAX }),
		with(func(s *scenario) { s.Runs = 3 }),
	}
}

// Seeds_variance drives the distribution: zero deviation, unit and two-grain gaps,
// the representable metric ceiling, wall spans, and large deviations reaching the
// wide accumulator's high word.
func seeds_variance() (seeds [][]byte) {
	return [][]byte{
		with(func(s *scenario) { s.Deltas = []int64{0} }),
		with(func(s *scenario) { s.Base = 0; s.Deltas = []int64{0} }),
		with(func(s *scenario) { s.Base = 1; s.Deltas = []int64{0} }),
		with(func(s *scenario) { s.Base = 2; s.Deltas = []int64{0} }),
		with(func(s *scenario) { s.Base = maddox.METRIC_MAX; s.Deltas = []int64{0} }),
		with(func(s *scenario) { s.Sleep = 0 }),
		with(func(s *scenario) { s.Sleep = 2 }),
		with(func(s *scenario) { s.Sleep = maddox.METRIC_MAX; s.Runs = 3 }),
		// Three symmetric wall samples have a standard deviation of exactly one. The
		// comparison and rendering paths therefore witness the middle deviation region.
		with(func(s *scenario) {
			s.Commands = 2
			s.Runs = 3
			s.Sleep = 1000
			s.Deltas = []int64{-1, 0, 1}
			s.Wall_Pattern = true
		}),
		// Large symmetric deviations sized so the sum of squares lands its high 64-bit word
		// on one and two: four runs of a 2^31 gap give a sum of exactly 2^64, eight give
		// 2^65. JSON output carries these without a render guard rejecting the magnitudes.
		with(func(s *scenario) {
			json(s)
			s.Base = 1 << 40
			s.Runs = 4
			s.Deltas = []int64{1 << 31, -(1 << 31)}
		}),
		with(func(s *scenario) {
			json(s)
			s.Base = 1 << 40
			s.Runs = 8
			s.Deltas = []int64{1 << 31, -(1 << 31)}
		}),
		// A flat distribution (zero IQR) with one and two spikes: each spike falls beyond
		// Tukey's fences, so the outlier count is the number of spikes.
		outlier_seed(10),
		outlier_seed(20),
		// Three spikes, so the outlier count is an ordinary value past its small shapes.
		outlier_seed(30),
		// A significant slowdown: the second command measures far above the reference with
		// tight variance, so the comparison clears the confidence band.
		with(func(s *scenario) {
			s.Base = 1000
			s.Divergence = 1000
			s.Deltas = []int64{0, 1, -1}
			s.Sleep = 1000
			s.Wall_Pattern = true
		}),
		// A wide byte-metric distribution (values 0 and ~2 TiB) whose mean and standard
		// deviation each render a full seven-byte cell.
		with(func(s *scenario) {
			s.Base = 1023 * (1 << 30)
			s.Runs = 4
			s.Deltas = []int64{1023 * (1 << 30), -(1023 * (1 << 30))}
		}),
		// A single unit deviation among three runs: the sum of squares is exactly one, so
		// the accumulator's low word reaches one at the root-of-quotient input.
		with(func(s *scenario) { s.Base = 1000; s.Runs = 3; s.Deltas = []int64{0, 0, 1} }),
		// A faster candidate: the second command measures below the reference.
		with(func(s *scenario) {
			s.Base = 1000
			s.Divergence = -100
			s.Deltas = []int64{0, 1, -1}
			s.Sleep = 1000
			s.Wall_Pattern = true
		}),
	}
}

// Outlier_seed repeats one spike per ten runs for every measured metric.
func outlier_seed(runs int) (seed []byte) {
	return with(func(s *scenario) {
		s.Base = 1000
		s.Runs = runs
		s.Deltas = spike_deltas()
		s.Wall_Pattern = true
	})
}

// Spike_deltas is nine zero deviations and one large one, so one in ten samples is a
// spike far beyond the flat bulk's fences.
func spike_deltas() (deltas []int64) {
	return []int64{0, 0, 0, 0, 0, 0, 0, 0, 0, 1 << 20}
}

// Seeds_failure exercises the abort path and the stderr capture lengths.
func seeds_failure() (seeds [][]byte) {
	fail := func(bytes int) (mutate func(s *scenario)) {
		return func(s *scenario) { s.Exit = 1; s.Fail_Every = 1; s.Stderr_Bytes = bytes }
	}
	return [][]byte{
		with(fail(4)),
		with(func(s *scenario) { s.Exit = 1; s.Fail_Every = 1; s.Allow_Fail = true }),
		with(fail(0)),
		with(fail(1)),
		with(fail(2)),
		with(fail(maddox.CAPTURE_BYTES_MAX)),
		with(func(s *scenario) { s.Exit = 2; s.Fail_Every = 1; s.Stderr_Bytes = 1 }),
		with(func(s *scenario) { s.Exit = 255; s.Fail_Every = 1; s.Stderr_Bytes = 1 }),
		// A mid-range exit code — the command status is an ordinary value, not a boundary.
		with(func(s *scenario) { s.Exit = 127; s.Fail_Every = 1; s.Stderr_Bytes = 1 }),
		// A failure at the fourth benchmark — the index is past its small shapes.
		with(func(s *scenario) {
			s.Commands = 5
			s.Fail_Command = 4
			s.Exit = 1
			s.Stderr_Bytes = 1
		}),
		// Failures at later benchmark positions, so the diagnostic sees indices 1 and 2.
		with(func(s *scenario) {
			s.Commands = 3
			s.Fail_Command = 2
			s.Exit = 1
			s.Stderr_Bytes = 1
		}),
		with(func(s *scenario) {
			s.Commands = 4
			s.Fail_Command = 3
			s.Exit = 1
			s.Stderr_Bytes = 1
		}),
		// A failure at the eighth benchmark of a full command set: the first seven succeed,
		// so the write-failure diagnostic sees the last benchmark position.
		with(func(s *scenario) {
			s.Commands = 8
			s.Fail_Command = 8
			s.Exit = 1
			s.Stderr_Bytes = 1
		}),
	}
}

// Seeds_configuration feeds the injected config ints across the range Main tolerates.
func seeds_configuration() (seeds [][]byte) {
	return [][]byte{
		with(func(s *scenario) { s.Runs = 0; s.Duration = 1; s.Sleep = 1_000_000_000 }),
		with(func(s *scenario) { s.Runs = 1 }),
		with(func(s *scenario) { s.Runs = 2 }),
		// A disabled or absurd run cap witnesses the Runs_Max boundary from the input
		// alone; one-second walls spend a one-second budget at the quorum instead of
		// grinding to the sample ceiling, so these seeds cost three runs, not ten thousand.
		with(func(s *scenario) { s.Runs = -1; s.Duration = 1; s.Sleep = 1_000_000_000 }),
		with(func(s *scenario) {
			s.Runs = int(bits.INTEGER_64_MINIMUM)
			s.Duration = 1
			s.Sleep = 1_000_000_000
		}),
		with(func(s *scenario) {
			s.Runs = int(bits.INTEGER_64_MAXIMUM)
			s.Duration = 1
			s.Sleep = 1_000_000_000
		}),
		with(func(s *scenario) { s.Warmup = 1 }),
		with(func(s *scenario) { s.Warmup = -1 }),
		with(func(s *scenario) { s.Warmup = int(bits.INTEGER_64_MINIMUM) }),
		with(func(s *scenario) { s.Commands = 1; s.Warmup = int(bits.INTEGER_64_MAXIMUM) }),
	}
}

// Seeds_machine drives the host-text field lengths to their boundaries; the numeric
// host fields are explored by the fuzzer within their render-safe realistic ranges.
func seeds_machine() (seeds [][]byte) {
	host := func(count int, format uint8) (mutate func(s *scenario)) {
		return func(s *scenario) {
			s.Format = format
			set_hosts(&s.Machine, count)
		}
	}
	return [][]byte{
		with(host(0, 0)), with(host(1, 0)), with(host(2, 0)),
		with(host(maddox.HOST_TEXT_BYTES_MAX, 0)),
		with(host(0, 1)), with(host(1, 1)), with(host(2, 1)),
		with(host(3, 1)), with(host(maddox.HOST_TEXT_BYTES_MAX, 1)),
	}
}

// Set_hosts sets every host-text field to a run of 'x' of the given byte count.
func set_hosts(m *maddox.Machine_Specs, count int) {
	text := maddox.Host_Text(text_repeat("x", count))
	m.CPU_Model = maddox.Processor_Model(text)
	m.CPU_Arch = maddox.Processor_Architecture(text)
	m.Operating_System_Name = maddox.Operating_System_Name(text)
	m.Operating_System_Version = maddox.Operating_System_Version(text)
	m.Kernel_Version = maddox.Kernel_Version(text)
}

// Both_formats is the seed as a table and as a JSON document, so the table and the document each
// render the values it reaches.
func both_formats(mutate func(s *scenario)) (seeds [][]byte) {
	return [][]byte{with(mutate), with(func(s *scenario) { mutate(s); json(s) })}
}

// Seeds_extremes drives each distribution statistic and each comparison to the exact bounds its
// arithmetic reaches, in both the reference and the candidate and in both formats.
func seeds_extremes() (seeds [][]byte) {
	metric := int64(maddox.METRIC_MAX)
	groups := [][][]byte{
		// Zero and two ceilings: the widest deviation three runs have, with the extreme
		// minimum, maximum, and quartiles.
		both_formats(func(s *scenario) {
			s.Runs = 3
			s.Base = 0
			s.Sleep = 0
			s.Deltas = []int64{0, metric, metric}
			s.Wall_Pattern = true
		}),
		// Every run at the ceiling, then every run at zero.
		both_formats(func(s *scenario) {
			s.Runs = 3
			s.Base = metric
			s.Sleep = metric
			s.Deltas = []int64{0}
		}),
		both_formats(func(s *scenario) {
			s.Runs = 3
			s.Base = 0
			s.Sleep = 0
			s.Deltas = []int64{0}
		}),
		// A candidate of zeros against a reference of ones: the comparison divides by the
		// reference mean while the candidate's statistics sit at zero.
		both_formats(func(s *scenario) {
			s.Runs = 3
			s.Base = 1
			s.Divergence = -1
			s.Sleep = 1
			s.Deltas = []int64{0}
			s.Wall_Pattern = true
		}),
	}
	for _, group := range groups {
		seeds = append(seeds, group...)
	}
	return seeds
}

// Seeds_comparison drives each comparison to the exact bounds its fixed-point arithmetic
// reaches, the outlier and wall shapes of the document, and the widest colored delta.
func seeds_comparison() (seeds [][]byte) {
	groups := [][][]byte{
		// A reference of ones against a candidate whose mean a hundredfold wraps to four
		// short of the signed word's top: twenty-four runs at a found center and one a unit
		// above.
		both_formats(func(s *scenario) {
			s.Runs = 25
			s.Base = 1
			s.Divergence = 1143492092887
			s.Sleep = 1
			s.Deltas = make([]int64, 25)
			s.Deltas[24] = 1
			s.Deltas_Command = 2
			s.Wall_Pattern = true
		}),
		// A candidate 2^41 units above a reference of ones: a hundred times the ratio wraps
		// to the signed word's bottom.
		both_formats(func(s *scenario) {
			s.Runs = 3
			s.Base = 1
			s.Divergence = 1 << 41
			s.Sleep = 1
			s.Deltas = []int64{0}
			s.Wall_Pattern = true
		}),
		// A flat reference at 2^20 against a candidate of two zeros and a found spike: the
		// pooled relative variance wraps to just under a quarter of the signed word times
		// the four degrees, the widest confidence half-interval.
		both_formats(func(s *scenario) {
			s.Runs = 3
			s.Base = 1 << 20
			s.Divergence = -(1 << 20)
			s.Sleep = 1 << 20
			s.Deltas = []int64{0, 0, 3808820005640}
			s.Deltas_Command = 2
			s.Wall_Pattern = true
		}),
		// A significant slowdown whose change and interval both overrun the display cap: a
		// reference of ones against a candidate a million units up with a spread a fifth of
		// that, colored, so the delta text reaches its widest.
		{with(func(s *scenario) {
			s.Runs = 3
			s.Base = 1
			s.Divergence = 999_999
			s.Sleep = 1
			s.Deltas = []int64{0, 0, 200_000}
			s.Deltas_Command = 2
			s.Color = true
			s.Wall_Pattern = true
		})},
	}
	for _, group := range groups {
		seeds = append(seeds, group...)
	}
	return seeds
}

// Seeds_document drives the document's and the row's small shapes: outlier counts of one and
// two, the smallest mean that is not zero, walls summing to one and two units, a two-byte
// failing executable, and command lines of one word and of the most words.
func seeds_document() (seeds [][]byte) {
	groups := [][][]byte{
		// Outlier counts of one and two in the document.
		{with(func(s *scenario) { mutate_outliers(s, 10); json(s) })},
		{with(func(s *scenario) { mutate_outliers(s, 20); json(s) })},
		// A reference whose one unit spreads over the sample cap, the smallest mean that
		// is not zero, rendered as a row. The candidate's walls spend its one-second
		// budget at the quorum, and its mean stays small enough that its ratio to the
		// reference fits a word.
		{with(func(s *scenario) {
			s.Runs = maddox.SAMPLES_MAX
			s.Duration = 1
			s.Base = 0
			s.Sleep = 0
			s.Divergence = 400_000_000
			s.Center_Count = maddox.SAMPLES_MAX - 1
			s.Deltas = []int64{1}
			s.Deltas_Command = 1
			s.Wall_Pattern = true
		})},
		// Kept walls summing to one and to two units.
		both_formats(func(s *scenario) {
			s.Runs = 3
			s.Sleep = 0
			s.Deltas = []int64{0, 0, 1}
			s.Deltas_Command = 1
			s.Wall_Pattern = true
		}),
		both_formats(func(s *scenario) {
			s.Runs = 3
			s.Sleep = 0
			s.Deltas = []int64{0, 1, 1}
			s.Deltas_Command = 1
			s.Wall_Pattern = true
		}),
		// A failing command whose executable is two bytes.
		{with(func(s *scenario) {
			s.Commands = 1
			s.Words = 1
			s.Path_Bytes = 2
			s.Exit = 1
			s.Fail_Every = 1
		})},
		// Command lines of one word and of the most words, in the document.
		{with(func(s *scenario) { json(s); s.Words = 1 })},
		{with(func(s *scenario) { json(s); s.Words = maddox.COMMAND_WORDS_MAX })},
	}
	for _, group := range groups {
		seeds = append(seeds, group...)
	}
	return seeds
}

// Mutate_outliers is outlier_seed's scenario, for composing with other overrides.
func mutate_outliers(s *scenario, runs int) {
	s.Base = 1000
	s.Runs = runs
	s.Deltas = spike_deltas()
	s.Wall_Pattern = true
}

// SIMULATION_YEAR_RUNS is how many ceiling walls a year of uptime holds, and one more.
const SIMULATION_YEAR_RUNS = int(SIMULATION_YEAR/maddox.METRIC_MAX) + 2

// Seeds_clock runs the virtual clock to its end: kept runs that spend the whole year from its
// start, a second command that starts there, warmups that spend the year, and warmup walls whose
// progress seconds land on one and two fixed-point units.
func seeds_clock() (seeds [][]byte) {
	return [][]byte{
		with(func(s *scenario) {
			s.Runs = SIMULATION_YEAR_RUNS
			s.Sleep = maddox.METRIC_MAX
			s.Progress = true
		}),
		with(func(s *scenario) {
			json(s)
			s.Commands = 1
			s.Runs = SIMULATION_YEAR_RUNS
			s.Sleep = maddox.METRIC_MAX
		}),
		with(func(s *scenario) {
			s.Commands = 1
			s.Runs = 3
			s.Warmup = SIMULATION_YEAR_RUNS
			s.Sleep = maddox.METRIC_MAX
			s.Progress = true
		}),
		with(func(s *scenario) { s.Progress = true; s.Warmup = 1; s.Sleep = 954 }),
		with(func(s *scenario) { s.Progress = true; s.Warmup = 1; s.Sleep = 1908 }),
	}
}

// Seeds_command_line drives the command line's inputs: the inherited environment's sizes, an
// assignment, the duration's extremes, an explicit progress mode, spawns that never start, and
// positional texts that fail to parse.
func seeds_command_line() (seeds [][]byte) {
	environment := func(count int) (data []byte) {
		return with(func(s *scenario) { s.Environment_Count = count; s.Assignment = true })
	}
	duration := func(seconds int64) (data []byte) {
		return with(func(s *scenario) { s.Runs = 3; s.Duration = seconds })
	}
	text := func(texts ...string) (data []byte) {
		return with(func(s *scenario) { s.Commands = 0; s.Texts = texts })
	}
	return [][]byte{
		environment(1), environment(2), environment(SIM_ENVIRONMENT_MAX),
		duration(bits.INTEGER_64_MINIMUM), duration(-1), duration(2),
		duration(bits.INTEGER_64_MAXIMUM),
		with(func(s *scenario) { s.Progress_Never = true }),
		with(func(s *scenario) { s.Spawn_Fail = true }),
		with(func(s *scenario) { s.Spawn_Fail = true; s.Allow_Fail = true; s.Runs = 3 }),
		with(func(s *scenario) { s.Progress = true; s.Exit = 1; s.Fail_Every = 1 }),
		text(""), text(" "), text("a="),
		text("a=" + text_repeat("x", maddox.COMMAND_TEXT_BYTES_MAX-2)),
		text(text_repeat("a ", maddox.COMMAND_TEXT_BYTES_MAX/2-1) + "a"),
		with(func(s *scenario) {
			s.Commands = 1
			s.Words = 1
			s.Path_Bytes = 1
			s.Exit = 1
			s.Fail_Every = 1
		}),
		with(func(s *scenario) {
			s.Commands = 1
			s.Words = 1
			s.Path_Bytes = maddox.COMMAND_WORD_BYTES_MAX
			s.Exit = 1
			s.Fail_Every = 1
		}),
		with(func(s *scenario) {
			s.Commands = 1
			s.Words = 1
			s.Path_Bytes = maddox.COMMAND_WORD_BYTES_MAX
			s.Progress = true
		}),
		with(func(s *scenario) {
			json(s)
			s.Commands = 1
			s.Words = 1
			s.Path_Bytes = maddox.COMMAND_WORD_BYTES_MAX
		}),
	}
}
