//go:build darwin || linux

package main

import (
	"os"

	"github.com/james-orcales/james-orcales/maddox/internal"
	invariant "github.com/james-orcales/james-orcales/shared/invariant/default"
	"github.com/james-orcales/james-orcales/shared/io"
)

// Stderr_bytes_max bounds how much of a failing command's stderr is read back, so
// the read into a fixed buffer satisfies the unbounded-read ban and a runaway
// command cannot exhaust memory.
const stderr_bytes_max = 65536

// Spawn_failure_exit is the exit code reported when the child could not be spawned at
// all — distinct from any code the child itself could return.
const spawn_failure_exit = 127

// Argv is a command flattened to the argv a spawn hands to execve.
type Argv []string

// Argv_Invariants bounds the argv word count.
func Argv_Invariants(argv Argv, namespace invariant.Namespace) {
	invariant.Always(len(argv) <= bound_max, "An argv is at most its max.")
	invariant.Always(len(argv) >= bound_min, "An argv is at least its min.")
	invariant.Always(len(argv) != bound_min, "An argv never reaches its min.")
	invariant.Always(len(argv) != bound_max, "An argv is below its max.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(argv) == 0, "An argv is empty."),
		invariant.Sometimes(len(argv) == 1, "An argv has one."),
		invariant.Sometimes(len(argv) == 2, "An argv has two."),
		invariant.Sometimes(len(argv) == bound_min, "An argv is at its min."),
		invariant.Sometimes(len(argv) == bound_max, "An argv is at its max."),
		invariant.Impossible(
			invariant.Event_True("An argv is empty."),
			invariant.Event_True("An argv has one."),
		),
		invariant.Impossible(
			invariant.Event_True("An argv is empty."),
			invariant.Event_True("An argv has two."),
		),
		invariant.Impossible(
			invariant.Event_True("An argv has one."),
			invariant.Event_True("An argv has two."),
		),
	)
}

// Command_argv flattens a command to argv: the executable followed by its arguments.
func command_argv(command io.Process_Request) (argv Argv) {
	defer func() { Argv_Invariants(argv, "command_argv.argv") }()
	argv = make(Argv, 0, 1+len(command.Arguments))
	argv = append(argv, command.Path)
	argv = append(argv, command.Arguments...)
	return argv
}

// Read_captured reads a failed command's redirected stderr back from the capture file
// into a fixed buffer, rewinding first since the child wrote from offset zero.
func read_captured(capture *os.File) (stderr maddox.Captured_Output) {
	defer func() { maddox.Captured_Output_Invariants(stderr, "read_captured.stderr") }()
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
