// Package nbio is dependency-injected completion surface of repository.
//
// No generic Cancel: owner use Shutdown, join every submitted operation, then submit Close.
// Repository-specific effect is marked extension and must retire through same completed queue.
package nbio

import (
	"errors"
	"unsafe"

	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/random/prng"
	"local/james-orcales/shared/simulation/time"
)

// IO is injected async IO submit surface. Code submit operation with
// time.Completion and callback, then react to completion. Code never drive loop — that is
// time.Driver job — thus holder submit IO, but cannot advance time.
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
	// State stays caller-owned because captured backend state allocates every surface closure.
	State unsafe.Pointer
	// Network is every transfer whose endpoint is socket.
	Network Network
	// Storage is every transfer whose endpoint is file or directory.
	Storage Storage
	// Close release descriptor of file. Callback fire once it is closed. It sit here, not on
	// one half: close(2) name descriptor, and both half hand descriptors out.
	Close_Procedure func(
		state unsafe.Pointer, completion *time.Completion, file File,
		callback time.Callback,
	)
	// Deinit assert every descriptor run open is closed. Surface own leak check, thus leaked
	// descriptor fail run. Caller need no census it must remember to read.
	Deinit_Procedure func(state unsafe.Pointer)
}

// IO_Close keeps backend state explicit because bound method state would allocate.
func IO_Close(
	loop IO,
	completion *time.Completion, file File, callback time.Callback,
) {
	loop.Close_Procedure(loop.State, completion, file, callback)
}

// IO_Deinit keeps leak validation on backend that owns descriptor records.
func IO_Deinit(loop IO) {
	loop.Deinit_Procedure(loop.State)
}

// Network is half of IO: every transfer whose endpoint is socket, plus socket lifecycle that
// open one. Holder that only speak to socket take this and cannot reach file.
//
// TLS is not member. No accept-secure syscall, no connect-secure syscall, thus secure_transport
// compose these raw operations under its own record layer.
type Network struct {
	// State remains caller-owned while static procedures borrow it.
	State unsafe.Pointer
	// Socket_TCP make one configured non-blocking close-on-exec TCP socket.
	Socket_TCP_Procedure func(
		state unsafe.Pointer, family Address_Family, options TCP_Options,
	) (socket File, err error)
	// Socket_UDP make one configured non-blocking close-on-exec UDP socket.
	Socket_UDP_Procedure func(
		state unsafe.Pointer, family Address_Family, options UDP_Options,
	) (socket File, err error)
	// Bind enable address reuse and give caller-owned socket its local address.
	Bind_Procedure func(state unsafe.Pointer, socket File, address Address) (err error)
	// Listen_Socket mark bound socket accepting, with backlog as queue depth.
	Listen_Socket_Procedure func(
		state unsafe.Pointer, socket File, backlog uint32,
	) (err error)
	// Get_Socket_Name report socket own address. That is how caller learn port kernel chose
	// for port zero.
	Get_Socket_Name_Procedure func(
		state unsafe.Pointer, socket File,
	) (address Address, err error)
	// Accept yield one inbound connection on listener before timeout, or report
	// Deadline_Exceeded. Timeout is finite deliberately: it keep every repository
	// submission bounded.
	Accept_Procedure func(
		state unsafe.Pointer, completion *time.Completion, listener File,
		timeout time.Duration,
		callback time.Callback,
	)
	// Connect borrow caller-owned socket until one kernel result or timeout. Callback report
	// outcome only. Backend never make, transfer, or close descriptor.
	Connect_Procedure func(
		state unsafe.Pointer, completion *time.Completion, socket File, address Address,
		timeout time.Duration,
		callback time.Callback,
	)
	// Receive read up to len(buffer) bytes before timeout. Callback report byte count once data
	// arrive, or zero with Deadline_Exceeded after the kernel request retire.
	Receive_Procedure func(
		state unsafe.Pointer, completion *time.Completion, socket File, buffer []byte,
		timeout time.Duration,
		callback time.Callback,
	)
	// Send write buffer before timeout. Callback report byte count once kernel accept it, or
	// zero with Deadline_Exceeded after the kernel request retire.
	Send_Procedure func(
		state unsafe.Pointer, completion *time.Completion, socket File, buffer []byte,
		timeout time.Duration,
		callback time.Callback,
	)
	// Shutdown synchronously disable one or both connected-socket direction. It neither own nor
	// close socket.
	Shutdown_Procedure func(state unsafe.Pointer, socket File, how Shutdown_How) (err error)
	// Peer_Address return remote IP address of connected socket, synchronously — getpeername
	// has no completion. It is source a control-plane connection is gated on.
	Peer_Address_Procedure func(
		state unsafe.Pointer, file File,
	) (address Address, err error)
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
	network Network, socket File,
) (address Address, err error) {
	return network.Get_Socket_Name_Procedure(network.State, socket)
}

// Network_Accept preserves callback-last submit shape while state remains explicit.
func Network_Accept(
	network Network,
	completion *time.Completion, listener File, timeout time.Duration,
	callback time.Callback,
) {
	network.Accept_Procedure(network.State, completion, listener, timeout, callback)
}

// Network_Connect preserves callback-last submit shape while state remains explicit.
func Network_Connect(
	network Network,
	completion *time.Completion, socket File, address Address, timeout time.Duration,
	callback time.Callback,
) {
	network.Connect_Procedure(network.State, completion, socket, address, timeout, callback)
}

// Network_Receive preserves callback-last submit shape while state remains explicit.
func Network_Receive(
	network Network,
	completion *time.Completion, socket File, buffer []byte, timeout time.Duration,
	callback time.Callback,
) {
	network.Receive_Procedure(network.State, completion, socket, buffer, timeout, callback)
}

// Network_Send preserves callback-last submit shape while state remains explicit.
func Network_Send(
	network Network,
	completion *time.Completion, socket File, buffer []byte, timeout time.Duration,
	callback time.Callback,
) {
	network.Send_Procedure(network.State, completion, socket, buffer, timeout, callback)
}

// Network_Shutdown passes caller-owned state to static direction control.
func Network_Shutdown(network Network, socket File, how Shutdown_How) (err error) {
	return network.Shutdown_Procedure(network.State, socket, how)
}

// Network_Peer_Address passes caller-owned state to static peer reader.
func Network_Peer_Address(network Network, file File) (address Address, err error) {
	return network.Peer_Address_Procedure(network.State, file)
}

// Storage is other half of IO: every transfer whose endpoint is file or directory, plus path
// primitives that name one. Holder that only read and write file take this and cannot reach
// socket.
type Storage struct {
	// State remains caller-owned while static procedures borrow it.
	State unsafe.Pointer
	// Read read len(buffer) bytes from file at offset before timeout. Timeout does not work on
	// Darwin because its current file path has no kernel timeout. Darwin completes the
	// operation.
	Read_Procedure func(
		state unsafe.Pointer, completion *time.Completion, file File, buffer []byte,
		offset int64,
		timeout time.Duration,
		callback time.Callback,
	)
	// Write write buffer to file at offset before timeout. Timeout does not work on Darwin
	// because its current file path has no kernel timeout. Darwin completes the operation.
	Write_Procedure func(
		state unsafe.Pointer, completion *time.Completion, file File, buffer []byte,
		offset int64,
		timeout time.Duration,
		callback time.Callback,
	)
	// Fsync synchronize file before timeout. Timeout does not work on Darwin because its
	// current file path has no kernel timeout. Darwin completes the operation.
	Fsync_Procedure func(
		state unsafe.Pointer, completion *time.Completion, file File, timeout time.Duration,
		callback time.Callback,
	)
	// Open_At asynchronously open file_path relative to directory and force close-on-exec.
	Open_At_Procedure func(
		state unsafe.Pointer, completion *time.Completion, directory File, file_path string,
		options Open_At_Options, callback time.Callback,
	)
	// Mkdir_At asynchronously make one directory named by file_path relative to directory. It
	// is mkdirat primitive, not parent-creating mkdir: parent must exist, and existing path
	// report operating system error rather than converge. Make_Directory compose this
	// primitive above surface, thus both backend run same composition.
	Mkdir_At_Procedure func(
		state unsafe.Pointer, completion *time.Completion, directory File, file_path string,
		mode uint32,
		callback time.Callback,
	)
	// Get_Directory_Entries read one pass of directory raw entries into buffer and return
	// children it name, each with whether it is itself directory. Zero count report end.
	// Dirent layout is per-platform, thus parse and kind stay in backend, and only pass loop
	// and descriptor lifetime compose above.
	Get_Directory_Entries_Procedure func(
		state unsafe.Pointer, completion *time.Completion, directory File, buffer []byte,
		entries []Directory_Entry, callback time.Callback,
	)
	// Status report whether path exist, whether it is directory, and its byte size,
	// synchronously. Absent path is Exists false with nil error, thus caller branch on status,
	// not on error.
	Status_Procedure func(
		state unsafe.Pointer, path string,
	) (status File_Status, err error)
}

// Storage_Read preserves callback-last submit shape while state remains explicit.
func Storage_Read(
	storage Storage,
	completion *time.Completion, file File, buffer []byte, offset int64,
	timeout time.Duration, callback time.Callback,
) {
	storage.Read_Procedure(
		storage.State, completion, file, buffer, offset, timeout, callback,
	)
}

// Storage_Write preserves callback-last submit shape while state remains explicit.
func Storage_Write(
	storage Storage,
	completion *time.Completion, file File, buffer []byte, offset int64,
	timeout time.Duration, callback time.Callback,
) {
	storage.Write_Procedure(
		storage.State, completion, file, buffer, offset, timeout, callback,
	)
}

// Storage_Fsync preserves callback-last submit shape while state remains explicit.
func Storage_Fsync(
	storage Storage,
	completion *time.Completion, file File, timeout time.Duration,
	callback time.Callback,
) {
	storage.Fsync_Procedure(storage.State, completion, file, timeout, callback)
}

// Storage_Open_At preserves callback-last submit shape while state remains explicit.
func Storage_Open_At(
	storage Storage,
	completion *time.Completion, directory File, file_path string,
	options Open_At_Options, callback time.Callback,
) {
	storage.Open_At_Procedure(
		storage.State, completion, directory, file_path, options, callback,
	)
}

// Storage_Mkdir_At preserves callback-last submit shape while state remains explicit.
func Storage_Mkdir_At(
	storage Storage,
	completion *time.Completion, directory File, file_path string, mode uint32,
	callback time.Callback,
) {
	storage.Mkdir_At_Procedure(
		storage.State, completion, directory, file_path, mode, callback,
	)
}

// Storage_Get_Directory_Entries fills caller-owned entry storage before callback.
func Storage_Get_Directory_Entries(
	storage Storage,
	completion *time.Completion, directory File, buffer []byte, entries []Directory_Entry,
	callback time.Callback,
) {
	storage.Get_Directory_Entries_Procedure(
		storage.State, completion, directory, buffer, entries, callback,
	)
}

// Storage_Status passes caller-owned state to static path reader.
func Storage_Status(storage Storage, path string) (status File_Status, err error) {
	return storage.Status_Procedure(storage.State, path)
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
		return Address{}, address_port_outside_uint16
	}
	if port > 65535 {
		return Address{}, address_port_outside_uint16
	}
	if len(host) > ADDRESS_TEXT_BYTES_MAXIMUM {
		return Address{}, address_host_too_large
	}
	if text_contains(host, ":") {
		sextets, found := address_parse_ipv6(host)
		if !found {
			return Address{}, address_host_not_literal
		}
		return Address_IPV6(sextets, uint16(port)), nil
	}
	quad, found := address_parse_ipv4(host)
	if !found {
		return Address{}, address_host_not_literal
	}
	return Address_IPV4(quad, uint16(port)), nil
}

var address_port_outside_uint16 = errors.New("io: port is outside uint16")
var address_host_too_large = errors.New("io: host is too large")
var address_host_not_literal = errors.New("io: host is not an IP literal")

// Parse dotted-quad into four octets.
func address_parse_ipv4(host string) (quad [IPV4_ADDRESS_BYTES]byte, found bool) {
	parts := [IPV4_ADDRESS_BYTES]string{}
	part_count, split := text_split(host, '.', parts[:])
	if !split {
		return [IPV4_ADDRESS_BYTES]byte{}, false
	}
	if part_count != IPV4_ADDRESS_BYTES {
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
	values, value_count, valid := address_parse_ipv6_parts(host)
	if !valid {
		return [IPV6_ADDRESS_BYTES]byte{}, false
	}
	if value_count != IPV6_ADDRESS_BYTES {
		return [IPV6_ADDRESS_BYTES]byte{}, false
	}
	return values, true
}

// Join both halves of a compressed literal, with zeros between them. Run must stand for at least
// one group, thus the two halves together leave two bytes free.
func address_join_ipv6(head string, tail string) (sextets [IPV6_ADDRESS_BYTES]byte, found bool) {
	if text_contains(tail, "::") {
		return [IPV6_ADDRESS_BYTES]byte{}, false
	}
	front, front_count, front_valid := address_parse_ipv6_parts(head)
	if !front_valid {
		return [IPV6_ADDRESS_BYTES]byte{}, false
	}
	back, back_count, back_valid := address_parse_ipv6_parts(tail)
	if !back_valid {
		return [IPV6_ADDRESS_BYTES]byte{}, false
	}
	if front_count+back_count > IPV6_ADDRESS_BYTES-2 {
		return [IPV6_ADDRESS_BYTES]byte{}, false
	}
	copy(sextets[:front_count], front[:front_count])
	copy(sextets[IPV6_ADDRESS_BYTES-back_count:], back[:back_count])
	return sextets, true
}

// Parse one colon-separated group run into its bytes. Empty text yield no bytes. Trailing
// dotted-quad contribute four bytes, which is how embedded IPv4 reach the low 32 bits.
func address_parse_ipv6_parts(
	text string,
) (values [IPV6_ADDRESS_BYTES]byte, count int, valid bool) {
	if text == "" {
		return values, 0, true
	}
	parts := [ADDRESS_IPV6_PARTS_MAX]string{}
	part_count, split := text_split(text, ':', parts[:])
	if !split {
		return values, 0, false
	}
	for index := 0; index < part_count; index++ {
		part := parts[index]
		if index == part_count-1 {
			if text_contains(part, ".") {
				quad, quad_valid := address_parse_ipv4(part)
				if !quad_valid {
					return values, 0, false
				}
				if count+IPV4_ADDRESS_BYTES > len(values) {
					return values, 0, false
				}
				copy(values[count:], quad[:])
				count += IPV4_ADDRESS_BYTES
				return values, count, true
			}
		}
		high, low, group_valid := address_parse_group(part)
		if !group_valid {
			return values, 0, false
		}
		if count+2 > len(values) {
			return values, 0, false
		}
		values[count] = high
		values[count+1] = low
		count += 2
	}
	return values, count, true
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
// not API to cancel operation: this surface deliberately has no generic Cancel.
var Canceled = errors.New("io: operation canceled")

// Path_Exists is portable Mkdir_At result for path already present. Make_Directory treat it as
// convergence, thus repeated create is not error.
var Path_Exists = errors.New("io: file exists")

// Number of virtual grains one simulated operation may take to complete, drawn from seed, thus
// completion order vary per run and still reproduce.
const SIM_LATENCY_GRAINS = 8

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

// Sim_Node is caller-owned storage for one generated or created filesystem node.
type Sim_Node struct {
	// Used separates free caller slots from seeded filesystem nodes.
	Used bool
	// Directory keeps node kind inline so lookup needs no interface value.
	Directory bool
	// Parent names another caller slot so tree edges need no pointer allocation.
	Parent int
	// Name keeps component bytes inline so generated paths need no string allocation.
	Name [SIM_PATH_COMPONENT_BYTES_MAXIMUM]byte
	// Name_Count bounds valid bytes without making a slice escape.
	Name_Count int
	// Contents keeps file bytes inline so writes cannot grow heap storage.
	Contents [SIM_FILE_BYTES_MAXIMUM]byte
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
	// Kind lets one static callback retire every operation shape.
	Kind Sim_Operation_Kind
	// State restores backend ownership without a captured callback.
	State *Sim
	// File identifies descriptor whose borrow must retire.
	File File
	// Node preserves file target even while work waits on timeline.
	Node int
	// Callback remains explicit because bound callback state allocates.
	Callback time.Callback
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
	Result unsafe.Pointer
	// Mask preserves platform stat selection until callback.
	Mask uint32
	// Borrowed marks descriptor count that retirement must release.
	Borrowed bool
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

// Sim_Memory states each independent bounded simulator resource without scripting outcomes.
type Sim_Memory struct {
	// Nodes bounds generated and runtime-created filesystem state.
	Nodes []Sim_Node
	// Descriptors bounds simultaneous caller ownership.
	Descriptors []Sim_Descriptor
	// Operations bounds simultaneous submitted work.
	Operations []Sim_Operation
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
	// Nodes borrows caller capacity for complete simulator lifetime.
	Nodes []Sim_Node
	// Descriptors borrows caller capacity for complete simulator lifetime.
	Descriptors []Sim_Descriptor
	// Operations borrows caller capacity for complete simulator lifetime.
	Operations []Sim_Operation
}

// New_Simulated_IO return deterministic submit surface seeded by seed. Caller memory selects
// capacity only, thus seed is only outcome input and run reproduce exact. Sim read no clock:
// seed draw every latency, and pump own now. Driver stay in harness: program under test hold
// loop, NEVER pump (see time.Driver banner).
func New_Simulated_IO(
	state *Sim, seed uint64, pump time.Timeline, memory Sim_Memory,
) (loop IO) {
	invariant.Always(state != nil, "A simulated IO backend has caller-owned state.")
	invariant.Always(len(memory.Nodes) > 0, "A simulated IO backend has node capacity.")
	invariant.Always(len(memory.Descriptors) > 0,
		"A simulated IO backend has descriptor capacity.")
	invariant.Always(len(memory.Operations) > 0,
		"A simulated IO backend has operation capacity.")
	for index := range memory.Nodes {
		memory.Nodes[index] = Sim_Node{}
	}
	for index := range memory.Descriptors {
		memory.Descriptors[index] = Sim_Descriptor{}
	}
	for index := range memory.Operations {
		memory.Operations[index] = Sim_Operation{}
	}
	root_generator := prng.New(seed)
	state.Timeline = pump
	state.Generator = prng.Generator_Split(&root_generator)
	state.Timeout_Order_Generator = prng.Generator_Split(&root_generator)
	state.Storage_Timeout_Order_Generator = prng.Generator_Split(&root_generator)
	state.Next_File = 0
	state.Nodes = memory.Nodes
	state.Descriptors = memory.Descriptors
	state.Operations = memory.Operations
	state.Nodes[0] = Sim_Node{Used: true, Directory: true, Parent: -1}
	sim_generate(state)
	loop.State = unsafe.Pointer(state)
	sim_wire_network(state, &loop.Network)
	sim_wire_storage(state, &loop.Storage)
	sim_wire_platform(state, &loop)
	loop.Close_Procedure = sim_close_procedure
	loop.Deinit_Procedure = sim_deinit_procedure
	return loop
}

func sim_close_procedure(
	state_pointer unsafe.Pointer, completion *time.Completion, file File,
	callback time.Callback,
) {
	state := (*Sim)(state_pointer)
	sim_assert_file_drained(state, file)
	operation := sim_operation_acquire(state, completion, SIM_OPERATION_KIND_CLOSE, callback)
	operation.File = file
	sim_operation_submit(operation, completion, sim_latency(state))
}

func sim_deinit_procedure(state_pointer unsafe.Pointer) {
	state := (*Sim)(state_pointer)
	open := false
	for index := range state.Descriptors {
		if state.Descriptors[index].Used {
			open = true
		}
	}
	invariant.Always(!open, "Every descriptor a simulated run opened is closed before Deinit.")
}

// Wire whole socket half of simulator onto network.
func sim_wire_network(state *Sim, network *Network) {
	network.State = unsafe.Pointer(state)
	sim_wire_socket_lifecycle(network)
	sim_wire_socket_bytes(network)
}

// Wire whole file half of simulator onto storage.
func sim_wire_storage(state *Sim, storage *Storage) {
	storage.State = unsafe.Pointer(state)
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
	state_pointer unsafe.Pointer, family Address_Family, options TCP_Options,
) (socket File, err error) {
	if !tcp_options_valid(options) {
		return File(-1), sim_invalid_socket_options
	}
	return sim_open_socket((*Sim)(state_pointer), family, false)
}

func sim_socket_udp(
	state_pointer unsafe.Pointer, family Address_Family, options UDP_Options,
) (socket File, err error) {
	if !udp_options_valid(options) {
		return File(-1), sim_invalid_socket_options
	}
	return sim_open_socket((*Sim)(state_pointer), family, true)
}

func sim_bind(
	state_pointer unsafe.Pointer, socket File, address Address,
) (err error) {
	descriptor := sim_descriptor_find((*Sim)(state_pointer), socket)
	if descriptor == nil {
		return sim_bind_socket_closed
	}
	if !descriptor.Socket {
		return sim_bind_socket_closed
	}
	bound := address
	if bound.Port == 0 {
		bound.Port = uint16(socket)
	}
	descriptor.Socket_State.Address = bound
	return nil
}

func sim_listen_socket(
	state_pointer unsafe.Pointer, socket File, backlog uint32,
) (err error) {
	return sim_listen((*Sim)(state_pointer), socket, backlog)
}

func sim_get_socket_name(
	state_pointer unsafe.Pointer, socket File,
) (address Address, err error) {
	descriptor := sim_descriptor_find((*Sim)(state_pointer), socket)
	if descriptor == nil {
		return Address{}, sim_socket_name_closed
	}
	if !descriptor.Socket {
		return Address{}, sim_socket_name_closed
	}
	return descriptor.Socket_State.Address, nil
}

func sim_shutdown_socket(
	state_pointer unsafe.Pointer, socket File, how Shutdown_How,
) (err error) {
	return sim_shutdown((*Sim)(state_pointer), socket, how)
}

func sim_peer_address_procedure(
	state_pointer unsafe.Pointer, file File,
) (address Address, err error) {
	return sim_peer_address((*Sim)(state_pointer), file), nil
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
var sim_node_capacity_exceeded = errors.New("io: node capacity exceeded")
var sim_path_component_too_large = errors.New("io: path component too large")
var sim_file_capacity_exceeded = errors.New("io: file capacity exceeded")
var sim_file_offset_invalid = errors.New("io: file offset invalid")

func sim_accept_procedure(
	state_pointer unsafe.Pointer, completion *time.Completion, listener File,
	timeout time.Duration, callback time.Callback,
) {
	state := (*Sim)(state_pointer)
	invariant.Always(timeout > 0, "An accept timeout is positive and finite.")
	latency := sim_latency(state)
	operation := sim_operation_acquire(state, completion, SIM_OPERATION_KIND_ACCEPT, callback)
	operation.File = listener
	sim_operation_borrow(operation)
	if sim_timeout_first(state, latency, timeout) {
		operation.Operation_Err = time.Deadline_Exceeded
		operation.Data = int(File(-1))
		sim_operation_submit(operation, completion, timeout)
		return
	}
	sim_operation_submit(operation, completion, latency)
}

func sim_connect_procedure(
	state_pointer unsafe.Pointer, completion *time.Completion, socket File, address Address,
	timeout time.Duration, callback time.Callback,
) {
	state := (*Sim)(state_pointer)
	invariant.Always(timeout > 0, "A connect timeout is positive and finite.")
	operation := sim_operation_acquire(state, completion, SIM_OPERATION_KIND_CONNECT, callback)
	operation.File = socket
	operation.Address = address
	sim_operation_borrow(operation)
	if prng.Generator_Below(&state.Generator, 4) == 0 {
		operation.Operation_Err = Connection_Refused
	}
	latency := sim_latency(state)
	if sim_timeout_first(state, latency, timeout) {
		operation.Operation_Err = time.Deadline_Exceeded
		sim_operation_submit(operation, completion, timeout)
		return
	}
	sim_operation_submit(operation, completion, latency)
}

func sim_receive_procedure(
	state_pointer unsafe.Pointer, completion *time.Completion, socket File, buffer []byte,
	timeout time.Duration, callback time.Callback,
) {
	state := (*Sim)(state_pointer)
	invariant.Always(timeout > 0, "A receive timeout is positive and finite.")
	operation := sim_operation_acquire(state, completion, SIM_OPERATION_KIND_RECEIVE, callback)
	operation.File = socket
	operation.Buffer = buffer
	sim_operation_borrow(operation)
	latency := sim_latency(state)
	if sim_timeout_first(state, latency, timeout) {
		operation.Operation_Err = time.Deadline_Exceeded
		sim_operation_submit(operation, completion, timeout)
		return
	}
	sim_operation_submit(operation, completion, latency)
}

func sim_send_procedure(
	state_pointer unsafe.Pointer, completion *time.Completion, socket File, buffer []byte,
	timeout time.Duration, callback time.Callback,
) {
	state := (*Sim)(state_pointer)
	invariant.Always(timeout > 0, "A send timeout is positive and finite.")
	operation := sim_operation_acquire(state, completion, SIM_OPERATION_KIND_SEND, callback)
	operation.File = socket
	operation.Buffer = buffer
	sim_operation_borrow(operation)
	latency := sim_latency(state)
	if sim_timeout_first(state, latency, timeout) {
		operation.Operation_Err = time.Deadline_Exceeded
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
	state_pointer unsafe.Pointer, completion *time.Completion, directory File,
	file_path string, options Open_At_Options, callback time.Callback,
) {
	state := (*Sim)(state_pointer)
	invariant.Always(options.Flags&^OPEN_AT_NO_FOLLOW == 0,
		"Open_At options contain only known flags.")
	operation := sim_operation_acquire(state, completion, SIM_OPERATION_KIND_OPEN_AT, callback)
	operation.Directory = directory
	operation.File_Path = file_path
	operation.Open_Options = options
	sim_operation_submit(operation, completion, sim_latency(state))
}

func sim_mkdir_at_procedure(
	state_pointer unsafe.Pointer, completion *time.Completion, directory File,
	file_path string, _ uint32, callback time.Callback,
) {
	state := (*Sim)(state_pointer)
	operation := sim_operation_acquire(state, completion, SIM_OPERATION_KIND_MKDIR_AT, callback)
	operation.Directory = directory
	operation.File_Path = file_path
	sim_operation_submit(operation, completion, sim_latency(state))
}

// Wire file transfers to caller-owned operation records retained through timeout races.
func sim_wire_file_bytes(storage *Storage) {
	storage.Read_Procedure = sim_storage_read
	storage.Write_Procedure = sim_storage_write
	storage.Fsync_Procedure = sim_storage_fsync
}

func sim_storage_read(
	state_pointer unsafe.Pointer, completion *time.Completion, file File, buffer []byte,
	offset int64, timeout time.Duration, callback time.Callback,
) {
	state := (*Sim)(state_pointer)
	invariant.Always(timeout > 0, "A storage read timeout is positive and finite.")
	descriptor := sim_descriptor_find(state, file)
	invariant.Always(descriptor != nil, "A Storage read names an open descriptor.")
	invariant.Always(!descriptor.Socket, "A Storage read names an open file.")
	operation := sim_operation_acquire(state, completion, SIM_OPERATION_KIND_READ, callback)
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
		operation.Operation_Err = time.Deadline_Exceeded
		sim_operation_submit(operation, completion, timeout)
		return
	}
	sim_operation_submit(operation, completion, latency)
}

func sim_storage_write(
	state_pointer unsafe.Pointer, completion *time.Completion, file File, buffer []byte,
	offset int64, timeout time.Duration, callback time.Callback,
) {
	state := (*Sim)(state_pointer)
	invariant.Always(timeout > 0, "A storage write timeout is positive and finite.")
	descriptor := sim_descriptor_find(state, file)
	invariant.Always(descriptor != nil, "A Storage write names an open descriptor.")
	invariant.Always(!descriptor.Socket, "A Storage write names an open file.")
	operation := sim_operation_acquire(state, completion, SIM_OPERATION_KIND_WRITE, callback)
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
		operation.Operation_Err = time.Deadline_Exceeded
		sim_operation_submit(operation, completion, timeout)
		return
	}
	sim_operation_submit(operation, completion, latency)
}

func sim_storage_fsync(
	state_pointer unsafe.Pointer, completion *time.Completion, file File,
	timeout time.Duration, callback time.Callback,
) {
	state := (*Sim)(state_pointer)
	invariant.Always(timeout > 0, "A storage fsync timeout is positive and finite.")
	descriptor := sim_descriptor_find(state, file)
	invariant.Always(descriptor != nil, "A Storage fsync names an open descriptor.")
	invariant.Always(!descriptor.Socket, "A Storage fsync names an open file.")
	operation := sim_operation_acquire(state, completion, SIM_OPERATION_KIND_FSYNC, callback)
	operation.File = file
	sim_operation_borrow(operation)
	latency := sim_latency(state)
	if sim_storage_timeout_first(state, latency, timeout) {
		operation.Operation_Err = time.Deadline_Exceeded
		sim_operation_submit(operation, completion, timeout)
		return
	}
	sim_operation_submit(operation, completion, latency)
}

// Directory result occupies caller slots; Completion.Data states populated count.
func sim_wire_directory(storage *Storage) {
	storage.Get_Directory_Entries_Procedure = sim_get_directory_entries
	storage.Status_Procedure = sim_status_procedure
}

func sim_get_directory_entries(
	state_pointer unsafe.Pointer, completion *time.Completion, directory File, _ []byte,
	entries []Directory_Entry, callback time.Callback,
) {
	state := (*Sim)(state_pointer)
	operation := sim_operation_acquire(
		state, completion, SIM_OPERATION_KIND_DIRECTORY, callback,
	)
	operation.File = directory
	operation.Entries = entries
	sim_operation_borrow(operation)
	sim_operation_submit(operation, completion, sim_latency(state))
}

func sim_status_procedure(
	state_pointer unsafe.Pointer, path string,
) (status File_Status, err error) {
	return sim_status((*Sim)(state_pointer), path), nil
}

// Close cannot release descriptor while any submitted operation still holds it.
func sim_assert_file_drained(state *Sim, file File) {
	descriptor := sim_descriptor_find(state, file)
	invariant.Always(descriptor != nil, "A Close names an open descriptor.")
	invariant.Always(descriptor.Borrowed == 0,
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
		state.Descriptors[index] = Sim_Descriptor{
			Used: true, File: state.Next_File, Node: node, Socket: socket,
			Socket_State: socket_state,
		}
		return &state.Descriptors[index], nil
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
		*descriptor = Sim_Descriptor{}
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

// Resolve path by scanning input views directly; component collection would allocate.
func sim_resolve(state *Sim, path string) (node_index int, found bool) {
	invariant.Always(len(path) <= SIM_PATH_TEXT_BYTES_MAXIMUM,
		"A simulated path stays inside the repository text budget.")
	node_index = 0
	start := 0
	for position := 0; position <= len(path); position++ {
		if position < len(path) {
			if path[position] != '/' {
				continue
			}
		}
		if position > start {
			node := &state.Nodes[node_index]
			if !node.Directory {
				return 0, false
			}
			child, child_found := sim_node_find_child(
				state, node_index, path[start:position],
			)
			if !child_found {
				return 0, false
			}
			node_index = child
		}
		start = position + 1
	}
	return node_index, true
}

// Resolve parent and retain leaf as input view so create and mkdir need no temporary list.
func sim_path_parent(
	state *Sim, path string,
) (parent int, leaf string, found bool) {
	invariant.Always(len(path) <= SIM_PATH_TEXT_BYTES_MAXIMUM,
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

func sim_node_acquire(
	state *Sim, parent int, name string, directory bool,
) (node_index int, err error) {
	if len(name) > SIM_PATH_COMPONENT_BYTES_MAXIMUM {
		return 0, sim_path_component_too_large
	}
	for index := 1; index < len(state.Nodes); index++ {
		if state.Nodes[index].Used {
			continue
		}
		state.Nodes[index] = Sim_Node{
			Used: true, Directory: directory, Parent: parent, Name_Count: len(name),
		}
		copy(state.Nodes[index].Name[:], name)
		return index, nil
	}
	return 0, sim_node_capacity_exceeded
}

// Make exactly one directory; parent creation belongs to higher composition.
func sim_mkdir(state *Sim, path string) (err error) {
	parent, leaf, found := sim_path_parent(state, path)
	if leaf == "" {
		return Path_Exists
	}
	if !found {
		return sim_file_absent
	}
	if !state.Nodes[parent].Directory {
		return sim_not_a_directory
	}
	if _, present := sim_node_find_child(state, parent, leaf); present {
		return Path_Exists
	}
	_, err = sim_node_acquire(state, parent, leaf, true)
	return err
}

// Status reads fixed node storage and returns no collection.
func sim_status(state *Sim, path string) (status File_Status) {
	node_index, found := sim_resolve(state, path)
	if !found {
		return File_Status{}
	}
	node := &state.Nodes[node_index]
	return File_Status{
		Exists: true, Is_Directory: node.Directory, Is_Regular: !node.Directory,
		Size: int64(node.Contents_Count),
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
	if !node.Directory {
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
			Name: sim_node_name(child), Is_Directory: child.Directory,
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
func sim_open(state *Sim, path string) (file File, err error) {
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

// Create or truncate file inside fixed node storage, then bind descriptor slot.
func sim_create(state *Sim, path string) (file File, err error) {
	node, create_err := sim_create_file(state, path)
	if create_err != nil {
		return File(-1), create_err
	}
	descriptor, acquire_err := sim_descriptor_acquire(state, node, false, Sim_Socket{})
	if acquire_err != nil {
		return File(-1), acquire_err
	}
	return descriptor.File, nil
}

func sim_create_file(state *Sim, path string) (node_index int, err error) {
	parent, leaf, found := sim_path_parent(state, path)
	if leaf == "" {
		return 0, sim_is_a_directory
	}
	if !found {
		return 0, sim_file_absent
	}
	if !state.Nodes[parent].Directory {
		return 0, sim_not_a_directory
	}
	if child, present := sim_node_find_child(state, parent, leaf); present {
		if state.Nodes[child].Directory {
			return 0, sim_is_a_directory
		}
		state.Nodes[child].Contents_Count = 0
		return child, nil
	}
	return sim_node_acquire(state, parent, leaf, false)
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
		if !state.Nodes[directory_index].Directory {
			continue
		}
		entry_index := 0
		for generated_count < generated_limit {
			if !prng.Generator_Chance(&state.Generator, sim_grow_chance()) {
				break
			}
			directory := prng.Generator_Chance(
				&state.Generator, sim_subdirectory_chance(),
			)
			child, acquire_err := sim_generated_node_acquire(
				state, directory_index, entry_index, directory,
			)
			invariant.Always(acquire_err == nil,
				"Generated nodes fit reserved caller-owned node storage.")
			generated_count++
			entry_index++
			if !directory {
				sim_generate_bytes(&state.Generator, &state.Nodes[child])
			}
		}
	}
}

// DECIMAL_RADIX selects base ten for generated entry names.
const DECIMAL_RADIX = 10

func sim_generated_node_acquire(
	state *Sim, parent int, value int, directory bool,
) (node_index int, err error) {
	invariant.Always(value >= 0, "A generated entry index is nonnegative.")
	for index := 1; index < len(state.Nodes); index++ {
		if state.Nodes[index].Used {
			continue
		}
		node := &state.Nodes[index]
		*node = Sim_Node{Used: true, Directory: directory, Parent: parent}
		buffer := [bits.WORD_SIZE]byte{}
		position_count := len(buffer) - 1
		buffer[position_count] = byte(value%DECIMAL_RADIX) + '0'
		value /= DECIMAL_RADIX
		for value > 0 {
			position_count--
			buffer[position_count] = byte(value%DECIMAL_RADIX) + '0'
			value /= DECIMAL_RADIX
		}
		node.Name[0] = 'e'
		node.Name_Count = 1 + copy(node.Name[1:], buffer[position_count:])
		return index, nil
	}
	return 0, sim_node_capacity_exceeded
}

// Draw file contents from seed: size Sampled from heavy-tailed distribution, filled with
// seed-drawn bytes.
func sim_generate_bytes(generator *prng.Generator, node *Sim_Node) {
	roll := prng.Generator_Below(generator, 100)
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
		node.Contents[index] = byte(prng.Generator_Next(generator))
	}
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
	state *Sim, completion *time.Completion, kind Sim_Operation_Kind, callback time.Callback,
) (operation *Sim_Operation) {
	invariant.Always(completion.Backend == nil,
		"A simulated completion has no retained operation before submission.")
	for index := range state.Operations {
		if state.Operations[index].Kind != SIM_OPERATION_KIND_FREE {
			continue
		}
		operation = &state.Operations[index]
		*operation = Sim_Operation{Kind: kind, State: state, Callback: callback}
		completion.Backend = unsafe.Pointer(operation)
		return operation
	}
	panic("io: simulated operation capacity exceeded")
}

func sim_operation_borrow(operation *Sim_Operation) {
	descriptor := sim_descriptor_find(operation.State, operation.File)
	invariant.Always(descriptor != nil, "A simulated operation borrows an open descriptor.")
	descriptor.Borrowed++
	operation.Borrowed = true
}

func sim_operation_submit(
	operation *Sim_Operation, completion *time.Completion, latency time.Duration,
) {
	time.Timeline_Submit(
		operation.State.Timeline, completion, latency, sim_operation_complete,
	)
}

// One static retirement callback decodes caller-owned state and frees it before user reentry.
func sim_operation_complete(completion *time.Completion) {
	operation := (*Sim_Operation)(completion.Backend)
	invariant.Always(operation != nil, "A simulated retirement has operation state.")
	data := operation.Data
	err := operation.Operation_Err
	if err == nil {
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
					descriptor.Socket_State.Peer = operation.Address
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
				err = sim_mkdir(operation.State, operation.File_Path)
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
		default:
			panic("io: unknown simulated operation kind")
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
		accepted.Socket_State.Peer = Address_IPV6(
			[IPV6_ADDRESS_BYTES]byte{15: 1}, 0,
		)
	} else {
		accepted.Socket_State.Peer = Address_IPV4(
			[IPV4_ADDRESS_BYTES]byte{127, 0, 0, 1}, 0,
		)
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
		file, err = sim_create(operation.State, operation.File_Path)
	} else {
		file, err = sim_open(operation.State, operation.File_Path)
	}
	return int(file), err
}

func sim_operation_read(operation *Sim_Operation) (data int, err error) {
	if operation.Operation_Err != nil {
		return 0, operation.Operation_Err
	}
	node := &operation.State.Nodes[operation.Node]
	if node.Directory {
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
	if node.Directory {
		return 0, sim_is_a_directory
	}
	write_err := sim_node_write(node, operation.Buffer, operation.Offset)
	if write_err != nil {
		return 0, write_err
	}
	return len(operation.Buffer), nil
}

func sim_operation_deliver(
	operation *Sim_Operation, completion *time.Completion, data int, err error,
) {
	state := operation.State
	file := operation.File
	borrowed := operation.Borrowed
	callback := operation.Callback
	if borrowed {
		descriptor := sim_descriptor_find(state, file)
		invariant.Always(descriptor != nil,
			"A simulated retirement releases a live descriptor.")
		invariant.Always(descriptor.Borrowed > 0,
			"A simulated retirement releases one descriptor borrow.")
		descriptor.Borrowed--
	}
	*operation = Sim_Operation{}
	completion.Backend = nil
	completion.Data = data
	completion.Error = err
	callback(completion)
}

// Stream keeps caller-owned state beside static procedure because returned capturing functions
// escape to heap before any operation starts.
type Stream struct {
	// State remains caller-owned because bound procedure state would allocate.
	State unsafe.Pointer
	// Procedure stays static so returned Stream contains no closure environment.
	Procedure Stream_Procedure
}

// Stream_Procedure owns completion policy because only concrete stream knows whether work is
// immediate, simulated, or kernel-backed.
type Stream_Procedure func(
	state unsafe.Pointer, completion *time.Completion, mode Stream_Mode, buffer []byte,
	offset int64, whence Seek_From, callback Stream_Callback,
)

// Stream_Callback carries explicit state through asynchronous composition because an adapter
// closure would escape until inner operation retires.
type Stream_Callback struct {
	// State restores composed Stream ownership without a closure.
	State unsafe.Pointer
	// Data preserves adapter-specific integer state until inner retirement.
	Data int
	// Callback preserves final receiver while static adapter runs first.
	Callback time.Callback
	// Procedure stays static so callback composition allocates nothing.
	Procedure Stream_Callback_Procedure
}

// Stream_Callback_Procedure is static callback half of Stream_Callback.
type Stream_Callback_Procedure func(
	state unsafe.Pointer, data int, callback time.Callback, completion *time.Completion,
)

// Stream_Callback_Call lets custom Stream procedures retire explicit callback state.
func Stream_Callback_Call(callback Stream_Callback, completion *time.Completion) {
	invariant.Always(callback.Procedure != nil, "A Stream callback has a procedure.")
	callback.Procedure(callback.State, callback.Data, callback.Callback, completion)
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
	stream Stream, completion *time.Completion, buffer []byte, callback time.Callback,
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
	stream Stream, completion *time.Completion, buffer []byte, offset int64,
	callback time.Callback,
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
	stream Stream, completion *time.Completion, buffer []byte, callback time.Callback,
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
	stream Stream, completion *time.Completion, buffer []byte, offset int64,
	callback time.Callback,
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
	stream Stream, completion *time.Completion, offset int64, whence Seek_From,
	callback time.Callback,
) {
	stream_submit(
		stream, completion, STREAM_MODE_SEEK, nil, offset, whence,
		stream_final_callback(callback),
	)
}

// Size reports the bounded storage size through Completion.Data.
func Size(stream Stream, completion *time.Completion, callback time.Callback) {
	stream_submit(
		stream, completion, STREAM_MODE_SIZE, nil, 0, SEEK_FROM_START,
		stream_final_callback(callback),
	)
}

// Flush remains a submitted operation because composition can place buffering behind Stream.
func Flush(stream Stream, completion *time.Completion, callback time.Callback) {
	stream_submit(
		stream, completion, STREAM_MODE_FLUSH, nil, 0, SEEK_FROM_START,
		stream_final_callback(callback),
	)
}

// Close retires on the timeline, so teardown can join it before release of owner state.
func Close(stream Stream, completion *time.Completion, callback time.Callback) {
	stream_submit(
		stream, completion, STREAM_MODE_CLOSE, nil, 0, SEEK_FROM_START,
		stream_final_callback(callback),
	)
}

// Destroy remains distinct from Close because some stream implementations can own storage.
func Destroy(stream Stream, completion *time.Completion, callback time.Callback) {
	stream_submit(
		stream, completion, STREAM_MODE_DESTROY, nil, 0, SEEK_FROM_START,
		stream_final_callback(callback),
	)
}

// Query reports capabilities through Completion.Data because the procedure owns that data.
func Query(stream Stream, completion *time.Completion, callback time.Callback) {
	stream_submit(
		stream, completion, STREAM_MODE_QUERY, nil, 0, SEEK_FROM_START,
		stream_final_callback(callback),
	)
}

func stream_final_callback(callback time.Callback) (stream_callback Stream_Callback) {
	return Stream_Callback{Callback: callback, Procedure: stream_callback_final}
}

func stream_callback_final(
	_ unsafe.Pointer, _ int, callback time.Callback, completion *time.Completion,
) {
	callback(completion)
}

func stream_read_complete(
	_ unsafe.Pointer, buffer_count int, callback time.Callback,
	completion *time.Completion,
) {
	completion.Data, completion.Error = stream_read_checked(
		completion.Data, completion.Error, buffer_count,
	)
	callback(completion)
}

func stream_write_complete(
	_ unsafe.Pointer, buffer_count int, callback time.Callback,
	completion *time.Completion,
) {
	completion.Data, completion.Error = stream_write_checked(
		completion.Data, completion.Error, buffer_count,
	)
	callback(completion)
}

// All public modes enter here, so nil checks and unsupported-mode behavior cannot diverge.
func stream_submit(
	stream Stream, completion *time.Completion, mode Stream_Mode, buffer []byte,
	offset int64, whence Seek_From, callback Stream_Callback,
) {
	invariant.Always(completion != nil, "A Stream operation has a completion.")
	invariant.Always(callback.Procedure != nil, "A Stream operation has a callback.")
	if stream.Procedure == nil {
		stream_complete(completion, 0, Stream_Empty, callback)
		return
	}
	stream.Procedure(stream.State, completion, mode, buffer, offset, whence, callback)
}

// Completion fields hold every scalar result, so Stream needs no second callback type.
func stream_complete(
	completion *time.Completion, data int, err error, callback Stream_Callback,
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
	invariant.Always(state != nil, "A memory Stream has state.")
	return Stream{State: unsafe.Pointer(state), Procedure: stream_memory_procedure}
}

// Memory completes inline because its concrete operation cannot wait on an external endpoint.
func stream_memory_procedure(
	state_pointer unsafe.Pointer, completion *time.Completion, mode Stream_Mode, buffer []byte,
	offset int64, whence Seek_From, callback Stream_Callback,
) {
	state := (*Stream_Memory)(state_pointer)
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
	return Stream{State: unsafe.Pointer(state), Procedure: stream_discard_procedure}
}

// Discard completes inline because it has no external endpoint or scheduled work.
func stream_discard_procedure(
	state_pointer unsafe.Pointer, completion *time.Completion, mode Stream_Mode, buffer []byte,
	_ int64, _ Seek_From, callback Stream_Callback,
) {
	state := (*Stream_Discard)(state_pointer)
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
	invariant.Always(state != nil, "A limit Stream has state.")
	return Stream{State: unsafe.Pointer(state), Procedure: stream_limit_procedure}
}

// Limit forwards one submitted operation and adjusts its budget only after Inner retires.
func stream_limit_procedure(
	state_pointer unsafe.Pointer, completion *time.Completion, mode Stream_Mode, buffer []byte,
	_ int64, _ Seek_From, callback Stream_Callback,
) {
	state := (*Stream_Limit)(state_pointer)
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
				State:     unsafe.Pointer(state),
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
	state *Stream_Limit, completion *time.Completion, buffer []byte,
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
			State: unsafe.Pointer(state), Data: allowed_count,
			Procedure: stream_limit_read_complete,
		})
}

// A limit reports the part it withheld even when Inner stored every byte it received.
func stream_limit_write(
	state *Stream_Limit, completion *time.Completion, buffer []byte,
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
			State: unsafe.Pointer(state), Data: allowed_count,
			Procedure: stream_limit_write_complete,
		})
}

func stream_limit_begin(
	state *Stream_Limit, callback Stream_Callback, buffer_count int,
) {
	invariant.Always(!state.Active, "A limit Stream has at most one operation in flight.")
	state.Active = true
	state.Callback = callback
	state.Buffer_Count = buffer_count
}

func stream_limit_finish(
	state *Stream_Limit, completion *time.Completion,
) {
	callback := state.Callback
	state.Callback = Stream_Callback{}
	state.Buffer_Count = 0
	state.Active = false
	Stream_Callback_Call(callback, completion)
}

func stream_limit_query_complete(
	state_pointer unsafe.Pointer, _ int, _ time.Callback, completion *time.Completion,
) {
	state := (*Stream_Limit)(state_pointer)
	completion.Data = int(Stream_Mode_Set(completion.Data) & STREAM_LIMIT_MODES)
	stream_limit_finish(state, completion)
}

func stream_limit_read_complete(
	state_pointer unsafe.Pointer, allowed_count int, _ time.Callback,
	completion *time.Completion,
) {
	state := (*Stream_Limit)(state_pointer)
	completion.Data, completion.Error = stream_read_checked(
		completion.Data, completion.Error, allowed_count,
	)
	state.Budget -= int64(completion.Data)
	stream_limit_finish(state, completion)
}

func stream_limit_write_complete(
	state_pointer unsafe.Pointer, allowed_count int, _ time.Callback,
	completion *time.Completion,
) {
	state := (*Stream_Limit)(state_pointer)
	completion.Data, completion.Error = stream_write_checked(
		completion.Data, completion.Error, allowed_count,
	)
	state.Budget -= int64(completion.Data)
	if completion.Error == nil {
		if allowed_count < state.Buffer_Count {
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
	invariant.Always(state != nil, "A count Stream has state.")
	return Stream{State: unsafe.Pointer(state), Procedure: stream_count_procedure}
}

// Count updates its tally inside Inner's callback, so callers cannot observe a result before
// its accounting.
func stream_count_procedure(
	state_pointer unsafe.Pointer, completion *time.Completion, mode Stream_Mode, buffer []byte,
	offset int64, whence Seek_From, callback Stream_Callback,
) {
	state := (*Stream_Count)(state_pointer)
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
	state *Stream_Count, completion *time.Completion, buffer []byte,
	callback Stream_Callback,
) {
	stream_count_transfer(state, completion, STREAM_MODE_READ, buffer, 0, callback)
}

// Count records a positioned read before it publishes Inner's completion.
func stream_count_read_at(
	state *Stream_Count, completion *time.Completion, buffer []byte, offset int64,
	callback Stream_Callback,
) {
	stream_count_transfer(state, completion, STREAM_MODE_READ_AT, buffer, offset, callback)
}

// Count records a sequential write before it publishes Inner's completion.
func stream_count_write(
	state *Stream_Count, completion *time.Completion, buffer []byte,
	callback Stream_Callback,
) {
	stream_count_transfer(state, completion, STREAM_MODE_WRITE, buffer, 0, callback)
}

// Count records a positioned write before it publishes Inner's completion.
func stream_count_write_at(
	state *Stream_Count, completion *time.Completion, buffer []byte, offset int64,
	callback Stream_Callback,
) {
	stream_count_transfer(state, completion, STREAM_MODE_WRITE_AT, buffer, offset, callback)
}

func stream_count_transfer(
	state *Stream_Count, completion *time.Completion, mode Stream_Mode, buffer []byte,
	offset int64, callback Stream_Callback,
) {
	invariant.Always(!state.Active, "A count Stream has at most one operation in flight.")
	state.Active = true
	state.Callback = callback
	state.Mode = mode
	stream_submit(state.Inner, completion, mode, buffer, offset, SEEK_FROM_START,
		Stream_Callback{
			State: unsafe.Pointer(state), Data: len(buffer),
			Procedure: stream_count_transfer_complete,
		})
}

func stream_count_transfer_complete(
	state_pointer unsafe.Pointer, buffer_count int, _ time.Callback,
	completion *time.Completion,
) {
	state := (*Stream_Count)(state_pointer)
	if state.Mode == STREAM_MODE_READ {
		completion.Data, completion.Error = stream_read_checked(
			completion.Data, completion.Error, buffer_count,
		)
	} else if state.Mode == STREAM_MODE_READ_AT {
		completion.Data, completion.Error = stream_read_checked(
			completion.Data, completion.Error, buffer_count,
		)
	} else {
		completion.Data, completion.Error = stream_write_checked(
			completion.Data, completion.Error, buffer_count,
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
	invariant.Always(state != nil, "A tee Stream has state.")
	return Stream{State: unsafe.Pointer(state), Procedure: stream_tee_procedure}
}

// Tee submits Second only after First retires, so one caller-owned completion remains valid.
func stream_tee_procedure(
	state_pointer unsafe.Pointer, completion *time.Completion, mode Stream_Mode, buffer []byte,
	_ int64, _ Seek_From, callback Stream_Callback,
) {
	state := (*Stream_Tee)(state_pointer)
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
	callback Stream_Callback,
) {
	stream_tee_begin(state, completion, mode, nil, callback)
}

// Tee reports the smaller count because that count exposes the destination that lost more
// bytes.
func stream_tee_write(
	state *Stream_Tee, completion *time.Completion, buffer []byte,
	callback Stream_Callback,
) {
	stream_tee_begin(state, completion, STREAM_MODE_WRITE, buffer, callback)
}

func stream_tee_begin(
	state *Stream_Tee, completion *time.Completion, mode Stream_Mode, buffer []byte,
	callback Stream_Callback,
) {
	invariant.Always(!state.Active, "A tee Stream has at most one operation in flight.")
	state.Active = true
	state.Callback = callback
	state.Mode = mode
	state.Buffer = buffer
	stream_submit(state.First, completion, mode, buffer, 0, SEEK_FROM_START,
		Stream_Callback{
			State: unsafe.Pointer(state), Procedure: stream_tee_first_complete,
		})
}

func stream_tee_first_complete(
	state_pointer unsafe.Pointer, _ int, _ time.Callback, completion *time.Completion,
) {
	state := (*Stream_Tee)(state_pointer)
	if state.Mode == STREAM_MODE_WRITE {
		completion.Data, completion.Error = stream_write_checked(
			completion.Data, completion.Error, len(state.Buffer),
		)
	}
	state.First_Count = completion.Data
	state.First_Err = completion.Error
	stream_submit(state.Second, completion, state.Mode, state.Buffer, 0, SEEK_FROM_START,
		Stream_Callback{
			State: unsafe.Pointer(state), Procedure: stream_tee_second_complete,
		})
}

func stream_tee_second_complete(
	state_pointer unsafe.Pointer, _ int, _ time.Callback, completion *time.Completion,
) {
	state := (*Stream_Tee)(state_pointer)
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
