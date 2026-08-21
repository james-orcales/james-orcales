//go:build darwin

package nbio

import (
	"syscall"
	"unsafe"

	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/sim/nbio"
	"local/james-orcales/shared/sim/time"
)

// SOCKET_RECEIVE_BUFFER_SIZE fix socket profile platform tests verify.
const SOCKET_RECEIVE_BUFFER_SIZE = 4 * 1024 * 1024

// SOCKET_SEND_BUFFER_SIZE fix socket profile platform tests verify.
const SOCKET_SEND_BUFFER_SIZE = 2 * 1024 * 1024

// SOCKET_NO_SIGPIPE identify SO_NOSIGPIPE, because syscall does not expose it.
const SOCKET_NO_SIGPIPE = 0x1022

// SOCKET_TCP_NOT_SENT_LOW_WATER identifies TCP_NOTSENT_LOWAT on Darwin.
const SOCKET_TCP_NOT_SENT_LOW_WATER = 0x201

// SOCKET_TCP_KEEPALIVE_IDLE keeps the portable test independent of platform option spelling.
const SOCKET_TCP_KEEPALIVE_IDLE = syscall.TCP_KEEPALIVE

// Cap one kqueue changelist and event batch to fixed flush buffer.
const POLL_EVENTS_MAX = 256

// DARWIN_OPEN_AT_CALL keep raw syscall compatible with Darwin amd64.
const DARWIN_OPEN_AT_CALL = 463

// DARWIN_MKDIR_AT_CALL keep raw syscall compatible with Darwin amd64. Go zsysnum table stop
// before at-family, same gap that make DARWIN_OPEN_AT_CALL literal.
const DARWIN_MKDIR_AT_CALL = 475

// Cap EINTR retries of one eager filesystem syscall. Signal can interrupt call, but only broken
// kernel interrupt it over and over, thus bound report error, not spin.
const PLATFORM_INTERRUPT_RETRIES_MAX = 16

// PLATFORM_STAT_AT_CALL is Darwin fstatat64, variant whose struct match syscall.Stat_t. Trap
// 469 is legacy layout and return fields that do not agree with syscall.Stat.
const PLATFORM_STAT_AT_CALL = 470

// PLATFORM_READ_LINK_CALL reads final symbolic link without libc path conversion.
const PLATFORM_READ_LINK_CALL = syscall.SYS_READLINK

// PLATFORM_SYMBOLIC_LINK_NO_FOLLOW is Darwin AT_SYMLINK_NOFOLLOW, thus directory pass report
// symbolic link as itself, not as its target.
const PLATFORM_SYMBOLIC_LINK_NO_FOLLOW = 0x0020

// DARWIN_CURRENT_DIRECTORY keep Open_At compatible with Darwin AT_FDCWD.
const DARWIN_CURRENT_DIRECTORY = -2

// DARWIN_BUFFER_SIZE_MAX stop byte count from overflow of signed kernel result.
const DARWIN_BUFFER_SIZE_MAX = 0x7fffffff

// Platform operation has no Darwin-only fields.
type Platform_Operation struct {
	// Backlogged report operation wait for its one-shot kevent registration.
	Backlogged bool
	// Kernel_Submitted report kqueue own one-shot registration.
	Kernel_Submitted bool
}

// Wire no Linux-only operation on Darwin.
func operating_system_wire_platform(state *Operating_System, loop *nbio.IO) {
	loop.Platform_IO = nbio.Platform_IO{}
}

// Build child process attributes. Setpgid put child in its own group, thus deadline kill its
// descendants too. Darwin watch exit by process identifier, thus it need no descriptor from
// fork.
func process_attributes(_ *Spawn) (attributes *syscall.SysProcAttr) {
	return &syscall.SysProcAttr{Setpgid: true}
}

// Report nothing to check after fork: EVFILT_PROC need only process identifier.
func process_watch_ready(_ *Spawn) (err error) { return nil }

// Apply buffer limit for Darwin before length reach signed kernel result.
func platform_buffer_limit(buffer []byte) (limited []byte) {
	if len(buffer) > DARWIN_BUFFER_SIZE_MAX {
		return buffer[:DARWIN_BUFFER_SIZE_MAX]
	}
	return buffer
}

// NOSIGPIPE prevents a closed peer from terminating the process, so every TCP socket gets it.
func socket_open_tcp_raw(family nbio.Address_Family) (descriptor int, err error) {
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

// UDP has no SIGPIPE path, so it needs only the common ownership flags.
func socket_open_udp_raw(family nbio.Address_Family) (descriptor int, err error) {
	return socket_create(&Socket_Create_Input{
		Family: family, Type: syscall.SOCK_DGRAM, Protocol: syscall.IPPROTO_UDP,
	})
}

// Input of socket_create.
type Socket_Create_Input struct {
	// Family is socket address family.
	Family nbio.Address_Family
	// Type is SOCK_STREAM or SOCK_DGRAM.
	Type int
	// Protocol is transport protocol.
	Protocol int
}

// Make one socket and apply Darwin nonblocking and close-on-exec setup.
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

// Write two header bytes of sockaddr. Darwin hold byte count in first byte and family in
// second. That is only reason encode is not shared whole.
func platform_address_header(
	storage *[SOCKET_ADDRESS_BYTES]byte, family int, size uint32,
) {
	storage[0] = byte(size)
	storage[1] = byte(family)
}

// Read family from sockaddr kernel wrote.
func platform_address_family(storage *[SOCKET_ADDRESS_BYTES]byte) (family int) {
	return int(storage[1])
}

// Apply receive-buffer size through the Darwin socket option.
func platform_receive_buffer_set(descriptor int, value int) (err error) {
	return syscall.SetsockoptInt(descriptor, syscall.SOL_SOCKET, syscall.SO_RCVBUF, value)
}

// Apply send-buffer size through the Darwin socket option.
func platform_send_buffer_set(descriptor int, value int) (err error) {
	return syscall.SetsockoptInt(descriptor, syscall.SOL_SOCKET, syscall.SO_SNDBUF, value)
}

// Darwin rejects a larger MSS before a route supplies the connected-path maximum.
func platform_tcp_maximum_segment_clamp(value uint32) (clamped uint32) {
	const UNCONNECTED_MAXIMUM uint32 = 512
	if value > UNCONNECTED_MAXIMUM {
		return UNCONNECTED_MAXIMUM
	}
	return value
}

// Apply the Darwin spelling for the keepalive idle period.
func platform_keepalive_idle_set(descriptor int, value int) (err error) {
	return syscall.SetsockoptInt(descriptor, syscall.IPPROTO_TCP, syscall.TCP_KEEPALIVE, value)
}

// Apply the keepalive probe interval.
func platform_keepalive_interval_set(descriptor int, value int) (err error) {
	return syscall.SetsockoptInt(descriptor, syscall.IPPROTO_TCP, syscall.TCP_KEEPINTVL, value)
}

// Apply the keepalive probe count.
func platform_keepalive_count_set(descriptor int, value int) (err error) {
	return syscall.SetsockoptInt(descriptor, syscall.IPPROTO_TCP, syscall.TCP_KEEPCNT, value)
}

// Apply two Darwin accepted-socket guarantees.
func socket_accept_configure(descriptor int) (err error) {
	close_err := descriptor_close_on_exec(descriptor)
	if close_err != nil {
		return close_err
	}
	return syscall.SetsockoptInt(descriptor, syscall.SOL_SOCKET, SOCKET_NO_SIGPIPE, 1)
}

// Platform scheduler is kqueue, plus its two kernel-facing queue counts.
type Platform_Scheduler struct {
	// Descriptor hold kqueue instance until deinitialization.
	Descriptor int
	// IO_Backlog hold operations that wait for one-shot registration.
	IO_Backlog []*Operating_System_Operation
	// IO_Inflight keep kernel-owned operation census exact.
	IO_Inflight int
	// Next_Event keep synthetic event identifiers separate from pointer values.
	Next_Event uint64
	// Changes keeps one bounded changelist inline because returning local scratch allocates.
	Changes [POLL_EVENTS_MAX]Kernel_Event
	// Events keeps one bounded result batch inline across the kernel call.
	Events [POLL_EVENTS_MAX]Kernel_Event
}

// Platform memory binds Darwin readiness backlog to caller capacity.
func platform_memory_set(
	platform *Platform_Scheduler, operations []*Operating_System_Operation,
) {
	platform.IO_Backlog = operations[:0]
}

// Kernel event is 64-bit struct kevent layout of Darwin, with integer udata. Use of UAPI layout
// avoid Go pointer in kernel while it keep completion correlation through kevent.udata.
type Kernel_Event struct {
	// Ident give kernel file descriptor, or synthetic event identifier.
	Ident uint64
	// Filter select kqueue operation class.
	Filter int16
	// Flags control one-shot registration and deletion.
	Flags uint16
	// Filter_Flags transfer operation-specific options to kqueue.
	Filter_Flags uint32
	// Data transfer count or error value between kernel and operation.
	Data int64
	// User_Data return operation identifier without Go pointer.
	User_Data uint64
}

// Platform initialize eagerly make kqueue. Darwin deliberately ignore entries and flags.
func platform_initialize(entries uint16, flags uint32) (platform Platform_Scheduler, err error) {
	descriptor, create_err := syscall.Kqueue()
	if create_err != nil {
		return Platform_Scheduler{}, create_err
	}
	return Platform_Scheduler{Descriptor: descriptor}, nil
}

// Platform deinitialize release kqueue after owner joined every operation.
func platform_deinitialize(state *Operating_System) {
	if state.Platform.Descriptor >= 0 {
		syscall.Close(state.Platform.Descriptor)
		state.Platform.Descriptor = -1
	}
}

// Platform uses kernel timeouts report that Darwin keep deadlines in its own queue and expire
// them before each kevent pass.
func platform_uses_kernel_timeouts() (uses bool) { return false }

// A storage timeout does not work on Darwin because the eager file path has no kernel timeout.
func platform_storage_deadline(
	_ *Operating_System, _ time.Duration,
) (deadline time.Monotonic_Moment) {
	return 0
}

// Platform submit eagerly attempt syscall. WouldBlock alone enter io_pending.
func platform_submit(
	state *Operating_System, operation *Operating_System_Operation,
) (err error) {
	return platform_submit_registered(state, operation)
}

// Platform submit registered is eager Darwin retry of already registered operation.
func platform_submit_registered(
	state *Operating_System, operation *Operating_System_Operation,
) (err error) {
	result, again, operation_err := operating_system_operation_do(operation)
	if again {
		operation.Backlogged = true
		platform_backlog_add(state, operation)
		return nil
	}
	operating_system_operation_complete(state, operation, result, operation_err)
	return nil
}

func platform_backlog_add(
	state *Operating_System, operation *Operating_System_Operation,
) {
	aver.Always(len(state.Platform.IO_Backlog) < cap(state.Platform.IO_Backlog),
		"The caller-owned Darwin backlog has capacity before readiness wait.")
	state.Platform.IO_Backlog = append(state.Platform.IO_Backlog, operation)
}

// Operating system operation do run one Darwin operation at completion time. Connect is
// attempted once. After kqueue requeue it, SO_ERROR is checked instead of second connect, thus
// no EISCONN.
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
			nbio.File(operation.Descriptor),
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
		// Kevent registration is whole operation. Backlog of it hand it to
		// platform_changes, and platform_complete_events retire it when NOTE_EXIT arrive.
		return 0, true, nil
	case OPERATING_SYSTEM_OPERATION_SEND:
		count, would_block, send_err := socket_send(operation.Descriptor, operation.Buffer)
		return count, would_block, socket_send_translate(send_err)
	case OPERATING_SYSTEM_OPERATION_WRITE:
		count, write_err := write_at(
			nbio.File(operation.Descriptor),
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

// Return Darwin AT_FDCWD value openat use.
func platform_current_directory() (descriptor int) { return DARWIN_CURRENT_DIRECTORY }

// Run Darwin openat synchronously when its eager completion run, retrying EINTR and forcing
// CLOEXEC.
func platform_open_at(operation *Operating_System_Operation) (descriptor int, err error) {
	flags := platform_open_flags(operation.Open_Options) | syscall.O_CLOEXEC
	path := unsafe.Pointer(&operation.File_Path[0])
	for retry := true; retry; {
		result, _, errno := syscall.Syscall6(
			DARWIN_OPEN_AT_CALL,
			uintptr(operation.Descriptor), uintptr(path), uintptr(flags),
			uintptr(nbio.File_Permissions_To_POSIX(operation.Open_Options.Permissions)),
			0, 0,
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

// Run Darwin mkdirat synchronously when its eager completion run, retrying EINTR. kqueue carry
// no filesystem filter, thus syscall run in submit exactly as Open_At do.
func platform_mkdir_at(operation *Operating_System_Operation) (err error) {
	path := unsafe.Pointer(&operation.File_Path[0])
	for retry_index := 0; retry_index < PLATFORM_INTERRUPT_RETRIES_MAX; retry_index++ {
		_, _, errno := syscall.Syscall(
			DARWIN_MKDIR_AT_CALL,
			uintptr(operation.Descriptor), uintptr(path),
			uintptr(nbio.File_Permissions_To_POSIX(operation.Open_Options.Permissions)),
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

// Read one pass of raw directory entries into buffer through getdirentries64, retrying EINTR. Go
// ReadDirent reach this kernel through fdopendir and readdir_r, not through syscall, and
// syscall_darwin.go:305 admit that resulting restart is quadratic in entry count. Trap keep
// resume position in file description, thus one pass need no lseek, no duplicate descriptor, and
// no re-read of entries already returned.
func platform_directory_read(descriptor int, buffer []byte) (count int, err error) {
	// Trap take in-out position, and kernel reject null pointer for it.
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

// Report whether directory record name file filesystem already removed. Darwin mark such record
// with zero inode, and syscall.ParseDirent drop it here for that reason.
func platform_directory_absent(record *syscall.Dirent) (absent bool) {
	return record.Ino == 0
}

// Translate portable Open_At option fields to Darwin posix.O bits.
func platform_open_flags(options nbio.Open_At_Options) (flags int) {
	flags = syscall.O_RDONLY
	if options.Access == nbio.OPEN_WRITE_ONLY {
		flags = syscall.O_WRONLY
	}
	if options.Access == nbio.OPEN_READ_WRITE {
		flags = syscall.O_RDWR
	}
	if options.Create {
		flags |= syscall.O_CREAT
	}
	if options.Truncate {
		flags |= syscall.O_TRUNC
	}
	if options.Flags&nbio.OPEN_AT_NO_FOLLOW != 0 {
		flags |= syscall.O_NOFOLLOW
	}
	return flags
}

// Socket connect attempt do first connect, or post-readiness SO_ERROR check.
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

// Platform run port Darwin flush pass: io_pending become one-shot changes, kevent return
// identifiers into completed work, and common completed queue drain callbacks later.
func platform_run(state *Operating_System, wait time.Monotonic_Moment) (err error) {
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
	events := state.Platform.Events[:]
	count, wait_err := kernel_kevent(&Kernel_Kevent_Input{
		Descriptor: state.Platform.Descriptor,
		Changes:    changes,
		Events:     events,
		Wait:       kernel_wait,
	})
	if wait_err != nil {
		return wait_err
	}
	for _, change := range changes {
		operation := operating_system_operation_find(state, change.User_Data)
		if operation != nil {
			operation.Backlogged = false
			operation.Kernel_Submitted = true
		}
	}
	platform_changes_remove(state, len(changes))
	state.Platform.IO_Inflight += len(changes)
	operation_events := 0
	for index := 0; index < count; index++ {
		if operating_system_operation_find(state, events[index].User_Data) != nil {
			operation_events++
		}
	}
	state.Platform.IO_Inflight -= operation_events
	return platform_complete_events(state, events[:count])
}

// Darwin eagerly attempt callback submission in live completed drain. WouldBlock work stay in
// io_pending until next flush, thus Linux post-callback SQ flush has no Darwin counterpart.
func platform_flush_submissions(state *Operating_System) (err error) { return nil }

// Platform changes remove at most one kevent pass of io_pending operations and encode their
// integer registry identifiers in one-shot kernel changes.
func platform_changes(state *Operating_System) (changes []Kernel_Event) {
	count := len(state.Platform.IO_Backlog)
	if count > POLL_EVENTS_MAX {
		count = POLL_EVENTS_MAX
	}
	changes = state.Platform.Changes[:count]
	for index := 0; index < count; index++ {
		operation := state.Platform.IO_Backlog[index]
		changes[index] = platform_change(operation)
	}
	return changes
}

func platform_changes_remove(state *Operating_System, count int) {
	backlog := state.Platform.IO_Backlog
	copy(backlog, backlog[count:])
	count_after := len(backlog) - count
	for index := count_after; index < len(backlog); index++ {
		backlog[index] = nil
	}
	state.Platform.IO_Backlog = backlog[:count_after]
}

// Platform change encode one backlogged operation as kevent. Descriptor operation key on its
// descriptor and readiness filter. Process-exit operation key on child process identifier under
// EVFILT_PROC instead, because no descriptor to watch on Darwin.
func platform_change(operation *Operating_System_Operation) (change Kernel_Event) {
	if operation.Kind == OPERATING_SYSTEM_OPERATION_PROCESS_EXIT {
		return Kernel_Event{
			Ident:  uint64(operation.Process_Identifier),
			Filter: syscall.EVFILT_PROC,
			Flags:  syscall.EV_ADD | syscall.EV_ENABLE | syscall.EV_ONESHOT,
			// NOTE_EXITSTATUS make kernel return wait status in Data, but reap still
			// has to run, thus operation read its status from wait4.
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

// Platform filter map each operation to read or write kqueue filter.
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

// Platform complete events re-attempt each one-shot operation and requeue only WouldBlock.
func platform_complete_events(
	state *Operating_System, events []Kernel_Event,
) (err error) {
	for _, event := range events {
		operation := operating_system_operation_find(state, event.User_Data)
		if operation == nil {
			continue
		}
		operation.Kernel_Submitted = false
		if operation.Kind == OPERATING_SYSTEM_OPERATION_EVENT {
			operating_system_operation_complete(state, operation, 0, nil)
			continue
		}
		if operation.Kind == OPERATING_SYSTEM_OPERATION_PROCESS_EXIT {
			// Re-attempt would only backlog it again. NOTE_EXIT fire once.
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

// Platform expire operation removes one bounded socket request from backlog or kqueue before its
// timeout callback can make descriptor teardown safe.
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
	aver.Always(state.Platform.IO_Inflight > 0,
		"An expired kqueue operation was counted in flight.")
	state.Platform.IO_Inflight--
	return nil
}

// Open persistent EVFILT_USER Event.
func platform_event_open(state *Operating_System) (event nbio.Event, err error) {
	state.Platform.Next_Event++
	event = nbio.Event(state.Platform.Next_Event)
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
	aver.Always(count == 0, "Opening an EVFILT_USER Event returns no completion.")
	return event, nil
}

// Arm already-open persistent EVFILT_USER Event by record of one operation in flight.
func platform_event_listen(
	state *Operating_System, operation *Operating_System_Operation,
) (err error) {
	state.Platform.IO_Inflight++
	return nil
}

// Trigger EVFILT_USER with stable integer token of listener in udata.
func platform_event_trigger(
	state *Operating_System, event nbio.Event, identifier uint64,
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
	aver.Always(trigger_err == nil, "Triggering an EVFILT_USER Event succeeds.")
	aver.Always(
		count == 0, "Triggering an EVFILT_USER Event returns no completion inline.",
	)
}

// Delete one persistent EVFILT_USER Event after its listener drained.
func platform_event_close(state *Operating_System, event nbio.Event) {
	change := Kernel_Event{
		Ident: uint64(event), Filter: syscall.EVFILT_USER, Flags: syscall.EV_DELETE,
	}
	count, close_err := kernel_kevent(&Kernel_Kevent_Input{
		Descriptor: state.Platform.Descriptor,
		Changes:    []Kernel_Event{change},
		Wait:       POLL_FOREVER,
	})
	aver.Always(close_err == nil, "Closing an EVFILT_USER Event succeeds.")
	aver.Always(count == 0, "Closing an EVFILT_USER Event returns no completion.")
}

// Platform in flight report work that can wake unbounded drive.
func platform_in_flight(state *Operating_System) (in_flight bool) {
	return len(state.Platform.IO_Backlog) > 0 || state.Platform.IO_Inflight > 0
}

// Platform counts return exact Darwin backlog and in-flight census.
func platform_counts(state *Operating_System) (backlog int, inflight int, queued int, kernel int) {
	return len(state.Platform.IO_Backlog), state.Platform.IO_Inflight, 0, 0
}

// Kernel kevent input carry one raw kevent syscall invocation.
type Kernel_Kevent_Input struct {
	// Descriptor select kqueue instance of this syscall.
	Descriptor int
	// Changes give kernel new one-shot registrations.
	Changes []Kernel_Event
	// Events receive retired kernel registrations.
	Events []Kernel_Event
	// Wait bound kernel sleep.
	Wait time.Monotonic_Moment
}

// Kernel kevent invoke Darwin kevent syscall with raw integer-udata UAPI layout.
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
	// Retry EINTR here instead of expose of it to driver: signal that interrupt kevent say
	// nothing about work in queue, thus driver has nothing to do with it.
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
