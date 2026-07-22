// Package timeout is the pure library tier of the timeout command: it reads a deadline
// and the command that deadline fences off the command line, runs that command through an
// injected child handle, and answers with the child's own exit code or with 124 when the
// deadline wins the race. It starts no process and signals none — package main binds both.
package timeout

import (
	"errors"
	"fmt"
	"io"
	"strconv"

	systime "local/james-orcales/shared/time"
)

// EXIT_USAGE is 2 by shell convention, keeping a command-line misuse distinct from a run
// that actually started.
const EXIT_USAGE = 2

// EXIT_TIMEOUT is 124, the code GNU coreutils' timeout answers with when the deadline
// elapses, so a caller can tell a child this program killed from one that failed by
// itself.
const EXIT_TIMEOUT = 124

// EXIT_START_FAILURE is 125, GNU timeout's code for a failure of the wrapper rather than
// of the command: nothing ran, so there is no exit code of the child's to report.
const EXIT_START_FAILURE = 125

// USAGE is the invocation written beside a usage error. The units are listed because they
// are the whole of the deadline's grammar.
const USAGE = "usage: timeout <deadline> <command> [argument...]\n" +
	"deadline: a positive whole number and one of ns, us, ms, s, m — as in 500ms"

// NANOSECONDS_MAX is the largest span a Duration counts. A magnitude is measured against
// it before being scaled, since the multiplication would otherwise wrap past the int64
// ceiling and turn an absurd deadline into a negative one that fires at once.
const NANOSECONDS_MAX systime.Duration = 1<<63 - 1

// Command is a parsed command line: the span the child is granted, and the command to run
// within it.
type Command struct {
	// Deadline is how long the child may run before it is terminated.
	Deadline systime.Duration
	// Path is the executable to run — the first argument behind the deadline.
	Path string
	// Arguments are the executable's own arguments, excluding its name.
	Arguments []string
}

// Child is the handle on a started process: a wait bounded by the deadline, and the
// signal that ends a process which outlived it. Package main binds the pair to a real
// process and a test to a script, so this tier races a deadline without owning one.
type Child struct {
	// Wait_Until waits for the process to exit, giving up once deadline has elapsed. It
	// reports the exit code and whether the process exited at all: false means the
	// deadline won and the process is still running.
	Wait_Until func(deadline systime.Duration) (status_code int, exited bool)
	// Terminate ends a process that outlived its deadline and reaps it, so the wrapper
	// never exits leaving the child it was asked to bound still running.
	Terminate func()
}

// Start launches path with arguments and hands back the handle on the running child. It
// is the one seam a real process enters through.
type Start func(path string, arguments []string) (child Child, err error)

// Main_Input carries the command line and the host bindings Main needs.
type Main_Input struct {
	// Arguments is the command line, including the program name.
	Arguments []string
	// Error_Output is where usage and this program's own diagnostics are written. The
	// child's output never passes through here — it goes straight to the real streams,
	// so a command under timeout reads as it would uninvoked.
	Error_Output io.Writer
	// Start launches the parsed command.
	Start Start
}

// Main parses the command line, runs the command under its deadline, and returns the
// process exit code: the child's own, or EXIT_TIMEOUT when the deadline elapsed first.
func Main(input *Main_Input) (status_code int) {
	command, parse_err := Command_Parse(input.Arguments)
	if parse_err != nil {
		fmt.Fprintf(input.Error_Output, "timeout: %v\n%s\n", parse_err, USAGE)
		return EXIT_USAGE
	}
	child, start_err := input.Start(command.Path, command.Arguments)
	if start_err != nil {
		fmt.Fprintf(input.Error_Output, "timeout: %v\n", start_err)
		return EXIT_START_FAILURE
	}
	child_code, exited := child.Wait_Until(command.Deadline)
	if exited {
		return child_code
	}
	child.Terminate()
	fmt.Fprintf(input.Error_Output, "timeout: %s: deadline exceeded\n", command.Path)
	return EXIT_TIMEOUT
}

// Command_Parse splits a command line — the program name, the deadline, then the command —
// into the deadline and the command it fences. Everything behind the deadline belongs to
// the child, so this program recognizes no flags of its own and a child's are left intact.
func Command_Parse(arguments []string) (command Command, err error) {
	argument_count := len(arguments)
	if argument_count < 2 {
		return Command{}, errors.New("no deadline given")
	}
	if argument_count < 3 {
		return Command{}, errors.New("no command given")
	}
	deadline, deadline_err := Duration_Parse(arguments[1])
	if deadline_err != nil {
		return Command{}, deadline_err
	}
	return Command{
		Deadline:  deadline,
		Path:      arguments[2],
		Arguments: arguments[3:],
	}, nil
}

// Duration_Parse converts a deadline such as "500ms" into the span it names. The unit is
// required: a bare number names no span, and guessing one would make the deadline a matter
// of this program's opinion rather than the caller's instruction.
func Duration_Parse(text string) (deadline systime.Duration, err error) {
	magnitude_text, unit_text := duration_split(text)
	unit, known := duration_unit(unit_text)
	if !known {
		return 0, fmt.Errorf("%q: unit must be one of ns, us, ms, s, m", text)
	}
	magnitude, magnitude_err := strconv.ParseInt(magnitude_text, 10, 64)
	if magnitude_err != nil {
		return 0, fmt.Errorf("%q: %q is not a whole number", text, magnitude_text)
	}
	if magnitude < 1 {
		return 0, fmt.Errorf("%q: a deadline is positive", text)
	}
	if magnitude > int64(NANOSECONDS_MAX)/int64(unit) {
		return 0, fmt.Errorf("%q: deadline is longer than a clock can count", text)
	}
	return systime.Duration(magnitude) * unit, nil
}

// Splits a deadline into its leading magnitude and its trailing unit, the unit being the
// run of letters the text ends in. Taking the whole run — rather than a fixed one or two
// characters — is what keeps 5ms from reading as five minutes with a stray s behind it.
func duration_split(text string) (magnitude string, unit string) {
	text_size := len(text)
	unit_index := text_size
	for unit_index > 0 {
		if !duration_letter(text[unit_index-1]) {
			break
		}
		unit_index--
	}
	return text[:unit_index], text[unit_index:]
}

// Returns the span a unit suffix names, and whether it is one of the five this program
// accepts. Hours and above are absent on purpose: a deadline that long is a scheduler's
// job, not a wrapper's.
func duration_unit(suffix string) (unit systime.Duration, known bool) {
	switch suffix {
	case "ns":
		return systime.NANOSECOND, true
	case "us":
		return systime.MICROSECOND, true
	case "ms":
		return systime.MILLISECOND, true
	case "s":
		return systime.SECOND, true
	case "m":
		return systime.MINUTE, true
	}
	return 0, false
}

// Reports whether a byte is a lowercase ASCII letter, the whole alphabet a unit is spelled
// from.
func duration_letter(character byte) (letter bool) {
	if character < 'a' {
		return false
	}
	return character <= 'z'
}
