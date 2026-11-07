// Package main is the maddox command: the thin composition root over the internal library tier,
// and the one place allowed to bind the real loop, the clock, and the host, and pump.
package main

import (
	"local/james-orcales/maddox/internal"
	"local/james-orcales/shared/std/jlog/default"
	shared_os "local/james-orcales/shared/std/os"
	"local/james-orcales/shared/std/os/default"
	"local/james-orcales/shared/std/sim/nbio"
	system_nbio "local/james-orcales/shared/std/sim/nbio/default"
	"local/james-orcales/shared/std/sim/time"
	system_time "local/james-orcales/shared/std/sim/time/default"
	"local/james-orcales/shared/std/slices"
)

// STANDARD_OUTPUT is the descriptor the report is written to.
const STANDARD_OUTPUT nbio.File = 1

// STANDARD_ERROR is the descriptor progress and diagnostics are written to.
const STANDARD_ERROR nbio.File = 2

// SPAWN_CAPACITY is one: the benchmark runs one command at a time.
const SPAWN_CAPACITY = 1

// OPERATION_CAPACITY covers one child's pipe reads, exit watch, deadline, pipe drain, a progress
// write, and a report write, with slack.
const OPERATION_CAPACITY = 16

// DESCRIPTOR_CAPACITY covers the child's three pipes, both ends each, with slack.
const DESCRIPTOR_CAPACITY = 8

// SCHEDULER_ENTRY_COUNT sizes the kernel queue; it only has to hold OPERATION_CAPACITY.
const SCHEDULER_ENTRY_COUNT = 32

// PUMP_DURATION caps one pump pass. A pass that ends without progress starts the next one.
const PUMP_DURATION = 1 * time.MINUTE

func main() {
	system := os.New_Operating_System(&os.Host{Self_Exec: main_self_exec_workspace()})
	clock := system_time.New_Operating_System_Clock()
	state := system_nbio.Operating_System{}
	loop, driver, loop_err := system_nbio.New_Operating_System_IO(
		&state, main_memory(), clock, SCHEDULER_ENTRY_COUNT, 0)
	if loop_err != nil {
		main_fail(system, loop_err)
	}
	output := jlog.Stream_File{Storage: loop.Storage, File: STANDARD_OUTPUT}
	error_output := jlog.Stream_File{Storage: loop.Storage, File: STANDARD_ERROR}
	environment := make(shared_os.Environment, slices.COUNT_MAXIMUM)
	environment = environment[:os.OS_Environment(system, environment)]
	arguments := make(shared_os.Arguments, slices.COUNT_MAXIMUM)
	arguments = arguments[:os.OS_Arguments(system, arguments)]
	start := maddox.Main(maddox.Main_Input{
		Arguments:   maddox.Arguments(arguments),
		Environment: maddox.Inherited_Environment(environment),
		Output_Is_Terminal: maddox.Output_Terminal_Status(
			main_terminal(os.Descriptor(STANDARD_OUTPUT))),
		Error_Output_Is_Terminal: maddox.Error_Output_Terminal_Status(
			main_terminal(os.Descriptor(STANDARD_ERROR))),
	})
	runner := maddox.Runner{
		IO:           loop,
		Clock:        clock,
		Output:       jlog.File_To_Stream(&output),
		Error_Output: jlog.File_To_Stream(&error_output),
		Machine:      main_machine(os.Machine_Read()),
		Start:        start,
	}
	// Built once: a closure literal inside the pump would allocate on every pass.
	done := func() (finished bool) { return bool(main_pump_done(&runner)) }
	for !maddox.Stopped(&runner) {
		main_pump(system, driver, &runner, done)
	}
	os.OS_Exit(system, shared_os.Exit_Code(runner.Status))
}

// Runs every recorded continuation, then makes one pump pass. A failed drive leaves work in
// flight, so it ends the process without Deinit, which would assert every operation retired.
func main_pump(
	system shared_os.OS, driver nbio.Driver, runner maddox.Runner_Handle, done Pump_Done,
) {
	shared_os.OS_Invariants(&system, "main_pump.system")
	for maddox.Rearm(runner) {
	}
	if maddox.Stopped(runner) {
		return
	}
	_, drive_err := nbio.Driver_Run_Until(driver, PUMP_DURATION, done)
	if drive_err != nil {
		main_fail(system, drive_err)
	}
}

// Pump_Done reports whether one pump pass may end.
type Pump_Done func() (finished bool)

// A pass ends once the run stops or a callback recorded work only Rearm can act on.
func main_pump_done(runner maddox.Runner_Handle) (finished maddox.Queue_State) {
	defer func() { maddox.Queue_State_Invariants(finished, "main_pump_done.finished") }()
	if maddox.Stopped(runner) {
		return true
	}
	return maddox.Work_Queued(runner)
}

// The loop is gone or broken, so the diagnostic bypasses it.
func main_fail(system shared_os.OS, err error) {
	shared_os.OS_Invariants(&system, "main_fail.system")
	os.Standard_Error_Write([]byte("maddox: " + err.Error() + "\n"))
	os.OS_Exit(system, shared_os.Exit_Code(maddox.EXIT_FAILURE))
}

// Reports a descriptor as a terminal status, so color and progress are suppressed when the
// stream is piped or redirected to a file.
func main_terminal(descriptor os.Descriptor) (status maddox.Terminal_Status) {
	os.Descriptor_Invariants(descriptor, "main_terminal.descriptor")
	if os.Descriptor_Terminal(descriptor) {
		return maddox.Terminal_Status(maddox.TERMINAL_STATUS_TERMINAL)
	}
	return maddox.Terminal_Status(maddox.TERMINAL_STATUS_NOT_TERMINAL)
}

// Converts the host description into the report's snapshot.
func main_machine(machine os.Machine) (specs maddox.Machine_Specs) {
	os.Machine_Invariants(&machine, "main_machine.machine")
	return maddox.Machine_Specs{
		CPU_Model: maddox.Processor_Model(machine.Processor_Model),
		CPU_Arch: maddox.Processor_Architecture(
			machine.Processor_Architecture),
		Physical_Cores: maddox.Physical_Core_Count(machine.Physical_Core_Count),
		Logical_Cores:  maddox.Logical_Core_Count(machine.Logical_Core_Count),
		Performance_Cores: maddox.Performance_Core_Count(
			machine.Performance_Core_Count),
		Efficiency_Cores: maddox.Efficiency_Core_Count(
			machine.Efficiency_Core_Count),
		CPU_Frequency_Hz_Max: maddox.Processor_Frequency(machine.Processor_Frequency),
		Cache_L1_Bytes:       maddox.Level_1_Cache_Size(machine.Level_1_Cache_Size),
		Cache_L2_Bytes:       maddox.Level_2_Cache_Size(machine.Level_2_Cache_Size),
		Cache_L3_Bytes:       maddox.Level_3_Cache_Size(machine.Level_3_Cache_Size),
		RAM_Total_Bytes:      maddox.Memory_Size(machine.Physical_Memory_Size),
		Storage_Total_Bytes:  maddox.Storage_Size(machine.Storage_Size),
		Operating_System_Name: maddox.Operating_System_Name(
			machine.Operating_System_Name),
		Operating_System_Version: maddox.Operating_System_Version(
			machine.Operating_System_Version),
		Kernel_Version: maddox.Kernel_Version(machine.Kernel_Version),
	}
}

// The host vtable requires exec storage. Maddox never replaces its image, so the pages stay
// untouched.
func main_self_exec_workspace() (workspace os.Self_Exec_Workspace_Pointer) {
	defer func() {
		os.Self_Exec_Workspace_Pointer_Invariants(workspace,
			"main_self_exec_workspace.workspace")
	}()
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

// Sizes every bounded queue the backend retains for one child, its output, and the writes.
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
