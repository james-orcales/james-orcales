package maddox_test

import (
	"testing"

	"local/james-orcales/maddox/internal"
	"local/james-orcales/shared/std/math/fixedpoint"
	"local/james-orcales/shared/std/sim/nbio"
	"local/james-orcales/shared/std/sim/time"
)

// The requirements tests mirror REQUIREMENTS.md: one leaf per heading, in heading order, asserting
// only the public surface of the maddox library. Runs go through a scripted loop: every spawn
// retires with the result the test states, and the clock advances by each run's wall.

// Test_Command_Line_Arguments verifies that Main parses injected operating-system arguments into
// commands and sampling limits before it benchmarks.
func Test_Command_Line_Arguments(t *testing.T) {
	subject := harness_new(maddox.Arguments{
		"maddox", "NAME=value echo hi", "-duration=0", "-runs=3", "-warmup=0", "-json",
	}, script_wall(time.MILLISECOND))
	harness_run(t, subject)
	if subject.Runner.Status != maddox.EXIT_SUCCESS {
		t.Fatalf("exit code = %d, want success: %s", subject.Runner.Status,
			harness_error_output(subject))
	}
	if subject.Fake.Calls != 3 {
		t.Fatalf("measurements = %d, want 3", subject.Fake.Calls)
	}
	command := subject.Fake.First
	if command.Path != "echo" {
		t.Fatalf("command path = %q, want echo", command.Path)
	}
	if len(command.Arguments) != 1 {
		t.Fatalf("command arguments = %v, want [hi]", command.Arguments)
	}
	if command.Arguments[0] != "hi" {
		t.Fatalf("command arguments = %v, want [hi]", command.Arguments)
	}
	if len(command.Environment) != 1 {
		t.Fatalf("command environment = %v, want [NAME=value]", command.Environment)
	}
	if command.Environment[0] != "NAME=value" {
		t.Fatalf("command environment = %v, want [NAME=value]", command.Environment)
	}
}

// Test_Statistics_Distribution verifies that Measurement_Compute reduces a set of samples to the
// mean, standard deviation, extrema, median, and quartiles poop reports — checked against a
// hand-computed five-point set.
func Test_Statistics_Distribution(t *testing.T) {
	measurement := maddox.Measurement_Compute([]int64{10, 20, 30, 40, 50}, "count")
	if measurement.Sample_Count != 5 {
		t.Fatalf("sample count = %d, want 5", measurement.Sample_Count)
	}
	if fixedpoint.Number(measurement.Mean) != fixedpoint_number(30) {
		t.Fatalf("mean = %d, want 30.0", measurement.Mean)
	}
	// Sample standard deviation uses n-1 in the denominator: sqrt(1000/4) = 15.811388.
	deviation := fixed_text(fixedpoint.Number(measurement.Standard_Deviation), 3)
	if deviation != "15.811" {
		t.Fatalf("stddev = %s, want 15.811", deviation)
	}
	if fixedpoint.Number(measurement.Min) != fixedpoint_number(10) {
		t.Fatalf("min = %d, want 10.0", measurement.Min)
	}
	if fixedpoint.Number(measurement.Max) != fixedpoint_number(50) {
		t.Fatalf("max = %d, want 50.0", measurement.Max)
	}
	if fixedpoint.Number(measurement.Median) != fixedpoint_number(30) {
		t.Fatalf("median = %d, want 30.0", measurement.Median)
	}
	if fixedpoint.Number(measurement.Q1) != fixedpoint_number(20) {
		t.Fatalf("q1 = %d, want 20.0", measurement.Q1)
	}
	if fixedpoint.Number(measurement.Q3) != fixedpoint_number(50) {
		t.Fatalf("q3 = %d, want 50.0", measurement.Q3)
	}
}

// Test_Statistics_Outliers verifies that Measurement_Compute counts a sample beyond Tukey's fences
// (1.5*IQR past a quartile) as an outlier: nine identical points collapse the IQR to zero, so the
// lone large point falls outside.
func Test_Statistics_Outliers(t *testing.T) {
	measurement := maddox.Measurement_Compute(
		[]int64{10, 10, 10, 10, 10, 10, 10, 10, 10, 1000}, "count")
	if measurement.Outlier_Count != 1 {
		t.Fatalf("outlier count = %d, want 1", measurement.Outlier_Count)
	}
}

// Test_Comparison_Reference verifies that comparing a measurement to itself — the reference
// command against its own numbers — yields a zero delta that is not significant, so the baseline
// never flags itself as a change.
func Test_Comparison_Reference(t *testing.T) {
	reference := maddox.Measurement{
		Mean:               maddox.Mean(fixedpoint_number(100)),
		Standard_Deviation: maddox.Standard_Deviation(fixedpoint_number(5)),
		Sample_Count:       10,
		Unit:               "count",
	}
	delta := maddox.Compare(reference, maddox.Candidate_Measurement(reference))
	if delta.Diff_Percent != 0 {
		t.Fatalf("diff percent = %d, want 0", delta.Diff_Percent)
	}
	if delta.Significant {
		t.Fatal("a measurement compared to itself must not be significant")
	}
	// A real program reports metrics in the billions; a standard deviation past 2^32 must not
	// overflow the pooled-variance arithmetic.
	large := maddox.Measurement{
		Mean:               maddox.Mean(fixedpoint_number(20_000_000_000)),
		Standard_Deviation: maddox.Standard_Deviation(fixedpoint_number(6_000_000_000)),
		Sample_Count:       10,
		Unit:               "count",
	}
	big := maddox.Compare(large, maddox.Candidate_Measurement(large))
	if big.Significant {
		t.Fatal("a large measurement compared to itself must not be significant")
	}
}

// Test_Comparison_Significance verifies that a candidate whose mean is double the reference's,
// with tight variance, is reported as a significant slowdown: the confidence interval clears the
// +1% band, and the candidate is not faster.
func Test_Comparison_Significance(t *testing.T) {
	reference := maddox.Measurement{
		Mean:               maddox.Mean(fixedpoint_number(100)),
		Standard_Deviation: maddox.Standard_Deviation(fixedpoint_number(1)),
		Sample_Count:       20,
		Unit:               "count",
	}
	candidate := maddox.Measurement{
		Mean:               maddox.Mean(fixedpoint_number(200)),
		Standard_Deviation: maddox.Standard_Deviation(fixedpoint_number(1)),
		Sample_Count:       20,
		Unit:               "count",
	}
	delta := maddox.Compare(reference, maddox.Candidate_Measurement(candidate))
	if !delta.Significant {
		t.Fatal("a doubled mean with tight variance must be significant")
	}
	if delta.Faster {
		t.Fatal("a doubled mean is slower, not faster")
	}
	if fixedpoint.Number(delta.Diff_Percent) != fixedpoint_number(100) {
		t.Fatalf("diff percent = %d, want 100.0", delta.Diff_Percent)
	}
}

// Test_Sampling_Budget verifies that Main keeps sampling until the parsed time budget is spent.
// Five 200ms runs consume a one-second budget.
func Test_Sampling_Budget(t *testing.T) {
	subject := harness_new(maddox.Arguments{
		"maddox", "noop", "-duration=1", "-runs=0", "-warmup=0",
	}, script_wall(200*time.MILLISECOND))
	harness_run(t, subject)
	if subject.Runner.Status != maddox.EXIT_SUCCESS {
		t.Fatalf("exit code = %d, want 0", subject.Runner.Status)
	}
	if subject.Fake.Calls != 5 {
		t.Fatalf("runs = %d, want 5 within a 1s budget at 200ms/run", subject.Fake.Calls)
	}
}

// Test_Sampling_Runs verifies that the run cap stops sampling: a cap of five with the time budget
// disabled (zero) yields exactly five runs.
func Test_Sampling_Runs(t *testing.T) {
	subject := harness_new(maddox.Arguments{
		"maddox", "noop", "-duration=0", "-runs=5", "-warmup=0",
	}, script_wall(10*time.MILLISECOND))
	harness_run(t, subject)
	if subject.Fake.Calls != 5 {
		t.Fatalf("runs = %d, want the 5-run cap", subject.Fake.Calls)
	}
}

// Test_Sampling_Minimum verifies that Main runs a command at least three times even when the
// budget is already spent, so the statistics always have a quorum.
func Test_Sampling_Minimum(t *testing.T) {
	subject := harness_new(maddox.Arguments{
		"maddox", "noop", "-duration=1", "-runs=0", "-warmup=0",
	}, script_wall(time.SECOND))
	harness_run(t, subject)
	if subject.Fake.Calls != 3 {
		t.Fatalf("runs = %d, want the 3-run minimum on a spent budget", subject.Fake.Calls)
	}
}

// Test_Sampling_Warmup verifies that Main discards the warmup runs before sampling: two warmup runs
// plus the three-run minimum is five measurements taken, but the report counts only the three
// that were kept.
func Test_Sampling_Warmup(t *testing.T) {
	subject := harness_new(maddox.Arguments{
		"maddox", "noop", "-duration=1", "-runs=0", "-warmup=2", "-json",
	}, script_wall(time.SECOND))
	harness_run(t, subject)
	if subject.Fake.Calls != 5 {
		t.Fatalf("total measure calls = %d, want 5 (2 warmup + 3 kept)", subject.Fake.Calls)
	}
	if !text_contains(harness_output(subject), `"runs": 3,`) {
		t.Fatalf("reported runs must be 3 (warmup excluded), got: %s", harness_output(
			subject))
	}
}

// Test_Output_Document verifies the JSON report shape: one benchmark per command in order, each
// labeled with its command words, the first (reference) compared against nothing and so carrying
// the zero delta.
func Test_Output_Document(t *testing.T) {
	subject := harness_new(maddox.Arguments{
		"maddox", "a", "b", "-duration=0", "-runs=3", "-warmup=0", "-json",
	}, script_wall(time.MILLISECOND))
	harness_run(t, subject)
	output := harness_output(subject)
	first := text_index(output, "\"command\": [\n        \"a\"\n      ]")
	second := text_index(output, "\"command\": [\n        \"b\"\n      ]")
	if first < 0 {
		t.Fatalf("first command missing, got: %s", output)
	}
	if second < first {
		t.Fatalf("second command must follow the first, got: %s", output)
	}
	// The reference is compared against nothing, so its deltas are the zero value.
	reference := output[first:second]
	if text_contains(reference, `"diff_percent": 1`) {
		t.Fatalf("the reference command must carry the zero delta, got: %s", reference)
	}
	if text_contains(reference, `"significant": true`) {
		t.Fatalf("the reference command must carry the zero delta, got: %s", reference)
	}
}

// Test_Output_Failure verifies that a command exiting non-zero aborts the run with a non-zero
// status and the command's captured stderr surfaced to the diagnostic sink, rather than reporting
// numbers for a broken command.
func Test_Output_Failure(t *testing.T) {
	subject := harness_new(maddox.Arguments{
		"maddox", "broken", "-duration=0", "-runs=3", "-warmup=0",
	}, script_failure("boom"))
	harness_run(t, subject)
	if subject.Runner.Status == maddox.EXIT_SUCCESS {
		t.Fatal("a non-zero exit must abort with a non-zero status")
	}
	if !text_contains(harness_error_output(subject), "boom") {
		t.Fatalf("diagnostics = %q, want the command's stderr", harness_error_output(
			subject))
	}
}

// Test_Table_Header verifies the rendered table names each benchmark with its position, kept-run
// count, and command words.
func Test_Table_Header(t *testing.T) {
	benchmark := maddox.Benchmark{
		Command:      maddox.Command_Line{"echo", "hi"},
		Runs:         5,
		Elapsed:      maddox.Elapsed(2 * time.SECOND),
		Measurements: filled_measurements(),
	}
	output := table_render(maddox.Machine_Specs{}, maddox.Color(false), benchmark)
	if !text_contains(output, "Benchmark 1") {
		t.Fatalf("missing benchmark header, got: %s", output)
	}
	if !text_contains(output, "echo hi") {
		t.Fatalf("missing command words, got: %s", output)
	}
	if !text_contains(output, "5 runs") {
		t.Fatalf("missing run count, got: %s", output)
	}
	if !text_contains(output, "2s") {
		t.Fatalf("missing elapsed time, got: %s", output)
	}
}

// Test_Table_Units verifies a raw nanosecond value is scaled to a human unit: a ~14.9ms mean
// renders in milliseconds, not bare nanoseconds.
func Test_Table_Units(t *testing.T) {
	wall_ns := fixedpoint_number(14906807)
	wall := maddox.Measurement{
		Mean: maddox.Mean(wall_ns), Min: maddox.Minimum(wall_ns),
		Max: maddox.Maximum(wall_ns), Median: maddox.Median(wall_ns),
		Q1: maddox.First_Quartile(wall_ns), Q3: maddox.Third_Quartile(wall_ns),
		Sample_Count: 3, Unit: "nanoseconds",
	}
	// 2 MiB exactly: binary scaling renders "2MiB", decimal would be "2.10MB".
	bytes_2mib := fixedpoint_number(2097152)
	memory := maddox.Measurement{
		Mean: maddox.Mean(bytes_2mib), Min: maddox.Minimum(bytes_2mib),
		Max: maddox.Maximum(bytes_2mib), Median: maddox.Median(bytes_2mib),
		Q1: maddox.First_Quartile(bytes_2mib), Q3: maddox.Third_Quartile(bytes_2mib),
		Sample_Count: 3, Unit: "bytes",
	}
	measurements := filled_measurements()
	measurements.Wall_Time = maddox.Wall_Time_Measurement(wall)
	measurements.Peak_RSS = maddox.Peak_Resident_Measurement(memory)
	benchmark := maddox.Benchmark{
		Command: maddox.Command_Line{"x"}, Runs: 3, Measurements: measurements,
	}
	output := table_render(maddox.Machine_Specs{}, maddox.Color(false), benchmark)
	if !text_contains(output, "14.9ms") {
		t.Fatalf("nanoseconds not scaled to ms, got: %s", output)
	}
	if !text_contains(output, "2MiB") {
		t.Fatalf("bytes not scaled to binary MiB, got: %s", output)
	}
}

// Test_Table_Delta verifies a non-reference benchmark's row carries its signed percentage change
// against the reference.
func Test_Table_Delta(t *testing.T) {
	output := table_render(maddox.Machine_Specs{}, maddox.Color(false), delta_benchmarks()...)
	if !text_contains(output, "50.0%") {
		t.Fatalf("missing delta percentage, got: %s", output)
	}
}

// Test_Table_Color verifies ANSI escapes mark the delta only when color is enabled, and never
// otherwise.
func Test_Table_Color(t *testing.T) {
	colored := table_render(maddox.Machine_Specs{}, maddox.Color(true), delta_benchmarks()...)
	plain := table_render(maddox.Machine_Specs{}, maddox.Color(false), delta_benchmarks()...)
	if !text_contains(colored, "\x1b[") {
		t.Fatal("color output lacks ANSI escapes in the delta")
	}
	if text_contains(plain, "\x1b[") {
		t.Fatal("plain output must not contain ANSI escapes")
	}
}

// Test_Table_Sparse verifies that a metric with no data — every value zero, like the Linux-only
// counters on macOS — is omitted from the table, while a metric that has data is shown.
func Test_Table_Sparse(t *testing.T) {
	mean := fixedpoint_number(1_000_000)
	wall := maddox.Measurement{
		Mean: maddox.Mean(mean), Max: maddox.Maximum(mean),
		Sample_Count: 3, Unit: "nanoseconds",
	}
	measurements := filled_measurements()
	measurements.Wall_Time = maddox.Wall_Time_Measurement(wall)
	benchmark := maddox.Benchmark{
		Command: maddox.Command_Line{"x"}, Runs: 3, Measurements: measurements,
	}
	output := table_render(maddox.Machine_Specs{}, maddox.Color(false), benchmark)
	if !text_contains(output, "wall_time") {
		t.Fatalf("a metric with data must render, got: %s", output)
	}
	if text_contains(output, "cache_references") {
		t.Fatalf("an all-zero metric must be omitted, got: %s", output)
	}
}

// Test_Machine_Document verifies the JSON document carries the injected machine specs as a
// top-level "machine" field alongside "benchmarks", with every provided field round-tripping
// through the rendered output.
func Test_Machine_Document(t *testing.T) {
	subject := harness_new(maddox.Arguments{
		"maddox", "noop", "-duration=0", "-runs=3", "-warmup=0", "-json",
	}, script_wall(time.MILLISECOND))
	subject.Runner.Machine = maddox.Machine_Specs{
		CPU_Model:                "TestCPU X1",
		CPU_Arch:                 "arm64",
		Physical_Cores:           8,
		Logical_Cores:            8,
		CPU_Frequency_Hz_Max:     3_600_000_000,
		RAM_Total_Bytes:          16 * 1024 * 1024 * 1024,
		Storage_Total_Bytes:      512_000_000_000,
		Operating_System_Name:    "TestOS",
		Operating_System_Version: "1.0",
		Kernel_Version:           "1.0.0",
	}
	harness_run(t, subject)
	output := harness_output(subject)
	expected := []string{
		`"machine": {`, `"cpu_model": "TestCPU X1",`, `"physical_cores": 8,`,
		`"ram_total_bytes": 17179869184,`, `"storage_total_bytes": 512000000000,`,
		`"benchmarks": [`,
	}
	for _, fragment := range expected {
		if !text_contains(output, fragment) {
			t.Fatalf("document lacks %s, got: %s", fragment, output)
		}
	}
}

// Test_Machine_Table verifies the table output carries a machine header block before the first
// benchmark, naming the CPU model and key specs.
func Test_Machine_Table(t *testing.T) {
	machine := maddox.Machine_Specs{
		CPU_Model:                "TestCPU X1",
		CPU_Arch:                 "arm64",
		Physical_Cores:           8,
		Logical_Cores:            8,
		Performance_Cores:        4,
		Efficiency_Cores:         4,
		CPU_Frequency_Hz_Max:     3_600_000_000,
		Cache_L1_Bytes:           64 * 1024,
		Cache_L2_Bytes:           4 * 1024 * 1024,
		RAM_Total_Bytes:          16 * 1024 * 1024 * 1024,
		Storage_Total_Bytes:      512 * 1024 * 1024 * 1024,
		Operating_System_Name:    "TestOS",
		Operating_System_Version: "1.0",
		Kernel_Version:           "1.0.0",
	}
	benchmark := maddox.Benchmark{
		Command: maddox.Command_Line{"echo"}, Runs: 3, Measurements: filled_measurements(),
	}
	output := table_render(machine, maddox.Color(false), benchmark)
	if !text_contains(output, "TestCPU X1") {
		t.Fatalf("machine header missing CPU model, got: %s", output)
	}
	if !text_contains(output, "512GiB") {
		t.Fatalf("machine header missing storage capacity, got: %s", output)
	}
	// The machine header must appear before the first benchmark.
	if text_index(output, "TestCPU X1") > text_index(output, "Benchmark 1") {
		t.Fatalf("machine header must precede Benchmark 1, got: %s", output)
	}
}

// Test_Progress verifies that, with progress enabled, Main writes the run counter to the
// diagnostic sink while sampling, and writes nothing there when it is disabled.
func Test_Progress(t *testing.T) {
	shown := harness_new(maddox.Arguments{
		"maddox", "noop", "-duration=0", "-runs=5", "-warmup=2", "-progress=always",
	}, script_wall(0))
	harness_run(t, shown)
	if !text_contains(harness_error_output(shown), "warmup 2/2") {
		t.Fatalf("progress should report warmup, got: %q", harness_error_output(shown))
	}
	if !text_contains(harness_error_output(shown), "5/5") {
		t.Fatalf("progress should report the 5/5 counter, got: %q", harness_error_output(
			shown))
	}
	hidden := harness_new(maddox.Arguments{
		"maddox", "noop", "-duration=0", "-runs=5", "-warmup=0", "-progress=never",
	}, script_wall(0))
	harness_run(t, hidden)
	if harness_error_output(hidden) != "" {
		t.Fatalf("progress disabled must write nothing, got: %q", harness_error_output(
			hidden))
	}
}

// HARNESS_PASS_COUNT bounds delivery passes past any run the tests script.
const HARNESS_PASS_COUNT = 1 << 16

// HARNESS_OUTPUT_BYTES holds any report or diagnostic the tests produce.
const HARNESS_OUTPUT_BYTES = 1 << 20

// Scripts one run's outcome.
type script func(call int) (result nbio.Process_Result, err error)

// The fake loop holds the spawn until the harness delivers it, and its clock advances by each
// delivered run's wall.
type fake struct {
	Script     script
	Calls      int
	First      nbio.Process_Request
	Completion *nbio.Completion
	Callback   nbio.Process_Callback
	Now        time.Monotonic_Moment
}

type harness struct {
	Fake         fake
	Runner       maddox.Runner
	Arguments    maddox.Arguments
	Output       nbio.Stream_Memory
	Error_Output nbio.Stream_Memory
}

func harness_new(arguments maddox.Arguments, run script) (h *harness) {
	h = &harness{Arguments: arguments}
	h.Fake.Script = run
	h.Output.Memory = make([]byte, HARNESS_OUTPUT_BYTES)
	h.Error_Output.Memory = make([]byte, HARNESS_OUTPUT_BYTES)
	loop := nbio.IO{}
	loop.Spawn = fake_spawn
	loop.Storage.State = nbio.State(&h.Fake)
	h.Runner = maddox.Runner{
		IO: loop,
		Clock: time.Clock{
			State: time.State(
				&h.Fake), Now_Monotonic: fake_now, Now_Realtime: fake_realtime,
		},
		Output:       nbio.Memory_To_Stream(&h.Output),
		Error_Output: nbio.Memory_To_Stream(&h.Error_Output),
	}
	return h
}

func harness_run(t *testing.T, h *harness) {
	t.Helper()
	h.Runner.Start = maddox.Main(maddox.Main_Input{Arguments: h.Arguments})
	for pass_index := 0; pass_index < HARNESS_PASS_COUNT; pass_index++ {
		for maddox.Rearm(&h.Runner) {
		}
		if maddox.Stopped(&h.Runner) {
			return
		}
		if !fake_deliver(&h.Fake) {
			t.Fatal("the run waits on nothing")
		}
	}
	t.Fatal("the run never stopped")
}

func harness_output(h *harness) (text string) {
	return string(h.Output.Memory[:h.Output.Cursor])
}

func harness_error_output(h *harness) (text string) {
	return string(h.Error_Output.Memory[:h.Error_Output.Cursor])
}

func fake_spawn(
	state nbio.State, completion *nbio.Completion, request nbio.Process_Request,
	_ time.Duration, callback nbio.Process_Callback,
) {
	loop := state.(*fake)
	if loop.Calls == 0 {
		loop.First = request
	}
	loop.Completion = completion
	loop.Callback = callback
}

func fake_deliver(loop *fake) (delivered bool) {
	if loop.Completion == nil {
		return false
	}
	completion := loop.Completion
	loop.Completion = nil
	result, err := loop.Script(loop.Calls)
	loop.Calls++
	loop.Now += time.Monotonic_Moment(result.Usage.Wall)
	loop.Callback(completion, result, err)
	return true
}

func fake_now(state time.State) (moment time.Monotonic_Moment) {
	return state.(*fake).Now
}

func fake_realtime(state time.State) (moment time.Moment) {
	return time.Moment(state.(*fake).Now)
}

// Every run succeeds and costs wall.
func script_wall(wall time.Duration) (run script) {
	return func(_ int) (result nbio.Process_Result, err error) {
		result.Usage.Wall = wall
		result.Usage.Instruction_Count = 100
		return result, nil
	}
}

// Every run exits 1 with stderr as its captured standard error.
func script_failure(stderr string) (run script) {
	return func(_ int) (result nbio.Process_Result, err error) {
		result.Exit = 1
		result.Error_Output = []byte(stderr)
		return result, nil
	}
}

// Converts one whole integer to the general fixed-point arithmetic type.
func fixedpoint_number(value int64) (number fixedpoint.Number) {
	return fixedpoint.Number(fixedpoint.From_Integer(fixedpoint.Whole_Integer(value)))
}

// Renders a fixed-point number with digits decimals.
func fixed_text(value fixedpoint.Number, digits int) (text string) {
	storage := [fixedpoint.TEXT_SIZE_MAXIMUM]byte{}
	count := fixedpoint.Into_Text(storage[:], value, fixedpoint.Digit_Count(digits))
	return string(storage[:count])
}

func text_index(source string, fragment string) (index int) {
	for start_index := 0; start_index+len(fragment) <= len(source); start_index++ {
		if source[start_index:start_index+len(fragment)] == fragment {
			return start_index
		}
	}
	return -1
}

func text_contains(source string, fragment string) (contained bool) {
	return text_index(source, fragment) >= 0
}

// Renders a table: the machine header, then each benchmark in order.
func table_render(
	machine maddox.Machine_Specs, color maddox.Color, benchmarks ...maddox.Benchmark,
) (output string) {
	draft := maddox.Table_Head_Render(machine)
	for index, benchmark := range benchmarks {
		draft = maddox.Table_Benchmark_Render(draft, maddox.Position(index), benchmark,
			color)
	}
	return string(draft.Memory[:draft.Cursor])
}

// Two benchmarks whose candidate carries a significant 50% wall-time change.
func delta_benchmarks() (benchmarks []maddox.Benchmark) {
	mean := fixedpoint_number(1_000_000)
	wall := maddox.Measurement{
		Mean: maddox.Mean(mean), Max: maddox.Maximum(mean),
		Sample_Count: 3, Unit: "nanoseconds",
	}
	deltas := maddox.Deltas{
		Wall_Time: maddox.Wall_Time_Delta(maddox.Delta{
			Diff_Percent: maddox.Difference_Percent(fixedpoint_number(50)),
			Half_Percent: maddox.Half_Percent(fixedpoint_number(2)),
			Significant:  true,
		}),
	}
	measurements := filled_measurements()
	measurements.Wall_Time = maddox.Wall_Time_Measurement(wall)
	return []maddox.Benchmark{
		{Command: maddox.Command_Line{"a"}, Runs: 3, Measurements: measurements},
		{Command: maddox.Command_Line{"b"}, Runs: 3, Measurements: measurements,
			Deltas: deltas},
	}
}

// Filled_measurements is a full set of measured-but-empty metrics, so a hand-built benchmark
// carries what Main always produces: a quorum of all-zero samples, each in its own unit. The
// table omits every one — their maxima are zero — just as it omits unmeasured metrics, but they
// carry the valid sample count and unit a real measurement always has.
func filled_measurements() (measurements maddox.Measurements) {
	span := maddox.Measurement{Sample_Count: 3, Unit: "nanoseconds"}
	size := maddox.Measurement{Sample_Count: 3, Unit: "bytes"}
	count := maddox.Measurement{Sample_Count: 3, Unit: "count"}
	return maddox.Measurements{
		Wall_Time:        maddox.Wall_Time_Measurement(span),
		Peak_RSS:         maddox.Peak_Resident_Measurement(size),
		CPU_Cycles:       maddox.Cycle_Measurement(count),
		Instructions:     maddox.Instruction_Measurement(count),
		Cache_References: maddox.Cache_Reference_Measurement(count),
		Cache_Misses:     maddox.Cache_Miss_Measurement(count),
		Branch_Misses:    maddox.Branch_Miss_Measurement(count),
		CPU_User:         maddox.User_Time_Measurement(span),
		CPU_System:       maddox.System_Time_Measurement(span),
	}
}
