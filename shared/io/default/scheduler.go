//go:build darwin || (linux && amd64)

package io

import (
	"runtime"
	"syscall"

	sharedio "local/james-orcales/g/shared/io"
	"local/james-orcales/g/shared/time"
)

// Operating system operation kind is one TigerBeetle kernel operation tag. The platform
// backend translates the tag directly to kqueue readiness plus a syscall on Darwin, or to
// the corresponding io_uring opcode on Linux.
type operating_system_operation_kind int

const operating_system_operation_accept operating_system_operation_kind = 0
const operating_system_operation_close operating_system_operation_kind = 1
const operating_system_operation_connect operating_system_operation_kind = 2
const operating_system_operation_read operating_system_operation_kind = 3
const operating_system_operation_receive operating_system_operation_kind = 4
const operating_system_operation_send operating_system_operation_kind = 5
const operating_system_operation_timeout operating_system_operation_kind = 6
const operating_system_operation_write operating_system_operation_kind = 7
const operating_system_operation_fsync operating_system_operation_kind = 8
const operating_system_operation_open_at operating_system_operation_kind = 9
const operating_system_operation_event operating_system_operation_kind = 10
const operating_system_operation_statx operating_system_operation_kind = 11
const operating_system_operation_bounded_deadline operating_system_operation_kind = 12

// Kernel timespec is Linux's stable UAPI timespec layout. It lives with the operation so an
// io_uring timeout never points at stack storage while it is in the kernel.
type kernel_timespec struct {
	Seconds     int64
	Nanoseconds int64
}

// Operating system operation is the Go counterpart of TigerBeetle IO.Completion.operation.
// Identifier is written to kernel user_data instead of a Go pointer: the registry owns the
// operation until the kernel retires that integer identifier.
type operating_system_operation struct {
	Platform_Operation
	Identifier          uint64
	Completion          *sharedio.Completion
	Kind                operating_system_operation_kind
	Descriptor          int
	Buffer              []byte
	Offset              uint64
	Address             sharedio.Address
	File_Path           []byte
	Open_Options        sharedio.Open_At_Options
	Event_Value         uint64
	Initiated           bool
	Timespec            kernel_timespec
	Deadline            time.Moment
	Deadline_Span       kernel_timespec
	Bounded             *operating_system_bounded_operation
	Socket_Address      [28]byte
	Socket_Address_Size uint32
	Deliver             func(result int, err error)
	Pinner              runtime.Pinner
	Pinned              bool
}

// Operating system bounded operation joins a primary operation and its internal Linux link timeout
// before exposing either result. Darwin uses Deadline directly and never allocates the join.
type operating_system_bounded_operation struct {
	Operation           *operating_system_operation
	Deadline_Operation  *operating_system_operation
	Operation_Result    int32
	Deadline_Result     int32
	Operation_Completed bool
	Deadline_Completed  bool
}

// Operating system operation submit gives an operation its kernel correlation identifier and
// hands it to the platform scheduler. A submission failure is delivered through the same
// completed queue as an ordinary kernel result.
func operating_system_operation_submit(
	state *operating_system, operation *operating_system_operation,
) {
	operating_system_operation_register(state, operation)
	err := platform_submit(state, operation)
	if err != nil {
		operating_system_operation_complete(state, operation, 0, err)
	}
}

// Operating system operation register allocates the generation token written to kernel userdata.
func operating_system_operation_register(
	state *operating_system, operation *operating_system_operation,
) {
	if operation.Kind == operating_system_operation_event {
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
	state *operating_system, operation *operating_system_operation,
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
	state *operating_system, operation *operating_system_operation,
	result int, err error,
) {
	if err == nil {
		if operation.Kind == operating_system_operation_accept {
			state.Raw_Open[result] = true
		}
	}
	if err == nil {
		if operation.Kind == operating_system_operation_open_at {
			state.Raw_Open[result] = true
		}
	}
	if err == nil {
		if operation.Kind == operating_system_operation_close {
			delete(state.Raw_Open, operation.Descriptor)
		}
	}
}

// Operating system operation retry resubmits the same registered operation after a transient
// kernel interruption. Its identifier and completion remain armed throughout the retry.
func operating_system_operation_retry(
	state *operating_system, operation *operating_system_operation,
) (err error) {
	return platform_submit_registered(state, operation)
}

// Operating system translate result applies the portable result variants shared/io exposes.
// Other errno values remain raw operating-system errors, as TigerBeetle's typed error unions do.
func operating_system_translate_result(
	operation *operating_system_operation, result int32,
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
	if operation.Kind == operating_system_operation_timeout {
		if errno == syscall.ETIME {
			return 0, nil
		}
	}
	if operation.Kind == operating_system_operation_close {
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
	operation *operating_system_operation, result int32,
) (retry bool) {
	if result >= 0 {
		return false
	}
	errno := syscall.Errno(-result)
	if errno == syscall.EINTR {
		return operation.Kind != operating_system_operation_close
	}
	if errno != syscall.EAGAIN {
		return false
	}
	if operation.Kind == operating_system_operation_read {
		return true
	}
	return operation.Kind == operating_system_operation_write
}

// Operating system timeout span converts a positive duration to the stable kernel layout.
func operating_system_timeout_span(duration time.Duration) (span kernel_timespec) {
	nanoseconds := int64(duration)
	span.Seconds = nanoseconds / 1_000_000_000
	span.Nanoseconds = nanoseconds % 1_000_000_000
	return span
}
