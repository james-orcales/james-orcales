// Package jlog is a zero-allocation, dependency-injected structured JSON logger.
//
// The house linter bans methods (except those satisfying a stdlib interface) and
// names a free function after its first parameter's type, so there is no fluent
// builder: a log line is one Logger_-prefixed call taking the logger, a message,
// and a variadic list of Field values built by the  constructors.
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
// (shared/sync/diode).
package jlog

import (
	"unsafe"

	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/sim/nbio"
	"local/james-orcales/shared/sim/time"
	"local/james-orcales/shared/strconv"
	"local/james-orcales/shared/unicode/utf8"
)

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

// Buffer is the byte accumulator the encoders append into. It is a named slice so
// the encoder helpers' (Buffer, []byte) signatures present two distinct types and
// do not trip the input-struct rule.
type Buffer []byte

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
	// Caller maps a frame-skip count to a "file:line" string; nil disables Caller.
	Caller func(skip int) (location string)
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
	// Storage is the caller-owned byte budget; a line that outgrows it is dropped, never kept.
	Storage []byte
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
	// Dropped counts lines that outgrew Line.Storage or arrived while it was still borrowed.
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
	aver.Always(cap(storage) <= LINE_BYTES_MAXIMUM, "jlog: line storage exceeds bound")
	if configuration.Duration_Unit == 0 {
		configuration.Duration_Unit = time.NANOSECOND
	}
	aver.Always(configuration.Duration_Unit > 0, "jlog: duration unit must be positive")
	configuration.Timestamp_Field_Name = string_or(
		configuration.Timestamp_Field_Name, DEFAULT_TIMESTAMP_FIELD_NAME)
	configuration.Level_Field_Name = string_or(
		configuration.Level_Field_Name, DEFAULT_LEVEL_FIELD_NAME)
	configuration.Message_Field_Name = string_or(
		configuration.Message_Field_Name, DEFAULT_MESSAGE_FIELD_NAME)
	configuration.Error_Field_Name = string_or(
		configuration.Error_Field_Name, DEFAULT_ERROR_FIELD_NAME)
	configuration.Caller_Field_Name = string_or(
		configuration.Caller_Field_Name, DEFAULT_CALLER_FIELD_NAME)
	configuration.Stack_Field_Name = string_or(
		configuration.Stack_Field_Name, DEFAULT_STACK_FIELD_NAME)
	logger.Configuration = configuration
	logger.Line.Storage = storage[:0]
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
	buffer := Buffer(line.Storage[:0])
	buffer = buffer_append_begin_marker(buffer)
	buffer = buffer_append_level(buffer, level, configuration)
	if logger.Auto_Timestamp {
		buffer = buffer_append_realtime(buffer, configuration)
	}
	if logger.Auto_Caller {
		seed := Field{Kind: KIND_CALLER, Number: DEFAULT_CALLER_SKIP}
		buffer = buffer_append_caller(buffer, &seed, configuration)
	}
	if len(logger.Prefix) > 1 {
		buffer = buffer_append_object_data(buffer, logger.Prefix)
	}
	for index := 0; index < len(fields); index++ {
		buffer = buffer_encode_field(buffer, &fields[index], configuration)
	}
	if message != "" {
		buffer = buffer_append_key(buffer, Key(configuration.Message_Field_Name))
		buffer = buffer_append_string(buffer, message)
	}
	buffer = buffer_append_end_marker(buffer)
	buffer = append(buffer, '\n')
	aver.Always(buffer[0] == '{', "jlog: line must open with a brace")
	aver.Always(buffer[len(buffer)-2] == '}', "jlog: line must close with a brace")
	aver.Always(buffer[len(buffer)-1] == '\n', "jlog: line must end with a newline")
	// The append grew past Storage: the line is complete but lives in a fresh allocation, so
	// it is dropped rather than retained.
	if cap(buffer) != cap(line.Storage) {
		logger.Dropped++
		return
	}
	line.Borrowed = true
	nbio.Write(configuration.Stream, &line.Completion, buffer, line_retire)
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
	aver.Always(cap(storage) <= LINE_BYTES_MAXIMUM, "jlog: line storage exceeds bound")
	aver.Always(cap(prefix) <= PREFIX_BYTES_MAXIMUM, "jlog: prefix storage exceeds bound")
	child.Configuration = parent.Configuration
	child.Line.Storage = storage[:0]
	child.Floor = parent.Floor
	child.Auto_Timestamp = parent.Auto_Timestamp
	child.Auto_Caller = parent.Auto_Caller
	budget_size := cap(prefix)
	prefix = prefix[:0]
	if len(parent.Prefix) > 0 {
		prefix = append(prefix, parent.Prefix...)
	} else {
		prefix = buffer_append_begin_marker(prefix)
	}
	for index := 0; index < len(fields); index++ {
		prefix = buffer_encode_field(prefix, &fields[index], parent.Configuration)
	}
	aver.Always(cap(prefix) == budget_size, "jlog: prefix exceeds its storage")
	child.Prefix = prefix
	return child
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
	for index := value.Count - 1; index >= 0; index-- {
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
	if count >= value.Count {
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
	return value.Digits[count] >= '5'
}

func decimal_round_down(value *Decimal, count int) {
	if count < 0 {
		return
	}
	if count >= value.Count {
		return
	}
	value.Count = count
	decimal_trim(value)
}

func decimal_round_up(value *Decimal, count int) {
	if count < 0 {
		return
	}
	if count >= value.Count {
		return
	}
	for index := count - 1; index >= 0; index-- {
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
	value *Decimal, mantissa uint64, exponent int, information Float_Information,
) {
	if mantissa == 0 {
		value.Count = 0
		return
	}
	minimum_exponent := information.Exponent_Bias + 1
	if exponent > minimum_exponent {
		decimal_distance := 332 * (value.Point - value.Count)
		binary_distance := 100 * (exponent - int(information.Mantissa_Bit_Count))
		if decimal_distance >= binary_distance {
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
	upper *Decimal, lower *Decimal, mantissa uint64, exponent int,
	minimum_exponent int, information Float_Information,
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
		if middle_index >= value.Count {
			break
		}
		lower_index := upper_index - upper.Point + lower.Point
		lower_digit := byte('0')
		if lower_index >= 0 {
			if lower_index < lower.Count {
				lower_digit = lower.Digits[lower_index]
			}
		}
		middle_digit := byte('0')
		if middle_index >= 0 {
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
func buffer_encode_field(
	destination Buffer, field *Field, configuration *Logger_Configuration,
) (output Buffer) {
	if field.Kind >= KIND_ERROR {
		switch field.Kind {
		case KIND_ERROR:
			return buffer_append_error(destination, field, configuration)
		case KIND_TIMESTAMP:
			return buffer_append_realtime(destination, configuration)
		case KIND_CALLER:
			return buffer_append_caller(destination, field, configuration)
		}
		return destination
	}
	destination = buffer_append_key(destination, field.Key)
	count := int(field.Number)
	switch field.Kind {
	case KIND_STRING:
		return buffer_append_string(destination, unsafe.String((*byte)(field.Data), count))
	case KIND_INTEGER:
		return buffer_append_int64(destination, field.Number)
	case KIND_UNSIGNED:
		return buffer_append_uint64(destination, uint64(field.Number))
	case KIND_FLOAT32:
		bits := float64_from_bits(uint64(field.Number))
		return buffer_append_float(destination, bits, FLOAT_BITS_32)
	case KIND_FLOAT64:
		bits := float64_from_bits(uint64(field.Number))
		return buffer_append_float(destination, bits, FLOAT_BITS_64)
	case KIND_BOOLEAN:
		return buffer_append_boolean(destination, field.Number == 1)
	case KIND_BYTES:
		return buffer_append_bytes(destination, unsafe.Slice((*byte)(field.Data), count))
	case KIND_HEXADECIMAL:
		raw := unsafe.Slice((*byte)(field.Data), count)
		return buffer_append_hexadecimal(destination, raw)
	case KIND_RAW_JSON:
		return append(destination, unsafe.Slice((*byte)(field.Data), count)...)
	case KIND_TIME:
		return buffer_append_rfc3339(destination, field.Number)
	case KIND_DURATION:
		divided := field.Number / int64(configuration.Duration_Unit)
		return buffer_append_int64(destination, divided)
	case KIND_STRINGS:
		values := unsafe.Slice((*string)(field.Data), count)
		return buffer_append_strings(destination, values)
	case KIND_INTEGERS:
		return buffer_append_integers(destination, unsafe.Slice((*int)(field.Data), count))
	case KIND_FLOATS:
		values := unsafe.Slice((*float64)(field.Data), count)
		return buffer_append_floats(destination, values)
	case KIND_BOOLEANS:
		return buffer_append_booleans(destination, unsafe.Slice((*bool)(field.Data), count))
	case KIND_DURATIONS:
		values := unsafe.Slice((*time.Duration)(field.Data), count)
		return buffer_append_durations(destination, values, configuration.Duration_Unit)
	}
	return destination
}

func buffer_append_realtime(
	destination Buffer, configuration *Logger_Configuration,
) (output Buffer) {
	destination = buffer_append_key(destination, Key(configuration.Timestamp_Field_Name))
	moment := time.Clock_Now_Realtime(configuration.Clock)
	return buffer_append_rfc3339(destination, int64(moment))
}

func buffer_append_caller(
	destination Buffer, field *Field, configuration *Logger_Configuration,
) (output Buffer) {
	if configuration.Caller == nil {
		return destination
	}
	location := configuration.Caller(int(field.Number))
	destination = buffer_append_key(destination, Key(configuration.Caller_Field_Name))
	return buffer_append_string(destination, location)
}

// The stack field precedes the error field, matching the convention that context
// comes before the value it explains.
func buffer_append_error(
	destination Buffer, field *Field, configuration *Logger_Configuration,
) (output Buffer) {
	error_value, _ := field.Boxed.(error)
	if configuration.Stack_Marshaler != nil {
		if error_value != nil {
			stack := configuration.Stack_Marshaler(error_value)
			stack_key := Key(configuration.Stack_Field_Name)
			destination = buffer_append_key(destination, stack_key)
			destination = buffer_append_string(destination, stack)
		}
	}
	destination = buffer_append_key(destination, Key(configuration.Error_Field_Name))
	if error_value == nil {
		return buffer_append_nil(destination)
	}
	return buffer_append_string(destination, error_value.Error())
}

func buffer_append_strings(destination Buffer, values []string) (output Buffer) {
	destination = append(destination, '[')
	for index := 0; index < len(values); index++ {
		if index > 0 {
			destination = append(destination, ',')
		}
		destination = buffer_append_string(destination, values[index])
	}
	return append(destination, ']')
}

func buffer_append_integers(destination Buffer, values []int) (output Buffer) {
	destination = append(destination, '[')
	for index := 0; index < len(values); index++ {
		if index > 0 {
			destination = append(destination, ',')
		}
		destination = buffer_append_int64(destination, int64(values[index]))
	}
	return append(destination, ']')
}

func buffer_append_floats(destination Buffer, values []float64) (output Buffer) {
	destination = append(destination, '[')
	for index := 0; index < len(values); index++ {
		if index > 0 {
			destination = append(destination, ',')
		}
		destination = buffer_append_float(destination, values[index], FLOAT_BITS_64)
	}
	return append(destination, ']')
}

func buffer_append_booleans(destination Buffer, values []bool) (output Buffer) {
	destination = append(destination, '[')
	for index := 0; index < len(values); index++ {
		if index > 0 {
			destination = append(destination, ',')
		}
		destination = buffer_append_boolean(destination, values[index])
	}
	return append(destination, ']')
}

func buffer_append_durations(
	destination Buffer, values []time.Duration, unit time.Duration,
) (output Buffer) {
	destination = append(destination, '[')
	for index := 0; index < len(values); index++ {
		if index > 0 {
			destination = append(destination, ',')
		}
		destination = buffer_append_int64(destination, int64(values[index])/int64(unit))
	}
	return append(destination, ']')
}

func buffer_append_begin_marker(destination Buffer) (output Buffer) {
	return append(destination, '{')
}

func buffer_append_end_marker(destination Buffer) (output Buffer) {
	return append(destination, '}')
}

func buffer_append_nil(destination Buffer) (output Buffer) {
	return append(destination, 'n', 'u', 'l', 'l')
}

func buffer_append_key(destination Buffer, key Key) (output Buffer) {
	if len(destination) > 0 {
		if destination[len(destination)-1] != '{' {
			destination = append(destination, ',')
		}
	}
	destination = buffer_append_string(destination, string(key))
	return append(destination, ':')
}

// The fragment's leading brace is dropped and a separating comma added so a
// sub-logger's pre-encoded fields splice into the open object.
func buffer_append_object_data(destination Buffer, prefix []byte) (output Buffer) {
	body := prefix
	leading_brace := false
	if len(body) > 0 {
		if body[0] == '{' {
			leading_brace = true
		}
	}
	if leading_brace {
		if len(destination) > 1 {
			destination = append(destination, ',')
		}
		body = body[1:]
		return append(destination, body...)
	}
	if len(destination) > 1 {
		destination = append(destination, ',')
	}
	return append(destination, body...)
}

func buffer_append_int64(destination Buffer, value int64) (output Buffer) {
	var storage [strconv.INTEGER_TEXT_SIZE_MAXIMUM]byte
	count := strconv.Format_Integer_Into(
		storage[:], strconv.Signed_Integer(value), strconv.DECIMAL_BASE,
	)
	return append(destination, storage[:int(count)]...)
}

func buffer_append_uint64(destination Buffer, value uint64) (output Buffer) {
	var storage [strconv.DIGIT_TEXT_SIZE_MAXIMUM]byte
	count := strconv.Format_Unsigned_Integer_Into(
		storage[:], strconv.Unsigned_Integer(value), strconv.DECIMAL_BASE,
	)
	return append(destination, storage[:int(count)]...)
}

func buffer_append_boolean(destination Buffer, value bool) (output Buffer) {
	if value {
		return append(destination, 't', 'r', 'u', 'e')
	}
	return append(destination, 'f', 'a', 'l', 's', 'e')
}

// Timestamps render as an RFC 3339 UTC string rather than a raw nanosecond integer:
// epoch-nanosecond integers exceed JSON's safe 2^53 range and get mangled by
// double-precision consumers (jq, JavaScript). A fractional part is emitted only
// when the sub-second nanoseconds are nonzero. See the package doc's referenced
// article ("Don't Use Very Large Numbers").
func buffer_append_rfc3339(destination Buffer, nanos int64) (output Buffer) {
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
	aver.Always(day_seconds >= 0 && day_seconds < SECONDS_PER_DAY, "jlog: bad day seconds")
	year, month, day := civil_from_days(days)
	aver.Always(month >= 1 && month <= 12, "jlog: civil month out of range")
	aver.Always(day >= 1 && day <= 31, "jlog: civil day out of range")
	within_hour := day_seconds % SECONDS_PER_HOUR
	destination = append(destination, '"')
	destination = buffer_append_four_digits(destination, year)
	destination = append(destination, '-')
	destination = buffer_append_two_digits(destination, int64(month))
	destination = append(destination, '-')
	destination = buffer_append_two_digits(destination, int64(day))
	destination = append(destination, 'T')
	destination = buffer_append_two_digits(destination, day_seconds/SECONDS_PER_HOUR)
	destination = append(destination, ':')
	destination = buffer_append_two_digits(destination, within_hour/SECONDS_PER_MINUTE)
	destination = append(destination, ':')
	destination = buffer_append_two_digits(destination, within_hour%SECONDS_PER_MINUTE)
	if fraction != 0 {
		destination = append(destination, '.')
		destination = buffer_append_nanoseconds(destination, fraction)
	}
	return append(destination, 'Z', '"')
}

// Two zero-padded decimal digits (00-99).
func buffer_append_two_digits(destination Buffer, value int64) (output Buffer) {
	return append(destination, byte('0'+value/10), byte('0'+value%10))
}

// Four zero-padded decimal digits (0000-9999). A year past 9999 drops its top
// digit, which no realistic log timestamp reaches.
func buffer_append_four_digits(destination Buffer, value int64) (output Buffer) {
	return append(destination,
		byte('0'+value/1000%10),
		byte('0'+value/100%10),
		byte('0'+value/10%10),
		byte('0'+value%10))
}

// Nine zero-padded decimal digits, most significant first, for the nanosecond
// fraction of a second.
func buffer_append_nanoseconds(destination Buffer, value int64) (output Buffer) {
	divisor := int64(100_000_000)
	for divisor > 0 {
		destination = append(destination, byte('0'+value/divisor%10))
		divisor /= 10
	}
	return destination
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
	if month_portion >= 10 {
		computed_month = month_portion - 9
	}
	if computed_month <= 2 {
		computed_year++
	}
	return computed_year, int(computed_month), int(computed_day)
}

func buffer_append_level(
	destination Buffer, level Level, configuration *Logger_Configuration,
) (output Buffer) {
	if level == LEVEL_NONE {
		return destination
	}
	if configuration.Level_Field_Name == "" {
		return destination
	}
	destination = buffer_append_key(destination, Key(configuration.Level_Field_Name))
	return buffer_append_string(destination, Level_String(level))
}

// JSON has no NaN/Inf, so those render as strings; a leading exponent zero is
// trimmed to match es6 number output.
func buffer_append_float(destination Buffer, value float64, bit_size int) (output Buffer) {
	bits := float64_bits(value)
	if float64_is_nan(bits) {
		return append(destination, '"', 'N', 'a', 'N', '"')
	}
	if bits == FLOAT64_POSITIVE_INFINITY_BITS {
		return append(destination, '"', '+', 'I', 'n', 'f', '"')
	}
	if bits == FLOAT64_NEGATIVE_INFINITY_BITS {
		return append(destination, '"', '-', 'I', 'n', 'f', '"')
	}
	format := byte('f')
	if float_needs_exponent(value, bit_size) {
		format = 'e'
	}
	destination = buffer_append_finite_float(destination, value, bit_size, format)
	if format == 'e' {
		destination = buffer_clean_exponent(destination)
	}
	return destination
}

func buffer_append_finite_float(
	destination Buffer, value float64, bit_size int, format byte,
) (output Buffer) {
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
		return buffer_append_float_exponent(
			destination, negative, &decimal, precision,
		)
	}
	return buffer_append_float_fixed(destination, negative, &decimal, precision)
}

func buffer_append_float_exponent(
	destination Buffer, negative bool, decimal *Decimal, precision int,
) (output Buffer) {
	if negative {
		destination = append(destination, '-')
	}
	first := byte('0')
	if decimal.Count != 0 {
		first = decimal.Digits[0]
	}
	destination = append(destination, first)
	if precision > 0 {
		destination = append(destination, '.')
		index := 1
		end_index := decimal.Count
		if end_index > precision+1 {
			end_index = precision + 1
		}
		if index < end_index {
			destination = append(destination, decimal.Digits[index:end_index]...)
			index = end_index
		}
		for index <= precision {
			destination = append(destination, '0')
			index++
		}
	}
	destination = append(destination, 'e')
	exponent := decimal.Point - 1
	if decimal.Count == 0 {
		exponent = 0
	}
	if exponent < 0 {
		destination = append(destination, '-')
		exponent = -exponent
	} else {
		destination = append(destination, '+')
	}
	if exponent < 10 {
		return append(destination, '0', byte(exponent)+'0')
	}
	if exponent < 100 {
		return append(destination, byte(exponent/10)+'0', byte(exponent%10)+'0')
	}
	return append(destination,
		byte(exponent/100)+'0', byte(exponent/10%10)+'0', byte(exponent%10)+'0',
	)
}

func buffer_append_float_fixed(
	destination Buffer, negative bool, decimal *Decimal, precision int,
) (output Buffer) {
	if negative {
		destination = append(destination, '-')
	}
	if decimal.Point > 0 {
		end_index := decimal.Count
		if end_index > decimal.Point {
			end_index = decimal.Point
		}
		destination = append(destination, decimal.Digits[:end_index]...)
		for index := end_index; index < decimal.Point; index++ {
			destination = append(destination, '0')
		}
	} else {
		destination = append(destination, '0')
	}
	if precision <= 0 {
		return destination
	}
	destination = append(destination, '.')
	for index := 0; index < precision; index++ {
		digit := byte('0')
		digit_index := decimal.Point + index
		if digit_index >= 0 {
			if digit_index < decimal.Count {
				digit = decimal.Digits[digit_index]
			}
		}
		destination = append(destination, digit)
	}
	return destination
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
		if magnitude >= FLOAT_EXPONENT_HIGH {
			return true
		}
		return false
	}
	if float32(magnitude) < FLOAT_EXPONENT_LOW {
		return true
	}
	if float32(magnitude) >= FLOAT_EXPONENT_HIGH {
		return true
	}
	return false
}

func buffer_clean_exponent(destination Buffer) (output Buffer) {
	count := len(destination)
	if count < 4 {
		return destination
	}
	if destination[count-4] != 'e' {
		return destination
	}
	if destination[count-3] != '-' {
		return destination
	}
	if destination[count-2] != '0' {
		return destination
	}
	destination[count-2] = destination[count-1]
	return destination[:count-1]
}

// The all-plain common case is appended in one shot; the escape path runs only
// once a byte needs it.
func buffer_append_string(destination Buffer, value string) (output Buffer) {
	destination = append(destination, '"')
	for index := 0; index < len(value); index++ {
		if byte_is_plain(value[index]) {
			continue
		}
		destination = buffer_append_string_complex(destination, value, index)
		return append(destination, '"')
	}
	destination = append(destination, value...)
	return append(destination, '"')
}

func buffer_append_string_complex(destination Buffer, value string, scan int) (output Buffer) {
	run_start := 0
	index := scan
	for index < len(value) {
		current := value[index]
		if current >= byte(utf8.CHARACTER_SELF) {
			decoded, width := utf8.Decode_Character_Text(utf8.Text(value[index:]))
			if decoded == utf8.REPLACEMENT_CHARACTER {
				if width == utf8.CHARACTER_SIZE_MINIMUM {
					destination = append(destination, value[run_start:index]...)
					destination = buffer_append_replacement(destination)
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
		destination = append(destination, value[run_start:index]...)
		destination = buffer_append_escaped_byte(destination, current)
		index++
		run_start = index
	}
	return append(destination, value[run_start:]...)
}

func buffer_append_hexadecimal(destination Buffer, value []byte) (output Buffer) {
	destination = append(destination, '"')
	for index := 0; index < len(value); index++ {
		current := value[index]
		high := hexadecimal_digit(current >> 4)
		low := hexadecimal_digit(current & 0x0f)
		destination = append(destination, high, low)
	}
	return append(destination, '"')
}

func buffer_append_bytes(destination Buffer, value []byte) (output Buffer) {
	destination = append(destination, '"')
	for index := 0; index < len(value); index++ {
		if byte_is_plain(value[index]) {
			continue
		}
		destination = buffer_append_bytes_complex(destination, value, index)
		return append(destination, '"')
	}
	destination = append(destination, value...)
	return append(destination, '"')
}

func buffer_append_bytes_complex(destination Buffer, value []byte, scan int) (output Buffer) {
	run_start := 0
	index := scan
	for index < len(value) {
		current := value[index]
		if current >= byte(utf8.CHARACTER_SELF) {
			decoded, width := utf8.Decode_Character(utf8.Bytes(value[index:]))
			if decoded == utf8.REPLACEMENT_CHARACTER {
				if width == utf8.CHARACTER_SIZE_MINIMUM {
					destination = append(destination, value[run_start:index]...)
					destination = buffer_append_replacement(destination)
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
		destination = append(destination, value[run_start:index]...)
		destination = buffer_append_escaped_byte(destination, current)
		index++
		run_start = index
	}
	return append(destination, value[run_start:]...)
}

func buffer_append_escaped_byte(destination Buffer, value byte) (output Buffer) {
	switch value {
	case '"', '\\':
		return append(destination, '\\', value)
	case '\b':
		return append(destination, '\\', 'b')
	case '\f':
		return append(destination, '\\', 'f')
	case '\n':
		return append(destination, '\\', 'n')
	case '\r':
		return append(destination, '\\', 'r')
	case '\t':
		return append(destination, '\\', 't')
	}
	high := hexadecimal_digit(value >> 4)
	low := hexadecimal_digit(value & 0x0f)
	return append(destination, '\\', 'u', '0', '0', high, low)
}

// Appends the JSON escape for the U+FFFD replacement character. zerolog emits this
// six-byte escape (not the raw rune) for a malformed UTF-8 byte; jlog matches it so
// the wire bytes are identical on bad input.
func buffer_append_replacement(destination Buffer) (output Buffer) {
	return append(destination, '\\', 'u', 'f', 'f', 'f', 'd')
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
// flat-JSON object. Being off the hot path, it may allocate freely. It is pure — it reaches for no
// OS, clock, or global state; every dependency arrives through a Console field — so it belongs
// here, not in the composition tier.

// An ANSI SGR sequence. A distinct type so buffer_paint takes it without colliding with its
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
// field names is not matched, which no caller needs. It is passed by value and never mutated, so a
// copy is a faithful, independent Console. Build one directly with a struct literal.
type Console struct {
	// Stream receives the rendered text.
	Stream nbio.Stream
	// Color enables ANSI color in the rendered output.
	Color bool
}

// TRANSFORM_MODES excludes reads and cursor modes because a transformed byte count names no
// position in the inner Stream.
const TRANSFORM_MODES nbio.Stream_Mode_Set = 1<<nbio.STREAM_MODE_CLOSE |
	1<<nbio.STREAM_MODE_FLUSH | 1<<nbio.STREAM_MODE_WRITE |
	1<<nbio.STREAM_MODE_DESTROY | 1<<nbio.STREAM_MODE_QUERY

// Console_To_Stream renders each written batch of newline-terminated JSON lines and submits the
// human-readable form to the inner Stream once.
func Console_To_Stream(console *Console) (stream nbio.Stream) {
	aver.Always(console != nil, "jlog: console must not be nil")
	return nbio.Stream{State: console, Procedure: console_procedure}
}

func console_procedure(
	state nbio.State, completion *nbio.Completion, mode nbio.Stream_Mode, payload []byte,
	offset int64, whence nbio.Seek_From, callback nbio.Stream_Callback,
) {
	console := state.(*Console)
	if mode == nbio.STREAM_MODE_WRITE {
		rendered := console_render(*console, payload)
		transform_write(console.Stream, completion, payload, rendered, callback)
		return
	}
	transform_submit(console.Stream, completion, mode, offset, whence, callback)
}

// Forward keeps the outer retirement and the transformed buffer alive until the inner Stream
// retires, since a kernel-backed Stream may retain only the Completion.
type Forward struct {
	// Callback is the outer retirement.
	Callback nbio.Stream_Callback
	// Buffer is the transformed bytes the inner Stream borrows.
	Buffer Buffer
	// Count is the outer payload length reported on success.
	Count int
}

// A transforming Stream reports len(payload) consumed on success because it emits a different
// byte count than it takes. Forward allocates per write; transforms are off the hot path.
func transform_write(
	inner nbio.Stream, completion *nbio.Completion, payload []byte, transformed Buffer,
	callback nbio.Stream_Callback,
) {
	if len(transformed) == 0 {
		completion.Data = len(payload)
		completion.Error = nil
		nbio.Stream_Callback_Call(callback, completion)
		return
	}
	if inner.Procedure == nil {
		completion.Data = 0
		completion.Error = nbio.Stream_Empty
		nbio.Stream_Callback_Call(callback, completion)
		return
	}
	forward := &Forward{Callback: callback, Buffer: transformed, Count: len(payload)}
	inner.Procedure(inner.State, completion, nbio.STREAM_MODE_WRITE, transformed, 0,
		nbio.SEEK_FROM_START, nbio.Stream_Callback{
			State: forward, Data: len(transformed), Procedure: transform_retire,
		})
}

func transform_retire(
	state nbio.State, transformed_count nbio.Stream_Callback_Data, _ nbio.Callback,
	completion nbio.Completion_Handle,
) {
	forward := state.(*Forward)
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
	inner nbio.Stream, completion *nbio.Completion, mode nbio.Stream_Mode, offset int64,
	whence nbio.Seek_From, callback nbio.Stream_Callback,
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

func console_render(console Console, payload []byte) (rendered Buffer) {
	rendered = make(Buffer, 0, len(payload)*2)
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
	return rendered
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

// One key/value member of a rendered line, holding the value already reduced to its display text
// and kind so rendering never re-parses.
type Member struct {
	// Key is the JSON object key.
	Key string
	// Text is the value's display form, already unescaped or compacted.
	Text string
	// Kind selects how Text is quoted when rendered.
	Kind Value_Kind
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
func scan_object(line []byte) (members []Member, parsed bool) {
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
		members = append(members, Member{Key: key, Text: text, Kind: kind})
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
			decoded, valid := json_string_decode(line[start+1 : index])
			return decoded, index + 1, valid
		}
		index++
	}
	return "", start, false
}

func json_string_decode(encoded []byte) (decoded string, valid bool) {
	storage := make(Buffer, 0, len(encoded))
	for index := 0; index < len(encoded); index++ {
		current := encoded[index]
		if current < 0x20 {
			return "", false
		}
		if current != '\\' {
			storage = append(storage, current)
			continue
		}
		index++
		if index >= len(encoded) {
			return "", false
		}
		switch encoded[index] {
		case '"', '\\', '/':
			storage = append(storage, encoded[index])
		case 'b':
			storage = append(storage, '\b')
		case 'f':
			storage = append(storage, '\f')
		case 'n':
			storage = append(storage, '\n')
		case 'r':
			storage = append(storage, '\r')
		case 't':
			storage = append(storage, '\t')
		case 'u':
			code, tail_index, known := json_hexadecimal_character(encoded, index+1)
			if !known {
				return "", false
			}
			index = tail_index
			storage = json_append_character(storage, code)
		default:
			return "", false
		}
	}
	return string(storage), true
}

func json_hexadecimal_character(
	encoded []byte, start int,
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
	if value >= '0' {
		if value <= '9' {
			return value - '0', true
		}
	}
	if value >= 'a' {
		if value <= 'f' {
			return value - 'a' + 10, true
		}
	}
	if value >= 'A' {
		if value <= 'F' {
			return value - 'A' + 10, true
		}
	}
	return 0, false
}

func json_append_character(destination Buffer, character rune) (output Buffer) {
	if character < 0x80 {
		return append(destination, byte(character))
	}
	if character < 0x800 {
		return append(destination,
			byte(0xc0|character>>6),
			byte(0x80|character&0x3f),
		)
	}
	return append(destination,
		byte(0xe0|character>>12),
		byte(0x80|character>>6&0x3f),
		byte(0x80|character&0x3f),
	)
}

// Reads one JSON value beginning at start, classifying it: a string is unescaped, an array or
// object is captured as its verbatim (already-compact) source bytes, and anything else is a scalar
// literal whose digits are kept exactly.
func scan_value(line []byte, start int) (text string, kind Value_Kind, next int, ok bool) {
	if start >= len(line) {
		return "", VALUE_IS_LITERAL, start, false
	}
	if line[start] == '"' {
		value, value_next, value_ok := scan_string(line, start)
		if !value_ok {
			return "", VALUE_IS_LITERAL, start, false
		}
		return value, VALUE_IS_STRING, value_next, true
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
func scan_compound(line []byte, start int) (text string, kind Value_Kind, next int, ok bool) {
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
				return "", VALUE_IS_COMPOUND, start, false
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
				return string(line[start:index]), VALUE_IS_COMPOUND, index, true
			}
			continue
		}
		index++
	}
	return "", VALUE_IS_COMPOUND, start, false
}

// Reads a scalar literal (number, true, false, or null) beginning at start, ending it at the first
// value terminator so its exact source digits are captured.
func scan_literal(line []byte, start int) (text string, kind Value_Kind, next int, ok bool) {
	index := start
	for index < len(line) {
		if byte_ends_literal(line[index]) {
			break
		}
		index++
	}
	if index == start {
		return "", VALUE_IS_LITERAL, start, false
	}
	return string(line[start:index]), VALUE_IS_LITERAL, index, true
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
	destination Buffer, members []Member, console Console,
) (output Buffer) {
	// An absent header key and an empty one render the same — nothing — so a plain non-empty
	// check covers both, and jlog never emits an empty timestamp, level, or message anyway. The
	// timestamp is coarsened to the second here; the tail keeps any Time field's precision.
	time_text := timestamp_seconds(member_text(members, string(DEFAULT_TIMESTAMP_FIELD_NAME)))
	level_text := member_text(members, string(DEFAULT_LEVEL_FIELD_NAME))
	message_text := member_text(members, string(DEFAULT_MESSAGE_FIELD_NAME))
	wrote := false
	if time_text != "" {
		destination = buffer_append_separated(destination, wrote)
		destination = buffer_paint(destination, ANSI_FAINT, console.Color, time_text)
		wrote = true
	}
	if level_text != "" {
		destination = buffer_append_separated(destination, wrote)
		destination = buffer_append_level_tag(destination, level_text, console.Color)
		wrote = true
	}
	if message_text != "" {
		destination = buffer_append_separated(destination, wrote)
		destination = buffer_paint(destination, ANSI_BOLD, console.Color, message_text)
		wrote = true
	}
	for index := 0; index < len(members); index++ {
		current := members[index]
		if member_is_header(current.Key) {
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
func member_is_header(key string) (header bool) {
	if key == string(DEFAULT_TIMESTAMP_FIELD_NAME) {
		return true
	}
	if key == string(DEFAULT_LEVEL_FIELD_NAME) {
		return true
	}
	if key == string(DEFAULT_MESSAGE_FIELD_NAME) {
		return true
	}
	return false
}

// Returns the display text of the first member with key, or "" when no member has it.
func member_text(members []Member, key string) (text string) {
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
func buffer_append_field(destination Buffer, field Member, console Console) (output Buffer) {
	destination = buffer_paint(destination, ANSI_CYAN, console.Color, field.Key)
	destination = append(destination, '=')
	value_color := Ansi_Code("")
	if field.Key == string(DEFAULT_ERROR_FIELD_NAME) {
		value_color = ANSI_RED
	}
	if field.Kind == VALUE_IS_STRING {
		if logfmt_needs_quote(field.Text) {
			if console.Color {
				destination = append(destination, value_color...)
			}
			destination = buffer_append_string(destination, field.Text)
			if console.Color {
				return append(destination, ANSI_RESET...)
			}
			return destination
		}
	}
	return buffer_paint(destination, value_color, console.Color, field.Text)
}

// Appends the level as its three-letter tag, painted by severity. Named apart from the encoder's
// buffer_append_level (which writes the level as a JSON field) since both share this package.
func buffer_append_level_tag(destination Buffer, wire string, color bool) (output Buffer) {
	return buffer_paint(destination, level_color(wire), color, level_label(wire))
}

// Appends text wrapped in code and a reset when color is on and code is a real color; otherwise
// appends text bare. An empty code means "no color for this part", so an unknown level or a
// non-error value stays uncolored even under color.
func buffer_paint(destination Buffer, code Ansi_Code, color bool, text string) (output Buffer) {
	if !color {
		return append(destination, text...)
	}
	if code == "" {
		return append(destination, text...)
	}
	destination = append(destination, code...)
	destination = append(destination, text...)
	return append(destination, ANSI_RESET...)
}

// Returns the display token for a field value: a string is bare unless logfmt requires quoting; a
// literal or compound value is already a safe bare token.
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
	count := len(wire)
	if count > 3 {
		count = 3
	}
	var storage [LEVEL_LABEL_SIZE_MAXIMUM]byte
	for index := 0; index < count; index++ {
		current := wire[index]
		if current >= 'a' {
			if current <= 'z' {
				current -= 'a' - 'A'
			}
		}
		storage[index] = current
	}
	return string(storage[:count])
}

// Maps a level's wire name to its color; an unknown level returns the empty code, meaning no
// color, so buffer_paint leaves it plain.
func level_color(wire string) (code Ansi_Code) {
	switch wire {
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

// Trims an RFC 3339 header timestamp to second precision by dropping its sub-second fraction and
// keeping the trailing zone, so "…:20.123456789Z" becomes "…:20Z" — a human reading the console
// needs no nanosecond granularity. A value with no fraction is returned unchanged; only the
// console display coarsens, never the JSON line.
func timestamp_seconds(value string) (seconds string) {
	fraction_offset := string_byte_index(value, '.')
	if fraction_offset < 0 {
		return value
	}
	// The zone is searched absolutely, not relative to the fraction, so the two offsets are
	// never added: an RFC 3339 UTC value carries exactly one 'Z', the terminal zone marker.
	zone_offset := string_byte_index(value, 'Z')
	if zone_offset < 0 {
		return value[:fraction_offset]
	}
	return value[:fraction_offset] + value[zone_offset:]
}

func string_byte_index(value string, sought byte) (index int) {
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
// trace and debug off the console. It is passed by value; New_Level_Filter sets its fields and
// nothing mutates them, so a copy is a faithful, independent filter.
type Level_Filter struct {
	// Stream receives the lines that clear the floor.
	Stream nbio.Stream
	// Floor is the lowest level forwarded; a line below it is dropped.
	Floor Level
	// Level_Field is the key whose value names each line's level.
	Level_Field string
}

// New_Level_Filter builds a Level_Filter over stream, resolving an empty level field name to
// jlog's default so it matches a default logger.
func New_Level_Filter(
	stream nbio.Stream, floor Level, level_field_name string,
) (filter Level_Filter) {
	filter.Stream = stream
	filter.Floor = floor
	filter.Level_Field = string_or(level_field_name, DEFAULT_LEVEL_FIELD_NAME)
	aver.Always(filter.Level_Field != "", "jlog: level field name is non-empty")
	return filter
}

// Level_Filter_To_Stream forwards each written line whose level clears the Floor and drops the
// rest, submitting the survivors to the inner Stream once.
func Level_Filter_To_Stream(filter *Level_Filter) (stream nbio.Stream) {
	aver.Always(filter != nil, "jlog: level filter must not be nil")
	return nbio.Stream{State: filter, Procedure: level_filter_procedure}
}

func level_filter_procedure(
	state nbio.State, completion *nbio.Completion, mode nbio.Stream_Mode, payload []byte,
	offset int64, whence nbio.Seek_From, callback nbio.Stream_Callback,
) {
	filter := state.(*Level_Filter)
	if mode == nbio.STREAM_MODE_WRITE {
		kept := level_filter_keep(*filter, payload)
		transform_write(filter.Stream, completion, payload, kept, callback)
		return
	}
	transform_submit(filter.Stream, completion, mode, offset, whence, callback)
}

func level_filter_keep(filter Level_Filter, payload []byte) (kept Buffer) {
	kept = make(Buffer, 0, len(payload))
	line_start := 0
	for index := 0; index < len(payload); index++ {
		if payload[index] != '\n' {
			continue
		}
		if filter_admits(filter, payload[line_start:index+1]) {
			kept = append(kept, payload[line_start:index+1]...)
		}
		line_start = index + 1
	}
	// A trailing chunk with no newline is not something jlog produces (every line ends in \n),
	// but a filter on a raw pipe might see one; classify it the same, rather than dropping it.
	if line_start < len(payload) {
		if filter_admits(filter, payload[line_start:]) {
			kept = append(kept, payload[line_start:]...)
		}
	}
	return kept
}

// Reports whether line clears the filter's floor. A line that is not a JSON object, or one with
// no level field, is admitted: the filter suppresses by level and never swallows output it
// cannot classify.
func filter_admits(filter Level_Filter, line []byte) (admit bool) {
	members, parsed := scan_object(line)
	if !parsed {
		return true
	}
	return level_from_wire(member_text(members, filter.Level_Field)) >= filter.Floor
}

// Maps a level's wire name back to its Level, the inverse of Level.String, so a writer can
// compare a rendered line's level to a floor. An unrecognized or absent name maps to LEVEL_NONE,
// which clears any real floor, so a level-less or custom line is never taken for noise.
func level_from_wire(wire string) (level Level) {
	switch wire {
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
// (its Stream and Color), Capture, and Configuration (Clock, Caller). It renders human-readable
// lines to the Console at floor and above and, when Capture is set, tees every level as raw
// JSON to Capture, so one logger drives a quiet console and a full capture at once. It is pure:
// every sink, the clock, and the caller lookup are injected, so the composition root decides
// what they bind to. The logger's own floor is Trace: the one gate runs before any writer, so
// only a Trace floor lets every level reach Capture; floor gates the console branch alone.
func New_Console_Logger(state *Console_Logger, storage []byte, floor Level) (logger Logger) {
	aver.Always(state != nil, "jlog: console logger storage is required")
	state.Filter = Level_Filter{
		Stream:      Console_To_Stream(&state.Console),
		Floor:       floor,
		Level_Field: string(DEFAULT_LEVEL_FIELD_NAME),
	}
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
	// Console renders lines for a human; the caller sets its Stream and Color.
	Console Console
	// Capture receives every raw JSON level; the caller leaves it zero to disable capture.
	Capture nbio.Stream
	// Filter gates the console branch by level.
	Filter Level_Filter
	// Tee fans out to Capture when one is set.
	Tee nbio.Stream_Tee
}
