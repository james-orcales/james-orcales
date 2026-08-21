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
	"net"
	"runtime"
	"syscall"
	"unsafe"

	"local/james-orcales/shared/jlog"
	"local/james-orcales/shared/simulation/time"
	system_time "local/james-orcales/shared/simulation/time/default"
	"local/james-orcales/shared/strconv"
	"local/james-orcales/shared/sync/diode"
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

// Data re-exports jlog.Data.
type Data = jlog.Data

// Data_Size re-exports jlog.Data_Size.
type Data_Size = jlog.Data_Size

// Write re-exports jlog.Write.
type Write = jlog.Write

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
const CALLER_BASE_FRAMES = 6

// The default diode ring capacity: how many finished lines can queue ahead of a slow
// stderr before the oldest are dropped. Line buffers are pooled, so memory tracks
// occupancy: an idle logger (or one whose sink keeps up) holds just the ~800 KB slot
// array, and it tops out near 55 MB (~512 B per line plus its bucket) only if the ring
// ever completely fills.
const DEFAULT_DIODE_COUNT = diode.SLOT_COUNT_MAXIMUM

// STDERR_DESCRIPTOR is the process standard error stream owned by this composition root.
const STDERR_DESCRIPTOR = 2

// DROP_REPORT_SIZE_MAXIMUM holds static text plus the widest bounded missed count.
const DROP_REPORT_SIZE_MAXIMUM = len("jlog: dropped  log lines (sink too slow)\n") +
	strconv.DECIMAL_TEXT_SIZE_MAXIMUM

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
	host := system_time.New_Operating_System_Clock()
	writer := diode.New(diode.New_Input{
		Write:         diode_stderr_write,
		Clock:         host,
		Sleep:         system_time.Sleep,
		Count:         DEFAULT_DIODE_COUNT,
		Poll_Interval: diode.Stored_Poll_Interval(100 * time.MILLISECOND),
		Alerter:       report_dropped,
	})
	return jlog.New(jlog.New_Input{
		Writer_State:   unsafe.Pointer(writer),
		Write:          jlog_diode_write,
		Clock:          host,
		Floor:          jlog.LEVEL_INFO,
		Auto_Timestamp: true,
		Caller:         operating_system_caller,
	})
}

// Surfaces diode drops on stderr so they never vanish silently, naming the cause so a slow
// sink and a rate limit are told apart.
func report_dropped(missed int, cause diode.Drop_Cause) {
	reason := []byte("sink too slow")
	if cause == diode.DROP_RATE_LIMIT {
		reason = []byte("rate limited")
	}
	var storage [DROP_REPORT_SIZE_MAXIMUM]byte
	report := storage[:0]
	report = append(report, "jlog: dropped "...)
	var digits [strconv.DECIMAL_TEXT_SIZE_MAXIMUM]byte
	count := strconv.Format_Decimal_Into(digits[:], strconv.Machine_Integer(missed))
	report = append(report, digits[:int(count)]...)
	report = append(report, " log lines ("...)
	report = append(report, reason...)
	report = append(report, ')', '\n')
	syscall.Write(STDERR_DESCRIPTOR, report)
}

func diode_stderr_write(
	_ unsafe.Pointer, data diode.Data,
) (written diode.Data_Size, err error) {
	count, write_error := syscall.Write(STDERR_DESCRIPTOR, data)
	return diode.Data_Size(count), write_error
}

func jlog_diode_write(
	state unsafe.Pointer, data jlog.Data,
) (written jlog.Data_Size, err error) {
	count, write_error := (*diode.Writer)(state).Write(data)
	return jlog.Data_Size(count), write_error
}

// Resolves a frame-skip count to a "file:line" location via runtime.Caller, the
// composition tier's one sanctioned reach into the runtime.
func operating_system_caller(skip int) (location string) {
	_, file, line, ok := runtime.Caller(skip + CALLER_BASE_FRAMES)
	if !ok {
		return ""
	}
	var storage [strconv.DECIMAL_TEXT_SIZE_MAXIMUM]byte
	count := strconv.Format_Decimal_Into(storage[:], strconv.Machine_Integer(line))
	return file + ":" + string(storage[:int(count)])
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

// Level_Filter re-exports jlog.Level_Filter.
type Level_Filter = jlog.Level_Filter

// New_Level_Filter_Input re-exports jlog.New_Level_Filter_Input.
type New_Level_Filter_Input = jlog.New_Level_Filter_Input

// New_Level_Filter re-exports jlog.New_Level_Filter.
func New_Level_Filter(input New_Level_Filter_Input) (filter Level_Filter) {
	return jlog.New_Level_Filter(input)
}
