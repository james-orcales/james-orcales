//go:build unix

package io

import (
	"errors"
	"net"
	"path/filepath"
	"syscall"

	"local/james-orcales/g/shared/io"
)

// Bounds one readdir pass into a fixed buffer, so a large directory is read in repeated
// passes rather than one unbounded allocation.
const directory_read_bytes = 8192

// Caps the number of readdir passes so a pathological directory errors rather than looping
// unbounded; 4096 passes of directory_read_bytes cover hundreds of thousands of entries.
const directory_read_passes_max = 4096

// Reads up to len(buffer) bytes from file at offset via the pread syscall — the raw
// positioned read TigerBeetle's posix backend uses.
func read_at(file io.File, buffer []byte, offset int64) (count int, err error) {
	return syscall.Pread(int(file), buffer, offset)
}

// Writes buffer to file at offset via the pwrite syscall.
func write_at(file io.File, buffer []byte, offset int64) (count int, err error) {
	return syscall.Pwrite(int(file), buffer, offset)
}

// Opens the file at path for reading via the open syscall, returning its descriptor.
func file_open(path string) (descriptor int, err error) {
	return syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC, 0)
}

// Creates or truncates path for writing via the open syscall, returning its descriptor.
func file_create(path string) (descriptor int, err error) {
	return syscall.Open(
		path, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_TRUNC|syscall.O_CLOEXEC, 0o644,
	)
}

// Reports whether path exists, whether it is a directory, and its byte size via lstat. An
// absent path is Exists false with a nil error, so a caller distinguishes "not there" from a
// real stat failure.
func file_status(path string) (status io.File_Status, err error) {
	metadata := syscall.Stat_t{}
	stat_err := syscall.Lstat(path, &metadata)
	if stat_err == syscall.ENOENT {
		return io.File_Status{}, nil
	}
	if stat_err != nil {
		return io.File_Status{}, stat_err
	}
	return io.File_Status{
		Exists:       true,
		Is_Directory: metadata.Mode&syscall.S_IFMT == syscall.S_IFDIR,
		Size:         metadata.Size,
	}, nil
}

// Lists path's immediate children, each named with whether it is a directory. It reads the
// directory's raw entries in fixed-size passes and stats each name for its kind — the same
// per-entry stat a walk over os.DirFS performs.
func file_read_directory(path string) (entries []io.Directory_Entry, err error) {
	descriptor, open_err := syscall.Open(path, syscall.O_RDONLY, 0)
	if open_err != nil {
		return nil, open_err
	}
	defer syscall.Close(descriptor)
	buffer := make([]byte, directory_read_bytes)
	for pass_index := 0; pass_index < directory_read_passes_max; pass_index++ {
		count, read_err := syscall.ReadDirent(descriptor, buffer)
		if read_err != nil {
			return nil, read_err
		}
		if count <= 0 {
			return entries, nil
		}
		_, _, names := syscall.ParseDirent(buffer[:count], -1, nil)
		for _, name := range names {
			status, status_err := file_status(filepath.Join(path, name))
			if status_err != nil {
				return nil, status_err
			}
			entries = append(entries, io.Directory_Entry{
				Name:         name,
				Is_Directory: status.Is_Directory,
			})
		}
	}
	return nil, errors.New("io: directory exceeds the maximum entry count")
}

// Reports the remote IP address of descriptor via getpeername; a non-IP peer yields the
// empty address with no error.
func socket_peer_address(descriptor int) (address string, err error) {
	name, get_err := syscall.Getpeername(descriptor)
	if get_err != nil {
		return "", get_err
	}
	switch peer := name.(type) {
	case *syscall.SockaddrInet4:
		return net.IP(peer.Addr[:]).String(), nil
	case *syscall.SockaddrInet6:
		return net.IP(peer.Addr[:]).String(), nil
	}
	return "", nil
}

// Reports whether err is the non-blocking "try again" signal that keeps an operation
// armed rather than completing it.
func socket_again(err error) (again bool) {
	if err == syscall.EAGAIN {
		return true
	}
	return err == syscall.EWOULDBLOCK
}

// Converts the backend-independent explicit-family address to a POSIX socket address.
func socket_address(address io.Address) (system syscall.Sockaddr, err error) {
	if address.Family == io.FAMILY_IPV4 {
		four := &syscall.SockaddrInet4{Port: int(address.Port)}
		copy(four.Addr[:], address.IP[:4])
		return four, nil
	}
	if address.Family == io.FAMILY_IPV6 {
		six := &syscall.SockaddrInet6{Port: int(address.Port)}
		copy(six.Addr[:], address.IP[:])
		return six, nil
	}
	return nil, syscall.EAFNOSUPPORT
}

// Resolve turns a host into the IPv4 literal socket_address requires: an IP literal passes through
// unchanged, a name is looked up and its first IPv4 returned. socket_address rejects a name so no
// lookup runs on the dial path; a caller resolves here first and hands Connect an address. It is
// the prod Resolver injected into the shared http client, failing closed when the host has no IPv4.
func Resolve(host string) (address string, err error) {
	if net.ParseIP(host) != nil {
		return host, nil
	}
	found, lookup_err := net.LookupIP(host)
	if lookup_err != nil {
		return "", lookup_err
	}
	for index := range found {
		if four := found[index].To4(); four != nil {
			return four.String(), nil
		}
	}
	return "", errors.New("io: no IPv4 address for host " + host)
}

// Binds and listens on a caller-owned socket, returning the resolved address for port zero.
func socket_listen(
	descriptor int, address io.Address, options io.Listen_Options,
) (resolved io.Address, err error) {
	system, address_err := socket_address(address)
	if address_err != nil {
		return io.Address{}, address_err
	}
	if options.Backlog == 0 {
		return io.Address{}, syscall.EINVAL
	}
	reuse_err := syscall.SetsockoptInt(descriptor, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	if reuse_err != nil {
		return io.Address{}, reuse_err
	}
	bind_err := syscall.Bind(descriptor, system)
	if bind_err != nil {
		return io.Address{}, bind_err
	}
	listen_err := syscall.Listen(descriptor, int(options.Backlog))
	if listen_err != nil {
		return io.Address{}, listen_err
	}
	name, name_err := syscall.Getsockname(descriptor)
	if name_err != nil {
		return io.Address{}, name_err
	}
	return socket_address_from_system(name)
}

// Accepts one pending connection on listener, returning a non-blocking connected
// descriptor; again is true when none is ready yet.
func socket_accept(listener int) (descriptor int, again bool, err error) {
	descriptor, _, err = syscall.Accept(listener)
	if err != nil {
		return -1, socket_again(err), err
	}
	non_block_err := syscall.SetNonblock(descriptor, true)
	if non_block_err != nil {
		syscall.Close(descriptor)
		return -1, false, non_block_err
	}
	configure_err := socket_accept_configure(descriptor)
	if configure_err != nil {
		syscall.Close(descriptor)
		return -1, false, configure_err
	}
	return descriptor, false, nil
}

type socket_connect_start_input struct {
	Descriptor int
	Address    io.Address
}

// Begins connecting a caller-owned descriptor; an in-progress handshake completes later.
func socket_connect_start(input *socket_connect_start_input) (err error) {
	address, address_err := socket_address(input.Address)
	if address_err != nil {
		return address_err
	}
	connect_err := syscall.Connect(input.Descriptor, address)
	if connect_err == nil {
		return nil
	}
	if connect_err == syscall.EINPROGRESS {
		return nil
	}
	return socket_connect_translate(connect_err)
}

// Returns the pending error on descriptor after a connect completes, or nil when the
// handshake succeeded.
func socket_connect_error(descriptor int) (err error) {
	value, get_err := syscall.GetsockoptInt(descriptor, syscall.SOL_SOCKET, syscall.SO_ERROR)
	if get_err != nil {
		return get_err
	}
	if value != 0 {
		return socket_connect_translate(syscall.Errno(value))
	}
	return nil
}

// Translates platform-specific connect refusal to the backend-independent sentinel.
func socket_connect_translate(err error) (translated error) {
	if err == syscall.ECONNREFUSED {
		return io.Connection_Refused
	}
	return err
}

// Reads up to len(buffer) bytes from descriptor; again is true when no data is ready
// yet, and a zero count with no error marks a closed peer.
func socket_receive(descriptor int, buffer []byte) (count int, again bool, err error) {
	count, err = syscall.Read(descriptor, buffer)
	if err != nil {
		return 0, socket_again(err), err
	}
	return count, false, nil
}

// Writes up to len(buffer) bytes to descriptor; again is true when the kernel buffer
// is full and the operation must stay armed.
func socket_send(descriptor int, buffer []byte) (count int, again bool, err error) {
	count, err = syscall.Write(descriptor, buffer)
	if err != nil {
		return 0, socket_again(err), err
	}
	return count, false, nil
}

// Attempts one synchronous send, returning sent false for would-block or another send failure.
func socket_send_now(descriptor int, buffer []byte) (count int, sent bool) {
	count, again, err := socket_send(descriptor, buffer)
	if again {
		return 0, false
	}
	if err != nil {
		return 0, false
	}
	return count, true
}

// Shuts down one or both connected-socket directions synchronously.
func socket_shutdown(descriptor int, how io.Shutdown_How) (err error) {
	system_how := syscall.SHUT_RD
	if how == io.SHUTDOWN_SEND {
		system_how = syscall.SHUT_WR
	}
	if how == io.SHUTDOWN_BOTH {
		system_how = syscall.SHUT_RDWR
	}
	err = syscall.Shutdown(descriptor, system_how)
	if err == syscall.ENOTCONN {
		return io.Socket_Not_Connected
	}
	return socket_send_translate(err)
}

// Converts a POSIX getsockname result to the backend-independent address.
func socket_address_from_system(system syscall.Sockaddr) (address io.Address, err error) {
	if four, ok := system.(*syscall.SockaddrInet4); ok {
		return io.Address_I_Pv4(four.Addr, uint16(four.Port)), nil
	}
	if six, ok := system.(*syscall.SockaddrInet6); ok {
		return io.Address_I_Pv6(six.Addr, uint16(six.Port)), nil
	}
	return io.Address{}, syscall.EAFNOSUPPORT
}

// Translates send-side broken-pipe errors to the portable result.
func socket_send_translate(err error) (translated error) {
	if err == syscall.EPIPE {
		return io.Broken_Pipe
	}
	return err
}

// Releases descriptor.
func socket_close(descriptor int) (err error) {
	return syscall.Close(descriptor)
}
