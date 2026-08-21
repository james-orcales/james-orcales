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

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/slices"
)

// PROCESS_IDENTIFIER_MINIMUM is first kernel-valid process identity.
const PROCESS_IDENTIFIER_MINIMUM = slices.COUNT_MINIMUM + 1

// Arguments is complete process argument sequence.
type Arguments []string

// Arguments_Invariants bounds caller and backend work.
func Arguments_Invariants(value Arguments, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Environment is complete process environment sequence.
type Environment []string

// Environment_Invariants bounds caller and backend work.
func Environment_Invariants(value Environment, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Entry_Count is populated prefix size in caller-owned storage.
type Entry_Count int

// Entry_Count_Invariants bounds populated prefix size.
func Entry_Count_Invariants(value Entry_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Variable_Name is environment key text.
type Variable_Name string

// Variable_Name_Invariants bounds lookup work.
func Variable_Name_Invariants(value Variable_Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Variable_Value is environment value text.
type Variable_Value string

// Variable_Value_Invariants bounds returned ambient data.
func Variable_Value_Invariants(value Variable_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Variable_Found distinguishes unset name from empty value.
type Variable_Found bool

// Variable_Found_Invariants covers both lookup outcomes.
func Variable_Found_Invariants(value Variable_Found, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(
			bool(value), "Environment variable lookup covers found and absent names.",
		).
		Ensure()
}

// Executable_Path is process image path.
type Executable_Path string

// Executable_Path_Invariants bounds returned and supplied image path text.
func Executable_Path_Invariants(value Executable_Path, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Working_Directory_Path is process working-directory path.
type Working_Directory_Path string

// Working_Directory_Path_Invariants bounds returned directory path text.
func Working_Directory_Path_Invariants(
	value Working_Directory_Path, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Hostname is machine name text.
type Hostname string

// Hostname_Invariants bounds returned machine name.
func Hostname_Invariants(value Hostname, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Process_Identifier is kernel process identity.
type Process_Identifier int

// Process_Identifier_Invariants excludes kernel-invalid nonpositive identity.
func Process_Identifier_Invariants(value Process_Identifier, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), PROCESS_IDENTIFIER_MINIMUM, bits.INTEGER_MAXIMUM).
		Ensure()
}

// Effective_User_Identifier is kernel permission identity.
type Effective_User_Identifier int

// Effective_User_Identifier_Invariants admits root and every nonnegative identity.
func Effective_User_Identifier_Invariants(
	value Effective_User_Identifier, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), slices.COUNT_MINIMUM, bits.INTEGER_MAXIMUM).
		Ensure()
}

// OS is the operating system as the running process sees it. It is a vtable, so a caller holds
// it by value and calls through it without knowing the backend, exactly as io.IO is held.
type OS struct {
	// State stays caller-owned because captured backend state would allocate.
	State unsafe.Pointer
	// Arguments copies process argv into destination and returns populated entry count. First
	// element names program as invoked, which is not always executable path.
	Arguments func(state unsafe.Pointer, destination Arguments) (count Entry_Count)
	// Environment copies every variable as one "NAME=VALUE" string and returns populated count.
	Environment func(state unsafe.Pointer, destination Environment) (count Entry_Count)
	// Variable reads one environment variable. found is false when the name is unset, which
	// a caller must tell apart from a name set to the empty string.
	Variable func(
		state unsafe.Pointer, name Variable_Name,
	) (value Variable_Value, found Variable_Found)
	// Executable returns the path of the running image.
	Executable func(state unsafe.Pointer) (path Executable_Path, err error)
	// Working_Directory returns the directory that resolves the process's relative paths.
	Working_Directory func(state unsafe.Pointer) (path Working_Directory_Path, err error)
	// Hostname returns the name the kernel gives this machine.
	Hostname func(state unsafe.Pointer) (name Hostname, err error)
	// Process_Identifier returns the process id.
	Process_Identifier func(state unsafe.Pointer) (identifier Process_Identifier)
	// Effective_User_Identifier returns the user id the kernel checks permission against,
	// which a setuid image makes different from the user who started the process.
	Effective_User_Identifier func(state unsafe.Pointer) (identifier Effective_User_Identifier)
	// Self_Exec replaces the process image and returns only on failure. Every descriptor is
	// close-on-exec, so a successful replacement closes listeners and the new image rebinds.
	// Nil preserves ambient values. Non-nil slice is complete replacement, thus empty inherits
	// nothing.
	Self_Exec func(
		state unsafe.Pointer, path Executable_Path,
		arguments Arguments, environment Environment,
	) (err error)
}

// OS_Invariants states that every reader is bound. An OS is a vtable, so its only property is
// that every slot is filled: the zero OS reads as an OS but panics on first use, and a backend
// that fills eight slots and forgets the ninth is the same failure one call later.
func OS_Invariants(system OS, namespace aver.Namespace) {
	aver.Always(system.Arguments != nil, "An OS reads its arguments.")
	aver.Always(system.Environment != nil, "An OS reads its environment.")
	aver.Always(system.Variable != nil, "An OS reads one environment variable.")
	aver.Always(system.Executable != nil, "An OS reads its executable path.")
	aver.Always(system.Working_Directory != nil, "An OS reads its working directory.")
	aver.Always(system.Hostname != nil, "An OS reads its host name.")
	aver.Always(system.Process_Identifier != nil, "An OS reads its process id.")
	aver.Always(
		system.Effective_User_Identifier != nil, "An OS reads its effective user id.")
	aver.Always(system.Self_Exec != nil, "An OS replaces its own image.")
}

// OS_Arguments keeps backend state explicit so reader needs no captured environment.
func OS_Arguments(system OS, destination Arguments) (count Entry_Count) {
	defer func() { Entry_Count_Invariants(count, "os_arguments.count") }()
	OS_Invariants(system, "os_arguments.system")
	Arguments_Invariants(destination, "os_arguments.destination")
	return system.Arguments(system.State, destination)
}

// OS_Environment keeps backend state explicit so reader needs no captured environment.
func OS_Environment(system OS, destination Environment) (count Entry_Count) {
	defer func() { Entry_Count_Invariants(count, "os_environment.count") }()
	OS_Invariants(system, "os_environment.system")
	Environment_Invariants(destination, "os_environment.destination")
	return system.Environment(system.State, destination)
}

// OS_Variable keeps backend state explicit so reader needs no captured environment.
func OS_Variable(
	system OS, name Variable_Name,
) (value Variable_Value, found Variable_Found) {
	defer func() {
		Variable_Value_Invariants(value, "os_variable.value")
		Variable_Found_Invariants(found, "os_variable.found")
	}()
	OS_Invariants(system, "os_variable.system")
	Variable_Name_Invariants(name, "os_variable.name")
	return system.Variable(system.State, name)
}

// OS_Executable keeps backend state explicit so reader needs no captured environment.
func OS_Executable(system OS) (path Executable_Path, err error) {
	defer func() { Executable_Path_Invariants(path, "os_executable.path") }()
	OS_Invariants(system, "os_executable.system")
	return system.Executable(system.State)
}

// OS_Working_Directory keeps backend state explicit so reader needs no captured environment.
func OS_Working_Directory(system OS) (path Working_Directory_Path, err error) {
	defer func() { Working_Directory_Path_Invariants(path, "os_working_directory.path") }()
	OS_Invariants(system, "os_working_directory.system")
	return system.Working_Directory(system.State)
}

// OS_Hostname keeps backend state explicit so reader needs no captured environment.
func OS_Hostname(system OS) (name Hostname, err error) {
	defer func() { Hostname_Invariants(name, "os_hostname.name") }()
	OS_Invariants(system, "os_hostname.system")
	return system.Hostname(system.State)
}

// OS_Process_Identifier keeps backend state explicit so reader needs no captured environment.
func OS_Process_Identifier(system OS) (identifier Process_Identifier) {
	defer func() {
		Process_Identifier_Invariants(identifier, "os_process_identifier.identifier")
	}()
	OS_Invariants(system, "os_process_identifier.system")
	return system.Process_Identifier(system.State)
}

// OS_Effective_User_Identifier keeps state explicit so reader needs no captured environment.
func OS_Effective_User_Identifier(system OS) (identifier Effective_User_Identifier) {
	defer func() {
		Effective_User_Identifier_Invariants(
			identifier, "os_effective_user_identifier.identifier",
		)
	}()
	OS_Invariants(system, "os_effective_user_identifier.system")
	return system.Effective_User_Identifier(system.State)
}

// OS_Self_Exec keeps backend state explicit so operation needs no captured environment.
func OS_Self_Exec(
	system OS, path Executable_Path, arguments Arguments, environment Environment,
) (err error) {
	OS_Invariants(system, "os_self_exec.system")
	Executable_Path_Invariants(path, "os_self_exec.path")
	Arguments_Invariants(arguments, "os_self_exec.arguments")
	Environment_Invariants(environment, "os_self_exec.environment")
	return system.Self_Exec(system.State, path, arguments, environment)
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
	Arguments Arguments
	// Environment becomes the simulated environment, each entry one "NAME=VALUE" string.
	Environment Environment
	// Executable becomes the simulated path of the running image.
	Executable Executable_Path
	// Working_Directory becomes the simulated directory resolving relative paths.
	Working_Directory Working_Directory_Path
	// Hostname becomes the simulated machine name.
	Hostname Hostname
	// Process_Identifier becomes the simulated process id.
	Process_Identifier Process_Identifier
	// Effective_User_Identifier becomes the simulated effective user id. Zero is root, so a
	// simulation states it deliberately and the field carries no positive bound.
	Effective_User_Identifier Effective_User_Identifier
}

// Virtual_OS_Invariants states the complete simulated domain. A process id is positive on every
// kernel this repository targets, and pid 1 is init, so a simulation that states zero has left
// the field unset rather than described a real process.
func Virtual_OS_Invariants(virtual *Virtual_OS, namespace aver.Namespace) {
	aver.Always(virtual != nil, "A virtual OS has caller-owned state.")
	Arguments_Invariants(virtual.Arguments, namespace)
	Environment_Invariants(virtual.Environment, namespace)
	Executable_Path_Invariants(virtual.Executable, namespace)
	Working_Directory_Path_Invariants(virtual.Working_Directory, namespace)
	Hostname_Invariants(virtual.Hostname, namespace)
	Process_Identifier_Invariants(virtual.Process_Identifier, namespace)
	Effective_User_Identifier_Invariants(virtual.Effective_User_Identifier, namespace)
}

// Virtual_OS_To_OS turns simulated ambient state into the vtable every caller holds, the
// counterpart of time.Virtual_Clock_To_Clock. Each slice reader copies into caller destination,
// so caller edit cannot change what next read sees and backend owns no result allocation.
func Virtual_OS_To_OS(virtual *Virtual_OS) (system OS) {
	defer func() { OS_Invariants(system, "virtual_os_to_os.system") }()
	Virtual_OS_Invariants(virtual, "virtual_os_to_os.virtual")
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
	return system
}

func virtual_os_arguments(
	state unsafe.Pointer, destination Arguments,
) (count Entry_Count) {
	defer func() { Entry_Count_Invariants(count, "virtual_os_arguments.count") }()
	Arguments_Invariants(destination, "virtual_os_arguments.destination")
	return copy_strings(destination, (*Virtual_OS)(state).Arguments)
}

func virtual_os_environment(
	state unsafe.Pointer, destination Environment,
) (count Entry_Count) {
	defer func() { Entry_Count_Invariants(count, "virtual_os_environment.count") }()
	Environment_Invariants(destination, "virtual_os_environment.destination")
	return copy_strings(destination, (*Virtual_OS)(state).Environment)
}

func virtual_os_variable(
	state unsafe.Pointer, name Variable_Name,
) (value Variable_Value, found Variable_Found) {
	defer func() {
		Variable_Value_Invariants(value, "virtual_os_variable.value")
		Variable_Found_Invariants(found, "virtual_os_variable.found")
	}()
	Variable_Name_Invariants(name, "virtual_os_variable.name")
	return Environment_Lookup((*Virtual_OS)(state).Environment, name)
}

func virtual_os_executable(state unsafe.Pointer) (path Executable_Path, err error) {
	defer func() { Executable_Path_Invariants(path, "virtual_os_executable.path") }()
	return (*Virtual_OS)(state).Executable, nil
}

func virtual_os_working_directory(
	state unsafe.Pointer,
) (path Working_Directory_Path, err error) {
	defer func() {
		Working_Directory_Path_Invariants(path, "virtual_os_working_directory.path")
	}()
	return (*Virtual_OS)(state).Working_Directory, nil
}

func virtual_os_hostname(state unsafe.Pointer) (name Hostname, err error) {
	defer func() { Hostname_Invariants(name, "virtual_os_hostname.name") }()
	return (*Virtual_OS)(state).Hostname, nil
}

func virtual_os_process_identifier(state unsafe.Pointer) (identifier Process_Identifier) {
	defer func() {
		Process_Identifier_Invariants(
			identifier, "virtual_os_process_identifier.identifier",
		)
	}()
	return (*Virtual_OS)(state).Process_Identifier
}

func virtual_os_effective_user_identifier(
	state unsafe.Pointer,
) (identifier Effective_User_Identifier) {
	defer func() {
		Effective_User_Identifier_Invariants(
			identifier, "virtual_os_effective_user_identifier.identifier",
		)
	}()
	return (*Virtual_OS)(state).Effective_User_Identifier
}

func virtual_os_self_exec(
	_ unsafe.Pointer, _ Executable_Path, _ Arguments, _ Environment,
) (err error) {
	return Self_Exec_Unsupported
}

// Environment_Lookup finds name in an environment holding "NAME=VALUE" entries. It is exported
// because both backends need it: a kernel answers Environment as a flat slice, and one lookup
// over that slice is the same work whichever backend produced it. A later duplicate wins, which
// is what execve leaves behind when a name is passed twice.
func Environment_Lookup(
	variables Environment, name Variable_Name,
) (value Variable_Value, found Variable_Found) {
	defer func() {
		Variable_Value_Invariants(value, "environment_lookup.value")
		Variable_Found_Invariants(found, "environment_lookup.found")
	}()
	Environment_Invariants(variables, "environment_lookup.variables")
	Variable_Name_Invariants(name, "environment_lookup.name")
	for _, variable := range variables {
		for index := 0; index < len(variable); index++ {
			if variable[index] != '=' {
				continue
			}
			if variable[:index] == string(name) {
				value = Variable_Value(variable[index+1:])
				found = true
			}
			break
		}
	}
	return value, found
}

// Caller destination prevents result ownership from allocating or reaching backend slice.
func copy_strings[Strings ~[]string](destination Strings, source Strings) (count Entry_Count) {
	defer func() { Entry_Count_Invariants(count, "copy_strings.count") }()
	aver.Always(len(destination) >= len(source),
		"Caller-owned string storage holds complete OS answer.")
	return Entry_Count(copy(destination, source))
}
