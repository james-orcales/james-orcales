//go:build darwin

package io

import (
	"syscall"
	"unsafe"

	invariant "local/james-orcales/shared/invariant/default"
	sharedio "local/james-orcales/shared/io"
	"local/james-orcales/shared/time"
)

// SOCKET_RECEIVE_BUFFER_SIZE fixes the socket profile that the platform tests verify.
const SOCKET_RECEIVE_BUFFER_SIZE = 4 * 1024 * 1024

// SOCKET_SEND_BUFFER_SIZE fixes the socket profile that the platform tests verify.
const SOCKET_SEND_BUFFER_SIZE = 2 * 1024 * 1024

// SOCKET_NO_SIGPIPE identifies SO_NOSIGPIPE because syscall does not expose it.
const SOCKET_NO_SIGPIPE = 0x1022

// DARWIN_OPEN_AT_CALL keeps the raw syscall compatible with Darwin amd64.
const DARWIN_OPEN_AT_CALL = 463

// DARWIN_CURRENT_DIRECTORY keeps Open_At compatible with Darwin AT_FDCWD.
const DARWIN_CURRENT_DIRECTORY = -2

// DARWIN_BUFFER_SIZE_MAX prevents a byte count from overflowing a signed kernel result.
const DARWIN_BUFFER_SIZE_MAX = 0x7fffffff

// Platform operation has no Darwin-only fields.
type Platform_Operation struct {
	// Backlogged reports the operation is waiting for its one-shot kevent registration.
	Backlogged bool
	// Kernel_Submitted reports kqueue owns the one-shot registration.
	Kernel_Submitted bool
}

// Wires no Linux-only operations on Darwin.
func operating_system_wire_platform(state *Operating_System, loop *sharedio.IO) {
	loop.Platform_IO = sharedio.Platform_IO{}
}

// Applies TigerBeetle io.buffer_limit for Darwin before a length reaches a signed kernel result.
func platform_buffer_limit(buffer []byte) (limited []byte) {
	if len(buffer) > DARWIN_BUFFER_SIZE_MAX {
		return buffer[:DARWIN_BUFFER_SIZE_MAX]
	}
	return buffer
}

// Opens and configures one non-blocking close-on-exec TCP socket, following
// third-party/tigerbeetle/src/io/darwin.zig:914-929 and common.zig:65-117.
func socket_open_tcp(
	family sharedio.Address_Family, options sharedio.TCP_Options,
) (descriptor int, err error) {
	descriptor, err = socket_create(&Socket_Create_Input{
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
	return socket_create(&Socket_Create_Input{
		Family: family, Type: syscall.SOCK_DGRAM, Protocol: syscall.IPPROTO_UDP,
	})
}

// Input for socket_create.
type Socket_Create_Input struct {
	// Family is the socket address family.
	Family sharedio.Address_Family
	// Type is SOCK_STREAM or SOCK_DGRAM.
	Type int
	// Protocol is the transport protocol.
	Protocol int
}

// Creates one socket and applies the Darwin nonblocking and close-on-exec setup.
func socket_create(input *Socket_Create_Input) (descriptor int, err error) {
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
	return syscall.SetsockoptInt(descriptor, syscall.SOL_SOCKET, SOCKET_NO_SIGPIPE, 1)
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
	return syscall.SetsockoptInt(descriptor, syscall.SOL_SOCKET, SOCKET_NO_SIGPIPE, 1)
}

// Platform scheduler is TigerBeetle Darwin IO's kqueue plus its two kernel-facing queue
// counts (third-party/tigerbeetle/src/io/darwin.zig:17-31).
type Platform_Scheduler struct {
	// Descriptor retains the kqueue instance until deinitialization.
	Descriptor int
	// IO_Backlog retains operations that wait for one-shot registration.
	IO_Backlog []*Operating_System_Operation
	// IO_Inflight keeps the kernel-owned operation census exact.
	IO_Inflight int
	// Next_Event keeps synthetic event identifiers separate from pointer values.
	Next_Event uint64
}

// Kernel event is Darwin's 64-bit struct kevent layout with integer udata. Using the UAPI
// layout avoids placing a Go pointer in the kernel while preserving TigerBeetle's completion
// correlation through kevent.udata (io/darwin.zig:158-182).
type Kernel_Event struct {
	// Ident gives the kernel the file descriptor or synthetic event identifier.
	Ident uint64
	// Filter selects the kqueue operation class.
	Filter int16
	// Flags controls one-shot registration and deletion.
	Flags uint16
	// Filter_Flags transfers operation-specific options to kqueue.
	Filter_Flags uint32
	// Data transfers a count or error value between the kernel and the operation.
	Data int64
	// User_Data returns the operation identifier without a Go pointer.
	User_Data uint64
}

// Platform initialize eagerly creates kqueue. Darwin intentionally ignores entries and flags,
// exactly as TigerBeetle IO.init does in io/darwin.zig:33-42.
func platform_initialize(entries uint16, flags uint32) (platform Platform_Scheduler, err error) {
	descriptor, create_err := syscall.Kqueue()
	if create_err != nil {
		return Platform_Scheduler{}, create_err
	}
	return Platform_Scheduler{Descriptor: descriptor}, nil
}

// Platform deinitialize releases kqueue after the owner has joined every operation.
func platform_deinitialize(state *Operating_System) {
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
	state *Operating_System, operation *Operating_System_Operation,
) (err error) {
	return platform_submit_registered(state, operation)
}

// Platform submit registered is the eager Darwin retry for an already registered operation.
func platform_submit_registered(
	state *Operating_System, operation *Operating_System_Operation,
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
	operation *Operating_System_Operation,
) (result int, again bool, err error) {
	switch operation.Kind {
	case OPERATING_SYSTEM_OPERATION_ACCEPT:
		return socket_accept(operation.Descriptor)
	case OPERATING_SYSTEM_OPERATION_CLOSE:
		return 0, false, socket_close(operation.Descriptor)
	case OPERATING_SYSTEM_OPERATION_CONNECT:
		return socket_connect_attempt(operation)
	case OPERATING_SYSTEM_OPERATION_READ:
		count, read_err := read_at(
			sharedio.File(operation.Descriptor),
			operation.Buffer,
			int64(operation.Offset),
		)
		return count, socket_again(read_err), read_err
	case OPERATING_SYSTEM_OPERATION_RECEIVE:
		return socket_receive(operation.Descriptor, operation.Buffer)
	case OPERATING_SYSTEM_OPERATION_SEND:
		count, would_block, send_err := socket_send(operation.Descriptor, operation.Buffer)
		return count, would_block, socket_send_translate(send_err)
	case OPERATING_SYSTEM_OPERATION_WRITE:
		count, write_err := write_at(
			sharedio.File(operation.Descriptor),
			operation.Buffer,
			int64(operation.Offset),
		)
		return count, socket_again(write_err), write_err
	case OPERATING_SYSTEM_OPERATION_FSYNC:
		return 0, false, syscall.Fsync(operation.Descriptor)
	case OPERATING_SYSTEM_OPERATION_OPEN_AT:
		descriptor, open_err := platform_open_at(operation)
		return descriptor, false, open_err
	default:
		return 0, false, syscall.EINVAL
	}
}

// Returns Darwin's AT_FDCWD value used by TigerBeetle IO.openat.
func platform_current_directory() (descriptor int) { return DARWIN_CURRENT_DIRECTORY }

// Executes Darwin openat synchronously when its eager completion runs, retrying EINTR and forcing
// CLOEXEC exactly as third-party/tigerbeetle/src/io/darwin.zig:492-558.
func platform_open_at(operation *Operating_System_Operation) (descriptor int, err error) {
	flags := platform_open_flags(operation.Open_Options) | syscall.O_CLOEXEC
	path := unsafe.Pointer(&operation.File_Path[0])
	for retry := true; retry; {
		result, _, errno := syscall.Syscall6(
			DARWIN_OPEN_AT_CALL,
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
	operation *Operating_System_Operation,
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
func platform_run(state *Operating_System, wait time.Moment) (err error) {
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
	events := make([]Kernel_Event, POLL_EVENTS_MAX)
	count, wait_err := kernel_kevent(&Kernel_Kevent_Input{
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
func platform_flush_submissions(state *Operating_System) (err error) { return nil }

// Platform changes removes at most one kevent pass of io_pending operations and encodes their
// integer registry identifiers in one-shot kernel changes.
func platform_changes(state *Operating_System) (changes []Kernel_Event) {
	count := len(state.Platform.IO_Backlog)
	if count > POLL_EVENTS_MAX {
		count = POLL_EVENTS_MAX
	}
	changes = make([]Kernel_Event, count)
	for index := 0; index < count; index++ {
		operation := state.Platform.IO_Backlog[index]
		operation.Backlogged = false
		changes[index] = Kernel_Event{
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
func platform_restore_changes(state *Operating_System, changes []Kernel_Event) {
	restored := make(
		[]*Operating_System_Operation,
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
func platform_filter(kind Operating_System_Operation_Kind) (filter int16) {
	if kind == OPERATING_SYSTEM_OPERATION_CONNECT {
		return syscall.EVFILT_WRITE
	}
	if kind == OPERATING_SYSTEM_OPERATION_SEND {
		return syscall.EVFILT_WRITE
	}
	if kind == OPERATING_SYSTEM_OPERATION_WRITE {
		return syscall.EVFILT_WRITE
	}
	return syscall.EVFILT_READ
}

// Platform complete events re-attempts each one-shot operation and requeues only WouldBlock.
func platform_complete_events(
	state *Operating_System, events []Kernel_Event,
) (err error) {
	for _, event := range events {
		operation := state.Operations[event.User_Data]
		if operation == nil {
			continue
		}
		operation.Kernel_Submitted = false
		if operation.Kind == OPERATING_SYSTEM_OPERATION_EVENT {
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
	state *Operating_System, operation *Operating_System_Operation,
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
	change := Kernel_Event{
		Ident: uint64(operation.Descriptor), Filter: platform_filter(operation.Kind),
		Flags: syscall.EV_DELETE,
	}
	_, delete_err := kernel_kevent(&Kernel_Kevent_Input{
		Descriptor: state.Platform.Descriptor,
		Changes:    []Kernel_Event{change},
		Wait:       POLL_FOREVER,
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
func platform_event_open(state *Operating_System) (event sharedio.Event, err error) {
	state.Platform.Next_Event++
	event = sharedio.Event(state.Platform.Next_Event)
	change := Kernel_Event{
		Ident: uint64(event), Filter: syscall.EVFILT_USER,
		Flags: syscall.EV_ADD | syscall.EV_ENABLE | syscall.EV_CLEAR,
	}
	count, open_err := kernel_kevent(&Kernel_Kevent_Input{
		Descriptor: state.Platform.Descriptor,
		Changes:    []Kernel_Event{change},
		Wait:       POLL_FOREVER,
	})
	if open_err != nil {
		return 0, open_err
	}
	invariant.Always(count == 0, "Opening an EVFILT_USER Event returns no completion.")
	return event, nil
}

// Arms an already-open persistent EVFILT_USER Event by recording one operation in flight.
func platform_event_listen(
	state *Operating_System, operation *Operating_System_Operation,
) (err error) {
	state.Platform.IO_Inflight++
	return nil
}

// Triggers EVFILT_USER with the listener's stable integer token in udata.
func platform_event_trigger(
	state *Operating_System, event sharedio.Event, identifier uint64,
) {
	change := Kernel_Event{
		Ident: uint64(event), Filter: syscall.EVFILT_USER,
		Filter_Flags: syscall.NOTE_TRIGGER,
		User_Data:    identifier,
	}
	count, trigger_err := kernel_kevent(&Kernel_Kevent_Input{
		Descriptor: state.Platform.Descriptor,
		Changes:    []Kernel_Event{change},
		Wait:       POLL_FOREVER,
	})
	invariant.Always(trigger_err == nil, "Triggering an EVFILT_USER Event succeeds.")
	invariant.Always(
		count == 0, "Triggering an EVFILT_USER Event returns no completion inline.",
	)
}

// Deletes one persistent EVFILT_USER Event after its listener has drained.
func platform_event_close(state *Operating_System, event sharedio.Event) {
	change := Kernel_Event{
		Ident: uint64(event), Filter: syscall.EVFILT_USER, Flags: syscall.EV_DELETE,
	}
	count, close_err := kernel_kevent(&Kernel_Kevent_Input{
		Descriptor: state.Platform.Descriptor,
		Changes:    []Kernel_Event{change},
		Wait:       POLL_FOREVER,
	})
	invariant.Always(close_err == nil, "Closing an EVFILT_USER Event succeeds.")
	invariant.Always(count == 0, "Closing an EVFILT_USER Event returns no completion.")
}

// Platform in flight reports work that can wake an unbounded drive.
func platform_in_flight(state *Operating_System) (in_flight bool) {
	return len(state.Platform.IO_Backlog) > 0 || state.Platform.IO_Inflight > 0
}

// Platform counts returns Darwin's exact backlog and in-flight census.
func platform_counts(state *Operating_System) (backlog int, inflight int, queued int, kernel int) {
	return len(state.Platform.IO_Backlog), state.Platform.IO_Inflight, 0, 0
}

// Kernel kevent input carries one raw kevent syscall invocation.
type Kernel_Kevent_Input struct {
	// Descriptor selects the kqueue instance for this syscall.
	Descriptor int
	// Changes gives the kernel new one-shot registrations.
	Changes []Kernel_Event
	// Events receives retired kernel registrations.
	Events []Kernel_Event
	// Wait bounds the kernel sleep.
	Wait time.Moment
}

// Kernel kevent invokes Darwin's kevent syscall with the raw integer-udata UAPI layout.
func kernel_kevent(input *Kernel_Kevent_Input) (count int, err error) {
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
