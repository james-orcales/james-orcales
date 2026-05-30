//go:build unix

package io

import (
	"syscall"

	"github.com/james-orcales/james-orcales/golang_snacks/io"
)

// Reads up to len(buffer) bytes from file at offset via the pread syscall — the raw
// positioned read TigerBeetle's posix backend uses.
func read_at(file io.File, buffer []byte, offset int64) (count int, err error) {
	return syscall.Pread(int(file), buffer, offset)
}

// Writes buffer to file at offset via the pwrite syscall.
func write_at(file io.File, buffer []byte, offset int64) (count int, err error) {
	return syscall.Pwrite(int(file), buffer, offset)
}
