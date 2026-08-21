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

// Self_Exec_Path_Bytes keeps path encoding outside process heap.
type Self_Exec_Path_Bytes []byte

// Self_Exec_Path_Bytes_Invariants fixes kernel path capacity.
func Self_Exec_Path_Bytes_Invariants(
	value Self_Exec_Path_Bytes, _ aver.Namespace,
) {
	aver.Always(
		len(value) == SELF_EXEC_PATH_STORAGE_BYTES,
		"Self-exec path storage holds maximum text and terminator.",
	)
}

// Self_Exec_Path_Storage points at caller-owned encoded path storage.
type Self_Exec_Path_Storage *Self_Exec_Path_Bytes

// Self_Exec_Path_Storage_Invariants fixes path storage capacity.
func Self_Exec_Path_Storage_Invariants(
	value Self_Exec_Path_Storage, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Self_Exec_Path_Bytes_Invariants(*value, namespace)
}

// Self_Exec_Argument_Bytes keeps argument encoding outside process heap.
type Self_Exec_Argument_Bytes []byte

// Self_Exec_Argument_Bytes_Invariants fixes complete argument text capacity.
func Self_Exec_Argument_Bytes_Invariants(
	value Self_Exec_Argument_Bytes, _ aver.Namespace,
) {
	aver.Always(
		len(value) == SELF_EXEC_VECTOR_STORAGE_BYTES,
		"Self-exec argument storage holds maximum vector text.",
	)
}

// Self_Exec_Argument_Storage points at caller-owned encoded argument storage.
type Self_Exec_Argument_Storage *Self_Exec_Argument_Bytes

// Self_Exec_Argument_Storage_Invariants fixes argument storage capacity.
func Self_Exec_Argument_Storage_Invariants(
	value Self_Exec_Argument_Storage, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Self_Exec_Argument_Bytes_Invariants(*value, namespace)
}

// Self_Exec_Environment_Bytes keeps environment encoding outside process heap.
type Self_Exec_Environment_Bytes []byte

// Self_Exec_Environment_Bytes_Invariants fixes complete environment text capacity.
func Self_Exec_Environment_Bytes_Invariants(
	value Self_Exec_Environment_Bytes, _ aver.Namespace,
) {
	aver.Always(
		len(value) == SELF_EXEC_VECTOR_STORAGE_BYTES,
		"Self-exec environment storage holds maximum vector text.",
	)
}

// Self_Exec_Environment_Storage points at caller-owned encoded environment storage.
type Self_Exec_Environment_Storage *Self_Exec_Environment_Bytes

// Self_Exec_Environment_Storage_Invariants fixes environment storage capacity.
func Self_Exec_Environment_Storage_Invariants(
	value Self_Exec_Environment_Storage, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Self_Exec_Environment_Bytes_Invariants(*value, namespace)
}

// Self_Exec_Argument_Pointer_Vector holds kernel argument addresses.
type Self_Exec_Argument_Pointer_Vector []*byte

// Self_Exec_Argument_Pointer_Vector_Invariants fixes argument vector capacity.
func Self_Exec_Argument_Pointer_Vector_Invariants(
	value Self_Exec_Argument_Pointer_Vector, _ aver.Namespace,
) {
	aver.Always(
		len(value) == SELF_EXEC_VECTOR_POINTER_COUNT,
		"Self-exec argument pointers hold maximum vector and terminator.",
	)
}

// Self_Exec_Argument_Pointers points at caller-owned argument pointer storage.
type Self_Exec_Argument_Pointers *Self_Exec_Argument_Pointer_Vector

// Self_Exec_Argument_Pointers_Invariants fixes argument pointer capacity.
func Self_Exec_Argument_Pointers_Invariants(
	value Self_Exec_Argument_Pointers, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Self_Exec_Argument_Pointer_Vector_Invariants(*value, namespace)
}

// Self_Exec_Environment_Pointer_Vector holds kernel environment addresses.
type Self_Exec_Environment_Pointer_Vector []*byte

// Self_Exec_Environment_Pointer_Vector_Invariants fixes environment vector capacity.
func Self_Exec_Environment_Pointer_Vector_Invariants(
	value Self_Exec_Environment_Pointer_Vector, _ aver.Namespace,
) {
	aver.Always(
		len(value) == SELF_EXEC_VECTOR_POINTER_COUNT,
		"Self-exec environment pointers hold maximum vector and terminator.",
	)
}

// Self_Exec_Environment_Pointers points at caller-owned environment pointer storage.
type Self_Exec_Environment_Pointers *Self_Exec_Environment_Pointer_Vector

// Self_Exec_Environment_Pointers_Invariants fixes environment pointer capacity.
func Self_Exec_Environment_Pointers_Invariants(
	value Self_Exec_Environment_Pointers, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Self_Exec_Environment_Pointer_Vector_Invariants(*value, namespace)
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
	workspace Self_Exec_Workspace, namespace aver.Namespace,
) {
	Self_Exec_Path_Storage_Invariants(workspace.Path, namespace)
	Self_Exec_Argument_Storage_Invariants(workspace.Arguments, namespace)
	Self_Exec_Environment_Storage_Invariants(workspace.Environment, namespace)
	Self_Exec_Argument_Pointers_Invariants(workspace.Argument_Pointers, namespace)
	Self_Exec_Environment_Pointers_Invariants(workspace.Environment_Pointers, namespace)
	aver.Always(workspace.Path != nil, "Self-exec path storage exists.")
	aver.Always(workspace.Arguments != nil, "Self-exec argument storage exists.")
	aver.Always(workspace.Environment != nil, "Self-exec environment storage exists.")
	aver.Always(
		workspace.Argument_Pointers != nil,
		"Self-exec argument pointer storage exists.",
	)
	aver.Always(
		workspace.Environment_Pointers != nil,
		"Self-exec environment pointer storage exists.",
	)
}

// Self_Exec_Workspace_Pointer names caller-owned syscall workspace.
type Self_Exec_Workspace_Pointer *Self_Exec_Workspace

// Self_Exec_Workspace_Pointer_Invariants composes workspace only when present.
func Self_Exec_Workspace_Pointer_Invariants(
	value Self_Exec_Workspace_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Self_Exec_Workspace_Invariants(*value, namespace)
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
	Self_Exec Self_Exec_Workspace_Pointer
}

// Host_Invariants keeps ambient state inside shared OS bound.
func Host_Invariants(host Host, namespace aver.Namespace) {
	os.Arguments_Invariants(host.Arguments, namespace)
	os.Environment_Invariants(host.Environment, namespace)
	os.Executable_Path_Invariants(host.Executable, namespace)
	os.Working_Directory_Path_Invariants(host.Working_Directory, namespace)
	os.Hostname_Invariants(host.Hostname, namespace)
	Self_Exec_Workspace_Pointer_Invariants(host.Self_Exec, namespace)
	aver.Always(host.Self_Exec != nil, "Host has caller-owned self-exec workspace.")
}

// Host_Pointer names caller-owned ambient process state.
type Host_Pointer *Host

// Host_Pointer_Invariants composes ambient state only when present.
func Host_Pointer_Invariants(value Host_Pointer, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Host_Invariants(*value, namespace)
}

func system_encode_arguments(
	values os.Arguments, storage Self_Exec_Argument_Storage,
	pointers Self_Exec_Argument_Pointers,
) (encode_err error) {
	os.Arguments_Invariants(values, "system_encode_arguments.values")
	Self_Exec_Argument_Storage_Invariants(storage, "system_encode_arguments.storage")
	Self_Exec_Argument_Pointers_Invariants(pointers, "system_encode_arguments.pointers")
	aver.Always(storage != nil, "Argument encoding storage exists.")
	aver.Always(pointers != nil, "Argument pointer storage exists.")
	byte_offset := 0
	for value_index, value := range values {
		value_bytes := len(value) + SELF_EXEC_TEXT_TERMINATOR_BYTES
		if value_bytes > len(*storage)-byte_offset {
			return syscall.E2BIG
		}
		for index := range value {
			if value[index] == 0 {
				return syscall.EINVAL
			}
		}
		(*pointers)[value_index] = &(*storage)[byte_offset]
		byte_offset += copy((*storage)[byte_offset:], value)
		(*storage)[byte_offset] = 0
		byte_offset++
	}
	(*pointers)[len(values)] = nil
	return nil
}

func system_encode_environment(
	values os.Environment, storage Self_Exec_Environment_Storage,
	pointers Self_Exec_Environment_Pointers,
) (encode_err error) {
	os.Environment_Invariants(values, "system_encode_environment.values")
	Self_Exec_Environment_Storage_Invariants(storage, "system_encode_environment.storage")
	Self_Exec_Environment_Pointers_Invariants(pointers, "system_encode_environment.pointers")
	aver.Always(storage != nil, "Environment encoding storage exists.")
	aver.Always(pointers != nil, "Environment pointer storage exists.")
	byte_offset := 0
	for value_index, value := range values {
		value_bytes := len(value) + SELF_EXEC_TEXT_TERMINATOR_BYTES
		if value_bytes > len(*storage)-byte_offset {
			return syscall.E2BIG
		}
		for index := range value {
			if value[index] == 0 {
				return syscall.EINVAL
			}
		}
		(*pointers)[value_index] = &(*storage)[byte_offset]
		byte_offset += copy((*storage)[byte_offset:], value)
		(*storage)[byte_offset] = 0
		byte_offset++
	}
	(*pointers)[len(values)] = nil
	return nil
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
	if len(path) > len(*workspace.Path)-SELF_EXEC_TEXT_TERMINATOR_BYTES {
		return syscall.E2BIG
	}
	for index := range path {
		if path[index] == 0 {
			return syscall.EINVAL
		}
	}
	copy(*workspace.Path, path)
	(*workspace.Path)[len(path)] = 0
	if encode_err := system_encode_arguments(
		arguments, workspace.Arguments, workspace.Argument_Pointers,
	); encode_err != nil {
		return encode_err
	}
	if encode_err := system_encode_environment(
		environment, workspace.Environment, workspace.Environment_Pointers,
	); encode_err != nil {
		return encode_err
	}
	return system_execve(unsafe.Pointer(workspace))
}

func system_arguments(
	state unsafe.Pointer, destination os.Arguments,
) (count os.Entry_Count) {
	defer func() { os.Entry_Count_Invariants(count, "system_arguments.count") }()
	os.Arguments_Invariants(destination, "system_arguments.destination")
	arguments := (*Host)(state).Arguments
	aver.Always(len(destination) >= len(arguments),
		"Caller-owned string storage holds complete host arguments.")
	return os.Entry_Count(copy(destination, arguments))
}

func system_environment(
	state unsafe.Pointer, destination os.Environment,
) (count os.Entry_Count) {
	defer func() { os.Entry_Count_Invariants(count, "system_environment.count") }()
	os.Environment_Invariants(destination, "system_environment.destination")
	variables := (*Host)(state).Environment
	aver.Always(len(destination) >= len(variables),
		"Caller-owned string storage holds complete host environment.")
	return os.Entry_Count(copy(destination, variables))
}

func system_variable(
	_ unsafe.Pointer, name os.Variable_Name,
) (value os.Variable_Value, found os.Variable_Found) {
	defer func() {
		os.Variable_Value_Invariants(value, "system_variable.value")
		os.Variable_Found_Invariants(found, "system_variable.found")
	}()
	os.Variable_Name_Invariants(name, "system_variable.name")
	text, exists := syscall.Getenv(string(name))
	return os.Variable_Value(text), os.Variable_Found(exists)
}

func system_executable(state unsafe.Pointer) (path os.Executable_Path, err error) {
	defer func() { os.Executable_Path_Invariants(path, "system_executable.path") }()
	return (*Host)(state).Executable, nil
}

func system_working_directory(
	state unsafe.Pointer,
) (path os.Working_Directory_Path, err error) {
	defer func() {
		os.Working_Directory_Path_Invariants(path, "system_working_directory.path")
	}()
	return (*Host)(state).Working_Directory, nil
}

func system_hostname(state unsafe.Pointer) (name os.Hostname, err error) {
	defer func() { os.Hostname_Invariants(name, "system_hostname.name") }()
	return (*Host)(state).Hostname, nil
}

func system_process_identifier(_ unsafe.Pointer) (identifier os.Process_Identifier) {
	defer func() {
		os.Process_Identifier_Invariants(identifier, "system_process_identifier.identifier")
	}()
	return os.Process_Identifier(syscall.Getpid())
}

func system_effective_user_identifier(
	_ unsafe.Pointer,
) (identifier os.Effective_User_Identifier) {
	defer func() {
		os.Effective_User_Identifier_Invariants(
			identifier, "system_effective_user_identifier.identifier",
		)
	}()
	return os.Effective_User_Identifier(syscall.Geteuid())
}

// New_Operating_System returns OS backed by host kernel. Static procedures keep vtable free of
// captured state while values supplied by machine need no separate domain bundle. Every reader
// here answers from ambient state alone, thus this constructor needs no loop and returns a whole
// vtable.
func New_Operating_System(state Host_Pointer) (host os.OS) {
	defer func() { os.OS_Invariants(host, "new_operating_system.host") }()
	Host_Pointer_Invariants(state, "new_operating_system.state")
	aver.Always(state != nil, "Operating system has caller-owned host state.")
	host = os.OS{
		State:                     unsafe.Pointer(state),
		Arguments:                 system_arguments,
		Environment:               system_environment,
		Variable:                  system_variable,
		Executable:                system_executable,
		Working_Directory:         system_working_directory,
		Hostname:                  system_hostname,
		Process_Identifier:        system_process_identifier,
		Effective_User_Identifier: system_effective_user_identifier,
		Self_Exec:                 system_self_exec,
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
