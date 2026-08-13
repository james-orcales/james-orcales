// Package io is the operating-system backend for shared/io.
//
// FAITHFUL PORT: Darwin follows third-party/tigerbeetle/src/io/darwin.zig and Linux follows
// io/linux.zig. DO NOT DIVERGE. There is no generic Cancel. Descriptor owners perform Shutdown,
// join submitted operations, and then asynchronous Close as in message_bus.zig:1057-1160.
package nbio

import (
	"errors"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	invariant "local/james-orcales/shared/invariant/default"
	stream "local/james-orcales/shared/simulation/io"
	io "local/james-orcales/shared/simulation/nbio"
	sysos "local/james-orcales/shared/simulation/os"
	"local/james-orcales/shared/simulation/time"
)

// Poll_forever, passed as an idle gap, blocks the wait until an event arrives rather than for a
// fixed span — the unbounded wait a run with no deadline needs, so the loop sleeps exactly
// until there is work instead of waking on an interval.
const POLL_FOREVER time.Moment = -1

// Buffers a few pending signals so a burst is not lost between drains.
const SIGNAL_QUEUE_DEPTH = 8

// Caps the idle gap while a signal watcher exists, since a signal does not wake the poll;
// the loop re-checks the signal channel at least this often.
const SIGNAL_POLL_INTERVAL = 10 * time.MILLISECOND

// Holds the host backend's state: the completed queue, Darwin timeout queue, platform scheduler,
// and repository-extension effects which all retire through completed.
type Operating_System struct {
	// Host is the real clock; deadlines and waits are measured against it.
	Host time.Any_Clock
	// Timeouts are pending timer completions ordered by Ready_At, earliest first.
	Timeouts []*time.Completion
	// Completed are completions whose callbacks are ready to run on the next drain.
	Completed []*time.Completion
	// Platform is kqueue on Darwin or io_uring on Linux, created eagerly by the constructor.
	Platform Platform_Scheduler
	// Operations maps integer kernel user_data values to caller-owned completion operations.
	Operations map[uint64]*Operating_System_Operation
	// Next_Identifier is the last non-zero kernel correlation identifier issued.
	Next_Identifier uint64
	// Signals receives OS signals from the os/signal notifier; nil until the first watch.
	Signals chan os.Signal
	// Signal_Waiters are the registered signal watchers, fired one-shot on delivery.
	Signal_Waiters []Signal_Waiter
	// Spawns tracks every child the loop has started and not yet reaped, keyed by process
	// identifier. An entry outlives its caller's completion, because a deadline retires that
	// completion while the child is still running.
	Spawns map[int]*Spawn
	// Extension_Submitted counts repository-extension completions not yet delivered to callers.
	Extension_Submitted int
	// Drive_Active is set while a Run* is driving the loop, so a Run* called from within a
	// completion callback — which would re-enter the driver mid-drain — panics loudly.
	Drive_Active bool
	// Raw_Open records every raw descriptor this backend holds open.
	Raw_Open map[int]bool
}

// One registered signal watcher: the OS signal it awaits, its backend-independent kind,
// and the completion and callback to fire once on delivery.
type Signal_Waiter struct {
	// System is the OS signal this watcher awaits.
	System os.Signal
	// Kind is the backend-independent signal reported to the callback.
	Kind sysos.Signal
	// Completion is the caller-owned completion fired on delivery.
	Completion *time.Completion
	// Callback is the typed callback run with the delivered signal.
	Callback sysos.Signal_Callback
	// Deadline is the finite moment when this repository-extension operation retires.
	Deadline time.Moment
}

// New_Operating_System_IO eagerly creates the TigerBeetle scheduler. Entries is the io_uring
// queue size on Linux and is accepted but ignored by kqueue on Darwin; it must fit TigerBeetle's
// u12 contract. Flags are passed directly to io_uring setup and ignored by Darwin.
// ambient states the process values the host reads back — os/default supplies them — and the
// returned system is that OS completed with the signal watch and the spawn, which retire on
// this backend's queue.
func New_Operating_System_IO(
	host time.Any_Clock, entries uint16, flags uint32, ambient sysos.OS,
) (loop io.IO, pump time.Timeline, driver time.Driver, system sysos.OS, err error) {
	if entries == 0 {
		return io.IO{}, time.Timeline{}, time.Driver{}, sysos.OS{}, errors.New(
			"io: scheduler entries must be in [1, 4095]",
		)
	}
	if entries > 4095 {
		return io.IO{}, time.Timeline{}, time.Driver{}, sysos.OS{}, errors.New(
			"io: scheduler entries must be in [1, 4095]",
		)
	}
	platform, initialize_err := platform_initialize(entries, flags)
	if initialize_err != nil {
		return io.IO{}, time.Timeline{}, time.Driver{}, sysos.OS{}, initialize_err
	}
	state := &Operating_System{
		Host:       host,
		Platform:   platform,
		Operations: map[uint64]*Operating_System_Operation{},
		Raw_Open:   map[int]bool{},
		Spawns:     map[int]*Spawn{},
	}
	operating_system_wire_file(state, &loop)
	operating_system_wire_timer(state, &loop, &pump)
	operating_system_wire_socket(state, &loop)
	system = ambient
	operating_system_wire_effects(state, &system)
	operating_system_wire_platform(state, &loop)
	time.Timeline_Invariants(pump, "new_operating_system_io.pump")
	sysos.OS_Invariants(system, "new_operating_system_io.system")
	return loop, pump, operating_system_to_driver(state), system, nil
}

// Arms a completion through the idle-to-armed lifecycle edge.
func operating_system_submit(completion *time.Completion) {
	original := completion.Self == nil || completion.Self == completion
	invariant.Always(original,
		"A submitted completion is its own original, never a by-value copy.")
	completion.Self = completion
	completion.Next_Tick = false
	time.Completion_Transition(completion, time.COMPLETION_IDLE, time.COMPLETION_ARMED)
}

// Wires the signal watch and the spawn onto the OS surface that owns them. Both retire on the
// same completed queue every IO operation uses, so one timeline holds the whole run.
func operating_system_wire_effects(state *Operating_System, system *sysos.OS) {
	system.Watch_Signal = func(
		completion *time.Completion, callback sysos.Signal_Callback, signal sysos.Signal,
		deadline time.Duration,
	) {
		invariant.Always(deadline > 0, "A signal-watch deadline is positive and finite.")
		state.Extension_Submitted++
		operating_system_submit(completion)
		operating_system_watch_signal(state, completion, func(
			completed *time.Completion, delivered sysos.Signal, watch_err error,
		) {
			state.Extension_Submitted--
			callback(completed, delivered, watch_err)
		}, signal, deadline)
	}
	system.Spawn = func(
		completion *time.Completion, callback sysos.Process_Callback,
		request sysos.Process_Request, deadline time.Duration,
	) {
		invariant.Always(deadline > 0, "A spawn deadline is positive and finite.")
		state.Extension_Submitted++
		operating_system_submit(completion)
		operating_system_spawn(state, completion, func(
			completed *time.Completion, result sysos.Process_Result, spawn_err error,
		) {
			state.Extension_Submitted--
			callback(completed, result, spawn_err)
		}, request, deadline)
	}
}

// Bounds one pipe read, so a chatty child is drained in repeated passes rather than into one
// unbounded allocation.
const PROCESS_PIPE_BYTES = 8192

// Caps the wait for a killed child's pipes to close. A grandchild that inherited the write end
// keeps the pipe open after its parent dies, so the drain needs its own bound.
const PROCESS_CLEANUP_DURATION = 1 * time.SECOND

// One spawned child the loop is tracking. It outlives the caller's completion: a deadline
// retires that completion early, and the exit event still arrives afterward and still has to
// reap. The entry leaves state.Spawns only when its reap finishes.
type Spawn struct {
	// Identifier is the child process id, and its process group id because Setpgid is set.
	Identifier int
	// Completion is the caller-owned completion the result is delivered on.
	Completion *time.Completion
	// Callback is the typed callback run once the result is whole.
	Callback sysos.Process_Callback
	// Started is the moment the child was forked, for the wall-time measurement.
	Started time.Moment
	// Exit_Descriptor is the Linux pidfd polled for exit, or -1 on Darwin.
	Exit_Descriptor int
	// Input_Descriptor is the write end of the child's standard input, -1 once closed.
	Input_Descriptor int
	// Output_Descriptor is the read end of the child's standard output, -1 once closed.
	Output_Descriptor int
	// Error_Descriptor is the read end of the child's standard error, -1 once closed.
	Error_Descriptor int
	// Input is the remaining bytes to write to the child's standard input.
	Input []byte
	// Output_Buffer receives one standard-output pass.
	Output_Buffer []byte
	// Error_Buffer receives one standard-error pass.
	Error_Buffer []byte
	// Result accumulates the captured output, exit code, and usage.
	Result sysos.Process_Result
	// Request retains the caller's live output sinks.
	Request sysos.Process_Request
	// Exit_Completion waits for the child to exit.
	Exit_Completion time.Completion
	// Output_Completion reads one standard-output pass.
	Output_Completion time.Completion
	// Error_Completion reads one standard-error pass.
	Error_Completion time.Completion
	// Input_Completion writes one standard-input pass.
	Input_Completion time.Completion
	// Deadline_Completion retires the caller's completion when the child outlives its deadline.
	Deadline_Completion time.Completion
	// Cleanup_Completion bounds the pipe drain after a deadline kill.
	Cleanup_Completion time.Completion
	// Exited reports the reap finished and Result carries the exit code and usage.
	Exited bool
	// Output_Drained reports the standard-output pipe reached its end.
	Output_Drained bool
	// Error_Drained reports the standard-error pipe reached its end.
	Error_Drained bool
	// Expired reports a deadline won, so the result carries Deadline_Exceeded.
	Expired bool
	// Cleanup_Armed reports the bounded pipe drain is already running, so neither the child's
	// exit nor the deadline arms a second one.
	Cleanup_Armed bool
	// Delivered reports the caller's completion already retired.
	Delivered bool
}

// Starts request's command and arms every operation that finishes it: one read for each output
// pipe, one write for the input, the exit watch, and the deadline. Nothing runs off the loop
// thread.
func operating_system_spawn(
	state *Operating_System, completion *time.Completion,
	callback sysos.Process_Callback, request sysos.Process_Request, deadline time.Duration,
) {
	spawn, start_err := process_start(state, request)
	if start_err != nil {
		completion.Callback = func() {
			callback(completion, sysos.Process_Result{}, start_err)
		}
		state.Completed = append(state.Completed, completion)
		return
	}
	spawn.Completion = completion
	spawn.Callback = callback
	spawn.Request = request
	spawn.Started = state.Host.Now_Monotonic()
	state.Spawns[spawn.Identifier] = spawn
	process_arm_pipes(state)
	process_watch_exit(state, spawn)
	operating_system_submit(&spawn.Deadline_Completion)
	operating_system_timeout(state, state.Host, &spawn.Deadline_Completion, func(
		_ *time.Completion, _ error,
	) {
		process_expire(state, spawn)
	}, deadline)
}

// Forks the child with its three pipes and returns the tracking entry. Every descriptor is
// released on a failure part-way through, so a failed start leaks nothing.
func process_start(
	state *Operating_System, request sysos.Process_Request,
) (spawn *Spawn, err error) {
	path, path_err := executable_path(request.Path, operating_system_search_path(request))
	if path_err != nil {
		return nil, path_err
	}
	spawn, child, pipe_err := process_pipes(request)
	if pipe_err != nil {
		return nil, pipe_err
	}
	attributes := &syscall.ProcAttr{
		Dir:   request.Working_Directory,
		Env:   request.Environment,
		Files: child,
		Sys:   process_attributes(spawn),
	}
	identifier, _, start_err := syscall.StartProcess(
		path, process_argv(path, request.Arguments), attributes,
	)
	// The child holds its own copies once StartProcess returns, so the parent releases the
	// three ends it handed over whether or not the fork succeeded.
	for _, descriptor := range child {
		syscall.Close(int(descriptor))
	}
	if start_err != nil {
		process_close_pipes(spawn)
		return nil, start_err
	}
	spawn.Identifier = identifier
	if ready_err := process_watch_ready(spawn); ready_err != nil {
		syscall.Kill(-identifier, syscall.SIGKILL)
		process_close_pipes(spawn)
		process_reap(identifier)
		return nil, ready_err
	}
	return spawn, nil
}

// Creates the child's three pipes, returning the tracking entry that holds the ends the loop
// keeps and the three ends the child receives as its standard descriptors.
//
// The kept ends take close-on-exec here, BEFORE the fork. StartProcess dups only the Files
// entries and leaves every other open descriptor to the exec, so a kept end without the flag
// reaches the child. The child would then hold a writer for its own standard input, and closing
// the parent's end would never give it end-of-file.
func process_pipes(
	request sysos.Process_Request,
) (spawn *Spawn, child []uintptr, err error) {
	input_read, input_write, input_err := pipe_open()
	if input_err != nil {
		return nil, nil, input_err
	}
	output_read, output_write, output_err := pipe_open()
	if output_err != nil {
		syscall.Close(input_read)
		syscall.Close(input_write)
		return nil, nil, output_err
	}
	error_read, error_write, error_err := pipe_open()
	if error_err != nil {
		syscall.Close(input_read)
		syscall.Close(input_write)
		syscall.Close(output_read)
		syscall.Close(output_write)
		return nil, nil, error_err
	}
	spawn = &Spawn{
		Exit_Descriptor:   -1,
		Input_Descriptor:  input_write,
		Output_Descriptor: output_read,
		Error_Descriptor:  error_read,
		Input:             request.Input,
		Output_Buffer:     make([]byte, PROCESS_PIPE_BYTES),
		Error_Buffer:      make([]byte, PROCESS_PIPE_BYTES),
	}
	child = []uintptr{
		uintptr(input_read), uintptr(output_write), uintptr(error_write),
	}
	if retain_err := process_retain_pipes(spawn); retain_err != nil {
		for _, descriptor := range child {
			syscall.Close(int(descriptor))
		}
		process_close_pipes(spawn)
		return nil, nil, retain_err
	}
	return spawn, child, nil
}

// Configures the three pipe ends the loop keeps.
func process_retain_pipes(spawn *Spawn) (err error) {
	if retain_err := pipe_retain(spawn.Input_Descriptor); retain_err != nil {
		return retain_err
	}
	if retain_err := pipe_retain(spawn.Output_Descriptor); retain_err != nil {
		return retain_err
	}
	return pipe_retain(spawn.Error_Descriptor)
}

// Builds the argv a child receives: the program name followed by the caller's arguments, which
// exclude it. exec.Command supplied this before, and StartProcess does not.
func process_argv(path string, arguments []string) (argv []string) {
	argv = make([]string, 0, len(arguments)+1)
	argv = append(argv, path)
	return append(argv, arguments...)
}

// Reports the PATH a spawn resolves a bare command name against. A request carrying its own
// environment resolves against that environment's PATH, so a caller cannot be surprised by the
// parent's, and one with no environment inherits the parent's.
func operating_system_search_path(request sysos.Process_Request) (search string) {
	if request.Environment == nil {
		search, _ = syscall.Getenv("PATH")
		return search
	}
	for index := len(request.Environment) - 1; index >= 0; index-- {
		entry := request.Environment[index]
		if strings.HasPrefix(entry, "PATH=") {
			return strings.TrimPrefix(entry, "PATH=")
		}
	}
	return ""
}

// Arms the read that drains the child's standard output.
func process_read_output(state *Operating_System, spawn *Spawn) {
	operating_system_submit(&spawn.Output_Completion)
	operating_system_pipe_read(
		state, &spawn.Output_Completion, spawn.Output_Descriptor, spawn.Output_Buffer,
		func(count int, read_err error) {
			process_output_pass(state, spawn, count, read_err)
		},
	)
}

// Arms the read that drains the child's standard error.
func process_read_error(state *Operating_System, spawn *Spawn) {
	operating_system_submit(&spawn.Error_Completion)
	operating_system_pipe_read(
		state, &spawn.Error_Completion, spawn.Error_Descriptor, spawn.Error_Buffer,
		func(count int, read_err error) {
			process_error_pass(state, spawn, count, read_err)
		},
	)
}

// Reports whether one output pass ended the pipe. A zero count is the child's write end
// closing, and an error ends the drain the same way, so a broken pipe cannot leave the spawn
// waiting forever.
func process_pass_ended(count int, read_err error) (ended bool) {
	if read_err != nil {
		return true
	}
	return count == 0
}

// Accepts one standard-output pass, then either rearms the read or marks the pipe drained.
func process_output_pass(
	state *Operating_System, spawn *Spawn, count int, read_err error,
) {
	if process_pass_ended(count, read_err) {
		spawn.Output_Drained = true
		process_close_input_or_output(&spawn.Output_Descriptor)
		process_finish(state, spawn)
		return
	}
	pass := spawn.Output_Buffer[:count]
	// A caller-supplied sink streams the pass live and leaves the captured field empty.
	// Without one the bytes accumulate for the result. The two stay exclusive, as they were
	// when os/exec owned the copy. The sink runs on the loop thread, so a Write that blocks
	// stalls every other operation.
	if spawn.Request.Stdout.Procedure != nil {
		stream.Write(spawn.Request.Stdout, pass)
	} else {
		spawn.Result.Output = append(spawn.Result.Output, pass...)
	}
}

// Accepts one standard-error pass, the counterpart of process_output_pass.
func process_error_pass(
	state *Operating_System, spawn *Spawn, count int, read_err error,
) {
	if process_pass_ended(count, read_err) {
		spawn.Error_Drained = true
		process_close_input_or_output(&spawn.Error_Descriptor)
		process_finish(state, spawn)
		return
	}
	pass := spawn.Error_Buffer[:count]
	if spawn.Request.Stderr.Procedure != nil {
		stream.Write(spawn.Request.Stderr, pass)
	} else {
		spawn.Result.Error_Output = append(spawn.Result.Error_Output, pass...)
	}
}

// Accepts one standard-input pass. A write failure ends the feed by closing the end, which the
// child observes as end-of-file.
func process_input_pass(spawn *Spawn, count int, write_err error) {
	if write_err != nil {
		process_close_input_or_output(&spawn.Input_Descriptor)
		return
	}
	spawn.Input = spawn.Input[count:]
	if len(spawn.Input) == 0 {
		process_close_input_or_output(&spawn.Input_Descriptor)
	}
}

// Closes one pipe end once and marks it released.
func process_close_input_or_output(descriptor *int) {
	if *descriptor < 0 {
		return
	}
	syscall.Close(*descriptor)
	*descriptor = -1
}

// Arms every spawn pipe whose previous pass has retired and whose end is still open. The flush
// loop calls this once each pass, so a pass rearms from the drain rather than from inside its
// own callback. That keeps each pipe to one operation in flight and leaves the arming in one
// place instead of a cycle between the arm and its handler.
func process_arm_pipes(state *Operating_System) {
	for _, spawn := range state.Spawns {
		if process_pipe_idle(spawn.Output_Descriptor, &spawn.Output_Completion) {
			process_read_output(state, spawn)
		}
		if process_pipe_idle(spawn.Error_Descriptor, &spawn.Error_Completion) {
			process_read_error(state, spawn)
		}
		if process_pipe_idle(spawn.Input_Descriptor, &spawn.Input_Completion) {
			process_write_input(state, spawn)
		}
	}
}

// Reports whether a pipe end is open and carries no operation in flight.
func process_pipe_idle(descriptor int, completion *time.Completion) (idle bool) {
	if descriptor < 0 {
		return false
	}
	return completion.State == time.COMPLETION_IDLE
}

// Arms the write that feeds the child's standard input, or closes the end when the request
// supplies none. The child reads end-of-file only once this end is closed.
func process_write_input(state *Operating_System, spawn *Spawn) {
	if len(spawn.Input) == 0 {
		process_close_input_or_output(&spawn.Input_Descriptor)
		return
	}
	operating_system_submit(&spawn.Input_Completion)
	operating_system_pipe_write(
		state, &spawn.Input_Completion, spawn.Input_Descriptor, spawn.Input,
		func(count int, write_err error) {
			process_input_pass(spawn, count, write_err)
		},
	)
}

// Arms the exit watch. Darwin watches the process identifier through EVFILT_PROC and Linux
// polls the pidfd, so neither waits in wait4.
func process_watch_exit(state *Operating_System, spawn *Spawn) {
	operating_system_submit(&spawn.Exit_Completion)
	operation := &Operating_System_Operation{
		Completion:         &spawn.Exit_Completion,
		Kind:               OPERATING_SYSTEM_OPERATION_PROCESS_EXIT,
		Descriptor:         spawn.Exit_Descriptor,
		Process_Identifier: spawn.Identifier,
		Deliver: func(_ int, _ error) {
			process_exit(state, spawn)
		},
	}
	operating_system_operation_submit(state, operation)
}

// Reaps the exited child and records its outcome. The reap is unconditional and happens here
// alone, so a spawn whose caller already retired on its deadline still releases the kernel's
// process entry.
func process_exit(state *Operating_System, spawn *Spawn) {
	defer process_finish(state, spawn)
	exit, usage, reap_err := process_reap(spawn.Identifier)
	spawn.Exited = true
	if spawn.Exit_Descriptor >= 0 {
		syscall.Close(spawn.Exit_Descriptor)
		spawn.Exit_Descriptor = -1
	}
	// A grandchild that inherited the pipes outlives the child and holds them open. Bound that
	// drain from the exit, the way exec.Cmd's WaitDelay did, so a child that finishes at once
	// does not wait out its whole deadline for a descendant it left behind.
	process_bound_cleanup(state, spawn)
	if reap_err != nil {
		return
	}
	spawn.Result.Exit = exit
	spawn.Result.Usage = usage
}

// Kills the child's whole process group when its deadline wins, then bounds the drain of the
// pipes a surviving grandchild may still hold open.
func process_expire(state *Operating_System, spawn *Spawn) {
	if spawn.Delivered {
		return
	}
	spawn.Expired = true
	syscall.Kill(-spawn.Identifier, syscall.SIGKILL)
	process_bound_cleanup(state, spawn)
}

// Starts the bounded pipe drain once. Both the child's exit and the deadline reach it, and only
// the first arms the timer. It is a no-op when both pipes have already ended, so a spawn that
// finished cleanly leaves no timer behind.
func process_bound_cleanup(state *Operating_System, spawn *Spawn) {
	if spawn.Cleanup_Armed {
		return
	}
	if process_pipes_drained(spawn) {
		return
	}
	spawn.Cleanup_Armed = true
	operating_system_submit(&spawn.Cleanup_Completion)
	operating_system_timeout(state, state.Host, &spawn.Cleanup_Completion, func(
		_ *time.Completion, _ error,
	) {
		process_cleanup(state, spawn)
	}, PROCESS_CLEANUP_DURATION)
}

// Reports whether both output pipes have reached their end.
func process_pipes_drained(spawn *Spawn) (drained bool) {
	if !spawn.Output_Drained {
		return false
	}
	return spawn.Error_Drained
}

// Forces the drain to end after the pipes stayed open past the cleanup bound. Expired is set
// even when the child itself exited cleanly, because the captured output is now incomplete and
// a nil error would report a whole result the caller did not get.
//
// Each abandoned pass is retired before its descriptor closes, so no kernel registration
// outlives the spawn. Deinit asserts the operation registry is empty, and a pass left armed
// here would trip it.
func process_cleanup(state *Operating_System, spawn *Spawn) {
	spawn.Expired = true
	spawn.Output_Drained = true
	spawn.Error_Drained = true
	process_retire_pass(state, &spawn.Output_Completion)
	process_retire_pass(state, &spawn.Error_Completion)
	process_retire_pass(state, &spawn.Input_Completion)
	process_close_pipes(spawn)
	process_finish(state, spawn)
}

// Retires one pipe pass the forced drain abandoned. A completion that already retired holds no
// registry entry, so this is a no-op for it.
func process_retire_pass(state *Operating_System, completion *time.Completion) {
	operation := state.Operations[completion.Kernel_Identifier]
	if operation == nil {
		return
	}
	if operation.Completion != completion {
		return
	}
	platform_expire_operation(state, operation)
	operating_system_operation_complete(state, operation, -1, io.Canceled)
}

// Delivers the result once the child has been reaped and both pipes have ended. It runs on
// every one of those paths and retires the caller's completion exactly once.
func process_finish(state *Operating_System, spawn *Spawn) {
	if spawn.Delivered {
		return
	}
	if !spawn.Exited {
		return
	}
	if !spawn.Output_Drained {
		return
	}
	if !spawn.Error_Drained {
		return
	}
	spawn.Delivered = true
	delete(state.Spawns, spawn.Identifier)
	process_close_pipes(spawn)
	spawn.Result.Usage.Wall = time.Duration(
		int64(state.Host.Now_Monotonic()) - int64(spawn.Started))
	result := spawn.Result
	err := error(nil)
	if spawn.Expired {
		err = time.Deadline_Exceeded
	}
	completion := spawn.Completion
	callback := spawn.Callback
	completion.Callback = func() { callback(completion, result, err) }
	state.Completed = append(state.Completed, completion)
}

// Releases every pipe end the loop still holds.
func process_close_pipes(spawn *Spawn) {
	process_close_input_or_output(&spawn.Input_Descriptor)
	process_close_input_or_output(&spawn.Output_Descriptor)
	process_close_input_or_output(&spawn.Error_Descriptor)
}

// Submits one pipe read through the platform scheduler.
func operating_system_pipe_read(
	state *Operating_System, completion *time.Completion,
	descriptor int, buffer []byte, deliver func(count int, err error),
) {
	operating_system_operation_submit(state, &Operating_System_Operation{
		Completion: completion,
		Kind:       OPERATING_SYSTEM_OPERATION_PIPE_READ,
		Descriptor: descriptor,
		Buffer:     platform_buffer_limit(buffer),
		Deliver:    deliver,
	})
}

// Submits one pipe write through the platform scheduler.
func operating_system_pipe_write(
	state *Operating_System, completion *time.Completion,
	descriptor int, buffer []byte, deliver func(count int, err error),
) {
	operating_system_operation_submit(state, &Operating_System_Operation{
		Completion: completion,
		Kind:       OPERATING_SYSTEM_OPERATION_PIPE_WRITE,
		Descriptor: descriptor,
		Buffer:     platform_buffer_limit(buffer),
		Deliver:    deliver,
	})
}

// Normalizes a rusage Maxrss to bytes: Darwin reports bytes, Linux reports KiB.
func process_rss_bytes(maxrss int64) (size int64) {
	if runtime.GOOS == "linux" {
		return maxrss * 1024
	}
	return maxrss
}

// Wires the file operations — read, write, open, create — onto loop.
func operating_system_wire_file(state *Operating_System, loop *io.IO) {
	loop.Read = func(
		completion *time.Completion, callback io.Callback,
		file io.File, buffer []byte, offset int64,
	) {
		operating_system_submit(completion)
		operating_system_read(state, completion, callback, file, buffer, offset)
	}
	loop.Write = func(
		completion *time.Completion, callback io.Callback,
		file io.File, buffer []byte, offset int64,
	) {
		operating_system_submit(completion)
		operating_system_write(state, completion, callback, file, buffer, offset)
	}
	loop.Fsync = func(
		completion *time.Completion, callback time.Timeout_Callback, file io.File,
	) {
		operating_system_submit(completion)
		operating_system_fsync(state, completion, callback, file)
	}
	loop.Open_At = func(
		completion *time.Completion, callback io.File_Callback, directory io.File,
		file_path string, options io.Open_At_Options,
	) {
		invariant.Always(
			options.Flags & ^io.OPEN_AT_NO_FOLLOW == 0,
			"Open_At options contain only known flags.",
		)
		operating_system_submit(completion)
		operating_system_open_at(
			state, completion, callback, directory, file_path, options,
		)
	}
	loop.Mkdir_At = func(
		completion *time.Completion, callback time.Timeout_Callback, directory io.File,
		file_path string, mode uint32,
	) {
		operating_system_submit(completion)
		operating_system_mkdir_at(state, completion, callback, directory, file_path, mode)
	}
	loop.Get_Directory_Entries = func(
		completion *time.Completion, callback io.Directory_Callback, directory io.File,
		buffer []byte,
	) {
		operating_system_submit(completion)
		operating_system_directory_pass(state, completion, callback, directory, buffer)
	}
	loop.Status = func(path string) (status io.File_Status, err error) {
		return file_status(path)
	}
}

// Wires the loop's own control plane — the timer, the deferred callback, and the cross-thread
// event — onto the vtable shared/time owns, and the close primitives onto the IO surface.
func operating_system_wire_timer(state *Operating_System, loop *io.IO, pump *time.Timeline) {
	pump.Submit = func(completion *time.Completion, delay time.Duration, callback func()) {
		operating_system_submit(completion)
		operating_system_timeout(state, state.Host, completion, func(
			_ *time.Completion, _ error,
		) {
			callback()
		}, delay)
	}
	pump.Classify = func(_ *time.Completion, _ time.Operation) {}
	pump.Report_Descriptors = func(_ func() (count int)) {}
	pump.Timeout = func(
		completion *time.Completion, callback time.Timeout_Callback, duration time.Duration,
	) {
		invariant.Always(
			duration > 0, "A timeout duration is positive; yields use Next_Tick.",
		)
		operating_system_submit(completion)
		operating_system_timeout(state, state.Host, completion, callback, duration)
	}
	pump.Next_Tick = func(
		completion *time.Completion, callback time.Next_Tick_Callback,
		source time.Next_Tick_Source,
	) {
		operating_system_submit(completion)
		completion.Next_Tick_Source = source
		completion.Next_Tick = true
		completion.Callback = func() { callback(completion) }
		state.Completed = append(state.Completed, completion)
	}
	pump.Reset_Next_Tick = func(source time.Next_Tick_Source) {
		operating_system_reset_next_tick(state, source)
	}
	pump.Open_Event = func() (event time.Event, err error) {
		return platform_event_open(state)
	}
	pump.Event_Listen = func(
		event time.Event, completion *time.Completion, callback time.Next_Tick_Callback,
	) {
		operating_system_submit(completion)
		operating_system_event_listen(state, event, completion, callback)
	}
	pump.Event_Trigger = func(event time.Event, completion *time.Completion) {
		platform_event_trigger(state, event, completion.Kernel_Identifier)
	}
	pump.Close_Event = func(event time.Event) {
		operating_system_assert_event_drained(state, event)
		platform_event_close(state, event)
	}
	loop.Close = func(
		completion *time.Completion, callback time.Timeout_Callback, file io.File,
	) {
		operating_system_assert_file_drained(state, file)
		operating_system_submit(completion)
		operating_system_close(state, completion, callback, file)
	}
	loop.Close_Socket = func(socket io.File) {
		operating_system_assert_file_drained(state, socket)
		delete(state.Raw_Open, int(socket))
		socket_close(int(socket))
	}
}

// Registers one TigerBeetle Event listener. The platform decides whether it is a persistent
// EVFILT_USER listener or an eventfd read operation.
func operating_system_event_listen(
	state *Operating_System, event time.Event, completion *time.Completion,
	callback time.Next_Tick_Callback,
) {
	operation := &Operating_System_Operation{
		Completion: completion,
		Kind:       OPERATING_SYSTEM_OPERATION_EVENT,
		Descriptor: int(event),
		Deliver:    func(result int, err error) { callback(completion) },
	}
	operating_system_operation_register(state, operation)
	listen_err := platform_event_listen(state, operation)
	invariant.Always(listen_err == nil, "A TigerBeetle Event listener arms successfully.")
}

// Asserts no Event listener remains armed before the backend Event resource is closed.
func operating_system_assert_event_drained(state *Operating_System, event time.Event) {
	armed := false
	for _, operation := range state.Operations {
		if operation.Kind == OPERATING_SYSTEM_OPERATION_EVENT {
			if operation.Descriptor == int(event) {
				armed = true
			}
		}
	}
	invariant.Always(!armed, "An Event listener is drained before Close_Event.")
}

// Asserts no submitted operation still borrows file. TigerBeetle's message bus joins every
// submitted operation before close (third-party/tigerbeetle/src/message_bus.zig:1104-1145); this
// fail-closed check prevents an old kernel completion from racing a recycled descriptor number.
func operating_system_assert_file_drained(state *Operating_System, file io.File) {
	borrowed := false
	for _, operation := range state.Operations {
		if operation.Descriptor == int(file) {
			borrowed = true
		}
	}
	invariant.Always(!borrowed,
		"A descriptor is drained before Close or Close_Socket releases it.")
}

// Wires the socket operations — listen, accept, connect, receive, send, peer address —
// onto loop.
func operating_system_wire_socket(state *Operating_System, loop *io.IO) {
	loop.Bind = func(socket io.File, address io.Address) (err error) {
		return socket_bind(int(socket), address)
	}
	loop.Listen_Socket = func(socket io.File, backlog uint32) (err error) {
		return socket_listen_mark(int(socket), backlog)
	}
	loop.Get_Socket_Name = func(socket io.File) (address io.Address, err error) {
		return socket_name(int(socket))
	}
	loop.Accept = func(
		completion *time.Completion, callback io.Socket_Callback, listener io.File,
		deadline time.Duration,
	) {
		invariant.Always(deadline > 0, "An accept deadline is positive and finite.")
		operating_system_submit(completion)
		operating_system_accept(state, completion, callback, listener, deadline)
	}
	loop.Socket = func(
		family io.Address_Family, transport io.Socket_Transport,
	) (socket io.File, err error) {
		descriptor, open_err := socket_open(family, transport)
		if open_err != nil {
			return io.File(-1), open_err
		}
		state.Raw_Open[descriptor] = true
		return io.File(descriptor), nil
	}
	loop.Set_Socket_Option = func(
		socket io.File, option io.Socket_Option, value int,
	) (err error) {
		return socket_option_set(int(socket), option, value)
	}
	loop.Connect = func(
		completion *time.Completion, callback time.Timeout_Callback, socket io.File,
		address io.Address, deadline time.Duration,
	) {
		invariant.Always(deadline > 0, "A connect deadline is positive and finite.")
		operating_system_submit(completion)
		operating_system_connect(state, completion, callback, socket, address, deadline)
	}
	loop.Receive = func(
		completion *time.Completion, callback io.Callback, socket io.File, buffer []byte,
	) {
		operating_system_submit(completion)
		operating_system_receive(state, completion, callback, socket, buffer)
	}
	loop.Send = func(
		completion *time.Completion, callback io.Callback, socket io.File, buffer []byte,
	) {
		operating_system_submit(completion)
		operating_system_send(state, completion, callback, socket, buffer)
	}
	loop.Send_Now = func(socket io.File, buffer []byte) (count int, sent bool) {
		return socket_send_now(int(socket), buffer)
	}
	loop.Shutdown = func(socket io.File, how io.Shutdown_How) (err error) {
		return socket_shutdown(int(socket), how)
	}
	loop.Peer_Address = func(file io.File) (address string, err error) {
		return socket_peer_address(int(file))
	}
}

// Submits a file read through the platform scheduler (io/darwin.zig:519-558;
// io/linux.zig Completion.prep.read).
func operating_system_read(
	state *Operating_System, completion *time.Completion, callback io.Callback,
	file io.File, buffer []byte, offset int64,
) {
	operation := &Operating_System_Operation{
		Completion: completion,
		Kind:       OPERATING_SYSTEM_OPERATION_READ,
		Descriptor: int(file),
		Buffer:     platform_buffer_limit(buffer),
		Offset:     uint64(offset),
		Deliver:    func(result int, err error) { callback(completion, result, err) },
	}
	operating_system_operation_submit(state, operation)
}

// Submits a file write through the platform scheduler.
func operating_system_write(
	state *Operating_System, completion *time.Completion, callback io.Callback,
	file io.File, buffer []byte, offset int64,
) {
	operation := &Operating_System_Operation{
		Completion: completion,
		Kind:       OPERATING_SYSTEM_OPERATION_WRITE,
		Descriptor: int(file),
		Buffer:     platform_buffer_limit(buffer),
		Offset:     uint64(offset),
		Deliver:    func(result int, err error) { callback(completion, result, err) },
	}
	operating_system_operation_submit(state, operation)
}

// Submits TigerBeetle's asynchronous fsync operation.
func operating_system_fsync(
	state *Operating_System, completion *time.Completion,
	callback time.Timeout_Callback, file io.File,
) {
	operation := &Operating_System_Operation{
		Completion: completion,
		Kind:       OPERATING_SYSTEM_OPERATION_FSYNC,
		Descriptor: int(file),
		Deliver:    func(result int, err error) { callback(completion, err) },
	}
	operating_system_operation_submit(state, operation)
}

// Submits TigerBeetle's asynchronous openat operation with a NUL-terminated path owned until the
// callback retires.
func operating_system_open_at(
	state *Operating_System, completion *time.Completion, callback io.File_Callback,
	directory io.File, file_path string, options io.Open_At_Options,
) {
	descriptor := int(directory)
	if directory == io.DIRECTORY_CURRENT {
		descriptor = platform_current_directory()
	}
	path := append([]byte(file_path), 0)
	operation := &Operating_System_Operation{
		Completion:   completion,
		Kind:         OPERATING_SYSTEM_OPERATION_OPEN_AT,
		Descriptor:   descriptor,
		File_Path:    path,
		Open_Options: options,
		Deliver: func(result int, err error) {
			callback(completion, io.File(result), err)
		},
	}
	operating_system_operation_submit(state, operation)
}

// Submits one directory pass. getdents has no asynchronous form on either backend, so the read
// runs inline and the completion retires on the next drain, exactly as Status does.
func operating_system_directory_pass(
	state *Operating_System, completion *time.Completion, callback io.Directory_Callback,
	directory io.File, buffer []byte,
) {
	entries, pass_err := file_directory_pass(int(directory), buffer)
	completion.Callback = func() { callback(completion, entries, pass_err) }
	state.Completed = append(state.Completed, completion)
}

// Submits one mkdirat through the platform scheduler. Mode travels in Open_Options because both
// operations carry a path and a creation mode, and the scheduler already retains that field.
func operating_system_mkdir_at(
	state *Operating_System, completion *time.Completion, callback time.Timeout_Callback,
	directory io.File, file_path string, mode uint32,
) {
	descriptor := int(directory)
	if directory == io.DIRECTORY_CURRENT {
		descriptor = platform_current_directory()
	}
	operating_system_operation_submit(state, &Operating_System_Operation{
		Completion:   completion,
		Kind:         OPERATING_SYSTEM_OPERATION_MKDIR_AT,
		Descriptor:   descriptor,
		File_Path:    append([]byte(file_path), 0),
		Open_Options: io.Open_At_Options{Mode: mode},
		Deliver:      func(_ int, err error) { callback(completion, err) },
	})
}

// Schedules a positive timeout to fire when the clock passes its deadline.
func operating_system_timeout(
	state *Operating_System, host time.Any_Clock, completion *time.Completion,
	callback time.Timeout_Callback, duration time.Duration,
) {
	if platform_uses_kernel_timeouts() {
		operation := &Operating_System_Operation{
			Completion: completion,
			Kind:       OPERATING_SYSTEM_OPERATION_TIMEOUT,
			Descriptor: -1,
			Timespec:   operating_system_timeout_span(duration),
			Deliver:    func(result int, err error) { callback(completion, err) },
		}
		operating_system_operation_submit(state, operation)
		return
	}
	completion.Ready_At = host.Now_Monotonic() + time.Moment(duration)
	completion.Callback = func() { callback(completion, nil) }
	operating_system_insert(state, completion)
}

// Builds the driver — the loop-advancing capability — over state; held only by the
// composition root or a test, never by code that merely submits IO.
func operating_system_to_driver(state *Operating_System) (driver time.Driver) {
	return time.Driver{
		Run: func() (err error) {
			return operating_system_drive(state, func() (err error) {
				return operating_system_run(state)
			})
		},
		Run_For: func(duration time.Duration) (err error) {
			return operating_system_drive(state, func() (err error) {
				return operating_system_run_for(state, duration)
			})
		},
		Run_Until: func(
			done func() (finished bool), timeout time.Duration,
		) (completed bool, err error) {
			err = operating_system_drive(state, func() (drive_err error) {
				completed, drive_err = operating_system_run_until(
					state, done, timeout,
				)
				return drive_err
			})
			return completed, err
		},
		Deinit: func() {
			operating_system_deinitialize(state)
		},
		Introspect: func() (counts time.Timeline_Counts) {
			return operating_system_introspect(state)
		},
	}
}

// Samples the loop's queue depths for an admin state snapshot. Every field is loop-thread state,
// because Introspect is root-only and nothing in the backend runs off the loop thread.
func operating_system_introspect(state *Operating_System) (counts time.Timeline_Counts) {
	backlog, inflight, queued, kernel := platform_counts(state)
	return time.Timeline_Counts{
		Completed:      len(state.Completed),
		Timeouts:       len(state.Timeouts),
		IO_Backlog:     backlog,
		IO_Inflight:    inflight,
		IO_Queued:      queued,
		IO_In_Kernel:   kernel,
		Signal_Waiters: len(state.Signal_Waiters),
		Spawns:         len(state.Spawns),
		Raw_Open:       len(state.Raw_Open),
	}
}

// Drives until done or the host deadline, propagating every scheduler error.
func operating_system_run_until(
	state *Operating_System, done func() (finished bool), timeout time.Duration,
) (completed bool, err error) {
	deadline := state.Host.Now_Monotonic() + time.Moment(timeout)
	for !done() {
		wait, expired := operating_system_run_until_wait(state, deadline, timeout)
		if expired {
			return false, nil
		}
		flush_err := operating_system_flush(state, wait)
		if flush_err != nil {
			return false, flush_err
		}
	}
	return true, nil
}

// Operating system run until wait chooses a nonblocking first pass, then the nearest deadline.
func operating_system_run_until_wait(
	state *Operating_System, deadline time.Moment, timeout time.Duration,
) (wait time.Moment, expired bool) {
	if len(state.Completed) > 0 {
		return 0, false
	}
	now := state.Host.Now_Monotonic()
	if timeout >= 0 {
		if now >= deadline {
			return 0, true
		}
	}
	wait = POLL_FOREVER
	if timeout >= 0 {
		wait = deadline - now
	}
	return operating_system_wait_cap(state, wait), false
}

// Reports whether an operation can wake an unbounded drive.
func operating_system_in_flight(state *Operating_System) (in_flight bool) {
	if platform_in_flight(state) {
		return true
	}
	if len(state.Timeouts) > 0 {
		return true
	}
	if len(state.Signal_Waiters) > 0 {
		return true
	}
	return len(state.Spawns) > 0
}

// Runs pump as the top-level drive, panicking if a drive is already in progress so a Run*
// called from within a completion callback fails loudly instead of re-entering the driver.
// The internal run functions call one another directly, not through here, so a drive's own
// iteration does not trip it.
func operating_system_drive(state *Operating_System, pump func() (err error)) (err error) {
	invariant.Always(!state.Drive_Active,
		"A drive begins at top level, never from within a completion callback.")
	state.Drive_Active = true
	defer func() { state.Drive_Active = false }()
	return pump()
}

// Runs one nonblocking TigerBeetle flush.
func operating_system_run(state *Operating_System) (err error) {
	return operating_system_flush(state, 0)
}

// Runs flush passes until duration elapses, with the deadline passed directly to the platform.
func operating_system_run_for(state *Operating_System, duration time.Duration) (err error) {
	deadline := state.Host.Now_Monotonic() + time.Moment(duration)
	for state.Host.Now_Monotonic() < deadline {
		now := state.Host.Now_Monotonic()
		if now >= deadline {
			return nil
		}
		wait := operating_system_wait_cap(state, deadline-now)
		flush_err := operating_system_flush(state, wait)
		if flush_err != nil {
			return flush_err
		}
	}
	return nil
}

// Operating system flush checks finite operation deadlines before extension events, then ports
// TigerBeetle's timeout, platform, and retire-before-deliver ordering. Deadline-first ordering
// makes an event first observed at the same pass lose the explicit finite tie.
func operating_system_flush(state *Operating_System, wait time.Moment) (err error) {
	expire_err := operating_system_expire(state)
	if expire_err != nil {
		return expire_err
	}
	operating_system_signals(state)
	process_arm_pipes(state)
	platform_err := platform_run(state, wait)
	if platform_err != nil {
		return platform_err
	}
	operating_system_flush_completed(state)
	return platform_flush_submissions(state)
}

// Operating system wait cap pulls an outer wait in to the nearest Darwin timeout and signal poll.
func operating_system_wait_cap(state *Operating_System, wait time.Moment) (capped time.Moment) {
	capped = wait
	if len(state.Timeouts) > 0 {
		until_timeout := state.Timeouts[0].Ready_At - state.Host.Now_Monotonic()
		if capped < 0 {
			capped = until_timeout
		} else if until_timeout < capped {
			capped = until_timeout
		}
	}
	if len(state.Signal_Waiters) > 0 {
		interval := time.Moment(SIGNAL_POLL_INTERVAL)
		if capped < 0 {
			capped = interval
		} else if interval < capped {
			capped = interval
		}
		until_signal := state.Signal_Waiters[0].Deadline - state.Host.Now_Monotonic()
		for index := 1; index < len(state.Signal_Waiters); index++ {
			candidate := state.Signal_Waiters[index].Deadline -
				state.Host.Now_Monotonic()
			if candidate < until_signal {
				until_signal = candidate
			}
		}
		if capped < 0 {
			capped = until_signal
		} else if until_signal < capped {
			capped = until_signal
		}
	}
	if !platform_uses_kernel_timeouts() {
		for _, operation := range state.Operations {
			if operation.Deadline == 0 {
				continue
			}
			until_operation := operation.Deadline - state.Host.Now_Monotonic()
			if capped < 0 {
				capped = until_operation
			} else if until_operation < capped {
				capped = until_operation
			}
		}
	}
	if capped < 0 {
		invariant.Always(operating_system_in_flight(state),
			"An unbounded run holds an operation in flight that can advance it.")
	}
	return capped
}

// Moves every elapsed timeout and finite extension waiter into the completed queue, and retires
// bounded Darwin operations before their callbacks become visible.
func operating_system_expire(state *Operating_System) (err error) {
	now := state.Host.Now_Monotonic()
	for len(state.Timeouts) > 0 && state.Timeouts[0].Ready_At <= now {
		expired := state.Timeouts[0]
		state.Timeouts = state.Timeouts[1:]
		state.Completed = append(state.Completed, expired)
	}
	operating_system_expire_signals(state, now)
	if platform_uses_kernel_timeouts() {
		return nil
	}
	expired_operations := []*Operating_System_Operation{}
	for _, operation := range state.Operations {
		if operation.Deadline == 0 {
			continue
		}
		if operation.Deadline <= now {
			expired_operations = append(expired_operations, operation)
		}
	}
	for _, operation := range expired_operations {
		cancel_err := platform_expire_operation(state, operation)
		if cancel_err != nil {
			return cancel_err
		}
		operating_system_operation_complete(state, operation, -1, time.Deadline_Exceeded)
	}
	return nil
}

// Runs and clears every ready completion callback, returning each completion to idle
// before its callback fires — so a callback may legally resubmit its own completion.
func operating_system_flush_completed(state *Operating_System) {
	for len(state.Completed) > 0 {
		completion := state.Completed[0]
		state.Completed = state.Completed[1:]
		time.Completion_Transition(completion, time.COMPLETION_ARMED, time.COMPLETION_IDLE)
		completion.Callback()
	}
}

// Removes every completed-queue next-tick operation for source and returns it to idle without
// invoking its callback (io/darwin.zig:783-796; io/linux.zig:354-367).
func operating_system_reset_next_tick(state *Operating_System, source time.Next_Tick_Source) {
	kept := state.Completed[:0]
	for _, completion := range state.Completed {
		if !completion.Next_Tick {
			kept = append(kept, completion)
			continue
		}
		if completion.Next_Tick_Source != source {
			kept = append(kept, completion)
			continue
		}
		time.Completion_Transition(completion, time.COMPLETION_ARMED, time.COMPLETION_IDLE)
	}
	state.Completed = kept
}

// Inserts completion into the timeout queue in Ready_At order.
func operating_system_insert(state *Operating_System, completion *time.Completion) {
	index := 0
	for index < len(state.Timeouts) && state.Timeouts[index].Ready_At <= completion.Ready_At {
		index++
	}
	state.Timeouts = append(state.Timeouts, nil)
	copy(state.Timeouts[index+1:], state.Timeouts[index:])
	state.Timeouts[index] = completion
}

// Submits accept through kqueue's eager/requeue path or io_uring's ACCEPT opcode.
func operating_system_accept(
	state *Operating_System, completion *time.Completion,
	callback io.Socket_Callback, listener io.File, deadline time.Duration,
) {
	operation := &Operating_System_Operation{
		Completion:    completion,
		Kind:          OPERATING_SYSTEM_OPERATION_ACCEPT,
		Descriptor:    int(listener),
		Deadline:      state.Host.Now_Monotonic() + time.Moment(deadline),
		Deadline_Span: operating_system_timeout_span(deadline),
		Deliver: func(result int, err error) {
			callback(completion, io.File(result), err)
		},
	}
	operating_system_operation_submit(state, operation)
}

// Begins a connection on the caller-owned socket and reports only the connect result. How the
// backend reaches that result is its own: io_uring submits a connect, and kqueue arms the
// socket for writability and then issues the connect itself.
func operating_system_connect(
	state *Operating_System, completion *time.Completion,
	callback time.Timeout_Callback, socket io.File, address io.Address, deadline time.Duration,
) {
	operation := &Operating_System_Operation{
		Completion:    completion,
		Kind:          OPERATING_SYSTEM_OPERATION_CONNECT,
		Descriptor:    int(socket),
		Address:       address,
		Deadline:      state.Host.Now_Monotonic() + time.Moment(deadline),
		Deadline_Span: operating_system_timeout_span(deadline),
		Deliver:       func(result int, err error) { callback(completion, err) },
	}
	operating_system_operation_submit(state, operation)
}

// Reads once from socket into buffer and reports the byte count. io_uring submits a receive,
// and kqueue arms the socket for readability and then reads it.
func operating_system_receive(
	state *Operating_System, completion *time.Completion,
	callback io.Callback, socket io.File, buffer []byte,
) {
	operation := &Operating_System_Operation{
		Completion: completion,
		Kind:       OPERATING_SYSTEM_OPERATION_RECEIVE,
		Descriptor: int(socket), Buffer: platform_buffer_limit(buffer),
		Deliver: func(result int, err error) { callback(completion, result, err) },
	}
	operating_system_operation_submit(state, operation)
}

// Writes buffer once to socket and reports the byte count. io_uring submits a send, and kqueue
// arms the socket for writability and then writes it.
func operating_system_send(
	state *Operating_System, completion *time.Completion,
	callback io.Callback, socket io.File, buffer []byte,
) {
	operation := &Operating_System_Operation{
		Completion: completion,
		Kind:       OPERATING_SYSTEM_OPERATION_SEND,
		Descriptor: int(socket), Buffer: platform_buffer_limit(buffer),
		Deliver: func(result int, err error) { callback(completion, result, err) },
	}
	operating_system_operation_submit(state, operation)
}

// Closes file and queues its completion after operating_system_assert_file_drained enforces the
// owner-side join from third-party/tigerbeetle/src/message_bus.zig:1104-1145.
func operating_system_close(
	state *Operating_System, completion *time.Completion,
	callback time.Timeout_Callback, file io.File,
) {
	operation := &Operating_System_Operation{
		Completion: completion,
		Kind:       OPERATING_SYSTEM_OPERATION_CLOSE,
		Descriptor: int(file),
		Deliver:    func(result int, err error) { callback(completion, err) },
	}
	operating_system_operation_submit(state, operation)
}

// Operating system deinitialize enforces TigerBeetle's join-before-deinit contract and releases
// the scheduler.
func operating_system_deinitialize(state *Operating_System) {
	invariant.Always(state.Extension_Submitted == 0,
		"Driver Deinit follows joining every repository-extension completion.")
	invariant.Always(len(state.Spawns) == 0,
		"Every spawned child is reaped before backend deinit.")
	invariant.Always(len(state.Operations) == 0,
		"Driver Deinit follows joining every TigerBeetle operation.")
	if state.Signals != nil {
		signal.Stop(state.Signals)
	}
	platform_deinitialize(state)
}

// Registers a watcher for signal and starts OS notification for it.
func operating_system_watch_signal(
	state *Operating_System, completion *time.Completion,
	callback sysos.Signal_Callback, kind sysos.Signal, deadline time.Duration,
) {
	operating_system_signal_ensure(state)
	system := signal_to_operating_system(kind)
	state.Signal_Waiters = append(state.Signal_Waiters, Signal_Waiter{
		System: system, Kind: kind, Completion: completion, Callback: callback,
		Deadline: state.Host.Now_Monotonic() + time.Moment(deadline),
	})
	signal.Notify(state.Signals, system)
}

// Creates the buffered signal channel on the first watch.
func operating_system_signal_ensure(state *Operating_System) {
	if state.Signals != nil {
		return
	}
	state.Signals = make(chan os.Signal, SIGNAL_QUEUE_DEPTH)
}

// Maps a backend-independent signal to its OS signal.
func signal_to_operating_system(kind sysos.Signal) (system os.Signal) {
	if kind == sysos.SIGNAL_INTERRUPT {
		return syscall.SIGINT
	}
	return syscall.SIGTERM
}

// Drains delivered signals without blocking, firing matching watchers.
func operating_system_signals(state *Operating_System) {
	for index := 0; index < SIGNAL_QUEUE_DEPTH; index++ {
		select {
		case received := <-state.Signals:
			operating_system_signal_deliver(state, received)
		default:
			return
		}
	}
}

// Fires every watcher matching received one-shot, keeping the rest for later deliveries.
func operating_system_signal_deliver(state *Operating_System, received os.Signal) {
	kept := state.Signal_Waiters[:0]
	for index := 0; index < len(state.Signal_Waiters); index++ {
		waiter := state.Signal_Waiters[index]
		if waiter.System != received {
			kept = append(kept, waiter)
			continue
		}
		delivered := waiter
		delivered.Completion.Callback = func() {
			delivered.Callback(delivered.Completion, delivered.Kind, nil)
		}
		state.Completed = append(state.Completed, delivered.Completion)
	}
	state.Signal_Waiters = kept
}

// Expires signal watchers before draining the os/signal channel, so an event first observed at the
// deadline is discarded and cannot leak into the next explicitly rearmed watch.
func operating_system_expire_signals(state *Operating_System, now time.Moment) {
	kept := state.Signal_Waiters[:0]
	for index := 0; index < len(state.Signal_Waiters); index++ {
		waiter := state.Signal_Waiters[index]
		if waiter.Deadline > now {
			kept = append(kept, waiter)
			continue
		}
		expired := waiter
		expired.Completion.Callback = func() {
			expired.Callback(
				expired.Completion, sysos.Signal(-1), time.Deadline_Exceeded,
			)
		}
		state.Completed = append(state.Completed, expired.Completion)
	}
	state.Signal_Waiters = kept
}
