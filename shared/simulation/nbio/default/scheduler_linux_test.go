//go:build linux && amd64

package nbio

import (
	"path/filepath"
	"syscall"
	"testing"
	"unsafe"

	"local/james-orcales/shared/simulation/nbio"
	"local/james-orcales/shared/simulation/time"
	"local/james-orcales/shared/testify"
)

// PLATFORM_TEST_CAPACITY holds both primary and linked-deadline test operations.
const PLATFORM_TEST_CAPACITY = 2 * nbio.IPV4_ADDRESS_BYTES

type platform_test_memory struct {
	State              Operating_System
	Clock              clock_value
	Operations         [PLATFORM_TEST_CAPACITY]Operating_System_Operation
	Operation_Registry [PLATFORM_TEST_CAPACITY]*Operating_System_Operation
	Completed          [PLATFORM_TEST_CAPACITY]*time.Completion
	Retry              [PLATFORM_TEST_CAPACITY]*Operating_System_Operation
}

func platform_test_state(moment time.Monotonic_Moment) (state *Operating_System) {
	memory := &platform_test_memory{Clock: clock_value{Moment: moment}}
	memory.State.Host = clock_value_to_clock(&memory.Clock)
	memory.State.Operation_Memory = memory.Operations[:]
	memory.State.Operations = memory.Operation_Registry[:0]
	memory.State.Completed = memory.Completed[:0]
	memory.State.Platform.Retry_Backlog = memory.Retry[:0]
	return &memory.State
}

// Linux statx must stay in this platform suite because the linter permits one white-box file
// for one build constraint.
func Test_Operating_System_IO_Statx(t *testing.T) {
	path := filepath.Join(t.TempDir(), "statx")
	clock := new_operating_system_clock()
	loop, _, driver := operating_system_loop(t, clock)
	opened := nbio.File(-1)
	var open_completion time.Completion
	nbio.Storage_Open_At(loop.Storage, &open_completion, nbio.DIRECTORY_CURRENT, path,
		nbio.Open_At_Options{
			Access: nbio.OPEN_READ_WRITE, Create: true, Truncate: true, Mode: 0o600,
		}, func(completed *time.Completion) {
			if !testify.No_Error(t, completed.Error) {
				return
			}
			opened = nbio.File(completed.Data)
		})
	testify.True(t,
		operating_system_run_until(
			t, driver, func() (finished bool) { return opened >= 0 },
		))
	written := false
	var write_completion time.Completion
	nbio.Storage_Write(loop.Storage, &write_completion, opened, []byte("hello"), 10,
		REAL_DEADLINE,
		func(completed *time.Completion) {
			testify.No_Error(t, completed.Error)
			written = completed.Data == 5
		})
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return written }))
	self_exec_close(loop, driver, opened)

	status := nbio.Statx{}
	statted := false
	var statx_completion time.Completion
	nbio.Platform_Statx(
		loop.Platform_IO, &statx_completion, nbio.DIRECTORY_CURRENT, path, 0,
		nbio.STATX_BASIC_STATS, &status,
		func(completed *time.Completion) {
			testify.No_Error(t, completed.Error)
			statted = true
		},
	)
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return statted }))
	testify.Equal(t, uint64(15), status.Size)
}

// Test_Operating_System_Statx_Heap_Allocation guards Linux path pinning and CQE retirement.
func Test_Operating_System_Statx_Heap_Allocation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "statx-allocation")
	clock := new_operating_system_clock()
	loop, _, driver := operating_system_loop(t, clock)
	write_file(t, loop, driver, path, []byte("data"))
	harness := platform_statx_allocation_harness{Driver: driver}
	harness.Completion.Backend = unsafe.Pointer(&harness)
	harness.Done = func() (finished bool) { return harness.Called }
	testify.Zero_Allocation(t, func() {
		harness.Called = false
		nbio.Platform_Statx(
			loop.Platform_IO, &harness.Completion, nbio.DIRECTORY_CURRENT, path, 0,
			nbio.STATX_BASIC_STATS, &harness.Result, platform_statx_allocation_callback,
		)
		_, harness.Error = time.Driver_Run_Until(driver, REAL_DEADLINE, harness.Done)
	})
	testify.No_Error(t, harness.Error)
	time.Driver_Deinit(driver)
}

type platform_statx_allocation_harness struct {
	Driver     time.Driver
	Completion time.Completion
	Result     nbio.Statx
	Called     bool
	Done       func() (finished bool)
	Error      error
}

func platform_statx_allocation_callback(completion *time.Completion) {
	harness := (*platform_statx_allocation_harness)(completion.Backend)
	harness.Called = true
}

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

// CQE drain must only queue interrupted operation. Inline retry can recurse when ring is full.
func Test_Platform_Retry_Is_Deferred(t *testing.T) {
	completion := &time.Completion{Armed: true}
	state := platform_test_state(0)
	operation := operating_system_operation_acquire(state, Operating_System_Operation{
		Completion: completion,
		Identifier: 1,
		Kind:       OPERATING_SYSTEM_OPERATION_READ,
		Deliver:    func(_ *time.Completion) {},
	})
	operating_system_operation_register(state, operation)
	err := platform_complete_entry(state, Kernel_Completion_Entry{
		User_Data: 1,
		Result:    -int32(syscall.EINTR),
	})
	testify.No_Error(t, err)
	testify.Equal(t, 1, len(state.Platform.Retry_Backlog))
	testify.True(t, state.Platform.Retry_Backlog[0] == operation)
	testify.True(t, operating_system_operation_find(state, 1) == operation)
	testify.True(t, completion.Armed)
}

// Parked retry keeps original absolute deadline; queue delay never extend operation lifetime.
func Test_Platform_Retry_Uses_Remaining_Deadline(t *testing.T) {
	completion := &time.Completion{Armed: true}
	state := platform_test_state(10)
	operation := operating_system_operation_acquire(state, Operating_System_Operation{
		Completion: completion,
		Identifier: 1,
		Kind:       OPERATING_SYSTEM_OPERATION_READ,
		Deadline:   9,
		Deliver:    func(_ *time.Completion) {},
	})
	operating_system_operation_register(state, operation)
	platform_retry_add(state, operation)
	testify.No_Error(t, platform_retry_operations(state))
	testify.Zero(t, len(state.Platform.Retry_Backlog))
	testify.Zero(t, len(state.Operations))
	testify.Zero(t, completion.Data)
	testify.Error_Is(t, completion.Error, time.Deadline_Exceeded)
	testify.Equal(t, 1, len(state.Completed))
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
		operation := operating_system_operation_acquire(state, Operating_System_Operation{
			Completion: &time.Completion{},
			Kind:       test.Kind,
			Descriptor: 9,
			Buffer:     []byte{1},
			Deadline:   10,
		})
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
		Host: clock_value_to_clock(&clock_value{Moment: 7}),
	}
	deadline := platform_storage_deadline(state, 3*time.NANOSECOND)
	testify.Equal(t, time.Monotonic_Moment(10), deadline)
}

// Build a private two-entry ring, so a test can inspect the linked chain before publication.
func platform_bounded_test_state() (state *Operating_System) {
	const ENTRIES = uint32(2)
	submission := make([]byte, 128)
	*platform_uint32(submission, 8) = ENTRIES - 1
	state = platform_test_state(1)
	state.Platform.Parameters = Kernel_Ring_Parameters{
		Submission_Entries: ENTRIES,
		Submission:         Kernel_Ring_Offsets{Head: 0, Ring_Mask: 8},
	}
	state.Platform.Submission_Ring = submission
	state.Platform.Entries = make([]byte, ENTRIES*64)
	return state
}

// An API timeout that elapses before SQE publication must not fabricate descriptor zero.
func Test_Platform_Expired_Accept_Submission_Yields_No_Descriptor(t *testing.T) {
	completion := &time.Completion{Armed: true}
	delivered := 0
	var delivered_err error
	state := platform_test_state(10)
	operation := operating_system_operation_acquire(state, Operating_System_Operation{
		Completion: completion,
		Kind:       OPERATING_SYSTEM_OPERATION_ACCEPT,
		Descriptor: 9,
		Deadline:   9,
		Deliver: func(completed *time.Completion) {
			delivered = completed.Data
			delivered_err = completed.Error
		},
	})
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
		state := platform_test_state(10)
		operation := operating_system_operation_acquire(state, Operating_System_Operation{
			Completion: completion,
			Kind:       kind,
			Descriptor: 9,
			Buffer:     []byte{1},
			Deadline:   9,
		})
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
	state := platform_test_state(3)
	state.Platform.Parameters = Kernel_Ring_Parameters{
		Submission_Entries: ENTRIES,
		Submission:         Kernel_Ring_Offsets{Head: 0, Ring_Mask: 8},
	}
	state.Platform.Submission_Ring = submission
	state.Platform.Entries = make([]byte, ENTRIES*64)
	(*clock_value)(state.Host.State).Step = time.NANOSECOND
	primary := operating_system_operation_acquire(state, Operating_System_Operation{
		Completion: &time.Completion{},
		Kind:       OPERATING_SYSTEM_OPERATION_ACCEPT,
		Descriptor: 9,
		Deadline:   10,
	})
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
	testify.Equal(t, 1, published)
	testify.Equal(t, uint32(1), *kernel_tail)
	array := (*uint32)(unsafe.Pointer(&submission[array_offset]))
	testify.Zero(t, *array)
}
