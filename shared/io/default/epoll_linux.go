//go:build linux && amd64

package io

import (
	"encoding/binary"
	"errors"
	"sync/atomic"
	"syscall"
	"unsafe"

	invariant "local/james-orcales/g/shared/invariant/default"
	sharedio "local/james-orcales/g/shared/io"
	"local/james-orcales/g/shared/time"
)

const socket_receive_buffer_size = 4 * 1024 * 1024
const socket_send_buffer_size = 2 * 1024 * 1024
const socket_keepalive_idle_seconds = 5
const socket_keepalive_interval_seconds = 4
const socket_keepalive_count = 3
const socket_user_timeout_milliseconds = 17 * 1000
const socket_receive_buffer_force = 33
const socket_send_buffer_force = 32
const socket_user_timeout = 18
const linux_buffer_size_max = 0x7ffff000

// Platform operation holds Linux statx arguments whose addresses remain stable in io_uring.
type Platform_Operation struct {
	// Statx_Result receives Linux statx output from io_uring.
	Statx_Result *sharedio.Statx
	// Statx_Flags are forwarded to Linux statx.
	Statx_Flags uint32
	// Statx_Mask selects the Linux statx fields to return.
	Statx_Mask uint32
}

// Wires Linux IORING_OP_STATX, the one operation absent from TigerBeetle's Darwin surface.
func operating_system_wire_platform(state *operating_system, loop *sharedio.IO) {
	loop.Statx = func(
		completion *sharedio.Completion, callback sharedio.Timeout_Callback,
		directory sharedio.File, file_path string, flags uint32, mask uint32,
		result *sharedio.Statx,
	) {
		operating_system_submit(completion)
		descriptor := int(directory)
		if directory == sharedio.DIRECTORY_CURRENT {
			descriptor = platform_current_directory()
		}
		operation := &operating_system_operation{
			Completion: completion,
			Kind:       operating_system_operation_statx,
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
	if len(buffer) > linux_buffer_size_max {
		return buffer[:linux_buffer_size_max]
	}
	return buffer
}

// Opens and configures one non-blocking close-on-exec TCP socket.
func socket_open_tcp(
	family sharedio.Address_Family, options sharedio.TCP_Options,
) (descriptor int, err error) {
	descriptor, err = syscall.Socket(
		socket_family(family),
		syscall.SOCK_STREAM|syscall.SOCK_NONBLOCK|syscall.SOCK_CLOEXEC,
		syscall.IPPROTO_TCP,
	)
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
	return syscall.Socket(
		socket_family(family),
		syscall.SOCK_DGRAM|syscall.SOCK_NONBLOCK|syscall.SOCK_CLOEXEC,
		syscall.IPPROTO_UDP,
	)
}

// Maps the backend-independent family to Linux's socket family constant.
func socket_family(family sharedio.Address_Family) (system int) {
	if family == sharedio.FAMILY_IPV6 {
		return syscall.AF_INET6
	}
	return syscall.AF_INET
}

// Applies CLOEXEC to the synchronous accept helper; io_uring accept supplies it in the SQE.
func socket_accept_configure(descriptor int) (err error) {
	syscall.CloseOnExec(descriptor)
	return nil
}

func socket_configure(descriptor int, options sharedio.TCP_Options) (err error) {
	if options.Receive_Buffer > 0 {
		err = syscall.SetsockoptInt(
			descriptor, syscall.SOL_SOCKET,
			socket_receive_buffer_force, options.Receive_Buffer,
		)
		if err == syscall.EPERM {
			err = syscall.SetsockoptInt(
				descriptor, syscall.SOL_SOCKET,
				syscall.SO_RCVBUF, options.Receive_Buffer,
			)
		}
		if err != nil {
			return err
		}
	}
	if options.Send_Buffer > 0 {
		err = syscall.SetsockoptInt(
			descriptor, syscall.SOL_SOCKET,
			socket_send_buffer_force, options.Send_Buffer,
		)
		if err == syscall.EPERM {
			err = syscall.SetsockoptInt(
				descriptor, syscall.SOL_SOCKET,
				syscall.SO_SNDBUF, options.Send_Buffer,
			)
		}
		if err != nil {
			return err
		}
	}
	if options.Keepalive != nil {
		err = socket_configure_keepalive(descriptor, options.Keepalive)
		if err != nil {
			return err
		}
	}
	if options.User_Timeout_Milliseconds > 0 {
		err = syscall.SetsockoptInt(
			descriptor, syscall.IPPROTO_TCP, socket_user_timeout,
			options.User_Timeout_Milliseconds,
		)
		if err != nil {
			return err
		}
	}
	if options.No_Delay {
		return syscall.SetsockoptInt(
			descriptor, syscall.IPPROTO_TCP, syscall.TCP_NODELAY, 1,
		)
	}
	return nil
}

// Applies Linux's full keepalive tuple after enabling SO_KEEPALIVE.
func socket_configure_keepalive(
	descriptor int, keepalive *sharedio.TCP_Keepalive,
) (err error) {
	err = syscall.SetsockoptInt(descriptor, syscall.SOL_SOCKET, syscall.SO_KEEPALIVE, 1)
	if err != nil {
		return err
	}
	err = syscall.SetsockoptInt(
		descriptor, syscall.IPPROTO_TCP, syscall.TCP_KEEPIDLE, keepalive.Idle_Seconds,
	)
	if err != nil {
		return err
	}
	err = syscall.SetsockoptInt(
		descriptor, syscall.IPPROTO_TCP, syscall.TCP_KEEPINTVL, keepalive.Interval_Seconds,
	)
	if err != nil {
		return err
	}
	return syscall.SetsockoptInt(
		descriptor, syscall.IPPROTO_TCP, syscall.TCP_KEEPCNT, keepalive.Count,
	)
}

const kernel_ring_setup_call = 425
const kernel_ring_enter_call = 426
const kernel_ring_submission_offset = 0
const kernel_ring_completion_offset = 0x08000000
const kernel_ring_entries_offset = 0x10000000
const kernel_ring_feature_single_mapping = 1
const kernel_ring_feature_extended_argument = 1 << 8
const kernel_ring_enter_get_events = 1
const kernel_ring_enter_submission_wakeup = 1 << 1
const kernel_ring_enter_extended_argument = 1 << 3
const kernel_ring_setup_io_poll = 1
const kernel_ring_setup_submission_poll = 1 << 1
const kernel_ring_submission_needs_wakeup = 1
const kernel_ring_operation_fsync = 3
const kernel_ring_operation_timeout = 11
const kernel_ring_operation_accept = 13
const kernel_ring_operation_link_timeout = 15
const kernel_ring_operation_connect = 16
const kernel_ring_operation_open_at = 18
const kernel_ring_operation_close = 19
const kernel_ring_operation_statx = 21
const kernel_ring_operation_read = 22
const kernel_ring_operation_write = 23
const kernel_ring_operation_send = 26
const kernel_ring_operation_receive = 27
const kernel_ring_submission_link = 1 << 2
const kernel_message_no_signal = 0x4000

// Kernel ring offsets is the stable Linux io_uring submission-ring offset layout.
type kernel_ring_offsets struct {
	Head         uint32
	Tail         uint32
	Ring_Mask    uint32
	Ring_Entries uint32
	Flags        uint32
	Dropped      uint32
	Array        uint32
	Reserved     uint32
	Reserved_Two uint64
}

// Kernel completion offsets is the stable Linux io_uring completion-ring offset layout.
type kernel_completion_offsets struct {
	Head         uint32
	Tail         uint32
	Ring_Mask    uint32
	Ring_Entries uint32
	Overflow     uint32
	Completions  uint32
	Flags        uint32
	Reserved     uint32
	Reserved_Two uint64
}

// Kernel ring parameters is struct io_uring_params from Linux's UAPI.
type kernel_ring_parameters struct {
	Submission_Entries uint32
	Completion_Entries uint32
	Flags              uint32
	Worker_CPU         uint32
	Worker_Idle        uint32
	Features           uint32
	Worker_Descriptor  uint32
	Reserved           [3]uint32
	Submission         kernel_ring_offsets
	Completion         kernel_completion_offsets
}

// Kernel submission entry is Linux's 64-byte io_uring SQE. The operation-specific union fields
// retain their UAPI offsets under the generic names used here.
type kernel_submission_entry struct {
	Opcode          uint8
	Flags           uint8
	Priority        uint16
	Descriptor      int32
	Offset          uint64
	Address         uint64
	Count           uint32
	Operation_Flags uint32
	User_Data       uint64
	Buffer_Index    uint16
	Personality     uint16
	Splice_Input    int32
	Address_Three   uint64
	Padding         uint64
}

// Kernel completion entry is Linux's 16-byte io_uring CQE.
type kernel_completion_entry struct {
	User_Data uint64
	Result    int32
	Flags     uint32
}

// Kernel enter argument is io_uring_getevents_arg for IORING_ENTER_EXT_ARG.
type kernel_enter_argument struct {
	Signal_Mask      uint64
	Signal_Mask_Size uint32
	Padding          uint32
	Timespec         uint64
}

// Platform scheduler is TigerBeetle Linux IO's io_uring plus exact queue counters.
type platform_scheduler struct {
	Descriptor      int
	Parameters      kernel_ring_parameters
	Submission_Ring []byte
	Completion_Ring []byte
	Entries         []byte
	Single_Mapping  bool
	Submission_Head uint32
	Submission_Tail uint32
	IO_Queued       int
	IO_Published    int
	IO_In_Kernel    int
	Bounded_Backlog []*operating_system_operation
}

// Platform initialize creates io_uring eagerly and rejects kernels without EXT_ARG, matching
// third-party/tigerbeetle/src/io/linux.zig:37-68. No epoll fallback exists.
func platform_initialize(entries uint16, flags uint32) (platform platform_scheduler, err error) {
	parameters := kernel_ring_parameters{Flags: flags}
	result, _, errno := syscall.Syscall(
		kernel_ring_setup_call, uintptr(entries), uintptr(unsafe.Pointer(&parameters)), 0,
	)
	if errno != 0 {
		return platform_scheduler{}, errno
	}
	descriptor := int(result)
	if parameters.Features&kernel_ring_feature_extended_argument == 0 {
		syscall.Close(descriptor)
		return platform_scheduler{}, errors.New(
			"io: Linux kernel 5.11 or newer with IORING_FEAT_EXT_ARG is required",
		)
	}
	platform = platform_scheduler{Descriptor: descriptor, Parameters: parameters}
	map_err := platform_map(&platform)
	if map_err != nil {
		platform_unmap(&platform)
		syscall.Close(descriptor)
		return platform_scheduler{}, map_err
	}
	return platform, nil
}

// Platform map maps the submission ring, completion ring, and SQE array described by setup.
func platform_map(platform *platform_scheduler) (err error) {
	submission_size := int(platform.Parameters.Submission.Array) +
		int(platform.Parameters.Submission_Entries)*4
	completion_size := int(platform.Parameters.Completion.Completions) +
		int(platform.Parameters.Completion_Entries)*16
	features := platform.Parameters.Features
	platform.Single_Mapping = features&kernel_ring_feature_single_mapping != 0
	if platform.Single_Mapping {
		mapping_size := submission_size
		if completion_size > mapping_size {
			mapping_size = completion_size
		}
		mapping, map_err := platform_mmap(&platform_mmap_input{
			Descriptor: platform.Descriptor,
			Offset:     kernel_ring_submission_offset,
			Size:       mapping_size,
		})
		if map_err != nil {
			return map_err
		}
		platform.Submission_Ring = mapping
		platform.Completion_Ring = mapping
	} else {
		submission, map_err := platform_mmap(&platform_mmap_input{
			Descriptor: platform.Descriptor,
			Offset:     kernel_ring_submission_offset,
			Size:       submission_size,
		})
		if map_err != nil {
			return map_err
		}
		platform.Submission_Ring = submission
		completion, completion_err := platform_mmap(&platform_mmap_input{
			Descriptor: platform.Descriptor,
			Offset:     kernel_ring_completion_offset,
			Size:       completion_size,
		})
		if completion_err != nil {
			return completion_err
		}
		platform.Completion_Ring = completion
	}
	entries_size := int(platform.Parameters.Submission_Entries) * 64
	entries, entries_err := platform_mmap(&platform_mmap_input{
		Descriptor: platform.Descriptor,
		Offset:     kernel_ring_entries_offset,
		Size:       entries_size,
	})
	if entries_err != nil {
		return entries_err
	}
	platform.Entries = entries
	return nil
}

// Platform mmap input identifies one io_uring memory mapping.
type platform_mmap_input struct {
	Descriptor int
	Offset     int64
	Size       int
}

// Platform mmap maps one io_uring region as shared read-write memory.
func platform_mmap(input *platform_mmap_input) (memory []byte, err error) {
	return syscall.Mmap(input.Descriptor, input.Offset, input.Size,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED|syscall.MAP_POPULATE)
}

// Platform unmap releases every successfully mapped io_uring region exactly once.
func platform_unmap(platform *platform_scheduler) {
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
func platform_deinitialize(state *operating_system) {
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
	state *operating_system, operation *operating_system_operation,
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
	state *operating_system, operation *operating_system_operation,
) (err error) {
	bounded := &operating_system_bounded_operation{Operation: operation}
	deadline := &operating_system_operation{
		Completion: &sharedio.Completion{},
		Kind:       operating_system_operation_bounded_deadline,
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
	entries[0].Flags |= kernel_ring_submission_link
	platform_prepare_entry(entries[1], deadline)
	state.Platform.IO_Queued += 2
	return nil
}

// Platform submit registered enqueues the same pinned operation for an EINTR/EAGAIN retry.
func platform_submit_registered(
	state *operating_system, operation *operating_system_operation,
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
func platform_pin(operation *operating_system_operation) (err error) {
	if operation.Pinned {
		return nil
	}
	if len(operation.Buffer) > 0 {
		operation.Pinner.Pin(&operation.Buffer[0])
	}
	if len(operation.File_Path) > 0 {
		operation.Pinner.Pin(&operation.File_Path[0])
	}
	if operation.Kind == operating_system_operation_connect {
		platform_address(operation)
		operation.Pinner.Pin(&operation.Socket_Address[0])
	}
	if operation.Kind == operating_system_operation_timeout {
		operation.Pinner.Pin(&operation.Timespec)
	}
	if operation.Kind == operating_system_operation_bounded_deadline {
		operation.Pinner.Pin(&operation.Timespec)
	}
	if operation.Kind == operating_system_operation_event {
		operation.Pinner.Pin(&operation.Event_Value)
	}
	if operation.Kind == operating_system_operation_statx {
		operation.Pinner.Pin(operation.Statx_Result)
	}
	operation.Pinned = true
	return nil
}

// Platform address encodes shared/io.Address as sockaddr_in or sockaddr_in6.
func platform_address(operation *operating_system_operation) {
	for index := range operation.Socket_Address {
		operation.Socket_Address[index] = 0
	}
	binary.LittleEndian.PutUint16(operation.Socket_Address[0:2], uint16(
		socket_family(operation.Address.Family)))
	binary.BigEndian.PutUint16(operation.Socket_Address[2:4], operation.Address.Port)
	if operation.Address.Family == sharedio.FAMILY_IPV4 {
		copy(operation.Socket_Address[4:8], operation.Address.IP[:4])
		operation.Socket_Address_Size = 16
		return
	}
	copy(operation.Socket_Address[8:24], operation.Address.IP[:])
	operation.Socket_Address_Size = 28
}

// Platform get entry reserves one SQE, flushing a full submission queue before retrying exactly
// as TigerBeetle enqueue does in io/linux.zig:218-239.
func platform_get_entry(
	state *operating_system,
) (entry *kernel_submission_entry, err error) {
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
	state *operating_system, count int,
) (entries []*kernel_submission_entry, err error) {
	if platform_entries_available(&state.Platform) < count {
		flush_err := platform_enter(state, 0, 0)
		if flush_err != nil {
			return nil, flush_err
		}
	}
	if platform_entries_available(&state.Platform) < count {
		return nil, errors.New("io: io_uring cannot reserve a bounded operation chain")
	}
	entries = make([]*kernel_submission_entry, count)
	for index := 0; index < count; index++ {
		entries[index] = platform_reserve_entry(&state.Platform)
		invariant.Always(entries[index] != nil,
			"A preflighted bounded operation chain reserves every SQE.")
	}
	return entries, nil
}

// Platform entries available reports private SQ capacity not yet consumed by the kernel.
func platform_entries_available(platform *platform_scheduler) (count int) {
	head := atomic.LoadUint32(platform_uint32(platform.Submission_Ring,
		platform.Parameters.Submission.Head))
	used := platform.Submission_Tail - head
	return int(platform.Parameters.Submission_Entries - used)
}

// Platform reserve entry advances only the private SQE tail. The shared kernel tail is published
// by platform_publish after every SQE is fully initialized, matching io_uring.flush_sq and keeping
// SQPOLL from observing a partial entry.
func platform_reserve_entry(platform *platform_scheduler) (entry *kernel_submission_entry) {
	head := atomic.LoadUint32(platform_uint32(platform.Submission_Ring,
		platform.Parameters.Submission.Head))
	tail := platform.Submission_Tail
	if tail-head >= platform.Parameters.Submission_Entries {
		return nil
	}
	mask := *platform_uint32(platform.Submission_Ring,
		platform.Parameters.Submission.Ring_Mask)
	index := tail & mask
	entry = (*kernel_submission_entry)(unsafe.Pointer(&platform.Entries[index*64]))
	*entry = kernel_submission_entry{}
	platform.Submission_Tail = tail + 1
	return entry
}

// Platform publish copies every fully prepared private SQE index into the shared submission array
// and releases the shared tail to the kernel in one final atomic store.
func platform_publish(platform *platform_scheduler) (published int) {
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
	entry *kernel_submission_entry, operation *operating_system_operation,
) {
	entry.Descriptor = int32(operation.Descriptor)
	entry.User_Data = operation.Identifier
	switch operation.Kind {
	case operating_system_operation_accept:
		entry.Opcode = kernel_ring_operation_accept
		entry.Operation_Flags = syscall.SOCK_CLOEXEC | syscall.SOCK_NONBLOCK
	case operating_system_operation_close:
		entry.Opcode = kernel_ring_operation_close
	case operating_system_operation_connect:
		entry.Opcode = kernel_ring_operation_connect
		entry.Address = uint64(uintptr(unsafe.Pointer(&operation.Socket_Address[0])))
		entry.Offset = uint64(operation.Socket_Address_Size)
	case operating_system_operation_read:
		platform_prepare_buffer(entry, operation, kernel_ring_operation_read)
		entry.Offset = operation.Offset
	case operating_system_operation_receive:
		platform_prepare_buffer(entry, operation, kernel_ring_operation_receive)
	case operating_system_operation_send:
		platform_prepare_buffer(entry, operation, kernel_ring_operation_send)
		entry.Operation_Flags = kernel_message_no_signal
	case operating_system_operation_timeout:
		entry.Opcode = kernel_ring_operation_timeout
		entry.Address = uint64(uintptr(unsafe.Pointer(&operation.Timespec)))
		entry.Count = 1
	case operating_system_operation_bounded_deadline:
		entry.Opcode = kernel_ring_operation_link_timeout
		entry.Address = uint64(uintptr(unsafe.Pointer(&operation.Timespec)))
		entry.Count = 1
	case operating_system_operation_write:
		platform_prepare_buffer(entry, operation, kernel_ring_operation_write)
		entry.Offset = operation.Offset
	case operating_system_operation_fsync:
		entry.Opcode = kernel_ring_operation_fsync
	case operating_system_operation_open_at:
		entry.Opcode = kernel_ring_operation_open_at
		entry.Address = uint64(uintptr(unsafe.Pointer(&operation.File_Path[0])))
		entry.Count = operation.Open_Options.Mode
		entry.Operation_Flags = uint32(platform_open_flags(operation.Open_Options))
	case operating_system_operation_event:
		entry.Opcode = kernel_ring_operation_read
		entry.Address = uint64(uintptr(unsafe.Pointer(&operation.Event_Value)))
		entry.Count = uint32(unsafe.Sizeof(operation.Event_Value))
	case operating_system_operation_statx:
		entry.Opcode = kernel_ring_operation_statx
		entry.Address = uint64(uintptr(unsafe.Pointer(&operation.File_Path[0])))
		entry.Offset = uint64(uintptr(unsafe.Pointer(operation.Statx_Result)))
		entry.Count = operation.Statx_Mask
		entry.Operation_Flags = operation.Statx_Flags
	}
}

// Returns Linux AT_FDCWD for TigerBeetle IO.openat.
func platform_current_directory() (descriptor int) { return -100 }

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
	return flags
}

// Opens Linux eventfd with CLOEXEC exactly as TigerBeetle IO.open_event.
func platform_event_open(state *operating_system) (event sharedio.Event, err error) {
	result, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, 0, syscall.O_CLOEXEC, 0)
	if errno != 0 {
		return 0, errno
	}
	return sharedio.Event(result), nil
}

// Arms Linux Event through the ordinary io_uring read path.
func platform_event_listen(
	state *operating_system, operation *operating_system_operation,
) (err error) {
	return platform_submit(state, operation)
}

// Writes one eventfd notification; identifier is used by Darwin and intentionally irrelevant on
// Linux, matching third-party/tigerbeetle/src/io/linux.zig:1329-1337.
func platform_event_trigger(
	state *operating_system, event sharedio.Event, _ uint64,
) {
	buffer := [8]byte{}
	binary.LittleEndian.PutUint64(buffer[:], 1)
	count, write_err := syscall.Write(int(event), buffer[:])
	for write_err == syscall.EINTR {
		count, write_err = syscall.Write(int(event), buffer[:])
	}
	invariant.Always(write_err == nil, "Triggering an eventfd Event succeeds.")
	invariant.Always(count == len(buffer), "Triggering an eventfd writes one uint64.")
}

// Closes Linux eventfd after its io_uring read listener has drained.
func platform_event_close(state *operating_system, event sharedio.Event) {
	close_err := syscall.Close(int(event))
	invariant.Always(close_err == nil, "Closing an eventfd Event succeeds.")
}

// Platform prepare buffer fills the shared read, write, recv, and send SQE fields.
func platform_prepare_buffer(
	entry *kernel_submission_entry, operation *operating_system_operation, opcode uint8,
) {
	entry.Opcode = opcode
	entry.Count = uint32(len(operation.Buffer))
	if len(operation.Buffer) > 0 {
		entry.Address = uint64(uintptr(unsafe.Pointer(&operation.Buffer[0])))
	}
}

// Platform run flushes submissions with EXT_ARG, optionally waits for one CQE, then retires every
// available completion (third-party/tigerbeetle/src/io/linux.zig:70-209).
func platform_run(state *operating_system, wait time.Moment) (err error) {
	retry_err := platform_retry_bounded_operations(state)
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
	return platform_retry_bounded_operations(state)
}

// Platform enter submits queued SQEs and uses IORING_ENTER_EXT_ARG for bounded or unbounded waits.
func platform_enter(
	state *operating_system, wait_count uint32, wait time.Moment,
) (err error) {
	platform_publish(&state.Platform)
	if state.Platform.IO_Published == 0 {
		if wait_count == 0 {
			return nil
		}
	}
	argument := kernel_enter_argument{Signal_Mask_Size: 8}
	span := kernel_timespec{}
	if wait_count > 0 {
		if wait > 0 {
			span.Seconds = int64(wait) / 1_000_000_000
			span.Nanoseconds = int64(wait) % 1_000_000_000
			argument.Timespec = uint64(uintptr(unsafe.Pointer(&span)))
		}
	}
	flags := uint32(kernel_ring_enter_extended_argument)
	if wait_count > 0 {
		flags |= kernel_ring_enter_get_events
	}
	if state.Platform.Parameters.Flags&kernel_ring_setup_io_poll != 0 {
		flags |= kernel_ring_enter_get_events
	}
	if state.Platform.Parameters.Flags&kernel_ring_setup_submission_poll != 0 {
		ring_flags := atomic.LoadUint32(platform_uint32(
			state.Platform.Submission_Ring, state.Platform.Parameters.Submission.Flags,
		))
		if ring_flags&kernel_ring_submission_needs_wakeup != 0 {
			flags |= kernel_ring_enter_submission_wakeup
		} else if wait_count == 0 {
			platform_account_submitted(state, state.Platform.IO_Published)
			return nil
		}
	}
	for retry := true; retry; {
		to_submit := state.Platform.IO_Published
		result, _, errno := syscall.Syscall6(
			kernel_ring_enter_call, uintptr(state.Platform.Descriptor),
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
func platform_enter_retry(state *operating_system) (err error) {
	wait_err := platform_wait_one(state)
	if wait_err != nil {
		return wait_err
	}
	return platform_drain(state)
}

// Platform account submitted moves SQEs from the queued/published state into kernel ownership.
func platform_account_submitted(state *operating_system, submitted int) {
	invariant.Always(submitted <= state.Platform.IO_Published,
		"io_uring never reports more submissions than were published.")
	state.Platform.IO_Queued -= submitted
	state.Platform.IO_Published -= submitted
	state.Platform.IO_In_Kernel += submitted
}

// Platform wait one mirrors TigerBeetle's completion-queue recovery: wait for one CQE without
// submitting again, then let the caller copy and retire completions before retrying submissions.
func platform_wait_one(state *operating_system) (err error) {
	argument := kernel_enter_argument{Signal_Mask_Size: 8}
	flags := uintptr(kernel_ring_enter_extended_argument | kernel_ring_enter_get_events)
	for retry := true; retry; {
		_, _, errno := syscall.Syscall6(
			kernel_ring_enter_call, uintptr(state.Platform.Descriptor), 0, 1, flags,
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
func platform_drain(state *operating_system) (err error) {
	entries := [256]kernel_completion_entry{}
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
	state *operating_system, entries []kernel_completion_entry,
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
		entries[count] = *(*kernel_completion_entry)(unsafe.Pointer(
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
func platform_flush_submissions(state *operating_system) (err error) {
	for flush := true; flush; {
		retry_err := platform_retry_bounded_operations(state)
		if retry_err != nil {
			return retry_err
		}
		enter_err := platform_enter(state, 0, 0)
		if enter_err != nil {
			return enter_err
		}
		if len(state.Platform.Bounded_Backlog) == 0 {
			return nil
		}
	}
	return nil
}

// Platform complete entry handles one CQE and preserves an operation's registration on retry.
func platform_complete_entry(
	state *operating_system, entry kernel_completion_entry,
) (err error) {
	operation := state.Operations[entry.User_Data]
	if operation == nil {
		return nil
	}
	if operation.Bounded != nil {
		return platform_complete_bounded_entry(state, operation, entry.Result)
	}
	if operating_system_retryable_result(operation, entry.Result) {
		return operating_system_operation_retry(state, operation)
	}
	result, operation_err := operating_system_translate_result(operation, entry.Result)
	operating_system_operation_complete(state, operation, result, operation_err)
	return nil
}

// Platform complete bounded entry joins both sides of a linked deadline. Deadline expiry wins a
// simultaneous primary result; an accepted descriptor is closed before delivery.
func platform_complete_bounded_entry(
	state *operating_system, operation *operating_system_operation, result int32,
) (err error) {
	bounded := operation.Bounded
	if operation.Kind == operating_system_operation_bounded_deadline {
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
		if primary.Kind == operating_system_operation_accept {
			if bounded.Operation_Result >= 0 {
				close_err := socket_close(int(bounded.Operation_Result))
				invariant.Always(close_err == nil,
					"An accepted descriptor that loses the deadline tie "+
						"closes immediately.")
			}
		}
		operating_system_operation_complete(
			state, primary, -1, sharedio.Deadline_Exceeded,
		)
		return nil
	}
	if operating_system_retryable_result(primary, bounded.Operation_Result) {
		state.Platform.Bounded_Backlog = append(state.Platform.Bounded_Backlog, primary)
		return nil
	}
	translated, operation_err := operating_system_translate_result(
		primary, bounded.Operation_Result,
	)
	operating_system_operation_complete(state, primary, translated, operation_err)
	return nil
}

// Platform retry bounded operations rearms interrupted Accept and Connect chains outside the CQE
// drain call graph, preserving their original absolute deadline without recursive ring entry.
func platform_retry_bounded_operations(state *operating_system) (err error) {
	for len(state.Platform.Bounded_Backlog) > 0 {
		operation := state.Platform.Bounded_Backlog[0]
		budget := operation.Deadline - state.Host.Now_Monotonic()
		if budget <= 0 {
			state.Platform.Bounded_Backlog = state.Platform.Bounded_Backlog[1:]
			operating_system_operation_complete(
				state, operation, -1, sharedio.Deadline_Exceeded,
			)
			continue
		}
		operation.Deadline_Span = operating_system_timeout_span(time.Duration(budget))
		submit_err := platform_submit_bounded_operation(state, operation)
		if submit_err != nil {
			return submit_err
		}
		state.Platform.Bounded_Backlog = state.Platform.Bounded_Backlog[1:]
	}
	return nil
}

// Platform expire operation is implemented by Linux's linked timeout SQE, so the common
// user-space expiry pass has nothing to remove.
func platform_expire_operation(
	state *operating_system, operation *operating_system_operation,
) (err error) {
	return nil
}

// Platform in flight reports work queued to or owned by io_uring.
func platform_in_flight(state *operating_system) (in_flight bool) {
	if len(state.Platform.Bounded_Backlog) > 0 {
		return true
	}
	return state.Platform.IO_Queued > 0 || state.Platform.IO_In_Kernel > 0
}

// Platform counts returns Linux's exact queued and in-kernel census.
func platform_counts(state *operating_system) (backlog int, inflight int, queued int, kernel int) {
	return len(state.Platform.Bounded_Backlog), 0,
		state.Platform.IO_Queued, state.Platform.IO_In_Kernel
}

// Platform uint32 addresses one aligned uint32 field inside an io_uring mapping.
func platform_uint32(memory []byte, offset uint32) (value *uint32) {
	return (*uint32)(unsafe.Pointer(&memory[offset]))
}
