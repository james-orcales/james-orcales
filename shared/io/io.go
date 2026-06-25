// Package io is a dependency-injected, completion-based async IO surface modeled on
// TigerBeetle's io.zig, expressed as a struct of closures so the backend is chosen
// by value: a deterministic simulated backend here, and a real OS backend
// (epoll/kqueue/IOCP, the production counterpart) in io/default. Every operation is
// submitted with a caller-owned Completion and a callback, and results arrive only
// when the loop is run. The IO owns a time.Clock so timeouts ride the same timeline
// as IO completions — the heart of the model.
package io

import (
	"errors"

	"github.com/james-orcales/james-orcales/shared/prng"
	"github.com/james-orcales/james-orcales/shared/time"
)

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

// Cancelled is the error a callback receives when its operation was cancelled before
// it completed. Cancelling still delivers the callback exactly once — with this error
// instead of a result — so every submission resolves.
var Cancelled = errors.New("io: operation cancelled")

// The number of virtual grains a simulated operation may take to complete, drawn from
// the seed so the completion order varies per run while staying reproducible.
const sim_latency_grains = 8

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
	// Cancelled marks that Cancel reached this operation before it fired; the backend's
	// Callback then delivers the Cancelled error instead of a result. Reset on submit.
	Cancelled bool
	// Armed is set while this completion sits in the queue, cleared when it fires or is
	// cancelled. sim_submit asserts it clear, so reusing one Completion for a second op
	// before the first fires panics loudly instead of corrupting the queue.
	Armed bool
}

// IO is the injected async IO submit surface — TigerBeetle's `IO`. Code submits
// operations with a Completion and callback and reacts to completions; it never drives
// the loop — that is the Driver's job — so a holder can submit IO but not advance time.
type IO struct {
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
	// synchronously — bind never blocks, so it carries no Completion.
	Listen func(host string, port int) (listener File, err error)
	// Accept yields one inbound connection on listener; callback fires with the
	// accepted socket once a peer arrives (TigerBeetle IO.accept).
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
	// Cancel stops an in-flight operation: its callback still fires exactly once, with
	// the Cancelled error rather than a result. Cancelling an already-completed or
	// unknown completion is a harmless no-op (TigerBeetle IO.cancel).
	Cancel func(completion *Completion)
}

// Driver advances the loop — the only capability that moves time and delivers
// completions. It is held solely by the composition root (package main) or a test
// harness, never by pure code, so submitting IO and driving the loop stay separate: a
// pure package holds an IO, the driver holds a Driver.
type Driver struct {
	// Run drains every ready completion without blocking, then advances the clock one
	// tick (TigerBeetle IO.run).
	Run func()
	// Run_For drives the loop until the duration has elapsed on the clock, delivering
	// completions as they come due (TigerBeetle IO.run_for_ns).
	Run_For func(duration time.Duration)
	// Run_Until drives the loop until done reports true — the run-until-complete pump
	// that lets straight-line code wait for its own op inline. Top-level and
	// single-loop only: never call it from within a completion callback.
	Run_Until func(done func() (finished bool))
}

// The sim type is the deterministic, in-memory IO backend — TigerBeetle's simulated
// Storage.
//
// ===========================================================================
// DO NOT ADD A SCRIPTING API TO THIS TYPE. READ THIS BEFORE ADDING ANY FIELD.
// ===========================================================================
//
// "Scripting" means any exported field or function that lets a caller pre-load specific
// outcomes for the sim to replay: Connect_Targets / Accept_Queue / Open_Targets maps,
// per-socket Inbound / Outbound / Connect_Error / Send_Error / Receive_Error fields, a
// Receive_Script, a Send_Limit, or Sim_Raise / Sim_Connect_Target / Sim_Open_Target
// helpers. The production io this was ported from had all of them. We removed them on
// purpose. Do not bring them back, and do not add a new one when a test feels awkward.
//
// Why it is banned: a simulation must be a PURE FUNCTION OF ITS SEED. Every outcome —
// latency, which connect fails, the bytes a receive delivers, the grain a signal lands
// on — is drawn from Generator, seeded by New_Sim. Correctness is asserted by the
// invariant framework's Always/Sometimes across many seeds (a plain seed sweep, like
// shared/vsr/simulation), and any failure reproduces by replaying its seed. Scripting
// destroys all three properties: the run becomes a function of author-chosen data
// instead of the seed, so (1) the fuzzer can only ever find bugs the author already
// imagined, (2) a seed no longer reproduces the run, and (3) the invariants end up
// judging hand-picked inputs instead of the whole reachable space. The right way to
// influence a run is to change the SEED, never to inject data.
//
// This type is unexported and New_Sim(seed) is the only entry precisely so this stays
// impossible to add by accident: no caller ever holds a *sim to hang a field on. If you
// find yourself wanting to export it, or wanting to add a parameter to New_Sim that is
// not the seed, stop — that is the scripting API trying to come back. Keep it shut.
type sim struct {
	// Clock is the read-only time source; "now" is Clock.Now_Monotonic.
	Clock time.Clock
	// Tick advances the virtual clock one resolution — the tick returned beside Clock
	// by Virtual_Clock_To_Clock. The driver calls it after each drain; it lives on the
	// sim (driver side) so pure code, which holds only an IO, can never advance time.
	Tick func()
	// Generator is the seeded entropy every outcome is drawn from; it is the sim's only
	// source of variation, so the whole run reproduces from the seed.
	Generator prng.Generator
	// Queue holds in-flight completions ordered by Ready_At, earliest first.
	Queue []*Completion
	// Next_File is the synthetic descriptor counter; Listen, Accept, and Connect
	// hand out the next value so every socket is distinct.
	Next_File File
}

// New_Sim returns a deterministic loop seeded by seed: the read-only clock and submit
// surface to inject into the program under test, and the driver to run it. The sim
// itself never escapes, and seed is the only input, so the run reproduces exactly and
// nothing can be scripted into it — correctness is asserted by invariants, not by
// hand-fed outcomes.
func New_Sim(seed uint64) (loop IO, driver Driver, clock time.Clock) {
	clock, tick := time.Virtual_Clock_To_Clock(time.Virtual_Clock{Resolution: time.Nanosecond})
	state := &sim{Clock: clock, Tick: tick, Generator: prng.New(seed)}
	loop = IO{
		Read: func(
			completion *Completion, callback Callback,
			file File, buffer []byte, offset int64,
		) {
			sim_bytes(state, completion, callback, buffer)
		},
		Write: func(
			completion *Completion, callback Callback,
			file File, buffer []byte, offset int64,
		) {
			sim_bytes(state, completion, callback, buffer)
		},
		Timeout: func(
			completion *Completion, callback Timeout_Callback,
			duration time.Duration,
		) {
			sim_submit(state, completion, duration, func() {
				if completion.Cancelled {
					callback(completion, Cancelled)
					return
				}
				callback(completion, nil)
			})
		},
		Listen: func(host string, port int) (listener File, err error) {
			state.Next_File++
			return state.Next_File, nil
		},
		Accept: func(completion *Completion, callback Socket_Callback, listener File) {
			sim_yield_socket(state, completion, callback)
		},
		Connect: func(
			completion *Completion, callback Socket_Callback, host string, port int,
		) {
			sim_yield_socket(state, completion, callback)
		},
		Receive: func(
			completion *Completion, callback Callback, socket File, buffer []byte,
		) {
			sim_bytes(state, completion, callback, buffer)
		},
		Send: func(completion *Completion, callback Callback, socket File, buffer []byte) {
			sim_bytes(state, completion, callback, buffer)
		},
		Close: func(completion *Completion, callback Timeout_Callback, file File) {
			sim_submit(state, completion, sim_latency(state), func() {
				if completion.Cancelled {
					callback(completion, Cancelled)
					return
				}
				callback(completion, nil)
			})
		},
		Cancel: func(completion *Completion) { sim_cancel(state, completion) },
	}
	return loop, sim_to_driver(state), clock
}

// Panics on a violated invariant, fail-closed — a tripped assert is always a bug in
// this package, so the simulation stops loudly instead of corrupting on.
func assert(ok bool) {
	if !ok {
		panic("io: assertion failed")
	}
}

// Draws the next operation's completion delay from the seed, so completion order
// varies per run yet reproduces exactly.
func sim_latency(state *sim) (latency time.Duration) {
	return time.Duration(prng.Generator_Below(&state.Generator, sim_latency_grains))
}

// Submits a byte-count operation — read, write, receive, or send — reporting the
// buffer length after the drawn latency, or the Cancelled error if cancelled.
func sim_bytes(state *sim, completion *Completion, callback Callback, buffer []byte) {
	count := len(buffer)
	sim_submit(state, completion, sim_latency(state), func() {
		if completion.Cancelled {
			callback(completion, 0, Cancelled)
			return
		}
		callback(completion, count, nil)
	})
}

// Builds the driver over state — the loop-advancing capability, held only by main or a
// test, never by code that merely submits IO.
func sim_to_driver(state *sim) (driver Driver) {
	return Driver{
		Run:       func() { sim_run(state) },
		Run_For:   func(duration time.Duration) { sim_run_for(state, duration) },
		Run_Until: func(done func() (finished bool)) { sim_run_until(state, done) },
	}
}

// Returns the current virtual Moment; the sim never reads the operating-system time.
func sim_now(state *sim) (now time.Moment) {
	return state.Clock.Now_Monotonic()
}

// Schedules completion to fire at now plus latency and inserts it in Ready_At order.
// Asserts the completion is not already armed, so reusing one for a second in-flight op
// panics loudly. Clears any stale Cancelled mark so a reused completion starts fresh.
func sim_submit(state *sim, completion *Completion, latency time.Duration, callback func()) {
	assert(!completion.Armed)
	completion.Armed = true
	completion.Ready_At = sim_now(state) + time.Moment(latency)
	completion.Callback = callback
	completion.Cancelled = false
	sim_enqueue(state, completion)
}

// Inserts completion into the queue in Ready_At order, earliest first.
func sim_enqueue(state *sim, completion *Completion) {
	index := 0
	for index < len(state.Queue) && state.Queue[index].Ready_At <= completion.Ready_At {
		index++
	}
	state.Queue = append(state.Queue, nil)
	copy(state.Queue[index+1:], state.Queue[index:])
	state.Queue[index] = completion
}

// Cancels an in-flight completion: if still queued, mark it Cancelled and make it due
// now so the next drain delivers its callback with the Cancelled error. A completion
// already fired or never queued is left untouched — cancel is then a harmless no-op.
func sim_cancel(state *sim, completion *Completion) {
	for index := 0; index < len(state.Queue); index++ {
		if state.Queue[index] != completion {
			continue
		}
		state.Queue = append(state.Queue[:index], state.Queue[index+1:]...)
		completion.Cancelled = true
		completion.Ready_At = sim_now(state)
		sim_enqueue(state, completion)
		return
	}
}

// Schedules callback to receive the next synthetic descriptor after the drawn latency,
// shared by Accept and Connect.
func sim_yield_socket(state *sim, completion *Completion, callback Socket_Callback) {
	state.Next_File++
	socket := state.Next_File
	sim_submit(state, completion, sim_latency(state), func() {
		if completion.Cancelled {
			callback(completion, 0, Cancelled)
			return
		}
		callback(completion, socket, nil)
	})
}

// Fires the earliest completion if it is due as of now, reporting whether it did,
// mirroring TigerBeetle's Storage.step.
func sim_step(state *sim) (advanced bool) {
	if len(state.Queue) == 0 {
		return false
	}
	if state.Queue[0].Ready_At > sim_now(state) {
		return false
	}
	completion := state.Queue[0]
	state.Queue = state.Queue[1:]
	completion.Armed = false
	completion.Callback()
	return true
}

// Drains every completion due as of now, in Ready_At order; it never advances time —
// advancing is the driver's job, so the queue itself stays passive.
func sim_drain(state *sim) {
	for sim_step(state) {
	}
}

// The driver's step: drain due completions, then advance the clock one grain via the
// injected tick (not the read-only Clock), mirroring TigerBeetle's Storage.run.
//
// The driver advances one grain per tick and never jumps ahead to the next Ready_At,
// even when the queue is idle until then. A jump would skip the grains where a
// time-triggered fault adversary — crash or partition a quiescent node, the faults
// that matter most because they strike while nothing is scheduled — would act. Uniform
// ticking keeps every grain a decision point, as TigerBeetle's simulator does for its
// per-tick crash/partition rolls.
func sim_run(state *sim) {
	sim_drain(state)
	state.Tick()
}

// Drives the loop until the duration has elapsed, delivering completions as due.
func sim_run_for(state *sim, duration time.Duration) {
	deadline := sim_now(state) + time.Moment(duration)
	for sim_now(state) < deadline {
		sim_run(state)
	}
}

// Drives the loop until done reports true — the run-until-complete pump. Top-level and
// single-loop only; each step drains then ticks.
func sim_run_until(state *sim, done func() (finished bool)) {
	for !done() {
		sim_run(state)
	}
}
