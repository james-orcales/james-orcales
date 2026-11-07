package jlog_test

import "errors"
import "testing"
import "unsafe"

import "local/james-orcales/shared/std/encoding/json"
import "local/james-orcales/shared/std/jlog"
import "local/james-orcales/shared/std/sim/nbio"
import "local/james-orcales/shared/std/sim/time"
import "local/james-orcales/shared/std/testify"

// Test_Message_Renders_Level_And_Message covers the minimal line: a level field
// then the trailing message field, terminated by a newline.
func Test_Message_Renders_Level_And_Message(t *testing.T) {
  buffer := &recording_buffer{}
  jlog.Logger_Info(new_logger(buffer), "hello")
  assert_output(t, buffer, "{\"level\":\"info\",\"message\":\"hello\"}\n")
}

// Test_Empty_Message_Is_Omitted covers an empty message producing no message field.
func Test_Empty_Message_Is_Omitted(t *testing.T) {
  buffer := &recording_buffer{}
  jlog.Logger_Info(new_logger(buffer), "", jlog.Integer("n", 42))
  assert_output(t, buffer, "{\"level\":\"info\",\"n\":42}\n")
}

// Test_Scalar_Fields covers the scalar encoders and that the message is rendered
// after the fields.
func Test_Scalar_Fields(t *testing.T) {
  buffer := &recording_buffer{}
  jlog.Logger_Info(
    new_logger(buffer),
    "done",
    jlog.String("s", "v"),
    jlog.Boolean("b", true),
    jlog.Integer("i", -7),
    jlog.Uint64("u", uint64(9)),
    jlog.Float64("f", 1.5),
  )
  assert_output(
    t,
    buffer,
    "{\"level\":\"info\",\"s\":\"v\",\"b\":true,\"i\":-7,\"u\":9,"+
      "\"f\":1.5,\"message\":\"done\"}\n",
  )
}

// Test_String_Escape_Sequences covers JSON escaping of control and quote characters.
func Test_String_Escape_Sequences(t *testing.T) {
  buffer := &recording_buffer{}
  jlog.Logger_Info(new_logger(buffer), "", jlog.String("s", "a\"b\nc"))
  assert_output(t, buffer, "{\"level\":\"info\",\"s\":\"a\\\"b\\nc\"}\n")
}

// Test_Bytes_Hexadecimal_Raw_JSON covers the []byte-valued field encoders.
func Test_Bytes_Hexadecimal_Raw_JSON(t *testing.T) {
  buffer := &recording_buffer{}
  jlog.Logger_Info(
    new_logger(buffer),
    "",
    jlog.Bytes("by", []byte("ab")),
    jlog.Hexadecimal("hx", []byte{0xab, 0xcd}),
    jlog.Raw_JSON("rj", []byte("{\"x\":1}")),
  )
  assert_output(t, buffer,
    "{\"level\":\"info\",\"by\":\"ab\",\"hx\":\"abcd\",\"rj\":{\"x\":1}}\n")
}

// Test_Timestamp_Uses_Injected_Clock covers the Timestamp field reading the
// injected clock and rendering it as an RFC 3339 string.
func Test_Timestamp_Uses_Injected_Clock(t *testing.T) {
  buffer := &recording_buffer{}
  jlog.Logger_Info(new_logger(buffer), "", jlog.Timestamp())
  assert_output(t, buffer, "{\"level\":\"info\",\"time\":\"2023-11-14T22:13:20Z\"}\n")
}

// Test_Auto_Timestamp covers Logger.Auto_Timestamp stamping every line.
func Test_Auto_Timestamp(t *testing.T) {
  buffer := &recording_buffer{}
  logger := new_logger(buffer)
  logger.Auto_Timestamp = true
  jlog.Logger_Info(logger, "")
  assert_output(t, buffer, "{\"level\":\"info\",\"time\":\"2023-11-14T22:13:20Z\"}\n")
}

// Test_Time_And_Duration covers the shared/std/sim/time value encoders.
func Test_Time_And_Duration(t *testing.T) {
  buffer := &recording_buffer{}
  jlog.Logger_Info(
    new_logger(buffer),
    "",
    jlog.Time("t", time.Moment(5)),
    jlog.Duration("d", time.SECOND),
  )
  assert_output(
    t,
    buffer,
    "{\"level\":\"info\",\"t\":\"1970-01-01T00:00:00.000000005Z\",\"d\":1000000000}\n",
  )
}

// Test_Scalar_Arrays covers typed slice fields rendering as flat JSON arrays.
func Test_Scalar_Arrays(t *testing.T) {
  buffer := &recording_buffer{}
  jlog.Logger_Info(
    new_logger(buffer),
    "",
    jlog.Strings("tags", []string{"a", "b"}),
    jlog.Integers("ids", []int{1, 2, 3}),
  )
  assert_output(t, buffer,
    "{\"level\":\"info\",\"tags\":[\"a\",\"b\"],\"ids\":[1,2,3]}\n")
}

// Test_Err_With_Stack covers Err appending the injected stack rendering before the
// error string when a stack marshaler is configured.
func Test_Err_With_Stack(t *testing.T) {
  buffer := &recording_buffer{}
  logger := build(
    jlog.Logger_Configuration{
      Stream:          recording_stream(buffer),
      Clock:           frozen_clock(),
      Stack_Marshaler: trace_stack,
    },
    jlog.LEVEL_TRACE,
  )
  jlog.Logger_Error(logger, "", jlog.Err(errors.New("boom")))
  assert_output(t, buffer,
    "{\"level\":\"error\",\"stack\":\"TRACE\",\"error\":\"boom\"}\n")
}

// Test_Err_Without_Stack covers Err omitting the stack when no marshaler is set.
func Test_Err_Without_Stack(t *testing.T) {
  buffer := &recording_buffer{}
  jlog.Logger_Error(new_logger(buffer), "", jlog.Err(errors.New("boom")))
  assert_output(t, buffer, "{\"level\":\"error\",\"error\":\"boom\"}\n")
}

// Test_Caller_Uses_Injected_Function covers the Caller field appending the injected
// location.
func Test_Caller_Uses_Injected_Function(t *testing.T) {
  buffer := &recording_buffer{}
  logger := build(
    jlog.Logger_Configuration{
      Stream: recording_stream(buffer),
      Clock:  frozen_clock(),
      Caller: file_caller,
    },
    jlog.LEVEL_TRACE,
  )
  jlog.Logger_Info(logger, "", jlog.Caller())
  assert_output(t, buffer, "{\"level\":\"info\",\"caller\":\"file.go:10\"}\n")
}

// Test_Child_Logger_Carries_Context covers Logger_With building a child whose fixed
// fields precede each line's own fields.
func Test_Child_Logger_Carries_Context(t *testing.T) {
  buffer := &recording_buffer{}
  child := with(new_logger(buffer), jlog.String("component", "auth"))
  jlog.Logger_Info(&child, "in", jlog.String("user", "bob"))
  assert_output(
    t,
    buffer,
    "{\"level\":\"info\",\"component\":\"auth\",\"user\":\"bob\",\"message\":\"in\"}\n",
  )
}

// Test_Level_Floor_Filters covers a line below the floor producing no output.
func Test_Level_Floor_Filters(t *testing.T) {
  buffer := &recording_buffer{}
  logger := build(
    jlog.Logger_Configuration{
      Stream: recording_stream(buffer), Clock: frozen_clock(),
    },
    jlog.LEVEL_INFO,
  )
  jlog.Logger_Debug(logger, "dropped", jlog.String("k", "v"))
  assert_output(t, buffer, "")
}

// Test_Nil_Logger_Is_Disabled covers a nil Logger emitting nothing rather than panicking.
func Test_Nil_Logger_Is_Disabled(t *testing.T) {
  jlog.Logger_Info(nil, "nope", jlog.String("k", "v"))
}

// Test_Hot_Path_Is_Zero_Allocation is the load-bearing guarantee: a steady-state
// log call to a discarding stream must not allocate.
func Test_Hot_Path_Is_Zero_Allocation(t *testing.T) {
  logger := discard_logger()
  testify.Zero_Allocation(t, func() {
    jlog.Logger_Info(
      logger,
      "done",
      jlog.String("user", "bob"),
      jlog.Integer("count", 7),
      jlog.Boolean("ok", true),
      jlog.Float64("ratio", 1.5),
    )
  })
}

// Test_Construction_Is_Zero_Allocation covers New and Logger_With filling caller-owned
// storage without allocating.
func Test_Construction_Is_Zero_Allocation(t *testing.T) {
  configuration := jlog.Logger_Configuration{Stream: discard_stream(), Clock: frozen_clock()}
  var logger, child jlog.Logger
  storage := make([]byte, TEST_LINE_BYTES)
  child_storage := make([]byte, TEST_LINE_BYTES)
  prefix := jlog.Buffer{Bytes: make([]byte, TEST_PREFIX_BYTES)}
  testify.Zero_Allocation(t, func() {
    store_result(&logger, jlog.New(&configuration, storage, jlog.LEVEL_TRACE))
    store_result(&child, jlog.Logger_With(&logger, child_storage, prefix, jlog.String("a", "1")))
  })
  jlog.Logger_Info(&child, "")
}

// Test_Borrowed_Line_Drops covers a Stream holding the line past the emit: the next emit drops
// rather than overwrite storage in flight, and the retirement frees it for the one after.
func Test_Borrowed_Line_Drops(t *testing.T) {
  deferred := &deferred_state{}
  logger := build(
    jlog.Logger_Configuration{
      Stream: nbio.Stream{State: deferred, Procedure: deferred_procedure},
      Clock:  frozen_clock(),
    },
    jlog.LEVEL_TRACE,
  )
  jlog.Logger_Info(logger, "a")
  jlog.Logger_Info(logger, "b")
  if logger.Dropped != 1 {
    t.Fatalf("dropped = %d, want 1", logger.Dropped)
  }
  assert_output(t, &deferred.Buffer, "{\"level\":\"info\",\"message\":\"a\"}\n")
  deferred_retire(deferred)
  jlog.Logger_Info(logger, "c")
  deferred_retire(deferred)
  assert_output(
    t,
    &deferred.Buffer,
    "{\"level\":\"info\",\"message\":\"a\"}\n{\"level\":\"info\",\"message\":\"c\"}\n",
  )
}

// Test_Oversized_Line_Drops covers a line that outgrows its storage being dropped whole and
// counted, so the Stream never sees a torn line.
func Test_Oversized_Line_Drops(t *testing.T) {
  buffer := &recording_buffer{}
  logger := new_logger(buffer)
  big := make([]byte, TEST_LINE_BYTES)
  for index := range big {
    big[index] = 'a'
  }
  jlog.Logger_Info(logger, "", jlog.String("x", string(big)))
  assert_output(t, buffer, "")
  if logger.Dropped != 1 {
    t.Fatalf("dropped = %d, want 1", logger.Dropped)
  }
  jlog.Logger_Info(logger, "after")
  assert_output(t, buffer, "{\"level\":\"info\",\"message\":\"after\"}\n")
}

// Test_Header_Renders_Time_Level_Message covers the header order — timestamp, level, message —
// whatever order those keys arrived in on the wire (jlog emits level first, message last).
func Test_Header_Renders_Time_Level_Message(t *testing.T) {
  got := render(
    t,
    false,
    "{\"level\":\"info\",\"time\":\"2023-11-14T22:13:20Z\","+
      "\"message\":\"request done\"}\n",
  )
  assert_output(t, got, "2023-11-14T22:13:20Z INF request done\n")
}

// Test_Timestamp_Drops_Fraction covers the header timestamp rendering to the second, its
// nanosecond fraction dropped, while the input line's own precision is untouched.
func Test_Timestamp_Drops_Fraction(t *testing.T) {
  got := render(
    t,
    false,
    "{\"level\":\"info\",\"time\":\"2023-11-14T22:13:20.123456789Z\","+
      "\"message\":\"m\"}\n",
  )
  assert_output(t, got, "2023-11-14T22:13:20Z INF m\n")
}

// Test_Level_Is_Three_Letter_Uppercase covers the level tag mapping and, feeding five lines in
// one Write, the newline split.
func Test_Level_Is_Three_Letter_Uppercase(t *testing.T) {
  got := render(
    t,
    false,
    "{\"level\":\"trace\",\"message\":\"m\"}\n"+
      "{\"level\":\"debug\",\"message\":\"m\"}\n"+
      "{\"level\":\"info\",\"message\":\"m\"}\n"+
      "{\"level\":\"warn\",\"message\":\"m\"}\n"+
      "{\"level\":\"error\",\"message\":\"m\"}\n",
  )
  assert_output(t, got, "TRC m\nDBG m\nINF m\nWRN m\nERR m\n")
}

// Test_Fields_Render_As_Logfmt_Pairs covers non-special keys rendering after the message in
// emission order.
func Test_Fields_Render_As_Logfmt_Pairs(t *testing.T) {
  got := render(
    t,
    false,
    "{\"level\":\"info\",\"method\":\"GET\",\"status\":200,\"message\":\"done\"}\n",
  )
  assert_output(t, got, "INF done method=GET status=200\n")
}

// Test_String_Value_Is_Quoted_Only_When_Needed covers logfmt quoting: bare for a simple token,
// quoted when empty or carrying a space, an equals, or a quote.
func Test_String_Value_Is_Quoted_Only_When_Needed(t *testing.T) {
  got := render(
    t,
    false,
    "{\"level\":\"info\",\"a\":\"simple\",\"b\":\"two words\","+
      "\"c\":\"\",\"d\":\"a=b\"}\n",
  )
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
  got := render(
    t,
    false,
    "{\"level\":\"info\",\"tags\":[\"a\",\"b\"],\"ids\":[1,2,3],\"obj\":{\"x\":1}}\n",
  )
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
  assert_output(
    t,
    render(t, true, line),
    "\x1b[31mERR\x1b[0m \x1b[1mdb timeout\x1b[0m "+
      "\x1b[36merror\x1b[0m=\x1b[31m\"context deadline exceeded\"\x1b[0m\n",
  )
}

// Test_Message_Is_Bold_When_Colored covers the full colored header: dim timestamp, colored
// level, bold message, dim key.
func Test_Message_Is_Bold_When_Colored(t *testing.T) {
  got := render(
    t,
    true,
    "{\"level\":\"info\",\"time\":\"2023-11-14T22:13:20Z\","+
      "\"message\":\"request done\",\"method\":\"GET\"}\n",
  )
  assert_output(
    t,
    got,
    "\x1b[2m2023-11-14T22:13:20Z\x1b[0m \x1b[32mINF\x1b[0m "+
      "\x1b[1mrequest done\x1b[0m \x1b[36mmethod\x1b[0m=GET\n",
  )
}

// Test_No_Color_When_Disabled covers a color-off Console emitting no ANSI escape byte.
func Test_No_Color_When_Disabled(t *testing.T) {
  got := render(t, false, "{\"level\":\"error\",\"error\":\"boom\",\"message\":\"m\"}\n")
  if contains([]byte(buffer_text(got)), []byte("\x1b[")) {
    t.Fatalf("plain output must carry no ANSI escape: %q", buffer_text(got))
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

// Test_Level_Filter_Drops_Below_Floor covers a Level_Filter forwarding lines at or above its floor
// and dropping the rest, so one logger can feed a verbose sink and a quiet one at once.
func Test_Level_Filter_Drops_Below_Floor(t *testing.T) {
  got := filter(
    t,
    jlog.LEVEL_INFO,
    "{\"level\":\"trace\",\"message\":\"a\"}\n"+
      "{\"level\":\"debug\",\"message\":\"b\"}\n"+
      "{\"level\":\"info\",\"message\":\"c\"}\n"+
      "{\"level\":\"warn\",\"message\":\"d\"}\n"+
      "{\"level\":\"error\",\"message\":\"e\"}\n",
  )
  assert_output(
    t,
    got,
    "{\"level\":\"info\",\"message\":\"c\"}\n"+
      "{\"level\":\"warn\",\"message\":\"d\"}\n"+
      "{\"level\":\"error\",\"message\":\"e\"}\n",
  )
}

// Test_Level_Filter_Passes_Level_Less_And_Non_JSON covers a Level_Filter passing a line with no
// level field (as Logger_Log writes) and a non-JSON line even at a high floor, since it
// suppresses by level alone and never swallows output it cannot classify.
func Test_Level_Filter_Passes_Level_Less_And_Non_JSON(t *testing.T) {
  got := filter(t, jlog.LEVEL_ERROR, "{\"message\":\"note\"}\nnot json\n")
  assert_output(t, got, "{\"message\":\"note\"}\nnot json\n")
}

// Test_Console_Logger_Tees_To_Capture covers New_Console_Logger rendering info and up to the
// console sink while teeing every level as raw JSON to the capture sink — the print-to-file,
// pure and driven by injected streams, with no OS in sight.
func Test_Console_Logger_Tees_To_Capture(t *testing.T) {
  var console, capture recording_buffer
  state := jlog.Console_Logger{
    Configuration: jlog.Logger_Configuration{Clock: frozen_clock()},
    Console:       new_console(recording_stream(&console), false),
    Capture:       recording_stream(&capture),
  }
  state.Filter.Kept = make([]byte, TEST_TRANSFORM_BYTES)
  state.Filter.Object = new_object()
  logger := jlog.New_Console_Logger(
    &state, make([]byte, TEST_LINE_BYTES), jlog.LEVEL_INFO,
  )
  jlog.Logger_Trace(&logger, "quiet")
  jlog.Logger_Info(&logger, "loud")
  // The console shows info and up, never the trace line.
  if contains([]byte(buffer_text(&console)), []byte("quiet")) {
    t.Fatalf("console must drop the trace line: %q", buffer_text(&console))
  }
  if !contains([]byte(buffer_text(&console)), []byte("INF loud")) {
    t.Fatalf("console must show the info line: %q", buffer_text(&console))
  }
  // The capture keeps every level as raw JSON, including the trace line the console dropped.
  if !contains([]byte(buffer_text(&capture)), []byte("\"message\":\"quiet\"")) {
    t.Fatalf("capture must keep the trace line: %q", buffer_text(&capture))
  }
  if !contains([]byte(buffer_text(&capture)), []byte("\"message\":\"loud\"")) {
    t.Fatalf("capture must keep the info line: %q", buffer_text(&capture))
  }
}

// Test_Transforms_Are_Zero_Allocation covers a Console render and a Level_Filter pass over a
// ready inner stream performing zero heap allocations: both work in caller-owned storage.
func Test_Transforms_Are_Zero_Allocation(t *testing.T) {
  line := []byte(
    "{\"level\":\"info\",\"time\":\"2023-11-14T22:13:20.5Z\",\"a\":\"x y\"," +
      "\"n\":1,\"error\":\"e\\n\",\"message\":\"m\"}\n",
  )
  console := new_console(discard_stream(), true)
  console_stream := jlog.Console_To_Stream(&console)
  level_filter := new_level_filter(discard_stream(), jlog.LEVEL_INFO)
  filter_stream := jlog.Level_Filter_To_Stream(level_filter)
  var completion nbio.Completion
  testify.Zero_Allocation(t, func() {
    nbio.Write(console_stream, &completion, line, disarm)
    nbio.Write(filter_stream, &completion, line, disarm)
  })
  if completion.Error != nil {
    t.Fatalf("transform write: %v", completion.Error)
  }
}

// Test_Oversized_Payload_Retires_Buffer_Full covers a Write whose transform outgrows its
// caller-owned storage retiring with nbio.Stream_Buffer_Full and passing nothing to the inner
// stream.
func Test_Oversized_Payload_Retires_Buffer_Full(t *testing.T) {
  big := make([]byte, TEST_TRANSFORM_BYTES+1)
  for index := range big {
    big[index] = 'a'
  }
  big[len(big)-1] = '\n'
  passed := &recording_buffer{}
  level_filter := new_level_filter(recording_stream(passed), jlog.LEVEL_TRACE)
  var completion nbio.Completion
  nbio.Write(jlog.Level_Filter_To_Stream(level_filter), &completion, big, disarm)
  if completion.Error != nbio.Stream_Buffer_Full {
    t.Fatalf("filter error = %v, want Stream_Buffer_Full", completion.Error)
  }
  assert_output(t, passed, "")
  rendered := &recording_buffer{}
  console := new_console(recording_stream(rendered), false)
  nbio.Write(jlog.Console_To_Stream(&console), &completion, big, disarm)
  if completion.Error != nbio.Stream_Buffer_Full {
    t.Fatalf("console error = %v, want Stream_Buffer_Full", completion.Error)
  }
  assert_output(t, rendered, "")
}

// Test_Transform_Busy_Drops_Second_Write covers a Write arriving while an earlier Write still
// holds the transform's storage, as a parent and a child logger sharing one Console produce: the
// second retires Stream_Buffer_Full and the in-flight bytes stay intact.
func Test_Transform_Busy_Drops_Second_Write(t *testing.T) {
  console_inner := &deferred_state{}
  console := new_console(
    nbio.Stream{State: console_inner, Procedure: deferred_procedure}, false,
  )
  test_transform_busy(t, jlog.Console_To_Stream(&console), console_inner)
  filter_inner := &deferred_state{}
  level_filter := new_level_filter(
    nbio.Stream{State: filter_inner, Procedure: deferred_procedure},
    jlog.LEVEL_TRACE,
  )
  test_transform_busy(t, jlog.Level_Filter_To_Stream(level_filter), filter_inner)
}

func test_transform_busy(t *testing.T, stream nbio.Stream, inner *deferred_state) {
  t.Helper()
  var first, second nbio.Completion
  nbio.Write(stream, &first, []byte("{\"message\":\"one\"}\n"), disarm)
  nbio.Write(stream, &second, []byte("{\"message\":\"two\"}\n"), disarm)
  if second.Error != nbio.Stream_Buffer_Full {
    t.Fatalf("second write error = %v, want Stream_Buffer_Full", second.Error)
  }
  if contains(inner.Buffer, []byte("two")) {
    t.Fatalf("busy transform forwarded the second line: %q", string(inner.Buffer))
  }
  deferred_retire(inner)
  if first.Error != nil {
    t.Fatalf("first write error = %v, want nil", first.Error)
  }
  if !contains(inner.Buffer, []byte("one")) {
    t.Fatalf("first line must reach the inner stream: %q", string(inner.Buffer))
  }
}

type recording_buffer []byte

func buffer_text(buffer *recording_buffer) (text string) { return string(*buffer) }

// A clock whose realtime reading is always FIXED_MOMENT.
func frozen_clock() (host time.Clock) {
  return time.Clock{Now_Monotonic: zero_monotonic, Now_Realtime: frozen_realtime}
}

func zero_monotonic(_ time.State) (moment time.Monotonic_Moment) { return 0 }

func frozen_realtime(_ time.State) (moment time.Moment) { return FIXED_MOMENT }

func pointer_realtime(state time.State) (moment time.Moment) {
  return time.Moment(*state.(*int64))
}

func recording_stream(buffer *recording_buffer) (stream nbio.Stream) {
  return nbio.Stream{State: buffer, Procedure: recording_procedure}
}

// Retires inline so a test reads the line right after the emit returns.
func recording_procedure(
  state nbio.State,
  completion *nbio.Completion,
  mode nbio.Stream_Mode,
  data []byte,
  _ int64,
  _ nbio.Seek_From,
  callback nbio.Stream_Callback,
) {
  completion.Data = 0
  completion.Error = nbio.Stream_Empty
  if mode == nbio.STREAM_MODE_WRITE {
    buffer := state.(*recording_buffer)
    *buffer = append(*buffer, data...)
    completion.Data = len(data)
    completion.Error = nil
  }
  nbio.Stream_Callback_Call(callback, completion)
}

func discard_stream() (stream nbio.Stream) {
  return nbio.Discard_To_Stream(&nbio.Stream_Discard{})
}

// Holds each write in flight until the test retires it, standing in for a kernel-backed Stream
// whose retirement the root delivers later.
type deferred_state struct {
  Buffer     recording_buffer
  Completion *nbio.Completion
  Callback   nbio.Stream_Callback
}

func deferred_procedure(
  state nbio.State,
  completion *nbio.Completion,
  _ nbio.Stream_Mode,
  data []byte,
  _ int64,
  _ nbio.Seek_From,
  callback nbio.Stream_Callback,
) {
  deferred := state.(*deferred_state)
  deferred.Buffer = append(deferred.Buffer, data...)
  completion.Data = len(data)
  completion.Error = nil
  deferred.Completion = completion
  deferred.Callback = callback
}

func deferred_retire(deferred *deferred_state) {
  completion := deferred.Completion
  deferred.Completion = nil
  nbio.Stream_Callback_Call(deferred.Callback, completion)
}

// Submits line to stream and fails the test on a retired error.
func write_stream(t *testing.T, stream nbio.Stream, line string) {
  t.Helper()
  var completion nbio.Completion
  retired := false
  nbio.Write(stream, &completion, []byte(line), func(_ nbio.Completion_Handle) {
    store_result(&retired, true)
  })
  if !retired {
    t.Fatal("stream must retire the write inline")
  }
  if completion.Error != nil {
    t.Fatalf("stream write: %v", completion.Error)
  }
}

func contains(source []byte, sought []byte) (present bool) {
  if len(sought) == 0 {
    return true
  }
  if len(sought) > len(source) {
    return false
  }
  last_start := len(source) - len(sought)
  for start_index := 0; !(start_index > last_start); start_index++ {
    match := true
    source_index := start_index
    for sought_index := 0; sought_index < len(sought); sought_index++ {
      if source[source_index] != sought[sought_index] {
        match = false
        break
      }
      source_index++
    }
    if match {
      return true
    }
  }
  return false
}

func float64_bits(value float64) (bits uint64) {
  return *(*uint64)(unsafe.Pointer(&value))
}

func float64_from_bits(bits uint64) (value float64) {
  return *(*float64)(unsafe.Pointer(&bits))
}

// TEST_LINE_BYTES is the line storage a test logger gets.
const TEST_LINE_BYTES = 4096

// TEST_PREFIX_BYTES is the prefix storage a test child gets.
const TEST_PREFIX_BYTES = 256

// Builds a logger over fresh configuration and line storage. A test allocates here so it needs
// no caller-owned variables; production roots keep both in a struct.
func build(configuration jlog.Logger_Configuration, floor jlog.Level) (logger *jlog.Logger) {
  stored := configuration
  built := jlog.New(&stored, make([]byte, TEST_LINE_BYTES), floor)
  return &built
}

// Builds a child of parent over fresh line and prefix storage.
func with(parent *jlog.Logger, fields ...jlog.Field) (child jlog.Logger) {
  storage := make([]byte, TEST_LINE_BYTES)
  prefix := jlog.Buffer{Bytes: make([]byte, TEST_PREFIX_BYTES)}
  return jlog.Logger_With(parent, storage, prefix, fields...)
}

// A logger writing JSON lines to buffer with the frozen clock and the lowest level
// floor, so every test line is emitted and timestamps are fixed.
func new_logger(buffer *recording_buffer) (logger *jlog.Logger) {
  return build(
    jlog.Logger_Configuration{
      Stream: recording_stream(buffer), Clock: frozen_clock(),
    },
    jlog.LEVEL_TRACE,
  )
}

// Fails the test unless buffer holds exactly want.
func assert_output(t *testing.T, buffer *recording_buffer, want string) {
  t.Helper()
  got := buffer_text(buffer)
  if got != want {
    t.Fatalf("got  %q\nwant %q", got, want)
  }
}

// Renders line through a Console with the given color setting and returns what the Console wrote,
// so a behaviour test is a single byte-for-byte comparison.
func render(t *testing.T, color bool, line string) (rendered *recording_buffer) {
  t.Helper()
  rendered = &recording_buffer{}
  console := new_console(recording_stream(rendered), color)
  write_stream(t, jlog.Console_To_Stream(&console), line)
  return rendered
}

// Runs line through a Level_Filter at floor and returns what passed, so a behaviour test is a
// single byte comparison. The inner writer is a plain buffer, so the test sees exactly which lines
// cleared the floor, unrendered.
func filter(t *testing.T, floor jlog.Level, line string) (passed *recording_buffer) {
  t.Helper()
  passed = &recording_buffer{}
  level_filter := new_level_filter(recording_stream(passed), floor)
  write_stream(t, jlog.Level_Filter_To_Stream(level_filter), line)
  return passed
}

// TEST_TRANSFORM_BYTES is the render, kept, and decoded storage a test transform gets.
const TEST_TRANSFORM_BYTES = 4096

// TEST_MEMBER_COUNT is the member storage a test transform gets.
const TEST_MEMBER_COUNT = 32

func new_object() (object jlog.Object) {
  return jlog.Object{
    Members: make([]jlog.Member, TEST_MEMBER_COUNT),
    Decoded: make([]byte, TEST_TRANSFORM_BYTES),
  }
}

// A Console over stream with fresh storage.
func new_console(stream nbio.Stream, color bool) (console jlog.Console) {
  return jlog.Console{
    Stream:   stream,
    Color:    color,
    Object:   new_object(),
    Rendered: make([]byte, TEST_TRANSFORM_BYTES),
  }
}

// A Level_Filter over stream at floor with fresh storage and the default level field.
func new_level_filter(stream nbio.Stream, floor jlog.Level) (filter *jlog.Level_Filter) {
  filter = &jlog.Level_Filter{
    Object: new_object(),
    Kept:   make([]byte, TEST_TRANSFORM_BYTES),
  }
  jlog.New_Level_Filter(filter, stream, floor, "")
  return filter
}

// The retirement a transform test's own submission takes; the test reads the completion after.
func disarm(completion nbio.Completion_Handle) {
  completion.Armed = false
}

// Every test clock returns this fixed reading so timestamp output is deterministic
// and assertable byte-for-byte.
const FIXED_MOMENT time.Moment = 1700000000000000000

// The exact benchmark message zerolog uses, so the benchmarks below are a 1:1
// workload comparison against zerolog's same-named benchmarks.
const FAKE_MESSAGE = "Test logging, but use a somewhat realistic message length."

// A logger discarding output with the frozen clock, for benchmarks.
func discard_logger() (logger *jlog.Logger) {
  return build(
    jlog.Logger_Configuration{
      Stream: discard_stream(), Clock: frozen_clock(),
    },
    jlog.LEVEL_TRACE,
  )
}

// Each parallel benchmark goroutine takes its own Logger, since a Logger has one owner.
func run_parallel(b *testing.B, shared *jlog.Logger, body func(logger *jlog.Logger)) {
  b.ReportAllocs()
  b.ResetTimer()
  b.RunParallel(func(pb *testing.PB) {
    run_owned(pb, shared, body)
  })
}

func store_result[T any](destination *T, value T) {
  *destination = value
}

func trace_stack(_ error) (stack string) {
  return "TRACE"
}

func short_stack(_ error) (stack string) {
  return "S"
}

func file_caller(_ int) (file string, line int) {
  return "file.go", 10
}

func short_caller(_ int) (file string, line int) {
  return "x.go", 1
}

func run_owned(pb *testing.PB, shared *jlog.Logger, body func(logger *jlog.Logger)) {
  own := with(shared)
  for pb.Next() {
    body(&own)
  }
}

// Benchmark_Log_Empty mirrors zerolog's BenchmarkLogEmpty.
func Benchmark_Log_Empty(b *testing.B) {
  run_parallel(b, discard_logger(), func(logger *jlog.Logger) {
    jlog.Logger_Log(logger, "")
  })
}

// Benchmark_Info mirrors zerolog's BenchmarkInfo.
func Benchmark_Info(b *testing.B) {
  run_parallel(b, discard_logger(), func(logger *jlog.Logger) {
    jlog.Logger_Info(logger, FAKE_MESSAGE)
  })
}

// Benchmark_Disabled mirrors zerolog's BenchmarkDisabled.
func Benchmark_Disabled(b *testing.B) {
  logger := build(
    jlog.Logger_Configuration{
      Stream: discard_stream(), Clock: frozen_clock(),
    },
    jlog.LEVEL_DISABLED,
  )
  run_parallel(b, logger, func(logger *jlog.Logger) {
    jlog.Logger_Info(logger, FAKE_MESSAGE)
  })
}

// Benchmark_Log_Fields mirrors zerolog's BenchmarkLogFields: a string, a time, an
// integer, and a float, plus the message.
func Benchmark_Log_Fields(b *testing.B) {
  run_parallel(b, discard_logger(), func(logger *jlog.Logger) {
    jlog.Logger_Info(
      logger,
      FAKE_MESSAGE,
      jlog.String("string", "four!"),
      jlog.Time("time", FIXED_MOMENT),
      jlog.Integer("int", 123),
      jlog.Float32("float", float32(-2.203230293249593)),
    )
  })
}

// Benchmark_Context_Fields mirrors zerolog's BenchmarkContextFields.
func Benchmark_Context_Fields(b *testing.B) {
  logger := with(
    discard_logger(),
    jlog.String("string", "four!"),
    jlog.Time("time", FIXED_MOMENT),
    jlog.Integer("int", 123),
    jlog.Float32("float", float32(-2.203230293249593)),
  )
  run_parallel(b, &logger, func(logger *jlog.Logger) {
    jlog.Logger_Info(logger, FAKE_MESSAGE)
  })
}

// Test_Cover_Levels exercises every level function and the level wire names.
func Test_Cover_Levels(t *testing.T) {
  buffer := &recording_buffer{}
  logger := new_logger(buffer)
  jlog.Logger_Trace(logger, "")
  jlog.Logger_Debug(logger, "")
  jlog.Logger_Warn(logger, "")
  jlog.Logger_Log(logger, "")
  jlog.Logger_At_Level(logger, jlog.LEVEL_ERROR, "")
  jlog.Logger_At_Level(logger, jlog.LEVEL_DISABLED, "")
  want := "{\"level\":\"trace\"}\n" +
    "{\"level\":\"debug\"}\n" +
    "{\"level\":\"warn\"}\n" +
    "{}\n" +
    "{\"level\":\"error\"}\n" +
    "{\"level\":\"disabled\"}\n"
  assert_output(t, buffer, want)
}

// Test_Cover_Level_String covers the cases not reached through emit.
func Test_Cover_Level_String(t *testing.T) {
  if jlog.Level_String(jlog.LEVEL_NONE) != "" {
    t.Fatalf("none = %q, want empty", jlog.Level_String(jlog.LEVEL_NONE))
  }
  testify.Panics(t, func() {
    t.Fatalf("invalid level rendered as %q", jlog.Level_String(jlog.Level(50)))
  })
}

// Test_Cover_Numeric_And_Slice_Fields covers the remaining scalar and array kinds.
func Test_Cover_Numeric_And_Slice_Fields(t *testing.T) {
  buffer := &recording_buffer{}
  jlog.Logger_Info(
    new_logger(buffer),
    "",
    jlog.Int64("a", int64(-5)),
    jlog.Uint("b", uint(6)),
    jlog.Float32("c", float32(1.5)),
    jlog.Floats64("d", []float64{1.5, 2.5}),
    jlog.Booleans("e", []bool{true, false}),
    jlog.Durations("f", []time.Duration{time.SECOND, 2 * time.SECOND}),
    jlog.Boolean("g", false),
  )
  want := "{\"level\":\"info\",\"a\":-5,\"b\":6,\"c\":1.5,\"d\":[1.5,2.5]," +
    "\"e\":[true,false],\"f\":[1000000000,2000000000],\"g\":false}\n"
  assert_output(t, buffer, want)
}

// Test_Cover_Err_Nil covers Err with a nil error and the skipped stack.
func Test_Cover_Err_Nil(t *testing.T) {
  buffer := &recording_buffer{}
  logger := build(
    jlog.Logger_Configuration{
      Stream:          recording_stream(buffer),
      Clock:           frozen_clock(),
      Stack_Marshaler: short_stack,
    },
    jlog.LEVEL_TRACE,
  )
  jlog.Logger_Error(logger, "", jlog.Err(nil))
  assert_output(t, buffer, "{\"level\":\"error\",\"error\":null}\n")
}

// Test_Cover_Caller_Without_Function covers Caller when no lookup is injected.
func Test_Cover_Caller_Without_Function(t *testing.T) {
  buffer := &recording_buffer{}
  jlog.Logger_Info(new_logger(buffer), "", jlog.Caller())
  assert_output(t, buffer, "{\"level\":\"info\"}\n")
}

// Test_Cover_Auto_Caller covers the per-line automatic caller.
func Test_Cover_Auto_Caller(t *testing.T) {
  buffer := &recording_buffer{}
  logger := build(
    jlog.Logger_Configuration{
      Stream: recording_stream(buffer),
      Clock:  frozen_clock(),
      Caller: short_caller,
    },
    jlog.LEVEL_TRACE,
  )
  logger.Auto_Caller = true
  jlog.Logger_Info(logger, "")
  assert_output(t, buffer, "{\"level\":\"info\",\"caller\":\"x.go:1\"}\n")
}

// Test_Cover_Disabled_Floor covers a logger whose floor disables all output, and a child of it
// staying disabled.
func Test_Cover_Disabled_Floor(t *testing.T) {
  buffer := &recording_buffer{}
  logger := build(
    jlog.Logger_Configuration{
      Stream: recording_stream(buffer), Clock: frozen_clock(),
    },
    jlog.LEVEL_DISABLED,
  )
  jlog.Logger_Info(logger, "x")
  child := with(logger, jlog.String("a", "1"))
  jlog.Logger_Info(&child, "y")
  assert_output(t, buffer, "")
}

// Test_Cover_Nested_With covers Logger_With on a logger that already has a prefix.
func Test_Cover_Nested_With(t *testing.T) {
  buffer := &recording_buffer{}
  parent := with(new_logger(buffer), jlog.String("a", "1"))
  child := with(&parent, jlog.String("b", "2"))
  jlog.Logger_Info(&child, "")
  assert_output(t, buffer, "{\"level\":\"info\",\"a\":\"1\",\"b\":\"2\"}\n")
}

// Test_Cover_Custom_Field_Names covers configured (non-default) key names.
func Test_Cover_Custom_Field_Names(t *testing.T) {
  buffer := &recording_buffer{}
  logger := build(
    jlog.Logger_Configuration{
      Stream:             recording_stream(buffer),
      Clock:              frozen_clock(),
      Level_Field_Name:   "lvl",
      Message_Field_Name: "msg",
    },
    jlog.LEVEL_TRACE,
  )
  jlog.Logger_Info(logger, "hi")
  assert_output(t, buffer, "{\"lvl\":\"info\",\"msg\":\"hi\"}\n")
}

// Test_Cover_Zero_Stream covers New accepting the zero Stream, which retires every line as
// Stream_Empty, and New refusing a missing configuration or missing storage.
func Test_Cover_Zero_Stream(t *testing.T) {
  logger := build(jlog.Logger_Configuration{Clock: frozen_clock()}, jlog.LEVEL_TRACE)
  jlog.Logger_Info(logger, "to nowhere")
  storage := make([]byte, TEST_LINE_BYTES)
  testify.Panics(t, func() { jlog.New(nil, storage, jlog.LEVEL_TRACE) })
  var configuration jlog.Logger_Configuration
  testify.Panics(t, func() { jlog.New(&configuration, nil, jlog.LEVEL_TRACE) })
}

// Test_Cover_String_Escape covers the escape path by round-tripping a value with
// control characters, a quote, a backslash, and a multibyte rune through the logger
// and back via JSON.
func Test_Cover_String_Escape(t *testing.T) {
  buffer := &recording_buffer{}
  value := "\b\f\n\r\t\"\\\x01世"
  jlog.Logger_Info(new_logger(buffer), "", jlog.String("a", value))
  assert_output(t, buffer,
    "{\"level\":\"info\",\"a\":\"\\b\\f\\n\\r\\t\\\"\\\\\\u0001世\"}\n")
}

// Test_Cover_Bytes_Escape covers the []byte escape path the same way.
func Test_Cover_Bytes_Escape(t *testing.T) {
  buffer := &recording_buffer{}
  value := "\t\"\\世"
  jlog.Logger_Info(new_logger(buffer), "", jlog.Bytes("a", []byte(value)))
  assert_output(t, buffer, "{\"level\":\"info\",\"a\":\"\\t\\\"\\\\世\"}\n")
}

// Test_Cover_Invalid_UTF8 covers the replacement-character path for an invalid byte
// in both a string and a []byte value, and pins the wire form to the escaped
// replacement sequence so it stays byte-for-byte identical to zerolog on malformed input.
func Test_Cover_Invalid_UTF8(t *testing.T) {
  buffer := &recording_buffer{}
  jlog.Logger_Info(
    new_logger(buffer),
    "",
    jlog.String("s", "\xff"),
    jlog.Bytes("b", []byte("ok\xff")),
  )
  wire := []byte(*buffer)
  if !contains(wire, []byte{'\\', 'u', 'f', 'f', 'f', 'd'}) {
    t.Fatalf("invalid byte must escape, got %q", buffer_text(buffer))
  }
  if contains(wire, []byte("�")) {
    t.Fatalf("output must not carry the raw replacement rune: %q", buffer_text(buffer))
  }
  assert_output(t, buffer,
    "{\"level\":\"info\",\"s\":\"\\ufffd\",\"b\":\"ok\\ufffd\"}\n")
}

// Test_Cover_Floats covers NaN, both infinities, zero, and exponent formatting at
// float32 and float64 precision.
func Test_Cover_Floats(t *testing.T) {
  buffer := &recording_buffer{}
  jlog.Logger_Info(
    new_logger(buffer),
    "",
    jlog.Float64("nan", float64_from_bits(0x7ff8000000000001)),
    jlog.Float64("pinf", float64_from_bits(0x7ff0000000000000)),
    jlog.Float64("ninf", float64_from_bits(0xfff0000000000000)),
    jlog.Float64("zero", float64(0)),
    jlog.Float64("small", float64(1e-7)),
    jlog.Float64("big", float64(1e21)),
    jlog.Float32("small32", float32(1e-7)),
    jlog.Float32("big32", float32(1e21)),
  )
  want := "{\"level\":\"info\",\"nan\":\"NaN\",\"pinf\":\"+Inf\",\"ninf\":\"-Inf\"," +
    "\"zero\":0,\"small\":1e-7,\"big\":1e+21,\"small32\":1e-7,\"big32\":1e+21}\n"
  assert_output(t, buffer, want)
}

// Test_Cover_Pre_Epoch_Time covers a timestamp before the Unix epoch, exercising
// the negative-fraction and negative-day paths of the RFC 3339 conversion.
func Test_Cover_Pre_Epoch_Time(t *testing.T) {
  buffer := &recording_buffer{}
  jlog.Logger_Info(new_logger(buffer), "", jlog.Time("t", time.Moment(-1)))
  want := "{\"level\":\"info\",\"t\":\"1969-12-31T23:59:59.999999999Z\"}\n"
  assert_output(t, buffer, want)
}

// Test_Cover_Unknown_Kind covers the defensive path for an unrecognized field kind.
func Test_Cover_Unknown_Kind(t *testing.T) {
  buffer := &recording_buffer{}
  jlog.Logger_Info(new_logger(buffer), "", jlog.Field{Key: "x", Kind: 100})
  assert_output(t, buffer, "{\"level\":\"info\"}\n")
}

// Test_Cover_Prefix_Without_Level covers merging a sub-logger prefix into a line
// that carries no level field, the no-separator branch of the object-data merge.
func Test_Cover_Prefix_Without_Level(t *testing.T) {
  buffer := &recording_buffer{}
  child := with(new_logger(buffer), jlog.String("a", "1"))
  jlog.Logger_Log(&child, "")
  assert_output(t, buffer, "{\"a\":\"1\"}\n")
}

// Test_Cover_Prefix_Too_Large covers Logger_With refusing a prefix that outgrows its storage.
func Test_Cover_Prefix_Too_Large(t *testing.T) {
  big := make([]byte, TEST_PREFIX_BYTES)
  for index := range big {
    big[index] = 'a'
  }
  logger := discard_logger()
  testify.Panics(t, func() { with(logger, jlog.String("x", string(big))) })
}

// Test_Cover_Float_Exponent_No_Trim covers an exponent whose second digit is not a
// zero, so the leading-zero trim is skipped.
func Test_Cover_Float_Exponent_No_Trim(t *testing.T) {
  buffer := &recording_buffer{}
  jlog.Logger_Info(new_logger(buffer), "", jlog.Float64("x", 1e-17))
  assert_output(t, buffer, "{\"level\":\"info\",\"x\":1e-17}\n")
}

// Fuzz_Encode throws random field values at the logger and asserts every line is valid
// JSON. It drives the string-escape paths (random and invalid-UTF-8 bytes), the float
// NaN/Inf/exponent paths, and — via the fuzzed moment feeding both Auto_Timestamp and a
// Time field — civil_from_days across the int64 epoch range, so the internal asserts that
// guard the civil-date math are proven; any regression panics with the named invariant.
func Fuzz_Encode(f *testing.F) {
  f.Add([]byte("hello"), int64(1700000000000000000), int64(42), float64_bits(3.14))
  f.Fuzz(fuzz_encode)
}

func fuzz_encode(t *testing.T, raw []byte, moment int64, number int64, float_bits uint64) {
  buffer := &recording_buffer{}
  host := time.Clock{
    State:         &moment,
    Now_Monotonic: zero_monotonic,
    Now_Realtime:  pointer_realtime,
  }
  logger := build(
    jlog.Logger_Configuration{
      Stream: recording_stream(buffer), Clock: host,
    },
    jlog.LEVEL_TRACE,
  )
  logger.Auto_Timestamp = true
  jlog.Logger_Info(
    logger,
    string(raw),
    jlog.String("s", string(raw)),
    jlog.Bytes("b", raw),
    jlog.Int64("n", number),
    jlog.Float64("f", float64_from_bits(float_bits)),
    jlog.Time("t", time.Moment(moment)),
    jlog.Duration("d", time.Duration(number)),
  )
  if logger.Dropped == 1 {
    return
  }
  line := []byte(*buffer)
  position, status := json.Validate(json.Encoded(line[:len(line)-1]))
  if !status {
    t.Fatalf("invalid JSON %q at %d", line, position)
  }
}
