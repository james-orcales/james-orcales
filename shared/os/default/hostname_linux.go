//go:build linux

package os

import (
	"syscall"
	"unsafe"
)

// Linux exposes execve through its native syscall table.
func system_execve(workspace_pointer unsafe.Pointer) (err error) {
	workspace := (*Self_Exec_Workspace)(workspace_pointer)
	_, _, operation_errno := syscall.RawSyscall(
		syscall.SYS_EXECVE,
		uintptr(unsafe.Pointer(&workspace.Path[0])),
		uintptr(unsafe.Pointer(&workspace.Argument_Pointers[0])),
		uintptr(unsafe.Pointer(&workspace.Environment_Pointers[0])),
	)
	if operation_errno != 0 {
		return operation_errno
	}
	return nil
}
