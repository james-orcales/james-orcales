package timeout_test

import (
	"errors"
	"strings"
	"testing"

	systime "local/james-orcales/shared/time"
	timeout "local/james-orcales/timeout/internal"
)

// Test_Duration_Units verifies every accepted unit scales its magnitude, and that a
// two-letter unit is read whole rather than as its one-letter tail.
func Test_Duration_Units(t *testing.T) {
	type unit_case struct {
		Text string
		Span systime.Duration
	}
	cases := []unit_case{
		{Text: "5ns", Span: 5 * systime.NANOSECOND},
		{Text: "5us", Span: 5 * systime.MICROSECOND},
		{Text: "5ms", Span: 5 * systime.MILLISECOND},
		{Text: "5s", Span: 5 * systime.SECOND},
		{Text: "5m", Span: 5 * systime.MINUTE},
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
	if err != nil {
		t.Fatalf("Command_Parse: %v", err)
	}
	if command.Deadline != 500*systime.MILLISECOND {
		t.Errorf("deadline = %d, want %d", command.Deadline, 500*systime.MILLISECOND)
	}
	if command.Path != "echo" {
		t.Errorf("path = %q, want %q", command.Path, "echo")
	}
	// -n is echo's flag: had the wrapper parsed it, the child would lose it.
	if strings.Join(command.Arguments, " ") != "-n hello" {
		t.Errorf("arguments = %v, want [-n hello]", command.Arguments)
	}
}

// Test_Command_Absent verifies a command line missing its deadline or its command is a
// usage error, reported as the usage exit code.
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
	code, problems := drive([]string{"timeout", "5s"}, nil)
	if code != 2 {
		t.Errorf("exit = %d, want 2 (usage)", code)
	}
	if problems == "" {
		t.Errorf("expected a usage message on standard error")
	}
}

// Test_Run_Exit_Code verifies a child that finishes inside its deadline yields its own
// exit code and is never terminated.
func Test_Run_Exit_Code(t *testing.T) {
	terminated := false
	code, _ := drive(
		[]string{"timeout", "1s", "false"}, scripted(7, true, &terminated))
	if code != 7 {
		t.Errorf("exit = %d, want 7 (the child's own)", code)
	}
	if terminated {
		t.Errorf("a child that exited on its own must not be terminated")
	}
}

// Test_Run_Timeout verifies a child still alive at its deadline is terminated, reported on
// standard error, and answered with 124.
func Test_Run_Timeout(t *testing.T) {
	terminated := false
	code, problems := drive(
		[]string{"timeout", "1ms", "sleep", "60"}, scripted(0, false, &terminated))
	if code != 124 {
		t.Errorf("exit = %d, want 124 (GNU timeout's code)", code)
	}
	if !terminated {
		t.Errorf("a child that outlived its deadline must be terminated")
	}
	if problems == "" {
		t.Errorf("expected the deadline to be reported on standard error")
	}
}

// Test_Run_Start_Failure verifies a command that never starts reports 125 and the reason,
// keeping the wrapper's own failure distinct from any exit code a child could return.
func Test_Run_Start_Failure(t *testing.T) {
	start := func(path string, arguments []string) (child timeout.Child, err error) {
		return timeout.Child{}, errors.New("executable file not found")
	}
	code, problems := drive([]string{"timeout", "1s", "nonesuch"}, start)
	if code != 125 {
		t.Errorf("exit = %d, want 125 (the wrapper's own failure)", code)
	}
	if !strings.Contains(problems, "not found") {
		t.Errorf("expected the start failure's reason, got %q", problems)
	}
}

// Runs Main over a scripted child and returns its exit code with everything the run wrote
// to standard error.
func drive(arguments []string, start timeout.Start) (status_code int, problems string) {
	problem_output := strings.Builder{}
	code := timeout.Main(&timeout.Main_Input{
		Arguments:    arguments,
		Error_Output: &problem_output,
		Start:        start,
	})
	return code, problem_output.String()
}

// Returns a Start whose child reports status_code and exited from its bounded wait, and
// records at terminated whether the run ended it.
func scripted(status_code int, exited bool, terminated *bool) (start timeout.Start) {
	return func(path string, arguments []string) (child timeout.Child, err error) {
		return timeout.Child{
			Wait_Until: func(deadline systime.Duration) (code int, finished bool) {
				return status_code, exited
			},
			Terminate: func() { *terminated = true },
		}, nil
	}
}
