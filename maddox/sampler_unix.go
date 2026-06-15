//go:build darwin || linux

package main

import (
	"os"

	"github.com/james-orcales/james-orcales/shared/sh"
)

// Stderr_bytes_max bounds how much of a failing command's stderr is read back, so
// the read into a fixed buffer satisfies the unbounded-read ban and a runaway
// command cannot exhaust memory.
const stderr_bytes_max = 65536

// Spawn_failure_exit is the exit code reported when the child could not be spawned at
// all — distinct from any code the child itself could return.
const spawn_failure_exit = 127

// Command_argv flattens a command to argv: the executable followed by its arguments.
func command_argv(command sh.Command) (argv []string) {
	argv = make([]string, 0, 1+len(command.Arguments))
	argv = append(argv, command.Path)
	argv = append(argv, command.Arguments...)
	return argv
}

// Read_captured reads a failed command's redirected stderr back from the capture file
// into a fixed buffer, rewinding first since the child wrote from offset zero.
func read_captured(capture *os.File) (stderr []byte) {
	_, seek_err := capture.Seek(0, 0)
	if seek_err != nil {
		return nil
	}
	buffer := make([]byte, stderr_bytes_max)
	total := 0
	for total < len(buffer) {
		n, read_err := capture.Read(buffer[total:])
		total += n
		if read_err != nil {
			break
		}
	}
	return buffer[:total]
}
