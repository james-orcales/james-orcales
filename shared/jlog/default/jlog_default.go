// Package jlog is the composition-tier sibling of the jlog library. It binds the
// library to the real world — os.Stderr, the operating-system clock, and
// runtime.Caller — in the one Default logger, exposes package-level convenience
// functions that log to it, and re-exports the library surface so callers can:
//
//	import jlog "github.com/james-orcales/james-orcales/shared/jlog/default"
//
//	jlog.Info("hello", jlog.String("user", name))
//
// and use the whole API from this one import as if no library/composition split
// had happened.
package jlog

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"

	"github.com/james-orcales/james-orcales/shared/diode"
	"github.com/james-orcales/james-orcales/shared/jlog"
	jtime "github.com/james-orcales/james-orcales/shared/time"
	system_time "github.com/james-orcales/james-orcales/shared/time/default"
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

// Level_Trace re-exports jlog.Level_Trace.
const Level_Trace = jlog.Level_Trace

// Level_Debug re-exports jlog.Level_Debug.
const Level_Debug = jlog.Level_Debug

// Level_Info re-exports jlog.Level_Info.
const Level_Info = jlog.Level_Info

// Level_Warn re-exports jlog.Level_Warn.
const Level_Warn = jlog.Level_Warn

// Level_Error re-exports jlog.Level_Error.
const Level_Error = jlog.Level_Error

// Level_None re-exports jlog.Level_None.
const Level_None = jlog.Level_None

// Level_Disabled re-exports jlog.Level_Disabled.
const Level_Disabled = jlog.Level_Disabled

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
	clock := system_time.New_Operating_System_Clock()
	writer := diode.New(diode.New_Input{
		Writer:        os.Stderr,
		Clock:         clock,
		Count:         default_diode_count,
		Poll_Interval: 100 * jtime.Millisecond,
		Alerter:       report_dropped,
	})
	return jlog.New(jlog.New_Input{
		Writer:         writer,
		Clock:          clock,
		Floor:          jlog.Level_Trace,
		Auto_Timestamp: true,
		Caller:         operating_system_caller,
	})
}

// Surfaces diode drops on stderr so they never vanish silently, naming the cause so a slow
// sink and a rate limit are told apart.
func report_dropped(missed int, cause diode.Drop_Cause) {
	reason := "sink too slow"
	if cause == diode.Drop_Rate_Limit {
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
func String(key Key, value string) (field Field) {
	return jlog.String(key, value)
}

// Integer re-exports jlog.Integer.
func Integer(key Key, value int) (field Field) {
	return jlog.Integer(key, value)
}

// Int64 re-exports jlog.Int64.
func Int64(key Key, value int64) (field Field) {
	return jlog.Int64(key, value)
}

// Uint re-exports jlog.Uint.
func Uint(key Key, value uint) (field Field) {
	return jlog.Uint(key, value)
}

// Uint64 re-exports jlog.Uint64.
func Uint64(key Key, value uint64) (field Field) {
	return jlog.Uint64(key, value)
}

// Float32 re-exports jlog.Float32.
func Float32(key Key, value float32) (field Field) {
	return jlog.Float32(key, value)
}

// Float64 re-exports jlog.Float64.
func Float64(key Key, value float64) (field Field) {
	return jlog.Float64(key, value)
}

// Boolean re-exports jlog.Boolean.
func Boolean(key Key, value bool) (field Field) {
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
func Time(key Key, value jtime.Moment) (field Field) {
	return jlog.Time(key, value)
}

// Duration re-exports jlog.Duration.
func Duration(key Key, value jtime.Duration) (field Field) {
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
func Durations(key Key, value []jtime.Duration) (field Field) {
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
