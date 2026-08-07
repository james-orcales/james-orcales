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

// DARWIN_MKDIR_AT_CALL keeps the raw syscall compatible with Darwin amd64. Go's zsysnum table
// stops before the at-family, the same gap that makes DARWIN_OPEN_AT_CALL a literal.
const DARWIN_MKDIR_AT_CALL = 475

// Caps the EINTR retries of one eager filesystem syscall. A signal can interrupt the call, but
// only a broken kernel interrupts it repeatedly, so a bound reports an error rather than a spin.
const PLATFORM_INTERRUPT_RETRIES_MAX = 16

// PLATFORM_STAT_AT_CALL is Darwin fstatat64, the variant whose struct matches syscall.Stat_t.
// Trap 469 is the legacy layout and returns fields that do not agree with syscall.Stat.
const PLATFORM_STAT_AT_CALL = 470

// PLATFORM_SYMBOLIC_LINK_NO_FOLLOW is Darwin AT_SYMLINK_NOFOLLOW, so a directory pass reports a
// symbolic link as itself rather than as its target.
const PLATFORM_SYMBOLIC_LINK_NO_FOLLOW = 0x0020

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

// Builds the child's process attributes. Setpgid puts the child in its own group so a deadline
// kills its descendants too. Darwin watches the exit by process identifier, so it needs no
// descriptor from the fork.
func process_attributes(_ *Spawn) (attributes *syscall.SysProcAttr) {
	return &syscall.SysProcAttr{Setpgid: true}
}

// Reports nothing to check after the fork: EVFILT_PROC needs only the process identifier.
func process_watch_ready(_ *Spawn) (err error) { return nil }

// Applies TigerBeetle io.buffer_limit for Darwin before a length reaches a signed kernel result.
func platform_buffer_limit(buffer []byte) (limited []byte) {
	if len(buffer) > DARWIN_BUFFER_SIZE_MAX {
		return buffer[:DARWIN_BUFFER_SIZE_MAX]
	}
	return buffer
}

// Opens one non-blocking close-on-exec socket. NOSIGPIPE is not a caller option on Darwin: a
// send to a closed peer raises SIGPIPE without it, so the platform applies it to every stream
// socket. Every caller-selected option arrives later through socket_option_set.
func socket_open(
	family sharedio.Address_Family, transport sharedio.Socket_Transport,
) (descriptor int, err error) {
	if transport == sharedio.SOCKET_TRANSPORT_UDP {
		return socket_create(&Socket_Create_Input{
			Family: family, Type: syscall.SOCK_DGRAM, Protocol: syscall.IPPROTO_UDP,
		})
	}
	descriptor, err = socket_create(&Socket_Create_Input{
		Family: family, Type: syscall.SOCK_STREAM, Protocol: syscall.IPPROTO_TCP,
	})
	if err != nil {
		return -1, err
	}
	signal_err := syscall.SetsockoptInt(
		descriptor, syscall.SOL_SOCKET, SOCKET_NO_SIGPIPE, 1,
	)
	if signal_err != nil {
		syscall.Close(descriptor)
		return -1, signal_err
	}
	return descriptor, nil
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
	close_on_exec_err := descriptor_close_on_exec(descriptor)
	if close_on_exec_err != nil {
		syscall.Close(descriptor)
		return -1, close_on_exec_err
	}
	return descriptor, nil
}

// Writes the two header bytes of a sockaddr. Darwin holds the byte count in the first byte and
// the family in the second, which is the only reason the encode is not shared whole.
func platform_address_header(
	storage *[SOCKET_ADDRESS_BYTES]byte, family int, size uint32,
) {
	storage[0] = byte(size)
	storage[1] = byte(family)
}

// Reads the family from the sockaddr the kernel wrote.
func platform_address_family(storage *[SOCKET_ADDRESS_BYTES]byte) (family int) {
	return int(storage[1])
}

// Applies one option that Darwin names differently from Linux. handled false sends the option
// on to the shared table.
func platform_option_set(
	descriptor int, option sharedio.Socket_Option, value int,
) (handled bool, err error) {
	// Darwin carries no TCP_USER_TIMEOUT. Accepting it keeps one TCP_Options profile portable,
	// and the transport still drops a dead connection through the keepalive tuple.
	if option == sharedio.SOCKET_OPTION_USER_TIMEOUT {
		return true, nil
	}
	name, known := platform_option_name(option)
	if !known {
		return false, nil
	}
	return true, syscall.SetsockoptInt(descriptor, syscall.IPPROTO_TCP, name, value)
}

// Maps the Darwin-only transport options to their names. Darwin spells the idle time
// TCP_KEEPALIVE, where Linux spells the same option TCP_KEEPIDLE.
func platform_option_name(option sharedio.Socket_Option) (name int, known bool) {
	switch option {
	case sharedio.SOCKET_OPTION_KEEPALIVE_IDLE:
		return syscall.TCP_KEEPALIVE, true
	case sharedio.SOCKET_OPTION_KEEPALIVE_INTERVAL:
		return syscall.TCP_KEEPINTVL, true
	case sharedio.SOCKET_OPTION_KEEPALIVE_COUNT:
		return syscall.TCP_KEEPCNT, true
	}
	return 0, false
}

// Applies the two Darwin accepted-socket guarantees from io/darwin.zig:324-360.
func socket_accept_configure(descriptor int) (err error) {
	close_err := descriptor_close_on_exec(descriptor)
	if close_err != nil {
		return close_err
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
	case OPERATING_SYSTEM_OPERATION_PIPE_READ:
		return pipe_read(operation.Descriptor, operation.Buffer)
	case OPERATING_SYSTEM_OPERATION_PIPE_WRITE:
		return pipe_write(operation.Descriptor, operation.Buffer)
	case OPERATING_SYSTEM_OPERATION_PROCESS_EXIT:
		// The kevent registration is the whole operation. Backlogging it hands it to
		// platform_changes, and platform_complete_events retires it when NOTE_EXIT arrives.
		return 0, true, nil
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
	case OPERATING_SYSTEM_OPERATION_MKDIR_AT:
		return 0, false, file_translate(platform_mkdir_at(operation))
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

// Executes Darwin mkdirat synchronously when its eager completion runs, retrying EINTR. kqueue
// carries no filesystem filter, so the syscall runs in the submit exactly as Open_At does.
func platform_mkdir_at(operation *Operating_System_Operation) (err error) {
	path := unsafe.Pointer(&operation.File_Path[0])
	for retry_index := 0; retry_index < PLATFORM_INTERRUPT_RETRIES_MAX; retry_index++ {
		_, _, errno := syscall.Syscall(
			DARWIN_MKDIR_AT_CALL,
			uintptr(operation.Descriptor), uintptr(path),
			uintptr(operation.Open_Options.Mode),
		)
		if errno == syscall.EINTR {
			continue
		}
		if errno != 0 {
			return errno
		}
		return nil
	}
	return syscall.EINTR
}

// Reads one pass of raw directory entries into buffer through getdirentries64, retrying EINTR.
// Go's ReadDirent reaches this kernel through fdopendir and readdir_r rather than through the
// syscall, and syscall_darwin.go:305 admits that the resulting restart is quadratic in the
// entry count. The trap keeps the resume position in the file description, so one pass needs
// no lseek, no duplicate descriptor, and no re-read of the entries already returned.
func platform_directory_read(descriptor int, buffer []byte) (count int, err error) {
	// The trap takes an in-out position, and the kernel rejects a null pointer for it.
	position := int64(0)
	for retry_index := 0; retry_index < PLATFORM_INTERRUPT_RETRIES_MAX; retry_index++ {
		result, _, errno := syscall.Syscall6(
			syscall.SYS_GETDIRENTRIES64, uintptr(descriptor),
			uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)),
			uintptr(unsafe.Pointer(&position)), 0, 0,
		)
		if errno == syscall.EINTR {
			continue
		}
		if errno != 0 {
			return 0, errno
		}
		return int(result), nil
	}
	return 0, syscall.EINTR
}

// Reports whether a directory record names a file the filesystem already removed. Darwin marks
// such a record with a zero inode, and syscall.ParseDirent drops it here for that reason.
func platform_directory_absent(record *syscall.Dirent) (absent bool) {
	return record.Ino == 0
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
	if options.Flags&sharedio.OPEN_AT_NO_FOLLOW != 0 {
		flags |= syscall.O_NOFOLLOW
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
	connect_err := socket_connect_call(operation.Descriptor, operation.Address)
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
		changes[index] = platform_change(operation)
	}
	state.Platform.IO_Backlog = state.Platform.IO_Backlog[count:]
	return changes
}

// Platform change encodes one backlogged operation as a kevent. A descriptor operation keys on
// its descriptor and a readiness filter. A process-exit operation keys on the child's process
// identifier under EVFILT_PROC instead, because there is no descriptor to watch on Darwin.
func platform_change(operation *Operating_System_Operation) (change Kernel_Event) {
	if operation.Kind == OPERATING_SYSTEM_OPERATION_PROCESS_EXIT {
		return Kernel_Event{
			Ident:  uint64(operation.Process_Identifier),
			Filter: syscall.EVFILT_PROC,
			Flags:  syscall.EV_ADD | syscall.EV_ENABLE | syscall.EV_ONESHOT,
			// NOTE_EXITSTATUS makes the kernel return the wait status in Data, but the
			// reap still has to run, so the operation reads its status from wait4.
			Filter_Flags: syscall.NOTE_EXIT,
			User_Data:    operation.Identifier,
		}
	}
	return Kernel_Event{
		Ident:     uint64(operation.Descriptor),
		Filter:    platform_filter(operation.Kind),
		Flags:     syscall.EV_ADD | syscall.EV_ENABLE | syscall.EV_ONESHOT,
		User_Data: operation.Identifier,
	}
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
		if operation.Kind == OPERATING_SYSTEM_OPERATION_PROCESS_EXIT {
			// Re-attempting would only backlog it again. NOTE_EXIT fires once.
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
	change := platform_change(operation)
	change.Flags = syscall.EV_DELETE
	change.Filter_Flags = 0
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
