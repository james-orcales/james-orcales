// Package jlog is a zero-allocation, dependency-injected structured JSON logger.
//
// The house linter bans methods (except those satisfying a stdlib interface) and
// names a free function after its first parameter's type, so there is no fluent
// builder: a log line is one Logger_-prefixed call taking the logger, a message,
// and a variadic list of Field values built by the  constructors.
//
//	logger := jlog.New(jlog.New_Input{Writer: os.Stderr, Floor: jlog.LEVEL_INFO})
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
// The hot path (scalar fields to a ready writer) is allocation-free: the line is
// assembled in a pooled buffer and the variadic Field slice stays on the caller's
// stack. The net, Any, and Raw_JSON paths may allocate via stdlib formatting.
package jlog

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
	"unsafe"

	"local/james-orcales/shared/time"
)

// A fresh line buffer holds a typical event without growing, so steady-state
// logging never reallocates.
const DEFAULT_BUFFER_CAPACITY = 500

// Buffers grown past this are dropped from the pool rather than pinning 64KiB per
// slot forever.
const POOLED_BUFFER_CAPACITY_MAX = 1 << 16

// DECIMAL_BASE renders integers in base 10 so JSON consumers read them as plain decimals.
const DECIMAL_BASE = 10

// DEFAULT_CALLER_SKIP passes zero frames because the injected Caller owns its own skip offset.
const DEFAULT_CALLER_SKIP = 0

// FLOAT_PRECISION_SHORTEST asks strconv for the fewest digits that round-trip exactly.
const FLOAT_PRECISION_SHORTEST = -1

// FLOAT_BITS_32 formats a Float32 at 32-bit precision so it round-trips as a float32.
const FLOAT_BITS_32 = 32

// FLOAT_BITS_64 formats a Float64 at full 64-bit precision, the widest JSON numbers carry.
const FLOAT_BITS_64 = 64

// A float of this magnitude or beyond renders in exponent form, matching the
// cutoffs JSON encoders use.
const FLOAT_EXPONENT_LOW = 1e-6

// FLOAT_EXPONENT_HIGH is the upper cutoff; at or above it, magnitudes render in exponent form.
const FLOAT_EXPONENT_HIGH = 1e21

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

// KIND_IP tags a net.IP behind Data, emitted as its dotted or colon string form.
const KIND_IP Field_Kind = 11

// KIND_MAC tags a net.HardwareAddr behind Data, emitted as its colon-separated string.
const KIND_MAC Field_Kind = 12

// KIND_ANY tags a value in Boxed marshaled by encoding/json; this path may allocate.
const KIND_ANY Field_Kind = 13

// KIND_STRINGS tags a []string behind Data, emitted as a JSON array of strings.
const KIND_STRINGS Field_Kind = 14

// KIND_INTEGERS tags a []int behind Data, emitted as a JSON array of numbers.
const KIND_INTEGERS Field_Kind = 15

// KIND_FLOATS tags a []float64 behind Data, emitted as a JSON array of numbers.
const KIND_FLOATS Field_Kind = 16

// KIND_BOOLEANS tags a []bool behind Data, emitted as a JSON array of booleans.
const KIND_BOOLEANS Field_Kind = 17

// KIND_DURATIONS tags a []Duration behind Data, each divided by the config unit.
const KIND_DURATIONS Field_Kind = 18

// KIND_ERROR keys from config, not field.Key; it emits the boxed error and optional stack.
const KIND_ERROR Field_Kind = 19

// KIND_TIMESTAMP keys from config; it stamps the realtime clock, carrying no value of its own.
const KIND_TIMESTAMP Field_Kind = 20

// KIND_CALLER keys from config; Number holds the frame skip for the injected Caller lookup.
const KIND_CALLER Field_Kind = 21

// Field is one structured key/value pair, built by a  constructor and consumed
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
	// Boxed holds the error for an Err field or the value for an Any field.
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
	// Writer receives each finished line in a single Write call.
	Writer io.Writer
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
	// Buffer_Pool recycles line buffers so emitting stays allocation-free.
	Buffer_Pool *sync.Pool
}

// Logger bundles a shared configuration with this logger's level floor and fixed
// prefix fields. It is passed by value; the zero value is a disabled no-op.
type Logger struct {
	// Configuration is the shared dependency set; nil marks a disabled logger.
	Configuration *Logger_Configuration
	// Floor is the lowest level emitted; lower levels are dropped.
	Floor Level
	// Prefix is the pre-encoded fixed fields ({...} fragment) added to every line.
	Prefix Buffer
	// Auto_Timestamp stamps every line with the realtime clock when set.
	Auto_Timestamp bool
	// Auto_Caller adds the caller location to every line when set.
	Auto_Caller bool
}

// New_Input configures New. Writer and Clock are the meaningful dependencies; the
// rest take sensible defaults when left zero.
type New_Input struct {
	// Writer receives finished lines; nil becomes io.Discard.
	Writer io.Writer
	// Clock supplies Timestamp and Auto_Timestamp readings.
	Clock time.Clock
	// Floor is the lowest emitted level.
	Floor Level
	// Caller maps a frame-skip count to a location string; nil disables Caller.
	Caller func(skip int) (location string)
	// Stack_Marshaler renders an error's stack; nil disables Err stacks.
	Stack_Marshaler func(value error) (stack string)
	// Timestamp_Field_Name overrides the timestamp key; empty uses the default.
	Timestamp_Field_Name string
	// Level_Field_Name overrides the level key; empty uses the default.
	Level_Field_Name string
	// Message_Field_Name overrides the message key; empty uses the default.
	Message_Field_Name string
	// Error_Field_Name overrides the error key; empty uses the default.
	Error_Field_Name string
	// Caller_Field_Name overrides the caller key; empty uses the default.
	Caller_Field_Name string
	// Stack_Field_Name overrides the stack key; empty uses the default.
	Stack_Field_Name string
	// Duration_Unit divides Duration values; zero uses NANOSECOND.
	Duration_Unit time.Duration
	// Auto_Timestamp stamps every line automatically.
	Auto_Timestamp bool
	// Auto_Caller adds the caller location to every line automatically.
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

// String returns the wire name of the level, empty for LEVEL_NONE.
func (level Level) String() (name string) {
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
	return strconv.Itoa(int(level))
}

// New builds a logger from input, allocating its shared configuration and pool.
func New(input New_Input) (logger Logger) {
	writer := input.Writer
	if writer == nil {
		writer = io.Discard
	}
	unit := input.Duration_Unit
	if unit == 0 {
		unit = time.NANOSECOND
	}
	assert(writer != nil, "jlog: writer must not be nil")
	assert(unit > 0, "jlog: duration unit must be positive")
	logger.Configuration = &Logger_Configuration{
		Writer:          writer,
		Clock:           input.Clock,
		Caller:          input.Caller,
		Stack_Marshaler: input.Stack_Marshaler,
		Timestamp_Field_Name: string_or(
			input.Timestamp_Field_Name, DEFAULT_TIMESTAMP_FIELD_NAME),
		Level_Field_Name: string_or(input.Level_Field_Name, DEFAULT_LEVEL_FIELD_NAME),
		Message_Field_Name: string_or(
			input.Message_Field_Name, DEFAULT_MESSAGE_FIELD_NAME),
		Error_Field_Name:  string_or(input.Error_Field_Name, DEFAULT_ERROR_FIELD_NAME),
		Caller_Field_Name: string_or(input.Caller_Field_Name, DEFAULT_CALLER_FIELD_NAME),
		Stack_Field_Name:  string_or(input.Stack_Field_Name, DEFAULT_STACK_FIELD_NAME),
		Duration_Unit:     unit,
		Buffer_Pool:       &sync.Pool{New: new_buffer},
	}
	logger.Floor = input.Floor
	logger.Auto_Timestamp = input.Auto_Timestamp
	logger.Auto_Caller = input.Auto_Caller
	return logger
}

func string_or(value string, fallback Default_String) (chosen string) {
	if value == "" {
		return string(fallback)
	}
	return value
}

// A *Buffer is pooled rather than a Buffer so Put boxes a pointer, not a header.
func new_buffer() (buffer any) {
	created := make(Buffer, 0, DEFAULT_BUFFER_CAPACITY)
	return &created
}

// Logger_Trace emits a trace-level line.
func Logger_Trace(logger Logger, message string, fields ...Field) {
	logger_emit(logger, LEVEL_TRACE, message, fields)
}

// Logger_Debug emits a debug-level line.
func Logger_Debug(logger Logger, message string, fields ...Field) {
	logger_emit(logger, LEVEL_DEBUG, message, fields)
}

// Logger_Info emits an info-level line.
func Logger_Info(logger Logger, message string, fields ...Field) {
	logger_emit(logger, LEVEL_INFO, message, fields)
}

// Logger_Warn emits a warn-level line.
func Logger_Warn(logger Logger, message string, fields ...Field) {
	logger_emit(logger, LEVEL_WARN, message, fields)
}

// Logger_Error emits an error-level line.
func Logger_Error(logger Logger, message string, fields ...Field) {
	logger_emit(logger, LEVEL_ERROR, message, fields)
}

// Logger_Log emits a line with no severity field.
func Logger_Log(logger Logger, message string, fields ...Field) {
	logger_emit(logger, LEVEL_NONE, message, fields)
}

// Logger_At_Level emits a line at the given level.
func Logger_At_Level(logger Logger, level Level, message string, fields ...Field) {
	logger_emit(logger, level, message, fields)
}

func logger_emit(logger Logger, level Level, message string, fields []Field) {
	if logger_is_disabled(logger) {
		return
	}
	if level < logger.Floor {
		return
	}
	configuration := logger.Configuration
	holder := configuration.Buffer_Pool.Get().(*Buffer)
	buffer := *holder
	buffer = buffer[:0]
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
	assert(buffer[0] == '{', "jlog: line must open with a brace")
	assert(buffer[len(buffer)-2] == '}', "jlog: line must close with a brace")
	assert(buffer[len(buffer)-1] == '\n', "jlog: line must end with a newline")
	configuration.Writer.Write(buffer)
	buffer_recycle(holder, configuration.Buffer_Pool, buffer)
}

func logger_is_disabled(logger Logger) (disabled bool) {
	if logger.Configuration == nil {
		return true
	}
	if logger.Floor == LEVEL_DISABLED {
		return true
	}
	return false
}

// Logger_With returns a child logger carrying fields as a fixed prefix on every line.
func Logger_With(logger Logger, fields ...Field) (child Logger) {
	child = logger
	prefix := make(Buffer, 0, DEFAULT_BUFFER_CAPACITY)
	if len(logger.Prefix) > 0 {
		prefix = append(prefix, logger.Prefix...)
	} else {
		prefix = buffer_append_begin_marker(prefix)
	}
	for index := 0; index < len(fields); index++ {
		prefix = buffer_encode_field(prefix, &fields[index], logger.Configuration)
	}
	child.Prefix = prefix
	return child
}

// Context_Key is the distinct key type a Logger is stored under in a context.Context,
// so the value cannot collide with keys of any other package.
type Context_Key struct{}

// Logger_With_Context returns a copy of parent carrying logger.
func Logger_With_Context(logger Logger, parent context.Context) (child context.Context) {
	return context.WithValue(parent, Context_Key{}, logger)
}

// From_Context returns the logger carried by ctx, or a disabled no-op logger when
// none is present.
func From_Context(ctx context.Context) (logger Logger) {
	stored, ok := ctx.Value(Context_Key{}).(Logger)
	if !ok {
		return Logger{}
	}
	return stored
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
	return Field{Key: key, Kind: KIND_FLOAT32, Number: int64(math.Float64bits(float64(value)))}
}

// Float64 builds a float field rendered at float64 precision.
func Float64[T ~float64](key Key, value T) (field Field) {
	return Field{Key: key, Kind: KIND_FLOAT64, Number: int64(math.Float64bits(float64(value)))}
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

// IP_Address builds a field rendering a net.IP as its string form.
func IP_Address(key Key, value net.IP) (field Field) {
	return Field{
		Key:    key,
		Kind:   KIND_IP,
		Data:   unsafe.Pointer(unsafe.SliceData(value)),
		Number: int64(len(value)),
	}
}

// MAC_Address builds a field rendering a net.HardwareAddr as its string form.
func MAC_Address(key Key, value net.HardwareAddr) (field Field) {
	return Field{
		Key:    key,
		Kind:   KIND_MAC,
		Data:   unsafe.Pointer(unsafe.SliceData(value)),
		Number: int64(len(value)),
	}
}

// Any builds a field whose value is marshaled by encoding/json; it may allocate.
func Any(key Key, value any) (field Field) {
	return Field{Key: key, Kind: KIND_ANY, Boxed: value}
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
		bits := math.Float64frombits(uint64(field.Number))
		return buffer_append_float(destination, bits, FLOAT_BITS_32)
	case KIND_FLOAT64:
		bits := math.Float64frombits(uint64(field.Number))
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
	case KIND_IP:
		address := net.IP(unsafe.Slice((*byte)(field.Data), count))
		return buffer_append_string(destination, address.String())
	case KIND_MAC:
		address := net.HardwareAddr(unsafe.Slice((*byte)(field.Data), count))
		return buffer_append_string(destination, address.String())
	case KIND_ANY:
		return buffer_append_any(destination, field.Boxed)
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
	return buffer_append_rfc3339(destination, int64(configuration.Clock.Now_Realtime()))
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

func buffer_append_any(destination Buffer, value any) (output Buffer) {
	encoded, marshal_err := json.Marshal(value)
	if marshal_err != nil {
		return buffer_append_string(destination, "json marshal error")
	}
	return append(destination, encoded...)
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
	return strconv.AppendInt(destination, value, DECIMAL_BASE)
}

func buffer_append_uint64(destination Buffer, value uint64) (output Buffer) {
	return strconv.AppendUint(destination, value, DECIMAL_BASE)
}

func buffer_append_boolean(destination Buffer, value bool) (output Buffer) {
	return strconv.AppendBool(destination, value)
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
	assert(day_seconds >= 0 && day_seconds < SECONDS_PER_DAY, "jlog: bad day seconds")
	year, month, day := civil_from_days(days)
	assert(month >= 1 && month <= 12, "jlog: civil month out of range")
	assert(day >= 1 && day <= 31, "jlog: civil day out of range")
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

func buffer_recycle(holder *Buffer, pool *sync.Pool, final Buffer) {
	if cap(final) > POOLED_BUFFER_CAPACITY_MAX {
		return
	}
	*holder = final[:0]
	pool.Put(holder)
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
	return buffer_append_string(destination, level.String())
}

// JSON has no NaN/Inf, so those render as strings; a leading exponent zero is
// trimmed to match es6 number output.
func buffer_append_float(destination Buffer, value float64, bit_size int) (output Buffer) {
	if math.IsNaN(value) {
		return append(destination, '"', 'N', 'a', 'N', '"')
	}
	if math.IsInf(value, 1) {
		return append(destination, '"', '+', 'I', 'n', 'f', '"')
	}
	if math.IsInf(value, -1) {
		return append(destination, '"', '-', 'I', 'n', 'f', '"')
	}
	format := byte('f')
	if float_needs_exponent(value, bit_size) {
		format = 'e'
	}
	destination = strconv.AppendFloat(
		destination, value, format, FLOAT_PRECISION_SHORTEST, bit_size)
	if format == 'e' {
		destination = buffer_clean_exponent(destination)
	}
	return destination
}

func float_needs_exponent(value float64, bit_size int) (needs bool) {
	magnitude := math.Abs(value)
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
		if current >= utf8.RuneSelf {
			decoded, width := utf8.DecodeRuneInString(value[index:])
			if decoded == utf8.RuneError {
				if width == 1 {
					destination = append(destination, value[run_start:index]...)
					destination = buffer_append_replacement(destination)
					index += width
					run_start = index
					continue
				}
			}
			index += width
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
		if current >= utf8.RuneSelf {
			decoded, width := utf8.DecodeRune(value[index:])
			if decoded == utf8.RuneError {
				if width == 1 {
					destination = append(destination, value[run_start:index]...)
					destination = buffer_append_replacement(destination)
					index += width
					run_start = index
					continue
				}
			}
			index += width
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

// Panics with message when condition is false. A cheap always-on invariant check: it
// inlines to a single predictable branch over a constant string and captures no caller
// site, so it stays off the allocation budget the hot path depends on. The panic's stack
// trace carries the location; message names the invariant.
func assert(condition bool, message string) {
	if !condition {
		panic(message)
	}
}

// The rest of this file is the pretty printer: the human-facing inverse of the encoder above. jlog
// emits flat JSON for machines; a Console reads those lines back and renders them for a human at a
// terminal. It plugs into the encoder only through the io.Writer seam (each emit writes one whole
// flat-JSON object per Write), so the zero-allocation hot path is never touched. Being off the hot
// path, it may allocate freely. It is pure — it reaches for no OS, clock, or global state; every
// dependency arrives through a Console field — so it belongs here, not in the composition tier.

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

// Console is an io.Writer that renders each flat-JSON jlog line as a human-readable console line.
// It reads jlog's default field names (time, level, message, error); a logger built with custom
// field names is not matched, which no caller needs. It is passed by value and never mutated, so a
// copy is a faithful, independent Console. Build one directly with a struct literal.
type Console struct {
	// Writer receives the rendered text; a caller must set it (a zero Console writes nowhere).
	Writer io.Writer
	// Color enables ANSI color in the rendered output.
	Color bool
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
	return buffer_paint(destination, value_color, console.Color, logfmt_token(field))
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
func logfmt_token(field Member) (token string) {
	if field.Kind != VALUE_IS_STRING {
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

// Level_Filter is an io.Writer that forwards only the jlog lines whose level is at or above
// its Floor and drops the rest — a floor at the writer seam, not at the emit gate. A terminal
// logger emits every level so a file sink keeps all of them; a Level_Filter is what still holds
// trace and debug off the console. It is a drop-in jlog writer that composes under io.MultiWriter.
// It is passed by value; New_Level_Filter sets its fields and nothing mutates them, so a copy is a
// faithful, independent filter.
type Level_Filter struct {
	// Writer receives the lines that clear the floor.
	Writer io.Writer
	// Floor is the lowest level forwarded; a line below it is dropped.
	Floor Level
	// Level_Field is the key whose value names each line's level.
	Level_Field string
}

// New_Level_Filter_Input configures New_Level_Filter.
type New_Level_Filter_Input struct {
	// Writer receives the lines that clear the floor; nil becomes io.Discard.
	Writer io.Writer
	// Floor is the lowest level forwarded; a line below it is dropped.
	Floor Level
	// Level_Field_Name overrides the level key; empty uses the default.
	Level_Field_Name string
}

// New_Level_Filter builds a Level_Filter from input, resolving an empty level field name to jlog's
// default so it matches a default logger.
func New_Level_Filter(input New_Level_Filter_Input) (filter Level_Filter) {
	writer := input.Writer
	if writer == nil {
		writer = io.Discard
	}
	filter.Writer = writer
	filter.Floor = input.Floor
	filter.Level_Field = string_or(input.Level_Field_Name, DEFAULT_LEVEL_FIELD_NAME)
	assert(filter.Level_Field != "", "jlog: level field name is non-empty")
	return filter
}

// Write forwards each newline-terminated line in payload whose level clears the Floor and drops the
// rest, writing the survivors to the destination in one Write. It reports len(payload) consumed on
// success — like Console, a filtering writer emits a different byte count than it takes — so both
// jlog's own length check and io.MultiWriter's short-write check are satisfied.
func (filter Level_Filter) Write(payload []byte) (written int, err error) {
	kept := make(Buffer, 0, len(payload))
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
	if len(kept) > 0 {
		if _, write_err := filter.Writer.Write(kept); write_err != nil {
			return 0, write_err
		}
	}
	return len(payload), nil
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

// New_Console_Logger_Input configures New_Console_Logger.
type New_Console_Logger_Input struct {
	// Console receives the rendered lines at Floor and above; nil becomes io.Discard.
	Console io.Writer
	// Capture receives every level as raw JSON, whatever the Floor; nil disables capture.
	Capture io.Writer
	// Color enables ANSI color on the console.
	Color bool
	// Floor is the console's lowest shown level; Capture keeps every level regardless.
	Floor Level
	// Clock stamps each line.
	Clock time.Clock
	// Caller resolves the Caller() field's location.
	Caller func(skip int) (location string)
}

// New_Console_Logger builds a console logger. It renders human-readable lines to the Console sink
// at Floor and above and, when Capture is set, tees every level as raw JSON to Capture, so one
// logger drives a quiet console and a full capture at once. It is pure: the console sink, the
// capture sink, the clock, and the caller lookup are all injected, so the composition root decides
// what they bind to. The logger's own floor is Trace: the one gate runs before any writer, so only
// a Trace floor lets every level reach Capture; the console Floor gates its branch alone.
func New_Console_Logger(input New_Console_Logger_Input) (logger Logger) {
	console_writer := input.Console
	if console_writer == nil {
		console_writer = io.Discard
	}
	console := New_Level_Filter(New_Level_Filter_Input{
		Floor:  input.Floor,
		Writer: Console{Writer: console_writer, Color: input.Color},
	})
	var writer io.Writer = console
	if input.Capture != nil {
		// Capture precedes the console, so a line is kept even if the console drops it.
		writer = io.MultiWriter(input.Capture, console)
	}
	return New(New_Input{
		Writer:         writer,
		Clock:          input.Clock,
		Floor:          LEVEL_TRACE,
		Auto_Timestamp: true,
		Caller:         input.Caller,
	})
}
