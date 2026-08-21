// Package os is the composition tier: the operating system read from the host kernel. It
// declares package os so callers import ".../os/default" and read it as the library with no
// alias.
//
// Every reader here is one syscall. Two are not: argv and the executable path are values the
// kernel handed the process at exec, and no syscall reads them back, so they come from what the
// Go runtime captured at startup.
package os

import (
	"errors"
	process "os"
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
		environment = syscall.Environ()
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

// Argv is what the runtime captured at exec, so reading it needs no syscall and no copy.
func system_arguments(
	_ unsafe.Pointer, destination os.Arguments,
) (count os.Entry_Count) {
	defer func() { os.Entry_Count_Invariants(count, "system_arguments.count") }()
	os.Arguments_Invariants(destination, "system_arguments.destination")
	arguments := process.Args
	aver.Always(len(destination) >= len(arguments),
		"Caller-owned string storage holds complete host arguments.")
	return os.Entry_Count(copy(destination, arguments))
}

// The runtime exposes the live environment only as a fresh copy, so this reader allocates.
func system_environment(
	_ unsafe.Pointer, destination os.Environment,
) (count os.Entry_Count) {
	defer func() { os.Entry_Count_Invariants(count, "system_environment.count") }()
	os.Environment_Invariants(destination, "system_environment.destination")
	variables := syscall.Environ()
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
		Exit:                      system_exit,
		Memory_Map:                system_memory_map,
		Memory_Unmap:              system_memory_unmap,
	}
	return host
}

func system_exit(_ unsafe.Pointer, code os.Exit_Code) {
	os.Exit_Code_Invariants(code, "system_exit.code")
	process.Exit(int(code))
}

// MEMORY_DESCRIPTOR_ANONYMOUS is fd -1, which portable anonymous map requires.
const MEMORY_DESCRIPTOR_ANONYMOUS = ^uintptr(0)

// Raw syscall rather than syscall.Mmap, because syscall.Mmap records every region in heap map.
func system_memory_map(
	_ unsafe.Pointer, size os.Memory_Size,
) (region os.Memory, err error) {
	defer func() { os.Memory_Invariants(region, "system_memory_map.region") }()
	os.Memory_Size_Invariants(size, "system_memory_map.size")
	address, _, operation_errno := syscall.RawSyscall6(
		syscall.SYS_MMAP, 0, uintptr(size),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON,
		MEMORY_DESCRIPTOR_ANONYMOUS, 0,
	)
	if operation_errno != 0 {
		return nil, operation_errno
	}
	// The region lives outside the Go heap, so the collector never moves it. Reading the
	// uintptr through its own address converts it without the form vet rejects.
	start := *(*unsafe.Pointer)(unsafe.Pointer(&address))
	return unsafe.Slice((*byte)(start), int(size)), nil
}

func system_memory_unmap(_ unsafe.Pointer, region os.Memory) (err error) {
	os.Memory_Invariants(region, "system_memory_unmap.region")
	// Kernel returns EINVAL for zero length; empty region is what failed map returned.
	if len(region) == 0 {
		return nil
	}
	_, _, operation_errno := syscall.RawSyscall(
		syscall.SYS_MUNMAP, uintptr(unsafe.Pointer(unsafe.SliceData(region))),
		uintptr(len(region)), 0,
	)
	if operation_errno != 0 {
		return operation_errno
	}
	return nil
}

// OS_Memory_Map keeps default-package callers on pure OS operation.
func OS_Memory_Map(host os.OS, size os.Memory_Size) (region os.Memory, err error) {
	defer func() { os.Memory_Invariants(region, "default_os_memory_map.region") }()
	os.OS_Invariants(host, "default_os_memory_map.host")
	os.Memory_Size_Invariants(size, "default_os_memory_map.size")
	return os.OS_Memory_Map(host, size)
}

// OS_Memory_Unmap keeps default-package callers on pure OS operation.
func OS_Memory_Unmap(host os.OS, region os.Memory) (err error) {
	os.OS_Invariants(host, "default_os_memory_unmap.host")
	os.Memory_Invariants(region, "default_os_memory_unmap.region")
	return os.OS_Memory_Unmap(host, region)
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

// OS_Exit keeps default-package callers on pure OS operation.
func OS_Exit(host os.OS, code os.Exit_Code) {
	os.OS_Invariants(host, "default_os_exit.host")
	os.Exit_Code_Invariants(code, "default_os_exit.code")
	os.OS_Exit(host, code)
}

// FILE_BYTES_MAXIMUM bounds one whole-file read or write; larger files belong on the IO loop,
// not on a synchronous gateway call.
const FILE_BYTES_MAXIMUM = 1 << 24

// FILE_BYTES_MINIMUM admits an empty file.
const FILE_BYTES_MINIMUM = 0

// FILE_PERMISSION_MAXIMUM is the full mode word: setuid, setgid, sticky, and rwxrwxrwx.
const FILE_PERMISSION_MAXIMUM = 0o7777

// FILE_PERMISSION_MINIMUM is a file nobody may touch.
const FILE_PERMISSION_MINIMUM = 0

// STANDARD_ERROR is the descriptor the process inherited for diagnostics.
const STANDARD_ERROR Descriptor = 2

// DESCRIPTOR_MINIMUM is standard input, the lowest descriptor a process holds.
const DESCRIPTOR_MINIMUM = 0

// DESCRIPTOR_MAXIMUM is the highest descriptor the default per-process limit hands out.
const DESCRIPTOR_MAXIMUM = 1<<20 - 1

// Descriptor is one open kernel file handle.
type Descriptor int

// Descriptor_Invariants keeps a handle inside the per-process table.
func Descriptor_Invariants(value Descriptor, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), DESCRIPTOR_MINIMUM, DESCRIPTOR_MAXIMUM).
		Ensure()
}

// File_Path names one file on the host filesystem.
type File_Path string

// File_Path_Invariants bounds path text the way every other host path is bounded.
func File_Path_Invariants(value File_Path, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// File_Content is one whole file's bytes.
type File_Content []byte

// File_Content_Invariants bounds a synchronous transfer.
func File_Content_Invariants(value File_Content, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), FILE_BYTES_MINIMUM, FILE_BYTES_MAXIMUM).
		Ensure()
}

// File_Permission is the mode word a created file gets, before the process umask.
type File_Permission uint32

// File_Permission_Invariants keeps the mode inside the twelve permission bits.
func File_Permission_Invariants(value File_Permission, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), FILE_PERMISSION_MINIMUM, FILE_PERMISSION_MAXIMUM).
		Ensure()
}

// File_Too_Large reports a file past FILE_BYTES_MAXIMUM rather than reading part of it.
var File_Too_Large = errors.New("os: file exceeds the synchronous read bound")

// File_Short_Write reports a write the kernel stopped advancing before the content ended.
var File_Short_Write = errors.New("os: file write made no progress")

// File_Read_Into reads one whole file into destination and returns the filled prefix. It is
// synchronous by design: it serves tools that run before or outside an IO loop, such as a test
// rewriting its own source. A file larger than destination is File_Too_Large, never a partial
// read, so the caller's storage is the bound and nothing allocates.
func File_Read_Into(path File_Path, destination File_Content) (content File_Content, err error) {
	defer func() { File_Content_Invariants(content, "File_Read_Into.content") }()
	File_Path_Invariants(path, "File_Read_Into.path")
	File_Content_Invariants(destination, "File_Read_Into.destination")
	file, open_err := syscall.Open(string(path), syscall.O_RDONLY|syscall.O_CLOEXEC, 0)
	if open_err != nil {
		return nil, open_err
	}
	defer syscall.Close(file)
	var facts syscall.Stat_t
	if stat_err := syscall.Fstat(file, &facts); stat_err != nil {
		return nil, stat_err
	}
	if facts.Size > int64(len(destination)) {
		return nil, File_Too_Large
	}
	content = destination[:int(facts.Size)]
	filled := 0
	// Each pass advances by at least one byte or ends, so the size bounds the loop.
	for pass_index := 0; pass_index < len(content); pass_index++ {
		if filled >= len(content) {
			break
		}
		count, read_err := syscall.Read(file, content[filled:])
		if read_err != nil {
			return nil, read_err
		}
		if count == 0 {
			break
		}
		filled += count
	}
	return content[:filled], nil
}

// File_Write replaces one whole file, creating it with permission when absent. Synchronous
// for the same reason as File_Read.
func File_Write(path File_Path, content File_Content, permission File_Permission) (err error) {
	File_Path_Invariants(path, "File_Write.path")
	File_Content_Invariants(content, "File_Write.content")
	File_Permission_Invariants(permission, "File_Write.permission")
	file, open_err := syscall.Open(
		string(path), syscall.O_WRONLY|syscall.O_CREAT|syscall.O_TRUNC|syscall.O_CLOEXEC,
		uint32(permission),
	)
	if open_err != nil {
		return open_err
	}
	defer syscall.Close(file)
	return descriptor_write(Descriptor(file), content)
}

// Standard_Error_Write sends content to the inherited standard error descriptor, for
// diagnostics that must land even when no IO loop is running.
func Standard_Error_Write(content File_Content) (err error) {
	File_Content_Invariants(content, "Standard_Error_Write.content")
	return descriptor_write(STANDARD_ERROR, content)
}

// A pipe or terminal may accept a partial write, so the remainder is resubmitted. Each pass
// advances by at least one byte or ends, so the content length bounds the loop.
func descriptor_write(descriptor Descriptor, content File_Content) (err error) {
	Descriptor_Invariants(descriptor, "descriptor_write.descriptor")
	File_Content_Invariants(content, "descriptor_write.content")
	written := 0
	for pass_index := 0; pass_index < len(content); pass_index++ {
		if written >= len(content) {
			break
		}
		count, write_err := syscall.Write(int(descriptor), content[written:])
		if write_err != nil {
			return write_err
		}
		if count <= 0 {
			return File_Short_Write
		}
		written += count
	}
	return nil
}
