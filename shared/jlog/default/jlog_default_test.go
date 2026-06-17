package jlog_test

import (
	"bytes"
	"errors"
	"net"
	"testing"

	jlog "github.com/james-orcales/james-orcales/shared/jlog/default"
	jtime "github.com/james-orcales/james-orcales/shared/time"
)

// Test_Default_Global_Info covers the package-level convenience API writing
// an info-level line to Default through a re-exported field constructor.
func Test_Default_Global_Info(t *testing.T) {
	buffer := &bytes.Buffer{}
	saved := jlog.Default
	defer func() { jlog.Default = saved }()
	jlog.Default = jlog.New(jlog.New_Input{
		Writer: buffer,
		Clock:  jtime.Clock{Now_Realtime: func() (moment jtime.Moment) { return 0 }},
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
func frozen() (clock jtime.Clock) {
	return jtime.Clock{Now_Realtime: func() (moment jtime.Moment) { return 0 }}
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
		jlog.Time("l", jtime.Moment(0)),
		jlog.Duration("m", jtime.Second),
		jlog.IP_Address("n", net.IPv4(1, 2, 3, 4)),
		jlog.MAC_Address("o", net.HardwareAddr{1, 2, 3, 4, 5, 6}),
		jlog.Any("p", 1),
		jlog.Strings("q", []string{"a"}),
		jlog.Integers("r", []int{1}),
		jlog.Floats64("s", []float64{1}),
		jlog.Booleans("u", []bool{true}),
		jlog.Durations("w", []jtime.Duration{jtime.Second}),
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
