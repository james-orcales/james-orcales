// Package main is the shell_sdk composition root: the one place that binds real
// stdin, the filesystem, and the terminal, then hands them to the pure library.
// The -install mode is handled inside the library; main only binds os.Symlink.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	shell_sdk "local/james-orcales/shell_sdk/internal"
)

// The cap on a single stdin read, bounding memory on a large pipe.
const MAIN_STDIN_BYTES_MAX = 256 << 20

// The cap on a single file read, bounding memory on a huge file.
const MAIN_FILE_BYTES_MAX = 256 << 20

// The chunk size for the streaming stdin read.
const MAIN_STDIN_CHUNK = 64 << 10

func main() {
	os.Exit(shell_sdk.Main(&shell_sdk.Main_Input{
		Arguments:          os.Args,
		Output:             os.Stdout,
		Error_Output:       os.Stderr,
		Read_Stdin:         main_read_stdin,
		Read_File:          main_read_file,
		Stdout_Is_Terminal: main_is_terminal(),
		Link:               main_link,
	}))
}

// Reports whether standard output is a terminal, so the tail verb renders a
// table rather than the binary wire format.
func main_is_terminal() (is_terminal bool) {
	information, stat_err := os.Stdout.Stat()
	if stat_err != nil {
		return false
	}
	return information.Mode()&os.ModeCharDevice != 0
}

// Reads standard input in bounded chunks, erroring when it exceeds the byte cap so a
// truncated value never parses silently downstream. The low-level Read is the bounded
// primitive; io.ReadAll is banned. Reading one chunk past the cap is what lets the
// overflow be detected rather than silently dropped.
func main_read_stdin() (data []byte, err error) {
	data = []byte{}
	chunk := make([]byte, MAIN_STDIN_CHUNK)
	for len(data) <= MAIN_STDIN_BYTES_MAX {
		count, read_err := os.Stdin.Read(chunk)
		if count > 0 {
			data = append(data, chunk[:count]...)
		}
		if read_err != nil {
			break
		}
	}
	if len(data) > MAIN_STDIN_BYTES_MAX {
		return nil, fmt.Errorf("input exceeds %d bytes", MAIN_STDIN_BYTES_MAX)
	}
	return data, nil
}

// Reads up to MAIN_FILE_BYTES_MAX bytes of a file, capping memory on a huge one.
func main_read_file(name string) (content []byte, err error) {
	file, open_err := os.Open(name)
	if open_err != nil {
		return nil, open_err
	}
	defer file.Close()
	information, stat_err := file.Stat()
	if stat_err != nil {
		return nil, stat_err
	}
	byte_size := information.Size()
	if byte_size > MAIN_FILE_BYTES_MAX {
		return nil, fmt.Errorf("file exceeds %d bytes", MAIN_FILE_BYTES_MAX)
	}
	buffer := make([]byte, byte_size)
	_, read_err := io.ReadFull(io.LimitReader(file, byte_size), buffer)
	if main_read_failed(read_err) {
		return nil, read_err
	}
	return buffer, nil
}

// Reports whether a bounded read ended in a real error, not a short or empty
// file.
func main_read_failed(read_err error) (failed bool) {
	if read_err == nil {
		return false
	}
	if errors.Is(read_err, io.EOF) {
		return false
	}
	return !errors.Is(read_err, io.ErrUnexpectedEOF)
}

// Points a destination path at the running binary, replacing any prior entry so
// re-installing is idempotent. A single binary backs every verb link.
func main_link(destination string) (err error) {
	source, executable_err := os.Executable()
	if executable_err != nil {
		return executable_err
	}
	remove_err := os.Remove(destination)
	if remove_err != nil {
		if !os.IsNotExist(remove_err) {
			return remove_err
		}
	}
	return os.Symlink(source, destination)
}
