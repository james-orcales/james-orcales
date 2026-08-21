// Package main is the timeout command: the thin composition root over the internal library
// tier, and the one place allowed to bind the real loop and pump it.
package main

import (
	"local/james-orcales/shared/jlog/default"
	shared_os "local/james-orcales/shared/os"
	"local/james-orcales/shared/os/default"
	"local/james-orcales/shared/sim/nbio"
	system_nbio "local/james-orcales/shared/sim/nbio/default"
	"local/james-orcales/shared/sim/time"
	system_time "local/james-orcales/shared/sim/time/default"
	"local/james-orcales/shared/slices"
	"local/james-orcales/timeout/internal"
)

// STANDARD_OUTPUT is the descriptor the child's output is relayed to.
const STANDARD_OUTPUT nbio.File = 1

// STANDARD_ERROR is the descriptor the child's errors and this program's diagnostic go to.
const STANDARD_ERROR nbio.File = 2

// EXIT_FAILURE reports a loop that could not be built or driven: nothing the child did.
const EXIT_FAILURE = timeout.EXIT_START_FAILURE

// SPAWN_CAPACITY is one: the run is one child.
const SPAWN_CAPACITY = 1

// OPERATION_CAPACITY covers the child's pipe reads, exit watch, deadline, pipe drain, relayed
// writes, and the diagnostic, with slack.
const OPERATION_CAPACITY = 16

// DESCRIPTOR_CAPACITY covers the child's three pipes, both ends each, with slack.
const DESCRIPTOR_CAPACITY = 8

// SCHEDULER_ENTRY_COUNT sizes the kernel queue; it only has to hold OPERATION_CAPACITY.
const SCHEDULER_ENTRY_COUNT = 32

// PUMP_DURATION caps one pump pass. The spawn's own deadline bounds the run, so a pass that
// ends without progress simply starts the next one.
const PUMP_DURATION = 1 * time.MINUTE

func main() {
	system := os.New_Operating_System(&os.Host{Self_Exec: main_self_exec_workspace()})
	memory := main_memory()
	state := system_nbio.Operating_System{}
	loop, driver, loop_err := system_nbio.New_Operating_System_IO(
		&state, memory, system_time.New_Operating_System_Clock(), SCHEDULER_ENTRY_COUNT, 0)
	if loop_err != nil {
		main_fail(system, loop_err)
	}
	child_output := jlog.Stream_File{Storage: loop.Storage, File: STANDARD_OUTPUT}
	child_error := jlog.Stream_File{Storage: loop.Storage, File: STANDARD_ERROR}
	own_error := jlog.Stream_File{Storage: loop.Storage, File: STANDARD_ERROR}
	environment := make(shared_os.Environment, slices.COUNT_MAXIMUM)
	environment = environment[:os.OS_Environment(system, environment)]
	arguments := make(shared_os.Arguments, slices.COUNT_MAXIMUM)
	arguments = arguments[:os.OS_Arguments(system, arguments)]
	runner := timeout.Runner{
		IO:           loop,
		Environment:  environment,
		Stdout:       jlog.File_To_Stream(&child_output),
		Stderr:       jlog.File_To_Stream(&child_error),
		Error_Output: jlog.File_To_Stream(&own_error),
		Message:      make([]byte, timeout.MESSAGE_SIZE_MAXIMUM),
	}
	// Built once: a closure literal inside the pump would allocate on every pass.
	done := func() (finished bool) { return main_pump_done(&runner) }
	timeout.Main(&runner, arguments)
	for !timeout.Stopped(&runner) {
		main_pump(system, driver, &runner, done)
	}
	os.OS_Exit(system, shared_os.Exit_Code(runner.Status))
}

// Runs every recorded continuation, then makes one pump pass. A failed drive leaves work in
// flight, so it ends the process without Deinit, which would assert every operation retired.
func main_pump(
	system shared_os.OS,
	driver nbio.Driver,
	runner *timeout.Runner,
	done func() (finished bool),
) {
	for timeout.Rearm(runner) {
	}
	if timeout.Stopped(runner) {
		return
	}
	_, drive_err := nbio.Driver_Run_Until(driver, PUMP_DURATION, done)
	if drive_err != nil {
		main_fail(system, drive_err)
	}
}

// The loop is gone or broken, so the diagnostic bypasses it. Only a failure to build or drive
// the loop reaches here, never a step of the run, so the allocation stays off the run's path.
func main_fail(system shared_os.OS, err error) {
	os.Standard_Error_Write([]byte(timeout.MESSAGE_PREFIX + err.Error() + "\n"))
	os.OS_Exit(system, shared_os.Exit_Code(EXIT_FAILURE))
}

// The host vtable requires exec storage. Timeout never replaces its image, so the pages stay
// untouched.
func main_self_exec_workspace() (workspace os.Self_Exec_Workspace_Pointer) {
	path := make(os.Self_Exec_Path_Bytes, os.SELF_EXEC_PATH_STORAGE_BYTES)
	arguments := make(os.Self_Exec_Argument_Bytes, os.SELF_EXEC_VECTOR_STORAGE_BYTES)
	environment := make(os.Self_Exec_Environment_Bytes, os.SELF_EXEC_VECTOR_STORAGE_BYTES)
	argument_pointers := make(
		os.Self_Exec_Argument_Pointer_Vector, os.SELF_EXEC_VECTOR_POINTER_COUNT)
	environment_pointers := make(
		os.Self_Exec_Environment_Pointer_Vector, os.SELF_EXEC_VECTOR_POINTER_COUNT)
	return &os.Self_Exec_Workspace{
		Path: &path, Arguments: &arguments, Environment: &environment,
		Argument_Pointers: &argument_pointers, Environment_Pointers: &environment_pointers,
	}
}

// A pass ends once the run stops or a callback recorded work only Rearm can act on.
func main_pump_done(runner *timeout.Runner) (finished bool) {
	if timeout.Stopped(runner) {
		return true
	}
	return timeout.Work_Queued(runner)
}

// Sizes every bounded queue the backend retains for one child and its relayed output.
func main_memory() (memory system_nbio.Operating_System_Memory) {
	return system_nbio.Operating_System_Memory{
		Timeouts:   make([]*nbio.Completion, OPERATION_CAPACITY),
		Completed:  make([]*nbio.Completion, OPERATION_CAPACITY),
		Operations: make([]system_nbio.Operating_System_Operation, OPERATION_CAPACITY),
		Operation_File_Paths: make(
			[]byte, OPERATION_CAPACITY*system_nbio.OPERATING_SYSTEM_PATH_BYTES_MAXIMUM),
		Operation_Socket_Addresses: make(
			[]byte, OPERATION_CAPACITY*system_nbio.SOCKET_ADDRESS_BYTES),
		Operation_Registry: make(
			[]*system_nbio.Operating_System_Operation, OPERATION_CAPACITY),
		Descriptors: make(
			[]system_nbio.Operating_System_Descriptor, DESCRIPTOR_CAPACITY),
		Signal_Waiters: make([]system_nbio.Signal_Waiter, SPAWN_CAPACITY),
		Spawns:         make([]system_nbio.Spawn, SPAWN_CAPACITY),
		Spawn_Registry: make([]*system_nbio.Spawn, SPAWN_CAPACITY),
		Spawn_Pipe_Buffers: make([]byte, SPAWN_CAPACITY*
			system_nbio.PROCESS_PIPE_BUFFER_COUNT*system_nbio.PROCESS_PIPE_BYTES),
		Spawn_Paths: make([]byte, SPAWN_CAPACITY*system_nbio.PROCESS_PATH_BYTES),
		Spawn_Arguments: make(
			[]string, SPAWN_CAPACITY*system_nbio.PROCESS_ARGUMENT_COUNT_MAXIMUM),
		Spawn_Child_Files: make(
			[]uintptr, SPAWN_CAPACITY*system_nbio.PROCESS_CHILD_DESCRIPTOR_COUNT),
		Platform_Operations: make(
			[]*system_nbio.Operating_System_Operation, OPERATION_CAPACITY),
		Platform_Changes: make([]system_nbio.Kernel_Event, system_nbio.POLL_EVENTS_MAX),
		Platform_Events:  make([]system_nbio.Kernel_Event, system_nbio.POLL_EVENTS_MAX),
	}
}
