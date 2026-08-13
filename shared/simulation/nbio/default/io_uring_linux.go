//go:build linux && amd64

package nbio

import (
	"encoding/binary"
	"errors"
	"sync/atomic"
	"syscall"
	"unsafe"

	invariant "local/james-orcales/shared/invariant/default"
	sharedio "local/james-orcales/shared/simulation/nbio"
	"local/james-orcales/shared/simulation/time"
)

// SOCKET_RECEIVE_BUFFER_SIZE fixes the socket profile that the platform tests verify.
const SOCKET_RECEIVE_BUFFER_SIZE = 4 * 1024 * 1024

// SOCKET_SEND_BUFFER_SIZE fixes the socket profile that the platform tests verify.
const SOCKET_SEND_BUFFER_SIZE = 2 * 1024 * 1024

// SOCKET_KEEPALIVE_IDLE_SECONDS fixes the keepalive profile that the platform tests verify.
const SOCKET_KEEPALIVE_IDLE_SECONDS = 5

// SOCKET_KEEPALIVE_INTERVAL_SECONDS fixes the keepalive profile that the platform tests verify.
const SOCKET_KEEPALIVE_INTERVAL_SECONDS = 4

// SOCKET_KEEPALIVE_COUNT fixes the keepalive profile that the platform tests verify.
const SOCKET_KEEPALIVE_COUNT = 3

// SOCKET_USER_TIMEOUT_MILLISECONDS fixes the timeout profile that the platform tests verify.
const SOCKET_USER_TIMEOUT_MILLISECONDS = 17 * 1000

// SOCKET_RECEIVE_BUFFER_FORCE identifies SO_RCVBUFFORCE because syscall does not expose it.
const SOCKET_RECEIVE_BUFFER_FORCE = 33

// SOCKET_SEND_BUFFER_FORCE identifies SO_SNDBUFFORCE because syscall does not expose it.
const SOCKET_SEND_BUFFER_FORCE = 32

// SOCKET_USER_TIMEOUT identifies TCP_USER_TIMEOUT because syscall does not expose it.
const SOCKET_USER_TIMEOUT = 18

// LINUX_BUFFER_SIZE_MAX prevents a byte count from overflowing a signed kernel result.
const LINUX_BUFFER_SIZE_MAX = 0x7ffff000

// Platform_Operation keeps Linux statx arguments stable until io_uring retires the operation.
type Platform_Operation struct {
	// Statx_Result receives Linux statx output from io_uring.
	Statx_Result *sharedio.Statx
	// Statx_Flags are forwarded to Linux statx.
	Statx_Flags uint32
	// Statx_Mask selects the Linux statx fields to return.
	Statx_Mask uint32
}

// Builds the child's process attributes. Setpgid puts the child in its own group so a deadline
// kills its descendants too. PidFD asks the kernel for a descriptor naming the child, created
// atomically at the fork, so the exit watch never names a process identifier that could be
// reused.
func process_attributes(spawn *Spawn) (attributes *syscall.SysProcAttr) {
	return &syscall.SysProcAttr{Setpgid: true, PidFD: &spawn.Exit_Descriptor}
}

// Reports whether the kernel supplied the pidfd. StartProcess leaves it negative below Linux
// 5.3, and the exit watch has nothing to poll without it, so the spawn fails loudly here
// rather than hanging.
func process_watch_ready(spawn *Spawn) (err error) {
	if spawn.Exit_Descriptor < 0 {
		return syscall.ENOSYS
	}
	return nil
}

// Wires Linux IORING_OP_STATX, the one operation absent from TigerBeetle's Darwin surface.
func operating_system_wire_platform(state *Operating_System, loop *sharedio.IO) {
	loop.Statx = func(
		completion *time.Completion, callback time.Timeout_Callback,
		directory sharedio.File, file_path string, flags uint32, mask uint32,
		result *sharedio.Statx,
	) {
		operating_system_submit(completion)
		descriptor := int(directory)
		if directory == sharedio.DIRECTORY_CURRENT {
			descriptor = platform_current_directory()
		}
		operation := &Operating_System_Operation{
			Completion: completion,
			Kind:       OPERATING_SYSTEM_OPERATION_STATX,
			Descriptor: descriptor,
			File_Path:  append([]byte(file_path), 0),
			Platform_Operation: Platform_Operation{
				Statx_Result: result,
				Statx_Flags:  flags,
				Statx_Mask:   mask,
			},
			Deliver: func(count int, err error) { callback(completion, err) },
		}
		operating_system_operation_submit(state, operation)
	}
}

// Applies TigerBeetle io.buffer_limit for Linux before a length reaches a signed kernel result.
func platform_buffer_limit(buffer []byte) (limited []byte) {
	if len(buffer) > LINUX_BUFFER_SIZE_MAX {
		return buffer[:LINUX_BUFFER_SIZE_MAX]
	}
	return buffer
}

// Opens one non-blocking close-on-exec socket. Linux carries both flags in the socket type, so
// this is one syscall. Every caller-selected option arrives later through socket_option_set.
func socket_open(
	family sharedio.Address_Family, transport sharedio.Socket_Transport,
) (descriptor int, err error) {
	if transport == sharedio.SOCKET_TRANSPORT_UDP {
		return syscall.Socket(
			socket_family(family),
			syscall.SOCK_DGRAM|syscall.SOCK_NONBLOCK|syscall.SOCK_CLOEXEC,
			syscall.IPPROTO_UDP,
		)
	}
	return syscall.Socket(
		socket_family(family),
		syscall.SOCK_STREAM|syscall.SOCK_NONBLOCK|syscall.SOCK_CLOEXEC,
		syscall.IPPROTO_TCP,
	)
}

// Applies CLOEXEC to the synchronous accept helper; io_uring accept supplies it in the SQE.
func socket_accept_configure(descriptor int) (err error) {
	syscall.CloseOnExec(descriptor)
	return nil
}

// Applies one option that Linux names differently from Darwin, or that Linux offers in a
// privileged variant. handled false sends the option on to the shared table.
func platform_option_set(
	descriptor int, option sharedio.Socket_Option, value int,
) (handled bool, err error) {
	if option == sharedio.SOCKET_OPTION_RECEIVE_BUFFER {
		return true, socket_buffer_force(
			descriptor, SOCKET_RECEIVE_BUFFER_FORCE, syscall.SO_RCVBUF, value)
	}
	if option == sharedio.SOCKET_OPTION_SEND_BUFFER {
		return true, socket_buffer_force(
			descriptor, SOCKET_SEND_BUFFER_FORCE, syscall.SO_SNDBUF, value)
	}
	name, known := platform_option_name(option)
	if !known {
		return false, nil
	}
	return true, syscall.SetsockoptInt(descriptor, syscall.IPPROTO_TCP, name, value)
}

// Maps the Linux-only transport options to their names.
func platform_option_name(option sharedio.Socket_Option) (name int, known bool) {
	switch option {
	case sharedio.SOCKET_OPTION_KEEPALIVE_IDLE:
		return syscall.TCP_KEEPIDLE, true
	case sharedio.SOCKET_OPTION_KEEPALIVE_INTERVAL:
		return syscall.TCP_KEEPINTVL, true
	case sharedio.SOCKET_OPTION_KEEPALIVE_COUNT:
		return syscall.TCP_KEEPCNT, true
	case sharedio.SOCKET_OPTION_USER_TIMEOUT:
		return SOCKET_USER_TIMEOUT, true
	}
	return 0, false
}

// Sizes a socket buffer through the privileged name first, because the unprivileged one silently
// caps the request at the kernel's rmem_max or wmem_max. A process without CAP_NET_ADMIN takes
// the capped size rather than failing, which is what TigerBeetle accepts on this platform.
func socket_buffer_force(descriptor int, forced int, plain int, value int) (err error) {
	err = syscall.SetsockoptInt(descriptor, syscall.SOL_SOCKET, forced, value)
	if err == syscall.EPERM {
		return syscall.SetsockoptInt(descriptor, syscall.SOL_SOCKET, plain, value)
	}
	return err
}

// KERNEL_RING_SETUP_CALL keeps the raw syscall compatible with Linux amd64.
const KERNEL_RING_SETUP_CALL = 425

// KERNEL_RING_ENTER_CALL keeps the raw syscall compatible with Linux amd64.
const KERNEL_RING_ENTER_CALL = 426

// KERNEL_RING_SUBMISSION_OFFSET keeps mmap compatible with the Linux UAPI.
const KERNEL_RING_SUBMISSION_OFFSET = 0

// KERNEL_RING_COMPLETION_OFFSET keeps mmap compatible with the Linux UAPI.
const KERNEL_RING_COMPLETION_OFFSET = 0x08000000

// KERNEL_RING_ENTRIES_OFFSET keeps mmap compatible with the Linux UAPI.
const KERNEL_RING_ENTRIES_OFFSET = 0x10000000

// KERNEL_RING_FEATURE_SINGLE_MAPPING selects the Linux shared-ring feature bit.
const KERNEL_RING_FEATURE_SINGLE_MAPPING = 1

// KERNEL_RING_FEATURE_EXTENDED_ARGUMENT selects the required Linux timeout feature bit.
const KERNEL_RING_FEATURE_EXTENDED_ARGUMENT = 1 << 8

// KERNEL_RING_ENTER_GET_EVENTS makes io_uring_enter wait for completions.
const KERNEL_RING_ENTER_GET_EVENTS = 1

// KERNEL_RING_ENTER_SUBMISSION_WAKEUP wakes a sleeping submission-poll thread.
const KERNEL_RING_ENTER_SUBMISSION_WAKEUP = 1 << 1

// KERNEL_RING_ENTER_EXTENDED_ARGUMENT permits a bounded wait through io_uring_enter.
const KERNEL_RING_ENTER_EXTENDED_ARGUMENT = 1 << 3

// KERNEL_RING_SETUP_IO_POLL identifies the kernel I/O-poll setup mode.
const KERNEL_RING_SETUP_IO_POLL = 1

// KERNEL_RING_SETUP_SUBMISSION_POLL identifies the kernel submission-poll setup mode.
const KERNEL_RING_SETUP_SUBMISSION_POLL = 1 << 1

// KERNEL_RING_SUBMISSION_NEEDS_WAKEUP reports a sleeping submission-poll thread.
const KERNEL_RING_SUBMISSION_NEEDS_WAKEUP = 1

// KERNEL_RING_OPERATION_FSYNC keeps the SQE opcode compatible with the Linux UAPI.
const KERNEL_RING_OPERATION_FSYNC = 3

// KERNEL_RING_OPERATION_POLL_ADD keeps the SQE opcode compatible with the Linux UAPI.
const KERNEL_RING_OPERATION_POLL_ADD = 6

// KERNEL_RING_OPERATION_MKDIR_AT keeps the SQE opcode compatible with the Linux UAPI. The kernel
// added it in 5.15.
const KERNEL_RING_OPERATION_MKDIR_AT = 37

// PLATFORM_STAT_AT_CALL is Linux newfstatat, whose struct matches syscall.Stat_t on amd64.
const PLATFORM_STAT_AT_CALL = 262

// PLATFORM_SYMBOLIC_LINK_NO_FOLLOW is Linux AT_SYMLINK_NOFOLLOW, so a directory pass reports a
// symbolic link as itself rather than as its target.
const PLATFORM_SYMBOLIC_LINK_NO_FOLLOW = 0x100

// Caps the EINTR retries of one eager filesystem syscall. A signal can interrupt the call, but
// only a broken kernel interrupts it repeatedly, so a bound reports an error rather than a spin.
const PLATFORM_INTERRUPT_RETRIES_MAX = 16

// KERNEL_PIPE_OFFSET tells the kernel to read or write at the descriptor's current position.
// A pipe is not seekable, so it rejects any other offset with ESPIPE.
const KERNEL_PIPE_OFFSET = ^uint64(0)

// KERNEL_POLL_INPUT is POLLIN. A pidfd reports it once its process has exited.
const KERNEL_POLL_INPUT = 0x001

// KERNEL_RING_OPERATION_TIMEOUT keeps the SQE opcode compatible with the Linux UAPI.
const KERNEL_RING_OPERATION_TIMEOUT = 11

// KERNEL_RING_OPERATION_ACCEPT keeps the SQE opcode compatible with the Linux UAPI.
const KERNEL_RING_OPERATION_ACCEPT = 13

// KERNEL_RING_OPERATION_LINK_TIMEOUT keeps the SQE opcode compatible with the Linux UAPI.
const KERNEL_RING_OPERATION_LINK_TIMEOUT = 15

// KERNEL_RING_OPERATION_CONNECT keeps the SQE opcode compatible with the Linux UAPI.
const KERNEL_RING_OPERATION_CONNECT = 16

// KERNEL_RING_OPERATION_OPEN_AT keeps the SQE opcode compatible with the Linux UAPI.
const KERNEL_RING_OPERATION_OPEN_AT = 18

// KERNEL_RING_OPERATION_CLOSE keeps the SQE opcode compatible with the Linux UAPI.
const KERNEL_RING_OPERATION_CLOSE = 19

// KERNEL_RING_OPERATION_STATX keeps the SQE opcode compatible with the Linux UAPI.
const KERNEL_RING_OPERATION_STATX = 21

// KERNEL_RING_OPERATION_READ keeps the SQE opcode compatible with the Linux UAPI.
const KERNEL_RING_OPERATION_READ = 22

// KERNEL_RING_OPERATION_WRITE keeps the SQE opcode compatible with the Linux UAPI.
const KERNEL_RING_OPERATION_WRITE = 23

// KERNEL_RING_OPERATION_SEND keeps the SQE opcode compatible with the Linux UAPI.
const KERNEL_RING_OPERATION_SEND = 26

// KERNEL_RING_OPERATION_RECEIVE keeps the SQE opcode compatible with the Linux UAPI.
const KERNEL_RING_OPERATION_RECEIVE = 27

// KERNEL_RING_SUBMISSION_LINK joins an operation to its internal deadline.
const KERNEL_RING_SUBMISSION_LINK = 1 << 2

// KERNEL_MESSAGE_NO_SIGNAL prevents SIGPIPE during a socket send.
const KERNEL_MESSAGE_NO_SIGNAL = 0x4000

// KERNEL_RING_RESERVED_VALUES keeps Kernel_Ring_Parameters equal to the Linux UAPI layout.
const KERNEL_RING_RESERVED_VALUES = 3

// EVENTFD_VALUE_BYTES keeps each eventfd notification equal to one uint64.
const EVENTFD_VALUE_BYTES = 8

// KERNEL_COMPLETIONS_BATCH bounds stack use while the code drains all available completions.
const KERNEL_COMPLETIONS_BATCH = 256

// Kernel ring offsets is the stable Linux io_uring submission-ring offset layout.
type Kernel_Ring_Offsets struct {
	// Head preserves the kernel consumer cursor position.
	Head uint32
	// Tail preserves the application producer cursor position.
	Tail uint32
	// Ring_Mask permits the UAPI ring-index calculation.
	Ring_Mask uint32
	// Ring_Entries preserves the UAPI ring capacity.
	Ring_Entries uint32
	// Flags exposes the submission-ring state that controls wakeups.
	Flags uint32
	// Dropped preserves the UAPI counter position.
	Dropped uint32
	// Array locates the submission index array.
	Array uint32
	// Reserved preserves the Linux UAPI layout.
	Reserved uint32
	// Reserved_Two preserves the Linux UAPI layout.
	Reserved_Two uint64
}

// Kernel completion offsets is the stable Linux io_uring completion-ring offset layout.
type Kernel_Completion_Offsets struct {
	// Head preserves the application consumer cursor position.
	Head uint32
	// Tail preserves the kernel producer cursor position.
	Tail uint32
	// Ring_Mask permits the UAPI ring-index calculation.
	Ring_Mask uint32
	// Ring_Entries preserves the UAPI ring capacity.
	Ring_Entries uint32
	// Overflow preserves the UAPI overflow counter position.
	Overflow uint32
	// Completions locates the completion entries.
	Completions uint32
	// Flags preserves the Linux UAPI layout.
	Flags uint32
	// Reserved preserves the Linux UAPI layout.
	Reserved uint32
	// Reserved_Two preserves the Linux UAPI layout.
	Reserved_Two uint64
}

// Kernel ring parameters is struct io_uring_params from Linux's UAPI.
type Kernel_Ring_Parameters struct {
	// Submission_Entries permits allocation of the submission ring.
	Submission_Entries uint32
	// Completion_Entries permits allocation of the completion ring.
	Completion_Entries uint32
	// Flags sends the requested setup modes to the kernel.
	Flags uint32
	// Worker_CPU preserves the Linux UAPI layout.
	Worker_CPU uint32
	// Worker_Idle preserves the Linux UAPI layout.
	Worker_Idle uint32
	// Features reports the kernel capabilities that this backend requires.
	Features uint32
	// Worker_Descriptor preserves the Linux UAPI layout.
	Worker_Descriptor uint32
	// Reserved preserves the Linux UAPI layout.
	Reserved [KERNEL_RING_RESERVED_VALUES]uint32
	// Submission locates each submission-ring field.
	Submission Kernel_Ring_Offsets
	// Completion locates each completion-ring field.
	Completion Kernel_Completion_Offsets
}

// Kernel submission entry is Linux's 64-byte io_uring SQE. The operation-specific union fields
// retain their UAPI offsets under the generic names used here.
type Kernel_Submission_Entry struct {
	// Opcode selects the kernel operation.
	Opcode uint8
	// Flags links an operation to its internal deadline.
	Flags uint8
	// Priority preserves the Linux UAPI layout.
	Priority uint16
	// Descriptor transfers the target file descriptor to the kernel.
	Descriptor int32
	// Offset transfers the file position or secondary pointer to the kernel.
	Offset uint64
	// Address transfers the primary operation pointer to the kernel.
	Address uint64
	// Count transfers the buffer length or operation mask to the kernel.
	Count uint32
	// Operation_Flags transfers operation-specific options to the kernel.
	Operation_Flags uint32
	// User_Data returns the operation identifier through the completion entry.
	User_Data uint64
	// Buffer_Index preserves the Linux UAPI layout.
	Buffer_Index uint16
	// Personality preserves the Linux UAPI layout.
	Personality uint16
	// Splice_Input preserves the Linux UAPI layout.
	Splice_Input int32
	// Address_Three preserves the Linux UAPI layout.
	Address_Three uint64
	// Padding preserves the 64-byte Linux UAPI layout.
	Padding uint64
}

// Kernel completion entry is Linux's 16-byte io_uring CQE.
type Kernel_Completion_Entry struct {
	// User_Data returns the submitted operation identifier.
	User_Data uint64
	// Result returns the operation count or negative errno.
	Result int32
	// Flags preserves the Linux UAPI layout.
	Flags uint32
}

// Kernel enter argument is io_uring_getevents_arg for IORING_ENTER_EXT_ARG.
type Kernel_Enter_Argument struct {
	// Signal_Mask preserves the Linux UAPI layout when this backend supplies no mask.
	Signal_Mask uint64
	// Signal_Mask_Size tells the kernel that Signal_Mask is absent.
	Signal_Mask_Size uint32
	// Padding preserves the Linux UAPI layout.
	Padding uint32
	// Timespec gives io_uring_enter a bounded wait.
	Timespec uint64
}

// Platform scheduler is TigerBeetle Linux IO's io_uring plus exact queue counters.
type Platform_Scheduler struct {
	// Descriptor retains the io_uring instance until deinitialization.
	Descriptor int
	// Parameters retain the kernel offsets that interpret each mapping.
	Parameters Kernel_Ring_Parameters
	// Submission_Ring retains the submission mapping until deinitialization.
	Submission_Ring []byte
	// Completion_Ring retains the completion mapping until deinitialization.
	Completion_Ring []byte
	// Entries retains the SQE mapping until deinitialization.
	Entries []byte
	// Single_Mapping prevents the code from releasing one shared mapping twice.
	Single_Mapping bool
	// Submission_Head tracks reserved SQEs before publication.
	Submission_Head uint32
	// Submission_Tail tracks published SQEs before kernel submission.
	Submission_Tail uint32
	// IO_Queued keeps the queue census exact before publication.
	IO_Queued int
	// IO_Published keeps the queue census exact before kernel submission.
	IO_Published int
	// IO_In_Kernel keeps the queue census exact before completion.
	IO_In_Kernel int
	// Retry_Backlog prevents a retry from reentering the CQE drain.
	Retry_Backlog []*Operating_System_Operation
}

// Platform initialize creates io_uring eagerly and rejects kernels without EXT_ARG, matching
// third-party/tigerbeetle/src/io/linux.zig:37-68.
func platform_initialize(entries uint16, flags uint32) (platform Platform_Scheduler, err error) {
	parameters := Kernel_Ring_Parameters{Flags: flags}
	result, _, errno := syscall.Syscall(
		KERNEL_RING_SETUP_CALL, uintptr(entries), uintptr(unsafe.Pointer(&parameters)), 0,
	)
	if errno != 0 {
		return Platform_Scheduler{}, errno
	}
	descriptor := int(result)
	if parameters.Features&KERNEL_RING_FEATURE_EXTENDED_ARGUMENT == 0 {
		syscall.Close(descriptor)
		return Platform_Scheduler{}, errors.New(
			"io: Linux kernel 5.11 or newer with IORING_FEAT_EXT_ARG is required",
		)
	}
	platform = Platform_Scheduler{Descriptor: descriptor, Parameters: parameters}
	map_err := platform_map(&platform)
	if map_err != nil {
		platform_unmap(&platform)
		syscall.Close(descriptor)
		return Platform_Scheduler{}, map_err
	}
	return platform, nil
}

// Platform map maps the submission ring, completion ring, and SQE array described by setup.
func platform_map(platform *Platform_Scheduler) (err error) {
	submission_size := int(platform.Parameters.Submission.Array) +
		int(platform.Parameters.Submission_Entries)*4
	completion_size := int(platform.Parameters.Completion.Completions) +
		int(platform.Parameters.Completion_Entries)*16
	features := platform.Parameters.Features
	platform.Single_Mapping = features&KERNEL_RING_FEATURE_SINGLE_MAPPING != 0
	if platform.Single_Mapping {
		mapping_size := submission_size
		if completion_size > mapping_size {
			mapping_size = completion_size
		}
		mapping, map_err := platform_mmap(&Platform_Mmap_Input{
			Descriptor: platform.Descriptor,
			Offset:     KERNEL_RING_SUBMISSION_OFFSET,
			Size:       mapping_size,
		})
		if map_err != nil {
			return map_err
		}
		platform.Submission_Ring = mapping
		platform.Completion_Ring = mapping
	} else {
		submission, map_err := platform_mmap(&Platform_Mmap_Input{
			Descriptor: platform.Descriptor,
			Offset:     KERNEL_RING_SUBMISSION_OFFSET,
			Size:       submission_size,
		})
		if map_err != nil {
			return map_err
		}
		platform.Submission_Ring = submission
		completion, completion_err := platform_mmap(&Platform_Mmap_Input{
			Descriptor: platform.Descriptor,
			Offset:     KERNEL_RING_COMPLETION_OFFSET,
			Size:       completion_size,
		})
		if completion_err != nil {
			return completion_err
		}
		platform.Completion_Ring = completion
	}
	entries_size := int(platform.Parameters.Submission_Entries) * 64
	entries, entries_err := platform_mmap(&Platform_Mmap_Input{
		Descriptor: platform.Descriptor,
		Offset:     KERNEL_RING_ENTRIES_OFFSET,
		Size:       entries_size,
	})
	if entries_err != nil {
		return entries_err
	}
	platform.Entries = entries
	return nil
}

// Platform mmap input identifies one io_uring memory mapping.
type Platform_Mmap_Input struct {
	// Descriptor selects the io_uring instance that owns the mapping.
	Descriptor int
	// Offset selects the required io_uring memory region.
	Offset int64
	// Size bounds the mapping to the kernel-reported region size.
	Size int
}

// Platform mmap maps one io_uring region as shared read-write memory.
func platform_mmap(input *Platform_Mmap_Input) (memory []byte, err error) {
	return syscall.Mmap(input.Descriptor, input.Offset, input.Size,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED|syscall.MAP_POPULATE)
}

// Platform unmap releases every successfully mapped io_uring region exactly once.
func platform_unmap(platform *Platform_Scheduler) {
	if len(platform.Entries) > 0 {
		syscall.Munmap(platform.Entries)
		platform.Entries = nil
	}
	if len(platform.Submission_Ring) > 0 {
		syscall.Munmap(platform.Submission_Ring)
		platform.Submission_Ring = nil
	}
	if !platform.Single_Mapping {
		if len(platform.Completion_Ring) > 0 {
			syscall.Munmap(platform.Completion_Ring)
		}
	}
	platform.Completion_Ring = nil
}

// Platform deinitialize releases io_uring after every registered operation has completed.
func platform_deinitialize(state *Operating_System) {
	platform_unmap(&state.Platform)
	if state.Platform.Descriptor >= 0 {
		syscall.Close(state.Platform.Descriptor)
		state.Platform.Descriptor = -1
	}
}

// Platform uses kernel timeouts reports that Linux submits Timeout through io_uring.
func platform_uses_kernel_timeouts() (uses bool) { return true }

// Platform submit pins operation memory and enqueues the corresponding SQE.
func platform_submit(
	state *Operating_System, operation *Operating_System_Operation,
) (err error) {
	pin_err := platform_pin(operation)
	if pin_err != nil {
		return pin_err
	}
	if operation.Deadline != 0 {
		return platform_submit_bounded_operation(state, operation)
	}
	return platform_submit_registered(state, operation)
}

// Platform submit bounded operation links Accept or Connect to a kernel timeout. The kernel
// retires both CQEs before the common completion becomes visible.
func platform_submit_bounded_operation(
	state *Operating_System, operation *Operating_System_Operation,
) (err error) {
	bounded := &Operating_System_Bounded_Operation{Operation: operation}
	deadline := &Operating_System_Operation{
		Completion: &time.Completion{},
		Kind:       OPERATING_SYSTEM_OPERATION_BOUNDED_DEADLINE,
		Descriptor: -1,
		Timespec:   operation.Deadline_Span,
		Bounded:    bounded,
	}
	bounded.Deadline_Operation = deadline
	operation.Bounded = bounded
	operating_system_operation_register(state, deadline)
	pin_err := platform_pin(deadline)
	if pin_err != nil {
		delete(state.Operations, deadline.Identifier)
		operation.Bounded = nil
		return pin_err
	}
	entries, entry_err := platform_get_entries(state, 2)
	if entry_err != nil {
		delete(state.Operations, deadline.Identifier)
		deadline.Pinner.Unpin()
		deadline.Pinned = false
		operation.Bounded = nil
		return entry_err
	}
	platform_prepare_entry(entries[0], operation)
	entries[0].Flags |= KERNEL_RING_SUBMISSION_LINK
	platform_prepare_entry(entries[1], deadline)
	state.Platform.IO_Queued += 2
	return nil
}

// Platform submit registered enqueues the same pinned operation for an EINTR/EAGAIN retry.
func platform_submit_registered(
	state *Operating_System, operation *Operating_System_Operation,
) (err error) {
	entry, entry_err := platform_get_entry(state)
	if entry_err != nil {
		return entry_err
	}
	platform_prepare_entry(entry, operation)
	state.Platform.IO_Queued++
	return nil
}

// Platform pin pins every Go address an SQE may outlive and builds connect's raw sockaddr.
func platform_pin(operation *Operating_System_Operation) (err error) {
	if operation.Pinned {
		return nil
	}
	if len(operation.Buffer) > 0 {
		operation.Pinner.Pin(&operation.Buffer[0])
	}
	if len(operation.File_Path) > 0 {
		operation.Pinner.Pin(&operation.File_Path[0])
	}
	if operation.Kind == OPERATING_SYSTEM_OPERATION_CONNECT {
		platform_address(operation)
		operation.Pinner.Pin(&operation.Socket_Address[0])
	}
	if operation.Kind == OPERATING_SYSTEM_OPERATION_TIMEOUT {
		operation.Pinner.Pin(&operation.Timespec)
	}
	if operation.Kind == OPERATING_SYSTEM_OPERATION_BOUNDED_DEADLINE {
		operation.Pinner.Pin(&operation.Timespec)
	}
	if operation.Kind == OPERATING_SYSTEM_OPERATION_EVENT {
		operation.Pinner.Pin(&operation.Event_Value)
	}
	if operation.Kind == OPERATING_SYSTEM_OPERATION_STATX {
		operation.Pinner.Pin(operation.Statx_Result)
	}
	operation.Pinned = true
	return nil
}

// Platform address encodes shared/io.Address as sockaddr_in or sockaddr_in6. The SQE takes the
// same bytes the synchronous calls pass, so both share one encoder. An address the encoder
// rejects leaves the size at zero, and the kernel then fails the operation with EINVAL.
func platform_address(operation *Operating_System_Operation) {
	size, encode_err := socket_address_encode(operation.Address, &operation.Socket_Address)
	if encode_err != nil {
		operation.Socket_Address_Size = 0
		return
	}
	operation.Socket_Address_Size = size
}

// Writes the two header bytes of a sockaddr. Linux holds the family as a host-order uint16 and
// carries no length byte, so size goes unused here and Darwin is the reason it is a parameter.
func platform_address_header(
	storage *[SOCKET_ADDRESS_BYTES]byte, family int, size uint32,
) {
	binary.LittleEndian.PutUint16(storage[0:2], uint16(family))
}

// Reads the family from the sockaddr the kernel wrote.
func platform_address_family(storage *[SOCKET_ADDRESS_BYTES]byte) (family int) {
	return int(binary.LittleEndian.Uint16(storage[0:2]))
}

// Platform get entry reserves one SQE, flushing a full submission queue before retrying exactly
// as TigerBeetle enqueue does in io/linux.zig:218-239.
func platform_get_entry(
	state *Operating_System,
) (entry *Kernel_Submission_Entry, err error) {
	entry = platform_reserve_entry(&state.Platform)
	if entry != nil {
		return entry, nil
	}
	flush_err := platform_enter(state, 0, 0)
	if flush_err != nil {
		return nil, flush_err
	}
	entry = platform_reserve_entry(&state.Platform)
	if entry == nil {
		return nil, errors.New("io: io_uring submission queue remained full after flush")
	}
	return entry, nil
}

// Platform get entries reserves one indivisible linked chain. It flushes before reserving so the
// accept SQE can never be published without its following timeout SQE.
func platform_get_entries(
	state *Operating_System, count int,
) (entries []*Kernel_Submission_Entry, err error) {
	if platform_entries_available(&state.Platform) < count {
		flush_err := platform_enter(state, 0, 0)
		if flush_err != nil {
			return nil, flush_err
		}
	}
	if platform_entries_available(&state.Platform) < count {
		return nil, errors.New("io: io_uring cannot reserve a bounded operation chain")
	}
	entries = make([]*Kernel_Submission_Entry, count)
	for index := 0; index < count; index++ {
		entries[index] = platform_reserve_entry(&state.Platform)
		invariant.Always(entries[index] != nil,
			"A preflighted bounded operation chain reserves every SQE.")
	}
	return entries, nil
}

// Platform entries available reports private SQ capacity not yet consumed by the kernel.
func platform_entries_available(platform *Platform_Scheduler) (count int) {
	head := atomic.LoadUint32(platform_uint32(platform.Submission_Ring,
		platform.Parameters.Submission.Head))
	used := platform.Submission_Tail - head
	return int(platform.Parameters.Submission_Entries - used)
}

// Platform reserve entry advances only the private SQE tail. The shared kernel tail is published
// by platform_publish after every SQE is fully initialized, matching io_uring.flush_sq and keeping
// SQPOLL from observing a partial entry.
func platform_reserve_entry(platform *Platform_Scheduler) (entry *Kernel_Submission_Entry) {
	head := atomic.LoadUint32(platform_uint32(platform.Submission_Ring,
		platform.Parameters.Submission.Head))
	tail := platform.Submission_Tail
	if tail-head >= platform.Parameters.Submission_Entries {
		return nil
	}
	mask := *platform_uint32(platform.Submission_Ring,
		platform.Parameters.Submission.Ring_Mask)
	index := tail & mask
	entry = (*Kernel_Submission_Entry)(unsafe.Pointer(&platform.Entries[index*64]))
	*entry = Kernel_Submission_Entry{}
	platform.Submission_Tail = tail + 1
	return entry
}

// Platform publish copies every fully prepared private SQE index into the shared submission array
// and releases the shared tail to the kernel in one final atomic store.
func platform_publish(platform *Platform_Scheduler) (published int) {
	head := platform.Submission_Head
	tail := platform.Submission_Tail
	if head == tail {
		return 0
	}
	mask := *platform_uint32(platform.Submission_Ring,
		platform.Parameters.Submission.Ring_Mask)
	kernel_tail_pointer := platform_uint32(platform.Submission_Ring,
		platform.Parameters.Submission.Tail)
	kernel_tail := atomic.LoadUint32(kernel_tail_pointer)
	for head != tail {
		submission_index := head & mask
		array_index := kernel_tail & mask
		array_offset := platform.Parameters.Submission.Array + array_index*4
		*platform_uint32(platform.Submission_Ring, array_offset) = submission_index
		head++
		kernel_tail++
		published++
	}
	platform.Submission_Head = tail
	platform.IO_Published += published
	atomic.StoreUint32(kernel_tail_pointer, kernel_tail)
	return published
}

// Platform prepare entry translates one operation tag to TigerBeetle's io_uring opcode.
func platform_prepare_entry(
	entry *Kernel_Submission_Entry, operation *Operating_System_Operation,
) {
	entry.Descriptor = int32(operation.Descriptor)
	entry.User_Data = operation.Identifier
	switch operation.Kind {
	case OPERATING_SYSTEM_OPERATION_ACCEPT:
		entry.Opcode = KERNEL_RING_OPERATION_ACCEPT
		entry.Operation_Flags = syscall.SOCK_CLOEXEC | syscall.SOCK_NONBLOCK
	case OPERATING_SYSTEM_OPERATION_CLOSE:
		entry.Opcode = KERNEL_RING_OPERATION_CLOSE
	case OPERATING_SYSTEM_OPERATION_CONNECT:
		entry.Opcode = KERNEL_RING_OPERATION_CONNECT
		entry.Address = uint64(uintptr(unsafe.Pointer(&operation.Socket_Address[0])))
		entry.Offset = uint64(operation.Socket_Address_Size)
	case OPERATING_SYSTEM_OPERATION_READ:
		platform_prepare_buffer(entry, operation, KERNEL_RING_OPERATION_READ)
		entry.Offset = operation.Offset
	case OPERATING_SYSTEM_OPERATION_RECEIVE:
		platform_prepare_buffer(entry, operation, KERNEL_RING_OPERATION_RECEIVE)
	case OPERATING_SYSTEM_OPERATION_PIPE_READ:
		platform_prepare_buffer(entry, operation, KERNEL_RING_OPERATION_READ)
		entry.Offset = KERNEL_PIPE_OFFSET
	case OPERATING_SYSTEM_OPERATION_PIPE_WRITE:
		platform_prepare_buffer(entry, operation, KERNEL_RING_OPERATION_WRITE)
		entry.Offset = KERNEL_PIPE_OFFSET
	case OPERATING_SYSTEM_OPERATION_MKDIR_AT:
		entry.Opcode = KERNEL_RING_OPERATION_MKDIR_AT
		entry.Address = uint64(uintptr(unsafe.Pointer(&operation.File_Path[0])))
		entry.Count = operation.Open_Options.Mode
	case OPERATING_SYSTEM_OPERATION_PROCESS_EXIT:
		entry.Opcode = KERNEL_RING_OPERATION_POLL_ADD
		entry.Descriptor = int32(operation.Descriptor)
		entry.Operation_Flags = KERNEL_POLL_INPUT
	case OPERATING_SYSTEM_OPERATION_SEND:
		platform_prepare_buffer(entry, operation, KERNEL_RING_OPERATION_SEND)
		entry.Operation_Flags = KERNEL_MESSAGE_NO_SIGNAL
	case OPERATING_SYSTEM_OPERATION_TIMEOUT:
		entry.Opcode = KERNEL_RING_OPERATION_TIMEOUT
		entry.Address = uint64(uintptr(unsafe.Pointer(&operation.Timespec)))
		entry.Count = 1
	case OPERATING_SYSTEM_OPERATION_BOUNDED_DEADLINE:
		entry.Opcode = KERNEL_RING_OPERATION_LINK_TIMEOUT
		entry.Address = uint64(uintptr(unsafe.Pointer(&operation.Timespec)))
		entry.Count = 1
	case OPERATING_SYSTEM_OPERATION_WRITE:
		platform_prepare_buffer(entry, operation, KERNEL_RING_OPERATION_WRITE)
		entry.Offset = operation.Offset
	case OPERATING_SYSTEM_OPERATION_FSYNC:
		entry.Opcode = KERNEL_RING_OPERATION_FSYNC
	case OPERATING_SYSTEM_OPERATION_OPEN_AT:
		entry.Opcode = KERNEL_RING_OPERATION_OPEN_AT
		entry.Address = uint64(uintptr(unsafe.Pointer(&operation.File_Path[0])))
		entry.Count = operation.Open_Options.Mode
		entry.Operation_Flags = uint32(platform_open_flags(operation.Open_Options))
	case OPERATING_SYSTEM_OPERATION_EVENT:
		entry.Opcode = KERNEL_RING_OPERATION_READ
		entry.Address = uint64(uintptr(unsafe.Pointer(&operation.Event_Value)))
		entry.Count = uint32(unsafe.Sizeof(operation.Event_Value))
	case OPERATING_SYSTEM_OPERATION_STATX:
		entry.Opcode = KERNEL_RING_OPERATION_STATX
		entry.Address = uint64(uintptr(unsafe.Pointer(&operation.File_Path[0])))
		entry.Offset = uint64(uintptr(unsafe.Pointer(operation.Statx_Result)))
		entry.Count = operation.Statx_Mask
		entry.Operation_Flags = operation.Statx_Flags
	}
}

// Returns Linux AT_FDCWD for TigerBeetle IO.openat.
func platform_current_directory() (descriptor int) { return -100 }

// Reads one pass of raw directory entries into buffer through getdents64, retrying EINTR. Go's
// ReadDirent already reaches this trap directly, so the raw call only drops the wrapper and
// keeps one shape with the Darwin backend, which needs the raw call for a stronger reason.
func platform_directory_read(descriptor int, buffer []byte) (count int, err error) {
	for retry_index := 0; retry_index < PLATFORM_INTERRUPT_RETRIES_MAX; retry_index++ {
		result, _, errno := syscall.Syscall(
			syscall.SYS_GETDENTS64, uintptr(descriptor),
			uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)),
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

// Reports whether a directory record names a file the filesystem already removed. Linux never
// does, so the answer is always false: an old XFS or a FUSE filesystem returns a valid file
// with a zero inode, and syscall.ParseDirent excludes Linux from that test for the same reason.
func platform_directory_absent(record *syscall.Dirent) (absent bool) {
	return false
}

// Translates the portable Open_At option fields to Linux posix.O bits and always forces CLOEXEC.
func platform_open_flags(options sharedio.Open_At_Options) (flags int) {
	flags = syscall.O_RDONLY | syscall.O_CLOEXEC
	if options.Access == sharedio.OPEN_WRITE_ONLY {
		flags = syscall.O_WRONLY | syscall.O_CLOEXEC
	}
	if options.Access == sharedio.OPEN_READ_WRITE {
		flags = syscall.O_RDWR | syscall.O_CLOEXEC
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

// Opens Linux eventfd with CLOEXEC exactly as TigerBeetle IO.open_event.
func platform_event_open(state *Operating_System) (event time.Event, err error) {
	result, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, 0, syscall.O_CLOEXEC, 0)
	if errno != 0 {
		return 0, errno
	}
	return time.Event(result), nil
}

// Arms Linux Event through the ordinary io_uring read path.
func platform_event_listen(
	state *Operating_System, operation *Operating_System_Operation,
) (err error) {
	return platform_submit(state, operation)
}

// Writes one eventfd notification; identifier is used by Darwin and intentionally irrelevant on
// Linux, matching third-party/tigerbeetle/src/io/linux.zig:1329-1337.
func platform_event_trigger(
	state *Operating_System, event time.Event, _ uint64,
) {
	buffer := [EVENTFD_VALUE_BYTES]byte{}
	binary.LittleEndian.PutUint64(buffer[:], 1)
	count, write_err := syscall.Write(int(event), buffer[:])
	for write_err == syscall.EINTR {
		count, write_err = syscall.Write(int(event), buffer[:])
	}
	invariant.Always(write_err == nil, "Triggering an eventfd Event succeeds.")
	invariant.Always(count == len(buffer), "Triggering an eventfd writes one uint64.")
}

// Closes Linux eventfd after its io_uring read listener has drained.
func platform_event_close(state *Operating_System, event time.Event) {
	close_err := syscall.Close(int(event))
	invariant.Always(close_err == nil, "Closing an eventfd Event succeeds.")
}

// Platform prepare buffer fills the shared read, write, recv, and send SQE fields.
func platform_prepare_buffer(
	entry *Kernel_Submission_Entry, operation *Operating_System_Operation, opcode uint8,
) {
	entry.Opcode = opcode
	entry.Count = uint32(len(operation.Buffer))
	if len(operation.Buffer) > 0 {
		entry.Address = uint64(uintptr(unsafe.Pointer(&operation.Buffer[0])))
	}
}

// Platform run flushes submissions with EXT_ARG, optionally waits for one CQE, then retires every
// available completion (third-party/tigerbeetle/src/io/linux.zig:70-209).
func platform_run(state *Operating_System, wait time.Moment) (err error) {
	retry_err := platform_retry_operations(state)
	if retry_err != nil {
		return retry_err
	}
	wait_count := uint32(0)
	if wait != 0 {
		if len(state.Completed) == 0 {
			wait_count = 1
		}
	}
	enter_err := platform_enter(state, wait_count, wait)
	if enter_err != nil {
		return enter_err
	}
	drain_err := platform_drain(state)
	if drain_err != nil {
		return drain_err
	}
	return platform_retry_operations(state)
}

// Platform enter submits queued SQEs and uses IORING_ENTER_EXT_ARG for bounded or unbounded waits.
func platform_enter(
	state *Operating_System, wait_count uint32, wait time.Moment,
) (err error) {
	platform_publish(&state.Platform)
	if state.Platform.IO_Published == 0 {
		if wait_count == 0 {
			return nil
		}
	}
	argument := Kernel_Enter_Argument{Signal_Mask_Size: 8}
	span := Kernel_Timespec{}
	if wait_count > 0 {
		if wait > 0 {
			span.Seconds = int64(wait) / 1_000_000_000
			span.Nanoseconds = int64(wait) % 1_000_000_000
			argument.Timespec = uint64(uintptr(unsafe.Pointer(&span)))
		}
	}
	flags := uint32(KERNEL_RING_ENTER_EXTENDED_ARGUMENT)
	if wait_count > 0 {
		flags |= KERNEL_RING_ENTER_GET_EVENTS
	}
	if state.Platform.Parameters.Flags&KERNEL_RING_SETUP_IO_POLL != 0 {
		flags |= KERNEL_RING_ENTER_GET_EVENTS
	}
	if state.Platform.Parameters.Flags&KERNEL_RING_SETUP_SUBMISSION_POLL != 0 {
		ring_flags := atomic.LoadUint32(platform_uint32(
			state.Platform.Submission_Ring, state.Platform.Parameters.Submission.Flags,
		))
		if ring_flags&KERNEL_RING_SUBMISSION_NEEDS_WAKEUP != 0 {
			flags |= KERNEL_RING_ENTER_SUBMISSION_WAKEUP
		} else if wait_count == 0 {
			platform_account_submitted(state, state.Platform.IO_Published)
			return nil
		}
	}
	for retry := true; retry; {
		to_submit := state.Platform.IO_Published
		result, _, errno := syscall.Syscall6(
			KERNEL_RING_ENTER_CALL, uintptr(state.Platform.Descriptor),
			uintptr(to_submit), uintptr(wait_count), uintptr(flags),
			uintptr(unsafe.Pointer(&argument)), unsafe.Sizeof(argument),
		)
		if errno == syscall.EINTR {
			continue
		}
		if errno == syscall.ETIME {
			platform_account_submitted(state, to_submit)
			return nil
		}
		switch errno {
		case syscall.EAGAIN, syscall.EBUSY:
			retry_err := platform_enter_retry(state)
			if retry_err != nil {
				return retry_err
			}
			continue
		}
		if errno != 0 {
			return errno
		}
		platform_account_submitted(state, int(result))
		return nil
	}
	return syscall.EINTR
}

// Platform enter retry retires one CQE before a temporarily blocked submission is retried.
func platform_enter_retry(state *Operating_System) (err error) {
	wait_err := platform_wait_one(state)
	if wait_err != nil {
		return wait_err
	}
	return platform_drain(state)
}

// Platform account submitted moves SQEs from the queued/published state into kernel ownership.
func platform_account_submitted(state *Operating_System, submitted int) {
	invariant.Always(submitted <= state.Platform.IO_Published,
		"io_uring never reports more submissions than were published.")
	state.Platform.IO_Queued -= submitted
	state.Platform.IO_Published -= submitted
	state.Platform.IO_In_Kernel += submitted
}

// Platform wait one mirrors TigerBeetle's completion-queue recovery: wait for one CQE without
// submitting again, then let the caller copy and retire completions before retrying submissions.
func platform_wait_one(state *Operating_System) (err error) {
	argument := Kernel_Enter_Argument{Signal_Mask_Size: 8}
	flags := uintptr(KERNEL_RING_ENTER_EXTENDED_ARGUMENT | KERNEL_RING_ENTER_GET_EVENTS)
	for retry := true; retry; {
		_, _, errno := syscall.Syscall6(
			KERNEL_RING_ENTER_CALL, uintptr(state.Platform.Descriptor), 0, 1, flags,
			uintptr(unsafe.Pointer(&argument)), unsafe.Sizeof(argument),
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

// Platform drain consumes all CQEs, retires operation identifiers before delivery, and rearms
// the repository-extension wake pipe after its reserved identifier zero fires.
func platform_drain(state *Operating_System) (err error) {
	entries := [KERNEL_COMPLETIONS_BATCH]Kernel_Completion_Entry{}
	for drain := true; drain; {
		count := platform_copy_completions(state, entries[:])
		if count == 0 {
			return nil
		}
		for index := 0; index < count; index++ {
			complete_err := platform_complete_entry(state, entries[index])
			if complete_err != nil {
				return complete_err
			}
		}
	}
	return nil
}

// Platform copy completions snapshots CQEs and advances the shared CQ head before any retry or
// callback path can fail, preventing one kernel completion from being replayed.
func platform_copy_completions(
	state *Operating_System, entries []Kernel_Completion_Entry,
) (count int) {
	head_pointer := platform_uint32(state.Platform.Completion_Ring,
		state.Platform.Parameters.Completion.Head)
	head := atomic.LoadUint32(head_pointer)
	tail := atomic.LoadUint32(platform_uint32(state.Platform.Completion_Ring,
		state.Platform.Parameters.Completion.Tail))
	mask := *platform_uint32(state.Platform.Completion_Ring,
		state.Platform.Parameters.Completion.Ring_Mask)
	for head != tail {
		if count == len(entries) {
			break
		}
		index := head & mask
		offset := state.Platform.Parameters.Completion.Completions + index*16
		entries[count] = *(*Kernel_Completion_Entry)(unsafe.Pointer(
			&state.Platform.Completion_Ring[offset]))
		count++
		head++
	}
	if count > 0 {
		state.Platform.IO_In_Kernel -= count
		atomic.StoreUint32(head_pointer, head)
	}
	return count
}

// Platform flush submissions performs TigerBeetle Linux run's final nonblocking flush after
// callbacks have queued more SQEs, without copying synchronously completed CQEs.
func platform_flush_submissions(state *Operating_System) (err error) {
	for flush := true; flush; {
		retry_err := platform_retry_operations(state)
		if retry_err != nil {
			return retry_err
		}
		enter_err := platform_enter(state, 0, 0)
		if enter_err != nil {
			return enter_err
		}
		if len(state.Platform.Retry_Backlog) == 0 {
			return nil
		}
	}
	return nil
}

// Platform complete entry handles one CQE and preserves an operation's registration on retry.
func platform_complete_entry(
	state *Operating_System, entry Kernel_Completion_Entry,
) (err error) {
	operation := state.Operations[entry.User_Data]
	if operation == nil {
		return nil
	}
	if operation.Bounded != nil {
		return platform_complete_bounded_entry(state, operation, entry.Result)
	}
	if operating_system_retryable_result(operation, entry.Result) {
		state.Platform.Retry_Backlog = append(state.Platform.Retry_Backlog, operation)
		return nil
	}
	result, operation_err := operating_system_translate_result(operation, entry.Result)
	operating_system_operation_complete(state, operation, result, operation_err)
	return nil
}

// Platform complete bounded entry joins both sides of a linked deadline. Deadline expiry wins a
// simultaneous primary result; an accepted descriptor is closed before delivery.
func platform_complete_bounded_entry(
	state *Operating_System, operation *Operating_System_Operation, result int32,
) (err error) {
	bounded := operation.Bounded
	if operation.Kind == OPERATING_SYSTEM_OPERATION_BOUNDED_DEADLINE {
		bounded.Deadline_Result = result
		bounded.Deadline_Completed = true
	} else {
		bounded.Operation_Result = result
		bounded.Operation_Completed = true
	}
	if !bounded.Operation_Completed {
		return nil
	}
	if !bounded.Deadline_Completed {
		return nil
	}
	primary := bounded.Operation
	deadline := bounded.Deadline_Operation
	delete(state.Operations, deadline.Identifier)
	if deadline.Pinned {
		deadline.Pinner.Unpin()
		deadline.Pinned = false
	}
	primary.Bounded = nil
	if bounded.Deadline_Result == -int32(syscall.ETIME) {
		if primary.Kind == OPERATING_SYSTEM_OPERATION_ACCEPT {
			if bounded.Operation_Result >= 0 {
				close_err := socket_close(int(bounded.Operation_Result))
				invariant.Always(close_err == nil,
					"An accepted descriptor that loses the deadline tie "+
						"closes immediately.")
			}
		}
		operating_system_operation_complete(
			state, primary, -1, time.Deadline_Exceeded,
		)
		return nil
	}
	if operating_system_retryable_result(primary, bounded.Operation_Result) {
		state.Platform.Retry_Backlog = append(state.Platform.Retry_Backlog, primary)
		return nil
	}
	translated, operation_err := operating_system_translate_result(
		primary, bounded.Operation_Result,
	)
	operating_system_operation_complete(state, primary, translated, operation_err)
	return nil
}

// Platform retry operations rearms interrupted operations after the CQE drain.
// Thus, a full submission queue cannot cause recursive completion processing.
func platform_retry_operations(state *Operating_System) (err error) {
	for len(state.Platform.Retry_Backlog) > 0 {
		operation := state.Platform.Retry_Backlog[0]
		var submit_err error
		if operation.Deadline != 0 {
			budget := operation.Deadline - state.Host.Now_Monotonic()
			if budget <= 0 {
				state.Platform.Retry_Backlog = state.Platform.Retry_Backlog[1:]
				operating_system_operation_complete(
					state, operation, -1, time.Deadline_Exceeded,
				)
				continue
			}
			operation.Deadline_Span = operating_system_timeout_span(
				time.Duration(budget),
			)
			submit_err = platform_submit_bounded_operation(state, operation)
		} else {
			submit_err = platform_submit_registered(state, operation)
		}
		if submit_err != nil {
			return submit_err
		}
		state.Platform.Retry_Backlog = state.Platform.Retry_Backlog[1:]
	}
	return nil
}

// Platform expire operation is implemented by Linux's linked timeout SQE, so the common
// user-space expiry pass has nothing to remove.
func platform_expire_operation(
	state *Operating_System, operation *Operating_System_Operation,
) (err error) {
	return nil
}

// Platform in flight reports work queued to or owned by io_uring.
func platform_in_flight(state *Operating_System) (in_flight bool) {
	if len(state.Platform.Retry_Backlog) > 0 {
		return true
	}
	return state.Platform.IO_Queued > 0 || state.Platform.IO_In_Kernel > 0
}

// Platform counts returns Linux's exact queued and in-kernel census.
func platform_counts(state *Operating_System) (backlog int, inflight int, queued int, kernel int) {
	return len(state.Platform.Retry_Backlog), 0,
		state.Platform.IO_Queued, state.Platform.IO_In_Kernel
}

// Platform uint32 addresses one aligned uint32 field inside an io_uring mapping.
func platform_uint32(memory []byte, offset uint32) (value *uint32) {
	return (*uint32)(unsafe.Pointer(&memory[offset]))
}
