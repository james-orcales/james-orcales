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
	"io"
	"slices"
	"strconv"
	"strings"

	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/prng"
	"local/james-orcales/shared/time"
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

// Signal identifies an operating-system signal in backend-independent form, so the
// deterministic and OS backends agree on a value without the pure tier importing syscall.
type Signal int

// SIGNAL_TERMINATE is the graceful-termination request (SIGTERM on the OS backend).
const SIGNAL_TERMINATE Signal = 0

// SIGNAL_INTERRUPT is the interactive interrupt (SIGINT on the OS backend).
const SIGNAL_INTERRUPT Signal = 1

// Signal_Callback receives a delivered signal on the loop thread.
type Signal_Callback func(completion *Completion, signal Signal)

// Compute_Callback fires on the loop thread once offloaded work has finished.
type Compute_Callback func(completion *Completion)

// Process_Request describes a subprocess to run: the executable, its arguments and
// environment, the working directory, and the bytes fed to its standard input.
type Process_Request struct {
	// Path is the executable to run.
	Path string
	// Arguments are the process arguments, excluding the program name.
	Arguments []string
	// Environment is the process environment; nil inherits the parent's.
	Environment []string
	// Working_Directory is the process's directory; empty uses the current one.
	Working_Directory string
	// Input is the bytes written to the process's standard input.
	Input []byte
	// Stdout, when non-nil, streams the process's standard output to the writer as it
	// runs instead of capturing it into Result.Output — the affordance a long build
	// needs so its progress reaches the user live. A nil sink keeps the captured-buffer
	// default. The simulated backend produces no output and ignores it.
	Stdout io.Writer
	// Stderr is the standard-error counterpart, same live-or-capture rule.
	Stderr io.Writer
}

// Process_Usage is the resource accounting a finished process reports.
type Process_Usage struct {
	// Wall is the elapsed wall-clock time the process ran.
	Wall time.Duration
	// CPU_User is the user-mode CPU time consumed.
	CPU_User time.Duration
	// CPU_System is the kernel-mode CPU time consumed.
	CPU_System time.Duration
	// RSS_Bytes_Max is the peak resident set size in bytes.
	RSS_Bytes_Max int64
}

// Process_Result is a finished process's outcome: its exit code, captured output, and
// resource usage.
type Process_Result struct {
	// Exit is the process exit code; zero on success.
	Exit int
	// Output is the captured standard output.
	Output []byte
	// Error_Output is the captured standard error.
	Error_Output []byte
	// Usage is the process's resource accounting.
	Usage Process_Usage
}

// Process_Callback receives a finished process's result, or an error when the process
// could not be started at all.
type Process_Callback func(completion *Completion, result Process_Result, err error)

// Directory_Entry is one child of a directory: its name and whether it is itself a
// directory, the two facts a walk needs to recurse into subdirectories and sync files.
type Directory_Entry struct {
	// Name is the child's name within its directory, not a full path.
	Name string
	// Is_Directory reports whether the child is a directory, so a walk knows to recurse.
	Is_Directory bool
}

// File_Status is what a stat reports: whether the path exists, and if so whether it is a
// directory and its byte length — metadata a mirror consults before it reads or writes.
type File_Status struct {
	// Exists reports whether the path is present; an absent path is not an error.
	Exists bool
	// Is_Directory reports whether an existing path is a directory rather than a file.
	Is_Directory bool
	// Size is the file's length in bytes, zero for a directory or an absent path.
	Size int64
}

// Cancelled is the error a callback receives when its operation was cancelled before
// it completed. Cancelling still delivers the callback exactly once — with this error
// instead of a result — so every submission resolves.
var Cancelled = errors.New("io: operation cancelled")

// FOREVER is the Run_Until timeout that never expires: the loop pumps until done reports
// true, however long that takes — for a caller (a server) that runs until an event, not a
// clock.
const FOREVER time.Duration = -1

// IMMEDIATE is the Run_Until timeout that expires at once: done is evaluated a single time
// and the loop is not driven — a non-blocking poll of the predicate.
const IMMEDIATE time.Duration = 0

// The number of virtual grains a simulated operation may take to complete, drawn from
// the seed so the completion order varies per run while staying reproducible.
const SIM_LATENCY_GRAINS = 8

// One in this many simulated spawns exits non-zero, so a seed sweep exercises both the
// success and the failure path without a scripted outcome.
const SIM_SPAWN_FAIL_GRAINS = 4

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
	// Cancelled is the delivery payload of the cancelled edge: which error the callback
	// delivers. Set when Cancel accepts, cleared on submit, read by the delivery closure
	// at fire time — the machine has already returned to COMPLETION_IDLE when the
	// callback reads it, so this cannot live in State.
	Cancelled bool
	// State is the completion's position in its lifecycle machine, mutated only through
	// Completion_Transition. It is backend-owned: applications never read or write it —
	// expose your own state, not the completion's.
	State Completion_State
	// Self is the completion's own address, stamped on its first submit and never cleared.
	// The loop tracks an in-flight op by pointer, so a by-value copy carries this original
	// address; submitting the copy trips the backend's assert instead of silently
	// splitting the loop's view from the caller's. Only the backends touch it.
	Self *Completion
}

// Completion_State is one position in a completion's lifecycle machine. The machine has
// four legal edges — idle to armed on submit, armed to idle on delivery, armed to
// cancelled on Cancel, cancelled to idle on the cancelled delivery — and every mutation
// goes through Completion_Transition, so an illegal move panics instead of corrupting a
// queue.
type Completion_State int

// COMPLETION_IDLE is the zero value: never submitted, or delivered and reusable. A
// delivery resets to idle before the callback runs, so a callback may resubmit its own
// completion — the repeating-timer pattern.
const COMPLETION_IDLE Completion_State = 0

// COMPLETION_ARMED marks an in-flight operation: submitted and owned by the loop until
// its delivery or a Cancel.
const COMPLETION_ARMED Completion_State = 1

// COMPLETION_CANCELLED marks the cancel window: Cancel accepted, the delivery with the
// Cancelled error still pending. Resubmitting inside the window is an illegal edge.
const COMPLETION_CANCELLED Completion_State = 2

// Input for Completion_Transition_Legal.
type Completion_Transition_Legal_Input struct {
	// From is the state the edge leaves.
	From Completion_State
	// To is the state the edge enters.
	To Completion_State
}

// Completion_Transition_Legal is the machine's transition table: it reports whether the
// edge from one state to another exists. A function rather than a table value because Go
// has no const maps and a package var is banned; the flat one-clause-per-edge shape is
// the point — the whole graph, readable in one place.
func Completion_Transition_Legal(input *Completion_Transition_Legal_Input) (legal bool) {
	if input.From == COMPLETION_IDLE {
		return input.To == COMPLETION_ARMED
	}
	if input.From == COMPLETION_ARMED {
		if input.To == COMPLETION_IDLE {
			return true
		}
		return input.To == COMPLETION_CANCELLED
	}
	if input.From == COMPLETION_CANCELLED {
		return input.To == COMPLETION_IDLE
	}
	return false
}

// Input for Completion_Transition.
type Completion_Transition_Input struct {
	// Completion is the completion being moved along an edge.
	Completion *Completion
	// From is the state the caller believes the completion is in.
	From Completion_State
	// To is the destination state.
	To Completion_State
}

// Completion_Transition moves a completion along one edge of its lifecycle machine. Its
// two Always guards fail loudly on a caller whose belief about the current state is
// stale — a reused or double-armed completion — and on an edge the machine does not
// have, so a lifecycle bug dies at the mutation instead of corrupting a queue. Every
// transition then records its edge on the io.completion.transition grid: this package's
// own suite registers the grid through its TestMain, so an edge the sim suite never
// witnesses fails the run — the graph is enforced by the guards and witnessed by the
// sweep. Backend code only; applications never transition a completion.
func Completion_Transition(input *Completion_Transition_Input) {
	invariant.Always(input.Completion.State == input.From,
		"A completion transitions from the state its caller expects.")
	legal := Completion_Transition_Legal(&Completion_Transition_Legal_Input{
		From: input.From, To: input.To,
	})
	invariant.Always(legal, "A completion transitions along an edge its machine has.")
	input.Completion.State = input.To
	// Three axes identify each legal edge as one grid cell; the Impossible carves remove
	// exactly the from-to tuples the legality table forbids, so the demanded grid is the
	// four legal edges and nothing else.
	invariant.Dot_Product("io.completion.transition",
		invariant.Sometimes(input.From == COMPLETION_IDLE, "the edge leaves idle"),
		invariant.Sometimes(
			input.From == COMPLETION_CANCELLED, "the edge leaves cancelled"),
		invariant.Sometimes(input.To == COMPLETION_IDLE, "the edge enters idle"),
		invariant.Impossible(
			invariant.Event_True("the edge leaves idle"),
			invariant.Event_True("the edge leaves cancelled")),
		invariant.Impossible(
			invariant.Event_True("the edge leaves idle"),
			invariant.Event_True("the edge enters idle")),
		invariant.Impossible(
			invariant.Event_True("the edge leaves cancelled"),
			invariant.Event_False("the edge enters idle")),
	)
}

// IO is the injected async IO submit surface — TigerBeetle's `IO`. Code submits
// operations with a Completion and callback and reacts to completions; it never drives
// the loop — that is the Driver's job — so a holder can submit IO but not advance time.
//
// CRITICAL: ONLY PACKAGE MAIN OR A TEST MAY DRIVE, RUN, OR TICK THE EVENT LOOP. An IO
// holder that wants to wait exposes doneness as state and lets the root pump; it never
// receives a pump. See shared/io/README.md, THE CRITICAL GUARANTEE.
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
	// IO completions use (TigerBeetle IO.timeout). THIS IS THE EQUIVALENT TO `stdtime.Sleep`.
	Timeout func(completion *Completion, callback Timeout_Callback, duration time.Duration)
	// Listen binds and listens on host:port, returning the listening socket
	// synchronously — bind never blocks, so it carries no Completion.
	Listen func(host string, port int) (listener File, err error)
	// Open opens the file at path for reading, returning its descriptor synchronously —
	// opening never blocks the loop, so it carries no Completion. A later Read of the
	// descriptor yields the file's bytes.
	Open func(path string) (file File, err error)
	// Create opens path for writing, truncating it, and returns its descriptor
	// synchronously; a later Write persists bytes to it.
	Create func(path string) (file File, err error)
	// Read_Directory lists path's immediate children synchronously — a readdir never blocks
	// the loop, so it carries no Completion; each entry names a child and whether it is
	// itself a directory.
	Read_Directory func(path string) (entries []Directory_Entry, err error)
	// Status reports whether path exists, whether it is a directory, and its byte size,
	// synchronously; an absent path is Exists false with a nil error, so a caller branches on
	// the status, not the error.
	Status func(path string) (status File_Status, err error)
	// Make_Directory creates path and any missing parents synchronously; an existing
	// directory is not an error, so a repeated mkdir converges.
	Make_Directory func(path string) (err error)
	// Accept yields one inbound connection on listener; callback fires with the
	// accepted socket once a peer arrives (TigerBeetle IO.accept).
	Accept func(completion *Completion, callback Socket_Callback, listener File)
	// Accept_Secure is Accept with server-side TLS termination: the accepted socket
	// carries decrypted bytes so the caller speaks plaintext while the backend owns TLS.
	// The certificate provider is opaque so this pure surface names no crypto/tls type;
	// the simulator has no TLS and drives the same socket as Accept, ignoring it.
	Accept_Secure func(
		completion *Completion, callback Socket_Callback,
		listener File, certificate func() (value any),
	)
	// Connect opens a socket to host:port; callback fires with the connected
	// socket once the handshake completes (TigerBeetle IO.connect).
	Connect func(completion *Completion, callback Socket_Callback, host string, port int)
	// Connect_Secure opens a verified TLS client connection to host:port; Receive and
	// Send on the returned socket carry decrypted bytes. The simulator models it like
	// Connect — TLS is a backend concern the pure tier never sees.
	Connect_Secure func(
		completion *Completion, callback Socket_Callback,
		host string, port int, server_name string,
	)
	// Connect_Insecure is Connect_Secure without certificate verification, for probing a
	// freshly-rebuilt host whose cert is self-signed until ACME runs; never use it
	// against a peer whose identity is depended on. The simulator models it like Connect.
	Connect_Insecure func(
		completion *Completion, callback Socket_Callback,
		host string, port int, server_name string,
	)
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
	// Peer_Address returns the remote IP address of a connected socket, synchronously —
	// a getpeername has no completion. It is the source a control-plane connection is
	// gated on.
	Peer_Address func(file File) (address string, err error)
	// Watch_Signal fires callback on the loop thread when the process receives signal,
	// so a buffered shutdown can drain before exit. In the simulator the signal arrives
	// at a seed-drawn grain — the OS event modeled as a seed outcome, not scripted.
	Watch_Signal func(completion *Completion, callback Signal_Callback, signal Signal)
	// ONLY FOR COMPUTE-INTENSIVE WORK THAT CAN BE HIGHLY PARALLELIZED — E.G. JSON
	// PARSING, HIGH-TRAFFIC REQUEST PROCESSING. IT IS NOT AN ESCAPE HATCH FOR BLOCKING
	// SYSCALLS OR IO; THOSE BELONG ON THE LOOP'S COMPLETION OPS.
	Compute func(completion *Completion, callback Compute_Callback, work func())
	// Spawn runs the command in request to completion off the loop thread, firing
	// callback on the loop with its exit code, captured output, and resource usage — the
	// subprocess counterpart of the other completion ops. The simulator draws the exit
	// code from the seed and returns no output, since scripted output is disallowed.
	Spawn func(completion *Completion, callback Process_Callback, request Process_Request)
	// Self_Exec replaces this process's image with path/argv (an execve — same PID, the
	// virtual memory replaced), layering extra_environment onto the current environment.
	// Every descriptor the backend opened is marked close-on-exec first except preserve,
	// which survives into the new image at the same descriptor numbers — the mechanism that
	// lets a listening socket's bind live across a binary update with no unbind gap. On
	// success it never returns; on failure it returns an error with the process and every
	// descriptor undisturbed, so the caller may fall back to another restart path. It
	// carries no Completion because neither outcome — vanishing or returning at once — is a
	// deferred delivery. The simulator cannot replace its own test process, so it always
	// returns an error.
	Self_Exec func(
		path string, argv []string, extra_environment []string, preserve []File,
	) (err error)
}

// Driver advances the loop — the only capability that moves time and delivers
// completions.
//
// ===========================================================================
// ONLY PACKAGE MAIN OR A TEST MAY DRIVE, RUN, OR TICK THE EVENT LOOP.
// NOT A LIBRARY. NOT A HELPER. NOT AN INJECTED FUNC VALUE. NOT ONCE.
// A VIOLATION IS AN ARCHITECTURAL BUG EVEN IF EVERY TEST PASSES.
// ===========================================================================
//
// Only the code that constructs a Driver may hold or call it: a binary's package main
// in production, a test harness in simulation (the universe package is one). Handing
// any of these funcs — or a bare func value of the same shape, which the lint cannot
// see — to a library hands it the timeline: assembled into the universe package, that
// library would advance every other application's events from inside its own call
// stack. That it compiles and passes tests does not make it legal; the bug is invisible
// where it is written and fatal where it composes.
type Driver struct {
	// Run drains every ready completion without blocking, then advances the clock one
	// tick (TigerBeetle IO.run). ROOT ONLY: never handed to, or called from, a library.
	Run func()
	// Run_For drives the loop until the duration has elapsed on the clock, delivering
	// completions as they come due (TigerBeetle IO.run_for_ns). Here time is the GOAL: it
	// advances exactly duration, draining as it goes, regardless of what completes — reach
	// for it to let a span of time pass, not to wait for a particular op.
	// ROOT ONLY: never handed to, or called from, a library.
	Run_For func(duration time.Duration)
	// Run_Until drives the loop until done reports true — the run-until-complete pump that
	// lets straight-line code wait for its own op inline. Here completion is the GOAL and
	// time is the GUARD: it stops the instant done holds, and timeout only caps the wait so
	// a stalled op can't hang the caller. This opposite emphasis — completion-first with a
	// time bound, versus Run_For's time-first — is why the two stay separate ops.
	//
	// timeout < 0 (FOREVER) waits unbounded — a server pumping until a shutdown signal.
	// timeout == 0 (IMMEDIATE) evaluates done once and returns without driving — a poll.
	// timeout > 0 pumps until done or the clock passes now+timeout. completed reports which
	// won: done (true) or the timeout (false).
	//
	// Top-level and single-loop only: never call it from within a completion callback.
	// ROOT ONLY: never inject it — or a func value of its shape — into a library; a
	// library that authors done predicates and timeouts is driving the loop.
	Run_Until func(done func() (finished bool), timeout time.Duration) (completed bool)
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

// Returned when a path resolves to nothing — the simulator's ENOENT.
var sim_file_absent = errors.New("io: no such file or directory")

// Returned when a path component that must be a directory is a file.
var sim_not_a_directory = errors.New("io: not a directory")

// Returned when a file operation names a directory.
var sim_is_a_directory = errors.New("io: is a directory")

// A sim_node is one entry in the simulator's in-memory filesystem: a directory with named
// children, or a file holding bytes. Generated from the seed at New_Sim and mutated by
// Create/Write/Make_Directory, so a later read reflects an earlier write.
type sim_node struct {
	// Directory reports whether this node is a directory rather than a file.
	Directory bool
	// Contents holds a file's bytes; nil for a directory.
	Contents []byte
	// Children maps a directory's entry names to their nodes; nil for a file.
	Children map[string]*sim_node
}

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
	// Next_File is the synthetic descriptor counter; Listen, Accept, Connect, Open, and
	// Create hand out the next value so every descriptor is distinct.
	Next_File File
	// Root is the in-memory filesystem the file ops read and mutate, fabricated from the
	// seed at New_Sim. Socket descriptors ignore it.
	Root *sim_node
	// Files binds an open file descriptor to its node, so Read/Write route to real tree
	// bytes; a descriptor absent from this map is a socket, whose bytes stay synthetic.
	Files map[File]*sim_node
	// Drive_Active is set while a Run* is driving the loop, so a Run* called from within a
	// completion callback — which would re-enter the driver mid-drain — panics loudly.
	Drive_Active bool
}

// New_Sim returns a deterministic loop seeded by seed: the read-only clock and submit
// surface to inject into the program under test, and the driver to run it. The sim
// itself never escapes, and seed is the only input, so the run reproduces exactly and
// nothing can be scripted into it — correctness is asserted by invariants, not by
// hand-fed outcomes. The driver stays in the harness: the program under test receives
// loop and clock, NEVER a pump (see the Driver banner).
func New_Sim(seed uint64) (loop IO, driver Driver, clock time.Clock) {
	clock, tick := time.Virtual_Clock_To_Clock(time.Virtual_Clock{Resolution: time.NANOSECOND})
	state := &sim{
		Clock:     clock,
		Tick:      tick,
		Generator: prng.New(seed),
		Files:     map[File]*sim_node{},
	}
	state.Root = sim_generate(&state.Generator)
	sim_wire_bytes(state, &loop)
	sim_wire_lifecycle(state, &loop)
	sim_wire_socket(state, &loop)
	sim_wire_effects(state, &loop)
	return loop, sim_to_driver(state), clock
}

// Wires the byte-count operations — read, write, receive, send — onto loop.
func sim_wire_bytes(state *sim, loop *IO) {
	loop.Read = func(
		completion *Completion, callback Callback, file File, buffer []byte, offset int64,
	) {
		node := state.Files[file]
		if node != nil {
			sim_file_read(state, completion, callback, node, buffer, offset)
			return
		}
		sim_bytes(state, completion, callback, buffer)
	}
	loop.Write = func(
		completion *Completion, callback Callback, file File, buffer []byte, offset int64,
	) {
		node := state.Files[file]
		if node != nil {
			sim_file_write(state, completion, callback, node, buffer, offset)
			return
		}
		sim_bytes(state, completion, callback, buffer)
	}
	loop.Receive = func(completion *Completion, callback Callback, socket File, buffer []byte) {
		sim_bytes(state, completion, callback, buffer)
	}
	loop.Send = func(completion *Completion, callback Callback, socket File, buffer []byte) {
		sim_bytes(state, completion, callback, buffer)
	}
}

// Wires the lifecycle operations — timer, listen, open, create, close, cancel — onto loop.
func sim_wire_lifecycle(state *sim, loop *IO) {
	loop.Timeout = func(
		completion *Completion, callback Timeout_Callback, duration time.Duration,
	) {
		sim_submit(state, completion, duration,
			sim_deliver_status(completion, callback, nil))
	}
	loop.Listen = func(host string, port int) (listener File, err error) {
		return sim_descriptor(state), nil
	}
	loop.Open = func(path string) (file File, err error) {
		return sim_open(state, path)
	}
	loop.Create = func(path string) (file File, err error) {
		return sim_create(state, path)
	}
	loop.Read_Directory = func(path string) (entries []Directory_Entry, err error) {
		return sim_read_directory(state.Root, path)
	}
	loop.Status = func(path string) (status File_Status, err error) {
		return sim_status(state.Root, path), nil
	}
	loop.Make_Directory = func(path string) (err error) {
		return sim_make_directory(state.Root, path)
	}
	loop.Close = func(completion *Completion, callback Timeout_Callback, file File) {
		delete(state.Files, file)
		sim_submit(state, completion, sim_latency(state),
			sim_deliver_status(completion, callback, nil))
	}
	loop.Cancel = func(completion *Completion) { sim_cancel(state, completion) }
}

// Wires the socket operations — accept, connect, their TLS variants, peer address —
// onto loop. The TLS variants reuse the plaintext socket: the simulator has no TLS, so
// the certificate provider and server name are ignored (a backend-only concern).
func sim_wire_socket(state *sim, loop *IO) {
	loop.Accept = func(completion *Completion, callback Socket_Callback, listener File) {
		sim_yield_socket(state, completion, callback)
	}
	loop.Accept_Secure = func(
		completion *Completion, callback Socket_Callback,
		listener File, _ func() (value any),
	) {
		sim_yield_socket(state, completion, callback)
	}
	loop.Connect = func(
		completion *Completion, callback Socket_Callback, host string, port int,
	) {
		sim_yield_socket(state, completion, callback)
	}
	loop.Connect_Secure = func(
		completion *Completion, callback Socket_Callback, host string, port int, _ string,
	) {
		sim_yield_socket(state, completion, callback)
	}
	loop.Connect_Insecure = func(
		completion *Completion, callback Socket_Callback, host string, port int, _ string,
	) {
		sim_yield_socket(state, completion, callback)
	}
	loop.Peer_Address = func(file File) (address string, err error) {
		return sim_peer_address(file), nil
	}
}

// Wires the effect operations — signal watch and compute offload — onto loop.
func sim_wire_effects(state *sim, loop *IO) {
	loop.Watch_Signal = func(
		completion *Completion, callback Signal_Callback, signal Signal,
	) {
		sim_watch_signal(state, completion, callback, signal)
	}
	loop.Compute = func(completion *Completion, callback Compute_Callback, work func()) {
		sim_compute(state, completion, callback, work)
	}
	loop.Spawn = func(
		completion *Completion, callback Process_Callback, request Process_Request,
	) {
		sim_spawn(state, completion, callback, request)
	}
	loop.Self_Exec = func(
		path string, argv []string, extra_environment []string, preserve []File,
	) (err error) {
		return sim_self_exec_unsupported
	}
}

// The error every simulated Self_Exec returns: the simulator cannot replace its own test
// process, so it reports the failure rather than pretending to succeed (which would destroy
// the run). A caller's real-backend success path never returns, so its fallback branch is
// exactly what the simulator exercises.
var sim_self_exec_unsupported = errors.New("io: self-exec is not supported by the simulator")

// Delivers a subprocess result drawn from the seed: the exit code varies (usually zero,
// occasionally non-zero for fault coverage) with no captured output — scripted output is
// disallowed, so the seed decides success or failure, not a canned payload.
func sim_spawn(
	state *sim, completion *Completion, callback Process_Callback, request Process_Request,
) {
	exit := 0
	if prng.Generator_Below(&state.Generator, SIM_SPAWN_FAIL_GRAINS) == 0 {
		exit = 1
	}
	sim_submit(state, completion, sim_latency(state), func() {
		if completion.Cancelled {
			callback(completion, Process_Result{}, Cancelled)
			return
		}
		callback(completion, Process_Result{Exit: exit}, nil)
	})
}

// Panics on a violated invariant, fail-closed — a tripped assert is always a bug in
// this package, so the simulation stops loudly instead of corrupting on.
// Hands out the next distinct synthetic descriptor.
func sim_descriptor(state *sim) (file File) {
	state.Next_File++
	return state.Next_File
}

// Returns a fresh empty directory node, the shape the root, mkdir, and the generator all
// build.
func sim_new_directory() (node *sim_node) {
	return &sim_node{Directory: true, Children: map[string]*sim_node{}}
}

// Splits an absolute path into its non-empty component names, so "/a/b" walks as a, b.
func sim_path_names(path string) (names []string) {
	names = []string{}
	for _, name := range strings.Split(path, "/") {
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

// Resolves path against root, returning the node it names and whether it was found.
func sim_resolve(root *sim_node, path string) (node *sim_node, found bool) {
	node = root
	for _, name := range sim_path_names(path) {
		if !node.Directory {
			return nil, false
		}
		child, present := node.Children[name]
		if !present {
			return nil, false
		}
		node = child
	}
	return node, true
}

// Reports path's status against root: an unresolved path is Exists false, else its kind.
func sim_status(root *sim_node, path string) (status File_Status) {
	node, found := sim_resolve(root, path)
	if !found {
		return File_Status{}
	}
	return File_Status{
		Exists:       true,
		Is_Directory: node.Directory,
		Size:         int64(len(node.Contents)),
	}
}

// Lists path's immediate children sorted by name — sorted so the order is deterministic
// despite the backing map, since a run must reproduce. Absent or non-directory paths error.
func sim_read_directory(root *sim_node, path string) (entries []Directory_Entry, err error) {
	node, found := sim_resolve(root, path)
	if !found {
		return nil, sim_file_absent
	}
	if !node.Directory {
		return nil, sim_not_a_directory
	}
	entries = []Directory_Entry{}
	for name, child := range node.Children {
		entry := Directory_Entry{Name: name, Is_Directory: child.Directory}
		entries = append(entries, entry)
	}
	slices.SortFunc(entries, func(left, right Directory_Entry) (order int) {
		return strings.Compare(left.Name, right.Name)
	})
	return entries, nil
}

// Creates path and any missing parents against root; an existing directory converges, and a
// file where a directory is needed errors.
func sim_make_directory(root *sim_node, path string) (err error) {
	node := root
	for _, name := range sim_path_names(path) {
		if !node.Directory {
			return sim_not_a_directory
		}
		child, present := node.Children[name]
		if !present {
			child = sim_new_directory()
			node.Children[name] = child
		}
		node = child
	}
	if !node.Directory {
		return sim_not_a_directory
	}
	return nil
}

// Opens path for reading against state.Root, binding a fresh descriptor to its node. An
// absent path, or a directory, errors — matching a real open of a missing or non-file path.
func sim_open(state *sim, path string) (file File, err error) {
	node, found := sim_resolve(state.Root, path)
	if !found {
		return 0, sim_file_absent
	}
	if node.Directory {
		return 0, sim_is_a_directory
	}
	descriptor := sim_descriptor(state)
	state.Files[descriptor] = node
	return descriptor, nil
}

// Creates or truncates the file at path against state.Root and binds a fresh descriptor to
// it. The parent directory must already exist, matching a real create.
func sim_create(state *sim, path string) (file File, err error) {
	node, create_err := sim_create_file(state.Root, path)
	if create_err != nil {
		return 0, create_err
	}
	descriptor := sim_descriptor(state)
	state.Files[descriptor] = node
	return descriptor, nil
}

// Resolves path's parent (which must be an existing directory), then creates a fresh file
// node under it or truncates an existing file, returning the node.
func sim_create_file(root *sim_node, path string) (node *sim_node, err error) {
	names := sim_path_names(path)
	if len(names) == 0 {
		return nil, sim_is_a_directory
	}
	parent := root
	for _, name := range names[:len(names)-1] {
		child, present := parent.Children[name]
		if !present {
			return nil, sim_file_absent
		}
		if !child.Directory {
			return nil, sim_not_a_directory
		}
		parent = child
	}
	leaf := names[len(names)-1]
	leaf_node, present := parent.Children[leaf]
	if present {
		if leaf_node.Directory {
			return nil, sim_is_a_directory
		}
		leaf_node.Contents = []byte{}
		return leaf_node, nil
	}
	created := &sim_node{Contents: []byte{}}
	parent.Children[leaf] = created
	return created, nil
}

// Submits a file read that, when it fires, copies the node's bytes from offset into the
// buffer and reports the count — so a read reflects whatever an earlier write stored.
func sim_file_read(
	state *sim, completion *Completion, callback Callback, node *sim_node,
	buffer []byte, offset int64,
) {
	sim_submit(state, completion, sim_latency(state), func() {
		if completion.Cancelled {
			callback(completion, 0, Cancelled)
			return
		}
		count := 0
		if offset < int64(len(node.Contents)) {
			count = copy(buffer, node.Contents[offset:])
		}
		callback(completion, count, nil)
	})
}

// Submits a file write that, when it fires, stores the buffer into the node at offset,
// growing its contents as needed, and reports the byte count.
func sim_file_write(
	state *sim, completion *Completion, callback Callback, node *sim_node,
	buffer []byte, offset int64,
) {
	sim_submit(state, completion, sim_latency(state), func() {
		if completion.Cancelled {
			callback(completion, 0, Cancelled)
			return
		}
		sim_node_write(node, buffer, offset)
		callback(completion, len(buffer), nil)
	})
}

// Stores buffer into node's contents at offset, growing the backing bytes when the write
// extends past the current end.
func sim_node_write(node *sim_node, buffer []byte, offset int64) {
	end_size := offset + int64(len(buffer))
	if end_size > int64(len(node.Contents)) {
		grown := make([]byte, end_size)
		copy(grown, node.Contents)
		node.Contents = grown
	}
	copy(node.Contents[offset:], buffer)
}

// The file-size percentiles the generated contents are sampled from: most files are a
// handful of bytes, a few reach hundreds, and the top one percent the largest — a
// heavy-tailed spread (prng.Percentile_Distribution) so a sweep meets many scales at once.
const SIM_SIZE_P50 = 4

// SIM_SIZE_P75 is 16, a 4x step past P50 that keeps the body of files a handful of bytes.
const SIM_SIZE_P75 = 16

// SIM_SIZE_P95 is 64, another 4x step marking where the common sizes end.
const SIM_SIZE_P95 = 64

// SIM_SIZE_P99 is 256, the hundreds-scale files only the last percent reach.
const SIM_SIZE_P99 = 256

// SIM_SIZE_P100 is 1024, the heavy tail's largest sample capping the sweep.
const SIM_SIZE_P100 = 1024

// Returns how often the generator adds another sibling — a 3-in-4 Chance, so a directory's
// breadth is geometric and any width is reachable rather than capped at a fixed count.
func sim_grow_chance() (chance prng.Ratio) {
	return prng.Ratio{Numerator: 3, Denominator: 4}
}

// Returns how often a generated child is a subdirectory rather than a file — kept below the
// grow chance so the branching stays subcritical and generation halts almost surely.
func sim_subdirectory_chance() (chance prng.Ratio) {
	return prng.Ratio{Numerator: 1, Denominator: 4}
}

// Fabricates a filesystem from the seed, drawing its shape from prng's distributions: each
// directory grows siblings on a Chance coin (geometric breadth and depth, no ceiling), a
// child is a subdirectory on another Chance, and a file's size is Sampled from a heavy-tailed
// Percentile spread. It knows no consumer's layout; a walker imposes its own meaning.
func sim_generate(generator *prng.Generator) (root *sim_node) {
	sizes := prng.Percentile_Distribution(&prng.Percentile_Distribution_Input{
		P25:  0,
		P50:  SIM_SIZE_P50,
		P75:  SIM_SIZE_P75,
		P95:  SIM_SIZE_P95,
		P99:  SIM_SIZE_P99,
		P100: SIM_SIZE_P100,
	})
	root = sim_new_directory()
	directories := []*sim_node{root}
	for len(directories) > 0 {
		directory := directories[len(directories)-1]
		directories = directories[:len(directories)-1]
		index := 0
		for prng.Generator_Chance(generator, sim_grow_chance()) {
			name := "e" + strconv.Itoa(index)
			index++
			if prng.Generator_Chance(generator, sim_subdirectory_chance()) {
				child := sim_new_directory()
				directory.Children[name] = child
				directories = append(directories, child)
				continue
			}
			contents := sim_generate_bytes(generator, sizes)
			directory.Children[name] = &sim_node{Contents: contents}
		}
	}
	return root
}

// Draws a file's contents from the seed: a size Sampled from the heavy-tailed distribution,
// filled with seed-drawn bytes.
func sim_generate_bytes(
	generator *prng.Generator, sizes prng.Distribution[uint64],
) (contents []byte) {
	contents = make([]byte, prng.Generator_Sample(generator, sizes))
	for index := range contents {
		contents[index] = byte(prng.Generator_Next(generator))
	}
	return contents
}

// Draws the next operation's completion delay from the seed, so completion order
// varies per run yet reproduces exactly.
func sim_latency(state *sim) (latency time.Duration) {
	return time.Duration(prng.Generator_Below(&state.Generator, SIM_LATENCY_GRAINS))
}

// Wraps a byte-count delivery so a cancelled completion reports Cancelled instead.
func sim_deliver_bytes(completion *Completion, callback Callback, count int) (deliver func()) {
	return func() {
		if completion.Cancelled {
			callback(completion, 0, Cancelled)
			return
		}
		callback(completion, count, nil)
	}
}

// Wraps a status delivery (timeout, close) so a cancelled completion reports Cancelled.
func sim_deliver_status(
	completion *Completion, callback Timeout_Callback, err error,
) (deliver func()) {
	return func() {
		if completion.Cancelled {
			callback(completion, Cancelled)
			return
		}
		callback(completion, err)
	}
}

// Wraps a socket delivery so a cancelled completion reports Cancelled instead.
func sim_deliver_socket(
	completion *Completion, callback Socket_Callback, socket File,
) (deliver func()) {
	return func() {
		if completion.Cancelled {
			callback(completion, 0, Cancelled)
			return
		}
		callback(completion, socket, nil)
	}
}

// Submits a byte-count operation — read, write, receive, or send — reporting the
// buffer length after the drawn latency, or the Cancelled error if cancelled.
func sim_bytes(state *sim, completion *Completion, callback Callback, buffer []byte) {
	sim_submit(state, completion, sim_latency(state),
		sim_deliver_bytes(completion, callback, len(buffer)))
}

// Schedules callback to receive the next synthetic descriptor after the drawn latency,
// shared by Accept and Connect and their TLS variants.
func sim_yield_socket(state *sim, completion *Completion, callback Socket_Callback) {
	socket := sim_descriptor(state)
	sim_submit(state, completion, sim_latency(state),
		sim_deliver_socket(completion, callback, socket))
}

// Reports a synthetic peer address for a live descriptor, the simulator's getpeername;
// empty with no error for an unknown descriptor.
func sim_peer_address(file File) (address string) {
	if file <= 0 {
		return ""
	}
	return "127.0.0.1"
}

// Watches for a signal that, in the simulator, arrives at a seed-drawn grain — the OS
// event modeled as a seed outcome. It fires callback with the signal exactly once.
func sim_watch_signal(
	state *sim, completion *Completion, callback Signal_Callback, signal Signal,
) {
	sim_submit(state, completion, sim_latency(state), func() {
		callback(completion, signal)
	})
}

// Runs work inline after the drawn latency, then fires callback on the loop — the
// deterministic counterpart of the OS backend's worker pool. A cancelled compute skips
// the work but still fires callback so the submission resolves.
func sim_compute(
	state *sim, completion *Completion, callback Compute_Callback, work func(),
) {
	sim_submit(state, completion, sim_latency(state), func() {
		if !completion.Cancelled {
			work()
		}
		callback(completion)
	})
}

// Builds the driver over state — the loop-advancing capability, held only by main or a
// test, never by code that merely submits IO.
func sim_to_driver(state *sim) (driver Driver) {
	return Driver{
		Run: func() { sim_drive(state, func() { sim_run(state) }) },
		Run_For: func(duration time.Duration) {
			sim_drive(state, func() { sim_run_for(state, duration) })
		},
		Run_Until: func(
			done func() (finished bool), timeout time.Duration,
		) (completed bool) {
			sim_drive(state, func() { completed = sim_run_until(state, done, timeout) })
			return completed
		},
	}
}

// Runs pump as the top-level drive, asserting no drive is already in progress so a Run*
// called from within a completion callback panics instead of re-entering the driver. The
// internal per-tick functions call one another directly, not through here, so nested
// ticking within one drive does not trip it.
func sim_drive(state *sim, pump func()) {
	invariant.Always(!state.Drive_Active,
		"A drive begins at top level, never from within a completion callback.")
	state.Drive_Active = true
	defer func() { state.Drive_Active = false }()
	pump()
}

// Returns the current virtual Moment; the sim never reads the operating-system time.
func sim_now(state *sim) (now time.Moment) {
	return state.Clock.Now_Monotonic()
}

// Schedules completion to fire at now plus latency and inserts it in Ready_At order.
// Asserts the completion is its own original (not a by-value copy), then arms it through
// the lifecycle machine — a reused in-flight completion panics as the armed-to-armed
// edge. Clears any stale Cancelled payload so a reused completion starts fresh.
func sim_submit(state *sim, completion *Completion, latency time.Duration, callback func()) {
	original := completion.Self == nil || completion.Self == completion
	invariant.Always(original,
		"A submitted completion is its own original, never a by-value copy.")
	completion.Self = completion
	Completion_Transition(&Completion_Transition_Input{
		Completion: completion, From: COMPLETION_IDLE, To: COMPLETION_ARMED,
	})
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

// Cancels an in-flight completion: if still armed and queued, move it to the cancelled
// state and make it due now so the next drain delivers its callback with the Cancelled
// error. A completion already fired, never queued, or already cancelled is left
// untouched — cancel is then a harmless no-op.
func sim_cancel(state *sim, completion *Completion) {
	if completion.State != COMPLETION_ARMED {
		return
	}
	for index := 0; index < len(state.Queue); index++ {
		if state.Queue[index] != completion {
			continue
		}
		state.Queue = append(state.Queue[:index], state.Queue[index+1:]...)
		Completion_Transition(&Completion_Transition_Input{
			Completion: completion, From: COMPLETION_ARMED, To: COMPLETION_CANCELLED,
		})
		completion.Cancelled = true
		completion.Ready_At = sim_now(state)
		sim_enqueue(state, completion)
		return
	}
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
	// Return to idle before the callback runs — TigerBeetle's ordering — so a callback
	// may legally resubmit its own completion, the repeating-timer pattern.
	if completion.State == COMPLETION_CANCELLED {
		Completion_Transition(&Completion_Transition_Input{
			Completion: completion, From: COMPLETION_CANCELLED, To: COMPLETION_IDLE,
		})
	} else {
		Completion_Transition(&Completion_Transition_Input{
			Completion: completion, From: COMPLETION_ARMED, To: COMPLETION_IDLE,
		})
	}
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

// Drives the loop until done reports true, or until timeout of virtual time has elapsed —
// the run-until-complete pump, capped so a stalled op cannot spin the sim forever. A
// negative timeout waits unbounded; a zero timeout checks done once and drives nothing.
// completed reports whether done tripped rather than the deadline. Top-level and
// single-loop only; each step drains then ticks.
func sim_run_until(
	state *sim, done func() (finished bool), timeout time.Duration,
) (completed bool) {
	deadline := sim_now(state) + time.Moment(timeout)
	for !done() {
		if timeout >= 0 {
			if sim_now(state) >= deadline {
				return false
			}
		}
		sim_run(state)
	}
	return true
}
