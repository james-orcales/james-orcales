package diode_test

import "sync"
import "testing"
import "unsafe"

import "local/james-orcales/shared/sim/nbio"
import "local/james-orcales/shared/sim/time"
import "local/james-orcales/shared/strconv"
import "local/james-orcales/shared/sync/diode"
import "local/james-orcales/shared/testify"

// Test_Write_Forwards_To_Sink checks that a written line reaches the inner Stream once the root
// rearms.
func Test_Write_Forwards_To_Sink(t *testing.T) {
  sink := &recording_sink{}
  writer, stream := new_diode(8, diode.New_Input{Stream: recording_stream(sink)})
  var completion nbio.Completion
  write(stream, &completion, "hello")
  assert_lines(t, sink.Lines, nil...)
  rearm(writer)
  assert_lines(t, sink.Lines, "hello")
}

// Test_Write_Does_Not_Block checks that Write retires at once and never touches the inner
// Stream, whether that Stream is idle or holds a submission in flight.
func Test_Write_Does_Not_Block(t *testing.T) {
  sink := &deferred_sink{}
  writer, stream := new_diode(4, diode.New_Input{Stream: deferred_stream(sink)})
  var completion nbio.Completion
  for index := 0; index < 16; index++ {
    if index == 8 {
      rearm(writer)
    }
    write(stream, &completion, "x")
    if completion.Data != 1 {
      t.Fatalf("write %d retired n=%d, want 1", index, completion.Data)
    }
    if completion.Error != nil {
      t.Fatalf("write %d retired err=%v, want nil", index, completion.Error)
    }
  }
  if len(sink.Lines) != 1 {
    t.Fatalf("inner Stream saw %d lines, want the one rearmed", len(sink.Lines))
  }
}

// Test_Overflow_Drops_Oldest checks that overrunning capacity keeps the newest entries.
func Test_Overflow_Drops_Oldest(t *testing.T) {
  sink := &deferred_sink{}
  writer, stream := new_diode(4, diode.New_Input{Stream: deferred_stream(sink)})
  var completion nbio.Completion
  for index := 0; index < 8; index++ {
    write(stream, &completion, decimal(index))
  }
  drain(writer, sink)
  assert_lines(t, sink.Lines, "4", "5", "6", "7")
}

// Test_Drop_Count_Is_Reported checks that the Alerter receives the overwritten count.
func Test_Drop_Count_Is_Reported(t *testing.T) {
  alerts := &alert_recorder{}
  sink := &deferred_sink{}
  writer, stream := new_diode(
    4,
    diode.New_Input{
      Stream:  deferred_stream(sink),
      Alerter: func(count int, cause diode.Drop_Cause) { alert_record(alerts, count, cause) },
    },
  )
  var completion nbio.Completion
  for index := 0; index < 8; index++ {
    write(stream, &completion, decimal(index))
  }
  drain(writer, sink)
  if len(alerts.Missed) != 1 {
    t.Fatalf("alerter reported %v, want [4]", alerts.Missed)
  }
  if alerts.Missed[0] != 4 {
    t.Fatalf("alerter reported %v, want [4]", alerts.Missed)
  }
  if alerts.Causes[0] != diode.DROP_OVERFLOW {
    t.Fatalf("drop cause %v, want DROP_OVERFLOW", alerts.Causes[0])
  }
}

// Test_Order_Is_Preserved checks that delivered entries keep write order.
func Test_Order_Is_Preserved(t *testing.T) {
  sink := &recording_sink{}
  writer, stream := new_diode(16, diode.New_Input{Stream: recording_stream(sink)})
  var completion nbio.Completion
  want := make([]string, 10)
  for index := range want {
    want[index] = decimal(index)
    write(stream, &completion, want[index])
  }
  rearm(writer)
  assert_lines(t, sink.Lines, want...)
}

// Test_Root_Rearms_The_Drain checks that one Writer_Rearm submits one line, that it submits
// nothing while that line is in flight, and that the retirement itself continues nothing.
func Test_Root_Rearms_The_Drain(t *testing.T) {
  sink := &deferred_sink{}
  writer, stream := new_diode(8, diode.New_Input{Stream: deferred_stream(sink)})
  var completion nbio.Completion
  write(stream, &completion, "a")
  write(stream, &completion, "b")
  if !diode.Writer_Work_Queued(writer) {
    t.Fatal("work must be queued after a write")
  }
  if !diode.Writer_Rearm(writer) {
    t.Fatal("first rearm must submit")
  }
  assert_lines(t, sink.Lines, "a")
  if diode.Writer_Rearm(writer) {
    t.Fatal("nothing may submit while a line is in flight")
  }
  // The root pumps until nothing is outstanding, and a line in flight is outstanding.
  if !diode.Writer_Work_Queued(writer) {
    t.Fatal("work is outstanding while a line is in flight")
  }
  retire_one(sink)
  assert_lines(t, sink.Lines, "a")
  if !diode.Writer_Rearm(writer) {
    t.Fatal("rearm after retirement must submit")
  }
  assert_lines(t, sink.Lines, "a", "b")
  retire_one(sink)
  assert_idle(t, writer, "an empty ring has nothing to submit")
}

// Test_Close_Flushes_And_Stops checks that buffered entries reach the Stream before Close
// reaches it, and that the caller's Close retires only after that.
func Test_Close_Flushes_And_Stops(t *testing.T) {
  sink := &deferred_sink{}
  writer, stream := new_diode(8, diode.New_Input{Stream: deferred_stream(sink)})
  var completion nbio.Completion
  for index := 0; index < 3; index++ {
    write(stream, &completion, decimal(index))
  }
  var close_completion nbio.Completion
  close_retired := false
  nbio.Close(stream, &close_completion, func(_ nbio.Completion_Handle) {
    flag_set(&close_retired)
  })
  if sink.Closed {
    t.Fatal("Close must record and return, never drive")
  }
  if close_retired {
    t.Fatal("Close must record and return, never drive")
  }
  drain(writer, sink)
  assert_lines(t, sink.Lines, "0", "1", "2")
  if !sink.Closed {
    t.Fatal("inner Stream must be closed after the flush")
  }
  if !close_retired {
    t.Fatal("the caller's Close must retire with the inner Close")
  }
  if close_completion.Error != nil {
    t.Fatalf("close err=%v", close_completion.Error)
  }
  write(stream, &completion, "late")
  if completion.Error != nbio.Stream_Empty {
    t.Fatalf("write after close err=%v, want Stream_Empty", completion.Error)
  }
  // A second Close retires inline as a no-op, and a closed diode has no work.
  nbio.Close(stream, &close_completion, retired)
  assert_idle(t, writer, "a closed diode has nothing to submit")
}

// Test_Dropping_Does_Not_Allocate checks that overwriting unread entries recycles buckets.
func Test_Dropping_Does_Not_Allocate(t *testing.T) {
  sink := &deferred_sink{}
  _, stream := new_diode(8, diode.New_Input{Stream: deferred_stream(sink)})
  var completion nbio.Completion
  line := []byte("a dropped line")
  // Warm past one full lap so the pool reaches steady-state recycling; nothing rearms, so
  // every write here drops.
  for index := 0; index < 64; index++ {
    nbio.Write(stream, &completion, line, retired)
  }
  testify.Zero_Allocation(t, func() {
    nbio.Write(stream, &completion, line, retired)
  })
  // The Stream's procedure is a func literal that captures nothing: a static value, so
  // building the Stream allocates nothing either.
  writer := &diode.Writer{}
  *writer = diode.New(
    unsafe.Pointer(&diode.Writer_State{}),
    diode.New_Input{
      Clock:   instant_clock(),
      Slots:   make(diode.Slots, 1),
      Buckets: make(diode.Buckets, diode.BUCKET_COUNT_MINIMUM),
      Arena:   make(diode.Line_Arena, diode.LINE_ARENA_SIZE_MINIMUM),
    },
  )
  testify.Zero_Allocation(t, func() {
    diode.Writer_To_Stream(writer)
  })
}

// Test_Rate_Limit_Sheds_By_Bytes checks the byte budget and clock-driven refill.
func Test_Rate_Limit_Sheds_By_Bytes(t *testing.T) {
  sink := &recording_sink{}
  var causes []diode.Drop_Cause
  writer, stream := new_diode(
    16,
    diode.New_Input{
      Stream:     recording_stream(sink),
      Clock:      instant_clock(),
      Rate_Limit: diode.Rate_Limit{Bytes_Per_Second: 1, Burst: 6},
      Alerter: func(missed int, cause diode.Drop_Cause) {
        append_causes(&causes, missed, cause)
      },
    },
  )
  var completion nbio.Completion
  for index := 0; index < 8; index++ {
    write(stream, &completion, "ab")
  }
  rearm(writer)
  assert_lines(t, sink.Lines, "ab", "ab", "ab")
  if len(causes) != 5 {
    t.Fatalf("shed %d lines, want 5", len(causes))
  }
  for _, cause := range causes {
    if cause != diode.DROP_RATE_LIMIT {
      t.Fatalf("shed cause %v, want DROP_RATE_LIMIT", cause)
    }
  }

  steady_sink := &recording_sink{}
  steady_writer, steady := new_diode(
    16,
    diode.New_Input{
      Stream:     recording_stream(steady_sink),
      Clock:      stepping_clock(time.SECOND),
      Rate_Limit: diode.Rate_Limit{Bytes_Per_Second: 1000, Burst: 2},
      Alerter: func(missed int, cause diode.Drop_Cause) {
        t.Errorf("unexpected drop of %d (%v); tokens should refill", missed, cause)
      },
    },
  )
  for index := 0; index < 8; index++ {
    write(steady, &completion, "ab")
  }
  rearm(steady_writer)
  if len(steady_sink.Lines) != 8 {
    t.Fatalf("steady delivered %d, want 8", len(steady_sink.Lines))
  }
}

// Test_Invariant_Boundaries drives every accepted configuration boundary through New.
func Test_Invariant_Boundaries(t *testing.T) {
  for _, boundary := range [...]struct {
    Count int
    Input diode.New_Input
  }{
    {
      1,
      diode.New_Input{
        Rate_Limit: diode.Rate_Limit{
          Bytes_Per_Second: diode.CONFIGURATION_MINIMUM,
          Burst:            diode.CONFIGURATION_MINIMUM,
        },
      },
    },
    {1, diode.New_Input{}},
    {1, diode.New_Input{Rate_Limit: diode.Rate_Limit{Bytes_Per_Second: 1, Burst: 1}}},
    {2, diode.New_Input{Rate_Limit: diode.Rate_Limit{Bytes_Per_Second: 2, Burst: 2}}},
    {
      diode.SLOT_COUNT_MAXIMUM,
      diode.New_Input{
        Rate_Limit: diode.Rate_Limit{
          Bytes_Per_Second: diode.BYTE_RATE_MAXIMUM,
          Burst:            diode.BURST_SIZE_MAXIMUM,
        },
      },
    },
  } {
    writer, stream := new_diode(boundary.Count, boundary.Input)
    var completion nbio.Completion
    for _, size := range [...]int{0, 1, 2, diode.LINE_BYTES_MAXIMUM} {
      nbio.Write(stream, &completion, make([]byte, size), retired)
      if completion.Error != nil {
        t.Fatalf("boundary write of %d bytes: %v", size, completion.Error)
      }
      if completion.Data != size {
        t.Fatalf("boundary write = %d, want %d", completion.Data, size)
      }
    }
    oversized := [...]int{diode.LINE_BYTES_MAXIMUM + 1, diode.DATA_SIZE_MAXIMUM}
    for _, size := range oversized {
      nbio.Write(stream, &completion, make([]byte, size), retired)
      if completion.Error != diode.ERR_DATA_TOO_LARGE {
        t.Fatalf("oversized write err=%v, want too large", completion.Error)
      }
    }
    rearm(writer)
  }
  for _, moment := range [...]time.Monotonic_Moment{
    time.MONOTONIC_MOMENT_MINIMUM,
    1,
    2,
    time.MONOTONIC_MOMENT_MAXIMUM,
  } {
    new_diode(1, diode.New_Input{Clock: constant_clock(moment)})
  }
  testify.Panics(t, func() {
    diode.Writer_To_Stream(&diode.Writer{})
  }, "zero writer")
  assert_storage_refused(t)
}

// Test_Maximum_Drop_Report proves one observed lap cannot overflow its bounded report.
func Test_Maximum_Drop_Report(t *testing.T) {
  var missed []int
  sink := &deferred_sink{}
  writer, stream := new_diode(
    1,
    diode.New_Input{
      Stream:  deferred_stream(sink),
      Alerter: func(count int, _ diode.Drop_Cause) { append_count(&missed, count) },
    },
  )
  var completion nbio.Completion
  for write_count := 0; !(write_count > diode.MISSED_COUNT_MAXIMUM); write_count++ {
    write(stream, &completion, "x")
  }
  diode.Writer_Rearm(writer)
  if len(missed) != 1 {
    t.Fatalf("maximum drop report = %v, want [%d]", missed, diode.MISSED_COUNT_MAXIMUM)
  }
  if missed[0] != diode.MISSED_COUNT_MAXIMUM {
    t.Fatalf("maximum drop report = %v, want [%d]", missed, diode.MISSED_COUNT_MAXIMUM)
  }
}

// Test_Rate_Limit_Survives_A_Large_Clock guards the refill math against int64 overflow.
func Test_Rate_Limit_Survives_A_Large_Clock(t *testing.T) {
  sink := &recording_sink{}
  writer, stream := new_diode(
    16,
    diode.New_Input{
      Stream:     recording_stream(sink),
      Clock:      stepping_clock(10_000 * time.SECOND),
      Rate_Limit: diode.Rate_Limit{Bytes_Per_Second: 1 << 20, Burst: 1 << 20},
      Alerter: func(missed int, cause diode.Drop_Cause) {
        t.Errorf("unexpected drop of %d (%v) under a large host", missed, cause)
      },
    },
  )
  var completion nbio.Completion
  for index := 0; index < 4; index++ {
    write(stream, &completion, "ab")
  }
  rearm(writer)
  assert_lines(t, sink.Lines, "ab", "ab", "ab", "ab")
}

// Alert_recorder keeps each Alerter report in call order.
type alert_recorder struct {
  Missed []int
  Causes []diode.Drop_Cause
}

func alert_record(recorder *alert_recorder, count int, cause diode.Drop_Cause) {
  recorder.Missed = append(recorder.Missed, count)
  recorder.Causes = append(recorder.Causes, cause)
}

func flag_set(flag *bool) {
  *flag = true
}

// Storage is caller-provided, so New must refuse pieces that do not fit one another.
func assert_storage_refused(t *testing.T) {
  t.Helper()
  testify.Panics(t, func() {
    diode.New(
      unsafe.Pointer(&diode.Writer_State{}),
      diode.New_Input{
        Clock:   instant_clock(),
        Slots:   make(diode.Slots, 1),
        Buckets: make(diode.Buckets, diode.BUCKET_COUNT_MINIMUM-1),
        Arena:   make(diode.Line_Arena, diode.LINE_ARENA_SIZE_MINIMUM),
      },
    )
  }, "buckets short of slots plus slack")
  testify.Panics(t, func() {
    diode.New(
      unsafe.Pointer(&diode.Writer_State{}),
      diode.New_Input{
        Clock:   instant_clock(),
        Slots:   make(diode.Slots, 1),
        Buckets: make(diode.Buckets, diode.BUCKET_COUNT_MINIMUM),
        Arena:   make(diode.Line_Arena, diode.LINE_ARENA_SIZE_MINIMUM-1),
      },
    )
  }, "arena short of one line per bucket")
  testify.Panics(t, func() {
    diode.New(
      nil,
      diode.New_Input{
        Clock:   instant_clock(),
        Slots:   make(diode.Slots, 1),
        Buckets: make(diode.Buckets, diode.BUCKET_COUNT_MINIMUM),
        Arena:   make(diode.Line_Arena, diode.LINE_ARENA_SIZE_MINIMUM),
      },
    )
  }, "nil storage")
}

func append_causes(destination *[]diode.Drop_Cause, count int, cause diode.Drop_Cause) {
  for index := 0; index < count; index++ {
    *destination = append(*destination, cause)
  }
}

func append_count(destination *[]int, count int) {
  *destination = append(*destination, count)
}

// Builds a diode over count ring slots, supplying a zero clock when the test does not care
// about time. Tests may allocate; the library never does, so storage is built here.
func new_diode(
  count int, input diode.New_Input,
) (writer *diode.Writer, stream nbio.Stream) {
  if input.Clock.Now_Monotonic == nil {
    input.Clock = instant_clock()
  }
  bucket_count := count + diode.BUCKET_SLACK
  input.Slots = make(diode.Slots, count)
  input.Buckets = make(diode.Buckets, bucket_count)
  input.Arena = make(diode.Line_Arena, bucket_count*diode.LINE_BYTES_MAXIMUM)
  handle := diode.New(unsafe.Pointer(&diode.Writer_State{}), input)
  writer = &handle
  return writer, diode.Writer_To_Stream(writer)
}

func write(stream nbio.Stream, completion *nbio.Completion, line string) {
  nbio.Write(stream, completion, []byte(line), retired)
}

// The retirement every test submission takes. A test reads the completion it owns after the
// call returns, so the callback only marks the handle it was given as delivered.
func retired(completion nbio.Completion_Handle) {
  completion.Armed = false
}

// The root's loop: rearm until nothing submits. Bounded by the ring, since each pass takes one
// entry and an inline inner Stream retires it before the next pass.
func rearm(writer *diode.Writer) {
  for pass_index := 0; !(pass_index > diode.SLOT_COUNT_MAXIMUM); pass_index++ {
    if !diode.Writer_Rearm(writer) {
      return
    }
  }
}

// The root's loop over a deferred inner Stream: rearm, then deliver the retirement the loop
// would have delivered, until both are quiet.
func drain(writer *diode.Writer, sink *deferred_sink) {
  for pass_index := 0; !(pass_index > diode.SLOT_COUNT_MAXIMUM); pass_index++ {
    submitted := diode.Writer_Rearm(writer)
    if len(sink.Queue) > 0 {
      retire_one(sink)
      continue
    }
    if !submitted {
      return
    }
  }
}

// Fails the test unless neither Writer_Rearm nor Writer_Work_Queued finds work.
func assert_idle(t *testing.T, writer *diode.Writer, reason string) {
  t.Helper()
  if diode.Writer_Rearm(writer) {
    t.Fatal(reason)
  }
  if diode.Writer_Work_Queued(writer) {
    t.Fatal(reason)
  }
}

func assert_lines(t *testing.T, got []string, want ...string) {
  t.Helper()
  if len(got) != len(want) {
    t.Fatalf("lines %q, want %q", got, want)
  }
  for index := range want {
    if got[index] != want[index] {
      t.Fatalf("lines %q, want %q", got, want)
    }
  }
}

// A Stream that retires every operation inline, like nbio's memory Stream.
type recording_sink struct {
  // Lines holds one copy of each forwarded line.
  Lines []string
}

func recording_stream(sink *recording_sink) (stream nbio.Stream) {
  return nbio.Stream{State: sink, Procedure: recording_procedure}
}

func recording_procedure(
  state nbio.State,
  completion *nbio.Completion,
  mode nbio.Stream_Mode,
  data []byte,
  _ int64,
  _ nbio.Seek_From,
  callback nbio.Stream_Callback,
) {
  sink := state.(*recording_sink)
  completion.Data = 0
  completion.Error = nil
  switch mode {
  case nbio.STREAM_MODE_WRITE:
    sink.Lines = append(sink.Lines, string(data))
    completion.Data = len(data)
  case nbio.STREAM_MODE_CLOSE, nbio.STREAM_MODE_FLUSH:
  default:
    completion.Error = nbio.Stream_Empty
  }
  nbio.Stream_Callback_Call(callback, completion)
}

// A Stream that holds each write in flight until the test retires it, standing in for a
// kernel-backed Stream whose retirement the root delivers on a later pass.
type deferred_sink struct {
  // Lines holds each submitted line in submission order.
  Lines []string
  // Queue holds submitted writes not yet retired.
  Queue []deferred_write
  // Closed records that Close reached this Stream.
  Closed bool
}

type deferred_write struct {
  Completion *nbio.Completion
  Callback   nbio.Stream_Callback
}

func deferred_stream(sink *deferred_sink) (stream nbio.Stream) {
  return nbio.Stream{State: sink, Procedure: deferred_procedure}
}

func deferred_procedure(
  state nbio.State,
  completion *nbio.Completion,
  mode nbio.Stream_Mode,
  data []byte,
  _ int64,
  _ nbio.Seek_From,
  callback nbio.Stream_Callback,
) {
  sink := state.(*deferred_sink)
  completion.Data = 0
  completion.Error = nil
  if mode == nbio.STREAM_MODE_WRITE {
    sink.Lines = append(sink.Lines, string(data))
    completion.Data = len(data)
    sink.Queue = append(sink.Queue, deferred_write{
      Completion: completion, Callback: callback,
    })
    return
  }
  if mode == nbio.STREAM_MODE_CLOSE {
    sink.Closed = true
  } else {
    completion.Error = nbio.Stream_Empty
  }
  nbio.Stream_Callback_Call(callback, completion)
}

func retire_one(sink *deferred_sink) {
  head := sink.Queue[0]
  sink.Queue = sink.Queue[1:]
  nbio.Stream_Callback_Call(head.Callback, head.Completion)
}

func instant_clock() (host time.Clock) {
  return time.Clock{
    Now_Monotonic: zero_monotonic,
    Now_Realtime:  zero_realtime,
  }
}

func constant_clock(moment time.Monotonic_Moment) (host time.Clock) {
  return time.Clock{
    State:         &moment,
    Now_Monotonic: constant_monotonic,
    Now_Realtime:  zero_realtime,
  }
}

func constant_monotonic(state time.State) (moment time.Monotonic_Moment) {
  return *state.(*time.Monotonic_Moment)
}

func zero_monotonic(_ time.State) (moment time.Monotonic_Moment) { return 0 }

func zero_realtime(_ time.State) (moment time.Moment) { return 0 }

// A read-only clock whose monotonic reading advances by step on every read, so a test can
// drive the rate limiter's token refill deterministically.
func stepping_clock(step time.Duration) (host time.Clock) {
  state := stepping_clock_state{Step: step}
  return time.Clock{
    State:         &state,
    Now_Monotonic: stepping_clock_now_monotonic,
    Now_Realtime:  zero_realtime,
  }
}

type stepping_clock_state struct {
  Elapsed time.Monotonic_Moment
  Step    time.Duration
}

func stepping_clock_now_monotonic(state time.State) (moment time.Monotonic_Moment) {
  host := state.(*stepping_clock_state)
  host.Elapsed += time.Monotonic_Moment(host.Step)
  return host.Elapsed
}

// Renders index as its base-ten string, a stable per-entry marker for tests.
func decimal(index int) (text string) {
  var storage [strconv.DECIMAL_TEXT_SIZE_MAXIMUM]byte
  count := strconv.Format_Decimal_Into(storage[:], strconv.Machine_Integer(index))
  return string(storage[:count])
}

// Matches the realistic message length used across the repo's logging benchmarks so
// diode numbers compare against jlog's.
const BENCHMARK_LINE = "Test logging, but use a somewhat realistic message length."

// Benchmark_Write measures the producer cost alone, with a roomy ring nobody rearms.
func Benchmark_Write(b *testing.B) {
  _, stream := new_diode(
    1024,
    diode.New_Input{
      Stream: nbio.Discard_To_Stream(&nbio.Stream_Discard{}),
    },
  )
  line := []byte(BENCHMARK_LINE)
  b.ReportAllocs()
  b.ResetTimer()
  b.RunParallel(func(pb *testing.PB) {
    benchmark_write(pb, stream, line)
  })
}

func benchmark_write(pb *testing.PB, stream nbio.Stream, line []byte) {
  var completion nbio.Completion
  for pb.Next() {
    nbio.Write(stream, &completion, line, retired)
  }
}

// Benchmark_Write_Full_Ring measures the producer cost when a tiny ring is saturated, the worst
// case for collision retries and bucket churn.
func Benchmark_Write_Full_Ring(b *testing.B) {
  _, stream := new_diode(8, diode.New_Input{Stream: deferred_stream(&deferred_sink{})})
  line := []byte(BENCHMARK_LINE)
  var completion nbio.Completion
  b.ReportAllocs()
  b.ResetTimer()
  for index := 0; index < b.N; index++ {
    nbio.Write(stream, &completion, line, retired)
  }
}

// The diode's inner Stream during fuzzing: it decodes each delivered line back to its
// (producer, sequence) and records the first invariant it sees broken. Only the root goroutine,
// which rearms, touches it.
type ring_witness struct {
  // Producers is how many concurrent writers this run spawned.
  Producers int
  // Writes is how many lines each producer emits.
  Writes int
  // Last is the highest sequence delivered per producer, -1 before any.
  Last []int
  // Delivered counts the lines forwarded to this sink.
  Delivered int
  // Fault holds the first broken invariant observed, empty when clean.
  Fault string
}

func decode_word_32(data []byte) (value uint32) {
  for byte_index := len(data) - 1; !(byte_index < 0); byte_index-- {
    value = value<<8 | uint32(data[byte_index])
  }
  return value
}

func encode_word_32(data []byte, value uint32) {
  for byte_index := range data {
    data[byte_index] = byte(value)
    value >>= 8
  }
}

func ring_witness_procedure(
  state nbio.State,
  completion *nbio.Completion,
  mode nbio.Stream_Mode,
  data []byte,
  _ int64,
  _ nbio.Seek_From,
  callback nbio.Stream_Callback,
) {
  witness := state.(*ring_witness)
  completion.Data = len(data)
  completion.Error = nil
  if mode == nbio.STREAM_MODE_WRITE {
    ring_witness_check(witness, data)
  }
  nbio.Stream_Callback_Call(callback, completion)
}

func ring_witness_check(witness *ring_witness, p []byte) {
  if len(p) != 8 {
    note_fault(witness, "torn line length")
    return
  }
  producer := int(decode_word_32(p[0:4]))
  sequence := int(decode_word_32(p[4:8]))
  if !(producer < witness.Producers) {
    note_fault(witness, "producer id out of range")
    return
  }
  if !(sequence < witness.Writes) {
    note_fault(witness, "sequence out of range")
    return
  }
  if !(sequence > witness.Last[producer]) {
    note_fault(witness, "duplicate or out-of-order delivery")
    return
  }
  witness.Last[producer] = sequence
  witness.Delivered++
}

// Records the first broken invariant; later ones stay quiet so the failure the test
// reports is the earliest and most diagnostic.
func note_fault(witness *ring_witness, fault string) {
  if witness.Fault != "" {
    return
  }
  witness.Fault = fault
}

// Fuzz_Ring drives many producers into a tiny ring while the root rearms concurrently, so
// overwrites, collision retries, and the recycle path all fire against a live drain. Every
// delivered line must decode to a real (producer, sequence), and each producer's deliveries
// must be strictly increasing.
func Fuzz_Ring(f *testing.F) {
  f.Add(uint8(0), uint16(3), uint32(400))
  f.Add(uint8(7), uint16(1), uint32(900))
  f.Fuzz(fuzz_ring)
}

func fuzz_ring(t *testing.T, ring_byte uint8, producer_word uint16, write_word uint32) {
  count := 1 + int(ring_byte%8)
  producer_count := 1 + int(producer_word%8)
  writes := 1 + int(write_word%256)
  witness := &ring_witness{
    Producers: producer_count,
    Writes:    writes,
    Last:      make([]int, producer_count),
  }
  for index := range witness.Last {
    witness.Last[index] = -1
  }
  writer, stream := new_diode(
    count,
    diode.New_Input{
      Stream:  nbio.Stream{State: witness, Procedure: ring_witness_procedure},
      Alerter: func(missed int, cause diode.Drop_Cause) {},
    },
  )
  var crew sync.WaitGroup
  for producer_index := 0; producer_index < producer_count; producer_index++ {
    crew.Add(1)
    go ring_producer(&crew, stream, producer_index, writes)
  }
  finished := make(chan struct{})
  go ring_wait(&crew, finished)
  // The root rearms while producers write, then once more after they stop.
  active := true
  for active {
    select {
    case <-finished:
      active = false
    default:
      diode.Writer_Rearm(writer)
    }
  }
  rearm(writer)
  var completion nbio.Completion
  nbio.Close(stream, &completion, retired)
  rearm(writer)
  if witness.Fault != "" {
    t.Fatalf("%s (count=%d producers=%d writes=%d)",
      witness.Fault, count, producer_count, writes)
  }
  if witness.Delivered > producer_count*writes {
    t.Fatalf("delivered %d exceeds %d written",
      witness.Delivered, producer_count*writes)
  }
}

func ring_producer(crew *sync.WaitGroup, stream nbio.Stream, identifier int, writes int) {
  defer crew.Done()
  var completion nbio.Completion
  line := make([]byte, 8)
  encode_word_32(line[0:4], uint32(identifier))
  for sequence_index := 0; sequence_index < writes; sequence_index++ {
    encode_word_32(line[4:8], uint32(sequence_index))
    nbio.Write(stream, &completion, line, retired)
  }
}

func ring_wait(crew *sync.WaitGroup, finished chan struct{}) {
  crew.Wait()
  close(finished)
}
