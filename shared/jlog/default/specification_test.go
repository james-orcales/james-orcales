package jlog_test

import (
	"bytes"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"testing"

	"local/james-orcales/shared/diode"
	jlog "local/james-orcales/shared/jlog/default"
	"local/james-orcales/shared/time"
	system_time "local/james-orcales/shared/time/default"
)

// Test_Header_Renders_Time_Level_Message covers the header order — timestamp, level, message —
// whatever order those keys arrived in on the wire (jlog emits level first, message last).
func Test_Header_Renders_Time_Level_Message(t *testing.T) {
	got := render(t, false,
		"{\"level\":\"info\",\"time\":\"2023-11-14T22:13:20Z\","+
			"\"message\":\"request done\"}\n")
	assert_output(t, got, "2023-11-14T22:13:20Z INF request done\n")
}

// Test_Timestamp_Drops_Fraction covers the header timestamp rendering to the second, its
// nanosecond fraction dropped, while the input line's own precision is untouched.
func Test_Timestamp_Drops_Fraction(t *testing.T) {
	got := render(t, false,
		"{\"level\":\"info\",\"time\":\"2023-11-14T22:13:20.123456789Z\","+
			"\"message\":\"m\"}\n")
	assert_output(t, got, "2023-11-14T22:13:20Z INF m\n")
}

// Test_Level_Is_Three_Letter_Uppercase covers the level tag mapping and, feeding five lines in
// one Write, the newline split.
func Test_Level_Is_Three_Letter_Uppercase(t *testing.T) {
	got := render(t, false,
		"{\"level\":\"trace\",\"message\":\"m\"}\n"+
			"{\"level\":\"debug\",\"message\":\"m\"}\n"+
			"{\"level\":\"info\",\"message\":\"m\"}\n"+
			"{\"level\":\"warn\",\"message\":\"m\"}\n"+
			"{\"level\":\"error\",\"message\":\"m\"}\n")
	assert_output(t, got, "TRC m\nDBG m\nINF m\nWRN m\nERR m\n")
}

// Test_Fields_Render_As_Logfmt_Pairs covers non-special keys rendering after the message in
// emission order.
func Test_Fields_Render_As_Logfmt_Pairs(t *testing.T) {
	got := render(t, false,
		"{\"level\":\"info\",\"method\":\"GET\",\"status\":200,\"message\":\"done\"}\n")
	assert_output(t, got, "INF done method=GET status=200\n")
}

// Test_String_Value_Is_Quoted_Only_When_Needed covers logfmt quoting: bare for a simple token,
// quoted when empty or carrying a space, an equals, or a quote.
func Test_String_Value_Is_Quoted_Only_When_Needed(t *testing.T) {
	got := render(t, false,
		"{\"level\":\"info\",\"a\":\"simple\",\"b\":\"two words\","+
			"\"c\":\"\",\"d\":\"a=b\"}\n")
	assert_output(t, got, "INF a=simple b=\"two words\" c=\"\" d=\"a=b\"\n")
}

// Test_Number_Value_Is_Exact covers a uint64 past the float64 safe range rendering its exact
// digits rather than a rounded float.
func Test_Number_Value_Is_Exact(t *testing.T) {
	got := render(t, false, "{\"level\":\"info\",\"big\":18446744073709551615}\n")
	assert_output(t, got, "INF big=18446744073709551615\n")
}

// Test_Compound_Value_Renders_Compact covers a scalar array and an embedded object (a Raw_JSON
// blob) each rendering as one compacted-JSON token.
func Test_Compound_Value_Renders_Compact(t *testing.T) {
	got := render(t, false,
		"{\"level\":\"info\",\"tags\":[\"a\",\"b\"],\"ids\":[1,2,3],\"obj\":{\"x\":1}}\n")
	assert_output(t, got, "INF tags=[\"a\",\"b\"] ids=[1,2,3] obj={\"x\":1}\n")
}

// Test_Caller_Renders_Inline covers the caller location rendering among the fields, not in the
// header.
func Test_Caller_Renders_Inline(t *testing.T) {
	got := render(t, false,
		"{\"level\":\"info\",\"caller\":\"api.go:42\",\"message\":\"done\"}\n")
	assert_output(t, got, "INF done caller=api.go:42\n")
}

// Test_Error_Value_Is_Red_When_Colored covers the error field: plain when color is off, its
// value painted red when on.
func Test_Error_Value_Is_Red_When_Colored(t *testing.T) {
	line := "{\"level\":\"error\",\"error\":\"context deadline exceeded\"," +
		"\"message\":\"db timeout\"}\n"
	assert_output(t, render(t, false, line),
		"ERR db timeout error=\"context deadline exceeded\"\n")
	assert_output(t, render(t, true, line),
		"\x1b[31mERR\x1b[0m \x1b[1mdb timeout\x1b[0m "+
			"\x1b[36merror\x1b[0m=\x1b[31m\"context deadline exceeded\"\x1b[0m\n")
}

// Test_Message_Is_Bold_When_Colored covers the full colored header: dim timestamp, colored
// level, bold message, dim key.
func Test_Message_Is_Bold_When_Colored(t *testing.T) {
	got := render(t, true,
		"{\"level\":\"info\",\"time\":\"2023-11-14T22:13:20Z\","+
			"\"message\":\"request done\",\"method\":\"GET\"}\n")
	assert_output(t, got,
		"\x1b[2m2023-11-14T22:13:20Z\x1b[0m \x1b[32mINF\x1b[0m "+
			"\x1b[1mrequest done\x1b[0m \x1b[36mmethod\x1b[0m=GET\n")
}

// Test_No_Color_When_Disabled covers a color-off Console emitting no ANSI escape byte.
func Test_No_Color_When_Disabled(t *testing.T) {
	got := render(t, false, "{\"level\":\"error\",\"error\":\"boom\",\"message\":\"m\"}\n")
	if strings.Contains(got.String(), "\x1b[") {
		t.Fatalf("plain output must carry no ANSI escape: %q", got.String())
	}
}

// Test_Level_None_Omits_Level covers a line with no level field (as Logger_Log writes) rendering
// no level tag, and a bare empty object rendering just a newline.
func Test_Level_None_Omits_Level(t *testing.T) {
	got := render(t, false, "{\"a\":\"1\",\"message\":\"note\"}\n{}\n")
	assert_output(t, got, "note a=1\n\n")
}

// Test_Malformed_Line_Passes_Through covers non-object lines (plain text, a broken object, a
// top-level array) being written through byte for byte.
func Test_Malformed_Line_Passes_Through(t *testing.T) {
	in := "not json at all\n{broken\n[1,2]\n"
	got := render(t, false, in)
	assert_output(t, got, in)
}

// Test_Terminal_Logger_Emits_Through_Console covers New_Terminal_Logger writing a line to
// standard output in the Console's form; a pipe is not a terminal, so color is off.
func Test_Terminal_Logger_Emits_Through_Console(t *testing.T) {
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	// New_Terminal_Logger binds os.Stdout at construction, so it must be redirected first.
	os.Stdout = writer
	logger := jlog.New_Terminal_Logger()
	jlog.Logger_Info(logger, "up", jlog.Integer("port", 8080))
	if close_err := writer.Close(); close_err != nil {
		t.Fatal(close_err)
	}
	os.Stdout = original

	// One bounded read captures the short line; io.ReadAll is banned as unbounded.
	output := make([]byte, 256)
	count, read_err := reader.Read(output)
	if read_err != nil {
		if read_err != io.EOF {
			t.Fatal(read_err)
		}
	}
	got := string(output[:count])
	// The OS clock's timestamp is not predictable, so only the fixed tail is asserted.
	if !strings.HasSuffix(got, " INF up port=8080\n") {
		t.Fatalf("got %q, want suffix %q", got, " INF up port=8080\n")
	}
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("a pipe is not a terminal; color must be off: %q", got)
	}
}

// Test_Default_Floor_Is_Info covers New_Default_Logger building with an Info floor, so trace and
// debug noise stays out of the stderr diode unless a caller raises verbosity explicitly.
func Test_Default_Floor_Is_Info(t *testing.T) {
	if got := jlog.New_Default_Logger().Floor; got != jlog.LEVEL_INFO {
		t.Fatalf("New_Default_Logger Floor = %v, want LEVEL_INFO", got)
	}
}

// Test_Terminal_Floor_Is_Debug covers New_Terminal_Logger building with a Debug floor, a step
// more verbose than the stderr default, since a developer watching a terminal wants debug lines
// without asking for trace-level noise too.
func Test_Terminal_Floor_Is_Debug(t *testing.T) {
	if got := jlog.New_Terminal_Logger().Floor; got != jlog.LEVEL_DEBUG {
		t.Fatalf("New_Terminal_Logger Floor = %v, want LEVEL_DEBUG", got)
	}
}

// Renders line through a Console with the given color setting and returns what the Console wrote,
// so a behaviour test is a single byte-for-byte comparison.
func render(t *testing.T, color bool, line string) (rendered *bytes.Buffer) {
	t.Helper()
	rendered = &bytes.Buffer{}
	console := jlog.New_Console(jlog.New_Console_Input{Writer: rendered, Color: color})
	if _, err := console.Write([]byte(line)); err != nil {
		t.Fatalf("console write: %v", err)
	}
	return rendered
}

// Fails the test unless the buffer holds exactly want.
func assert_output(t *testing.T, buffer *bytes.Buffer, want string) {
	t.Helper()
	if buffer.String() != want {
		t.Fatalf("got  %q\nwant %q", buffer.String(), want)
	}
}

// Test_Default_Global_Info covers the package-level convenience API writing an info-level line to
// Default through a re-exported field constructor.
func Test_Default_Global_Info(t *testing.T) {
	buffer := &bytes.Buffer{}
	saved := jlog.Default
	defer func() { jlog.Default = saved }()
	jlog.Default = jlog.New(jlog.New_Input{
		Writer: buffer,
		Clock:  time.Clock{Now_Realtime: func() (moment time.Moment) { return 0 }},
		Floor:  jlog.LEVEL_TRACE,
	})
	jlog.Info("hello", jlog.String("user", "bob"))
	got := buffer.String()
	want := "{\"level\":\"info\",\"user\":\"bob\",\"message\":\"hello\"}\n"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

// A clock whose realtime reading is always zero.
func frozen() (clock time.Clock) {
	return time.Clock{Now_Realtime: func() (moment time.Moment) { return 0 }}
}

// Test_Default_Caller_Uses_Runtime exercises the OS-backed caller lookup wired into Default; its
// one line goes to the real stderr.
func Test_Default_Caller_Uses_Runtime(t *testing.T) {
	jlog.Info("runtime caller coverage", jlog.Caller())
}

// Test_Default_Cover_API calls every re-exported wrapper and convenience function so the
// composition tier is fully exercised.
func Test_Default_Cover_API(t *testing.T) {
	buffer := &bytes.Buffer{}
	saved := jlog.Default
	defer func() { jlog.Default = saved }()
	jlog.Default = jlog.New(jlog.New_Input{
		Writer: buffer,
		Clock:  frozen(),
		Floor:  jlog.LEVEL_TRACE,
	})

	jlog.Trace("a")
	jlog.Debug("b")
	jlog.Info("c")
	jlog.Warn("d")
	jlog.Error("e")
	jlog.Log("f")
	jlog.Logger_Info(jlog.With(jlog.String("k", "v")), "g")

	logger := jlog.New(jlog.New_Input{
		Writer:          buffer,
		Clock:           frozen(),
		Floor:           jlog.LEVEL_TRACE,
		Stack_Marshaler: func(value error) (stack string) { return "S" },
	})
	jlog.Logger_Trace(logger, "")
	jlog.Logger_Debug(logger, "")
	jlog.Logger_Warn(logger, "")
	jlog.Logger_Error(logger, "")
	jlog.Logger_Log(logger, "")
	jlog.Logger_At_Level(logger, jlog.LEVEL_WARN, "")
	jlog.Logger_Info(jlog.Logger_With(logger, jlog.String("x", "y")), "with")
	jlog.Logger_Info(logger, "fields",
		jlog.String("a", "s"),
		jlog.Integer("b", 1),
		jlog.Int64("c", int64(2)),
		jlog.Uint("d", uint(3)),
		jlog.Uint64("e", uint64(4)),
		jlog.Float32("f", float32(1.5)),
		jlog.Float64("g", 2.5),
		jlog.Boolean("h", true),
		jlog.Bytes("i", []byte("x")),
		jlog.Hexadecimal("j", []byte{1}),
		jlog.Raw_JSON("k", []byte("1")),
		jlog.Time("l", time.Moment(0)),
		jlog.Duration("m", time.SECOND),
		jlog.IP_Address("n", net.IPv4(1, 2, 3, 4)),
		jlog.MAC_Address("o", net.HardwareAddr{1, 2, 3, 4, 5, 6}),
		jlog.Any("p", 1),
		jlog.Strings("q", []string{"a"}),
		jlog.Integers("r", []int{1}),
		jlog.Floats64("s", []float64{1}),
		jlog.Booleans("u", []bool{true}),
		jlog.Durations("w", []time.Duration{time.SECOND}),
		jlog.Err(errors.New("boom")),
		jlog.Timestamp(),
		jlog.Caller(),
	)

	carrier := jlog.Logger_With_Context(logger, t.Context())
	jlog.Logger_Info(jlog.From_Context(carrier), "ctx")

	if buffer.Len() == 0 {
		t.Fatal("expected output from the default wrappers")
	}
}

// Opens the OS bit bucket as a real sink: writing to it still costs a write syscall (unlike
// io.Discard), so benchmarks against it reflect a real backend.
func null_sink(b *testing.B) (sink *os.File) {
	b.Helper()
	handle, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatal(err)
	}
	return handle
}

// Benchmark_Caller_Synchronous measures what a caller pays to log one line straight to a real
// sink: formatting plus the write syscall, on the caller's goroutine.
func Benchmark_Caller_Synchronous(b *testing.B) {
	sink := null_sink(b)
	defer sink.Close()
	logger := jlog.New(jlog.New_Input{Writer: sink, Clock: frozen(), Floor: jlog.LEVEL_TRACE})
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			jlog.Logger_Info(logger, "request done", jlog.String("method", "GET"))
		}
	})
}

// Benchmark_Caller_Diode measures the same line through the default's non-blocking diode: the
// caller pays formatting plus a ring handoff; the write syscall is moved to the drain goroutine,
// which sleeps the real ten-millisecond poll interval.
func Benchmark_Caller_Diode(b *testing.B) {
	sink := null_sink(b)
	defer sink.Close()
	clock, _ := system_time.New_Operating_System_Clock()
	writer := diode.New(diode.New_Input{
		Writer: sink,
		Clock:  clock,
		Sleep:  system_time.Sleep,
		Count:  1024,
	})
	defer writer.Close()
	logger := jlog.New(jlog.New_Input{Writer: writer, Clock: frozen(), Floor: jlog.LEVEL_TRACE})
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			jlog.Logger_Info(logger, "request done", jlog.String("method", "GET"))
		}
	})
}
