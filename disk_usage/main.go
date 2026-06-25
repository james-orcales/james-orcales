// Package main is the disk_usage command: the thin composition root over the internal
// library tier, and the one place allowed to bind the real filesystem and read the
// platform stat fields the library cannot.
package main

import (
	"io/fs"
	"os"
	"syscall"

	disk_usage "local/james-orcales/disk_usage/internal"
)

// A stat block is 512 bytes on every Unix — the unit st_blocks counts and du divides by —
// so a file's disk usage is its block count times this.
const main_block_bytes = 512

func main() {
	os.Exit(disk_usage.Main(&disk_usage.Main_Input{
		Arguments:    os.Args,
		Output:       os.Stdout,
		Error_Output: os.Stderr,
		Open:         func(root string) (file_system fs.FS) { return os.DirFS(root) },
		Disk_Usage:   main_disk_usage,
	}))
}

// Reports a file's physical disk usage from its allocated block count, not its logical
// size, so a sparse or cloned file is charged only the extent it really occupies. The
// device and inode form its identity, letting the library count a hardlink once. When
// the host exposes no stat (any non-Unix file system), it falls back to the logical size
// and a zero identity, which the library reads as "always count".
func main_disk_usage(info fs.FileInfo) (bytes int64, identifier disk_usage.File_Identifier) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return info.Size(), disk_usage.File_Identifier{}
	}
	return stat.Blocks * main_block_bytes, disk_usage.File_Identifier{
		Device: int64(stat.Dev),
		Inode:  stat.Ino,
	}
}
