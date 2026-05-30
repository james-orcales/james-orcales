// Package sh runs external commands through an injected capability rather than
// reaching for the OS directly. The library tier is pure: every effect arrives
// as a field of Shell or an argument. The composition tier (sh/default) binds
// those fields to os/exec/syscall. Construct a Shell once at the program's
// entry point and thread it down; in tests, construct one with a fake Run.
package sh

import (
	"bytes"
	"io"
	"strings"

	"github.com/james-orcales/james-orcales/golang_snacks/time"
)

// Shell is a configured environment for running commands, plus the one seam it
// reaches the world through. Every field is immutable config or an injected
// capability; the library tier never touches the OS.
type Shell struct {
	// Working_Directory is where commands run. A field, not an os.Chdir effect,
	// so Shells with different directories coexist and a Shell is safe to copy.
	// Push_Dir doesn't exist for this reason: cd is derivation
	// (Shell_With_Working_Directory), not process-global mutation.
	Working_Directory string

	// Stdout is the default stdout for a Command that doesn't set its own.
	Stdout io.Writer
	// Stderr is the default stderr for a Command that doesn't set its own.
	Stderr io.Writer

	// No Stdin field. Stdout/Stderr are sinks: writers fan in, so one default
	// serves every command. Stdin is a consume-once source — a shared reader would
	// let the first command drain it and leave the rest at EOF. Input is bound to a
	// single invocation instead: Command.Stdin.

	// Environ returns the base environment a command starts from. os.Environ in
	// production; a fixed slice in tests.
	Environ func() (environment []string)

	// Run spawns an external process and reports what happened. The one
	// irreducible capability; every other field is config. Fake it in tests by
	// recording the Command and returning a canned Outcome.
	Run func(command Command) (outcome Outcome)
}

// Command fully describes one process invocation. Run consumes it; nothing about
// the invocation is read ambiently.
type Command struct {
	// Path is the executable name or path. Run resolves it the way the OS does.
	Path string
	// Arguments are the args after the executable, excluding Path.
	Arguments []string
	// Environment is appended to the Shell's base Environ for this command.
	// Entries are "KEY=VALUE".
	Environment []string
	// Working_Directory overrides the Shell's directory when non-empty.
	Working_Directory string
	// Stdin is this command's input. Consume-once, so it is per-invocation.
	Stdin io.Reader
	// Stdout is this command's stdout. Nil falls back to the Shell's default.
	Stdout io.Writer
	// Stderr is this command's stderr. Nil falls back to the Shell's default.
	Stderr io.Writer
}

// Outcome is everything a finished process tells the program. A command's
// relationship to the code flows through here, not through the OS, which is what
// a fake Run returns.
type Outcome struct {
	// Exit is the process exit status; 0 means success, -1 means it never ran.
	Exit int
	// Usage is resource accounting in portable units.
	Usage Usage
}

// Usage is resource accounting for one finished process, in portable units —
// never a raw kernel struct. The composition tier translates the platform clock
// and syscall.Rusage into this, settling the Darwin/Linux unit difference.
type Usage struct {
	// Wall is elapsed wall-clock time, measured by the Shell's clock.
	Wall time.Duration
	// CPU_User is CPU time spent in user space.
	CPU_User time.Duration
	// CPU_System is CPU time spent in kernel space.
	CPU_System time.Duration
	// RSS_Bytes_Max is peak resident set size in bytes, normalized across platforms.
	RSS_Bytes_Max int64
}

// Spawn_Raw_Plan is the pure kernel: it partitions arguments into a Command.
// Leading "KEY=VALUE" entries (a non-empty key before the first '=') are the
// environment; the first argument that is not such an entry is the executable,
// and the rest are its arguments. ok is false only when no executable remains —
// an empty argument list, or one that is all environment.
func Spawn_Raw_Plan(arguments []string) (command Command, ok bool) {
	var environment []string
	cut := 0
	for _, argument := range arguments {
		// A valid KEY=VALUE has its '=' past index 0 so the key is non-empty;
		// IndexByte returns -1 with no '=', which is also <= 0. The first argument
		// failing this is the executable, so stop.
		if strings.IndexByte(argument, '=') <= 0 {
			break
		}
		environment = append(environment, argument)
		cut++
	}
	remainder := arguments[cut:]
	if len(remainder) == 0 {
		return Command{}, false
	}
	if remainder[0] == "" {
		return Command{}, false
	}

	command = Command{Environment: environment, Path: remainder[0]}
	if len(remainder) > 1 {
		command.Arguments = remainder[1:]
	}
	return command, true
}

// Shell_Run_Command resolves the Shell's defaults into command — its streams,
// working directory, and base environment — then hands the fully-specified
// command to the injected Run. Resolution lives here, in the pure tier, so Run
// stays a dumb spawn.
func Shell_Run_Command(s *Shell, command Command) (outcome Outcome) {
	if command.Stdout == nil {
		command.Stdout = s.Stdout
	}
	if command.Stderr == nil {
		command.Stderr = s.Stderr
	}
	if command.Working_Directory == "" {
		command.Working_Directory = s.Working_Directory
	}
	// Base environment first so per-command entries win on a duplicate key: exec
	// honors the last assignment of a key.
	command.Environment = append(s.Environ(), command.Environment...)
	return s.Run(command)
}

// Shell_Spawn runs the command described by arguments with the Shell's default
// streams and reports whether it exited cleanly.
func Shell_Spawn(s *Shell, arguments ...string) (ok bool) {
	command, planned := Spawn_Raw_Plan(arguments)
	if !planned {
		return false
	}
	return Shell_Run_Command(s, command).Exit == 0
}

// Shell_Pipe runs the command described by arguments, captures its stdout, and
// returns it whitespace-trimmed.
func Shell_Pipe(s *Shell, arguments ...string) (stdout string) {
	command, planned := Spawn_Raw_Plan(arguments)
	if !planned {
		return ""
	}
	captured := bytes.Buffer{}
	command.Stdout = &captured
	Shell_Run_Command(s, command)
	return strings.TrimSpace(captured.String())
}

// Shell_Working_Directory returns the directory the Shell runs commands in. It
// is authoritative — the Shell's own field, not a query of the process.
func Shell_Working_Directory(s *Shell) (directory string) {
	return s.Working_Directory
}

// Shell_With_Working_Directory derives a Shell that runs in directory, sharing
// every capability with s. It copies rather than mutating, so s is untouched and
// the two coexist.
func Shell_With_Working_Directory(s *Shell, directory string) (derived *Shell) {
	clone := *s
	clone.Working_Directory = directory
	return &clone
}
