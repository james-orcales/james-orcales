package maddox_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/james-orcales/james-orcales/maddox/internal"
	"github.com/james-orcales/james-orcales/shared/fixedpoint"
	"github.com/james-orcales/james-orcales/shared/sh"
	"github.com/james-orcales/james-orcales/shared/time"
)

// The specification tests mirror SPECIFICATION.md: one leaf per heading, in
// heading order, asserting only the public surface of the maddox library.

// Test_Statistics_Distribution verifies that Measurement_Compute reduces a set of
// samples to the mean, standard deviation, extrema, median, and quartiles poop
// reports — checked against a hand-computed five-point set.
func Test_Statistics_Distribution(t *testing.T) {
	measurement := maddox.Measurement_Compute([]int64{10, 20, 30, 40, 50}, "count")
	if measurement.Sample_Count != 5 {
		t.Fatalf("sample count = %d, want 5", measurement.Sample_Count)
	}
	if measurement.Mean != fixedpoint.From_Integer(30) {
		t.Fatalf("mean = %d, want 30.0", measurement.Mean)
	}
	// Sample standard deviation uses n-1 in the denominator: sqrt(1000/4) = 15.811388.
	if fixedpoint.Format(measurement.Standard_Deviation, 3) != "15.811" {
		t.Fatalf("stddev = %s, want 15.811",
			fixedpoint.Format(measurement.Standard_Deviation, 3))
	}
	if measurement.Min != fixedpoint.From_Integer(10) {
		t.Fatalf("min = %d, want 10.0", measurement.Min)
	}
	if measurement.Max != fixedpoint.From_Integer(50) {
		t.Fatalf("max = %d, want 50.0", measurement.Max)
	}
	if measurement.Median != fixedpoint.From_Integer(30) {
		t.Fatalf("median = %d, want 30.0", measurement.Median)
	}
	if measurement.Q1 != fixedpoint.From_Integer(20) {
		t.Fatalf("q1 = %d, want 20.0", measurement.Q1)
	}
	if measurement.Q3 != fixedpoint.From_Integer(50) {
		t.Fatalf("q3 = %d, want 50.0", measurement.Q3)
	}
}

// Test_Statistics_Outliers verifies that Measurement_Compute counts a sample
// beyond Tukey's fences (1.5*IQR past a quartile) as an outlier: nine identical
// points collapse the IQR to zero, so the lone large point falls outside.
func Test_Statistics_Outliers(t *testing.T) {
	measurement := maddox.Measurement_Compute(
		[]int64{10, 10, 10, 10, 10, 10, 10, 10, 10, 1000}, "count")
	if measurement.Outlier_Count != 1 {
		t.Fatalf("outlier count = %d, want 1", measurement.Outlier_Count)
	}
}

// Test_Comparison_Reference verifies that comparing a measurement to itself — the
// reference command against its own numbers — yields a zero delta that is not
// significant, so the baseline never flags itself as a change.
func Test_Comparison_Reference(t *testing.T) {
	reference := maddox.Measurement{
		Mean:               fixedpoint.From_Integer(100),
		Standard_Deviation: fixedpoint.From_Integer(5),
		Sample_Count:       10,
	}
	delta := maddox.Compare(&maddox.Compare_Input{Reference: reference, Candidate: reference})
	if delta.Diff_Percent != 0 {
		t.Fatalf("diff percent = %d, want 0", delta.Diff_Percent)
	}
	if delta.Significant {
		t.Fatal("a measurement compared to itself must not be significant")
	}
	// A real program reports metrics in the billions; a standard deviation past 2^32
	// must not overflow the pooled-variance arithmetic.
	large := maddox.Measurement{
		Mean:               fixedpoint.From_Integer(20_000_000_000),
		Standard_Deviation: fixedpoint.From_Integer(6_000_000_000),
		Sample_Count:       10,
	}
	big := maddox.Compare(&maddox.Compare_Input{Reference: large, Candidate: large})
	if big.Significant {
		t.Fatal("a large measurement compared to itself must not be significant")
	}
}

// Test_Comparison_Significance verifies that a candidate whose mean is double the
// reference's, with tight variance, is reported as a significant slowdown: the
// confidence interval clears the +1% band, and the candidate is not faster.
func Test_Comparison_Significance(t *testing.T) {
	reference := maddox.Measurement{
		Mean:               fixedpoint.From_Integer(100),
		Standard_Deviation: fixedpoint.From_Integer(1),
		Sample_Count:       20,
	}
	candidate := maddox.Measurement{
		Mean:               fixedpoint.From_Integer(200),
		Standard_Deviation: fixedpoint.From_Integer(1),
		Sample_Count:       20,
	}
	delta := maddox.Compare(&maddox.Compare_Input{Reference: reference, Candidate: candidate})
	if !delta.Significant {
		t.Fatal("a doubled mean with tight variance must be significant")
	}
	if delta.Faster {
		t.Fatal("a doubled mean is slower, not faster")
	}
	if delta.Diff_Percent != fixedpoint.From_Integer(100) {
		t.Fatalf("diff percent = %d, want 100.0", delta.Diff_Percent)
	}
}

// Test_Sampling_Budget verifies that Main keeps sampling until the time budget is
// spent: at 10ms per run a 55ms budget admits exactly six runs, the run after the
// budget elapses being the one that stops the loop.
func Test_Sampling_Budget(t *testing.T) {
	clock := time.Virtual_Clock_To_Clock(time.Virtual_Clock{Resolution: time.Millisecond})
	per_run := 10 * time.Millisecond
	calls := 0
	sampler := maddox.Sampler{
		Measure: func(_ sh.Command) (result maddox.Run_Result) {
			calls++
			clock.Sleep(per_run)
			return result
		},
	}
	input := &maddox.Main_Input{
		Commands:     []sh.Command{{Path: "noop"}},
		Clock:        clock,
		Sampler:      sampler,
		Duration_Max: 55 * time.Millisecond,
		Output:       &bytes.Buffer{},
		Stderr:       &bytes.Buffer{},
	}
	if code := maddox.Main(input); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if calls != 6 {
		t.Fatalf("runs = %d, want 6 within a 55ms budget at 10ms/run", calls)
	}
}

// Test_Sampling_Runs verifies that the run cap stops sampling: a cap of five with the
// time budget disabled (zero) yields exactly five runs.
func Test_Sampling_Runs(t *testing.T) {
	clock := time.Virtual_Clock_To_Clock(time.Virtual_Clock{Resolution: time.Millisecond})
	calls := 0
	sampler := maddox.Sampler{
		Measure: func(_ sh.Command) (result maddox.Run_Result) {
			calls++
			clock.Sleep(10 * time.Millisecond)
			return result
		},
	}
	input := &maddox.Main_Input{
		Commands:     []sh.Command{{Path: "noop"}},
		Clock:        clock,
		Sampler:      sampler,
		Duration_Max: 0,
		Runs_Max:     5,
		Output:       &bytes.Buffer{},
		Stderr:       &bytes.Buffer{},
	}
	maddox.Main(input)
	if calls != 5 {
		t.Fatalf("runs = %d, want the 5-run cap", calls)
	}
}

// Test_Sampling_Minimum verifies that Main runs a command at least three times even
// when the budget is already spent, so the statistics always have a quorum.
func Test_Sampling_Minimum(t *testing.T) {
	clock := time.Virtual_Clock_To_Clock(time.Virtual_Clock{Resolution: time.Millisecond})
	calls := 0
	sampler := maddox.Sampler{
		Measure: func(_ sh.Command) (result maddox.Run_Result) {
			calls++
			clock.Sleep(10 * time.Millisecond)
			return result
		},
	}
	input := &maddox.Main_Input{
		Commands:     []sh.Command{{Path: "noop"}},
		Clock:        clock,
		Sampler:      sampler,
		Duration_Max: time.Nanosecond,
		Output:       &bytes.Buffer{},
		Stderr:       &bytes.Buffer{},
	}
	maddox.Main(input)
	if calls != 3 {
		t.Fatalf("runs = %d, want the 3-run minimum on a spent budget", calls)
	}
}

// Test_Sampling_Warmup verifies that Main discards the warmup runs before sampling:
// two warmup runs plus the three-run minimum is five measurements taken, but the
// report counts only the three that were kept.
func Test_Sampling_Warmup(t *testing.T) {
	clock := time.Virtual_Clock_To_Clock(time.Virtual_Clock{Resolution: time.Millisecond})
	calls := 0
	sampler := maddox.Sampler{
		Measure: func(_ sh.Command) (result maddox.Run_Result) {
			calls++
			clock.Sleep(10 * time.Millisecond)
			return result
		},
	}
	output := &bytes.Buffer{}
	input := &maddox.Main_Input{
		Commands:     []sh.Command{{Path: "noop"}},
		Clock:        clock,
		Sampler:      sampler,
		Duration_Max: time.Nanosecond,
		Warmup_Count: 2,
		Format:       maddox.Output_Format_Json,
		Output:       output,
		Stderr:       &bytes.Buffer{},
	}
	maddox.Main(input)
	if calls != 5 {
		t.Fatalf("total measure calls = %d, want 5 (2 warmup + 3 kept)", calls)
	}
	document := decode(t, output)
	runs := document.Benchmarks[0].Runs
	if runs != 3 {
		t.Fatalf("reported runs = %d, want 3 (warmup excluded)", runs)
	}
}

// Test_Output_Document verifies the JSON report shape: one benchmark per command in
// order, the first (reference) carrying no deltas and the rest carrying them, each
// labeled with its command words.
func Test_Output_Document(t *testing.T) {
	clock := time.Virtual_Clock_To_Clock(time.Virtual_Clock{Resolution: time.Millisecond})
	sampler := maddox.Sampler{
		Measure: func(_ sh.Command) (result maddox.Run_Result) {
			clock.Sleep(time.Millisecond)
			result.Sample = maddox.Sample{Instructions: 100}
			return result
		},
	}
	output := &bytes.Buffer{}
	input := &maddox.Main_Input{
		Commands:     []sh.Command{{Path: "a"}, {Path: "b"}},
		Clock:        clock,
		Sampler:      sampler,
		Duration_Max: time.Nanosecond,
		Format:       maddox.Output_Format_Json,
		Output:       output,
		Stderr:       &bytes.Buffer{},
	}
	maddox.Main(input)
	document := decode(t, output)
	if len(document.Benchmarks) != 2 {
		t.Fatalf("benchmarks = %d, want 2", len(document.Benchmarks))
	}
	if document.Benchmarks[0].Command[0] != "a" {
		t.Fatalf("first command = %v, want [a]", document.Benchmarks[0].Command)
	}
	if document.Benchmarks[0].Deltas != nil {
		t.Fatal("the reference command must carry no deltas")
	}
	if document.Benchmarks[1].Deltas == nil {
		t.Fatal("a non-reference command must carry deltas")
	}
}

// Test_Output_Failure verifies that a command exiting non-zero aborts the run with
// a non-zero status and the command's captured stderr surfaced to the diagnostic
// sink, rather than reporting numbers for a broken command.
func Test_Output_Failure(t *testing.T) {
	clock := time.Virtual_Clock_To_Clock(time.Virtual_Clock{Resolution: time.Millisecond})
	sampler := maddox.Sampler{
		Measure: func(_ sh.Command) (result maddox.Run_Result) {
			result.Exit = 1
			result.Stderr = []byte("boom")
			return result
		},
	}
	stderr := &bytes.Buffer{}
	input := &maddox.Main_Input{
		Commands:     []sh.Command{{Path: "broken"}},
		Clock:        clock,
		Sampler:      sampler,
		Duration_Max: time.Nanosecond,
		Output:       &bytes.Buffer{},
		Stderr:       stderr,
	}
	if code := maddox.Main(input); code == 0 {
		t.Fatal("a non-zero exit must abort with a non-zero status")
	}
	if !strings.Contains(stderr.String(), "boom") {
		t.Fatalf("diagnostics = %q, want the command's stderr", stderr.String())
	}
}

// Test_Table_Header verifies the rendered table names each benchmark with its
// position, kept-run count, and command words.
func Test_Table_Header(t *testing.T) {
	document := maddox.Document{Benchmarks: []maddox.Benchmark{
		{Command: []string{"echo", "hi"}, Runs: 5, Elapsed: 2 * time.Second},
	}}
	output := string(maddox.Render_Table(&maddox.Render_Table_Input{Document: document}))
	if !strings.Contains(output, "Benchmark 1") {
		t.Fatalf("missing benchmark header, got: %s", output)
	}
	if !strings.Contains(output, "echo hi") {
		t.Fatalf("missing command words, got: %s", output)
	}
	if !strings.Contains(output, "5 runs") {
		t.Fatalf("missing run count, got: %s", output)
	}
	if !strings.Contains(output, "2s") {
		t.Fatalf("missing elapsed time, got: %s", output)
	}
}

// Test_Table_Units verifies a raw nanosecond value is scaled to a human unit: a
// ~14.9ms mean renders in milliseconds, not bare nanoseconds.
func Test_Table_Units(t *testing.T) {
	wall_ns := fixedpoint.From_Integer(14906807)
	wall := maddox.Measurement{
		Mean: wall_ns, Min: wall_ns, Max: wall_ns, Median: wall_ns,
		Q1: wall_ns, Q3: wall_ns, Sample_Count: 3, Unit: "nanoseconds",
	}
	// 2 MiB exactly: binary scaling renders "2MiB", decimal would be "2.10MB".
	bytes_2mib := fixedpoint.From_Integer(2097152)
	memory := maddox.Measurement{
		Mean: bytes_2mib, Min: bytes_2mib, Max: bytes_2mib, Median: bytes_2mib,
		Q1: bytes_2mib, Q3: bytes_2mib, Sample_Count: 3, Unit: "bytes",
	}
	document := maddox.Document{Benchmarks: []maddox.Benchmark{{
		Command:      []string{"x"},
		Runs:         3,
		Measurements: maddox.Measurements{Wall_Time: wall, Peak_RSS: memory},
	}}}
	output := string(maddox.Render_Table(&maddox.Render_Table_Input{Document: document}))
	if !strings.Contains(output, "14.9ms") {
		t.Fatalf("nanoseconds not scaled to ms, got: %s", output)
	}
	if !strings.Contains(output, "2MiB") {
		t.Fatalf("bytes not scaled to binary MiB, got: %s", output)
	}
}

// Test_Table_Delta verifies a non-reference benchmark's row carries its signed
// percentage change against the reference.
func Test_Table_Delta(t *testing.T) {
	mean := fixedpoint.From_Integer(1_000_000)
	wall := maddox.Measurement{Mean: mean, Max: mean, Sample_Count: 3, Unit: "nanoseconds"}
	deltas := maddox.Deltas{
		Wall_Time: maddox.Delta{
			Diff_Percent: fixedpoint.From_Integer(50),
			Half_Percent: fixedpoint.From_Integer(2),
			Significant:  true,
		},
	}
	measurements := maddox.Measurements{Wall_Time: wall}
	document := maddox.Document{Benchmarks: []maddox.Benchmark{
		{Command: []string{"a"}, Runs: 3, Measurements: measurements},
		{Command: []string{"b"}, Runs: 3, Measurements: measurements, Deltas: &deltas},
	}}
	output := string(maddox.Render_Table(&maddox.Render_Table_Input{Document: document}))
	if !strings.Contains(output, "50.0%") {
		t.Fatalf("missing delta percentage, got: %s", output)
	}
}

// Test_Table_Color verifies ANSI escapes mark the delta only when color is enabled,
// and never otherwise.
func Test_Table_Color(t *testing.T) {
	mean := fixedpoint.From_Integer(1_000_000)
	wall := maddox.Measurement{Mean: mean, Max: mean, Sample_Count: 3, Unit: "nanoseconds"}
	deltas := maddox.Deltas{
		Wall_Time: maddox.Delta{
			Diff_Percent: fixedpoint.From_Integer(50),
			Half_Percent: fixedpoint.From_Integer(2),
			Significant:  true,
		},
	}
	measurements := maddox.Measurements{Wall_Time: wall}
	document := maddox.Document{Benchmarks: []maddox.Benchmark{
		{Command: []string{"a"}, Runs: 3, Measurements: measurements},
		{Command: []string{"b"}, Runs: 3, Measurements: measurements, Deltas: &deltas},
	}}
	colored := string(maddox.Render_Table(&maddox.Render_Table_Input{
		Document: document, Color: true,
	}))
	plain := string(maddox.Render_Table(&maddox.Render_Table_Input{
		Document: document, Color: false,
	}))
	if !strings.Contains(colored, "\x1b[") {
		t.Fatal("color output lacks ANSI escapes in the delta")
	}
	if strings.Contains(plain, "\x1b[") {
		t.Fatal("plain output must not contain ANSI escapes")
	}
}

// Test_Table_Sparse verifies that a metric with no data — every value zero, like the
// Linux-only counters on macOS — is omitted from the table, while a metric that has
// data is shown.
func Test_Table_Sparse(t *testing.T) {
	mean := fixedpoint.From_Integer(1_000_000)
	wall := maddox.Measurement{Mean: mean, Max: mean, Sample_Count: 3, Unit: "nanoseconds"}
	document := maddox.Document{Benchmarks: []maddox.Benchmark{{
		Command:      []string{"x"},
		Runs:         3,
		Measurements: maddox.Measurements{Wall_Time: wall},
	}}}
	output := string(maddox.Render_Table(&maddox.Render_Table_Input{Document: document}))
	if !strings.Contains(output, "wall_time") {
		t.Fatalf("a metric with data must render, got: %s", output)
	}
	if strings.Contains(output, "cache_references") {
		t.Fatalf("an all-zero metric must be omitted, got: %s", output)
	}
}

// Test_Machine_Document verifies the JSON document carries the injected machine
// specs as a top-level "machine" field alongside "benchmarks", with every provided
// field round-tripping through the marshaled output.
func Test_Machine_Document(t *testing.T) {
	clock := time.Virtual_Clock_To_Clock(time.Virtual_Clock{Resolution: time.Millisecond})
	sampler := maddox.Sampler{
		Measure: func(_ sh.Command) (result maddox.Run_Result) {
			clock.Sleep(time.Millisecond)
			return result
		},
	}
	output := &bytes.Buffer{}
	specs := maddox.Machine_Specs{
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
	maddox.Main(&maddox.Main_Input{
		Commands:     []sh.Command{{Path: "noop"}},
		Clock:        clock,
		Sampler:      sampler,
		Duration_Max: time.Nanosecond,
		Machine:      specs,
		Format:       maddox.Output_Format_Json,
		Output:       output,
		Stderr:       &bytes.Buffer{},
	})
	document := decode(t, output)
	if document.Machine.CPU_Model != "TestCPU X1" {
		t.Fatalf("machine cpu model = %q, want TestCPU X1", document.Machine.CPU_Model)
	}
	if document.Machine.Physical_Cores != 8 {
		t.Fatalf("machine physical cores = %d, want 8", document.Machine.Physical_Cores)
	}
	if document.Machine.RAM_Total_Bytes != 16*1024*1024*1024 {
		t.Fatalf("machine ram bytes = %d, want 16GiB", document.Machine.RAM_Total_Bytes)
	}
	if document.Machine.Storage_Total_Bytes != 512_000_000_000 {
		t.Fatalf("machine storage bytes = %d, want 512GB",
			document.Machine.Storage_Total_Bytes)
	}
}

// Test_Machine_Table verifies the table output carries a machine header block before
// the first benchmark, naming the CPU model and key specs.
func Test_Machine_Table(t *testing.T) {
	specs := maddox.Machine_Specs{
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
	document := maddox.Document{
		Machine:    specs,
		Benchmarks: []maddox.Benchmark{{Command: []string{"echo"}, Runs: 3}},
	}
	output := string(maddox.Render_Table(&maddox.Render_Table_Input{Document: document}))
	if !strings.Contains(output, "TestCPU X1") {
		t.Fatalf("machine header missing CPU model, got: %s", output)
	}
	if !strings.Contains(output, "512GiB") {
		t.Fatalf("machine header missing storage capacity, got: %s", output)
	}
	// The machine header must appear before the first benchmark.
	machine_offset := strings.Index(output, "TestCPU X1")
	benchmark_offset := strings.Index(output, "Benchmark 1")
	if machine_offset > benchmark_offset {
		t.Fatalf("machine header must precede Benchmark 1, got: %s", output)
	}
}

// Test_Progress verifies that, with progress enabled, Main writes the run counter to
// the diagnostic sink while sampling, and writes nothing there when it is disabled.
func Test_Progress(t *testing.T) {
	clock := time.Virtual_Clock_To_Clock(time.Virtual_Clock{Resolution: time.Millisecond})
	sampler := maddox.Sampler{
		Measure: func(_ sh.Command) (result maddox.Run_Result) {
			return result
		},
	}
	shown := &bytes.Buffer{}
	maddox.Main(&maddox.Main_Input{
		Commands:     []sh.Command{{Path: "noop"}},
		Clock:        clock,
		Sampler:      sampler,
		Duration_Max: 0,
		Runs_Max:     5,
		Warmup_Count: 2,
		Progress:     true,
		Output:       &bytes.Buffer{},
		Stderr:       shown,
	})
	if !strings.Contains(shown.String(), "warmup 2/2") {
		t.Fatalf("progress should report warmup, got: %q", shown.String())
	}
	if !strings.Contains(shown.String(), "5/5") {
		t.Fatalf("progress should report the 5/5 counter, got: %q", shown.String())
	}

	hidden := &bytes.Buffer{}
	maddox.Main(&maddox.Main_Input{
		Commands:     []sh.Command{{Path: "noop"}},
		Clock:        clock,
		Sampler:      sampler,
		Duration_Max: 0,
		Runs_Max:     5,
		Progress:     false,
		Output:       &bytes.Buffer{},
		Stderr:       hidden,
	})
	if hidden.Len() != 0 {
		t.Fatalf("progress disabled must write nothing, got: %q", hidden.String())
	}
}

// Decode unmarshals a JSON report from the buffer, failing the test if it is not
// well-formed.
func decode(t *testing.T, buffer *bytes.Buffer) (document maddox.Document) {
	unmarshal_err := json.Unmarshal(buffer.Bytes(), &document)
	if unmarshal_err != nil {
		t.Fatalf("report is not valid JSON: %v", unmarshal_err)
	}
	return document
}
