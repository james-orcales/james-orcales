// Package jlog is a zero-allocation, dependency-injected structured JSON logger.
//
// The house linter bans methods (except those satisfying a stdlib interface) and
// names a free function after its first parameter's type, so there is no fluent
// builder: a log line is one Logger_-prefixed call taking the logger, a message,
// and a variadic list of Field values built by the  constructors.
//
//	logger := jlog.New(jlog.New_Input{Writer: os.Stderr, Floor: jlog.Level_Info})
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
	"sync"
	"unicode/utf8"
	"unsafe"

	jtime "github.com/james-orcales/james-orcales/shared/time"
)

// A fresh line buffer holds a typical event without growing, so steady-state
// logging never reallocates.
const default_buffer_capacity = 500

// Buffers grown past this are dropped from the pool rather than pinning 64KiB per
// slot forever.
const pooled_buffer_capacity_max = 1 << 16

const decimal_base = 10

const default_caller_skip = 0

const float_precision_shortest = -1

const float_bits_32 = 32

const float_bits_64 = 64

// A float of this magnitude or beyond renders in exponent form, matching the
// cutoffs JSON encoders use.
const float_exponent_low = 1e-6
const float_exponent_high = 1e21

// A configuration default is a distinct type so string_or's two parameters never
// repeat a type, which the input-struct rule would otherwise force into a struct.
type default_string string

const default_timestamp_field_name default_string = "time"
const default_level_field_name default_string = "level"
const default_message_field_name default_string = "message"
const default_error_field_name default_string = "error"
const default_caller_field_name default_string = "caller"
const default_stack_field_name default_string = "stack"

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
// against kind_error rather than a switch on the hot value path.
const kind_string Field_Kind = 0
const kind_integer Field_Kind = 1
const kind_unsigned Field_Kind = 2
const kind_float32 Field_Kind = 3
const kind_float64 Field_Kind = 4
const kind_boolean Field_Kind = 5
const kind_bytes Field_Kind = 6
const kind_hexadecimal Field_Kind = 7
const kind_raw_json Field_Kind = 8
const kind_time Field_Kind = 9
const kind_duration Field_Kind = 10
const kind_ip Field_Kind = 11
const kind_mac Field_Kind = 12
const kind_any Field_Kind = 13
const kind_strings Field_Kind = 14
const kind_integers Field_Kind = 15
const kind_floats Field_Kind = 16
const kind_booleans Field_Kind = 17
const kind_durations Field_Kind = 18
const kind_error Field_Kind = 19
const kind_timestamp Field_Kind = 20
const kind_caller Field_Kind = 21

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
	Clock jtime.Clock
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
	// Duration_Unit divides Duration values before rendering; defaults to Nanosecond.
	Duration_Unit jtime.Duration
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
	Clock jtime.Clock
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
	// Duration_Unit divides Duration values; zero uses Nanosecond.
	Duration_Unit jtime.Duration
	// Auto_Timestamp stamps every line automatically.
	Auto_Timestamp bool
	// Auto_Caller adds the caller location to every line automatically.
	Auto_Caller bool
}

// Level is a log severity.
type Level int8

// Level_Trace is the most verbose level.
const Level_Trace Level = -1

// Level_Debug is the debugging level.
const Level_Debug Level = 0

// Level_Info is the informational level.
const Level_Info Level = 1

// Level_Warn is the warning level.
const Level_Warn Level = 2

// Level_Error is the error level.
const Level_Error Level = 3

// Level_None is a line with no severity field, emitted by Logger_Log.
const Level_None Level = 6

// Level_Disabled marks a logger that emits nothing.
const Level_Disabled Level = 7

// String returns the wire name of the level, empty for Level_None.
func (level Level) String() (name string) {
	switch level {
	case Level_Trace:
		return "trace"
	case Level_Debug:
		return "debug"
	case Level_Info:
		return "info"
	case Level_Warn:
		return "warn"
	case Level_Error:
		return "error"
	case Level_Disabled:
		return "disabled"
	case Level_None:
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
		unit = jtime.Nanosecond
	}
	assert(writer != nil, "jlog: writer must not be nil")
	assert(unit > 0, "jlog: duration unit must be positive")
	logger.Configuration = &Logger_Configuration{
		Writer:          writer,
		Clock:           input.Clock,
		Caller:          input.Caller,
		Stack_Marshaler: input.Stack_Marshaler,
		Timestamp_Field_Name: string_or(
			input.Timestamp_Field_Name, default_timestamp_field_name),
		Level_Field_Name: string_or(input.Level_Field_Name, default_level_field_name),
		Message_Field_Name: string_or(
			input.Message_Field_Name, default_message_field_name),
		Error_Field_Name:  string_or(input.Error_Field_Name, default_error_field_name),
		Caller_Field_Name: string_or(input.Caller_Field_Name, default_caller_field_name),
		Stack_Field_Name:  string_or(input.Stack_Field_Name, default_stack_field_name),
		Duration_Unit:     unit,
		Buffer_Pool:       &sync.Pool{New: new_buffer},
	}
	logger.Floor = input.Floor
	logger.Auto_Timestamp = input.Auto_Timestamp
	logger.Auto_Caller = input.Auto_Caller
	return logger
}

func string_or(value string, fallback default_string) (chosen string) {
	if value == "" {
		return string(fallback)
	}
	return value
}

// A *Buffer is pooled rather than a Buffer so Put boxes a pointer, not a header.
func new_buffer() (buffer any) {
	created := make(Buffer, 0, default_buffer_capacity)
	return &created
}

// Logger_Trace emits a trace-level line.
func Logger_Trace(logger Logger, message string, fields ...Field) {
	logger_emit(logger, Level_Trace, message, fields)
}

// Logger_Debug emits a debug-level line.
func Logger_Debug(logger Logger, message string, fields ...Field) {
	logger_emit(logger, Level_Debug, message, fields)
}

// Logger_Info emits an info-level line.
func Logger_Info(logger Logger, message string, fields ...Field) {
	logger_emit(logger, Level_Info, message, fields)
}

// Logger_Warn emits a warn-level line.
func Logger_Warn(logger Logger, message string, fields ...Field) {
	logger_emit(logger, Level_Warn, message, fields)
}

// Logger_Error emits an error-level line.
func Logger_Error(logger Logger, message string, fields ...Field) {
	logger_emit(logger, Level_Error, message, fields)
}

// Logger_Log emits a line with no severity field.
func Logger_Log(logger Logger, message string, fields ...Field) {
	logger_emit(logger, Level_None, message, fields)
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
		seed := Field{Kind: kind_caller, Number: default_caller_skip}
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
	if logger.Floor == Level_Disabled {
		return true
	}
	return false
}

// Logger_With returns a child logger carrying fields as a fixed prefix on every line.
func Logger_With(logger Logger, fields ...Field) (child Logger) {
	child = logger
	prefix := make(Buffer, 0, default_buffer_capacity)
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

type context_key struct{}

// Logger_With_Context returns a copy of parent carrying logger.
func Logger_With_Context(logger Logger, parent context.Context) (child context.Context) {
	return context.WithValue(parent, context_key{}, logger)
}

// From_Context returns the logger carried by ctx, or a disabled no-op logger when
// none is present.
func From_Context(ctx context.Context) (logger Logger) {
	stored, ok := ctx.Value(context_key{}).(Logger)
	if !ok {
		return Logger{}
	}
	return stored
}

// String builds a string field.
func String(key Key, value string) (field Field) {
	return Field{
		Key:    key,
		Kind:   kind_string,
		Data:   unsafe.Pointer(unsafe.StringData(value)),
		Number: int64(len(value)),
	}
}

// Integer builds a signed-integer field.
func Integer(key Key, value int) (field Field) {
	return Field{Key: key, Kind: kind_integer, Number: int64(value)}
}

// Int64 builds a 64-bit signed-integer field.
func Int64(key Key, value int64) (field Field) {
	return Field{Key: key, Kind: kind_integer, Number: value}
}

// Uint builds an unsigned-integer field.
func Uint(key Key, value uint) (field Field) {
	return Field{Key: key, Kind: kind_unsigned, Number: int64(value)}
}

// Uint64 builds a 64-bit unsigned-integer field.
func Uint64(key Key, value uint64) (field Field) {
	return Field{Key: key, Kind: kind_unsigned, Number: int64(value)}
}

// Float32 builds a float field rendered at float32 precision.
func Float32(key Key, value float32) (field Field) {
	return Field{Key: key, Kind: kind_float32, Number: int64(math.Float64bits(float64(value)))}
}

// Float64 builds a float field rendered at float64 precision.
func Float64(key Key, value float64) (field Field) {
	return Field{Key: key, Kind: kind_float64, Number: int64(math.Float64bits(value))}
}

// Boolean builds a boolean field.
func Boolean(key Key, value bool) (field Field) {
	return Field{Key: key, Kind: kind_boolean, Number: boolean_to_int64(value)}
}

// Bytes builds a field whose []byte value is rendered as a JSON string.
func Bytes(key Key, value []byte) (field Field) {
	return Field{
		Key:    key,
		Kind:   kind_bytes,
		Data:   unsafe.Pointer(unsafe.SliceData(value)),
		Number: int64(len(value)),
	}
}

// Hexadecimal builds a field whose []byte value is rendered as a hex string.
func Hexadecimal(key Key, value []byte) (field Field) {
	return Field{
		Key:    key,
		Kind:   kind_hexadecimal,
		Data:   unsafe.Pointer(unsafe.SliceData(value)),
		Number: int64(len(value)),
	}
}

// Raw_JSON builds a field whose pre-encoded JSON value is copied verbatim.
func Raw_JSON(key Key, value []byte) (field Field) {
	return Field{
		Key:    key,
		Kind:   kind_raw_json,
		Data:   unsafe.Pointer(unsafe.SliceData(value)),
		Number: int64(len(value)),
	}
}

// Time builds a field rendering a Moment as an RFC 3339 UTC timestamp string.
func Time(key Key, value jtime.Moment) (field Field) {
	return Field{Key: key, Kind: kind_time, Number: int64(value)}
}

// Duration builds a field rendering a Duration as an integer in the logger unit.
func Duration(key Key, value jtime.Duration) (field Field) {
	return Field{Key: key, Kind: kind_duration, Number: int64(value)}
}

// IP_Address builds a field rendering a net.IP as its string form.
func IP_Address(key Key, value net.IP) (field Field) {
	return Field{
		Key:    key,
		Kind:   kind_ip,
		Data:   unsafe.Pointer(unsafe.SliceData(value)),
		Number: int64(len(value)),
	}
}

// MAC_Address builds a field rendering a net.HardwareAddr as its string form.
func MAC_Address(key Key, value net.HardwareAddr) (field Field) {
	return Field{
		Key:    key,
		Kind:   kind_mac,
		Data:   unsafe.Pointer(unsafe.SliceData(value)),
		Number: int64(len(value)),
	}
}

// Any builds a field whose value is marshaled by encoding/json; it may allocate.
func Any(key Key, value any) (field Field) {
	return Field{Key: key, Kind: kind_any, Boxed: value}
}

// Err builds the error field; its key, and an optional stack, come from the config.
func Err(value error) (field Field) {
	return Field{Kind: kind_error, Boxed: value}
}

// Timestamp builds a field stamping the realtime clock under the configured key.
func Timestamp() (field Field) {
	return Field{Kind: kind_timestamp}
}

// Caller builds a field adding the injected caller location under the configured key.
func Caller() (field Field) {
	return Field{Kind: kind_caller, Number: default_caller_skip}
}

// Strings builds a field rendering a []string as a JSON array.
func Strings(key Key, value []string) (field Field) {
	return Field{
		Key:    key,
		Kind:   kind_strings,
		Data:   unsafe.Pointer(unsafe.SliceData(value)),
		Number: int64(len(value)),
	}
}

// Integers builds a field rendering a []int as a JSON array.
func Integers(key Key, value []int) (field Field) {
	return Field{
		Key:    key,
		Kind:   kind_integers,
		Data:   unsafe.Pointer(unsafe.SliceData(value)),
		Number: int64(len(value)),
	}
}

// Floats64 builds a field rendering a []float64 as a JSON array.
func Floats64(key Key, value []float64) (field Field) {
	return Field{
		Key:    key,
		Kind:   kind_floats,
		Data:   unsafe.Pointer(unsafe.SliceData(value)),
		Number: int64(len(value)),
	}
}

// Booleans builds a field rendering a []bool as a JSON array.
func Booleans(key Key, value []bool) (field Field) {
	return Field{
		Key:    key,
		Kind:   kind_booleans,
		Data:   unsafe.Pointer(unsafe.SliceData(value)),
		Number: int64(len(value)),
	}
}

// Durations builds a field rendering a []Duration as a JSON array.
func Durations(key Key, value []jtime.Duration) (field Field) {
	return Field{
		Key:    key,
		Kind:   kind_durations,
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
	if field.Kind >= kind_error {
		switch field.Kind {
		case kind_error:
			return buffer_append_error(destination, field, configuration)
		case kind_timestamp:
			return buffer_append_realtime(destination, configuration)
		case kind_caller:
			return buffer_append_caller(destination, field, configuration)
		}
		return destination
	}
	destination = buffer_append_key(destination, field.Key)
	count := int(field.Number)
	switch field.Kind {
	case kind_string:
		return buffer_append_string(destination, unsafe.String((*byte)(field.Data), count))
	case kind_integer:
		return buffer_append_int64(destination, field.Number)
	case kind_unsigned:
		return buffer_append_uint64(destination, uint64(field.Number))
	case kind_float32:
		bits := math.Float64frombits(uint64(field.Number))
		return buffer_append_float(destination, bits, float_bits_32)
	case kind_float64:
		bits := math.Float64frombits(uint64(field.Number))
		return buffer_append_float(destination, bits, float_bits_64)
	case kind_boolean:
		return buffer_append_boolean(destination, field.Number == 1)
	case kind_bytes:
		return buffer_append_bytes(destination, unsafe.Slice((*byte)(field.Data), count))
	case kind_hexadecimal:
		raw := unsafe.Slice((*byte)(field.Data), count)
		return buffer_append_hexadecimal(destination, raw)
	case kind_raw_json:
		return append(destination, unsafe.Slice((*byte)(field.Data), count)...)
	case kind_time:
		return buffer_append_rfc3339(destination, field.Number)
	case kind_duration:
		divided := field.Number / int64(configuration.Duration_Unit)
		return buffer_append_int64(destination, divided)
	case kind_ip:
		address := net.IP(unsafe.Slice((*byte)(field.Data), count))
		return buffer_append_string(destination, address.String())
	case kind_mac:
		address := net.HardwareAddr(unsafe.Slice((*byte)(field.Data), count))
		return buffer_append_string(destination, address.String())
	case kind_any:
		return buffer_append_any(destination, field.Boxed)
	case kind_strings:
		values := unsafe.Slice((*string)(field.Data), count)
		return buffer_append_strings(destination, values)
	case kind_integers:
		return buffer_append_integers(destination, unsafe.Slice((*int)(field.Data), count))
	case kind_floats:
		values := unsafe.Slice((*float64)(field.Data), count)
		return buffer_append_floats(destination, values)
	case kind_booleans:
		return buffer_append_booleans(destination, unsafe.Slice((*bool)(field.Data), count))
	case kind_durations:
		values := unsafe.Slice((*jtime.Duration)(field.Data), count)
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
		destination = buffer_append_float(destination, values[index], float_bits_64)
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
	destination Buffer, values []jtime.Duration, unit jtime.Duration,
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
	return strconv.AppendInt(destination, value, decimal_base)
}

func buffer_append_uint64(destination Buffer, value uint64) (output Buffer) {
	return strconv.AppendUint(destination, value, decimal_base)
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
	const ns_per_second = 1_000_000_000
	const seconds_per_day = 86400
	const seconds_per_hour = 3600
	const seconds_per_minute = 60
	seconds := nanos / ns_per_second
	fraction := nanos % ns_per_second
	if fraction < 0 {
		fraction += ns_per_second
		seconds--
	}
	days := seconds / seconds_per_day
	day_seconds := seconds % seconds_per_day
	if day_seconds < 0 {
		day_seconds += seconds_per_day
		days--
	}
	assert(day_seconds >= 0 && day_seconds < seconds_per_day, "jlog: bad day seconds")
	year, month, day := civil_from_days(days)
	assert(month >= 1 && month <= 12, "jlog: civil month out of range")
	assert(day >= 1 && day <= 31, "jlog: civil day out of range")
	within_hour := day_seconds % seconds_per_hour
	destination = append(destination, '"')
	destination = buffer_append_four_digits(destination, year)
	destination = append(destination, '-')
	destination = buffer_append_two_digits(destination, int64(month))
	destination = append(destination, '-')
	destination = buffer_append_two_digits(destination, int64(day))
	destination = append(destination, 'T')
	destination = buffer_append_two_digits(destination, day_seconds/seconds_per_hour)
	destination = append(destination, ':')
	destination = buffer_append_two_digits(destination, within_hour/seconds_per_minute)
	destination = append(destination, ':')
	destination = buffer_append_two_digits(destination, within_hour%seconds_per_minute)
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
	const days_shift = 719468
	const days_per_era = 146097
	z := days + days_shift
	era := z
	if era < 0 {
		era -= days_per_era - 1
	}
	era /= days_per_era
	day_of_era := z - era*days_per_era
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
	if cap(final) > pooled_buffer_capacity_max {
		return
	}
	*holder = final[:0]
	pool.Put(holder)
}

func buffer_append_level(
	destination Buffer, level Level, configuration *Logger_Configuration,
) (output Buffer) {
	if level == Level_None {
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
		destination, value, format, float_precision_shortest, bit_size)
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
	if bit_size == float_bits_64 {
		if magnitude < float_exponent_low {
			return true
		}
		if magnitude >= float_exponent_high {
			return true
		}
		return false
	}
	if float32(magnitude) < float_exponent_low {
		return true
	}
	if float32(magnitude) >= float_exponent_high {
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
	const table = "\x00\x00\x00\x00\xfb\xff\xff\xff\xff\xff\xff\xef\xff\xff\xff\x7f" +
		"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"
	return table[value>>3]&(byte(1)<<(value&7)) != 0
}

func hexadecimal_digit(nibble byte) (digit byte) {
	const hexadecimal_digits = "0123456789abcdef"
	return hexadecimal_digits[nibble]
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
