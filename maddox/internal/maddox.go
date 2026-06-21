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

	"github.com/james-orcales/james-orcales/shared/fixedpoint"
	"github.com/james-orcales/james-orcales/shared/sh"
	"github.com/james-orcales/james-orcales/shared/time"
)

// Exit_success is the status Main returns when every command was benchmarked and
// the report was written.
const exit_success = 0

// Exit_failure is the status Main returns when a command failed or the report could
// not be written.
const exit_failure = 1

// Runs_min is the smallest number of samples a command is run, so a spent budget
// still leaves a quorum for the statistics — poop's min_samples.
const runs_min = 3

// Samples_max caps the samples held for one command, bounding memory against a
// command fast enough to run unboundedly within the budget — poop's MAX_SAMPLES.
const samples_max = 10000

// Output_Format selects how Main renders the report.
type Output_Format uint8

// Output_Format_Table renders the human-readable comparison table; the default.
const Output_Format_Table Output_Format = 0

// Output_Format_Json renders the machine-readable JSON document.
const Output_Format_Json Output_Format = 1

// Sample is one run's measurements.
type Sample struct {
	// Wall is the run's elapsed time, filled by Main from the injected clock.
	Wall time.Duration
	// RSS_Bytes_Max is the run's peak physical memory footprint, in bytes.
	RSS_Bytes_Max int64
	// CPU_Cycles is the run's CPU cycle count from the hardware counters.
	CPU_Cycles uint64
	// Instructions is the run's retired-instruction count from the counters.
	Instructions uint64
	// Cache_References is the run's last-level cache reference count (Linux only).
	Cache_References uint64
	// Cache_Misses is the run's last-level cache miss count (Linux only).
	Cache_Misses uint64
	// Branch_Misses is the run's mispredicted-branch count (Linux only).
	Branch_Misses uint64
	// CPU_User is the run's user-space CPU time.
	CPU_User time.Duration
	// CPU_System is the run's kernel-space CPU time.
	CPU_System time.Duration
}

// Run_Result is everything one measured run reports: its Sample, the exit code,
// and the stderr captured for a failing run.
type Run_Result struct {
	// Sample is the run's measurements.
	Sample Sample
	// Exit is the command's exit code; non-zero is a failure.
	Exit int
	// Stderr is the command's captured stderr, surfaced on failure.
	Stderr []byte
}

// Sampler is the one capability Main reaches the world through: run a command once
// and report what happened.
type Sampler struct {
	// Measure runs the command once and reports the Run_Result.
	Measure func(command sh.Command) (result Run_Result)
}

// Machine_Specs is a snapshot of the host hardware and OS taken once at startup,
// carried in the report so benchmark results are reproducible across machines.
type Machine_Specs struct {
	// CPU_Model is the CPU's brand string, e.g. "Apple M4 Max".
	CPU_Model string `json:"cpu_model"`
	// CPU_Arch is the instruction-set architecture, e.g. "arm64".
	CPU_Arch string `json:"cpu_arch"`
	// Physical_Cores is the total physical core count across all performance levels.
	Physical_Cores int `json:"physical_cores"`
	// Logical_Cores is the OS-visible thread count, which may exceed Physical_Cores
	// when hyperthreading or SMT is active.
	Logical_Cores int `json:"logical_cores"`
	// Performance_Cores is the P-core count on hybrid CPUs (Apple Silicon, Alder Lake+).
	// Zero when the CPU does not expose a performance/efficiency split.
	Performance_Cores int `json:"performance_cores,omitempty"`
	// Efficiency_Cores is the E-core count on hybrid CPUs.
	Efficiency_Cores int `json:"efficiency_cores,omitempty"`
	// CPU_Frequency_Hz_Max is the maximum rated CPU frequency in Hz; zero when the
	// kernel does not expose it (e.g. Apple Silicon with no cpufrequency_max sysctl).
	CPU_Frequency_Hz_Max uint64 `json:"cpu_frequency_hz_max,omitempty"`
	// Cache_L1_Bytes is the per-core L1 data cache size in bytes.
	Cache_L1_Bytes uint64 `json:"cache_l1_bytes,omitempty"`
	// Cache_L2_Bytes is the per-core L2 cache size in bytes.
	Cache_L2_Bytes uint64 `json:"cache_l2_bytes,omitempty"`
	// Cache_L3_Bytes is the shared L3 cache size in bytes.
	Cache_L3_Bytes uint64 `json:"cache_l3_bytes,omitempty"`
	// RAM_Total_Bytes is the total installed physical memory in bytes.
	RAM_Total_Bytes uint64 `json:"ram_total_bytes"`
	// Storage_Total_Bytes is the total capacity of the boot filesystem in bytes. It
	// is a benchmark-relevant proxy for SSD throughput: on Apple Silicon a larger
	// drive spreads I/O across more NAND dies, so the 512GB model reads and writes
	// faster than the 256GB even on identical silicon.
	Storage_Total_Bytes uint64 `json:"storage_total_bytes"`
	// Operating_System_Name is the operating-system name, e.g. "macOS".
	Operating_System_Name string `json:"operating_system_name"`
	// Operating_System_Version is the OS release, e.g. "15.2".
	Operating_System_Version string `json:"operating_system_version"`
	// Kernel_Version is the kernel release string, e.g. "Darwin 25.2.0".
	Kernel_Version string `json:"kernel_version"`
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
	Outlier_Count int `json:"outliers"`
	// Sample_Count is how many values the distribution was computed from.
	Sample_Count int `json:"count"`
	// Unit names the unit the raw values are in.
	Unit string `json:"unit"`
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

// Benchmark is one command's entry in the report.
type Benchmark struct {
	// Command is the command's words, as a reader recognizes them.
	Command []string `json:"command"`
	// Runs is how many samples were kept, warmup excluded.
	Runs int `json:"runs"`
	// Elapsed is the total wall time of the kept runs.
	Elapsed time.Duration `json:"elapsed_ns"`
	// Measurements is the per-metric distribution.
	Measurements Measurements `json:"measurements"`
	// Deltas is the change against the reference; nil for the reference itself.
	Deltas *Deltas `json:"deltas,omitempty"`
}

// Document is the whole report: one Benchmark per command, in the order given,
// preceded by the host machine specs.
type Document struct {
	// Machine is the host hardware and OS snapshot, taken once at startup.
	Machine Machine_Specs `json:"machine"`
	// Benchmarks is one entry per command, in invocation order.
	Benchmarks []Benchmark `json:"benchmarks"`
}

// Main_Input carries the injected dependencies Main needs, so the library tier
// spawns nothing and reads no ambient clock.
type Main_Input struct {
	// Commands are the commands to benchmark; the first is the reference.
	Commands []sh.Command
	// Clock times each run; production wires the OS clock, tests a virtual one.
	Clock time.Clock
	// Sampler runs and measures one command; production wires the cgo measurer.
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

// Main benchmarks each command in turn — the binary's one entry point — and writes
// the JSON report to Output. The first command is the reference the rest report
// deltas against, matching poop. A command that exits non-zero aborts the run with
// exit_failure, its stderr surfaced, unless Allow_Failures is set.
func Main(input *Main_Input) (exit_code int) {
	benchmarks := make([]Benchmark, 0, len(input.Commands))
	reference := Measurements{}
	have_reference := false
	for index, command := range input.Commands {
		samples, run_exit, child_stderr := main_input_collect_samples(input, command)
		// Erase the progress line on both paths — before the failure message below or
		// before the report that prints once every command is sampled.
		if input.Progress {
			input.Stderr.Write([]byte(progress_clear))
		}
		if run_exit != 0 {
			write_failure(&write_failure_input{
				Stderr:       input.Stderr,
				Index:        index,
				Exit:         run_exit,
				Child_Stderr: child_stderr,
			})
			return exit_failure
		}
		measurements := measurements_compute(samples)
		benchmark := Benchmark{
			Command:      command_words(command),
			Runs:         len(samples),
			Elapsed:      samples_elapsed(samples),
			Measurements: measurements,
		}
		if have_reference {
			deltas := deltas_compute(&deltas_compute_input{
				Reference: reference,
				Candidate: measurements,
			})
			benchmark.Deltas = &deltas
		} else {
			reference = measurements
			have_reference = true
		}
		benchmarks = append(benchmarks, benchmark)
	}
	document := Document{Machine: input.Machine, Benchmarks: benchmarks}
	if input.Format == Output_Format_Json {
		return write_report(input.Output, document)
	}
	return write_table(input.Output, &Render_Table_Input{
		Document: document,
		Color:    input.Color,
	})
}

// Main_input_collect_samples runs command through the warmup discards and then the
// measured loop, timing each kept run with the injected clock. A non-zero exit
// returns that exit and its stderr to abort the run; with Allow_Failures the run is
// kept and a zero exit is returned so sampling continues.
func main_input_collect_samples(
	input *Main_Input, command sh.Command,
) (samples []Sample, exit int, stderr []byte) {
	warmup_start := input.Clock.Now_Monotonic()
	warmups := 0
	for warmups < input.Warmup_Count {
		warm := input.Sampler.Measure(command)
		if warm.Exit != 0 {
			if !input.Allow_Failures {
				return nil, warm.Exit, warm.Stderr
			}
		}
		warmups++
		if input.Progress {
			render_progress(input.Stderr, &render_progress_input{
				Command: command,
				Elapsed: time.Duration(input.Clock.Now_Monotonic() - warmup_start),
				Phase:   "warmup",
				Count:   warmups,
				Total:   input.Warmup_Count,
			})
		}
	}

	start := input.Clock.Now_Monotonic()
	samples = make([]Sample, 0)
	for sampling_should_continue(&sampling_should_continue_input{
		Clock:        input.Clock,
		Start:        start,
		Duration_Max: input.Duration_Max,
		Runs_Max:     input.Runs_Max,
		Count:        len(samples),
	}) {
		run_start := input.Clock.Now_Monotonic()
		result := input.Sampler.Measure(command)
		if result.Exit != 0 {
			if !input.Allow_Failures {
				return nil, result.Exit, result.Stderr
			}
		}
		sample := result.Sample
		sample.Wall = time.Duration(input.Clock.Now_Monotonic() - run_start)
		samples = append(samples, sample)
		if input.Progress {
			render_progress(input.Stderr, &render_progress_input{
				Command: command,
				Elapsed: time.Duration(input.Clock.Now_Monotonic() - start),
				Count:   len(samples),
				Total:   input.Runs_Max,
			})
		}
	}
	return samples, 0, nil
}

// Sampling_should_continue_input is the loop state sampling_should_continue judges.
type sampling_should_continue_input struct {
	Clock        time.Clock
	Start        time.Moment
	Duration_Max time.Duration
	Runs_Max     int
	Count        int
}

// Sampling_should_continue decides whether to take another sample. The 3-run minimum
// always wins first and the 10000-run cap always stops; between them, sampling stops
// when any active limit is met — the run cap or the time budget — and a limit of zero
// is inactive, so both zero leaves only the safety cap. The compound condition is
// split into nested single-term ifs for the linter.
func sampling_should_continue(input *sampling_should_continue_input) (yes bool) {
	if input.Count < runs_min {
		return true
	}
	if input.Count >= samples_max {
		return false
	}
	if input.Runs_Max > 0 {
		if input.Count >= input.Runs_Max {
			return false
		}
	}
	if input.Duration_Max > 0 {
		elapsed := time.Duration(input.Clock.Now_Monotonic() - input.Start)
		if elapsed >= input.Duration_Max {
			return false
		}
	}
	return true
}

// Samples_elapsed sums the wall time of the kept runs — how long the command's
// measured sampling took in total.
func samples_elapsed(samples []Sample) (elapsed time.Duration) {
	for _, sample := range samples {
		elapsed += sample.Wall
	}
	return elapsed
}

// Measurements_compute reduces the samples to one Measurement per metric, tagging
// each with the unit its raw values are in.
func measurements_compute(samples []Sample) (measurements Measurements) {
	measurements.Wall_Time = Measurement_Compute(extract(samples, sample_wall), "nanoseconds")
	measurements.Peak_RSS = Measurement_Compute(extract(samples, sample_rss), "bytes")
	measurements.CPU_Cycles = Measurement_Compute(extract(samples, sample_cycles), "count")
	measurements.Instructions = Measurement_Compute(
		extract(samples, sample_instructions), "count")
	measurements.Cache_References = Measurement_Compute(
		extract(samples, sample_cache_references), "count")
	measurements.Cache_Misses = Measurement_Compute(
		extract(samples, sample_cache_misses), "count")
	measurements.Branch_Misses = Measurement_Compute(
		extract(samples, sample_branch_misses), "count")
	measurements.CPU_User = Measurement_Compute(extract(samples, sample_user), "nanoseconds")
	measurements.CPU_System = Measurement_Compute(
		extract(samples, sample_system), "nanoseconds")
	return measurements
}

// Deltas_compute_input pairs a reference and candidate distribution for comparison.
type deltas_compute_input struct {
	Reference Measurements
	Candidate Measurements
}

// Deltas_compute compares every metric of the candidate against the reference.
func deltas_compute(input *deltas_compute_input) (deltas Deltas) {
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
func extract(samples []Sample, selector func(sample Sample) (value int64)) (values []int64) {
	values = make([]int64, len(samples))
	for index, sample := range samples {
		values[index] = selector(sample)
	}
	return values
}

// Sample_wall reads a sample's wall time as an int64.
func sample_wall(sample Sample) (value int64) { return int64(sample.Wall) }

// Sample_rss reads a sample's peak resident size as an int64.
func sample_rss(sample Sample) (value int64) { return sample.RSS_Bytes_Max }

// Sample_cycles reads a sample's CPU cycle count as an int64.
func sample_cycles(sample Sample) (value int64) { return int64(sample.CPU_Cycles) }

// Sample_instructions reads a sample's retired-instruction count as an int64.
func sample_instructions(sample Sample) (value int64) { return int64(sample.Instructions) }

// Sample_cache_references reads a sample's cache-reference count as an int64.
func sample_cache_references(sample Sample) (value int64) {
	return int64(sample.Cache_References)
}

// Sample_cache_misses reads a sample's cache-miss count as an int64.
func sample_cache_misses(sample Sample) (value int64) {
	return int64(sample.Cache_Misses)
}

// Sample_branch_misses reads a sample's branch-miss count as an int64.
func sample_branch_misses(sample Sample) (value int64) {
	return int64(sample.Branch_Misses)
}

// Sample_user reads a sample's user CPU time as an int64.
func sample_user(sample Sample) (value int64) { return int64(sample.CPU_User) }

// Sample_system reads a sample's system CPU time as an int64.
func sample_system(sample Sample) (value int64) { return int64(sample.CPU_System) }

// Command_words flattens a command back to the words a reader recognizes:
// environment assignments, the executable, then its arguments.
func command_words(command sh.Command) (words []string) {
	words = make([]string, 0, len(command.Environment)+1+len(command.Arguments))
	words = append(words, command.Environment...)
	words = append(words, command.Path)
	words = append(words, command.Arguments...)
	return words
}

// Write_report marshals the document to indented JSON and writes it to output,
// returning exit_failure if marshaling or writing fails.
func write_report(output io.Writer, document Document) (exit_code int) {
	payload, marshal_err := json.MarshalIndent(document, "", "  ")
	if marshal_err != nil {
		return exit_failure
	}
	payload = append(payload, '\n')
	_, write_err := output.Write(payload)
	if write_err != nil {
		return exit_failure
	}
	return exit_success
}

// Write_failure_input carries what a benchmarked command's failure is reported from.
type write_failure_input struct {
	Stderr       io.Writer
	Index        int
	Exit         int
	Child_Stderr []byte
}

// Write_failure reports a benchmarked command's non-zero exit to the diagnostic
// sink: a one-line header naming the command's position and code, then the
// command's own stderr.
func write_failure(input *write_failure_input) {
	header := "maddox: benchmark " + strconv.Itoa(input.Index+1) +
		" exited " + strconv.Itoa(input.Exit) + "\n"
	input.Stderr.Write([]byte(header))
	input.Stderr.Write(input.Child_Stderr)
}

// Tukey_fence_ratio is poop's 1.5*IQR outlier fence multiplier as a fixed ratio.
const tukey_fence_ratio fixedpoint.Ratio = 3 * fixedpoint.Scale / 2

// Int64_max is the largest signed 64-bit value, used to test whether a 128-bit variance
// still fits a word before the fixed-point square root.
const int64_max = 1<<63 - 1

// Measurement_Compute reduces one metric's per-run values to its distribution. It
// sorts a copy (the caller's slice is left untouched), then takes the mean, the
// sample standard deviation (n-1 denominator, matching poop), the extrema, the
// median, the quartiles by poop's index math, and the Tukey's-fences outlier count.
func Measurement_Compute(values []int64, unit string) (measurement Measurement) {
	count := len(values)
	if count == 0 {
		return measurement
	}
	sorted := make([]int64, count)
	copy(sorted, values)
	slices.Sort(sorted)

	total := int64(0)
	for _, value := range sorted {
		total += value
	}
	mean_integer := total / int64(count)

	// Quartiles by position, exactly as poop indexes them: q3 falls back to the
	// maximum when there are too few points to take the upper quarter.
	q1 := fixedpoint.From_Integer(sorted[count/4])
	q3 := fixedpoint.From_Integer(sorted[count-1])
	if count >= 4 {
		q3 = fixedpoint.From_Integer(sorted[count-count/4])
	}
	margin := fixedpoint.Apply(q3-q1, tukey_fence_ratio)
	mean := fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
		Numerator: total, Denominator: int64(count),
	})

	measurement = Measurement{
		Mean:               mean,
		Standard_Deviation: standard_deviation(sorted, mean_integer, count),
		Min:                fixedpoint.From_Integer(sorted[0]),
		Max:                fixedpoint.From_Integer(sorted[count-1]),
		Median:             fixedpoint.From_Integer(sorted[count/2]),
		Q1:                 q1,
		Q3:                 q3,
		Outlier_Count: outlier_count(&outlier_count_input{
			Sorted: sorted, Low_Fence: q1 - margin, High_Fence: q3 + margin,
		}),
		Sample_Count: count,
		Unit:         unit,
	}
	return measurement
}

// Wide is a 128-bit unsigned accumulator, for a sum of squares that overflows int64.
type wide struct {
	// High is the upper 64 bits.
	High uint64
	// Low is the lower 64 bits.
	Low uint64
}

// Wide_add_square adds value squared into a 128-bit accumulator.
func wide_add_square(accumulator wide, value uint64) (sum wide) {
	product_high, product_low := bits.Mul64(value, value)
	low, carry := bits.Add64(accumulator.Low, product_low, 0)
	high, _ := bits.Add64(accumulator.High, product_high, carry)
	return wide{High: high, Low: low}
}

// Standard_deviation is the sample standard deviation with an n-1 denominator. The sum
// of squared deviations is accumulated in 128 bits so a large metric's deviations cannot
// overflow before the divide and the fixed-point root.
func standard_deviation(sorted []int64, mean int64, count int) (deviation fixedpoint.Number) {
	if count <= 1 {
		return 0
	}
	sum := wide{}
	for _, value := range sorted {
		gap := value - mean
		if gap < 0 {
			gap = -gap
		}
		sum = wide_add_square(sum, uint64(gap))
	}
	return root_of_quotient(&root_of_quotient_input{
		High: sum.High, Low: sum.Low, Denominator: int64(count - 1),
	})
}

// Root_of_quotient_input bundles a 128-bit numerator with its divisor.
type root_of_quotient_input struct {
	// High is the numerator's upper 64 bits.
	High uint64
	// Low is the numerator's lower 64 bits.
	Low uint64
	// Denominator divides the numerator before the root is taken.
	Denominator int64
}

// Root_of_quotient returns the fixed-point square root of a 128-bit numerator over a
// denominator — the shared tail of the sample and pooled deviations. A quotient that fits
// a signed word keeps full fractional precision; a larger one, a multi-second jitter far
// outside maddox's fast-command envelope, falls back to the integer root.
func root_of_quotient(input *root_of_quotient_input) (deviation fixedpoint.Number) {
	if input.Denominator <= 0 {
		return 0
	}
	denominator := uint64(input.Denominator)
	quotient_high := input.High / denominator
	quotient_low, _ := bits.Div64(input.High%denominator, input.Low, denominator)
	fits := quotient_high == 0
	if fits {
		fits = quotient_low <= int64_max
	}
	if fits {
		return fixedpoint.Square_Root_Scaled(int64(quotient_low))
	}
	return fixedpoint.From_Integer(fixedpoint.Integer_Root(&fixedpoint.Integer_Root_Input{
		High: quotient_high, Low: quotient_low,
	}))
}

// Outlier_count_input bundles the sorted values with Tukey's fences.
type outlier_count_input struct {
	// Sorted is the ascending metric values.
	Sorted []int64
	// Low_Fence is q1 minus 1.5*IQR; a value below it is an outlier.
	Low_Fence fixedpoint.Number
	// High_Fence is q3 plus 1.5*IQR; a value above it is an outlier.
	High_Fence fixedpoint.Number
}

// Outlier_count counts the values beyond Tukey's fences.
func outlier_count(input *outlier_count_input) (count int) {
	for _, value := range input.Sorted {
		point := fixedpoint.From_Integer(value)
		if point < input.Low_Fence {
			count++
			continue
		}
		if point > input.High_Fence {
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

// Compare reports how the candidate's mean differs from the reference's, with the
// 95% confidence half-interval from a pooled-variance two-sample t-test — poop's
// ratio computation. A zero or degenerate reference yields a zero, non-significant
// delta rather than a divide-by-zero.
func Compare(input *Compare_Input) (delta Delta) {
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
	delta.Half_Percent = half_interval(&half_interval_input{
		Reference: reference, Candidate: candidate, Degrees: degrees,
	})
	delta.Significant = significant(&significant_input{
		Diff_Percent: delta.Diff_Percent,
		Half_Percent: delta.Half_Percent,
	})
	return delta
}

// Half_interval_input carries the two measurements and the degrees of freedom.
type half_interval_input struct {
	// Reference is the baseline measurement.
	Reference Measurement
	// Candidate is the measurement compared to the reference.
	Candidate Measurement
	// Degrees is the pooled degrees of freedom, n1 + n2 - 2.
	Degrees int
}

// Half_interval is the 95% confidence half-width on Diff_Percent, from a pooled-variance
// two-sample t-test — poop's score*pooled*normalizer*100/mean. The pooled deviation is
// taken relative to the reference mean, folding in that final divide, so the math never
// forms a raw variance — which, for a metric in the billions, overflows.
func half_interval(input *half_interval_input) (half fixedpoint.Number) {
	first := fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
		Numerator: 1, Denominator: int64(input.Candidate.Sample_Count),
	})
	second := fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
		Numerator: 1, Denominator: int64(input.Reference.Sample_Count),
	})
	normalizer := fixedpoint.Square_Root(first + second)
	pooled := pooled_deviation(&pooled_deviation_input{
		Candidate: input.Candidate, Reference: input.Reference, Degrees: input.Degrees,
	})
	score := student_t_score(input.Degrees)
	band := fixedpoint.Multiply(&fixedpoint.Multiply_Input{A: score, B: pooled})
	band = fixedpoint.Multiply(&fixedpoint.Multiply_Input{A: band, B: normalizer})
	return band * 100
}

// Pooled_deviation_input carries the two measurements and the degrees of freedom.
type pooled_deviation_input struct {
	// Candidate is the measurement compared to the reference.
	Candidate Measurement
	// Reference is the baseline measurement.
	Reference Measurement
	// Degrees is the pooled degrees of freedom, n1 + n2 - 2.
	Degrees int
}

// Pooled_deviation is the pooled standard deviation as a fraction of the reference mean:
// the root of the degrees-weighted mean of the two relative variances. Dividing each
// deviation by the mean before squaring keeps every value near one, so a metric in the
// billions and its enormous raw variance never overflow.
func pooled_deviation(input *pooled_deviation_input) (pooled fixedpoint.Number) {
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
	return fixedpoint.Square_Root(weighted / fixedpoint.Number(input.Degrees))
}

// Student_t_score returns the Student-t critical value for 95% confidence at the given
// degrees of freedom as a fixed-point number, falling back to the normal-distribution
// 1.96 past the tabulated range — poop's getStatScore95. The tables hold thousandths so
// they read as the published constants, and From_Ratio puts them on the fixed-point grid.
// The tables are local, not package globals, so the package keeps no mutable state.
func student_t_score(degrees_of_freedom int) (score fixedpoint.Number) {
	table_1to30 := []int64{
		12706, 4303, 3182, 2776, 2571, 2447, 2365, 2306, 2262, 2228,
		2201, 2179, 2160, 2145, 2131, 2120, 2110, 2101, 2093, 2086,
		2080, 2074, 2069, 2064, 2060, 2056, 2052, 2045, 2048, 2042,
	}
	table_10s := []int64{
		2228, 2086, 2042, 2021, 2009, 2000, 1994, 1990, 1987, 1984, 1982, 1980,
	}
	milli := int64(1960)
	if degrees_of_freedom >= 1 {
		if degrees_of_freedom <= 30 {
			milli = table_1to30[degrees_of_freedom-1]
		}
	}
	if degrees_of_freedom > 30 {
		if degrees_of_freedom <= 120 {
			milli = table_10s[degrees_of_freedom/10-1]
		}
	}
	return fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
		Numerator: milli, Denominator: 1000,
	})
}

// Significant_input carries the difference and its confidence half-interval, both as
// fixed-point percentages.
type significant_input struct {
	Diff_Percent fixedpoint.Number
	Half_Percent fixedpoint.Number
}

// Significant decides whether a difference clears poop's ±1% band: the whole
// confidence interval must sit beyond ±1% with a single sign. The && and || of
// poop's check are split into nested single-term ifs to satisfy the linter.
func significant(input *significant_input) (is bool) {
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

// Ansi_code is an ANSI escape sequence; a distinct type so paint can take it
// without colliding with its string text argument under the same-type-param rule.
type ansi_code string

// Ansi_reset clears all set attributes.
const ansi_reset ansi_code = "\x1b[0m"

// Ansi_faint sets faint, for an insignificant delta.
const ansi_faint ansi_code = "\x1b[2m"

// Ansi_bright_green sets bright green, for a significant speedup.
const ansi_bright_green ansi_code = "\x1b[92m"

// Ansi_bright_red sets bright red, for a significant slowdown.
const ansi_bright_red ansi_code = "\x1b[91m"

// Render_Table_Input carries the report and whether to color it.
type Render_Table_Input struct {
	// Document is the report to render.
	Document Document
	// Color enables ANSI color codes.
	Color bool
}

// Render_Table renders the report as poop's aligned, optionally colored comparison
// table — the default human-readable output. The machine specs block is written
// first, before Benchmark 1.
func Render_Table(input *Render_Table_Input) (output []byte) {
	builder := strings.Builder{}
	render_machine_header(&builder, input.Document.Machine)
	for index, benchmark := range input.Document.Benchmarks {
		render_benchmark(&builder, index, benchmark, input.Color)
	}
	return []byte(builder.String())
}

// Render_machine_header writes the host hardware block before the first benchmark.
// Sparse fields (zero values) are omitted, matching the sparse-metric convention
// for table rows.
func render_machine_header(builder *strings.Builder, m Machine_Specs) {
	// An empty CPU model with no architecture means specs were never acquired (e.g.
	// the stub platform); the header is skipped so output stays clean.
	if m.CPU_Model == "" {
		if m.CPU_Arch == "" {
			return
		}
	}
	builder.WriteString("Machine: " + m.CPU_Model + " (" + m.CPU_Arch + ")\n")
	builder.WriteString("  cores: " + machine_specs_cores(m) + "   " +
		"freq: " + format_hz(m.CPU_Frequency_Hz_Max) + "   " +
		"ram: " + format_bytes(m.RAM_Total_Bytes) + "   " +
		"storage: " + format_bytes(m.Storage_Total_Bytes) + "\n")
	cache_line := ""
	if m.Cache_L1_Bytes > 0 {
		cache_line += "L1: " + format_bytes(m.Cache_L1_Bytes) + "   "
	}
	if m.Cache_L2_Bytes > 0 {
		cache_line += "L2: " + format_bytes(m.Cache_L2_Bytes) + "   "
	}
	if m.Cache_L3_Bytes > 0 {
		cache_line += "L3: " + format_bytes(m.Cache_L3_Bytes) + "   "
	}
	operating_system_part := "OS: " + m.Operating_System_Name + " " +
		m.Operating_System_Version + "   kernel: " + m.Kernel_Version
	if cache_line != "" {
		builder.WriteString("  " + cache_line + operating_system_part + "\n")
	} else {
		builder.WriteString("  " + operating_system_part + "\n")
	}
	builder.WriteString("\n")
}

// Machine_specs_cores renders the core topology line, distinguishing P/E cores on
// hybrid CPUs and collapsing to a single count when physical equals logical.
func machine_specs_cores(m Machine_Specs) (text string) {
	if m.Performance_Cores > 0 {
		if m.Efficiency_Cores > 0 {
			return fmt.Sprintf("%d P + %d E = %d logical",
				m.Performance_Cores, m.Efficiency_Cores, m.Logical_Cores)
		}
	}
	if m.Physical_Cores == m.Logical_Cores {
		return fmt.Sprintf("%d", m.Physical_Cores)
	}
	return fmt.Sprintf("%d physical, %d logical", m.Physical_Cores, m.Logical_Cores)
}

// Format_hz renders a frequency in Hz to a human-readable GHz or MHz string.
func format_hz(hz uint64) (text string) {
	if hz == 0 {
		return "?"
	}
	if hz >= 1_000_000_000 {
		gigahertz := fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
			Numerator: int64(hz), Denominator: 1_000_000_000,
		})
		return fixedpoint.Format(gigahertz, 2) + " GHz"
	}
	megahertz := fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
		Numerator: int64(hz), Denominator: 1_000_000,
	})
	return fixedpoint.Format(megahertz, 0) + " MHz"
}

// Format_bytes renders a byte count, drawn from the machine specs as an unsigned
// integer, with the binary suffix ladder.
func format_bytes(value uint64) (text string) {
	return format_quantity(fixedpoint.From_Integer(int64(value)), "bytes")
}

// Write_table renders the table and writes it to output, returning exit_failure if
// the write fails.
func write_table(output io.Writer, input *Render_Table_Input) (exit_code int) {
	_, write_err := output.Write(Render_Table(input))
	if write_err != nil {
		return exit_failure
	}
	return exit_success
}

// Column_name_width is the metric-name column width.
const column_name_width = 12

// Column_value_width is the width of each scaled-quantity column.
const column_value_width = 8

// Column_outliers_width is the outlier-count column width.
const column_outliers_width = 9

// Render_benchmark writes one benchmark's header, its column header, and its metric
// rows. The header and the rows go through render_cells, so labels align with data.
func render_benchmark(builder *strings.Builder, index int, benchmark Benchmark, color bool) {
	elapsed := format_quantity(fixedpoint.From_Integer(int64(benchmark.Elapsed)), "nanoseconds")
	header := fmt.Sprintf("Benchmark %d (%d runs, %s): %s",
		index+1, benchmark.Runs, elapsed, strings.Join(benchmark.Command, " "))
	builder.WriteString(header)
	builder.WriteString("\n")

	columns := render_cells(&render_cells_input{
		Name: "measurement", Mean: "mean", Sigma: "σ",
		Low: "min", High: "max", Outliers: "outliers",
	})
	if benchmark.Deltas != nil {
		columns = columns + "  delta"
	}
	builder.WriteString(columns)
	builder.WriteString("\n")

	render_metric_rows(builder, benchmark, color)
	builder.WriteString("\n")
}

// Render_metric_rows writes one aligned row per metric, in report order.
func render_metric_rows(builder *strings.Builder, benchmark Benchmark, color bool) {
	m := benchmark.Measurements
	d := Deltas{}
	has_delta := benchmark.Deltas != nil
	if has_delta {
		d = *benchmark.Deltas
	}
	rows := []metric_line_input{
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
	for index := range rows {
		// A metric with no data on this platform — every value zero, e.g. the
		// Linux-only cache and branch counters on macOS — is left out of the table.
		if rows[index].Measurement.Max == 0 {
			continue
		}
		rows[index].Has_Delta = has_delta
		rows[index].Color = color
		builder.WriteString(metric_line(&rows[index]))
		builder.WriteString("\n")
	}
}

// Metric_line_input carries everything one metric row is rendered from.
type metric_line_input struct {
	Name        string
	Measurement Measurement
	Delta       Delta
	Has_Delta   bool
	Color       bool
}

// Metric_line renders one metric row: its name, scaled mean ± σ, min … max, outlier
// count, and — when the benchmark is not the reference — its delta.
func metric_line(input *metric_line_input) (line string) {
	measurement := input.Measurement
	unit := measurement.Unit
	outlier_percent := fixedpoint.Number(0)
	if measurement.Sample_Count > 0 {
		outlier_percent = fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
			Numerator:   int64(measurement.Outlier_Count) * 100,
			Denominator: int64(measurement.Sample_Count),
		})
	}
	outliers := strconv.Itoa(measurement.Outlier_Count) +
		" (" + fixedpoint.Format(outlier_percent, 0) + "%)"
	line = render_cells(&render_cells_input{
		Name:     input.Name,
		Mean:     format_quantity(measurement.Mean, unit),
		Sigma:    format_quantity(measurement.Standard_Deviation, unit),
		Low:      format_quantity(measurement.Min, unit),
		High:     format_quantity(measurement.Max, unit),
		Outliers: outliers,
	})
	if input.Has_Delta {
		line = line + "  " + delta_render(input.Delta, input.Color)
	}
	return line
}

// Render_cells_input is one table row's cell texts.
type render_cells_input struct {
	Name     string
	Mean     string
	Sigma    string
	Low      string
	High     string
	Outliers string
}

// Render_cells lays one row — header or data — into aligned, uncolored columns.
// Header and data share this layout, so a label always sits above its column.
func render_cells(input *render_cells_input) (line string) {
	return "  " +
		pad_right(input.Name, column_name_width) + " " +
		pad_left(input.Mean, column_value_width) + " ± " +
		pad_right(input.Sigma, column_value_width) + "  " +
		pad_left(input.Low, column_value_width) + " ... " +
		pad_right(input.High, column_value_width) + "  " +
		pad_left(input.Outliers, column_outliers_width)
}

// Pad_left right-aligns text to a visible width with leading spaces. The width is
// the rune count, so a multibyte glyph like σ still counts as one column.
func pad_left(text string, width int) (padded string) {
	gap := width - utf8.RuneCountInString(text)
	if gap < 0 {
		return text
	}
	return strings.Repeat(" ", gap) + text
}

// Pad_right left-aligns text to a visible width with trailing spaces.
func pad_right(text string, width int) (padded string) {
	gap := width - utf8.RuneCountInString(text)
	if gap < 0 {
		return text
	}
	return text + strings.Repeat(" ", gap)
}

// Delta_render formats one metric's change: a sign, the percentage, and its
// confidence half-interval. A significant change is colored — red slower, green
// faster — while an insignificant one stays faint.
func delta_render(delta Delta, color bool) (text string) {
	sign := "+"
	if delta.Faster {
		sign = "-"
	}
	code := ansi_faint
	if delta.Significant {
		code = ansi_bright_red
		if delta.Faster {
			code = ansi_bright_green
		}
	}
	difference := delta.Diff_Percent
	if difference < 0 {
		difference = -difference
	}
	body := sign + pad_left(fixedpoint.Format(difference, 1), 5) + "% ± " +
		pad_left(fixedpoint.Format(delta.Half_Percent, 1), 4) + "%"
	return paint(body, code, color)
}

// Scale_step is one rung of a scaling ladder: the divisor above which the suffix applies.
type scale_step struct {
	// Divisor is the magnitude the value is divided by at this rung.
	Divisor fixedpoint.Number
	// Suffix names the unit at this rung.
	Suffix string
}

// Format_quantity renders a raw value scaled to a human unit with three significant
// figures — poop's printUnit, e.g. 14906807 nanoseconds becomes "14.9ms".
func format_quantity(value fixedpoint.Number, unit string) (text string) {
	scaled, suffix := scale_quantity(value, unit)
	return format_significant(scaled) + suffix
}

// Scale_quantity divides a value down to its human magnitude and names the unit
// suffix, dispatching on the metric's unit.
func scale_quantity(
	value fixedpoint.Number, unit string,
) (scaled fixedpoint.Number, suffix string) {
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
	value fixedpoint.Number, ladder []scale_step,
) (scaled fixedpoint.Number, suffix string) {
	for _, step := range ladder {
		if value >= step.Divisor {
			scaled = fixedpoint.Divide(&fixedpoint.Divide_Input{
				Dividend: value, Divisor: step.Divisor,
			})
			return scaled, step.Suffix
		}
	}
	return value, ""
}

// Time_ladder is the nanosecond-to-kilosecond ladder; the base unit is ns.
func time_ladder() (ladder []scale_step) {
	return []scale_step{
		{Divisor: fixedpoint.From_Integer(1_000_000_000_000), Suffix: "ks"},
		{Divisor: fixedpoint.From_Integer(1_000_000_000), Suffix: "s"},
		{Divisor: fixedpoint.From_Integer(1_000_000), Suffix: "ms"},
		{Divisor: fixedpoint.From_Integer(1_000), Suffix: "us"},
		{Divisor: fixedpoint.From_Integer(1), Suffix: "ns"},
	}
}

// Byte_ladder is the binary (1024) ladder with IEC suffixes, since memory is a
// power-of-two quantity; the base unit is B.
func byte_ladder() (ladder []scale_step) {
	return []scale_step{
		{Divisor: fixedpoint.From_Integer(1024 * 1024 * 1024 * 1024), Suffix: "TiB"},
		{Divisor: fixedpoint.From_Integer(1024 * 1024 * 1024), Suffix: "GiB"},
		{Divisor: fixedpoint.From_Integer(1024 * 1024), Suffix: "MiB"},
		{Divisor: fixedpoint.From_Integer(1024), Suffix: "KiB"},
		{Divisor: fixedpoint.From_Integer(1), Suffix: "B"},
	}
}

// Count_ladder is the metric-prefix ladder for a bare count; the base unit is unnamed.
func count_ladder() (ladder []scale_step) {
	return []scale_step{
		{Divisor: fixedpoint.From_Integer(1_000_000_000_000), Suffix: "T"},
		{Divisor: fixedpoint.From_Integer(1_000_000_000), Suffix: "G"},
		{Divisor: fixedpoint.From_Integer(1_000_000), Suffix: "M"},
		{Divisor: fixedpoint.From_Integer(1_000), Suffix: "K"},
	}
}

// Format_significant renders a scaled value to three significant figures: whole
// numbers and hundreds with no decimals, tens with one, units with two.
func format_significant(value fixedpoint.Number) (text string) {
	if value >= fixedpoint.From_Integer(1000) {
		return fixedpoint.Format(value, 0)
	}
	if fixedpoint.Is_Integer(value) {
		return fixedpoint.Format(value, 0)
	}
	if value >= fixedpoint.From_Integer(100) {
		return fixedpoint.Format(value, 0)
	}
	if value >= fixedpoint.From_Integer(10) {
		return fixedpoint.Format(value, 1)
	}
	return fixedpoint.Format(value, 2)
}

// Paint wraps text in an ANSI color when color is enabled; the codes have zero
// visible width, so wrapping after padding leaves alignment intact.
func paint(text string, code ansi_code, color bool) (painted string) {
	if !color {
		return text
	}
	return string(code) + text + string(ansi_reset)
}

// Progress_clear returns to the start of the line and erases it, so the next progress
// update — or the report — overwrites the previous progress text cleanly.
const progress_clear = "\r\x1b[K"

// Progress_label_runes_max caps the command text in the progress line so the
// carriage-return update never wraps and strands a stale partial line.
const progress_label_runes_max = 50

// Render_progress_input is one progress update: the command being sampled, how long
// it has been sampling, and how many runs are done against the cap.
type render_progress_input struct {
	Command sh.Command
	Elapsed time.Duration
	Phase   string
	Count   int
	Total   int
}

// Render_progress writes an in-place progress line to stderr: elapsed seconds, an
// optional phase word (warmup), the counter (count over its total, or just the count
// when the total is disabled), and the command. Gated by Progress at the call site.
func render_progress(stderr io.Writer, input *render_progress_input) {
	seconds := fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
		Numerator: int64(input.Elapsed), Denominator: int64(time.Second),
	})
	counter := strconv.Itoa(input.Count)
	if input.Total > 0 {
		counter = counter + "/" + strconv.Itoa(input.Total)
	}
	phase := ""
	if input.Phase != "" {
		phase = input.Phase + " "
	}
	label := progress_label(command_words(input.Command))
	line := progress_clear + fixedpoint.Format(seconds, 1) + "s  " +
		phase + counter + "  " + label
	stderr.Write([]byte(line))
}

// Progress_label joins the command words and truncates them to keep the progress
// line on one terminal row.
func progress_label(words []string) (label string) {
	joined := strings.Join(words, " ")
	runes := []rune(joined)
	if len(runes) <= progress_label_runes_max {
		return joined
	}
	return string(runes[:progress_label_runes_max-1]) + "…"
}
