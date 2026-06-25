// Package io is the composition tier: the operating-system-backed async IO, the Go
// translation of TigerBeetle's io/darwin.zig and io/linux.zig. It declares package
// io so callers import ".../io/default" and read it as the library with no alias.
package io

import (
	"bytes"
	"crypto/tls"
	"errors"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"syscall"

	"github.com/james-orcales/james-orcales/shared/io"
	"github.com/james-orcales/james-orcales/shared/time"
)

// Caps the ready set one poll_file_wait returns; sized to drain a busy loop in few
// syscalls without an unbounded buffer.
const poll_events_max = 64

// Operating_system_tick bounds each Run_Until pump step: the loop blocks at most this
// long waiting for real events before re-checking done, so the pump neither spins nor
// oversleeps.
const operating_system_tick = 10 * time.Millisecond

// Bounds one wake-pipe drain so a flood of pokes cannot spin the loop.
const wake_drain_passes_max = 16

// Buffers a few pending signals so a burst is not lost between drains.
const signal_queue_depth = 8

// Caps the idle gap while a signal watcher exists, since a signal does not wake the poll;
// the loop re-checks the signal channel at least this often.
const signal_poll_interval = 10 * time.Millisecond

// Buffers submitted compute jobs so bursts do not block the loop thread.
const compute_queue_depth = 1024

// Buffers a TLS connection's in-flight requests.
const tls_request_depth = 4

// The synthetic-descriptor base for TLS connections, above any real file descriptor, so
// Receive/Send/Close can route a TLS socket to its goroutine by the descriptor alone.
const tls_file_base io.File = 1 << 30

// The tls_kind type names the request a TLS connection goroutine handles.
type tls_kind int

// The tls_send kind writes plaintext through the TLS connection.
const tls_send tls_kind = 0

// The tls_receive kind reads plaintext from the TLS connection.
const tls_receive tls_kind = 1

// The tls_close kind shuts the TLS connection down.
const tls_close tls_kind = 2

// One ready descriptor the poll reports, decoded from the platform's native event
// into a direction the dispatch understands.
type poll_ready struct {
	// Descriptor is the socket the readiness applies to.
	Descriptor int
	// Writable is true for write readiness, false for read readiness.
	Writable bool
}

// One pending socket completion: the syscall to run when its descriptor becomes
// ready.
type socket_operation struct {
	// Completion is the caller-owned completion this operation belongs to.
	Completion *io.Completion
	// Perform runs the non-blocking syscall and fires the callback, returning false
	// when the syscall reported EAGAIN so the operation stays armed.
	Perform func() (done bool)
}

// Holds the host backend's state: the injected clock, the timeout and completion
// queues the loop drains, and the readiness poll with its per-descriptor socket
// waiters, mirroring TigerBeetle's IO struct.
type operating_system struct {
	// Host is the real clock; deadlines and waits are measured against it.
	Host time.Clock
	// Timeouts are pending timer completions ordered by Ready_At, earliest first.
	Timeouts []*io.Completion
	// Completed are completions whose callbacks are ready to run on the next drain.
	Completed []*io.Completion
	// Poll is the kqueue/epoll descriptor, created on the first socket operation.
	Poll poll_file
	// Poll_Active reports whether Poll has been created.
	Poll_Active bool
	// Read_Waiters maps a descriptor to the operation awaiting its readability.
	Read_Waiters map[int]*socket_operation
	// Write_Waiters maps a descriptor to the operation awaiting its writability.
	Write_Waiters map[int]*socket_operation
	// Signals receives OS signals from the os/signal notifier; nil until the first watch.
	Signals chan os.Signal
	// Signal_Waiters are the registered signal watchers, fired one-shot on delivery.
	Signal_Waiters []signal_waiter
	// Jobs carries compute work to the worker pool; nil until the first Compute.
	Jobs chan *compute_job
	// Results holds finished compute jobs handed back from workers, guarded by the mutex.
	Results []*compute_job
	// Posted holds completions finished off the loop thread (TLS), guarded by the mutex.
	Posted []*io.Completion
	// Results_Mutex guards Results and Posted, the only cross-thread state.
	Results_Mutex sync.Mutex
	// Wake_Read and Wake_Write are the self-pipe ends that wake a blocked poll.
	Wake_Read  int
	Wake_Write int
	// Wake_Active reports whether the wake pipe has been created and armed.
	Wake_Active bool
	// Compute_Active reports whether the worker pool has been started.
	Compute_Active bool
	// TLS maps a synthetic descriptor to its connection goroutine's request channel.
	TLS map[io.File]*tls_connection
	// Next_TLS is the synthetic TLS descriptor counter, based at tls_file_base.
	Next_TLS io.File
}

// One registered signal watcher: the OS signal it awaits, its backend-independent kind,
// and the completion and callback to fire once on delivery.
type signal_waiter struct {
	// System is the OS signal this watcher awaits.
	System os.Signal
	// Kind is the backend-independent signal reported to the callback.
	Kind io.Signal
	// Completion is the caller-owned completion fired on delivery.
	Completion *io.Completion
	// Callback is the typed callback run with the delivered signal.
	Callback io.Signal_Callback
}

// One offloaded compute job: the work to run on a worker thread and the completion and
// callback to fire back on the loop thread once it finishes.
type compute_job struct {
	// Completion is the caller-owned completion fired once the work finishes.
	Completion *io.Completion
	// Callback is the typed callback run on the loop thread after the work.
	Callback io.Compute_Callback
	// Work is the offloaded function; it must touch only memory the loop leaves alone.
	Work func()
}

// One live TLS connection, serviced by its own goroutine that owns the tls.Conn and
// reads requests off Requests.
type tls_connection struct {
	// Requests carries send/receive/close requests to the connection goroutine.
	Requests chan tls_request
}

// One request to a TLS connection goroutine.
type tls_request struct {
	// Kind is the operation to perform.
	Kind tls_kind
	// Completion is the caller-owned completion fired when the request finishes.
	Completion *io.Completion
	// Buffer is the plaintext to send, or the destination for a receive.
	Buffer []byte
	// Byte_Callback reports a send's or receive's byte count.
	Byte_Callback io.Callback
	// Close_Callback reports a close's result.
	Close_Callback io.Timeout_Callback
}

// The endpoint and verification policy for an outbound TLS connection.
type tls_target struct {
	// Host is the endpoint host.
	Host string
	// Port is the endpoint port.
	Port int
	// Server_Name is the SNI name and the name the certificate is verified against.
	Server_Name string
	// Insecure skips certificate verification (Connect_Insecure).
	Insecure bool
	// Completion is the caller-owned completion fired when the connect finishes.
	Completion *io.Completion
	// Callback reports the connected socket or the connect error.
	Callback io.Socket_Callback
}

// New_Operating_System_IO returns an IO backed by the host operating system. File
// reads and writes run inside the loop via pread/pwrite; sockets ride a real
// kqueue/epoll readiness loop; timeouts fire when the clock passes their deadline,
// and Run_For blocks real time bounded by the nearest deadline or socket event —
// the same deadline-bounded wait TigerBeetle performs in kevent/io_uring.
func New_Operating_System_IO(host time.Clock) (loop io.IO, driver io.Driver) {
	state := &operating_system{Host: host}
	operating_system_wire_file(state, &loop)
	operating_system_wire_timer(state, &loop)
	operating_system_wire_socket(state, &loop)
	operating_system_wire_secure(state, &loop)
	operating_system_wire_effects(state, &loop)
	return loop, operating_system_to_driver(state)
}

// Wires the TLS socket operations — accept-secure, connect-secure, connect-insecure —
// onto loop.
func operating_system_wire_secure(state *operating_system, loop *io.IO) {
	loop.Accept_Secure = func(
		completion *io.Completion, callback io.Socket_Callback,
		listener io.File, certificate func() (value any),
	) {
		operating_system_accept_secure(state, completion, callback, listener, certificate)
	}
	loop.Connect_Secure = func(
		completion *io.Completion, callback io.Socket_Callback,
		host_address string, port int, server_name string,
	) {
		operating_system_connect_secure(state, tls_target{
			Host: host_address, Port: port, Server_Name: server_name,
			Completion: completion, Callback: callback,
		})
	}
	loop.Connect_Insecure = func(
		completion *io.Completion, callback io.Socket_Callback,
		host_address string, port int, server_name string,
	) {
		operating_system_connect_secure(state, tls_target{
			Host: host_address, Port: port, Server_Name: server_name, Insecure: true,
			Completion: completion, Callback: callback,
		})
	}
}

// Wires the signal-watch and compute-offload operations onto loop.
func operating_system_wire_effects(state *operating_system, loop *io.IO) {
	loop.Watch_Signal = func(
		completion *io.Completion, callback io.Signal_Callback, signal io.Signal,
	) {
		operating_system_watch_signal(state, completion, callback, signal)
	}
	loop.Compute = func(completion *io.Completion, callback io.Compute_Callback, work func()) {
		operating_system_compute_submit(state, completion, callback, work)
	}
	loop.Spawn = func(
		completion *io.Completion, callback io.Process_Callback, request io.Process_Request,
	) {
		operating_system_spawn(state, completion, callback, request)
	}
}

// Runs request's command on its own goroutine and posts the finished result to the loop.
func operating_system_spawn(
	state *operating_system, completion *io.Completion,
	callback io.Process_Callback, request io.Process_Request,
) {
	operating_system_wake_ensure(state)
	if !state.Wake_Active {
		completion.Callback = func() {
			callback(completion, io.Process_Result{},
				errors.New("io: wake pipe unavailable"))
		}
		state.Completed = append(state.Completed, completion)
		return
	}
	go process_run(state, completion, callback, request)
}

// Executes the command and posts its result — or a start failure — back on the loop.
func process_run(
	state *operating_system, completion *io.Completion,
	callback io.Process_Callback, request io.Process_Request,
) {
	start := state.Host.Now_Monotonic()
	result, err := process_execute(request)
	result.Usage.Wall = time.Duration(int64(state.Host.Now_Monotonic()) - int64(start))
	operating_system_post(state, completion, func() {
		callback(completion, result, err)
	})
}

// Runs the command to completion, capturing stdout and stderr. A non-zero exit is
// reported in the result with a nil error; a failure to start is the error.
func process_execute(request io.Process_Request) (result io.Process_Result, err error) {
	command := exec.Command(request.Path, request.Arguments...)
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
	return process_execute_result(result, run_err)
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
func operating_system_wire_file(state *operating_system, loop *io.IO) {
	loop.Read = func(
		completion *io.Completion, callback io.Callback,
		file io.File, buffer []byte, offset int64,
	) {
		operating_system_read(state, completion, callback, file, buffer, offset)
	}
	loop.Write = func(
		completion *io.Completion, callback io.Callback,
		file io.File, buffer []byte, offset int64,
	) {
		operating_system_write(state, completion, callback, file, buffer, offset)
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

// Wires the lifecycle operations — timeout, close, cancel — onto loop.
func operating_system_wire_timer(state *operating_system, loop *io.IO) {
	loop.Timeout = func(
		completion *io.Completion, callback io.Timeout_Callback, duration time.Duration,
	) {
		operating_system_timeout(state, state.Host, completion, callback, duration)
	}
	loop.Close = func(completion *io.Completion, callback io.Timeout_Callback, file io.File) {
		operating_system_close(state, completion, callback, file)
	}
	loop.Cancel = func(completion *io.Completion) {
		operating_system_cancel(state, completion)
	}
}

// Wires the socket operations — listen, accept, connect, receive, send, peer address —
// onto loop.
func operating_system_wire_socket(state *operating_system, loop *io.IO) {
	loop.Listen = func(host_address string, port int) (listener io.File, err error) {
		descriptor, listen_err := socket_listen(host_address, port)
		return io.File(descriptor), listen_err
	}
	loop.Accept = func(
		completion *io.Completion, callback io.Socket_Callback, listener io.File,
	) {
		operating_system_accept(state, completion, callback, listener)
	}
	loop.Connect = func(
		completion *io.Completion, callback io.Socket_Callback,
		host_address string, port int,
	) {
		operating_system_connect(state, completion, callback, host_address, port)
	}
	loop.Receive = func(
		completion *io.Completion, callback io.Callback, socket io.File, buffer []byte,
	) {
		operating_system_receive(state, completion, callback, socket, buffer)
	}
	loop.Send = func(
		completion *io.Completion, callback io.Callback, socket io.File, buffer []byte,
	) {
		operating_system_send(state, completion, callback, socket, buffer)
	}
	loop.Peer_Address = func(file io.File) (address string, err error) {
		return socket_peer_address(int(file))
	}
}

// Queues a file read to run inside the loop, delivering the byte count — or the
// Cancelled error if the completion was cancelled first. Clears any stale Cancelled
// mark so a reused completion starts fresh.
func operating_system_read(
	state *operating_system, completion *io.Completion, callback io.Callback,
	file io.File, buffer []byte, offset int64,
) {
	completion.Cancelled = false
	completion.Callback = func() {
		if completion.Cancelled {
			callback(completion, 0, io.Cancelled)
			return
		}
		count, err := read_at(file, buffer, offset)
		callback(completion, count, err)
	}
	state.Completed = append(state.Completed, completion)
}

// Queues a file write to run inside the loop, delivering the byte count — or the
// Cancelled error if the completion was cancelled first.
func operating_system_write(
	state *operating_system, completion *io.Completion, callback io.Callback,
	file io.File, buffer []byte, offset int64,
) {
	completion.Cancelled = false
	completion.Callback = func() {
		if completion.Cancelled {
			callback(completion, 0, io.Cancelled)
			return
		}
		count, err := write_at(file, buffer, offset)
		callback(completion, count, err)
	}
	state.Completed = append(state.Completed, completion)
}

// Schedules a timeout to fire when the clock passes its deadline, delivering success —
// or the Cancelled error if the completion was cancelled first.
func operating_system_timeout(
	state *operating_system, host time.Clock, completion *io.Completion,
	callback io.Timeout_Callback, duration time.Duration,
) {
	completion.Cancelled = false
	completion.Ready_At = host.Now_Monotonic() + time.Moment(duration)
	completion.Callback = func() {
		if completion.Cancelled {
			callback(completion, io.Cancelled)
			return
		}
		callback(completion, nil)
	}
	operating_system_insert(state, completion)
}

// Builds the driver — the loop-advancing capability — over state; held only by the
// composition root or a test, never by code that merely submits IO.
func operating_system_to_driver(state *operating_system) (driver io.Driver) {
	return io.Driver{
		Run:     func() { operating_system_run(state) },
		Run_For: func(duration time.Duration) { operating_system_run_for(state, duration) },
		Run_Until: func(done func() (finished bool)) {
			for !done() {
				operating_system_run_for(state, operating_system_tick)
			}
		},
	}
}

// Runs ready completions and one non-blocking socket poll; the host clock moves on
// its own, so Run advances nothing.
func operating_system_run(state *operating_system) {
	operating_system_signals(state)
	operating_system_compute(state)
	operating_system_expire(state)
	operating_system_flush_completed(state)
	if !state.Poll_Active {
		return
	}
	operating_system_poll(state, 0)
}

// Drains ready completions until the duration elapses on the host clock, idling
// each gap in the socket poll (or sleeping when no socket is armed).
func operating_system_run_for(state *operating_system, duration time.Duration) {
	deadline := state.Host.Now_Monotonic() + time.Moment(duration)
	for state.Host.Now_Monotonic() < deadline {
		operating_system_signals(state)
		operating_system_compute(state)
		operating_system_expire(state)
		operating_system_flush_completed(state)
		now := state.Host.Now_Monotonic()
		if now >= deadline {
			return
		}
		operating_system_idle(state, operating_system_wake(state, deadline)-now)
	}
}

// Idles for gap nanoseconds by blocking on the readiness poll with that timeout: an
// empty poll wait is a portable real-time sleep, and an armed socket's readiness wakes
// it early. The block lives on the poll fd, not on the clock — the clock is read-only,
// and the backend must not import stdlib time (the time/default gateway's alone).
func operating_system_idle(state *operating_system, gap time.Moment) {
	if gap <= 0 {
		return
	}
	gap = operating_system_signal_cap(state, gap)
	operating_system_poll_ensure(state)
	operating_system_poll(state, int64(gap))
}

// Caps gap while a signal watcher exists, since a delivered signal does not wake the
// poll; the loop must re-check the signal channel within one interval.
func operating_system_signal_cap(
	state *operating_system, gap time.Moment,
) (capped time.Moment) {
	if len(state.Signal_Waiters) == 0 {
		return gap
	}
	interval := time.Moment(signal_poll_interval)
	if gap > interval {
		return interval
	}
	return gap
}

// Moves every timeout whose deadline has passed into the completed queue.
func operating_system_expire(state *operating_system) {
	now := state.Host.Now_Monotonic()
	for len(state.Timeouts) > 0 && state.Timeouts[0].Ready_At <= now {
		expired := state.Timeouts[0]
		state.Timeouts = state.Timeouts[1:]
		state.Completed = append(state.Completed, expired)
	}
}

// Runs and clears every ready completion callback.
func operating_system_flush_completed(state *operating_system) {
	ready := state.Completed
	state.Completed = nil
	for index := 0; index < len(ready); index++ {
		ready[index].Callback()
	}
}

// Returns the earliest moment the loop must wake: the outer deadline, pulled in by
// the nearest pending timeout.
func operating_system_wake(state *operating_system, deadline time.Moment) (wake time.Moment) {
	wake = deadline
	if len(state.Timeouts) == 0 {
		return wake
	}
	if state.Timeouts[0].Ready_At < wake {
		return state.Timeouts[0].Ready_At
	}
	return wake
}

// Polls the readiness set with the given nanosecond timeout and dispatches each
// ready descriptor to its waiting operation.
func operating_system_poll(state *operating_system, timeout_ns int64) {
	ready, err := poll_file_wait(state.Poll, timeout_ns)
	if err != nil {
		return
	}
	for index := 0; index < len(ready); index++ {
		operating_system_dispatch(state, ready[index].Descriptor, ready[index].Writable)
	}
}

// Runs the operation waiting on descriptor in the given direction; on completion it
// drops the waiter and disarms the poll, leaving it armed on EAGAIN.
func operating_system_dispatch(state *operating_system, descriptor int, writable bool) {
	waiters := state.Read_Waiters
	if writable {
		waiters = state.Write_Waiters
	}
	operation := waiters[descriptor]
	if operation == nil {
		return
	}
	if !operation.Perform() {
		return
	}
	delete(waiters, descriptor)
	poll_file_disarm(state.Poll, descriptor, writable)
}

// Lazily creates the readiness poll on the first socket operation, so file-only and
// Windows callers never touch it.
func operating_system_poll_ensure(state *operating_system) {
	if state.Poll_Active {
		return
	}
	poll, err := poll_create()
	if err != nil {
		return
	}
	state.Poll = poll
	state.Poll_Active = true
	state.Read_Waiters = make(map[int]*socket_operation)
	state.Write_Waiters = make(map[int]*socket_operation)
}

// Inserts completion into the timeout queue in Ready_At order.
func operating_system_insert(state *operating_system, completion *io.Completion) {
	index := 0
	for index < len(state.Timeouts) && state.Timeouts[index].Ready_At <= completion.Ready_At {
		index++
	}
	state.Timeouts = append(state.Timeouts, nil)
	copy(state.Timeouts[index+1:], state.Timeouts[index:])
	state.Timeouts[index] = completion
}

// Arms listener for readability; on readiness it accepts one connection and fires
// callback with the new descriptor.
func operating_system_accept(
	state *operating_system, completion *io.Completion,
	callback io.Socket_Callback, listener io.File,
) {
	operating_system_poll_ensure(state)
	descriptor := int(listener)
	completion.Callback = func() { callback(completion, 0, io.Cancelled) }
	state.Read_Waiters[descriptor] = &socket_operation{
		Completion: completion,
		Perform: func() (done bool) {
			accepted, again, err := socket_accept(descriptor)
			if again {
				return false
			}
			callback(completion, io.File(accepted), err)
			return true
		},
	}
	poll_file_arm(state.Poll, descriptor, false)
}

// Begins a connection to host_address:port and arms the new socket for writability;
// on readiness it reports the connect result and the connected descriptor.
func operating_system_connect(
	state *operating_system, completion *io.Completion,
	callback io.Socket_Callback, host_address string, port int,
) {
	operating_system_poll_ensure(state)
	descriptor, start_err := socket_connect_start(host_address, port)
	if start_err != nil {
		completion.Callback = func() { callback(completion, io.File(-1), start_err) }
		state.Completed = append(state.Completed, completion)
		return
	}
	completion.Callback = func() { callback(completion, 0, io.Cancelled) }
	state.Write_Waiters[descriptor] = &socket_operation{
		Completion: completion,
		Perform: func() (done bool) {
			callback(completion, io.File(descriptor), socket_connect_error(descriptor))
			return true
		},
	}
	poll_file_arm(state.Poll, descriptor, true)
}

// Arms socket for readability; on readiness it reads once into buffer and reports
// the byte count.
func operating_system_receive(
	state *operating_system, completion *io.Completion,
	callback io.Callback, socket io.File, buffer []byte,
) {
	if operating_system_tls_receive(state, completion, callback, socket, buffer) {
		return
	}
	operating_system_poll_ensure(state)
	descriptor := int(socket)
	completion.Callback = func() { callback(completion, 0, io.Cancelled) }
	state.Read_Waiters[descriptor] = &socket_operation{
		Completion: completion,
		Perform: func() (done bool) {
			count, again, err := socket_receive(descriptor, buffer)
			if again {
				return false
			}
			callback(completion, count, err)
			return true
		},
	}
	poll_file_arm(state.Poll, descriptor, false)
}

// Arms socket for writability; on readiness it writes buffer once and reports the
// byte count.
func operating_system_send(
	state *operating_system, completion *io.Completion,
	callback io.Callback, socket io.File, buffer []byte,
) {
	if operating_system_tls_send(state, completion, callback, socket, buffer) {
		return
	}
	operating_system_poll_ensure(state)
	descriptor := int(socket)
	completion.Callback = func() { callback(completion, 0, io.Cancelled) }
	state.Write_Waiters[descriptor] = &socket_operation{
		Completion: completion,
		Perform: func() (done bool) {
			count, again, err := socket_send(descriptor, buffer)
			if again {
				return false
			}
			callback(completion, count, err)
			return true
		},
	}
	poll_file_arm(state.Poll, descriptor, true)
}

// Closes file and queues its completion to fire on the next drain.
func operating_system_close(
	state *operating_system, completion *io.Completion,
	callback io.Timeout_Callback, file io.File,
) {
	if operating_system_tls_close(state, completion, callback, file) {
		return
	}
	completion.Cancelled = false
	err := socket_close(int(file))
	completion.Callback = func() {
		if completion.Cancelled {
			callback(completion, io.Cancelled)
			return
		}
		callback(completion, err)
	}
	state.Completed = append(state.Completed, completion)
}

// Cancels an in-flight operation. A socket op armed on the poll is dropped, disarmed,
// and queued so its cancel callback fires; a pending timeout is moved to the completed
// queue, where its Cancelled-marked callback delivers the error; a file op already
// queued fires cancelled the same way. An already-delivered completion is a no-op.
func operating_system_cancel(state *operating_system, completion *io.Completion) {
	completion.Cancelled = true
	if operating_system_cancel_socket(state, completion) {
		return
	}
	operating_system_cancel_timeout(state, completion)
}

// Drops and queues completion's socket waiter, from either direction, reporting
// whether one was armed.
func operating_system_cancel_socket(
	state *operating_system, completion *io.Completion,
) (found bool) {
	if operating_system_cancel_waiter(state, state.Read_Waiters, completion, false) {
		return true
	}
	return operating_system_cancel_waiter(state, state.Write_Waiters, completion, true)
}

// Removes completion's waiter from waiters, disarms the poll for its descriptor and
// direction, and queues the completion so its cancel callback fires on the next drain.
func operating_system_cancel_waiter(
	state *operating_system, waiters map[int]*socket_operation,
	completion *io.Completion, writable bool,
) (found bool) {
	for descriptor, operation := range waiters {
		if operation.Completion != completion {
			continue
		}
		delete(waiters, descriptor)
		poll_file_disarm(state.Poll, descriptor, writable)
		state.Completed = append(state.Completed, completion)
		return true
	}
	return false
}

// Moves a pending timeout matching completion into the completed queue so its
// Cancelled-marked callback fires promptly, reporting whether it was present.
func operating_system_cancel_timeout(
	state *operating_system, completion *io.Completion,
) (found bool) {
	for index := 0; index < len(state.Timeouts); index++ {
		if state.Timeouts[index] != completion {
			continue
		}
		state.Timeouts = append(state.Timeouts[:index], state.Timeouts[index+1:]...)
		state.Completed = append(state.Completed, completion)
		return true
	}
	return false
}

// Submits work to the worker pool, delivering callback on the loop thread once it
// finishes. A cancel on a compute is a no-op: the work is already queued off-thread.
func operating_system_compute_submit(
	state *operating_system, completion *io.Completion,
	callback io.Compute_Callback, work func(),
) {
	completion.Cancelled = false
	operating_system_compute_ensure(state)
	state.Jobs <- &compute_job{Completion: completion, Callback: callback, Work: work}
}

// Starts the worker pool and the wake pipe on the first Compute.
func operating_system_compute_ensure(state *operating_system) {
	if state.Compute_Active {
		return
	}
	operating_system_wake_ensure(state)
	state.Jobs = make(chan *compute_job, compute_queue_depth)
	state.Compute_Active = true
	workers := compute_worker_count()
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
func operating_system_compute_worker(state *operating_system) {
	for job := range state.Jobs {
		job.Work()
		state.Results_Mutex.Lock()
		state.Results = append(state.Results, job)
		state.Results_Mutex.Unlock()
		wake_poke(state.Wake_Write)
	}
}

// Creates the wake pipe and arms its read end on the poll so a worker or TLS goroutine
// poking it returns a blocked poll.
func operating_system_wake_ensure(state *operating_system) {
	if state.Wake_Active {
		return
	}
	read, write, err := wake_create()
	if err != nil {
		return
	}
	operating_system_poll_ensure(state)
	state.Wake_Read = read
	state.Wake_Write = write
	state.Wake_Active = true
	poll_file_arm(state.Poll, read, false)
}

// Drains finished compute jobs and posted TLS completions onto the completed queue, on
// the loop thread; a no-op until the wake pipe exists.
func operating_system_compute(state *operating_system) {
	if !state.Wake_Active {
		return
	}
	wake_drain(state.Wake_Read)
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
func operating_system_compute_finish(state *operating_system, job *compute_job) {
	completion := job.Completion
	callback := job.Callback
	completion.Callback = func() { callback(completion) }
	state.Completed = append(state.Completed, completion)
}

// Registers a watcher for signal and starts OS notification for it.
func operating_system_watch_signal(
	state *operating_system, completion *io.Completion,
	callback io.Signal_Callback, kind io.Signal,
) {
	operating_system_signal_ensure(state)
	system := signal_to_operating_system(kind)
	state.Signal_Waiters = append(state.Signal_Waiters, signal_waiter{
		System: system, Kind: kind, Completion: completion, Callback: callback,
	})
	signal.Notify(state.Signals, system)
}

// Creates the buffered signal channel on the first watch.
func operating_system_signal_ensure(state *operating_system) {
	if state.Signals != nil {
		return
	}
	state.Signals = make(chan os.Signal, signal_queue_depth)
}

// Maps a backend-independent signal to its OS signal.
func signal_to_operating_system(kind io.Signal) (system os.Signal) {
	if kind == io.Signal_Interrupt {
		return syscall.SIGINT
	}
	return syscall.SIGTERM
}

// Drains delivered signals without blocking, firing matching watchers.
func operating_system_signals(state *operating_system) {
	for index := 0; index < signal_queue_depth; index++ {
		select {
		case received := <-state.Signals:
			operating_system_signal_deliver(state, received)
		default:
			return
		}
	}
}

// Fires every watcher matching received one-shot, keeping the rest for later deliveries.
func operating_system_signal_deliver(state *operating_system, received os.Signal) {
	kept := state.Signal_Waiters[:0]
	for index := 0; index < len(state.Signal_Waiters); index++ {
		waiter := state.Signal_Waiters[index]
		if waiter.System != received {
			kept = append(kept, waiter)
			continue
		}
		delivered := waiter
		delivered.Completion.Callback = func() {
			delivered.Callback(delivered.Completion, delivered.Kind)
		}
		state.Completed = append(state.Completed, delivered.Completion)
	}
	state.Signal_Waiters = kept
}

// Begins an outbound TLS connection: allocates a synthetic descriptor, records the
// connection, and spawns its goroutine to dial and service requests.
func operating_system_connect_secure(state *operating_system, target tls_target) {
	operating_system_wake_ensure(state)
	if !state.Wake_Active {
		completion := target.Completion
		callback := target.Callback
		completion.Callback = func() {
			callback(completion, io.File(-1), errors.New("io: wake pipe unavailable"))
		}
		state.Completed = append(state.Completed, completion)
		return
	}
	file := operating_system_next_tls(state)
	connection := &tls_connection{Requests: make(chan tls_request, tls_request_depth)}
	state.TLS[file] = connection
	go tls_serve(state, target, file, connection)
}

// Hands out the next synthetic TLS descriptor and ensures the TLS map exists.
func operating_system_next_tls(state *operating_system) (file io.File) {
	if state.TLS == nil {
		state.TLS = make(map[io.File]*tls_connection)
	}
	if state.Next_TLS < tls_file_base {
		state.Next_TLS = tls_file_base
	}
	state.Next_TLS++
	return state.Next_TLS
}

// Posts a completion finished off the loop thread: records its callback under the mutex
// and wakes the loop to run it on the next drain.
func operating_system_post(
	state *operating_system, completion *io.Completion, callback func(),
) {
	completion.Callback = callback
	state.Results_Mutex.Lock()
	state.Posted = append(state.Posted, completion)
	state.Results_Mutex.Unlock()
	wake_poke(state.Wake_Write)
}

// Dials the TLS target on its own goroutine, posts the connect result, then services
// send/receive/close requests until the connection closes.
func tls_serve(
	state *operating_system, target tls_target, file io.File, connection *tls_connection,
) {
	secure, err := tls_dial(target)
	if err != nil {
		operating_system_post(state, target.Completion, func() {
			delete(state.TLS, file)
			target.Callback(target.Completion, io.File(-1), err)
		})
		return
	}
	operating_system_post(state, target.Completion, func() {
		target.Callback(target.Completion, file, nil)
	})
	for request := range connection.Requests {
		if tls_handle(state, secure, request) {
			return
		}
	}
}

// Dials host:port and completes the client TLS handshake, verifying against Server_Name
// unless the target is insecure.
func tls_dial(target tls_target) (secure *tls.Conn, err error) {
	address := net.JoinHostPort(target.Host, strconv.Itoa(target.Port))
	raw, dial_err := net.Dial("tcp", address)
	if dial_err != nil {
		return nil, dial_err
	}
	secure = tls.Client(raw, &tls.Config{
		ServerName:         target.Server_Name,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: target.Insecure,
	})
	handshake_err := secure.Handshake()
	if handshake_err != nil {
		raw.Close()
		return nil, handshake_err
	}
	return secure, nil
}

// Arms listener for readability; on readiness it accepts a raw connection and hands it
// to a TLS goroutine that completes the server handshake.
func operating_system_accept_secure(
	state *operating_system, completion *io.Completion,
	callback io.Socket_Callback, listener io.File, certificate func() (value any),
) {
	operating_system_wake_ensure(state)
	operating_system_poll_ensure(state)
	descriptor := int(listener)
	completion.Callback = func() { callback(completion, 0, io.Cancelled) }
	state.Read_Waiters[descriptor] = &socket_operation{
		Completion: completion,
		Perform: func() (done bool) {
			accepted, again, err := socket_accept(descriptor)
			if again {
				return false
			}
			operating_system_secure_accepted(
				state, completion, callback, accepted, err, certificate)
			return true
		},
	}
	poll_file_arm(state.Poll, descriptor, false)
}

// Handles a completed raw accept for a secure listener: on error reports it; otherwise
// registers a TLS connection and spawns its server goroutine.
func operating_system_secure_accepted(
	state *operating_system, completion *io.Completion, callback io.Socket_Callback,
	accepted int, err error, certificate func() (value any),
) {
	if err != nil {
		callback(completion, io.File(-1), err)
		return
	}
	file, connection := operating_system_tls_register(state)
	go tls_accept_serve(state, completion, callback, accepted, file, connection, certificate)
}

// Allocates a synthetic descriptor and records a fresh TLS connection.
func operating_system_tls_register(
	state *operating_system,
) (file io.File, connection *tls_connection) {
	file = operating_system_next_tls(state)
	connection = &tls_connection{Requests: make(chan tls_request, tls_request_depth)}
	state.TLS[file] = connection
	return file, connection
}

// Completes the server TLS handshake on its own goroutine, posts the connect result,
// then services requests until the connection closes.
func tls_accept_serve(
	state *operating_system, completion *io.Completion, callback io.Socket_Callback,
	accepted int, file io.File, connection *tls_connection, certificate func() (value any),
) {
	secure, err := tls_accept(accepted, certificate)
	if err != nil {
		operating_system_post(state, completion, func() {
			delete(state.TLS, file)
			callback(completion, io.File(-1), err)
		})
		return
	}
	operating_system_post(state, completion, func() {
		callback(completion, file, nil)
	})
	for request := range connection.Requests {
		if tls_handle(state, secure, request) {
			return
		}
	}
}

// Wraps an accepted raw descriptor in a server tls.Conn and completes the handshake.
func tls_accept(
	accepted int, certificate func() (value any),
) (secure *tls.Conn, err error) {
	raw_file := os.NewFile(uintptr(accepted), "tcp")
	raw, connection_err := net.FileConn(raw_file)
	raw_file.Close()
	if connection_err != nil {
		return nil, connection_err
	}
	secure = tls.Server(raw, &tls.Config{
		MinVersion:     tls.VersionTLS12,
		GetCertificate: tls_server_certificate(certificate),
	})
	handshake_err := secure.Handshake()
	if handshake_err != nil {
		raw.Close()
		return nil, handshake_err
	}
	return secure, nil
}

// Adapts the opaque certificate provider to a tls.Config.GetCertificate callback,
// asserting it yields a *tls.Certificate.
func tls_server_certificate(
	certificate func() (value any),
) (get func(hello *tls.ClientHelloInfo) (chosen *tls.Certificate, err error)) {
	return func(hello *tls.ClientHelloInfo) (chosen *tls.Certificate, err error) {
		value := certificate()
		typed, ok := value.(*tls.Certificate)
		if !ok {
			return nil, errors.New("io: no server certificate")
		}
		return typed, nil
	}
}

// Handles one TLS request on the connection goroutine, posting the result; returns true
// when the connection should close.
func tls_handle(
	state *operating_system, secure *tls.Conn, request tls_request,
) (closed bool) {
	if request.Kind == tls_close {
		close_err := secure.Close()
		operating_system_post(state, request.Completion, func() {
			request.Close_Callback(request.Completion, close_err)
		})
		return true
	}
	if request.Kind == tls_receive {
		count, err := secure.Read(request.Buffer)
		operating_system_post(state, request.Completion, func() {
			request.Byte_Callback(request.Completion, count, err)
		})
		return false
	}
	count, err := secure.Write(request.Buffer)
	operating_system_post(state, request.Completion, func() {
		request.Byte_Callback(request.Completion, count, err)
	})
	return false
}

// Routes a receive to its TLS connection goroutine when socket is a TLS descriptor,
// reporting whether it did.
func operating_system_tls_receive(
	state *operating_system, completion *io.Completion,
	callback io.Callback, socket io.File, buffer []byte,
) (routed bool) {
	connection := state.TLS[socket]
	if connection == nil {
		return false
	}
	connection.Requests <- tls_request{
		Kind: tls_receive, Completion: completion, Buffer: buffer, Byte_Callback: callback,
	}
	return true
}

// Routes a send to its TLS connection goroutine when socket is a TLS descriptor,
// reporting whether it did.
func operating_system_tls_send(
	state *operating_system, completion *io.Completion,
	callback io.Callback, socket io.File, buffer []byte,
) (routed bool) {
	connection := state.TLS[socket]
	if connection == nil {
		return false
	}
	connection.Requests <- tls_request{
		Kind: tls_send, Completion: completion, Buffer: buffer, Byte_Callback: callback,
	}
	return true
}

// Routes a close to its TLS connection goroutine when file is a TLS descriptor, dropping
// the routing entry, and reporting whether it did.
func operating_system_tls_close(
	state *operating_system, completion *io.Completion,
	callback io.Timeout_Callback, file io.File,
) (routed bool) {
	connection := state.TLS[file]
	if connection == nil {
		return false
	}
	delete(state.TLS, file)
	connection.Requests <- tls_request{
		Kind: tls_close, Completion: completion, Close_Callback: callback,
	}
	return true
}
