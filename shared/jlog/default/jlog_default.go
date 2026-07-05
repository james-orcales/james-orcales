// Package jlog is the composition-tier sibling of the jlog library. It binds the
// library to the real world — os.Stderr, the operating-system clock, and
// runtime.Caller — in the one Default logger, exposes package-level convenience
// functions that log to it, and re-exports the library surface so callers can:
//
//	import jlog "local/james-orcales/shared/jlog/default"
//
//	jlog.Info("hello", jlog.String("user", name))
//
// and use the whole API from this one import as if no library/composition split
// had happened.
package jlog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"

	"local/james-orcales/shared/diode"
	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/jlog"
	"local/james-orcales/shared/time"
	system_time "local/james-orcales/shared/time/default"
)

// Logger re-exports jlog.Logger.
type Logger = jlog.Logger

// Logger_Configuration re-exports jlog.Logger_Configuration.
type Logger_Configuration = jlog.Logger_Configuration

// New_Input re-exports jlog.New_Input.
type New_Input = jlog.New_Input

// Field re-exports jlog.Field.
type Field = jlog.Field

// Field_Kind re-exports jlog.Field_Kind.
type Field_Kind = jlog.Field_Kind

// Key re-exports jlog.Key.
type Key = jlog.Key

// Level re-exports jlog.Level.
type Level = jlog.Level

// Buffer re-exports jlog.Buffer.
type Buffer = jlog.Buffer

// LEVEL_TRACE re-exports jlog.LEVEL_TRACE.
const LEVEL_TRACE = jlog.LEVEL_TRACE

// LEVEL_DEBUG re-exports jlog.LEVEL_DEBUG.
const LEVEL_DEBUG = jlog.LEVEL_DEBUG

// LEVEL_INFO re-exports jlog.LEVEL_INFO.
const LEVEL_INFO = jlog.LEVEL_INFO

// LEVEL_WARN re-exports jlog.LEVEL_WARN.
const LEVEL_WARN = jlog.LEVEL_WARN

// LEVEL_ERROR re-exports jlog.LEVEL_ERROR.
const LEVEL_ERROR = jlog.LEVEL_ERROR

// LEVEL_NONE re-exports jlog.LEVEL_NONE.
const LEVEL_NONE = jlog.LEVEL_NONE

// LEVEL_DISABLED re-exports jlog.LEVEL_DISABLED.
const LEVEL_DISABLED = jlog.LEVEL_DISABLED

// The number of internal frames between the injected caller lookup and a
// package-level convenience function's call site. Tuned for the global helpers
// (Info, Error, ...); explicit Logger_* calls sit one frame shallower, so
// their Caller location is off by one.
const caller_base_frames = 6

// The default diode ring capacity: how many finished lines can queue ahead of a slow
// stderr before the oldest are dropped. Line buffers are pooled, so memory tracks
// occupancy: an idle logger (or one whose sink keeps up) holds just the ~800 KB slot
// array, and it tops out near 55 MB (~512 B per line plus its bucket) only if the ring
// ever completely fills.
const default_diode_count = 100_000

// Default is the OS-bound logger the package-level convenience functions write to.
// It writes JSON lines to stderr through a non-blocking diode, stamps every line from
// the OS clock, and is reassignable: tests redirect it to a buffer, and programs may
// replace it at startup. Build a private logger with New when you need an independent
// one.
//
// Because the diode is non-blocking, a stderr that cannot keep up never stalls a
// caller; instead the oldest queued lines are dropped (reported via report_dropped),
// and lines still buffered at an unflushed exit are lost. A program that needs every
// line guaranteed should build its own synchronous logger with New.
var Default = New_Default_Logger()

// New_Default_Logger builds the OS-bound logger: a non-blocking diode over stderr, the
// operating system clock (shared by the logger and the diode's drain), automatic
// timestamps, and a runtime-backed caller lookup. This is the one place in the jlog
// tree where ambient binding is permitted.
func New_Default_Logger() (logger Logger) {
	clock, _ := system_time.New_Operating_System_Clock()
	writer := diode.New(diode.New_Input{
		Writer:        os.Stderr,
		Clock:         clock,
		Sleep:         system_time.Sleep,
		Count:         default_diode_count,
		Poll_Interval: 100 * time.MILLISECOND,
		Alerter:       report_dropped,
	})
	return jlog.New(jlog.New_Input{
		Writer:         writer,
		Clock:          clock,
		Floor:          jlog.LEVEL_TRACE,
		Auto_Timestamp: true,
		Caller:         operating_system_caller,
	})
}

// Surfaces diode drops on stderr so they never vanish silently, naming the cause so a slow
// sink and a rate limit are told apart.
func report_dropped(missed int, cause diode.Drop_Cause) {
	reason := "sink too slow"
	if cause == diode.DROP_RATE_LIMIT {
		reason = "rate limited"
	}
	fmt.Fprintf(os.Stderr, "jlog: dropped %d log lines (%s)\n", missed, reason)
}

// Resolves a frame-skip count to a "file:line" location via runtime.Caller, the
// composition tier's one sanctioned reach into the runtime.
func operating_system_caller(skip int) (location string) {
	_, file, line, ok := runtime.Caller(skip + caller_base_frames)
	if !ok {
		return ""
	}
	return file + ":" + strconv.Itoa(line)
}

// Trace logs a trace-level line to Default.
func Trace(message string, fields ...Field) {
	jlog.Logger_Trace(Default, message, fields...)
}

// Debug logs a debug-level line to Default.
func Debug(message string, fields ...Field) {
	jlog.Logger_Debug(Default, message, fields...)
}

// Info logs an info-level line to Default.
func Info(message string, fields ...Field) {
	jlog.Logger_Info(Default, message, fields...)
}

// Warn logs a warn-level line to Default.
func Warn(message string, fields ...Field) {
	jlog.Logger_Warn(Default, message, fields...)
}

// Error logs an error-level line to Default.
func Error(message string, fields ...Field) {
	jlog.Logger_Error(Default, message, fields...)
}

// Log logs a line with no severity field to Default.
func Log(message string, fields ...Field) {
	jlog.Logger_Log(Default, message, fields...)
}

// With returns a child of Default carrying fields as a fixed prefix.
func With(fields ...Field) (child Logger) {
	return jlog.Logger_With(Default, fields...)
}

// New re-exports jlog.New for building an independent logger from this import.
func New(input New_Input) (logger Logger) {
	return jlog.New(input)
}

// Logger_Trace re-exports jlog.Logger_Trace.
func Logger_Trace(logger Logger, message string, fields ...Field) {
	jlog.Logger_Trace(logger, message, fields...)
}

// Logger_Debug re-exports jlog.Logger_Debug.
func Logger_Debug(logger Logger, message string, fields ...Field) {
	jlog.Logger_Debug(logger, message, fields...)
}

// Logger_Info re-exports jlog.Logger_Info.
func Logger_Info(logger Logger, message string, fields ...Field) {
	jlog.Logger_Info(logger, message, fields...)
}

// Logger_Warn re-exports jlog.Logger_Warn.
func Logger_Warn(logger Logger, message string, fields ...Field) {
	jlog.Logger_Warn(logger, message, fields...)
}

// Logger_Error re-exports jlog.Logger_Error.
func Logger_Error(logger Logger, message string, fields ...Field) {
	jlog.Logger_Error(logger, message, fields...)
}

// Logger_Log re-exports jlog.Logger_Log.
func Logger_Log(logger Logger, message string, fields ...Field) {
	jlog.Logger_Log(logger, message, fields...)
}

// Logger_At_Level re-exports jlog.Logger_At_Level.
func Logger_At_Level(logger Logger, level Level, message string, fields ...Field) {
	jlog.Logger_At_Level(logger, level, message, fields...)
}

// Logger_With re-exports jlog.Logger_With.
func Logger_With(logger Logger, fields ...Field) (child Logger) {
	return jlog.Logger_With(logger, fields...)
}

// Logger_With_Context re-exports jlog.Logger_With_Context.
func Logger_With_Context(logger Logger, parent context.Context) (child context.Context) {
	return jlog.Logger_With_Context(logger, parent)
}

// From_Context re-exports jlog.From_Context.
func From_Context(ctx context.Context) (logger Logger) {
	return jlog.From_Context(ctx)
}

// String re-exports jlog.String.
func String[T ~string](key Key, value T) (field Field) {
	return jlog.String(key, value)
}

// Integer re-exports jlog.Integer.
func Integer[T ~int](key Key, value T) (field Field) {
	return jlog.Integer(key, value)
}

// Int8 re-exports jlog.Int8.
func Int8[T ~int8](key Key, value T) (field Field) {
	return jlog.Int8(key, value)
}

// Int16 re-exports jlog.Int16.
func Int16[T ~int16](key Key, value T) (field Field) {
	return jlog.Int16(key, value)
}

// Int32 re-exports jlog.Int32.
func Int32[T ~int32](key Key, value T) (field Field) {
	return jlog.Int32(key, value)
}

// Int64 re-exports jlog.Int64.
func Int64[T ~int64](key Key, value T) (field Field) {
	return jlog.Int64(key, value)
}

// Uint re-exports jlog.Uint.
func Uint[T ~uint](key Key, value T) (field Field) {
	return jlog.Uint(key, value)
}

// Uint8 re-exports jlog.Uint8.
func Uint8[T ~uint8](key Key, value T) (field Field) {
	return jlog.Uint8(key, value)
}

// Uint16 re-exports jlog.Uint16.
func Uint16[T ~uint16](key Key, value T) (field Field) {
	return jlog.Uint16(key, value)
}

// Uint32 re-exports jlog.Uint32.
func Uint32[T ~uint32](key Key, value T) (field Field) {
	return jlog.Uint32(key, value)
}

// Uint64 re-exports jlog.Uint64.
func Uint64[T ~uint64](key Key, value T) (field Field) {
	return jlog.Uint64(key, value)
}

// Uintptr re-exports jlog.Uintptr.
func Uintptr[T ~uintptr](key Key, value T) (field Field) {
	return jlog.Uintptr(key, value)
}

// Float32 re-exports jlog.Float32.
func Float32[T ~float32](key Key, value T) (field Field) {
	return jlog.Float32(key, value)
}

// Float64 re-exports jlog.Float64.
func Float64[T ~float64](key Key, value T) (field Field) {
	return jlog.Float64(key, value)
}

// Boolean re-exports jlog.Boolean.
func Boolean[T ~bool](key Key, value T) (field Field) {
	return jlog.Boolean(key, value)
}

// Bytes re-exports jlog.Bytes.
func Bytes(key Key, value []byte) (field Field) {
	return jlog.Bytes(key, value)
}

// Hexadecimal re-exports jlog.Hexadecimal.
func Hexadecimal(key Key, value []byte) (field Field) {
	return jlog.Hexadecimal(key, value)
}

// Raw_JSON re-exports jlog.Raw_JSON.
func Raw_JSON(key Key, value []byte) (field Field) {
	return jlog.Raw_JSON(key, value)
}

// Time re-exports jlog.Time.
func Time(key Key, value time.Moment) (field Field) {
	return jlog.Time(key, value)
}

// Duration re-exports jlog.Duration.
func Duration(key Key, value time.Duration) (field Field) {
	return jlog.Duration(key, value)
}

// IP_Address re-exports jlog.IP_Address.
func IP_Address(key Key, value net.IP) (field Field) {
	return jlog.IP_Address(key, value)
}

// MAC_Address re-exports jlog.MAC_Address.
func MAC_Address(key Key, value net.HardwareAddr) (field Field) {
	return jlog.MAC_Address(key, value)
}

// Any re-exports jlog.Any.
func Any(key Key, value any) (field Field) {
	return jlog.Any(key, value)
}

// Strings re-exports jlog.Strings.
func Strings(key Key, value []string) (field Field) {
	return jlog.Strings(key, value)
}

// Integers re-exports jlog.Integers.
func Integers(key Key, value []int) (field Field) {
	return jlog.Integers(key, value)
}

// Floats64 re-exports jlog.Floats64.
func Floats64(key Key, value []float64) (field Field) {
	return jlog.Floats64(key, value)
}

// Booleans re-exports jlog.Booleans.
func Booleans(key Key, value []bool) (field Field) {
	return jlog.Booleans(key, value)
}

// Durations re-exports jlog.Durations.
func Durations(key Key, value []time.Duration) (field Field) {
	return jlog.Durations(key, value)
}

// Err re-exports jlog.Err.
func Err(value error) (field Field) {
	return jlog.Err(value)
}

// Timestamp re-exports jlog.Timestamp.
func Timestamp() (field Field) {
	return jlog.Timestamp()
}

// Caller re-exports jlog.Caller.
func Caller() (field Field) {
	return jlog.Caller()
}

// The rest of this file is the pretty printer the library core deliberately omits. jlog emits
// flat JSON for machines; a Console reads those lines back and renders them for a human at a
// terminal — the human-facing inverse of the encoder. It plugs into the encoder only through the
// io.Writer seam (each emit writes one whole flat-JSON object per Write), so the zero-allocation
// hot path is never touched. Being off the hot path, it may allocate freely.

// An ANSI SGR sequence. A distinct type so buffer_paint takes it without colliding with its
// plain-text argument under the same-type-parameter rule, mirroring maddox's ansi_code.
type ansi_code string

const ansi_reset ansi_code = "\x1b[0m"
const ansi_faint ansi_code = "\x1b[2m"
const ansi_bold ansi_code = "\x1b[1m"
const ansi_red ansi_code = "\x1b[31m"
const ansi_green ansi_code = "\x1b[32m"
const ansi_yellow ansi_code = "\x1b[33m"
const ansi_cyan ansi_code = "\x1b[36m"

// A field-name default is a distinct type so string_or's two parameters never repeat a type,
// which the input-struct rule would otherwise force into a struct — the trick jlog.go uses.
type default_field_name string

// These mirror jlog.go's default field names, which are unexported there and so cannot be
// referenced; a Console built with no overrides must match a logger built with no overrides.
const default_level_field_name default_field_name = "level"
const default_message_field_name default_field_name = "message"
const default_timestamp_field_name default_field_name = "time"
const default_error_field_name default_field_name = "error"

// New_Console_Input configures New_Console. Empty field names take jlog's defaults, so a Console
// matches a default logger. Only the keys a Console treats specially are configurable: the
// timestamp, level, and message keys form the header, and the error key's value is painted red;
// every other key (caller, stack, and the caller's own fields) renders the same generic way, so
// renaming one needs no knob here.
type New_Console_Input struct {
	// Writer receives the rendered text; nil becomes io.Discard.
	Writer io.Writer
	// Color enables ANSI color; the caller decides it (e.g. from a TTY check), not the Console.
	Color bool
	// Level_Field_Name overrides the level key; empty uses the default.
	Level_Field_Name string
	// Message_Field_Name overrides the message key; empty uses the default.
	Message_Field_Name string
	// Timestamp_Field_Name overrides the timestamp key; empty uses the default.
	Timestamp_Field_Name string
	// Error_Field_Name overrides the error key; empty uses the default.
	Error_Field_Name string
}

// Console is an io.Writer that renders each flat-JSON jlog line as a human-readable console line.
// It is passed by value; its fields are set only by New_Console and never mutated, so a copy is a
// faithful, independent Console.
type Console struct {
	// Writer receives the rendered text.
	Writer io.Writer
	// Color enables ANSI color in the rendered output.
	Color bool
	// Level_Field is the key whose value becomes the level tag.
	Level_Field string
	// Message_Field is the key whose value becomes the bold message.
	Message_Field string
	// Time_Field is the key whose value opens the header verbatim.
	Time_Field string
	// Error_Field is the key whose value is painted red.
	Error_Field string
}

// New_Console builds a Console from input, resolving empty field names to jlog's defaults.
func New_Console(input New_Console_Input) (console Console) {
	writer := input.Writer
	if writer == nil {
		writer = io.Discard
	}
	console.Writer = writer
	console.Color = input.Color
	console.Level_Field = string_or(input.Level_Field_Name, default_level_field_name)
	console.Message_Field = string_or(input.Message_Field_Name, default_message_field_name)
	console.Time_Field = string_or(input.Timestamp_Field_Name, default_timestamp_field_name)
	console.Error_Field = string_or(input.Error_Field_Name, default_error_field_name)
	invariant.Always(console.Level_Field != "", "jlog: level field name is non-empty")
	invariant.Always(console.Message_Field != "", "jlog: message field name is non-empty")
	invariant.Always(console.Time_Field != "", "jlog: time field name is non-empty")
	invariant.Always(console.Error_Field != "", "jlog: error field name is non-empty")
	return console
}

func string_or(value string, fallback default_field_name) (chosen string) {
	if value == "" {
		return string(fallback)
	}
	return value
}

// Write renders the newline-terminated JSON lines in payload and writes the human-readable form
// to the destination in one Write. It satisfies io.Writer — the one method the house style
// permits — so a Console is a drop-in jlog writer. It reports len(payload) consumed on success (a
// transforming writer emits a different byte count than it takes), so jlog's own length check is
// satisfied.
func (console Console) Write(payload []byte) (written int, err error) {
	rendered := make(Buffer, 0, len(payload)*2)
	line_start := 0
	for index := 0; index < len(payload); index++ {
		if payload[index] != '\n' {
			continue
		}
		rendered = buffer_append_pretty_line(rendered, payload[line_start:index], console)
		rendered = append(rendered, '\n')
		line_start = index + 1
	}
	// A trailing chunk with no newline is not something jlog produces (every line ends in \n),
	// but a Console used on a raw pipe might see one; render it without inventing a newline.
	if line_start < len(payload) {
		rendered = buffer_append_pretty_line(rendered, payload[line_start:], console)
	}
	_, write_err := console.Writer.Write(rendered)
	if write_err != nil {
		return 0, write_err
	}
	return len(payload), nil
}

// New_Terminal_Logger returns a jlog.Logger whose lines are rendered by a Console to os.Stdout,
// with color enabled only when os.Stdout is a terminal. It is synchronous (no diode): a terminal
// dev logger values losing nothing at exit over never blocking, and a human watching a terminal
// is not a hot path. Binding os.Stdout here is composition-tier wiring, so it is allowed.
func New_Terminal_Logger() (logger Logger) {
	clock, _ := system_time.New_Operating_System_Clock()
	console := New_Console(New_Console_Input{
		Writer: os.Stdout,
		Color:  file_is_terminal(os.Stdout),
	})
	return jlog.New(jlog.New_Input{
		Writer:         console,
		Clock:          clock,
		Floor:          jlog.LEVEL_TRACE,
		Auto_Timestamp: true,
		Caller:         operating_system_caller,
	})
}

// Reports whether file is a terminal by its character-device bit — the same stdlib-only check
// maddox/main.go uses, so no golang.org/x/term dependency is added against the zero-deps rule.
func file_is_terminal(file *os.File) (terminal bool) {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// How a JSON value is displayed: a string is unquoted (and logfmt-requoted only if needed), a
// literal (number/bool/null) is copied verbatim so a large integer keeps its exact digits, and a
// compound (array/object) is copied as compacted JSON so it stays one flat token.
type value_kind uint8

const value_is_string value_kind = 0
const value_is_literal value_kind = 1
const value_is_compound value_kind = 2

// One key/value member of a rendered line, holding the value already reduced to its display text
// and kind so rendering never re-parses.
type member struct {
	// Key is the JSON object key.
	Key string
	// Text is the value's display form, already unescaped or compacted.
	Text string
	// Kind selects how Text is quoted when rendered.
	Kind value_kind
}

// Appends line rendered as one console line (no trailing newline; Write adds that). A line that is
// not a JSON object is appended verbatim, so interleaved non-JSON output is never lost.
func buffer_append_pretty_line(
	destination Buffer, line []byte, console Console,
) (output Buffer) {
	members, parsed := scan_object(line)
	if !parsed {
		return append(destination, line...)
	}
	return buffer_append_members(destination, members, console)
}

// Scans one flat-JSON object into its members in wire order. It is hand-rolled rather than built
// on json.NewDecoder because the house style bans that unbounded streaming API, and json.Unmarshal
// into a map would scramble jlog's deliberate field order and round a large uint64 through float64.
// Each value's exact source bytes are captured (numbers stay exact, an embedded Raw_JSON blob
// survives). Any deviation from a lone, well-formed object returns parsed=false, and the caller
// passes the line through untouched.
func scan_object(line []byte) (members []member, parsed bool) {
	index := scan_space(line, 0)
	if index >= len(line) {
		return nil, false
	}
	if line[index] != '{' {
		return nil, false
	}
	index = scan_space(line, index+1)
	if index < len(line) {
		if line[index] == '}' {
			return nil, scan_space(line, index+1) == len(line)
		}
	}
	// Each member is at least one byte, so len(line) bounds the count and keeps the loop
	// bounded; a well-formed object returns through '}' before that ceiling is reached.
	for guard_index := 0; guard_index < len(line); guard_index++ {
		key, key_next, key_ok := scan_string(line, index)
		if !key_ok {
			return nil, false
		}
		index = scan_space(line, key_next)
		if index >= len(line) {
			return nil, false
		}
		if line[index] != ':' {
			return nil, false
		}
		index = scan_space(line, index+1)
		text, kind, value_next, value_ok := scan_value(line, index)
		if !value_ok {
			return nil, false
		}
		members = append(members, member{Key: key, Text: text, Kind: kind})
		index = scan_space(line, value_next)
		if index >= len(line) {
			return nil, false
		}
		if line[index] == ',' {
			index = scan_space(line, index+1)
			continue
		}
		if line[index] == '}' {
			return members, scan_space(line, index+1) == len(line)
		}
		return nil, false
	}
	return nil, false
}

// Skips JSON insignificant whitespace from start and returns the next index. jlog output carries
// none, but a Console fed a hand-formatted line still parses.
func scan_space(line []byte, start int) (next int) {
	index := start
	for index < len(line) {
		if !byte_is_space(line[index]) {
			return index
		}
		index++
	}
	return index
}

func byte_is_space(value byte) (space bool) {
	switch value {
	case ' ', '\t', '\n', '\r':
		return true
	}
	return false
}

// Reads a JSON string beginning at start, returning its unescaped value and the index past the
// closing quote. The raw span (quotes included) is handed to json.Unmarshal — a bounded []byte, the
// API the house style prefers — so escapes decode exactly; the scan only has to find the real close
// quote, which means skipping a backslash-escaped byte so an escaped quote does not end it early.
func scan_string(line []byte, start int) (value string, next int, ok bool) {
	if start >= len(line) {
		return "", start, false
	}
	if line[start] != '"' {
		return "", start, false
	}
	index := start + 1
	for index < len(line) {
		if line[index] == '\\' {
			index += 2
			continue
		}
		if line[index] == '"' {
			var decoded string
			if err := json.Unmarshal(line[start:index+1], &decoded); err != nil {
				return "", start, false
			}
			return decoded, index + 1, true
		}
		index++
	}
	return "", start, false
}

// Reads one JSON value beginning at start, classifying it: a string is unescaped, an array or
// object is captured as its verbatim (already-compact) source bytes, and anything else is a scalar
// literal whose digits are kept exactly.
func scan_value(line []byte, start int) (text string, kind value_kind, next int, ok bool) {
	if start >= len(line) {
		return "", value_is_literal, start, false
	}
	if line[start] == '"' {
		value, value_next, value_ok := scan_string(line, start)
		if !value_ok {
			return "", value_is_literal, start, false
		}
		return value, value_is_string, value_next, true
	}
	if line[start] == '{' {
		return scan_compound(line, start)
	}
	if line[start] == '[' {
		return scan_compound(line, start)
	}
	return scan_literal(line, start)
}

// Reads a brace- or bracket-delimited value beginning at start, returning its verbatim source
// bytes. A depth counter over the matching delimiter finds the close; strings are skipped whole so
// a delimiter inside a string does not miscount. Cross-type nesting (an array inside an object) is
// transparent — the other delimiter is an ordinary byte — and any real imbalance surfaces as a
// parse failure at the enclosing object, which passes the line through.
func scan_compound(line []byte, start int) (text string, kind value_kind, next int, ok bool) {
	opener := line[start]
	closer := byte('}')
	if opener == '[' {
		closer = ']'
	}
	depth := 0
	index := start
	for index < len(line) {
		if line[index] == '"' {
			_, string_next, string_ok := scan_string(line, index)
			if !string_ok {
				return "", value_is_compound, start, false
			}
			index = string_next
			continue
		}
		if line[index] == opener {
			depth++
			index++
			continue
		}
		if line[index] == closer {
			depth--
			index++
			if depth == 0 {
				return string(line[start:index]), value_is_compound, index, true
			}
			continue
		}
		index++
	}
	return "", value_is_compound, start, false
}

// Reads a scalar literal (number, true, false, or null) beginning at start, ending it at the first
// value terminator so its exact source digits are captured.
func scan_literal(line []byte, start int) (text string, kind value_kind, next int, ok bool) {
	index := start
	for index < len(line) {
		if byte_ends_literal(line[index]) {
			break
		}
		index++
	}
	if index == start {
		return "", value_is_literal, start, false
	}
	return string(line[start:index]), value_is_literal, index, true
}

// Reports whether value terminates a scalar literal: a member separator, a container close, or
// whitespace.
func byte_ends_literal(value byte) (ends bool) {
	switch value {
	case ',', '}', ']', ' ', '\t', '\n', '\r':
		return true
	}
	return false
}

// Renders the header (timestamp, level, message — in that order, whatever their wire order) then
// every remaining member as a logfmt pair, each part separated from the last by a single space.
func buffer_append_members(
	destination Buffer, members []member, console Console,
) (output Buffer) {
	// An absent header key and an empty one render the same — nothing — so a plain non-empty
	// check covers both, and jlog never emits an empty timestamp, level, or message anyway. The
	// timestamp is coarsened to the second here; the tail keeps any Time field's precision.
	time_text := timestamp_seconds(member_text(members, console.Time_Field))
	level_text := member_text(members, console.Level_Field)
	message_text := member_text(members, console.Message_Field)
	wrote := false
	if time_text != "" {
		destination = buffer_append_separated(destination, wrote)
		destination = buffer_paint(destination, ansi_faint, console.Color, time_text)
		wrote = true
	}
	if level_text != "" {
		destination = buffer_append_separated(destination, wrote)
		destination = buffer_append_level(destination, level_text, console.Color)
		wrote = true
	}
	if message_text != "" {
		destination = buffer_append_separated(destination, wrote)
		destination = buffer_paint(destination, ansi_bold, console.Color, message_text)
		wrote = true
	}
	for index := 0; index < len(members); index++ {
		current := members[index]
		if member_is_header(current.Key, console) {
			continue
		}
		destination = buffer_append_separated(destination, wrote)
		destination = buffer_append_field(destination, current, console)
		wrote = true
	}
	return destination
}

// Reports whether key is one of the three header keys, which render in the header rather than
// repeat in the logfmt tail.
func member_is_header(key string, console Console) (header bool) {
	if key == console.Time_Field {
		return true
	}
	if key == console.Level_Field {
		return true
	}
	if key == console.Message_Field {
		return true
	}
	return false
}

// Returns the display text of the first member with key, or "" when no member has it.
func member_text(members []member, key string) (text string) {
	for index := 0; index < len(members); index++ {
		if members[index].Key == key {
			return members[index].Text
		}
	}
	return ""
}

// Appends a single space before a part when a previous part was already written, so the line
// never opens with or doubles a separator.
func buffer_append_separated(destination Buffer, wrote bool) (output Buffer) {
	if wrote {
		return append(destination, ' ')
	}
	return destination
}

// Appends one logfmt "key=value" pair: the key in cyan so it reads as a label yet stays legible
// (unlike a dim gray), the value logfmt-quoted, and — for the error field — the value painted red
// so a failure stands out.
func buffer_append_field(destination Buffer, field member, console Console) (output Buffer) {
	destination = buffer_paint(destination, ansi_cyan, console.Color, field.Key)
	destination = append(destination, '=')
	value_color := ansi_code("")
	if field.Key == console.Error_Field {
		value_color = ansi_red
	}
	return buffer_paint(destination, value_color, console.Color, logfmt_token(field))
}

// Appends the level as its three-letter tag, painted by severity.
func buffer_append_level(destination Buffer, wire string, color bool) (output Buffer) {
	return buffer_paint(destination, level_color(wire), color, level_label(wire))
}

// Appends text wrapped in code and a reset when color is on and code is a real color; otherwise
// appends text bare. An empty code means "no color for this part", so an unknown level or a
// non-error value stays uncolored even under color.
func buffer_paint(destination Buffer, code ansi_code, color bool, text string) (output Buffer) {
	if !color {
		return append(destination, text...)
	}
	if code == "" {
		return append(destination, text...)
	}
	destination = append(destination, code...)
	destination = append(destination, text...)
	return append(destination, ansi_reset...)
}

// Returns the display token for a field value: a string is bare unless logfmt requires quoting; a
// literal or compound value is already a safe bare token.
func logfmt_token(field member) (token string) {
	if field.Kind != value_is_string {
		return field.Text
	}
	if logfmt_needs_quote(field.Text) {
		return strconv.Quote(field.Text)
	}
	return field.Text
}

// Reports whether a logfmt value must be double-quoted: an empty value, or one carrying a space,
// an equals, a quote, or a control byte, is ambiguous or unreadable bare.
func logfmt_needs_quote(value string) (needs bool) {
	if value == "" {
		return true
	}
	for index := 0; index < len(value); index++ {
		current := value[index]
		switch current {
		case ' ', '=', '"':
			return true
		}
		if current < 0x20 {
			return true
		}
	}
	return false
}

// Maps a level's wire name to its three-letter tag. An unknown level (a custom numeric level, or
// "disabled") falls back to its first three characters uppercased, so nothing renders blank.
func level_label(wire string) (label string) {
	switch wire {
	case "trace":
		return "TRC"
	case "debug":
		return "DBG"
	case "info":
		return "INF"
	case "warn":
		return "WRN"
	case "error":
		return "ERR"
	}
	upper := strings.ToUpper(wire)
	if len(upper) > 3 {
		return upper[:3]
	}
	return upper
}

// Maps a level's wire name to its color; an unknown level returns the empty code, meaning no
// color, so buffer_paint leaves it plain.
func level_color(wire string) (code ansi_code) {
	switch wire {
	case "trace":
		return ansi_faint
	case "debug":
		return ansi_cyan
	case "info":
		return ansi_green
	case "warn":
		return ansi_yellow
	case "error":
		return ansi_red
	}
	return ""
}

// Trims an RFC 3339 header timestamp to second precision by dropping its sub-second fraction and
// keeping the trailing zone, so "…:20.123456789Z" becomes "…:20Z" — a human reading the console
// needs no nanosecond granularity. A value with no fraction is returned unchanged; only the
// console display coarsens, never the JSON line.
func timestamp_seconds(value string) (seconds string) {
	fraction_offset := strings.IndexByte(value, '.')
	if fraction_offset < 0 {
		return value
	}
	// The zone is searched absolutely, not relative to the fraction, so the two offsets are
	// never added: an RFC 3339 UTC value carries exactly one 'Z', the terminal zone marker.
	zone_offset := strings.IndexByte(value, 'Z')
	if zone_offset < 0 {
		return value[:fraction_offset]
	}
	return value[:fraction_offset] + value[zone_offset:]
}
