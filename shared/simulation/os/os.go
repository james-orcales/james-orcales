// Package os is a dependency-injected operating system, built the same way as shared/time: the
// backend is a struct of closures (a vtable), production wires the host kernel (os/default), and
// a simulation wires a Virtual_OS. The code between never knows which it holds.
//
// What the OS answers here is ambient state: what the kernel handed the process at exec and what
// it says about the process now — argv, the environment, the executable path, the working
// directory, the host name, the process id, and the effective user id. None of it is IO: nothing
// here opens a file, a socket, or a pipe, and nothing here can block, so none of it belongs on
// the io.IO event loop. Self_Exec sits here for the same reason. It replaces the process image
// rather than transferring bytes.
//
// The Virtual backend lives in this package because ambient state is plain data. It needs no
// seed: a simulation states the values it wants and reads exactly those back.
package os

import (
	"errors"

	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/random/prng"
	"local/james-orcales/shared/simulation/nbio"
	"local/james-orcales/shared/simulation/time"
)

// OS is the operating system as the running process sees it. It is a vtable, so a caller holds
// it by value and calls through it without knowing the backend, exactly as io.IO is held.
type OS struct {
	// Arguments returns the process argv. The first element names the program as it was
	// invoked, which is not always the executable path.
	Arguments func() (arguments []string)
	// Environment returns every variable as one "NAME=VALUE" string.
	Environment func() (variables []string)
	// Variable reads one environment variable. found is false when the name is unset, which
	// a caller must tell apart from a name set to the empty string.
	Variable func(name string) (value string, found bool)
	// Executable returns the path of the running image.
	Executable func() (path string, err error)
	// Working_Directory returns the directory that resolves the process's relative paths.
	Working_Directory func() (path string, err error)
	// Hostname returns the name the kernel gives this machine.
	Hostname func() (name string, err error)
	// Process_Identifier returns the process id.
	Process_Identifier func() (identifier int)
	// Effective_User_Identifier returns the user id the kernel checks permission against,
	// which a setuid image makes different from the user who started the process.
	Effective_User_Identifier func() (identifier int)
	// Self_Exec replaces the process image and returns only on failure. Every descriptor is
	// close-on-exec, so a successful replacement closes listeners and the new image rebinds.
	// Nil preserves ambient values. Non-nil slice is complete replacement, thus empty inherits
	// nothing.
	Self_Exec func(path string, arguments []string, environment []string) (err error)
	// Watch_Signal fires callback when the process receives signal before the finite
	// deadline, or with time.Deadline_Exceeded. The lifetime is finite deliberately: a
	// permanent waiter is a process that cannot state when it is done.
	Watch_Signal func(
		completion *time.Completion, signal Signal, deadline time.Duration,
		callback Signal_Callback,
	)
	// Spawn runs request until it finishes or the deadline expires. Expiry kills the
	// subprocess group and returns time.Deadline_Exceeded with any partial result. The
	// simulated backend draws the exit code from its seed and returns no output, since
	// scripted output is disallowed.
	Spawn func(
		completion *time.Completion, request Process_Request, deadline time.Duration,
		callback Process_Callback,
	)
}

// OS_Invariants states that every reader is bound. An OS is a vtable, so its only property is
// that every slot is filled: the zero OS reads as an OS but panics on first use, and a backend
// that fills eight slots and forgets the ninth is the same failure one call later.
func OS_Invariants(system OS, namespace invariant.Namespace) {
	invariant.Always(system.Arguments != nil, "An OS reads its arguments.")
	invariant.Always(system.Environment != nil, "An OS reads its environment.")
	invariant.Always(system.Variable != nil, "An OS reads one environment variable.")
	invariant.Always(system.Executable != nil, "An OS reads its executable path.")
	invariant.Always(system.Working_Directory != nil, "An OS reads its working directory.")
	invariant.Always(system.Hostname != nil, "An OS reads its host name.")
	invariant.Always(system.Process_Identifier != nil, "An OS reads its process id.")
	invariant.Always(
		system.Effective_User_Identifier != nil, "An OS reads its effective user id.")
	invariant.Always(system.Self_Exec != nil, "An OS replaces its own image.")
	invariant.Always(system.Watch_Signal != nil, "An OS watches for one signal.")
	invariant.Always(system.Spawn != nil, "An OS spawns a subprocess.")
}

// Self_Exec_Unsupported is what every simulated Self_Exec returns. A simulation cannot replace
// its own test process, so it reports the failure rather than pretending to succeed, which
// would destroy the run. A caller's real-backend success path never returns, so its failure
// branch is exactly what a simulation exercises.
var Self_Exec_Unsupported = errors.New("os: self-exec is not supported by a Virtual_OS")

// Virtual_OS is the ambient state a simulation gives the process, as plain data. It holds no
// closure and no clock, so a run reads back exactly the values it stated.
type Virtual_OS struct {
	// Arguments becomes the simulated argv.
	Arguments []string
	// Environment becomes the simulated environment, each entry one "NAME=VALUE" string.
	Environment []string
	// Executable becomes the simulated path of the running image.
	Executable string
	// Working_Directory becomes the simulated directory resolving relative paths.
	Working_Directory string
	// Hostname becomes the simulated machine name.
	Hostname string
	// Process_Identifier becomes the simulated process id.
	Process_Identifier int
	// Effective_User_Identifier becomes the simulated effective user id. Zero is root, so a
	// simulation states it deliberately and the field carries no positive bound.
	Effective_User_Identifier int
}

// Virtual_OS_Invariants states the complete simulated domain. A process id is positive on every
// kernel this repository targets, and pid 1 is init, so a simulation that states zero has left
// the field unset rather than described a real process.
func Virtual_OS_Invariants(virtual Virtual_OS, namespace invariant.Namespace) {
	invariant.Always(virtual.Process_Identifier > 0, "A Virtual_OS has a positive process id.")
}

// Virtual_OS_To_OS turns simulated ambient state into the vtable every caller holds, the
// counterpart of time.Virtual_Clock_To_Clock. Each reader copies before it answers, so a caller
// that keeps or edits a returned slice cannot change what the next read sees.
// It fills the ambient readers alone, so the OS it returns is not yet whole: the signal watch
// and the spawn retire a completion, which is a backend's queue and not plain data.
// New_Simulated_OS adds those two, and it is what asserts OS_Invariants.
func Virtual_OS_To_OS(virtual Virtual_OS) (system OS) {
	Virtual_OS_Invariants(virtual, "virtual_os_to_os.virtual")
	system = OS{
		Arguments: func() (arguments []string) {
			return copy_strings(virtual.Arguments)
		},
		Environment: func() (variables []string) {
			return copy_strings(virtual.Environment)
		},
		Variable: func(name string) (value string, found bool) {
			return Environment_Lookup(virtual.Environment, name)
		},
		Executable: func() (path string, err error) {
			return virtual.Executable, nil
		},
		Working_Directory: func() (path string, err error) {
			return virtual.Working_Directory, nil
		},
		Hostname: func() (name string, err error) {
			return virtual.Hostname, nil
		},
		Process_Identifier: func() (identifier int) {
			return virtual.Process_Identifier
		},
		Effective_User_Identifier: func() (identifier int) {
			return virtual.Effective_User_Identifier
		},
		Self_Exec: func(_ string, _ []string, _ []string) (err error) {
			return Self_Exec_Unsupported
		},
	}
	return system
}

// Environment_Lookup finds name in an environment holding "NAME=VALUE" entries. It is exported
// because both backends need it: a kernel answers Environment as a flat slice, and one lookup
// over that slice is the same work whichever backend produced it. A later duplicate wins, which
// is what execve leaves behind when a name is passed twice.
func Environment_Lookup(variables []string, name string) (value string, found bool) {
	for _, variable := range variables {
		for index := 0; index < len(variable); index++ {
			if variable[index] != '=' {
				continue
			}
			if variable[:index] == name {
				value = variable[index+1:]
				found = true
			}
			break
		}
	}
	return value, found
}

// Returns a copy, so a caller that edits the result cannot reach the backend's own slice.
func copy_strings(source []string) (copied []string) {
	copied = make([]string, len(source))
	copy(copied, source)
	return copied
}

// Signal identifies an operating-system signal in backend-independent form, so the
// deterministic and OS backends agree on a value without the pure tier importing syscall.
type Signal int

// SIGNAL_EXPIRED names no delivered signal. Caller retain signal it armed to identify watch.
const SIGNAL_EXPIRED Signal = -1

// SIGNAL_TERMINATE is the graceful-termination request (SIGTERM on the OS backend).
const SIGNAL_TERMINATE Signal = 0

// SIGNAL_INTERRUPT is the interactive interrupt (SIGINT on the OS backend).
const SIGNAL_INTERRUPT Signal = 1

// Signal_Callback receives a delivered signal on the loop thread, or Deadline_Exceeded when
// finite watch retires before a signal arrives.
type Signal_Callback func(completion *time.Completion, signal Signal, err error)

// Process_Request describes a subprocess to run: the executable, its arguments and
// environment, the working directory, and the bytes fed to its standard input.
type Process_Request struct {
	// Path is the executable to run.
	Path string
	// Arguments are the process arguments, excluding the program name.
	Arguments []string
	// Environment is the complete process environment. A nil value gives the child none, so a
	// caller that wants an ambient value must inject it from its root.
	Environment []string
	// Working_Directory is the process's directory; empty uses the current one.
	Working_Directory string
	// Input is the bytes written to the process's standard input.
	Input []byte
	// Stdout, when its Procedure is set, streams the process's standard output to the stream
	// as it runs instead of capturing it into Result.Output — the affordance a long build
	// needs so its progress reaches the user live. A zero Stream keeps the captured-buffer
	// default. The simulated backend produces no output and ignores it.
	//
	// The loop writes to the stream on its own thread, so a Stream that waits on the world
	// stalls every other operation. A Stream moves memory only, which is what makes it the
	// right sink here.
	Stdout nbio.Stream
	// Stderr is the standard-error counterpart, same live-or-capture rule.
	Stderr nbio.Stream
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
type Process_Callback func(completion *time.Completion, result Process_Result, err error)

// Sim is the simulated backend for the two operations an OS retires a completion for. It holds
// no queue: shared/time owns the order, and the seed owns every outcome, so a run reproduces
// and nothing is scriptable.
type Sim struct {
	// Timeline is the control plane both operations arm through.
	Timeline time.Timeline
	// Generator draws every latency and every exit code from the seed.
	Generator prng.Generator
}

// The number of virtual grains a simulated operation may take, drawn from the seed so the
// retirement order varies per run while staying reproducible.
const SIM_LATENCY_GRAINS = 8

// One in this many simulated spawns exits non-zero, so a seed sweep exercises both the success
// and the failure path without a scripted outcome.
const SIM_SPAWN_FAIL_GRAINS = 4

// New_Simulated_OS returns the simulated operating system: the ambient values virtual
// states, plus the
// signal watch and the spawn drawn from seed. Both retire on pump, so a simulated spawn and a
// simulated read hold one order. The caller owns the loop and its driver, so this backend
// submits and never pumps.
func New_Simulated_OS(seed uint64, virtual Virtual_OS, pump time.Timeline) (system OS) {
	state := &Sim{Timeline: pump, Generator: prng.New(seed)}
	system = Virtual_OS_To_OS(virtual)
	system.Watch_Signal = func(
		completion *time.Completion, signal Signal, deadline time.Duration,
		callback Signal_Callback,
	) {
		invariant.Always(deadline > 0, "A signal-watch deadline is positive and finite.")
		sim_watch_signal(state, completion, signal, deadline, callback)
	}
	system.Spawn = func(
		completion *time.Completion, request Process_Request, deadline time.Duration,
		callback Process_Callback,
	) {
		invariant.Always(deadline > 0, "A spawn deadline is positive and finite.")
		sim_spawn(state, completion, deadline, callback)
	}
	OS_Invariants(system, "new_sim.system")
	return system
}

// Watches for a signal that, in the simulation, arrives at a seed-drawn grain — the operating
// system event modeled as a seed outcome. It fires callback exactly once.
func sim_watch_signal(
	state *Sim, completion *time.Completion, signal Signal, deadline time.Duration,
	callback Signal_Callback,
) {
	latency := sim_latency(state)
	if latency >= deadline {
		state.Timeline.Submit(completion, deadline, func(_ *time.Completion) {
			callback(completion, SIGNAL_EXPIRED, time.Deadline_Exceeded)
		})
	} else {
		state.Timeline.Submit(completion, latency, func(_ *time.Completion) {
			callback(completion, signal, nil)
		})
	}
}

// Delivers a subprocess result drawn from the seed: the exit code varies (usually zero,
// occasionally non-zero for fault coverage) with no captured output — scripted output is
// disallowed, so the seed decides success or failure, not a canned payload.
func sim_spawn(
	state *Sim, completion *time.Completion, deadline time.Duration,
	callback Process_Callback,
) {
	exit := 0
	if prng.Generator_Below(&state.Generator, SIM_SPAWN_FAIL_GRAINS) == 0 {
		exit = 1
	}
	latency := sim_latency(state)
	if latency >= deadline {
		state.Timeline.Submit(completion, deadline, func(_ *time.Completion) {
			callback(completion, Process_Result{}, time.Deadline_Exceeded)
		})
		return
	}
	state.Timeline.Submit(completion, latency, func(_ *time.Completion) {
		callback(completion, Process_Result{Exit: exit}, nil)
	})
}

// Draws one simulated operation's virtual latency from the seed.
func sim_latency(state *Sim) (latency time.Duration) {
	return time.Duration(prng.Generator_Below(&state.Generator, SIM_LATENCY_GRAINS))
}
