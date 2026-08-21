// Package nbio is dependency-injected completion surface of repository.
//
// No generic Cancel: owner use Shutdown, join every submitted operation, then submit Close.
// Repository-specific effect is marked extension and must retire through same completed queue.
package nbio

import (
	"errors"

	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/random/prng"
	"local/james-orcales/shared/simulation/time"
	"local/james-orcales/shared/slices"
)

// IO is injected async IO submit surface. Code submit operation with
// time.Completion and callback, then react to completion. Code never drive loop — that is
// time.Driver job — thus holder submit IO, but cannot advance time.
//
// This is I/O seam, not kitchen sink of syscalls. Every member is data transfer with external
// endpoint — file, socket, pipe — or loop own control plane that schedule those transfers
// (timer, signal watch, spawn). Not all syscalls are I/O operations: getrandom, getpid, mmap,
// and nanosleep are syscalls, but transfer no data with endpoint, thus they do not belong here.
// Dependency that need one of those (secure_transport entropy, for instance) declare it itself
// and take it injected, rather than widen this surface.
//
// Seam cut two ways, thus IO hold two halves. Network carry every socket endpoint, Storage every
// file endpoint, and holder take half it use: package that only write files hold Storage and
// cannot reach socket. Close and Deinit stay flat here, because both name descriptor, and both
// half hand descriptors out.
//
// CRITICAL: ONLY PACKAGE MAIN OR A TEST MAY DRIVE, RUN, OR TICK THE EVENT LOOP. IO holder that
// want to wait expose doneness as state and let root pump. It never receive pump. See
// shared/io/README.md, THE CRITICAL GUARANTEE.
type IO struct {
	Platform_IO
	// Network is every transfer whose endpoint is socket.
	Network Network
	// Storage is every transfer whose endpoint is file or directory.
	Storage Storage
	// Close release descriptor of file. Callback fire once it is closed. It sit here, not on
	// one half: close(2) name descriptor, and both half hand descriptors out.
	Close func(completion *time.Completion, file File, callback time.Callback)
	// Deinit assert every descriptor run open is closed. Surface own leak check, thus leaked
	// descriptor fail run. Caller need no census it must remember to read.
	Deinit func()
}

// Network is half of IO: every transfer whose endpoint is socket, plus socket lifecycle that
// open one. Holder that only speak to socket take this and cannot reach file.
//
// TLS is not member. No accept-secure syscall, no connect-secure syscall, thus secure_transport
// compose these raw operations under its own record layer.
type Network struct {
	// Socket_TCP make one configured non-blocking close-on-exec TCP socket.
	Socket_TCP func(family Address_Family, options TCP_Options) (socket File, err error)
	// Socket_UDP make one configured non-blocking close-on-exec UDP socket.
	Socket_UDP func(family Address_Family, options UDP_Options) (socket File, err error)
	// Bind enable address reuse and give caller-owned socket its local address.
	Bind func(socket File, address Address) (err error)
	// Listen_Socket mark bound socket accepting, with backlog as queue depth.
	Listen_Socket func(socket File, backlog uint32) (err error)
	// Get_Socket_Name report socket own address. That is how caller learn port kernel chose
	// for port zero.
	Get_Socket_Name func(socket File) (address Address, err error)
	// Accept yield one inbound connection on listener before timeout, or report
	// Deadline_Exceeded. Timeout is finite deliberately: it keep every repository
	// submission bounded.
	Accept func(
		completion *time.Completion, listener File, timeout time.Duration,
		callback time.Callback,
	)
	// Connect borrow caller-owned socket until one kernel result or timeout. Callback report
	// outcome only. Backend never make, transfer, or close descriptor.
	Connect func(
		completion *time.Completion, socket File, address Address, timeout time.Duration,
		callback time.Callback,
	)
	// Receive read up to len(buffer) bytes before timeout. Callback report byte count once data
	// arrive, or zero with Deadline_Exceeded after the kernel request retire.
	Receive func(
		completion *time.Completion, socket File, buffer []byte, timeout time.Duration,
		callback time.Callback,
	)
	// Send write buffer before timeout. Callback report byte count once kernel accept it, or
	// zero with Deadline_Exceeded after the kernel request retire.
	Send func(
		completion *time.Completion, socket File, buffer []byte, timeout time.Duration,
		callback time.Callback,
	)
	// Shutdown synchronously disable one or both connected-socket direction. It neither own nor
	// close socket.
	Shutdown func(socket File, how Shutdown_How) (err error)
	// Peer_Address return remote IP address of connected socket, synchronously — getpeername
	// has no completion. It is source a control-plane connection is gated on.
	Peer_Address func(file File) (address string, err error)
}

// Storage is other half of IO: every transfer whose endpoint is file or directory, plus path
// primitives that name one. Holder that only read and write file take this and cannot reach
// socket.
type Storage struct {
	// Read read len(buffer) bytes from file at offset before timeout. Timeout does not work on
	// Darwin because its current file path has no kernel timeout. Darwin completes the
	// operation.
	Read func(
		completion *time.Completion, file File, buffer []byte, offset int64,
		timeout time.Duration,
		callback time.Callback,
	)
	// Write write buffer to file at offset before timeout. Timeout does not work on Darwin
	// because its current file path has no kernel timeout. Darwin completes the operation.
	Write func(
		completion *time.Completion, file File, buffer []byte, offset int64,
		timeout time.Duration,
		callback time.Callback,
	)
	// Fsync synchronize file before timeout. Timeout does not work on Darwin because its
	// current file path has no kernel timeout. Darwin completes the operation.
	Fsync func(
		completion *time.Completion, file File, timeout time.Duration,
		callback time.Callback,
	)
	// Open_At asynchronously open file_path relative to directory and force close-on-exec.
	Open_At func(
		completion *time.Completion, directory File, file_path string,
		options Open_At_Options, callback time.Callback,
	)
	// Mkdir_At asynchronously make one directory named by file_path relative to directory. It
	// is mkdirat primitive, not parent-creating mkdir: parent must exist, and existing path
	// report operating system error rather than converge. Make_Directory compose this
	// primitive above surface, thus both backend run same composition.
	Mkdir_At func(
		completion *time.Completion, directory File, file_path string, mode uint32,
		callback time.Callback,
	)
	// Get_Directory_Entries read one pass of directory raw entries into buffer and return
	// children it name, each with whether it is itself directory. Empty slice report end.
	// Dirent layout is per-platform, thus parse and kind stay in backend, and only pass loop
	// and descriptor lifetime compose above.
	Get_Directory_Entries func(
		completion *time.Completion, directory File, buffer []byte,
		callback Directory_Callback,
	)
	// Status report whether path exist, whether it is directory, and its byte size,
	// synchronously. Absent path is Exists false with nil error, thus caller branch on status,
	// not on error.
	Status func(path string) (status File_Status, err error)
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
	// Mode is permission mode, used only when Create is true.
	Mode uint32
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

// Address is IP address and port with explicit family. IP store IPv4 bytes in first four
// positions, IPv6 bytes in all 16.
type Address struct {
	// Family select how IP is read.
	Family Address_Family
	// IP hold network address bytes.
	IP [IPV6_ADDRESS_BYTES]byte
	// Port is host-order TCP or UDP port.
	Port uint16
}

// Address_IPV4 return IPv4 address from four octets and host-order port.
func Address_IPV4(ip [IPV4_ADDRESS_BYTES]byte, port uint16) (address Address) {
	address.Family = FAMILY_IPV4
	copy(address.IP[:4], ip[:])
	address.Port = port
	return address
}

// Address_IPV6 return IPv6 address from 16 octets and host-order port.
func Address_IPV6(ip [IPV6_ADDRESS_BYTES]byte, port uint16) (address Address) {
	return Address{Family: FAMILY_IPV6, IP: ip, Port: port}
}

// Address_Parse parse IP literal without DNS and return explicit-family address. Host that hold
// colon parse as IPv6, else as IPv4. Anything else error.
//
// Parse is written here, not taken from net/netip, thus this tier import no part of net tree. One
// import of net is one step from a resolver, and resolver block. Address is 20 bytes of plain
// data, and its text form is small enough to read here.
func Address_Parse(host string, port int) (address Address, err error) {
	if port < 0 {
		return Address{}, errors.New("io: port is outside uint16")
	}
	if port > 65535 {
		return Address{}, errors.New("io: port is outside uint16")
	}
	if len(host) > ADDRESS_TEXT_BYTES_MAXIMUM {
		return Address{}, errors.New("io: host is too large")
	}
	if text_contains(host, ":") {
		sextets, found := address_parse_ipv6(host)
		if !found {
			return Address{}, errors.New("io: host is not an IP literal")
		}
		return Address_IPV6(sextets, uint16(port)), nil
	}
	quad, found := address_parse_ipv4(host)
	if !found {
		return Address{}, errors.New("io: host is not an IP literal")
	}
	return Address_IPV4(quad, uint16(port)), nil
}

// Parse dotted-quad into four octets.
func address_parse_ipv4(host string) (quad [IPV4_ADDRESS_BYTES]byte, found bool) {
	parts, split := text_split(host, '.', IPV4_ADDRESS_BYTES)
	if !split {
		return [IPV4_ADDRESS_BYTES]byte{}, false
	}
	if len(parts) != IPV4_ADDRESS_BYTES {
		return [IPV4_ADDRESS_BYTES]byte{}, false
	}
	for index, part := range parts {
		octet, valid := address_parse_octet(part)
		if !valid {
			return [IPV4_ADDRESS_BYTES]byte{}, false
		}
		quad[index] = octet
	}
	return quad, true
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
func address_parse_ipv6(host string) (sextets [IPV6_ADDRESS_BYTES]byte, found bool) {
	if text_contains(host, "%") {
		return [IPV6_ADDRESS_BYTES]byte{}, false
	}
	head, tail, compressed := text_cut(host, "::")
	if compressed {
		return address_join_ipv6(head, tail)
	}
	values, valid := address_parse_ipv6_parts(host)
	if !valid {
		return [IPV6_ADDRESS_BYTES]byte{}, false
	}
	if len(values) != IPV6_ADDRESS_BYTES {
		return [IPV6_ADDRESS_BYTES]byte{}, false
	}
	copy(sextets[:], values)
	return sextets, true
}

// Join both halves of a compressed literal, with zeros between them. Run must stand for at least
// one group, thus the two halves together leave two bytes free.
func address_join_ipv6(head string, tail string) (sextets [IPV6_ADDRESS_BYTES]byte, found bool) {
	if text_contains(tail, "::") {
		return [IPV6_ADDRESS_BYTES]byte{}, false
	}
	front, front_valid := address_parse_ipv6_parts(head)
	if !front_valid {
		return [IPV6_ADDRESS_BYTES]byte{}, false
	}
	back, back_valid := address_parse_ipv6_parts(tail)
	if !back_valid {
		return [IPV6_ADDRESS_BYTES]byte{}, false
	}
	if len(front)+len(back) > IPV6_ADDRESS_BYTES-2 {
		return [IPV6_ADDRESS_BYTES]byte{}, false
	}
	copy(sextets[:len(front)], front)
	copy(sextets[IPV6_ADDRESS_BYTES-len(back):], back)
	return sextets, true
}

// Parse one colon-separated group run into its bytes. Empty text yield no bytes. Trailing
// dotted-quad contribute four bytes, which is how embedded IPv4 reach the low 32 bits.
func address_parse_ipv6_parts(text string) (values []byte, valid bool) {
	values = []byte{}
	if text == "" {
		return values, true
	}
	parts, split := text_split(text, ':', ADDRESS_IPV6_PARTS_MAX)
	if !split {
		return nil, false
	}
	for index, part := range parts {
		if index == len(parts)-1 {
			if text_contains(part, ".") {
				quad, quad_valid := address_parse_ipv4(part)
				if !quad_valid {
					return nil, false
				}
				return append(values, quad[:]...), true
			}
		}
		high, low, group_valid := address_parse_group(part)
		if !group_valid {
			return nil, false
		}
		values = append(values, high, low)
	}
	return values, true
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
	source string, separator byte, part_limit_count int,
) (parts []string, valid bool) {
	parts = make([]string, 0, part_limit_count)
	start := 0
	for position := range source {
		if source[position] != separator {
			continue
		}
		if len(parts) >= part_limit_count {
			return nil, false
		}
		parts = append(parts, source[start:position])
		start = position + 1
	}
	if len(parts) >= part_limit_count {
		return nil, false
	}
	parts = append(parts, source[start:])
	return parts, true
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

// Listen_Options are options applied after bind of caller-owned socket.
type Listen_Options struct {
	// Backlog is requested completed-connection queue size.
	Backlog uint32
}

// Shutdown_How select which connected-socket direction shutdown disable.
type Shutdown_How int

// SHUTDOWN_RECEIVE disable further receive.
const SHUTDOWN_RECEIVE Shutdown_How = 0

// SHUTDOWN_SEND disable further send.
const SHUTDOWN_SEND Shutdown_How = 1

// SHUTDOWN_BOTH disable receive and send.
const SHUTDOWN_BOTH Shutdown_How = 2

// Directory_Callback receive one pass of directory entries, or error that end walk. Empty slice
// with nil error report end of directory.
type Directory_Callback func(
	completion *time.Completion, entries []Directory_Entry, err error,
)

// Directory_Entry is one child of directory: name, and whether it is itself directory. Those two
// facts are what walk need to recurse into subdirectory and sync file.
type Directory_Entry struct {
	// Name is child name inside its directory, not full path.
	Name string
	// Is_Directory report whether child is directory, thus walk know to recurse.
	Is_Directory bool
}

// File_Status is what stat report: whether path exist, and if so whether it is directory and its
// byte length — metadata mirror consult before it read or write.
type File_Status struct {
	// Exists report whether path is present. Absent path is not error.
	Exists bool
	// Is_Directory report whether existing path is directory, not file.
	Is_Directory bool
	// Is_Regular report whether existing path is regular file.
	Is_Regular bool
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
// not API to cancel operation: shared/io deliberately has no generic Cancel.
var Canceled = errors.New("io: operation canceled")

// Path_Exists is portable Mkdir_At result for path already present. Make_Directory treat it as
// convergence, thus repeated create is not error.
var Path_Exists = errors.New("io: file exists")

// Number of virtual grains one simulated operation may take to complete, drawn from seed, thus
// completion order vary per run and still reproduce.
const SIM_LATENCY_GRAINS = 8

// One in this many simulated spawns exit non-zero, thus seed sweep reach both success path and
// failure path without scripted outcome.
const SIM_SPAWN_FAIL_GRAINS = 4

// Returned when path resolve to nothing — simulator ENOENT.
var sim_file_absent = errors.New("io: no such file or directory")

// Returned when path component that must be directory is file.
var sim_not_a_directory = errors.New("io: not a directory")

// Returned when file operation name directory.
var sim_is_a_directory = errors.New("io: is a directory")

// A directory cannot cross the shared slice boundary when it is read back.
var sim_directory_full = errors.New("io: directory entry limit exceeded")

// Sim_Node is one entry in simulator in-memory filesystem: directory with named children, or
// file holding bytes. Generated from seed at New_Simulated_IO and mutated by
// Create/Write/Make_Directory, thus later read reflect earlier write.
type Sim_Node struct {
	// Directory report whether this node is directory, not file.
	Directory bool
	// Contents hold file bytes. Nil for directory.
	Contents []byte
	// Children map directory entry names to their nodes. Nil for file.
	Children map[string]*Sim_Node
}

// Sim_Socket is simulator caller-owned socket state. Shutdown is directional and never release
// ownership. Close is only operation that remove entry.
type Sim_Socket struct {
	// Family is address family selected at creation.
	Family Address_Family
	// Datagram tell UDP apart from TCP.
	Datagram bool
	// Connected report whether Connect or Accept established socket.
	Connected bool
	// Listener report whether Listen_Socket consumed this socket.
	Listener bool
	// Address is local address Bind gave socket.
	Address Address
	// Receive_Shutdown record SHUTDOWN_RECEIVE or SHUTDOWN_BOTH.
	Receive_Shutdown bool
	// Send_Shutdown record SHUTDOWN_SEND or SHUTDOWN_BOTH.
	Send_Shutdown bool
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
// New_Simulated_IO is only entry, it take seed and pump alone, and it return IO alone. No caller
// ever hold *Sim, thus no caller can hang field on one. If you find yourself wanting to return
// it, or wanting to add parameter that is neither seed nor pump, stop — that is scripting API
// trying to come back. Keep it shut.
type Sim struct {
	// Timeline is control plane every simulated operation retire through. Queue, tick, and
	// order live in shared/time, thus this backend arm work and can never advance it.
	Timeline time.Timeline
	// Generator excludes equal-time ordering so timeout races cannot perturb ordinary outcomes.
	Generator prng.Generator
	// Timeout_Order_Generator isolates equal-time kernel ordering from operation outcomes.
	Timeout_Order_Generator prng.Generator
	// Storage_Timeout_Order_Generator isolates storage ordering from the existing network
	// stream.
	Storage_Timeout_Order_Generator prng.Generator
	// Next_File is synthetic descriptor counter. Listen, Accept, Open_Socket, Open, and Create
	// hand out next value, thus every descriptor is distinct.
	Next_File File
	// Root is in-memory filesystem file operations read and mutate, fabricated from seed at
	// New_Simulated_IO. Socket descriptors ignore it.
	Root *Sim_Node
	// Files bind open file descriptor to its node, thus Read/Write route to real tree bytes.
	Files map[File]*Sim_Node
	// Directory_Drained record descriptors whose entries one pass already reported, thus second
	// Get_Directory_Entries report end, same as real getdents.
	Directory_Drained map[File]bool
	// Raw_Open track every synthetic descriptor until caller submit Close.
	Raw_Open map[File]bool
	// Sockets hold lifecycle and directional-shutdown state of synthetic sockets.
	Sockets map[File]*Sim_Socket
	// Operation_Files make Close reject a descriptor until each borrower retires.
	Operation_Files map[*time.Completion]File
}

// New_Simulated_IO return deterministic submit surface seeded by seed. Sim never escape, and
// seed is only input, thus run reproduce exact and nothing script into it — invariants assert
// correctness, not hand-fed outcome. Sim read no clock: seed draw every latency, and pump own
// now. Driver stay in harness: program under test hold loop, NEVER pump (see time.Driver banner).
func New_Simulated_IO(seed uint64, pump time.Timeline) (loop IO) {
	root_generator := prng.New(seed)
	state := &Sim{
		Timeline:                        pump,
		Generator:                       prng.Generator_Split(&root_generator),
		Timeout_Order_Generator:         prng.Generator_Split(&root_generator),
		Storage_Timeout_Order_Generator: prng.Generator_Split(&root_generator),
		Files:                           map[File]*Sim_Node{},
		Directory_Drained:               map[File]bool{},
		Raw_Open:                        map[File]bool{},
		Sockets:                         map[File]*Sim_Socket{},
		Operation_Files:                 map[*time.Completion]File{},
	}
	state.Root = sim_generate(&state.Generator)
	sim_wire_network(state, &loop.Network)
	sim_wire_storage(state, &loop.Storage)
	sim_wire_platform(state, &loop)
	loop.Close = func(completion *time.Completion, file File, callback time.Callback) {
		sim_assert_file_drained(state, file)
		sim_submit(
			state, completion, sim_latency(state), func(completion *time.Completion) {
				delete(state.Files, file)
				delete(state.Sockets, file)
				delete(state.Raw_Open, file)
				sim_deliver(completion, 0, nil, callback)
			})
	}
	loop.Deinit = func() {
		invariant.Always(len(state.Raw_Open) == 0,
			"Every descriptor a simulated run opened is closed before Deinit.")
	}
	return loop
}

// Wire whole socket half of simulator onto network.
func sim_wire_network(state *Sim, network *Network) {
	sim_wire_socket_lifecycle(state, network)
	sim_wire_socket_bytes(state, network)
}

// Wire whole file half of simulator onto storage.
func sim_wire_storage(state *Sim, storage *Storage) {
	sim_wire_path(state, storage)
	sim_wire_file_bytes(state, storage)
	sim_wire_directory(state, storage)
}

// Wire socket lifecycle: descriptor a socket come from, options and address it carry, and
// direction shutdown disable.
func sim_wire_socket_lifecycle(state *Sim, network *Network) {
	network.Socket_TCP = func(
		family Address_Family, options TCP_Options,
	) (socket File, err error) {
		if !tcp_options_valid(options) {
			return File(-1), errors.New("io: invalid socket options")
		}
		return sim_open_socket(state, family, false), nil
	}
	network.Socket_UDP = func(
		family Address_Family, options UDP_Options,
	) (socket File, err error) {
		if !udp_options_valid(options) {
			return File(-1), errors.New("io: invalid socket options")
		}
		return sim_open_socket(state, family, true), nil
	}
	network.Bind = func(socket File, address Address) (err error) {
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
	network.Listen_Socket = func(socket File, backlog uint32) (err error) {
		return sim_listen(state, socket, backlog)
	}
	network.Get_Socket_Name = func(socket File) (address Address, err error) {
		socket_state := state.Sockets[socket]
		if socket_state == nil {
			return Address{}, errors.New("io: getsockname requires an open socket")
		}
		return socket_state.Address, nil
	}
	network.Shutdown = func(socket File, how Shutdown_How) (err error) {
		return sim_shutdown(state, socket, how)
	}
	network.Peer_Address = func(file File) (address string, err error) {
		return sim_peer_address(state, file), nil
	}
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

// Wire socket transfers: two that establish connection, and three that move bytes across one.
func sim_wire_socket_bytes(state *Sim, network *Network) {
	network.Accept = func(
		completion *time.Completion, listener File, timeout time.Duration,
		callback time.Callback,
	) {
		invariant.Always(timeout > 0, "An accept timeout is positive and finite.")
		sim_yield_socket(state, completion, listener, timeout, callback)
		state.Operation_Files[completion] = listener
	}
	network.Connect = func(
		completion *time.Completion, socket File, address Address, timeout time.Duration,
		callback time.Callback,
	) {
		sim_connect(state, completion, socket, address, timeout, callback)
	}
	network.Receive = func(
		completion *time.Completion, socket File, buffer []byte, timeout time.Duration,
		callback time.Callback,
	) {
		sim_receive(state, completion, socket, buffer, timeout, callback)
	}
	network.Send = func(
		completion *time.Completion, socket File, buffer []byte, timeout time.Duration,
		callback time.Callback,
	) {
		sim_send(state, completion, socket, buffer, timeout, callback)
	}
}

// Mark socket accepting. Datagram socket and zero backlog are both rejected, same as real listen.
func sim_listen(state *Sim, socket File, backlog uint32) (err error) {
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

// Disable selected direction on socket without release of ownership.
func sim_shutdown(state *Sim, socket File, how Shutdown_How) (err error) {
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

// Wire two path primitives, Open_At and Mkdir_At, onto storage.
func sim_wire_path(state *Sim, storage *Storage) {
	storage.Open_At = func(
		completion *time.Completion, directory File, file_path string,
		options Open_At_Options, callback time.Callback,
	) {
		invariant.Always(
			options.Flags & ^OPEN_AT_NO_FOLLOW == 0,
			"Open_At options contain only known flags.",
		)
		sim_submit(
			state, completion, sim_latency(state), func(completion *time.Completion) {
				file := File(-1)
				var open_err error
				if directory != DIRECTORY_CURRENT {
					open_err = sim_not_a_directory
				} else if options.Create {
					file, open_err = sim_create(state, file_path)
				} else {
					file, open_err = sim_open(state, file_path)
				}
				sim_deliver(completion, int(file), open_err, callback)
			})
	}
	storage.Mkdir_At = func(
		completion *time.Completion, directory File, file_path string, _ uint32,
		callback time.Callback,
	) {
		sim_submit(
			state, completion, sim_latency(state), func(completion *time.Completion) {
				if directory != DIRECTORY_CURRENT {
					sim_deliver(completion, 0, sim_not_a_directory, callback)
					return
				}
				sim_deliver(completion, 0, sim_mkdir(state, file_path), callback)
			})
	}
}

// Wire file transfers onto storage. Read and Write name file, never socket. Assert stand at
// submit, thus caller that meant Network.Receive fail here rather than read fabricated byte
// count.
func sim_wire_file_bytes(state *Sim, storage *Storage) {
	storage.Read = func(
		completion *time.Completion, file File, buffer []byte, offset int64,
		timeout time.Duration,
		callback time.Callback,
	) {
		invariant.Always(timeout > 0, "A storage read timeout is positive and finite.")
		node := state.Files[file]
		invariant.Always(node != nil, "A Storage read names an open file.")
		sim_file_read(state, completion, node, buffer, offset, timeout, callback)
		state.Operation_Files[completion] = file
	}
	storage.Write = func(
		completion *time.Completion, file File, buffer []byte, offset int64,
		timeout time.Duration,
		callback time.Callback,
	) {
		invariant.Always(timeout > 0, "A storage write timeout is positive and finite.")
		node := state.Files[file]
		invariant.Always(node != nil, "A Storage write names an open file.")
		sim_file_write(state, completion, node, buffer, offset, timeout, callback)
		state.Operation_Files[completion] = file
	}
	storage.Fsync = func(
		completion *time.Completion, file File, timeout time.Duration,
		callback time.Callback,
	) {
		invariant.Always(timeout > 0, "A storage fsync timeout is positive and finite.")
		latency := sim_latency(state)
		if sim_storage_timeout_first(state, latency, timeout) {
			sim_submit(state, completion, timeout, func(completion *time.Completion) {
				sim_deliver(completion, 0, time.Deadline_Exceeded, callback)
			})
			state.Operation_Files[completion] = file
			return
		}
		sim_submit(state, completion, latency, func(completion *time.Completion) {
			sim_deliver(completion, 0, nil, callback)
		})
		state.Operation_Files[completion] = file
	}
}

// Wire two directory readers onto storage: one pass of descriptor entries, and synchronous stat
// of path.
func sim_wire_directory(state *Sim, storage *Storage) {
	storage.Get_Directory_Entries = func(
		completion *time.Completion, directory File, _ []byte, callback Directory_Callback,
	) {
		sim_submit(
			state, completion, sim_latency(state), func(completion *time.Completion) {
				callback(completion, sim_directory_pass(state, directory), nil)
			})
	}
	storage.Status = func(path string) (status File_Status, err error) {
		return sim_status(state.Root, path), nil
	}
}

// Assert no submitted operation still borrow file. Teardown owner do this join before close, and
// library surface enforce that ownership boundary, thus descriptor reuse cannot race old
// operation.
func sim_assert_file_drained(state *Sim, file File) {
	borrowed := false
	for _, operation_file := range state.Operation_Files {
		if operation_file == file {
			borrowed = true
		}
	}
	invariant.Always(!borrowed,
		"A descriptor is drained before Close releases it.")
}

// A receive timeout retires the borrow before delivery, so caller-owned close stays ordered.
func sim_receive(
	state *Sim, completion *time.Completion, socket File, buffer []byte,
	timeout time.Duration, callback time.Callback,
) {
	invariant.Always(timeout > 0, "A receive timeout is positive and finite.")
	latency := sim_latency(state)
	if sim_timeout_first(state, latency, timeout) {
		sim_submit(state, completion, timeout, func(completion *time.Completion) {
			sim_deliver(completion, 0, time.Deadline_Exceeded, callback)
		})
		state.Operation_Files[completion] = socket
		return
	}
	sim_submit(state, completion, latency, func(completion *time.Completion) {
		socket_state := state.Sockets[socket]
		if socket_state == nil {
			sim_deliver(completion, 0, nil, callback)
			return
		}
		if socket_state.Receive_Shutdown {
			sim_deliver(completion, 0, nil, callback)
			return
		}
		sim_deliver(completion, len(buffer), nil, callback)
	})
	state.Operation_Files[completion] = socket
}

// A send timeout retires the borrow before delivery, so caller-owned close stays ordered.
func sim_send(
	state *Sim, completion *time.Completion, socket File, buffer []byte,
	timeout time.Duration, callback time.Callback,
) {
	invariant.Always(timeout > 0, "A send timeout is positive and finite.")
	latency := sim_latency(state)
	if sim_timeout_first(state, latency, timeout) {
		sim_submit(state, completion, timeout, func(completion *time.Completion) {
			sim_deliver(completion, 0, time.Deadline_Exceeded, callback)
		})
		state.Operation_Files[completion] = socket
		return
	}
	sim_submit(state, completion, latency, func(completion *time.Completion) {
		socket_state := state.Sockets[socket]
		if socket_state == nil {
			sim_deliver(completion, 0, Broken_Pipe, callback)
			return
		}
		if socket_state.Send_Shutdown {
			sim_deliver(completion, 0, Broken_Pipe, callback)
			return
		}
		sim_deliver(completion, len(buffer), nil, callback)
	})
	state.Operation_Files[completion] = socket
}

// Submit one simulator Connect with one latency draw and independent equal-time order.
func sim_connect(
	state *Sim, completion *time.Completion, socket File, _ Address, timeout time.Duration,
	callback time.Callback,
) {
	invariant.Always(timeout > 0, "A connect timeout is positive and finite.")
	connect_err := error(nil)
	if prng.Generator_Below(&state.Generator, 4) == 0 {
		connect_err = Connection_Refused
	}
	latency := sim_latency(state)
	if sim_timeout_first(state, latency, timeout) {
		sim_submit(state, completion, timeout, func(completion *time.Completion) {
			sim_deliver(completion, 0, time.Deadline_Exceeded, callback)
		})
		state.Operation_Files[completion] = socket
		return
	}
	sim_submit(state, completion, latency, func(completion *time.Completion) {
		if connect_err == nil {
			socket_state := state.Sockets[socket]
			if socket_state != nil {
				socket_state.Connected = true
			}
		}
		sim_deliver(completion, 0, connect_err, callback)
	})
	state.Operation_Files[completion] = socket
}

// Hand out next distinct synthetic descriptor.
func sim_descriptor(state *Sim) (file File) {
	state.Next_File++
	return state.Next_File
}

// Open and record one caller-owned synthetic socket.
func sim_open_socket(
	state *Sim, family Address_Family, datagram bool,
) (socket File) {
	socket = sim_descriptor(state)
	state.Raw_Open[socket] = true
	state.Sockets[socket] = &Sim_Socket{Family: family, Datagram: datagram}
	return socket
}

// Return fresh empty directory node, shape root, mkdir, and generator all build.
func sim_new_directory() (node *Sim_Node) {
	return &Sim_Node{Directory: true, Children: map[string]*Sim_Node{}}
}

// SIM_PATH_TEXT_BYTES_MAXIMUM keeps one simulated path within the repository text budget.
const SIM_PATH_TEXT_BYTES_MAXIMUM = 4 * bits.KIBIBYTE_BYTES

// Split absolute path into non-empty component names, thus "/a/b" walk as a, b.
func sim_path_names(path string) (names []string) {
	invariant.Always(
		len(path) <= SIM_PATH_TEXT_BYTES_MAXIMUM,
		"A simulated path stays inside the repository text budget.",
	)
	names = []string{}
	start := 0
	for position := 0; position <= len(path); position++ {
		if position < len(path) {
			if path[position] != '/' {
				continue
			}
		}
		if position > start {
			names = append(names, path[start:position])
		}
		start = position + 1
	}
	return names
}

// Make exactly one directory at path against state.Root. This is mkdirat primitive, thus parent
// must already exist, and existing path is error. Make_Directory supply convergence and parent
// creation above surface.
func sim_mkdir(state *Sim, path string) (err error) {
	names := sim_path_names(path)
	if len(names) == 0 {
		return Path_Exists
	}
	parent, found := sim_resolve_names(state.Root, names[:len(names)-1])
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
	if len(parent.Children) >= slices.SLICE_COUNT_MAXIMUM {
		return sim_directory_full
	}
	parent.Children[name] = sim_new_directory()
	return nil
}

// Resolve path against root. Return node it name, and whether it was found.
func sim_resolve(root *Sim_Node, path string) (node *Sim_Node, found bool) {
	return sim_resolve_names(root, sim_path_names(path))
}

func sim_resolve_names(root *Sim_Node, names []string) (node *Sim_Node, found bool) {
	node = root
	for _, name := range names {
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

// Report path status against root: unresolved path is Exists false, else its kind.
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

// List path immediate children sorted by name — sorted, thus order is deterministic despite
// backing map, since run must reproduce. Absent path and non-directory path both error.
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
	slices.Sort_Function(entries, directory_entry_compare)
	return entries, nil
}

// Return one directory pass for open directory descriptor. Simulated tree fit in one pass, thus
// first call give every child and second give none. That is how real getdents report end.
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
	slices.Sort_Function(entries, directory_entry_compare)
	return entries
}

// Map order must become explicit because the simulator cannot import the stream-dependent text
// package that normally owns repository string comparison.
func directory_entry_compare(
	left Directory_Entry, right Directory_Entry,
) (order slices.Comparison) {
	if left.Name < right.Name {
		return slices.ORDERING_LESS
	}
	if left.Name > right.Name {
		return slices.ORDERING_GREATER
	}
	return slices.ORDERING_EQUAL
}

// Make path and any missing parent against root. Existing directory converge. File where
// directory is needed error.
func sim_make_directory(root *Sim_Node, path string) (err error) {
	node := root
	for _, name := range sim_path_names(path) {
		if !node.Directory {
			return sim_not_a_directory
		}
		child, present := node.Children[name]
		if !present {
			if len(node.Children) >= slices.SLICE_COUNT_MAXIMUM {
				return sim_directory_full
			}
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

// Open path for read against state.Root, binding fresh descriptor to its node. Absent path, or
// directory, error — same as real open of missing or non-file path.
func sim_open(state *Sim, path string) (file File, err error) {
	node, found := sim_resolve(state.Root, path)
	if !found {
		return 0, sim_file_absent
	}
	// Directory open, because Read_Directory compose Open_At with Get_Directory_Entries. Read
	// of descriptor still fail, same as read of real directory.
	descriptor := sim_descriptor(state)
	state.Files[descriptor] = node
	state.Raw_Open[descriptor] = true
	return descriptor, nil
}

// Make or truncate file at path against state.Root and bind fresh descriptor to it. Parent
// directory must already exist, same as real create.
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

// Resolve path parent, which must be existing directory, then make fresh file node under it, or
// truncate existing file. Return node.
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
	if len(parent.Children) >= slices.SLICE_COUNT_MAXIMUM {
		return nil, sim_directory_full
	}
	created := &Sim_Node{Contents: []byte{}}
	parent.Children[leaf] = created
	return created, nil
}

// Submit file read that, when it fire, copy node bytes from offset into buffer and report count
// — thus read reflect whatever earlier write store.
func sim_file_read(
	state *Sim, completion *time.Completion, node *Sim_Node, buffer []byte, offset int64,
	timeout time.Duration,
	callback time.Callback,
) {
	if len(buffer) == 0 {
		sim_submit(state, completion, 0, func(completion *time.Completion) {
			sim_deliver(completion, 0, nil, callback)
		})
		return
	}
	latency := sim_latency(state)
	if sim_storage_timeout_first(state, latency, timeout) {
		sim_submit(state, completion, timeout, func(completion *time.Completion) {
			sim_deliver(completion, 0, time.Deadline_Exceeded, callback)
		})
		return
	}
	sim_submit(state, completion, latency, func(completion *time.Completion) {
		count := 0
		if offset < int64(len(node.Contents)) {
			count = copy(buffer, node.Contents[offset:])
		}
		sim_deliver(completion, count, nil, callback)
	})
}

// Submit file write that, when it fire, store buffer into node at offset, grow its contents as
// needed, and report byte count.
func sim_file_write(
	state *Sim, completion *time.Completion, node *Sim_Node, buffer []byte, offset int64,
	timeout time.Duration,
	callback time.Callback,
) {
	if len(buffer) == 0 {
		sim_submit(state, completion, 0, func(completion *time.Completion) {
			sim_deliver(completion, 0, nil, callback)
		})
		return
	}
	latency := sim_latency(state)
	if sim_storage_timeout_first(state, latency, timeout) {
		sim_submit(state, completion, timeout, func(completion *time.Completion) {
			sim_deliver(completion, 0, time.Deadline_Exceeded, callback)
		})
		return
	}
	sim_submit(state, completion, latency, func(completion *time.Completion) {
		sim_node_write(node, buffer, offset)
		sim_deliver(completion, len(buffer), nil, callback)
	})
}

// Store buffer into node contents at offset, growing backing bytes when write extend past
// current end.
func sim_node_write(node *Sim_Node, buffer []byte, offset int64) {
	end_size := offset + int64(len(buffer))
	if end_size > int64(len(node.Contents)) {
		grown := make([]byte, end_size)
		copy(grown, node.Contents)
		node.Contents = grown
	}
	copy(node.Contents[offset:], buffer)
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
	node_budget := slices.SLICE_COUNT_MAXIMUM - 1
	for len(directories) > 0 {
		directory := directories[len(directories)-1]
		directories = directories[:len(directories)-1]
		index := 0
		for node_budget > 0 && prng.Generator_Chance(generator, sim_grow_chance()) {
			name := "e" + decimal_text(index)
			index++
			node_budget--
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

// DECIMAL_RADIX selects base ten for generated entry names.
const DECIMAL_RADIX = 10

func decimal_text(value int) (text string) {
	invariant.Always(value >= 0, "A generated entry index is nonnegative.")
	buffer := [bits.WORD_SIZE]byte{}
	position_count := len(buffer)
	for digit_index := 0; digit_index < len(buffer); digit_index++ {
		position_count--
		buffer[position_count] = byte(value%DECIMAL_RADIX) + '0'
		value /= DECIMAL_RADIX
		if value == 0 {
			return string(buffer[position_count:])
		}
	}
	panic("io: a machine integer exceeds its bit count in decimal digits")
}

// Draw file contents from seed: size Sampled from heavy-tailed distribution, filled with
// seed-drawn bytes.
func sim_generate_bytes(
	generator *prng.Generator, sizes prng.Distribution[uint64],
) (contents []byte) {
	contents = make([]byte, prng.Generator_Sample(generator, sizes))
	for index := range contents {
		contents[index] = byte(prng.Generator_Next(generator))
	}
	return contents
}

// Draw next operation completion delay from seed, thus completion order vary per run yet
// reproduce exact.
func sim_latency(state *Sim) (latency time.Duration) {
	return time.Duration(prng.Generator_Below(&state.Generator, SIM_LATENCY_GRAINS))
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
	generator *prng.Generator, latency time.Duration, timeout time.Duration,
) (timeout_first bool) {
	if latency < timeout {
		return false
	}
	if latency > timeout {
		return true
	}
	return prng.Generator_Below(generator, 2) == 0
}

// Schedule callback to receive next connected synthetic descriptor after drawn latency.
func sim_yield_socket(
	state *Sim, completion *time.Completion, listener File, timeout time.Duration,
	callback time.Callback,
) {
	latency := sim_latency(state)
	if sim_timeout_first(state, latency, timeout) {
		sim_submit(state, completion, timeout, func(completion *time.Completion) {
			sim_deliver(completion, int(File(-1)), time.Deadline_Exceeded, callback)
		})
		return
	}
	sim_submit(state, completion, latency, func(completion *time.Completion) {
		listener_state := state.Sockets[listener]
		if listener_state == nil {
			sim_deliver(
				completion, 0, errors.New("io: socket is not listening"), callback,
			)
			return
		}
		if !listener_state.Listener {
			sim_deliver(
				completion, 0, errors.New("io: socket is not listening"), callback,
			)
			return
		}
		socket := sim_open_socket(state, listener_state.Family, false)
		state.Sockets[socket].Connected = true
		sim_deliver(completion, int(socket), nil, callback)
	})
}

// Report synthetic peer address of live descriptor, simulator getpeername. Empty with no error
// for unknown descriptor.
func sim_peer_address(state *Sim, file File) (address string) {
	socket := state.Sockets[file]
	if socket == nil {
		return ""
	}
	if !socket.Connected {
		return ""
	}
	if socket.Family == FAMILY_IPV6 {
		return "::1"
	}
	return "127.0.0.1"
}

// Store an operation-defined integer and error before the one public callback observes them.
func sim_deliver(
	completion *time.Completion, data int, err error, callback time.Callback,
) {
	completion.Data = data
	completion.Error = err
	callback(completion)
}

// Submit one simulated operation through loop that own order.
func sim_submit(
	state *Sim, completion *time.Completion, latency time.Duration, callback time.Callback,
) {
	// Borrow end where completion retire. Loop own drain and know nothing of descriptors,
	// thus release ride callback loop run — else retired operation still read as borrowing
	// its file, and Close assert.
	time.Timeline_Submit(state.Timeline, completion, latency, func(completed *time.Completion) {
		delete(state.Operation_Files, completion)
		callback(completed)
	})
}

// Stream owns completion policy because only the concrete function knows if an operation is
// immediate, simulated, or kernel-backed.
type Stream func(
	completion *time.Completion, mode Stream_Mode, buffer []byte, offset int64,
	whence Seek_From, callback time.Callback,
)

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
	stream Stream, completion *time.Completion, buffer []byte, callback time.Callback,
) {
	stream_submit(
		stream, completion, STREAM_MODE_READ, buffer, 0, SEEK_FROM_START,
		func(completed *time.Completion) {
			completed.Data, completed.Error = stream_read_checked(
				completed.Data, completed.Error, buffer,
			)
			callback(completed)
		},
	)
}

// Read_At keeps the cursor unchanged while it applies the same result validation as Read.
func Read_At(
	stream Stream, completion *time.Completion, buffer []byte, offset int64,
	callback time.Callback,
) {
	stream_submit(
		stream, completion, STREAM_MODE_READ_AT, buffer, offset, SEEK_FROM_START,
		func(completed *time.Completion) {
			completed.Data, completed.Error = stream_read_checked(
				completed.Data, completed.Error, buffer,
			)
			callback(completed)
		},
	)
}

// Write converts a silent short write into an error before the callback observes it.
func Write(
	stream Stream, completion *time.Completion, buffer []byte, callback time.Callback,
) {
	stream_submit(
		stream, completion, STREAM_MODE_WRITE, buffer, 0, SEEK_FROM_START,
		func(completed *time.Completion) {
			completed.Data, completed.Error = stream_write_checked(
				completed.Data, completed.Error, buffer,
			)
			callback(completed)
		},
	)
}

// Write_At leaves the cursor unchanged while it rejects an invalid procedure count.
func Write_At(
	stream Stream, completion *time.Completion, buffer []byte, offset int64,
	callback time.Callback,
) {
	stream_submit(
		stream, completion, STREAM_MODE_WRITE_AT, buffer, offset, SEEK_FROM_START,
		func(completed *time.Completion) {
			completed.Data, completed.Error = stream_write_checked(
				completed.Data, completed.Error, buffer,
			)
			callback(completed)
		},
	)
}

// Seek reports the new cursor through Completion.Data, so it follows the same lifecycle as a
// byte transfer.
func Seek(
	stream Stream, completion *time.Completion, offset int64, whence Seek_From,
	callback time.Callback,
) {
	stream_submit(
		stream, completion, STREAM_MODE_SEEK, nil, offset, whence, callback,
	)
}

// Size reports the bounded storage size through Completion.Data.
func Size(stream Stream, completion *time.Completion, callback time.Callback) {
	stream_submit(
		stream, completion, STREAM_MODE_SIZE, nil, 0, SEEK_FROM_START, callback,
	)
}

// Flush remains a submitted operation because composition can place buffering behind Stream.
func Flush(stream Stream, completion *time.Completion, callback time.Callback) {
	stream_submit(
		stream, completion, STREAM_MODE_FLUSH, nil, 0, SEEK_FROM_START, callback,
	)
}

// Close retires on the timeline, so teardown can join it before release of owner state.
func Close(stream Stream, completion *time.Completion, callback time.Callback) {
	stream_submit(
		stream, completion, STREAM_MODE_CLOSE, nil, 0, SEEK_FROM_START, callback,
	)
}

// Destroy remains distinct from Close because some stream implementations can own storage.
func Destroy(stream Stream, completion *time.Completion, callback time.Callback) {
	stream_submit(
		stream, completion, STREAM_MODE_DESTROY, nil, 0, SEEK_FROM_START, callback,
	)
}

// Query reports capabilities through Completion.Data because the procedure owns that data.
func Query(stream Stream, completion *time.Completion, callback time.Callback) {
	stream_submit(
		stream, completion, STREAM_MODE_QUERY, nil, 0, SEEK_FROM_START, callback,
	)
}

// All public modes enter here, so nil checks and unsupported-mode behavior cannot diverge.
func stream_submit(
	stream Stream, completion *time.Completion, mode Stream_Mode, buffer []byte,
	offset int64, whence Seek_From, callback time.Callback,
) {
	invariant.Always(completion != nil, "A Stream operation has a completion.")
	invariant.Always(callback != nil, "A Stream operation has a callback.")
	if stream == nil {
		stream_complete(completion, 0, Stream_Empty, callback)
		return
	}
	stream(completion, mode, buffer, offset, whence, callback)
}

// Completion fields hold every scalar result, so Stream needs no second callback type.
func stream_complete(
	completion *time.Completion, data int, err error, callback time.Callback,
) {
	completion.Data = data
	completion.Error = err
	callback(completion)
}

// A read count outside its borrowed buffer is a procedure defect, not caller data.
func stream_read_checked(
	count int, err error, buffer []byte,
) (checked int, checked_err error) {
	if count < 0 {
		return 0, Stream_Negative_Read
	}
	if count > len(buffer) {
		return 0, Stream_Short_Buffer
	}
	return count, err
}

// A write must report every unstored byte because silent truncation loses caller data.
func stream_write_checked(
	count int, err error, buffer []byte,
) (checked int, checked_err error) {
	if count < 0 {
		return 0, Stream_Negative_Write
	}
	if count > len(buffer) {
		return 0, Stream_Invalid_Write
	}
	if err != nil {
		return count, err
	}
	if count < len(buffer) {
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
	invariant.Always(state != nil, "A memory Stream has state.")
	return func(
		completion *time.Completion, mode Stream_Mode, buffer []byte, offset int64,
		whence Seek_From, callback time.Callback,
	) {
		stream_memory_procedure(
			state, completion, mode, buffer, offset, whence, callback,
		)
	}
}

// Memory completes inline because its concrete operation cannot wait on an external endpoint.
func stream_memory_procedure(
	state *Stream_Memory, completion *time.Completion, mode Stream_Mode, buffer []byte,
	offset int64, whence Seek_From, callback time.Callback,
) {
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
	invariant.Always(state != nil, "A discard Stream has state.")
	return func(
		completion *time.Completion, mode Stream_Mode, buffer []byte, _ int64,
		_ Seek_From, callback time.Callback,
	) {
		stream_discard_procedure(state, completion, mode, buffer, callback)
	}
}

// Discard completes inline because it has no external endpoint or scheduled work.
func stream_discard_procedure(
	state *Stream_Discard, completion *time.Completion, mode Stream_Mode, buffer []byte,
	callback time.Callback,
) {
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
}

// Limit_To_Stream keeps the inner Stream completion policy unchanged.
func Limit_To_Stream(state *Stream_Limit) (stream Stream) {
	invariant.Always(state != nil, "A limit Stream has state.")
	return func(
		completion *time.Completion, mode Stream_Mode, buffer []byte, _ int64,
		_ Seek_From, callback time.Callback,
	) {
		stream_limit_procedure(state, completion, mode, buffer, callback)
	}
}

// Limit forwards one submitted operation and adjusts its budget only after Inner retires.
func stream_limit_procedure(
	state *Stream_Limit, completion *time.Completion, mode Stream_Mode, buffer []byte,
	callback time.Callback,
) {
	if mode == STREAM_MODE_CLOSE {
		Close(state.Inner, completion, callback)
		return
	}
	if mode == STREAM_MODE_DESTROY {
		Destroy(state.Inner, completion, callback)
		return
	}
	if mode == STREAM_MODE_FLUSH {
		Flush(state.Inner, completion, callback)
		return
	}
	if mode == STREAM_MODE_QUERY {
		Query(state.Inner, completion, func(completed *time.Completion) {
			completed.Data = int(Stream_Mode_Set(completed.Data) & STREAM_LIMIT_MODES)
			callback(completed)
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
	state *Stream_Limit, completion *time.Completion, buffer []byte,
	callback time.Callback,
) {
	if state.Budget <= 0 {
		stream_complete(completion, 0, Stream_EOF, callback)
		return
	}
	allowed_count := len(buffer)
	if state.Budget < int64(allowed_count) {
		allowed_count = int(state.Budget)
	}
	Read(state.Inner, completion, buffer[:allowed_count], func(completed *time.Completion) {
		state.Budget = state.Budget - int64(completed.Data)
		callback(completed)
	})
}

// A limit reports the part it withheld even when Inner stored every byte it received.
func stream_limit_write(
	state *Stream_Limit, completion *time.Completion, buffer []byte,
	callback time.Callback,
) {
	if state.Budget <= 0 {
		stream_complete(completion, 0, Stream_Short_Write, callback)
		return
	}
	allowed_count := len(buffer)
	if state.Budget < int64(allowed_count) {
		allowed_count = int(state.Budget)
	}
	Write(state.Inner, completion, buffer[:allowed_count], func(completed *time.Completion) {
		state.Budget = state.Budget - int64(completed.Data)
		if completed.Error == nil {
			if allowed_count < len(buffer) {
				completed.Error = Stream_Short_Write
			}
		}
		callback(completed)
	})
}

// Stream_Count changes only its tally, so Inner remains the authority for every mode result.
type Stream_Count struct {
	// Inner performs the operation that Count observes.
	Inner Stream
	// Tally records every transferred byte without constraining the transfer.
	Tally int64
}

// Count_To_Stream keeps the inner Stream completion policy unchanged.
func Count_To_Stream(state *Stream_Count) (stream Stream) {
	invariant.Always(state != nil, "A count Stream has state.")
	return func(
		completion *time.Completion, mode Stream_Mode, buffer []byte, offset int64,
		whence Seek_From, callback time.Callback,
	) {
		stream_count_procedure(
			state, completion, mode, buffer, offset, whence, callback,
		)
	}
}

// Count updates its tally inside Inner's callback, so callers cannot observe a result before
// its accounting.
func stream_count_procedure(
	state *Stream_Count, completion *time.Completion, mode Stream_Mode, buffer []byte,
	offset int64, whence Seek_From, callback time.Callback,
) {
	if mode == STREAM_MODE_CLOSE {
		Close(state.Inner, completion, callback)
		return
	}
	if mode == STREAM_MODE_DESTROY {
		Destroy(state.Inner, completion, callback)
		return
	}
	if mode == STREAM_MODE_FLUSH {
		Flush(state.Inner, completion, callback)
		return
	}
	if mode == STREAM_MODE_SEEK {
		Seek(state.Inner, completion, offset, whence, callback)
		return
	}
	if mode == STREAM_MODE_SIZE {
		Size(state.Inner, completion, callback)
		return
	}
	if mode == STREAM_MODE_QUERY {
		Query(state.Inner, completion, callback)
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
	state *Stream_Count, completion *time.Completion, buffer []byte,
	callback time.Callback,
) {
	Read(state.Inner, completion, buffer, stream_count_callback(state, callback))
}

// Count records a positioned read before it publishes Inner's completion.
func stream_count_read_at(
	state *Stream_Count, completion *time.Completion, buffer []byte, offset int64,
	callback time.Callback,
) {
	Read_At(
		state.Inner, completion, buffer, offset, stream_count_callback(state, callback),
	)
}

// Count records a sequential write before it publishes Inner's completion.
func stream_count_write(
	state *Stream_Count, completion *time.Completion, buffer []byte,
	callback time.Callback,
) {
	Write(state.Inner, completion, buffer, stream_count_callback(state, callback))
}

// Count records a positioned write before it publishes Inner's completion.
func stream_count_write_at(
	state *Stream_Count, completion *time.Completion, buffer []byte, offset int64,
	callback time.Callback,
) {
	Write_At(
		state.Inner, completion, buffer, offset, stream_count_callback(state, callback),
	)
}

// One wrapper keeps all four transfer modes on the same accounting rule.
func stream_count_callback(
	state *Stream_Count, callback time.Callback,
) (counted time.Callback) {
	return func(completed *time.Completion) {
		state.Tally = state.Tally + int64(completed.Data)
		callback(completed)
	}
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
}

// Tee_To_Stream sequences the two Stream policies instead of selecting a third policy.
func Tee_To_Stream(state *Stream_Tee) (stream Stream) {
	invariant.Always(state != nil, "A tee Stream has state.")
	return func(
		completion *time.Completion, mode Stream_Mode, buffer []byte, _ int64,
		_ Seek_From, callback time.Callback,
	) {
		stream_tee_procedure(state, completion, mode, buffer, callback)
	}
}

// Tee submits Second only after First retires, so one caller-owned completion remains valid.
func stream_tee_procedure(
	state *Stream_Tee, completion *time.Completion, mode Stream_Mode, buffer []byte,
	callback time.Callback,
) {
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
	state *Stream_Tee, completion *time.Completion, mode Stream_Mode,
	callback time.Callback,
) {
	stream_submit(
		state.First, completion, mode, nil, 0, SEEK_FROM_START,
		func(first *time.Completion) {
			first_err := first.Error
			stream_submit(
				state.Second, first, mode, nil, 0, SEEK_FROM_START,
				func(second *time.Completion) {
					if first_err != nil {
						second.Error = first_err
					}
					callback(second)
				},
			)
		},
	)
}

// Tee reports the smaller count because that count exposes the destination that lost more
// bytes.
func stream_tee_write(
	state *Stream_Tee, completion *time.Completion, buffer []byte,
	callback time.Callback,
) {
	Write(state.First, completion, buffer, func(first *time.Completion) {
		first_count := first.Data
		first_err := first.Error
		Write(state.Second, first, buffer, func(second *time.Completion) {
			if first_count < second.Data {
				second.Data = first_count
			}
			if first_err != nil {
				second.Error = first_err
			}
			callback(second)
		})
	})
}
