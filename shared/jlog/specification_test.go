package jlog_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net"
	"testing"

	"github.com/james-orcales/james-orcales/shared/jlog"
	jtime "github.com/james-orcales/james-orcales/shared/time"
)

// Test_Message_Renders_Level_And_Message covers the minimal line: a level field
// then the trailing message field, terminated by a newline.
func Test_Message_Renders_Level_And_Message(t *testing.T) {
	buffer := &bytes.Buffer{}
	jlog.Logger_Info(new_logger(buffer), "hello")
	assert_output(t, buffer, "{\"level\":\"info\",\"message\":\"hello\"}\n")
}

// Test_Empty_Message_Is_Omitted covers an empty message producing no message field.
func Test_Empty_Message_Is_Omitted(t *testing.T) {
	buffer := &bytes.Buffer{}
	jlog.Logger_Info(new_logger(buffer), "", jlog.Integer("n", 42))
	assert_output(t, buffer, "{\"level\":\"info\",\"n\":42}\n")
}

// Test_Scalar_Fields covers the scalar encoders and that the message is rendered
// after the fields.
func Test_Scalar_Fields(t *testing.T) {
	buffer := &bytes.Buffer{}
	jlog.Logger_Info(new_logger(buffer), "done",
		jlog.String("s", "v"),
		jlog.Boolean("b", true),
		jlog.Integer("i", -7),
		jlog.Uint64("u", 9),
		jlog.Float64("f", 1.5),
	)
	assert_output(t, buffer,
		"{\"level\":\"info\",\"s\":\"v\",\"b\":true,\"i\":-7,\"u\":9,"+
			"\"f\":1.5,\"message\":\"done\"}\n")
}

// Test_String_Escape_Sequences covers JSON escaping of control and quote characters.
func Test_String_Escape_Sequences(t *testing.T) {
	buffer := &bytes.Buffer{}
	jlog.Logger_Info(new_logger(buffer), "", jlog.String("s", "a\"b\nc"))
	assert_output(t, buffer, "{\"level\":\"info\",\"s\":\"a\\\"b\\nc\"}\n")
}

// Test_Bytes_Hexadecimal_Raw_JSON covers the []byte-valued field encoders.
func Test_Bytes_Hexadecimal_Raw_JSON(t *testing.T) {
	buffer := &bytes.Buffer{}
	jlog.Logger_Info(new_logger(buffer), "",
		jlog.Bytes("by", []byte("ab")),
		jlog.Hexadecimal("hx", []byte{0xab, 0xcd}),
		jlog.Raw_JSON("rj", []byte("{\"x\":1}")),
	)
	assert_output(t, buffer,
		"{\"level\":\"info\",\"by\":\"ab\",\"hx\":\"abcd\",\"rj\":{\"x\":1}}\n")
}

// Test_Timestamp_Uses_Injected_Clock covers the Timestamp field reading the
// injected clock and rendering it as integer nanoseconds since the epoch.
func Test_Timestamp_Uses_Injected_Clock(t *testing.T) {
	buffer := &bytes.Buffer{}
	jlog.Logger_Info(new_logger(buffer), "", jlog.Timestamp())
	assert_output(t, buffer, "{\"level\":\"info\",\"time\":\"2023-11-14T22:13:20Z\"}\n")
}

// Test_Auto_Timestamp covers New_Input.Auto_Timestamp stamping every line.
func Test_Auto_Timestamp(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger := jlog.New(jlog.New_Input{
		Writer:         buffer,
		Clock:          frozen_clock(),
		Floor:          jlog.Level_Trace,
		Auto_Timestamp: true,
	})
	jlog.Logger_Info(logger, "")
	assert_output(t, buffer, "{\"level\":\"info\",\"time\":\"2023-11-14T22:13:20Z\"}\n")
}

// Test_Time_And_Duration covers the shared/time value encoders.
func Test_Time_And_Duration(t *testing.T) {
	buffer := &bytes.Buffer{}
	jlog.Logger_Info(new_logger(buffer), "",
		jlog.Time("t", jtime.Moment(5)),
		jlog.Duration("d", jtime.Second),
	)
	assert_output(t, buffer,
		"{\"level\":\"info\",\"t\":\"1970-01-01T00:00:00.000000005Z\",\"d\":1000000000}\n")
}

// Test_Network_Fields covers the net.IP and net.HardwareAddr encoders.
func Test_Network_Fields(t *testing.T) {
	buffer := &bytes.Buffer{}
	jlog.Logger_Info(new_logger(buffer), "",
		jlog.IP_Address("ip", net.IPv4(127, 0, 0, 1)),
		jlog.MAC_Address("mac", net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0x00, 0x01}),
	)
	assert_output(t, buffer,
		"{\"level\":\"info\",\"ip\":\"127.0.0.1\",\"mac\":\"de:ad:be:ef:00:01\"}\n")
}

// Test_Scalar_Arrays covers typed slice fields rendering as flat JSON arrays.
func Test_Scalar_Arrays(t *testing.T) {
	buffer := &bytes.Buffer{}
	jlog.Logger_Info(new_logger(buffer), "",
		jlog.Strings("tags", []string{"a", "b"}),
		jlog.Integers("ids", []int{1, 2, 3}),
	)
	assert_output(t, buffer,
		"{\"level\":\"info\",\"tags\":[\"a\",\"b\"],\"ids\":[1,2,3]}\n")
}

// Test_Err_With_Stack covers Err appending the injected stack rendering before the
// error string when a stack marshaler is configured.
func Test_Err_With_Stack(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger := jlog.New(jlog.New_Input{
		Writer:          buffer,
		Clock:           frozen_clock(),
		Floor:           jlog.Level_Trace,
		Stack_Marshaler: func(value error) (stack string) { return "TRACE" },
	})
	jlog.Logger_Error(logger, "", jlog.Err(errors.New("boom")))
	assert_output(t, buffer,
		"{\"level\":\"error\",\"stack\":\"TRACE\",\"error\":\"boom\"}\n")
}

// Test_Err_Without_Stack covers Err omitting the stack when no marshaler is set.
func Test_Err_Without_Stack(t *testing.T) {
	buffer := &bytes.Buffer{}
	jlog.Logger_Error(new_logger(buffer), "", jlog.Err(errors.New("boom")))
	assert_output(t, buffer, "{\"level\":\"error\",\"error\":\"boom\"}\n")
}

// Test_Caller_Uses_Injected_Function covers the Caller field appending the injected
// location.
func Test_Caller_Uses_Injected_Function(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger := jlog.New(jlog.New_Input{
		Writer: buffer,
		Clock:  frozen_clock(),
		Floor:  jlog.Level_Trace,
		Caller: func(skip int) (location string) { return "file.go:10" },
	})
	jlog.Logger_Info(logger, "", jlog.Caller())
	assert_output(t, buffer, "{\"level\":\"info\",\"caller\":\"file.go:10\"}\n")
}

// Test_Child_Logger_Carries_Context covers Logger_With building a child whose fixed
// fields precede each line's own fields.
func Test_Child_Logger_Carries_Context(t *testing.T) {
	buffer := &bytes.Buffer{}
	child := jlog.Logger_With(new_logger(buffer), jlog.String("component", "auth"))
	jlog.Logger_Info(child, "in", jlog.String("user", "bob"))
	assert_output(t, buffer,
		"{\"level\":\"info\",\"component\":\"auth\",\"user\":\"bob\",\"message\":\"in\"}\n")
}

// Test_Level_Floor_Filters covers a line below the floor producing no output.
func Test_Level_Floor_Filters(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger := jlog.New(jlog.New_Input{
		Writer: buffer,
		Clock:  frozen_clock(),
		Floor:  jlog.Level_Info,
	})
	jlog.Logger_Debug(logger, "dropped", jlog.String("k", "v"))
	assert_output(t, buffer, "")
}

// Test_From_Context_Round_Trips covers carrying a logger through a context.Context
// and recovering it.
func Test_From_Context_Round_Trips(t *testing.T) {
	buffer := &bytes.Buffer{}
	carrier := jlog.Logger_With_Context(new_logger(buffer), t.Context())
	jlog.Logger_Info(jlog.From_Context(carrier), "via ctx")
	assert_output(t, buffer, "{\"level\":\"info\",\"message\":\"via ctx\"}\n")
}

// Test_From_Context_Missing_Is_Disabled covers the empty-context path returning a
// no-op logger rather than panicking.
func Test_From_Context_Missing_Is_Disabled(t *testing.T) {
	buffer := &bytes.Buffer{}
	jlog.Logger_Info(jlog.From_Context(t.Context()), "nope", jlog.String("k", "v"))
	assert_output(t, buffer, "")
}

// Test_Hot_Path_Is_Zero_Allocation is the load-bearing guarantee: a steady-state
// log call to a discarding writer must not allocate.
func Test_Hot_Path_Is_Zero_Allocation(t *testing.T) {
	logger := jlog.New(jlog.New_Input{
		Writer: io.Discard,
		Clock:  frozen_clock(),
		Floor:  jlog.Level_Trace,
	})
	allocations := testing.AllocsPerRun(1000, func() {
		jlog.Logger_Info(logger, "done",
			jlog.String("user", "bob"),
			jlog.Integer("count", 7),
			jlog.Boolean("ok", true),
		)
	})
	if allocations != 0 {
		t.Fatalf("hot path allocated %.1f times per call; want 0", allocations)
	}
}

// A clock whose realtime reading is always fixed_moment.
func frozen_clock() (clock jtime.Clock) {
	return jtime.Clock{Now_Realtime: func() (moment jtime.Moment) { return fixed_moment }}
}

// A logger writing JSON lines to buffer with the frozen clock and the lowest level
// floor, so every test line is emitted and timestamps are fixed.
func new_logger(buffer io.Writer) (logger jlog.Logger) {
	return jlog.New(jlog.New_Input{
		Writer: buffer,
		Clock:  frozen_clock(),
		Floor:  jlog.Level_Trace,
	})
}

// Fails the test unless buffer holds exactly want.
func assert_output(t *testing.T, buffer *bytes.Buffer, want string) {
	t.Helper()
	got := buffer.String()
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

// Every test clock returns this fixed reading so timestamp output is deterministic
// and assertable byte-for-byte.
const fixed_moment jtime.Moment = 1700000000000000000

// The exact benchmark message zerolog uses, so the benchmarks below are a 1:1
// workload comparison against zerolog's same-named benchmarks.
const fake_message = "Test logging, but use a somewhat realistic message length."

// A logger discarding output with the frozen clock, for benchmarks.
func discard_logger() (logger jlog.Logger) {
	return jlog.New(jlog.New_Input{
		Writer: io.Discard,
		Clock:  frozen_clock(),
		Floor:  jlog.Level_Trace,
	})
}

// Benchmark_Log_Empty mirrors zerolog's BenchmarkLogEmpty.
func Benchmark_Log_Empty(b *testing.B) {
	logger := discard_logger()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			jlog.Logger_Log(logger, "")
		}
	})
}

// Benchmark_Info mirrors zerolog's BenchmarkInfo.
func Benchmark_Info(b *testing.B) {
	logger := discard_logger()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			jlog.Logger_Info(logger, fake_message)
		}
	})
}

// Benchmark_Disabled mirrors zerolog's BenchmarkDisabled.
func Benchmark_Disabled(b *testing.B) {
	logger := jlog.New(jlog.New_Input{
		Writer: io.Discard,
		Clock:  frozen_clock(),
		Floor:  jlog.Level_Disabled,
	})
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			jlog.Logger_Info(logger, fake_message)
		}
	})
}

// Benchmark_Log_Fields mirrors zerolog's BenchmarkLogFields: a string, a time, an
// integer, and a float, plus the message.
func Benchmark_Log_Fields(b *testing.B) {
	logger := discard_logger()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			jlog.Logger_Info(logger, fake_message,
				jlog.String("string", "four!"),
				jlog.Time("time", fixed_moment),
				jlog.Integer("int", 123),
				jlog.Float32("float", -2.203230293249593),
			)
		}
	})
}

// Benchmark_Context_Fields mirrors zerolog's BenchmarkContextFields.
func Benchmark_Context_Fields(b *testing.B) {
	logger := jlog.Logger_With(discard_logger(),
		jlog.String("string", "four!"),
		jlog.Time("time", fixed_moment),
		jlog.Integer("int", 123),
		jlog.Float32("float", -2.203230293249593),
	)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			jlog.Logger_Info(logger, fake_message)
		}
	})
}

// Test_Cover_Levels exercises every level function and the level wire names.
func Test_Cover_Levels(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger := new_logger(buffer)
	jlog.Logger_Trace(logger, "")
	jlog.Logger_Debug(logger, "")
	jlog.Logger_Warn(logger, "")
	jlog.Logger_Log(logger, "")
	jlog.Logger_At_Level(logger, jlog.Level_Error, "")
	jlog.Logger_At_Level(logger, jlog.Level_Disabled, "")
	want := "{\"level\":\"trace\"}\n" + "{\"level\":\"debug\"}\n" +
		"{\"level\":\"warn\"}\n" + "{}\n" + "{\"level\":\"error\"}\n" +
		"{\"level\":\"disabled\"}\n"
	assert_output(t, buffer, want)
}

// Test_Cover_Level_String covers the cases not reached through emit.
func Test_Cover_Level_String(t *testing.T) {
	if jlog.Level_None.String() != "" {
		t.Fatalf("none = %q, want empty", jlog.Level_None.String())
	}
	if jlog.Level(50).String() != "50" {
		t.Fatalf("custom = %q, want 50", jlog.Level(50).String())
	}
}

// Test_Cover_Numeric_And_Slice_Fields covers the remaining scalar and array kinds.
func Test_Cover_Numeric_And_Slice_Fields(t *testing.T) {
	buffer := &bytes.Buffer{}
	jlog.Logger_Info(new_logger(buffer), "",
		jlog.Int64("a", -5),
		jlog.Uint("b", 6),
		jlog.Float32("c", 1.5),
		jlog.Floats64("d", []float64{1.5, 2.5}),
		jlog.Booleans("e", []bool{true, false}),
		jlog.Durations("f", []jtime.Duration{jtime.Second, 2 * jtime.Second}),
		jlog.Boolean("g", false),
	)
	want := "{\"level\":\"info\",\"a\":-5,\"b\":6,\"c\":1.5,\"d\":[1.5,2.5]," +
		"\"e\":[true,false],\"f\":[1000000000,2000000000],\"g\":false}\n"
	assert_output(t, buffer, want)
}

// Test_Cover_Any covers reflection marshaling and its failure path.
func Test_Cover_Any(t *testing.T) {
	buffer := &bytes.Buffer{}
	jlog.Logger_Info(new_logger(buffer), "",
		jlog.Any("a", map[string]int{"x": 1}),
		jlog.Any("b", make(chan int)),
	)
	want := "{\"level\":\"info\",\"a\":{\"x\":1},\"b\":\"json marshal error\"}\n"
	assert_output(t, buffer, want)
}

// Test_Cover_Err_Nil covers Err with a nil error and the skipped stack.
func Test_Cover_Err_Nil(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger := jlog.New(jlog.New_Input{
		Writer:          buffer,
		Clock:           frozen_clock(),
		Floor:           jlog.Level_Trace,
		Stack_Marshaler: func(value error) (stack string) { return "S" },
	})
	jlog.Logger_Error(logger, "", jlog.Err(nil))
	assert_output(t, buffer, "{\"level\":\"error\",\"error\":null}\n")
}

// Test_Cover_Caller_Without_Function covers Caller when no lookup is injected.
func Test_Cover_Caller_Without_Function(t *testing.T) {
	buffer := &bytes.Buffer{}
	jlog.Logger_Info(new_logger(buffer), "", jlog.Caller())
	assert_output(t, buffer, "{\"level\":\"info\"}\n")
}

// Test_Cover_Auto_Caller covers the per-line automatic caller.
func Test_Cover_Auto_Caller(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger := jlog.New(jlog.New_Input{
		Writer:      buffer,
		Clock:       frozen_clock(),
		Floor:       jlog.Level_Trace,
		Auto_Caller: true,
		Caller:      func(skip int) (location string) { return "x.go:1" },
	})
	jlog.Logger_Info(logger, "")
	assert_output(t, buffer, "{\"level\":\"info\",\"caller\":\"x.go:1\"}\n")
}

// Test_Cover_Disabled_Floor covers a logger whose floor disables all output.
func Test_Cover_Disabled_Floor(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger := jlog.New(jlog.New_Input{
		Writer: buffer,
		Clock:  frozen_clock(),
		Floor:  jlog.Level_Disabled,
	})
	jlog.Logger_Info(logger, "x")
	assert_output(t, buffer, "")
}

// Test_Cover_Nested_With covers Logger_With on a logger that already has a prefix.
func Test_Cover_Nested_With(t *testing.T) {
	buffer := &bytes.Buffer{}
	parent := jlog.Logger_With(new_logger(buffer), jlog.String("a", "1"))
	child := jlog.Logger_With(parent, jlog.String("b", "2"))
	jlog.Logger_Info(child, "")
	assert_output(t, buffer, "{\"level\":\"info\",\"a\":\"1\",\"b\":\"2\"}\n")
}

// Test_Cover_Custom_Field_Names covers configured (non-default) key names.
func Test_Cover_Custom_Field_Names(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger := jlog.New(jlog.New_Input{
		Writer:             buffer,
		Clock:              frozen_clock(),
		Floor:              jlog.Level_Trace,
		Level_Field_Name:   "lvl",
		Message_Field_Name: "msg",
	})
	jlog.Logger_Info(logger, "hi")
	assert_output(t, buffer, "{\"lvl\":\"info\",\"msg\":\"hi\"}\n")
}

// Test_Cover_Nil_Writer covers New defaulting a nil writer to a discard sink.
func Test_Cover_Nil_Writer(t *testing.T) {
	logger := jlog.New(jlog.New_Input{Clock: frozen_clock(), Floor: jlog.Level_Trace})
	jlog.Logger_Info(logger, "to nowhere")
}

// Parses the single JSON line written to buffer into a field map.
func decode_line(t *testing.T, buffer *bytes.Buffer) (fields map[string]any) {
	t.Helper()
	line := buffer.Bytes()
	fields = map[string]any{}
	if err := json.Unmarshal(line[:len(line)-1], &fields); err != nil {
		t.Fatalf("output is not valid JSON: %v (%q)", err, buffer.String())
	}
	return fields
}

// Test_Cover_String_Escape covers the escape path by round-tripping a value with
// control characters, a quote, a backslash, and a multibyte rune through the logger
// and back via JSON.
func Test_Cover_String_Escape(t *testing.T) {
	buffer := &bytes.Buffer{}
	value := "\b\f\n\r\t\"\\\x01世"
	jlog.Logger_Info(new_logger(buffer), "", jlog.String("a", value))
	if decode_line(t, buffer)["a"] != value {
		t.Fatalf("string did not round-trip: %q", buffer.String())
	}
}

// Test_Cover_Bytes_Escape covers the []byte escape path the same way.
func Test_Cover_Bytes_Escape(t *testing.T) {
	buffer := &bytes.Buffer{}
	value := "\t\"\\世"
	jlog.Logger_Info(new_logger(buffer), "", jlog.Bytes("a", []byte(value)))
	if decode_line(t, buffer)["a"] != value {
		t.Fatalf("bytes did not round-trip: %q", buffer.String())
	}
}

// Test_Cover_Invalid_UTF8 covers the replacement-character path for an invalid byte
// in both a string and a []byte value, and pins the wire form to the escaped
// replacement sequence so it stays byte-for-byte identical to zerolog on malformed input.
func Test_Cover_Invalid_UTF8(t *testing.T) {
	buffer := &bytes.Buffer{}
	jlog.Logger_Info(new_logger(buffer), "",
		jlog.String("s", "\xff"),
		jlog.Bytes("b", []byte("ok\xff")),
	)
	wire := buffer.Bytes()
	if !bytes.Contains(wire, []byte{'\\', 'u', 'f', 'f', 'f', 'd'}) {
		t.Fatalf("invalid byte must escape, got %q", buffer.String())
	}
	if bytes.ContainsRune(wire, '�') {
		t.Fatalf("output must not carry the raw replacement rune: %q", buffer.String())
	}
	fields := decode_line(t, buffer)
	if fields["s"] != "�" {
		t.Fatalf("invalid string byte not replaced: %v", fields["s"])
	}
	if fields["b"] != "ok�" {
		t.Fatalf("invalid byte not replaced: %v", fields["b"])
	}
}

// Test_Cover_Floats covers NaN, both infinities, zero, and exponent formatting at
// float32 and float64 precision.
func Test_Cover_Floats(t *testing.T) {
	buffer := &bytes.Buffer{}
	jlog.Logger_Info(new_logger(buffer), "",
		jlog.Float64("nan", math.NaN()),
		jlog.Float64("pinf", math.Inf(1)),
		jlog.Float64("ninf", math.Inf(-1)),
		jlog.Float64("zero", 0),
		jlog.Float64("small", 1e-7),
		jlog.Float64("big", 1e21),
		jlog.Float32("small32", 1e-7),
		jlog.Float32("big32", 1e21),
	)
	want := "{\"level\":\"info\",\"nan\":\"NaN\",\"pinf\":\"+Inf\",\"ninf\":\"-Inf\"," +
		"\"zero\":0,\"small\":1e-7,\"big\":1e+21,\"small32\":1e-7,\"big32\":1e+21}\n"
	assert_output(t, buffer, want)
}

// Test_Cover_Pre_Epoch_Time covers a timestamp before the Unix epoch, exercising
// the negative-fraction and negative-day paths of the RFC 3339 conversion.
func Test_Cover_Pre_Epoch_Time(t *testing.T) {
	buffer := &bytes.Buffer{}
	jlog.Logger_Info(new_logger(buffer), "", jlog.Time("t", jtime.Moment(-1)))
	want := "{\"level\":\"info\",\"t\":\"1969-12-31T23:59:59.999999999Z\"}\n"
	assert_output(t, buffer, want)
}

// Test_Cover_Oversized_Line covers dropping a buffer that grew past the pool cap.
func Test_Cover_Oversized_Line(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger := new_logger(buffer)
	big := make([]byte, 70000)
	for index := range big {
		big[index] = 'a'
	}
	jlog.Logger_Info(logger, "", jlog.String("x", string(big)))
	if buffer.Len() < 70000 {
		t.Fatalf("output was %d bytes; want a large line", buffer.Len())
	}
}

// Test_Cover_Unknown_Kind covers the defensive path for an unrecognized field kind.
func Test_Cover_Unknown_Kind(t *testing.T) {
	buffer := &bytes.Buffer{}
	jlog.Logger_Info(new_logger(buffer), "", jlog.Field{Key: "x", Kind: 100})
	assert_output(t, buffer, "{\"level\":\"info\"}\n")
}

// Test_Cover_Prefix_Without_Level covers merging a sub-logger prefix into a line
// that carries no level field, the no-separator branch of the object-data merge.
func Test_Cover_Prefix_Without_Level(t *testing.T) {
	buffer := &bytes.Buffer{}
	child := jlog.Logger_With(new_logger(buffer), jlog.String("a", "1"))
	jlog.Logger_Log(child, "")
	assert_output(t, buffer, "{\"a\":\"1\"}\n")
}

// Test_Cover_Float_Exponent_No_Trim covers an exponent whose second digit is not a
// zero, so the leading-zero trim is skipped.
func Test_Cover_Float_Exponent_No_Trim(t *testing.T) {
	buffer := &bytes.Buffer{}
	jlog.Logger_Info(new_logger(buffer), "", jlog.Float64("x", 1e-17))
	assert_output(t, buffer, "{\"level\":\"info\",\"x\":1e-17}\n")
}

// Fuzz_Encode throws random field values at the logger and asserts every line is valid
// JSON. It drives the string-escape paths (random and invalid-UTF-8 bytes), the float
// NaN/Inf/exponent paths, and — via the fuzzed moment feeding both Auto_Timestamp and a
// Time field — civil_from_days across the int64 epoch range, so the internal asserts that
// guard the civil-date math are proven; any regression panics with the named invariant.
func Fuzz_Encode(f *testing.F) {
	f.Add([]byte("hello"), int64(1700000000000000000), int64(42), math.Float64bits(3.14))
	f.Fuzz(func(t *testing.T, raw []byte, moment int64, number int64, float_bits uint64) {
		buffer := &bytes.Buffer{}
		clock := jtime.Clock{
			Now_Realtime: func() (now jtime.Moment) { return jtime.Moment(moment) },
		}
		logger := jlog.New(jlog.New_Input{
			Writer:         buffer,
			Clock:          clock,
			Floor:          jlog.Level_Trace,
			Auto_Timestamp: true,
		})
		jlog.Logger_Info(logger, string(raw),
			jlog.String("s", string(raw)),
			jlog.Bytes("b", raw),
			jlog.Int64("n", number),
			jlog.Float64("f", math.Float64frombits(float_bits)),
			jlog.Time("t", jtime.Moment(moment)),
			jlog.Duration("d", jtime.Duration(number)),
		)
		line := buffer.Bytes()
		var decoded map[string]any
		if err := json.Unmarshal(line[:len(line)-1], &decoded); err != nil {
			t.Fatalf("invalid JSON %q: %v", line, err)
		}
	})
}
