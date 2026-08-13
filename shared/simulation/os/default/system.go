// Package os is the composition tier: the operating system read from the host kernel. It
// declares package os so callers import ".../os/default" and read it as the library with no
// alias.
//
// Every reader here is one syscall. Two are not: argv and the executable path are values the
// kernel handed the process at exec, and no syscall reads them back, so they come from what the
// Go runtime captured at startup.
package os

import (
	startup "os"
	"syscall"

	"local/james-orcales/shared/simulation/os"
)

// New_Operating_System returns an OS backed by the host kernel. Every reading stays inside a
// closure and crosses no function of this package, because a value the machine supplies carries
// no domain a bundle could state.
// It fills the ambient readers alone. The signal watch and the spawn retire a completion on a
// loop's queue, so the backend that owns that queue completes this OS — see
// nbio/default.New_Operating_System_IO, which takes what this returns and hands back the whole
// vtable.
func New_Operating_System() (host os.OS) {
	host = os.OS{
		// The runtime captured argv at startup and os.Args is that slice itself, so the
		// copy stops a caller from editing what every later reader sees.
		Arguments:   func() (arguments []string) { return system_arguments() },
		Environment: func() (variables []string) { return syscall.Environ() },
		Variable: func(name string) (value string, found bool) {
			return syscall.Getenv(name)
		},
		Executable:        startup.Executable,
		Working_Directory: syscall.Getwd,
		Hostname:          system_hostname,
		Identifier:        syscall.Getpid,

		Effective_User_Identifier: syscall.Geteuid,
		Self_Exec: func(path string, arguments []string, environment []string) (err error) {
			return syscall.Exec(path, arguments, environment)
		},
	}
	return host
}

// Returns a copy of the runtime's argv, so a caller that edits the result cannot reach os.Args.
func system_arguments() (arguments []string) {
	arguments = make([]string, len(startup.Args))
	copy(arguments, startup.Args)
	return arguments
}
