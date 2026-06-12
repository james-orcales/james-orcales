// Package io is a dependency-injected, completion-based async IO surface modeled on
// TigerBeetle's io.zig, expressed as a struct of closures so the backend is chosen
// by value: a deterministic simulated backend here, and a real OS backend
// (epoll/kqueue/IOCP, the production counterpart) in io/default. Every operation is
// submitted with a caller-owned Completion and a callback, and results arrive only
// when the loop is run. The IO owns a time.Clock so timeouts ride the same timeline
// as IO completions — the heart of the model.
package io

import "github.com/james-orcales/james-orcales/shared/time"

// File identifies an open file or socket. The simulated backend ignores it (its
// storage is in-memory); a real backend maps it to a descriptor or handle.
type File int32

// Callback receives the result of a read or write: the byte count, or an error.
type Callback func(completion *Completion, count int, err error)

// Timeout_Callback receives the result of a timeout: success, or a cancellation.
type Timeout_Callback func(completion *Completion, err error)

// Socket_Callback receives a newly available socket — an accepted server
// connection or a completed client connect — or an error.
type Socket_Callback func(completion *Completion, socket File, err error)

// Completion is the caller-owned storage for one in-flight operation —
// TigerBeetle's IO.Completion. The caller allocates it, so the loop never does, and
// must keep it alive until the callback fires.
type Completion struct {
	// Callback is the closure the backend runs on completion; it closes over the
	// typed callback and the computed result.
	Callback func()
	// Ready_At is the virtual Moment this operation completes, mirroring
	// TigerBeetle's Storage.Read.ready_at.
	Ready_At time.Moment
}

// IO is the injected async IO surface — TigerBeetle's `IO`. Code submits operations
// with a Completion and callback and drives the loop with Run or Run_For, never
// knowing which backend it holds. Recv, Send, Accept, Connect, Close, and Fsync
// follow Read and Write's shape and are omitted here.
type IO struct {
	// Run submits queued operations and reaps every ready completion without
	// blocking, then advances the clock one tick (TigerBeetle IO.run).
	Run func()
	// Run_For runs the loop until the duration has elapsed on the clock, delivering
	// completions as they come due (TigerBeetle IO.run_for_ns).
	Run_For func(duration time.Duration)
	// Read reads len(buffer) bytes from file at offset; callback fires with the byte
	// count or error once the operation completes (TigerBeetle IO.read).
	Read func(
		completion *Completion, callback Callback, file File, buffer []byte, offset int64,
	)
	// Write writes buffer to file at offset (TigerBeetle IO.write).
	Write func(
		completion *Completion, callback Callback, file File, buffer []byte, offset int64,
	)
	// Timeout fires callback after the duration on the clock, off the same queue the
	// IO completions use (TigerBeetle IO.timeout).
	Timeout func(completion *Completion, callback Timeout_Callback, duration time.Duration)
	// Listen binds and listens on host:port, returning the listening socket
	// synchronously — bind never blocks, so it carries no Completion
	// (TigerBeetle IO.open_socket then listen).
	Listen func(host string, port int) (listener File, err error)
	// Accept yields one inbound connection on listener; callback fires with the
	// accepted socket once a peer arrives, and the caller re-arms for the next
	// (TigerBeetle IO.accept).
	Accept func(completion *Completion, callback Socket_Callback, listener File)
	// Connect opens a socket to host:port; callback fires with the connected
	// socket once the handshake completes (TigerBeetle IO.connect).
	Connect func(completion *Completion, callback Socket_Callback, host string, port int)
	// Receive reads up to len(buffer) bytes from socket; callback reports the byte
	// count once data arrives (TigerBeetle IO.recv).
	Receive func(completion *Completion, callback Callback, socket File, buffer []byte)
	// Send writes buffer to socket; callback reports the byte count once the kernel
	// accepts it (TigerBeetle IO.send).
	Send func(completion *Completion, callback Callback, socket File, buffer []byte)
	// Close releases file's descriptor; callback fires once it is closed
	// (TigerBeetle IO.close).
	Close func(completion *Completion, callback Timeout_Callback, file File)
}

// Sim is the deterministic, in-memory IO backend — TigerBeetle's simulated Storage
// plus PacketSimulator. Each operation is scheduled to complete at now plus a
// latency on the injected clock; a run fires every operation whose Ready_At has
// arrived, then advances the clock one tick. No real syscalls and no waiting, so a
// run is fully reproducible from the clock and the latency model.
type Sim struct {
	// Clock is the time source; "now" is Clock.Now_Monotonic.
	Clock time.Clock
	// Latency returns the modeled completion delay for the next operation;
	// deterministic when seeded, mirroring TigerBeetle's Storage.read_latency.
	Latency func() (duration time.Duration)
	// Queue holds in-flight completions ordered by Ready_At, earliest first — a
	// per-subsystem ready_at queue.
	Queue []*Completion
	// Next_File is the synthetic descriptor counter; Listen, Accept, and Connect
	// hand out the next value so every socket is distinct.
	Next_File File
}

// Sim_To_IO returns an IO backed by sim. The closures share sim, so a submitted
// operation and the loop that completes it see the same queue and time.
func Sim_To_IO(sim *Sim) (loop IO) {
	return IO{
		Run:     func() { sim_run(sim) },
		Run_For: func(duration time.Duration) { sim_run_for(sim, duration) },
		Read: func(
			completion *Completion, callback Callback,
			file File, buffer []byte, offset int64,
		) {
			sim_submit(sim, completion, sim.Latency(), func() {
				callback(completion, len(buffer), nil)
			})
		},
		Write: func(
			completion *Completion, callback Callback,
			file File, buffer []byte, offset int64,
		) {
			sim_submit(sim, completion, sim.Latency(), func() {
				callback(completion, len(buffer), nil)
			})
		},
		Timeout: func(
			completion *Completion, callback Timeout_Callback,
			duration time.Duration,
		) {
			sim_submit(sim, completion, duration, func() {
				callback(completion, nil)
			})
		},
		Listen: func(host string, port int) (listener File, err error) {
			sim.Next_File++
			return sim.Next_File, nil
		},
		Accept: func(completion *Completion, callback Socket_Callback, listener File) {
			sim_yield_socket(sim, completion, callback)
		},
		Connect: func(
			completion *Completion, callback Socket_Callback, host string, port int,
		) {
			sim_yield_socket(sim, completion, callback)
		},
		Receive: func(
			completion *Completion, callback Callback, socket File, buffer []byte,
		) {
			sim_submit(sim, completion, sim.Latency(), func() {
				callback(completion, len(buffer), nil)
			})
		},
		Send: func(completion *Completion, callback Callback, socket File, buffer []byte) {
			sim_submit(sim, completion, sim.Latency(), func() {
				callback(completion, len(buffer), nil)
			})
		},
		Close: func(completion *Completion, callback Timeout_Callback, file File) {
			sim_submit(sim, completion, sim.Latency(), func() {
				callback(completion, nil)
			})
		},
	}
}

// Returns the current virtual Moment; Sim never reads the operating-system time.
func sim_now(sim *Sim) (now time.Moment) {
	return sim.Clock.Now_Monotonic()
}

// Schedules completion to fire at now plus latency and inserts it in Ready_At order,
// mirroring TigerBeetle's ready_at = tick_instant + latency.
func sim_submit(sim *Sim, completion *Completion, latency time.Duration, callback func()) {
	completion.Ready_At = sim_now(sim) + time.Moment(latency)
	completion.Callback = callback

	index := 0
	for index < len(sim.Queue) && sim.Queue[index].Ready_At <= completion.Ready_At {
		index++
	}
	sim.Queue = append(sim.Queue, nil)
	copy(sim.Queue[index+1:], sim.Queue[index:])
	sim.Queue[index] = completion
}

// Schedules callback to receive the next synthetic descriptor after the modeled
// latency, shared by Accept and Connect.
func sim_yield_socket(sim *Sim, completion *Completion, callback Socket_Callback) {
	sim.Next_File++
	socket := sim.Next_File
	sim_submit(sim, completion, sim.Latency(), func() {
		callback(completion, socket, nil)
	})
}

// Fires the earliest completion if it is due as of now, reporting whether it did,
// mirroring TigerBeetle's Storage.step.
func sim_step(sim *Sim) (advanced bool) {
	if len(sim.Queue) == 0 {
		return false
	}
	if sim.Queue[0].Ready_At > sim_now(sim) {
		return false
	}
	completion := sim.Queue[0]
	sim.Queue = sim.Queue[1:]
	completion.Callback()
	return true
}

// Drains every completion that is due, then advances the clock one tick, mirroring
// TigerBeetle's Storage.run.
func sim_run(sim *Sim) {
	for sim_step(sim) {
	}
	sim.Clock.Tick()
}

// Ticks until the duration has elapsed, delivering completions as they come due.
func sim_run_for(sim *Sim, duration time.Duration) {
	deadline := sim_now(sim) + time.Moment(duration)
	for sim_now(sim) < deadline {
		sim_run(sim)
	}
}
