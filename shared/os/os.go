// Package os is a dependency-injected operating system, built the same way as shared/time: the
// backend is a struct of closures (a vtable), production wires the host kernel (os/default), and
// a simulation wires a Virtual_OS. The code between never knows which it holds.
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

	invariant "local/james-orcales/shared/invariant/default"
)

// OS is the operating system as the running process sees it. It is a vtable, so a caller holds
// it by value and calls through it without knowing the backend, exactly as io.IO is held.
type OS struct {
	// Arguments returns the process argv. The first element names the program as it was
	// invoked, which is not always the executable path.
	Arguments func() (arguments []string)
	// Environment returns every variable as one "NAME=VALUE" string.
	Environment func() (variables []string)
	// Variable reads one environment variable. found is false when the name is unset, which
	// a caller must tell apart from a name set to the empty string.
	Variable func(name string) (value string, found bool)
	// Executable returns the path of the running image.
	Executable func() (path string, err error)
	// Working_Directory returns the directory that resolves the process's relative paths.
	Working_Directory func() (path string, err error)
	// Hostname returns the name the kernel gives this machine.
	Hostname func() (name string, err error)
	// Identifier returns the process id.
	Identifier func() (identifier int)
	// Effective_User_Identifier returns the user id the kernel checks permission against,
	// which a setuid image makes different from the user who started the process.
	Effective_User_Identifier func() (identifier int)
	// Self_Exec replaces the process image and returns only on failure. Every descriptor is
	// close-on-exec, so a successful replacement closes listeners and the new image rebinds.
	// An empty environment inherits nothing.
	Self_Exec func(path string, arguments []string, environment []string) (err error)
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
	invariant.Always(system.Identifier != nil, "An OS reads its process id.")
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
	// Identifier becomes the simulated process id.
	Identifier int
	// Effective_User_Identifier becomes the simulated effective user id. Zero is root, so a
	// simulation states it deliberately and the field carries no positive bound.
	Effective_User_Identifier int
}

// Virtual_OS_Invariants states the complete simulated domain. A process id is positive on every
// kernel this repository targets, and pid 1 is init, so a simulation that states zero has left
// the field unset rather than described a real process.
func Virtual_OS_Invariants(virtual Virtual_OS, namespace invariant.Namespace) {
	invariant.Always(virtual.Identifier > 0, "A Virtual_OS has a positive process id.")
}

// Virtual_OS_To_OS turns simulated ambient state into the vtable every caller holds, the
// counterpart of time.Virtual_Clock_To_Clock. Each reader copies before it answers, so a caller
// that keeps or edits a returned slice cannot change what the next read sees.
func Virtual_OS_To_OS(virtual Virtual_OS) (system OS) {
	defer func() { OS_Invariants(system, "virtual_os_to_os.system") }()
	Virtual_OS_Invariants(virtual, "virtual_os_to_os.virtual")
	system = OS{
		Arguments: func() (arguments []string) {
			return copy_strings(virtual.Arguments)
		},
		Environment: func() (variables []string) {
			return copy_strings(virtual.Environment)
		},
		Variable: func(name string) (value string, found bool) {
			return Environment_Lookup(virtual.Environment, name)
		},
		Executable: func() (path string, err error) {
			return virtual.Executable, nil
		},
		Working_Directory: func() (path string, err error) {
			return virtual.Working_Directory, nil
		},
		Hostname: func() (name string, err error) {
			return virtual.Hostname, nil
		},
		Identifier: func() (identifier int) {
			return virtual.Identifier
		},
		Effective_User_Identifier: func() (identifier int) {
			return virtual.Effective_User_Identifier
		},
		Self_Exec: func(_ string, _ []string, _ []string) (err error) {
			return Self_Exec_Unsupported
		},
	}
	return system
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

// Returns a copy, so a caller that edits the result cannot reach the backend's own slice.
func copy_strings(source []string) (copied []string) {
	copied = make([]string, len(source))
	copy(copied, source)
	return copied
}
