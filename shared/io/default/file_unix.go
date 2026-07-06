//go:build unix

package io

import (
	"errors"
	"net"
	"path/filepath"
	"syscall"

	"local/james-orcales/shared/io"
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
	return syscall.Open(path, syscall.O_RDONLY, 0)
}

// Creates or truncates path for writing via the open syscall, returning its descriptor.
func file_create(path string) (descriptor int, err error) {
	return syscall.Open(path, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_TRUNC, 0o644)
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

// Resolves host:port into an IPv4 socket address without DNS; a non-literal host is
// rejected so no blocking lookup ever runs on the loop.
func socket_address(host string, port int) (address syscall.SockaddrInet4, err error) {
	four := net.ParseIP(host).To4()
	if four == nil {
		return address, syscall.EINVAL
	}
	address.Port = port
	copy(address.Addr[:], four)
	return address, nil
}

// Creates a non-blocking TCP socket bound to host:port and starts listening,
// returning the listening descriptor.
func socket_listen(host string, port int) (descriptor int, err error) {
	address, address_err := socket_address(host, port)
	if address_err != nil {
		return -1, address_err
	}
	descriptor, err = syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		return -1, err
	}
	socket_prepare(descriptor)
	bind_err := syscall.Bind(descriptor, &address)
	if bind_err != nil {
		syscall.Close(descriptor)
		return -1, bind_err
	}
	listen_err := syscall.Listen(descriptor, syscall.SOMAXCONN)
	if listen_err != nil {
		syscall.Close(descriptor)
		return -1, listen_err
	}
	return descriptor, nil
}

// Sets a fresh descriptor non-blocking and reusable, ignoring the rare option errors
// that never occur on a just-created socket.
func socket_prepare(descriptor int) {
	non_block_err := syscall.SetNonblock(descriptor, true)
	if non_block_err != nil {
		return
	}
	syscall.SetsockoptInt(descriptor, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
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
	return descriptor, false, nil
}

// Opens a non-blocking TCP socket and begins connecting to host:port; an in-progress
// handshake returns no error, completion comes later.
func socket_connect_start(host string, port int) (descriptor int, err error) {
	address, address_err := socket_address(host, port)
	if address_err != nil {
		return -1, address_err
	}
	descriptor, err = syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		return -1, err
	}
	socket_prepare(descriptor)
	connect_err := syscall.Connect(descriptor, &address)
	if connect_err == nil {
		return descriptor, nil
	}
	if connect_err == syscall.EINPROGRESS {
		return descriptor, nil
	}
	syscall.Close(descriptor)
	return -1, connect_err
}

// Returns the pending error on descriptor after a connect completes, or nil when the
// handshake succeeded.
func socket_connect_error(descriptor int) (err error) {
	value, get_err := syscall.GetsockoptInt(descriptor, syscall.SOL_SOCKET, syscall.SO_ERROR)
	if get_err != nil {
		return get_err
	}
	if value != 0 {
		return syscall.Errno(value)
	}
	return nil
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

// Releases descriptor.
func socket_close(descriptor int) (err error) {
	return syscall.Close(descriptor)
}

// Creates a non-blocking self-pipe used to wake the loop out of a blocking poll when a
// worker or TLS goroutine posts a completion from off the loop thread.
func wake_create() (read int, write int, err error) {
	pair := [2]int{}
	pipe_err := syscall.Pipe(pair[:])
	if pipe_err != nil {
		return -1, -1, pipe_err
	}
	syscall.SetNonblock(pair[0], true)
	syscall.SetNonblock(pair[1], true)
	return pair[0], pair[1], nil
}

// Writes one byte to the wake pipe so a blocked poll returns; a full pipe's failed
// non-blocking write is ignored, since one pending byte already wakes the loop.
func wake_poke(write int) {
	one := [1]byte{}
	syscall.Write(write, one[:])
}

// Drains the wake pipe's pending bytes, bounded so a flood cannot spin the loop.
func wake_drain(read int) {
	scratch := make([]byte, 4096)
	for pass_index := 0; pass_index < wake_drain_passes_max; pass_index++ {
		count, err := syscall.Read(read, scratch)
		if err != nil {
			return
		}
		if count < len(scratch) {
			return
		}
	}
}
