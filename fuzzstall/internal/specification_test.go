package fuzzstall_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	fuzzstall "local/james-orcales/fuzzstall/internal"
	systime "local/james-orcales/shared/simulation/time"
)

// Test_Rate_Quota verifies a rate carries the quota written before its separator, and that a
// bare window asks for the one input it has always asked for.
func Test_Rate_Quota(t *testing.T) {
	type quota_case struct {
		Text   string
		Quota  fuzzstall.Quota
		Window fuzzstall.Window
	}
	cases := []quota_case{
		{Text: "90/30s", Quota: 90, Window: fuzzstall.Window(30 * systime.SECOND)},
		// A bare window and the same window written with its quota name one rate.
		{Text: "30s", Quota: 1, Window: fuzzstall.Window(30 * systime.SECOND)},
		{Text: "1/30s", Quota: 1, Window: fuzzstall.Window(30 * systime.SECOND)},
		// Both bounds a quota may reach.
		{Text: "1/1s", Quota: fuzzstall.QUOTA_MIN, Window: fuzzstall.WINDOW_MIN},
		{Text: "1000000/3600s",
			Quota: fuzzstall.QUOTA_MAX, Window: fuzzstall.WINDOW_MAX},
	}
	for _, want := range cases {
		if err := fuzzstall.Rate_Check(fuzzstall.Text(want.Text)); err != nil {
			t.Fatalf("Rate_Check(%q): %v", want.Text, err)
		}
		rate := fuzzstall.Rate_Parse(fuzzstall.Rate_Text(want.Text))
		if rate.Quota != want.Quota {
			t.Errorf("Rate_Parse(%q) quota = %d, want %d",
				want.Text, rate.Quota, want.Quota)
		}
		if rate.Window != want.Window {
			t.Errorf("Rate_Parse(%q) window = %d, want %d",
				want.Text, rate.Window, want.Window)
		}
	}
}

// Test_Rate_Seconds verifies a window scales its magnitude by the second, the one unit
// there is, over the whole span a window may name.
func Test_Rate_Seconds(t *testing.T) {
	type unit_case struct {
		Text string
		Span fuzzstall.Window
	}
	cases := []unit_case{
		{Text: "5s", Span: fuzzstall.Window(5 * systime.SECOND)},
		{Text: "60s", Span: fuzzstall.Window(60 * systime.SECOND)},
		// The shortest window that still names a span.
		{Text: "1s", Span: fuzzstall.WINDOW_MIN},
		// The bound itself, which a window may reach and may not pass.
		{Text: "3600s", Span: fuzzstall.WINDOW_MAX},
	}
	for _, want := range cases {
		if err := fuzzstall.Duration_Check(fuzzstall.Text(want.Text)); err != nil {
			t.Fatalf("Duration_Check(%q): %v", want.Text, err)
		}
		window := fuzzstall.Duration_Parse(fuzzstall.Window_Text(want.Text))
		if window != want.Span {
			t.Errorf("Duration_Parse(%q) = %d, want %d", want.Text, window, want.Span)
		}
	}
}

// Test_Rate_Malformed verifies a rate that names no single unambiguous minimum is rejected
// rather than read as some other minimum.
func Test_Rate_Malformed(t *testing.T) {
	cases := []string{
		"",                     // nothing at all
		"5",                    // no suffix
		"s",                    // no magnitude
		"/30s",                 // a separator with no quota before it
		"90/",                  // a quota with no window behind it
		"x/30s",                // a quota that is not a number
		"0/30s",                // a quota that asks for nothing
		"-1/30s",               // a quota below every quota there is
		"1000001/30s",          // one past the largest quota there is
		"90/30s/1",             // a second separator, which names no rate
		"5ns",                  // nanoseconds are not the unit
		"5us",                  // microseconds are not the unit
		"5ms",                  // milliseconds are not the unit
		"5m",                   // minutes are not the unit
		"5h",                   // hours are not the unit
		"5sec",                 // not a suffix this program spells
		"-5s",                  // a window that ran out before it started
		"0s",                   // a window with no span at all
		"3601s",                // one second past the longest window there is
		"9223372036854775807s", // scales past the nanosecond ceiling
	}
	for _, text := range cases {
		if fuzzstall.Rate_Check(fuzzstall.Text(text)) == nil {
			t.Errorf("Rate_Check(%q) accepted a malformed rate", text)
		}
	}
}

// Test_Command_Split verifies the window is the first argument and every argument behind
// it reaches `go test` verbatim, dashes included.
func Test_Command_Split(t *testing.T) {
	command_line := fuzzstall.Command_Line{
		"fuzzstall", "30s", "-fuzz=Fuzz_Parse", "./shared/cli"}
	if err := fuzzstall.Command_Check(command_line); err != nil {
		t.Fatalf("Command_Check: %v", err)
	}
	command := fuzzstall.Command_Parse(fuzzstall.Checked_Command_Line(command_line))
	if command.Rate.Window != fuzzstall.Window(30*systime.SECOND) {
		t.Errorf("window = %d, want %d", command.Rate.Window, 30*systime.SECOND)
	}
	if command.Path != fuzzstall.GO {
		t.Errorf("path = %q, want %q", command.Path, fuzzstall.GO)
	}
	// -fuzz is go test's flag: had the wrapper claimed it, the run would lose its target.
	want := "test -fuzz=Fuzz_Parse ./shared/cli"
	if strings.Join([]string(command.Arguments), " ") != want {
		t.Errorf("arguments = %v, want [%s]", command.Arguments, want)
	}
}

// Test_Command_Absent verifies a command line missing its window or its `go test`
// arguments is a usage error, reported as the usage exit code.
func Test_Command_Absent(t *testing.T) {
	cases := []fuzzstall.Command_Line{
		{"fuzzstall"},        // no window, no arguments
		{"fuzzstall", "30s"}, // a window over nothing
	}
	for _, command_line := range cases {
		if fuzzstall.Command_Check(command_line) == nil {
			t.Errorf("Command_Check(%v) accepted an incomplete line", command_line)
		}
	}
	code, _, problems := drive(fuzzstall.Command_Line{"fuzzstall", "30s"}, nil, 0)
	if code != fuzzstall.EXIT_USAGE {
		t.Errorf("exit = %d, want %d (usage)", code, fuzzstall.EXIT_USAGE)
	}
	if problems == "" {
		t.Errorf("expected a usage message on standard error")
	}
}

// Test_Progress_Count verifies the number behind the interesting field is the count read,
// whatever surrounds it on the line.
func Test_Progress_Count(t *testing.T) {
	line := "fuzz: elapsed: 9s, execs: 5 (1/sec), new interesting: 42 (total: 51)"
	interesting_count, found := fuzzstall.Interesting_Parse(fuzzstall.Line(line))
	if !found {
		t.Fatalf("Interesting_Parse found no count in %q", line)
	}
	if interesting_count != 42 {
		t.Errorf("count = %d, want 42", interesting_count)
	}
}

// Test_Progress_Other verifies a line carrying no interesting field reports no count, so
// the run's other output never moves the record.
func Test_Progress_Other(t *testing.T) {
	cases := []string{
		"fuzz: elapsed: 0s, gathering baseline coverage: 0/38 completed",
		"--- FAIL: Fuzz_Parse (3.02s)",
		"PASS",
		"",
	}
	for _, line := range cases {
		_, found := fuzzstall.Interesting_Parse(fuzzstall.Line(line))
		if found {
			t.Errorf("Interesting_Parse(%q) read a count that is not there", line)
		}
	}
}

// Test_Progress_Malformed verifies a field with no readable number behind it reports no
// count rather than some other number.
func Test_Progress_Malformed(t *testing.T) {
	cases := []string{
		"new interesting: ",                     // nothing behind the field
		"new interesting: x",                    // not a number
		"new interesting: 99999999999999999999", // more digits than an int64 counts
	}
	for _, line := range cases {
		_, found := fuzzstall.Interesting_Parse(fuzzstall.Line(line))
		if found {
			t.Errorf("Interesting_Parse(%q) read a malformed count", line)
		}
	}
}

// Test_Line_Split verifies a line broken across chunks is assembled whole, and that a
// chunk holding several lines yields each of them.
func Test_Line_Split(t *testing.T) {
	reader := fuzzstall.Line_Reader{}
	lines := collect(&reader, []string{"one\ntw", "o\nthre", "e"})
	if strings.Join(lines, "|") != "one|two" {
		t.Errorf("lines = %v, want [one two]", lines)
	}
	// The trailing "three" carries no newline, so it is not a line yet.
	lines = collect(&reader, []string{"\n"})
	if strings.Join(lines, "|") != "three" {
		t.Errorf("lines = %v, want [three]", lines)
	}
}

// Test_Line_Overflow verifies a line longer than the buffer keeps its first bytes and
// still yields the count a progress line carries near its start.
func Test_Line_Overflow(t *testing.T) {
	reader := fuzzstall.Line_Reader{}
	head := "new interesting: 7 "
	long := head + strings.Repeat("x", fuzzstall.LINE_SIZE_MAX*3)
	lines := collect(&reader, []string{long, "\n"})
	if len(lines) != 1 {
		t.Fatalf("lines = %v, want one truncated line", lines)
	}
	if len(lines[0]) != fuzzstall.LINE_SIZE_MAX {
		t.Errorf("line held %d bytes, want the %d-byte bound",
			len(lines[0]), fuzzstall.LINE_SIZE_MAX)
	}
	interesting_count, found := fuzzstall.Interesting_Parse(fuzzstall.Line(lines[0]))
	if !found {
		t.Fatalf("a truncated line lost the count near its start")
	}
	if interesting_count != 7 {
		t.Errorf("count = %d, want 7", interesting_count)
	}
}

// Test_Run_Exit_Code verifies a run that stops by itself yields its own exit code and is
// never terminated, so a failing fuzz target reads as a failure.
func Test_Run_Exit_Code(t *testing.T) {
	steps := []step{
		{Chunk: progress(3, 2), Event: fuzzstall.EVENT_OUTPUT, At: moment(3)},
		{Chunk: "--- FAIL: Fuzz_Parse\n", Event: fuzzstall.EVENT_OUTPUT, At: moment(4)},
	}
	code, output, _ := drive(watch("30s"), steps, 1)
	if code != 1 {
		t.Errorf("exit = %d, want 1 (the run's own)", code)
	}
	if !strings.Contains(output, "--- FAIL") {
		t.Errorf("the run's output must pass through, got %q", output)
	}
}

// Test_Run_Saturation verifies a count that holds still for the whole window terminates
// the run and reports 124.
func Test_Run_Saturation(t *testing.T) {
	steps := []step{
		{Chunk: progress(3, 2), Event: fuzzstall.EVENT_OUTPUT, At: moment(3)},
		{Chunk: progress(9, 2), Event: fuzzstall.EVENT_OUTPUT, At: moment(9)},
		{Event: fuzzstall.EVENT_DEADLINE, At: moment(13)},
	}
	code, _, problems := drive(watch("10s"), steps, 0)
	if code != fuzzstall.EXIT_SATURATED {
		t.Errorf("exit = %d, want %d (saturated)", code, fuzzstall.EXIT_SATURATED)
	}
	if !strings.Contains(problems, "saturated") {
		t.Errorf("expected saturation on standard error, got %q", problems)
	}
}

// Test_Run_Baseline verifies the window is measured from the first progress line, so a
// long baseline pass is never read as saturation.
func Test_Run_Baseline(t *testing.T) {
	steps := []step{
		{Chunk: BASELINE, Event: fuzzstall.EVENT_OUTPUT, At: moment(1)},
		{Event: fuzzstall.EVENT_DEADLINE, At: moment(90)},
		{Chunk: progress(91, 0), Event: fuzzstall.EVENT_OUTPUT, At: moment(91)},
	}
	code, _, problems := drive(watch("10s"), steps, 0)
	if code != fuzzstall.EXIT_SUCCESS {
		t.Errorf("exit = %d, want 0, a baseline is not saturation (%q)", code, problems)
	}
}

// Test_Run_Start_Failure verifies a `go test` that never starts reports 125 and the
// reason, keeping the wrapper's own failure distinct from any exit code a run could yield.
func Test_Run_Start_Failure(t *testing.T) {
	start := func(
		path fuzzstall.Toolchain, arguments fuzzstall.Arguments,
	) (child fuzzstall.Child, err error) {
		return fuzzstall.Child{}, errors.New("executable file not found")
	}
	code, _, problems := run(watch("30s"), start)
	if code != fuzzstall.EXIT_START_FAILURE {
		t.Errorf("exit = %d, want %d (the wrapper's own failure)",
			code, fuzzstall.EXIT_START_FAILURE)
	}
	if !strings.Contains(problems, "not found") {
		t.Errorf("expected the start failure's reason, got %q", problems)
	}
}

// PROGRESS is a `go test -fuzz` progress line with a substitutable count, the one shape
// this program reads. It is written as a format so a case can name its own count.
const PROGRESS = "fuzz: elapsed: %ds, execs: 100 (33/sec), new interesting: %d (total: 7)\n"

// BASELINE is the line `go test -fuzz` writes while it gathers baseline coverage, before
// any count exists to read.
const BASELINE = "fuzz: elapsed: 0s, gathering baseline coverage: 3/38 completed\n"

// One turn of a scripted run: what the run's output does, and the moment the clock reads
// once it has done it.
type step struct {
	// Chunk is the bytes the run wrote, empty when the step is not an output step.
	Chunk string
	// Event is what the bounded wait on the output produced.
	Event fuzzstall.Event
	// At is the moment the clock reads from this step onward.
	At fuzzstall.Reading
}

// Returns the command line that watches one package over window_text.
func watch(window_text string) (command_line fuzzstall.Command_Line) {
	return fuzzstall.Command_Line{"fuzzstall", window_text, "./shared/cli"}
}

// Returns the clock reading a scripted step lands on, counted in whole seconds off an
// epoch of zero. Only the difference between two readings matters, thus the epoch is free.
func moment(second int) (reading fuzzstall.Reading) {
	return fuzzstall.Reading(systime.Duration(second) * systime.SECOND)
}

// Returns the progress line a run writes at elapsed_second holding interesting_count.
func progress(elapsed_second int, interesting_count int) (line string) {
	return fmt.Sprintf(PROGRESS, elapsed_second, interesting_count)
}

// Runs Main over a scripted run and returns its exit code, everything the run wrote, and
// everything this program wrote to standard error.
func drive(
	command_line fuzzstall.Command_Line, steps []step, status_code fuzzstall.Status_Code,
) (code fuzzstall.Status_Code, output string, problems string) {
	index := 0
	reading := fuzzstall.Reading(0)
	start := func(
		path fuzzstall.Toolchain, arguments fuzzstall.Arguments,
	) (child fuzzstall.Child, err error) {
		return fuzzstall.Child{
			Next_Output: func(
				deadline fuzzstall.Deadline,
			) (chunk fuzzstall.Chunk, event fuzzstall.Event) {
				if index == len(steps) {
					return nil, fuzzstall.EVENT_END
				}
				current := steps[index]
				index++
				reading = current.At
				return fuzzstall.Chunk(current.Chunk), current.Event
			},
			Terminate: func() {},
			Wait: func() (finished fuzzstall.Status_Code) {
				return status_code
			},
		}, nil
	}
	return run_at(command_line, start, func() (now fuzzstall.Reading) { return reading })
}

// Runs Main over start with a clock parked at zero, for the cases that never reach it.
func run(
	command_line fuzzstall.Command_Line, start fuzzstall.Start,
) (code fuzzstall.Status_Code, output string, problems string) {
	return run_at(command_line, start, func() (now fuzzstall.Reading) { return 0 })
}

// Runs Main over start and now, returning its exit code and both of its output streams.
func run_at(
	command_line fuzzstall.Command_Line,
	start fuzzstall.Start,
	now func() (reading fuzzstall.Reading),
) (code fuzzstall.Status_Code, output string, problems string) {
	run_output := strings.Builder{}
	problem_output := strings.Builder{}
	status_code := fuzzstall.Main(&fuzzstall.Main_Input{
		Arguments:    command_line,
		Output:       &run_output,
		Error_Output: &problem_output,
		Now:          now,
		Start:        start,
	})
	return status_code, run_output.String(), problem_output.String()
}

// Feeds every chunk through reader and returns the lines it completed.
func collect(reader *fuzzstall.Line_Reader, chunks []string) (lines []string) {
	lines = []string{}
	for _, text := range chunks {
		rest := fuzzstall.Chunk(text)
		for range len(text) {
			line, remainder, complete := fuzzstall.Line_Reader_Next(reader, rest)
			if !complete {
				break
			}
			lines = append(lines, string(line))
			rest = fuzzstall.Chunk(remainder)
		}
	}
	return lines
}
