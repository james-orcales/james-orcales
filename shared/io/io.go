// Package io is the dependency-injected completion surface used by the repository.
//
// FAITHFUL PORT: its socket lifecycle and completion ordering follow TigerBeetle's
// third-party/tigerbeetle/src/io/darwin.zig and io/linux.zig. DO NOT DIVERGE. There is no
// generic Cancel: owners use Shutdown, join every submitted operation, and then submit Close,
// following third-party/tigerbeetle/src/message_bus.zig:1057-1160. Repository-specific effects
// are marked as extensions and must retire through the same completed queue.
package io

import (
	"errors"
	"io"
	"net/netip"
	"slices"
	"strconv"
	"strings"

	invariant "local/james-orcales/g/shared/invariant/default"
	"local/james-orcales/g/shared/prng"
	"local/james-orcales/g/shared/time"
)

// File identifies an open file or socket. The simulated backend maps it to tracked
// in-memory state; a real backend maps it to a descriptor or handle.
type File int32

// DIRECTORY_CURRENT selects the process current directory for Open_At. The default backend maps
// this portable value to the platform AT_FDCWD constant used by TigerBeetle IO.openat.
const DIRECTORY_CURRENT File = -1

// Open_Access selects the read/write access mode for TigerBeetle's asynchronous Open_At.
type Open_Access int

// OPEN_READ_ONLY opens a file for reads.
const OPEN_READ_ONLY Open_Access = 0

// OPEN_WRITE_ONLY opens a file for writes.
const OPEN_WRITE_ONLY Open_Access = 1

// OPEN_READ_WRITE opens a file for reads and writes.
const OPEN_READ_WRITE Open_Access = 2

// Open_At_Options is the Go representation of the posix.O fields used by TigerBeetle openat.
type Open_At_Options struct {
	// Access selects read-only, write-only, or read-write access.
	Access Open_Access
	// Create creates the path when absent.
	Create bool
	// Truncate clears an existing file before the callback receives it.
	Truncate bool
	// Mode is the permission mode used only when Create is true.
	Mode uint32
}

// Event identifies TigerBeetle's cross-thread event primitive: EVFILT_USER on Darwin and eventfd
// on Linux. It is backend-owned until Close_Event.
type Event uintptr

// Address_Family is the address family used to create and bind a socket.
type Address_Family int

// FAMILY_IPV4 selects an IPv4 socket.
const FAMILY_IPV4 Address_Family = 0

// FAMILY_IPV6 selects an IPv6 socket.
const FAMILY_IPV6 Address_Family = 1

// Address is an IP address and port with an explicit family, corresponding to TigerBeetle's
// stdx.SocketAddress. IP stores IPv4 bytes in its first four positions and IPv6 bytes in all 16.
type Address struct {
	// Family selects how IP is interpreted.
	Family Address_Family
	// IP holds the network address bytes.
	IP [16]byte
	// Port is the host-order TCP or UDP port.
	Port uint16
}

// Address_I_Pv4 returns an IPv4 address from its four octets and host-order port.
func Address_I_Pv4(ip [4]byte, port uint16) (address Address) {
	address.Family = FAMILY_IPV4
	copy(address.IP[:4], ip[:])
	address.Port = port
	return address
}

// Address_I_Pv6 returns an IPv6 address from its 16 octets and host-order port.
func Address_I_Pv6(ip [16]byte, port uint16) (address Address) {
	return Address{Family: FAMILY_IPV6, IP: ip, Port: port}
}

// Address_Parse parses an IP literal without DNS and returns an explicit-family address.
func Address_Parse(host string, port int) (address Address, err error) {
	if port < 0 {
		return Address{}, errors.New("io: port is outside uint16")
	}
	if port > 65535 {
		return Address{}, errors.New("io: port is outside uint16")
	}
	parsed, parse_err := netip.ParseAddr(host)
	if parse_err != nil {
		return Address{}, parse_err
	}
	if parsed.Is4() {
		return Address_I_Pv4(parsed.As4(), uint16(port)), nil
	}
	return Address_I_Pv6(parsed.As16(), uint16(port)), nil
}

// TCP_Keepalive is the optional keepalive tuple applied to a TCP socket.
type TCP_Keepalive struct {
	// Idle_Seconds is the idle period before probes begin.
	Idle_Seconds int
	// Interval_Seconds is the interval between probes.
	Interval_Seconds int
	// Count is the number of failed probes before the connection is abandoned.
	Count int
}

// TCP_Options are the options applied while creating a TCP socket, matching
// third-party/tigerbeetle/src/io/common.zig:14-28.
type TCP_Options struct {
	// Receive_Buffer is the requested receive-buffer size; zero leaves the system default.
	Receive_Buffer int
	// Send_Buffer is the requested send-buffer size; zero leaves the system default.
	Send_Buffer int
	// Keepalive is nil when TCP keepalive is disabled.
	Keepalive *TCP_Keepalive
	// User_Timeout_Milliseconds is the maximum unacknowledged duration on Linux.
	User_Timeout_Milliseconds int
	// No_Delay enables TCP_NODELAY where TigerBeetle enables it.
	No_Delay bool
}

// Listen_Options are the options applied after binding a caller-owned socket.
type Listen_Options struct {
	// Backlog is the requested completed-connection queue size.
	Backlog uint32
}

// Shutdown_How selects which connected-socket direction shutdown disables.
type Shutdown_How int

// SHUTDOWN_RECEIVE disables further receives.
const SHUTDOWN_RECEIVE Shutdown_How = 0

// SHUTDOWN_SEND disables further sends.
const SHUTDOWN_SEND Shutdown_How = 1

// SHUTDOWN_BOTH disables receives and sends.
const SHUTDOWN_BOTH Shutdown_How = 2

// Next_Tick_Source groups deferred callbacks so Reset_Next_Tick can remove a whole source.
type Next_Tick_Source int

// NEXT_TICK_LSM is TigerBeetle's storage-origin next-tick source.
const NEXT_TICK_LSM Next_Tick_Source = 0

// NEXT_TICK_VSR is TigerBeetle's replication-origin next-tick source.
const NEXT_TICK_VSR Next_Tick_Source = 1

// Next_Tick_Callback receives a deferred next-tick completion.
type Next_Tick_Callback func(completion *Completion)

// Callback receives the result of a read or write: the byte count, or an error.
type Callback func(completion *Completion, count int, err error)

// Timeout_Callback receives a status-only result from a timeout, connect, close, or
// another operation that returns no value beyond its error.
type Timeout_Callback func(completion *Completion, err error)

// File_Callback receives a descriptor returned by an asynchronous filesystem operation.
type File_Callback func(completion *Completion, file File, err error)

// Socket_Callback receives a newly accepted server connection or an error.
type Socket_Callback func(completion *Completion, socket File, err error)

// Signal identifies an operating-system signal in backend-independent form, so the
// deterministic and OS backends agree on a value without the pure tier importing syscall.
type Signal int

// SIGNAL_TERMINATE is the graceful-termination request (SIGTERM on the OS backend).
const SIGNAL_TERMINATE Signal = 0

// SIGNAL_INTERRUPT is the interactive interrupt (SIGINT on the OS backend).
const SIGNAL_INTERRUPT Signal = 1

// Signal_Callback receives a delivered signal on the loop thread, or Deadline_Exceeded when the
// finite watch retires before a signal arrives.
type Signal_Callback func(completion *Completion, signal Signal, err error)

// Compute_Callback fires on the loop thread once offloaded work has finished.
type Compute_Callback func(completion *Completion)

// Process_Request describes a subprocess to run: the executable, its arguments and
// environment, the working directory, and the bytes fed to its standard input.
type Process_Request struct {
	// Path is the executable to run.
	Path string
	// Arguments are the process arguments, excluding the program name.
	Arguments []string
	// Environment is the process environment; nil inherits the parent's.
	Environment []string
	// Working_Directory is the process's directory; empty uses the current one.
	Working_Directory string
	// Input is the bytes written to the process's standard input.
	Input []byte
	// Stdout, when non-nil, streams the process's standard output to the writer as it
	// runs instead of capturing it into Result.Output — the affordance a long build
	// needs so its progress reaches the user live. A nil sink keeps the captured-buffer
	// default. The simulated backend produces no output and ignores it.
	Stdout io.Writer
	// Stderr is the standard-error counterpart, same live-or-capture rule.
	Stderr io.Writer
}

// Process_Usage is the resource accounting a finished process reports.
type Process_Usage struct {
	// Wall is the elapsed wall-clock time the process ran.
	Wall time.Duration
	// CPU_User is the user-mode CPU time consumed.
	CPU_User time.Duration
	// CPU_System is the kernel-mode CPU time consumed.
	CPU_System time.Duration
	// RSS_Bytes_Max is the peak resident set size in bytes.
	RSS_Bytes_Max int64
}

// Process_Result is a finished process's outcome: its exit code, captured output, and
// resource usage.
type Process_Result struct {
	// Exit is the process exit code; zero on success.
	Exit int
	// Output is the captured standard output.
	Output []byte
	// Error_Output is the captured standard error.
	Error_Output []byte
	// Usage is the process's resource accounting.
	Usage Process_Usage
}

// Process_Callback receives a finished process's result, or an error when the process
// could not be started at all.
type Process_Callback func(completion *Completion, result Process_Result, err error)

// Directory_Entry is one child of a directory: its name and whether it is itself a
// directory, the two facts a walk needs to recurse into subdirectories and sync files.
type Directory_Entry struct {
	// Name is the child's name within its directory, not a full path.
	Name string
	// Is_Directory reports whether the child is a directory, so a walk knows to recurse.
	Is_Directory bool
}

// File_Status is what a stat reports: whether the path exists, and if so whether it is a
// directory and its byte length — metadata a mirror consults before it reads or writes.
type File_Status struct {
	// Exists reports whether the path is present; an absent path is not an error.
	Exists bool
	// Is_Directory reports whether an existing path is a directory rather than a file.
	Is_Directory bool
	// Size is the file's length in bytes, zero for a directory or an absent path.
	Size int64
}

// Connection_Refused is the portable error returned when a remote endpoint rejects a connect.
var Connection_Refused = errors.New("io: connection refused")

// Broken_Pipe is the portable send result for a socket whose send half is shut down.
var Broken_Pipe = errors.New("io: broken pipe")

// Socket_Not_Connected is returned when Shutdown is submitted before a socket is connected.
var Socket_Not_Connected = errors.New("io: socket not connected")

// Canceled is a raw kernel operation result, matching TigerBeetle's error.Canceled variants.
// It is not an API for canceling an operation: shared/io deliberately has no generic Cancel.
var Canceled = errors.New("io: operation canceled")

// Deadline_Exceeded is returned after a finite operation retires without its external event.
var Deadline_Exceeded = errors.New("io: deadline exceeded")

// FOREVER is the Run_Until timeout that never expires: the loop pumps until done reports
// true, however long that takes — for a caller (a server) that runs until an event, not a
// clock.
const FOREVER time.Duration = -1

// IMMEDIATE is the Run_Until timeout that expires at once: done is evaluated a single time
// and the loop is not driven — a non-blocking poll of the predicate.
const IMMEDIATE time.Duration = 0

// The number of virtual grains a simulated operation may take to complete, drawn from
// the seed so the completion order varies per run while staying reproducible.
const sim_latency_grains = 8

// One in this many simulated spawns exits non-zero, so a seed sweep exercises both the
// success and the failure path without a scripted outcome.
const sim_spawn_fail_grains = 4

// Completion is the caller-owned storage for one in-flight operation —
// TigerBeetle's IO.Completion. The caller allocates it, so the loop never does, and
// must keep it alive until the callback fires.
type Completion struct {
	// Callback is the closure the backend runs on completion; it closes over the
	// typed callback and the computed result.
	Callback func()
	// Ready_At is the virtual Moment this operation completes, mirroring
	// TigerBeetle's Storage.Read.ready_at.
	Ready_At time.Moment
	// Next_Tick_Source identifies next-tick completions for Reset_Next_Tick. Other operations
	// leave it untouched; it is backend-owned metadata.
	Next_Tick_Source Next_Tick_Source
	// Next_Tick reports whether this armed completion is a next-tick operation.
	Next_Tick bool
	// State is the completion's position in its lifecycle machine, mutated only through
	// Completion_Transition. It is backend-owned: applications never read or write it —
	// expose your own state, not the completion's.
	State Completion_State
	// Self is the completion's own address, stamped on its first submit and never cleared.
	// The loop tracks an in-flight op by pointer, so a by-value copy carries this original
	// address; submitting the copy trips the backend's assert instead of silently
	// splitting the loop's view from the caller's. Only the backends touch it.
	Self *Completion
	// Kernel_Identifier is the generation token stored in kqueue udata or io_uring user_data.
	// Backends own it; Event_Trigger reads it only after Event_Listen has armed the completion.
	Kernel_Identifier uint64
}

// Completion_State is one position in a completion's lifecycle machine. The machine has
// exactly two legal edges: idle to armed on submit and armed to idle before delivery.
type Completion_State int

// COMPLETION_IDLE is the zero value: never submitted, or delivered and reusable. A
// delivery resets to idle before the callback runs, so a callback may resubmit its own
// completion — the repeating-timer pattern.
const COMPLETION_IDLE Completion_State = 0

// COMPLETION_ARMED marks an in-flight operation: submitted and owned until delivery.
const COMPLETION_ARMED Completion_State = 1

// Input for Completion_Transition_Legal.
type Completion_Transition_Legal_Input struct {
	// From is the state the edge leaves.
	From Completion_State
	// To is the state the edge enters.
	To Completion_State
}

// Completion_Transition_Legal is the machine's transition table: it reports whether the
// edge from one state to another exists. A function rather than a table value because Go
// has no const maps and a package var is banned; the flat one-clause-per-edge shape is
// the point — the whole graph, readable in one place.
func Completion_Transition_Legal(input *Completion_Transition_Legal_Input) (legal bool) {
	if input.From == COMPLETION_IDLE {
		return input.To == COMPLETION_ARMED
	}
	if input.From == COMPLETION_ARMED {
		return input.To == COMPLETION_IDLE
	}
	return false
}

// Input for Completion_Transition.
type Completion_Transition_Input struct {
	// Completion is the completion being moved along an edge.
	Completion *Completion
	// From is the state the caller believes the completion is in.
	From Completion_State
	// To is the destination state.
	To Completion_State
}

// Completion_Transition moves a completion along one edge of its lifecycle machine. Its
// two Always guards fail loudly on a caller whose belief about the current state is
// stale — a reused or double-armed completion — and on an edge the machine does not
// have, so a lifecycle bug dies at the mutation instead of corrupting a queue. Every
// transition then records its edge on the io.completion.transition grid: this package's
// own suite registers the grid through its TestMain, so an edge the sim suite never
// witnesses fails the run — the graph is enforced by the guards and witnessed by the
// sweep. Backend code only; applications never transition a completion.
func Completion_Transition(input *Completion_Transition_Input) {
	invariant.Always(input.Completion.State == input.From,
		"A completion transitions from the state its caller expects.")
	legal := Completion_Transition_Legal(&Completion_Transition_Legal_Input{
		From: input.From, To: input.To,
	})
	invariant.Always(legal, "A completion transitions along an edge its machine has.")
	input.Completion.State = input.To
	// The two axes identify the two legal inverse edges and carve away both illegal cells.
	invariant.Dot_Product("io.completion.transition",
		invariant.Sometimes(input.From == COMPLETION_IDLE, "the edge leaves idle"),
		invariant.Sometimes(input.To == COMPLETION_IDLE, "the edge enters idle"),
		invariant.Impossible(
			invariant.Event_True("the edge leaves idle"),
			invariant.Event_True("the edge enters idle")),
		invariant.Impossible(
			invariant.Event_False("the edge leaves idle"),
			invariant.Event_False("the edge enters idle")),
	)
}

// IO is the injected async IO submit surface — TigerBeetle's `IO`. Code submits
// operations with a Completion and callback and reacts to completions; it never drives
// the loop — that is the Driver's job — so a holder can submit IO but not advance time.
//
// This is an I/O seam, not a kitchen sink of syscalls. Every member is a data transfer with an
// external endpoint — a file, a socket, a pipe — or the loop's own control plane for scheduling
// those transfers (timers, signal watches, and spawn). Not all syscalls are I/O
// operations: getrandom, getpid, mmap, and nanosleep are syscalls but transfer no data with an
// endpoint, so they do not belong here. A dependency that needs one of those (secure_transport's
// entropy, for instance) declares it itself and has it injected, rather than widening this surface.
//
// CRITICAL: ONLY PACKAGE MAIN OR A TEST MAY DRIVE, RUN, OR TICK THE EVENT LOOP. An IO
// holder that wants to wait exposes doneness as state and lets the root pump; it never
// receives a pump. See shared/io/README.md, THE CRITICAL GUARANTEE.
type IO struct {
	Platform_IO
	// Read reads len(buffer) bytes from file at offset; callback fires with the byte
	// count or error once the operation completes (TigerBeetle IO.read).
	Read func(
		completion *Completion, callback Callback, file File, buffer []byte, offset int64,
	)
	// Write writes buffer to file at offset (TigerBeetle IO.write).
	Write func(
		completion *Completion, callback Callback, file File, buffer []byte, offset int64,
	)
	// Fsync synchronizes file through TigerBeetle's asynchronous IO operation.
	Fsync func(completion *Completion, callback Timeout_Callback, file File)
	// Open_At asynchronously opens file_path relative to directory and forces close-on-exec.
	Open_At func(
		completion *Completion, callback File_Callback, directory File, file_path string,
		options Open_At_Options,
	)
	// Timeout fires callback after the duration on the clock, off the same queue the
	// IO completions use. Duration must be positive; use Next_Tick to yield.
	Timeout func(completion *Completion, callback Timeout_Callback, duration time.Duration)
	// Next_Tick defers a callback without kernel IO, matching io/linux.zig:332-352 and
	// io/darwin.zig:757-781.
	Next_Tick func(
		completion *Completion, callback Next_Tick_Callback, source Next_Tick_Source,
	)
	// Reset_Next_Tick removes every queued next-tick completion for source without delivery.
	Reset_Next_Tick func(source Next_Tick_Source)
	// Open_Event creates TigerBeetle's platform Event primitive.
	Open_Event func() (event Event, err error)
	// Event_Listen arms completion for one Event notification.
	Event_Listen func(event Event, completion *Completion, callback Next_Tick_Callback)
	// Event_Trigger makes an armed Event completion ready. It is the only core operation
	// safe to call from another thread.
	Event_Trigger func(event Event, completion *Completion)
	// Close_Event releases an Event after its listener has drained.
	Close_Event func(event Event)
	// Listen binds a caller-owned socket and returns the resolved address, including the actual
	// port chosen for port zero (third-party/tigerbeetle/src/io/common.zig:32-59).
	Listen func(
		socket File, address Address, options Listen_Options,
	) (resolved Address, err error)
	// Open opens the file at path for reading, returning its descriptor synchronously —
	// opening never blocks the loop, so it carries no Completion. A later Read of the
	// descriptor yields the file's bytes.
	Open func(path string) (file File, err error)
	// Create opens path for writing, truncating it, and returns its descriptor
	// synchronously; a later Write persists bytes to it.
	Create func(path string) (file File, err error)
	// Read_Directory lists path's immediate children synchronously — a readdir never blocks
	// the loop, so it carries no Completion; each entry names a child and whether it is
	// itself a directory.
	Read_Directory func(path string) (entries []Directory_Entry, err error)
	// Status reports whether path exists, whether it is a directory, and its byte size,
	// synchronously; an absent path is Exists false with a nil error, so a caller branches on
	// the status, not the error.
	Status func(path string) (status File_Status, err error)
	// Make_Directory creates path and any missing parents synchronously; an existing
	// directory is not an error, so a repeated mkdir converges.
	Make_Directory func(path string) (err error)
	// Open_Socket_TCP creates a non-blocking close-on-exec TCP socket with explicit options.
	Open_Socket_TCP func(
		family Address_Family, options TCP_Options,
	) (socket File, err error)
	// Open_Socket_UDP creates a non-blocking close-on-exec UDP socket.
	Open_Socket_UDP func(family Address_Family) (socket File, err error)
	// Accept yields one inbound connection on listener before deadline, or Deadline_Exceeded.
	// The finite deadline deliberately diverges from TigerBeetle's unbounded IO.accept; it
	// keeps every repository submission bounded. TLS termination remains a secure_transport
	// concern.
	Accept func(
		completion *Completion, callback Socket_Callback, listener File,
		deadline time.Duration,
	)
	// Connect borrows caller-owned socket until it connects before deadline; callback reports
	// only the outcome. The deadline wins ties. The backend never creates, transfers, or closes
	// the descriptor. This finite lifetime is an explicit repository extension to TigerBeetle.
	Connect func(
		completion *Completion, callback Timeout_Callback,
		socket File, address Address, deadline time.Duration,
	)
	// Receive reads up to len(buffer) bytes from socket; callback reports the byte
	// count once data arrives (TigerBeetle IO.recv).
	Receive func(completion *Completion, callback Callback, socket File, buffer []byte)
	// Send writes buffer to socket; callback reports the byte count once the kernel
	// accepts it (TigerBeetle IO.send).
	Send func(completion *Completion, callback Callback, socket File, buffer []byte)
	// Send_Now makes TigerBeetle's best-effort synchronous datagram send. Sent is false only
	// when the datagram would block or the socket cannot send.
	Send_Now func(socket File, buffer []byte) (count int, sent bool)
	// Shutdown synchronously disables one or both connected-socket directions. It does not own
	// or close the socket (io/darwin.zig:973-975; io/linux.zig:1393-1395).
	Shutdown func(socket File, how Shutdown_How) (err error)
	// Close releases file's descriptor; callback fires once it is closed
	// (TigerBeetle IO.close).
	Close func(completion *Completion, callback Timeout_Callback, file File)
	// Close_Socket synchronously releases a socket during setup failure or final deinit. It is
	// TigerBeetle's close_socket, distinct from asynchronous Close.
	Close_Socket func(socket File)
	// Peer_Address returns the remote IP address of a connected socket, synchronously —
	// a getpeername has no completion. It is the source a control-plane connection is
	// gated on.
	Peer_Address func(file File) (address string, err error)
	// Watch_Signal fires callback on the loop thread when the process receives signal before
	// the finite deadline, or with Deadline_Exceeded. TigerBeetle has no signal-watch
	// counterpart; this repository extension is bounded rather than becoming a permanent
	// waiter.
	Watch_Signal func(
		completion *Completion, callback Signal_Callback, signal Signal,
		deadline time.Duration,
	)
	// ONLY FOR COMPUTE-INTENSIVE WORK THAT CAN BE HIGHLY PARALLELIZED — E.G. JSON
	// PARSING, HIGH-TRAFFIC REQUEST PROCESSING. IT IS NOT AN ESCAPE HATCH FOR BLOCKING
	// SYSCALLS OR IO; THOSE BELONG ON THE LOOP'S COMPLETION OPS.
	Compute func(completion *Completion, callback Compute_Callback, work func())
	// Spawn runs request off the loop thread until completion or deadline. Expiry kills the
	// subprocess group and returns Deadline_Exceeded with any partial result. The simulator
	// draws the exit code from the seed and returns no output, since scripted output is
	// disallowed.
	Spawn func(
		completion *Completion, callback Process_Callback, request Process_Request,
		deadline time.Duration,
	)
	// Self_Exec is a repository extension that replaces the process image. All descriptors are
	// close-on-exec, so a successful replacement closes listeners and the new image rebinds. A
	// nil environment preserves the backend's ambient environment; a non-nil slice completely
	// replaces it, including an explicitly empty slice that inherits nothing.
	Self_Exec func(path string, argv []string, environment []string) (err error)
}

// Driver advances the loop — the only capability that moves time and delivers
// completions.
//
// ===========================================================================
// ONLY PACKAGE MAIN OR A TEST MAY DRIVE, RUN, OR TICK THE EVENT LOOP.
// NOT A LIBRARY. NOT A HELPER. NOT AN INJECTED FUNC VALUE. NOT ONCE.
// A VIOLATION IS AN ARCHITECTURAL BUG EVEN IF EVERY TEST PASSES.
// ===========================================================================
//
// Only the code that constructs a Driver may hold or call it: a binary's package main
// in production, a test harness in simulation (the universe package is one). Handing
// any of these funcs — or a bare func value of the same shape, which the lint cannot
// see — to a library hands it the timeline: assembled into the universe package, that
// library would advance every other application's events from inside its own call
// stack. That it compiles and passes tests does not make it legal; the bug is invisible
// where it is written and fatal where it composes.
type Driver struct {
	// Run drains every ready completion without blocking, then advances the clock one
	// tick (TigerBeetle IO.run). ROOT ONLY: never handed to, or called from, a library.
	Run func() (err error)
	// Run_For drives the loop until the duration has elapsed on the clock, delivering
	// completions as they come due (TigerBeetle IO.run_for_ns). Here time is the GOAL: it
	// advances exactly duration, draining as it goes, regardless of what completes — reach
	// for it to let a span of time pass, not to wait for a particular op.
	// ROOT ONLY: never handed to, or called from, a library.
	Run_For func(duration time.Duration) (err error)
	// Run_Until drives the loop until done reports true — the run-until-complete pump that
	// lets straight-line code wait for its own op inline. Here completion is the GOAL and
	// time is the GUARD: it stops the instant done holds, and timeout only caps the wait so
	// a stalled op can't hang the caller. This opposite emphasis — completion-first with a
	// time bound, versus Run_For's time-first — is why the two stay separate ops.
	//
	// timeout < 0 (FOREVER) waits unbounded — a server pumping until a shutdown signal.
	// timeout == 0 (IMMEDIATE) evaluates done once and returns without driving — a poll.
	// timeout > 0 pumps until done or the clock passes now+timeout. completed reports which
	// won: done (true) or the timeout (false).
	//
	// Top-level and single-loop only: never call it from within a completion callback.
	// ROOT ONLY: never inject it — or a func value of its shape — into a library; a
	// library that authors done predicates and timeouts is driving the loop.
	Run_Until func(
		done func() (finished bool), timeout time.Duration,
	) (completed bool, err error)
	// Deinit releases the backend's kernel resources after every submitted operation is joined.
	Deinit func()
	// Introspect returns a point-in-time census of the loop's internal queues — the depths a
	// stall shows up in. Read-only, safe only on the loop thread, so like the rest of Driver
	// it is ROOT ONLY: the root may sample it (for an admin snapshot); a library may not. The
	// simulator reports the equivalent synthetic queues and lifecycle flags.
	Introspect func() (counts Loop_Counts)
}

// Loop_Counts is a census of an IO backend's internal queues at one instant — how many completions
// are ready to run, how many sockets await readability or writability, how many timers and signal
// watchers are pending, how much cross-thread work is posted back, and how many raw descriptors are
// held open. It is the loop-internals view of an admin state snapshot: a stall is usually visible
// here as a queue that will not drain (a backed-up accept, a write that never completes).
type Loop_Counts struct {
	// Completed is the number of completions whose callbacks are ready to run next drain.
	Completed int
	// Timeouts is the number of pending timer completions.
	Timeouts int
	// IO_Backlog is the number of Darwin operations waiting to enter kqueue.
	IO_Backlog int
	// IO_Inflight is the number of Darwin operations registered with kqueue.
	IO_Inflight int
	// IO_Queued is the number of Linux submissions not yet flushed to the kernel.
	IO_Queued int
	// IO_In_Kernel is the number of Linux submissions awaiting completion.
	IO_In_Kernel int
	// Signal_Waiters is the number of registered signal watchers.
	Signal_Waiters int
	// Posted is the number of completions finished off the loop thread awaiting the next drain.
	Posted int
	// Results is the number of finished compute jobs awaiting the next drain.
	Results int
	// Raw_Open is the number of raw descriptors the backend holds open.
	Raw_Open int
	// Wake_Active reports whether the wake pipe has been created and armed.
	Wake_Active bool
	// Compute_Active reports whether the compute worker pool has been started.
	Compute_Active bool
}

// The sim type is the deterministic, in-memory IO backend — TigerBeetle's simulated
// Storage.
//
// ===========================================================================
// DO NOT ADD A SCRIPTING API TO THIS TYPE. READ THIS BEFORE ADDING ANY FIELD.
// ===========================================================================
//
// "Scripting" means any exported field or function that lets a caller pre-load specific
// outcomes for the sim to replay: Connect_Targets / Accept_Queue / Open_Targets maps,
// per-socket Inbound / Outbound / Connect_Error / Send_Error / Receive_Error fields, a
// Receive_Script, a Send_Limit, or Sim_Raise / Sim_Connect_Target / Sim_Open_Target
// helpers. The production io this was ported from had all of them. We removed them on
// purpose. Do not bring them back, and do not add a new one when a test feels awkward.
//
// Why it is banned: a simulation must be a PURE FUNCTION OF ITS SEED. Every outcome —
// latency, which connect fails, the bytes a receive delivers, the grain a signal lands
// on — is drawn from Generator, seeded by New_Sim. Correctness is asserted by the
// invariant framework's Always/Sometimes across many seeds (a plain seed sweep, like
// shared/vsr/simulation), and any failure reproduces by replaying its seed. Scripting
// destroys all three properties: the run becomes a function of author-chosen data
// instead of the seed, so (1) the fuzzer can only ever find bugs the author already
// imagined, (2) a seed no longer reproduces the run, and (3) the invariants end up
// judging hand-picked inputs instead of the whole reachable space. The right way to
// influence a run is to change the SEED, never to inject data.
//
// This type is unexported and New_Sim(seed) is the only entry precisely so this stays
// impossible to add by accident: no caller ever holds a *sim to hang a field on. If you
// find yourself wanting to export it, or wanting to add a parameter to New_Sim that is
// not the seed, stop — that is the scripting API trying to come back. Keep it shut.

// Returned when a path resolves to nothing — the simulator's ENOENT.
var sim_file_absent = errors.New("io: no such file or directory")

// Returned when a path component that must be a directory is a file.
var sim_not_a_directory = errors.New("io: not a directory")

// Returned when a file operation names a directory.
var sim_is_a_directory = errors.New("io: is a directory")

// A sim_node is one entry in the simulator's in-memory filesystem: a directory with named
// children, or a file holding bytes. Generated from the seed at New_Sim and mutated by
// Create/Write/Make_Directory, so a later read reflects an earlier write.
type sim_node struct {
	// Directory reports whether this node is a directory rather than a file.
	Directory bool
	// Contents holds a file's bytes; nil for a directory.
	Contents []byte
	// Children maps a directory's entry names to their nodes; nil for a file.
	Children map[string]*sim_node
}

type sim_operation int

const sim_operation_completed sim_operation = 0
const sim_operation_timeout sim_operation = 1
const sim_operation_read_waiter sim_operation = 2
const sim_operation_write_waiter sim_operation = 3
const sim_operation_signal sim_operation = 4
const sim_operation_posted sim_operation = 5
const sim_operation_result sim_operation = 6
const sim_operation_next_tick sim_operation = 7
const sim_operation_event sim_operation = 8

// Sim event is the deterministic counterpart of TigerBeetle's EVFILT_USER/eventfd primitive.
type sim_event struct {
	// Completion is the currently armed listener, nil while detached.
	Completion *Completion
	// Triggered counts notifications accumulated before a listener attaches.
	Triggered int
}

// Sim_socket is the simulator's caller-owned socket state. Shutdown is directional and never
// releases ownership; Close and Close_Socket are the only operations that remove the entry.
type sim_socket struct {
	// Family is the address family selected at creation.
	Family Address_Family
	// Datagram distinguishes UDP from TCP.
	Datagram bool
	// Connected reports whether Connect or Accept established the socket.
	Connected bool
	// Listener reports whether Listen consumed this socket.
	Listener bool
	// Receive_Shutdown records SHUTDOWN_RECEIVE or SHUTDOWN_BOTH.
	Receive_Shutdown bool
	// Send_Shutdown records SHUTDOWN_SEND or SHUTDOWN_BOTH.
	Send_Shutdown bool
}

type sim struct {
	// Clock is the read-only time source; "now" is Clock.Now_Monotonic.
	Clock time.Clock
	// Tick advances the virtual clock one resolution — the tick returned beside Clock
	// by Virtual_Clock_To_Clock. The driver calls it after each drain; it lives on the
	// sim (driver side) so pure code, which holds only an IO, can never advance time.
	Tick func()
	// Generator is the seeded entropy every outcome is drawn from; it is the sim's only
	// source of variation, so the whole run reproduces from the seed.
	Generator prng.Generator
	// Queue holds in-flight completions ordered by Ready_At, earliest first.
	Queue []*Completion
	// Next_File is the synthetic descriptor counter; Listen, Accept, Open_Socket, Open,
	// and Create hand out the next value so every descriptor is distinct.
	Next_File File
	// Root is the in-memory filesystem the file ops read and mutate, fabricated from the
	// seed at New_Sim. Socket descriptors ignore it.
	Root *sim_node
	// Files binds an open file descriptor to its node, so Read/Write route to real tree
	// bytes; a descriptor absent from this map is a socket, whose bytes stay synthetic.
	Files map[File]*sim_node
	// Raw_Open tracks every synthetic descriptor until the caller submits Close.
	Raw_Open map[File]bool
	// Sockets holds lifecycle and directional-shutdown state for synthetic sockets.
	Sockets map[File]*sim_socket
	// Events holds backend Event state and is excluded from caller-owned Raw_Open accounting.
	Events map[Event]*sim_event
	// Next_Event supplies stable nonzero synthetic Event identifiers.
	Next_Event Event
	// Operations classifies each queued completion for full Driver.Introspect output.
	Operations map[*Completion]sim_operation
	// Operation_Files binds socket and file waiters to their descriptor so Close cancels them.
	Operation_Files map[*Completion]File
	// Drive_Active is set while a Run* is driving the loop, so a Run* called from within a
	// completion callback — which would re-enter the driver mid-drain — panics loudly.
	Drive_Active bool
}

// New_Sim returns a deterministic loop seeded by seed: the read-only clock and submit
// surface to inject into the program under test, and the driver to run it. The sim
// itself never escapes, and seed is the only input, so the run reproduces exactly and
// nothing can be scripted into it — correctness is asserted by invariants, not by
// hand-fed outcomes. The driver stays in the harness: the program under test receives
// loop and clock, NEVER a pump (see the Driver banner).
func New_Sim(seed uint64) (loop IO, driver Driver, clock time.Clock) {
	clock, tick := time.Virtual_Clock_To_Clock(time.Virtual_Clock{Resolution: time.NANOSECOND})
	state := &sim{
		Clock:           clock,
		Tick:            tick,
		Generator:       prng.New(seed),
		Files:           map[File]*sim_node{},
		Raw_Open:        map[File]bool{},
		Sockets:         map[File]*sim_socket{},
		Events:          map[Event]*sim_event{},
		Operations:      map[*Completion]sim_operation{},
		Operation_Files: map[*Completion]File{},
	}
	state.Root = sim_generate(&state.Generator)
	sim_wire_bytes(state, &loop)
	sim_wire_lifecycle(state, &loop)
	sim_wire_socket(state, &loop)
	sim_wire_effects(state, &loop)
	sim_wire_platform(state, &loop)
	return loop, sim_to_driver(state), clock
}

// Wires the byte-count operations — read, write, receive, send — onto loop.
func sim_wire_bytes(state *sim, loop *IO) {
	loop.Read = func(
		completion *Completion, callback Callback, file File, buffer []byte, offset int64,
	) {
		node := state.Files[file]
		if node != nil {
			sim_file_read(state, completion, callback, node, buffer, offset)
			state.Operation_Files[completion] = file
			return
		}
		sim_bytes(state, completion, callback, buffer)
		state.Operation_Files[completion] = file
	}
	loop.Write = func(
		completion *Completion, callback Callback, file File, buffer []byte, offset int64,
	) {
		node := state.Files[file]
		if node != nil {
			sim_file_write(state, completion, callback, node, buffer, offset)
			state.Operation_Files[completion] = file
			return
		}
		sim_bytes(state, completion, callback, buffer)
		state.Operation_Files[completion] = file
	}
	loop.Receive = func(completion *Completion, callback Callback, socket File, buffer []byte) {
		sim_submit(state, completion, sim_latency(state), func() {
			socket_state := state.Sockets[socket]
			if socket_state == nil {
				callback(completion, 0, nil)
				return
			}
			if socket_state.Receive_Shutdown {
				callback(completion, 0, nil)
				return
			}
			callback(completion, len(buffer), nil)
		})
		state.Operations[completion] = sim_operation_read_waiter
		state.Operation_Files[completion] = socket
	}
	loop.Send = func(completion *Completion, callback Callback, socket File, buffer []byte) {
		sim_submit(state, completion, sim_latency(state), func() {
			socket_state := state.Sockets[socket]
			if socket_state == nil {
				callback(completion, 0, Broken_Pipe)
				return
			}
			if socket_state.Send_Shutdown {
				callback(completion, 0, Broken_Pipe)
				return
			}
			callback(completion, len(buffer), nil)
		})
		state.Operations[completion] = sim_operation_write_waiter
		state.Operation_Files[completion] = socket
	}
	loop.Fsync = func(completion *Completion, callback Timeout_Callback, file File) {
		sim_submit(state, completion, sim_latency(state),
			sim_deliver_status(completion, callback, nil))
		state.Operation_Files[completion] = file
	}
}

// Wires timers, next ticks, filesystem lifecycle, and both close primitives onto loop.
func sim_wire_lifecycle(state *sim, loop *IO) {
	loop.Open_At = func(
		completion *Completion, callback File_Callback, directory File, file_path string,
		options Open_At_Options,
	) {
		sim_submit(state, completion, sim_latency(state), func() {
			file := File(-1)
			var open_err error
			if directory != DIRECTORY_CURRENT {
				open_err = sim_not_a_directory
			} else if options.Create {
				file, open_err = sim_create(state, file_path)
			} else {
				file, open_err = sim_open(state, file_path)
			}
			callback(completion, file, open_err)
		})
	}
	loop.Timeout = func(
		completion *Completion, callback Timeout_Callback, duration time.Duration,
	) {
		invariant.Always(
			duration > 0, "A timeout duration is positive; yields use Next_Tick.",
		)
		sim_submit(state, completion, duration,
			sim_deliver_status(completion, callback, nil))
		state.Operations[completion] = sim_operation_timeout
	}
	loop.Next_Tick = func(
		completion *Completion, callback Next_Tick_Callback, source Next_Tick_Source,
	) {
		sim_submit(state, completion, 0, func() { callback(completion) })
		completion.Next_Tick_Source = source
		completion.Next_Tick = true
		state.Operations[completion] = sim_operation_next_tick
	}
	loop.Reset_Next_Tick = func(source Next_Tick_Source) {
		sim_reset_next_tick(state, source)
	}
	loop.Open_Event = func() (event Event, err error) {
		state.Next_Event++
		event = state.Next_Event
		state.Events[event] = &sim_event{}
		return event, nil
	}
	loop.Event_Listen = func(
		event Event, completion *Completion, callback Next_Tick_Callback,
	) {
		sim_event_listen(state, event, completion, callback)
	}
	loop.Event_Trigger = func(event Event, completion *Completion) {
		sim_event_trigger(state, event, completion)
	}
	loop.Close_Event = func(event Event) {
		entry := state.Events[event]
		invariant.Always(entry != nil, "A closed Event is open.")
		invariant.Always(
			entry.Completion == nil, "An Event listener is drained before close.",
		)
		delete(state.Events, event)
	}
	sim_wire_filesystem(state, loop)
}

// Wires simulated filesystem and socket lifecycle operations onto loop.
func sim_wire_filesystem(state *sim, loop *IO) {
	loop.Listen = func(
		socket File, address Address, options Listen_Options,
	) (resolved Address, err error) {
		socket_state := state.Sockets[socket]
		if socket_state == nil {
			return Address{}, errors.New("io: listen requires an open TCP socket")
		}
		if socket_state.Datagram {
			return Address{}, errors.New("io: listen requires an open TCP socket")
		}
		if options.Backlog == 0 {
			return Address{}, errors.New("io: listen backlog must be positive")
		}
		socket_state.Listener = true
		resolved = address
		if resolved.Port == 0 {
			resolved.Port = uint16(socket)
		}
		return resolved, nil
	}
	loop.Open = func(path string) (file File, err error) {
		return sim_open(state, path)
	}
	loop.Create = func(path string) (file File, err error) {
		return sim_create(state, path)
	}
	loop.Read_Directory = func(path string) (entries []Directory_Entry, err error) {
		return sim_read_directory(state.Root, path)
	}
	loop.Status = func(path string) (status File_Status, err error) {
		return sim_status(state.Root, path), nil
	}
	loop.Make_Directory = func(path string) (err error) {
		return sim_make_directory(state.Root, path)
	}
	loop.Close = func(completion *Completion, callback Timeout_Callback, file File) {
		sim_assert_file_drained(state, file)
		sim_submit(state, completion, sim_latency(state), func() {
			delete(state.Files, file)
			delete(state.Sockets, file)
			delete(state.Raw_Open, file)
			callback(completion, nil)
		})
	}
	loop.Close_Socket = func(socket File) {
		sim_assert_file_drained(state, socket)
		delete(state.Sockets, socket)
		delete(state.Raw_Open, socket)
	}
}

// Arms one simulated Event listener without making it ready until Event_Trigger fires.
func sim_event_listen(
	state *sim, event Event, completion *Completion, callback Next_Tick_Callback,
) {
	entry := state.Events[event]
	invariant.Always(entry != nil, "An Event listener attaches to an open Event.")
	invariant.Always(entry.Completion == nil, "An Event has at most one armed listener.")
	sim_arm(state, completion, func() { callback(completion) })
	state.Operations[completion] = sim_operation_event
	entry.Completion = completion
	if entry.Triggered > 0 {
		entry.Triggered--
		entry.Completion = nil
		completion.Ready_At = sim_now(state)
		sim_enqueue(state, completion)
	}
}

// Triggers one simulated Event notification, accumulating it when no listener is armed.
func sim_event_trigger(state *sim, event Event, completion *Completion) {
	entry := state.Events[event]
	invariant.Always(entry != nil, "A triggered Event is open.")
	if entry.Completion == nil {
		entry.Triggered++
		return
	}
	invariant.Always(entry.Completion == completion,
		"Event_Trigger names the Completion armed by Event_Listen.")
	entry.Completion = nil
	completion.Ready_At = sim_now(state)
	sim_enqueue(state, completion)
}

// Asserts no submitted operation still borrows file. TigerBeetle's teardown owner performs this
// join before close (third-party/tigerbeetle/src/message_bus.zig:1104-1145); the Go port enforces
// that ownership boundary at the library surface so descriptor reuse cannot race an old operation.
func sim_assert_file_drained(state *sim, file File) {
	borrowed := false
	for _, operation_file := range state.Operation_Files {
		if operation_file == file {
			borrowed = true
		}
	}
	invariant.Always(!borrowed,
		"A descriptor is drained before Close or Close_Socket releases it.")
}

// Wires the socket operations — accept, connect, peer address — onto loop. TLS is not an
// io concern: there is no accept-secure or connect-secure syscall, so secure_transport
// composes these raw ops with its own record layer.
func sim_wire_socket(state *sim, loop *IO) {
	loop.Accept = func(
		completion *Completion, callback Socket_Callback, listener File,
		deadline time.Duration,
	) {
		invariant.Always(deadline > 0, "An accept deadline is positive and finite.")
		sim_yield_socket(state, completion, callback, listener, deadline)
		state.Operations[completion] = sim_operation_read_waiter
		state.Operation_Files[completion] = listener
	}
	loop.Open_Socket_TCP = func(
		family Address_Family, options TCP_Options,
	) (socket File, err error) {
		return sim_open_socket(state, family, false), nil
	}
	loop.Open_Socket_UDP = func(family Address_Family) (socket File, err error) {
		return sim_open_socket(state, family, true), nil
	}
	loop.Connect = func(
		completion *Completion, callback Timeout_Callback,
		socket File, address Address, deadline time.Duration,
	) {
		sim_connect(state, completion, callback, socket, address, deadline)
	}
	loop.Send_Now = func(socket File, buffer []byte) (count int, sent bool) {
		socket_state := state.Sockets[socket]
		if socket_state == nil {
			return 0, false
		}
		if socket_state.Send_Shutdown {
			return 0, false
		}
		return len(buffer), true
	}
	loop.Shutdown = func(socket File, how Shutdown_How) (err error) {
		socket_state := state.Sockets[socket]
		if socket_state == nil {
			return Socket_Not_Connected
		}
		if !socket_state.Connected {
			return Socket_Not_Connected
		}
		if how == SHUTDOWN_RECEIVE {
			socket_state.Receive_Shutdown = true
		}
		if how == SHUTDOWN_BOTH {
			socket_state.Receive_Shutdown = true
		}
		if how == SHUTDOWN_SEND {
			socket_state.Send_Shutdown = true
		}
		if how == SHUTDOWN_BOTH {
			socket_state.Send_Shutdown = true
		}
		return nil
	}
	loop.Peer_Address = func(file File) (address string, err error) {
		return sim_peer_address(file), nil
	}
}

// Submits one simulator Connect with one latency draw; the finite deadline wins a tie.
func sim_connect(
	state *sim, completion *Completion, callback Timeout_Callback,
	socket File, _ Address, deadline time.Duration,
) {
	invariant.Always(deadline > 0, "A connect deadline is positive and finite.")
	connect_err := error(nil)
	if prng.Generator_Below(&state.Generator, 4) == 0 {
		connect_err = Connection_Refused
	}
	latency := sim_latency(state)
	if latency >= deadline {
		sim_submit(state, completion, deadline,
			sim_deliver_status(completion, callback, Deadline_Exceeded))
		state.Operations[completion] = sim_operation_write_waiter
		state.Operation_Files[completion] = socket
		return
	}
	sim_submit(state, completion, latency, func() {
		if connect_err == nil {
			socket_state := state.Sockets[socket]
			if socket_state != nil {
				socket_state.Connected = true
			}
		}
		callback(completion, connect_err)
	})
	state.Operations[completion] = sim_operation_write_waiter
	state.Operation_Files[completion] = socket
}

// Wires the effect operations — signal watch and compute offload — onto loop.
func sim_wire_effects(state *sim, loop *IO) {
	loop.Watch_Signal = func(
		completion *Completion, callback Signal_Callback, signal Signal,
		deadline time.Duration,
	) {
		invariant.Always(deadline > 0, "A signal-watch deadline is positive and finite.")
		sim_watch_signal(state, completion, callback, signal, deadline)
	}
	loop.Compute = func(completion *Completion, callback Compute_Callback, work func()) {
		sim_compute(state, completion, callback, work)
	}
	loop.Spawn = func(
		completion *Completion, callback Process_Callback, request Process_Request,
		deadline time.Duration,
	) {
		invariant.Always(deadline > 0, "A spawn deadline is positive and finite.")
		sim_spawn(state, completion, callback, request, deadline)
	}
	loop.Self_Exec = func(path string, argv []string, _ []string) (err error) {
		return sim_self_exec_unsupported
	}
}

// The error every simulated Self_Exec returns: the simulator cannot replace its own test
// process, so it reports the failure rather than pretending to succeed (which would destroy
// the run). A caller's real-backend success path never returns, so its fallback branch is
// exactly what the simulator exercises.
var sim_self_exec_unsupported = errors.New("io: self-exec is not supported by the simulator")

// Delivers a subprocess result drawn from the seed: the exit code varies (usually zero,
// occasionally non-zero for fault coverage) with no captured output — scripted output is
// disallowed, so the seed decides success or failure, not a canned payload.
func sim_spawn(
	state *sim, completion *Completion, callback Process_Callback, request Process_Request,
	deadline time.Duration,
) {
	exit := 0
	if prng.Generator_Below(&state.Generator, sim_spawn_fail_grains) == 0 {
		exit = 1
	}
	latency := sim_latency(state)
	if latency >= deadline {
		sim_submit(state, completion, deadline, func() {
			callback(completion, Process_Result{}, Deadline_Exceeded)
		})
		state.Operations[completion] = sim_operation_posted
		return
	}
	sim_submit(state, completion, latency, func() {
		callback(completion, Process_Result{Exit: exit}, nil)
	})
	state.Operations[completion] = sim_operation_posted
}

// Panics on a violated invariant, fail-closed — a tripped assert is always a bug in
// this package, so the simulation stops loudly instead of corrupting on.
// Hands out the next distinct synthetic descriptor.
func sim_descriptor(state *sim) (file File) {
	state.Next_File++
	return state.Next_File
}

// Opens and records one caller-owned synthetic socket.
func sim_open_socket(
	state *sim, family Address_Family, datagram bool,
) (socket File) {
	socket = sim_descriptor(state)
	state.Raw_Open[socket] = true
	state.Sockets[socket] = &sim_socket{Family: family, Datagram: datagram}
	return socket
}

// Returns a fresh empty directory node, the shape the root, mkdir, and the generator all
// build.
func sim_new_directory() (node *sim_node) {
	return &sim_node{Directory: true, Children: map[string]*sim_node{}}
}

// Splits an absolute path into its non-empty component names, so "/a/b" walks as a, b.
func sim_path_names(path string) (names []string) {
	names = []string{}
	for _, name := range strings.Split(path, "/") {
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

// Resolves path against root, returning the node it names and whether it was found.
func sim_resolve(root *sim_node, path string) (node *sim_node, found bool) {
	node = root
	for _, name := range sim_path_names(path) {
		if !node.Directory {
			return nil, false
		}
		child, present := node.Children[name]
		if !present {
			return nil, false
		}
		node = child
	}
	return node, true
}

// Reports path's status against root: an unresolved path is Exists false, else its kind.
func sim_status(root *sim_node, path string) (status File_Status) {
	node, found := sim_resolve(root, path)
	if !found {
		return File_Status{}
	}
	return File_Status{
		Exists:       true,
		Is_Directory: node.Directory,
		Size:         int64(len(node.Contents)),
	}
}

// Lists path's immediate children sorted by name — sorted so the order is deterministic
// despite the backing map, since a run must reproduce. Absent or non-directory paths error.
func sim_read_directory(root *sim_node, path string) (entries []Directory_Entry, err error) {
	node, found := sim_resolve(root, path)
	if !found {
		return nil, sim_file_absent
	}
	if !node.Directory {
		return nil, sim_not_a_directory
	}
	entries = []Directory_Entry{}
	for name, child := range node.Children {
		entry := Directory_Entry{Name: name, Is_Directory: child.Directory}
		entries = append(entries, entry)
	}
	slices.SortFunc(entries, func(left, right Directory_Entry) (order int) {
		return strings.Compare(left.Name, right.Name)
	})
	return entries, nil
}

// Creates path and any missing parents against root; an existing directory converges, and a
// file where a directory is needed errors.
func sim_make_directory(root *sim_node, path string) (err error) {
	node := root
	for _, name := range sim_path_names(path) {
		if !node.Directory {
			return sim_not_a_directory
		}
		child, present := node.Children[name]
		if !present {
			child = sim_new_directory()
			node.Children[name] = child
		}
		node = child
	}
	if !node.Directory {
		return sim_not_a_directory
	}
	return nil
}

// Opens path for reading against state.Root, binding a fresh descriptor to its node. An
// absent path, or a directory, errors — matching a real open of a missing or non-file path.
func sim_open(state *sim, path string) (file File, err error) {
	node, found := sim_resolve(state.Root, path)
	if !found {
		return 0, sim_file_absent
	}
	if node.Directory {
		return 0, sim_is_a_directory
	}
	descriptor := sim_descriptor(state)
	state.Files[descriptor] = node
	state.Raw_Open[descriptor] = true
	return descriptor, nil
}

// Creates or truncates the file at path against state.Root and binds a fresh descriptor to
// it. The parent directory must already exist, matching a real create.
func sim_create(state *sim, path string) (file File, err error) {
	node, create_err := sim_create_file(state.Root, path)
	if create_err != nil {
		return 0, create_err
	}
	descriptor := sim_descriptor(state)
	state.Files[descriptor] = node
	state.Raw_Open[descriptor] = true
	return descriptor, nil
}

// Resolves path's parent (which must be an existing directory), then creates a fresh file
// node under it or truncates an existing file, returning the node.
func sim_create_file(root *sim_node, path string) (node *sim_node, err error) {
	names := sim_path_names(path)
	if len(names) == 0 {
		return nil, sim_is_a_directory
	}
	parent := root
	for _, name := range names[:len(names)-1] {
		child, present := parent.Children[name]
		if !present {
			return nil, sim_file_absent
		}
		if !child.Directory {
			return nil, sim_not_a_directory
		}
		parent = child
	}
	leaf := names[len(names)-1]
	leaf_node, present := parent.Children[leaf]
	if present {
		if leaf_node.Directory {
			return nil, sim_is_a_directory
		}
		leaf_node.Contents = []byte{}
		return leaf_node, nil
	}
	created := &sim_node{Contents: []byte{}}
	parent.Children[leaf] = created
	return created, nil
}

// Submits a file read that, when it fires, copies the node's bytes from offset into the
// buffer and reports the count — so a read reflects whatever an earlier write stored.
func sim_file_read(
	state *sim, completion *Completion, callback Callback, node *sim_node,
	buffer []byte, offset int64,
) {
	sim_submit(state, completion, sim_latency(state), func() {
		count := 0
		if offset < int64(len(node.Contents)) {
			count = copy(buffer, node.Contents[offset:])
		}
		callback(completion, count, nil)
	})
}

// Submits a file write that, when it fires, stores the buffer into the node at offset,
// growing its contents as needed, and reports the byte count.
func sim_file_write(
	state *sim, completion *Completion, callback Callback, node *sim_node,
	buffer []byte, offset int64,
) {
	sim_submit(state, completion, sim_latency(state), func() {
		sim_node_write(node, buffer, offset)
		callback(completion, len(buffer), nil)
	})
}

// Stores buffer into node's contents at offset, growing the backing bytes when the write
// extends past the current end.
func sim_node_write(node *sim_node, buffer []byte, offset int64) {
	end_size := offset + int64(len(buffer))
	if end_size > int64(len(node.Contents)) {
		grown := make([]byte, end_size)
		copy(grown, node.Contents)
		node.Contents = grown
	}
	copy(node.Contents[offset:], buffer)
}

// The file-size percentiles the generated contents are sampled from: most files are a
// handful of bytes, a few reach hundreds, and the top one percent the largest — a
// heavy-tailed spread (prng.Percentile_Distribution) so a sweep meets many scales at once.
const sim_size_p50 = 4
const sim_size_p75 = 16
const sim_size_p95 = 64
const sim_size_p99 = 256
const sim_size_p100 = 1024

// Returns how often the generator adds another sibling — a 3-in-4 Chance, so a directory's
// breadth is geometric and any width is reachable rather than capped at a fixed count.
func sim_grow_chance() (chance prng.Ratio) {
	return prng.Ratio{Numerator: 3, Denominator: 4}
}

// Returns how often a generated child is a subdirectory rather than a file — kept below the
// grow chance so the branching stays subcritical and generation halts almost surely.
func sim_subdirectory_chance() (chance prng.Ratio) {
	return prng.Ratio{Numerator: 1, Denominator: 4}
}

// Fabricates a filesystem from the seed, drawing its shape from prng's distributions: each
// directory grows siblings on a Chance coin (geometric breadth and depth, no ceiling), a
// child is a subdirectory on another Chance, and a file's size is Sampled from a heavy-tailed
// Percentile spread. It knows no consumer's layout; a walker imposes its own meaning.
func sim_generate(generator *prng.Generator) (root *sim_node) {
	sizes := prng.Percentile_Distribution(&prng.Percentile_Distribution_Input{
		P25:  0,
		P50:  sim_size_p50,
		P75:  sim_size_p75,
		P95:  sim_size_p95,
		P99:  sim_size_p99,
		P100: sim_size_p100,
	})
	root = sim_new_directory()
	directories := []*sim_node{root}
	for len(directories) > 0 {
		directory := directories[len(directories)-1]
		directories = directories[:len(directories)-1]
		index := 0
		for prng.Generator_Chance(generator, sim_grow_chance()) {
			name := "e" + strconv.Itoa(index)
			index++
			if prng.Generator_Chance(generator, sim_subdirectory_chance()) {
				child := sim_new_directory()
				directory.Children[name] = child
				directories = append(directories, child)
				continue
			}
			contents := sim_generate_bytes(generator, sizes)
			directory.Children[name] = &sim_node{Contents: contents}
		}
	}
	return root
}

// Draws a file's contents from the seed: a size Sampled from the heavy-tailed distribution,
// filled with seed-drawn bytes.
func sim_generate_bytes(
	generator *prng.Generator, sizes prng.Distribution[uint64],
) (contents []byte) {
	contents = make([]byte, prng.Generator_Sample(generator, sizes))
	for index := range contents {
		contents[index] = byte(prng.Generator_Next(generator))
	}
	return contents
}

// Draws the next operation's completion delay from the seed, so completion order
// varies per run yet reproduces exactly.
func sim_latency(state *sim) (latency time.Duration) {
	return time.Duration(prng.Generator_Below(&state.Generator, sim_latency_grains))
}

// Wraps a successful byte-count delivery.
func sim_deliver_bytes(completion *Completion, callback Callback, count int) (deliver func()) {
	return func() { callback(completion, count, nil) }
}

// Wraps a status delivery for a timeout or another result-only operation.
func sim_deliver_status(
	completion *Completion, callback Timeout_Callback, err error,
) (deliver func()) {
	return func() { callback(completion, err) }
}

// Submits a byte-count operation reporting the buffer length after the drawn latency.
func sim_bytes(state *sim, completion *Completion, callback Callback, buffer []byte) {
	sim_submit(state, completion, sim_latency(state),
		sim_deliver_bytes(completion, callback, len(buffer)))
}

// Schedules callback to receive the next connected synthetic descriptor after drawn latency.
func sim_yield_socket(
	state *sim, completion *Completion, callback Socket_Callback, listener File,
	deadline time.Duration,
) {
	latency := sim_latency(state)
	if latency >= deadline {
		sim_submit(state, completion, deadline, func() {
			callback(completion, File(-1), Deadline_Exceeded)
		})
		return
	}
	sim_submit(state, completion, latency, func() {
		listener_state := state.Sockets[listener]
		if listener_state == nil {
			callback(completion, 0, errors.New("io: socket is not listening"))
			return
		}
		if !listener_state.Listener {
			callback(completion, 0, errors.New("io: socket is not listening"))
			return
		}
		socket := sim_open_socket(state, listener_state.Family, false)
		state.Sockets[socket].Connected = true
		callback(completion, socket, nil)
	})
}

// Reports a synthetic peer address for a live descriptor, the simulator's getpeername;
// empty with no error for an unknown descriptor.
func sim_peer_address(file File) (address string) {
	if file <= 0 {
		return ""
	}
	return "127.0.0.1"
}

// Watches for a signal that, in the simulator, arrives at a seed-drawn grain — the OS
// event modeled as a seed outcome. It fires callback with the signal exactly once.
func sim_watch_signal(
	state *sim, completion *Completion, callback Signal_Callback, signal Signal,
	deadline time.Duration,
) {
	latency := sim_latency(state)
	if latency >= deadline {
		sim_submit(state, completion, deadline, func() {
			callback(completion, Signal(-1), Deadline_Exceeded)
		})
	} else {
		sim_submit(state, completion, latency, func() { callback(completion, signal, nil) })
	}
	state.Operations[completion] = sim_operation_signal
}

// Runs work inline after the drawn latency, then fires callback on the loop — the
// deterministic counterpart of the OS backend's worker pool.
func sim_compute(
	state *sim, completion *Completion, callback Compute_Callback, work func(),
) {
	sim_submit(state, completion, sim_latency(state), func() {
		work()
		callback(completion)
	})
	state.Operations[completion] = sim_operation_result
}

// Builds the driver over state — the loop-advancing capability, held only by main or a
// test, never by code that merely submits IO.
func sim_to_driver(state *sim) (driver Driver) {
	return Driver{
		Run: func() (err error) {
			sim_drive(state, func() { sim_run(state) })
			return nil
		},
		Run_For: func(duration time.Duration) (err error) {
			sim_drive(state, func() { sim_run_for(state, duration) })
			return nil
		},
		Run_Until: func(
			done func() (finished bool), timeout time.Duration,
		) (completed bool, err error) {
			sim_drive(state, func() { completed = sim_run_until(state, done, timeout) })
			return completed, nil
		},
		Deinit:     func() {},
		Introspect: func() (counts Loop_Counts) { return sim_introspect(state) },
	}
}

// Samples every simulated queue class and lifecycle flag without exposing simulator state.
func sim_introspect(state *sim) (counts Loop_Counts) {
	for _, operation := range state.Operations {
		if operation == sim_operation_completed {
			counts.Completed++
		}
		if operation == sim_operation_timeout {
			counts.Timeouts++
		}
		if operation == sim_operation_read_waiter {
			counts.IO_Backlog++
		}
		if operation == sim_operation_write_waiter {
			counts.IO_Backlog++
		}
		if operation == sim_operation_signal {
			counts.Signal_Waiters++
		}
		if operation == sim_operation_posted {
			counts.Posted++
		}
		if operation == sim_operation_result {
			counts.Results++
		}
	}
	counts.Raw_Open = len(state.Raw_Open)
	counts.Wake_Active = counts.Posted > 0 || counts.Results > 0
	counts.Compute_Active = counts.Results > 0
	return counts
}

// Runs pump as the top-level drive, asserting no drive is already in progress so a Run*
// called from within a completion callback panics instead of re-entering the driver. The
// internal per-tick functions call one another directly, not through here, so nested
// ticking within one drive does not trip it.
func sim_drive(state *sim, pump func()) {
	invariant.Always(!state.Drive_Active,
		"A drive begins at top level, never from within a completion callback.")
	state.Drive_Active = true
	defer func() { state.Drive_Active = false }()
	pump()
}

// Returns the current virtual Moment; the sim never reads the operating-system time.
func sim_now(state *sim) (now time.Moment) {
	return state.Clock.Now_Monotonic()
}

// Schedules completion to fire at now plus latency and inserts it in Ready_At order.
// Asserts the completion is its own original (not a by-value copy), then arms it through
// the lifecycle machine — a reused in-flight completion panics as the armed-to-armed
// edge.
func sim_submit(state *sim, completion *Completion, latency time.Duration, callback func()) {
	sim_arm(state, completion, callback)
	completion.Ready_At = sim_now(state) + time.Moment(latency)
	sim_enqueue(state, completion)
}

// Arms completion without placing it on the ready-time queue, for TigerBeetle Event listeners.
func sim_arm(state *sim, completion *Completion, callback func()) {
	original := completion.Self == nil || completion.Self == completion
	invariant.Always(original,
		"A submitted completion is its own original, never a by-value copy.")
	completion.Self = completion
	Completion_Transition(&Completion_Transition_Input{
		Completion: completion, From: COMPLETION_IDLE, To: COMPLETION_ARMED,
	})
	completion.Callback = callback
	completion.Next_Tick = false
	state.Operations[completion] = sim_operation_completed
}

// Inserts completion into the queue in Ready_At order, earliest first.
func sim_enqueue(state *sim, completion *Completion) {
	index := 0
	for index < len(state.Queue) && state.Queue[index].Ready_At <= completion.Ready_At {
		index++
	}
	state.Queue = append(state.Queue, nil)
	copy(state.Queue[index+1:], state.Queue[index:])
	state.Queue[index] = completion
}

// Removes every queued next-tick completion for source and retires it without delivery,
// matching io/linux.zig:354-367 and io/darwin.zig:783-796.
func sim_reset_next_tick(state *sim, source Next_Tick_Source) {
	kept := state.Queue[:0]
	for _, completion := range state.Queue {
		operation := state.Operations[completion]
		if operation != sim_operation_next_tick {
			kept = append(kept, completion)
			continue
		}
		if completion.Next_Tick_Source != source {
			kept = append(kept, completion)
			continue
		}
		delete(state.Operations, completion)
		delete(state.Operation_Files, completion)
		Completion_Transition(&Completion_Transition_Input{
			Completion: completion, From: COMPLETION_ARMED, To: COMPLETION_IDLE,
		})
	}
	state.Queue = kept
}

// Fires the earliest completion if it is due as of now, reporting whether it did,
// mirroring TigerBeetle's Storage.step.
func sim_step(state *sim) (advanced bool) {
	if len(state.Queue) == 0 {
		return false
	}
	if state.Queue[0].Ready_At > sim_now(state) {
		return false
	}
	completion := state.Queue[0]
	state.Queue = state.Queue[1:]
	delete(state.Operations, completion)
	delete(state.Operation_Files, completion)
	// Return to idle before the callback runs — TigerBeetle's ordering — so a callback
	// may legally resubmit its own completion, the repeating-timer pattern.
	Completion_Transition(&Completion_Transition_Input{
		Completion: completion, From: COMPLETION_ARMED, To: COMPLETION_IDLE,
	})
	completion.Callback()
	return true
}

// Drains every completion due as of now, in Ready_At order; it never advances time —
// advancing is the driver's job, so the queue itself stays passive.
func sim_drain(state *sim) {
	for sim_step(state) {
	}
}

// The driver's step: drain due completions, then advance the clock one grain via the
// injected tick (not the read-only Clock), mirroring TigerBeetle's Storage.run.
//
// The driver advances one grain per tick and never jumps ahead to the next Ready_At,
// even when the queue is idle until then. A jump would skip the grains where a
// time-triggered fault adversary — crash or partition a quiescent node, the faults
// that matter most because they strike while nothing is scheduled — would act. Uniform
// ticking keeps every grain a decision point, as TigerBeetle's simulator does for its
// per-tick crash/partition rolls.
func sim_run(state *sim) {
	sim_drain(state)
	state.Tick()
}

// Drives the loop until the duration has elapsed, delivering completions as due.
func sim_run_for(state *sim, duration time.Duration) {
	deadline := sim_now(state) + time.Moment(duration)
	for sim_now(state) < deadline {
		sim_run(state)
	}
}

// Drives the loop until done reports true, or until timeout of virtual time has elapsed —
// the run-until-complete pump, capped so a stalled op cannot spin the sim forever. A
// negative timeout waits unbounded; a zero timeout checks done once and drives nothing.
// completed reports whether done tripped rather than the deadline. Top-level and
// single-loop only; each step drains then ticks.
func sim_run_until(
	state *sim, done func() (finished bool), timeout time.Duration,
) (completed bool) {
	deadline := sim_now(state) + time.Moment(timeout)
	for !done() {
		if timeout >= 0 {
			if sim_now(state) >= deadline {
				return false
			}
		}
		sim_run(state)
	}
	return true
}
