// Package jlog is a zero-allocation, dependency-injected structured JSON logger.
//
// The house linter bans methods (except those satisfying a stdlib interface) and
// names a free function after its first parameter's type, so there is no fluent
// builder: a log line is one Logger_-prefixed call taking the logger, a message,
// and a variadic list of Field values built by the constructors.
//
//	configuration := jlog.Logger_Configuration{Stream: sink, Clock: clock}
//	logger := jlog.New(&configuration, storage[:], jlog.LEVEL_INFO)
//	jlog.Logger_Info(logger, "request done",
//	    jlog.String("method", method),
//	    jlog.Integer("status", status),
//	)
//	// {"level":"info","method":"GET","status":200,"message":"request done"}
//
// Output is flat: every value is a scalar or a scalar slice. Nested objects are
// deliberately not supported. Flat keys (prefix when grouping is wanted, e.g.
// addr_city) are far easier to query with jq and friends than traversing nested
// structures, and nesting is the one construct that would force an allocation in
// this model. For the flattening rationale see
// documentation/resources/kellybrazil.Tips_On_Adding_JSON_Output_To_Your_CLI_App.md
// ("Flatten the Structure"). Embed a pre-encoded blob with Raw_JSON when
// structure is truly unavoidable.
//
// jlog allocates nothing, ever: the configuration, the line storage, and every prefix are
// caller-owned, and the variadic Field slice stays on the caller's stack. The price is
// ownership: a Logger belongs to one goroutine. Goroutines share a Logger_Configuration and each
// keep their own Logger; ordering and concurrency toward the sink belong to the Stream below
// (shared/std/sync/diode).
package jlog

import "unsafe"

import "local/james-orcales/shared/std/sim/aver/default"
import "local/james-orcales/shared/std/sim/nbio"
import "local/james-orcales/shared/std/sim/time"
import "local/james-orcales/shared/std/strconv"
import "local/james-orcales/shared/std/unicode/utf8"

// LINE_BYTES_MAXIMUM bounds the line storage a caller may hand a Logger; a line that outgrows
// its storage is dropped and counted.
const LINE_BYTES_MAXIMUM = 1 << 16

// PREFIX_BYTES_MAXIMUM bounds the prefix storage a caller may hand a child Logger.
const PREFIX_BYTES_MAXIMUM = 1 << 12

// DEFAULT_CALLER_SKIP passes zero frames because the injected Caller owns its own skip offset.
const DEFAULT_CALLER_SKIP = 0

// FLOAT_BITS_32 formats a Float32 at 32-bit precision so it round-trips as a float32.
const FLOAT_BITS_32 = 32

// FLOAT_BITS_64 formats a Float64 at full 64-bit precision, the widest JSON numbers carry.
const FLOAT_BITS_64 = 64

// A float of this magnitude or beyond renders in exponent form, matching the
// cutoffs JSON encoders use.
const FLOAT_EXPONENT_LOW = 1e-6

// FLOAT_EXPONENT_HIGH is the upper cutoff; at or above it, magnitudes render in exponent form.
const FLOAT_EXPONENT_HIGH = 1e21

// LEVEL_LABEL_SIZE_MAXIMUM is the fixed human severity width.
const LEVEL_LABEL_SIZE_MAXIMUM = len("TRC")

// FLOAT64_MAGNITUDE_MASK removes only the IEEE-754 sign bit.
const FLOAT64_MAGNITUDE_MASK uint64 = 1<<63 - 1

// FLOAT64_EXPONENT_MASK selects the IEEE-754 exponent.
const FLOAT64_EXPONENT_MASK uint64 = 0x7ff0000000000000

// FLOAT64_FRACTION_MASK selects the IEEE-754 significand tail.
const FLOAT64_FRACTION_MASK uint64 = 0x000fffffffffffff

// FLOAT64_POSITIVE_INFINITY_BITS is the sole positive infinite encoding.
const FLOAT64_POSITIVE_INFINITY_BITS uint64 = FLOAT64_EXPONENT_MASK

// FLOAT64_NEGATIVE_INFINITY_BITS adds the sign bit to positive infinity.
const FLOAT64_NEGATIVE_INFINITY_BITS uint64 = ^FLOAT64_MAGNITUDE_MASK | FLOAT64_EXPONENT_MASK

// FLOAT64_MANTISSA_BIT_COUNT is the stored IEEE-754 fraction width.
const FLOAT64_MANTISSA_BIT_COUNT = 52

// FLOAT64_EXPONENT_BIT_COUNT is the stored IEEE-754 exponent width.
const FLOAT64_EXPONENT_BIT_COUNT = 11

// FLOAT64_EXPONENT_BIAS shifts the stored exponent to its binary power.
const FLOAT64_EXPONENT_BIAS = -1023

// FLOAT32_MANTISSA_BIT_COUNT is the stored IEEE-754 fraction width.
const FLOAT32_MANTISSA_BIT_COUNT = 23

// FLOAT32_EXPONENT_BIT_COUNT is the stored IEEE-754 exponent width.
const FLOAT32_EXPONENT_BIT_COUNT = 8

// FLOAT32_EXPONENT_BIAS shifts the stored exponent to its binary power.
const FLOAT32_EXPONENT_BIAS = -127

// FLOAT64_BINARY_SHIFT_MAXIMUM reaches the least nonzero float64 from its mantissa.
const FLOAT64_BINARY_SHIFT_MAXIMUM = 1074

// FLOAT_DECIMAL_DIGIT_CAPACITY follows one possible decimal digit per binary shift,
// plus the widest starting unsigned mantissa.
const FLOAT_DECIMAL_DIGIT_CAPACITY = FLOAT64_BINARY_SHIFT_MAXIMUM +
  strconv.DIGIT_TEXT_SIZE_MAXIMUM

// A configuration default is a distinct type so string_or's two parameters never
// repeat a type, which the input-struct rule would otherwise force into a struct.
type Default_String string

// DEFAULT_TIMESTAMP_FIELD_NAME keys the timestamp; "time" is the common structured-log key.
const DEFAULT_TIMESTAMP_FIELD_NAME Default_String = "time"

// DEFAULT_LEVEL_FIELD_NAME keys the severity; "level" is the name log tooling greps for.
const DEFAULT_LEVEL_FIELD_NAME Default_String = "level"

// DEFAULT_MESSAGE_FIELD_NAME keys the free-text human message; "message" is the widely used name.
const DEFAULT_MESSAGE_FIELD_NAME Default_String = "message"

// DEFAULT_ERROR_FIELD_NAME keys the string Err emits; "error" is the conventional error key.
const DEFAULT_ERROR_FIELD_NAME Default_String = "error"

// DEFAULT_CALLER_FIELD_NAME keys the "file:line" location Caller and Auto_Caller emit.
const DEFAULT_CALLER_FIELD_NAME Default_String = "caller"

// DEFAULT_STACK_FIELD_NAME keys the rendered stack trace Err writes beside the error.
const DEFAULT_STACK_FIELD_NAME Default_String = "stack"

// Buffer is a bounded writer over caller storage. Encoders write by index and never grow
// Bytes: a write that does not fit sets Overflow and stops, so the caller drops or reports the
// content instead of holding a fresh heap allocation.
type Buffer struct {
  // Bytes is the caller-owned storage; len(Bytes) is the write limit.
  Bytes []byte
  // Count is the bytes written so far.
  Count int
  // Overflow marks content cut short by the limit; Bytes[:Count] is then not a whole value.
  Overflow bool
}

// CALLER_FILE_SIZE_MAXIMUM bounds the file half of a caller location; a longer path is cut.
const CALLER_FILE_SIZE_MAXIMUM = 1 << 10

// TRANSFORM_SIZE_MAXIMUM bounds one Write through a Console or a Level_Filter; a transform
// that outgrows it retires with nbio.Stream_Buffer_Full.
const TRANSFORM_SIZE_MAXIMUM = LINE_BYTES_MAXIMUM

// MEMBER_COUNT_MAXIMUM bounds the members a Console or Level_Filter reads from one line; a
// line with more passes through unrendered.
const MEMBER_COUNT_MAXIMUM = 256

// Data is one complete encoded line passed to an injected sink.
type Data []byte

// Key is a JSON object key. It is a distinct type so the two-argument field
// constructors never repeat a parameter type and cannot be called with the key and
// value transposed; string literals convert implicitly, a dynamic string key
// needs an explicit Key conversion.
type Key string

// Field_Kind tags which value member a Field carries and how it is encoded.
type Field_Kind uint8

// The value kinds (those whose key is field.Key) occupy the low range and the
// config-keyed kinds (error/timestamp/caller, whose key comes from the config)
// occupy the top, so buffer_encode_field tells them apart with one comparison
// against KIND_ERROR rather than a switch on the hot value path.
const KIND_STRING Field_Kind = 0

// KIND_INTEGER tags a signed integer stored in Number and emitted as a JSON number.
const KIND_INTEGER Field_Kind = 1

// KIND_UNSIGNED tags an unsigned integer stored in Number and emitted as a JSON number.
const KIND_UNSIGNED Field_Kind = 2

// KIND_FLOAT32 tags a float held as bits in Number, rendered at float32 precision.
const KIND_FLOAT32 Field_Kind = 3

// KIND_FLOAT64 tags a float held as bits in Number, rendered at full float64 precision.
const KIND_FLOAT64 Field_Kind = 4

// KIND_BOOLEAN tags a bool packed into Number and emitted as JSON true or false.
const KIND_BOOLEAN Field_Kind = 5

// KIND_BYTES tags a []byte behind Data, emitted as a JSON string with escaping.
const KIND_BYTES Field_Kind = 6

// KIND_HEXADECIMAL tags a []byte behind Data, emitted as a hex-encoded JSON string.
const KIND_HEXADECIMAL Field_Kind = 7

// KIND_RAW_JSON tags a pre-encoded []byte behind Data, copied verbatim into the line.
const KIND_RAW_JSON Field_Kind = 8

// KIND_TIME tags a Moment in Number, rendered as an RFC 3339 UTC timestamp string.
const KIND_TIME Field_Kind = 9

// KIND_DURATION tags a Duration in Number, divided by the config unit before rendering.
const KIND_DURATION Field_Kind = 10

// KIND_STRINGS tags a []string behind Data, emitted as a JSON array of strings.
const KIND_STRINGS Field_Kind = 13

// KIND_INTEGERS tags a []int behind Data, emitted as a JSON array of numbers.
const KIND_INTEGERS Field_Kind = 14

// KIND_FLOATS tags a []float64 behind Data, emitted as a JSON array of numbers.
const KIND_FLOATS Field_Kind = 15

// KIND_BOOLEANS tags a []bool behind Data, emitted as a JSON array of booleans.
const KIND_BOOLEANS Field_Kind = 16

// KIND_DURATIONS tags a []Duration behind Data, each divided by the config unit.
const KIND_DURATIONS Field_Kind = 17

// KIND_ERROR keys from config, not field.Key; it emits the boxed error and optional stack.
const KIND_ERROR Field_Kind = 18

// KIND_TIMESTAMP keys from config; it stamps the realtime clock, carrying no value of its own.
const KIND_TIMESTAMP Field_Kind = 19

// KIND_CALLER keys from config; Number holds the frame skip for the injected Caller lookup.
const KIND_CALLER Field_Kind = 20

// Field is one structured key/value pair, built by a constructor and consumed
// by an emit function. It is a compact 56-byte value (no per-type slots) so a
// variadic of Fields is cheap to build and copy. Reference values — string, []byte,
// and the scalar slices — are packed as a Data pointer plus a Number length and
// reconstructed at encode time, which keeps every field type allocation-free while
// holding the struct small. The Data pointer is a real (GC-scanned) pointer into
// the caller's value, so the backing memory stays alive for the synchronous emit;
// a Field must not be stored and used after its source goes out of scope.
type Field struct {
  // Key is the JSON key. Empty for Err/Timestamp/Caller, which key from config.
  Key Key
  // Boxed holds the error for an Err field.
  Boxed any
  // Data is the backing pointer of a string/[]byte/scalar-slice value, or nil.
  Data unsafe.Pointer
  // Number holds a scalar (int/uint/bool/Moment/Duration/float bits) or, for a
  // Data-backed value, that value's element length; for Caller, the frame skip.
  Number int64
  // Kind selects the active representation and its encoding.
  Kind Field_Kind
}

// Logger_Configuration is the immutable dependency set shared by a logger and all
// its sub-loggers through one pointer, so Logger_With never copies it.
type Logger_Configuration struct {
  // Stream receives each finished line once through nbio.Write.
  Stream nbio.Stream
  // Clock supplies wall-clock readings; injected so tests are deterministic and
  // the library tier never imports stdlib time.
  Clock time.Clock
  // Caller maps a frame-skip count to a file and line; nil disables Caller. The two halves
  // come back apart so the encoder joins them in the line, never in a fresh string.
  Caller func(skip int) (file string, line int)
  // Stack_Marshaler renders an error's stack; nil disables Err stacks.
  Stack_Marshaler func(value error) (stack string)
  // Timestamp_Field_Name is the key used by Timestamp and Auto_Timestamp.
  Timestamp_Field_Name string
  // Level_Field_Name is the key used for the severity field.
  Level_Field_Name string
  // Message_Field_Name is the key used for the trailing message field.
  Message_Field_Name string
  // Error_Field_Name is the key used by Err for the error string.
  Error_Field_Name string
  // Caller_Field_Name is the key used by Caller and Auto_Caller.
  Caller_Field_Name string
  // Stack_Field_Name is the key used by Err for the rendered stack.
  Stack_Field_Name string
  // Duration_Unit divides Duration values before rendering; defaults to NANOSECOND.
  Duration_Unit time.Duration
}

// Line is one logger's encode storage. Completion sits first so the retire callback, which
// receives only the Completion, recovers its Line without a lookup or a captured closure.
type Line struct {
  // Completion is the nbio storage for this line's Write.
  Completion nbio.Completion
  // Buffer is the caller-owned byte budget; a line that outgrows it is dropped, never kept.
  Buffer Buffer
  // Borrowed marks Storage as held by the Stream until it retires the Write. A plain bool,
  // because one goroutine owns the Logger and the root delivers the retirement.
  Borrowed bool
}

// Logger bundles a shared configuration with this logger's own line storage, level floor, and
// fixed prefix fields. One goroutine owns a Logger: it encodes into Line with no
// synchronization, so goroutines share the Configuration, never the Logger. It is used by
// pointer; the zero value is a disabled no-op.
type Logger struct {
  // Configuration is the shared read-only dependency set; nil marks a disabled logger.
  Configuration *Logger_Configuration
  // Line is this logger's encode storage.
  Line Line
  // Floor is the lowest level emitted; lower levels are dropped.
  Floor Level
  // Prefix is the pre-encoded fixed fields ({... fragment) added to every line, in
  // caller-owned storage.
  Prefix Buffer
  // Dropped counts lines that outgrew Line.Buffer or arrived while it was still borrowed.
  Dropped int
  // Auto_Timestamp stamps every line with the realtime clock when set.
  Auto_Timestamp bool
  // Auto_Caller adds the caller location to every line when set.
  Auto_Caller bool
}

// Level is a log severity.
type Level int8

// LEVEL_TRACE is the most verbose level.
const LEVEL_TRACE Level = -1

// LEVEL_DEBUG is the debugging level.
const LEVEL_DEBUG Level = 0

// LEVEL_INFO is the informational level.
const LEVEL_INFO Level = 1

// LEVEL_WARN is the warning level.
const LEVEL_WARN Level = 2

// LEVEL_ERROR is the error level.
const LEVEL_ERROR Level = 3

// LEVEL_NONE is a line with no severity field, emitted by Logger_Log.
const LEVEL_NONE Level = 6

// LEVEL_DISABLED marks a logger that emits nothing.
const LEVEL_DISABLED Level = 7

// Level_String returns the wire name of the level, empty for LEVEL_NONE.
func Level_String(level Level) (name string) {
  switch level {
  case LEVEL_TRACE:
    return "trace"
  case LEVEL_DEBUG:
    return "debug"
  case LEVEL_INFO:
    return "info"
  case LEVEL_WARN:
    return "warn"
  case LEVEL_ERROR:
    return "error"
  case LEVEL_DISABLED:
    return "disabled"
  case LEVEL_NONE:
    return ""
  }
  aver.Always(false, "jlog: invalid level")
  return ""
}

// New returns a logger over configuration, which the caller filled, and the caller's line
// storage. It fills in defaults for empty field names and a zero Duration_Unit in place and
// allocates nothing. Set Auto_Timestamp and Auto_Caller on the result.
func New(configuration *Logger_Configuration, storage []byte, floor Level) (logger Logger) {
  aver.Always(configuration != nil, "jlog: configuration storage is required")
  aver.Always(cap(storage) > 0, "jlog: line storage is required")
  aver.Always(!(cap(storage) > LINE_BYTES_MAXIMUM), "jlog: line storage exceeds bound")
  if configuration.Duration_Unit == 0 {
    configuration.Duration_Unit = time.NANOSECOND
  }
  aver.Always(configuration.Duration_Unit > 0, "jlog: duration unit must be positive")
  configuration.Timestamp_Field_Name = string_or(
    configuration.Timestamp_Field_Name,
    DEFAULT_TIMESTAMP_FIELD_NAME,
  )
  configuration.Level_Field_Name = string_or(
    configuration.Level_Field_Name,
    DEFAULT_LEVEL_FIELD_NAME,
  )
  configuration.Message_Field_Name = string_or(
    configuration.Message_Field_Name,
    DEFAULT_MESSAGE_FIELD_NAME,
  )
  configuration.Error_Field_Name = string_or(
    configuration.Error_Field_Name,
    DEFAULT_ERROR_FIELD_NAME,
  )
  configuration.Caller_Field_Name = string_or(
    configuration.Caller_Field_Name,
    DEFAULT_CALLER_FIELD_NAME,
  )
  configuration.Stack_Field_Name = string_or(
    configuration.Stack_Field_Name,
    DEFAULT_STACK_FIELD_NAME,
  )
  logger.Configuration = configuration
  logger.Line.Buffer = Buffer{Bytes: storage[:cap(storage)]}
  logger.Floor = floor
  return logger
}

func string_or(value string, fallback Default_String) (chosen string) {
  if value == "" {
    return string(fallback)
  }
  return value
}

// Logger_Trace emits a trace-level line.
func Logger_Trace(logger *Logger, message string, fields ...Field) {
  logger_emit(logger, LEVEL_TRACE, message, fields)
}

// Logger_Debug emits a debug-level line.
func Logger_Debug(logger *Logger, message string, fields ...Field) {
  logger_emit(logger, LEVEL_DEBUG, message, fields)
}

// Logger_Info emits an info-level line.
func Logger_Info(logger *Logger, message string, fields ...Field) {
  logger_emit(logger, LEVEL_INFO, message, fields)
}

// Logger_Warn emits a warn-level line.
func Logger_Warn(logger *Logger, message string, fields ...Field) {
  logger_emit(logger, LEVEL_WARN, message, fields)
}

// Logger_Error emits an error-level line.
func Logger_Error(logger *Logger, message string, fields ...Field) {
  logger_emit(logger, LEVEL_ERROR, message, fields)
}

// Logger_Log emits a line with no severity field.
func Logger_Log(logger *Logger, message string, fields ...Field) {
  logger_emit(logger, LEVEL_NONE, message, fields)
}

// Logger_At_Level emits a line at the given level.
func Logger_At_Level(logger *Logger, level Level, message string, fields ...Field) {
  logger_emit(logger, level, message, fields)
}

func logger_emit(logger *Logger, level Level, message string, fields []Field) {
  if logger_is_disabled(logger) {
    return
  }
  if level < logger.Floor {
    return
  }
  configuration := logger.Configuration
  line := &logger.Line
  // The Stream still holds Storage; encoding over it would tear the line in flight.
  if line.Borrowed {
    logger.Dropped++
    return
  }
  buffer := &line.Buffer
  buffer.Count = 0
  buffer.Overflow = false
  buffer_write_begin_marker(buffer)
  buffer_write_level(buffer, level, configuration)
  if logger.Auto_Timestamp {
    buffer_write_realtime(buffer, configuration)
  }
  if logger.Auto_Caller {
    seed := Field{Kind: KIND_CALLER, Number: DEFAULT_CALLER_SKIP}
    buffer_write_caller(buffer, &seed, configuration)
  }
  if logger.Prefix.Count > 1 {
    buffer_write_object_data(buffer, logger.Prefix.Bytes[:logger.Prefix.Count])
  }
  for index := 0; index < len(fields); index++ {
    buffer_encode_field(buffer, &fields[index], configuration)
  }
  if message != "" {
    buffer_write_key(buffer, Key(configuration.Message_Field_Name))
    buffer_write_string(buffer, message)
  }
  buffer_write_end_marker(buffer)
  buffer_write_byte(buffer, '\n')
  // The line outgrew its storage: what was written is a torn line, so it is dropped whole.
  if buffer.Overflow {
    logger.Dropped++
    return
  }
  written := buffer.Bytes[:buffer.Count]
  aver.Always(written[0] == '{', "jlog: line must open with a brace")
  aver.Always(written[len(written)-2] == '}', "jlog: line must close with a brace")
  aver.Always(written[len(written)-1] == '\n', "jlog: line must end with a newline")
  line.Borrowed = true
  nbio.Write(configuration.Stream, &line.Completion, written, line_retire)
}

// Stream failures have no reporting channel above a logger, so retirement only returns the
// storage. It records state and continues nothing: the root drives.
func line_retire(completion nbio.Completion_Handle) {
  line := (*Line)(unsafe.Pointer(completion))
  line.Borrowed = false
}

func logger_is_disabled(logger *Logger) (disabled bool) {
  if logger == nil {
    return true
  }
  if logger.Configuration == nil {
    return true
  }
  if logger.Floor == LEVEL_DISABLED {
    return true
  }
  return false
}

// Logger_With returns a child logger carrying fields as a fixed prefix on every line. storage
// is the child's line buffer and prefix its prefix buffer, which must hold the parent's prefix
// plus the new fields; both are caller-owned. A child of a disabled parent is disabled; a
// prefix that outgrows its storage is a configuration error and panics.
func Logger_With(parent *Logger, storage []byte, prefix Buffer, fields ...Field) (child Logger) {
  if logger_is_disabled(parent) {
    return Logger{}
  }
  aver.Always(cap(storage) > 0, "jlog: line storage is required")
  aver.Always(!(cap(storage) > LINE_BYTES_MAXIMUM), "jlog: line storage exceeds bound")
  aver.Always(!(len(prefix.Bytes) > PREFIX_BYTES_MAXIMUM), "jlog: prefix storage exceeds bound")
  child.Configuration = parent.Configuration
  child.Line.Buffer = Buffer{Bytes: storage[:cap(storage)]}
  child.Floor = parent.Floor
  child.Auto_Timestamp = parent.Auto_Timestamp
  child.Auto_Caller = parent.Auto_Caller
  child.Prefix = Buffer{Bytes: prefix.Bytes}
  if parent.Prefix.Count > 0 {
    buffer_write_bytes(&child.Prefix, parent.Prefix.Bytes[:parent.Prefix.Count])
  } else {
    buffer_write_begin_marker(&child.Prefix)
  }
  for index := 0; index < len(fields); index++ {
    buffer_encode_field(&child.Prefix, &fields[index], parent.Configuration)
  }
  aver.Always(!child.Prefix.Overflow, "jlog: prefix exceeds its storage")
  return child
}

// The primitive writers below are the only place bytes reach storage; every encoder goes
// through them so the bound is checked once, here.
func buffer_write_byte(buffer *Buffer, value byte) {
  if !(buffer.Count < len(buffer.Bytes)) {
    buffer.Overflow = true
    return
  }
  buffer.Bytes[buffer.Count] = value
  buffer.Count++
}

func buffer_write_text(buffer *Buffer, value string) {
  if len(value) > len(buffer.Bytes)-buffer.Count {
    buffer.Overflow = true
    return
  }
  copy(buffer.Bytes[buffer.Count:], value)
  buffer.Count += len(value)
}

func buffer_write_bytes(buffer *Buffer, value []byte) {
  if len(value) > len(buffer.Bytes)-buffer.Count {
    buffer.Overflow = true
    return
  }
  copy(buffer.Bytes[buffer.Count:], value)
  buffer.Count += len(value)
}

// String builds a string field.
func String[T ~string](key Key, value T) (field Field) {
  s := string(value)
  return Field{
    Key:    key,
    Kind:   KIND_STRING,
    Data:   unsafe.Pointer(unsafe.StringData(s)),
    Number: int64(len(s)),
  }
}

// Integer builds a signed-integer field.
func Integer[T ~int](key Key, value T) (field Field) {
  return Field{Key: key, Kind: KIND_INTEGER, Number: int64(value)}
}

// Int8 builds an 8-bit signed-integer field.
func Int8[T ~int8](key Key, value T) (field Field) {
  return Field{Key: key, Kind: KIND_INTEGER, Number: int64(value)}
}

// Int16 builds a 16-bit signed-integer field.
func Int16[T ~int16](key Key, value T) (field Field) {
  return Field{Key: key, Kind: KIND_INTEGER, Number: int64(value)}
}

// Int32 builds a 32-bit signed-integer field.
func Int32[T ~int32](key Key, value T) (field Field) {
  return Field{Key: key, Kind: KIND_INTEGER, Number: int64(value)}
}

// Int64 builds a 64-bit signed-integer field.
func Int64[T ~int64](key Key, value T) (field Field) {
  return Field{Key: key, Kind: KIND_INTEGER, Number: int64(value)}
}

// Uint builds an unsigned-integer field.
func Uint[T ~uint](key Key, value T) (field Field) {
  return Field{Key: key, Kind: KIND_UNSIGNED, Number: int64(value)}
}

// Uint8 builds an 8-bit unsigned-integer field.
func Uint8[T ~uint8](key Key, value T) (field Field) {
  return Field{Key: key, Kind: KIND_UNSIGNED, Number: int64(value)}
}

// Uint16 builds a 16-bit unsigned-integer field.
func Uint16[T ~uint16](key Key, value T) (field Field) {
  return Field{Key: key, Kind: KIND_UNSIGNED, Number: int64(value)}
}

// Uint32 builds a 32-bit unsigned-integer field.
func Uint32[T ~uint32](key Key, value T) (field Field) {
  return Field{Key: key, Kind: KIND_UNSIGNED, Number: int64(value)}
}

// Uint64 builds a 64-bit unsigned-integer field.
func Uint64[T ~uint64](key Key, value T) (field Field) {
  return Field{Key: key, Kind: KIND_UNSIGNED, Number: int64(value)}
}

// Uintptr builds an unsigned-pointer-sized integer field.
func Uintptr[T ~uintptr](key Key, value T) (field Field) {
  return Field{Key: key, Kind: KIND_UNSIGNED, Number: int64(value)}
}

// Float32 builds a float field rendered at float32 precision.
func Float32[T ~float32](key Key, value T) (field Field) {
  return Field{Key: key, Kind: KIND_FLOAT32, Number: int64(float64_bits(float64(value)))}
}

// Float64 builds a float field rendered at float64 precision.
func Float64[T ~float64](key Key, value T) (field Field) {
  return Field{Key: key, Kind: KIND_FLOAT64, Number: int64(float64_bits(float64(value)))}
}

// Boolean builds a boolean field.
func Boolean[T ~bool](key Key, value T) (field Field) {
  return Field{Key: key, Kind: KIND_BOOLEAN, Number: boolean_to_int64(bool(value))}
}

// Bytes builds a field whose []byte value is rendered as a JSON string.
func Bytes(key Key, value []byte) (field Field) {
  return Field{
    Key:    key,
    Kind:   KIND_BYTES,
    Data:   unsafe.Pointer(unsafe.SliceData(value)),
    Number: int64(len(value)),
  }
}

// Hexadecimal builds a field whose []byte value is rendered as a hex string.
func Hexadecimal(key Key, value []byte) (field Field) {
  return Field{
    Key:    key,
    Kind:   KIND_HEXADECIMAL,
    Data:   unsafe.Pointer(unsafe.SliceData(value)),
    Number: int64(len(value)),
  }
}

// Raw_JSON builds a field whose pre-encoded JSON value is copied verbatim.
func Raw_JSON(key Key, value []byte) (field Field) {
  return Field{
    Key:    key,
    Kind:   KIND_RAW_JSON,
    Data:   unsafe.Pointer(unsafe.SliceData(value)),
    Number: int64(len(value)),
  }
}

// Time builds a field rendering a Moment as an RFC 3339 UTC timestamp string.
func Time(key Key, value time.Moment) (field Field) {
  return Field{Key: key, Kind: KIND_TIME, Number: int64(value)}
}

// Duration builds a field rendering a Duration as an integer in the logger unit.
func Duration(key Key, value time.Duration) (field Field) {
  return Field{Key: key, Kind: KIND_DURATION, Number: int64(value)}
}

// Err builds the error field; its key, and an optional stack, come from the config.
func Err(value error) (field Field) {
  return Field{Kind: KIND_ERROR, Boxed: value}
}

// Timestamp builds a field stamping the realtime clock under the configured key.
func Timestamp() (field Field) {
  return Field{Kind: KIND_TIMESTAMP}
}

// Caller builds a field adding the injected caller location under the configured key.
func Caller() (field Field) {
  return Field{Kind: KIND_CALLER, Number: DEFAULT_CALLER_SKIP}
}

// Strings builds a field rendering a []string as a JSON array.
func Strings(key Key, value []string) (field Field) {
  return Field{
    Key:    key,
    Kind:   KIND_STRINGS,
    Data:   unsafe.Pointer(unsafe.SliceData(value)),
    Number: int64(len(value)),
  }
}

// Integers builds a field rendering a []int as a JSON array.
func Integers(key Key, value []int) (field Field) {
  return Field{
    Key:    key,
    Kind:   KIND_INTEGERS,
    Data:   unsafe.Pointer(unsafe.SliceData(value)),
    Number: int64(len(value)),
  }
}

// Floats64 builds a field rendering a []float64 as a JSON array.
func Floats64(key Key, value []float64) (field Field) {
  return Field{
    Key:    key,
    Kind:   KIND_FLOATS,
    Data:   unsafe.Pointer(unsafe.SliceData(value)),
    Number: int64(len(value)),
  }
}

// Booleans builds a field rendering a []bool as a JSON array.
func Booleans(key Key, value []bool) (field Field) {
  return Field{
    Key:    key,
    Kind:   KIND_BOOLEANS,
    Data:   unsafe.Pointer(unsafe.SliceData(value)),
    Number: int64(len(value)),
  }
}

// Durations builds a field rendering a []Duration as a JSON array.
func Durations(key Key, value []time.Duration) (field Field) {
  return Field{
    Key:    key,
    Kind:   KIND_DURATIONS,
    Data:   unsafe.Pointer(unsafe.SliceData(value)),
    Number: int64(len(value)),
  }
}

func boolean_to_int64(value bool) (number int64) {
  if value {
    return 1
  }
  return 0
}

func float64_bits(value float64) (bits uint64) {
  return *(*uint64)(unsafe.Pointer(&value))
}

func float64_from_bits(bits uint64) (value float64) {
  return *(*float64)(unsafe.Pointer(&bits))
}

func float64_is_nan(bits uint64) (nan bool) {
  if bits&FLOAT64_EXPONENT_MASK != FLOAT64_EXPONENT_MASK {
    return false
  }
  return bits&FLOAT64_FRACTION_MASK != 0
}

func float32_bits(value float32) (bits uint32) {
  return *(*uint32)(unsafe.Pointer(&value))
}

// Float_Information states one IEEE-754 storage layout.
type Float_Information struct {
  // Mantissa_Bit_Count locates the implicit leading bit.
  Mantissa_Bit_Count uint
  // Exponent_Bit_Count locates the sign and exponent mask.
  Exponent_Bit_Count uint
  // Exponent_Bias restores the stored exponent's binary power.
  Exponent_Bias int
}

// Decimal stores an exact finite binary float as decimal digits and a decimal point.
type Decimal struct {
  // Digits hold the exact expansion over the caller's array, so nothing heap-falls-back.
  Digits []byte
  // Count excludes meaningless trailing zeroes.
  Count int
  // Point stays independent from Count so trimmed integers keep magnitude.
  Point int
  // Truncated preserves rounding direction if fixed storage ever fills.
  Truncated bool
}

func decimal_trim(value *Decimal) {
  for value.Count > 0 {
    if value.Digits[value.Count-1] != '0' {
      break
    }
    value.Count--
  }
  if value.Count == 0 {
    value.Point = 0
  }
}

func decimal_assign(value *Decimal, number uint64) {
  var reverse [strconv.DIGIT_TEXT_SIZE_MAXIMUM]byte
  count := 0
  for number > 0 {
    quotient := number / 10
    reverse[count] = byte(number-quotient*10) + '0'
    count++
    number = quotient
  }
  value.Count = count
  value.Point = count
  reverse_index := count
  for index := 0; index < count; index++ {
    reverse_index--
    value.Digits[index] = reverse[reverse_index]
  }
  decimal_trim(value)
}

func decimal_multiply_two(value *Decimal) {
  carry := byte(0)
  for index := value.Count - 1; !(index < 0); index-- {
    doubled := (value.Digits[index]-'0')*2 + carry
    value.Digits[index] = doubled%10 + '0'
    carry = doubled / 10
  }
  if carry > 0 {
    copy(value.Digits[1:], value.Digits[:value.Count])
    value.Digits[0] = carry + '0'
    value.Count++
    value.Point++
  }
  decimal_trim(value)
}

func decimal_divide_two(value *Decimal) {
  carry := byte(0)
  write_index := 0
  for read_index := 0; read_index < value.Count; read_index++ {
    combined := carry*10 + value.Digits[read_index] - '0'
    quotient := combined / 2
    carry = combined % 2
    if write_index == 0 {
      if quotient == 0 {
        value.Point--
        continue
      }
    }
    value.Digits[write_index] = quotient + '0'
    write_index++
  }
  if carry > 0 {
    if write_index < len(value.Digits) {
      value.Digits[write_index] = '5'
      write_index++
    } else {
      value.Truncated = true
    }
  }
  value.Count = write_index
  decimal_trim(value)
}

func decimal_shift(value *Decimal, shift int) {
  for shift > 0 {
    decimal_multiply_two(value)
    shift--
  }
  for shift < 0 {
    decimal_divide_two(value)
    shift++
  }
}

func decimal_should_round_up(value *Decimal, count int) (up bool) {
  if count < 0 {
    return false
  }
  if !(count < value.Count) {
    return false
  }
  if value.Digits[count] == '5' {
    if count+1 == value.Count {
      if value.Truncated {
        return true
      }
      if count > 0 {
        return (value.Digits[count-1]-'0')%2 != 0
      }
      return false
    }
  }
  return !(value.Digits[count] < '5')
}

func decimal_round_down(value *Decimal, count int) {
  if count < 0 {
    return
  }
  if !(count < value.Count) {
    return
  }
  value.Count = count
  decimal_trim(value)
}

func decimal_round_up(value *Decimal, count int) {
  if count < 0 {
    return
  }
  if !(count < value.Count) {
    return
  }
  for index := count - 1; !(index < 0); index-- {
    if value.Digits[index] < '9' {
      value.Digits[index]++
      value.Count = index + 1
      return
    }
  }
  value.Digits[0] = '1'
  value.Count = 1
  value.Point++
}

func decimal_round(value *Decimal, count int) {
  if decimal_should_round_up(value, count) {
    decimal_round_up(value, count)
    return
  }
  decimal_round_down(value, count)
}

func decimal_round_shortest(
  value *Decimal,
  mantissa uint64,
  exponent int,
  information Float_Information,
) {
  if mantissa == 0 {
    value.Count = 0
    return
  }
  minimum_exponent := information.Exponent_Bias + 1
  if exponent > minimum_exponent {
    decimal_distance := 332 * (value.Point - value.Count)
    binary_distance := 100 * (exponent - int(information.Mantissa_Bit_Count))
    if !(decimal_distance < binary_distance) {
      return
    }
  }
  var upper_digits, lower_digits [FLOAT_DECIMAL_DIGIT_CAPACITY]byte
  upper := Decimal{Digits: upper_digits[:]}
  lower := Decimal{Digits: lower_digits[:]}
  decimal_shortest_bounds(
    &upper, &lower, mantissa, exponent, minimum_exponent, information,
  )
  decimal_round_between(value, &lower, &upper, mantissa%2 == 0)
}

func decimal_shortest_bounds(
  upper *Decimal,
  lower *Decimal,
  mantissa uint64,
  exponent int,
  minimum_exponent int,
  information Float_Information,
) {
  decimal_assign(upper, mantissa*2+1)
  decimal_shift(upper, exponent-int(information.Mantissa_Bit_Count)-1)
  var lower_mantissa uint64
  var lower_exponent int
  if mantissa > 1<<information.Mantissa_Bit_Count {
    lower_mantissa = mantissa - 1
    lower_exponent = exponent
  } else if exponent == minimum_exponent {
    lower_mantissa = mantissa - 1
    lower_exponent = exponent
  } else {
    lower_mantissa = mantissa*2 - 1
    lower_exponent = exponent - 1
  }
  decimal_assign(lower, lower_mantissa*2+1)
  decimal_shift(lower, lower_exponent-int(information.Mantissa_Bit_Count)-1)
}

func decimal_round_between(
  value *Decimal, lower *Decimal, upper *Decimal, inclusive bool,
) {
  upper_delta := uint8(0)
  for upper_index := 0; ; upper_index++ {
    middle_index := upper_index - upper.Point + value.Point
    if !(middle_index < value.Count) {
      break
    }
    lower_index := upper_index - upper.Point + lower.Point
    lower_digit := byte('0')
    if !(lower_index < 0) {
      if lower_index < lower.Count {
        lower_digit = lower.Digits[lower_index]
      }
    }
    middle_digit := byte('0')
    if !(middle_index < 0) {
      middle_digit = value.Digits[middle_index]
    }
    upper_digit := byte('0')
    if upper_index < upper.Count {
      upper_digit = upper.Digits[upper_index]
    }
    down := lower_digit != middle_digit
    if inclusive {
      if lower_index+1 == lower.Count {
        down = true
      }
    }
    if upper_delta == 0 {
      if middle_digit+1 < upper_digit {
        upper_delta = 2
      } else if middle_digit != upper_digit {
        upper_delta = 1
      }
    } else if upper_delta == 1 {
      if middle_digit != '9' {
        upper_delta = 2
      } else if upper_digit != '0' {
        upper_delta = 2
      }
    }
    up := false
    if upper_delta > 0 {
      up = inclusive
      if upper_delta > 1 {
        up = true
      }
      if upper_index+1 < upper.Count {
        up = true
      }
    }
    if down {
      if up {
        decimal_round(value, middle_index+1)
        return
      }
      decimal_round_down(value, middle_index+1)
      return
    }
    if up {
      decimal_round_up(value, middle_index+1)
      return
    }
  }
}

// The config-keyed kinds (timestamp, caller, error) supply their own key; every
// other kind writes field.Key then its value.
func buffer_encode_field(buffer *Buffer, field *Field, configuration *Logger_Configuration) {
  if !(field.Kind < KIND_ERROR) {
    switch field.Kind {
    case KIND_ERROR:
      buffer_write_error(buffer, field, configuration)
    case KIND_TIMESTAMP:
      buffer_write_realtime(buffer, configuration)
    case KIND_CALLER:
      buffer_write_caller(buffer, field, configuration)
    }
    return
  }
  buffer_write_key(buffer, field.Key)
  count := int(field.Number)
  switch field.Kind {
  case KIND_STRING:
    buffer_write_string(buffer, unsafe.String((*byte)(field.Data), count))
  case KIND_INTEGER:
    buffer_write_int64(buffer, field.Number)
  case KIND_UNSIGNED:
    buffer_write_uint64(buffer, uint64(field.Number))
  case KIND_FLOAT32:
    buffer_write_float(buffer, float64_from_bits(uint64(field.Number)), FLOAT_BITS_32)
  case KIND_FLOAT64:
    buffer_write_float(buffer, float64_from_bits(uint64(field.Number)), FLOAT_BITS_64)
  case KIND_BOOLEAN:
    buffer_write_boolean(buffer, field.Number == 1)
  case KIND_BYTES:
    buffer_write_quoted_bytes(buffer, unsafe.Slice((*byte)(field.Data), count))
  case KIND_HEXADECIMAL:
    buffer_write_hexadecimal(buffer, unsafe.Slice((*byte)(field.Data), count))
  case KIND_RAW_JSON:
    buffer_write_bytes(buffer, unsafe.Slice((*byte)(field.Data), count))
  case KIND_TIME:
    buffer_write_rfc3339(buffer, field.Number)
  case KIND_DURATION:
    buffer_write_int64(buffer, field.Number/int64(configuration.Duration_Unit))
  case KIND_STRINGS:
    buffer_write_strings(buffer, unsafe.Slice((*string)(field.Data), count))
  case KIND_INTEGERS:
    buffer_write_integers(buffer, unsafe.Slice((*int)(field.Data), count))
  case KIND_FLOATS:
    buffer_write_floats(buffer, unsafe.Slice((*float64)(field.Data), count))
  case KIND_BOOLEANS:
    buffer_write_booleans(buffer, unsafe.Slice((*bool)(field.Data), count))
  case KIND_DURATIONS:
    values := unsafe.Slice((*time.Duration)(field.Data), count)
    buffer_write_durations(buffer, values, configuration.Duration_Unit)
  }
}

func buffer_write_realtime(buffer *Buffer, configuration *Logger_Configuration) {
  buffer_write_key(buffer, Key(configuration.Timestamp_Field_Name))
  moment := time.Clock_Now_Realtime(configuration.Clock)
  buffer_write_rfc3339(buffer, int64(moment))
}

// The location is joined as "file:line" inside the value, so a file with a quote or a
// backslash is escaped like any other string.
func buffer_write_caller(buffer *Buffer, field *Field, configuration *Logger_Configuration) {
  if configuration.Caller == nil {
    return
  }
  file, line := configuration.Caller(int(field.Number))
  if len(file) > CALLER_FILE_SIZE_MAXIMUM {
    file = file[:CALLER_FILE_SIZE_MAXIMUM]
  }
  buffer_write_key(buffer, Key(configuration.Caller_Field_Name))
  buffer_write_byte(buffer, '"')
  buffer_write_string_body(buffer, file)
  buffer_write_byte(buffer, ':')
  buffer_write_int64(buffer, int64(line))
  buffer_write_byte(buffer, '"')
}

// The stack field precedes the error field, matching the convention that context
// comes before the value it explains.
func buffer_write_error(buffer *Buffer, field *Field, configuration *Logger_Configuration) {
  error_value, _ := field.Boxed.(error)
  if configuration.Stack_Marshaler != nil {
    if error_value != nil {
      stack := configuration.Stack_Marshaler(error_value)
      buffer_write_key(buffer, Key(configuration.Stack_Field_Name))
      buffer_write_string(buffer, stack)
    }
  }
  buffer_write_key(buffer, Key(configuration.Error_Field_Name))
  if error_value == nil {
    buffer_write_text(buffer, "null")
    return
  }
  buffer_write_string(buffer, error_value.Error())
}

func buffer_write_strings(buffer *Buffer, values []string) {
  buffer_write_byte(buffer, '[')
  for index := 0; index < len(values); index++ {
    if index > 0 {
      buffer_write_byte(buffer, ',')
    }
    buffer_write_string(buffer, values[index])
  }
  buffer_write_byte(buffer, ']')
}

func buffer_write_integers(buffer *Buffer, values []int) {
  buffer_write_byte(buffer, '[')
  for index := 0; index < len(values); index++ {
    if index > 0 {
      buffer_write_byte(buffer, ',')
    }
    buffer_write_int64(buffer, int64(values[index]))
  }
  buffer_write_byte(buffer, ']')
}

func buffer_write_floats(buffer *Buffer, values []float64) {
  buffer_write_byte(buffer, '[')
  for index := 0; index < len(values); index++ {
    if index > 0 {
      buffer_write_byte(buffer, ',')
    }
    buffer_write_float(buffer, values[index], FLOAT_BITS_64)
  }
  buffer_write_byte(buffer, ']')
}

func buffer_write_booleans(buffer *Buffer, values []bool) {
  buffer_write_byte(buffer, '[')
  for index := 0; index < len(values); index++ {
    if index > 0 {
      buffer_write_byte(buffer, ',')
    }
    buffer_write_boolean(buffer, values[index])
  }
  buffer_write_byte(buffer, ']')
}

func buffer_write_durations(buffer *Buffer, values []time.Duration, unit time.Duration) {
  buffer_write_byte(buffer, '[')
  for index := 0; index < len(values); index++ {
    if index > 0 {
      buffer_write_byte(buffer, ',')
    }
    buffer_write_int64(buffer, int64(values[index])/int64(unit))
  }
  buffer_write_byte(buffer, ']')
}

func buffer_write_begin_marker(buffer *Buffer) {
  buffer_write_byte(buffer, '{')
}

func buffer_write_end_marker(buffer *Buffer) {
  buffer_write_byte(buffer, '}')
}

func buffer_write_key(buffer *Buffer, key Key) {
  if buffer.Count > 0 {
    if buffer.Bytes[buffer.Count-1] != '{' {
      buffer_write_byte(buffer, ',')
    }
  }
  buffer_write_string(buffer, string(key))
  buffer_write_byte(buffer, ':')
}

// The fragment's leading brace is dropped and a separating comma added so a
// sub-logger's pre-encoded fields splice into the open object.
func buffer_write_object_data(buffer *Buffer, prefix []byte) {
  body := prefix
  if len(body) > 0 {
    if body[0] == '{' {
      body = body[1:]
    }
  }
  if buffer.Count > 1 {
    buffer_write_byte(buffer, ',')
  }
  buffer_write_bytes(buffer, body)
}

func buffer_write_int64(buffer *Buffer, value int64) {
  var storage [strconv.INTEGER_TEXT_SIZE_MAXIMUM]byte
  count := strconv.Format_Integer_Into(
    storage[:],
    strconv.Signed_Integer(value),
    strconv.DECIMAL_BASE,
  )
  buffer_write_bytes(buffer, storage[:int(count)])
}

func buffer_write_uint64(buffer *Buffer, value uint64) {
  var storage [strconv.DIGIT_TEXT_SIZE_MAXIMUM]byte
  count := strconv.Format_Unsigned_Integer_Into(
    storage[:],
    strconv.Unsigned_Integer(value),
    strconv.DECIMAL_BASE,
  )
  buffer_write_bytes(buffer, storage[:int(count)])
}

func buffer_write_boolean(buffer *Buffer, value bool) {
  if value {
    buffer_write_text(buffer, "true")
    return
  }
  buffer_write_text(buffer, "false")
}

// Timestamps render as an RFC 3339 UTC string rather than a raw nanosecond integer:
// epoch-nanosecond integers exceed JSON's safe 2^53 range and get mangled by
// double-precision consumers (jq, JavaScript). A fractional part is emitted only
// when the sub-second nanoseconds are nonzero. See the package doc's referenced
// article ("Don't Use Very Large Numbers").
func buffer_write_rfc3339(buffer *Buffer, nanos int64) {
  const NS_PER_SECOND = 1_000_000_000
  const SECONDS_PER_DAY = 86400
  const SECONDS_PER_HOUR = 3600
  const SECONDS_PER_MINUTE = 60
  seconds := nanos / NS_PER_SECOND
  fraction := nanos % NS_PER_SECOND
  if fraction < 0 {
    fraction += NS_PER_SECOND
    seconds--
  }
  days := seconds / SECONDS_PER_DAY
  day_seconds := seconds % SECONDS_PER_DAY
  if day_seconds < 0 {
    day_seconds += SECONDS_PER_DAY
    days--
  }
  aver.Always(uint64(day_seconds) < uint64(SECONDS_PER_DAY), "jlog: bad day seconds")
  year, month, day := civil_from_days(days)
  aver.Always(uint(month)-1 < 12, "jlog: civil month out of range")
  aver.Always(uint(day)-1 < 31, "jlog: civil day out of range")
  within_hour := day_seconds % SECONDS_PER_HOUR
  buffer_write_byte(buffer, '"')
  buffer_write_four_digits(buffer, year)
  buffer_write_byte(buffer, '-')
  buffer_write_two_digits(buffer, int64(month))
  buffer_write_byte(buffer, '-')
  buffer_write_two_digits(buffer, int64(day))
  buffer_write_byte(buffer, 'T')
  buffer_write_two_digits(buffer, day_seconds/SECONDS_PER_HOUR)
  buffer_write_byte(buffer, ':')
  buffer_write_two_digits(buffer, within_hour/SECONDS_PER_MINUTE)
  buffer_write_byte(buffer, ':')
  buffer_write_two_digits(buffer, within_hour%SECONDS_PER_MINUTE)
  if fraction != 0 {
    buffer_write_byte(buffer, '.')
    buffer_write_nanoseconds(buffer, fraction)
  }
  buffer_write_text(buffer, "Z\"")
}

// Two zero-padded decimal digits (00-99).
func buffer_write_two_digits(buffer *Buffer, value int64) {
  buffer_write_byte(buffer, byte('0'+value/10))
  buffer_write_byte(buffer, byte('0'+value%10))
}

// Four zero-padded decimal digits (0000-9999). A year past 9999 drops its top
// digit, which no realistic log timestamp reaches.
func buffer_write_four_digits(buffer *Buffer, value int64) {
  buffer_write_byte(buffer, byte('0'+value/1000%10))
  buffer_write_byte(buffer, byte('0'+value/100%10))
  buffer_write_byte(buffer, byte('0'+value/10%10))
  buffer_write_byte(buffer, byte('0'+value%10))
}

// Nine zero-padded decimal digits, most significant first, for the nanosecond
// fraction of a second.
func buffer_write_nanoseconds(buffer *Buffer, value int64) {
  divisor := int64(100_000_000)
  for divisor > 0 {
    buffer_write_byte(buffer, byte('0'+value/divisor%10))
    divisor /= 10
  }
}

// Howard Hinnant's days-from-civil inverse: maps days since 1970-01-01 to the
// proleptic Gregorian calendar date, exact for any int64 day count.
func civil_from_days(days int64) (year int64, month int, day int) {
  const DAYS_SHIFT = 719468
  const DAYS_PER_ERA = 146097
  z := days + DAYS_SHIFT
  era := z
  if era < 0 {
    era -= DAYS_PER_ERA - 1
  }
  era /= DAYS_PER_ERA
  day_of_era := z - era*DAYS_PER_ERA
  year_of_era := (day_of_era - day_of_era/1460 + day_of_era/36524 - day_of_era/146096) / 365
  computed_year := year_of_era + era*400
  day_of_year := day_of_era - (365*year_of_era + year_of_era/4 - year_of_era/100)
  month_portion := (5*day_of_year + 2) / 153
  computed_day := day_of_year - (153*month_portion+2)/5 + 1
  computed_month := month_portion + 3
  if !(month_portion < 10) {
    computed_month = month_portion - 9
  }
  if !(computed_month > 2) {
    computed_year++
  }
  return computed_year, int(computed_month), int(computed_day)
}

func buffer_write_level(buffer *Buffer, level Level, configuration *Logger_Configuration) {
  if level == LEVEL_NONE {
    return
  }
  if configuration.Level_Field_Name == "" {
    return
  }
  buffer_write_key(buffer, Key(configuration.Level_Field_Name))
  buffer_write_string(buffer, Level_String(level))
}

// JSON has no NaN/Inf, so those render as strings; a leading exponent zero is
// trimmed to match es6 number output.
func buffer_write_float(buffer *Buffer, value float64, bit_size int) {
  bits := float64_bits(value)
  if float64_is_nan(bits) {
    buffer_write_text(buffer, "\"NaN\"")
    return
  }
  if bits == FLOAT64_POSITIVE_INFINITY_BITS {
    buffer_write_text(buffer, "\"+Inf\"")
    return
  }
  if bits == FLOAT64_NEGATIVE_INFINITY_BITS {
    buffer_write_text(buffer, "\"-Inf\"")
    return
  }
  format := byte('f')
  if float_needs_exponent(value, bit_size) {
    format = 'e'
  }
  buffer_write_finite_float(buffer, value, bit_size, format)
  if format == 'e' {
    buffer_clean_exponent(buffer)
  }
}

func buffer_write_finite_float(buffer *Buffer, value float64, bit_size int, format byte) {
  information := Float_Information{
    Mantissa_Bit_Count: FLOAT64_MANTISSA_BIT_COUNT,
    Exponent_Bit_Count: FLOAT64_EXPONENT_BIT_COUNT,
    Exponent_Bias:      FLOAT64_EXPONENT_BIAS,
  }
  bits := float64_bits(value)
  if bit_size == FLOAT_BITS_32 {
    information = Float_Information{
      Mantissa_Bit_Count: FLOAT32_MANTISSA_BIT_COUNT,
      Exponent_Bit_Count: FLOAT32_EXPONENT_BIT_COUNT,
      Exponent_Bias:      FLOAT32_EXPONENT_BIAS,
    }
    bits = uint64(float32_bits(float32(value)))
  }
  negative := bits>>(information.Exponent_Bit_Count+information.Mantissa_Bit_Count) != 0
  exponent_mask := uint64(1)<<information.Exponent_Bit_Count - 1
  exponent := int(bits>>information.Mantissa_Bit_Count) & int(exponent_mask)
  mantissa := bits & (uint64(1)<<information.Mantissa_Bit_Count - 1)
  if exponent == 0 {
    exponent++
  } else {
    mantissa |= uint64(1) << information.Mantissa_Bit_Count
  }
  exponent += information.Exponent_Bias
  var digits [FLOAT_DECIMAL_DIGIT_CAPACITY]byte
  decimal := Decimal{Digits: digits[:]}
  decimal_assign(&decimal, mantissa)
  decimal_shift(&decimal, exponent-int(information.Mantissa_Bit_Count))
  decimal_round_shortest(&decimal, mantissa, exponent, information)
  precision := decimal.Count - decimal.Point
  if precision < 0 {
    precision = 0
  }
  if format == 'e' {
    precision = decimal.Count - 1
    if precision < 0 {
      precision = 0
    }
    buffer_write_float_exponent(buffer, negative, &decimal, precision)
    return
  }
  buffer_write_float_fixed(buffer, negative, &decimal, precision)
}

func buffer_write_float_exponent(buffer *Buffer, negative bool, decimal *Decimal, precision int) {
  if negative {
    buffer_write_byte(buffer, '-')
  }
  first := byte('0')
  if decimal.Count != 0 {
    first = decimal.Digits[0]
  }
  buffer_write_byte(buffer, first)
  if precision > 0 {
    buffer_write_byte(buffer, '.')
    index := 1
    end_index := decimal.Count
    if end_index > precision+1 {
      end_index = precision + 1
    }
    if index < end_index {
      buffer_write_bytes(buffer, decimal.Digits[index:end_index])
      index = end_index
    }
    for !(index > precision) {
      buffer_write_byte(buffer, '0')
      index++
    }
  }
  buffer_write_byte(buffer, 'e')
  exponent := decimal.Point - 1
  if decimal.Count == 0 {
    exponent = 0
  }
  if exponent < 0 {
    buffer_write_byte(buffer, '-')
    exponent = -exponent
  } else {
    buffer_write_byte(buffer, '+')
  }
  if !(exponent < 100) {
    buffer_write_byte(buffer, byte(exponent/100)+'0')
  }
  buffer_write_byte(buffer, byte(exponent/10%10)+'0')
  buffer_write_byte(buffer, byte(exponent%10)+'0')
}

func buffer_write_float_fixed(buffer *Buffer, negative bool, decimal *Decimal, precision int) {
  if negative {
    buffer_write_byte(buffer, '-')
  }
  if decimal.Point > 0 {
    end_index := decimal.Count
    if end_index > decimal.Point {
      end_index = decimal.Point
    }
    buffer_write_bytes(buffer, decimal.Digits[:end_index])
    for index := end_index; index < decimal.Point; index++ {
      buffer_write_byte(buffer, '0')
    }
  } else {
    buffer_write_byte(buffer, '0')
  }
  if !(precision > 0) {
    return
  }
  buffer_write_byte(buffer, '.')
  for index := 0; index < precision; index++ {
    digit := byte('0')
    digit_index := decimal.Point + index
    if !(digit_index < 0) {
      if digit_index < decimal.Count {
        digit = decimal.Digits[digit_index]
      }
    }
    buffer_write_byte(buffer, digit)
  }
}

func float_needs_exponent(value float64, bit_size int) (needs bool) {
  magnitude := float64_from_bits(float64_bits(value) & FLOAT64_MAGNITUDE_MASK)
  if magnitude == 0 {
    return false
  }
  if bit_size == FLOAT_BITS_64 {
    if magnitude < FLOAT_EXPONENT_LOW {
      return true
    }
    if magnitude > FLOAT_EXPONENT_HIGH {
      return true
    }
    return magnitude == FLOAT_EXPONENT_HIGH
  }
  if float32(magnitude) < FLOAT_EXPONENT_LOW {
    return true
  }
  if float32(magnitude) > FLOAT_EXPONENT_HIGH {
    return true
  }
  return float32(magnitude) == FLOAT_EXPONENT_HIGH
}

func buffer_clean_exponent(buffer *Buffer) {
  count := buffer.Count
  if count < 4 {
    return
  }
  if buffer.Bytes[count-4] != 'e' {
    return
  }
  if buffer.Bytes[count-3] != '-' {
    return
  }
  if buffer.Bytes[count-2] != '0' {
    return
  }
  buffer.Bytes[count-2] = buffer.Bytes[count-1]
  buffer.Count = count - 1
}

func buffer_write_string(buffer *Buffer, value string) {
  buffer_write_byte(buffer, '"')
  buffer_write_string_body(buffer, value)
  buffer_write_byte(buffer, '"')
}

// The all-plain common case is copied in one shot; the escape path runs only
// once a byte needs it.
func buffer_write_string_body(buffer *Buffer, value string) {
  for index := 0; index < len(value); index++ {
    if byte_is_plain(value[index]) {
      continue
    }
    buffer_write_string_complex(buffer, value, index)
    return
  }
  buffer_write_text(buffer, value)
}

func buffer_write_string_complex(buffer *Buffer, value string, scan int) {
  run_start := 0
  index := scan
  for index < len(value) {
    current := value[index]
    if !(current < byte(utf8.CHARACTER_SELF)) {
      decoded, width := utf8.Decode_Character_Text(utf8.Text(value[index:]))
      if decoded == utf8.REPLACEMENT_CHARACTER {
        if width == utf8.CHARACTER_SIZE_MINIMUM {
          buffer_write_text(buffer, value[run_start:index])
          buffer_write_replacement(buffer)
          index += int(width)
          run_start = index
          continue
        }
      }
      index += int(width)
      continue
    }
    if byte_is_plain(current) {
      index++
      continue
    }
    buffer_write_text(buffer, value[run_start:index])
    buffer_write_escaped_byte(buffer, current)
    index++
    run_start = index
  }
  buffer_write_text(buffer, value[run_start:])
}

func buffer_write_hexadecimal(buffer *Buffer, value []byte) {
  buffer_write_byte(buffer, '"')
  for index := 0; index < len(value); index++ {
    current := value[index]
    buffer_write_byte(buffer, hexadecimal_digit(current>>4))
    buffer_write_byte(buffer, hexadecimal_digit(current&0x0f))
  }
  buffer_write_byte(buffer, '"')
}

func buffer_write_quoted_bytes(buffer *Buffer, value []byte) {
  buffer_write_byte(buffer, '"')
  buffer_write_bytes_body(buffer, value)
  buffer_write_byte(buffer, '"')
}

func buffer_write_bytes_body(buffer *Buffer, value []byte) {
  for index := 0; index < len(value); index++ {
    if byte_is_plain(value[index]) {
      continue
    }
    buffer_write_bytes_complex(buffer, value, index)
    return
  }
  buffer_write_bytes(buffer, value)
}

func buffer_write_bytes_complex(buffer *Buffer, value []byte, scan int) {
  run_start := 0
  index := scan
  for index < len(value) {
    current := value[index]
    if !(current < byte(utf8.CHARACTER_SELF)) {
      decoded, width := utf8.Decode_Character(utf8.Bytes(value[index:]))
      if decoded == utf8.REPLACEMENT_CHARACTER {
        if width == utf8.CHARACTER_SIZE_MINIMUM {
          buffer_write_bytes(buffer, value[run_start:index])
          buffer_write_replacement(buffer)
          index += int(width)
          run_start = index
          continue
        }
      }
      index += int(width)
      continue
    }
    if byte_is_plain(current) {
      index++
      continue
    }
    buffer_write_bytes(buffer, value[run_start:index])
    buffer_write_escaped_byte(buffer, current)
    index++
    run_start = index
  }
  buffer_write_bytes(buffer, value[run_start:])
}

func buffer_write_escaped_byte(buffer *Buffer, value byte) {
  switch value {
  case '"', '\\':
    buffer_write_byte(buffer, '\\')
    buffer_write_byte(buffer, value)
  case '\b':
    buffer_write_text(buffer, "\\b")
  case '\f':
    buffer_write_text(buffer, "\\f")
  case '\n':
    buffer_write_text(buffer, "\\n")
  case '\r':
    buffer_write_text(buffer, "\\r")
  case '\t':
    buffer_write_text(buffer, "\\t")
  default:
    buffer_write_text(buffer, "\\u00")
    buffer_write_byte(buffer, hexadecimal_digit(value>>4))
    buffer_write_byte(buffer, hexadecimal_digit(value&0x0f))
  }
}

// Writes the JSON escape for the U+FFFD replacement character. zerolog emits this
// six-byte escape (not the raw rune) for a malformed UTF-8 byte; jlog matches it so
// the wire bytes are identical on bad input.
func buffer_write_replacement(buffer *Buffer) {
  buffer_write_text(buffer, "\\ufffd")
}

// A plain byte is printable ASCII that is neither a quote nor a backslash and so
// needs no escaping in a JSON string. The 256-bit table (one bit per byte value,
// LSB first within each byte) is a 32-byte const string so the test is a branchless
// lookup with no package-level var and no init: bits 0x20..0x7e are set except the
// quote (0x22) and backslash (0x5c).
func byte_is_plain(value byte) (plain bool) {
  const TABLE = "\x00\x00\x00\x00\xfb\xff\xff\xff\xff\xff\xff\xef\xff\xff\xff\x7f" +
    "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"
  return TABLE[value>>3]&(byte(1)<<(value&7)) != 0
}

func hexadecimal_digit(nibble byte) (digit byte) {
  const HEXADECIMAL_DIGITS = "0123456789abcdef"
  return HEXADECIMAL_DIGITS[nibble]
}

// The rest of this file is the pretty printer: the human-facing inverse of the encoder above. jlog
// emits flat JSON for machines; a Console reads those lines back and renders them for a human at a
// terminal. It plugs into the encoder only through the Write procedure: each emit sends one whole
// flat-JSON object. It is pure — it reaches for no OS, clock, or global state; every dependency
// arrives through a Console field — so it belongs here, not in the composition tier. It works in
// its own fixed storage and allocates nothing.

// An ANSI SGR sequence. A distinct type so buffer_paint_begin takes it without colliding with its
// plain-text argument under the same-type-parameter rule, mirroring maddox's ansi_code.
type Ansi_Code string

// ANSI_RESET closes every painted span so a color never bleeds past the part it marks.
const ANSI_RESET Ansi_Code = "\x1b[0m"

// ANSI_FAINT dims the timestamp and trace lines so they recede behind the message.
const ANSI_FAINT Ansi_Code = "\x1b[2m"

// ANSI_BOLD lifts the message, the part a human scans for first.
const ANSI_BOLD Ansi_Code = "\x1b[1m"

// ANSI_RED marks an error value and the error level, so a failure stands out.
const ANSI_RED Ansi_Code = "\x1b[31m"

// ANSI_GREEN tags the info level.
const ANSI_GREEN Ansi_Code = "\x1b[32m"

// ANSI_YELLOW tags the warn level.
const ANSI_YELLOW Ansi_Code = "\x1b[33m"

// ANSI_CYAN labels logfmt keys and the debug level, legible where a dim gray would not be.
const ANSI_CYAN Ansi_Code = "\x1b[36m"

// Console renders each flat-JSON jlog line as a human-readable console line.
// It reads jlog's default field names (time, level, message, error); a logger built with custom
// field names is not matched, which no caller needs. It is used by pointer: one Write is in
// flight at a time over its caller-owned Rendered storage. Build one directly with a struct
// literal, filling Rendered and Object.
type Console struct {
  // Stream receives the rendered text.
  Stream nbio.Stream
  // Color enables ANSI color in the rendered output.
  Color bool
  // Forward holds the one in-flight write toward Stream.
  Forward Forward
  // Object is the scan of the line being rendered; the caller fills its storage.
  Object Object
  // Rendered is the caller-owned render storage for one Write, at most
  // TRANSFORM_SIZE_MAXIMUM bytes; a Write whose render outgrows it retires
  // Stream_Buffer_Full.
  Rendered []byte
}

// TRANSFORM_MODES excludes reads and cursor modes because a transformed byte count names no
// position in the inner Stream.
const TRANSFORM_MODES nbio.Stream_Mode_Set = 1<<nbio.STREAM_MODE_CLOSE |
  1<<nbio.STREAM_MODE_FLUSH |
  1<<nbio.STREAM_MODE_WRITE |
  1<<nbio.STREAM_MODE_DESTROY |
  1<<nbio.STREAM_MODE_QUERY

// Console_To_Stream renders each written batch of newline-terminated JSON lines and submits the
// human-readable form to the inner Stream once.
func Console_To_Stream(console *Console) (stream nbio.Stream) {
  aver.Always(console != nil, "jlog: console must not be nil")
  aver.Always(len(console.Rendered) > 0, "jlog: console render storage is required")
  aver.Always(!(len(console.Rendered) > TRANSFORM_SIZE_MAXIMUM), "jlog: render exceeds bound")
  object_check(&console.Object)
  return nbio.Stream{State: console, Procedure: console_procedure}
}

func object_check(object *Object) {
  aver.Always(len(object.Members) > 0, "jlog: object member storage is required")
  aver.Always(!(len(object.Members) > MEMBER_COUNT_MAXIMUM), "jlog: members exceed bound")
  aver.Always(len(object.Decoded) > 0, "jlog: object decoded storage is required")
  aver.Always(!(len(object.Decoded) > TRANSFORM_SIZE_MAXIMUM), "jlog: decoded exceeds bound")
}

func console_procedure(
  state nbio.State,
  completion *nbio.Completion,
  mode nbio.Stream_Mode,
  payload []byte,
  offset int64,
  whence nbio.Seek_From,
  callback nbio.Stream_Callback,
) {
  console := state.(*Console)
  if mode == nbio.STREAM_MODE_WRITE {
    if transform_busy(&console.Forward, completion, callback) {
      return
    }
    rendered := Buffer{Bytes: console.Rendered}
    console_render(console, &rendered, payload)
    transform_write(
      console.Stream, &console.Forward, completion, len(payload), &rendered,
      callback,
    )
    return
  }
  transform_submit(console.Stream, completion, mode, offset, whence, callback)
}

// Forward keeps the outer retirement alive until the inner Stream retires, since a kernel-backed
// Stream may retain only the Completion. It lives in the transform's state, so one write is in
// flight per transform.
type Forward struct {
  // Callback is the outer retirement.
  Callback nbio.Stream_Callback
  // Count is the outer payload length reported on success.
  Count int
  // Busy marks the inner write as still in flight; a second submission is a caller defect.
  Busy bool
}

// A transform owns one storage run, so a Write arriving while an earlier one is still in flight,
// as a parent and a child logger sharing one Console produce, would overwrite bytes the inner
// Stream borrows. The new Write retires Stream_Buffer_Full before touching storage: the same drop
// a logger makes when its own line is still borrowed.
func transform_busy(
  forward *Forward,
  completion *nbio.Completion,
  callback nbio.Stream_Callback,
) (busy bool) {
  if !forward.Busy {
    return false
  }
  completion.Data = 0
  completion.Error = nbio.Stream_Buffer_Full
  nbio.Stream_Callback_Call(callback, completion)
  return true
}

// A transforming Stream reports the payload count consumed on success because it emits a
// different byte count than it takes. A transform that outgrew its storage retires as
// Stream_Buffer_Full and reaches the inner Stream with nothing, never with a torn tail.
func transform_write(
  inner nbio.Stream,
  forward *Forward,
  completion *nbio.Completion,
  payload_count int,
  transformed *Buffer,
  callback nbio.Stream_Callback,
) {
  completion.Data = 0
  completion.Error = nil
  if transformed.Overflow {
    completion.Error = nbio.Stream_Buffer_Full
    nbio.Stream_Callback_Call(callback, completion)
    return
  }
  if transformed.Count == 0 {
    completion.Data = payload_count
    nbio.Stream_Callback_Call(callback, completion)
    return
  }
  if inner.Procedure == nil {
    completion.Error = nbio.Stream_Empty
    nbio.Stream_Callback_Call(callback, completion)
    return
  }
  aver.Always(!forward.Busy, "jlog: a transform has one write in flight")
  forward.Busy = true
  forward.Callback = callback
  forward.Count = payload_count
  written := transformed.Bytes[:transformed.Count]
  retire := nbio.Stream_Callback{
    State: forward, Data: len(written), Procedure: transform_retire,
  }
  inner.Procedure(
    inner.State,
    completion,
    nbio.STREAM_MODE_WRITE,
    written,
    0,
    nbio.SEEK_FROM_START,
    retire,
  )
}

func transform_retire(
  state nbio.State,
  transformed_count nbio.Stream_Callback_Data,
  _ nbio.Callback,
  completion nbio.Completion_Handle,
) {
  forward := state.(*Forward)
  forward.Busy = false
  if completion.Error == nil {
    if completion.Data != int(transformed_count) {
      completion.Error = nbio.Stream_Short_Write
    }
  }
  completion.Data = 0
  if completion.Error == nil {
    completion.Data = forward.Count
  }
  nbio.Stream_Callback_Call(forward.Callback, completion)
}

// Lifecycle modes pass through untouched so teardown reaches the inner Stream.
func transform_submit(
  inner nbio.Stream,
  completion *nbio.Completion,
  mode nbio.Stream_Mode,
  offset int64,
  whence nbio.Seek_From,
  callback nbio.Stream_Callback,
) {
  completion.Data = 0
  completion.Error = nil
  switch mode {
  case nbio.STREAM_MODE_QUERY:
    completion.Data = int(TRANSFORM_MODES)
  case nbio.STREAM_MODE_FLUSH, nbio.STREAM_MODE_CLOSE, nbio.STREAM_MODE_DESTROY:
    if inner.Procedure != nil {
      inner.Procedure(
        inner.State, completion, mode, nil, offset, whence, callback,
      )
      return
    }
    completion.Error = nbio.Stream_Empty
  default:
    completion.Error = nbio.Stream_Empty
  }
  nbio.Stream_Callback_Call(callback, completion)
}

func console_render(console *Console, rendered *Buffer, payload []byte) {
  line_start := 0
  for index := 0; index < len(payload); index++ {
    if payload[index] != '\n' {
      continue
    }
    buffer_write_pretty_line(
      rendered, &console.Object, payload[line_start:index], console.Color,
    )
    buffer_write_byte(rendered, '\n')
    line_start = index + 1
  }
  // A trailing chunk with no newline is not something jlog produces (every line ends in \n),
  // but a Console used on a raw pipe might see one; render it without inventing a newline.
  if line_start < len(payload) {
    buffer_write_pretty_line(
      rendered, &console.Object, payload[line_start:], console.Color,
    )
  }
}

// How a JSON value is displayed: a string is unquoted (and logfmt-requoted only if needed), a
// literal (number/bool/null) is copied verbatim so a large integer keeps its exact digits, and a
// compound (array/object) is copied as compacted JSON so it stays one flat token.
type Value_Kind uint8

// VALUE_IS_STRING tags a value rendered unquoted, logfmt-requoted only when needed.
const VALUE_IS_STRING Value_Kind = 0

// VALUE_IS_LITERAL tags a number/bool/null copied verbatim so its exact digits survive.
const VALUE_IS_LITERAL Value_Kind = 1

// VALUE_IS_COMPOUND tags an array/object copied as one compact JSON token.
const VALUE_IS_COMPOUND Value_Kind = 2

// Span locates one run of bytes inside an Object's Decoded storage.
type Span struct {
  // Offset is the index of the first byte.
  Offset int
  // Size is the byte count.
  Size int
}

// One key/value member of a scanned line, holding the value already reduced to its display text
// and kind so rendering never re-parses.
type Member struct {
  // Key locates the JSON object key in Decoded.
  Key Span
  // Text locates the value's display form, already unescaped or compacted, in Decoded.
  Text Span
  // Kind selects how Text is quoted when rendered.
  Kind Value_Kind
}

// Object is the scan of one flat-JSON line in caller-owned storage: the members in wire order
// and the decoded text they point into. A line past either bound does not parse and passes
// through unrendered.
type Object struct {
  // Members holds the scanned members in wire order, at most MEMBER_COUNT_MAXIMUM.
  Members []Member
  // Count is the members scanned.
  Count int
  // Decoded holds every key and value text, at most TRANSFORM_SIZE_MAXIMUM bytes.
  Decoded []byte
}

func object_span(object *Object, span Span) (text []byte) {
  return object.Decoded[span.Offset : span.Offset+span.Size]
}

// Writes line rendered as one console line (no trailing newline; Write adds that). A line that is
// not a JSON object is written verbatim, so interleaved non-JSON output is never lost.
func buffer_write_pretty_line(buffer *Buffer, object *Object, line []byte, color bool) {
  if !scan_object(object, line) {
    buffer_write_bytes(buffer, line)
    return
  }
  buffer_write_members(buffer, object, color)
}

// Scans one flat-JSON object into object in wire order. It is hand-rolled rather than built on
// json.NewDecoder because the house style bans that unbounded streaming API, and json.Unmarshal
// into a map would scramble jlog's deliberate field order and round a large uint64 through float64.
// Each value's exact source bytes are captured (numbers stay exact, an embedded Raw_JSON blob
// survives). Any deviation from a lone, well-formed object, or a line past the member or text
// bound, returns parsed=false, and the caller passes the line through untouched.
func scan_object(object *Object, line []byte) (parsed bool) {
  object.Count = 0
  decoded := Buffer{Bytes: object.Decoded}
  index := scan_space(line, 0)
  if !(index < len(line)) {
    return false
  }
  if line[index] != '{' {
    return false
  }
  index = scan_space(line, index+1)
  if index < len(line) {
    if line[index] == '}' {
      return scan_space(line, index+1) == len(line)
    }
  }
  // Each member is at least one byte, so len(line) bounds the count and keeps the loop
  // bounded; a well-formed object returns through '}' before that ceiling is reached.
  for guard_index := 0; guard_index < len(line); guard_index++ {
    if object.Count == len(object.Members) {
      return false
    }
    key, key_next, key_ok := scan_string(&decoded, line, index)
    if !key_ok {
      return false
    }
    index = scan_space(line, key_next)
    if !(index < len(line)) {
      return false
    }
    if line[index] != ':' {
      return false
    }
    index = scan_space(line, index+1)
    text, kind, value_next, value_ok := scan_value(&decoded, line, index)
    if !value_ok {
      return false
    }
    object.Members[object.Count] = Member{Key: key, Text: text, Kind: kind}
    object.Count++
    index = scan_space(line, value_next)
    if !(index < len(line)) {
      return false
    }
    if line[index] == ',' {
      index = scan_space(line, index+1)
      continue
    }
    if line[index] == '}' {
      return scan_space(line, index+1) == len(line)
    }
    return false
  }
  return false
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

// Reads a JSON string beginning at start, decoding its escapes into decoded and returning the
// index past the closing quote. The scan only has to find the real close quote, which means
// skipping a backslash-escaped byte so an escaped quote does not end it early.
func scan_string(decoded *Buffer, line []byte, start int) (text Span, next int, ok bool) {
  if !(start < len(line)) {
    return Span{}, start, false
  }
  if line[start] != '"' {
    return Span{}, start, false
  }
  index := start + 1
  for index < len(line) {
    if line[index] == '\\' {
      index += 2
      continue
    }
    if line[index] == '"' {
      text.Offset = decoded.Count
      valid := json_string_decode(decoded, line[start+1:index])
      text.Size = decoded.Count - text.Offset
      return text, index + 1, valid
    }
    index++
  }
  return Span{}, start, false
}

func json_string_decode(decoded *Buffer, encoded []byte) (valid bool) {
  for index := 0; index < len(encoded); index++ {
    current := encoded[index]
    if current < 0x20 {
      return false
    }
    if current != '\\' {
      buffer_write_byte(decoded, current)
      continue
    }
    index++
    if !(index < len(encoded)) {
      return false
    }
    switch encoded[index] {
    case '"', '\\', '/':
      buffer_write_byte(decoded, encoded[index])
    case 'b':
      buffer_write_byte(decoded, '\b')
    case 'f':
      buffer_write_byte(decoded, '\f')
    case 'n':
      buffer_write_byte(decoded, '\n')
    case 'r':
      buffer_write_byte(decoded, '\r')
    case 't':
      buffer_write_byte(decoded, '\t')
    case 'u':
      code, tail_index, known := json_hexadecimal_character(encoded, index+1)
      if !known {
        return false
      }
      index = tail_index
      buffer_write_character(decoded, code)
    default:
      return false
    }
  }
  return !decoded.Overflow
}

func json_hexadecimal_character(
  encoded []byte,
  start int,
) (character rune, tail_index int, known bool) {
  if start+4 > len(encoded) {
    return 0, start, false
  }
  for index := start; index < start+4; index++ {
    digit, found := hexadecimal_value(encoded[index])
    if !found {
      return 0, start, false
    }
    character = character<<4 | rune(digit)
  }
  return character, start + 3, true
}

func hexadecimal_value(value byte) (digit byte, known bool) {
  if !(value < '0') {
    if !(value > '9') {
      return value - '0', true
    }
  }
  if !(value < 'a') {
    if !(value > 'f') {
      return value - 'a' + 10, true
    }
  }
  if !(value < 'A') {
    if !(value > 'F') {
      return value - 'A' + 10, true
    }
  }
  return 0, false
}

func buffer_write_character(buffer *Buffer, character rune) {
  if character < 0x80 {
    buffer_write_byte(buffer, byte(character))
    return
  }
  if character < 0x800 {
    buffer_write_byte(buffer, byte(0xc0|character>>6))
    buffer_write_byte(buffer, byte(0x80|character&0x3f))
    return
  }
  buffer_write_byte(buffer, byte(0xe0|character>>12))
  buffer_write_byte(buffer, byte(0x80|character>>6&0x3f))
  buffer_write_byte(buffer, byte(0x80|character&0x3f))
}

// Reads one JSON value beginning at start, classifying it: a string is unescaped, an array or
// object is captured as its verbatim (already-compact) source bytes, and anything else is a scalar
// literal whose digits are kept exactly. Every form lands in decoded so a member points one way.
func scan_value(
  decoded *Buffer,
  line []byte,
  start int,
) (text Span, kind Value_Kind, next int, ok bool) {
  if !(start < len(line)) {
    return Span{}, VALUE_IS_LITERAL, start, false
  }
  if line[start] == '"' {
    text, next, ok = scan_string(decoded, line, start)
    return text, VALUE_IS_STRING, next, ok
  }
  kind = VALUE_IS_LITERAL
  if line[start] == '{' {
    kind = VALUE_IS_COMPOUND
    next, ok = scan_compound(line, start)
  } else if line[start] == '[' {
    kind = VALUE_IS_COMPOUND
    next, ok = scan_compound(line, start)
  } else {
    next, ok = scan_literal(line, start)
  }
  if !ok {
    return Span{}, kind, start, false
  }
  text.Offset = decoded.Count
  buffer_write_bytes(decoded, line[start:next])
  text.Size = decoded.Count - text.Offset
  return text, kind, next, !decoded.Overflow
}

// Reads a brace- or bracket-delimited value beginning at start, returning the index past its
// close. A depth counter over the matching delimiter finds the close; strings are skipped whole so
// a delimiter inside a string does not miscount. Cross-type nesting (an array inside an object) is
// transparent — the other delimiter is an ordinary byte — and any real imbalance surfaces as a
// parse failure at the enclosing object, which passes the line through.
func scan_compound(line []byte, start int) (next int, ok bool) {
  opener := line[start]
  closer := byte('}')
  if opener == '[' {
    closer = ']'
  }
  depth := 0
  index := start
  for index < len(line) {
    if line[index] == '"' {
      string_next, string_ok := scan_string_end(line, index)
      if !string_ok {
        return start, false
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
        return index, true
      }
      continue
    }
    index++
  }
  return start, false
}

// Finds the index past the close quote of the string at start without decoding it, for the
// compound scan that copies its source bytes verbatim.
func scan_string_end(line []byte, start int) (next int, ok bool) {
  index := start + 1
  for index < len(line) {
    if line[index] == '\\' {
      index += 2
      continue
    }
    if line[index] == '"' {
      return index + 1, true
    }
    index++
  }
  return start, false
}

// Reads a scalar literal (number, true, false, or null) beginning at start, ending it at the first
// value terminator so its exact source digits are captured.
func scan_literal(line []byte, start int) (next int, ok bool) {
  index := start
  for index < len(line) {
    if byte_ends_literal(line[index]) {
      break
    }
    index++
  }
  if index == start {
    return start, false
  }
  return index, true
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
func buffer_write_members(buffer *Buffer, object *Object, color bool) {
  // An absent header key and an empty one render the same — nothing — so a plain non-empty
  // check covers both, and jlog never emits an empty timestamp, level, or message anyway. The
  // timestamp is coarsened to the second here; the tail keeps any Time field's precision.
  time_text := member_text(object, string(DEFAULT_TIMESTAMP_FIELD_NAME))
  level_text := member_text(object, string(DEFAULT_LEVEL_FIELD_NAME))
  message_text := member_text(object, string(DEFAULT_MESSAGE_FIELD_NAME))
  wrote := false
  if len(time_text) > 0 {
    buffer_write_separated(buffer, wrote)
    buffer_paint_begin(buffer, ANSI_FAINT, color)
    buffer_write_timestamp_seconds(buffer, time_text)
    buffer_paint_end(buffer, ANSI_FAINT, color)
    wrote = true
  }
  if len(level_text) > 0 {
    buffer_write_separated(buffer, wrote)
    buffer_write_level_tag(buffer, level_text, color)
    wrote = true
  }
  if len(message_text) > 0 {
    buffer_write_separated(buffer, wrote)
    buffer_paint_begin(buffer, ANSI_BOLD, color)
    buffer_write_bytes(buffer, message_text)
    buffer_paint_end(buffer, ANSI_BOLD, color)
    wrote = true
  }
  for index := 0; index < object.Count; index++ {
    member := &object.Members[index]
    if member_is_header(object_span(object, member.Key)) {
      continue
    }
    buffer_write_separated(buffer, wrote)
    buffer_write_field(buffer, object, member, color)
    wrote = true
  }
}

// Reports whether key is one of the three header keys, which render in the header rather than
// repeat in the logfmt tail.
func member_is_header(key []byte) (header bool) {
  if string(key) == string(DEFAULT_TIMESTAMP_FIELD_NAME) {
    return true
  }
  if string(key) == string(DEFAULT_LEVEL_FIELD_NAME) {
    return true
  }
  if string(key) == string(DEFAULT_MESSAGE_FIELD_NAME) {
    return true
  }
  return false
}

// Returns the display text of the first member with key, or nothing when no member has it.
func member_text(object *Object, key string) (text []byte) {
  for index := 0; index < object.Count; index++ {
    member := &object.Members[index]
    if string(object_span(object, member.Key)) == key {
      return object_span(object, member.Text)
    }
  }
  return nil
}

// Writes a single space before a part when a previous part was already written, so the line
// never opens with or doubles a separator.
func buffer_write_separated(buffer *Buffer, wrote bool) {
  if wrote {
    buffer_write_byte(buffer, ' ')
  }
}

// Writes one logfmt "key=value" pair: the key in cyan so it reads as a label yet stays legible
// (unlike a dim gray), the value logfmt-quoted, and — for the error field — the value painted red
// so a failure stands out.
func buffer_write_field(buffer *Buffer, object *Object, member *Member, color bool) {
  key := object_span(object, member.Key)
  text := object_span(object, member.Text)
  buffer_paint_begin(buffer, ANSI_CYAN, color)
  buffer_write_bytes(buffer, key)
  buffer_paint_end(buffer, ANSI_CYAN, color)
  buffer_write_byte(buffer, '=')
  value_color := Ansi_Code("")
  if string(key) == string(DEFAULT_ERROR_FIELD_NAME) {
    value_color = ANSI_RED
  }
  buffer_paint_begin(buffer, value_color, color)
  if member.Kind == VALUE_IS_STRING {
    if logfmt_needs_quote(text) {
      buffer_write_quoted_bytes(buffer, text)
      buffer_paint_end(buffer, value_color, color)
      return
    }
  }
  buffer_write_bytes(buffer, text)
  buffer_paint_end(buffer, value_color, color)
}

// Writes the level as its three-letter tag, painted by severity. Named apart from the encoder's
// buffer_write_level (which writes the level as a JSON field) since both share this package.
func buffer_write_level_tag(buffer *Buffer, wire []byte, color bool) {
  code := level_color(wire)
  buffer_paint_begin(buffer, code, color)
  buffer_write_level_label(buffer, wire)
  buffer_paint_end(buffer, code, color)
}

// Opens a painted span when color is on and code is a real color. An empty code means "no color
// for this part", so an unknown level or a non-error value stays uncolored even under color.
func buffer_paint_begin(buffer *Buffer, code Ansi_Code, color bool) {
  if !color {
    return
  }
  if code == "" {
    return
  }
  buffer_write_text(buffer, string(code))
}

// Closes the span buffer_paint_begin opened under the same code and color.
func buffer_paint_end(buffer *Buffer, code Ansi_Code, color bool) {
  if !color {
    return
  }
  if code == "" {
    return
  }
  buffer_write_text(buffer, string(ANSI_RESET))
}

// Reports whether a logfmt value must be double-quoted: an empty value, or one carrying a space,
// an equals, a quote, or a control byte, is ambiguous or unreadable bare.
func logfmt_needs_quote(value []byte) (needs bool) {
  if len(value) == 0 {
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

// Writes a level's three-letter tag from its wire name. An unknown level (a custom numeric level,
// or "disabled") falls back to its first three characters uppercased, so nothing renders blank.
func buffer_write_level_label(buffer *Buffer, wire []byte) {
  switch string(wire) {
  case "trace":
    buffer_write_text(buffer, "TRC")
    return
  case "debug":
    buffer_write_text(buffer, "DBG")
    return
  case "info":
    buffer_write_text(buffer, "INF")
    return
  case "warn":
    buffer_write_text(buffer, "WRN")
    return
  case "error":
    buffer_write_text(buffer, "ERR")
    return
  }
  count := len(wire)
  if count > LEVEL_LABEL_SIZE_MAXIMUM {
    count = LEVEL_LABEL_SIZE_MAXIMUM
  }
  for index := 0; index < count; index++ {
    current := wire[index]
    if !(current < 'a') {
      if !(current > 'z') {
        current -= 'a' - 'A'
      }
    }
    buffer_write_byte(buffer, current)
  }
}

// Maps a level's wire name to its color; an unknown level returns the empty code, meaning no
// color, so buffer_paint_begin leaves it plain.
func level_color(wire []byte) (code Ansi_Code) {
  switch string(wire) {
  case "trace":
    return ANSI_FAINT
  case "debug":
    return ANSI_CYAN
  case "info":
    return ANSI_GREEN
  case "warn":
    return ANSI_YELLOW
  case "error":
    return ANSI_RED
  }
  return ""
}

// Writes an RFC 3339 header timestamp trimmed to second precision by dropping its sub-second
// fraction and keeping the trailing zone, so "…:20.123456789Z" becomes "…:20Z" — a human reading
// the console needs no nanosecond granularity. A value with no fraction is written unchanged; only
// the console display coarsens, never the JSON line.
func buffer_write_timestamp_seconds(buffer *Buffer, value []byte) {
  fraction_offset := bytes_index(value, '.')
  if fraction_offset < 0 {
    buffer_write_bytes(buffer, value)
    return
  }
  buffer_write_bytes(buffer, value[:fraction_offset])
  // The zone is searched absolutely, not relative to the fraction, so the two offsets are
  // never added: an RFC 3339 UTC value carries exactly one 'Z', the terminal zone marker.
  zone_offset := bytes_index(value, 'Z')
  if zone_offset < 0 {
    return
  }
  buffer_write_bytes(buffer, value[zone_offset:])
}

func bytes_index(value []byte, sought byte) (index int) {
  for index = 0; index < len(value); index++ {
    if value[index] == sought {
      return index
    }
  }
  return -1
}

// Level_Filter forwards only the jlog lines whose level is at or above
// its Floor and drops the rest — a floor at the writer seam, not at the emit gate. A terminal
// logger emits every level so a file sink keeps all of them; a Level_Filter is what still holds
// trace and debug off the console. It is used by pointer: one Write is in flight at a time over
// its caller-owned Kept storage.
type Level_Filter struct {
  // Stream receives the lines that clear the floor.
  Stream nbio.Stream
  // Floor is the lowest level forwarded; a line below it is dropped.
  Floor Level
  // Level_Field is the key whose value names each line's level.
  Level_Field string
  // Forward holds the one in-flight write toward Stream.
  Forward Forward
  // Object is the scan of the line being classified; the caller fills its storage.
  Object Object
  // Kept is the caller-owned storage for the lines of one Write that clear the floor, at
  // most TRANSFORM_SIZE_MAXIMUM bytes; a Write that outgrows it retires Stream_Buffer_Full.
  Kept []byte
}

// New_Level_Filter fills filter over stream in place, resolving an empty level field name to
// jlog's default so it matches a default logger. The caller filled Kept and Object first.
func New_Level_Filter(
  filter *Level_Filter,
  stream nbio.Stream,
  floor Level,
  level_field_name string,
) {
  aver.Always(filter != nil, "jlog: level filter storage is required")
  aver.Always(len(filter.Kept) > 0, "jlog: level filter kept storage is required")
  aver.Always(!(len(filter.Kept) > TRANSFORM_SIZE_MAXIMUM), "jlog: kept exceeds bound")
  object_check(&filter.Object)
  filter.Stream = stream
  filter.Floor = floor
  filter.Level_Field = string_or(level_field_name, DEFAULT_LEVEL_FIELD_NAME)
  filter.Forward = Forward{}
  aver.Always(filter.Level_Field != "", "jlog: level field name is non-empty")
}

// Level_Filter_To_Stream forwards each written line whose level clears the Floor and drops the
// rest, submitting the survivors to the inner Stream once.
func Level_Filter_To_Stream(filter *Level_Filter) (stream nbio.Stream) {
  aver.Always(filter != nil, "jlog: level filter must not be nil")
  return nbio.Stream{State: filter, Procedure: level_filter_procedure}
}

func level_filter_procedure(
  state nbio.State,
  completion *nbio.Completion,
  mode nbio.Stream_Mode,
  payload []byte,
  offset int64,
  whence nbio.Seek_From,
  callback nbio.Stream_Callback,
) {
  filter := state.(*Level_Filter)
  if mode == nbio.STREAM_MODE_WRITE {
    if transform_busy(&filter.Forward, completion, callback) {
      return
    }
    kept := Buffer{Bytes: filter.Kept}
    level_filter_keep(filter, &kept, payload)
    transform_write(
      filter.Stream, &filter.Forward, completion, len(payload), &kept, callback,
    )
    return
  }
  transform_submit(filter.Stream, completion, mode, offset, whence, callback)
}

func level_filter_keep(filter *Level_Filter, kept *Buffer, payload []byte) {
  line_start := 0
  for index := 0; index < len(payload); index++ {
    if payload[index] != '\n' {
      continue
    }
    if filter_admits(filter, payload[line_start:index+1]) {
      buffer_write_bytes(kept, payload[line_start:index+1])
    }
    line_start = index + 1
  }
  // A trailing chunk with no newline is not something jlog produces (every line ends in \n),
  // but a filter on a raw pipe might see one; classify it the same, rather than dropping it.
  if line_start < len(payload) {
    if filter_admits(filter, payload[line_start:]) {
      buffer_write_bytes(kept, payload[line_start:])
    }
  }
}

// Reports whether line clears the filter's floor. A line that is not a JSON object, or one with
// no level field, is admitted: the filter suppresses by level and never swallows output it
// cannot classify.
func filter_admits(filter *Level_Filter, line []byte) (admit bool) {
  if !scan_object(&filter.Object, line) {
    return true
  }
  return !(level_from_wire(member_text(&filter.Object, filter.Level_Field)) < filter.Floor)
}

// Maps a level's wire name back to its Level, the inverse of Level_String, so a writer can
// compare a rendered line's level to a floor. An unrecognized or absent name maps to LEVEL_NONE,
// which clears any real floor, so a level-less or custom line is never taken for noise.
func level_from_wire(wire []byte) (level Level) {
  switch string(wire) {
  case "trace":
    return LEVEL_TRACE
  case "debug":
    return LEVEL_DEBUG
  case "info":
    return LEVEL_INFO
  case "warn":
    return LEVEL_WARN
  case "error":
    return LEVEL_ERROR
  }
  return LEVEL_NONE
}

// New_Console_Logger builds a console logger over state, which the caller filled: Console
// (its Stream, Color, Rendered, and Object storage), Filter (its Kept and Object storage),
// Capture, and Configuration (Clock, Caller). It renders human-readable
// lines to the Console at floor and above and, when Capture is set, tees every level as raw
// JSON to Capture, so one logger drives a quiet console and a full capture at once. It is pure:
// every sink, the clock, and the caller lookup are injected, so the composition root decides
// what they bind to. The logger's own floor is Trace: the one gate runs before any writer, so
// only a Trace floor lets every level reach Capture; floor gates the console branch alone.
func New_Console_Logger(state *Console_Logger, storage []byte, floor Level) (logger Logger) {
  aver.Always(state != nil, "jlog: console logger storage is required")
  New_Level_Filter(
    &state.Filter,
    Console_To_Stream(&state.Console),
    floor,
    string(DEFAULT_LEVEL_FIELD_NAME),
  )
  stream := Level_Filter_To_Stream(&state.Filter)
  if state.Capture.Procedure != nil {
    state.Tee = nbio.Stream_Tee{First: state.Capture, Second: stream}
    stream = nbio.Tee_To_Stream(&state.Tee)
  }
  state.Configuration.Stream = stream
  logger = New(&state.Configuration, storage, LEVEL_TRACE)
  logger.Auto_Timestamp = true
  return logger
}

// Console_Logger is the caller-owned state behind one console logger, so New_Console_Logger
// allocates nothing.
type Console_Logger struct {
  // Configuration is the logger's shared dependency set; the caller sets Clock and Caller.
  Configuration Logger_Configuration
  // Console renders lines for a human; the caller sets its Stream, Color, and storage.
  Console Console
  // Capture receives every raw JSON level; the caller leaves it zero to disable capture.
  Capture nbio.Stream
  // Filter gates the console branch by level; the caller sets its storage.
  Filter Level_Filter
  // Tee fans out to Capture when one is set.
  Tee nbio.Stream_Tee
}
