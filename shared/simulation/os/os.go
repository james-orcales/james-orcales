// Package os is dependency-injected operating system built same way as shared/time: backend
// vtable holds static procedures over explicit caller-owned state. Production wires host kernel
// while simulation wires Virtual_OS. Code between never knows which it holds.
//
// What the OS answers here is ambient state: what the kernel handed the process at exec and what
// it says about the process now — argv, the environment, the executable path, the working
// directory, the host name, the process id, and the effective user id. None of it is IO: nothing
// here opens a file, a socket, or a pipe, and nothing here can block, so none of it belongs on
// the io.IO event loop. Self_Exec sits here for the same reason. It replaces the process image
// rather than transferring bytes.
//
// The Virtual backend lives in this package because ambient state is plain data. It needs no
// seed: a simulation states the values it wants and reads exactly those back.
package os

import (
	"errors"
	"unsafe"

	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/random/prng"
	"local/james-orcales/shared/simulation/nbio"
	"local/james-orcales/shared/simulation/time"
	"local/james-orcales/shared/slices"
)

// OS is the operating system as the running process sees it. It is a vtable, so a caller holds
// it by value and calls through it without knowing the backend, exactly as io.IO is held.
type OS struct {
	// State stays caller-owned because captured backend state would allocate.
	State unsafe.Pointer
	// Arguments copies process argv into destination and returns populated entry count. First
	// element names program as invoked, which is not always executable path.
	Arguments func(state unsafe.Pointer, destination []string) (count int)
	// Environment copies every variable as one "NAME=VALUE" string and returns populated count.
	Environment func(state unsafe.Pointer, destination []string) (count int)
	// Variable reads one environment variable. found is false when the name is unset, which
	// a caller must tell apart from a name set to the empty string.
	Variable func(state unsafe.Pointer, name string) (value string, found bool)
	// Executable returns the path of the running image.
	Executable func(state unsafe.Pointer) (path string, err error)
	// Working_Directory returns the directory that resolves the process's relative paths.
	Working_Directory func(state unsafe.Pointer) (path string, err error)
	// Hostname returns the name the kernel gives this machine.
	Hostname func(state unsafe.Pointer) (name string, err error)
	// Process_Identifier returns the process id.
	Process_Identifier func(state unsafe.Pointer) (identifier int)
	// Effective_User_Identifier returns the user id the kernel checks permission against,
	// which a setuid image makes different from the user who started the process.
	Effective_User_Identifier func(state unsafe.Pointer) (identifier int)
	// Self_Exec replaces the process image and returns only on failure. Every descriptor is
	// close-on-exec, so a successful replacement closes listeners and the new image rebinds.
	// Nil preserves ambient values. Non-nil slice is complete replacement, thus empty inherits
	// nothing.
	Self_Exec func(
		state unsafe.Pointer, path string, arguments []string, environment []string,
	) (err error)
	// Watch_Signal fires callback when the process receives signal before the finite
	// deadline, or with time.Deadline_Exceeded. The lifetime is finite deliberately: a
	// permanent waiter is a process that cannot state when it is done.
	Watch_Signal func(
		state unsafe.Pointer, completion *time.Completion, signal Signal,
		deadline time.Duration,
		callback Signal_Callback,
	)
	// Spawn runs request until it finishes or the deadline expires. Expiry kills the
	// subprocess group and returns time.Deadline_Exceeded with any partial result. The
	// simulated backend draws the exit code from its seed and returns no output, since
	// scripted output is disallowed.
	Spawn func(
		state unsafe.Pointer, completion *time.Completion, request Process_Request,
		deadline time.Duration,
		callback Process_Callback,
	)
}

// OS_Arguments keeps backend state explicit so reader needs no captured environment.
func OS_Arguments(system OS, destination []string) (count int) {
	return system.Arguments(system.State, destination)
}

// OS_Environment keeps backend state explicit so reader needs no captured environment.
func OS_Environment(system OS, destination []string) (count int) {
	return system.Environment(system.State, destination)
}

// OS_Variable keeps backend state explicit so reader needs no captured environment.
func OS_Variable(system OS, name string) (value string, found bool) {
	return system.Variable(system.State, name)
}

// OS_Executable keeps backend state explicit so reader needs no captured environment.
func OS_Executable(system OS) (path string, err error) {
	return system.Executable(system.State)
}

// OS_Working_Directory keeps backend state explicit so reader needs no captured environment.
func OS_Working_Directory(system OS) (path string, err error) {
	return system.Working_Directory(system.State)
}

// OS_Hostname keeps backend state explicit so reader needs no captured environment.
func OS_Hostname(system OS) (name string, err error) {
	return system.Hostname(system.State)
}

// OS_Process_Identifier keeps backend state explicit so reader needs no captured environment.
func OS_Process_Identifier(system OS) (identifier int) {
	return system.Process_Identifier(system.State)
}

// OS_Effective_User_Identifier keeps state explicit so reader needs no captured environment.
func OS_Effective_User_Identifier(system OS) (identifier int) {
	return system.Effective_User_Identifier(system.State)
}

// OS_Self_Exec keeps backend state explicit so operation needs no captured environment.
func OS_Self_Exec(
	system OS, path string, arguments []string, environment []string,
) (err error) {
	return system.Self_Exec(system.State, path, arguments, environment)
}

// OS_Watch_Signal keeps backend state explicit so operation needs no captured environment.
func OS_Watch_Signal(
	system OS, completion *time.Completion, signal Signal, deadline time.Duration,
	callback Signal_Callback,
) {
	system.Watch_Signal(system.State, completion, signal, deadline, callback)
}

// OS_Spawn keeps backend state explicit so operation needs no captured environment.
func OS_Spawn(
	system OS, completion *time.Completion, request Process_Request, deadline time.Duration,
	callback Process_Callback,
) {
	system.Spawn(system.State, completion, request, deadline, callback)
}

// OS_Invariants states that every reader is bound. An OS is a vtable, so its only property is
// that every slot is filled: the zero OS reads as an OS but panics on first use, and a backend
// that fills eight slots and forgets the ninth is the same failure one call later.
func OS_Invariants(system OS, namespace invariant.Namespace) {
	invariant.Always(system.Arguments != nil, "An OS reads its arguments.")
	invariant.Always(system.Environment != nil, "An OS reads its environment.")
	invariant.Always(system.Variable != nil, "An OS reads one environment variable.")
	invariant.Always(system.Executable != nil, "An OS reads its executable path.")
	invariant.Always(system.Working_Directory != nil, "An OS reads its working directory.")
	invariant.Always(system.Hostname != nil, "An OS reads its host name.")
	invariant.Always(system.Process_Identifier != nil, "An OS reads its process id.")
	invariant.Always(
		system.Effective_User_Identifier != nil, "An OS reads its effective user id.")
	invariant.Always(system.Self_Exec != nil, "An OS replaces its own image.")
	invariant.Always(system.Watch_Signal != nil, "An OS watches for one signal.")
	invariant.Always(system.Spawn != nil, "An OS spawns a subprocess.")
}

// Self_Exec_Unsupported is what every simulated Self_Exec returns. A simulation cannot replace
// its own test process, so it reports the failure rather than pretending to succeed, which
// would destroy the run. A caller's real-backend success path never returns, so its failure
// branch is exactly what a simulation exercises.
var Self_Exec_Unsupported = errors.New("os: self-exec is not supported by a Virtual_OS")

// Virtual_OS is the ambient state a simulation gives the process, as plain data. It holds no
// closure and no clock, so a run reads back exactly the values it stated.
type Virtual_OS struct {
	// Arguments becomes the simulated argv.
	Arguments []string
	// Environment becomes the simulated environment, each entry one "NAME=VALUE" string.
	Environment []string
	// Executable becomes the simulated path of the running image.
	Executable string
	// Working_Directory becomes the simulated directory resolving relative paths.
	Working_Directory string
	// Hostname becomes the simulated machine name.
	Hostname string
	// Process_Identifier becomes the simulated process id.
	Process_Identifier int
	// Effective_User_Identifier becomes the simulated effective user id. Zero is root, so a
	// simulation states it deliberately and the field carries no positive bound.
	Effective_User_Identifier int
}

// Virtual_OS_Invariants states the complete simulated domain. A process id is positive on every
// kernel this repository targets, and pid 1 is init, so a simulation that states zero has left
// the field unset rather than described a real process.
func Virtual_OS_Invariants(virtual Virtual_OS, namespace invariant.Namespace) {
	invariant.Always(virtual.Process_Identifier > 0, "A Virtual_OS has a positive process id.")
}

// Virtual_OS_To_OS turns simulated ambient state into the vtable every caller holds, the
// counterpart of time.Virtual_Clock_To_Clock. Each slice reader copies into caller destination,
// so caller edit cannot change what next read sees and backend owns no result allocation.
// It fills the ambient readers alone, so the OS it returns is not yet whole: the signal watch
// and the spawn retire a completion, which is a backend's queue and not plain data.
// New_Simulated_OS adds those two, and it is what asserts OS_Invariants.
func Virtual_OS_To_OS(virtual *Virtual_OS) (system OS) {
	invariant.Always(virtual != nil, "A virtual OS has caller-owned state.")
	Virtual_OS_Invariants(*virtual, "virtual_os_to_os.virtual")
	return OS{
		State:                     unsafe.Pointer(virtual),
		Arguments:                 virtual_os_arguments,
		Environment:               virtual_os_environment,
		Variable:                  virtual_os_variable,
		Executable:                virtual_os_executable,
		Working_Directory:         virtual_os_working_directory,
		Hostname:                  virtual_os_hostname,
		Process_Identifier:        virtual_os_process_identifier,
		Effective_User_Identifier: virtual_os_effective_user_identifier,
		Self_Exec:                 virtual_os_self_exec,
	}
}

func virtual_os_arguments(
	state unsafe.Pointer, destination []string,
) (count int) {
	return copy_strings(destination, (*Virtual_OS)(state).Arguments)
}

func virtual_os_environment(
	state unsafe.Pointer, destination []string,
) (count int) {
	return copy_strings(destination, (*Virtual_OS)(state).Environment)
}

func virtual_os_variable(
	state unsafe.Pointer, name string,
) (value string, found bool) {
	return Environment_Lookup((*Virtual_OS)(state).Environment, name)
}

func virtual_os_executable(state unsafe.Pointer) (path string, err error) {
	return (*Virtual_OS)(state).Executable, nil
}

func virtual_os_working_directory(state unsafe.Pointer) (path string, err error) {
	return (*Virtual_OS)(state).Working_Directory, nil
}

func virtual_os_hostname(state unsafe.Pointer) (name string, err error) {
	return (*Virtual_OS)(state).Hostname, nil
}

func virtual_os_process_identifier(state unsafe.Pointer) (identifier int) {
	return (*Virtual_OS)(state).Process_Identifier
}

func virtual_os_effective_user_identifier(state unsafe.Pointer) (identifier int) {
	return (*Virtual_OS)(state).Effective_User_Identifier
}

func virtual_os_self_exec(
	_ unsafe.Pointer, _ string, _ []string, _ []string,
) (err error) {
	return Self_Exec_Unsupported
}

// Environment_Lookup finds name in an environment holding "NAME=VALUE" entries. It is exported
// because both backends need it: a kernel answers Environment as a flat slice, and one lookup
// over that slice is the same work whichever backend produced it. A later duplicate wins, which
// is what execve leaves behind when a name is passed twice.
func Environment_Lookup(variables []string, name string) (value string, found bool) {
	for _, variable := range variables {
		for index := 0; index < len(variable); index++ {
			if variable[index] != '=' {
				continue
			}
			if variable[:index] == name {
				value = variable[index+1:]
				found = true
			}
			break
		}
	}
	return value, found
}

// Caller destination prevents result ownership from allocating or reaching backend slice.
func copy_strings(destination []string, source []string) (count int) {
	invariant.Always(len(destination) >= len(source),
		"Caller-owned string storage holds complete OS answer.")
	return copy(destination, source)
}

// Signal identifies an operating-system signal in backend-independent form, so the
// deterministic and OS backends agree on a value without the pure tier importing syscall.
type Signal int

// SIGNAL_EXPIRED names no delivered signal. Caller retain signal it armed to identify watch.
const SIGNAL_EXPIRED Signal = -1

// SIGNAL_TERMINATE is the graceful-termination request (SIGTERM on the OS backend).
const SIGNAL_TERMINATE Signal = 0

// SIGNAL_INTERRUPT is the interactive interrupt (SIGINT on the OS backend).
const SIGNAL_INTERRUPT Signal = 1

// Signal_Callback receives a delivered signal on the loop thread, or Deadline_Exceeded when
// finite watch retires before a signal arrives.
type Signal_Callback func(completion *time.Completion, signal Signal, err error)

// Process_Request describes a subprocess to run: the executable, its arguments and
// environment, the working directory, and the bytes fed to its standard input.
type Process_Request struct {
	// Path is the executable to run.
	Path string
	// Arguments are the process arguments, excluding the program name.
	Arguments []string
	// Environment is the complete process environment. A nil value gives the child none, so a
	// caller that wants an ambient value must inject it from its root.
	Environment []string
	// Working_Directory is the process's directory; empty uses the current one.
	Working_Directory string
	// Input is the bytes written to the process's standard input.
	Input []byte
	// Stdout, when its Procedure is set, streams the process's standard output to the stream
	// as it runs instead of capturing it into Result.Output — the affordance a long build
	// needs so its progress reaches the user live. A zero Stream keeps the captured-buffer
	// default. The simulated backend produces no output and ignores it.
	//
	// The loop writes to the stream on its own thread, so a Stream that waits on the world
	// stalls every other operation. A Stream moves memory only, which is what makes it the
	// right sink here.
	Stdout nbio.Stream
	// Stderr is the standard-error counterpart, same live-or-capture rule.
	Stderr nbio.Stream
}

// Process_Usage is the resource accounting a finished process reports.
type Process_Usage struct {
	// Wall is the elapsed wall-clock time the process ran.
	Wall time.Duration
	// CPU_User is the user-mode CPU time consumed.
	CPU_User time.Duration
	// CPU_System is the kernel-mode CPU time consumed.
	CPU_System time.Duration
	// RSS_Bytes_Max is the peak resident set size in bytes.
	RSS_Bytes_Max int64
}

// Process_Result is a finished process's outcome: its exit code, captured output, and
// resource usage.
type Process_Result struct {
	// Exit is the process exit code; zero on success.
	Exit int
	// Output is the captured standard output.
	Output []byte
	// Error_Output is the captured standard error.
	Error_Output []byte
	// Usage is the process's resource accounting.
	Usage Process_Usage
}

// Process_Callback receives a finished process's result, or an error when the process
// could not be started at all.
type Process_Callback func(completion *time.Completion, result Process_Result, err error)

// Sim is caller-owned simulated OS state. Shared/time owns retirement order, while Operations
// hold specialized callback results until that retirement.
type Sim struct {
	// Virtual stays beside completion state so one explicit pointer backs vtable.
	Virtual Virtual_OS
	// Timeline is the control plane both operations arm through.
	Timeline time.Timeline
	// Generator draws every latency and every exit code from the seed.
	Generator prng.Generator
	// Operations is bounded caller-owned state for simultaneous signal and process operations.
	Operations []Sim_Operation
}

// Sim_Memory gives simulated OS bounded operation storage without owning an allocation.
type Sim_Memory struct {
	// Operations holds one entry for each simultaneously armed OS operation.
	Operations []Sim_Operation
}

// Sim_Operation is caller-owned storage for one specialized callback and its result.
type Sim_Operation struct {
	// Kind prevents one completion path from decoding other callback type.
	Kind Sim_Operation_Kind
	// Signal preserves seed outcome until timeline reaches retirement grain.
	Signal Signal
	// Result preserves seed outcome until timeline reaches retirement grain.
	Result Process_Result
	// Error preserves deadline result until timeline reaches retirement grain.
	Error error
	// Signal_Callback avoids captured adapter state between arm and retirement.
	Signal_Callback Signal_Callback
	// Process_Callback avoids captured adapter state between arm and retirement.
	Process_Callback Process_Callback
}

// Sim_Operation_Kind prevents static retirement callback from decoding wrong state.
type Sim_Operation_Kind uint8

// SIM_OPERATION_KIND_FREE lets bounded storage expose unused entry without side index.
const SIM_OPERATION_KIND_FREE Sim_Operation_Kind = 0

// SIM_OPERATION_KIND_SIGNAL makes static signal callback reject process state.
const SIM_OPERATION_KIND_SIGNAL Sim_Operation_Kind = 1

// SIM_OPERATION_KIND_PROCESS makes static process callback reject signal state.
const SIM_OPERATION_KIND_PROCESS Sim_Operation_Kind = 2

// The number of virtual grains a simulated operation may take, drawn from the seed so the
// retirement order varies per run while staying reproducible.
const SIM_LATENCY_GRAINS = 8

// One in this many simulated spawns exits non-zero, so a seed sweep exercises both the success
// and the failure path without a scripted outcome.
const SIM_SPAWN_FAIL_GRAINS = 4

// New_Simulated_OS returns simulated operating system from caller-owned state and memory. Signal
// watch and spawn retire on pump, so simulated spawn and read hold one order. Caller owns loop
// and driver, so backend submits and never pumps.
func New_Simulated_OS(
	state *Sim, seed uint64, virtual Virtual_OS, pump time.Timeline, memory Sim_Memory,
) (system OS) {
	invariant.Always(state != nil, "A simulated OS has caller-owned state.")
	invariant.Always(len(memory.Operations) > 0,
		"A simulated OS has operation capacity.")
	invariant.Always(len(memory.Operations) <= slices.SLICE_COUNT_MAXIMUM,
		"Simulated OS operations stay within repository slice boundary.")
	Virtual_OS_Invariants(virtual, "new_simulated_os.virtual")
	for index := range memory.Operations {
		memory.Operations[index] = Sim_Operation{}
	}
	state.Virtual = virtual
	state.Timeline = pump
	state.Generator = prng.New(seed)
	state.Operations = memory.Operations
	system = simulated_os_to_os(state)
	OS_Invariants(system, "new_sim.system")
	return system
}

func simulated_os_to_os(state *Sim) (system OS) {
	return OS{
		State:                     unsafe.Pointer(state),
		Arguments:                 simulated_os_arguments,
		Environment:               simulated_os_environment,
		Variable:                  simulated_os_variable,
		Executable:                simulated_os_executable,
		Working_Directory:         simulated_os_working_directory,
		Hostname:                  simulated_os_hostname,
		Process_Identifier:        simulated_os_process_identifier,
		Effective_User_Identifier: simulated_os_effective_user_identifier,
		Self_Exec:                 virtual_os_self_exec,
		Watch_Signal:              simulated_os_watch_signal,
		Spawn:                     simulated_os_spawn,
	}
}

func simulated_os_arguments(
	state unsafe.Pointer, destination []string,
) (count int) {
	return copy_strings(destination, (*Sim)(state).Virtual.Arguments)
}

func simulated_os_environment(
	state unsafe.Pointer, destination []string,
) (count int) {
	return copy_strings(destination, (*Sim)(state).Virtual.Environment)
}

func simulated_os_variable(
	state unsafe.Pointer, name string,
) (value string, found bool) {
	return Environment_Lookup((*Sim)(state).Virtual.Environment, name)
}

func simulated_os_executable(state unsafe.Pointer) (path string, err error) {
	return (*Sim)(state).Virtual.Executable, nil
}

func simulated_os_working_directory(state unsafe.Pointer) (path string, err error) {
	return (*Sim)(state).Virtual.Working_Directory, nil
}

func simulated_os_hostname(state unsafe.Pointer) (name string, err error) {
	return (*Sim)(state).Virtual.Hostname, nil
}

func simulated_os_process_identifier(state unsafe.Pointer) (identifier int) {
	return (*Sim)(state).Virtual.Process_Identifier
}

func simulated_os_effective_user_identifier(state unsafe.Pointer) (identifier int) {
	return (*Sim)(state).Virtual.Effective_User_Identifier
}

func simulated_os_watch_signal(
	state unsafe.Pointer, completion *time.Completion, signal Signal, deadline time.Duration,
	callback Signal_Callback,
) {
	invariant.Always(deadline > 0, "A signal-watch deadline is positive and finite.")
	sim_watch_signal((*Sim)(state), completion, signal, deadline, callback)
}

func simulated_os_spawn(
	state unsafe.Pointer, completion *time.Completion, _ Process_Request,
	deadline time.Duration, callback Process_Callback,
) {
	invariant.Always(deadline > 0, "A spawn deadline is positive and finite.")
	sim_spawn((*Sim)(state), completion, deadline, callback)
}

// Watches for a signal that, in the simulation, arrives at a seed-drawn grain — the operating
// system event modeled as a seed outcome. It fires callback exactly once.
func sim_watch_signal(
	state *Sim, completion *time.Completion, signal Signal, deadline time.Duration,
	callback Signal_Callback,
) {
	operation := sim_operation_acquire(state, SIM_OPERATION_KIND_SIGNAL)
	operation.Signal_Callback = callback
	latency := sim_latency(state)
	if latency >= deadline {
		operation.Signal = SIGNAL_EXPIRED
		operation.Error = time.Deadline_Exceeded
		completion.Backend = unsafe.Pointer(operation)
		time.Timeline_Submit(state.Timeline, completion, deadline, sim_signal_complete)
	} else {
		operation.Signal = signal
		completion.Backend = unsafe.Pointer(operation)
		time.Timeline_Submit(state.Timeline, completion, latency, sim_signal_complete)
	}
}

func sim_signal_complete(completion *time.Completion) {
	operation := (*Sim_Operation)(completion.Backend)
	invariant.Always(operation != nil,
		"A simulated signal completion owns specialized operation state.")
	invariant.Always(operation.Kind == SIM_OPERATION_KIND_SIGNAL,
		"A simulated signal completion owns signal state.")
	callback := operation.Signal_Callback
	signal := operation.Signal
	err := operation.Error
	*operation = Sim_Operation{}
	completion.Backend = nil
	callback(completion, signal, err)
}

// Delivers a subprocess result drawn from the seed: the exit code varies (usually zero,
// occasionally non-zero for fault coverage) with no captured output — scripted output is
// disallowed, so the seed decides success or failure, not a canned payload.
func sim_spawn(
	state *Sim, completion *time.Completion, deadline time.Duration,
	callback Process_Callback,
) {
	operation := sim_operation_acquire(state, SIM_OPERATION_KIND_PROCESS)
	operation.Process_Callback = callback
	exit := 0
	if prng.Generator_Below(&state.Generator, SIM_SPAWN_FAIL_GRAINS) == 0 {
		exit = 1
	}
	latency := sim_latency(state)
	if latency >= deadline {
		operation.Error = time.Deadline_Exceeded
		completion.Backend = unsafe.Pointer(operation)
		time.Timeline_Submit(state.Timeline, completion, deadline, sim_process_complete)
		return
	}
	operation.Result.Exit = exit
	completion.Backend = unsafe.Pointer(operation)
	time.Timeline_Submit(state.Timeline, completion, latency, sim_process_complete)
}

func sim_process_complete(completion *time.Completion) {
	operation := (*Sim_Operation)(completion.Backend)
	invariant.Always(operation != nil,
		"A simulated process completion owns specialized operation state.")
	invariant.Always(operation.Kind == SIM_OPERATION_KIND_PROCESS,
		"A simulated process completion owns process state.")
	callback := operation.Process_Callback
	result := operation.Result
	err := operation.Error
	*operation = Sim_Operation{}
	completion.Backend = nil
	callback(completion, result, err)
}

func sim_operation_acquire(
	state *Sim, kind Sim_Operation_Kind,
) (operation *Sim_Operation) {
	for index := range state.Operations {
		if state.Operations[index].Kind == SIM_OPERATION_KIND_FREE {
			operation = &state.Operations[index]
			break
		}
	}
	invariant.Always(operation != nil,
		"A simulated OS never exceed caller-owned operation capacity.")
	operation.Kind = kind
	return operation
}

// Draws one simulated operation's virtual latency from the seed.
func sim_latency(state *Sim) (latency time.Duration) {
	return time.Duration(prng.Generator_Below(&state.Generator, SIM_LATENCY_GRAINS))
}
