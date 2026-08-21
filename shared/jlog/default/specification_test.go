package jlog_test

import (
	"errors"
	"net"
	"testing"
	"unsafe"

	"local/james-orcales/shared/jlog/default"
	"local/james-orcales/shared/simulation/time"
)

// Test_Default_Floor_Is_Info covers New_Default_Logger building with an Info floor, so trace and
// debug noise stays out of the stderr diode unless a caller raises verbosity explicitly.
func Test_Default_Floor_Is_Info(t *testing.T) {
	if got := jlog.New_Default_Logger().Floor; got != jlog.LEVEL_INFO {
		t.Fatalf("New_Default_Logger Floor = %v, want LEVEL_INFO", got)
	}
}

type recording_buffer []byte

func (buffer recording_buffer) String() (text string) { return string(buffer) }

// Test_Default_Global_Info covers the package-level convenience API writing an info-level line to
// Default through a re-exported field constructor.
func Test_Default_Global_Info(t *testing.T) {
	buffer := &recording_buffer{}
	saved := jlog.Default
	defer func() { jlog.Default = saved }()
	jlog.Default = jlog.New(jlog.New_Input{
		Writer_State: unsafe.Pointer(buffer),
		Write:        buffer_write,
		Clock:        frozen(),
		Floor:        jlog.LEVEL_TRACE,
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
	return time.Clock{Now_Realtime: zero_realtime}
}

func zero_realtime(_ unsafe.Pointer) (moment time.Moment) { return 0 }

func buffer_write(
	state unsafe.Pointer, data jlog.Data,
) (written jlog.Data_Size, err error) {
	buffer := (*recording_buffer)(state)
	*buffer = append(*buffer, data...)
	return jlog.Data_Size(len(data)), nil
}

// Test_Default_Caller_Uses_Runtime exercises the OS-backed caller lookup wired into Default; its
// one line goes to the real stderr.
func Test_Default_Caller_Uses_Runtime(t *testing.T) {
	jlog.Info("runtime caller coverage", jlog.Caller())
}

// Test_Default_Cover_API calls every re-exported wrapper and convenience function so the
// composition tier is fully exercised.
func Test_Default_Cover_API(t *testing.T) {
	buffer := &recording_buffer{}
	saved := jlog.Default
	defer func() { jlog.Default = saved }()
	jlog.Default = jlog.New(jlog.New_Input{
		Writer_State: unsafe.Pointer(buffer),
		Write:        buffer_write,
		Clock:        frozen(),
		Floor:        jlog.LEVEL_TRACE,
	})

	jlog.Trace("a")
	jlog.Debug("b")
	jlog.Info("c")
	jlog.Warn("d")
	jlog.Error("e")
	jlog.Log("f")
	jlog.Logger_Info(jlog.With(jlog.String("k", "v")), "g")

	logger := jlog.New(jlog.New_Input{
		Writer_State:    unsafe.Pointer(buffer),
		Write:           buffer_write,
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

	if len(*buffer) == 0 {
		t.Fatal("expected output from the default wrappers")
	}
}
