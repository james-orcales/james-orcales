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
	"math/bits"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"local/james-orcales/shared/fixedpoint"
	invariant "local/james-orcales/shared/invariant/default"
	sysio "local/james-orcales/shared/io"
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
	defer func() { Exit_Code_Invariants(code, "Main.exit_code") }()
	Main_Input_Invariants(input, "Main.input")
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
			deltas := deltas_compute(&Deltas_Compute_Input{
				Reference: reference,
				Candidate: measurements,
			})
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

// Output_Format_Invariants bounds an Output_Format to its declared rendering modes
// and claims each boundary of that range.
func Output_Format_Invariants(format Output_Format, namespace invariant.Namespace) {
	invariant.Always(format <= OUTPUT_FORMAT_JSON, "An Output_Format is at most Json.")
	invariant.Always(format >= OUTPUT_FORMAT_TABLE, "An Output_Format is at least Table.")
	invariant.Always(format != 2, "An Output_Format is never two.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(format == 0, "An Output_Format is zero."),
		invariant.Sometimes(format == 1, "An Output_Format is one."),
		invariant.Sometimes(format == OUTPUT_FORMAT_TABLE, "An Output_Format is Table."),
		invariant.Sometimes(format == OUTPUT_FORMAT_JSON, "An Output_Format is Json."),
		// Table is zero and Json is one, so those axes are pinned to the literals; a
		// format is one of the two, so it is never both and never neither.
		invariant.Impossible(
			invariant.Event_True("An Output_Format is zero."),
			invariant.Event_False("An Output_Format is Table."),
		),
		invariant.Impossible(
			invariant.Event_False("An Output_Format is zero."),
			invariant.Event_True("An Output_Format is Table."),
		),
		invariant.Impossible(
			invariant.Event_True("An Output_Format is one."),
			invariant.Event_False("An Output_Format is Json."),
		),
		invariant.Impossible(
			invariant.Event_False("An Output_Format is one."),
			invariant.Event_True("An Output_Format is Json."),
		),
		invariant.Impossible(
			invariant.Event_True("An Output_Format is zero."),
			invariant.Event_True("An Output_Format is one."),
		),
		invariant.Impossible(
			invariant.Event_False("An Output_Format is zero."),
			invariant.Event_False("An Output_Format is one."),
		),
	)
}

// OUTPUT_FORMAT_TABLE renders the human-readable comparison table; the default.
const OUTPUT_FORMAT_TABLE Output_Format = 0

// OUTPUT_FORMAT_JSON renders the machine-readable JSON document.
const OUTPUT_FORMAT_JSON Output_Format = 1

// Sample is one run's measurements.
type Sample struct {
	// Wall is the run's elapsed time, measured and reported by the sampler that ran it.
	Wall time.Duration
	// RSS_Bytes_Max is the run's peak physical memory footprint, in bytes.
	RSS_Bytes_Max Metric
	// CPU_Cycles is the run's CPU cycle count from the hardware counters.
	CPU_Cycles Metric
	// Instructions is the run's retired-instruction count from the counters.
	Instructions Metric
	// Cache_References is the run's last-level cache reference count (Linux only).
	Cache_References Metric
	// Cache_Misses is the run's last-level cache miss count (Linux only).
	Cache_Misses Metric
	// Branch_Misses is the run's mispredicted-branch count (Linux only).
	Branch_Misses Metric
	// CPU_User is the run's user-space CPU time.
	CPU_User time.Duration
	// CPU_System is the run's kernel-space CPU time.
	CPU_System time.Duration
}

// Sample_Invariants states a Sample's counter fields as the bounded metrics they are;
// the time.Duration fields carry their metric bound at the selector, not here. A counter
// past the representable ceiling, or a garbage negative one, trips the metric guard rather
// than silently overflowing the statistics downstream.
func Sample_Invariants(sample Sample, namespace invariant.Namespace) {
	Metric_Invariants(sample.RSS_Bytes_Max, "Sample.RSS_Bytes_Max")
	Metric_Invariants(sample.CPU_Cycles, "Sample.CPU_Cycles")
	Metric_Invariants(sample.Instructions, "Sample.Instructions")
	Metric_Invariants(sample.Cache_References, "Sample.Cache_References")
	Metric_Invariants(sample.Cache_Misses, "Sample.Cache_Misses")
	Metric_Invariants(sample.Branch_Misses, "Sample.Branch_Misses")
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
	invariant.Always(len(output) <= CAPTURE_BYTES_MAX, "Captured output is at most its max.")
	invariant.Always(len(output) >= CAPTURE_BYTES_MIN, "Captured output is at least its min.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(output) == 0, "Captured output is empty."),
		invariant.Sometimes(len(output) == 1, "Captured output is one byte."),
		invariant.Sometimes(len(output) == 2, "Captured output is two bytes."),
		invariant.Sometimes(len(output) == CAPTURE_BYTES_MIN, "Captured output is at min."),
		invariant.Sometimes(len(output) == CAPTURE_BYTES_MAX, "Captured output is full."),
		invariant.Impossible(
			invariant.Event_True("Captured output is empty."),
			invariant.Event_False("Captured output is at min."),
		),
		invariant.Impossible(
			invariant.Event_False("Captured output is empty."),
			invariant.Event_True("Captured output is at min."),
		),
		invariant.Impossible(
			invariant.Event_True("Captured output is empty."),
			invariant.Event_True("Captured output is one byte."),
		),
		invariant.Impossible(
			invariant.Event_True("Captured output is empty."),
			invariant.Event_True("Captured output is two bytes."),
		),
		invariant.Impossible(
			invariant.Event_True("Captured output is empty."),
			invariant.Event_True("Captured output is full."),
		),
		invariant.Impossible(
			invariant.Event_True("Captured output is at min."),
			invariant.Event_True("Captured output is one byte."),
		),
		invariant.Impossible(
			invariant.Event_True("Captured output is at min."),
			invariant.Event_True("Captured output is two bytes."),
		),
		invariant.Impossible(
			invariant.Event_True("Captured output is at min."),
			invariant.Event_True("Captured output is full."),
		),
		invariant.Impossible(
			invariant.Event_True("Captured output is one byte."),
			invariant.Event_True("Captured output is two bytes."),
		),
		invariant.Impossible(
			invariant.Event_True("Captured output is one byte."),
			invariant.Event_True("Captured output is full."),
		),
		invariant.Impossible(
			invariant.Event_True("Captured output is two bytes."),
			invariant.Event_True("Captured output is full."),
		),
	)
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

// Run_Result_Invariants states a Run_Result's fields.
func Run_Result_Invariants(result Run_Result, namespace invariant.Namespace) {
	Sample_Invariants(result.Sample, "Run_Result.Sample")
	Exit_Status_Invariants(result.Exit, "Run_Result.Exit")
	Captured_Output_Invariants(result.Stderr, "Run_Result.Stderr")
}

// COLLECTION_MIN is the empty count: the floor a failure or an empty input leaves a
// sample or value collection at. QUORUM_MIN and DEVIATION_MIN are the non-empty floors
// of a distribution and a deviation input; SAMPLES_MAX is the kept-run ceiling they share.
const COLLECTION_MIN = 0

// QUORUM_MIN is 3, the floor below which a distribution is too thin to trust.
const QUORUM_MIN = 3

// DEVIATION_MIN is 1, since a deviation is defined only once there is a value to spread.
const DEVIATION_MIN = 1

// Samples is what one command's collection yields: empty when the first run fails before
// the minimum, or a full distribution. The one- and two-run counts between never occur,
// since the 3-run minimum is reached in one uninterrupted stretch or not at all.
type Samples []Sample

// Samples_Invariants bounds the collected count: the empty floor and the kept-run ceiling
// are witnessed, the one- and two-run shapes claimed unreachable.
func Samples_Invariants(samples Samples, namespace invariant.Namespace) {
	invariant.Always(len(samples) <= SAMPLES_MAX, "A sample set is at most its max.")
	invariant.Always(len(samples) >= COLLECTION_MIN, "A sample set is at least its min.")
	invariant.Always(len(samples) != 1, "A sample set never has one.")
	invariant.Always(len(samples) != 2, "A sample set never has two.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(samples) == 0, "A sample set is empty."),
		invariant.Sometimes(len(samples) == COLLECTION_MIN, "A sample set at min."),
		invariant.Impossible(
			invariant.Event_True("A sample set is empty."),
			invariant.Event_False("A sample set at min."),
		),
		invariant.Impossible(
			invariant.Event_False("A sample set is empty."),
			invariant.Event_True("A sample set at min."),
		),
	)
}

// Distribution is a quorum of kept runs the statistics reduce — always at least the 3-run
// minimum, never the empty, one, or two counts the reducers cannot summarize.
type Distribution []Sample

// Distribution_Invariants bounds the run count: the quorum floor and the kept-run ceiling
// are witnessed, every short boundary claimed unreachable.
func Distribution_Invariants(samples Distribution, namespace invariant.Namespace) {
	invariant.Always(len(samples) <= SAMPLES_MAX, "A distribution is at most its max.")
	invariant.Always(len(samples) >= QUORUM_MIN, "A distribution is at least its min.")
	invariant.Always(len(samples) != 0, "A distribution is never empty.")
	invariant.Always(len(samples) != 1, "A distribution never has one.")
	invariant.Always(len(samples) != 2, "A distribution never has two.")
	// The ceiling is the arbitrary sample cap, reached only by grinding a full run — the
	// Always guard holds it, so only the quorum floor is witnessed.
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(samples) == QUORUM_MIN, "A distribution is at min."),
	)
}

// Values is one metric's value pulled from every sample, the int64 the statistics work in.
type Values []int64

// Values_Invariants bounds the value count: the empty min and the kept-run max are
// witnessed alongside the one- and two-value shapes between.
func Values_Invariants(values Values, namespace invariant.Namespace) {
	invariant.Always(len(values) <= SAMPLES_MAX, "A value set is at most its max.")
	invariant.Always(len(values) >= COLLECTION_MIN, "A value set is at least its min.")
	// A metric's values are extracted from a kept distribution, which the 3-run quorum
	// makes at least three — never empty, one, or two — so those counts are guarded, not
	// witnessed. Reducing a distribution to statistics is only ever asked of a quorum.
	invariant.Always(len(values) != 0, "A value set is never empty.")
	invariant.Always(len(values) != 1, "A value set never has one.")
	invariant.Always(len(values) != 2, "A value set never has two.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(values) == QUORUM_MIN, "A value set is at its quorum."),
	)
}

// Series is one metric pulled from a distribution — always the quorum's worth, never the
// empty, one, or two counts a distribution cannot hold.
type Series []int64

// Series_Invariants bounds the extracted count: it mirrors a distribution's quorum floor
// and kept-run ceiling, claiming every short boundary unreachable.
func Series_Invariants(values Series, namespace invariant.Namespace) {
	invariant.Always(len(values) <= SAMPLES_MAX, "A series is at most its max.")
	invariant.Always(len(values) >= QUORUM_MIN, "A series is at least its min.")
	invariant.Always(len(values) != 0, "A series is never empty.")
	invariant.Always(len(values) != 1, "A series never has one.")
	invariant.Always(len(values) != 2, "A series never has two.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(values) == QUORUM_MIN, "A series is at min."),
	)
}

// Deviations is the sorted values the standard deviation reduces; the compute guards the
// empty case, so it is never empty, but the one- and two-value sets do reach it.
type Deviations []int64

// Deviations_Invariants bounds the count: empty is claimed unreachable (the compute
// returns early), while the single-value min, the two-value shape, and the max are witnessed.
func Deviations_Invariants(values Deviations, namespace invariant.Namespace) {
	invariant.Always(len(values) <= SAMPLES_MAX, "A deviation set is at most its max.")
	invariant.Always(len(values) >= DEVIATION_MIN, "A deviation set is at least its min.")
	// The deviations are a kept distribution's, which the 3-run quorum makes at least three
	// — never empty, one, or two — so those counts are guarded, not witnessed.
	invariant.Always(len(values) != 0, "A deviation set is never empty.")
	invariant.Always(len(values) != 1, "A deviation set never has one.")
	invariant.Always(len(values) != 2, "A deviation set never has two.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(values) == QUORUM_MIN, "A deviation set is at its quorum."),
	)
}

// WORD_MIN is the lone executable a command line always carries; STRUCTURE_MAX is the
// ceiling the small structural collections — command lines, command sets, benchmark
// sets — share, generous enough to never be reached by a real invocation.
const WORD_MIN = 1

// STRUCTURE_MAX is 1<<8, headroom no real command structure reaches yet bounds the checks.
const STRUCTURE_MAX = 1 << 8

// Command_Line is one command flattened to its words — the executable and its arguments.
type Command_Line []Command_Word

// Command_Line_Invariants bounds the word count; a command always has its executable, so
// the empty count is unreachable while the one-word min, two-word shape, and max are witnessed.
func Command_Line_Invariants(words Command_Line, namespace invariant.Namespace) {
	invariant.Always(len(words) <= STRUCTURE_MAX, "A command line is at most its max.")
	invariant.Always(len(words) >= WORD_MIN, "A command line is at least its min.")
	invariant.Always(len(words) != 0, "A command line is never empty.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(words) == 1, "A command line has one word."),
		invariant.Sometimes(len(words) == 2, "A command line has two words."),
		invariant.Sometimes(len(words) == WORD_MIN, "A command line is at min."),
		invariant.Impossible(
			invariant.Event_True("A command line has one word."),
			invariant.Event_False("A command line is at min."),
		),
		invariant.Impossible(
			invariant.Event_False("A command line has one word."),
			invariant.Event_True("A command line is at min."),
		),
		invariant.Impossible(
			invariant.Event_True("A command line has one word."),
			invariant.Event_True("A command line has two words."),
		),
		invariant.Impossible(
			invariant.Event_True("A command line has two words."),
			invariant.Event_True("A command line is at min."),
		),
	)
}

// Commands is the set of commands a run benchmarks, in invocation order.
type Commands []sysio.Process_Request

// Commands_Invariants bounds the command count: the empty min, the one- and two-command
// shapes, and the max are witnessed.
func Commands_Invariants(commands Commands, namespace invariant.Namespace) {
	invariant.Always(len(commands) <= STRUCTURE_MAX, "A command set is at most its max.")
	invariant.Always(len(commands) >= COLLECTION_MIN, "A command set is at least its min.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(commands) == 0, "A command set is empty."),
		invariant.Sometimes(len(commands) == 1, "A command set has one."),
		invariant.Sometimes(len(commands) == 2, "A command set has two."),
		invariant.Sometimes(len(commands) == COLLECTION_MIN, "A command set at min."),
		invariant.Impossible(
			invariant.Event_True("A command set is empty."),
			invariant.Event_False("A command set at min."),
		),
		invariant.Impossible(
			invariant.Event_False("A command set is empty."),
			invariant.Event_True("A command set at min."),
		),
		invariant.Impossible(
			invariant.Event_True("A command set is empty."),
			invariant.Event_True("A command set has one."),
		),
		invariant.Impossible(
			invariant.Event_True("A command set is empty."),
			invariant.Event_True("A command set has two."),
		),
		invariant.Impossible(
			invariant.Event_True("A command set at min."),
			invariant.Event_True("A command set has one."),
		),
		invariant.Impossible(
			invariant.Event_True("A command set at min."),
			invariant.Event_True("A command set has two."),
		),
		invariant.Impossible(
			invariant.Event_True("A command set has one."),
			invariant.Event_True("A command set has two."),
		),
	)
}

// Benchmarks is one entry per benchmarked command, in invocation order.
type Benchmarks []Benchmark

// Benchmarks_Invariants bounds the entry count: the empty min, the one- and two-entry
// shapes, and the max are witnessed.
func Benchmarks_Invariants(benchmarks Benchmarks, namespace invariant.Namespace) {
	invariant.Always(len(benchmarks) <= STRUCTURE_MAX, "A benchmark set is at most its max.")
	invariant.Always(len(benchmarks) >= COLLECTION_MIN, "A benchmark set is at least its min.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(benchmarks) == 0, "A benchmark set is empty."),
		invariant.Sometimes(len(benchmarks) == 1, "A benchmark set has one."),
		invariant.Sometimes(len(benchmarks) == 2, "A benchmark set has two."),
		invariant.Sometimes(len(benchmarks) == COLLECTION_MIN, "A benchmark set at min."),
		invariant.Impossible(
			invariant.Event_True("A benchmark set is empty."),
			invariant.Event_False("A benchmark set at min."),
		),
		invariant.Impossible(
			invariant.Event_False("A benchmark set is empty."),
			invariant.Event_True("A benchmark set at min."),
		),
		invariant.Impossible(
			invariant.Event_True("A benchmark set is empty."),
			invariant.Event_True("A benchmark set has one."),
		),
		invariant.Impossible(
			invariant.Event_True("A benchmark set is empty."),
			invariant.Event_True("A benchmark set has two."),
		),
		invariant.Impossible(
			invariant.Event_True("A benchmark set at min."),
			invariant.Event_True("A benchmark set has one."),
		),
		invariant.Impossible(
			invariant.Event_True("A benchmark set at min."),
			invariant.Event_True("A benchmark set has two."),
		),
		invariant.Impossible(
			invariant.Event_True("A benchmark set has one."),
			invariant.Event_True("A benchmark set has two."),
		),
	)
}

// REPORT_MAX bounds the rendered bytes — a ceiling a real document's render stays well
// under, witnessed directly at its full length.
const REPORT_MAX = 1 << 16

// Report is the rendered output of a document — the table or the JSON bytes.
type Report []byte

// Report_Invariants bounds the rendered length; a report is empty only for an empty
// document and otherwise carries structure, never a lone one or two bytes.
func Report_Invariants(report Report, namespace invariant.Namespace) {
	invariant.Always(len(report) <= REPORT_MAX, "A report is at most its max.")
	invariant.Always(len(report) >= COLLECTION_MIN, "A report is at least its min.")
	invariant.Always(len(report) != 1, "A report is never one byte.")
	invariant.Always(len(report) != 2, "A report is never two bytes.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(report) == 0, "A report is empty."),
		invariant.Sometimes(len(report) == COLLECTION_MIN, "A report is at min."),
		invariant.Impossible(
			invariant.Event_True("A report is empty."),
			invariant.Event_False("A report is at min."),
		),
		invariant.Impossible(
			invariant.Event_False("A report is empty."),
			invariant.Event_True("A report is at min."),
		),
	)
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
	invariant.Always(len(text) <= HOST_TEXT_BYTES_MAX, "A host field is at most its max.")
	invariant.Always(len(text) >= HOST_TEXT_BYTES_MIN, "A host field is at least its min.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(text) == 0, "A host field is empty."),
		invariant.Sometimes(len(text) == 1, "A host field is one byte."),
		invariant.Sometimes(len(text) == 2, "A host field is two bytes."),
		invariant.Sometimes(len(text) == HOST_TEXT_BYTES_MIN, "A host field is at min."),
		invariant.Sometimes(len(text) == HOST_TEXT_BYTES_MAX, "A host field is at max."),
		invariant.Impossible(
			invariant.Event_True("A host field is empty."),
			invariant.Event_False("A host field is at min."),
		),
		invariant.Impossible(
			invariant.Event_False("A host field is empty."),
			invariant.Event_True("A host field is at min."),
		),
		invariant.Impossible(
			invariant.Event_True("A host field is empty."),
			invariant.Event_True("A host field is one byte."),
		),
		invariant.Impossible(
			invariant.Event_True("A host field is empty."),
			invariant.Event_True("A host field is two bytes."),
		),
		invariant.Impossible(
			invariant.Event_True("A host field is empty."),
			invariant.Event_True("A host field is at max."),
		),
		invariant.Impossible(
			invariant.Event_True("A host field is at min."),
			invariant.Event_True("A host field is one byte."),
		),
		invariant.Impossible(
			invariant.Event_True("A host field is at min."),
			invariant.Event_True("A host field is two bytes."),
		),
		invariant.Impossible(
			invariant.Event_True("A host field is at min."),
			invariant.Event_True("A host field is at max."),
		),
		invariant.Impossible(
			invariant.Event_True("A host field is one byte."),
			invariant.Event_True("A host field is two bytes."),
		),
		invariant.Impossible(
			invariant.Event_True("A host field is one byte."),
			invariant.Event_True("A host field is at max."),
		),
		invariant.Impossible(
			invariant.Event_True("A host field is two bytes."),
			invariant.Event_True("A host field is at max."),
		),
	)
}

// Machine_Specs is a snapshot of the host hardware and OS taken once at startup,
// carried in the report so benchmark results are reproducible across machines.
type Machine_Specs struct {
	// CPU_Model is the CPU's brand string, e.g. "Apple M4 Max".
	CPU_Model Host_Text `json:"cpu_model"`
	// CPU_Arch is the instruction-set architecture, e.g. "arm64".
	CPU_Arch Host_Text `json:"cpu_arch"`
	// Physical_Cores is the total physical core count across all performance levels.
	Physical_Cores Cores `json:"physical_cores"`
	// Logical_Cores is the OS-visible thread count, which may exceed Physical_Cores
	// when hyperthreading or SMT is active.
	Logical_Cores Cores `json:"logical_cores"`
	// Performance_Cores is the P-core count on hybrid CPUs (Apple Silicon, Alder Lake+).
	// Zero when the CPU does not expose a performance/efficiency split.
	Performance_Cores Cores `json:"performance_cores,omitempty"`
	// Efficiency_Cores is the E-core count on hybrid CPUs.
	Efficiency_Cores Cores `json:"efficiency_cores,omitempty"`
	// CPU_Frequency_Hz_Max is the maximum rated CPU frequency in Hz; zero when the
	// kernel does not expose it (e.g. Apple Silicon with no cpufrequency_max sysctl).
	CPU_Frequency_Hz_Max Hertz `json:"cpu_frequency_hz_max,omitempty"`
	// Cache_L1_Bytes is the per-core L1 data cache size in bytes.
	Cache_L1_Bytes Byte_Size `json:"cache_l1_bytes,omitempty"`
	// Cache_L2_Bytes is the per-core L2 cache size in bytes.
	Cache_L2_Bytes Byte_Size `json:"cache_l2_bytes,omitempty"`
	// Cache_L3_Bytes is the shared L3 cache size in bytes.
	Cache_L3_Bytes Byte_Size `json:"cache_l3_bytes,omitempty"`
	// RAM_Total_Bytes is the total installed physical memory in bytes.
	RAM_Total_Bytes Byte_Size `json:"ram_total_bytes"`
	// Storage_Total_Bytes is the total capacity of the boot filesystem in bytes. It
	// is a benchmark-relevant proxy for SSD throughput: on Apple Silicon a larger
	// drive spreads I/O across more NAND dies, so the 512GB model reads and writes
	// faster than the 256GB even on identical silicon.
	Storage_Total_Bytes Byte_Size `json:"storage_total_bytes"`
	// Operating_System_Name is the operating-system name, e.g. "macOS".
	Operating_System_Name Host_Text `json:"operating_system_name"`
	// Operating_System_Version is the OS release, e.g. "15.2".
	Operating_System_Version Host_Text `json:"operating_system_version"`
	// Kernel_Version is the kernel release string, e.g. "Darwin 25.2.0".
	Kernel_Version Host_Text `json:"kernel_version"`
}

// Machine_Specs_Invariants states every primitive field of the host snapshot.
func Machine_Specs_Invariants(specs Machine_Specs, namespace invariant.Namespace) {
	Host_Text_Invariants(specs.CPU_Model, "Machine_Specs.CPU_Model")
	Host_Text_Invariants(specs.CPU_Arch, "Machine_Specs.CPU_Arch")
	Cores_Invariants(specs.Physical_Cores, "Machine_Specs.Physical_Cores")
	Cores_Invariants(specs.Logical_Cores, "Machine_Specs.Logical_Cores")
	Cores_Invariants(specs.Performance_Cores, "Machine_Specs.Performance_Cores")
	Cores_Invariants(specs.Efficiency_Cores, "Machine_Specs.Efficiency_Cores")
	Hertz_Invariants(specs.CPU_Frequency_Hz_Max, "Machine_Specs.CPU_Frequency_Hz_Max")
	Byte_Size_Invariants(specs.Cache_L1_Bytes, "Machine_Specs.Cache_L1_Bytes")
	Byte_Size_Invariants(specs.Cache_L2_Bytes, "Machine_Specs.Cache_L2_Bytes")
	Byte_Size_Invariants(specs.Cache_L3_Bytes, "Machine_Specs.Cache_L3_Bytes")
	Byte_Size_Invariants(specs.RAM_Total_Bytes, "Machine_Specs.RAM_Total_Bytes")
	Byte_Size_Invariants(specs.Storage_Total_Bytes, "Machine_Specs.Storage_Total_Bytes")
	Host_Text_Invariants(
		specs.Operating_System_Name, "Machine_Specs.Operating_System_Name")
	Host_Text_Invariants(
		specs.Operating_System_Version, "Machine_Specs.Operating_System_Version")
	Host_Text_Invariants(specs.Kernel_Version, "Machine_Specs.Kernel_Version")
}

// UNIT_BYTES_MIN is the empty unit: the zero value a unitless Measurement carries.
const UNIT_BYTES_MIN = 0

// UNIT_BYTES_MAX is the longest unit in the vocabulary — "nanoseconds", eleven bytes.
// Units are a closed set, so a longer value is a malformed probe to reject.
const UNIT_BYTES_MAX = 11

// Unit names the dimension a Measurement's raw values are in — a short word from a fixed
// vocabulary, or empty when the distribution carries no unit (a zero-value Measurement).
type Unit string

// Unit_Invariants bounds a unit's length: the empty min and the longest-word max are
// witnessed, and the one- and two-byte shapes between are claimed unreachable.
func Unit_Invariants(name Unit, namespace invariant.Namespace) {
	invariant.Always(len(name) <= UNIT_BYTES_MAX, "A unit is at most its max.")
	invariant.Always(len(name) >= UNIT_BYTES_MIN, "A unit is at least its min.")
	// A rendered unit is always one of the fixed vocabulary — "count", "bytes",
	// "nanoseconds" — never empty, one, or two bytes, so those lengths are guarded. The
	// longest word is the max; the shorter words leave it unwitnessed.
	invariant.Always(len(name) != 0, "A unit is never unset.")
	invariant.Always(len(name) != 1, "A unit is never one byte.")
	invariant.Always(len(name) != 2, "A unit is never two bytes.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(name) == UNIT_BYTES_MAX, "A unit is at max."),
	)
}

// Measurement is the distribution of one metric across a command's runs — poop's
// Measurement, reduced from the raw Samples. Every field is a number so the report
// marshals to JSON without custom encoders.
type Measurement struct {
	// Mean is the arithmetic mean of the metric's values.
	Mean fixedpoint.Number `json:"mean"`
	// Standard_Deviation is the sample standard deviation, with an n-1 denominator.
	Standard_Deviation fixedpoint.Number `json:"stddev"`
	// Min is the smallest value observed.
	Min fixedpoint.Number `json:"min"`
	// Max is the largest value observed.
	Max fixedpoint.Number `json:"max"`
	// Median is the middle value of the sorted values.
	Median fixedpoint.Number `json:"median"`
	// Q1 is the first quartile by poop's index math.
	Q1 fixedpoint.Number `json:"q1"`
	// Q3 is the third quartile by poop's index math.
	Q3 fixedpoint.Number `json:"q3"`
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
	Strays_Invariants(measurement.Outlier_Count, "Measurement.Outlier_Count")
	Kept_Invariants(measurement.Sample_Count, "Measurement.Sample_Count")
	Unit_Invariants(measurement.Unit, "Measurement.Unit")
}

// Delta is one metric's change in a candidate command relative to the reference —
// poop's colored ratio column, as data.
type Delta struct {
	// Diff_Percent is the candidate's mean as a signed percentage of the reference's.
	Diff_Percent fixedpoint.Number `json:"diff_percent"`
	// Half_Percent is the half-width of the 95% confidence interval on Diff_Percent.
	Half_Percent fixedpoint.Number `json:"half_percent"`
	// Significant is true only when the interval clears the ±1% band.
	Significant bool `json:"significant"`
	// Faster is true when the candidate's mean is below the reference's.
	Faster bool `json:"faster"`
}

// Delta_Invariants states the boolean fields of a metric's change; the
// fixedpoint.Number fields have no preset of their own.
func Delta_Invariants(delta Delta, namespace invariant.Namespace) {
	invariant.Boolean_Invariants(delta.Significant, "Delta.Significant")
	invariant.Boolean_Invariants(delta.Faster, "Delta.Faster")
}

// Measurements is the distribution of every metric for one command, in report
// order. The field order is the JSON order, so the report needs no custom encoder.
type Measurements struct {
	// Wall_Time is the elapsed-time distribution.
	Wall_Time Measurement `json:"wall_time"`
	// Peak_RSS is the peak-memory distribution.
	Peak_RSS Measurement `json:"peak_rss"`
	// CPU_Cycles is the CPU-cycle distribution.
	CPU_Cycles Measurement `json:"cpu_cycles"`
	// Instructions is the retired-instruction distribution.
	Instructions Measurement `json:"instructions"`
	// Cache_References is the cache-reference distribution (Linux only).
	Cache_References Measurement `json:"cache_references"`
	// Cache_Misses is the cache-miss distribution (Linux only).
	Cache_Misses Measurement `json:"cache_misses"`
	// Branch_Misses is the branch-miss distribution (Linux only).
	Branch_Misses Measurement `json:"branch_misses"`
	// CPU_User is the user-CPU-time distribution.
	CPU_User Measurement `json:"cpu_user"`
	// CPU_System is the system-CPU-time distribution.
	CPU_System Measurement `json:"cpu_system"`
}

// Measurements_Invariants composes the per-metric distribution of every field.
func Measurements_Invariants(measurements Measurements, namespace invariant.Namespace) {
	Measurement_Invariants(measurements.Wall_Time, "Measurements.Wall_Time")
	Measurement_Invariants(measurements.Peak_RSS, "Measurements.Peak_RSS")
	Measurement_Invariants(measurements.CPU_Cycles, "Measurements.CPU_Cycles")
	Measurement_Invariants(measurements.Instructions, "Measurements.Instructions")
	Measurement_Invariants(measurements.Cache_References, "Measurements.Cache_References")
	Measurement_Invariants(measurements.Cache_Misses, "Measurements.Cache_Misses")
	Measurement_Invariants(measurements.Branch_Misses, "Measurements.Branch_Misses")
	Measurement_Invariants(measurements.CPU_User, "Measurements.CPU_User")
	Measurement_Invariants(measurements.CPU_System, "Measurements.CPU_System")
}

// Deltas is every metric's change for one command relative to the reference, in the
// same order as Measurements.
type Deltas struct {
	// Wall_Time is the elapsed-time delta.
	Wall_Time Delta `json:"wall_time"`
	// Peak_RSS is the peak-memory delta.
	Peak_RSS Delta `json:"peak_rss"`
	// CPU_Cycles is the CPU-cycle delta.
	CPU_Cycles Delta `json:"cpu_cycles"`
	// Instructions is the retired-instruction delta.
	Instructions Delta `json:"instructions"`
	// Cache_References is the cache-reference delta (Linux only).
	Cache_References Delta `json:"cache_references"`
	// Cache_Misses is the cache-miss delta (Linux only).
	Cache_Misses Delta `json:"cache_misses"`
	// Branch_Misses is the branch-miss delta (Linux only).
	Branch_Misses Delta `json:"branch_misses"`
	// CPU_User is the user-CPU-time delta.
	CPU_User Delta `json:"cpu_user"`
	// CPU_System is the system-CPU-time delta.
	CPU_System Delta `json:"cpu_system"`
}

// Deltas_Invariants composes the change of every metric field.
func Deltas_Invariants(deltas Deltas, namespace invariant.Namespace) {
	Delta_Invariants(deltas.Wall_Time, "Deltas.Wall_Time")
	Delta_Invariants(deltas.Peak_RSS, "Deltas.Peak_RSS")
	Delta_Invariants(deltas.CPU_Cycles, "Deltas.CPU_Cycles")
	Delta_Invariants(deltas.Instructions, "Deltas.Instructions")
	Delta_Invariants(deltas.Cache_References, "Deltas.Cache_References")
	Delta_Invariants(deltas.Cache_Misses, "Deltas.Cache_Misses")
	Delta_Invariants(deltas.Branch_Misses, "Deltas.Branch_Misses")
	Delta_Invariants(deltas.CPU_User, "Deltas.CPU_User")
	Delta_Invariants(deltas.CPU_System, "Deltas.CPU_System")
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

// Benchmark_Invariants states a Benchmark's fields, composing the delta distribution
// like any other value field.
func Benchmark_Invariants(benchmark Benchmark, namespace invariant.Namespace) {
	Command_Line_Invariants(benchmark.Command, "Benchmark.Command")
	Kept_Invariants(benchmark.Runs, "Benchmark.Runs")
	Measurements_Invariants(benchmark.Measurements, "Benchmark.Measurements")
	Deltas_Invariants(benchmark.Deltas, "Benchmark.Deltas")
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
	Machine_Specs_Invariants(document.Machine, "Document.Machine")
	Benchmarks_Invariants(document.Benchmarks, "Document.Benchmarks")
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
	Runs_Max int
	// Warmup_Count is how many runs are taken and discarded before sampling.
	Warmup_Count int
	// Allow_Failures keeps benchmarking a command that exits non-zero.
	Allow_Failures bool
	// Format selects the report rendering; the zero value is the table.
	Format Output_Format
	// Color enables ANSI color in the table rendering.
	Color bool
	// Progress writes an in-place run counter to Stderr while sampling; gate it on an
	// interactive Stderr so piped output stays clean.
	Progress bool
	// Output is where the report is written.
	Output io.Writer
	// Stderr is where a failing command's diagnostics are written.
	Stderr io.Writer
	// Machine is the host hardware and OS snapshot; injected so the library makes no
	// ambient OS reads. Production wires acquire_machine_specs(); tests inject a stub.
	Machine Machine_Specs
}

// Main_Input_Invariants states the injected configuration's coverable fields; the
// clock, sampler functions, and writers have no preset of their own.
func Main_Input_Invariants(input Main_Input, namespace invariant.Namespace) {
	Commands_Invariants(input.Commands, "Main_Input.Commands")
	Sampler_Invariants(input.Sampler, "Main_Input.Sampler")
	invariant.Int_Invariants(input.Runs_Max, "Main_Input.Runs_Max")
	invariant.Int_Invariants(input.Warmup_Count, "Main_Input.Warmup_Count")
	invariant.Boolean_Invariants(input.Allow_Failures, "Main_Input.Allow_Failures")
	Output_Format_Invariants(input.Format, "Main_Input.Format")
	invariant.Boolean_Invariants(input.Color, "Main_Input.Color")
	invariant.Boolean_Invariants(input.Progress, "Main_Input.Progress")
	Machine_Specs_Invariants(input.Machine, "Main_Input.Machine")
}

// Main_input_collect_samples runs command through the warmup discards and then the
// measured loop, timing each kept run with the injected clock. A non-zero exit
// returns that exit and its stderr to abort the run; with Allow_Failures the run is
// kept and a zero exit is returned so sampling continues.
func main_input_collect_samples(
	input *Main_Input, command sysio.Process_Request,
) (samples Samples, exit Exit_Status, stderr Captured_Output) {
	defer func() {
		Samples_Invariants(samples, "main_input_collect_samples.samples")
		Exit_Status_Invariants(exit, "main_input_collect_samples.exit")
		Captured_Output_Invariants(stderr, "main_input_collect_samples.stderr")
	}()
	Main_Input_Invariants(*input, "main_input_collect_samples.input")
	warmups := 0
	// Elapsed is the running sum of measured wall time, the budget's clock: the library
	// reads no ambient clock, so a run's cost is the wall the sampler reports, accrued.
	var warmup_elapsed time.Duration
	for warmups < input.Warmup_Count {
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
		warmup_elapsed += warm.Sample.Wall
		if input.Progress {
			render_progress(input.Stderr, &Render_Progress_Input{
				Command: command,
				Elapsed: warmup_elapsed,
				Phase:   "warmup",
				Count:   Census(warmups),
				Total:   input.Warmup_Count,
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
			stopwatch_start = result.Completed_At - time.Moment(sample.Wall)
		}
		samples = append(samples, sample)
		elapsed = time.Duration(result.Completed_At - stopwatch_start)
		if input.Progress {
			render_progress(input.Stderr, &Render_Progress_Input{
				Command: command,
				Elapsed: elapsed,
				Count:   Census(len(samples)),
				Total:   input.Runs_Max,
			})
		}
	}
	return samples, 0, nil
}

// Sampling_Should_Continue_Input is the loop state sampling_should_continue judges.
type Sampling_Should_Continue_Input struct {
	// Elapsed is the wall time spent sampling so far.
	Elapsed time.Duration
	// Duration_Max is the time budget; zero disables it.
	Duration_Max time.Duration
	// Runs_Max is the run cap; zero disables it.
	Runs_Max int
	// Count is how many runs have been kept so far.
	Count Tally
}

// Sampling_Should_Continue_Input_Invariants states the loop state's integer fields;
// the durations have no preset of their own.
func Sampling_Should_Continue_Input_Invariants(
	input Sampling_Should_Continue_Input, namespace invariant.Namespace,
) {
	invariant.Int_Invariants(input.Runs_Max, "Sampling_Should_Continue_Input.Runs_Max")
	Tally_Invariants(input.Count, "Sampling_Should_Continue_Input.Count")
}

// Sampling_should_continue decides whether to take another sample. The 3-run minimum
// always wins first and the 10000-run cap always stops; between them, sampling stops
// when any active limit is met — the run cap or the time budget — and a limit of zero
// is inactive, so both zero leaves only the safety cap. The compound condition is
// split into nested single-term ifs for the linter.
func sampling_should_continue(input *Sampling_Should_Continue_Input) (yes bool) {
	defer func() {
		invariant.Boolean_Invariants(yes, "sampling_should_continue.yes")
	}()
	Sampling_Should_Continue_Input_Invariants(*input, "sampling_should_continue.input")
	if int(input.Count) < RUNS_MIN {
		return true
	}
	if int(input.Count) >= SAMPLES_MAX {
		return false
	}
	if input.Runs_Max > 0 {
		if int(input.Count) >= input.Runs_Max {
			return false
		}
	}
	if input.Duration_Max > 0 {
		if input.Elapsed >= input.Duration_Max {
			return false
		}
	}
	return true
}

// Samples_elapsed sums the wall time of the kept runs — how long the command's
// measured sampling took in total.
func samples_elapsed(samples Distribution) (elapsed time.Duration) {
	Distribution_Invariants(samples, "samples_elapsed.samples")
	for _, sample := range samples {
		elapsed += sample.Wall
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
	measurements.Wall_Time = Measurement_Compute(
		Values(extract(samples, sample_wall)), "nanoseconds")
	measurements.Peak_RSS = Measurement_Compute(
		Values(extract(samples, sample_rss)), "bytes")
	measurements.CPU_Cycles = Measurement_Compute(
		Values(extract(samples, sample_cycles)), "count")
	measurements.Instructions = Measurement_Compute(
		Values(extract(samples, sample_instructions)), "count")
	measurements.Cache_References = Measurement_Compute(
		Values(extract(samples, sample_cache_references)), "count")
	measurements.Cache_Misses = Measurement_Compute(
		Values(extract(samples, sample_cache_misses)), "count")
	measurements.Branch_Misses = Measurement_Compute(
		Values(extract(samples, sample_branch_misses)), "count")
	measurements.CPU_User = Measurement_Compute(
		Values(extract(samples, sample_user)), "nanoseconds")
	measurements.CPU_System = Measurement_Compute(
		Values(extract(samples, sample_system)), "nanoseconds")
	return measurements
}

// Deltas_Compute_Input pairs a reference and candidate distribution for comparison.
type Deltas_Compute_Input struct {
	// Reference is the baseline distribution deltas are measured against.
	Reference Measurements
	// Candidate is the distribution compared to the reference.
	Candidate Measurements
}

// Deltas_Compute_Input_Invariants composes the reference and candidate distributions.
func Deltas_Compute_Input_Invariants(
	input Deltas_Compute_Input, namespace invariant.Namespace,
) {
	Measurements_Invariants(input.Reference, "Deltas_Compute_Input.Reference")
	Measurements_Invariants(input.Candidate, "Deltas_Compute_Input.Candidate")
}

// Deltas_compute compares every metric of the candidate against the reference.
func deltas_compute(input *Deltas_Compute_Input) (deltas Deltas) {
	defer func() {
		Deltas_Invariants(deltas, "deltas_compute.deltas")
	}()
	Deltas_Compute_Input_Invariants(*input, "deltas_compute.input")
	reference := input.Reference
	candidate := input.Candidate
	deltas.Wall_Time = Compare(&Compare_Input{
		Reference: reference.Wall_Time, Candidate: candidate.Wall_Time,
	})
	deltas.Peak_RSS = Compare(&Compare_Input{
		Reference: reference.Peak_RSS, Candidate: candidate.Peak_RSS,
	})
	deltas.CPU_Cycles = Compare(&Compare_Input{
		Reference: reference.CPU_Cycles, Candidate: candidate.CPU_Cycles,
	})
	deltas.Instructions = Compare(&Compare_Input{
		Reference: reference.Instructions, Candidate: candidate.Instructions,
	})
	deltas.Cache_References = Compare(&Compare_Input{
		Reference: reference.Cache_References, Candidate: candidate.Cache_References,
	})
	deltas.Cache_Misses = Compare(&Compare_Input{
		Reference: reference.Cache_Misses, Candidate: candidate.Cache_Misses,
	})
	deltas.Branch_Misses = Compare(&Compare_Input{
		Reference: reference.Branch_Misses, Candidate: candidate.Branch_Misses,
	})
	deltas.CPU_User = Compare(&Compare_Input{
		Reference: reference.CPU_User, Candidate: candidate.CPU_User,
	})
	deltas.CPU_System = Compare(&Compare_Input{
		Reference: reference.CPU_System, Candidate: candidate.CPU_System,
	})
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
	return sample.RSS_Bytes_Max
}

// Sample_cycles reads a sample's CPU cycle count as a metric.
func sample_cycles(sample Sample) (value Metric) {
	defer func() { Metric_Invariants(value, "sample_cycles.value") }()
	Sample_Invariants(sample, "sample_cycles.sample")
	return sample.CPU_Cycles
}

// Sample_instructions reads a sample's retired-instruction count as a metric.
func sample_instructions(sample Sample) (value Metric) {
	defer func() { Metric_Invariants(value, "sample_instructions.value") }()
	Sample_Invariants(sample, "sample_instructions.sample")
	return sample.Instructions
}

// Sample_cache_references reads a sample's cache-reference count as a metric.
func sample_cache_references(sample Sample) (value Metric) {
	defer func() { Metric_Invariants(value, "sample_cache_references.value") }()
	Sample_Invariants(sample, "sample_cache_references.sample")
	return sample.Cache_References
}

// Sample_cache_misses reads a sample's cache-miss count as a metric.
func sample_cache_misses(sample Sample) (value Metric) {
	defer func() { Metric_Invariants(value, "sample_cache_misses.value") }()
	Sample_Invariants(sample, "sample_cache_misses.sample")
	return sample.Cache_Misses
}

// Sample_branch_misses reads a sample's branch-miss count as a metric.
func sample_branch_misses(sample Sample) (value Metric) {
	defer func() { Metric_Invariants(value, "sample_branch_misses.value") }()
	Sample_Invariants(sample, "sample_branch_misses.sample")
	return sample.Branch_Misses
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
func Command_Word_Invariants(word Command_Word, namespace invariant.Namespace) {
	invariant.Always(len(word) <= COMMAND_WORD_BYTES_MAX, "A word is at most its max.")
	invariant.Always(len(word) >= COMMAND_WORD_BYTES_MIN, "A word is at least its min.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(word) == 0, "A word is empty."),
		invariant.Sometimes(len(word) == 1, "A word is one byte."),
		invariant.Sometimes(len(word) == 2, "A word is two bytes."),
		invariant.Sometimes(len(word) == COMMAND_WORD_BYTES_MIN, "A word is at min."),
		invariant.Sometimes(len(word) == COMMAND_WORD_BYTES_MAX, "A word is at max."),
		invariant.Impossible(
			invariant.Event_True("A word is empty."),
			invariant.Event_False("A word is at min."),
		),
		invariant.Impossible(
			invariant.Event_False("A word is empty."),
			invariant.Event_True("A word is at min."),
		),
		invariant.Impossible(
			invariant.Event_True("A word is empty."),
			invariant.Event_True("A word is one byte."),
		),
		invariant.Impossible(
			invariant.Event_True("A word is empty."),
			invariant.Event_True("A word is two bytes."),
		),
		invariant.Impossible(
			invariant.Event_True("A word is empty."),
			invariant.Event_True("A word is at max."),
		),
		invariant.Impossible(
			invariant.Event_True("A word is at min."),
			invariant.Event_True("A word is one byte."),
		),
		invariant.Impossible(
			invariant.Event_True("A word is at min."),
			invariant.Event_True("A word is two bytes."),
		),
		invariant.Impossible(
			invariant.Event_True("A word is at min."),
			invariant.Event_True("A word is at max."),
		),
		invariant.Impossible(
			invariant.Event_True("A word is one byte."),
			invariant.Event_True("A word is two bytes."),
		),
		invariant.Impossible(
			invariant.Event_True("A word is one byte."),
			invariant.Event_True("A word is at max."),
		),
		invariant.Impossible(
			invariant.Event_True("A word is two bytes."),
			invariant.Event_True("A word is at max."),
		),
	)
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
func write_report(output io.Writer, document Document) (code Exit_Code) {
	defer func() { Exit_Code_Invariants(code, "write_report.exit_code") }()
	Document_Invariants(document, "write_report.document")
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
func Write_Failure_Input_Invariants(input Write_Failure_Input, namespace invariant.Namespace) {
	Position_Invariants(input.Index, "Write_Failure_Input.Index")
	Failure_Status_Invariants(input.Exit, "Write_Failure_Input.Exit")
	Captured_Output_Invariants(input.Child_Stderr, "Write_Failure_Input.Child_Stderr")
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
	q1 := fixedpoint.From_Integer(low_quartile)
	q3 := fixedpoint.From_Integer(high_quartile)
	mean := fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
		Numerator: total, Denominator: int64(count),
	})
	deviation := standard_deviation(sorted, Average(mean_integer), Kept(count))

	measurement = Measurement{
		Mean:               mean,
		Standard_Deviation: deviation,
		Min:                fixedpoint.From_Integer(sorted[0]),
		Max:                fixedpoint.From_Integer(sorted[count-1]),
		Median:             fixedpoint.From_Integer(sorted[count/2]),
		Q1:                 q1,
		Q3:                 q3,
		Outlier_Count: outlier_count(&Outlier_Count_Input{
			Sorted:        Points(sorted),
			Low_Quartile:  Metric(low_quartile),
			High_Quartile: Metric(high_quartile),
		}),
		Sample_Count: Kept(count),
		Unit:         unit,
	}
	return measurement
}

// Wide is a 128-bit unsigned accumulator, for a sum of squares that overflows int64.
type Wide struct {
	// High is the upper 64 bits.
	High Accumulator
	// Low is the lower 64 bits.
	Low Accumulator
}

// Wide_Invariants states the two halves of the 128-bit accumulator.
func Wide_Invariants(value Wide, namespace invariant.Namespace) {
	Accumulator_Invariants(value.High, "Wide.High")
	Accumulator_Invariants(value.Low, "Wide.Low")
}

// Wide_add_square adds value squared into a 128-bit accumulator.
func wide_add_square(subtotal Wide, value Gap) (sum Wide) {
	defer func() { Wide_Invariants(sum, "wide_add_square.sum") }()
	Wide_Invariants(subtotal, "wide_add_square.accumulator")
	Gap_Invariants(value, "wide_add_square.value")
	product_high, product_low := bits.Mul64(uint64(value), uint64(value))
	low, carry := bits.Add64(uint64(subtotal.Low), product_low, 0)
	high, _ := bits.Add64(uint64(subtotal.High), product_high, carry)
	return Wide{High: Accumulator(high), Low: Accumulator(low)}
}

// Standard_deviation is the sample standard deviation with an n-1 denominator. The sum
// of squared deviations is accumulated in 128 bits so a large metric's deviations cannot
// overflow before the divide and the fixed-point root.
func standard_deviation(
	sorted Deviations, center Average, count Kept,
) (deviation fixedpoint.Number) {
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
	High Accumulator
	// Low is the numerator's lower 64 bits.
	Low Accumulator
	// Denominator is the sample count less one, the Bessel correction.
	Denominator Divisor
}

// Root_Of_Quotient_Input_Invariants states the numerator halves and the divisor.
func Root_Of_Quotient_Input_Invariants(
	input Root_Of_Quotient_Input, namespace invariant.Namespace,
) {
	Accumulator_Invariants(input.High, "Root_Of_Quotient_Input.High")
	Accumulator_Invariants(input.Low, "Root_Of_Quotient_Input.Low")
	Divisor_Invariants(input.Denominator, "Root_Of_Quotient_Input.Denominator")
}

// Root_of_quotient returns the fixed-point square root of a 128-bit numerator over a
// denominator — the shared tail of the sample and pooled deviations. A quotient that fits
// a signed word keeps full fractional precision; a larger one, a multi-second jitter far
// outside maddox's fast-command envelope, falls back to the integer root.
func root_of_quotient(input *Root_Of_Quotient_Input) (deviation fixedpoint.Number) {
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
	Low_Quartile Metric
	// High_Quartile is the metric at q3; a value above q3 plus 1.5*IQR is an outlier.
	High_Quartile Metric
}

// Outlier_Count_Input_Invariants states the sorted values and the two quartiles they bracket.
func Outlier_Count_Input_Invariants(input Outlier_Count_Input, namespace invariant.Namespace) {
	Points_Invariants(input.Sorted, "Outlier_Count_Input.Sorted")
	Metric_Invariants(input.Low_Quartile, "Outlier_Count_Input.Low_Quartile")
	Metric_Invariants(input.High_Quartile, "Outlier_Count_Input.High_Quartile")
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

// Compare_Input pairs the reference and candidate measurements Compare contrasts.
type Compare_Input struct {
	// Reference is the baseline measurement deltas are taken against.
	Reference Measurement
	// Candidate is the measurement compared to the reference.
	Candidate Measurement
}

// Compare_Input_Invariants composes the reference and candidate measurements.
func Compare_Input_Invariants(input Compare_Input, namespace invariant.Namespace) {
	Measurement_Invariants(input.Reference, "Compare_Input.Reference")
	Measurement_Invariants(input.Candidate, "Compare_Input.Candidate")
}

// Compare reports how the candidate's mean differs from the reference's, with the
// 95% confidence half-interval from a pooled-variance two-sample t-test — poop's
// ratio computation. A zero or degenerate reference yields a zero, non-significant
// delta rather than a divide-by-zero.
func Compare(input *Compare_Input) (delta Delta) {
	defer func() { Delta_Invariants(delta, "Compare.delta") }()
	Compare_Input_Invariants(*input, "Compare.input")
	reference := input.Reference
	candidate := input.Candidate
	if reference.Mean == 0 {
		return delta
	}
	ratio := fixedpoint.Divide(&fixedpoint.Divide_Input{
		Dividend: candidate.Mean - reference.Mean, Divisor: reference.Mean,
	})
	delta.Diff_Percent = ratio * 100
	delta.Faster = candidate.Mean < reference.Mean

	degrees := candidate.Sample_Count + reference.Sample_Count - 2
	if degrees < 1 {
		return delta
	}
	delta.Half_Percent = half_interval(&Half_Interval_Input{
		Reference: reference, Candidate: candidate, Degrees: Degree(degrees),
	})
	delta.Significant = significant(&Significant_Input{
		Diff_Percent: delta.Diff_Percent,
		Half_Percent: delta.Half_Percent,
	})
	return delta
}

// Half_Interval_Input carries the two measurements and the degrees of freedom.
type Half_Interval_Input struct {
	// Reference is the baseline measurement.
	Reference Measurement
	// Candidate is the measurement compared to the reference.
	Candidate Measurement
	// Degrees is the pooled degrees of freedom, n1 + n2 - 2.
	Degrees Degree
}

// Half_Interval_Input_Invariants composes the measurements and states the degrees.
func Half_Interval_Input_Invariants(input Half_Interval_Input, namespace invariant.Namespace) {
	Measurement_Invariants(input.Reference, "Half_Interval_Input.Reference")
	Measurement_Invariants(input.Candidate, "Half_Interval_Input.Candidate")
	Degree_Invariants(input.Degrees, "Half_Interval_Input.Degrees")
}

// Half_interval is the 95% confidence half-width on Diff_Percent, from a pooled-variance
// two-sample t-test — poop's score*pooled*normalizer*100/mean. The pooled deviation is
// taken relative to the reference mean, folding in that final divide, so the math never
// forms a raw variance — which, for a metric in the billions, overflows.
func half_interval(input *Half_Interval_Input) (half fixedpoint.Number) {
	Half_Interval_Input_Invariants(*input, "half_interval.input")
	first := fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
		Numerator: 1, Denominator: int64(input.Candidate.Sample_Count),
	})
	second := fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
		Numerator: 1, Denominator: int64(input.Reference.Sample_Count),
	})
	normalizer := fixedpoint.Square_Root(first + second)
	pooled := pooled_deviation(&Pooled_Deviation_Input{
		Candidate: input.Candidate, Reference: input.Reference, Degrees: input.Degrees,
	})
	score := student_t_score(input.Degrees)
	band := fixedpoint.Multiply(&fixedpoint.Multiply_Input{A: score, B: pooled})
	band = fixedpoint.Multiply(&fixedpoint.Multiply_Input{A: band, B: normalizer})
	return band * 100
}

// Pooled_Deviation_Input carries the two measurements and the degrees of freedom.
type Pooled_Deviation_Input struct {
	// Candidate is the measurement compared to the reference.
	Candidate Measurement
	// Reference is the baseline measurement.
	Reference Measurement
	// Degrees is the pooled degrees of freedom, n1 + n2 - 2.
	Degrees Degree
}

// Pooled_Deviation_Input_Invariants composes the measurements and states the degrees.
func Pooled_Deviation_Input_Invariants(
	input Pooled_Deviation_Input, namespace invariant.Namespace,
) {
	Measurement_Invariants(input.Candidate, "Pooled_Deviation_Input.Candidate")
	Measurement_Invariants(input.Reference, "Pooled_Deviation_Input.Reference")
	Degree_Invariants(input.Degrees, "Pooled_Deviation_Input.Degrees")
}

// Pooled_deviation is the pooled standard deviation as a fraction of the reference mean:
// the root of the degrees-weighted mean of the two relative variances. Dividing each
// deviation by the mean before squaring keeps every value near one, so a metric in the
// billions and its enormous raw variance never overflow.
func pooled_deviation(input *Pooled_Deviation_Input) (pooled fixedpoint.Number) {
	Pooled_Deviation_Input_Invariants(*input, "pooled_deviation.input")
	mean := input.Reference.Mean
	candidate := fixedpoint.Divide(&fixedpoint.Divide_Input{
		Dividend: input.Candidate.Standard_Deviation, Divisor: mean,
	})
	reference := fixedpoint.Divide(&fixedpoint.Divide_Input{
		Dividend: input.Reference.Standard_Deviation, Divisor: mean,
	})
	candidate_variance := fixedpoint.Multiply(&fixedpoint.Multiply_Input{
		A: candidate, B: candidate,
	})
	reference_variance := fixedpoint.Multiply(&fixedpoint.Multiply_Input{
		A: reference, B: reference,
	})
	weighted := candidate_variance*fixedpoint.Number(input.Candidate.Sample_Count-1) +
		reference_variance*fixedpoint.Number(input.Reference.Sample_Count-1)
	return fixedpoint.Square_Root(weighted / fixedpoint.Number(int(input.Degrees)))
}

// Student_t_score returns the Student-t critical value for 95% confidence at the given
// degrees of freedom as a fixed-point number, falling back to the normal-distribution
// 1.96 past the tabulated range — poop's getStatScore95. The tables hold thousandths so
// they read as the published constants, and From_Ratio puts them on the fixed-point grid.
// The tables are local, not package globals, so the package keeps no mutable state.
func student_t_score(degrees_of_freedom Degree) (score fixedpoint.Number) {
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
	return fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
		Numerator: milli, Denominator: 1000,
	})
}

// Significant_Input carries the difference and its confidence half-interval, both as
// fixed-point percentages.
type Significant_Input struct {
	// Diff_Percent is the signed percentage difference under test.
	Diff_Percent fixedpoint.Number
	// Half_Percent is the confidence interval's half-width, as a percentage.
	Half_Percent fixedpoint.Number
}

// Significant_Input_Invariants states the one property the fields carry that
// fixedpoint.Number cannot: the half-interval is a confidence half-width, so it is
// never negative. The difference may have either sign and is left unconstrained.
func Significant_Input_Invariants(input Significant_Input, namespace invariant.Namespace) {
	invariant.Always(input.Half_Percent >= 0, "A confidence half-interval is never negative.")
}

// Significant decides whether a difference clears poop's ±1% band: the whole
// confidence interval must sit beyond ±1% with a single sign. The && and || of
// poop's check are split into nested single-term ifs to satisfy the linter.
func significant(input *Significant_Input) (is bool) {
	defer func() { invariant.Boolean_Invariants(is, "significant.is") }()
	Significant_Input_Invariants(*input, "significant.input")
	if input.Diff_Percent >= fixedpoint.From_Integer(1) {
		if input.Diff_Percent-input.Half_Percent >= fixedpoint.From_Integer(1) {
			return true
		}
	}
	if input.Diff_Percent <= fixedpoint.From_Integer(-1) {
		if input.Diff_Percent+input.Half_Percent <= fixedpoint.From_Integer(-1) {
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
func Ansi_Code_Invariants(code Ansi_Code, namespace invariant.Namespace) {
	invariant.Always(len(code) <= ANSI_CODE_BYTES_MAX, "ansi_code within max length.")
	invariant.Always(len(code) >= ANSI_CODE_BYTES_MIN, "ansi_code within min length.")
	invariant.Always(len(code) != 0, "ansi_code is never empty.")
	invariant.Always(len(code) != 1, "ansi_code is never one byte.")
	invariant.Always(len(code) != 2, "ansi_code is never two bytes.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(code) == ANSI_CODE_BYTES_MIN, "ansi_code length is min."),
		invariant.Sometimes(len(code) == ANSI_CODE_BYTES_MAX, "ansi_code length is max."),
		invariant.Impossible(
			invariant.Event_True("ansi_code length is min."),
			invariant.Event_True("ansi_code length is max."),
		),
		invariant.Impossible(
			invariant.Event_False("ansi_code length is min."),
			invariant.Event_False("ansi_code length is max."),
		),
	)
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
	Color bool
}

// Render_Table_Input_Invariants composes the report and states the color flag.
func Render_Table_Input_Invariants(input Render_Table_Input, namespace invariant.Namespace) {
	Document_Invariants(input.Document, "Render_Table_Input.Document")
	invariant.Boolean_Invariants(input.Color, "Render_Table_Input.Color")
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
	builder.WriteString("  cores: " + string(machine_specs_cores(m)) + "   " +
		"freq: " + string(format_hz(m.CPU_Frequency_Hz_Max)) + "   " +
		"ram: " + string(format_bytes(m.RAM_Total_Bytes)) + "   " +
		"storage: " + string(format_bytes(m.Storage_Total_Bytes)) + "\n")
	cache_line := ""
	if m.Cache_L1_Bytes > 0 {
		cache_line += "L1: " + string(format_bytes(m.Cache_L1_Bytes)) + "   "
	}
	if m.Cache_L2_Bytes > 0 {
		cache_line += "L2: " + string(format_bytes(m.Cache_L2_Bytes)) + "   "
	}
	if m.Cache_L3_Bytes > 0 {
		cache_line += "L3: " + string(format_bytes(m.Cache_L3_Bytes)) + "   "
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

// CORES_MAX bounds the core-layout string: the widest hybrid form with the largest counts
// the int fields hold in practice.
const CORES_MAX = 48

// Cores_Line is the rendered core-layout line — "8", "4 P + 4 E = 8 logical". A distinct type
// from a cell: it is wider than a formatted value, so it carries its own length range.
type Cores_Line string

// Cores_Line_Invariants bounds the core-layout length; it is never empty, with the single-digit
// min, the widest hybrid max, and the one- and two-byte shapes between witnessed.
func Cores_Line_Invariants(text Cores_Line, namespace invariant.Namespace) {
	invariant.Always(len(text) <= CORES_MAX, "A cores line is at most its max.")
	invariant.Always(len(text) >= CORES_MIN, "A cores line is at least its min.")
	invariant.Always(len(text) != 0, "A cores line is never empty.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(text) == 1, "A cores line is one byte."),
		invariant.Sometimes(len(text) == 2, "A cores line is two bytes."),
		invariant.Sometimes(len(text) == CORES_MIN, "A cores line is at min."),
		invariant.Impossible(
			invariant.Event_True("A cores line is one byte."),
			invariant.Event_False("A cores line is at min."),
		),
		invariant.Impossible(
			invariant.Event_False("A cores line is one byte."),
			invariant.Event_True("A cores line is at min."),
		),
		invariant.Impossible(
			invariant.Event_True("A cores line is one byte."),
			invariant.Event_True("A cores line is two bytes."),
		),
		invariant.Impossible(
			invariant.Event_True("A cores line is two bytes."),
			invariant.Event_True("A cores line is at min."),
		),
	)
}

// Machine_specs_cores renders the CPU core layout, naming performance and efficiency cores on
// hybrid CPUs and collapsing to a single count when physical equals logical.
func machine_specs_cores(m Machine_Specs) (text Cores_Line) {
	defer func() { Cores_Line_Invariants(text, "machine_specs_cores.text") }()
	Machine_Specs_Invariants(m, "machine_specs_cores.m")
	if m.Performance_Cores > 0 {
		if m.Efficiency_Cores > 0 {
			return Cores_Line(fmt.Sprintf("%d P + %d E = %d logical",
				m.Performance_Cores, m.Efficiency_Cores, m.Logical_Cores))
		}
	}
	if m.Physical_Cores == m.Logical_Cores {
		return Cores_Line(fmt.Sprintf("%d", m.Physical_Cores))
	}
	return Cores_Line(fmt.Sprintf("%d physical, %d logical", m.Physical_Cores, m.Logical_Cores))
}

// Format_hz renders a frequency in Hz to a human-readable GHz or MHz string.
func format_hz(hz Hertz) (text Frequency) {
	defer func() { Frequency_Invariants(text, "format_hz.text") }()
	Hertz_Invariants(hz, "format_hz.hz")
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
	defer func() { Cell_Invariants(text, "format_bytes.text") }()
	Byte_Size_Invariants(value, "format_bytes.value")
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
func Span_Invariants(text Span, namespace invariant.Namespace) {
	invariant.Always(len(text) <= SPAN_BYTES_MAX, "A span is at most its max length.")
	invariant.Always(len(text) >= SPAN_BYTES_MIN, "A span is at least its min length.")
	invariant.Always(len(text) != 0, "A span is never empty.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(text) == 1, "A span is one byte."),
		invariant.Sometimes(len(text) == 2, "A span is two bytes."),
		invariant.Sometimes(len(text) == SPAN_BYTES_MIN, "A span is at min."),
		invariant.Sometimes(len(text) == SPAN_BYTES_MAX, "A span is at max."),
		// One byte is the min, so those two events coincide; two bytes and the six-byte max
		// are each their own length, so no two boundary events ever share a span.
		invariant.Impossible(
			invariant.Event_True("A span is one byte."),
			invariant.Event_False("A span is at min."),
		),
		invariant.Impossible(
			invariant.Event_False("A span is one byte."),
			invariant.Event_True("A span is at min."),
		),
		invariant.Impossible(
			invariant.Event_True("A span is one byte."),
			invariant.Event_True("A span is two bytes."),
		),
		invariant.Impossible(
			invariant.Event_True("A span is one byte."),
			invariant.Event_True("A span is at max."),
		),
		invariant.Impossible(
			invariant.Event_True("A span is two bytes."),
			invariant.Event_True("A span is at max."),
		),
	)
}

// Format_elapsed renders a total sampling duration for the benchmark header with the time
// ladder. Like format_bytes it scales the raw nanoseconds down their rung before lifting into
// fixed-point, so a duration past the 2^20 scale's ceiling — a sum of many large run walls —
// reduces without overflow; and it caps the magnitude first, so an absurd multi-year span still
// lands within the header's glyph rather than rendering a five-digit count. The suffix is the
// rung's own, witnessed by time_ladder.
func format_elapsed(elapsed time.Duration) (text Span) {
	defer func() { Span_Invariants(text, "format_elapsed.text") }()
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
	defer func() { Exit_Code_Invariants(code, "write_table.exit_code") }()
	Render_Table_Input_Invariants(*input, "write_table.input")
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
func render_benchmark(builder *strings.Builder, index Position, benchmark Benchmark, color bool) {
	Position_Invariants(index, "render_benchmark.index")
	Benchmark_Invariants(benchmark, "render_benchmark.benchmark")
	invariant.Boolean_Invariants(color, "render_benchmark.color")
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
func render_metric_rows(builder *strings.Builder, index Position, benchmark Benchmark, color bool) {
	Position_Invariants(index, "render_metric_rows.index")
	Benchmark_Invariants(benchmark, "render_metric_rows.benchmark")
	invariant.Boolean_Invariants(color, "render_metric_rows.color")
	m := benchmark.Measurements
	d := benchmark.Deltas
	// The reference is first; only later benchmarks carry a comparison.
	has_delta := index != 0
	rows := []Metric_Line_Input{
		{Name: "wall_time", Measurement: m.Wall_Time, Delta: d.Wall_Time},
		{Name: "peak_rss", Measurement: m.Peak_RSS, Delta: d.Peak_RSS},
		{Name: "cpu_cycles", Measurement: m.CPU_Cycles, Delta: d.CPU_Cycles},
		{Name: "instructions", Measurement: m.Instructions, Delta: d.Instructions},
		{
			Name:        "cache_references",
			Measurement: m.Cache_References,
			Delta:       d.Cache_References,
		},
		{
			Name:        "cache_misses",
			Measurement: m.Cache_Misses,
			Delta:       d.Cache_Misses,
		},
		{
			Name:        "branch_misses",
			Measurement: m.Branch_Misses,
			Delta:       d.Branch_Misses,
		},
		{Name: "cpu_user", Measurement: m.CPU_User, Delta: d.CPU_User},
		{Name: "cpu_system", Measurement: m.CPU_System, Delta: d.CPU_System},
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
	Has_Delta bool
	// Color is whether the delta column is ANSI-colored.
	Color bool
}

// Metric_Line_Input_Invariants states the row's name, distribution, delta, and flags.
func Metric_Line_Input_Invariants(input Metric_Line_Input, namespace invariant.Namespace) {
	Caption_Invariants(input.Name, "Metric_Line_Input.Name")
	Measurement_Invariants(input.Measurement, "Metric_Line_Input.Measurement")
	Delta_Invariants(input.Delta, "Metric_Line_Input.Delta")
	invariant.Boolean_Invariants(input.Has_Delta, "Metric_Line_Input.Has_Delta")
	invariant.Boolean_Invariants(input.Color, "Metric_Line_Input.Color")
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
		outlier_percent = fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
			Numerator:   int64(measurement.Outlier_Count) * 100,
			Denominator: int64(measurement.Sample_Count),
		})
	}
	outlier_text := strconv.Itoa(int(measurement.Outlier_Count)) +
		" (" + fixedpoint.Format(outlier_percent, 0) + "%)"
	row := render_cells(&Render_Cells_Input{
		Name:     input.Name,
		Mean:     format_quantity(measurement.Mean, unit),
		Sigma:    format_quantity(measurement.Standard_Deviation, unit),
		Low:      format_quantity(measurement.Min, unit),
		High:     format_quantity(measurement.Max, unit),
		Outliers: Outliers(outlier_text),
	})
	text = Full_Row(row)
	if input.Has_Delta {
		delta := delta_render(input.Delta, input.Color)
		text = Full_Row(string(text) + "  " + string(delta))
	}
	return text
}

// Render_Cells_Input is one table row's cell texts.
type Render_Cells_Input struct {
	// Name is the row label.
	Name Caption
	// Mean is the mean cell.
	Mean Cell
	// Sigma is the standard-deviation cell.
	Sigma Cell
	// Low is the minimum cell.
	Low Cell
	// High is the maximum cell.
	High Cell
	// Outliers is the outlier-count cell.
	Outliers Outliers
}

// Render_Cells_Input_Invariants states every cell text of one row.
func Render_Cells_Input_Invariants(input Render_Cells_Input, namespace invariant.Namespace) {
	Caption_Invariants(input.Name, "Render_Cells_Input.Name")
	Cell_Invariants(input.Mean, "Render_Cells_Input.Mean")
	Cell_Invariants(input.Sigma, "Render_Cells_Input.Sigma")
	Cell_Invariants(input.Low, "Render_Cells_Input.Low")
	Cell_Invariants(input.High, "Render_Cells_Input.High")
	Outliers_Invariants(input.Outliers, "Render_Cells_Input.Outliers")
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
	invariant.Always(len(text) <= CELL_BYTES_MAX, "A cell is at most its max length.")
	invariant.Always(len(text) >= CELL_BYTES_MIN, "A cell is at least its min length.")
	invariant.Always(len(text) != 0, "A cell is never empty.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(text) == 1, "A cell is one byte."),
		invariant.Sometimes(len(text) == 2, "A cell is two bytes."),
		invariant.Sometimes(len(text) == CELL_BYTES_MIN, "A cell is at min."),
		invariant.Sometimes(len(text) == CELL_BYTES_MAX, "A cell is at max."),
		invariant.Impossible(
			invariant.Event_True("A cell is one byte."),
			invariant.Event_False("A cell is at min."),
		),
		invariant.Impossible(
			invariant.Event_False("A cell is one byte."),
			invariant.Event_True("A cell is at min."),
		),
		invariant.Impossible(
			invariant.Event_True("A cell is one byte."),
			invariant.Event_True("A cell is two bytes."),
		),
		invariant.Impossible(
			invariant.Event_True("A cell is one byte."),
			invariant.Event_True("A cell is at max."),
		),
		invariant.Impossible(
			invariant.Event_True("A cell is two bytes."),
			invariant.Event_True("A cell is at min."),
		),
		invariant.Impossible(
			invariant.Event_True("A cell is two bytes."),
			invariant.Event_True("A cell is at max."),
		),
		invariant.Impossible(
			invariant.Event_True("A cell is at min."),
			invariant.Event_True("A cell is at max."),
		),
	)
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
func Glyph_Invariants(text Glyph, namespace invariant.Namespace) {
	invariant.Always(len(text) <= GLYPH_MAX, "A glyph is at most its max.")
	invariant.Always(len(text) >= GLYPH_MIN, "A glyph is at least its min.")
	invariant.Always(len(text) != 0, "A glyph is never empty.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(text) == 1, "A glyph is one byte."),
		invariant.Sometimes(len(text) == 2, "A glyph is two bytes."),
		invariant.Sometimes(len(text) == GLYPH_MIN, "A glyph is at min."),
		invariant.Sometimes(len(text) == GLYPH_MAX, "A glyph is at max."),
		invariant.Impossible(
			invariant.Event_True("A glyph is one byte."),
			invariant.Event_False("A glyph is at min."),
		),
		invariant.Impossible(
			invariant.Event_False("A glyph is one byte."),
			invariant.Event_True("A glyph is at min."),
		),
		invariant.Impossible(
			invariant.Event_True("A glyph is one byte."),
			invariant.Event_True("A glyph is two bytes."),
		),
		invariant.Impossible(
			invariant.Event_True("A glyph is one byte."),
			invariant.Event_True("A glyph is at max."),
		),
		invariant.Impossible(
			invariant.Event_True("A glyph is two bytes."),
			invariant.Event_True("A glyph is at min."),
		),
		invariant.Impossible(
			invariant.Event_True("A glyph is two bytes."),
			invariant.Event_True("A glyph is at max."),
		),
		invariant.Impossible(
			invariant.Event_True("A glyph is at min."),
			invariant.Event_True("A glyph is at max."),
		),
	)
}

// COLUMN_MIN is the single byte a padded column floors at before alignment.
const COLUMN_MIN = 1

// COLUMN_MAX bounds the unpadded text a column pads: a metric name, an outlier tally, or a
// scaled value, the widest of which is the longest metric name with its delta.
const COLUMN_MAX = 25

// Column is the text handed to the padder for one table column — a value, a name, or an
// outlier count, before it is widened to the column. A distinct type covering that union,
// wider than a single scaled cell.
type Column string

// Column_Invariants bounds the column-text length; never empty, with the single-byte min,
// the widest text max, and the one- and two-byte shapes between witnessed.
func Column_Invariants(text Column, namespace invariant.Namespace) {
	invariant.Always(len(text) <= COLUMN_MAX, "A column is at most its max.")
	invariant.Always(len(text) >= COLUMN_MIN, "A column is at least its min.")
	invariant.Always(len(text) != 0, "A column is never empty.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(text) == 1, "A column is one byte."),
		invariant.Sometimes(len(text) == 2, "A column is two bytes."),
		invariant.Sometimes(len(text) == COLUMN_MIN, "A column is at min."),
		invariant.Impossible(
			invariant.Event_True("A column is one byte."),
			invariant.Event_False("A column is at min."),
		),
		invariant.Impossible(
			invariant.Event_False("A column is one byte."),
			invariant.Event_True("A column is at min."),
		),
		invariant.Impossible(
			invariant.Event_True("A column is one byte."),
			invariant.Event_True("A column is two bytes."),
		),
		invariant.Impossible(
			invariant.Event_True("A column is two bytes."),
			invariant.Event_True("A column is at min."),
		),
	)
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
	invariant.Always(len(text) <= CAPTION_MAX, "A caption is at most its max.")
	invariant.Always(len(text) >= CAPTION_MIN, "A caption is at least its min.")
	invariant.Always(len(text) != 0, "A caption is never empty.")
	invariant.Always(len(text) != 1, "A caption is never one byte.")
	invariant.Always(len(text) != 2, "A caption is never two bytes.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(text) == CAPTION_MIN, "A caption is at min."),
		invariant.Sometimes(len(text) == CAPTION_MAX, "A caption is at max."),
		invariant.Impossible(
			invariant.Event_True("A caption is at min."),
			invariant.Event_True("A caption is at max."),
		),
	)
}

// OUTLIERS_MIN is the shortest outlier tally, the six-byte "0 (0%)".
const OUTLIERS_MIN = 6

// OUTLIERS_MAX bounds the outlier tally, the widest a large count over a tiny sample reaches.
const OUTLIERS_MAX = 25

// Outliers is the rendered outlier count with its percentage — "0 (0%)", "3 (10%)". A
// distinct type from a line: a tally is always several bytes, never the full row a line is.
type Outliers string

// Outliers_Invariants bounds the tally's length; always several bytes, so the empty, one, and
// two-byte boundaries are unreachable while the min and max are witnessed.
func Outliers_Invariants(text Outliers, namespace invariant.Namespace) {
	invariant.Always(len(text) <= OUTLIERS_MAX, "An outliers tally is at most its max.")
	invariant.Always(len(text) >= OUTLIERS_MIN, "An outliers tally is at least its min.")
	invariant.Always(len(text) != 0, "An outliers tally is never empty.")
	invariant.Always(len(text) != 1, "An outliers tally is never one byte.")
	invariant.Always(len(text) != 2, "An outliers tally is never two bytes.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(text) == OUTLIERS_MIN, "An outliers tally is at min."),
	)
}

// PADDED_MIN is the narrowest padded column, the four-wide delta interval.
const PADDED_MIN = 4

// PADDED_MAX bounds a padded column: the widest unpadded text a column ever holds.
const PADDED_MAX = 25

// Padded is one column widened to its alignment width — the padder's output, between the
// narrowest delta column and the widest value. A distinct type from a whole row.
type Padded string

// Padded_Invariants bounds a padded column's length; always several bytes, so the empty, one,
// and two-byte boundaries are unreachable while the min and max are witnessed.
func Padded_Invariants(text Padded, namespace invariant.Namespace) {
	invariant.Always(len(text) <= PADDED_MAX, "A padded column is at most its max.")
	invariant.Always(len(text) >= PADDED_MIN, "A padded column is at least its min.")
	invariant.Always(len(text) != 0, "A padded column is never empty.")
	invariant.Always(len(text) != 1, "A padded column is never one byte.")
	invariant.Always(len(text) != 2, "A padded column is never two bytes.")
	// The maximum is the widest a column could theoretically hold — every component at its
	// own max at once — which no real row reaches: names, value cells, and outlier tallies
	// never peak together. So it is the Always guard, not a witnessed extreme.
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(text) == PADDED_MIN, "A padded column is at min."),
	)
}

// BARE_ROW_MIN is the narrowest assembled data row, all columns at their minimum.
const BARE_ROW_MIN = 69

// BARE_ROW_MAX bounds an assembled data row before any delta column is appended.
const BARE_ROW_MAX = 89

// Bare_Row is one assembled, aligned table row without its delta — the fixed columns joined.
// A distinct type from the full row, which carries the delta and so runs wider.
type Bare_Row string

// Bare_Row_Invariants bounds an assembled row's length; always many bytes, so the empty, one,
// and two-byte boundaries are unreachable while the min and max are witnessed.
func Bare_Row_Invariants(text Bare_Row, namespace invariant.Namespace) {
	invariant.Always(len(text) <= BARE_ROW_MAX, "A bare row is at most its max.")
	invariant.Always(len(text) >= BARE_ROW_MIN, "A bare row is at least its min.")
	invariant.Always(len(text) != 0, "A bare row is never empty.")
	invariant.Always(len(text) != 1, "A bare row is never one byte.")
	invariant.Always(len(text) != 2, "A bare row is never two bytes.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(text) == BARE_ROW_MIN, "A bare row is at min."),
	)
}

// FULL_ROW_MIN is the narrowest complete metric row, the reference's bare row with no delta.
const FULL_ROW_MIN = 69

// FULL_ROW_MAX bounds a complete metric row, the widest data row plus its delta column.
const FULL_ROW_MAX = 116

// Full_Row is one complete metric row as written to the report — an assembled row plus its
// delta column when the benchmark is a candidate. A distinct type, wider than the bare row.
type Full_Row string

// Full_Row_Invariants bounds a complete row's length; always many bytes, so the empty, one,
// and two-byte boundaries are unreachable while the min and max are witnessed.
func Full_Row_Invariants(text Full_Row, namespace invariant.Namespace) {
	invariant.Always(len(text) <= FULL_ROW_MAX, "A full row is at most its max.")
	invariant.Always(len(text) >= FULL_ROW_MIN, "A full row is at least its min.")
	invariant.Always(len(text) != 0, "A full row is never empty.")
	invariant.Always(len(text) != 1, "A full row is never one byte.")
	invariant.Always(len(text) != 2, "A full row is never two bytes.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(text) == FULL_ROW_MIN, "A full row is at min."),
	)
}

// DELTA_BODY_MIN is the narrowest delta body, a sign over two padded percentages.
const DELTA_BODY_MIN = 16

// DELTA_BODY_MAX bounds the unpainted delta body, both percentages at their widest.
const DELTA_BODY_MAX = 25

// Delta_Body is the uncolored delta text — a sign, a percentage, its confidence interval —
// before any ANSI color wraps it. A distinct type from the painted form, which runs wider.
type Delta_Body string

// Delta_Body_Invariants bounds the body's length; always many bytes, so the empty, one, and
// two-byte boundaries are unreachable while the min and max are witnessed.
func Delta_Body_Invariants(text Delta_Body, namespace invariant.Namespace) {
	invariant.Always(len(text) <= DELTA_BODY_MAX, "A delta body is at most its max.")
	invariant.Always(len(text) >= DELTA_BODY_MIN, "A delta body is at least its min.")
	invariant.Always(len(text) != 0, "A delta body is never empty.")
	invariant.Always(len(text) != 1, "A delta body is never one byte.")
	invariant.Always(len(text) != 2, "A delta body is never two bytes.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(text) == DELTA_BODY_MIN, "A delta body is at min."),
	)
}

// DELTA_TEXT_MIN is the narrowest rendered delta, an uncolored body.
const DELTA_TEXT_MIN = 16

// DELTA_TEXT_MAX bounds the rendered delta: the widest body wrapped in a color code and its
// reset.
const DELTA_TEXT_MAX = 34

// Delta_Text is the rendered delta column as written — the body, optionally wrapped in an ANSI
// color and its reset. A distinct type, wider than the bare body by the escape sequences.
type Delta_Text string

// Delta_Text_Invariants bounds the rendered delta's length; always many bytes, so the empty,
// one, and two-byte boundaries are unreachable while the min and max are witnessed.
func Delta_Text_Invariants(text Delta_Text, namespace invariant.Namespace) {
	invariant.Always(len(text) <= DELTA_TEXT_MAX, "A delta text is at most its max.")
	invariant.Always(len(text) >= DELTA_TEXT_MIN, "A delta text is at least its min.")
	invariant.Always(len(text) != 0, "A delta text is never empty.")
	invariant.Always(len(text) != 1, "A delta text is never one byte.")
	invariant.Always(len(text) != 2, "A delta text is never two bytes.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(text) == DELTA_TEXT_MIN, "A delta text is at min."),
	)
}

// FREQUENCY_BYTES_MIN is the shortest rendered frequency: the one-byte "?" placeholder.
const FREQUENCY_BYTES_MIN = 1

// FREQUENCY_BYTES_MAX bounds a rendered frequency's length, the widest "N.NN GHz" form.
const FREQUENCY_BYTES_MAX = 1 << 3

// Frequency is a rendered CPU frequency — "?", "3.60 GHz", "800 MHz". A distinct type:
// a frequency is one byte or at least five, never two, so it carries its own invariant.
type Frequency string

// Frequency_Invariants bounds a frequency's length; a frequency is "?" or a number with
// a unit, so the empty and two-byte boundaries are unreachable while the min and max witness.
func Frequency_Invariants(text Frequency, namespace invariant.Namespace) {
	invariant.Always(len(text) <= FREQUENCY_BYTES_MAX, "A frequency is within its max length.")
	invariant.Always(len(text) >= FREQUENCY_BYTES_MIN, "A frequency clears its min length.")
	invariant.Always(len(text) != 0, "A frequency is never empty.")
	invariant.Always(len(text) != 2, "A frequency is never two bytes.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(text) == 1, "A frequency is one byte."),
		invariant.Sometimes(len(text) == FREQUENCY_BYTES_MIN, "A frequency is at min."),
		invariant.Sometimes(len(text) == FREQUENCY_BYTES_MAX, "A frequency is at max."),
		invariant.Impossible(
			invariant.Event_True("A frequency is one byte."),
			invariant.Event_False("A frequency is at min."),
		),
		invariant.Impossible(
			invariant.Event_False("A frequency is one byte."),
			invariant.Event_True("A frequency is at min."),
		),
		invariant.Impossible(
			invariant.Event_True("A frequency is one byte."),
			invariant.Event_True("A frequency is at max."),
		),
		invariant.Impossible(
			invariant.Event_True("A frequency is at min."),
			invariant.Event_True("A frequency is at max."),
		),
	)
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
func Suffix_Invariants(unit Suffix, namespace invariant.Namespace) {
	invariant.Always(len(unit) <= SUFFIX_BYTES_MAX, "A suffix is at most its max length.")
	invariant.Always(len(unit) >= SUFFIX_BYTES_MIN, "A suffix is at least its min length.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(unit) == 0, "A suffix is empty."),
		invariant.Sometimes(len(unit) == 1, "A suffix is one byte."),
		invariant.Sometimes(len(unit) == 2, "A suffix is two bytes."),
		invariant.Sometimes(len(unit) == SUFFIX_BYTES_MAX, "A suffix is max bytes."),
		invariant.Sometimes(len(unit) == SUFFIX_BYTES_MIN, "A suffix is at min."),
		invariant.Impossible(
			invariant.Event_True("A suffix is empty."),
			invariant.Event_False("A suffix is at min."),
		),
		invariant.Impossible(
			invariant.Event_False("A suffix is empty."),
			invariant.Event_True("A suffix is at min."),
		),
		invariant.Impossible(
			invariant.Event_True("A suffix is at min."),
			invariant.Event_True("A suffix is one byte."),
		),
		invariant.Impossible(
			invariant.Event_True("A suffix is at min."),
			invariant.Event_True("A suffix is two bytes."),
		),
		invariant.Impossible(
			invariant.Event_True("A suffix is at min."),
			invariant.Event_True("A suffix is max bytes."),
		),
		invariant.Impossible(
			invariant.Event_True("A suffix is empty."),
			invariant.Event_True("A suffix is one byte."),
		),
		invariant.Impossible(
			invariant.Event_True("A suffix is empty."),
			invariant.Event_True("A suffix is two bytes."),
		),
		invariant.Impossible(
			invariant.Event_True("A suffix is empty."),
			invariant.Event_True("A suffix is max bytes."),
		),
		invariant.Impossible(
			invariant.Event_True("A suffix is one byte."),
			invariant.Event_True("A suffix is two bytes."),
		),
		invariant.Impossible(
			invariant.Event_True("A suffix is one byte."),
			invariant.Event_True("A suffix is max bytes."),
		),
		invariant.Impossible(
			invariant.Event_True("A suffix is two bytes."),
			invariant.Event_True("A suffix is max bytes."),
		),
		invariant.Impossible(
			invariant.Event_False("A suffix is empty."),
			invariant.Event_False("A suffix is one byte."),
			invariant.Event_False("A suffix is two bytes."),
			invariant.Event_False("A suffix is max bytes."),
		),
	)
}

// PHASE_BYTES_MIN is the empty sampling phase a phase word floors at.
const PHASE_BYTES_MIN = 0

// PHASE_BYTES_MAX is the longest a progress phase is: the six-byte "warmup".
const PHASE_BYTES_MAX = 6

// Phase is a progress-line phase word — "" while sampling, "warmup" while warming up. A
// distinct type so the phase word carries a length invariant.
type Phase string

// Phase_Invariants bounds a phase's length; a phase is empty or six bytes, so the one-
// and two-byte boundaries are unreachable while the empty min and the max are witnessed.
func Phase_Invariants(name Phase, namespace invariant.Namespace) {
	invariant.Always(len(name) <= PHASE_BYTES_MAX, "A phase is at most its max length.")
	invariant.Always(len(name) >= PHASE_BYTES_MIN, "A phase is at least its min length.")
	invariant.Always(len(name) != 1, "A phase is never one byte.")
	invariant.Always(len(name) != 2, "A phase is never two bytes.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(name) == 0, "A phase is empty."),
		invariant.Sometimes(len(name) == PHASE_BYTES_MAX, "A phase is max bytes."),
		invariant.Sometimes(len(name) == PHASE_BYTES_MIN, "A phase is at min."),
		invariant.Impossible(
			invariant.Event_True("A phase is empty."),
			invariant.Event_False("A phase is at min."),
		),
		invariant.Impossible(
			invariant.Event_False("A phase is empty."),
			invariant.Event_True("A phase is at min."),
		),
		invariant.Impossible(
			invariant.Event_True("A phase is at min."),
			invariant.Event_True("A phase is max bytes."),
		),
		invariant.Impossible(
			invariant.Event_True("A phase is empty."),
			invariant.Event_True("A phase is max bytes."),
		),
		invariant.Impossible(
			invariant.Event_False("A phase is empty."),
			invariant.Event_False("A phase is max bytes."),
		),
	)
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
	invariant.Always(value <= EXTENT_MAX, "An extent is at most its max.")
	invariant.Always(value >= EXTENT_MIN, "An extent is at least its min.")
	invariant.Always(value != 0, "An extent is never zero.")
	invariant.Always(value != 1, "An extent is never one.")
	invariant.Always(value != 2, "An extent is never two.")
	invariant.Always(value != -1, "An extent is never negative one.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(value == EXTENT_MIN, "An extent is at min."),
		invariant.Sometimes(value == EXTENT_MAX, "An extent is at max."),
		invariant.Impossible(
			invariant.Event_True("An extent is at min."),
			invariant.Event_True("An extent is at max."),
		),
	)
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
	invariant.Always(value <= KEPT_MAX, "A kept count is at most its max.")
	invariant.Always(value >= KEPT_MIN, "A kept count is at least its min.")
	invariant.Always(value != -1, "A kept count is never negative one.")
	invariant.Always(value != 0, "A kept count is never zero.")
	invariant.Always(value != 1, "A kept count is never one.")
	invariant.Always(value != 2, "A kept count is never two.")
	// The max is the sampling ceiling — reachable only by running the full ten thousand
	// samples, an arbitrary safety cap, not a meaningful count to witness. It is the
	// Always guard above, not a demanded extreme, so the quorum alone is witnessed.
	invariant.Dot_Product(namespace,
		invariant.Sometimes(value == KEPT_MIN, "A kept count is at its quorum."),
	)
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
	invariant.Always(value <= DEGREE_MAX, "A degree is at most its max.")
	invariant.Always(value >= DEGREE_MIN, "A degree is at least its min.")
	invariant.Always(value != 0, "A degree is never zero.")
	invariant.Always(value != -1, "A degree is never negative one.")
	invariant.Always(value != 1, "A degree is never one.")
	invariant.Always(value != 2, "A degree is never two.")
	// The max degree comes only from a full ten-thousand-sample run — the arbitrary cap,
	// guarded by the Always above, not a meaningful degree to witness.
	invariant.Dot_Product(namespace,
		invariant.Sometimes(value == DEGREE_MIN, "A degree is at min."),
	)
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
	invariant.Always(value <= CENSUS_MAX, "A census is at most its max.")
	invariant.Always(value >= CENSUS_MIN, "A census is at least its min.")
	invariant.Always(value != 0, "A census is never zero.")
	invariant.Always(value != -1, "A census is never negative one.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(value == 1, "A census is one."),
		invariant.Sometimes(value == 2, "A census is two."),
		invariant.Sometimes(value == CENSUS_MIN, "A census is at min."),
		invariant.Impossible(
			invariant.Event_True("A census is one."),
			invariant.Event_False("A census is at min."),
		),
		invariant.Impossible(
			invariant.Event_False("A census is one."),
			invariant.Event_True("A census is at min."),
		),
		invariant.Impossible(
			invariant.Event_True("A census is one."),
			invariant.Event_True("A census is two."),
		),
		invariant.Impossible(
			invariant.Event_True("A census is two."),
			invariant.Event_True("A census is at min."),
		),
	)
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
	invariant.Always(value <= DIVISOR_MAX, "A divisor is at most its max.")
	invariant.Always(value >= DIVISOR_MIN, "A divisor is at least its min.")
	invariant.Always(value != 0, "A divisor is never zero.")
	invariant.Always(value != -1, "A divisor is never negative one.")
	invariant.Always(value != 1, "A divisor is never one.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(value == 2, "A divisor is two."),
	)
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
	invariant.Always(value <= TALLY_MAX, "A tally is at most its max.")
	invariant.Always(value >= TALLY_MIN, "A tally is at least its min.")
	invariant.Always(value != -1, "A tally is never negative one.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(value == 0, "A tally is zero."),
		invariant.Sometimes(value == 1, "A tally is one."),
		invariant.Sometimes(value == 2, "A tally is two."),
		invariant.Sometimes(value == TALLY_MIN, "A tally is at min."),
		invariant.Impossible(
			invariant.Event_True("A tally is zero."),
			invariant.Event_False("A tally is at min."),
		),
		invariant.Impossible(
			invariant.Event_False("A tally is zero."),
			invariant.Event_True("A tally is at min."),
		),
		invariant.Impossible(
			invariant.Event_True("A tally is zero."),
			invariant.Event_True("A tally is one."),
		),
		invariant.Impossible(
			invariant.Event_True("A tally is zero."),
			invariant.Event_True("A tally is two."),
		),
		invariant.Impossible(
			invariant.Event_True("A tally is at min."),
			invariant.Event_True("A tally is one."),
		),
		invariant.Impossible(
			invariant.Event_True("A tally is at min."),
			invariant.Event_True("A tally is two."),
		),
		invariant.Impossible(
			invariant.Event_True("A tally is one."),
			invariant.Event_True("A tally is two."),
		),
	)
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
	invariant.Always(value <= STRAYS_MAX, "A strays count is at most its max.")
	invariant.Always(value >= STRAYS_MIN, "A strays count is at least its min.")
	invariant.Always(value != -1, "A strays count is never negative one.")
	// The outer-half maximum is a theoretical ceiling — reaching it needs a distribution
	// whose spikes stay beyond fences the same spikes would widen — so it is the Always
	// guard, not a witnessed extreme. A real run's outlier count is small.
	invariant.Dot_Product(namespace,
		invariant.Sometimes(value == 0, "A strays count is zero."),
		invariant.Sometimes(value == 1, "A strays count is one."),
		invariant.Sometimes(value == 2, "A strays count is two."),
		invariant.Impossible(
			invariant.Event_True("A strays count is zero."),
			invariant.Event_True("A strays count is one."),
		),
		invariant.Impossible(
			invariant.Event_True("A strays count is zero."),
			invariant.Event_True("A strays count is two."),
		),
		invariant.Impossible(
			invariant.Event_True("A strays count is one."),
			invariant.Event_True("A strays count is two."),
		),
	)
}

// Index_min is the first position a zero-based index floors at: zero.
const POSITION_MIN = 0

// Index_max bounds a benchmark or command index: the last position in a full report, one
// less than the structural cap.
const POSITION_MAX = STRUCTURE_MAX - 1

// Position is a zero-based index into the benchmarks or commands — a distinct type from a
// tally, bounded by the structural cap rather than the run cap.
type Position int

// Position_Invariants bounds a position; it is never negative, with the zero min, the last
// slot max, and the one- and two-position shapes between witnessed.
func Position_Invariants(value Position, namespace invariant.Namespace) {
	invariant.Always(value <= POSITION_MAX, "An index is at most its max.")
	invariant.Always(value >= POSITION_MIN, "An index is at least its min.")
	invariant.Always(value != -1, "An index is never negative one.")
	// The last-slot maximum needs a full report — more benchmarks than fit under the
	// render ceiling, or more metrics than exist — so it is the Always guard, not a
	// witnessed extreme. A real index is small: a handful of benchmarks, nine metrics.
	invariant.Dot_Product(namespace,
		invariant.Sometimes(value == 0, "An index is zero."),
		invariant.Sometimes(value == 1, "An index is one."),
		invariant.Sometimes(value == 2, "An index is two."),
		invariant.Impossible(
			invariant.Event_True("An index is zero."),
			invariant.Event_True("An index is one."),
		),
		invariant.Impossible(
			invariant.Event_True("An index is zero."),
			invariant.Event_True("An index is two."),
		),
		invariant.Impossible(
			invariant.Event_True("An index is one."),
			invariant.Event_True("An index is two."),
		),
	)
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
func Exit_Code_Invariants(value Exit_Code, namespace invariant.Namespace) {
	invariant.Always(value <= EXIT_CODE_MAX, "An exit code is at most its max.")
	invariant.Always(value >= EXIT_CODE_MIN, "An exit code is at least its min.")
	invariant.Always(value != 2, "An exit code is never two.")
	invariant.Always(value != -1, "An exit code is never negative one.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(value == 0, "An exit code is success."),
		invariant.Sometimes(value == 1, "An exit code is failure."),
		invariant.Sometimes(value == EXIT_CODE_MIN, "An exit code is at min."),
		invariant.Sometimes(value == EXIT_CODE_MAX, "An exit code is at max."),
		invariant.Impossible(
			invariant.Event_True("An exit code is success."),
			invariant.Event_False("An exit code is at min."),
		),
		invariant.Impossible(
			invariant.Event_False("An exit code is success."),
			invariant.Event_True("An exit code is at min."),
		),
		invariant.Impossible(
			invariant.Event_True("An exit code is failure."),
			invariant.Event_False("An exit code is at max."),
		),
		invariant.Impossible(
			invariant.Event_False("An exit code is failure."),
			invariant.Event_True("An exit code is at max."),
		),
		invariant.Impossible(
			invariant.Event_True("An exit code is success."),
			invariant.Event_True("An exit code is failure."),
		),
		invariant.Impossible(
			invariant.Event_False("An exit code is success."),
			invariant.Event_False("An exit code is failure."),
		),
	)
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
	invariant.Always(value <= EXIT_STATUS_MAX, "A command status is at most its max.")
	invariant.Always(value >= EXIT_STATUS_MIN, "A command status is at least its min.")
	invariant.Always(value != -1, "A command status is never negative one.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(value == 0, "A command status is zero."),
		invariant.Sometimes(value == 1, "A command status is one."),
		invariant.Sometimes(value == 2, "A command status is two."),
		invariant.Sometimes(value == EXIT_STATUS_MIN, "A command status is at min."),
		invariant.Sometimes(value == EXIT_STATUS_MAX, "A command status is at max."),
		invariant.Impossible(
			invariant.Event_True("A command status is zero."),
			invariant.Event_False("A command status is at min."),
		),
		invariant.Impossible(
			invariant.Event_False("A command status is zero."),
			invariant.Event_True("A command status is at min."),
		),
		invariant.Impossible(
			invariant.Event_True("A command status is zero."),
			invariant.Event_True("A command status is one."),
		),
		invariant.Impossible(
			invariant.Event_True("A command status is zero."),
			invariant.Event_True("A command status is two."),
		),
		invariant.Impossible(
			invariant.Event_True("A command status is zero."),
			invariant.Event_True("A command status is at max."),
		),
		invariant.Impossible(
			invariant.Event_True("A command status is at min."),
			invariant.Event_True("A command status is one."),
		),
		invariant.Impossible(
			invariant.Event_True("A command status is at min."),
			invariant.Event_True("A command status is two."),
		),
		invariant.Impossible(
			invariant.Event_True("A command status is at min."),
			invariant.Event_True("A command status is at max."),
		),
		invariant.Impossible(
			invariant.Event_True("A command status is one."),
			invariant.Event_True("A command status is two."),
		),
		invariant.Impossible(
			invariant.Event_True("A command status is one."),
			invariant.Event_True("A command status is at max."),
		),
		invariant.Impossible(
			invariant.Event_True("A command status is two."),
			invariant.Event_True("A command status is at max."),
		),
	)
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
	invariant.Always(value <= FAILURE_STATUS_MAX, "A failure status is at most its max.")
	invariant.Always(value >= FAILURE_STATUS_MIN, "A failure status is at least its min.")
	invariant.Always(value != 0, "A failure status is never zero.")
	invariant.Always(value != -1, "A failure status is never negative one.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(value == 1, "A failure status is one."),
		invariant.Sometimes(value == 2, "A failure status is two."),
		invariant.Sometimes(value == FAILURE_STATUS_MAX, "A failure status is at max."),
		invariant.Impossible(
			invariant.Event_True("A failure status is one."),
			invariant.Event_True("A failure status is two."),
		),
		invariant.Impossible(
			invariant.Event_True("A failure status is one."),
			invariant.Event_True("A failure status is at max."),
		),
		invariant.Impossible(
			invariant.Event_True("A failure status is two."),
			invariant.Event_True("A failure status is at max."),
		),
	)
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
	invariant.Always(value <= CORES_COUNT_MAX, "A core count is at most its max.")
	invariant.Always(value >= CORES_COUNT_MIN, "A core count is at least its min.")
	invariant.Always(value != -1, "A core count is never negative one.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(value == 0, "A core count is zero."),
		invariant.Sometimes(value == 1, "A core count is one."),
		invariant.Sometimes(value == 2, "A core count is two."),
		invariant.Sometimes(value == CORES_COUNT_MAX, "A core count is at max."),
		invariant.Impossible(
			invariant.Event_True("A core count is zero."),
			invariant.Event_True("A core count is one."),
		),
		invariant.Impossible(
			invariant.Event_True("A core count is zero."),
			invariant.Event_True("A core count is two."),
		),
		invariant.Impossible(
			invariant.Event_True("A core count is zero."),
			invariant.Event_True("A core count is at max."),
		),
		invariant.Impossible(
			invariant.Event_True("A core count is one."),
			invariant.Event_True("A core count is two."),
		),
		invariant.Impossible(
			invariant.Event_True("A core count is one."),
			invariant.Event_True("A core count is at max."),
		),
		invariant.Impossible(
			invariant.Event_True("A core count is two."),
			invariant.Event_True("A core count is at max."),
		),
	)
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
	invariant.Always(value <= HERTZ_MAX, "A frequency value is at most its max.")
	invariant.Always(value >= HERTZ_MIN, "A frequency value is at least its min.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(value == 0, "A frequency value is zero."),
		invariant.Sometimes(value == 1, "A frequency value is one."),
		invariant.Sometimes(value == 2, "A frequency value is two."),
		invariant.Sometimes(value == HERTZ_MAX, "A frequency value is at max."),
		invariant.Impossible(
			invariant.Event_True("A frequency value is zero."),
			invariant.Event_True("A frequency value is one."),
		),
		invariant.Impossible(
			invariant.Event_True("A frequency value is zero."),
			invariant.Event_True("A frequency value is two."),
		),
		invariant.Impossible(
			invariant.Event_True("A frequency value is zero."),
			invariant.Event_True("A frequency value is at max."),
		),
		invariant.Impossible(
			invariant.Event_True("A frequency value is one."),
			invariant.Event_True("A frequency value is two."),
		),
		invariant.Impossible(
			invariant.Event_True("A frequency value is one."),
			invariant.Event_True("A frequency value is at max."),
		),
		invariant.Impossible(
			invariant.Event_True("A frequency value is two."),
			invariant.Event_True("A frequency value is at max."),
		),
	)
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
	invariant.Always(value <= BYTE_SIZE_MAX, "A byte size is at most its max.")
	invariant.Always(value >= BYTE_SIZE_MIN, "A byte size is at least its min.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(value == 0, "A byte size is zero."),
		invariant.Sometimes(value == 1, "A byte size is one."),
		invariant.Sometimes(value == 2, "A byte size is two."),
		invariant.Sometimes(value == BYTE_SIZE_MAX, "A byte size is at max."),
		invariant.Impossible(
			invariant.Event_True("A byte size is zero."),
			invariant.Event_True("A byte size is one."),
		),
		invariant.Impossible(
			invariant.Event_True("A byte size is zero."),
			invariant.Event_True("A byte size is two."),
		),
		invariant.Impossible(
			invariant.Event_True("A byte size is zero."),
			invariant.Event_True("A byte size is at max."),
		),
		invariant.Impossible(
			invariant.Event_True("A byte size is one."),
			invariant.Event_True("A byte size is two."),
		),
		invariant.Impossible(
			invariant.Event_True("A byte size is one."),
			invariant.Event_True("A byte size is at max."),
		),
		invariant.Impossible(
			invariant.Event_True("A byte size is two."),
			invariant.Event_True("A byte size is at max."),
		),
	)
}

// GAP_MIN is the smallest a deviation magnitude is: zero, when a value equals the mean.
const GAP_MIN = 0

// GAP_MAX bounds a deviation magnitude to the representable metric ceiling: a gap is the
// distance between a value and the mean, both non-negative metrics below 2^43, so it fits
// well inside a signed word. It is only the Always guard, not a witnessed extreme — the
// exact maximum is never reached (a value at the ceiling pulls the mean up with it), so
// demanding it would force a crafted distribution rather than a benchmark.
const GAP_MAX = 1<<43 - 1

// Gap is the magnitude of one value's deviation from the mean, the unsigned the 128-bit
// accumulator squares. Bounded by the representable metric span, so its bundle witnesses the
// small shapes a real distribution drives it through, not the word ceiling.
type Gap uint64

// Gap_Invariants bounds a gap to the representable range and witnesses the small shapes; the
// ceiling and the zero floor are the Always guards, not Sometimes claims to allocate.
func Gap_Invariants(value Gap, namespace invariant.Namespace) {
	invariant.Always(value <= GAP_MAX, "A gap is at most its max.")
	invariant.Always(value >= GAP_MIN, "A gap is at least its min.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(value == 0, "A gap is zero."),
		invariant.Sometimes(value == 1, "A gap is one."),
		invariant.Sometimes(value == 2, "A gap is two."),
		invariant.Impossible(
			invariant.Event_True("A gap is zero."),
			invariant.Event_True("A gap is one."),
		),
		invariant.Impossible(
			invariant.Event_True("A gap is zero."),
			invariant.Event_True("A gap is two."),
		),
		invariant.Impossible(
			invariant.Event_True("A gap is one."),
			invariant.Event_True("A gap is two."),
		),
	)
}

// ACCUMULATOR_MIN is the smallest a 128-bit accumulator word is: zero, an empty sum.
const ACCUMULATOR_MIN = 0

// ACCUMULATOR_MAX is the width ceiling of a 128-bit accumulator word — the all-ones
// uint64. It is only the Always guard's safety bound, never a value to witness: the sum
// of squared deviations, bounded by maddox's sample count and per-value gap, never
// saturates a word (the high half uses a few dozen bits), and the low half is a modular
// residue whose all-ones value only a crafted congruence would hit. Neither is a domain
// boundary, so the bundle witnesses the small shapes a real sum reaches, not the ceiling.
const ACCUMULATOR_MAX = 1<<64 - 1

// Accumulator is one 64-bit half of the 128-bit sum-of-squares accumulator. A distinct
// type so its bundle witnesses only the shapes a real distribution drives the register
// through — zero (an all-equal sample set), one, two — and leaves the word ceiling as an
// unwitnessed Always guard, since a computational register has no domain maximum.
type Accumulator uint64

// Accumulator_Invariants bounds an accumulator word to the word width and witnesses the
// small shapes a real sum reaches. The ceiling and floor are the Always guards, not
// Sometimes claims: an internal register is not a domain quantity with a boundary to
// allocate, and demanding the all-ones word forces a crafted congruence, not a benchmark.
func Accumulator_Invariants(value Accumulator, namespace invariant.Namespace) {
	invariant.Always(value <= ACCUMULATOR_MAX, "An accumulator is at most its max.")
	invariant.Always(value >= ACCUMULATOR_MIN, "An accumulator is at least its min.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(value == 0, "An accumulator is zero."),
		invariant.Sometimes(value == 1, "An accumulator is one."),
		invariant.Sometimes(value == 2, "An accumulator is two."),
		invariant.Impossible(
			invariant.Event_True("An accumulator is zero."),
			invariant.Event_True("An accumulator is one."),
		),
		invariant.Impossible(
			invariant.Event_True("An accumulator is zero."),
			invariant.Event_True("An accumulator is two."),
		),
		invariant.Impossible(
			invariant.Event_True("An accumulator is one."),
			invariant.Event_True("An accumulator is two."),
		),
	)
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
	invariant.Always(value <= METRIC_MAX, "A metric is at most its max.")
	invariant.Always(value >= METRIC_MIN, "A metric is at least its min.")
	invariant.Always(value != -1, "A metric is never negative one.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(value == 0, "A metric is zero."),
		invariant.Sometimes(value == 1, "A metric is one."),
		invariant.Sometimes(value == 2, "A metric is two."),
		invariant.Sometimes(value == METRIC_MIN, "A metric is its min."),
		invariant.Sometimes(value == METRIC_MAX, "A metric is its max."),
		invariant.Impossible(
			invariant.Event_True("A metric is zero."),
			invariant.Event_False("A metric is its min."),
		),
		invariant.Impossible(
			invariant.Event_False("A metric is zero."),
			invariant.Event_True("A metric is its min."),
		),
		invariant.Impossible(
			invariant.Event_True("A metric is zero."),
			invariant.Event_True("A metric is one."),
		),
		invariant.Impossible(
			invariant.Event_True("A metric is zero."),
			invariant.Event_True("A metric is two."),
		),
		invariant.Impossible(
			invariant.Event_True("A metric is zero."),
			invariant.Event_True("A metric is its max."),
		),
		invariant.Impossible(
			invariant.Event_True("A metric is its min."),
			invariant.Event_True("A metric is one."),
		),
		invariant.Impossible(
			invariant.Event_True("A metric is its min."),
			invariant.Event_True("A metric is two."),
		),
		invariant.Impossible(
			invariant.Event_True("A metric is its min."),
			invariant.Event_True("A metric is its max."),
		),
		invariant.Impossible(
			invariant.Event_True("A metric is one."),
			invariant.Event_True("A metric is two."),
		),
		invariant.Impossible(
			invariant.Event_True("A metric is one."),
			invariant.Event_True("A metric is its max."),
		),
		invariant.Impossible(
			invariant.Event_True("A metric is two."),
			invariant.Event_True("A metric is its max."),
		),
	)
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
	invariant.Always(value <= AVERAGE_MAX, "An average is at most its max.")
	invariant.Always(value >= AVERAGE_MIN, "An average is at least its min.")
	invariant.Always(value != -1, "An average is never negative one.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(value == 0, "zero"),
		invariant.Sometimes(value == 1, "one"),
		invariant.Sometimes(value == 2, "two"),
		invariant.Sometimes(value == AVERAGE_MAX, "max"),
		invariant.Impossible(invariant.Event_True("zero"), invariant.Event_True("one")),
		invariant.Impossible(invariant.Event_True("zero"), invariant.Event_True("two")),
		invariant.Impossible(invariant.Event_True("zero"), invariant.Event_True("max")),
		invariant.Impossible(invariant.Event_True("one"), invariant.Event_True("two")),
		invariant.Impossible(invariant.Event_True("one"), invariant.Event_True("max")),
		invariant.Impossible(invariant.Event_True("two"), invariant.Event_True("max")),
	)
}

// POINTS_MIN is the single-element run the distribution floors at — at least one sample.
const POINTS_MIN = 1

// POINTS_MAX bounds a sorted run's length: the per-command sample cap, since the run is the
// kept samples sorted.
const POINTS_MAX = SAMPLES_MAX

// Points is a non-empty ascending run of metric values an outlier scan walks. A distinct
// type: the distribution always has at least one sample, so its bundle claims the empty
// run as never reached rather than the nil-or-empty distinction the slice preset demands.
type Points []int64

// Points_Invariants bounds a sorted run's length; the run is never empty, with the
// single-element min, the cap max, and the two-element shape witnessed.
func Points_Invariants(values Points, namespace invariant.Namespace) {
	invariant.Always(len(values) <= POINTS_MAX, "A points run is at most its max length.")
	invariant.Always(len(values) >= POINTS_MIN, "A points run is at least its min length.")
	// The run is a kept distribution's sorted samples, which the 3-run quorum makes at
	// least three — never empty, one, or two — so those lengths are guarded, not witnessed.
	invariant.Always(len(values) != 0, "A points run is never empty.")
	invariant.Always(len(values) != 1, "A points run is never one element.")
	invariant.Always(len(values) != 2, "A points run is never two elements.")
	// The max length is the sampling ceiling — ten thousand kept samples, an arbitrary
	// safety cap reached only by grinding a full run, not a meaningful length to witness.
	// The Always guard holds it; only the quorum length is witnessed.
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(values) == QUORUM_MIN, "A points run is at its quorum."),
	)
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
func Label_Invariants(text Label, namespace invariant.Namespace) {
	invariant.Always(len(text) <= LABEL_BYTES_MAX, "A label is at most its max length.")
	invariant.Always(len(text) >= LABEL_BYTES_MIN, "A label is at least its min length.")
	// The label names the command being benchmarked, so it always carries an executable
	// — never empty. The multibyte ceiling (fifty four-byte runes) is a guard, not a
	// witnessed extreme; a real command's words are short.
	invariant.Always(len(text) != 0, "A label is never empty.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(text) == 1, "A label is one byte."),
		invariant.Sometimes(len(text) == 2, "A label is two bytes."),
		invariant.Impossible(
			invariant.Event_True("A label is one byte."),
			invariant.Event_True("A label is two bytes."),
		),
	)
}

// Pad aligns text to a visible width with spaces — on the left when right is set, so the
// text right-aligns, on the right otherwise. The width is the rune count, so a multibyte
// glyph like σ still counts as one column. One padder, so every column width passes one site
// and the width and padded-line invariants see the whole layout's range, not a per-side slice.
func pad(text Column, width Extent, right bool) (result Padded) {
	defer func() { Padded_Invariants(result, "pad.padded") }()
	Column_Invariants(text, "pad.text")
	Extent_Invariants(width, "pad.width")
	invariant.Boolean_Invariants(right, "pad.right")
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
func delta_render(delta Delta, color bool) (text Delta_Text) {
	defer func() { Delta_Text_Invariants(text, "delta_render.text") }()
	Delta_Invariants(delta, "delta_render.delta")
	invariant.Boolean_Invariants(color, "delta_render.color")
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
	difference := delta.Diff_Percent
	if difference < 0 {
		difference = -difference
	}
	half_percent := delta.Half_Percent
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

// Scale_Step_Invariants states a rung's suffix; the divisor has no preset of its own.
func Scale_Step_Invariants(step Scale_Step, namespace invariant.Namespace) {
	Suffix_Invariants(step.Suffix, "Scale_Step.Suffix")
}

// Format_quantity renders a raw value scaled to a human unit with three significant
// figures — poop's printUnit, e.g. 14906807 nanoseconds becomes "14.9ms".
func format_quantity(value fixedpoint.Number, unit Unit) (text Cell) {
	defer func() { Cell_Invariants(text, "format_quantity.text") }()
	Unit_Invariants(unit, "format_quantity.unit")
	scaled, unit_suffix := scale_quantity(value, unit)
	return Cell(string(format_significant(scaled)) + string(unit_suffix))
}

// Scale_quantity divides a value down to its human magnitude and names the unit
// suffix, dispatching on the metric's unit.
func scale_quantity(
	value fixedpoint.Number, unit Unit,
) (scaled fixedpoint.Number, unit_suffix Suffix) {
	defer func() { Suffix_Invariants(unit_suffix, "scale_quantity.suffix") }()
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
	value fixedpoint.Number, ladder Ladder,
) (scaled fixedpoint.Number, unit_suffix Suffix) {
	defer func() { Suffix_Invariants(unit_suffix, "scale_ladder.suffix") }()
	Ladder_Invariants(ladder, "scale_ladder.ladder")
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
		Ladder_Invariants(ladder, "time_ladder.ladder")
		for _, x := range ladder {
			Scale_Step_Invariants(x, "time_ladder.step")
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
		Ladder_Invariants(ladder, "byte_ladder.ladder")
		for _, x := range ladder {
			Scale_Step_Invariants(x, "byte_ladder.step")
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
		Ladder_Invariants(ladder, "count_ladder.ladder")
		for _, x := range ladder {
			Scale_Step_Invariants(x, "count_ladder.step")
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
	defer func() { Glyph_Invariants(text, "format_significant.text") }()
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
func paint(text Delta_Body, code Ansi_Code, color bool) (painted Delta_Text) {
	defer func() { Delta_Text_Invariants(painted, "paint.painted") }()
	Delta_Body_Invariants(text, "paint.text")
	Ansi_Code_Invariants(code, "paint.code")
	invariant.Boolean_Invariants(color, "paint.color")
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
	Total int
}

// Render_Progress_Input_Invariants states the update's coverable fields; the command
// and elapsed duration have no preset of their own.
func Render_Progress_Input_Invariants(input Render_Progress_Input, namespace invariant.Namespace) {
	Phase_Invariants(input.Phase, "Render_Progress_Input.Phase")
	Census_Invariants(input.Count, "Render_Progress_Input.Count")
	invariant.Int_Invariants(input.Total, "Render_Progress_Input.Total")
}

// Render_progress writes an in-place progress line to stderr: elapsed seconds, an
// optional phase word (warmup), the counter (count over its total, or just the count
// when the total is disabled), and the command. Gated by Progress at the call site.
func render_progress(stderr io.Writer, input *Render_Progress_Input) {
	Render_Progress_Input_Invariants(*input, "render_progress.input")
	seconds := fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
		Numerator: int64(input.Elapsed), Denominator: int64(time.SECOND),
	})
	counter := strconv.Itoa(int(input.Count))
	if input.Total > 0 {
		counter = counter + "/" + strconv.Itoa(input.Total)
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
