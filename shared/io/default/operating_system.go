// Package io is the operating-system backend for shared/io.
//
// FAITHFUL PORT: Darwin follows third-party/tigerbeetle/src/io/darwin.zig and Linux follows
// io/linux.zig. DO NOT DIVERGE. There is no generic Cancel. Descriptor owners perform Shutdown,
// join submitted operations, and then asynchronous Close as in message_bus.zig:1057-1160.
package io

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"

	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/io"
	"local/james-orcales/shared/time"
)

// Caps one Darwin kqueue changelist and event batch, matching TigerBeetle's fixed flush buffer.
const POLL_EVENTS_MAX = 256

// Poll_forever, passed as an idle gap, blocks the readiness poll until an event arrives rather
// than for a fixed span — the unbounded wait a run with no deadline needs, so the loop sleeps
// exactly until there is work instead of waking on an interval.
const POLL_FOREVER time.Moment = -1

// Buffers a few pending signals so a burst is not lost between drains.
const SIGNAL_QUEUE_DEPTH = 8

// Caps the idle gap while a signal watcher exists, since a signal does not wake the poll;
// the loop re-checks the signal channel at least this often.
const SIGNAL_POLL_INTERVAL = 10 * time.MILLISECOND

// Buffers submitted compute jobs so bursts do not block the loop thread.
const COMPUTE_QUEUE_DEPTH = 1024

// Holds the host backend's state: the completed queue, Darwin timeout queue, platform scheduler,
// and repository-extension effects which all retire through completed.
type Operating_System struct {
	// Host is the real clock; deadlines and waits are measured against it.
	Host time.Clock
	// Timeouts are pending timer completions ordered by Ready_At, earliest first.
	Timeouts []*io.Completion
	// Completed are completions whose callbacks are ready to run on the next drain.
	Completed []*io.Completion
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
	// Jobs carries compute work to the worker pool; nil until the first Compute.
	Jobs chan *Compute_Job
	// Results holds finished compute jobs handed back from workers, guarded by the mutex.
	Results []*Compute_Job
	// Posted holds completions finished off the loop thread (spawn), guarded by the mutex.
	Posted []*io.Completion
	// Results_Mutex guards Results and Posted, the only cross-thread state.
	Results_Mutex sync.Mutex
	// Wake_Event is TigerBeetle's Event used only to bridge marked repository extensions
	// back to the loop thread. It is an internal backend resource excluded from Raw_Open.
	Wake_Event io.Event
	// Wake_Completion is the one persistent listener rearmed after each extension drain.
	Wake_Completion io.Completion
	// Wake_Identifier is the immutable kernel token worker threads use to trigger Wake_Event.
	Wake_Identifier uint64
	// Wake_Active reports whether the extension Event has been opened and armed.
	Wake_Active bool
	// Compute_Active reports whether the worker pool has been started.
	Compute_Active bool
	// Extension_Submitted counts repository-extension completions not yet delivered to callers.
	Extension_Submitted int
	// Extension_Workers joins compute and spawn goroutines before the Event/backend closes.
	Extension_Workers sync.WaitGroup
	// Extension_Stop suppresses Event reattachment during Deinit's internal final trigger.
	Extension_Stop bool
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
	Kind io.Signal
	// Completion is the caller-owned completion fired on delivery.
	Completion *io.Completion
	// Callback is the typed callback run with the delivered signal.
	Callback io.Signal_Callback
	// Deadline is the finite moment when this repository-extension operation retires.
	Deadline time.Moment
}

// One offloaded compute job: the work to run on a worker thread and the completion and
// callback to fire back on the loop thread once it finishes.
type Compute_Job struct {
	// Completion is the caller-owned completion fired once the work finishes.
	Completion *io.Completion
	// Callback is the typed callback run on the loop thread after the work.
	Callback io.Compute_Callback
	// Work is the offloaded function; it must touch only memory the loop leaves alone.
	Work func()
}

// New_Operating_System_IO eagerly creates the TigerBeetle scheduler. Entries is the io_uring
// queue size on Linux and is accepted but ignored by kqueue on Darwin; it must fit TigerBeetle's
// u12 contract. Flags are passed directly to io_uring setup and ignored by Darwin.
func New_Operating_System_IO(
	host time.Clock, entries uint16, flags uint32,
) (loop io.IO, driver io.Driver, err error) {
	if entries == 0 {
		return io.IO{}, io.Driver{}, errors.New(
			"io: scheduler entries must be in [1, 4095]",
		)
	}
	if entries > 4095 {
		return io.IO{}, io.Driver{}, errors.New(
			"io: scheduler entries must be in [1, 4095]",
		)
	}
	platform, initialize_err := platform_initialize(entries, flags)
	if initialize_err != nil {
		return io.IO{}, io.Driver{}, initialize_err
	}
	state := &Operating_System{
		Host:       host,
		Platform:   platform,
		Operations: map[uint64]*Operating_System_Operation{},
		Raw_Open:   map[int]bool{},
	}
	operating_system_wire_file(state, &loop)
	operating_system_wire_timer(state, &loop)
	operating_system_wire_socket(state, &loop)
	operating_system_wire_effects(state, &loop)
	operating_system_wire_platform(state, &loop)
	return loop, operating_system_to_driver(state), nil
}

// Arms a completion through the idle-to-armed lifecycle edge.
func operating_system_submit(completion *io.Completion) {
	original := completion.Self == nil || completion.Self == completion
	invariant.Always(original,
		"A submitted completion is its own original, never a by-value copy.")
	completion.Self = completion
	completion.Next_Tick = false
	io.Completion_Transition(&io.Completion_Transition_Input{
		Completion: completion, From: io.COMPLETION_IDLE, To: io.COMPLETION_ARMED,
	})
}

// Wires the signal-watch and compute-offload operations onto loop.
func operating_system_wire_effects(state *Operating_System, loop *io.IO) {
	loop.Watch_Signal = func(
		completion *io.Completion, callback io.Signal_Callback, signal io.Signal,
		deadline time.Duration,
	) {
		invariant.Always(deadline > 0, "A signal-watch deadline is positive and finite.")
		state.Extension_Submitted++
		operating_system_submit(completion)
		operating_system_watch_signal(state, completion, func(
			completed *io.Completion, delivered io.Signal, watch_err error,
		) {
			state.Extension_Submitted--
			callback(completed, delivered, watch_err)
		}, signal, deadline)
	}
	loop.Compute = func(completion *io.Completion, callback io.Compute_Callback, work func()) {
		state.Extension_Submitted++
		operating_system_submit(completion)
		operating_system_compute_submit(state, completion, func(completed *io.Completion) {
			state.Extension_Submitted--
			callback(completed)
		}, work)
	}
	loop.Spawn = func(
		completion *io.Completion, callback io.Process_Callback, request io.Process_Request,
		deadline time.Duration,
	) {
		invariant.Always(deadline > 0, "A spawn deadline is positive and finite.")
		state.Extension_Submitted++
		operating_system_submit(completion)
		operating_system_spawn(state, completion, func(
			completed *io.Completion, result io.Process_Result, spawn_err error,
		) {
			state.Extension_Submitted--
			callback(completed, result, spawn_err)
		}, request, deadline)
	}
	// Repository extension: successful exec atomically closes every CLOEXEC descriptor.
	// TigerBeetle's multiversion behavior does not inherit listeners; the new image rebinds.
	loop.Self_Exec = func(path string, argv []string, environment []string) (err error) {
		if environment == nil {
			environment = os.Environ()
		}
		return Self_Exec(Self_Exec_Input{
			Path: path, Arguments: argv, Environment: environment,
		})
	}
}

// Self_Exec_Input is an explicit process-image replacement, including its complete environment.
type Self_Exec_Input struct {
	// Path is the executable image that replaces the current process.
	Path string
	// Arguments become the replacement process argv.
	Arguments []string
	// Environment becomes the replacement process environment; an empty slice inherits nothing.
	Environment []string
}

// Self_Exec replaces the current process image with exactly the injected arguments and environment.
func Self_Exec(input Self_Exec_Input) (err error) {
	return syscall.Exec(input.Path, input.Arguments, input.Environment)
}

// Runs request's command on its own goroutine and posts the finished result to the loop.
func operating_system_spawn(
	state *Operating_System, completion *io.Completion,
	callback io.Process_Callback, request io.Process_Request, deadline time.Duration,
) {
	operating_system_wake_ensure(state)
	invariant.Always(state.Wake_Active, "A Spawn posts through an open TigerBeetle Event.")
	state.Extension_Workers.Add(1)
	go process_run(state, completion, callback, request, deadline)
}

// Executes the command and posts its result — or a start failure — back on the loop.
func process_run(
	state *Operating_System, completion *io.Completion,
	callback io.Process_Callback, request io.Process_Request, deadline time.Duration,
) {
	defer state.Extension_Workers.Done()
	start := state.Host.Now_Monotonic()
	result, err := process_execute(request, deadline)
	result.Usage.Wall = time.Duration(int64(state.Host.Now_Monotonic()) - int64(start))
	operating_system_post(state, completion, func() {
		callback(completion, result, err)
	})
}

// Runs the command to completion, capturing stdout and stderr. A non-zero exit is
// reported in the result with a nil error; a failure to start is the error.
func process_execute(
	request io.Process_Request, deadline time.Duration,
) (result io.Process_Result, err error) {
	context_value, cancel := process_context(context.WithTimeout, deadline)
	defer cancel()
	command := exec.CommandContext(context_value, request.Path, request.Arguments...)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	deadline_triggered := atomic.Bool{}
	command.Cancel = func() (cancel_err error) {
		deadline_triggered.Store(true)
		kill_err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		if kill_err == syscall.ESRCH {
			return os.ErrProcessDone
		}
		return kill_err
	}
	command.WaitDelay = 1_000_000_000
	command.Dir = request.Working_Directory
	command.Env = request.Environment
	if len(request.Input) > 0 {
		command.Stdin = bytes.NewReader(request.Input)
	}
	// A caller-supplied sink streams the child's output live as it runs; without one the
	// output is captured into a buffer and returned. The two are exclusive: streamed
	// output leaves the corresponding Result field empty.
	output := bytes.Buffer{}
	error_output := bytes.Buffer{}
	if request.Stdout != nil {
		command.Stdout = request.Stdout
	} else {
		command.Stdout = &output
	}
	if request.Stderr != nil {
		command.Stderr = request.Stderr
	} else {
		command.Stderr = &error_output
	}
	run_err := command.Run()
	result.Output = output.Bytes()
	result.Error_Output = error_output.Bytes()
	if command.ProcessState != nil {
		result.Exit = command.ProcessState.ExitCode()
		result.Usage = process_usage(command.ProcessState)
	}
	if deadline_triggered.Load() {
		return result, io.Deadline_Exceeded
	}
	if errors.Is(context_value.Err(), context.DeadlineExceeded) {
		return result, io.Deadline_Exceeded
	}
	return process_execute_result(result, run_err)
}

// Converts the repository duration into context.WithTimeout's duration type without importing
// the standard time package outside shared/time/default.
func process_context[Duration ~int64](
	with_timeout func(context.Context, Duration) (
		context_value context.Context, cancel context.CancelFunc,
	),
	deadline time.Duration,
) (context_value context.Context, cancel context.CancelFunc) {
	return with_timeout(context.Background(), Duration(deadline))
}

// Distinguishes a non-zero exit (reported in the result, nil error) from a real
// start/run failure (returned as the error).
func process_execute_result(
	result io.Process_Result, run_err error,
) (final io.Process_Result, err error) {
	if run_err == nil {
		return result, nil
	}
	var exit_err *exec.ExitError
	if errors.As(run_err, &exit_err) {
		return result, nil
	}
	return result, run_err
}

// Extracts CPU and peak-RSS accounting from a finished process's state.
func process_usage(state *os.ProcessState) (usage io.Process_Usage) {
	usage.CPU_User = time.Duration(int64(state.UserTime()))
	usage.CPU_System = time.Duration(int64(state.SystemTime()))
	rusage, ok := state.SysUsage().(*syscall.Rusage)
	if !ok {
		return usage
	}
	usage.RSS_Bytes_Max = process_rss_bytes(int64(rusage.Maxrss))
	return usage
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
		completion *io.Completion, callback io.Callback,
		file io.File, buffer []byte, offset int64,
	) {
		operating_system_submit(completion)
		operating_system_read(state, completion, callback, file, buffer, offset)
	}
	loop.Write = func(
		completion *io.Completion, callback io.Callback,
		file io.File, buffer []byte, offset int64,
	) {
		operating_system_submit(completion)
		operating_system_write(state, completion, callback, file, buffer, offset)
	}
	loop.Fsync = func(
		completion *io.Completion, callback io.Timeout_Callback, file io.File,
	) {
		operating_system_submit(completion)
		operating_system_fsync(state, completion, callback, file)
	}
	loop.Open_At = func(
		completion *io.Completion, callback io.File_Callback, directory io.File,
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
	loop.Open = func(path string) (file io.File, err error) {
		descriptor, open_err := file_open(path)
		return io.File(descriptor), open_err
	}
	loop.Create = func(path string) (file io.File, err error) {
		descriptor, create_err := file_create(path)
		return io.File(descriptor), create_err
	}
	loop.Read_Directory = func(path string) (entries []io.Directory_Entry, err error) {
		return file_read_directory(path)
	}
	loop.Status = func(path string) (status io.File_Status, err error) {
		return file_status(path)
	}
	loop.Make_Directory = func(path string) (err error) {
		return file_make_directory(path)
	}
}

// Creates path and any missing parents; an existing directory is not an error, so a
// repeated mkdir converges. os.MkdirAll is the gateway's bounded parent-creating mkdir.
func file_make_directory(path string) (err error) {
	return os.MkdirAll(path, 0o755)
}

// Wires Timeout, Next_Tick, Reset_Next_Tick, and both close primitives onto loop.
func operating_system_wire_timer(state *Operating_System, loop *io.IO) {
	loop.Timeout = func(
		completion *io.Completion, callback io.Timeout_Callback, duration time.Duration,
	) {
		invariant.Always(
			duration > 0, "A timeout duration is positive; yields use Next_Tick.",
		)
		operating_system_submit(completion)
		operating_system_timeout(state, state.Host, completion, callback, duration)
	}
	loop.Next_Tick = func(
		completion *io.Completion, callback io.Next_Tick_Callback,
		source io.Next_Tick_Source,
	) {
		operating_system_submit(completion)
		completion.Next_Tick_Source = source
		completion.Next_Tick = true
		completion.Callback = func() { callback(completion) }
		state.Completed = append(state.Completed, completion)
	}
	loop.Reset_Next_Tick = func(source io.Next_Tick_Source) {
		operating_system_reset_next_tick(state, source)
	}
	loop.Open_Event = func() (event io.Event, err error) {
		return platform_event_open(state)
	}
	loop.Event_Listen = func(
		event io.Event, completion *io.Completion, callback io.Next_Tick_Callback,
	) {
		operating_system_submit(completion)
		operating_system_event_listen(state, event, completion, callback)
	}
	loop.Event_Trigger = func(event io.Event, completion *io.Completion) {
		platform_event_trigger(state, event, completion.Kernel_Identifier)
	}
	loop.Close_Event = func(event io.Event) {
		operating_system_assert_event_drained(state, event)
		platform_event_close(state, event)
	}
	loop.Close = func(completion *io.Completion, callback io.Timeout_Callback, file io.File) {
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
	state *Operating_System, event io.Event, completion *io.Completion,
	callback io.Next_Tick_Callback,
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
func operating_system_assert_event_drained(state *Operating_System, event io.Event) {
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
	loop.Listen = func(
		socket io.File, address io.Address, options io.Listen_Options,
	) (resolved io.Address, err error) {
		return socket_listen(int(socket), address, options)
	}
	loop.Accept = func(
		completion *io.Completion, callback io.Socket_Callback, listener io.File,
		deadline time.Duration,
	) {
		invariant.Always(deadline > 0, "An accept deadline is positive and finite.")
		operating_system_submit(completion)
		operating_system_accept(state, completion, callback, listener, deadline)
	}
	loop.Open_Socket_TCP = func(
		family io.Address_Family, options io.TCP_Options,
	) (socket io.File, err error) {
		descriptor, open_err := socket_open_tcp(family, options)
		if open_err != nil {
			return io.File(-1), open_err
		}
		state.Raw_Open[descriptor] = true
		return io.File(descriptor), nil
	}
	loop.Open_Socket_UDP = func(family io.Address_Family) (socket io.File, err error) {
		descriptor, open_err := socket_open_udp(family)
		if open_err != nil {
			return io.File(-1), open_err
		}
		state.Raw_Open[descriptor] = true
		return io.File(descriptor), nil
	}
	loop.Connect = func(
		completion *io.Completion, callback io.Timeout_Callback, socket io.File,
		address io.Address, deadline time.Duration,
	) {
		invariant.Always(deadline > 0, "A connect deadline is positive and finite.")
		operating_system_submit(completion)
		operating_system_connect(state, completion, callback, socket, address, deadline)
	}
	loop.Receive = func(
		completion *io.Completion, callback io.Callback, socket io.File, buffer []byte,
	) {
		operating_system_submit(completion)
		operating_system_receive(state, completion, callback, socket, buffer)
	}
	loop.Send = func(
		completion *io.Completion, callback io.Callback, socket io.File, buffer []byte,
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
	state *Operating_System, completion *io.Completion, callback io.Callback,
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
	state *Operating_System, completion *io.Completion, callback io.Callback,
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
	state *Operating_System, completion *io.Completion,
	callback io.Timeout_Callback, file io.File,
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
	state *Operating_System, completion *io.Completion, callback io.File_Callback,
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

// Schedules a positive timeout to fire when the clock passes its deadline.
func operating_system_timeout(
	state *Operating_System, host time.Clock, completion *io.Completion,
	callback io.Timeout_Callback, duration time.Duration,
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
func operating_system_to_driver(state *Operating_System) (driver io.Driver) {
	return io.Driver{
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
		Introspect: func() (counts io.Loop_Counts) {
			return operating_system_introspect(state)
		},
	}
}

// Samples the loop's queue depths for an admin state snapshot. Loop-thread fields are read
// directly — Introspect is root-only and called from the drive goroutine — but Posted and Results
// are written by off-loop workers under the mutex, so their lengths are taken under it.
func operating_system_introspect(state *Operating_System) (counts io.Loop_Counts) {
	state.Results_Mutex.Lock()
	posted_count := len(state.Posted)
	results_count := len(state.Results)
	state.Results_Mutex.Unlock()
	backlog, inflight, queued, kernel := platform_counts(state)
	return io.Loop_Counts{
		Completed:      len(state.Completed),
		Timeouts:       len(state.Timeouts),
		IO_Backlog:     backlog,
		IO_Inflight:    inflight,
		IO_Queued:      queued,
		IO_In_Kernel:   kernel,
		Signal_Waiters: len(state.Signal_Waiters),
		Posted:         posted_count,
		Results:        results_count,
		Raw_Open:       len(state.Raw_Open),
		Wake_Active:    state.Wake_Active,
		Compute_Active: state.Compute_Active,
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
	return state.Wake_Active
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
	operating_system_compute(state)
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
		operating_system_operation_complete(state, operation, -1, io.Deadline_Exceeded)
	}
	return nil
}

// Runs and clears every ready completion callback, returning each completion to idle
// before its callback fires — so a callback may legally resubmit its own completion.
func operating_system_flush_completed(state *Operating_System) {
	for len(state.Completed) > 0 {
		completion := state.Completed[0]
		state.Completed = state.Completed[1:]
		io.Completion_Transition(&io.Completion_Transition_Input{
			Completion: completion, From: io.COMPLETION_ARMED, To: io.COMPLETION_IDLE,
		})
		completion.Callback()
	}
}

// Removes every completed-queue next-tick operation for source and returns it to idle without
// invoking its callback (io/darwin.zig:783-796; io/linux.zig:354-367).
func operating_system_reset_next_tick(state *Operating_System, source io.Next_Tick_Source) {
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
		io.Completion_Transition(&io.Completion_Transition_Input{
			Completion: completion, From: io.COMPLETION_ARMED, To: io.COMPLETION_IDLE,
		})
	}
	state.Completed = kept
}

// Inserts completion into the timeout queue in Ready_At order.
func operating_system_insert(state *Operating_System, completion *io.Completion) {
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
	state *Operating_System, completion *io.Completion,
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

// Begins a connection to host_address:port on the caller-owned socket and arms it for
// writability; on readiness it reports only the connect result.
func operating_system_connect(
	state *Operating_System, completion *io.Completion,
	callback io.Timeout_Callback, socket io.File, address io.Address, deadline time.Duration,
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

// Arms socket for readability; on readiness it reads once into buffer and reports
// the byte count.
func operating_system_receive(
	state *Operating_System, completion *io.Completion,
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

// Arms socket for writability; on readiness it writes buffer once and reports the
// byte count.
func operating_system_send(
	state *Operating_System, completion *io.Completion,
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
	state *Operating_System, completion *io.Completion,
	callback io.Timeout_Callback, file io.File,
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
// the scheduler plus repository-extension wake resources.
func operating_system_deinitialize(state *Operating_System) {
	invariant.Always(state.Extension_Submitted == 0,
		"Driver Deinit follows joining every repository-extension completion.")
	invariant.Always(operating_system_user_operations(state) == 0,
		"Driver Deinit follows joining every TigerBeetle operation.")
	if state.Signals != nil {
		signal.Stop(state.Signals)
	}
	if state.Compute_Active {
		close(state.Jobs)
	}
	state.Extension_Workers.Wait()
	if state.Wake_Active {
		state.Extension_Stop = true
		platform_event_trigger(state, state.Wake_Event, state.Wake_Identifier)
		for state.Wake_Completion.State == io.COMPLETION_ARMED {
			flush_err := operating_system_flush(state, 0)
			invariant.Always(flush_err == nil,
				"The internal extension Event drains before backend deinit.")
		}
		platform_event_close(state, state.Wake_Event)
		state.Wake_Active = false
	}
	invariant.Always(len(state.Operations) == 0,
		"The internal Event is retired before backend deinit.")
	platform_deinitialize(state)
}

// Counts core operations other than the internal extension Event listener.
func operating_system_user_operations(state *Operating_System) (count int) {
	for _, operation := range state.Operations {
		internal_event := false
		if operation.Kind == OPERATING_SYSTEM_OPERATION_EVENT {
			internal_event = operation.Completion == &state.Wake_Completion
		}
		if !internal_event {
			count++
		}
	}
	return count
}

// Submits work to the worker pool, delivering callback on the loop thread once it
// finishes. A cancel on a compute is a no-op: the work is already queued off-thread.
func operating_system_compute_submit(
	state *Operating_System, completion *io.Completion,
	callback io.Compute_Callback, work func(),
) {
	operating_system_compute_ensure(state)
	state.Jobs <- &Compute_Job{Completion: completion, Callback: callback, Work: work}
}

// Starts the worker pool and the wake pipe on the first Compute.
func operating_system_compute_ensure(state *Operating_System) {
	if state.Compute_Active {
		return
	}
	operating_system_wake_ensure(state)
	state.Jobs = make(chan *Compute_Job, COMPUTE_QUEUE_DEPTH)
	state.Compute_Active = true
	workers := compute_worker_count()
	state.Extension_Workers.Add(workers)
	for index := 0; index < workers; index++ {
		go operating_system_compute_worker(state)
	}
}

// Returns the worker-pool size, one per GOMAXPROCS, floored at one.
func compute_worker_count() (workers int) {
	workers = runtime.GOMAXPROCS(0)
	if workers < 1 {
		return 1
	}
	return workers
}

// Runs jobs off the loop thread, pushing each finished job back through the mutex-guarded
// Results and waking the loop.
func operating_system_compute_worker(state *Operating_System) {
	defer state.Extension_Workers.Done()
	for job := range state.Jobs {
		job.Work()
		state.Results_Mutex.Lock()
		state.Results = append(state.Results, job)
		state.Results_Mutex.Unlock()
		platform_event_trigger(state, state.Wake_Event, state.Wake_Identifier)
	}
}

// Opens and arms TigerBeetle's Event primitive so marked repository extensions can wake the loop.
func operating_system_wake_ensure(state *Operating_System) {
	if state.Wake_Active {
		return
	}
	event, err := platform_event_open(state)
	if err != nil {
		return
	}
	state.Wake_Event = event
	state.Wake_Active = true
	operating_system_wake_listen(state)
	state.Wake_Identifier = state.Wake_Completion.Kernel_Identifier
}

// Rearms the persistent extension Event listener; its callback drains every result accumulated by
// workers before attaching the same completion again.
func operating_system_wake_listen(state *Operating_System) {
	operating_system_submit(&state.Wake_Completion)
	operating_system_event_listen(
		state, state.Wake_Event, &state.Wake_Completion,
		func(completion *io.Completion) {
			invariant.Always(completion == &state.Wake_Completion,
				"The internal Event delivers its registered completion.")
		},
	)
}

// Drains finished compute jobs and posted spawn completions onto the completed queue, on
// the loop thread; a no-op until the wake pipe exists.
func operating_system_compute(state *Operating_System) {
	if !state.Wake_Active {
		return
	}
	if !state.Extension_Stop {
		if state.Wake_Completion.State == io.COMPLETION_IDLE {
			operating_system_wake_listen(state)
		}
	}
	state.Results_Mutex.Lock()
	results := state.Results
	posted := state.Posted
	state.Results = nil
	state.Posted = nil
	state.Results_Mutex.Unlock()
	for index := 0; index < len(results); index++ {
		operating_system_compute_finish(state, results[index])
	}
	state.Completed = append(state.Completed, posted...)
}

// Queues a finished compute job's callback to run on the next flush.
func operating_system_compute_finish(state *Operating_System, job *Compute_Job) {
	completion := job.Completion
	callback := job.Callback
	completion.Callback = func() { callback(completion) }
	state.Completed = append(state.Completed, completion)
}

// Registers a watcher for signal and starts OS notification for it.
func operating_system_watch_signal(
	state *Operating_System, completion *io.Completion,
	callback io.Signal_Callback, kind io.Signal, deadline time.Duration,
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
func signal_to_operating_system(kind io.Signal) (system os.Signal) {
	if kind == io.SIGNAL_INTERRUPT {
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
			expired.Callback(expired.Completion, io.Signal(-1), io.Deadline_Exceeded)
		}
		state.Completed = append(state.Completed, expired.Completion)
	}
	state.Signal_Waiters = kept
}

// Posts a completion finished off the loop thread: records its callback under the mutex
// and wakes the loop to run it on the next drain.
func operating_system_post(
	state *Operating_System, completion *io.Completion, callback func(),
) {
	completion.Callback = callback
	state.Results_Mutex.Lock()
	state.Posted = append(state.Posted, completion)
	state.Results_Mutex.Unlock()
	platform_event_trigger(state, state.Wake_Event, state.Wake_Identifier)
}
