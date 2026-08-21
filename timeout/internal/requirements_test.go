package timeout_test

import (
	"errors"
	"testing"

	"local/james-orcales/shared/sim/nbio"
	"local/james-orcales/shared/sim/time"
	"local/james-orcales/shared/testify"
	"local/james-orcales/timeout/internal"
)

// Test_Duration_Units verifies every accepted unit scales its magnitude, and that a
// two-letter unit is read whole rather than as its one-letter tail.
func Test_Duration_Units(t *testing.T) {
	type unit_case struct {
		Text string
		Span time.Duration
	}
	cases := []unit_case{
		{Text: "5ns", Span: 5 * time.NANOSECOND},
		{Text: "5us", Span: 5 * time.MICROSECOND},
		{Text: "5ms", Span: 5 * time.MILLISECOND},
		{Text: "5s", Span: 5 * time.SECOND},
		{Text: "5m", Span: 5 * time.MINUTE},
	}
	for _, want := range cases {
		deadline, err := timeout.Duration_Parse(want.Text)
		if err != nil {
			t.Fatalf("Duration_Parse(%q): %v", want.Text, err)
		}
		if deadline != want.Span {
			t.Errorf("Duration_Parse(%q) = %d, want %d", want.Text, deadline, want.Span)
		}
	}
}

// Test_Duration_Malformed verifies a deadline that names no single unambiguous span is
// rejected rather than read as some other span.
func Test_Duration_Malformed(t *testing.T) {
	cases := []string{
		"",                     // nothing at all
		"5",                    // no unit
		"ms",                   // no magnitude
		"5h",                   // hours are not among the supported units
		"5sec",                 // not a unit this program spells
		"-5s",                  // a deadline in the past
		"0s",                   // a deadline that has already elapsed
		"9223372036854775807s", // scales past the nanosecond ceiling
	}
	for _, text := range cases {
		_, err := timeout.Duration_Parse(text)
		if err == nil {
			t.Errorf("Duration_Parse(%q) accepted a malformed deadline", text)
		}
	}
}

// Test_Command_Split verifies the deadline is the first argument and everything behind it
// is the command, dashes included — the wrapper claims no flags of its own.
func Test_Command_Split(t *testing.T) {
	command, err := timeout.Command_Parse(
		[]string{"timeout", "500ms", "echo", "-n", "hello"})
	testify.No_Error(t, err)
	testify.Equal(t, 500*time.MILLISECOND, command.Deadline)
	testify.Equal(t, "echo", command.Path)
	// -n is echo's flag: had the wrapper parsed it, the child would lose it.
	testify.Equal(t, []string{"-n", "hello"}, command.Arguments)
}

// Test_Command_Absent verifies a command line missing its deadline or its command is a
// usage error, reported as the usage exit code before any child starts.
func Test_Command_Absent(t *testing.T) {
	cases := [][]string{
		{"timeout"},       // no deadline, no command
		{"timeout", "5s"}, // a deadline fencing nothing
	}
	for _, arguments := range cases {
		_, err := timeout.Command_Parse(arguments)
		if err == nil {
			t.Errorf("Command_Parse(%v) accepted an incomplete command line", arguments)
		}
	}
	subject := harness_new([]string{"timeout", "5s"}, outcome{})
	testify.True(t, harness_run(subject))
	testify.Equal(t, timeout.EXIT_USAGE, subject.Runner.Status)
	testify.Equal(t, 0, subject.Fake.Spawn_Count)
	testify.True(t, harness_message(subject) != "")
}

// Test_Run_Exit_Code verifies a child that finishes inside its deadline yields its own
// exit code, runs under the parsed deadline, and writes no diagnostic.
func Test_Run_Exit_Code(t *testing.T) {
	subject := harness_new([]string{"timeout", "1s", "false", "-x"}, outcome{Exit: 7})
	testify.True(t, harness_run(subject))
	testify.Equal(t, timeout.Status_Code(7), subject.Runner.Status)
	testify.Equal(t, 1, subject.Fake.Spawn_Count)
	testify.Equal(t, "false", subject.Fake.Request.Path)
	testify.Equal(t, []string{"-x"}, subject.Fake.Request.Arguments)
	// A wrapper is transparent: the child reads the terminal and hears its signals.
	testify.True(t, subject.Fake.Request.Input_Inherited)
	testify.False(t, subject.Fake.Request.Signals_Kept)
	testify.Equal(t, subject.Template.Environment, subject.Fake.Request.Environment)
	testify.Equal(t, time.SECOND, subject.Fake.Deadline)
	testify.Equal(t, "", harness_message(subject))
}

// Test_Run_Timeout verifies a child still alive at its deadline is reported on standard
// error and answered with 124.
func Test_Run_Timeout(t *testing.T) {
	subject := harness_new([]string{"timeout", "1ms", "sleep", "60"},
		outcome{Err: nbio.Deadline_Exceeded})
	testify.True(t, harness_run(subject))
	testify.Equal(t, timeout.EXIT_TIMEOUT, subject.Runner.Status)
	testify.Equal(t, time.MILLISECOND, subject.Fake.Deadline)
	testify.Equal(t, "timeout: sleep: deadline exceeded\n", harness_message(subject))
}

// Test_Run_Start_Failure verifies a command that never starts reports 125 and the reason,
// keeping the wrapper's own failure distinct from any exit code a child could return.
func Test_Run_Start_Failure(t *testing.T) {
	subject := harness_new([]string{"timeout", "1s", "nonesuch"},
		outcome{Err: errors.New("executable file not found")})
	testify.True(t, harness_run(subject))
	testify.Equal(t, timeout.EXIT_START_FAILURE, subject.Runner.Status)
	testify.Equal(t, "timeout: nonesuch: executable file not found\n",
		harness_message(subject))
}

// Test_Run_Message_Full verifies a diagnostic past the message store is replaced whole, never
// cut short and never grown.
func Test_Run_Message_Full(t *testing.T) {
	long_path := string(make([]byte, timeout.MESSAGE_SIZE_MAXIMUM))
	subject := harness_new([]string{"timeout", "1s", long_path},
		outcome{Err: nbio.Deadline_Exceeded})
	testify.True(t, harness_run(subject))
	testify.Equal(t, timeout.EXIT_TIMEOUT, subject.Runner.Status)
	testify.Equal(t, timeout.MESSAGE_FULL, harness_message(subject))
}

// Test_Allocation_Usage verifies the usage failure allocates nothing.
func Test_Allocation_Usage(t *testing.T) {
	subject := harness_new([]string{"timeout", "5x", "sleep"}, outcome{})
	allocation_check(t, subject, timeout.EXIT_USAGE)
}

// Test_Allocation_Exit_Code verifies a child that exits inside its deadline allocates
// nothing.
func Test_Allocation_Exit_Code(t *testing.T) {
	subject := harness_new([]string{"timeout", "1s", "false"}, outcome{Exit: 1})
	allocation_check(t, subject, timeout.Status_Code(1))
}

// Test_Allocation_Timeout verifies a child killed at its deadline allocates nothing.
func Test_Allocation_Timeout(t *testing.T) {
	subject := harness_new([]string{"timeout", "1ms", "sleep", "60"},
		outcome{Err: nbio.Deadline_Exceeded})
	allocation_check(t, subject, timeout.EXIT_TIMEOUT)
}

// Test_Allocation_Start_Failure verifies a child that never starts allocates nothing.
func Test_Allocation_Start_Failure(t *testing.T) {
	subject := harness_new([]string{"timeout", "1s", "nonesuch"},
		outcome{Err: errors.New("executable file not found")})
	allocation_check(t, subject, timeout.EXIT_START_FAILURE)
}

func allocation_check(t *testing.T, subject *harness, status_code timeout.Status_Code) {
	t.Helper()
	stopped := false
	testify.Zero_Allocation(t, func() { stopped = harness_run(subject) })
	testify.True(t, stopped)
	testify.Equal(t, status_code, subject.Runner.Status)
}

// HARNESS_PASS_COUNT bounds delivery passes; a run submits one child and one diagnostic.
const HARNESS_PASS_COUNT = 8

// HARNESS_OUTPUT_BYTES holds any diagnostic this program writes.
const HARNESS_OUTPUT_BYTES = 4096

type outcome struct {
	Exit int
	Err  error
}

// The fake loop holds the spawn until the harness delivers it, so no callback runs inside
// a submission.
type fake struct {
	Outcome     outcome
	Spawn_Count int
	Process     *nbio.Completion
	Request     nbio.Process_Request
	Deadline    time.Duration
	On_Process  nbio.Process_Callback
}

type harness struct {
	Fake      fake
	Runner    timeout.Runner
	Template  timeout.Runner
	Arguments []string
	Output    nbio.Stream_Memory
	Discard   nbio.Stream_Discard
}

func harness_new(arguments []string, scripted outcome) (h *harness) {
	h = &harness{Arguments: arguments}
	h.Fake.Outcome = scripted
	h.Output.Memory = make([]byte, HARNESS_OUTPUT_BYTES)
	loop := nbio.IO{}
	loop.Spawn = fake_spawn
	loop.Storage.State = nbio.State(&h.Fake)
	h.Template = timeout.Runner{
		IO:           loop,
		Environment:  []string{"PATH=/bin"},
		Stdout:       nbio.Discard_To_Stream(&h.Discard),
		Stderr:       nbio.Discard_To_Stream(&h.Discard),
		Error_Output: nbio.Memory_To_Stream(&h.Output),
		Message:      make([]byte, timeout.MESSAGE_SIZE_MAXIMUM),
	}
	return h
}

// Starts from the template each time, so every measured run starts from the same state.
func harness_run(h *harness) (stopped bool) {
	h.Runner = h.Template
	h.Fake.Spawn_Count = 0
	h.Fake.Process = nil
	h.Output.Cursor = 0
	timeout.Main(&h.Runner, h.Arguments)
	for pass_index := 0; pass_index < HARNESS_PASS_COUNT; pass_index++ {
		for timeout.Rearm(&h.Runner) {
		}
		if timeout.Stopped(&h.Runner) {
			return true
		}
		if !fake_deliver(&h.Fake) {
			return false
		}
	}
	return false
}

func harness_message(h *harness) (message string) {
	return string(h.Output.Memory[:h.Output.Cursor])
}

func fake_spawn(
	state nbio.State, completion *nbio.Completion, request nbio.Process_Request,
	deadline time.Duration, callback nbio.Process_Callback,
) {
	loop := state.(*fake)
	loop.Spawn_Count++
	loop.Process = completion
	loop.Request = request
	loop.Deadline = deadline
	loop.On_Process = callback
}

// Reports false when the run waits on nothing, which no correct run does.
func fake_deliver(loop *fake) (delivered bool) {
	if loop.Process == nil {
		return false
	}
	completion := loop.Process
	loop.Process = nil
	result := nbio.Process_Result{Exit: loop.Outcome.Exit}
	loop.On_Process(completion, result, loop.Outcome.Err)
	return true
}
