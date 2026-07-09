// Package simulation_test is maddox's blackbox witness: a single fuzz target
// drives internal.Main end to end — sampling, statistics, comparison, and
// rendering — so every production invariant in the maddox library is exercised
// through the one public entry point, never by reaching into a helper. Under a
// plain `go test` the seed corpus is replayed and the invariant recorder judges
// coverage; under `-fuzz` the same driver explores.
package simulation_test

import (
	"io"
	"math"
	"strconv"
	"strings"
	"testing"

	maddox "local/james-orcales/maddox/internal"
	invariant "local/james-orcales/shared/invariant/default"
	sysio "local/james-orcales/shared/io"
	"local/james-orcales/shared/time"
)

// TestMain wires the coverage recorder over the internal tree ("../**"): every
// Always reached must hold and every Sometimes both fire, or the run fails.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m, "../**")
}

// SIM_STRUCTURE_MAX is the command-set and command-line ceiling the driver decodes
// against, mirrored from the library.
const SIM_STRUCTURE_MAX = 1 << 8

// SIM_CAPTURE_MAX is the stderr-capture ceiling.
const SIM_CAPTURE_MAX = 1 << 16

// SIM_HOST_MAX is the host-text field ceiling.
const SIM_HOST_MAX = 1 << 8

// SIM_WORD_MAX is the command-word byte ceiling.
const SIM_WORD_MAX = 1 << 12

// SIM_METRIC_MAX is the representable metric ceiling a real sampler stays within.
const SIM_METRIC_MAX = 1<<43 - 1

// SIM_DELTAS_MAX is the per-sample deviation pattern length.
const SIM_DELTAS_MAX = 16

// SIM_CORES_MAX mirrors the render-safe core-count ceiling.
const SIM_CORES_MAX = 1023

// SIM_HERTZ_MAX mirrors the render-safe frequency ceiling.
const SIM_HERTZ_MAX = 8_000_000_000

// SIM_BYTE_SIZE_MAX mirrors the render-safe hardware byte-size ceiling.
const SIM_BYTE_SIZE_MAX = 1 << 53

// Fuzz_Main is the sole witness. It decodes the fuzz bytes into a benchmark
// scenario and drives internal.Main; the seed corpus is a battery of honest
// end-to-end runs, each reaching a real boundary of the pipeline.
func Fuzz_Main(f *testing.F) {
	for _, seed := range seed_corpus() {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		drive(decode_scenario(data))
	})
}

// Scenario is one benchmark run fully described: the commands, the sampling
// limits, and the per-sample values the injected Sampler will report. Every
// field is decoded totally from the fuzz bytes so any input is a valid run.
type scenario struct {
	// Commands is the command count, 0..SIM_STRUCTURE_MAX.
	Commands int
	// Words is the word count in each command line, 1..SIM_STRUCTURE_MAX.
	Words int
	// Word_Bytes is the byte length of each argument word, 0..SIM_WORD_MAX.
	Word_Bytes int
	// Path_Bytes, for a single-command run, sets the executable's byte length, so the
	// progress label reaches its short shapes; 0 keeps the default distinct path.
	Path_Bytes int
	// Runs is Runs_Max, the full int range (Main caps kept runs internally).
	Runs int
	// Warmup is Warmup_Count, the full int range.
	Warmup int
	// Duration is Duration_Max, in nanoseconds.
	Duration int64
	// Base is the center every metric's samples deviate around.
	Base int64
	// Deltas is the repeating deviation pattern added to Base.
	Deltas []int64
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
	// Machine is the injected host snapshot.
	Machine maddox.Machine_Specs
}

// Broken_writer is a report sink whose every write fails, so write_report and
// write_table reach the failure exit their io.Discard runs never do.
type broken_writer struct{}

func (broken_writer) Write(p []byte) (n int, err error) {
	return 0, io.ErrClosedPipe
}

// Report_sink is the scenario's report writer: a failing one when Broken_Output is set,
// else the discard sink.
func report_sink(s scenario) (sink io.Writer) {
	if s.Broken_Output {
		return broken_writer{}
	}
	return io.Discard
}

// Drive builds the Main_Input a scenario describes and runs it. The Sampler
// synthesizes each run's measurements from the scenario, the clock is virtual,
// and the report is discarded — only the invariants observed along the way matter.
func drive(s scenario) {
	run := 0
	command_index := -1
	previous := ""
	// The sim is the sampler's clock: virtual time accrues by each run's sleep, and the
	// completion stamp reports it so the budget stopwatch advances without a real clock.
	var elapsed_virtual time.Duration
	sampler := maddox.Sampler{
		Measure: func(command sysio.Process_Request) (result maddox.Run_Result) {
			// Commands are measured in sequence, each with a distinct path, so a path
			// change marks the next benchmark and resets the run counter.
			if command.Path != previous {
				previous = command.Path
				command_index++
				run = 0
			}
			value := s.Base + int64(command_index)*s.Divergence
			if len(s.Deltas) > 0 {
				value += s.Deltas[run%len(s.Deltas)]
			}
			result.Sample = sample_from(value)
			// Wall is this run's tight cost, extracted as a Metric into the
			// statistics, so it carries the same representable-ceiling contract as
			// every other metric and passes through the same clamp. The completion
			// stamp is the accrued virtual time for the budget stopwatch, which spans
			// the arbitrary gaps between runs and so stays on the full, unclamped
			// sleep. A run costs its sleep.
			result.Sample.Wall = time.Duration(metric_clamp(s.Sleep))
			elapsed_virtual += time.Duration(s.Sleep)
			result.Completed_At = time.Moment(elapsed_virtual)
			fail := false
			if s.Fail_Command > 0 {
				if command_index == s.Fail_Command-1 {
					fail = true
				}
			}
			if !fail {
				if s.Fail_Every > 0 {
					if run%s.Fail_Every == 0 {
						fail = true
					}
				}
			}
			if fail {
				result.Exit = maddox.Exit_Status(s.Exit)
				result.Stderr = make(maddox.Captured_Output, s.Stderr_Bytes)
			}
			run++
			return result
		},
	}
	maddox.Main(maddox.Main_Input{
		Commands:       command_set(s),
		Sampler:        sampler,
		Duration_Max:   time.Duration(s.Duration),
		Runs_Max:       s.Runs,
		Warmup_Count:   s.Warmup,
		Allow_Failures: s.Allow_Fail,
		Format:         maddox.Output_Format(s.Format),
		Color:          s.Color,
		Progress:       s.Progress,
		Output:         report_sink(s),
		Stderr:         io.Discard,
		Machine:        s.Machine,
	})
}

// Command_set is the scenario's commands, each an executable plus Words-1 argument
// words of Word_Bytes bytes, so the command and command-word length invariants see
// the empty, one-, two-, and full-length shapes.
func command_set(s scenario) (commands maddox.Commands) {
	commands = make(maddox.Commands, s.Commands)
	argument := strings.Repeat("a", s.Word_Bytes)
	for index := range commands {
		// A distinct path per command lets the sampler tell benchmarks apart; the argument
		// words carry the command-word length shapes. A single-command run may set a short
		// executable so the progress label reaches its short shapes.
		path := "cmd" + strconv.Itoa(index)
		if s.Commands == 1 {
			if s.Path_Bytes > 0 {
				path = strings.Repeat("z", s.Path_Bytes)
			}
		}
		command := sysio.Process_Request{Path: path}
		for word_index := 1; word_index < s.Words; word_index++ {
			command.Arguments = append(command.Arguments, argument)
		}
		commands[index] = command
	}
	return commands
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
	if value > SIM_METRIC_MAX {
		return SIM_METRIC_MAX
	}
	return value
}

// Sample_from builds a sample whose every metric is the given value, clamped to the valid
// metric range. A real sampler reports valid metrics: non-negative and within the
// representable ceiling, so the clamp keeps the simulated measurements in contract rather
// than tripping the metric guard, which only an out-of-contract sampler would.
func sample_from(value int64) (sample maddox.Sample) {
	value = metric_clamp(value)
	sample.RSS_Bytes_Max = maddox.Metric(value)
	sample.CPU_Cycles = maddox.Metric(value)
	sample.Instructions = maddox.Metric(value)
	sample.Cache_References = maddox.Metric(value)
	sample.Cache_Misses = maddox.Metric(value)
	sample.Branch_Misses = maddox.Metric(value)
	sample.CPU_User = time.Duration(value)
	sample.CPU_System = time.Duration(value)
	return sample
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
	// A real invocation benchmarks a handful of short commands. Clamp the command and
	// word counts, and budget the total argument bytes, so the rendered report stays
	// under its ceiling — the structural maxima are not reachable through a real run.
	s.Commands = min(int(cursor_u16(c)), 8)
	s.Words = max(1, min(int(cursor_u16(c)), 8))
	budget := 32768 / (s.Commands*s.Words + 1)
	s.Word_Bytes = min(min(int(cursor_u16(c)), SIM_WORD_MAX), budget)
	s.Path_Bytes = min(int(cursor_u16(c)), SIM_WORD_MAX)
	// Runs_Max and Warmup_Count take the full int range: a caller may set any value
	// (Main caps kept runs at samples_max and treats a non-positive limit as disabled).
	s.Runs = int(cursor_i64(c))
	s.Warmup = int(cursor_i64(c))
	s.Duration = int64(cursor_u16(c))
	s.Base = cursor_i64(c)
	deltas := min(int(cursor_u8(c)), SIM_DELTAS_MAX)
	for index := 0; index < deltas; index++ {
		s.Deltas = append(s.Deltas, cursor_i64(c))
	}
	s.Sleep = cursor_i64(c)
	s.Exit = int(cursor_u8(c))
	s.Fail_Every = int(cursor_u8(c))
	s.Fail_Command = int(cursor_u8(c))
	s.Divergence = cursor_i64(c)
	s.Stderr_Bytes = min(int(cursor_u32(c)), SIM_CAPTURE_MAX)
	s.Machine = decode_machine(c)
	return s
}

// Decode_machine reads the host snapshot: five text fields by length, four core
// counts as full-range ints, and six sizes as full-range uint64s — the ranges the
// Machine_Specs invariants witness at their boundaries.
func decode_machine(c *cursor) (m maddox.Machine_Specs) {
	m.CPU_Model = host_text(c)
	m.CPU_Arch = host_text(c)
	m.Operating_System_Name = host_text(c)
	m.Operating_System_Version = host_text(c)
	m.Kernel_Version = host_text(c)
	// A real acquire_machine_specs reports realistic, render-safe values; clamp to the
	// domain ceilings so the simulated host honors that contract rather than tripping the
	// table's render guards, which only an out-of-contract probe would.
	m.Physical_Cores = maddox.Cores(min(int(cursor_u16(c)), SIM_CORES_MAX))
	m.Logical_Cores = maddox.Cores(min(int(cursor_u16(c)), SIM_CORES_MAX))
	m.Performance_Cores = maddox.Cores(min(int(cursor_u16(c)), SIM_CORES_MAX))
	m.Efficiency_Cores = maddox.Cores(min(int(cursor_u16(c)), SIM_CORES_MAX))
	m.CPU_Frequency_Hz_Max = maddox.Hertz(min(cursor_u64(c), SIM_HERTZ_MAX))
	m.Cache_L1_Bytes = maddox.Byte_Size(min(cursor_u64(c), SIM_BYTE_SIZE_MAX))
	m.Cache_L2_Bytes = maddox.Byte_Size(min(cursor_u64(c), SIM_BYTE_SIZE_MAX))
	m.Cache_L3_Bytes = maddox.Byte_Size(min(cursor_u64(c), SIM_BYTE_SIZE_MAX))
	m.RAM_Total_Bytes = maddox.Byte_Size(min(cursor_u64(c), SIM_BYTE_SIZE_MAX))
	m.Storage_Total_Bytes = maddox.Byte_Size(min(cursor_u64(c), SIM_BYTE_SIZE_MAX))
	return m
}

// Host_text is a run of 'x' of the decoded length, clamped to the host-field
// ceiling — the invariants witness the field's length, not its bytes.
func host_text(c *cursor) (text maddox.Host_Text) {
	return maddox.Host_Text(strings.Repeat("x", min(int(cursor_u16(c)), SIM_HOST_MAX)))
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
	writer_u8(w, flags)
	writer_u16(w, uint16(s.Commands))
	writer_u16(w, uint16(s.Words))
	writer_u16(w, uint16(s.Word_Bytes))
	writer_u16(w, uint16(s.Path_Bytes))
	writer_i64(w, int64(s.Runs))
	writer_i64(w, int64(s.Warmup))
	writer_u16(w, uint16(s.Duration))
	writer_i64(w, s.Base)
	writer_u8(w, uint8(len(s.Deltas)))
	for _, delta := range s.Deltas {
		writer_i64(w, delta)
	}
	writer_i64(w, s.Sleep)
	writer_u8(w, uint8(s.Exit))
	writer_u8(w, uint8(s.Fail_Every))
	writer_u8(w, uint8(s.Fail_Command))
	writer_i64(w, s.Divergence)
	writer_u32(w, uint32(s.Stderr_Bytes))
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
	return seeds
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
			s.Divergence = SIM_METRIC_MAX
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

// Seeds_progress exercises the progress counter with varied command shapes, so the
// progress-label path witnesses the one-word, empty, two-byte, and full-length words a
// plain non-progress run renders elsewhere.
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
			s.Sleep = 8
		})
	}
	return [][]byte{
		progress(func(s *scenario) { s.Words = 1 }),
		progress(func(s *scenario) { s.Words = 3 }),
		progress(func(s *scenario) { s.Word_Bytes = 0 }),
		progress(func(s *scenario) { s.Word_Bytes = 2 }),
		progress(func(s *scenario) { s.Word_Bytes = SIM_WORD_MAX }),
		total(1), total(-1), total(math.MinInt64), total(math.MaxInt64),
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
		return func(s *scenario) { full_machine(s); s.Machine.CPU_Frequency_Hz_Max = value }
	}
	return [][]byte{
		with(cores(0)), with(cores(1)), with(cores(2)), with(cores(SIM_CORES_MAX)),
		with(size(0)), with(size(1)), with(size(2)), with(size(SIM_BYTE_SIZE_MAX)),
		// 1023 GiB renders "1023GiB" — a full seven-byte cell, within the representable
		// range the fixed-point render can lift without overflow.
		with(size(1023 * (1 << 30))),
		with(frequency(0)), with(frequency(1)), with(frequency(2)),
		with(frequency(SIM_HERTZ_MAX)),
	}
}

// Set_all_cores sets every core-count field to value.
func set_all_cores(m *maddox.Machine_Specs, value maddox.Cores) {
	m.Physical_Cores = value
	m.Logical_Cores = value
	m.Performance_Cores = value
	m.Efficiency_Cores = value
}

// Set_all_sizes sets every byte-size field to value.
func set_all_sizes(m *maddox.Machine_Specs, value maddox.Byte_Size) {
	m.Cache_L1_Bytes = value
	m.Cache_L2_Bytes = value
	m.Cache_L3_Bytes = value
	m.RAM_Total_Bytes = value
	m.Storage_Total_Bytes = value
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
		with(func(s *scenario) { s.Words = 1 }),
		with(func(s *scenario) { s.Words = 3 }),
		with(func(s *scenario) { s.Word_Bytes = 0 }),
		with(func(s *scenario) { s.Word_Bytes = 2 }),
		with(func(s *scenario) { s.Word_Bytes = SIM_WORD_MAX }),
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
		with(func(s *scenario) { s.Base = SIM_METRIC_MAX; s.Deltas = []int64{0} }),
		with(func(s *scenario) { s.Sleep = 0 }),
		with(func(s *scenario) { s.Sleep = 2 }),
		with(func(s *scenario) { s.Sleep = SIM_METRIC_MAX; s.Runs = 3 }),
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
		with(func(s *scenario) { s.Base = 1000; s.Runs = 10; s.Deltas = spike_deltas() }),
		with(func(s *scenario) { s.Base = 1000; s.Runs = 20; s.Deltas = spike_deltas() }),
		// Three spikes, so the outlier count is an ordinary value past its small shapes.
		with(func(s *scenario) { s.Base = 1000; s.Runs = 30; s.Deltas = spike_deltas() }),
		// A significant slowdown: the second command measures far above the reference with
		// tight variance, so the comparison clears the confidence band.
		with(func(s *scenario) {
			s.Base = 1000
			s.Divergence = 1000
			s.Deltas = []int64{0, 1, -1}
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
		}),
	}
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
		with(fail(SIM_CAPTURE_MAX)),
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
	}
}

// Seeds_configuration feeds the injected config ints across the range Main tolerates.
func seeds_configuration() (seeds [][]byte) {
	return [][]byte{
		with(func(s *scenario) { s.Runs = 1 }),
		// A disabled or absurd run cap witnesses the Runs_Max boundary from the input
		// alone; a one-grain budget stops the loop at the quorum instead of grinding to
		// the sample ceiling, so these seeds cost three runs, not ten thousand.
		with(func(s *scenario) { s.Runs = -1; s.Duration = 1 }),
		with(func(s *scenario) { s.Runs = math.MinInt64; s.Duration = 1 }),
		with(func(s *scenario) { s.Runs = math.MaxInt64; s.Duration = 1 }),
		with(func(s *scenario) { s.Warmup = 1 }),
		with(func(s *scenario) { s.Warmup = -1 }),
		with(func(s *scenario) { s.Warmup = math.MinInt64 }),
		with(func(s *scenario) { s.Warmup = math.MaxInt64 }),
	}
}

// Seeds_machine drives the host-text field lengths to their boundaries; the numeric
// host fields are explored by the fuzzer within their render-safe realistic ranges.
func seeds_machine() (seeds [][]byte) {
	host := func(count int) (mutate func(s *scenario)) {
		return func(s *scenario) { json(s); set_hosts(&s.Machine, count) }
	}
	return [][]byte{
		with(host(0)), with(host(1)), with(host(2)), with(host(3)),
		with(host(SIM_HOST_MAX)),
	}
}

// Set_hosts sets every host-text field to a run of 'x' of the given byte count.
func set_hosts(m *maddox.Machine_Specs, count int) {
	text := maddox.Host_Text(strings.Repeat("x", count))
	m.CPU_Model = text
	m.CPU_Arch = text
	m.Operating_System_Name = text
	m.Operating_System_Version = text
	m.Kernel_Version = text
}
