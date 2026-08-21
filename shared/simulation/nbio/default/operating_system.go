// Package nbio is operating-system backend of shared/simulation/nbio.
//
// Darwin ride kqueue, Linux ride io_uring. No generic Cancel. Descriptor owner do Shutdown, join
// submitted operations, then asynchronous Close.
package nbio

import (
	"errors"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/simulation/nbio"
	sysos "local/james-orcales/shared/simulation/os"
	"local/james-orcales/shared/simulation/time"
)

// Poll_forever, passed as idle gap, block wait until event arrive, not for fixed span — the
// unbounded wait a run with no deadline need, thus loop sleep exactly until there is work,
// instead of wake on interval.
const POLL_FOREVER time.Monotonic_Moment = -1

// Buffer few pending signals, thus burst is not lost between drains.
const SIGNAL_QUEUE_DEPTH = 8

// Cap idle gap while signal watcher exist, since signal does not wake poll. Loop re-check
// signal channel at least this often.
const SIGNAL_POLL_INTERVAL = 10 * time.MILLISECOND

// Hold host backend state: completed queue, Darwin timeout queue, platform scheduler, and
// repository-extension effects, which all retire through completed.
type Operating_System struct {
	// Host is real clock. Deadline and wait are measured against it.
	Host time.Clock
	// Timeouts are pending timer completions ordered by Ready_At, earliest first.
	Timeouts []*time.Completion
	// Completed are completions whose callbacks are ready to run on next drain.
	Completed []*time.Completion
	// Platform is kqueue on Darwin, or io_uring on Linux, made eagerly by constructor.
	Platform Platform_Scheduler
	// Operations map integer kernel user_data values to caller-owned completion operations.
	Operations map[uint64]*Operating_System_Operation
	// Next_Identifier is last non-zero kernel correlation identifier issued.
	Next_Identifier uint64
	// Signals receive OS signals from os/signal notifier. Nil until first watch.
	Signals chan os.Signal
	// Signal_Waiters are registered signal watchers, fired one-shot on delivery.
	Signal_Waiters []Signal_Waiter
	// Spawns track every child loop started and not yet reaped, keyed by process identifier.
	// Entry outlive its caller completion, because deadline retire that completion while child
	// is still running.
	Spawns map[int]*Spawn
	// Extension_Submitted count repository-extension completions not yet delivered to caller.
	Extension_Submitted int
	// Drive_Active is set while Run* drive loop, thus Run* called from inside completion
	// callback — which would re-enter driver mid-drain — panic loud.
	Drive_Active bool
	// Raw_Open record every raw descriptor this backend hold open.
	Raw_Open map[int]bool
}

// One registered signal watcher: OS signal it await, its backend-independent kind, and
// completion and callback to fire once on delivery.
type Signal_Waiter struct {
	// System is OS signal this watcher await.
	System os.Signal
	// Kind is backend-independent signal reported to callback.
	Kind sysos.Signal
	// Completion is caller-owned completion fired on delivery.
	Completion *time.Completion
	// Callback is typed callback run with delivered signal.
	Callback sysos.Signal_Callback
	// Deadline is finite moment this repository-extension operation retire at.
	Deadline time.Monotonic_Moment
}

// New_Operating_System_IO eagerly make platform scheduler. Entries is io_uring queue size on
// Linux, and kqueue on Darwin accept but ignore it. It must fit in twelve bits. Flags
// pass direct to io_uring setup, and Darwin ignore them. ambient state process values host read
// back — os/default supply them — and returned system is that OS completed with signal watch and
// spawn, which retire on queue of this backend.
func New_Operating_System_IO(
	host time.Clock, entries uint16, flags uint32, ambient sysos.OS,
) (loop nbio.IO, pump time.Timeline, driver time.Driver, system sysos.OS, err error) {
	if entries == 0 {
		return nbio.IO{}, time.Timeline{}, time.Driver{}, sysos.OS{}, errors.New(
			"io: scheduler entries must be in [1, 4095]",
		)
	}
	if entries > 4095 {
		return nbio.IO{}, time.Timeline{}, time.Driver{}, sysos.OS{}, errors.New(
			"io: scheduler entries must be in [1, 4095]",
		)
	}
	platform, initialize_err := platform_initialize(entries, flags)
	if initialize_err != nil {
		return nbio.IO{}, time.Timeline{}, time.Driver{}, sysos.OS{}, initialize_err
	}
	state := &Operating_System{
		Host:       host,
		Platform:   platform,
		Operations: map[uint64]*Operating_System_Operation{},
		Raw_Open:   map[int]bool{},
		Spawns:     map[int]*Spawn{},
	}
	operating_system_wire_file(state, &loop.Storage)
	operating_system_wire_timer(state, &pump)
	operating_system_wire_socket(state, &loop.Network)
	operating_system_wire_close(state, &loop)
	system = ambient
	operating_system_wire_effects(state, &system)
	operating_system_wire_platform(state, &loop)
	time.Timeline_Invariants(pump, "new_operating_system_io.pump")
	sysos.OS_Invariants(system, "new_operating_system_io.system")
	return loop, pump, operating_system_to_driver(state), system, nil
}

// Arm completion through idle-to-armed lifecycle edge, same edge virtual timeline stamp
// (shared/simulation/time/time.go:535-540), thus both backend read alike.
func operating_system_submit(completion *time.Completion) {
	original := completion.Self == nil || completion.Self == completion
	invariant.Always(original,
		"A submitted completion is its own original, never a by-value copy.")
	completion.Self = completion
	invariant.Always(!completion.Armed, "An armed completion is never armed a second time.")
	completion.Data = 0
	completion.Error = nil
	completion.Armed = true
}

// Wire signal watch and spawn onto OS surface that own them. Both retire on same completed queue
// every IO operation use, thus one timeline hold whole run.
func operating_system_wire_effects(state *Operating_System, system *sysos.OS) {
	system.Watch_Signal = func(
		completion *time.Completion, signal sysos.Signal, deadline time.Duration,
		callback sysos.Signal_Callback,
	) {
		invariant.Always(deadline > 0, "A signal-watch deadline is positive and finite.")
		state.Extension_Submitted++
		operating_system_submit(completion)
		operating_system_watch_signal(state, completion, signal, deadline, func(
			completed *time.Completion, delivered sysos.Signal, watch_err error,
		) {
			state.Extension_Submitted--
			callback(completed, delivered, watch_err)
		})
	}
	system.Spawn = func(
		completion *time.Completion, request sysos.Process_Request, deadline time.Duration,
		callback sysos.Process_Callback,
	) {
		invariant.Always(deadline > 0, "A spawn deadline is positive and finite.")
		state.Extension_Submitted++
		operating_system_submit(completion)
		operating_system_spawn(state, completion, request, deadline, func(
			completed *time.Completion, result sysos.Process_Result, spawn_err error,
		) {
			state.Extension_Submitted--
			callback(completed, result, spawn_err)
		})
	}
}

// Bound one pipe read, thus chatty child is drained in repeated passes, not into one unbounded
// allocation.
const PROCESS_PIPE_BYTES = 8192

// Cap wait for pipes of killed child to close. Grandchild that inherited write end keep pipe
// open after its parent die, thus drain need its own bound.
const PROCESS_CLEANUP_DURATION = 1 * time.SECOND

// One spawned child loop track. It outlive caller completion: deadline retire that completion
// early, and exit event still arrive after, and still has to reap. Entry leave state.Spawns only
// when its reap finish.
type Spawn struct {
	// Identifier is child process id, and its process group id, because Setpgid is set.
	Identifier int
	// Completion is caller-owned completion result is delivered on.
	Completion *time.Completion
	// Callback is typed callback run once result is whole.
	Callback sysos.Process_Callback
	// Started is moment child was forked, for wall-time measurement.
	Started time.Monotonic_Moment
	// Exit_Descriptor is Linux pidfd polled for exit, or -1 on Darwin.
	Exit_Descriptor int
	// Input_Descriptor is write end of child standard input. -1 once closed.
	Input_Descriptor int
	// Output_Descriptor is read end of child standard output. -1 once closed.
	Output_Descriptor int
	// Error_Descriptor is read end of child standard error. -1 once closed.
	Error_Descriptor int
	// Input is remaining bytes to write to child standard input.
	Input []byte
	// Output_Buffer receive one standard-output pass.
	Output_Buffer []byte
	// Error_Buffer receive one standard-error pass.
	Error_Buffer []byte
	// Result accumulate captured output, exit code, and usage.
	Result sysos.Process_Result
	// Request hold caller live output sinks.
	Request sysos.Process_Request
	// Exit_Completion wait for child to exit.
	Exit_Completion time.Completion
	// Output_Completion read one standard-output pass.
	Output_Completion time.Completion
	// Error_Completion read one standard-error pass.
	Error_Completion time.Completion
	// Output_Stream_Completion keeps the pipe buffer borrowed until its sink retires.
	Output_Stream_Completion time.Completion
	// Error_Stream_Completion keeps the pipe buffer borrowed until its sink retires.
	Error_Stream_Completion time.Completion
	// Input_Completion write one standard-input pass.
	Input_Completion time.Completion
	// Deadline_Completion stays backend-owned so an earlier child exit can withdraw its bound.
	Deadline_Completion time.Completion
	// Cleanup_Completion bound pipe drain after deadline kill.
	Cleanup_Completion time.Completion
	// Exited report reap finished, and Result carry exit code and usage.
	Exited bool
	// Output_Drained report standard-output pipe reached its end.
	Output_Drained bool
	// Error_Drained report standard-error pipe reached its end.
	Error_Drained bool
	// Expired report deadline won, thus result carry Deadline_Exceeded.
	Expired bool
	// Cleanup_Armed report bounded pipe drain already run, thus neither child exit nor
	// deadline arm second one.
	Cleanup_Armed bool
	// Stream_Error preserves the first sink failure while the loop continues to drain pipes.
	Stream_Error error
	// Delivered report caller completion already retired.
	Delivered bool
}

// Start command of request and arm every operation that finish it: one read for each output
// pipe, one write for input, exit watch, and deadline. Nothing run off loop thread.
func operating_system_spawn(
	state *Operating_System, completion *time.Completion, request sysos.Process_Request,
	deadline time.Duration, callback sysos.Process_Callback,
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
	operating_system_internal_timeout(
		state, &spawn.Deadline_Completion, deadline, func(_ *time.Completion) {
			process_expire(state, spawn)
		},
	)
}

// Fork child with its three pipes and return tracking entry. Every descriptor is released on
// failure part-way through, thus failed start leak nothing.
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
	// Child hold its own copies once StartProcess return, thus parent release three ends it
	// handed over, whether or not fork succeeded.
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

// Make three pipes of child. Return tracking entry that hold ends loop keep, and three ends child
// receive as its standard descriptors.
//
// Kept ends take close-on-exec here, BEFORE fork. StartProcess dup only Files entries and leave
// every other open descriptor to exec, thus kept end without flag reach child. Child would then
// hold writer for its own standard input, and close of parent end would never give it
// end-of-file.
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

// Config three pipe ends loop keep.
func process_retain_pipes(spawn *Spawn) (err error) {
	if retain_err := pipe_retain(spawn.Input_Descriptor); retain_err != nil {
		return retain_err
	}
	if retain_err := pipe_retain(spawn.Output_Descriptor); retain_err != nil {
		return retain_err
	}
	return pipe_retain(spawn.Error_Descriptor)
}

// Build argv child receive: program name, then caller arguments, which exclude it. exec.Command
// supplied this before, and StartProcess does not.
func process_argv(path string, arguments []string) (argv []string) {
	argv = make([]string, 0, len(arguments)+1)
	argv = append(argv, path)
	return append(argv, arguments...)
}

// Report PATH a spawn resolve bare command name against. Request carrying its own environment
// resolve against PATH of that environment, thus parent PATH cannot surprise caller. Request with
// no environment inherit parent PATH.
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

// Arm read that drain child standard output.
func process_read_output(state *Operating_System, spawn *Spawn) {
	operating_system_submit(&spawn.Output_Completion)
	operating_system_pipe_read(
		state, &spawn.Output_Completion, spawn.Output_Descriptor, spawn.Output_Buffer,
		func(count int, read_err error) {
			process_output_pass(state, spawn, count, read_err)
		},
	)
}

// Arm read that drain child standard error.
func process_read_error(state *Operating_System, spawn *Spawn) {
	operating_system_submit(&spawn.Error_Completion)
	operating_system_pipe_read(
		state, &spawn.Error_Completion, spawn.Error_Descriptor, spawn.Error_Buffer,
		func(count int, read_err error) {
			process_error_pass(state, spawn, count, read_err)
		},
	)
}

// Report whether one output pass ended pipe. Zero count is child write end closing, and error
// end drain same way, thus broken pipe cannot leave spawn waiting forever.
func process_pass_ended(count int, read_err error) (ended bool) {
	if read_err != nil {
		return true
	}
	return count == 0
}

// Accept one standard-output pass, then rearm read, or mark pipe drained.
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
	// Caller-supplied sink stream pass live and leave captured field empty. Without one,
	// bytes accumulate for result. Two stay exclusive, same as when os/exec owned copy. Sink
	// run on loop thread, thus Write that block stall every other operation.
	if spawn.Request.Stdout != nil {
		nbio.Write(
			spawn.Request.Stdout, &spawn.Output_Stream_Completion, pass,
			func(completed *time.Completion) {
				process_stream_pass(state, spawn, completed)
			},
		)
	} else {
		spawn.Result.Output = append(spawn.Result.Output, pass...)
	}
}

// Accept one standard-error pass, counterpart of process_output_pass.
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
	if spawn.Request.Stderr != nil {
		nbio.Write(
			spawn.Request.Stderr, &spawn.Error_Stream_Completion, pass,
			func(completed *time.Completion) {
				process_stream_pass(state, spawn, completed)
			},
		)
	} else {
		spawn.Result.Error_Output = append(spawn.Result.Error_Output, pass...)
	}
}

// A sink failure cannot stop pipe drain because a child can block on its full pipe. Preserve the
// failure, continue the drain, and publish it only after the child and both pipes finish.
func process_stream_pass(
	state *Operating_System, spawn *Spawn, completion *time.Completion,
) {
	if completion.Error != nil {
		if spawn.Stream_Error == nil {
			spawn.Stream_Error = completion.Error
		}
	}
	process_finish(state, spawn)
}

// Accept one standard-input pass. Write failure end feed by close of end, which child observe as
// end-of-file.
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

// Close one pipe end once and mark it released.
func process_close_input_or_output(descriptor *int) {
	if *descriptor < 0 {
		return
	}
	syscall.Close(*descriptor)
	*descriptor = -1
}

// Arm every spawn pipe whose previous pass retired and whose end is still open. Flush loop call
// this once each pass, thus pass rearm from drain, not from inside its own callback. That keep
// each pipe to one operation in flight, and leave arming in one place instead of cycle between
// arm and its handler.
func process_arm_pipes(state *Operating_System) {
	for _, spawn := range state.Spawns {
		if process_pipe_idle(spawn.Output_Descriptor, &spawn.Output_Completion) {
			if !spawn.Output_Stream_Completion.Armed {
				process_read_output(state, spawn)
			}
		}
		if process_pipe_idle(spawn.Error_Descriptor, &spawn.Error_Completion) {
			if !spawn.Error_Stream_Completion.Armed {
				process_read_error(state, spawn)
			}
		}
		if process_pipe_idle(spawn.Input_Descriptor, &spawn.Input_Completion) {
			process_write_input(state, spawn)
		}
	}
}

// Report whether pipe end is open and carry no operation in flight.
func process_pipe_idle(descriptor int, completion *time.Completion) (idle bool) {
	if descriptor < 0 {
		return false
	}
	return !completion.Armed
}

// Arm write that feed child standard input, or close end when request supply none. Child read
// end-of-file only once this end is closed.
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

// Arm exit watch. Darwin watch process identifier through EVFILT_PROC, Linux poll pidfd, thus
// neither wait in wait4.
func process_watch_exit(state *Operating_System, spawn *Spawn) {
	operating_system_submit(&spawn.Exit_Completion)
	operating_system_operation_submit(state, &Operating_System_Operation{
		Completion:         &spawn.Exit_Completion,
		Kind:               OPERATING_SYSTEM_OPERATION_PROCESS_EXIT,
		Descriptor:         spawn.Exit_Descriptor,
		Process_Identifier: spawn.Identifier,
		Deliver: func(_ *time.Completion) {
			process_exit(state, spawn)
		},
	})
}

// Reap exited child and record its outcome. Reap is unconditional and happen here alone, thus
// spawn whose caller already retired on its deadline still release kernel process entry.
func process_exit(state *Operating_System, spawn *Spawn) {
	defer process_finish(state, spawn)
	exit, usage, reap_err := process_reap(spawn.Identifier)
	spawn.Exited = true
	if spawn.Exit_Descriptor >= 0 {
		syscall.Close(spawn.Exit_Descriptor)
		spawn.Exit_Descriptor = -1
	}
	// Grandchild that inherited pipes outlive child and hold them open. Bound that drain from
	// exit, same way exec.Cmd WaitDelay did, thus child that finish at once does not wait out
	// its whole deadline for descendant it left behind.
	process_bound_cleanup(state, spawn)
	if reap_err != nil {
		return
	}
	spawn.Result.Exit = exit
	spawn.Result.Usage = usage
}

// Kill whole process group of child when its deadline win, then bound drain of pipes a surviving
// grandchild may still hold open.
func process_expire(state *Operating_System, spawn *Spawn) {
	if spawn.Delivered {
		return
	}
	spawn.Expired = true
	syscall.Kill(-spawn.Identifier, syscall.SIGKILL)
	process_bound_cleanup(state, spawn)
}

// Start bounded pipe drain once. Both child exit and deadline reach it, and only first arm timer.
// It is no-op when both pipes already ended, thus spawn that finished cleanly leave no timer
// behind.
func process_bound_cleanup(state *Operating_System, spawn *Spawn) {
	if spawn.Cleanup_Armed {
		return
	}
	if process_pipes_drained(spawn) {
		return
	}
	spawn.Cleanup_Armed = true
	operating_system_submit(&spawn.Cleanup_Completion)
	operating_system_internal_timeout(state, &spawn.Cleanup_Completion,
		PROCESS_CLEANUP_DURATION, func(_ *time.Completion) {
			process_cleanup(state, spawn)
		})
}

// Report whether both output pipes reached their end.
func process_pipes_drained(spawn *Spawn) (drained bool) {
	if !spawn.Output_Drained {
		return false
	}
	return spawn.Error_Drained
}

// Force drain to end after pipes stayed open past cleanup bound. Expired is set even when child
// itself exited cleanly, because captured output is now incomplete, and nil error would report
// whole result caller did not get.
//
// Each abandoned pass is retired before its descriptor close, thus no kernel registration outlive
// spawn. Deinit assert operation registry is empty, and pass left armed here would trip it.
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

// Retire one pipe pass forced drain abandoned. Completion that already retired hold no registry
// entry, thus this is no-op for it.
func process_retire_pass(state *Operating_System, completion *time.Completion) {
	operation := state.Operations[completion.Kernel_Identifier]
	if operation == nil {
		return
	}
	if operation.Completion != completion {
		return
	}
	platform_expire_operation(state, operation)
	operating_system_operation_complete(state, operation, -1, nbio.Canceled)
}

// Deliver result once child was reaped and both pipes ended. It run on every one of those paths,
// and retire caller completion exactly once.
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
	if spawn.Output_Stream_Completion.Armed {
		return
	}
	if spawn.Error_Stream_Completion.Armed {
		return
	}
	operating_system_internal_timeout_cancel(state, &spawn.Deadline_Completion)
	operating_system_internal_timeout_cancel(state, &spawn.Cleanup_Completion)
	spawn.Delivered = true
	delete(state.Spawns, spawn.Identifier)
	process_close_pipes(spawn)
	spawn.Result.Usage.Wall = time.Duration(
		int64(state.Host.Now_Monotonic()) - int64(spawn.Started))
	result := spawn.Result
	err := error(nil)
	if spawn.Expired {
		err = time.Deadline_Exceeded
	} else if spawn.Stream_Error != nil {
		err = spawn.Stream_Error
	}
	completion := spawn.Completion
	callback := spawn.Callback
	completion.Callback = func() { callback(completion, result, err) }
	state.Completed = append(state.Completed, completion)
}

// Release every pipe end loop still hold.
func process_close_pipes(spawn *Spawn) {
	process_close_input_or_output(&spawn.Input_Descriptor)
	process_close_input_or_output(&spawn.Output_Descriptor)
	process_close_input_or_output(&spawn.Error_Descriptor)
}

// Submit one pipe read through platform scheduler.
func operating_system_pipe_read(
	state *Operating_System, completion *time.Completion,
	descriptor int, buffer []byte, deliver func(count int, err error),
) {
	operating_system_operation_submit(state, &Operating_System_Operation{
		Completion: completion,
		Kind:       OPERATING_SYSTEM_OPERATION_PIPE_READ,
		Descriptor: descriptor,
		Buffer:     platform_buffer_limit(buffer),
		Deliver: func(completed *time.Completion) {
			deliver(completed.Data, completed.Error)
		},
	})
}

// Submit one pipe write through platform scheduler.
func operating_system_pipe_write(
	state *Operating_System, completion *time.Completion,
	descriptor int, buffer []byte, deliver func(count int, err error),
) {
	operating_system_operation_submit(state, &Operating_System_Operation{
		Completion: completion,
		Kind:       OPERATING_SYSTEM_OPERATION_PIPE_WRITE,
		Descriptor: descriptor,
		Buffer:     platform_buffer_limit(buffer),
		Deliver: func(completed *time.Completion) {
			deliver(completed.Data, completed.Error)
		},
	})
}

// Normalize rusage Maxrss to bytes: Darwin report bytes, Linux report KiB.
func process_rss_bytes(maxrss int64) (size int64) {
	if runtime.GOOS == "linux" {
		return maxrss * 1024
	}
	return maxrss
}

// Wire file operations — read, write, open, create — onto storage.
func operating_system_wire_file(state *Operating_System, loop *nbio.Storage) {
	loop.Read = func(
		completion *time.Completion, file nbio.File, buffer []byte, offset int64,
		timeout time.Duration,
		callback time.Callback,
	) {
		invariant.Always(timeout > 0, "A storage read timeout is positive and finite.")
		deadline := platform_storage_deadline(state, timeout)
		operating_system_submit(completion)
		operating_system_read(
			state, completion, file, buffer, offset, deadline, callback,
		)
	}
	loop.Write = func(
		completion *time.Completion, file nbio.File, buffer []byte, offset int64,
		timeout time.Duration,
		callback time.Callback,
	) {
		invariant.Always(timeout > 0, "A storage write timeout is positive and finite.")
		deadline := platform_storage_deadline(state, timeout)
		operating_system_submit(completion)
		operating_system_write(
			state, completion, file, buffer, offset, deadline, callback,
		)
	}
	loop.Fsync = func(
		completion *time.Completion, file nbio.File, timeout time.Duration,
		callback time.Callback,
	) {
		invariant.Always(timeout > 0, "A storage fsync timeout is positive and finite.")
		deadline := platform_storage_deadline(state, timeout)
		operating_system_submit(completion)
		operating_system_fsync(state, completion, file, deadline, callback)
	}
	loop.Open_At = func(
		completion *time.Completion, directory nbio.File, file_path string,
		options nbio.Open_At_Options, callback time.Callback,
	) {
		invariant.Always(
			options.Flags & ^nbio.OPEN_AT_NO_FOLLOW == 0,
			"Open_At options contain only known flags.",
		)
		operating_system_submit(completion)
		operating_system_open_at(
			state, completion, directory, file_path, options, callback,
		)
	}
	loop.Mkdir_At = func(
		completion *time.Completion, directory nbio.File, file_path string, mode uint32,
		callback time.Callback,
	) {
		operating_system_submit(completion)
		operating_system_mkdir_at(state, completion, directory, file_path, mode, callback)
	}
	loop.Get_Directory_Entries = func(
		completion *time.Completion, directory nbio.File, buffer []byte,
		callback nbio.Directory_Callback,
	) {
		operating_system_submit(completion)
		operating_system_directory_pass(state, completion, directory, buffer, callback)
	}
	loop.Status = func(path string) (status nbio.File_Status, err error) {
		return file_status(path)
	}
}

// Wire loop own control plane — timer and cross-thread event — onto vtable shared/time own.
// Close primitives are not here: they name descriptor, which is business of IO surface, not of
// timeline.
func operating_system_wire_timer(state *Operating_System, pump *time.Timeline) {
	pump.Submit = func(
		completion *time.Completion, delay time.Duration, callback time.Callback,
	) {
		operating_system_submit(completion)
		operating_system_timeout(state, state.Host, completion, delay, callback)
	}
	pump.Timeout = func(
		completion *time.Completion, duration time.Duration, callback time.Callback,
	) {
		invariant.Always(duration > 0, "A timeout duration is positive.")
		operating_system_submit(completion)
		operating_system_timeout(state, state.Host, completion, duration, callback)
	}
	pump.Open_Event = func() (event time.Event, err error) {
		return platform_event_open(state)
	}
	pump.Event_Listen = func(
		event time.Event, completion *time.Completion,
		callback time.Callback,
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
}

// Wire two members both halves share: asynchronous close of any descriptor, and leak check that
// state run released every one it took.
func operating_system_wire_close(state *Operating_System, loop *nbio.IO) {
	loop.Close = func(
		completion *time.Completion, file nbio.File, callback time.Callback,
	) {
		operating_system_assert_file_drained(state, file)
		operating_system_submit(completion)
		operating_system_close(state, completion, file, callback)
	}
	loop.Deinit = func() {
		invariant.Always(len(state.Raw_Open) == 0,
			"Every descriptor the backend opened is closed before Deinit.")
	}
}

// Register one Event listener. Platform decide whether it is persistent EVFILT_USER
// listener, or eventfd read operation.
func operating_system_event_listen(
	state *Operating_System, event time.Event, completion *time.Completion,
	callback time.Callback,
) {
	operation := &Operating_System_Operation{
		Completion: completion,
		Kind:       OPERATING_SYSTEM_OPERATION_EVENT,
		Descriptor: int(event),
		Deliver:    callback,
	}
	operating_system_operation_register(state, operation)
	listen_err := platform_event_listen(state, operation)
	invariant.Always(listen_err == nil, "An Event listener arms successfully.")
}

// Assert no Event listener stay armed before backend Event resource is closed.
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

// Assert no submitted operation still borrow file. Owner join every submitted operation before
// close, and this fail-closed check stop old kernel completion from race with recycled descriptor
// number.
func operating_system_assert_file_drained(state *Operating_System, file nbio.File) {
	borrowed := false
	for _, operation := range state.Operations {
		if operation.Descriptor == int(file) {
			borrowed = true
		}
	}
	invariant.Always(!borrowed,
		"A descriptor is drained before Close releases it.")
}

// Wire socket lifecycle separately from byte transfers so neither boundary hides in one table.
func operating_system_wire_socket(state *Operating_System, loop *nbio.Network) {
	loop.Bind = func(socket nbio.File, address nbio.Address) (err error) {
		return socket_bind(int(socket), address)
	}
	loop.Listen_Socket = func(socket nbio.File, backlog uint32) (err error) {
		return socket_listen_mark(int(socket), backlog)
	}
	loop.Get_Socket_Name = func(socket nbio.File) (address nbio.Address, err error) {
		return socket_name(int(socket))
	}
	loop.Accept = func(
		completion *time.Completion, listener nbio.File, timeout time.Duration,
		callback time.Callback,
	) {
		invariant.Always(timeout > 0, "An accept timeout is positive and finite.")
		operating_system_submit(completion)
		operating_system_accept(state, completion, listener, timeout, callback)
	}
	loop.Socket_TCP = func(
		family nbio.Address_Family, options nbio.TCP_Options,
	) (socket nbio.File, err error) {
		descriptor, open_err := socket_open_tcp(family, options)
		if open_err != nil {
			return nbio.File(-1), open_err
		}
		state.Raw_Open[descriptor] = true
		return nbio.File(descriptor), nil
	}
	loop.Socket_UDP = func(
		family nbio.Address_Family, options nbio.UDP_Options,
	) (socket nbio.File, err error) {
		descriptor, open_err := socket_open_udp(family, options)
		if open_err != nil {
			return nbio.File(-1), open_err
		}
		state.Raw_Open[descriptor] = true
		return nbio.File(descriptor), nil
	}
	loop.Connect = func(
		completion *time.Completion, socket nbio.File, address nbio.Address,
		timeout time.Duration,
		callback time.Callback,
	) {
		invariant.Always(timeout > 0, "A connect timeout is positive and finite.")
		operating_system_submit(completion)
		operating_system_connect(state, completion, socket, address, timeout, callback)
	}
	loop.Shutdown = func(socket nbio.File, how nbio.Shutdown_How) (err error) {
		return socket_shutdown(int(socket), how)
	}
	loop.Peer_Address = func(file nbio.File) (address string, err error) {
		return socket_peer_address(int(file))
	}
	operating_system_wire_socket_transfers(state, loop)
}

// Byte transfers share one required timeout contract on TCP and UDP descriptors.
func operating_system_wire_socket_transfers(state *Operating_System, loop *nbio.Network) {
	loop.Receive = func(
		completion *time.Completion, socket nbio.File, buffer []byte, timeout time.Duration,
		callback time.Callback,
	) {
		invariant.Always(timeout > 0, "A receive timeout is positive and finite.")
		operating_system_submit(completion)
		operating_system_receive(state, completion, socket, buffer, timeout, callback)
	}
	loop.Send = func(
		completion *time.Completion, socket nbio.File, buffer []byte, timeout time.Duration,
		callback time.Callback,
	) {
		invariant.Always(timeout > 0, "A send timeout is positive and finite.")
		operating_system_submit(completion)
		operating_system_send(state, completion, socket, buffer, timeout, callback)
	}
}

// Submit file read through platform scheduler.
func operating_system_read(
	state *Operating_System, completion *time.Completion, file nbio.File, buffer []byte,
	offset int64, deadline time.Monotonic_Moment, callback time.Callback,
) {
	if len(buffer) == 0 {
		operating_system_empty_transfer_complete(state, completion, callback)
		return
	}
	operation := &Operating_System_Operation{
		Completion: completion,
		Kind:       OPERATING_SYSTEM_OPERATION_READ,
		Descriptor: int(file),
		Buffer:     platform_buffer_limit(buffer),
		Offset:     uint64(offset),
		Deadline:   deadline,
		Deliver:    callback,
	}
	operating_system_operation_submit(state, operation)
}

// Submit file write through platform scheduler.
func operating_system_write(
	state *Operating_System, completion *time.Completion, file nbio.File, buffer []byte,
	offset int64, deadline time.Monotonic_Moment, callback time.Callback,
) {
	if len(buffer) == 0 {
		operating_system_empty_transfer_complete(state, completion, callback)
		return
	}
	operation := &Operating_System_Operation{
		Completion: completion,
		Kind:       OPERATING_SYSTEM_OPERATION_WRITE,
		Descriptor: int(file),
		Buffer:     platform_buffer_limit(buffer),
		Offset:     uint64(offset),
		Deadline:   deadline,
		Deliver:    callback,
	}
	operating_system_operation_submit(state, operation)
}

// Submit asynchronous fsync operation.
func operating_system_fsync(
	state *Operating_System, completion *time.Completion, file nbio.File,
	deadline time.Monotonic_Moment, callback time.Callback,
) {
	operation := &Operating_System_Operation{
		Completion: completion,
		Kind:       OPERATING_SYSTEM_OPERATION_FSYNC,
		Descriptor: int(file),
		Deadline:   deadline,
		Deliver:    callback,
	}
	operating_system_operation_submit(state, operation)
}

// No kernel request exists for an empty transfer, thus callback delivery needs only the queue.
func operating_system_empty_transfer_complete(
	state *Operating_System, completion *time.Completion, callback time.Callback,
) {
	completion.Data = 0
	completion.Error = nil
	completion.Callback = func() { callback(completion) }
	state.Completed = append(state.Completed, completion)
}

// Submit asynchronous openat operation with NUL-terminated path owned until callback
// retire.
func operating_system_open_at(
	state *Operating_System, completion *time.Completion, directory nbio.File,
	file_path string, options nbio.Open_At_Options, callback time.Callback,
) {
	descriptor := int(directory)
	if directory == nbio.DIRECTORY_CURRENT {
		descriptor = platform_current_directory()
	}
	path := append([]byte(file_path), 0)
	operation := &Operating_System_Operation{
		Completion:   completion,
		Kind:         OPERATING_SYSTEM_OPERATION_OPEN_AT,
		Descriptor:   descriptor,
		File_Path:    path,
		Open_Options: options,
		Deliver:      callback,
	}
	operating_system_operation_submit(state, operation)
}

// Submit one directory pass. getdents has no asynchronous form on either backend, thus read run
// inline and completion retire on next drain, exactly as Status do.
func operating_system_directory_pass(
	state *Operating_System, completion *time.Completion, directory nbio.File, buffer []byte,
	callback nbio.Directory_Callback,
) {
	entries, pass_err := file_directory_pass(int(directory), buffer)
	completion.Callback = func() { callback(completion, entries, pass_err) }
	state.Completed = append(state.Completed, completion)
}

// Submit one mkdirat through platform scheduler. Mode travel in Open_Options because both
// operations carry path and creation mode, and scheduler already hold that field.
func operating_system_mkdir_at(
	state *Operating_System, completion *time.Completion, directory nbio.File,
	file_path string, mode uint32, callback time.Callback,
) {
	descriptor := int(directory)
	if directory == nbio.DIRECTORY_CURRENT {
		descriptor = platform_current_directory()
	}
	operating_system_operation_submit(state, &Operating_System_Operation{
		Completion:   completion,
		Kind:         OPERATING_SYSTEM_OPERATION_MKDIR_AT,
		Descriptor:   descriptor,
		File_Path:    append([]byte(file_path), 0),
		Open_Options: nbio.Open_At_Options{Mode: mode},
		Deliver:      callback,
	})
}

// Schedule positive timeout to fire when clock pass its deadline.
func operating_system_timeout(
	state *Operating_System, host time.Clock, completion *time.Completion,
	duration time.Duration, callback time.Callback,
) {
	if platform_uses_kernel_timeouts() {
		operation := &Operating_System_Operation{
			Completion: completion,
			Kind:       OPERATING_SYSTEM_OPERATION_TIMEOUT,
			Descriptor: -1,
			Timespec:   operating_system_timeout_span(duration),
			Deliver:    callback,
		}
		operating_system_operation_submit(state, operation)
		return
	}
	completion.Ready_At = host.Now_Monotonic() + time.Monotonic_Moment(duration)
	completion.Callback = func() { callback(completion) }
	operating_system_insert(state, completion)
}

// Keep backend-owned process bounds in the loop queue because their owner must withdraw them
// when another terminal event wins. Public Linux deadlines stay in io_uring and retire by kernel
// completion instead.
func operating_system_internal_timeout(
	state *Operating_System, completion *time.Completion, duration time.Duration,
	callback time.Callback,
) {
	completion.Ready_At = state.Host.Now_Monotonic() + time.Monotonic_Moment(duration)
	completion.Callback = func() { callback(completion) }
	operating_system_insert(state, completion)
}

// Withdraw one backend-owned bound before its resource can finish. It can already be ready but
// not delivered when another completion earlier in the same pass wins, thus both queues belong
// to cancellation.
func operating_system_internal_timeout_cancel(
	state *Operating_System, completion *time.Completion,
) {
	if !completion.Armed {
		return
	}
	state.Timeouts = operating_system_completion_remove(state.Timeouts, completion)
	state.Completed = operating_system_completion_remove(state.Completed, completion)
	completion.Armed = false
}

// Preserve queue order because equal-time completion order is observable by callbacks.
func operating_system_completion_remove(
	completions []*time.Completion, removed *time.Completion,
) (kept []*time.Completion) {
	kept = completions[:0]
	for _, completion := range completions {
		if completion != removed {
			kept = append(kept, completion)
		}
	}
	return kept
}

// Build driver — loop-advancing capability — over state. Only composition root or test hold it,
// never code that only submit IO.
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
			timeout time.Duration, done func() (finished bool),
		) (completed bool, err error) {
			err = operating_system_drive(state, func() (drive_err error) {
				completed, drive_err = operating_system_run_until(
					state, timeout, done,
				)
				return drive_err
			})
			return completed, err
		},
		Deinit: func() {
			operating_system_deinitialize(state)
		},
	}
}

// Drive until done, or until host deadline. Propagate every scheduler error.
func operating_system_run_until(
	state *Operating_System, timeout time.Duration, done func() (finished bool),
) (completed bool, err error) {
	deadline := state.Host.Now_Monotonic() + time.Monotonic_Moment(timeout)
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

// Operating system run until wait choose nonblocking first pass, then nearest deadline.
func operating_system_run_until_wait(
	state *Operating_System, deadline time.Monotonic_Moment, timeout time.Duration,
) (wait time.Monotonic_Moment, expired bool) {
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

// Report whether operation can wake unbounded drive.
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

// Run pump as top-level drive. Panic when drive is already in progress, thus Run* called from
// inside completion callback fail loud instead of re-enter driver. Internal run functions call
// one another direct, not through here, thus own iteration of drive does not trip it.
func operating_system_drive(state *Operating_System, pump func() (err error)) (err error) {
	invariant.Always(!state.Drive_Active,
		"A drive begins at top level, never from within a completion callback.")
	state.Drive_Active = true
	defer func() { state.Drive_Active = false }()
	return pump()
}

// Run one nonblocking flush.
func operating_system_run(state *Operating_System) (err error) {
	return operating_system_flush(state, 0)
}

// Run flush passes until duration elapse, with deadline passed direct to platform.
func operating_system_run_for(state *Operating_System, duration time.Duration) (err error) {
	deadline := state.Host.Now_Monotonic() + time.Monotonic_Moment(duration)
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

// Operating system flush lets a ready kernel event produce its terminal syscall result before
// expiration retires registrations that remain pending.
func operating_system_flush(state *Operating_System, wait time.Monotonic_Moment) (err error) {
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
	operation_expire_err := operating_system_expire_operations(state)
	if operation_expire_err != nil {
		return operation_expire_err
	}
	operating_system_flush_completed(state)
	return platform_flush_submissions(state)
}

// Operating system wait cap pull outer wait in to nearest Darwin timeout and signal poll.
func operating_system_wait_cap(
	state *Operating_System, wait time.Monotonic_Moment,
) (capped time.Monotonic_Moment) {
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
		interval := time.Monotonic_Moment(SIGNAL_POLL_INTERVAL)
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

// Move every elapsed timeout and finite extension waiter into completed queue.
func operating_system_expire(state *Operating_System) (err error) {
	now := state.Host.Now_Monotonic()
	for len(state.Timeouts) > 0 && state.Timeouts[0].Ready_At <= now {
		expired := state.Timeouts[0]
		state.Timeouts = state.Timeouts[1:]
		state.Completed = append(state.Completed, expired)
	}
	operating_system_expire_signals(state, now)
	return nil
}

// Retire only operations still pending after the platform consumed every ready event.
func operating_system_expire_operations(state *Operating_System) (err error) {
	if platform_uses_kernel_timeouts() {
		return nil
	}
	now := state.Host.Now_Monotonic()
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
		operating_system_operation_complete(
			state, operation, operating_system_timeout_result(operation),
			time.Deadline_Exceeded,
		)
	}
	return nil
}

// Run and clear every ready completion callback. Return each completion to idle before its
// callback fire — thus callback may legally resubmit its own completion.
func operating_system_flush_completed(state *Operating_System) {
	for len(state.Completed) > 0 {
		completion := state.Completed[0]
		state.Completed = state.Completed[1:]
		invariant.Always(completion.Armed, "A delivered completion was armed.")
		completion.Armed = false
		completion.Callback()
	}
}

// Insert completion into timeout queue in Ready_At order.
func operating_system_insert(state *Operating_System, completion *time.Completion) {
	index := 0
	for index < len(state.Timeouts) && state.Timeouts[index].Ready_At <= completion.Ready_At {
		index++
	}
	state.Timeouts = append(state.Timeouts, nil)
	copy(state.Timeouts[index+1:], state.Timeouts[index:])
	state.Timeouts[index] = completion
}

// Submit accept through kqueue eager/requeue path, or io_uring ACCEPT opcode.
func operating_system_accept(
	state *Operating_System, completion *time.Completion, listener nbio.File,
	timeout time.Duration, callback time.Callback,
) {
	operation := &Operating_System_Operation{
		Completion:    completion,
		Kind:          OPERATING_SYSTEM_OPERATION_ACCEPT,
		Descriptor:    int(listener),
		Deadline:      state.Host.Now_Monotonic() + time.Monotonic_Moment(timeout),
		Deadline_Span: operating_system_timeout_span(timeout),
		Deliver:       callback,
	}
	operating_system_operation_submit(state, operation)
}

// Begin connection on caller-owned socket and report only connect result. How backend reach that
// result is its own: io_uring submit connect, and kqueue arm socket for writability, then issue
// connect itself.
func operating_system_connect(
	state *Operating_System, completion *time.Completion, socket nbio.File,
	address nbio.Address, timeout time.Duration, callback time.Callback,
) {
	operation := &Operating_System_Operation{
		Completion:    completion,
		Kind:          OPERATING_SYSTEM_OPERATION_CONNECT,
		Descriptor:    int(socket),
		Address:       address,
		Deadline:      state.Host.Now_Monotonic() + time.Monotonic_Moment(timeout),
		Deadline_Span: operating_system_timeout_span(timeout),
		Deliver:       callback,
	}
	operating_system_operation_submit(state, operation)
}

// Read once from socket into buffer and report byte count. io_uring submit receive, and kqueue
// arm socket for readability, then read it.
func operating_system_receive(
	state *Operating_System, completion *time.Completion, socket nbio.File, buffer []byte,
	timeout time.Duration, callback time.Callback,
) {
	operation := &Operating_System_Operation{
		Completion: completion,
		Kind:       OPERATING_SYSTEM_OPERATION_RECEIVE,
		Descriptor: int(socket), Buffer: platform_buffer_limit(buffer),
		Deadline:      state.Host.Now_Monotonic() + time.Monotonic_Moment(timeout),
		Deadline_Span: operating_system_timeout_span(timeout),
		Deliver:       callback,
	}
	operating_system_operation_submit(state, operation)
}

// Write buffer once to socket and report byte count. io_uring submit send, and kqueue arm socket
// for writability, then write it.
func operating_system_send(
	state *Operating_System, completion *time.Completion, socket nbio.File, buffer []byte,
	timeout time.Duration, callback time.Callback,
) {
	operation := &Operating_System_Operation{
		Completion: completion,
		Kind:       OPERATING_SYSTEM_OPERATION_SEND,
		Descriptor: int(socket), Buffer: platform_buffer_limit(buffer),
		Deadline:      state.Host.Now_Monotonic() + time.Monotonic_Moment(timeout),
		Deadline_Span: operating_system_timeout_span(timeout),
		Deliver:       callback,
	}
	operating_system_operation_submit(state, operation)
}

// Close file and queue its completion after operating_system_assert_file_drained enforce
// owner-side join.
func operating_system_close(
	state *Operating_System, completion *time.Completion, file nbio.File,
	callback time.Callback,
) {
	operation := &Operating_System_Operation{
		Completion: completion,
		Kind:       OPERATING_SYSTEM_OPERATION_CLOSE,
		Descriptor: int(file),
		Deliver:    callback,
	}
	operating_system_operation_submit(state, operation)
}

// Operating system deinitialize enforce join-before-deinit contract and release
// scheduler.
func operating_system_deinitialize(state *Operating_System) {
	invariant.Always(state.Extension_Submitted == 0,
		"Driver Deinit follows joining every repository-extension completion.")
	invariant.Always(len(state.Spawns) == 0,
		"Every spawned child is reaped before backend deinit.")
	invariant.Always(len(state.Timeouts) == 0,
		"Driver Deinit follows joining every userspace timeout.")
	invariant.Always(len(state.Operations) == 0,
		"Driver Deinit follows joining every operation.")
	if state.Signals != nil {
		signal.Stop(state.Signals)
	}
	platform_deinitialize(state)
}

// Register watcher for signal and start OS notification for it.
func operating_system_watch_signal(
	state *Operating_System, completion *time.Completion, kind sysos.Signal,
	deadline time.Duration, callback sysos.Signal_Callback,
) {
	operating_system_signal_ensure(state)
	system := signal_to_operating_system(kind)
	state.Signal_Waiters = append(state.Signal_Waiters, Signal_Waiter{
		System: system, Kind: kind, Completion: completion, Callback: callback,
		Deadline: state.Host.Now_Monotonic() + time.Monotonic_Moment(deadline),
	})
	signal.Notify(state.Signals, system)
}

// Make buffered signal channel on first watch.
func operating_system_signal_ensure(state *Operating_System) {
	if state.Signals != nil {
		return
	}
	state.Signals = make(chan os.Signal, SIGNAL_QUEUE_DEPTH)
}

// Map backend-independent signal to its OS signal.
func signal_to_operating_system(kind sysos.Signal) (system os.Signal) {
	if kind == sysos.SIGNAL_INTERRUPT {
		return syscall.SIGINT
	}
	return syscall.SIGTERM
}

// Drain delivered signals without block. Fire matching watchers.
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

// Fire every watcher matching received, one-shot. Keep rest for later delivery.
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

// Expire signal watchers before drain of os/signal channel, thus event first observed at deadline
// is discarded and cannot leak into next explicitly rearmed watch.
func operating_system_expire_signals(state *Operating_System, now time.Monotonic_Moment) {
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
