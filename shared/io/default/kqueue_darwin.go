//go:build darwin

package io

import "syscall"

// The kqueue descriptor backing the readiness loop on Darwin.
type poll_file int

// Opens a fresh kqueue.
func poll_create() (poll poll_file, err error) {
	descriptor, create_err := syscall.Kqueue()
	if create_err != nil {
		return 0, create_err
	}
	return poll_file(descriptor), nil
}

// Registers read or write interest in descriptor on poll, level-triggered so the
// readiness re-reports until the operation disarms it.
func poll_file_arm(poll poll_file, descriptor int, writable bool) (err error) {
	change := syscall.Kevent_t{
		Ident:  uint64(descriptor),
		Filter: poll_filter(writable),
		Flags:  syscall.EV_ADD | syscall.EV_ENABLE,
	}
	_, err = syscall.Kevent(int(poll), []syscall.Kevent_t{change}, nil, nil)
	return err
}

// Removes read or write interest in descriptor from poll.
func poll_file_disarm(poll poll_file, descriptor int, writable bool) (err error) {
	change := syscall.Kevent_t{
		Ident:  uint64(descriptor),
		Filter: poll_filter(writable),
		Flags:  syscall.EV_DELETE,
	}
	_, err = syscall.Kevent(int(poll), []syscall.Kevent_t{change}, nil, nil)
	return err
}

// Maps a direction to its kqueue filter.
func poll_filter(writable bool) (filter int16) {
	if writable {
		return syscall.EVFILT_WRITE
	}
	return syscall.EVFILT_READ
}

// Blocks for up to timeout_ns and returns the ready descriptors, decoding each kqueue
// event into a direction.
func poll_file_wait(poll poll_file, timeout_ns int64) (ready []poll_ready, err error) {
	native := make([]syscall.Kevent_t, poll_events_max)
	timeout := syscall.NsecToTimespec(timeout_ns)
	count, wait_err := syscall.Kevent(int(poll), nil, native, &timeout)
	if wait_err != nil {
		return nil, wait_err
	}
	for index := 0; index < count; index++ {
		ready = append(ready, poll_ready{
			Descriptor: int(native[index].Ident),
			Writable:   native[index].Filter == syscall.EVFILT_WRITE,
		})
	}
	return ready, nil
}
