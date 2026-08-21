//go:build linux && amd64

package nbio

import (
	"errors"
	"sync/atomic"
	"syscall"
	"unsafe"

	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/simulation/aver/default"
	"local/james-orcales/shared/simulation/nbio"
	"local/james-orcales/shared/simulation/time"
)

// SOCKET_RECEIVE_BUFFER_SIZE fix socket profile platform tests verify.
const SOCKET_RECEIVE_BUFFER_SIZE = 4 * 1024 * 1024

// SOCKET_SEND_BUFFER_SIZE fix socket profile platform tests verify.
const SOCKET_SEND_BUFFER_SIZE = 2 * 1024 * 1024

// Static identities keep scheduler failure paths inside caller allocation budget.
var kernel_extended_argument_required = errors.New(
	"io: Linux kernel 5.11 or newer with IORING_FEAT_EXT_ARG is required",
)

var kernel_submission_queue_full = errors.New(
	"io: io_uring submission queue remained full after flush",
)

var kernel_bounded_chain_unavailable = errors.New(
	"io: io_uring cannot reserve a bounded operation chain",
)

// SOCKET_RECEIVE_BUFFER_FORCE identify SO_RCVBUFFORCE, because syscall does not expose it.
const SOCKET_RECEIVE_BUFFER_FORCE = 33

// SOCKET_SEND_BUFFER_FORCE identify SO_SNDBUFFORCE, because syscall does not expose it.
const SOCKET_SEND_BUFFER_FORCE = 32

// SOCKET_TCP_NOT_SENT_LOW_WATER identifies TCP_NOTSENT_LOWAT missing from this syscall table.
const SOCKET_TCP_NOT_SENT_LOW_WATER = 25

// SOCKET_TCP_KEEPALIVE_IDLE keeps the portable test independent of platform option spelling.
const SOCKET_TCP_KEEPALIVE_IDLE = syscall.TCP_KEEPIDLE

// LINUX_BUFFER_SIZE_MAX stop byte count from overflow of signed kernel result.
const LINUX_BUFFER_SIZE_MAX = 0x7ffff000

// Platform_Operation keep Linux statx arguments stable until io_uring retire operation.
type Platform_Operation struct {
	// Statx_Result receive Linux statx output from io_uring.
	Statx_Result *nbio.Statx
	// Statx_Flags are forwarded to Linux statx.
	Statx_Flags uint32
	// Statx_Mask select Linux statx fields to return.
	Statx_Mask uint32
}

// Build child process attributes. Setpgid put child in its own group, thus deadline kill its
// descendants too. PidFD ask kernel for descriptor naming child, made atomic at fork, thus exit
// watch never name process identifier that could be reused.
func process_attributes(spawn *Spawn) (attributes *syscall.SysProcAttr) {
	return &syscall.SysProcAttr{Setpgid: true, PidFD: &spawn.Exit_Descriptor}
}

// Report whether kernel supplied pidfd. StartProcess leave it negative below Linux 5.3, and exit
// watch has nothing to poll without it, thus spawn fail loud here, not hang.
func process_watch_ready(spawn *Spawn) (err error) {
	if spawn.Exit_Descriptor < 0 {
		return syscall.ENOSYS
	}
	return nil
}

// Wire Linux IORING_OP_STATX, one operation absent from Darwin surface.
func operating_system_wire_platform(_ *Operating_System, loop *nbio.IO) {
	loop.Statx_Procedure = operating_system_statx
}

func operating_system_statx(
	state_pointer unsafe.Pointer, completion *nbio.Completion, directory nbio.File,
	file_path string, flags uint32, mask uint32, result *nbio.Statx,
	callback nbio.Callback,
) {
	state := (*Operating_System)(state_pointer)
	operating_system_submit(completion)
	descriptor := int(directory)
	if directory == nbio.DIRECTORY_CURRENT {
		descriptor = platform_current_directory()
	}
	operation := operating_system_operation_acquire(state, Operating_System_Operation{
		Completion: completion,
		Kind:       OPERATING_SYSTEM_OPERATION_STATX,
		Descriptor: descriptor,
		Platform_Operation: Platform_Operation{
			Statx_Result: result,
			Statx_Flags:  flags,
			Statx_Mask:   mask,
		},
		Deliver: callback,
	})
	operating_system_operation_submit_path(state, operation, file_path)
}

// Apply buffer limit for Linux before length reach signed kernel result.
func platform_buffer_limit(buffer []byte) (limited []byte) {
	if len(buffer) > LINUX_BUFFER_SIZE_MAX {
		return buffer[:LINUX_BUFFER_SIZE_MAX]
	}
	return buffer
}

// Linux carries both ownership flags in the socket type, so creation is atomic.
func socket_open_tcp_raw(family nbio.Address_Family) (descriptor int, err error) {
	return syscall.Socket(
		socket_family(family),
		syscall.SOCK_STREAM|syscall.SOCK_NONBLOCK|syscall.SOCK_CLOEXEC,
		syscall.IPPROTO_TCP,
	)
}

// Linux carries both ownership flags in the socket type, so creation is atomic.
func socket_open_udp_raw(family nbio.Address_Family) (descriptor int, err error) {
	return syscall.Socket(
		socket_family(family),
		syscall.SOCK_DGRAM|syscall.SOCK_NONBLOCK|syscall.SOCK_CLOEXEC,
		syscall.IPPROTO_UDP,
	)
}

// Apply CLOEXEC to synchronous accept helper. io_uring accept supply it in SQE.
func socket_accept_configure(descriptor int) (err error) {
	syscall.CloseOnExec(descriptor)
	return nil
}

// Apply receive-buffer size through the privileged name when the process permits it.
func platform_receive_buffer_set(descriptor int, value int) (err error) {
	return socket_buffer_force(
		descriptor, SOCKET_RECEIVE_BUFFER_FORCE, syscall.SO_RCVBUF, value)
}

// Apply send-buffer size through the privileged name when the process permits it.
func platform_send_buffer_set(descriptor int, value int) (err error) {
	return socket_buffer_force(
		descriptor, SOCKET_SEND_BUFFER_FORCE, syscall.SO_SNDBUF, value)
}

// Linux accepts the pure unconnected MSS default without a smaller platform cap.
func platform_tcp_maximum_segment_clamp(value uint32) (clamped uint32) {
	return value
}

// Apply the Linux spelling for the keepalive idle period.
func platform_keepalive_idle_set(descriptor int, value int) (err error) {
	return syscall.SetsockoptInt(descriptor, syscall.IPPROTO_TCP, syscall.TCP_KEEPIDLE, value)
}

// Apply the keepalive probe interval.
func platform_keepalive_interval_set(descriptor int, value int) (err error) {
	return syscall.SetsockoptInt(descriptor, syscall.IPPROTO_TCP, syscall.TCP_KEEPINTVL, value)
}

// Apply the keepalive probe count.
func platform_keepalive_count_set(descriptor int, value int) (err error) {
	return syscall.SetsockoptInt(descriptor, syscall.IPPROTO_TCP, syscall.TCP_KEEPCNT, value)
}

// Size socket buffer through privileged name first, because unprivileged one silently cap
// request at kernel rmem_max or wmem_max. Process without CAP_NET_ADMIN take capped size, not
// fail.
func socket_buffer_force(descriptor int, forced int, plain int, value int) (err error) {
	err = syscall.SetsockoptInt(descriptor, syscall.SOL_SOCKET, forced, value)
	if err == syscall.EPERM {
		return syscall.SetsockoptInt(descriptor, syscall.SOL_SOCKET, plain, value)
	}
	return err
}

// KERNEL_RING_SETUP_CALL keep raw syscall compatible with Linux amd64.
const KERNEL_RING_SETUP_CALL = 425

// KERNEL_RING_ENTER_CALL keep raw syscall compatible with Linux amd64.
const KERNEL_RING_ENTER_CALL = 426

// KERNEL_RING_SUBMISSION_OFFSET keep mmap compatible with Linux UAPI.
const KERNEL_RING_SUBMISSION_OFFSET = 0

// KERNEL_RING_COMPLETION_OFFSET keep mmap compatible with Linux UAPI.
const KERNEL_RING_COMPLETION_OFFSET = 0x08000000

// KERNEL_RING_ENTRIES_OFFSET keep mmap compatible with Linux UAPI.
const KERNEL_RING_ENTRIES_OFFSET = 0x10000000

// KERNEL_RING_FEATURE_SINGLE_MAPPING select Linux shared-ring feature bit.
const KERNEL_RING_FEATURE_SINGLE_MAPPING = 1

// KERNEL_RING_FEATURE_EXTENDED_ARGUMENT select required Linux timeout feature bit.
const KERNEL_RING_FEATURE_EXTENDED_ARGUMENT = 1 << 8

// KERNEL_RING_ENTER_GET_EVENTS make io_uring_enter wait for completions.
const KERNEL_RING_ENTER_GET_EVENTS = 1

// KERNEL_RING_ENTER_SUBMISSION_WAKEUP wake sleeping submission-poll thread.
const KERNEL_RING_ENTER_SUBMISSION_WAKEUP = 1 << 1

// KERNEL_RING_ENTER_EXTENDED_ARGUMENT permit bounded wait through io_uring_enter.
const KERNEL_RING_ENTER_EXTENDED_ARGUMENT = 1 << 3

// KERNEL_RING_SETUP_IO_POLL identify kernel I/O-poll setup mode.
const KERNEL_RING_SETUP_IO_POLL = 1

// KERNEL_RING_SETUP_SUBMISSION_POLL identify kernel submission-poll setup mode.
const KERNEL_RING_SETUP_SUBMISSION_POLL = 1 << 1

// KERNEL_RING_SUBMISSION_NEEDS_WAKEUP report sleeping submission-poll thread.
const KERNEL_RING_SUBMISSION_NEEDS_WAKEUP = 1

// KERNEL_RING_OPERATION_FSYNC keep SQE opcode compatible with Linux UAPI.
const KERNEL_RING_OPERATION_FSYNC = 3

// KERNEL_RING_OPERATION_POLL_ADD keep SQE opcode compatible with Linux UAPI.
const KERNEL_RING_OPERATION_POLL_ADD = 6

// KERNEL_RING_OPERATION_MKDIR_AT keep SQE opcode compatible with Linux UAPI. Kernel added it in
// 5.15.
const KERNEL_RING_OPERATION_MKDIR_AT = 37

// PLATFORM_STAT_AT_CALL is Linux newfstatat, whose struct match syscall.Stat_t on amd64.
const PLATFORM_STAT_AT_CALL = 262

// PLATFORM_READ_LINK_CALL reads final symbolic link without libc path conversion.
const PLATFORM_READ_LINK_CALL = syscall.SYS_READLINK

// PLATFORM_SYMBOLIC_LINK_NO_FOLLOW is Linux AT_SYMLINK_NOFOLLOW, thus directory pass report
// symbolic link as itself, not as its target.
const PLATFORM_SYMBOLIC_LINK_NO_FOLLOW = 0x100

// Cap EINTR retries of one eager filesystem syscall. Signal can interrupt call, but only broken
// kernel interrupt it over and over, thus bound report error, not spin.
const PLATFORM_INTERRUPT_RETRIES_MAX = 16

// KERNEL_PIPE_OFFSET tell kernel to read or write at current position of descriptor. Pipe is not
// seekable, thus it reject any other offset with ESPIPE.
const KERNEL_PIPE_OFFSET = ^uint64(0)

// KERNEL_POLL_INPUT is POLLIN. Pidfd report it once its process exited.
const KERNEL_POLL_INPUT = 0x001

// KERNEL_RING_OPERATION_TIMEOUT keep SQE opcode compatible with Linux UAPI.
const KERNEL_RING_OPERATION_TIMEOUT = 11

// KERNEL_RING_OPERATION_ACCEPT keep SQE opcode compatible with Linux UAPI.
const KERNEL_RING_OPERATION_ACCEPT = 13

// KERNEL_RING_OPERATION_LINK_TIMEOUT keep SQE opcode compatible with Linux UAPI.
const KERNEL_RING_OPERATION_LINK_TIMEOUT = 15

// KERNEL_BOUNDED_OPERATION_ENTRIES joins one primary SQE with one deadline SQE.
const KERNEL_BOUNDED_OPERATION_ENTRIES = 2

// KERNEL_RING_OPERATION_CONNECT keep SQE opcode compatible with Linux UAPI.
const KERNEL_RING_OPERATION_CONNECT = 16

// KERNEL_RING_OPERATION_OPEN_AT keep SQE opcode compatible with Linux UAPI.
const KERNEL_RING_OPERATION_OPEN_AT = 18

// KERNEL_RING_OPERATION_CLOSE keep SQE opcode compatible with Linux UAPI.
const KERNEL_RING_OPERATION_CLOSE = 19

// KERNEL_RING_OPERATION_STATX keep SQE opcode compatible with Linux UAPI.
const KERNEL_RING_OPERATION_STATX = 21

// KERNEL_RING_OPERATION_READ keep SQE opcode compatible with Linux UAPI.
const KERNEL_RING_OPERATION_READ = 22

// KERNEL_RING_OPERATION_WRITE keep SQE opcode compatible with Linux UAPI.
const KERNEL_RING_OPERATION_WRITE = 23

// KERNEL_RING_OPERATION_SEND keep SQE opcode compatible with Linux UAPI.
const KERNEL_RING_OPERATION_SEND = 26

// KERNEL_RING_OPERATION_RECEIVE keep SQE opcode compatible with Linux UAPI.
const KERNEL_RING_OPERATION_RECEIVE = 27

// KERNEL_RING_SUBMISSION_LINK join operation to its internal deadline.
const KERNEL_RING_SUBMISSION_LINK = 1 << 2

// KERNEL_MESSAGE_NO_SIGNAL stop SIGPIPE during socket send.
const KERNEL_MESSAGE_NO_SIGNAL = 0x4000

// KERNEL_RING_RESERVED_VALUES keep Kernel_Ring_Parameters equal to Linux UAPI layout.
const KERNEL_RING_RESERVED_VALUES = 3

// EVENTFD_VALUE_BYTES keep each eventfd notification equal to one uint64.
const EVENTFD_VALUE_BYTES = 8

// KERNEL_COMPLETIONS_BATCH bound stack use while code drain all available completions.
const KERNEL_COMPLETIONS_BATCH = 256

// Kernel ring offsets is stable Linux io_uring submission-ring offset layout.
type Kernel_Ring_Offsets struct {
	// Head keep kernel consumer cursor position.
	Head uint32
	// Tail keep application producer cursor position.
	Tail uint32
	// Ring_Mask permit UAPI ring-index calculation.
	Ring_Mask uint32
	// Ring_Entries keep UAPI ring capacity.
	Ring_Entries uint32
	// Flags expose submission-ring state that control wakeup.
	Flags uint32
	// Dropped keep UAPI counter position.
	Dropped uint32
	// Array locate submission index array.
	Array uint32
	// Reserved keep Linux UAPI layout.
	Reserved uint32
	// Reserved_Two keep Linux UAPI layout.
	Reserved_Two uint64
}

// Kernel completion offsets is stable Linux io_uring completion-ring offset layout.
type Kernel_Completion_Offsets struct {
	// Head keep application consumer cursor position.
	Head uint32
	// Tail keep kernel producer cursor position.
	Tail uint32
	// Ring_Mask permit UAPI ring-index calculation.
	Ring_Mask uint32
	// Ring_Entries keep UAPI ring capacity.
	Ring_Entries uint32
	// Overflow keep UAPI overflow counter position.
	Overflow uint32
	// Completions locate completion entries.
	Completions uint32
	// Flags keep Linux UAPI layout.
	Flags uint32
	// Reserved keep Linux UAPI layout.
	Reserved uint32
	// Reserved_Two keep Linux UAPI layout.
	Reserved_Two uint64
}

// Kernel ring parameters is struct io_uring_params from Linux UAPI.
type Kernel_Ring_Parameters struct {
	// Submission_Entries permit allocation of submission ring.
	Submission_Entries uint32
	// Completion_Entries permit allocation of completion ring.
	Completion_Entries uint32
	// Flags send requested setup modes to kernel.
	Flags uint32
	// Worker_CPU keep Linux UAPI layout.
	Worker_CPU uint32
	// Worker_Idle keep Linux UAPI layout.
	Worker_Idle uint32
	// Features report kernel capabilities this backend require.
	Features uint32
	// Worker_Descriptor keep Linux UAPI layout.
	Worker_Descriptor uint32
	// Reserved keep Linux UAPI layout.
	Reserved [KERNEL_RING_RESERVED_VALUES]uint32
	// Submission locate each submission-ring field.
	Submission Kernel_Ring_Offsets
	// Completion locate each completion-ring field.
	Completion Kernel_Completion_Offsets
}

// Kernel submission entry is 64-byte io_uring SQE of Linux. Operation-specific union fields keep
// their UAPI offsets under generic names used here.
type Kernel_Submission_Entry struct {
	// Opcode select kernel operation.
	Opcode uint8
	// Flags link operation to its internal deadline.
	Flags uint8
	// Priority keep Linux UAPI layout.
	Priority uint16
	// Descriptor transfer target file descriptor to kernel.
	Descriptor int32
	// Offset transfer file position, or secondary pointer, to kernel.
	Offset uint64
	// Address transfer primary operation pointer to kernel.
	Address uint64
	// Count transfer buffer length, or operation mask, to kernel.
	Count uint32
	// Operation_Flags transfer operation-specific options to kernel.
	Operation_Flags uint32
	// User_Data return operation identifier through completion entry.
	User_Data uint64
	// Buffer_Index keep Linux UAPI layout.
	Buffer_Index uint16
	// Personality keep Linux UAPI layout.
	Personality uint16
	// Splice_Input keep Linux UAPI layout.
	Splice_Input int32
	// Address_Three keep Linux UAPI layout.
	Address_Three uint64
	// Padding keep 64-byte Linux UAPI layout.
	Padding uint64
}

// Kernel completion entry is 16-byte io_uring CQE of Linux.
type Kernel_Completion_Entry struct {
	// User_Data return submitted operation identifier.
	User_Data uint64
	// Result return operation count, or negative errno.
	Result int32
	// Flags keep Linux UAPI layout.
	Flags uint32
}

// Kernel enter argument is io_uring_getevents_arg for IORING_ENTER_EXT_ARG.
type Kernel_Enter_Argument struct {
	// Signal_Mask keep Linux UAPI layout when this backend supply no mask.
	Signal_Mask uint64
	// Signal_Mask_Size tell kernel Signal_Mask is absent.
	Signal_Mask_Size uint32
	// Padding keep Linux UAPI layout.
	Padding uint32
	// Timespec give io_uring_enter bounded wait.
	Timespec uint64
}

// Platform scheduler is io_uring, plus exact queue counters.
type Platform_Scheduler struct {
	// Descriptor hold io_uring instance until deinitialization.
	Descriptor int
	// Parameters hold kernel offsets that read each mapping.
	Parameters Kernel_Ring_Parameters
	// Submission_Ring hold submission mapping until deinitialization.
	Submission_Ring []byte
	// Completion_Ring hold completion mapping until deinitialization.
	Completion_Ring []byte
	// Entries hold SQE mapping until deinitialization.
	Entries []byte
	// Single_Mapping stop code from release of one shared mapping twice.
	Single_Mapping bool
	// Submission_Head track reserved SQEs before publication.
	Submission_Head uint32
	// Submission_Tail track published SQEs before kernel submission.
	Submission_Tail uint32
	// IO_Queued keep queue census exact before publication.
	IO_Queued int
	// IO_Published keep queue census exact before kernel submission.
	IO_Published int
	// IO_In_Kernel keep queue census exact before completion.
	IO_In_Kernel int
	// Retry_Backlog stop retry from re-entry into CQE drain.
	Retry_Backlog []*Operating_System_Operation
}

// Platform memory binds Linux retry backlog to caller capacity.
func platform_memory_set(
	platform *Platform_Scheduler, operations []*Operating_System_Operation,
) {
	platform.Retry_Backlog = operations[:0]
}

// Platform initialize make io_uring eagerly and reject kernel without EXT_ARG.
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
		return Platform_Scheduler{}, kernel_extended_argument_required
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

// Platform map map submission ring, completion ring, and SQE array setup describe.
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

// Platform mmap input identify one io_uring memory mapping.
type Platform_Mmap_Input struct {
	// Descriptor select io_uring instance that own mapping.
	Descriptor int
	// Offset select required io_uring memory region.
	Offset int64
	// Size bound mapping to kernel-reported region size.
	Size int
}

// Platform mmap map one io_uring region as shared read-write memory.
func platform_mmap(input *Platform_Mmap_Input) (memory []byte, err error) {
	return syscall.Mmap(input.Descriptor, input.Offset, input.Size,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED|syscall.MAP_POPULATE)
}

// Platform unmap release every mapped io_uring region exactly once.
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

// Platform deinitialize release io_uring after every registered operation completed.
func platform_deinitialize(state *Operating_System) {
	platform_unmap(&state.Platform)
	if state.Platform.Descriptor >= 0 {
		syscall.Close(state.Platform.Descriptor)
		state.Platform.Descriptor = -1
	}
}

// Platform uses kernel timeouts report that Linux submit Timeout through io_uring.
func platform_uses_kernel_timeouts() (uses bool) { return true }

// The absolute moment keeps scheduler backlog and every retry inside the caller's first bound.
func platform_storage_deadline(
	state *Operating_System, timeout time.Duration,
) (deadline time.Monotonic_Moment) {
	return time.Clock_Now_Monotonic(state.Host) + time.Monotonic_Moment(timeout)
}

// Platform submit pin operation memory and enqueue matching SQE.
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

// Platform submit bounded operation links one request to its remaining API-level budget.
// Kernel retire both CQEs before common completion become visible.
func platform_submit_bounded_operation(
	state *Operating_System, operation *Operating_System_Operation,
) (err error) {
	budget := operation.Deadline - time.Clock_Now_Monotonic(state.Host)
	if budget <= 0 {
		return nbio.Deadline_Exceeded
	}
	bounded := &operation.Bounded_State
	*bounded = Operating_System_Bounded_Operation{Operation: operation}
	deadline := operating_system_operation_acquire(state, Operating_System_Operation{
		Completion: &operation.Internal_Completion,
		Kind:       OPERATING_SYSTEM_OPERATION_BOUNDED_DEADLINE,
		Descriptor: -1,
		Bounded:    bounded,
	})
	bounded.Deadline_Operation = deadline
	operation.Bounded = bounded
	operating_system_operation_register(state, deadline)
	pin_err := platform_pin(deadline)
	if pin_err != nil {
		operating_system_operation_unregister(state, deadline)
		operating_system_operation_release(deadline)
		operation.Bounded = nil
		return pin_err
	}
	primary_entry, deadline_entry, entry_err := platform_get_entry_pair(state)
	if entry_err != nil {
		operating_system_operation_unregister(state, deadline)
		deadline.Pinner.Unpin()
		deadline.Pinned = false
		operating_system_operation_release(deadline)
		operation.Bounded = nil
		return entry_err
	}
	budget = operation.Deadline - time.Clock_Now_Monotonic(state.Host)
	if budget <= 0 {
		state.Platform.Submission_Tail -= KERNEL_BOUNDED_OPERATION_ENTRIES
		operating_system_operation_unregister(state, deadline)
		deadline.Pinner.Unpin()
		deadline.Pinned = false
		operating_system_operation_release(deadline)
		operation.Bounded = nil
		return nbio.Deadline_Exceeded
	}
	operation.Deadline_Span = operating_system_timeout_span(time.Duration(budget))
	deadline.Timespec = operation.Deadline_Span
	platform_prepare_entry(primary_entry, operation)
	primary_entry.Flags |= KERNEL_RING_SUBMISSION_LINK
	platform_prepare_entry(deadline_entry, deadline)
	state.Platform.IO_Queued += KERNEL_BOUNDED_OPERATION_ENTRIES
	return nil
}

// Platform submit registered enqueue same pinned operation for EINTR/EAGAIN retry.
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

// Platform pin pin every Go address SQE may outlive, and build raw sockaddr of connect.
func platform_pin(operation *Operating_System_Operation) (err error) {
	if operation.Pinned {
		return nil
	}
	if len(operation.Buffer) > 0 {
		operation.Pinner.Pin(&operation.Buffer[0])
	}
	if operation.File_Path_Count > 0 {
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

// Platform address encode nbio.Address as sockaddr_in, or sockaddr_in6. SQE take same bytes
// synchronous calls pass, thus both share one encoder. Address encoder reject leave size at zero,
// and kernel then fail operation with EINVAL.
func platform_address(operation *Operating_System_Operation) {
	size, encode_err := socket_address_encode(operation.Address, &operation.Socket_Address)
	if encode_err != nil {
		operation.Socket_Address_Size = 0
		return
	}
	operation.Socket_Address_Size = size
}

// Write two header bytes of sockaddr. Linux hold family as host-order uint16 and carry no length
// byte, thus size go unused here, and Darwin is reason it is parameter.
func platform_address_header(
	storage *[SOCKET_ADDRESS_BYTES]byte, family int, size uint32,
) {
	binary.Put_Uint_16(
		binary.Bytes(storage[0:2]), binary.Word_16(family), binary.LITTLE_ENDIAN,
	)
}

// Read family from sockaddr kernel wrote.
func platform_address_family(storage *[SOCKET_ADDRESS_BYTES]byte) (family int) {
	return int(binary.Uint_16(binary.Bytes(storage[0:2]), binary.LITTLE_ENDIAN))
}

// Platform get entry reserve one SQE. It flush full submission queue before retry.
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
		return nil, kernel_submission_queue_full
	}
	return entry, nil
}

// Platform get entries reserve one indivisible linked chain. It flush before reserve, thus accept
// SQE can never be published without its following timeout SQE.
func platform_get_entry_pair(
	state *Operating_System,
) (first *Kernel_Submission_Entry, second *Kernel_Submission_Entry, err error) {
	if platform_entries_available(&state.Platform) < KERNEL_BOUNDED_OPERATION_ENTRIES {
		flush_err := platform_enter(state, 0, 0)
		if flush_err != nil {
			return nil, nil, flush_err
		}
	}
	if platform_entries_available(&state.Platform) < KERNEL_BOUNDED_OPERATION_ENTRIES {
		return nil, nil, kernel_bounded_chain_unavailable
	}
	first = platform_reserve_entry(&state.Platform)
	second = platform_reserve_entry(&state.Platform)
	aver.Always(first != nil,
		"A preflighted bounded operation chain reserves its primary SQE.")
	aver.Always(second != nil,
		"A preflighted bounded operation chain reserves its deadline SQE.")
	return first, second, nil
}

// Platform entries available report private SQ capacity kernel not yet consumed.
func platform_entries_available(platform *Platform_Scheduler) (count int) {
	head := atomic.LoadUint32(platform_uint32(platform.Submission_Ring,
		platform.Parameters.Submission.Head))
	used := platform.Submission_Tail - head
	return int(platform.Parameters.Submission_Entries - used)
}

// Platform reserve entry advance only private SQE tail. platform_publish publish shared kernel
// tail after every SQE is fully initialized, same as io_uring.flush_sq, thus SQPOLL never observe
// partial entry.
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

// Platform publish copy every fully prepared private SQE index into shared submission array, and
// release shared tail to kernel in one final atomic store.
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

// Platform prepare entry translate one operation tag to io_uring opcode.
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
		entry.Count = uint32(
			nbio.File_Permissions_To_POSIX(operation.Open_Options.Permissions),
		)
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
		entry.Count = uint32(
			nbio.File_Permissions_To_POSIX(operation.Open_Options.Permissions),
		)
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

// Return Linux AT_FDCWD for openat.
func platform_current_directory() (descriptor int) { return -100 }

// Read one pass of raw directory entries into buffer through getdents64, retrying EINTR. Go
// ReadDirent already reach this trap direct, thus raw call only drop wrapper and keep one shape
// with Darwin backend, which need raw call for stronger reason.
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

// Report whether directory record name file filesystem already removed. Linux never do, thus
// answer is always false: old XFS, or FUSE filesystem, return valid file with zero inode, and
// syscall.ParseDirent exclude Linux from that test for same reason.
func platform_directory_absent(record *syscall.Dirent) (absent bool) {
	return false
}

// Translate portable Open_At option fields to Linux posix.O bits, and always force CLOEXEC.
func platform_open_flags(options nbio.Open_At_Options) (flags int) {
	flags = syscall.O_RDONLY | syscall.O_CLOEXEC
	if options.Access == nbio.OPEN_WRITE_ONLY {
		flags = syscall.O_WRONLY | syscall.O_CLOEXEC
	}
	if options.Access == nbio.OPEN_READ_WRITE {
		flags = syscall.O_RDWR | syscall.O_CLOEXEC
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

// Open Linux eventfd with CLOEXEC.
func platform_event_open(state *Operating_System) (event nbio.Event, err error) {
	result, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, 0, syscall.O_CLOEXEC, 0)
	if errno != 0 {
		return 0, errno
	}
	return nbio.Event(result), nil
}

// Arm Linux Event through ordinary io_uring read path.
func platform_event_listen(
	state *Operating_System, operation *Operating_System_Operation,
) (err error) {
	return platform_submit(state, operation)
}

// Write one eventfd notification. Darwin use identifier, and it is deliberately irrelevant on
// Linux.
func platform_event_trigger(
	state *Operating_System, event nbio.Event, _ uint64,
) {
	buffer := [EVENTFD_VALUE_BYTES]byte{}
	binary.Put_Uint_64(binary.Bytes(buffer[:]), 1, binary.LITTLE_ENDIAN)
	count, write_err := syscall.Write(int(event), buffer[:])
	for write_err == syscall.EINTR {
		count, write_err = syscall.Write(int(event), buffer[:])
	}
	aver.Always(write_err == nil, "Triggering an eventfd Event succeeds.")
	aver.Always(count == len(buffer), "Triggering an eventfd writes one uint64.")
}

// Close Linux eventfd after its io_uring read listener drained.
func platform_event_close(state *Operating_System, event nbio.Event) {
	close_err := syscall.Close(int(event))
	aver.Always(close_err == nil, "Closing an eventfd Event succeeds.")
}

// Platform prepare buffer fill shared read, write, recv, and send SQE fields.
func platform_prepare_buffer(
	entry *Kernel_Submission_Entry, operation *Operating_System_Operation, opcode uint8,
) {
	entry.Opcode = opcode
	entry.Count = uint32(len(operation.Buffer))
	if len(operation.Buffer) > 0 {
		entry.Address = uint64(uintptr(unsafe.Pointer(&operation.Buffer[0])))
	}
}

// Platform run flush submissions with EXT_ARG, may wait for one CQE, then retire every available
// completion.
func platform_run(state *Operating_System, wait time.Monotonic_Moment) (err error) {
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

// Platform enter submit queued SQEs and use IORING_ENTER_EXT_ARG for bounded or unbounded wait.
func platform_enter(
	state *Operating_System, wait_count uint32, wait time.Monotonic_Moment,
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

// Platform enter retry retire one CQE before retry of temporarily blocked submission.
func platform_enter_retry(state *Operating_System) (err error) {
	wait_err := platform_wait_one(state)
	if wait_err != nil {
		return wait_err
	}
	return platform_drain(state)
}

// Platform account submitted move SQEs from queued/published state into kernel ownership.
func platform_account_submitted(state *Operating_System, submitted int) {
	aver.Always(submitted <= state.Platform.IO_Published,
		"io_uring never reports more submissions than were published.")
	state.Platform.IO_Queued -= submitted
	state.Platform.IO_Published -= submitted
	state.Platform.IO_In_Kernel += submitted
}

// Platform wait one is completion-queue recovery: wait for one CQE without second submit, then
// let caller copy and retire completions before retry of submissions.
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

// Platform drain consume all CQEs, retire operation identifiers before delivery, and rearm
// repository-extension wake pipe after its reserved identifier zero fire.
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

// Platform copy completions snapshot CQEs and advance shared CQ head before any retry or callback
// path can fail, thus one kernel completion is never replayed.
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

// Platform flush submissions do final nonblocking flush of platform run after callbacks
// queued more SQEs, without copy of synchronously completed CQEs.
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

// Platform complete entry handle one CQE and keep operation registration on retry.
func platform_complete_entry(
	state *Operating_System, entry Kernel_Completion_Entry,
) (err error) {
	operation := operating_system_operation_find(state, entry.User_Data)
	if operation == nil {
		return nil
	}
	if operation.Bounded != nil {
		return platform_complete_bounded_entry(state, operation, entry.Result)
	}
	if operating_system_retryable_result(operation, entry.Result) {
		platform_retry_add(state, operation)
		return nil
	}
	result, operation_err := operating_system_translate_result(operation, entry.Result)
	operating_system_operation_complete(state, operation, result, operation_err)
	return nil
}

// Platform complete bounded entry join both sides because callback-visible retirement requires
// the kernel to release every address from both SQEs.
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
	operating_system_operation_unregister(state, deadline)
	if deadline.Pinned {
		deadline.Pinner.Unpin()
		deadline.Pinned = false
	}
	operating_system_operation_release(deadline)
	primary.Bounded = nil
	if platform_bounded_timeout_won(
		bounded.Operation_Result, bounded.Deadline_Result,
	) {
		operating_system_operation_complete(
			state, primary, operating_system_timeout_result(primary),
			nbio.Deadline_Exceeded,
		)
		return nil
	}
	if operating_system_retryable_result(primary, bounded.Operation_Result) {
		platform_retry_add(state, primary)
		return nil
	}
	translated, operation_err := operating_system_translate_result(
		primary, bounded.Operation_Result,
	)
	operating_system_operation_complete(state, primary, translated, operation_err)
	return nil
}

// Only cancellation-shaped primary results prove the linked timeout stopped the operation.
func platform_bounded_timeout_won(operation_result int32, deadline_result int32) (won bool) {
	if deadline_result != -int32(syscall.ETIME) {
		return false
	}
	errno := syscall.Errno(-operation_result)
	return errno == syscall.ECANCELED || errno == syscall.EINTR
}

// Platform retry operations rearm interrupted operations after CQE drain. Thus full submission
// queue cannot cause recursive completion processing.
func platform_retry_operations(state *Operating_System) (err error) {
	for len(state.Platform.Retry_Backlog) > 0 {
		operation := state.Platform.Retry_Backlog[0]
		var submit_err error
		if operation.Deadline != 0 {
			budget := operation.Deadline - time.Clock_Now_Monotonic(state.Host)
			if budget <= 0 {
				platform_retry_pop(state)
				timeout_result := operating_system_timeout_result(operation)
				operating_system_operation_complete(
					state, operation, timeout_result, nbio.Deadline_Exceeded,
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
		platform_retry_pop(state)
	}
	return nil
}

func platform_retry_pop(state *Operating_System) {
	backlog := state.Platform.Retry_Backlog
	copy(backlog, backlog[1:])
	backlog[len(backlog)-1] = nil
	state.Platform.Retry_Backlog = backlog[:len(backlog)-1]
}

func platform_retry_add(
	state *Operating_System, operation *Operating_System_Operation,
) {
	aver.Always(len(state.Platform.Retry_Backlog) < cap(state.Platform.Retry_Backlog),
		"The caller-owned Linux retry backlog has capacity before retry.")
	state.Platform.Retry_Backlog = append(state.Platform.Retry_Backlog, operation)
}

// Linked timeout SQE of Linux implement platform expire operation, thus common user-space expiry
// pass has nothing to remove.
func platform_expire_operation(
	state *Operating_System, operation *Operating_System_Operation,
) (err error) {
	return nil
}

// Platform in flight report work queued to io_uring, or owned by it.
func platform_in_flight(state *Operating_System) (in_flight bool) {
	if len(state.Platform.Retry_Backlog) > 0 {
		return true
	}
	return state.Platform.IO_Queued > 0 || state.Platform.IO_In_Kernel > 0
}

// Platform counts return exact Linux queued and in-kernel census.
func platform_counts(state *Operating_System) (backlog int, inflight int, queued int, kernel int) {
	return len(state.Platform.Retry_Backlog), 0,
		state.Platform.IO_Queued, state.Platform.IO_In_Kernel
}

// Platform uint32 address one aligned uint32 field inside io_uring mapping.
func platform_uint32(memory []byte, offset uint32) (value *uint32) {
	return (*uint32)(unsafe.Pointer(&memory[offset]))
}
