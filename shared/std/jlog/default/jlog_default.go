// Package jlog is the composition-tier sibling of the jlog library. It binds the
// library to the real world — stderr through the root's nbio.Storage, the
// operating-system clock, and runtime.Caller — in the one Default logger, exposes
// package-level convenience functions that log to it, and forwards library operations:
//
//	import jlog "local/james-orcales/shared/jlog/default"
//
//	jlog.Info("hello", jlog.String("user", name))
//
// Explicit logger, configuration, and field types belong to shared/jlog.
package jlog

import "runtime"
import "unsafe"

import "local/james-orcales/shared/jlog"
import "local/james-orcales/shared/sim/aver/default"
import "local/james-orcales/shared/sim/nbio"
import "local/james-orcales/shared/sim/time"
import system_time "local/james-orcales/shared/sim/time/default"

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

// WRITE_TIMEOUT bounds one stderr write on the timeline; a stuck pipe retires with
// Deadline_Exceeded instead of holding the line forever.
const WRITE_TIMEOUT = 1 * time.SECOND

// FILE_MODES lists what a borrowed descriptor supports: it has no cursor, and Close must not
// release a descriptor this package never opened.
const FILE_MODES nbio.Stream_Mode_Set = 1<<nbio.STREAM_MODE_CLOSE |
  1<<nbio.STREAM_MODE_FLUSH |
  1<<nbio.STREAM_MODE_WRITE |
  1<<nbio.STREAM_MODE_DESTROY |
  1<<nbio.STREAM_MODE_QUERY

// Default is the logger the package-level convenience functions write through. It is the zero
// Logger, which emits nothing, until the root that owns the IO assigns it:
//
//	jlog.Default = jlog.New_Default_Logger(loop.Storage, stderr)
//
//	var line [jlog.LINE_BYTES]byte
//	state := jlog.Default_Logger{Line: line[:]}
//	jlog.Default = jlog.New_Default_Logger(&state, loop.Storage, stderr)
//
// Like every jlog.Logger it has one owner: a program whose goroutines all log builds one Logger
// per goroutine with New or Logger_With instead of sharing Default.
var Default jlog.Logger

// LINE_BYTES holds a typical stderr line with room for a few long strings.
const LINE_BYTES = 4096

// Default_Logger is the caller-owned state behind the OS-bound logger: the root keeps one so
// New_Default_Logger allocates nothing and Default stays the package's only variable.
type Default_Logger struct {
  // Configuration is the logger's dependency set, bound to the OS by New_Default_Logger.
  Configuration jlog.Logger_Configuration
  // File is the Stream over the borrowed descriptor.
  File Stream_File
  // Line is the caller-owned line storage, LINE_BYTES for a typical stderr line.
  Line []byte
}

// New_Default_Logger fills state in place and builds the OS-bound logger over the root's
// Storage and the descriptor it names, typically stderr: lines go out through
// nbio.Storage_Write, stamped by the OS clock, with a runtime-backed caller lookup. This is the
// one place in the jlog tree where ambient binding is permitted. The caller filled Line first.
func New_Default_Logger(
  state *Default_Logger,
  storage nbio.Storage,
  file nbio.File,
) (logger jlog.Logger) {
  aver.Always(state != nil, "jlog: default logger storage is required")
  aver.Always(len(state.Line) > 0, "jlog: default logger line storage is required")
  state.File = Stream_File{Storage: storage, File: file}
  state.Configuration = jlog.Logger_Configuration{
    Stream: File_To_Stream(&state.File),
    Clock:  system_time.New_Operating_System_Clock(),
    Caller: operating_system_caller,
  }
  logger = jlog.New(&state.Configuration, state.Line, jlog.LEVEL_INFO)
  logger.Auto_Timestamp = true
  return logger
}

// Stream_File is an nbio.Stream over one descriptor the process already holds, written through
// the injected Storage so the root's loop retires it. Completion sits first so the static
// retire callback recovers this state from the handle alone.
type Stream_File struct {
  // Completion is this Stream's own submission storage; the caller's is filled on retire.
  Completion nbio.Completion
  // Storage is the injected file half of the IO surface.
  Storage nbio.Storage
  // File is the borrowed descriptor.
  File nbio.File
  // Caller_Completion is the outer completion awaiting the in-flight write.
  Caller_Completion *nbio.Completion
  // Callback is the outer retirement for the in-flight write.
  Callback nbio.Stream_Callback
  // Closed refuses writes after Close without releasing the descriptor.
  Closed bool
}

// File_To_Stream keeps descriptor ownership in the caller state.
func File_To_Stream(state *Stream_File) (stream nbio.Stream) {
  aver.Always(state != nil, "A file Stream has state.")
  return nbio.Stream{State: state, Procedure: stream_file_procedure}
}

// One write is in flight at a time, which holds because a Logger has one owner and a diode
// submits one line at a time; a second submission while one is pending is a caller defect.
func stream_file_procedure(
  state_value nbio.State,
  completion *nbio.Completion,
  mode nbio.Stream_Mode,
  data []byte,
  _ int64,
  _ nbio.Seek_From,
  callback nbio.Stream_Callback,
) {
  state := state_value.(*Stream_File)
  completion.Data = 0
  completion.Error = nil
  switch mode {
  case nbio.STREAM_MODE_WRITE:
    if state.Closed {
      completion.Error = nbio.Stream_Empty
      break
    }
    aver.Always(
      state.Caller_Completion == nil, "A file Stream has one write in flight.",
    )
    state.Caller_Completion = completion
    state.Callback = callback
    nbio.Storage_Write(
      state.Storage,
      &state.Completion,
      state.File,
      data,
      nbio.FILE_POSITION_CURRENT,
      WRITE_TIMEOUT,
      stream_file_retire,
    )
    return
  case nbio.STREAM_MODE_QUERY:
    completion.Data = int(FILE_MODES)
  case nbio.STREAM_MODE_CLOSE, nbio.STREAM_MODE_DESTROY:
    state.Closed = true
  case nbio.STREAM_MODE_FLUSH:
    if state.Closed {
      completion.Error = nbio.Stream_Empty
    }
  default:
    completion.Error = nbio.Stream_Empty
  }
  nbio.Stream_Callback_Call(callback, completion)
}

// Copies the result out and clears the in-flight slot before calling back, so a retirement
// that submits again finds the Stream free.
func stream_file_retire(completion nbio.Completion_Handle) {
  state := (*Stream_File)(unsafe.Pointer(completion))
  caller := state.Caller_Completion
  callback := state.Callback
  state.Caller_Completion = nil
  state.Callback = nbio.Stream_Callback{}
  caller.Data = completion.Data
  caller.Error = completion.Error
  nbio.Stream_Callback_Call(callback, caller)
}

// Resolves a frame-skip count to a file and line via runtime.Caller, the composition tier's one
// sanctioned reach into the runtime. The file string points into the binary's line table, so
// nothing is copied. An unknown frame reports an empty file at line 0.
func operating_system_caller(skip int) (file string, line int) {
  _, file, line, ok := runtime.Caller(skip + CALLER_BASE_FRAMES)
  if !ok {
    return "", 0
  }
  return file, line
}

// Trace logs a trace-level line through Default.
func Trace(message string, fields ...jlog.Field) {
  jlog.Logger_Trace(&Default, message, fields...)
}

// Debug logs a debug-level line through Default.
func Debug(message string, fields ...jlog.Field) {
  jlog.Logger_Debug(&Default, message, fields...)
}

// Info logs an info-level line through Default.
func Info(message string, fields ...jlog.Field) {
  jlog.Logger_Info(&Default, message, fields...)
}

// Warn logs a warn-level line through Default.
func Warn(message string, fields ...jlog.Field) {
  jlog.Logger_Warn(&Default, message, fields...)
}

// Error logs an error-level line through Default.
func Error(message string, fields ...jlog.Field) {
  jlog.Logger_Error(&Default, message, fields...)
}

// Log logs a line with no severity field through Default.
func Log(message string, fields ...jlog.Field) {
  jlog.Logger_Log(&Default, message, fields...)
}

// New re-exports jlog.New for building an independent logger from this import.
func New(
  configuration *jlog.Logger_Configuration,
  storage []byte,
  floor jlog.Level,
) (logger jlog.Logger) {
  return jlog.New(configuration, storage, floor)
}

// Logger_Trace re-exports jlog.Logger_Trace.
func Logger_Trace(logger *jlog.Logger, message string, fields ...jlog.Field) {
  jlog.Logger_Trace(logger, message, fields...)
}

// Logger_Debug re-exports jlog.Logger_Debug.
func Logger_Debug(logger *jlog.Logger, message string, fields ...jlog.Field) {
  jlog.Logger_Debug(logger, message, fields...)
}

// Logger_Info re-exports jlog.Logger_Info.
func Logger_Info(logger *jlog.Logger, message string, fields ...jlog.Field) {
  jlog.Logger_Info(logger, message, fields...)
}

// Logger_Warn re-exports jlog.Logger_Warn.
func Logger_Warn(logger *jlog.Logger, message string, fields ...jlog.Field) {
  jlog.Logger_Warn(logger, message, fields...)
}

// Logger_Error re-exports jlog.Logger_Error.
func Logger_Error(logger *jlog.Logger, message string, fields ...jlog.Field) {
  jlog.Logger_Error(logger, message, fields...)
}

// Logger_Log re-exports jlog.Logger_Log.
func Logger_Log(logger *jlog.Logger, message string, fields ...jlog.Field) {
  jlog.Logger_Log(logger, message, fields...)
}

// Logger_At_Level re-exports jlog.Logger_At_Level.
func Logger_At_Level(logger *jlog.Logger, level jlog.Level, message string, fields ...jlog.Field) {
  jlog.Logger_At_Level(logger, level, message, fields...)
}

// Logger_With re-exports jlog.Logger_With.
func Logger_With(
  parent *jlog.Logger,
  storage []byte,
  prefix jlog.Buffer,
  fields ...jlog.Field,
) (child jlog.Logger) {
  return jlog.Logger_With(parent, storage, prefix, fields...)
}

// String re-exports jlog.String.
func String[T ~string](key jlog.Key, value T) (field jlog.Field) {
  return jlog.String(key, value)
}

// Integer re-exports jlog.Integer.
func Integer[T ~int](key jlog.Key, value T) (field jlog.Field) {
  return jlog.Integer(key, value)
}

// Int8 re-exports jlog.Int8.
func Int8[T ~int8](key jlog.Key, value T) (field jlog.Field) {
  return jlog.Int8(key, value)
}

// Int16 re-exports jlog.Int16.
func Int16[T ~int16](key jlog.Key, value T) (field jlog.Field) {
  return jlog.Int16(key, value)
}

// Int32 re-exports jlog.Int32.
func Int32[T ~int32](key jlog.Key, value T) (field jlog.Field) {
  return jlog.Int32(key, value)
}

// Int64 re-exports jlog.Int64.
func Int64[T ~int64](key jlog.Key, value T) (field jlog.Field) {
  return jlog.Int64(key, value)
}

// Uint re-exports jlog.Uint.
func Uint[T ~uint](key jlog.Key, value T) (field jlog.Field) {
  return jlog.Uint(key, value)
}

// Uint8 re-exports jlog.Uint8.
func Uint8[T ~uint8](key jlog.Key, value T) (field jlog.Field) {
  return jlog.Uint8(key, value)
}

// Uint16 re-exports jlog.Uint16.
func Uint16[T ~uint16](key jlog.Key, value T) (field jlog.Field) {
  return jlog.Uint16(key, value)
}

// Uint32 re-exports jlog.Uint32.
func Uint32[T ~uint32](key jlog.Key, value T) (field jlog.Field) {
  return jlog.Uint32(key, value)
}

// Uint64 re-exports jlog.Uint64.
func Uint64[T ~uint64](key jlog.Key, value T) (field jlog.Field) {
  return jlog.Uint64(key, value)
}

// Uintptr re-exports jlog.Uintptr.
func Uintptr[T ~uintptr](key jlog.Key, value T) (field jlog.Field) {
  return jlog.Uintptr(key, value)
}

// Float32 re-exports jlog.Float32.
func Float32[T ~float32](key jlog.Key, value T) (field jlog.Field) {
  return jlog.Float32(key, value)
}

// Float64 re-exports jlog.Float64.
func Float64[T ~float64](key jlog.Key, value T) (field jlog.Field) {
  return jlog.Float64(key, value)
}

// Boolean re-exports jlog.Boolean.
func Boolean[T ~bool](key jlog.Key, value T) (field jlog.Field) {
  return jlog.Boolean(key, value)
}

// Bytes re-exports jlog.Bytes.
func Bytes(key jlog.Key, value []byte) (field jlog.Field) {
  return jlog.Bytes(key, value)
}

// Hexadecimal re-exports jlog.Hexadecimal.
func Hexadecimal(key jlog.Key, value []byte) (field jlog.Field) {
  return jlog.Hexadecimal(key, value)
}

// Raw_JSON re-exports jlog.Raw_JSON.
func Raw_JSON(key jlog.Key, value []byte) (field jlog.Field) {
  return jlog.Raw_JSON(key, value)
}

// Time re-exports jlog.Time.
func Time(key jlog.Key, value time.Moment) (field jlog.Field) {
  return jlog.Time(key, value)
}

// Duration re-exports jlog.Duration.
func Duration(key jlog.Key, value time.Duration) (field jlog.Field) {
  return jlog.Duration(key, value)
}

// Strings re-exports jlog.Strings.
func Strings(key jlog.Key, value []string) (field jlog.Field) {
  return jlog.Strings(key, value)
}

// Integers re-exports jlog.Integers.
func Integers(key jlog.Key, value []int) (field jlog.Field) {
  return jlog.Integers(key, value)
}

// Floats64 re-exports jlog.Floats64.
func Floats64(key jlog.Key, value []float64) (field jlog.Field) {
  return jlog.Floats64(key, value)
}

// Booleans re-exports jlog.Booleans.
func Booleans(key jlog.Key, value []bool) (field jlog.Field) {
  return jlog.Booleans(key, value)
}

// Durations re-exports jlog.Durations.
func Durations(key jlog.Key, value []time.Duration) (field jlog.Field) {
  return jlog.Durations(key, value)
}

// Err re-exports jlog.Err.
func Err(value error) (field jlog.Field) {
  return jlog.Err(value)
}

// Timestamp re-exports jlog.Timestamp.
func Timestamp() (field jlog.Field) {
  return jlog.Timestamp()
}

// Caller re-exports jlog.Caller.
func Caller() (field jlog.Field) {
  return jlog.Caller()
}

// New_Level_Filter re-exports jlog.New_Level_Filter.
func New_Level_Filter(
  filter *jlog.Level_Filter,
  stream nbio.Stream,
  floor jlog.Level,
  level_field_name string,
) {
  jlog.New_Level_Filter(filter, stream, floor, level_field_name)
}
