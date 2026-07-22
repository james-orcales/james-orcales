//go:build linux

package io

import "syscall"

// SO_REUSEPORT socket option, hand-written because the stdlib syscall package omits it on Linux
// (unlike Darwin) and pulling in golang.org/x/sys/unix for one constant is not worth it.
const SOCKET_REUSEPORT = 0xf

// The epoll descriptor plus the per-descriptor interest mask epoll needs to combine
// read and write directions on one entry.
type Poll_File struct {
	// Descriptor is the epoll instance.
	Descriptor int
	// Interest tracks the current event mask armed for each socket descriptor.
	Interest map[int]uint32
}

// Opens a fresh epoll instance.
func poll_create() (poll Poll_File, err error) {
	descriptor, create_err := syscall.EpollCreate1(syscall.EPOLL_CLOEXEC)
	if create_err != nil {
		return poll, create_err
	}
	return Poll_File{Descriptor: descriptor, Interest: make(map[int]uint32)}, nil
}

// Adds read or write interest in descriptor, level-triggered, merging it into any
// direction already armed on the same descriptor.
func poll_file_arm(poll Poll_File, descriptor int, writable bool) (err error) {
	mask := poll.Interest[descriptor]
	operation := syscall.EPOLL_CTL_MOD
	if mask == 0 {
		operation = syscall.EPOLL_CTL_ADD
	}
	mask = mask | poll_bit(writable)
	poll.Interest[descriptor] = mask
	event := syscall.EpollEvent{Events: mask, Fd: int32(descriptor)}
	return syscall.EpollCtl(poll.Descriptor, operation, descriptor, &event)
}

// Clears read or write interest in descriptor, removing the entry once no direction
// remains.
func poll_file_disarm(poll Poll_File, descriptor int, writable bool) (err error) {
	mask := poll.Interest[descriptor] &^ poll_bit(writable)
	if mask == 0 {
		delete(poll.Interest, descriptor)
		return syscall.EpollCtl(poll.Descriptor, syscall.EPOLL_CTL_DEL, descriptor, nil)
	}
	poll.Interest[descriptor] = mask
	event := syscall.EpollEvent{Events: mask, Fd: int32(descriptor)}
	return syscall.EpollCtl(poll.Descriptor, syscall.EPOLL_CTL_MOD, descriptor, &event)
}

// Maps a direction to its epoll event bit.
func poll_bit(writable bool) (bit uint32) {
	if writable {
		return uint32(syscall.EPOLLOUT)
	}
	return uint32(syscall.EPOLLIN)
}

// Blocks for up to timeout_ns — or until an event when timeout_ns is negative, the -1
// EpollWait waits on unbounded — and returns the ready descriptors, emitting a separate entry
// per ready direction so both can be dispatched.
func poll_file_wait(poll Poll_File, timeout_ns int64) (ready []Poll_Ready, err error) {
	native := make([]syscall.EpollEvent, POLL_EVENTS_MAX)
	milliseconds := -1
	if timeout_ns >= 0 {
		milliseconds = int(timeout_ns / 1_000_000)
	}
	count, wait_err := syscall.EpollWait(poll.Descriptor, native, milliseconds)
	if wait_err != nil {
		return nil, wait_err
	}
	for index := 0; index < count; index++ {
		descriptor := int(native[index].Fd)
		if native[index].Events&poll_bit(false) != 0 {
			ready = append(ready, Poll_Ready{Descriptor: descriptor, Writable: false})
		}
		if native[index].Events&poll_bit(true) != 0 {
			ready = append(ready, Poll_Ready{Descriptor: descriptor, Writable: true})
		}
	}
	return ready, nil
}
