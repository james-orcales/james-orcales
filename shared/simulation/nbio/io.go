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

	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/random/prng"
	"local/james-orcales/shared/simulation/time"
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

// File_Callback receives a descriptor returned by an asynchronous filesystem operation.
type File_Callback func(completion *time.Completion, file File, err error)

// Socket_Callback receives a newly accepted server connection or an error.
type Socket_Callback func(completion *time.Completion, socket File, err error)

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

// SOCKET_OPTION_KEEPALIVE_IDLE sets the idle seconds before the first keepalive probe.
const SOCKET_OPTION_KEEPALIVE_IDLE Socket_Option = 5

// SOCKET_OPTION_KEEPALIVE_INTERVAL sets the seconds between two keepalive probes.
const SOCKET_OPTION_KEEPALIVE_INTERVAL Socket_Option = 6

// SOCKET_OPTION_KEEPALIVE_COUNT sets how many keepalive probes may fail before the transport
// drops the connection.
const SOCKET_OPTION_KEEPALIVE_COUNT Socket_Option = 7

// SOCKET_OPTION_USER_TIMEOUT caps the milliseconds unacknowledged data may stay in flight
// before the transport drops the connection. Darwin has no equivalent and ignores it.
const SOCKET_OPTION_USER_TIMEOUT Socket_Option = 8

// Directory_Callback receives one pass of directory entries, or the error that ended the walk.
// An empty slice with a nil error reports the end of the directory.
type Directory_Callback func(
	completion *time.Completion, entries []Directory_Entry, err error,
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

// The number of virtual grains a simulated operation may take to complete, drawn from
// the seed so the completion order varies per run while staying reproducible.
const SIM_LATENCY_GRAINS = 8

// One in this many simulated spawns exits non-zero, so a seed sweep exercises both the
// success and the failure path without a scripted outcome.
const SIM_SPAWN_FAIL_GRAINS = 4

// IO is the injected async IO submit surface — TigerBeetle's `IO`. Code submits
// operations with a time.Completion and callback and reacts to completions; it never drives
// the loop — that is the time.Driver's job — so a holder can submit IO but not advance time.
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
		completion *time.Completion, callback Callback, file File,
		buffer []byte, offset int64,
	)
	// Write writes buffer to file at offset (TigerBeetle IO.write).
	Write func(
		completion *time.Completion, callback Callback, file File,
		buffer []byte, offset int64,
	)
	// Fsync synchronizes file through TigerBeetle's asynchronous IO operation.
	Fsync func(completion *time.Completion, callback time.Timeout_Callback, file File)
	// Open_At asynchronously opens file_path relative to directory and forces close-on-exec.
	Open_At func(
		completion *time.Completion, callback File_Callback, directory File,
		file_path string,
		options Open_At_Options,
	)
	// Mkdir_At asynchronously creates one directory named by file_path relative to directory.
	// It is the mkdirat primitive, not a parent-creating mkdir: the parent must exist, and an
	// existing path reports the operating system error rather than converging. Make_Directory
	// composes this primitive above the surface, so both backends run the same composition.
	Mkdir_At func(
		completion *time.Completion, callback time.Timeout_Callback,
		directory File, file_path string,
		mode uint32,
	)
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
		completion *time.Completion, callback Directory_Callback, directory File,
		buffer []byte,
	)
	// Status reports whether path exists, whether it is a directory, and its byte size,
	// synchronously; an absent path is Exists false with a nil error, so a caller branches on
	// the status, not the error.
	Status func(path string) (status File_Status, err error)
	// Accept yields one inbound connection on listener before the deadline, or reports
	// Deadline_Exceeded.
	// The finite deadline deliberately diverges from TigerBeetle's unbounded IO.accept; it
	// keeps every repository submission bounded. TLS termination remains a secure_transport
	// concern.
	Accept func(
		completion *time.Completion, callback Socket_Callback, listener File,
		deadline time.Duration,
	)
	// Connect borrows caller-owned socket until it connects before deadline; callback reports
	// only the outcome. The deadline wins ties. The backend never creates, transfers, or closes
	// the descriptor. This finite lifetime is an explicit repository extension to TigerBeetle.
	Connect func(
		completion *time.Completion, callback time.Timeout_Callback,
		socket File, address Address, deadline time.Duration,
	)
	// Receive reads up to len(buffer) bytes from socket; callback reports the byte
	// count once data arrives (TigerBeetle IO.recv).
	Receive func(completion *time.Completion, callback Callback, socket File, buffer []byte)
	// Send writes buffer to socket; callback reports the byte count once the kernel
	// accepts it (TigerBeetle IO.send).
	Send func(completion *time.Completion, callback Callback, socket File, buffer []byte)
	// Send_Now makes TigerBeetle's best-effort synchronous datagram send. Sent is false only
	// when the datagram would block or the socket cannot send.
	Send_Now func(socket File, buffer []byte) (count int, sent bool)
	// Shutdown synchronously disables one or both connected-socket directions. It does not own
	// or close the socket (io/darwin.zig:973-975; io/linux.zig:1393-1395).
	Shutdown func(socket File, how Shutdown_How) (err error)
	// Close releases file's descriptor; callback fires once it is closed
	// (TigerBeetle IO.close).
	Close func(completion *time.Completion, callback time.Timeout_Callback, file File)
	// Close_Socket synchronously releases a socket during setup failure or final deinit. It is
	// TigerBeetle's close_socket, distinct from asynchronous Close.
	Close_Socket func(socket File)
	// Peer_Address returns the remote IP address of a connected socket, synchronously —
	// a getpeername has no completion. It is the source a control-plane connection is
	// gated on.
	Peer_Address func(file File) (address string, err error)
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
// on — is drawn from Generator, seeded by New_Simulated_IO. Correctness is asserted by the
// invariant framework's Always/Sometimes across many seeds (a plain seed sweep, like
// shared/vsr/simulation), and any failure reproduces by replaying its seed. Scripting
// destroys all three properties: the run becomes a function of author-chosen data
// instead of the seed, so (1) the fuzzer can only ever find bugs the author already
// imagined, (2) a seed no longer reproduces the run, and (3) the invariants end up
// judging hand-picked inputs instead of the whole reachable space. The right way to
// influence a run is to change the SEED, never to inject data.
//
// This type is unexported and New_Simulated_IO(seed) is the only entry precisely so this stays
// impossible to add by accident: no caller ever holds a *sim to hang a field on. If you
// find yourself wanting to export it, or wanting to add a parameter to New_Simulated_IO that is
// not the seed, stop — that is the scripting API trying to come back. Keep it shut.

// Returned when a path resolves to nothing — the simulator's ENOENT.
var sim_file_absent = errors.New("io: no such file or directory")

// Returned when a path component that must be a directory is a file.
var sim_not_a_directory = errors.New("io: not a directory")

// Returned when a file operation names a directory.
var sim_is_a_directory = errors.New("io: is a directory")

// Sim_Node is one entry in the simulator's in-memory filesystem: a directory with named
// children, or a file holding bytes. Generated from the seed at New_Simulated_IO and mutated by
// Create/Write/Make_Directory, so a later read reflects an earlier write.
type Sim_Node struct {
	// Directory reports whether this node is a directory rather than a file.
	Directory bool
	// Contents holds a file's bytes; nil for a directory.
	Contents []byte
	// Children maps a directory's entry names to their nodes; nil for a file.
	Children map[string]*Sim_Node
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
	// Timeline is the control plane every simulated operation retires through. The queue, the
	// tick, and the order live in shared/time, so this backend arms work and can never
	// advance it.
	Timeline time.Timeline
	// Any_Clock is the read-only time source; "now" is Any_Clock.Now_Monotonic.
	Any_Clock time.Any_Clock
	// Generator is the seeded entropy every outcome is drawn from; it is the sim's only
	// source of variation, so the whole run reproduces from the seed.
	Generator prng.Generator
	// Next_File is the synthetic descriptor counter; Listen, Accept, Open_Socket, Open,
	// and Create hand out the next value so every descriptor is distinct.
	Next_File File
	// Root is the in-memory filesystem the file ops read and mutate, fabricated from the
	// seed at New_Simulated_IO. Socket descriptors ignore it.
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
	// Operation_Files binds socket and file waiters to their descriptor so Close cancels them.
	Operation_Files map[*time.Completion]File
}

// New_Simulated_IO returns a deterministic loop seeded by seed: the read-only clock and submit
// surface to inject into the program under test, and the driver to run it. The sim
// itself never escapes, and seed is the only input, so the run reproduces exactly and
// nothing can be scripted into it — correctness is asserted by invariants, not by
// hand-fed outcomes. The driver stays in the harness: the program under test receives
// loop and clock, NEVER a pump (see the time.Driver banner).
func New_Simulated_IO(seed uint64, pump time.Timeline, clock time.Any_Clock) (loop IO) {
	state := &Sim{
		Timeline:          pump,
		Any_Clock:         clock,
		Generator:         prng.New(seed),
		Files:             map[File]*Sim_Node{},
		Directory_Drained: map[File]bool{},
		Raw_Open:          map[File]bool{},
		Sockets:           map[File]*Sim_Socket{},
		Operation_Files:   map[*time.Completion]File{},
	}
	state.Root = sim_generate(&state.Generator)
	pump.Report_Descriptors(func() (count int) { return len(state.Raw_Open) })
	sim_wire_bytes(state, &loop)
	sim_wire_lifecycle(state, &loop)
	sim_wire_socket(state, &loop)
	sim_wire_platform(state, &loop)
	return loop
}

// Wires the byte-count operations — read, write, receive, send — onto loop.
func sim_wire_bytes(state *Sim, loop *IO) {
	loop.Read = func(
		completion *time.Completion, callback Callback, file File,
		buffer []byte, offset int64,
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
		completion *time.Completion, callback Callback, file File,
		buffer []byte, offset int64,
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
	loop.Receive = func(
		completion *time.Completion, callback Callback, socket File, buffer []byte,
	) {
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
		state.Timeline.Classify(completion, time.OPERATION_READ_WAITER)
		state.Operation_Files[completion] = socket
	}
	loop.Send = func(
		completion *time.Completion, callback Callback, socket File, buffer []byte,
	) {
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
		state.Timeline.Classify(completion, time.OPERATION_WRITE_WAITER)
		state.Operation_Files[completion] = socket
	}
	loop.Fsync = func(completion *time.Completion, callback time.Timeout_Callback, file File) {
		sim_submit(state, completion, sim_latency(state),
			sim_deliver_status(completion, callback, nil))
		state.Operation_Files[completion] = file
	}
}

// Wires the two path primitives, Open_At and Mkdir_At, onto loop.
func sim_wire_path(state *Sim, loop *IO) {
	loop.Open_At = func(
		completion *time.Completion, callback File_Callback, directory File,
		file_path string,
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
		completion *time.Completion, callback time.Timeout_Callback,
		directory File, file_path string,
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

// Wires the filesystem lifecycle and both close primitives onto loop. Timers, next ticks, and
// the cross-thread event are not here: they are the loop's own control plane, and shared/time
// owns them.
func sim_wire_lifecycle(state *Sim, loop *IO) {
	sim_wire_path(state, loop)
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
		completion *time.Completion, callback Directory_Callback, directory File, _ []byte,
	) {
		sim_submit(state, completion, sim_latency(state), func() {
			callback(completion, sim_directory_pass(state, directory), nil)
		})
	}
	loop.Status = func(path string) (status File_Status, err error) {
		return sim_status(state.Root, path), nil
	}
	loop.Close = func(completion *time.Completion, callback time.Timeout_Callback, file File) {
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
		completion *time.Completion, callback Socket_Callback, listener File,
		deadline time.Duration,
	) {
		invariant.Always(deadline > 0, "An accept deadline is positive and finite.")
		sim_yield_socket(state, completion, callback, listener, deadline)
		state.Timeline.Classify(completion, time.OPERATION_READ_WAITER)
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
		completion *time.Completion, callback time.Timeout_Callback,
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
	state *Sim, completion *time.Completion, callback time.Timeout_Callback,
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
			sim_deliver_status(completion, callback, time.Deadline_Exceeded))
		state.Timeline.Classify(completion, time.OPERATION_WRITE_WAITER)
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
	state.Timeline.Classify(completion, time.OPERATION_WRITE_WAITER)
	state.Operation_Files[completion] = socket
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
	state *Sim, completion *time.Completion, callback Callback, node *Sim_Node,
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
	state *Sim, completion *time.Completion, callback Callback, node *Sim_Node,
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
func sim_deliver_bytes(completion *time.Completion, callback Callback, count int) (deliver func()) {
	return func() { callback(completion, count, nil) }
}

// Wraps a status delivery for a timeout or another result-only operation.
func sim_deliver_status(
	completion *time.Completion, callback time.Timeout_Callback, err error,
) (deliver func()) {
	return func() { callback(completion, err) }
}

// Submits a byte-count operation reporting the buffer length after the drawn latency.
func sim_bytes(state *Sim, completion *time.Completion, callback Callback, buffer []byte) {
	sim_submit(state, completion, sim_latency(state),
		sim_deliver_bytes(completion, callback, len(buffer)))
}

// Schedules callback to receive the next connected synthetic descriptor after drawn latency.
func sim_yield_socket(
	state *Sim, completion *time.Completion, callback Socket_Callback, listener File,
	deadline time.Duration,
) {
	latency := sim_latency(state)
	if latency >= deadline {
		sim_submit(state, completion, deadline, func() {
			callback(completion, File(-1), time.Deadline_Exceeded)
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

// Make_Directory_Input names one parent-creating directory create.
type Make_Directory_Input struct {
	// Timeline supplies the Mkdir_At primitive each component uses.
	Timeline IO
	// Completion is the caller-owned completion the result retires on.
	Completion *time.Completion
	// Callback runs once, after the last component or after the first real failure.
	Callback time.Timeout_Callback
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
	invariant.Always(
		input.Timeline.Mkdir_At != nil, "Make_Directory needs the Mkdir_At primitive.")
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
	state *Make_Directory_State, completion *time.Completion, err error,
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
	state.Input.Timeline.Mkdir_At(state.Input.Completion, func(
		completion *time.Completion, mkdir_err error,
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
	// Timeline supplies the Open_At, Get_Directory_Entries, and Close primitives.
	Timeline IO
	// Completion is the caller-owned completion the result retires on.
	Completion *time.Completion
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
		input.Timeline.Get_Directory_Entries != nil,
		"Read_Directory needs the Get_Directory_Entries primitive.",
	)
	state := &Read_Directory_State{
		Input:     input,
		Buffer:    make([]byte, DIRECTORY_PASS_BYTES),
		Entries:   []Directory_Entry{},
		Directory: -1,
	}
	state.Continue = func() { read_directory_pass(state) }
	input.Timeline.Open_At(input.Completion, func(
		completion *time.Completion, directory File, open_err error,
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
	state *Read_Directory_State, completion *time.Completion,
	entries []Directory_Entry, err error,
) {
	if state.Delivered {
		state.Input.Callback(completion, nil, time.Retired_Twice)
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
	state.Input.Timeline.Get_Directory_Entries(state.Input.Completion, func(
		_ *time.Completion, entries []Directory_Entry, pass_err error,
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
	state.Input.Timeline.Close(state.Input.Completion, func(
		completion *time.Completion, close_err error,
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
	keepalive := TCP_Keepalive{}
	if options.Keepalive != nil {
		keepalive = *options.Keepalive
	}
	// SOCKET_OPTION_KEEPALIVE must precede the tuple, because a transport rejects the timing
	// of a probe it is not yet sending.
	settings := []struct {
		Option Socket_Option
		Value  int
		Set    bool
	}{
		{SOCKET_OPTION_RECEIVE_BUFFER, options.Receive_Buffer, options.Receive_Buffer > 0},
		{SOCKET_OPTION_SEND_BUFFER, options.Send_Buffer, options.Send_Buffer > 0},
		{SOCKET_OPTION_KEEPALIVE, 1, options.Keepalive != nil},
		{SOCKET_OPTION_KEEPALIVE_IDLE, keepalive.Idle_Seconds, keepalive.Idle_Seconds > 0},
		{
			SOCKET_OPTION_KEEPALIVE_INTERVAL, keepalive.Interval_Seconds,
			keepalive.Interval_Seconds > 0,
		},
		{SOCKET_OPTION_KEEPALIVE_COUNT, keepalive.Count, keepalive.Count > 0},
		{
			SOCKET_OPTION_USER_TIMEOUT, options.User_Timeout_Milliseconds,
			options.User_Timeout_Milliseconds > 0,
		},
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

// Callback receives the result of a read or write: the byte count, or an error.
type Callback func(completion *time.Completion, count int, err error)

// Submits one simulated operation through the loop that owns the order. The class is stated
// here rather than at each call site, because a submit that reaches the queue unclassified is
// invisible to the census a stall shows up in.
func sim_submit(
	state *Sim, completion *time.Completion, latency time.Duration, callback func(),
) {
	// The borrow ends where the completion retires. The loop owns the drain and knows
	// nothing of descriptors, so the release rides the callback the loop runs — otherwise a
	// retired operation would still read as borrowing its file, and Close would assert.
	state.Timeline.Submit(completion, latency, func() {
		delete(state.Operation_Files, completion)
		callback()
	})
}

// Reads the moment the loop measures every Ready_At against.
func sim_now(state *Sim) (now time.Moment) {
	return state.Any_Clock.Now_Monotonic()
}
