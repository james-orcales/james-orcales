// Package nbio is operating-system backend of shared/sim/nbio.
//
// Darwin ride kqueue, Linux ride io_uring. No generic Cancel. Descriptor owner do Shutdown, join
// submitted operations, then asynchronous Close.
package nbio

import "errors"
import "runtime"
import "syscall"
import "unsafe"

import "local/james-orcales/shared/filepath"
import "local/james-orcales/shared/math/bits"
import "local/james-orcales/shared/sim/aver/default"
import "local/james-orcales/shared/sim/nbio"
import "local/james-orcales/shared/sim/time"

// Poll_forever, passed as idle gap, block wait until event arrive, not for fixed span — the
// unbounded wait a run with no deadline need, thus loop sleep exactly until there is work,
// instead of wake on interval.
const POLL_FOREVER time.Monotonic_Moment = -1

// Buffer few pending signals, thus burst is not lost between drains.
const SIGNAL_QUEUE_DEPTH = 8

// Cap idle gap while signal watcher exist, since signal does not wake poll. Loop re-check
// signal channel at least this often.
const SIGNAL_POLL_INTERVAL = 10 * time.MILLISECOND

// Hold host backend state: completed queue, Darwin timeout queue, platform scheduler, and
// repository-extension effects, which all retire through completed.
type Operating_System struct {
  // Host is real clock. Deadline and wait are measured against it.
  Host time.Clock
  // Timeouts borrow caller capacity for pending timers ordered by Ready_At.
  Timeouts []*nbio.Completion
  // Completed borrow caller capacity for callbacks ready on next drain.
  Completed []*nbio.Completion
  // Platform is kqueue on Darwin, or io_uring on Linux, made eagerly by constructor.
  Platform Platform_Scheduler
  // Operations borrow caller registry capacity for active kernel work.
  Operations []*Operating_System_Operation
  // Operation_Memory owns stable caller slots retained while kernel holds their addresses.
  Operation_Memory []Operating_System_Operation
  // Next_Identifier is last non-zero kernel correlation identifier issued.
  Next_Identifier uint64
  // Signals receive OS signals from os/signal notifier. Nil until first watch.
  Signals Operating_System_Signal_Channel
  // Forwarded receive the signals a live child is owed. A channel apart from Signals, thus
  // stopping it hands those signals back to their default action without touching a watch.
  Forwarded Operating_System_Signal_Channel
  // Forward_Notified report Forwarded is registered with the notifier. Registration outlive
  // each child: a register and stop per spawn allocate every time.
  Forward_Notified bool
  // Signal_Waiters borrow caller capacity for one-shot signal watches.
  Signal_Waiters []Signal_Waiter
  // Spawns borrow caller registry capacity for children retained through reap.
  Spawns []*Spawn
  // Spawn_Memory owns stable caller slots for retained child state.
  Spawn_Memory []Spawn
  // Extension_Submitted count repository-extension completions not yet delivered to caller.
  Extension_Submitted int
  // Drive_Active is set while Run* drive loop, thus Run* called from inside completion
  // callback — which would re-enter driver mid-drain — panic loud.
  Drive_Active bool
  // Raw_Open are caller-owned descriptor census slots.
  Raw_Open []Operating_System_Descriptor
}

// Operating_System_Descriptor is one caller-owned raw-descriptor census slot.
type Operating_System_Descriptor struct {
  // Used separates live descriptors from available capacity.
  Used bool
  // Descriptor is the kernel descriptor whose ownership the backend tracks.
  Descriptor int
  // Pipe routes Storage transfers to pipe operations, because pread and an io_uring file
  // offset reject a pipe with ESPIPE.
  Pipe bool
}

// Operating_System_Memory supplies every bounded queue and registry the backend retains.
type Operating_System_Memory struct {
  // Timeouts bound simultaneous userspace deadlines.
  Timeouts []*nbio.Completion
  // Completed bound callbacks ready before one drain pass.
  Completed []*nbio.Completion
  // Operations bound simultaneous kernel work, including internal linked deadlines.
  Operations []Operating_System_Operation
  // Operation_File_Paths supply one stable path region per operation.
  Operation_File_Paths []byte
  // Operation_Socket_Addresses supply one stable sockaddr region per operation.
  Operation_Socket_Addresses []byte
  // Operation_Registry bounds active operation lookup by kernel identifier.
  Operation_Registry []*Operating_System_Operation
  // Descriptors bound simultaneous caller-owned descriptors.
  Descriptors []Operating_System_Descriptor
  // Signal_Waiters bound simultaneous signal watches.
  Signal_Waiters []Signal_Waiter
  // Spawns bound simultaneous child processes retained through reap.
  Spawns []Spawn
  // Spawn_Registry bounds lookup of active child slots.
  Spawn_Registry []*Spawn
  // Spawn_Pipe_Buffers supply standard-output and standard-error pass storage for each spawn.
  Spawn_Pipe_Buffers []byte
  // Spawn_Paths supply one resolved executable path region for each spawn.
  Spawn_Paths []byte
  // Spawn_Arguments supply one argv region for each spawn.
  Spawn_Arguments []string
  // Spawn_Child_Files supply the child's three standard descriptors for each spawn.
  Spawn_Child_Files []uintptr
  // Platform_Operations bound Darwin readiness backlog or Linux retry backlog.
  Platform_Operations []*Operating_System_Operation
  // Platform_Changes bound one Darwin kevent changelist; Linux leaves it unused.
  Platform_Changes []Kernel_Event
  // Platform_Events bound one Darwin kevent result batch; Linux leaves it unused.
  Platform_Events []Kernel_Event
}

// Kernel_Event is the Darwin kevent layout. Linux retains only its caller-memory type so one
// composition root can provide platform memory without build-specific API shape.
type Kernel_Event struct {
  // Ident preserves the first UAPI word across the raw syscall.
  Ident uint64
  // Filter preserves the signed UAPI selector across the raw syscall.
  Filter int16
  // Flags preserves the UAPI control bits across the raw syscall.
  Flags uint16
  // Filter_Flags preserves filter-specific UAPI bits across the raw syscall.
  Filter_Flags uint32
  // Data preserves the signed UAPI result word across the raw syscall.
  Data int64
  // User_Data keeps correlation as integer, never a Go pointer retained by kernel.
  User_Data uint64
}

// Static identity keeps validation failure outside caller allocation budget.
var scheduler_entries_outside_range = errors.New("io: scheduler entries must be in [1, 4095]")

// One registered signal watcher: OS signal it await, its backend-independent kind, and
// completion and callback to fire once on delivery.
type Signal_Waiter struct {
  // System is OS signal this watcher await.
  System syscall.Signal
  // Kind is backend-independent signal reported to callback.
  Kind nbio.Signal
  // Completion is caller-owned completion fired on delivery.
  Completion *nbio.Completion
  // Callback is typed callback run with delivered signal.
  Callback nbio.Signal_Callback
  // Deadline is finite moment this repository-extension operation retire at.
  Deadline time.Monotonic_Moment
}

// New_Operating_System_IO eagerly make platform scheduler. Entries is io_uring queue size on
// Linux, and kqueue on Darwin accept but ignore it. It must fit in twelve bits. Flags
// pass direct to io_uring setup, and Darwin ignore them. Ambient process values host read back
// come from os/default, which needs nothing from here.
func New_Operating_System_IO(
  state *Operating_System,
  memory Operating_System_Memory,
  host time.Clock,
  entries uint16,
  flags uint32,
) (loop nbio.IO, driver nbio.Driver, err error) {
  aver.Always(state != nil, "An operating-system IO backend has caller-owned state.")
  aver.Always(len(memory.Timeouts) > 0,
    "An operating-system IO backend has timeout capacity.")
  aver.Always(len(memory.Completed) > 0,
    "An operating-system IO backend has completion capacity.")
  aver.Always(len(memory.Operations) > 0,
    "An operating-system IO backend has operation capacity.")
  aver.Always(
    len(memory.Descriptors) > 0,
    "An operating-system IO backend has descriptor capacity.",
  )
  aver.Always(
    len(memory.Signal_Waiters) > 0,
    "An operating-system IO backend has signal-waiter capacity.",
  )
  aver.Always(len(memory.Spawns) > 0,
    "An operating-system IO backend has spawn capacity.")
  aver.Always(
    len(memory.Platform_Operations) > 0,
    "An operating-system IO backend has platform-operation capacity.",
  )
  if entries == 0 {
    return nbio.IO{}, nbio.Driver{}, scheduler_entries_outside_range
  }
  if entries > 4095 {
    return nbio.IO{}, nbio.Driver{}, scheduler_entries_outside_range
  }
  platform, initialize_err := platform_initialize(entries, flags)
  if initialize_err != nil {
    return nbio.IO{}, nbio.Driver{}, initialize_err
  }
  operating_system_operation_memory_initialize(memory)
  for index := range memory.Descriptors {
    memory.Descriptors[index] = Operating_System_Descriptor{}
  }
  operating_system_spawn_memory_initialize(memory)
  *state = Operating_System{
    Host:             host,
    Platform:         platform,
    Timeouts:         memory.Timeouts[:0],
    Completed:        memory.Completed[:0],
    Operations:       memory.Operation_Registry[:0],
    Operation_Memory: memory.Operations,
    Raw_Open:         memory.Descriptors,
    Signal_Waiters:   memory.Signal_Waiters[:0],
    Spawns:           memory.Spawn_Registry[:0],
    Spawn_Memory:     memory.Spawns,
  }
  platform_memory_set(&state.Platform, memory)
  operating_system_wire_file(state, &loop.Storage)
  operating_system_wire_timer(state, &loop.Timeline)
  operating_system_wire_socket(state, &loop.Network)
  operating_system_wire_close(&loop)
  operating_system_wire_effects(state, &loop)
  operating_system_wire_platform(state, &loop)
  nbio.IO_Invariants(loop, "new_operating_system_io.loop")
  nbio.Timeline_Invariants(loop.Timeline, "new_operating_system_io.timeline")
  return loop, operating_system_to_driver(state), nil
}

// Each kernel-retained pointer gets one disjoint caller region so slot reuse cannot alias work
// still owned by the kernel.
func operating_system_operation_memory_initialize(memory Operating_System_Memory) {
  operation_count := len(memory.Operations)
  aver.Always(
    len(memory.Operation_File_Paths) ==
      operation_count*OPERATING_SYSTEM_PATH_BYTES_MAXIMUM,
    "Operating-system operations have exact path storage.",
  )
  aver.Always(
    len(memory.Operation_Socket_Addresses) == operation_count*SOCKET_ADDRESS_BYTES,
    "Operating-system operations have exact socket-address storage.",
  )
  for index := range memory.Operations {
    path_start := index * OPERATING_SYSTEM_PATH_BYTES_MAXIMUM
    path_end := path_start + OPERATING_SYSTEM_PATH_BYTES_MAXIMUM
    socket_start := index * SOCKET_ADDRESS_BYTES
    socket_end := socket_start + SOCKET_ADDRESS_BYTES
    file_path := memory.Operation_File_Paths[path_start:path_end:path_end]
    socket_address := memory.Operation_Socket_Addresses[socket_start:socket_end]
    socket_address = socket_address[:len(socket_address):len(socket_address)]
    memory.Operations[index] = Operating_System_Operation{
      File_Path:      file_path,
      Socket_Address: socket_address,
    }
  }
}

// Each spawn slot gets disjoint windows of caller memory, so a slot reused after delivery never
// aliases bytes an earlier child still borrows, and a spawn needs no storage of its own.
func operating_system_spawn_memory_initialize(memory Operating_System_Memory) {
  spawn_count := len(memory.Spawns)
  aver.Always(
    len(memory.Spawn_Pipe_Buffers) ==
      spawn_count*PROCESS_PIPE_BUFFER_COUNT*PROCESS_PIPE_BYTES,
    "Spawns have exact pipe buffer storage.",
  )
  aver.Always(
    len(memory.Spawn_Paths) == spawn_count*filepath.PATH_SIZE_MAXIMUM,
    "Spawns have exact path storage.",
  )
  aver.Always(
    len(memory.Spawn_Arguments) == spawn_count*PROCESS_ARGUMENT_COUNT_MAXIMUM,
    "Spawns have exact argument storage.",
  )
  aver.Always(
    len(memory.Spawn_Child_Files) == spawn_count*PROCESS_CHILD_DESCRIPTOR_COUNT,
    "Spawns have exact child descriptor storage.",
  )
  for index := range memory.Spawns {
    pipe_start := index * PROCESS_PIPE_BUFFER_COUNT * PROCESS_PIPE_BYTES
    pipe_middle := pipe_start + PROCESS_PIPE_BYTES
    pipe_end := pipe_middle + PROCESS_PIPE_BYTES
    path_start := index * filepath.PATH_SIZE_MAXIMUM
    path_end := path_start + filepath.PATH_SIZE_MAXIMUM
    argument_start := index * PROCESS_ARGUMENT_COUNT_MAXIMUM
    argument_end := argument_start + PROCESS_ARGUMENT_COUNT_MAXIMUM
    file_start := index * PROCESS_CHILD_DESCRIPTOR_COUNT
    file_end := file_start + PROCESS_CHILD_DESCRIPTOR_COUNT
    pipes, arguments := memory.Spawn_Pipe_Buffers, memory.Spawn_Arguments
    memory.Spawns[index] = Spawn{
      Output_Buffer:   pipes[pipe_start:pipe_middle:pipe_middle],
      Error_Buffer:    pipes[pipe_middle:pipe_end:pipe_end],
      Path_Storage:    memory.Spawn_Paths[path_start:path_end:path_end],
      Argument_Vector: arguments[argument_start:argument_end:argument_end],
      Child_Files:     memory.Spawn_Child_Files[file_start:file_end:file_end],
    }
  }
}

// Zero a slot's state while keeping its caller memory windows, which stay fixed for the life of
// the backend.
func spawn_reset(slot *Spawn) {
  output_buffer, error_buffer := slot.Output_Buffer, slot.Error_Buffer
  path, arguments, files := slot.Path_Storage, slot.Argument_Vector, slot.Child_Files
  *slot = Spawn{}
  slot.Output_Buffer, slot.Error_Buffer = output_buffer, error_buffer
  slot.Path_Storage, slot.Argument_Vector, slot.Child_Files = path, arguments, files
}

// Arm completion through idle-to-armed lifecycle edge, same edge virtual timeline stamp
// (shared/sim/time/time.go:535-540), thus both backend read alike.
func operating_system_submit(completion *nbio.Completion) {
  original := completion.Self == nil
  if completion.Self == completion {
    original = true
  }
  aver.Always(original,
    "A submitted completion is its own original, never a by-value copy.")
  completion.Self = completion
  aver.Always(!completion.Armed, "An armed completion is never armed a second clock.")
  completion.Data = 0
  completion.Error = nil
  completion.Armed = true
}

// Wire signal watch and spawn onto IO surface that own them. Both retire on same completed queue
// every IO operation use, thus one timeline hold whole run.
func operating_system_wire_effects(state *Operating_System, loop *nbio.IO) {
  loop.Watch_Signal = operating_system_watch_signal_procedure
  loop.Spawn = operating_system_spawn_procedure
}

func operating_system_watch_signal_procedure(
  state_pointer nbio.State,
  completion *nbio.Completion,
  signal nbio.Signal,
  deadline time.Duration,
  callback nbio.Signal_Callback,
) {
  state := state_pointer.(*Operating_System)
  aver.Always(deadline > 0, "A signal-watch deadline is positive and finite.")
  state.Extension_Submitted++
  operating_system_submit(completion)
  operating_system_watch_signal(state, completion, signal, deadline, callback)
}

func operating_system_spawn_procedure(
  state_pointer nbio.State,
  completion *nbio.Completion,
  request nbio.Process_Request,
  deadline time.Duration,
  callback nbio.Process_Callback,
) {
  state := state_pointer.(*Operating_System)
  aver.Always(deadline > 0, "A spawn deadline is positive and finite.")
  state.Extension_Submitted++
  operating_system_submit(completion)
  operating_system_spawn(state, completion, request, deadline, callback)
}

// Bound one pipe read, thus chatty child is drained in repeated passes, not into one unbounded
// allocation.
const PROCESS_PIPE_BYTES = 8 * bits.KIBIBYTE_BYTES

// PROCESS_ARGUMENT_COUNT_MAXIMUM bounds argv held in a spawn slot, program name included. A
// command past it fails with E2BIG, the verdict the kernel gives an oversized argv.
const PROCESS_ARGUMENT_COUNT_MAXIMUM = 256

// PROCESS_CHILD_DESCRIPTOR_COUNT is the child's standard input, output, and error.
const PROCESS_CHILD_DESCRIPTOR_COUNT = 3

// PROCESS_PIPE_BUFFER_COUNT is one pass buffer for standard output and one for standard error.
const PROCESS_PIPE_BUFFER_COUNT = 2

// PROCESS_PATH_BYTES is the resolved executable path region of one spawn slot.
const PROCESS_PATH_BYTES = filepath.PATH_SIZE_MAXIMUM

// Cap wait for pipes of killed child to close. Grandchild that inherited write end keep pipe
// open after its parent die, thus drain need its own bound.
const PROCESS_CLEANUP_DURATION = 1 * time.SECOND

// One spawned child loop track. It outlive caller completion: deadline retire that completion
// early, and exit event still arrive after, and still has to reap. Entry leave state.Spawns only
// when its reap finish.
type Spawn struct {
  // Used separates retained child state from caller-owned capacity.
  Used bool
  // Identifier is child process id, and its process group id, because Setpgid is set.
  Identifier int
  // Completion is caller-owned completion result is delivered on.
  Completion *nbio.Completion
  // Callback is typed callback run once result is whole.
  Callback nbio.Process_Callback
  // Started is moment child was forked, for wall-time measurement.
  Started time.Monotonic_Moment
  // Exit_Descriptor is Linux pidfd polled for exit, or -1 on Darwin.
  Exit_Descriptor int
  // Cycle_Descriptor is the Linux perf event counting cycles, leader of the child's counter
  // group. It and the four below are -1 when never opened or already closed; Darwin opens
  // none.
  Cycle_Descriptor int
  // Instruction_Descriptor is the Linux perf event counting instructions.
  Instruction_Descriptor int
  // Cache_Reference_Descriptor is the Linux perf event counting cache references.
  Cache_Reference_Descriptor int
  // Cache_Miss_Descriptor is the Linux perf event counting cache misses.
  Cache_Miss_Descriptor int
  // Branch_Miss_Descriptor is the Linux perf event counting branch misses.
  Branch_Miss_Descriptor int
  // Input_Descriptor is write end of child standard input. -1 once closed.
  Input_Descriptor int
  // Output_Descriptor is read end of child standard output. -1 once closed.
  Output_Descriptor int
  // Error_Descriptor is read end of child standard error. -1 once closed.
  Error_Descriptor int
  // Operating_System is the backend owning this slot, so a static callback reaches it from
  // the slot alone.
  Operating_System *Operating_System
  // Input is remaining bytes to write to child standard input.
  Input []byte
  // Input_Inherited report child read this process's descriptor 0, which the slot never owns.
  Input_Inherited bool
  // Signals_Forwarded report signals this process receive pass on to child group.
  Signals_Forwarded bool
  // Output_Buffer receive one standard-output pass. It and the four fields below are windows
  // of caller spawn memory, fixed for the life of the backend.
  Output_Buffer []byte
  // Error_Buffer receive one standard-error pass.
  Error_Buffer []byte
  // Path_Storage holds the resolved executable path; argv[0] and StartProcess view it.
  Path_Storage []byte
  // Argument_Vector holds argv the child receives: program name, then caller arguments.
  Argument_Vector []string
  // Child_Files hold the three pipe ends the child receives as its standard descriptors.
  Child_Files []uintptr
  // Attributes is the fork request StartProcess reads.
  Attributes syscall.ProcAttr
  // System_Attributes is the platform part of Attributes.
  System_Attributes syscall.SysProcAttr
  // Final_Error is the error delivered beside Result.
  Final_Error error
  // Caller_Backend keeps the caller's Completion.Backend while delivery borrows it.
  Caller_Backend nbio.State
  // Result accumulate captured output, exit code, and usage.
  Result nbio.Process_Result
  // Request hold caller live output sinks.
  Request nbio.Process_Request
  // Exit_Completion wait for child to exit.
  Exit_Completion nbio.Completion
  // Output_Completion read one standard-output pass.
  Output_Completion nbio.Completion
  // Error_Completion read one standard-error pass.
  Error_Completion nbio.Completion
  // Output_Stream_Completion keeps the pipe buffer borrowed until its sink retires.
  Output_Stream_Completion nbio.Completion
  // Error_Stream_Completion keeps the pipe buffer borrowed until its sink retires.
  Error_Stream_Completion nbio.Completion
  // Input_Completion write one standard-input pass.
  Input_Completion nbio.Completion
  // Deadline_Completion stays backend-owned so an earlier child exit can withdraw its bound.
  Deadline_Completion nbio.Completion
  // Cleanup_Completion bound pipe drain after deadline kill.
  Cleanup_Completion nbio.Completion
  // Exited report reap finished, and Result carry exit code and usage.
  Exited bool
  // Output_Drained report standard-output pipe reached its end.
  Output_Drained bool
  // Error_Drained report standard-error pipe reached its end.
  Error_Drained bool
  // Expired report deadline won, thus result carry Deadline_Exceeded.
  Expired bool
  // Cleanup_Armed report bounded pipe drain already run, thus neither child exit nor
  // deadline arm second one.
  Cleanup_Armed bool
  // Stream_Error preserves the first sink failure while the loop continues to drain pipes.
  Stream_Error error
  // Delivered report caller completion already retired.
  Delivered bool
}

// Start command of request and arm every operation that finish it: one read for each output
// pipe, one write for input, exit watch, and deadline. Nothing run off loop thread.
func operating_system_spawn(
  state *Operating_System,
  completion *nbio.Completion,
  request nbio.Process_Request,
  deadline time.Duration,
  callback nbio.Process_Callback,
) {
  spawn := operating_system_spawn_acquire(state)
  spawn.Completion = completion
  spawn.Callback = callback
  if start_err := process_start(state, spawn, request); start_err != nil {
    spawn.Final_Error = start_err
    process_publish(state, spawn)
    return
  }
  spawn.Request = request
  spawn.Started = time.Clock_Now_Monotonic(state.Host)
  aver.Always(
    len(state.Spawns) < cap(state.Spawns),
    "The caller-owned spawn registry has capacity before child ownership.",
  )
  state.Spawns = append(state.Spawns, spawn)
  process_forward_start(state, spawn, request)
  process_arm_pipes(state)
  process_watch_exit(state, spawn)
  operating_system_submit(&spawn.Deadline_Completion)
  operating_system_internal_timeout(
    state,
    &spawn.Deadline_Completion,
    deadline,
    process_deadline_fired,
  )
}

// Take one caller-owned spawn slot. The slot holds every byte a spawn needs until its result is
// delivered, so starting a child allocates nothing of the backend's own.
func operating_system_spawn_acquire(state *Operating_System) (spawn *Spawn) {
  for index := range state.Spawn_Memory {
    slot := &state.Spawn_Memory[index]
    if slot.Used {
      continue
    }
    spawn_reset(slot)
    slot.Used = true
    slot.Operating_System = state
    slot.Exit_Descriptor = -1
    slot.Cycle_Descriptor = -1
    slot.Instruction_Descriptor = -1
    slot.Cache_Reference_Descriptor = -1
    slot.Cache_Miss_Descriptor = -1
    slot.Branch_Miss_Descriptor = -1
    slot.Input_Descriptor = -1
    slot.Output_Descriptor = -1
    slot.Error_Descriptor = -1
    return slot
  }
  aver.Always(false, "The caller-owned spawn pool has capacity before child ownership.")
  return nil
}

// Every backend completion of a spawn is a field of its slot, so the field offset leads from the
// completion back to the slot without a captured closure.
func spawn_of(completion *nbio.Completion, offset uintptr) (spawn *Spawn) {
  return (*Spawn)(unsafe.Add(unsafe.Pointer(completion), -int(offset)))
}

func process_deadline_fired(completion nbio.Completion_Handle) {
  spawn := spawn_of(completion, unsafe.Offsetof(Spawn{}.Deadline_Completion))
  process_expire(spawn.Operating_System, spawn)
}

// Fork child with its three pipes into slot. Every descriptor is released on failure part-way
// through, thus failed start leak nothing.
func process_start(
  state *Operating_System,
  spawn *Spawn,
  request nbio.Process_Request,
) (err error) {
  path, path_err := executable_path(
    request.Path,
    operating_system_search_path(request),
    spawn.Path_Storage[:],
  )
  if path_err != nil {
    return path_err
  }
  argv, argv_err := process_argv(spawn, path, request.Arguments)
  if argv_err != nil {
    return argv_err
  }
  spawn.Input_Inherited = request.Input_Inherited
  spawn.Child_Files[0] = 0
  if pipe_err := process_pipes(state, spawn, request); pipe_err != nil {
    return pipe_err
  }
  spawn.Attributes = syscall.ProcAttr{
    Dir:   request.Working_Directory,
    Env:   request.Environment,
    Files: spawn.Child_Files[:],
    Sys:   process_attributes(spawn),
  }
  identifier, start_err := process_fork(spawn, path, argv)
  // Child hold its own copies once StartProcess return, thus parent release three ends it
  // handed over, whether or not fork succeeded.
  process_release_child(state, process_child_owned(spawn))
  if start_err != nil {
    process_counters_close(spawn)
    process_close_pipes(state, spawn)
    return start_err
  }
  spawn.Identifier = identifier
  if ready_err := process_watch_ready(spawn); ready_err != nil {
    syscall.Kill(-identifier, syscall.SIGKILL)
    process_counters_close(spawn)
    process_close_pipes(state, spawn)
    process_reap(identifier)
    return ready_err
  }
  return nil
}

// Make three pipes of child through Pipe, thus every end sit in descriptor census and Deinit
// catch leaked one. Slot keep ends loop hold, and Child_Files the three ends child receive as its
// standard descriptors.
//
// Pipe make every end close-on-exec BEFORE fork. StartProcess dup only Files entries and leave
// every other open descriptor to exec, thus kept end never reach child. Child would otherwise
// hold writer for its own standard input, and close of parent end would never give it
// end-of-file.
func process_pipes(
  state *Operating_System,
  spawn *Spawn,
  request nbio.Process_Request,
) (err error) {
  if request.Input_Inherited {
    aver.Always(
      len(request.Input) == 0,
      "An inherited standard input carries no bytes for the loop to write.",
    )
    return process_output_pipes(state, spawn, request)
  }
  input_read, input_write, input_err := operating_system_pipe_procedure(state)
  if input_err != nil {
    return input_err
  }
  spawn.Input_Descriptor = int(input_write)
  spawn.Input = request.Input
  spawn.Child_Files[0] = uintptr(input_read)
  output_err := process_output_pipes(state, spawn, request)
  if output_err != nil {
    process_release_child(state, []uintptr{uintptr(input_read)})
    process_close_input_or_output(state, &spawn.Input_Descriptor)
  }
  return output_err
}

// Make standard output and error pipes, and give the child its ends their blocking mode. Input is
// already in place, thus the mode pass cover every end the slot own.
func process_output_pipes(
  state *Operating_System,
  spawn *Spawn,
  request nbio.Process_Request,
) (err error) {
  output_read, output_write, output_err := operating_system_pipe_procedure(state)
  if output_err != nil {
    return output_err
  }
  error_read, error_write, error_err := operating_system_pipe_procedure(state)
  if error_err != nil {
    process_release_child(state, []uintptr{uintptr(output_read), uintptr(output_write)})
    return error_err
  }
  spawn.Output_Descriptor = int(output_read)
  spawn.Error_Descriptor = int(error_read)
  spawn.Child_Files[1] = uintptr(output_write)
  spawn.Child_Files[2] = uintptr(error_write)
  if request.Inherited_Nonblock {
    return nil
  }
  for _, descriptor := range process_child_owned(spawn) {
    if inherit_err := pipe_inherit(int(descriptor)); inherit_err != nil {
      process_release_child(state, process_child_owned(spawn))
      process_close_pipes(state, spawn)
      return inherit_err
    }
  }
  return nil
}

// Report child ends the slot own. Inherited descriptor 0 belong to this process, thus release
// and mode change must never reach it.
func process_child_owned(spawn *Spawn) (owned []uintptr) {
  if spawn.Input_Inherited {
    return spawn.Child_Files[1:]
  }
  return spawn.Child_Files[:]
}

// Release pipe ends child hold its own copies of, or ends a failed start never handed over.
func process_release_child(state *Operating_System, child []uintptr) {
  for _, descriptor := range child {
    operating_system_descriptor_release(state, int(descriptor))
  }
}

// Build argv child receive: program name, then caller arguments, which exclude it. exec.Command
// supplied this before, and StartProcess does not. It lives in the slot, bounded like the
// kernel bounds argv.
func process_argv(
  spawn *Spawn, path string, arguments []string,
) (argv []string, err error) {
  if !(len(arguments) < len(spawn.Argument_Vector)) {
    return nil, syscall.E2BIG
  }
  spawn.Argument_Vector[0] = path
  count := 1 + copy(spawn.Argument_Vector[1:], arguments)
  return spawn.Argument_Vector[:count], nil
}

// Report PATH a spawn resolve bare command name against. Request carrying its own environment
// resolve against PATH of that environment, thus parent PATH cannot surprise caller. Request with
// no environment inherit parent PATH.
func operating_system_search_path(request nbio.Process_Request) (search string) {
  if request.Environment == nil {
    search, _ = syscall.Getenv("PATH")
    return search
  }
  for index := len(request.Environment) - 1; !(index < 0); index-- {
    entry := request.Environment[index]
    if !(len(entry) < len("PATH=")) {
      if entry[:len("PATH=")] == "PATH=" {
        return entry[len("PATH="):]
      }
    }
  }
  return ""
}

// Arm read that drain child standard output.
func process_read_output(state *Operating_System, spawn *Spawn) {
  operating_system_submit(&spawn.Output_Completion)
  operating_system_pipe_read(
    state,
    &spawn.Output_Completion,
    spawn.Output_Descriptor,
    spawn.Output_Buffer[:],
    process_output_read,
  )
}

func process_output_read(completion nbio.Completion_Handle) {
  spawn := spawn_of(completion, unsafe.Offsetof(Spawn{}.Output_Completion))
  process_output_pass(spawn.Operating_System, spawn, completion.Data, completion.Error)
}

// Arm read that drain child standard error.
func process_read_error(state *Operating_System, spawn *Spawn) {
  operating_system_submit(&spawn.Error_Completion)
  operating_system_pipe_read(
    state,
    &spawn.Error_Completion,
    spawn.Error_Descriptor,
    spawn.Error_Buffer[:],
    process_error_read,
  )
}

func process_error_read(completion nbio.Completion_Handle) {
  spawn := spawn_of(completion, unsafe.Offsetof(Spawn{}.Error_Completion))
  process_error_pass(spawn.Operating_System, spawn, completion.Data, completion.Error)
}

// Report whether one output pass ended pipe. Zero count is child write end closing, and error
// end drain same way, thus broken pipe cannot leave spawn waiting forever.
func process_pass_ended(count int, read_err error) (ended bool) {
  if read_err != nil {
    return true
  }
  return count == 0
}

// Accept one standard-output pass, then rearm read, or mark pipe drained.
func process_output_pass(
  state *Operating_System, spawn *Spawn, count int, read_err error,
) {
  if process_pass_ended(count, read_err) {
    spawn.Output_Drained = true
    process_close_input_or_output(state, &spawn.Output_Descriptor)
    process_finish(state, spawn)
    return
  }
  pass := spawn.Output_Buffer[:count]
  // Caller-supplied sink stream pass live and leave captured field empty. Without one,
  // bytes accumulate for result. Two stay exclusive, same as when os/exec owned copy. Sink
  // run on loop thread, thus Write that block stall every other operation.
  if spawn.Request.Stdout.Procedure != nil {
    nbio.Write(
      spawn.Request.Stdout, &spawn.Output_Stream_Completion, pass,
      process_output_streamed,
    )
  } else {
    process_capture(spawn, &spawn.Result.Output, spawn.Request.Output_Storage, pass)
  }
}

// Accept one standard-error pass, counterpart of process_output_pass.
func process_error_pass(
  state *Operating_System, spawn *Spawn, count int, read_err error,
) {
  if process_pass_ended(count, read_err) {
    spawn.Error_Drained = true
    process_close_input_or_output(state, &spawn.Error_Descriptor)
    process_finish(state, spawn)
    return
  }
  pass := spawn.Error_Buffer[:count]
  if spawn.Request.Stderr.Procedure != nil {
    nbio.Write(
      spawn.Request.Stderr, &spawn.Error_Stream_Completion, pass,
      process_error_streamed,
    )
  } else {
    process_capture(
      spawn, &spawn.Result.Error_Output, spawn.Request.Error_Output_Storage, pass,
    )
  }
}

// Copy one pass into caller capture storage, never growing it. Output past storage is dropped
// and recorded as the first failure, while the pipe keeps draining so the child cannot block on
// a full pipe.
func process_capture(spawn *Spawn, captured *[]byte, storage []byte, pass []byte) {
  count := len(*captured)
  written := copy(storage[count:], pass)
  *captured = storage[:count+written]
  if written == len(pass) {
    return
  }
  if spawn.Stream_Error == nil {
    spawn.Stream_Error = nbio.Process_Output_Too_Large
  }
}

func process_output_streamed(completion nbio.Completion_Handle) {
  spawn := spawn_of(completion, unsafe.Offsetof(Spawn{}.Output_Stream_Completion))
  process_stream_pass(spawn.Operating_System, spawn, completion)
}

func process_error_streamed(completion nbio.Completion_Handle) {
  spawn := spawn_of(completion, unsafe.Offsetof(Spawn{}.Error_Stream_Completion))
  process_stream_pass(spawn.Operating_System, spawn, completion)
}

// A sink failure cannot stop pipe drain because a child can block on its full pipe. Preserve the
// failure, continue the drain, and publish it only after the child and both pipes finish.
func process_stream_pass(
  state *Operating_System, spawn *Spawn, completion *nbio.Completion,
) {
  if completion.Error != nil {
    if spawn.Stream_Error == nil {
      spawn.Stream_Error = completion.Error
    }
  }
  process_finish(state, spawn)
}

// Accept one standard-input pass. Write failure end feed by close of end, which child observe as
// end-of-file.
func process_input_pass(
  state *Operating_System, spawn *Spawn, count int, write_err error,
) {
  if write_err != nil {
    process_close_input_or_output(state, &spawn.Input_Descriptor)
    return
  }
  spawn.Input = spawn.Input[count:]
  if len(spawn.Input) == 0 {
    process_close_input_or_output(state, &spawn.Input_Descriptor)
  }
}

// Close one pipe end once and mark it released.
func process_close_input_or_output(state *Operating_System, descriptor *int) {
  if *descriptor < 0 {
    return
  }
  operating_system_descriptor_release(state, *descriptor)
  *descriptor = -1
}

// Arm every spawn pipe whose previous pass retired and whose end is still open. Flush loop call
// this once each pass, thus pass rearm from drain, not from inside its own callback. That keep
// each pipe to one operation in flight, and leave arming in one place instead of cycle between
// arm and its handler.
func process_arm_pipes(state *Operating_System) {
  for _, spawn := range state.Spawns {
    if process_pipe_idle(spawn.Output_Descriptor, &spawn.Output_Completion) {
      if !spawn.Output_Stream_Completion.Armed {
        process_read_output(state, spawn)
      }
    }
    if process_pipe_idle(spawn.Error_Descriptor, &spawn.Error_Completion) {
      if !spawn.Error_Stream_Completion.Armed {
        process_read_error(state, spawn)
      }
    }
    if process_pipe_idle(spawn.Input_Descriptor, &spawn.Input_Completion) {
      process_write_input(state, spawn)
    }
  }
}

// Report whether pipe end is open and carry no operation in flight.
func process_pipe_idle(descriptor int, completion *nbio.Completion) (idle bool) {
  if descriptor < 0 {
    return false
  }
  return !completion.Armed
}

// Arm write that feed child standard input, or close end when request supply none. Child read
// end-of-file only once this end is closed.
func process_write_input(state *Operating_System, spawn *Spawn) {
  if len(spawn.Input) == 0 {
    process_close_input_or_output(state, &spawn.Input_Descriptor)
    return
  }
  operating_system_submit(&spawn.Input_Completion)
  operating_system_pipe_write(
    state,
    &spawn.Input_Completion,
    spawn.Input_Descriptor,
    spawn.Input,
    process_input_written,
  )
}

func process_input_written(completion nbio.Completion_Handle) {
  spawn := spawn_of(completion, unsafe.Offsetof(Spawn{}.Input_Completion))
  process_input_pass(spawn.Operating_System, spawn, completion.Data, completion.Error)
}

// Arm exit watch. Darwin watch process identifier through EVFILT_PROC, Linux poll pidfd, thus
// neither wait in wait4.
func process_watch_exit(state *Operating_System, spawn *Spawn) {
  operating_system_submit(&spawn.Exit_Completion)
  operating_system_operation_submit(
    state,
    operating_system_operation_acquire(
      state,
      Operating_System_Operation{
        Completion:         &spawn.Exit_Completion,
        Kind:               OPERATING_SYSTEM_OPERATION_PROCESS_EXIT,
        Descriptor:         spawn.Exit_Descriptor,
        Process_Identifier: spawn.Identifier,
        Deliver:            process_exited,
      },
    ),
  )
}

func process_exited(completion nbio.Completion_Handle) {
  spawn := spawn_of(completion, unsafe.Offsetof(Spawn{}.Exit_Completion))
  process_exit(spawn.Operating_System, spawn)
}

// Reap exited child and record its outcome. Reap is unconditional and happen here alone, thus
// spawn whose caller already retired on its deadline still release kernel process entry.
func process_exit(state *Operating_System, spawn *Spawn) {
  defer process_finish(state, spawn)
  // Read before reap: Darwin answer only while the exited child is still a zombie.
  counters := process_counters_read(spawn)
  process_counters_close(spawn)
  exit, usage, reap_err := process_reap(spawn.Identifier)
  spawn.Exited = true
  if !(spawn.Exit_Descriptor < 0) {
    syscall.Close(spawn.Exit_Descriptor)
    spawn.Exit_Descriptor = -1
  }
  // Grandchild that inherited pipes outlive child and hold them open. Bound that drain from
  // exit, same way exec.Cmd WaitDelay did, thus child that finish at once does not wait out
  // its whole deadline for descendant it left behind.
  process_bound_cleanup(state, spawn)
  if reap_err != nil {
    return
  }
  usage.Cycle_Count = counters.Cycle_Count
  usage.Instruction_Count = counters.Instruction_Count
  usage.Cache_Reference_Count = counters.Cache_Reference_Count
  usage.Cache_Miss_Count = counters.Cache_Miss_Count
  usage.Branch_Miss_Count = counters.Branch_Miss_Count
  spawn.Result.Exit = exit
  spawn.Result.Usage = usage
}

// Fork child. Counter setup and fork share one OS thread, because a Linux perf event opened for
// the calling thread reach only child that thread fork.
func process_fork(spawn *Spawn, path string, argv []string) (identifier int, err error) {
  runtime.LockOSThread()
  defer runtime.UnlockOSThread()
  process_counters_open(spawn)
  identifier, _, err = syscall.StartProcess(path, argv, &spawn.Attributes)
  return identifier, err
}

// Kill whole process group of child when its deadline win, then bound drain of pipes a surviving
// grandchild may still hold open.
func process_expire(state *Operating_System, spawn *Spawn) {
  if spawn.Delivered {
    return
  }
  spawn.Expired = true
  syscall.Kill(-spawn.Identifier, syscall.SIGKILL)
  process_bound_cleanup(state, spawn)
}

// Start bounded pipe drain once. Both child exit and deadline reach it, and only first arm timer.
// It is no-op when both pipes already ended, thus spawn that finished cleanly leave no timer
// behind.
func process_bound_cleanup(state *Operating_System, spawn *Spawn) {
  if spawn.Cleanup_Armed {
    return
  }
  if process_pipes_drained(spawn) {
    return
  }
  spawn.Cleanup_Armed = true
  operating_system_submit(&spawn.Cleanup_Completion)
  operating_system_internal_timeout(
    state,
    &spawn.Cleanup_Completion,
    PROCESS_CLEANUP_DURATION,
    process_cleanup_fired,
  )
}

func process_cleanup_fired(completion nbio.Completion_Handle) {
  spawn := spawn_of(completion, unsafe.Offsetof(Spawn{}.Cleanup_Completion))
  process_cleanup(spawn.Operating_System, spawn)
}

// Report whether both output pipes reached their end.
func process_pipes_drained(spawn *Spawn) (drained bool) {
  if !spawn.Output_Drained {
    return false
  }
  return spawn.Error_Drained
}

// Force drain to end after pipes stayed open past cleanup bound. Expired is set even when child
// itself exited cleanly, because captured output is now incomplete, and nil error would report
// whole result caller did not get.
//
// Each abandoned pass is retired before its descriptor close, thus no kernel registration outlive
// spawn. Deinit assert operation registry is empty, and pass left armed here would trip it.
func process_cleanup(state *Operating_System, spawn *Spawn) {
  spawn.Expired = true
  spawn.Output_Drained = true
  spawn.Error_Drained = true
  process_retire_pass(state, &spawn.Output_Completion)
  process_retire_pass(state, &spawn.Error_Completion)
  process_retire_pass(state, &spawn.Input_Completion)
  process_close_pipes(state, spawn)
  process_finish(state, spawn)
}

// Retire one pipe pass forced drain abandoned. Completion that already retired hold no registry
// entry, thus this is no-op for it.
func process_retire_pass(state *Operating_System, completion *nbio.Completion) {
  operation := operating_system_operation_find(state, completion.Kernel_Identifier)
  if operation == nil {
    return
  }
  if operation.Completion != completion {
    return
  }
  platform_expire_operation(state, operation)
  operating_system_operation_complete(state, operation, -1, nbio.Canceled)
}

// Deliver result once child was reaped and both pipes ended. It run on every one of those paths,
// and retire caller completion exactly once.
func process_finish(state *Operating_System, spawn *Spawn) {
  if spawn.Delivered {
    return
  }
  if !spawn.Exited {
    return
  }
  if !spawn.Output_Drained {
    return
  }
  if !spawn.Error_Drained {
    return
  }
  if spawn.Output_Stream_Completion.Armed {
    return
  }
  if spawn.Error_Stream_Completion.Armed {
    return
  }
  operating_system_internal_timeout_cancel(state, &spawn.Deadline_Completion)
  operating_system_internal_timeout_cancel(state, &spawn.Cleanup_Completion)
  spawn.Delivered = true
  operating_system_spawn_remove(state, spawn)
  process_close_pipes(state, spawn)
  spawn.Result.Usage.Wall = time.Duration(
    int64(time.Clock_Now_Monotonic(state.Host)) - int64(spawn.Started),
  )
  if spawn.Expired {
    spawn.Final_Error = nbio.Deadline_Exceeded
  } else if spawn.Stream_Error != nil {
    spawn.Final_Error = spawn.Stream_Error
  }
  process_publish(state, spawn)
}

// Queue the caller completion. The slot carries result and callback to delivery, and Backend
// carries the slot, since a caller completion is not a field of the slot. The caller's own
// Backend value is kept and put back before its callback runs.
func process_publish(state *Operating_System, spawn *Spawn) {
  completion := spawn.Completion
  spawn.Caller_Backend = completion.Backend
  completion.Backend = nbio.State(spawn)
  completion.Callback = process_delivered
  operating_system_completion_add(state, completion)
}

// One process completion reaches its caller: the extension count drops as the callback runs,
// thus the loop sees the retirement and the caller sees the result in one step. The slot frees
// before the callback so the callback may spawn again into it; locals keep what it receives.
func process_delivered(completion nbio.Completion_Handle) {
  spawn := completion.Backend.(*Spawn)
  completion.Backend = spawn.Caller_Backend
  state := spawn.Operating_System
  callback := spawn.Callback
  result := spawn.Result
  err := spawn.Final_Error
  spawn_reset(spawn)
  state.Extension_Submitted--
  callback(completion, result, err)
}

func operating_system_spawn_remove(state *Operating_System, spawn *Spawn) {
  found := false
  kept := state.Spawns[:0]
  for _, candidate := range state.Spawns {
    if candidate == spawn {
      found = true
      continue
    }
    kept = append(kept, candidate)
  }
  state.Spawns = kept
  aver.Always(found, "A reaped child occupied one caller-owned registry slot.")
}

// Start forwarding for child unless request keep signals. Registration happen once, and stay
// until a signal arrive with no forwarding child to receive it.
func process_forward_start(state *Operating_System, spawn *Spawn, request nbio.Process_Request) {
  if request.Signals_Kept {
    return
  }
  spawn.Signals_Forwarded = true
  if state.Forward_Notified {
    return
  }
  if state.Forwarded.Channel == nil {
    state.Forwarded = operating_system_signal_channel(SIGNAL_QUEUE_DEPTH)
  }
  state.Forward_Notified = true
  // Each is a signal a terminal or supervisor aim at this process to end it. The child is out
  // of this process group, thus each is owed to it too.
  operating_system_signal_notify(state.Forwarded, syscall.SIGINT)
  operating_system_signal_notify(state.Forwarded, syscall.SIGTERM)
  operating_system_signal_notify(state.Forwarded, syscall.SIGHUP)
  operating_system_signal_notify(state.Forwarded, syscall.SIGQUIT)
}

// Report whether any child still in the registry forward signals.
func operating_system_forward_live(state *Operating_System) (live bool) {
  for _, spawn := range state.Spawns {
    if spawn.Signals_Forwarded {
      return true
    }
  }
  return false
}

// Pass every received signal on to each forwarding child's group. A reaped leader's identifier
// may be reused, thus only child not yet exited receive it. A signal no child is owed was meant
// for this process: registration stop and signal is raised again, thus its default action run.
func operating_system_forward_signals(state *Operating_System) {
  if !state.Forward_Notified {
    return
  }
  for index := 0; index < SIGNAL_QUEUE_DEPTH; index++ {
    received, found := operating_system_signal_receive(state.Forwarded)
    if !found {
      return
    }
    if !operating_system_forward_live(state) {
      state.Forward_Notified = false
      operating_system_signal_stop(state.Forwarded)
      syscall.Kill(syscall.Getpid(), received)
      return
    }
    for _, spawn := range state.Spawns {
      if !spawn.Signals_Forwarded {
        continue
      }
      if spawn.Exited {
        continue
      }
      syscall.Kill(-spawn.Identifier, received)
    }
  }
}

// Release every pipe end loop still hold.
func process_close_pipes(state *Operating_System, spawn *Spawn) {
  process_close_input_or_output(state, &spawn.Input_Descriptor)
  process_close_input_or_output(state, &spawn.Output_Descriptor)
  process_close_input_or_output(state, &spawn.Error_Descriptor)
}

// Submit one pipe read through platform scheduler.
func operating_system_pipe_read(
  state *Operating_System,
  completion *nbio.Completion,
  descriptor int,
  buffer []byte,
  deliver nbio.Callback,
) {
  operating_system_operation_submit(
    state,
    operating_system_operation_acquire(
      state,
      Operating_System_Operation{
        Completion: completion,
        Kind:       OPERATING_SYSTEM_OPERATION_PIPE_READ,
        Descriptor: descriptor,
        Buffer:     platform_buffer_limit(buffer),
        Deliver:    deliver,
      },
    ),
  )
}

// Submit one pipe write through platform scheduler.
func operating_system_pipe_write(
  state *Operating_System,
  completion *nbio.Completion,
  descriptor int,
  buffer []byte,
  deliver nbio.Callback,
) {
  operating_system_operation_submit(
    state,
    operating_system_operation_acquire(
      state,
      Operating_System_Operation{
        Completion: completion,
        Kind:       OPERATING_SYSTEM_OPERATION_PIPE_WRITE,
        Descriptor: descriptor,
        Buffer:     platform_buffer_limit(buffer),
        Deliver:    deliver,
      },
    ),
  )
}

// Normalize rusage Maxrss to bytes: Darwin report bytes, Linux report KiB.
func process_rss_bytes(maxrss int64) (size int64) {
  if runtime.GOOS == "linux" {
    return maxrss * 1024
  }
  return maxrss
}

// Wire file operations — read, write, open, create — onto storage.
func operating_system_wire_file(state *Operating_System, loop *nbio.Storage) {
  loop.State = nbio.State(state)
  loop.Read_Procedure = operating_system_storage_read
  loop.Write_Procedure = operating_system_storage_write
  loop.Fsync_Procedure = operating_system_storage_fsync
  loop.Open_At_Procedure = operating_system_storage_open_at
  loop.Mkdir_At_Procedure = operating_system_storage_mkdir_at
  loop.Get_Directory_Entries_Procedure = operating_system_storage_directory_entries
  loop.Status_Procedure = operating_system_storage_status
  loop.Read_Link_Procedure = operating_system_storage_read_link
}

func operating_system_storage_read(
  state_pointer nbio.State,
  completion *nbio.Completion,
  file nbio.File,
  buffer []byte,
  offset int64,
  timeout time.Duration,
  callback nbio.Callback,
) {
  state := state_pointer.(*Operating_System)
  aver.Always(timeout > 0, "A storage read timeout is positive and finite.")
  deadline := platform_storage_deadline(state, timeout)
  operating_system_submit(completion)
  if operating_system_descriptor_pipe(state, int(file)) {
    aver.Always(
      offset == nbio.FILE_POSITION_CURRENT,
      "A pipe transfer names no offset, because pipe is not seekable.",
    )
    operating_system_pipe_transfer(
      state,
      completion,
      file,
      buffer,
      timeout,
      callback,
      OPERATING_SYSTEM_OPERATION_PIPE_READ,
    )
    return
  }
  operating_system_read(state, completion, file, buffer, offset, deadline, callback)
}

func operating_system_storage_write(
  state_pointer nbio.State,
  completion *nbio.Completion,
  file nbio.File,
  buffer []byte,
  offset int64,
  timeout time.Duration,
  callback nbio.Callback,
) {
  state := state_pointer.(*Operating_System)
  aver.Always(timeout > 0, "A storage write timeout is positive and finite.")
  deadline := platform_storage_deadline(state, timeout)
  operating_system_submit(completion)
  if operating_system_descriptor_pipe(state, int(file)) {
    aver.Always(
      offset == nbio.FILE_POSITION_CURRENT,
      "A pipe transfer names no offset, because pipe is not seekable.",
    )
    operating_system_pipe_transfer(
      state,
      completion,
      file,
      buffer,
      timeout,
      callback,
      OPERATING_SYSTEM_OPERATION_PIPE_WRITE,
    )
    return
  }
  operating_system_write(state, completion, file, buffer, offset, deadline, callback)
}

// Pipe can stay empty or full forever, unlike file, thus its transfer carry socket deadline on
// both platforms, not storage deadline Darwin ignore.
func operating_system_pipe_transfer(
  state *Operating_System,
  completion *nbio.Completion,
  file nbio.File,
  buffer []byte,
  timeout time.Duration,
  callback nbio.Callback,
  kind Operating_System_Operation_Kind,
) {
  if len(buffer) == 0 {
    operating_system_empty_transfer_complete(state, completion, callback)
    return
  }
  operation := operating_system_operation_acquire(
    state,
    Operating_System_Operation{
      Completion: completion,
      Kind:       kind,
      Descriptor: int(file),
      Buffer:     platform_buffer_limit(buffer),
      Deadline: time.Clock_Now_Monotonic(state.Host) +
        time.Monotonic_Moment(timeout),
      Deadline_Span: operating_system_timeout_span(timeout),
      Deliver:       callback,
    },
  )
  operating_system_operation_submit(state, operation)
}

func operating_system_storage_fsync(
  state_pointer nbio.State,
  completion *nbio.Completion,
  file nbio.File,
  timeout time.Duration,
  callback nbio.Callback,
) {
  state := state_pointer.(*Operating_System)
  aver.Always(timeout > 0, "A storage fsync timeout is positive and finite.")
  deadline := platform_storage_deadline(state, timeout)
  operating_system_submit(completion)
  operating_system_fsync(state, completion, file, deadline, callback)
}

func operating_system_storage_open_at(
  state_pointer nbio.State,
  completion *nbio.Completion,
  directory nbio.File,
  file_path string,
  options nbio.Open_At_Options,
  callback nbio.Callback,
) {
  state := state_pointer.(*Operating_System)
  aver.Always(
    options.Flags & ^nbio.OPEN_AT_NO_FOLLOW == 0,
    "Open_At options contain only known flags.",
  )
  operating_system_submit(completion)
  operating_system_open_at(state, completion, directory, file_path, options, callback)
}

func operating_system_storage_mkdir_at(
  state_pointer nbio.State,
  completion *nbio.Completion,
  directory nbio.File,
  file_path string,
  permissions nbio.File_Permissions,
  callback nbio.Callback,
) {
  state := state_pointer.(*Operating_System)
  operating_system_submit(completion)
  operating_system_mkdir_at(state, completion, directory, file_path, permissions, callback)
}

func operating_system_storage_directory_entries(
  state_pointer nbio.State,
  completion *nbio.Completion,
  directory nbio.File,
  buffer []byte,
  entries []nbio.Directory_Entry,
  callback nbio.Callback,
) {
  state := state_pointer.(*Operating_System)
  operating_system_submit(completion)
  operating_system_directory_pass(state, completion, directory, buffer, entries, callback)
}

func operating_system_storage_status(
  _ nbio.State,
  path string,
) (status nbio.File_Status, err error) {
  return file_status(path)
}

func operating_system_storage_read_link(
  _ nbio.State,
  path string,
  destination []byte,
) (count int, err error) {
  return file_read_link(path, destination)
}

// Wire loop own control plane — timer and cross-thread event — onto vtable shared/time own.
// Close primitives are not here: they name descriptor, which is business of IO surface, not of
// timeline.
func operating_system_wire_timer(state *Operating_System, pump *nbio.Timeline) {
  pump.State = nbio.State(state)
  pump.Submit = operating_system_timeline_submit
  pump.Open_Event = operating_system_timeline_open_event
  pump.Event_Listen = operating_system_timeline_event_listen
  pump.Event_Trigger = operating_system_timeline_event_trigger
  pump.Close_Event = operating_system_timeline_close_event
}

func operating_system_timeline_submit(
  state nbio.State,
  completion *nbio.Completion,
  delay time.Duration,
  callback nbio.Callback,
) {
  system := state.(*Operating_System)
  operating_system_submit(completion)
  operating_system_timeout(system, system.Host, completion, delay, callback)
}

func operating_system_timeline_open_event(
  state nbio.State,
) (event nbio.Event, err error) {
  return platform_event_open(state.(*Operating_System))
}

func operating_system_timeline_event_listen(
  state nbio.State,
  event nbio.Event,
  completion *nbio.Completion,
  callback nbio.Callback,
) {
  system := state.(*Operating_System)
  operating_system_submit(completion)
  operating_system_event_listen(system, event, completion, callback)
}

func operating_system_timeline_event_trigger(
  state nbio.State,
  event nbio.Event,
  completion *nbio.Completion,
) {
  platform_event_trigger(
    state.(*Operating_System), event, completion.Kernel_Identifier,
  )
}

func operating_system_timeline_close_event(state nbio.State, event nbio.Event) {
  system := state.(*Operating_System)
  operating_system_assert_event_drained(system, event)
  platform_event_close(system, event)
}

// Wire members both halves share: pipe that make descriptors, asynchronous close of any
// descriptor, and leak check that state run released every one it took.
func operating_system_wire_close(loop *nbio.IO) {
  loop.Pipe_Procedure = operating_system_pipe_procedure
  loop.Close_Procedure = operating_system_close_procedure
  loop.Deinit_Procedure = operating_system_deinit_procedure
}

func operating_system_pipe_procedure(
  state_pointer nbio.State,
) (read nbio.File, write nbio.File, err error) {
  state := state_pointer.(*Operating_System)
  read_end, write_end, open_err := pipe_open_retained()
  if open_err != nil {
    return nbio.File(-1), nbio.File(-1), open_err
  }
  operating_system_descriptor_add(state, read_end)
  operating_system_descriptor_add(state, write_end)
  operating_system_descriptor_mark_pipe(state, read_end)
  operating_system_descriptor_mark_pipe(state, write_end)
  return nbio.File(read_end), nbio.File(write_end), nil
}

func operating_system_close_procedure(
  state_pointer nbio.State,
  completion *nbio.Completion,
  file nbio.File,
  callback nbio.Callback,
) {
  state := state_pointer.(*Operating_System)
  operating_system_assert_file_drained(state, file)
  operating_system_submit(completion)
  operating_system_close(state, completion, file, callback)
}

func operating_system_deinit_procedure(state_pointer nbio.State) {
  state := state_pointer.(*Operating_System)
  aver.Always(
    operating_system_descriptor_count(state) == 0,
    "Every descriptor the backend opened is closed before Deinit.",
  )
}

// Register one Event listener. Platform decide whether it is persistent EVFILT_USER
// listener, or eventfd read operation.
func operating_system_event_listen(
  state *Operating_System,
  event nbio.Event,
  completion *nbio.Completion,
  callback nbio.Callback,
) {
  operation := operating_system_operation_acquire(
    state,
    Operating_System_Operation{
      Completion: completion,
      Kind:       OPERATING_SYSTEM_OPERATION_EVENT,
      Descriptor: int(event),
      Deliver:    callback,
    },
  )
  operating_system_operation_register(state, operation)
  listen_err := platform_event_listen(state, operation)
  aver.Always(listen_err == nil, "An Event listener arms successfully.")
}

// Assert no Event listener stay armed before backend Event resource is closed.
func operating_system_assert_event_drained(state *Operating_System, event nbio.Event) {
  armed := false
  for _, operation := range state.Operations {
    if operation.Kind == OPERATING_SYSTEM_OPERATION_EVENT {
      if operation.Descriptor == int(event) {
        armed = true
      }
    }
  }
  aver.Always(!armed, "An Event listener is drained before Close_Event.")
}

// Assert no submitted operation still borrow file. Owner join every submitted operation before
// close, and this fail-closed check stop old kernel completion from race with recycled descriptor
// number.
func operating_system_assert_file_drained(state *Operating_System, file nbio.File) {
  borrowed := false
  for _, operation := range state.Operations {
    if operation.Descriptor == int(file) {
      borrowed = true
    }
  }
  aver.Always(!borrowed,
    "A descriptor is drained before Close releases it.")
}

// Wire socket lifecycle separately from byte transfers so neither boundary hides in one table.
func operating_system_wire_socket(state *Operating_System, loop *nbio.Network) {
  loop.State = nbio.State(state)
  loop.Bind_Procedure = operating_system_socket_bind
  loop.Listen_Socket_Procedure = operating_system_socket_listen
  loop.Get_Socket_Name_Procedure = operating_system_socket_name
  loop.Accept_Procedure = operating_system_socket_accept
  loop.Socket_TCP_Procedure = operating_system_socket_tcp
  loop.Socket_UDP_Procedure = operating_system_socket_udp
  loop.Connect_Procedure = operating_system_socket_connect
  loop.Shutdown_Procedure = operating_system_socket_shutdown
  loop.Peer_Address_Procedure = operating_system_socket_peer_address
  operating_system_wire_socket_transfers(loop)
}

func operating_system_socket_bind(
  _ nbio.State,
  socket nbio.File,
  address nbio.Address,
) (err error) {
  return socket_bind(int(socket), address)
}

func operating_system_socket_listen(
  _ nbio.State, socket nbio.File, backlog uint32,
) (err error) {
  return socket_listen_mark(int(socket), backlog)
}

func operating_system_socket_name(
  _ nbio.State,
  socket nbio.File,
  destination *nbio.Address,
) (err error) {
  return socket_name(int(socket), destination)
}

func operating_system_socket_accept(
  state_pointer nbio.State,
  completion *nbio.Completion,
  listener nbio.File,
  timeout time.Duration,
  callback nbio.Callback,
) {
  state := state_pointer.(*Operating_System)
  aver.Always(timeout > 0, "An accept timeout is positive and finite.")
  operating_system_submit(completion)
  operating_system_accept(state, completion, listener, timeout, callback)
}

func operating_system_socket_tcp(
  state_pointer nbio.State,
  family nbio.Address_Family,
  options nbio.TCP_Options,
) (socket nbio.File, err error) {
  descriptor, open_err := socket_open_tcp(family, options)
  if open_err != nil {
    return nbio.File(-1), open_err
  }
  operating_system_descriptor_add(state_pointer.(*Operating_System), descriptor)
  return nbio.File(descriptor), nil
}

func operating_system_socket_udp(
  state_pointer nbio.State,
  family nbio.Address_Family,
  options nbio.UDP_Options,
) (socket nbio.File, err error) {
  descriptor, open_err := socket_open_udp(family, options)
  if open_err != nil {
    return nbio.File(-1), open_err
  }
  operating_system_descriptor_add(state_pointer.(*Operating_System), descriptor)
  return nbio.File(descriptor), nil
}

func operating_system_socket_connect(
  state_pointer nbio.State,
  completion *nbio.Completion,
  socket nbio.File,
  address nbio.Address,
  timeout time.Duration,
  callback nbio.Callback,
) {
  state := state_pointer.(*Operating_System)
  aver.Always(timeout > 0, "A connect timeout is positive and finite.")
  operating_system_submit(completion)
  operating_system_connect(state, completion, socket, address, timeout, callback)
}

func operating_system_socket_shutdown(
  _ nbio.State,
  socket nbio.File,
  how nbio.Shutdown_How,
) (err error) {
  return socket_shutdown(int(socket), how)
}

func operating_system_socket_peer_address(
  _ nbio.State,
  file nbio.File,
  destination *nbio.Address,
) (err error) {
  return socket_peer_address(int(file), destination)
}

// Byte transfers share one required timeout contract on TCP and UDP descriptors.
func operating_system_wire_socket_transfers(loop *nbio.Network) {
  loop.Receive_Procedure = operating_system_socket_receive
  loop.Send_Procedure = operating_system_socket_send
}

func operating_system_socket_receive(
  state_pointer nbio.State,
  completion *nbio.Completion,
  socket nbio.File,
  buffer []byte,
  timeout time.Duration,
  callback nbio.Callback,
) {
  state := state_pointer.(*Operating_System)
  aver.Always(timeout > 0, "A receive timeout is positive and finite.")
  operating_system_submit(completion)
  operating_system_receive(state, completion, socket, buffer, timeout, callback)
}

func operating_system_socket_send(
  state_pointer nbio.State,
  completion *nbio.Completion,
  socket nbio.File,
  buffer []byte,
  timeout time.Duration,
  callback nbio.Callback,
) {
  state := state_pointer.(*Operating_System)
  aver.Always(timeout > 0, "A send timeout is positive and finite.")
  operating_system_submit(completion)
  operating_system_send(state, completion, socket, buffer, timeout, callback)
}

// Submit file read through platform scheduler.
func operating_system_read(
  state *Operating_System,
  completion *nbio.Completion,
  file nbio.File,
  buffer []byte,
  offset int64,
  deadline time.Monotonic_Moment,
  callback nbio.Callback,
) {
  if len(buffer) == 0 {
    operating_system_empty_transfer_complete(state, completion, callback)
    return
  }
  operation := operating_system_operation_acquire(
    state,
    Operating_System_Operation{
      Completion: completion,
      Kind:       OPERATING_SYSTEM_OPERATION_READ,
      Descriptor: int(file),
      Buffer:     platform_buffer_limit(buffer),
      Offset:     uint64(offset),
      Deadline:   deadline,
      Deliver:    callback,
    },
  )
  operating_system_operation_submit(state, operation)
}

// Submit file write through platform scheduler.
func operating_system_write(
  state *Operating_System,
  completion *nbio.Completion,
  file nbio.File,
  buffer []byte,
  offset int64,
  deadline time.Monotonic_Moment,
  callback nbio.Callback,
) {
  if len(buffer) == 0 {
    operating_system_empty_transfer_complete(state, completion, callback)
    return
  }
  operation := operating_system_operation_acquire(
    state,
    Operating_System_Operation{
      Completion: completion,
      Kind:       OPERATING_SYSTEM_OPERATION_WRITE,
      Descriptor: int(file),
      Buffer:     platform_buffer_limit(buffer),
      Offset:     uint64(offset),
      Deadline:   deadline,
      Deliver:    callback,
    },
  )
  operating_system_operation_submit(state, operation)
}

// Submit asynchronous fsync operation.
func operating_system_fsync(
  state *Operating_System,
  completion *nbio.Completion,
  file nbio.File,
  deadline time.Monotonic_Moment,
  callback nbio.Callback,
) {
  operation := operating_system_operation_acquire(
    state,
    Operating_System_Operation{
      Completion: completion,
      Kind:       OPERATING_SYSTEM_OPERATION_FSYNC,
      Descriptor: int(file),
      Deadline:   deadline,
      Deliver:    callback,
    },
  )
  operating_system_operation_submit(state, operation)
}

// No kernel request exists for an empty transfer, thus callback delivery needs only the queue.
func operating_system_empty_transfer_complete(
  state *Operating_System,
  completion *nbio.Completion,
  callback nbio.Callback,
) {
  completion.Data = 0
  completion.Error = nil
  completion.Callback = callback
  operating_system_completion_add(state, completion)
}

// Submit asynchronous openat operation with NUL-terminated path owned until callback
// retire.
func operating_system_open_at(
  state *Operating_System,
  completion *nbio.Completion,
  directory nbio.File,
  file_path string,
  options nbio.Open_At_Options,
  callback nbio.Callback,
) {
  descriptor := int(directory)
  if directory == nbio.DIRECTORY_CURRENT {
    descriptor = platform_current_directory()
  }
  operation := operating_system_operation_acquire(
    state,
    Operating_System_Operation{
      Completion:   completion,
      Kind:         OPERATING_SYSTEM_OPERATION_OPEN_AT,
      Descriptor:   descriptor,
      Open_Options: options,
      Deliver:      callback,
    },
  )
  operating_system_operation_submit_path(state, operation, file_path)
}

// Submit one directory pass. getdents has no asynchronous form on either backend, thus read run
// inline and completion retire on next drain, exactly as Status do.
func operating_system_directory_pass(
  state *Operating_System,
  completion *nbio.Completion,
  directory nbio.File,
  buffer []byte,
  entries []nbio.Directory_Entry,
  callback nbio.Callback,
) {
  completion.Data, completion.Error = file_directory_pass(int(directory), buffer, entries)
  completion.Callback = callback
  operating_system_completion_add(state, completion)
}

// Submit one mkdirat through platform scheduler. Permissions travel in Open_Options because both
// operations carry path and creation permissions, and scheduler already hold that field.
func operating_system_mkdir_at(
  state *Operating_System,
  completion *nbio.Completion,
  directory nbio.File,
  file_path string,
  permissions nbio.File_Permissions,
  callback nbio.Callback,
) {
  descriptor := int(directory)
  if directory == nbio.DIRECTORY_CURRENT {
    descriptor = platform_current_directory()
  }
  operation := operating_system_operation_acquire(
    state,
    Operating_System_Operation{
      Completion:   completion,
      Kind:         OPERATING_SYSTEM_OPERATION_MKDIR_AT,
      Descriptor:   descriptor,
      Open_Options: nbio.Open_At_Options{Permissions: permissions},
      Deliver:      callback,
    },
  )
  operating_system_operation_submit_path(state, operation, file_path)
}

// Schedule positive timeout to fire when clock pass its deadline.
func operating_system_timeout(
  state *Operating_System,
  host time.Clock,
  completion *nbio.Completion,
  duration time.Duration,
  callback nbio.Callback,
) {
  if platform_uses_kernel_timeouts() {
    operation := operating_system_operation_acquire(
      state,
      Operating_System_Operation{
        Completion: completion,
        Kind:       OPERATING_SYSTEM_OPERATION_TIMEOUT,
        Descriptor: -1,
        Timespec:   operating_system_timeout_span(duration),
        Deliver:    callback,
      },
    )
    operating_system_operation_submit(state, operation)
    return
  }
  completion.Ready_At = time.Clock_Now_Monotonic(host) + time.Monotonic_Moment(duration)
  completion.Callback = callback
  operating_system_insert(state, completion)
}

// Keep backend-owned process bounds in the loop queue because their owner must withdraw them
// when another terminal event wins. Public Linux deadlines stay in io_uring and retire by kernel
// completion instead.
func operating_system_internal_timeout(
  state *Operating_System,
  completion *nbio.Completion,
  duration time.Duration,
  callback nbio.Callback,
) {
  completion.Ready_At = time.Clock_Now_Monotonic(state.Host) +
    time.Monotonic_Moment(duration)
  completion.Callback = callback
  operating_system_insert(state, completion)
}

// Withdraw one backend-owned bound before its resource can finish. It can already be ready but
// not delivered when another completion earlier in the same pass wins, thus both queues belong
// to cancellation.
func operating_system_internal_timeout_cancel(
  state *Operating_System,
  completion *nbio.Completion,
) {
  if !completion.Armed {
    return
  }
  state.Timeouts = operating_system_completion_remove(state.Timeouts, completion)
  state.Completed = operating_system_completion_remove(state.Completed, completion)
  completion.Armed = false
}

// Preserve queue order because equal-time completion order is observable by callbacks.
func operating_system_completion_remove(
  completions []*nbio.Completion,
  removed *nbio.Completion,
) (kept []*nbio.Completion) {
  kept = completions[:0]
  for _, completion := range completions {
    if completion != removed {
      kept = append(kept, completion)
    }
  }
  return kept
}

// Build driver — loop-advancing capability — over state. Only composition root or test hold it,
// never code that only submit IO.
func operating_system_to_driver(state *Operating_System) (driver nbio.Driver) {
  return nbio.Driver{
    State:     nbio.State(state),
    Run:       operating_system_driver_run,
    Run_For:   operating_system_driver_run_for,
    Run_Until: operating_system_driver_run_until,
    Deinit:    operating_system_driver_deinit,
  }
}

func operating_system_driver_run(state nbio.State) (err error) {
  system := state.(*Operating_System)
  operating_system_drive_begin(system)
  defer operating_system_drive_end(system)
  return operating_system_run(system)
}

func operating_system_driver_run_for(
  state nbio.State, duration time.Duration,
) (err error) {
  system := state.(*Operating_System)
  operating_system_drive_begin(system)
  defer operating_system_drive_end(system)
  return operating_system_run_for(system, duration)
}

func operating_system_driver_run_until(
  state nbio.State,
  timeout time.Duration,
  done func() (finished bool),
) (completed bool, err error) {
  system := state.(*Operating_System)
  operating_system_drive_begin(system)
  defer operating_system_drive_end(system)
  return operating_system_drive_until(system, timeout, done)
}

func operating_system_driver_deinit(state nbio.State) {
  operating_system_deinitialize(state.(*Operating_System))
}

// Drive until done, or until host deadline. Propagate every scheduler error.
func operating_system_drive_until(
  state *Operating_System,
  timeout time.Duration,
  done func() (finished bool),
) (completed bool, err error) {
  deadline := time.Clock_Now_Monotonic(state.Host) + time.Monotonic_Moment(timeout)
  for !done() {
    wait, expired := operating_system_run_until_wait(state, deadline, timeout)
    if expired {
      return false, nil
    }
    flush_err := operating_system_flush(state, wait)
    if flush_err != nil {
      return false, flush_err
    }
  }
  return true, nil
}

// Operating system run until wait choose nonblocking first pass, then nearest deadline.
func operating_system_run_until_wait(
  state *Operating_System,
  deadline time.Monotonic_Moment,
  timeout time.Duration,
) (wait time.Monotonic_Moment, expired bool) {
  if len(state.Completed) > 0 {
    return 0, false
  }
  now := time.Clock_Now_Monotonic(state.Host)
  if !(timeout < 0) {
    if !(now < deadline) {
      return 0, true
    }
  }
  wait = POLL_FOREVER
  if !(timeout < 0) {
    wait = deadline - now
  }
  return operating_system_wait_cap(state, wait), false
}

// Report whether operation can wake unbounded drive.
func operating_system_in_flight(state *Operating_System) (in_flight bool) {
  if platform_in_flight(state) {
    return true
  }
  if len(state.Timeouts) > 0 {
    return true
  }
  if len(state.Signal_Waiters) > 0 {
    return true
  }
  return len(state.Spawns) > 0
}

// Run pump as top-level drive. Panic when drive is already in progress, thus Run* called from
// inside completion callback fail loud instead of re-enter driver. Internal run functions call
// one another direct, not through here, thus own iteration of drive does not trip it.
func operating_system_drive_begin(state *Operating_System) {
  aver.Always(
    !state.Drive_Active,
    "A drive begins at top level, never from within a completion callback.",
  )
  state.Drive_Active = true
}

func operating_system_drive_end(state *Operating_System) {
  state.Drive_Active = false
}

// Run one nonblocking flush.
func operating_system_run(state *Operating_System) (err error) {
  return operating_system_flush(state, 0)
}

// Run flush passes until duration elapse, with deadline passed direct to platform.
func operating_system_run_for(state *Operating_System, duration time.Duration) (err error) {
  deadline := time.Clock_Now_Monotonic(state.Host) + time.Monotonic_Moment(duration)
  for time.Clock_Now_Monotonic(state.Host) < deadline {
    now := time.Clock_Now_Monotonic(state.Host)
    if !(now < deadline) {
      return nil
    }
    wait := operating_system_wait_cap(state, deadline-now)
    flush_err := operating_system_flush(state, wait)
    if flush_err != nil {
      return flush_err
    }
  }
  return nil
}

// Operating system flush lets a ready kernel event produce its terminal syscall result before
// expiration retires registrations that remain pending.
func operating_system_flush(state *Operating_System, wait time.Monotonic_Moment) (err error) {
  expire_err := operating_system_expire(state)
  if expire_err != nil {
    return expire_err
  }
  operating_system_signals(state)
  operating_system_forward_signals(state)
  process_arm_pipes(state)
  platform_err := platform_run(state, wait)
  if platform_err != nil {
    return platform_err
  }
  operation_expire_err := operating_system_expire_operations(state)
  if operation_expire_err != nil {
    return operation_expire_err
  }
  operating_system_flush_completed(state)
  return platform_flush_submissions(state)
}

// Operating system wait cap pull outer wait in to nearest Darwin timeout and signal poll.
func operating_system_wait_cap(
  state *Operating_System,
  wait time.Monotonic_Moment,
) (capped time.Monotonic_Moment) {
  capped = wait
  // A forwarded signal is discovered by poll, like a watched one.
  if operating_system_forward_live(state) {
    interval := time.Monotonic_Moment(SIGNAL_POLL_INTERVAL)
    if capped < 0 {
      capped = interval
    } else if interval < capped {
      capped = interval
    }
  }
  if len(state.Timeouts) > 0 {
    until_timeout := state.Timeouts[0].Ready_At - time.Clock_Now_Monotonic(state.Host)
    if capped < 0 {
      capped = until_timeout
    } else if until_timeout < capped {
      capped = until_timeout
    }
  }
  if len(state.Signal_Waiters) > 0 {
    interval := time.Monotonic_Moment(SIGNAL_POLL_INTERVAL)
    if capped < 0 {
      capped = interval
    } else if interval < capped {
      capped = interval
    }
    until_signal := state.Signal_Waiters[0].Deadline -
      time.Clock_Now_Monotonic(state.Host)
    for index := 1; index < len(state.Signal_Waiters); index++ {
      candidate := state.Signal_Waiters[index].Deadline -
        time.Clock_Now_Monotonic(state.Host)
      if candidate < until_signal {
        until_signal = candidate
      }
    }
    if capped < 0 {
      capped = until_signal
    } else if until_signal < capped {
      capped = until_signal
    }
  }
  if !platform_uses_kernel_timeouts() {
    for _, operation := range state.Operations {
      if operation.Deadline == 0 {
        continue
      }
      now := time.Clock_Now_Monotonic(state.Host)
      until_operation := operation.Deadline - now
      if capped < 0 {
        capped = until_operation
      } else if until_operation < capped {
        capped = until_operation
      }
    }
  }
  if capped < 0 {
    aver.Always(
      operating_system_in_flight(state),
      "An unbounded run holds an operation in flight that can advance it.",
    )
  }
  return capped
}

// Move every elapsed timeout and finite extension waiter into completed queue.
func operating_system_expire(state *Operating_System) (err error) {
  now := time.Clock_Now_Monotonic(state.Host)
  for len(state.Timeouts) > 0 {
    if state.Timeouts[0].Ready_At > now {
      break
    }
    expired := operating_system_completion_pop(&state.Timeouts)
    operating_system_completion_add(state, expired)
  }
  operating_system_expire_signals(state, now)
  return nil
}

// Retire only operations still pending after the platform consumed every ready event.
func operating_system_expire_operations(state *Operating_System) (err error) {
  if platform_uses_kernel_timeouts() {
    return nil
  }
  now := time.Clock_Now_Monotonic(state.Host)
  operation := operating_system_expired_operation(state, now)
  for operation != nil {
    cancel_err := platform_expire_operation(state, operation)
    if cancel_err != nil {
      return cancel_err
    }
    operating_system_operation_complete(
      state,
      operation,
      operating_system_timeout_result(operation),
      nbio.Deadline_Exceeded,
    )
    operation = operating_system_expired_operation(state, now)
  }
  return nil
}

func operating_system_expired_operation(
  state *Operating_System,
  now time.Monotonic_Moment,
) (expired *Operating_System_Operation) {
  for _, operation := range state.Operations {
    if operation.Deadline != 0 {
      if !(operation.Deadline > now) {
        return operation
      }
    }
  }
  return nil
}

// Run and clear every ready completion callback. Return each completion to idle before its
// callback fire — thus callback may legally resubmit its own completion.
func operating_system_flush_completed(state *Operating_System) {
  for len(state.Completed) > 0 {
    completion := operating_system_completion_pop(&state.Completed)
    aver.Always(completion.Armed, "A delivered completion was armed.")
    completion.Armed = false
    callback := completion.Callback
    completion.Callback = nil
    callback(completion)
  }
}

func operating_system_completion_pop(
  queue *[]*nbio.Completion,
) (completion *nbio.Completion) {
  values := *queue
  completion = values[0]
  copy(values, values[1:])
  values[len(values)-1] = nil
  *queue = values[:len(values)-1]
  return completion
}

// Insert completion into timeout queue in Ready_At order.
func operating_system_insert(state *Operating_System, completion *nbio.Completion) {
  index := 0
  for index < len(state.Timeouts) {
    if state.Timeouts[index].Ready_At > completion.Ready_At {
      break
    }
    index++
  }
  aver.Always(
    len(state.Timeouts) < cap(state.Timeouts),
    "The caller-owned timeout queue has capacity before insertion.",
  )
  state.Timeouts = append(state.Timeouts, nil)
  copy(state.Timeouts[index+1:], state.Timeouts[index:])
  state.Timeouts[index] = completion
}

// Submit accept through kqueue eager/requeue path, or io_uring ACCEPT opcode.
func operating_system_accept(
  state *Operating_System,
  completion *nbio.Completion,
  listener nbio.File,
  timeout time.Duration,
  callback nbio.Callback,
) {
  operation := operating_system_operation_acquire(
    state,
    Operating_System_Operation{
      Completion: completion,
      Kind:       OPERATING_SYSTEM_OPERATION_ACCEPT,
      Descriptor: int(listener),
      Deadline: time.Clock_Now_Monotonic(state.Host) +
        time.Monotonic_Moment(timeout),
      Deadline_Span: operating_system_timeout_span(timeout),
      Deliver:       callback,
    },
  )
  operating_system_operation_submit(state, operation)
}

// Begin connection on caller-owned socket and report only connect result. How backend reach that
// result is its own: io_uring submit connect, and kqueue arm socket for writability, then issue
// connect itself.
func operating_system_connect(
  state *Operating_System,
  completion *nbio.Completion,
  socket nbio.File,
  address nbio.Address,
  timeout time.Duration,
  callback nbio.Callback,
) {
  operation := operating_system_operation_acquire(
    state,
    Operating_System_Operation{
      Completion: completion,
      Kind:       OPERATING_SYSTEM_OPERATION_CONNECT,
      Descriptor: int(socket),
      Address:    address,
      Deadline: time.Clock_Now_Monotonic(state.Host) +
        time.Monotonic_Moment(timeout),
      Deadline_Span: operating_system_timeout_span(timeout),
      Deliver:       callback,
    },
  )
  operating_system_operation_submit(state, operation)
}

// Read once from socket into buffer and report byte count. io_uring submit receive, and kqueue
// arm socket for readability, then read it.
func operating_system_receive(
  state *Operating_System,
  completion *nbio.Completion,
  socket nbio.File,
  buffer []byte,
  timeout time.Duration,
  callback nbio.Callback,
) {
  operation := operating_system_operation_acquire(
    state,
    Operating_System_Operation{
      Completion: completion,
      Kind:       OPERATING_SYSTEM_OPERATION_RECEIVE,
      Descriptor: int(socket),
      Buffer:     platform_buffer_limit(buffer),
      Deadline: time.Clock_Now_Monotonic(state.Host) +
        time.Monotonic_Moment(timeout),
      Deadline_Span: operating_system_timeout_span(timeout),
      Deliver:       callback,
    },
  )
  operating_system_operation_submit(state, operation)
}

// Write buffer once to socket and report byte count. io_uring submit send, and kqueue arm socket
// for writability, then write it.
func operating_system_send(
  state *Operating_System,
  completion *nbio.Completion,
  socket nbio.File,
  buffer []byte,
  timeout time.Duration,
  callback nbio.Callback,
) {
  operation := operating_system_operation_acquire(
    state,
    Operating_System_Operation{
      Completion: completion,
      Kind:       OPERATING_SYSTEM_OPERATION_SEND,
      Descriptor: int(socket),
      Buffer:     platform_buffer_limit(buffer),
      Deadline: time.Clock_Now_Monotonic(state.Host) +
        time.Monotonic_Moment(timeout),
      Deadline_Span: operating_system_timeout_span(timeout),
      Deliver:       callback,
    },
  )
  operating_system_operation_submit(state, operation)
}

// Close file and queue its completion after operating_system_assert_file_drained enforce
// owner-side join.
func operating_system_close(
  state *Operating_System,
  completion *nbio.Completion,
  file nbio.File,
  callback nbio.Callback,
) {
  operation := operating_system_operation_acquire(
    state,
    Operating_System_Operation{
      Completion: completion,
      Kind:       OPERATING_SYSTEM_OPERATION_CLOSE,
      Descriptor: int(file),
      Deliver:    callback,
    },
  )
  operating_system_operation_submit(state, operation)
}

// Operating system deinitialize enforce join-before-deinit contract and release
// scheduler.
func operating_system_deinitialize(state *Operating_System) {
  aver.Always(
    state.Extension_Submitted == 0,
    "Driver Deinit follows joining every repository-extension completion.",
  )
  aver.Always(len(state.Spawns) == 0,
    "Every spawned child is reaped before backend deinit.")
  aver.Always(len(state.Timeouts) == 0,
    "Driver Deinit follows joining every userspace timeout.")
  aver.Always(len(state.Operations) == 0,
    "Driver Deinit follows joining every operation.")
  if state.Signals.Channel != nil {
    operating_system_signal_stop(state.Signals)
  }
  if state.Forward_Notified {
    state.Forward_Notified = false
    operating_system_signal_stop(state.Forwarded)
  }
  platform_deinitialize(state)
}

// Register watcher for signal and start OS notification for it.
func operating_system_watch_signal(
  state *Operating_System,
  completion *nbio.Completion,
  kind nbio.Signal,
  deadline time.Duration,
  callback nbio.Signal_Callback,
) {
  operating_system_signal_ensure(state)
  system := signal_to_operating_system(kind)
  aver.Always(
    len(state.Signal_Waiters) < cap(state.Signal_Waiters),
    "The caller-owned signal-waiter queue has capacity before registration.",
  )
  state.Signal_Waiters = append(
    state.Signal_Waiters,
    Signal_Waiter{
      System:     system,
      Kind:       kind,
      Completion: completion,
      Callback:   callback,
      Deadline:   time.Clock_Now_Monotonic(state.Host) + time.Monotonic_Moment(deadline),
    },
  )
  operating_system_signal_notify(state.Signals, system)
}

// Make buffered signal channel on first watch.
func operating_system_signal_ensure(state *Operating_System) {
  if state.Signals.Channel != nil {
    return
  }
  state.Signals = operating_system_signal_channel(SIGNAL_QUEUE_DEPTH)
}

// Map backend-independent signal to its OS signal.
func signal_to_operating_system(kind nbio.Signal) (system syscall.Signal) {
  if kind == nbio.SIGNAL_INTERRUPT {
    return syscall.SIGINT
  }
  return syscall.SIGTERM
}

// Drain delivered signals without block. Fire matching watchers.
func operating_system_signals(state *Operating_System) {
  for index := 0; index < SIGNAL_QUEUE_DEPTH; index++ {
    received, found := operating_system_signal_receive(state.Signals)
    if !found {
      return
    }
    operating_system_signal_deliver(state, received)
  }
}

// Fire every watcher matching received, one-shot. Keep rest for later delivery.
func operating_system_signal_deliver(
  state *Operating_System, received syscall.Signal,
) {
  kept := state.Signal_Waiters[:0]
  for index := 0; index < len(state.Signal_Waiters); index++ {
    waiter := state.Signal_Waiters[index]
    if waiter.System != received {
      kept = append(kept, waiter)
      continue
    }
    delivered := waiter
    delivered.Completion.Callback = func(_ nbio.Completion_Handle) {
      signal_deliver(state, delivered, delivered.Kind, nil)
    }
    operating_system_completion_add(state, delivered.Completion)
  }
  state.Signal_Waiters = kept
}

// Expire signal watchers before drain of os/signal channel, thus event first observed at deadline
// is discarded and cannot leak into next explicitly rearmed watch.
func operating_system_expire_signals(state *Operating_System, now time.Monotonic_Moment) {
  kept := state.Signal_Waiters[:0]
  for index := 0; index < len(state.Signal_Waiters); index++ {
    waiter := state.Signal_Waiters[index]
    if waiter.Deadline > now {
      kept = append(kept, waiter)
      continue
    }
    expired := waiter
    expired.Completion.Callback = func(_ nbio.Completion_Handle) {
      signal_deliver(state, expired, nbio.SIGNAL_EXPIRED, nbio.Deadline_Exceeded)
    }
    operating_system_completion_add(state, expired.Completion)
  }
  state.Signal_Waiters = kept
}

// One signal watch reaches its caller, delivered or expired, with the same retirement step as
// a process completion.
func signal_deliver(
  state *Operating_System, waiter Signal_Waiter, signal nbio.Signal, err error,
) {
  state.Extension_Submitted--
  waiter.Callback(waiter.Completion, signal, err)
}
