//go:build darwin

package io

import (
	"syscall"
	"unsafe"

	invariant "local/james-orcales/g/shared/invariant/default"
	sharedio "local/james-orcales/g/shared/io"
	"local/james-orcales/g/shared/time"
)

const socket_receive_buffer_size = 4 * 1024 * 1024
const socket_send_buffer_size = 2 * 1024 * 1024
const socket_no_sigpipe = 0x1022
const darwin_open_at_call = 463
const darwin_current_directory = -2
const darwin_buffer_size_max = 0x7fffffff

// Platform operation has no Darwin-only fields.
type Platform_Operation struct {
	// Backlogged reports the operation is waiting for its one-shot kevent registration.
	Backlogged bool
	// Kernel_Submitted reports kqueue owns the one-shot registration.
	Kernel_Submitted bool
}

// Wires no Linux-only operations on Darwin.
func operating_system_wire_platform(state *operating_system, loop *sharedio.IO) {
	loop.Platform_IO = sharedio.Platform_IO{}
}

// Applies TigerBeetle io.buffer_limit for Darwin before a length reaches a signed kernel result.
func platform_buffer_limit(buffer []byte) (limited []byte) {
	if len(buffer) > darwin_buffer_size_max {
		return buffer[:darwin_buffer_size_max]
	}
	return buffer
}

// Opens and configures one non-blocking close-on-exec TCP socket, following
// third-party/tigerbeetle/src/io/darwin.zig:914-929 and common.zig:65-117.
func socket_open_tcp(
	family sharedio.Address_Family, options sharedio.TCP_Options,
) (descriptor int, err error) {
	descriptor, err = socket_create(&socket_create_input{
		Family: family, Type: syscall.SOCK_STREAM, Protocol: syscall.IPPROTO_TCP,
	})
	if err != nil {
		return -1, err
	}
	setup_err := socket_configure(descriptor, options)
	if setup_err != nil {
		syscall.Close(descriptor)
		return -1, setup_err
	}
	return descriptor, nil
}

// Opens one non-blocking close-on-exec UDP socket.
func socket_open_udp(family sharedio.Address_Family) (descriptor int, err error) {
	return socket_create(&socket_create_input{
		Family: family, Type: syscall.SOCK_DGRAM, Protocol: syscall.IPPROTO_UDP,
	})
}

// Input for socket_create.
type socket_create_input struct {
	// Family is the socket address family.
	Family sharedio.Address_Family
	// Type is SOCK_STREAM or SOCK_DGRAM.
	Type int
	// Protocol is the transport protocol.
	Protocol int
}

// Creates one socket and applies the Darwin nonblocking and close-on-exec setup.
func socket_create(input *socket_create_input) (descriptor int, err error) {
	descriptor, err = syscall.Socket(
		socket_family(input.Family), input.Type, input.Protocol,
	)
	if err != nil {
		return -1, err
	}
	non_block_err := syscall.SetNonblock(descriptor, true)
	if non_block_err != nil {
		syscall.Close(descriptor)
		return -1, non_block_err
	}
	close_on_exec_err := socket_close_on_exec(descriptor)
	if close_on_exec_err != nil {
		syscall.Close(descriptor)
		return -1, close_on_exec_err
	}
	return descriptor, nil
}

// Maps the backend-independent family to Darwin's socket family constant.
func socket_family(family sharedio.Address_Family) (system int) {
	if family == sharedio.FAMILY_IPV6 {
		return syscall.AF_INET6
	}
	return syscall.AF_INET
}

// Marks descriptor close-on-exec while preserving the fcntl failure so socket_open can return
// the original setup error after releasing the descriptor.
func socket_close_on_exec(descriptor int) (err error) {
	_, _, errno := syscall.Syscall(
		syscall.SYS_FCNTL, uintptr(descriptor), uintptr(syscall.F_SETFD),
		uintptr(syscall.FD_CLOEXEC),
	)
	if errno != 0 {
		return errno
	}
	return nil
}

// Applies the two Darwin accepted-socket guarantees from io/darwin.zig:324-360.
func socket_accept_configure(descriptor int) (err error) {
	close_err := socket_close_on_exec(descriptor)
	if close_err != nil {
		return close_err
	}
	return syscall.SetsockoptInt(descriptor, syscall.SOL_SOCKET, socket_no_sigpipe, 1)
}

func socket_configure(descriptor int, options sharedio.TCP_Options) (err error) {
	if options.Receive_Buffer > 0 {
		err = syscall.SetsockoptInt(
			descriptor, syscall.SOL_SOCKET, syscall.SO_RCVBUF, options.Receive_Buffer,
		)
		if err != nil {
			return err
		}
	}
	if options.Send_Buffer > 0 {
		err = syscall.SetsockoptInt(
			descriptor, syscall.SOL_SOCKET, syscall.SO_SNDBUF, options.Send_Buffer,
		)
		if err != nil {
			return err
		}
	}
	if options.Keepalive != nil {
		err = syscall.SetsockoptInt(descriptor, syscall.SOL_SOCKET, syscall.SO_KEEPALIVE, 1)
		if err != nil {
			return err
		}
	}
	return syscall.SetsockoptInt(descriptor, syscall.SOL_SOCKET, socket_no_sigpipe, 1)
}

// Platform scheduler is TigerBeetle Darwin IO's kqueue plus its two kernel-facing queue
// counts (third-party/tigerbeetle/src/io/darwin.zig:17-31).
type platform_scheduler struct {
	Descriptor  int
	IO_Backlog  []*operating_system_operation
	IO_Inflight int
	Next_Event  uint64
}

// Kernel event is Darwin's 64-bit struct kevent layout with integer udata. Using the UAPI
// layout avoids placing a Go pointer in the kernel while preserving TigerBeetle's completion
// correlation through kevent.udata (io/darwin.zig:158-182).
type kernel_event struct {
	Ident        uint64
	Filter       int16
	Flags        uint16
	Filter_Flags uint32
	Data         int64
	User_Data    uint64
}

// Platform initialize eagerly creates kqueue. Darwin intentionally ignores entries and flags,
// exactly as TigerBeetle IO.init does in io/darwin.zig:33-42.
func platform_initialize(entries uint16, flags uint32) (platform platform_scheduler, err error) {
	descriptor, create_err := syscall.Kqueue()
	if create_err != nil {
		return platform_scheduler{}, create_err
	}
	return platform_scheduler{Descriptor: descriptor}, nil
}

// Platform deinitialize releases kqueue after the owner has joined every operation.
func platform_deinitialize(state *operating_system) {
	if state.Platform.Descriptor >= 0 {
		syscall.Close(state.Platform.Descriptor)
		state.Platform.Descriptor = -1
	}
}

// Platform uses kernel timeouts reports that Darwin keeps deadlines in io.timeouts and expires
// them before each kevent pass (io/darwin.zig:184-209).
func platform_uses_kernel_timeouts() (uses bool) { return false }

// Platform submit eagerly attempts the syscall. WouldBlock alone enters io_pending, matching
// the submit wrapper at io/darwin.zig:268-312.
func platform_submit(
	state *operating_system, operation *operating_system_operation,
) (err error) {
	return platform_submit_registered(state, operation)
}

// Platform submit registered is the eager Darwin retry for an already registered operation.
func platform_submit_registered(
	state *operating_system, operation *operating_system_operation,
) (err error) {
	result, again, operation_err := operating_system_operation_do(operation)
	if again {
		operation.Backlogged = true
		state.Platform.IO_Backlog = append(state.Platform.IO_Backlog, operation)
		return nil
	}
	operating_system_operation_complete(state, operation, result, operation_err)
	return nil
}

// Operating system operation do executes one Darwin operation at completion time. Connect is
// attempted once; after kqueue requeues it, SO_ERROR is checked instead of connect being issued
// again, avoiding EISCONN (io/darwin.zig:442-458).
func operating_system_operation_do(
	operation *operating_system_operation,
) (result int, again bool, err error) {
	switch operation.Kind {
	case operating_system_operation_accept:
		return socket_accept(operation.Descriptor)
	case operating_system_operation_close:
		return 0, false, socket_close(operation.Descriptor)
	case operating_system_operation_connect:
		return socket_connect_attempt(operation)
	case operating_system_operation_read:
		count, read_err := read_at(
			sharedio.File(operation.Descriptor),
			operation.Buffer,
			int64(operation.Offset),
		)
		return count, socket_again(read_err), read_err
	case operating_system_operation_receive:
		return socket_receive(operation.Descriptor, operation.Buffer)
	case operating_system_operation_send:
		count, would_block, send_err := socket_send(operation.Descriptor, operation.Buffer)
		return count, would_block, socket_send_translate(send_err)
	case operating_system_operation_write:
		count, write_err := write_at(
			sharedio.File(operation.Descriptor),
			operation.Buffer,
			int64(operation.Offset),
		)
		return count, socket_again(write_err), write_err
	case operating_system_operation_fsync:
		return 0, false, syscall.Fsync(operation.Descriptor)
	case operating_system_operation_open_at:
		descriptor, open_err := platform_open_at(operation)
		return descriptor, false, open_err
	default:
		return 0, false, syscall.EINVAL
	}
}

// Returns Darwin's AT_FDCWD value used by TigerBeetle IO.openat.
func platform_current_directory() (descriptor int) { return darwin_current_directory }

// Executes Darwin openat synchronously when its eager completion runs, retrying EINTR and forcing
// CLOEXEC exactly as third-party/tigerbeetle/src/io/darwin.zig:492-558.
func platform_open_at(operation *operating_system_operation) (descriptor int, err error) {
	flags := platform_open_flags(operation.Open_Options) | syscall.O_CLOEXEC
	path := unsafe.Pointer(&operation.File_Path[0])
	for retry := true; retry; {
		result, _, errno := syscall.Syscall6(
			darwin_open_at_call,
			uintptr(operation.Descriptor), uintptr(path), uintptr(flags),
			uintptr(operation.Open_Options.Mode), 0, 0,
		)
		if errno == syscall.EINTR {
			continue
		}
		if errno != 0 {
			return -1, errno
		}
		return int(result), nil
	}
	return -1, syscall.EINTR
}

// Translates the portable Open_At option fields to Darwin posix.O bits.
func platform_open_flags(options sharedio.Open_At_Options) (flags int) {
	flags = syscall.O_RDONLY
	if options.Access == sharedio.OPEN_WRITE_ONLY {
		flags = syscall.O_WRONLY
	}
	if options.Access == sharedio.OPEN_READ_WRITE {
		flags = syscall.O_RDWR
	}
	if options.Create {
		flags |= syscall.O_CREAT
	}
	if options.Truncate {
		flags |= syscall.O_TRUNC
	}
	return flags
}

// Socket connect attempt performs the first connect or the post-readiness SO_ERROR check.
func socket_connect_attempt(
	operation *operating_system_operation,
) (result int, again bool, err error) {
	if operation.Initiated {
		return 0, false, socket_connect_error(operation.Descriptor)
	}
	operation.Initiated = true
	system, address_err := socket_address(operation.Address)
	if address_err != nil {
		return 0, false, address_err
	}
	connect_err := syscall.Connect(operation.Descriptor, system)
	if connect_err == syscall.EINPROGRESS {
		return 0, true, nil
	}
	if connect_err == syscall.EALREADY {
		return 0, true, nil
	}
	return 0, socket_again(connect_err), socket_connect_translate(connect_err)
}

// Platform run ports the Darwin flush pass: io_pending becomes one-shot changes, kevent returns
// identifiers into completed work, and callbacks are drained later by the common completed queue.
func platform_run(state *operating_system, wait time.Moment) (err error) {
	changes := platform_changes(state)
	if len(changes) == 0 {
		if len(state.Completed) > 0 {
			return nil
		}
	}
	if len(changes) == 0 {
		if state.Platform.IO_Inflight == 0 {
			if wait == 0 {
				return nil
			}
		}
	}
	kernel_wait := wait
	if len(changes) > 0 {
		kernel_wait = 0
	}
	if len(state.Completed) > 0 {
		kernel_wait = 0
	}
	events := make([]kernel_event, poll_events_max)
	count, wait_err := kernel_kevent(&kernel_kevent_input{
		Descriptor: state.Platform.Descriptor,
		Changes:    changes,
		Events:     events,
		Wait:       kernel_wait,
	})
	if wait_err != nil {
		platform_restore_changes(state, changes)
		return wait_err
	}
	for _, change := range changes {
		operation := state.Operations[change.User_Data]
		if operation != nil {
			operation.Kernel_Submitted = true
		}
	}
	state.Platform.IO_Inflight += len(changes)
	operation_events := 0
	for index := 0; index < count; index++ {
		if state.Operations[events[index].User_Data] != nil {
			operation_events++
		}
	}
	state.Platform.IO_Inflight -= operation_events
	return platform_complete_events(state, events[:count])
}

// Darwin eagerly attempts callback submissions in the live completed drain; WouldBlock work stays
// in io_pending until the next flush, so Linux's post-callback SQ flush has no Darwin counterpart.
func platform_flush_submissions(state *operating_system) (err error) { return nil }

// Platform changes removes at most one kevent pass of io_pending operations and encodes their
// integer registry identifiers in one-shot kernel changes.
func platform_changes(state *operating_system) (changes []kernel_event) {
	count := len(state.Platform.IO_Backlog)
	if count > poll_events_max {
		count = poll_events_max
	}
	changes = make([]kernel_event, count)
	for index := 0; index < count; index++ {
		operation := state.Platform.IO_Backlog[index]
		operation.Backlogged = false
		changes[index] = kernel_event{
			Ident:     uint64(operation.Descriptor),
			Filter:    platform_filter(operation.Kind),
			Flags:     syscall.EV_ADD | syscall.EV_ENABLE | syscall.EV_ONESHOT,
			User_Data: operation.Identifier,
		}
	}
	state.Platform.IO_Backlog = state.Platform.IO_Backlog[count:]
	return changes
}

// Platform restore changes puts changes back at the head if kevent rejects the changelist.
func platform_restore_changes(state *operating_system, changes []kernel_event) {
	restored := make(
		[]*operating_system_operation,
		0,
		len(changes)+len(state.Platform.IO_Backlog),
	)
	for _, change := range changes {
		operation := state.Operations[change.User_Data]
		if operation != nil {
			operation.Backlogged = true
			restored = append(restored, operation)
		}
	}
	state.Platform.IO_Backlog = append(restored, state.Platform.IO_Backlog...)
}

// Platform filter maps each operation to TigerBeetle's read or write kqueue filter.
func platform_filter(kind operating_system_operation_kind) (filter int16) {
	if kind == operating_system_operation_connect {
		return syscall.EVFILT_WRITE
	}
	if kind == operating_system_operation_send {
		return syscall.EVFILT_WRITE
	}
	if kind == operating_system_operation_write {
		return syscall.EVFILT_WRITE
	}
	return syscall.EVFILT_READ
}

// Platform complete events re-attempts each one-shot operation and requeues only WouldBlock.
func platform_complete_events(
	state *operating_system, events []kernel_event,
) (err error) {
	for _, event := range events {
		operation := state.Operations[event.User_Data]
		if operation == nil {
			continue
		}
		operation.Kernel_Submitted = false
		if operation.Kind == operating_system_operation_event {
			operating_system_operation_complete(state, operation, 0, nil)
			continue
		}
		submit_err := platform_submit_registered(state, operation)
		if submit_err != nil {
			return submit_err
		}
	}
	return nil
}

// Platform expire operation removes a bounded one-shot accept from either io_pending or kqueue.
// This explicit deadline is a documented repository divergence from TigerBeetle's unbounded
// accept; it is internal retirement, not a public Cancel surface.
func platform_expire_operation(
	state *operating_system, operation *operating_system_operation,
) (err error) {
	if operation.Backlogged {
		kept := state.Platform.IO_Backlog[:0]
		for _, operation_value := range state.Platform.IO_Backlog {
			if operation_value != operation {
				kept = append(kept, operation_value)
			}
		}
		state.Platform.IO_Backlog = kept
		operation.Backlogged = false
	}
	if !operation.Kernel_Submitted {
		return nil
	}
	change := kernel_event{
		Ident: uint64(operation.Descriptor), Filter: platform_filter(operation.Kind),
		Flags: syscall.EV_DELETE,
	}
	_, delete_err := kernel_kevent(&kernel_kevent_input{
		Descriptor: state.Platform.Descriptor,
		Changes:    []kernel_event{change},
		Wait:       poll_forever,
	})
	if delete_err != nil {
		if delete_err != syscall.ENOENT {
			return delete_err
		}
	}
	operation.Kernel_Submitted = false
	invariant.Always(state.Platform.IO_Inflight > 0,
		"An expired kqueue operation was counted in flight.")
	state.Platform.IO_Inflight--
	return nil
}

// Opens a persistent EVFILT_USER Event exactly as TigerBeetle Darwin IO.open_event.
func platform_event_open(state *operating_system) (event sharedio.Event, err error) {
	state.Platform.Next_Event++
	event = sharedio.Event(state.Platform.Next_Event)
	change := kernel_event{
		Ident: uint64(event), Filter: syscall.EVFILT_USER,
		Flags: syscall.EV_ADD | syscall.EV_ENABLE | syscall.EV_CLEAR,
	}
	count, open_err := kernel_kevent(&kernel_kevent_input{
		Descriptor: state.Platform.Descriptor,
		Changes:    []kernel_event{change},
		Wait:       poll_forever,
	})
	if open_err != nil {
		return 0, open_err
	}
	invariant.Always(count == 0, "Opening an EVFILT_USER Event returns no completion.")
	return event, nil
}

// Arms an already-open persistent EVFILT_USER Event by recording one operation in flight.
func platform_event_listen(
	state *operating_system, operation *operating_system_operation,
) (err error) {
	state.Platform.IO_Inflight++
	return nil
}

// Triggers EVFILT_USER with the listener's stable integer token in udata.
func platform_event_trigger(
	state *operating_system, event sharedio.Event, identifier uint64,
) {
	change := kernel_event{
		Ident: uint64(event), Filter: syscall.EVFILT_USER,
		Filter_Flags: syscall.NOTE_TRIGGER,
		User_Data:    identifier,
	}
	count, trigger_err := kernel_kevent(&kernel_kevent_input{
		Descriptor: state.Platform.Descriptor,
		Changes:    []kernel_event{change},
		Wait:       poll_forever,
	})
	invariant.Always(trigger_err == nil, "Triggering an EVFILT_USER Event succeeds.")
	invariant.Always(
		count == 0, "Triggering an EVFILT_USER Event returns no completion inline.",
	)
}

// Deletes one persistent EVFILT_USER Event after its listener has drained.
func platform_event_close(state *operating_system, event sharedio.Event) {
	change := kernel_event{
		Ident: uint64(event), Filter: syscall.EVFILT_USER, Flags: syscall.EV_DELETE,
	}
	count, close_err := kernel_kevent(&kernel_kevent_input{
		Descriptor: state.Platform.Descriptor,
		Changes:    []kernel_event{change},
		Wait:       poll_forever,
	})
	invariant.Always(close_err == nil, "Closing an EVFILT_USER Event succeeds.")
	invariant.Always(count == 0, "Closing an EVFILT_USER Event returns no completion.")
}

// Platform in flight reports work that can wake an unbounded drive.
func platform_in_flight(state *operating_system) (in_flight bool) {
	return len(state.Platform.IO_Backlog) > 0 || state.Platform.IO_Inflight > 0
}

// Platform counts returns Darwin's exact backlog and in-flight census.
func platform_counts(state *operating_system) (backlog int, inflight int, queued int, kernel int) {
	return len(state.Platform.IO_Backlog), state.Platform.IO_Inflight, 0, 0
}

// Kernel kevent input carries one raw kevent syscall invocation.
type kernel_kevent_input struct {
	Descriptor int
	Changes    []kernel_event
	Events     []kernel_event
	Wait       time.Moment
}

// Kernel kevent invokes Darwin's kevent syscall with the raw integer-udata UAPI layout.
func kernel_kevent(input *kernel_kevent_input) (count int, err error) {
	change_pointer := unsafe.Pointer(nil)
	if len(input.Changes) > 0 {
		change_pointer = unsafe.Pointer(&input.Changes[0])
	}
	event_pointer := unsafe.Pointer(nil)
	if len(input.Events) > 0 {
		event_pointer = unsafe.Pointer(&input.Events[0])
	}
	timeout_pointer := unsafe.Pointer(nil)
	span := syscall.Timespec{}
	if input.Wait >= 0 {
		span = syscall.NsecToTimespec(int64(input.Wait))
		timeout_pointer = unsafe.Pointer(&span)
	}
	result, _, errno := syscall.Syscall6(
		syscall.SYS_KEVENT,
		uintptr(input.Descriptor),
		uintptr(change_pointer),
		uintptr(len(input.Changes)),
		uintptr(event_pointer),
		uintptr(len(input.Events)),
		uintptr(timeout_pointer),
	)
	// TigerBeetle calls posix.kevent at third-party/tigerbeetle/src/io/darwin.zig:129-134;
	// Zig's POSIX wrapper retries EINTR instead of exposing it to the driver.
	for errno == syscall.EINTR {
		result, _, errno = syscall.Syscall6(
			syscall.SYS_KEVENT,
			uintptr(input.Descriptor),
			uintptr(change_pointer),
			uintptr(len(input.Changes)),
			uintptr(event_pointer),
			uintptr(len(input.Events)),
			uintptr(timeout_pointer),
		)
	}
	if errno != 0 {
		return 0, errno
	}
	return int(result), nil
}
