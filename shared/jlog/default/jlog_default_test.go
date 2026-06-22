package jlog_test

import (
	"bytes"
	"errors"
	"net"
	"os"
	"testing"

	"github.com/james-orcales/james-orcales/shared/diode"
	jlog "github.com/james-orcales/james-orcales/shared/jlog/default"
	"github.com/james-orcales/james-orcales/shared/time"
	system_time "github.com/james-orcales/james-orcales/shared/time/default"
)

// Test_Default_Global_Info covers the package-level convenience API writing
// an info-level line to Default through a re-exported field constructor.
func Test_Default_Global_Info(t *testing.T) {
	buffer := &bytes.Buffer{}
	saved := jlog.Default
	defer func() { jlog.Default = saved }()
	jlog.Default = jlog.New(jlog.New_Input{
		Writer: buffer,
		Clock:  time.Clock{Now_Realtime: func() (moment time.Moment) { return 0 }},
		Floor:  jlog.Level_Trace,
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

// Test_Default_Caller_Uses_Runtime exercises the OS-backed caller lookup wired into
// Default; its one line goes to the real stderr.
func Test_Default_Caller_Uses_Runtime(t *testing.T) {
	jlog.Info("runtime caller coverage", jlog.Caller())
}

// Test_Default_Cover_API calls every re-exported wrapper and convenience function so
// the composition tier is fully exercised.
func Test_Default_Cover_API(t *testing.T) {
	buffer := &bytes.Buffer{}
	saved := jlog.Default
	defer func() { jlog.Default = saved }()
	jlog.Default = jlog.New(jlog.New_Input{
		Writer: buffer,
		Clock:  frozen(),
		Floor:  jlog.Level_Trace,
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
		Floor:           jlog.Level_Trace,
		Stack_Marshaler: func(value error) (stack string) { return "S" },
	})
	jlog.Logger_Trace(logger, "")
	jlog.Logger_Debug(logger, "")
	jlog.Logger_Warn(logger, "")
	jlog.Logger_Error(logger, "")
	jlog.Logger_Log(logger, "")
	jlog.Logger_At_Level(logger, jlog.Level_Warn, "")
	jlog.Logger_Info(jlog.Logger_With(logger, jlog.String("x", "y")), "with")
	jlog.Logger_Info(logger, "fields",
		jlog.String("a", "s"),
		jlog.Integer("b", 1),
		jlog.Int64("c", 2),
		jlog.Uint("d", 3),
		jlog.Uint64("e", 4),
		jlog.Float32("f", 1.5),
		jlog.Float64("g", 2.5),
		jlog.Boolean("h", true),
		jlog.Bytes("i", []byte("x")),
		jlog.Hexadecimal("j", []byte{1}),
		jlog.Raw_JSON("k", []byte("1")),
		jlog.Time("l", time.Moment(0)),
		jlog.Duration("m", time.Second),
		jlog.IP_Address("n", net.IPv4(1, 2, 3, 4)),
		jlog.MAC_Address("o", net.HardwareAddr{1, 2, 3, 4, 5, 6}),
		jlog.Any("p", 1),
		jlog.Strings("q", []string{"a"}),
		jlog.Integers("r", []int{1}),
		jlog.Floats64("s", []float64{1}),
		jlog.Booleans("u", []bool{true}),
		jlog.Durations("w", []time.Duration{time.Second}),
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

// Opens the OS bit bucket as a real sink: writing to it still costs a write syscall
// (unlike io.Discard), so benchmarks against it reflect a real backend.
func null_sink(b *testing.B) (sink *os.File) {
	b.Helper()
	handle, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatal(err)
	}
	return handle
}

// Benchmark_Caller_Synchronous measures what a caller pays to log one line straight to
// a real sink: formatting plus the write syscall, on the caller's goroutine.
func Benchmark_Caller_Synchronous(b *testing.B) {
	sink := null_sink(b)
	defer sink.Close()
	logger := jlog.New(jlog.New_Input{Writer: sink, Clock: frozen(), Floor: jlog.Level_Trace})
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			jlog.Logger_Info(logger, "request done", jlog.String("method", "GET"))
		}
	})
}

// Benchmark_Caller_Diode measures the same line through the default's non-blocking
// diode: the caller pays formatting plus a ring handoff; the write syscall is moved to
// the drain goroutine, which sleeps the real ten-millisecond poll interval.
func Benchmark_Caller_Diode(b *testing.B) {
	sink := null_sink(b)
	defer sink.Close()
	writer := diode.New(diode.New_Input{
		Writer: sink,
		Clock:  system_time.New_Operating_System_Clock(),
		Count:  1024,
	})
	defer writer.Close()
	logger := jlog.New(jlog.New_Input{Writer: writer, Clock: frozen(), Floor: jlog.Level_Trace})
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			jlog.Logger_Info(logger, "request done", jlog.String("method", "GET"))
		}
	})
}
