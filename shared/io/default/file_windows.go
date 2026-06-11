//go:build windows

package io

import "github.com/james-orcales/james-orcales/shared/io"

// Panics: file IO is not yet supported on Windows. The faithful backend issues
// overlapped ReadFile/WriteFile (TigerBeetle io/windows.zig); this stub keeps the
// package building on Windows until that lands.
func read_at(file io.File, buffer []byte, offset int64) (count int, err error) {
	panic("io: file IO is not yet supported on Windows")
}

// Panics: file IO is not yet supported on Windows.
func write_at(file io.File, buffer []byte, offset int64) (count int, err error) {
	panic("io: file IO is not yet supported on Windows")
}
