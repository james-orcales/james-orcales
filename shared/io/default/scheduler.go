//go:build darwin || (linux && amd64)

package io

import (
	"runtime"
	"syscall"

	sharedio "local/james-orcales/shared/io"
	"local/james-orcales/shared/time"
)

// This bound holds the largest socket address from either supported platform.
const SOCKET_ADDRESS_BYTES = 28

// Operating system operation kind is one TigerBeetle kernel operation tag. The platform
// backend translates the tag directly to kqueue readiness plus a syscall on Darwin, or to
// the corresponding io_uring opcode on Linux.
type Operating_System_Operation_Kind int

// OPERATING_SYSTEM_OPERATION_ACCEPT selects the platform accept operation.
const OPERATING_SYSTEM_OPERATION_ACCEPT Operating_System_Operation_Kind = 0

// OPERATING_SYSTEM_OPERATION_CLOSE selects the platform close operation.
const OPERATING_SYSTEM_OPERATION_CLOSE Operating_System_Operation_Kind = 1

// OPERATING_SYSTEM_OPERATION_CONNECT selects the platform connect operation.
const OPERATING_SYSTEM_OPERATION_CONNECT Operating_System_Operation_Kind = 2

// OPERATING_SYSTEM_OPERATION_READ selects the platform file-read operation.
const OPERATING_SYSTEM_OPERATION_READ Operating_System_Operation_Kind = 3

// OPERATING_SYSTEM_OPERATION_RECEIVE selects the platform socket-receive operation.
const OPERATING_SYSTEM_OPERATION_RECEIVE Operating_System_Operation_Kind = 4

// OPERATING_SYSTEM_OPERATION_SEND selects the platform socket-send operation.
const OPERATING_SYSTEM_OPERATION_SEND Operating_System_Operation_Kind = 5

// OPERATING_SYSTEM_OPERATION_TIMEOUT selects the platform timeout operation.
const OPERATING_SYSTEM_OPERATION_TIMEOUT Operating_System_Operation_Kind = 6

// OPERATING_SYSTEM_OPERATION_WRITE selects the platform file-write operation.
const OPERATING_SYSTEM_OPERATION_WRITE Operating_System_Operation_Kind = 7

// OPERATING_SYSTEM_OPERATION_FSYNC selects the platform fsync operation.
const OPERATING_SYSTEM_OPERATION_FSYNC Operating_System_Operation_Kind = 8

// OPERATING_SYSTEM_OPERATION_OPEN_AT selects the platform Open_At operation.
const OPERATING_SYSTEM_OPERATION_OPEN_AT Operating_System_Operation_Kind = 9

// OPERATING_SYSTEM_OPERATION_EVENT selects the platform event operation.
const OPERATING_SYSTEM_OPERATION_EVENT Operating_System_Operation_Kind = 10

// OPERATING_SYSTEM_OPERATION_STATX selects the Linux-only statx operation.
const OPERATING_SYSTEM_OPERATION_STATX Operating_System_Operation_Kind = 11

// OPERATING_SYSTEM_OPERATION_BOUNDED_DEADLINE identifies an internal linked deadline.
const OPERATING_SYSTEM_OPERATION_BOUNDED_DEADLINE Operating_System_Operation_Kind = 12

// OPERATING_SYSTEM_OPERATION_PROCESS_EXIT waits for one spawned child to exit. Darwin arms an
// EVFILT_PROC/NOTE_EXIT kevent on the process identifier and Linux polls the child's pidfd, so
// neither platform blocks a thread in wait4.
const OPERATING_SYSTEM_OPERATION_PROCESS_EXIT Operating_System_Operation_Kind = 13

// OPERATING_SYSTEM_OPERATION_PIPE_READ reads one buffer from a pipe. It is distinct from READ
// because a pipe is not seekable: READ issues pread on Darwin and carries a file offset on Linux,
// and both reject a pipe with ESPIPE.
const OPERATING_SYSTEM_OPERATION_PIPE_READ Operating_System_Operation_Kind = 14

// OPERATING_SYSTEM_OPERATION_PIPE_WRITE writes one buffer to a pipe, the counterpart of PIPE_READ.
const OPERATING_SYSTEM_OPERATION_PIPE_WRITE Operating_System_Operation_Kind = 15

// Kernel timespec is Linux's stable UAPI timespec layout. It lives with the operation so an
// io_uring timeout never points at stack storage while it is in the kernel.
type Kernel_Timespec struct {
	// Seconds preserves the Linux UAPI layout.
	Seconds int64
	// Nanoseconds preserves the Linux UAPI layout.
	Nanoseconds int64
}

// Operating system operation is the Go counterpart of TigerBeetle IO.Completion.operation.
// Identifier is written to kernel user_data instead of a Go pointer: the registry owns the
// operation until the kernel retires that integer identifier.
type Operating_System_Operation struct {
	Platform_Operation
	// Identifier correlates the operation without a Go pointer in the kernel.
	Identifier uint64
	// Completion preserves the caller identity until callback delivery.
	Completion *sharedio.Completion
	// Kind selects the platform operation.
	Kind Operating_System_Operation_Kind
	// Descriptor preserves caller ownership during asynchronous kernel use.
	Descriptor int
	// Buffer retains operation memory until the kernel retires it.
	Buffer []byte
	// Offset preserves the requested file position.
	Offset uint64
	// Address retains the typed socket address until submission.
	Address sharedio.Address
	// File_Path retains zero-terminated path memory until the kernel retires it.
	File_Path []byte
	// Open_Options retain Open_At behavior until submission.
	Open_Options sharedio.Open_At_Options
	// Event_Value correlates a synthetic event without a Go pointer.
	Event_Value uint64
	// Process_Identifier names the spawned child a PROCESS_EXIT operation waits for. Darwin
	// uses it as the kevent Ident, and both platforms reap with it.
	Process_Identifier int
	// Initiated prevents Darwin from issuing connect twice after readiness.
	Initiated bool
	// Timespec retains timeout memory until the kernel retires it.
	Timespec Kernel_Timespec
	// Deadline bounds an operation on Darwin.
	Deadline time.Moment
	// Deadline_Span retains linked-timeout memory until Linux retires it.
	Deadline_Span Kernel_Timespec
	// Bounded joins one Linux operation to its linked deadline.
	Bounded *Operating_System_Bounded_Operation
	// Socket_Address retains sockaddr memory until the kernel retires it.
	Socket_Address [SOCKET_ADDRESS_BYTES]byte
	// Socket_Address_Size tells the kernel which sockaddr bytes are valid.
	Socket_Address_Size uint32
	// Deliver keeps callback delivery behind scheduler retirement.
	Deliver func(result int, err error)
	// Pinner prevents the Go runtime from moving kernel-owned memory.
	Pinner runtime.Pinner
	// Pinned prevents duplicate unpin operations.
	Pinned bool
}

// Operating system bounded operation joins a primary operation and its internal Linux link timeout
// before exposing either result. Darwin uses Deadline directly and never allocates the join.
type Operating_System_Bounded_Operation struct {
	// Operation retains the primary operation until both linked SQEs retire.
	Operation *Operating_System_Operation
	// Deadline_Operation retains the internal timeout until both linked SQEs retire.
	Deadline_Operation *Operating_System_Operation
	// Operation_Result preserves the primary result until the linked timeout retires.
	Operation_Result int32
	// Deadline_Result preserves the timeout result until the primary operation retires.
	Deadline_Result int32
	// Operation_Completed prevents duplicate primary retirement.
	Operation_Completed bool
	// Deadline_Completed prevents duplicate deadline retirement.
	Deadline_Completed bool
}

// Operating system operation submit gives an operation its kernel correlation identifier and
// hands it to the platform scheduler. A submission failure is delivered through the same
// completed queue as an ordinary kernel result.
func operating_system_operation_submit(
	state *Operating_System, operation *Operating_System_Operation,
) {
	operating_system_operation_register(state, operation)
	err := platform_submit(state, operation)
	if err != nil {
		operating_system_operation_complete(state, operation, 0, err)
	}
}

// Operating system operation register allocates the generation token written to kernel userdata.
func operating_system_operation_register(
	state *Operating_System, operation *Operating_System_Operation,
) {
	if operation.Kind == OPERATING_SYSTEM_OPERATION_EVENT {
		operation.Identifier = operation.Completion.Kernel_Identifier
	}
	if operation.Identifier == 0 {
		state.Next_Identifier++
		operation.Identifier = state.Next_Identifier
	}
	operation.Completion.Kernel_Identifier = operation.Identifier
	state.Operations[operation.Identifier] = operation
}

// Operating system operation complete retires the registry entry and every pinned address
// before exposing the result to application code. This is the retire-before-deliver ordering
// in third-party/tigerbeetle/src/io/darwin.zig:101-156 and io/linux.zig:125-209.
func operating_system_operation_complete(
	state *Operating_System, operation *Operating_System_Operation,
	result int, err error,
) {
	delete(state.Operations, operation.Identifier)
	if operation.Pinned {
		operation.Pinner.Unpin()
		operation.Pinned = false
	}
	operating_system_operation_account(state, operation, result, err)
	operation.Completion.Callback = func() { operation.Deliver(result, err) }
	state.Completed = append(state.Completed, operation.Completion)
}

// Operating system operation account updates descriptor ownership only after the kernel has
// completed the operation that creates or releases the descriptor.
func operating_system_operation_account(
	state *Operating_System, operation *Operating_System_Operation,
	result int, err error,
) {
	if err == nil {
		if operation.Kind == OPERATING_SYSTEM_OPERATION_ACCEPT {
			state.Raw_Open[result] = true
		}
	}
	if err == nil {
		if operation.Kind == OPERATING_SYSTEM_OPERATION_OPEN_AT {
			state.Raw_Open[result] = true
		}
	}
	if err == nil {
		if operation.Kind == OPERATING_SYSTEM_OPERATION_CLOSE {
			delete(state.Raw_Open, operation.Descriptor)
		}
	}
}

// Operating system translate result applies the portable result variants shared/io exposes.
// Other errno values remain raw operating-system errors, as TigerBeetle's typed error unions do.
func operating_system_translate_result(
	operation *Operating_System_Operation, result int32,
) (count int, err error) {
	if result >= 0 {
		return int(result), nil
	}
	errno := syscall.Errno(-result)
	if errno == syscall.ECONNREFUSED {
		return 0, sharedio.Connection_Refused
	}
	if errno == syscall.EPIPE {
		return 0, sharedio.Broken_Pipe
	}
	if errno == syscall.ENOTCONN {
		return 0, sharedio.Socket_Not_Connected
	}
	if errno == syscall.ECANCELED {
		return 0, sharedio.Canceled
	}
	if operation.Kind == OPERATING_SYSTEM_OPERATION_TIMEOUT {
		if errno == syscall.ETIME {
			return 0, nil
		}
	}
	if operation.Kind == OPERATING_SYSTEM_OPERATION_CLOSE {
		if errno == syscall.EINTR {
			return 0, nil
		}
	}
	return 0, errno
}

// Operating system retryable result reports the io_uring results TigerBeetle resubmits rather
// than delivers. EINTR applies to every kernel operation except close; file reads and writes
// also retry EAGAIN, matching io/linux.zig's Completion.complete switch.
func operating_system_retryable_result(
	operation *Operating_System_Operation, result int32,
) (retry bool) {
	if result >= 0 {
		return false
	}
	errno := syscall.Errno(-result)
	if errno == syscall.EINTR {
		return operation.Kind != OPERATING_SYSTEM_OPERATION_CLOSE
	}
	if errno != syscall.EAGAIN {
		return false
	}
	if operation.Kind == OPERATING_SYSTEM_OPERATION_READ {
		return true
	}
	if operation.Kind == OPERATING_SYSTEM_OPERATION_PIPE_READ {
		return true
	}
	if operation.Kind == OPERATING_SYSTEM_OPERATION_PIPE_WRITE {
		return true
	}
	return operation.Kind == OPERATING_SYSTEM_OPERATION_WRITE
}

// Operating system timeout span converts a positive duration to the stable kernel layout.
func operating_system_timeout_span(duration time.Duration) (span Kernel_Timespec) {
	nanoseconds := int64(duration)
	span.Seconds = nanoseconds / 1_000_000_000
	span.Nanoseconds = nanoseconds % 1_000_000_000
	return span
}
