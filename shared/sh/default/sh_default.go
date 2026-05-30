// Package sh is the composition-tier sibling of the sh library. It wires the
// pure Shell to the real OS — os.Stdout/os.Stderr, os.Environ, exec.Command, an
// operating-system clock, and syscall.Rusage accounting — and re-exports the
// surface so callers can write:
//
//	import sh "github.com/james-orcales/james-orcales/shared/sh/default"
//
// and use sh.Init_Default_Shell / sh.Shell_Spawn / … as if no split between the
// pure and OS-bound tiers existed. This is the one place in the sh tree where
// binding to the real world is permitted.
package sh

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/james-orcales/james-orcales/shared/sh"
	"github.com/james-orcales/james-orcales/shared/time"
	time_default "github.com/james-orcales/james-orcales/shared/time/default"
)

// Shell re-exports the library's Shell so callers need only this import.
type Shell = sh.Shell

// Command re-exports the library's Command.
type Command = sh.Command

// Outcome re-exports the library's Outcome.
type Outcome = sh.Outcome

// Usage re-exports the library's Usage.
type Usage = sh.Usage

// Spawn_Raw_Plan forwards to the library kernel.
func Spawn_Raw_Plan(arguments []string) (command sh.Command, ok bool) {
	return sh.Spawn_Raw_Plan(arguments)
}

// Shell_Run_Command forwards to the library op.
func Shell_Run_Command(s *sh.Shell, command sh.Command) (outcome sh.Outcome) {
	return sh.Shell_Run_Command(s, command)
}

// Shell_Spawn forwards to the library op.
func Shell_Spawn(s *sh.Shell, arguments ...string) (ok bool) {
	return sh.Shell_Spawn(s, arguments...)
}

// Shell_Pipe forwards to the library op.
func Shell_Pipe(s *sh.Shell, arguments ...string) (stdout string) {
	return sh.Shell_Pipe(s, arguments...)
}

// Shell_Working_Directory forwards to the library op.
func Shell_Working_Directory(s *sh.Shell) (directory string) {
	return sh.Shell_Working_Directory(s)
}

// Shell_With_Working_Directory forwards to the library op.
func Shell_With_Working_Directory(s *sh.Shell, directory string) (derived *sh.Shell) {
	return sh.Shell_With_Working_Directory(s, directory)
}

// Init_Default_Shell builds a Shell wired to the host OS: os.Stdout, os.Stderr,
// os.Environ, and a Run backed by exec.Command and an operating-system clock.
// Thread it through a tool that wants tests, or call the package-level Spawn and
// Pipe directly in a script.
func Init_Default_Shell() (shell *sh.Shell) {
	clock := time_default.New_Operating_System_Clock()
	return &sh.Shell{
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Environ: os.Environ,
		Run: func(command sh.Command) (outcome sh.Outcome) {
			return default_run(clock, command)
		},
	}
}

// Spawn runs a command on a host-OS Shell with inherited streams and reports
// whether it exited cleanly — the package-level convenience for scripts, which
// are composition roots and so may reach for an ambient host shell. Library-tier
// code cannot import this package to misuse it.
func Spawn(arguments ...string) (ok bool) {
	return sh.Shell_Spawn(Init_Default_Shell(), arguments...)
}

// Pipe runs a command on a host-OS Shell and returns its stdout, trimmed.
func Pipe(arguments ...string) (stdout string) {
	return sh.Shell_Pipe(Init_Default_Shell(), arguments...)
}

// Which returns the cleaned path of the named executable as found on PATH, or an
// empty string when it is not found.
func Which(name string) (path string) {
	resolved, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return filepath.Clean(resolved)
}

// File_Exists reports whether a file or directory exists at path.
func File_Exists(path string) (exists bool) {
	_, err := os.Stat(path)
	return err == nil
}

// Make_Directory_All creates path and any missing parents, like `mkdir -p`.
func Make_Directory_All(path string) (err error) {
	return os.MkdirAll(path, 0o755)
}

// Push_Directory changes the process working directory to directory and returns
// a function that restores the previous one. Process-global mutation suits a
// sequential script; it would not suit a threaded Shell, which is why this is a
// script convenience and not a Shell op. It panics if either chdir fails.
func Push_Directory(directory string) (pop func()) {
	previous, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	if chdir_err := os.Chdir(directory); chdir_err != nil {
		panic(chdir_err)
	}
	return func() {
		if restore_err := os.Chdir(previous); restore_err != nil {
			panic(restore_err)
		}
	}
}

// Lines splits s on newlines. A positive n keeps the first n fields, the last
// holding the remainder; a negative n keeps the final |n| lines; zero returns nil.
func Lines(s string, n int) (lines []string) {
	if n == 0 {
		return nil
	}
	if n > 0 {
		return strings.SplitN(s, "\n", n)
	}
	all := strings.Split(s, "\n")
	tail := -n
	if len(all) <= tail {
		return all
	}
	return all[len(all)-tail:]
}

// Spawns command as a real process and reports its Outcome. The Command arrives
// fully resolved from Shell_Run_Command, so this stays a dumb spawn: set the
// fields on exec.Cmd, run, and translate the result and accounting.
func default_run(clock time.Clock, command sh.Command) (outcome sh.Outcome) {
	process := exec.Command(command.Path, command.Arguments...)
	process.Env = command.Environment
	// An empty Dir leaves exec to inherit this process's cwd. The old SpawnRaw
	// auto-created the directory; that surprise is intentionally gone.
	process.Dir = command.Working_Directory
	process.Stdin = command.Stdin
	process.Stdout = command.Stdout
	process.Stderr = command.Stderr

	started := clock.Now_Monotonic()
	process.Run()
	wall := time.Duration(clock.Now_Monotonic() - started)

	// Exit -1 marks a process that never produced a ProcessState (e.g. the
	// executable was not found), distinct from a real exit code.
	outcome = sh.Outcome{Exit: -1, Usage: sh.Usage{Wall: wall}}
	if process.ProcessState == nil {
		return outcome
	}
	outcome.Exit = process.ProcessState.ExitCode()
	outcome.Usage.CPU_User = time.Duration(process.ProcessState.UserTime().Nanoseconds())
	outcome.Usage.CPU_System = time.Duration(process.ProcessState.SystemTime().Nanoseconds())
	if rusage, is_rusage := process.ProcessState.SysUsage().(*syscall.Rusage); is_rusage {
		outcome.Usage.RSS_Bytes_Max = default_normalize_rss_max(rusage.Maxrss)
	}
	return outcome
}

// Converts a syscall.Rusage Maxrss to bytes. Darwin reports bytes, Linux reports
// KiB; normalizing here keeps Usage platform-independent. Mirrors lint/main.go's
// RSS handling.
func default_normalize_rss_max(maxrss int64) (rss_bytes_max int64) {
	if runtime.GOOS == "linux" {
		return maxrss * 1024
	}
	return maxrss
}
