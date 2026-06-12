// Package io is the composition tier: the operating-system-backed async IO, the Go
// translation of TigerBeetle's io/darwin.zig and io/linux.zig. It declares package
// io so callers import ".../io/default" and read it as the library with no alias.
package io

import (
	"github.com/james-orcales/james-orcales/shared/io"
	"github.com/james-orcales/james-orcales/shared/time"
)

// Caps the ready set one poll_file_wait returns; sized to drain a busy loop in few
// syscalls without an unbounded buffer.
const poll_events_max = 64

// One ready descriptor the poll reports, decoded from the platform's native event
// into a direction the dispatch understands.
type poll_ready struct {
	// Descriptor is the socket the readiness applies to.
	Descriptor int
	// Writable is true for write readiness, false for read readiness.
	Writable bool
}

// One pending socket completion: the syscall to run when its descriptor becomes
// ready.
type socket_operation struct {
	// Completion is the caller-owned completion this operation belongs to.
	Completion *io.Completion
	// Perform runs the non-blocking syscall and fires the callback, returning false
	// when the syscall reported EAGAIN so the operation stays armed.
	Perform func() (done bool)
}

// Holds the host backend's state: the injected clock, the timeout and completion
// queues the loop drains, and the readiness poll with its per-descriptor socket
// waiters, mirroring TigerBeetle's IO struct.
type operating_system struct {
	// Host is the real clock; deadlines and waits are measured against it.
	Host time.Clock
	// Timeouts are pending timer completions ordered by Ready_At, earliest first.
	Timeouts []*io.Completion
	// Completed are completions whose callbacks are ready to run on the next drain.
	Completed []*io.Completion
	// Poll is the kqueue/epoll descriptor, created on the first socket operation.
	Poll poll_file
	// Poll_Active reports whether Poll has been created.
	Poll_Active bool
	// Read_Waiters maps a descriptor to the operation awaiting its readability.
	Read_Waiters map[int]*socket_operation
	// Write_Waiters maps a descriptor to the operation awaiting its writability.
	Write_Waiters map[int]*socket_operation
}

// New_Operating_System_IO returns an IO backed by the host operating system. File
// reads and writes run inside the loop via pread/pwrite; sockets ride a real
// kqueue/epoll readiness loop; timeouts fire when the clock passes their deadline,
// and Run_For blocks real time bounded by the nearest deadline or socket event —
// the same deadline-bounded wait TigerBeetle performs in kevent/io_uring.
func New_Operating_System_IO(host time.Clock) (loop io.IO) {
	state := &operating_system{Host: host}
	return io.IO{
		Run:     func() { operating_system_run(state) },
		Run_For: func(duration time.Duration) { operating_system_run_for(state, duration) },
		Read: func(
			completion *io.Completion, callback io.Callback,
			file io.File, buffer []byte, offset int64,
		) {
			completion.Callback = func() {
				count, err := read_at(file, buffer, offset)
				callback(completion, count, err)
			}
			state.Completed = append(state.Completed, completion)
		},
		Write: func(
			completion *io.Completion, callback io.Callback,
			file io.File, buffer []byte, offset int64,
		) {
			completion.Callback = func() {
				count, err := write_at(file, buffer, offset)
				callback(completion, count, err)
			}
			state.Completed = append(state.Completed, completion)
		},
		Timeout: func(
			completion *io.Completion, callback io.Timeout_Callback,
			duration time.Duration,
		) {
			completion.Ready_At = host.Now_Monotonic() + time.Moment(duration)
			completion.Callback = func() { callback(completion, nil) }
			operating_system_insert(state, completion)
		},
		Listen: func(host_address string, port int) (listener io.File, err error) {
			descriptor, listen_err := socket_listen(host_address, port)
			return io.File(descriptor), listen_err
		},
		Accept: func(
			completion *io.Completion, callback io.Socket_Callback, listener io.File,
		) {
			operating_system_accept(state, completion, callback, listener)
		},
		Connect: func(
			completion *io.Completion, callback io.Socket_Callback,
			host_address string, port int,
		) {
			operating_system_connect(state, completion, callback, host_address, port)
		},
		Receive: func(
			completion *io.Completion, callback io.Callback,
			socket io.File, buffer []byte,
		) {
			operating_system_receive(state, completion, callback, socket, buffer)
		},
		Send: func(
			completion *io.Completion, callback io.Callback,
			socket io.File, buffer []byte,
		) {
			operating_system_send(state, completion, callback, socket, buffer)
		},
		Close: func(completion *io.Completion, callback io.Timeout_Callback, file io.File) {
			operating_system_close(state, completion, callback, file)
		},
	}
}

// Runs ready completions and one non-blocking socket poll; the host clock moves on
// its own, so Run advances nothing.
func operating_system_run(state *operating_system) {
	operating_system_expire(state)
	operating_system_flush_completed(state)
	if !state.Poll_Active {
		return
	}
	operating_system_poll(state, 0)
}

// Drains ready completions until the duration elapses on the host clock, idling
// each gap in the socket poll (or sleeping when no socket is armed).
func operating_system_run_for(state *operating_system, duration time.Duration) {
	deadline := state.Host.Now_Monotonic() + time.Moment(duration)
	for state.Host.Now_Monotonic() < deadline {
		operating_system_expire(state)
		operating_system_flush_completed(state)
		now := state.Host.Now_Monotonic()
		if now >= deadline {
			return
		}
		operating_system_idle(state, operating_system_wake(state, deadline)-now)
	}
}

// Idles for gap nanoseconds: blocks in the socket poll when a socket is armed so
// readiness wakes it early, otherwise sleeps the host clock.
func operating_system_idle(state *operating_system, gap time.Moment) {
	if gap <= 0 {
		return
	}
	armed := len(state.Read_Waiters) + len(state.Write_Waiters)
	if state.Poll_Active {
		if armed > 0 {
			operating_system_poll(state, int64(gap))
			return
		}
	}
	state.Host.Sleep(time.Duration(gap))
}

// Moves every timeout whose deadline has passed into the completed queue.
func operating_system_expire(state *operating_system) {
	now := state.Host.Now_Monotonic()
	for len(state.Timeouts) > 0 && state.Timeouts[0].Ready_At <= now {
		expired := state.Timeouts[0]
		state.Timeouts = state.Timeouts[1:]
		state.Completed = append(state.Completed, expired)
	}
}

// Runs and clears every ready completion callback.
func operating_system_flush_completed(state *operating_system) {
	ready := state.Completed
	state.Completed = nil
	for index := 0; index < len(ready); index++ {
		ready[index].Callback()
	}
}

// Returns the earliest moment the loop must wake: the outer deadline, pulled in by
// the nearest pending timeout.
func operating_system_wake(state *operating_system, deadline time.Moment) (wake time.Moment) {
	wake = deadline
	if len(state.Timeouts) == 0 {
		return wake
	}
	if state.Timeouts[0].Ready_At < wake {
		return state.Timeouts[0].Ready_At
	}
	return wake
}

// Polls the readiness set with the given nanosecond timeout and dispatches each
// ready descriptor to its waiting operation.
func operating_system_poll(state *operating_system, timeout_ns int64) {
	ready, err := poll_file_wait(state.Poll, timeout_ns)
	if err != nil {
		return
	}
	for index := 0; index < len(ready); index++ {
		operating_system_dispatch(state, ready[index].Descriptor, ready[index].Writable)
	}
}

// Runs the operation waiting on descriptor in the given direction; on completion it
// drops the waiter and disarms the poll, leaving it armed on EAGAIN.
func operating_system_dispatch(state *operating_system, descriptor int, writable bool) {
	waiters := state.Read_Waiters
	if writable {
		waiters = state.Write_Waiters
	}
	operation := waiters[descriptor]
	if operation == nil {
		return
	}
	if !operation.Perform() {
		return
	}
	delete(waiters, descriptor)
	poll_file_disarm(state.Poll, descriptor, writable)
}

// Lazily creates the readiness poll on the first socket operation, so file-only and
// Windows callers never touch it.
func operating_system_poll_ensure(state *operating_system) {
	if state.Poll_Active {
		return
	}
	poll, err := poll_create()
	if err != nil {
		return
	}
	state.Poll = poll
	state.Poll_Active = true
	state.Read_Waiters = make(map[int]*socket_operation)
	state.Write_Waiters = make(map[int]*socket_operation)
}

// Inserts completion into the timeout queue in Ready_At order.
func operating_system_insert(state *operating_system, completion *io.Completion) {
	index := 0
	for index < len(state.Timeouts) && state.Timeouts[index].Ready_At <= completion.Ready_At {
		index++
	}
	state.Timeouts = append(state.Timeouts, nil)
	copy(state.Timeouts[index+1:], state.Timeouts[index:])
	state.Timeouts[index] = completion
}

// Arms listener for readability; on readiness it accepts one connection and fires
// callback with the new descriptor.
func operating_system_accept(
	state *operating_system, completion *io.Completion,
	callback io.Socket_Callback, listener io.File,
) {
	operating_system_poll_ensure(state)
	descriptor := int(listener)
	state.Read_Waiters[descriptor] = &socket_operation{
		Completion: completion,
		Perform: func() (done bool) {
			accepted, again, err := socket_accept(descriptor)
			if again {
				return false
			}
			callback(completion, io.File(accepted), err)
			return true
		},
	}
	poll_file_arm(state.Poll, descriptor, false)
}

// Begins a connection to host_address:port and arms the new socket for writability;
// on readiness it reports the connect result and the connected descriptor.
func operating_system_connect(
	state *operating_system, completion *io.Completion,
	callback io.Socket_Callback, host_address string, port int,
) {
	operating_system_poll_ensure(state)
	descriptor, start_err := socket_connect_start(host_address, port)
	if start_err != nil {
		completion.Callback = func() { callback(completion, io.File(-1), start_err) }
		state.Completed = append(state.Completed, completion)
		return
	}
	state.Write_Waiters[descriptor] = &socket_operation{
		Completion: completion,
		Perform: func() (done bool) {
			callback(completion, io.File(descriptor), socket_connect_error(descriptor))
			return true
		},
	}
	poll_file_arm(state.Poll, descriptor, true)
}

// Arms socket for readability; on readiness it reads once into buffer and reports
// the byte count.
func operating_system_receive(
	state *operating_system, completion *io.Completion,
	callback io.Callback, socket io.File, buffer []byte,
) {
	operating_system_poll_ensure(state)
	descriptor := int(socket)
	state.Read_Waiters[descriptor] = &socket_operation{
		Completion: completion,
		Perform: func() (done bool) {
			count, again, err := socket_receive(descriptor, buffer)
			if again {
				return false
			}
			callback(completion, count, err)
			return true
		},
	}
	poll_file_arm(state.Poll, descriptor, false)
}

// Arms socket for writability; on readiness it writes buffer once and reports the
// byte count.
func operating_system_send(
	state *operating_system, completion *io.Completion,
	callback io.Callback, socket io.File, buffer []byte,
) {
	operating_system_poll_ensure(state)
	descriptor := int(socket)
	state.Write_Waiters[descriptor] = &socket_operation{
		Completion: completion,
		Perform: func() (done bool) {
			count, again, err := socket_send(descriptor, buffer)
			if again {
				return false
			}
			callback(completion, count, err)
			return true
		},
	}
	poll_file_arm(state.Poll, descriptor, true)
}

// Closes file and queues its completion to fire on the next drain.
func operating_system_close(
	state *operating_system, completion *io.Completion,
	callback io.Timeout_Callback, file io.File,
) {
	err := socket_close(int(file))
	completion.Callback = func() { callback(completion, err) }
	state.Completed = append(state.Completed, completion)
}
