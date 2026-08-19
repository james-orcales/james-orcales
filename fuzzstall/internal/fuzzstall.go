// Package fuzzstall is the pure library tier of the fuzzstall command: it reads a stall
// window and the `go test` arguments that window bounds off the command line, follows the
// run's progress lines through an injected handle, and answers with the run's own exit code
// or with 124 once the count of new interesting inputs stops moving. It starts no process
// and reads no clock — package main binds both.
package fuzzstall

import (
	"errors"
	"fmt"
	"io"
	"strconv"

	invariant "local/james-orcales/shared/invariant/default"
	systime "local/james-orcales/shared/simulation/time"
)

// EXIT_SUCCESS is the process exit code of a run that passed inside its window.
const EXIT_SUCCESS Status_Code = 0

// EXIT_USAGE is 2 by shell convention, keeping a command-line misuse distinct from a run
// that actually started.
const EXIT_USAGE Status_Code = 2

// EXIT_SATURATED is 124, the code GNU coreutils' timeout answers with when a span the
// wrapper imposed ended the command rather than the command ending itself. Saturation is
// that same event, thus it carries that same code and one code keeps one meaning.
const EXIT_SATURATED Status_Code = 124

// EXIT_START_FAILURE is 125, GNU timeout's code for a failure of the wrapper rather than
// of the command: nothing ran, thus there is no exit code of the run's to report.
const EXIT_START_FAILURE Status_Code = 125

// GO is the toolchain this program wraps. It carries no directory, so the PATH picks the
// same toolchain the caller already builds with.
const GO Toolchain = "go"

// TEST is the one `go` subcommand this program supplies. It supplies no more than that:
// only the caller knows which target to fuzz, thus -fuzz stays the caller's to write.
const TEST = "test"

// USAGE is the invocation written beside a usage error. The grammar of a rate is the whole of
// what is written here.
const USAGE = "usage: fuzzstall <rate> <go test argument...>\n" +
	"rate: new interesting inputs per window, as in 90/30s\n" +
	"window alone: a positive whole number of seconds, which asks for one input — as in 30s\n" +
	"example: fuzzstall 90/30s -fuzz=Fuzz_Parse ./shared/cli"

// INTERESTING_FIELD is the field a `go test -fuzz` progress line writes its count behind.
// It is the one field of that line this program reads.
const INTERESTING_FIELD = "new interesting: "

// LINE_SIZE_MAX bounds the bytes one assembled line holds. A run's output is unvalidated
// input, thus a line that grows without a newline must cost this program no more memory
// than this. A progress line is near 90 bytes, so the bound holds one whole and stays free.
const LINE_SIZE_MAX = 512

// CHUNK_SIZE_MAX bounds one read of the run's output. A read this wide collects many
// progress lines at once and still holds a fixed cost.
const CHUNK_SIZE_MAX = 4096

// DIGIT_COUNT_MAX is the most digits a count may carry. Nineteen digits reach past what an
// int64 counts, thus eighteen is the widest run that cannot wrap.
const DIGIT_COUNT_MAX = 18

// BUFFER_SIZE_MIN is an empty buffer, which is where a line starts and where it returns
// once the line is handed over.
const BUFFER_SIZE_MIN = 0

// BYTE_OFFSET_ABSENT is the offset a search answers with when it found nothing. It is below
// every real offset, so a caller that forgets the check reads no byte at all.
const BYTE_OFFSET_ABSENT = -1

// BYTE_OFFSET_MAX is the largest offset a search over one read returns: the last byte of the
// widest chunk, since an offset names a byte that is there.
const BYTE_OFFSET_MAX = CHUNK_SIZE_MAX - 1

// REST_SIZE_MAX is the widest remainder one read carries behind the line it closed: a read
// without the newline that closed the line.
const REST_SIZE_MAX = CHUNK_SIZE_MAX - 1

// FIELD_OFFSET_ABSENT is the offset a field search answers with when the line carries no such
// field.
const FIELD_OFFSET_ABSENT Field_Offset = -1

// FIELD_OFFSET_MAX is the furthest into a line the interesting field can start and still fit:
// a field that started past this would run off the end of the widest line there is.
const FIELD_OFFSET_MAX = LINE_SIZE_MAX - len(INTERESTING_FIELD)

// STATUS_CODE_MIN is the exit code of a run that succeeded.
const STATUS_CODE_MIN = 0

// STATUS_CODE_MAX is the largest exit code a shell reads, because a wait status carries the
// code in one byte.
const STATUS_CODE_MAX = 255

// INTERESTING_COUNT_MIN is the count a run reports before it found anything at all.
const INTERESTING_COUNT_MIN = 0

// INTERESTING_COUNT_MAX is the largest count the digits of a progress line spell, which is
// what DIGIT_COUNT_MAX admits.
const INTERESTING_COUNT_MAX = 999999999999999999

// TOOLCHAIN_BYTES is the length of the go command's name, which is the whole domain of a
// toolchain this program starts.
const TOOLCHAIN_BYTES = 2

// COMMAND_LINE_COUNT_MIN is the program name, which every command line carries.
const COMMAND_LINE_COUNT_MIN = 1

// CHECKED_COMMAND_LINE_COUNT_MIN is the program name, the window, and the one argument the
// caller must leave for `go test`.
const CHECKED_COMMAND_LINE_COUNT_MIN = 3

// ARGUMENT_COUNT_MIN is the `test` subcommand and the one argument the caller must add to
// it, because `go test` with no target names no package to fuzz.
const ARGUMENT_COUNT_MIN = 2

// COMMAND_LINE_COUNT_MAX bounds a command line. A `go test` invocation names a handful of
// flags and a handful of packages, and a caller past this bound is passing a file list that
// belongs in a package pattern instead.
const COMMAND_LINE_COUNT_MAX = 256

// ARGUMENT_COUNT_MAX bounds what reaches the toolchain. A command line loses its own program
// name and its window, and gains the `test` subcommand, thus it is one below the line itself.
const ARGUMENT_COUNT_MAX = COMMAND_LINE_COUNT_MAX - 1

// TEXT_BYTES_MIN is empty text, which is what an argument the caller left blank carries.
const TEXT_BYTES_MIN = 0

// TEXT_BYTES_MAX bounds one argument off the command line, before anything validated it.
const TEXT_BYTES_MAX = 4096

// SECOND_SUFFIX is the one unit a window is written in. It is the whole of the grammar: a
// window names a count of seconds, thus no other suffix is read as a span.
const SECOND_SUFFIX Text = "s"

// RATE_SEPARATOR stands between a rate's quota and its window, as in 90/30s.
const RATE_SEPARATOR = '/'

// QUOTA_TEXT_ONE is the quota a rate written as a bare window carries. A window alone has
// always meant one new interesting input inside it, thus that is what it keeps meaning.
const QUOTA_TEXT_ONE Quota_Text = "1"

// QUOTA_TEXT_BYTES_MAX is the widest a quota text is: a rate of that width whose last byte is
// the separator, which leaves every byte before it to the quota.
const QUOTA_TEXT_BYTES_MAX = TEXT_BYTES_MAX - 1

// QUOTA_MIN is the smallest quota there is: one new interesting input inside the window,
// which is the least progress that is still progress.
const QUOTA_MIN = 1

// QUOTA_MAX bounds a quota. A run required to add a million corpus entries inside one window
// is past what this program needs to name.
const QUOTA_MAX = 1000000

// RATE_TEXT_BYTES_MIN is the shortest rate a caller can write: a bare one-digit window.
const RATE_TEXT_BYTES_MIN = WINDOW_TEXT_BYTES_ONE

// RATE_TEXT_BYTES_MAX is the longest rate a caller can write: the seven digits of QUOTA_MAX,
// the separator, and the widest window.
const RATE_TEXT_BYTES_MAX = 7 + 1 + WINDOW_TEXT_BYTES_FOUR

// MAGNITUDE_MIN is the shortest window a caller can ask for, counted in seconds. A window
// names a span, thus the shortest one there is counts one second.
const MAGNITUDE_MIN = 1

// MAGNITUDE_MAX is the longest window a caller can ask for, counted in seconds. A magnitude
// is measured against it before being scaled, since the multiplication would otherwise wrap
// past the int64 ceiling and turn an absurd window into a negative one that fires at once. An
// hour is the ceiling because a run that found nothing new for a whole hour saturated long
// before the hour was out.
const MAGNITUDE_MAX = 3600

// WINDOW_TEXT_BYTES_ONE is a one-digit window and its suffix, the shortest one there is.
const WINDOW_TEXT_BYTES_ONE = 2

// WINDOW_TEXT_BYTES_TWO is a two-digit window and its suffix.
const WINDOW_TEXT_BYTES_TWO = 3

// WINDOW_TEXT_BYTES_THREE is a three-digit window and its suffix.
const WINDOW_TEXT_BYTES_THREE = 4

// WINDOW_TEXT_BYTES_FOUR is a four-digit window and its suffix, which is the width
// MAGNITUDE_MAX takes and so the longest one there is.
const WINDOW_TEXT_BYTES_FOUR = 5

// WINDOW_MIN is the shortest window a caller can ask for.
const WINDOW_MIN = Window(MAGNITUDE_MIN) * Window(systime.SECOND)

// WINDOW_MAX is the longest window a caller can ask for.
const WINDOW_MAX = Window(MAGNITUDE_MAX) * Window(systime.SECOND)

// DEADLINE_MIN is the shortest wait there is: one nanosecond of a window still to run. It is
// far below WINDOW_MIN because a wait is what a window has left, and a window is nearly over
// long before it is over.
const DEADLINE_MIN Deadline = 1

// DEADLINE_MAX is the longest wait there is, which is the longest window there is.
const DEADLINE_MAX = Deadline(WINDOW_MAX)

// READING_MIN is the first reading a monotonic clock gives.
const READING_MIN Reading = 0

// READING_MAX is a century of uptime, which is past any host this program runs on and still
// narrow enough to name a domain a test can reach both ends of.
const READING_MAX = Reading(100 * 365 * systime.DAY)

// Main parses the command line, follows the `go test` run it names, and returns the process
// exit code: the run's own, or EXIT_SATURATED once the count of new interesting inputs
// holds still for the whole window.
func Main(input *Main_Input) (status_code Status_Code) {
	defer func() { Status_Code_Invariants(status_code, "Main.status_code") }()
	Main_Input_Invariants(*input, "Main.input")
	check_err := Command_Check(input.Arguments)
	if check_err != nil {
		fmt.Fprintf(input.Error_Output, "fuzzstall: %v\n%s\n", check_err, USAGE)
		return EXIT_USAGE
	}
	command := Command_Parse(Checked_Command_Line(input.Arguments))
	child, start_err := input.Start(command.Path, command.Arguments)
	if start_err != nil {
		fmt.Fprintf(input.Error_Output, "fuzzstall: %v\n", start_err)
		return EXIT_START_FAILURE
	}
	// The handle comes from outside this tier, thus it is checked here rather than trusted:
	// a Start that answered with a hole would fail later, inside the watch, where the run it
	// left running is already the thing that cannot be stopped.
	Child_Invariants(child, "Main.child")
	progress := Progress{Rate: command.Rate}
	reader := Line_Reader{}
	event := EVENT_DEADLINE
	for event != EVENT_END {
		chunk, reported := child.Next_Output(Progress_Deadline(progress, input.Now()))
		event = reported
		if event == EVENT_OUTPUT {
			input.Output.Write(chunk)
			progress_read(&progress, &reader, chunk, input.Now())
		}
		if !Progress_Saturated(progress, input.Now()) {
			continue
		}
		child.Terminate()
		fmt.Fprintf(input.Error_Output,
			"fuzzstall: saturated: the run fell below the rate %s (%d found)\n",
			command.Rate_Text, progress.Interesting_Count)
		return EXIT_SATURATED
	}
	return child.Wait()
}

// Status_Code is one process exit code.
type Status_Code int

// Status_Code_Invariants holds an exit code to what a wait status carries.
func Status_Code_Invariants(status_code Status_Code, namespace invariant.Namespace) {
	invariant.Tree(status_code, namespace).
		Range_Int(int(status_code), STATUS_CODE_MIN, STATUS_CODE_MAX).
		Ensure()
}

// Window is the span a caller asked for: how long a run may go without a new interesting
// input before it counts as saturated.
type Window systime.Duration

// Window_Invariants bounds a window to a span this program measures. It admits no zero,
// because a window of no span is a window that was already over when the run started, and
// Duration_Check rejects one.
func Window_Invariants(window Window, namespace invariant.Namespace) {
	invariant.Tree(window, namespace).
		Range_Int64(int64(window), int64(WINDOW_MIN), int64(WINDOW_MAX)).
		Ensure()
}

// Deadline is how long one wait on a run's output may last. It is a separate type from Window
// because it admits no span at all and a window never does: one bound cannot hold both.
type Deadline systime.Duration

// Deadline_Invariants bounds one bounded wait.
func Deadline_Invariants(deadline Deadline, namespace invariant.Namespace) {
	invariant.Tree(deadline, namespace).
		Range_Int64(int64(deadline), int64(DEADLINE_MIN), int64(DEADLINE_MAX)).
		Ensure()
}

// Reading is one reading of the injected monotonic clock. Only the difference between two
// readings is used, thus the epoch it counts from does not matter.
type Reading systime.Moment

// Reading_Invariants bounds a clock reading to a century of uptime.
func Reading_Invariants(reading Reading, namespace invariant.Namespace) {
	invariant.Tree(reading, namespace).
		Range_Int64(int64(reading), int64(READING_MIN), int64(READING_MAX)).
		Ensure()
}

// Event is what a bounded wait on a run's output produced.
type Event uint8

// Event_Invariants holds an event to the three outcomes a bounded wait has.
func Event_Invariants(event Event, namespace invariant.Namespace) {
	invariant.Tree(event, namespace).
		Enum_3_Uint8(
			uint8(event), uint8(EVENT_OUTPUT), uint8(EVENT_DEADLINE), uint8(EVENT_END)).
		Ensure()
}

// EVENT_OUTPUT reports that the run wrote bytes.
const EVENT_OUTPUT Event = 0

// EVENT_DEADLINE reports that the wait ran out before the run wrote anything. It is what
// makes a run that goes silent judgeable rather than waited on forever.
const EVENT_DEADLINE Event = 1

// EVENT_END reports that the run closed its output and its exit code can be collected.
const EVENT_END Event = 2

// Toolchain is the command that runs a Go test.
type Toolchain string

// Toolchain_Invariants states that the only toolchain this program starts is the go
// command, whose name is the whole of the domain and so admits one length alone.
func Toolchain_Invariants(path Toolchain, _ invariant.Namespace) {
	invariant.Always(len(path) == TOOLCHAIN_BYTES, "The toolchain name is two bytes.")
	invariant.Always(path == GO, "The toolchain is the go command.")
}

// Command_Line is a command line as the operating system handed it over, before anything
// validated it.
type Command_Line []string

// Command_Line_Invariants bounds a command line before it is read.
func Command_Line_Invariants(command_line Command_Line, namespace invariant.Namespace) {
	invariant.Tree(command_line, namespace).
		Range_Int(len(command_line), COMMAND_LINE_COUNT_MIN, COMMAND_LINE_COUNT_MAX).
		Ensure()
}

// Checked_Command_Line is a command line that already passed Command_Check: it carries a
// window that names a span and at least one argument behind it for `go test`.
type Checked_Command_Line []string

// Checked_Command_Line_Invariants states the parts a checked command line must carry.
func Checked_Command_Line_Invariants(
	command_line Checked_Command_Line, namespace invariant.Namespace,
) {
	invariant.Tree(command_line, namespace).
		Range_Int(
			len(command_line), CHECKED_COMMAND_LINE_COUNT_MIN, COMMAND_LINE_COUNT_MAX).
		Ensure()
}

// Arguments are the toolchain's own arguments, excluding its name.
type Arguments []string

// Arguments_Invariants bounds the arguments handed to the toolchain.
func Arguments_Invariants(arguments Arguments, namespace invariant.Namespace) {
	invariant.Tree(arguments, namespace).
		Range_Int(len(arguments), ARGUMENT_COUNT_MIN, ARGUMENT_COUNT_MAX).
		Ensure()
}

// Text is a run of bytes off the command line, before anything validated it.
type Text string

// Text_Invariants bounds unvalidated command-line text.
func Text_Invariants(text Text, namespace invariant.Namespace) {
	invariant.Tree(text, namespace).
		Range_Int(len(text), TEXT_BYTES_MIN, TEXT_BYTES_MAX).
		Ensure()
}

// Anchor_Count is the count a run stood at when it last met its quota. It is a type of its
// own because it and the running count are two readings of one quantity, and one chain states
// one type one time. Both are bounded by the same constants, which is what says they are.
type Anchor_Count int64

// Anchor_Count_Invariants bounds the count a run last met its quota at.
func Anchor_Count_Invariants(count Anchor_Count, namespace invariant.Namespace) {
	invariant.Tree(count, namespace).
		Range_Int64(int64(count), INTERESTING_COUNT_MIN, INTERESTING_COUNT_MAX).
		Ensure()
}

// Quota_Text is the run of bytes before a rate's separator. It is never as wide as the rate
// itself, because the separator is a byte of that rate this no longer holds, and a rate with
// no separator carries the standing quota text instead of any part of itself.
type Quota_Text string

// Quota_Text_Invariants bounds the text a quota is read from.
func Quota_Text_Invariants(quota_text Quota_Text, namespace invariant.Namespace) {
	invariant.Tree(quota_text, namespace).
		Range_Int(len(quota_text), TEXT_BYTES_MIN, QUOTA_TEXT_BYTES_MAX).
		Ensure()
}

// Quota is how many new interesting inputs a window must carry for a run to keep going.
type Quota int64

// Quota_Invariants bounds the least progress a caller can require.
func Quota_Invariants(quota Quota, namespace invariant.Namespace) {
	invariant.Tree(quota, namespace).
		Range_Int64(int64(quota), QUOTA_MIN, QUOTA_MAX).
		Ensure()
}

// Rate is the least progress a run must keep: a count of new interesting inputs, and the
// window that count must arrive in.
type Rate struct {
	// Quota is how many new interesting inputs the window must carry.
	Quota Quota
	// Window is the span the quota is measured over.
	Window Window
}

// Rate_Invariants states both parts of the least progress a run must keep.
func Rate_Invariants(rate Rate, namespace invariant.Namespace) {
	Quota_Invariants(rate.Quota, namespace)
	Window_Invariants(rate.Window, namespace)
}

// Rate_Text is a rate as the caller spelled it, after it parsed.
type Rate_Text string

// Rate_Text_Invariants bounds a rate that already named a minimum.
func Rate_Text_Invariants(rate_text Rate_Text, namespace invariant.Namespace) {
	invariant.Tree(rate_text, namespace).
		Range_Int(len(rate_text), RATE_TEXT_BYTES_MIN, RATE_TEXT_BYTES_MAX).
		Ensure()
}

// Window_Text is a window as the caller spelled it, after it parsed.
type Window_Text string

// Window_Text_Invariants states the four widths a window that already named a span has. It is
// a set and not an interval, because a magnitude carries one, two, three, or four digits and
// the suffix is one byte behind it.
func Window_Text_Invariants(window_text Window_Text, namespace invariant.Namespace) {
	invariant.Tree(window_text, namespace).
		Enum_4_Int(
			len(window_text), WINDOW_TEXT_BYTES_ONE, WINDOW_TEXT_BYTES_TWO,
			WINDOW_TEXT_BYTES_THREE, WINDOW_TEXT_BYTES_FOUR).
		Ensure()
}

// Line is one whole line of a run's output.
type Line []byte

// Line_Invariants bounds one assembled line.
func Line_Invariants(line Line, namespace invariant.Namespace) {
	invariant.Tree(line, namespace).
		Range_Int(len(line), BUFFER_SIZE_MIN, LINE_SIZE_MAX).
		Ensure()
}

// Chunk is the bytes one read of a run's output produced. It starts and stops at any byte,
// because a pipe carries no line structure of its own.
type Chunk []byte

// Chunk_Invariants bounds one read of a run's output.
func Chunk_Invariants(chunk Chunk, namespace invariant.Namespace) {
	invariant.Tree(chunk, namespace).
		Range_Int(len(chunk), BUFFER_SIZE_MIN, CHUNK_SIZE_MAX).
		Ensure()
}

// Rest is what one read carries behind the line it closed. It is never a whole read, because
// the newline it closed on is a byte of that read this no longer holds.
type Rest []byte

// Rest_Invariants bounds what a read carries behind the line it closed.
func Rest_Invariants(rest Rest, namespace invariant.Namespace) {
	invariant.Tree(rest, namespace).
		Range_Int(len(rest), BUFFER_SIZE_MIN, REST_SIZE_MAX).
		Ensure()
}

// Digits is what a line carries behind the interesting field. It is never a whole line,
// because the field itself is bytes of that line this no longer holds.
type Digits []byte

// Digits_Invariants bounds what a line carries behind the interesting field.
func Digits_Invariants(digits Digits, namespace invariant.Namespace) {
	invariant.Tree(digits, namespace).
		Range_Int(len(digits), BUFFER_SIZE_MIN, FIELD_OFFSET_MAX).
		Ensure()
}

// Byte_Offset is a position in a line or a chunk, or BYTE_OFFSET_ABSENT when a search found
// nothing.
type Byte_Offset int

// Byte_Offset_Invariants bounds a position inside one read.
func Byte_Offset_Invariants(offset Byte_Offset, namespace invariant.Namespace) {
	invariant.Tree(offset, namespace).
		Range_Int(int(offset), BYTE_OFFSET_ABSENT, BYTE_OFFSET_MAX).
		Ensure()
}

// Field_Offset is where the interesting field starts in a line, or FIELD_OFFSET_ABSENT when
// the line carries no such field.
type Field_Offset int

// Field_Offset_Invariants bounds where a field can start and still fit in a line.
func Field_Offset_Invariants(offset Field_Offset, namespace invariant.Namespace) {
	invariant.Tree(offset, namespace).
		Range_Int(int(offset), int(FIELD_OFFSET_ABSENT), FIELD_OFFSET_MAX).
		Ensure()
}

// Buffer_Size is how many bytes of a line buffer are filled.
type Buffer_Size int

// Buffer_Size_Invariants bounds the filled part of a line buffer.
func Buffer_Size_Invariants(buffer_size Buffer_Size, namespace invariant.Namespace) {
	invariant.Tree(buffer_size, namespace).
		Range_Int(int(buffer_size), BUFFER_SIZE_MIN, LINE_SIZE_MAX).
		Ensure()
}

// Interesting_Count is how many new interesting inputs a run reported.
type Interesting_Count int64

// Interesting_Count_Invariants bounds a count to what a progress line spells.
func Interesting_Count_Invariants(
	interesting_count Interesting_Count, namespace invariant.Namespace,
) {
	invariant.Tree(interesting_count, namespace).
		Range_Int64(int64(interesting_count), INTERESTING_COUNT_MIN, INTERESTING_COUNT_MAX).
		Ensure()
}

// Started reports whether a run reported a count at all.
type Started bool

// Started_Invariants states both states a run's start has.
func Started_Invariants(started Started, namespace invariant.Namespace) {
	invariant.Tree(started, namespace).
		Sometimes(bool(started), "The run reported a progress line.").
		Ensure()
}

// Saturated reports whether a run found nothing new for its whole window.
type Saturated bool

// Saturated_Invariants states both outcomes of the saturation judgment.
func Saturated_Invariants(saturated Saturated, namespace invariant.Namespace) {
	invariant.Tree(saturated, namespace).
		Sometimes(bool(saturated), "The run saturated.").
		Ensure()
}

// Found reports whether a search found what it looked for.
type Found bool

// Found_Invariants states both outcomes of a search.
func Found_Invariants(found Found, namespace invariant.Namespace) {
	invariant.Tree(found, namespace).
		Sometimes(bool(found), "A search found what it looked for.").
		Ensure()
}

// Complete reports whether a line reached its newline.
type Complete bool

// Complete_Invariants states both states of an assembled line.
func Complete_Invariants(complete Complete, namespace invariant.Namespace) {
	invariant.Tree(complete, namespace).
		Sometimes(bool(complete), "A line reached its newline.").
		Ensure()
}

// Truncated reports whether a line outgrew the buffer that assembles it.
type Truncated bool

// Truncated_Invariants states both states of a line's fit.
func Truncated_Invariants(truncated Truncated, namespace invariant.Namespace) {
	invariant.Tree(truncated, namespace).
		Sometimes(bool(truncated), "A line outgrew its buffer.").
		Ensure()
}

// Command is a parsed command line: the span a run may go without progress, and the
// `go test` invocation to watch over that span.
type Command struct {
	// Rate is the least progress the run must keep.
	Rate Rate
	// Rate_Text is the rate as the caller spelled it. The saturation report names it in the
	// caller's own words rather than in nanoseconds.
	Rate_Text Rate_Text
	// Path is the toolchain to run.
	Path Toolchain
	// Arguments are the toolchain's arguments, excluding its name.
	Arguments Arguments
}

// Command_Invariants states each part of a command line that already parsed.
func Command_Invariants(command Command, namespace invariant.Namespace) {
	Rate_Invariants(command.Rate, namespace)
	Rate_Text_Invariants(command.Rate_Text, namespace)
	Toolchain_Invariants(command.Path, namespace)
	Arguments_Invariants(command.Arguments, namespace)
}

// Child is the handle on a started run: a bounded wait on its next output, the signal that
// ends a run this program judged saturated, and the wait that collects its exit code.
// Package main binds the three to a real process and a test to a script, so this tier
// follows a run without owning one.
type Child struct {
	// Next_Output waits for the next bytes the run writes.
	Next_Output Next_Output
	// Terminate ends a saturated run and reaps it, so this program never exits and leaves
	// the fuzzing it was watching still running.
	Terminate Terminate
	// Wait collects the exit code of a run that ended by itself.
	Wait Wait
}

// Child_Invariants rejects a handle that cannot follow, stop, or reap its run.
func Child_Invariants(child Child, _ invariant.Namespace) {
	invariant.Always(child.Next_Output != nil, "A started run can be read.")
	invariant.Always(child.Terminate != nil, "A started run can be terminated.")
	invariant.Always(child.Wait != nil, "A started run can be reaped.")
}

// Next_Output waits for the next bytes a run writes, giving up once deadline has elapsed.
// The bound is what lets a run that writes nothing at all still be judged.
type Next_Output func(deadline Deadline) (chunk Chunk, event Event)

// Terminate ends a run and reaps it.
type Terminate func()

// Wait collects the exit code of a run that ended by itself.
type Wait func() (status_code Status_Code)

// Child_Over returns the handle on a run that never started: it is over, it kills nothing,
// and it reports this program's own failure. A Child always answers, thus a failed start
// hands back a handle rather than a hole every caller would have to check.
func Child_Over() (child Child) {
	defer func() { Child_Invariants(child, "Child_Over.child") }()
	return Child{
		Next_Output: func(_ Deadline) (chunk Chunk, event Event) {
			return nil, EVENT_END
		},
		Terminate: func() {},
		Wait:      func() (status_code Status_Code) { return EXIT_START_FAILURE },
	}
}

// Start launches path with arguments and hands back the handle on the running child. It is
// the one seam a real process enters through.
type Start func(path Toolchain, arguments Arguments) (child Child, err error)

// Main_Input carries the command line and the host bindings Main needs.
type Main_Input struct {
	// Arguments is the command line, including the program name.
	Arguments Command_Line
	// Output is where the run's own output is echoed byte for byte, so a run watched
	// through this program reads as the same run started directly.
	Output io.Writer
	// Error_Output is where usage and this program's own diagnostics are written.
	Error_Output io.Writer
	// Now reads a monotonic clock. Only the difference between two readings is used, thus
	// the epoch it counts from does not matter.
	Now func() (reading Reading)
	// Start launches the parsed `go test`.
	Start Start
}

// Main_Input_Invariants rejects a composition root that left a binding out.
func Main_Input_Invariants(input Main_Input, namespace invariant.Namespace) {
	Command_Line_Invariants(input.Arguments, namespace)
	invariant.Always(input.Output != nil, "A run has somewhere to echo its output.")
	invariant.Always(input.Error_Output != nil, "This program has somewhere to report.")
	invariant.Always(input.Now != nil, "This program has a clock.")
	invariant.Always(input.Start != nil, "This program can start a run.")
}

// Progress is a run's progress: the highest count of new interesting inputs it reported, and
// where it last stood when it met its quota.
type Progress struct {
	// Rate is the least progress the run must keep.
	Rate Rate
	// Interesting_Count is the highest count the run reported so far.
	Interesting_Count Interesting_Count
	// Started reports whether a progress line arrived at all. Until one has, the run is
	// still gathering baseline coverage and no count exists that could hold still.
	Started Started
	// Anchor_Count is the count the run stood at when it last met its quota.
	Anchor_Count Anchor_Count
	// Anchor_At is the moment it last met that quota, or the moment the first progress line
	// arrived. The window is measured from here, thus meeting the quota starts it again.
	Anchor_At Reading
}

// Progress_Invariants states each part of a run's progress record.
func Progress_Invariants(progress Progress, namespace invariant.Namespace) {
	Rate_Invariants(progress.Rate, namespace)
	Interesting_Count_Invariants(progress.Interesting_Count, namespace)
	Started_Invariants(progress.Started, namespace)
	Anchor_Count_Invariants(progress.Anchor_Count, namespace)
	Reading_Invariants(progress.Anchor_At, namespace)
}

// Line_Reader assembles output lines out of chunks that start and stop at any byte.
type Line_Reader struct {
	// Buffer holds the line being assembled. It is a fixed array because a run's output is
	// unvalidated input: a line that grows without a newline must cost no more than this.
	Buffer [LINE_SIZE_MAX]byte
	// Buffer_Size is how many bytes of Buffer the line being assembled fills.
	Buffer_Size Buffer_Size
	// Truncated reports that the line outgrew Buffer and its remaining bytes were dropped.
	Truncated Truncated
}

// Line_Reader_Invariants states the fill and the fit of the line being assembled.
func Line_Reader_Invariants(reader Line_Reader, namespace invariant.Namespace) {
	Buffer_Size_Invariants(reader.Buffer_Size, namespace)
	Truncated_Invariants(reader.Truncated, namespace)
}

// Command_Parse splits a command line — the program name, the window, then the `go test`
// arguments — into the window and the invocation it bounds. Everything behind the window
// belongs to `go test`, thus this program recognizes no flag of its own and the run keeps
// every flag the caller wrote for it.
func Command_Parse(command_line Checked_Command_Line) (command Command) {
	defer func() { Command_Invariants(command, "Command_Parse.command") }()
	Checked_Command_Line_Invariants(command_line, "Command_Parse.command_line")
	// Command_Check already read this rate, thus the error here cannot fire. The check is
	// what makes this parse total, which is what lets the command carry an invariant set no
	// failed parse could satisfy.
	return Command{
		Rate:      Rate_Parse(Rate_Text(command_line[1])),
		Rate_Text: Rate_Text(command_line[1]),
		Path:      GO,
		Arguments: append(Arguments{TEST}, command_line[2:]...),
	}
}

// Command_Check reports why a command line names no run to watch, and nil once it names
// one. It is separate from Command_Parse because a parse that can fail has no whole command
// to hand back on the failure, and a half-built one states nothing a caller can rely on.
func Command_Check(command_line Command_Line) (err error) {
	Command_Line_Invariants(command_line, "Command_Check.command_line")
	argument_count := len(command_line)
	if argument_count < 2 {
		return errors.New("no rate given")
	}
	if argument_count < CHECKED_COMMAND_LINE_COUNT_MIN {
		return errors.New("no go test arguments given")
	}
	return Rate_Check(Text(command_line[1]))
}

// Duration_Parse converts a window such as "60s" into the span it names. The unit is
// required: a bare number names no span, and guessing one would make the window a matter of
// this program's opinion rather than the caller's instruction.
func Duration_Parse(window_text Window_Text) (window Window) {
	defer func() { Window_Invariants(window, "Duration_Parse.window") }()
	Window_Text_Invariants(window_text, "Duration_Parse.window_text")
	// Duration_Check already read this text, thus every guard it holds still holds here.
	// The check is what makes this parse total, which is what lets a window carry a bound
	// no failed parse could satisfy, and what makes the text here a checked one.
	magnitude_text, _ := duration_split(Text(window_text))
	magnitude, _ := strconv.ParseInt(string(magnitude_text), 10, 64)
	return Window(magnitude) * Window(systime.SECOND)
}

// Rate_Parse converts a checked rate such as "90/30s" into the least progress it names.
func Rate_Parse(rate_text Rate_Text) (rate Rate) {
	defer func() { Rate_Invariants(rate, "Rate_Parse.rate") }()
	Rate_Text_Invariants(rate_text, "Rate_Parse.rate_text")
	// Rate_Check already read this text, thus every guard it holds still holds here.
	quota_text, window_text := rate_split(Text(rate_text))
	quota, _ := strconv.ParseInt(string(quota_text), 10, 64)
	return Rate{
		Quota:  Quota(quota),
		Window: Duration_Parse(Window_Text(window_text)),
	}
}

// Rate_Check reports why a rate names no minimum this program measures, and nil once it names
// one. A rate is a quota and the window it must arrive in, thus both are checked here.
func Rate_Check(text Text) (err error) {
	Text_Invariants(text, "Rate_Check.text")
	quota_text, window_text := rate_split(text)
	quota, quota_err := strconv.ParseInt(string(quota_text), 10, 64)
	if quota_err != nil {
		return fmt.Errorf(
			"%q: %q is not a whole number", string(text), string(quota_text))
	}
	if quota < QUOTA_MIN {
		return fmt.Errorf("%q: a quota is positive", string(text))
	}
	if quota > QUOTA_MAX {
		return fmt.Errorf("%q: a quota is at most %d", string(text), QUOTA_MAX)
	}
	window_err := Duration_Check(window_text)
	if window_err != nil {
		return fmt.Errorf("%q: %w", string(text), window_err)
	}
	return nil
}

// Splits a rate into the quota before its separator and the window behind it. A rate written
// as a bare window names no quota of its own, thus it takes the least quota there is: one new
// interesting input inside the window, which is what a bare window has always asked for.
func rate_split(text Text) (quota_text Quota_Text, window_text Text) {
	defer func() {
		Quota_Text_Invariants(quota_text, "rate_split.quota_text")
		Text_Invariants(window_text, "rate_split.window_text")
	}()
	Text_Invariants(text, "rate_split.text")
	for index := range len(text) {
		if text[index] != RATE_SEPARATOR {
			continue
		}
		return Quota_Text(text[:index]), text[index+1:]
	}
	return QUOTA_TEXT_ONE, text
}

// Duration_Check reports why a window names no span this program measures, and nil once it
// names one. The suffix is required: a bare number names no span, and reading one as seconds
// anyway would make the window a matter of this program's opinion rather than the caller's
// instruction.
func Duration_Check(text Text) (err error) {
	Text_Invariants(text, "Duration_Check.text")
	// The rate the caller typed is named by Rate_Check, which is the one that holds it. A
	// window is a part of that rate, and a message that quoted the part alone would answer
	// a caller of "90/30ms" with "30ms" — a text they never wrote.
	magnitude_text, unit_text := duration_split(text)
	if unit_text != SECOND_SUFFIX {
		return errors.New("a window is counted in seconds, as in 60s")
	}
	magnitude, magnitude_err := strconv.ParseInt(string(magnitude_text), 10, 64)
	if magnitude_err != nil {
		return fmt.Errorf("%q is not a whole number", string(magnitude_text))
	}
	if magnitude < MAGNITUDE_MIN {
		return errors.New("a window is positive")
	}
	if magnitude > MAGNITUDE_MAX {
		return errors.New("a window is at most an hour")
	}
	return nil
}

// Interesting_Parse reads the count of new interesting inputs out of one line of a
// `go test -fuzz` run. A line that carries no such field reports no count, which is how
// every other line the run writes leaves the record alone.
func Interesting_Parse(line Line) (interesting_count Interesting_Count, found Found) {
	defer func() {
		Interesting_Count_Invariants(interesting_count, "Interesting_Parse.count")
		Found_Invariants(found, "Interesting_Parse.found")
	}()
	Line_Invariants(line, "Interesting_Parse.line")
	field_offset := interesting_offset(line)
	if field_offset == FIELD_OFFSET_ABSENT {
		return 0, false
	}
	return interesting_digits(Digits(line[int(field_offset)+len(INTERESTING_FIELD):]))
}

// Progress_Observe reads line into the record. A line carrying a higher count moves the
// record and restarts the window. A count that did not rise is not new coverage, thus it
// leaves the record — and the window it started — exactly as they were.
func Progress_Observe(progress *Progress, line Line, now Reading) {
	Progress_Invariants(*progress, "Progress_Observe.progress")
	Line_Invariants(line, "Progress_Observe.line")
	Reading_Invariants(now, "Progress_Observe.now")
	interesting_count, found := Interesting_Parse(line)
	if !found {
		return
	}
	if !progress.Started {
		progress.Started = true
		progress.Interesting_Count = interesting_count
		progress.Anchor_Count = Anchor_Count(interesting_count)
		progress.Anchor_At = now
		return
	}
	if interesting_count <= progress.Interesting_Count {
		return
	}
	progress.Interesting_Count = interesting_count
	// The window starts again only where the quota is met, thus a run that finds inputs
	// steadily but below its rate still saturates. A quota of one makes every rise meet it,
	// which is what a bare window has always asked for.
	found_count := Quota(interesting_count) - Quota(progress.Anchor_Count)
	if found_count < progress.Rate.Quota {
		return
	}
	progress.Anchor_Count = Anchor_Count(interesting_count)
	progress.Anchor_At = now
}

// Progress_Saturated reports whether the count held still for the whole window. A run that
// reported no count at all is never saturated: it is still gathering baseline coverage, and
// a corpus large enough to take minutes over that must not read as a run that gave up.
func Progress_Saturated(progress Progress, now Reading) (saturated Saturated) {
	defer func() { Saturated_Invariants(saturated, "Progress_Saturated.saturated") }()
	Progress_Invariants(progress, "Progress_Saturated.progress")
	Reading_Invariants(now, "Progress_Saturated.now")
	if !progress.Started {
		return false
	}
	return Saturated(Window(now-progress.Anchor_At) >= progress.Rate.Window)
}

// Progress_Deadline returns how long a wait on the run's next output may last before the
// window elapses. It is what lets a run that stops writing altogether still be judged: the
// wait ends on the window even when nothing ends it first.
func Progress_Deadline(progress Progress, now Reading) (deadline Deadline) {
	defer func() { Deadline_Invariants(deadline, "Progress_Deadline.deadline") }()
	Progress_Invariants(progress, "Progress_Deadline.progress")
	Reading_Invariants(now, "Progress_Deadline.now")
	if !progress.Started {
		return Deadline(progress.Rate.Window)
	}
	elapsed := Window(now - progress.Anchor_At)
	// A wait is asked for only where the window has not elapsed: Main asks at the top of a
	// turn, and a turn it reached at all is one whose saturation check came back false. A
	// window that had elapsed would make this a wait of no span, which is not a wait.
	invariant.Always(elapsed < progress.Rate.Window, "A wait is asked for inside its window.")
	return Deadline(progress.Rate.Window - elapsed)
}

// Line_Reader_Next takes the next complete line out of chunk and carries an unfinished tail
// over into reader for the chunk behind it. The line points into reader and the call behind
// this one overwrites it, thus a caller reads the line before it calls again.
func Line_Reader_Next(
	reader *Line_Reader, chunk Chunk,
) (line Line, rest Rest, complete Complete) {
	defer func() {
		Line_Invariants(line, "Line_Reader_Next.line")
		Rest_Invariants(rest, "Line_Reader_Next.rest")
		Complete_Invariants(complete, "Line_Reader_Next.complete")
	}()
	Line_Reader_Invariants(*reader, "Line_Reader_Next.reader")
	Chunk_Invariants(chunk, "Line_Reader_Next.chunk")
	newline_offset := line_newline_offset(chunk)
	if newline_offset == BYTE_OFFSET_ABSENT {
		line_reader_fill(reader, chunk)
		return nil, nil, false
	}
	line_reader_fill(reader, chunk[:newline_offset])
	line = Line(reader.Buffer[:reader.Buffer_Size])
	reader.Buffer_Size = BUFFER_SIZE_MIN
	reader.Truncated = false
	return line, Rest(chunk[newline_offset+1:]), true
}

// Reads every complete line in chunk into progress. A chunk can hold several lines and can
// stop in the middle of one, thus the reader carries the tail over to the chunk behind it.
func progress_read(
	progress *Progress, reader *Line_Reader, chunk Chunk, now Reading,
) {
	Progress_Invariants(*progress, "progress_read.progress")
	Line_Reader_Invariants(*reader, "progress_read.reader")
	Chunk_Invariants(chunk, "progress_read.chunk")
	Reading_Invariants(now, "progress_read.now")
	rest := chunk
	// A chunk holds at most one line for each of its bytes, which bounds this loop by the
	// read that produced the chunk.
	for range len(chunk) {
		line, remainder, complete := Line_Reader_Next(reader, rest)
		if !complete {
			return
		}
		Progress_Observe(progress, line, now)
		rest = Chunk(remainder)
	}
}

// Returns where the interesting field starts in line, or FIELD_OFFSET_ABSENT when the line
// carries no such field. It names the field's own first byte rather than the byte behind it,
// so the offset a line carrying the field at its very front gives is zero.
func interesting_offset(line Line) (field_offset Field_Offset) {
	defer func() { Field_Offset_Invariants(field_offset, "interesting_offset.offset") }()
	Line_Invariants(line, "interesting_offset.line")
	field_size := len(INTERESTING_FIELD)
	line_size := len(line)
	if line_size < field_size {
		return FIELD_OFFSET_ABSENT
	}
	// The field can start anywhere that still leaves room for the whole of it. The match
	// runs here rather than in a function of its own, because a function would state a line
	// and an offset whose domains this loop has already narrowed past what either type says.
	start_size := line_size - field_size + 1
	for start_offset := range start_size {
		matched := true
		// The match walks its own offset forward rather than adding an index to the start,
		// because an offset and an index measure different things and adding them states a
		// quantity that is neither.
		match_offset := start_offset
		for field_index := range field_size {
			if line[match_offset] != INTERESTING_FIELD[field_index] {
				matched = false
				break
			}
			match_offset++
		}
		if matched {
			return Field_Offset(start_offset)
		}
	}
	return FIELD_OFFSET_ABSENT
}

// Reads the count the digits at the front of text spell. Text with no digit at all reports
// no count, which is what keeps a garbled field from reading as a number.
func interesting_digits(text Digits) (interesting_count Interesting_Count, found Found) {
	defer func() {
		Interesting_Count_Invariants(interesting_count, "interesting_digits.count")
		Found_Invariants(found, "interesting_digits.found")
	}()
	Digits_Invariants(text, "interesting_digits.text")
	digit_count := 0
	total := Interesting_Count(0)
	for index := range len(text) {
		character := text[index]
		if character < '0' {
			break
		}
		if character > '9' {
			break
		}
		// A longer run than an int64 counts wraps into some other number, and a wrapped
		// count is worse than no count: it would read as progress that never happened.
		if digit_count == DIGIT_COUNT_MAX {
			return 0, false
		}
		total = total*10 + Interesting_Count(character-'0')
		digit_count++
	}
	if digit_count == 0 {
		return 0, false
	}
	return total, true
}

// Appends as much of text as the reader's buffer still holds and marks the line truncated
// once the rest no longer fits. The drop is safe because the field this program reads sits
// near the start of a progress line, well inside the bound.
func line_reader_fill(reader *Line_Reader, text Chunk) {
	Line_Reader_Invariants(*reader, "line_reader_fill.reader")
	Chunk_Invariants(text, "line_reader_fill.text")
	free_size := LINE_SIZE_MAX - int(reader.Buffer_Size)
	text_size := len(text)
	if text_size > free_size {
		reader.Truncated = true
		text_size = free_size
	}
	copy(reader.Buffer[reader.Buffer_Size:], text[:text_size])
	reader.Buffer_Size += Buffer_Size(text_size)
}

// Returns the offset of the first newline in chunk, or BYTE_OFFSET_ABSENT when it holds
// none.
func line_newline_offset(chunk Chunk) (newline_offset Byte_Offset) {
	defer func() {
		Byte_Offset_Invariants(newline_offset, "line_newline_offset.offset")
	}()
	Chunk_Invariants(chunk, "line_newline_offset.chunk")
	for index := range len(chunk) {
		if chunk[index] == '\n' {
			return Byte_Offset(index)
		}
	}
	return BYTE_OFFSET_ABSENT
}

// Splits a window into its leading magnitude and its trailing unit, the unit being the run
// of letters the text ends in. Taking the whole run — rather than a fixed one or two
// characters — is what keeps 5ms from reading as five minutes with a stray s behind it.
func duration_split(text Text) (magnitude Text, unit Text) {
	defer func() {
		Text_Invariants(magnitude, "duration_split.magnitude")
		Text_Invariants(unit, "duration_split.unit")
	}()
	Text_Invariants(text, "duration_split.text")
	magnitude_size := len(text)
	for magnitude_size > 0 {
		character := text[magnitude_size-1]
		if character < 'a' {
			break
		}
		if character > 'z' {
			break
		}
		magnitude_size--
	}
	return text[:magnitude_size], text[magnitude_size:]
}
