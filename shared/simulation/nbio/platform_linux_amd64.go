//go:build linux && amd64

package io

import "local/james-orcales/shared/simulation/time"

// STATX_BASIC_STATS requests TigerBeetle's basic Linux statx fields.
const STATX_BASIC_STATS uint32 = 0x7ff

// This bound keeps the Go struct layout equal to Linux struct statx.
const STATX_RESERVED_VALUES = 12

// Statx_Timestamp is Linux struct statx_timestamp's stable UAPI layout.
type Statx_Timestamp struct {
	// Seconds is seconds since the Unix epoch.
	Seconds int64
	// Nanoseconds is the sub-second component.
	Nanoseconds uint32
	// Reserved preserves the kernel UAPI layout.
	Reserved int32
}

// Statx is Linux struct statx's stable 256-byte UAPI layout.
type Statx struct {
	// Mask reports which fields the kernel filled.
	Mask uint32
	// Block_Size is the preferred IO block size.
	Block_Size uint32
	// Attributes reports filesystem attributes.
	Attributes uint64
	// Link_Count is the hard-link count.
	Link_Count uint32
	// User_Identifier owns the inode.
	User_Identifier uint32
	// Group_Identifier owns the inode.
	Group_Identifier uint32
	// Mode is the file type and permissions.
	Mode uint16
	// Reserved_Zero preserves the kernel UAPI layout.
	Reserved_Zero uint16
	// Inode is the inode number.
	Inode uint64
	// Size is the file length in bytes.
	Size uint64
	// Blocks is the allocated 512-byte block count.
	Blocks uint64
	// Attributes_Mask reports supported attribute bits.
	Attributes_Mask uint64
	// Access_Time is the last-access timestamp.
	Access_Time Statx_Timestamp
	// Birth_Time is the creation timestamp.
	Birth_Time Statx_Timestamp
	// Change_Time is the metadata-change timestamp.
	Change_Time Statx_Timestamp
	// Modify_Time is the data-modification timestamp.
	Modify_Time Statx_Timestamp
	// Device_Major is the represented device major number.
	Device_Major uint32
	// Device_Minor is the represented device minor number.
	Device_Minor uint32
	// Filesystem_Major is the containing filesystem major number.
	Filesystem_Major uint32
	// Filesystem_Minor is the containing filesystem minor number.
	Filesystem_Minor uint32
	// Mount_Identifier identifies the mount.
	Mount_Identifier uint64
	// Direct_IO_Memory_Alignment is direct-IO memory alignment.
	Direct_IO_Memory_Alignment uint32
	// Direct_IO_Offset_Alignment is direct-IO offset alignment.
	Direct_IO_Offset_Alignment uint32
	// Reserved preserves the remainder of the kernel UAPI layout.
	Reserved [STATX_RESERVED_VALUES]uint64
}

// Platform_IO is the Linux-only TigerBeetle surface.
type Platform_IO struct {
	// Statx asynchronously fills result from Linux IORING_OP_STATX.
	Statx func(
		completion *time.Completion, callback time.Timeout_Callback,
		directory File, file_path string,
		flags uint32, mask uint32, result *Statx,
	)
}

// Wires the Linux simulator's statx counterpart over its deterministic in-memory filesystem.
func sim_wire_platform(state *Sim, loop *IO) {
	loop.Statx = func(
		completion *time.Completion, callback time.Timeout_Callback,
		directory File, file_path string,
		flags uint32, mask uint32, result *Statx,
	) {
		sim_submit(state, completion, sim_latency(state), func() {
			if directory != DIRECTORY_CURRENT {
				callback(completion, sim_not_a_directory)
				return
			}
			node, found := sim_resolve(state.Root, file_path)
			if !found {
				callback(completion, sim_file_absent)
				return
			}
			result.Mask = mask
			result.Size = uint64(len(node.Contents))
			callback(completion, nil)
		})
	}
}
