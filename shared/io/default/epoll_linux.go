//go:build linux

package io

import "syscall"

// The epoll descriptor plus the per-descriptor interest mask epoll needs to combine
// read and write directions on one entry.
type poll_file struct {
	// Descriptor is the epoll instance.
	Descriptor int
	// Interest tracks the current event mask armed for each socket descriptor.
	Interest map[int]uint32
}

// Opens a fresh epoll instance.
func poll_create() (poll poll_file, err error) {
	descriptor, create_err := syscall.EpollCreate1(syscall.EPOLL_CLOEXEC)
	if create_err != nil {
		return poll, create_err
	}
	return poll_file{Descriptor: descriptor, Interest: make(map[int]uint32)}, nil
}

// Adds read or write interest in descriptor, level-triggered, merging it into any
// direction already armed on the same descriptor.
func poll_file_arm(poll poll_file, descriptor int, writable bool) (err error) {
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
func poll_file_disarm(poll poll_file, descriptor int, writable bool) (err error) {
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

// Blocks for up to timeout_ns and returns the ready descriptors, emitting a separate
// entry per ready direction so both can be dispatched.
func poll_file_wait(poll poll_file, timeout_ns int64) (ready []poll_ready, err error) {
	native := make([]syscall.EpollEvent, poll_events_max)
	count, wait_err := syscall.EpollWait(poll.Descriptor, native, int(timeout_ns/1_000_000))
	if wait_err != nil {
		return nil, wait_err
	}
	for index := 0; index < count; index++ {
		descriptor := int(native[index].Fd)
		if native[index].Events&poll_bit(false) != 0 {
			ready = append(ready, poll_ready{Descriptor: descriptor, Writable: false})
		}
		if native[index].Events&poll_bit(true) != 0 {
			ready = append(ready, poll_ready{Descriptor: descriptor, Writable: true})
		}
	}
	return ready, nil
}
