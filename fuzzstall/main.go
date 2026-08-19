// Package main is the fuzzstall command: the composition root over the internal library tier,
// and the one place a binary may bind the real world — lint permits a binary no default tier.
//
// The run starts with os/exec rather than through shared/simulation/os, whose Spawn has no
// production backend (os/default fills the ambient readers alone and leaves that slot nil) and
// whose only kill is a deadline fixed at submit. A stall window slides every time the run finds
// something new, and shared/io has no generic cancel to slide it with.
package main

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	fuzzstall "local/james-orcales/fuzzstall/internal"
	invariant "local/james-orcales/shared/invariant/default"
	time_system "local/james-orcales/shared/simulation/time/default"
)

// PROCESS_GROUP_MIN is the first identifier a started process takes. Zero and below name the
// caller's own group, which a kill must never reach.
const PROCESS_GROUP_MIN = 1

// PROCESS_GROUP_MAX is the Linux pid_max ceiling. Every other host stays below it.
const PROCESS_GROUP_MAX = 4194304

// SIGNAL_EXIT_BASE is the shell's convention for a process a signal killed: 128 plus the
// signal number. A signalled process carries no exit status of its own, thus this is the only
// code there is to report for one. It lives here rather than in the library tier because a
// signal reaches this program alone: nothing below it holds a process.
const SIGNAL_EXIT_BASE = 128

func main() {
	clock, _ := time_system.New_Operating_System_Any_Clock()
	os.Exit(int(fuzzstall.Main(&fuzzstall.Main_Input{
		Arguments:    fuzzstall.Command_Line(os.Args),
		Output:       os.Stdout,
		Error_Output: os.Stderr,
		Now: func() (reading fuzzstall.Reading) {
			return fuzzstall.Reading(clock.Now_Monotonic())
		},
		Start: main_start,
	})))
}

// Process_Group identifies the group the started run leads. The run leads a group of its own
// because a kill must reach the whole tree it spawns: SIGTERM to the toolchain alone leaves
// the test binary and every fuzz worker running, which a measurement of this confirmed.
type Process_Group int

// Process_Group_Invariants rejects a group a kill must never be aimed at.
func Process_Group_Invariants(group Process_Group, namespace invariant.Namespace) {
	invariant.Tree(group, namespace).
		Range_Int(int(group), PROCESS_GROUP_MIN, PROCESS_GROUP_MAX).
		Ensure()
}

// Starts path with arguments as the run to watch. Its standard input and standard error are
// this program's own, so a build failure reads as it would with go test run alone. Its
// standard output is a pipe, because that is the stream go test writes its progress to.
func main_start(
	path fuzzstall.Toolchain, arguments fuzzstall.Arguments,
) (child fuzzstall.Child, err error) {
	defer func() { fuzzstall.Child_Invariants(child, "main_start.child") }()
	fuzzstall.Toolchain_Invariants(path, "main_start.path")
	fuzzstall.Arguments_Invariants(arguments, "main_start.arguments")
	reader, writer, pipe_err := os.Pipe()
	if pipe_err != nil {
		return fuzzstall.Child_Over(), pipe_err
	}
	command := exec.Command(string(path), []string(arguments)...)
	command.Stdin = os.Stdin
	// An *os.File standard output is handed to the child as a descriptor, so os/exec starts
	// no copier goroutine of its own and this program owns the read.
	command.Stdout = writer
	command.Stderr = os.Stderr
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	start_err := command.Start()
	// The parent's own copy of the write end must go, or the read below never reaches the
	// end of the output and the watcher never learns the run is over.
	writer.Close()
	if start_err != nil {
		reader.Close()
		return fuzzstall.Child_Over(), start_err
	}
	// A group led by the run is numbered by the run's own pid.
	group := Process_Group(command.Process.Pid)
	main_forward(group)
	return fuzzstall.Child{
		Next_Output: main_next_output(reader),
		Terminate:   main_reap(group, command, reader),
		Wait:        main_wait(command, reader),
	}, nil
}

// Returns the bounded wait on the run's next output. A pipe is pollable, so a read deadline
// bounds the read in place: the wait needs no second thread to time it, and the whole watch
// runs on one goroutine with no queue between the read and the judgment.
//
// One buffer serves every call because Main echoes and reads a chunk whole before it asks for
// the next one, so no chunk is alive when the read that would overwrite it starts.
func main_next_output(reader *os.File) (next fuzzstall.Next_Output) {
	buffer := make([]byte, fuzzstall.CHUNK_SIZE_MAX)
	return func(deadline fuzzstall.Deadline) (chunk fuzzstall.Chunk, event fuzzstall.Event) {
		reader.SetReadDeadline(time.Now().Add(time.Duration(deadline)))
		read_size, read_err := reader.Read(buffer)
		// Bytes and an error can arrive together, and the bytes are the run's progress —
		// report them now and let the next call answer with the error still standing.
		if read_size > 0 {
			return fuzzstall.Chunk(buffer[:read_size]), fuzzstall.EVENT_OUTPUT
		}
		if errors.Is(read_err, os.ErrDeadlineExceeded) {
			return nil, fuzzstall.EVENT_DEADLINE
		}
		return nil, fuzzstall.EVENT_END
	}
}

// Returns the wait that collects the exit code of a run that ended by itself.
func main_wait(command *exec.Cmd, reader *os.File) (wait fuzzstall.Wait) {
	return func() (status_code fuzzstall.Status_Code) {
		reader.Close()
		command.Wait()
		return main_exit_code(command.ProcessState)
	}
}

// Returns the terminator: it kills the run's whole process group and reaps it, so the run is
// over here rather than orphaned as this program ends. Main calls this or the plain wait and
// never both, thus the run is reaped exactly one time either way.
func main_reap(
	group Process_Group, command *exec.Cmd, reader *os.File,
) (terminate fuzzstall.Terminate) {
	Process_Group_Invariants(group, "main_reap.group")
	return func() {
		// A negated group id is kill(2)'s spelling of "every process in this group". The
		// group outlives its leader, thus this still reaches the test binary `go test`
		// spawned even when the toolchain itself already exited.
		//
		// SIGKILL, not SIGTERM: a measurement of SIGTERM to the toolchain left every fuzz
		// worker alive and orphaned, and a run that found nothing new for the whole window
		// has nothing left to finish anyway.
		syscall.Kill(-int(group), syscall.SIGKILL)
		reader.Close()
		command.Wait()
	}
}

// Forwards an interrupt or a termination aimed at this program on to the run's group. The
// group above is what takes the run out of the terminal's foreground group, thus without this
// a Ctrl-C would kill the wrapper alone and leave the fuzzing orphaned and running. The
// channel is signal.Notify's own API, not a queue this program chose to put in its own path.
func main_forward(group Process_Group) {
	Process_Group_Invariants(group, "main_forward.group")
	signals := make(chan os.Signal, 1)
	signal.Notify(signals,
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	go func() {
		for received := range signals {
			forwarded, ok := received.(syscall.Signal)
			if !ok {
				continue
			}
			syscall.Kill(-int(group), forwarded)
		}
	}()
}

// Reports the exit code of a finished process. A process whose state cannot be read at all
// yields no outcome to report and so answers with this program's own failure code.
func main_exit_code(state *os.ProcessState) (status_code fuzzstall.Status_Code) {
	defer func() {
		fuzzstall.Status_Code_Invariants(status_code, "main_exit_code.status_code")
	}()
	if state == nil {
		return fuzzstall.EXIT_START_FAILURE
	}
	code := state.ExitCode()
	if code >= 0 {
		return fuzzstall.Status_Code(code)
	}
	status, ok := state.Sys().(syscall.WaitStatus)
	if !ok {
		return fuzzstall.EXIT_START_FAILURE
	}
	return SIGNAL_EXIT_BASE + fuzzstall.Status_Code(status.Signal())
}
