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

	"local/james-orcales/shared/os"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/slices"
)

// SELF_EXEC_TEXT_TERMINATOR_BYTES is one trailing NUL.
const SELF_EXEC_TEXT_TERMINATOR_BYTES = 1

// SELF_EXEC_TEXT_BYTES_MAXIMUM follows bounded OS text size.
const SELF_EXEC_TEXT_BYTES_MAXIMUM = slices.COUNT_MAXIMUM

// SELF_EXEC_PATH_STORAGE_BYTES includes maximum text and trailing NUL.
const SELF_EXEC_PATH_STORAGE_BYTES = SELF_EXEC_TEXT_BYTES_MAXIMUM +
	SELF_EXEC_TEXT_TERMINATOR_BYTES

// SELF_EXEC_VECTOR_POINTER_COUNT includes maximum entries and trailing nil.
const SELF_EXEC_VECTOR_POINTER_COUNT = slices.COUNT_MAXIMUM +
	SELF_EXEC_TEXT_TERMINATOR_BYTES

// SELF_EXEC_VECTOR_STORAGE_BYTES lets every entry hold maximum text and NUL.
const SELF_EXEC_VECTOR_STORAGE_BYTES = slices.COUNT_MAXIMUM *
	SELF_EXEC_PATH_STORAGE_BYTES

// Self_Exec_Path_Storage points at caller-owned encoded path storage.
type Self_Exec_Path_Storage *[SELF_EXEC_PATH_STORAGE_BYTES]byte

// Self_Exec_Path_Storage_Invariants fixes path storage capacity.
func Self_Exec_Path_Storage_Invariants(
	value Self_Exec_Path_Storage, _ aver.Namespace,
) {
	aver.Always(value != nil, "Self-exec path storage exists.")
	aver.Always(
		len(value) == SELF_EXEC_PATH_STORAGE_BYTES,
		"Self-exec path storage holds maximum text and terminator.",
	)
}

// Self_Exec_Argument_Storage points at caller-owned encoded argument storage.
type Self_Exec_Argument_Storage *[SELF_EXEC_VECTOR_STORAGE_BYTES]byte

// Self_Exec_Argument_Storage_Invariants fixes argument storage capacity.
func Self_Exec_Argument_Storage_Invariants(
	value Self_Exec_Argument_Storage, _ aver.Namespace,
) {
	aver.Always(value != nil, "Self-exec argument storage exists.")
	aver.Always(
		len(value) == SELF_EXEC_VECTOR_STORAGE_BYTES,
		"Self-exec argument storage holds maximum vector text.",
	)
}

// Self_Exec_Environment_Storage points at caller-owned encoded environment storage.
type Self_Exec_Environment_Storage *[SELF_EXEC_VECTOR_STORAGE_BYTES]byte

// Self_Exec_Environment_Storage_Invariants fixes environment storage capacity.
func Self_Exec_Environment_Storage_Invariants(
	value Self_Exec_Environment_Storage, _ aver.Namespace,
) {
	aver.Always(value != nil, "Self-exec environment storage exists.")
	aver.Always(
		len(value) == SELF_EXEC_VECTOR_STORAGE_BYTES,
		"Self-exec environment storage holds maximum vector text.",
	)
}

// Self_Exec_Argument_Pointers points at caller-owned argument pointer storage.
type Self_Exec_Argument_Pointers *[SELF_EXEC_VECTOR_POINTER_COUNT]*byte

// Self_Exec_Argument_Pointers_Invariants fixes argument pointer capacity.
func Self_Exec_Argument_Pointers_Invariants(
	value Self_Exec_Argument_Pointers, _ aver.Namespace,
) {
	aver.Always(value != nil, "Self-exec argument pointer storage exists.")
	aver.Always(
		len(value) == SELF_EXEC_VECTOR_POINTER_COUNT,
		"Self-exec argument pointers hold maximum vector and terminator.",
	)
}

// Self_Exec_Environment_Pointers points at caller-owned environment pointer storage.
type Self_Exec_Environment_Pointers *[SELF_EXEC_VECTOR_POINTER_COUNT]*byte

// Self_Exec_Environment_Pointers_Invariants fixes environment pointer capacity.
func Self_Exec_Environment_Pointers_Invariants(
	value Self_Exec_Environment_Pointers, _ aver.Namespace,
) {
	aver.Always(value != nil, "Self-exec environment pointer storage exists.")
	aver.Always(
		len(value) == SELF_EXEC_VECTOR_POINTER_COUNT,
		"Self-exec environment pointers hold maximum vector and terminator.",
	)
}

// Self_Exec_Workspace owns encoded text and pointer vectors passed to kernel.
type Self_Exec_Workspace struct {
	// Path prevents path-to-C-string allocation.
	Path Self_Exec_Path_Storage
	// Arguments prevents argument-vector allocation.
	Arguments Self_Exec_Argument_Storage
	// Environment prevents environment-vector allocation.
	Environment Self_Exec_Environment_Storage
	// Argument_Pointers addresses encoded argument entries.
	Argument_Pointers Self_Exec_Argument_Pointers
	// Environment_Pointers addresses encoded environment entries.
	Environment_Pointers Self_Exec_Environment_Pointers
}

// Self_Exec_Workspace_Invariants requires every caller-owned region.
func Self_Exec_Workspace_Invariants(
	workspace *Self_Exec_Workspace, namespace aver.Namespace,
) {
	aver.Always(workspace != nil, "Self-exec has caller-owned workspace.")
	Self_Exec_Path_Storage_Invariants(workspace.Path, namespace)
	Self_Exec_Argument_Storage_Invariants(workspace.Arguments, namespace)
	Self_Exec_Environment_Storage_Invariants(workspace.Environment, namespace)
	Self_Exec_Argument_Pointers_Invariants(workspace.Argument_Pointers, namespace)
	Self_Exec_Environment_Pointers_Invariants(workspace.Environment_Pointers, namespace)
}

// Host holds caller-owned ambient state unavailable through allocation-free syscalls.
type Host struct {
	// Arguments is caller-captured argv.
	Arguments os.Arguments
	// Environment is caller-captured ambient environment.
	Environment os.Environment
	// Executable is caller-captured process image path.
	Executable os.Executable_Path
	// Working_Directory is caller-captured relative-path root.
	Working_Directory os.Working_Directory_Path
	// Hostname is caller-captured machine name.
	Hostname os.Hostname
	// Self_Exec holds large caller-owned syscall workspace.
	Self_Exec *Self_Exec_Workspace
}

// Host_Invariants keeps ambient state inside shared OS bound.
func Host_Invariants(host *Host, namespace aver.Namespace) {
	aver.Always(host != nil, "Host has caller-owned state.")
	os.Arguments_Invariants(host.Arguments, namespace)
	os.Environment_Invariants(host.Environment, namespace)
	os.Executable_Path_Invariants(host.Executable, namespace)
	os.Working_Directory_Path_Invariants(host.Working_Directory, namespace)
	os.Hostname_Invariants(host.Hostname, namespace)
	Self_Exec_Workspace_Invariants(host.Self_Exec, namespace)
}

func system_self_exec(
	state unsafe.Pointer, path os.Executable_Path,
	arguments os.Arguments, environment os.Environment,
) (err error) {
	os.Executable_Path_Invariants(path, "system_self_exec.path")
	os.Arguments_Invariants(arguments, "system_self_exec.arguments")
	os.Environment_Invariants(environment, "system_self_exec.environment")
	if environment == nil {
		environment = (*Host)(state).Environment
	}
	workspace := (*Host)(state).Self_Exec
	if len(path) > len(workspace.Path)-SELF_EXEC_TEXT_TERMINATOR_BYTES {
		return syscall.E2BIG
	}
	for index := range path {
		if path[index] == 0 {
			return syscall.EINVAL
		}
	}
	copy(workspace.Path[:], path)
	workspace.Path[len(path)] = 0
	encode := func(
		values []string, storage []byte, pointers []*byte,
	) (encode_err error) {
		byte_offset := 0
		for value_index, value := range values {
			value_bytes := len(value) + SELF_EXEC_TEXT_TERMINATOR_BYTES
			if value_bytes > len(storage)-byte_offset {
				return syscall.E2BIG
			}
			for index := range value {
				if value[index] == 0 {
					return syscall.EINVAL
				}
			}
			pointers[value_index] = &storage[byte_offset]
			byte_offset += copy(storage[byte_offset:], value)
			storage[byte_offset] = 0
			byte_offset++
		}
		pointers[len(values)] = nil
		return nil
	}
	if encode_err := encode(
		arguments, workspace.Arguments[:], workspace.Argument_Pointers[:],
	); encode_err != nil {
		return encode_err
	}
	if encode_err := encode(
		environment, workspace.Environment[:],
		workspace.Environment_Pointers[:],
	); encode_err != nil {
		return encode_err
	}
	return system_execve(unsafe.Pointer(workspace))
}

// New_Operating_System returns OS backed by host kernel. Static procedures keep vtable free of
// captured state while values supplied by machine need no separate domain bundle. Every reader
// here answers from ambient state alone, thus this constructor needs no loop and returns a whole
// vtable.
func New_Operating_System(state *Host) (host os.OS) {
	defer func() { os.OS_Invariants(host, "new_operating_system.host") }()
	Host_Invariants(state, "new_operating_system.state")
	host = os.OS{
		State: unsafe.Pointer(state),
		Arguments: func(
			state unsafe.Pointer, destination os.Arguments,
		) (count os.Entry_Count) {
			arguments := (*Host)(state).Arguments
			aver.Always(len(destination) >= len(arguments),
				"Caller-owned string storage holds complete host arguments.")
			return os.Entry_Count(copy(destination, arguments))
		},
		Environment: func(
			state unsafe.Pointer, destination os.Environment,
		) (count os.Entry_Count) {
			variables := (*Host)(state).Environment
			aver.Always(len(destination) >= len(variables),
				"Caller-owned string storage holds complete host environment.")
			return os.Entry_Count(copy(destination, variables))
		},
		Variable: func(
			_ unsafe.Pointer, name os.Variable_Name,
		) (value os.Variable_Value, found os.Variable_Found) {
			text, exists := syscall.Getenv(string(name))
			return os.Variable_Value(text), os.Variable_Found(exists)
		},
		Executable: func(state unsafe.Pointer) (path os.Executable_Path, err error) {
			return (*Host)(state).Executable, nil
		},
		Working_Directory: func(
			state unsafe.Pointer,
		) (path os.Working_Directory_Path, err error) {
			return (*Host)(state).Working_Directory, nil
		},
		Hostname: func(state unsafe.Pointer) (name os.Hostname, err error) {
			return (*Host)(state).Hostname, nil
		},
		Process_Identifier: func(_ unsafe.Pointer) (identifier os.Process_Identifier) {
			return os.Process_Identifier(syscall.Getpid())
		},
		Effective_User_Identifier: func(
			_ unsafe.Pointer,
		) (identifier os.Effective_User_Identifier) {
			return os.Effective_User_Identifier(syscall.Geteuid())
		},
		Self_Exec: system_self_exec,
	}
	return host
}

// OS_Arguments keeps default-package callers on pure OS operation.
func OS_Arguments(host os.OS, destination os.Arguments) (count os.Entry_Count) {
	defer func() { os.Entry_Count_Invariants(count, "default_os_arguments.count") }()
	os.OS_Invariants(host, "default_os_arguments.host")
	os.Arguments_Invariants(destination, "default_os_arguments.destination")
	return os.OS_Arguments(host, destination)
}

// OS_Environment keeps default-package callers on pure OS operation.
func OS_Environment(host os.OS, destination os.Environment) (count os.Entry_Count) {
	defer func() { os.Entry_Count_Invariants(count, "default_os_environment.count") }()
	os.OS_Invariants(host, "default_os_environment.host")
	os.Environment_Invariants(destination, "default_os_environment.destination")
	return os.OS_Environment(host, destination)
}

// OS_Variable keeps default-package callers on pure OS operation.
func OS_Variable(
	host os.OS, name os.Variable_Name,
) (value os.Variable_Value, found os.Variable_Found) {
	defer func() {
		os.Variable_Value_Invariants(value, "default_os_variable.value")
		os.Variable_Found_Invariants(found, "default_os_variable.found")
	}()
	os.OS_Invariants(host, "default_os_variable.host")
	os.Variable_Name_Invariants(name, "default_os_variable.name")
	return os.OS_Variable(host, name)
}

// OS_Executable keeps default-package callers on pure OS operation.
func OS_Executable(host os.OS) (path os.Executable_Path, err error) {
	defer func() { os.Executable_Path_Invariants(path, "default_os_executable.path") }()
	os.OS_Invariants(host, "default_os_executable.host")
	return os.OS_Executable(host)
}

// OS_Working_Directory keeps default-package callers on pure OS operation.
func OS_Working_Directory(host os.OS) (path os.Working_Directory_Path, err error) {
	defer func() {
		os.Working_Directory_Path_Invariants(path, "default_os_working_directory.path")
	}()
	os.OS_Invariants(host, "default_os_working_directory.host")
	return os.OS_Working_Directory(host)
}

// OS_Hostname keeps default-package callers on pure OS operation.
func OS_Hostname(host os.OS) (name os.Hostname, err error) {
	defer func() { os.Hostname_Invariants(name, "default_os_hostname.name") }()
	os.OS_Invariants(host, "default_os_hostname.host")
	return os.OS_Hostname(host)
}

// OS_Process_Identifier keeps default-package callers on pure OS operation.
func OS_Process_Identifier(host os.OS) (identifier os.Process_Identifier) {
	defer func() {
		os.Process_Identifier_Invariants(
			identifier, "default_os_process_identifier.identifier",
		)
	}()
	os.OS_Invariants(host, "default_os_process_identifier.host")
	return os.OS_Process_Identifier(host)
}

// OS_Effective_User_Identifier keeps default-package callers on pure OS operation.
func OS_Effective_User_Identifier(host os.OS) (identifier os.Effective_User_Identifier) {
	defer func() {
		os.Effective_User_Identifier_Invariants(
			identifier, "default_os_effective_user_identifier.identifier",
		)
	}()
	os.OS_Invariants(host, "default_os_effective_user_identifier.host")
	return os.OS_Effective_User_Identifier(host)
}

// OS_Self_Exec keeps default-package callers on pure OS operation.
func OS_Self_Exec(
	host os.OS, path os.Executable_Path, arguments os.Arguments, environment os.Environment,
) (err error) {
	os.OS_Invariants(host, "default_os_self_exec.host")
	os.Executable_Path_Invariants(path, "default_os_self_exec.path")
	os.Arguments_Invariants(arguments, "default_os_self_exec.arguments")
	os.Environment_Invariants(environment, "default_os_self_exec.environment")
	return os.OS_Self_Exec(host, path, arguments, environment)
}
