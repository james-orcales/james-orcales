// Package io is the composition tier: the operating-system-backed async IO, the Go
// translation of TigerBeetle's io/darwin.zig and io/linux.zig. It declares package
// io so callers import ".../io/default" and read it as the library with no alias.
package io

import (
	"github.com/james-orcales/james-orcales/shared/io"
	"github.com/james-orcales/james-orcales/shared/time"
)

// Holds the host backend's state: the injected clock plus the timeout and completion
// queues the loop drains, mirroring TigerBeetle's IO struct.
type operating_system struct {
	// Host is the real clock; deadlines and waits are measured against it.
	Host time.Clock
	// Timeouts are pending timer completions ordered by Ready_At, earliest first.
	Timeouts []*io.Completion
	// Completed are completions whose callbacks are ready to run on the next drain.
	Completed []*io.Completion
}

// New_Operating_System_IO returns an IO backed by the host operating system. File
// reads and writes run inside the loop via pread/pwrite; timeouts fire when the
// clock passes their deadline, and Run_For blocks real time bounded by the nearest
// deadline — the same deadline-bounded wait TigerBeetle performs in kevent/io_uring.
func New_Operating_System_IO(host time.Clock) (loop io.IO) {
	state := &operating_system{Host: host}
	return io.IO{
		Run: func() { operating_system_run(state) },
		Run_For: func(duration time.Duration) {
			operating_system_run_for(state, duration)
		},
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
	}
}

// Moves expired timeouts into the completed queue, then runs every ready callback.
func operating_system_run(state *operating_system) {
	now := state.Host.Now_Monotonic()
	for len(state.Timeouts) > 0 && state.Timeouts[0].Ready_At <= now {
		expired := state.Timeouts[0]
		state.Timeouts = state.Timeouts[1:]
		state.Completed = append(state.Completed, expired)
	}
	ready := state.Completed
	state.Completed = nil
	for _, completion := range ready {
		completion.Callback()
	}
}

// Drains ready completions until the duration has elapsed on the host clock,
// sleeping real time up to the nearest timeout deadline between drains.
func operating_system_run_for(state *operating_system, duration time.Duration) {
	deadline := state.Host.Now_Monotonic() + time.Moment(duration)
	for state.Host.Now_Monotonic() < deadline {
		operating_system_run(state)
		now := state.Host.Now_Monotonic()
		if now >= deadline {
			return
		}
		wake := deadline
		if len(state.Timeouts) > 0 {
			if state.Timeouts[0].Ready_At < wake {
				wake = state.Timeouts[0].Ready_At
			}
		}
		gap := wake - now
		if gap > 0 {
			state.Host.Sleep(time.Duration(gap))
		}
	}
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
