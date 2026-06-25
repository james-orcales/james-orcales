//go:build unix

package io

import (
	"net"
	"syscall"

	"github.com/james-orcales/james-orcales/shared/io"
)

// Reads up to len(buffer) bytes from file at offset via the pread syscall — the raw
// positioned read TigerBeetle's posix backend uses.
func read_at(file io.File, buffer []byte, offset int64) (count int, err error) {
	return syscall.Pread(int(file), buffer, offset)
}

// Writes buffer to file at offset via the pwrite syscall.
func write_at(file io.File, buffer []byte, offset int64) (count int, err error) {
	return syscall.Pwrite(int(file), buffer, offset)
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
