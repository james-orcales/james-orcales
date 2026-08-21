// Package os is the composition tier: the operating system read from the host kernel. It
// declares package os so callers import ".../os/default" and read it as the library with no
// alias.
//
// Every reader here is one syscall. Two are not: argv and the executable path are values the
// kernel handed the process at exec, and no syscall reads them back, so they come from what the
// Go runtime captured at startup.
package os

import (
	"syscall"
	"unsafe"

	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/simulation/os"
)

// New_Operating_System returns OS backed by host kernel. Static procedures keep vtable free of
// captured state while values supplied by machine need no separate domain bundle. Every reader
// here answers from ambient state alone, thus this constructor needs no loop and returns a whole
// vtable.
func New_Operating_System() (host os.OS) {
	host = os.OS{
		Arguments:                 system_arguments_from_state,
		Environment:               system_environment,
		Variable:                  system_variable,
		Executable:                system_executable_from_state,
		Working_Directory:         system_working_directory,
		Hostname:                  system_hostname_from_state,
		Process_Identifier:        system_process_identifier,
		Effective_User_Identifier: system_effective_user_identifier,
		Self_Exec:                 system_self_exec,
	}
	os.OS_Invariants(host, "new_operating_system.host")
	return host
}

// OS_Arguments keeps default-package callers on pure OS operation.
func OS_Arguments(host os.OS, destination []string) (count int) {
	return os.OS_Arguments(host, destination)
}

// OS_Environment keeps default-package callers on pure OS operation.
func OS_Environment(host os.OS, destination []string) (count int) {
	return os.OS_Environment(host, destination)
}

// OS_Variable keeps default-package callers on pure OS operation.
func OS_Variable(host os.OS, name string) (value string, found bool) {
	return os.OS_Variable(host, name)
}

// OS_Executable keeps default-package callers on pure OS operation.
func OS_Executable(host os.OS) (path string, err error) {
	return os.OS_Executable(host)
}

// OS_Working_Directory keeps default-package callers on pure OS operation.
func OS_Working_Directory(host os.OS) (path string, err error) {
	return os.OS_Working_Directory(host)
}

// OS_Hostname keeps default-package callers on pure OS operation.
func OS_Hostname(host os.OS) (name string, err error) {
	return os.OS_Hostname(host)
}

// OS_Process_Identifier keeps default-package callers on pure OS operation.
func OS_Process_Identifier(host os.OS) (identifier int) {
	return os.OS_Process_Identifier(host)
}

// OS_Effective_User_Identifier keeps default-package callers on pure OS operation.
func OS_Effective_User_Identifier(host os.OS) (identifier int) {
	return os.OS_Effective_User_Identifier(host)
}

// OS_Self_Exec keeps default-package callers on pure OS operation.
func OS_Self_Exec(
	host os.OS, path string, arguments []string, environment []string,
) (err error) {
	return os.OS_Self_Exec(host, path, arguments, environment)
}

func system_arguments_from_state(
	_ unsafe.Pointer, destination []string,
) (count int) {
	invariant.Always(len(destination) >= system_argument_count(),
		"Caller-owned string storage holds complete host arguments.")
	return system_arguments(destination)
}

func system_environment(
	_ unsafe.Pointer, destination []string,
) (count int) {
	variables := syscall.Environ()
	invariant.Always(len(destination) >= len(variables),
		"Caller-owned string storage holds complete host environment.")
	return copy(destination, variables)
}

func system_variable(
	_ unsafe.Pointer, name string,
) (value string, found bool) {
	return syscall.Getenv(name)
}

func system_executable_from_state(_ unsafe.Pointer) (path string, err error) {
	return system_executable()
}

func system_working_directory(_ unsafe.Pointer) (path string, err error) {
	return syscall.Getwd()
}

func system_hostname_from_state(_ unsafe.Pointer) (name string, err error) {
	return system_hostname()
}

func system_process_identifier(_ unsafe.Pointer) (identifier int) {
	return syscall.Getpid()
}

func system_effective_user_identifier(_ unsafe.Pointer) (identifier int) {
	return syscall.Geteuid()
}

func system_self_exec(
	_ unsafe.Pointer, path string, arguments []string, environment []string,
) (err error) {
	if environment == nil {
		environment = syscall.Environ()
	}
	return syscall.Exec(path, arguments, environment)
}
