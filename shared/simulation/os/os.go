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
// seed: a simulation states the values it wants and reads exactly those back. Nothing here
// retires a completion, thus this package holds no queue, takes no timeline, and draws no seed.
// A signal watch and a spawn are discovered by the poll pass rather than by a timer, so both
// live on nbio.IO with the descriptors and kernel events they need.
package os

import (
	"errors"
	"unsafe"

	"local/james-orcales/shared/invariant/default"
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
func Virtual_OS_To_OS(virtual *Virtual_OS) (system OS) {
	invariant.Always(virtual != nil, "A virtual OS has caller-owned state.")
	Virtual_OS_Invariants(*virtual, "virtual_os_to_os.virtual")
	system = OS{
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
	OS_Invariants(system, "virtual_os_to_os.system")
	return system
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
