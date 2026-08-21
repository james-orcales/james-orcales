//go:build linux && amd64

package nbio

import (
	"syscall"
	"testing"
	"unsafe"

	"local/james-orcales/shared/simulation/time"
	"local/james-orcales/shared/testify"
)

// A terminal primary CQE is kernel truth even when the linked timeout also reached ETIME.
func Test_Platform_Bounded_Timeout_Winner(t *testing.T) {
	tests := []struct {
		Name      string
		Operation int32
		Deadline  int32
		Timeout   bool
	}{
		{Name: "success", Operation: 7, Deadline: -int32(syscall.ETIME)},
		{Name: "socket error", Operation: -int32(syscall.ECONNREFUSED),
			Deadline: -int32(syscall.ETIME)},
		{Name: "canceled", Operation: -int32(syscall.ECANCELED),
			Deadline: -int32(syscall.ETIME), Timeout: true},
		{Name: "interrupted", Operation: -int32(syscall.EINTR),
			Deadline: -int32(syscall.ETIME), Timeout: true},
		{Name: "deadline canceled", Operation: -int32(syscall.ECANCELED),
			Deadline: -int32(syscall.ECANCELED)},
	}
	for _, test := range tests {
		got := platform_bounded_timeout_won(test.Operation, test.Deadline)
		testify.Equal(t, test.Timeout, got, test.Name)
	}
}

// Storage needs the same indivisible primary-plus-timeout chain that bounds socket operations.
func Test_Platform_Bounded_Storage_Submissions(t *testing.T) {
	tests := []struct {
		Name   string
		Kind   Operating_System_Operation_Kind
		Opcode uint8
	}{
		{Name: "read", Kind: OPERATING_SYSTEM_OPERATION_READ,
			Opcode: KERNEL_RING_OPERATION_READ},
		{Name: "write", Kind: OPERATING_SYSTEM_OPERATION_WRITE,
			Opcode: KERNEL_RING_OPERATION_WRITE},
		{Name: "fsync", Kind: OPERATING_SYSTEM_OPERATION_FSYNC,
			Opcode: KERNEL_RING_OPERATION_FSYNC},
	}
	for _, test := range tests {
		state := platform_bounded_test_state()
		operation := &Operating_System_Operation{
			Completion: &time.Completion{},
			Kind:       test.Kind,
			Descriptor: 9,
			Buffer:     []byte{1},
			Deadline:   10,
		}
		operating_system_operation_register(state, operation)
		if !testify.No_Error(t, platform_submit_bounded_operation(
			state, operation,
		), test.Name) {
			continue
		}
		primary := (*Kernel_Submission_Entry)(unsafe.Pointer(&state.Platform.Entries[0]))
		deadline := (*Kernel_Submission_Entry)(unsafe.Pointer(&state.Platform.Entries[64]))
		testify.Equal(t, test.Opcode, primary.Opcode, test.Name)
		testify.Not_Zero(t, primary.Flags&KERNEL_RING_SUBMISSION_LINK, test.Name)
		testify.Equal(t, uint8(KERNEL_RING_OPERATION_LINK_TIMEOUT), deadline.Opcode,
			test.Name)
		testify.Zero(t, operating_system_timeout_result(operation), test.Name)
		operation.Bounded.Deadline_Operation.Pinner.Unpin()
	}
}

// The platform derives one absolute moment before scheduler backlog can consume the budget.
func Test_Platform_Storage_Deadline_Linux_Starts_At_Submission(t *testing.T) {
	state := &Operating_System{
		Host: time.Clock{
			Now_Monotonic: func() (moment time.Monotonic_Moment) { return 7 },
		},
	}
	deadline := platform_storage_deadline(state, 3*time.NANOSECOND)
	testify.Equal(t, time.Monotonic_Moment(10), deadline)
}

// Build a private two-entry ring, so a test can inspect the linked chain before publication.
func platform_bounded_test_state() (state *Operating_System) {
	const ENTRIES = uint32(2)
	submission := make([]byte, 128)
	*platform_uint32(submission, 8) = ENTRIES - 1
	return &Operating_System{
		Operations: make(map[uint64]*Operating_System_Operation),
		Host: time.Clock{
			Now_Monotonic: func() (moment time.Monotonic_Moment) { return 1 },
		},
		Platform: Platform_Scheduler{
			Parameters: Kernel_Ring_Parameters{
				Submission_Entries: ENTRIES,
				Submission:         Kernel_Ring_Offsets{Head: 0, Ring_Mask: 8},
			},
			Submission_Ring: submission,
			Entries:         make([]byte, ENTRIES*64),
		},
	}
}

// An API timeout that elapses before SQE publication must not fabricate descriptor zero.
func Test_Platform_Expired_Accept_Submission_Yields_No_Descriptor(t *testing.T) {
	completion := &time.Completion{Armed: true}
	delivered := 0
	var delivered_err error
	operation := &Operating_System_Operation{
		Completion: completion,
		Kind:       OPERATING_SYSTEM_OPERATION_ACCEPT,
		Descriptor: 9,
		Deadline:   9,
		Deliver: func(completed *time.Completion) {
			delivered = completed.Data
			delivered_err = completed.Error
		},
	}
	state := &Operating_System{
		Host: time.Clock{
			Now_Monotonic: func() (moment time.Monotonic_Moment) { return 10 },
		},
		Operations: map[uint64]*Operating_System_Operation{},
	}
	operating_system_operation_submit(state, operation)
	operating_system_flush_completed(state)
	testify.Equal(t, -1, delivered)
	testify.Error_Is(t, delivered_err, time.Deadline_Exceeded)
}

// An elapsed storage budget must stop publication and report no transferred data.
func Test_Platform_Expired_Storage_Submission_Yields_Zero(t *testing.T) {
	for _, kind := range []Operating_System_Operation_Kind{
		OPERATING_SYSTEM_OPERATION_READ,
		OPERATING_SYSTEM_OPERATION_WRITE,
		OPERATING_SYSTEM_OPERATION_FSYNC,
	} {
		completion := &time.Completion{Armed: true}
		operation := &Operating_System_Operation{
			Completion: completion,
			Kind:       kind,
			Descriptor: 9,
			Buffer:     []byte{1},
			Deadline:   9,
		}
		state := &Operating_System{
			Host: time.Clock{
				Now_Monotonic: func() (moment time.Monotonic_Moment) { return 10 },
			},
			Operations: map[uint64]*Operating_System_Operation{},
		}
		operating_system_operation_submit(state, operation)
		testify.Zero(t, completion.Data, kind)
		testify.Error_Is(t, completion.Error, time.Deadline_Exceeded, kind)
	}
}

// Test_Platform_Bounded_Operation_Internal_Completion verify synthetic linked deadline own
// completion storage before registration write its kernel identifier.
func Test_Platform_Bounded_Operation_Internal_Completion(t *testing.T) {
	const ENTRIES = uint32(2)
	submission := make([]byte, 128)
	*platform_uint32(submission, 8) = ENTRIES - 1
	now := time.Monotonic_Moment(3)
	state := &Operating_System{
		Operations: make(map[uint64]*Operating_System_Operation),
		Host: time.Clock{
			Now_Monotonic: func() (moment time.Monotonic_Moment) {
				now++
				return now
			},
		},
		Platform: Platform_Scheduler{
			Parameters: Kernel_Ring_Parameters{
				Submission_Entries: ENTRIES,
				Submission:         Kernel_Ring_Offsets{Head: 0, Ring_Mask: 8},
			},
			Submission_Ring: submission,
			Entries:         make([]byte, ENTRIES*64),
		},
	}
	primary := &Operating_System_Operation{
		Completion: &time.Completion{},
		Kind:       OPERATING_SYSTEM_OPERATION_ACCEPT,
		Descriptor: 9,
		Deadline:   10,
	}
	operating_system_operation_register(state, primary)
	if !testify.No_Error(t, platform_submit_bounded_operation(state, primary)) {
		return
	}
	deadline := primary.Bounded.Deadline_Operation
	if !testify.Not_Nil(t, deadline.Completion) {
		return
	}
	testify.Equal(t, deadline.Identifier, deadline.Completion.Kernel_Identifier)
	testify.Zero(t, deadline.Timespec.Seconds)
	testify.Equal(t, int64(5), deadline.Timespec.Nanoseconds)
}

// Test_Platform_Submission_Publication verify get_sqe keep partly prepared SQE private, and
// platform_publish expose it only after preparation, IORING_SETUP_SQPOLL included.
func Test_Platform_Submission_Publication(t *testing.T) {
	const ENTRIES = uint32(2)
	submission := make([]byte, 128)
	kernel_tail := platform_uint32(submission, 4)
	ring_mask := platform_uint32(submission, 8)
	*ring_mask = ENTRIES - 1
	array_offset := uint32(64)
	platform := Platform_Scheduler{
		Parameters: Kernel_Ring_Parameters{
			Submission_Entries: ENTRIES,
			Submission: Kernel_Ring_Offsets{
				Head: 0, Tail: 4, Ring_Mask: 8, Array: array_offset,
			},
		},
		Submission_Ring: submission,
		Entries:         make([]byte, ENTRIES*64),
	}
	entry := platform_reserve_entry(&platform)
	if !testify.Not_Nil(t, entry) {
		return
	}
	testify.Zero(t, *kernel_tail)
	entry.Opcode = 99
	published := platform_publish(&platform)
	testify.Equal(t, uint32(1), published)
	testify.Equal(t, uint32(1), *kernel_tail)
	array := (*uint32)(unsafe.Pointer(&submission[array_offset]))
	testify.Zero(t, *array)
}
