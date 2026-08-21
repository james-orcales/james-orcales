//go:build linux && amd64

package nbio

import (
	"unsafe"

	"local/james-orcales/shared/simulation/time"
)

// STATX_BASIC_STATS request basic Linux statx fields.
const STATX_BASIC_STATS uint32 = 0x7ff

// This bound keep Go struct layout equal to Linux struct statx.
const STATX_RESERVED_VALUES = 12

// Statx_Timestamp is stable UAPI layout of Linux struct statx_timestamp.
type Statx_Timestamp struct {
	// Seconds is seconds since Unix epoch.
	Seconds int64
	// Nanoseconds is sub-second component.
	Nanoseconds uint32
	// Reserved keep kernel UAPI layout.
	Reserved int32
}

// Statx is stable 256-byte UAPI layout of Linux struct statx.
type Statx struct {
	// Mask report which fields kernel filled.
	Mask uint32
	// Block_Size is preferred IO block size.
	Block_Size uint32
	// Attributes report filesystem attributes.
	Attributes uint64
	// Link_Count is hard-link count.
	Link_Count uint32
	// User_Identifier own inode.
	User_Identifier uint32
	// Group_Identifier own inode.
	Group_Identifier uint32
	// Mode is file type and permissions.
	Mode uint16
	// Reserved_Zero keep kernel UAPI layout.
	Reserved_Zero uint16
	// Inode is inode number.
	Inode uint64
	// Size is file length in bytes.
	Size uint64
	// Blocks is allocated 512-byte block count.
	Blocks uint64
	// Attributes_Mask report supported attribute bits.
	Attributes_Mask uint64
	// Access_Time is last-access timestamp.
	Access_Time Statx_Timestamp
	// Birth_Time is creation timestamp.
	Birth_Time Statx_Timestamp
	// Change_Time is metadata-change timestamp.
	Change_Time Statx_Timestamp
	// Modify_Time is data-modification timestamp.
	Modify_Time Statx_Timestamp
	// Device_Major is represented device major number.
	Device_Major uint32
	// Device_Minor is represented device minor number.
	Device_Minor uint32
	// Filesystem_Major is containing filesystem major number.
	Filesystem_Major uint32
	// Filesystem_Minor is containing filesystem minor number.
	Filesystem_Minor uint32
	// Mount_Identifier identify mount.
	Mount_Identifier uint64
	// Direct_IO_Memory_Alignment is direct-IO memory alignment.
	Direct_IO_Memory_Alignment uint32
	// Direct_IO_Offset_Alignment is direct-IO offset alignment.
	Direct_IO_Offset_Alignment uint32
	// Reserved keep remainder of kernel UAPI layout.
	Reserved [STATX_RESERVED_VALUES]uint64
}

// Platform_IO is Linux-only surface.
type Platform_IO struct {
	// State remains caller-owned while static Statx procedure borrows it.
	State unsafe.Pointer
	// Statx asynchronously fill result from Linux IORING_OP_STATX.
	Statx_Procedure func(
		state unsafe.Pointer, completion *time.Completion, directory File, file_path string,
		flags uint32, mask uint32, result *Statx, callback time.Callback,
	)
}

// Platform_Statx preserves callback-last submit shape while state remains explicit.
func Platform_Statx(
	platform Platform_IO,
	completion *time.Completion, directory File, file_path string,
	flags uint32, mask uint32, result *Statx, callback time.Callback,
) {
	platform.Statx_Procedure(
		platform.State, completion, directory, file_path, flags, mask, result, callback,
	)
}

// Wire Linux simulator statx counterpart over its deterministic in-memory filesystem.
func sim_wire_platform(state *Sim, loop *IO) {
	loop.Platform_IO.State = unsafe.Pointer(state)
	loop.Statx_Procedure = sim_statx
}

func sim_statx(
	state_pointer unsafe.Pointer, completion *time.Completion, directory File,
	file_path string, _ uint32, mask uint32, result *Statx, callback time.Callback,
) {
	state := (*Sim)(state_pointer)
	operation := sim_operation_acquire(state, completion, SIM_OPERATION_KIND_STATX, callback)
	operation.Directory = directory
	operation.File_Path = file_path
	operation.Mask = mask
	operation.Result = unsafe.Pointer(result)
	sim_operation_submit(operation, completion, sim_latency(state))
}

func sim_statx_operation_complete(operation *Sim_Operation) (data int, err error) {
	if operation.Directory != DIRECTORY_CURRENT {
		return 0, sim_not_a_directory
	}
	node_index, found := sim_resolve(operation.State, operation.File_Path)
	if !found {
		return 0, sim_file_absent
	}
	result := (*Statx)(operation.Result)
	result.Mask = operation.Mask
	result.Size = uint64(operation.State.Nodes[node_index].Contents_Count)
	return 0, nil
}
