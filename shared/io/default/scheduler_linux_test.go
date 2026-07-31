//go:build linux && amd64

package io

import (
	"testing"
	"unsafe"

	sharedio "local/james-orcales/shared/io"
)

// Test_Platform_Bounded_Operation_Internal_Completion verifies the synthetic linked deadline owns
// completion storage before registration writes its kernel identifier.
func Test_Platform_Bounded_Operation_Internal_Completion(t *testing.T) {
	const ENTRIES = uint32(2)
	submission := make([]byte, 128)
	*platform_uint32(submission, 8) = ENTRIES - 1
	state := &Operating_System{
		Operations: make(map[uint64]*Operating_System_Operation),
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
		Completion: &sharedio.Completion{},
		Kind:       OPERATING_SYSTEM_OPERATION_ACCEPT,
		Descriptor: 9,
	}
	operating_system_operation_register(state, primary)
	if submit_err := platform_submit_bounded_operation(state, primary); submit_err != nil {
		t.Fatalf("submit bounded operation: %v", submit_err)
	}
	deadline := primary.Bounded.Deadline_Operation
	if deadline.Completion == nil {
		t.Fatal("synthetic deadline completion is nil")
	}
	if deadline.Completion.Kernel_Identifier != deadline.Identifier {
		t.Fatalf("deadline completion identifier = %d, want %d",
			deadline.Completion.Kernel_Identifier, deadline.Identifier)
	}
}

// Test_Platform_Submission_Publication verifies get_sqe keeps a partially prepared SQE private and
// platform_publish exposes it only after preparation, including under IORING_SETUP_SQPOLL.
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
	if entry == nil {
		t.Fatal("reserve returned nil")
	}
	if *kernel_tail != 0 {
		t.Fatalf("kernel tail published before SQE preparation: %d", *kernel_tail)
	}
	entry.Opcode = 99
	published := platform_publish(&platform)
	if published != 1 {
		t.Fatalf("published %d entries, want 1", published)
	}
	if *kernel_tail != 1 {
		t.Fatalf("kernel tail = %d, want 1", *kernel_tail)
	}
	array := (*uint32)(unsafe.Pointer(&submission[array_offset]))
	if *array != 0 {
		t.Fatalf("submission array index = %d, want 0", *array)
	}
}
