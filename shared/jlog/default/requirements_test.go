package jlog_test

import "errors"
import "testing"

import library "local/james-orcales/shared/jlog"
import "local/james-orcales/shared/jlog/default"
import "local/james-orcales/shared/math/bits"
import "local/james-orcales/shared/sim/nbio"
import system_io "local/james-orcales/shared/sim/nbio/default"
import "local/james-orcales/shared/sim/time"
import system_time "local/james-orcales/shared/sim/time/default"

// Test_Default_Floor_Is_Info covers New_Default_Logger building with an Info floor, so trace and
// debug noise stays out of the stderr stream unless a caller raises verbosity explicitly.
func Test_Default_Floor_Is_Info(t *testing.T) {
  writes := &storage_writes{}
  logger := jlog.New_Default_Logger(new_state(), recording_storage(writes), TEST_STDERR)
  if logger.Floor != jlog.LEVEL_INFO {
    t.Fatalf("New_Default_Logger Floor = %v, want LEVEL_INFO", logger.Floor)
  }
}

// Test_Default_Writes_Stderr_Through_Storage covers the Default logger submitting each line to
// the named descriptor at its current position through the injected Storage, and the emit
// returning before that write retires.
func Test_Default_Writes_Stderr_Through_Storage(t *testing.T) {
  writes := &storage_writes{}
  saved := jlog.Default
  t.Cleanup(func() { store_result(&jlog.Default, saved) })
  jlog.Default = jlog.New_Default_Logger(new_state(), recording_storage(writes), TEST_STDERR)
  jlog.Default.Auto_Timestamp = false
  jlog.Info("hello")
  if len(writes.Queue) != 1 {
    t.Fatalf("submitted %d writes, want 1", len(writes.Queue))
  }
  submission := writes.Queue[0]
  if submission.File != TEST_STDERR {
    t.Fatalf("write to file %d, want %d", submission.File, TEST_STDERR)
  }
  if submission.Offset != nbio.FILE_POSITION_CURRENT {
    t.Fatalf("write at offset %d, want the current position", submission.Offset)
  }
  if !jlog.Default.Line.Borrowed {
    t.Fatal("line must stay borrowed until the Storage retires the write")
  }
  storage_retire(writes)
  if jlog.Default.Line.Borrowed {
    t.Fatal("retirement must free the line")
  }
  if !contains(submission.Data, []byte("\"message\":\"hello\"")) {
    t.Fatalf("stderr got %q", submission.Data)
  }
}

// Test_Default_Writes_To_A_Pipe covers the Default logger delivering a line when its descriptor
// is a pipe, the shape stderr takes under `2>&1 | ...`, through the real operating-system IO.
func Test_Default_Writes_To_A_Pipe(t *testing.T) {
  var state system_io.Operating_System
  var memory Memory
  loop, driver, loop_err := system_io.New_Operating_System_IO(
    &state,
    memory_view(&memory),
    system_time.New_Operating_System_Clock(),
    SCHEDULER_ENTRY_COUNT,
    0,
  )
  if loop_err != nil {
    t.Fatalf("operating system io: %v", loop_err)
  }
  defer nbio.Driver_Deinit(driver)
  defer nbio.IO_Deinit(loop)
  read_end, write_end, pipe_err := nbio.IO_Pipe(loop)
  if pipe_err != nil {
    t.Fatalf("pipe: %v", pipe_err)
  }
  defer close_file(t, loop, driver, read_end)
  defer close_file(t, loop, driver, write_end)

  saved := jlog.Default
  t.Cleanup(func() { store_result(&jlog.Default, saved) })
  jlog.Default = jlog.New_Default_Logger(new_state(), loop.Storage, write_end)
  jlog.Default.Auto_Timestamp = false
  jlog.Info("through a pipe")

  for pass_index := 0; pass_index < PUMP_PASSES_MAX; pass_index++ {
    if !jlog.Default.Line.Borrowed {
      break
    }
    if drive_err := nbio.Driver_Run(driver); drive_err != nil {
      t.Fatalf("drive: %v", drive_err)
    }
  }
  if jlog.Default.Line.Borrowed {
    t.Fatal("the pipe write never retired")
  }
  if jlog.Default.Line.Completion.Error != nil {
    t.Fatalf("pipe write retired with %v", jlog.Default.Line.Completion.Error)
  }

  want := "{\"level\":\"info\",\"message\":\"through a pipe\"}\n"
  got := make([]byte, len(want))
  var read nbio.Completion
  nbio.Storage_Read(
    loop.Storage,
    &read,
    read_end,
    got,
    nbio.FILE_POSITION_CURRENT,
    READ_TIMEOUT,
    retired,
  )
  pump(t, driver, &read)
  if read.Error != nil {
    t.Fatalf("read: %v", read.Error)
  }
  if string(got[:read.Data]) != want {
    t.Fatalf("pipe got %q, want %q", got[:read.Data], want)
  }
}

// TEST_STDERR stands in for the descriptor a root would name; the test only checks it is
// passed through untouched.
const TEST_STDERR nbio.File = 2

// Fresh default-logger state with its own line storage, as a root would keep.
func new_state() (state *jlog.Default_Logger) {
  return &jlog.Default_Logger{Line: make([]byte, jlog.LINE_BYTES)}
}

// One submitted Storage write, held until the test retires it.
type storage_write struct {
  Completion *nbio.Completion
  File       nbio.File
  Offset     int64
  Data       []byte
  Callback   nbio.Callback
}

type storage_writes struct {
  Queue []storage_write
}

// A Storage whose Write records the submission and retires when the test says, standing in for
// the root's loop.
func recording_storage(writes *storage_writes) (storage nbio.Storage) {
  return nbio.Storage{State: writes, Write_Procedure: recording_storage_write}
}

func recording_storage_write(
  state nbio.State,
  completion *nbio.Completion,
  file nbio.File,
  buffer []byte,
  offset int64,
  _ time.Duration,
  callback nbio.Callback,
) {
  writes := state.(*storage_writes)
  writes.Queue = append(
    writes.Queue,
    storage_write{
      Completion: completion,
      File:       file,
      Offset:     offset,
      Data:       append([]byte(nil), buffer...),
      Callback:   callback,
    },
  )
}

func storage_retire(writes *storage_writes) {
  submission := writes.Queue[0]
  writes.Queue = writes.Queue[1:]
  submission.Completion.Data = len(submission.Data)
  submission.Completion.Error = nil
  submission.Callback(submission.Completion)
}

func contains(source []byte, sought []byte) (present bool) {
  for start := 0; !(start+len(sought) > len(source)); start++ {
    if string(source[start:start+len(sought)]) == string(sought) {
      return true
    }
  }
  return false
}

type recording_buffer []byte

func buffer_text(buffer *recording_buffer) (text string) { return string(*buffer) }

// Builds a trace-floor logger writing to buffer with a zero clock.
func buffer_logger(buffer *recording_buffer) (logger library.Logger) {
  configuration := &library.Logger_Configuration{
    Stream: recording_stream(buffer), Clock: frozen(),
  }
  return jlog.New(configuration, make([]byte, jlog.LINE_BYTES), jlog.LEVEL_TRACE)
}

// Redirects Default at buffer for one test.
func redirect(t *testing.T, buffer *recording_buffer) {
  t.Helper()
  saved := jlog.Default
  t.Cleanup(func() { store_result(&jlog.Default, saved) })
  jlog.Default = buffer_logger(buffer)
}

// Test_Default_Global_Info covers the package-level convenience API writing an info-level line
// through Default with a re-exported field constructor.
func Test_Default_Global_Info(t *testing.T) {
  buffer := &recording_buffer{}
  redirect(t, buffer)
  jlog.Info("hello", jlog.String("user", "bob"))
  got := buffer_text(buffer)
  want := "{\"level\":\"info\",\"user\":\"bob\",\"message\":\"hello\"}\n"
  if got != want {
    t.Fatalf("got  %q\nwant %q", got, want)
  }
}

// A clock whose realtime reading is always zero.
func frozen() (clock time.Clock) {
  return time.Clock{Now_Monotonic: zero_monotonic, Now_Realtime: zero_realtime}
}

func zero_monotonic(_ time.State) (moment time.Monotonic_Moment) { return 0 }

func zero_realtime(_ time.State) (moment time.Moment) { return 0 }

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

// Test_Default_Caller_Uses_Runtime exercises the OS-backed caller lookup wired into Default and
// checks it names this file.
func Test_Default_Caller_Uses_Runtime(t *testing.T) {
  writes := &storage_writes{}
  saved := jlog.Default
  t.Cleanup(func() { store_result(&jlog.Default, saved) })
  jlog.Default = jlog.New_Default_Logger(new_state(), recording_storage(writes), TEST_STDERR)
  jlog.Info("runtime caller coverage", jlog.Caller())
  if len(writes.Queue) != 1 {
    t.Fatalf("submitted %d writes, want 1", len(writes.Queue))
  }
  if !contains(writes.Queue[0].Data, []byte("requirements_test.go:")) {
    t.Fatalf("caller must name this file: %q", writes.Queue[0].Data)
  }
}

// Test_Default_Cover_API calls every re-exported wrapper and convenience function so the
// composition tier is fully exercised.
func Test_Default_Cover_API(t *testing.T) {
  buffer := &recording_buffer{}
  redirect(t, buffer)

  jlog.Trace("a")
  jlog.Debug("b")
  jlog.Info("c")
  jlog.Warn("d")
  jlog.Error("e")
  jlog.Log("f")

  configuration := &library.Logger_Configuration{
    Stream:          recording_stream(buffer),
    Clock:           frozen(),
    Stack_Marshaler: short_stack,
  }
  logger := jlog.New(configuration, make([]byte, jlog.LINE_BYTES), jlog.LEVEL_TRACE)
  jlog.Logger_Trace(&logger, "")
  jlog.Logger_Debug(&logger, "")
  jlog.Logger_Warn(&logger, "")
  jlog.Logger_Error(&logger, "")
  jlog.Logger_Log(&logger, "")
  jlog.Logger_At_Level(&logger, jlog.LEVEL_WARN, "")
  child := jlog.Logger_With(
    &logger,
    make([]byte, jlog.LINE_BYTES),
    library.Buffer{Bytes: make([]byte, 64)},
    jlog.String("x", "y"),
  )
  jlog.Logger_Info(&child, "with")
  jlog.Logger_Info(
    &logger,
    "fields",
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
    jlog.Strings("q", []string{"a"}),
    jlog.Integers("r", []int{1}),
    jlog.Floats64("s", []float64{1}),
    jlog.Booleans("u", []bool{true}),
    jlog.Durations("w", []time.Duration{time.SECOND}),
    jlog.Err(errors.New("boom")),
    jlog.Timestamp(),
    jlog.Caller(),
  )

  if len(*buffer) == 0 {
    t.Fatal("expected output from the default wrappers")
  }
}

// OPERATION_CAPACITY bounds the backend queues; one line is in flight at a time.
const OPERATION_CAPACITY = 32

// SCHEDULER_ENTRY_COUNT sizes the kernel ring.
const SCHEDULER_ENTRY_COUNT uint16 = 32

// PUMP_PASSES_MAX bounds the root loop; one line needs one submission and one retirement.
const PUMP_PASSES_MAX = 8

// READ_TIMEOUT bounds the read of the one line the test expects.
const READ_TIMEOUT = 1 * time.SECOND

// The retirement the test's own submissions take; the test reads the completion after.
func retired(completion nbio.Completion_Handle) {
  completion.Armed = false
}

// Runs the loop until completion retires, bounded by PUMP_PASSES_MAX.
func pump(t *testing.T, driver nbio.Driver, completion *nbio.Completion) {
  t.Helper()
  for pass_index := 0; pass_index < PUMP_PASSES_MAX; pass_index++ {
    if !completion.Armed {
      return
    }
    if drive_err := nbio.Driver_Run(driver); drive_err != nil {
      t.Fatalf("drive: %v", drive_err)
    }
  }
  t.Fatal("the operation never retired")
}

// Closes one pipe end through the loop, so Deinit's leak census finds nothing open.
func close_file(t *testing.T, loop nbio.IO, driver nbio.Driver, file nbio.File) {
  t.Helper()
  var completion nbio.Completion
  nbio.IO_Close(loop, &completion, file, retired)
  pump(t, driver, &completion)
}

// Memory is the backend's caller-owned queues.
type Memory struct {
  Timeouts            []*nbio.Completion
  Completed           []*nbio.Completion
  Operations          []system_io.Operating_System_Operation
  Operation_Registry  []*system_io.Operating_System_Operation
  Operation_Paths     []byte
  Operation_Addresses []byte
  Descriptors         []system_io.Operating_System_Descriptor
  Signal_Waiters      []system_io.Signal_Waiter
  Spawns              []system_io.Spawn
  Spawn_Registry      []*system_io.Spawn
  Spawn_Pipe_Buffers  []byte
  Spawn_Paths         []byte
  Spawn_Arguments     []string
  Spawn_Child_Files   []uintptr
  Platform_Operations []*system_io.Operating_System_Operation
  Platform_Changes    []system_io.Kernel_Event
  Platform_Events     []system_io.Kernel_Event
}

func memory_view(memory *Memory) (view system_io.Operating_System_Memory) {
  memory.Timeouts = make([]*nbio.Completion, OPERATION_CAPACITY)
  memory.Completed = make([]*nbio.Completion, OPERATION_CAPACITY)
  memory.Operations = make([]system_io.Operating_System_Operation, OPERATION_CAPACITY)
  memory.Operation_Registry = make(
    []*system_io.Operating_System_Operation, OPERATION_CAPACITY,
  )
  memory.Operation_Paths = make(
    []byte,
    OPERATION_CAPACITY*system_io.OPERATING_SYSTEM_PATH_BYTES_MAXIMUM,
  )
  memory.Operation_Addresses = make(
    []byte, OPERATION_CAPACITY*system_io.SOCKET_ADDRESS_BYTES,
  )
  memory.Descriptors = make([]system_io.Operating_System_Descriptor, bits.WORD_SIZE)
  memory.Signal_Waiters = make([]system_io.Signal_Waiter, 1)
  memory.Spawns = make([]system_io.Spawn, 1)
  memory.Spawn_Registry = make([]*system_io.Spawn, 1)
  memory.Spawn_Pipe_Buffers = make(
    []byte,
    system_io.PROCESS_PIPE_BUFFER_COUNT*system_io.PROCESS_PIPE_BYTES,
  )
  memory.Spawn_Paths = make([]byte, system_io.PROCESS_PATH_BYTES)
  memory.Spawn_Arguments = make([]string, system_io.PROCESS_ARGUMENT_COUNT_MAXIMUM)
  memory.Spawn_Child_Files = make([]uintptr, system_io.PROCESS_CHILD_DESCRIPTOR_COUNT)
  memory.Platform_Operations = make(
    []*system_io.Operating_System_Operation, OPERATION_CAPACITY,
  )
  memory.Platform_Changes = make([]system_io.Kernel_Event, system_io.POLL_EVENTS_MAX)
  memory.Platform_Events = make([]system_io.Kernel_Event, system_io.POLL_EVENTS_MAX)
  return system_io.Operating_System_Memory{
    Timeouts:                   memory.Timeouts,
    Completed:                  memory.Completed,
    Operations:                 memory.Operations,
    Operation_Registry:         memory.Operation_Registry,
    Operation_File_Paths:       memory.Operation_Paths,
    Operation_Socket_Addresses: memory.Operation_Addresses,
    Descriptors:                memory.Descriptors,
    Signal_Waiters:             memory.Signal_Waiters,
    Spawns:                     memory.Spawns,
    Spawn_Registry:             memory.Spawn_Registry,
    Spawn_Pipe_Buffers:         memory.Spawn_Pipe_Buffers,
    Spawn_Paths:                memory.Spawn_Paths,
    Spawn_Arguments:            memory.Spawn_Arguments,
    Spawn_Child_Files:          memory.Spawn_Child_Files,
    Platform_Operations:        memory.Platform_Operations,
    Platform_Changes:           memory.Platform_Changes,
    Platform_Events:            memory.Platform_Events,
  }
}

func store_result[T any](destination *T, value T) {
  *destination = value
}

func short_stack(_ error) (stack string) {
  return "S"
}
