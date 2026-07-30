// Package maddox compares the performance of commands on macOS, the way poop does
// on Linux. It is the pure tier of the maddox binary: the statistics, the
// command-to-command comparison, and the JSON report are computed here over
// injected Samples, so this package spawns nothing and reads no clock of its own —
// package main wires the cgo measurer and the operating-system clock and hands them
// in through Main_Input.
package maddox

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/bits"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

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

// RUNS_MIN is the smallest number of samples a command is run, so a spent budget
// still leaves a quorum for the statistics — poop's min_samples.
const RUNS_MIN = 3

// SAMPLES_MAX caps the samples held for one command, bounding memory against a
// command fast enough to run unboundedly within the budget — poop's MAX_SAMPLES. Exported so
// the blackbox simulation sizes its scenarios against the real ceiling rather than a duplicate.
const SAMPLES_MAX = 10000

// Main benchmarks each command in turn — the binary's one entry point — and writes
// the JSON report to Output. The first command is the reference the rest report
// deltas against, matching poop. A command that exits non-zero aborts the run with
// EXIT_FAILURE, its stderr surfaced, unless Allow_Failures is set.
func Main(input Main_Input) (code Exit_Code) {
	defer func() { Exit_Code_Invariants("Main.exit_code", code) }()
	Main_Input_Invariants("Main.input", input)
	benchmarks := make([]Benchmark, 0, len(input.Commands))
	reference := Measurements{}
	have_reference := false
	for index, command := range input.Commands {
		samples, run_exit, child_stderr := main_input_collect_samples(&input, command)
		// Erase the progress line on both paths — before the failure message below or
		// before the report that prints once every command is sampled.
		if input.Progress {
			input.Stderr.Write([]byte(PROGRESS_CLEAR))
		}
		if run_exit != 0 {
			write_failure(&Write_Failure_Input{
				Stderr:       input.Stderr,
				Index:        Position(index),
				Exit:         Failure_Status(run_exit),
				Child_Stderr: child_stderr,
			})
			return EXIT_FAILURE
		}
		measurements := measurements_compute(Distribution(samples))
		benchmark := Benchmark{
			Command:      command_words(command),
			Runs:         Kept(len(samples)),
			Elapsed:      samples_elapsed(Distribution(samples)),
			Measurements: measurements,
		}
		if have_reference {
			deltas := deltas_compute(
				Reference_Measurements(reference),
				Candidate_Measurements(measurements))
			benchmark.Deltas = deltas
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

// Output_Format selects how Main renders the report.
type Output_Format uint8

// Output_Format_Invariants states the two rendering modes by hand so the bundle
// composes under any root: the bound refuses an undeclared mode eagerly, and each
// mode is a witnessed shape of its own.
func Output_Format_Invariants(identifier string, format Output_Format) {
	invariant.Always(format <= OUTPUT_FORMAT_JSON,
		"An output format is one of its declared rendering modes.")
	invariant.Sometimes(identifier, format == OUTPUT_FORMAT_TABLE,
		"The output format is the table.")
	invariant.Sometimes(identifier, format == OUTPUT_FORMAT_JSON,
		"The output format is the JSON document.")
}

// OUTPUT_FORMAT_TABLE renders the human-readable comparison table; the default.
const OUTPUT_FORMAT_TABLE Output_Format = 0

// OUTPUT_FORMAT_JSON renders the machine-readable JSON document.
const OUTPUT_FORMAT_JSON Output_Format = 1

// Sample is one run's measurements. Each field is its own semantic type: the six
// counters share the metric domain but never the same contract slot, so a composed
// Sample seeds a distinct coverage family per field instead of colliding.
type Sample struct {
	// Wall is the run's elapsed time, measured and reported by the sampler that ran it.
	Wall Wall_Time
	// RSS_Bytes_Max is the run's peak physical memory footprint, in bytes.
	RSS_Bytes_Max RSS_Bytes_Max
	// CPU_Cycles is the run's CPU cycle count from the hardware counters.
	CPU_Cycles CPU_Cycles
	// Instructions is the run's retired-instruction count from the counters.
	Instructions Instructions
	// Cache_References is the run's last-level cache reference count (Linux only).
	Cache_References Cache_References
	// Cache_Misses is the run's last-level cache miss count (Linux only).
	Cache_Misses Cache_Misses
	// Branch_Misses is the run's mispredicted-branch count (Linux only).
	Branch_Misses Branch_Misses
	// CPU_User is the run's user-space CPU time.
	CPU_User CPU_User_Time
	// CPU_System is the run's kernel-space CPU time.
	CPU_System CPU_System_Time
}

// Sample_Invariants composes every field's own family.
func Sample_Invariants(identifier string, sample Sample) {
	Wall_Time_Invariants(identifier, sample.Wall)
	RSS_Bytes_Max_Invariants(identifier, sample.RSS_Bytes_Max)
	CPU_Cycles_Invariants(identifier, sample.CPU_Cycles)
	Instructions_Invariants(identifier, sample.Instructions)
	Cache_References_Invariants(identifier, sample.Cache_References)
	Cache_Misses_Invariants(identifier, sample.Cache_Misses)
	Branch_Misses_Invariants(identifier, sample.Branch_Misses)
	CPU_User_Time_Invariants(identifier, sample.CPU_User)
	CPU_System_Time_Invariants(identifier, sample.CPU_System)
}

// Wall_Time is a run's elapsed time in nanoseconds. Declared on the primitive — never on
// time.Duration — because a semantic type owns its contract instead of aliasing another's.
type Wall_Time int64

// Wall_Time_Invariants bounds the span; a measured wall is a monotonic difference, so a
// negative one is sampler garbage to refuse. The old model demanded no witnesses here.
func Wall_Time_Invariants(identifier string, value Wall_Time) {
	invariant.Always(value >= 0, "A wall time is never negative.")
}

// RSS_Bytes_Max is a run's peak physical memory footprint in bytes.
type RSS_Bytes_Max int64

// RSS_Bytes_Max_Invariants bounds the footprint to the metric domain and witnesses the
// shapes the old per-field grid demanded: the boundaries, the small counts, the interior.
func RSS_Bytes_Max_Invariants(identifier string, value RSS_Bytes_Max) {
	invariant.Always(value >= METRIC_MIN, "A peak RSS footprint is never negative.")
	invariant.Always(value <= METRIC_MAX,
		"A peak RSS footprint stays within the representable metric ceiling.")
	invariant.Sometimes(identifier, value == METRIC_MIN, "The peak RSS footprint is zero.")
	invariant.Sometimes(identifier, value == 1, "The peak RSS footprint is one byte.")
	invariant.Sometimes(identifier, value == 2, "The peak RSS footprint is two bytes.")
	invariant.Sometimes(identifier, value == METRIC_MAX,
		"The peak RSS footprint fills the metric ceiling.")
	invariant.Sometimes(identifier, 2 < value && value < METRIC_MAX,
		"The peak RSS footprint is an ordinary size.")
}

// CPU_Cycles is a run's CPU cycle count from the hardware counters.
type CPU_Cycles int64

// CPU_Cycles_Invariants bounds the count to the metric domain and witnesses the old
// per-field grid's shapes.
func CPU_Cycles_Invariants(identifier string, value CPU_Cycles) {
	invariant.Always(value >= METRIC_MIN, "A cycle count is never negative.")
	invariant.Always(value <= METRIC_MAX,
		"A cycle count stays within the representable metric ceiling.")
	invariant.Sometimes(identifier, value == METRIC_MIN, "The cycle count is zero.")
	invariant.Sometimes(identifier, value == 1, "The cycle count is one.")
	invariant.Sometimes(identifier, value == 2, "The cycle count is two.")
	invariant.Sometimes(identifier, value == METRIC_MAX,
		"The cycle count fills the metric ceiling.")
	invariant.Sometimes(identifier, 2 < value && value < METRIC_MAX,
		"The cycle count is an ordinary magnitude.")
}

// Instructions is a run's retired-instruction count from the counters.
type Instructions int64

// Instructions_Invariants bounds the count to the metric domain and witnesses the old
// per-field grid's shapes.
func Instructions_Invariants(identifier string, value Instructions) {
	invariant.Always(value >= METRIC_MIN, "An instruction count is never negative.")
	invariant.Always(value <= METRIC_MAX,
		"An instruction count stays within the representable metric ceiling.")
	invariant.Sometimes(identifier, value == METRIC_MIN, "The instruction count is zero.")
	invariant.Sometimes(identifier, value == 1, "The instruction count is one.")
	invariant.Sometimes(identifier, value == 2, "The instruction count is two.")
	invariant.Sometimes(identifier, value == METRIC_MAX,
		"The instruction count fills the metric ceiling.")
	invariant.Sometimes(identifier, 2 < value && value < METRIC_MAX,
		"The instruction count is an ordinary magnitude.")
}

// Cache_References is a run's last-level cache reference count (Linux only).
type Cache_References int64

// Cache_References_Invariants bounds the count to the metric domain and witnesses the old
// per-field grid's shapes.
func Cache_References_Invariants(identifier string, value Cache_References) {
	invariant.Always(value >= METRIC_MIN, "A cache reference count is never negative.")
	invariant.Always(value <= METRIC_MAX,
		"A cache reference count stays within the representable metric ceiling.")
	invariant.Sometimes(identifier, value == METRIC_MIN,
		"The cache reference count is zero.")
	invariant.Sometimes(identifier, value == 1, "The cache reference count is one.")
	invariant.Sometimes(identifier, value == 2, "The cache reference count is two.")
	invariant.Sometimes(identifier, value == METRIC_MAX,
		"The cache reference count fills the metric ceiling.")
	invariant.Sometimes(identifier, 2 < value && value < METRIC_MAX,
		"The cache reference count is an ordinary magnitude.")
}

// Cache_Misses is a run's last-level cache miss count (Linux only).
type Cache_Misses int64

// Cache_Misses_Invariants bounds the count to the metric domain and witnesses the old
// per-field grid's shapes.
func Cache_Misses_Invariants(identifier string, value Cache_Misses) {
	invariant.Always(value >= METRIC_MIN, "A cache miss count is never negative.")
	invariant.Always(value <= METRIC_MAX,
		"A cache miss count stays within the representable metric ceiling.")
	invariant.Sometimes(identifier, value == METRIC_MIN, "The cache miss count is zero.")
	invariant.Sometimes(identifier, value == 1, "The cache miss count is one.")
	invariant.Sometimes(identifier, value == 2, "The cache miss count is two.")
	invariant.Sometimes(identifier, value == METRIC_MAX,
		"The cache miss count fills the metric ceiling.")
	invariant.Sometimes(identifier, 2 < value && value < METRIC_MAX,
		"The cache miss count is an ordinary magnitude.")
}

// Branch_Misses is a run's mispredicted-branch count (Linux only).
type Branch_Misses int64

// Branch_Misses_Invariants bounds the count to the metric domain and witnesses the old
// per-field grid's shapes.
func Branch_Misses_Invariants(identifier string, value Branch_Misses) {
	invariant.Always(value >= METRIC_MIN, "A branch miss count is never negative.")
	invariant.Always(value <= METRIC_MAX,
		"A branch miss count stays within the representable metric ceiling.")
	invariant.Sometimes(identifier, value == METRIC_MIN, "The branch miss count is zero.")
	invariant.Sometimes(identifier, value == 1, "The branch miss count is one.")
	invariant.Sometimes(identifier, value == 2, "The branch miss count is two.")
	invariant.Sometimes(identifier, value == METRIC_MAX,
		"The branch miss count fills the metric ceiling.")
	invariant.Sometimes(identifier, 2 < value && value < METRIC_MAX,
		"The branch miss count is an ordinary magnitude.")
}

// CPU_User_Time is a run's user-space CPU time in nanoseconds.
type CPU_User_Time int64

// CPU_User_Time_Invariants bounds the span; rusage reports an accumulated non-negative
// time. The old model demanded no witnesses here.
func CPU_User_Time_Invariants(identifier string, value CPU_User_Time) {
	invariant.Always(value >= 0, "A user CPU time is never negative.")
}

// CPU_System_Time is a run's kernel-space CPU time in nanoseconds.
type CPU_System_Time int64

// CPU_System_Time_Invariants bounds the span; rusage reports an accumulated non-negative
// time. The old model demanded no witnesses here.
func CPU_System_Time_Invariants(identifier string, value CPU_System_Time) {
	invariant.Always(value >= 0, "A system CPU time is never negative.")
}

// CAPTURE_BYTES_MIN is the empty capture: a command that wrote nothing to stderr.
const CAPTURE_BYTES_MIN = 0

// CAPTURE_BYTES_MAX is the sampler's stderr capture cap; a verbose failure fills it.
const CAPTURE_BYTES_MAX = 1 << 16

// Captured_Output is a failing command's stderr, read back into a bounded buffer — the
// bytes the command wrote, never trusted, never unbounded.
type Captured_Output []byte

// Captured_Output_Invariants bounds the captured stderr by hand — the type is composed,
// so its texts must be its own — and witnesses the old grid's shapes: the empty min, the
// full-buffer max, the one- and two-byte shapes, and the interior.
func Captured_Output_Invariants(identifier string, output Captured_Output) {
	invariant.Always(len(output) <= CAPTURE_BYTES_MAX,
		"A captured output never exceeds the capture cap.")
	invariant.Sometimes(identifier, len(output) == CAPTURE_BYTES_MIN,
		"The captured output is empty.")
	invariant.Sometimes(identifier, len(output) == 1, "The captured output is one byte.")
	invariant.Sometimes(identifier, len(output) == 2, "The captured output is two bytes.")
	invariant.Sometimes(identifier, len(output) == CAPTURE_BYTES_MAX,
		"The captured output fills the capture cap.")
	invariant.Sometimes(identifier, 2 < len(output) && len(output) < CAPTURE_BYTES_MAX,
		"The captured output is an ordinary length.")
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
	Completed_At time.Moment
	// Exit is the command's exit code; non-zero is a failure.
	Exit Exit_Status
	// Stderr is the command's captured stderr, surfaced on failure.
	Stderr Captured_Output
}

// Run_Result_Invariants composes a Run_Result's fields.
func Run_Result_Invariants(identifier string, result Run_Result) {
	Sample_Invariants(identifier, result.Sample)
	Exit_Status_Invariants(identifier, result.Exit)
	Captured_Output_Invariants(identifier, result.Stderr)
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
// demanded rather than reducing the contract to bound guards; the interior witness keeps the
// old grid's ordinary-count cell demanded, which the bare guard no longer seeds.
func Samples_Invariants(identifier string, samples Samples) {
	invariant.Range(identifier, len(samples), COLLECTION_MIN, SAMPLES_MAX, 1, 2)
	invariant.Sometimes(identifier, 2 < len(samples) && len(samples) < SAMPLES_MAX,
		"The collected count sits strictly between empty and the cap.")
}

// Distribution is a quorum of kept runs the statistics reduce — always at least the 3-run
// minimum, never the empty, one, or two counts the reducers cannot summarize.
type Distribution []Sample

// Distribution_Invariants bounds the run count: the quorum floor and the kept-run ceiling
// are witnessed, every short boundary claimed unreachable, and the interior witness keeps
// the old grid's ordinary-count cell demanded.
func Distribution_Invariants(identifier string, samples Distribution) {
	invariant.Range(identifier, len(samples), QUORUM_MIN, SAMPLES_MAX)
	invariant.Sometimes(identifier, QUORUM_MIN < len(samples) && len(samples) < SAMPLES_MAX,
		"The kept run count sits strictly between the quorum and the cap.")
}

// Values is one metric's value pulled from every sample, the int64 the statistics work in.
type Values []int64

// Values_Invariants bounds the value count. The values are pulled from a kept distribution,
// floored at the 3-run quorum, so the quorum floor and the kept-run ceiling are the witnessed
// shapes; the empty, one, and two counts a quorum never holds fall below the range and are
// guarded, and the interior witness keeps the old grid's ordinary-count cell demanded.
func Values_Invariants(identifier string, values Values) {
	invariant.Range(identifier, len(values), QUORUM_MIN, SAMPLES_MAX)
	invariant.Sometimes(identifier, QUORUM_MIN < len(values) && len(values) < SAMPLES_MAX,
		"The value count sits strictly between the quorum and the cap.")
}

// Series is one metric pulled from a distribution — always the quorum's worth, never the
// empty, one, or two counts a distribution cannot hold.
type Series []int64

// Series_Invariants bounds the extracted count: it mirrors a distribution's quorum floor
// and kept-run ceiling, claiming every short boundary unreachable, with the interior
// witness keeping the old grid's ordinary-count cell demanded.
func Series_Invariants(identifier string, values Series) {
	invariant.Range(identifier, len(values), QUORUM_MIN, SAMPLES_MAX)
	invariant.Sometimes(identifier, QUORUM_MIN < len(values) && len(values) < SAMPLES_MAX,
		"The extracted count sits strictly between the quorum and the cap.")
}

// Deviations is the sorted values the standard deviation reduces — a kept distribution's, so
// always at least the 3-run quorum, never the empty, one, or two counts below it.
type Deviations []int64

// Deviations_Invariants bounds the count. The deviations are a kept distribution's, floored at
// the 3-run quorum, so the quorum floor and the kept-run ceiling are witnessed and the shorter
// counts are guarded out, with the interior witness keeping the old grid's ordinary cell.
func Deviations_Invariants(identifier string, values Deviations) {
	invariant.Range(identifier, len(values), QUORUM_MIN, SAMPLES_MAX)
	invariant.Sometimes(identifier, QUORUM_MIN < len(values) && len(values) < SAMPLES_MAX,
		"The deviation count sits strictly between the quorum and the cap.")
}

// WORD_MIN is the lone executable a command line always carries.
const WORD_MIN = 1

// COMMAND_WORDS_MAX bounds a command line's token count (executable + env + arguments).
const COMMAND_WORDS_MAX = 32

// COMMAND_SET_MAX bounds how many commands/benchmarks one run compares.
const COMMAND_SET_MAX = 8

// Command_Line is one command flattened to its words — the executable and its arguments.
type Command_Line []Command_Word

// Command_Line_Invariants bounds the word count by hand — the type is composed, so its
// texts must be its own; a command always has its executable, so the empty count is
// unreachable while the one-word min, two-word shape, max, and interior are witnessed.
func Command_Line_Invariants(identifier string, words Command_Line) {
	invariant.Always(len(words) >= WORD_MIN, "A command line always carries its executable.")
	invariant.Always(len(words) <= COMMAND_WORDS_MAX,
		"A command line never exceeds its word cap.")
	invariant.Sometimes(identifier, len(words) == WORD_MIN,
		"The command line is the lone executable.")
	invariant.Sometimes(identifier, len(words) == 2, "The command line is two words.")
	invariant.Sometimes(identifier, len(words) == COMMAND_WORDS_MAX,
		"The command line fills its word cap.")
	invariant.Sometimes(identifier, 2 < len(words) && len(words) < COMMAND_WORDS_MAX,
		"The command line is an ordinary length.")
}

// Commands is the set of commands a run benchmarks, in invocation order.
type Commands []sysio.Process_Request

// Commands_Invariants bounds the command count by hand — the type is composed, so its
// texts must be its own: the empty min, the one- and two-command shapes, the max, and
// the interior are witnessed.
func Commands_Invariants(identifier string, commands Commands) {
	invariant.Always(len(commands) <= COMMAND_SET_MAX,
		"A command set never exceeds its cap.")
	invariant.Sometimes(identifier, len(commands) == COLLECTION_MIN,
		"The command set is empty.")
	invariant.Sometimes(identifier, len(commands) == 1, "The command set is one command.")
	invariant.Sometimes(identifier, len(commands) == 2, "The command set is two commands.")
	invariant.Sometimes(identifier, len(commands) == COMMAND_SET_MAX,
		"The command set fills its cap.")
	invariant.Sometimes(identifier, 2 < len(commands) && len(commands) < COMMAND_SET_MAX,
		"The command set is an ordinary size.")
}

// Benchmarks is one entry per benchmarked command, in invocation order.
type Benchmarks []Benchmark

// Benchmarks_Invariants bounds the entry count by hand — the type is composed, so its
// texts must be its own: the empty min, the one- and two-entry shapes, the max, and the
// interior are witnessed.
func Benchmarks_Invariants(identifier string, benchmarks Benchmarks) {
	invariant.Always(len(benchmarks) <= COMMAND_SET_MAX,
		"A benchmark set never exceeds its cap.")
	invariant.Sometimes(identifier, len(benchmarks) == COLLECTION_MIN,
		"The benchmark set is empty.")
	invariant.Sometimes(identifier, len(benchmarks) == 1, "The benchmark set is one entry.")
	invariant.Sometimes(identifier, len(benchmarks) == 2,
		"The benchmark set is two entries.")
	invariant.Sometimes(identifier, len(benchmarks) == COMMAND_SET_MAX,
		"The benchmark set fills its cap.")
	invariant.Sometimes(identifier, 2 < len(benchmarks) && len(benchmarks) < COMMAND_SET_MAX,
		"The benchmark set is an ordinary size.")
}

// REPORT_MAX bounds the rendered bytes at the widest document the pipeline produces: a full
// command set of ceiling-scale, outlier-saturated benchmarks, colored, under a full machine
// header. Every field is at its own ceiling, so no document renders wider.
const REPORT_MAX = 9925

// Report is the rendered output of a document — the table or the JSON bytes.
type Report []byte

// Report_Invariants bounds the rendered length; a report is empty only for an empty
// document and otherwise carries structure, never a lone one or two bytes. The interior
// witness keeps the old grid's ordinary-length cell demanded.
func Report_Invariants(identifier string, report Report) {
	invariant.Range(identifier, len(report), COLLECTION_MIN, REPORT_MAX, 1, 2)
	invariant.Sometimes(identifier, 2 < len(report) && len(report) < REPORT_MAX,
		"The report is an ordinary length.")
}

// LADDER_SIZE is the fixed rung count of every scale ladder.
const LADDER_SIZE = 5

// Ladder is a scale's rungs from largest divisor to smallest, fixed per unit family. A
// fixed array, not a slice: the rung count is part of the type, so the length never varies
// and carries no boundary discipline — only the steady fixed size, asserted below.
type Ladder [LADDER_SIZE]Scale_Step

// Ladder_Invariants states the one thing a ladder's length can be: its fixed size. The array
// type pins the count at compile time, so this is a steady truth, not a witnessed boundary.
func Ladder_Invariants(identifier string, ladder Ladder) {
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
func Sampler_Invariants(identifier string, sampler Sampler) {
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
// capped max, the one- and two-byte shapes between, and the interior the old grid's
// ordinary cell demanded.
func Host_Text_Invariants(identifier string, text Host_Text) {
	invariant.Range(identifier, len(text), HOST_TEXT_BYTES_MIN, HOST_TEXT_BYTES_MAX)
	invariant.Sometimes(identifier, len(text) == 1, "The host text is one byte.")
	invariant.Sometimes(identifier, len(text) == 2, "The host text is two bytes.")
	invariant.Sometimes(identifier, 2 < len(text) && len(text) < HOST_TEXT_BYTES_MAX,
		"The host text is an ordinary length.")
}

// Machine_Specs is a snapshot of the host hardware and OS taken once at startup,
// carried in the report so benchmark results are reproducible across machines. Each
// field is its own semantic type so a composed snapshot seeds a distinct coverage
// family per field, as the old per-field grid instances did.
type Machine_Specs struct {
	// CPU_Model is the CPU's brand string, e.g. "Apple M4 Max".
	CPU_Model CPU_Model `json:"cpu_model"`
	// CPU_Arch is the instruction-set architecture, e.g. "arm64".
	CPU_Arch CPU_Arch `json:"cpu_arch"`
	// Physical_Cores is the total physical core count across all performance levels.
	Physical_Cores Physical_Cores `json:"physical_cores"`
	// Logical_Cores is the OS-visible thread count, which may exceed Physical_Cores
	// when hyperthreading or SMT is active.
	Logical_Cores Logical_Cores `json:"logical_cores"`
	// Performance_Cores is the P-core count on hybrid CPUs (Apple Silicon, Alder Lake+).
	// Zero when the CPU does not expose a performance/efficiency split.
	Performance_Cores Performance_Cores `json:"performance_cores,omitempty"`
	// Efficiency_Cores is the E-core count on hybrid CPUs.
	Efficiency_Cores Efficiency_Cores `json:"efficiency_cores,omitempty"`
	// CPU_Frequency_Hz_Max is the maximum rated CPU frequency in Hz; zero when the
	// kernel does not expose it (e.g. Apple Silicon with no cpufrequency_max sysctl).
	CPU_Frequency_Hz_Max Hertz `json:"cpu_frequency_hz_max,omitempty"`
	// Cache_L1_Bytes is the per-core L1 data cache size in bytes.
	Cache_L1_Bytes Cache_L1_Bytes `json:"cache_l1_bytes,omitempty"`
	// Cache_L2_Bytes is the per-core L2 cache size in bytes.
	Cache_L2_Bytes Cache_L2_Bytes `json:"cache_l2_bytes,omitempty"`
	// Cache_L3_Bytes is the shared L3 cache size in bytes.
	Cache_L3_Bytes Cache_L3_Bytes `json:"cache_l3_bytes,omitempty"`
	// RAM_Total_Bytes is the total installed physical memory in bytes.
	RAM_Total_Bytes RAM_Total_Bytes `json:"ram_total_bytes"`
	// Storage_Total_Bytes is the total capacity of the boot filesystem in bytes. It
	// is a benchmark-relevant proxy for SSD throughput: on Apple Silicon a larger
	// drive spreads I/O across more NAND dies, so the 512GB model reads and writes
	// faster than the 256GB even on identical silicon.
	Storage_Total_Bytes Storage_Total_Bytes `json:"storage_total_bytes"`
	// Operating_System_Name is the operating-system name, e.g. "macOS".
	Operating_System_Name Operating_System_Name `json:"operating_system_name"`
	// Operating_System_Version is the OS release, e.g. "15.2".
	Operating_System_Version Operating_System_Version `json:"operating_system_version"`
	// Kernel_Version is the kernel release string, e.g. "Darwin 25.2.0".
	Kernel_Version Kernel_Version `json:"kernel_version"`
}

// Machine_Specs_Invariants composes every field's own family.
func Machine_Specs_Invariants(identifier string, specs Machine_Specs) {
	CPU_Model_Invariants(identifier, specs.CPU_Model)
	CPU_Arch_Invariants(identifier, specs.CPU_Arch)
	Physical_Cores_Invariants(identifier, specs.Physical_Cores)
	Logical_Cores_Invariants(identifier, specs.Logical_Cores)
	Performance_Cores_Invariants(identifier, specs.Performance_Cores)
	Efficiency_Cores_Invariants(identifier, specs.Efficiency_Cores)
	Hertz_Invariants(identifier, specs.CPU_Frequency_Hz_Max)
	Cache_L1_Bytes_Invariants(identifier, specs.Cache_L1_Bytes)
	Cache_L2_Bytes_Invariants(identifier, specs.Cache_L2_Bytes)
	Cache_L3_Bytes_Invariants(identifier, specs.Cache_L3_Bytes)
	RAM_Total_Bytes_Invariants(identifier, specs.RAM_Total_Bytes)
	Storage_Total_Bytes_Invariants(identifier, specs.Storage_Total_Bytes)
	Operating_System_Name_Invariants(identifier, specs.Operating_System_Name)
	Operating_System_Version_Invariants(identifier, specs.Operating_System_Version)
	Kernel_Version_Invariants(identifier, specs.Kernel_Version)
}

// CPU_Model is the CPU's brand string as probed from the host.
type CPU_Model string

// CPU_Model_Invariants bounds the brand string and witnesses the old per-field grid's
// shapes with its own texts, so the bundle composes under any root.
func CPU_Model_Invariants(identifier string, value CPU_Model) {
	invariant.Always(len(value) <= HOST_TEXT_BYTES_MAX,
		"A CPU model never exceeds the host-text cap.")
	invariant.Sometimes(identifier, len(value) == HOST_TEXT_BYTES_MIN,
		"The CPU model is empty.")
	invariant.Sometimes(identifier, len(value) == 1, "The CPU model is one byte.")
	invariant.Sometimes(identifier, len(value) == 2, "The CPU model is two bytes.")
	invariant.Sometimes(identifier, len(value) == HOST_TEXT_BYTES_MAX,
		"The CPU model fills the host-text cap.")
	invariant.Sometimes(identifier, 2 < len(value) && len(value) < HOST_TEXT_BYTES_MAX,
		"The CPU model is an ordinary length.")
}

// CPU_Arch is the instruction-set architecture string as probed from the host.
type CPU_Arch string

// CPU_Arch_Invariants bounds the architecture string and witnesses the old per-field
// grid's shapes with its own texts.
func CPU_Arch_Invariants(identifier string, value CPU_Arch) {
	invariant.Always(len(value) <= HOST_TEXT_BYTES_MAX,
		"A CPU architecture never exceeds the host-text cap.")
	invariant.Sometimes(identifier, len(value) == HOST_TEXT_BYTES_MIN,
		"The CPU architecture is empty.")
	invariant.Sometimes(identifier, len(value) == 1, "The CPU architecture is one byte.")
	invariant.Sometimes(identifier, len(value) == 2, "The CPU architecture is two bytes.")
	invariant.Sometimes(identifier, len(value) == HOST_TEXT_BYTES_MAX,
		"The CPU architecture fills the host-text cap.")
	invariant.Sometimes(identifier, 2 < len(value) && len(value) < HOST_TEXT_BYTES_MAX,
		"The CPU architecture is an ordinary length.")
}

// Physical_Cores is the host's total physical core count.
type Physical_Cores int

// Physical_Cores_Invariants bounds the count to the realistic topology and witnesses
// the old per-field grid's shapes with its own texts.
func Physical_Cores_Invariants(identifier string, value Physical_Cores) {
	invariant.Always(int(value) >= CORES_COUNT_MIN,
		"A physical core count is never negative.")
	invariant.Always(int(value) <= CORES_COUNT_MAX,
		"A physical core count stays within the realistic topology.")
	invariant.Sometimes(identifier, int(value) == CORES_COUNT_MIN,
		"The physical core count is zero.")
	invariant.Sometimes(identifier, int(value) == 1, "The physical core count is one.")
	invariant.Sometimes(identifier, int(value) == 2, "The physical core count is two.")
	invariant.Sometimes(identifier, int(value) == CORES_COUNT_MAX,
		"The physical core count fills the topology ceiling.")
	invariant.Sometimes(identifier, 2 < int(value) && int(value) < CORES_COUNT_MAX,
		"The physical core count is an ordinary topology.")
}

// Logical_Cores is the host's OS-visible thread count.
type Logical_Cores int

// Logical_Cores_Invariants bounds the count to the realistic topology and witnesses
// the old per-field grid's shapes with its own texts.
func Logical_Cores_Invariants(identifier string, value Logical_Cores) {
	invariant.Always(int(value) >= CORES_COUNT_MIN,
		"A logical core count is never negative.")
	invariant.Always(int(value) <= CORES_COUNT_MAX,
		"A logical core count stays within the realistic topology.")
	invariant.Sometimes(identifier, int(value) == CORES_COUNT_MIN,
		"The logical core count is zero.")
	invariant.Sometimes(identifier, int(value) == 1, "The logical core count is one.")
	invariant.Sometimes(identifier, int(value) == 2, "The logical core count is two.")
	invariant.Sometimes(identifier, int(value) == CORES_COUNT_MAX,
		"The logical core count fills the topology ceiling.")
	invariant.Sometimes(identifier, 2 < int(value) && int(value) < CORES_COUNT_MAX,
		"The logical core count is an ordinary topology.")
}

// Performance_Cores is the host's P-core count on a hybrid CPU.
type Performance_Cores int

// Performance_Cores_Invariants bounds the count to the realistic topology and witnesses
// the old per-field grid's shapes with its own texts.
func Performance_Cores_Invariants(identifier string, value Performance_Cores) {
	invariant.Always(int(value) >= CORES_COUNT_MIN,
		"A performance core count is never negative.")
	invariant.Always(int(value) <= CORES_COUNT_MAX,
		"A performance core count stays within the realistic topology.")
	invariant.Sometimes(identifier, int(value) == CORES_COUNT_MIN,
		"The performance core count is zero.")
	invariant.Sometimes(identifier, int(value) == 1, "The performance core count is one.")
	invariant.Sometimes(identifier, int(value) == 2, "The performance core count is two.")
	invariant.Sometimes(identifier, int(value) == CORES_COUNT_MAX,
		"The performance core count fills the topology ceiling.")
	invariant.Sometimes(identifier, 2 < int(value) && int(value) < CORES_COUNT_MAX,
		"The performance core count is an ordinary topology.")
}

// Efficiency_Cores is the host's E-core count on a hybrid CPU.
type Efficiency_Cores int

// Efficiency_Cores_Invariants bounds the count to the realistic topology and witnesses
// the old per-field grid's shapes with its own texts.
func Efficiency_Cores_Invariants(identifier string, value Efficiency_Cores) {
	invariant.Always(int(value) >= CORES_COUNT_MIN,
		"An efficiency core count is never negative.")
	invariant.Always(int(value) <= CORES_COUNT_MAX,
		"An efficiency core count stays within the realistic topology.")
	invariant.Sometimes(identifier, int(value) == CORES_COUNT_MIN,
		"The efficiency core count is zero.")
	invariant.Sometimes(identifier, int(value) == 1, "The efficiency core count is one.")
	invariant.Sometimes(identifier, int(value) == 2, "The efficiency core count is two.")
	invariant.Sometimes(identifier, int(value) == CORES_COUNT_MAX,
		"The efficiency core count fills the topology ceiling.")
	invariant.Sometimes(identifier, 2 < int(value) && int(value) < CORES_COUNT_MAX,
		"The efficiency core count is an ordinary topology.")
}

// Cache_L1_Bytes is the host's per-core L1 data cache size in bytes.
type Cache_L1_Bytes uint64

// Cache_L1_Bytes_Invariants bounds the size to the realistic ceiling and witnesses the
// old per-field grid's shapes with its own texts.
func Cache_L1_Bytes_Invariants(identifier string, value Cache_L1_Bytes) {
	invariant.Always(uint64(value) <= BYTE_SIZE_MAX,
		"An L1 cache size stays within the realistic ceiling.")
	invariant.Sometimes(identifier, uint64(value) == BYTE_SIZE_MIN,
		"The L1 cache size is unreported.")
	invariant.Sometimes(identifier, uint64(value) == 1, "The L1 cache size is one byte.")
	invariant.Sometimes(identifier, uint64(value) == 2, "The L1 cache size is two bytes.")
	invariant.Sometimes(identifier, uint64(value) == BYTE_SIZE_MAX,
		"The L1 cache size fills the realistic ceiling.")
	invariant.Sometimes(identifier, 2 < uint64(value) && uint64(value) < BYTE_SIZE_MAX,
		"The L1 cache size is an ordinary size.")
}

// Cache_L2_Bytes is the host's per-core L2 cache size in bytes.
type Cache_L2_Bytes uint64

// Cache_L2_Bytes_Invariants bounds the size to the realistic ceiling and witnesses the
// old per-field grid's shapes with its own texts.
func Cache_L2_Bytes_Invariants(identifier string, value Cache_L2_Bytes) {
	invariant.Always(uint64(value) <= BYTE_SIZE_MAX,
		"An L2 cache size stays within the realistic ceiling.")
	invariant.Sometimes(identifier, uint64(value) == BYTE_SIZE_MIN,
		"The L2 cache size is unreported.")
	invariant.Sometimes(identifier, uint64(value) == 1, "The L2 cache size is one byte.")
	invariant.Sometimes(identifier, uint64(value) == 2, "The L2 cache size is two bytes.")
	invariant.Sometimes(identifier, uint64(value) == BYTE_SIZE_MAX,
		"The L2 cache size fills the realistic ceiling.")
	invariant.Sometimes(identifier, 2 < uint64(value) && uint64(value) < BYTE_SIZE_MAX,
		"The L2 cache size is an ordinary size.")
}

// Cache_L3_Bytes is the host's shared L3 cache size in bytes.
type Cache_L3_Bytes uint64

// Cache_L3_Bytes_Invariants bounds the size to the realistic ceiling and witnesses the
// old per-field grid's shapes with its own texts.
func Cache_L3_Bytes_Invariants(identifier string, value Cache_L3_Bytes) {
	invariant.Always(uint64(value) <= BYTE_SIZE_MAX,
		"An L3 cache size stays within the realistic ceiling.")
	invariant.Sometimes(identifier, uint64(value) == BYTE_SIZE_MIN,
		"The L3 cache size is unreported.")
	invariant.Sometimes(identifier, uint64(value) == 1, "The L3 cache size is one byte.")
	invariant.Sometimes(identifier, uint64(value) == 2, "The L3 cache size is two bytes.")
	invariant.Sometimes(identifier, uint64(value) == BYTE_SIZE_MAX,
		"The L3 cache size fills the realistic ceiling.")
	invariant.Sometimes(identifier, 2 < uint64(value) && uint64(value) < BYTE_SIZE_MAX,
		"The L3 cache size is an ordinary size.")
}

// RAM_Total_Bytes is the host's total installed physical memory in bytes.
type RAM_Total_Bytes uint64

// RAM_Total_Bytes_Invariants bounds the size to the realistic ceiling and witnesses the
// old per-field grid's shapes with its own texts.
func RAM_Total_Bytes_Invariants(identifier string, value RAM_Total_Bytes) {
	invariant.Always(uint64(value) <= BYTE_SIZE_MAX,
		"A RAM total stays within the realistic ceiling.")
	invariant.Sometimes(identifier, uint64(value) == BYTE_SIZE_MIN,
		"The RAM total is unreported.")
	invariant.Sometimes(identifier, uint64(value) == 1, "The RAM total is one byte.")
	invariant.Sometimes(identifier, uint64(value) == 2, "The RAM total is two bytes.")
	invariant.Sometimes(identifier, uint64(value) == BYTE_SIZE_MAX,
		"The RAM total fills the realistic ceiling.")
	invariant.Sometimes(identifier, 2 < uint64(value) && uint64(value) < BYTE_SIZE_MAX,
		"The RAM total is an ordinary size.")
}

// Storage_Total_Bytes is the host's boot-filesystem capacity in bytes.
type Storage_Total_Bytes uint64

// Storage_Total_Bytes_Invariants bounds the size to the realistic ceiling and witnesses
// the old per-field grid's shapes with its own texts.
func Storage_Total_Bytes_Invariants(identifier string, value Storage_Total_Bytes) {
	invariant.Always(uint64(value) <= BYTE_SIZE_MAX,
		"A storage total stays within the realistic ceiling.")
	invariant.Sometimes(identifier, uint64(value) == BYTE_SIZE_MIN,
		"The storage total is unreported.")
	invariant.Sometimes(identifier, uint64(value) == 1, "The storage total is one byte.")
	invariant.Sometimes(identifier, uint64(value) == 2, "The storage total is two bytes.")
	invariant.Sometimes(identifier, uint64(value) == BYTE_SIZE_MAX,
		"The storage total fills the realistic ceiling.")
	invariant.Sometimes(identifier, 2 < uint64(value) && uint64(value) < BYTE_SIZE_MAX,
		"The storage total is an ordinary size.")
}

// Operating_System_Name is the host's operating-system name.
type Operating_System_Name string

// Operating_System_Name_Invariants bounds the name and witnesses the old per-field
// grid's shapes with its own texts.
func Operating_System_Name_Invariants(identifier string, value Operating_System_Name) {
	invariant.Always(len(value) <= HOST_TEXT_BYTES_MAX,
		"An operating-system name never exceeds the host-text cap.")
	invariant.Sometimes(identifier, len(value) == HOST_TEXT_BYTES_MIN,
		"The operating-system name is empty.")
	invariant.Sometimes(identifier, len(value) == 1,
		"The operating-system name is one byte.")
	invariant.Sometimes(identifier, len(value) == 2,
		"The operating-system name is two bytes.")
	invariant.Sometimes(identifier, len(value) == HOST_TEXT_BYTES_MAX,
		"The operating-system name fills the host-text cap.")
	invariant.Sometimes(identifier, 2 < len(value) && len(value) < HOST_TEXT_BYTES_MAX,
		"The operating-system name is an ordinary length.")
}

// Operating_System_Version is the host's OS release string.
type Operating_System_Version string

// Operating_System_Version_Invariants bounds the release and witnesses the old
// per-field grid's shapes with its own texts.
func Operating_System_Version_Invariants(
	identifier string, value Operating_System_Version,
) {
	invariant.Always(len(value) <= HOST_TEXT_BYTES_MAX,
		"An operating-system version never exceeds the host-text cap.")
	invariant.Sometimes(identifier, len(value) == HOST_TEXT_BYTES_MIN,
		"The operating-system version is empty.")
	invariant.Sometimes(identifier, len(value) == 1,
		"The operating-system version is one byte.")
	invariant.Sometimes(identifier, len(value) == 2,
		"The operating-system version is two bytes.")
	invariant.Sometimes(identifier, len(value) == HOST_TEXT_BYTES_MAX,
		"The operating-system version fills the host-text cap.")
	invariant.Sometimes(identifier, 2 < len(value) && len(value) < HOST_TEXT_BYTES_MAX,
		"The operating-system version is an ordinary length.")
}

// Kernel_Version is the host's kernel release string.
type Kernel_Version string

// Kernel_Version_Invariants bounds the release and witnesses the old per-field grid's
// shapes with its own texts.
func Kernel_Version_Invariants(identifier string, value Kernel_Version) {
	invariant.Always(len(value) <= HOST_TEXT_BYTES_MAX,
		"A kernel version never exceeds the host-text cap.")
	invariant.Sometimes(identifier, len(value) == HOST_TEXT_BYTES_MIN,
		"The kernel version is empty.")
	invariant.Sometimes(identifier, len(value) == 1, "The kernel version is one byte.")
	invariant.Sometimes(identifier, len(value) == 2, "The kernel version is two bytes.")
	invariant.Sometimes(identifier, len(value) == HOST_TEXT_BYTES_MAX,
		"The kernel version fills the host-text cap.")
	invariant.Sometimes(identifier, 2 < len(value) && len(value) < HOST_TEXT_BYTES_MAX,
		"The kernel version is an ordinary length.")
}

// UNIT_BYTES_MIN is the shortest unit in the vocabulary — "count" and "bytes", five bytes
// each. Every rendered unit is one of the closed vocabulary, so no shorter width ever occurs.
const UNIT_BYTES_MIN = 5

// UNIT_BYTES_MAX is the longest unit in the vocabulary — "nanoseconds", eleven bytes.
// Units are a closed set, so a longer value is a malformed probe to reject.
const UNIT_BYTES_MAX = 11

// Unit names the dimension a Measurement's raw values are in — a word from a closed vocabulary
// of exactly two widths: the five-byte "count"/"bytes" and the eleven-byte "nanoseconds".
type Unit string

// Unit_Invariants states the two-width vocabulary by hand — the type is composed, so its
// texts must be its own: the membership bound refuses an invented third width eagerly,
// and each real width is a witnessed shape.
func Unit_Invariants(identifier string, name Unit) {
	invariant.Always(len(name) == UNIT_BYTES_MIN || len(name) == UNIT_BYTES_MAX,
		"A unit is one of the vocabulary's two widths.")
	invariant.Sometimes(identifier, len(name) == UNIT_BYTES_MIN,
		"The unit is a five-byte word.")
	invariant.Sometimes(identifier, len(name) == UNIT_BYTES_MAX,
		"The unit is the eleven-byte nanoseconds.")
}

// Measurement is the distribution of one metric across a command's runs — poop's
// Measurement, reduced from the raw Samples. It is the compute and compare currency;
// the report serializes the per-metric variant structs, so its statistics need no
// custom encoders. Each field is its own semantic type so a composed Measurement
// seeds a distinct family per field.
type Measurement struct {
	// Mean is the arithmetic mean of the metric's values.
	Mean Mean `json:"mean"`
	// Standard_Deviation is the sample standard deviation, with an n-1 denominator.
	Standard_Deviation Standard_Deviation `json:"stddev"`
	// Min is the smallest value observed.
	Min Min `json:"min"`
	// Max is the largest value observed.
	Max Max `json:"max"`
	// Median is the middle value of the sorted values.
	Median Median `json:"median"`
	// Q1 is the first quartile by poop's index math.
	Q1 Q1 `json:"q1"`
	// Q3 is the third quartile by poop's index math.
	Q3 Q3 `json:"q3"`
	// Outlier_Count is how many values fall beyond Tukey's fences.
	Outlier_Count Strays `json:"outliers"`
	// Sample_Count is how many values the distribution was computed from.
	Sample_Count Kept `json:"count"`
	// Unit names the unit the raw values are in.
	Unit Unit `json:"unit"`
}

// Measurement_Invariants composes every field's own family.
func Measurement_Invariants(identifier string, measurement Measurement) {
	Mean_Invariants(identifier, measurement.Mean)
	Standard_Deviation_Invariants(identifier, measurement.Standard_Deviation)
	Min_Invariants(identifier, measurement.Min)
	Max_Invariants(identifier, measurement.Max)
	Median_Invariants(identifier, measurement.Median)
	Q1_Invariants(identifier, measurement.Q1)
	Q3_Invariants(identifier, measurement.Q3)
	Strays_Invariants(identifier, measurement.Outlier_Count)
	Kept_Invariants(identifier, measurement.Sample_Count)
	Unit_Invariants(identifier, measurement.Unit)
}

// Mean is a distribution's arithmetic mean, fixedpoint-scaled.
type Mean int64

// Mean_Invariants bounds the mean; the statistics summarize non-negative metrics, so
// their mean never dips below zero. The old model demanded no witnesses here.
func Mean_Invariants(identifier string, value Mean) {
	invariant.Always(value >= 0, "A distribution mean is never negative.")
}

// Standard_Deviation is a distribution's sample standard deviation, fixedpoint-scaled.
type Standard_Deviation int64

// Standard_Deviation_Invariants bounds the deviation; a root of squares is never
// negative. The old model demanded no witnesses here.
func Standard_Deviation_Invariants(identifier string, value Standard_Deviation) {
	invariant.Always(value >= 0, "A distribution standard deviation is never negative.")
}

// Min is a distribution's smallest observed value, fixedpoint-scaled.
type Min int64

// Min_Invariants bounds the minimum; the metrics it summarizes are non-negative. The
// old model demanded no witnesses here.
func Min_Invariants(identifier string, value Min) {
	invariant.Always(value >= 0, "A distribution minimum is never negative.")
}

// Max is a distribution's largest observed value, fixedpoint-scaled.
type Max int64

// Max_Invariants bounds the maximum; the metrics it summarizes are non-negative. The
// old model demanded no witnesses here.
func Max_Invariants(identifier string, value Max) {
	invariant.Always(value >= 0, "A distribution maximum is never negative.")
}

// Median is a distribution's middle sorted value, fixedpoint-scaled.
type Median int64

// Median_Invariants bounds the median; the metrics it summarizes are non-negative.
// The old model demanded no witnesses here.
func Median_Invariants(identifier string, value Median) {
	invariant.Always(value >= 0, "A distribution median is never negative.")
}

// Q1 is a distribution's first quartile by poop's index math, fixedpoint-scaled.
type Q1 int64

// Q1_Invariants bounds the quartile; the metrics it summarizes are non-negative. The
// old model demanded no witnesses here.
func Q1_Invariants(identifier string, value Q1) {
	invariant.Always(value >= 0, "A distribution first quartile is never negative.")
}

// Q3 is a distribution's third quartile by poop's index math, fixedpoint-scaled.
type Q3 int64

// Q3_Invariants bounds the quartile; the metrics it summarizes are non-negative. The
// old model demanded no witnesses here.
func Q3_Invariants(identifier string, value Q3) {
	invariant.Always(value >= 0, "A distribution third quartile is never negative.")
}

// Delta is one metric's change in a candidate command relative to the reference —
// poop's colored ratio column, as data. It is the compare and render currency; the
// report serializes the per-metric variant structs. Each field is its own semantic
// type so a composed Delta seeds a distinct family per field.
type Delta struct {
	// Diff_Percent is the candidate's mean as a signed percentage of the reference's.
	Diff_Percent Diff_Percent `json:"diff_percent"`
	// Half_Percent is the half-width of the 95% confidence interval on Diff_Percent.
	Half_Percent Half_Percent `json:"half_percent"`
	// Significant is true only when the interval clears the ±1% band.
	Significant Significant `json:"significant"`
	// Faster is true when the candidate's mean is below the reference's.
	Faster Faster `json:"faster"`
}

// Delta_Invariants composes every field's own family.
func Delta_Invariants(identifier string, delta Delta) {
	Diff_Percent_Invariants(identifier, delta.Diff_Percent)
	Half_Percent_Invariants(identifier, delta.Half_Percent)
	Significant_Invariants(identifier, delta.Significant)
	Faster_Invariants(identifier, delta.Faster)
}

// Diff_Percent is a candidate mean's signed percentage difference against the
// reference, fixedpoint-scaled.
type Diff_Percent int64

// Diff_Percent_Invariants bounds the difference; against a non-negative reference it
// cannot fall below −100%. The old model demanded no witnesses here.
func Diff_Percent_Invariants(identifier string, value Diff_Percent) {
	invariant.Always(value >= DIFF_PERCENT_MIN,
		"A diff percent never falls below negative one hundred percent.")
}

// Half_Percent is a confidence interval's half-width as a percentage,
// fixedpoint-scaled.
type Half_Percent int64

// Half_Percent_Invariants states the one property the width carries: it is a
// confidence half-width, so it is never negative.
func Half_Percent_Invariants(identifier string, value Half_Percent) {
	invariant.Always(value >= 0, "A confidence half-interval is never negative.")
}

// Significant is the verdict of the ±1% significance band.
type Significant bool

// Significant_Invariants witnesses both verdicts of the band, carrying the demand the
// old Delta.Significant grid stated.
func Significant_Invariants(identifier string, value Significant) {
	invariant.Sometimes(identifier, bool(value),
		"The delta clears the significance band.")
}

// Faster is the verdict of the mean comparison: the candidate's mean sits below the
// reference's.
type Faster bool

// Faster_Invariants witnesses both verdicts of the comparison, carrying the demand
// the old Delta.Faster grid stated.
func Faster_Invariants(identifier string, value Faster) {
	invariant.Sometimes(identifier, bool(value),
		"The candidate mean sits below the reference mean.")
}

// Benchmark is one command's entry in the report.
type Benchmark struct {
	// Command is the command's words, as a reader recognizes them.
	Command Command_Line `json:"command"`
	// Runs is how many samples were kept, warmup excluded.
	Runs Kept `json:"runs"`
	// Elapsed is the total wall time of the kept runs.
	Elapsed time.Duration `json:"elapsed_ns"`
	// Measurements is the per-metric distribution.
	Measurements Measurements `json:"measurements"`
	// Deltas is the change against the reference; the reference compared against
	// itself is the zero delta. Which command is the reference is positional — the
	// first — not encoded here, so every benchmark carries a comparable delta.
	Deltas Deltas `json:"deltas"`
}

// Benchmark_Invariants composes a Benchmark's fields, the delta distribution like any
// other value field.
func Benchmark_Invariants(identifier string, benchmark Benchmark) {
	Command_Line_Invariants(identifier, benchmark.Command)
	Kept_Invariants(identifier, benchmark.Runs)
	Measurements_Invariants(identifier, benchmark.Measurements)
	Deltas_Invariants(identifier, benchmark.Deltas)
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
func Document_Invariants(identifier string, document Document) {
	Machine_Specs_Invariants(identifier, document.Machine)
	Benchmarks_Invariants(identifier, document.Benchmarks)
}

// Main_Input carries the injected dependencies Main needs, so the library tier
// spawns nothing and reads no ambient clock.
type Main_Input struct {
	// Commands are the commands to benchmark; the first is the reference.
	Commands Commands
	// Sampler runs and measures one command, reporting its wall time; production wires
	// the cgo measurer. Main never reads a clock — only the driver in func main advances
	// one — so wall time is a measurement the sampler returns, not a clock delta.
	Sampler Sampler
	// Duration_Max is the per-command time budget; 0 disables it, leaving Runs_Max.
	Duration_Max time.Duration
	// Runs_Max is the per-command run cap; 0 disables it, leaving Duration_Max. The
	// 3-run minimum still applies. Sampling stops at whichever limit is met first.
	Runs_Max Runs_Max
	// Warmup_Count is how many runs are taken and discarded before sampling.
	Warmup_Count Warmup_Count
	// Allow_Failures keeps benchmarking a command that exits non-zero.
	Allow_Failures Allow_Failures
	// Format selects the report rendering; the zero value is the table.
	Format Output_Format
	// Color enables ANSI color in the table rendering.
	Color Color
	// Progress writes an in-place run counter to Stderr while sampling; gate it on an
	// interactive Stderr so piped output stays clean.
	Progress Progress
	// Output is where the report is written. Its own interface declaration, distinct
	// from Diagnostic_Sink so the two writer fields are distinct types.
	Output Report_Sink
	// Stderr is where a failing command's diagnostics are written.
	Stderr Diagnostic_Sink
	// Machine is the host hardware and OS snapshot; injected so the library makes no
	// ambient OS reads. Production wires acquire_machine_specs(); tests inject a stub.
	Machine Machine_Specs
}

// Main_Input_Invariants composes the injected configuration's fields; the clock,
// sampler function, and writers carry no bundle of their own.
func Main_Input_Invariants(identifier string, input Main_Input) {
	Commands_Invariants(identifier, input.Commands)
	Sampler_Invariants(identifier, input.Sampler)
	Runs_Max_Invariants(identifier, input.Runs_Max)
	Warmup_Count_Invariants(identifier, input.Warmup_Count)
	Allow_Failures_Invariants(identifier, input.Allow_Failures)
	Output_Format_Invariants(identifier, input.Format)
	Color_Invariants(identifier, input.Color)
	Progress_Invariants(identifier, input.Progress)
	Machine_Specs_Invariants(identifier, input.Machine)
}

// Report_Sink is where the report is written.
type Report_Sink interface{ io.Writer }

// Diagnostic_Sink is where a failing command's diagnostics are written.
type Diagnostic_Sink interface{ io.Writer }

// Runs_Max is the per-command run cap; zero disables it.
type Runs_Max int

// Runs_Max_Invariants witnesses the shapes the old full-width grid demanded — the
// unit values and the word extremes — with the cap's own texts; the full-width
// domain enforces nothing, so no vacuous bound accompanies them.
func Runs_Max_Invariants(identifier string, value Runs_Max) {
	invariant.Sometimes(identifier, value == 1, "The run cap is one.")
	invariant.Sometimes(identifier, value == -1, "The run cap is negative one.")
	invariant.Sometimes(identifier, int64(value) == math.MinInt64,
		"The run cap is the word minimum.")
	invariant.Sometimes(identifier, int64(value) == math.MaxInt64,
		"The run cap is the word maximum.")
	invariant.Sometimes(identifier,
		value != 1 && value != -1 &&
			int64(value) != math.MinInt64 && int64(value) != math.MaxInt64,
		"The run cap is an ordinary count.")
}

// Warmup_Count is how many runs are taken and discarded before sampling.
type Warmup_Count int

// Warmup_Count_Invariants witnesses the shapes the old full-width grid demanded with
// the count's own texts; the full-width domain enforces nothing.
func Warmup_Count_Invariants(identifier string, value Warmup_Count) {
	invariant.Sometimes(identifier, value == 1, "The warmup count is one.")
	invariant.Sometimes(identifier, value == -1, "The warmup count is negative one.")
	invariant.Sometimes(identifier, int64(value) == math.MinInt64,
		"The warmup count is the word minimum.")
	invariant.Sometimes(identifier, int64(value) == math.MaxInt64,
		"The warmup count is the word maximum.")
	invariant.Sometimes(identifier,
		value != 1 && value != -1 &&
			int64(value) != math.MinInt64 && int64(value) != math.MaxInt64,
		"The warmup count is an ordinary count.")
}

// Allow_Failures keeps benchmarking a command that exits non-zero.
type Allow_Failures bool

// Allow_Failures_Invariants witnesses both settings, carrying the demand the old
// Main_Input.Allow_Failures grid stated.
func Allow_Failures_Invariants(identifier string, value Allow_Failures) {
	invariant.Sometimes(identifier, bool(value), "A failing command is kept in the run.")
}

// Color enables ANSI color in the table rendering.
type Color bool

// Color_Invariants witnesses both settings, carrying the demand the old color grids
// stated at every color-flag site.
func Color_Invariants(identifier string, value Color) {
	invariant.Sometimes(identifier, bool(value), "The rendering carries ANSI color.")
}

// Progress enables the in-place run counter on the diagnostic sink.
type Progress bool

// Progress_Invariants witnesses both settings, carrying the demand the old
// Main_Input.Progress grid stated.
func Progress_Invariants(identifier string, value Progress) {
	invariant.Sometimes(identifier, bool(value), "The progress line renders.")
}

// Main_input_collect_samples runs command through the warmup discards and then the
// measured loop, timing each kept run with the injected clock. A non-zero exit
// returns that exit and its stderr to abort the run; with Allow_Failures the run is
// kept and a zero exit is returned so sampling continues.
func main_input_collect_samples(
	input *Main_Input, command sysio.Process_Request,
) (samples Samples, exit Exit_Status, stderr Captured_Output) {
	defer func() {
		Samples_Invariants("main_input_collect_samples.samples", samples)
		Exit_Status_Invariants("main_input_collect_samples.exit", exit)
		Captured_Output_Invariants("main_input_collect_samples.stderr", stderr)
	}()
	Main_Input_Invariants("main_input_collect_samples.input", *input)
	warmups := 0
	// Elapsed is the running sum of measured wall time, the budget's clock: the library
	// reads no ambient clock, so a run's cost is the wall the sampler reports, accrued.
	var warmup_elapsed time.Duration
	for warmups < int(input.Warmup_Count) {
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
		warmup_elapsed += time.Duration(warm.Sample.Wall)
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
	var elapsed time.Duration
	var stopwatch_start time.Moment
	for sampling_should_continue(&Sampling_Should_Continue_Input{
		Duration_Max: input.Duration_Max,
		Runs_Max:     input.Runs_Max,
		Count:        Tally(len(samples)),
		Elapsed:      Sampling_Elapsed(elapsed),
	}) {
		result := input.Sampler.Measure(command)
		Run_Result_Invariants("main_input_collect_samples.result", result)
		if result.Exit != 0 {
			if !input.Allow_Failures {
				return nil, result.Exit, result.Stderr
			}
		}
		sample := result.Sample
		if len(samples) == 0 {
			// Anchor at the first run's start so between-run gaps count in the budget.
			stopwatch_start = result.Completed_At - time.Moment(sample.Wall)
		}
		samples = append(samples, sample)
		elapsed = time.Duration(result.Completed_At - stopwatch_start)
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
	// Elapsed is the wall time spent sampling so far. Its own semantic type, distinct
	// from the budget so the two duration fields are distinct types.
	Elapsed Sampling_Elapsed
	// Duration_Max is the time budget; zero disables it.
	Duration_Max time.Duration
	// Runs_Max is the run cap; zero disables it.
	Runs_Max Runs_Max
	// Count is how many runs have been kept so far.
	Count Tally
}

// Sampling_Should_Continue_Input_Invariants composes the loop state's fields; the
// budget duration carries no bundle of its own.
func Sampling_Should_Continue_Input_Invariants(
	identifier string, input Sampling_Should_Continue_Input,
) {
	Sampling_Elapsed_Invariants(identifier, input.Elapsed)
	Runs_Max_Invariants(identifier, input.Runs_Max)
	Tally_Invariants(identifier, input.Count)
}

// Sampling_Elapsed is the wall time spent sampling so far, in nanoseconds. Declared on
// the primitive — never on time.Duration — because a semantic type owns its contract.
type Sampling_Elapsed int64

// Sampling_Elapsed_Invariants observes the stopwatch's sign rather than bounding it:
// the sampler owns the completion stamps, and a stamp source running backward — a
// hostile or broken clock — is a state the loop must survive, not crash on, exactly
// as the old model accepted it silently.
func Sampling_Elapsed_Invariants(identifier string, value Sampling_Elapsed) {
	invariant.Sometimes(identifier, value >= 0, "The sampling stopwatch reads non-negative.")
}

// Sampling_should_continue decides whether to take another sample. The 3-run minimum
// always wins first and the 10000-run cap always stops; between them, sampling stops
// when any active limit is met — the run cap or the time budget — and a limit of zero
// is inactive, so both zero leaves only the safety cap. The compound condition is
// split into nested single-term ifs for the linter.
func sampling_should_continue(input *Sampling_Should_Continue_Input) (yes Should_Continue) {
	defer func() {
		Should_Continue_Invariants("sampling_should_continue.yes", yes)
	}()
	Sampling_Should_Continue_Input_Invariants("sampling_should_continue.input", *input)
	if int(input.Count) < RUNS_MIN {
		return true
	}
	if int(input.Count) >= SAMPLES_MAX {
		return false
	}
	if input.Runs_Max > 0 {
		if int(input.Count) >= int(input.Runs_Max) {
			return false
		}
	}
	if input.Duration_Max > 0 {
		if time.Duration(input.Elapsed) >= input.Duration_Max {
			return false
		}
	}
	return true
}

// Should_Continue is the sampling loop's verdict: take another run or stop.
type Should_Continue bool

// Should_Continue_Invariants witnesses both verdicts, carrying the demand the old
// sampling_should_continue.yes grid stated.
func Should_Continue_Invariants(identifier string, value Should_Continue) {
	invariant.Sometimes(identifier, bool(value), "The sampler takes another run.")
}

// Samples_elapsed sums the wall time of the kept runs — how long the command's
// measured sampling took in total.
func samples_elapsed(samples Distribution) (elapsed time.Duration) {
	Distribution_Invariants("samples_elapsed.samples", samples)
	for _, sample := range samples {
		elapsed += time.Duration(sample.Wall)
	}
	return elapsed
}

// Measurements_compute reduces the samples to one Measurement per metric, tagging
// each with the unit its raw values are in.
func measurements_compute(samples Distribution) (measurements Measurements) {
	defer func() {
		Measurements_Invariants("measurements_compute.measurements", measurements)
	}()
	Distribution_Invariants("measurements_compute.samples", samples)
	measurements.Wall_Time = Measurement_Compute(
		Values(extract(samples, sample_wall)), "nanoseconds").As_Wall_Time()
	measurements.Peak_RSS = Measurement_Compute(
		Values(extract(samples, sample_rss)), "bytes").As_Peak_RSS()
	measurements.CPU_Cycles = Measurement_Compute(
		Values(extract(samples, sample_cycles)), "count").As_CPU_Cycles()
	measurements.Instructions = Measurement_Compute(
		Values(extract(samples, sample_instructions)), "count").As_Instructions()
	measurements.Cache_References = Measurement_Compute(
		Values(extract(samples, sample_cache_references)), "count").As_Cache_References()
	measurements.Cache_Misses = Measurement_Compute(
		Values(extract(samples, sample_cache_misses)), "count").As_Cache_Misses()
	measurements.Branch_Misses = Measurement_Compute(
		Values(extract(samples, sample_branch_misses)), "count").As_Branch_Misses()
	measurements.CPU_User = Measurement_Compute(
		Values(extract(samples, sample_user)), "nanoseconds").As_CPU_User()
	measurements.CPU_System = Measurement_Compute(
		Values(extract(samples, sample_system)), "nanoseconds").As_CPU_System()
	return measurements
}

// Reference_Measurements is the baseline distribution set deltas are measured against —
// a field-identical clone of Measurements, so the reference role is its own type and
// deltas_compute takes two distinct-typed parameters instead of an input struct of
// duplicates.
type Reference_Measurements struct {
	// Wall_Time is the elapsed-time distribution.
	Wall_Time Wall_Time_Measurement
	// Peak_RSS is the peak-memory distribution.
	Peak_RSS Peak_RSS_Measurement
	// CPU_Cycles is the CPU-cycle distribution.
	CPU_Cycles CPU_Cycles_Measurement
	// Instructions is the retired-instruction distribution.
	Instructions Instructions_Measurement
	// Cache_References is the cache-reference distribution (Linux only).
	Cache_References Cache_References_Measurement
	// Cache_Misses is the cache-miss distribution (Linux only).
	Cache_Misses Cache_Misses_Measurement
	// Branch_Misses is the branch-miss distribution (Linux only).
	Branch_Misses Branch_Misses_Measurement
	// CPU_User is the user-CPU-time distribution.
	CPU_User CPU_User_Measurement
	// CPU_System is the system-CPU-time distribution.
	CPU_System CPU_System_Measurement
}

// Reference_Measurements_Invariants composes every metric's family.
func Reference_Measurements_Invariants(identifier string, reference Reference_Measurements) {
	Wall_Time_Measurement_Invariants(identifier, reference.Wall_Time)
	Peak_RSS_Measurement_Invariants(identifier, reference.Peak_RSS)
	CPU_Cycles_Measurement_Invariants(identifier, reference.CPU_Cycles)
	Instructions_Measurement_Invariants(identifier, reference.Instructions)
	Cache_References_Measurement_Invariants(identifier, reference.Cache_References)
	Cache_Misses_Measurement_Invariants(identifier, reference.Cache_Misses)
	Branch_Misses_Measurement_Invariants(identifier, reference.Branch_Misses)
	CPU_User_Measurement_Invariants(identifier, reference.CPU_User)
	CPU_System_Measurement_Invariants(identifier, reference.CPU_System)
}

// Candidate_Measurements is the distribution set compared against the reference — the
// candidate role's own clone of Measurements.
type Candidate_Measurements struct {
	// Wall_Time is the elapsed-time distribution.
	Wall_Time Wall_Time_Measurement
	// Peak_RSS is the peak-memory distribution.
	Peak_RSS Peak_RSS_Measurement
	// CPU_Cycles is the CPU-cycle distribution.
	CPU_Cycles CPU_Cycles_Measurement
	// Instructions is the retired-instruction distribution.
	Instructions Instructions_Measurement
	// Cache_References is the cache-reference distribution (Linux only).
	Cache_References Cache_References_Measurement
	// Cache_Misses is the cache-miss distribution (Linux only).
	Cache_Misses Cache_Misses_Measurement
	// Branch_Misses is the branch-miss distribution (Linux only).
	Branch_Misses Branch_Misses_Measurement
	// CPU_User is the user-CPU-time distribution.
	CPU_User CPU_User_Measurement
	// CPU_System is the system-CPU-time distribution.
	CPU_System CPU_System_Measurement
}

// Candidate_Measurements_Invariants composes every metric's family.
func Candidate_Measurements_Invariants(identifier string, candidate Candidate_Measurements) {
	Wall_Time_Measurement_Invariants(identifier, candidate.Wall_Time)
	Peak_RSS_Measurement_Invariants(identifier, candidate.Peak_RSS)
	CPU_Cycles_Measurement_Invariants(identifier, candidate.CPU_Cycles)
	Instructions_Measurement_Invariants(identifier, candidate.Instructions)
	Cache_References_Measurement_Invariants(identifier, candidate.Cache_References)
	Cache_Misses_Measurement_Invariants(identifier, candidate.Cache_Misses)
	Branch_Misses_Measurement_Invariants(identifier, candidate.Branch_Misses)
	CPU_User_Measurement_Invariants(identifier, candidate.CPU_User)
	CPU_System_Measurement_Invariants(identifier, candidate.CPU_System)
}

// Deltas_compute compares every metric of the candidate against the reference. The
// per-metric variants convert to the plain compare currency and back, so the
// comparison math stays written once.
func deltas_compute(
	reference Reference_Measurements, candidate Candidate_Measurements,
) (deltas Deltas) {
	defer func() {
		Deltas_Invariants("deltas_compute.deltas", deltas)
	}()
	Reference_Measurements_Invariants("deltas_compute.reference", reference)
	Candidate_Measurements_Invariants("deltas_compute.candidate", candidate)
	deltas.Wall_Time = Compare(
		Reference_Measurement(reference.Wall_Time.As_Measurement()),
		Candidate_Measurement(candidate.Wall_Time.As_Measurement())).As_Wall_Time_Delta()
	deltas.Peak_RSS = Compare(
		Reference_Measurement(reference.Peak_RSS.As_Measurement()),
		Candidate_Measurement(candidate.Peak_RSS.As_Measurement())).As_Peak_RSS_Delta()
	deltas.CPU_Cycles = Compare(
		Reference_Measurement(reference.CPU_Cycles.As_Measurement()),
		Candidate_Measurement(candidate.CPU_Cycles.As_Measurement())).As_CPU_Cycles_Delta()
	deltas.Instructions = Compare(
		Reference_Measurement(reference.Instructions.As_Measurement()),
		Candidate_Measurement(
			candidate.Instructions.As_Measurement())).As_Instructions_Delta()
	deltas.Cache_References = Compare(
		Reference_Measurement(reference.Cache_References.As_Measurement()),
		Candidate_Measurement(
			candidate.Cache_References.As_Measurement())).As_Cache_References_Delta()
	deltas.Cache_Misses = Compare(
		Reference_Measurement(reference.Cache_Misses.As_Measurement()),
		Candidate_Measurement(
			candidate.Cache_Misses.As_Measurement())).As_Cache_Misses_Delta()
	deltas.Branch_Misses = Compare(
		Reference_Measurement(reference.Branch_Misses.As_Measurement()),
		Candidate_Measurement(
			candidate.Branch_Misses.As_Measurement())).As_Branch_Misses_Delta()
	deltas.CPU_User = Compare(
		Reference_Measurement(reference.CPU_User.As_Measurement()),
		Candidate_Measurement(candidate.CPU_User.As_Measurement())).As_CPU_User_Delta()
	deltas.CPU_System = Compare(
		Reference_Measurement(reference.CPU_System.As_Measurement()),
		Candidate_Measurement(candidate.CPU_System.As_Measurement())).As_CPU_System_Delta()
	return deltas
}

// Extract pulls one metric's value out of every sample as the int64 the statistics
// work in; every raw metric is already an integer count, so no float ever enters.
func extract[T ~int64](
	samples Distribution, selector func(sample Sample) (value T),
) (values Series) {
	defer func() { Series_Invariants("extract.values", values) }()
	Distribution_Invariants("extract.samples", samples)
	values = make([]int64, len(samples))
	for index, sample := range samples {
		values[index] = int64(selector(sample))
	}
	return values
}

// Sample_wall reads a sample's wall time, a non-negative monotonic span.
func sample_wall(sample Sample) (value Metric) {
	defer func() { Metric_Invariants("sample_wall.value", value) }()
	Sample_Invariants("sample_wall.sample", sample)
	return Metric(sample.Wall)
}

// Sample_rss reads a sample's peak resident size as a metric.
func sample_rss(sample Sample) (value Metric) {
	defer func() { Metric_Invariants("sample_rss.value", value) }()
	Sample_Invariants("sample_rss.sample", sample)
	return Metric(sample.RSS_Bytes_Max)
}

// Sample_cycles reads a sample's CPU cycle count as a metric.
func sample_cycles(sample Sample) (value Metric) {
	defer func() { Metric_Invariants("sample_cycles.value", value) }()
	Sample_Invariants("sample_cycles.sample", sample)
	return Metric(sample.CPU_Cycles)
}

// Sample_instructions reads a sample's retired-instruction count as a metric.
func sample_instructions(sample Sample) (value Metric) {
	defer func() { Metric_Invariants("sample_instructions.value", value) }()
	Sample_Invariants("sample_instructions.sample", sample)
	return Metric(sample.Instructions)
}

// Sample_cache_references reads a sample's cache-reference count as a metric.
func sample_cache_references(sample Sample) (value Metric) {
	defer func() { Metric_Invariants("sample_cache_references.value", value) }()
	Sample_Invariants("sample_cache_references.sample", sample)
	return Metric(sample.Cache_References)
}

// Sample_cache_misses reads a sample's cache-miss count as a metric.
func sample_cache_misses(sample Sample) (value Metric) {
	defer func() { Metric_Invariants("sample_cache_misses.value", value) }()
	Sample_Invariants("sample_cache_misses.sample", sample)
	return Metric(sample.Cache_Misses)
}

// Sample_branch_misses reads a sample's branch-miss count as a metric.
func sample_branch_misses(sample Sample) (value Metric) {
	defer func() { Metric_Invariants("sample_branch_misses.value", value) }()
	Sample_Invariants("sample_branch_misses.sample", sample)
	return Metric(sample.Branch_Misses)
}

// Sample_user reads a sample's user CPU time as a metric.
func sample_user(sample Sample) (value Metric) {
	defer func() { Metric_Invariants("sample_user.value", value) }()
	Sample_Invariants("sample_user.sample", sample)
	return Metric(sample.CPU_User)
}

// Sample_system reads a sample's system CPU time as a metric.
func sample_system(sample Sample) (value Metric) {
	defer func() { Metric_Invariants("sample_system.value", value) }()
	Sample_Invariants("sample_system.sample", sample)
	return Metric(sample.CPU_System)
}

// COMMAND_WORD_BYTES_MIN is the empty word: a blank argument the shell can still pass.
const COMMAND_WORD_BYTES_MIN = 0

// COMMAND_WORD_BYTES_MAX caps one argv word: a few kilobytes is already an unreasonable
// single token, and a longer one is a probe to reject rather than benchmark.
const COMMAND_WORD_BYTES_MAX = 1 << 12

// Command_Word is one word of a benchmarked command — an environment assignment, the
// executable, or an argument. It is what the user typed at the shell, bounded short.
type Command_Word string

// Command_Word_Invariants bounds a command word's length, witnessing the empty min, the
// capped max, and the one- and two-byte shapes between.
func Command_Word_Invariants(identifier string, word Command_Word) {
	invariant.Range(identifier, len(word), COMMAND_WORD_BYTES_MIN, COMMAND_WORD_BYTES_MAX)
	invariant.Sometimes(identifier, len(word) == 1, "The command word is one byte.")
	invariant.Sometimes(identifier, len(word) == 2, "The command word is two bytes.")
	invariant.Sometimes(identifier, 2 < len(word) && len(word) < COMMAND_WORD_BYTES_MAX,
		"The command word is an ordinary length.")
}

// Command_words flattens a command back to the words a reader recognizes:
// environment assignments, the executable, then its arguments.
func command_words(command sysio.Process_Request) (words Command_Line) {
	defer func() {
		Command_Line_Invariants("command_words.words", words)
		for _, x := range words {
			Command_Word_Invariants("command_words.word", x)
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
func write_report(output io.Writer, document Document) (code Exit_Code) {
	defer func() { Exit_Code_Invariants("write_report.exit_code", code) }()
	Document_Invariants("write_report.document", document)
	payload, marshal_err := json.MarshalIndent(document, "", "  ")
	if marshal_err != nil {
		return EXIT_FAILURE
	}
	payload = append(payload, '\n')
	_, write_err := output.Write(payload)
	if write_err != nil {
		return EXIT_FAILURE
	}
	return EXIT_SUCCESS
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
func Write_Failure_Input_Invariants(identifier string, input Write_Failure_Input) {
	Position_Invariants(identifier, input.Index)
	Failure_Status_Invariants(identifier, input.Exit)
	Captured_Output_Invariants(identifier, input.Child_Stderr)
}

// Write_failure reports a benchmarked command's non-zero exit to the diagnostic
// sink: a one-line header naming the command's position and code, then the
// command's own stderr.
func write_failure(input *Write_Failure_Input) {
	Write_Failure_Input_Invariants("write_failure.input", *input)
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
		Measurement_Invariants("Measurement_Compute.measurement", measurement)
	}()
	Values_Invariants("Measurement_Compute.values", values)
	for _, x := range values {
		Metric_Invariants("Measurement_Compute.value", Metric(x))
	}
	Unit_Invariants("Measurement_Compute.unit", unit)
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
	q1 := fixedpoint.From_Integer(low_quartile)
	q3 := fixedpoint.From_Integer(high_quartile)
	mean := fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
		Numerator: total, Denominator: int64(count),
	})
	deviation := standard_deviation(sorted, Average(mean_integer), Kept(count))

	measurement = Measurement{
		Mean:               Mean(mean),
		Standard_Deviation: Standard_Deviation(deviation),
		Min:                Min(fixedpoint.From_Integer(sorted[0])),
		Max:                Max(fixedpoint.From_Integer(sorted[count-1])),
		Median:             Median(fixedpoint.From_Integer(sorted[count/2])),
		Q1:                 Q1(q1),
		Q3:                 Q3(q3),
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
func Wide_Invariants(identifier string, value Wide) {
	Accumulator_High_Invariants(identifier, value.High)
	Accumulator_Low_Invariants(identifier, value.Low)
}

// Wide_add_square adds value squared into a 128-bit accumulator.
func wide_add_square(subtotal Wide, value Gap) (sum Wide) {
	defer func() { Wide_Invariants("wide_add_square.sum", sum) }()
	Wide_Invariants("wide_add_square.accumulator", subtotal)
	Gap_Invariants("wide_add_square.value", value)
	product_high, product_low := bits.Mul64(uint64(value), uint64(value))
	low, carry := bits.Add64(uint64(subtotal.Low), product_low, 0)
	high, _ := bits.Add64(uint64(subtotal.High), product_high, carry)
	return Wide{High: Accumulator_High(high), Low: Accumulator_Low(low)}
}

// Standard_deviation is the sample standard deviation with an n-1 denominator. The sum
// of squared deviations is accumulated in 128 bits so a large metric's deviations cannot
// overflow before the divide and the fixed-point root.
func standard_deviation(
	sorted Deviations, center Average, count Kept,
) (deviation fixedpoint.Number) {
	Deviations_Invariants("standard_deviation.sorted", sorted)
	Average_Invariants("standard_deviation.mean", center)
	Kept_Invariants("standard_deviation.count", count)
	if count <= 1 {
		return 0
	}
	sum := Wide{}
	for _, value := range sorted {
		distance := value - int64(center)
		if distance < 0 {
			distance = -distance
		}
		sum = wide_add_square(sum, Gap(uint64(distance)))
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
func Root_Of_Quotient_Input_Invariants(identifier string, input Root_Of_Quotient_Input) {
	Accumulator_High_Invariants(identifier, input.High)
	Accumulator_Low_Invariants(identifier, input.Low)
	Divisor_Invariants(identifier, input.Denominator)
}

// Root_of_quotient returns the fixed-point square root of a 128-bit numerator over a
// denominator — the shared tail of the sample and pooled deviations. A quotient that fits
// a signed word keeps full fractional precision; a larger one, a multi-second jitter far
// outside maddox's fast-command envelope, falls back to the integer root.
func root_of_quotient(input *Root_Of_Quotient_Input) (deviation fixedpoint.Number) {
	Root_Of_Quotient_Input_Invariants("root_of_quotient.input", *input)
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
		return fixedpoint.Square_Root_Scaled(int64(quotient_low))
	}
	return fixedpoint.From_Integer(fixedpoint.Integer_Root(&fixedpoint.Integer_Root_Input{
		High: quotient_high, Low: quotient_low,
	}))
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

// Outlier_Count_Input_Invariants composes the sorted values and the two quartiles.
func Outlier_Count_Input_Invariants(identifier string, input Outlier_Count_Input) {
	Points_Invariants(identifier, input.Sorted)
	Low_Quartile_Invariants(identifier, input.Low_Quartile)
	High_Quartile_Invariants(identifier, input.High_Quartile)
}

// Low_Quartile is the raw metric at q1, the lower fence's anchor.
type Low_Quartile int64

// Low_Quartile_Invariants bounds the anchor to the metric domain and witnesses the old
// per-field grid's shapes with its own texts.
func Low_Quartile_Invariants(identifier string, value Low_Quartile) {
	invariant.Always(value >= METRIC_MIN, "A low quartile is never negative.")
	invariant.Always(value <= METRIC_MAX,
		"A low quartile stays within the representable metric ceiling.")
	invariant.Sometimes(identifier, value == METRIC_MIN, "The low quartile is zero.")
	invariant.Sometimes(identifier, value == 1, "The low quartile is one.")
	invariant.Sometimes(identifier, value == 2, "The low quartile is two.")
	invariant.Sometimes(identifier, value == METRIC_MAX,
		"The low quartile fills the metric ceiling.")
	invariant.Sometimes(identifier, 2 < value && value < METRIC_MAX,
		"The low quartile is an ordinary magnitude.")
}

// High_Quartile is the raw metric at q3, the upper fence's anchor.
type High_Quartile int64

// High_Quartile_Invariants bounds the anchor to the metric domain and witnesses the old
// per-field grid's shapes with its own texts.
func High_Quartile_Invariants(identifier string, value High_Quartile) {
	invariant.Always(value >= METRIC_MIN, "A high quartile is never negative.")
	invariant.Always(value <= METRIC_MAX,
		"A high quartile stays within the representable metric ceiling.")
	invariant.Sometimes(identifier, value == METRIC_MIN, "The high quartile is zero.")
	invariant.Sometimes(identifier, value == 1, "The high quartile is one.")
	invariant.Sometimes(identifier, value == 2, "The high quartile is two.")
	invariant.Sometimes(identifier, value == METRIC_MAX,
		"The high quartile fills the metric ceiling.")
	invariant.Sometimes(identifier, 2 < value && value < METRIC_MAX,
		"The high quartile is an ordinary magnitude.")
}

// Outlier_count counts the values beyond Tukey's fences — a point more than 1.5 interquartile
// ranges past a quartile. The whole test runs in the raw metric domain, doubled so the 3/2 is
// an integer 3 over a factored-out 2: a value is an outlier when 2*value falls outside
// [2*q1 - 3*IQR, 2*q3 + 3*IQR]. Every term stays within int64 for metrics up to the
// representable ceiling — 2*value and 2*q are at most 2^44, 3*IQR at most ~2^45 — where the
// fixed-point form, each quartile first lifted by the 2^20 scale to near 2^63, overflowed on a
// wide spread and produced garbage fences that miscounted the whole run as outliers.
func outlier_count(input *Outlier_Count_Input) (count Strays) {
	defer func() { Strays_Invariants("outlier_count.count", count) }()
	Outlier_Count_Input_Invariants("outlier_count.input", *input)
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

// Reference_Measurement is the baseline measurement deltas are taken against — a
// field-identical clone of Measurement, so the reference role is its own type and the
// comparison functions take two distinct-typed parameters instead of an input struct
// of duplicates.
type Reference_Measurement struct {
	// Mean is the arithmetic mean of the metric's values.
	Mean Mean
	// Standard_Deviation is the sample standard deviation, with an n-1 denominator.
	Standard_Deviation Standard_Deviation
	// Min is the smallest value observed.
	Min Min
	// Max is the largest value observed.
	Max Max
	// Median is the middle value of the sorted values.
	Median Median
	// Q1 is the first quartile by poop's index math.
	Q1 Q1
	// Q3 is the third quartile by poop's index math.
	Q3 Q3
	// Outlier_Count is how many values fall beyond Tukey's fences.
	Outlier_Count Strays
	// Sample_Count is how many values the distribution was computed from.
	Sample_Count Kept
	// Unit names the unit the raw values are in.
	Unit Unit
}

// Reference_Measurement_Invariants composes every field's own family.
func Reference_Measurement_Invariants(identifier string, reference Reference_Measurement) {
	Mean_Invariants(identifier, reference.Mean)
	Standard_Deviation_Invariants(identifier, reference.Standard_Deviation)
	Min_Invariants(identifier, reference.Min)
	Max_Invariants(identifier, reference.Max)
	Median_Invariants(identifier, reference.Median)
	Q1_Invariants(identifier, reference.Q1)
	Q3_Invariants(identifier, reference.Q3)
	Strays_Invariants(identifier, reference.Outlier_Count)
	Kept_Invariants(identifier, reference.Sample_Count)
	Unit_Invariants(identifier, reference.Unit)
}

// Candidate_Measurement is the measurement compared to the reference — the candidate
// role's own clone of Measurement.
type Candidate_Measurement struct {
	// Mean is the arithmetic mean of the metric's values.
	Mean Mean
	// Standard_Deviation is the sample standard deviation, with an n-1 denominator.
	Standard_Deviation Standard_Deviation
	// Min is the smallest value observed.
	Min Min
	// Max is the largest value observed.
	Max Max
	// Median is the middle value of the sorted values.
	Median Median
	// Q1 is the first quartile by poop's index math.
	Q1 Q1
	// Q3 is the third quartile by poop's index math.
	Q3 Q3
	// Outlier_Count is how many values fall beyond Tukey's fences.
	Outlier_Count Strays
	// Sample_Count is how many values the distribution was computed from.
	Sample_Count Kept
	// Unit names the unit the raw values are in.
	Unit Unit
}

// Candidate_Measurement_Invariants composes every field's own family.
func Candidate_Measurement_Invariants(identifier string, candidate Candidate_Measurement) {
	Mean_Invariants(identifier, candidate.Mean)
	Standard_Deviation_Invariants(identifier, candidate.Standard_Deviation)
	Min_Invariants(identifier, candidate.Min)
	Max_Invariants(identifier, candidate.Max)
	Median_Invariants(identifier, candidate.Median)
	Q1_Invariants(identifier, candidate.Q1)
	Q3_Invariants(identifier, candidate.Q3)
	Strays_Invariants(identifier, candidate.Outlier_Count)
	Kept_Invariants(identifier, candidate.Sample_Count)
	Unit_Invariants(identifier, candidate.Unit)
}

// Compare reports how the candidate's mean differs from the reference's, with the
// 95% confidence half-interval from a pooled-variance two-sample t-test — poop's
// ratio computation. A zero or degenerate reference yields a zero, non-significant
// delta rather than a divide-by-zero.
func Compare(reference Reference_Measurement, candidate Candidate_Measurement) (delta Delta) {
	defer func() { Delta_Invariants("Compare.delta", delta) }()
	Reference_Measurement_Invariants("Compare.reference", reference)
	Candidate_Measurement_Invariants("Compare.candidate", candidate)
	if reference.Mean == 0 {
		return delta
	}
	ratio := fixedpoint.Divide(&fixedpoint.Divide_Input{
		Dividend: fixedpoint.Number(candidate.Mean - reference.Mean),
		Divisor:  fixedpoint.Number(reference.Mean),
	})
	delta.Diff_Percent = Diff_Percent(ratio * 100)
	delta.Faster = Faster(candidate.Mean < reference.Mean)

	degrees := candidate.Sample_Count + reference.Sample_Count - 2
	if degrees < 1 {
		return delta
	}
	delta.Half_Percent = Half_Percent(half_interval(reference, candidate, Degree(degrees)))
	delta.Significant = significant(delta.Diff_Percent, delta.Half_Percent)
	return delta
}

// Half_interval is the 95% confidence half-width on Diff_Percent, from a pooled-variance
// two-sample t-test — poop's score*pooled*normalizer*100/mean. The pooled deviation is
// taken relative to the reference mean, folding in that final divide, so the math never
// forms a raw variance — which, for a metric in the billions, overflows.
func half_interval(
	reference Reference_Measurement, candidate Candidate_Measurement, degrees Degree,
) (half fixedpoint.Number) {
	Reference_Measurement_Invariants("half_interval.reference", reference)
	Candidate_Measurement_Invariants("half_interval.candidate", candidate)
	Degree_Invariants("half_interval.degrees", degrees)
	first := fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
		Numerator: 1, Denominator: int64(candidate.Sample_Count),
	})
	second := fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
		Numerator: 1, Denominator: int64(reference.Sample_Count),
	})
	normalizer := fixedpoint.Square_Root(first + second)
	pooled := pooled_deviation(candidate, reference, degrees)
	score := student_t_score(degrees)
	band := fixedpoint.Multiply(&fixedpoint.Multiply_Input{A: score, B: pooled})
	band = fixedpoint.Multiply(&fixedpoint.Multiply_Input{A: band, B: normalizer})
	return band * 100
}

// Pooled_deviation is the pooled standard deviation as a fraction of the reference mean:
// the root of the degrees-weighted mean of the two relative variances. Dividing each
// deviation by the mean before squaring keeps every value near one, so a metric in the
// billions and its enormous raw variance never overflow.
func pooled_deviation(
	candidate Candidate_Measurement, reference Reference_Measurement, degrees Degree,
) (pooled fixedpoint.Number) {
	Candidate_Measurement_Invariants("pooled_deviation.candidate", candidate)
	Reference_Measurement_Invariants("pooled_deviation.reference", reference)
	Degree_Invariants("pooled_deviation.degrees", degrees)
	mean := fixedpoint.Number(reference.Mean)
	relative_candidate := fixedpoint.Divide(&fixedpoint.Divide_Input{
		Dividend: fixedpoint.Number(candidate.Standard_Deviation), Divisor: mean,
	})
	relative_reference := fixedpoint.Divide(&fixedpoint.Divide_Input{
		Dividend: fixedpoint.Number(reference.Standard_Deviation), Divisor: mean,
	})
	candidate_variance := fixedpoint.Multiply(&fixedpoint.Multiply_Input{
		A: relative_candidate, B: relative_candidate,
	})
	reference_variance := fixedpoint.Multiply(&fixedpoint.Multiply_Input{
		A: relative_reference, B: relative_reference,
	})
	weighted := candidate_variance*fixedpoint.Number(candidate.Sample_Count-1) +
		reference_variance*fixedpoint.Number(reference.Sample_Count-1)
	return fixedpoint.Square_Root(weighted / fixedpoint.Number(int(degrees)))
}

// Student_t_score returns the Student-t critical value for 95% confidence at the given
// degrees of freedom as a fixed-point number, falling back to the normal-distribution
// 1.96 past the tabulated range — poop's getStatScore95. The tables hold thousandths so
// they read as the published constants, and From_Ratio puts them on the fixed-point grid.
// The tables are local, not package globals, so the package keeps no mutable state.
func student_t_score(degrees_of_freedom Degree) (score fixedpoint.Number) {
	Degree_Invariants("student_t_score.degrees_of_freedom", degrees_of_freedom)
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
	return fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
		Numerator: milli, Denominator: 1000,
	})
}

// Significant decides whether a difference clears poop's ±1% band: the whole
// confidence interval must sit beyond ±1% with a single sign. The && and || of
// poop's check are split into nested single-term ifs to satisfy the linter.
func significant(difference Diff_Percent, half Half_Percent) (is Significant) {
	defer func() { Significant_Invariants("significant.is", is) }()
	Diff_Percent_Invariants("significant.difference", difference)
	Half_Percent_Invariants("significant.half", half)
	change := fixedpoint.Number(difference)
	interval := fixedpoint.Number(half)
	if change >= fixedpoint.From_Integer(1) {
		if change-interval >= fixedpoint.From_Integer(1) {
			return true
		}
	}
	if change <= fixedpoint.From_Integer(-1) {
		if change+interval <= fixedpoint.From_Integer(-1) {
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

// Ansi_Code_Invariants bounds an ansi_code's byte length and claims its boundaries.
// An SGR code is always four or five bytes, so the empty and one/two-byte boundaries
// are claimed as never reached rather than as observed.
func Ansi_Code_Invariants(identifier string, code Ansi_Code) {
	invariant.Range(identifier, len(code), ANSI_CODE_BYTES_MIN, ANSI_CODE_BYTES_MAX)
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

// Render_Table_Input_Invariants composes the report and the color flag.
func Render_Table_Input_Invariants(identifier string, input Render_Table_Input) {
	Document_Invariants(identifier, input.Document)
	Color_Invariants(identifier, input.Color)
}

// Render_Table renders the report as poop's aligned, optionally colored comparison
// table — the default human-readable output. The machine specs block is written
// first, before Benchmark 1.
func Render_Table(input *Render_Table_Input) (output Report) {
	// The rendered bytes are text — printable ASCII, newlines, and ANSI escapes — so a
	// per-byte numeric bundle would demand the null, one, and all-ones bytes a text report
	// never carries. The report's shape is witnessed by its length, not its every byte.
	defer func() { Report_Invariants("Render_Table.output", output) }()
	Render_Table_Input_Invariants("Render_Table.input", *input)
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
	Machine_Specs_Invariants("render_machine_header.m", m)
	// An empty CPU model with no architecture means specs were never acquired (e.g.
	// the stub platform); the header is skipped so output stays clean.
	if m.CPU_Model == "" {
		if m.CPU_Arch == "" {
			return
		}
	}
	builder.WriteString("Machine: " + string(m.CPU_Model) + " (" + string(m.CPU_Arch) + ")\n")
	builder.WriteString("  cores: " + string(machine_specs_cores(m)) + "   " +
		"freq: " + string(format_hz(m.CPU_Frequency_Hz_Max)) + "   " +
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
func Cores_Line_Invariants(identifier string, text Cores_Line) {
	invariant.Range(identifier, len(text), CORES_MIN, CORES_MAX)
	invariant.Sometimes(identifier, len(text) == 2, "The core layout is two bytes.")
	invariant.Sometimes(identifier, 2 < len(text) && len(text) < CORES_MAX,
		"The core layout is an ordinary length.")
}

// Machine_specs_cores renders the CPU core layout, naming performance and efficiency cores on
// hybrid CPUs and collapsing to a single count when physical equals logical.
func machine_specs_cores(m Machine_Specs) (text Cores_Line) {
	defer func() { Cores_Line_Invariants("machine_specs_cores.text", text) }()
	Machine_Specs_Invariants("machine_specs_cores.m", m)
	if m.Performance_Cores > 0 {
		if m.Efficiency_Cores > 0 {
			return Cores_Line(fmt.Sprintf("%d P + %d E = %d logical",
				m.Performance_Cores, m.Efficiency_Cores, m.Logical_Cores))
		}
	}
	if int(m.Physical_Cores) == int(m.Logical_Cores) {
		return Cores_Line(fmt.Sprintf("%d", m.Physical_Cores))
	}
	return Cores_Line(fmt.Sprintf("%d physical, %d logical", m.Physical_Cores, m.Logical_Cores))
}

// Format_hz renders a frequency in Hz to a human-readable GHz or MHz string.
func format_hz(hz Hertz) (text Frequency) {
	defer func() { Frequency_Invariants("format_hz.text", text) }()
	Hertz_Invariants("format_hz.hz", hz)
	if hz == 0 {
		return "?"
	}
	if hz >= 1_000_000_000 {
		gigahertz := fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
			Numerator: int64(hz), Denominator: 1_000_000_000,
		})
		return Frequency(fixedpoint.Format(gigahertz, 2) + " GHz")
	}
	megahertz := fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
		Numerator: int64(hz), Denominator: 1_000_000,
	})
	return Frequency(fixedpoint.Format(megahertz, 0) + " MHz")
}

// Format_bytes renders a byte count, drawn from the machine specs as an unsigned integer, with
// the binary suffix ladder. The raw count is scaled down its rung and only then lifted into
// fixed-point, dividing through a 128-bit ratio: lifting the whole value first would overflow
// the 2^20 scale for a multi-petabyte size — the byte-size ceiling reaches 2^53 — and render a
// garbage, over-width cell. Each rung's raw divisor is recovered from its fixed-point form; the
// suffix is that rung's own, already witnessed by byte_ladder's invariants.
func format_bytes(value Byte_Size) (text Cell) {
	defer func() { Cell_Invariants("format_bytes.text", text) }()
	Byte_Size_Invariants("format_bytes.value", value)
	raw := int64(value)
	for _, step := range byte_ladder() {
		rung := fixedpoint.Whole(step.Divisor)
		if raw >= rung {
			scaled := fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
				Numerator: raw, Denominator: rung,
			})
			return Cell(string(format_significant(scaled)) + string(step.Suffix))
		}
	}
	return Cell(string(format_significant(fixedpoint.From_Integer(raw))))
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
func Span_Invariants(identifier string, text Span) {
	invariant.Range(identifier, len(text), SPAN_BYTES_MIN, SPAN_BYTES_MAX)
	invariant.Sometimes(identifier, len(text) == 2, "The rendered span is two bytes.")
	invariant.Sometimes(identifier, 2 < len(text) && len(text) < SPAN_BYTES_MAX,
		"The rendered span is an ordinary length.")
}

// Format_elapsed renders a total sampling duration for the benchmark header with the time
// ladder. Like format_bytes it scales the raw nanoseconds down their rung before lifting into
// fixed-point, so a duration past the 2^20 scale's ceiling — a sum of many large run walls —
// reduces without overflow; and it caps the magnitude first, so an absurd multi-year span still
// lands within the header's glyph rather than rendering a five-digit count. The suffix is the
// rung's own, witnessed by time_ladder.
func format_elapsed(elapsed time.Duration) (text Span) {
	defer func() { Span_Invariants("format_elapsed.text", text) }()
	raw := int64(elapsed)
	if raw > ELAPSED_DISPLAY_MAX {
		raw = ELAPSED_DISPLAY_MAX
	}
	for _, step := range time_ladder() {
		rung := fixedpoint.Whole(step.Divisor)
		if raw >= rung {
			scaled := fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
				Numerator: raw, Denominator: rung,
			})
			return Span(string(format_significant(scaled)) + string(step.Suffix))
		}
	}
	return Span(string(format_significant(fixedpoint.From_Integer(raw))))
}

// Write_table renders the table and writes it to output, returning EXIT_FAILURE if
// the write fails.
func write_table(output io.Writer, input *Render_Table_Input) (code Exit_Code) {
	defer func() { Exit_Code_Invariants("write_table.exit_code", code) }()
	Render_Table_Input_Invariants("write_table.input", *input)
	_, write_err := output.Write(Render_Table(input))
	if write_err != nil {
		return EXIT_FAILURE
	}
	return EXIT_SUCCESS
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
	Position_Invariants("render_benchmark.index", index)
	Benchmark_Invariants("render_benchmark.benchmark", benchmark)
	Color_Invariants("render_benchmark.color", color)
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
	Position_Invariants("render_metric_rows.index", index)
	Benchmark_Invariants("render_metric_rows.benchmark", benchmark)
	Color_Invariants("render_metric_rows.color", color)
	m := benchmark.Measurements
	d := benchmark.Deltas
	// The reference is first; only later benchmarks carry a comparison.
	has_delta := Has_Delta(index != 0)
	rows := []Metric_Line_Input{
		{
			Name:        "wall_time",
			Measurement: m.Wall_Time.As_Measurement(),
			Delta:       d.Wall_Time.As_Delta(),
		},
		{
			Name:        "peak_rss",
			Measurement: m.Peak_RSS.As_Measurement(),
			Delta:       d.Peak_RSS.As_Delta(),
		},
		{
			Name:        "cpu_cycles",
			Measurement: m.CPU_Cycles.As_Measurement(),
			Delta:       d.CPU_Cycles.As_Delta(),
		},
		{
			Name:        "instructions",
			Measurement: m.Instructions.As_Measurement(),
			Delta:       d.Instructions.As_Delta(),
		},
		{
			Name:        "cache_references",
			Measurement: m.Cache_References.As_Measurement(),
			Delta:       d.Cache_References.As_Delta(),
		},
		{
			Name:        "cache_misses",
			Measurement: m.Cache_Misses.As_Measurement(),
			Delta:       d.Cache_Misses.As_Delta(),
		},
		{
			Name:        "branch_misses",
			Measurement: m.Branch_Misses.As_Measurement(),
			Delta:       d.Branch_Misses.As_Delta(),
		},
		{
			Name:        "cpu_user",
			Measurement: m.CPU_User.As_Measurement(),
			Delta:       d.CPU_User.As_Delta(),
		},
		{
			Name:        "cpu_system",
			Measurement: m.CPU_System.As_Measurement(),
			Delta:       d.CPU_System.As_Delta(),
		},
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

// Metric_Line_Input carries everything one metric row is rendered from.
type Metric_Line_Input struct {
	// Name is the metric's row label.
	Name Caption
	// Measurement is the metric's distribution summary.
	Measurement Measurement
	// Delta is the signed change against the reference.
	Delta Delta
	// Has_Delta is false for the reference row, which carries no delta.
	Has_Delta Has_Delta
	// Color is whether the delta column is ANSI-colored.
	Color Color
}

// Metric_Line_Input_Invariants composes the row's name, distribution, delta, and flags.
func Metric_Line_Input_Invariants(identifier string, input Metric_Line_Input) {
	Caption_Invariants(identifier, input.Name)
	Measurement_Invariants(identifier, input.Measurement)
	Delta_Invariants(identifier, input.Delta)
	Has_Delta_Invariants(identifier, input.Has_Delta)
	Color_Invariants(identifier, input.Color)
}

// Has_Delta is whether a metric row carries a delta column; the reference row does not.
type Has_Delta bool

// Has_Delta_Invariants witnesses both row shapes, carrying the demand the old
// Metric_Line_Input.Has_Delta grid stated.
func Has_Delta_Invariants(identifier string, value Has_Delta) {
	invariant.Sometimes(identifier, bool(value), "The row carries a delta column.")
}

// Metric_line renders one metric row: its name, scaled mean ± σ, min … max, outlier
// count, and — when the benchmark is not the reference — its delta.
func metric_line(input *Metric_Line_Input) (text Full_Row) {
	defer func() { Full_Row_Invariants("metric_line.line", text) }()
	Metric_Line_Input_Invariants("metric_line.input", *input)
	measurement := input.Measurement
	unit := measurement.Unit
	outlier_percent := fixedpoint.Number(0)
	if measurement.Sample_Count > 0 {
		outlier_percent = fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
			Numerator:   int64(measurement.Outlier_Count) * 100,
			Denominator: int64(measurement.Sample_Count),
		})
	}
	outlier_text := strconv.Itoa(int(measurement.Outlier_Count)) +
		" (" + fixedpoint.Format(outlier_percent, 0) + "%)"
	row := render_cells(&Render_Cells_Input{
		Name: input.Name,
		Mean: Mean_Cell(format_quantity(fixedpoint.Number(measurement.Mean), unit)),
		Sigma: Sigma_Cell(format_quantity(
			fixedpoint.Number(measurement.Standard_Deviation), unit)),
		Low:      Low_Cell(format_quantity(fixedpoint.Number(measurement.Min), unit)),
		High:     High_Cell(format_quantity(fixedpoint.Number(measurement.Max), unit)),
		Outliers: Outliers(outlier_text),
	})
	text = Full_Row(row)
	if input.Has_Delta {
		delta := delta_render(input.Delta, input.Color)
		text = Full_Row(string(text) + "  " + string(delta))
	}
	return text
}

// Render_Cells_Input is one table row's cell texts. Each cell column is its own
// semantic type so the composed row seeds a distinct family per column, as the old
// per-field grid instances did.
type Render_Cells_Input struct {
	// Name is the row label.
	Name Caption
	// Mean is the mean cell.
	Mean Mean_Cell
	// Sigma is the standard-deviation cell.
	Sigma Sigma_Cell
	// Low is the minimum cell.
	Low Low_Cell
	// High is the maximum cell.
	High High_Cell
	// Outliers is the outlier-count cell.
	Outliers Outliers
}

// Render_Cells_Input_Invariants composes every cell text of one row.
func Render_Cells_Input_Invariants(identifier string, input Render_Cells_Input) {
	Caption_Invariants(identifier, input.Name)
	Mean_Cell_Invariants(identifier, input.Mean)
	Sigma_Cell_Invariants(identifier, input.Sigma)
	Low_Cell_Invariants(identifier, input.Low)
	High_Cell_Invariants(identifier, input.High)
	Outliers_Invariants(identifier, input.Outliers)
}

// Mean_Cell is the mean column's rendered cell.
type Mean_Cell string

// Mean_Cell_Invariants bounds the cell and witnesses the old per-field grid's shapes
// with its own texts.
func Mean_Cell_Invariants(identifier string, text Mean_Cell) {
	invariant.Always(len(text) >= CELL_BYTES_MIN, "A mean cell is never empty.")
	invariant.Always(len(text) <= CELL_BYTES_MAX,
		"A mean cell never exceeds the widest scaled value.")
	invariant.Sometimes(identifier, len(text) == CELL_BYTES_MIN,
		"The mean cell is one byte.")
	invariant.Sometimes(identifier, len(text) == 2, "The mean cell is two bytes.")
	invariant.Sometimes(identifier, len(text) == CELL_BYTES_MAX,
		"The mean cell is the widest form.")
	invariant.Sometimes(identifier, 2 < len(text) && len(text) < CELL_BYTES_MAX,
		"The mean cell is an ordinary length.")
}

// Sigma_Cell is the standard-deviation column's rendered cell.
type Sigma_Cell string

// Sigma_Cell_Invariants bounds the cell and witnesses the old per-field grid's shapes
// with its own texts.
func Sigma_Cell_Invariants(identifier string, text Sigma_Cell) {
	invariant.Always(len(text) >= CELL_BYTES_MIN, "A sigma cell is never empty.")
	invariant.Always(len(text) <= CELL_BYTES_MAX,
		"A sigma cell never exceeds the widest scaled value.")
	invariant.Sometimes(identifier, len(text) == CELL_BYTES_MIN,
		"The sigma cell is one byte.")
	invariant.Sometimes(identifier, len(text) == 2, "The sigma cell is two bytes.")
	invariant.Sometimes(identifier, len(text) == CELL_BYTES_MAX,
		"The sigma cell is the widest form.")
	invariant.Sometimes(identifier, 2 < len(text) && len(text) < CELL_BYTES_MAX,
		"The sigma cell is an ordinary length.")
}

// Low_Cell is the minimum column's rendered cell.
type Low_Cell string

// Low_Cell_Invariants bounds the cell and witnesses the old per-field grid's shapes
// with its own texts.
func Low_Cell_Invariants(identifier string, text Low_Cell) {
	invariant.Always(len(text) >= CELL_BYTES_MIN, "A low cell is never empty.")
	invariant.Always(len(text) <= CELL_BYTES_MAX,
		"A low cell never exceeds the widest scaled value.")
	invariant.Sometimes(identifier, len(text) == CELL_BYTES_MIN,
		"The low cell is one byte.")
	invariant.Sometimes(identifier, len(text) == 2, "The low cell is two bytes.")
	invariant.Sometimes(identifier, len(text) == CELL_BYTES_MAX,
		"The low cell is the widest form.")
	invariant.Sometimes(identifier, 2 < len(text) && len(text) < CELL_BYTES_MAX,
		"The low cell is an ordinary length.")
}

// High_Cell is the maximum column's rendered cell.
type High_Cell string

// High_Cell_Invariants bounds the cell and witnesses the old per-field grid's shapes
// with its own texts.
func High_Cell_Invariants(identifier string, text High_Cell) {
	invariant.Always(len(text) >= CELL_BYTES_MIN, "A high cell is never empty.")
	invariant.Always(len(text) <= CELL_BYTES_MAX,
		"A high cell never exceeds the widest scaled value.")
	invariant.Sometimes(identifier, len(text) == CELL_BYTES_MIN,
		"The high cell is one byte.")
	invariant.Sometimes(identifier, len(text) == 2, "The high cell is two bytes.")
	invariant.Sometimes(identifier, len(text) == CELL_BYTES_MAX,
		"The high cell is the widest form.")
	invariant.Sometimes(identifier, 2 < len(text) && len(text) < CELL_BYTES_MAX,
		"The high cell is an ordinary length.")
}

// Render_cells lays one row — header or data — into aligned, uncolored columns.
// Header and data share this layout, so a label always sits above its column.
func render_cells(input *Render_Cells_Input) (text Bare_Row) {
	defer func() { Bare_Row_Invariants("render_cells.line", text) }()
	Render_Cells_Input_Invariants("render_cells.input", *input)
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
func Cell_Invariants(identifier string, text Cell) {
	invariant.Range(identifier, len(text), CELL_BYTES_MIN, CELL_BYTES_MAX)
	invariant.Sometimes(identifier, len(text) == 2, "The cell is two bytes.")
	invariant.Sometimes(identifier, 2 < len(text) && len(text) < CELL_BYTES_MAX,
		"The cell is an ordinary length.")
}

// GLYPH_MIN is the single digit a formatted figure floors at, like "0".
const GLYPH_MIN = 1

// GLYPH_MAX bounds the bare significant figures before a suffix: a sign over three figures
// with a decimal point.
const GLYPH_MAX = 4

// Glyph is the bare significant-figure text a scale produces before its unit suffix — "0",
// "9.99". A distinct type, narrower than a whole cell, since the suffix is appended after.
type Glyph string

// Glyph_Invariants bounds the figure length; never empty, with the single-digit min, the
// widest figure max, and the one- and two-byte shapes between witnessed.
func Glyph_Invariants(identifier string, text Glyph) {
	invariant.Range(identifier, len(text), GLYPH_MIN, GLYPH_MAX)
	invariant.Sometimes(identifier, len(text) == 2, "The figure is two bytes.")
	invariant.Sometimes(identifier, 2 < len(text) && len(text) < GLYPH_MAX,
		"The figure is three bytes.")
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
func Column_Invariants(identifier string, text Column) {
	invariant.Range(identifier, len(text), COLUMN_MIN, COLUMN_MAX)
	invariant.Sometimes(identifier, len(text) == 2, "The column text is two bytes.")
	invariant.Sometimes(identifier, 2 < len(text) && len(text) < COLUMN_MAX,
		"The column text is an ordinary length.")
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
func Caption_Invariants(identifier string, text Caption) {
	invariant.Always(len(text) >= CAPTION_MIN, "A caption is never below its narrow band.")
	invariant.Always(len(text) <= CAPTION_MAX, "A caption never exceeds its widest name.")
	invariant.Sometimes(identifier, len(text) == CAPTION_MIN,
		"The caption is the shortest name.")
	invariant.Sometimes(identifier, len(text) == CAPTION_MAX,
		"The caption is the widest name.")
	invariant.Sometimes(identifier, CAPTION_MIN < len(text) && len(text) < CAPTION_MAX,
		"The caption is an ordinary name.")
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
func Outliers_Invariants(identifier string, text Outliers) {
	invariant.Always(len(text) >= OUTLIERS_MIN,
		"An outlier tally is never below its shortest form.")
	invariant.Always(len(text) <= OUTLIERS_MAX,
		"An outlier tally never exceeds its widest form.")
	invariant.Sometimes(identifier, len(text) == OUTLIERS_MIN,
		"The outlier tally is the shortest form.")
	invariant.Sometimes(identifier, len(text) == OUTLIERS_MAX,
		"The outlier tally is the widest form.")
	invariant.Sometimes(identifier, OUTLIERS_MIN < len(text) && len(text) < OUTLIERS_MAX,
		"The outlier tally is an ordinary form.")
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
func Padded_Invariants(identifier string, text Padded) {
	invariant.Range(identifier, len(text), PADDED_MIN, PADDED_MAX)
	invariant.Sometimes(identifier, PADDED_MIN < len(text) && len(text) < PADDED_MAX,
		"The padded column is an ordinary width.")
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
func Bare_Row_Invariants(identifier string, text Bare_Row) {
	invariant.Range(identifier, len(text), BARE_ROW_MIN, BARE_ROW_MAX)
	invariant.Sometimes(identifier, BARE_ROW_MIN < len(text) && len(text) < BARE_ROW_MAX,
		"The assembled row is an ordinary width.")
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
func Full_Row_Invariants(identifier string, text Full_Row) {
	invariant.Range(identifier, len(text), FULL_ROW_MIN, FULL_ROW_MAX)
	invariant.Sometimes(identifier, FULL_ROW_MIN < len(text) && len(text) < FULL_ROW_MAX,
		"The complete row is an ordinary width.")
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
func Delta_Body_Invariants(identifier string, text Delta_Body) {
	invariant.Range(identifier, len(text), DELTA_BODY_MIN, DELTA_BODY_MAX)
	invariant.Sometimes(identifier, DELTA_BODY_MIN < len(text) && len(text) < DELTA_BODY_MAX,
		"The delta body is an ordinary width.")
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
func Delta_Text_Invariants(identifier string, text Delta_Text) {
	invariant.Range(identifier, len(text), DELTA_TEXT_MIN, DELTA_TEXT_MAX)
	invariant.Sometimes(identifier, DELTA_TEXT_MIN < len(text) && len(text) < DELTA_TEXT_MAX,
		"The rendered delta is an ordinary width.")
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
func Frequency_Invariants(identifier string, text Frequency) {
	invariant.Range(identifier, len(text), FREQUENCY_BYTES_MIN, FREQUENCY_BYTES_MAX, 2)
	invariant.Sometimes(identifier, 2 < len(text) && len(text) < FREQUENCY_BYTES_MAX,
		"The rendered frequency is an ordinary length.")
}

// SUFFIX_BYTES_MIN is the empty base unit a suffix floors at.
const SUFFIX_BYTES_MIN = 0

// SUFFIX_BYTES_MAX is the longest a unit suffix is: the binary byte suffixes like "MiB".
const SUFFIX_BYTES_MAX = 3

// Suffix is a unit suffix on a scaled quantity — "", "s", "ms", "MiB". A distinct type
// so the suffix ladder's trusted text carries a length invariant.
type Suffix string

// Suffix_Invariants bounds a suffix's length and witnesses each boundary; a suffix spans
// the empty base unit through the three-byte binary suffixes.
func Suffix_Invariants(identifier string, unit Suffix) {
	invariant.Always(len(unit) <= SUFFIX_BYTES_MAX,
		"A unit suffix never exceeds the binary width.")
	invariant.Sometimes(identifier, len(unit) == SUFFIX_BYTES_MIN,
		"The unit suffix is the empty base.")
	invariant.Sometimes(identifier, len(unit) == 1, "The unit suffix is one byte.")
	invariant.Sometimes(identifier, len(unit) == 2, "The unit suffix is two bytes.")
	invariant.Sometimes(identifier, len(unit) == SUFFIX_BYTES_MAX,
		"The unit suffix is the binary width.")
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
func Phase_Invariants(identifier string, name Phase) {
	invariant.Always(len(name) == PHASE_BYTES_MIN || len(name) == PHASE_BYTES_MAX,
		"A phase is the empty sampling word or the warmup word.")
	invariant.Sometimes(identifier, len(name) == PHASE_BYTES_MIN,
		"The phase is the sampling default.")
	invariant.Sometimes(identifier, len(name) == PHASE_BYTES_MAX,
		"The phase is the warmup word.")
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
func Extent_Invariants(identifier string, value Extent) {
	invariant.Range(identifier, int(value), EXTENT_MIN, EXTENT_MAX)
	invariant.Sometimes(identifier, EXTENT_MIN < int(value) && int(value) < EXTENT_MAX,
		"The column width is an ordinary extent.")
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
func Kept_Invariants(identifier string, value Kept) {
	invariant.Always(int(value) >= KEPT_MIN, "A kept-run count is never below the quorum.")
	invariant.Always(int(value) <= KEPT_MAX, "A kept-run count is never above the run cap.")
	invariant.Sometimes(identifier, int(value) == KEPT_MIN,
		"The kept-run count is the quorum floor.")
	invariant.Sometimes(identifier, int(value) == KEPT_MAX,
		"The kept-run count fills the run cap.")
	invariant.Sometimes(identifier, KEPT_MIN < int(value) && int(value) < KEPT_MAX,
		"The kept-run count is an ordinary quorum.")
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
func Degree_Invariants(identifier string, value Degree) {
	invariant.Range(identifier, int(value), DEGREE_MIN, DEGREE_MAX)
	invariant.Sometimes(identifier, DEGREE_MIN < int(value) && int(value) < DEGREE_MAX,
		"The pooled degrees are an ordinary count.")
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
func Census_Invariants(identifier string, value Census) {
	invariant.Always(int(value) >= CENSUS_MIN, "A census is never below one.")
	invariant.Always(int(value) <= CENSUS_MAX, "A census never exceeds the run cap.")
	invariant.Sometimes(identifier, int(value) == CENSUS_MIN, "The census is one.")
	invariant.Sometimes(identifier, int(value) == 2, "The census is two.")
	invariant.Sometimes(identifier, int(value) == CENSUS_MAX,
		"The census fills the run cap.")
	invariant.Sometimes(identifier, 2 < int(value) && int(value) < CENSUS_MAX,
		"The census is an ordinary count.")
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
func Divisor_Invariants(identifier string, value Divisor) {
	invariant.Always(int(value) >= DIVISOR_MIN,
		"A Bessel divisor is never below the quorum less one.")
	invariant.Always(int(value) <= DIVISOR_MAX,
		"A Bessel divisor is never above a full run less one.")
	invariant.Sometimes(identifier, int(value) == DIVISOR_MIN,
		"The Bessel divisor is the quorum floor's.")
	invariant.Sometimes(identifier, int(value) == DIVISOR_MAX,
		"The Bessel divisor is a full run's.")
	invariant.Sometimes(identifier, DIVISOR_MIN < int(value) && int(value) < DIVISOR_MAX,
		"The Bessel divisor is an ordinary count.")
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
func Tally_Invariants(identifier string, value Tally) {
	invariant.Always(int(value) >= TALLY_MIN, "A tally is never negative.")
	invariant.Always(int(value) <= TALLY_MAX, "A tally never exceeds the run cap.")
	invariant.Sometimes(identifier, int(value) == TALLY_MIN, "The tally is zero.")
	invariant.Sometimes(identifier, int(value) == 1, "The tally is one.")
	invariant.Sometimes(identifier, int(value) == 2, "The tally is two.")
	invariant.Sometimes(identifier, int(value) == TALLY_MAX, "The tally fills the run cap.")
	invariant.Sometimes(identifier, 2 < int(value) && int(value) < TALLY_MAX,
		"The tally is an ordinary count.")
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
func Strays_Invariants(identifier string, value Strays) {
	invariant.Always(int(value) >= STRAYS_MIN, "An outlier count is never negative.")
	invariant.Always(int(value) <= STRAYS_MAX,
		"An outlier count never exceeds the outer half.")
	invariant.Sometimes(identifier, int(value) == STRAYS_MIN, "The outlier count is zero.")
	invariant.Sometimes(identifier, int(value) == 1, "The outlier count is one.")
	invariant.Sometimes(identifier, int(value) == 2, "The outlier count is two.")
	invariant.Sometimes(identifier, int(value) == STRAYS_MAX,
		"The outlier count fills the outer half.")
	invariant.Sometimes(identifier, 2 < int(value) && int(value) < STRAYS_MAX,
		"The outlier count is an ordinary tally.")
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
func Position_Invariants(identifier string, value Position) {
	invariant.Always(int(value) >= POSITION_MIN, "A position is never negative.")
	invariant.Always(int(value) <= POSITION_MAX, "A position never exceeds the last slot.")
	invariant.Sometimes(identifier, int(value) == POSITION_MIN, "The position is the first.")
	invariant.Sometimes(identifier, int(value) == 1, "The position is the second.")
	invariant.Sometimes(identifier, int(value) == 2, "The position is the third.")
	invariant.Sometimes(identifier, int(value) == POSITION_MAX,
		"The position is the last slot.")
	invariant.Sometimes(identifier, 2 < int(value) && int(value) < POSITION_MAX,
		"The position is an ordinary slot.")
}

// EXIT_CODE_MIN is the success code zero a code floors at.
const EXIT_CODE_MIN = 0

// EXIT_CODE_MAX is the failure code one a code ceilings at; the binary returns only
// success or failure.
const EXIT_CODE_MAX = 1

// Exit_Code is the process exit code — success or failure, never more. A distinct type
// bounding it to the two codes the binary actually returns.
type Exit_Code int

// Exit_Code_Invariants bounds an exit code to success or failure; the higher and negative
// codes are unreachable, and the success min and failure max are witnessed.
func Exit_Code_Invariants(identifier string, value Exit_Code) {
	invariant.Range(identifier, int(value), EXIT_CODE_MIN, EXIT_CODE_MAX)
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
func Exit_Status_Invariants(identifier string, value Exit_Status) {
	invariant.Always(int(value) >= EXIT_STATUS_MIN,
		"A command exit status is never negative.")
	invariant.Always(int(value) <= EXIT_STATUS_MAX,
		"A command exit status never exceeds the wait byte.")
	invariant.Sometimes(identifier, int(value) == EXIT_STATUS_MIN,
		"The command exit status is success.")
	invariant.Sometimes(identifier, int(value) == 1, "The command exit status is one.")
	invariant.Sometimes(identifier, int(value) == 2, "The command exit status is two.")
	invariant.Sometimes(identifier, int(value) == EXIT_STATUS_MAX,
		"The command exit status is the byte ceiling.")
	invariant.Sometimes(identifier, 2 < int(value) && int(value) < EXIT_STATUS_MAX,
		"The command exit status is an ordinary code.")
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
func Failure_Status_Invariants(identifier string, value Failure_Status) {
	invariant.Always(int(value) >= FAILURE_STATUS_MIN,
		"A failure status is never success.")
	invariant.Always(int(value) <= FAILURE_STATUS_MAX,
		"A failure status never exceeds the wait byte.")
	invariant.Sometimes(identifier, int(value) == FAILURE_STATUS_MIN,
		"The failure status is one.")
	invariant.Sometimes(identifier, int(value) == 2, "The failure status is two.")
	invariant.Sometimes(identifier, int(value) == FAILURE_STATUS_MAX,
		"The failure status is the byte ceiling.")
	invariant.Sometimes(identifier, 2 < int(value) && int(value) < FAILURE_STATUS_MAX,
		"The failure status is an ordinary code.")
}

// CORES_COUNT_MIN is the zero a core count floors at: a CPU without a hybrid split
// reports no performance or efficiency cores.
const CORES_COUNT_MIN = 0

// CORES_COUNT_MAX bounds a core count to a realistic, render-safe topology: even the
// widest server fits, and the core-layout line stays within its length.
const CORES_COUNT_MAX = 1023

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
func Hertz_Invariants(identifier string, value Hertz) {
	invariant.Always(uint64(value) <= HERTZ_MAX,
		"A CPU frequency stays within the realistic ceiling.")
	invariant.Sometimes(identifier, uint64(value) == HERTZ_MIN,
		"The CPU frequency is unknown.")
	invariant.Sometimes(identifier, uint64(value) == 1, "The CPU frequency is one hertz.")
	invariant.Sometimes(identifier, uint64(value) == 2, "The CPU frequency is two hertz.")
	invariant.Sometimes(identifier, uint64(value) == HERTZ_MAX,
		"The CPU frequency fills the realistic ceiling.")
	invariant.Sometimes(identifier, 2 < uint64(value) && uint64(value) < HERTZ_MAX,
		"The CPU frequency is an ordinary rating.")
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
func Byte_Size_Invariants(identifier string, value Byte_Size) {
	invariant.Range(identifier, uint64(value), BYTE_SIZE_MIN, BYTE_SIZE_MAX)
	invariant.Sometimes(identifier, uint64(value) == 1, "The byte size is one.")
	invariant.Sometimes(identifier, uint64(value) == 2, "The byte size is two.")
	invariant.Sometimes(identifier, 2 < uint64(value) && uint64(value) < BYTE_SIZE_MAX,
		"The byte size is an ordinary capacity.")
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
func Gap_Invariants(identifier string, value Gap) {
	invariant.Range(identifier, uint64(value), GAP_MIN, GAP_MAX)
	invariant.Sometimes(identifier, uint64(value) == 1, "The deviation magnitude is one.")
	invariant.Sometimes(identifier, uint64(value) == 2, "The deviation magnitude is two.")
	invariant.Sometimes(identifier, 2 < uint64(value) && uint64(value) < GAP_MAX,
		"The deviation magnitude is an ordinary distance.")
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

// Accumulator_High is the upper 64-bit half of the 128-bit sum-of-squares accumulator — a small
// register value, bounded by the high half of the greatest sum a distribution reaches.
type Accumulator_High uint64

// Accumulator_High_Invariants bounds the high word to the high half of the greatest sum of
// squared deviations a distribution reaches.
func Accumulator_High_Invariants(identifier string, value Accumulator_High) {
	invariant.Always(uint64(value) <= ACCUMULATOR_HIGH_MAX,
		"A high accumulator word never exceeds the greatest sum of squares.")
	invariant.Sometimes(identifier, uint64(value) == ACCUMULATOR_MIN,
		"The high accumulator word is zero.")
	invariant.Sometimes(identifier, uint64(value) == 1,
		"The high accumulator word is one.")
	invariant.Sometimes(identifier, uint64(value) == 2,
		"The high accumulator word is two.")
	invariant.Sometimes(identifier, uint64(value) == ACCUMULATOR_HIGH_MAX,
		"The high accumulator word fills its ceiling.")
	invariant.Sometimes(identifier,
		2 < uint64(value) && uint64(value) < ACCUMULATOR_HIGH_MAX,
		"The high accumulator word is an ordinary sum.")
}

// ACCUMULATOR_LOW_MAX is the low word's ceiling: the full word width. The low half is a modular
// residue of the running sum, so as the sum grows past 2^64 it ranges across the whole word.
const ACCUMULATOR_LOW_MAX uint64 = 1<<64 - 1

// Accumulator_Low is the lower 64-bit half of the 128-bit sum-of-squares accumulator — a
// full-width modular residue of the running sum.
type Accumulator_Low uint64

// Accumulator_Low_Invariants bounds the low word to the full word width it spans as a modular
// residue of the running sum.
func Accumulator_Low_Invariants(identifier string, value Accumulator_Low) {
	invariant.Sometimes(identifier, uint64(value) == ACCUMULATOR_MIN,
		"The low accumulator word is zero.")
	invariant.Sometimes(identifier, uint64(value) == 1, "The low accumulator word is one.")
	invariant.Sometimes(identifier, uint64(value) == 2, "The low accumulator word is two.")
	invariant.Sometimes(identifier, uint64(value) == ACCUMULATOR_LOW_MAX,
		"The low accumulator word is saturated.")
	invariant.Sometimes(identifier,
		2 < uint64(value) && uint64(value) < ACCUMULATOR_LOW_MAX,
		"The low accumulator word is an ordinary residue.")
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
func Metric_Invariants(identifier string, value Metric) {
	invariant.Range(identifier, int64(value), METRIC_MIN, METRIC_MAX)
	invariant.Sometimes(identifier, value == 1, "The metric is one.")
	invariant.Sometimes(identifier, value == 2, "The metric is two.")
	invariant.Sometimes(identifier, 2 < value && value < METRIC_MAX,
		"The metric is an ordinary magnitude.")
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
func Average_Invariants(identifier string, value Average) {
	invariant.Range(identifier, int64(value), AVERAGE_MIN, AVERAGE_MAX)
	invariant.Sometimes(identifier, value == 1, "The integer mean is one.")
	invariant.Sometimes(identifier, value == 2, "The integer mean is two.")
	invariant.Sometimes(identifier, 2 < value && value < AVERAGE_MAX,
		"The integer mean is an ordinary magnitude.")
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
func Points_Invariants(identifier string, values Points) {
	invariant.Always(len(values) >= QUORUM_MIN, "A sorted run is never below the quorum.")
	invariant.Always(len(values) <= POINTS_MAX, "A sorted run never exceeds the sample cap.")
	invariant.Sometimes(identifier, len(values) == QUORUM_MIN,
		"The sorted run is the quorum floor.")
	invariant.Sometimes(identifier, len(values) == POINTS_MAX,
		"The sorted run fills the sample cap.")
	invariant.Sometimes(identifier, QUORUM_MIN < len(values) && len(values) < POINTS_MAX,
		"The sorted run is an ordinary length.")
}

// LABEL_BYTES_MIN is the empty command label a progress line floors at.
const LABEL_BYTES_MIN = 0

// LABEL_BYTES_MAX bounds a progress label: the command text is truncated to a rune cap, so
// at most that many runes survive, each at most four UTF-8 bytes.
const LABEL_BYTES_MAX = PROGRESS_LABEL_RUNES_MAX * 4

// Label is the truncated command text on a progress line — bounded to one terminal row,
// so a distinct type holds it to a length invariant, not the untrusted-string content
// preset whose megabyte axis a truncated label can never reach.
type Label string

// Label_Invariants bounds a label's byte length: the empty min, the truncation max, and
// the one- and two-byte shapes between are witnessed.
func Label_Invariants(identifier string, text Label) {
	invariant.Range(identifier, len(text), LABEL_BYTES_MIN, LABEL_BYTES_MAX)
	invariant.Sometimes(identifier, len(text) == 1, "The progress label is one byte.")
	invariant.Sometimes(identifier, len(text) == 2, "The progress label is two bytes.")
	invariant.Sometimes(identifier, 2 < len(text) && len(text) < LABEL_BYTES_MAX,
		"The progress label is an ordinary length.")
}

// Right_Aligned is which side the padder aligns to: right when set, left otherwise.
type Right_Aligned bool

// Right_Aligned_Invariants witnesses both alignments, carrying the demand the old
// pad.right grid stated.
func Right_Aligned_Invariants(identifier string, value Right_Aligned) {
	invariant.Sometimes(identifier, bool(value), "The column right-aligns.")
}

// Pad aligns text to a visible width with spaces — on the left when right is set, so the
// text right-aligns, on the right otherwise. The width is the rune count, so a multibyte
// glyph like σ still counts as one column. One padder, so every column width passes one site
// and the width and padded-line invariants see the whole layout's range, not a per-side slice.
func pad(text Column, width Extent, right Right_Aligned) (result Padded) {
	defer func() { Padded_Invariants("pad.padded", result) }()
	Column_Invariants("pad.text", text)
	Extent_Invariants("pad.width", width)
	Right_Aligned_Invariants("pad.right", right)
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
	defer func() { Delta_Text_Invariants("delta_render.text", text) }()
	Delta_Invariants("delta_render.delta", delta)
	Color_Invariants("delta_render.color", color)
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

// Scale_Step is one rung of a scaling ladder: the divisor above which the suffix applies.
type Scale_Step struct {
	// Divisor is the magnitude the value is divided by at this rung.
	Divisor fixedpoint.Number
	// Suffix names the unit at this rung — empty at the base rung, so it is a suffix, not a
	// cell, whose length range admits the empty base unit.
	Suffix Suffix
}

// Scale_Step_Invariants composes a rung's suffix; the divisor carries no bundle of
// its own.
func Scale_Step_Invariants(identifier string, step Scale_Step) {
	Suffix_Invariants(identifier, step.Suffix)
}

// Format_quantity renders a raw value scaled to a human unit with three significant
// figures — poop's printUnit, e.g. 14906807 nanoseconds becomes "14.9ms".
func format_quantity(value fixedpoint.Number, unit Unit) (text Cell) {
	defer func() { Cell_Invariants("format_quantity.text", text) }()
	Unit_Invariants("format_quantity.unit", unit)
	scaled, unit_suffix := scale_quantity(value, unit)
	return Cell(string(format_significant(scaled)) + string(unit_suffix))
}

// Scale_quantity divides a value down to its human magnitude and names the unit
// suffix, dispatching on the metric's unit.
func scale_quantity(
	value fixedpoint.Number, unit Unit,
) (scaled fixedpoint.Number, unit_suffix Suffix) {
	defer func() { Suffix_Invariants("scale_quantity.suffix", unit_suffix) }()
	Unit_Invariants("scale_quantity.unit", unit)
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
	value fixedpoint.Number, ladder Ladder,
) (scaled fixedpoint.Number, unit_suffix Suffix) {
	defer func() { Suffix_Invariants("scale_ladder.suffix", unit_suffix) }()
	Ladder_Invariants("scale_ladder.ladder", ladder)
	for _, step := range ladder {
		if value >= step.Divisor {
			scaled = fixedpoint.Divide(&fixedpoint.Divide_Input{
				Dividend: value, Divisor: step.Divisor,
			})
			return scaled, Suffix(step.Suffix)
		}
	}
	return value, ""
}

// Time_ladder is the nanosecond-to-kilosecond ladder; the base unit is ns.
func time_ladder() (ladder Ladder) {
	defer func() {
		Ladder_Invariants("time_ladder.ladder", ladder)
		for _, x := range ladder {
			Scale_Step_Invariants("time_ladder.step", x)
		}
	}()
	return Ladder{
		{Divisor: fixedpoint.From_Integer(1_000_000_000_000), Suffix: "ks"},
		{Divisor: fixedpoint.From_Integer(1_000_000_000), Suffix: "s"},
		{Divisor: fixedpoint.From_Integer(1_000_000), Suffix: "ms"},
		{Divisor: fixedpoint.From_Integer(1_000), Suffix: "us"},
		{Divisor: fixedpoint.From_Integer(1), Suffix: "ns"},
	}
}

// Byte_ladder is the binary (1024) ladder with IEC suffixes, since memory is a
// power-of-two quantity; the base unit is B.
func byte_ladder() (ladder Ladder) {
	defer func() {
		Ladder_Invariants("byte_ladder.ladder", ladder)
		for _, x := range ladder {
			Scale_Step_Invariants("byte_ladder.step", x)
		}
	}()
	return Ladder{
		{Divisor: fixedpoint.From_Integer(1024 * 1024 * 1024 * 1024), Suffix: "TiB"},
		{Divisor: fixedpoint.From_Integer(1024 * 1024 * 1024), Suffix: "GiB"},
		{Divisor: fixedpoint.From_Integer(1024 * 1024), Suffix: "MiB"},
		{Divisor: fixedpoint.From_Integer(1024), Suffix: "KiB"},
		{Divisor: fixedpoint.From_Integer(1), Suffix: "B"},
	}
}

// Count_ladder is the metric-prefix ladder for a bare count; the base unit is unnamed.
func count_ladder() (ladder Ladder) {
	defer func() {
		Ladder_Invariants("count_ladder.ladder", ladder)
		for _, x := range ladder {
			Scale_Step_Invariants("count_ladder.step", x)
		}
	}()
	return Ladder{
		{Divisor: fixedpoint.From_Integer(1_000_000_000_000), Suffix: "T"},
		{Divisor: fixedpoint.From_Integer(1_000_000_000), Suffix: "G"},
		{Divisor: fixedpoint.From_Integer(1_000_000), Suffix: "M"},
		{Divisor: fixedpoint.From_Integer(1_000), Suffix: "K"},
	}
}

// Format_significant renders a scaled value to three significant figures: whole
// numbers and hundreds with no decimals, tens with one, units with two.
func format_significant(value fixedpoint.Number) (text Glyph) {
	defer func() { Glyph_Invariants("format_significant.text", text) }()
	if value >= fixedpoint.From_Integer(1000) {
		return Glyph(fixedpoint.Format(value, 0))
	}
	if fixedpoint.Is_Integer(value) {
		return Glyph(fixedpoint.Format(value, 0))
	}
	if value >= fixedpoint.From_Integer(100) {
		return Glyph(fixedpoint.Format(value, 0))
	}
	if value >= fixedpoint.From_Integer(10) {
		return Glyph(fixedpoint.Format(value, 1))
	}
	return Glyph(fixedpoint.Format(value, 2))
}

// Paint wraps text in an ANSI color when color is enabled; the codes have zero
// visible width, so wrapping after padding leaves alignment intact.
func paint(text Delta_Body, code Ansi_Code, color Color) (painted Delta_Text) {
	defer func() { Delta_Text_Invariants("paint.painted", painted) }()
	Delta_Body_Invariants("paint.text", text)
	Ansi_Code_Invariants("paint.code", code)
	Color_Invariants("paint.color", color)
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

// Render_Progress_Input is one progress update: the command being sampled, how long
// it has been sampling, and how many runs are done against the cap.
type Render_Progress_Input struct {
	// Command is the command being sampled.
	Command sysio.Process_Request
	// Elapsed is how long it has been sampling.
	Elapsed time.Duration
	// Phase is whether the run is warming up or sampling.
	Phase Phase
	// Count is how many runs have completed so far.
	Count Census
	// Total is the run cap the count is measured against.
	Total Progress_Total
}

// Render_Progress_Input_Invariants composes the update's fields; the command and
// elapsed duration carry no bundle of their own.
func Render_Progress_Input_Invariants(identifier string, input Render_Progress_Input) {
	Phase_Invariants(identifier, input.Phase)
	Census_Invariants(identifier, input.Count)
	Progress_Total_Invariants(identifier, input.Total)
}

// Progress_Total is the run cap a progress counter is measured against; zero when the
// cap is disabled.
type Progress_Total int

// Progress_Total_Invariants witnesses the shapes the old full-width grid demanded with
// the cap's own texts; the full-width domain enforces nothing.
func Progress_Total_Invariants(identifier string, value Progress_Total) {
	invariant.Sometimes(identifier, value == 1, "The progress total is one.")
	invariant.Sometimes(identifier, value == -1, "The progress total is negative one.")
	invariant.Sometimes(identifier, int64(value) == math.MinInt64,
		"The progress total is the word minimum.")
	invariant.Sometimes(identifier, int64(value) == math.MaxInt64,
		"The progress total is the word maximum.")
	invariant.Sometimes(identifier,
		value != 1 && value != -1 &&
			int64(value) != math.MinInt64 && int64(value) != math.MaxInt64,
		"The progress total is an ordinary count.")
}

// Render_progress writes an in-place progress line to stderr: elapsed seconds, an
// optional phase word (warmup), the counter (count over its total, or just the count
// when the total is disabled), and the command. Gated by Progress at the call site.
func render_progress(stderr io.Writer, input *Render_Progress_Input) {
	Render_Progress_Input_Invariants("render_progress.input", *input)
	seconds := fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
		Numerator: int64(input.Elapsed), Denominator: int64(time.SECOND),
	})
	counter := strconv.Itoa(int(input.Count))
	if input.Total > 0 {
		counter = counter + "/" + strconv.Itoa(int(input.Total))
	}
	phase_text := ""
	if input.Phase != "" {
		phase_text = string(input.Phase) + " "
	}
	command_label := progress_label(command_words(input.Command))
	output := PROGRESS_CLEAR + fixedpoint.Format(seconds, 1) + "s  " +
		phase_text + counter + "  " + string(command_label)
	stderr.Write([]byte(output))
}

// Progress_label joins the command words and truncates them to keep the progress
// line on one terminal row.
func progress_label(words Command_Line) (text Label) {
	defer func() { Label_Invariants("progress_label.label", text) }()
	Command_Line_Invariants("progress_label.words", words)
	parts := make([]string, len(words))
	for index, word := range words {
		Command_Word_Invariants("progress_label.word", word)
		parts[index] = string(word)
	}
	joined := strings.Join(parts, " ")
	runes := []rune(joined)
	if len(runes) <= PROGRESS_LABEL_RUNES_MAX {
		return Label(joined)
	}
	return Label(string(runes[:PROGRESS_LABEL_RUNES_MAX-1]) + "…")
}
