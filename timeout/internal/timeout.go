// Package timeout is the pure library tier of the timeout command: it reads a deadline
// and the command that deadline fences off the command line, spawns that command through
// the injected IO surface under the deadline, and answers with the child's own exit code or
// with 124 when the deadline wins the race. Package main binds the real loop and pumps it.
package timeout

import (
	"errors"
	"unsafe"

	"local/james-orcales/shared/sim/nbio"
	"local/james-orcales/shared/sim/time"
	"local/james-orcales/shared/strconv"
)

// Deadline_Absent reports a command line that ends at the program name. A sentinel, so a
// usage failure allocates nothing.
var Deadline_Absent = errors.New("no deadline given")

// Command_Absent reports a deadline with no command behind it.
var Command_Absent = errors.New("no command given")

// Unit_Unknown reports a deadline whose unit is not one this program spells.
var Unit_Unknown = errors.New("deadline unit must be one of ns, us, ms, s, m")

// Magnitude_Malformed reports a deadline whose magnitude is not a whole number.
var Magnitude_Malformed = errors.New("deadline magnitude is not a whole number")

// Deadline_Not_Positive reports a deadline that has already elapsed.
var Deadline_Not_Positive = errors.New("deadline is not positive")

// Deadline_Too_Long reports a deadline past the span a Duration counts.
var Deadline_Too_Long = errors.New("deadline is longer than a clock can count")

// EXIT_USAGE is 2 by shell convention, keeping a command-line misuse distinct from a run
// that actually started.
const EXIT_USAGE Status_Code = 2

// EXIT_TIMEOUT is 124, the code GNU coreutils' timeout answers with when the deadline
// elapses, so a caller can tell a child this program killed from one that failed by
// itself.
const EXIT_TIMEOUT Status_Code = 124

// EXIT_START_FAILURE is 125, GNU timeout's code for a failure of the wrapper rather than
// of the command: nothing ran, so there is no exit code of the child's to report.
const EXIT_START_FAILURE Status_Code = 125

// USAGE is the invocation written beside a usage error. The units are listed because they
// are the whole of the deadline's grammar.
const USAGE = "usage: timeout <deadline> <command> [argument...]\n" +
	"deadline: a positive whole number and one of ns, us, ms, s, m — as in 500ms\n"

// MESSAGE_PREFIX starts every diagnostic.
const MESSAGE_PREFIX = "timeout: "

// MESSAGE_SIZE_MAXIMUM holds any diagnostic naming an ordinary path. The path comes off the
// command line, so a longer one gets MESSAGE_FULL rather than growing the store.
const MESSAGE_SIZE_MAXIMUM = 4096

// MESSAGE_FULL replaces a diagnostic past the store. A whole distinct line, because a
// truncated one would read as a different, complete diagnostic.
const MESSAGE_FULL = "timeout: diagnostic exceeds message storage\n"

// NANOSECONDS_MAX is the largest span a Duration counts. A magnitude is measured against
// it before being scaled, since the multiplication would otherwise wrap past the int64
// ceiling and turn an absurd deadline into a negative one that fires at once.
const NANOSECONDS_MAX time.Duration = 1<<63 - 1

// PHASE_READY is a child Rearm has yet to spawn.
const PHASE_READY Phase = 0

// PHASE_SUBMITTED is a child in flight on the loop.
const PHASE_SUBMITTED Phase = 1

// PHASE_RETIRED is a child whose outcome the callback recorded and Rearm has yet to judge.
const PHASE_RETIRED Phase = 2

// PHASE_REPORT is a diagnostic Rearm has yet to submit.
const PHASE_REPORT Phase = 3

// PHASE_REPORTED is a diagnostic in flight on the loop.
const PHASE_REPORTED Phase = 4

// PHASE_STOPPED is a finished run whose Status is final.
const PHASE_STOPPED Phase = 5

// Status_Code is the process exit code a run answers with.
type Status_Code int

// Phase is where the run stands; callbacks only move it forward, Rearm acts on it.
type Phase int

// Command is a parsed command line: the span the child is granted, and the command to run
// within it.
type Command struct {
	// Deadline is how long the child may run before its process group is killed.
	Deadline time.Duration
	// Path is the executable to run — the first argument behind the deadline.
	Path string
	// Arguments are the executable's own arguments, excluding its name.
	Arguments []string
}

// Process_Slot is the spawn submission. Completion sits first so the static callback
// recovers the slot, and through it the runner, from the completion pointer alone.
type Process_Slot struct {
	// Completion is the in-flight spawn.
	Completion nbio.Completion
	// Runner owns the slot.
	Runner *Runner
}

// Message_Slot is the diagnostic write submission, laid out like Process_Slot.
type Message_Slot struct {
	// Completion is the in-flight write.
	Completion nbio.Completion
	// Runner owns the slot.
	Runner *Runner
}

// Runner is one whole run's caller-owned state. Callbacks record into it; the root calls
// Rearm to turn what they recorded into the next submission, then pumps.
type Runner struct {
	// IO is the injected loop surface the child is spawned through.
	IO nbio.IO
	// Environment is the complete environment the child receives; its PATH finds the child.
	Environment []string
	// Stdout receives the child's standard output live.
	Stdout nbio.Stream
	// Stderr receives the child's standard error live.
	Stderr nbio.Stream
	// Error_Output receives this program's own diagnostic, never the child's output.
	Error_Output nbio.Stream
	// Message is the diagnostic store, MESSAGE_SIZE_MAXIMUM bytes; the root allocates it.
	Message []byte
	// Command is the parsed command line.
	Command Command
	// Phase is where the run stands.
	Phase Phase
	// Status is the exit code, final once Phase is PHASE_STOPPED.
	Status Status_Code
	// Report_Size is the byte count of the pending diagnostic in Message.
	Report_Size int
	// Result is the recorded outcome of the child.
	Result nbio.Process_Result
	// Result_Error is the recorded failure of the child, nil when it ran to exit.
	Result_Error error
	// Process is the spawn submission.
	Process Process_Slot
	// Report is the diagnostic write submission.
	Report Message_Slot
}

// Main parses the command line and readies the spawn. It returns without waiting: the root
// repeats Rearm and pumps.
func Main(runner *Runner, arguments []string) {
	runner.Process.Runner = runner
	runner.Report.Runner = runner
	command, parse_err := Command_Parse(arguments)
	if parse_err != nil {
		runner_fail(runner, EXIT_USAGE, MESSAGE_PREFIX, parse_err.Error(), "\n", USAGE)
		return
	}
	runner.Command = command
	runner.Phase = PHASE_READY
}

// Rearm turns what a callback recorded into the next submission and reports whether it
// did anything, so the root repeats it until the run needs the loop again.
func Rearm(runner *Runner) (progressed bool) {
	switch runner.Phase {
	case PHASE_READY:
		runner.Phase = PHASE_SUBMITTED
		nbio.IO_Spawn(runner.IO, &runner.Process.Completion, nbio.Process_Request{
			Path:        runner.Command.Path,
			Arguments:   runner.Command.Arguments,
			Environment: runner.Environment,
			// Signals forward by default; input is inherited so an interactive child
			// reads the terminal as it would run alone.
			Input_Inherited: true,
			Stdout:          runner.Stdout,
			Stderr:          runner.Stderr,
		}, runner.Command.Deadline, runner_process_retired)
		return true
	case PHASE_RETIRED:
		runner_judge(runner)
		return true
	case PHASE_REPORT:
		runner.Phase = PHASE_REPORTED
		nbio.Write(runner.Error_Output, &runner.Report.Completion,
			runner.Message[:runner.Report_Size], runner_message_retired)
		return true
	}
	return false
}

// Work_Queued reports whether a callback recorded work only Rearm can act on.
func Work_Queued(runner *Runner) (queued bool) {
	switch runner.Phase {
	case PHASE_READY, PHASE_RETIRED, PHASE_REPORT:
		return true
	}
	return false
}

// Stopped reports whether the run is over and Status is final.
func Stopped(runner *Runner) (stopped bool) {
	return runner.Phase == PHASE_STOPPED
}

// Command_Parse splits a command line — the program name, the deadline, then the command —
// into the deadline and the command it fences. Everything behind the deadline belongs to
// the child, so this program recognizes no flags of its own and a child's are left intact.
func Command_Parse(arguments []string) (command Command, err error) {
	argument_count := len(arguments)
	if argument_count < 2 {
		return Command{}, Deadline_Absent
	}
	if argument_count < 3 {
		return Command{}, Command_Absent
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
func Duration_Parse(text string) (deadline time.Duration, err error) {
	magnitude_text, unit_text := duration_split(text)
	unit, known := duration_unit(unit_text)
	if !known {
		return 0, Unit_Unknown
	}
	magnitude, magnitude_err := strconv.Parse_Integer(
		strconv.Text(magnitude_text), strconv.DECIMAL_BASE, strconv.BIT_SIZE_MAXIMUM)
	if magnitude_err != nil {
		return 0, Magnitude_Malformed
	}
	if magnitude < 1 {
		return 0, Deadline_Not_Positive
	}
	if int64(magnitude) > int64(NANOSECONDS_MAX)/int64(unit) {
		return 0, Deadline_Too_Long
	}
	return time.Duration(magnitude) * unit, nil
}

// Splits a deadline into its leading magnitude and its trailing unit, the unit being the
// run of letters the text ends in. Taking the whole run — rather than a fixed one or two
// characters — is what keeps 5ms from reading as five minutes with a stray s behind it.
func duration_split(text string) (magnitude string, unit string) {
	unit_count := len(text)
	for unit_count > 0 {
		if !duration_letter(text[unit_count-1]) {
			break
		}
		unit_count--
	}
	return text[:unit_count], text[unit_count:]
}

// Returns the span a unit suffix names, and whether it is one of the five this program
// accepts. Hours and above are absent on purpose: a deadline that long is a scheduler's
// job, not a wrapper's.
func duration_unit(suffix string) (unit time.Duration, known bool) {
	switch suffix {
	case "ns":
		return time.NANOSECOND, true
	case "us":
		return time.MICROSECOND, true
	case "ms":
		return time.MILLISECOND, true
	case "s":
		return time.SECOND, true
	case "m":
		return time.MINUTE, true
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

// A spawn error that is not the deadline means nothing ran: the backend reports a child
// that ran and was signalled through its exit code, never through an error.
func runner_judge(runner *Runner) {
	if errors.Is(runner.Result_Error, nbio.Deadline_Exceeded) {
		runner_fail(runner, EXIT_TIMEOUT,
			MESSAGE_PREFIX, runner.Command.Path, ": deadline exceeded\n")
		return
	}
	if runner.Result_Error != nil {
		reason := runner.Result_Error.Error()
		runner_fail(runner, EXIT_START_FAILURE,
			MESSAGE_PREFIX, runner.Command.Path, ": ", reason, "\n")
		return
	}
	runner.Status = Status_Code(runner.Result.Exit)
	runner.Phase = PHASE_STOPPED
}

// Renders into the message store, since the parts come off the command line and a store
// that grew would leave the run unbounded.
func runner_fail(runner *Runner, status_code Status_Code, parts ...string) {
	size := 0
	for _, part := range parts {
		size += len(part)
	}
	cursor := 0
	if size > len(runner.Message) {
		cursor = copy(runner.Message, MESSAGE_FULL)
	} else {
		for _, part := range parts {
			cursor += copy(runner.Message[cursor:], part)
		}
	}
	runner.Status = status_code
	runner.Report_Size = cursor
	runner.Phase = PHASE_REPORT
}

// Records only; Rearm judges, so no continuation runs inside the loop's own stack.
func runner_process_retired(
	completion *nbio.Completion, result nbio.Process_Result, err error,
) {
	slot := (*Process_Slot)(unsafe.Pointer(completion))
	slot.Runner.Result = result
	slot.Runner.Result_Error = err
	slot.Runner.Phase = PHASE_RETIRED
}

// A diagnostic that fails to land changes nothing: the status already says what happened.
func runner_message_retired(completion nbio.Completion_Handle) {
	slot := (*Message_Slot)(unsafe.Pointer(completion))
	slot.Runner.Phase = PHASE_STOPPED
}
