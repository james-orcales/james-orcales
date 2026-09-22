// Package main is the harness command: the thin composition root over the library tier, and the
// one place allowed to bind the real loop, allocate, and pump. `go run ./harness <directory>`.
package main

import "runtime"

import "local/james-orcales/harness/pkg"
import "local/james-orcales/harness/pkg/requirements"
import "local/james-orcales/shared/filepath"
import "local/james-orcales/shared/go/ast"
import "local/james-orcales/shared/go/token"
import "local/james-orcales/shared/jlog/default"
import shared_os "local/james-orcales/shared/os"
import "local/james-orcales/shared/os/default"
import "local/james-orcales/shared/sim/nbio"
import system_nbio "local/james-orcales/shared/sim/nbio/default"
import "local/james-orcales/shared/sim/time"
import system_time "local/james-orcales/shared/sim/time/default"
import "local/james-orcales/shared/slices"

// STANDARD_OUTPUT is the descriptor diagnostics go to.
const STANDARD_OUTPUT nbio.File = 1

// STANDARD_ERROR is the descriptor a failed step goes to.
const STANDARD_ERROR nbio.File = 2

// CAPACITY bounds every backend store: a run keeps one operation live, one git spawn among them.
const CAPACITY = 8

// PUMP_DURATION outlasts READ_TIMEOUT: equal deadlines race, and a read must retire first.
const PUMP_DURATION = harness.READ_TIMEOUT + time.MINUTE

// Pump_Done reports whether one pump pass is over.
type Pump_Done func() (finished bool)

// LOOP_FAILURE is a fixed line, because rendering the loop error would allocate.
const LOOP_FAILURE = "harness: failed to drive the loop\n"

func main() {
  system := os.New_Operating_System(&os.Host{Self_Exec: main_self_exec_workspace()})
  state := system_nbio.Operating_System{}
  clock := system_time.New_Operating_System_Clock()
  memory := main_loop_memory()
  loop, driver, err := system_nbio.New_Operating_System_IO(&state, memory, clock, CAPACITY, 0)
  // A failed loop skips Deinit: work stays in flight, and Deinit asserts all of it retired.
  if err != nil {
    os.Standard_Error_Write([]byte(LOOP_FAILURE))
    os.OS_Exit(system, shared_os.Exit_Code(harness.EXIT_FAILURE))
  }
  output := jlog.Stream_File{Storage: loop.Storage, File: STANDARD_OUTPUT}
  error_output := jlog.Stream_File{Storage: loop.Storage, File: STANDARD_ERROR}
  run_memory, run_format_state, run_command := harness.New_Memory()
  read_path := make(harness.Read_Path_Slice, 0, filepath.PATH_SIZE_MAXIMUM)
  read_path = append(read_path, harness.READ_PATH_NONE...)
  // The walk stands on one directory at a time, in a store the root owns like every other.
  current := make(harness.Current_Path, 0, filepath.PATH_SIZE_MAXIMUM)
  arguments := make(shared_os.Arguments, slices.COUNT_MAXIMUM)
  arguments = arguments[:os.OS_Arguments(system, arguments)]
  runner := harness.Runner{
    Runner_Fields: harness.Runner_Fields{
      Loop:         harness.Loop{IO: loop},
      Work_Queued:  harness.Runner_Work_Queued,
      Stopped:      harness.Runner_Stopped,
      Output:       jlog.File_To_Stream(&output),
      Error_Output: jlog.File_To_Stream(&error_output),
      Clock:        clock,
      Read:         harness.Read{Path: read_path},
      Arguments:    arguments,
      Memory:       run_memory,
      Syntax:       main_syntax(),
      Format:       run_format_state,
      Command:      run_command,
      Directory:    harness.Directory{Current: current},
      Reports: harness.Reports{
        Report: harness.Report{
          Rule:    harness.RULE_TEXT_REFUSED,
          Tier:    harness.Report_Tier(harness.TIER_TWO),
          Path:    harness.PATH_TEXT_NONE,
          Message: harness.MESSAGE_OVERSIZED,
        },
        Summary: harness.Summary{Elapsed_Seconds: harness.ELAPSED_TEXT_NONE},
      },
      Flags: harness.Run_Flag_Storage{Values: make(harness.Run_Flags, harness.RUN_FLAG_COUNT)},
    },
  }
  // Built once: a closure literal inside the pump would allocate on every pass.
  done := func() (finished bool) { return bool(main_queued(&runner)) }
  stats := runtime.MemStats{}
  runtime.ReadMemStats(&stats)
  tenths := stats.TotalAlloc * 10 >> 20
  print("preallocated_mib=", tenths/10, ".", tenths%10, "\n")
  phase := (*harness.Opening_Phase)(&runner.Progress.Phase)
  harness.Main(harness.Main_Input{Phase: phase, Start: &runner.Start, Clock: clock})
  main_pump(&runner, driver, done)
  os.OS_Exit(system, shared_os.Exit_Code(runner.Status))
}

// The root alternation: run each recorded continuation, then make one pump pass.
func main_pump(runner harness.Runner_Handle, driver nbio.Driver, done Pump_Done) {
  input := harness.Rearm_Input{
    Memory:    runner.Memory,
    Syntax:    runner.Syntax,
    Progress:  &runner.Progress,
    Arguments: runner.Arguments,
    Command:   runner.Command,
    State: harness.Streams{
      Loop:           &runner.Loop,
      Stage:          &runner.Stage,
      Walk:           &runner.Walk,
      Package_Cursor: &runner.Package_Cursor,
      Orphans:        &runner.Orphan_Cursor,
      Findings:       &runner.Findings,
      Flags:          runner.Flags,
      Files:          &runner.Files,
      File_Cursor:    &runner.File_Cursor,
      Read:           &runner.Read,
      Format:         runner.Format,
      Directory:      &runner.Directory,
      Outcome:        &runner.Outcome,
      Report:         &runner.Report,
      Summary:        &runner.Summary,
      Finding:        &runner.Finding,
      Fix_State:      &runner.Fix_State,
      Tally:          &runner.Tally,
      History:        &runner.History,
      Output:         runner.Output,
      Errors:         runner.Error_Output,
    },
  }
  for !runner.Stopped(harness.Pass_Phase(runner.Progress.Phase)) {
    input.Elapsed = harness.Elapsed(time.Clock_Now_Monotonic(runner.Clock) - runner.Start)
    for harness.Rearm(input) {
    }
    // A stopped run counts as queued work, thus this pass returns at once.
    _, drive_err := nbio.Driver_Run_Until(driver, PUMP_DURATION, done)
    if drive_err != nil {
      os.Standard_Error_Write([]byte(LOOP_FAILURE))
      return
    }
  }
}

// Reports whether one drive pass is over, which the run states of the phase it stands in.
func main_queued(runner harness.Runner_Handle) (finished harness.Boolean) {
  defer func() { harness.Boolean_Invariants(finished, "main_queued.finished") }()
  harness.Runner_Handle_Invariants(runner, "main_queued.runner")
  return runner.Work_Queued(harness.Driven_Phase(runner.Progress.Phase))
}

// The host requires exec storage; harness never replaces its image, so the pages stay untouched.
func main_self_exec_workspace() (workspace os.Self_Exec_Workspace_Pointer) {
  defer func() {
    os.Self_Exec_Workspace_Pointer_Invariants(workspace, "main_self_exec.workspace")
  }()
  path := make(os.Self_Exec_Path_Bytes, os.SELF_EXEC_PATH_STORAGE_BYTES)
  arguments := make(os.Self_Exec_Argument_Bytes, os.SELF_EXEC_VECTOR_STORAGE_BYTES)
  environment := make(os.Self_Exec_Environment_Bytes, os.SELF_EXEC_VECTOR_STORAGE_BYTES)
  count := os.SELF_EXEC_VECTOR_POINTER_COUNT
  argument_pointers := make(os.Self_Exec_Argument_Pointer_Vector, count)
  environment_pointers := make(os.Self_Exec_Environment_Pointer_Vector, count)
  workspace = &os.Self_Exec_Workspace{Path: &path, Arguments: &arguments}
  workspace.Environment, workspace.Argument_Pointers = &environment, &argument_pointers
  workspace.Environment_Pointers = &environment_pointers
  return workspace
}

// Sizes every bounded queue the backend retains for one operation at a time.
func main_loop_memory() (memory system_nbio.Operating_System_Memory) {
  memory.Timeouts = make([]*nbio.Completion, CAPACITY)
  memory.Completed = make([]*nbio.Completion, CAPACITY)
  memory.Operations = make([]system_nbio.Operating_System_Operation, CAPACITY)
  path_size := system_nbio.OPERATING_SYSTEM_PATH_BYTES_MAXIMUM
  memory.Operation_File_Paths = make([]byte, CAPACITY*path_size)
  memory.Operation_Socket_Addresses = make([]byte, CAPACITY*system_nbio.SOCKET_ADDRESS_BYTES)
  memory.Operation_Registry = make([]*system_nbio.Operating_System_Operation, CAPACITY)
  memory.Descriptors = make([]system_nbio.Operating_System_Descriptor, CAPACITY)
  memory.Signal_Waiters = make([]system_nbio.Signal_Waiter, CAPACITY)
  memory.Spawns = make([]system_nbio.Spawn, CAPACITY)
  memory.Spawn_Registry = make([]*system_nbio.Spawn, CAPACITY)
  pipe_size := system_nbio.PROCESS_PIPE_BUFFER_COUNT * system_nbio.PROCESS_PIPE_BYTES
  memory.Spawn_Pipe_Buffers = make([]byte, CAPACITY*pipe_size)
  memory.Spawn_Paths = make([]byte, CAPACITY*system_nbio.PROCESS_PATH_BYTES)
  memory.Spawn_Arguments = make([]string, CAPACITY*system_nbio.PROCESS_ARGUMENT_COUNT_MAXIMUM)
  memory.Spawn_Child_Files = make([]uintptr, CAPACITY*system_nbio.PROCESS_CHILD_DESCRIPTOR_COUNT)
  memory.Platform_Operations = make([]*system_nbio.Operating_System_Operation, CAPACITY)
  memory.Platform_Changes = make([]system_nbio.Kernel_Event, system_nbio.POLL_EVENTS_MAX)
  memory.Platform_Events = make([]system_nbio.Kernel_Event, system_nbio.POLL_EVENTS_MAX)
  return memory
}

// A constructor would state the one parse no file ever left, thus the root builds the parse stores.
func main_syntax() (syntax requirements.Syntax) {
  defer func() { requirements.Syntax_Invariants(syntax, "main_syntax.syntax") }()
  tree := ast.Syntax_Tree{}
  tree.Tokens.Values = make(ast.Tokens, ast.TOKEN_COUNT_MAXIMUM)
  tree.Nodes.Values = make(ast.Nodes, ast.NODE_COUNT_MAXIMUM)
  tree.Parents.Values = make(ast.Parents, ast.DEPTH_MAXIMUM)
  tree.Last_Children.Values = make(ast.Last_Children, ast.DEPTH_MAXIMUM)
  tree.Earlier_Children.Values = make(ast.Earlier_Children, ast.DEPTH_MAXIMUM)
  starts := make(token.Line_Starts, token.LINE_COUNT_MAXIMUM)
  return requirements.Syntax{Syntax_Tree: &tree, Lines: &token.Line_Index{Starts: starts}}
}
