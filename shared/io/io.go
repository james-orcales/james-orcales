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
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
	"unsafe"

	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/random/prng"
	"local/james-orcales/shared/time"
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

// Open_At_Flags selects independent Open_At controls.
type Open_At_Flags uint32

// OPEN_AT_NO_FOLLOW rejects a symbolic link in the final path part.
const OPEN_AT_NO_FOLLOW Open_At_Flags = 1 << 0

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
	// Flags contains independent Open_At controls.
	Flags Open_At_Flags
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

// These bounds keep the public address layout equal to the network protocol layouts.
const IPV4_ADDRESS_BYTES = 4

// IPV6_ADDRESS_BYTES keeps the public address layout equal to the IPv6 protocol layout.
const IPV6_ADDRESS_BYTES = 16

// Address is an IP address and port with an explicit family, corresponding to TigerBeetle's
// stdx.SocketAddress. IP stores IPv4 bytes in its first four positions and IPv6 bytes in all 16.
type Address struct {
	// Family selects how IP is interpreted.
	Family Address_Family
	// IP holds the network address bytes.
	IP [IPV6_ADDRESS_BYTES]byte
	// Port is the host-order TCP or UDP port.
	Port uint16
}

// Address_I_Pv4 returns an IPv4 address from its four octets and host-order port.
func Address_I_Pv4(ip [IPV4_ADDRESS_BYTES]byte, port uint16) (address Address) {
	address.Family = FAMILY_IPV4
	copy(address.IP[:4], ip[:])
	address.Port = port
	return address
}

// Address_I_Pv6 returns an IPv6 address from its 16 octets and host-order port.
func Address_I_Pv6(ip [IPV6_ADDRESS_BYTES]byte, port uint16) (address Address) {
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

// Process_Request describes a subprocess to run: the executable, its arguments and
// environment, the working directory, and the bytes fed to its standard input.
type Process_Request struct {
	// Path is the executable to run.
	Path string
	// Arguments are the process arguments, excluding the program name.
	Arguments []string
	// Environment is the complete process environment. A nil value gives the child none, so a
	// caller that wants an ambient value must inject it from its root.
	Environment []string
	// Working_Directory is the process's directory; empty uses the current one.
	Working_Directory string
	// Input is the bytes written to the process's standard input.
	Input []byte
	// Stdout, when its Procedure is set, streams the process's standard output to the stream
	// as it runs instead of capturing it into Result.Output — the affordance a long build
	// needs so its progress reaches the user live. A zero Stream keeps the captured-buffer
	// default. The simulated backend produces no output and ignores it.
	//
	// The loop writes to the stream on its own thread, so a Stream that waits on the world
	// stalls every other operation. A Stream moves memory only, which is what makes it the
	// right sink here.
	Stdout Stream
	// Stderr is the standard-error counterpart, same live-or-capture rule.
	Stderr Stream
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

// Socket_Transport selects the transport a Socket carries.
type Socket_Transport int

// SOCKET_TRANSPORT_TCP selects a stream socket.
const SOCKET_TRANSPORT_TCP Socket_Transport = 0

// SOCKET_TRANSPORT_UDP selects a datagram socket.
const SOCKET_TRANSPORT_UDP Socket_Transport = 1

// Socket_Option names one integer socket option the backend can set.
type Socket_Option int

// SOCKET_OPTION_RECEIVE_BUFFER sets the receive buffer size in bytes.
const SOCKET_OPTION_RECEIVE_BUFFER Socket_Option = 0

// SOCKET_OPTION_SEND_BUFFER sets the send buffer size in bytes.
const SOCKET_OPTION_SEND_BUFFER Socket_Option = 1

// SOCKET_OPTION_KEEPALIVE enables the transport keepalive probe.
const SOCKET_OPTION_KEEPALIVE Socket_Option = 2

// SOCKET_OPTION_REUSE_ADDRESS lets a listener rebind a recently released local address.
const SOCKET_OPTION_REUSE_ADDRESS Socket_Option = 3

// SOCKET_OPTION_NO_DELAY disables the transport's send coalescing.
const SOCKET_OPTION_NO_DELAY Socket_Option = 4

// Directory_Callback receives one pass of directory entries, or the error that ended the walk.
// An empty slice with a nil error reports the end of the directory.
type Directory_Callback func(
	completion *Completion, entries []Directory_Entry, err error,
)

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
	// Is_Regular reports whether an existing path is a regular file.
	Is_Regular bool
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

// Path_Exists is the portable Mkdir_At result for a path that is already present. Make_Directory
// treats it as convergence, so a repeated create is not an error.
var Path_Exists = errors.New("io: file exists")

// Retired_Twice reports that a backend retired one completion more than one time. A derived
// function delivers it rather than hiding it, because the caller owns the completion and must
// learn that its lifecycle broke.
var Retired_Twice = errors.New("io: the completion retired more than once")

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
const SIM_LATENCY_GRAINS = 8

// One in this many simulated spawns exits non-zero, so a seed sweep exercises both the
// success and the failure path without a scripted outcome.
const SIM_SPAWN_FAIL_GRAINS = 4

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
// Each transition records both ends of its edge. Thus, the suite must use each legal edge.
// Backend code only. Applications never transition a completion.
func Completion_Transition(input *Completion_Transition_Input) {
	invariant.Always(input.Completion.State == input.From,
		"A completion transitions from the state its caller expects.")
	legal := Completion_Transition_Legal(&Completion_Transition_Legal_Input{
		From: input.From, To: input.To,
	})
	invariant.Always(legal, "A completion transitions along an edge its machine has.")
	input.Completion.State = input.To
	invariant.Sometimes(input.From == COMPLETION_IDLE, "the edge leaves idle")
	invariant.Sometimes(input.To == COMPLETION_IDLE, "the edge enters idle")
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
	// Mkdir_At asynchronously creates one directory named by file_path relative to directory.
	// It is the mkdirat primitive, not a parent-creating mkdir: the parent must exist, and an
	// existing path reports the operating system error rather than converging. Make_Directory
	// composes this primitive above the surface, so both backends run the same composition.
	Mkdir_At func(
		completion *Completion, callback Timeout_Callback, directory File, file_path string,
		mode uint32,
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
	// Bind gives a caller-owned socket its local address. It is the bind primitive: Listen
	// composes it with Listen_Socket and Get_Socket_Name above the surface.
	Bind func(socket File, address Address) (err error)
	// Listen_Socket marks a bound socket as accepting, with backlog as its queue depth.
	Listen_Socket func(socket File, backlog uint32) (err error)
	// Get_Socket_Name reports a socket's own address, which is how a caller learns the port
	// the kernel chose for port zero.
	Get_Socket_Name func(socket File) (address Address, err error)
	// Socket creates one non-blocking close-on-exec socket of the given family and transport.
	// Open_Socket_TCP and Open_Socket_UDP compose it with Set_Socket_Option above the surface.
	Socket func(family Address_Family, transport Socket_Transport) (socket File, err error)
	// Set_Socket_Option sets one integer socket option, the setsockopt primitive.
	Set_Socket_Option func(socket File, option Socket_Option, value int) (err error)
	// Get_Directory_Entries reads one pass of a directory's raw entries into buffer and
	// returns the children it names, each with whether it is itself a directory. An empty
	// slice reports the end. The dirent layout is per-platform, so the parse and the kind
	// stay in the backend, and only the pass loop and the descriptor lifetime compose above.
	Get_Directory_Entries func(
		completion *Completion, callback Directory_Callback, directory File, buffer []byte,
	)
	// Status reports whether path exists, whether it is a directory, and its byte size,
	// synchronously; an absent path is Exists false with a nil error, so a caller branches on
	// the status, not the error.
	Status func(path string) (status File_Status, err error)
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
	// Spawns is the number of started children the backend has not yet reaped.
	Spawns int
	// Raw_Open is the number of raw descriptors the backend holds open.
	Raw_Open int
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

// Sim_Node is one entry in the simulator's in-memory filesystem: a directory with named
// children, or a file holding bytes. Generated from the seed at New_Sim and mutated by
// Create/Write/Make_Directory, so a later read reflects an earlier write.
type Sim_Node struct {
	// Directory reports whether this node is a directory rather than a file.
	Directory bool
	// Contents holds a file's bytes; nil for a directory.
	Contents []byte
	// Children maps a directory's entry names to their nodes; nil for a file.
	Children map[string]*Sim_Node
}

// Sim_Operation classifies a queued simulator completion.
type Sim_Operation int

// SIM_OPERATION_COMPLETED keeps completed callbacks in one introspection class.
const SIM_OPERATION_COMPLETED Sim_Operation = 0

// SIM_OPERATION_TIMEOUT keeps timers in one introspection class.
const SIM_OPERATION_TIMEOUT Sim_Operation = 1

// SIM_OPERATION_READ_WAITER keeps blocked receives in one introspection class.
const SIM_OPERATION_READ_WAITER Sim_Operation = 2

// SIM_OPERATION_WRITE_WAITER keeps blocked sends in one introspection class.
const SIM_OPERATION_WRITE_WAITER Sim_Operation = 3

// SIM_OPERATION_SIGNAL keeps signal waits in one introspection class.
const SIM_OPERATION_SIGNAL Sim_Operation = 4

// SIM_OPERATION_SPAWN keeps started children in one introspection class.
const SIM_OPERATION_SPAWN Sim_Operation = 5

// SIM_OPERATION_NEXT_TICK keeps next-tick callbacks in one introspection class.
const SIM_OPERATION_NEXT_TICK Sim_Operation = 6

// SIM_OPERATION_EVENT keeps event listeners in one introspection class.
const SIM_OPERATION_EVENT Sim_Operation = 7

// Sim_Event is the deterministic counterpart of TigerBeetle's EVFILT_USER/eventfd primitive.
type Sim_Event struct {
	// Completion is the currently armed listener, nil while detached.
	Completion *Completion
	// Triggered counts notifications accumulated before a listener attaches.
	Triggered int
}

// Sim_Socket is the simulator's caller-owned socket state. Shutdown is directional and never
// releases ownership; Close and Close_Socket are the only operations that remove the entry.
type Sim_Socket struct {
	// Family is the address family selected at creation.
	Family Address_Family
	// Datagram distinguishes UDP from TCP.
	Datagram bool
	// Connected reports whether Connect or Accept established the socket.
	Connected bool
	// Listener reports whether Listen_Socket consumed this socket.
	Listener bool
	// Address is the local address Bind gave the socket.
	Address Address
	// Receive_Shutdown records SHUTDOWN_RECEIVE or SHUTDOWN_BOTH.
	Receive_Shutdown bool
	// Send_Shutdown records SHUTDOWN_SEND or SHUTDOWN_BOTH.
	Send_Shutdown bool
}

// Sim holds a simulator's mutable state.
type Sim struct {
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
	Root *Sim_Node
	// Files binds an open file descriptor to its node, so Read/Write route to real tree
	// bytes; a descriptor absent from this map is a socket, whose bytes stay synthetic.
	Files map[File]*Sim_Node
	// Directory_Drained records the descriptors whose entries one pass already reported, so
	// a second Get_Directory_Entries reports the end as a real getdents does.
	Directory_Drained map[File]bool
	// Raw_Open tracks every synthetic descriptor until the caller submits Close.
	Raw_Open map[File]bool
	// Sockets holds lifecycle and directional-shutdown state for synthetic sockets.
	Sockets map[File]*Sim_Socket
	// Events holds backend Event state and is excluded from caller-owned Raw_Open accounting.
	Events map[Event]*Sim_Event
	// Next_Event supplies stable nonzero synthetic Event identifiers.
	Next_Event Event
	// Operations classifies each queued completion for full Driver.Introspect output.
	Operations map[*Completion]Sim_Operation
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
	state := &Sim{
		Clock:             clock,
		Tick:              tick,
		Generator:         prng.New(seed),
		Files:             map[File]*Sim_Node{},
		Directory_Drained: map[File]bool{},
		Raw_Open:          map[File]bool{},
		Sockets:           map[File]*Sim_Socket{},
		Events:            map[Event]*Sim_Event{},
		Operations:        map[*Completion]Sim_Operation{},
		Operation_Files:   map[*Completion]File{},
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
func sim_wire_bytes(state *Sim, loop *IO) {
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
		state.Operations[completion] = SIM_OPERATION_READ_WAITER
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
		state.Operations[completion] = SIM_OPERATION_WRITE_WAITER
		state.Operation_Files[completion] = socket
	}
	loop.Fsync = func(completion *Completion, callback Timeout_Callback, file File) {
		sim_submit(state, completion, sim_latency(state),
			sim_deliver_status(completion, callback, nil))
		state.Operation_Files[completion] = file
	}
}

// Wires the two path primitives, Open_At and Mkdir_At, onto loop.
func sim_wire_path(state *Sim, loop *IO) {
	loop.Open_At = func(
		completion *Completion, callback File_Callback, directory File, file_path string,
		options Open_At_Options,
	) {
		invariant.Always(
			options.Flags & ^OPEN_AT_NO_FOLLOW == 0,
			"Open_At options contain only known flags.",
		)
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
	loop.Mkdir_At = func(
		completion *Completion, callback Timeout_Callback, directory File, file_path string,
		_ uint32,
	) {
		sim_submit(state, completion, sim_latency(state), func() {
			if directory != DIRECTORY_CURRENT {
				callback(completion, sim_not_a_directory)
				return
			}
			callback(completion, sim_mkdir(state, file_path))
		})
	}
}

// Wires timers, next ticks, filesystem lifecycle, and both close primitives onto loop.
func sim_wire_lifecycle(state *Sim, loop *IO) {
	sim_wire_path(state, loop)
	loop.Timeout = func(
		completion *Completion, callback Timeout_Callback, duration time.Duration,
	) {
		invariant.Always(
			duration > 0, "A timeout duration is positive; yields use Next_Tick.",
		)
		sim_submit(state, completion, duration,
			sim_deliver_status(completion, callback, nil))
		state.Operations[completion] = SIM_OPERATION_TIMEOUT
	}
	loop.Next_Tick = func(
		completion *Completion, callback Next_Tick_Callback, source Next_Tick_Source,
	) {
		sim_submit(state, completion, 0, func() { callback(completion) })
		completion.Next_Tick_Source = source
		completion.Next_Tick = true
		state.Operations[completion] = SIM_OPERATION_NEXT_TICK
	}
	loop.Reset_Next_Tick = func(source Next_Tick_Source) {
		sim_reset_next_tick(state, source)
	}
	loop.Open_Event = func() (event Event, err error) {
		state.Next_Event++
		event = state.Next_Event
		state.Events[event] = &Sim_Event{}
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
func sim_wire_filesystem(state *Sim, loop *IO) {
	loop.Bind = func(socket File, address Address) (err error) {
		socket_state := state.Sockets[socket]
		if socket_state == nil {
			return errors.New("io: bind requires an open socket")
		}
		bound := address
		if bound.Port == 0 {
			bound.Port = uint16(socket)
		}
		socket_state.Address = bound
		return nil
	}
	loop.Listen_Socket = func(socket File, backlog uint32) (err error) {
		socket_state := state.Sockets[socket]
		if socket_state == nil {
			return errors.New("io: listen requires an open TCP socket")
		}
		if socket_state.Datagram {
			return errors.New("io: listen requires an open TCP socket")
		}
		if backlog == 0 {
			return errors.New("io: listen backlog must be positive")
		}
		socket_state.Listener = true
		return nil
	}
	loop.Get_Socket_Name = func(socket File) (address Address, err error) {
		socket_state := state.Sockets[socket]
		if socket_state == nil {
			return Address{}, errors.New("io: getsockname requires an open socket")
		}
		return socket_state.Address, nil
	}
	loop.Get_Directory_Entries = func(
		completion *Completion, callback Directory_Callback, directory File, _ []byte,
	) {
		sim_submit(state, completion, sim_latency(state), func() {
			callback(completion, sim_directory_pass(state, directory), nil)
		})
	}
	loop.Status = func(path string) (status File_Status, err error) {
		return sim_status(state.Root, path), nil
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
	state *Sim, event Event, completion *Completion, callback Next_Tick_Callback,
) {
	entry := state.Events[event]
	invariant.Always(entry != nil, "An Event listener attaches to an open Event.")
	invariant.Always(entry.Completion == nil, "An Event has at most one armed listener.")
	sim_arm(state, completion, func() { callback(completion) })
	state.Operations[completion] = SIM_OPERATION_EVENT
	entry.Completion = completion
	if entry.Triggered > 0 {
		entry.Triggered--
		entry.Completion = nil
		completion.Ready_At = sim_now(state)
		sim_enqueue(state, completion)
	}
}

// Triggers one simulated Event notification, accumulating it when no listener is armed.
func sim_event_trigger(state *Sim, event Event, completion *Completion) {
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
func sim_assert_file_drained(state *Sim, file File) {
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
func sim_wire_socket(state *Sim, loop *IO) {
	loop.Accept = func(
		completion *Completion, callback Socket_Callback, listener File,
		deadline time.Duration,
	) {
		invariant.Always(deadline > 0, "An accept deadline is positive and finite.")
		sim_yield_socket(state, completion, callback, listener, deadline)
		state.Operations[completion] = SIM_OPERATION_READ_WAITER
		state.Operation_Files[completion] = listener
	}
	loop.Socket = func(
		family Address_Family, transport Socket_Transport,
	) (socket File, err error) {
		return sim_open_socket(state, family, transport == SOCKET_TRANSPORT_UDP), nil
	}
	loop.Set_Socket_Option = func(socket File, _ Socket_Option, _ int) (err error) {
		if state.Sockets[socket] == nil {
			return errors.New("io: setsockopt requires an open socket")
		}
		return nil
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
	state *Sim, completion *Completion, callback Timeout_Callback,
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
		state.Operations[completion] = SIM_OPERATION_WRITE_WAITER
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
	state.Operations[completion] = SIM_OPERATION_WRITE_WAITER
	state.Operation_Files[completion] = socket
}

// Wires the effect operations — signal watch, spawn, and self-exec — onto loop.
func sim_wire_effects(state *Sim, loop *IO) {
	loop.Watch_Signal = func(
		completion *Completion, callback Signal_Callback, signal Signal,
		deadline time.Duration,
	) {
		invariant.Always(deadline > 0, "A signal-watch deadline is positive and finite.")
		sim_watch_signal(state, completion, callback, signal, deadline)
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
	state *Sim, completion *Completion, callback Process_Callback, request Process_Request,
	deadline time.Duration,
) {
	exit := 0
	if prng.Generator_Below(&state.Generator, SIM_SPAWN_FAIL_GRAINS) == 0 {
		exit = 1
	}
	latency := sim_latency(state)
	if latency >= deadline {
		sim_submit(state, completion, deadline, func() {
			callback(completion, Process_Result{}, Deadline_Exceeded)
		})
		state.Operations[completion] = SIM_OPERATION_SPAWN
		return
	}
	sim_submit(state, completion, latency, func() {
		callback(completion, Process_Result{Exit: exit}, nil)
	})
	state.Operations[completion] = SIM_OPERATION_SPAWN
}

// Panics on a violated invariant, fail-closed — a tripped assert is always a bug in
// this package, so the simulation stops loudly instead of corrupting on.
// Hands out the next distinct synthetic descriptor.
func sim_descriptor(state *Sim) (file File) {
	state.Next_File++
	return state.Next_File
}

// Opens and records one caller-owned synthetic socket.
func sim_open_socket(
	state *Sim, family Address_Family, datagram bool,
) (socket File) {
	socket = sim_descriptor(state)
	state.Raw_Open[socket] = true
	state.Sockets[socket] = &Sim_Socket{Family: family, Datagram: datagram}
	return socket
}

// Returns a fresh empty directory node, the shape the root, mkdir, and the generator all
// build.
func sim_new_directory() (node *Sim_Node) {
	return &Sim_Node{Directory: true, Children: map[string]*Sim_Node{}}
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

// Creates exactly one directory at path against state.Root. This is the mkdirat primitive, so
// the parent must already exist and an existing path is an error. Make_Directory supplies the
// convergence and the parent creation above the surface.
func sim_mkdir(state *Sim, path string) (err error) {
	names := sim_path_names(path)
	if len(names) == 0 {
		return Path_Exists
	}
	parent, found := sim_resolve(state.Root, strings.Join(names[:len(names)-1], "/"))
	if !found {
		return sim_file_absent
	}
	if !parent.Directory {
		return sim_not_a_directory
	}
	name := names[len(names)-1]
	if _, present := parent.Children[name]; present {
		return Path_Exists
	}
	parent.Children[name] = sim_new_directory()
	return nil
}

// Resolves path against root, returning the node it names and whether it was found.
func sim_resolve(root *Sim_Node, path string) (node *Sim_Node, found bool) {
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
func sim_status(root *Sim_Node, path string) (status File_Status) {
	node, found := sim_resolve(root, path)
	if !found {
		return File_Status{}
	}
	return File_Status{
		Exists:       true,
		Is_Directory: node.Directory,
		Is_Regular:   !node.Directory,
		Size:         int64(len(node.Contents)),
	}
}

// Lists path's immediate children sorted by name — sorted so the order is deterministic
// despite the backing map, since a run must reproduce. Absent or non-directory paths error.
func sim_read_directory(root *Sim_Node, path string) (entries []Directory_Entry, err error) {
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

// Returns one directory pass for an open directory descriptor. The simulated tree fits in one
// pass, so the first call gives every child and the second gives none, which is how a real
// getdents reports the end.
func sim_directory_pass(state *Sim, directory File) (entries []Directory_Entry) {
	node := state.Files[directory]
	if node == nil {
		return []Directory_Entry{}
	}
	if state.Directory_Drained[directory] {
		return []Directory_Entry{}
	}
	state.Directory_Drained[directory] = true
	entries = []Directory_Entry{}
	for name, child := range node.Children {
		entry := Directory_Entry{Name: name, Is_Directory: child.Directory}
		entries = append(entries, entry)
	}
	slices.SortFunc(entries, func(left, right Directory_Entry) (order int) {
		return strings.Compare(left.Name, right.Name)
	})
	return entries
}

// Creates path and any missing parents against root; an existing directory converges, and a
// file where a directory is needed errors.
func sim_make_directory(root *Sim_Node, path string) (err error) {
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
func sim_open(state *Sim, path string) (file File, err error) {
	node, found := sim_resolve(state.Root, path)
	if !found {
		return 0, sim_file_absent
	}
	// A directory opens, because Read_Directory composes Open_At with Get_Directory_Entries.
	// A read of the descriptor still fails, the same as a read of a real directory.
	descriptor := sim_descriptor(state)
	state.Files[descriptor] = node
	state.Raw_Open[descriptor] = true
	return descriptor, nil
}

// Creates or truncates the file at path against state.Root and binds a fresh descriptor to
// it. The parent directory must already exist, matching a real create.
func sim_create(state *Sim, path string) (file File, err error) {
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
func sim_create_file(root *Sim_Node, path string) (node *Sim_Node, err error) {
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
	created := &Sim_Node{Contents: []byte{}}
	parent.Children[leaf] = created
	return created, nil
}

// Submits a file read that, when it fires, copies the node's bytes from offset into the
// buffer and reports the count — so a read reflects whatever an earlier write stored.
func sim_file_read(
	state *Sim, completion *Completion, callback Callback, node *Sim_Node,
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
	state *Sim, completion *Completion, callback Callback, node *Sim_Node,
	buffer []byte, offset int64,
) {
	sim_submit(state, completion, sim_latency(state), func() {
		sim_node_write(node, buffer, offset)
		callback(completion, len(buffer), nil)
	})
}

// Stores buffer into node's contents at offset, growing the backing bytes when the write
// extends past the current end.
func sim_node_write(node *Sim_Node, buffer []byte, offset int64) {
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
const SIM_SIZE_P50 = 4

// SIM_SIZE_P75 preserves the deterministic filesystem size distribution.
const SIM_SIZE_P75 = 16

// SIM_SIZE_P95 preserves the deterministic filesystem size distribution.
const SIM_SIZE_P95 = 64

// SIM_SIZE_P99 preserves the deterministic filesystem size distribution.
const SIM_SIZE_P99 = 256

// SIM_SIZE_P100 preserves the deterministic filesystem size distribution.
const SIM_SIZE_P100 = 1024

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
func sim_generate(generator *prng.Generator) (root *Sim_Node) {
	sizes := prng.Percentile_Distribution(&prng.Percentile_Distribution_Input{
		P25:  0,
		P50:  SIM_SIZE_P50,
		P75:  SIM_SIZE_P75,
		P95:  SIM_SIZE_P95,
		P99:  SIM_SIZE_P99,
		P100: SIM_SIZE_P100,
	})
	root = sim_new_directory()
	directories := []*Sim_Node{root}
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
			directory.Children[name] = &Sim_Node{Contents: contents}
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
func sim_latency(state *Sim) (latency time.Duration) {
	return time.Duration(prng.Generator_Below(&state.Generator, SIM_LATENCY_GRAINS))
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
func sim_bytes(state *Sim, completion *Completion, callback Callback, buffer []byte) {
	sim_submit(state, completion, sim_latency(state),
		sim_deliver_bytes(completion, callback, len(buffer)))
}

// Schedules callback to receive the next connected synthetic descriptor after drawn latency.
func sim_yield_socket(
	state *Sim, completion *Completion, callback Socket_Callback, listener File,
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
	state *Sim, completion *Completion, callback Signal_Callback, signal Signal,
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
	state.Operations[completion] = SIM_OPERATION_SIGNAL
}

// Builds the driver over state — the loop-advancing capability, held only by main or a
// test, never by code that merely submits IO.
func sim_to_driver(state *Sim) (driver Driver) {
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
func sim_introspect(state *Sim) (counts Loop_Counts) {
	for _, operation := range state.Operations {
		if operation == SIM_OPERATION_COMPLETED {
			counts.Completed++
		}
		if operation == SIM_OPERATION_TIMEOUT {
			counts.Timeouts++
		}
		if operation == SIM_OPERATION_READ_WAITER {
			counts.IO_Backlog++
		}
		if operation == SIM_OPERATION_WRITE_WAITER {
			counts.IO_Backlog++
		}
		if operation == SIM_OPERATION_SIGNAL {
			counts.Signal_Waiters++
		}
		if operation == SIM_OPERATION_SPAWN {
			counts.Spawns++
		}
	}
	counts.Raw_Open = len(state.Raw_Open)
	return counts
}

// Runs pump as the top-level drive, asserting no drive is already in progress so a Run*
// called from within a completion callback panics instead of re-entering the driver. The
// internal per-tick functions call one another directly, not through here, so nested
// ticking within one drive does not trip it.
func sim_drive(state *Sim, pump func()) {
	invariant.Always(!state.Drive_Active,
		"A drive begins at top level, never from within a completion callback.")
	state.Drive_Active = true
	defer func() { state.Drive_Active = false }()
	pump()
}

// Returns the current virtual Moment; the sim never reads the operating-system time.
func sim_now(state *Sim) (now time.Moment) {
	return state.Clock.Now_Monotonic()
}

// Schedules completion to fire at now plus latency and inserts it in Ready_At order.
// Asserts the completion is its own original (not a by-value copy), then arms it through
// the lifecycle machine — a reused in-flight completion panics as the armed-to-armed
// edge.
func sim_submit(state *Sim, completion *Completion, latency time.Duration, callback func()) {
	sim_arm(state, completion, callback)
	completion.Ready_At = sim_now(state) + time.Moment(latency)
	sim_enqueue(state, completion)
}

// Arms completion without placing it on the ready-time queue, for TigerBeetle Event listeners.
func sim_arm(state *Sim, completion *Completion, callback func()) {
	original := completion.Self == nil || completion.Self == completion
	invariant.Always(original,
		"A submitted completion is its own original, never a by-value copy.")
	completion.Self = completion
	Completion_Transition(&Completion_Transition_Input{
		Completion: completion, From: COMPLETION_IDLE, To: COMPLETION_ARMED,
	})
	completion.Callback = callback
	completion.Next_Tick = false
	state.Operations[completion] = SIM_OPERATION_COMPLETED
}

// Inserts completion into the queue in Ready_At order, earliest first.
func sim_enqueue(state *Sim, completion *Completion) {
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
func sim_reset_next_tick(state *Sim, source Next_Tick_Source) {
	kept := state.Queue[:0]
	for _, completion := range state.Queue {
		operation := state.Operations[completion]
		if operation != SIM_OPERATION_NEXT_TICK {
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
func sim_step(state *Sim) (advanced bool) {
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
func sim_drain(state *Sim) {
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
func sim_run(state *Sim) {
	sim_drain(state)
	state.Tick()
}

// Drives the loop until the duration has elapsed, delivering completions as due.
func sim_run_for(state *Sim, duration time.Duration) {
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
	state *Sim, done func() (finished bool), timeout time.Duration,
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

// Stream is a byte transport behind one procedure — a port of Odin core:io's Stream
// (core/io/stream.odin). One procedure pointer covers ten modes, so the value stays two words
// however many modes exist, and a caller names one type instead of ten vtable slots.
//
// MEMORY ONLY. A Stream moves bytes that are already in this process. It never waits on the
// world, and no Stream may ever be built over a file, a socket, a pipe, or a subprocess.
//
// This is not a limitation of the port; it is the difference between the two languages. Odin
// has os.stream_from_handle because core:os makes a blocking syscall, and Odin's timeline is
// whatever the kernel decides. This repository made the timeline an injected dependency: a file
// read is IO.Read with a Completion, ordered by the driver the composition root holds. A
// synchronous file read below that root would advance nothing, order nothing, and could not be
// faulted by the simulator — README.md section 2 states why that is an architectural bug and
// not a style preference. There is therefore nothing to build a file stream out of, because
// IO.Read and IO.Write are the only reads and writes there are, and both are asynchronous.
//
// The rule for anyone adding an implementation: if it waits on anything outside this process,
// it is not a Stream — it is an IO operation, and it belongs on the IO vtable with a Completion.
// If it only transforms bytes on their way through, it is a Stream.
//
// TigerBeetle draws the same boundary, and draws it twice. Its byte path erases nothing:
// third-party/tigerbeetle/src/storage.zig:12 is StorageType(comptime IO: type), monomorphized
// per backend, and its surface is read_sectors and write_sectors (storage.zig:215,386), each
// taking a caller-owned completion struct that holds an IO.Completion, the buffer, the offset,
// and the callback (storage.zig:22-40). No mode enum, no data pointer — which is what IO here
// already is. Its type-erased two-word writer, std.io.AnyWriter, appears only in inspect.zig,
// benchmark_load.zig, trace.zig:164, and snaptest.zig:340: diagnostics, dumps, and text. So a
// type-erased byte sink is for bytes whose order nobody replays, and the path that must stay
// deterministic gets static dispatch and explicit completions. This Stream is the first kind.
//
// That boundary is what the type is for. Its implementations are transforms, not destinations:
// Stream_Memory stores, Stream_Discard absorbs, Stream_Limit truncates, Stream_Count tallies,
// Stream_Tee forks. An encoder written once against Stream works against every one of them, and
// against every transform added later, none of which can smuggle a syscall in behind it.
type Stream struct {
	// Procedure runs every mode. A zero Stream has none, and reports Stream_Empty.
	Procedure Stream_Procedure
	// Data is the stream's own state. Its own Procedure asserts the concrete type.
	Data any
}

// Stream_Procedure runs one mode — Odin's Stream_Proc. One signature serves every mode, so
// count carries the byte count, the new position, the size, or the mode set, as the mode
// dictates, and the arguments a mode does not use are ignored.
type Stream_Procedure func(
	data any, mode Stream_Mode, buffer []byte, offset int64, whence Seek_From,
) (count int64, err error)

// Stream_Mode selects the operation a Stream_Procedure performs.
type Stream_Mode int

// STREAM_MODE_CLOSE ends the stream. It is idempotent.
const STREAM_MODE_CLOSE Stream_Mode = 0

// STREAM_MODE_FLUSH pushes buffered bytes to whatever is behind the stream.
const STREAM_MODE_FLUSH Stream_Mode = 1

// STREAM_MODE_READ reads at the cursor and advances it.
const STREAM_MODE_READ Stream_Mode = 2

// STREAM_MODE_READ_AT reads at an explicit offset and leaves the cursor.
const STREAM_MODE_READ_AT Stream_Mode = 3

// STREAM_MODE_WRITE writes at the cursor and advances it.
const STREAM_MODE_WRITE Stream_Mode = 4

// STREAM_MODE_WRITE_AT writes at an explicit offset and leaves the cursor.
const STREAM_MODE_WRITE_AT Stream_Mode = 5

// STREAM_MODE_SEEK moves the cursor.
const STREAM_MODE_SEEK Stream_Mode = 6

// STREAM_MODE_SIZE reports the whole size, not the bytes remaining.
const STREAM_MODE_SIZE Stream_Mode = 7

// STREAM_MODE_DESTROY releases what the stream owns. Odin separates it from Close because a
// stream there can hold an allocation.
const STREAM_MODE_DESTROY Stream_Mode = 8

// STREAM_MODE_QUERY reports the mode set, so a caller learns what a stream cannot do without
// provoking a failure.
const STREAM_MODE_QUERY Stream_Mode = 9

// Stream_Mode_Set names a set of modes, one bit per mode — Odin's bit_set.
type Stream_Mode_Set uint64

// Mode_Set_Add returns the set with mode added.
func Mode_Set_Add(modes Stream_Mode_Set, mode Stream_Mode) (extended Stream_Mode_Set) {
	return modes | 1<<uint(mode)
}

// Mode_Set_Has reports whether modes contains mode.
func Mode_Set_Has(modes Stream_Mode_Set, mode Stream_Mode) (present bool) {
	return modes&(1<<uint(mode)) != 0
}

// Seek_From selects the origin a seek offset is measured from.
type Seek_From int

// SEEK_FROM_START measures from the first byte.
const SEEK_FROM_START Seek_From = 0

// SEEK_FROM_CURRENT measures from the cursor.
const SEEK_FROM_CURRENT Seek_From = 1

// SEEK_FROM_END measures from one past the last byte.
const SEEK_FROM_END Seek_From = 2

// Stream_EOF reports that a read found nothing left. It is deliberately distinct from the
// standard library io.EOF: a Stream is not a stdlib reader, and one value must not stand for
// both contracts.
var Stream_EOF = errors.New("io: stream end of file")

// Stream_Unexpected_EOF reports an end reached partway through a value that needed more bytes.
var Stream_Unexpected_EOF = errors.New("io: stream unexpected end of file")

// Stream_Short_Write reports that a write stored fewer bytes than it was given.
var Stream_Short_Write = errors.New("io: stream short write")

// Stream_Invalid_Write reports a procedure that claimed more bytes than it was given.
var Stream_Invalid_Write = errors.New("io: stream invalid write")

// Stream_Short_Buffer reports a buffer too small to hold what the stream produced.
var Stream_Short_Buffer = errors.New("io: stream short buffer")

// Stream_No_Progress reports a read or write that moved no bytes and reported no reason.
var Stream_No_Progress = errors.New("io: stream made no progress")

// Stream_Invalid_Whence reports a seek origin outside the three Seek_From values.
var Stream_Invalid_Whence = errors.New("io: stream invalid whence")

// Stream_Invalid_Offset reports an offset outside the stream.
var Stream_Invalid_Offset = errors.New("io: stream invalid offset")

// Stream_Invalid_Unread reports an unread of a byte the stream did not read.
var Stream_Invalid_Unread = errors.New("io: stream invalid unread")

// Stream_Negative_Read reports a procedure that returned a negative read count.
var Stream_Negative_Read = errors.New("io: stream negative read")

// Stream_Negative_Write reports a procedure that returned a negative write count.
var Stream_Negative_Write = errors.New("io: stream negative write")

// Stream_Negative_Count reports a negative count where only a positive one is meaningful.
var Stream_Negative_Count = errors.New("io: stream negative count")

// Stream_Buffer_Full reports a buffer that cannot accept another byte.
var Stream_Buffer_Full = errors.New("io: stream buffer full")

// Stream_Unknown reports a failure the stream cannot name.
var Stream_Unknown = errors.New("io: stream unknown error")

// Stream_Empty reports a zero Stream, a mode the stream does not answer, and a stream that has
// been closed. All three mean the same thing to a caller: there is nothing there to do this.
var Stream_Empty = errors.New("io: stream empty")

// Read reads into buffer at the cursor and advances the cursor.
func Read(stream Stream, buffer []byte) (count int64, err error) {
	if stream.Procedure == nil {
		return 0, Stream_Empty
	}
	count, err = stream.Procedure(stream.Data, STREAM_MODE_READ, buffer, 0, SEEK_FROM_START)
	return stream_read_checked(count, err, buffer)
}

// Read_At reads into buffer at offset and leaves the cursor where it was.
func Read_At(stream Stream, buffer []byte, offset int64) (count int64, err error) {
	if stream.Procedure == nil {
		return 0, Stream_Empty
	}
	count, err = stream.Procedure(
		stream.Data, STREAM_MODE_READ_AT, buffer, offset, SEEK_FROM_START,
	)
	return stream_read_checked(count, err, buffer)
}

// Write writes buffer at the cursor and advances the cursor.
func Write(stream Stream, buffer []byte) (count int64, err error) {
	if stream.Procedure == nil {
		return 0, Stream_Empty
	}
	count, err = stream.Procedure(stream.Data, STREAM_MODE_WRITE, buffer, 0, SEEK_FROM_START)
	return stream_write_checked(count, err, buffer)
}

// Write_At writes buffer at offset and leaves the cursor where it was.
func Write_At(stream Stream, buffer []byte, offset int64) (count int64, err error) {
	if stream.Procedure == nil {
		return 0, Stream_Empty
	}
	count, err = stream.Procedure(
		stream.Data, STREAM_MODE_WRITE_AT, buffer, offset, SEEK_FROM_START,
	)
	return stream_write_checked(count, err, buffer)
}

// Seek moves the cursor to offset measured from whence and reports the new position.
func Seek(stream Stream, offset int64, whence Seek_From) (position int64, err error) {
	if stream.Procedure == nil {
		return 0, Stream_Empty
	}
	return stream.Procedure(stream.Data, STREAM_MODE_SEEK, nil, offset, whence)
}

// Size reports the whole size of the stream, not the bytes remaining after the cursor.
func Size(stream Stream) (size int64, err error) {
	if stream.Procedure == nil {
		return 0, Stream_Empty
	}
	return stream.Procedure(stream.Data, STREAM_MODE_SIZE, nil, 0, SEEK_FROM_START)
}

// Flush pushes any buffered bytes onward.
func Flush(stream Stream) (err error) {
	if stream.Procedure == nil {
		return Stream_Empty
	}
	_, flush_err := stream.Procedure(stream.Data, STREAM_MODE_FLUSH, nil, 0, SEEK_FROM_START)
	return flush_err
}

// Close ends the stream. It is idempotent, and a closed stream answers no data mode.
func Close(stream Stream) (err error) {
	if stream.Procedure == nil {
		return Stream_Empty
	}
	_, close_err := stream.Procedure(stream.Data, STREAM_MODE_CLOSE, nil, 0, SEEK_FROM_START)
	return close_err
}

// Destroy releases what the stream owns. A memory stream owns nothing but its cursor, so this
// is a close; the mode exists because Odin streams can hold an allocation.
func Destroy(stream Stream) (err error) {
	if stream.Procedure == nil {
		return Stream_Empty
	}
	_, destroy_err := stream.Procedure(
		stream.Data, STREAM_MODE_DESTROY, nil, 0, SEEK_FROM_START,
	)
	return destroy_err
}

// Query reports every mode the stream answers. A zero Stream answers none.
func Query(stream Stream) (modes Stream_Mode_Set) {
	if stream.Procedure == nil {
		return 0
	}
	count, err := stream.Procedure(stream.Data, STREAM_MODE_QUERY, nil, 0, SEEK_FROM_START)
	if err != nil {
		return 0
	}
	return Stream_Mode_Set(count)
}

// Applies Odin's read checks to a procedure result, so one wrong procedure cannot corrupt every
// caller that trusted its count.
func stream_read_checked(
	count int64, err error, buffer []byte,
) (checked int64, checked_err error) {
	if count < 0 {
		return 0, Stream_Negative_Read
	}
	if count > int64(len(buffer)) {
		return 0, Stream_Short_Buffer
	}
	return count, err
}

// Applies Odin's write checks: a count past the buffer is the procedure lying, and a short
// count with no error is the caller's silent data loss, so it becomes an error here.
func stream_write_checked(
	count int64, err error, buffer []byte,
) (checked int64, checked_err error) {
	if count < 0 {
		return 0, Stream_Negative_Write
	}
	if count > int64(len(buffer)) {
		return 0, Stream_Invalid_Write
	}
	if err != nil {
		return count, err
	}
	if count < int64(len(buffer)) {
		return count, Stream_Short_Write
	}
	return count, nil
}

// STREAM_MEMORY_MODES names every mode a memory stream answers, which is all ten: memory is
// the only transport that can honestly answer them all, because it is the only one that stores.
const STREAM_MEMORY_MODES Stream_Mode_Set = 1<<STREAM_MODE_CLOSE |
	1<<STREAM_MODE_FLUSH | 1<<STREAM_MODE_READ | 1<<STREAM_MODE_READ_AT |
	1<<STREAM_MODE_WRITE | 1<<STREAM_MODE_WRITE_AT | 1<<STREAM_MODE_SEEK |
	1<<STREAM_MODE_SIZE | 1<<STREAM_MODE_DESTROY | 1<<STREAM_MODE_QUERY

// STREAM_DISCARD_MODES names the modes a discard stream answers. It cannot seek or report a
// size, because it stores nothing to seek within.
const STREAM_DISCARD_MODES Stream_Mode_Set = 1<<STREAM_MODE_CLOSE |
	1<<STREAM_MODE_FLUSH | 1<<STREAM_MODE_WRITE | 1<<STREAM_MODE_WRITE_AT |
	1<<STREAM_MODE_DESTROY | 1<<STREAM_MODE_QUERY

// Stream_Memory is the state behind a memory stream. The caller owns it and passes its address
// to Memory_To_Stream, as Odin's bytes.buffer_to_stream takes a caller-owned Buffer: the stream
// allocates nothing, so a caller can see every byte the transport will ever hold.
type Stream_Memory struct {
	// Memory is the caller's slice. Its length is the stream's whole budget, and the stream
	// never grows it, so an encoder that writes without asking cannot make this allocate.
	Memory []byte
	// Cursor is where Read and Write act. Read_At and Write_At do not move it.
	Cursor int64
	// Closed reports that Close or Destroy ran. A closed stream answers no data mode.
	Closed bool
}

// Memory_To_Stream returns a Stream over the caller's memory. It answers every mode, because
// memory is the only transport that stores, and storing is what Seek and Size need.
//
// A write that reaches the end of the slice stores what fits and reports Stream_Short_Write.
// That is what makes this safe to hand to an encoder that does not know how much it is about to
// produce: the budget is stated once, at the slice, and the stream cannot exceed it.
func Memory_To_Stream(state *Stream_Memory) (stream Stream) {
	invariant.Always(state != nil, "A memory stream has state.")
	return Stream{Procedure: stream_memory_procedure, Data: state}
}

// Runs one mode against memory. Lifecycle first, so a closed stream answers no data mode.
func stream_memory_procedure(
	data any, mode Stream_Mode, buffer []byte, offset int64, whence Seek_From,
) (count int64, err error) {
	state, held := data.(*Stream_Memory)
	invariant.Always(held, "A memory stream procedure receives its own state.")
	if mode == STREAM_MODE_QUERY {
		return int64(STREAM_MEMORY_MODES), nil
	}
	if mode == STREAM_MODE_CLOSE {
		state.Closed = true
		return 0, nil
	}
	if mode == STREAM_MODE_DESTROY {
		state.Closed = true
		return 0, nil
	}
	if state.Closed {
		return 0, Stream_Empty
	}
	return stream_memory_data(state, mode, buffer, offset, whence)
}

// Runs the modes that touch the bytes, on a stream known to be open.
func stream_memory_data(
	state *Stream_Memory, mode Stream_Mode, buffer []byte, offset int64, whence Seek_From,
) (count int64, err error) {
	if mode == STREAM_MODE_FLUSH {
		return 0, nil
	}
	if mode == STREAM_MODE_SIZE {
		return int64(len(state.Memory)), nil
	}
	if mode == STREAM_MODE_SEEK {
		return stream_memory_seek(state, offset, whence)
	}
	if mode == STREAM_MODE_READ {
		moved, read_err := stream_memory_read(state, buffer, state.Cursor)
		state.Cursor = state.Cursor + moved
		return moved, read_err
	}
	if mode == STREAM_MODE_READ_AT {
		return stream_memory_read(state, buffer, offset)
	}
	if mode == STREAM_MODE_WRITE {
		moved, write_err := stream_memory_write(state, buffer, state.Cursor)
		state.Cursor = state.Cursor + moved
		return moved, write_err
	}
	if mode == STREAM_MODE_WRITE_AT {
		return stream_memory_write(state, buffer, offset)
	}
	return 0, Stream_Empty
}

// Copies out of the memory at offset. An offset at the end is the end of the data, not a fault.
func stream_memory_read(
	state *Stream_Memory, buffer []byte, offset int64,
) (count int64, err error) {
	if offset < 0 {
		return 0, Stream_Invalid_Offset
	}
	if offset > int64(len(state.Memory)) {
		return 0, Stream_Invalid_Offset
	}
	if offset == int64(len(state.Memory)) {
		return 0, Stream_EOF
	}
	return int64(copy(buffer, state.Memory[offset:])), nil
}

// Copies into the memory at offset. What does not fit is reported, never grown into.
func stream_memory_write(
	state *Stream_Memory, buffer []byte, offset int64,
) (count int64, err error) {
	if offset < 0 {
		return 0, Stream_Invalid_Offset
	}
	if offset > int64(len(state.Memory)) {
		return 0, Stream_Invalid_Offset
	}
	stored := int64(copy(state.Memory[offset:], buffer))
	if stored < int64(len(buffer)) {
		return stored, Stream_Short_Write
	}
	return stored, nil
}

// Moves the cursor. A position outside the memory is rejected rather than clamped: a stream
// that silently moves a cursor somewhere else hides the caller's arithmetic error.
func stream_memory_seek(
	state *Stream_Memory, offset int64, whence Seek_From,
) (position int64, err error) {
	target := offset
	if whence == SEEK_FROM_CURRENT {
		target = state.Cursor + offset
	}
	if whence == SEEK_FROM_END {
		target = int64(len(state.Memory)) + offset
	}
	if whence > SEEK_FROM_END {
		return 0, Stream_Invalid_Whence
	}
	if whence < SEEK_FROM_START {
		return 0, Stream_Invalid_Whence
	}
	if target < 0 {
		return 0, Stream_Invalid_Offset
	}
	if target > int64(len(state.Memory)) {
		return 0, Stream_Invalid_Offset
	}
	state.Cursor = target
	return target, nil
}

// Stream_Discard is the state behind a discard stream. It keeps nothing, so only the closed
// flag distinguishes one discard stream from another.
type Stream_Discard struct {
	// Closed reports that Close or Destroy ran.
	Closed bool
}

// Discard_To_Stream returns a Stream that absorbs every write and reports the whole buffer
// written. It is the sink an encoder writes to when the caller wants the size or the side
// effects, not the bytes. It answers no read mode: there is nothing there to read.
func Discard_To_Stream(state *Stream_Discard) (stream Stream) {
	invariant.Always(state != nil, "A discard stream has state.")
	return Stream{Procedure: stream_discard_procedure, Data: state}
}

// Runs one mode against nothing.
func stream_discard_procedure(
	data any, mode Stream_Mode, buffer []byte, offset int64, whence Seek_From,
) (count int64, err error) {
	state, held := data.(*Stream_Discard)
	invariant.Always(held, "A discard stream procedure receives its own state.")
	if mode == STREAM_MODE_QUERY {
		return int64(STREAM_DISCARD_MODES), nil
	}
	if mode == STREAM_MODE_CLOSE {
		state.Closed = true
		return 0, nil
	}
	if mode == STREAM_MODE_DESTROY {
		state.Closed = true
		return 0, nil
	}
	if state.Closed {
		return 0, Stream_Empty
	}
	if mode == STREAM_MODE_FLUSH {
		return 0, nil
	}
	if mode == STREAM_MODE_WRITE {
		return int64(len(buffer)), nil
	}
	if mode == STREAM_MODE_WRITE_AT {
		return int64(len(buffer)), nil
	}
	return 0, Stream_Empty
}

// STREAM_LIMIT_MODES names the modes a limit forwards. It drops Read_At, Write_At, Seek, and
// Size deliberately: a budget counts bytes as they pass, and an operation at an explicit offset
// passes no bytes through the budget, so the two ideas cannot both be honest at once.
const STREAM_LIMIT_MODES Stream_Mode_Set = 1<<STREAM_MODE_CLOSE |
	1<<STREAM_MODE_FLUSH | 1<<STREAM_MODE_READ | 1<<STREAM_MODE_WRITE |
	1<<STREAM_MODE_DESTROY | 1<<STREAM_MODE_QUERY

// STREAM_TEE_MODES names the modes a tee forwards. It cannot read, seek, or report a size,
// because two streams would give two answers and a tee has no way to choose between them.
const STREAM_TEE_MODES Stream_Mode_Set = 1<<STREAM_MODE_CLOSE |
	1<<STREAM_MODE_FLUSH | 1<<STREAM_MODE_WRITE | 1<<STREAM_MODE_DESTROY |
	1<<STREAM_MODE_QUERY

// Stream_Limit is the state behind a limit. It narrows a stream that is already bounded, which
// is not how a stream becomes bounded: every constructor states its own budget.
type Stream_Limit struct {
	// Inner is the stream the bytes pass through to.
	Inner Stream
	// Budget is how many bytes may still pass. It falls as they do.
	Budget int64
}

// Limit_To_Stream returns a Stream that passes bytes to Inner until Budget runs out, then
// reports Stream_EOF on reads and Stream_Short_Write on writes.
//
// The bytes past the budget never reach Inner. That is the difference between a limit and a
// caller who counts: a caller who counts can forget, and a limit cannot.
func Limit_To_Stream(state *Stream_Limit) (stream Stream) {
	invariant.Always(state != nil, "A limit stream has state.")
	return Stream{Procedure: stream_limit_procedure, Data: state}
}

// Runs one mode against the budget, then against the stream behind it.
func stream_limit_procedure(
	data any, mode Stream_Mode, buffer []byte, offset int64, whence Seek_From,
) (count int64, err error) {
	state, held := data.(*Stream_Limit)
	invariant.Always(held, "A limit stream procedure receives its own state.")
	if mode == STREAM_MODE_QUERY {
		return int64(Query(state.Inner) & STREAM_LIMIT_MODES), nil
	}
	if mode == STREAM_MODE_CLOSE {
		return 0, Close(state.Inner)
	}
	if mode == STREAM_MODE_DESTROY {
		return 0, Destroy(state.Inner)
	}
	if mode == STREAM_MODE_FLUSH {
		return 0, Flush(state.Inner)
	}
	if mode == STREAM_MODE_READ {
		return stream_limit_read(state, buffer)
	}
	if mode == STREAM_MODE_WRITE {
		return stream_limit_write(state, buffer)
	}
	return 0, Stream_Empty
}

// Reads no more than the budget allows, and spends what it read.
func stream_limit_read(state *Stream_Limit, buffer []byte) (count int64, err error) {
	if state.Budget <= 0 {
		return 0, Stream_EOF
	}
	allowed := int64(len(buffer))
	if state.Budget < allowed {
		allowed = state.Budget
	}
	moved, read_err := Read(state.Inner, buffer[:allowed])
	state.Budget = state.Budget - moved
	return moved, read_err
}

// Writes no more than the budget allows, and reports the truncation the caller cannot see.
func stream_limit_write(state *Stream_Limit, buffer []byte) (count int64, err error) {
	if state.Budget <= 0 {
		return 0, Stream_Short_Write
	}
	allowed := int64(len(buffer))
	if state.Budget < allowed {
		allowed = state.Budget
	}
	moved, write_err := Write(state.Inner, buffer[:allowed])
	state.Budget = state.Budget - moved
	if write_err != nil {
		return moved, write_err
	}
	if allowed < int64(len(buffer)) {
		return moved, Stream_Short_Write
	}
	return moved, nil
}

// Stream_Count is the state behind a count. Tally is the caller's: the stream adds to it and
// never reads it, so the caller decides when a measurement starts and stops.
type Stream_Count struct {
	// Inner is the stream the bytes pass through to.
	Inner Stream
	// Tally is every byte that has passed, read and written alike.
	Tally int64
}

// Count_To_Stream returns a Stream that forwards every mode to Inner and adds each byte to
// Tally. Over a discard stream it measures what an encoder would produce; over memory it
// measures what an encoder did produce. The encoder cannot tell the two apart, which is the
// point of measuring this way rather than running the encoder twice.
func Count_To_Stream(state *Stream_Count) (stream Stream) {
	invariant.Always(state != nil, "A count stream has state.")
	return Stream{Procedure: stream_count_procedure, Data: state}
}

// Runs one mode against the stream behind it and tallies what moved.
func stream_count_procedure(
	data any, mode Stream_Mode, buffer []byte, offset int64, whence Seek_From,
) (count int64, err error) {
	state, held := data.(*Stream_Count)
	invariant.Always(held, "A count stream procedure receives its own state.")
	if mode == STREAM_MODE_QUERY {
		return int64(Query(state.Inner)), nil
	}
	if mode == STREAM_MODE_CLOSE {
		return 0, Close(state.Inner)
	}
	if mode == STREAM_MODE_DESTROY {
		return 0, Destroy(state.Inner)
	}
	if mode == STREAM_MODE_FLUSH {
		return 0, Flush(state.Inner)
	}
	if mode == STREAM_MODE_SEEK {
		return Seek(state.Inner, offset, whence)
	}
	if mode == STREAM_MODE_SIZE {
		return Size(state.Inner)
	}
	return stream_count_moved(state, mode, buffer, offset)
}

// Runs the modes that move bytes and adds each one to the tally.
func stream_count_moved(
	state *Stream_Count, mode Stream_Mode, buffer []byte, offset int64,
) (count int64, err error) {
	if mode == STREAM_MODE_READ {
		moved, read_err := Read(state.Inner, buffer)
		state.Tally = state.Tally + moved
		return moved, read_err
	}
	if mode == STREAM_MODE_READ_AT {
		moved, read_err := Read_At(state.Inner, buffer, offset)
		state.Tally = state.Tally + moved
		return moved, read_err
	}
	if mode == STREAM_MODE_WRITE {
		moved, write_err := Write(state.Inner, buffer)
		state.Tally = state.Tally + moved
		return moved, write_err
	}
	if mode == STREAM_MODE_WRITE_AT {
		moved, write_err := Write_At(state.Inner, buffer, offset)
		state.Tally = state.Tally + moved
		return moved, write_err
	}
	return 0, Stream_Empty
}

// Stream_Tee is the state behind a tee: one write, two destinations.
type Stream_Tee struct {
	// First receives every buffer.
	First Stream
	// Second receives every buffer, after First.
	Second Stream
}

// Tee_To_Stream returns a Stream that writes each buffer to both streams and reports the
// smaller count. It reports the smaller one because the caller must learn about the tighter of
// the two destinations: a tee that reported the wider count would hide the loss on the other.
func Tee_To_Stream(state *Stream_Tee) (stream Stream) {
	invariant.Always(state != nil, "A tee stream has state.")
	return Stream{Procedure: stream_tee_procedure, Data: state}
}

// Runs one mode against both streams.
func stream_tee_procedure(
	data any, mode Stream_Mode, buffer []byte, offset int64, whence Seek_From,
) (count int64, err error) {
	state, held := data.(*Stream_Tee)
	invariant.Always(held, "A tee stream procedure receives its own state.")
	if mode == STREAM_MODE_QUERY {
		return int64(STREAM_TEE_MODES), nil
	}
	if mode == STREAM_MODE_CLOSE {
		return 0, stream_tee_both(Close(state.First), Close(state.Second))
	}
	if mode == STREAM_MODE_DESTROY {
		return 0, stream_tee_both(Destroy(state.First), Destroy(state.Second))
	}
	if mode == STREAM_MODE_FLUSH {
		return 0, stream_tee_both(Flush(state.First), Flush(state.Second))
	}
	if mode == STREAM_MODE_WRITE {
		return stream_tee_write(state, buffer)
	}
	return 0, Stream_Empty
}

// Reports the first failure of the two, so neither destination can fail in silence.
func stream_tee_both(first error, second error) (err error) {
	if first != nil {
		return first
	}
	return second
}

// Writes the buffer to both and reports the smaller count.
func stream_tee_write(state *Stream_Tee, buffer []byte) (count int64, err error) {
	first_count, first_err := Write(state.First, buffer)
	second_count, second_err := Write(state.Second, buffer)
	smaller := first_count
	if second_count < smaller {
		smaller = second_count
	}
	if first_err != nil {
		return smaller, first_err
	}
	if second_err != nil {
		return smaller, second_err
	}
	return smaller, nil
}

// STREAM_BYTE_SIZE is the buffer one byte needs, the unit Read_Byte and Write_Byte move.
const STREAM_BYTE_SIZE = 1

// Read_At_Least reads until it has at least minimum bytes, or until the stream stops. It is
// Odin's read_at_least, written as the loop Odin writes: a synchronous stream needs no
// continuation to read a second time.
//
// An end reached after some bytes is Stream_Unexpected_EOF, not Stream_EOF: the caller asked
// for a quantity and got part of one, which is a different fact from finding nothing at all.
func Read_At_Least(stream Stream, buffer []byte, minimum int64) (count int64, err error) {
	if minimum < 0 {
		return 0, Stream_Negative_Count
	}
	if int64(len(buffer)) < minimum {
		return 0, Stream_Short_Buffer
	}
	for count < minimum {
		if err != nil {
			break
		}
		moved, read_err := Read(stream, buffer[count:])
		count = count + moved
		err = read_err
	}
	if count >= minimum {
		return count, nil
	}
	if err == Stream_EOF {
		if count > 0 {
			return count, Stream_Unexpected_EOF
		}
	}
	return count, err
}

// Make_Directory_Input names one parent-creating directory create.
type Make_Directory_Input struct {
	// Loop supplies the Mkdir_At primitive each component uses.
	Loop IO
	// Completion is the caller-owned completion the result retires on.
	Completion *Completion
	// Callback runs once, after the last component or after the first real failure.
	Callback Timeout_Callback
	// Path is the directory to create, including every missing parent.
	Path string
	// Mode is the permission mode each created component takes.
	Mode uint32
}

// Make_Directory creates Path and every missing parent. It is a derived function, not a member
// of IO: it walks the path in pure code and submits one Mkdir_At for each component, so the
// simulated backend and the operating-system backend run this same walk.
//
// An existing component converges rather than failing, because Path_Exists is what mkdirat
// reports for a directory that is already there. Any other error retires the completion at once.
func Make_Directory(input *Make_Directory_Input) {
	invariant.Always(input != nil, "A Make_Directory input is present.")
	invariant.Always(input.Loop.Mkdir_At != nil, "Make_Directory needs the Mkdir_At primitive.")
	state := &Make_Directory_State{Input: input, Bounds: make_directory_bounds(input.Path)}
	// The continuation is a field, not a direct call, so one component's callback advances
	// to the next without the step naming itself. It is the shape setup uses for its own
	// rearms, and it keeps each component a separate submission rather than a nested call.
	state.Continue = func() { make_directory_step(state) }
	state.Continue()
}

// One parent-creating create in progress: the component boundaries and the index reached.
type Make_Directory_State struct {
	// Input holds the caller's loop, completion, callback, path, and mode.
	Input *Make_Directory_Input
	// Bounds are the end offsets of each path component, shortest first.
	Bounds []int
	// Index is the component the next step creates.
	Index int
	// Delivered reports the caller's callback already ran. A backend may retire the same
	// completion more than once, and a derived function must still deliver one result.
	Delivered bool
	// Submitted reports one component create in flight, so a second retirement of the same
	// completion does not submit the next component twice.
	Submitted bool
	// Continue advances to the next component. It is a field so a callback reaches the step
	// without the step naming itself.
	Continue func()
}

// One directory listing in progress: the descriptor, the pass buffer, and the entries so far.
type Read_Directory_State struct {
	// Input holds the caller's loop, completion, callback, and path.
	Input *Read_Directory_Input
	// Directory is the open descriptor every pass reads.
	Directory File
	// Buffer receives one pass of raw entries.
	Buffer []byte
	// Entries accumulates every child the passes reported.
	Entries []Directory_Entry
	// Passes counts the passes taken, so the walk stays bounded.
	Passes int
	// Failure is the error that ended the walk, delivered after the close.
	Failure error
	// Delivered reports the caller's callback already ran.
	Delivered bool
	// Continue takes the next pass. It is a field so a callback reaches the pass without the
	// pass naming itself.
	Continue func()
}

// Delivers the create's one result. A second retirement reaches this and returns.
func make_directory_deliver(
	state *Make_Directory_State, completion *Completion, err error,
) {
	if state.Delivered {
		return
	}
	state.Delivered = true
	state.Input.Callback(completion, err)
}

// Returns the end offset of every path component, so "/a/b" gives 2 and 4. The walk is bounded
// by the path length, and a repeated separator contributes no bound.
func make_directory_bounds(path string) (bounds []int) {
	bounds = []int{}
	for index := 1; index <= len(path); index++ {
		if index < len(path) {
			if path[index] != '/' {
				continue
			}
		}
		if path[index-1] == '/' {
			continue
		}
		bounds = append(bounds, index)
	}
	return bounds
}

// Submits the next component, or retires the caller's completion once every component exists.
func make_directory_step(state *Make_Directory_State) {
	if state.Index == len(state.Bounds) {
		make_directory_deliver(state, state.Input.Completion, nil)
		return
	}
	component := state.Input.Path[:state.Bounds[state.Index]]
	state.Index++
	state.Submitted = true
	state.Input.Loop.Mkdir_At(state.Input.Completion, func(
		completion *Completion, mkdir_err error,
	) {
		// A second retirement of the same completion finds Pending already cleared and
		// submits nothing, so each component is created one time.
		if !state.Submitted {
			return
		}
		state.Submitted = false
		if mkdir_err != nil {
			if mkdir_err != Path_Exists {
				make_directory_deliver(state, completion, mkdir_err)
				return
			}
		}
		state.Continue()
	}, DIRECTORY_CURRENT, component, state.Input.Mode)
}

// Bounds one directory pass, so a large directory is read in repeated passes rather than one
// unbounded allocation.
const DIRECTORY_PASS_BYTES = 8192

// Caps the number of directory passes, so a pathological directory reports an error rather than
// a walk without end.
const DIRECTORY_PASSES_MAX = 4096

// Read_Directory_Input names one directory listing.
type Read_Directory_Input struct {
	// Loop supplies the Open_At, Get_Directory_Entries, and Close primitives.
	Loop IO
	// Completion is the caller-owned completion the result retires on.
	Completion *Completion
	// Callback runs once, with every child or with the error that ended the walk.
	Callback Directory_Callback
	// Path is the directory to list.
	Path string
}

// Read_Directory lists a directory's immediate children. It is a derived function: it opens the
// directory, reads repeated passes until one reports none, and closes the descriptor. Both
// backends run this same sequence, and only the single-pass primitive below it differs.
func Read_Directory(input *Read_Directory_Input) {
	invariant.Always(input != nil, "A Read_Directory input is present.")
	invariant.Always(
		input.Loop.Get_Directory_Entries != nil,
		"Read_Directory needs the Get_Directory_Entries primitive.",
	)
	state := &Read_Directory_State{
		Input:     input,
		Buffer:    make([]byte, DIRECTORY_PASS_BYTES),
		Entries:   []Directory_Entry{},
		Directory: -1,
	}
	state.Continue = func() { read_directory_pass(state) }
	input.Loop.Open_At(input.Completion, func(
		completion *Completion, directory File, open_err error,
	) {
		if open_err != nil {
			read_directory_deliver(state, completion, nil, open_err)
			return
		}
		if state.Delivered {
			return
		}
		if state.Directory >= 0 {
			return
		}
		state.Directory = directory
		state.Continue()
	}, DIRECTORY_CURRENT, input.Path, Open_At_Options{Access: OPEN_READ_ONLY})
}

// Delivers the walk's one result. A second retirement of the same completion reaches this and
// returns without a second callback.
func read_directory_deliver(
	state *Read_Directory_State, completion *Completion,
	entries []Directory_Entry, err error,
) {
	if state.Delivered {
		state.Input.Callback(completion, nil, Retired_Twice)
		return
	}
	state.Delivered = true
	state.Input.Callback(completion, entries, err)
}

// Reads one pass, then either takes another or closes the descriptor and delivers.
func read_directory_pass(state *Read_Directory_State) {
	if state.Passes == DIRECTORY_PASSES_MAX {
		state.Failure = errors.New("io: directory exceeds the maximum entry count")
		read_directory_close(state)
		return
	}
	state.Passes++
	state.Input.Loop.Get_Directory_Entries(state.Input.Completion, func(
		_ *Completion, entries []Directory_Entry, pass_err error,
	) {
		if pass_err != nil {
			state.Failure = pass_err
			read_directory_close(state)
			return
		}
		if len(entries) == 0 {
			read_directory_close(state)
			return
		}
		state.Entries = append(state.Entries, entries...)
		state.Continue()
	}, state.Directory, state.Buffer)
}

// Releases the descriptor, then delivers the listing or the failure that ended it.
func read_directory_close(state *Read_Directory_State) {
	state.Input.Loop.Close(state.Input.Completion, func(
		completion *Completion, close_err error,
	) {
		if state.Delivered {
			read_directory_deliver(state, completion, nil, nil)
			return
		}
		if state.Failure != nil {
			read_directory_deliver(state, completion, nil, state.Failure)
			return
		}
		if close_err != nil {
			read_directory_deliver(state, completion, nil, close_err)
			return
		}
		read_directory_deliver(state, completion, state.Entries, nil)
	}, state.Directory)
}

// Listen binds a caller-owned socket, marks it accepting, and reports the address the kernel
// settled on — the actual port when the caller asked for port zero. It is a derived function
// over the Bind, Listen_Socket, and Get_Socket_Name primitives, so both backends run this same
// sequence (third-party/tigerbeetle/src/io/common.zig:32-59).
func Listen(
	loop IO, socket File, address Address, options Listen_Options,
) (resolved Address, err error) {
	invariant.Always(loop.Bind != nil, "Listen needs the Bind primitive.")
	reuse_err := loop.Set_Socket_Option(socket, SOCKET_OPTION_REUSE_ADDRESS, 1)
	if reuse_err != nil {
		return Address{}, reuse_err
	}
	if bind_err := loop.Bind(socket, address); bind_err != nil {
		return Address{}, bind_err
	}
	if listen_err := loop.Listen_Socket(socket, options.Backlog); listen_err != nil {
		return Address{}, listen_err
	}
	return loop.Get_Socket_Name(socket)
}

// Open_Socket_TCP creates a stream socket and applies the caller's TCP options. It is a derived
// function over the Socket and Set_Socket_Option primitives.
func Open_Socket_TCP(
	loop IO, family Address_Family, options TCP_Options,
) (socket File, err error) {
	invariant.Always(loop.Socket != nil, "Open_Socket_TCP needs the Socket primitive.")
	socket, open_err := loop.Socket(family, SOCKET_TRANSPORT_TCP)
	if open_err != nil {
		return 0, open_err
	}
	settings := []struct {
		Option Socket_Option
		Value  int
		Set    bool
	}{
		{SOCKET_OPTION_RECEIVE_BUFFER, options.Receive_Buffer, options.Receive_Buffer > 0},
		{SOCKET_OPTION_SEND_BUFFER, options.Send_Buffer, options.Send_Buffer > 0},
		{SOCKET_OPTION_KEEPALIVE, 1, options.Keepalive != nil},
		{SOCKET_OPTION_NO_DELAY, 1, options.No_Delay},
	}
	for _, setting := range settings {
		if !setting.Set {
			continue
		}
		set_err := loop.Set_Socket_Option(socket, setting.Option, setting.Value)
		if set_err != nil {
			return 0, set_err
		}
	}
	return socket, nil
}

// Open_Socket_UDP creates a datagram socket. It is a derived function over the Socket primitive.
func Open_Socket_UDP(loop IO, family Address_Family) (socket File, err error) {
	invariant.Always(loop.Socket != nil, "Open_Socket_UDP needs the Socket primitive.")
	return loop.Socket(family, SOCKET_TRANSPORT_UDP)
}

// Read_Full reads until the buffer is full or the stream stops — Odin's read_full.
func Read_Full(stream Stream, buffer []byte) (count int64, err error) {
	return Read_At_Least(stream, buffer, int64(len(buffer)))
}

// Write_String writes the bytes of text.
//
// It copies. Odin transmutes the string, because a Stream there cannot write into what it was
// given; here nothing stops an implementation from doing so, and a stream that wrote into a Go
// string would corrupt a value the language guarantees is immutable.
func Write_String(stream Stream, text string) (count int64, err error) {
	return Write(stream, []byte(text))
}

// Read_Byte reads one byte. An end found here is Stream_Unexpected_EOF, because one byte was
// asked for and none arrived.
func Read_Byte(stream Stream) (value byte, err error) {
	var storage [STREAM_BYTE_SIZE]byte
	count, read_err := Read(stream, storage[:])
	if read_err != nil {
		return 0, read_err
	}
	if count != 1 {
		return 0, Stream_Unexpected_EOF
	}
	return storage[0], nil
}

// Write_Byte writes one byte.
func Write_Byte(stream Stream, value byte) (err error) {
	storage := [STREAM_BYTE_SIZE]byte{value}
	_, write_err := Write(stream, storage[:])
	return write_err
}

// Read_Rune reads one UTF-8 character and reports how many bytes it consumed. The lead byte
// states the length, so the rest is one further read, never a search.
func Read_Rune(stream Stream) (character rune, size int64, err error) {
	var sequence [utf8.UTFMax]byte
	count, read_err := Read(stream, sequence[:STREAM_BYTE_SIZE])
	if read_err != nil {
		return 0, 0, read_err
	}
	if count != 1 {
		return 0, 0, Stream_Unexpected_EOF
	}
	if sequence[0] < utf8.RuneSelf {
		return rune(sequence[0]), 1, nil
	}
	sequence_size := stream_sequence_size(sequence[0])
	if sequence_size == 0 {
		return utf8.RuneError, STREAM_BYTE_SIZE, nil
	}
	_, rest_err := Read_Full(stream, sequence[STREAM_BYTE_SIZE:sequence_size])
	if rest_err != nil {
		return 0, 0, rest_err
	}
	decoded, decoded_size := utf8.DecodeRune(sequence[:sequence_size])
	return decoded, int64(decoded_size), nil
}

// Reports how many bytes a UTF-8 sequence holds, from its lead byte. Zero rejects a lead byte
// that starts no sequence, which is a continuation byte or an invalid one.
func stream_sequence_size(lead byte) (size int64) {
	if lead < 0xC0 {
		return 0
	}
	if lead < 0xE0 {
		return 2
	}
	if lead < 0xF0 {
		return 3
	}
	if lead < 0xF8 {
		return 4
	}
	return 0
}

// Write_Rune writes one UTF-8 character and reports how many bytes it wrote.
func Write_Rune(stream Stream, character rune) (size int64, err error) {
	var sequence [utf8.UTFMax]byte
	sequence_size := utf8.EncodeRune(sequence[:], character)
	return Write(stream, sequence[:sequence_size])
}

// Read_Pointer reads size bytes into the memory at pointer — Odin's read_ptr.
//
// The caller states the size, so this cannot check it. It is the one place a Stream trusts a
// caller with a length, and the only reason this package names unsafe.
func Read_Pointer(stream Stream, pointer unsafe.Pointer, size int64) (count int64, err error) {
	if size < 0 {
		return 0, Stream_Negative_Count
	}
	return Read(stream, unsafe.Slice((*byte)(pointer), size))
}

// Write_Pointer writes size bytes from the memory at pointer — Odin's write_ptr.
func Write_Pointer(stream Stream, pointer unsafe.Pointer, size int64) (count int64, err error) {
	if size < 0 {
		return 0, Stream_Negative_Count
	}
	return Write(stream, unsafe.Slice((*byte)(pointer), size))
}
