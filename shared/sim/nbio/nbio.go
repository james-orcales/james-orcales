// Package nbio is dependency-injected completion surface of repository.
//
// No generic Cancel: owner use Shutdown, join every submitted operation, then submit Close.
// Repository-specific effect is marked extension and must retire through same completed queue.
package nbio

import (
	"errors"
	"unsafe"

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/sim/prng"
	"local/james-orcales/shared/sim/time"
)

// State carries caller-owned procedure state without an unsafe pointer or captured closure.
type State interface{}

// IO is injected async IO submit surface. Code submit operation with
// Completion and callback, then react to completion. Code never drive loop — that is
// Driver job — thus holder submit IO, but cannot advance clock.
//
// This is I/O seam, not kitchen sink of syscalls. Members transfer through file or socket
// endpoint, or own descriptor and path lifecycle. getrandom, getpid, mmap, and nanosleep do not
// belong because they name neither endpoint nor descriptor lifecycle. Dependency that need one
// declare it itself and take it injected, rather than widen this surface.
//
// Seam cut two ways, thus IO hold two halves. Network carry every socket endpoint, Storage every
// file endpoint, and holder take half it use: package that only write files hold Storage and
// cannot reach socket. Close and Deinit stay flat here, because both name descriptor, and both
// half hand descriptors out.
//
// CRITICAL: ONLY PACKAGE MAIN OR A TEST MAY DRIVE, RUN, OR TICK THE EVENT LOOP. IO holder that
// want to wait expose doneness as state and let root pump. It never receive pump. See
// time package driver contract.
type IO struct {
	Platform_IO
	// Network is every transfer whose endpoint is socket.
	Network Network
	// Storage is every transfer whose endpoint is file or directory.
	Storage Storage
	// Timeline is timer and cross-thread wakeup: when a completion run, with no endpoint.
	Timeline Timeline
	// Close release descriptor of file. Callback fire once it is closed. It sit here, not on
	// one half: close(2) name descriptor, and both half hand descriptors out. No state of own:
	// halves carry the backend pointer because each is handed out alone, and IO_Invariants
	// hold them equal, thus flat operations read Storage and a third copy buy nothing.
	Close_Procedure func(
		state State, completion *Completion, file File,
		callback Callback,
	)
	// Deinit assert every descriptor run open is closed. Surface own leak check, thus leaked
	// descriptor fail run. Caller need no census it must remember to read.
	Deinit_Procedure func(state State)
	// Watch_Signal fires callback when the process receives signal before the finite
	// deadline, or with Deadline_Exceeded. The lifetime is finite deliberately: a
	// permanent waiter is a process that cannot state when it is done. It sit here because
	// signal readiness is discovered by the poll pass, not by a timer: the OS backend caps its
	// own idle gap while a waiter live, and counts that waiter as work in flight.
	Watch_Signal func(
		state State, completion *Completion, signal Signal,
		deadline time.Duration, callback Signal_Callback,
	)
	// Spawn runs request until it finishes or the deadline expires. Expiry kills the
	// subprocess group and returns Deadline_Exceeded with any partial result. The
	// simulated backend draws the exit code from its seed and returns no output, since
	// scripted output is disallowed. It sit here because a child's pipes and its exit are
	// descriptor and kernel-event work this surface already owns.
	Spawn func(
		state State, completion *Completion, request Process_Request,
		deadline time.Duration, callback Process_Callback,
	)
}

// IO_Invariants admit an absent half: a holder's storage is zero before its Init binds a loop,
// and a test double fills only the half its subject use. Two present transfer halves name one
// backend, because flat operations read Storage.State and two backends composed into one IO
// would close on one and leak on the other. A timeline is absent or complete, never partial.
// Straight-line on purpose: a bundle carries no control flow, so every clause run every time.
func IO_Invariants(loop IO, _ aver.Namespace) {
	half := loop.Network.State == nil || loop.Storage.State == nil
	aver.Always(half || loop.Network.State == loop.Storage.State,
		"An IO carries one backend across both transfer halves.")
	empty := loop.Timeline.Submit == nil
	aver.Always((loop.Timeline.Open_Event == nil) == empty,
		"An IO timeline opens a cross-thread event, or is absent.")
	aver.Always((loop.Timeline.Event_Listen == nil) == empty,
		"An IO timeline listens for that event, or is absent.")
	aver.Always((loop.Timeline.Event_Trigger == nil) == empty,
		"An IO timeline triggers that event, or is absent.")
	aver.Always((loop.Timeline.Close_Event == nil) == empty,
		"An IO timeline closes that event, or is absent.")
}

// IO_Close keeps backend state explicit because bound method state would allocate.
func IO_Close(
	loop IO,
	completion *Completion, file File, callback Callback,
) {
	loop.Close_Procedure(loop.Storage.State, completion, file, callback)
}

// IO_Deinit keeps leak validation on backend that owns descriptor records.
func IO_Deinit(loop IO) {
	loop.Deinit_Procedure(loop.Storage.State)
}

// IO_Watch_Signal keeps backend state explicit so operation needs no captured environment.
func IO_Watch_Signal(
	loop IO, completion *Completion, signal Signal, deadline time.Duration,
	callback Signal_Callback,
) {
	loop.Watch_Signal(loop.Storage.State, completion, signal, deadline, callback)
}

// IO_Spawn keeps backend state explicit so operation needs no captured environment.
func IO_Spawn(
	loop IO, completion *Completion, request Process_Request, deadline time.Duration,
	callback Process_Callback,
) {
	loop.Spawn(loop.Storage.State, completion, request, deadline, callback)
}

// Signal identifies an operating-system signal in backend-independent form, so the
// deterministic and OS backends agree on a value without the pure tier importing syscall.
type Signal int

// SIGNAL_EXPIRED names no delivered signal. Caller retain signal it armed to identify watch.
const SIGNAL_EXPIRED Signal = -1

// SIGNAL_TERMINATE is the graceful-termination request (SIGTERM on the OS backend).
const SIGNAL_TERMINATE Signal = 0

// SIGNAL_INTERRUPT is the interactive interrupt (SIGINT on the OS backend).
const SIGNAL_INTERRUPT Signal = 1

// Signal_Callback receives a delivered signal on the loop thread, or Deadline_Exceeded when
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

// Network is half of IO: every transfer whose endpoint is socket, plus socket lifecycle that
// open one. Holder that only speak to socket take this and cannot reach file.
//
// TLS is not member. No accept-secure syscall, no connect-secure syscall, thus secure_transport
// compose these raw operations under its own record layer.
type Network struct {
	// State remains caller-owned while static procedures borrow it.
	State State
	// Socket_TCP make one configured non-blocking close-on-exec TCP socket.
	Socket_TCP_Procedure func(
		state State, family Address_Family, options TCP_Options,
	) (socket File, err error)
	// Socket_UDP make one configured non-blocking close-on-exec UDP socket.
	Socket_UDP_Procedure func(
		state State, family Address_Family, options UDP_Options,
	) (socket File, err error)
	// Bind enable address reuse and give caller-owned socket its local address.
	Bind_Procedure func(state State, socket File, address Address) (err error)
	// Listen_Socket mark bound socket accepting, with backlog as queue depth.
	Listen_Socket_Procedure func(
		state State, socket File, backlog uint32,
	) (err error)
	// Get_Socket_Name report socket own address. That is how caller learn port kernel chose
	// for port zero.
	Get_Socket_Name_Procedure func(
		state State, socket File, destination *Address,
	) (err error)
	// Accept yield one inbound connection on listener before timeout, or report
	// Deadline_Exceeded. Timeout is finite deliberately: it keep every repository
	// submission bounded.
	Accept_Procedure func(
		state State, completion *Completion, listener File,
		timeout time.Duration,
		callback Callback,
	)
	// Connect borrow caller-owned socket until one kernel result or timeout. Callback report
	// outcome only. Backend never make, transfer, or close descriptor.
	Connect_Procedure func(
		state State, completion *Completion, socket File, address Address,
		timeout time.Duration,
		callback Callback,
	)
	// Receive read up to len(buffer) bytes before timeout. Callback report byte count once data
	// arrive, or zero with Deadline_Exceeded after the kernel request retire.
	Receive_Procedure func(
		state State, completion *Completion, socket File, buffer []byte,
		timeout time.Duration,
		callback Callback,
	)
	// Send write buffer before timeout. Callback report byte count once kernel accept it, or
	// zero with Deadline_Exceeded after the kernel request retire.
	Send_Procedure func(
		state State, completion *Completion, socket File, buffer []byte,
		timeout time.Duration,
		callback Callback,
	)
	// Shutdown synchronously disable one or both connected-socket direction. It neither own nor
	// close socket.
	Shutdown_Procedure func(state State, socket File, how Shutdown_How) (err error)
	// Peer_Address return remote IP address of connected socket, synchronously — getpeername
	// has no completion. It is source a control-plane connection is gated on.
	Peer_Address_Procedure func(
		state State, file File, destination *Address,
	) (err error)
}

// Network_Socket_TCP passes caller-owned state to static TCP constructor.
func Network_Socket_TCP(
	network Network,
	family Address_Family, options TCP_Options,
) (socket File, err error) {
	return network.Socket_TCP_Procedure(network.State, family, options)
}

// Network_Socket_UDP passes caller-owned state to static UDP constructor.
func Network_Socket_UDP(
	network Network,
	family Address_Family, options UDP_Options,
) (socket File, err error) {
	return network.Socket_UDP_Procedure(network.State, family, options)
}

// Network_Bind passes caller-owned state to static address binder.
func Network_Bind(network Network, socket File, address Address) (err error) {
	return network.Bind_Procedure(network.State, socket, address)
}

// Network_Listen_Socket passes caller-owned state to static listener constructor.
func Network_Listen_Socket(network Network, socket File, backlog uint32) (err error) {
	return network.Listen_Socket_Procedure(network.State, socket, backlog)
}

// Network_Get_Socket_Name passes caller-owned state to static address reader.
func Network_Get_Socket_Name(
	network Network, socket File, destination *Address,
) (err error) {
	return network.Get_Socket_Name_Procedure(network.State, socket, destination)
}

// Network_Accept preserves callback-last submit shape while state remains explicit.
func Network_Accept(
	network Network,
	completion *Completion, listener File, timeout time.Duration,
	callback Callback,
) {
	network.Accept_Procedure(network.State, completion, listener, timeout, callback)
}

// Network_Connect preserves callback-last submit shape while state remains explicit.
func Network_Connect(
	network Network,
	completion *Completion, socket File, address Address, timeout time.Duration,
	callback Callback,
) {
	network.Connect_Procedure(network.State, completion, socket, address, timeout, callback)
}

// Network_Receive preserves callback-last submit shape while state remains explicit.
func Network_Receive(
	network Network,
	completion *Completion, socket File, buffer []byte, timeout time.Duration,
	callback Callback,
) {
	network.Receive_Procedure(network.State, completion, socket, buffer, timeout, callback)
}

// Network_Send preserves callback-last submit shape while state remains explicit.
func Network_Send(
	network Network,
	completion *Completion, socket File, buffer []byte, timeout time.Duration,
	callback Callback,
) {
	network.Send_Procedure(network.State, completion, socket, buffer, timeout, callback)
}

// Network_Shutdown passes caller-owned state to static direction control.
func Network_Shutdown(network Network, socket File, how Shutdown_How) (err error) {
	return network.Shutdown_Procedure(network.State, socket, how)
}

// Network_Peer_Address passes caller-owned state to static peer reader.
func Network_Peer_Address(
	network Network, file File, destination *Address,
) (err error) {
	return network.Peer_Address_Procedure(network.State, file, destination)
}

// Storage is other half of IO: every transfer whose endpoint is file or directory, plus path
// primitives that name one. Holder that only read and write file take this and cannot reach
// socket.
type Storage struct {
	// State remains caller-owned while static procedures borrow it.
	State State
	// Read read len(buffer) bytes from file at offset before timeout. Timeout does not work on
	// Darwin because its current file path has no kernel timeout. Darwin completes the
	// operation.
	Read_Procedure func(
		state State, completion *Completion, file File, buffer []byte,
		offset int64,
		timeout time.Duration,
		callback Callback,
	)
	// Write write buffer to file at offset before timeout. Timeout does not work on Darwin
	// because its current file path has no kernel timeout. Darwin completes the operation.
	Write_Procedure func(
		state State, completion *Completion, file File, buffer []byte,
		offset int64,
		timeout time.Duration,
		callback Callback,
	)
	// Fsync synchronize file before timeout. Timeout does not work on Darwin because its
	// current file path has no kernel timeout. Darwin completes the operation.
	Fsync_Procedure func(
		state State, completion *Completion, file File, timeout time.Duration,
		callback Callback,
	)
	// Open_At asynchronously open file_path relative to directory and force close-on-exec.
	Open_At_Procedure func(
		state State, completion *Completion, directory File, file_path string,
		options Open_At_Options, callback Callback,
	)
	// Mkdir_At asynchronously make one directory named by file_path relative to directory. It
	// is mkdirat primitive, not parent-creating mkdir: parent must exist, and existing path
	// report operating system error rather than converge. Make_Directory compose this
	// primitive above surface, thus both backend run same composition.
	Mkdir_At_Procedure func(
		state State, completion *Completion, directory File, file_path string,
		permissions File_Permissions,
		callback Callback,
	)
	// Get_Directory_Entries read one pass of directory raw entries into buffer and return
	// children it name, each with whether it is itself directory. Zero count report end.
	// Dirent layout is per-platform, thus parse and kind stay in backend, and only pass loop
	// and descriptor lifetime compose above.
	Get_Directory_Entries_Procedure func(
		state State, completion *Completion, directory File, buffer []byte,
		entries []Directory_Entry, callback Callback,
	)
	// Status report whether path exist, its portable mode, and its byte size, synchronously.
	// Absent path is Exists false with nil error, thus caller branch on status, not on error.
	Status_Procedure func(
		state State, path string,
	) (status File_Status, err error)
	// Read_Link writes symbolic-link target into caller storage. It is synchronous like Status:
	// neither operation owns descriptor or waits for readiness.
	Read_Link_Procedure func(
		state State, path string, destination []byte,
	) (count int, err error)
}

// Storage_Read preserves callback-last submit shape while state remains explicit.
func Storage_Read(
	storage Storage,
	completion *Completion, file File, buffer []byte, offset int64,
	timeout time.Duration, callback Callback,
) {
	storage.Read_Procedure(
		storage.State, completion, file, buffer, offset, timeout, callback,
	)
}

// Storage_Write preserves callback-last submit shape while state remains explicit.
func Storage_Write(
	storage Storage,
	completion *Completion, file File, buffer []byte, offset int64,
	timeout time.Duration, callback Callback,
) {
	storage.Write_Procedure(
		storage.State, completion, file, buffer, offset, timeout, callback,
	)
}

// Storage_Fsync preserves callback-last submit shape while state remains explicit.
func Storage_Fsync(
	storage Storage,
	completion *Completion, file File, timeout time.Duration,
	callback Callback,
) {
	storage.Fsync_Procedure(storage.State, completion, file, timeout, callback)
}

// Storage_Open_At preserves callback-last submit shape while state remains explicit.
func Storage_Open_At(
	storage Storage,
	completion *Completion, directory File, file_path string,
	options Open_At_Options, callback Callback,
) {
	File_Permissions_Invariants(options.Permissions, "Storage_Open_At.options.Permissions")
	storage.Open_At_Procedure(
		storage.State, completion, directory, file_path, options, callback,
	)
}

// Storage_Mkdir_At preserves callback-last submit shape while state remains explicit.
func Storage_Mkdir_At(
	storage Storage,
	completion *Completion, directory File, file_path string,
	permissions File_Permissions, callback Callback,
) {
	File_Permissions_Invariants(permissions, "Storage_Mkdir_At.permissions")
	storage.Mkdir_At_Procedure(
		storage.State, completion, directory, file_path, permissions, callback,
	)
}

// Storage_Get_Directory_Entries fills caller-owned entry storage before callback.
func Storage_Get_Directory_Entries(
	storage Storage,
	completion *Completion, directory File, buffer []byte, entries []Directory_Entry,
	callback Callback,
) {
	aver.Always(len(buffer) >= DIRECTORY_BUFFER_SIZE_MINIMUM,
		"Directory entry buffer holds at least one byte.")
	aver.Always(len(buffer) <= DIRECTORY_BUFFER_SIZE_MAXIMUM,
		"Directory entry buffer stays inside seam block budget.")
	aver.Always(len(entries) > 0, "Directory entry storage holds at least one result.")
	aver.Always(len(entries) <= DIRECTORY_BUFFER_SIZE_MAXIMUM,
		"Directory entry storage cannot exceed one result per record byte.")
	storage.Get_Directory_Entries_Procedure(
		storage.State, completion, directory, buffer, entries, callback,
	)
}

// Storage_Status passes caller-owned state to static path reader.
func Storage_Status(storage Storage, path string) (status File_Status, err error) {
	return storage.Status_Procedure(storage.State, path)
}

// Storage_Read_Link passes caller-owned path and result storage to static link reader.
func Storage_Read_Link(
	storage Storage, path string, destination []byte,
) (count int, err error) {
	return storage.Read_Link_Procedure(storage.State, path, destination)
}

// File identify one open file or socket. Simulated backend map it to tracked in-memory state.
// Real backend map it to descriptor or handle.
type File int32

// DIRECTORY_CURRENT select process current directory for Open_At. Default backend map this
// portable value to platform AT_FDCWD constant.
const DIRECTORY_CURRENT File = -1

// Open_Access select read/write access mode of asynchronous Open_At.
type Open_Access int

// OPEN_READ_ONLY open file for read.
const OPEN_READ_ONLY Open_Access = 0

// OPEN_WRITE_ONLY open file for write.
const OPEN_WRITE_ONLY Open_Access = 1

// OPEN_READ_WRITE open file for read and write.
const OPEN_READ_WRITE Open_Access = 2

// Open_At_Flags select independent Open_At control.
type Open_At_Flags uint32

// OPEN_AT_NO_FOLLOW reject symbolic link in final path part.
const OPEN_AT_NO_FOLLOW Open_At_Flags = 1 << 0

// Open_At_Options is Go form of posix.O fields openat use.
type Open_At_Options struct {
	// Access select read-only, write-only, or read-write access.
	Access Open_Access
	// Create make path when absent.
	Create bool
	// Truncate clear existing file before callback receive it.
	Truncate bool
	// Permissions is creation permission operand, used only when Create is true. Kernel may
	// still reduce it through process umask, thus stored mode is not always what caller state.
	Permissions File_Permissions
	// Flags hold independent Open_At control.
	Flags Open_At_Flags
}

// Address_Family is address family used to make and bind socket.
type Address_Family int

// FAMILY_IPV4 select IPv4 socket.
const FAMILY_IPV4 Address_Family = 0

// FAMILY_IPV6 select IPv6 socket.
const FAMILY_IPV6 Address_Family = 1

// These bounds keep public address layout equal to network protocol layout.
const IPV4_ADDRESS_BYTES = 4

// IPV6_ADDRESS_BYTES keep public address layout equal to IPv6 protocol layout.
const IPV6_ADDRESS_BYTES = 16

// ADDRESS_IPV4_OCTET_TEXT_BYTES_MAXIMUM includes every decimal byte value.
const ADDRESS_IPV4_OCTET_TEXT_BYTES_MAXIMUM = 3

// ADDRESS_IPV4_TEXT_BYTES_MAXIMUM includes four octets and their separators.
const ADDRESS_IPV4_TEXT_BYTES_MAXIMUM = (IPV4_ADDRESS_BYTES*ADDRESS_IPV4_OCTET_TEXT_BYTES_MAXIMUM +
	IPV4_ADDRESS_BYTES - 1)

// ADDRESS_IPV6_GROUP_COUNT is the number of two-byte groups in one IPv6 address.
const ADDRESS_IPV6_GROUP_COUNT = IPV6_ADDRESS_BYTES / 2

// ADDRESS_IPV6_GROUP_TEXT_BYTES_MAXIMUM includes every hexadecimal group value.
const ADDRESS_IPV6_GROUP_TEXT_BYTES_MAXIMUM = 4

// ADDRESS_TEXT_BYTES_MAXIMUM includes the longest IPv6 form with an embedded dotted quad.
const ADDRESS_TEXT_BYTES_MAXIMUM = (ADDRESS_IPV6_GROUP_COUNT-2)*
	ADDRESS_IPV6_GROUP_TEXT_BYTES_MAXIMUM +
	(ADDRESS_IPV6_GROUP_COUNT - 2) + ADDRESS_IPV4_TEXT_BYTES_MAXIMUM

// Address is IP address and port with explicit family. IP borrows exact family-sized caller
// storage so value never hides allocation.
type Address struct {
	// Family select how IP is read.
	Family Address_Family
	// IP hold network address bytes.
	IP []byte
	// Port is host-order TCP or UDP port.
	Port uint16
}

// Address_IPV4 borrows four caller-owned octets and host-order port.
func Address_IPV4(ip []byte, port uint16) (address Address) {
	aver.Always(len(ip) == IPV4_ADDRESS_BYTES, "An IPv4 address has four octets.")
	return Address{
		Family: FAMILY_IPV4, IP: ip[:IPV4_ADDRESS_BYTES:IPV4_ADDRESS_BYTES], Port: port,
	}
}

// Address_IPV6 borrows 16 caller-owned octets and host-order port.
func Address_IPV6(ip []byte, port uint16) (address Address) {
	aver.Always(len(ip) == IPV6_ADDRESS_BYTES, "An IPv6 address has 16 octets.")
	return Address{
		Family: FAMILY_IPV6, IP: ip[:IPV6_ADDRESS_BYTES:IPV6_ADDRESS_BYTES], Port: port,
	}
}

// Address_Parse parse IP literal without DNS into caller-owned destination. Host that hold colon
// parse as IPv6, else as IPv4. Anything else error.
//
// Parse is written here, not taken from net/netip, thus this tier import no part of net tree. One
// import of net is one step from a resolver, and resolver block. Address is 20 bytes of plain
// data, and its text form is small enough to read here.
func Address_Parse(destination *Address, host string, port int) (err error) {
	storage, storage_err := address_destination_storage(destination)
	if storage_err != nil {
		return storage_err
	}
	if port < 0 {
		return address_port_outside_uint16
	}
	if port > 65535 {
		return address_port_outside_uint16
	}
	if len(host) > ADDRESS_TEXT_BYTES_MAXIMUM {
		return address_host_too_large
	}
	if text_contains(host, ":") {
		found := address_parse_ipv6(host, storage)
		if !found {
			return address_host_not_literal
		}
		*destination = Address_IPV6(storage, uint16(port))
		return nil
	}
	quad := storage[:IPV4_ADDRESS_BYTES:IPV4_ADDRESS_BYTES]
	found := address_parse_ipv4(host, quad)
	if !found {
		return address_host_not_literal
	}
	*destination = Address{
		Family: FAMILY_IPV4, IP: storage[:IPV4_ADDRESS_BYTES:IPV6_ADDRESS_BYTES],
		Port: uint16(port),
	}
	return nil
}

// Preserve backing capacity across IPv4, empty, and failed results so one destination can serve
// every address family without heap ownership moving into this package.
func address_destination_storage(destination *Address) (storage []byte, err error) {
	if destination == nil {
		return nil, address_storage_too_small
	}
	if cap(destination.IP) < IPV6_ADDRESS_BYTES {
		return nil, address_storage_too_small
	}
	storage = destination.IP[:IPV6_ADDRESS_BYTES:IPV6_ADDRESS_BYTES]
	*destination = Address{IP: storage[:0:IPV6_ADDRESS_BYTES]}
	return storage, nil
}

var address_storage_too_small = errors.New("io: address storage is smaller than IPv6")
var address_port_outside_uint16 = errors.New("io: port is outside uint16")
var address_host_too_large = errors.New("io: host is too large")
var address_host_not_literal = errors.New("io: host is not an IP literal")

// Parse dotted-quad into four octets.
func address_parse_ipv4(host string, quad []byte) (found bool) {
	parts := [IPV4_ADDRESS_BYTES]string{}
	part_count, split := text_split(host, '.', parts[:])
	if !split {
		return false
	}
	if part_count != IPV4_ADDRESS_BYTES {
		return false
	}
	for index, part := range parts {
		octet, valid := address_parse_octet(part)
		if !valid {
			return false
		}
		quad[index] = octet
	}
	return true
}

// Parse one decimal octet: one to three digits, at most 255, and no leading zero. Leading zero is
// rejected because one stack read it octal and another read it decimal, thus "010.1.1.1" name two
// different hosts.
func address_parse_octet(part string) (octet byte, valid bool) {
	if len(part) == 0 {
		return 0, false
	}
	if len(part) > 3 {
		return 0, false
	}
	if len(part) > 1 {
		if part[0] == '0' {
			return 0, false
		}
	}
	value := 0
	for index := range part {
		digit := part[index]
		if digit < '0' {
			return 0, false
		}
		if digit > '9' {
			return 0, false
		}
		value = value*10 + int(digit-'0')
	}
	if value > 255 {
		return 0, false
	}
	return byte(value), true
}

// An IPv6 literal hold at most this many colon-separated parts, thus a long host string cannot
// make the parse walk without bound.
const ADDRESS_IPV6_PARTS_MAX = 8

// Parse IPv6 literal into 16 octets. It accept full form, one "::" run, and trailing embedded
// IPv4. It reject zone identifier: Address hold no zone, thus accept of one would drop it silent.
func address_parse_ipv6(host string, sextets []byte) (found bool) {
	for index := range sextets {
		sextets[index] = 0
	}
	if text_contains(host, "%") {
		return false
	}
	head, tail, compressed := text_cut(host, "::")
	if compressed {
		return address_join_ipv6(head, tail, sextets)
	}
	value_count, valid := address_parse_ipv6_parts(host, sextets)
	if !valid {
		return false
	}
	if value_count != IPV6_ADDRESS_BYTES {
		return false
	}
	return true
}

// Join both halves of a compressed literal, with zeros between them. Run must stand for at least
// one group, thus the two halves together leave two bytes free.
func address_join_ipv6(head string, tail string, sextets []byte) (found bool) {
	if text_contains(tail, "::") {
		return false
	}
	front_count, front_valid := address_parse_ipv6_parts(head, sextets)
	if !front_valid {
		return false
	}
	back_count, back_valid := address_parse_ipv6_parts(tail, sextets[front_count:])
	if !back_valid {
		return false
	}
	if front_count+back_count > IPV6_ADDRESS_BYTES-2 {
		return false
	}
	copy(
		sextets[IPV6_ADDRESS_BYTES-back_count:],
		sextets[front_count:front_count+back_count],
	)
	for index := front_count; index < IPV6_ADDRESS_BYTES-back_count; index++ {
		sextets[index] = 0
	}
	return true
}

// Parse one colon-separated group run into its bytes. Empty text yield no bytes. Trailing
// dotted-quad contribute four bytes, which is how embedded IPv4 reach the low 32 bits.
func address_parse_ipv6_parts(
	text string, values []byte,
) (count int, valid bool) {
	if text == "" {
		return 0, true
	}
	parts := [ADDRESS_IPV6_PARTS_MAX]string{}
	part_count, split := text_split(text, ':', parts[:])
	if !split {
		return 0, false
	}
	for index := 0; index < part_count; index++ {
		part := parts[index]
		if index == part_count-1 {
			if text_contains(part, ".") {
				quad := values[count:]
				if len(quad) < IPV4_ADDRESS_BYTES {
					return 0, false
				}
				quad = quad[:IPV4_ADDRESS_BYTES:IPV4_ADDRESS_BYTES]
				quad_valid := address_parse_ipv4(part, quad)
				if !quad_valid {
					return 0, false
				}
				count += IPV4_ADDRESS_BYTES
				return count, true
			}
		}
		high, low, group_valid := address_parse_group(part)
		if !group_valid {
			return 0, false
		}
		if count+2 > len(values) {
			return 0, false
		}
		values[count] = high
		values[count+1] = low
		count += 2
	}
	return count, true
}

// Parse one IPv6 group: one to four hexadecimal digits.
func address_parse_group(part string) (high byte, low byte, valid bool) {
	if len(part) == 0 {
		return 0, 0, false
	}
	if len(part) > 4 {
		return 0, 0, false
	}
	value := 0
	for index := range part {
		digit, digit_valid := address_hexadecimal(part[index])
		if !digit_valid {
			return 0, 0, false
		}
		value = value*16 + int(digit)
	}
	return byte(value >> 8), byte(value), true
}

// Map one ASCII hexadecimal digit to its value.
func address_hexadecimal(character byte) (value byte, valid bool) {
	if character >= '0' {
		if character <= '9' {
			return character - '0', true
		}
	}
	if character >= 'a' {
		if character <= 'f' {
			return character - 'a' + 10, true
		}
	}
	if character >= 'A' {
		if character <= 'F' {
			return character - 'A' + 10, true
		}
	}
	return 0, false
}

// These low-level readers stay here because shared/strings depends on the stream layer that
// nbio defines, so importing it would make the dependency cycle back into this package.
func text_contains(source string, fragment string) (contained bool) {
	_, _, contained = text_cut(source, fragment)
	return contained
}

func text_cut(source string, separator string) (before string, after string, found bool) {
	if separator == "" {
		return "", source, true
	}
	if len(separator) > len(source) {
		return source, "", false
	}
	for position := 0; position <= len(source)-len(separator); position++ {
		if source[position:position+len(separator)] == separator {
			return source[:position], source[position+len(separator):], true
		}
	}
	return source, "", false
}

func text_split(
	source string, separator byte, destination []string,
) (count int, valid bool) {
	start := 0
	for position := range source {
		if source[position] != separator {
			continue
		}
		if count == len(destination) {
			return 0, false
		}
		destination[count] = source[start:position]
		count++
		start = position + 1
	}
	if count == len(destination) {
		return 0, false
	}
	destination[count] = source[start:]
	return count + 1, true
}

// TCP_BUFFER_KIBIBYTES_DEFAULT names the profile size before conversion to bytes.
const TCP_BUFFER_KIBIBYTES_DEFAULT uint32 = 128

// TCP_RECEIVE_BUFFER_BYTES_DEFAULT uses the higher supported value to keep the pure profile fixed.
const TCP_RECEIVE_BUFFER_BYTES_DEFAULT uint32 = (TCP_BUFFER_KIBIBYTES_DEFAULT *
	bits.KIBIBYTE_BYTES)

// TCP_SEND_BUFFER_BYTES_DEFAULT uses the higher supported value to keep the pure profile fixed.
const TCP_SEND_BUFFER_BYTES_DEFAULT uint32 = (TCP_BUFFER_KIBIBYTES_DEFAULT *
	bits.KIBIBYTE_BYTES)

// IPV6_SOCKET_ADDRESS_BITS uses the larger address record so both address families fit.
const IPV6_SOCKET_ADDRESS_BITS uint32 = bits.BIT_COUNT_16_MAXIMUM*2 +
	bits.BIT_COUNT_32_MAXIMUM*2 + bits.BIT_COUNT_64_MAXIMUM*2

// IPV6_SOCKET_ADDRESS_BYTES keeps UDP record storage tied to the address field widths.
const IPV6_SOCKET_ADDRESS_BYTES uint32 = (IPV6_SOCKET_ADDRESS_BITS /
	bits.BIT_COUNT_8_MAXIMUM)

// UDP_RECEIVE_RECORD_BYTES includes source storage because UDP preserves message origin.
const UDP_RECEIVE_RECORD_BYTES uint32 = (bits.KIBIBYTE_BYTES +
	IPV6_SOCKET_ADDRESS_BYTES)

// UDP_RECEIVE_BUFFER_BASELINE_BYTES keeps the historical record capacity, not its rounded total.
const UDP_RECEIVE_BUFFER_BASELINE_BYTES uint32 = 192 * bits.KIBIBYTE_BYTES

// UDP_RECEIVE_BUFFER_SCALE_DEFAULT selects the high profile without an opaque byte total.
const UDP_RECEIVE_BUFFER_SCALE_DEFAULT uint32 = 4

// UDP_RECEIVE_RECORD_COUNT_BASELINE rounds up because a partial record needs full storage.
const UDP_RECEIVE_RECORD_COUNT_BASELINE uint32 = ((UDP_RECEIVE_BUFFER_BASELINE_BYTES +
	UDP_RECEIVE_RECORD_BYTES - 1) / UDP_RECEIVE_RECORD_BYTES)

// UDP_RECEIVE_RECORD_COUNT_DEFAULT scales records so the result cannot split one datagram.
const UDP_RECEIVE_RECORD_COUNT_DEFAULT uint32 = (UDP_RECEIVE_BUFFER_SCALE_DEFAULT *
	UDP_RECEIVE_RECORD_COUNT_BASELINE)

// UDP_RECEIVE_BUFFER_BYTES_DEFAULT uses whole records because UDP preserves message boundaries.
const UDP_RECEIVE_BUFFER_BYTES_DEFAULT uint32 = (UDP_RECEIVE_RECORD_COUNT_DEFAULT *
	UDP_RECEIVE_RECORD_BYTES)

// UDP_SEND_BUFFER_KIBIBYTES_DEFAULT names the profile size before conversion to bytes.
const UDP_SEND_BUFFER_KIBIBYTES_DEFAULT uint32 = 208

// UDP_SEND_BUFFER_BYTES_DEFAULT uses the higher supported value to keep the pure profile fixed.
const UDP_SEND_BUFFER_BYTES_DEFAULT uint32 = (UDP_SEND_BUFFER_KIBIBYTES_DEFAULT *
	bits.KIBIBYTE_BYTES)

// IPV4_REASSEMBLY_BYTES_MINIMUM makes the MSS usable without route information.
const IPV4_REASSEMBLY_BYTES_MINIMUM uint32 = 576

// INTERNET_HEADER_WORD_BYTES follows the shared IPv4 and TCP header-length unit.
const INTERNET_HEADER_WORD_BYTES uint32 = (bits.BIT_COUNT_32_MAXIMUM /
	bits.BIT_COUNT_8_MAXIMUM)

// IPV4_HEADER_BYTES_MINIMUM leaves no optional IPv4 words in the default MSS.
const IPV4_HEADER_BYTES_MINIMUM uint32 = 5 * INTERNET_HEADER_WORD_BYTES

// TCP_HEADER_BYTES_MINIMUM leaves no optional TCP words in the default MSS.
const TCP_HEADER_BYTES_MINIMUM uint32 = 5 * INTERNET_HEADER_WORD_BYTES

// TCP_MAXIMUM_SEGMENT_BYTES_DEFAULT leaves room for minimum IPv4 and TCP headers.
const TCP_MAXIMUM_SEGMENT_BYTES_DEFAULT uint32 = IPV4_REASSEMBLY_BYTES_MINIMUM -
	IPV4_HEADER_BYTES_MINIMUM - TCP_HEADER_BYTES_MINIMUM

// TCP_NOT_SENT_LOW_WATER_BYTES_DEFAULT bounds queued bytes without restricting small writes.
const TCP_NOT_SENT_LOW_WATER_BYTES_DEFAULT uint32 = bits.KIBIBYTE_BYTES

// TCP_KEEPALIVE_IDLE_DEFAULT uses two hours because all supported backends use that value.
const TCP_KEEPALIVE_IDLE_DEFAULT time.Duration = 2 * time.HOUR

// TCP_KEEPALIVE_INTERVAL_DEFAULT uses 75 seconds because all supported backends use that value.
const TCP_KEEPALIVE_INTERVAL_DEFAULT time.Duration = 75 * time.SECOND

// TCP_KEEPALIVE_PROBE_COUNT_DEFAULT uses the higher supported value to keep the pure profile fixed.
const TCP_KEEPALIVE_PROBE_COUNT_DEFAULT uint32 = 9

// TCP_Keepalive selects per-socket limits for an enabled keepalive probe sequence.
type TCP_Keepalive struct {
	// A positive value bounds the idle period before the first probe.
	Idle time.Duration
	// A positive value bounds the delay between probes.
	Interval time.Duration
	// A positive value bounds the probes before connection termination.
	Probe_Count uint32
}

// TCP_Options selects limits for one TCP socket.
type TCP_Options struct {
	// A positive value bounds the receive queue.
	Receive_Buffer_Bytes uint32
	// A positive value bounds the send queue.
	Send_Buffer_Bytes uint32
	// A positive value keeps the receive threshold enabled.
	Receive_Low_Water_Bytes uint32
	// A positive value bounds close after shutdown.
	Linger_Timeout time.Duration
	// A positive value bounds one TCP segment before connect.
	Maximum_Segment_Bytes uint32
	// A positive value bounds bytes that TCP has not sent.
	Not_Sent_Low_Water_Bytes uint32
	// Direct ownership prevents a caller from disabling keepalive.
	Keepalive TCP_Keepalive
	// Use TCP_NO_DELAY_DEFAULT unless packet coalescing is necessary.
	No_Delay bool
}

// UDP_Options selects limits for one UDP socket.
type UDP_Options struct {
	// A positive value bounds the receive queue.
	Receive_Buffer_Bytes uint32
	// A positive value bounds the send queue.
	Send_Buffer_Bytes uint32
	// A positive value keeps the receive threshold enabled.
	Receive_Low_Water_Bytes uint32
	// A positive value bounds close after shutdown.
	Linger_Timeout time.Duration
}

// Shutdown_How select which connected-socket direction shutdown disable.
type Shutdown_How int

// SHUTDOWN_RECEIVE disable further receive.
const SHUTDOWN_RECEIVE Shutdown_How = 0

// SHUTDOWN_SEND disable further send.
const SHUTDOWN_SEND Shutdown_How = 1

// SHUTDOWN_BOTH disable receive and send.
const SHUTDOWN_BOTH Shutdown_How = 2

// Directory_Entry is one child of directory: name, and whether it is itself directory. Those two
// facts are what walk need to recurse into subdirectory and sync file.
type Directory_Entry struct {
	// Name is child name inside its directory, not full path.
	Name string
	// Is_Directory report whether child is directory, thus walk know to recurse.
	Is_Directory bool
}

// DIRECTORY_BUFFER_SIZE_MINIMUM lets one raw directory record enter backend storage.
const DIRECTORY_BUFFER_SIZE_MINIMUM = 1

// DIRECTORY_BUFFER_SIZE_MAXIMUM matches Go os Unix directory block budget.
const DIRECTORY_BUFFER_SIZE_MAXIMUM = 8 * bits.KIBIBYTE_BYTES

// File_Mode keeps portable permission and entry-kind metadata scalar. Layout follows Go
// io/fs.FileMode, thus a caller crosses any standard-library boundary with one cast and no
// translation table. Platform mode words are decoded into this domain, never handed out raw.
type File_Mode uint32

// FILE_MODE_MINIMUM begins portable mode domain.
const FILE_MODE_MINIMUM uint32 = 0

// FILE_MODE_MAXIMUM closes portable mode domain.
const FILE_MODE_MAXIMUM uint32 = uint32(FILE_MODE_VALID)

// FILE_MODE_PERMISSION_MINIMUM begins permission bits.
const FILE_MODE_PERMISSION_MINIMUM uint32 = 0

// FILE_MODE_PERMISSION_MAXIMUM closes permission bits.
const FILE_MODE_PERMISSION_MAXIMUM uint32 = uint32(FILE_MODE_PERMISSIONS)

// FILE_MODE_OTHER_EXECUTE is the lowest portable permission bit.
const FILE_MODE_OTHER_EXECUTE File_Mode = 1

// FILE_MODE_OTHER_WRITE follows other-execute permission.
const FILE_MODE_OTHER_WRITE File_Mode = FILE_MODE_OTHER_EXECUTE << 1

// FILE_MODE_OTHER_READ follows other-write permission.
const FILE_MODE_OTHER_READ File_Mode = FILE_MODE_OTHER_WRITE << 1

// FILE_MODE_PERMISSION_CLASS_BIT_COUNT is one rwx permission group.
const FILE_MODE_PERMISSION_CLASS_BIT_COUNT = 3

// FILE_MODE_EXECUTE_PERMISSIONS repeats execute across all permission classes.
const FILE_MODE_EXECUTE_PERMISSIONS File_Mode = FILE_MODE_OTHER_EXECUTE |
	FILE_MODE_OTHER_EXECUTE<<FILE_MODE_PERMISSION_CLASS_BIT_COUNT |
	FILE_MODE_OTHER_EXECUTE<<(2*FILE_MODE_PERMISSION_CLASS_BIT_COUNT)

// FILE_MODE_WRITE_PERMISSIONS repeats write across all permission classes.
const FILE_MODE_WRITE_PERMISSIONS File_Mode = FILE_MODE_OTHER_WRITE |
	FILE_MODE_OTHER_WRITE<<FILE_MODE_PERMISSION_CLASS_BIT_COUNT |
	FILE_MODE_OTHER_WRITE<<(2*FILE_MODE_PERMISSION_CLASS_BIT_COUNT)

// FILE_MODE_READ_PERMISSIONS repeats read across all permission classes.
const FILE_MODE_READ_PERMISSIONS File_Mode = FILE_MODE_OTHER_READ |
	FILE_MODE_OTHER_READ<<FILE_MODE_PERMISSION_CLASS_BIT_COUNT |
	FILE_MODE_OTHER_READ<<(2*FILE_MODE_PERMISSION_CLASS_BIT_COUNT)

// FILE_MODE_OWNER_WRITE locates owner mutation permission.
const FILE_MODE_OWNER_WRITE = FILE_MODE_OTHER_WRITE <<
	(2 * FILE_MODE_PERMISSION_CLASS_BIT_COUNT)

// FILE_MODE_PERMISSIONS keeps portable permission bits.
const FILE_MODE_PERMISSIONS File_Mode = FILE_MODE_EXECUTE_PERMISSIONS |
	FILE_MODE_WRITE_PERMISSIONS | FILE_MODE_READ_PERMISSIONS

// FILE_MODE_DIRECTORY identifies directory entry.
const FILE_MODE_DIRECTORY File_Mode = 1 << (bits.BIT_COUNT_32_MAXIMUM - 1)

// FILE_MODE_APPEND identifies append-only entry.
const FILE_MODE_APPEND File_Mode = FILE_MODE_DIRECTORY >> 1

// FILE_MODE_EXCLUSIVE identifies exclusive-use entry.
const FILE_MODE_EXCLUSIVE File_Mode = FILE_MODE_APPEND >> 1

// FILE_MODE_TEMPORARY identifies temporary entry.
const FILE_MODE_TEMPORARY File_Mode = FILE_MODE_EXCLUSIVE >> 1

// FILE_MODE_SYMBOLIC_LINK identifies symbolic link entry.
const FILE_MODE_SYMBOLIC_LINK File_Mode = FILE_MODE_TEMPORARY >> 1

// FILE_MODE_DEVICE identifies device entry.
const FILE_MODE_DEVICE File_Mode = FILE_MODE_SYMBOLIC_LINK >> 1

// FILE_MODE_NAMED_PIPE identifies FIFO entry.
const FILE_MODE_NAMED_PIPE File_Mode = FILE_MODE_DEVICE >> 1

// FILE_MODE_SOCKET identifies socket entry.
const FILE_MODE_SOCKET File_Mode = FILE_MODE_NAMED_PIPE >> 1

// FILE_MODE_SET_USER_IDENTIFIER identifies setuid entry.
const FILE_MODE_SET_USER_IDENTIFIER File_Mode = FILE_MODE_SOCKET >> 1

// FILE_MODE_SET_GROUP_IDENTIFIER identifies setgid entry.
const FILE_MODE_SET_GROUP_IDENTIFIER File_Mode = FILE_MODE_SET_USER_IDENTIFIER >> 1

// FILE_MODE_CHARACTER_DEVICE refines device entry.
const FILE_MODE_CHARACTER_DEVICE File_Mode = FILE_MODE_SET_GROUP_IDENTIFIER >> 1

// FILE_MODE_STICKY identifies sticky entry.
const FILE_MODE_STICKY File_Mode = FILE_MODE_CHARACTER_DEVICE >> 1

// FILE_MODE_IRREGULAR identifies unknown entry kind.
const FILE_MODE_IRREGULAR File_Mode = FILE_MODE_STICKY >> 1

// FILE_MODE_TYPE combines portable entry-kind bits. Regular file is absence of every one of
// them, thus a zero mode word reads as a regular file with no permission.
const FILE_MODE_TYPE = FILE_MODE_DIRECTORY | FILE_MODE_SYMBOLIC_LINK |
	FILE_MODE_NAMED_PIPE | FILE_MODE_SOCKET | FILE_MODE_DEVICE |
	FILE_MODE_CHARACTER_DEVICE | FILE_MODE_IRREGULAR

// FILE_MODE_VALID combines all representable mode bits.
const FILE_MODE_VALID = FILE_MODE_PERMISSIONS | FILE_MODE_DIRECTORY |
	FILE_MODE_APPEND | FILE_MODE_EXCLUSIVE | FILE_MODE_TEMPORARY |
	FILE_MODE_SYMBOLIC_LINK | FILE_MODE_DEVICE | FILE_MODE_NAMED_PIPE |
	FILE_MODE_SOCKET | FILE_MODE_SET_USER_IDENTIFIER |
	FILE_MODE_SET_GROUP_IDENTIFIER | FILE_MODE_CHARACTER_DEVICE |
	FILE_MODE_STICKY | FILE_MODE_IRREGULAR

// File_Mode_Invariants rejects bits without repository meaning.
func File_Mode_Invariants(value File_Mode, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), FILE_MODE_MINIMUM, FILE_MODE_MAXIMUM).
		Ensure()
	aver.Always(value&^FILE_MODE_VALID == 0,
		"A portable file mode carries only representable bits.")
}

// File_Permissions is creation permission subset one open or one directory make may request.
// Kind is not requestable: the operation names the kind, thus a creation operand that carried
// one would be describing state the kernel derives, not state the caller supplies.
type File_Permissions uint32

// FILE_PERMISSIONS_VALID combines every requestable creation bit.
const FILE_PERMISSIONS_VALID File_Permissions = File_Permissions(
	FILE_MODE_PERMISSIONS | FILE_MODE_SET_USER_IDENTIFIER |
		FILE_MODE_SET_GROUP_IDENTIFIER | FILE_MODE_STICKY,
)

// FILE_PERMISSIONS_MINIMUM begins creation permission domain.
const FILE_PERMISSIONS_MINIMUM uint32 = 0

// FILE_PERMISSIONS_MAXIMUM closes creation permission domain.
const FILE_PERMISSIONS_MAXIMUM uint32 = uint32(FILE_PERMISSIONS_VALID)

// File_Permissions_Invariants rejects kind bits inside one creation operand.
func File_Permissions_Invariants(value File_Permissions, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(
			uint32(value), FILE_PERMISSIONS_MINIMUM, FILE_PERMISSIONS_MAXIMUM,
		).
		Ensure()
	aver.Always(value&^FILE_PERMISSIONS_VALID == 0,
		"A creation permission operand carries no entry-kind bit.")
}

// POSIX_FILE_MODE_MINIMUM begins the hostile 16-bit platform mode domain.
const POSIX_FILE_MODE_MINIMUM uint16 = 0

// POSIX_FILE_MODE_MAXIMUM closes the hostile 16-bit platform mode domain.
const POSIX_FILE_MODE_MAXIMUM uint16 = 1<<bits.BIT_COUNT_16_MAXIMUM - 1

// POSIX_FILE_TYPE_BIT_COUNT is one half-byte kind field, thus kind occupies the top nibble.
const POSIX_FILE_TYPE_BIT_COUNT = bits.BIT_COUNT_8_MAXIMUM / 2

// POSIX_FILE_TYPE_SHIFT locates the POSIX kind nibble.
const POSIX_FILE_TYPE_SHIFT = bits.BIT_COUNT_16_MAXIMUM - POSIX_FILE_TYPE_BIT_COUNT

// POSIX_FILE_TYPE_MASK selects the POSIX kind nibble.
const POSIX_FILE_TYPE_MASK = ((1 << POSIX_FILE_TYPE_BIT_COUNT) - 1) <<
	POSIX_FILE_TYPE_SHIFT

// POSIX_FILE_MODE_NAMED_PIPE encodes the POSIX FIFO kind.
const POSIX_FILE_MODE_NAMED_PIPE uint16 = 1 << POSIX_FILE_TYPE_SHIFT

// POSIX_FILE_MODE_CHARACTER_DEVICE encodes the POSIX character-device kind.
const POSIX_FILE_MODE_CHARACTER_DEVICE uint16 = 2 << POSIX_FILE_TYPE_SHIFT

// POSIX_FILE_MODE_DIRECTORY encodes the POSIX directory kind.
const POSIX_FILE_MODE_DIRECTORY uint16 = 4 << POSIX_FILE_TYPE_SHIFT

// POSIX_FILE_MODE_DEVICE encodes the POSIX block-device kind.
const POSIX_FILE_MODE_DEVICE uint16 = 6 << POSIX_FILE_TYPE_SHIFT

// POSIX_FILE_MODE_REGULAR encodes the POSIX regular-file kind.
const POSIX_FILE_MODE_REGULAR uint16 = 8 << POSIX_FILE_TYPE_SHIFT

// POSIX_FILE_MODE_SYMBOLIC_LINK encodes the POSIX symbolic-link kind.
const POSIX_FILE_MODE_SYMBOLIC_LINK uint16 = 10 << POSIX_FILE_TYPE_SHIFT

// POSIX_FILE_MODE_SOCKET encodes the POSIX socket kind.
const POSIX_FILE_MODE_SOCKET uint16 = 12 << POSIX_FILE_TYPE_SHIFT

// POSIX_FILE_MODE_SET_USER_IDENTIFIER encodes the POSIX set-user-ID bit.
const POSIX_FILE_MODE_SET_USER_IDENTIFIER uint16 = 1 << (POSIX_FILE_TYPE_SHIFT - 1)

// POSIX_FILE_MODE_SET_GROUP_IDENTIFIER encodes the POSIX set-group-ID bit.
const POSIX_FILE_MODE_SET_GROUP_IDENTIFIER uint16 = 1 << (POSIX_FILE_TYPE_SHIFT - 2)

// POSIX_FILE_MODE_STICKY encodes the POSIX sticky bit.
const POSIX_FILE_MODE_STICKY uint16 = 1 << (POSIX_FILE_TYPE_SHIFT - 3)

// Decoded_File_Mode is one portable mode reconstructed from a platform mode word. Its kind comes
// from a single POSIX nibble, thus it can never carry two kinds the way the portable domain
// permits, and its own maximum is a value a real stat can actually report.
type Decoded_File_Mode File_Mode

// FILE_MODE_DECODED_MINIMUM begins the decoded domain.
const FILE_MODE_DECODED_MINIMUM uint32 = 0

// FILE_MODE_DECODED_MAXIMUM closes it: directory is the numerically largest POSIX kind.
const FILE_MODE_DECODED_MAXIMUM uint32 = uint32(
	FILE_MODE_DIRECTORY | FILE_MODE_SET_USER_IDENTIFIER |
		FILE_MODE_SET_GROUP_IDENTIFIER | FILE_MODE_STICKY | FILE_MODE_PERMISSIONS,
)

// Decoded_File_Mode_Invariants bounds one mode a platform word can produce.
func Decoded_File_Mode_Invariants(value Decoded_File_Mode, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(
			uint32(value), FILE_MODE_DECODED_MINIMUM, FILE_MODE_DECODED_MAXIMUM,
		).
		Ensure()
}

// File_Mode_From_POSIX decodes one platform mode word. Linux and Darwin agree on every kind
// nibble and every high bit here, thus one decoder serves both and no platform file repeats it.
func File_Mode_From_POSIX(raw uint16) (mode Decoded_File_Mode) {
	mode = Decoded_File_Mode(File_Mode(raw) & FILE_MODE_PERMISSIONS)
	mode |= Decoded_File_Mode(file_mode_kind_from_posix(raw & POSIX_FILE_TYPE_MASK))
	if raw&POSIX_FILE_MODE_SET_USER_IDENTIFIER != 0 {
		mode |= Decoded_File_Mode(FILE_MODE_SET_USER_IDENTIFIER)
	}
	if raw&POSIX_FILE_MODE_SET_GROUP_IDENTIFIER != 0 {
		mode |= Decoded_File_Mode(FILE_MODE_SET_GROUP_IDENTIFIER)
	}
	if raw&POSIX_FILE_MODE_STICKY != 0 {
		mode |= Decoded_File_Mode(FILE_MODE_STICKY)
	}
	Decoded_File_Mode_Invariants(mode, "File_Mode_From_POSIX.mode")
	return mode
}

// Kind decode is its own function because the switch plus the flag decode would otherwise put
// one operand conversion past the function length cap.
func file_mode_kind_from_posix(kind uint16) (mode File_Mode) {
	switch kind {
	case POSIX_FILE_MODE_DEVICE:
		return FILE_MODE_DEVICE
	case POSIX_FILE_MODE_CHARACTER_DEVICE:
		return FILE_MODE_DEVICE | FILE_MODE_CHARACTER_DEVICE
	case POSIX_FILE_MODE_DIRECTORY:
		return FILE_MODE_DIRECTORY
	case POSIX_FILE_MODE_NAMED_PIPE:
		return FILE_MODE_NAMED_PIPE
	case POSIX_FILE_MODE_SYMBOLIC_LINK:
		return FILE_MODE_SYMBOLIC_LINK
	case POSIX_FILE_MODE_SOCKET:
		return FILE_MODE_SOCKET
	}
	return 0
}

// File_Mode_To_POSIX encodes one portable mode into the platform word. A mode naming no kind
// encodes as regular, because POSIX has no "kind absent" nibble.
func File_Mode_To_POSIX(mode File_Mode) (raw uint16) {
	File_Mode_Invariants(mode, "File_Mode_To_POSIX.mode")
	raw = file_mode_kind_to_posix(mode & FILE_MODE_TYPE)
	return raw | File_Permissions_To_POSIX(File_Permissions(mode)&FILE_PERMISSIONS_VALID)
}

// Kind encode is its own function so the caller stays one expression per concern.
func file_mode_kind_to_posix(kind File_Mode) (raw uint16) {
	switch kind {
	case FILE_MODE_DIRECTORY:
		return POSIX_FILE_MODE_DIRECTORY
	case FILE_MODE_SYMBOLIC_LINK:
		return POSIX_FILE_MODE_SYMBOLIC_LINK
	case FILE_MODE_NAMED_PIPE:
		return POSIX_FILE_MODE_NAMED_PIPE
	case FILE_MODE_SOCKET:
		return POSIX_FILE_MODE_SOCKET
	case FILE_MODE_DEVICE:
		return POSIX_FILE_MODE_DEVICE
	case FILE_MODE_DEVICE | FILE_MODE_CHARACTER_DEVICE:
		return POSIX_FILE_MODE_CHARACTER_DEVICE
	}
	return POSIX_FILE_MODE_REGULAR
}

// File_Permissions_To_POSIX encodes one creation operand. It emits no kind nibble: openat and
// mkdirat take permission bits only, and a kind bit there would be a mode the caller invented.
func File_Permissions_To_POSIX(permissions File_Permissions) (raw uint16) {
	File_Permissions_Invariants(permissions, "File_Permissions_To_POSIX.permissions")
	raw = uint16(permissions) & uint16(FILE_MODE_PERMISSIONS)
	if permissions&File_Permissions(FILE_MODE_SET_USER_IDENTIFIER) != 0 {
		raw |= POSIX_FILE_MODE_SET_USER_IDENTIFIER
	}
	if permissions&File_Permissions(FILE_MODE_SET_GROUP_IDENTIFIER) != 0 {
		raw |= POSIX_FILE_MODE_SET_GROUP_IDENTIFIER
	}
	if permissions&File_Permissions(FILE_MODE_STICKY) != 0 {
		raw |= POSIX_FILE_MODE_STICKY
	}
	return raw
}

// File_Mode_Is_Directory reports directory kind. Meaningful only where the mode came from an
// existing entry: a zero mode names a regular file with no permission, not an absent path.
func File_Mode_Is_Directory(mode File_Mode) (directory bool) {
	return mode&FILE_MODE_DIRECTORY != 0
}

// File_Mode_Is_Regular reports plain-file kind, which POSIX spells as no kind bit at all.
func File_Mode_Is_Regular(mode File_Mode) (regular bool) {
	return mode&FILE_MODE_TYPE == 0
}

// File_Mode_Is_Symbolic_Link reports link kind. Status never follows a final link, thus a
// walker sees this bit rather than the target metadata.
func File_Mode_Is_Symbolic_Link(mode File_Mode) (link bool) {
	return mode&FILE_MODE_SYMBOLIC_LINK != 0
}

// File_Status is what stat report: whether path exist, and if so its portable mode and byte
// length — metadata mirror consult before it read or write.
type File_Status struct {
	// Exists report whether path is present. Absent path is not error.
	Exists bool
	// Mode is portable permission and entry-kind metadata. Absent path leave it zero, thus
	// every kind question is meaningful only after Exists is true. Status never follows final
	// symbolic link, so walkers cannot cross it by accident.
	Mode File_Mode
	// Size is file length in bytes. Zero for directory or absent path.
	Size int64
}

// Connection_Refused is portable error returned when remote endpoint reject connect.
var Connection_Refused = errors.New("io: connection refused")

// Broken_Pipe is portable send result for socket whose send half is shut down.
var Broken_Pipe = errors.New("io: broken pipe")

// Socket_Not_Connected is returned when Shutdown is submitted before socket connect.
var Socket_Not_Connected = errors.New("io: socket not connected")

// Canceled is raw kernel operation result. It is
// not API to cancel operation: this surface deliberately has no generic Cancel.
var Canceled = errors.New("io: operation canceled")

// Path_Exists is portable Mkdir_At result for path already present. Make_Directory treat it as
// convergence, thus repeated create is not error.
var Path_Exists = errors.New("io: file exists")

// Not_Symbolic_Link reports Read_Link target path names another filesystem kind.
var Not_Symbolic_Link = errors.New("io: not a symbolic link")

// Symbolic_Link_Not_Followed is portable Open_At result when OPEN_AT_NO_FOLLOW name symbolic
// link in final path part. Kernel report ELOOP there, thus refusal is result, not silent follow.
var Symbolic_Link_Not_Followed = errors.New("io: symbolic link not followed")

// Number of virtual grains one simulated operation may take to complete, drawn from seed, thus
// completion order vary per run and still reproduce.
const SIM_LATENCY_GRAINS = 8

// One in this many simulated spawns exits non-zero, so a seed sweep exercises both the success
// and the failure path without a scripted outcome.
const SIM_SPAWN_FAIL_GRAINS = 4

// Span of wall-clock origins the seed draw a box's boot from, in seconds past the Unix epoch.
// One century cover every civil date a run can produce, and keep the draw inside a Bound.
const SIM_EPOCH_SECONDS = 100 * 365 * time.SECOND_COUNT_PER_DAY

// Bound on skew coefficient A — drift per tick, wobble amplitude, or step size — in grains.
// Same reach as latency, thus a skewed clock and a slow operation disagree by comparable spans.
const SIM_SKEW_MAGNITUDE_GRAINS = SIM_LATENCY_GRAINS

// Bound on skew coefficient B — wobble period, step onset, or initial offset — in ticks. Small
// enough that a run of a few hundred ticks see the onset and several periods.
const SIM_SKEW_PERIOD_GRAINS = 64

// SIM_SKEW_KIND_COUNT is the draw bound over the three deviation models; the kinds are
// consecutive from SKEW_KIND_LINEAR, which Skew_Kind_Invariants hold.
const SIM_SKEW_KIND_COUNT = prng.Bound(time.SKEW_KIND_STEP) + 1

// Returned when path resolve to nothing — simulator ENOENT.
var sim_file_absent = errors.New("io: no such file or directory")

// Returned when path component that must be directory is file.
var sim_not_a_directory = errors.New("io: not a directory")

// Returned when file operation name directory.
var sim_is_a_directory = errors.New("io: is a directory")

// A directory cannot cross the shared slice boundary when it is read back.
var sim_directory_full = errors.New("io: directory entry limit exceeded")

// SIM_PATH_COMPONENT_BYTES_MAXIMUM follows kernel component boundary so one node owns fixed name
// storage without accepting path that real backend rejects.
const SIM_PATH_COMPONENT_BYTES_MAXIMUM = int(bits.WORD_8_MAXIMUM)

// SIM_FILE_BYTES_MAXIMUM bounds one simulated file at generated distribution maximum.
const SIM_FILE_BYTES_MAXIMUM = SIM_SIZE_P100

// SIM_UMASK is permission mask simulated kernel apply at creation. Real kernel consult process
// umask, which this surface cannot read and must not. Simulator therefore state one value, thus
// run stay pure function of its seed rather than of ambient process state.
const SIM_UMASK File_Permissions = 0o022

// SIM_PERMISSION_DRAW_COUNT is every permission word one generated node may draw. Simulator
// store permissions but never enforce them: no simulated operation is refused for permission,
// because the modeled kernel has no user identity to check one against.
const SIM_PERMISSION_DRAW_COUNT = int(FILE_MODE_PERMISSIONS) + 1

// SIM_LINK_RESOLVE_MAXIMUM bounds symbolic-link hops in one path resolve, thus a link cycle
// reports an error rather than spins the loop thread.
const SIM_LINK_RESOLVE_MAXIMUM = 8

// SIM_ROOT_PERMISSIONS is what simulated root directory carry. Root predate every draw, thus it
// takes a stated value rather than a seeded one.
const SIM_ROOT_PERMISSIONS File_Permissions = 0o755

// SIM_LINK_CHANCE_DENOMINATOR sets how often one generated non-directory child is symbolic link
// instead of regular file. Kept low, thus most generated bytes still reach file contents.
const SIM_LINK_CHANCE_DENOMINATOR = 8

// Sim_Node is caller-owned storage for one generated or created filesystem node.
type Sim_Node struct {
	// Used separates free caller slots from seeded filesystem nodes.
	Used bool
	// Mode keeps node kind and stored permissions inline so lookup needs no interface value.
	// It is the one kind representation: no separate directory flag can disagree with it.
	Mode File_Mode
	// Parent names another caller slot so tree edges need no pointer allocation.
	Parent int
	// Name borrows exact caller storage so generated paths need no hidden allocation.
	Name []byte
	// Name_Count bounds valid bytes without making a slice escape.
	Name_Count int
	// Contents keeps file bytes inline so writes cannot grow heap storage. A symbolic link
	// stores its target here, the way POSIX filesystems already do, thus link needs no second
	// buffer and lstat size stays target length.
	Contents []byte
	// Contents_Count bounds valid bytes without reslicing stored state.
	Contents_Count int
}

// Sim_Socket is simulator caller-owned socket state. Shutdown is directional and never release
// ownership. Close is only operation that remove entry.
type Sim_Socket struct {
	// Family preserves address interpretation after descriptor creation.
	Family Address_Family
	// Datagram preserves transport without a second descriptor type.
	Datagram bool
	// Connected gates peer identity and directional shutdown.
	Connected bool
	// Listener gates Accept without a separate listener allocation.
	Listener bool
	// Address preserves kernel-selected port behavior in simulation.
	Address Address
	// Peer preserves remote identity without formatting it into allocated text.
	Peer Address
	// Receive_Shutdown makes later Receive resolve as EOF.
	Receive_Shutdown bool
	// Send_Shutdown makes later Send resolve as Broken_Pipe.
	Send_Shutdown bool
}

// Sim_Descriptor is caller-owned storage for one open file or socket descriptor.
type Sim_Descriptor struct {
	// Address_IP keeps bound-address bytes in caller storage after Bind input retires.
	Address_IP []byte
	// Peer_IP keeps connected peer bytes in caller storage after Connect input retires.
	Peer_IP []byte
	// Used separates free caller slots from owned descriptors.
	Used bool
	// File remains unique after a slot is recycled.
	File File
	// Node indexes caller-owned filesystem storage without a pointer escape.
	Node int
	// Socket selects inline socket state instead of node state.
	Socket bool
	// Directory_Drained preserves one-pass getdents behavior.
	Directory_Drained bool
	// Borrowed prevents Close from racing submitted operations.
	Borrowed int
	// Socket_State keeps transport lifetime inside descriptor storage.
	Socket_State Sim_Socket
}

// Sim_Operation is caller-owned storage retained until one asynchronous operation retires.
type Sim_Operation struct {
	// Address_IP keeps asynchronous address input in caller-owned operation storage.
	Address_IP []byte
	// Kind lets one static callback retire every operation shape.
	Kind Sim_Operation_Kind
	// State restores backend ownership without a captured callback.
	State *Sim
	// File identifies descriptor whose borrow must retire.
	File File
	// Node preserves file target even while work waits on timeline.
	Node int
	// Callback remains explicit because bound callback state allocates.
	Callback Callback
	// Entries borrows caller result slots until directory callback.
	Entries []Directory_Entry
	// Buffer borrows caller transfer bytes until callback.
	Buffer []byte
	// Offset preserves positioned transfer input until callback.
	Offset int64
	// Address preserves socket target until connection retirement.
	Address Address
	// Directory preserves path base until callback.
	Directory File
	// File_Path borrows caller path bytes until callback.
	File_Path string
	// Open_Options preserves open policy until callback.
	Open_Options Open_At_Options
	// Operation_Err stores seeded terminal result without a closure.
	Operation_Err error
	// Data stores seeded integer result without a closure.
	Data int
	// Result carries platform result pointer without platform type in common file.
	Result State
	// Mask preserves platform stat selection until callback.
	Mask uint32
	// Borrowed marks descriptor count that retirement must release.
	Borrowed bool
	// Signal preserves seed outcome until timeline reaches retirement grain.
	Signal Signal
	// Process preserves seed outcome until timeline reaches retirement grain.
	Process Process_Result
	// Signal_Callback avoids captured adapter state between arm and retirement.
	Signal_Callback Signal_Callback
	// Process_Callback avoids captured adapter state between arm and retirement.
	Process_Callback Process_Callback
}

// Sim_Operation_Kind makes one static retirement procedure decode bounded operation state.
type Sim_Operation_Kind uint8

// SIM_OPERATION_KIND_FREE identifies caller slot with no retained operation.
const SIM_OPERATION_KIND_FREE Sim_Operation_Kind = 0

// SIM_OPERATION_KIND_CLOSE releases one descriptor at retirement.
const SIM_OPERATION_KIND_CLOSE Sim_Operation_Kind = 1

// SIM_OPERATION_KIND_ACCEPT creates one descriptor at retirement.
const SIM_OPERATION_KIND_ACCEPT Sim_Operation_Kind = 2

// SIM_OPERATION_KIND_CONNECT mutates caller-owned socket state at retirement.
const SIM_OPERATION_KIND_CONNECT Sim_Operation_Kind = 3

// SIM_OPERATION_KIND_RECEIVE reads shutdown state at retirement.
const SIM_OPERATION_KIND_RECEIVE Sim_Operation_Kind = 4

// SIM_OPERATION_KIND_SEND reads shutdown state at retirement.
const SIM_OPERATION_KIND_SEND Sim_Operation_Kind = 5

// SIM_OPERATION_KIND_OPEN_AT resolves borrowed path at retirement.
const SIM_OPERATION_KIND_OPEN_AT Sim_Operation_Kind = 6

// SIM_OPERATION_KIND_MKDIR_AT resolves borrowed path at retirement.
const SIM_OPERATION_KIND_MKDIR_AT Sim_Operation_Kind = 7

// SIM_OPERATION_KIND_READ copies fixed node bytes at retirement.
const SIM_OPERATION_KIND_READ Sim_Operation_Kind = 8

// SIM_OPERATION_KIND_WRITE mutates fixed node bytes at retirement.
const SIM_OPERATION_KIND_WRITE Sim_Operation_Kind = 9

// SIM_OPERATION_KIND_FSYNC preserves descriptor borrow until retirement.
const SIM_OPERATION_KIND_FSYNC Sim_Operation_Kind = 10

// SIM_OPERATION_KIND_DIRECTORY fills caller result slots at retirement.
const SIM_OPERATION_KIND_DIRECTORY Sim_Operation_Kind = 11

// SIM_OPERATION_KIND_STATX fills Linux caller result at retirement.
const SIM_OPERATION_KIND_STATX Sim_Operation_Kind = 12

// SIM_OPERATION_KIND_SIGNAL makes static signal callback reject process state.
const SIM_OPERATION_KIND_SIGNAL Sim_Operation_Kind = 13

// SIM_OPERATION_KIND_PROCESS makes static process callback reject signal state.
const SIM_OPERATION_KIND_PROCESS Sim_Operation_Kind = 14

// Sim_Memory states each independent bounded simulator resource without scripting outcomes.
type Sim_Memory struct {
	// Nodes bounds generated and runtime-created filesystem state.
	Nodes []Sim_Node
	// Descriptors bounds simultaneous caller ownership.
	Descriptors []Sim_Descriptor
	// Operations bounds simultaneous submitted work.
	Operations []Sim_Operation
	// Queue bounds simultaneously armed completions on the timeline.
	Queue []*Completion
	// Events bounds simultaneously open cross-thread events.
	Events []Virtual_Event
	// Clocks bounds application views of the one counter; the constructor draw each skew.
	Clocks []Sim_Clock
}

// Sim hold mutable state of deterministic, in-memory IO backend.
//
// ===========================================================================
// DO NOT ADD A SCRIPTING API TO THIS TYPE. READ THIS BEFORE ADDING ANY FIELD.
// ===========================================================================
//
// "Scripting" mean any field or function that let caller pre-load specific outcome for sim to
// replay: Connect_Targets / Accept_Queue / Open_Targets maps, per-socket Inbound / Outbound /
// Connect_Error / Send_Error / Receive_Error fields, Receive_Script, Send_Limit, or Sim_Raise /
// Sim_Connect_Target / Sim_Open_Target helpers. Production io this port come from had all of
// them. We removed them on purpose. Do not bring them back, and do not add new one when test
// feel awkward.
//
// Why it is banned: simulation must be PURE FUNCTION OF ITS SEED. Every outcome — latency, which
// connect fail, bytes a receive deliver, grain a signal land on — is drawn from Generator,
// seeded by New_Simulated_IO. Invariant framework Always/Sometimes across many seeds assert
// correctness (plain seed sweep, like shared/vsr/simulation), and any failure reproduce by
// replay of its seed. Scripting destroy all three properties: run become function of
// author-chosen data instead of seed, thus (1) fuzzer only ever find bugs author already
// imagined, (2) seed no longer reproduce run, and (3) invariants end up judging hand-picked
// input instead of whole reachable space. Right way to influence run is to change SEED, never to
// inject data.
//
// New_Simulated_IO is only entry. Caller own Sim and finite memory because hidden ownership
// allocates. Those slots expose capacity, not outcomes: if you find yourself wanting to add a
// field that chooses an operation result, stop — that is scripting API trying to come back.
// Keep it shut.
type Sim struct {
	// Timeline is the one queue every simulated operation retire through, and the one counter
	// every Sim_Clock read. Sim own it; only the Driver the constructor return advance it.
	Timeline Virtual_Timeline
	// Clocks borrows caller capacity; each entry is one application's skewed view.
	Clocks []Sim_Clock
	// Generator excludes equal-time ordering so timeout races cannot perturb ordinary outcomes.
	Generator prng.Xoshiro
	// Timeout_Order_Generator isolates equal-time kernel ordering from operation outcomes.
	Timeout_Order_Generator prng.Xoshiro
	// Storage_Timeout_Order_Generator isolates storage ordering from the existing network
	// stream.
	Storage_Timeout_Order_Generator prng.Xoshiro
	// Signal_Generator isolates signal arrival from every transfer stream, thus a run that
	// arms one more read cannot move which grain a signal lands on.
	Signal_Generator prng.Xoshiro
	// Process_Generator isolates exit code and child latency the same way.
	Process_Generator prng.Xoshiro
	// Link_Generator isolates whether a generated node is a symbolic link, thus adding links
	// leaves the tree shape every banked seed already produces.
	Link_Generator prng.Xoshiro
	// Permission_Generator isolates generated permission bits from every other axis.
	Permission_Generator prng.Xoshiro
	// Clock_Generator isolates epoch and per-view skew from every transfer stream, thus one
	// more application clock cannot move any grain a banked seed already produce.
	Clock_Generator prng.Xoshiro
	// Next_File is synthetic descriptor counter. Listen, Accept, Open_Socket, Open, and Create
	// hand out next value, thus every descriptor is distinct.
	Next_File File
	// Nodes borrows caller capacity for complete simulator lifetime.
	Nodes []Sim_Node
	// Descriptors borrows caller capacity for complete simulator lifetime.
	Descriptors []Sim_Descriptor
	// Operations borrows caller capacity for complete simulator lifetime.
	Operations []Sim_Operation
}

// New_Simulated_IO return deterministic submit surface and the driver over it, seeded by seed.
// Resolution is the grain one tick advance; caller memory selects capacity; nothing else is
// input, thus seed is the only outcome input and a run reproduce exact. Epoch and every
// application clock's skew are drawn, never supplied. Driver stay in harness: program under
// test hold loop, NEVER pump (see Driver banner).
func New_Simulated_IO(
	state *Sim, seed uint64, resolution time.Duration, memory Sim_Memory,
) (loop IO, driver Driver) {
	aver.Always(state != nil, "A simulated IO backend has caller-owned state.")
	aver.Always(len(memory.Nodes) > 0, "A simulated IO backend has node capacity.")
	aver.Always(len(memory.Descriptors) > 0,
		"A simulated IO backend has descriptor capacity.")
	aver.Always(len(memory.Operations) > 0,
		"A simulated IO backend has operation capacity.")
	aver.Always(len(memory.Clocks) > 0, "A simulated IO backend has clock capacity.")
	sim_memory_initialize(memory)
	root_generator := prng.New(prng.Seed(seed))
	state.Generator = prng.Xoshiro_Split(&root_generator)
	state.Timeout_Order_Generator = prng.Xoshiro_Split(&root_generator)
	state.Storage_Timeout_Order_Generator = prng.Xoshiro_Split(&root_generator)
	// Split after every existing stream, thus adding an axis leaves the grain each older
	// stream draws for a given seed unchanged.
	state.Signal_Generator = prng.Xoshiro_Split(&root_generator)
	state.Process_Generator = prng.Xoshiro_Split(&root_generator)
	state.Link_Generator = prng.Xoshiro_Split(&root_generator)
	state.Permission_Generator = prng.Xoshiro_Split(&root_generator)
	state.Clock_Generator = prng.Xoshiro_Split(&root_generator)
	virtual_timeline_initialize(
		&state.Timeline, resolution, sim_epoch_draw(state), memory.Queue, memory.Events,
	)
	state.Clocks = memory.Clocks
	for index := range state.Clocks {
		state.Clocks[index] = Sim_Clock{
			Timeline: &state.Timeline, Skew: sim_skew_draw(state),
		}
	}
	state.Next_File = 0
	state.Nodes = memory.Nodes
	state.Descriptors = memory.Descriptors
	state.Operations = memory.Operations
	root := &state.Nodes[0]
	sim_node_reset(root)
	root.Used = true
	root.Mode = FILE_MODE_DIRECTORY | File_Mode(SIM_ROOT_PERMISSIONS)
	root.Parent = -1
	sim_generate(state)
	sim_wire_network(state, &loop.Network)
	sim_wire_storage(state, &loop.Storage)
	sim_wire_platform(state, &loop)
	loop.Timeline = virtual_timeline_to_timeline(&state.Timeline)
	loop.Close_Procedure = sim_close_procedure
	loop.Deinit_Procedure = sim_deinit_procedure
	loop.Watch_Signal = sim_watch_signal_procedure
	loop.Spawn = sim_spawn_procedure
	IO_Invariants(loop, "new_simulated_io.loop")
	Timeline_Invariants(loop.Timeline, "new_simulated_io.timeline")
	return loop, virtual_timeline_to_driver(&state.Timeline)
}

func sim_memory_initialize(memory Sim_Memory) {
	for index := range memory.Nodes {
		node := &memory.Nodes[index]
		aver.Always(len(node.Name) == SIM_PATH_COMPONENT_BYTES_MAXIMUM,
			"A simulated node has component-name storage.")
		aver.Always(len(node.Contents) == SIM_FILE_BYTES_MAXIMUM,
			"A simulated node has file-content storage.")
		sim_node_reset(node)
	}
	for index := range memory.Descriptors {
		descriptor := &memory.Descriptors[index]
		aver.Always(len(descriptor.Address_IP) == IPV6_ADDRESS_BYTES,
			"A simulated descriptor has local-address storage.")
		aver.Always(len(descriptor.Peer_IP) == IPV6_ADDRESS_BYTES,
			"A simulated descriptor has peer-address storage.")
		sim_descriptor_reset(descriptor)
	}
	for index := range memory.Operations {
		operation := &memory.Operations[index]
		aver.Always(len(operation.Address_IP) == IPV6_ADDRESS_BYTES,
			"A simulated operation has address storage.")
		sim_operation_reset(operation)
	}
}

// Draw the wall-clock origin of this box. A box's boot time is an outcome, not an input.
func sim_epoch_draw(state *Sim) (epoch time.Moment) {
	seconds := prng.Xoshiro_Below(&state.Clock_Generator, prng.Bound(SIM_EPOCH_SECONDS))
	return time.Moment(seconds) * time.Moment(time.SECOND)
}

// Draw one application clock's deviation model and both coefficients from the clock stream.
func sim_skew_draw(state *Sim) (skew time.Offset) {
	kind := time.Skew_Kind(prng.Xoshiro_Below(&state.Clock_Generator, SIM_SKEW_KIND_COUNT))
	magnitude := prng.Xoshiro_Below(&state.Clock_Generator, SIM_SKEW_MAGNITUDE_GRAINS)
	// Period and onset start at one: a zero period reads as no wobble, and a zero onset is a
	// step that already happened, so neither would exercise its model.
	period := 1 + prng.Xoshiro_Below(&state.Clock_Generator, SIM_SKEW_PERIOD_GRAINS)
	return time.Skew(kind, time.Duration(magnitude), time.Tick_Count(period))
}

func sim_watch_signal_procedure(
	state_pointer State, completion *Completion, signal Signal,
	deadline time.Duration, callback Signal_Callback,
) {
	aver.Always(deadline > 0, "A signal-watch deadline is positive and finite.")
	sim_watch_signal(state_pointer.(*Sim), completion, signal, deadline, callback)
}

func sim_spawn_procedure(
	state_pointer State, completion *Completion, _ Process_Request,
	deadline time.Duration, callback Process_Callback,
) {
	aver.Always(deadline > 0, "A spawn deadline is positive and finite.")
	sim_spawn(state_pointer.(*Sim), completion, deadline, callback)
}

// Watches for a signal that, in the simulation, arrives at a seed-drawn grain — the operating
// system event modeled as a seed outcome. It fires callback exactly once.
func sim_watch_signal(
	state *Sim, completion *Completion, signal Signal, deadline time.Duration,
	callback Signal_Callback,
) {
	// Callback slot stay nil: signal retire through its own typed static callback, thus the
	// shared Callback dispatcher never decodes this kind.
	operation := sim_operation_acquire(state, completion, SIM_OPERATION_KIND_SIGNAL, nil)
	if operation == nil {
		completion.Error = sim_operation_capacity_exceeded
		callback(completion, 0, sim_operation_capacity_exceeded)
		return
	}
	operation.Signal_Callback = callback
	latency := sim_latency_from(&state.Signal_Generator)
	if latency >= deadline {
		operation.Signal = SIGNAL_EXPIRED
		operation.Operation_Err = Deadline_Exceeded
		virtual_submit(&state.Timeline, completion, deadline, sim_signal_complete)
		return
	}
	operation.Signal = signal
	virtual_submit(&state.Timeline, completion, latency, sim_signal_complete)
}

func sim_signal_complete(completion Completion_Handle) {
	operation := completion.Backend.(*Sim_Operation)
	aver.Always(operation != nil,
		"A simulated signal completion owns specialized operation state.")
	aver.Always(operation.Kind == SIM_OPERATION_KIND_SIGNAL,
		"A simulated signal completion owns signal state.")
	callback := operation.Signal_Callback
	signal := operation.Signal
	err := operation.Operation_Err
	sim_operation_reset(operation)
	completion.Backend = nil
	callback(completion, signal, err)
}

// Delivers a subprocess result drawn from the seed: the exit code varies (usually zero,
// occasionally non-zero for fault coverage) with no captured output — scripted output is
// disallowed, so the seed decides success or failure, not a canned payload.
func sim_spawn(
	state *Sim, completion *Completion, deadline time.Duration,
	callback Process_Callback,
) {
	// Callback slot stay nil for the same reason the signal watch leaves it nil.
	operation := sim_operation_acquire(state, completion, SIM_OPERATION_KIND_PROCESS, nil)
	if operation == nil {
		completion.Error = sim_operation_capacity_exceeded
		callback(completion, Process_Result{}, sim_operation_capacity_exceeded)
		return
	}
	operation.Process_Callback = callback
	exit := 0
	if prng.Xoshiro_Below(&state.Process_Generator, SIM_SPAWN_FAIL_GRAINS) == 0 {
		exit = 1
	}
	latency := sim_latency_from(&state.Process_Generator)
	if latency >= deadline {
		operation.Operation_Err = Deadline_Exceeded
		virtual_submit(&state.Timeline, completion, deadline, sim_process_complete)
		return
	}
	operation.Process.Exit = exit
	virtual_submit(&state.Timeline, completion, latency, sim_process_complete)
}

func sim_process_complete(completion Completion_Handle) {
	operation := completion.Backend.(*Sim_Operation)
	aver.Always(operation != nil,
		"A simulated process completion owns specialized operation state.")
	aver.Always(operation.Kind == SIM_OPERATION_KIND_PROCESS,
		"A simulated process completion owns process state.")
	callback := operation.Process_Callback
	result := operation.Process
	err := operation.Operation_Err
	sim_operation_reset(operation)
	completion.Backend = nil
	callback(completion, result, err)
}

func sim_close_procedure(
	state_pointer State, completion *Completion, file File,
	callback Callback,
) {
	state := state_pointer.(*Sim)
	sim_assert_file_drained(state, file)
	operation := sim_operation_acquire(state, completion, SIM_OPERATION_KIND_CLOSE, callback)
	if operation == nil {
		return
	}
	operation.File = file
	sim_operation_submit(operation, completion, sim_latency(state))
}

func sim_deinit_procedure(state_pointer State) {
	state := state_pointer.(*Sim)
	open := false
	for index := range state.Descriptors {
		if state.Descriptors[index].Used {
			open = true
		}
	}
	aver.Always(!open, "Every descriptor a simulated run opened is closed before Deinit.")
}

// Wire whole socket half of simulator onto network.
func sim_wire_network(state *Sim, network *Network) {
	network.State = State(state)
	sim_wire_socket_lifecycle(network)
	sim_wire_socket_bytes(network)
}

// Wire whole file half of simulator onto storage.
func sim_wire_storage(state *Sim, storage *Storage) {
	storage.State = State(state)
	sim_wire_path(storage)
	sim_wire_file_bytes(storage)
	sim_wire_directory(storage)
}

// Wire socket lifecycle: descriptor a socket come from, options and address it carry, and
// direction shutdown disable.
func sim_wire_socket_lifecycle(network *Network) {
	network.Socket_TCP_Procedure = sim_socket_tcp
	network.Socket_UDP_Procedure = sim_socket_udp
	network.Bind_Procedure = sim_bind
	network.Listen_Socket_Procedure = sim_listen_socket
	network.Get_Socket_Name_Procedure = sim_get_socket_name
	network.Shutdown_Procedure = sim_shutdown_socket
	network.Peer_Address_Procedure = sim_peer_address_procedure
}

var sim_invalid_socket_options = errors.New("io: invalid socket options")
var sim_bind_socket_closed = errors.New("io: bind requires an open socket")
var sim_socket_name_closed = errors.New("io: getsockname requires an open socket")

func sim_socket_tcp(
	state_pointer State, family Address_Family, options TCP_Options,
) (socket File, err error) {
	if !tcp_options_valid(options) {
		return File(-1), sim_invalid_socket_options
	}
	return sim_open_socket(state_pointer.(*Sim), family, false)
}

func sim_socket_udp(
	state_pointer State, family Address_Family, options UDP_Options,
) (socket File, err error) {
	if !udp_options_valid(options) {
		return File(-1), sim_invalid_socket_options
	}
	return sim_open_socket(state_pointer.(*Sim), family, true)
}

func sim_bind(
	state_pointer State, socket File, address Address,
) (err error) {
	descriptor := sim_descriptor_find(state_pointer.(*Sim), socket)
	if descriptor == nil {
		return sim_bind_socket_closed
	}
	if !descriptor.Socket {
		return sim_bind_socket_closed
	}
	bound := sim_address_copy(descriptor.Address_IP, address)
	if bound.Port == 0 {
		bound.Port = uint16(socket)
	}
	descriptor.Socket_State.Address = bound
	return nil
}

func sim_listen_socket(
	state_pointer State, socket File, backlog uint32,
) (err error) {
	return sim_listen(state_pointer.(*Sim), socket, backlog)
}

func sim_get_socket_name(
	state_pointer State, socket File, destination *Address,
) (err error) {
	storage, storage_err := address_destination_storage(destination)
	if storage_err != nil {
		return storage_err
	}
	descriptor := sim_descriptor_find(state_pointer.(*Sim), socket)
	if descriptor == nil {
		return sim_socket_name_closed
	}
	if !descriptor.Socket {
		return sim_socket_name_closed
	}
	*destination = sim_address_copy(storage, descriptor.Socket_State.Address)
	return nil
}

func sim_shutdown_socket(
	state_pointer State, socket File, how Shutdown_How,
) (err error) {
	return sim_shutdown(state_pointer.(*Sim), socket, how)
}

func sim_peer_address_procedure(
	state_pointer State, file File, destination *Address,
) (err error) {
	storage, storage_err := address_destination_storage(destination)
	if storage_err != nil {
		return storage_err
	}
	*destination = sim_address_copy(storage, sim_peer_address(state_pointer.(*Sim), file))
	return nil
}

// SOCKET_RECEIVE_LOW_WATER_BYTES_DEFAULT uses one byte to keep readiness enabled.
const SOCKET_RECEIVE_LOW_WATER_BYTES_DEFAULT uint32 = 1

// SOCKET_LINGER_TIMEOUT_DEFAULT bounds close without a long descriptor hold.
const SOCKET_LINGER_TIMEOUT_DEFAULT time.Duration = 1 * time.SECOND

// TCP_NO_DELAY_DEFAULT disables packet coalescing to prevent its added send delay.
const TCP_NO_DELAY_DEFAULT bool = true

// SOCKET_OPTION_INTEGER_MAXIMUM keeps an integer option positive in each implementation.
const SOCKET_OPTION_INTEGER_MAXIMUM uint32 = uint32(bits.INTEGER_32_MAXIMUM)

// TCP_MAXIMUM_SEGMENT_BYTES_MAXIMUM is the largest value the TCP MSS field can represent.
const TCP_MAXIMUM_SEGMENT_BYTES_MAXIMUM uint32 = uint32(bits.WORD_16_MAXIMUM)

// TCP_KEEPALIVE_PROBE_COUNT_MAXIMUM is the largest portable probe count.
const TCP_KEEPALIVE_PROBE_COUNT_MAXIMUM uint32 = uint32(bits.INTEGER_8_MAXIMUM)

// Reject an invalid selected override before simulation diverges from the operating system.
func tcp_options_valid(options TCP_Options) (valid bool) {
	if !socket_options_valid(
		options.Receive_Buffer_Bytes,
		options.Send_Buffer_Bytes,
		options.Receive_Low_Water_Bytes,
		options.Linger_Timeout,
	) {
		return false
	}
	if options.Maximum_Segment_Bytes > TCP_MAXIMUM_SEGMENT_BYTES_MAXIMUM {
		return false
	}
	if options.Maximum_Segment_Bytes == 0 {
		return false
	}
	if !socket_option_integer_valid(options.Not_Sent_Low_Water_Bytes) {
		return false
	}
	return tcp_keepalive_valid(options.Keepalive)
}

// Reject an invalid selected override before simulation diverges from the operating system.
func udp_options_valid(options UDP_Options) (valid bool) {
	return socket_options_valid(
		options.Receive_Buffer_Bytes,
		options.Send_Buffer_Bytes,
		options.Receive_Low_Water_Bytes,
		options.Linger_Timeout,
	)
}

// Reject each disabled limit before the simulator makes a descriptor.
func socket_options_valid(
	receive_buffer_bytes uint32,
	send_buffer_bytes uint32,
	receive_low_water_bytes uint32,
	linger_timeout time.Duration,
) (valid bool) {
	if !socket_option_integer_valid(receive_buffer_bytes) {
		return false
	}
	if !socket_option_integer_valid(send_buffer_bytes) {
		return false
	}
	if !socket_option_integer_valid(receive_low_water_bytes) {
		return false
	}
	return socket_linger_valid(linger_timeout)
}

// The syscall takes a positive signed integer, thus zero cannot enable a limit.
func socket_option_integer_valid(value uint32) (valid bool) {
	return value > 0 && value <= SOCKET_OPTION_INTEGER_MAXIMUM
}

// Linger has a signed whole-second field on both supported kernels.
func socket_linger_valid(value time.Duration) (valid bool) {
	if !socket_seconds_valid(value) {
		return false
	}
	return value/time.SECOND <= time.Duration(SOCKET_OPTION_INTEGER_MAXIMUM)
}

// A positive whole-second value keeps the limit enabled on each backend.
func socket_seconds_valid(value time.Duration) (valid bool) {
	return value > 0 && value%time.SECOND == 0
}

// Validate all fields because direct ownership makes every keepalive limit mandatory.
func tcp_keepalive_valid(keepalive TCP_Keepalive) (valid bool) {
	if !socket_seconds_valid(keepalive.Idle) {
		return false
	}
	if keepalive.Idle/time.SECOND > time.Duration(SOCKET_OPTION_INTEGER_MAXIMUM) {
		return false
	}
	if !socket_seconds_valid(keepalive.Interval) {
		return false
	}
	if keepalive.Interval/time.SECOND > time.Duration(SOCKET_OPTION_INTEGER_MAXIMUM) {
		return false
	}
	return keepalive.Probe_Count > 0 &&
		keepalive.Probe_Count <= TCP_KEEPALIVE_PROBE_COUNT_MAXIMUM
}

// Wire socket transfers without bound closures because each operation may outlive submission.
func sim_wire_socket_bytes(network *Network) {
	network.Accept_Procedure = sim_accept_procedure
	network.Connect_Procedure = sim_connect_procedure
	network.Receive_Procedure = sim_receive_procedure
	network.Send_Procedure = sim_send_procedure
}

var sim_listen_socket_required = errors.New("io: listen requires an open TCP socket")
var sim_listen_backlog_required = errors.New("io: listen backlog must be positive")
var sim_socket_listener_required = errors.New("io: socket is not listening")
var sim_descriptor_capacity_exceeded = errors.New("io: descriptor capacity exceeded")
var sim_operation_capacity_exceeded = errors.New("io: operation capacity exceeded")
var sim_node_capacity_exceeded = errors.New("io: node capacity exceeded")
var sim_path_component_too_large = errors.New("io: path component too large")
var sim_file_capacity_exceeded = errors.New("io: file capacity exceeded")
var sim_file_offset_invalid = errors.New("io: file offset invalid")

func sim_accept_procedure(
	state_pointer State, completion *Completion, listener File,
	timeout time.Duration, callback Callback,
) {
	state := state_pointer.(*Sim)
	aver.Always(timeout > 0, "An accept timeout is positive and finite.")
	latency := sim_latency(state)
	operation := sim_operation_acquire(state, completion, SIM_OPERATION_KIND_ACCEPT, callback)
	if operation == nil {
		return
	}
	operation.File = listener
	sim_operation_borrow(operation)
	if sim_timeout_first(state, latency, timeout) {
		operation.Operation_Err = Deadline_Exceeded
		operation.Data = int(File(-1))
		sim_operation_submit(operation, completion, timeout)
		return
	}
	sim_operation_submit(operation, completion, latency)
}

func sim_connect_procedure(
	state_pointer State, completion *Completion, socket File, address Address,
	timeout time.Duration, callback Callback,
) {
	state := state_pointer.(*Sim)
	aver.Always(timeout > 0, "A connect timeout is positive and finite.")
	operation := sim_operation_acquire(state, completion, SIM_OPERATION_KIND_CONNECT, callback)
	if operation == nil {
		return
	}
	operation.File = socket
	operation.Address = sim_address_copy(operation.Address_IP, address)
	sim_operation_borrow(operation)
	if prng.Xoshiro_Below(&state.Generator, 4) == 0 {
		operation.Operation_Err = Connection_Refused
	}
	latency := sim_latency(state)
	if sim_timeout_first(state, latency, timeout) {
		operation.Operation_Err = Deadline_Exceeded
		sim_operation_submit(operation, completion, timeout)
		return
	}
	sim_operation_submit(operation, completion, latency)
}

func sim_receive_procedure(
	state_pointer State, completion *Completion, socket File, buffer []byte,
	timeout time.Duration, callback Callback,
) {
	state := state_pointer.(*Sim)
	aver.Always(timeout > 0, "A receive timeout is positive and finite.")
	operation := sim_operation_acquire(state, completion, SIM_OPERATION_KIND_RECEIVE, callback)
	if operation == nil {
		return
	}
	operation.File = socket
	operation.Buffer = buffer
	sim_operation_borrow(operation)
	latency := sim_latency(state)
	if sim_timeout_first(state, latency, timeout) {
		operation.Operation_Err = Deadline_Exceeded
		sim_operation_submit(operation, completion, timeout)
		return
	}
	sim_operation_submit(operation, completion, latency)
}

func sim_send_procedure(
	state_pointer State, completion *Completion, socket File, buffer []byte,
	timeout time.Duration, callback Callback,
) {
	state := state_pointer.(*Sim)
	aver.Always(timeout > 0, "A send timeout is positive and finite.")
	operation := sim_operation_acquire(state, completion, SIM_OPERATION_KIND_SEND, callback)
	if operation == nil {
		return
	}
	operation.File = socket
	operation.Buffer = buffer
	sim_operation_borrow(operation)
	latency := sim_latency(state)
	if sim_timeout_first(state, latency, timeout) {
		operation.Operation_Err = Deadline_Exceeded
		sim_operation_submit(operation, completion, timeout)
		return
	}
	sim_operation_submit(operation, completion, latency)
}

// Mark socket accepting only after transport and backlog satisfy kernel requirements.
func sim_listen(state *Sim, socket File, backlog uint32) (err error) {
	descriptor := sim_descriptor_find(state, socket)
	if descriptor == nil {
		return sim_listen_socket_required
	}
	if !descriptor.Socket {
		return sim_listen_socket_required
	}
	if descriptor.Socket_State.Datagram {
		return sim_listen_socket_required
	}
	if backlog == 0 {
		return sim_listen_backlog_required
	}
	descriptor.Socket_State.Listener = true
	return nil
}

// Disable selected direction without releasing descriptor ownership.
func sim_shutdown(state *Sim, socket File, how Shutdown_How) (err error) {
	descriptor := sim_descriptor_find(state, socket)
	if descriptor == nil {
		return Socket_Not_Connected
	}
	if !descriptor.Socket {
		return Socket_Not_Connected
	}
	if !descriptor.Socket_State.Connected {
		return Socket_Not_Connected
	}
	if how == SHUTDOWN_RECEIVE {
		descriptor.Socket_State.Receive_Shutdown = true
	}
	if how == SHUTDOWN_BOTH {
		descriptor.Socket_State.Receive_Shutdown = true
	}
	if how == SHUTDOWN_SEND {
		descriptor.Socket_State.Send_Shutdown = true
	}
	if how == SHUTDOWN_BOTH {
		descriptor.Socket_State.Send_Shutdown = true
	}
	return nil
}

// Wire path primitives to static procedures so path views remain caller-owned until retirement.
func sim_wire_path(storage *Storage) {
	storage.Open_At_Procedure = sim_open_at_procedure
	storage.Mkdir_At_Procedure = sim_mkdir_at_procedure
}

func sim_open_at_procedure(
	state_pointer State, completion *Completion, directory File,
	file_path string, options Open_At_Options, callback Callback,
) {
	state := state_pointer.(*Sim)
	aver.Always(options.Flags&^OPEN_AT_NO_FOLLOW == 0,
		"Open_At options contain only known flags.")
	operation := sim_operation_acquire(state, completion, SIM_OPERATION_KIND_OPEN_AT, callback)
	if operation == nil {
		return
	}
	operation.Directory = directory
	operation.File_Path = file_path
	operation.Open_Options = options
	sim_operation_submit(operation, completion, sim_latency(state))
}

func sim_mkdir_at_procedure(
	state_pointer State, completion *Completion, directory File,
	file_path string, permissions File_Permissions, callback Callback,
) {
	state := state_pointer.(*Sim)
	operation := sim_operation_acquire(state, completion, SIM_OPERATION_KIND_MKDIR_AT, callback)
	if operation == nil {
		return
	}
	operation.Directory = directory
	operation.File_Path = file_path
	// Mkdir has no other options, thus it borrows the open operand slot rather than making the
	// operation record carry a second permission field only one kind would ever fill.
	operation.Open_Options = Open_At_Options{Permissions: permissions}
	sim_operation_submit(operation, completion, sim_latency(state))
}

// Wire file transfers to caller-owned operation records retained through timeout races.
func sim_wire_file_bytes(storage *Storage) {
	storage.Read_Procedure = sim_storage_read
	storage.Write_Procedure = sim_storage_write
	storage.Fsync_Procedure = sim_storage_fsync
}

func sim_storage_read(
	state_pointer State, completion *Completion, file File, buffer []byte,
	offset int64, timeout time.Duration, callback Callback,
) {
	state := state_pointer.(*Sim)
	aver.Always(timeout > 0, "A storage read timeout is positive and finite.")
	descriptor := sim_descriptor_find(state, file)
	aver.Always(descriptor != nil, "A Storage read names an open descriptor.")
	aver.Always(!descriptor.Socket, "A Storage read names an open file.")
	operation := sim_operation_acquire(state, completion, SIM_OPERATION_KIND_READ, callback)
	if operation == nil {
		return
	}
	operation.File = file
	operation.Node = descriptor.Node
	operation.Buffer = buffer
	operation.Offset = offset
	sim_operation_borrow(operation)
	if len(buffer) == 0 {
		sim_operation_submit(operation, completion, 0)
		return
	}
	latency := sim_latency(state)
	if sim_storage_timeout_first(state, latency, timeout) {
		operation.Operation_Err = Deadline_Exceeded
		sim_operation_submit(operation, completion, timeout)
		return
	}
	sim_operation_submit(operation, completion, latency)
}

func sim_storage_write(
	state_pointer State, completion *Completion, file File, buffer []byte,
	offset int64, timeout time.Duration, callback Callback,
) {
	state := state_pointer.(*Sim)
	aver.Always(timeout > 0, "A storage write timeout is positive and finite.")
	descriptor := sim_descriptor_find(state, file)
	aver.Always(descriptor != nil, "A Storage write names an open descriptor.")
	aver.Always(!descriptor.Socket, "A Storage write names an open file.")
	operation := sim_operation_acquire(state, completion, SIM_OPERATION_KIND_WRITE, callback)
	if operation == nil {
		return
	}
	operation.File = file
	operation.Node = descriptor.Node
	operation.Buffer = buffer
	operation.Offset = offset
	sim_operation_borrow(operation)
	if len(buffer) == 0 {
		sim_operation_submit(operation, completion, 0)
		return
	}
	latency := sim_latency(state)
	if sim_storage_timeout_first(state, latency, timeout) {
		operation.Operation_Err = Deadline_Exceeded
		sim_operation_submit(operation, completion, timeout)
		return
	}
	sim_operation_submit(operation, completion, latency)
}

func sim_storage_fsync(
	state_pointer State, completion *Completion, file File,
	timeout time.Duration, callback Callback,
) {
	state := state_pointer.(*Sim)
	aver.Always(timeout > 0, "A storage fsync timeout is positive and finite.")
	descriptor := sim_descriptor_find(state, file)
	aver.Always(descriptor != nil, "A Storage fsync names an open descriptor.")
	aver.Always(!descriptor.Socket, "A Storage fsync names an open file.")
	operation := sim_operation_acquire(state, completion, SIM_OPERATION_KIND_FSYNC, callback)
	if operation == nil {
		return
	}
	operation.File = file
	sim_operation_borrow(operation)
	latency := sim_latency(state)
	if sim_storage_timeout_first(state, latency, timeout) {
		operation.Operation_Err = Deadline_Exceeded
		sim_operation_submit(operation, completion, timeout)
		return
	}
	sim_operation_submit(operation, completion, latency)
}

// Directory result occupies caller slots; Completion.Data states populated count.
func sim_wire_directory(storage *Storage) {
	storage.Get_Directory_Entries_Procedure = sim_get_directory_entries
	storage.Status_Procedure = sim_status_procedure
	storage.Read_Link_Procedure = sim_read_link_procedure
}

func sim_get_directory_entries(
	state_pointer State, completion *Completion, directory File, _ []byte,
	entries []Directory_Entry, callback Callback,
) {
	state := state_pointer.(*Sim)
	operation := sim_operation_acquire(
		state, completion, SIM_OPERATION_KIND_DIRECTORY, callback,
	)
	if operation == nil {
		return
	}
	operation.File = directory
	operation.Entries = entries
	sim_operation_borrow(operation)
	sim_operation_submit(operation, completion, sim_latency(state))
}

func sim_status_procedure(
	state_pointer State, path string,
) (status File_Status, err error) {
	return sim_status(state_pointer.(*Sim), path), nil
}

func sim_read_link_procedure(
	state_pointer State, path string, destination []byte,
) (count int, err error) {
	state := state_pointer.(*Sim)
	node_index, found := sim_resolve_no_follow(state, path)
	if !found {
		return 0, sim_file_absent
	}
	node := &state.Nodes[node_index]
	if !File_Mode_Is_Symbolic_Link(node.Mode) {
		return 0, Not_Symbolic_Link
	}
	return copy(destination, node.Contents[:node.Contents_Count]), nil
}

// Close cannot release descriptor while any submitted operation still holds it.
func sim_assert_file_drained(state *Sim, file File) {
	descriptor := sim_descriptor_find(state, file)
	aver.Always(descriptor != nil, "A Close names an open descriptor.")
	aver.Always(descriptor.Borrowed == 0,
		"A descriptor is drained before Close releases it.")
}

// Descriptor slots bound simulator ownership without maps or per-open heap state.
func sim_descriptor_acquire(
	state *Sim, node int, socket bool, socket_state Sim_Socket,
) (descriptor *Sim_Descriptor, err error) {
	for index := range state.Descriptors {
		if state.Descriptors[index].Used {
			continue
		}
		state.Next_File++
		descriptor = &state.Descriptors[index]
		sim_descriptor_reset(descriptor)
		descriptor.Used = true
		descriptor.File = state.Next_File
		descriptor.Node = node
		descriptor.Socket = socket
		descriptor.Socket_State = socket_state
		return descriptor, nil
	}
	return nil, sim_descriptor_capacity_exceeded
}

func sim_descriptor_find(state *Sim, file File) (descriptor *Sim_Descriptor) {
	for index := range state.Descriptors {
		candidate := &state.Descriptors[index]
		if candidate.Used {
			if candidate.File == file {
				return candidate
			}
		}
	}
	return nil
}

func sim_descriptor_release(state *Sim, file File) {
	descriptor := sim_descriptor_find(state, file)
	if descriptor != nil {
		sim_descriptor_reset(descriptor)
	}
}

// Reset keeps caller-provided address backing while descriptor ownership changes.
func sim_descriptor_reset(descriptor *Sim_Descriptor) {
	address_ip := descriptor.Address_IP
	peer_ip := descriptor.Peer_IP
	*descriptor = Sim_Descriptor{Address_IP: address_ip, Peer_IP: peer_ip}
}

// Copy breaks retained socket identity from caller mutation after operation retirement.
func sim_address_copy(storage []byte, address Address) (copied Address) {
	if len(address.IP) == 0 {
		return Address{IP: storage[:0:IPV6_ADDRESS_BYTES]}
	}
	size := IPV4_ADDRESS_BYTES
	if address.Family == FAMILY_IPV6 {
		size = IPV6_ADDRESS_BYTES
	}
	aver.Always(len(address.IP) == size, "A socket address matches its family size.")
	copy(storage[:size], address.IP)
	return Address{
		Family: address.Family, IP: storage[:size:IPV6_ADDRESS_BYTES], Port: address.Port,
	}
}

// Open socket state inside descriptor slot so Socket constructors allocate nothing.
func sim_open_socket(
	state *Sim, family Address_Family, datagram bool,
) (socket File, err error) {
	descriptor, acquire_err := sim_descriptor_acquire(
		state, -1, true, Sim_Socket{Family: family, Datagram: datagram},
	)
	if acquire_err != nil {
		return File(-1), acquire_err
	}
	return descriptor.File, nil
}

// SIM_PATH_TEXT_BYTES_MAXIMUM keeps one simulated path within repository text budget.
const SIM_PATH_TEXT_BYTES_MAXIMUM = 4 * bits.KIBIBYTE_BYTES

// Resolve path, following every symbolic link including the final one, the way open does.
func sim_resolve(state *Sim, path string) (node_index int, found bool) {
	return sim_walk(state, 0, path, true, 0)
}

// Resolve path without following a final symbolic link, the way lstat does. Links in earlier
// components still resolve, because a kernel has no way to name a path through an unfollowed one.
func sim_resolve_no_follow(state *Sim, path string) (node_index int, found bool) {
	return sim_walk(state, 0, path, false, 0)
}

// Walk path by scanning input views directly; component collection would allocate. Fixed
// continuation stack turns nested links into bounded iteration.
func sim_walk(
	state *Sim, start_node int, path string, follow_final bool, hops int,
) (node_index int, found bool) {
	aver.Always(len(path) <= SIM_PATH_TEXT_BYTES_MAXIMUM,
		"A simulated path stays inside the repository text budget.")
	type walk_frame struct {
		Path         string
		Follow_Final bool
	}
	frames := [SIM_LINK_RESOLVE_MAXIMUM]walk_frame{}
	frame_count := 0
	node_index = start_node
Path:
	for frame_count <= len(frames) {
		leaf_start := sim_path_leaf_start(path)
		start := 0
		for position := 0; position <= len(path); position++ {
			if position < len(path) {
				if path[position] != '/' {
					continue
				}
			}
			if position > start {
				step, link, step_found := sim_walk_step(
					state, node_index, path[start:position],
					start == leaf_start, follow_final,
				)
				if !step_found {
					return 0, false
				}
				if link {
					hops++
					if hops > SIM_LINK_RESOLVE_MAXIMUM {
						return 0, false
					}
					if position < len(path) {
						if frame_count == len(frames) {
							return 0, false
						}
						frames[frame_count] = walk_frame{
							Path:         path[position+1:],
							Follow_Final: follow_final,
						}
						frame_count++
					}
					node := &state.Nodes[step]
					var path_found bool
					path, path_found = sim_symbolic_link_target(node)
					if !path_found {
						return 0, false
					}
					follow_final = true
					if path[0] == '/' {
						node_index = 0
					} else {
						node_index = node.Parent
					}
					continue Path
				}
				node_index = step
			}
			start = position + 1
		}
		if frame_count == 0 {
			return node_index, true
		}
		frame_count--
		path = frames[frame_count].Path
		follow_final = frames[frame_count].Follow_Final
	}
	return 0, false
}

// Keep the target as a view because allocation changes simulation behavior.
func sim_symbolic_link_target(node *Sim_Node) (target string, found bool) {
	if node.Contents_Count == 0 {
		return "", false
	}
	target = unsafe.String(&node.Contents[0], node.Contents_Count)
	aver.Always(
		len(target) <= SIM_PATH_TEXT_BYTES_MAXIMUM,
		"A simulated symbolic-link target stays inside path budget.",
	)
	return target, true
}

// Take one path component. Dot forms move inside the tree without a lookup, and a link resolves
// unless it is the final component under a no-follow policy.
func sim_walk_step(
	state *Sim, node_index int, component string, final bool, follow_final bool,
) (step int, link bool, found bool) {
	if component == "." {
		return node_index, false, true
	}
	if component == ".." {
		if node_index == 0 {
			return node_index, false, true
		}
		return state.Nodes[node_index].Parent, false, true
	}
	if !File_Mode_Is_Directory(state.Nodes[node_index].Mode) {
		return 0, false, false
	}
	child, child_found := sim_node_find_child(state, node_index, component)
	if !child_found {
		return 0, false, false
	}
	if !File_Mode_Is_Symbolic_Link(state.Nodes[child].Mode) {
		return child, false, true
	}
	if final {
		if !follow_final {
			return child, false, true
		}
	}
	return child, true, true
}

// Locate final component start, thus one walk applies the no-follow policy to exactly that
// component and follows every earlier link.
func sim_path_leaf_start(path string) (leaf_start int) {
	leaf_start = -1
	start := 0
	for position := 0; position <= len(path); position++ {
		if position < len(path) {
			if path[position] != '/' {
				continue
			}
		}
		if position > start {
			leaf_start = start
		}
		start = position + 1
	}
	return leaf_start
}

// Resolve parent and retain leaf as input view so create and mkdir need no temporary list.
func sim_path_parent(
	state *Sim, path string,
) (parent int, leaf string, found bool) {
	aver.Always(len(path) <= SIM_PATH_TEXT_BYTES_MAXIMUM,
		"A simulated parent path stays inside the repository text budget.")
	leaf_start := -1
	leaf_end := -1
	start := 0
	for position := 0; position <= len(path); position++ {
		if position < len(path) {
			if path[position] != '/' {
				continue
			}
		}
		if position > start {
			leaf_start = start
			leaf_end = position
		}
		start = position + 1
	}
	if leaf_start < 0 {
		return 0, "", false
	}
	parent, found = sim_resolve(state, path[:leaf_start])
	return parent, path[leaf_start:leaf_end], found
}

func sim_node_find_child(
	state *Sim, parent int, name string,
) (node_index int, found bool) {
	for index := range state.Nodes {
		node := &state.Nodes[index]
		if !node.Used {
			continue
		}
		if node.Parent != parent {
			continue
		}
		if node.Name_Count != len(name) {
			continue
		}
		matched := true
		for position := range name {
			if node.Name[position] != name[position] {
				matched = false
				break
			}
		}
		if matched {
			return index, true
		}
	}
	return 0, false
}

// Mode reaches here already bounded by Storage_Open_At or Storage_Mkdir_At, thus repeating that
// assertion would only register a domain no simulated node can span.
func sim_node_acquire(
	state *Sim, parent int, name string, mode File_Mode,
) (node_index int, err error) {
	if len(name) > SIM_PATH_COMPONENT_BYTES_MAXIMUM {
		return 0, sim_path_component_too_large
	}
	for index := 1; index < len(state.Nodes); index++ {
		if state.Nodes[index].Used {
			continue
		}
		node := &state.Nodes[index]
		sim_node_reset(node)
		node.Used = true
		node.Mode = mode
		node.Parent = parent
		node.Name_Count = len(name)
		copy(node.Name, name)
		return index, nil
	}
	return 0, sim_node_capacity_exceeded
}

// Make exactly one directory; parent creation belongs to higher composition.
func sim_mkdir(state *Sim, path string, permissions File_Permissions) (err error) {
	parent, leaf, found := sim_path_parent(state, path)
	if leaf == "" {
		return Path_Exists
	}
	if !found {
		return sim_file_absent
	}
	if !File_Mode_Is_Directory(state.Nodes[parent].Mode) {
		return sim_not_a_directory
	}
	if _, present := sim_node_find_child(state, parent, leaf); present {
		return Path_Exists
	}
	mode := FILE_MODE_DIRECTORY | File_Mode(permissions&^SIM_UMASK)
	_, err = sim_node_acquire(state, parent, leaf, mode)
	return err
}

// Status reads fixed node storage and returns no collection. It never follows a final symbolic
// link, thus a walker sees the link itself the way lstat reports one.
func sim_status(state *Sim, path string) (status File_Status) {
	node_index, found := sim_resolve_no_follow(state, path)
	if !found {
		return File_Status{}
	}
	node := &state.Nodes[node_index]
	return File_Status{
		Exists: true, Mode: node.Mode, Size: int64(node.Contents_Count),
	}
}

// Directory name views stable caller-owned node bytes until simulator state is reinitialized.
func sim_node_name(node *Sim_Node) (name string) {
	return unsafe.String(&node.Name[0], node.Name_Count)
}

// One pass fills caller slots and sorts them because node slot order is ownership order, not name.
func sim_directory_pass(
	state *Sim, directory File, entries []Directory_Entry,
) (entry_count int, err error) {
	descriptor := sim_descriptor_find(state, directory)
	if descriptor == nil {
		return 0, sim_not_a_directory
	}
	if descriptor.Socket {
		return 0, sim_not_a_directory
	}
	node := &state.Nodes[descriptor.Node]
	if !File_Mode_Is_Directory(node.Mode) {
		return 0, sim_not_a_directory
	}
	if descriptor.Directory_Drained {
		return 0, nil
	}
	descriptor.Directory_Drained = true
	for index := range state.Nodes {
		child := &state.Nodes[index]
		if !child.Used {
			continue
		}
		if child.Parent != descriptor.Node {
			continue
		}
		if entry_count == len(entries) {
			return 0, sim_directory_full
		}
		entries[entry_count] = Directory_Entry{
			Name:         sim_node_name(child),
			Is_Directory: File_Mode_Is_Directory(child.Mode),
		}
		entry_count++
	}
	for index := 1; index < entry_count; index++ {
		entry := entries[index]
		position := index
		for position > 0 && entry.Name < entries[position-1].Name {
			entries[position] = entries[position-1]
			position--
		}
		entries[position] = entry
	}
	return entry_count, nil
}

// Open existing node and bind one caller-owned descriptor slot.
func sim_open(state *Sim, path string, flags Open_At_Flags) (file File, err error) {
	if flags&OPEN_AT_NO_FOLLOW != 0 {
		return sim_open_no_follow(state, path)
	}
	node, found := sim_resolve(state, path)
	if !found {
		return File(-1), sim_file_absent
	}
	descriptor, acquire_err := sim_descriptor_acquire(state, node, false, Sim_Socket{})
	if acquire_err != nil {
		return File(-1), acquire_err
	}
	return descriptor.File, nil
}

// Refuse final symbolic link the way a kernel refuses O_NOFOLLOW, rather than opening target.
func sim_open_no_follow(state *Sim, path string) (file File, err error) {
	node, found := sim_resolve_no_follow(state, path)
	if !found {
		return File(-1), sim_file_absent
	}
	if File_Mode_Is_Symbolic_Link(state.Nodes[node].Mode) {
		return File(-1), Symbolic_Link_Not_Followed
	}
	descriptor, acquire_err := sim_descriptor_acquire(state, node, false, Sim_Socket{})
	if acquire_err != nil {
		return File(-1), acquire_err
	}
	return descriptor.File, nil
}

// Create or truncate file inside fixed node storage, then bind descriptor slot.
func sim_create(
	state *Sim, path string, permissions File_Permissions,
) (file File, err error) {
	node, create_err := sim_create_file(state, path, permissions)
	if create_err != nil {
		return File(-1), create_err
	}
	descriptor, acquire_err := sim_descriptor_acquire(state, node, false, Sim_Socket{})
	if acquire_err != nil {
		return File(-1), acquire_err
	}
	return descriptor.File, nil
}

func sim_create_file(
	state *Sim, path string, permissions File_Permissions,
) (node_index int, err error) {
	parent, leaf, found := sim_path_parent(state, path)
	if leaf == "" {
		return 0, sim_is_a_directory
	}
	if !found {
		return 0, sim_file_absent
	}
	if !File_Mode_Is_Directory(state.Nodes[parent].Mode) {
		return 0, sim_not_a_directory
	}
	if child, present := sim_node_find_child(state, parent, leaf); present {
		if File_Mode_Is_Directory(state.Nodes[child].Mode) {
			return 0, sim_is_a_directory
		}
		// Truncate keeps the mode an existing node already carries, because creation
		// permissions describe a node being made, not one being reopened.
		state.Nodes[child].Contents_Count = 0
		return child, nil
	}
	return sim_node_acquire(state, parent, leaf, File_Mode(permissions&^SIM_UMASK))
}

func sim_node_write(node *Sim_Node, buffer []byte, offset int64) (err error) {
	if offset < 0 {
		return sim_file_offset_invalid
	}
	if offset > int64(SIM_FILE_BYTES_MAXIMUM) {
		return sim_file_offset_invalid
	}
	end := offset + int64(len(buffer))
	if end < offset {
		return sim_file_capacity_exceeded
	}
	if end > int64(SIM_FILE_BYTES_MAXIMUM) {
		return sim_file_capacity_exceeded
	}
	copy(node.Contents[int(offset):int(end)], buffer)
	if int(end) > node.Contents_Count {
		node.Contents_Count = int(end)
	}
	return nil
}

// File-size percentiles generated contents are sampled from: most file is handful of bytes, few
// reach hundreds, and top one percent is largest — heavy-tailed spread
// (prng.Percentile_Distribution), thus sweep meet many scales at once.
const SIM_SIZE_P50 = 4

// SIM_SIZE_P75 keep deterministic filesystem size distribution.
const SIM_SIZE_P75 = 16

// SIM_SIZE_P95 keep deterministic filesystem size distribution.
const SIM_SIZE_P95 = 64

// SIM_SIZE_P99 keep deterministic filesystem size distribution.
const SIM_SIZE_P99 = 256

// SIM_SIZE_P100 keep deterministic filesystem size distribution.
const SIM_SIZE_P100 = 1024

// Report how often generator add another sibling — 3-in-4 Chance, thus directory breadth is
// geometric and any width is reachable, not capped at fixed count.
func sim_grow_chance() (chance prng.Ratio) {
	return prng.Ratio{Numerator: 3, Denominator: 4}
}

// Report how often generated child is subdirectory, not file — kept below grow chance, thus
// branching stay subcritical and generation halt almost surely.
func sim_subdirectory_chance() (chance prng.Ratio) {
	return prng.Ratio{Numerator: 1, Denominator: 4}
}

// Fabricate filesystem from seed, drawing shape from prng distributions. The shared slice limit
// caps the complete tree, because every directory must remain sortable through that boundary.
// It know no consumer layout. Walker impose its own meaning.
func sim_generate(state *Sim) {
	generated_limit := 1 + (len(state.Nodes)-1)/2
	generated_count := 1
	for directory_index := 0; directory_index < generated_count; directory_index++ {
		if !state.Nodes[directory_index].Used {
			continue
		}
		if !File_Mode_Is_Directory(state.Nodes[directory_index].Mode) {
			continue
		}
		entry_index := 0
		for generated_count < generated_limit {
			if !prng.Xoshiro_Chance(&state.Generator, sim_grow_chance()) {
				break
			}
			directory := prng.Xoshiro_Chance(
				&state.Generator, sim_subdirectory_chance(),
			)
			sim_generate_child(state, directory_index, entry_index, bool(directory))
			generated_count++
			entry_index++
		}
	}
}

// Make one generated child and fill whatever its kind stores. Kind shape, permissions, and link
// choice each draw from their own stream, thus adding either later axis leaves the tree every
// banked seed already produces unchanged.
func sim_generate_child(state *Sim, parent int, entry_index int, directory bool) {
	child, acquire_err := sim_generated_node_acquire(
		state, parent, entry_index, sim_generate_mode(state, directory, entry_index),
	)
	aver.Always(acquire_err == nil,
		"Generated nodes fit reserved caller-owned node storage.")
	node := &state.Nodes[child]
	if File_Mode_Is_Symbolic_Link(node.Mode) {
		node.Contents_Count = sim_generated_name(entry_index-1, node.Contents[:])
		return
	}
	if directory {
		return
	}
	sim_generate_bytes(&state.Generator, node)
}

// Draw one generated node mode. A link points at an earlier sibling, thus the first entry of any
// directory is never one and no generated link can dangle.
func sim_generate_mode(state *Sim, directory bool, entry_index int) (mode File_Mode) {
	mode = File_Mode(prng.Xoshiro_Below(
		&state.Permission_Generator, prng.Bound(SIM_PERMISSION_DRAW_COUNT),
	))
	if directory {
		return mode | FILE_MODE_DIRECTORY
	}
	if entry_index == 0 {
		return mode
	}
	if prng.Xoshiro_Below(&state.Link_Generator, SIM_LINK_CHANCE_DENOMINATOR) != 0 {
		return mode
	}
	return mode | FILE_MODE_SYMBOLIC_LINK
}

// DECIMAL_RADIX selects base ten for generated entry names.
const DECIMAL_RADIX = 10

func sim_generated_node_acquire(
	state *Sim, parent int, value int, mode File_Mode,
) (node_index int, err error) {
	for index := 1; index < len(state.Nodes); index++ {
		if state.Nodes[index].Used {
			continue
		}
		node := &state.Nodes[index]
		sim_node_reset(node)
		node.Used = true
		node.Mode = mode
		node.Parent = parent
		node.Name_Count = sim_generated_name(value, node.Name)
		return index, nil
	}
	return 0, sim_node_capacity_exceeded
}

// Reset keeps caller-provided node backing while filesystem ownership changes.
func sim_node_reset(node *Sim_Node) {
	name := node.Name
	contents := node.Contents
	*node = Sim_Node{Name: name, Contents: contents}
}

// Write one generated entry name into caller storage. Node naming and link targets share this
// formatter, thus a generated link cannot name a sibling that naming never produces.
func sim_generated_name(value int, storage []byte) (count int) {
	aver.Always(value >= 0, "A generated entry index is nonnegative.")
	buffer := [bits.WORD_SIZE]byte{}
	position_count := len(buffer) - 1
	buffer[position_count] = byte(value%DECIMAL_RADIX) + '0'
	value /= DECIMAL_RADIX
	for value > 0 {
		position_count--
		buffer[position_count] = byte(value%DECIMAL_RADIX) + '0'
		value /= DECIMAL_RADIX
	}
	storage[0] = 'e'
	return 1 + copy(storage[1:], buffer[position_count:])
}

// Draw file contents from seed: size Sampled from heavy-tailed distribution, filled with
// seed-drawn bytes.
func sim_generate_bytes(generator *prng.Xoshiro, node *Sim_Node) {
	roll := prng.Xoshiro_Below(generator, 100)
	size := SIM_SIZE_P100
	if roll < 25 {
		size = 0
	} else if roll < 50 {
		size = SIM_SIZE_P50
	} else if roll < 75 {
		size = SIM_SIZE_P75
	} else if roll < 95 {
		size = SIM_SIZE_P95
	} else if roll < 99 {
		size = SIM_SIZE_P99
	}
	node.Contents_Count = size
	for index := 0; index < size; index++ {
		node.Contents[index] = byte(prng.Xoshiro_Next(generator))
	}
}

// Draw next operation completion delay from seed, thus completion order vary per run yet
// reproduce exact.
func sim_latency(state *Sim) (latency time.Duration) {
	return sim_latency_from(&state.Generator)
}

// Draws one simulated operation's virtual latency from the stream that owns its axis.
func sim_latency_from(generator *prng.Xoshiro) (latency time.Duration) {
	return time.Duration(prng.Xoshiro_Below(generator, SIM_LATENCY_GRAINS))
}

// Kernels can publish either terminal event when operation and timeout become ready together.
func sim_timeout_first(
	state *Sim, latency time.Duration, timeout time.Duration,
) (timeout_first bool) {
	return sim_timeout_order_first(&state.Timeout_Order_Generator, latency, timeout)
}

// A separate draw keeps new storage races from changing existing network order.
func sim_storage_timeout_first(
	state *Sim, latency time.Duration, timeout time.Duration,
) (timeout_first bool) {
	return sim_timeout_order_first(
		&state.Storage_Timeout_Order_Generator, latency, timeout,
	)
}

// One comparison rule keeps all simulated kernel timeout races consistent.
func sim_timeout_order_first(
	generator *prng.Xoshiro, latency time.Duration, timeout time.Duration,
) (timeout_first bool) {
	if latency < timeout {
		return false
	}
	if latency > timeout {
		return true
	}
	return prng.Xoshiro_Below(generator, 2) == 0
}

// Report synthetic peer address of live descriptor, simulator getpeername. Empty with no error
// for unknown descriptor.
func sim_peer_address(state *Sim, file File) (address Address) {
	descriptor := sim_descriptor_find(state, file)
	if descriptor == nil {
		return Address{}
	}
	if !descriptor.Socket {
		return Address{}
	}
	if !descriptor.Socket_State.Connected {
		return Address{}
	}
	return descriptor.Socket_State.Peer
}

// Acquire one caller-owned operation record; capacity is explicit simulator configuration.
func sim_operation_acquire(
	state *Sim, completion *Completion, kind Sim_Operation_Kind, callback Callback,
) (operation *Sim_Operation) {
	aver.Always(completion.Backend == nil,
		"A simulated completion has no retained operation before submission.")
	for index := range state.Operations {
		if state.Operations[index].Kind != SIM_OPERATION_KIND_FREE {
			continue
		}
		operation = &state.Operations[index]
		break
	}
	if operation == nil {
		if callback != nil {
			sim_operation_reject(state, completion, callback)
		}
		return nil
	}
	sim_operation_reset(operation)
	operation.Kind = kind
	operation.State = state
	operation.Callback = callback
	completion.Backend = State(operation)
	return operation
}

// Capacity exhaustion joins the same timeline as accepted work so rejection cannot reenter the
// submitter and every callback still observes an ordinary idle completion.
func sim_operation_reject(state *Sim, completion *Completion, callback Callback) {
	virtual_submit(&state.Timeline, completion, 0, callback)
	completion.Error = sim_operation_capacity_exceeded
}

// Reset keeps caller-provided address backing while one operation slot changes owner.
func sim_operation_reset(operation *Sim_Operation) {
	address_ip := operation.Address_IP
	*operation = Sim_Operation{Address_IP: address_ip}
}

func sim_operation_borrow(operation *Sim_Operation) {
	descriptor := sim_descriptor_find(operation.State, operation.File)
	aver.Always(descriptor != nil, "A simulated operation borrows an open descriptor.")
	descriptor.Borrowed++
	operation.Borrowed = true
}

func sim_operation_submit(
	operation *Sim_Operation, completion *Completion, latency time.Duration,
) {
	virtual_submit(&operation.State.Timeline, completion, latency, sim_operation_complete)
}

// One static retirement callback decodes caller-owned state and frees it before user reentry.
func sim_operation_complete(completion Completion_Handle) {
	operation := completion.Backend.(*Sim_Operation)
	aver.Always(operation != nil, "A simulated retirement has operation state.")
	data := operation.Data
	err := operation.Operation_Err
	if err == nil {
		aver.Always(
			operation.Kind >= SIM_OPERATION_KIND_CLOSE &&
				operation.Kind <= SIM_OPERATION_KIND_PROCESS,
			"A simulated operation kind is known.",
		)
		switch operation.Kind {
		case SIM_OPERATION_KIND_CLOSE:
			sim_descriptor_release(operation.State, operation.File)
		case SIM_OPERATION_KIND_ACCEPT:
			data, err = sim_operation_accept(operation)
		case SIM_OPERATION_KIND_CONNECT:
			descriptor := sim_descriptor_find(operation.State, operation.File)
			if descriptor != nil {
				if descriptor.Socket {
					descriptor.Socket_State.Connected = true
					descriptor.Socket_State.Peer = sim_address_copy(
						descriptor.Peer_IP, operation.Address,
					)
				}
			}
		case SIM_OPERATION_KIND_RECEIVE:
			data = sim_operation_receive(operation)
		case SIM_OPERATION_KIND_SEND:
			data, err = sim_operation_send(operation)
		case SIM_OPERATION_KIND_OPEN_AT:
			data, err = sim_operation_open_at(operation)
		case SIM_OPERATION_KIND_MKDIR_AT:
			if operation.Directory != DIRECTORY_CURRENT {
				err = sim_not_a_directory
			} else {
				err = sim_mkdir(
					operation.State, operation.File_Path,
					operation.Open_Options.Permissions,
				)
			}
		case SIM_OPERATION_KIND_READ:
			data, err = sim_operation_read(operation)
		case SIM_OPERATION_KIND_WRITE:
			data, err = sim_operation_write(operation)
		case SIM_OPERATION_KIND_FSYNC:
		case SIM_OPERATION_KIND_DIRECTORY:
			data, err = sim_directory_pass(
				operation.State, operation.File, operation.Entries,
			)
		case SIM_OPERATION_KIND_STATX:
			data, err = sim_statx_operation_complete(operation)
		}
	}
	sim_operation_deliver(operation, completion, data, err)
}

func sim_operation_accept(operation *Sim_Operation) (data int, err error) {
	listener := sim_descriptor_find(operation.State, operation.File)
	if listener == nil {
		return 0, sim_socket_listener_required
	}
	if !listener.Socket {
		return 0, sim_socket_listener_required
	}
	if !listener.Socket_State.Listener {
		return 0, sim_socket_listener_required
	}
	socket, open_err := sim_open_socket(
		operation.State, listener.Socket_State.Family, false,
	)
	if open_err != nil {
		return int(File(-1)), open_err
	}
	accepted := sim_descriptor_find(operation.State, socket)
	accepted.Socket_State.Connected = true
	if accepted.Socket_State.Family == FAMILY_IPV6 {
		for index := range accepted.Peer_IP {
			accepted.Peer_IP[index] = 0
		}
		accepted.Peer_IP[IPV6_ADDRESS_BYTES-1] = 1
		accepted.Socket_State.Peer = Address_IPV6(accepted.Peer_IP, 0)
	} else {
		copy(accepted.Peer_IP, []byte{127, 0, 0, 1})
		accepted.Socket_State.Peer = Address_IPV4(accepted.Peer_IP[:IPV4_ADDRESS_BYTES], 0)
	}
	return int(socket), nil
}

func sim_operation_receive(operation *Sim_Operation) (data int) {
	descriptor := sim_descriptor_find(operation.State, operation.File)
	if descriptor == nil {
		return 0
	}
	if !descriptor.Socket {
		return 0
	}
	if descriptor.Socket_State.Receive_Shutdown {
		return 0
	}
	return len(operation.Buffer)
}

func sim_operation_send(operation *Sim_Operation) (data int, err error) {
	descriptor := sim_descriptor_find(operation.State, operation.File)
	if descriptor == nil {
		return 0, Broken_Pipe
	}
	if !descriptor.Socket {
		return 0, Broken_Pipe
	}
	if descriptor.Socket_State.Send_Shutdown {
		return 0, Broken_Pipe
	}
	return len(operation.Buffer), nil
}

func sim_operation_open_at(operation *Sim_Operation) (data int, err error) {
	if operation.Directory != DIRECTORY_CURRENT {
		return int(File(-1)), sim_not_a_directory
	}
	file := File(-1)
	if operation.Open_Options.Create {
		file, err = sim_create(
			operation.State, operation.File_Path, operation.Open_Options.Permissions,
		)
	} else {
		file, err = sim_open(
			operation.State, operation.File_Path, operation.Open_Options.Flags,
		)
	}
	return int(file), err
}

func sim_operation_read(operation *Sim_Operation) (data int, err error) {
	if operation.Operation_Err != nil {
		return 0, operation.Operation_Err
	}
	node := &operation.State.Nodes[operation.Node]
	if File_Mode_Is_Directory(node.Mode) {
		return 0, sim_is_a_directory
	}
	if operation.Offset < 0 {
		return 0, sim_file_offset_invalid
	}
	if operation.Offset >= int64(node.Contents_Count) {
		return 0, nil
	}
	return copy(
		operation.Buffer, node.Contents[int(operation.Offset):node.Contents_Count],
	), nil
}

func sim_operation_write(operation *Sim_Operation) (data int, err error) {
	node := &operation.State.Nodes[operation.Node]
	if File_Mode_Is_Directory(node.Mode) {
		return 0, sim_is_a_directory
	}
	write_err := sim_node_write(node, operation.Buffer, operation.Offset)
	if write_err != nil {
		return 0, write_err
	}
	return len(operation.Buffer), nil
}

func sim_operation_deliver(
	operation *Sim_Operation, completion *Completion, data int, err error,
) {
	state := operation.State
	file := operation.File
	borrowed := operation.Borrowed
	callback := operation.Callback
	if borrowed {
		descriptor := sim_descriptor_find(state, file)
		aver.Always(descriptor != nil,
			"A simulated retirement releases a live descriptor.")
		aver.Always(descriptor.Borrowed > 0,
			"A simulated retirement releases one descriptor borrow.")
		descriptor.Borrowed--
	}
	sim_operation_reset(operation)
	completion.Backend = nil
	completion.Data = data
	completion.Error = err
	callback(completion)
}

// Stream keeps caller-owned state beside static procedure because returned capturing functions
// escape to heap before any operation starts.
type Stream struct {
	// State remains caller-owned because bound procedure state would allocate.
	State State
	// Procedure stays static so returned Stream contains no closure environment.
	Procedure Stream_Procedure
}

// Stream_Procedure owns completion policy because only concrete stream knows whether work is
// immediate, simulated, or kernel-backed.
type Stream_Procedure func(
	state State, completion *Completion, mode Stream_Mode, buffer []byte,
	offset int64, whence Seek_From, callback Stream_Callback,
)

// Stream_Callback carries explicit state through asynchronous composition because an adapter
// closure would escape until inner operation retires.
type Stream_Callback struct {
	// State restores composed Stream ownership without a closure.
	State State
	// Data preserves adapter-specific integer state until inner retirement.
	Data int
	// Callback preserves final receiver while static adapter runs first.
	Callback Callback
	// Procedure stays static so callback composition allocates nothing.
	Procedure Stream_Callback_Procedure
}

// Stream_Callback_Data is opaque integer state interpreted by one static adapter.
type Stream_Callback_Data int

// Stream_Callback_Procedure is static callback half of Stream_Callback.
type Stream_Callback_Procedure func(
	state State, data Stream_Callback_Data, callback Callback,
	completion Completion_Handle,
)

// Stream_Callback_Call lets custom Stream procedures retire explicit callback state.
func Stream_Callback_Call(callback Stream_Callback, completion *Completion) {
	aver.Always(callback.Procedure != nil, "A Stream callback has a procedure.")
	callback.Procedure(
		callback.State, Stream_Callback_Data(callback.Data), callback.Callback, completion,
	)
}

// Stream_Mode keeps the Odin stream operation set while result delivery uses the repository
// completion model.
type Stream_Mode int

// STREAM_MODE_CLOSE ends the stream.
const STREAM_MODE_CLOSE Stream_Mode = 0

// STREAM_MODE_FLUSH pushes buffered bytes onward.
const STREAM_MODE_FLUSH Stream_Mode = 1

// STREAM_MODE_READ reads at the cursor.
const STREAM_MODE_READ Stream_Mode = 2

// STREAM_MODE_READ_AT reads at an explicit offset.
const STREAM_MODE_READ_AT Stream_Mode = 3

// STREAM_MODE_WRITE writes at the cursor.
const STREAM_MODE_WRITE Stream_Mode = 4

// STREAM_MODE_WRITE_AT writes at an explicit offset.
const STREAM_MODE_WRITE_AT Stream_Mode = 5

// STREAM_MODE_SEEK moves the cursor.
const STREAM_MODE_SEEK Stream_Mode = 6

// STREAM_MODE_SIZE reports the total size.
const STREAM_MODE_SIZE Stream_Mode = 7

// STREAM_MODE_DESTROY releases stream state.
const STREAM_MODE_DESTROY Stream_Mode = 8

// STREAM_MODE_QUERY names capability inspection through the procedure that owns the modes.
const STREAM_MODE_QUERY Stream_Mode = 9

// Stream_Mode_Set keeps capability checks separate from attempts that would fail asynchronously.
type Stream_Mode_Set uint64

// Mode_Set_Add permits composition code to extend a capability set without integer casts.
func Mode_Set_Add(modes Stream_Mode_Set, mode Stream_Mode) (extended Stream_Mode_Set) {
	return modes | 1<<uint(mode)
}

// Mode_Set_Has permits callers to reject unsupported work before they borrow a completion.
func Mode_Set_Has(modes Stream_Mode_Set, mode Stream_Mode) (present bool) {
	return modes&(1<<uint(mode)) != 0
}

// Seek_From keeps cursor arithmetic explicit because a plain offset cannot name its origin.
type Seek_From int

// SEEK_FROM_START measures from the first byte.
const SEEK_FROM_START Seek_From = 0

// SEEK_FROM_CURRENT measures from the cursor.
const SEEK_FROM_CURRENT Seek_From = 1

// SEEK_FROM_END measures from one position after the last byte.
const SEEK_FROM_END Seek_From = 2

// Stream_EOF differs from a standard-library error because this package does not implement the
// standard-library reader contract.
var Stream_EOF = errors.New("io: stream end of file")

// Stream_Unexpected_EOF preserves the distinction between no bytes and an incomplete value.
var Stream_Unexpected_EOF = errors.New("io: stream unexpected end of file")

// Stream_Short_Write prevents a bounded sink from hiding bytes that it could not store.
var Stream_Short_Write = errors.New("io: stream short write")

// Stream_Invalid_Write rejects a Stream that reports more bytes than its borrowed buffer.
var Stream_Invalid_Write = errors.New("io: stream invalid write")

// Stream_Short_Buffer rejects a read result larger than its borrowed destination.
var Stream_Short_Buffer = errors.New("io: stream short buffer")

// Stream_No_Progress lets higher-level bounded loops distinguish a stall from an end.
var Stream_No_Progress = errors.New("io: stream made no progress")

// Stream_Invalid_Whence rejects cursor arithmetic whose origin has no defined meaning.
var Stream_Invalid_Whence = errors.New("io: stream invalid whence")

// Stream_Invalid_Offset rejects a cursor position outside the stream budget.
var Stream_Invalid_Offset = errors.New("io: stream invalid offset")

// Stream_Invalid_Unread prevents restoration of a byte that the stream did not produce.
var Stream_Invalid_Unread = errors.New("io: stream invalid unread")

// Stream_Negative_Read rejects a Stream result that cannot name a byte count.
var Stream_Negative_Read = errors.New("io: stream negative read")

// Stream_Negative_Write rejects a Stream result that cannot name a byte count.
var Stream_Negative_Write = errors.New("io: stream negative write")

// Stream_Negative_Count rejects a requested quantity that cannot name bytes.
var Stream_Negative_Count = errors.New("io: stream negative count")

// Stream_Buffer_Full identifies a bounded destination that has no remaining capacity.
var Stream_Buffer_Full = errors.New("io: stream buffer full")

// Stream_Unknown preserves a terminal failure when no more specific stream error applies.
var Stream_Unknown = errors.New("io: stream unknown error")

// Stream_Empty gives absent, closed, and unsupported operations one terminal result.
var Stream_Empty = errors.New("io: stream empty")

// Read validates the procedure result before the callback can trust its byte count.
func Read(
	stream Stream, completion *Completion, buffer []byte, callback Callback,
) {
	stream_submit(
		stream, completion, STREAM_MODE_READ, buffer, 0, SEEK_FROM_START,
		Stream_Callback{
			Data: len(buffer), Callback: callback, Procedure: stream_read_complete,
		},
	)
}

// Read_At keeps the cursor unchanged while it applies the same result validation as Read.
func Read_At(
	stream Stream, completion *Completion, buffer []byte, offset int64,
	callback Callback,
) {
	stream_submit(
		stream, completion, STREAM_MODE_READ_AT, buffer, offset, SEEK_FROM_START,
		Stream_Callback{
			Data: len(buffer), Callback: callback, Procedure: stream_read_complete,
		},
	)
}

// Write converts a silent short write into an error before the callback observes it.
func Write(
	stream Stream, completion *Completion, buffer []byte, callback Callback,
) {
	stream_submit(
		stream, completion, STREAM_MODE_WRITE, buffer, 0, SEEK_FROM_START,
		Stream_Callback{
			Data: len(buffer), Callback: callback, Procedure: stream_write_complete,
		},
	)
}

// Write_At leaves the cursor unchanged while it rejects an invalid procedure count.
func Write_At(
	stream Stream, completion *Completion, buffer []byte, offset int64,
	callback Callback,
) {
	stream_submit(
		stream, completion, STREAM_MODE_WRITE_AT, buffer, offset, SEEK_FROM_START,
		Stream_Callback{
			Data: len(buffer), Callback: callback, Procedure: stream_write_complete,
		},
	)
}

// Seek reports the new cursor through Completion.Data, so it follows the same lifecycle as a
// byte transfer.
func Seek(
	stream Stream, completion *Completion, offset int64, whence Seek_From,
	callback Callback,
) {
	stream_submit(
		stream, completion, STREAM_MODE_SEEK, nil, offset, whence,
		stream_final_callback(callback),
	)
}

// Size reports the bounded storage size through Completion.Data.
func Size(stream Stream, completion *Completion, callback Callback) {
	stream_submit(
		stream, completion, STREAM_MODE_SIZE, nil, 0, SEEK_FROM_START,
		stream_final_callback(callback),
	)
}

// Flush remains a submitted operation because composition can place buffering behind Stream.
func Flush(stream Stream, completion *Completion, callback Callback) {
	stream_submit(
		stream, completion, STREAM_MODE_FLUSH, nil, 0, SEEK_FROM_START,
		stream_final_callback(callback),
	)
}

// Close retires on the timeline, so teardown can join it before release of owner state.
func Close(stream Stream, completion *Completion, callback Callback) {
	stream_submit(
		stream, completion, STREAM_MODE_CLOSE, nil, 0, SEEK_FROM_START,
		stream_final_callback(callback),
	)
}

// Destroy remains distinct from Close because some stream implementations can own storage.
func Destroy(stream Stream, completion *Completion, callback Callback) {
	stream_submit(
		stream, completion, STREAM_MODE_DESTROY, nil, 0, SEEK_FROM_START,
		stream_final_callback(callback),
	)
}

// Query reports capabilities through Completion.Data because the procedure owns that data.
func Query(stream Stream, completion *Completion, callback Callback) {
	stream_submit(
		stream, completion, STREAM_MODE_QUERY, nil, 0, SEEK_FROM_START,
		stream_final_callback(callback),
	)
}

func stream_final_callback(callback Callback) (stream_callback Stream_Callback) {
	return Stream_Callback{Callback: callback, Procedure: stream_callback_final}
}

func stream_callback_final(
	_ State, _ Stream_Callback_Data, callback Callback,
	completion Completion_Handle,
) {
	callback(completion)
}

func stream_read_complete(
	_ State, buffer_count Stream_Callback_Data, callback Callback,
	completion Completion_Handle,
) {
	completion.Data, completion.Error = stream_read_checked(
		completion.Data, completion.Error, int(buffer_count),
	)
	callback(completion)
}

func stream_write_complete(
	_ State, buffer_count Stream_Callback_Data, callback Callback,
	completion Completion_Handle,
) {
	completion.Data, completion.Error = stream_write_checked(
		completion.Data, completion.Error, int(buffer_count),
	)
	callback(completion)
}

// All public modes enter here, so nil checks and unsupported-mode behavior cannot diverge.
func stream_submit(
	stream Stream, completion *Completion, mode Stream_Mode, buffer []byte,
	offset int64, whence Seek_From, callback Stream_Callback,
) {
	aver.Always(completion != nil, "A Stream operation has a completion.")
	aver.Always(callback.Procedure != nil, "A Stream operation has a callback.")
	if stream.Procedure == nil {
		stream_complete(completion, 0, Stream_Empty, callback)
		return
	}
	stream.Procedure(stream.State, completion, mode, buffer, offset, whence, callback)
}

// Completion fields hold every scalar result, so Stream needs no second callback type.
func stream_complete(
	completion *Completion, data int, err error, callback Stream_Callback,
) {
	completion.Data = data
	completion.Error = err
	Stream_Callback_Call(callback, completion)
}

// A read count outside its borrowed buffer is a procedure defect, not caller data.
func stream_read_checked(
	count int, err error, buffer_count int,
) (checked int, checked_err error) {
	if count < 0 {
		return 0, Stream_Negative_Read
	}
	if count > buffer_count {
		return 0, Stream_Short_Buffer
	}
	return count, err
}

// A write must report every unstored byte because silent truncation loses caller data.
func stream_write_checked(
	count int, err error, buffer_count int,
) (checked int, checked_err error) {
	if count < 0 {
		return 0, Stream_Negative_Write
	}
	if count > buffer_count {
		return 0, Stream_Invalid_Write
	}
	if err != nil {
		return count, err
	}
	if count < buffer_count {
		return count, Stream_Short_Write
	}
	return count, nil
}

// STREAM_MEMORY_MODES records all ten modes because bounded memory can implement each one.
const STREAM_MEMORY_MODES Stream_Mode_Set = 1<<STREAM_MODE_CLOSE |
	1<<STREAM_MODE_FLUSH | 1<<STREAM_MODE_READ | 1<<STREAM_MODE_READ_AT |
	1<<STREAM_MODE_WRITE | 1<<STREAM_MODE_WRITE_AT | 1<<STREAM_MODE_SEEK |
	1<<STREAM_MODE_SIZE | 1<<STREAM_MODE_DESTROY | 1<<STREAM_MODE_QUERY

// Stream_Memory keeps caller-owned storage bounded at the supplied slice.
type Stream_Memory struct {
	// Memory is the complete storage budget.
	Memory []byte
	// Cursor is the next sequential operation position.
	Cursor int64
	// Closed prevents later data modes from reaching released state.
	Closed bool
}

// Memory_To_Stream keeps storage ownership in the caller state.
func Memory_To_Stream(state *Stream_Memory) (stream Stream) {
	aver.Always(state != nil, "A memory Stream has state.")
	return Stream{State: State(state), Procedure: stream_memory_procedure}
}

// Memory completes inline because its concrete operation cannot wait on an external endpoint.
func stream_memory_procedure(
	state_pointer State, completion *Completion, mode Stream_Mode, buffer []byte,
	offset int64, whence Seek_From, callback Stream_Callback,
) {
	state := state_pointer.(*Stream_Memory)
	count, operation_err := stream_memory(state, mode, buffer, offset, whence)
	stream_complete(completion, count, operation_err, callback)
}

// Lifecycle checks precede data modes, so close and destroy seal all access in one place.
func stream_memory(
	state *Stream_Memory, mode Stream_Mode, buffer []byte, offset int64,
	whence Seek_From,
) (count int, err error) {
	if mode == STREAM_MODE_QUERY {
		return int(STREAM_MEMORY_MODES), nil
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
	if mode == STREAM_MODE_SIZE {
		return len(state.Memory), nil
	}
	if mode == STREAM_MODE_SEEK {
		return stream_memory_seek(state, offset, whence)
	}
	if mode == STREAM_MODE_READ {
		moved, read_err := stream_memory_read(state, buffer, state.Cursor)
		state.Cursor = state.Cursor + int64(moved)
		return moved, read_err
	}
	if mode == STREAM_MODE_READ_AT {
		return stream_memory_read(state, buffer, offset)
	}
	if mode == STREAM_MODE_WRITE {
		moved, write_err := stream_memory_write(state, buffer, state.Cursor)
		state.Cursor = state.Cursor + int64(moved)
		return moved, write_err
	}
	if mode == STREAM_MODE_WRITE_AT {
		return stream_memory_write(state, buffer, offset)
	}
	return 0, Stream_Empty
}

// A read outside the fixed slice would turn an arithmetic error into fabricated bytes.
func stream_memory_read(
	state *Stream_Memory, buffer []byte, offset int64,
) (count int, err error) {
	if offset < 0 {
		return 0, Stream_Invalid_Offset
	}
	if offset > int64(len(state.Memory)) {
		return 0, Stream_Invalid_Offset
	}
	if offset == int64(len(state.Memory)) {
		return 0, Stream_EOF
	}
	return copy(buffer, state.Memory[offset:]), nil
}

// A bounded memory Stream reports truncation instead of increasing caller-owned storage.
func stream_memory_write(
	state *Stream_Memory, buffer []byte, offset int64,
) (count int, err error) {
	if offset < 0 {
		return 0, Stream_Invalid_Offset
	}
	if offset > int64(len(state.Memory)) {
		return 0, Stream_Invalid_Offset
	}
	count = copy(state.Memory[offset:], buffer)
	if count < len(buffer) {
		return count, Stream_Short_Write
	}
	return count, nil
}

// Cursor bounds reject silent clamping because a changed position hides caller arithmetic.
func stream_memory_seek(
	state *Stream_Memory, offset int64, whence Seek_From,
) (position int, err error) {
	target := offset
	if whence == SEEK_FROM_CURRENT {
		target = state.Cursor + offset
	}
	if whence == SEEK_FROM_END {
		target = int64(len(state.Memory)) + offset
	}
	if whence < SEEK_FROM_START {
		return 0, Stream_Invalid_Whence
	}
	if whence > SEEK_FROM_END {
		return 0, Stream_Invalid_Whence
	}
	if target < 0 {
		return 0, Stream_Invalid_Offset
	}
	if target > int64(len(state.Memory)) {
		return 0, Stream_Invalid_Offset
	}
	state.Cursor = target
	return int(target), nil
}

// STREAM_DISCARD_MODES omits reads and cursor operations because discarded bytes leave no state
// that can answer them.
const STREAM_DISCARD_MODES Stream_Mode_Set = 1<<STREAM_MODE_CLOSE |
	1<<STREAM_MODE_FLUSH | 1<<STREAM_MODE_WRITE | 1<<STREAM_MODE_WRITE_AT |
	1<<STREAM_MODE_DESTROY | 1<<STREAM_MODE_QUERY

// Stream_Discard keeps lifecycle state even though it keeps no bytes.
type Stream_Discard struct {
	// Closed prevents a reused sink from accepting bytes after teardown.
	Closed bool
}

// Discard_To_Stream keeps lifecycle ownership in the caller state.
func Discard_To_Stream(state *Stream_Discard) (stream Stream) {
	aver.Always(state != nil, "A discard Stream has state.")
	return Stream{State: State(state), Procedure: stream_discard_procedure}
}

// Discard completes inline because it has no external endpoint or scheduled work.
func stream_discard_procedure(
	state_pointer State, completion *Completion, mode Stream_Mode, buffer []byte,
	_ int64, _ Seek_From, callback Stream_Callback,
) {
	state := state_pointer.(*Stream_Discard)
	count, operation_err := stream_discard(state, mode, buffer)
	stream_complete(completion, count, operation_err, callback)
}

// Lifecycle state distinguishes a valid sink from one whose owner released it.
func stream_discard(
	state *Stream_Discard, mode Stream_Mode, buffer []byte,
) (count int, err error) {
	if mode == STREAM_MODE_QUERY {
		return int(STREAM_DISCARD_MODES), nil
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
		return len(buffer), nil
	}
	if mode == STREAM_MODE_WRITE_AT {
		return len(buffer), nil
	}
	return 0, Stream_Empty
}

// STREAM_LIMIT_MODES omits positioned and cursor modes because a pass budget cannot describe
// bytes that bypass its sequence.
const STREAM_LIMIT_MODES Stream_Mode_Set = 1<<STREAM_MODE_CLOSE |
	1<<STREAM_MODE_FLUSH | 1<<STREAM_MODE_READ | 1<<STREAM_MODE_WRITE |
	1<<STREAM_MODE_DESTROY | 1<<STREAM_MODE_QUERY

// Stream_Limit makes the remaining byte budget part of the transport instead of caller policy.
type Stream_Limit struct {
	// Inner receives only bytes inside the remaining budget.
	Inner Stream
	// Budget decreases by the count that Inner reports.
	Budget int64
	// Callback preserves outer completion state while Inner is armed.
	Callback Stream_Callback
	// Buffer_Count remembers original request when budget truncates it.
	Buffer_Count int
	// Active rejects a second operation that would overwrite retained callback state.
	Active bool
}

// Limit_To_Stream keeps the inner Stream completion policy unchanged.
func Limit_To_Stream(state *Stream_Limit) (stream Stream) {
	aver.Always(state != nil, "A limit Stream has state.")
	return Stream{State: State(state), Procedure: stream_limit_procedure}
}

// Limit forwards one submitted operation and adjusts its budget only after Inner retires.
func stream_limit_procedure(
	state_pointer State, completion *Completion, mode Stream_Mode, buffer []byte,
	_ int64, _ Seek_From, callback Stream_Callback,
) {
	state := state_pointer.(*Stream_Limit)
	if mode == STREAM_MODE_CLOSE {
		stream_submit(
			state.Inner, completion, mode, nil, 0, SEEK_FROM_START, callback,
		)
		return
	}
	if mode == STREAM_MODE_DESTROY {
		stream_submit(
			state.Inner, completion, mode, nil, 0, SEEK_FROM_START, callback,
		)
		return
	}
	if mode == STREAM_MODE_FLUSH {
		stream_submit(
			state.Inner, completion, mode, nil, 0, SEEK_FROM_START, callback,
		)
		return
	}
	if mode == STREAM_MODE_QUERY {
		stream_limit_begin(state, callback, 0)
		stream_submit(state.Inner, completion, mode, nil, 0, SEEK_FROM_START,
			Stream_Callback{
				State:     State(state),
				Procedure: stream_limit_query_complete,
			})
		return
	}
	if mode == STREAM_MODE_READ {
		stream_limit_read(state, completion, buffer, callback)
		return
	}
	if mode == STREAM_MODE_WRITE {
		stream_limit_write(state, completion, buffer, callback)
		return
	}
	stream_complete(completion, 0, Stream_Empty, callback)
}

// A spent read budget cannot reach Inner, thus Limit owns this immediate result.
func stream_limit_read(
	state *Stream_Limit, completion *Completion, buffer []byte,
	callback Stream_Callback,
) {
	if state.Budget <= 0 {
		stream_complete(completion, 0, Stream_EOF, callback)
		return
	}
	allowed_count := len(buffer)
	if state.Budget < int64(allowed_count) {
		allowed_count = int(state.Budget)
	}
	stream_limit_begin(state, callback, len(buffer))
	stream_submit(state.Inner, completion, STREAM_MODE_READ, buffer[:allowed_count], 0,
		SEEK_FROM_START, Stream_Callback{
			State: State(state), Data: allowed_count,
			Procedure: stream_limit_read_complete,
		})
}

// A limit reports the part it withheld even when Inner stored every byte it received.
func stream_limit_write(
	state *Stream_Limit, completion *Completion, buffer []byte,
	callback Stream_Callback,
) {
	if state.Budget <= 0 {
		stream_complete(completion, 0, Stream_Short_Write, callback)
		return
	}
	allowed_count := len(buffer)
	if state.Budget < int64(allowed_count) {
		allowed_count = int(state.Budget)
	}
	stream_limit_begin(state, callback, len(buffer))
	stream_submit(state.Inner, completion, STREAM_MODE_WRITE, buffer[:allowed_count], 0,
		SEEK_FROM_START, Stream_Callback{
			State: State(state), Data: allowed_count,
			Procedure: stream_limit_write_complete,
		})
}

func stream_limit_begin(
	state *Stream_Limit, callback Stream_Callback, buffer_count int,
) {
	aver.Always(!state.Active, "A limit Stream has at most one operation in flight.")
	state.Active = true
	state.Callback = callback
	state.Buffer_Count = buffer_count
}

func stream_limit_finish(
	state *Stream_Limit, completion *Completion,
) {
	callback := state.Callback
	state.Callback = Stream_Callback{}
	state.Buffer_Count = 0
	state.Active = false
	Stream_Callback_Call(callback, completion)
}

func stream_limit_query_complete(
	state_pointer State, _ Stream_Callback_Data, _ Callback,
	completion Completion_Handle,
) {
	state := state_pointer.(*Stream_Limit)
	completion.Data = int(Stream_Mode_Set(completion.Data) & STREAM_LIMIT_MODES)
	stream_limit_finish(state, completion)
}

func stream_limit_read_complete(
	state_pointer State, allowed_count Stream_Callback_Data, _ Callback,
	completion Completion_Handle,
) {
	state := state_pointer.(*Stream_Limit)
	completion.Data, completion.Error = stream_read_checked(
		completion.Data, completion.Error, int(allowed_count),
	)
	state.Budget -= int64(completion.Data)
	stream_limit_finish(state, completion)
}

func stream_limit_write_complete(
	state_pointer State, allowed_count Stream_Callback_Data, _ Callback,
	completion Completion_Handle,
) {
	state := state_pointer.(*Stream_Limit)
	completion.Data, completion.Error = stream_write_checked(
		completion.Data, completion.Error, int(allowed_count),
	)
	state.Budget -= int64(completion.Data)
	if completion.Error == nil {
		if int(allowed_count) < state.Buffer_Count {
			completion.Error = Stream_Short_Write
		}
	}
	stream_limit_finish(state, completion)
}

// Stream_Count changes only its tally, so Inner remains the authority for every mode result.
type Stream_Count struct {
	// Inner performs the operation that Count observes.
	Inner Stream
	// Tally records every transferred byte without constraining the transfer.
	Tally int64
	// Callback preserves outer completion state while Inner is armed.
	Callback Stream_Callback
	// Mode selects whether retired bytes contribute to Tally.
	Mode Stream_Mode
	// Active rejects a second operation that would overwrite retained callback state.
	Active bool
}

// Count_To_Stream keeps the inner Stream completion policy unchanged.
func Count_To_Stream(state *Stream_Count) (stream Stream) {
	aver.Always(state != nil, "A count Stream has state.")
	return Stream{State: State(state), Procedure: stream_count_procedure}
}

// Count updates its tally inside Inner's callback, so callers cannot observe a result before
// its accounting.
func stream_count_procedure(
	state_pointer State, completion *Completion, mode Stream_Mode, buffer []byte,
	offset int64, whence Seek_From, callback Stream_Callback,
) {
	state := state_pointer.(*Stream_Count)
	if mode == STREAM_MODE_CLOSE {
		stream_submit(state.Inner, completion, mode, nil, 0, whence, callback)
		return
	}
	if mode == STREAM_MODE_DESTROY {
		stream_submit(state.Inner, completion, mode, nil, 0, whence, callback)
		return
	}
	if mode == STREAM_MODE_FLUSH {
		stream_submit(state.Inner, completion, mode, nil, 0, whence, callback)
		return
	}
	if mode == STREAM_MODE_SEEK {
		stream_submit(state.Inner, completion, mode, nil, offset, whence, callback)
		return
	}
	if mode == STREAM_MODE_SIZE {
		stream_submit(state.Inner, completion, mode, nil, 0, whence, callback)
		return
	}
	if mode == STREAM_MODE_QUERY {
		stream_submit(state.Inner, completion, mode, nil, 0, whence, callback)
		return
	}
	if mode == STREAM_MODE_READ {
		stream_count_read(state, completion, buffer, callback)
		return
	}
	if mode == STREAM_MODE_READ_AT {
		stream_count_read_at(state, completion, buffer, offset, callback)
		return
	}
	if mode == STREAM_MODE_WRITE {
		stream_count_write(state, completion, buffer, callback)
		return
	}
	if mode == STREAM_MODE_WRITE_AT {
		stream_count_write_at(state, completion, buffer, offset, callback)
		return
	}
	stream_complete(completion, 0, Stream_Empty, callback)
}

// Count records a sequential read before it publishes Inner's completion.
func stream_count_read(
	state *Stream_Count, completion *Completion, buffer []byte,
	callback Stream_Callback,
) {
	stream_count_transfer(state, completion, STREAM_MODE_READ, buffer, 0, callback)
}

// Count records a positioned read before it publishes Inner's completion.
func stream_count_read_at(
	state *Stream_Count, completion *Completion, buffer []byte, offset int64,
	callback Stream_Callback,
) {
	stream_count_transfer(state, completion, STREAM_MODE_READ_AT, buffer, offset, callback)
}

// Count records a sequential write before it publishes Inner's completion.
func stream_count_write(
	state *Stream_Count, completion *Completion, buffer []byte,
	callback Stream_Callback,
) {
	stream_count_transfer(state, completion, STREAM_MODE_WRITE, buffer, 0, callback)
}

// Count records a positioned write before it publishes Inner's completion.
func stream_count_write_at(
	state *Stream_Count, completion *Completion, buffer []byte, offset int64,
	callback Stream_Callback,
) {
	stream_count_transfer(state, completion, STREAM_MODE_WRITE_AT, buffer, offset, callback)
}

func stream_count_transfer(
	state *Stream_Count, completion *Completion, mode Stream_Mode, buffer []byte,
	offset int64, callback Stream_Callback,
) {
	aver.Always(!state.Active, "A count Stream has at most one operation in flight.")
	state.Active = true
	state.Callback = callback
	state.Mode = mode
	stream_submit(state.Inner, completion, mode, buffer, offset, SEEK_FROM_START,
		Stream_Callback{
			State: State(state), Data: len(buffer),
			Procedure: stream_count_transfer_complete,
		})
}

func stream_count_transfer_complete(
	state_pointer State, buffer_count Stream_Callback_Data, _ Callback,
	completion Completion_Handle,
) {
	state := state_pointer.(*Stream_Count)
	if state.Mode == STREAM_MODE_READ {
		completion.Data, completion.Error = stream_read_checked(
			completion.Data, completion.Error, int(buffer_count),
		)
	} else if state.Mode == STREAM_MODE_READ_AT {
		completion.Data, completion.Error = stream_read_checked(
			completion.Data, completion.Error, int(buffer_count),
		)
	} else {
		completion.Data, completion.Error = stream_write_checked(
			completion.Data, completion.Error, int(buffer_count),
		)
	}
	state.Tally += int64(completion.Data)
	callback := state.Callback
	state.Callback = Stream_Callback{}
	state.Mode = 0
	state.Active = false
	Stream_Callback_Call(callback, completion)
}

// STREAM_TEE_MODES excludes reads and cursor operations because two inner streams can return
// different answers.
const STREAM_TEE_MODES Stream_Mode_Set = 1<<STREAM_MODE_CLOSE |
	1<<STREAM_MODE_FLUSH | 1<<STREAM_MODE_WRITE | 1<<STREAM_MODE_DESTROY |
	1<<STREAM_MODE_QUERY

// Stream_Tee sends one borrowed buffer to two destinations in a defined order.
type Stream_Tee struct {
	// First receives the operation before Second.
	First Stream
	// Second receives the operation after First retires.
	Second Stream
	// Callback preserves outer completion state across both destinations.
	Callback Stream_Callback
	// Mode preserves operation kind until second destination retires.
	Mode Stream_Mode
	// Buffer stays borrowed until both destinations retire.
	Buffer []byte
	// First_Count preserves first destination result for final minimum.
	First_Count int
	// First_Err preserves first destination failure across second submission.
	First_Err error
	// Active rejects a second operation that would overwrite retained callback state.
	Active bool
}

// Tee_To_Stream sequences the two Stream policies instead of selecting a third policy.
func Tee_To_Stream(state *Stream_Tee) (stream Stream) {
	aver.Always(state != nil, "A tee Stream has state.")
	return Stream{State: State(state), Procedure: stream_tee_procedure}
}

// Tee submits Second only after First retires, so one caller-owned completion remains valid.
func stream_tee_procedure(
	state_pointer State, completion *Completion, mode Stream_Mode, buffer []byte,
	_ int64, _ Seek_From, callback Stream_Callback,
) {
	state := state_pointer.(*Stream_Tee)
	if mode == STREAM_MODE_CLOSE {
		stream_tee_pair(state, completion, STREAM_MODE_CLOSE, callback)
		return
	}
	if mode == STREAM_MODE_DESTROY {
		stream_tee_pair(state, completion, STREAM_MODE_DESTROY, callback)
		return
	}
	if mode == STREAM_MODE_FLUSH {
		stream_tee_pair(state, completion, STREAM_MODE_FLUSH, callback)
		return
	}
	if mode == STREAM_MODE_WRITE {
		stream_tee_write(state, completion, buffer, callback)
		return
	}
	if mode == STREAM_MODE_QUERY {
		stream_complete(completion, int(STREAM_TEE_MODES), nil, callback)
		return
	}
	stream_complete(completion, 0, Stream_Empty, callback)
}

// A lifecycle operation reaches both destinations even when First reports an error.
func stream_tee_pair(
	state *Stream_Tee, completion *Completion, mode Stream_Mode,
	callback Stream_Callback,
) {
	stream_tee_begin(state, completion, mode, nil, callback)
}

// Tee reports the smaller count because that count exposes the destination that lost more
// bytes.
func stream_tee_write(
	state *Stream_Tee, completion *Completion, buffer []byte,
	callback Stream_Callback,
) {
	stream_tee_begin(state, completion, STREAM_MODE_WRITE, buffer, callback)
}

func stream_tee_begin(
	state *Stream_Tee, completion *Completion, mode Stream_Mode, buffer []byte,
	callback Stream_Callback,
) {
	aver.Always(!state.Active, "A tee Stream has at most one operation in flight.")
	state.Active = true
	state.Callback = callback
	state.Mode = mode
	state.Buffer = buffer
	stream_submit(state.First, completion, mode, buffer, 0, SEEK_FROM_START,
		Stream_Callback{
			State: State(state), Procedure: stream_tee_first_complete,
		})
}

func stream_tee_first_complete(
	state_pointer State, _ Stream_Callback_Data, _ Callback,
	completion Completion_Handle,
) {
	state := state_pointer.(*Stream_Tee)
	if state.Mode == STREAM_MODE_WRITE {
		completion.Data, completion.Error = stream_write_checked(
			completion.Data, completion.Error, len(state.Buffer),
		)
	}
	state.First_Count = completion.Data
	state.First_Err = completion.Error
	stream_submit(state.Second, completion, state.Mode, state.Buffer, 0, SEEK_FROM_START,
		Stream_Callback{
			State: State(state), Procedure: stream_tee_second_complete,
		})
}

func stream_tee_second_complete(
	state_pointer State, _ Stream_Callback_Data, _ Callback,
	completion Completion_Handle,
) {
	state := state_pointer.(*Stream_Tee)
	if state.Mode == STREAM_MODE_WRITE {
		completion.Data, completion.Error = stream_write_checked(
			completion.Data, completion.Error, len(state.Buffer),
		)
		if state.First_Count < completion.Data {
			completion.Data = state.First_Count
		}
	}
	if state.First_Err != nil {
		completion.Error = state.First_Err
	}
	callback := state.Callback
	state.Callback = Stream_Callback{}
	state.Mode = 0
	state.Buffer = nil
	state.First_Count = 0
	state.First_Err = nil
	state.Active = false
	Stream_Callback_Call(callback, completion)
}

// Completion_Handle is one live caller-owned completion delivered by a backend.
type Completion_Handle *Completion

// Callback receives the caller-owned completion after the backend retires its operation.
type Callback func(completion Completion_Handle)

// Retired_Twice report backend retire one completion more than one time. Derived function
// deliver it, never hide it. Caller own completion. Caller must learn lifecycle broke.
var Retired_Twice = errors.New("io: the completion retired more than once")

// Deadline_Exceeded come back when finite operation retire without its external event.
var Deadline_Exceeded = errors.New("io: deadline exceeded")

// Virtual_Event_Capacity_Exceeded reports no free entry in caller-owned event storage.
var Virtual_Event_Capacity_Exceeded = errors.New("io: virtual event capacity exceeded")

// Completion: caller-owned storage for one in-flight operation. Caller allocate it, thus loop
// never allocate. Caller keep it alive until callback fire.
type Completion struct {
	// Data is an opaque int whose submitting operation defines. A transfer stores its byte
	// count, while an open or accept stores its descriptor.
	Data int
	// Error is nil on success and otherwise stores the operation failure.
	Error error
	// Callback stays specialized so retirement needs no captured adapter closure.
	Callback Callback
	// Ready_At: uptime this operation complete at. Sit on monotonic timeline. Realtime jump
	// must not retire operation early, or hold it late.
	Ready_At time.Monotonic_Moment
	// Armed: completion is in flight. False mean never submitted, or delivered and free
	// again. Backend own it: it flip true on submit, false before delivery. Application never
	// read it, never write it. Show own state, not state of completion.
	Armed bool
	// Self: address of completion. First submit stamp it. Nothing clear it. Timeline track
	// in-flight operation by pointer, thus by-value copy carry this original address. Submit
	// of copy trip backend assertion. Without it, view of timeline and view of caller split
	// in silence. Only backend touch it.
	Self *Completion
	// Kernel_Identifier: generation token in kqueue udata or io_uring user_data. Backend own
	// it. Event_Trigger read it only after Event_Listen arm completion.
	Kernel_Identifier uint64
	// Event lets simulated retirement clear listener state without a captured callback.
	Event Event
	// Backend lets an outer backend correlate specialized result state without a captured
	// adapter. Backend clears it before delivering callback.
	Backend State
}

// Event: cross-thread wakeup handle of backend. kqueue EVFILT_USER ident, or eventfd
// descriptor. Primitive of loop. Carry no bytes. Name no endpoint. One purpose: make armed
// completion ready from other thread.
type Event uintptr

// Timeline: third half of IO, beside Network and Storage. Timer and cross-thread wakeup live
// here, not on a transfer surface: neither one move bytes with an endpoint. Each one decide
// WHEN a completion run.
//
// Backend fill this vtable and return Driver beside it: deterministic simulator, kqueue, or
// io_uring. Code that hold Timeline arm work, never advance it. A holder that only need a
// timer take this half alone and cannot reach a socket or a file.
type Timeline struct {
	// State stays caller-owned because every backend operation shares one loop.
	State State
	// Submit arm completion to retire one delay from now, in Ready_At order. Sim IO schedule
	// every operation through it, so one queue hold the whole simulated order. OS backend
	// keep IO order in its kernel queue and fill this slot as a timer.
	Submit func(
		state State, completion *Completion, delay time.Duration,
		callback Callback,
	)
	// Open_Event make platform Event primitive.
	Open_Event func(state State) (event Event, err error)
	// Event_Listen arm completion for one Event notification.
	Event_Listen func(
		state State, event Event, completion *Completion, callback Callback,
	)
	// Event_Trigger make armed Event completion ready. Only operation safe to call from other
	// thread.
	Event_Trigger func(state State, event Event, completion *Completion)
	// Close_Event release Event after listener drain.
	Close_Event func(state State, event Event)
}

// Timeline_Invariants state every slot full. Timeline is vtable. Zero Timeline read as
// Timeline, then panic on first use. Backend that fill four slots and forget fifth fail one
// call later.
func Timeline_Invariants(loop Timeline, namespace aver.Namespace) {
	aver.Always(loop.Submit != nil, "A Timeline arms a completion.")
	aver.Always(loop.Open_Event != nil, "A Timeline opens a cross-thread event.")
	aver.Always(loop.Event_Listen != nil, "A Timeline listens for that event.")
	aver.Always(loop.Event_Trigger != nil, "A Timeline triggers that event.")
	aver.Always(loop.Close_Event != nil, "A Timeline closes that event.")
}

// Timeline_Submit passes loop state explicitly because a bound submitter would allocate.
func Timeline_Submit(
	loop Timeline, completion *Completion, delay time.Duration, callback Callback,
) {
	loop.Submit(loop.State, completion, delay, callback)
}

// Timeline_Timeout is Submit with one guard. Same queue, same order, same backend body, thus
// no vtable slot of its own. Simulated IO submit delay 0 to retire in submit tick; timer with
// duration 0 is caller mistake, never modeled outcome, so guard sit here, once, above every
// backend.
func Timeline_Timeout(
	loop Timeline, completion *Completion, duration time.Duration, callback Callback,
) {
	aver.Always(duration > 0, "A timeout duration is positive.")
	loop.Submit(loop.State, completion, duration, callback)
}

// Timeline_Open_Event keeps event ownership with the backend state that opened it.
func Timeline_Open_Event(loop Timeline) (event Event, err error) {
	return loop.Open_Event(loop.State)
}

// Timeline_Event_Listen keeps listener state on its owning backend.
func Timeline_Event_Listen(
	loop Timeline, event Event, completion *Completion, callback Callback,
) {
	loop.Event_Listen(loop.State, event, completion, callback)
}

// Timeline_Event_Trigger keeps trigger state on its owning backend.
func Timeline_Event_Trigger(loop Timeline, event Event, completion *Completion) {
	loop.Event_Trigger(loop.State, event, completion)
}

// Timeline_Close_Event keeps release on the backend that opened the event.
func Timeline_Close_Event(loop Timeline, event Event) {
	loop.Close_Event(loop.State, event)
}

// Driver advance loop. Only capability that move time and deliver completions.
//
// ===========================================================================
// ONLY PACKAGE MAIN OR A TEST MAY DRIVE, RUN, OR TICK THE EVENT LOOP.
// NOT A LIBRARY. NOT A HELPER. NOT AN INJECTED FUNC VALUE. NOT ONCE.
// A VIOLATION IS AN ARCHITECTURAL BUG EVEN IF EVERY TEST PASSES.
// ===========================================================================
//
// Only code that build Driver hold it or call it: package main in production, or test harness
// in simulation. Library that pump work while its binary own full process. Library fail where
// it compose. Put together with others, it deliver completions of every other application
// from inside own call stack. That destroy absolute order assembly exist to hold.
type Driver struct {
	// State stays caller-owned because binding it into each drive operation would allocate.
	State State
	// Run drain every ready completion without block, then advance clock one tick. ROOT ONLY:
	// never hand to library, never call from library.
	Run func(state State) (err error)
	// Run_For drive loop until duration elapse on clock. Deliver each completion as it come
	// due. Time is GOAL here. Advance exactly duration, drain as it go, whatever complete.
	// Use to let span of time pass, not to wait for one operation.
	// ROOT ONLY: never hand to library, never call from library.
	Run_For func(state State, duration time.Duration) (err error)
	// Run_Until drive loop until done report true. Run-until-complete pump. Straight-line
	// code wait for own operation inline with it. Completion is GOAL here. Time is GUARD.
	// Stop instant done hold. Timeout only cap wait, thus stalled operation cannot hang
	// caller. Run_For put time first, thus two stay separate operations.
	//
	// timeout < 0 panic: unbounded pump put no cap on stalled operation.
	// timeout == 0 check done one time, return without drive. Poll.
	// timeout > 0 pump until done, or until clock pass now+timeout. completed report which
	// win: done (true), or timeout (false).
	//
	// Top-level and single-loop only: never call from inside completion callback.
	// ROOT ONLY: never inject it into library, and never inject func value of its shape.
	// Library that write done predicate and timeout is driving loop.
	Run_Until func(
		state State, timeout time.Duration, done func() (finished bool),
	) (completed bool, err error)
	// Deinit release kernel resources of backend, after every submitted operation join.
	Deinit func(state State)
}

// Driver_Run passes loop state explicitly because a bound driver would allocate.
func Driver_Run(driver Driver) (err error) {
	return driver.Run(driver.State)
}

// Driver_Run_For passes loop state explicitly because a bound driver would allocate.
func Driver_Run_For(driver Driver, duration time.Duration) (err error) {
	return driver.Run_For(driver.State, duration)
}

// Driver_Run_Until passes loop state explicitly because a bound driver would allocate.
func Driver_Run_Until(
	driver Driver, timeout time.Duration, done func() (finished bool),
) (completed bool, err error) {
	return driver.Run_Until(driver.State, timeout, done)
}

// Driver_Deinit releases resources through their owning backend state.
func Driver_Deinit(driver Driver) {
	driver.Deinit(driver.State)
}

// Virtual_Timeline: deterministic loop backend. One ready-time queue, one tick counter, no
// kernel. Sim own it and never hand it out. Run thus reproduce from the counter alone.
// Nothing can script order.
type Virtual_Timeline struct {
	// Resolution: how far the counter advance on each tick. Grain of simulated oscillator.
	Resolution time.Duration
	// Epoch: wall-clock origin at tick zero, before any view's skew.
	Epoch time.Moment
	// Ticks is the one counter every Sim_Clock view read, thus views never drift apart.
	Ticks time.Tick_Count
	// Queue hold armed completions in Ready_At order, earliest first.
	Queue []*Completion
	// Queue_Count separates occupied entries from caller-owned capacity.
	Queue_Count int
	// Events hold cross-thread event entries indexed by handle minus one.
	Events []Virtual_Event
	// Drive_Active true while Run drive. Run called from inside completion callback thus
	// panic, never re-enter driver.
	Drive_Active bool
}

// Virtual_Event: one simulated cross-thread event.
type Virtual_Event struct {
	// Open prevents a closed slot from accepting listener or trigger operations.
	Open bool
	// Armed report listener attached and wait for next trigger.
	Armed bool
	// Triggered coalesces pending wakeups because one listener retirement is one notification.
	Triggered bool
	// Ready prevents repeated trigger from enqueueing one completion more than once.
	Ready bool
	// Listener keeps ownership direct so no listener map or captured callback is needed.
	Listener *Completion
}

// Sim_Clock is one application's view of the shared counter: own skew, same ticks. Root hand
// one to each application because real boxes have separate, skewed clocks, and one loop
// decide what actually happen in what order.
type Sim_Clock struct {
	// Timeline is the counter every view share; the constructor set it.
	Timeline *Virtual_Timeline
	// Skew bend this view's Now_Realtime away from true elapsed time. Seed-drawn, never
	// caller-supplied: a supplied skew is a scripted outcome.
	Skew time.Offset
}

// Sim_Clock_To_Clock build a read-only Clock over one view. Three words, free to build, thus
// nothing store it. Root call it once per application on `&state.Clocks[index]`.
func Sim_Clock_To_Clock(view *Sim_Clock) (host time.Clock) {
	aver.Always(view != nil, "A simulated clock view has caller-owned state.")
	aver.Always(view.Timeline != nil, "A simulated clock view names its timeline.")
	host = time.Clock{
		State:         unsafe.Pointer(view),
		Now_Monotonic: sim_clock_now_monotonic,
		Now_Realtime:  sim_clock_now_realtime,
	}
	time.Clock_Invariants(host, "sim_clock_to_clock.host")
	return host
}

func sim_clock_now_monotonic(state unsafe.Pointer) (moment time.Monotonic_Moment) {
	return virtual_now((*Sim_Clock)(state).Timeline)
}

func sim_clock_now_realtime(state unsafe.Pointer) (moment time.Moment) {
	view := (*Sim_Clock)(state)
	now := view.Timeline.Epoch + time.Moment(virtual_now(view.Timeline))
	return now - time.Moment(time.Offset_Read(view.Skew, view.Timeline.Ticks))
}

// Bind caller-owned capacity so construction cannot allocate.
func virtual_timeline_initialize(
	state *Virtual_Timeline, resolution time.Duration, epoch time.Moment,
	queue []*Completion, events []Virtual_Event,
) {
	aver.Always(len(queue) > 0, "A virtual timeline has queue capacity.")
	aver.Always(len(events) > 0, "A virtual timeline has event capacity.")
	time.Duration_Invariants(resolution, "virtual_timeline_initialize.resolution")
	aver.Always(resolution > 0, "A virtual timeline advances on every tick.")
	// Epoch carry no assertion: the seed draw it below SIM_EPOCH_SECONDS, so no caller can
	// reach an edge of its domain, and an unreachable assertion is a permanent coverage gap.
	for index := range queue {
		queue[index] = nil
	}
	for index := range events {
		events[index] = Virtual_Event{}
	}
	*state = Virtual_Timeline{
		Resolution: resolution,
		Epoch:      epoch,
		Queue:      queue,
		Events:     events,
	}
}

// Wire control plane onto vtable every backend and every caller hold.
func virtual_timeline_to_timeline(state *Virtual_Timeline) (loop Timeline) {
	return Timeline{
		State:         State(state),
		Submit:        virtual_timeline_submit,
		Open_Event:    virtual_timeline_open_event,
		Event_Listen:  virtual_timeline_event_listen,
		Event_Trigger: virtual_timeline_event_trigger,
		Close_Event:   virtual_timeline_close_event,
	}
}

func virtual_timeline_submit(
	state State, completion *Completion, delay time.Duration,
	callback Callback,
) {
	virtual_submit(state.(*Virtual_Timeline), completion, delay, callback)
}

func virtual_timeline_open_event(state State) (event Event, err error) {
	timeline := state.(*Virtual_Timeline)
	for index := range timeline.Events {
		if !timeline.Events[index].Open {
			timeline.Events[index] = Virtual_Event{Open: true}
			return Event(index + 1), nil
		}
	}
	return 0, Virtual_Event_Capacity_Exceeded
}

func virtual_timeline_event_listen(
	state State, event Event, completion *Completion, callback Callback,
) {
	virtual_event_listen(state.(*Virtual_Timeline), event, completion, callback)
}

func virtual_timeline_event_trigger(
	state State, event Event, completion *Completion,
) {
	virtual_event_trigger(state.(*Virtual_Timeline), event, completion)
}

func virtual_timeline_close_event(state State, event Event) {
	timeline := state.(*Virtual_Timeline)
	entry := virtual_event_entry(timeline, event)
	aver.Always(!entry.Armed, "An event listener is drained before close.")
	*entry = Virtual_Event{}
}

// Arm event listener. Deliver at once when trigger already arrive.
func virtual_event_listen(
	state *Virtual_Timeline, event Event, completion *Completion,
	callback Callback,
) {
	entry := virtual_event_entry(state, event)
	aver.Always(!entry.Armed, "An event has at most one armed listener.")
	if entry.Triggered {
		virtual_queue_has_capacity(state)
	}
	virtual_arm(completion, callback)
	completion.Event = event
	entry.Armed = true
	entry.Listener = completion
	if entry.Triggered {
		entry.Triggered = false
		entry.Ready = true
		virtual_enqueue_now(state, completion)
	}
}

// Make armed event listener ready, or record trigger for later listener.
func virtual_event_trigger(state *Virtual_Timeline, event Event, completion *Completion) {
	entry := virtual_event_entry(state, event)
	if !entry.Armed {
		entry.Triggered = true
		return
	}
	aver.Always(entry.Listener == completion,
		"A trigger names the completion its event armed.")
	if entry.Ready {
		entry.Triggered = true
		return
	}
	virtual_queue_has_capacity(state)
	entry.Ready = true
	virtual_enqueue_now(state, completion)
}

func virtual_event_entry(state *Virtual_Timeline, event Event) (entry *Virtual_Event) {
	aver.Always(event > 0, "A virtual event handle is never zero.")
	aver.Always(event <= Event(len(state.Events)),
		"A virtual event handle names caller-owned storage.")
	entry = &state.Events[int(event)-1]
	aver.Always(entry.Open, "A virtual event operation names an open event.")
	return entry
}

// Read simulated moment every Ready_At measure against.
func virtual_now(state *Virtual_Timeline) (now time.Monotonic_Moment) {
	now = time.Monotonic_Moment(int64(state.Ticks) * int64(state.Resolution))
	time.Monotonic_Moment_Invariants(now, "virtual_now.now")
	return now
}

// Schedule completion to fire at now plus delay. Insert it in Ready_At order.
func virtual_submit(
	state *Virtual_Timeline, completion *Completion, delay time.Duration, callback Callback,
) {
	virtual_queue_has_capacity(state)
	virtual_arm(completion, callback)
	completion.Ready_At = virtual_now(state) + time.Monotonic_Moment(delay)
	virtual_enqueue(state, completion)
}

// Arm completion, but never put it on ready-time queue. Event listener use this. Assert
// completion is own original, not by-value copy. Then move it along lifecycle machine.
// Completion armed while armed panic on armed-to-armed edge.
func virtual_arm(completion *Completion, callback Callback) {
	original := completion.Self == nil || completion.Self == completion
	aver.Always(original,
		"A submitted completion is its own original, never a by-value copy.")
	completion.Self = completion
	aver.Always(!completion.Armed, "An armed completion is never armed a second time.")
	completion.Data = 0
	completion.Error = nil
	completion.Armed = true
	completion.Callback = callback
	completion.Event = 0
}

// Put already armed completion on queue as due now.
func virtual_enqueue_now(state *Virtual_Timeline, completion *Completion) {
	completion.Ready_At = virtual_now(state)
	virtual_enqueue(state, completion)
}

// Insert completion into queue in Ready_At order, earliest first.
func virtual_enqueue(state *Virtual_Timeline, completion *Completion) {
	index := 0
	for index < state.Queue_Count && state.Queue[index].Ready_At <= completion.Ready_At {
		index++
	}
	copy(state.Queue[index+1:state.Queue_Count+1], state.Queue[index:state.Queue_Count])
	state.Queue[index] = completion
	state.Queue_Count++
}

// Capacity rejects new ownership before any caller or event state changes.
func virtual_queue_has_capacity(state *Virtual_Timeline) {
	aver.Always(state.Queue_Count < len(state.Queue),
		"A virtual timeline never exceed caller-owned queue capacity.")
}

// Fire earliest completion when due as of now. Report whether it fire.
func virtual_step(state *Virtual_Timeline) (advanced bool) {
	if state.Queue_Count == 0 {
		return false
	}
	if state.Queue[0].Ready_At > virtual_now(state) {
		return false
	}
	completion := state.Queue[0]
	last := state.Queue_Count - 1
	copy(state.Queue[:last], state.Queue[1:state.Queue_Count])
	state.Queue[last] = nil
	state.Queue_Count--
	// Go back to idle before callback run. Callback can then submit own completion again.
	// Repeating-timer pattern.
	aver.Always(completion.Armed, "A delivered completion was armed.")
	completion.Armed = false
	callback := completion.Callback
	completion.Callback = nil
	event := completion.Event
	completion.Event = 0
	if event != 0 {
		entry := virtual_event_entry(state, event)
		entry.Armed = false
		entry.Ready = false
		entry.Listener = nil
	}
	callback(completion)
	return true
}

// Drain every completion due as of now, in Ready_At order. Never advance time. That is job of
// driver, thus queue stay passive.
func virtual_drain(state *Virtual_Timeline) {
	for virtual_step(state) {
	}
}

// Step of driver: drain what is due, then advance clock one grain.
//
// Driver advance one grain per tick. Never jump ahead to next Ready_At, even when queue idle
// until then. Jump would skip grains where time-triggered fault adversary act. Those faults
// crash or partition quiet node. They matter most: they strike while nothing scheduled.
func virtual_run(state *Virtual_Timeline) {
	virtual_drain(state)
	state.Ticks++
}

// Drive until duration elapse. Deliver completions as they come due.
func virtual_run_for(state *Virtual_Timeline, duration time.Duration) {
	deadline := virtual_now(state) + time.Monotonic_Moment(duration)
	for virtual_now(state) < deadline {
		virtual_run(state)
	}
}

// Drive until done report true, or until timeout of virtual time elapse. Run-until-complete
// pump. Cap stop stalled operation from spin without end. Negative timeout is that uncapped
// pump, thus it panic. Never hand caller drive with no bound.
func virtual_run_until(
	state *Virtual_Timeline, timeout time.Duration, done func() (finished bool),
) (completed bool) {
	aver.Always(timeout >= 0, "A Run_Until timeout is never negative.")
	deadline := virtual_now(state) + time.Monotonic_Moment(timeout)
	for !done() {
		if virtual_now(state) >= deadline {
			return false
		}
		virtual_run(state)
	}
	return true
}

// Drive begin rejects reentrancy before any queue state can change.
func virtual_drive_begin(state *Virtual_Timeline) {
	aver.Always(!state.Drive_Active,
		"A drive begins at top level, never from within a completion callback.")
	state.Drive_Active = true
}

// Drive end remains deferred so callback panic cannot leave the driver permanently active.
func virtual_drive_end(state *Virtual_Timeline) {
	state.Drive_Active = false
}

// Build driver over state. Capability that advance time. Only main or test hold it.
func virtual_timeline_to_driver(state *Virtual_Timeline) (driver Driver) {
	return Driver{
		State:     State(state),
		Run:       virtual_driver_run,
		Run_For:   virtual_driver_run_for,
		Run_Until: virtual_driver_run_until,
		Deinit:    virtual_driver_deinit,
	}
}

func virtual_driver_run(state State) (err error) {
	timeline := state.(*Virtual_Timeline)
	virtual_drive_begin(timeline)
	defer virtual_drive_end(timeline)
	virtual_run(timeline)
	return nil
}

func virtual_driver_run_for(state State, duration time.Duration) (err error) {
	timeline := state.(*Virtual_Timeline)
	virtual_drive_begin(timeline)
	defer virtual_drive_end(timeline)
	virtual_run_for(timeline, duration)
	return nil
}

func virtual_driver_run_until(
	state State, timeout time.Duration, done func() (finished bool),
) (completed bool, err error) {
	timeline := state.(*Virtual_Timeline)
	virtual_drive_begin(timeline)
	defer virtual_drive_end(timeline)
	return virtual_run_until(timeline, timeout, done), nil
}

func virtual_driver_deinit(state State) {
	aver.Always(state != nil, "A virtual driver deinitializes caller-owned state.")
}
