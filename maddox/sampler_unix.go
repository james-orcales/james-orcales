//go:build darwin || linux

package main

import (
	"os"

	"local/james-orcales/maddox/internal"
	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/io"
)

// STDERR_BYTES_MAX bounds how much of a failing command's stderr is read back, so
// the read into a fixed buffer satisfies the unbounded-read ban and a runaway
// command cannot exhaust memory.
const STDERR_BYTES_MAX = 65536

// SPAWN_FAILURE_EXIT is the exit code reported when the child could not be spawned at
// all — distinct from any code the child itself could return.
const SPAWN_FAILURE_EXIT = 127

// Argv is a command flattened to the argv a spawn hands to execve.
type Argv []string

// Argv_Invariants bounds the argv word count.
func Argv_Invariants(identifier string, argv Argv) {
	invariant.Range(identifier, len(argv), BOUND_MIN, BOUND_MAX)
}

// Word_Count is how many C strings a spawned array carries, the trailing NULL excluded.
type Word_Count int

// Word_Count_Invariants bounds the count with the command tier's vacuous-min, eager-max
// interval; argv and envp both stay far below it on any real invocation.
func Word_Count_Invariants(identifier string, value Word_Count) {
	invariant.Range(identifier, int(value), BOUND_MIN, BOUND_MAX)
}

// Command_argv flattens a command to argv: the executable followed by its arguments.
func command_argv(command io.Process_Request) (argv Argv) {
	defer func() { Argv_Invariants("command_argv.argv", argv) }()
	argv = make(Argv, 0, 1+len(command.Arguments))
	argv = append(argv, command.Path)
	argv = append(argv, command.Arguments...)
	return argv
}

// Read_captured reads a failed command's redirected stderr back from the capture file
// into a fixed buffer, rewinding first since the child wrote from offset zero.
func read_captured(capture *os.File) (stderr maddox.Captured_Output) {
	defer func() { maddox.Captured_Output_Invariants("read_captured.stderr", stderr) }()
	_, seek_err := capture.Seek(0, 0)
	if seek_err != nil {
		return nil
	}
	buffer := make([]byte, STDERR_BYTES_MAX)
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
